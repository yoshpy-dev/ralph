# Repo map

## Core files

- `AGENTS.md`: vendor-neutral map
- `CLAUDE.md`: Claude-specific always-on additions
- `README.md`: human entry point

## Claude control plane

- `.claude/rules/`: path-scoped or topic-scoped guidance
- `.claude/skills/`: on-demand workflows
- `.claude/agents/`: specialized subagents
- `.claude/hooks/`: deterministic hook scripts
- `.claude/settings.*.example.json`: hook profiles

## Process artifacts

- `docs/plans/active/`: in-flight plans
- `docs/plans/archive/`: completed plans
- `docs/reports/`: review, verify, and walkthrough reports
- `docs/quality/`: definition of done and gates
- `docs/tech-debt/`: known debt and follow-ups
- `docs/evidence/`: what counts as evidence

## Extensions

- `packs/languages/`: stack-specific rules and verification
- `scripts/`: bootstrap, plan creation, verification, status
- `examples/`: testing prompts and examples
- `.github/workflows/`: outer loop checks

## Runtime state

- `.harness/state/`: transient markers and summaries
- `.harness/logs/`: local logs
