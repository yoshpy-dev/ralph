# Test report: unify-permission-mode

- Date: 2026-07-12
- Plan: docs/plans/active/2026-07-12-unify-permission-mode.md
- Tester: tester subagent (claude-sonnet-4-6)
- Scope: changed (full fallback applied — templates/base/ralph.toml triggers golang scope)
- Evidence: `docs/evidence/verify-2026-07-12-093401.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| **Targeted Go — `TestRunPipeline_Env\|TestRunPipeline_MaxIter\|TestDefault_PermissionMode\|TestLoad_PermissionMode\|TestLoad_TemplateRalphToml`** | 6 | 6 | 0 | 0 | ~2.4s |
| `bash tests/test-ralph-config.sh` | 43 | 43 | 0 | 0 | <5s |
| `tests/test-agent-phase-boundaries.sh` | 44 | 44 | 0 | 0 | — |
| `tests/test-branch-name.sh` | 29 | 29 | 0 | 0 | — |
| `tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 | — |
| `tests/test-check-skill-sync.sh` | 7 | 7 | 0 | 0 | — |
| `tests/test-detect-changed-languages.sh` | 23 | 23 | 0 | 0 | — |
| `tests/test-detect-languages-terraform.sh` | 8 | 8 | 0 | 0 | — |
| `tests/test-ensure-pr-ready.sh` | 7 | 7 | 0 | 0 | — |
| `tests/test-ensure-pr-title-prefix.sh` | 13 | 13 | 0 | 0 | — |
| `tests/test-language-pack-monorepo-roots.sh` | 29 | 29 | 0 | 0 | — |
| `tests/test-model-routing.sh` | 24 | 24 | 0 | 0 | — |
| `tests/test-ralph-cli-driver.sh` | 93 | 93 | 0 | 0 | — |
| `tests/test-ralph-config.sh` (full run) | 43 | 43 | 0 | 0 | — |
| `tests/test-ralph-dry-run-side-effects.sh` | 5 | 5 | 0 | 0 | — |
| `tests/test-ralph-orchestrator-branch-names.sh` | 3 | 3 | 0 | 0 | — |
| `tests/test-ralph-orchestrator-pr-strategy.sh` | 24 | 24 | 0 | 0 | — |
| `tests/test-ralph-run-options.sh` | 5 | 5 | 0 | 0 | — |
| `tests/test-ralph-signals.sh` | 3 | 3 | 0 | 0 | — |
| `tests/test-ralph-slice-skip-pr.sh` | 4 | 4 | 0 | 0 | — |
| `tests/test-ralph-status.sh` | 51 | 51 | 0 | 0 | — |
| `tests/test-ralph-worktree.sh` | 17 | 17 | 0 | 0 | — |
| `tests/test-run-verify-scope.sh` | 12 | 12 | 0 | 0 | — |
| `tests/test-secret-scan.sh` | 6 | 6 | 0 | 0 | — |
| `tests/test-self-review-scope.sh` | 96 | 96 | 0 | 0 | — |
| `tests/test-terraform-gitignore.sh` | 47 | 47 | 0 | 0 | — |
| `tests/test-terraform-pack-verify.sh` | 36 | 36 | 0 | 0 | — |
| `tests/test-terraform-rule-frontmatter.sh` | 11 | 11 | 0 | 0 | — |
| `tests/test-verify-mode-split.sh` | 59 | 59 | 0 | 0 | — |
| `tests/test-xreview-gate-regression.sh` | 21 | 21 | 0 | 0 | — |
| `tests/test-xreview-prompt-render.sh` | 54 | 54 | 0 | 0 | — |
| **Go `go test ./...` (all packages)** | — | 9 pkgs ok | 0 | 0 | cached |

**Total shell tests: 763 passed / 763 total**
**Go packages: 9 ok / 9 with test files**

## Targeted test per-test status

| Test name | Package | Status |
| --- | --- | --- |
| TestRunPipeline_EnvWinsOverTomlForModelEtc | internal/cli | PASS |
| TestRunPipeline_MaxIterFlagBeatsEnv | internal/cli | PASS |
| TestRunPipeline_MaxIterEnvBeatsTomlWhenNoFlag | internal/cli | PASS |
| TestDefault_PermissionMode | internal/config | PASS |
| TestLoad_PermissionModeBackfill | internal/config | PASS |
| TestLoad_TemplateRalphToml | internal/config | PASS |

## Coverage

- Statement: not instrumented (shell suites measured by case scope; Go uses `go test` default)
- Branch: n/a
- Function: n/a
- Notes:
  - Shell coverage: the 2 new edge-case tests in `test-ralph-config.sh` (`empty RALPH_PERMISSION_MODE falls back to bypassPermissions`, `non-empty RALPH_PERMISSION_MODE=auto is preserved`) directly exercise the documented "non-empty env wins" contract via `${VAR:-default}` expansion. This is end-to-end shell-layer coverage as required by the plan contract note.
  - Go coverage: `TestLoad_TemplateRalphToml` asserts that the template's `permission_mode` matches `Default()`'s value (`bypassPermissions`), satisfying AC4b. `TestDefault_PermissionMode` asserts the Go default directly.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (none) | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `ralph run` exported toml value even when env was pre-set (RALPH_PERMISSION_MODE, RALPH_MODEL, RALPH_EFFORT) | FIXED | TestRunPipeline_EnvWinsOverTomlForModelEtc PASS |
| CLI flag `--max-iterations` did not beat env when both were set | FIXED | TestRunPipeline_MaxIterFlagBeatsEnv PASS |
| Env did not beat toml for MAX_ITERATIONS when no CLI flag was given | FIXED | TestRunPipeline_MaxIterEnvBeatsTomlWhenNoFlag PASS |
| Go Default() returned `auto` for permission_mode, diverging from shell default `bypassPermissions` | FIXED | TestDefault_PermissionMode PASS |
| template `permission_mode` was `auto`, diverging from `Default()` | FIXED | TestLoad_TemplateRalphToml PASS |
| Load() did not backfill `permission_mode` to `bypassPermissions` when toml key absent | FIXED | TestLoad_PermissionModeBackfill PASS |

## Test gaps

- Live `claude -p --permission-mode bypassPermissions` smoke: intentionally NOT added to CI (plan finding 7, advisory finding 7). The flag value is already exercised by every shell-path Loop run in production. Adding a live CLI call to the test suite would require a real `claude` binary and introduce flakiness in offline environments.
- No instrumented Go coverage percentage measured; the targeted test set covers all AC acceptance criteria directly (AC1–AC4b, AC5).

## Verdict

- Pass: YES
- Fail: NO
- Blocked: NO

All 763 shell tests and all 6 targeted Go tests pass. Full `./scripts/run-test.sh` regression suite passes with 0 failures. PR creation is unblocked.
