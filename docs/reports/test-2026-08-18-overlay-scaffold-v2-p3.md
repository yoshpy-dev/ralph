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

## Cycle 2 (final — pipeline cycle 2/2)

- Date: 2026-08-18
- HEAD: `aa7fb9c` (branch `feat/overlay-scaffold-v2-p3`)
- Delta since cycle 1 (`62631aa..aa7fb9c`): `fc8e9a9` (cross-review
  containment fixes: symlinked-parent preflight in
  `internal/upgrade/replaceplan.go`'s `applyOps`, plus symlink-escape
  guards for `.claude/settings.json`, its snapshot, and the reports dir in
  `internal/cli/upgrade_v2.go` — 7 new tests), `78d9039`/`4b53880`
  (cycle-2 self-review LOW fixes, including 2 test renames), plus a verify
  cycle-2 artifact commit.
- Evidence: `docs/evidence/verify-2026-08-18-052329.log` (raw
  `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` output from the run
  below, exit 0)

### Verdict: PASS

25/25 shell test files, all `FAIL: 0` (17 numeric-summary suites all
report `FAIL: 0`, the remaining suites report a plain `OK`/`PASS: N`
marker with zero failed cases inline; every one of the 25
`==> tests/test-*.sh` headers has a matching clean finish). 8/8 Go
packages `ok` on a fresh `go test ./... -count=1 -cover` run (no cache).
All 7 new containment tests from `fc8e9a9` individually re-confirmed with
`-run '<name>$' -v -count=1` (not just batch-passed):

| Test | Package | Result | Duration |
| --- | --- | --- | --- |
| `TestApplyOps_ParentChainPreflightRejectsSymlinkedParent_ZeroWrites` | `internal/upgrade` | PASS | 0.00s |
| `TestApplyOps_ParentChainPreflightRejectsDeeplyNestedSymlinkedParent` | `internal/upgrade` | PASS | 0.00s |
| `TestApplyOps_ParentChainPreflightAllowsRealNestedDirectories` | `internal/upgrade` | PASS | 0.00s |
| `TestApplyOps_ParentChainPreflightAllowsMissingIntermediateDirectories` | `internal/upgrade` | PASS | 0.00s |
| `TestRunUpgradeV2_SymlinkedSettingsJSON_RejectedNoEscape` | `internal/cli` | PASS | 0.10s |
| `TestRunUpgradeV2_SymlinkedSettingsSnapshot_RejectedAfterSettingsJSONWritten` | `internal/cli` | PASS | 0.07s |
| `TestRunUpgradeV2_SymlinkedReportsDir_RejectedNoEscape` | `internal/cli` | PASS | 0.07s |

No test weakening; no test-isolation issues found running these
individually vs. in the batch.

### Coverage deltas vs. cycle 1

Fresh (`-count=1`, no cache) per-package coverage:

| Package | Cycle 1 | Cycle 2 (this run) | Delta |
| --- | --- | --- | --- |
| `internal/cli` | 75.8% | 75.9% | +0.1pp |
| `internal/scaffold` | 75.7% | 75.7% | 0.0pp |
| `internal/upgrade` | 91.5% | 91.1% | -0.4pp |

`internal/upgrade`'s small drop despite adding 4 new tests
(`replaceplan_test.go` +125 lines) is consistent with the new production
code outpacing the lines it exercises in this delta:
`internal/upgrade/replaceplan.go`'s `applyOps` gained 72 lines of
parent-chain-walk preflight logic (multiple symlink/dangling/depth branch
permutations — the 4 new tests cover the reject/reject-deep/allow-real/
allow-missing primary paths, not every intermediate-depth permutation),
and `internal/upgrade/report.go` gained 20 lines for the new rejection
reporting path. `internal/cli`'s small rise reflects the 3 new symlink
guard tests in `upgrade_v2_test.go` (+151 lines) landing net-positive
against the 23 new guard-check lines in `upgrade_v2.go`. Neither delta
indicates an untested containment gap: all 7 new tests directly exercise
the fix's reject/allow boundary conditions (see table above), and the
pre-existing gap classes documented in the Cycle 1 report (pager I/O
helpers, `--dry-run` preview formatting, cobra command wiring) are
unchanged this cycle.

