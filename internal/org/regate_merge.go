package org

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// How Regate (regate.go) merges origin's default branch in and pushes: the
// merge commit (regateMerged, regateMergeTip, D3), its undo (regateUndoMerge,
// D5), the push and its retry (regatePush, regateHeadNotOnOrigin, D4), and
// the git they run that ctx does not stop (regateUncancelledGit).

// regateUncancelledGit runs git args in dir as regateRemoteGit does, with
// env, on a context ctx's cancellation does not reach (context.WithoutCancel),
// so only regateRemoteTimeout stops it: the merge, its abort, and its undo
// change the worktree, and one stopped half-way would leave it in neither
// state. Regate looks at ctx after it.
func regateUncancelledGit(ctx context.Context, dir string, env []string, args ...string) (gitResult, bool, error) {
	return regateRemoteGit(context.WithoutCancel(ctx), dir, env, args...)
}

// regateMergeEnv is env, copied, with GIT_MERGE_AUTOEDIT=no after it, so the
// merge opens no editor whatever the caller sets; regateUncancelledGit adds
// regateRemoteEnv's LC_ALL=C, so regateMergeTip reads git's messages
// untranslated (regateNoIdentity).
func regateMergeEnv(env []string) []string {
	return append(slices.Clip(env), "GIT_MERGE_AUTOEDIT=no")
}

// regateHeadNotOnOrigin is Regate's path for a HEAD that is not originHead,
// origin's tip of the branch after the fetch (D4). Regate gates only a head
// origin has, with one exception, the retry of a push that failed: when
// originHead is under HEAD and the branch's latest gate record is a pass of
// HEAD on t's base tip (regateBranch.passedOn), it pushes HEAD and does
// nothing else (regatePush, RegateOutcomePushOnly). Otherwise it refuses with
// regateCheckOriginHead's reason and regateResetHint's.
func regateHeadNotOnOrigin(ctx context.Context, b regateBranch, t gateTarget, originHead string, env []string, w gateWording,
	res RegateResult) (RegateResult, error) {
	if b.passedOn(t) {
		under, err := regateIsAncestor(ctx, t.root, originHead, t.head)
		if err != nil {
			return RegateResult{}, regateInterrupted(ctx, err, "")
		}
		if under {
			res.Outcome, res.Passed, res.LogPath = RegateOutcomePushOnly, true, b.gate.record.Log
			return regatePush(ctx, t, env, res)
		}
	}
	refused, err := regateCheckOriginHead(ctx, t, originHead, w)
	if err != nil {
		return RegateResult{}, regateInterrupted(ctx, err, "")
	}
	hint, err := regateResetHint(ctx, b, t, originHead, w)
	if err != nil {
		return RegateResult{}, regateInterrupted(ctx, err, "")
	}
	return RegateResult{Refused: refused + hint}, nil
}

// regateResetHint is what the refusal of a HEAD that is not originHead adds
// when HEAD is a merge commit an earlier Regate made and could not push (D4):
// b's latest gate record is readable with no corrupt line after it, is one
// Regate wrote (via=regate) of HEAD, and HEAD's first parent is originHead.
// The hint is the git reset --keep that drops that commit, and running Regate
// again, which merges anew. For any other HEAD it is "": the reset could drop
// commits nobody pushed, a leader's gated head among them.
func regateResetHint(ctx context.Context, b regateBranch, t gateTarget, originHead string, w gateWording) (string, error) {
	g := b.gate
	if g == nil || !g.readable || b.corruptAfterGate || g.record.Head != t.head || g.record.Via != gateViaRegate {
		return "", nil
	}
	parents, err := regateParents(ctx, t.root, t.head)
	if err != nil || len(parents) == 0 || parents[0] != originHead {
		return "", err
	}
	return fmt.Sprintf("; or, since HEAD is the merge commit an earlier regate made on origin's tip of the branch and did not "+
		"push, drop it with git -C %s reset --keep refs/remotes/origin/%s and %s, which merges origin's %s anew",
		t.root, t.branch, w.again, t.base.branch), nil
}

