# Sync-docs report: verify-test-split

- Date: 2026-05-13
- Plan: `docs/plans/archive/2026-05-13-verify-test-split.md`
- Scope: verification/test/self-review phase contracts

## Documents Updated

| File | Change |
| --- | --- |
| `docs/quality/quality-gates.md` | Added explicit non-overlap contract for `run-static-verify.sh` and `run-test.sh`; clarified pipeline phase scopes. |
| `docs/quality/definition-of-done.md` | Made phase-boundary compliance part of done criteria. |
| `.agents/skills/self-review/SKILL.md` and `.claude/skills/self-review/SKILL.md` | Tightened self-review to diff quality only and forbade tests/static/spec/doc-drift/broad audits. |
| Pipeline self-review prompts and reviewer agents | Added the same scope boundary and targeted-read guidance. |
| `templates/base/` mirrors | Kept scaffolded docs, prompts, and reviewer surfaces synchronized. |

## Verification

| Command | Result |
| --- | --- |
| `./scripts/check-sync.sh` | pass via `./scripts/run-static-verify.sh` |
| `./scripts/check-skill-sync.sh` | pass via `./scripts/run-static-verify.sh` |
| `./scripts/check-pipeline-sync.sh` | pass via `./scripts/run-static-verify.sh` |

## Drift Assessment

No remaining documentation drift found for the changed phase boundaries.
