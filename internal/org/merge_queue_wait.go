package org

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// The wait for the merge queue of `ralph org merge --wait-checks <d>` and
// what follows it (Merge, merge_run.go; S3b of plan
// 2026-11-20-org-merge-queue-verb, D1, D2, D5 to D7; spec
// docs/specs/2026-11-19-org-merge-queue.md, FR-6 to FR-11, NFR-3 and NFR-4).
// Once mergeRun.enqueue (merge_queue_run.go) left the PR in the merge queue
// and --wait-checks has time left, mergeLocked releases regate.lock and
// returns a mergeQueueWaitFor, and Merge calls mergeThroughQueue. The wait
// runs on the same mergeWait as the waits of merge_wait.go (its budget,
// counted from the first wait of the Merge, its interval, its lines, its
// poll, and its test clock) and reads only the PR's merge queue state
// (readMergeQueuePR) every mergeWaitInterval, through runDetached. It does
// not run the checks again, since step 9 refuses a PR that is MERGED (D2). A
// read that finds the PR MERGED at the gate's head into origin's default
// branch ends with pr_merged recorded, with via=merge_queue, under the
// manifest lock only and once (recordQueueMerge, D7), and then with the
// cleanup of the branch's feature worktree under regate.lock taken again
// (queueMergeCleanup, D1). Every other end records nothing (NFR-3): the PR
// merged at another head or into another base, the PR out of the queue and
// not merged, the time used up with the PR in the queue, which is left there,
// or a last read that failed.

const (
	// mergeQueueWaitedFor is what the wait for the merge queue adds to
	// mergeWait.waitedFor, which mergeWait.waited names.
	mergeQueueWaitedFor = "the merge queue"
	// mergeQueueWaitWhen says when ctx was done, for mergeInterruptedThen,
	// once a read of the wait for the merge queue ended the wait.
	mergeQueueWaitWhen = "while it waited for the merge queue"
	// mergeQueueRecordVia ends the Details of a pr_merged that
	// recordQueueMerge appends (FR-8).
	mergeQueueRecordVia = " via=merge_queue"
)

// mergeQueueWaitFor is what the wait for the merge queue needs once
// mergeRun.enqueue left the PR in the queue: the run, the result so far
// (Queued set), and the PR's entry in the queue as the enqueue or the read
// before it gave it, nil when GitHub gave none.
type mergeQueueWaitFor struct {
	run   mergeRun
	res   MergeResult
	entry *mergeQueueEntry
}

// mergeQueueRead is a read of the wait for the merge queue that did not end
// the wait: pr when it found the PR in the queue, err when it failed.
type mergeQueueRead struct {
	pr  mergeQueuePR
	err error
}

// mergeThroughQueue is Merge once q's PR is in the merge queue and w has
// time left, with regate.lock released: the wait (waitForQueue), then, for a
// PR the queue merged at the gate's head into origin's default branch, the
// record (recordQueueMerge) and, with p.RepoRoot set and ctx not done, the
// cleanup under regate.lock taken again (queueMergeCleanup). With ctx done
// once a read found the PR merged, the record is still made, the cleanup is
// skipped with a Refused that says so (mergeCleanupInterrupted), and the error
// says Merge was interrupted and how the merge ended (mergeInterruptedThen);
// with ctx done before a read ended the wait, or once one ended it with
// Failed, it is the error of queueWaitInterrupted. The other error is a
// pr_merged that could not be recorded (Merged set, Recorded not).
func (o *Org) mergeThroughQueue(ctx context.Context, p MergeParams, w *mergeWait, q mergeQueueWaitFor) (MergeResult, error) {
	res, merged, interrupted := q.run.waitForQueue(ctx, w, q)
	if !merged {
		if interrupted || ctx.Err() != nil {
			return res, q.run.queueWaitInterrupted(ctx, res)
		}
		return res, nil
	}
	res, err := o.recordQueueMerge(ctx, res, p.HintStateDir)
	switch {
	case err != nil || p.RepoRoot == "":
		// No cleanup: the merge was not recorded, or no repo root was given.
	case ctx.Err() != nil:
		res.Cleanup = &CleanupResult{Branch: res.Branch, Head: res.Head, Refused: mergeCleanupInterrupted(context.Cause(ctx),
			res.Branch, p.HintStateDir)}
	default:
		res.Cleanup, res.CleanupErr = o.queueMergeCleanup(ctx, p, res)
	}
	if ctx.Err() != nil {
		return res, mergeInterruptedThen(ctx, mergeQueueWaitWhen, res, err)
	}
	return res, err
}

