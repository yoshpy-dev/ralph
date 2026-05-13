# Sync-docs report: codex-hook-duplicate-doctor

- Date: 2026-05-13
- Plan: `docs/plans/archive/2026-05-13-codex-hook-duplicate-doctor.md`
- Maintainer: Codex
- Scope: Documentation drift after adding the doctor duplicate hook guard.

## Results

| Surface | Action | Evidence |
| --- | --- | --- |
| `.codex/README.md` and template mirror | Updated hooks guidance to say both `ralph doctor` and the local verifier catch duplicate hook representations. | `scripts/check-sync.sh` PASS. |
| `.codex/config.toml` and template mirror | Updated inline hook comment to name both `ralph doctor` and `scripts/verify.local.sh` as drift guards. | `scripts/check-sync.sh` PASS. |
| `AGENTS.md`, `README.md`, `CLAUDE.md` | No change needed; they describe surfaces and setup at a higher level. | No behavior or workflow order change. |

## Verdict

Pass.
