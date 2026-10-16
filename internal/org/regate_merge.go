package org

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// How Regate (regate.go) merges origin's default branch in and pushes: the
// merge commit (regateMerged, regateMergeTip, D3), the check of what is on
// disk in the merge's way (regateCheckInTheWay, regateMergeWrites), the merge
// commit's undo (regateUndoMerge, D5), the push and its retry (regatePush,
// regateHeadNotOnOrigin, D4), and the git they run that ctx does not stop
// (regateUncancelledGit).

// regateUncancelledGit runs git args in dir with env (regateBoundedGit) in a
// process group of its own, on a context ctx's cancellation does not reach
// (context.WithoutCancel): neither an interrupt nor the Ctrl-C or the hangup
// a terminal sends its foreground group stops it, only regateGitTimeout. The
// merge, its abort, and its undo change the worktree, and one stopped
// half-way would leave it in neither state. Regate looks at ctx after it.
// When ctx is done while git runs, one line on stderr says that Regate waits
// for git, so that a human who pressed Ctrl-C does not kill ralph or git
// half-way. A git that starts once ctx is done, the abort or the undo after
// an interrupt, adds no line: the merge said it, or the interrupt stopped a
// script, which ends at once.
func regateUncancelledGit(ctx context.Context, dir string, env []string, stderr io.Writer, args ...string) (gitResult, bool, error) {
	if ctx.Err() != nil {
		return regateBoundedGit(context.WithoutCancel(ctx), dir, env, true, args...)
	}
	said := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		defer close(said)
		_, _ = fmt.Fprintf(stderr, "org: regate: interrupted (%v): waiting for git %s in %s to finish (at most %s) before it "+
			"stops, since a git stopped half-way would leave the worktree in neither state\n", context.Cause(ctx),
			strings.Join(args[:min(2, len(args))], " "), dir, regateGitTimeout)
	})
	r, timedOut, err := regateBoundedGit(context.WithoutCancel(ctx), dir, env, true, args...)
	if !stop() {
		<-said
	}
	return r, timedOut, err
}

// regateMergeEnv is env, copied, with GIT_MERGE_AUTOEDIT=no after it, so the
// merge opens no editor whatever the caller sets. Nothing else is added: the
// merge's hooks run in the caller's environment and locale, and
// regateMergeTip reads no message of git's (regateCheckIdentity reads git's
// exit code instead). The git merge-tree before the merge gets the same
// (regateMergeWrites), so that it merges as the merge does.
func regateMergeEnv(env []string) []string {
	return append(slices.Clip(env), "GIT_MERGE_AUTOEDIT=no")
}

