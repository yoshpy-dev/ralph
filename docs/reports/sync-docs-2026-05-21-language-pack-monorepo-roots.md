# Sync-docs report: language-pack-monorepo-roots

- Date: 2026-05-21
- Plan: `docs/plans/archive/2026-05-21-language-pack-monorepo-roots.md`
- Maintainer: Codex

## Documentation and contract updates

| Area | Status | Notes |
| --- | --- | --- |
| Active plan | Updated | Progress and verification targets recorded. |
| Pack READMEs/rules | Updated | Dart README documents non-mutating format; Terraform README/rule documents per-root `fmt -check`. |
| Template mirrors | Updated | `templates/packs/*` and `templates/base/scripts/*` copied from root sources. |
| Quality docs | No change needed | Existing gates already require `run-static-verify.sh`, `run-test.sh`, sync checks, and reports. |
| AGENTS/CLAUDE/README | No change needed | The high-level workflow and repo map remain accurate; no new top-level workflow or pack was added. |

## Checks

| Command | Result |
| --- | --- |
| `scripts/check-sync.sh` | PASS |
| `scripts/check-template.sh` | PASS |
| `./scripts/run-static-verify.sh` | PASS |
| `./scripts/run-test.sh` | PASS |

## Remaining drift

None found.
