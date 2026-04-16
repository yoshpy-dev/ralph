# Self-review report: ralph-tui (cycle 2)

- Date: 2026-04-15
- Plan: docs/plans/active/2026-04-15-ralph-tui/_manifest.md
- Reviewer: pipeline-self-review (autonomous)
- Scope: diff quality

## Evidence reviewed

- `git diff main...HEAD --stat` (76 files, 8732 insertions, 21 deletions)
- `git diff main...HEAD -- '*.go' 'scripts/' '.gitignore' 'go.mod' 'go.sum'` (full source diff)
- Files inspected: cmd/ralph-tui/main.go, cmd/ralph-tui/version.go, go.mod, go.sum, internal/action/*.go, internal/deps/deps.go, internal/state/reader.go, internal/state/types.go, internal/state/testdata/checkpoint.json, internal/ui/model.go, internal/ui/layout.go, internal/ui/confirm.go, internal/ui/panes/slicelist.go, internal/ui/panes/actions.go, internal/watcher/watcher.go, scripts/ralph, scripts/build-tui.sh, .gitignore
- Cross-referenced: `.harness/state/pipeline/checkpoint.json` (live pipeline data) vs Go type definitions in `types.go`

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| CRITICAL | unnecessary-change | `coverage.out` (654 lines) committed to git. This is a generated test coverage artifact that should never be tracked in version control. | `git diff main...HEAD -- coverage.out` shows 654 lines of coverage data added. `.gitignore` adds `bin/` but not `coverage.out`. | Add `coverage.out` to `.gitignore` and remove from tracking with `git rm --cached coverage.out`. |
| CRITICAL | null-safety | `SelfReviewResult` type mismatch in `types.go:86`. Field is declared as `*string` but the pipeline writes it as a JSON object `{"critical":N,"high":N,...}`. This causes `json.Unmarshal` to fail, silently discarding all checkpoint data for any slice past self-review. | `internal/state/types.go:86`: `SelfReviewResult *string json:"self_review_result"`. `.harness/state/pipeline/checkpoint.json:12-17`: `"self_review_result": {"critical": 2, "high": 0, "medium": 3, "low": 3}`. Test data uses `null` so tests pass, masking the bug. | Change type to a struct matching the actual JSON shape (e.g., `*SelfReviewCounts` with critical/high/medium/low int fields), or use `json.RawMessage` to defer parsing. |
| MEDIUM | exception-handling | Executor creation error silently swallowed in `cmd/ralph-tui/main.go:90-92`. If `scripts/ralph` is missing, `executor` stays nil and all actions fail silently with no user feedback. | `main.go:90-92`: `if exec, err := action.NewExecutor(repoRoot); err == nil { executor = exec }` — no log, no warning. | Log a warning when executor creation fails so the user knows actions are unavailable. |
| MEDIUM | maintainability | `internal/deps/deps.go` — blank import stub for go.mod dependency retention. Comment says "Remove this file once all slices are integrated." This IS the integration slice, so the file is now unnecessary. | `deps.go:3-6`: "This file is imported by no one; it exists only to prevent go mod tidy from removing dependencies that downstream slices (2-6) will use. Remove this file once all slices are integrated." | Remove `internal/deps/deps.go` and run `go mod tidy` to verify dependencies remain. |
| MEDIUM | security | `scripts/ralph cmd_abort()` audit JSON uses raw shell string interpolation without escaping. Slice names with special characters could produce malformed JSON in the audit log. Not exploitable (no eval) but reduces audit reliability. | `scripts/ralph:435`: `"target_slice": $([ -n "$_target_slice" ] && echo "\"${_target_slice}\"" || echo "null")` — no escaping of `_target_slice`. | Validate or sanitize `_target_slice` before embedding in JSON, or use `jq` for safe JSON construction. |
| LOW | readability | Comment in `cmd/ralph-tui/main.go:86-87` says "repo root is two levels up from orch-dir: .harness/state/loop/ -> repo root" but `resolveRepoRoot` actually walks up searching for `.git` or `scripts/ralph`, which is smarter than the comment implies. | Comment suggests fixed 3-level traversal; code does dynamic search with 10-iteration cap and fallback. | Update comment to match actual behavior. |
| LOW | naming | Help text for "e" key says "Expand detail" but the actual action is "Open editor". | `help.go:31` vs `actions.go:227` — label mismatch. (Carried over from previous review.) | Align help text with actual behavior: "Open editor". |

## Positive notes

- **Security**: `ValidateSliceName()` in `internal/action/executor.go` has thorough input validation rejecting shell metacharacters, path traversal, and whitespace. The test suite covers 20+ injection variants.
- **Architecture**: Clean separation of concerns — state reading, file watching, UI rendering, and action execution are well-isolated packages with clear interfaces and no circular imports.
- **Concurrency safety**: The watcher uses `sync.Once` for clean shutdown, buffered channels to avoid blocking, and properly handles the `done` channel to prevent goroutine leaks.
- **Graceful degradation**: fsnotify failure falls back to polling; small terminals get a compact single-pane view; missing TUI binary falls back to table output.
- **Shell script quality**: `scripts/ralph` uses `set -eu`, proper quoting, and POSIX-compatible constructs. The retry subcommand properly checks orchestrator state, parallel limits, and PID tracking before launching.
- **Test coverage**: Tests include edge cases (empty names, injection attempts, closed channels, missing files) alongside happy paths.
- **No secrets, credentials, or debug code** found in the diff.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| `internal/deps/deps.go` blank import stub | Minor — dead code in binary, confusing to readers | Created during multi-worktree parallel development to prevent `go mod tidy` from pruning dependencies | Integration merge complete (now) | slice-1 plan |
| `SelfReviewResult` type mismatch | HIGH — prevents reading checkpoint data after self-review | Test data uses null, masking the issue; likely a schema drift between pipeline scripts and Go types | Before first real use of TUI on an active pipeline | This report |
| `coverage.out` in git | Minor — repo bloat, confusing for contributors | Accidentally committed during test run | Immediate — add to .gitignore and remove | This report |
| Audit JSON escaping in `cmd_abort` | LOW — malformed audit logs for unusual slice names | Shell-native JSON construction is inherently fragile | When `jq` dependency is acceptable or when audit logs are consumed programmatically | This report |
| Help text/action label mismatch ("Expand detail" vs "Editor") | User confusion | Low severity, cosmetic | Before v1 release | Previous review |

## Recommendation

- Merge: **no** (2 CRITICAL findings)
- Follow-ups:
  1. **CRITICAL**: Remove `coverage.out` from tracking and add to `.gitignore`
  2. **CRITICAL**: Fix `SelfReviewResult` type in `types.go` to match actual checkpoint.json schema (struct with critical/high/medium/low int fields)
  3. **MEDIUM**: Add a log warning when executor creation fails in `main.go`
  4. **MEDIUM**: Remove `internal/deps/deps.go` now that all slices are integrated
  5. **MEDIUM**: Sanitize slice names in audit JSON or use `jq`
  6. **LOW**: Update misleading comment in `main.go:86-87`
  7. **LOW**: Fix help text for "e" key: "Expand detail" -> "Open editor"
