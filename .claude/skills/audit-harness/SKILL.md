---
name: audit-harness
description: Audit the harness itself for drift, weak spots, overgrown instructions, missing deterministic checks, or language-pack gaps. Invoke automatically when harness-level changes accumulate and need consistency review.
allowed-tools: Read, Grep, Glob, Bash, Write
---
Audit the harness, not the product code alone.

## Inspect

- `AGENTS.md` and `CLAUDE.md`
- `.claude/rules/`
- `.claude/skills/` (including `/test` skill)
- `.claude/hooks/`
- `.codex/config.toml`, `.codex/AGENTS.override.md`, `.codex/README.md`, `.codex/hooks/`
- `.agents/skills/` (Codex skill mirrors)
- `scripts/run-verify.sh`, `scripts/run-static-verify.sh`, `scripts/run-test.sh`, `scripts/check-skill-sync.sh`
- `packs/languages/`
- CI and report templates

## AGENTS.md size budget

Codex caps the merged AGENTS.md prompt at `project_doc_max_bytes`
(default **32 KiB**, see [config-reference](https://developers.openai.com/codex/config-reference)).
Anything beyond that is silently truncated, which strips the bottom of the
file with no warning. Run during the audit:

```sh
size="$(wc -c < AGENTS.md)"
if [ "$size" -gt 24576 ]; then
  echo "WARN: AGENTS.md=${size} bytes (>24 KiB)"
fi
if [ "$size" -gt 32768 ]; then
  echo "FAIL: AGENTS.md=${size} bytes (>32 KiB Codex cap)"
fi
```

Promote anything that crosses the warning threshold into `.claude/rules/`,
`.claude/skills/`, or `.codex/AGENTS.override.md` instead of bloating the
shared map. Record the size and remediation in the audit memo.

## Questions

- Is always-on context too large?
- Are there rules that should become scripts, tests, or hooks?
- Are there repeated review comments that justify automation?
- Are there missing language packs or pack-specific verifiers?
- Are reports and plans actually helping, or only adding ceremony?
- Is the harness complexity still justified by the current task and model quality?
- Do `/self-review`, `/verify`, and `/test` have clear non-overlapping responsibilities?

## Quality gate alignment

Check whether `docs/quality/` still matches reality:

- Does `docs/quality/definition-of-done.md` reflect the actual completion workflow? Compare against `/work`, `/self-review`, `/verify`, `/test`, and `/pr` skill steps.
- Does `docs/quality/quality-gates.md` list the verifiers and CI checks that actually exist in `scripts/` and `.github/workflows/`?
- Are there new verification tools, linters, or test frameworks in use that are not mentioned in the quality gates?
- Are there gates listed that no longer apply or have been removed?

If drift is found, update the quality docs or flag them in the audit memo.

## Output

Write a short audit memo with:
- strengths
- pain points
- missing guardrails
- proposed promotions from prose to code
- simplifications worth trying

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
