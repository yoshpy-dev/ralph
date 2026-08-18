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
