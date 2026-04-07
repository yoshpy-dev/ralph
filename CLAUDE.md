@AGENTS.md

# Claude Code

Use this file only for Claude-specific guidance that must be always-on.

## Default behavior

- `/plan` is the only manual-trigger skill. All others (work, review, verify, test, pr, sync-docs, audit-harness, loop) are auto-invoked.
- Use `/plan` before risky, ambiguous, or multi-file work. It does not create a branch — branch/worktree creation is deferred to the chosen flow skill.
- `/plan` ends with a flow selection prompt: standard (/work) or Ralph Loop (/loop). Follow the user's choice.
- `/work` creates a normal branch (`git checkout -b`) and starts implementation.
- `/loop` creates a Git Worktree (`git worktree add`) for isolated autonomous iteration.
- After /work or /loop, proceed through /review, /verify, /test, then /pr automatically.
- `/review` is self-review (diff quality only). `/verify` is spec compliance + static analysis. `/test` is behavioral tests. Each produces a separate report.
- `/pr` creates the pull request, archives the plan, and completes the hand-off. A task is "done" when the PR is created.
- Prefer `.claude/rules/` for topic or path-specific guidance.
- Prefer `.claude/skills/` for workflows and reusable playbooks.
- Use `planner`, `reviewer`, `verifier`, `tester`, and `doc-maintainer` subagents when they clearly reduce context pressure or improve auditability.
- Run `./scripts/run-verify.sh` or an equivalent deterministic check before claiming success.
- If context is getting crowded, checkpoint progress in the active plan before compaction.
- Keep this file small; if a rule grows, move it out.

## Claude-specific directories

- `.claude/rules/` for conditional rules
- `.claude/skills/` for on-demand workflows
- `.claude/agents/` for specialized subagents
- `.claude/hooks/` for deterministic runtime controls