### Cycle 2 verdict

- Pass: yes — 25/25 shell files clean, 8/8 Go packages `ok`, all 7 new
  containment tests individually re-confirmed
- Fail: none
- Blocked: none
- This is pipeline cycle 2 (final, per `RALPH_STANDARD_MAX_PIPELINE_CYCLES`
  default cap of 2) for overlay-scaffold-v2 Phase 3.

## Cycle 3 (final — pipeline cycle 3/3, cap raised)

- Date: 2026-08-18
- HEAD: `739c9c1` (branch `feat/overlay-scaffold-v2-p3`)
- Delta since cycle 2 test (`e21b1b1..739c9c1`): `41ad745` (dry-run exception
  preview + no-op short-circuit made write-free, plus 2 test rewrites —
  `TestRunUpgradeV2_DryRunPreview_NamesExceptionFaceChanges` and a rewritten
  `TestRunUpgradeV2_Idempotent_SecondRunIsNoOp`), `6533227` (veto the no-op
  short-circuit on pending seed advisories and version-only bumps, plus 2 new
  tests — `TestRunUpgradeV2_SeedAdvisoryOnly_NotANoOp` and
  `TestRunUpgradeV2_ConvergedButVersionBumped_NotANoOp`), plus doc/bookkeeping
  commits (`a0b9363`, `4df2ce9`, `1bb8367`, `b80b5f9`).
- Evidence: `docs/evidence/verify-2026-08-18-061741.log` (raw
  `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` output from the run below,
  exit 0, produced by `run-verify.sh` itself)

### Verdict: PASS

25/25 shell test files clean (25 `==> tests/test-*.sh` headers, 25 matching
trailing `OK` markers, zero `FAIL` lines anywhere in the log other than
`FAIL: 0`). `./scripts/run-test.sh` exit code `0`. Fresh (`-count=1`, no
cache) `go test ./... -count=1 -cover`: 8/8 packages `ok`, zero failures.
Working tree confirmed clean (`git status --porcelain`) before and after
this run — no stray artifacts left behind by the individually re-run tests.

All 5 tests named in the assignment individually re-confirmed with
`go test ./internal/cli/... -run '<name>$' -v -count=1` (not just
batch-passed):

| Test | Result | Duration |
| --- | --- | --- |
| `TestRunUpgradeV2_DryRunPreview_NamesExceptionFaceChanges` | PASS | 0.08s |
| `TestRunUpgradeV2_Idempotent_SecondRunIsNoOp` (rewritten) | PASS | 0.12s |
| `TestRunUpgradeV2_MixedScenario_AllClassesLandCorrectly` (strengthened) | PASS | 0.13s |
| `TestRunUpgradeV2_SeedAdvisoryOnly_NotANoOp` | PASS | 0.12s |
| `TestRunUpgradeV2_ConvergedButVersionBumped_NotANoOp` | PASS | 0.13s |

Read the assertion bodies directly (not just the pass/fail line) to confirm
each test proves what the delta claims:

- `TestRunUpgradeV2_Idempotent_SecondRunIsNoOp` (`internal/cli/upgrade_v2_test.go:725-784`):
  snapshots the **whole target tree** before/after the second run via
  `snapshotTreeHashesExcluding` with no path exclusions (manifest.toml and
  `docs/reports/` included), asserts byte-for-byte manifest equality, and
  asserts the manifest's `ModTime` is unchanged (not just content-equal —
  rules out an identical rewrite). This is the "no exclusions" full-tree
  zero-write check the delta describes.