// regateHeadNotOnOrigin is Regate's path for a HEAD that is not originHead,
// origin's tip of the branch after the fetch (D4). Regate gates only a head
// origin has, with one exception, the retry of a push that failed: when the
// branch's latest gate record is a pass of HEAD on t's base tip
// (regateBranch.passedOn) and HEAD is a head whose push may have failed, it
// pushes HEAD and does nothing else (regatePush, RegateOutcomePushOnly). For
// a pass Regate recorded (via=regate), HEAD must be its own merge commit on
// originHead (regateIsOwnMerge): a gate-only Regate (AC9) records via=regate
// too, and the head it gated was origin's tip then, so once origin's branch
// was put back under that head (a force push), pushing it again would undo
// what someone did to origin. For a pass with no via (ralph org gate, the
// leader's), any HEAD with originHead under it will do: the leader's push of
// its gated head may have failed. Otherwise it refuses with
// regateCheckOriginHead's reason and regateResetHint's.
func regateHeadNotOnOrigin(ctx context.Context, b regateBranch, t gateTarget, originHead string, env []string, w gateWording,
	res RegateResult) (RegateResult, error) {
	if b.passedOn(t) {
		var retry bool
		var err error
		if b.gate.record.Via == gateViaRegate {
			retry, err = regateIsOwnMerge(ctx, t, originHead, b.gate.record.Base)
		} else {
			retry, err = regateIsAncestor(ctx, t.root, originHead, t.head)
		}
		if err != nil {
			return RegateResult{}, regateInterrupted(ctx, err, "")
		}
		if retry {
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
// Regate wrote (via=regate) of HEAD, and HEAD is Regate's own merge commit on
// originHead (regateIsOwnMerge). The record alone does not say that: a
// gate-only Regate (AC9) records via=regate too, of a head that was origin's
// tip when it was gated. The hint is the git reset --keep that drops that
// commit, and running Regate again, which merges anew. For any other HEAD it
// is "": the reset could drop commits nobody pushed, a leader's gated head
// among them, or a commit origin had until its branch was put back under it.
func regateResetHint(ctx context.Context, b regateBranch, t gateTarget, originHead string, w gateWording) (string, error) {
	g := b.gate
	if g == nil || !g.readable || b.corruptAfterGate || g.record.Head != t.head || g.record.Via != gateViaRegate {
		return "", nil
	}
	own, err := regateIsOwnMerge(ctx, t, originHead, g.record.Base)
	if err != nil || !own {
		return "", err
	}
	return fmt.Sprintf("; or, since HEAD is the merge commit an earlier regate made on origin's tip of the branch and did not "+
		"push, drop it with git -C %s reset --keep refs/remotes/origin/%s and %s, which merges origin's %s anew",
		t.root, t.branch, w.again, t.base.branch), nil
}

// regateIsOwnMerge reports whether t's HEAD has the shape of the merge commit
// Regate makes on originHead, origin's tip of the branch after the fetch:
// exactly two parents, originHead first and base second, base being the
// default branch tip a gate record of HEAD names. regateMergeTip makes its
// merge commit with the parents HEAD before the merge and t.base.sha, which
// that record then names as its base. The commit's message is not read.
func regateIsOwnMerge(ctx context.Context, t gateTarget, originHead, base string) (bool, error) {
	parents, err := regateParents(ctx, t.root, t.head)
	if err != nil {
		return false, err
	}
	return slices.Equal(parents, []string{originHead, base}), nil
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
// default branch tip (AC4): it checks that git has the identity the merge
// commit needs (regateCheckIdentity) and that nothing git does not see is on
// disk where the merge writes (regateCheckInTheWay), merges the tip in
// (regateMergeTip, D3), runs and records the gate on the merge commit
// (regateGate), and pushes a recorded gate_passed (regatePush). Until a
// gate_passed is recorded, the merge commit is undone (regateUndoMerge, D5):
// when Regate is interrupted before the scripts, when nothing is recorded
// (the record is refused, or Regate is interrupted or fails before the
// append), and after a recorded static_failed or test_failed. After a
// recorded head_moved or tree_changed it is kept, and res.NotUndone says how
// to undo it (after tree_changed, also how to set the scripts' changes aside
// first). An error or refusal after the merge says what became of the merge
// commit (regateUndoneText). stderr gets the line regateUncancelledGit
// writes.
func (o *Org) regateMerged(ctx context.Context, b regateBranch, t gateTarget, env []string, stderr io.Writer, w gateWording,
	res RegateResult) (RegateResult, error) {
	refused, err := regateCheckIdentity(ctx, t, env, w)
	if err != nil {
		return RegateResult{}, regateInterrupted(ctx, err, "")
	}
	if refused != "" {
		return RegateResult{Refused: refused}, nil
	}
	if refused, err = regateCheckInTheWay(ctx, t, env, w); err != nil {
		return RegateResult{}, regateInterrupted(ctx, err, "")
	}
	if refused != "" {
		return RegateResult{Refused: refused}, nil
	}
	merge, refused, err := regateMergeTip(ctx, t, env, stderr, w)
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
		res = regateUndoMerge(ctx, t, env, stderr, res)
		return res, fmt.Errorf("%w; %s", regateInterrupted(ctx, ctx.Err(), ""), regateUndoneText(res))
	}

	res, err = o.regateGate(ctx, b, t, env, res)
	switch {
	case err != nil:
		res = regateUndoMerge(ctx, t, env, stderr, res)
		return res, fmt.Errorf("%w; %s", err, regateUndoneText(res))
	case res.Refused != "":
		res = regateUndoMerge(ctx, t, env, stderr, res)
		res.Refused += "; " + regateUndoneText(res)
		return res, nil
	}
	res.Outcome = RegateOutcomeMerged
	switch res.Reason {
	case "":
		return regatePush(ctx, t, env, res)
	case GateReasonStaticFailed, GateReasonTestFailed:
		return regateUndoMerge(ctx, t, env, stderr, res), nil
	case GateReasonHeadMoved:
		why := "HEAD moved off it while the scripts ran (head_moved), and git reset --keep would drop what came after it"
		if head, err := gateHead(context.WithoutCancel(ctx), regatePrefix, t.root); err == nil {
			why = "HEAD moved off it to " + head + " while the scripts ran (head_moved), and git reset --keep would drop what came after it"
		}
		res.NotUndone = regateMergeLeft(t.root, res, why, "")
	case GateReasonTreeChanged:
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("the scripts left %d uncommitted change(s) or untracked file(s) "+
			"in the worktree (tree_changed), which git reset --keep stops on or carries along", len(res.Dirty)),
			fmt.Sprintf("; if git reset --keep stops on those changes, set them aside first (git -C %s stash -u, or move the files "+
				"out of the worktree)", t.root))
	default:
		// No reason regateGate records leads here; one added later keeps the
		// merge commit until this switch says what to do after it.
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("the gate failed with the reason %s, after which regate does not "+
			"undo it", res.Reason), "")
	}
	return res, nil
}

// regateCheckIdentity refuses t when git, run with env in t.root, has no
// author or no committer to make D3's merge commit with: `git var
// GIT_AUTHOR_IDENT` or `git var GIT_COMMITTER_IDENT` fails (no user.name or
// user.email, and none git may make up, as with user.useConfigOnly). It
// reads git's exit code, not its messages, so the merge needs no LC_ALL=C
// (regateMergeEnv). The error is for a git that cannot run, runs past
// regateGitTimeout, or is ended by a signal, and for ctx done; nothing is
// merged or recorded then either.
func regateCheckIdentity(ctx context.Context, t gateTarget, env []string, w gateWording) (string, error) {
	for _, ident := range []string{"GIT_AUTHOR_IDENT", "GIT_COMMITTER_IDENT"} {
		// In a group of its own, git var gets a terminal's Ctrl-C only via
		// ctx, so the error is ctx's, not a race with regateGitSignaled.
		r, timedOut, err := regateBoundedGit(ctx, t.root, env, true, "var", ident)
		switch {
		case err != nil:
			return "", err
		case timedOut:
			return "", fmt.Errorf("org: regate: git var %s in %s ran past %s and was stopped, so nothing was merged or recorded",
				ident, t.root, regateGitTimeout)
		case r.code != 0:
			return w.prefix + fmt.Sprintf("git in %s has no user.name and user.email to make the merge commit of origin's tip %s "+
				"of %s with (git var %s failed), so regate merged nothing and recorded nothing: set them (git -C %s config "+
				"user.name <name> and git -C %s config user.email <email>, or in your global git config) and %s%s", t.root,
				t.base.sha, t.base.branch, ident, t.root, t.root, w.again, regateStderrTail(r.stderr)), nil
		}
	}
	return "", nil
}

// regateInTheWayListed is how many of the paths in the way of the merge
// regateCheckInTheWay's refusal names before it says how many more there are.
const regateInTheWayListed = 10

// regateCheckInTheWay refuses t when D3's merge of t's base tip would write
// over something on disk in t.root that git does not see: without a word,
// git merge overwrites an ignored file at a path it writes and removes one in
// a directory it replaces with a file (--no-overwrite-ignore guards a
// fast-forward only), and the clean-tree check (gateDirtyLines) lists no
// ignored file. The paths the merge writes that HEAD does not track are
// regateMergeWrites's; one of them is in the way when regatePathInTheWay
// finds something untracked there. The refusal names what is on disk, as it
// is spelled there, the first regateInTheWayListed of them. With no merge base
// it refuses nothing: git merge refuses unrelated histories itself
// (regateMergeTip). The error is for a git that fails, runs past
// regateGitTimeout, or is ended by a signal, ctx done, a path lstat cannot
// read, and a directory that cannot be listed; nothing is merged or recorded
// then either.
func regateCheckInTheWay(ctx context.Context, t gateTarget, env []string, w gateWording) (string, error) {
	r, err := gateGit(ctx, regatePrefix, t.root, "merge-base", t.head, t.base.sha)
	switch {
	case err != nil:
		return "", err
	case r.code == 1 && strings.TrimSpace(r.stdout) == "":
		return "", nil
	case r.code != 0:
		return "", fmt.Errorf("%sgit merge-base %s %s in %s: exit %d: %s", regatePrefix, t.head, t.base.sha, t.root, r.code, r.stderr)
	}
	writes, err := regateMergeWrites(ctx, t, env, strings.TrimSpace(r.stdout))
	if err != nil {
		return "", err
	}
	var inTheWay []string
	seen := make(map[string]bool)
	for _, p := range writes {
		found, err := regatePathInTheWay(ctx, t.root, p)
		if err != nil {
			return "", err
		}
		if found != "" && !seen[found] {
			seen[found] = true
			inTheWay = append(inTheWay, found)
		}
	}
	if len(inTheWay) == 0 {
		return "", nil
	}
	listed := strings.Join(inTheWay[:min(len(inTheWay), regateInTheWayListed)], ", ")
	if more := len(inTheWay) - regateInTheWayListed; more > 0 {
		listed += fmt.Sprintf(", and %d more", more)
	}
	return w.prefix + fmt.Sprintf("merging origin's tip %s of %s into %s in %s would write over what is on disk at %s, which "+
		"HEAD does not track: untracked files, or directories with untracked files in them, which the clean-tree check passes "+
		"over as ignored and git merge overwrites without a word (a file where the merge needs a directory too), so regate "+
		"merged nothing and recorded nothing: move them out of the worktree and %s",
		t.base.sha, t.base.branch, t.branch, t.root, listed, w.again), nil
}

// regateMergeWrites returns the paths D3's merge of t's base tip into HEAD
// writes in t.root that HEAD does not have, mergeBase being HEAD's and the
// tip's merge base. They are what the tree of `git merge-tree --write-tree
// --no-messages HEAD <tip>` adds to HEAD, `git diff --name-only -z
// --no-renames --diff-filter=A HEAD <tree>`; merge-tree prints the tree on
// its first line both for a clean merge (exit 0) and for one that conflicts
// (exit 1). merge-tree runs the merge git merge runs: with env as the merge
// gets it (regateMergeEnv) and no setting changed, so it reads the merge's
// config and attributes, and with HEAD and the tip named as git merge names
// them. Its tree then has what a directory rename moves into the renamed
// directory, and the file of a file/directory conflict under the name git
// merge gives it (<path>~HEAD, <path>~<tip>). It runs in a process group of
// its own, as git var does (regateCheckIdentity). A git whose merge-tree has
// no --write-tree (before git 2.38: `git merge-tree -h` does not name it, the
// way scripts/secret-scan-branch.sh tells) falls back to the paths the tip's
// side changed since mergeBase, `git diff --name-only -z --no-renames
// <mergeBase> <tip>`, that HEAD does not have, `git diff --name-only -z
// --no-renames --diff-filter=A HEAD <tip>`, which misses a directory rename's
// moves. Either way, a path HEAD deleted that the tip did not touch stays
// deleted and is not among them. Any other exit of merge-tree is an error.
func regateMergeWrites(ctx context.Context, t gateTarget, env []string, mergeBase string) ([]string, error) {
	nul := func(c rune) bool { return c == 0 }
	args := []string{"merge-tree", "--write-tree", "--no-messages", "HEAD", t.base.sha}
	r, timedOut, err := regateBoundedGit(ctx, t.root, regateMergeEnv(env), true, args...)
	switch {
	case err != nil:
		return nil, err
	case timedOut:
		return nil, fmt.Errorf("%sgit %s in %s ran past %s and was stopped, so nothing was merged or recorded", regatePrefix,
			strings.Join(args, " "), t.root, regateGitTimeout)
	case r.code == 0, r.code == 1:
		tree, _, _ := strings.Cut(r.stdout, "\n")
		if tree == "" {
			return nil, fmt.Errorf("%sgit %s in %s: exit %d and no tree on its first line", regatePrefix, strings.Join(args, " "),
				t.root, r.code)
		}
		added, err := gateGitOutput(ctx, regatePrefix, t.root, "diff", "--name-only", "-z", "--no-renames", "--diff-filter=A", "HEAD",
			tree)
		return strings.FieldsFunc(added, nul), err
	}
	help, err := gateGit(ctx, regatePrefix, t.root, "merge-tree", "-h")
	if err != nil {
		return nil, err
	}
	if strings.Contains(help.stdout+help.stderr, "--write-tree") {
		return nil, fmt.Errorf("%sgit %s in %s: exit %d: %s", regatePrefix, strings.Join(args, " "), t.root, r.code, r.stderr)
	}
	changed, err := gateGitOutput(ctx, regatePrefix, t.root, "diff", "--name-only", "-z", "--no-renames", mergeBase, t.base.sha)
	if err != nil {
		return nil, err
	}
	added, err := gateGitOutput(ctx, regatePrefix, t.root, "diff", "--name-only", "-z", "--no-renames", "--diff-filter=A", t.head,
		t.base.sha)
	if err != nil {
		return nil, err
	}
	onTipSide := make(map[string]bool)
	for _, p := range strings.FieldsFunc(changed, nul) {
		onTipSide[p] = true
	}
	var writes []string
	for _, p := range strings.FieldsFunc(added, nul) {
		if onTipSide[p] {
			writes = append(writes, p)
		}
	}
	return writes, nil
}

// regatePathInTheWay is what is on disk in root in the way of the merge
// writing p, a path HEAD does not track, relative to root as git names it,
// with slashes; "" for nothing. On a file system that does not tell case
// apart (macOS's default), lstat finds at p, or at a directory above it, what
// is spelled otherwise on disk: lstat of README.md finds readme.md. So what
// lstat finds is judged under its spelling on disk (regateSpellingOnDisk), and
// that is the path returned. It is a non-directory at p (a file, a symlink),
// unless it is spelled otherwise and HEAD tracks it under that spelling (it is
// in the index, which is HEAD's in the clean tree Regate checked): its content
// is then in git and the merge replaces it, as when origin's default branch
// renames readme.md to README.md. It is a directory at p with something
// untracked in it (`git ls-files --others --directory --no-empty-directory`),
// whose files git merge removes when it writes a file there; an empty
// directory, or one whose files HEAD tracks and the merge deletes, is not in
// the way. It is the first directory above p that lstat finds as a
// non-directory, unless HEAD tracks it: the merge then replaces it with the
// directory. In that clean tree, whatever untracked it finds is ignored. Git
// runs only for a directory at p, a non-directory above it, and a
// non-directory at p spelled otherwise, with --literal-pathspecs. The error is
// for a path lstat cannot read, a directory that cannot be listed, a git that
// fails, and ctx done.
func regatePathInTheWay(ctx context.Context, root, p string) (string, error) {
	parts := strings.Split(p, "/")
	for i := range len(parts) {
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(strings.Join(parts[:i+1], "/"))))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return "", nil
		case err != nil:
			return "", fmt.Errorf("%scheck what is on disk in %s where the merge writes %s: %w", regatePrefix, root, p, err)
		case info.IsDir() && i < len(parts)-1:
			continue
		}
		disk, err := regateSpellingOnDisk(root, parts[:i+1])
		if err != nil {
			return "", fmt.Errorf("%scheck what is on disk in %s where the merge writes %s: %w", regatePrefix, root, p, err)
		}
		switch {
		case info.IsDir():
			untracked, err := gateGitOutput(ctx, regatePrefix, root, "--literal-pathspecs", "ls-files", "-z", "--others",
				"--directory", "--no-empty-directory", "--", disk)
			if err != nil || untracked == "" {
				return "", err
			}
		case i < len(parts)-1 || disk != p:
			tracked, err := gateGitOutput(ctx, regatePrefix, root, "--literal-pathspecs", "ls-files", "-z", "--", disk)
			if err != nil || slices.Contains(strings.Split(tracked, "\x00"), disk) {
				return "", err
			}
		}
		return disk, nil
	}
	return "", nil
}

