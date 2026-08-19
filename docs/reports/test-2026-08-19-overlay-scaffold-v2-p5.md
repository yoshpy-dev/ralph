# Test report: overlay-scaffold-v2 Phase 5 (eject/adopt, doctor --strict, status ownership, purity guard, Codex hooks parity)

- Date: 2026-08-19
- Plan: `docs/plans/active/2026-08-19-overlay-scaffold-v2-p5.md`
- Tester: `tester` subagent (Claude Code, Sonnet 5)
- Scope: behavioral tests via `./scripts/run-test.sh` (default `changed`
  scope), on `feat/overlay-scaffold-v2-p5`, base `main` `1025d13`, HEAD
  `74d28ef`. No static analysis, linting, or spec-compliance re-checking —
  that is `/verify`'s job and is already reported PASS in
  `docs/reports/verify-2026-08-19-overlay-scaffold-v2-p5.md`.
- Evidence: `docs/evidence/verify-2026-08-19-034501.log` (raw `run-test.sh`
  output from the confirmed exit-0 run below)

## Verdict: PASS

614/614 shell tests across 27 files, all passing (zero `FAIL` lines
anywhere in the run), 8/8 Go packages `ok` (fresh, `-count=1 -cover`, no
build cache), zero test failures. All five named regression tests from the
assignment plus the two new shell fixture suites (AC-11 purity guard, AC-12b
pre_bash_guard jq-nested-payload extraction) were individually re-run 3x in
isolation with zero flakes. Both watchlist flaky tests
(`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`,
`TestRunWatcher_TimeoutIndependentOfSmallInterval`) also re-confirmed stable
3x in isolation. No test weakening found.

`./scripts/run-test.sh` selected `golang` as the sole changed-language pack
(`==> Language scope: full fallback
(unclassified:.claude/hooks/pre_bash_guard.sh)` → `==> Language packs
selected: golang`) — the shell hook change with no recognized extension
triggered a full-fallback language classification, but the wrapper still ran
the complete 27-file shell suite ahead of the language-scoped Go dispatch
(`grep -c '^==> tests/'` = 27, `grep -c 'FAIL: [1-9]'` = 0 anywhere in the
log), so shell scope was effectively full regardless of the golang-only Go
selection.

## Test execution

| Suite / Command | Files/Packages | Passed | Failed | Duration |
| --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (default `changed` scope) | 27 shell test files + golang pack | 614 shell + 8/8 Go pkgs | 0 | ~50s |
| `go test ./... -count=1 -cover` (fresh, no cache) | 8 Go packages with tests | 8 | 0 | ~1m1s |

`./scripts/run-test.sh` exit code: `0` (captured via the tool's own exit
status, not piped through `tee`, per [[feedback_tee_masks_exit]]). All 27
shell files were also re-run individually (`sh tests/test-*.sh`) with their
own exit codes captured directly, confirming the batch run's zero-failure
result file-by-file.

Per-file shell breakdown (all 27 files, cross-checked against both the
batch `run-test.sh` log and individual `sh tests/test-*.sh` runs):

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
| test-hook-wiring.sh | 22 |
| test-insights-append.sh | 39 |
| test-language-pack-monorepo-roots.sh | 29 |
| test-no-loop-references.sh | 1 |
| **test-pre-bash-guard.sh (new, AC-12b)** | **4** |
| test-ralph-config.sh | 15 |
| test-ralph-dispatch.sh | 26 |
| test-ralph-worktree.sh | 29 |
| test-run-verify-scope.sh | 12 |
| test-secret-scan.sh | 6 |
| test-self-review-scope.sh | 64 |
| test-sync-skills.sh | 22 |
| **test-template-purity.sh (new, AC-11)** | **7** |
| test-terraform-gitignore.sh | 47 |
| test-terraform-pack-verify.sh | 36 |
| test-terraform-rule-frontmatter.sh | 11 |
| test-verify-mode-split.sh | 59 |
| test-xreview-helpers.sh | 29 |
| **Total** | **614** |

Delta from the Phase-4 baseline (599 across 25 files,
`docs/reports/test-2026-08-18-overlay-scaffold-v2-p4.md`, cycle 3): +15
tests, +2 files. Both new files (`test-pre-bash-guard.sh` = 4,
`test-template-purity.sh` = 7 = 11 tests) plus `test-hook-wiring.sh` growing
from 18 to 22 (+4, the AC-12 direct-call-detection hardening) account for
the full +15 exactly. Every other file's count is byte-identical to the
Phase-4 baseline.

