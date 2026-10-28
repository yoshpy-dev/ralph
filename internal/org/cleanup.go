package org

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// The cleanup of a merged feature org's worktree (CleanupFeatureWorktree), S1
// of plan docs/plans/active/2026-10-28-org-merge-cleanup.md (issue #263): once
// a feature org's PR is merged, ralph org merge (right after it records
// pr_merged, regate.lock still held) and ralph org cleanup (for a PR merged on
// GitHub) remove the feature worktree ralph org start --plan made (feature.go),
// its local branch, and its scripts/ralph-worktree.sh record. The removal is
// the script's `cleanup --id org-<org_id> --force-branch`
// (FeatureWorktrees.Cleanup, D1); this file holds the checks before it (D2)
// and the record after it (D5), in this order:
//
//  0. regate.lock is held, so no ralph org regate, merge, or other cleanup of
//     the ledger runs, and once 1 passed its marker names the org, so a spawn
//     or a start of that org is refused until the lock is released
//     (regateSpawnErr);
//  1. under the manifest lock, a complete feature binding names the branch
//     and its org is not running (cleanupFirstCheck);
//  2. the record org-<org_id> exists and is the binding's: its branch,
//     worktree_path, and kind org (checkRecord), so that a worktree another
//     feature made on the same org_id is never removed;
//  3. the caller does not run inside the worktree;
//  4. the worktree is the root of a git worktree, with no rebase or merge in
//     progress and nothing uncommitted or untracked (a file .gitignore names
//     does not count and goes with the worktree, D6);
//  5. the local branch's tip is the head the PR merged, so the branch holds no
//     commit that was not merged;
//  6. the worktree has that branch checked out at that head, so no commit made
//     there on a detached HEAD or another branch is lost.
//
// The script removes the worktree, then the branch, then the record, and
// stops at the first step that fails. The next cleanup goes on from what is
// left (D2's resume), with the checks of what is left: with the worktree's
// directory gone and the branch left, 1, 2, and 5, and git's registration of
// the missing worktree, if any, is removed alone (git worktree remove of its
// path; git worktree prune would remove every other missing worktree's too);
// with only the record left, 1 and 2. With the record gone, nothing is
// removed: what is left cannot be told from what another feature left. A
// refusal removes nothing and says why and what to do next. When the next
// cleanup can go on once the cause is fixed (3 to 6), what to do is that fix
// and the next cleanup, with no command that cleans up by hand, since one
// would skip the checks after the one that refused; only where ralph cannot
// go on by itself does the refusal give such a command (CleanupResult.Manual).

const (
	// CleanupViaMerge and CleanupViaCleanup are the values of
	// CleanupParams.Via, recorded as worktree_cleaned's via=: ralph org merge
	// cleaned up the branch it merged, or ralph org cleanup a branch merged
	// on GitHub.
	CleanupViaMerge   = "merge"
	CleanupViaCleanup = "cleanup"
	// CleanupResumedWorktreeGone and CleanupResumedRecordOnly are the values
	// of CleanupResult.Resumed, what an earlier cleanup that stopped part way
	// left: the branch and the record with the worktree's directory gone, or
	// the record alone.
	CleanupResumedWorktreeGone = "worktree_gone"
	CleanupResumedRecordOnly   = "record_only"
	// cleanupPrefix starts every error the cleanup returns.
	cleanupPrefix = "org: cleanup: "
	// cleanupDirtyListed is how many entries of git status the refusal of a
	// dirty worktree names, on its one line.
	cleanupDirtyListed = 3
)

// cleanupPlainWord matches a word that no shell reads specially anywhere in
// it, which cleanupShellWord leaves unquoted.
var cleanupPlainWord = regexp.MustCompile(`^[A-Za-z0-9_./:@+,-]+$`)

// cleanupTestHooks are the points where a test stops the cleanup. Each is nil
// outside tests.
var cleanupTestHooks struct {
	// beforeScript runs right before anything is removed, once every check
	// passed, regate.lock held with its marker naming the org.
	beforeScript func()
}

// CleanupParams describes one cleanup of a merged feature branch's worktree.
type CleanupParams struct {
	// Branch is the merged feature branch.
	Branch string
	// Head is the head the branch's PR merged: the head the gate passed for
	// ralph org merge, gh's headRefOid for ralph org cleanup. The local branch
	// and the worktree must be at it (D2's 5 and 6).
	Head string
	// RepoRoot is the root of the main worktree, where
	// scripts/ralph-worktree.sh runs (FeatureRepoRoot).
	RepoRoot string
	// Cwd is the caller's working directory, which must not be inside the
	// worktree (D2's 3); "" means os.Getwd().
	Cwd string
	// Via is CleanupViaMerge or CleanupViaCleanup, which worktree_cleaned
	// records.
	Via string
}

