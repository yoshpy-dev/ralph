# Test report: upgrade detect local edits

- Date: 2026-04-22
- Plan: docs/plans/active/2026-04-22-upgrade-detect-local-edits.md
- Branch: feat/upgrade-detect-local-edits
- Tester: tester subagent (Claude Code)
- Scope: `internal/upgrade`, `internal/cli`, `internal/scaffold` (behavioral tests for conflict detection, unified diff, Managed=false convergence, DI-able resolveConflict, disk-read fallback) + full Go suite regression
- Evidence: `docs/evidence/test-2026-04-22-upgrade-detect-local-edits.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `go test ./internal/upgrade/... -v -count=1` | 23 | 23 | 0 | 0 | 0.77s |
| `go test ./internal/cli/... -v -count=1` | 18 | 18 | 0 | 0 | 1.33s |
| `go test ./internal/scaffold/... -v -count=1` | 13 | 11 | 0 | 2 | 1.19s |
| `go test ./... -count=1` (full) | 10 pkgs | 10 | 0 | 2 skipped inside scaffold | ~3.7s longest pkg |
| `./scripts/run-test.sh` (mojibake + gofmt + staticcheck + go test) | — | pass | 0 | — | ~10s |

Full suite package roll-up (`go test ./... -count=1`, all `ok`):

- `internal/action` (3.68s)
- `internal/cli` (1.00s)
- `internal/config` (0.88s)
- `internal/scaffold` (1.41s)
- `internal/state` (1.24s)
- `internal/ui` (1.77s)
- `internal/ui/panes` (3.08s)
- `internal/upgrade` (2.29s)
- `internal/watcher` (3.72s)
- `./`, `cmd/ralph`, `cmd/ralph-tui` — no test files (expected)

## Coverage

Per-package coverage (go test `-cover`):

| Package | Coverage |
| --- | --- |
| `internal/upgrade` | 93.6% |
| `internal/cli` | 37.8% |
| `internal/scaffold` | 65.9% |

Per-function highlights for functions introduced or modified by the plan (from `go tool cover -func`):

| Function | Coverage | Notes |
| --- | --- | --- |
| `internal/upgrade/unified_diff.go:UnifiedDiff` | 100.0% | 10 cases: identical, empty, add-only, remove-only, replace, empty-to-nonempty, nonempty-to-empty, trailing-newline, context window, order stability |
| `internal/upgrade/diff.go:ComputeDiffsWithManifest` | 91.7% | Covers Conflict on local-edit-with-unchanged-template, Skip on Managed=false (with and without disk file) |
| `internal/cli/upgrade.go:runUpgrade` | 100.0% | entry point |
| `internal/cli/upgrade.go:runUpgradeIO` | 68.8% | Interactive stdin/stdout DI covered for overwrite/skip/diff/invalid-then-skip/next-run-silent/disk-read-failure |
| `internal/cli/upgrade.go:resolveConflict` | 81.8% | Unified-diff branch, repeated `d` handling, EOF fallback, and disk-read-failure hash-only fallback all exercised |
| `internal/scaffold/manifest.go:SetFileUnmanaged` | 0.0% in scaffold package, but exercised via `internal/upgrade/diff_test.go` and `internal/cli` interactive tests. See *Test gaps* below. |

Notes:

- Coverage for `internal/upgrade` is dominated by the new logic and unchanged-template paths; no significant dead code was introduced.
- `internal/cli` package-local coverage of 37.8% is consistent with the pre-existing ratio (CLI has many wiring paths that rely on integration behavior rather than package-internal coverage). The plan's new conflict-resolution paths are explicitly covered by the six new `TestRunUpgrade_*` tests.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | None | — | — |

No failing tests.

## Regression checks

Regression suites from the plan's test plan all pass:

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| `TestComputeDiffs_Skip_PreservesHash` | PASS | evidence log L9 |
| `TestComputeDiffs_HealsEmptyHash_WhenDiskMatchesTemplate` | PASS | evidence log L14 |
| `TestComputeDiffs_EmptyHashConflictsWhenDiskDiffers` | PASS | evidence log L16 |
| `TestComputeDiffs_AddBecomesConflictWhenDiskDiffers` | PASS | evidence log L18 |
| `TestComputeDiffsWithManifest_PackPrefixedSubset` | PASS | evidence log L12 |
| `TestComputeDiffs_AddStaysAddWhenDiskMatchesTemplate` | PASS | evidence log L26 |
| `TestRunUpgrade_HealsCorruptedManifest` | PASS | full suite |
| `TestRunUpgrade_DropsPacksRemovedFromTemplates` | PASS | full suite |
| `TestRunUpgrade_ReportsDeletedPackFileOnceThenDrops` | PASS | full suite |
| `TestRunUpgrade_SurvivesAvailablePacksFailure` | PASS | full suite |
| `TestRunUpgrade_SameVersionIsIdempotent` | PASS | full suite |

New plan-mandated tests added and passing:

| Test | Purpose | Status |
| --- | --- | --- |
| `TestComputeDiffs_LocalEditWithUnchangedTemplate` | Template unchanged + local edit → Conflict | PASS |
| `TestComputeDiffs_Unmanaged_IsSilentSkip` | Managed=false → Skip (no prompt, no auto-update) | PASS |
| `TestComputeDiffs_Unmanaged_SilentSkipWhenDiskMissing` | Managed=false + disk absent → Skip (graceful) | PASS |
| `TestUnifiedDiff_*` (10 variants) | LCS-based unified diff determinism | PASS |
| `TestRunUpgrade_ForceOverwritesLocalEdit` | `--force` regression | PASS |
| `TestRunUpgrade_InteractiveOverwrite_WritesManaged` | `o\n` → `{Hash: newHash, Managed: true}` | PASS |
| `TestRunUpgrade_InteractiveSkip_WritesUnmanaged` | `s\n` → `{Hash: diskHash, Managed: false}` | PASS |
| `TestRunUpgrade_InteractiveDiff_ShowsUnifiedDiff` | `d\ns\n` → `---`/`+++`/`-`/`+` lines in stdout | PASS |
| `TestRunUpgrade_InteractiveDiff_RepromptsOnInvalid` | Invalid input → reprompt without loop | PASS |
| `TestRunUpgrade_NextRunAfterSkip_IsSilent` | After skip convergence, second upgrade is silent | PASS |
| `TestRunUpgrade_DiskReadFailure_FallsBackToHash` | Disk read failure → warn + hash-only fallback, no abort | PASS |

Acceptance-criteria to test-case mapping (all plan criteria covered):

- AC1 unchanged-template + local-edit → Conflict: `TestComputeDiffs_LocalEditWithUnchangedTemplate`
- AC2 unchanged-template + no-local-edit → Skip: `TestComputeDiffs_Skip_PreservesHash`
- AC3 changed-template + local-edit → Conflict: pre-existing `TestComputeDiffs_Conflict`
- AC4 `[d]iff` unified output shape: `TestRunUpgrade_InteractiveDiff_ShowsUnifiedDiff` + `TestUnifiedDiff_*`
- AC5 `d` non-loop / EOF-safe: `TestRunUpgrade_InteractiveDiff_RepromptsOnInvalid`
- AC6 disk-read-failure fallback: `TestRunUpgrade_DiskReadFailure_FallsBackToHash`
- AC7 overwrite → Managed=true: `TestRunUpgrade_InteractiveOverwrite_WritesManaged`
- AC8 skip → Managed=false convergence: `TestRunUpgrade_InteractiveSkip_WritesUnmanaged`
- AC9 Managed=false silent-skip: `TestComputeDiffs_Unmanaged_IsSilentSkip` + `TestRunUpgrade_NextRunAfterSkip_IsSilent`
- AC10 `--force` overwrites local edit: `TestRunUpgrade_ForceOverwritesLocalEdit`
- AC11 `./scripts/run-verify.sh` + `go test ./...` green: captured above (run-test.sh passes, full suite passes)

Skipped tests (unchanged, pre-existing):

- `TestBaseFS_WithMockFS`, `TestAvailablePacks_WithMockFS` in `internal/scaffold` — skip with reason "EmbeddedFS not initialized (only available when built from cmd/ralph/)". These are environment-conditional tests that only run when the package is invoked from the binary entrypoint, and are not regressions.

## Test gaps

Minor and non-blocking:

1. `SetFileUnmanaged` shows 0.0% in the scaffold package's local coverage profile because no unit test inside `internal/scaffold/` calls it directly. The function is exercised end-to-end via `internal/upgrade/diff_test.go` and `internal/cli` interactive tests, so behavior is covered. A 5-line unit test inside `internal/scaffold/manifest_test.go` would lift the package-local number without adding behavioral coverage. Not required by the plan.
2. CRLF / BOM-tagged input for `UnifiedDiff` is treated as opaque byte input (plan's Windows risk section accepts this). No dedicated test exists for CRLF-in-hunk rendering; relied on byte-faithful passthrough.
3. `ComputeDiffsNoRemovals` remains at 0.0% — not touched by this plan, pre-existing.

Not blocking the PR.

## Flakes

None observed. All tests deterministic across the two full-suite runs executed (direct `go test ./...` and `./scripts/run-test.sh`). The one timing-sensitive suite tracked in tester memory (`test-ralph-signals.sh::test_loop_sigint`) is not in scope for this plan and was not run.

## Verdict

- Pass: yes
- Fail: no
- Blocked: no

Tests pass the gate. Safe to proceed to `/sync-docs` → `/codex-review` → `/pr`.
