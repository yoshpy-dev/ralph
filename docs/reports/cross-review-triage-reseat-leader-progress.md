# Cross-review triage report: reseat-leader-progress

- Date: 2026-11-16
- Plan: docs/plans/active/2026-11-15-reseat-leader-progress.md
- Base branch: main (48a8a661; the branch merged origin/main at 3ac29c5b, 95b5e805, e3914c93; reviewed at bd7c6a29)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 4
- After triage: ACTION_REQUIRED=4, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-15-reseat-leader-progress.md (D2, D7, AC6, AC9; the Progress checklist records S1–S4, the three self-review runs and their fixes)
- Self-review report: docs/reports/self-review-2026-11-15-reseat-leader-progress.md (three runs; the last fix, N4/N5, was inline)
- Verify report: docs/reports/verify-2026-11-15-reseat-leader-progress.md (partial-pass after the V1 fix)
- Test report: docs/reports/test-2026-11-16-reseat-leader-progress.md (pass)
- Implementation context summary: the leader's task gets a fenced 「立て直しの経過」 section on a re-seat (reseat_progress.go, StartFeature), and leader.md's rule tells the re-seated leader how to resume: 「座席の確認」 (status, read the screen, wait or check history), 計画, gate (stop seats, recheck, `gh pr list --head <branch> --state all`, step 14 or 15), 受信箱, 座席.
- Reviewer: codex gpt-6-astra, reasoning effort xhigh, `-c sandbox_mode=read-only exec review --ignore-rules --base main`. Its summary line: "The restart instructions contain recovery-path defects affecting task correlation, PR completion reporting and configured plan review. Formatting and diff checks passed, but targeted Go tests could not run because the read-only sandbox prevented creation of the build directory."
- The git state before and after the reviewer's run was identical (`cmp` of the two state files).
- Checked by the triager: `Org.Send` records the message in the manifest as a `sent` event whose Details hold the text (`TYPE: TASK`, `TASK_ID: …`; internal/org/verbs.go, `truncateForDetails(p.Text)`); it types into the seat's pane and does not post to agmsg, so agmsg history holds the seats' replies but not the leader's TASKs. The plan advisory's outcome is recorded only on the approval QUESTION's `ADVISORY:` line (leader.md, 手順 3 and 「計画助言」).

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] Recover outgoing TASK boundaries from the manifest (leader.md:181-186): when only the leader is re-seated while a worker is idle, the procedure looks for the last outgoing TASK in agmsg history, but `Org.Send` records `EventSent` in the manifest and never posts to agmsg, so the leader cannot tell which RESULT belongs to the current task; repeated work or an obsolete result. | Real: confirmed in verbs.go. Introduced by this branch's 「座席の確認」 (the second self-review fix). A wrong correlation hides that a seat's work was lost or redone. Fix: the section carries, per seat, the last TASK sent in the current life (TASK_ID and time) from the manifest's `sent` events, and the rule matches agmsg RESULT/BLOCKED by that TASK_ID. | internal/org/reseat_progress.go, internal/org/prompts/leader.md |
| 2 | [P2] Preserve the RESULT handoff when a PR already exists (leader.md:220-221): a leader that stopped after `gh pr create` but before escalating RESULT is sent to step 15; step 14's RESULT (PR URL, gate SHAs, doc follow-ups) is never raised and the org disbands without it. | Real: the gate item skips all of step 14 when a PR exists. The director loses the hand-off, a notice that disappears. Fix: skip only PR creation; the section lists this life's RESULT items (all states), and if none names the PR the leader raises step 14's RESULT before step 15. | internal/org/reseat_progress.go, internal/org/prompts/leader.md |
| 3 | [P2] Distinguish closed PRs before skipping publication (leader.md:219-221): `gh pr list --state all` also returns a PR closed without merge, so the leader skips step 14 and heads to disband with no open or merged PR. | Real: the rule treats any PR as published. The feature silently goes undelivered. Fix: read the PR's state; skip creation only for OPEN or MERGED; with only CLOSED-unmerged PRs, do not recreate and raise BLOCKED to the director. | internal/org/prompts/leader.md |
| 4 | [P2] Verify plan advisory completion before skipping step 1 (leader.md:205-208): a leader re-seated after committing its plan but before the plan advisory goes to steps 2–3 or 5, bypassing a configured advisory. | Real: the advisory belongs to step 1 and its only record is the approval QUESTION's `ADVISORY:` line. A configured review is skipped silently. Fix: in the committed-plan branch, if no open/acked QUESTION for that plan is in the section, run the advisory as 「計画助言」 says (when the switch is on) before steps 2–3 or 5; if such a QUESTION is pending, the advisory ran: wait for the answer. | internal/org/prompts/leader.md |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
