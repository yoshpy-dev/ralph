# Self-review report: 6-ralph-tui

- Date: 2026-04-15
- Plan: docs/plans/active/2026-04-15-ralph-tui/slice-4-ralph-tui.md (slice 6 integration)
- Reviewer: pipeline-self-review (autonomous)
- Scope: diff quality

## Evidence reviewed

- `git diff main...HEAD --stat` (72 files, ~7876 insertions)
- `git diff main...HEAD -- '*.go' 'scripts/' '.gitignore' 'go.mod' 'go.sum'` (full source diff)
- Individual file reads: cmd/ralph-tui/main.go, version.go, internal/state/types.go, reader.go, internal/ui/model.go, layout.go, pane.go, keys.go, styles.go, help.go, confirm.go, messages.go, internal/ui/panes/slicelist.go, detail.go, deps.go, progress.go, actions.go, logview.go, internal/watcher/watcher.go, tailer.go, messages.go, internal/action/executor.go, abort.go, retry.go, external.go, messages.go, internal/deps/deps.go, scripts/build-tui.sh

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | null-safety | `rebuildFiltered` re-slices `m.filtered[:0]` which panics if `filtered` is nil on first call before any allocation | `slicelist.go:153` — `m.filtered = m.filtered[:0]` is safe only because `NewSliceList` calls `rebuildFiltered()` which triggers the append path, but if `SetSlices` is called before any construction path that allocates, it would panic | Initialize `filtered` to `[]int{}` in `NewSliceList` or use `m.filtered = nil` + append pattern |
| MEDIUM | maintainability | `appModel` in `main.go` silently ignores `ReadFullStatus` errors on state change events | `main.go:141` — `if status, err := ...; err == nil {` discards errors without logging | Log the error or surface it as a StatusMsg to the UI |
| MEDIUM | maintainability | `resolveRepoRoot` fallback returns a relative `../../..` join without cleanup | `main.go:104` — `filepath.Join(orchDir, "..", "..", "..")` — while `filepath.Join` cleans the path, the comment says "3 levels up" but the loop tries 10 levels; the fallback is a silent guess | Add a log warning when falling back |
| LOW | readability | Duplicate `LogLineMsg` types in `watcher/messages.go` and `ui/messages.go` with slightly different shapes (watcher has `SliceName` field, ui does not) | `watcher/messages.go:18-21` vs `ui/messages.go:11-13` | Acceptable for layer separation, but note the mapping in `main.go:154-155` must stay in sync |
| LOW | naming | `helpEntry` field "e" in help overlay says "Expand detail" but in `actions.go` it maps to "Editor" (open editor) | `help.go:31` says "Expand detail" vs `actions.go:227` says "Editor" | Align the help text with the actual behavior: should be "Open editor" |
| LOW | readability | `opString` allocates a 4-cap slice every call even when only 1 op is typically set | `watcher.go:247` — `parts := make([]string, 0, 4)` | Minor — acceptable for this call frequency |

## Positive notes

- **Clean module boundaries**: `state/`, `ui/`, `watcher/`, `action/` packages have clear single responsibilities with no circular imports.
- **Security**: `ValidateSliceName` in `action/executor.go` properly rejects shell metacharacters and path traversal. External commands use `exec.Command` directly (no shell interpretation).
- **No secrets or credentials** anywhere in the diff.
- **No debug code** — no leftover `fmt.Println`, `log.Println` debug statements, or TODO markers.
- **Good test coverage structure** — test files exist for all major packages (state, ui, panes, watcher, action).
- **Proper resource cleanup** — `sync.Once` for watcher/tailer shutdown, deferred file closes, channel lifecycle management.
- **Well-structured Bubble Tea architecture** — message types, key handling, and pane rendering follow idiomatic patterns.
- **Build script** validates Go version and uses ldflags for version embedding.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Help text/action label mismatch ("Expand detail" vs "Editor") | User confusion | Low severity, cosmetic | Before v1 release | This report |
| Silent error swallowing on state refresh in `appModel.Update` | Debugging difficulty | Non-blocking for MVP | When users report stale state | This report |
| Two `LogLineMsg` types across packages | Maintenance burden if fields diverge | Layer separation is intentional | If additional fields needed | This report |

## Recommendation

- Merge: **yes**
- Follow-ups:
  1. Fix help text for "e" key: "Expand detail" → "Open editor" (LOW)
  2. Add error logging in `appModel.Update` state refresh path (MEDIUM)
  3. Consider adding a log warning to `resolveRepoRoot` fallback (MEDIUM)