// CleanupResult is the cleanup's outcome. With an error once the removal ran
// (a worktree_cleaned that could not be appended), it is returned too, as far
// as it got.
type CleanupResult struct {
	// Refused is set when the cleanup removed nothing: why, and the next step.
	// Nothing was recorded. It does not start with "org: cleanup: ", so a
	// caller can put it in a line of its own, and it holds no line break.
	Refused string
	// Failed is set when every check passed but the removal failed: what
	// failed. What the script removed before it stopped stays removed, and
	// the next cleanup goes on from what is left. Nothing was recorded.
	Failed string
	// Manual is the shell command that cleans up by hand, set with Refused or
	// Failed only where ralph cannot go on by itself, to run only once the
	// reason has been checked: ralph-worktree.sh cleanup --force-branch in
	// RepoRoot when the record's path or kind is not the binding's, when the
	// worktree is left with no local branch, and when the removal failed; git
	// worktree remove and git branch -D of what git still has when the record
	// is gone; and git branch -D of the binding's branch alone when the record
	// is on another branch. A refusal the next cleanup can go on from once its
	// cause is fixed (D2's 3 to 6) has none: its reason names the fix and the
	// next cleanup, which runs the checks a command by hand would skip.
	Manual string
	// Cleaned reports that the script removed the worktree, the branch, and
	// the record, or what of them was left.
	Cleaned bool
	// Recorded reports that worktree_cleaned was appended. With Cleaned it is
	// false only when the append failed, and the error says so.
	Recorded bool
	// Resumed is set with Cleaned when an earlier cleanup had stopped part
	// way: CleanupResumedWorktreeGone or CleanupResumedRecordOnly.
	Resumed string
	// NothingLeft is set with Refused when nothing was left and so there was
	// nothing to remove: the record org-<org_id>, the feature worktree, git's
	// registration of it, and the local branch are all gone, as a cleanup
	// that ended leaves them. ralph org cleanup does not count such a branch
	// as left.
	NothingLeft bool
	// OrgID and Worktree are the org and the feature worktree of the branch's
	// binding, set once the first check passed; Branch and Head are
	// CleanupParams'.
	OrgID, Worktree, Branch, Head string
}

// CleanupFeatureWorktree cleans up p.Branch's feature worktree (see the top of
// this file) with regate.lock, which it takes the way Regate and Merge do
// (acquireHeldFileLock, regateLockWait) and releases when it returns: it
// refuses while another ralph org regate, merge, or cleanup of the ledger
// holds the lock. Once it holds the lock it empties the marker an earlier
// Regate left there, as Regate does, and goes on as
// cleanupFeatureWorktreeLocked. ralph org cleanup calls it; ralph org merge,
// which holds the lock already, calls cleanupFeatureWorktreeLocked.
func (o *Org) CleanupFeatureWorktree(ctx context.Context, p CleanupParams) (CleanupResult, error) {
	res := CleanupResult{Branch: p.Branch, Head: p.Head}
	if refused, err := checkCleanupParams(p); err != nil || refused != "" {
		res.Refused = PrintableInboxText(refused, false)
		return res, err
	}
	stateDir := filepath.Dir(o.Manifest.Path())
	lock, err := acquireHeldFileLock(stateDir, regateLockFile, "regate", regateLockWait)
	switch {
	case errors.Is(err, errFileLockHeld):
		res.Refused = PrintableInboxText(fmt.Sprintf("another ralph org regate, merge, or cleanup runs on this ledger (%s is held): "+
			"one runs at a time, so wait for it to end and %s", regateLockPathIn(stateDir), cleanupAgain(p.Branch)), false)
		return res, nil
	case err != nil:
		// err names the lock's path, or its dir.
		return res, fmt.Errorf(cleanupPrefix+"nothing was removed, since regate.lock could not be taken: %w", err)
	}
	defer func() { _ = lock.release() }()
	// The marker an earlier Regate left goes at once, so that a spawn of its
	// org is not refused while this cleanup waits for the manifest lock.
	if err := writeLockMarker(cleanupPrefix, lock.f, nil); err != nil {
		return res, err
	}
	return o.cleanupFeatureWorktreeLocked(ctx, lock, p)
}

