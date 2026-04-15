# Self-review report: ralph-tui (slice-2)

- Date: 2026-04-15
- Plan: docs/plans/active/2026-04-15-ralph-tui/slice-2-ralph-tui.md
- Reviewer: pipeline-self-review (autonomous)
- Scope: diff quality

## Evidence reviewed

- `git diff main...HEAD` — full diff of branch against main (4 new Go source files, go.mod, go.sum)
- `internal/watcher/messages.go` — message type definitions (28 lines)
- `internal/watcher/tailer.go` — log file tail follower (215 lines)
- `internal/watcher/watcher.go` — fsnotify/polling file watcher (264 lines)
- `internal/watcher/watcher_test.go` — test suite (455 lines)
- `go.mod` / `go.sum` — module dependencies
- `docs/plans/active/2026-04-15-ralph-tui/_manifest.md` — plan manifest
- `docs/plans/active/2026-04-15-ralph-tui/slice-2-ralph-tui.md` — slice plan

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | Goroutine leak in `Tailer.SwitchFile()`: each call spawns a new `readLoop` goroutine (tailer.go:106) without canceling the previous one. Old goroutines continue ticking via their own `time.Ticker` until `Stop()` closes the `done` channel. No data corruption (offset is mutex-protected), but leaked goroutines accumulate with each switch. | tailer.go:106 `go t.readLoop()` — no per-loop cancellation channel | Add a per-readLoop `context.Context` or a `stopLoop chan struct{}` that is closed before starting a new loop. Reset it in SwitchFile before launching the new goroutine. |
| MEDIUM | exception-handling | `fsWatcher.Add()` errors silently discarded in `New()` (watcher.go:48) and `addWorktreeWatches()` (watcher.go:232). If the kernel watch limit is hit or the directory is not watchable, no events will fire with zero diagnostic signal. | watcher.go:48 `_ = fsw.Add(orchDir)`, watcher.go:232 `_ = w.fsWatcher.Add(dir)` | Log a warning or fall back to polling for the affected directory. At minimum, track failed watches so callers can diagnose missing events. |
| LOW | readability | `TestOpString` (watcher_test.go:412-426) is an empty test — the table-driven structure has no assertions. Comment claims indirect coverage, but the test body is a no-op. | watcher_test.go:421-424 — empty test body | Either add real cases (`fsnotify.Write` -> `"write"`, combined ops -> `"write|create"`, zero -> `"unknown"`) or delete the test to avoid confusion. |
| LOW | readability | Manual newline trim in `readNewLines` (tailer.go:173-176) is verbose. The `if len(trimmed) > 0 && trimmed[len(trimmed)-1] == '\n'` guard is always true when reached via `ReadString('\n')` on a complete line. | tailer.go:173-176 | Use `strings.TrimRight(line, "\n")` for clarity, or simply `line[:len(line)-1]` with the existing length check. |

## Positive notes

- Clean package structure: messages separated from logic, watcher and tailer responsibilities clearly split across files.
- Good use of `sync.Once` for idempotent `Stop()` on both Watcher and Tailer — prevents double-close panics.
- Polling fallback is well-implemented with proper initial-scan baseline to avoid false positives on startup.
- Comprehensive test suite covering: file write, file create, missing directory, polling fallback, polling create/remove detection, double-stop safety, tailer new lines, multiple lines, missing file wait, file switch, and error message formatting.
- Proper mutex discipline — `t.offset` and `t.file` are protected, and the mutex is released before blocking channel sends to avoid deadlock.
- Non-blocking `sendMsg` helper prevents goroutine hangs when the done channel is closed.
- `waitForFile` correctly transitions into `readLoop` (blocking in-goroutine), so only one goroutine exists for the initial file-wait case.
- Test helper `msgSource` interface is a nice pattern for testing both Watcher and Tailer with the same `waitForMsg` function.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Goroutine leak in SwitchFile | LOW (infrequent calls, goroutines are lightweight) | SwitchFile is called only on user slice selection; accumulation is bounded by session lifetime | If SwitchFile is called in a loop or automated context | slice-2 |
| Silent fsWatcher.Add failures | LOW (polling fallback exists as safety net) | fsnotify errors are rare in practice; polling fallback covers the gap | If users report missing events on specific platforms | slice-2 |
| Empty TestOpString | NONE (no runtime impact) | Test was likely a placeholder during development | Next test coverage pass | slice-2 |

## Recommendation

- Merge: yes (conditional — the MEDIUM findings are non-blocking for this slice but should be addressed before slice-6 integration)
- Follow-ups:
  1. Add per-readLoop cancellation to `SwitchFile` to prevent goroutine leaks
  2. Handle or log `fsWatcher.Add()` errors instead of silently discarding
  3. Fill in or remove the empty `TestOpString` test
  4. Simplify newline trimming in `readNewLines`
