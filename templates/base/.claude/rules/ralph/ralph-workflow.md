# Ralph workflow

This file is auto-loaded as a project rule; it carries the always-on ralph
workflow guidance formerly in CLAUDE.md.

## Default behavior

- All skills (`/spec`, `/plan`, `/work`, `/self-review`, `/verify`,
  `/test`, `/sync-docs`, `/cross-review`, `/pr`, `/audit-harness`, `/org`) are
  auto-invoked. The scaffold ships no manual-trigger skill.
- Use `/spec` when the request is too vague for `/plan`. `/spec` refines
  abstract ideas through decision-tree questioning with recommended answers,
  codebase exploration, web research, and interactive clarification. Issue-only
  specs use a temporary clean-base worktree and cleanup; saved specs either
  create a docs/spec PR or hand off to `/plan` in the same task worktree.
- Use `/plan` before risky, ambiguous, or multi-file work. `/plan` ensures a
  clean-base task worktree before writing plan artifacts.
- `/work` resumes the task worktree and starts interactive implementation.
  Post-impl pipeline runs via subagents.
- Autonomous multi-seat execution outside this interactive flow is the org
  runtime's job (`ralph org spawn/send/wait/...`). See the org runtime spec
  shipped with your project and `.claude/rules/ralph/agent-messaging.md`.
- After `/work`, the post-implementation pipeline runs via subagents
  (`/self-review` → `/verify` → `/test` → `/sync-docs`), then `/cross-review`
  (optional, inline), then `/pr`.
- `/self-review` is diff quality only. `/verify` is spec compliance + static
  analysis. `/test` is behavioral tests. Each produces a separate report.
- Codex advisory is optional. If the `codex` binary is available, `/plan` and
  `/cross-review` invoke Codex for second-opinion feedback. If unavailable, the
  step is silently skipped and the flow continues unchanged.
- Codex findings are presented to the user for judgment — never auto-applied.
- `/pr` creates the pull request, archives the plan, cleans up the task
  worktree/local branch, and completes the hand-off. A task is "done" when the
  PR is created and cleanup has either succeeded or reported recoverable state.
- Subagent execution model: in `/work`, the post-impl pipeline runs via
  `Task(subagent_type=...)` calls (`reviewer`, `verifier`, `tester`,
  `doc-maintainer`). See `.claude/rules/ralph/subagent-policy.md`.
- Run `./scripts/run-verify.sh` or an equivalent deterministic check before
  claiming success.
- If context is getting crowded, checkpoint progress in the active plan before
  compaction.
- Keep this file small; if a rule grows, move it out.

## Claude Code surfaces

- `.claude/rules/` — conditional rules (also read by Codex)
- `.claude/skills/` — Claude Code skill bodies (mirrored in `.agents/skills/`; regenerate with `scripts/sync-skills.sh`)
- `.claude/agents/` — Claude Code subagent definitions (Codex custom agents live under `.codex/agents/`)
- `.claude/hooks/` — Claude Code runtime hooks (Codex equivalents in
  `.codex/config.toml` `[hooks]`)
