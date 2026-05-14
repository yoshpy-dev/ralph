# Test report: doc-drift-audit

- Date: 2026-05-14
- Plan: N/A (operator-requested sync-docs audit; no active plan in `docs/plans/active/`)
- Tester: Codex inline tester
- Scope: behavioral regression tests for the harness after docs/rules/skill drift fixes
- Evidence: `docs/evidence/verify-2026-05-14-111200.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | harness test wrapper | all reported tests passed | 0 | language packs skipped by docs-only changed scope | ~20s |

Key suites included:

- `tests/test-check-mojibake.sh`: 11/11 PASS
- `tests/test-secret-scan.sh`: 6/6 PASS
- `tests/test-check-skill-sync.sh`: 7/7 PASS
- `tests/test-ralph-cli-driver.sh`: 48/48 PASS
- `tests/test-detect-languages-terraform.sh`: 8/8 PASS
- `tests/test-detect-changed-languages.sh`: 19/19 PASS
- `tests/test-run-verify-scope.sh`: 12/12 PASS
- `tests/test-terraform-pack-verify.sh`: 32/32 PASS
- `tests/test-terraform-rule-frontmatter.sh`: 9/9 PASS
- `tests/test-terraform-gitignore.sh`: 47/47 PASS
- `tests/test-verify-mode-split.sh`: 59/59 PASS
- `tests/test-self-review-scope.sh`: 96/96 PASS
- `tests/test-agent-phase-boundaries.sh`: 44/44 PASS
- `tests/test-branch-name.sh`: 29/29 PASS
- `tests/test-ralph-worktree.sh`: 17/17 PASS
- `tests/test-ensure-pr-ready.sh`: 7/7 PASS
- `tests/test-ensure-pr-title-prefix.sh`: 13/13 PASS

## Coverage

- Statement: not measured for shell-heavy harness tests.
- Branch: covered by focused shell regression suites for branch naming, PR readiness/title enforcement, verify/test split, language detection, Terraform pack behavior, and cross-review driver dispatch.
- Function: covered at script level through the harness test wrapper.
- Notes: changed-language scope classified this diff as docs-only, so language pack verifiers were not selected beyond local harness regression tests.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |

No failures.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Skill mirror drift detection | PASS | `tests/test-check-skill-sync.sh` 7/7 |
| Cross-review driver inversion and triage counting | PASS | `tests/test-ralph-cli-driver.sh` 48/48 |
| Changed-language docs-only scope | PASS | `tests/test-detect-changed-languages.sh` and `tests/test-run-verify-scope.sh` |
| PR ready/title enforcement | PASS | `tests/test-ensure-pr-ready.sh`, `tests/test-ensure-pr-title-prefix.sh` |

## Test gaps

- Remote GitHub Actions were not run locally.

## Verdict

- Pass: yes.
- Fail: none.
- Blocked: none.
