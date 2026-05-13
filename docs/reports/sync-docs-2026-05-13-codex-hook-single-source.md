# Sync-docs report: codex-hook-single-source

- Date: 2026-05-13
- Plan: `docs/plans/archive/2026-05-13-codex-hook-single-source.md`
- Maintainer: Codex
- Scope: Codex hook documentation and verification drift.

## Documentation Updates

| File | Status | Notes |
| --- | --- | --- |
| `.codex/config.toml` | Updated | Documents `.codex/config.toml` as the inline hook source of truth. |
| `templates/base/.codex/config.toml` | Updated | Mirrors root. |
| `.codex/README.md` | Updated | Explains not to add `.codex/hooks.json` beside inline hooks. |
| `templates/base/.codex/README.md` | Updated | Mirrors root. |
| `.codex/hooks/README.md` | Updated | Keeps the directory as wrapper-reserved and warns against `hooks.json`. |
| `templates/base/.codex/hooks/README.md` | Updated | Mirrors root. |
| `scripts/verify.local.sh` | Updated | Enforces duplicate hook representation guard. |

## Drift Checks

- `scripts/check-sync.sh`: PASS.
- `git diff --check`: PASS.

## Verdict

Pass.
