# Test report: verify-test-split

- Date: 2026-05-13
- Plan: `docs/plans/archive/2026-05-13-verify-test-split.md`
- Tester: Codex
- Scope: behavioral tests
- Evidence: `docs/evidence/verify-2026-05-13-141237.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `tests/test-verify-mode-split.sh` | 59 | 59 | 0 | 0 | <1s focused; also inside `run-test.sh` |
| `tests/test-self-review-scope.sh` | 96 | 96 | 0 | 0 | <1s focused; also inside `run-test.sh` |
| `tests/test-agent-phase-boundaries.sh` | 44 | 44 | 0 | 0 | <1s focused; also inside `run-test.sh` |
| `./scripts/run-test.sh` | aggregate | pass | 0 | 0 | ~16s |
| `./scripts/run-verify.sh` | aggregate all-mode regression | pass | 0 | 0 | ~19s; final evidence `docs/evidence/verify-2026-05-13-170321.log` |

## Coverage

- Statement: N/A
- Branch: N/A
- Function: N/A
- Notes: shell routing behavior is covered by command-dispatch assertions using
  fake language tools.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| Initial sandboxed `./scripts/run-test.sh` | Go test could not write to the user Go build cache. | Sandbox permission boundary, not a product failure. | Re-ran with escalated permissions; passed. |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `run-test.sh` re-runs Go static checks through `packs/languages/golang/verify.sh`. | fixed | `tests/test-verify-mode-split.sh` verifies Go test mode calls `go test ./...` and skips `gofmt`, `go vet`, and `staticcheck`. |
| Non-Go language verifiers ignore `HARNESS_VERIFY_MODE`. | fixed | Same regression suite covers TypeScript, Python, Rust, and Dart. |
| Self-review scope can drift into verification/test responsibilities. | fixed | `tests/test-self-review-scope.sh` checks self-review files for boundary language and banned wrapper calls. |
| Verifier/tester subagents can drift back to aggregate verification or cross-phase commands. | fixed | `tests/test-agent-phase-boundaries.sh` checks Claude/Codex verifier and tester definitions plus template mirrors. |

## Test gaps

- No real TypeScript/Python/Rust/Dart project fixtures are executed; the new
  tests intentionally use fake tools to make routing deterministic and cheap.

## Verdict

- Pass: yes
- Fail: 0
- Blocked: none
