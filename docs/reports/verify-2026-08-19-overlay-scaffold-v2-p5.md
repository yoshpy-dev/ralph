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