- `TestRunUpgradeV2_MixedScenario_AllClassesLandCorrectly`'s second-run
  section (`internal/cli/upgrade_v2_test.go:1503-1600`): confirms drift
  persists forever (the drifted path is never touched by `ApplyOps`) while
  every other class converges, and the second run still returns
  `ErrUpgradeDriftRemaining` (exit 3) *and* announces `"no-op"` in stdout,
  with the same full-tree zero-write and manifest-mtime-unchanged assertions
  as above. This is the "drift + no-op → exit 3, zero writes" case.
- `TestRunUpgradeV2_SeedAdvisoryOnly_NotANoOp` (`internal/cli/upgrade_v2_test.go:797-...`):
  asserts a run with a pending seed advisory does **not** announce `"no-op"`,
  that seed content is left untouched on disk (user-owned once created), that
  the upgrade report mentions the pending advisory, and that the manifest's
  `TemplateHash` advances to clear it — then a regression check that the
  *next* run at the same version is a true no-op.

  **Cycle 4 correction:** this table entry described the cycle-3 version of
  the test. `7f1c531` reworked it (see the Cycle 4 section below) to pin the
  version deliberately at `"1.0.0-test"` instead of letting it bump, so the
  seed-advisory veto is exercised in isolation from the version-bump veto.
- `TestRunUpgradeV2_ConvergedButVersionBumped_NotANoOp` (`internal/cli/upgrade_v2_test.go:853-883`):
  templates are byte-for-byte identical to the prior run (no drift, no
  block/seed changes) but `Version` advances; asserts the run does not
  announce `"no-op"` and that the manifest's `Meta.Version` is rewritten to
  the new version, then a regression check that a second run at the now-caught-up
  version is a true no-op.

No test weakening found. No test-isolation issues found running any of the
5 individually vs. in the batch.

### Coverage deltas vs. cycle 2

Fresh (`-count=1`, no cache) per-package coverage:

| Package | Cycle 2 | Cycle 3 (this run) | Delta |
| --- | --- | --- | --- |
| `internal/cli` | 75.9% | 76.4% | +0.5pp |
| `internal/scaffold` | 75.7% | 75.7% | 0.0pp |
| `internal/upgrade` | 91.1% | 91.1% | 0.0pp |

