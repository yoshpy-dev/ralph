@AGENTS.md

# Claude Code

Use this file only for Claude-specific guidance that must be always-on.

## Default behavior

- Use `/plan` before risky, ambiguous, or multi-file work.
- Use `/review` and `/verify` before presenting a task as complete.
- Prefer `.claude/rules/` for topic or path-specific guidance.
- Prefer `.claude/skills/` for workflows and reusable playbooks.
- Use `planner`, `reviewer`, `verifier`, and `doc-maintainer` subagents when they clearly reduce context pressure or improve auditability.
- Run `./scripts/run-verify.sh` or an equivalent deterministic check before claiming success.
- If context is getting crowded, checkpoint progress in the active plan before compaction.
- Keep this file small; if a rule grows, move it out.

## Claude-specific directories

- `.claude/rules/` for conditional rules
- `.claude/skills/` for on-demand workflows
- `.claude/agents/` for specialized subagents
- `.claude/hooks/` for deterministic runtime controls
