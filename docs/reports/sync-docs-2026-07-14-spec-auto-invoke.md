# sync-docs report: spec-auto-invoke

- Date: 2026-07-14
- Plan: docs/plans/active/2026-07-14-spec-auto-invoke.md
- Branch: feat/spec-auto-invoke
- Agent: doc-maintainer

## Scope

Documentation drift sweep for the `/spec` trigger-policy change: `disable-model-invocation: true` removed from `.claude/skills/spec/SKILL.md` (and template mirrors). Checked all locations that enumerate skill trigger policies or reference `/spec` as a manual-trigger skill.

## Files checked

| File | Status | Notes |
|------|--------|-------|
| `docs/architecture/repo-map.md` | CLEAN | Line 40: "auto-invoked when a request is too vague for /plan" — correct |
| `docs/recipes/codex-setup.md` | CLEAN | `$spec # optional` is a user-flow hint, not a trigger-policy statement; `disable-model-invocation` mention on L82 is check-skill-sync.sh feature description |
| `docs/quality/definition-of-done.md` | CLEAN | No trigger-policy claims; pipeline order is correct |
| `docs/quality/quality-gates.md` | CLEAN | No `/spec` trigger references |
| `docs/architecture/design-principles.md` | CLEAN | No skill trigger references |
| `.codex/README.md` | CLEAN | `/spec` mention is invocation-syntax caution (do not use `/spec` in Codex — use `$spec`) |
| `.codex/AGENTS.override.md` | CLEAN | `$spec` mention is invocation-syntax example |
| `.claude/rules/subagent-policy.md` | CLEAN | "Spec — always inline" section correctly describes execution context, not trigger policy |
| `.claude/rules/post-implementation-pipeline.md` | CLEAN | No spec trigger references |
| `README.md` | CLEAN | L194: "Every step in the loop, including `/spec`, is auto-invoked. `/release` is the only manual-trigger skill" — correct |
| `CLAUDE.md` | CLEAN | "Manual-trigger skills: `/release` only" — correct |
| `AGENTS.md` | CLEAN | "Spec (auto, optional)" — correct |
| `templates/base/CLAUDE.md` | CLEAN | "The scaffold ships no manual-trigger skill" — correct |
| `templates/base/AGENTS.md` | CLEAN | "Spec (auto, optional)" — correct |
| `templates/base/docs/recipes/codex-setup.md` | CLEAN | Same as root copy — no trigger-policy drift |
| `templates/base/docs/quality/definition-of-done.md` | CLEAN | No trigger-policy claims |
| `templates/base/.codex/README.md` | CLEAN | Same as root `.codex/README.md` |
| `templates/base/.codex/AGENTS.override.md` | CLEAN | Same as root `.codex/AGENTS.override.md` |
| `.agents/skills/spec/SKILL.md` | CLEAN | Frontmatter: no `disable-model-invocation`; description updated to positive auto-invoke trigger conditions |
| `templates/base/.agents/skills/spec/SKILL.md` | CLEAN | Same as root mirror |
| `templates/base/.claude/skills/spec/SKILL.md` | CLEAN | Same as root skill |

## Drift found and fixed

**None.** All candidate files were already correct after the primary diff (changes to `CLAUDE.md`, `AGENTS.md`, `README.md`, `docs/architecture/repo-map.md`, `templates/base/CLAUDE.md`, `templates/base/AGENTS.md`, and both skill files).

No additional edits were required in this sync-docs pass.

## Gates re-run

```
./scripts/run-verify.sh
```

Result: **PASS** (exit 0)
Evidence: `docs/evidence/verify-2026-07-14-033440.log`

Key gates confirmed passing:
- check-skill-sync: all skills in lock-step
- golang verifier: gofmt ok, 0 vet issues, all Go tests pass
- scaffold renderer tests: 54 passed, 0 failed
- template check: clean

## Verdict

COMPLETE — no drift found, no fixes required, gates pass.
