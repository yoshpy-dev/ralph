# Test report: check-template-scaffold-green

- Date: 2026-10-01
- Plan: docs/plans/active/2026-09-30-check-template-scaffold-green.md (issue #189)
- Tester: tester subagent (Claude Code), cycle 1
- Scope: behavioral tests only (`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh`, `tests/test-check-template.sh`, `go test ./internal/scaffold/...`), plus independent shell-matrix runs, PR-CI simulation, mutation re-testing, and signal handling. No static analysis (that is `/verify`'s job; see `docs/reports/verify-2026-10-01-check-template-scaffold-green.md`, PASS, cycle 1).
- Evidence: this report's own command output (captured inline below); full-scope run wrote `docs/evidence/verify-2026-09-30-164004.log` (test mode) inside the worktree.

Worktree: `.claude/worktrees/check-template-scaffold-green` (branch `fix/check-template-scaffold-green`), HEAD `842856a` confirmed at start and end of this cycle, `git status --porcelain` empty throughout in both the worktree and the main checkout.

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` (all shell suites + all 8 Go packages) | full repo test scope | all green | 0 | 0 | ~ (exit 0; `test-check-template.sh` section: 40/0/0) |
| `bash tests/test-check-template.sh` x3 | 40 each run | 40 | 0 | 0 | sub-second each |
| `go test ./internal/scaffold/... -count=1 -v` | 27 | 27 | 0 | 0 | 0.421s |

`RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` ran every shell test file in `tests/` plus the Go suite (8 packages, all `ok`, `internal/org` 9.849s, rest cached) and exited 0 with no `FAIL:` anywhere in the log. `test-check-template.sh`'s three repeat runs were bit-for-bit identical: `40 passed, 0 failed, 0 skipped`, exit 0. Non-root execution (uid 502) means cases E's and F's root-skip branches (unreadable-settings.json, unreadable-subtree) ran for real rather than skipping, matching the verify report's own note.

## Shells table

| Shell | Target | Result |
| --- | --- | --- |
| `sh` (`/bin/sh`) | worktree root, `CI=true` | exit 0, no `FAIL:` line, `Template structure looks good.` |
| `dash` (`/bin/dash`) | worktree root, `CI=true` | exit 0, no `FAIL:` line |
| `bash --posix` | worktree root, `CI=true` | exit 0, no `FAIL:` line |
| `sh` | fresh scaffold (`go run ./cmd/ralph init --yes <scratch>/fresh1`) | exit 0, no `FAIL:` line |
| `dash` | fresh scaffold | exit 0, no `FAIL:` line |
| `bash --posix` | fresh scaffold | exit 0, no `FAIL:` line |
| busybox sh | — | not available on this machine (`command -v busybox` empty); skipped, noted as a gap below |

The fresh scaffold was built once via `go run ./cmd/ralph init --yes <scratch>/fresh1` (0.28s wall, cached build) and reused for both the shell matrix and the CI simulation below, matching the plan's own guidance that the `go run` build should happen once.

## CI simulation

Simulated `templates/base/.github/workflows/verify.yml`'s "Check template structure" step (`run: ./scripts/check-template.sh`, triggered `on: pull_request`, where GitHub Actions sets `CI=true` in the runner environment for every step) inside the fresh scaffold:

1. `git init` (already done by `ralph init`), `git add -A`, `git -c user.email=... -c user.name=... commit -m "chore: initial scaffold"` — succeeded on the second attempt after the scaffold's own installed `commit-msg-guard` hook rejected a non-conventional first message (`initial scaffold`); `chore: initial scaffold` passed.
2. `CI=true ./scripts/check-template.sh` from the scaffold root (the exact form of the workflow's `run:` line) — **exit 0**, `Template structure looks good.`, no `FAIL:` line.

This directly exercises AC-3 end to end through the same code path the shipped workflow uses, not just through the test suite's own invocation of that workflow's logic.

## Mutation table

All six mutations were applied only to scratch copies inside a `git archive HEAD`-built mirror (`<scratch>/mut`, tracked worktree files at `scripts/check-template.sh` / `templates/base/scripts/check-template.sh` never touched — confirmed by `git status --porcelain` and `cmp` after every mutation). Baseline on the mirror: `40 passed, 0 failed, 0 skipped` (identical to the real worktree). Each mutation was restored before the next.

| # | Mutation | Predicted (plan) | Observed | Match |
| --- | --- | --- | --- | --- |
| a-1 | Scripts-executable loop reverted to `find ... \| while read` (subshell) | breaks case F | `38 passed, 2 failed`: F1 (non-executable script not detected, exit 0) + F5 (unreadable subtree not reported, exit 0) | Yes |
| a-2 | Skills loop reverted to `find ... \| while read` | breaks case F | `39 passed, 1 failed`: F2 (skill missing SKILL.md not detected, exit 0) | Yes |
| a-3 | Agents loop reverted to `find ... \| while read` | breaks case F | `39 passed, 1 failed`: F3 (agent missing `tools:` not detected, exit 0) | Yes |
| b | Hook check reverted to `grep \| tr \| while` (subshell, full command string as key, no dedup) | breaks case E | `35 passed, 5 failed`: D (root, 7 spurious `FAIL:` lines yet exit 0) + all 4 E sub-cases (existing-hook-with-arg now flagged, missing-hook not detected, dispatcher dedup 3 lines not 1, unreadable-settings not reported) | Yes, and reproduces the plan's own documented fail-open bug exactly |
| c | Drop `sort -u` from the hook-path pipeline | breaks the dedup sub-case | `39 passed, 1 failed`: only "a dispatcher missing across several events produces exactly one FAIL line" (got 3) | Yes |
| d | Restore `2>/dev/null \|\| true` on the scripts `find` (swallow its exit code) | breaks the unreadable-subtree case | `39 passed, 1 failed`: only F5 (unreadable subtree not reported) | Yes |
| e | Drop the settings.json read-check (`rc=$?` / `[ "$rc" -gt 1 ]` guard) | breaks the unreadable-settings case | `39 passed, 1 failed`: only the unreadable-settings.json sub-case | Yes |
| f | Re-add `README.md` to `required_files` | breaks cases A and G | First pass (mutating only `scripts/check-template.sh`): `36 passed, 4 failed` — A failed as predicted, but **G passed** (the fresh-scaffold case builds via `go:embed` from `templates/base/scripts/check-template.sh`, unaffected), while B and one sub-case each of E and F failed as an unpredicted side effect (`build_fixture` never creates `README.md`, so any fixture-based case now hits "Missing required file: README.md" too). Second pass, mutating **both** copies (matching how the real diff keeps them byte-identical): `35 passed, 5 failed` — A and G both failed as predicted, plus the same B/E/F side effects | Partial on the first pass; full match once both copies were mutated (see Findings) |

No mutation survived undetected. Mutation (f)'s nuance is discussed below.

## Signal handling

Fixture: a synthetic project (all 24 `required_files` scripts present as stubs, plus 50,022 extra executable `*.sh` files under `scripts/`) sized so an unmutated run takes ~1.8s wall (`1.06s user 0.70s system`, 50,024 files total). `sh`/`dash` were each started, sent `SIGTERM` at three different delays (0.1s, 0.5s, 1.0s) mid-run, and checked against a baseline sweep of `${TMPDIR:-/tmp}` for any `check-template.*` leftover:

| Shell | Delay | Exit code | Leftover `check-template.*` dirs |
| --- | --- | --- | --- |
| sh | 0.1s | 143 | 0 |
| sh | 0.5s | 143 | 0 |
| sh | 1.0s | 143 | 0 |
| dash | 0.1s | 143 | 0 |
| dash | 0.5s | 143 | 0 |
| dash | 1.0s | 143 | 0 |

All six runs: exit 143 (`128 + SIGTERM`), zero leftover temp directories, both before and after each run. This directly confirms the self-review's finding 9 fix (dash previously left its `mktemp -d` temp dir behind on `SIGTERM`; the added `trap 'exit 143' TERM` now runs `cleanup` via the `EXIT` trap chain on all three shells tested).

## Findings

1. **Mutation (f) needs both copies mutated to match its own prediction, not a script defect.** `required_files` and the hook-check loops are all local to `scripts/check-template.sh`; the fresh-scaffold case (G) instead exercises `templates/base/scripts/check-template.sh` through `go:embed`, since `ralph init` never reads the root-level script. A mutation that only touches the root copy (the most literal reading of "mutations on scratch copies of scripts/check-template.sh") makes G pass and therefore understates the mutation's real blast radius; mutating both copies together (matching how the actual diff keeps them byte-identical, and how a real revert of this fix would behave) reproduces the plan's prediction exactly. This is a property of AC-4's byte-identity requirement working as designed, not a gap in the check itself — noted here only because the first pass's result could otherwise be misread as the check missing the fresh-scaffold regression.
2. **Mutation (f)'s side effects (B, one sub-case each of E and F) are expected, not noise.** `required_files` is read directly by the script; `build_fixture` in the test file builds fixtures from the test's own hardcoded `GOLDEN_ENTRIES`, not from the script's `required_files`. Re-adding an entry to the script's list without updating the fixture builder necessarily makes every fixture-based case fail on "Missing required file: README.md" before it reaches whatever else that case is testing. This is exactly the kind of drift the 4-location-sync discipline documented in the plan's own "調査で確認したこと" section exists to prevent, and it is why case A (the golden-list comparison) is the AC's primary signal rather than a coincidental one.
3. **Case D staying green under mutation (f) is a fixture property, not a blind spot.** The meta-repo's own root already has a `README.md`, so re-adding it to `required_files` cannot make the live root-sanity check (case D) fail — the repo genuinely satisfies that stricter requirement by accident of already having the file. Case A (comparing against the hardcoded `GOLDEN_ENTRIES`, which does not include `README.md`) is what actually catches the drift; this is consistent with the plan's own design decision to keep the meta-repo-only exclusions as a hardcoded list rather than environment-conditional logic.
4. **busybox is unavailable on this machine** (`command -v busybox` empty), so that leg of the requested shell matrix could not be run. Not attributable to the change under test; flagging as an environment gap rather than a defect.
5. No leftover fixtures, scratch mutations, or processes remain. All scratch work lived under the session scratchpad and the fresh scaffold(s); none were created inside the worktree or the main checkout, and `git status --porcelain` was empty in both at the end of every phase.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Hook-reference check silently passed (fail-open) while printing 7 spurious `FAIL:` lines at the meta-repo root, because `| while` ran in a subshell | Fixed | Mutation (b) reproduces the exact old symptom (7 `FAIL:` lines, exit 0) when reverted; current code exits 0 with zero `FAIL:` lines (item 3 above) |
| Fresh `ralph init` scaffold's own PR CI failed on its first PR (`required_files` demanded `README.md` and two meta-repo-only docs that scaffolds never receive) | Fixed | Live CI-step simulation (above) plus test case G, both exit 0 with no `FAIL:` line; mutation (f) reproduces the regression when README.md is re-added to both copies |
| Three find-driven loops (executable check, SKILL.md check, agent frontmatter check) did not propagate `fail()`'s `status=1` to the exit code when run as `$(find ...)` word-split over a pipe/subshell | Fixed | Mutations a-1/a-2/a-3 each reproduce a silent-pass regression in exactly the loop they target and no other; live fixture cases F1-F3 pass on the current code |
| dash left its `mktemp -d` temp directory behind on `SIGTERM` | Fixed | Signal-handling matrix above: 0 leftover dirs across 6 sh/dash x delay combinations |

## Test gaps

- **busybox sh** was not exercised (not installed on this machine). All three POSIX-family shells that are available (`sh`, `dash`, `bash --posix`) passed identically at both the worktree root and the fresh scaffold.
- **AC-7's own mutation claim was independently re-verified here** (three separate find-loop mutations, item a-1/a-2/a-3 above), closing the one item the verify report explicitly deferred to `/test` ("AC-7's own mutation claim ... was not independently re-mutated here").
- The suite's own root-skip branches (case E's unreadable-`settings.json`, case F's unreadable-subtree) were exercised for real in this non-root environment; no independent evidence was gathered for root-user behavior (the suite's own `skip()` path for that case is trusted as documentation, not re-verified against an actual root run, since running as root was out of scope and not requested).
- No dedicated automated test exists for `SIGTERM`/`SIGINT`/`SIGHUP` handling inside `tests/test-check-template.sh` itself (the plan's own Deviation notes record this as a known, deliberate gap — timing-dependent tests are flaky). This cycle's manual signal matrix (6 runs, 2 shells x 3 delays, 0 leftovers) is a one-time confirmation, not a standing regression guard; a future regression in the signal traps would not be caught by the automated suite.

## Verdict

- **Pass.** `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` exit 0 (all shell suites and all 8 Go packages green); `tests/test-check-template.sh` 40/0/0 across 3 repeat runs; `go test ./internal/scaffold/...` 27/0; all three available shells (`sh`, `dash`, `bash --posix`) pass cleanly at both the worktree root and a fresh scaffold; the scaffold's real PR-CI step simulates to exit 0; all 6 requested mutations discriminate correctly (with the one documented nuance on mutation (f) requiring both copies mutated to match its full predicted blast radius — not a defect); signal handling confirmed clean (exit 143, 0 leftover temp dirs) across 6 sh/dash runs at varying delays on a 50k-file fixture.
- Fail: none.
- Blocked: none.
- Worktree and main checkout both had empty `git status --porcelain` throughout; HEAD unchanged at `842856a`.