// cleanupFeatureWorktreeLocked is CleanupFeatureWorktree for a caller that
// holds regate.lock (lock) already, as ralph org merge does from its checks
// through its record (mergeLocked), and still holds it once this returns: it
// writes lock's marker for the binding's org under the manifest lock when the
// first check passes (cleanupFirstCheck) and leaves it there, so a spawn of
// that org is refused until the caller releases the lock. It runs the checks
// of the top of this file in their order, then the removal and the record
// (cleanupRun.remove). The error is for a manifest or a marker that cannot be
// read or written, a git that cannot run or fails in a way no check expects,
// ctx done before the removal (nothing removed), and a worktree_cleaned that
// cannot be appended after it (Cleaned set, Recorded not).
func (o *Org) cleanupFeatureWorktreeLocked(ctx context.Context, lock *heldFileLock, p CleanupParams) (CleanupResult, error) {
	res := CleanupResult{Branch: p.Branch, Head: p.Head}
	if refused, err := checkCleanupParams(p); err != nil || refused != "" {
		res.Refused = PrintableInboxText(refused, false)
		return res, err
	}
	cwd := p.Cwd
	if cwd == "" {
		wd, err := os.Getwd()
		if err != nil {
			return res, fmt.Errorf(cleanupPrefix+"nothing was removed, since the working directory cannot be read: %w", err)
		}
		cwd = wd
	}
	b, refused, err := o.cleanupFirstCheck(ctx, p.Branch, lock)
	switch {
	case err != nil:
		return res, err
	case refused != "":
		res.Refused = PrintableInboxText(refused, false)
		return res, nil
	}
	c := &cleanupRun{o: o, p: p, root: resolvedOrClean(mustAbs(p.RepoRoot)), cwd: resolvedOrClean(mustAbs(cwd)),
		orgID: b.orgID, f: b.binding, worktrees: o.worktrees()}
	res.OrgID, res.Worktree = c.orgID, c.f.Worktree
	return c.run(ctx, res)
}

// checkCleanupParams is the check of p before any lock. A branch that is not
// named, or a head that is not a commit SHA (gitObjectPattern), with which
// the cleanup could not tell whether the branch holds commits the PR did not
// merge, is refused. An empty RepoRoot or a Via that is neither
// CleanupViaMerge nor CleanupViaCleanup is the caller's mistake, an error.
func checkCleanupParams(p CleanupParams) (string, error) {
	switch {
	case p.RepoRoot == "":
		return "", errors.New(cleanupPrefix + "no repo root given: the cleanup runs " + worktreeScriptRel + " in the main worktree")
	case p.Via != CleanupViaMerge && p.Via != CleanupViaCleanup:
		return "", fmt.Errorf(cleanupPrefix+"via %q is neither %s nor %s", p.Via, CleanupViaMerge, CleanupViaCleanup)
	case p.Branch == "":
		return "no branch named: name the merged feature branch to clean up (ralph org merge-order --all lists the merged ones)", nil
	case !gitObjectPattern.MatchString(p.Head):
		return fmt.Sprintf("the head the PR of %s merged is not known (%q is not a commit SHA), so whether the branch holds "+
			"commits that were not merged cannot be told: check the PR's head with gh pr view --json headRefOid and %s",
			p.Branch, p.Head, cleanupAgain(p.Branch)), nil
	}
	return "", nil
}

// cleanupAgain is the last step of a refusal that the cleanup can go on from
// once its cause is gone, the branch a shell word (cleanupShellWord).
func cleanupAgain(branch string) string {
	return "run ralph org cleanup --branch " + cleanupShellWord(branch) + " again"
}

// cleanupFirstCheck is D2's check 1, made under the manifest lock with
// regate.lock (lock) held: it reads the ledger line by line
// (readMergeLedger), finds branch in it as Regate does (readRegateBranch),
// and returns the refusal of a branch no complete binding names (a branch
// with gate records only included), whose binding's org_id is not one ralph
// makes (ValidateIdentifier: the script reads --id org-Auth;x as
// org-auth-x, the record of another org, which RunningOrgs, reading Auth;x,
// does not see run), or whose org runs. When the check passes
// it writes lock's marker before the manifest lock is released: the
// binding's org, branch, and worktree, so a spawn of that org that comes
// after it refuses (regateSpawnErr), and one that came before it is running
// here. A refusal leaves the marker as it was.
func (o *Org) cleanupFirstCheck(ctx context.Context, branch string, lock *heldFileLock) (regateBranch, string, error) {
	var b regateBranch
	var refused string
	err := withManifestLockContext(ctx, filepath.Dir(o.Manifest.Path()), func() error {
		l, err := readMergeLedger(o.Manifest.Path())
		if err != nil {
			return fmt.Errorf(cleanupPrefix+"read manifest: %w", err)
		}
		b = readRegateBranch(l, branch)
		switch {
		case b.binding == nil:
			refused = fmt.Sprintf("no complete feature binding in the ledger names the branch %s, so it has no feature worktree "+
				"to clean up: the cleanup removes only the worktree ralph org start --plan made for a feature org, and a branch "+
				"with gate records only has none", branch)
			return nil
		case ValidateIdentifier("org_id", b.orgID) != nil:
			refused = fmt.Sprintf("the ledger's binding of the branch %s names org_id %q, which is not of the form ralph makes "+
				"(%s), so which worktree record and org it names cannot be told and nothing is removed: check the binding in %s",
				branch, b.orgID, identifierPattern, o.Manifest.Path())
			return nil
		case b.running && b.runsOn != "":
			refused = fmt.Sprintf("the branch %s is bound to org_id %q, which is running, now for another feature, on the branch "+
				"%s, whose worktree the record %s may name: wait for the org to be disbanded and %s", branch, b.orgID, b.runsOn,
				featureWorktreeID(b.orgID), cleanupAgain(branch))
			return nil
		case b.running:
			refused = fmt.Sprintf("the branch %s is bound to org_id %q, which is running and may still use its feature worktree "+
				"%s: wait for the org to be disbanded and %s", branch, b.orgID, b.binding.Worktree, cleanupAgain(branch))
			return nil
		}
		return writeLockMarker(cleanupPrefix, lock.f, &regateMarker{OrgID: b.orgID, Branch: branch, Worktree: b.binding.Worktree,
			PID: os.Getpid()})
	})
	return b, refused, err
}

