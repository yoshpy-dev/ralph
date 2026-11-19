package org

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// The queue path of `ralph org merge` up to the enqueue (Merge, merge_run.go;
// S3a of plan 2026-11-20-org-merge-queue-verb, D4 to D6; spec
// docs/specs/2026-11-19-org-merge-queue.md, FR-4, FR-5, FR-5b, FR-7 and
// FR-10). Once every check of checkMerge passed on a default branch with a
// merge queue (mergeReady.queue), with regate.lock still held, Merge reads
// the PR's merge queue state (readMergeQueuePR, merge_queue.go), refuses a
// read that does not agree with step 9 of the checks, and adds the PR to the
// queue at the gate's head with enqueuePullRequest (enqueueMergeQueuePR),
// unless it is there already at that head (FR-5) or GitHub removed it from
// the queue after the branch's latest gate record and --requeue was not
// given (FR-5b). It ends with the PR in the queue (MergeResult.Queued) and
// records nothing; when --wait-checks has time left, Merge then waits for the
// queue to merge the PR with regate.lock released (merge_queue_wait.go). Both
// GraphQL calls here run through MergeOrderExec in ralph's process group, as
// the checks' do, each bounded by the gh timeout; no gh pr merge, read-back,
// or --disable-auto runs on this path.

// mergeGateTime is the time of a branch's gate record whose ts is gateTS, or
// of accepted, the acceptance of its failure for a branch merge-order lists
// as accepted, when that is later: FR-5b's latest gate record, which a
// removal from the merge queue is compared with (mergeQueueRemovalAfterGate,
// D4 of the plan). A ts that does not parse as RFC 3339 gives the zero time,
// which counts every removal, so a ts ralph cannot read refuses a re-enqueue
// without --requeue rather than lets one through.
func mergeGateTime(gateTS string, accepted *GateAcceptance) time.Time {
	at, err := time.Parse(time.RFC3339, gateTS)
	if err != nil {
		return time.Time{}
	}
	if accepted == nil {
		return at
	}
	acceptedAt, err := time.Parse(time.RFC3339, accepted.TS)
	if err != nil {
		return time.Time{}
	}
	if acceptedAt.After(at) {
		return acceptedAt
	}
	return at
}

// enqueue is Merge on the queue path once its checks passed: res holds what
// the checks found, and the outcome is added to it.
//
//  1. It reads the PR's merge queue state (readQueue); a read that fails is
//     refused as a gh call of the checks is.
//  2. The read must agree with step 9 of the checks (queueReadDisagrees): the
//     PR is OPEN, at the gate's head, and merges into origin's default
//     branch.
//  3. A PR in the queue already, at the gate's head, is not added again: the
//     outcome is Queued with the entry the read gave (FR-5).
//  4. A PR GitHub removed from the queue after the branch's latest gate
//     record (mergeQueueRemovalAfterGate on mergeReady.gateAt) is refused
//     unless requeue, MergeParams.Requeue, is set (FR-5b); with it,
//     QueueNote says so.
//  5. enqueuePullRequest at the gate's head, once (sendEnqueue).
//
// Refused clears res. With Queued, entry is the PR's entry in the queue, the
// answer's or the read's, which the wait for the queue starts from (nil when
// GitHub gave none). With ctx done, the error says Merge was interrupted:
// before step 5 sent the mutation, with nothing merged or enqueued and an
// empty result (queueInterruptedBefore); once it was sent, that the PR may
// be in the merge queue, or is there, with res as far as it got
// (queueInterruptedAfter). It records nothing.
func (run mergeRun) enqueue(ctx context.Context, res MergeResult, requeue bool) (_ MergeResult, entry *mergeQueueEntry, _ error) {
	if ctx.Err() != nil {
		return MergeResult{}, nil, run.queueInterruptedBefore(ctx, "after the checks passed, before the PR's merge queue state "+
			"was read")
	}
	pr, refused, err := run.readQueue(ctx)
	if err != nil || refused != "" {
		return MergeResult{Refused: refused}, nil, err
	}
	if refused := run.queueReadDisagrees(pr); refused != "" {
		return MergeResult{Refused: refused}, nil, nil
	}
	if pr.InQueue {
		res.QueueNote = PrintableInboxText(fmt.Sprintf("PR #%d was in the merge queue already at the head the gate passed, so "+
			"ralph org merge did not add it again", run.r.prNumber), false)
		return run.queued(res, false, pr.Entry), pr.Entry, nil
	}
	if removal, ok := mergeQueueRemovalAfterGate(pr.Removed, run.r.gateAt); ok {
		if !requeue {
			return MergeResult{Refused: run.removedAfterGate(removal)}, nil, nil
		}
		res.QueueNote = PrintableInboxText(fmt.Sprintf("PR #%d was %s, after %s, and --requeue added it to the merge queue again",
			run.r.prNumber, mergeQueueRemovalText(removal), run.gateWhen()), false)
	}
	if ctx.Err() != nil {
		return MergeResult{}, nil, run.queueInterruptedBefore(ctx, "before enqueuePullRequest was sent")
	}
	return run.sendEnqueue(ctx, res, pr.ID)
}

