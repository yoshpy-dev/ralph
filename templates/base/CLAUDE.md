@AGENTS.md

# Claude Code

Use this file only for Claude-Code-specific guidance that must be always-on.
Shared workflow rules live in `AGENTS.md` (read by both CLIs). Codex-specific
guidance lives in `.codex/AGENTS.override.md` and `.codex/README.md`.

## Default behavior

- `/spec` is the only manual-trigger skill. All others (`/plan`, `/work`,
  `/loop`, `/self-review`, `/verify`, `/test`, `/sync-docs`, `/cross-review`,
  `/pr`, `/audit-harness`) are auto-invoked.
- Use `/spec` when the request is too vague for `/plan`. `/spec` refines
  abstract ideas into detailed specifications (`docs/specs/`) through iterative
  brainstorming (壁打ち), codebase exploration, web research, and interactive
  clarification. It can hand off to `/plan` or create a GitHub issue.
- Use `/plan` before risky, ambiguous, or multi-file work. It does not create a
  branch — branch/worktree creation is deferred to the chosen flow skill.
- `/work` creates a normal branch (`git checkout -b`) and starts interactive
  implementation. Post-impl pipeline runs via subagents.
- `/loop` uses a directory-based plan and runs `ralph-orchestrator.sh` for
  autonomous parallel-slice execution. Use `./scripts/ralph run` or
  `./scripts/ralph status` to operate.
- After `/work`, the post-implementation pipeline runs via subagents
  (`/self-review` → `/verify` → `/test` → `/sync-docs`), then `/cross-review`
  (optional, inline), then `/pr`.
- `/self-review` is diff quality only. `/verify` is spec compliance + static
  analysis. `/test` is behavioral tests. Each produces a separate report.
- Codex advisory is optional. If `codex` CLI is available, `/plan` and
  `/cross-review` invoke it for second-opinion feedback. If unavailable, the
  step is silently skipped and the flow continues unchanged.
- Codex findings are presented to the user for judgment — never auto-applied.
- `/pr` creates the pull request, archives the plan, and completes the
  hand-off. A task is "done" when the PR is created.
- Subagent execution model: in `/work`, the post-impl pipeline runs via
  `Task(subagent_type=...)` calls (`reviewer`, `verifier`, `tester`,
  `doc-maintainer`). In Ralph Loop, the same pipeline runs internally via
  dedicated `claude -p` prompts per slice. See
  `.claude/rules/subagent-policy.md`.
- Run `./scripts/run-verify.sh` or an equivalent deterministic check before
  claiming success.
- If context is getting crowded, checkpoint progress in the active plan before
  compaction.
- Keep this file small; if a rule grows, move it out.

## Claude Code surfaces

- `.claude/rules/` — conditional rules (also read by Codex)
- `.claude/skills/` — Claude Code skill bodies (mirrored in `.agents/skills/`)
- `.claude/agents/` — Claude Code subagent definitions (no Codex equivalent)
- `.claude/hooks/` — Claude Code runtime hooks (Codex equivalents in
  `.codex/config.toml` `[hooks]`)