// cleanupRun is one cleanup once its first check passed.
type cleanupRun struct {
	o         *Org
	p         CleanupParams
	root      string          // the main worktree, symlinks resolved
	cwd       string          // the caller's working directory, symlinks resolved
	orgID     string          // the org of the branch's binding
	f         *FeatureBinding // the branch's latest complete binding
	worktrees FeatureWorktrees
}

// cleanupState is what git and the file system have of the feature worktree
// before anything is removed.
type cleanupState struct {
	dirExists  bool   // the worktree's directory is there
	notDir     bool   // something that is not a directory is at its path
	registered bool   // git worktree list names a worktree at its path
	branchTip  string // the local branch's tip; "" when there is no such branch
}

// id is the ralph-worktree.sh record id of the binding's org.
func (c *cleanupRun) id() string { return featureWorktreeID(c.orgID) }

// scriptCmd is the Manual of a record that is the binding's: the script's
// cleanup, run in the main worktree.
func (c *cleanupRun) scriptCmd() string {
	return "cd " + cleanupShellWord(c.root) + " && " + featureWorktreeCleanup(c.orgID) + " --force-branch"
}

// gitCmd is `git -C <main worktree> args...` as a shell command
// (cleanupGitCmd).
func (c *cleanupRun) gitCmd(args ...string) string {
	return cleanupGitCmd(c.root, args...)
}

// cleanupGitCmd is `git -C <dir> args...` as a shell command, for a Manual
// and for a command a refusal's reason names: each word is cleanupShellWord,
// so the command runs as it is written whatever the path holds.
func cleanupGitCmd(dir string, args ...string) string {
	words := []string{"git", "-C", cleanupShellWord(dir)}
	for _, a := range args {
		words = append(words, cleanupShellWord(a))
	}
	return strings.Join(words, " ")
}

// run is D2's checks 2 to 6, or those of what an earlier cleanup left (see
// the top of this file), then the removal. res holds what the first check
// found.
func (c *cleanupRun) run(ctx context.Context, res CleanupResult) (CleanupResult, error) {
	refuse := func(reason, manual string) (CleanupResult, error) {
		res.Refused, res.Manual = PrintableInboxText(reason, false), PrintableInboxText(manual, false)
		return res, nil
	}
	rec, ok, err := c.worktrees.Lookup(c.root, c.id())
	switch {
	case errors.Is(err, errWorktreeScriptMissing):
		// err ends with start's next step, not the cleanup's, so it is not
		// quoted.
		return refuse(fmt.Sprintf("%s is missing from %s (ralph init ships it into the repo), so the worktree record %s cannot "+
			"be read and nothing is removed: restore the script and %s", worktreeScriptRel, c.root, c.id(),
			cleanupAgain(c.p.Branch)), "")
	case err != nil:
		return refuse(fmt.Sprintf("the worktree record %s cannot be read (%v), so nothing is removed: fix it and %s", c.id(), err,
			cleanupAgain(c.p.Branch)), "")
	case ok:
		if reason, manual := c.checkRecord(rec); reason != "" {
			return refuse(reason, manual)
		}
	}
	st, err := c.readState(ctx)
	if err != nil {
		return res, err
	}
	if !ok {
		reason, manual, nothingLeft := c.recordGone(st)
		res.NothingLeft = nothingLeft
		return refuse(reason, manual)
	}
	var resumed, reason, manual string
	switch {
	case st.notDir:
		reason = fmt.Sprintf("%s, the path of the feature worktree, is not a directory: check what it is, remove it by hand, and %s",
			c.f.Worktree, cleanupAgain(c.p.Branch))
	case st.dirExists:
		reason, manual, err = c.checkWorktree(ctx, st.branchTip)
	case st.branchTip != "":
		resumed = CleanupResumedWorktreeGone
		// The worktree's directory is gone, so the branch is put back from the
		// main worktree. git branch -f refuses a branch a registered worktree
		// has checked out, even one whose directory is gone; git update-ref
		// moves it, and no working tree of it is left to fall behind.
		reset := c.gitCmd("branch", "-f", c.f.Branch, c.p.Head)
		if st.registered {
			reset = c.gitCmd("update-ref", "refs/heads/"+c.f.Branch, c.p.Head)
		}
		reason, manual = c.checkBranchTip(st.branchTip, reset)
	default:
		resumed = CleanupResumedRecordOnly
	}
	switch {
	case err != nil:
		return res, err
	case reason != "":
		return refuse(reason, manual)
	}
	return c.remove(ctx, st, resumed, res)
}