// regateSpellingOnDisk is how the path of parts (its names from root down),
// which lstat finds in root, is spelled on disk, relative to root with
// slashes: each name is its directory's entry of that name, else the first
// entry (in name order) whose name is the same when case is not told apart
// (strings.EqualFold), so on a file system that tells case apart it is the
// path parts spell. A name with no such entry (one the file system finds
// under another Unicode normalization form, say) stays as it is. The error is
// for a directory that cannot be listed.
func regateSpellingOnDisk(root string, parts []string) (string, error) {
	disk := make([]string, 0, len(parts))
	for _, part := range parts {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(strings.Join(disk, "/"))))
		if err != nil {
			return "", err
		}
		i := slices.IndexFunc(entries, func(e fs.DirEntry) bool { return e.Name() == part })
		if i < 0 {
			i = slices.IndexFunc(entries, func(e fs.DirEntry) bool { return strings.EqualFold(e.Name(), part) })
		}
		if i >= 0 {
			part = entries[i].Name()
		}
		disk = append(disk, part)
	}
	return strings.Join(disk, "/"), nil
}

// regateMergeTip merges t's base tip into the branch checked out in t.root
// with D3's merge commit, `git merge --no-ff --no-edit -m 'Merge <origin/name>
// <12 hex> into <branch>' <sha>`, run with env, so the commit has the
// caller's identity and the caller's hooks run, and GIT_MERGE_AUTOEDIT=no
// (regateMergeEnv), in a process group of its own on a context ctx's
// cancellation does not reach (regateUncancelledGit). It returns the merge
// commit once it has checked that the commit's parents are HEAD and the tip.
// When git makes no merge commit (a conflict, a pre-merge-commit hook that
// fails, a merge past regateGitTimeout or ended by a signal), it undoes a
// merge left in progress and checks that HEAD and the tree are as they were
// (regateAbortLeftMerge), and returns the refusal, which names the
// conflicting files for a conflict (AC5). The error is for a git that cannot
// run; for a worktree it could not put back as it was (a merge stopped in a
// hook leaves its result staged with no merge in progress), which says how
// the worktree is and what restores it (regateMergeNotBack); and for a merge
// git says it made whose HEAD or parents cannot be read after it, or whose
// parents are others, which says that the merge is still there and the git
// reset --keep that undoes it (regateMergeUnchecked). Nothing is recorded
// then. stderr gets the line regateUncancelledGit writes.
func regateMergeTip(ctx context.Context, t gateTarget, env []string, stderr io.Writer, w gateWording) (merge, refused string, err error) {
	uncancelled := context.WithoutCancel(ctx)
	msg := fmt.Sprintf("Merge %s %s into %s", strings.TrimPrefix(t.base.ref, "refs/remotes/"), t.base.sha[:12], t.branch)
	command := fmt.Sprintf("git merge --no-ff --no-edit -m '%s' %s", msg, t.base.sha)
	r, timedOut, err := regateUncancelledGit(ctx, t.root, regateMergeEnv(env), stderr, "merge", "--no-ff", "--no-edit", "-m", msg,
		t.base.sha)
	var signaled regateGitSignaled
	ended := errors.As(err, &signaled)
	if err != nil && !ended {
		return "", "", err
	}
	if !timedOut && !ended && r.code == 0 {
		head, err := gateHead(uncancelled, w.prefix, t.root)
		if err != nil {
			return "", "", regateMergeUnchecked(t, command, fmt.Sprintf("made a merge commit, but HEAD cannot be read after it (%v)", err))
		}
		parents, err := regateParents(uncancelled, t.root, head)
		if err != nil {
			return "", "", regateMergeUnchecked(t, command, fmt.Sprintf("made a merge commit, but the parents of HEAD %s cannot be "+
				"read (%v)", head, err))
		}
		if !slices.Equal(parents, []string{t.head, t.base.sha}) {
			return "", "", regateMergeUnchecked(t, command, fmt.Sprintf("left HEAD at %s, whose parents are %q, not HEAD %s and "+
				"origin's tip %s of %s (a hook may have made another commit)", head, parents, t.head, t.base.sha, t.base.branch))
		}
		return head, "", nil
	}

	outcome := "made no merge commit"
	switch {
	case timedOut:
		outcome = fmt.Sprintf("ran past %s and was stopped", regateGitTimeout)
	case ended:
		outcome = signaled.Error()
	}
	files, aborted, err := regateAbortLeftMerge(ctx, t, env, stderr, w, command, outcome)
	if err != nil {
		return "", "", err
	}

	said := r.stderr
	if said == "" {
		said = r.stdout
	}
	switch conflicted := strings.FieldsFunc(files, func(c rune) bool { return c == 0 }); {
	case timedOut:
		return "", w.prefix + fmt.Sprintf("%s in %s ran past %s and was stopped%s, so nothing was recorded: check the hooks a "+
			"merge runs there (pre-merge-commit, prepare-commit-msg, commit-msg) and %s%s", command, t.root, regateGitTimeout,
			aborted, w.again, regateStderrTail(said)), nil
	case ended:
		return "", w.prefix + fmt.Sprintf("%s in %s %s before it made a merge commit%s, so nothing was recorded: %s%s", command,
			t.root, signaled, aborted, w.again, regateStderrTail(said)), nil
	case len(conflicted) > 0:
		return "", w.prefix + fmt.Sprintf("merging origin's tip %s of %s into %s in %s conflicts in %s, and regate does not "+
			"resolve conflicts%s, so nothing was recorded: merge it yourself (git -C %s merge %s), resolve the conflicts, "+
			"commit, push, and %s", t.base.sha, t.base.branch, t.branch, t.root, strings.Join(conflicted, ", "), aborted, t.root,
			t.base.ref, w.again), nil
	}
	return "", w.prefix + fmt.Sprintf("%s in %s made no merge commit (exit %d)%s, so nothing was recorded: a hook such as "+
		"pre-merge-commit may have stopped it; fix what git said and %s%s", command, t.root, r.code, aborted, w.again,
		regateStderrTail(said)), nil
}

