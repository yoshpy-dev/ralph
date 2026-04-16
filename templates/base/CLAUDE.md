@AGENTS.md

# Claude Code

Use this file only for Claude-specific guidance that must be always-on.

## Default behavior

- `/spec` is the only manual-trigger skill. All others are auto-invoked.
- Use `/plan` before risky, ambiguous, or multi-file work.
- After /work, the post-implementation pipeline runs: `/self-review` → `/verify` → `/test` → `/sync-docs` → `/codex-review` → `/pr`.
- Run `./scripts/run-verify.sh` or an equivalent deterministic check before claiming success.
- Keep this file small; if a rule grows, move it out.

## Claude-specific directories

- `.claude/rules/` for conditional rules
- `.claude/skills/` for on-demand workflows
- `.claude/agents/` for specialized subagents
- `.claude/hooks/` for deterministic runtime controls