// readQueue is the read of the PR's merge queue state before the enqueue
// (readMergeQueuePR), through the runner of the checks, in ralph's process
// group, bounded by the gh timeout. It returns the interruption's error when
// ctx is done, and the refusal, escaped, of a read whose values failed their
// check (errMergeQueueInput), of a gh that is not found, failed, or timed out
// (ghFailed, as for a gh call of the checks), and of an answer it cannot
// read.
func (run mergeRun) readQueue(ctx context.Context) (mergeQueuePR, string, error) {
	r := run.r
	readCtx, cancel := context.WithTimeout(ctx, run.m.ghTimeout)
	defer cancel()
	pr, err := readMergeQueuePR(readCtx, run.m.run, run.m.dir, r.apiHost, r.repo, r.prNumber)
	timedOut := err != nil && readCtx.Err() != nil && ctx.Err() == nil
	unknown := fmt.Sprintf("whether PR #%d is in the merge queue is unknown", r.prNumber)
	var ghErr *mergeQueueGHError
	switch {
	case ctx.Err() != nil:
		return mergeQueuePR{}, "", run.queueInterruptedBefore(ctx, "while the PR's merge queue state was read")
	case err == nil:
		return pr, "", nil
	case errors.Is(err, errMergeQueueInput):
		return mergeQueuePR{}, PrintableInboxText(fmt.Sprintf(mergePrefix+"the merge queue state of PR #%d was not read (%v), so "+
			"nothing was merged: merge the PR through the queue yourself", r.prNumber, err), false), nil
	case errors.As(err, &ghErr):
		cause := ghErr.err
		if timedOut {
			cause = fmt.Errorf("timed out after %s: %w", run.m.ghTimeout, cause)
		}
		refused, err := run.m.ghFailed(ctx, r, ghErr.cmd, cause, unknown)
		return mergeQueuePR{}, PrintableInboxText(refused, false), err
	}
	return mergeQueuePR{}, PrintableInboxText(fmt.Sprintf(mergePrefix+"%v, so %s and nothing was merged: %s", err, unknown,
		run.m.ghAccess(r)), false), nil
}

// mergeQueueReadLead starts each refusal of queueReadDisagrees, %d being the
// PR's number.
const mergeQueueReadLead = mergePrefix + "the read of PR #%d's merge queue state after the checks found "

// queueReadDisagrees is the refusal of a read of the PR's merge queue state
// that does not agree with step 9 of the checks (viewPR): the PR is not
// OPEN, its head is not the gate's (in the merge queue or not), or its base
// is not origin's default branch; "" when it agrees. Escaped: the state,
// head, and base are GitHub's.
func (run mergeRun) queueReadDisagrees(pr mergeQueuePR) string {
	r := run.r
	var refused string
	switch {
	case pr.State != "OPEN":
		refused = fmt.Sprintf(mergeQueueReadLead+"it %s, not OPEN, so nothing was merged and it was not added to the merge queue: "+
			"run %s to see the branch's state", r.prNumber, pr.State, run.m.verbCommand("merge-order"))
	case pr.HeadRefOID != r.head && pr.InQueue:
		refused = fmt.Sprintf(mergeQueueReadLead+"it in the merge queue at the head %s, not %s, the head the gate passed, so "+
			"nothing was merged and it was not added to the merge queue again: run %s again and follow the branch's Detail",
			r.prNumber, shortMergeSHA(pr.HeadRefOID), shortMergeSHA(r.head), run.m.verbCommand("merge-order"))
	case pr.HeadRefOID != r.head:
		refused = fmt.Sprintf(mergeQueueReadLead+"its head at %s, not %s, the head the gate passed, so nothing was merged and it "+
			"was not added to the merge queue: run %s again and follow the branch's Detail", r.prNumber,
			shortMergeSHA(pr.HeadRefOID), shortMergeSHA(r.head), run.m.verbCommand("merge-order"))
	case pr.BaseRefName != r.defaultBranch:
		refused = fmt.Sprintf(mergeQueueReadLead+"it merging into %s, not origin's default branch %s, so nothing was merged and "+
			"it was not added to the merge queue: change its base (gh pr edit %d --base %s -R %s), and once its CI passed, %s",
			r.prNumber, pr.BaseRefName, r.defaultBranch, r.prNumber, r.defaultBranch, r.repoArg, run.m.again())
	default:
		return ""
	}
	return PrintableInboxText(refused, false)
}

