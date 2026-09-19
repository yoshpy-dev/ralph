# Self-review report: org-send-enter-timing

- Date: 2026-09-19
- Plan: `docs/plans/active/2026-09-19-org-send-enter-timing.md` (issue #163)
- Reviewer: `reviewer` subagent (Claude Code, opus), standard flow, pipeline cycle 1 of cap 2
- Scope: diff quality only for `git diff main...HEAD` on `fix/org-send-enter-timing` (base `f006200`, HEAD `468f4b2`, working tree clean). No tests, static analysis, spec-compliance, or doc-drift checks were run — those belong to `/verify` and `/test`.

## Verdict

MERGE with fixes. No CRITICAL and no HIGH findings. The central design decision
the user settled — exactly one `Enter` per `Send`, `blocked` counted as
submitted, an unconfirmed submit recorded rather than retried — is honoured by
the code: `internal/org/verbs.go:187` is the only `PaneSendKeys` call in the
send path, and the only other `PaneSendKeys` call sites in the package are
`Stop`'s two `C-c` sends (`spawn.go:1324`, `verbs.go:413`). Three MEDIUM
findings are cheap, local fixes (one comment, one flag help string, one
fail-closed guard); four LOW findings are wording, a test-flake margin, and a
duplicated literal.

## Finding counts by severity

| Severity | Count |
|----------|-------|
| CRITICAL | 0 |
| HIGH     | 0 |
| MEDIUM   | 3 |
| LOW      | 4 |

## Evidence reviewed

- `git status --porcelain` empty; `git log --oneline main..HEAD` — 6 commits (`c40a83e` plan, `85af422` advisory revision, `c990363` Slice A code+tests, `4c2d134` evidence, `77371a8` docs, `468f4b2` plan bookkeeping).
- `git diff --stat main...HEAD` — 15 files, +787 / −51. Full diff read for `internal/org/verbs.go`, `internal/org/spawn.go`, `internal/cli/org.go`, `internal/org/verbs_test.go`, `internal/org/spawn_test.go`, `internal/cli/org_test.go`, the recipe (+ template copy), the `/org` skill (4 surfaces), both evidence files, and the plan.
- Targeted reads outside the diff to check callee/caller contracts: `internal/org/driver/herdr.go:132-201` (`AgentStart`/`AgentWait`/`checkHerdrEnvelopeError`, `PaneSendText`/`PaneSendKeys` argv shapes), `internal/org/verbs.go:294-336` (`Wait`, the other `AgentWait` call site), `internal/cli/org.go:420-495` (`wait`/`read` flag names), `internal/org/report.go:164-182` and `internal/org/watch.go:828` (the only non-test readers of `ManifestEvent.Details`).
- Live probe of the real CLI the confirmation depends on (read-only, no server or seat started): `herdr --version` → `herdr 0.7.5`; `herdr agent wait --help` → `--timeout <MS>  Fail after this many milliseconds` and `Without --timeout, waits indefinitely`, `[possible values: idle, working, blocked, done, unknown]`.
- Mirror byte-identity: `diff docs/recipes/codex-seat-permissions.md templates/base/docs/recipes/codex-seat-permissions.md`, `diff .claude/skills/org/SKILL.md .agents/skills/org/SKILL.md`, `diff .claude/skills/org/SKILL.md templates/base/.claude/skills/org/SKILL.md` — all identical, no output.
- Invariant sweeps: `grep -rn 'PaneSendKeys(' internal/ | grep -v _test` (one Enter site, two `C-c` sites); `grep -rn 'raw=true'` across code, skills, rules and templates (no consumer parses the Details prefix — `report.go` and `watcher.go` only render it, `watch.go:828` uses `strings.Contains` on an unrelated substring, so prepending a second marker breaks nothing); `grep -rn '750'` across code, skills, recipes and templates.
- Secret sweep over the diff: `git diff main...HEAD | grep -Ei '(secret|token|passwd|password|apikey|api_key|credential|private_key)'` → only the plan's own references to running the range secret scan. No credential-shaped assignment anywhere in the diff; the evidence file names `auth.json` as a file path only, with no value.
- Doc-comment orphaning check (the 4 lines above every `+func ` hunk): the five new helpers are appended after `Send` and each keeps `truncateForDetails`'s doc comment intact above its own `func` (`verbs.go:286-293`); the new tests land at EOF of `verbs_test.go` and immediately above `// TestOrgWait_HappyPath_Succeeds` in `org_test.go` with a blank line between.

## Findings

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | maintainability | `capMSToContext`'s justification comment states the inverse of herdr's documented contract. It says the minimum-1 clamp exists because "herdr's AgentWait timeoutMS parameter carries no useful meaning at 0 -- none of its other call sites in this package pass 0 deliberately". Both halves are false. `driver.Herdr.AgentWait` omits `--timeout` when `timeoutMS <= 0`, and `herdr agent wait --help` (v0.7.5) says "Without --timeout, waits indefinitely" — so 0 has a very specific meaning, and it is the one thing a confirmation wait must never do. The other call site, `Wait` (`verbs.go:334`), passes the operator's value straight through, and `ralph org wait --timeout-ms 0` is documented as "pass 0 to explicitly wait unbounded". The clamp is correct; the reason recorded next to it would lead a future reader to remove it. | `internal/org/verbs.go:264-268` vs `internal/org/driver/herdr.go:168-170`, `internal/org/verbs.go:330-334`, `internal/cli/org.go:456` (`--timeout-ms` help for `wait`), and `herdr agent wait --help` | Rewrite the parenthetical to the real reason: 0 makes the driver omit `--timeout`, which herdr treats as an unbounded wait, so the confirmation must never pass 0 even when the remaining ctx budget rounds down to zero. |
| MEDIUM | exception-handling | A `--timeout-ms` budget that cannot fund the pre-Enter wait strands the message in the seat's composer, and nothing prevents or records it. `Send` types the text (`verbs.go:175`), then waits `enterDelay` bounded by the same ctx (`verbs.go:183`); if ctx wins, `Send` returns an error and appends no `sent` event, leaving typed-but-unsubmitted text in the pane. This is deterministic for `--enter-delay-ms`/default ≥ the remaining budget (`ralph org send --timeout-ms 500` can never submit anything, yet always types), and reachable at the default when the idle/done wait returns in the last 750 ms of the 30 s budget. A retry then types a second copy on top of the first, so the seat can receive two concatenated protocol messages as one. The residue class is pre-existing (a `PaneSendKeys` failure does the same), but this diff adds a 750 ms window and a deterministic trigger. | `internal/org/verbs.go:159-189`; `SendResult.PaneID` is documented as "Empty … for any error return" (`verbs.go:103-106`), so the CLI cannot even name the pane to clean up | Fail closed before typing: after the idle/done wait, compare `time.Until(deadline)` against `enterDelay` and return an error naming both values *before* `PaneSendText`, so nothing is typed when the budget cannot fund the submit. Additionally carry `seat.PaneID` on the post-`PaneSendText` error returns so the operator is told which pane holds the residue. |
| MEDIUM | maintainability | `ralph org send --timeout-ms`'s help no longer describes what the flag bounds. It still reads "idle-wait timeout in milliseconds before sending", but after this change the same ctx also funds the pre-Enter wait and the submit confirmation — the code says so (`waitBeforeEnter`'s comment: "the mechanism that lets --timeout-ms bound this wait too") and so does the plan ("`--timeout-ms` の期限は全体に掛かったまま"). An operator sizing the flag for the idle wait alone walks straight into the previous finding. | `internal/cli/org.go:406` vs `internal/org/verbs.go:163` (ctx creation), `226-229` (`waitBeforeEnter`), `259` (confirm), and `docs/plans/active/2026-09-19-org-send-enter-timing.md` Risks section | Reword to cover the whole verb, e.g. "overall herdr timeout in milliseconds for one send (idle wait + pre-Enter wait + submit confirmation)". The sibling `spawn`/`start` flags already use the broader "per-step herdr timeout in milliseconds" wording (`internal/cli/org.go:242,335`). |
| LOW | exception-handling | Two adjacent failures leave the pane in exactly the same state, but only one says so. The ctx-expiry error ends with "(text typed but not submitted)"; the `PaneSendKeys` failure one line later returns a bare "send Enter to seat %q: %w" even though the text is equally typed and equally unsubmitted. `Send`'s doc comment enumerates only the ctx case as leaving typed text behind, which reads as if the Enter-failure path does not. | `internal/org/verbs.go:183-189`; `Send` doc comment `verbs.go:123-127` | Append the same "(text typed but not submitted)" clause to the `PaneSendKeys` error, and widen the doc comment's sentence to cover both post-`PaneSendText` error returns. |
| LOW | maintainability | The confirmation is described as stronger than it is. `defaultSendSubmitConfirmTimeout`'s comment says the wait confirms "the Enter actually submitted the message rather than merely landing in the input box", but `confirmSubmitted` only observes that the seat left idle/done within the window — it cannot attribute the transition to our keystroke. A seat that moves to `working` for another reason inside the window (a queued follow-up firing, a concurrent `ralph org send`, a background hook) yields `SubmitConfirmed=true` and a `sent` event with no marker, even if the text is still in the composer. `SendResult.SubmitConfirmed`'s doc explains the false-negative direction thoroughly (short turns, lagging state) and is silent on the false-positive one. | `internal/org/verbs.go:31-36` (const comment), `239-244` (`confirmSubmitted`), `91-102` (`SubmitConfirmed` doc) | Soften to what is observed ("the seat left idle/done within the window, which is the best available proxy for a submit") and add one sentence to `SubmitConfirmed`'s doc noting that an unrelated transition inside the window reads as confirmed. No code change needed. |
| LOW | maintainability | `TestOrgSend_ConfirmTimeout_CappedByRemainingCtxBudget` gives itself ~20 ms of slack and hard-fails (not skips) if it runs out: `TimeoutMS: 40` against `SendEnterDelay = 20ms` means the 20 ms timer must fire before the 40 ms ctx deadline, otherwise `waitBeforeEnter` returns `ctx.Err()` and the test hits `t.Fatalf("expected Send to succeed")`. Timer delivery under a loaded CI runner (and under `-race`) can drift by that much. The neighbouring ctx-expiry test is the well-built counterexample: `TimeoutMS: 5` against a 200 ms delay gives it a 40× margin in the direction it needs. | `internal/org/verbs_test.go:630-641` vs `internal/org/verbs_test.go:600-608` | Raise `TimeoutMS` to ~400 while leaving `SendSubmitConfirmTimeout = 10 * time.Second`. The cap assertion (`confirmMS >= 10000` must not hold) still discriminates at 400 ms, and the margin grows from 20 ms to ~380 ms. |
| LOW | maintainability | The `750` default is now written out as a literal in eight normative places with no gate keeping them together: the constant, the CLI flag help, both recipe copies, and all four `/org` skill surfaces. Changing the constant — which the plan explicitly contemplates ("実機で足りなければ既定値を上げ") — silently falsifies seven documents. The repo already has the pattern for pinning a doc literal to a code default (`internal/config/defaults_sync_test.go`). | `internal/org/verbs.go:30`; `internal/cli/org.go:409-410`; `docs/recipes/codex-seat-permissions.md:140` (+ `templates/base/` copy); `.claude/skills/org/SKILL.md:105` (+ 3 mirrors) | Cheapest: add a "if you change this, update:" list to the constant's comment naming the flag help and the two doc families. Stronger: export the default in ms and build the flag help from it, leaving only the doc surfaces hand-written. |