// readState reads cleanupState: the worktree's path with os.Stat, git
// worktree list --porcelain in the main worktree for its registration, and
// the local branch's tip there.
func (c *cleanupRun) readState(ctx context.Context) (cleanupState, error) {
	var st cleanupState
	info, err := os.Stat(c.f.Worktree)
	switch {
	case err == nil:
		st.dirExists, st.notDir = info.IsDir(), !info.IsDir()
	case !errors.Is(err, fs.ErrNotExist):
		return st, fmt.Errorf(cleanupPrefix+"nothing was removed, since the feature worktree %s cannot be read: %w", c.f.Worktree,
			pathErrorCause(err))
	}
	list, err := cleanupGitOutput(ctx, c.root, "worktree", "list", "--porcelain")
	if err != nil {
		return st, err
	}
	for line := range strings.SplitSeq(list, "\n") {
		if path, ok := strings.CutPrefix(line, "worktree "); ok && samePath(path, c.f.Worktree) {
			st.registered = true
		}
	}
	r, err := cleanupGit(ctx, c.root, "rev-parse", "--verify", "--quiet", "refs/heads/"+c.f.Branch+"^{commit}")
	switch {
	case err != nil:
		return st, err
	case r.code == 0:
		st.branchTip = strings.TrimSpace(r.stdout)
	case r.code != 1:
		return st, fmt.Errorf(cleanupPrefix+"git rev-parse --verify refs/heads/%s in %s: exit %d: %s", c.f.Branch, c.root, r.code, r.stderr)
	}
	return st, nil
}

// recordGone is the refusal when the record org-<org_id> does not exist
// (D2's resume): what is left cannot be told from what another feature
// left, so nothing is removed. Manual removes by hand what git still has of
// the binding's worktree and branch: the worktree, or its registration when
// its directory is gone (git worktree remove takes a missing directory too),
// and the branch, whose tip the reason says to check first. nothingLeft
// reports that git has neither (CleanupResult.NothingLeft), with no Manual.
func (c *cleanupRun) recordGone(st cleanupState) (reason, manual string, nothingLeft bool) {
	var left, cmds []string
	if st.dirExists || st.registered {
		left = append(left, "the feature worktree "+c.f.Worktree)
		cmds = append(cmds, c.gitCmd("worktree", "remove", c.f.Worktree))
	}
	if st.branchTip != "" {
		left = append(left, fmt.Sprintf("the branch %s at %s", c.f.Branch, st.branchTip))
		cmds = append(cmds, c.gitCmd("branch", "-D", c.f.Branch))
	}
	gone := fmt.Sprintf("the worktree record %s of the branch %s does not exist", c.id(), c.f.Branch)
	if len(left) == 0 {
		return gone + fmt.Sprintf(", and neither its feature worktree %s nor the branch is left: nothing is left to clean up",
			c.f.Worktree), "", true
	}
	reason = gone + fmt.Sprintf(", so ralph cannot tell that what is left is this feature's and removes nothing: %s left; "+
		"remove it by hand", strings.Join(left, " and "))
	if st.branchTip != "" {
		reason += fmt.Sprintf(" once you have checked that the branch holds nothing that was not merged (%s lists what %s, "+
			"the head its PR merged, does not have)", c.gitCmd("log", c.p.Head+".."+c.f.Branch), c.p.Head)
	}
	return reason, strings.Join(cmds, " && "), false
}

// checkRecord is D2's check 2: rec, the record org-<org_id>, has the
// binding's branch, worktree_path (compared with symlinks resolved), and
// kind org. Otherwise it may be the record of another feature started on the
// same org_id after this one, so nothing is removed; when it is on another
// branch, Manual deletes only the binding's branch, and when only its path
// or kind differs, Manual is the script's cleanup, once the record has been
// checked.
func (c *cleanupRun) checkRecord(rec WorktreeRecord) (string, string) {
	var diffs []string
	if rec.Branch != c.f.Branch {
		diffs = append(diffs, fmt.Sprintf("branch %q, not %q", rec.Branch, c.f.Branch))
	}
	if !samePath(rec.WorktreePath, c.f.Worktree) {
		diffs = append(diffs, fmt.Sprintf("worktree_path %q, not %q", rec.WorktreePath, c.f.Worktree))
	}
	if rec.Kind != featureWorktreeKind {
		diffs = append(diffs, fmt.Sprintf("kind %q, not %q", rec.Kind, featureWorktreeKind))
	}
	if len(diffs) == 0 {
		return "", ""
	}
	reason := fmt.Sprintf("the worktree record %s records %s: it is not the record of the feature worktree org_id %q had for "+
		"the branch %s and may be another feature's on the same org_id, so nothing is removed", c.id(), strings.Join(diffs, ", "),
		c.orgID, c.f.Branch)
	if rec.Branch != c.f.Branch {
		reason += fmt.Sprintf(": if the branch %s is left in %s, delete it by hand once you have checked that it holds "+
			"nothing that was not merged (%s)", c.f.Branch, c.root, c.gitCmd("log", c.p.Head+".."+c.f.Branch))
		return reason, c.gitCmd("branch", "-D", c.f.Branch)
	}
	return reason + fmt.Sprintf(": check the record (%s state-path %s names its file) and, once it is this feature's, "+
		"clean up by hand", worktreeScriptRel, c.id()), c.scriptCmd()
}

