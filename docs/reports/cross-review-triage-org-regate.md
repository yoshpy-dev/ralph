# Cross-review triage report: org-regate

- Date: 2026-10-17
- Plan: docs/plans/active/2026-10-16-org-regate.md
- Base branch: main (670ec219)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 2
- After triage: ACTION_REQUIRED=2, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-16-org-regate.md (Approved sha256:f841204a7bc0)
- Self-review report: docs/reports/self-review-2026-10-16-org-regate.md (second pass, Merge: yes, LOW 4 fixed in 0b6ba232)
- Verify report: docs/reports/verify-2026-10-16-org-regate.md (pass)
- Implementation context summary: the reviewed head is 413ab307. regate merges origin's default branch tip into a disbanded feature org's branch with `git merge --no-ff`, gates the merge commit, records the result under the org with via=regate, and pushes. merge-order and regate read the gate records through `collectMergeTargets` (`internal/org/mergeorder.go`).

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] The merge can overwrite an ignored file in the worktree at a path the default branch newly tracks: the clean-tree check does not see ignored files, git merge overwrites them, and `reset --keep` cannot bring their content back. The reviewer recommends `--no-overwrite-ignore` | Real, and the loss is of uncommitted data (an ignored `.env` or local config). Reproduced on git 2.49.0: with a feature branch that ignores `secret.txt` and a main that tracks it, `git merge --no-ff` replaced the local content, and so did `git merge --no-ff --no-overwrite-ignore` (exit 0, MERGE_HEAD absent): that option guards the fast-forward checkout only, so the recommended fix does not stop it. The fix is a check before the merge that refuses, with nothing merged or recorded, when a path the merge would write (added on the tip's side) or a directory above it is already on disk in the worktree | internal/org/regate_merge.go (regateMergeTip and the checks before it), docs/plans/active/2026-10-16-org-regate.md (D3) |
| 2 | [P2] With an org_id reused from branch A to branch B and both lives disbanded, a regate of A writes A's record under the org, but an unreadable gate record is attached to the org's latest binding (B), so a newer unreadable `gate_failed` of A leaves A's older pass as its latest: `passedOn` can then allow UP_TO_DATE or PUSH_ONLY and merge-order can say ready | Real, and new with this PR: before regate a gate record was written only while its org ran, so the latest binding was its branch; regate writes records for an older binding of a reused org_id, which makes the attachment of an unreadable record ambiguous. Narrow (a reused org_id and an unreadable record), but it can let a failed head be pushed or merged. The fix is the conservative one: an unreadable gate record of an org is attached to every branch that org has had a complete binding or a readable gate record for so far, so none of them keeps an older pass | internal/org/mergeorder.go (collectMergeTargets), internal/org/regate.go (readRegateBranch reads the same), docs/plans/active/2026-10-16-org-regate.md (D2) |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