## What was checked and found sound

- **Exactly one Enter, structurally.** `verbs.go:187` is the sole `PaneSendKeys` in `Send`; the package's only other call sites are `Stop`'s `C-c` (`spawn.go:1324`, `verbs.go:413`). `TestOrgSend_UnconfirmedSubmit_NeverResendsEnter` scripts the confirm wait to fail — precisely the branch a resend would be added to — and asserts `len(h.sendKeysKeys) == 1` with keys exactly `["Enter"]`, so a second call fails the test on both count and content.
- **Why there is no resend is written down where the next reader will be.** `confirmSubmitted`'s doc comment states the hazard (a submitted seat already in an approval dialog), the asymmetry that decides it (an unsent message is recoverable, an approval is not), and points back from `Send`'s doc comment and from the CLI's inline comment. The live evidence backs the hazard rather than asserting it (`docs/evidence/org-send-enter-timing-2026-09-19.md:25-34`: `blocked`, "Yes, proceed" preselected, "Press enter to confirm").
- **The confirmation's premise holds.** It depends on `herdr agent wait` failing rather than exiting 0 on timeout; `herdr 0.7.5 agent wait --help` documents `--timeout <MS>  Fail after this many milliseconds`. `checkHerdrEnvelopeError` preserves that failure as an error, so `submit_unconfirmed` is reachable in production and not inert.
- **Folding non-timeout herdr errors into "unconfirmed" is defensible here.** A server-down or agent-gone error at confirm time would have to occur *after* `PaneSendKeys` already succeeded against the same herdr, so the failure is late and rare; treating it as "cannot confirm" rather than "send failed" matches the decision that an unconfirmed submit is not an error. It is documented at `verbs.go:253-257` ("a confirmation failure here (error, or the confirm timeout elapsing)"), which is where a reader of `confirmSubmitted` would look.
- **`submit_unconfirmed` is not overstated.** `SendResult.SubmitConfirmed`'s doc, the CLI's inline comment, and the skill text all stop at "could not positively confirm"; the recipe conditions the manual Enter on "only if the message is still in the composer". The short-turn caveat the plan called for is present at `verbs.go:94-96` and `internal/cli/org.go:384-386`.
- **Operator-facing strings are accurate against the real CLIs.** The warning's four format verbs map to `to, *orgID, to, result.PaneID` in order; `ralph org read --org-id … --seat …` matches `newOrgReadCmd`'s real flags (`internal/cli/org.go:493-494`); `herdr pane send-keys <pane> Enter` matches `driver.Herdr.PaneSendKeys`'s positional argv (`herdr.go:233-234`). Exit code stays 0 (the warning goes to `ErrOrStderr` and `RunE` returns nil), as the skill and recipe both state.
- **Details prefix ordering and its consumers.** `raw=true ` then `submit_unconfirmed=true ` then the truncated text, pinned by `TestOrgSend_RawAndUnconfirmed_DetailsPrefixOrder`; no code parses the Details prefix (only `report.go`/`watcher.go` render it), so the added marker breaks no reader. The pre-existing quirk that the prefixes sit outside `sendDetailsMaxLen` is unchanged in kind.
- **Test doubles keep their old meaning.** `fakeHerdr.agentWaitErrs` is only consulted when non-empty, so every pre-existing test behaves exactly as before; the `1 → 2` AgentWait count updates in four existing tests are the honest consequence of the second wait, and each still asserts the target name for both calls. `testOrg` pinning `SendEnterDelay = time.Millisecond` is documented at `spawn_test.go:206-213` and overridden by the three tests that care about timing.
- **Timer handling.** `waitBeforeEnter` uses `time.After` without stopping the timer; the un-fired timer lives at most `enterDelay` (750 ms default) and `Send` is not a hot loop, so no stopped-timer rewrite is warranted. No test asserts an upper bound on elapsed time except the ctx-budget one covered above.
- **Evidence honesty.** `docs/evidence/org-send-enter-timing-2026-09-19.md` states its sample size in the table (`3 / 5`, `5 / 5`), says in prose that the defect is timing-dependent rather than deterministic, and records the one off-table pre-fix run that did submit. It does not claim the 750 ms value is optimal, only that it was sufficient in the runs performed. The environment section records the side effects that were left behind rather than omitting them.
- **Mirrors and hygiene.** Recipe root/template and all four skill surfaces are byte-identical. No debug statements, commented-out code, `TODO`/`FIXME`, or credential-shaped assignments in the added lines.