// checkWorktree is D2's checks 3 to 6 of the feature worktree, whose
// directory is there, and tip, the local branch's tip ("" for none). The next
// cleanup goes on from each refusal once its cause is fixed, so none has a
// Manual (a command by hand would skip the checks after the one that
// refused), but for a worktree left with no local branch (checkBranchTip).
// A directory that is not a git worktree's root has none either: the script
// cannot remove it. Check 5 runs once the branch the worktree has checked
// out is read, which picks the way its refusal puts the branch back.
func (c *cleanupRun) checkWorktree(ctx context.Context, tip string) (reason, manual string, err error) {
	wt, again := c.f.Worktree, cleanupAgain(c.p.Branch)
	if resolved := resolvedOrClean(wt); c.cwd == resolved || strings.HasPrefix(c.cwd, resolved+string(filepath.Separator)) {
		return fmt.Sprintf("ralph runs in %s, inside the feature worktree %s, which the cleanup removes: run ralph org cleanup "+
			"--branch %s from outside it, such as from the main worktree %s", c.cwd, wt, cleanupShellWord(c.p.Branch),
			c.root), "", nil
	}
	r, err := cleanupGit(ctx, wt, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", "", err
	}
	if top := strings.TrimSuffix(r.stdout, "\n"); r.code != 0 || !samePath(top, wt) {
		said := top
		if r.code != 0 {
			said = r.stderr
		}
		return fmt.Sprintf("the feature worktree %s is not the root of a git worktree (git rev-parse --show-toplevel: %s), so "+
			"the script cannot remove it: check what is there, remove it by hand, and %s", wt, said, again), "", nil
	}
	if reason, err = c.checkClean(ctx); err != nil || reason != "" {
		return reason, "", err
	}
	branch, err := c.checkedOut(ctx)
	if err != nil {
		return "", "", err
	}
	// With the branch checked out here, git reset --hard puts the worktree
	// back with it (check 4 passed, so it holds no change to lose). With a
	// detached HEAD or another branch here, git reset --hard would move that
	// instead, so the branch is put back from the main worktree, and check 6
	// refuses next.
	reset := c.gitCmd("branch", "-f", c.f.Branch, c.p.Head)
	if branch == c.f.Branch {
		reset = cleanupGitCmd(wt, "reset", "--hard", c.p.Head)
	}
	if reason, manual = c.checkBranchTip(tip, reset); reason != "" {
		return reason, manual, nil
	}
	reason, err = c.checkCheckout(ctx, branch)
	return reason, "", err
}

// checkClean is D2's check 4 past the worktree's root: no rebase or merge is
// stopped in the worktree (cleanupInProgress), and git status --porcelain
// --untracked-files=all lists nothing. Ignored files are not listed (D6).
func (c *cleanupRun) checkClean(ctx context.Context) (string, error) {
	wt, again := c.f.Worktree, cleanupAgain(c.p.Branch)
	what, err := cleanupInProgress(ctx, wt)
	if err != nil {
		return "", err
	}
	if what != "" {
		return fmt.Sprintf("%s is in progress in the feature worktree %s: finish it or undo it (git rebase --abort, git merge "+
			"--abort), keep what you need of it elsewhere, and %s", what, wt, again), nil
	}
	out, err := cleanupGitOutput(ctx, wt, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return "", err
	}
	var dirty []string
	for line := range strings.SplitSeq(out, "\n") {
		if line != "" {
			dirty = append(dirty, line)
		}
	}
	if len(dirty) == 0 {
		return "", nil
	}
	shown := make([]string, 0, cleanupDirtyListed)
	for _, line := range dirty[:min(len(dirty), cleanupDirtyListed)] {
		shown = append(shown, strings.TrimSpace(line))
	}
	list := strings.Join(shown, ", ")
	if more := len(dirty) - len(shown); more > 0 {
		list += fmt.Sprintf(", and %d more", more)
	}
	return fmt.Sprintf("the feature worktree %s has %d uncommitted change(s) or untracked file(s) (%s), which removing it would "+
		"lose: move what you need out of the worktree, remove the rest, and %s", wt, len(dirty), list, again), nil
}

