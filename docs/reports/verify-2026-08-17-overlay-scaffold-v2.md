# Verify report: overlay-scaffold-v2 (Phase 1 — エンジン基盤)

- Date: 2026-08-17
- Plan: `docs/plans/active/2026-08-17-overlay-scaffold-v2.md`
- Spec: `docs/specs/2026-08-17-overlay-scaffold-v2.md`
- Verifier: `verifier` subagent (`/verify`)
- Scope: `git diff main...HEAD` on `docs/spec-overlay-scaffold-v2` (base `main` @ 1025d13). Spec compliance + static analysis only. Behavioral test execution is out of scope by design (`/test`).
- Prior artifact: `docs/reports/self-review-2026-08-17-overlay-scaffold-v2.md` — 3 MEDIUM + 8 LOW findings. Re-verified below that the 3 MEDIUM findings and the LOW spec typo were addressed by commits `5a90118` and `d4a0e88`.

## Verdict: PASS

No blocking gaps. All 9 Phase-1 acceptance criteria are satisfied by code + tests. Static analysis is clean. One pending doc-drift item (AGENTS.md repo map) is explicitly deferred to `/sync-docs` per the plan's own scope, not a verify failure.

## Static analysis

Ran `./scripts/run-static-verify.sh` (default changed-language scope; resolved to `golang`, evidence at `docs/evidence/verify-2026-08-17-070508.log`):

