# Self-review report: ralph-tui (slice-1)

- Date: 2026-04-15
- Plan: docs/plans/active/2026-04-15-ralph-tui/slice-1-ralph-tui.md
- Reviewer: pipeline-self-review (autonomous)
- Scope: diff quality

## Evidence reviewed

- `git diff main...HEAD` — 814 insertions across 11 files
- `git diff --stat main...HEAD` — file list and line counts
- `internal/state/types.go` — full read (112 lines)
- `internal/state/reader.go` — full read (212 lines)
- `internal/state/reader_test.go` — full read (335 lines)
- `go.mod` — full read (25 lines)
- All testdata fixtures — full read
- Plan manifest and slice-1 plan — full read

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | `ReadFullStatus` returns `Slices` in non-deterministic order due to map iteration over `statuses` | `reader.go:148-167` iterates `for name, status := range statuses` where `statuses` is `map[string]string` | Sort `slices` by `Name` before returning, or document that consumers must sort |
| LOW | unnecessary-change | All `go.mod` dependencies are `// indirect` because only `internal/state/` (stdlib-only) exists so far; `go mod tidy` would remove them | `go.mod:6-15` — all entries are `// indirect`; diagnostics show 10 unused module warnings | Intentional per plan ("全依存を前もって追加") — will resolve as later slices import charm packages. Consider adding a `// DO NOT TIDY` comment to prevent accidental cleanup |
| LOW | maintainability | `sliceDepRe` regex uses `[^─]*` which greedily matches any non-arrow character; fragile if manifest format changes | `reader.go:72` — `^slice-(\d+)[^─]*──→\s*slice-(\d+)` | Acceptable for known manifest format; test coverage confirms correctness for current patterns |

## Positive notes

- **Error handling is consistent** — every file-reading function wraps errors with `fmt.Errorf("context: %w", err)`, preserving the error chain
- **Nullable JSON fields handled correctly** — `*string` pointer types for `LastTestResult`, `SelfReviewResult`, `VerifyResult`, `SessionID`, `PRUrl` correctly model JSON null vs empty string
- **Graceful degradation** — `ReadFullStatus` does not fail if dependencies are missing (`reader.go:131` swallows the error and sets `deps = nil`), and missing checkpoints are handled per-slice without aborting
- **Comprehensive test coverage** — tests include happy path, missing file, invalid JSON, edge cases (directory-as-file for read error), and helper function unit tests
- **Test data is realistic** — fixtures mirror actual `.harness/state/` structure with complete and running slices, nullable fields, and multi-source dependency lines
- **Clean type design** — types are well-documented, grep-able, and map directly to the JSON schema used by the shell-based pipeline
- **`splitDependencyLine` handles multi-source deps** — correctly parses `slice-2, slice-4 ──→ slice-5` into individual edges
- **No debug code, no secrets, no security concerns** — clean production code

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| Non-deterministic slice ordering in `ReadFullStatus` | TUI display may flicker if slices render in random order | UI layer (slice-3/4) can sort before display; state reader should not impose ordering policy | When slice-3 (layout) implements the TUI table view | slice-3-ralph-tui.md |
| `go.mod` indirect-only dependencies | `go mod tidy` would strip all charm dependencies | Other slices will import them, making them direct | After slice-2 or slice-3 adds imports | slice-2-ralph-tui.md |

## Recommendation

- Merge: **yes**
- Follow-ups:
  - Slice-3 or slice-4 should sort `FullStatus.Slices` before rendering (or `ReadFullStatus` should sort before returning)
  - Avoid running `go mod tidy` until at least one slice imports charm packages
