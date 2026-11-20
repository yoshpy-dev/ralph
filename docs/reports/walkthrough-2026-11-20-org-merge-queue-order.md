# Walkthrough: org-merge-queue-order

- Date: 2026-11-20
- Plan: docs/plans/archive/2026-11-20-org-merge-queue-order.md (#342, the second of two PRs)
- Spec: docs/specs/2026-11-19-org-merge-queue.md (FR-12, FR-13, AC7, AC9, AC11)
- Author: hiroki-yoshioka

## What changed

After the first PR (#352), `ralph org merge` could put a PR in origin's merge
queue, but merge-order and merge still classified a branch as `rebase` once
origin's default branch moved past its gate's base. On a queue repo every
merge therefore sent the director around the regate loop, although the queue
itself tests each PR on the latest default branch before it merges it.

Now, when origin's default branch has a merge queue (GraphQL
`mergeQueue(branch)` is set):

- `ralph org merge-order` reads the queue once per run (only the verb; the
  merge-order inside `ralph org cleanup` does not), classifies a branch whose
  gate base only moved as `ready` (or `accepted`), says so in its Detail,
  prints a `merge queue:` line after `merge guarantee:` (only when the queue
  is present or cannot be read), and carries `merge_queue` in `--json`. With
  two or more ready branches its note says to put them all in the queue first
  instead of "the others become rebase".
- `ralph org merge` reads the queue at most once per run of its checks: step
  7 reads it only when the class would change with a queue (a moved base),
  step 11 reuses that read, and step 13 on the queue path no longer requires
  the default branch's tip to be the gate's base (it still checks the default
  branch's name and the branch's tip). A re-run after exit 7 keeps waiting
  even after the default branch moved.
- The director skill has a two-stage procedure for a queue repo: stage 1 puts
  every ready/accepted branch in the queue with `--wait-checks 0` and defers
  the branches whose org still runs or whose checks are pending, retrying
  them each time a 5-minute inbox wait returns; stage 2 waits for each with
  `--wait-checks 60m`. A branch the queue merged meanwhile is cleaned up with
  `ralph org cleanup`.

With no queue (mergeQueue null) the classification, the notes, and merge's
call order for an unmoved base are as before. When the queue cannot be read
the branch stays `rebase`, with the reason in its Detail and a note.

## Reading order

1. `internal/org/mergeenforce.go`: `mergeQueueLookup` (states "", absent,
   present, unknown), `lookupMergeQueue`, and the unexported
   `MergeEnforcementParams.queue` that `CheckMergeEnforcement` uses instead of
   its own read.
2. `internal/org/mergeorder.go` and `mergeorder_queue.go`: `mergeRemote.queue`,
   the queue branch of `classifyGatePassed` and `classifyAccepted`, the Detail
   texts, `MergeOrderParams.ReadMergeQueue`, `MergeOrderResult.MergeQueue`,
   the read in `MergeOrder` and the notes.
3. `internal/org/merge.go` and `merge_queue_order.go`: step 7's
   `readQueueIfOnlyBaseMoved`, `queueReadFor` for step 11, step 13 on the
   queue path; `merge_queue_run.go`'s `queueNext` without the old condition.
4. `internal/cli/org_merge_order.go`: `ReadMergeQueue` on, `mergeQueueLine`,
   the help; `org_merge.go`'s help.
5. Tests: `mergeorder_queue_test.go`, `merge_queue_order_test.go` (AC2, AC3,
   AC4 with three branches, AC7), `director_skill_queue_test.go` (the skill's
   stderr fragments pinned to the Go texts that print them),
   `internal/cli/org_merge_order_test.go`.
6. Docs: the director skill's `merge と auto_merge`, the org skill's
   merge-order and merge rows and `merge してよい順`, README, the spec marks,
   AGENTS.md, the tech-debt register.

## How the pipeline shaped it

The pipeline ran once.

- The consult on the plan kept the three recommended choices and added three
  gaps: stage 1's `--wait-checks 0` refuses at once for a running org or
  pending checks, a queue that cannot be read must give its reason in the
  refusal, and the "others become rebase" note is false on a queue repo.
- The plan advisory (codex) found that deferred branches waited on one at a
  time would block a branch whose checks passed sooner; deferred branches are
  retried with `--wait-checks 0` at each 5-minute wake instead.
- The self-review's MEDIUM (a checks list cut at five names can hide a
  failed check) is handled in the skill: the hidden failure shows once the
  pending checks before it finish. Its root cause is pinned by
  `director_skill_queue_test.go`.
- The cross-review had no findings.

## What was checked live, and what was not

Not run against real GitHub (the repo's owner is a personal account, which
cannot use a merge queue): merge-order's GraphQL read, merge's steps 7 and
13 with a moved base, and the director's two-stage procedure. By the user's
decision they are tried after the release in a downstream repo; the
tech-debt register lists them with the first PR's items.
