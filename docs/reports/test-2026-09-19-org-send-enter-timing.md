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

## Cycle 2 (2026-09-19)

- Scope: re-test against HEAD `9daabc0` (do not treat the cycle-1 section
  above as current for anything past AC wording — it describes the
  superseded `SendResult.TextTyped`/`EnterPressed` two-bool contract at
  HEAD `354e580`). `git status --porcelain` confirmed empty before
  starting. Diff since the cycle-1 test commit: `git diff c3ec8a1..9daabc0`,
  18 files, +980/−182. Verify cycle 2 (`docs/reports/verify-2026-09-19-org-send-enter-timing.md`,
  section "Cycle 2") passed against `bf4c8c8`; `git diff bf4c8c8..9daabc0`
  touches only docs (verify report, insight event, one tech-debt fix), so
  the code under test at `9daabc0` is exactly what verify's cycle-2 already
  checked statically.
- What changed since the cycle-1 report (commit `c3ec8a1`): (1) `3e35c4d`
  (team lead) added `fakeHerdr.paneSendTextErr` and a send-text-failure
  test, closing the `PaneSendText`-failure gap the cycle-1 report named.
  (2) Cross-review AR-1/AR-2 (`35a3594`) replaced the two-bool
  `TextTyped`/`EnterPressed` contract with `SendResult.Progress`
  (`SendProgress`, five states — see `verbs.go:82-128`'s doc comment) and
  added `orgReadCommandHint`/`shellQuoteIfNeeded` (`internal/cli/org.go`),
  which append a `--state-dir` recovery hint to the CLI's post-error notes
  and the exit-0 unconfirmed-submit warning only when `--state-dir` was
  explicitly passed on the invocation. (3) Self-review cycle 2 (`646fb4b`)
  made six further fixes: a CLI timing test's margin, one added sentence
  ("do not press Enter" ... "If the pane shows anything else") to the
  exit-0 warning, two test renames, a doc-comment reorder, and a
  recipe/skill wording qualification.

### Test execution

| Suite / Command | Tests | Passed | Failed | Skipped | Duration |
| --- | --- | --- | --- | --- | --- |
| `./scripts/run-test.sh` | 710 shell assertions across 23 files + 8 Go packages | 710 shell, 8 Go pkgs | 0 | 0 | ~57s |
| `go test ./internal/org/... ./internal/cli/... -count=1` | 4 packages | 4 | 0 | 0 | ~53s |
| `go test -race ./internal/org/... -count=1` | 3 packages | 3 | 0 | 0 | ~13s |
| `TMPDIR=/tmp go test ./internal/org/... ./internal/cli/... -count=1` | 4 packages | 4 | 0 | 0 | ~52s |
| `go test ./internal/org/ -run 'TestOrgSend' -count=50` | 1 package, 50 iterations | 50/50 | 0 | 0 | 12.5–14.7s |
| `go test -race ./internal/org/ -run 'TestOrgSend_CtxExpiresDuringEnterDelay\|TestOrgSend_ConfirmTimeout_CappedByRemainingCtxBudget\|TestOrgSend_BudgetTooSmall' -count=30` | 3 tests x 30 iterations = 90 subtest runs | 90 | 0 | 0 | 6.1–6.7s |
| Under-load timing test: `go test ./internal/cli/ -run 'TestOrgSend_CtxExpiresDuringEnterDelay_NotesTextTypedOnStderr' -count=10` while `go test ./internal/cli/... -count=1` ran in parallel (background, exit 0, 46.3s) | 10 iterations (foreground) + 1 full-package run (background) | 10 foreground + full background suite | 0 | 0 | foreground 29.5s total (2.72–3.65s/iter); background 46.3s |

Every command was run twice: once before adding the `SendProgress.String()`
test below, once after, at HEAD `9daabc0` with only
`internal/org/verbs_test.go` modified in the working tree. Both runs are
0-failure; only the second (final) run's numbers are in the table above.

For the under-load check, the real-subprocess-backed timing test
(`TestOrgSend_CtxExpiresDuringEnterDelay_NotesTextTypedOnStderr`, which
sleeps ~1.1s per herdr-stub call and needs its ctx-vs-timer race to land
inside a ~700ms–1.4s window per the test's own doc comment) ran 10 times in
the foreground while the *entire* `internal/cli` package (all ~150+ tests,
several of which spawn their own herdr-stub subprocesses) ran once in the
background — the closest local stand-in for a busy CI runner available
without a container. All 10 foreground iterations passed
(2.72s–3.65s each, consistent with the ~2.8s the test's own doc comment
predicts), and the background full-package run also passed (exit 0,
46.286s). **Zero failures, including under contention.**

### Coverage

Two profiles were generated (both outside the repo, per the handoff):
`go test ./internal/org/ -coverprofile=...` and
`go test ./internal/cli/ -coverprofile=...`, both `-count=1`, both after
adding the new test below.

- `internal/org` statement total: 89.8% (was 89.2% before the new test;
  89.6% at the end of cycle 1 before this cycle's `SendProgress` refactor
  temporarily dropped it back to 89.2% by adding new code, then the new
  test brought it past the cycle-1 figure).
- `internal/org/verbs.go`, function-level (`go tool cover -func`):
  - `Send`: **95.9%** (was 93.9% at the end of cycle 1 — the send-text
    failure test team lead added in `3e35c4d` closed that gap; see "Read
    directly from statement ranges" below for what is still uncovered).
  - `SendProgress.String`: **100%** (was **0%** — closed this cycle, see
    below).
  - `waitBeforeEnter`, `confirmSubmitted`, `capMSToContext`,
    `sendSubmitConfirmTimeout`, `resolvedHerdrAgentName`: 100% (unchanged).
  - `findSeat`: 85.7% (unchanged), `sendEnterDelay`: 66.7% (unchanged),
    `truncateForDetails`: 66.7% (unchanged, both pre-existing and untouched
    by this cycle's diff).
- `internal/cli/org.go`, function-level: **`orgReadCommandHint`: 100%**,
  **`shellQuoteIfNeeded`: 100%** — both fully covered already
  (`TestOrgReadCommandHint_Table` plus the two `TestOrgSend_StateDirHint_*`
  end-to-end tests). `newOrgSendCmd` (the function `go tool cover -func`
  attributes the whole `RunE` closure to): 93.9% — see the two minor,
  pre-existing gaps noted below.

**Read directly from `go tool cover`'s per-statement output** (not just the
per-function percentage), for the functions the handoff named:

- **`internal/org/verbs.go:274-276`** (`o.findSeat` returning a
  manifest-read error inside `Send`) — same gap as the cycle-1 report
  named (there at lines 220-222; line numbers shifted because `SendProgress`
  and its 75-line doc comment were inserted above `Send` this cycle).
  Reasoning unchanged: needs a corrupt/unreadable manifest at the moment
  `Send` calls it, distinct from the write-failure trick the
  `AppendEventFailsAfterEnter` tests already use.
- **`internal/org/verbs.go:328-330`** (the `remainingMS < 0` clamp inside
  the fail-closed budget check's error message) — same gap as cycle 1's
  274-276 (now shifted to 328-330). Still needs a `fakeHerdr` delay hook to
  reach deterministically (self-review revalidation's own NEW-3-adjacent
  seam), still outside this cycle's file scope.
- **`internal/org/verbs.go:408`** (`sendEnterDelay`'s
  `return defaultSendEnterDelay` fallback line) — same gap as cycle 1's
  line 340 (now shifted to 408). Unchanged reasoning: every existing test
  overrides `Org.SendEnterDelay` to keep `waitBeforeEnter`'s real
  `time.After` fast; nothing exercises the case where it is left at its
  zero value and the literal 750ms production default is what gets used.
- **`internal/org/verbs.go:134-149`** (`SendProgress.String()`) — **was
  entirely uncovered (0%)** before this cycle's addition. Confirmed via
  `grep -rn '\.String()\|SendProgress('` over both test files: nothing
  anywhere called it, not even implicitly through a `%v`/`%q` format verb
  inside a `t.Errorf` that never actually fired (every existing test
  currently passes, so those `Errorf` call sites are never reached). This
  is the exact edge case the handoff named
  ("`SendProgress.String()` for an out-of-range value") — closed below
  with `TestSendProgress_String_Table`, which also covers all five named
  states, not only the out-of-range default branch, since none of the six
  had a direct unit pin either (only indirect coverage through `Send`'s
  integration tests, which never call `.String()`).
- **`internal/cli/org.go:419-421` and `:435-437`** (inside `newOrgSendCmd`'s
  `RunE`: `requireOrgID(*orgID)` returning an error, and
  `newOrgRuntimeAt(...)` returning an error) — noted incidentally while
  generating the full `internal/cli` coverage profile, not part of the
  handoff's named targets. `grep -n '"send"' internal/cli/org_test.go`
  confirms no test invokes `send` without `--org-id` or with a broken
  `--config`/state dir. Not closing: this is the same boilerplate
  `requireOrgID`/`newOrgRuntimeAt` pattern repeated near-identically at the
  top of every `newOrg*Cmd` RunE (`spawn`, `wait`, `read`, `stop`, `status`,
  `disband`, `report`, `watch`), pre-existing and untouched by the AR-1/AR-2
  diff — out of this fix's scope, not something introduced or regressed by
  it.

### Red/green spot checks (throwaway, all reverted)

| Spot check | Throwaway edit | Test(s) expected red | Result |
| --- | --- | --- | --- |
| a | `Send`'s `PaneSendKeys`-failure return: `Progress: SendProgressEnterUnacknowledged` → `SendProgressTextTyped` | `TestOrgSend_PaneSendKeysFails_ReportsEnterUnacknowledged` (org), `TestOrgSend_PaneSendKeysFails_NotesEnterUnacknowledgedOnStderr` (cli) | **RED** on both: org — `expected Progress SendProgressEnterUnacknowledged, got text-typed`; cli — 5/5 assertions failed (the CLI printed the "typed but not submitted" note, the wrong one, because `SendProgressTextTyped` routes to a different `switch` case) |
| b | `Send`'s `PaneSendText`-failure return: `Progress: SendProgressTextUnacknowledged` + `PaneID: seat.PaneID` → `Progress: SendProgressNothingSent`, no `PaneID` | `TestOrgSend_PaneSendTextFails_ReportsTextUnacknowledged` (org), `TestOrgSend_PaneSendTextFails_NotesTextUnacknowledgedOnStderr` (cli) | **RED** on both: org — 2/2 assertions failed (wrong `Progress`, empty `PaneID`); cli — 4/4 assertions failed (the CLI's `switch` has no case for `SendProgressNothingSent`, so it printed **no note at all**, confirming that state's "prints nothing" contract doubles as this test's discriminator) |
| c (unconditional) | `orgReadCommandHint` appends `--state-dir` unconditionally (dropped the `if stateDirFlagSet` guard) | `TestOrgSend_StateDirHint_OmitsFlagWhenResolvedFromEnv`, `TestOrgReadCommandHint_Table`'s "flag not set" subtest | **RED** on both: the omit test failed (unexpected `--state-dir` in output); the table's first subtest failed (`want "ralph org read --org-id org-a --seat seat-1"`, got the flag appended) |
| c (never) | `orgReadCommandHint` never appends `--state-dir` (dropped the append entirely) | `TestOrgSend_StateDirHint_IncludesFlagWhenExplicit`, `TestOrgReadCommandHint_Table`'s three "flag set" subtests | **RED** on both: the includes test failed twice (both the warning and the post-error note lacked `--state-dir`); all three "flag set" table subtests failed |
| d | Dropped the "do not press Enter and do not send the message again" sentence from the enter-unacknowledged CLI note, replacing it with a shorter "Do not send the message again." | `TestOrgSend_PaneSendKeysFails_NotesEnterUnacknowledgedOnStderr` | **RED**: `expected the note to explicitly say not to press Enter unconditionally` |
| e (cycle-1 regression re-check) | Re-applied cycle-1's spot check (a): injected a second `PaneSendKeys(ctx, seat.PaneID, "Enter")` call after `confirmSubmitted` fails | `TestOrgSend_UnconfirmedSubmit_NeverResendsEnter` | **RED**: `expected PaneSendKeys to be called exactly once, got 2 calls: [[Enter] [Enter]]` — the no-resend regression test still bites unchanged after the `SendProgress` refactor |

Every edit was reverted with `git checkout -- <file>` immediately after
confirming red, and `git status --porcelain` was empty after each revert.
The affected test(s) were re-confirmed green in the subsequent full test
runs (all commands above ran clean after every revert).

### Edge cases (handoff step 8)

- **`--state-dir` with a space and with a single quote, through
  `shellQuoteIfNeeded`**: already covered.
  `TestOrgReadCommandHint_Table` (`internal/cli/org_test.go:1652`) has both
  cases — `/tmp/my state/dir` → single-quoted, `/tmp/o'brien` → POSIX-style
  escaped (`'/tmp/o'\''brien'`) — confirmed present, nothing to add. Spot
  check c's four discriminating failures above prove both subtests are
  load-bearing, not just present.
- **`SendProgress.String()` for an out-of-range value**: was completely
  untested (0% function coverage, see Coverage above) — closed with
  `TestSendProgress_String_Table`, which covers all five named states plus
  `SendProgress(99)` and `SendProgress(-1)`, proven to discriminate by
  weakening the `default` branch to `return "unknown"` (both out-of-range
  subtests went red; the five named-state subtests correctly stayed green,
  since they never touch the `default` branch).
- **Dry-run prints neither warning nor note**: the warning half is already
  tested (`TestOrgSend_DryRun_NeverWarnsAboutUnconfirmedSubmit`). The note
  half (the post-error `switch` on `result.Progress`) is structurally
  guaranteed rather than independently tested: `Send`'s `DryRun` branch
  (`verbs.go:265-271`) returns `SendResult{Err: err}` with `Progress` left
  at its zero value (`SendProgressNothingSent`) on every path, including
  `appendEvent` failing, and the CLI's `switch result.Progress` has no case
  for `SendProgressNothingSent` — confirmed by reading both call sites
  directly, the same "no note" contract spot check b's revert-state also
  demonstrated for a non-dry-run send. Not adding a dedicated test: it
  would need the same read-only-manifest trick as the
  `AppendEventFailsAfterEnter` tests, applied to a `--dry-run` invocation,
  for a property that is already implied by `Progress`'s own zero-value
  semantics and the `switch`'s exhaustive case list — lower value than the
  four gaps named above.

### Verdict

- Pass: all six numbered commands (1–6 including both flakiness checks and
  the under-load timing check), all five red/green spot checks (a–e,
  seven discriminating sub-results across them), and all four edge cases
  from step 8 — three already covered, one (`SendProgress.String()`)
  closed this cycle.
- Fail: none.
- Blocked: none.

One test was added to `internal/org/verbs_test.go`
(`TestSendProgress_String_Table`, a 7-case table test) to close a real,
previously entirely-uncovered function found via `go tool cover -func`;
proven to discriminate before being kept. Four gaps remain named rather
than closed: three carried over unchanged from cycle 1 (`findSeat`'s
manifest-read error, the `remainingMS<0` clamp, `sendEnterDelay`'s literal
750ms default) plus two minor, pre-existing, out-of-scope `newOrgSendCmd`
boilerplate branches (`requireOrgID`/`newOrgRuntimeAt` failures) noted
incidentally. This is pipeline cycle 2 of 2 (the last cycle per
`RALPH_STANDARD_MAX_PIPELINE_CYCLES`); recommend proceeding to
`/sync-docs`.
