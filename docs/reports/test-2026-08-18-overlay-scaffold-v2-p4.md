# Test report: overlay-scaffold-v2 Phase 4 (legacy→v2 migration + owner-aware untracked classification)

- Date: 2026-08-18
- Plan: `docs/plans/active/2026-08-18-overlay-scaffold-v2-p4.md`
- Tester: `tester` subagent (Claude Code, Sonnet 5)
- Scope: behavioral tests via `./scripts/run-test.sh` (`RALPH_VERIFY_SCOPE=full`),
  on `feat/overlay-scaffold-v2-p4`, base `main` `1025d13`, HEAD `1dfba22`.
  No static analysis, linting, or spec-compliance re-checking — that is
  `/verify`'s job and is already reported PASS in
  `docs/reports/verify-2026-08-18-overlay-scaffold-v2-p4.md`.
- Evidence: `docs/evidence/verify-2026-08-18-093215.log` (raw `run-test.sh`
  output from the confirmed exit-0 run below)

## Verdict: PASS

599/599 shell tests across 25 files, all passing (zero `FAIL` lines
anywhere in the run), 8/8 Go packages `ok` (fresh, `-count=1 -cover`), zero
test failures. Every test named in the verifier's focus list — the
collision-matrix and rerun-stability tests pinning the HIGH-1/MEDIUM-2
fixes, plus the migration happy path, all-modified fixture, drift-sentinel
passthrough, symlinked delete parent, settings near-miss report,
`--yes`/poisoned-reader, and the known-gap seed-collision e2e — was
individually re-run 3x in isolation (`-count=3`) with zero flakes. No test
weakening found; no test-isolation bugs found this cycle.

## Test execution