## Pre-existing, not introduced by this diff

- `appendEvent` failing after Enter loses the `sent` record for a message that *was* submitted (`verbs.go:198-203`). The same shape exists on `main`; this diff only adds `SubmitConfirmed` to what is lost with it. Not a finding against this PR.
- `Send` has no unit test for the `PaneSendText` / `PaneSendKeys` error paths — noted by the implementer in the plan's Deviation notes and pre-dating this change. Coverage belongs to `/test`, recorded here only so the two reports do not contradict each other.

## Tech debt identified

| Debt item | Impact | Why deferred | Trigger to pay down | Related plan/report |
| --- | --- | --- | --- | --- |
| (none appended to the register) | — | — | — | — |

No row was appended to `docs/tech-debt/README.md`. All seven findings are
in-cycle fixes rather than deferrals, and this is cycle 1 of cap 2. If the LOW
about the duplicated `750` literal is accepted as deferred rather than fixed, it
is the one item that warrants a register row (trigger: the first time
`defaultSendEnterDelay` changes, or the next `/org` skill edit).

No open row in `docs/tech-debt/README.md` is invalidated by this diff.

## Recommendation

- Merge: yes, after the three MEDIUM findings. None of them blocks correctness
  of the shipped behaviour on default settings; two are one-line text fixes and
  the third is a fail-closed guard of a few lines.
