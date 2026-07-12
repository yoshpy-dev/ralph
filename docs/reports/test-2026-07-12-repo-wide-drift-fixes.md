# Test report: repo-wide-drift-fixes

- Date: 2026-07-12
- Plan: docs/plans/active/2026-07-12-repo-wide-drift-fixes.md
- Tester: tester subagent (claude-sonnet-4-6)
- Scope: changed (shell glob + go test ./...)
- Evidence: `docs/evidence/test-2026-07-12-repo-wide-drift-fixes.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped |
| --- | --- | --- | --- | --- |
| test-check-skill-sync.sh (A–K, incl. new H–K prompts parity) | 11 | 11 | 0 | 0 |
| test-agent-phase-boundaries.sh | 44 | 44 | 0 | 0 |
| test-branch-name.sh | 29 | 29 | 0 | 0 |
| test-check-mojibake.sh | 11 | 11 | 0 | 0 |
| test-detect-changed-languages.sh | 23 | 23 | 0 | 0 |
| test-detect-languages-terraform.sh | 8 | 8 | 0 | 0 |
| test-ensure-pr-ready.sh | 7 | 7 | 0 | 0 |
| test-ensure-pr-title-prefix.sh | 13 | 13 | 0 | 0 |
| test-language-pack-monorepo-roots.sh | 29 | 29 | 0 | 0 |
| test-model-routing.sh | 24 | 24 | 0 | 0 |
| test-ralph-cli-driver.sh | 93 | 93 | 0 | 0 |
| test-ralph-dry-run-side-effects.sh | 6 | 6 | 0 | 0 |
| test-ralph-orchestrator-branch-names.sh | 3 | 3 | 0 | 0 |
| test-ralph-orchestrator-pr-strategy.sh | 24 | 24 | 0 | 0 |
| test-ralph-run-options.sh | 5 | 5 | 0 | 0 |
| test-ralph-signals.sh | 12 | 12 | 0 | 0 |
| test-ralph-slice-skip-pr.sh | 4 | 4 | 0 | 0 |
| test-ralph-status.sh | 51 | 51 | 0 | 0 |
| test-ralph-worktree.sh | 17 | 17 | 0 | 0 |
| test-run-verify-scope.sh | 59 | 59 | 0 | 0 |
| test-secret-scan.sh | 12 | 12 | 0 | 0 |
| test-self-review-scope.sh | 96 | 96 | 0 | 0 |
| test-terraform-gitignore.sh | 47 | 47 | 0 | 0 |
| test-terraform-pack-verify.sh | 36 | 36 | 0 | 0 |
| test-terraform-rule-frontmatter.sh | 11 | 11 | 0 | 0 |
| test-verify-mode-split.sh | 59 | 59 | 0 | 0 |
| test-xreview-gate-regression.sh | 21 | 21 | 0 | 0 |
| test-xreview-prompt-render.sh | 54 | 54 | 0 | 0 |
| go test ./... (internal/action, cli, config, scaffold, state, ui, ui/panes, upgrade, watcher) | 9 pkgs | all ok | 0 | 0 |

**Total shell tests: 870 passed, 0 failed**
**Go packages: 9 ok, 0 failed**

## Coverage

- Statement: not instrumented (shell tests)
- Branch: not instrumented (shell tests)
- Function: not instrumented (shell tests)
- Go: cached from prior run (all 9 packages ok)
- Notes: Shell coverage measured by test-case scope per suite. No blind spots introduced by this PR (docs/comment-only changes + new prompts-parity gate in check-skill-sync.sh fully exercised by cases H–K).

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

No failures.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| check-skill-sync failed to detect missing prompts/ mirror | FIXED | Cases H (missing), I (differ), J (parity), K (real tree) all PASS |
| Real skill tree (cross-review + loop prompts) passes prompts parity gate | VERIFIED | Case K PASS |

## Test gaps

None introduced. The new prompts-parity gate (H–K) covers both directions (missing-in-mirror and extra-in-mirror) and the real skill tree.

## Verdict

- Pass: YES — 870 shell tests + 9 Go packages, 0 failures
- Fail: NO
- Blocked: NO — safe to proceed to /pr
