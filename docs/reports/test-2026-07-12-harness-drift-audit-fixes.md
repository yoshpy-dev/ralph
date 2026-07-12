# Test report: harness-drift-audit-fixes

- Date: 2026-07-12
- Plan: docs/plans/active/ (docs-only diff — commits fa811a3 + report commits)
- Tester: tester subagent (claude-sonnet-4-6)
- Scope: full regression (changed scope expands to full because `templates/base/ralph.toml` is unclassified)
- Evidence: `docs/evidence/test-2026-07-12-harness-drift-audit-fixes.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| test-agent-phase-boundaries.sh | 44 | 44 | 0 | 0 | — |
| test-branch-name.sh | 29 | 29 | 0 | 0 | — |
| test-check-mojibake.sh | 11 | 11 | 0 | 0 | — |
| test-check-skill-sync.sh | 7 | 7 | 0 | 0 | — |
| test-detect-changed-languages.sh | 23 | 23 | 0 | 0 | — |
| test-detect-languages-terraform.sh | 8 | 8 | 0 | 0 | — |
| test-ensure-pr-ready.sh | 7 | 7 | 0 | 0 | — |
| test-ensure-pr-title-prefix.sh | 13 | 13 | 0 | 0 | — |
| test-language-pack-monorepo-roots.sh | 29 | 29 | 0 | 0 | — |
| test-model-routing.sh | 24 | 24 | 0 | 0 | — |
| test-ralph-cli-driver.sh | 93 | 93 | 0 | 0 | — |
| test-ralph-config.sh | 41 | 41 | 0 | 0 | — |
| test-ralph-dry-run-side-effects.sh | 5 | 5 | 0 | 0 | — |
| test-ralph-orchestrator-branch-names.sh | 3 | 3 | 0 | 0 | — |
| test-ralph-orchestrator-pr-strategy.sh | 24 | 24 | 0 | 0 | — |
| test-ralph-run-options.sh | 5 | 5 | 0 | 0 | — |
| test-ralph-signals.sh | 3 | 3 | 0 | 0 | — |
| test-ralph-slice-skip-pr.sh | 4 | 4 | 0 | 0 | — |
| test-ralph-status.sh | 51 | 51 | 0 | 0 | — |
| test-ralph-worktree.sh | 17 | 17 | 0 | 0 | — |
| test-run-verify-scope.sh | 12 | 12 | 0 | 0 | — |
| test-secret-scan.sh | 6 | 6 | 0 | 0 | — |
| test-self-review-scope.sh | 96 | 96 | 0 | 0 | — |
| test-terraform-gitignore.sh | 47 | 47 | 0 | 0 | — |
| test-terraform-pack-verify.sh | 36 | 36 | 0 | 0 | — |
| test-terraform-rule-frontmatter.sh | 11 | 11 | 0 | 0 | — |
| test-verify-mode-split.sh | 59 | 59 | 0 | 0 | — |
| test-xreview-gate-regression.sh | 21 | 21 | 0 | 0 | — |
| test-xreview-prompt-render.sh | 54 | 54 | 0 | 0 | — |
| go test ./internal/config/... | — | PASS | 0 | 0 | cached |
| go test ./internal/cli/... | — | PASS | 0 | 0 | 8.18s |
| **TOTAL** | **783** | **783** | **0** | **0** | — |

## Coverage

- Statement: Go packages — instrumented via `go test ./...` (cached); all packages pass
- Branch: Not instrumented for shell tests; coverage is by test case scope (783 shell assertions)
- Function: N/A (shell), Go packages all pass with existing test functions
- Notes:
  - `TestLoad_TemplateRalphToml` in `internal/config` passes — the ralph.toml comment added in fa811a3 does not break toml parsing (targeted delegation from verify gate)
  - `TestRunPipeline_ExportsPhaseModelEnv` and `TestRunPipeline_ExportsForceModelWhenSet` in `internal/cli` pass
  - docs-only diff: no new shell tests warranted; regression suite is the guard

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (none) | — | — | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| ralph.toml comment line breaks TOML parsing | PASS | TestLoad_TemplateRalphToml passes |
| Agent phase boundary text assertions (verifier/tester) | PASS | test-agent-phase-boundaries.sh: 44/44 |
| Self-review scope guard (96 assertions across 12 files) | PASS | test-self-review-scope.sh: 96/96 |
| Cross-review gate regression (ACTION_REQUIRED / clean / fix-all / RENDER_FAILED) | PASS | test-xreview-gate-regression.sh: 21/21 |
| Model routing receipts parseable via jq | PASS | test-model-routing.sh: 24/24 |

## Test gaps

None identified. The diff is docs-only (fa811a3 fixes six doc-drift findings in
`.claude/rules/`, `.claude/agents/`, `AGENTS.md`, `CLAUDE.md`, and plan templates).
No behavioral code changed. Full regression suite covers all touched areas indirectly
through text-pattern assertions in test-agent-phase-boundaries.sh and
test-self-review-scope.sh.

## Verdict

- Pass: 783 / 783 (shell) + all Go packages
- Fail: 0
- Blocked: none

PASS — no failures. Safe to proceed to /pr.