// regateParents returns commit's parents in root's repository, the first
// parent first (`git rev-list --parents -n 1`): none for a root commit.
func regateParents(ctx context.Context, root, commit string) ([]string, error) {
	out, err := gateGitOutput(ctx, regatePrefix, root, "rev-list", "--parents", "-n", "1", commit)
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return nil, fmt.Errorf("org: regate: git rev-list --parents -n 1 %s in %s printed nothing", commit, root)
	}
	return fields[1:], nil
}

// regateMerged is Regate's path for a HEAD that does not contain origin's
// default branch tip (AC4): it merges the tip in (regateMergeTip, D3), runs
// and records the gate on the merge commit (regateGate), and pushes a
// recorded gate_passed (regatePush). Until a gate_passed is recorded, the
// merge commit is undone (regateUndoMerge, D5): when Regate is interrupted
// before the scripts, when nothing is recorded (the record is refused, or
// Regate is interrupted or fails before the append), and after a recorded
// static_failed or test_failed. After a recorded head_moved or tree_changed
// it is kept, and res.NotUndone says how to undo it (after tree_changed, also
// how to set the scripts' changes aside first). An error or refusal after
// the merge says what became of the merge commit (regateUndoneText).
func (o *Org) regateMerged(ctx context.Context, b regateBranch, t gateTarget, env []string, w gateWording, res RegateResult) (RegateResult, error) {
	merge, refused, err := regateMergeTip(ctx, t, env, w)
	if err != nil {
		// The merge ran on a context the interrupt does not reach, so err
		// is not ctx's: it says how the worktree is.
		return RegateResult{}, err
	}
	if refused != "" {
		return RegateResult{Refused: refused}, nil
	}
	res.Head, res.Merge, res.PreMergeHead = merge, merge, t.head
	t.head = merge
	if ctx.Err() != nil {
		res = regateUndoMerge(ctx, t, env, res)
		return res, fmt.Errorf("%w; %s", regateInterrupted(ctx, ctx.Err(), ""), regateUndoneText(res))
	}

	res, err = o.regateGate(ctx, b, t, env, res)
	switch {
	case err != nil:
		res = regateUndoMerge(ctx, t, env, res)
		return res, fmt.Errorf("%w; %s", err, regateUndoneText(res))
	case res.Refused != "":
		res = regateUndoMerge(ctx, t, env, res)
		res.Refused += "; " + regateUndoneText(res)
		return res, nil
	}
	res.Outcome = RegateOutcomeMerged
	switch res.Reason {
	case "":
		return regatePush(ctx, t, env, res)
	case GateReasonStaticFailed, GateReasonTestFailed:
		return regateUndoMerge(ctx, t, env, res), nil
	case GateReasonHeadMoved:
		why := "HEAD moved off it while the scripts ran (head_moved), and git reset --keep would drop what came after it"
		if head, err := gateHead(context.WithoutCancel(ctx), regatePrefix, t.root); err == nil {
			why = "HEAD moved off it to " + head + " while the scripts ran (head_moved), and git reset --keep would drop what came after it"
		}
		res.NotUndone = regateMergeLeft(t.root, res, why, "")
	default:
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("the scripts left %d uncommitted change(s) or untracked file(s) "+
			"in the worktree (tree_changed), which git reset --keep stops on or carries along", len(res.Dirty)),
			fmt.Sprintf("; if git reset --keep stops on those changes, set them aside first (git -C %s stash -u, or move the files "+
				"out of the worktree)", t.root))
	}
	return res, nil
}