// regateMergeUnchecked is regateMergeTip's error for a merge, command, that
// git says it made (exit 0) but whose commit regateMergeTip cannot take as
// D3's, for why: HEAD or its parents cannot be read after it, or the parents
// are not HEAD and the tip. Regate then runs no script, records nothing, and
// leaves the merge as it is: the error says to check git log and names the
// git reset --keep to t.head, the HEAD before the merge, that undoes it.
func regateMergeUnchecked(t gateTarget, command, why string) error {
	return fmt.Errorf("org: regate: %s in %s %s, so regate ran nothing and recorded nothing: check git -C %s log, and to undo "+
		"the merge run git -C %s reset --keep %s", command, t.root, why, t.root, t.root, t.head)
}

// regateAbortLeftMerge is what regateMergeTip does after a merge, command,
// that made no merge commit, as outcome says ("made no merge commit", or how
// it was stopped): it reads the conflicting files, undoes a merge left in
// progress with git merge --abort (aborted then says so, for the refusal),
// and checks that HEAD is t.head and the tree is clean, as they were before
// the merge. files is `git diff --name-only --diff-filter=U -z`, the
// conflicting files, each ended by a NUL. Its git commands run on a context
// ctx's cancellation does not reach, the abort in a process group of its own
// (regateUncancelledGit, which writes its line to stderr). The error is for a
// worktree it cannot read or could not put back as it was
// (regateMergeNotBack).
func regateAbortLeftMerge(ctx context.Context, t gateTarget, env []string, stderr io.Writer, w gateWording, command,
	outcome string) (files, aborted string, err error) {
	uncancelled := context.WithoutCancel(ctx)
	// The conflicting files are read before the abort clears them, split on
	// NUL (-z), so a name with a space stays whole and one with non-ASCII
	// letters is not quoted (core.quotePath); a failure to read them only
	// leaves them out of the refusal, and the checks after the abort read the
	// worktree again.
	files, _ = gateGitOutput(uncancelled, w.prefix, t.root, "diff", "--name-only", "--diff-filter=U", "-z")
	merging, err := gateGitPathExists(uncancelled, w.prefix, t.root, "a merge", "MERGE_HEAD")
	if err != nil {
		return "", "", regateMergeNotBack(t, command, outcome, fmt.Sprintf("whether a merge is in progress cannot be read (%v)", err),
			false)
	}
	if merging {
		ar, abortTimedOut, err := regateUncancelledGit(ctx, t.root, env, stderr, "merge", "--abort")
		var abortSignaled regateGitSignaled
		switch {
		case errors.As(err, &abortSignaled):
			return "", "", regateMergeNotBack(t, command, outcome, "a merge was in progress, and git merge --abort "+abortSignaled.Error(),
				false)
		case err != nil:
			return "", "", regateMergeNotBack(t, command, outcome, fmt.Sprintf("a merge is in progress, and git merge --abort could not "+
				"run (%v)", err), false)
		case abortTimedOut || ar.code != 0:
			return "", "", regateMergeNotBack(t, command, outcome, "a merge is in progress, and git merge --abort failed"+
				regateStderrTail(ar.stderr), false)
		}
		aborted = " (git merge --abort undid the merge it left in progress)"
	}
	head, err := gateHead(uncancelled, w.prefix, t.root)
	if err != nil {
		return "", "", regateMergeNotBack(t, command, outcome, fmt.Sprintf("HEAD cannot be read (%v)", err), false)
	}
	dirty, err := gateDirtyLines(uncancelled, w.prefix, t.root)
	if err != nil {
		return "", "", regateMergeNotBack(t, command, outcome, fmt.Sprintf("the worktree cannot be read (%v)", err), head != t.head)
	}
	if head != t.head || len(dirty) > 0 {
		return "", "", regateMergeNotBack(t, command, outcome, fmt.Sprintf("HEAD is %s (it was %s), and the worktree has %d "+
			"uncommitted change(s) or untracked file(s)", head, t.head, len(dirty)), head != t.head)
	}
	return files, aborted, nil
}