## Fresh Go coverage vs Phase-4 baseline

Fresh `go test ./... -count=1 -cover` (bypassing `run-test.sh`'s build cache
to rule out stale results from before the Slice 1-5 commits):

| Package | Phase-4 baseline (78.7%/…) | Phase-5 (this run) | Delta |
| --- | --- | --- | --- |
| `internal/cli` | 78.7% | 79.9% | +1.2pp |
| `internal/config` | 94.2% | 94.2% | — |
| `internal/insights` | 86.1% | 86.1% | — |
| `internal/org` | 89.1% | 89.1% | — |
| `internal/org/driver` | 92.0% | 92.0% | — |
| `internal/org/protocol` | 97.9% | 97.9% | — |
| `internal/scaffold` | 75.7% | 75.7% | — |
| `internal/upgrade` | 91.2% | 91.2% | — |

`internal/cli` is the only package touched by this plan (`eject.go`,
`adopt.go`, `doctor.go` strict checks, `status.go` ownership section, plus
their test files) and rose +1.2pp; every other package is byte-identical to
the Phase-4 baseline, as expected — the FR-10/FR-12b shell-only slice
touches no Go source.

## Named regression tests (individually re-run 3x in isolation)

Per the assignment's focus list, plus the two new shell fixture suites
covering AC-11 and AC-12b:

| Test | Command | Result (3/3) |
| --- | --- | --- |
| Eject/adopt round-trip (AC-13) | `go test ./internal/cli/ -run '^TestEjectAdoptRoundTrip_AC13$' -count=3 -v` | PASS x3, ~0.85s each |
| `TestRunAdoptIO_PreflightFailure_AllBatchZeroWrites` (M3-strengthened) | `go test ./internal/cli/ -run '^TestRunAdoptIO_PreflightFailure_AllBatchZeroWrites$' -count=3 -v` | PASS x3, ~0.82s each |
| Doctor strict exit-code (`TestRunDoctorFull_StrictFlipsExitCode_DriftedCore`) | `go test ./internal/cli/ -run '^TestRunDoctorFull_StrictFlipsExitCode_DriftedCore$' -count=3 -v` | PASS x3, ~0.48s each |
| Corrupt-manifest strict (M2, `TestCheckScaffoldIntegrity_CorruptManifest_StrictFails`) | `go test ./internal/cli/ -run '^TestCheckScaffoldIntegrity_CorruptManifest_StrictFails$' -count=3 -v` | PASS x3, ~0.47s each |
| Status degradation (M7, `TestStatusCmd_ScaffoldSection_ComputationError_DegradesGracefully`) | `go test ./internal/cli/ -run '^TestStatusCmd_ScaffoldSection_ComputationError_DegradesGracefully$' -count=3 -v` | PASS x3, ~0.07s each |
| `tests/test-template-purity.sh` cases E/F (H1 regex branch: dated `docs/reports/`/`docs/plans/` path exclusion) | `sh tests/test-template-purity.sh` x3 (full 7-case file, cases E/F are `PASS  E./F.` lines within it) | PASS x3, exit 0 each, cases E and F both green every run |
| `tests/test-pre-bash-guard.sh` (AC-12b, jq-nested `tool_input.command` extraction, cases A-D) | `sh tests/test-pre-bash-guard.sh` x3 | PASS x3, exit 0 each, 4/4 cases green every run |

The eject/adopt round-trip test constructs an initial owned tracked file,
ejects it to `owner=fork`, mutates disk content, adopts it back to
`owner=core`, and asserts both the manifest state and disk content converge
to the pre-ejection baseline — this is the direct behavioral proof of AC-13.

The template-purity suite's cases E and F are the regex-branch regression
specifically called out in the plan (tech-debt line 106's lesson: a
detection branch without its own fixture is unverified) — they assert that
a *dated* `docs/reports/<date>-<slug>` or `docs/plans/<date>-<slug>` path
string fails the purity check (distinct from the plain
`yoshpy-dev`/repo-URL literal-match cases A-D covered separately).

## Known-flaky watchlist (re-checked 3x isolated)

