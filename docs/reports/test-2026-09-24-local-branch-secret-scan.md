# Test report: local-branch-secret-scan

- Date: 2026-09-24
- Plan: `docs/plans/active/2026-09-20-local-branch-secret-scan.md` (issue #169)
- Tester: `tester` subagent (Claude Code), pipeline cycle 1
- Scope: behavioral tests only for `git diff main...HEAD` at `ca3d824` (branch `feat/local-branch-secret-scan`). Not in scope: static analysis (`/verify`, already run — `docs/reports/verify-2026-09-22-local-branch-secret-scan.md`, pass) or diff quality (`/self-review`, already run — `docs/reports/self-review-2026-09-21-local-branch-secret-scan.md`, 2 MEDIUM + 10 LOW, all fixed in `3afac6e`).
- Evidence: `docs/evidence/verify-2026-09-24-012309.log` (first full `run-test.sh` pass, before this cycle's test additions), `docs/evidence/verify-2026-09-24-020430.log` (final full `run-test.sh` pass, after this cycle's test additions). Both are `run-test.sh`'s own auto-generated logs (`HARNESS_VERIFY_MODE=test`); per-command detail for everything else is in this report.

Precondition confirmed before starting: HEAD was `ca3d824faf1e0a55cfa8740156fc1c6f94bc6a55`, `git status --porcelain` was empty. (This is a respawn of the `/test` step — a prior tester session was started for this step but restarted before producing any output, no report, no commit, clean tree — so this cycle started from scratch.)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (shell, changed scope, before this cycle's additions) | full shell suite + golang | all | 0 | 0 | ~3 min (`internal/cli` uncached, 42.9s) |
| `sh tests/test-secret-scan.sh` | 6 | 6 | 0 | 0 | <1s |
| `sh tests/test-secret-scan-branch.sh` (before additions) | 60 | 60 | 0 | 0 | ~2s |
| `sh tests/test-run-verify-branch-secret-scan.sh` (before additions) | 29 | 29 | 0 | 0 | ~2s |
| `sh tests/test-run-verify-scope.sh` | 12 | 12 | 0 | 0 | <1s |
| `sh tests/test-verify-mode-split.sh` | 59 | 59 | 0 | 0 | ~1s |
| `sh tests/test-xreview-helpers.sh` | 29 | 29 | 0 | 0 | <1s |
| `sh tests/test-template-purity.sh` | 10 | 10 | 0 | 0 | <1s |
| `go test ./internal/scaffold/... ./internal/upgrade/... ./internal/cli/... -count=1` | 3 packages | 3 | 0 | 0 | ~45s |
| Environment matrix (8 variants, item 4) | 16 file-runs | 15 | 1 (variant f, both files) | 0 | see below |
| Repeat runs (`for i in 1..5`, item 5) | 10 runs | 10 | 0 | 0 | ~15s |
| Concurrency, distinct repos (item 5, 4×) | 4 parallel runs | 4 | 0 | 0 | <5s |
| Concurrency, same repo (item 8) | 8 parallel invocations | 8 | 0 | 0 | <2s |
| Red/green mutations 6a–6f (item 6) | 6 mutations | 5 matched exactly, 1 confirmed a real coverage gap | — | — | — |
| Live demonstration, scratch repo (item 7) | 3 steps | 3 | 0 | 0 | — |
| Edge cases (item 8): symlink `.gitallowed`, slash base branch, renamed file, test-mode+finding | 4 explored | 2 clean (no gap), 1 minor cosmetic quirk (no test added, see below), 1 real gap (test added) | — | — | — |
| `sh tests/test-secret-scan-branch.sh` (after additions) | 64 | 64 | 0 | 0 | ~2s |
| `sh tests/test-run-verify-branch-secret-scan.sh` (after additions) | 32 | 32 | 0 | 0 | ~2s |
| `./scripts/run-test.sh` (final, after this cycle's additions) | full shell suite (597 shell `PASS:` lines) + golang | all | 0 | 0 | ~2 min (`internal/cli` cached) |
| `go test ./internal/scaffold/... ./internal/upgrade/... ./internal/cli/... -count=1` (re-run after additions) | 3 packages | 3 | 0 | 0 | ~46s |

## Environment matrix (item 4)

Both `tests/test-secret-scan-branch.sh` and `tests/test-run-verify-branch-secret-scan.sh` run under each variant; results were identical for both files in every variant.

| Variant | Result |
| --- | --- |
| (a) `GITHUB_BASE_REF=main` | Pass, both files (60+29 assertions) |
| (b) `GITHUB_BASE_REF=some-other-branch` | Pass, both files |
| (c) `RALPH_XREVIEW_BASE=develop` | Pass, both files |
| (d) `RALPH_VERIFY_SCOPE=changed` | Pass, both files |
| (e) `RALPH_SECRET_ALLOWLIST=/nonexistent` | Pass, both files |
| (f) `HOME=/nonexistent-home-for-test` + `GIT_CONFIG_GLOBAL` pointing at a file with `init.defaultBranch=trunk` and `commit.gpgsign=true` | **Failed before the fix** (both files, exit 128: `gpg failed to sign the data... can't create directory '/nonexistent-home-for-test/.gnupg'`) — a real hermeticity gap in the *test files*, not the scripts under test. **Fixed** by isolating `HOME`/`GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM`/`GIT_TERMINAL_PROMPT` to a throwaway dir in both files, matching the existing `tests/test-terraform-gitignore.sh` pattern. Re-verified pass after the fix (60+29 assertions). See "Test gaps" for the whose-fault judgment. |
| (g) `TMPDIR=/tmp` | Pass, both files |
| (h) cwd = `tests/` (not repo root) | Pass, both files |

## Repeat and concurrency (item 5, item 8)

- 5 sequential runs each of both new test files: 10/10 passed, no output on failure paths.
- 4 concurrent instances of each test file (8 total, in a single Bash session, `wait` on all): all 8 passed, no temp-file collisions (each file's own `mktemp -d` workdir differs per invocation, as expected).
- Item 8's stronger form — running `scripts/secret-scan-branch.sh` **directly**, 8 times concurrently, against the **same** git repo (not 8 separate repos): all 8 exited 0, and `ls "${TMPDIR:-/tmp}" | grep ralph-secret-scan-branch-allowlist` found zero leftover allowlist temp files afterward. `mktemp`'s `XXXXXX` suffix gives each invocation a unique path; nothing in the script writes to a repo-shared path.

## Red/green mutation results (item 6)

All mutations were applied in-place to `scripts/secret-scan-branch.sh`, `scripts/run-verify.sh`, or `scripts/check-template.sh`, tested, then reverted with `git checkout -- <file>`; `git status --porcelain` was confirmed empty (showing only the two test files) after every single revert.

| # | Mutation | Expected | Actual | Verdict |
| --- | --- | --- | --- | --- |
| 6a | `nothing_to_scan` always exits 0, even under `--strict` (revert the self-review cycle-1 fix) | The two bypass tests + strict base-branch/empty-range assertions fail | 4 failed exactly: "on base branch: strict mode exits 3", "empty range: strict mode exits 3", "base-override bypass: strict mode now exits 3", "fast-forward bypass: strict mode now exits 3" | Matches exactly |
| 6b | Read `.gitallowed` from the working tree instead of `HEAD` | AC-2c tests fail | 1 failed: "AC-2c: uncommitted allowlist + env override still finds" (exit 0, want 1) — the ignore-notice assertion still passed because it's driven by the test's own `RALPH_SECRET_ALLOWLIST` env var, not by the mutated code path; the exit-code assertion is the one that actually discriminates this mutation, and it did | Matches (discriminated correctly; only the exit-code assertion is sensitive to this specific mutation) |
| 6c | `origin/<base>` loses priority to the local `<base>` | Origin-preference test fails | 2 failed: "origin/<base> ref present: preferred over local, clean" and "origin preference: report names origin/main" | Matches exactly |
| 6d | Every non-zero scanner exit reported as "findings" | Stub-scanner (exit 2, exit 130) tests fail | 4 failed: both "labeled a scanner failure, not findings" and both "never labeled findings" assertions | Matches exactly |
| 6e-1 | `run-verify.sh`: drop the final "Branch secret scan failed" line | Docs-only + `ran_any=1` finding tests fail | 2 failed: "finding: an explicit failure line appears..." and "finding+ran_any=1: explicit failure line also appears" | Matches exactly |
| 6e-2 | `run-verify.sh`: skip var acts on any non-empty value | `=0` and `=true` tests fail | 6 failed: all three assertions each for `=0` and `=true` | Matches exactly |
| 6f | Remove `scripts/xreview-helpers.sh` from `check-template.sh`'s `required_files` list | `test-template-purity.sh` or another test fails | **Nothing failed.** `test-template-purity.sh` (10/10), both new test files, and `go test ./internal/scaffold/... -run TestTemplate` (3/3) all stayed green. `./scripts/check-template.sh` itself still printed `Template structure looks good.` and exited 0 | **Gap confirmed, named, not closed** — see "Test gaps" below |

## Live demonstration (item 7)

Run in a scratch repo under the session scratchpad, outside the worktree, cleaned up afterward. Commands and exit codes only (no secret-shaped literal below — every value was assembled at command runtime from split `printf` pieces, same discipline as the test fixtures):

1. **AC-1**: `main` → `feature` with an early commit adding a token-shaped line, a later commit deleting it. `./scripts/secret-scan-branch.sh` → `secret-scan-branch: scanned <a>..<b> against main: findings`, exit 1.
2. Same branch, then a commit adding a bracket-expression `.gitallowed` line targeting that exact token → `secret-scan-branch: scanned <a>..<c> against main: clean`, exit 0.
3. A fresh branch off `main`: a commit adds a **non-bracketed**, self-matching `.gitallowed` rule, a later commit empties the file again (the PR #168 shape). `./scripts/secret-scan-branch.sh` → BLOCKED, with the guidance text visible (`Write the key name as a bracket expression (e.g. api_ke[y])...`), exit 1.

All three matched the plan's stated behavior exactly.

## Edge cases explored (item 8)

- **`.gitallowed` as a symlink at HEAD**: `git show HEAD:.gitallowed` returns the git blob content of a symlink, which is the literal target-path string (e.g. `real-allowlist.conf`), not the dereferenced file content — this is expected git behavior, not a bug in this script. The scan itself stayed correct (a finding elsewhere in the branch was still reported). One cosmetic side effect: the worktree-diff check (`diff -q "$worktree_allowlist" "$tmp_allowlist"`) compares the *dereferenced* filesystem content of the symlink (`-f` follows symlinks) against the *literal blob* content from `git show`, so the "ignoring uncommitted .gitallowed edits" notice fires even when nothing is actually uncommitted. Narrow (nobody has a legitimate reason to symlink `.gitallowed`), not security-relevant (findings are still correctly reported), not a documented AC. **No test added** — named here as a known, low-priority quirk.
- **Base branch name with a slash** (`release/1.0`): confirmed correct via both `refs/heads/release/1.0` and `refs/remotes/origin/release/1.0` resolution paths, and the report line correctly shows the full slash-containing name. Red-checked against a plausible regression (`base_name="${base_name%%/*}"`, a naive "split on first slash" mutation) — the existing test suite did **not** catch this (no assertion exercised a slash-containing base name before this cycle). **Test added**: `test_base_name_with_slash` in `tests/test-secret-scan-branch.sh` (4 assertions), confirmed red against the mutation, green at HEAD, reverted.
- **HEAD equal to merge-base but on a differently-named branch (empty range)**: already covered by the existing `test_on_base_branch_and_empty_range` test (the "empty range" half specifically checks a branch named `feature` pointed at the same commit as `main`).
- **A finding surviving a file rename across commits**: confirmed correct (a secret added under one filename, then `git mv`'d in a later commit, is still detected). This is entirely `secret-scan.sh`'s own pre-existing `git log -p` mechanism (unchanged by this diff, out of scope per the plan's Non-goals) — no gap, no test added.
- **`run-verify.sh` in `test` mode never invoking the script even when it would fail**: the pre-existing `test_mode_routing_clean` only proves this on a *clean* branch (absence of the "Running branch secret scan" log line). Red-checked a case-statement mutation (`static|all)` → `static|all|test)`) — the pre-existing clean-branch assertions already caught it (the log-line check is unconditional on branch content), but there was no assertion proving the *exit code* stays correct on a *secret-carrying* branch in test mode specifically. **Test added**: `test_mode_test_never_runs_even_with_a_finding` in `tests/test-run-verify-branch-secret-scan.sh` (3 assertions), confirmed it independently discriminates the same mutation, reverted.
- **Concurrent runs of the script in the same repo**: see "Repeat and concurrency" above — clean, no test added (inherent `mktemp` uniqueness, not specific application logic).

## Coverage

No instrumented shell coverage tool exists in this repo (per project convention — coverage for shell is measured by test-case scope, see project memory). Go coverage for the three packages touched by this plan's Go-side changes (`internal/scaffold`, `internal/upgrade`, `internal/cli`) was not separately re-measured this cycle since this plan's Go-side change is a single list-literal addition (`internal/scaffold/embed_test.go`'s `TestTemplateBaseScriptsExist`) with no new branches; `go test` passing for all three packages is the meaningful signal here, not a percentage.

## Failure analysis

No test failures against HEAD's actual production code in any of the "before additions" or "after additions" full-suite runs, the environment matrix (after the hermeticity fix), the repeat/concurrency checks, or the live demonstration. The only failures observed were the expected red-phase failures of the 6 deliberate mutations (6a–6e-2) and their pre-existing assertions, plus the pre-fix environment-matrix variant (f) failure — all diagnosed above, all resolved (mutations reverted; variant (f) fixed at the test-fixture level).

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Self-review M1 (docs-only-looking run with a finding closes on a reassuring summary) | Still fixed, pinned | `test_finding_fails_even_with_ran_any_zero` passes at HEAD, fails under mutation 6e-1 |
| Self-review M2 (strict exit 0 could mean "nothing to scan", not just "scanned, clean") | Still fixed, pinned | `test_strict_closes_base_override_bypass` / `test_strict_closes_local_base_fast_forward_bypass` pass at HEAD, both fail under mutation 6a |
| Self-review LOW D3 (skip var must be exactly `"1"`) | Still fixed, pinned | `test_skip_env_var` (`=0`, `=true` cases) passes at HEAD, fails under mutation 6e-2 |
| PR #168's own shape (a self-matching, non-bracketed `.gitallowed` line forward-fixed in a later commit) | Correctly still a finding | `test_allowlist_self_match` passes at HEAD; independently reconfirmed live in the scratch-repo demonstration, step 3 |

## Test gaps

1. **Hermeticity gap in the two new test files (fixed this cycle).** Neither file isolated `HOME`/`GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM` before running `git commit` in its fixture repos. A host or CI runner with a global `commit.gpgsign = true` breaks every `git commit` in both files with an unrelated gpg-signing error (exit 128) — this is **the test's problem, not the environment's**: this repo already has an established pattern for exactly this class of interference (`tests/test-terraform-gitignore.sh`'s hermetic-`HOME` isolation), and the two new files simply didn't follow it. Fixed by adding the same isolation (throwaway `HOME` + `GIT_CONFIG_GLOBAL=/dev/null`-backed `GIT_CONFIG_SYSTEM`) to both files' setup; re-verified clean under the hostile config afterward, and unaffected in every other environment variant.
2. **`scripts/check-template.sh`'s own `required_files` list has no regression coverage (confirmed, not closed).** Mutation 6f showed that silently dropping an entry from that list produces zero test failures anywhere in the suite — `test-template-purity.sh` tests a different thing (meta-repo-reference leakage into templates), and `internal/scaffold/embed_test.go`'s `TestTemplateBaseScriptsExist` is a separate, independently-maintained list in Go that happens to also name `xreview-helpers.sh` but isn't wired to `check-template.sh`'s shell-side list at all. This is a **pre-existing gap in the check-template.sh contract itself**, not introduced by this plan (the list has never had dedicated shell-test coverage) — this plan's own addition (`scripts/xreview-helpers.sh` to the list, per the self-review LOW fix) is therefore also not independently regression-tested by anything other than mutation testing. Out of this plan's stated scope (no AC covers testing `check-template.sh`'s own required-files contents), not fixed here — flagged for whoever next touches `check-template.sh` or its required-files list.
3. **`.gitallowed`-as-a-symlink cosmetic quirk (named, not fixed).** See "Edge cases" above — the "ignoring uncommitted .gitallowed edits" notice can fire on a symlinked `.gitallowed` even with nothing actually uncommitted, because the worktree-diff check dereferences the symlink while the HEAD-content check does not. Not security-relevant (scan results stay correct) and not a named AC; low priority.

## Secret hygiene (item 9)

- After every mutation was reverted and `git status --porcelain` confirmed clean (only the two intended test files staged), ran:
  - `./scripts/secret-scan-branch.sh --strict` → `secret-scan-branch: scanned a0d5bf5..ca3d824 against origin/main: clean`, exit 0.
  - `./scripts/secret-scan.sh --range "$(git merge-base HEAD origin/main)..HEAD"` → exit 0, no output.
  - `./scripts/secret-scan.sh --staged` (after staging the two test files, before commit) → exit 0, no output.
- No secret-shaped literal appears anywhere in this report; all fixture values in the transcripts above were assembled at command runtime from split pieces, matching the test files' own discipline.

## Verdict

- **Pass.** Every command in the handoff's numbered list (1–9) was run; the full shell + Go suite is green both before and after this cycle's additions (597 shell `PASS:` lines in the final run, 3/3 Go packages). The environment matrix passed 7/8 variants outright; the 8th (global `commit.gpgsign=true`) was a genuine test-only hermeticity gap, fixed in this cycle (not a production-code issue — `scripts/secret-scan-branch.sh` and `scripts/run-verify.sh` themselves never touched `HOME` or global git config). Repeat (10/10) and concurrency (12/12 across two shapes) checks were clean. 6 of 6 requested red/green mutations discriminated correctly; one of them (6f) additionally surfaced a genuine, pre-existing gap in `check-template.sh`'s own test coverage, named above but out of this plan's scope to close. Two new tests were added (`test_base_name_with_slash`, `test_mode_test_never_runs_even_with_a_finding`), both confirmed red/green-discriminating before being kept. No production script (`scripts/secret-scan-branch.sh`, `scripts/run-verify.sh`, `scripts/secret-scan.sh`, `scripts/check-template.sh`) differs from `ca3d824` — the diff at the end of this cycle is additive-only across the two test files (`tests/test-secret-scan-branch.sh`, `tests/test-run-verify-branch-secret-scan.sh`).
- Fail: none.
- Blocked: none.