// removedAfterGate is the refusal of FR-5b: GitHub removed the PR from the
// merge queue (removal, the latest such) after the branch's latest gate
// record, so the queue has tried the gate's outcome once, and the PR is not
// added again without --requeue. Escaped; mergeQueueRemovalText escapes
// GitHub's reason and actor.
func (run mergeRun) removedAfterGate(removal mergeQueueRemoval) string {
	r := run.r
	return PrintableInboxText(fmt.Sprintf(mergePrefix+"PR #%d was %s, after %s, so ralph org merge did not add it to the merge "+
		"queue again and nothing was merged: %s", r.prNumber, mergeQueueRemovalText(removal), run.gateWhen(), run.requeueNext()),
		false)
}

// requeueNext is the next step once GitHub removed the PR from the merge
// queue after the branch's latest gate record, which FR-5b refuses to add
// again without --requeue: a new gate record of the branch, or ralph org
// merge --requeue, each with MergeParams.HintStateDir (orgRetryCommand). Not
// escaped: the sentence that holds it is.
func (run mergeRun) requeueNext() string {
	return fmt.Sprintf("get a new gate record of the branch (%s, or a gate of a new push), or run %s --requeue to add it again "+
		"with the same gate on purpose", orgRetryCommand("regate", run.r.branch, run.m.hintStateDir),
		orgRetryCommand("merge", run.r.branch, run.m.hintStateDir))
}

// gateWhen names, for the texts of FR-5b, what a removal was compared with:
// the branch's latest gate record, and its acceptance for an accepted
// branch, with the time of mergeReady.gateAt and the skew it allows
// (mergeQueueClockSkew), or with no time when it could not be read, which
// counts every removal.
func (run mergeRun) gateWhen() string {
	what, at := "the branch's latest gate record", "at"
	if run.r.accepted != nil {
		what, at = "the branch's latest gate record and its acceptance", "the later at"
	}
	if run.r.gateAt.IsZero() {
		return what + ", whose time could not be read, so every removal counts"
	}
	return fmt.Sprintf("%s (%s %s; a removal up to %s before it counts as after it)", what, at,
		run.r.gateAt.UTC().Format(time.RFC3339), mergeQueueClockSkew)
}

// sendEnqueue sends enqueuePullRequest for the PR, whose node ID is prID, at
// the gate's head (enqueueMergeQueuePR), once, through the runner of the
// checks, in ralph's process group, bounded by the gh timeout (FR-4), and
// returns res with its outcome:
//
//   - its entry in the queue: Queued, with the entry the answer gave and no
//     read after it (D5 of the plan);
//   - values its check refused (errMergeQueueInput), or gh not found: Refused
//     with an empty result, since nothing was sent;
//   - gh failed or timed out, or the answer has no entry: Failed, since the
//     PR may or may not be in the queue, with the next step (queueNext).
//
// With Queued it returns the answer's entry too. With ctx done once the
// mutation was sent, it returns res as far as it got and the error of
// queueInterruptedAfter.
func (run mergeRun) sendEnqueue(ctx context.Context, res MergeResult, prID string) (MergeResult, *mergeQueueEntry, error) {
	r := run.r
	enqueueCtx, cancel := context.WithTimeout(ctx, run.m.ghTimeout)
	defer cancel()
	entry, err := enqueueMergeQueuePR(enqueueCtx, run.m.run, run.m.dir, r.apiHost, prID, r.head)
	timedOut := err != nil && enqueueCtx.Err() != nil && ctx.Err() == nil
	var ghErr *mergeQueueGHError
	var queuedEntry *mergeQueueEntry
	switch {
	case errors.Is(err, errMergeQueueInput):
		return MergeResult{Refused: PrintableInboxText(fmt.Sprintf(mergePrefix+"PR #%d was not added to the merge queue (%v), so "+
			"nothing was merged: merge the PR through the queue yourself", r.prNumber, err), false)}, nil, nil
	case err == nil:
		res, queuedEntry = run.queued(res, true, &entry), &entry
	case ctx.Err() != nil:
		// The mutation may have reached GitHub: queueInterruptedAfter says so.
	case errors.As(err, &ghErr) && errors.Is(ghErr.err, exec.ErrNotFound):
		return MergeResult{Refused: PrintableInboxText(run.m.ghMissing(r, ghErr.err), false)}, nil, nil
	default:
		what := err.Error()
		if timedOut && ghErr != nil {
			what = fmt.Sprintf("%s timed out after %s", ghErr.cmd, run.m.ghTimeout)
		}
		res.Failed = PrintableInboxText(fmt.Sprintf(mergePrefix+"%s, so whether PR #%d was added to the merge queue is unknown, "+
			"and nothing was recorded: %s", what, r.prNumber, run.queueNext()), false)
	}
	if ctx.Err() != nil {
		return res, queuedEntry, run.queueInterruptedAfter(ctx, res)
	}
	return res, queuedEntry, nil
}

