# Test report: org-send-enter-timing

- Date: 2026-09-19
- Plan: `docs/plans/active/2026-09-19-org-send-enter-timing.md` (issue #163)
- Tester: `tester` subagent (Claude Code), standard flow, pipeline cycle 1 of cap 2
- Scope: behavioral tests only for `git diff main...HEAD` on `fix/org-send-enter-timing`, base `f006200`, plan/self-review/verify HEAD `354e580`. No static analysis, diff quality, or spec-compliance review — those are `/self-review`'s and `/verify`'s jobs (self-review: MERGE, verdict already recorded; verify: Pass, both already on record).
- Evidence: `docs/evidence/test-2026-09-19-org-send-enter-timing.log`

## Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` (23 shell test files + golang verifier) | 710 shell assertions + 8 Go packages | 710 shell, 8 Go pkgs | 0 | 0 | ~35s |
| `go test ./internal/org/... ./internal/cli/... -count=1` | 4 packages | 4 | 0 | 0 | ~44s |
| `go test -race ./internal/org/... -count=1` | 3 packages | 3 | 0 | 0 | ~13s |
| `TMPDIR=/tmp go test ./internal/org/... ./internal/cli/... -count=1` | 4 packages | 4 | 0 | 0 | ~45s |
| `go test ./internal/org/ -run 'TestOrgSend' -count=50` | 1 package, 50 iterations | 50/50 iter | 0 | 0 | 12.2s |
| `go test -race ./internal/org/ -run 'TestOrgSend_CtxExpiresDuringEnterDelay\|TestOrgSend_ConfirmTimeout_CappedByRemainingCtxBudget\|TestOrgSend_BudgetTooSmall' -count=30` | 3 tests x 30 iterations = 90 subtest runs | 90 | 0 | 0 | 6.1s |

All commands from the handoff ran clean, both before and after the two tests
added below (full re-run after the additions is recorded in the evidence log
under "Final full re-validation").

## Coverage

- Statement: `internal/org` 89.6% (`go test -coverprofile`), up from 89.4%
  before the two new tests.
- Function (touched functions, `go tool cover -func`, final run):
  - `Send` 93.9% (was 91.8%)
  - `waitBeforeEnter` 100.0%
  - `confirmSubmitted` 100.0%
  - `capMSToContext` 100.0% (was 80.0%)
  - `sendEnterDelay` 66.7% (unchanged)
  - `sendSubmitConfirmTimeout` 100.0%
- Notes: see "Coverage gaps" below for the specific uncovered branches named
  by reading the code, not just the percentage, and which ones were closed
  vs. left as named gaps.

## Failure analysis

No failing tests in the current tree. The table below is the red/green
discrimination evidence from Step 7's spot checks — each row is a
deliberate, throwaway production edit that was made, confirmed to turn the
named test(s) red, then reverted via `git checkout -- internal/org/verbs.go`
(confirmed by `git status --porcelain` returning empty after every revert,
and a full green re-run of the affected test(s) afterward).

| Spot check | Throwaway edit | Test(s) expected red | Result |
| --- | --- | --- | --- |
| a (handoff) | Added a second `PaneSendKeys(ctx, seat.PaneID, "Enter")` call after `confirmSubmitted` fails | `TestOrgSend_UnconfirmedSubmit_NeverResendsEnter` | **RED**: `expected PaneSendKeys to be called exactly once, got 2 calls: [[Enter] [Enter]]` |
| b (handoff) | Removed the fail-closed `--timeout-ms` budget check (the `if deadline, ok := ctx.Deadline(); ok { ... }` block before `PaneSendText`) | `TestOrgSend_BudgetTooSmallForEnterDelay_NothingTyped` | **RED**: 4 of 4 assertions failed — wrong error text, `TextTyped` true instead of false, non-empty `PaneID`, and `pane_send_text` appearing in the call list |
| c (handoff) | Set `EnterPressed: false` on the appendEvent-failure return (instead of `true`) | `TestOrgSend_AppendEventFailsAfterEnter_ReportsSubmittedButUnrecorded` (org-level) | **RED**: `expected EnterPressed true: PaneSendKeys succeeded before appendEvent failed` |
| c (handoff, CLI half) | Same edit as above | `TestOrgSend_AppendEventFailsAfterEnter_NotesSubmittedNotRecorded` (cli-level) | **RED**: 3 of 3 assertions failed — the CLI printed the "typed but not submitted" note instead of the "Enter was already pressed" note, on a path where Enter had in fact succeeded |
| d (added while closing a coverage gap) | Weakened `capMSToContext`'s two floors from `< 1` to `< 0` | new `TestCapMSToContext_FloorsAtOneMillisecond` | **RED**: 2 of 4 subtests failed (`want=0, no deadline` and `want=0, expired deadline` — exactly the two subtests that only the `< 1` floor, not `< 0`, catches) |
| e (added while closing a coverage gap) | Discarded the idle/done `AgentWait` error instead of returning it | new `TestOrgSend_IdleDoneWaitFails_ErrorsBeforeAnyTypingOrEvent` | **RED**: `expected a non-nil Err when the idle/done AgentWait fails` |

After each spot check, `git checkout -- internal/org/verbs.go` restored the
file and `git status --porcelain` was empty. The final `git status
--porcelain` shows only the intentional addition to
`internal/org/verbs_test.go` (81 lines, two new test functions) — no other
file is touched.

## Regression checks

| Previously broken behavior | Status | Evidence |
| --- | --- | --- |
| Enter is pressed exactly once per `Send` call, even on an unconfirmed submit (the #155/#163 no-resend design decision) | Held | `TestOrgSend_UnconfirmedSubmit_NeverResendsEnter` passes; spot check (a) above confirms it is not vacuous |
| A `--timeout-ms` budget too small to fund the pre-Enter pause is refused before anything is typed (self-review M2) | Held | `TestOrgSend_BudgetTooSmallForEnterDelay_NothingTyped` passes; spot check (b) confirms |
| `EnterPressed` distinguishes "typed, not submitted" from "typed, submitted, only the history record lost" on the appendEvent-failure path (self-review NEW-1) | Held | Both `TestOrgSend_AppendEventFailsAfterEnter_ReportsSubmittedButUnrecorded` (org) and `TestOrgSend_AppendEventFailsAfterEnter_NotesSubmittedNotRecorded` (cli) pass; spot check (c) confirms both are load-bearing |
| The confirm-wait timeout is capped by the remaining ctx budget and never allowed to fall to herdr's "0 = unbounded" sentinel (self-review M1) | Held, and now more completely pinned | `TestOrgSend_ConfirmTimeout_CappedByRemainingCtxBudget` (the "cap to remaining" branch) plus the new `TestCapMSToContext_FloorsAtOneMillisecond` (both floor branches, previously unexercised) |
| The idle/done wait failing leaves the message entirely untyped, with no manifest event | Newly covered, not a prior regression | New `TestOrgSend_IdleDoneWaitFails_ErrorsBeforeAnyTypingOrEvent`; spot check (e) confirms |

## Test gaps

Read directly from `go tool cover`'s per-statement output (not just the
per-function percentage) for the functions the handoff named. Three
statement ranges in `Send` remain uncovered; none were added by this diff,
and none are exercisable from `internal/org/verbs_test.go` or
`internal/cli/org_test.go` alone without either editing `fakeHerdr`
(`internal/org/spawn_test.go`, out of the file scope the handoff set for
this cycle) or a real filesystem-permission trick with more setup cost than
the branch's risk warrants — named here rather than added, consistent with
self-review's own precedent of naming vs. fixing:

- **`internal/org/verbs.go:220-222`** — `o.findSeat(p.OrgID, p.To)` returning
  a manifest-read error inside `Send`. `findSeat` itself is 85.7% covered
  (its own not-found branch is tested via
  `TestOrgSend_UnknownSeat_ErrorsWithoutDriverCall`), but the read-error
  branch specifically requires a corrupt/unreadable manifest file at the
  moment `Send` calls it — the existing read-only-manifest trick used by
  `TestOrgSend_AppendEventFailsAfterEnter_ReportsSubmittedButUnrecorded`
  fails a *write*, not this *read*, and Spawn (needed to seed the seat)
  itself writes the manifest, so a chmod-read-only-before-Send setup would
  work but wants its own dedicated test, not a quick addition.
- **`internal/org/verbs.go:274-276`** — the `remainingMS < 0` clamp inside
  the fail-closed budget check's error message (defensive: clamps the
  printed "Xms of --timeout-ms left" to a non-negative number). Reaching it
  requires `remaining` to already be negative at the moment the check runs,
  i.e. the ctx deadline has already passed by the time the idle/done wait
  returns — narrower than the `remaining <= enterDelay` case
  `TestOrgSend_BudgetTooSmallForEnterDelay_NothingTyped` already covers, and
  not deterministically reachable without a `fakeHerdr` delay hook (the same
  seam the self-review revalidation report's NEW-3 LOW finding already named
  for a related branch, at `internal/org/spawn_test.go`).