// waitForQueue is the wait for the merge queue (FR-6, FR-7) on w, with
// regate.lock released. It starts w's budget when no wait of this Merge did
// before, so what is left is counted from the first wait, and writes one line
// when it starts; then, again and again, it sleeps w.interval, or what is left
// of the budget when that is less, and reads the PR's merge queue state
// (readQueueWaiting), until a read finds the PR MERGED or out of the queue,
// the budget is used up, or ctx is done (mergeWait.poll). A read that finds
// the PR in the queue, at whatever head, and a read that fails let the wait
// go on (FR-11). When it ends it writes one line that says why and how long
// it waited, and returns q.res with the outcome: queueDecided's for a read
// that ended the wait, merged when that found the PR MERGED at the gate's
// head into the default branch; at the end of the budget, queueUnread's when
// the last read failed, else queuedAtEnd's (FR-10); and q.res as it is with
// interrupted when ctx was done, or the sleep failed, before a read ended it.
func (run mergeRun) waitForQueue(ctx context.Context, w *mergeWait, q mergeQueueWaitFor) (res MergeResult, merged, interrupted bool) {
	start := w.now()
	if w.started.IsZero() {
		w.started, w.deadline = start, start.Add(w.budget)
	}
	if !slices.Contains(w.waitedFor, mergeQueueWaitedFor) {
		w.waitedFor = append(w.waitedFor, mergeQueueWaitedFor)
	}
	where := "at a position GitHub did not give"
	if q.entry != nil {
		where = "at position " + strconv.Itoa(q.entry.Position)
	}
	w.line(fmt.Sprintf("PR #%d is in the merge queue of %s %s, so ralph org merge waits for it to merge for up to %s "+
		"(--wait-checks %s), with regate.lock released, reading the PR's merge queue state every %s", run.r.prNumber,
		run.r.defaultBranch, where, mergeWaitDuration(max(w.deadline.Sub(start), 0)), w.budget, w.interval))
	var last *mergeQueueRead
	var decided *mergeQueuePR
	ended, stopped := w.poll(ctx, func() (string, bool) {
		pr, err := run.readQueueWaiting(ctx)
		switch {
		case err == nil && (pr.State == "MERGED" || !pr.InQueue):
			// A read that ends the wait counts even with ctx done: what it
			// found is GitHub's, whatever ctx says.
			decided = &pr
			return run.queueReadEnded(pr), false
		case ctx.Err() != nil:
			return "", true
		}
		last = &mergeQueueRead{pr: pr, err: err}
		return "", false
	})
	end := fmt.Sprintf("stopped waiting for PR #%d in the merge queue after %s: %s", run.r.prNumber,
		mergeWaitDuration(w.now().Sub(start)), ended)
	switch {
	case decided != nil:
		w.line(end)
		res, merged = run.queueDecided(q.res, *decided, q.entry)
		return res, merged, false
	case stopped || ctx.Err() != nil:
		w.line(end + "; nothing was recorded")
		return q.res, false, true
	case last != nil && last.err != nil:
		w.line(end + "; its last read failed")
		return run.queueUnread(q.res, last.err, w), false, false
	}
	entry := q.entry
	if last != nil {
		entry = last.pr.Entry
	}
	w.line(end + "; it is still in the merge queue")
	return run.queuedAtEnd(q.res, entry, w), false, false
}

// readQueueWaiting is one read of the PR's merge queue state in the wait for
// the queue (readMergeQueuePR): through runDetached, as every gh call after
// the enqueue (NFR-4, AC13), so for the real commands in a process group of
// its own, bounded by the gh timeout, under ctx, so ctx done stops it. A gh
// that the timeout stopped gives an error that says so.
func (run mergeRun) readQueueWaiting(ctx context.Context) (mergeQueuePR, error) {
	r := run.r
	readCtx, cancel := context.WithTimeout(ctx, run.m.ghTimeout)
	defer cancel()
	pr, err := readMergeQueuePR(readCtx, run.m.runDetached, run.m.dir, r.apiHost, r.repo, r.prNumber)
	var ghErr *mergeQueueGHError
	if err != nil && readCtx.Err() != nil && ctx.Err() == nil && errors.As(err, &ghErr) {
		return mergeQueuePR{}, fmt.Errorf("%s timed out after %s", ghErr.cmd, run.m.ghTimeout)
	}
	return pr, err
}