// queued is res with Queued set for the PR, in the merge queue at the gate's
// head: added says whether Merge added it (FR-4) or found it there (FR-5),
// and entry is its entry, nil when GitHub gave none. QueuePosition and
// EnqueuedAt are set from entry (queueEntryWhere).
func (run mergeRun) queued(res MergeResult, added bool, entry *mergeQueueEntry) MergeResult {
	r := run.r
	queue := "the merge queue of " + r.defaultBranch
	how := "is in " + queue + " already"
	if added {
		how = "was added to " + queue
	}
	where := queueEntryWhere(&res, entry)
	res.Queued = PrintableInboxText(fmt.Sprintf(mergePrefix+"PR #%d (%s) %s at the head %s, the head the gate passed, and %s; it "+
		"is not merged yet, so nothing was recorded: %s", r.prNumber, res.PRURL, how, shortMergeSHA(r.head), where,
		run.queueNext()), false)
	return res
}

// queueEntryWhere sets res's QueuePosition and EnqueuedAt (UTC, RFC 3339)
// from entry, the PR's entry in the merge queue, or to 0 and "" when entry is
// nil, and says where the PR is in the queue, for a Queued text.
func queueEntryWhere(res *MergeResult, entry *mergeQueueEntry) string {
	if entry == nil {
		res.QueuePosition, res.EnqueuedAt = 0, ""
		return "GitHub gave no entry of it, so its position is unknown"
	}
	res.QueuePosition, res.EnqueuedAt = entry.Position, entry.EnqueuedAt.UTC().Format(time.RFC3339)
	return fmt.Sprintf("it is at position %d, enqueued at %s", res.QueuePosition, res.EnqueuedAt)
}

// queueNext is the next step of an outcome that leaves the PR in the merge
// queue, or may (D6 of the plan), worded once for MergeResult.Queued, the
// Failed of an enqueuePullRequest whose outcome is unknown, and the error of
// an interruption once it was sent: run ralph org merge again, which reads
// the queue first and does not add the PR twice (FR-5), and once GitHub
// merged the PR, ralph org cleanup, each with MergeParams.HintStateDir
// (orgRetryCommand). Not escaped: the sentence that holds it is.
func (run mergeRun) queueNext() string {
	return fmt.Sprintf("run %s again, which reads the merge queue first and adds the PR only when it is not there, and once "+
		"GitHub merged the PR, run %s to clean up the branch's feature worktree",
		orgRetryCommand("merge", run.r.branch, run.m.hintStateDir), orgRetryCommand("cleanup", run.r.branch, run.m.hintStateDir))
}

// queueInterruptedBefore is Merge's error when ctx is done on the queue path
// before enqueuePullRequest was sent, at step: nothing was merged, and the PR
// was not added to the merge queue.
func (run mergeRun) queueInterruptedBefore(ctx context.Context, step string) error {
	return fmt.Errorf(mergePrefix+"interrupted (%v) %s: nothing was merged, and PR #%d was not added to the merge queue",
		context.Cause(ctx), step, run.r.prNumber)
}

// queueInterruptedAfter is Merge's error when ctx was done once
// enqueuePullRequest had been sent: that Merge was interrupted, then how the
// enqueue ended, as mergeInterruptedAfterStart words it for gh pr merge:
// res.Queued when the answer gave the PR's entry, else that the PR may be in
// the merge queue, with the next step (queueNext). Escaped.
func (run mergeRun) queueInterruptedAfter(ctx context.Context, res MergeResult) error {
	outcome := fmt.Sprintf("PR #%d may be in the merge queue, and nothing was recorded: %s", run.r.prNumber, run.queueNext())
	if res.Queued != "" {
		outcome = strings.TrimPrefix(res.Queued, mergePrefix)
	}
	return fmt.Errorf(mergePrefix+"interrupted (%v) once enqueuePullRequest had been sent, and then: %s", context.Cause(ctx),
		PrintableInboxText(outcome, false))
}
