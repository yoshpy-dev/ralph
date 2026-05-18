# Test report: upgrade edit conflict markers

- Date: 2026-05-19
- Plan: N/A (direct bugfix follow-up)
- Verdict: PASS
- Evidence: `docs/evidence/test-2026-05-19-upgrade-edit-conflict-markers.log`

## Test Runs

| Command | Verdict | Notes |
| --- | --- | --- |
| `./scripts/run-test.sh` | PASS | Changed-language test mode selected the Go verifier/test path and all checks passed. |

## Coverage

| Area | Verdict | Evidence |
| --- | --- | --- |
| Normal path: user edits marker block into resolved file content | PASS | `TestRunUpgrade_FileEdit_UsesEditor` |
| Normal path: user applies the template for the whole file | PASS | `TestRunUpgrade_FileApply_WritesTemplateManaged` |
| Normal path: user keeps the local file as a managed partial resolution | PASS | `TestRunUpgrade_InteractiveKeep_RecordsPartialManaged` |
| Regression: kept local file does not re-prompt on the same version | PASS | `TestRunUpgrade_NextRunAfterKeep_IsSilent` |
| Regression: local side empty no longer opens a blank editor | PASS | `TestRunUpgrade_FileEdit_SeedsConflictMarkersWhenLocalSideEmpty` |
| Error path: user leaves conflict markers unresolved | PASS | `TestRunUpgrade_FileEdit_RejectsUnresolvedConflictMarkers` |
| Broader Go package regression suite | PASS | `go test` output for all project packages in the evidence log |
| Shell regression tests | PASS | Shell test sections in the evidence log all passed |

## Failure Analysis

No failures.

## Remaining Gaps

None identified for this change.