// checkBranchTip is D2's check 5 of tip, the local branch's tip ("" for no
// such branch): it is the head the PR merged. A branch at another commit is
// refused with no Manual: its reason says to look at the commits the PR did
// not merge, keep what is needed on another branch, put the branch back at
// the head with reset, the command for where the branch is checked out
// (checkWorktree, run), and clean up again. With no such branch ralph cannot
// go on, since only the worktree can tell what was not merged, so Manual is
// the script's cleanup, once that has been checked.
func (c *cleanupRun) checkBranchTip(tip, reset string) (string, string) {
	switch tip {
	case c.p.Head:
		return "", ""
	case "":
		return fmt.Sprintf("the local branch %s does not exist in %s, though its feature worktree %s does: check what the "+
			"worktree has checked out (%s) and keep what you need of it elsewhere before it is removed", c.f.Branch, c.root,
			c.f.Worktree, cleanupGitCmd(c.f.Worktree, "status")), c.scriptCmd()
	}
	return fmt.Sprintf("the local branch %s is at %s, not at %s, the head its PR merged, so it may hold commits that were not "+
		"merged, which git branch -D would delete: look at them (%s), keep what you need on another branch, put the branch "+
		"back at the head its PR merged (%s), and %s", c.f.Branch, tip, c.p.Head, c.gitCmd("log", c.p.Head+".."+c.f.Branch),
		reset, cleanupAgain(c.p.Branch)), ""
}

// checkedOut is the branch the feature worktree has checked out (git
// symbolic-ref --short HEAD), "" for a detached HEAD.
func (c *cleanupRun) checkedOut(ctx context.Context) (string, error) {
	wt := c.f.Worktree
	r, err := cleanupGit(ctx, wt, "symbolic-ref", "--quiet", "--short", "HEAD")
	switch {
	case err != nil:
		return "", err
	case r.code == 1:
		return "", nil
	case r.code != 0:
		return "", fmt.Errorf(cleanupPrefix+"git symbolic-ref --short HEAD in %s: exit %d: %s", wt, r.code, r.stderr)
	}
	return strings.TrimSpace(r.stdout), nil
}

// checkCheckout is D2's check 6 of branch, the one the feature worktree has
// checked out ("" for a detached HEAD, checkedOut): it is the binding's
// branch, not a detached HEAD or another branch, whose commits made there
// would go with the worktree, and the worktree's HEAD is the head the PR
// merged.
func (c *cleanupRun) checkCheckout(ctx context.Context, branch string) (string, error) {
	wt, again := c.f.Worktree, cleanupAgain(c.p.Branch)
	switchCmd := cleanupGitCmd(wt, "switch", c.f.Branch)
	switch {
	case branch == "":
		return fmt.Sprintf("HEAD is detached in the feature worktree %s, not on the branch %s, and a commit made there would be "+
			"lost with the worktree: keep what you need on another branch, run %s, and %s", wt, c.f.Branch, switchCmd, again), nil
	case branch != c.f.Branch:
		return fmt.Sprintf("the feature worktree %s has the branch %s checked out, not %s, and its commits would be lost with the "+
			"worktree if they are on no other branch: keep what you need, run %s, and %s", wt, branch, c.f.Branch, switchCmd,
			again), nil
	}
	head, err := cleanupGitOutput(ctx, wt, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return "", err
	}
	if head = strings.TrimSpace(head); head != c.p.Head {
		return fmt.Sprintf("HEAD %s of the feature worktree %s is not %s, the head the PR merged: keep what you need on another "+
			"branch and %s", head, wt, c.p.Head, again), nil
	}
	return "", nil
}

// remove is the removal once the checks of st passed: with the worktree's
// directory gone and git still listing it, git worktree remove of its path
// alone; then the script's cleanup (FeatureWorktrees.Cleanup); then the
// record (record). resumed is CleanupResult.Resumed of a cleanup that
// succeeds. ctx done before anything is removed is an error.
func (c *cleanupRun) remove(ctx context.Context, st cleanupState, resumed string, res CleanupResult) (CleanupResult, error) {
	if hook := cleanupTestHooks.beforeScript; hook != nil {
		hook()
	}
	if ctx.Err() != nil {
		return res, fmt.Errorf(cleanupPrefix+"interrupted (%v) after the checks passed, before anything was removed", context.Cause(ctx))
	}
	if !st.dirExists && st.registered {
		r, err := cleanupGit(ctx, c.root, "worktree", "remove", c.f.Worktree)
		if err != nil {
			return res, err
		}
		if r.code != 0 {
			res.Failed = PrintableInboxText(fmt.Sprintf("git worktree remove %s in %s, which takes git's registration of the missing "+
				"feature worktree away, failed (exit %d: %s), so nothing was removed: fix what it says and %s", c.f.Worktree, c.root,
				r.code, r.stderr, cleanupAgain(c.p.Branch)), false)
			res.Manual = PrintableInboxText(c.gitCmd("worktree", "remove", c.f.Worktree)+" && "+c.scriptCmd(), false)
			return res, nil
		}
	}
	if err := c.worktrees.Cleanup(c.root, c.id()); err != nil {
		res.Failed = PrintableInboxText(fmt.Sprintf("%s cleanup --id %s --force-branch failed (%v): what it removed before it "+
			"stopped stays removed, and once its cause is fixed, ralph org cleanup --branch %s goes on from what is left",
			worktreeScriptRel, c.id(), err, cleanupShellWord(c.p.Branch)), false)
		res.Manual = PrintableInboxText(c.scriptCmd(), false)
		return res, nil
	}
	res.Cleaned, res.Resumed = true, resumed
	if err := c.record(ctx); err != nil {
		return res, err
	}
	res.Recorded = true
	return res, nil
}

