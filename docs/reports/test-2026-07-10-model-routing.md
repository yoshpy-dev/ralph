# Test report: model-routing

- Date: 2026-07-10
- Plan: docs/plans/active/2026-07-10-model-routing.md
- Tester: tester subagent (Claude Code)
- Scope: `./scripts/run-test.sh` (changed-language default scope) + explicit plan-relevant tests run out-of-scope (test-ralph-config.sh, Go config/scaffold/cli) + full `go test ./...` fresh
- Evidence: `docs/evidence/test-2026-07-10-model-routing.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (changed scope: 23 shell suites + golang pack) | 23 shell suites, all Go pkgs | all | 0 | 0 | ~ full run |
| `bash tests/test-ralph-config.sh` (explicit) | 27 | 27 | 0 | 0 | <1s |
| `go test -count=1 -run 'TestDefault\|TestLoad_FullRoundTrip' ./internal/config` | 3 | 3 | 0 | 0 | 0.59s |
| `go test -count=1 -run 'TestTemplateBaseRalphTomlHasLoopSection' ./internal/scaffold` | 1 | 1 | 0 | 0 | 0.45s |
| `go test -count=1 -run 'Doctor\|Loop' ./internal/cli` | doctor + loop-driver suites | all | 0 | 0 | 1.30s |
| `go test -count=1 ./...` (full, fresh, cache-busted) | 9 pkgs w/ tests | all `ok` | 0 | 3 no-test pkgs | ~34s aggregate |

Overall `./scripts/run-test.sh` exit code: **0** (verified via a clean re-run).

## Plan-relevant test callouts

- **tests/test-ralph-config.sh** — 27/27 PASS. Explicitly includes `default RALPH_MODEL` (asserts `opus`) and `default RALPH_EFFORT` (asserts `high`), plus env-override-priority cases (`override RALPH_MODEL=sonnet`, `override RALPH_EFFORT=low`). Confirms the new shell defaults and that env overrides still win. Note: this suite was NOT selected by the changed-language scope of `run-test.sh` (see Test gaps); it was run directly to validate the plan's default-value assertion update.
- **internal/config TestDefault / TestLoad_FullRoundTrip** — PASS (fresh). Confirms Go layer `Default()` now yields opus/high/opus and full round-trip load/save preserves them, matching the self-review HIGH fix (Scope item 10) that keeps the Go layer from masking new shell defaults at `ralph run` export time.
- **internal/scaffold TestTemplateBaseRalphTomlHasLoopSection** — PASS (fresh). Pins `templates/base/ralph.toml` loop section including `claude_reviewer_model="opus"` (Scope item 9 expected-value update).
- **internal/cli doctor loop tests** — PASS (fresh). `doctor_loop_test.go` fixtures use `ClaudeReviewerModel: "opus"` (confirmed at lines 42, 101, 130); `TestRunDoctor_Passes` and `TestCheckLoopDriver_*` all green.

## Coverage

- Statement/Branch/Function: not measured (repo test harness does not emit coverage numbers; shell suites are behavioral, Go suites run without `-cover` in the harness).
- Notes: The change is configuration/default-value routing. Behavioral coverage of the changed defaults is provided by test-ralph-config.sh (shell defaults + override priority), Go config tests (Default/round-trip), scaffold test (template toml pin), and cli doctor tests (ClaudeReviewerModel fixture). These are the correct assertion points for a default-tier change; no new logic branches were introduced that lack a test.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

No failures. All "fail"-containing log lines are PASS assertions verifying fail-closed behavior (e.g. "still-draft PR fails closed", "invalid branch fails closed").

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Downstream template `claude_reviewer_model` pinned to stale full model ID | Fixed / guarded | scaffold TestTemplateBaseRalphTomlHasLoopSection PASS (asserts `opus`) |
| Go `Default()` masking new shell defaults at `ralph run` export | Fixed / guarded | config TestDefault + TestLoad_FullRoundTrip PASS (opus/high/opus) |
| Env override priority over new defaults | Intact | test-ralph-config.sh override cases PASS |

## Test gaps

- **Scope selection gap (informational, not a defect):** The changed-language scope of `./scripts/run-test.sh` did not select `tests/test-ralph-config.sh` even though the diff touches `scripts/ralph-config.sh`. The 23 auto-selected shell suites all passed, but the single most plan-relevant shell suite was outside that set. It was run explicitly here (27/27 PASS), so the plan's assertion is validated. Consider confirming whether the changed-file→test mapping should associate `ralph-config.sh` edits with `test-ralph-config.sh` so the standard `run-test.sh` invocation covers it automatically.
- No coverage instrumentation in the harness; acceptable for a default-tier routing change where the assertion points are exact-value tests.

## Verdict

- Pass: YES — `./scripts/run-test.sh` exits 0; all 4 plan-relevant test groups PASS (fresh, cache-busted); full `go test ./...` all packages `ok`; test-ralph-config.sh 27/27.
- Fail: none
- Blocked: none

Tests pass — no blocker to /pr. One informational scope-mapping observation recorded above (not a gate failure).
