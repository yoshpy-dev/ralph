# Test report: overlay-scaffold-v2 Phase 2

- Date: 2026-08-17
- Plan: `docs/plans/active/2026-08-17-overlay-scaffold-v2-p2.md`
- Tester: `tester` subagent (Claude Code, Sonnet 5)
- Scope: behavioral tests via `./scripts/run-test.sh` (`RALPH_VERIFY_SCOPE=full`),
  on `feat/overlay-scaffold-v2-p2`, base `main`, HEAD before this run `4d4bdfc`.
  No static analysis, linting, or spec-compliance re-checking — that is
  `/verify`'s job and is already reported PASS in
  `docs/reports/verify-2026-08-17-overlay-scaffold-v2-p2.md`.
- Evidence: `docs/evidence/test-2026-08-17-overlay-scaffold-v2-p2.log`
  (copy of `docs/evidence/verify-2026-08-17-103002.log`, the raw
  `run-test.sh` output from the final green run below)

## Verdict: PASS

593/593 shell test cases across 25 files, 8/8 Go packages `ok` (fresh,
`-count=1`), zero failures. One real test-isolation bug found and fixed
during this run (see below); one new dispatcher fixture case added to close
a gap the verify report flagged.

## What changed in this run (not just "ran the suite")

1. **Added Case F2 to `tests/test-ralph-dispatch.sh`** (per the verify
   report's "Minimal next check for highest confidence gain"): a fixture
   exercising all three dispatcher layers in one payload —
   `.claude/hooks/<event>.d/` (core) → `.ralph/local/hooks/<event>.d/`
   (downstream local, committed) → `.claude/hooks/local/<event>.d/`
   (downstream local, gitignored) — asserting the full three-part order
   string `core,local,gitignored,`. This closes the AC-4 gap the verify
   report explicitly named as untested (dispatcher wires all 3 layers but
   only 2 were order-tested). 20/20 in the file now (was 19/19).

2. **Found and fixed a real test-isolation bug in Case H3** of the same
   file. Root cause: `scripts/run-test.sh` runs
   `HARNESS_VERIFY_MODE=test exec ./scripts/run-verify.sh`, and
   `run-verify.sh` `export`s `HARNESS_VERIFY_MODE` — so when the shell test
   suite itself is invoked via `run-test.sh` (as this task requires), every
   subprocess in that tree, including `tests/test-ralph-dispatch.sh` and
   the fixture `run-verify.sh` it spawns inside Case H3, inherits
   `HARNESS_VERIFY_MODE=test` from the ambient environment. Case H3 asserts
   "direct `run-verify.sh` (mode=all default) runs both `verify.d` and
   `test.d`" but never overrode `HARNESS_VERIFY_MODE` before invoking the
   fixture, so the inherited `test` value silently changed the mode under
   test and only `test.d` ran. Cases H1/H2 are unaffected because they go
   through the `run-static-verify.sh`/`run-test.sh` wrappers, which each
   force their own `HARNESS_VERIFY_MODE` value regardless of the ambient
   environment — only H3 (direct `run-verify.sh` call) lacked that
   isolation.

   Reproduced deterministically:
   ```
   HARNESS_VERIFY_MODE=test RALPH_VERIFY_SCOPE=full bash tests/test-ralph-dispatch.sh
   # → FAIL H3 (expected [test.d,verify.d,], got [test.d,])
   ```
   Fixed by explicitly `unset HARNESS_VERIFY_MODE` inside Case H3's fixture
   subshell before calling `run-verify.sh`, with a comment explaining why
   (`tests/test-ralph-dispatch.sh`, the H3 block). Re-ran the same
   reproduction command after the fix: 20/20 pass. This was not a
   regression from the plan's implementation — the dispatcher and
   `run-verify.sh` mode-selection logic are correct; the test fixture just
   wasn't hermetic against the exact invocation path (`run-test.sh`) that
   `/test`'s own contract mandates. Worth calling out because it would have
   produced a false negative for every future `/test` run against this
   suite, and a false positive for any prior verify run that invoked
   `run-verify.sh` directly (mode defaults to `all`, so the ambient leak
   never manifested there).

Both changes are test-file-only (`tests/test-ralph-dispatch.sh`); no
production code changed.

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (`RALPH_VERIFY_SCOPE=full`) — 25 shell files | 593 | 593 | 0 | 0 | ~34s (shell) |
| `go test ./... -count=1 -cover` — 8 packages with tests | 8 pkgs | 8 | 0 | 0 | ~55s |

Notable shell files for this plan (new or materially changed):

| File | Cases | Notes |
| --- | --- | --- |
| `tests/test-ralph-dispatch.sh` | 20 (was 19) | AC-4/AC-5. Cases A–H cover deny-decision short-circuit, `additionalContext` merge, byte-exact single-output, non-zero-exit propagation, stdin fan-out, core→`.ralph/local` order (F), **new** all-3-layer order (F2), missing-dir no-op (G), `run-verify.sh`/`run-static-verify.sh`/`run-test.sh` drop-in mode selection + failure propagation (H1–H4) |
| `tests/test-hook-wiring.sh` | 18 | AC-8 hook-command executability smoke: every `settings.json`/`config.toml` hook command resolves to an executable file, root and `templates/base/` mirrors |
| `tests/test-terraform-gitignore.sh` | 47 | unaffected by this plan's scope but reconfirmed green; root/templates-base/cross-check parity |
| `tests/test-terraform-rule-frontmatter.sh` | 11 | AC-6 follow-through: asserts rule now lives at `.claude/rules/ralph/terraform.md` |
| `tests/test-language-pack-monorepo-roots.sh` | 29 | unaffected, reconfirmed green |

Go packages relevant to this plan:

| Package | Tests | Coverage | Notes |
| --- | --- | --- | --- |
| `internal/cli` | pass | 77.6% | includes `internal/cli/init_v2_test.go` — `TestExecuteInit_V2_FreshInit_LayoutAndOwners`, `TestExecuteInit_V2_PreExistingFiles_AppendsBlockPreservesSeed`, `TestExecuteInit_V2_MalformedBlock_LeftUntouchedWithWarning`, `TestExecuteInit_V2_LanguagePack_RuleUnderRalphSubdir`, `TestRunUpgradeIO_V2Layout_FailsClosedWithoutWrites` — all verified individually with `-run` + `-v`, all PASS |
| `internal/scaffold` | pass | 78.9% | manifest v3 ownership API (`SetLayoutV2`, `SetFileOwned`, `SetOwner`, `validateManagedOwner`, `IsLegacyOwner`) is 100% function-covered |
| `internal/upgrade` | pass | 90.0% | AC-10 fail-closed guard covered by the `TestRunUpgradeIO_V2Layout_FailsClosedWithoutWrites` subtests (`force`, `dry-run`) |

## Coverage

- Statement: `internal/cli` 77.6%, `internal/scaffold` 78.9%, `internal/upgrade` 90.0% (all three from a fresh, non-cached `go test ./... -count=1 -cover` run; unrelated packages — `internal/config` 94.2%, `internal/insights` 86.1%, `internal/org` 89.1%, `internal/org/driver` 92.0%, `internal/org/protocol` 97.9% — reconfirmed stable, no regression)
- Branch: not separately instrumented (Go's `-cover` is statement-based); function-level breakdown captured via `go tool cover -func` for `internal/cli`'s `init.go`/`language_pack.go`/`upgrade.go` and `internal/scaffold`'s `manifest.go`
- Function: manifest v3 ownership functions (`SetLayoutV2`, `SetFileOwned`, `SetFileFork`, `SetOwner`, `ownerForScaffoldPath`, `packRuleRelPath`, `packRelDir`, `isInsideDir`) are 100% covered; `executeInit` 82.4%, `renderPackInto` 84.4%
- Notes: no instrumented coverage tool exists for the shell suites — coverage there is measured by test-case scope (593 cases across 25 files), same convention as prior cycles

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| `tests/test-ralph-dispatch.sh` Case H3 (pre-fix) | `expected [test.d,verify.d,], got [test.d,]` when the suite is driven through `./scripts/run-test.sh` | `HARNESS_VERIFY_MODE=test` exported by `run-test.sh` leaked into the fixture's own `run-verify.sh` invocation because Case H3 didn't isolate it | Fixed in this run — `unset HARNESS_VERIFY_MODE` before the fixture call; verified with a targeted repro before and after |

No unresolved failures remain.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Dispatcher deny-decision short-circuit (Case A) | Still correct | 20/20 `test-ralph-dispatch.sh` |
| `additionalContext` aggregation across scripts (Case B) | Still correct | same |
| Non-zero exit propagation + later-script skip (Case D) | Still correct | same |
| `check-pipeline-sync.sh` `REFS` vs. canonical doc parity (fixed in self-review commit `c2d501e`) | Not re-broken | `run-verify.sh` static section (part of the same green `run-test.sh`-adjacent full run) still reports PASS — re-confirmed via the earlier `./scripts/run-verify.sh` full-mode pass in the verify report; not re-run standalone here since it's a static check, out of `/test`'s scope |
| Full shell + Go suite (`docs/reports/verify-2026-08-17-overlay-scaffold-v2-p2.md`'s "All verifiers passed" claim) | Reconfirmed independently via `/test`'s own `run-test.sh` invocation | This report |

## Test gaps

- **`scripts/check-sync.sh`'s block-aware `AGENTS.md` comparison (AC-7) has no isolated fixture test.** `extract_agents_md_block` and the `DRIFTED` branch (`diff -q` mismatch against `.ralph/core/AGENTS.core.md`) are only exercised by running `check-sync.sh` against the real repo tree, which currently has zero drift — so the drift-detected code path itself is never exercised by any test, only the happy path. A cheap follow-up: a fixture repo with an intentionally mutated managed block, asserting `check-sync.sh` reports `DRIFTED` and a non-zero exit.
- `runInitNonInteractive` (`internal/cli/init.go:107`) is 0% covered. Pre-existing gap, not introduced by this plan (confirmed via `git log -- internal/cli/init.go`, the function predates this branch; this plan only added a nearby line). Noted for completeness, not a regression.
- Real Claude Code / Codex runtime execution of the dispatcher against a live hook invocation remains unverifiable by any shell-level test — same caveat the verify report already recorded, inherent to the tooling.

## Verdict

- Pass: yes — 593/593 shell cases, 8/8 Go packages, all green after the F2 addition and the H3 hermeticity fix
- Fail: none outstanding
- Blocked: none

## Cycle 2 (2026-08-17, re-run after cross-review fixes + cycle-2 self-review/verify)

- Delta since cycle 1 (`0132d9f..54f97ac`): cross-review fixes `f80e60f`
  (owner=seed for `.ralph/local/**` and pack-add-created manifest entries,
  plus 2 new Go tests — `TestAddPack_V2Manifest_AssignsOwnerCore`,
  `TestAddPack_LegacyManifest_StaysOwnerless`), cycle-2 self-review cleanup
  `bd2d867` (dispatcher signal-trap rework — `trap cleanup EXIT INT TERM HUP`
  installed before the first `mktemp` so an early kill can't leak the temp
  payload, plus new Case I in `tests/test-ralph-dispatch.sh`; `packRuleRelPath`
  switched to `path.Join`; `pack.go`'s `ownerForScaffoldPath` call site), and
  cycle-2 verify artifact `54f97ac`.
- Evidence: `docs/evidence/test-2026-08-17-overlay-scaffold-v2-p2-cycle2.log`
  (full `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` output, this cycle)

### Verdict: PASS

595/595 shell test cases across the same 25 files (up from 593 — the only
count change is `tests/test-ralph-dispatch.sh` 20 → 22, both new cases from
Case I), 8/8 Go packages `ok` on a fresh `go test ./... -count=1 -cover` run
(not cached), zero failures.

### New tests confirmed passing at runtime

| Test | Command | Result |
| --- | --- | --- |
| `tests/test-ralph-dispatch.sh` Case I (SIGTERM mid-run terminates, not resumed; no stray `.first` file) | `RALPH_VERIFY_SCOPE=full bash tests/test-ralph-dispatch.sh`, run 3x in isolation | PASS all 3 runs, `exit 143` (SIGTERM) each time, no leaked temp file |
| `TestAddPack_V2Manifest_AssignsOwnerCore` (`internal/cli`) | `go test ./internal/cli/... -run 'TestAddPack_V2Manifest_AssignsOwnerCore' -v -count=1` | PASS — asserts `addPack` on a v2-layout project sets owner=core on `packs/languages/<lang>/verify.sh`, `packs/languages/<lang>/README.md`, and `.claude/rules/ralph/<lang>.md` |
| `TestAddPack_LegacyManifest_StaysOwnerless` (`internal/cli`) | same command, `-run 'TestAddPack_LegacyManifest_StaysOwnerless'` | PASS — asserts `addPack` on a legacy (pre-v2, `Meta.Layout` unset) manifest does not backfill ownership metadata on newly-added entries and does not silently upgrade the manifest to v2 |
| `.ralph/local/**` owner=seed spot-check | inspected `internal/cli/init.go:295-310` (`ownerForScaffoldPath`) directly, cross-checked against `TestExecuteInit_V2_FreshInit_LayoutAndOwners`'s `wantOwners[".ralph/local/verify.d/.gitkeep"] = scaffold.OwnerSeed` assertion (already part of the full green run above) | Confirmed: `.ralph/local/` is special-cased to `OwnerSeed` ahead of the catch-all `OwnerCore`, with an inline comment explaining why (L3 overlay, 不可侵, would otherwise be treated as a full-replace target by the Phase 3 replace planner) |

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped |
| --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` — 25 shell files | 595 | 595 | 0 | 0 |
| `go test ./... -count=1 -cover` — 8 packages with tests | 8 pkgs | 8 | 0 | 0 |

### Coverage (fresh, non-cached, cycle 2)

`internal/cli` 77.6%, `internal/scaffold` 78.9%, `internal/upgrade` 90.0% —
byte-identical to cycle 1 despite the 2 new `internal/cli` tests and the
`ownerForScaffoldPath`/`packRuleRelPath` edits; the new tests exercise
already-covered code paths (ownership assignment, pack-add flow), so no
percentage shift. Other packages reconfirmed stable: `internal/config`
94.2%, `internal/insights` 86.1%, `internal/org` 89.1%, `internal/org/driver`
92.0%, `internal/org/protocol` 97.9%.

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Dispatcher deny-decision short-circuit, `additionalContext` aggregation, non-zero exit propagation (Cases A/B/D) | Still correct | 22/22 `test-ralph-dispatch.sh`, this cycle |
| H3 `HARNESS_VERIFY_MODE` hermeticity fix (cycle 1) | Not re-broken | `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` full-suite run (ambient `HARNESS_VERIFY_MODE=test` present) still shows H3 PASS |
| Cycle-1 dispatcher `trap cleanup EXIT` (single-signal) | Superseded, not regressed | Cycle-2 self-review flagged it as leaking on signal kill; `bd2d867` widened it to `EXIT INT TERM HUP`, and Case I now exercises the SIGTERM path directly (new coverage, not just non-regression) |

### Test gaps (unchanged from cycle 1, reconfirmed still open)

- `scripts/check-sync.sh`'s block-aware `AGENTS.md` `DRIFTED` code path
  still has no isolated fixture test (only the happy path is exercised
  against the real repo tree).
- `runInitNonInteractive` (`internal/cli/init.go:107`) remains 0% covered;
  pre-existing gap, not introduced by this plan or this cycle's changes.

### Cycle 2 verdict

- Pass: yes — 595/595 shell cases, 8/8 Go packages, all new/changed tests
  individually re-confirmed at runtime
- Fail: none outstanding
- Blocked: none

## Cycle 3 (2026-08-17, final — re-run after cross-review c2/c3 fixes, cap raised 2 → 3)

- Delta since cycle 2 (`d21273b..a7c857b`): cross-review cycle-2 fixes
  `c289875` (doctor now parses dispatcher hook commands with arguments
  instead of comparing the whole command string, closing a false-negative
  where `./.claude/hooks/ralph-dispatch.sh <event>` always failed the
  executability check; scaffold refuses to write into a symlinked block
  surface and warns instead of following it; 4 new Go tests — 3 in
  `internal/cli/doctor_hooks_test.go`,
  `TestExecuteInit_V2_FreshInit_DoctorHooksIntegrityPasses` and
  `TestExecuteInit_V2_SymlinkedBlockSurface_LeftUntouchedWithWarning` in
  `internal/cli/init_v2_test.go`), cycle-3 self-review fixes `2b1a855`
  (`internal/scaffold/render.go` now `Lstat`s before following a symlink so
  a *dangling* symlinked block surface is also left untouched with a
  warning rather than erroring or silently overwriting the link target;
  2 new tests — `TestRenderFS_DanglingSymlink_NonForce_SkippedNotFollowed`
  in `internal/scaffold/render_test.go`,
  `TestExecuteInit_V2_DanglingSymlinkBlockSurface_LeftUntouchedWithWarning`
  in `internal/cli/init_v2_test.go` — plus a reworked, previously-vacuous
  Case I in `tests/test-ralph-dispatch.sh`: the old version asserted only
  that a background dispatcher process could be killed, not that the
  SIGTERM path itself ran; it now asserts `exit 143`, no leaked `.first`
  fixture-cwd file, and no leaked `ralph-dispatch-*` temp file, exercising
  the `trap cleanup EXIT INT TERM HUP` handler added in cycle 2), and a new
  tech-debt row for two cosmetic cycle-3 self-review findings deferred at
  the raised cap (insight-event `cycle` mis-stamping, one `t.Fatalf`
  precondition-style test message).
- Evidence: `docs/evidence/test-2026-08-17-overlay-scaffold-v2-p2-cycle3.log`
  (full `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` output, this cycle)

### Verdict: PASS

596/596 shell test cases across the same 25 files (up from 595 — the only
count change is `tests/test-ralph-dispatch.sh` 22 → 23, the reworked Case I
now carries 3 assertions instead of 1), 8/8 Go packages `ok` on a fresh
`go test ./... -count=1 -cover` run (not cached), zero failures.

### New tests confirmed passing at runtime

| Test | Command | Result |
| --- | --- | --- |
| `TestExecuteInit_V2_FreshInit_DoctorHooksIntegrityPasses` (`internal/cli`) | `go test ./internal/cli/... -run 'TestExecuteInit_V2_FreshInit_DoctorHooksIntegrityPasses' -v -count=1` | PASS |
| `TestCheckHooks_DispatcherCommandWithArgs_ScriptPresentReportsPass` (`internal/cli`, `doctor_hooks_test.go`) | `go test ./internal/cli/... -run 'TestCheckHooks_DispatcherCommandWithArgs_ScriptPresentReportsPass' -v -count=1` | PASS |
| `TestCheckHooks_DispatcherCommandWithArgs_ScriptMissingReportsFail` (`internal/cli`, `doctor_hooks_test.go`) | same command, `-run 'TestCheckHooks_DispatcherCommandWithArgs_ScriptMissingReportsFail'` | PASS |
| `TestCheckHooks_ArgLessCommand_ScriptPresentReportsPass` (`internal/cli`, `doctor_hooks_test.go`) | same command, `-run 'TestCheckHooks_ArgLessCommand_ScriptPresentReportsPass'` | PASS |
| `TestExecuteInit_V2_SymlinkedBlockSurface_LeftUntouchedWithWarning` (`internal/cli`) | `go test ./internal/cli/... -run 'TestExecuteInit_V2_SymlinkedBlockSurface_LeftUntouchedWithWarning' -v -count=1` | PASS |
| `TestRenderFS_DanglingSymlink_NonForce_SkippedNotFollowed` (`internal/scaffold`) | `go test ./internal/scaffold/... -run 'TestRenderFS_DanglingSymlink_NonForce_SkippedNotFollowed' -v -count=1` | PASS |
| `TestExecuteInit_V2_DanglingSymlinkBlockSurface_LeftUntouchedWithWarning` (`internal/cli`) | `go test ./internal/cli/... -run 'TestExecuteInit_V2_DanglingSymlinkBlockSurface_LeftUntouchedWithWarning' -v -count=1` | PASS |
| Reworked `tests/test-ralph-dispatch.sh` Case I (SIGTERM → exit 143, no leaked `.first`/`ralph-dispatch-*` temp files) | `./tests/test-ralph-dispatch.sh`, run 3x in isolation | PASS all 3 runs, identical `exit 143` and zero-leak results each time |

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped |
| --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` — 25 shell files | 596 | 596 | 0 | 0 |
| `go test ./... -count=1 -cover` — 8 packages with tests | 8 pkgs | 8 | 0 | 0 |

### Coverage (fresh, non-cached, cycle 3)

`internal/cli` 78.9% (up from 77.6% in cycle 2 — the doctor-hooks parsing
fix and the two new symlink-guard code paths in `init.go`/`doctor.go` are
now exercised), `internal/scaffold` 78.9% (unchanged — the new
`Lstat`-before-follow branch in `render.go` sits inside an already-covered
function so the statement percentage doesn't move even though a new
branch is now covered), `internal/upgrade` 90.0% (unchanged). Other
packages reconfirmed stable: `internal/config` 94.2%, `internal/insights`
86.1%, `internal/org` 89.1%, `internal/org/driver` 92.0%,
`internal/org/protocol` 97.9%.

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Dispatcher deny-decision short-circuit, `additionalContext` aggregation, non-zero exit propagation (Cases A/B/D) | Still correct | 23/23 `test-ralph-dispatch.sh`, this cycle |
| H3 `HARNESS_VERIFY_MODE` hermeticity fix (cycle 1) | Not re-broken | full-suite run (ambient `HARNESS_VERIFY_MODE=test` present) still shows H3 PASS |
| Cycle-2 dispatcher `trap cleanup EXIT INT TERM HUP` widening | Now directly exercised, not just non-regression | Reworked Case I asserts SIGTERM (`exit 143`) and zero temp-file leakage in 3/3 isolated runs — cycle 2's version of Case I only asserted the process could be killed, not that the trap ran |
| `addPack` owner assignment (`TestAddPack_V2Manifest_AssignsOwnerCore`, `TestAddPack_LegacyManifest_StaysOwnerless`, cycle 2) | Not re-broken | Part of the fresh `go test ./internal/cli/...` full-package run above, still PASS |

### Test gaps (updated from cycle 2)

- `scripts/check-sync.sh`'s block-aware `AGENTS.md` `DRIFTED` code path
  still has no isolated fixture test (only the happy path is exercised
  against the real repo tree). Unchanged from cycle 1/2.
- `runInitNonInteractive` (`internal/cli/init.go:107`) remains 0% covered;
  pre-existing gap, not introduced by this plan or this cycle's changes.
- **New, cosmetic, deferred to tech debt (not a test gap in the strict
  sense but flagged here for traceability):** two entries in
  `docs/insights/events/2026-08-17-overlay-scaffold-v2-p2.jsonl` are
  stamped `cycle:1` for events actually appended by cycle-2 fix commits,
  and cycle 2 has no `self_review` event recorded at all — `ralph insights`
  will under/mis-report this slug's pipeline cadence until corrected. See
  `docs/tech-debt/README.md`'s newest row for the fix pointer.

### Cycle 3 verdict

- Pass: yes — 596/596 shell cases, 8/8 Go packages, all new/changed tests
  individually re-confirmed at runtime, reworked Case I stable across 3
  isolated flake-check runs
- Fail: none outstanding
- Blocked: none
- This is the final pipeline cycle for overlay-scaffold-v2 Phase 2 (cap
  raised 2 → 3); ready for `/pr`

## Cycle 4 (2026-08-17, final — re-run after post-cycle-3 dispatcher child-kill fix)

- Delta since cycle 3 (`bb22a24..34120b3`): `28842a0` (dispatcher now kills
  the in-flight hook script's child process on INT/TERM/HUP instead of only
  killing itself and leaving the child to finish detached — a background
  `"$script" ... & child=$!; wait "$child"` plus a `kill_child` helper
  invoked before `cleanup` in each fatal-signal trap; `renderMappedFile`
  switched `os.Stat` → `os.Lstat` before the exists-check so a *dangling*
  symlink at a pack rule's target path is skipped rather than followed and
  written through — mirrors the `scaffold.RenderFS` fix from cycle 3, closing
  the same containment gap at the `internal/cli/language_pack.go` mapped-file
  path; one new Go test,
  `TestRenderMappedFile_DanglingSymlink_NonForce_SkippedNotFollowed`; Case I
  in `tests/test-ralph-dispatch.sh` extended with three new assertions —
  prompt-interruption timing (`<=3s`, not deferred until the 5s hook script
  finishes on its own), a `finished_marker` check proving the hook child was
  actually killed rather than merely orphaned, and a defense-in-depth `pgrep`
  check for no surviving process), plus five docs/bookkeeping-only commits
  (`88ba334`, `1e03cd5`, `ef9e8e2`, `756aac8`, `c5771af`, `34120b3` — insight
  event cycle-stamp correction, tech-debt row refresh, cycle-4 self-review
  and verify report additions). No other production code changed.
- Evidence: `docs/evidence/test-2026-08-17-overlay-scaffold-v2-p2-cycle4.log`
  (full `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` output, this cycle;
  copy of `docs/evidence/verify-2026-08-17-160754.log`, the raw run behind
  this cycle's green result)

### Verdict: PASS

599/599 shell test cases across the same 25 files (up from 596 — the only
count change is `tests/test-ralph-dispatch.sh` 23 → 26, the three new Case I
assertions), 8/8 Go packages `ok` on a fresh `go test ./... -count=1 -cover`
run (not cached), zero failures.

### New tests confirmed passing at runtime

| Test | Command | Result |
| --- | --- | --- |
| `TestRenderMappedFile_DanglingSymlink_NonForce_SkippedNotFollowed` (`internal/cli`) | `go test ./internal/cli/... -run 'TestRenderMappedFile_DanglingSymlink_NonForce_SkippedNotFollowed' -v -count=1` | PASS — asserts a dangling symlink at a pack rule's `.claude/rules/ralph/<lang>.md` target is left untouched (still a symlink, listed in `Skipped` not `Created`) and nothing is written at the external dangling-link target |
| Extended `tests/test-ralph-dispatch.sh` Case I (prompt-interruption timing, `finished_marker` absence, `pgrep` no-survivor check) | `RALPH_VERIFY_SCOPE=full bash tests/test-ralph-dispatch.sh`, run 3x in isolation | PASS all 3 runs — SIGTERM interrupted the dispatcher in ~0s (well under the 3s bound), `finished_marker` absent every run (the hook child's 5s `sleep` never ran to completion), no `PreCompact.d/10-slow.sh` process survived per `pgrep` |

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped |
| --- | --- | --- | --- | --- |
| `RALPH_VERIFY_SCOPE=full ./scripts/run-test.sh` — 25 shell files | 599 | 599 | 0 | 0 |
| `go test ./... -count=1 -cover` — 8 packages with tests | 8 pkgs | 8 | 0 | 0 |

### Coverage (fresh, non-cached, cycle 4)

`internal/cli` 79.0% (up from 78.9% in cycle 3 — the new
`TestRenderMappedFile_DanglingSymlink_NonForce_SkippedNotFollowed` test
exercises the new `os.Lstat` branch in `renderMappedFile`),
`internal/scaffold` 78.9% (unchanged), `internal/upgrade` 90.0% (unchanged).
Other packages reconfirmed stable: `internal/config` 94.2%, `internal/insights`
86.1%, `internal/org` 89.1%, `internal/org/driver` 92.0%,
`internal/org/protocol` 97.9%.

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Dispatcher deny-decision short-circuit, `additionalContext` aggregation, non-zero exit propagation (Cases A/B/D) | Still correct | 26/26 `test-ralph-dispatch.sh`, this cycle |
| H3 `HARNESS_VERIFY_MODE` hermeticity fix (cycle 1) | Not re-broken | full-suite run (ambient `HARNESS_VERIFY_MODE=test` present) still shows H3 PASS |
| Cycle-2 dispatcher `trap cleanup EXIT INT TERM HUP` widening, cycle-3 reworked Case I (SIGTERM → exit 143, no leaked temp files) | Not re-broken, and now covers the previously-unverified "child process is actually killed" gap | Extended Case I passes 3/3 isolated runs this cycle; cycle-3's version only proved the dispatcher process itself died, not that the hook script running underneath it was also terminated |
| `scaffold.RenderFS` dangling-symlink guard (cycle 3, `TestRenderFS_DanglingSymlink_NonForce_SkippedNotFollowed`) | Not re-broken | Part of the fresh `go test ./internal/scaffold/...` full-package run above, still PASS |
| `addPack` owner assignment (cycle 2), doctor hook-arg parsing + scaffold symlink guards (cycle 3) | Not re-broken | Part of the fresh `go test ./internal/cli/...` full-package run above, still PASS |

### Test gaps (unchanged from cycle 3)

- `scripts/check-sync.sh`'s block-aware `AGENTS.md` `DRIFTED` code path
  still has no isolated fixture test (only the happy path is exercised
  against the real repo tree). Unchanged from cycle 1/2/3.
- `runInitNonInteractive` (`internal/cli/init.go:107`) remains 0% covered;
  pre-existing gap, not introduced by this plan or this cycle's changes.
- The tech-debt row from cycle 3 (insight-event `cycle` mis-stamping) was
  corrected in this cycle's delta (`88ba334`); no longer an open gap.

### Cycle 4 verdict

- Pass: yes — 599/599 shell cases, 8/8 Go packages, both new/changed tests
  (the Go dangling-symlink test and the three extended Case I assertions)
  individually re-confirmed at runtime, extended Case I stable across 3
  isolated flake-check runs
- Fail: none outstanding
- Blocked: none
- This is pipeline cycle 4/4 (cap raised 2 → 3 → 4) for overlay-scaffold-v2
  Phase 2; ready for `/pr`
