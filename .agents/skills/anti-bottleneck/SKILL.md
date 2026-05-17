---
name: anti-bottleneck
description: Load this skill BEFORE asking the user for confirmation, approval, next steps, or choices that can be resolved through verification, repo context, or reasonable defaults. Also load when you are unsure how to proceed and might otherwise stop early.
user-invocable: false
---
Human attention is the scarcest resource in the harness.

Before asking the user anything, check whether you can:
- inspect the codebase
- inspect plans or docs
- run tests or verification scripts
- gather logs or screenshots
- use a focused subagent
- choose a reasonable default and document it

Escalate only for:
- irreversible destructive actions
- external approvals that are genuinely required
- missing credentials or secrets
- product or design judgments that cannot be grounded in the repo or evidence

When stuck:
1. reduce scope
2. gather evidence
3. update the plan
4. try a different verifier or reviewer
5. present the best grounded answer you can

## CLI execution modes

This skill runs under both Claude Code and Codex. The execution mode follows
the conventions in AGENTS.md and `.codex/AGENTS.override.md`.

| Aspect | Claude Code | Codex |
|--------|-------------|-------|
| Skill invocation | `/skill-name` slash command | `$skill-name` mention or the `/skills` menu (avoid the `/skill-name` form — it collides with built-ins) |
| Skill body path | `.claude/skills/<name>/SKILL.md` | `.agents/skills/<name>/SKILL.md` |
| Subagent mechanism | `Task(subagent_type=...)` when a policy delegates | `.codex/agents/` custom agents when a policy delegates |
| Structured prompts | `AskUserQuestion` | Numbered options printed to stdout, awaiting a digit reply |
| Artifacts | `docs/reports/`, `docs/plans/`, `docs/specs/` (shared) | Same (CLI-agnostic) |

The drift check (`./scripts/check-skill-sync.sh`) cross-checks both bodies and
invocation metadata — editing only one side will fail CI.
