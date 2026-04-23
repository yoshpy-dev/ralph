# Test report: colorize-upgrade-diff

- Date: 2026-04-23
- Plan: `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/plans/active/2026-04-23-colorize-upgrade-diff.md`
- Tester: tester subagent (`/test`)
- Scope: behavioral tests for commit `cd5dd69` on branch `feat/colorize-upgrade-diff`. Two affected packages (`internal/upgrade`, `internal/cli`). Verified via `./scripts/run-test.sh` (full suite) and `go test ./internal/upgrade/... ./internal/cli/... -count=1 -coverprofile=...`.
- Evidence: `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/evidence/test-2026-04-23-colorize-upgrade-diff.log` (raw test output) + `/Users/hiroki.yoshioka/MyDev/github.com/yoshpy-dev/ralph/docs/evidence/verify-2026-04-23-043941.log` (post-edit verify)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (mode=test) — full project suite | — | all | 0 | 0 | (cached) |
| `tests/test-check-mojibake.sh` (shell) | 11 | 11 | 0 | 0 | <1s |
| `go test ./internal/upgrade/... -v -count=1` | 25 | 25 | 0 | 0 | 4.05s |
| `go test ./internal/cli/... -v -count=1` | 22 | 22 | 0 | 0 | 3.95s |
| `go test ./...` (all Go packages) | — | ok | 0 | 0 | 1.4s+0.8s (cached) |

Per-package breakdown (changed-area focus):

- `internal/upgrade` (25 tests): all `TestColorize_*` (10), all `TestUnifiedDiff_*` (12, including the new `TestUnifiedDiff_GutterWidth_FiveDigitLineNumbers`), all `TestComputeDiffs_*` regression coverage carried forward.
- `internal/cli` (22 tests): the two pivotal integration tests for this change pass — `TestRunUpgrade_InteractiveDiff_ShowsUnifiedDiff` (non-TTY, asserts no `\x1b[`) and `TestRunUpgrade_InteractiveDiff_ColorizesWhenEnabled` (asserts the exact `\x1b[1;31m`, `\x1b[1;32m`, `\x1b[36m` sequences). The unit-level gate `TestShouldColorize_HonorsNoColorAndTTY` is also green.

No flaky behavior observed. No tests skipped.

## Coverage

- Statement (changed files):
  - `internal/upgrade/colorize.go` — **100%** (`Colorize` 100%, `ansiForLine` 100%)
  - `internal/upgrade/unified_diff.go` — `UnifiedDiff` 100%, `formatRange` 100%, `gutterWidth` 100%, `splitLines` 100%, `equalSlices` 100%, `lcsDiff` 100%, `groupHunks` 95.8% (one defensive branch — pre-existing, not touched by this commit)
  - `internal/cli/upgrade.go` (changed lines) — `runUpgrade` 100%, `shouldColorize` 100%, `showDiff` 92.9% (the disk-read-failure fallback is exercised by `TestRunUpgrade_DiskReadFailure_FallsBackToHash`), `resolveConflict` 81.8% (uncovered branch is the EOF/non-interactive path, pre-existing). `runUpgradeIO` 69.7% (uncovered branches are pack-failure-handling paths unrelated to colorize).
- Package totals:
  - `internal/upgrade` — **95.2%** (up slightly with the new test)
  - `internal/cli` — **39.0%** (unchanged — large package with extensive non-colorize surface)
