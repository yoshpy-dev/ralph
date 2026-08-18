# Test report: overlay-scaffold-v2 Phase 3

- Date: 2026-08-18
- Plan: `docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md`
- Tester: `tester` subagent (Claude Code, Sonnet 5)
- Scope: behavioral tests via `./scripts/run-test.sh` (`RALPH_VERIFY_SCOPE=full`),
  on `feat/overlay-scaffold-v2-p3`, base `main`, HEAD before this run `6b617dc`.
  No static analysis, linting, or spec-compliance re-checking — that is
  `/verify`'s job and is already reported PASS in
  `docs/reports/verify-2026-08-18-overlay-scaffold-v2-p3.md`.
- Evidence: `docs/evidence/verify-2026-08-18-044102.log` (raw `run-test.sh`
  output from the confirmed exit-0 run below)

## Verdict: PASS

25/25 shell test files, all passing (zero `FAIL` lines anywhere in the
run), 8/8 Go packages `ok` (fresh, `-count=1 -cover`), zero test failures.
All 6 named marquee e2e tests and the legacy fail-closed trio individually
re-confirmed with `-run` + `-v` (not just as part of the batch). No test
weakening; no test-isolation bugs found this cycle.

## Test execution

| Suite / Command | Files/Packages | Passed | Failed | Duration |
| --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` | 25 shell test files | 25 | 0 | ~40s |
| `go test ./... -count=1 -cover` (fresh, no cache) | 8 Go packages with tests | 8 | 0 | ~49s |

`./scripts/run-test.sh` exit code: `0` (confirmed via a second isolated run
without `tee`, since the first invocation piped through `tee` and lost the
real exit code under `$PIPESTATUS` in this shell's non-interactive
invocation — re-ran directly to `>file 2>&1; echo $?` to get an honest
signal).

Shell suite: no `grep FAIL` hits other than `FAIL: 0` anywhere in the
40s+ log — every one of the 25 files that prints a per-file summary line
(`PASS: N / N` / `PASS: N` + `FAIL: 0`) reports zero failures, and the
files that print a plain `OK` marker instead of a numeric summary
(`test-gc-artifacts.sh`, `test-no-loop-references.sh`,
`test-check-mojibake.sh`, `test-check-skill-sync.sh`,
`test-hook-wiring.sh`, `test-ralph-dispatch.sh`, `test-insights-append.sh`,
`test-sync-skills.sh`) also report `OK` with zero failed cases inline
(`gc-artifacts tests: 11 passed, 0 failed, 11 total`, etc.). 25 `==> tests/`
headers and 25 trailing `    OK` markers in the log — every test file ran
and finished clean.

## Marquee e2e tests (individually confirmed, not just batch-passed)

Ran each with `go test ./internal/cli/... -run '<name>$' -v -count=1` to
rule out any batch-ordering or shared-state masking:

| Test | Result | Duration |
| --- | --- | --- |
| `TestRunUpgradeV2_FullPass` | PASS | 0.12s |
| `TestRunUpgradeV2_MixedScenario_AllClassesLandCorrectly` | PASS | 0.16s |
| `TestRunUpgradeV2_PartialFailure_ManifestNotAdvanced_ResumeCompletes` | PASS | 0.24s |
| `TestRunUpgradeV2_Idempotent_SecondRunIsNoOp` | PASS | 0.15s |
| `TestRunUpgradeV2_MissingAgentsCoreTemplate_LeavesAgentsMdUntouchedWithNote` | PASS | 0.11s |
| `TestRunUpgradeV2_PackRuleReadFailure_FullyPreserved` | PASS | 0.11s |
| `TestAddPack_LegacyManifest_FailsClosedZeroWrites` | PASS | 0.00s |
| `TestExecuteInit_LegacyManifest_ReInit_FailsClosedZeroWrites` | PASS | 0.00s |
| `TestRunUpgradeIOWithOptions_LegacyManifest_FailsClosedZeroWrites` | PASS | 0.00s |

All 9 pass. The first 6 exercise the v2 upgrade engine's core paths (AC-1,
AC-5, AC-6, AC-9, AC-11, AC-9 preserve-prefix); the trailing 3 confirm the
legacy (non-v2) manifest fail-closed guard across all three entry points
that could touch a legacy layout (`ralph pack add`, `ralph init` re-init,
`ralph upgrade`) — AC-7.

## Coverage

Fresh (`-count=1`, no cache) per-package coverage, compared against the
Phase-2 baselines recorded in prior cycles (`internal/cli` 79.0%,
`internal/scaffold` 78.9%, `internal/upgrade` 90.0%):

| Package | Phase-2 baseline | Phase-3 (this run) | Delta |
| --- | --- | --- | --- |
| `internal/cli` | 79.0% | 75.8% | -3.2pp |
| `internal/scaffold` | 78.9% | 75.7% | -3.2pp |
| `internal/upgrade` | 90.0% | 91.5% | +1.5pp |
| `internal/config` | (unaffected by this plan) | 94.2% | — |
| `internal/insights` | (unaffected) | 86.1% | — |
| `internal/org` | (unaffected) | 89.1% | — |
| `internal/org/driver` | (unaffected) | 92.0% | — |
| `internal/org/protocol` | (unaffected) | 97.9% | — |

### Explaining the `internal/cli` and `internal/scaffold` deltas

This plan is the largest single-cycle rewrite in the series to date on
these two packages: `git diff --shortstat main...HEAD -- internal/` shows
**3,144 insertions, 4,176 deletions** across `internal/cli`,
`internal/scaffold`, and `internal/upgrade` (net -1,032 lines), matching
the plan's stated "~4,100 lines removed" legacy-deletion scope
(`internal/upgrade/diff.go` + `diff_test.go`, `merge.go` + `merge_test.go`,
`internal/scaffold/baseline.go` + `baseline_test.go`, ~840 lines cut from
`internal/cli/upgrade.go`, plus a 699-line new `upgrade_v2.go` and a
1,356-line new `upgrade_v2_test.go`). This is not a like-for-like
comparison against Phase 2's coverage snapshot — it's a different set of
statements.

The `internal/cli` drop traces to specific, low-risk gaps rather than a
broad regression:

- `internal/cli/upgrade.go`: `writeDiffOutput`, `shouldUsePager`,
  `writeThroughPager` (0% — terminal/pager I/O helpers behind `--diff`,
  same class of gap as before: interactive-terminal code paths are not
  exercised by unit tests in this repo, consistent with prior cycles'
  documented blind spots) and `runUpgradeWithOptions` (0% — a thin wrapper
  kept for a call site outside the v2 path; its logic is exercised via
  `runUpgradeIOWithOptions`, which is 68.4% covered directly).
- `internal/cli/upgrade_v2.go` (all-new, 699 lines): several helpers sit in
  the 57–88% range — `readFinalDiskContent` (57.1%), `updateOneBlockV2`
  (57.7%), `renderUpgradeV2Preview` (58.5%, `--dry-run` display
  formatting), `writeFileV2` (75.0%), `sortedDriftV2` (80.0%),
  `buildDesiredStateV2` (82.0%), `runUpgradeV2` (82.7%),
  `applyBlockUpdatesV2` (85.2%), `rebuildManifestV2` (88.0%). These are
  branch-heavy functions (multiple owner classes, drift/advisory/seed
  permutations) where the 9 marquee e2e tests plus the broader
  `upgrade_v2_test.go` suite cover the primary paths and the plan's stated
  AC-1/3/4/4b/4c/5/6/9/11/12 edge cases, but not every permutation of every
  branch — expected for a first-cycle land of this much new branching
  logic, not a coverage regression against removed code.
- `internal/cli/pack.go`: `newPackListCmd` (12.5%) and `newPackAddCmd`
  (50.0%) are cobra command constructors (flag wiring only; the actual
  `addPack` logic they call is 86.2% covered) — pre-existing shape, not new
  to this cycle.

`internal/scaffold`'s drop is smaller in absolute terms and traces mostly
to `render.go` (`FilePerm` 60.0%, `HashFile` 77.8%, `RenderFS` 82.4%) and
`manifest.go` (`Write` 80.0%, `ReadManifest` 88.9%) — pre-existing
partial-coverage functions, not newly introduced by this plan's
`baseline.go` removal or the `SetLayoutV2`/`SetFileOwned`/`SetOwner`
changes. Confirmed directly: every ownership-related `manifest.go`
function this plan touches or added (`NewManifest`, `SetFile`,
`SetLayoutV2`, `SetFileOwned`, `SetFileFork`, `SetOwner`,
`validateManagedOwner`, `IsLegacyOwner`) is **100.0% covered** in this
run — the deletion of `baseline.go`/`baseline_test.go` (matched pairs,
both removed together) did not leave a coverage hole in the ownership API
surface this plan is responsible for. `embed.go`'s `BaseFS`/`PackFS`/
`AvailablePacks` remain at the same pre-existing 0% (go:embed accessor
functions, not meaningfully unit-testable) documented in prior cycles.

`internal/upgrade` rose (90.0% → 91.5%) because the deleted `diff.go`/
`merge.go` (494 lines of production code, previously well-tested) is gone
entirely, while the new `snapshot.go` (settings-snapshot read/fallback,
128 lines counting its test) and the expanded `replaceplan.go`
(desired-state input, preserve-prefix machinery) are both thoroughly
covered by `replaceplan_test.go` (net +229 lines) and `snapshot_test.go`
(new, 87 lines).

### Net assessment

No coverage regression in tested logic that matters to this plan's
acceptance criteria. All AC-tagged behavior (AC-1 through AC-12) has
dedicated test coverage in `upgrade_v2_test.go`, `replaceplan_test.go`,
and `snapshot_test.go`, individually confirmed above for the 6 marquee
scenarios plus the legacy fail-closed trio. The uncovered lines are
concentrated in interactive-terminal/pager helpers (pre-existing blind
spot) and edge branches of newly-added, already-tested functions — not
gaps in the core replace/block/settings/manifest-barrier flow this phase
exists to deliver.

## Test gaps

- `writeDiffOutput` / `shouldUsePager` / `writeThroughPager`
  (`internal/cli/upgrade.go`) remain untested — same class of
  interactive-terminal gap noted in prior cycles for the legacy `--diff`
  pager path; unchanged behavior carried forward, not a new regression.
- `runUpgradeV2`'s advisory-diff and drift-report branches
  (`renderUpgradeV2Preview`, `sortedDriftV2`, `readFinalDiskContent`) have
  lower per-branch coverage than the core apply path; every AC this plan
  claims is exercised, but exhaustive permutation coverage of every
  drift/advisory/seed combination is a reasonable follow-up rather than a
  blocker.
- `newPackListCmd` / `newPackAddCmd` cobra wiring remains thinly covered;
  pre-existing, not introduced by this plan.

## Cycle 1 verdict

- Pass: yes — 25/25 shell files clean, 8/8 Go packages `ok`, 9/9 named
  marquee tests individually re-confirmed
- Fail: none
- Blocked: none
- This is pipeline cycle 1 for overlay-scaffold-v2 Phase 3.
