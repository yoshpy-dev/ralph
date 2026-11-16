# Cross-review triage report: reseat-leader-progress

- Date: 2026-11-16
- Plan: docs/plans/active/2026-11-15-reseat-leader-progress.md
- Base branch: main (48a8a661), reviewed at 478d1d6f
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 4/5 (pipeline run 7; the cap was raised from 4 to 5 after the cycle 4 ACTION_REQUIRED, which does not increment the cycle)
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Since the cycle 4 review: the cycle 4 fix (a seat waiting for input is checked for leftover input first; anything there means no send and a BLOCKED), the run 7 self-review and its fix (an empty box's grey hints are not leftover), the run 7 verify, test, and sync-docs.
- Reviewer summary line: "The restart instructions can wait indefinitely on nonblocking notifications and redispatch work after an unrecorded delivery. Formatting, mirror checks, and diff checks passed; focused Go tests could not start under the read-only sandbox."
- The git state before and after the reviewer's run was identical (`cmp` of the two state files).

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] Keep hook-notice questions nonblocking after a restart (leader.md:329-331): the inbox item stops the work for every open or acked QUESTION until a DECISION comes, but the QUESTION that carries a codex hook's `note:` line is an exception that does not stop the work, and the director rule resolves it with no DECISION; a restarted leader waits for ever. | Real and new with the inbox item: it ignores the exception that 「件を上げる」 and 「運用規律」 make, and a leader that waits for a DECISION that never comes stalls with no notice. Fix: the inbox item names that QUESTION (raised by the 「運用規律」 item, with the `HOOK_SEAT:` way to answer in its body) as an exception the leader does not wait for, read from the item's first body line and `ralph org inbox show`. | internal/org/prompts/leader.md |
| 2 | [P2] Reconcile unrecorded sends before assigning new work (leader.md:195-197): a missing last-TASK line does not prove the seat got no TASK; `Org.Send` presses Enter before it appends `sent`, so an unacknowledged Enter or a failed append leaves a delivered TASK with no record, and the seat's QUESTION or BLOCKED to it is never looked at before another TASK goes out. | Real and new with the seat check's send for a seat with no line. It can repeat a TASK and pass over the seat's BLOCKED. The section has no seat start time to bound the history with, so the fix uses the screen the item already reads: if it shows work past the seat's start-up answer (the TASK text or a paste placeholder, the work's output, a reply sent), send nothing and raise a BLOCKED; only a screen with no such trace gets the TASK. A screen that scrolled the trace away still gets a TASK (an extra TASK, the old behavior), recorded as an edge. | internal/org/prompts/leader.md |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Cycle 4 history (cycle 4, cap 4)

- Date: 2026-11-16. Base main (48a8a661), reviewed at 69257dc5. Total reviewer findings: 1. After triage (cycle 4): 1 ACTION_REQUIRED, 0 WORTH_CONSIDERING, 0 DISMISSED. Committed as 48e31ba5; fixed in 3a5922e3; the cap was raised to 5.

### Cycle 4 triage context

- Since the run 5 review: the run 5 fix (a reviewer's QUESTION about a malformed gate or cross-review TASK goes to its resend item; the commits count only what neither copy of the default branch has), the cycle 4 self-review and its fix (`--` after the revisions), the cycle 4 verify and its comment fix, the cycle 4 test, and sync-docs.
- Reviewer summary line: "The recovery instructions can concatenate a new TASK with an unsubmitted message in a surviving seat. Formatting and diff checks passed; Go tests could not run because the read-only sandbox blocked creation of the build directory."
- The git state before and after the reviewer's run was identical (`cmp` of the two state files).

### Cycle 4 ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] Check for leftover input before resending a TASK (leader.md:184-186): a send that typed a TASK and stopped before Enter leaves it in the seat's input box and records no `sent` (`Send` returns `SendProgressTextTyped` first); the seat check reads the missing line as "no TASK received" and sends another, which is typed after the leftover and submitted with it. | Real: the seat check's sends to an input-waiting seat are new in this branch, and the typed protocol section already says a leftover must go to a human, not be typed over. Two messages submitted as one can garble a TASK with no notice. Fix: when the seat waits for input, read its input box first (the screen the item already reads); if text or a paste placeholder is there, send nothing, press no key, and raise a BLOCKED as the typed protocol paragraph on `(text typed but not submitted)` says. | internal/org/prompts/leader.md |

### Cycle 4 WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

### Cycle 4 DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

## Run 5 history (cycle-count 3, cap 4)

- Date: 2026-11-16. Base main (48a8a661), reviewed at 6bebbd66. Total reviewer findings: 2. After triage (run 5): 2 ACTION_REQUIRED, 0 WORTH_CONSIDERING, 0 DISMISSED. Committed as 5528e51c; both fixed in 4096ae88; the cycle went to 4.

### Run 5 triage context

- Since the cycle 3 review: the cycle 3 fix (count only replies since the seat's last TASK, check the seat's QUESTION before sending anything), the run 5 self-review and its wording fix, the run 5 verify and its fix (widen the history read), the run 5 test, and sync-docs.
- Reviewer summary line: "The recovery rules can stall reviewer work and mistake inherited default-branch commits for prior feature execution. This was a read-only review; tests and live reseat flows were not run."
- The git state before and after the reviewer's run was identical (`cmp` of the two state files).

### Run 5 ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] Dispatch recovered reviewer questions by task type (leader.md:220-222): a reviewer's QUESTION about a malformed `gate-*` or `xreview-*` TASK gets a DECISION and no TASK, but step 12 and the cross-review subsection answer such a QUESTION with a corrected TASK under a new TASK_ID; the reviewer waits for a TASK while the leader waits for its RESULT. | Real: the QUESTION check this run added sends every QUESTION to mission item 5, so the two items that resend a corrected TASK are bypassed and the org stalls with no notice. Fix: a reviewer's QUESTION that says it did not run a malformed TASK goes to step 12's or the cross-review subsection's reviewer-QUESTION item (with their resend limit and BLOCKED), told apart by the QUESTION's TASK_ID or the seat line's; every other QUESTION gets a DECISION. | internal/org/prompts/leader.md |
| 2 | [P2] Exclude inherited base commits from restart progress (reseat_progress.go:564-569): the feature worktree is made from the local default branch (`ralph-worktree.sh ensure`), while `resolveGateBase` prefers origin's; a local main ahead of origin makes its commits count as the feature's, a first start gets a section, and the plan item can take an unrelated active plan from them. | Real (checked in `scripts/ralph-worktree.sh`: `git worktree add -b <branch> <base>` with the local default branch; `resolveGateBase` tries `refs/remotes/origin/<name>` first). New with this branch, and it can make the leader go on with a plan that is not the feature's (under split, with no human look). Fix: count only commits that neither origin's nor the local default branch has (`HEAD --not <both refs that exist>`), and use the same range in the plan item's `git log` command. | internal/org/reseat_progress.go, internal/org/prompts/leader.md |

### Run 5 WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

### Run 5 DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

## Cycle 3 history (cycle 3, cap 3)

- Date: 2026-11-16. Base main (48a8a661), reviewed at cef100ae. Total reviewer findings: 2. After triage (cycle 3): 2 ACTION_REQUIRED, 0 WORTH_CONSIDERING, 0 DISMISSED. Committed as 780a7adf; both fixed in 5ab60578; the cap was raised to 4.

### Cycle 3 triage context

- Since the third-run review: the fix that never discards a reply (skip only actions shown done; a fix of unknown state is asked again), the cycle 3 self-review, verify, test, and sync-docs.
- Reviewer summary line: "Recovery can mistake historical replies for current completion and overlook pending clarification questions. Formatting and diff checks passed, but targeted Go tests could not start because the read-only sandbox prevented creation of the build directory."
- The git state before and after the reviewer's run was identical (`cmp` of the two state files).

### Cycle 3 ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] Exclude replies from previous org lifetimes (leader.md:194-197): an org restarted after disband that reuses a TASK_ID can match a previous life's RESULT, because the agmsg team `ralph-<org_id>` keeps its history; unfinished work is read as done. | Real: the last-TASK item is limited to the current life, but the history lookup matches only seat and TASK_ID. Reading unfinished work as done hides it, and the lookup is new in this branch. Fix: count only replies whose time is after that seat's last TASK time in 「座席ごとの最後の TASK」 (a reply to that TASK cannot come before it). | internal/org/prompts/leader.md |
| 2 | [P2] Recover clarification questions before retrying tasks (leader.md:195-196): a seat's QUESTION that the previous leader's monitor marked read is ignored, and the seat check falls through to resending the TASK without answering it. | Real and new with the resend rule. It leans toward reporting (the seat would ask again), but the fix is one clause in the same sentence as finding 1, so it is fixed with it rather than recorded: before resending, look in the same history for that seat's QUESTION after its last TASK time and answer it (a DECISION) first. | internal/org/prompts/leader.md |

### Cycle 3 WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

### Cycle 3 DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

## Third-run history (cycle-count 2, cap 3)

- Date: 2026-11-16. Base main (48a8a661), reviewed at 8eeb7f69. Total reviewer findings: 2. After triage (third run): 1 ACTION_REQUIRED, 1 WORTH_CONSIDERING, 0 DISMISSED. Finding 1 fixed (replies are never discarded); finding 2 went to tech-debt (row 526) and Known gaps.

### Third-run triage context

- Since the cycle 2 review: the fix d85fc0ef (PR head comparison, skipping a found reply when a later TASK exists, unknown TASK_ID for cut Details), the third-run self-review (Merge: yes), verify (partial-pass), test (pass), and sync-docs.
- Reviewer summary line: "The restart instructions can discard unhandled failures and block recovery of interrupted slices with uncommitted edits. This was a static review; tests and live herdr execution were not run."
- The git state before and after the reviewer's run was identical (`cmp` of the two state files).
- The user's rule for sorting: a hole this change brings in that makes a notice disappear is fixed; a gap that leans toward reporting (an extra action, a BLOCKED to a human) goes to tech-debt and the PR's Known gaps. Finding 1 and the cycle 2 finding 2 pull in opposite directions (discarding a reply can drop a failure; replaying it can repeat work); the rule picks replaying, with a check that skips an action already done.

### Third-run ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] Require task-specific evidence before discarding a reply (leader.md:202-205): a seat's BLOCKED that arrived while the leader was sending an independent TASK to another seat is discarded by the restarted leader because a later TASK exists; an unresolved failure is dropped. | Real: the condition d85fc0ef added uses any seat's later send as proof of handling. Dropping a BLOCKED makes a notice disappear, and the condition is new in this branch. Fix: never discard a matching reply; treat it as just arrived, and before each action it calls for, check whether that action is already visible (the plan's place, the branch's commits, a TASK already sent for it) and skip only that action. The duplicate risk the cycle 2 finding 2 named stays as a reporting-side gap. | internal/org/prompts/leader.md |

### Third-run WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 2 | [P2] Include an ownership handoff when resuming dirty slices (leader.md:299-301): a reseat that interrupts an implementer before it commits leaves uncommitted edits (org panes turn off the WIP checkpoint); the resent TASK, built from the plan and commits, has no `RESUME:` paths, so the new implementer treats the edits as pre-existing and returns BLOCKED. | Real, and new with this branch's resend rule, but it leans toward reporting: the implementer's safe refusal raises BLOCKED, so a human or the leader sees it; nothing is lost silently. Under the user's rule it goes to tech-debt and the PR's Known gaps; the payoff is to name the leftover paths from `git status --porcelain` in a `RESUME:` header when resending an implementer slice. | internal/org/prompts/leader.md |

### Third-run DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

## Cycle 2 history

- Date: 2026-11-16. Base main (48a8a661), reviewed at b9dd5714. Cycle 2/2 (cap reached). Total reviewer findings: 3. After triage (cycle 2): 3 ACTION_REQUIRED, 0 WORTH_CONSIDERING, 0 DISMISSED. Committed as 51119734; all three fixed in d85fc0ef; the cap was raised to 3.

### Cycle 2 triage context

- Since cycle 1: the cross-review fix ec10ba6f (the section's last TASK per seat and the RESULTs; leader.md's TASK_ID matching, PR state branches, plan advisory check), the cycle 2 self-review fix (unanswered review/gate TASKs are re-asked with a new TASK_ID; the last-TASK line keeps the TASK_ID one word), and the cycle 2 sync-docs.
- Reviewer summary line: "Recovery can report an outdated PR as complete, replay already-handled work, and reconstruct incorrect task identities from truncated records. Targeted Go tests could not run because the read-only sandbox prevented creation of the temporary build directory."
- The git state before and after the reviewer's run was identical (`cmp` of the two state files).
- Checked by the triager: `truncateForDetails` (internal/org/verbs.go) cuts the sent text at 200 bytes and appends `...(truncated)`; a TASK_ID header that starts after byte 200, or crosses it, is lost or carries the marker.

### Cycle 2 ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] Verify the existing PR contains the gated commit (leader.md:248-253): an OPEN PR that still points at commit A while the gate passed for local commit B makes the leader skip step 14's push and report completion; the RESULT claims HEAD B with a PR holding only A, and the new work stays unpushed. | Real: the gate item reads only the PR's state, not its head. New work is lost while the org reports done, a notice that disappears; the rule is new in this branch. Fix: read the PR's head (`headRefOid`); when an OPEN PR's head is not the gate's head, push as step 14 does (no force) and confirm the PR's head equals the gate's head before the RESULT; a MERGED PR whose head is not the gate's head is BLOCKED. | internal/org/prompts/leader.md |
| 2 | [P2] Distinguish handled replies before replaying history (leader.md:187-191): on a leader-only restart, an idle reviewer's last BLOCKED may already have made the previous leader send a repair TASK to the implementer; treating it as new repeats the failure handling (for a failed gate, also the archive revert). | Real: the seat check treats every matching reply as unhandled. Repeating a failure path redoes work and can undo a step already redone; the rule is new in this branch. Fix: a found reply counts as unhandled only when no TASK to any seat in 「座席ごとの最後の TASK」 was sent after the reply's time; otherwise the previous leader handled it. | internal/org/prompts/leader.md |
| 3 | [P2] Preserve task identity independently of truncated details (reseat_progress.go:240-244): a TASK whose TASK_ID header comes after byte 200, or crosses it, yields `-` or an ID with `...(truncated)`, which cannot match the RESULT and can make the leader resend completed work under a wrong identity. | Real: confirmed `truncateForDetails` behavior. A wrong identity hides which work was done; the reader is new in this branch. Fix: when the Details were cut (the marker is present) and the TASK_ID is missing or carries the marker, show the TASK_ID as unknown (a fixed word the rule names), and the rule treats an unknown TASK_ID like a seat whose answer cannot be matched: read the seat's screen and, if idle, decide from 「branch の commit」 and the plan as for an unanswered slice, never resend under a guessed ID. | internal/org/reseat_progress.go, internal/org/prompts/leader.md |

### Cycle 2 WORTH_CONSIDERING

None.

### Cycle 2 DISMISSED

None.

## Cycle 1 history

- Date: 2026-11-16. Base main (48a8a661), reviewed at bd7c6a29. Cycle 1/2. Total reviewer findings: 4. After triage (cycle 1): 4 ACTION_REQUIRED, 0 WORTH_CONSIDERING, 0 DISMISSED. Committed as 0d660e77; all four fixed in ec10ba6f.

### Cycle 1 triage context

- Active plan: docs/plans/active/2026-11-15-reseat-leader-progress.md (D2, D7, AC6, AC9; the Progress checklist records S1–S4, the three self-review runs and their fixes)
- Self-review report: docs/reports/self-review-2026-11-15-reseat-leader-progress.md (three runs; the last fix, N4/N5, was inline)
- Verify report: docs/reports/verify-2026-11-15-reseat-leader-progress.md (partial-pass after the V1 fix)
- Test report: docs/reports/test-2026-11-16-reseat-leader-progress.md (pass)
- Implementation context summary: the leader's task gets a fenced 「立て直しの経過」 section on a re-seat (reseat_progress.go, StartFeature), and leader.md's rule tells the re-seated leader how to resume: 「座席の確認」 (status, read the screen, wait or check history), 計画, gate (stop seats, recheck, `gh pr list --head <branch> --state all`, step 14 or 15), 受信箱, 座席.
- Reviewer: codex gpt-6-astra, reasoning effort xhigh, `-c sandbox_mode=read-only exec review --ignore-rules --base main`. Its summary line: "The restart instructions contain recovery-path defects affecting task correlation, PR completion reporting and configured plan review. Formatting and diff checks passed, but targeted Go tests could not run because the read-only sandbox prevented creation of the build directory."
- The git state before and after the reviewer's run was identical (`cmp` of the two state files).
- Checked by the triager: `Org.Send` records the message in the manifest as a `sent` event whose Details hold the text (`TYPE: TASK`, `TASK_ID: …`; internal/org/verbs.go, `truncateForDetails(p.Text)`); it types into the seat's pane and does not post to agmsg, so agmsg history holds the seats' replies but not the leader's TASKs. The plan advisory's outcome is recorded only on the approval QUESTION's `ADVISORY:` line (leader.md, 手順 3 and 「計画助言」).

### Cycle 1 ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] Recover outgoing TASK boundaries from the manifest (leader.md:181-186): when only the leader is re-seated while a worker is idle, the procedure looks for the last outgoing TASK in agmsg history, but `Org.Send` records `EventSent` in the manifest and never posts to agmsg, so the leader cannot tell which RESULT belongs to the current task; repeated work or an obsolete result. | Real: confirmed in verbs.go. Introduced by this branch's 「座席の確認」 (the second self-review fix). A wrong correlation hides that a seat's work was lost or redone. Fix: the section carries, per seat, the last TASK sent in the current life (TASK_ID and time) from the manifest's `sent` events, and the rule matches agmsg RESULT/BLOCKED by that TASK_ID. | internal/org/reseat_progress.go, internal/org/prompts/leader.md |
| 2 | [P2] Preserve the RESULT handoff when a PR already exists (leader.md:220-221): a leader that stopped after `gh pr create` but before escalating RESULT is sent to step 15; step 14's RESULT (PR URL, gate SHAs, doc follow-ups) is never raised and the org disbands without it. | Real: the gate item skips all of step 14 when a PR exists. The director loses the hand-off, a notice that disappears. Fix: skip only PR creation; the section lists this life's RESULT items (all states), and if none names the PR the leader raises step 14's RESULT before step 15. | internal/org/reseat_progress.go, internal/org/prompts/leader.md |
| 3 | [P2] Distinguish closed PRs before skipping publication (leader.md:219-221): `gh pr list --state all` also returns a PR closed without merge, so the leader skips step 14 and heads to disband with no open or merged PR. | Real: the rule treats any PR as published. The feature silently goes undelivered. Fix: read the PR's state; skip creation only for OPEN or MERGED; with only CLOSED-unmerged PRs, do not recreate and raise BLOCKED to the director. | internal/org/prompts/leader.md |
| 4 | [P2] Verify plan advisory completion before skipping step 1 (leader.md:205-208): a leader re-seated after committing its plan but before the plan advisory goes to steps 2–3 or 5, bypassing a configured advisory. | Real: the advisory belongs to step 1 and its only record is the approval QUESTION's `ADVISORY:` line. A configured review is skipped silently. Fix: in the committed-plan branch, if no open/acked QUESTION for that plan is in the section, run the advisory as 「計画助言」 says (when the switch is on) before steps 2–3 or 5; if such a QUESTION is pending, the advisory ran: wait for the answer. | internal/org/prompts/leader.md |

### Cycle 1 WORTH_CONSIDERING

None.

### Cycle 1 DISMISSED

None.
