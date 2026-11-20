package org

import "fmt"

// The merge queue in merge-order's classification (FR-12 of spec
// docs/specs/2026-11-19-org-merge-queue.md, plan
// docs/plans/active/2026-11-20-org-merge-queue-order.md): the Detail texts
// classifyGatePassed and classifyAccepted give a branch whose gate's base
// origin's default branch moved past, by the read of that branch's merge
// queue in mergeRemote.queue (lookupMergeQueue, mergeenforce.go).

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
