# Test report: worktree-gc-exit-code

- Date: 2026-07-12
- Plan: docs/plans/active/2026-07-12-worktree-gc-exit-code/ (fix/worktree-gc-exit-code)
- Tester: tester subagent (claude-sonnet-4-6)
- Scope: changed-language (shell + golang full fallback; ralph-worktree.sh detected unclassified -> full)
- Evidence: `docs/evidence/verify-2026-07-12-131235.log`

## Test execution

### Target: test-ralph-worktree.sh (29 tests incl. 4 gc scenarios)

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `bash tests/test-ralph-worktree.sh` | 29 | 29 | 0 | 0 | ~2s |

All gc sub-scenarios:

| Sub-test | Description | Result |
| --- | --- | --- |
| gc (a) | no state files -> exits 0, "No stale ralph worktree state." | PASS |
| gc (b) | one stale entry (path missing) -> exits 0, lists STALE, no delete without --prune | PASS |
| gc (c) | gc --prune -> exits 0, stale file deleted; second gc after prune -> exits 0 + "No stale" | PASS |
| gc (d) | non-stale entry (path exists) -> exits 0, "No stale", file not deleted | PASS |

### Regression: ./scripts/run-test.sh (full suite)

| Suite | Tests | Passed | Failed | Notes |
| --- | --- | --- | --- | --- |
| test-agent-phase-boundaries.sh | 44 | 44 | 0 | |
| test-branch-name.sh | 29 | 29 | 0 | |
| test-check-mojibake.sh | 23 | 23 | 0 | |
| test-check-skill-sync.sh | 8 | 8 | 0 | |
| test-ensure-pr-ready.sh | 7 | 7 | 0 | |
| test-ensure-pr-title-prefix.sh | 13 | 13 | 0 | |
| test-ralph-cli-driver.sh | 29 | 29 | 0 | |
| test-model-routing.sh | 24 | 24 | 0 | |
| test-ralph-config.sh | 43 | 43 | 0 | |
| test-ralph-orchestrator.sh dry-run | 5 | 5 | 0 | |
| test-ralph-orchestrator.sh branch-name | 3 | 3 | 0 | |
| test-ralph-orchestrator.sh PR strategy | 24 | 24 | 0 | |
| test-ralph-run.sh | 5 | 5 | 0 | |
| test-ralph-signals.sh | 3 | 3 | 0 | |
| test-ralph-loop-skip-pr.sh | 4 | 4 | 0 | |
| test-ralph-status.sh | 51 | 51 | 0 | |
| test-ralph-worktree.sh | 29 | 29 | 0 | NEW suite |
| test-check-mojibake-hooks.sh | 12 | 12 | 0 | |
| test-self-review-scope.sh | 96 | 96 | 0 | |
| test-detect-languages-terraform.sh | 36 | 36 | 0 | |
| test-terraform-rule-frontmatter.sh | 11 | 11 | 0 | |
| test-verify-mode-split.sh | 59 | 59 | 0 | |
| test-xreview-gate-regression.sh | 21 | 21 | 0 | |
| test-xreview-prompt-render.sh | 54 | 54 | 0 | |
| Go: internal/action | ok | — | 0 | cached |
| Go: internal/cli | ok | — | 0 | cached |
| Go: internal/config | ok | — | 0 | cached |
| Go: internal/scaffold | ok | — | 0 | cached |
| Go: internal/state | ok | — | 0 | cached |
| Go: internal/ui | ok | — | 0 | cached |
| Go: internal/ui/panes | ok | — | 0 | cached |
| Go: internal/upgrade | ok | — | 0 | cached |
| Go: internal/watcher | ok | — | 0 | cached |

## Coverage

- Statement: shell scripts — no instrumented coverage tool; coverage measured by test case scope
- Branch: gc command: all 4 branches covered (empty/stale/stale+prune/live)
- Function: gc, state-root, state-path, default-branch, validate-clean-base, ensure, resume, current, cleanup — all exercised
- Notes: Go packages report cached results (no code changes in Go layer for this fix)

## Failure analysis

No failures. All 29 worktree tests and all regression suites passed.

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| gc with no stale entries exits non-zero (bug) | FIXED | gc (a) exits 0 confirmed |
| gc with stale entry exits non-zero (bug) | FIXED | gc (b) exits 0 confirmed |
| gc --prune exits non-zero after deletion (bug) | FIXED | gc (c) exits 0 confirmed |
| gc second run after prune exits non-zero (bug) | FIXED | gc (c) post-prune exits 0 confirmed |
| gc with live worktree exits non-zero (bug) | FIXED | gc (d) exits 0 confirmed |

## Test gaps

None identified for this feature. The gc command has 4 distinct scenarios and all are covered. The existing pre-gc suites (state-paths, ensure/resume, collision/dirty-base, cleanup) remain intact and green.

## Verdict

- Pass: 29/29 (target worktree suite) + all regression suites green
- Fail: 0
- Blocked: none — cleared to proceed to /pr