- Follow-ups: the four LOW findings, in the same cycle if the fix round is
  taken.

_(Superseded by the Revalidation section below — all seven were fixed in this
cycle.)_

## Revalidation (cycle 1 fix round, 2026-09-19)

- Fix range: `git diff a88ce2f..HEAD` — `7813c2a` (Go fixes, implementer) and `1461aac` (plan notes). HEAD `1461aac`, working tree clean.
- Scope unchanged: diff quality only. No tests, static analysis, spec-compliance or doc-drift checks were run.
- Evidence: full read of the fix diff for `internal/org/verbs.go`, `internal/cli/org.go`, `internal/org/verbs_test.go`, `internal/cli/org_test.go`, the new `internal/org/send_defaults_sync_test.go`, and the plan; re-verification of each cycle-1 citation against post-fix line numbers; `grep -rn 'TextTyped'` and `grep -rn 'rt.Send('` for every producer and consumer of the new field; marker-occurrence counts in all six doc surfaces; `grep -rn CtxExpiresDuringEnterDelay` (zero hits — the renamed test is not referenced anywhere); credential and debug-code sweeps over the added lines (clean).

### Verdict

MERGE. All seven cycle-1 findings are resolved. The fix round introduces one
MEDIUM and three LOW findings, all in the new code and all with local fixes; none
of them affects the default-settings happy path.