| Test | Command | Result (3/3) |
| --- | --- | --- |
| `TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess` | `go test ./internal/cli/ -run '^TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess$' -count=3 -v` | PASS x3, ~0.60s each |
| `TestRunWatcher_TimeoutIndependentOfSmallInterval` | `go test ./internal/org/ -run '^TestRunWatcher_TimeoutIndependentOfSmallInterval$' -count=3 -v` | PASS x3, ~1.2s each |

Both tests are documented as transient subprocess-timeout flakes under
parallel/adjacent-real-subprocess contention (tester agent memory,
[[flaky_test_notes]]); neither is in this plan's delta, and both re-ran
cleanly in isolation, consistent with every prior cycle's re-check.

## Coverage gaps / blind spots

- No new shell-level coverage gap identified: both new shell suites
  (`test-pre-bash-guard.sh`, `test-template-purity.sh`) each exercise
  their happy-path and rejection branches (allow/deny x jq-present/jq-absent
  for pre_bash_guard; clean-tree/injected-leak/allowlisted/unallowlisted/
  dated-path-regex for template-purity).
- `internal/cli` remains the largest package (79.9%) with room below other
  packages' averages; this is consistent with prior-phase notes that CLI
  command wiring (flag parsing, cobra boilerplate, human-facing print
  paths) is inherently harder to drive to high coverage than pure logic
  packages like `internal/org/protocol` (97.9%) — not a regression
  introduced by this plan.
- Codex hooks dispatcher parity (Slice 4) is validated by
  `tests/test-hook-wiring.sh` (22 cases, direct-call-detection hardened)
  plus `scripts/check-sync.sh` (byte-identical root/template `.codex/`
  trees) — both are static/structural checks. The plan's own "Open
  questions" section records a residual gap: interactive `codex trust`
  real-firing confirmation for project-scoped `[[hooks.*]]` was not
  reproducible in a scratch repo and remains unverified by any automated
  test (recorded as a known gap in the plan itself, not newly discovered
  here).

## What was NOT run (out of scope for `/test`)

- Static analysis (gofmt, vet, golangci-lint, staticcheck, shellcheck,
  purity guard as a CI gate) — `/verify`'s job, already PASS in
  `docs/reports/verify-2026-08-19-overlay-scaffold-v2-p5.md`.
- Documentation drift checks — `/sync-docs`'s job, not yet run.
- Cross-model second opinion — `/cross-review`'s job, not yet run.

## Recommendation

Tests pass. Proceed to `/sync-docs`.

## Cycle 2

