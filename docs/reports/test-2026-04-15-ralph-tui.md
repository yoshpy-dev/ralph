# Test report: ralph-tui (cycle 2)

- Date: 2026-04-15
- Plan: docs/plans/active/2026-04-15-ralph-tui/_manifest.md (slice-6)
- Tester: pipeline-test (autonomous)
- Scope: behavioral tests
- Evidence: `docs/evidence/test-2026-04-15-ralph-tui.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `internal/action` | 9 | 9 | 0 | 0 | 3.4s |
| `internal/state` | 12 | 12 | 0 | 0 | 1.1s |
| `internal/ui` | 41 | 41 | 0 | 0 | 1.5s |
| `internal/ui/panes` | 57 | 57 | 0 | 0 | 2.8s |
| `internal/watcher` | 14 | 14 | 0 | 0 | 2.8s |
| `cmd/ralph-tui` | 0 | 0 | 0 | 0 | — |
| **Total** | **133** | **133** | **0** | **0** | **~11.6s** |

## Coverage

- Statement: ~87% (weighted average of tested packages)
- Branch: N/A (Go does not report branch coverage natively)
- Function: N/A
- Notes:
  - `internal/action`: 95.7%
  - `internal/state`: 86.2%
  - `internal/ui`: 88.1%
  - `internal/ui/panes`: 88.9%
  - `internal/watcher`: 80.6%
  - `cmd/ralph-tui`: 0.0% (no test files; thin entrypoint delegates to internal packages)
  - `internal/deps`: no test files (blank import stub, no runtime logic)
  - All testable packages exceed 80% coverage.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (none) | — | — | — |

No test failures.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| State reader returns zero-value on missing orchestrator file | OK | `TestReadFullStatus_NoOrchestrator` passes |
| Watcher graceful on missing directory | OK | `TestWatcher_GracefulOnMissingDir` passes |
| Watcher polling fallback and detection | OK | `TestWatcher_PollingFallback`, `TestWatcher_PollingDetectsNewFile`, `TestWatcher_PollingDetectsRemoval` pass |
| Tailer handles missing/switched files | OK | `TestTailer_MissingFile`, `TestTailer_SwitchFile` pass |
| Slice name validation rejects shell metacharacters | OK | `TestValidateSliceName` (22 injection variants) all pass |
| Action availability per slice status | OK | `TestActionsModel_FailedSliceActions`, `_RunningSliceActions`, `_CompleteSliceActions`, `_PendingSliceActions`, `_StuckSliceActions` pass |
| ANSI stripping in log view | OK | `TestLogViewANSIStripping`, `TestLogViewAppendLineANSI` pass |
| Confirm dialog key handling | OK | `TestConfirmModel_UpdateYes`, `_UpdateNo`, `_UpdateIgnoredWhenHidden` pass |
| Filter input/escape in slice list | OK | `TestSliceListFilter`, `TestSliceListFilterEscape`, `TestSliceListFilterBackspace` pass |

## Test gaps

### Plan test items vs actual coverage

| Plan test item | Covered? | Notes |
| --- | --- | --- |
| `go test ./cmd/ralph-tui/...` — flag parsing, init logic | NO | No test files. Thin entrypoint; low risk but below plan expectations. |
| `scripts/build-tui.sh` builds successfully | NO (shell) | Verified by verify agent via `go build`. |
| `ralph status --json` output unchanged (regression) | NO (shell) | Requires shell-level regression test. Verified by verify agent (spec compliance). |
| `ralph status --no-tui` returns table output | NO (shell) | Requires shell-level test. Verified by verify agent. |
| `ralph retry` checks status/locklist/parallel/deps | PARTIAL | Go tests cover retry execution and name validation. Locklist and dependency checks are **not implemented** per verify report. |
| `ralph abort --slice` aborts single slice | YES | `TestAbortSlice` covers valid and invalid cases. |
| Go not installed for build-tui.sh | NO (shell) | Not testable from Go. |
| bin/ directory absent | NO (shell) | Shell-level edge case. |
| retry on running slice returns error | PARTIAL | UI test confirms retry disabled for running. No CLI-level test. |
| retry with locklist conflict | N/A | Not implemented per verify report. |

### Critical test gap: SelfReviewResult type mismatch

Self-review identified a CRITICAL bug: `SelfReviewResult` in `types.go` is `*string` but the actual checkpoint JSON writes an object `{"critical":N,...}`. Test data uses `null`, so tests pass while masking the bug. **A test with a real object-typed `self_review_result` should be added to catch this.**

## Verdict

- Pass: **yes** — all 133 tests pass, 0 failures
- Fail: 0
- Blocked: 0
- Coverage: ~87% across tested packages (all exceed 80%); `cmd/ralph-tui` has 0% but is a thin entrypoint
- Notable gaps: SelfReviewResult type mismatch masked by null in test data; shell-level integration tests for `scripts/ralph` subcommands absent; retry locklist/dependency checks not implemented
