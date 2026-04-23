# Test report: pipeline-max-cycles-cap

- Date: 2026-04-23
- Plan: `docs/plans/active/2026-04-23-pipeline-max-cycles-cap.md`
- Tester: Claude Code (tester subagent)
- Branch: `feat/pipeline-max-cycles-cap`
- Scope: Behavioral verification of new default `RALPH_MAX_OUTER_CYCLES=2` and new variable `RALPH_STANDARD_MAX_PIPELINE_CYCLES` (default 2) including `validate_all_numeric` integration; regression of `tests/test-ralph-status.sh`; full `run-verify.sh`; `--help` string parity; edge cases for low/negative cap values.
- Evidence: `docs/evidence/test-2026-04-23-pipeline-max-cycles-cap.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `bash tests/test-ralph-config.sh` | 27 | 27 | 0 | 0 | ~1s |
| `bash tests/test-ralph-status.sh` | 40 | 40 | 0 | 0 | ~1s |
| `./scripts/run-verify.sh` (shellcheck + sh -n hooks + jq settings.json + mojibake tests + check-sync + go verifier) | - | all | 0 | 0 | ~10s |
| `tests/test-check-mojibake.sh` (inside run-verify) | 11 | 11 | 0 | 0 | <1s |
| Go `go test ./...` (inside run-verify) | 8 pkgs | 8 ok | 0 | 0 | cached |
| `ralph-pipeline.sh --help` grep for `default: 2` | 1 | 1 | 0 | 0 | <1s |
| Edge case: `RALPH_STANDARD_MAX_PIPELINE_CYCLES=1` accepted by `validate_all_numeric` | 1 | 1 | 0 | 0 | <1s |
| Edge case: `RALPH_STANDARD_MAX_PIPELINE_CYCLES=-3` rejected by `validate_numeric` | 1 | 1 | 0 | 0 | <1s |

### New assertions introduced by this plan (test-ralph-config.sh)

All four target new assertions are present in `tests/test-ralph-config.sh` and passed:

| # | New assertion | Result |
| --- | --- | --- |
| 1 | `PASS: default RALPH_STANDARD_MAX_PIPELINE_CYCLES` (default=2) | PASS |
| 2 | `PASS: override RALPH_STANDARD_MAX_PIPELINE_CYCLES=5` | PASS |
| 3 | `PASS: validate_all_numeric rejects bad RALPH_STANDARD_MAX_PIPELINE_CYCLES` (non-numeric) | PASS |
| 4 | `PASS: validate_all_numeric rejects zero RALPH_STANDARD_MAX_PIPELINE_CYCLES` | PASS |

Additionally the companion default change was verified:

- `PASS: default RALPH_MAX_OUTER_CYCLES` (now 2, existing assertion in suite)

## Coverage

- Statement: n/a (POSIX shell — no instrumented coverage tool). Go packages execute via `go test ./...` (all 8 testable packages ok, cached from earlier run; no code changes to Go packages in this plan).
- Branch: n/a for shell.
- Function: `validate_numeric` and `validate_all_numeric` both exercised (valid path, non-numeric path, zero path, negative path via explicit edge-case invocation).
- Notes:
  - Shell coverage measured by case scope. All added variable paths (`RALPH_STANDARD_MAX_PIPELINE_CYCLES` default, override, invalid, zero) are covered.
  - `--help` string update (`scripts/ralph-pipeline.sh`) is verified by grep assertion on live output; no unit test added but behavioral check is in place.
  - Runtime wiring of `RALPH_STANDARD_MAX_PIPELINE_CYCLES` into the skill docs (`.claude/skills/codex-review/SKILL.md`, `work/SKILL.md`, `pr/SKILL.md`) is documentation-level and is validated separately by `/verify` (static analysis + spec compliance). No end-to-end harness test exists for the cap-gating behavior under a real `/codex-review` run — see **Test gaps** below.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

No failures.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `tests/test-ralph-status.sh` (40 assertions, status rendering) | PASS | Results: 40/40 passed (evidence log) |
| Go packages unaffected by shell config changes | PASS | `ok` for all 8 packages in run-verify |
| `check-sync.sh` source↔template parity | PASS | `IDENTICAL: 107, DRIFTED: 0` in run-verify output |
| mojibake guard | PASS | 11/11 in run-verify output |

## Test gaps

- **No end-to-end test for standard-flow cap-gating.** The behavior where `/codex-review` reads `.harness/state/standard-pipeline/active-plan.json` and `cycle-count.json`, increments the counter, and presents the cap-reached AskUserQuestion at cycle==cap is documented in SKILL.md but not exercised by an automated test. This is expected (skill docs drive interactive flows) but worth noting as a test blind spot. A future shell-level smoke test could simulate the counter file and assert that cap-reached branching would trigger.
- **No test for `active-plan.json` fallback** (empty `docs/plans/active/` scenario per Open question). Documented behavior only.
- **No test for cross-session cycle-count persistence** (counter survives Claude Code session compaction). Not feasibly testable without a harness runner; documented assumption.
- **`--help` text is validated only via grep, not a golden-file diff.** Low risk since the assertion is on the substring `default: 2`.

## Verdict

- Pass: Yes — all mandatory test suites pass; 4 new assertions for `RALPH_STANDARD_MAX_PIPELINE_CYCLES` pass; `run-verify.sh` exits 0; help output advertises default 2; edge cases (cap=1 accepted, cap=-3 rejected) behave as specified.
- Fail: No
- Blocked: No

Proceed to `/sync-docs` and then `/codex-review` (optional) and `/pr`. Tests are not a blocker.

## Cycle 2 test

- Date: 2026-04-23
- Trigger: Codex ACTION_REQUIRED (×2) fixed in commit `e27102a` — `.claude/skills/work/SKILL.md` cycle-counter preservation on resume and `.claude/skills/codex-review/SKILL.md` Case B cap-reached AskUserQuestion gate. Fixes are docs-only (prompt text); no new test coverage required because they govern inline interactive flows.
- Focused re-run scope: `tests/test-ralph-config.sh` (regression) + `./scripts/run-verify.sh` (full gate). `test-ralph-status.sh` skipped — verifier cycle 2 flagged no status-helper regressions and no status code paths were touched.

| Suite / Command | Tests | Passed | Failed | Notes |
| --- | --- | --- | --- | --- |
| `bash tests/test-ralph-config.sh` | 27 | 27 | 0 | Unchanged from cycle 1; all 4 new `RALPH_STANDARD_MAX_PIPELINE_CYCLES` assertions still pass |
| `./scripts/run-verify.sh` | — | all | 0 | shellcheck + `sh -n` hooks + jq settings.json×2 + mojibake 11/11 + `check-sync.sh` (IDENTICAL 107, DRIFTED 0) + go verifier (8 pkgs ok); exit 0 |

### Verdict

- Cycle 2 pass: Yes. Re-tested suites green; docs-only fixes introduced no regression. Proceed to `/pr`.

## Cycle 3 test

- Date: 2026-04-23
- Trigger: Codex cycle-3 fixes in commit `12b87ee` — docs-only prompt text in `.claude/skills/work/SKILL.md` and `.claude/skills/codex-review/SKILL.md`. No runtime code changed.
- Focused re-run scope: `tests/test-ralph-config.sh` (regression) + `./scripts/run-verify.sh` (full gate).

| Suite / Command | Tests | Passed | Failed | Notes |
| --- | --- | --- | --- | --- |
| `bash tests/test-ralph-config.sh` | 27 | 27 | 0 | Unchanged from cycles 1–2 |
| `./scripts/run-verify.sh` | — | all | 0 | shellcheck + `sh -n` hooks + jq settings.json×2 + mojibake 11/11 + `check-sync.sh` (IDENTICAL 107, DRIFTED 0) + go verifier (8 pkgs ok); exit 0 |

### Verdict

- Cycle 3 pass: Yes. Docs-only fixes introduced no regression. Proceed to `/pr`.
