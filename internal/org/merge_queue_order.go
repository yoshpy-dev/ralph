package org

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
)

// The merge queue in the checks of `ralph org merge` (FR-12 of spec
// docs/specs/2026-11-19-org-merge-queue.md; plan
// docs/plans/active/2026-11-20-org-merge-queue-order.md, D2 and D3): step 7
// of checkMerge (classify) reads the merge queue of origin's default branch
// only when a queue would make the branch ready or accepted, that is when
// only that branch's tip moved past the gate's base
// (readQueueIfOnlyBaseMoved), and keeps the read in mergeCheck.queue, which
// step 11 hands to CheckMergeEnforcement (queueReadFor), so one run of the
// checks reads the queue once at most. On the queue path step 13
// (checkOriginUnmoved, merge.go) does not compare origin's default branch's
// tip with the gate's base.

// classify is step 7 of checkMerge: merge-order's class of t on remote
// (classifyMergeTarget), and the refusal when it is not ready or accepted.
// When that class is rebase only because origin's default branch moved past
// the gate's base, readQueueIfOnlyBaseMoved reads that branch's merge queue
// once and t is classified again with the read: a queue keeps the branch
// ready or accepted, no queue keeps it rebase, and a read that failed keeps it
// rebase with why at the end of the refusal's Detail (mergeQueueUnreadText),
// so a gh that cannot read the queue does not read as a reason to regate.
// The refusal is also for a gh not found while the queue was read, and the
// error is for ctx done then.
func (m *mergeCheck) classify(ctx context.Context, r mergeReady, t *mergeTarget, remote mergeRemote) (class, refused string, err error) {
	class, detail := classifyMergeTarget(t, remote)
	read, refused, err := m.readQueueIfOnlyBaseMoved(ctx, r, t, remote, class)
	switch {
	case err != nil || refused != "":
		return "", refused, err
	case read:
		remote.queue = m.queue
		class, detail = classifyMergeTarget(t, remote)
	}
	if class != MergeClassReady && class != MergeClassAccepted {
		return class, fmt.Sprintf(mergePrefix+"%s lists the branch %s as %s, not ready or accepted, so nothing was merged: %s",
			m.verbCommand("merge-order"), m.branch, class, detail), nil
	}
	return class, "", nil
}

// readQueueIfOnlyBaseMoved is the read of the merge queue of step 7 of
// checkMerge. class is t's class on remote with no queue read; when it is
// rebase and a merge queue on remote's default branch would make it ready or
// accepted, it reads that queue once with lookupMergeQueue, in m.dir under
// m.enforcementTimeout, of the repository r names (origin's, as readOrigin
// read it), through m.run, and keeps the read in m.queue for step 11
// (queueReadFor); read reports that it did. With any other class it reads
// nothing. The refusal is for a gh not found (ghMissing), as for any other gh
// call of checkMerge, and the error for ctx done while the queue was read; a
// read that failed otherwise is no refusal here: it is in m.queue, and step 7
// refuses the branch as rebase with why.
func (m *mergeCheck) readQueueIfOnlyBaseMoved(ctx context.Context, r mergeReady, t *mergeTarget, remote mergeRemote,
	class string) (read bool, refused string, err error) {
	if class != MergeClassRebase {
		return false, "", nil
	}
	withQueue := remote
	withQueue.queue = mergeQueueLookup{state: mergeQueuePresent}
	if c, _ := classifyMergeTarget(t, withQueue); c != MergeClassReady && c != MergeClassAccepted {
		return false, "", nil
	}
	def := remote.defaultBranch
	var ghNotFound error
	lookup := lookupMergeQueue(ctx, MergeEnforcementParams{
		Dir: m.dir, Exec: m.runNotingGHNotFound(&ghNotFound), Timeout: m.enforcementTimeout, Host: r.apiHost, Owner: r.repo.owner,
		Name: r.repo.name, DefaultBranch: def, HintStateDir: m.hintStateDir,
	})
	switch {
	case ctx.Err() != nil:
		return false, "", mergeInterrupted(ctx, "the merge queue of origin/"+def+" was read")
	case ghNotFound != nil:
		return false, m.ghMissing(r, ghNotFound), nil
	}
	m.queue, m.queueOf = lookup, def
	return true, "", nil
}

// queueReadFor is the read of the merge queue that step 7 kept in m.queue
// when it read the queue of def, origin's default branch as step 11 checks
// it, and mergeQueueNotRead otherwise, so CheckMergeEnforcement takes a read
// only of the branch it checks and reads the queue itself when step 7 did
// not.
func (m *mergeCheck) queueReadFor(def string) mergeQueueLookup {
	if m.queueOf != def {
		return mergeQueueLookup{}
	}
	return m.queue
}

// runNotingGHNotFound is m.run, which also keeps in *notFound the error of
// the first call that is not found (exec.ErrNotFound). Step 7's read of the
// merge queue and CheckMergeEnforcement run their gh calls through it, so a
// gh not found there is refused as any other gh call of checkMerge is
// (ghMissing).
func (m *mergeCheck) runNotingGHNotFound(notFound *error) MergeOrderExec {
	return func(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
		out, err := m.run(ctx, dir, env, name, args...)
		if errors.Is(err, exec.ErrNotFound) && *notFound == nil {
			*notFound = err
		}
		return out, err
	}
}
