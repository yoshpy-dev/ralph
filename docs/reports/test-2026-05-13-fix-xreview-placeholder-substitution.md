# Test report: fix-xreview-placeholder-substitution

- Date: 2026-05-13
- Plan: `docs/plans/active/2026-05-13-fix-xreview-placeholder-substitution.md` (issue #50)
- Tester: tester subagent (Claude Code)
- Scope: Six-suite regression run covering the placeholder renderer, the end-to-end gate-regression contract, existing CLI-driver coverage, skill-sync, mojibake guard, and the deterministic verifier (gofmt + staticcheck + `go test ./...`). All edge cases enumerated in the plan's Test plan section were exercised.
- Evidence: `docs/evidence/test-xreview-2026-05-13.log` (aggregate)
  - Per-test evidence from the implementation phase: `docs/evidence/test-xreview-prompt-render-2026-05-13.log`, `docs/evidence/test-xreview-gate-regression-2026-05-13.log`
  - Run-verify per-invocation logs: `docs/evidence/verify-2026-05-13-025819.log` (mode=test), `docs/evidence/verify-2026-05-13-025832.log` (mode=all)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Exit | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `./tests/test-xreview-prompt-render.sh` | 54 | 54 | 0 | 0 | 0 | 3 drift-guard assertions + 7 parameterized renderer cases (7 assertions each) + 2 allowlist negatives |
| `./tests/test-xreview-gate-regression.sh` | 21 | 21 | 0 | 0 | 0 | 5 phases: render, ACTION_REQUIRED, clean triage, --fix-all WORTH_CONSIDERING, RENDER_FAILED end-to-end |
| `./tests/test-ralph-cli-driver.sh` | 48 | 48 | 0 | 0 | 0 | 7 sections: driver=claude, driver=codex, codex fallback, DRY_RUN, pick_reviewer, cross-review dispatcher, count_triage_findings |
| `./tests/test-check-skill-sync.sh` | 6 | 6 | 0 | 0 | 0 | A–F: parity, inventory drift, body drift, description drift, policy drift, policy parity |
| `./tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 | 0 | A–F: U+FFFD trigger, clean UTF-8, missing path, allowlist, jq-missing fallback, Edit/Write/MultiEdit payload fixtures |
| `./scripts/run-verify.sh` (mode=test) | – | all | 0 | – | 0 | Embeds the 5 shell suites above + golang verifier (gofmt ok, staticcheck 0 issues, `go test ./...` all OK across 9 packages) |
| `./scripts/run-verify.sh` (mode=all) | – | all | 0 | – | 0 | Same plus static analysis (shellcheck, `sh -n`, jq settings.json, `scripts/check-sync.sh`) |

### Suite-level command + evidence

| Suite | Command (cwd = repo root) | Aggregate log section |
| --- | --- | --- |
| Renderer unit | `./tests/test-xreview-prompt-render.sh` | lines 1–76 of `docs/evidence/test-xreview-2026-05-13.log` |
| Gate regression | `./tests/test-xreview-gate-regression.sh` | lines 78–112 |
| CLI driver | `./tests/test-ralph-cli-driver.sh` | lines 114–181 |
| Skill sync | `./tests/test-check-skill-sync.sh` | lines 183–193 |
| Mojibake | `./tests/test-check-mojibake.sh` | lines 195–212 |
| run-verify (test) | `HARNESS_VERIFY_MODE=test ./scripts/run-verify.sh` | lines 214–336 |
| run-verify (all) | `./scripts/run-verify.sh` | lines 338–end |

## Coverage

- Statement / Branch / Function (Go): not separately profiled in this run. `go test ./...` ran across all 9 packages with tests (`internal/action`, `internal/cli`, `internal/config`, `internal/scaffold`, `internal/state`, `internal/ui`, `internal/ui/panes`, `internal/upgrade`, `internal/watcher`) — all OK, cached. The plan only requires the standard `./scripts/run-verify.sh` golang verifier, which does not gate on coverage thresholds.
- Shell coverage: no instrumented tool exists for POSIX shell scripts in this repo; coverage is by enumerated test case scope. The renderer + gate-regression suites together exercise every documented edge case in the plan.
- Notes:
  - Renderer drift guard (cases 0a–0c) asserts the pipeline source still contains the literal `lreplace` helper and calls it for both `BASE_BRANCH` and `REPORTS_DIR` — guards against future contributors swapping the helper without updating the test.
  - Gate-regression Phase 1 re-renders the live adversarial prompt (no fixture) and asserts both substitution and allowlist-guard contracts end-to-end.

## Edge cases — confirmation against test output

Each edge case the plan flagged is covered by an explicit assertion. All passed.

| Plan-flagged edge case | Suite / Phase | Assertion(s) | Result |
| --- | --- | --- | --- |
| `_base` containing `#` | Renderer Case 3 (`feature#1`) | 3.a–3.g | PASS |
| `_base` containing `&` | Renderer Case 4 (`feature&1`) | 4.a–4.g | PASS |
| `_base` containing `\` | Renderer Case 5 (`feature\back`) | 5.a–5.g | PASS |
| `_base` containing `/` | Renderer Case 2 (`release/3.5`) | 2.a–2.g | PASS |
| `REPORTS_DIR` containing `#` | Renderer Case 6 (`docs/reports#1`) | 6.a–6.g | PASS |
| `REPORTS_DIR` containing `&` | Renderer Case 7 (`docs/reports&backup`) | 7.a–7.g | PASS |
| Unsupported `${UNKNOWN}` placeholder | Renderer negative + Gate Phase 5a | neg.a, neg.b, 5a | PASS |
| `RENDER_FAILED=1` + non-existent triage file | Gate Phase 5b | 5b | PASS (regresses without consulting parser) |
| `RENDER_FAILED=0` + non-existent triage file | Gate Phase 5c | 5c | PASS (no false positive) |
| `ACTION_REQUIRED=1` triage report | Gate Phase 2 | 2a–2d | PASS (gate returns non-zero) |
| Clean triage (no findings) | Gate Phase 3 | 3a, 3b | PASS (gate proceeds) |
| `--fix-all` + `WORTH_CONSIDERING=1` | Gate Phase 4 | 4a, 4b (no --fix-all), 4c (with --fix-all) | PASS (regresses only under --fix-all) |

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

No failures.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Cross-review gate silently bypassed when `RALPH_LOOP_DRIVER=codex` triggers claude-reviewer with unrendered `${BASE_BRANCH}` / `${REPORTS_DIR}` (issue #50) | FIXED — gate now returns non-zero under ACTION_REQUIRED=1 and under render failure | Gate-regression Phase 2d (line 92) + Phase 5b (line 105) |
| `pick_reviewer` inversion and `count_triage_findings` summary-line parsing | NO REGRESSION | `tests/test-ralph-cli-driver.sh` sections 5 and 7 (48/48 pass) |
| Skill-sync drift gate | NO REGRESSION | `tests/test-check-skill-sync.sh` 6/6 pass |
| Mojibake hook | NO REGRESSION | `tests/test-check-mojibake.sh` 11/11 pass |
| Go test suite | NO REGRESSION | `go test ./...` all OK across 9 packages (lines 320–331, 833–842) |

## Test gaps

- **Live `claude -p` round-trip not exercised.** The gate-regression test uses a fake `claude` (per the plan's explicit design — reusing the `tests/fixtures/fake-claude` pattern). A reviewer LLM that hallucinates an alternate triage report path is documented as out of scope in the plan's risk register (Risk: "Reviewer LLM ignores the expanded path and writes elsewhere") and tracked as a follow-up if it manifests in practice.
- **No shell coverage tool.** Acceptable per existing repo conventions; case-based enumeration is the project standard.
- **Go coverage thresholds not enforced.** Consistent with `./scripts/run-verify.sh` semantics; not a regression introduced by this PR.

## Verdict

- Pass: yes — all six suites and both run-verify modes returned exit 0; every plan-flagged edge case has an explicit passing assertion.
- Fail: none
- Blocked: none

Tests are green. The /test gate is satisfied. Proceed to `/sync-docs` → `/cross-review` (optional) → `/pr` per `.claude/rules/post-implementation-pipeline.md`.