- Date: 2026-08-19
- Tester: `tester` subagent (Claude Code, Sonnet 5)
- Scope: pipeline cycle 2/2, re-running behavioral tests after the cycle-1
  fix-and-revalidate delta on top of the cycle-1-passing base `38b7c63`
  (614 shell + 8/8 Go, `internal/cli` 79.9%): `b9a956f` (drift guidance
  strings), `e01939e` (AR#1/AR#2 doctor `--strict` fixes, +4 Go tests),
  `e39d436` (C2 fixes: parallel-array allowlist + shell test case C3, strict
  help text, skip-set derivation), `1f50407` (stale allowlist name in
  purity-guard failure message), plus report-only commits (`2bc988f`,
  `c805cf5`, `348801c`, `7286251`, `74d28ef`). HEAD at test time:
  `1f50407`.
- Evidence: `docs/evidence/verify-2026-08-19-045214.log` (raw
  `run-test.sh` output, exit 0)

### Verdict: PASS

615/615 shell tests across the same 27 files (+1 from cycle 1's 614, exactly
matching the expected delta: `tests/test-template-purity.sh` grew from 7 to
8 with the new case C3, `e39d436`'s "allowlist entry is exact-path scoped"
regression test — every other file's per-file count is byte-identical to
cycle 1), 8/8 Go packages `ok` (fresh `go test ./... -count=1 -cover`,
no build cache), zero failures anywhere. All 6 named
`TestCheckScaffoldIntegrity_*` regression tests plus both flake-watchlist
tests plus both shell fixture suites individually re-run 3x in isolation
with zero flakes. No test weakening found.

`./scripts/run-test.sh` again selected `golang` as the sole changed-language
Go pack via the same full-fallback shell classification as cycle 1
(`==> Language scope: full fallback (unclassified:.claude/hooks/pre_bash_guard.sh)`),
but ran the complete 27-file shell suite regardless (27 `==> tests/` blocks,
zero non-zero `FAIL:` counts anywhere in the log), so shell scope was
effectively full.

### Test execution

| Suite / Command | Files/Packages | Passed | Failed | Exit |
| --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (default `changed` scope) | 27 shell test files + golang pack | 615 shell + 8/8 Go pkgs | 0 | 0 |
| `go test ./... -count=1 -cover` (fresh, no cache) | 8 Go packages with tests | 8 | 0 | 0 |

Per-file shell delta from cycle 1 (all other 26 files byte-identical, not
re-tabulated here — see the Cycle 1 table above):

| File | Cycle 1 | Cycle 2 | Delta |
| --- | --- | --- | --- |
| test-template-purity.sh | 7 | 8 | +1 (new case C3) |
| **Total (27 files)** | **614** | **615** | **+1** |

### Fresh Go coverage vs cycle-1 baseline

| Package | Cycle 1 | Cycle 2 (this run) | Delta |
| --- | --- | --- | --- |
| `internal/cli` | 79.9% | 80.0% | +0.1pp |
| `internal/config` | 94.2% | 94.2% | — |
| `internal/insights` | 86.1% | 86.1% | — |
| `internal/org` | 89.1% | 89.1% | — |
| `internal/org/driver` | 92.0% | 92.0% | — |
| `internal/org/protocol` | 97.9% | 97.9% | — |
| `internal/scaffold` | 75.7% | 75.7% | — |
| `internal/upgrade` | 91.2% | 91.2% | — |

`internal/cli` is the only package touched by the cycle-2 delta
(`e01939e`'s AR#1/AR#2 doctor `--strict` fixes added 4 new Go test cases:
`TestCheckScaffoldIntegrity_OwnershipPlanningError_StrictFails`,
`TestCheckScaffoldIntegrity_SettingsSnapshotDeleted_StrictFails`,
`TestCheckScaffoldIntegrity_SettingsJSONDeleted_OwnedKeysCatchesIt`,
`TestCheckScaffoldIntegrity_AgentsMdDeleted_ManagedBlocksCatchesIt`); it
rose +0.1pp. Every other package is byte-identical to cycle 1, as expected
— `b9a956f`, `e39d436`, and `1f50407` touch only docs/shell strings, not Go
source.

### Named regression tests (individually re-run 3x in isolation)

| Test | Command | Result (3/3) |
| --- | --- | --- |
| `TestCheckScaffoldIntegrity_OwnershipPlanningError_StrictFails` | `go test ./internal/cli/ -run '^TestCheckScaffoldIntegrity_OwnershipPlanningError_StrictFails$' -count=3 -v` | PASS x3, ~0.6-0.9s each |
| `TestCheckScaffoldIntegrity_SettingsSnapshotDeleted_StrictFails` | `go test ./internal/cli/ -run '^TestCheckScaffoldIntegrity_SettingsSnapshotDeleted_StrictFails$' -count=3 -v` | PASS x3, ~0.07-0.08s each |
| `TestCheckScaffoldIntegrity_SettingsJSONDeleted_OwnedKeysCatchesIt` | `go test ./internal/cli/ -run '^TestCheckScaffoldIntegrity_SettingsJSONDeleted_OwnedKeysCatchesIt$' -count=3 -v` | PASS x3, ~0.07s each |
| `TestCheckScaffoldIntegrity_AgentsMdDeleted_ManagedBlocksCatchesIt` | `go test ./internal/cli/ -run '^TestCheckScaffoldIntegrity_AgentsMdDeleted_ManagedBlocksCatchesIt$' -count=3 -v` | PASS x3, ~0.06-0.07s each |
| `TestCheckScaffoldIntegrity_FreshInit_AllPass` (strictness-expansion false-positive guard) | `go test ./internal/cli/ -run '^TestCheckScaffoldIntegrity_FreshInit_AllPass$' -count=3 -v` | PASS x3, ~0.07s each |
| `TestCheckScaffoldIntegrity_ConvergedUpgrade_AllPass` (strictness-expansion false-positive guard) | `go test ./internal/cli/ -run '^TestCheckScaffoldIntegrity_ConvergedUpgrade_AllPass$' -count=3 -v` | PASS x3, ~0.10-0.11s each |
| `tests/test-template-purity.sh` (8 cases incl. new C3 exact-path-scoping, AC-11) | `sh tests/test-template-purity.sh` x3 (full 8-case file) | PASS x3, exit 0 each, 8/8 green every run |
| `tests/test-pre-bash-guard.sh` (AC-12b, jq-nested extraction, cases A-D) | `sh tests/test-pre-bash-guard.sh` x3 | PASS x3, exit 0 each, 4/4 green every run |

The two `_StrictFails` tests and `TestCheckScaffoldIntegrity_SettingsJSONDeleted_OwnedKeysCatchesIt`
/ `TestCheckScaffoldIntegrity_AgentsMdDeleted_ManagedBlocksCatchesIt` are the
direct behavioral proof of AR#1/AR#2 (`doctor --strict` now fails on an
ownership-planning error and on a deleted settings snapshot, rather than
silently passing). The two `_AllPass` tests confirm the strictness expansion
introduced no false positives against a fresh-init or converged-upgrade
fixture — required because a strict-mode tightening that also flags healthy
state is worse than the bug it fixes.

