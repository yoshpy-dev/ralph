# Verify report: overlay-scaffold-v2 Phase 4 (legacy→v2 migration + owner-aware untracked classification)

- Date: 2026-08-18
- Plan: `docs/plans/active/2026-08-18-overlay-scaffold-v2-p4.md`
- Spec: `docs/specs/2026-08-17-overlay-scaffold-v2.md` (FR-6/7/8 + Phase-4 amendment)
- Verifier: `verifier` subagent (Claude Code, `/verify`)
- Branch: `feat/overlay-scaffold-v2-p4`, HEAD `b1babe7` (base `main` `1025d13`)
- Scope: spec compliance + static analysis, per `.claude/rules/ralph/post-implementation-pipeline.md`. No behavioral test execution (`/test`'s job).

## Verdict: PASS

`./scripts/run-static-verify.sh` (changed-language scope, Go) passes clean: `gofmt: ok`, golangci-lint `0 issues.`, plus repo-wide `check-sync.sh` / `check-pipeline-sync.sh` / `check-skill-sync.sh` / Codex hook guards all `OK`. `go build ./...` and `go vet ./...` both pass with no output. The prior self-review's 1 HIGH + 4 MEDIUM findings are all fixed correctly in `b1babe7`, each cross-checked against its own recommendation (not just "a fix landed"). All 16 acceptance criteria have code + test evidence; AC boxes in the plan are still unchecked (`[ ]`) — that is doc-lag against the actual implementation state, not a spec-compliance gap, and is called out below rather than silently ticked.

## Evidence

- `RALPH_VERIFY_SCOPE=changed ./scripts/run-static-verify.sh` → exit 0, evidence log `docs/evidence/verify-2026-08-18-092821.log`.
- `go build ./...` → clean. `go vet ./...` → clean (both run manually in addition to the wrapper, since the wrapper's Go verifier only surfaces gofmt/golangci-lint output explicitly).
- Self-review fix-commit crosscheck (`b1babe7` diff read in full for `internal/cli/migrate.go`, `README.md`, `docs/tech-debt/README.md`):
  - **HIGH-1** (fork attribution lost on rerun / collision case (a) with modified source) — fixed via new `OpDeleteOldPathAdoptFork` kind, `classifyRerunRelocatedDestination` (rerun pre-check) and `relocationOutcome`'s `sourceUnmodified` branch (collision case (a)). Test `TestRunMigrateLegacy_CollisionMatrixA_HalfMigrated_RerunDeletesOldOnly` (`internal/cli/migrate_test.go:1848`) was rewritten to assert `Owner == scaffold.OwnerFork` — the self-review explicitly flagged this test as "pinning the defect"; it now asserts the corrected behavior. A second test, `TestRunMigrateLegacy_RerunAfterPartialRelocation_ModifiedSource_DestAdoptsFork` (`:2003`), covers HIGH-1 item (1), the rerun-pre-check path, directly.
  - **MEDIUM-1** (`ErrUpgradeDriftRemaining` swallowed) — `runMigrateLegacy` now checks `errors.Is(err, ErrUpgradeDriftRemaining)` and returns it before the warning-only fallback (`internal/cli/migrate.go:~905`).
  - **MEDIUM-2** (delete/dir ops skip parent-chain symlink guard) — `validateMigrationOp` now calls `upgrade.ValidateRealParentChain(absDir, e.OldPath)` once, ahead of the per-kind switch, so no future op kind can skip it. New tests `TestRunMigrateLegacy_SymlinkedRelocationDestParent_ZeroWrites` and `TestRunMigrateLegacy_SymlinkedDeleteParent_ZeroWrites` exercise this.
  - **MEDIUM-3** (README "manifest-less" claim contradicts fail-closed code) — both README passages (`### ralph upgrade`, `#### Migrating from a legacy layout`) now say "legacy (pre-v2)" only.
  - **MEDIUM-4** (`PrunedHookCommands` computed by substring, applied by exact match) — `pruneLegacySettingsHooks` now returns the commands actually removed (exact-match subset) plus `nearMisses`; `executeMigrationEntries` writes both back into `plan.Entries` by index for the report to read; `renderMigrationReport` surfaces near-misses separately so an operator does not mistake survival for an oversight.
  - LOW items also addressed in `b1babe7`: `checkGitCleanForMigration` now captures git stderr (LOW-9); block-face report wording changed from past to future tense (LOW-11); `OpKeepInPlace` carry-forward now clears `BaselineStatus`/`BaselinePath` (LOW-6); `OpSettingsPrune` skips the read/rewrite entirely when there is nothing to prune (LOW-5); tech-debt row (3) count updated to "now-6 hardcoded-permission sites" including `writeMigrationFile` (LOW-8). LOW-7 (`buildMigratedManifest`'s optimistic `DiskHash = templateHash`) and LOW-10 (double pack-warning print) were left as documented, not silently dropped — both now carry an explanatory doc comment citing the self-review finding, consistent with the self-review's own framing ("acceptable as-is" for LOW-10, "note the dependency" for LOW-7).
- Owner-aware untracked classification (AC-1) traced end to end: `ReplaceOptions.OwnerForPath` (`internal/upgrade/replaceplan.go:447`) → `classifyUntracked`'s seed branch (`:464-467`, `diskHash != templateHash` && `ownerForPath(path) == OwnerSeed` → `Advisories`, not `Drift`) → `runUpgradeV2` wires `OwnerForPath: ownerForScaffoldPath` (`internal/cli/upgrade_v2.go:89`) → `rebuildManifestV2`'s `default` branch records `owner=seed`/`DiskHash=<actual disk content hash>` for seed paths (`:651-659`), not the template hash. No write to the seed-owned file happens anywhere in this path (no `FileOp` is appended for it) — the file stays byte-unchanged, matching AC-1's "ファイル不可侵" requirement.
- Spec FR-8 Phase-4 amendment (`docs/specs/2026-08-17-overlay-scaffold-v2.md:62`) reads consistently with the plan's own Design decisions section and with the shipped behavior (AGENTS.md/.gitignore modified → left in place, block appended by the chained v2 upgrade, duplication called out in the migration report — confirmed in `renderMigrationReport`'s "Block face duplicate-content guidance" section).
- `docs/tech-debt/README.md`: both named rows (seed-collision drift, `.codex/AGENTS.override.md` re-attribution) carry `RESOLVED 2026-08-18 in feat/overlay-scaffold-v2-p4` annotations that accurately describe the fixing mechanism (`ReplaceOptions.OwnerForPath` + `classifyUntracked`'s seed branch; `classifyLegacyPath`'s `pathCodexOverride` case). AC-12 holds.
- `internal/cli/init.go` / `internal/cli/pack.go` fail-closed doc comments (AC-10) now point operators at `ralph upgrade` / `runMigrateLegacy` instead of the stale "Phase 4" forward-reference.
- `AGENTS.md` repo map (`internal/upgrade/` entry) updated to describe the migration hand-off instead of "legacy layouts fail closed until Phase 4" — doc drift item from the plan's scope D is closed.

## Acceptance criteria

| AC | Description (short) | Evidence | Status |
| --- | --- | --- | --- |
| AC-1 | Owner-aware untracked classification: seed-owned collision → advisory, not drift | `internal/upgrade/replaceplan.go:439-469` (`classifyUntracked`), `internal/cli/upgrade_v2.go:89,651-659` | Holds |
| AC-2 | `OwnerForPath` nil → prior behavior; existing tests green | `internal/upgrade/replaceplan_test.go` (+120 lines, nil-resolver cases); no existing call site changed signature incompatibly | Holds (behavioral confirmation is `/test`'s job) |
| AC-3 | Legacy detection → git precondition → preview → confirm (`y`/`N`, `--yes`, `--dry-run`) → execute | `internal/cli/upgrade.go` dispatch to `runMigrateLegacy`; `internal/cli/migrate.go` `checkGitCleanForMigration`, `confirmMigration`, `RenderMigrationPreview` | Holds |
| AC-4 | Classification rules (FR-7): unmodified-relocated / unmodified-retired / modified-fork / unmanaged-fork | `classifyUnmodifiedGeneric`, `classifyForkCandidate`, `internal/cli/migrate_test.go` table-driven cases | Holds |
| AC-5 | Special faces (FR-8): CLAUDE.md seed-replace/untouched; AGENTS.md/.gitignore block-replace (unmodified) or keep+append (modified) | `internal/cli/migrate.go` special-path handling; spec FR-8 amendment matches | Holds |
| AC-6 | `.codex/AGENTS.override.md` → owner=seed unconditionally | `classifyLegacyPath`'s `pathCodexOverride` case; tech-debt row RESOLVED annotation | Holds |
| AC-7 | Post-migration state: v3 manifest, `.ralph/baseline/` gone, chained v2 upgrade converges, immediate re-run is no-op | `buildMigratedManifest` + `runMigrateLegacy`'s chain into `runUpgradeV2`; rerun-stability tests | Holds |
| AC-8 | Migration report at `docs/reports/ralph-migration-<date>.md` with classification list, fork diffs, block-face guidance | `renderMigrationReport` (report sections: classification table, fork diffs, settings prune + near-misses, block-face guidance) | Holds |
| AC-9 | Mid-failure: no v3 manifest write, error names git recovery; `--dry-run` zero writes | Preflight validation batch (`validateMigrationOp`) ahead of any write; manifest write is the barrier after `executeMigrationEntries` succeeds | Holds |
| AC-10 | fail-closed messages point to `ralph upgrade` for migration | `internal/cli/init.go`, `internal/cli/pack.go` diffs (this commit) | Holds |
| AC-11 | Pack rule relocation/fork; `Meta.Packs` carried into v3 manifest | `relocatedRulePath`, `buildMigratedManifest`'s `Meta.Packs` propagation | Holds |
| AC-12 | 2 tech-debt rows RESOLVED | `docs/tech-debt/README.md` diff (this commit) — both rows struck through with accurate RESOLVED annotations | Holds |
| AC-13 | Settings migration: unmodified → full template replace + snapshot; modified → exact-match legacy hook prune, near-misses reported, user hooks preserved | `pruneLegacySettingsHooks` (post-fix: returns `removed`/`nearMisses`), `OpSettingsPrune` dispatch, `SnapshotCreate` handling | Holds |
| AC-14 | Preflight batch validation before any write; rerun plans only remaining work | `validateMigrationOp` called over every entry before `executeMigrationEntries`; `ClassifyMigration`'s rerun-stability pre-check (now fork-attribution-correct post HIGH-1 fix) | Holds |
| AC-15 | Collision matrix (a)/(b)/(c) | `relocationOutcome`, `TestClassifyMigration_CollisionMatrix` (cases a/b/c), `TestRunMigrateLegacy_Collision_ZeroWrites` | Holds |
| AC-16 | Path safety: `CleanLocalRelPath` + `ValidateRealParentChain` + leaf `Lstat` on every migration op | `validateMigrationOp`'s hoisted `ValidateRealParentChain` call (post MEDIUM-2 fix) + existing `validateMigrationLeaf`/`validateMigrationWriteTarget`; symlink tests | Holds |

Plan checkboxes for AC-1..AC-16 are all still `[ ]` in `docs/plans/active/2026-08-18-overlay-scaffold-v2-p4.md` — per prior guidance this is treated as doc-lag, not a compliance failure, and is flagged under Documentation drift below rather than silently ticked by the verifier (ticking ACs is the plan owner's / `/pr` archival step's call, not `/verify`'s).

## Documentation drift

- Plan AC-1..AC-16 checkboxes are unchecked despite all 16 holding — recommend ticking before `/pr` archival, but not a spec-compliance blocker.
- Plan `Progress checklist`: "Plan reviewed", "Review artifact created", "Verification artifact created", "Test artifact created", "PR created" are all still unchecked; this verify pass produces the verification artifact this report covers.
- No other doc drift found: README migration section, spec FR-8 amendment, AGENTS.md map, and tech-debt rows are all in sync with the shipped code as of `b1babe7`.

## What remains unverified

- Behavioral correctness of the fixes (test execution) — explicitly out of scope for `/verify`, belongs to `/test`.
- `go test ./...` was not run here (tester's job); only `go build`/`go vet`/gofmt/golangci-lint were run as static checks.
- Runtime confirmation that `ralph upgrade` against a real legacy fixture project produces the exact report/manifest shape described (covered by the e2e tests referenced above, but not independently re-executed by hand in this pass).

## Minimal additional check for highest confidence gain

Run `/test` against this plan with focus on the collision-matrix and rerun-stability test files (`internal/cli/migrate_test.go`), since those are exactly the tests rewritten or added to pin the HIGH-1/MEDIUM-2 fixes — a green run there is the strongest behavioral confirmation that the self-review's own "test that pins the defect" concern is fully closed.

---

# Cycle 2 (final, 2/2)

- Date: 2026-08-18
- Verifier: `verifier` subagent (Claude Code, `/verify`)
- Scope: **delta only** — `b1babe7..HEAD`. Delta commits: `c51497e` (fixes for cross-review cycle-1 AR#1/AR#2/AR#3), `13adf85` (self-review cycle-2 section, pass, 1 MEDIUM + 3 LOW), `aabca2d` (fixes for self-review C2-1..C2-4). HEAD is `aabca2d`, working tree clean (`git status --porcelain` empty).
- Correction to cycle-1: this report's own Evidence section (line 24, cycle-1) claimed self-review LOW-6 and LOW-7 "both now carry an explanatory doc comment" at `b1babe7`. That was wrong for LOW-6: `b1babe7` only fixed LOW-7 (`buildMigratedManifest`'s optimistic `DiskHash` comment); LOW-6 (`buildDesiredStateV2` called twice, duplicate pack warning) had no comment at `b1babe7` and was correctly flagged as unaddressed by the cycle-2 self-review (its own finding C2-3). LOW-6's doc comment was only added in `aabca2d`, at the `buildDesiredStateV2` call inside `runMigrateLegacy` (`internal/cli/migrate.go:838-840`, in `buildMigratedManifest`'s call chain). Do not repeat the stale cycle-1 sentence going forward.

## Verdict: PASS

`./scripts/run-static-verify.sh` (changed-language scope, Go) passes clean at `aabca2d`: `gofmt: ok`, golangci-lint `0 issues.`, plus repo-wide `check-sync.sh` / `check-pipeline-sync.sh` / `check-skill-sync.sh` / Codex hook guards all `OK`. `go build ./...` and `go vet ./...` both pass with no output. All three cross-review AR fixes (`c51497e`) and all four self-review C2 fixes (`aabca2d`) were cross-checked line-by-line against their respective triage/review contracts, not just "a fix landed." AC-1..AC-16 still hold; no AC's contract was invalidated by the delta.

## Evidence

- `RALPH_VERIFY_SCOPE=changed ./scripts/run-static-verify.sh` → exit 0, evidence log `docs/evidence/verify-2026-08-18-132705.log`.
- `go build ./...` → clean. `go vet ./...` → clean.
- `git status --porcelain` → empty (nothing uncommitted).
- Cross-review AR fix crosscheck (`c51497e` diff read in full for `internal/cli/migrate.go`), against `docs/reports/cross-review-triage-overlay-scaffold-v2-p4.md`'s ACTION_REQUIRED table:
  - **AR#1** (`OpDeleteOldPathAdoptFork` validated `OldPath` only, not `NewPath`) — `validateMigrationOp`'s `OpDeleteOldPath, OpDeleteOldPathAdoptFork` case now additionally calls the new `validateMigrationForkAdoptionTarget(absDir, e.NewPath)` for the adopt-fork kind, after the existing `OldPath` leaf check. The new function applies `scaffold.CleanLocalRelPath` + `upgrade.ValidateRealParentChain` + `validateMigrationLeaf(..., mustExist=true)` — the same chain `validateMigrationWriteTarget` uses for write targets, with `mustExist=true` because this kind reads-and-trusts `NewPath` rather than writing it (doc comment states this explicitly and correctly). Matches the triage's prescribed fix exactly (`internal/cli/migrate.go:990-1009` doc, `:1010-1019` dispatch, `:1075-1093` new function).
  - **AR#2** (generic desired-sweep recorded `TemplateHash=current` for untracked seed collisions, making the chained upgrade's `classifyUntracked` see a "tracked" seed and swallowing the AC-1 advisory) — `buildMigratedManifest` now checks, for `owner == OwnerSeed` paths with existing diverging disk content and no legacy manifest entry (`!trackedInLegacy`), `continue`s before calling `SetFileOwned`, omitting the path from the v3 manifest entirely. Traced against `internal/upgrade/replaceplan.go:438-469`'s `classifyUntracked`: it fires only for paths with "no manifest entry at all," so omission is exactly what makes the chained call's `classifyUntracked` (not `classifyCore`/`classifySeed`, which require a manifest entry) see the path and route it to the seed-advisory branch instead of drift. This is the implementation choice the triage explicitly left to the implementer ("実装は連鎖側分類と整合する形を実装者が選択"), and it is the one shape that satisfies AC-1 through the migration path (`internal/cli/migrate.go:1505-1524`).
  - **AR#3** (migration report's fork-diff section omitted `OpDeleteOldPathAdoptFork` diffs, violating AC-8) — `renderMigrationReport`'s `forkEntries` now appends `entriesForKind(plan.Entries, OpDeleteOldPathAdoptFork)` alongside the existing `OpForkRelocate`/`OpForkInPlace` entries, before the `len(forkEntries) > 0` report-section gate. One-line fix, matches the triage recommendation (`internal/cli/migrate.go:1573-1581`).
- Self-review C2 fix crosscheck (`aabca2d` diff read in full), against `docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p4.md`'s cycle-2 findings table:
  - **C2-1** (8 bare `AR#1`/`AR#2`/`AR#3` citations would go stale once `/cross-review` rewrites the triage report this cycle) — all 8 sites now read `AR#1, cycle 1, ...` / `AR#2, cycle 1, ...` / `AR#3, cycle 1, ...`. Grepped `AR#1\|AR#2\|AR#3` across `internal/cli/migrate.go` and `internal/cli/migrate_test.go`: 8 hits, all cycle-qualified, matching the `upgrade_v2.go` convention the finding cited. No bare citation remains.
  - **C2-2** (settings-prune rewrite guard checked candidate count, not actual-removal count, so a near-miss-only prune still rewrote and re-marshaled the file) — `executeMigrationEntries`'s `OpSettingsPrune` case now wraps the `writeMigrationFile` call in `if len(removed) > 0`, where `removed` is `pruneLegacySettingsHooks`'s actual-removal return value (distinct from the pre-existing `len(e.PrunedHookCommands) == 0` outer guard, which only skips when there were zero *candidates*). Confirmed this is the narrower, correct guard: a near-miss-only run now produces zero writes, matching the finding's "leaving the file byte-identical is strictly the safer outcome" recommendation exactly (`internal/cli/migrate.go:1171-1202`).
  - **C2-3** (cycle-1 LOW-6, duplicate `buildDesiredStateV2` pack warning, was unfixed and undocumented at `b1babe7` despite the cycle-1 verify report claiming otherwise) — a new comment now precedes the `buildDesiredStateV2` call in `runMigrateLegacy`: "buildDesiredStateV2 also runs inside the chained runUpgradeV2, so any pack-availability warning prints twice per migration (self-review cycle-1 LOW-6; accepted duplication, not worth threading state)." This is the fix; see the correction note above the verdict for the cycle-1 report's own stale claim.
  - **C2-4** (3 doc comments left stale by the new `OpDeleteOldPathAdoptFork` kind) — all three updated: `MigrationEntry.ForkedFromVersion`'s doc now lists `OpDeleteOldPathAdoptFork` as a third producer; `MigrationEntry.NewPath`'s doc now describes the adopt-fork destination role; `classifyLegacyEntryState`'s doc no longer claims the top-level caller never consults it for the already-relocated case — it now correctly says `classifyRerunRelocatedDestination` deliberately calls it with `hasDisk=false` for that case. Verified the third one against the actual function body (`internal/cli/migrate.go:411-427`): `classifyLegacyEntryState(entry, "", false)` is called exactly as the updated doc describes. The lower-priority sibling item (`validateMigrationOp`'s doc not mentioning it also runs for no-write kinds) was fixed too, though the finding only flagged it as optional.
- AC re-confirmation for the delta: AR#1/AR#3 touch AC-16 (path safety) and AC-8 (migration report fork diffs) respectively — both still hold, now with a previously-missing case closed rather than newly broken. AR#2 touches AC-1 (owner-aware untracked classification) specifically through the migration path — AC-1's original evidence (cycle-1, `internal/upgrade/replaceplan.go`'s `classifyUntracked` for the direct-`upgrade` path) is unchanged; AR#2 additionally closes the migration-path route to the same AC, which cycle-1's AC-1 evidence did not yet cover. No other AC's evidence changed shape.
- `docs/plans/active/2026-08-18-overlay-scaffold-v2-p4.md`: all 16 AC checkboxes are now `[x]` (16 of 16 `- [x] AC-` lines counted) — the doc-lag flagged in cycle-1 is resolved as of this delta.

## Acceptance criteria (delta-relevant)

| AC | Status | Delta effect |
| --- | --- | --- |
| AC-1 | Holds | AR#2 additionally closes the migration-path route to the seed-collision advisory (cycle-1 evidence covered the direct-`upgrade` path only) |
| AC-8 | Holds | AR#3 closes a report-completeness gap (adopt-fork diffs were missing from "Fork diffs") |
| AC-14 | Holds | Preflight validation now also covers `OpDeleteOldPathAdoptFork`'s `NewPath` (AR#1); rerun-stability path unaffected by this delta beyond the AR#1 fix already covered by cycle-1's HIGH-1 |
| AC-16 | Holds, strengthened | AR#1 adds `NewPath` validation for the adopt-fork kind, closing the one op-kind gap the cycle-1 self-review's own HIGH-1 fix left open |
| AC-2..AC-7, AC-9..AC-13, AC-15 | Holds, unchanged | Not touched by this delta's commits (verified via `git diff b1babe7..HEAD --stat`: only `internal/cli/migrate.go`, `internal/cli/migrate_test.go`, plan, and report files changed) |

## Documentation drift

- None found in the delta beyond the correction noted above (this report's own cycle-1 line 24). `README.md`, `docs/tech-debt/README.md`, and the spec were not touched by `c51497e`/`13adf85`/`aabca2d` and remain in sync as established in cycle 1.

## What remains unverified

- Behavioral correctness of the AR#1/AR#2/AR#3 and C2-1..C2-4 fixes (test execution) — out of scope for `/verify`, belongs to `/test`. The delta adds tests directly exercising each (`internal/cli/migrate_test.go` +554 lines across the three delta commits, per self-review cycle-2's own evidence), but this pass did not execute `go test ./...`.
- Runtime confirmation of AR#2's "the chained upgrade's own report/no-op behavior is genuinely restored" claim — covered by a dedicated end-to-end test per the self-review's positive notes, not independently re-executed by hand here.

## Minimal additional check for highest confidence gain

Run `/test` focused on the four new/changed test names covering AR#1 (`TestRunMigrateLegacy_SymlinkedAdoptForkDestParent_ZeroWrites`), AR#2 (the end-to-end seed-advisory-through-migration test near `internal/cli/migrate_test.go:2405`), AR#3 (`TestRunMigrateLegacy_AdoptedForkDiff_IncludedInReport`), and C2-2 (the near-miss-only byte-identity regression test) — these four are the exact behavioral claims this verify pass could only confirm structurally, not by execution.
