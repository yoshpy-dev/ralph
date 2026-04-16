# Test report: Ralph Pipeline Hardening

- Date: 2026-04-15
- Plan: feat/ralph-pipeline-hardening
- Tester: tester subagent (claude-opus-4-6)
- Scope: ralph-config.sh, ralph-signals, ralph-status, numeric validation, env overrides, hardcoded value elimination
- Evidence: `docs/evidence/test-2026-04-15-ralph-pipeline-hardening.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `tests/test-ralph-config.sh` | 23 | 23 | 0 | 0 | <1s |
| `tests/test-ralph-signals.sh` | 3 | 3 | 0 | 0 | ~2s |
| `tests/test-ralph-status.sh` | 40 | 40 | 0 | 0 | <1s |
| `./scripts/run-verify.sh` | 1 | 1 | 0 | 0 | <1s |
| Numeric validation rejection (`validate_numeric test abc`) | 1 | 1 | 0 | 0 | <1s |
| Environment variable override (`RALPH_MODEL=sonnet`) | 1 | 1 | 0 | 0 | <1s |
| Hardcoded value check (`grep --model opus / --effort high / --effort max`) | 1 | 1 | 0 | 0 | <1s |
| **Total** | **70** | **70** | **0** | **0** | ~3s |

## Coverage

- Statement: N/A (shell scripts -- no instrumented coverage tool)
- Branch: Partial (see test gaps below)
- Function: `validate_numeric`, `validate_all_numeric`, `format_duration`, `iso_to_epoch`, `render_progress_bar`, `estimate_eta`, `detect_color`, `resolve_display_phase`, `status_icon`, `_render_table`, `_render_json` -- all exercised
- Notes: Shell script coverage is measured by test case scope rather than instrumented line coverage. All public functions in `ralph-config.sh`, `ralph-status-helpers.sh` are exercised by the test suites.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (none) | -- | -- | -- |

No failures detected.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Hardcoded `--model opus` in pipeline/loop scripts | FIXED -- confirmed 0 grep matches | `grep -rn` returns exit code 1 (no matches) |
| Hardcoded `--effort high` / `--effort max` in pipeline/loop scripts | FIXED -- confirmed 0 grep matches | Same grep check |
| Missing numeric validation for config values | FIXED -- `validate_numeric` rejects abc, empty, negative, 0, float, mixed | test-ralph-config.sh suite |
| Environment variable overrides not respected | FIXED -- `RALPH_MODEL=sonnet` override confirmed | test-ralph-config.sh suite + ad hoc test |
| SIGINT leaves orphan processes | FIXED -- no orphans detected after SIGINT | test-ralph-signals.sh |
| Loop status not set on interrupt | FIXED -- status file written correctly | test-ralph-signals.sh |
| Status display with whitespace in status files | FIXED -- trimming works correctly | test-ralph-status.sh whitespace trimming tests |

## Test gaps

1. **ralph-orchestrator.sh integration test**: The orchestrator is only partially tested (SIGINT cleanup, dry-run). A full integration test of multi-slice orchestration with mock `claude` CLI would improve confidence but requires substantial test infrastructure.

2. **ralph-pipeline.sh end-to-end**: Pipeline phases (inner loop, outer loop, PR creation) are not tested in isolation because they depend on `claude -p` CLI. A mock-based approach (e.g., replacing `claude` with a script that echoes expected output) would enable deterministic testing.

3. **Edge case: concurrent writes to status files**: The whitespace trimming test covers reading, but concurrent write scenarios (e.g., two slices writing to orchestrator.json simultaneously) are not tested.

4. **Timeout enforcement (`RALPH_SLICE_TIMEOUT`)**: The timeout value is validated as numeric but the actual timeout enforcement mechanism in the orchestrator is not covered by tests.

5. **`--resume` checkpoint recovery**: The pipeline supports `--resume` from checkpoint, but no test exercises this path.

6. **run-verify.sh with actual language pack**: Verification passed with "docs/scaffold-only" detection. No language-specific verifier was exercised because no language pack is configured for this repository.

## Verdict

- Pass: **YES** -- 70/70 tests passed, 0 failures, 0 skipped
- Fail: No
- Blocked: No

All requested test suites passed. The hardening changes (centralized config, numeric validation, env overrides, hardcoded value elimination, signal handling, status display) are confirmed working. The branch is ready to proceed to `/pr`.