### Known-flaky watchlist (re-checked 3x isolated)

| Test | Command | Result (3/3) |
| --- | --- | --- |
| `TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess` | `go test ./internal/cli/ -run '^TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess$' -count=3 -v` | PASS x3, ~0.66-0.71s each |
| `TestRunWatcher_TimeoutIndependentOfSmallInterval` | `go test ./internal/org/ -run '^TestRunWatcher_TimeoutIndependentOfSmallInterval$' -count=3 -v` | PASS x3, ~1.23-1.35s each |

Both re-confirmed stable in isolation again this cycle, consistent with
every prior cycle's re-check (tester agent memory, [[flaky_test_notes]]).
Neither is in this cycle's delta.

### Coverage gaps / blind spots

No new gap introduced this cycle. `internal/cli` (80.0%) remains the
largest package with room below other packages' averages, consistent with
cycle 1's note that CLI command wiring is inherently harder to drive to
high coverage than pure logic packages. The cycle-2 delta's own coverage
(doctor strict-check paths) is the reason `internal/cli` rose rather than
stayed flat, so this is progress, not a new blind spot.

### What was NOT run (out of scope for `/test`)

- Static analysis — `/verify`'s job, already PASS in
  `docs/reports/verify-2026-08-19-overlay-scaffold-v2-p5.md` (cycle 2
  section).
- Documentation drift checks — `/sync-docs`'s job.
- Cross-model second opinion — `/cross-review`'s job.

### Recommendation

Tests pass. Cycle 2/2 (pipeline cycle cap reached). Proceed to
`/sync-docs`.

## Cycle 3

- Date: 2026-08-19
- Tester: `tester` subagent (Claude Code, Sonnet 5)
- Scope: pipeline cycle 3/3 (cap raised to 3 after cycle 2), re-running
  behavioral tests after the cycle-3 self-review fix delta on top of the
  cycle-2-passing base `1f50407` (615 shell + 8/8 Go, `internal/cli` 80.0%):
  `25b4f79` (cycle-2 cross-review AR#1-#4 fixes — strict-eligible Go error
  paths for `checkSettingsOwnedKeys`/`checkManagedBlocks`/`status.go`, +6 Go
  tests, plus shell path-scan cases G/H for the purity guard), `fd136ae`
  (unreadable-tracked-file defense-in-depth for the conflict-marker scan,
  +1 Go test), `6353a07` (C3 self-review fixes M1/L1-L6/L7(c) — purity-guard
  case H folded into case A, new case I, +1 Go test:
  `TestCheckScaffoldIntegrity_UnreadableSkipPathFace_ConflictScanBranchFires`),
  plus report-only commits (`e5329f4`, `71236d8`). HEAD at test time:
  `71236d8`.
- Evidence: `docs/evidence/verify-2026-08-19-063303.log` (raw `run-test.sh`
  output, exit 0)

### Verdict: PASS

617/617 shell tests across the same 27 files (+2 from cycle 2's 615, all of
it in `tests/test-template-purity.sh`, which grew from 8 to 10 cases — `+2`
net across `25b4f79`'s new path-scan cases G/H and `6353a07`'s fold of case
H into case A plus new case I; every other file's per-file count is
byte-identical to cycle 2), 8/8 Go packages `ok` (fresh `go test ./...
-count=1 -cover`, `go clean -testcache` run first, no stale cache), zero
failures anywhere. All 8 named `TestCheckScaffoldIntegrity_*`/
`TestStatusCmd_*` regression tests plus both flake-watchlist tests plus
both shell fixture suites individually re-run 3x in isolation with zero
flakes. No test weakening found.

