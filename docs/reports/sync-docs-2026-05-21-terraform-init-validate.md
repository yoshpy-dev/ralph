# Sync-docs report: terraform-init-validate

- Date: 2026-05-21
- Scope: Documentation and template synchronization after Terraform pack behavior change.

## Documentation updates

- `packs/languages/terraform/README.md` documents `RALPH_TERRAFORM_INIT_VALIDATE=true`.
- `templates/packs/terraform/README.md` mirrors the same pack contract.
- The README now states that backend-backed plan evidence belongs in repository-specific trusted CI, not the generic Ralph language pack.
- Provider plugin cache handling is documented as inherited CLI behavior rather than a pack-managed responsibility.

## Sync checks

| Command | Result |
| --- | --- |
| `./scripts/check-sync.sh` | PASS |

## Verdict

- Documentation drift: None found.
- Template drift: None found.