- Notes: 100% coverage on both new helpers (`Colorize`, `shouldColorize`) and on every UnifiedDiff function the commit modified. The two coverage gaps the verifier flagged are now both closed:
  1. `NO_COLOR=1` is asserted at the unit level by `TestShouldColorize_HonorsNoColorAndTTY` (`internal/cli/cli_test.go:921-943`) using `t.Setenv("NO_COLOR", "1")` and asserting `shouldColorize(f) == false`. End-to-end propagation through `runUpgrade → runUpgradeIO → showDiff` is covered by the pair of `TestRunUpgrade_InteractiveDiff_*` tests that exercise both legs of the gate (`colorize=false` produces zero ANSI bytes; `colorize=true` produces the expected sequences). The integration-side `NO_COLOR` path is therefore *equivalent* to the colorize=false path and does not require a duplicate test.
  2. **5+ digit line numbers**: closed by the new `TestUnifiedDiff_GutterWidth_FiveDigitLineNumbers` (`internal/upgrade/unified_diff_test.go:154-184`), which builds a 10,000-line file with a change near line 9999 and asserts the byte-exact 5-column gutter alignment for change lines (`" 9999       │ -X\n"`, `"       9999 │ +Y\n"`) and trailing-context lines (`"10000 10000 │  line\n"`).

## Added tests

- `TestUnifiedDiff_GutterWidth_FiveDigitLineNumbers` (`internal/upgrade/unified_diff_test.go`) — single new test, ~30 lines including a tagged switch to satisfy staticcheck QF1002. Runs in 0.39s. Confirms the dynamic `gutterWidth` algorithm (`unified_diff.go:98-116`) handles the 5-digit boundary without alignment drift.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (none) | — | — | — |

One staticcheck issue (QF1002, "could use tagged switch on i") was triggered by my first draft of the new test and fixed in-place before the report was finalized. Evidence of the fix is in `docs/evidence/verify-2026-04-23-043941.log` (post-fix verify is fully green: shellcheck OK, all `sh -n` OK, both jq OK, check-sync OK, mojibake 11/11, gofmt OK, **0 staticcheck issues**, all `go test` packages OK).

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `UnifiedDiff` non-determinism for the same input | Still fixed | `TestUnifiedDiff_OrderStability` PASS (`internal/upgrade/unified_diff_test.go:97-107`) |
| Disk read failure during `[d]iff` should not panic | Still fixed | `TestRunUpgrade_DiskReadFailure_FallsBackToHash` PASS |
| Invalid prompt input must re-prompt instead of looping | Still fixed | `TestRunUpgrade_InteractiveDiff_RepromptsOnInvalid` PASS |
| Non-TTY destination must not emit ANSI | Newly enforced | `TestRunUpgrade_InteractiveDiff_ShowsUnifiedDiff:608-610` asserts absence of `\x1b[` byte against `bytes.Buffer` |
| `ralph upgrade` end-to-end (auto-update, force, conflict, skip) | Still working | All 8 `TestRunUpgrade_*` PASS |

## Test gaps

- TTY detection (`term.IsTerminal`) cannot be exercised end-to-end without spawning a pty. Acceptable — the underlying library has its own coverage and `shouldColorize` is unit-tested with both `nil` and `*os.File` (regular file) inputs.
- The `resolveConflict` EOF branch (line 342-345) is still uncovered by behavioral tests. Pre-existing, not introduced by this commit; suitable follow-up only if the prompt logic is touched again.
- Windows-specific virtual-terminal mode is not auto-enabled (per plan non-goal); manual `NO_COLOR=1` workaround is documented in code comments. No test required — outside scope.

No new gaps introduced by this commit.

## Verdict

- **Pass**: yes — proceed to `/sync-docs` then `/codex-review` then `/pr` per `.claude/rules/post-implementation-pipeline.md`.
- Fail: 0 tests failing across the full suite (47+ Go tests in changed packages, 11 shell tests, all other Go packages green).
- Blocked: none.

All acceptance-criteria-relevant behaviors are exercised by behavioral tests with byte-exact assertions. Both coverage gaps the verifier flagged are now closed (one was already closed by `TestShouldColorize_HonorsNoColorAndTTY`, one by the newly added `TestUnifiedDiff_GutterWidth_FiveDigitLineNumbers`). Coverage on the changed code is 100% for new helpers and ≥92.9% for modified existing functions.
