---
name: audit-harness
description: Audit the harness itself for drift, weak spots, overgrown instructions, missing deterministic checks, or language-pack gaps.
disable-model-invocation: true
allowed-tools: Read, Grep, Glob, Bash, Write
---
Audit the harness, not the product code alone.

## Inspect

- `AGENTS.md` and `CLAUDE.md`
- `.claude/rules/`
- `.claude/skills/`
- `.claude/hooks/`
- `scripts/run-verify.sh`
- `packs/languages/`
- CI and report templates

## Questions

- Is always-on context too large?
- Are there rules that should become scripts, tests, or hooks?
- Are there repeated review comments that justify automation?
- Are there missing language packs or pack-specific verifiers?
- Are reports and plans actually helping, or only adding ceremony?
- Is the harness complexity still justified by the current task and model quality?

## Output

Write a short audit memo with:
- strengths
- pain points
- missing guardrails
- proposed promotions from prose to code
- simplifications worth trying
