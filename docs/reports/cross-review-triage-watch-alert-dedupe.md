# Cross-review triage report: watch-alert-dedupe

- Date: 2026-11-12
- Plan: docs/plans/active/2026-11-12-watch-alert-dedupe.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 3/3 (cap reached; the cap was raised from 2 to 3 after cycle 2, and `cycle-count.json` still counts 2 by the cap-raise rule)
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-12-watch-alert-dedupe.md (D1–D6, AC1–AC15; the Progress checklist records S8, S9, 3bb1828b, S10, S11)
- Self-review report: docs/reports/self-review-2026-11-12-watch-alert-dedupe.md (cycle 3, second run, Merge: yes)
- Verify report: docs/reports/verify-2026-11-12-watch-alert-dedupe.md (cycle 3, pass)
- Implementation context summary: a pending alert lives in the watch-status file until it is escalated or cleared; escalation creates the inbox item first and the status file is saved at the end of the cycle. A watch that stops between the two leaves a pending entry whose alert is already an item. Before this branch that entry only produced a replayed escalation that `escalateRecordedItem` absorbs; this branch's fold (D1) now also attaches new firings to it, so a stale entry can swallow a new firing after its item was resolved. That is a regression of this branch toward silence, which the branch's rule (fail toward alerting) forbids.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] After a watch stops between escalating a pending alert and saving its status, the pending entry survives; once its item is resolved, a re-firing without leader activity folds into that stale entry, `checkDeadman` replays the old alert_id, `escalateRecordedItem` treats the resolved item as handled, and the entry is deleted: the new anomaly is neither sent nor recorded, and stays suppressed while active. | Real and introduced by this branch's fold. The fix reconciles the watch-status file with the inbox at the start of each cycle, before the seats are evaluated: a pending alert whose alert_id the inbox already holds (as an item's alert_id or on a recurred event) was escalated by a watch that stopped before saving, so it is dropped (and noted in Escalated). The fold then never sees a stale entry, and the inbox branch decides by the item's current state (open or acked: a recurrence; resolved: a new ALERT, as the messaging contract requires). | internal/org/watch.go, internal/org/watch_recurrence.go, internal/org/watch_test.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Reviewer note

codex could not run the Go tests in its read-only sandbox; diff checks and the mirrored-file comparisons passed.

## Earlier cycles

- Cycle 1/2 (18bb7424): 4 findings, all ACTION_REQUIRED (P2): a re-firing folded into a pending alert the leader had already answered; merge-authored edits missed by `git log`; a dangling `origin/HEAD` stopping commit-path detection; an escalation replay counted twice. All four were fixed in S8 (907afde3) and confirmed by the cycle 2 self-review, verify, and test.
- Cycle 2/2 (7aee125e): 1 finding, ACTION_REQUIRED (P2): with `diff.relative=true` and a subdirectory cwd the commit listing came back cwd-relative. Fixed in S10 (ec538952) and S11 (e90c85da, `-c diff.relative=false` so git older than 2.28 works), confirmed by the cycle 3 self-review, verify, and test. The cap was raised from 2 to 3 for it.

## Decision

Three pipeline runs are done and an ACTION_REQUIRED finding is left. Following the user's standing instruction of 2026-10-11 (fix ACTION_REQUIRED findings without asking; when the cap is reached, raise it and re-run), the finding is fixed and the full pipeline re-runs from /self-review as run 4. In `cycle-count.json` (cycle 2, cap 3 after the cycle 2 cap raise), cycle 2 is below the cap, so the skill's non-cap re-run applies: the counter goes to 3, and the next /cross-review reads it as the last run under cap 3.
