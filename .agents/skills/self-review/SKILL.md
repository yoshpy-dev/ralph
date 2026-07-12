---
name: self-review
description: Self-review the diff for code quality before formal verification. Covers naming, readability, unnecessary changes, typos, null safety, debug code, secrets, exception handling, security, and maintainability. Invoke automatically after /work completes or when significant code changes are staged.
---
Perform a self-review of the current diff and write a report to `docs/reports/`.

## Review scope — diff quality only

Focus exclusively on the diff itself. Do NOT evaluate spec compliance, test coverage, or documentation drift — those belong to `/verify` and `/test`.
Do NOT run tests, static analysis, formatters, linters, type checks,
spec-compliance verification, documentation drift checks, or broad unrelated
repo audits. Use `git diff` and targeted file reads only.

Evaluate the diff for:
1. **Unnecessary changes** — unrelated modifications, formatting-only diffs, accidental includes
2. **Naming** — clarity, consistency with surrounding code, grep-ability
3. **Readability** — function length, nesting depth, comment quality
4. **Typos and copy-paste errors**
5. **Null safety and defensive checks** — missing guards at boundaries
6. **Debug code** — leftover console.log, print, TODO markers, commented-out code
7. **Secrets and credentials** — hardcoded keys, tokens, passwords
8. **Exception handling** — swallowed errors, generic catches, missing error paths
9. **Security** — injection risks, XSS, CSRF, unsafe deserialization, path traversal
10. **Maintainability** — tight coupling, hidden side effects, magic numbers

## Review method

1. Inspect the active plan and changed files via `git diff`.
2. Prefer evidence from the diff and repository contracts over intuition.
3. Record findings in a report using [template.md](template.md).
4. Separate blocking issues from follow-up suggestions.
5. If any finding represents deferred work, known shortcuts, or accumulated complexity, append it to `docs/tech-debt/README.md` or create a dedicated file in `docs/tech-debt/`.
6. If there are no findings, say what was checked and what evidence supports that conclusion.

## Output

- `docs/reports/self-review-<date>-<slug>.md`
- severity-tagged findings
- merge or no-merge recommendation
- tech-debt entries in `docs/tech-debt/` if deferred work was identified

## Insight event (best-effort)

After writing the report, append one insight event (errors are non-fatal):
```
./scripts/insights-append.sh --slug <slug> --flow standard --phase self_review \
  --verdict pass --critical <N> --high <N> --medium <N> --low <N> --source skill || true
```
Use `--verdict fail` if CRITICAL findings block the review; set counts from the report.

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
