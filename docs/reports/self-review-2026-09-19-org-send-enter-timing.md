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