// queueReadEnded says, for the line that ends the wait, what a read that
// ended it found: the PR merged, into which base at which head, or out of
// the queue and not merged.
func (run mergeRun) queueReadEnded(pr mergeQueuePR) string {
	if pr.State == "MERGED" {
		return fmt.Sprintf("it was merged into %s at the head %s", pr.BaseRefName, shortMergeSHA(pr.HeadRefOID))
	}
	return fmt.Sprintf("it left the merge queue without being merged (it is %s)", pr.State)
}

// queueDecided is res once a read of the wait, pr, found the PR MERGED or
// out of the merge queue, with Queued, QueuePosition, and EnqueuedAt
// cleared: MERGED at the gate's head into origin's default branch sets Merged
// and MergeCommit (mergeCommit.oid, "" when GitHub gave none that is a commit
// SHA) and reports merged (FR-8); MERGED at another head or into another base
// is Failed (AC8: otherHeadClause, otherBaseClause; queueMergedElsewhere); out
// of the queue and not MERGED is Failed (FR-9: queueLeft, with entry, the
// PR's entry in the queue when the wait started). Nothing is recorded here.
func (run mergeRun) queueDecided(res MergeResult, pr mergeQueuePR, entry *mergeQueueEntry) (_ MergeResult, merged bool) {
	r := run.r
	res.Queued, res.QueuePosition, res.EnqueuedAt = "", 0, ""
	switch {
	case pr.State == "MERGED" && pr.HeadRefOID != r.head:
		res.Failed = run.queueMergedElsewhere(run.otherHeadClause(pr.HeadRefOID), "")
	case pr.State == "MERGED" && pr.BaseRefName != r.defaultBranch:
		res.Failed = run.queueMergedElsewhere(run.otherBaseClause(pr.BaseRefName))
	case pr.State == "MERGED":
		res.Merged, res.MergeCommit = true, pr.MergeCommit
		return res, true
	default:
		res.Failed = run.queueLeft(pr, entry)
	}
	return res, false
}

// queueMergedElsewhere is the Failed of a PR the merge queue merged in a way
// Merge does not record (AC8, NFR-3), which merged says: that nothing was
// recorded, to check that merge with gh pr view --json mergeOutcomeFields, to
// run ralph org merge-order again, with listed right after it, and to go on
// with the other branches, as mergedElsewhere says for gh pr merge. Escaped.
func (run mergeRun) queueMergedElsewhere(merged, listed string) string {
	return PrintableInboxText(fmt.Sprintf(mergePrefix+"the merge queue merged PR #%d %s, and nothing was recorded: check that "+
		"merge yourself (%s --json %s), run %s again%s, and go on with the other branches", run.r.prNumber, merged,
		run.viewCmd(), mergeOutcomeFields, run.m.verbCommand("merge-order"), listed), false)
}

// queueLeft is the Failed of a PR that a read of the wait found out of the
// merge queue and not MERGED (FR-9): GitHub's latest removal of it from the
// queue strictly after entry's enqueuedAt (latestMergeQueueRemovalAfter; any
// removal when entry is nil), with its reason and actor
// (mergeQueueRemovalText), and the next step FR-5b leaves (requeueNext); or,
// when no such removal was read, that none was, and to check the PR. Nothing
// was recorded. Escaped.
func (run mergeRun) queueLeft(pr mergeQueuePR, entry *mergeQueueEntry) string {
	r := run.r
	var after time.Time
	var since string
	if entry != nil {
		after = entry.EnqueuedAt
		since = " after it entered the queue at " + after.UTC().Format(time.RFC3339)
	}
	left := fmt.Sprintf(mergePrefix+"PR #%d (%s) left the merge queue of %s without being merged and is %s now", r.prNumber,
		r.prURL, r.defaultBranch, pr.State)
	if removal, ok := latestMergeQueueRemovalAfter(pr.Removed, after); ok {
		return PrintableInboxText(fmt.Sprintf("%s: it was %s%s, so nothing was recorded: fix what made the queue remove it, then %s",
			left, mergeQueueRemovalText(removal), since, run.requeueNext()), false)
	}
	return PrintableInboxText(fmt.Sprintf("%s, and no removal from the queue%s was read, so nothing was recorded: check the PR "+
		"(%s --json %s) and run %s to see the branch's state", left, since, run.viewCmd(), mergeOutcomeFields,
		run.m.verbCommand("merge-order")), false)
}

