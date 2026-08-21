@AGENTS.md

# Claude Code

Use this file only for Claude-specific guidance that must be always-on.

Ralph's always-on workflow guidance auto-loads as a project rule from
`.claude/rules/ralph/ralph-workflow.md`; no need to duplicate it here.

## Default behavior (meta-repo addition)

- Manual-trigger skills (`disable-model-invocation: true`): `/release` only (cut a Homebrew release tag for the `ralph` CLI — repo-maintainer only, not included in `ralph init`). Every other skill listed in `.claude/rules/ralph/ralph-workflow.md` is auto-invoked in this repo too. `anti-bottleneck` is a model-internal support skill (`user-invocable: false`) and belongs to neither list.
- Autonomous multi-seat execution outside the interactive flow is the org runtime's job (`ralph org spawn/send/wait/...`); see `docs/specs/2026-08-01-org-runtime.md` and `.claude/rules/ralph/agent-messaging.md`.

## Claude-specific directories (meta-repo addition)

- `.claude/skills/` drift-checked against `.agents/skills/` by `scripts/check-skill-sync.sh`, which also ships to scaffolded projects (`templates/base/scripts/check-skill-sync.sh`)
- `.claude/hooks/` Codex equivalents live under `.codex/hooks.json` for the meta-repo and `templates/base/.codex/hooks.json` for scaffolded projects; the two are kept byte-identical