// regateMergeNotBack is regateAbortLeftMerge's error for a merge that ended
// with outcome ("made no merge commit", or how it was stopped) and left the
// worktree in state, not as it was: nothing was recorded, and what puts the
// worktree back. With HEAD still the HEAD before the merge, t.head (moved
// false, or HEAD not read), that is git reset --merge to it, whatever the
// merge left: a merge in progress (as git merge --abort does), or its result
// staged with no merge in progress (a git stopped in a hook, which git merge
// --abort refuses and git reset --keep leaves in the files). The tree was
// clean before the merge, so what it drops from the index is the merge's.
// With HEAD moved off t.head (moved: a commit was made, the merge commit
// before a post-merge hook was stopped, or a hook's), it is git reset --keep,
// as for a merge commit Regate keeps (regateMergeLeft): it stops rather than
// drop a change of the user's to a file those commits changed.
func regateMergeNotBack(t gateTarget, command, outcome, state string, moved bool) error {
	undo := fmt.Sprintf("git -C %s reset --merge %s, which also undoes a merge in progress", t.root, t.head)
	if moved {
		undo = fmt.Sprintf("git -C %s reset --keep %s, which drops the commits made on it and stops rather than drop a change of "+
			"yours", t.root, t.head)
	}
	return fmt.Errorf("org: regate: %s in %s %s, and the worktree is not as it was before: %s; nothing was recorded: check it "+
		"with git -C %s status, put it back with %s, and run ralph org regate --branch %s again", command, t.root, outcome, state,
		t.root, undo, t.branch)
}