// queueUnread is res once the budget of the wait was used up and its last
// read failed (err): Failed, since whether the PR is still in the merge queue
// is unknown (FR-11), with the next step a queued outcome gives (queueNext)
// and how long Merge waited (mergeWait.waited); Queued, QueuePosition, and
// EnqueuedAt cleared. Nothing was recorded.
func (run mergeRun) queueUnread(res MergeResult, err error, w *mergeWait) MergeResult {
	res.Queued, res.QueuePosition, res.EnqueuedAt = "", 0, ""
	res.Failed = PrintableInboxText(fmt.Sprintf(mergePrefix+"%v at the last read of the wait, so whether PR #%d is still in the "+
		"merge queue is unknown and nothing was recorded: it may still be in the merge queue, so %s", err, run.r.prNumber,
		run.queueNext()), false) + w.waited()
	return res
}

// queuedAtEnd is res once the budget of the wait was used up and its last
// read found the PR in the merge queue, or the wait read nothing: Queued,
// with entry, the last read's entry, or the one the wait started from
// (queueEntryWhere), the next step (queueNext), and how long Merge waited
// (mergeWait.waited) (FR-10). Nothing was recorded, and the PR is left in
// the queue: ralph never removes it.
func (run mergeRun) queuedAtEnd(res MergeResult, entry *mergeQueueEntry, w *mergeWait) MergeResult {
	r := run.r
	where := queueEntryWhere(&res, entry)
	res.Queued = PrintableInboxText(fmt.Sprintf(mergePrefix+"PR #%d (%s) is still in the merge queue of %s and not merged yet, "+
		"and %s, so nothing was recorded and ralph org merge left it in the queue: %s", r.prNumber, res.PRURL, r.defaultBranch,
		where, run.queueNext()), false) + w.waited()
	return res
}

// queueWaitInterrupted is Merge's error when ctx was done during the wait
// for the merge queue and no read found the PR merged: with res.Failed, the
// end a read found before ctx was done, as mergeInterruptedThen words it;
// else that nothing was recorded and the PR may still be in the merge queue,
// with the next step a queued outcome gives (queueNext, D6). Escaped.
func (run mergeRun) queueWaitInterrupted(ctx context.Context, res MergeResult) error {
	if res.Failed != "" {
		return mergeInterruptedThen(ctx, mergeQueueWaitWhen, res, nil)
	}
	return fmt.Errorf(mergePrefix+"interrupted (%v) while it waited for PR #%d in the merge queue: %s", context.Cause(ctx),
		run.r.prNumber, PrintableInboxText(fmt.Sprintf("nothing was recorded, and PR #%d may still be in the merge queue: %s",
			run.r.prNumber, run.queueNext()), false))
}

// recordQueueMerge records res, a merge the merge queue made at the gate's
// head into origin's default branch (FR-8), once (D7). Under the manifest
// lock, with ctx's cancellation dropped as recordMerge has it, it reads the
// ledger and appends pr_merged with recordMerge's Details and via=merge_queue
// after them (mergeRecordDetails, mergeQueueRecordVia), unless the ledger
// holds a pr_merged of the same branch, PR, and head after the branch's
// latest gate record (queueMergeRecorded), as another ralph org merge that
// waited for the same queue appends: then it appends nothing and sets Note.
// Either way it sets Recorded. The error is recordMerge's (mergeRecordErr),
// for a ledger that cannot be read and a pr_merged that cannot be appended.
func (o *Org) recordQueueMerge(ctx context.Context, res MergeResult, hintStateDir string) (MergeResult, error) {
	var already bool
	err := withManifestLockContext(context.WithoutCancel(ctx), filepath.Dir(o.Manifest.Path()), func() error {
		ledger, err := readMergeLedger(o.Manifest.Path())
		if err != nil {
			return err
		}
		if already = queueMergeRecorded(ledger, res); already {
			return nil
		}
		return o.appendEvent(ManifestEvent{TS: o.now(), OrgID: res.OrgID, SeatID: "", Event: EventPRMerged,
			Details: mergeRecordDetails(res) + mergeQueueRecordVia})
	})
	if err != nil {
		return res, o.mergeRecordErr(res, err, hintStateDir)
	}
	res.Recorded = true
	if already {
		res.Note = PrintableInboxText(fmt.Sprintf("the ledger held %s of PR #%d at the head %s after the branch's latest gate record "+
			"already, so ralph org merge did not record it again: another ralph org merge that waited for the same merge queue "+
			"recorded it", EventPRMerged, res.PRNumber, shortMergeSHA(res.Head)), false)
	}
	return res, nil
}

