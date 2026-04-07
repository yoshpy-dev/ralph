---
name: review
description: Produce a written review artifact for a change, covering correctness, security, maintainability, testing, and documentation gaps. Invoke automatically after /work completes or when significant code changes are staged.
allowed-tools: Read, Grep, Glob, Bash, Write
---
Perform a review and write a report to `docs/reports/`.

## Review method

1. Inspect the active plan and changed files.
2. Prefer evidence from the diff, tests, logs, and repository contracts over intuition.
3. Evaluate:
   - correctness
   - security and destructive-risk handling
   - maintainability and architecture fit
   - test coverage and verification quality
   - documentation drift
4. Record findings in a report using [template.md](template.md).
5. Separate blocking issues from follow-up suggestions.
6. If any finding represents deferred work, known shortcuts, or accumulated complexity, append it to `docs/tech-debt/README.md` or create a dedicated file in `docs/tech-debt/` with: debt item, impact, why deferred, trigger for paying it down, and related plan or report.
7. If there are no findings, say what was checked and what evidence supports that conclusion.

## Output

- `docs/reports/review-<date>-<slug>.md`
- severity-tagged findings
- merge or no-merge recommendation
- tech-debt entries in `docs/tech-debt/` if deferred work was identified
