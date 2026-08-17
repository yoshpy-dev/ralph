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