| Suite / Command | Files/Packages | Passed | Failed | Duration |
| --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` | 25 shell test files | 25 | 0 | ~45s |
| `go test ./... -count=1 -cover` (fresh, no cache) | 8 Go packages with tests | 8 | 0 | ~50s |

`./scripts/run-test.sh` exit code: `0` (captured directly via
`>file 2>&1; echo $?`, not piped through `tee`, per
[[feedback_tee_masks_exit]]).

Per-file shell breakdown, reconstructed from the log by pairing each
`==> tests/<file>` header with its trailing summary line(s) (8 of the 25
files print a `PASS: N passed, 0 failed, N total` / plain `OK` form instead
of the `PASS: N / N` + `FAIL: 0` pair, so a naive single-pattern grep
undercounts — verified by cross-checking `grep -c "^==> tests/"` = 25
headers against 25 distinct files below, all summing to 599):

| File | Passed |
| --- | --- |
| test-agent-phase-boundaries.sh | 44 |
| test-branch-name.sh | 26 |
| test-check-mojibake.sh | 11 |
| test-check-skill-sync.sh | 13 |
| test-detect-changed-languages.sh | 23 |
| test-detect-languages-terraform.sh | 8 |
| test-ensure-pr-ready.sh | 7 |
| test-ensure-pr-title-prefix.sh | 13 |
| test-gc-artifacts.sh | 11 |
| test-hook-wiring.sh | 18 |
| test-insights-append.sh | 39 |
| test-language-pack-monorepo-roots.sh | 29 |
| test-no-loop-references.sh | 1 |
| test-ralph-config.sh | 15 |
| test-ralph-dispatch.sh | 26 |
| test-ralph-worktree.sh | 29 |
| test-run-verify-scope.sh | 12 |
| test-secret-scan.sh | 6 |
| test-self-review-scope.sh | 64 |
| test-sync-skills.sh | 22 |
| test-terraform-gitignore.sh | 47 |
| test-terraform-pack-verify.sh | 36 |
| test-terraform-rule-frontmatter.sh | 11 |
| test-verify-mode-split.sh | 59 |
| test-xreview-helpers.sh | 29 |
| **Total** | **599** |

This matches the previously established 599/599-across-25-files baseline
(Phase 2 cycle 4 onward) byte-for-byte — this plan's scope
(`internal/cli`, `internal/upgrade`, docs) touches no shell test file, so
no shell count drift was expected or found.

## Marquee focus tests (individually confirmed 3x each, not just batch-passed)

Per the verifier's report (`docs/reports/verify-2026-08-18-overlay-scaffold-v2-p4.md`,
"Minimal additional check for highest confidence gain"), these are exactly
the tests rewritten or added to pin the HIGH-1/MEDIUM-2 self-review fixes,
plus the rest of the plan's named test-plan focus areas. Ran each with
`go test ./internal/cli/... -run '^(<names>)$' -count=3 -v` to rule out any
batch-ordering or shared-state masking:

| Test | Covers | Result (3 runs) |
| --- | --- | --- |
| `TestRunMigrateLegacy_HappyPath_Yes` | migration happy path | 3/3 PASS |
| `TestRunMigrateLegacy_AllModifiedFixture_EveryTrackedFileForked` | all-modified fixture | 3/3 PASS |
| `TestClassifyMigration_CollisionMatrix` | collision matrix (a)/(b)/(c), classify-level | 3/3 PASS |
| `TestRunMigrateLegacy_CollisionMatrixA_HalfMigrated_RerunDeletesOldOnly` | collision (a), e2e — HIGH-1 regression pin | 3/3 PASS |
| `TestRunMigrateLegacy_CollisionMatrixB_DestMatchesTemplate_CoreAdopted` | collision (b), e2e | 3/3 PASS |
| `TestRunMigrateLegacy_Collision_ZeroWrites` | collision (c) divergent case, zero-write abort | 3/3 PASS |
| `TestRunMigrateLegacy_PartialFailure_ManifestNotAdvanced_ResumeCompletes` | rerun-after-partial-failure, unmodified | 3/3 PASS |
| `TestRunMigrateLegacy_RerunAfterPartialRelocation_ModifiedSource_DestAdoptsFork` | rerun-after-partial-failure, modified-source fork adoption — HIGH-1 pin | 3/3 PASS |
| `TestRunMigrateLegacy_ChainedDriftSentinel_SurvivesAsExitSentinel` | drift-sentinel passthrough (MEDIUM-1 fix) | 3/3 PASS |
| `TestRunMigrateLegacy_SymlinkedDeleteParent_ZeroWrites` | symlinked delete parent (MEDIUM-2 fix) | 3/3 PASS |
| `TestRunMigrateLegacy_SettingsPruneReport_NearMissNotClaimedAsRemoved` | settings near-miss report (MEDIUM-4 fix) | 3/3 PASS |
| `TestPruneLegacySettingsHooks_ArgumentVariant_LeftInPlaceAsNearMiss` | settings near-miss, unit level | 3/3 PASS |
| `TestRunMigrateLegacy_YesFlag_NeverReadsStdin` | `--yes` / poisoned-reader stdin guard | 3/3 PASS |
| `TestRunUpgradeV2_UntrackedSeedPathCollision_AdoptedNotDrift` | known-gap seed-collision e2e (AC-1) | 3/3 PASS |

39/39 runs pass for the batched group (13 tests × 3), plus 3/3 for the
seed-collision test run separately — 42 total invocations, 0 failures.

## Coverage

Fresh (`-count=1`, no cache) per-package coverage, compared against the
Phase-3 baselines recorded in `docs/reports/test-2026-08-18-overlay-scaffold-v2-p3.md`:

| Package | Phase-3 baseline | Phase-4 (this run) | Delta |
| --- | --- | --- | --- |
| `internal/cli` | 75.8% | 78.5% | +2.7pp |
| `internal/upgrade` | 91.5% | 91.2% | -0.3pp |
| `internal/scaffold` | 75.7% | 75.7% (unaffected by this plan) | — |
| `internal/config` | 94.2% | 94.2% (unaffected) | — |
| `internal/insights` | 86.1% | 86.1% (unaffected) | — |
| `internal/org` | 89.1% | 89.1% (unaffected) | — |
| `internal/org/driver` | 92.0% | 92.0% (unaffected) | — |
| `internal/org/protocol` | 97.9% | 97.9% (unaffected) | — |

`internal/cli` rose despite the plan adding a large new surface
(`internal/cli/migrate.go`, ~68KB / ~1,600 statement-lines, plus
`migrate_test.go`, ~98KB / 38 test functions) because the new migration
tests exercise their own code heavily and the surface is materially more
covered than the package average. `internal/upgrade`'s small drop is
consistent with adding the `OwnerForPath` resolver parameter and the
`classifyUntracked` seed branch — both are well covered individually (see
below) — diluted slightly by the package's pre-existing partial-coverage
pager/diff-output helpers noted in the Phase-3 report, which this plan did
not touch.

### `internal/cli/migrate.go` per-function coverage (new ~2,000-line migrate surface)

Isolated run: `go test ./internal/cli/... -run 'Migrate|Migration' -count=1
-coverprofile=... -coverpkg=./internal/cli/...`. Every one of the 36
functions in `migrate.go` has coverage — no 0%-covered function on the new
surface:

| Coverage band | Functions |
| --- | --- |
| 100.0% | `classifyLegacyEntryState`, `classifyLegacyPath`, `classifyClaudeMD`, `classifyBlockFace`, `classifySettingsFace`, `prunedLegacyHookCommands`, `relocationOutcome`, `RenderMigrationPreview`, `entriesForKind`, `hasBaselineDir`, `sortedStringSet`, `migrationReportRelPath`, `blockFaceUntouchedEntries` (13 functions) |
| 85–95% | `ClassifyMigration` 88.6%, `classifyRerunRelocatedDestination` 87.5%, `classifyUnmodifiedGeneric` 86.7%, `classifyForkCandidate` 90.5%, `buildMigratedManifest` 87.3%, `renderMigrationReport` 92.7%, `readDiskFileForMigration` 85.7%, `confirmMigration` 87.5% |
| 70–85% | `relocatedRulePath` 83.3%, `cleanMigrationPath` 75.0%, `runMigrateLegacy` 78.7%, `checkGitCleanForMigration` 72.2%, `validateMigrationOp` 80.0%, `validateMigrationLeaf` 71.4%, `validateMigrationWriteTarget` 80.0%, `writeMigrationFile` 71.4%, `removeMigrationPath` 71.4%, `removeMigrationDir` 75.0%, `pruneLegacySettingsHooks` 79.3%, `legacyHookNearMiss` 75.0% |
| <70% | `validateMigrationDirOp` 57.1% (an internal branch of the already-hoisted `validateMigrationOp` symlink guard covered by the MEDIUM-2 fix; the guard itself is exercised by the two symlinked-parent tests above, but not every internal early-return branch), `executeMigrationEntries` 68.8%, `relocateMigrationFile` 62.5% (both branch-heavy dispatch functions where the marquee e2e tests hit the primary op kinds but not every kind × failure-mode permutation) |

Every function pinning an AC (owner-aware classification, the two collision
matrices, the rerun-stability pre-check, the settings prune/near-miss
split, path-safety guards) sits at 71%+ individually, and the specific
lines the self-review flagged as defect-pinning (HIGH-1's
`classifyRerunRelocatedDestination`, MEDIUM-2's hoisted
`ValidateRealParentChain` call inside `validateMigrationOp`) are directly
exercised by the 3x-confirmed marquee tests above.

### `internal/upgrade/replaceplan.go` — `classifyUntracked` (AC-1/AC-2)

`classifyUntracked` (the owner-aware untracked-path classifier at the
center of AC-1/AC-2): **92.9%** covered in a fresh, isolated
`internal/upgrade` run. The seed-owned collision branch this plan added
(disk-existing seed path → advisory, not drift, no write) is exercised
directly by `TestRunUpgradeV2_UntrackedSeedPathCollision_AdoptedNotDrift`
(3/3 PASS above) plus the `replaceplan_test.go` nil-resolver and non-seed
cases the verifier traced for AC-2's backward-compat requirement.

## Known gaps / not independently re-verified

- `validateMigrationDirOp`, `executeMigrationEntries`, `relocateMigrationFile`
  sit at 57–69% coverage — every AC-pinning path through them is tested
  (confirmed above), but not every kind × failure-mode permutation. Same
  class of gap the Phase-3 report documented for `upgrade_v2.go`'s
  branch-heavy helpers on first-cycle land of new branching logic; not a
  regression, but worth a follow-up edge-case pass if this code sees a
  Phase-5 revision.
- Interactive terminal/pager helpers (`writeDiffOutput`, `shouldUsePager`,
  `writeThroughPager` in `upgrade.go`) remain at the pre-existing 0%
  documented in every prior cycle's test report — unaffected by this
  plan, not exercised by unit tests in this repo.
- `TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess` and
  `TestRunWatcher_TimeoutIndependentOfSmallInterval` are the two tests with
  a documented history of transient subprocess-contention flakiness
  (tester agent memory); both passed cleanly in the full-suite run this cycle
  and were not independently re-isolated since neither is in this plan's
  scope and neither showed any sign of flaking here.
- Runtime confirmation against a hand-built real legacy fixture project
  (as opposed to the in-repo Go test fixtures) was not performed — the
  in-repo `migrate_test.go` fixtures reconstruct v1-era manifest/disk
  shapes directly, which the verify report already noted as the boundary
  of what was independently re-executed.

## Test plan cross-check (plan §Test plan)

- Unit tests (classifier table-driven, owner-aware classification, preview
  rendering) — present and passing (`TestClassifyMigration_*`,
  `TestClassifyLegacyEntryState`, `TestRenderMigrationPreview_*`).
- Integration tests (legacy fixture → migration → AC verification e2e,
  all-modified fixture, pack-carrying fixture) — present and passing
  (`TestRunMigrateLegacy_HappyPath_Yes`, `TestRunMigrateLegacy_AllModifiedFixture_*`,
  `TestClassifyMigration_PacksSortedAndCarried`).
- Regression tests (full suite green, resolver-nil compat) — 599/599 shell
  + 8/8 Go packages, `replaceplan_test.go` nil-resolver cases pass.
- Edge cases (dirty git / non-git, `--yes`/`--dry-run`, empty-hash heal
  targets, both old+new path existing, symlinks, post-migration no-op) —
  all present: `TestRunMigrateLegacy_DirtyGitTree_ZeroWrites`,
  `TestClassifyMigration_EmptyHashHealTarget_*` (2 variants),
  `TestRunMigrateLegacy_SymlinkedRelocationDestParent_ZeroWrites` /
  `_SymlinkedDeleteParent_ZeroWrites`, `TestClassifyMigration_RerunStability_*`
  (2 variants).

No gap found between the plan's stated test plan and what actually shipped
in `migrate_test.go` / `replaceplan_test.go`.

## What remains unverified

- Static analysis and spec compliance — `/verify`'s job, already PASS
  (`docs/reports/verify-2026-08-18-overlay-scaffold-v2-p4.md`).
- Documentation sync — `/sync-docs`'s job, not yet run for this plan.
- Cross-model second opinion — `/cross-review`'s job, not yet run.

## Cycle 2

- Date: 2026-08-18
- Tester: `tester` subagent (Claude Code, Sonnet 5)
- Scope: behavioral tests via `./scripts/run-test.sh` (default changed-language
  scope — the cycle-2 delta touches only `internal/cli/*.go` and
  `internal/upgrade/*.go`, plus docs), on `feat/overlay-scaffold-v2-p4`, HEAD
  `0cc50e5`. This is pipeline cycle 2/2 per
  `RALPH_STANDARD_MAX_PIPELINE_CYCLES` (default 2); cycle-1 test already
  passed at `b1babe7` (599/599 shell + full `go test`, recorded above). No
  static analysis re-run — that is `/verify`'s job, already PASS in
  `docs/reports/verify-2026-08-18-overlay-scaffold-v2-p4.md`.
- Delta since cycle-1 test: `c51497e` (cross-review AR#1/AR#2/AR#3 fixes, +3
  new tests), `aabca2d` (self-review C2-1..C2-4 fixes; behavioral change —
  `settings.json` is no longer rewritten when a prune removes nothing; +1 new
  test), plus report-only commits `13adf85` / `0cc50e5`.

### Verdict: PASS

599/599 shell tests across 25 files (byte-identical to cycle 1 — the delta
touches no shell test file), 8/8 Go packages `ok` (fresh, `-count=1
-cover`, no cache), zero failures anywhere. All four regression tests named
in the assignment exist, ran, and passed 3/3 each in isolation with zero
flakes. No test weakening found.

### Test execution

| Suite / Command | Scope | Passed | Failed | Duration |
| --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (default changed-language scope) | 25 shell test files + golang pack | 599 shell + 8/8 Go pkgs (cached) | 0 | ~40s |
| `go test ./... -count=1 -cover` (fresh, no cache) | 8 Go packages with tests | 8 | 0 | ~65s |

`./scripts/run-test.sh` exit code: `0` (captured directly via
`>file 2>&1; echo $?`, not piped through `tee`, per
[[feedback_tee_masks_exit]]). `run-test.sh` selected `golang` as the sole
changed language pack (`==> Language packs selected: golang`), and still ran
the full 25-file shell suite ahead of the language-scoped dispatch, matching
cycle 1's file/test counts exactly (`grep -c '^==> tests/'` = 25; no `FAIL`
line found anywhere in the log).

Fresh `go test ./... -count=1 -cover` (bypassing the `run-test.sh` build
cache to rule out stale results from before the cycle-2 commits):

| Package | Result | Coverage |
| --- | --- | --- |
| `internal/cli` | ok | 78.6% |
| `internal/config` | ok | 94.2% |
| `internal/insights` | ok | 86.1% |
| `internal/org` | ok | 89.1% |
| `internal/org/driver` | ok | 92.0% |
| `internal/org/protocol` | ok | 97.9% |
| `internal/scaffold` | ok | 75.7% |
| `internal/upgrade` | ok | 91.2% |

`internal/cli` coverage rose from cycle 1's 78.5% to 78.6% (the four new
tests added their own well-covered lines). All other packages are
byte-identical to cycle 1's coverage baseline, as expected — this cycle's
delta touches only `internal/cli` and `internal/upgrade` source, and
`internal/upgrade` itself is unchanged at 91.2%.

`internal/cli` package total: 195 test functions, 0 `FAIL` (`go test
./internal/cli/... -v -count=1 | grep -c '^--- PASS'` = 195, `grep -c
'^--- FAIL'` = 0).

### Named regression tests (per assignment)

Ran with `go test ./internal/cli/... -run '^(<names>)$' -count=3 -v` to rule
out any batch-ordering or shared-state masking, isolated from the rest of
the suite:

| Test | Covers | Result (3 runs) |
| --- | --- | --- |
| `TestRunMigrateLegacy_SymlinkedAdoptForkDestParent_ZeroWrites` | AR#1 — symlinked adopt-fork destination parent validation | 3/3 PASS |
| `TestRunMigrateLegacy_UntrackedSeedCollision_AdvisorySurvivesChainedUpgrade` | AR#2 — seed-advisory bypass survives a chained `upgrade` after `migrate` | 3/3 PASS |
| `TestRunMigrateLegacy_AdoptedForkDiff_IncludedInReport` | AR#3 — adopted-fork diff included in the migration report | 3/3 PASS |
| `TestExecuteMigrationEntries_NearMissOnlyPrune_SettingsFileUntouched` | C2-2 — `settings.json` left byte-identical when a prune removes nothing | 3/3 PASS |

12/12 invocations pass (4 tests × 3 runs), 0 failures, 0 flakes.

### C2-2 behavioral-change verification

`aabca2d` changes production behavior: `executeMigrationEntries` now skips
rewriting `.claude/settings.json` when `pruneLegacySettingsHooks` reports it
removed nothing (a near-miss-only prune), leaving the user's file bytes
untouched instead of rewriting it to an equivalent-but-not-identical form.
`TestExecuteMigrationEntries_NearMissOnlyPrune_SettingsFileUntouched`
directly pins this by asserting byte-for-byte file identity (not just JSON
equivalence) before/after migration when only a near-miss is present — this
is the correct assertion shape for the claimed behavior and it passed 3/3.

### Known-flaky tests re-checked in isolation

Per tester agent memory, `TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`
(`internal/cli`) and `TestRunWatcher_TimeoutIndependentOfSmallInterval`
(`internal/org`) have a documented history of transient subprocess-contention
flakiness. Neither is in this cycle's delta, but both were re-run 3x in
isolation as a sanity check: 3/3 PASS each, no flakes observed.

### Test inventory staleness (verifier cycle-1 note) — resolved

The cycle-1 verify report flagged that the test inventory grew by three
tests since the cycle-1 test report's count. This section's counts (599
shell / 25 files, 8/8 Go packages, 195 `internal/cli` test functions, four
new regression tests individually confirmed) are the current, up-to-date
counts as of `0cc50e5` and supersede the cycle-1 numbers for anything
downstream (`/sync-docs`, `/cross-review`, `/pr`) that needs a live count.

### What remains unverified (cycle 2)

- Static analysis and spec compliance — `/verify`'s job, already PASS for
  cycle 2 (`docs/reports/verify-2026-08-18-overlay-scaffold-v2-p4.md`,
  "Verify cycle 2" section).
- Documentation sync for the cycle-2 delta — `/sync-docs`'s job, not yet run.
- Cross-model second opinion on the cycle-2 delta — `/cross-review`'s job,
  not yet run (this delta was produced *by* the cycle-1 cross-review triage
  plus self-review fixes; a fresh cross-review pass on the fix commits
  themselves has not happened).

---

## Cycle 3

- Date: 2026-08-18
- Tester: `tester` subagent (Claude Code, Sonnet 5)
- Scope: behavioral tests via `./scripts/run-test.sh` (default changed-language
  scope — the cycle-3 delta touches only `internal/cli/*.go` plus docs), on
  `feat/overlay-scaffold-v2-p4`, HEAD `803d6ff`. This is pipeline cycle 3/3
  (cap raised from the default 2 to 3 per the plan's cross-review cycle-2
  ACTION_REQUIRED table); cycle-2 test already passed at `0cc50e5` (599/599
  shell + full `go test`, recorded above). No static analysis re-run — that
  is `/verify`'s job, already PASS in
  `docs/reports/verify-2026-08-18-overlay-scaffold-v2-p4.md` ("Cycle 3"
  section).
- Delta since cycle-2 test: `215c676` (cycle-2 cross-review AR#1/AR#2/AR#3
  fixes, +3 new e2e/classifier tests plus 2 companion tests
  (`TestClassifyMigration_UnmodifiedSeedSamePath_ReplacesWithTemplate`,
  `TestClassifyMigration_ModifiedSeedSamePath_StillForksInPlace`)),
  `9ac24de` (self-review C3-1..C3-7 fixes; +2 new tests
  (`TestClassifyMigration_UnavailablePackLegacyRulePath_Preserved` plus an
  extension of `TestRunMigrateLegacy_UnavailablePack_FilesPreservedNotDeleted`),
  one dead assertion removed from an existing test as never-firing per
  C3-6), `803d6ff` (doc-comment-only fix, no test surface change), plus
  report-only commits `cd125b7` / `5b88bae`.

### Verdict: PASS

599/599 shell tests across 25 files (byte-identical file set and per-file
counts to cycles 1–2 — the delta touches no shell test file), 8/8 Go
packages `ok` (fresh, `-count=1 -cover`, no cache), zero failures anywhere.
All five regression tests named in the assignment exist, ran, and passed
3/3 each in isolation with zero flakes; the two companion AR#1 tests
referenced by the verifier's "minimal additional check" section were also
independently confirmed 3/3. No test weakening found — the one assertion
removed (C3-6, a dead second check inside
`TestClassifyMigration_UnavailablePackPrefix_Preserved` that could never
fire independently of the surviving assertion) is documented in both the
self-review and verify cycle-3 reports as a correctness cleanup, not a
weakening of coverage.

### Test execution

| Suite / Command | Scope | Passed | Failed | Duration |
| --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (default changed-language scope) | 25 shell test files + golang pack | 599 shell + 8/8 Go pkgs | 0 | ~1m40s |
| `go test ./... -count=1 -cover` (fresh, no cache) | 8 Go packages with tests | 8 | 0 | ~50s |

`./scripts/run-test.sh` exit code: `0` (captured directly via
`>file 2>&1; echo $?`, not piped through `tee`, per
[[feedback_tee_masks_exit]]). `run-test.sh` selected `golang` as the sole
changed-language pack (`==> Language packs selected: golang`), and still ran
the full 25-file shell suite ahead of the language-scoped dispatch
(`grep -c '^==> tests/'` = 25; `grep -c '^FAIL'` = 0 anywhere in the log).
Per-file shell counts spot-checked against the established cycle-1/cycle-2
baseline (`test-check-mojibake.sh`=11, `test-check-skill-sync.sh`=13,
`test-hook-wiring.sh`=18, `test-ralph-dispatch.sh`=26, `test-secret-scan.sh`=6,
`test-sync-skills.sh`=22, `test-terraform-gitignore.sh`=47,
`test-xreview-helpers.sh`=29, `test-branch-name.sh`=26,
`test-ensure-pr-ready.sh`=7, `test-ensure-pr-title-prefix.sh`=13,
`test-gc-artifacts.sh`=11, `test-insights-append.sh`=39,
`test-ralph-worktree.sh`=29) — every spot-checked file matches the baseline
count exactly, confirming the 599 total holds without a shell test file in
this cycle's delta.

Fresh `go test ./... -count=1 -cover` (bypassing the `run-test.sh` build
cache to rule out stale results from before the cycle-3 commits):

| Package | Result | Coverage |
| --- | --- | --- |
| `internal/cli` | ok | 78.7% |
| `internal/config` | ok | 94.2% |
| `internal/insights` | ok | 86.1% |
| `internal/org` | ok | 89.1% |
| `internal/org/driver` | ok | 92.0% |
| `internal/org/protocol` | ok | 97.9% |
| `internal/scaffold` | ok | 75.7% |
| `internal/upgrade` | ok | 91.2% |

`internal/cli` coverage rose from cycle 2's 78.6% to 78.7% (the new
cycle-3 tests add their own well-covered lines). All other packages are
byte-identical to cycle 1/2's coverage baseline, as expected — this
cycle's delta touches only `internal/cli` source (`migrate.go`,
`language_pack.go`, `upgrade_v2.go`) plus test/doc files.

`internal/cli` package total: 202 test functions, 0 `FAIL` (`go test
./internal/cli/... -v -count=1 | grep -c '^--- PASS'` = 202, `grep -c
'^--- FAIL'` = 0; up from cycle 2's 195, consistent with the net test-count
delta above).

### Named regression tests (per assignment)

Ran with `go test ./internal/cli/... -run '^(<names>)$' -count=3 -v` to rule
out any batch-ordering or shared-state masking, isolated from the rest of
the suite:

| Test | Covers | Result (3 runs) |
| --- | --- | --- |
| `TestRunMigrateLegacy_UnmodifiedSeedSamePath_ReplacedOnDisk` | AR#1 — e2e, unmodified generic seed path replaced with template content on disk | 3/3 PASS |
| `TestClassifyMigration_UnavailablePackPrefix_Preserved` | AR#2 — classifier-level, unavailable-pack files under a preserve prefix classify as `OpUntouched`/`Preserved` | 3/3 PASS |
| `TestRunMigrateLegacy_UnavailablePack_FilesPreservedNotDeleted` | AR#2 — e2e (extended in `9ac24de` with C3-1's legacy rule-path assertion) | 3/3 PASS |
| `TestRunMigrateLegacy_SymlinkedRelocationDestParent_PlainDelete_ZeroWrites` | AR#3 — plain relocated `OpDeleteOldPath` validates `NewPath`'s parent chain against symlink escape | 3/3 PASS |
| `TestClassifyMigration_UnavailablePackLegacyRulePath_Preserved` | C3-1 — classifier-level, unavailable-pack's legacy-location rule file (`.claude/rules/<pack>.md`) preserved, not deleted | 3/3 PASS |

15/15 invocations pass (5 tests × 3 runs), 0 failures, 0 flakes.

**Companion tests** (referenced by the verifier's "minimal additional
check" section as part of the same AR#1 behavioral claim, run for
completeness beyond the assignment's five names):

| Test | Covers | Result (3 runs) |
| --- | --- | --- |
| `TestClassifyMigration_UnmodifiedSeedSamePath_ReplacesWithTemplate` | AR#1 — classifier-level positive case | 3/3 PASS |
| `TestClassifyMigration_ModifiedSeedSamePath_StillForksInPlace` | AR#1 — classifier-level negative/boundary case | 3/3 PASS |

6/6 additional invocations pass, 0 failures.

### Known-flaky tests re-checked in isolation

Per tester agent memory, `TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`
(`internal/cli`) and `TestRunWatcher_TimeoutIndependentOfSmallInterval`
(`internal/org`) have a documented history of transient subprocess-contention
flakiness. Neither is in this cycle's delta, but both were re-run 3x in
isolation as a sanity check: 3/3 PASS each, no flakes observed.

### Test plan cross-check (cycle-3 delta)

- AR#1 (unmodified generic seed path replacement) — unit test (classifier
  positive + negative boundary) and e2e test both present and passing.
- AR#2 (preserve-prefix threading through `ClassifyMigration`) — unit test
  and e2e test both present and passing.
- AR#3 (relocated plain-delete parent-chain symlink validation) — e2e test
  present and passing, mirrors the cycle-1 fix's existing
  `OpDeleteOldPathAdoptFork` coverage shape.
- C3-1 (unavailable pack's legacy-location rule file preserved) — unit test
  and e2e-extension both present and passing.
- C3-4 (dangling `BaselinePath` on the preserved carry-forward) and C3-7
  (dead `handled[path] = true` store) are non-test-surface cleanups; their
  correctness is covered transitively by the AR#2/C3-1 e2e tests exercising
  the same preserved-carry-forward code path, not by a dedicated new test —
  consistent with the self-review's own framing of these as low-severity
  code-hygiene fixes rather than behavioral changes needing independent
  pinning.
- C3-6 (dead assertion removed) — confirmed the removed check was genuinely
  unreachable independently of the surviving assertion (both would only
  fire together in the classifier under test), not a coverage loss.

No gap found between the cycle-3 delta's fix set and what actually shipped
in `migrate_test.go`.

### What remains unverified (cycle 3)

- Static analysis and spec compliance — `/verify`'s job, already PASS for
  cycle 3 (`docs/reports/verify-2026-08-18-overlay-scaffold-v2-p4.md`,
  "Cycle 3" section). That report also flags one open documentation-drift
  item (`relocatedRulePath`'s doc comment still describes the pre-C3-1
  invariant) as a minor, non-blocking gap — unaffected by this test cycle,
  belongs to `/sync-docs` or a follow-up doc fix.
- Documentation sync for the cycle-3 delta — `/sync-docs`'s job, not yet
  run for this cycle.
- Cross-model second opinion on the cycle-3 delta — `/cross-review`'s job,
  not yet run (this delta was produced *by* the cycle-2 cross-review
  triage plus self-review fixes; a fresh cross-review pass on the fix
  commits themselves has not happened, and the pipeline cycle cap is now
  at 3/3 — no further raise is available without an explicit operator
  decision per `.claude/rules/ralph/post-implementation-pipeline.md`).
- Whether the chained `runUpgradeV2`'s report/no-op behavior after an
  AR#1-driven seed replacement is genuinely idempotent on a second run —
  the existing rerun-stability test suite covers this in shape, but this
  test cycle did not add a dedicated second-run assertion for the new
  AR#1 replacement path specifically (same gap the verify cycle-3 report
  already noted as unverified from its own side).