| Severity | Cycle 1 | New in fix round |
|----------|---------|------------------|
| CRITICAL | 0 | 0 |
| HIGH     | 0 | 0 |
| MEDIUM   | 3 (all resolved) | 1 |
| LOW      | 4 (all resolved) | 3 |

### Per-finding status

| # | Cycle-1 finding | Status | Post-fix evidence |
| --- | --- | --- | --- |
| M1 | `capMSToContext` comment inverts herdr's contract | **Resolved** | `internal/org/verbs.go:346-358`. The new text states the driver behaviour (`AgentWait` omits `--timeout` for values <= 0), quotes herdr's own help ("without --timeout, waits indefinitely"), and names `Wait` as a caller for which 0 *is* a valid choice. Every clause re-checked against `internal/org/driver/herdr.go:168-170`, `internal/org/verbs.go:330-334` and `internal/cli/org.go:456`. Code unchanged, as intended. |
| M2 | Short `--timeout-ms` strands typed text with no record | **Resolved** (see NEW-1 for a consumer-side defect the fix introduced) | `internal/org/verbs.go:234-243`: the check runs after the idle/done wait and before `PaneSendText`, so the deterministic case never types. `enterDelay` was hoisted above the wait to make that possible. All four returns after a successful `PaneSendText` now carry `PaneID` and `TextTyped` (`verbs.go:252,260,275,277`). `TestOrgSend_BudgetTooSmallForEnterDelay_NothingTyped` asserts the full call list is exactly `["agent_wait"]`, `TextTyped` false, `PaneID` empty, and no `sent` event — it fails on a revert in four independent ways. The repurposed fixture (`TimeoutMS: 5` vs a 200 ms delay) is deterministic: `5 <= 200` holds regardless of jitter. |
| M3 | `--timeout-ms` help describes only the idle wait | **Resolved** | `internal/cli/org.go:422-423` now reads "overall herdr timeout in milliseconds for one send (idle wait + pre-Enter wait + submit confirmation)", matching the ctx's real span and the sibling `spawn`/`start` wording. |
| L4 | Enter-failure error omits "typed but not submitted" | **Resolved** | `internal/org/verbs.go:257-262` appends the same clause; `Send`'s doc comment (`verbs.go:157-165`) now names both paths under one "Typed-but-unsubmitted text" paragraph instead of only the ctx one. New `TestOrgSend_PaneSendKeysFails_ReportsTypedButNotSubmitted` pins the message, `TextTyped`, `PaneID` and the absence of a `sent` event. |
| L5 | Confirmation described as stronger than it is | **Resolved** | Three surfaces updated consistently: `verbs.go:44-47` ("best available proxy … though not proof of it"), `verbs.go:319-327` (a dedicated "What this does NOT prove" paragraph naming the three racing causes), and `SubmitConfirmed`'s own doc (`verbs.go:107-118`, "true does not prove OUR Enter caused the transition"). No code change, which was the recommendation. |
| L6 | 20 ms margin in the cap test | **Resolved** | `internal/org/verbs_test.go:712` uses `TimeoutMS: 400` against a 20 ms delay (~380 ms margin), and the added comment explains the direction the margin protects. The assertion still discriminates: `confirmMS` lands near 380 and the test fails if it reaches the configured 10 s. |
| L7 | `750` duplicated across 8 normative surfaces with no gate | **Resolved** | `internal/org/verbs.go:16-26` exports `DefaultSendEnterDelayMS`; `defaultSendEnterDelay` derives from it; `internal/cli/org.go:427` builds the flag help with `fmt.Sprintf`, so the CLI literal is gone; `internal/org/send_defaults_sync_test.go` pins the remaining six hand-written doc surfaces. Marker spellings verified against the files: each of the four skill mirrors contains exactly one `750ms` and one `--enter-delay-ms`, each recipe copy exactly one `750 ms` and one `--enter-delay-ms`, so the check cannot pass on an unrelated occurrence today. |

