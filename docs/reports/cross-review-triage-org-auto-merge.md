# Cross-review triage report: org-auto-merge

- Date: 2026-10-18
- Plan: docs/plans/active/2026-10-17-org-auto-merge.md
- Base branch: main (15d22b1a)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 2/2 (cap reached)
- Total reviewer findings: 1
- After triage: ACTION_REQUIRED=1, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-17-org-auto-merge.md (Approved sha256:305d97070f38)
- Self-review report: docs/reports/self-review-2026-10-17-org-auto-merge.md (cycle 2, Merge: yes, LOW 6 fixed in S9)
- Verify report: docs/reports/verify-2026-10-17-org-auto-merge.md (cycle 2, pass)
- Test report: docs/reports/test-2026-10-17-org-auto-merge.md (cycle 2, pass; its LOW T1 fixed in 425a8543)
- Implementation context summary: the reviewed head is 14b3ec6d. Cycle 1's three ACTION_REQUIRED findings were fixed in S8 (9506ec64, b2f0b534, 320b4d75), and cycle 2's self-review, verify, and test LOWs in S9 (e5831d14, 7d0d08dd), 425a8543, and the sync-docs commit 56efae4a. The reviewer (gpt-6-astra, xhigh, read-only sandbox) could not run the Go tests (the sandbox blocks the build directory) and said so. The finding was checked against `merge_run.go` and `merge.go` before it was classified.
- The cycle-1 version of this report (cycle 1/2, three findings) is in git history at b1cbe054.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P2] If someone retargets the PR after `viewPR` checks its base, a merge into that other branch can keep the gated head SHA: `--match-head-commit` does not constrain the base and the last `ls-remote` cannot see a retarget, so Merge records `pr_merged` and reports success with the protection evidence of the default branch, which received nothing. Read `baseRefName` back and require it to be `run.r.defaultBranch` before recording. | Real: step 9 (`viewPR`, `merge.go:551`) refuses a PR whose base is not the default branch, but the read-back after `gh pr merge` asks only for `state,headRefOid,mergeCommit,autoMergeRequest` (`mergeReadBackFields`, `merge_run.go:36`), and `mergeAndRecord` records a MERGED PR whose head is the gate's whatever its base is now (`merge_run.go:339-342`). The record would then say the gated head reached the default branch under the checked protection, which is what the verb exists to prevent; the same shape as cycle 1's #3 (another head). Worth fixing now: the fix is the same small change as S8b, with `baseRefName` beside `headRefOid` and a Failed with nothing recorded when it differs or is empty. | `internal/org/merge_run.go`, `internal/org/merge_run_test.go`, the plan's D6, AC7, AC8 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
