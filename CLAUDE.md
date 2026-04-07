@AGENTS.md

# Claude Code

Use this file only for Claude-specific guidance that must be always-on.

## Default behavior

- `/plan` is the only manual-trigger skill. All others (work, review, verify, pr, sync-docs, audit-harness, loop) are auto-invoked.
- Use `/plan` before risky, ambiguous, or multi-file work. It creates the branch.
- After /work, proceed through /review, /verify, then /pr automatically.
- `/pr` creates the pull request, archives the plan, and completes the hand-off. A task is "done" when the PR is created.
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
