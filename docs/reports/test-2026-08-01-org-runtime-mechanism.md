# Test report: org-runtime-mechanism

- Date: 2026-08-01
- Plan: `docs/plans/active/2026-08-01-org-runtime-mechanism.md`
- Tester: `tester` subagent (Claude Code, `/test`)
- Scope: behavioral tests via `./scripts/run-test.sh` (`HARNESS_VERIFY_MODE=test`, changed-language scope requested; resolver fell back to **full** scope — see Notes) on branch `docs/spec-org-runtime`, HEAD `365f1ff`. No static analysis (delegated to `/verify`, already passed — see `docs/reports/verify-2026-08-01-org-runtime-mechanism.md`).
- Evidence: `docs/evidence/test-2026-08-01-org-runtime-mechanism.log` (full `run-test.sh` output), `docs/evidence/verify-2026-08-01-035849.log` (Go verifier sub-log referenced by the run), and supplementary race-detector run (see Coverage section).

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` — full shell suite (36 `tests/*.sh` scripts under the local verifier) | ~900+ (aggregate across all suites) | all | 0 | 0 | ~2 min |
| `go test ./...` (12 Go packages, run inside `run-test.sh`) | all packages | all | 0 | 0 | included above |
| `go test ./internal/org/... ./internal/cli/... ./internal/config/... -race -count=1 -v` (explicit request) | 181 top-level `Test*` funcs, 218 incl. subtests | 218 | 0 | 0 | ~16s (cli 16.5s, org 1.5s, org/driver 1.9s, config 2.1s) |

No `FAIL`, no `not ok`, and no `DATA RACE` occurrences anywhere in either run's output (`grep -c "DATA RACE"` = 0 on the race log; `grep -c "FAIL"` = 0 on both logs).

## Coverage

- Statement: `internal/org` 63.6%, `internal/org/driver` 96.8%, `internal/cli` 63.3–63.7% (combined `-coverpkg` run), `internal/config` 74.1%.
- Branch: not separately instrumented (Go's `go tool cover` reports statement coverage only; no branch-coverage tool configured in this repo — consistent with prior reports).
- Function: see gaps below — `Verbs.Send`, `Verbs.Wait`, `Verbs.Read`, `truncateForDetails` in `internal/org/verbs.go` are **0.0%** covered under both the package-local run and a combined `internal/org` + `internal/cli` `-coverpkg` run (i.e. not exercised even indirectly through the CLI test suite). `NewManifestStore`, `NewReceiptStore`, `Path()` accessors, and the package-level (non-`Store`) `Roster`/`ActiveSeatCount` convenience wrappers are also 0% — these are thin unused-by-tests wrapper functions, lower risk than the verb gap.
- Notes: Coverage was measured with `go test ./internal/org/... ./internal/cli/... ./internal/config/... -coverprofile=... -count=1` and cross-checked with `-coverpkg=./internal/org/...,./internal/cli/...` run from `internal/cli` to confirm CLI-level tests don't indirectly exercise the org-package verb gaps. Both runs agree.

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | No failures observed | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Self-review fix: herdr agent name not `org_id`-namespaced (commit `9bfe07e`) | Still fixed | `TestOrgSpawn_HerdrAgentNameNamespacedByOrgID`, `TestHerdrAgentName_NamespacesBySeatAndOrg` (`internal/org/spawn_test.go:184,210`) — both pass under `-race`. |
| Self-review fix: dry-run events could override real seat state in `status --all` | Still fixed | `TestRoster_DryRunDisbandLeavesRealSeatsActive`, `TestRoster_RealDisbandLeavesDryRunSeatsUntouched`, `TestRoster_DryRunStoppedDoesNotDeactivateRealSeatWithSameID` (`internal/org/manifest_test.go:204,225,246`) — all pass. |
| AC-9 compat: `[org]`-less `ralph.toml` loads without warning; existing tests still pass with no new warnings | Confirmed | `TestLoad_OrgMissingSection` (`internal/config/config_test.go:457`) passes; full `internal/config` suite (74.1% cov) green with no new skips. |
| `defaults_sync_test.go` drift tripwire (config.go / templates/base/ralph.toml / ralph-config.sh 3-way lockstep) | Green | `TestDefaultsLockStep` passes (part of the `-race` run); `run-test.sh`'s `check-sync.sh`/`check-skill-sync.sh` gates also pass. |
| Existing (pre-PR) test suites: no new warnings/regressions from `[org]` addition | Confirmed | All 36 shell test suites and all 12 Go packages report 0 failures; `internal/ui`, `internal/state`, `internal/action`, `scripts/ralph-orchestrator.sh`, `scripts/ralph-pipeline.sh` paths (explicitly untouched per plan) show no behavior change. |

## AC coverage mapping (named tests from the verify report, confirmed to exist and pass)

| AC | Named test(s) | Location | Result |
| --- | --- | --- | --- |
| AC-1 (reject out-of-pool model) | `TestOrgSpawn_Rejected_OutOfPoolModel` | `internal/org/spawn_test.go:254` | PASS |
| AC-2 (max_seats org-isolated) | `TestOrgSpawn_Rejected_MaxSeatsReached_OrgIsolated` | `internal/org/spawn_test.go:285` | PASS |
| AC-3 (idempotent respawn) | `TestOrgSpawn_Idempotent_AlreadySpawned_NoNewDriverCalls`, `TestOrgSpawn_StaleInFlight_CompensatesThenRespawnsFresh` | `internal/org/spawn_test.go:309,330` | PASS |
| AC-4 (status works with everything stopped; dry-run excluded by default) | `TestRoster_DryRunExclusionAndInclusion`, `TestManifestStore_Read_SkipsAndCountsCorruptLines` | `internal/org/manifest_test.go:95,57` | PASS |
| AC-5 (doctor reports herdr/agmsg absence; `--probe-models` warns) | `internal/cli/doctor_org_test.go` (all funcs) | `internal/cli/doctor_org_test.go` | PASS |
| AC-6 (`[org]` defaults 3-way lockstep) | `TestDefaultsLockStep` + `go test ./internal/config/...` | `internal/config/defaults_sync_test.go` | PASS |
| AC-7 (`go test ./...` and `run-verify.sh` green; no diff to untouched files) | full `go test ./...` (inside `run-test.sh`) | all packages | PASS — 0 failures across 12 packages |
| AC-8 (all mutating verbs support `--dry-run`) | `TestOrgSpawn_DryRun_NoDriverCalls_EventsFlaggedAndExcludedByDefault`, `TestOrgSpawn_DryRun_NoPATHNeeded_StatusExclusionAndAll` | `internal/org/spawn_test.go:465`, `internal/cli/org_test.go:425` | PASS |
| AC-9 (compat: missing `[org]` loads clean; doctor exit code unaffected) | `TestLoad_OrgMissingSection` | `internal/config/config_test.go:457` | PASS |
| AC-10 (saga failure injection → `spawn_failed` + compensation) | `TestOrgSpawn_FailureInjection_WorkspaceCreate_NoCompensationAttempted`, `TestOrgSpawn_FailureInjection_TabCreate_NoCompensationAttempted`, `TestOrgSpawn_FailureInjection_AgentStart_CompensatesExistingPane`, `TestOrgSpawn_FailureInjection_AgmsgSend_CompensatesExistingPane` (+ CLI-level equivalents `TestOrgSpawn_FailureInjection_TabCreate_NoCompensation`, `TestOrgSpawn_FailureInjection_AgentStart_CompensatesPane`, `TestOrgSpawn_FailureInjection_AgmsgSend_CompensatesPane`) | `internal/org/spawn_test.go:364,390,413,439`; `internal/cli/org_test.go:242,274,309` | PASS |

All named tests cited by the verify report were confirmed to exist (`grep -rn "func Test..."`) before execution, and all passed both in the default `run-test.sh` run and under `-race`.

## Test gaps

- **`ralph org send` / `ralph org wait` / `ralph org read` have zero test coverage** — `Verbs.Send`, `Verbs.Wait`, `Verbs.Read` (`internal/org/verbs.go:54,131,156`) and their CLI wiring (`internal/cli/org.go` `newOrgSendCmd`/`newOrgWaitCmd`/`newOrgReadCmd`) are 0.0% covered under both an `internal/org`-only run and a combined `-coverpkg=./internal/org/...,./internal/cli/...` run from `internal/cli`. No unit or CLI-level test exercises these three verbs at all (confirmed via `grep -n "func Test" internal/cli/org_test.go` — only spawn/status/disband are covered). This is a real, non-trivial gap: `send`/`wait`/`read` are 3 of the 7 verbs in the plan's scope and are exercised by no test in this PR. Recommend adding at minimum: `Send` happy-path + stub-agmsg-failure case, `Wait` happy-path + timeout case, `Read` happy-path + seat-not-found case, using the same stub herdr/agmsg pattern already established in `spawn_test.go`/`org_test.go`.
- `truncateForDetails` helper (`internal/org/verbs.go:104`) — also 0% covered; low risk (pure string helper) but untested.
- `NewManifestStore`/`NewReceiptStore`/their `Path()` accessors, and the package-level (non-method) `Roster`/`ActiveSeatCount` wrapper functions — 0% covered. Self-review already flagged these as dead code candidates (see verify report's "not asked to fix" list); not a behavioral risk, just untested dead code.
- Per the verify report: `agmsgArgs` CLI flag shape (`--team`/`--as`) remains an unverified assumption against the real agmsg CLI — no live-binary integration test exists or is expected in this PR (CI has no herdr/agmsg per plan Assumptions); tracked in `docs/tech-debt/README.md`.
- Two-org concurrency for the herdr-namespace fix is proven only by a synthetic unit test with a fake `HerdrClient` (`TestOrgSpawn_HerdrAgentNameNamespacedByOrgID`) — no live herdr binary exercised. Expected per plan Assumptions, not a defect.

## Verdict

- Pass: all 218 `Test*` funcs (incl. subtests) across `internal/org`, `internal/org/driver`, `internal/cli`, `internal/config` pass under `-race`; full `run-test.sh` (36 shell suites + `go test ./...` across 12 packages) passes with 0 failures.
- Fail: none.
- Blocked: none.

**Overall verdict: PASS.** All 10 named tests (and their variants) cited by the verify report as delegated proof for AC-1 through AC-10 exist and pass. The race detector found no data races in the new `internal/org`, `internal/org/driver`, `internal/cli`, or `internal/config` code. One coverage gap is flagged (send/wait/read verbs untested) — not a blocking regression since these verbs are out of this PR's AC set for behavioral proof, but should be added before or alongside PR② (seat activation), which will exercise these paths live.

## Notes

- `run-test.sh` reported `Language scope: full fallback (unclassified:scripts/ralph-config.sh)` rather than the requested changed-language (`golang`) scope — same fallback behavior the verify report observed on the same diff (an unclassified shell diff in `scripts/ralph-config.sh` forces full-language scope in the language detector). This is expected/known behavior, not a test-runner defect, and it means the *entire* shell test suite ran (not just Go), which is a superset of the requested scope — strictly more coverage, not less.
