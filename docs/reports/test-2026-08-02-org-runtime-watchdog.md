# Test report: org-runtime-watchdog

- Date: 2026-08-02
- Plan: `docs/plans/active/2026-08-02-org-runtime-watchdog.md`
- Tester: `tester` subagent (behavioral tests only, no static analysis)
- Scope: `git diff e7a32b9...HEAD` on `feat/org-runtime-watchdog` (HEAD `6a09e64`). Handed off from `docs/reports/verify-2026-08-02-org-runtime-watchdog.md` (PASS with two follow-up items).
- Evidence: `docs/evidence/test-2026-08-02-org-runtime-watchdog.log` (gitignored per `docs/evidence/*.log`, not committed)

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (30 shell suites, `tests/*.sh`) | ~761 assertions | ~761 | 0 | 0 | ~1 min |
| `./scripts/run-test.sh` → Go verifier, `go test ./...` (full-scope fallback, 13 packages) | 13 packages | 13 (all `ok`) | 0 | 0 | ~35s (mixed cached/fresh) |
| `go test ./internal/org/... ./internal/cli/... ./internal/config/... -race -count=1 -v` | 393 subtests, 5 packages | 393 | 0 | 0 | ~30s |

`./scripts/run-test.sh` fell back to full language scope because `scripts/ralph-config.sh` is classified "unclassified" by `detect-languages.sh` (same fallback the verify report observed) — not an override. This means `go test ./...` ran across the entire module, not just `internal/org`, and every package reported `ok`.

No test failures, no skips, no flakes observed across two independent runs (once inside `run-test.sh`, once standalone with `-race`).

## Coverage

