# AGENTS.md

Treat this file as a **map**: short, stable, cross-vendor, easy to verify against the repo.

## Mission

Build coding-agent workflows that are reliable, inspectable, evidence-backed, and easy to extend.

## Primary loop

1. Spec (manual, optional)
2. Plan (auto)
3. Work (auto) or Loop (auto, parallel slices)
4. Self-review (auto)
5. Verify (auto)
6. Test (auto)
7. Sync-docs (auto)
8. Codex-review (auto, optional)
9. PR (auto)
10. CI verify + human merge

## Source of truth

- Repo files beat memory
- Versioned docs beat chat history
- Deterministic scripts beat informal promises
- Evidence beats confidence statements

## Repo map

<!-- Update this section to reflect your project's structure -->

- `docs/specs/` — spec files
- `docs/plans/active/` — current plans
- `docs/plans/archive/` — completed plans
- `docs/reports/` — review, verify, test artifacts
- `docs/quality/` — definition of done and quality gates
- `.claude/rules/` — path-scoped guidance
- `.claude/skills/` — on-demand workflows
- `.claude/agents/` — specialized subagents
- `.claude/hooks/` — deterministic runtime checks

## Planning contract

Every non-trivial task should have: objective, scope, acceptance criteria, verify plan, test plan, risk register, and evidence targets.

## Hard rules

- Keep this file short
- Keep `CLAUDE.md` short
- Move detailed guidance into `.claude/rules/`
- Move workflows into `.claude/skills/`
- Promote repeated mistakes into hooks, tests, CI, or scripts
- Update docs when behavior changes
