# Test report: terraform-init-validate

- Date: 2026-05-21
- Scope: Behavioral regression tests for Terraform pack opt-in backend-less init validation.
- Evidence: `docs/evidence/verify-2026-05-21-033603.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Notes |
| --- | ---: | ---: | ---: | --- |
| `tests/test-language-pack-monorepo-roots.sh` | 29 | 29 | 0 | Covers nested Terraform roots, default compatibility, opt-in init/validate, and `.terraform/` pruning. |
| `tests/test-terraform-pack-verify.sh` | 36 | 36 | 0 | Covers default validate skip, `.terraform/` validate, and opt-in backend-less init/validate. |
| `./scripts/run-test.sh` | aggregate | PASS | 0 | Full repository test wrapper passed. |

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| N/A | N/A | No failures in final run. | N/A |

## Test gaps

- Terraform/OpenTofu invocations are stubbed to keep tests hermetic; real provider plugin cache behavior is left to the selected CLI and repository CI.

## Verdict

- Pass: Yes.
- Fail: None.
- Blocked: None.
