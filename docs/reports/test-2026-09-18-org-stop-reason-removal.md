# Test report: org-stop-reason-removal

- Date: 2026-09-18
- Plan: `docs/plans/active/2026-09-17-org-stop-reason-removal.md` (issue #153)
- Tester: `tester` subagent (Claude Code, standard flow)
- Branch: `refactor/org-stop-reason-removal`, HEAD `eff00e1` (base `main` @ `c6b8127`)
- Scope: changed-language scope (golang) via `./scripts/run-test.sh`, plus fresh `go test ./internal/... -count=1` and targeted `-run`/`-v` invocations. No static analysis run here (verifier's job; see `docs/reports/verify-2026-09-18-org-stop-reason-removal.md`, which reports AC-1..AC-4/AC-6/AC-7 Pass and defers AC-5's test clause to this report).
- Evidence: `docs/evidence/test-2026-09-18-org-stop-reason-removal.log` (gitignored, `./scripts/run-test.sh` full output)

## Overall verdict: Pass

All behavioral tests pass. The four targeted deadman tests named in the
handoff (2 renamed, 1 signature-changed, 1 new with 2 subtests) all pass
individually and within the full `TestWatch_Deadman` family (15 top-level
functions, 17 including the new test's 2 subtests). `internal/cli` tests
that compile against the now-Reason-less `StopParams` pass. Full regression
(`go test ./internal/... -count=1`, fresh, no cache) is 8/8 packages green.
`./scripts/run-test.sh` exits 0 across 28 shell test files plus the same 8
Go packages.

## Test execution

### `./scripts/run-test.sh` (changed-language scope = golang; shell suite runs unscoped as a regression net)

| Suite | Result |
| --- | --- |
| 28 shell test files under `tests/*.sh` (`test-agent-phase-boundaries.sh`, `test-branch-name.sh`, `test-check-mojibake.sh`, `test-check-skill-sync.sh`, `test-detect-changed-languages.sh`, `test-detect-languages-terraform.sh`, `test-ensure-pr-ready.sh`, `test-ensure-pr-title-prefix.sh`, `test-gc-artifacts.sh`, `test-hook-wiring.sh`, `test-insights-append.sh`, `test-language-pack-monorepo-roots.sh`, `test-no-loop-references.sh`, `test-post-edit-verify.sh`, `test-pre-bash-guard.sh`, `test-ralph-config.sh`, `test-ralph-dispatch.sh`, `test-ralph-worktree.sh`, `test-run-verify-scope.sh`, `test-secret-scan.sh`, `test-self-review-scope.sh`, `test-sync-skills.sh`, `test-template-purity.sh`, `test-terraform-gitignore.sh`, `test-terraform-pack-verify.sh`, `test-terraform-rule-frontmatter.sh`, `test-verify-mode-split.sh`, `test-xreview-helpers.sh`) | All PASS. Zero `FAIL: <n>` (n≥1) lines anywhere in the log; every suite that prints an explicit summary reports `FAIL: 0`. None of these suites touch `internal/org`/`internal/cli` — they are an unrelated regression net, unaffected by this PR's Go-only + docs diff. |
| `golang` language verifier (`go test ./...` for the 8 test-bearing packages, cached at this point in the wrapper) | `ok` for `internal/cli`, `internal/config`, `internal/insights`, `internal/org`, `internal/org/driver`, `internal/org/protocol`, `internal/scaffold`, `internal/upgrade`; `[no test files]` for the two `main`-package roots (`github.com/yoshpy-dev/ralph`, `cmd/ralph`) |
| Wrapper exit code | 0 (`==> All verifiers passed.`) |

### Fresh `go test ./internal/... -count=1` (no cache, superset of AC-5's named command)

```
ok  	github.com/yoshpy-dev/ralph/internal/cli	38.428s
ok  	github.com/yoshpy-dev/ralph/internal/config	1.090s
ok  	github.com/yoshpy-dev/ralph/internal/insights	1.094s
ok  	github.com/yoshpy-dev/ralph/internal/org	10.733s
ok  	github.com/yoshpy-dev/ralph/internal/org/driver	0.475s
ok  	github.com/yoshpy-dev/ralph/internal/org/protocol	2.783s
ok  	github.com/yoshpy-dev/ralph/internal/scaffold	2.126s
ok  	github.com/yoshpy-dev/ralph/internal/upgrade	2.924s
```

8/8 packages ok, 0 failures. This is a superset of AC-5's `go test
./internal/org/... ./internal/cli/... -count=1`; both named packages are
included and green.

### Targeted tests (the 4 named in the handoff), `-run ... -count=1 -v`

| Test | Result |
| --- | --- |
| `TestWatch_Deadman_LegacyWatchdogStopEvent_DoesNotClearPendingAlert` (renamed; legacy event via `Manifest.Append`) | PASS |
| `TestWatch_Deadman_SeatSentEvent_ClearsPendingAlert_LegacyWatchdogStopDoesNot` (renamed; legacy event via `Manifest.Append`; positive `sent` case kept) | PASS |
| `TestWatch_Deadman_ManualStopOfOtherSeat_ClearsPendingAlert` (Reason dropped from the `Stop` call) | PASS |
| `TestWatch_Deadman_PersistedAlertBaseline_SurvivesLegacyWatchdogStop` (new) | PASS (parent) |
| &nbsp;&nbsp;subtest `no_new_events:_alert_survives_and_escalates_once_at_deadline` | PASS |
| &nbsp;&nbsp;subtest `genuine_lead_sent_event_clears_the_alert_without_escalation` | PASS |

Raw run:

```
=== RUN   TestWatch_Deadman_LegacyWatchdogStopEvent_DoesNotClearPendingAlert
--- PASS: TestWatch_Deadman_LegacyWatchdogStopEvent_DoesNotClearPendingAlert (0.00s)
=== RUN   TestWatch_Deadman_PersistedAlertBaseline_SurvivesLegacyWatchdogStop
=== RUN   TestWatch_Deadman_PersistedAlertBaseline_SurvivesLegacyWatchdogStop/no_new_events:_alert_survives_and_escalates_once_at_deadline
=== RUN   TestWatch_Deadman_PersistedAlertBaseline_SurvivesLegacyWatchdogStop/genuine_lead_sent_event_clears_the_alert_without_escalation
--- PASS: TestWatch_Deadman_PersistedAlertBaseline_SurvivesLegacyWatchdogStop (0.01s)
    --- PASS: TestWatch_Deadman_PersistedAlertBaseline_SurvivesLegacyWatchdogStop/no_new_events:_alert_survives_and_escalates_once_at_deadline (0.00s)
    --- PASS: TestWatch_Deadman_PersistedAlertBaseline_SurvivesLegacyWatchdogStop/genuine_lead_sent_event_clears_the_alert_without_escalation (0.00s)
=== RUN   TestWatch_Deadman_SeatSentEvent_ClearsPendingAlert_LegacyWatchdogStopDoesNot
--- PASS: TestWatch_Deadman_SeatSentEvent_ClearsPendingAlert_LegacyWatchdogStopDoesNot (0.00s)
=== RUN   TestWatch_Deadman_ManualStopOfOtherSeat_ClearsPendingAlert
--- PASS: TestWatch_Deadman_ManualStopOfOtherSeat_ClearsPendingAlert (0.00s)
PASS
ok  	github.com/yoshpy-dev/ralph/internal/org	0.255s
```

### Full `TestWatch_Deadman` family (regression), `-run 'TestWatch_Deadman' -count=1 -v`

15 top-level test functions (`grep -c '^func TestWatch_Deadman' internal/org/watch_test.go` → 15; 17 counting the new test's 2 subtests separately), all PASS:

`NoActivity_EscalatesOnceAfterTimeout`, `LeadActivity_PreventsEscalation`,
`LeadIsAnomalySubject_EscalatesImmediately`,
`LegacyWatchdogStopEvent_DoesNotClearPendingAlert`,
`PersistedAlertBaseline_SurvivesLegacyWatchdogStop` (+2 subtests),
`CrossOrgActivity_DoesNotClearPendingAlert`,
`SeatSentEvent_ClearsPendingAlert_LegacyWatchdogStopDoesNot`,
`LeadSpawnedEvent_ClearsPendingAlert`,
`LeadSpawnsReplacementSeat_ClearsPendingAlert`,
`ManualStopOfOtherSeat_ClearsPendingAlert`,
`WatchdogAlertHistoryLine_DoesNotClearPendingAlert`,
`LeadHistoryLine_ClearsPendingAlert`,
`HistoryWindowEviction_DoesNotFalselyClearPendingAlert`,
`ProbeOutageRecoveryAlone_DoesNotClearPendingAlert`,
`ProbeOutageThenGenuineManifestActivity_ClearsPendingAlert`.

Result: `PASS`, `ok github.com/yoshpy-dev/ralph/internal/org 0.288s`. No name in
this set matches the retired `WatchdogsOwnStopEvent`/`_WatchdogStopDoesNot`
forms (consistent with AC-3, already confirmed by `/verify`).

### `internal/cli` — `ralph org stop` / `Disband` callers (compile + behavior)

```
=== RUN   TestOrgDisband_StopsActiveSeatsAndDisbandsOrg
--- PASS: TestOrgDisband_StopsActiveSeatsAndDisbandsOrg (0.29s)
=== RUN   TestOrgStop_UnknownSeat_NonZeroExit
--- PASS: TestOrgStop_UnknownSeat_NonZeroExit (0.00s)
=== RUN   TestOrgStop_ExistingSeat_LeavesAndRecordsOutcome
--- PASS: TestOrgStop_ExistingSeat_LeavesAndRecordsOutcome (0.31s)
PASS
ok  	github.com/yoshpy-dev/ralph/internal/cli	0.903s
```

All 3 pass, confirming both that `internal/cli` compiles against the
Reason-less `StopParams` (`org.go` never set `Reason`, so no call-site edit
was needed) and that `stop`/`disband` behavior is unchanged.

## Coverage

| Package | Coverage | Note |
| --- | --- | --- |
| `internal/org` | 89.1% | Unchanged from the last recorded baseline (2026-09-17, `org-implementer-seat-envelope` cycle 2). No regression from this PR's comment-only `watch.go` diff + `verbs.go` field removal + 4 test edits. |
| `internal/org/driver` | 92.0% | Unchanged. |
| `internal/org/protocol` | 97.9% | Unchanged. |
| `internal/cli` | 81.0% | +0.1pp vs. the 80.9% baseline; `internal/cli` source is untouched by this PR (only `StopParams` construction sites, which never set `Reason`), so this is measurement noise (timing/goroutine-order across runs), not a signal from this change. |

`go test -cover` was run per-package (`./internal/org/...` combined
profile, `./internal/cli/...` separately) rather than a single
whole-repo profile, since the plan's test plan scopes coverage interest to
`internal/org` (the changed package) and `internal/cli` (the compile-check
package) specifically.

## Discrimination assessment: would subtest (i) fail if the guard were removed?

**Yes — confirmed from the test and production code, without editing source.**

`leadActivityEventCount` (`internal/org/watch.go:816-834`) only skips a
lifecycle event when `strings.Contains(ev.Details, "reason=watchdog_")`. If
that `if` were removed (i.e., every `EventStopped`/etc. always incremented
`n`), the count changes exactly for events carrying that substring — which
is exactly the legacy cutoff this test appends.

Subtest (i) (`newFixture`, `internal/org/watch_test.go:965-1033`) computes
`baseline` from `leadActivityEventCount(rr.Events, "org-a")` **before**
appending the legacy cutoff event, then persists that `baseline` as the
pending alert's `manifest_len`. It appends the legacy cutoff, and
`evaluateCycle` → `checkDeadman` (`watch.go:971,983`) recounts the *same,
now-larger* event set and compares `recount > pending.ManifestLen`:

- **With the guard** (shipped code): the legacy cutoff is excluded from the
  recount, so `recount == baseline`, `activity` is `false`, and the alert
  survives to escalate once — matching the assertion `f.escalateCalls != 1`
  failing the test if it were anything but 1 (confirmed PASS above, 1
  escalation).
- **Without the guard** (hypothetical): the recount would include the
  legacy cutoff and read `baseline + 1`, `recount > baseline` would be
  `true`, `checkDeadman` would treat this as genuine lead activity, clear
  the pending alert, and record zero escalations. `f.escalateCalls != 1`
  would then fail (0 ≠ 1), and the follow-on assertions (`PendingAlerts`
  still containing `f.alertID`, `Escalated[f.alertID]` false, 0 escalation
  records) would also fail.

So subtest (i) is guard-discriminating exactly as the plan's Discrimination
note (`docs/plans/active/2026-09-17-org-stop-reason-removal.md:112`, and the
test's own doc comment) claims. Subtest (ii) (appends a genuine `sent`
event on top) is **not** discriminating: a `sent` event always increments
the count under `leadActivityEventCount`'s case (a) regardless of the
guard, so `recount > baseline` is `true` either way and the alert clears —
this is the sanity check the plan and the test's own comment describe it
as, not a second guard-behavior pin.

I did not edit `watch.go` to empirically reproduce the guard-removed
failure; the conclusion above is derived by tracing the exact comparison
`checkDeadman` performs against the exact `Details` string the fixture
writes, which is a closed, deterministic calculation (no probes, no time
skew — `f.clk.Advance` isn't even called in subtest (i), only the alert's
persisted `ts` is already 10 minutes stale relative to a 5-minute
`DeadmanMinutes`).

## Test plan coverage vs. plan section

| Plan item | Status |
| --- | --- |
| Unit: deadman family (`LegacyWatchdogStopEvent` / `CrossOrgActivity` / `SeatSentEvent…_LegacyWatchdogStopDoesNot` / `LeadSpawnedEvent` / `LeadSpawnsReplacementSeat` / `ManualStopOfOtherSeat` / new `PersistedAlertBaseline…`) | All PASS, individually and as part of the full family (see above). |
| Unit: `Stop` / `Disband` existing tests | Covered by `internal/org`'s full suite (8/8 packages green includes these); no test names this list beyond what's already enumerated in `TestWatch_Deadman` and the `internal/cli` stop/disband tests above. |
| Integration: `internal/cli` `ralph org stop` tests (compile + behavior) | PASS (3/3, see above). |
| Regression: `go test ./internal/... -count=1` | PASS, fresh, 8/8 packages. |
| Edge case (1): legacy-baseline discrimination | Confirmed guard-discriminating (see Discrimination assessment above). |
| Edge case (2): manual `stopped` (no `reason=`) still counts as lead activity | Confirmed by `TestWatch_Deadman_ManualStopOfOtherSeat_ClearsPendingAlert` PASS — the test's own assertion is that seat-1's alert clears after the reason-less manual stop. |
| Edge case (3): legacy fixture `Details` matches the real historical `Stop` output shape | Confirmed by reading `Stop`'s pre-removal `paneNote + " " + leaveNote` construction (`internal/org/verbs.go:283-284`) — `"pane=ok leave=ok"` is exactly what both edited fixtures and the new test's fixture write before the `reason=watchdog_...` suffix. |

## Coverage gaps

- **No direct unit test for `leadActivityEventCount` in isolation.** It is
  only exercised indirectly through the `TestWatch_Deadman_*` family via
  `evaluateCycle`/`checkDeadman`. This is a pre-existing pattern (the
  function predates this PR and was already only integration-tested this
  way) — not a gap introduced by this change, but worth flagging since a
  future edit to the lifecycle-event switch (`watch.go:826-831`) has no
  table-driven unit test to pin its cases individually.
- **No test exercises a mixed-version window directly** (case (b) in the
  rewritten doc comment: an old `ralph org watch` binary appending a legacy
  cutoff *after* a new binary already recorded a guard-aware baseline). The
  new `PersistedAlertBaseline…` test pins case (a) (a pre-#152 *persisted*
  baseline) but case (b) (a live *mixed-binary* window within one process)
  is asserted only in the doc comment, not in a test. This matches the
  plan's Non-goals (no baseline-migration logic is being added) and Risks
  section, which accepts this as a documented, not tested, invariant — not
  a defect in this PR, but a residual blind spot for the legacy-compat
  guard specifically.
- **Tech-debt doc change (`docs/tech-debt/README.md`) has no executable
  test** — expected, it's a docs-only edit (AC-6), already grep-verified by
  `/verify`.

## Failure analysis

None. No test failed in any run (targeted, family, package, or full
wrapper).

## Regression checks

| Previously passing behavior | Status | Evidence |
| --- | --- | --- |
| Full `TestWatch_Deadman` family (15 functions / 17 with subtests) | PASS, no regressions | `/tmp` raw run above, reproduced in this report |
| `internal/org` coverage | Stable at 89.1% (no drop) | `go test -coverprofile` |
| `internal/cli` `ralph org stop`/`disband` CLI behavior | PASS, unchanged | 3/3 targeted CLI tests |
| Known flaky tests (`TestRunDoctorOpts_ProbeModelsFalse_NoSubprocess`, `TestRunWatcher_TimeoutIndependentOfSmallInterval`, per prior test-agent memory) | Did not flake this run | Both included in the fresh `go test ./internal/...` pass (`internal/cli` 38.428s ok, `internal/org` 10.733s ok) |

## Verdict

- Pass: YES — 28 shell test files + 8 Go packages green, fresh full-package
  run green, all 4 targeted tests (6 including subtests) individually
  confirmed PASS, deadman family regression (15/15, 17/17 with subtests)
  clean, `internal/cli` stop/disband compile+behavior confirmed.
- Fail: NO
- Blocked: NO — safe to proceed to `/sync-docs`.
