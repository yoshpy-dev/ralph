# Codex agent overrides

Loaded by Codex on top of the project-root `AGENTS.md` (later files win in Codex's
root-down concatenation). This file applies only to the Codex CLI; Claude Code
ignores it.

## Codex execution model

- **Skill invocation**: use `$skill-name` mention syntax (e.g. `$spec`, `$work`)
  or pick from the `/skills` menu. Do **not** type `/skill-name` — the leading
  slash collides with Codex built-ins (`/plan`, `/review`, `/status`, etc.) and
  triggers the wrong handler.
- **Subagents**: ralph runs the post-implementation pipeline (`self-review` →
  `verify` → `test` → `sync-docs`) **sequentially in this single agent**. Do
  not spawn parallel agents. Reports go to `docs/reports/*.md`.
- **Interactive prompts**: when a skill needs the operator to choose between
  options, present a numbered list and ask the operator to reply with a single
  digit. Treat that as the equivalent of Claude Code's `AskUserQuestion`.
- **Cross-model review**: the `/cross-review` skill runs `claude -p` to get a
  Claude second opinion when Codex is the primary driver. Set
  `RALPH_PRIMARY_CLI=codex` before invoking the post-implementation pipeline so
  the skill picks the correct reviewer side.

## Permission and sandbox mapping

| ralph concept | Codex equivalent | Default in this template |
|---------------|------------------|---------------------------|
| auto-approve safe writes | `sandbox_mode = "workspace-write"` | enabled |
| confirm risky tools only | `approval_policy = "on-request"` | enabled |
| project-level hooks | `[features] codex_hooks = true` + project trust | enabled, requires `codex trust .` |
| autonomous loops | `approval_policy = "never"` | **not** enabled by default |

If `ralph doctor` reports any of these as missing, follow the remediation it
suggests before relying on the harness.

## What Codex must not do

- Do not edit `.claude/skills/`, `.claude/agents/`, or `.claude/hooks/` — those
  are Claude Code's surface. Edit `.agents/skills/` and `.codex/` instead, and
  let `scripts/check-skill-sync.sh` keep them in step.
- Do not invent skill names. Use the inventory in `.agents/skills/`.
- Do not bypass `./scripts/run-verify.sh` before claiming work is done.