- `scripts/check-sync.sh`: PASS — `DRIFTED: 0`, 3 pre-existing `KNOWN_DIFF` entries unrelated to this diff, 0 `ROOT_ONLY`.
- `scripts/check-pipeline-sync.sh`: PASS — all pipeline-order references in sync.
- `scripts/check-skill-sync.sh`: PASS — 13 skills in lock-step.
- Go verifier (`packs/languages/golang/verify.sh`):
  - `gofmt -l .` → `gofmt: ok`
  - `go vet ./...` → silent (pass); independently reran `go vet ./...` directly → clean
  - `golangci-lint run ./...` → `0 issues.`
  - `staticcheck ./...` → silent (pass; binary present at `~/go/bin/staticcheck`, confirmed it executes via the verifier script's `command -v` gate)
- `go build ./...` (independent confirmation) → builds clean, confirming test files also type-check.

No findings from static analysis.

## Self-review follow-up verification

Re-checked the self-review's 3 MEDIUM findings against the current worktree state (not just the fix commits, per re-verify discipline — confirmed via `git status`/`git diff`, working tree is clean and matches `HEAD`):

| Finding | Fix commit | Verified as |
| --- | --- | --- |
| `SetFileOwned` silently accepted `OwnerFork` and produced a fork entry inconsistent with `SetFileFork` | `5a90118` | Fixed. `validateManagedOwner` (renamed from `managedForOwner`) now returns an error for `OwnerFork` (`internal/scaffold/manifest.go:208-217`); `SetFileOwned` propagates it and writes no entry (`manifest.go:176-191`). New negative test `TestSetFileOwned_RejectsFork` (`manifest_test.go:191-199`) confirms no entry is recorded. Repo-wide grep for `managedForOwner` returns zero hits — rename is complete, no dangling references. |
| `OwnedSettingsPaths` was an exported, documented contract with zero references anywhere, including tests | `5a90118` | Fixed via the "pin it with a test" option the self-review offered. `TestOwnedSettingsPaths_AnchorsMergeBehavior` (`settingsmerge_test.go:314-346`) asserts `OwnedSettingsPaths` equals the exact literal set the merge handlers use, plus a behavioral check that an unowned template key (`outputStyle`) is never introduced and an owned one (`env`) is merged. This closes the "declared vs. implemented contract can silently diverge" gap without requiring a larger refactor to drive the merge from the slice. |
| `AdvisoryEntry` doc claimed advisories only fire when disk differs from template, but `classifyFork` appends unconditionally | `5a90118` | Fixed. Doc comment now states the per-owner rule explicitly: fork paths always get an advisory (diff may render empty), seed paths only when the template hash changed (`replaceplan.go:77-84`). Matches `classifyFork` (unconditional append) vs. `classifySeed` (hash-gated) behavior exactly. |
| Spec's security section named the deleted `cleanTemplateRelPath` symbol | `d4a0e88` | Fixed. Spec line 172 now reads `scaffold.CleanLocalRelPath`, matching the actual exported symbol at `internal/scaffold/paths.go:12`. Repo-wide grep for `cleanTemplateRelPath` in `.go`/`.md` files under the worktree returns only the plan's line 36 (describing pre-PR state, correctly left as-is per the self-review's own note) and the self-review report itself (historical record). No remaining stale references in current-state docs. |

The 5 remaining LOW findings (fork `DiskHash` when disk-missing, `date` sanitization symmetry, `LegacySkipped` doc wording, `getObjPath`/`getArrayPath` signature asymmetry, root-runnable chmod test) were explicitly flagged by the self-review as optional/not required before merge. None were addressed in these commits; none affect spec compliance or static analysis, so they don't block this verdict. They remain open follow-ups, consistent with the self-review's own recommendation tier ("optional now, cheap").

## Spec compliance (Phase 1 acceptance criteria)

Phase 1 targets AC-1 through AC-9 of the plan (not the full spec's FR-1..FR-13/AC-1..AC-12, which span later phases per the plan's PR-series roadmap). Each is evidenced against code and tests below.

| AC | Requirement | Evidence | Status |
| --- | --- | --- | --- |
| AC-1 | manifest v3 owner/fork/layout round-trip via TOML; v1/v2 manifests read without error, owner stays legacy (unset) | `internal/scaffold/manifest.go:24-27` (`Layout` field), `:161-168` (`Owner`/`ForkedFromVersion` fields, both `omitempty`); tests `TestManifestRoundTripV3OwnershipFields` (`manifest_test.go:127`), `TestReadManifestV1V2_OwnerStaysLegacy` (`manifest_test.go:201`), `TestReadManifestV1Compatibility` (`manifest_test.go:95`) | PASS |
| AC-2 | replace planner: fork-recorded paths excluded from ops; fork-less hash mismatch classified as unresolved drift (no write); ops ordered (delete/create/update → report → manifest); no manifest-advance op on partial failure; re-plan after partial apply is stable | `internal/upgrade/replaceplan.go` (`PlanCoreReplace`, `ApplyOps`, `ManifestRefreshEntry`); tests `TestPlanCoreReplace_ForkSuppressedFromOpsAndCollectedAsAdvisory` (`:243`), `TestPlanCoreReplace_CoreUnresolvedDriftDoesNotOverwrite` (`:173`), `TestPlanCoreReplace_DeterministicOpOrdering` (`:446`), `TestApplyOps_PartialFailureStopsSubsequentOps` (`:538`), `TestPlanCoreReplace_ReplanAfterPartialFailureIsStable` (`:578`) | PASS |
| AC-3 | managed block: block-interior-only update, exterior bytes preserved, missing-marker append, corrupt-marker (one-sided/duplicate) error classification, no write on corruption | `internal/upgrade/block.go` (`UpdateManagedBlock`); table-driven `block_test.go` (21 test functions) including `TestUpdateManagedBlock_CRLFFile` (`:193`), `TestBeginMarker` (marker validation, `:251`); self-review confirmed CRLF/no-trailing-newline/blank-line-at-EOF coverage in its "Positive notes" | PASS |
| AC-4 | settings.json 3-way merge: user-added keys/array entries preserved, no writes outside the owned JSON path set, owned entries added/updated/removed by diffing old→new template, deterministic output order | `internal/upgrade/settingsmerge.go` (`MergeOwnedSettings`); tests `TestMergeOwnedSettings_UserAddedPermissionPreserved` (`:56`), `TestMergeOwnedSettings_UnknownTopLevelKeyPreserved` (`:178`), `TestMergeOwnedSettings_HooksEventArrayAddUpdateRemove` (`:128`), `TestMergeOwnedSettings_DuplicateEntriesDeduplication` (`:208`), `TestMergeOwnedSettings_DeterministicDoubleMergeIsNoop` (`:223`), plus the new `TestOwnedSettingsPaths_AnchorsMergeBehavior` (`:314`) pinning the owned-path contract itself | PASS |
| AC-5 | advisory diff = unified diff of disk-vs-new-template, collected for files where the recorded template hash changed | `internal/upgrade/advisory.go` (`RenderAdvisoryDiffs`); tests `TestRenderAdvisoryDiffs_IdenticalContents` (`:22`), `TestRenderAdvisoryDiffs_ChangedContents` (`:45`); planner-side gating in `classifySeed` (`replaceplan.go:304-309`, hash-gated) vs. `classifyFork` (unconditional, per the now-corrected doc comment) | PASS |
| AC-6 | existing test suite passes; no `internal/cli` behavior change in this PR | `git diff main...HEAD -- internal/cli/` is empty — zero lines touched. (Full suite execution is `/test`'s responsibility; static analysis — build + vet — confirms all packages including `internal/cli` still compile against the new `internal/scaffold`/`internal/upgrade` APIs.) | PASS (no-diff evidence); full-suite pass confirmation deferred to `/test` per pipeline division of labor |
| AC-7 | spec document included in the same PR | `docs/specs/2026-08-17-overlay-scaffold-v2.md` present and committed (`7d3faca`), later corrected by `d4a0e88` | PASS |
| AC-8 | v3 opt-in isolation: `ralph init`/`ralph upgrade`'s current manifest-writing paths (legacy constructors) never emit `layout`/`owner`/fork-record fields | `TestExistingConstructorsWriteNoV3Fields` (`manifest_test.go:238-256`) marshals output of every legacy setter (`SetFile`, `SetFileWithBaseline`, `SetFileResolvedWithBaseline`, `SetFileUnmanaged`, `WithTemplateHash`) and asserts none of `layout`/`owner`/`forked_from_version` appear in the TOML | PASS |
| AC-9 | all primitives reject non-local paths (`..`, absolute) via the shared validator before generating operations | `internal/scaffold/paths.go` (`CleanLocalRelPath`, shared); consumers: `replaceplan.go:408` (`cleanPathKey`), `advisory.go:33`, `report.go:198`; negative tests `TestPlanCoreReplace_RejectsNonLocalManifestPath`/`RejectsAbsoluteManifestPath` (`replaceplan_test.go:420,432`), `TestRenderAdvisoryDiffs_InvalidPath` (`advisory_test.go:125`), `TestWriteUpgradeReport_RejectsParentEscape` (`report_test.go:150`). `block.go`'s `UpdateManagedBlock` and `settingsmerge.go`'s `MergeOwnedSettings` take only in-memory byte content (no path argument), so the "operates before generating operations" path-validation requirement does not apply to them directly — their callers (which resolve file paths) are the ones that must validate, and those callers are exactly the three tested above. This is consistent with the plan's Scope section describing these two as content-only primitives. | PASS |

### Explicitly out of scope for this Phase-1 verify (per plan's Verify plan section)

The spec's wiring-level FRs are later-phase scope, not gaps in this PR:

- FR-1 (`ralph upgrade`'s full 8-step non-interactive flow) — requires CLI wiring, Phase 3.
- FR-2/FR-3 (`ralph eject`/`ralph adopt` commands) — Phase 5.
- FR-6/FR-7/FR-8 (legacy-layout migration, migration classification, CLAUDE.md/AGENTS.md migration) — Phase 4.
- FR-9 (`ralph doctor --strict`), FR-10 (upstream-leak guard), FR-12 (`ralph status` ownership display) — Phase 5.
- FR-11 (`ralph init` v2 layout generation) — Phase 2.
- FR-13 (removal of interactive conflict resolution) — Phase 3.
- NFR-1/NFR-3/NFR-5 (idempotency, worktree compat, Codex parity) — depend on the wired flow, Phase 3+.

Phase 1's own plan-level Verify plan asked to confirm foundational alignment with spec FR-1 (primitives 1–6 exist), FR-4 (non-destructive drift classification), FR-5 (block spec), and NFR-2 (non-destructiveness). All four are satisfied at the primitive level:

- FR-1 primitives exist and are tested: replace planner, block engine, settings merge, advisory diff, report writer — all present, all unwired (confirmed via the AC-6 empty `internal/cli` diff and repo-wide grep for non-test callers of new exported symbols, which the self-review already ran and this verify's AC-6/AC-9 checks corroborate).
- FR-4: `TestPlanCoreReplace_CoreUnresolvedDriftDoesNotOverwrite` proves no write occurs on fork-less hash mismatch.
- FR-5: block corruption/missing-marker classification tested without writes (see AC-3 row).
- NFR-2: no destructive path found in the diff; `ApplyOps`'s ordered plan + commit barrier is structurally incapable of partial-manifest-advance (self-review's "Positive notes" independently confirms this).

## Documentation drift

- `AGENTS.md` repo map line for `internal/upgrade/` (`AGENTS.md:60` in this worktree) still reads "hash-based diff engine, conflict resolution (auto-update, conflict, add, remove)" — describes the pre-PR `diff.go`/`merge.go` machinery only, with no mention of the five new Phase-1 primitives (`replaceplan.go`, `block.go`, `settingsmerge.go`, `advisory.go`, `report.go`). This is a known, expected gap: the plan's own Affected areas section (`docs/plans/active/2026-08-17-overlay-scaffold-v2.md:58`) and Implementation outline step 6 both name this line explicitly as `/sync-docs` follow-up work, not something this PR was expected to close. Flagged here as doc drift, not a verify failure.
- Spec ↔ plan cross-references checked: plan line 6/25 point at the correct spec path; spec's References section lists the pre-PR implementation files (`diff.go`, `merge.go`, `upgrade.go`, `manifest.go`, `baseline.go`) as the "related implementation" for background, which remains accurate — those files still exist and are what Phase 3 will replace.

## Evidence

- Static analysis log: `docs/evidence/verify-2026-08-17-070508.log`
- Independent build/vet confirmation: `go build ./...` and `go vet ./...` both clean, run directly in the worktree (2026-08-17)
- Fix-commit review: `git show 5a90118`, `git show d4a0e88` (full diffs inspected)
- Working tree state at verify time: `git status` clean, `HEAD` = `d4a0e88`

## What remains unverified

- Full behavioral test suite execution (`go test ./...` pass/fail, coverage) — `/test`'s responsibility, not run here by design.
- `AGENTS.md` repo map sync — `/sync-docs`'s responsibility; flagged above as known drift, not blocking.
- Runtime behavior of the wired flow (FR-1 full 8-step upgrade, migration, eject/adopt, doctor --strict) — no runtime to exercise yet; these primitives are deliberately unwired in Phase 1 and will be verified against the CLI in Phase 3–5.

## Cycle 2 (re-run after cross-review fixes + cycle-2 self-review cleanup)

- Date: 2026-08-17
- Pipeline cycle: 2/2 (`RALPH_STANDARD_MAX_PIPELINE_CYCLES` default cap; no automatic third cycle)
- Scope: delta `6564230..HEAD` — `4c38cf1` (AGENTS.md repo-map sync + `check-sync.sh` `KNOWN_DIFFS`), `ba2384f` (cross-review triage report), `1ef5be7` (cross-review fixes: `ManifestRemove` signal, `ApplyOps` validate-all-upfront, report-path prefix hardening), `c81eee2` (cycle-2 self-review report), `3b32c50` (cycle-2 self-review fixes)
- Worktree state at this verify: `git status` clean, `HEAD = 3b32c50`

### Verdict: PASS

No blocking gaps. Static analysis clean. The two ACTION_REQUIRED cross-review findings and the one WORTH_CONSIDERING finding were closed exactly as triaged. All eight cycle-2 self-review findings (C2-1..C2-8) were addressed — six documentation/naming fixes plus two tech-debt register entries — with no regression to AC-2, AC-6, AC-8, or AC-9.

### Static analysis (re-run)

`./scripts/run-static-verify.sh` (default changed-language scope, resolved to `golang` via full-fallback because `scripts/check-sync.sh` is unclassified; evidence at `docs/evidence/verify-2026-08-17-074703.log`):

- `scripts/check-sync.sh` — PASS, `DRIFTED: 0`, 4 `KNOWN_DIFF` (one more than cycle 1: `AGENTS.md`, added in `4c38cf1` and accounted for in the cycle-2 self-review's C2-3/C2-4 tech-debt rows — see Documentation drift below), 0 `ROOT_ONLY`.
- `scripts/check-pipeline-sync.sh` — PASS, all references in sync.
- `scripts/check-skill-sync.sh` — PASS, 13 skills in lock-step.
- Go verifier: `gofmt -l .` → `gofmt: ok`; `golangci-lint run ./...` → `0 issues.`
- Independently re-ran outside the wrapper: `go build ./...` clean, `go vet ./...` clean, `staticcheck ./internal/upgrade/... ./internal/scaffold/...` silent (pass).

No findings from static analysis. Same clean result as cycle 1, now covering the cycle-2 delta.

### Cross-review fix verification (`1ef5be7` against `docs/reports/cross-review-triage-overlay-scaffold-v2.md`)

| Triage item | Fix | Verified as |
| --- | --- | --- |
| ACTION_REQUIRED #1 — planner emits no removal signal when a template-deleted core path is already absent from disk | `ReplacePlan.ManifestRemove` (new field, `replaceplan.go:102-111`) | Fixed and covers both cases named in the triage: disk-present-unmodified (`OpDelete` + `ManifestRemove`, existing test extended) and disk-already-absent (`ManifestRemove` only, new `TestPlanCoreReplace_CoreManifestRemoveWhenDiskAlreadyAbsent`). Negative check confirmed: the drifted case (modified core file) explicitly asserts `len(plan.ManifestRemove) == 0`, so the non-destructive default is not weakened. |
| ACTION_REQUIRED #2 — `ApplyOps` does not re-validate op paths, so a hand-built `ReplacePlan` could write/delete outside `targetDir` | Upfront validation loop over `plan.Ops` via `cleanPathKey` before any op executes (`replaceplan.go:376-381`) | Fixed. `TestApplyOps_RejectsInvalidOpPathBeforeWritingAnything` proves both properties the triage asked for: the invalid path is rejected (error names the path) and the *valid* op preceding it in the list is never written — genuine validate-all-upfront, not per-op validation with partial writes. |
| WORTH_CONSIDERING #3 — `UpgradeReportRelPath` does not sanitize `date`, allowing `docs/reports/...` escape via a crafted date string | `date` now sanitized through the same regex as `version`; `WriteUpgradeReport` gained a prefix check requiring the cleaned path to resolve under `docs/reports/` | Fixed, and hardened further in cycle 2 (see C2-7 below). `TestUpgradeReportRelPath_SanitizesDateParentEscape` and `TestWriteUpgradeReport_RejectsPathOutsideReportsDir` both pass and assert the file was never written on rejection. |

### Cycle-2 self-review fix verification (`3b32c50` against `docs/reports/self-review-2026-08-17-overlay-scaffold-v2.md` Cycle 2 section)

| Finding | Fix | Verified as |
| --- | --- | --- |
| C2-1 (MEDIUM) — `AdvisoryEntry` doc's second attempt still misdescribed rendering ("skips it" for empty-diff fork advisories, when the renderer emits a `_No differences._` section) | Doc comment rewritten at `replaceplan.go:77-83` | Fixed. Cross-checked against the actual renderer: `renderAdvisoriesSection` (`report.go:131-149`) emits `_No differences._` for `Diff == ""` and non-`Skipped`, distinct from the `Skipped` branch's `_Skipped: %s._`. The new doc text ("renders in the report as a '_No differences._' section, not a hidden or omitted one") now matches code exactly, and no longer overloads "skip" for two different behaviors. |
| C2-2 (MEDIUM) — `TestOwnedSettingsPaths_AnchorsMergeBehavior`'s doc comment overclaimed anchoring against the merge implementation when it only compares two literals | Doc comment reworded, no behavior change | Fixed as a documentation correction (the self-review offered "reword or strengthen"; the weaker option was taken). New comment accurately states it is "a deliberate change-detector on the declared list ... plus a separate regression test" and explicitly disclaims verifying each handler. Test logic itself (`settingsmerge_test.go:311-320`) is unchanged — still a `slices.Equal` literal comparison plus the `env`/`outputStyle` behavioral check, exactly as before. |
| C2-3 (MEDIUM) — `AGENTS.md` `KNOWN_DIFFS` whole-file suppression untracked in tech-debt register or plan Deviations | New `docs/tech-debt/README.md` row + plan Deviations entry (`3b32c50`) | Fixed. Tech-debt row names the exact trigger ("Remove the `AGENTS.md` entry ... when Phase 2 lands the template-facing rollout") and cross-references the self-review, `check-sync.sh`, and the plan. Plan `docs/plans/active/2026-08-17-overlay-scaffold-v2.md:123` now records the `check-sync.sh` edit alongside the two other out-of-band fixes. |
| C2-4 (MEDIUM) — 6 of 8 cycle-1 LOWs unfixed and absent from any register | Batched tech-debt row (5 items) + LOW-7 fixed outright (see below) | Fixed. Batched row lists all 5 remaining items (fork `DiskHash` fallback, `LegacySkipped` doc, `WriteUpgradeReport` perm literals, `getObjPath`/`getArrayPath` shape, unset-map-key test) individually — spot-checked each against current code, all 5 confirmed still present exactly as described (none silently regressed or double-counted). LOW-7 (`chmod 0444` root-defeat) was fixed via a root-skip guard (`os.Geteuid() == 0` → `t.Skip`) rather than the self-review's suggested uid-independent technique; a code comment explains why the directory-collision technique doesn't fit this test's OpUpdate-shape requirement. This is a documented, reasoned deviation from the recommendation, not a gap — it resolves the actual failure mode (hard-fail under root CI) the finding named. |
| C2-5 (LOW) — `ManifestRemove` absent from `ApplyOps`' manifest-state enumeration despite `ManifestRemove`'s own doc pointing there | `ApplyOps` doc comment extended (`replaceplan.go:369-374`) | Fixed. Enumeration now names `ManifestRemove` explicitly as "the one entry in this list that removes rather than advances a manifest hash." |
| C2-6 (LOW) — `versionSanitizeRe` name no longer matches its scope (applies to `date` too) | Renamed to `reportNameSanitizeRe` (both declaration and both call sites) | Fixed. Repo-wide grep for `versionSanitizeRe` returns zero hits; `reportNameSanitizeRe` used consistently at declaration and both `UpgradeReportRelPath` call sites. |
| C2-7 (LOW) — `WriteUpgradeReport`'s prefix guard carried an unexplained, untestable exception (`relPath == "docs/reports"` admitted) and duplicated the reports-dir literal | Guard simplified to a plain `strings.HasPrefix(clean, upgradeReportDir+"/")`; `upgradeReportDir` const (now `"docs/reports"`, no trailing slash) is the single source both `UpgradeReportRelPath` and `WriteUpgradeReport` build from | Fixed. New `TestWriteUpgradeReport_RejectsReportsDirItself` proves the directory-only path is now rejected and confirms no file is created at `docs/reports`. `UpgradeReportRelPath` now calls `filepath.Join(upgradeReportDir, ...)` instead of the literal `filepath.Join("docs", "reports", ...)` — verified via `git show 3b32c50 -- internal/upgrade/report.go`, one source of truth as claimed. |
| C2-8 (LOW) — insight events file under-reports the pipeline (missing `verify`/`sync_docs` phases) | Two events appended to `docs/insights/events/2026-08-17-overlay-scaffold-v2.jsonl` | Fixed. File now has 5 lines (self_review, test, cross_review, verify, sync_docs), matching the 5 report files under `docs/reports/`. Both new lines use `"cycle":1` (correct per project memory: insight cycle tracks Ralph Loop outer-cycle, not the pipeline cycle cap — cycle-2 pipeline events would still record `cycle:1` unless a new outer Ralph Loop cycle started, which it did not here). |

### Plan AC re-check (no regression)

- AC-2 (replace planner ordering/drift/manifest-barrier semantics) — extended, not weakened: `ManifestRemove` only fires for the two non-destructive template-removal cases (unmodified-disk-delete, already-absent), and the drifted case explicitly asserts zero `ManifestRemove` entries. `ApplyOps`' pre-existing failure semantics (stop-at-first-failure, no manifest advance) are unchanged; the new upfront path validation runs *before* the existing op loop and returns before any write, so it composes with — doesn't replace — the partial-failure contract cycle 1 verified. AC-2: **holds**.
- AC-6 (no `internal/cli` behavior change) — `git diff main...HEAD -- internal/cli/` is still empty at `HEAD = 3b32c50`. AC-6: **holds**.
- AC-8 (v3 opt-in isolation) — no touch to `manifest.go`'s v3 write paths or `TestExistingConstructorsWriteNoV3Fields` in this cycle's delta. AC-8: **holds**.
- AC-9 (shared path validator) — strengthened: `ApplyOps` now also runs `cleanPathKey` on every op path before executing any op, closing the one gap the cross-review found (a hand-built `ReplacePlan` bypassing `PlanCoreReplace`'s own validation). `WriteUpgradeReport`'s guard is stricter than cycle 1 (prefix match, no directory-itself exception). AC-9: **holds, strengthened**.

### Documentation drift (cycle 2)

- The cycle-1 verify report's flagged `AGENTS.md` repo-map drift was closed by `4c38cf1` (`internal/upgrade/` line now describes the Phase-1 primitives). That fix introduced a new, smaller drift-adjacent item: `AGENTS.md` added to `check-sync.sh`'s `KNOWN_DIFFS` as a whole-file suppression. This is not a doc-drift *failure* — it's an intentional, now-tracked scope trade-off (Phase 1 cannot touch `templates/base/AGENTS.md` per plan scope) with a concrete removal trigger recorded in both `docs/tech-debt/README.md` and the plan's Deviations section (C2-3 fix, verified above). Flagged here for visibility, not blocking.
- No other new documentation drift found in the cycle-2 delta.

### Evidence (cycle 2)

- Static analysis log: `docs/evidence/verify-2026-08-17-074703.log`
- Independent build/vet/staticcheck confirmation: run directly in the worktree, 2026-08-17, all clean
- Fix-commit review: `git show 1ef5be7`, `git show 3b32c50` (full diffs inspected, including test files)
- Triage cross-reference: `docs/reports/cross-review-triage-overlay-scaffold-v2.md`
- Self-review cross-reference: `docs/reports/self-review-2026-08-17-overlay-scaffold-v2.md` (Cycle 2 section, C2-1..C2-8)
- Sanity test run (supplementary; full-suite pass/fail remains `/test`'s responsibility): `go test ./internal/upgrade/... ./internal/scaffold/...` — both packages pass

### What remains unverified (cycle 2)

- Full behavioral test suite re-execution for this delta — `/test`'s responsibility; not re-run here beyond the two-package sanity check above.
- The 5 cycle-1 LOWs now recorded in the batched tech-debt row remain unfixed by design (pipeline cap reached); they are follow-ups, not blockers, and are honestly tracked.
- Runtime behavior of the wired flow — unchanged from cycle 1, still Phase 3+ scope.
