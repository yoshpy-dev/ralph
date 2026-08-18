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

## Cycle 2 (final cycle, 2/2)

- Date: 2026-08-18
- Scope: delta since cycle-1 (`c785300..HEAD`, HEAD `4b53880`, working tree
  clean). Commits reviewed: `6b617dc` (tech-debt row for the cycle-1 LOW
  finding), `62631aa`/`7602317` (test/sync-docs artifacts), `df1a529`
  (cross-review triage, 2 ACTION_REQUIRED), `fc8e9a9` (AR fixes:
  `ValidateRealParentChain` + exception-face/report write guards + 7 new
  tests), `cee54b2` (cycle-2 self-review, 4 new LOW findings C2-1..C2-4),
  `78d9039`/`4b53880` (C2-1..C2-4 fixes).
- Verdict: **PASS**. No CRITICAL/HIGH/MEDIUM findings. No static-analysis
  regressions. The one cycle-1 LOW gap (tech-debt register missing 3 rows)
  is now closed.

### Cycle-1 LOW gap resolution

`6b617dc` (committed before the AR fixes, in the same delta) appends the row
the cycle-1 verify finding asked for, naming all three items from the
self-review's "Tech debt identified" table verbatim: `installGitHook`'s
untested user-hook-chaining branch, the untested `--pager` flag, and
`buildDesiredStateV2`'s untested `apErr` branch
(`docs/tech-debt/README.md`, last row). `grep -n
"installGitHook\|pager\|apErr" docs/tech-debt/README.md` now matches. This
closes the cycle-1 verify finding.

### Cross-review AR fix crosscheck (`fc8e9a9` vs. triage table)

| # | Triage finding | Fixed? | Evidence |
| --- | --- | --- | --- |
| AR#1 | `ApplyOps`'s Lstat preflight is leaf-only; a symlinked parent directory with a missing leaf reports `os.ErrNotExist` and MkdirAll/WriteFile writes through the symlink outside `targetDir` | Yes | `internal/upgrade/replaceplan.go`: new `ValidateRealParentChain(targetDir, relPath)` walks every existing path component from `targetDir` to the op's parent, rejecting any non-directory component; called for every op in `ApplyOps`'s validate-all-upfront pass (`replaceplan.go:481-486`), before the existing leaf-Lstat pass. Doc comment above `ApplyOps` rewritten to explain why the leaf check alone is insufficient. Tests: `TestApplyOps_ParentChainPreflightRejectsSymlinkedParent_ZeroWrites`, `TestApplyOps_ParentChainPreflightRejectsDeeplyNestedSymlinkedParent`, plus two positive controls (`_AllowsRealNestedDirectories`, `_AllowsMissingIntermediateDirectories`) that prove the guard doesn't over-reject. |
| AR#2 | `.claude/settings.json` / `.ralph/core/settings.ralph.json` exception-face writes bypass `ApplyOps` via direct `os.WriteFile`, so a symlinked target can write outside `targetDir` | Yes | `internal/cli/upgrade_v2.go`'s `writeFileV2` now calls `upgrade.ValidateRealParentChain` plus an `os.Lstat`-based leaf check (rejects symlink/non-regular, allows absent) before every write — applied to both exception-face writes it serves. Same containment pair was also added to `internal/upgrade/report.go`'s `WriteUpgradeReport` (upgrade report write, which also bypasses `ApplyOps` and was not named in the triage but shares the same gap shape — a reasonable scope extension, not an unrequested change). Tests: `TestRunUpgradeV2_SymlinkedSettingsJSON_RejectedNoEscape`, `TestRunUpgradeV2_SymlinkedSettingsSnapshot_RejectedAfterSettingsJSONWritten`, `TestRunUpgradeV2_SymlinkedReportsDir_RejectedNoEscape`. |

All 7 new tests claimed by the commit message (`internal/upgrade/replaceplan_test.go`
+4, `internal/cli/upgrade_v2_test.go` +3) were confirmed present by name via
`grep -n "^func Test"`.

### Cycle-2 self-review fix crosscheck (`78d9039`/`4b53880` vs. C2-1..C2-4)

