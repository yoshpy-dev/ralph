# Walkthrough: org-merge-queue-verb

- Date: 2026-11-20
- Plan: docs/plans/archive/2026-11-20-org-merge-queue-verb.md (#342, the first of two PRs)
- Spec: docs/specs/2026-11-19-org-merge-queue.md
- Author: hiroki-yoshioka

## What changed

When origin's default branch had a GitHub merge queue, `ralph org merge`
refused at step 11 with nothing done, so a director with `auto_merge` and a
merge queue could not work together. Now, on a default branch with a merge
queue and at least one required status check, `ralph org merge --branch <b>`
runs its usual checks (steps 1 to 10 and 13), skips step 12 (the merge
method comes from the queue's configuration), and puts the PR in the queue
with the GraphQL mutation `enqueuePullRequest` at the gate's head
(`expectedHeadOid`). It releases `regate.lock`, waits within the
`--wait-checks` budget (shared with the wait for checks, read every 30 s),
and once GitHub reports the PR MERGED at the gate's head on the default
branch it records `pr_merged` with `via=merge_queue` and the queue's merge
commit, then takes `regate.lock` again to clean up the feature worktree.

The other ends:

- removed from the queue after ralph added it: no record, exit 1, with the
  removal's reason and actor (FR-9)
- the budget ends (or is 0) with the PR still in the queue: no record, the PR
  stays in the queue, exit 7 (`ErrMergeQueued`); running the same command
  again keeps waiting while the default branch is where the gate left it
  (FR-10)
- the state cannot be read at the end: no record, exit 1, "may be in the
  queue" (FR-11)
- Ctrl-C while waiting: the PR stays in the queue, no record
- already in the queue at the gate's head: ralph waits without adding it
  (FR-5); removed after the branch's latest gate record: refused unless
  `--requeue` (FR-5b, with a 5-minute allowance for clock skew on the
  refusing side)

A repo without a merge queue behaves as before (NFR-1; main's merge tests
pass on HEAD except the three queue rows and three help sentences this PR
replaces).

## Reading order

1. `internal/org/merge_queue.go`: the one GraphQL query that reads the PR's
   node ID, state, head, base, merge commit, `isInMergeQueue`,
   `mergeQueueEntry`, and its `RemovedFromMergeQueueEvent`s; the
   `enqueuePullRequest` mutation (`-F number`, since `number` is `Int!`); the
   checks of the node ID and SHA; `mergeQueueClockSkew` and the removal
   helpers.
2. `internal/org/mergeenforce.go`: step 11's queue path (a queue plus at
   least one required check; the queue configuration's `mergeMethod`).
3. `internal/org/merge.go`: `MergeParams.Requeue`, the queue fields of
   `mergeReady` and `mergeCheck`, step 12 skipped on the queue path.
4. `internal/org/merge_queue_run.go`: the read before adding, step 9's
   agreement, FR-5, FR-5b, the enqueue, and a budget of 0.
5. `internal/org/merge_queue_wait.go`: the wait with the lock released,
   `recordQueueMerge` (under the manifest lock only; `queueMergeRecorded`
   keeps a second waiter from writing the same `pr_merged`, searching the
   whole ledger), and `queueMergeCleanup` (re-taking `regate.lock`).
6. `internal/cli/org_merge.go`, `exitcode.go`: `--requeue`, exit code 7, the
   `QUEUED:` block, and the help.
7. Tests: `merge_queue_test.go`, `merge_queue_path_test.go`,
   `merge_queue_run_test.go`, `merge_queue_wait_test.go`, the queue rows of
   `mergeenforce_test.go` and `merge_test.go`, and
   `internal/cli/org_merge_queue_test.go`, all against a fake gh.
8. Docs: README (the merge paragraph and the command table), the org and
   director skills' four copies (only the sentences the queue path made
   false; the two-step procedure is the second PR), the spec's FR-4 note
   (D3) and checkboxes, the 2026-10-07 spec's note, AGENTS.md, and the
   tech-debt register (a row for the GitHub behavior not tried live, the
   800-line rows, and the test gaps G1 to G6).

## How the pipeline shaped it

The pipeline ran twice.

- The plan advisory (codex) found three things, all taken: `number` must go
  with `-F`; FR-5b's clock comparison needs an allowance on the refusing
  side; and two `ralph org merge` waiting for the same queue with the lock
  released could both record the merge (D7).
- The first cross-review found that D7 looked for an earlier `pr_merged` only
  after the branch's latest gate record, so a gate or regate during the wait
  hid the first record. Fixed in 2c9a7fe6: the search covers the whole
  ledger, since a PR is merged once.
- The second cross-review had no findings.

## What was checked live, and what was not

Not run against real GitHub: this repo's owner is a personal account, which
cannot use a merge queue. By the user's decision, the queue path is built
and tested with a fake gh and is tried after the release in a downstream
repo. The two queries were checked against github.com's schema in the verify
run. The behavior the code assumes and nobody has observed (the permission
`enqueuePullRequest` needs, whether GitHub checks `expectedHeadOid` when it
adds, the removal reasons, GHES) is listed in the tech-debt register's row
for the first PR of #342.
