# Test report: ralph-tui (slice-2)

- Date: 2026-04-15
- Plan: docs/plans/active/2026-04-15-ralph-tui/slice-2-ralph-tui.md
- Tester: pipeline-test (autonomous)
- Scope: behavioral tests
- Evidence: `docs/evidence/test-2026-04-15-ralph-tui-slice-2.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `go test -v -cover -race ./internal/watcher/...` | 14 | 14 | 0 | 0 | 3.585s |

## Coverage

- Statement: 78.6% (78.2% with -race)
- Branch: N/A (Go coverage tool reports statement coverage)
- Function: see per-function breakdown below
- Notes: **Below 80% target** (AC7). Gap is 1.4pp. Lowest-coverage functions: `Tail()` 50%, `Watch()` 50%, `eventLoop` 58.3%, `addWorktreeWatches` 63.6%.

### Per-function coverage

| Function | Coverage | Notes |
| --- | --- | --- |
| `Error` (messages.go) | 100% | |
| `NewTailer` | 80% | |
| `Tail` | 50% | tea.Cmd wrapper; inner func not exercised by unit tests |
| `SwitchFile` | 70% | Error paths partially uncovered |
| `Stop` (tailer) | 100% | |
| `readLoop` | 100% | |
| `readNewLines` | 68.8% | EOF / partial-line edge cases |
| `waitForFile` | 85.7% | |
| `New` (watcher) | 75% | fsnotify success path + fallback tested; some branches missed |
| `NewWithPolling` | 100% | |
| `Watch` | 50% | tea.Cmd wrapper; inner func not exercised |
| `Stop` (watcher) | 100% | |
| `eventLoop` | 58.3% | fsnotify error channel not tested |
| `pollLoop` | 100% | |
| `scanFiles` | 87% | |
| `collectWatchPaths` | 100% | |
| `addWorktreeWatches` | 63.6% | Partial dir-walk coverage |
| `sendMsg` | 100% | |
| `opString` | 83.3% | Zero-value case not tested |

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (none) | — | — | — |

All 14 tests pass with race detector enabled. No failures.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| N/A (first implementation — no prior regressions to check) | — | — |

## Test gaps

### Scenarios from plan's test plan

| Plan scenario | Covered? | Notes |
| --- | --- | --- |
| Unit tests: `go test ./internal/watcher/...` | yes | 14 tests pass |
| Integration: temp dir file create → modify → watcher msg | yes | `TestStateChangedMsg_OnFileWrite`, `TestStateChangedMsg_OnFileCreate` |
| Edge: missing watch directory | yes | `TestWatcher_GracefulOnMissingDir` |
| Edge: file deletion | yes | `TestWatcher_PollingDetectsRemoval` |
| Edge: rapid consecutive writes | **no** | Not tested. Could add a test writing N lines in quick succession to verify no dropped messages. |

### Coverage gap analysis (functions below 80%)

To reach 80% total coverage, focus on:

1. **`Watch()` / `Tail()` (both 50%)**: These are `tea.Cmd` wrappers that return a closure. The inner function is tested indirectly through other tests but the Go coverage tool doesn't attribute those lines. Adding a test that calls `Watch()()`/`Tail()()` directly would cover these.

2. **`eventLoop` (58.3%)**: The fsnotify error channel path (`case err := <-w.fsWatcher.Errors`) is not tested. Adding a test that injects an error via the fsnotify watcher's error channel would cover this.

3. **`addWorktreeWatches` (63.6%)**: The successful directory walk path with actual worktree subdirectories is not fully exercised. Adding a test with a mock worktree directory structure would help.

4. **`readNewLines` (68.8%)**: Partial-line reads (line without trailing newline at EOF) are not fully tested.

5. **`SwitchFile` (70%)**: Error path when opening the new file fails is not tested.

### Estimated impact

Adding tests for items 1-2 above would likely bring coverage from 78.6% to ~82-84%, meeting the 80% target.

## Verdict

- Pass: yes (all tests pass, no failures, no race conditions)
- Fail: 0
- Blocked: none
- **Coverage warning**: 78.6% is below the 80% AC target. This is a soft failure for AC7. The gap is small (1.4pp) and addressable by adding 2-3 targeted tests for the `Watch()`/`Tail()` tea.Cmd wrappers and the `eventLoop` error path.