| ID | Finding | Fixed? | Evidence |
| --- | --- | --- | --- |
| C2-1 | Tech-debt row's `(M3, row reconciliation)` pointer resolves to the wrong finding ID (should be `M4`) | Yes | `docs/tech-debt/README.md:102` now reads `(M4, row reconciliation)` (`78d9039`). |
| C2-2 | `writeFileV2`'s doc comment said "both" exception-face writes when `v2SkipPaths` names four | Yes | Doc comment reworded to "two of the four v2 exception faces," names the other two (AGENTS.md/.gitignore via `updateOneBlockV2`), and explains why they don't need `ValidateRealParentChain` (root-level targets, own leaf guard) (`78d9039`, `internal/cli/upgrade_v2.go:562-568`). |
| C2-3 | Two new tests named `_RejectedZeroWrites` for runs that are not zero-write | Yes | Both renamed to `_RejectedNoEscape` (`78d9039`, `internal/cli/upgrade_v2_test.go`). Confirmed old names are gone (`grep -n "SymlinkedSettingsJSON_RejectedZeroWrites\|SymlinkedReportsDir_RejectedZeroWrites"` → no matches) and new names present. |
| C2-4 | Cycle-1 `L1` (`_ = in`, unused `io.Reader` param) reached neither a fix nor the tech-debt register | Yes | `4b53880` appends a 6th item to the existing 5-LOW register row, naming the parameter, its file, and its ~25 call sites; matches the self-review's "add one line to the existing row" disposition exactly. |

### Static analysis (re-run at cycle-2 HEAD)

`./scripts/run-static-verify.sh` (log: `docs/evidence/verify-2026-08-18-051933.log`):

- `scripts/check-sync.sh`: PASS — IDENTICAL 158, DRIFTED 0, ROOT_ONLY 0,
  TEMPLATE_ONLY 11, KNOWN_DIFF 3 (same counts as cycle 1, no regression)
- `scripts/check-pipeline-sync.sh`: PASS — all 6 referencing docs in sync
- `scripts/check-skill-sync.sh`: PASS — 13 skills in lock-step
- Codex hook guards: all PASS
- Go verifier (scope fell back to `full`, same unclassified-file reason as
  cycle 1): `gofmt: ok`; `go vet` silent (pass); `golangci-lint run` → "0
  issues."; `staticcheck` silent (pass)
- Overall: `All verifiers passed.`

### Plan acceptance criteria — checkbox adjudication

