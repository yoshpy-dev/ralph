# Test report: language-pack-monorepo-roots

- Date: 2026-05-21
- Plan: `docs/plans/archive/2026-05-21-language-pack-monorepo-roots.md`
- Tester: Codex
- Scope: Behavioral and regression tests for language pack root discovery, changed-scope narrowing, and aggregate test execution.
- Evidence: `docs/evidence/test-2026-05-21-language-pack-monorepo-roots.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `tests/test-language-pack-monorepo-roots.sh` | 21 | 21 | 0 | 0 | < 2s |
| `tests/test-detect-changed-languages.sh` | 23 | 23 | 0 | 0 | ~6s |
| `tests/test-run-verify-scope.sh` | 12 | 12 | 0 | 0 | < 2s |
| `tests/test-terraform-pack-verify.sh` | 32 | 32 | 0 | 0 | < 2s |
| `tests/test-verify-mode-split.sh` | 59 | 59 | 0 | 0 | < 2s |
| `./scripts/run-test.sh` | aggregate | PASS | 0 | 0 | ~55s |

## Coverage

- Statement: Not instrumented for shell scripts.
- Branch: Covered by shell regression cases for no markers, missing tools, mode split, nested roots, root filters, full fallback, and changed-scope narrowing.
- Function: Pack command dispatch and detector output contracts covered.
- Notes: Go package tests also pass through the aggregate `run-test.sh` execution.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| N/A | N/A | No failures in final run. | N/A |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Nested language project roots skipped because repo root lacks markers | Fixed | `tests/test-language-pack-monorepo-roots.sh` verifies commands run from nested roots. |
| Changed scope can only narrow by language | Improved | `tests/test-run-verify-scope.sh` verifies root handoff via `RALPH_VERIFY_PROJECT_ROOTS`. |
| `jvm` detection silently selects a missing pack | Fixed | `tests/test-detect-changed-languages.sh` verifies JVM markers fall back instead. |

## Test gaps

- No real TypeScript/Python/Rust/Dart/Terraform toolchains are invoked; fake tool tests intentionally isolate verifier dispatch. Existing aggregate Go tests cover real Go execution.

## Verdict

- Pass: Yes.
- Fail: None.
- Blocked: None.