// regateMergeTip merges t's base tip into the branch checked out in t.root
// with D3's merge commit, `git merge --no-ff --no-edit -m 'Merge <origin/name>
// <12 hex> into <branch>' <sha>`, run with env, so the commit has the
// caller's identity and the caller's hooks run, and GIT_MERGE_AUTOEDIT=no
// (regateMergeEnv), on a context ctx's cancellation does not reach
// (regateUncancelledGit). It returns the merge commit once it has checked
// that the commit's parents are HEAD and the tip. When git makes no merge
// commit (a conflict, a pre-merge-commit hook that fails, no user.name or
// user.email, a merge past regateRemoteTimeout), it undoes a merge left in
// progress with git merge --abort, checks that HEAD and the tree are as they
// were, and returns the refusal, which names the conflicting files for a
// conflict (AC5). The error is for a git that cannot run, a worktree it
// could not put back as it was, and a merge commit with other parents: it
// says how the worktree is and what restores it. Nothing is recorded then.
func regateMergeTip(ctx context.Context, t gateTarget, env []string, w gateWording) (merge, refused string, err error) {
	uncancelled := context.WithoutCancel(ctx)
	msg := fmt.Sprintf("Merge %s %s into %s", strings.TrimPrefix(t.base.ref, "refs/remotes/"), t.base.sha[:12], t.branch)
	command := fmt.Sprintf("git merge --no-ff --no-edit -m '%s' %s", msg, t.base.sha)
	r, timedOut, err := regateUncancelledGit(ctx, t.root, regateMergeEnv(env), "merge", "--no-ff", "--no-edit", "-m", msg, t.base.sha)
	if err != nil {
		return "", "", err
	}
	if !timedOut && r.code == 0 {
		head, err := gateHead(uncancelled, w.prefix, t.root)
		if err != nil {
			return "", "", err
		}
		parents, err := regateParents(uncancelled, t.root, head)
		if err != nil {
			return "", "", err
		}
		if !slices.Equal(parents, []string{t.head, t.base.sha}) {
			return "", "", fmt.Errorf("org: regate: %s in %s left HEAD at %s, whose parents are %q, not HEAD %s and origin's tip %s "+
				"of %s (a hook may have made another commit), so regate ran nothing and recorded nothing: check git -C %s log, "+
				"and to undo the merge run git -C %s reset --keep %s", command, t.root, head, parents, t.head, t.base.sha,
				t.base.branch, t.root, t.root, t.head)
		}
		return head, "", nil
	}

	// The conflicting files are read before the abort clears them; a failure
	// to read them only leaves them out of the refusal, and the checks after
	// the abort read the worktree again.
	files, _ := gateGitOutput(uncancelled, w.prefix, t.root, "diff", "--name-only", "--diff-filter=U")
	merging, err := gateGitPathExists(uncancelled, w.prefix, t.root, "a merge", "MERGE_HEAD")
	if err != nil {
		return "", "", regateMergeNotBack(t, command, fmt.Sprintf("whether a merge is in progress cannot be read (%v)", err))
	}
	aborted := ""
	if merging {
		ar, abortTimedOut, err := regateUncancelledGit(ctx, t.root, env, "merge", "--abort")
		switch {
		case err != nil:
			return "", "", regateMergeNotBack(t, command, fmt.Sprintf("a merge is in progress, and git merge --abort could not run (%v)", err))
		case abortTimedOut || ar.code != 0:
			return "", "", regateMergeNotBack(t, command, "a merge is in progress, and git merge --abort failed"+regateStderrTail(ar.stderr))
		}
		aborted = " (git merge --abort undid the merge it left in progress)"
	}
	head, err := gateHead(uncancelled, w.prefix, t.root)
	if err != nil {
		return "", "", regateMergeNotBack(t, command, fmt.Sprintf("HEAD cannot be read (%v)", err))
	}
	dirty, err := gateDirtyLines(uncancelled, w.prefix, t.root)
	if err != nil {
		return "", "", regateMergeNotBack(t, command, fmt.Sprintf("the worktree cannot be read (%v)", err))
	}
	if head != t.head || len(dirty) > 0 {
		return "", "", regateMergeNotBack(t, command, fmt.Sprintf("HEAD is %s (it was %s), and the worktree has %d uncommitted "+
			"change(s) or untracked file(s)", head, t.head, len(dirty)))
	}

	said := r.stderr
	if said == "" {
		said = r.stdout
	}
	switch conflicted := strings.Fields(files); {
	case timedOut:
		return "", w.prefix + fmt.Sprintf("%s in %s ran past %s and was stopped%s, so nothing was recorded: check the hooks a "+
			"merge runs there (pre-merge-commit, prepare-commit-msg, commit-msg) and %s%s", command, t.root, regateRemoteTimeout,
			aborted, w.again, regateStderrTail(said)), nil
	case len(conflicted) > 0:
		return "", w.prefix + fmt.Sprintf("merging origin's tip %s of %s into %s in %s conflicts in %s, and regate does not "+
			"resolve conflicts%s, so nothing was recorded: merge it yourself (git -C %s merge %s), resolve the conflicts, "+
			"commit, push, and %s", t.base.sha, t.base.branch, t.branch, t.root, strings.Join(conflicted, ", "), aborted, t.root,
			t.base.ref, w.again), nil
	case strings.Contains(r.stderr, regateNoIdentity):
		return "", w.prefix + fmt.Sprintf("%s in %s could not make the merge commit, since git has no user.name and user.email "+
			"to make it with%s, so nothing was recorded: set them (git -C %s config user.name <name> and git -C %s config "+
			"user.email <email>, or in your global git config) and %s%s", command, t.root, aborted, t.root, t.root, w.again,
			regateStderrTail(said)), nil
	}
	return "", w.prefix + fmt.Sprintf("%s in %s made no merge commit (exit %d)%s, so nothing was recorded: a hook such as "+
		"pre-merge-commit may have stopped it; fix what git said and %s%s", command, t.root, r.code, aborted, w.again,
		regateStderrTail(said)), nil
}