(The assignment's stated expectation was cli 75.9/scaffold 75.7/upgrade
91.1 — `internal/cli` actually came in 0.5pp higher than that reference
point, the other two matched exactly.) The `internal/cli` rise is
consistent with the delta's shape: `41ad745` and `6533227` together add 4
new tests exercising previously-thin branches of `runUpgradeV2`'s no-op
short-circuit and dry-run preview path (`renderUpgradeV2Preview`,
`buildDesiredStateV2`'s advisory/version-veto logic) without adding a
proportionally larger amount of new production branching — the opposite
shape of Phase 3's Cycle 1 land, where a large batch of new production code
landed with only its primary paths covered. `internal/scaffold` is
untouched by this delta (0.0pp, expected — no changes in scope this
cycle). `internal/upgrade` is also untouched (0.0pp) — the delta's changes
are confined to `internal/cli/upgrade_v2.go` and its test file, not
`internal/upgrade`.

### Cycle 3 verdict

- Pass: yes — 25/25 shell files clean, 8/8 Go packages `ok`, all 5 named
  tests individually re-confirmed with assertion-level review, working tree
  clean
- Fail: none
- Blocked: none
- This is pipeline cycle 3 (final; the default
  `RALPH_STANDARD_MAX_PIPELINE_CYCLES` cap of 2 was consciously raised per
  the assignment to re-validate AR#2 fixes) for overlay-scaffold-v2 Phase 3.

## Cycle 4 (final — pipeline cycle 4/4)

- Date: 2026-08-18
- HEAD: `4aba6f3` (branch `feat/overlay-scaffold-v2-p3`)
- Delta since cycle 3 test (`f45463a..4aba6f3`): `07239cf` (sync-docs c3:
  fix stale ralph-upgrade report/dry-run description), `1f594c7`/`ae295f8`
  (cycle-3 cross-review triage), `b3c6307` (fix: classify `.codex/`
  `AGENTS.override.md` as seed per the L3 overlay contract — was
  misclassified core by `ownerForScaffoldPath`'s catch-all, which would have
  made a user-edited override permanently show as unresolved drift; adds 2
  new e2e tests), `d3a9fa6` (cycle-4 self-review section), `7f1c531` (fix:
  address cycle-4 self-review findings, including reworking
  `TestRunUpgradeV2_SeedAdvisoryOnly_NotANoOp` to pin the version at
  `"1.0.0-test"` so the seed-advisory veto is exercised independently of the
  version-bump veto — C4-1), `4aba6f3` (verify c4).
- Evidence: `docs/evidence/verify-2026-08-18-065930.log` (raw
  `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` output from the run below,
  exit 0)

### Verdict: PASS

25/25 shell test files clean (25 `==> tests/test-*.sh` headers, 25 matching
trailing `OK` markers; the 17 suites that print a numeric summary all report
`FAIL: 0`, the remaining 8 print a plain `OK`/`PASS: N` marker with zero
failed cases inline — same split documented in prior cycles). `./scripts/run-test.sh`
exit code `0`. Fresh (`-count=1`, no cache, `go clean -testcache` run first)
`go test ./... -count=1 -cover`: 8/8 packages `ok`, zero failures. Working
tree confirmed clean (`git status --porcelain`) before and after this run.

All 3 tests named in the assignment individually re-confirmed with
`go test ./internal/cli/... -run '<name>$' -v -count=1` — `SeedAdvisoryOnly`
run 3x for stability per the assignment, the other two 1x each alongside it
(all 3 in the same `-run` invocation each time, so each run already
exercises all three together):

| Test | Run 1 | Run 2 | Run 3 |
| --- | --- | --- | --- |
| `TestRunUpgradeV2_CodexAgentsOverride_UserEdited_SeedNeverReplaced` | PASS (0.10s) | PASS (0.11s) | PASS (0.12s) |
| `TestRunUpgradeV2_CodexAgentsOverride_Unedited_TemplateChangeIsAdvisoryOnly` | PASS (0.10s) | PASS (0.11s) | PASS (0.12s) |
| `TestRunUpgradeV2_SeedAdvisoryOnly_NotANoOp` (reworked) | PASS (0.10s) | PASS (0.11s) | PASS (0.13s) |

No flakes across 3 runs. Read the assertion bodies directly
(`internal/cli/upgrade_v2_test.go:805-943`), not just the pass/fail line:

- `TestRunUpgradeV2_SeedAdvisoryOnly_NotANoOp` (`:805-854`): the C4-1 rework
  pins `Version = "1.0.0-test"` unchanged (no version bump) so that with the
  version conjunct of `isFullyConvergedV2`'s no-op gate already satisfied,
  the only thing that can still stop the short-circuit is the seed-advisory
  veto (C3-1) — the docstring at `:811-815` states this is a deliberate
  regression pin: with a bumped version, the version-bump veto (C4-1's
  sibling, `TestRunUpgradeV2_ConvergedButVersionBumped_NotANoOp`) would
  short-circuit first and the advisory veto would go untested. Asserts
  `"no-op"` absent from stdout, `docs/notes.md` untouched on disk (seed,
  user-owned once created), the upgrade report mentions the pending
  advisory, the manifest's `TemplateHash` advances to the new template hash
  while `DiskHash` stays at the old content — then a second run at the same
  version is confirmed a true no-op now that the advisory cleared.
- `TestRunUpgradeV2_CodexAgentsOverride_UserEdited_SeedNeverReplaced`
  (`:865-904`): pins the `b3c6307` fix. Writes a user-edited
  `.codex/AGENTS.override.md`, bumps the embedded template content and the
  binary `Version`, and asserts the user's edit survives byte-for-byte on
  disk, the upgrade report surfaces the pending template change as an
  advisory (not silently swallowed), and the manifest records
  `Owner=OwnerSeed` with `TemplateHash` advanced to the new template while
  `DiskHash` reflects the user's edit. Before the fix, the catch-all in
  `ownerForScaffoldPath` classified this path core, which would have flagged
  it as unresolved drift (exit 3) forever on any upstream template change,
  since a core-owned file with a user edit is drift by definition.
- `TestRunUpgradeV2_CodexAgentsOverride_Unedited_TemplateChangeIsAdvisoryOnly`
  (`:913-943`): the control case for the same fix — an *unedited* copy must
  also survive a template change untouched. Same fixture minus the
  user-edit step; asserts the on-disk content stays exactly the v1 template
  (not silently overwritten with v2), the report surfaces the advisory, and
  the manifest shows `Owner=OwnerSeed`, `TemplateHash` at the new template
  hash, `DiskHash` at the unchanged v1 content. Before the fix, a core
  classification would have let the replace planner silently overwrite this
  file with the new template instead of leaving it as an advisory-only seed.

No test weakening found. No test-isolation issues found running any of the
3 individually vs. in the batch vs. as part of the full 8-package run.

### Self-review C4-4 correction (stale line pointer)

The cycle-3 test report's marquee-test table cited
`TestRunUpgradeV2_ConvergedButVersionBumped_NotANoOp` at
`internal/cli/upgrade_v2_test.go:853-883`. `b3c6307` inserted 97 lines of new
tests ahead of it (the two `CodexAgentsOverride` tests above), shifting the
function to `:955-978` at this cycle's HEAD (confirmed via
`grep -n '^func TestRunUpgradeV2_ConvergedButVersionBumped'` →
`upgrade_v2_test.go:955`). The Cycle 3 section above (line ~320) is left
as-is for historical accuracy (it was correct at cycle-3 HEAD); this note
records the corrected pointer for anyone using it going forward:
`TestRunUpgradeV2_ConvergedButVersionBumped_NotANoOp` is at
`internal/cli/upgrade_v2_test.go:955-978` as of `4aba6f3`.

### Coverage deltas vs. cycle 3

Fresh (`-count=1`, no cache) per-package coverage:

| Package | Cycle 3 | Cycle 4 (this run) | Delta |
| --- | --- | --- | --- |
| `internal/cli` | 76.4% | 76.4% | 0.0pp |
| `internal/scaffold` | 75.7% | 75.7% | 0.0pp |
| `internal/upgrade` | 91.1% | 91.1% | 0.0pp |

Matches the assignment's stated expectation (cli 76.4/scaffold 75.7/upgrade
91.1) exactly. Flat coverage despite `b3c6307` adding ~130 lines of new
production logic (the `.codex/` seed-classification fix in
`ownerForScaffoldPath`) and 2 new e2e tests, plus `7f1c531`'s
`SeedAdvisoryOnly` rework, is consistent with the delta's shape: the new
classification branch and its 2 new tests land net-neutral on the package
percentage (small numerator and denominator increase in proportion), and
the `SeedAdvisoryOnly` rework changes which veto path is exercised without
changing line count meaningfully. `internal/scaffold` and `internal/upgrade`
are untouched by this delta (0.0pp expected — no changes in scope this
cycle; `b3c6307`/`7f1c531` are confined to `internal/cli/upgrade_v2.go` and
its test file).

### Cycle 4 verdict

- Pass: yes — 25/25 shell files clean, 8/8 Go packages `ok`, all 3 named
  tests individually re-confirmed 3x each with assertion-level review,
  working tree clean before and after
- Fail: none
- Blocked: none
- This is pipeline cycle 4 (final, per the assignment) for
  overlay-scaffold-v2 Phase 3. Tests pass; proceed to `/pr`.
