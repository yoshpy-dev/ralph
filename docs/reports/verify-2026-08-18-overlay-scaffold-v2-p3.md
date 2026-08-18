# Verify report: overlay-scaffold-v2 Phase 3

- Date: 2026-08-18
- Plan: `docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md`
- Spec: `docs/specs/2026-08-17-overlay-scaffold-v2.md`
- Verifier: `verifier` subagent (Claude Code, Sonnet 5)
- Scope: spec compliance (Phase-3-scoped FRs/NFRs + plan AC-1..AC-12) +
  static analysis via `./scripts/run-static-verify.sh`, on
  `feat/overlay-scaffold-v2-p3`, base `main`, HEAD `3d8461d` (working tree
  clean). No behavioral test execution — that is `/test`'s job. This is a
  fresh retry after a prior verifier run died on a transient network error
  before producing any artifact.

## Verdict: PASS

No spec-compliance gaps and no static-analysis failures against Phase 3's
scope. One partial-fix gap noted below (non-blocking, tracked here as a
verify finding since it was not caught by self-review's own recommendation
tracking).

## Self-review fix-commit crosscheck

Commit `3d8461d` ("fix: address phase-3 self-review findings") was checked
against `docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p3.md`'s
findings table, row by row:

| Finding | Fixed in 3d8461d? | Evidence |
| --- | --- | --- |
| MEDIUM 1 — `applyBlockUpdatesV2` missing `.ralph/core/AGENTS.core.md` guard, would silently blank AGENTS.md's managed block | Yes | `internal/cli/upgrade_v2.go:344-354` adds `hasAgentsTemplate` check mirroring the `.gitignore` guard; new test `TestRunUpgradeV2_MissingAgentsCoreTemplate_LeavesAgentsMdUntouchedWithNote` |
| MEDIUM 2 — `preserve()` doesn't actually preserve pack payload on the failure branches (payload already written into `desired` before the failure check) | Yes | `internal/cli/upgrade_v2.go:279-317` buffers into a local `packEntries` map, merged into `desired` only after full success; new test `TestRunUpgradeV2_PackRuleReadFailure_FullyPreserved` |
| MEDIUM 3 — `settingsSnapshotRelPathForSkip` duplicate constant, no drift detection | Yes | `internal/upgrade/snapshot.go` exports `SettingsSnapshotRelPath`; cli-side duplicate deleted, both `upgrade_v2.go` and `rebuildManifestV2` reference the single source |
| MEDIUM 4 — `docs/tech-debt/README.md` stale (6 rows premised on deleted mechanisms) | Yes, with one gap | Rows 25 (struck through, `RESOLVED-by-removal`), the batched cycle-1 LOW row (now names the 5th `writeFileV2` 0755 permission site), the shape-transition row (rewritten for `PlanCoreReplaceDesired`, `UPDATED` comment), the `pre_bash_guard.sh` row (expired Phase-3 trigger dropped), and the Codex-dispatcher-parity row (re-pointed Phase 3 → Phase 5) were all updated. **Gap:** the self-review's own recommendation #3 also asked to "add the three new rows from this report" (git-hook chaining coverage loss, `--pager` coverage loss, `buildDesiredStateV2`'s untested `apErr` branch) — `grep -rn "installGitHook\|pager\|apErr" docs/tech-debt/README.md` finds none of the three added. See Findings below. |
| MEDIUM 5 — 4 unreachable `internal/scaffold` manifest API members (`SetFileUnmanaged`, `WithTemplateHash`, `FileStatePartial`, `BaselineStatusAvailable`) | Yes | All four deleted; `internal/scaffold/manifest_test.go` rewritten to construct `ManifestFile{}` literals directly and use bare string literals (`"partial"`, `"available"`) with an explanatory comment instead of the deleted constants |
| LOW — `runUpgradeIOWithOptions`'s unused `in io.Reader` param | Deferred, documented | `internal/cli/upgrade.go:117-119` adds a doc comment explaining `in` is kept only for call-site signature stability; not a silent drop |
| LOW — `--diff` help text says "conflict diffs" | Yes | `internal/cli/upgrade.go:55` now reads "show advisory diffs without writing files (implies --dry-run)" |
| LOW — `errLegacyLayoutFailClosed` doc says "wrapped" but all call sites return it bare | Yes | Doc comment reworded to "returned as-is (never wrapped)... so callers... can match on it directly with errors.Is or ==" |
| LOW — bare `return err` on `.ralph/baseline` path-clean failure | Yes | `internal/cli/upgrade_v2.go:190` now wraps: `fmt.Errorf("resolving .ralph/baseline path: %w", err)` |
| LOW — `writeFileV2` hardcoded `0755`, a 5th permission-decision site | Deferred, documented | Folded into the updated tech-debt row (item 3) instead of a code fix — consistent with the row's own convention of tracking hardcoded-permission sites collectively rather than fixing them piecemeal |
| LOW — AGENTS.md `.ralph/local/` bullet stale "Phase 3 tech debt" label | Yes | `AGENTS.md` now reads "Phase 5 tech debt", matching the plan's Non-goals and the re-pointed tech-debt row |