`./scripts/run-test.sh` again selected `golang` as the sole changed-language
Go pack via the same full-fallback shell classification as cycles 1-2
(`==> Language scope: full fallback (unclassified:.claude/hooks/pre_bash_guard.sh)`),
but ran the complete 27-file shell suite regardless (27 `==> tests/` blocks,
zero non-zero `FAIL:` counts anywhere in the log), so shell scope was
effectively full.

### Test execution

| Suite / Command | Files/Packages | Passed | Failed | Exit |
| --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (default `changed` scope) | 27 shell test files + golang pack | 617 shell + 8/8 Go pkgs | 0 | 0 |
| `go test ./... -count=1 -cover` (fresh, `go clean -testcache` first) | 8 Go packages with tests | 8 | 0 | 0 |

Per-file shell delta from cycle 2 (all other 26 files byte-identical, not
re-tabulated here — see the Cycle 2 table above):

| File | Cycle 2 | Cycle 3 | Delta |
| --- | --- | --- | --- |
| test-template-purity.sh | 8 | 10 | +2 (new path-scan cases G/H added by `25b4f79`, then H folded into A and case I added by `6353a07`, net +2) |
| **Total (27 files)** | **615** | **617** | **+2** |

### Fresh Go coverage vs cycle-2 baseline

| Package | Cycle 2 | Cycle 3 (this run) | Delta |
| --- | --- | --- | --- |
| `internal/cli` | 80.0% | 80.7% | +0.7pp |
| `internal/config` | 94.2% | 94.2% | — |
| `internal/insights` | 86.1% | 86.1% | — |
| `internal/org` | 89.1% | 89.1% | — |
| `internal/org/driver` | 92.0% | 92.0% | — |
| `internal/org/protocol` | 97.9% | 97.9% | — |
| `internal/scaffold` | 75.7% | 75.7% | — |
| `internal/upgrade` | 91.2% | 91.2% | — |

`internal/cli` is the only package touched by the cycle-3 delta. `25b4f79`
added 6 new Go tests (`TestCheckScaffoldIntegrity_SettingsJSONInvalidJSON_StrictFails`,
`TestCheckScaffoldIntegrity_SettingsSnapshotCorrupt_StrictFails`,
`TestCheckScaffoldIntegrity_GitignoreTemplateMarkersBroken_ManagedBlocksStrictFails`,
`TestCheckScaffoldIntegrity_SettingsSnapshotUnreadable_OwnedKeysStrictFails`,
`TestCheckScaffoldIntegrity_SettingsJSONReplacedByDirectory_OwnedKeysStrictFails`,
`TestStatusCmd_ScaffoldSection_CorruptManifest_ShowsUnavailable`), `fd136ae`
added `TestCheckScaffoldIntegrity_UnreadableTrackedFile_StrictFails`, and
`6353a07` added `TestCheckScaffoldIntegrity_UnreadableSkipPathFace_ConflictScanBranchFires`
— 8 new tests total, rose +0.7pp. Every other package is byte-identical to
cycle 2, as expected — none of the cycle-3 delta commits touch their
source.

### Named regression tests (individually re-run 3x in isolation)

