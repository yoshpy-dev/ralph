---
name: sync-docs
description: Sync plans, docs, and instruction files after behavior, commands, contracts, or workflows change.
disable-model-invocation: true
allowed-tools: Read, Grep, Glob, Write, Edit
---
Use this skill when the implementation changed enough that documentation may have drifted.

Update, as needed:
- active plan progress
- `README.md`
- `AGENTS.md`
- `CLAUDE.md`
- `.claude/rules/`
- `docs/quality/`
- `docs/reports/` links or references

Keep `AGENTS.md` short and stable. If a new rule is path- or topic-specific, put it in `.claude/rules/` instead.
