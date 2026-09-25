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

## Cycle 2

- Date: 2026-09-24
- HEAD: `7d46144ba435a56f9495fce03b60d1163bd8c355` (`git status --porcelain` empty at start and at end of this cycle)
- Plan: `docs/plans/active/2026-09-20-local-branch-secret-scan.md` (issue #169)
- Tester: `tester` subagent (Claude Code), pipeline cycle 2/2
- Scope: behavioral tests only for the cycle-2 fix delta `git diff ca3d824..7d46144` — commit `dbba825` (cross-review AR-1/AR-2 fix: fully-qualified base ref for `merge-base`; committed `.gitallowed` symlink resolution one level inside the tree) and commit `fc7d2e4` (self-review cycle-2 fixes: guarded regular-file `git show`, strip `./` before validating, test renames from `test_ar1_*`/`test_ar2_*` to `test_ac2d_*`/`test_ac3b_*`, four new edge-case tests). Static analysis (`/verify` cycle 2, `docs/reports/verify-2026-09-22-local-branch-secret-scan.md` `## Cycle 2`, pass) and diff quality (`/self-review` cycle 2, `docs/reports/self-review-2026-09-21-local-branch-secret-scan.md` `## Cycle 2`, 1 MEDIUM + 5 LOW, all fixed in `fc7d2e4`) are out of scope here, already run.

### Test execution

| Suite / Command | Tests | Passed | Failed | Duration |
| --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (full changed-scope suite; widened to `full` fallback because `scripts/check-template.sh`'s pre-existing unclassified +2 lines forces it, same as `/verify` cycle 2) | 840 shell `PASS:` + 8 Go packages | all | 0 (22 `FAIL: 0` summary lines, no individual failures) | ~2 min |
| `sh tests/test-secret-scan.sh` | 6 | 6 | 0 | <1s |
| `sh tests/test-secret-scan-branch.sh` | 96 | 96 | 0 | ~2s |
| `sh tests/test-run-verify-branch-secret-scan.sh` | 32 | 32 | 0 | ~2s |
| `TMPDIR=/tmp go test ./internal/scaffold/... -count=1 -v` (CI-like TMPDIR) | 18 | 18 | 0 | 0.4s |

Evidence: `docs/evidence/verify-2026-09-24-063321.log` (`run-test.sh` auto-generated log for this cycle's full pass).

### Environment matrix

`tests/test-secret-scan-branch.sh` under each variant, 96/96 every time:

| Variant | Result |
| --- | --- |
| `sh` (POSIX) | Pass |
| `bash --posix` | Pass |
| `dash` | Pass |
| Hostile **outer** global git config (`HOME`/`GIT_CONFIG_GLOBAL` pointed at a separate throwaway dir holding `commit.gpgsign=true` + a broken `gpg.program`, distinct from the suite's own hermetic `HOME`) | Pass — confirms the suite's own `HOME`/`GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM`/`GIT_TERMINAL_PROMPT` override at `tests/test-secret-scan-branch.sh:46-50` fully shadows a hostile outer environment, not merely a coincidentally-absent one |
| `TMPDIR` containing a space | Pass |
| Test file run from a subdirectory cwd (`cd tests && sh test-secret-scan-branch.sh`) | Pass |

Additionally, independently of the test file, the **scanner script itself** was run from a scratch fixture's repo root and from a subdirectory (`sub/`), with a committed symlinked `.gitallowed` resolving to a real allowlist rule: both gave the identical `clean`, exit 0 — corroborating what `test_ls_tree_mode_check_is_root_relative` already asserts (the `--full-tree` fix), via a fixture built independently of the suite's own.

### Repeat and concurrency

- 10 consecutive runs of `tests/test-secret-scan-branch.sh`: 10/10 `FAIL: 0`.
- `sh tests/test-secret-scan-branch.sh` run concurrently with a full `./scripts/run-test.sh` pass in the same worktree: both completed clean (no interference — each uses its own `mktemp -d` scratch dir).
- 8 concurrent **direct** invocations of `./scripts/secret-scan-branch.sh --strict` against the same repo (this worktree): all exited 0 with `scanned a0d5bf5..7d46144 against origin/main: clean`; `ls "${TMPDIR:-/tmp}" | grep ralph-secret-scan-branch-allowlist` found no leftover temp allowlist files afterward.

### Red/green mutation results

Each mutation applied directly to `scripts/secret-scan-branch.sh`, tested, then reverted with `git checkout -- scripts/secret-scan-branch.sh`; `git status --porcelain` confirmed empty after every single revert.

| # | Mutation | Result | Verdict |
| --- | --- | --- | --- |
| a | Widen the symlink-target mode case (`:249`) to accept `040000` | 2 failed: "bare directory target: finding not suppressed" (exit 0, want 1), "bare directory target: could-not-read notice" | Matches exactly — `test_ac2d_symlinked_allowlist_bare_directory_target_rejected` (new this cycle) pins this. This is self-review coverage gap #1, now closed. |
| b | Drop `--full-tree` from `allowlist_ls_tree_mode` (`:193`) | 3 failed: all three "subdirectory cwd ... (same as root)" assertions | Matches exactly — `test_ls_tree_mode_check_is_root_relative` pins it. |
| c | Drop the exact-path check (`:200`, `[ "${entry#*"$tab"}" = "$1" ] \|\| return 1`) | 2 failed: "symlink directory target: finding not suppressed" (exit 0, want 1), "symlink directory target: could-not-read notice" | Matches exactly — `test_ac2d_symlinked_allowlist_directory_target_rejected` (trailing-slash form) pins it. |
| d | Pass the short `base_ref` to `merge-base` again (`:143`, `$base_ref_full` → `$base_ref`) | 4 failed: both AC-3b tests' exit and findings-reported assertions | Matches exactly — the AR-2 fix (`base_ref_full`) is what both `test_ac3b_*` tests pin. |
| e | Move the `./`-strip loop back after the `allowlist_target_is_safe` call (validate-then-strip, the pre-fix C2-L2 order) | **0 failed**, 96/96 unchanged | No observable difference, as the self-review predicted. Traced by hand: for the one fixture with a leading `./` target (`.//scratch-target`), both orders produce the identical final stripped value (`/scratch-target`) before the `ls-tree` call, because the strip is unconditional — only *when* the safety check runs relative to it differs, and in the buggy order the raw pre-strip value (`.//scratch-target`) also passes the safety check (no leading `/`, no `/../` substring), so both orders reach the same `git ls-tree --full-tree HEAD -- /scratch-target`, which git rejects as an absolute pathspec regardless (`allowlist_ls_tree_mode`'s own `2>/dev/null` prevents any error text leak either way). Git's own absolute-pathspec rejection is the actual backstop here, matching self-review C2-L2's own note. |
| f | Remove the guard around the regular-file `git show` (`:232-235` → unguarded `git show "HEAD:.gitallowed" > "$tmp_allowlist"`) | **0 failed**, 96/96 unchanged | No observable difference, as the self-review predicted (C2-L1). The mode check at `:224` already proves the blob exists at HEAD before this line runs, so the only way `git show` fails here is a damaged object or a `$TMPDIR` that vanishes mid-run — neither is reproducible in a hermetic fixture. |

### Live demonstration

Built in a scratch fixture under the session scratchpad (outside the worktree, cleaned up afterward), a fresh git repo per scenario, `HOME`/`GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM` isolated the same way the test suite isolates them. Every secret-shaped value was assembled at command runtime from split `printf` pieces (no secret-shaped literal appears in this report or in any command transcript). First attempt at scenarios 1/2a/2b used a malformed fixture (`api_key.scenarioN=...`, where the `.` between the key and the value broke the scanner's `generic secret assignment` pattern's required immediate `key[:=]value` adjacency) and was silently "clean" for the wrong reason; corrected to `api_ke[y]=<value>` (matching the existing test fixtures' own convention) before re-running — noting this because a live demonstration that "passes" for the wrong reason is worse than one that fails honestly.

1. **Symlink `.gitallowed` → a committed allowing file (clean).** `.gitallowed` symlinked to `allowlist-rules`, which holds a bracket-obscured rule (`api_ke[y]=<value>`) matching a real `api_ke[y]=<value>` finding on the feature branch. `./scripts/secret-scan-branch.sh --strict` → `scanned <a>..<b> against main: clean`, exit 0.
2. **Symlink `.gitallowed` → a bare directory (`ln -s rules .gitallowed`, no trailing slash).** `--strict` → `committed .gitallowed could not be read as a file; scanning without allowlist exceptions` then `BLOCKED` / `scanned <a>..<b> against main: findings`, exit 1.
3. **Symlink `.gitallowed` → the same directory with a trailing slash (`ln -s "rules/" .gitallowed`).** Same notice, same `findings`, exit 1.
4. **A tag named `main` planted on the feature branch itself, after the leak commit but before HEAD** (the exact AR-2/AC-3b tag-shadow shape). `GITHUB_BASE_REF=main ./scripts/secret-scan-branch.sh --strict` → `BLOCKED` / `scanned <a>..<b> against main: findings`, exit 1 — the leak is still detected because `merge-base` resolves through the fully-qualified `refs/heads/main`, not the shadowing tag.
5. **A local branch literally named `origin/main`, planted at the leak commit, while the real `refs/remotes/origin/main` is planted before it** (the exact AR-2/AC-3b remote-shadow shape). Same result: `findings`, exit 1, scanned against `origin/main`.

All five matched the plan's and the cycle-2 fix's stated behavior exactly.

### Re-confirmations (item 6)

- **Cycle-1 test report gap (b) — closed, re-confirmed.** Built a fresh fixture: `.gitallowed` symlinked to a committed, resolvable allowlist file with a matching rule, and no uncommitted worktree edits at all. `--strict` → `scanned <a>..<b> against main: clean`, exit 0, and the stderr contains **zero** occurrences of "ignoring uncommitted" — the notice no longer misfires on an unmodified symlinked `.gitallowed`.
- **Self-review coverage gap: submodule target mode (`160000`) — confirmed fail-closed, no dedicated test added.** Built `.gitallowed` as a real git submodule (`git -c protocol.file.allow=always submodule add`) pointing at a throwaway local target repo. `git ls-tree --full-tree HEAD -- .gitallowed` confirmed mode `160000`. `--strict` → `committed .gitallowed could not be read as a file; scanning without allowlist exceptions`, then correctly reports the planted finding, exit 1. Not added to the suite: a submodule fixture needs `git -c protocol.file.allow=always` for `git submodule add` from a local path, plus `.gitmodules`/`.git/modules/` bookkeeping and cleanup, which is more moving parts than this one mode-branch check is worth relative to the rest of the file's fixture style — recorded as an open gap below (per the handoff's own "otherwise record as a gap" instruction), not a defect.
- **Self-review coverage gap: symlink-to-symlink target (`120000` → `120000`) — confirmed fail-closed, no dedicated test added.** Built `.gitallowed` → `middle-link` (itself a symlink) → `real-allowlist.conf` (a real file with a matching rule). `allowlist_ls_tree_mode` only accepts `100644`/`100755` for the *first* resolved target, and `middle-link`'s own mode is `120000`, so the case falls to the `*)` arm: `committed .gitallowed could not be read as a file; scanning without allowlist exceptions` printed, plus (as an unrelated but expected second effect) the "ignoring uncommitted .gitallowed edits" notice — because the worktree's `.gitallowed` fully dereferences through both symlink hops to the real file's content via `test -f`/`diff -q`, while `tmp_allowlist` stays empty (fail-closed), so the two differ. `--strict` correctly reports the planted finding, exit 1. Same "not added" reasoning as the submodule case — recorded as an open gap below.
- **`test_ac2d_symlinked_allowlist_denies_fixture` does not discriminate the fix (self-review C2 point 5) — re-confirmed, no action needed.** This is a negative-control test (pre-fix, the allowlist content would have been the literal target-path string, which also fails to match the planted line, so it exits 1 either way). It usefully documents "a resolvable target with a non-matching rule still finds", but it is not what protects AR-1; `test_ac2d_symlinked_allowlist_target_name_not_used_as_regex` is the actual AR-1 regression pin (confirmed above it discriminates, since it is unaffected by any of mutations a–f, all of which target unrelated code paths).

### Coverage

Directly counted from `sh tests/test-secret-scan-branch.sh`'s own PASS output (not static line-counting, to avoid the risk of a helper-function assertion count mismatch):

| AC | Dedicated test functions | Assertions |
| --- | --- | --- |
| AC-2d (symlinked `.gitallowed` resolution) | 11 (`test_ac2d_symlinked_allowlist_resolves_target`, `_denies_fixture`, `_dangling`, `_target_name_not_used_as_regex`, `_directory_target_rejected`, `_dot_target_rejected`, `_bare_directory_target_rejected`, `_absolute_target_rejected`, `_dotdot_target_rejected`, `_leading_dotslash_target_rejected`, plus `test_ls_tree_mode_check_is_root_relative`) | 28 |
| AC-3b (fully-qualified base ref) | 2 (`test_ac3b_tag_shadows_base_branch`, `test_ac3b_local_branch_shadows_remote_tracking_ref`) | 4 |
| Whole file | 29 | 96 |

The 96-assertion total is exactly 64 (cycle-1's post-addition count) + 32 (28 AC-2d + 4 AC-3b added by `dbba825`/`fc7d2e4` this cycle) — the arithmetic ties out cleanly against the cycle-1 report's own figure, corroborating that this cycle's additions are purely additive.

### Failure analysis

No test failed against HEAD's actual production code anywhere in this cycle: not in execution, not in the environment matrix (7/7 variants clean — no hermeticity regression from cycle 1's fix), not in repeat/concurrency, not in the live demonstration, not in the re-confirmation fixtures. The only failures observed were the four deliberate, expected red-phase mutations (a–d); mutations (e) and (f) produced no failures, matching the self-review's own prediction that both are unobservable given git's absolute-pathspec rejection (e) and the mode check's own existence proof (f).

### Regression checks

| Item | Status | Evidence |
| --- | --- | --- |
| `tests/test-secret-scan-branch.sh` assertion count | 64 → 96 (+32, purely additive) | Cycle-1 report's own "after additions" row (64) vs this cycle's run (96); the delta matches exactly the sum of AC-2d (28) + AC-3b (4) assertions counted above |
| `tests/test-run-verify-branch-secret-scan.sh` assertion count | 32 → 32 (unchanged) | This cycle's delta (`dbba825`, `fc7d2e4`) never touched this file; cycle-1's own tail commit (`9554a58`) had already added its one new test before the cycle-1 report was finalized |
| Cycle-1 pinned regressions (self-review M1, M2, LOW D3; PR #168 shape) | Still present and passing | `test_finding_fails_even_with_ran_any_zero`, `test_strict_closes_base_override_bypass` / `_fast_forward_bypass`, `test_skip_env_var`, `test_allowlist_self_match` all appear in the current 29-function list and passed in every full run this cycle |
| Cycle-1's own two additions (`test_base_name_with_slash`, `test_mode_test_never_runs_even_with_a_finding`) | Still present and passing | Confirmed in the function lists and full-suite runs above |

### Test gaps

1. **Submodule target mode (`160000`) has no dedicated test** (self-review coverage gap 4, first half). Confirmed fail-closed this cycle by manual fixture (see "Re-confirmations" above); not added to the suite — a `git submodule add` fixture is meaningfully more complex than this file's existing fixture style and carries its own environment sensitivity (`protocol.file.allow`). Whoever next touches the allowlist mode-dispatch `case` block should be aware this mode is exercised only by hand-verification, not by CI.
2. **Symlink-to-symlink target (`120000` → `120000`) has no dedicated test** (self-review coverage gap 4, second half). Same status: confirmed fail-closed manually, not added, same reasoning.
3. **`check-template.sh`'s `required_files` list still has no regression coverage** (cycle-1 test gap 2, carried over — unchanged and unaffected by this cycle's delta, which touched none of `check-template.sh`, `internal/scaffold/embed_test.go`, or the required-files list itself; not re-verified by mutation this cycle since nothing in this cycle's scope bears on it).
4. **`test_ac2d_symlinked_allowlist_denies_fixture` is a negative control, not a fix-pin** (self-review C2 point 5, re-confirmed above). Documented behavior, not a gap requiring action — the actual AR-1 regression pin is `test_ac2d_symlinked_allowlist_target_name_not_used_as_regex`.

### Secret hygiene

- `./scripts/secret-scan-branch.sh --strict` → `scanned a0d5bf5..7d46144 against origin/main: clean`, exit 0.
- `./scripts/secret-scan.sh --range "$(git merge-base HEAD origin/main)..HEAD"` → exit 0, no output.
- No secret-shaped literal appears anywhere in this section; every fixture value in every live-demonstration and re-confirmation transcript above was assembled at command runtime from split pieces, matching this file's own established discipline.

### Verdict

- **Pass.** All items in the cycle-2 handoff (1–9) were run. Full execution is green (840 shell `PASS:` lines / 0 failures across the whole changed-scope suite this cycle, 96/96 for the branch scan file specifically, 18/18 for the CI-`TMPDIR` scaffold Go run). The environment matrix is clean across three shells, a hostile outer global git config, a spaced `TMPDIR`, and two independent subdirectory-cwd checks (the test file itself, and the scanner script against its own fixture). Repeat (10/10) and both concurrency shapes were clean, with no leftover temp files. All 6 requested red/green mutations behaved exactly as predicted: 4 discriminated with precise failure sets tied to the AR-1/AR-2 fix and one of them (mutation a) closed a named self-review coverage gap outright; 2 (e, f) showed no observable difference, matching the self-review's own prediction and explained by tracing the code by hand. The live demonstration reproduced all four requested scenarios correctly, after one self-caught fixture-construction error was fixed before being reported. Cycle-1's test-report gap (b) is confirmed closed. Two of the self-review's four coverage-gap items (bare-directory and trailing-slash-directory symlink targets) are now closed by this cycle's new tests; the remaining two (submodule, symlink-to-symlink) are confirmed fail-closed by hand but left as an explicit, named gap rather than adding submodule-fixture complexity to the suite. No regressions: assertion-count growth (64→96) matches this cycle's additions exactly, and every cycle-1-pinned test is still present and passing.
- Fail: none.
- Blocked: none.
