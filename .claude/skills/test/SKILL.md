---
name: test
description: Run behavioral tests (unit, integration, regression) and produce a test report. Tests must pass before PR creation. Invoke automatically after /verify completes.
allowed-tools: Read, Grep, Glob, Bash, Write
---
Run tests and write a report to `docs/reports/`.

## Preferred flow

1. Read the active plan and its test plan section.
2. Run `./scripts/run-test.sh` (tests only) unless there is a stronger project-specific test runner.
3. Capture test results, coverage, and failure analysis in a report from [template.md](template.md).
4. Save raw test output to `docs/evidence/test-<date>-<slug>.log`.
5. If no tests exist or test infrastructure is missing, say so explicitly and propose the smallest useful test to add.
6. Distinguish:
   - passing
   - failing (with root cause analysis)
   - skipped (with reason)

## Test categories

- **Normal path**: Expected inputs produce expected outputs
- **Error path**: Invalid inputs, missing dependencies, boundary conditions
- **Regression**: Previously broken behavior stays fixed

## Gate

**Tests must pass before PR creation.** If any test fails:
- Record the failure in the report
- Do NOT proceed to /pr
- Propose a fix or flag the failure for human decision

## What /test does NOT do

- **Static analysis**: That is the responsibility of `/verify`.
- **Diff quality**: That is the responsibility of `/review`.
- **Spec compliance**: That is the responsibility of `/verify`.

## Output

- `docs/reports/test-<date>-<slug>.md` — human-readable summary
- `docs/evidence/test-<date>-<slug>.log` — raw test output
- clear pass/fail verdict
- explicit test gaps
