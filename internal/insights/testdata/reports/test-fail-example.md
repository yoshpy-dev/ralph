# Test report: some-failing-task

- Date: 2026-07-11
- Plan: docs/plans/active/2026-07-11-some-failing-task.md

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `bash tests/test-example.sh` | 10 | 8 | 2 | 0 | ~1s |

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| test_case_a | assertion failed | logic error | fix condition |
| test_case_b | timeout | network issue | mock network |

## Verdict

- Pass: 8/10
- Fail: 2
- Blocked: yes — 2 failures
