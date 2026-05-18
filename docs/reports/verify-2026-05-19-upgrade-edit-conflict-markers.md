# Verify report: upgrade edit conflict markers

- Date: 2026-05-19
- Plan: N/A (direct bugfix follow-up)
- Verdict: PASS
- Evidence: `docs/evidence/verify-2026-05-19-upgrade-edit-conflict-markers.log`

## Acceptance Criteria

| Criterion | Verdict | Evidence |
| --- | --- | --- |
| `ralph upgrade` baseline-backed conflicts use file-level `apply / keep / edit` choices | PASS | `TestRunUpgrade_FileApply_WritesTemplateManaged`, `TestRunUpgrade_InteractiveKeep_RecordsPartialManaged`, `TestRunUpgrade_InteractiveDiff_RepromptsOnInvalid` |
| File `edit` opens a Git-style conflict marker block with both local and template sides | PASS | `TestRunUpgrade_FileEdit_SeedsConflictMarkersWhenLocalSideEmpty` |
| Edited file output is accepted only after conflict markers are removed | PASS | `TestRunUpgrade_FileEdit_RejectsUnresolvedConflictMarkers` |
| Unresolved marker saves fail closed and return to the file prompt without applying partial disk changes | PASS | `TestRunUpgrade_FileEdit_RejectsUnresolvedConflictMarkers` |
| Existing editor success behavior remains intact | PASS | `TestRunUpgrade_FileEdit_UsesEditor` |
| File-level `keep local file` records a managed partial resolution and avoids same-version re-prompts | PASS | `TestRunUpgrade_InteractiveKeep_RecordsPartialManaged`, `TestRunUpgrade_NextRunAfterKeep_IsSilent` |
| Obsolete conflict-selection terminology is removed from current code and current PR docs | PASS | Targeted text search across `internal`, `README.md`, CLI spec, and current PR reports returned no matches. |
| User-facing docs describe the marker-based edit flow | PASS | `README.md`, `docs/specs/2026-04-16-ralph-cli-tool.md` |

## Static Verification

- `./scripts/run-static-verify.sh` passed in changed-language scope.
- Included sync guards, pipeline sync, skill sync, changed-language detection, Go formatting, and Go static verification.
- `golangci-lint` emitted sandbox cache persistence warnings under `~/Library/Caches/golangci-lint`, but reported `0 issues` and the verifier exited successfully.

## Documentation Drift

No drift found. The README and CLI spec now describe the file-level `apply / keep / edit` prompt, `edit file` as a Git-style conflict marker block, `keep local file` as a managed partial resolution, and unresolved-marker rejection.

## Remaining Gaps

None identified for this change.