// queueMergeRecorded reports whether l holds a pr_merged (EventPRMerged, an
// org-level event that is not a dry run) of res's branch, PR, and head after
// the branch's latest gate record (collectMergeTargets), or anywhere when the
// branch has none: the record D7 does not write twice. A pr_merged from
// before that gate record is of an earlier merge and does not count. Its
// Details are read as FollowupRecordedMerge reads them, branch= from the first
// token (recordedMergeBranch) and the head from head= (recordedMergeHead), and
// the PR from a pr= token.
func queueMergeRecorded(l mergeLedger, res MergeResult) bool {
	after := -1
	targets, _ := collectMergeTargets(l)
	if i := slices.IndexFunc(targets, func(t *mergeTarget) bool { return t.branch == res.Branch }); i >= 0 && targets[i].gate != nil {
		after = targets[i].gate.pos
	}
	pr := "pr=" + strconv.Itoa(res.PRNumber)
	for _, ev := range l.events[after+1:] {
		if ev.Event != EventPRMerged || ev.DryRun || ev.SeatID != "" {
			continue
		}
		branch, branchOK := recordedMergeBranch(ev.Details)
		head, headOK := recordedMergeHead(ev.Details)
		if branchOK && headOK && branch == res.Branch && head == res.Head && slices.Contains(strings.Fields(ev.Details), pr) {
			return true
		}
	}
	return false
}

// queueMergeCleanup is the cleanup of the feature worktree of res.Branch once
// Merge recorded a merge the merge queue made (D1). The wait ran with
// regate.lock released, so it takes the lock again as mergeLocked takes it
// (acquireHeldFileLock, regateLockWait), empties the marker an earlier regate
// left, runs mergeCleanup with the lock held, and releases it. When another
// ralph org regate, merge, or cleanup holds the lock, or the lock cannot be
// taken or its marker emptied, the cleanup does not run, and the result holds
// only a Refused that says why and names ralph org cleanup of the branch
// (orgRetryCommand). The merge stands either way.
func (o *Org) queueMergeCleanup(ctx context.Context, p MergeParams, res MergeResult) (*CleanupResult, string) {
	stateDir := filepath.Dir(o.Manifest.Path())
	skipped := func(why, next string) (*CleanupResult, string) {
		return &CleanupResult{Branch: res.Branch, Head: res.Head, Refused: PrintableInboxText(fmt.Sprintf("%s, so the feature "+
			"worktree of the branch %s was not cleaned up once pr_merged was recorded: %s %s", why, res.Branch, next,
			orgRetryCommand("cleanup", res.Branch, p.HintStateDir)), false)}, ""
	}
	lock, err := acquireHeldFileLock(stateDir, regateLockFile, "regate", regateLockWait)
	switch {
	case errors.Is(err, errFileLockHeld):
		return skipped(fmt.Sprintf("another ralph org regate, merge, or cleanup runs on this ledger (%s is held)",
			regateLockPathIn(stateDir)), "once it ends, run")
	case err != nil:
		// err names the lock's path, or its dir.
		return skipped(fmt.Sprintf("regate.lock could not be taken (%v)", err), "run")
	}
	defer func() { _ = lock.release() }()
	if err := lock.f.Truncate(0); err != nil {
		return skipped(fmt.Sprintf("the marker in regate.lock could not be emptied (%v)", err), "run")
	}
	return o.mergeCleanup(ctx, lock, p, res)
}
