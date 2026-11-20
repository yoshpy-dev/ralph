package org

import (
	"context"
	"fmt"
)

// The merge queue in merge-order (FR-12 of spec
// docs/specs/2026-11-19-org-merge-queue.md, plan
// docs/plans/active/2026-11-20-org-merge-queue-order.md): with
// MergeOrderParams.ReadMergeQueue, MergeOrder reads the merge queue of
// origin's default branch once (readMergeOrderQueue) into mergeRemote.queue,
// reports the read (MergeOrderResult.MergeQueue), and notes a read that
// failed (mergeOrderQueueNotes); the Detail texts classifyGatePassed and
// classifyAccepted give a branch whose gate's base origin's default branch
// moved past, by that read; and the note of mergeOrderListNotes about several
// ready branches when the branch has a queue (mergeQueueSeveralReadyNote).

// The values of MergeOrderQueue.State: the states of a read of the merge
// queue (mergeQueueLookup) as MergeOrderResult.MergeQueue carries them.
const (
	// MergeOrderQueuePresent: origin's default branch has a merge queue.
	MergeOrderQueuePresent = string(mergeQueuePresent)
	// MergeOrderQueueAbsent: GraphQL's mergeQueue of origin's default branch
	// is null, so it has no queue GraphQL sees.
	MergeOrderQueueAbsent = string(mergeQueueAbsent)
	// MergeOrderQueueUnknown: the read failed, so whether origin's default
	// branch has a queue is unknown.
	MergeOrderQueueUnknown = string(mergeQueueUnknown)
)

// MergeOrderQueue is the merge queue of origin's default branch as MergeOrder
// read it with MergeOrderParams.ReadMergeQueue (MergeOrderResult.MergeQueue).
type MergeOrderQueue struct {
	// State is MergeOrderQueuePresent, MergeOrderQueueAbsent or
	// MergeOrderQueueUnknown.
	State string `json:"state"`
	// Method is, with present, the merge method of the queue's configuration
	// (MERGE, SQUASH or REBASE); "", and left out of the JSON, when the
	// configuration names none or another value, and with another state.
	Method string `json:"method,omitempty"`
	// Why says, with unknown, why the read failed: origin's repository could
	// not be read, or gh's read of the queue failed; "", and left out of the
	// JSON, with another state.
	Why string `json:"why,omitempty"`
}

// present reports whether q is a read that found a merge queue; false for a
// nil q, no read.
func (q *MergeOrderQueue) present() bool {
	return q != nil && q.State == MergeOrderQueuePresent
}

// mergeOrderQueueResult is the read of the merge queue as
// MergeOrderResult.MergeQueue carries it: nil for mergeQueueNotRead, no read.
func mergeOrderQueueResult(read mergeQueueLookup) *MergeOrderQueue {
	if read.state == mergeQueueNotRead {
		return nil
	}
	return &MergeOrderQueue{State: string(read.state), Method: read.method, Why: read.why}
}

