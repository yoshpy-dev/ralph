---
name: sync-docs
description: Sync plans, docs, and instruction files after behavior, commands, contracts, or workflows change. Also covers harness-internal consistency after skill, hook, rule, or script changes. Invoke automatically after behavior, workflow, or harness structure changes.
allowed-tools: Read, Grep, Glob, Write, Edit, Bash
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
- **Hooks added/removed**: Do `.claude/settings.minimal.example.json` and `.claude/settings.advanced.example.json` reference the correct hook scripts? Are removed hooks cleaned out of both profiles?
- **Rules added/removed**: Does `.claude/rules/` match the languages and topics actually in the project? Are `paths:` globs still accurate?
- **Language packs added/removed**: Does `scripts/detect-languages.sh` detect the language? Is there a matching `.claude/rules/<lang>.md`? Does `packs/languages/<lang>/verify.sh` run a real verifier (not the placeholder)?
- **Scripts added/removed**: Does `README.md` Quick Start still reference valid scripts? Does `docs/architecture/repo-map.md` list the current scripts?
- **Quality gates changed**: Does `docs/quality/definition-of-done.md` match the actual completion workflow in `/work`? Does `docs/quality/quality-gates.md` list verifiers that actually exist?
- **PR skill consistency**: Does `/pr` SKILL.md pre-checks align with `/self-review`, `/verify`, and `/test` output? Does the PR template match the current plan template fields? Does `AGENTS.md` primary loop include the PR step?
