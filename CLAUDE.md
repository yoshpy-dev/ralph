@AGENTS.md

# Claude Code

Use this file only for Claude-specific guidance that must be always-on.

## Default behavior

- `/spec` is the only manual-trigger skill. All others (plan, work, loop, self-review, verify, test, codex-review, pr, sync-docs, audit-harness) are auto-invoked.
- Use `/spec` when the request is too vague for `/plan`. `/spec` refines abstract ideas into detailed specifications (`docs/specs/`) through codebase exploration, web research, and interactive clarification. It can then hand off to `/plan` or create a GitHub issue.
- Use `/plan` before risky, ambiguous, or multi-file work. It does not create a branch — branch/worktree creation is deferred to the chosen flow skill.
- `/plan` asks one decision: 標準フロー (/work) or Ralph Loop (/loop)。Follow the user's choice.
- `/work` creates a normal branch (`git checkout -b`) and starts interactive implementation. Post-impl pipeline runs via subagents.
- `/loop` uses a directory-based plan and runs `ralph-orchestrator.sh` for autonomous parallel-slice execution: multi-worktree (`git worktree add` × N) → `ralph-pipeline.sh` per slice → integration branch → sequential merge → integration pipeline (`--skip-pr --fix-all`) → unified PR.
- In Ralph Loop, the scripts handle the full lifecycle autonomously — no manual subagent chain needed. Use `./scripts/ralph run` or `./scripts/ralph status` to operate.
- After /work, the post-implementation pipeline runs via subagents (`/self-review` → `/verify` → `/test` → `/sync-docs`), then `/codex-review` (optional, inline), then `/pr`.
- `/self-review` is diff quality only. `/verify` is spec compliance + static analysis. `/test` is behavioral tests. Each produces a separate report.
- Codex advisory is optional. If `codex` CLI is available, `/plan` and `/codex-review` invoke it for second-opinion feedback. If unavailable, the step is silently skipped and the flow continues unchanged.
- Codex findings are presented to the user for judgment — never auto-applied.
- `/pr` creates the pull request, archives the plan, and completes the hand-off. A task is "done" when the PR is created.
- Prefer `.claude/rules/` for topic or path-specific guidance.
- Prefer `.claude/skills/` for workflows and reusable playbooks.
- In `/work`, the post-implementation pipeline (`/self-review` → `/verify` → `/test` → `/sync-docs`) runs via subagents (`reviewer`, `verifier`, `tester`, `doc-maintainer`). In Ralph Loop, the same pipeline runs internally via dedicated `claude -p` prompts per slice. See `.claude/rules/subagent-policy.md` for details.
- Run `./scripts/run-verify.sh` or an equivalent deterministic check before claiming success.
- If context is getting crowded, checkpoint progress in the active plan before compaction.
- Keep this file small; if a rule grows, move it out.

## Claude-specific directories

- `.claude/rules/` for conditional rules
- `.claude/skills/` for on-demand workflows
- `.claude/agents/` for specialized subagents
- `.claude/hooks/` for deterministic runtime controls
