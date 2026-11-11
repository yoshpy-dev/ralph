# Cross-review triage report: watch-alert-dedupe

- Date: 2026-11-12
- Plan: docs/plans/active/2026-11-12-watch-alert-dedupe.md
- Base branch: main
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 4
- After triage: ACTION_REQUIRED=4, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-11-12-watch-alert-dedupe.md (D1–D6, AC1–AC15; the Progress checklist records the deviations D6/AC15, L2, L3, N5)
- Self-review report: docs/reports/self-review-2026-11-12-watch-alert-dedupe.md (third run, Merge: yes, LOW 1)
- Verify report: docs/reports/verify-2026-11-12-watch-alert-dedupe.md (pass)
- Implementation context summary: the branch makes one inbox item per (org, seat, condition) episode, suppresses the stall of a waiting leader, and limits a reserved org's scope_change to paths outside the reservation. The design rule throughout is to fail toward alerting: when the watch cannot tell, it raises. All four findings are cases where the code fails toward silence or double counting instead, so each is a real defect against that rule, and each fix is small.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] A firing that recurs in the same cycle in which the leader's activity is first seen is folded into the old pending alert, which `checkDeadman` then deletes for that activity: the new firing is neither sent nor pending, and the Active condition keeps it from firing again until a recovery. | Real: `evaluateSeat` runs before `checkDeadman` in `evaluateCycle`, so the fold happens against a pending alert whose episode the leader has already answered. It loses an alert, which D1 does not intend (a pending alert cleared by leader activity ends the episode). Fix: before folding into a pending alert, apply the same leader-activity test `checkDeadman` uses; when it holds for a non-leader subject, end that episode and raise a new ALERT. | internal/org/watch_recurrence.go, internal/org/watch.go |
| 2 | [P2] `git log --name-only` shows no paths for a merge commit, so an edit made in a merge (a conflict resolution outside the reservation) is missed once the merge is committed and the worktree is clean. | Real, though narrow (feature-org leaders rebase, and regate's merge runs after disband). D6 excluded upstream commits on purpose; the merge's own edits are the org's writes and should count. Fix: list merge commits with the dense combined diff (`--cc`), which names only paths that differ from every parent, so inherited upstream changes stay out. | internal/org/watch_scope.go |
| 3 | [P2] A dangling `origin/HEAD` (a renamed default branch with the old remote-tracking branch pruned) still resolves through `git symbolic-ref`, so every `git log ... --not <ref>` fails; with S6's retry rule the previous HEAD is then kept for ever and committed-path detection stops. | Real: the main/master fallback already verifies its ref, the symbolic one does not, and the combination with L1's retry turns a transient-failure rule into a permanent stop. Fix: verify the symbolic target with `git rev-parse --verify --quiet <ref>^{commit}` and fall through to main/master, then to the `git diff` fallback. | internal/org/watch_scope.go |
| 4 | [P2] If the watch exits after `escalateAsRecurrence` appended a recurred event but before it saved the watch-status file, the restarted watch replays the same pending alert: the lookup checks only the item's own alert_id, so the firing is counted twice, or, when the item was resolved meanwhile, a new item is created for a firing that already has a record. | Real but limited to a crash window; the identical-alert_id rule (D4 of plan 2026-10-23-watch-escalation-inbox) exists for exactly this replay and should cover the alert_ids carried by recurred events too. Fix: treat an alert_id found on any item's recurred event (any state) like the item's own alert_id, and record nothing new. | internal/org/watch.go, internal/org/watch_recurrence.go |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe

## Reviewer note

codex could not run the Go tests in its read-only sandbox (it could not create the build directory); `git diff --check` passed.
