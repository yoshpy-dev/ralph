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