// readMergeOrderQueue is MergeOrder's read of the merge queue of def, origin's
// default branch, with MergeOrderParams.ReadMergeQueue. Origin's repository
// is p.Origin when it names one; otherwise ReadMergeOrigin reads it in p.Dir
// with run, with an ssh origin's host resolved by `ssh -G` through p.Exec
// (realMergeDetachedExec when it is nil). lookupMergeQueue then reads the
// queue with one `gh api graphql` in p.Dir through run. The get-url and the
// read are each bounded by p.GHTimeout (defaultMergeOrderGHTimeout, 15 s, when
// it is zero or less), as merge-order's other gh calls are. An origin whose
// get-url fails or times out, or whose URL names no repository, makes the
// read mergeQueueUnknown with why, and no gh runs. A read that fails once gh
// runs ends its why with the ssh key fix when ssh -G did not resolve an ssh
// origin's host (mergeSSHAliasHint), as doctor and ralph org merge do. The
// error is for ctx done while origin or the queue was read; nothing is
// classified then.
func readMergeOrderQueue(ctx context.Context, p MergeOrderParams, run MergeOrderExec, def string) (mergeQueueLookup, error) {
	timeout := p.GHTimeout
	if timeout <= 0 {
		timeout = defaultMergeOrderGHTimeout
	}
	origin := p.Origin
	if origin == (MergeOrigin{}) {
		o, problem, err := ReadMergeOrigin(ctx, p.Dir, run, p.Exec, timeout)
		switch problem {
		case MergeOriginInterrupted:
			return mergeQueueLookup{}, mergeOrderInterrupted(ctx, mergeOriginReadStep)
		case MergeOriginTimedOut, MergeOriginGetURLFailed:
			return mergeQueueLookup{state: mergeQueueUnknown, why: fmt.Sprintf("%s failed (%s), so origin's repository is unknown",
				mergeGuaranteeGetURL, mergeGuaranteeErrText(err))}, nil
		case MergeOriginNoRepository:
			return mergeQueueLookup{state: mergeQueueUnknown, why: "origin's URL names no host and owner/repo (a local path, " +
				"file://, or a scheme other than https, http, ssh and git), so there is no GitHub repository to read"}, nil
		}
		origin = o
	}
	read := lookupMergeQueue(ctx, MergeEnforcementParams{
		Dir: p.Dir, Exec: run, Timeout: timeout, Host: origin.apiHost, Owner: origin.repo.owner, Name: origin.repo.name,
		DefaultBranch: def,
	})
	if ctx.Err() != nil {
		return mergeQueueLookup{}, mergeOrderInterrupted(ctx, "gh read the merge queue of origin/"+def)
	}
	if read.state == mergeQueueUnknown {
		read.why += mergeSSHAliasHint(origin.originHost, origin.apiHost, origin.unresolvedWhy)
	}
	return read, nil
}

// mergeOrderQueueNotes are MergeOrder's notes about remote.queue, the read of
// the merge queue of origin's default branch: with a read that failed
// (mergeQueueUnknown), one that says so, that a branch whose gate base that
// branch moved past is listed as rebase, as with no queue, and why the read
// failed; none for any other read.
func mergeOrderQueueNotes(remote mergeRemote) []string {
	if remote.queue.state != mergeQueueUnknown {
		return nil
	}
	return []string{fmt.Sprintf("the merge queue of origin/%s could not be read, so a branch whose gate base origin/%s moved "+
		"past is listed as rebase, as with no queue: %s", remote.defaultBranch, remote.defaultBranch, remote.queue.why)}
}

// mergeQueueSeveralReadyNote is the note of mergeOrderListNotes about n ready
// or accepted branches, accepted of them accepted, when origin/def has a merge
// queue (FR-12 and FR-13 of spec docs/specs/2026-11-19-org-merge-queue.md):
// the queue tests each PR on the latest origin/def before it merges it, so
// merging one leaves the others as they are, and they can all be put in the
// queue first, then waited for one at a time.
func mergeQueueSeveralReadyNote(def string, n, accepted int) string {
	which := "ready"
	if accepted > 0 {
		which = "ready or accepted"
	}
	return fmt.Sprintf("%d branches are %s, and origin/%s has a merge queue, so merging one leaves the others %s: add them all "+
		"to the queue first (ralph org merge --branch <branch> --wait-checks 0 for each), then wait for them one at a time with "+
		"a budget (ralph org merge --branch <branch> --wait-checks <duration>, such as 60m)", n, which, def, which)
}

// mergeQueueMovedText is what the Detail of a ready or accepted branch says
// when origin's default branch moved past its gate's base and has a merge
// queue (mergeQueuePresent): that the queue tests the PR on the default
// branch's latest tip before it merges it, so the branch needs no regate.
func mergeQueueMovedText(remote mergeRemote) string {
	return fmt.Sprintf("origin/%s moved to %s since, and its merge queue tests the PR on the latest origin/%s before it merges "+
		"it, so no regate is needed", remote.defaultBranch, shortMergeSHA(remote.defaultHead), remote.defaultBranch)
}

// mergeQueueUnreadText ends the Detail of a rebase branch whose gate's base
// origin's default branch moved past when the read of that branch's merge
// queue failed (mergeQueueUnknown): that the queue could not be read, that a
// queue would make the branch ready, and why the read failed, so that a gh
// that cannot read the queue (a lapsed login, say) does not read as a reason
// to regate. It is "" for every other read.
func mergeQueueUnreadText(remote mergeRemote) string {
	if remote.queue.state != mergeQueueUnknown {
		return ""
	}
	return fmt.Sprintf("; the merge queue of origin/%s could not be read, and with one the branch would be ready: %s",
		remote.defaultBranch, remote.queue.why)
}
