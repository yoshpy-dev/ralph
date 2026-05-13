# Test report: scoped-verify-test

- Date: 2026-05-14
- Plan: `docs/plans/active/2026-05-14-scoped-verify-test.md`
- Scope: behavioral tests
- Evidence: `docs/evidence/verify-2026-05-13-173944.log`
- Verdict: pass

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `tests/test-detect-changed-languages.sh` | 19 | 19 | 0 | 0 | N/A |
| `tests/test-run-verify-scope.sh` | 9 | 9 | 0 | 0 | N/A |
| `tests/test-verify-mode-split.sh` | 59 | 59 | 0 | 0 | N/A |
| `tests/test-detect-languages-terraform.sh` | 8 | 8 | 0 | 0 | N/A |
| `./scripts/run-test.sh` | full repo harness suite + Go tests | pass | 0 | 0 | N/A |
| `go test ./...` | Go packages | pass | 0 | 0 | N/A |

## Coverage

- Statement: N/A
- Branch: N/A
- Function: N/A
- Notes: This change is shell workflow logic; targeted shell integration tests cover scope selection and fallback behavior.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| N/A | N/A | No failures in final runs. | N/A |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Verify/test mode split remains non-overlapping. | pass | `tests/test-verify-mode-split.sh` passed 59/59. |
| Template mirrors stay synchronized. | pass | `scripts/check-sync.sh` passed. |
| Skill mirrors stay synchronized. | pass | `scripts/check-skill-sync.sh` passed. |
| Full fallback runs all detected language packs. | pass | `tests/test-run-verify-scope.sh` full-fallback case ran Go and Python fixture packs. |

## Test gaps

- No end-to-end GitHub Actions run is available until the PR is opened.
- No full Ralph Loop orchestrator run was performed; this is covered by targeted script behavior and docs sync checks.

## Verdict

- Pass: yes
- Fail: 0
- Blocked: none