- **`internal/org/verbs.go:284-286`** — `o.Herdr.PaneSendText` itself
  returning an error. `fakeHerdr.PaneSendText` (`internal/org/spawn_test.go`)
  has no error-injection field today (unlike `paneSendKeysErr` for the
  sibling `PaneSendKeys` failure, which now has a full test:
  `TestOrgSend_PaneSendKeysFails_ReportsTypedButNotSubmitted`). This is the
  second half of the gap the plan's own Deviation notes flagged
  ("`Send` の send-text / Enter のエラー経路に既存の単体テストがない") — the
  Enter half was closed during the self-review fix rounds, the send-text
  half was not. Closing it needs a `fakeHerdr.paneSendTextErr` field, which
  is a `spawn_test.go` change outside this cycle's file scope
  (`internal/org/verbs_test.go` / `internal/cli/org_test.go` only per the
  handoff); flagging for a follow-up rather than widening scope here.

One edge case named in the handoff has no direct test, by design rather than
oversight:

- **`--enter-delay-ms 0` uses the built-in 750ms default.** The *mechanism*
  (`SendParams.EnterDelayMS <= 0` falls back to `Org.SendEnterDelay`, which
  itself falls back to `defaultSendEnterDelay`) is unit-tested via
  `TestOrgSend_EnterDelay_WaitsAtLeastConfiguredDuration` and
  `TestOrgSend_EnterDelayMS_OverridesOrgSendEnterDelay`, and every CLI-level
  `send` test that omits `--enter-delay-ms` exercises the flag's literal
  `0` default end to end through `newOrgSendCmd`. But no test asserts the
  literal production value (750ms) is what actually gets used when nothing
  overrides it — doing that deterministically would mean either sleeping
  750ms in a test or adding a time-injection seam solely to assert a
  constant against itself, which is lower value than the coverage gaps
  above. `internal/org/send_defaults_sync_test.go` (added during self-review
  LOW-7) is the safeguard against the constant and its doc surfaces
  drifting apart; that is the more useful check for this exact literal
  value.

## Verdict

- Pass: `./scripts/run-test.sh`, `go test ./internal/org/... ./internal/cli/... -count=1`,
  `go test -race ./internal/org/... -count=1`,
  `TMPDIR=/tmp go test ./internal/org/... ./internal/cli/... -count=1`,
  the `-count=50` and `-race -count=30` flakiness checks (0 failures across
  all of them), and all six red/green spot checks discriminated correctly on
  the first attempt (each expected-red edit went red on exactly the named
  test(s), each was reverted, and the affected test(s) were re-confirmed
  green afterward).
- Fail: none.
- Blocked: none.

Two tests were added to `internal/org/verbs_test.go`
(`TestCapMSToContext_FloorsAtOneMillisecond`,
`TestOrgSend_IdleDoneWaitFails_ErrorsBeforeAnyTypingOrEvent`) to close two
real, previously-unexercised branches found via `go tool cover -func`; both
were proven to discriminate (spot checks d and e) before being kept. Three
further branches and one edge case are named as gaps above rather than
closed, because closing them needs either a `fakeHerdr` field addition
(outside this cycle's file scope) or a dedicated manifest-corruption setup
whose cost outweighs the branch's risk. Recommend proceeding to `/sync-docs`.
