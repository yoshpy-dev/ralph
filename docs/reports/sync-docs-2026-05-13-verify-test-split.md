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
| Claude/Codex verifier agents | Explicitly require `./scripts/run-static-verify.sh` and forbid `./scripts/run-test.sh` / behavioral tests. |
| Claude/Codex tester agents | Explicitly require `./scripts/run-test.sh` and forbid `./scripts/run-static-verify.sh`, static analyzers, type checks, and drift checks. |
| `.claude/rules/post-implementation-pipeline.md` | Clarified phase responsibilities for self-review, verify, and test in the standard pipeline table. |
| `templates/base/` mirrors | Kept scaffolded docs, prompts, rules, and agent surfaces synchronized. |

## Verification

| Command | Result |
| --- | --- |
| `./scripts/check-sync.sh` | pass via `./scripts/run-static-verify.sh` |
| `./scripts/check-skill-sync.sh` | pass via `./scripts/run-static-verify.sh` |
| `./scripts/check-pipeline-sync.sh` | pass via `./scripts/run-static-verify.sh` |
| `tests/test-agent-phase-boundaries.sh` | pass; 44/44 assertions |

## Drift Assessment

No remaining documentation drift found for the changed phase boundaries.
