# Verify — overlay-scaffold-v2-p5

- Date: 2026-08-19
- Plan: `docs/plans/active/2026-08-19-overlay-scaffold-v2-p5.md`
- Spec: `docs/specs/2026-08-17-overlay-scaffold-v2.md`
- Cycle: 1
- Scope: `git diff main...HEAD` — commits `9bf09e2` (eject/adopt), `b884a38` (doctor --strict), `a61dbb5` (status), `742260f` (purity guard + Codex hooks parity + pre_bash_guard fix), `a164b2c` (docs alignment + leak fixes), `7cf4f47` (self-review C1 fixes: H1, M1–M8, L1–L4), plus plan-bookkeeping commits
- Dimension: spec compliance + static analysis. No behavioral test execution (tester's job).
- Verdict: **pass**

## Self-review C1 fix confirmation

Before auditing the plan's ACs, confirmed commit `7cf4f47` genuinely resolves every finding in `docs/reports/self-review-2026-08-19-overlay-scaffold-v2-p5.md` (1 HIGH, 8 MEDIUM, 4 LOW):

| Finding | Fix verified |
|---|---|
| H1 (dead regex branch) | `REGEX_PATTERNS`/`REGEX_REASONS` split into parallel arrays (`scripts/check-template-purity.sh:53-64`); `grep_scan_or_fail` hard-fails on grep exit >1 instead of swallowing it. New fixture cases E/F (`tests/test-template-purity.sh`) positively exercise the `docs/reports\|plans` alternation. Ran `bash tests/test-template-purity.sh`: 7/7 pass, including E and F. |
| M1 (doctor (a) vs (b)/(c) asymmetry) | `--strict` flag help (`internal/cli/doctor.go:38-46`), README's doctor row, and doc comments on `checkManagedBlocks`/`checkSettingsOwnedKeys` now state the asymmetry is intentional. |
| M2 (corrupt manifest fails open) | `checkScaffoldIntegrity` distinguishes `errors.Is(err, fs.ErrNotExist)` (nil, correct) from any other parse error (now `scaffoldViolationStatus`-eligible). `TestCheckScaffoldIntegrity_CorruptManifest_StrictFails` pins strict=fail / non-strict=warn plus an integration exit-code assertion. |
| M3 (untested preflight guarantee) | `TestRunAdoptIO_PreflightFailure_AllBatchZeroWrites` restructured to corrupt the later-sorting target (`scripts/`) and modify+eject the earlier-sorting one (`packs/languages/golang/verify.sh`) first, so the untouched-file assertion now actually distinguishes preflight-first from write-as-you-go. |
| M4 (wrong confirm prompt) | `confirmDestructiveOp(in, out, autoYes, opName)` extracted; `runAdoptIO` now calls it with `"adopt"` (`internal/cli/adopt.go:147`); `confirmMigration` is a thin wrapper preserving `"migration"`. |
| M5 (unfiled KNOWN_DIFFS) | New tech-debt row added for the two `quality-gates.md` / `adding-a-language-pack.md` whole-file suppressions, with a concrete removal trigger. |
| M6 (guarded substring, not the goal) | `templates/base/docs/recipes/adding-a-language-pack.md` fully rewritten for the downstream reader (`ralph pack add/list`, rule ownership via eject, no mention of `check-sync.sh`, `new-language-pack.sh`, `packs/languages/`, or `templates/packs/`). |
| M7 (status hard-fails on scaffold error) | `runStatus` now degrades to `scaffoldStatus{Err: ...}` and prints `"Scaffold ownership: unavailable (...)"` instead of aborting; `TestStatusCmd_ScaffoldSection_ComputationError_DegradesGracefully` pins it; JSON shape is additive (`Error` field, `omitempty`). |
| M8 (docs overclaim Codex verified) | README's Hooks section now reads "...though live firing under Codex has not yet been confirmed on a trusted checkout (see `docs/tech-debt/README.md`)". |
| L1 (wrong assertion count) | tech-debt row 100 now says "both the jq and sed-fallback paths, deny and allow, against the real nested payload shape" instead of a stale count. |
| L2 (stale allowlist comments) | Both the script header and the test file comments now describe the current empty-allowlist state. |
| L3 (narrow hook-command extraction) | `tests/test-hook-wiring.sh` now matches any quoted string ending in `.sh` (with or without leading `./`), not just a literal `./`-prefixed string. |
| L4 (predictable /tmp path) | `tests/test-template-purity.sh` now uses `mktemp`. |

All fixes verified by reading the diff and, where a runnable fixture exists, by executing it directly (purity guard: 7/7 pass; `go build ./...`: clean).

## Per-AC verification

| AC | Status | Evidence |
|---|---|---|
| AC-1 (eject: core→fork, zero disk writes) | Pass | `internal/cli/eject.go`; `TestRunEjectIO_CoreClean_ForksRecorded` (`internal/cli/eject_adopt_test.go:30`) |
| AC-2 (eject on unresolved drift → fork advisory) | Pass | `TestRunEjectIO_DriftedCore_ForksRecordedWithDriftedHash` (`:73`) |
| AC-3 (eject/adopt error matrix incl. v2 exception faces) | Pass | `TestRunEjectIO_ErrorMatrix` (`:102`) enumerates untracked / already-fork / already-core / owner=seed·block / AGENTS.md / .gitignore ("v2 exception face") / legacy manifest / disk-missing cases, each asserting the manifest is unchanged |
| AC-4 (adopt: fork→core, disk replaced with template) | Pass | `TestRunAdoptIO_SingleFork_ResetToTemplate` (`:228`) |
| AC-5 (adopt on unresolved drift) | Pass | `TestRunAdoptIO_DriftedCore_ResetToTemplate` (`:265`) |
| AC-6 (adopt safety: git-clean precondition, y/N confirm, preflight before any write) | Pass | `TestRunAdoptIO_DirtyGitTree_ZeroWrites` (`:337`), `TestRunAdoptIO_ConfirmationDeclined_ZeroWrites` (`:493`), `TestRunAdoptIO_RetiredFork` (`:290`, rejects adopt of a template-retired path) |
| AC-7 (adopt --all: batch preflight, zero-target no-op, partial-failure guarantee) | Pass | `TestRunAdoptIO_All_ZeroTargets_NoOp` (`:465`), `TestRunAdoptIO_PreflightFailure_AllBatchZeroWrites` (`:402`, restructured post-M3 to actually pin preflight-first semantics) |
| AC-8 (doctor --strict: 5 FR-9 checks, exit 1 on violation, exit 0 warn-only without --strict) | Pass | `internal/cli/doctor.go` — `checkScaffoldCoreHashes` (a), `checkManagedBlocks` (b), `checkSettingsOwnedKeys` (c), `checkConflictMarkers` (d, `:899`), `checkManifestConsistency` (e, `:929`); `scaffoldViolationStatus` is the single fail/warn switch; `TestRunDoctorFull_StrictFlipsExitCode_DriftedCore` (`internal/cli/doctor_scaffold_test.go:132`) |
| AC-9 (doctor --strict green on fresh init / converged upgrade; ejected fork not a violation) | Pass | `TestCheckScaffoldIntegrity_FreshInit_AllPass` (`:59`), `TestCheckScaffoldIntegrity_ConvergedUpgrade_AllPass` (`:69`), `TestCheckScaffoldIntegrity_EjectedFork_CoreHashesPass` (`:90`) |
| AC-10 (status: per-path ownership + drift, text/--json matrix, org output unchanged) | Pass | `internal/cli/status_scaffold_test.go` — 7 tests covering v2/no-issues, fork+drift, legacy, no-manifest, empty-org-roster, computation-error-degrades, org/scaffold-flag-scoping |
| AC-11 (purity guard detects leaks, exit 1; green on current templates/; CI-wired) | Pass | `bash scripts/check-template-purity.sh` on real `templates/` → exit 0; `bash tests/test-template-purity.sh` → 7/7 pass (cases A–F); wired into `scripts/verify.local.sh:run_static_checks` (`:204-206`), invoked by `./scripts/run-verify.sh` (`CI: .github/workflows/verify.yml` → "Run verification" step) |
| AC-12 (Codex hooks route through dispatcher, root/template byte-identical, direct-call detection) | Pass | `cmp .codex/config.toml templates/base/.codex/config.toml` → identical; `bash scripts/check-sync.sh` → 0 DRIFTED; `tests/test-hook-wiring.sh` broadened extraction (L3 fix) |
| AC-12b (pre_bash_guard jq path reads nested `.tool_input.command`) | Pass | `.claude/hooks/pre_bash_guard.sh` calls `extract_json_field "$payload" "tool_input.command"`; self-review security probe independently re-ran the real hook against a real-shape payload with jq present and confirmed `deny` |
| AC-13 (eject→adopt round trip converges) | Pass | `TestEjectAdoptRoundTrip_AC13` (`:556`) |
| AC-14 (spec/tech-debt/README/AGENTS.md alignment) | Pass, with one judgment item recorded below | FR-2/3/9/10/12/13 spec checkboxes ticked; tech-debt rows 100/103 struck RESOLVED; README/AGENTS.md updated (see spot-checks below) |
| AC-15 (`run-verify.sh` exit 0, shell + Go green) | Pass (static portion; behavioral is tester's scope) | `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh` → exit 0 (see Static analysis below); `go build ./...` → clean |

## Spec compliance (FR-2/3/9/10/12)

- **FR-2 (eject)**: wording matches implementation — core/undecided-drift path → fork, manifest records `owner=fork` + `forked_from_version` + disk hash, zero disk writes. Confirmed via `internal/cli/eject.go` and the AC-1/AC-2 tests above.
- **FR-3 (adopt, "唯一の再採用経路")**: confirmed no `--force` flag exists on `adopt`, `eject`, or `upgrade`. `grep -rn 'force' internal/cli/*.go` finds only `ralph init --force` (an unrelated, pre-existing overwrite flag for scaffolding a fresh directory) — no `--force` flag survives on the destructive-reset path the spec describes as removed. `adopt` exposes only `--yes` for confirmation skip.
- **FR-4 (drift resolution via eject/adopt)**: both eject and adopt operate on fork-record-less drift (AC-2, AC-5), matching the spec's "eject（改変維持）または adopt（改変破棄）で解消" language.
- **FR-9 (doctor --strict (a)–(e))**: all five sub-checks present and independently named in code (`checkScaffoldCoreHashes`/`checkManagedBlocks`/`checkSettingsOwnedKeys`/`checkConflictMarkers`/`checkManifestConsistency`); exit semantics traced: any `Status == "fail"` increments `countFailed` and `runDoctorFull` returns a non-nil error → non-zero process exit; `--strict` is the single switch (`scaffoldViolationStatus`) between "fail" and "warn" for all five.
- **FR-10 (purity guard wired into CI)**: traced the full chain — `.github/workflows/verify.yml` "Run verification" step → `./scripts/run-verify.sh` (`RALPH_VERIFY_SCOPE=full` in CI) → `./scripts/verify.local.sh` (`scripts/run-verify.sh:48-51`) → `run_static_checks()` → `scripts/check-template-purity.sh` (`scripts/verify.local.sh:204-206`, gated on `[ -x scripts/check-template-purity.sh ]`). Confirmed the guard actually runs and passes as part of `RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh`.
- **FR-12 (status ownership)**: `ralph status` gained the scaffold-ownership section (`internal/cli/status.go`), reusing `resolveOwnershipPlan` shared with eject/adopt/doctor so classification cannot diverge across commands (confirmed by self-review's own positive-notes cross-check, re-verified by reading `printScaffoldSection`/`buildStatusScaffoldJSON`).

### FR-13 / NFR-1 / NFR-5 spot-checks (requested — confirm still holds at HEAD)

- **FR-13 (interactive conflict-resolution removal)**: `grep -rn "resolveConflict\|resolveConflictWithBaseline"` across `internal/` returns no function definitions — only historical/legacy references to the retired `.ralph/baseline/` directory in comments and its one-time cleanup path (`removeLegacyBaselineIfPresent`, `internal/cli/upgrade_v2.go:311`). No interactive prompt or conflict-marker-editing code path exists. Holds.
- **NFR-1 (idempotent no-op)**: `internal/upgrade/` has zero changes in this branch (`git diff main...HEAD --stat -- internal/upgrade/` is empty) — Phase 5 never touched the upgrade engine. `finishNoOpUpgradeV2` (`internal/cli/upgrade_v2.go:293`) still writes zero files and prints `"Upgrade no-op: ... (already up to date, zero writes)"` with no dated report. Holds unchanged from Phase 3/4's verified state.
- **NFR-5 (Codex parity / sync gates)**: `bash scripts/check-sync.sh` → `PASS: all files in sync` (156 IDENTICAL, 0 DRIFTED, 5 KNOWN_DIFF, 11 TEMPLATE_ONLY); `check-skill-sync.sh` → `13 skill(s) in lock-step`. Both gates green at HEAD. Holds.

## Judgment items (recorded for the orchestrator; spec file not edited)

### (a) Spec's own AC-1..AC-12 checkboxes (docs/specs/2026-08-17-overlay-scaffold-v2.md:83-94)

All 12 are currently unticked (`- [ ]`), distinct from the FR/NFR checkboxes above them (which are ticked). Walked each against the current implementation:

| Spec AC | Satisfied at HEAD? |
|---|---|
| AC-1 (upgrade touches only machine-owned content) | Yes — Phase 1–3, unchanged this phase |
| AC-2 (AGENTS.md block-only update) | Yes — Phase 1/2 |
| AC-3 (settings.json 3-way merge preserves user entries) | Yes — Phase 2/3 |
| AC-4 (unresolved drift reported, `doctor --strict` exit 1) | Yes — drift reporting (Phase 3) + `doctor --strict` (this phase) now close the loop end-to-end |
| AC-5 (ejected core not replaced, advisory diff reported) | Yes — this phase's eject + existing `classifyFork` advisory reporting |
| AC-6 (legacy migration: fork preserved, unmodified replaced, v3 manifest) | Yes — Phase 4 |
| AC-7 (interrupted migration/upgrade resumable, version doesn't advance early) | Yes — Phase 3/4 commit-barrier design |
| AC-8 (same-version re-upgrade no-op) | Yes — NFR-1 spot-check above |
| AC-9 (fresh init matches migrated layout, `doctor --strict` exit 0) | Yes — `TestCheckScaffoldIntegrity_FreshInit_AllPass` |
| AC-10 (`.ralph/local/hooks/PostToolUse.d/` drop-in runs after core hook) | Yes for Claude Code (dispatcher wiring, tested); Codex wiring is structurally routed through the same dispatcher but **live firing is unverified** (recorded as its own tech-debt row, and M8's fix already tempers the README's claim about it) |
| AC-11 (purity guard detects injected leak, exit 1) | Yes — case B/D/E/F |
| AC-12 (no interactive/conflict-marker/baseline code paths) | Yes — FR-13 spot-check above |

**Recommendation**: AC-1 through AC-9, AC-11, and AC-12 are fully satisfied and can be ticked. AC-10 should be ticked with a footnote (or left unticked with a one-line note) pointing at the Codex-live-firing tech-debt row, since the "when 対象イベントが発火する" clause is confirmed for Claude Code but not empirically confirmed for Codex — the honest state is "wired, not yet observed firing under Codex." This is a documentation-completeness judgment, not a code defect; I have not edited the spec file myself.

### (b) `templates/base/docs/quality/quality-gates.md` cites scripts not invoked by the template's own CI

Confirmed: `templates/base/docs/quality/quality-gates.md`'s "Must pass in CI before merge" section lists `./scripts/check-coverage.sh` and `./scripts/check-pipeline-sync.sh`, both attributed to `` (`.github/workflows/verify.yml`) ``. But `templates/base/.github/workflows/verify.yml` only runs `secret-scan.sh`, `check-template.sh`, `run-verify.sh`, and `check-skill-sync.sh` — it does not invoke `check-coverage.sh` or `check-pipeline-sync.sh` at all (confirmed by `diff .github/workflows/verify.yml templates/base/.github/workflows/verify.yml`).

Traced when this attribution was written: `git log main.. -- templates/base/.github/workflows/verify.yml` is empty (the template's workflow file is untouched on this branch), and `git show a164b2c^:templates/base/docs/quality/quality-gates.md` shows the same `check-coverage.sh`/`check-pipeline-sync.sh` → `verify.yml` attribution already present **before** this phase's Slice-5 docs commit. `a164b2c` (Slice 5) only rewrote the two lines it needed to fix the purity-guard leak (`check-template.sh`/`check-sync.sh`, which referenced the meta-repo-only `check-template.yml` workflow) and did not touch the pre-existing `check-coverage.sh`/`check-pipeline-sync.sh` lines.

**Confirmed**: pre-existing mismatch, not introduced by Phase 5, and not in Phase 5's scope (`templates/base/docs/quality/quality-gates.md` is not listed in the plan's Affected areas beyond the purity-leak line-level fix). **Recommendation**: file a tech-debt row rather than fixing it in this cycle — either wire `check-coverage.sh`/`check-pipeline-sync.sh` into `templates/base/.github/workflows/verify.yml`, or correct the doc to describe what the template's CI actually runs.

## Static analysis

```
$ RALPH_VERIFY_SCOPE=full ./scripts/run-static-verify.sh
...
==> scripts/check-sync.sh
  IDENTICAL:      156
  DRIFTED:        0
  ROOT_ONLY:      0
  TEMPLATE_ONLY:  11
  KNOWN_DIFF:     5
PASS: all files in sync.
==> scripts/check-pipeline-sync.sh
[ok] Canonical source valid (6 reference files all pipeline-step-consistent)
==> scripts/check-skill-sync.sh
[ok] check-skill-sync: 13 skill(s) in lock-step
==> scripts/check-template-purity.sh
PASS: no meta-repo-specific references found in templates.
==> Language scope: full
==> Language packs selected: golang
gofmt: ok
0 issues.
==> All verifiers passed.
Exit code: 0
Evidence saved to: docs/evidence/verify-2026-08-19-033947.log
```

Additional targeted checks run directly:
- `go build ./...` → clean, no errors
- `bash tests/test-template-purity.sh` → `PASS: 7, FAIL: 0` (cases A–F, including the H1 regression guards E/F)
- `staticcheck ./...` → silent (no output = clean; per prior verifier convention `staticcheck` prints nothing on a clean pass, unlike `golangci-lint`'s `"0 issues."`)

## Known gaps

- Behavioral test execution is out of scope for `/verify` (tester's responsibility — `./scripts/run-test.sh`).
- Codex project-scoped `[[hooks.*]]` live firing under a trusted checkout remains unverified at runtime (pre-existing open item, tracked in `docs/tech-debt/README.md`, referenced by judgment item (a) above for spec AC-10).
- The two judgment items above are recorded for the orchestrator's disposition; the spec file itself was not modified by this verify pass.

# Cycle 2

- Date: 2026-08-19
- Cycle: 2 (final under `RALPH_STANDARD_MAX_PIPELINE_CYCLES=2`)
- Scope: cycle-2 delta since the cycle-1 verify (33c1166) — `b9a956f` (drift guidance → eject/adopt), `348801c` (cross-review c1 triage), `e01939e` (AR#1/AR#2 fixes), `2bc988f` (self-review c2 section), `e39d436` (C2-M1/M2/L1-L3 fixes + C2-L4/L5 tech-debt row), plus report/insight commits.
- Verdict: **pass**

## 1. AR#1/AR#2 fixes vs. triage contract

Both fixes read from `e01939e`'s diff against the triage's stated remediation, plus their regression tests.

- **AR#1** (`internal/cli/doctor.go:739-746`): `checkScaffoldIntegrity`'s `resolveOwnershipPlan` error path changed from a hardcoded `Status: "warn"` to `Status: scaffoldViolationStatus(true, strict)` — a planning-error scaffold is now a strict-eligible violation, matching the triage's "検査不能=違反として扱うべき" remediation exactly. Pinned by `TestCheckScaffoldIntegrity_OwnershipPlanningError_StrictFails` (`internal/cli/doctor_scaffold_test.go`), which replaces `scripts/run-verify.sh` with a directory (a real `PlanCoreReplaceDesired` disk-read failure, not a synthetic error), and asserts `Status="fail"` under `--strict` / `"warn"` without, plus an integration pin through `runDoctorFull`. The doc comment added alongside contrasts this correctly with `status.go`'s own `resolveOwnershipPlan` caller (M7), which degrades gracefully by design since `status` is best-effort and `doctor --strict` is a gate — read both call sites to confirm the asymmetry is intentional, not a leftover inconsistency.
- **AR#2** (`internal/cli/doctor.go:982-991`, cycle-2-refined): `checkManifestConsistency`'s skip set no longer excludes the settings snapshot (`.ralph/core/settings.ralph.json`) from FR-9(e)'s existence sweep — only `v2SettingsPath` and the two `blockSurfaces` stay excluded, matching the triage's "(e) の除外集合からスナップショットを外す" remediation. Pinned by `TestCheckScaffoldIntegrity_SettingsSnapshotDeleted_StrictFails`, which deletes the snapshot file while leaving its manifest entry intact and asserts `checkManifestConsistency` fails under `--strict` while `checkSettingsOwnedKeys` (FR-9(c)) stays "pass" (confirming the snapshot's `{}`-fallback tolerance in FR-9(c) is real and this is genuinely FR-9(e)'s regression to own, not a duplicate of an existing check).
- **Exception-face audit is sound**, not just asserted. Read `checkManagedBlocks` and `checkSettingsOwnedKeys` directly (not just the doctor.go comment) to confirm the claim that the two faces that *stay* excluded (`settings.json`, `AGENTS.md`/`.gitignore`) are independently guaranteed by FR-9(b)/(c): `checkSettingsOwnedKeys` calls `readFinalDiskContent`, which returns `nil` for a missing file, and `MergeOwnedSettings` treats nil as `"{}"`, which diverges from a non-empty owned template — confirmed this reports `Changed=true`/fail, not a silent pass. `checkManagedBlocks`'s `updateOneBlockV2` returns a zero-value `ok=false` for a missing surface, which the caller treats as an offender. Both directions are also pinned by dedicated audit tests (`TestCheckScaffoldIntegrity_SettingsJSONDeleted_OwnedKeysCatchesIt`, `TestCheckScaffoldIntegrity_AgentsMdDeleted_ManagedBlocksCatchesIt`) that pass on both pre- and post-fix code — correctly labelled in their doc comments as audits, not regression guards.
- **Cycle-2 refinement (C2-L3) verified**: `e39d436` replaced the cycle-1 hand-copied `skip` map literal with `skip := v2SkipPaths(); delete(skip, upgrade.SettingsSnapshotRelPath)`. Read `v2SkipPaths()` (`internal/cli/upgrade_v2.go:43-50`) to confirm it still holds exactly `v2SettingsPath` + the two `blockSurfaces` + the snapshot path — the derived form is behaviorally identical to the cycle-1 literal it replaced, now compile-time linked so a future fifth exception face auto-propagates instead of silently going stale.

## 2. C2 fixes vs. self-review cycle-2 recommendations

Checked each of the five in-cycle findings (C2-M1, C2-M2, C2-L1, C2-L2, C2-L3) against `e39d436`'s diff.

- **C2-M2 (parallel-array allowlist, no delimiter parsing)** — confirmed. `is_allowlisted` (`scripts/check-template-purity.sh:84-96`) now does a plain `[ "$path" = "${ALLOWLIST_PATHS[$i]}" ] && [ "$pattern" = "${ALLOWLIST_PATTERNS[$i]}" ]` index-parallel comparison — no string splitting anywhere in the function, so the fix eliminates the delimiter-collision bug class structurally, independent of whether any given allowlist pattern happens to contain `|`. Ran `grep -n '%%|\*\|#\*|' scripts/check-template-purity.sh` for any surviving delimiter-parsing idiom: one hit remains, at `:167-169`, in the unrelated `FIXED_PATTERNS` loop (`pattern="${entry%%|*}"` / `reason="${entry#*|}"`). This is not the same bug: `FIXED_PATTERNS` entries are `grep -F` literal-substring patterns, all nine of which are plain identifiers/paths (`yoshpy-dev`, `github.com/yoshpy-dev`, `hiroki`, `/Users/`, `skills/release`, `check-sync.sh`, `check-template.yml`, `release.yml`, `overlay-scaffold`, `org-runtime-retire-loop`) — none contain a literal `|`, so splitting them on first `|` cannot truncate a payload the way `REGEX_PATTERNS`/pre-fix-`ALLOWLIST` could. Verified this by reading the full `FIXED_PATTERNS` array (`:39-50`); no entry needs the alternation the bug required. Minor test-coverage note (not a functional gap, since the fix is structural): the commit message describes case C3 as proving scoping "with a live regex-safe entry shape," but C3 (`tests/test-template-purity.sh:118-152`) uses the fixed-string pattern `yoshpy-dev`, not a `REGEX_PATTERNS`-style pattern containing `|` — no test exercises an `ALLOWLIST_PATTERNS` entry that itself contains `|`. Confirmed via code read (not test execution) that this doesn't matter functionally: `is_allowlisted` never splits on any character, so a `|`-containing pattern would compare correctly regardless.
- **C2-M1 (help text + README meta-failure clause)** — confirmed at both surfaces. `--strict` flag help (`internal/cli/doctor.go:38-47`) now adds "plus the meta-failures that make those checks impossible to run: an unparseable manifest or an ownership-planning error such as an unreadable tracked file" and drops the stale "these five scaffold checks" count claim (now "these scaffold checks"). README's `ralph doctor [--strict]` row gets the matching clause. Both read consistently with AR#1/M2's actual `--strict` behavior.
- **C2-L1 (case C/D comments)** — confirmed. New case C3 (`tests/test-template-purity.sh:139-152`) adds a second, non-allowlisted occurrence of the same pattern at a different path and asserts the run fails naming only that second path while the allowlisted path stays suppressed — this is what actually demonstrates exact-(path,pattern) scoping, closing the gap the cycle-1 L2 fix left. Case D's comment now points at C3 instead of the non-demonstrating case C.
- **C2-L2 (grep stderr not merged into parsed hits)** — confirmed. `grep_scan_or_fail` (`scripts/check-template-purity.sh:112-131`) captures via plain `grep ... "$SCAN_ROOT"` (no `2>&1`) on the success path; stderr flows to the script's own stderr instead of `_scan_output`, and the exit->1 branch still surfaces `_scan_output` as diagnostic. Confirmed the parsed-hit loops (`scan_fixed`/`scan_regex`) only ever see stdout now.
- **C2-L3 (skip-set derivation)** — confirmed, see AR#2 discussion above.
- **C2-L4/C2-L5** — confirmed deferred as a single batched tech-debt row in `docs/tech-debt/README.md` (new final row, dated 2026-08-19), consistent with the self-review's own recommendation given the pipeline is at cap 2/2. Row content matches both findings accurately: (1) the 4-site drift-guidance string duplication, (2) the line-number pointer drift in the cycle-1 triage/verify reports caused by `e01939e`'s comment insertions — both carry a concrete trigger, not an open-ended "someday."
- **New observation (not in self-review, informational only)**: `e39d436` renamed `ALLOWLIST` → `ALLOWLIST_PATHS`/`ALLOWLIST_PATTERNS` but left the FAIL-path user guidance text unchanged — `scripts/check-template-purity.sh:188` still reads "add a path+pattern pair to ALLOWLIST in this script with a reason," naming a variable that no longer exists. Cosmetic only (the script's actual behavior is unaffected; a contributor reading this message would still find the right mechanism by reading the script header/comments immediately above the array declarations), but worth a one-line fix on the next touch to this file. Not blocking.

## 3. AC-8/AC-9 after the strictness expansion

Confirmed by reading the changed conditions against the unmodified AC-9 regression tests, not by executing the test suite (tester's responsibility for this cycle).

- `TestCheckScaffoldIntegrity_FreshInit_AllPass` and `TestCheckScaffoldIntegrity_ConvergedUpgrade_AllPass` (`internal/cli/doctor_scaffold_test.go:56-81`) were **not modified** by either `e01939e` or `e39d436` — both still assert `assertAllScaffoldChecksPass`, i.e. exactly the five named FR-9 sub-check results, all `"pass"`, under both `strict=false` and `strict=true`.
- AR#1's changed code path (the `resolveOwnershipPlan` error early-return) is only reachable when `resolveOwnershipPlan` itself errors. A fresh `ralph init` or a cleanly converged `ralph upgrade` never hits this — `resolveOwnershipPlan` succeeds and `checkScaffoldIntegrity` falls through to the normal 5-result path (`internal/cli/doctor.go:748-754`), which AR#1 did not touch. No new false-positive surface for AC-9's two fixtures.
- AR#2's narrowed skip set only changes behavior for a manifest that tracks the settings snapshot path while the file is missing from disk. Both `initV2Project` fixtures (used by both AC-9 tests) write the snapshot as part of normal init/upgrade — the snapshot is present and manifest-tracked in the non-corrupted case, so `checkManifestConsistency`'s existence sweep (now including the snapshot) still finds it and stays "pass." No new false-positive surface here either.
- Net: AC-8/AC-9 continue to hold by construction — the two cycle-1/cycle-2 strictness expansions are additive violation surfaces reachable only by deliberately corrupting a scaffold (as the new regression tests do), not by anything present in a fresh-init or converged state.

## 4. FR-4 drift-guidance alignment (`b9a956f`)

Diffed all four sites directly; the appended guidance line is byte-identical across all of them:

```
Resolve with `ralph eject <path>` (keep the local change, tracked as a fork) or `ralph adopt <path>` (discard the local change, revert to template).
```

- `internal/cli/upgrade_v2.go:229` — real-run error path (`runUpgradeV2`)
- `internal/cli/upgrade_v2.go:309` — no-op tail (`finishNoOpUpgradeV2`)
- `internal/cli/upgrade_v2.go:790` — `--dry-run` preview
- `internal/upgrade/report.go:124` — `## Unresolved drift` markdown section (`renderDriftSection`)

Also confirmed (per `b9a956f`'s own sync-docs report) that no existing test asserts an exact/whole message string at any of the four sites — all assertions use `strings.Contains` on substrings that remain unchanged, so the appended line is additive and safe.

## 5. Static analysis

```
$ ./scripts/run-static-verify.sh
...
==> scripts/check-template-purity.sh
=== Checking templates for meta-repo-specific references ===

PASS: no meta-repo-specific references found in templates.
    OK
==> Language scope: full fallback (unclassified:.claude/hooks/pre_bash_guard.sh)
==> Language packs selected: golang
==> Running golang verifier
==> Go project root: .
gofmt: ok
0 issues.

==> All verifiers passed.

Evidence saved to: docs/evidence/verify-2026-08-19-044835.log
```

Exit code: 0 (confirmed separately via `echo $?` after a clean run, not just tail inspection).

## Cycle 2 known gaps

- Behavioral test execution (including whether `assertAllScaffoldChecksPass`, `TestCheckScaffoldIntegrity_OwnershipPlanningError_StrictFails`, and `TestCheckScaffoldIntegrity_SettingsSnapshotDeleted_StrictFails` actually pass when run) is tester's responsibility for this cycle — not executed here.
- The stale `ALLOWLIST` reference in `check-template-purity.sh`'s FAIL-path message (section 2, new observation) is cosmetic and not filed as a new tech-debt row given the pipeline is at cap 2/2; flagging here for the orchestrator's disposition (PR known-gaps note, or a trivial one-line fix before `/pr` if churn is acceptable).
- Codex project-scoped `[[hooks.*]]` live-firing verification remains open from cycle 1 (unchanged this cycle, tracked in `docs/tech-debt/README.md`).
