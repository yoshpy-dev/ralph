# Test report: org-debt-batch

- Date: 2026-08-03
- Plan: docs/plans/active/2026-08-03-org-debt-batch.md
- Tester: tester subagent
- Scope: `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` (shell + golang, full language scope) + `go test ./... -count=1` (fresh, no cache) + targeted isolate-reruns
- Evidence: `docs/evidence/test-2026-08-03-org-debt-batch.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `tests/test-agent-phase-boundaries.sh` | 44 | 44 | 0 | 0 | <1s |
| `tests/test-branch-name.sh` | 26 | 26 | 0 | 0 | <1s |
| `tests/test-check-mojibake.sh` | 11 | 11 | 0 | 0 | <1s |
| `tests/test-check-skill-sync.sh` | 13 | 13 | 0 | 0 | <1s |
| `tests/test-detect-changed-languages.sh` | 23 | 23 | 0 | 0 | <1s |
| `tests/test-detect-languages-terraform.sh` | 8 | 8 | 0 | 0 | <1s |
| `tests/test-ensure-pr-ready.sh` | 7 | 7 | 0 | 0 | <1s |
| `tests/test-ensure-pr-title-prefix.sh` | 13 | 13 | 0 | 0 | <1s |
| `tests/test-gc-artifacts.sh` | 11 | 11 | 0 | 0 | <1s |
| `tests/test-insights-append.sh` | 39 | 39 | 0 | 0 | <1s |
| `tests/test-language-pack-monorepo-roots.sh` | 29 | 29 | 0 | 0 | <1s |
| `tests/test-no-loop-references.sh` (AC-2/AC-5 guard, extended-pattern) | 1 | 1 | 0 | 0 | <1s |
| `tests/test-ralph-config.sh` | 15 | 15 | 0 | 0 | <1s |
| `tests/test-ralph-worktree.sh` | 29 | 29 | 0 | 0 | <1s |
| `tests/test-run-verify-scope.sh` | 12 | 12 | 0 | 0 | <1s |
| `tests/test-secret-scan.sh` | 6 | 6 | 0 | 0 | <1s |
| `tests/test-self-review-scope.sh` | 64 | 64 | 0 | 0 | <1s |
| `tests/test-sync-skills.sh` | 22 | 22 | 0 | 0 | <1s |
| `tests/test-terraform-gitignore.sh` | 47 | 47 | 0 | 0 | <1s |
| `tests/test-terraform-pack-verify.sh` | 36 | 36 | 0 | 0 | <1s |
| `tests/test-terraform-rule-frontmatter.sh` | 11 | 11 | 0 | 0 | <1s |
| `tests/test-verify-mode-split.sh` | 59 | 59 | 0 | 0 | <1s |
| `tests/test-xreview-helpers.sh` (AC-2 provenance-only exclusion) | 29 | 29 | 0 | 0 | <1s |
| `go test ./... -count=1` (cached, via run-test.sh full scope) | 8 pkgs | 8 | 0 | 0 | ~3s |
| `go test ./... -count=1` (fresh rerun, no cache) | 8 pkgs | 8 | 0 | 0 | ~45s |
| `go test ./internal/insights/... -v` (AC-3 focus) | 29 | 29 | 0 | 0 | <1s |
| `go test ./internal/cli/... -run Insights -v` (AC-3 focus) | 11 | 11 | 0 | 0 | <1s |
| `go test ./internal/org/... -v` watchdog subset (AC-4 focus, a/b/c/d) | 12 | 12 | 0 | 0 | <1s |
| `go test ./internal/cli/... -run TestRunUpgrade_RemovesRetiredLoopArtifactsFromManifest` (AC-5) | 1 | 1 | 0 | 0 | 0.18s |
| `go test ./internal/org/... -run TestWatch_TotalBudgetCutoff_ResumeAfterAllSeatsStop_ReAlertsOnNewCutoff` (AC-4d) | 1 | 1 | 0 | 0 | <0.01s |
| Isolate-rerun `TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess` (x3) | 3 | 3 | 0 | 0 | ~0.6s each |
| Isolate-rerun `TestRunWatcher_TimeoutIndependentOfSmallInterval` (x3) | 3 | 3 | 0 | 0 | ~1.2-1.5s each |

**Total shell tests: 23 files, 555 assertions, 0 failures.**
**Total Go: 8 packages, all `ok`, 0 failures (both cached-via-run-test.sh and fresh `-count=1` runs).**

## Coverage

- Statement (per package, `go test ./... -count=1 -cover`):
  - `internal/cli`: 77.5%
  - `internal/config`: 94.2%
  - `internal/insights`: 86.1%
  - `internal/org`: 89.1%
  - `internal/org/driver`: 92.0%
  - `internal/org/protocol`: 97.9%
  - `internal/scaffold`: 67.2%
  - `internal/upgrade`: 90.9%
  - overall: 83.5%
- Branch: not separately instrumented (Go `-cover` reports statement coverage only); no dedicated branch-coverage tool in this repo.
- Function: not separately instrumented.
- Notes: `internal/org` coverage rose from 88.3% (recorded in tester memory as of the prior org-runtime-retire-loop cycle) to 89.1% — consistent with the new AC-4 watchdog regression tests added by Slice 4 (`TestPruneEscalated_*`, `TestWatch_EscalateAlert_PrunesAt101stEscalation`, `TestWatch_TotalBudgetCutoff_ResumeAfterAllSeatsStop_ReAlertsOnNewCutoff`, `TestSendAlert_ProtocolValidationFailureFallback_IncludesSeatHeader`, `TestWatch_ScopeChange_TruncatesOversizedPorcelainOutput`, `TestTruncateScopeOutput_LineCountThenByteBudget`). `internal/cli` rose from 76.2% to 77.5%, consistent with the new AC-3 insights receipts tests and AC-4a status/watch source-display tests. `cmd/ralph` remains 0% (thin main wrapper, no dedicated unit tests — pre-existing, out of scope for this plan).

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| (none) | — | — | — |

No failures observed across any suite, in either the cached full-scope run-test.sh pass or the fresh `-count=1` rerun.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Total-budget cutoff not re-alerting after resume→respawn→re-cutoff (AC-4d, watchdog item d) | Fixed, regression-tested | `TestWatch_TotalBudgetCutoff_ResumeAfterAllSeatsStop_ReAlertsOnNewCutoff` PASS (asserts a fresh `FirstTS` on the re-raised alert, not a stale one carried over from the cleared condition) |
| `Escalated` map unbounded growth in `watch-status-<org>.json` (AC-4c, watchdog item c) | Fixed, regression-tested | `TestPruneEscalated_CapsAtMaxEntries_DropsOldestByEmbeddedTimestamp`, `TestPruneEscalated_AtExactlyMaxEntries_NoPruning`, `TestWatch_EscalateAlert_PrunesAt101stEscalation` all PASS |
| Unbounded `git status --porcelain` output in scope-change ALERT bodies; missing `SEAT:` header on protocol-validation fallback (AC-4b, watchdog item b) | Fixed, regression-tested | `TestWatch_ScopeChange_TruncatesOversizedPorcelainOutput`, `TestTruncateScopeOutput_LineCountThenByteBudget`, `TestSendAlert_ProtocolValidationFailureFallback_IncludesSeatHeader` all PASS |
| `ResolveOrgStateDir` source not surfaced in `ralph org watch` banner / `ralph status` (AC-4a, watchdog item a) | Fixed, regression-tested | `TestOrgWatch_Once_BannerShowsStateDirSource`, `TestStatusCmd_ShowsStateDirSource`, `TestStatusCmd_EmptyStateDirShowsSourceToo` all PASS |
| `ralph upgrade` remove path for retired Loop artifacts unverified by automated test (AC-5) | Fixed, regression-tested | `TestRunUpgrade_RemovesRetiredLoopArtifactsFromManifest` PASS — confirms manifest-entry removal per the plan's Slice 5 deviation note (disk files intentionally retained; only the manifest tracking entry is removed) |
| `test-no-loop-references.sh` guard pattern incomplete (8 of 9 model knobs) / `xreview-helpers.sh` file-wide exclusion too broad (AC-2) | Fixed, regression-tested | `tests/test-no-loop-references.sh` 1/1 PASS against the 9-token pattern; `tests/test-xreview-helpers.sh` 29/29 PASS with provenance-line-only exclusion in place |
| `TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`, `TestRunWatcher_TimeoutIndependentOfSmallInterval` (known flaky-under-adjacent-subprocess-contention, per tester memory) | Stable | Both isolate-reran 3x each, all PASS; no contention trigger present in this run's batching |

## Test gaps

- AC-3's output-contract text example (`ORG demo  SEAT lead  commanded=opus  honored: true=3 false=1 unknown=2  rate=75% (unknown 2 excluded)`) is fixed via `TestInsightsCmd_ReceiptsTextContractExample`; no gap.
- `internal/scaffold` remains the lowest-coverage package (67.2%) — pre-existing, unrelated to this plan's scope (no scaffold-side changes in this batch).
- `cmd/ralph` has 0% coverage (thin cobra-root wrapper) — pre-existing, tracked as a known blind spot, out of this plan's scope.
- No branch/function-level coverage instrumentation exists in this repo (Go `-cover` is statement-only); this is a standing tooling gap, not specific to this change.
- Manual evidence for the AC-5 "real ralph binary" smoke path (as opposed to the Go integration test) is captured in `docs/evidence/org-debt-batch-2026-08-03.txt` per the plan's evidence targets — not re-verified here since AC-5's automated-test requirement is independently satisfied by `TestRunUpgrade_RemovesRetiredLoopArtifactsFromManifest`.

## Verdict

- Pass: 23 shell suites (555 assertions) + 8 Go packages (fresh `-count=1` and cached full-scope run), including all plan-specified focus tests (AC-3 insights output contract, AC-4 all four watchdog regressions with FirstTS assertion, AC-5 upgrade-removal test, AC-2 guard suites at their expected counts)
- Fail: 0
- Blocked: none
