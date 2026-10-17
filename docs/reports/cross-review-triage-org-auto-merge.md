# Cross-review triage report: org-auto-merge

- Date: 2026-10-18
- Plan: docs/plans/active/2026-10-17-org-auto-merge.md
- Base branch: main (15d22b1a)
- Driver: claude
- Reviewer: codex
- Triager: Claude Code (main context)
- Reviewer status: complete
- Self-review cross-ref: yes
- Cycle: 1/2
- Total reviewer findings: 3
- After triage: ACTION_REQUIRED=3, WORTH_CONSIDERING=0, DISMISSED=0

## Triage context

- Active plan: docs/plans/active/2026-10-17-org-auto-merge.md (Approved sha256:76d09b87e280)
- Self-review report: docs/reports/self-review-2026-10-17-org-auto-merge.md (cycle 1 and the S6・S7 re-review, Merge: yes)
- Verify report: docs/reports/verify-2026-10-17-org-auto-merge.md (cycle 1, pass)
- Test report: docs/reports/test-2026-10-17-org-auto-merge.md (cycle 1, pass)
- Implementation context summary: the reviewed head is 124ed5df. The reviewer (gpt-6-astra, xhigh, read-only sandbox) could not run the Go tests (the sandbox blocks the build directory) and said so. All three findings were checked against the code, the leader prompt, and GitHub's documentation before they were classified.

## ACTION_REQUIRED

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|
| 1 | [P1] The classic protection check passes a caller whose `permissions.admin` is false without reading `enforce_admins`, but a custom repository role with the "bypass branch protections" permission is not an admin and can still bypass the strict checks, so a base that moved after the last `ls-remote` is not necessarily rejected by GitHub. | Real: GitHub's "About protected branches" says that by default the restrictions of a branch protection rule "do not apply to people with admin permissions to the repository or custom roles with the 'bypass branch protections' permission", and that "Do not allow bypassing the above settings" (`enforce_admins`) is what applies them to both. `enforceClassic` returns ok for `!*admin` without reading `enforce_admins` (`mergeenforce.go:291-292`), so the merge guarantee D11 rests on can be absent while the verb reports it. Worth fixing now: the fix is to require `enforce_admins` enabled whatever `permissions.admin` says, and the non-admin path does not work on real GitHub anyway (tech-debt row on the classic non-admin path), so nothing working is lost. | `internal/org/mergeenforce.go`, `internal/org/mergeenforce_test.go`, the plan's D11 and AC5, help and docs that state the non-admin case |
| 2 | [P2] The leader escalates RESULT before it disbands (`internal/org/prompts/leader.md`, steps 14 to 16), and merge-order classifies the branch `ready` while the org still runs, so with `auto_merge = true` the director's merge is refused as "org is running" before `--wait-checks` applies, and the skill's rule to stop a branch at exit 1 and not run the verb again unasked leaves a healthy PR out until a human steps in. | Real: `classifyMergeTarget` (`mergeorder.go`) does not read `orgRunning` for `ready`, `runningRefusal` (`merge.go:480-495`) refuses and itself says to run merge again once the org is disbanded, and `/director` (`.claude/skills/director/SKILL.md:466-491`) stops the branch at that refusal. The common path of auto_merge (RESULT, then disband a moment later) therefore stops at its first merge. Worth fixing now: it defeats AC11 in its main case. | `internal/org/merge.go`, `internal/org/merge_wait.go`, `internal/org/merge_run.go`, `.claude/skills/director/SKILL.md` (4 copies), the plan's D13, AC11, AC13 |
| 3 | [P2] When someone else pushes a new head and merges it after the last origin check, `--match-head-commit` makes this gh pr merge fail, but the read-back returns MERGED, and Merge records `pr_merged` with the gate's head and reports success, though another head was merged; the read-back does not ask for `headRefOid`, so it cannot tell this case from a merge whose answer was lost. | Real: `mergeReadBackFields` is `state,mergeCommit,autoMergeRequest` (`merge_run.go:34`), and `mergeAndRecord` records on MERGED whatever `gh pr merge` returned (`merge_run.go:325-335`). The record then says the gated head was merged when it was not, which is what the merge verb exists to prevent. Worth fixing now: read `headRefOid` back and record only when it is the gate's head; otherwise report the merge as someone else's, with nothing recorded. | `internal/org/merge_run.go`, `internal/org/merge_run_test.go`, the plan's D6, AC7, AC8 |

## WORTH_CONSIDERING

| # | Reviewer finding | Triage rationale | Affected file(s) |
|---|-------------------|------------------|-------------------|

## DISMISSED

| # | Reviewer finding | Dismissal reason | Category |
|---|-------------------|------------------|----------|

Categories: false-positive, already-addressed, style-preference, out-of-scope, context-aware-safe