All 12 plan ACs (`AC-1`..`AC-12`) were adjudicated "Met" (or "Met, with a
documented caveat" for AC-4c) in the cycle-1 FR/NFR and AC-mapping tables
above, and nothing in the cycle-2 delta regresses any of them — the AR fixes
strengthen AC-4b (parent-chain containment, beyond the plan's original
leaf-only wording) and AC-9 (extends the same containment guarantee to the
settings/snapshot/report exception-face writes) without changing their
pass/fail status. Ticked all 12 checkboxes in
`docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md`, adding inline notes
on AC-4b and AC-9 for the cycle-2 strengthening and on AC-4c for its
pre-existing root-CI self-skip caveat (this is a verifier judgment call per
the standing instruction to adjudicate and tick ACs that hold — the "Plan
reviewed" and "PR created" progress-checklist items are left untouched, as
they are outside verify's scope).

### Known gaps (cycle 2)

- No behavioral test execution was performed here (by design — `/test`'s
  job); `/test` already produced `docs/reports/test-2026-08-18-overlay-scaffold-v2-p3.md`
  at cycle 1 but has not re-run against the cycle-2 delta as of this verify
  pass.
- No new tech-debt or doc-drift gaps were found in the cycle-2 delta.

## Cycle 3 (final cycle, 3/3 — after cap raise)

- Date: 2026-08-18
- Scope: delta since cycle-2 (`aa7fb9c..HEAD`, HEAD `6533227`, working tree
  clean). Commits reviewed: `e21b1b1`/`b80b5f9` (cycle-2 test/sync-docs
  artifacts), `1bb8367` (cycle-2 cross-review triage, 2 ACTION_REQUIRED),
  `41ad745` (AR fixes: dry-run exception-face preview + no-op short-circuit
  + spec NFR-1 amendment), `a0b9363` (cycle-3 self-review, 1 HIGH + 1 MEDIUM
  + 3 LOW: C3-1..C3-5), `6533227` (C3 fixes: seed-advisory/version vetoes on
  the no-op short-circuit + LOW cleanups).
- Verdict: **PASS**. No CRITICAL/HIGH/MEDIUM findings against this delta. No
  static-analysis regressions. One new LOW doc-completeness gap (below),
  non-blocking.

### Cross-review AR fix crosscheck (`41ad745` vs. cycle-2 triage table)

| # | Triage finding | Fixed? | Evidence |
| --- | --- | --- | --- |
| AR#1 (cycle 2) | `--dry-run` doesn't preview pending block/settings ("exception face") changes — a run that would rewrite AGENTS.md/.gitignore/settings.json on a real execution can show "no changes" under `--dry-run` | Yes | `renderUpgradeV2Preview` now calls new `renderUpgradeV2ExceptionFacePreview` (`internal/cli/upgrade_v2.go:767-770`), which computes the same block/settings outcomes purely in memory (`applyBlockUpdatesV2(..., write=false)`, `MergeOwnedSettings` is pure) without ever touching disk. `applyBlockUpdatesV2`/`updateOneBlockV2` gained a `write bool` parameter threaded through every call site. |
| AR#2 (cycle 2) | A converged re-run at the same version still writes a dated report + rewrites the manifest, violating NFR-1's "no-op re-run = zero writes" | Yes | New `isFullyConvergedV2` gate (`upgrade_v2.go:157` at the time, now `oldManifest.Meta.Version == Version && isFullyConvergedV2(...)` after the C3-2 fix) short-circuits to `finishNoOpUpgradeV2`, which skips the report, the manifest rebuild, and `installManagedGitHooks`, printing `"Upgrade no-op: ... (already up to date, zero writes)"` instead. Spec `docs/specs/2026-08-17-overlay-scaffold-v2.md` NFR-1 amended in the same commit to state the no-op is announced on stdout and that a converged re-run does not write a dated report file at all — resolving the triage's noted tension between "no-op" and "record it in the report" by choosing stdout as the recording surface. |

Both fixes are exactly what the triage rationale (`docs/reports/cross-review-triage-overlay-scaffold-v2-p3.md`, ACTION_REQUIRED #1/#2) asked for, including the file:line pointers named there (`internal/cli/upgrade_v2.go:95-96` → dry-run path; `:172-177` → no-op-skip path — both areas are inside the functions touched by this commit).

### Cycle-3 self-review fix crosscheck (`6533227` vs. C3-1..C3-5)

`a0b9363` (cycle-3 self-review) found that `41ad745`'s own no-op short-circuit gated more than it should have: `isFullyConvergedV2` never consulted `plan.Advisories`, so a run that only had a pending **seed** advisory (no core ops, no block/settings changes) was misclassified as a full no-op — silently and permanently swallowing the advisory (a seed advisory only clears via the manifest rebuild the no-op path skips) and, separately, never advancing `meta.version` on a Go-only release with byte-identical templates.

| ID | Finding | Fixed? | Evidence |
| --- | --- | --- | --- |
| C3-1 (HIGH) | `isFullyConvergedV2` ignores `plan.Advisories`; a pending seed advisory gets silently and permanently swallowed by the no-op short-circuit (report never written, `TemplateHash` never advanced) | Yes | `isFullyConvergedV2` now loops `plan.Advisories` and returns `false` on any `scaffold.OwnerSeed` entry (`internal/cli/upgrade_v2.go`, +4 lines); doc comment extended to explain fork advisories are correctly excluded (they never clear) while seed advisories must not be. New test `TestRunUpgradeV2_SeedAdvisoryOnly_NotANoOp`: asserts stdout does not say "no-op", the report is written and names `docs/notes.md`, the manifest's `TemplateHash` advances to clear the advisory, and a follow-up run at the same state is a genuine no-op. Confirmed by reading the assertions against the pre-fix predicate: without the `plan.Advisories` loop, `isFullyConvergedV2` would still return `true` for this scenario (zero `Ops`/`ManifestRefresh`/`ManifestRemove`, unchanged blocks/settings), so the test would fail on every assertion against the pre-fix code — i.e. the test is falsifiable against the bug it names, not tautological. |
| C3-2 (MEDIUM) | `meta.version` never advances on a converged run because the short-circuit also skips `scaffold.NewManifest(version)`, permanently desyncing `ralph doctor`'s version check from a no-op-reporting `ralph upgrade` | Yes | Gate changed to `oldManifest.Meta.Version == Version && isFullyConvergedV2(...)` (`internal/cli/upgrade_v2.go:157`). New test `TestRunUpgradeV2_ConvergedButVersionBumped_NotANoOp`: templates held byte-identical to gen1, `Version` bumped; asserts stdout does not say "no-op" and the manifest's `Meta.Version` is rewritten to the new value, then a follow-up run at the caught-up version is a genuine no-op. Same falsifiability check: without the `Meta.Version ==` conjunct, the pre-fix predicate returns `true` here too, so the test's first two assertions fail against the unfixed gate. |
| C3-3 (LOW) | Plan's AC-5 parenthetical ("レポートに no-op 明記") still contradicted the spec's NFR-1 as amended by `41ad745` ("標準出力に no-op を明記... レポートファイル自体を書かない") | Yes | `docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md` AC-5 now reads "同一バイナリでの再実行が書き込みゼロの no-op(標準出力に no-op 明記、レポートファイルは書かない)" — verbatim-consistent with the spec's amended NFR-1 wording (`docs/specs/2026-08-17-overlay-scaffold-v2.md:71`). |
| C3-4 (LOW) | Insight event appended by `1bb8367` (subject line says "for cycle 2") recorded `"cycle":1` for the `cross_review` phase | Yes | `docs/insights/events/2026-08-18-overlay-scaffold-v2-p3.jsonl` line for `ts":"2026-08-18T05:41:33Z"` now reads `"cycle":2`. Replayed the full event sequence: `self_review` c1→c2→c3, `verify` c1→c2, `test` c1→c2, `sync_docs` c1→c2, `cross_review` c1→c2 — all monotonic per phase, no other mis-stamps found in this delta. |
| C3-5 (LOW) | `AR#1`/`AR#2` now label two different pairs of findings against the same cited triage report across cycles; plan's AC-4b/AC-9 notes (cycle-1 fixes, `fc8e9a9`) and the code comments touched by `41ad745`/`6533227` (cycle-2 fixes) both say bare "AR#1"/"AR#2" | **Partially.** The code comments in `internal/cli/upgrade_v2.go` were qualified (`"AR#2"` → `"AR#2, cycle 2"`, `"AR#1"` → `"AR#1 (cycle 2)"` at 5 sites — confirmed via `git show 6533227 -- internal/cli/upgrade_v2.go`). The plan's own AC-4b/AC-9 notes, however, were **not** touched by `6533227` and still read bare `cross-review AR#1 の fix、fc8e9a9` / `cross-review AR#2 の fix、fc8e9a9` (`docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md:87,93`) — the recommendation's first clause ("qualify the plan's two references as 'cycle-1 cross-review AR#1/AR#2'") was not applied. See Findings below. |

4 of 5 cycle-3 findings (including the HIGH and the MEDIUM) are fully closed with falsifiable regression tests. The one gap (C3-5's plan-side half) is a LOW doc-precision miss, not a fresh defect — flagged below.

### Findings (cycle 3)

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| LOW | doc drift | C3-5's recommendation had two clauses: (a) qualify the plan's AC-4b/AC-9 `AR#1`/`AR#2` references as "cycle-1 cross-review", and (b) have the next `/cross-review` pass label its own triage rows explicitly. `6533227` implements neither (b), which is expected — cycle-3 `/cross-review` has not run yet at this HEAD — nor (a), which was actionable now and wasn't done. `docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md:87,93` still cite bare `AR#1`/`AR#2` against a triage report (`docs/reports/cross-review-triage-overlay-scaffold-v2-p3.md`) whose rows were overwritten in cycle 2, so a reader resolving the plan's citation today lands on the wrong (cycle-2) findings. | `docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md:87,93`; `docs/reports/self-review-2026-08-18-overlay-scaffold-v2-p3.md` C3-5 recommendation; `git show 6533227 -- docs/plans/active/2026-08-18-overlay-scaffold-v2-p3.md` (only the AC-5 line changed) | One-line edit to each of AC-4b and AC-9: prefix with "cycle-1 cross-review AR#1/AR#2" (or equivalent). Small enough to fold into a `/pr` hand-off note or a trivial follow-up commit; not worth another fix-and-revalidate cycle at the cap. |

No CRITICAL, HIGH, or MEDIUM findings from this cycle-3 verify pass.

### AC-5 / AC-10 under the new no-op semantics

- **AC-5** (idempotency): plan text now reads "同一バイナリでの再実行が書き込みゼロの no-op(標準出力に no-op 明記、レポートファイルは書かない)" — matches the shipped behavior exactly: `finishNoOpUpgradeV2` writes to `out` only (`"Upgrade no-op: %s (already up to date, zero writes)\n"`), never calls `WriteUpgradeReport`, and the C3-1/C3-2 vetoes ensure the short-circuit only fires when there is truly nothing pending (no ops, no advisories of the kind that would otherwise clear, same binary version). `TestRunUpgradeV2_Idempotent_SecondRunIsNoOp` (unchanged since cycle 1) plus the two new cycle-3 tests jointly cover the no-op path's positive and negative cases.
- **AC-10** (no `--force`; `--dry-run`/`--diff` work, dry-run is zero-write): `grep -n force internal/cli/upgrade.go internal/cli/upgrade_v2.go` still returns no matches (unchanged from cycle 1). `--dry-run` now additionally previews the four exception faces via `renderUpgradeV2ExceptionFacePreview`, which is explicitly documented as never writing to `targetDir` (`applyBlockUpdatesV2` called with `write=false`, `MergeOwnedSettings` is pure) — the dry-run-is-zero-write half of AC-10 holds under the widened preview, not just the original core-plan-only preview.

Both ACs hold under the cycle-3 semantics change; no regression against their cycle-1/cycle-2 adjudication.

### Spec NFR-1 amendment consistency

`41ad745` amended `docs/specs/2026-08-17-overlay-scaffold-v2.md` NFR-1 from "書き込みゼロ、レポートに no-op 明記" to "書き込みゼロ、標準出力に no-op を明記。収束済みの再実行では日付付きレポートファイル自体を書かない — レポートは新たな適用結果があった実行でのみ生成される". Cross-checked against the implementation:

- "標準出力に no-op を明記" → `finishNoOpUpgradeV2`'s `writef(out, "Upgrade no-op: %s (already up to date, zero writes)\n", version)`. Confirmed.
- "収束済みの再実行では日付付きレポートファイル自体を書かない" → `finishNoOpUpgradeV2` returns before reaching `upgrade.WriteUpgradeReport` (that call only exists on the non-no-op path in `runUpgradeV2`). Confirmed.
- "レポートは新たな適用結果があった実行でのみ生成される" → the only `WriteUpgradeReport` call site in `runUpgradeV2` is reached exclusively past the `isFullyConvergedV2` gate. Confirmed.

The plan's AC-5 (see above) was brought into line with this same wording by the cycle-3 fix commit. Spec and implementation agree; no drift.

### Static analysis (re-run at cycle-3 HEAD)

`./scripts/run-static-verify.sh` (log: `docs/evidence/verify-2026-08-18-061413.log`):

- `scripts/check-sync.sh`: PASS — IDENTICAL 158, DRIFTED 0, ROOT_ONLY 0,
  TEMPLATE_ONLY 11, KNOWN_DIFF 3 (same counts as cycles 1 and 2, no
  regression)
- `scripts/check-pipeline-sync.sh`: PASS — all 6 referencing docs in sync
- `scripts/check-skill-sync.sh`: PASS — 13 skills in lock-step
- Codex hook guards (single-source, inline-hook-detector smoke test, PR
  provenance policy guard): all PASS
- Go verifier (scope fell back to `full`, same unclassified-file reason as
  cycles 1 and 2 — `templates/base/.ralph/core/settings.ralph.json`):
  `gofmt -l .` → "gofmt: ok"; `go vet ./...` → silent (pass); `golangci-lint
  run ./...` → "0 issues."; `staticcheck` → silent (pass, binary present)
- Overall: `All verifiers passed.`

### Plan coherence

- Objective, Scope, Non-goals, Design decisions, and Risks sections are
  unchanged from cycle 1/2 and remain internally consistent with the
  shipped code (re-read in full at this HEAD).
- Progress checklist: `Plan reviewed` and `PR created` remain unchecked as
  expected pre-`/pr` state (outside verify's scope, per the standing
  cycle-2 convention). All artifact-creation checkboxes (`Review`,
  `Verification`, `Test`, `Sync-docs`) are checked and match the actual
  reports present under `docs/reports/`.
- Deviations section still accurately describes the one implementation
  deviation (missing `installManagedGitHooks` call, closed in slice 4); no
  new deviation was introduced by the cycle-2/cycle-3 fix commits that
  warrants a Deviations-section entry (the AR/self-review fixes are
  documented in-place via AC notes and code comments instead, consistent
  with how cycle-2's fixes were recorded).

### Known gaps (cycle 3)

- No behavioral test execution was performed here (by design — `/test`'s
  job); `/test` has not yet re-run against the cycle-3 delta as of this
  verify pass.
- The C3-5 plan-reference gap (above) is tracked as a LOW verify finding
  here rather than fixed, since the pipeline cycle cap (raised to 3) means
  this is the final scheduled verify pass; recommend closing it as a
  trivial follow-up rather than triggering another fix-and-revalidate
  cycle.
- Cycle-3 `/cross-review` has not yet run against this delta; its own
  triage — if it produces new ACTION_REQUIRED findings — would fall outside
  the pipeline cap per `.claude/rules/ralph/post-implementation-pipeline.md`'s
  cap-reached branching (raise cap / proceed to `/pr` with known gaps /
  abort).
