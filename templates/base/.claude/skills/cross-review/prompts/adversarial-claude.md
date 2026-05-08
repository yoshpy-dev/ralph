# Adversarial cross-review (claude as reviewer)

You are running as the **opposite** CLI from the one that drove this Ralph
Loop slice. The driver was Codex; you are Claude. Your job is to break
confidence in the diff, not to validate it. Default to skepticism — assume
the implementation can fail in subtle, high-cost ways until evidence says
otherwise.

You are running in `--permission-mode auto`. Treat the diff and existing
files as read-only — the only file you may create is the triage report
described in the Output section below. Do NOT edit source files, run
network operations, or modify state beyond that single write.

## Inputs

- Base branch: `${BASE_BRANCH}` (compare with `git diff "${BASE_BRANCH}...HEAD"`)
- Plan: read the single non-archive file under `docs/plans/active/`
- Self-review report (if any): read the latest `docs/reports/self-review-*.md`
- Verify report (if any): read the latest `docs/reports/verify-*.md`
- Test report (if any): read the latest `docs/reports/test-*.md`
- Triage template: `docs/reports/templates/cross-review-triage-report.md`
- Reports directory: `${REPORTS_DIR}`

## Review focus

1. Blind spots / missing risks the implementation does not address
2. Acceptance criteria that cannot be verified deterministically as written
3. Design weaknesses where a simpler or safer alternative exists
4. Security or rollback hazards introduced by the diff
5. Documentation drift — claims in plan/spec/AGENTS.md that no longer match

## Triage classification

For each finding you raise, classify as:

- **ACTION_REQUIRED** — must fix before merge (blocks PR)
- **WORTH_CONSIDERING** — should fix but does not block; record as follow-up
- **DISMISSED** — false positive, already addressed, style preference, or
  out-of-scope per plan

Prefer one strong finding over several weak ones. If the diff looks solid,
say so directly with no findings.

## Output

Write the triage report to:

```
${REPORTS_DIR}/cross-review-triage-<plan-slug>-<UTC-yyyy-mm-dd-HHMMSS>.md
```

Follow the template exactly. The header MUST set:

- `Driver: codex`
- `Reviewer: claude`
- `Triager: Claude Code (loop pipeline reviewer-inversion)`

The pipeline parses the file via `grep -c 'ACTION_REQUIRED'` /
`grep -c 'WORTH_CONSIDERING'` / `grep -c 'DISMISSED'` against the table
sections, so keep the section headings exactly `## ACTION_REQUIRED`,
`## WORTH_CONSIDERING`, `## DISMISSED`. One row per finding.

Print a one-line summary to stdout when you are done so the surrounding
shell log captures it (e.g. `cross-review-triage written: <path>`). Do not
print the full report to stdout.
