---
name: verify
description: Run or design evidence-backed verification for a change. Use before claiming a task is done, especially after non-trivial edits. Invoke automatically after /review completes or when deterministic verification is needed.
allowed-tools: Read, Grep, Glob, Bash, Write
---
Verify the current work using deterministic checks first.

## Preferred flow

1. Read the active plan and acceptance criteria.
2. Run `./scripts/run-verify.sh` unless there is a stronger project-specific verifier.
3. Capture commands, outcomes, failures, and coverage gaps in a report from [template.md](template.md).
4. Save raw verification output (test results, lint output, type-check logs) to `docs/evidence/verify-<date>-<slug>.log`. This preserves machine-readable evidence beyond the summary report.
5. If deterministic checks are missing, say so explicitly and propose the smallest useful verifier to add.
6. For UI or behavior-heavy work, add observational evidence such as screenshots, logs, traces, or walkthrough notes to `docs/evidence/`.
7. Distinguish:
   - verified
   - likely but unverified
   - unknown

## Output

- `docs/reports/verify-<date>-<slug>.md` — human-readable summary
- `docs/evidence/verify-<date>-<slug>.log` — raw verification output
- clear pass/fail/partial verdict
- explicit remaining gaps