| Test | Command | Result (3/3) |
| --- | --- | --- |
| `TestCheckScaffoldIntegrity_SettingsJSONInvalidJSON_StrictFails` | `go test ./internal/cli/ -run '^TestCheckScaffoldIntegrity_SettingsJSONInvalidJSON_StrictFails$' -count=3 -v` | PASS x3, ~0.07-0.08s each |
| `TestCheckScaffoldIntegrity_SettingsSnapshotCorrupt_StrictFails` | `go test ./internal/cli/ -run '^TestCheckScaffoldIntegrity_SettingsSnapshotCorrupt_StrictFails$' -count=3 -v` | PASS x3, ~0.45-0.46s each |
| `TestCheckScaffoldIntegrity_GitignoreTemplateMarkersBroken_ManagedBlocksStrictFails` | `go test ./internal/cli/ -run '^TestCheckScaffoldIntegrity_GitignoreTemplateMarkersBroken_ManagedBlocksStrictFails$' -count=3 -v` | PASS x3, ~0.41-0.56s each |
| `TestCheckScaffoldIntegrity_SettingsSnapshotUnreadable_OwnedKeysStrictFails` | `go test ./internal/cli/ -run '^TestCheckScaffoldIntegrity_SettingsSnapshotUnreadable_OwnedKeysStrictFails$' -count=3 -v` | PASS x3, ~0.07-0.08s each |
| `TestCheckScaffoldIntegrity_SettingsJSONReplacedByDirectory_OwnedKeysStrictFails` | `go test ./internal/cli/ -run '^TestCheckScaffoldIntegrity_SettingsJSONReplacedByDirectory_OwnedKeysStrictFails$' -count=3 -v` | PASS x3, ~0.08-0.10s each |
| `TestStatusCmd_ScaffoldSection_CorruptManifest_ShowsUnavailable` | `go test ./internal/cli/ -run '^TestStatusCmd_ScaffoldSection_CorruptManifest_ShowsUnavailable$' -count=3 -v` | PASS x3, ~0.00s each |
| `TestCheckScaffoldIntegrity_UnreadableTrackedFile_StrictFails` | `go test ./internal/cli/ -run '^TestCheckScaffoldIntegrity_UnreadableTrackedFile_StrictFails$' -count=3 -v` | PASS x3, ~0.07s each |
| `TestCheckScaffoldIntegrity_UnreadableSkipPathFace_ConflictScanBranchFires` | `go test ./internal/cli/ -run '^TestCheckScaffoldIntegrity_UnreadableSkipPathFace_ConflictScanBranchFires$' -count=3 -v` | PASS x3, ~0.07-0.08s each |
| `tests/test-template-purity.sh` (10 cases incl. folded case H→A and new case I) | `sh tests/test-template-purity.sh` x3 (full 10-case file) | PASS x3, exit 0 each, 10/10 green every run |
| `tests/test-pre-bash-guard.sh` (AC-12b, jq-nested extraction, cases A-D) | `sh tests/test-pre-bash-guard.sh` x3 | PASS x3, exit 0 each, 4/4 green every run |

The two `TestCheckScaffoldIntegrity_UnreadableTrackedFile_StrictFails` /
`TestCheckScaffoldIntegrity_UnreadableSkipPathFace_ConflictScanBranchFires`
tests chmod a fixture file to `0000` to simulate an unreadable tracked
file; both `t.Skip` when running as root (chmod 0000 doesn't make files
unreadable to root) and both register `t.Cleanup` to restore `0644`
(`internal/cli/doctor_scaffold_test.go:283-331`) — confirmed self-cleaning,
and the full-suite run above (which includes these tests alongside every
other `internal/cli` test) completed with zero failures, so no cross-test
pollution was observed.

### Known-flaky watchlist (re-checked 3x isolated)

| Test | Command | Result (3/3) |
| --- | --- | --- |
| `TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess` | `go test ./internal/cli/ -run '^TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess$' -count=3 -v` | PASS x3, ~0.58-0.62s each |
| `TestRunWatcher_TimeoutIndependentOfSmallInterval` | `go test ./internal/org/ -run '^TestRunWatcher_TimeoutIndependentOfSmallInterval$' -count=3 -v` | PASS x3, ~1.21-1.35s each |

Both re-confirmed stable in isolation again this cycle, consistent with
every prior cycle's re-check (tester agent memory, [[flaky_test_notes]]).
Neither is in this cycle's delta.

### Coverage gaps / blind spots

No new gap introduced this cycle. `internal/cli` (80.7%) remains the
largest package with room below other packages' averages, consistent with
prior cycles' note that CLI command wiring is inherently harder to drive to
high coverage than pure logic packages. The cycle-3 delta's own coverage
(the class-closing audit of every `r.Status="warn"` site) is the reason
`internal/cli` rose rather than stayed flat, so this is progress, not a
new blind spot.

### What was NOT run (out of scope for `/test`)

- Static analysis — `/verify`'s job, already PASS in
  `docs/reports/verify-2026-08-19-overlay-scaffold-v2-p5.md` (cycle 3
  section).
- Documentation drift checks — `/sync-docs`'s job.
- Cross-model second opinion — `/cross-review`'s job.

### Recommendation

Tests pass. Cycle 3/3 (pipeline cycle cap reached). Proceed to
`/sync-docs`.
