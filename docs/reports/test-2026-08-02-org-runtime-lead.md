# Test report: org-runtime-lead

- Date: 2026-08-02
- Plan: `docs/plans/active/2026-08-02-org-runtime-lead.md`
- Tester: `tester` subagent (`/test`)
- Scope: `./scripts/run-test.sh` (changed-language scope; escalated to golang full-package fallback because `scripts/ralph-config.sh` is unclassified by language detection) + targeted `go test ./internal/org/... ./internal/cli/... ./internal/config/... -race -count=1`. Behavioral tests only — no static analysis (that is `/verify`'s scope, already reported in `docs/reports/verify-2026-08-02-org-runtime-lead.md`).
- Evidence: `docs/evidence/test-2026-08-02-org-runtime-lead.log` (run-test.sh full shell+go output, followed by the `-race -count=1 -v` targeted run); also `docs/evidence/verify-2026-08-02-124720.log` (run-test.sh's own auto-log)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` — 37 shell suites (`tests/test-*.sh`) | 978 assertions (aggregated `PASS`/`FAIL:` lines) | 978 | 0 | 0 | ~ (see log) |
| `./scripts/run-test.sh` — golang verifier (`go test ./...`, all 13 test-bearing packages, cached where unchanged) | 13 packages | 13 ok | 0 | 0 (2 known scaffold skips not re-triggered, unrelated to this branch) | see log |
| `go test ./internal/org/... ./internal/cli/... ./internal/config/... -race -count=1 -v` | 456 `=== RUN` entries (incl. subtests) | 345 top-level `--- PASS` (remainder are subtests rolled into parents / package-level `ok`) | 0 | 0 | `internal/org` 3.017s, `internal/org/driver` 1.852s, `internal/org/protocol` 1.406s, `internal/cli` 20.433s, `internal/config` 2.898s |

No `--- FAIL`, no `DATA RACE` in the `-race` run. All 5 packages reported `PASS` / `ok`.

## Coverage

- `internal/org`: **90.0%** of statements (`go tool cover -func`, fresh run this cycle) — up from 86.7% at the org-runtime-seats (PR②) baseline. New coverage from this branch's slices: `report.go` (`BuildOrgReport` 98.2%, `reportDate` 66.7%, `filterEventsByOrg`/`filterReceiptsByOrg`/`permissionModeFromDetails` 100%), `permissions.go` (argv-mapping + codex fail-closed, covered by `permissions_test.go`), `spawn.go` additions (`compensateLeave` 80.0%, `agmsgTypeForDriver` 75.0%, minimum-control-gate branches in `Spawn` 86.8%).
- `internal/org/driver`: 92.0% (unchanged from PR② baseline).
- `internal/org/protocol`: 97.9% (unchanged from PR② baseline).
- `internal/cli`, `internal/config`: not independently re-profiled this cycle (no `-coverprofile` requested for these two in the plan's test plan); both pass under `-race -count=1` with the new `org start` / `org report` / permission-mode CLI tests included.
- Branch/function: not separately measured (Go tooling reports statement coverage only in this repo's convention; see `go_test_packages.md` memory for per-func breakdown used above).
- Notes: lowest-covered functions in `internal/org` remain `truncateForDetails` (66.7%) and `Disband` (78.6%) — both pre-existing gaps from PR②, not touched by this branch's ACs, not blocking.

## AC → behavioral test mapping

| AC | Claim | Representative tests | File |
| --- | --- | --- | --- |
| AC-1 | `[org.permissions]` 3-way lockstep, defaults_sync_test detects drift | `TestDefaultsLockStep` (extended), `TestLoad_...` permission tests | `internal/config/config_test.go`, `internal/config/defaults_sync_test.go` |
| AC-2 | claude argv gets `--permission-mode` per mode (3-mode); codex rejects non-`guarded` | `TestPermissionArgsForDriver_Claude` (3 modes), `TestPermissionArgsForDriver_Codex_FailClosed`, `TestResolvePermissionMode`, `TestOrgSpawn_PermissionMode_Claude_ArgvMapping`, `TestOrgSpawn_Codex_AutonomousDefault_RejectedFailClosed_WithReceipt`, `TestOrgSpawn_Codex_GuardedRoleOverride_Spawns` | `internal/org/permissions_test.go`, `internal/org/spawn_test.go` |
| AC-2b | autonomous spawn fails closed without `--scope`; `--allow-unscoped` records escape hatch; applied mode recorded on `spawned` event | `TestOrgSpawn_MinimumControlGate_Autonomous_EmptyScope_RejectedWithEvent`, `TestOrgSpawn_MinimumControlGate_EditsAndGuarded_EmptyScope_Proceeds`, `TestOrgSpawn_MinimumControlGate_DryRun_AppliesSameGate`, `TestOrgSpawn_ScopeRecordedOnSpawnedEventDetails`, `TestOrgSpawn_NoScope_SpawnedEventDetailsOmitsScopeFragment`, `TestOrgStart_MissingScope_AutonomousDefault_RejectedUnlessAllowUnscoped` (rejected-event audit trail covered by the `o.reject(...)` post-fix path per verify report row) | `internal/org/spawn_test.go`, `internal/cli/org_test.go` |
| AC-3 | `ralph org start` spawns role=lead, expands lead.md with task/envelope | `TestOrgStart_HappyPath_SpawnsLeadSeat_SingleAgmsgJoin_NoHello`, `TestOrgStart_TaskAndEnvelopeLandInPromptFile`, `TestOrgStart_ModelFlagOmitted_DefaultsToFirstMatchingPoolEntry`, `TestOrgStart_ModelFlagExplicit_OverridesDefault`, `TestOrgStart_NoMatchingModelPoolEntryForDriver_NonZeroExit`, `TestOrgStart_BlankTaskArg_NonZeroExit`, `TestOrgStart_RequiresCwd`, `TestOrgStart_RequiresOrgID`; lead-self-spawn mechanism: `TestOrgSpawn_LeadSelfSpawn_SingleAgmsgJoin_NoHelloSend`, `TestOrgSpawn_LeadSelfSpawn_DryRun_MirrorsSameSkip`, `TestOrgSpawn_LeadRole_TaskAndEnvelopeSubstitutedIntoPromptFile` | `internal/cli/org_test.go`, `internal/org/spawn_test.go` |
| AC-4 | `ralph org report` generates manifest+receipts history to `docs/reports/` | `TestBuildOrgReport_RosterTimelineReceiptsAndResiduals`, `TestBuildOrgReport_EmptyOrg_NoEventsNote`, `TestOrgReport_WritesFileFilteredByOrgAndUsesInjectedClock`, `TestOrgReport_DefaultOutDir_WritesUnderDocsReports`, `TestOrgReport_EmptyOrg_StillWritesReportWithNoEventsNote`, `TestOrgReport_CLI_WritesFileWithRosterTimelineAndReceipts`, `TestOrgReport_CLI_EmptyOrg_StillWritesReport`, `TestOrgReport_CLI_RequiresOrgID` | `internal/org/report_test.go`, `internal/cli/org_test.go` |
| AC-6 (tech-debt #1) | best-effort `Leave` on announce failure, recorded in Details | `TestOrgSpawn_FailureInjection_AgmsgSend_LeavesJoinedSeat`, `TestOrgSpawn_FailureInjection_AgmsgSend_LeaveFailure_RecordedInDetails` (internal/org); `TestOrgSpawn_FailureInjection_AgentStart_CompensatesPane` extended for `leave=ok` (internal/cli) | `internal/org/spawn_test.go`, `internal/cli/org_test.go` |
| AC-7 (tech-debt #2) | `LeadIdentity` const; agmsg type derived from lead driver | `TestLeadIdentity_ConstantValue`, `TestOrgSpawn_EnsureLeadJoined_DefaultLeadDriver_ClaudeCodeType`, `TestOrgSpawn_EnsureLeadJoined_LeadDriverCodex_UsesCodexAgmsgType` | `internal/org/spawn_test.go` |
| AC-8 (tech-debt #3) | `herdr_agent_name` persisted; send/wait/stop prefer recorded value, fallback for legacy events | `TestOrgSpawn_SpawnedEventRecordsHerdrAgentName` (write side), `TestOrgSend_PrefersRecordedHerdrAgentName_OverDerivedConvention`, `TestOrgSend_LegacySpawnedEvent_FallsBackToDerivedHerdrAgentName`, `TestResolvedHerdrAgentName_EmptyVsSet` (read side + compat), `TestOrgWait_HappyPath_ReturnsHerdrOutputAndTargetsNamespacedAgent`, `TestOrgWait_UnknownSeat_StillDrivesHerdr_NoManifestCheck` (Wait's documented derive-only deviation) | `internal/org/verbs_test.go` |
| AC-9 (tech-debt #4) | concurrent spawn serialized via flock, no `max_seats` overshoot | `TestOrgSpawn_ConcurrentSpawns_MaxSeatsNeverExceeded` (10 goroutines vs. `max_seats=3`), `TestWithManifestLock_SerializesConcurrentCallers` | `internal/org/spawn_test.go` |
| AC-10 (tech-debt #5/#6) | doctor names resolved agmsg home; dry-run propagates prompt-render errors | doctor: covered under `internal/cli` package tests for `checkAgmsgAvailable` (see `internal/cli/doctor_org_test.go`); dry-run: `TestOrgSpawn_DryRun_PromptFilePathError_FailsStep`, `TestOrgSpawn_DryRun_PromptFilePathError_NoManifestEventForRealSeat` | `internal/cli/doctor_org_test.go`, `internal/org/spawn_test.go` |
| AC-11 / AC-11b | live smoke (bypass-dialog-free bash, headless lead E2E) | **Live-evidence-only**, no automated test — see `docs/evidence/org-lead-smoke-2026-08-02.txt` and `docs/reports/verify-2026-08-02-org-runtime-lead.md`'s Observational checks / Coverage gaps sections. Not re-verified by this report (behavioral test suite has no substitute for a live herdr/agmsg pane run). |
| AC-12 | `go test ./...` green | This report's execution table above — 13/13 packages `ok`, 0 FAIL, 0 SKIP (beyond the 2 pre-existing scaffold skips), 0 DATA RACE on the targeted `-race` run. Test half of AC-12 is now confirmed; static half was already confirmed PASS in the verify report. |

`Wait`'s "idle,done" default (self-review HIGH-1 fix, commit `9a22942`) is covered by `TestOrgWait_DefaultUntilAndTimeout_AreBoundedAndDone` and `TestOrgWait_ExplicitZeroTimeout_StaysUnbounded` in `internal/cli/org_test.go`.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | No failures observed in this run. | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| PR① / PR② (`org-runtime-mechanism`, `org-runtime-seats`) full test suite | Still green | `go test ./...` (all 13 packages, run-test.sh) reports `ok` for every package including `internal/org`, `internal/org/driver`, `internal/org/protocol` from the prior two PRs |
| `defaults_sync_test.go` (3-surface model/effort/permission-mode lockstep) | Still green | Included in `internal/config` package pass; no drift reported |
| `check-sync.sh` / `check-skill-sync.sh` drift regressions | Out of `/test` scope (static/doc-drift, already confirmed PASS in verify report) | `docs/reports/verify-2026-08-02-org-runtime-lead.md` Static analysis table |
| Self-review fix commit `9a22942` (wait idle/done default, lock two-phase reshape, permission-mode constants) | Behaviorally proven, not just statically clean | `TestOrgWait_DefaultUntilAndTimeout_AreBoundedAndDone` (wait defaults), `TestOrgSpawn_ConcurrentSpawns_MaxSeatsNeverExceeded` + `TestWithManifestLock_SerializesConcurrentCallers` (two-phase lock still serializes correctly), `TestOrgSpawn_PermissionMode_Claude_ArgvMapping` (constants used correctly at the gate) |

## Test gaps

- AC-11 / AC-11b remain **live-evidence-only**: no automated test substitutes for a real herdr/agmsg pane interaction (bypass-permissions dialog behavior, a live headless lead session). This is an accepted, documented gap per the plan's own design (real terminal automation is out of scope) and is carried in the verify report's Coverage gaps as "evidenced, not independently re-executed."
- `internal/org` `truncateForDetails` (66.7%) and `Disband` (78.6%) remain the lowest-covered functions in the package — pre-existing from PR②, not named by any AC in this plan, not a regression.
- `internal/cli` and `internal/config` coverage percentages were not independently re-profiled with `-coverprofile` this cycle (only `internal/org` and its subpackages were profiled per the plan's evidence-capture list); both pass under `-race`, so behavioral correctness is confirmed even without a fresh percentage snapshot.
- No dedicated test exercises the interaction between `--allow-unscoped` and the manifest-recorded "escape hatch used" audit trail under a genuinely concurrent spawn (AC-2b + AC-9 combined edge case) — each is tested independently but not in combination. Low risk: both mechanisms are independent gates in the code (scope check happens before the lock-protected section), so a combined test would mostly be confirming test infrastructure, not new production risk.

## Verdict

- **Pass** — `./scripts/run-test.sh` (37 shell suites, 978 assertions) and the targeted `go test ./internal/org/... ./internal/cli/... ./internal/config/... -race -count=1 -v` (5 packages, 345 top-level test functions incl. subtests) both report 0 failures, 0 skips (beyond the 2 known pre-existing scaffold-embed skips, unrelated to this branch), 0 data races.
- Fail: none.
- Blocked: none.
- AC-12's test half is now confirmed green, completing the AC (static half already green per `docs/reports/verify-2026-08-02-org-runtime-lead.md`).
- AC-11 / AC-11b remain live-evidence-only per plan design — not re-verifiable by an automated behavioral test suite in this pass; unchanged from the verify report's own framing.
- Proceed to `/sync-docs` — tests pass, no blocking failures.

## Cycle 2

- Date: 2026-08-02
- Tester: `tester` subagent (`/test`, pipeline cycle 2)
- Trigger: cross-review AR#1 fix (spawn ordering) + cycle-2 self-review M1/M2 fixes, landed as commits `de4de50` (fix: return idempotent spawn before autonomous scope gate) and `69be944` (fix: align dry-run gate ordering and recheck idempotency under phase-2 lock). Both commits add net-new regression tests to `internal/org/spawn_test.go` (no other test files touched in this delta).
- Scope: same as cycle 1 — `./scripts/run-test.sh` (changed-language scope; escalates to golang full-package fallback since `scripts/ralph-config.sh` is unclassified by language detection) + targeted `go test ./internal/org/... ./internal/cli/... ./internal/config/... -race -count=1`.
- Evidence: `docs/evidence/test-2026-08-02-org-runtime-lead-cycle2.log` (`run-test.sh` full shell+go output), `docs/evidence/test-2026-08-02-org-runtime-lead-cycle2-race.log` (targeted `-race -count=1 -v` run), `docs/evidence/verify-2026-08-02-132746.log` (`run-test.sh`'s own auto-log)

### New regression tests (mapped to AR#1 / M1 / M2)

| Test | File | Maps to | What it locks down |
| --- | --- | --- | --- |
| `TestOrgSpawn_Idempotent_NoScopeRetry_ReturnsExistingSeat` | `internal/org/spawn_test.go:450` | cross-review AR#1 | A retried spawn for an already-`spawned` seat returns the existing seat idempotently even when called without `--scope`, instead of being rejected by the autonomous-mode scope gate — proves the idempotency check now runs *before* the scope gate. |
| `TestOrgSpawn_DryRun_And_Real_AgreeOnRejectionCause_EnvelopeBeforeScopeGate` | `internal/org/spawn_test.go:745` | cycle-2 self-review M1 | Dry-run and real spawn paths agree on rejection cause when both the envelope check and the scope gate could fire — envelope validation is asserted to short-circuit ahead of the scope gate in both paths, so dry-run planning doesn't diverge from what a real spawn would reject on. |
| `TestOrgSpawn_StaleInFlight_RacerCompletesDuringCompensationWindow_Phase2ReturnsIdempotent` | `internal/org/spawn_test.go:681` | cycle-2 self-review M2 | If a racing spawn attempt completes (writes its `spawned` event) during another caller's phase-2 compensation window for a stale in-flight seat, the second caller's phase-2 re-check under the manifest lock now correctly observes the completed seat and returns idempotent, instead of proceeding to double-spawn or clobbering the completed seat's state. |

All three: confirmed present via `grep -n` against `internal/org/spawn_test.go` before running tests, and confirmed passing individually via `--- PASS:` lines in `docs/evidence/test-2026-08-02-org-runtime-lead-cycle2-race.log:150,161-164`.

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` — 37 shell suites (`tests/test-*.sh`) | 978 assertions (aggregated `PASS`/`FAIL:` lines) | 978 | 0 | 0 | ~ (see log) |
| `./scripts/run-test.sh` — golang verifier (`go test ./...`, all 13 test-bearing packages) | 13 packages | 13 ok | 0 | 0 (2 known pre-existing scaffold skips, unrelated) | see log |
| `go test ./internal/org/... ./internal/cli/... ./internal/config/... -race -count=1 -v` | 459 `=== RUN` entries (incl. subtests) | 348 top-level `--- PASS` | 0 | 0 | `internal/org` 2.351s, `internal/org/driver` 1.880s, `internal/org/protocol` 2.117s, `internal/cli` 20.396s, `internal/config` 3.091s |

No `--- FAIL`, no `DATA RACE` in the `-race` run. All 5 packages reported `PASS` / `ok`. Shell-suite assertion count (978) and package count (13 ok / 0 FAIL) are unchanged from cycle 1; the `-race` run's top-level test count rose from 345 to 348 (the 3 new regression tests), consistent with the delta.

### Coverage delta

- `internal/org`: 90.3% combined-package statement coverage this cycle (`go tool cover -func` on a fresh `-coverprofile` covering `internal/org`, `internal/org/driver`, `internal/org/protocol` together) vs. 90.0% (`internal/org` alone) reported in cycle 1. Effectively flat — the two fix commits added ~250 lines to `spawn.go` (idempotency-before-scope-gate reordering, dry-run/real rejection-cause alignment, phase-2 re-check under lock) fully exercised by the 3 new tests plus existing coverage; no net coverage regression.
- `internal/org/driver` (92.0%) and `internal/org/protocol` (97.9%): unchanged from cycle 1 and PR② baseline — not touched by this delta.
- Lowest-covered functions remain the same pre-existing gaps noted in cycle 1 (`truncateForDetails` 66.7%, `Disband` 78.6%, plus zero-covered simple constructors `NewManifestStore`/`NewReceiptStore`/`Roster`/`Path` that are exercised indirectly through other test files not included in this narrower coverage run) — none touched by this branch's ACs or by AR#1/M1/M2.

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | No failures observed in this run. | — |

### Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Cycle 1 full suite (978 shell assertions, 13 Go packages, 345 targeted tests) | Still green after the 2 new fix commits | `docs/evidence/test-2026-08-02-org-runtime-lead-cycle2.log` shows identical 978/978 shell assertions and 13/13 `ok` packages |
| Pre-fix spawn ordering bug (AR#1: scope gate ran before idempotency check) | Fixed and now regression-tested | `TestOrgSpawn_Idempotent_NoScopeRetry_ReturnsExistingSeat` |
| Pre-fix dry-run/real rejection-cause divergence (M1) | Fixed and now regression-tested | `TestOrgSpawn_DryRun_And_Real_AgreeOnRejectionCause_EnvelopeBeforeScopeGate` |
| Pre-fix stale in-flight race window (M2) | Fixed and now regression-tested | `TestOrgSpawn_StaleInFlight_RacerCompletesDuringCompensationWindow_Phase2ReturnsIdempotent` |

### Test gaps (unchanged from cycle 1)

- AC-11 / AC-11b remain live-evidence-only (unchanged).
- `truncateForDetails` / `Disband` remain the lowest-covered pre-existing functions (unchanged).
- The combined AC-2b + AC-9 concurrent-spawn-with-escape-hatch edge case remains untested in combination (unchanged, low risk per cycle 1's assessment).

### Verdict (Cycle 2)

- **Pass** — all 3 regression tests mapped to cross-review AR#1 and cycle-2 self-review M1/M2 exist, run, and pass. Full `./scripts/run-test.sh` (978 shell assertions, 13/13 Go packages) and the targeted `-race -count=1` run (348 top-level tests across 5 packages) report 0 failures, 0 skips beyond the 2 known pre-existing scaffold skips, 0 data races.
- Fail: none.
- Blocked: none.
- Coverage is flat (90.3% vs 90.0% baseline for `internal/org`), no regression.
- Proceed to `/sync-docs` — tests pass, no blocking failures.