// record appends worktree_cleaned as an org-level event of the binding's org
// under the manifest lock (D5), with ctx's cancellation dropped: what the
// script removed is gone whatever ctx says. manifestLockTimeout still bounds
// the wait for the lock. The ledger is not read again, since nothing in it
// can undo the removal.
func (c *cleanupRun) record(ctx context.Context) error {
	ev := ManifestEvent{TS: c.o.now(), OrgID: c.orgID, SeatID: "", Event: EventWorktreeCleaned,
		Details: cleanupDetails(c.p.Branch, c.p.Head, c.p.Via, c.f.Worktree)}
	err := withManifestLockContext(context.WithoutCancel(ctx), filepath.Dir(c.o.Manifest.Path()), func() error {
		return c.o.appendEvent(ev)
	})
	if err != nil {
		return fmt.Errorf(cleanupPrefix+"the feature worktree %s of the branch %s and its record %s were removed, but %s could not "+
			"be recorded (%w): only the ledger misses it; check that %s can be written", c.f.Worktree, c.p.Branch, c.id(),
			EventWorktreeCleaned, err, c.o.Manifest.Path())
	}
	return nil
}

// cleanupDetails is worktree_cleaned's Details (EventWorktreeCleaned): the
// worktree last and strconv.Quote'd, since a path may hold spaces.
func cleanupDetails(branch, head, via, worktree string) string {
	return "branch=" + branch + " head=" + head + " via=" + via + " worktree=" + strconv.Quote(worktree)
}

// cleanupInProgress names the operation stopped in the worktree wt: "a
// rebase" when git's rebase-merge or rebase-apply directory for it exists,
// "a merge" when its MERGE_HEAD does, "" for neither. It reads git's paths
// with git rev-parse --git-path, as gateGitPathExists does, through
// boundedGit.
func cleanupInProgress(ctx context.Context, wt string) (string, error) {
	names := []string{"rebase-merge", "rebase-apply", "MERGE_HEAD"}
	whats := []string{"a rebase", "a rebase", "a merge"}
	args := []string{"rev-parse"}
	for _, name := range names {
		args = append(args, "--git-path", name)
	}
	out, err := cleanupGitOutput(ctx, wt, args...)
	if err != nil {
		return "", err
	}
	paths := strings.Split(strings.TrimSpace(out), "\n")
	if len(paths) != len(names) {
		return "", fmt.Errorf(cleanupPrefix+"git rev-parse --git-path in %s printed %d line(s) for %d paths", wt, len(paths), len(names))
	}
	for i, path := range paths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(wt, path)
		}
		_, err := os.Stat(path)
		switch {
		case err == nil:
			return whats[i], nil
		case !errors.Is(err, fs.ErrNotExist):
			return "", fmt.Errorf(cleanupPrefix+"check for %s in progress in %s: %w", whats[i], wt, err)
		}
	}
	return "", nil
}

// cleanupGit runs git in dir for the cleanup through boundedGit, with ralph's
// environment, in ralph's process group, so an interrupt reaches git as it
// reaches ralph. The error is for a git that could not run, ran past
// regateGitTimeout, or a signal ended, and for ctx done; a non-zero exit is
// not one.
func cleanupGit(ctx context.Context, dir string, args ...string) (gitResult, error) {
	r, timedOut, err := boundedGit(ctx, cleanupPrefix, dir, nil, false, args...)
	if err == nil && timedOut {
		err = fmt.Errorf(cleanupPrefix+"git %s in %s ran past %s and was stopped", strings.Join(args, " "), dir, regateGitTimeout)
	}
	return r, err
}

// cleanupGitOutput is cleanupGit with a non-zero exit as an error naming
// git's stderr, and returns stdout.
func cleanupGitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	r, err := cleanupGit(ctx, dir, args...)
	if err != nil {
		return "", err
	}
	if r.code != 0 {
		return "", fmt.Errorf(cleanupPrefix+"git %s in %s: exit %d: %s", strings.Join(args, " "), dir, r.code, r.stderr)
	}
	return r.stdout, nil
}

// cleanupShellWord is s as one shell word in a Manual command: as it is when
// cleanupPlainWord matches it, else in single quotes (shellQuote).
func cleanupShellWord(s string) string {
	if cleanupPlainWord.MatchString(s) {
		return s
	}
	return shellQuote(s)
}
