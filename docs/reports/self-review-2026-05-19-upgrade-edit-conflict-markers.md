# Self-review report: upgrade edit conflict markers

- Date: 2026-05-19
- Plan: N/A (direct bugfix follow-up)
- Reviewer: Codex
- Scope: Current working-tree diff for `ralph upgrade` file-level conflict resolution and terminology cleanup

## Evidence reviewed

- `git diff -- internal/cli/upgrade.go internal/cli/cli_test.go internal/upgrade/merge.go internal/upgrade/unified_diff.go README.md docs/specs/2026-04-16-ralph-cli-tool.md`
- Targeted text search for obsolete conflict-selection terms across current code, README, CLI spec, and current PR reports.
- Changed files:
  - `internal/cli/upgrade.go`
  - `internal/cli/cli_test.go`
  - `internal/upgrade/merge.go`
  - `internal/upgrade/merge_test.go`
  - `internal/upgrade/unified_diff.go`
  - `internal/upgrade/unified_diff_test.go`
  - `internal/upgrade/colorize.go`
  - `internal/upgrade/colorize_test.go`
  - `README.md`
  - `docs/specs/2026-04-16-ralph-cli-tool.md`

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| None | N/A | No blocking or follow-up findings in the diff quality review. | The implementation keeps the behavior scoped to baseline-backed file conflict resolution, uses file-level `apply / keep / edit` choices, reuses existing editor plumbing, rejects unresolved marker blocks before staging a decision, and removes obsolete conflict-selection terminology from current code and current PR docs. | Merge after normal verification remains green. |

## Positive notes

- The helper split (`editConflictFile`, `conflictMarkerFileLines`, `hasConflictMarkers`) keeps prompt control flow readable and isolates full-file marker construction.
- The unresolved-marker path fails closed: it returns to the same file prompt without writing target files, manifest state, or baseline cache.
- Tests cover the reported empty-editor path and the safety behavior when a user saves the marker block unchanged.
- File-level `keep local file` records a managed partial resolution instead of using the legacy unmanaged skip path.
- Current implementation and current PR reports no longer use the old conflict-selection terminology.
- Documentation now describes the Git-style edit surface and unresolved-marker rejection.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| None | N/A | N/A | N/A | N/A |

## Recommendation

- Merge: Yes, after the already-run Go test suite remains green in CI.
- Follow-ups: None required from this review.