- `internal/org`: 88.1% of statements (`go test ./internal/org/... -coverprofile`)
- `internal/org/driver`: 92.0%
- `internal/org/protocol`: 97.9%
- `internal/cli`: 67.3% (module-wide; not watchdog-specific — `org.go`'s `watch` subcommand wiring is exercised indirectly via `internal/org` unit tests, not `internal/cli` package tests)
- `internal/config`: 77.1% (covers `[org.watchdog]` `Default()`/`Load()`/lockstep validation added for AC-1)
- Branch/function: not measured separately; `go tool cover -func` statement-level figures above are the available granularity.
- Notes: coverage is comparable to the prior cycle's org baseline (memory: 86.7%→90.0% from an earlier PR); this PR's new files land at 88.1% package-wide, pulled down by a few real-side-effect functions intentionally left untested at the unit level (see Test gaps).

## AC-to-test mapping

| AC | Behavior | Covering test(s) |
| --- | --- | --- |
| AC-1 | `[org.watchdog]` 3-face lockstep | `internal/config` `defaults_sync_test.go` (drift tripwire, part of the 13-package full run) |
| AC-2 | `ALERT` protocol type | `internal/org/protocol` package tests (97.9% cov, part of full run) |
| AC-3 | Seat budget cutoff + `Reason` + ALERT, cutoff only on `Stop` success | `TestWatch_SeatBudgetCutoff_AtBoundary_NotBeforeThenCutoffThenDeduped`, `TestWatch_SeatBudgetCutoff_StopFails_RetriesThenSucceeds` |
| AC-3b | Total budget cuts all active seats, org-level record | `TestWatch_TotalBudgetCutoff_CutsAllActiveSeats_OneOrgLevelAlert` |
| AC-3c | Idempotency / dedupe across cycles | `TestWatch_SeatBudgetCutoff_AtBoundary_NotBeforeThenCutoffThenDeduped`, `TestWatch_Stall_AlertsNoCutoff_RecoversAndRefires` |
| AC-4 | Stall / liveness / scope-change ALERT only, no cutoff | `TestWatch_Stall_AlertsNoCutoff_RecoversAndRefires`, `TestWatch_Liveness_AlertOnAgentGetError`, `TestWatch_ScopeChange_AlertCarriesScopeText_NoCutoff`, `TestWatch_Stall_UsesLatestEventOfAnyType_NotOnlyStateEvents` (M-6 regression) |
| AC-5 | Deadman: 3-source OR, escalate on timeout/no-activity/lead-is-subject; watchdog's own event excluded from activity | `TestWatch_Deadman_NoActivity_EscalatesOnceAfterTimeout`, `TestWatch_Deadman_LeadActivity_PreventsEscalation`, `TestWatch_Deadman_LeadIsAnomalySubject_EscalatesImmediately`, `TestWatch_Deadman_WatchdogsOwnCutoffEvent_DoesNotClearPendingAlert` (M-4 regression) |
| AC-6 | On-demand watcher: non-blocking, hang→`watcher_error`, budget cutoff unaffected by hang, honored tri-state | `TestRunWatcher_ValidJSON_NormalVerdict`, `_CircularVerdict`, `TestRunWatcher_ReportedModel_HonoredTrue`, `_MismatchHonoredFalse`, `TestRunWatcher_FencedJSON_Tolerated`, `TestRunWatcher_MalformedEnvelope_WatcherErrorReceiptAndError`, `_MalformedVerdictJSON_...`, `_UnknownVerdictValue_TreatedAsMalformed`, `TestRunWatcher_Timeout_BoundedAndWatcherErrorReceipt`, `TestRunWatcher_TimeoutIndependentOfSmallInterval`, `TestNewWatchdogHooks_Dispatch_NeverBlocksCaller`, `TestNewWatchdogHooks_SingleFlight_SecondTriggerSkippedWhileBusy`, `TestWatcherInvokeTimeout_DefaultIsSixtySeconds` |
| AC-7 | state-dir resolution: flag > env > git toplevel > cwd (5 cases) | `TestResolveOrgStateDir_ExplicitFlagWins`, `_EnvWinsOverGitAndCwd`, `_GitSubdirResolvesToToplevel`, `_GitRoot`, `_NonGitCwdFallsBackToCwd` |
| AC-8 | codex permission fail-closed / verified unlock | `TestPermissionArgsForDriver_Codex_FailClosed`, `_VerifiedUnlocksMapping`, `TestPermissionArgsForDriver_Codex_UnknownMode_DistinctError` (L-7 regression) |
| AC-9 | evidence redaction convention + LOW batch (signature cleanups) | No dedicated unit test needed (doc/signature-only changes); confirmed by reading `docs/quality/definition-of-done.md` and `docs/evidence/org-watchdog-smoke-2026-08-02.txt` per the verify report; `checkCapacityAndStart`/`newOrgRuntime` signature changes are exercised transitively by all `internal/org` spawn/verb tests continuing to pass. |
| AC-10 | Live smoke (real-machine) | **Out of scope for `/test`** — this is a live-evidence-only AC per the plan/verify report; `docs/evidence/org-watchdog-smoke-2026-08-02.txt` is the artifact, not a `go test` target. Confirmed the evidence file exists and is referenced. |
| AC-11 | `go test ./...` / `./scripts/run-verify.sh` green | **This report's own scope** — both `./scripts/run-test.sh` (full-scope `go test ./...`, 13 packages) and the explicit `-race` run are green. `./scripts/run-verify.sh` itself (static half) was already confirmed PASS by the verify report; not re-run here per the `/verify`-vs-`/test` split. |

## Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | No failures observed | — |

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| H-1: shared `watch-status.json` across orgs (self-review finding) | Fixed, regression-tested | `TestRunWatch_MultipleOrgs_SeparateStatusFiles` passes |
| H-2: cutoff ratchet set even when `Stop` fails (self-review finding) | Fixed, regression-tested | `TestWatch_SeatBudgetCutoff_StopFails_RetriesThenSucceeds` passes |
| M-3: `Honored` tri-state ignored model mismatch (self-review finding) | Fixed, regression-tested | `TestRunWatcher_ReportedModel_MismatchHonoredFalse` passes (previously unreachable `HonoredFalse` branch) |
| M-4: deadman activity source counted watchdog's own events (self-review finding) | Partially fixed, narrow regression-tested (see Test gaps) | `TestWatch_Deadman_WatchdogsOwnCutoffEvent_DoesNotClearPendingAlert` passes |
| M-5: `--once` didn't join the in-flight watcher goroutine (self-review finding) | Fixed, regression-tested | `TestNewWatchdogHooks_*` assert on `wg.Wait()` |
| M-6: stall time term used state-only `SeatStatus.TS` (self-review finding) | Fixed, regression-tested | `TestWatch_Stall_UsesLatestEventOfAnyType_NotOnlyStateEvents` passes |
| L-7: codex unknown-mode conflated with not-yet-verified (self-review finding) | Fixed, regression-tested | `TestPermissionArgsForDriver_Codex_UnknownMode_DistinctError` passes |
| Regression suite (PR①–③ org runtime, lockstep, check-sync) | Green | All 30 shell suites + full `go test ./...` pass; no PR①–③ test broke |

## Test gaps

- **AC-5 / M-4 (carried from verify report, confirmed still open at test level):** `leadActivityEventCount` only excludes the watchdog's *own* cutoff-reason events from the manifest-growth activity source; it is not scoped to `orgID` or to lead-attributable seats. No test exists for the broader case (an unrelated seat's own manifest event, or a different org sharing the same manifest file, prematurely clearing a pending alert). This is the same gap the verify report flagged — confirmed here as a genuine coverage gap, not just a static-reading concern, since no test drives `leadActivityEventCount` with a multi-org or multi-seat manifest to prove/disprove the false-positive path.
- **`realEscalate` (watch.go:336): 0% unit coverage.** The `osascript`-invoking real escalation path is entirely replaced by a stub in all `internal/org` tests (as expected for a side-effecting OS integration point), so its behavior is only proven by the live smoke evidence (AC-10), not by `go test`. This is consistent with the plan's design (best-effort, real-machine-verified) but worth naming explicitly as a permanent unit-test blind spot for this function.
- **`loadWatchStatus` (watch.go:227): 44.4% statement coverage.** Error-handling branches (e.g., corrupt/partial JSON on disk, permission errors) are under-exercised; only the happy-path load/round-trip is driven by current tests.
- **`detailsSuffix` (watcher.go:155): 0% coverage.** Small formatting helper in the watcher-error receipt path; not directly asserted by any test (only indirectly via the receipt's final JSON shape in the malformed-JSON tests, which may not force this specific branch).
- **AC-10 state-dir cross-cwd live confirmation:** per the verify report, no line in the committed smoke evidence transcript demonstrated two different cwds inside the repo converging on the same manifest in a live run; the plan/manifest-provided worktree note in the latest commit (`6a09e64 docs: add state-dir cross-cwd confirmation to watchdog smoke evidence`) appears to address this — confirmed the commit exists and updates the smoke evidence, but per AC-10's own scope this is a live-evidence artifact, not a `go test` target, so it is out of this report's verification surface.
- **`internal/cli` 67.3% is module-wide, not watchdog-scoped.** The `watch` subcommand's CLI wiring (`org.go`) is exercised indirectly through `internal/org` unit tests (`ResolveOrgStateDir`, `RunWatch`, etc.) rather than dedicated `internal/cli` command tests; no `internal/cli` test directly invokes `ralph org watch` end-to-end (consistent with the rest of the `org` verb family, which is also unit-tested at the `internal/org` layer rather than the CLI layer).

## Verdict

- **Pass**: All 30 shell test suites (`./scripts/run-test.sh`) and all Go packages (`go test ./...` full scope, plus the explicit `go test ./internal/org/... ./internal/cli/... ./internal/config/... -race -count=1` targeted run) are green. Zero failures, zero skips, no flakes across two independent runs. All self-review regression tests (H-1, H-2, M-3, M-5, M-6, L-7) pass. AC-1 through AC-9, AC-11 are behaviorally verified by the mapped tests above.
- **Fail**: None.
- **Blocked**: None.
- **Carried-forward gaps (not blocking, per verify report's own recommendation):** AC-5/M-4's narrower-than-recommended fix (no regression test for the broader cross-org/cross-seat false-positive case) and AC-10's state-dir live-evidence line remain open follow-ups — consistent with the verify report's "PASS with two follow-up items" verdict. Recommend tracking both as a tech-debt row or a fix-and-revalidate pass at the human operator's discretion; neither blocks proceeding to `/sync-docs`.

**Recommendation: proceed to `/sync-docs`.** Tests pass; safe to continue the pipeline toward `/cross-review` and `/pr`.

## Cycle 2

- Date: 2026-08-03
- Tester: `tester` subagent (behavioral tests only, no static analysis)
- Scope: delta `19c7630` + `ccf506e` on `feat/org-runtime-watchdog` (HEAD `b7110c6`), handed off after `docs/reports/verify-2026-08-02-org-runtime-watchdog.md`'s cycle-2 section (PASS). This delta closes cross-review AR-1/AR-2 and cycle-2 self-review M2-1/M2-2, all four of which cycle 1 above flagged as carried-forward gaps.
- Evidence: `docs/evidence/test-2026-08-03-org-runtime-watchdog-cycle2.log` (`./scripts/run-test.sh`), `docs/evidence/test-2026-08-03-org-runtime-watchdog-targeted.log` (targeted `-race` run) — both gitignored per `docs/evidence/*.log`, not committed.

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (33 shell suites, `tests/*.sh`) | all suites report `OK` | all | 0 | 0 | ~2 min |
| `./scripts/run-test.sh` → Go verifier, `go test ./...` (full-scope fallback, 13 packages) | 13 packages | 13 (all `ok`) | 0 | 0 | mixed cached/fresh |
| `go test ./internal/org/... ./internal/cli/... ./internal/config/... -race -count=1 -v` | 399 subtests, 5 packages (up from cycle 1's 393 — exactly the 6 new tests) | 399 | 0 | 0 | ~30s |

`./scripts/run-test.sh` again fell back to full language scope (same "unclassified" `scripts/ralph-config.sh` classification noted in cycle 1) — not an override. No failures, no skips, no flakes across either run.

### New-test confirmation (AR-1 / AR-2 / M2-1 / M2-2)

| Finding | Test | File | Package | Result |
| --- | --- | --- | --- | --- |
| AR-1 (cross-review): `leadActivityEventCount` not scoped to the watched org — an unrelated org sharing the same manifest could clear a different org's pending deadman alert | `TestWatch_Deadman_CrossOrgActivity_DoesNotClearPendingAlert` | `internal/org/watch_test.go` | `internal/org` | PASS |
| AR-2 (cross-review): total-budget cutoff false-positive-ALERTs an org with zero active seats, repeating every cycle forever | `TestWatch_TotalBudgetCutoff_NoActiveSeats_NoAlertNoEscalation_ThenCutsNewSeat` | `internal/org/watch_test.go` | `internal/org` | PASS |
| M2-1 (cycle-2 self-review): lead activity must be lead-*attributable*, not merely org-scoped — an unrelated seat's own event in the same org must not clear another seat's pending alert; lead's own `spawned` event and a manual (non-watchdog) stop of another seat must still clear it | `TestWatch_Deadman_UnrelatedSeatEvent_DoesNotClearPendingAlert`, `TestWatch_Deadman_LeadSpawnedEvent_ClearsPendingAlert`, `TestWatch_Deadman_ManualStopOfOtherSeat_ClearsPendingAlert` | `internal/org/watch_test.go` | `internal/org` | PASS (3/3) |
| M2-2 (cycle-2 self-review): `newWatchdogHooks` must thread the caller's `ctx` (`cmd.Context()`) into the tracked goroutine's `RunWatcher` call instead of a fixed `context.Background()`, so a cancelled command context (e.g. SIGINT) returns quickly instead of waiting out the full watcher timeout | `TestNewWatchdogHooks_CtxCancelled_RunWatcherReturnsQuickly` | `internal/cli/org_test.go` | `internal/cli` | PASS |

All 6 new/changed regression tests exist, run, and pass. Each is traced to its originating finding by comment header in the source (verified by reading the diff, not just the test name).

### Coverage delta

- `internal/org`: 88.1% (unchanged from cycle 1) — the 5 new `internal/org` tests exercise already-instrumented lines in `watch.go` (lead-activity scoping, total-budget zero-active-seat guard), not new lines, so the aggregate percentage does not move.
- `internal/cli`: 67.3% (unchanged from cycle 1) — the 1 new test exercises the existing `newWatchdogHooks`/`RunWatcher` goroutine path with a pre-cancelled context, again not new lines.
- No coverage regressions introduced by the delta.

### Regression checks (cycle 2)

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| AR-1: cross-org lead-activity leak (cross-review finding) | Fixed, regression-tested | `TestWatch_Deadman_CrossOrgActivity_DoesNotClearPendingAlert` passes |
| AR-2: zero-active-seat total-budget false ALERT loop (cross-review finding) | Fixed, regression-tested | `TestWatch_TotalBudgetCutoff_NoActiveSeats_NoAlertNoEscalation_ThenCutsNewSeat` passes |
| M2-1: lead-activity not attribution-scoped (cycle-2 self-review finding) | Fixed, regression-tested | `TestWatch_Deadman_UnrelatedSeatEvent_DoesNotClearPendingAlert`, `TestWatch_Deadman_LeadSpawnedEvent_ClearsPendingAlert`, `TestWatch_Deadman_ManualStopOfOtherSeat_ClearsPendingAlert` all pass |
| M2-2: watcher goroutine ignored caller's cancelled context (cycle-2 self-review finding) | Fixed, regression-tested | `TestNewWatchdogHooks_CtxCancelled_RunWatcherReturnsQuickly` passes |
| Cycle-1 carried-forward AC-5/M-4 gap | Superseded — the broader cross-org/cross-seat false-positive case cycle 1 flagged as untested is now exactly what AR-1's and M2-1's new tests cover | See rows above |
| Full regression suite (all cycle-1 tests + PR①–③ org runtime, lockstep, check-sync) | Green | 399/399 targeted subtests pass; 33/33 shell suites report `OK`; all 13 Go packages `ok` |

### Failure analysis

| Test | Error | Root cause | Proposed fix |
| --- | --- | --- | --- |
| — | — | No failures observed | — |

### Test gaps (cycle 2)

- The cycle-1 AC-5/M-4 gap ("no test drives `leadActivityEventCount` with a multi-org or multi-seat manifest to prove/disprove the false-positive path") is now closed by AR-1's and M2-1's tests — removed from the open-gaps list.
- Cycle-1's other named gaps (`realEscalate` 0% unit coverage, `loadWatchStatus` 44.4% branch coverage, `detailsSuffix` 0% coverage, `internal/cli` 67.3% module-wide vs watchdog-scoped) are unchanged by this delta and remain open at the same scope described in cycle 1 — none of them are touched by `19c7630`/`ccf506e`.
- AC-10 live-smoke state-dir cross-cwd confirmation remains out of `/test`'s scope (live-evidence artifact, not a `go test` target), unchanged from cycle 1.

### Verdict (cycle 2)

- **Pass**: All 33 shell test suites (`./scripts/run-test.sh`) and all 13 Go packages (`go test ./...` full scope, plus the explicit `go test ./internal/org/... ./internal/cli/... ./internal/config/... -race -count=1` targeted run, 399 subtests) are green. Zero failures, zero skips, no flakes. All 6 new regression tests for AR-1, AR-2, M2-1 (×3), and M2-2 exist and pass, confirmed against their source comments.
- **Fail**: None.
- **Blocked**: None.
- **Carried-forward gaps (not blocking):** `realEscalate`/`loadWatchStatus`/`detailsSuffix` unit-coverage blind spots and AC-10's live-evidence scope, all unchanged from cycle 1 and unrelated to this delta.

**Recommendation: proceed to `/sync-docs` → `/cross-review` → `/pr`.** Tests pass on cycle 2; both cross-review AR findings and both cycle-2 self-review M2 findings are now regression-tested.