### New findings introduced by the fix round

| Severity | Area | Finding | Evidence | Recommendation |
| --- | --- | --- | --- | --- |
| MEDIUM | exception-handling | The CLI's new typed-but-unsubmitted note also fires on the one error return where the message **was** submitted, and tells the operator to press Enter on it. `Send` sets `TextTyped: true` on four returns; three of them follow a *successful* `PaneSendKeys`, and one of those three is an error — the `appendEvent` failure at `verbs.go:275`, reached when the manifest write fails (`appendEvent` → `Manifest.Append`, a flock + JSONL append that can fail on a lock timeout, a full disk, or permissions) after Enter was delivered and `confirmSubmitted` has already run. The CLI gates only on `result.TextTyped` (`internal/cli/org.go:390`), so that path prints "the message text was typed into pane %s of seat %q **but not submitted** … **clear or submit it** there before sending again". An operator obeying it on a seat that submitted and has since reached an approval dialog presses exactly the blind Enter this PR exists to prevent — `confirmSubmitted`'s own doc calls that "an operation Send has no authorization to take" (`verbs.go:328-337`), and the live evidence shows the dialog preselects "Yes, proceed". The CLI's justifying comment states the opposite of the code: "TextTyped is false for every other error, including the fail-closed --timeout-ms budget check" (`internal/cli/org.go:386-388`) — the `appendEvent` failure is another error with `TextTyped` true. The plan's revised Scope row 1 lists all three post-typing failures (including `sent` イベントの記録エラー) under the single instruction "その pane を確認してから送り直す", so the two surfaces written in this fix round disagree about how many paths there are. `TextTyped`'s own doc (`verbs.go:119-130`) enumerates only the two unsubmitted returns plus success, which is how the omission slipped through. | `internal/org/verbs.go:275` vs `internal/cli/org.go:386-397`; `internal/org/verbs.go:328-337`; `docs/plans/active/2026-09-19-org-send-enter-timing.md` Scope row 1 | Distinguish "typed, not submitted" from "typed and submitted". Smallest honest change: add an `EnterPressed bool` (set from the successful `PaneSendKeys` onward), gate the existing note on `TextTyped && !EnterPressed`, and give the `appendEvent` path its own one-line note saying the message *was* submitted but the `sent` event could not be recorded, so it must **not** be resent. Alternative with a smaller diff: rename the field to what the CLI actually needs (e.g. `UnsubmittedText`) and leave it false on the `appendEvent` return — then fix `TextTyped`'s doc, the CLI comment, and the plan row to match. Either way the CLI comment's "every other error" sentence needs correcting. |
| LOW | null-safety | The budget check discards `ctx.Deadline()`'s `ok` and would refuse every send, with a nonsense number, if a deadline-less ctx ever reached it. `deadline, _ := ctx.Deadline()` (`verbs.go:234`) is safe today — `Send` always builds ctx with `context.WithTimeout`, and `timeoutMS <= 0` is normalised to the 30 s default at `verbs.go:199-202` — and the comment says so. But the sibling verb makes the trap likely: `ralph org wait --timeout-ms 0` is documented as an unbounded wait and `Wait` skips `WithTimeout` for it (`verbs.go:330-332`), and this same fix round added a comment at `verbs.go:355-358` explaining that asymmetry. A contributor extending send with the same `0 = unbounded` semantics gets `time.Until(time.Time{})`, a hugely negative `remaining`, and every send refused with "-62135596800000ms of --timeout-ms left". The printed value is also unclamped for the (much rarer) case where the idle/done wait returns success just past the deadline. | `internal/org/verbs.go:234-243`; `internal/org/verbs.go:330-334`; `internal/cli/org.go:456` | `if deadline, ok := ctx.Deadline(); ok { … }` so a deadline-less ctx skips the check rather than failing every call, and clamp `remainingMS` at 0 in the message. Two lines, and it removes the trap the next likely edit would spring. |
| LOW | maintainability | The block comment covering the now-untested ctx-inside-the-pause branch overstates the difficulty and points at a non-repo artifact. `verbs_test.go:639-653` says the branch is "Not independently testable without changing fakeHerdr (out of scope for this slice -- see the Slice D handoff's file list)" and then argues that shrinking the margin would only produce a flaky test. The second half is correct; the first is scoped to this slice rather than to the repo, and `internal/org/spawn_test.go` (where `fakeHerdr` lives) was already edited by this PR's own cycle-1 commit. A deterministic test does exist and needs no margin shaving: add a `paneSendTextDelay time.Duration` to `fakeHerdr` that `PaneSendText` sleeps, then run `TimeoutMS: 100`, `SendEnterDelay: 50ms`, `paneSendTextDelay: 80ms` — the budget check passes (100 > 50), `PaneSendText` consumes 80 ms, and `waitBeforeEnter` faces a 20 ms deadline against a 50 ms timer, i.e. a 30 ms margin in the direction that must win, the same robustness as the passing tests around it. That also exercises the exact production scenario the comment names ("PaneSendText's real round-trip … eats into the margin"). Separately, "the Slice D handoff's file list" is a message, not a repo artifact, so a future reader cannot resolve the pointer. | `internal/org/verbs_test.go:639-653`; `internal/org/spawn_test.go` (`fakeHerdr`, already modified by `c990363`) | Either add the fake's delay hook and the test, or keep the branch untested and reword the comment to say what is true: a deterministic test needs an injectable `PaneSendText` delay in `fakeHerdr`, which this slice chose not to add. Replace the handoff pointer with the plan's Deviation-notes line, which is in the repo. |
| LOW | maintainability | The sync test's failure message and doc comment both promise an adjacency check the assertion does not perform. `send_defaults_sync_test.go:81-85` is a whole-file `strings.Contains(text, s.marker)`, while the message says the value is expected "next to its --enter-delay-ms mention" and the file header repeats "next to its own --enter-delay-ms mention". Harmless today (verified: exactly one `750ms` / `750 ms` and one `--enter-delay-ms` per surface), but the gate would pass on an unrelated occurrence of the new number elsewhere in the file after a constant change — which is precisely the drift it exists to catch. | `internal/org/send_defaults_sync_test.go:12-13,77-85` | Either drop "next to its --enter-delay-ms mention" from both strings, or assert it: each surface has exactly one `--enter-delay-ms` line, so scanning for the line containing the flag and checking the marker within it is a few lines and makes the message true. |

