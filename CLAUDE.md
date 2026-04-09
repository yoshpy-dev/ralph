@AGENTS.md

# Claude Code

Use this file only for Claude-specific guidance that must be always-on.

## Default behavior

- `/plan` is the only manual-trigger skill. All others (work, loop, self-review, verify, test, codex-review, pr, sync-docs, audit-harness) are auto-invoked.
- Use `/plan` before risky, ambiguous, or multi-file work. It does not create a branch — branch/worktree creation is deferred to the chosen flow skill.
- `/plan` ends with a flow selection prompt: standard (/work) or Ralph Loop (/loop). Follow the user's choice.
- `/work` creates a normal branch (`git checkout -b`) and starts implementation.
- `/loop` creates a Git Worktree (`git worktree add`) for isolated autonomous iteration.
- After /work or /loop, the post-implementation pipeline runs automatically via subagents (`/self-review` → `/verify` → `/test` → `/sync-docs`), then `/codex-review` (optional, inline), then `/pr`.
- `/self-review` is diff quality only. `/verify` is spec compliance + static analysis. `/test` is behavioral tests. Each produces a separate report.
- Codex advisory is optional. If `codex` CLI is available, `/plan` and `/codex-review` invoke it for second-opinion feedback. If unavailable, the step is silently skipped and the flow continues unchanged.
- Codex findings are presented to the user for judgment — never auto-applied.
- `/pr` creates the pull request, archives the plan, and completes the hand-off. A task is "done" when the PR is created.
- Prefer `.claude/rules/` for topic or path-specific guidance.
- Prefer `.claude/skills/` for workflows and reusable playbooks.
- Post-implementation pipeline (`/self-review` → `/verify` → `/test` → `/sync-docs`) always runs via subagents (`reviewer`, `verifier`, `tester`, `doc-maintainer`). See `.claude/rules/subagent-policy.md` for the full delegation policy.
- Run `./scripts/run-verify.sh` or an equivalent deterministic check before claiming success.
- If context is getting crowded, checkpoint progress in the active plan before compaction.
- Keep this file small; if a rule grows, move it out.

## Claude-specific directories

- `.claude/rules/` for conditional rules
- `.claude/skills/` for on-demand workflows
- `.claude/agents/` for specialized subagents
- `.claude/hooks/` for deterministic runtime controls