// regateMergeNotBack is regateMergeTip's error for a merge that made no merge
// commit and left the worktree in state, not as it was: nothing was
// recorded, and what puts the worktree back.
func regateMergeNotBack(t gateTarget, command, state string) error {
	return fmt.Errorf("org: regate: %s in %s made no merge commit, and the worktree is not as it was before: %s; nothing was "+
		"recorded: check it with git -C %s status, undo a merge in progress with git -C %s merge --abort, put HEAD back with "+
		"git -C %s reset --keep %s, and run ralph org regate --branch %s again", command, t.root, state, t.root, t.root, t.root,
		t.head, t.branch)
}

// regateUndoMerge undoes res.Merge, the merge commit Regate made on
// res.PreMergeHead in t.root (D5), with git reset --keep res.PreMergeHead,
// and sets res.Undone. It undoes only a HEAD that is still res.Merge in a
// clean tree: a commit made after it would go with it, and reset --keep stops
// on a dirty tree's changes or carries them along. Its git commands run on a
// context ctx's cancellation does not reach, so an interrupt does not stop
// the undo half-way (regateUncancelledGit). When it does not undo, it sets
// res.NotUndone (regateMergeLeft).
func regateUndoMerge(ctx context.Context, t gateTarget, env []string, res RegateResult) RegateResult {
	uncancelled := context.WithoutCancel(ctx)
	head, err := gateHead(uncancelled, regatePrefix, t.root)
	if err != nil {
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("HEAD cannot be read (%v)", err), "")
		return res
	}
	if head != res.Merge {
		res.NotUndone = regateMergeLeft(t.root, res, "HEAD is now "+head+", not the merge commit", "")
		return res
	}
	dirty, err := gateDirtyLines(uncancelled, regatePrefix, t.root)
	switch {
	case err != nil:
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("the worktree cannot be read (%v)", err), "")
		return res
	case len(dirty) > 0:
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("the worktree has %d uncommitted change(s) or untracked file(s)", len(dirty)), "")
		return res
	}
	r, timedOut, err := regateUncancelledGit(ctx, t.root, env, "reset", "--keep", res.PreMergeHead)
	switch {
	case err != nil:
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("git reset --keep %s could not run (%v)", res.PreMergeHead, err), "")
	case timedOut:
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("git reset --keep %s ran past %s and was stopped", res.PreMergeHead,
			regateRemoteTimeout), regateStderrTail(r.stderr))
	case r.code != 0:
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("git reset --keep %s failed (exit %d)", res.PreMergeHead, r.code),
			regateStderrTail(r.stderr))
	default:
		res.Undone = true
	}
	return res
}

// regateMergeLeft is res.NotUndone for res.Merge kept because of why: how to
// look at the worktree and the git reset --keep that undoes the merge, then
// tail (regateStderrTail).
func regateMergeLeft(root string, res RegateResult, why, tail string) string {
	return fmt.Sprintf("the merge commit %s was not undone, since %s: check the worktree (git -C %s status, git -C %s log), "+
		"and to undo the merge run git -C %s reset --keep %s, which also drops every commit made after it%s",
		res.Merge, why, root, root, root, res.PreMergeHead, tail)
}

// regateUndoneText says what became of res.Merge when Regate was to undo it
// (regateUndoMerge): undone, or res.NotUndone.
func regateUndoneText(res RegateResult) string {
	if res.Undone {
		return fmt.Sprintf("the merge commit %s was undone (git reset --keep %s), so only the reflog has it now", res.Merge,
			res.PreMergeHead)
	}
	return res.NotUndone
}