11 of 12 findings closed exactly as claimed by the plan's implicit "M1-M5 +
4 LOWs fixed, 2 LOWs deferred with documented reasons" summary. The one gap
(MEDIUM 4's three new tech-debt rows not appended) is a partial fix of a
MEDIUM finding, not a new defect — see Findings.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | doc drift | The self-review's MEDIUM-4 recommendation had two parts: (a) re-point/rewrite the 6 stale tech-debt rows, and (b) "add the three new rows from this report" (`installGitHook` chaining coverage loss, `--pager` coverage loss, `buildDesiredStateV2`'s untested `apErr` branch). Part (a) is fully done in `3d8461d`; part (b) was not done — `grep -rn "installGitHook\|pager\|apErr\|AvailablePacks" docs/tech-debt/README.md` returns no matches. These three gaps remain real (the self-review's own "Tech debt identified" table documents them with impact/trigger/evidence) but are invisible to anyone reading only `docs/tech-debt/README.md`. | `docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p3.md` ("Tech debt identified" table, Recommendation #3); `docs/tech-debt/README.md` (no matching rows) | Append the three rows from the self-review's "Tech debt identified" table to `docs/tech-debt/README.md` in a follow-up commit before or shortly after merge — the register is the canonical place these are supposed to be discoverable from, per this repo's own tech-debt convention. |

No CRITICAL, HIGH, or MEDIUM findings from this verify pass. The above is a
LOW doc-completeness gap in a MEDIUM finding's fix, not a fresh defect.

## Spec compliance — FR/NFR mapping (Phase 3 scope)

| Requirement | Status | Evidence |
| --- | --- | --- |
| FR-1 (non-interactive upgrade, steps 1-8) | Met | `internal/cli/upgrade_v2.go` `runUpgradeV2`: desired-state build → skip-paths → `ApplyOps` → block update → settings 2-phase snapshot merge → seed/advisory → report write → manifest rebuild (barrier: manifest only rebuilt after all prior steps succeed, `upgrade_v2.go:190-210`) |
| FR-3 (`--force` removed) | Met | `grep -n force internal/cli/upgrade.go internal/cli/upgrade_v2.go` → no matches; `init.go`'s unrelated `--force` (existing-file overwrite on `ralph init`) is untouched and out of FR-3's scope |
| FR-4 (drift non-destructive, exit-code warning) | Met | Drift paths left untouched (`ApplyOps` never targets `plan.Drift`), `ErrUpgradeDriftRemaining` wraps the count + report path (`upgrade_v2.go:205`), `cmd/ralph/main.go:28-30` maps it to `os.Exit(3)`; all other errors → exit 1; success → exit 0. `root.go`'s `SilenceErrors`/`SilenceUsage` keep `main.go` the sole exit-code source (confirmed unchanged from self-review's positive note) |
| FR-5 (managed block update) | Met | `applyBlockUpdatesV2` (`upgrade_v2.go:337-380`) updates AGENTS.md/`.gitignore` block interiors only via `UpdateManagedBlockStyled`; malformed → left in place + report note (AC-12, see below) |
| FR-13 (interactive engine + baseline removal) | Met | `grep -rn "resolveConflict\|PlanMerge\|ComputeDiffs"` → zero hits repo-wide; `internal/upgrade/diff.go`, `merge.go`, `internal/scaffold/baseline.go` and their tests are deleted (`ls internal/upgrade/*.go` confirms absence); `.ralph/baseline/` cleanup runs on every successful v2 upgrade (`upgrade_v2.go:187-194`) |
| NFR-1 (idempotency) | Met | `TestRunUpgradeV2_Idempotent_SecondRunIsNoOp` (per self-review's deleted-test inventory crosscheck) |
| NFR-2 (non-destructive) | Met | `ApplyOps` Lstat preflight (AC-4b) + preserve-prefix fix (M2) + drift/fork carry-over in `rebuildManifestV2` together keep any non-machine-owned content untouched; static analysis found no regression in this area |
| NFR-4 (network independence) | Met | `runUpgradeV2` reads only from the embedded `baseFS`/pack `embed.FS` and on-disk state; no network client imports appear in `internal/cli/upgrade_v2.go` or `internal/upgrade/` |
| FR-2, FR-6..FR-12, NFR-3, NFR-5 | Out of scope (Phase 4/5, or already delivered in Phase 1/2) | Plan's Non-goals section names these explicitly; not re-verified here |

## Plan acceptance criteria mapping

| AC | Status | Evidence |
| --- | --- | --- |
| AC-1 | Met | End-to-end flow present and ordered as specified; `TestRunUpgradeV2_FullPass` / `TestRunUpgradeV2_MixedScenario_AllClassesLandCorrectly` per self-review's test inventory |
| AC-2 | Met | `in io.Reader` parameter is accepted but never read (`_ = in`, documented); no `Scan`/`Read` call on stdin anywhere in `internal/cli/upgrade_v2.go` (`grep -n "in\.\(Read\|Scan\)"` → no matches) |
| AC-3 | Met | `.claude/settings.json` and the settings snapshot are both in `v2SkipPaths()` (`upgrade_v2.go:35-51`), routed instead through `MergeOwnedSettings` |
| AC-4 | Met | See FR-4 row above |
| AC-4b | Met | `ApplyOps`'s Lstat preflight; 5 dedicated tests (`TestApplyOps_LstatPreflightRejects*`, `TestApplyOps_LstatPreflightAllowsMissingCreateTarget`) in `internal/upgrade/replaceplan_test.go:929-1080` |
| AC-4c | Met (with a pre-existing, documented root-CI caveat) | `TestRunUpgradeV2_SettingsWriteFailure_SnapshotNotAdvanced` injects the failure between snapshot-write and settings-write; self-skips under root CI per self-review's positive note — unchanged by this fix commit, not a regression |
| AC-5 | Met | Same idempotency test as NFR-1 |
| AC-6 | Met | `rebuildManifestV2` carries fork/drift/legacy-skipped/preserved entries forward before the desired-state loop (self-review's positive note, unchanged) |
| AC-7 | Met | `TestAddPack_LegacyManifest_FailsClosedZeroWrites` / `RunUpgradeIO_V2Layout_DispatchesToV2Engine` replace the legacy-manifest fail-closed coverage per self-review's deleted-test inventory |
| AC-8 | Met (after M5 fix) | Zero references to `resolveConflict`/`PlanMerge`/`ComputeDiffs`/baseline-writers; the 4 dead manifest members M5 flagged are now deleted, closing the "complete removal" gap self-review raised |
| AC-9 | Met (after M2 fix) | `packEntries` local-map buffering makes preserve-on-failure real; `TestRunUpgradeV2_UnavailablePack_FullyPreserved` + new `TestRunUpgradeV2_PackRuleReadFailure_FullyPreserved` |
| AC-10 | Met | No `--force` flag; `--dry-run`/`--diff` wired to the v2 preview path (`renderUpgradeV2Preview`) |
| AC-11 | Met | `ralph init` (Phase 2, unchanged by this PR) writes the settings snapshot and no baseline; `{}` fallback + report note tested (self-review's test inventory) |
| AC-12 | Met (after M1 fix) | Malformed/missing block source now explicitly left untouched with a report note, closing the silent-blanking gap M1 flagged; `.gitignore`'s pre-existing `hasGitignoreTemplate` guard and AGENTS.md's new `hasAgentsTemplate` guard are now symmetric |

## Static analysis

`./scripts/run-static-verify.sh` (log: `docs/evidence/verify-2026-08-18-043532.log`):

- `scripts/check-sync.sh`: PASS — IDENTICAL 158, DRIFTED 0, ROOT_ONLY 0,
  TEMPLATE_ONLY 11 (all expected generated/seed-only paths, e.g.
  `.ralph/core/settings.ralph.json`, `ralph.toml`, `docs/*/.gitkeep`),
  KNOWN_DIFF 3 (pre-existing, unrelated to this PR)
- `scripts/check-pipeline-sync.sh`: PASS — all 6 referencing docs in sync
- `scripts/check-skill-sync.sh`: PASS — 13 skills in lock-step
- Codex hook guards (single-source, inline-hook-detector smoke test, PR
  provenance policy guard): all PASS
- Go verifier (scope fell back to `full` because
  `templates/base/.ralph/core/settings.ralph.json` is an unclassified
  language for the changed-file scope detector — expected, not a failure):
  `gofmt -l .` → "gofmt: ok"; `go vet ./...` → silent (pass); `golangci-lint
  run ./...` → "0 issues."; `staticcheck` → silent (pass, binary present)
- Overall: `All verifiers passed.`

## Documentation drift

- `AGENTS.md` repo map: updated for `internal/upgrade`'s new v2-engine
  description, `.ralph/local/` bullet's Phase 3 → 5 relabel (part of the
  fix commit), new `.ralph/core/settings.ralph.json` generation source —
  all consistent with the shipped code.
- `README.md`: `ralph upgrade` section fully rewritten for the
  non-interactive engine — no `--force` flag, exit codes 0/1/3 documented,
  `docs/reports/upgrade-<version>-<date>.md` report path stated.
- Spec (`docs/specs/2026-08-17-overlay-scaffold-v2.md`) Open questions: the
  exit-code open question is resolved in place (struck through, replaced
  with "Phase 3 で確定: 成功 = 0 / 実行エラー = 1 / 未解決 drift 残存で完走
  = 3", pointing at this plan's Design decisions section).
- `docs/tech-debt/README.md`: 6 stale rows re-pointed/rewritten (see
  crosscheck table above); 3 new rows from the self-review's own
  recommendation were not added — see Findings.
- No stale "対話"/"conflict resolution"/"--force" references found in
  `docs/quality/` or `docs/recipes/` related to `ralph upgrade`
  (`docs/recipes/worktrees.md`'s `--force-branch` hit is an unrelated
  worktree-cleanup flag).

## Known gaps

- No behavioral test execution was performed here (by design — `/test`'s
  job). Static analysis (`gofmt`, `go vet`, `golangci-lint`, `staticcheck`)
  passing is a proxy for "the new/changed code compiles and passes lint,"
  not "the new tests actually pass and assert what they claim."
- `TestRunUpgradeV2_SettingsWriteFailure_SnapshotNotAdvanced` (AC-4c) is
  known to self-skip on a root-runner CI; whether this environment's CI
  runs as root was not checked here (pre-existing condition, not
  introduced by this fix commit).
- The tech-debt register gap noted above (3 rows from self-review's own
  table not yet appended) is tracked as a LOW finding, not re-verified
  against a follow-up commit since none exists yet at HEAD `3d8461d`.