### Answer to the explicit question (the untested ctx-inside-the-pause branch)

Keeping the branch is right — it is genuinely reachable in production once a real
`PaneSendText` round trip eats into the margin the budget check measured, and
deleting it would re-open M2 in a narrower form. Leaving it untested is *not*
necessary, though: the `paneSendTextDelay` hook described in the LOW above makes
it deterministic with a 30 ms margin and about six lines in `fakeHerdr`. My
recommendation is to add it (it is the same class of seam the fake already has
for `agentWaitErrs` and `paneSendKeysErr`); if the slice boundary matters more,
the acceptable minimum is rewording the comment, since as written it tells a
future reader the test cannot be built.

### Re-checked and still sound

- Exactly one `PaneSendKeys` in the send path (`verbs.go:256`); the no-resend regression test is unchanged and still asserts one call with exactly `["Enter"]`.
- The renamed `TestOrgSend_BudgetTooSmallForEnterDelay_NothingTyped` left no dangling references: `grep -rn CtxExpiresDuringEnterDelay` over `*.go` and `*.md` returns nothing.
- `SubmitConfirmed` remains false on every error return including the `appendEvent` one, matching its doc; the CLI's unconfirmed-submit warning is unreachable on error paths because the error return precedes it, so no double message.
- The new CLI tests discriminate: the `PaneSendKeys`-failure test asserts the note is present, the budget test asserts it is absent, and both assert non-zero exit. The budget test's generous `--timeout-ms 2000 / --enter-delay-ms 60000` pairing is deterministic against a real stub subprocess and carries no flake risk.
- `send_defaults_sync_test.go`'s `t.Skipf` on a missing surface halts the whole test (fail-open for a vendored checkout), which the file's own comment states plainly and which matches the existing convention in `internal/config/defaults_sync_test.go` (three `t.Skipf` sites for the same reason). Consistent with the repo, so not raised as a finding.
- The doubled `org: send: org: send:` error prefix is pre-existing on `main` (the CLI wraps an already-prefixed error); the new budget-check message inherits it. Confirmed out of scope and recorded in the plan.
- Hygiene over the fix range: no credential-shaped assignments, no `TODO`/`FIXME`/debug prints, no orphaned doc comments above the inserted test functions.

### Recommendation (updated)

- Merge: yes. Cycle-1 findings are 7/7 resolved with regression tests that fail
  on a revert. Fix NEW-1 before merge — it is a wrong and actively unsafe
  instruction to a human on a reachable path, and the fix is one field plus one
  branch. The three LOWs are worth taking in the same commit; none blocks merge.
- Tech debt: still no register row. If NEW-3's test is not added, that is the one
  item that becomes a deferral, and the drafted row should say "the
  ctx-expiry-inside-the-pre-Enter-pause branch in `Send` has no test; trigger:
  the next `fakeHerdr` change".
