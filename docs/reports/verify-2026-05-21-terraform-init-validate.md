# Verify report: terraform-init-validate

- Date: 2026-05-21
- Scope: Spec compliance, static analysis, and template sync for Terraform opt-in init validation.
- Evidence: `docs/evidence/verify-2026-05-21-033538.log`

## Deterministic checks run

| Command | Result | Notes |
| --- | --- | --- |
| `sh -n packs/languages/terraform/verify.sh templates/packs/terraform/verify.sh tests/test-language-pack-monorepo-roots.sh tests/test-terraform-pack-verify.sh` | PASS | Edited shell scripts parse cleanly. |
| `./scripts/check-sync.sh` | PASS | `templates/packs/terraform/*` is in sync with `packs/languages/terraform/*`. |
| `./scripts/run-static-verify.sh` | PASS | Local static gates, sync gates, Go verifier, `gofmt`, and `staticcheck` passed. |

## Acceptance criteria

| Criterion | Status | Evidence |
| --- | --- | --- |
| Default Terraform pack behavior remains compatible | Verified | Default static mode still skips validate when `.terraform/` is absent and still validates when `.terraform/` exists. |
| `RALPH_TERRAFORM_INIT_VALIDATE=true` runs backend-less init and validate per root | Verified | `tests/test-language-pack-monorepo-roots.sh` covers nested roots and `tests/test-terraform-pack-verify.sh` covers the pack-local opt-in branch. |
| Backend-backed plan/workflow behavior is not implemented in the generic pack | Verified | Diff contains no plan/workflow dispatch logic; README documents the repo-specific trusted CI boundary. |
| Provider plugin cache remains caller-managed | Verified | The pack does not set or require cache configuration; selected CLI inherits existing environment/config. |
| Template mirror is synchronized | Verified | `./scripts/check-sync.sh` passed with 0 drifted files. |

## Coverage gaps

- Tests use fake Terraform/OpenTofu binaries for command dispatch; no real provider plugin downloads are attempted.

## Verdict

- Verified: All requested behavior.
- Partially verified: None.
- Not verified: None.