// regateUndoMerge undoes res.Merge, the merge commit Regate made on
// res.PreMergeHead in t.root (D5), with git reset --keep res.PreMergeHead,
// and sets res.Undone. It undoes only a HEAD that is still res.Merge in a
// clean tree: a commit made after it would go with it, and reset --keep stops
// on a dirty tree's changes or carries them along. Its git commands run on a
// context ctx's cancellation does not reach, and the reset in a process
// group of its own, so an interrupt does not stop the undo half-way
// (regateUncancelledGit, which writes its line to stderr). When it does not
// undo, it sets res.NotUndone (regateMergeLeft).
func regateUndoMerge(ctx context.Context, t gateTarget, env []string, stderr io.Writer, res RegateResult) RegateResult {
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
	r, timedOut, err := regateUncancelledGit(ctx, t.root, env, stderr, "reset", "--keep", res.PreMergeHead)
	var signaled regateGitSignaled
	switch {
	case errors.As(err, &signaled):
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("git reset --keep %s %v", res.PreMergeHead, signaled),
			regateStderrTail(r.stderr))
	case err != nil:
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("git reset --keep %s could not run (%v)", res.PreMergeHead, err), "")
	case timedOut:
		res.NotUndone = regateMergeLeft(t.root, res, fmt.Sprintf("git reset --keep %s ran past %s and was stopped", res.PreMergeHead,
			regateGitTimeout), regateStderrTail(r.stderr))
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
// when the push has not and nothing else moved it (regateSetTrackingRef,
// from the value it read right before the push). A push that fails or runs
// past regateGitTimeout sets res.PushFailed. The record and HEAD stay as they
// are either way (D5): running Regate again finds the push landed (AC3) or
// pushes only (D4). The error is ctx done before or during the push, or a
// push a signal ended (regatePushInterrupted for both), a git that cannot
// run, or a remote-tracking ref that cannot be read before the push or
// updated after it.
func regatePush(ctx context.Context, t gateTarget, env []string, res RegateResult) (RegateResult, error) {
	if ctx.Err() != nil {
		return res, regatePushInterrupted(fmt.Sprintf("interrupted (%v)", context.Cause(ctx)), t, false)
	}
	spec := t.head + ":refs/heads/" + t.branch
	command := "git push origin " + spec
	kept := fmt.Sprintf("the gate_passed record of %s and HEAD are kept: check origin (git remote get-url origin), its "+
		"credentials, and whether someone pushed to %s after the fetch, and run ralph org regate --branch %s again, which "+
		"pushes only, or says what changed", t.head, t.branch, t.branch)
	before, err := regateTrackingRef(ctx, t)
	switch {
	case err != nil && ctx.Err() != nil:
		return res, regatePushInterrupted(fmt.Sprintf("interrupted (%v)", context.Cause(ctx)), t, false)
	case err != nil:
		return res, fmt.Errorf("%w, so %s was not pushed to origin's %s: the gate_passed record of it and HEAD are kept; run "+
			"ralph org regate --branch %s again, which pushes it", err, t.head, t.branch, t.branch)
	}
	r, timedOut, err := regateRemoteGit(ctx, t.root, regateRemoteEnv(env), "push", "origin", spec)
	var signaled regateGitSignaled
	switch {
	case err != nil && ctx.Err() != nil:
		return res, regatePushInterrupted(fmt.Sprintf("interrupted (%v)", context.Cause(ctx)), t, true)
	case errors.As(err, &signaled):
		return res, regatePushInterrupted(fmt.Sprintf("%s in %s %v", command, t.root, signaled), t, true)
	case err != nil:
		return res, fmt.Errorf("%w; %s", err, kept)
	case timedOut:
		res.PushFailed = fmt.Sprintf("org: regate: %s in %s ran past %s and was stopped, so origin may not have it: %s%s",
			command, t.root, regateGitTimeout, kept, regateStderrTail(r.stderr))
		return res, nil
	case r.code != 0:
		res.PushFailed = fmt.Sprintf("org: regate: %s in %s failed (exit %d), so origin's %s is not %s: %s%s",
			command, t.root, r.code, t.branch, t.head, kept, regateStderrTail(r.stderr))
		return res, nil
	}
	res.Pushed = true
	if hook := regateTestHooks.afterPush; hook != nil {
		hook()
	}
	return res, regateSetTrackingRef(context.WithoutCancel(ctx), t, before)
}