// regatePush pushes t's HEAD, the head of a gate_passed, to origin's branch
// without force (`git push origin <sha>:refs/heads/<branch>`, regateRemoteGit,
// D8), sets res.Pushed, and then points refs/remotes/origin/<branch> at it
// when the push has not (regateSetTrackingRef). A push that fails or runs past
// regateRemoteTimeout sets res.PushFailed. The record and HEAD stay as they
// are either way (D5): running Regate again finds the push landed (AC3) or
// pushes only (D4). The error is ctx done before or during the push
// (regatePushInterrupted), a git that cannot run, or a remote-tracking ref
// that cannot be updated after the push.
func regatePush(ctx context.Context, t gateTarget, env []string, res RegateResult) (RegateResult, error) {
	if ctx.Err() != nil {
		return res, regatePushInterrupted(ctx, t, false)
	}
	spec := t.head + ":refs/heads/" + t.branch
	command := "git push origin " + spec
	kept := fmt.Sprintf("the gate_passed record of %s and HEAD are kept: check origin (git remote get-url origin), its "+
		"credentials, and whether someone pushed to %s after the fetch, and run ralph org regate --branch %s again, which "+
		"pushes only, or says what changed", t.head, t.branch, t.branch)
	r, timedOut, err := regateRemoteGit(ctx, t.root, env, "push", "origin", spec)
	switch {
	case err != nil && ctx.Err() != nil:
		return res, regatePushInterrupted(ctx, t, true)
	case err != nil:
		return res, fmt.Errorf("%w; %s", err, kept)
	case timedOut:
		res.PushFailed = fmt.Sprintf("org: regate: %s in %s ran past %s and was stopped, so origin may not have it: %s%s",
			command, t.root, regateRemoteTimeout, kept, regateStderrTail(r.stderr))
		return res, nil
	case r.code != 0:
		res.PushFailed = fmt.Sprintf("org: regate: %s in %s failed (exit %d), so origin's %s is not %s: %s%s",
			command, t.root, r.code, t.branch, t.head, kept, regateStderrTail(r.stderr))
		return res, nil
	}
	res.Pushed = true
	return res, regateSetTrackingRef(context.WithoutCancel(ctx), t)
}

// regatePushInterrupted is regatePush's error for ctx done before the push or
// while it ran (during): the record and HEAD are kept, and running Regate
// again pushes only, or finds the push landed.
func regatePushInterrupted(ctx context.Context, t gateTarget, during bool) error {
	when := fmt.Sprintf("before %s was pushed to origin's %s, so nothing was pushed", t.head, t.branch)
	if during {
		when = fmt.Sprintf("while %s was pushed to origin's %s, so origin may or may not have it", t.head, t.branch)
	}
	return fmt.Errorf("org: regate: interrupted (%v) %s: the gate_passed record of it and HEAD are kept; run ralph org regate "+
		"--branch %s again, which pushes it, or finds it pushed", context.Cause(ctx), when, t.branch)
}

// regateSetTrackingRef points refs/remotes/origin/<branch> at t's HEAD, which
// Regate pushed to origin's branch, when the push has not: git updates it
// only when remote.origin.fetch maps the branch there, which Regate does not
// count on. It updates the ref from the value it read (`git update-ref`
// with the old value), so a fetch in between is not overwritten.
func regateSetTrackingRef(ctx context.Context, t gateTarget) error {
	ref := "refs/remotes/origin/" + t.branch
	r, err := gateGit(ctx, regatePrefix, t.root, "rev-parse", "--verify", "--quiet", ref)
	if err != nil {
		return fmt.Errorf("%w; %s was pushed to origin's %s", err, t.head, t.branch)
	}
	old := strings.TrimSpace(r.stdout)
	if old == t.head {
		return nil
	}
	if _, err := gateGitOutput(ctx, regatePrefix, t.root, "update-ref", ref, t.head, old); err != nil {
		return fmt.Errorf("%w; %s was pushed to origin's %s, and git fetch origin %s brings %s up to date", err, t.head, t.branch,
			t.branch, ref)
	}
	return nil
}
