# Test report: ralph-pipeline-hardening v2

- Date: 2026-04-15
- Plan: `docs/plans/active/2026-04-15-ralph-pipeline-hardening/`
- Tester: `tester` subagent (shell)
- Scope: All test suites + spot checks for ralph-pipeline-hardening branch
- Evidence: `docs/evidence/test-2026-04-15-ralph-pipeline-hardening-v2.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `tests/test-ralph-config.sh` | 23 | 23 | 0 | 0 | <1s |
| `tests/test-ralph-signals.sh` | 3 | 3 | 0 | 0 | <1s |
| `tests/test-ralph-status.sh` | 40 | 40 | 0 | 0 | <1s |
| `./scripts/run-verify.sh` | 1 (scaffold gate) | 1 | 0 | 0 | <1s |
| Spot check: numeric validation rejects `abc` | 1 | 1 | 0 | 0 | <1s |
| Spot check: env override `RALPH_MODEL=sonnet` | 1 | 1 | 0 | 0 | <1s |
| Spot check: no hardcoded `--model`/`--effort` values | 1 | 1 | 0 | 0 | <1s |
| **Total** | **70** | **70** | **0** | **0** | **<3s** |

## Coverage

- Statement: N/A (POSIX shell -- no instrumented coverage tool)
- Branch: N/A
- Function: N/A
- Notes: Coverage is assessed by test case scope, not instrumented metrics. All exported functions in `ralph-config.sh`, `ralph-status.sh`, and signal handling in `ralph-signals.sh` are covered by the test suites. The spot checks verify specific hardening behaviors (numeric validation, env overrides, no hardcoded values).

### Covered areas

| Module | Coverage level |
| --- | --- |
| `ralph-config.sh` defaults | Full (9 config vars) |
| `ralph-config.sh` env overrides | Sampled (4 overrides) |
| `ralph-config.sh` `validate_numeric` | Full (6 rejection + 2 acceptance cases) |
| `ralph-config.sh` `validate_all_numeric` | Partial (2 cases: defaults pass, bad value rejects) |
| `ralph-signals.sh` SIGINT cleanup | Full (orphan check, loop status, orchestrator JSON) |
| `ralph-status.sh` helpers | Full (format_duration, iso_to_epoch, checkpoint, progress_bar, estimate_eta) |
| `ralph-status.sh` table rendering | Full (12 assertions) |
| `ralph-status.sh` JSON rendering | Full (8 assertions) |
| `ralph-status.sh` no-color | Covered |
| `ralph-status.sh` no-state | Covered |
| `ralph-status.sh` whitespace trimming | Covered |
| Pipeline hardcoded value elimination | Verified via grep spot check |

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (none) | -- | -- | -- |

No failures detected.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Hardcoded `--model opus` in pipeline scripts | Fixed -- grep returns 0 matches | Spot check 7 in evidence log |
| Hardcoded `--effort high`/`--effort max` in pipeline scripts | Fixed -- grep returns 0 matches | Spot check 7 in evidence log |
| `validate_numeric` rejecting invalid input | Working correctly | Suite 1 + spot check 5 |
| Env var overrides lost after sourcing config | Working correctly | Suite 1 + spot check 6 |

## Test gaps

1. **ralph-orchestrator.sh multi-slice integration**: Requires mock `claude` CLI to test actual orchestration. No test exists.
2. **ralph-pipeline.sh inner/outer loop phases**: Depends on `claude -p` invocations. Cannot be tested without a mock.
3. **Concurrent status file writes**: Multiple slices writing to `orchestrator.json` simultaneously is untested.
4. **RALPH_SLICE_TIMEOUT actual enforcement**: The timeout variable is validated as numeric, but actual `timeout(1)` enforcement in `ralph-loop.sh` is not tested.
5. **`--resume` checkpoint recovery path**: No test for resuming from a checkpoint after interruption.
6. **Language-specific verifiers**: No language packs installed in scaffold repo; `run-verify.sh` exits 2 as expected.
7. **`--permission-mode bypassPermissions` flag propagation**: Added by recent commit `8c640d2` but not tested (requires mock `claude` CLI).
8. **Dependency slug resolution**: Added by commit `085ae31` (short slug to full slug mapping in orchestrator) but not tested.

## Flaky test notes

- `test-ralph-signals.sh / test_loop_sigint`: In dry-run mode with small `--max-iterations`, the loop may complete before SIGINT arrives. The test accepts both "interrupted" and terminal status ("stuck" appeared this run, meaning the loop completed before the signal). This is not a true flake but is timing-dependent.

## Verdict

- **Pass**: YES -- 70/70 tests passed, 0 failures, all spot checks green
- Fail: No
- Blocked: No

The test suite is healthy and ready for PR creation.