// regatePushInterrupted is regatePush's error for a push that why (ctx done,
// "interrupted (<cause>)", or a signal that ended git) kept from running, or
// ended while it ran (during): the record and HEAD are kept, and running
// Regate again pushes only, or finds the push landed.
func regatePushInterrupted(why string, t gateTarget, during bool) error {
	when := fmt.Sprintf("before %s was pushed to origin's %s, so nothing was pushed", t.head, t.branch)
	if during {
		when = fmt.Sprintf("while %s was pushed to origin's %s, so origin may or may not have it", t.head, t.branch)
	}
	return fmt.Errorf("org: regate: %s %s: the gate_passed record of it and HEAD are kept; run ralph org regate --branch %s "+
		"again, which pushes it, or finds it pushed", why, when, t.branch)
}

// regateSetTrackingRef points refs/remotes/origin/<branch> at t's HEAD, which
// Regate pushed to origin's branch, when the push has not: git updates it
// only when remote.origin.fetch maps the branch there, which Regate does not
// count on. before is the ref's value right before the push
// (regateTrackingRef). The ref is moved only from before (`git update-ref`
// with before as the old value): a value that is neither HEAD nor before was
// set after the push by something else, such as a fetch that brought a newer
// push to origin's branch, and is left as it is, so that Regate does not
// rewind it to HEAD. So is a ref that moves off before between the read and
// the update-ref, which then fails; an update-ref that fails with the ref
// still at before is the error.
func regateSetTrackingRef(ctx context.Context, t gateTarget, before string) error {
	ref := "refs/remotes/origin/" + t.branch
	now, err := regateTrackingRef(ctx, t)
	if err != nil {
		return fmt.Errorf("%w; %s was pushed to origin's %s", err, t.head, t.branch)
	}
	if now == t.head || now != before {
		return nil
	}
	if hook := regateTestHooks.beforeTrackingUpdate; hook != nil {
		hook()
	}
	if _, err := gateGitOutput(ctx, regatePrefix, t.root, "update-ref", ref, t.head, before); err != nil {
		if again, readErr := regateTrackingRef(ctx, t); readErr == nil && again != before {
			return nil
		}
		return fmt.Errorf("%w; %s was pushed to origin's %s, and git fetch origin %s brings %s up to date", err, t.head, t.branch,
			t.branch, ref)
	}
	return nil
}

// regateTrackingRef reads refs/remotes/origin/<branch> in t.root, "" when
// there is none (`git rev-parse --verify --quiet`). The error is for a git
// that could not run or was stopped by ctx.
func regateTrackingRef(ctx context.Context, t gateTarget) (string, error) {
	r, err := gateGit(ctx, regatePrefix, t.root, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+t.branch)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(r.stdout), nil
}
