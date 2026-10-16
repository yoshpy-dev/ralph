# Cross-review triage report: org-regate

- Date: 2026-10-17
- Plan: docs/plans/active/2026-10-16-org-regate.md
- Base branch: main (670ec219)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-16-org-regate.md (Approved sha256:260c539299bc)
- Self-review report: docs/reports/self-review-2026-10-16-org-regate.md (cycle 2, Merge: yes)
- Verify report: docs/reports/verify-2026-10-16-org-regate.md (cycle 2, second pass, pass)
- Test report: docs/reports/test-2026-10-16-org-regate.md (cycle 2, pass)
- Implementation context summary: the reviewed head is c60b2651. Cycle 1's two ACTION_REQUIRED findings were fixed in 85558f2c, ca840a45, and c621ccc9 (the check before the merge for something ignored in its way, D3) and f793924d (an unreadable gate record attached to every branch of its org, D2). The reviewer ran in the read-only sandbox and could not build the Go tests there (it said so); its findings come from reading the code.
- The cycle-1 version of this report (cycle 1/2, the two findings above) is in git history at edb42681.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] The `reset --keep` hint of a refused HEAD can name a person's commit as an unpushed regate merge: a gate-only regate also records `via=regate`, and the hint accepts a single-parent commit. After a gate-only run on a normal commit X, a force push of origin back to X's parent satisfies every condition, and the hint tells the human to drop X. The reviewer recommends recording whether regate made the merge and leaving the hint out when that cannot be established | Real: `regateResetHint` checks only that the latest record is a readable `via=regate` one of HEAD and that HEAD's first parent is origin's tip, and `regateGate` writes `via=regate` on the gate-only path too (AC9). The dropped commit stays in the reflog, but the hint recommends dropping someone's implementation work, which D4 meant to rule out. The same root (a gate-only regate's `via=regate` is not told apart from regate's own merge) also lets the push-only exception of D4 push X again after such a force push, which the reviewer did not flag; the fix covers both | internal/org/regate_merge.go (regateResetHint, regateHeadNotOnOrigin), docs/plans/active/2026-10-16-org-regate.md (D4) |
| 2 | [P2] `regateSetTrackingRef` reads `refs/remotes/origin/<branch>` after the push and updates it from that value, so a fetch that lands between the push and the read (origin moved on to N) is overwritten back to the pushed head, and git then reports the branch as in sync | Real: with the default `remote.origin.fetch`, git push already moves the ref to the pushed head; a background fetch that then moves it to N makes the read return N, and `update-ref <ref> <head> N` succeeds and rewinds it. The window is short and a later fetch restores the ref, but a rewound tracking ref is also the expected value `git push --force-with-lease` uses. The fix is the one the reviewer gives: read the ref before the push, and after it update the ref only from that value, leaving any other value alone | internal/org/regate_merge.go (regatePush, regateSetTrackingRef) |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
