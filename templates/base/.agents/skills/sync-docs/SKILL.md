---
name: sync-docs
description: Sync plans, docs, and instruction files after behavior, commands, contracts, or workflows change. Also covers harness-internal consistency after skill, hook, rule, or script changes. Invoked after /test and before /cross-review as the doc-maintainer agent; inline execution is only a dispatch-failure fallback.
---
Use this skill when implementation or harness structure changed enough that documentation may have drifted.

## Product-level sync

Update, as needed:
- active plan progress
- `README.md`
- `AGENTS.md`
- `CLAUDE.md`
- `.claude/rules/`
- `docs/quality/`
- `docs/reports/` links or references

Keep `AGENTS.md` short and stable. If a new rule is path- or topic-specific, put it in `.claude/rules/` instead.

## Harness-internal sync

When skills, hooks, rules, scripts, or language packs changed, also check:

- **Skills added/removed/renamed**: Does `AGENTS.md` Repo map still reflect the skill set? Does `README.md` list the current operating loop?
- **Hooks added/removed**: Does `.claude/settings.json` reference the correct hook scripts? Are removed hooks cleaned out?
- **Rules added/removed**: Does `.claude/rules/` match the languages and topics actually in the project? Are `paths:` globs still accurate?
- **Language packs added/removed**: Does `scripts/detect-languages.sh` detect the language? Is there a matching `.claude/rules/ralph/<lang>.md`? Does `packs/languages/<lang>/verify.sh` run a real verifier (not the placeholder)?
- **Scripts added/removed**: Does `README.md` Quick Start still reference valid scripts? Does `docs/architecture/repo-map.md` list the current scripts?
- **Quality gates changed**: Does `docs/quality/definition-of-done.md` match the actual completion workflow in `/work`? Does `docs/quality/quality-gates.md` list verifiers that actually exist?
- **PR skill consistency**: Does `/pr` SKILL.md pre-checks align with `/self-review`, `/verify`, and `/test` output? Does the PR template match the current plan template fields? Does `AGENTS.md` primary loop include the PR step?

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
