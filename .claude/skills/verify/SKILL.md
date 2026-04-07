---
name: verify
description: Verify spec compliance and run static analysis for a change. Checks acceptance criteria, documentation drift, linters, and type checks. Does NOT run tests — that is /test. Invoke automatically after /self-review completes.
allowed-tools: Read, Grep, Glob, Bash, Write
---
Verify the current work against the plan's acceptance criteria and run static analysis.

## Preferred flow

1. Read the active plan and acceptance criteria.
2. **Spec compliance**: Walk through each acceptance criterion and record whether it is met, partially met, or not met, with evidence.
3. **Documentation drift**: Check whether behavior changes are reflected in docs, contracts, and rules. Flag any drift.
4. Run `./scripts/run-static-verify.sh` (static analysis only) unless there is a stronger project-specific verifier.
5. Capture commands, outcomes, failures, and coverage gaps in a report from [template.md](template.md).
6. Save raw verification output to `docs/evidence/verify-<date>-<slug>.log`.
7. If deterministic checks are missing, say so explicitly and propose the smallest useful verifier to add.
8. For UI or behavior-heavy work, add observational evidence such as screenshots, logs, traces, or walkthrough notes to `docs/evidence/`.
9. Distinguish:
   - verified
   - likely but unverified
   - unknown

## What /verify does NOT do

- **Tests**: Running tests is the responsibility of `/test`. Do not run `run-test.sh` here.
- **Diff quality**: That is the responsibility of `/self-review`.

## Output

- `docs/reports/verify-<date>-<slug>.md` — human-readable summary
- `docs/evidence/verify-<date>-<slug>.log` — raw verification output
- clear pass/fail/partial verdict
- explicit remaining gaps
