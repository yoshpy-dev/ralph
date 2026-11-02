package org

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/gitroot"
)

// The cleanup's refusal text and the commands it names, and the checks that read no git; cleanup.go has the rest.

// shellPlainWord matches a word that no shell reads specially anywhere in
// it, which ShellWord leaves unquoted.
var shellPlainWord = regexp.MustCompile(`^[A-Za-z0-9_./:@+,-]+$`)

// cleanupAgain is the last step of a refusal that the cleanup can go on from
// once its cause is gone: run p's cleanupCommand again.
func cleanupAgain(p CleanupParams) string {
	return "run " + cleanupCommand(p) + " again"
}

// cleanupCommand is the ralph org cleanup command of p's branch that a
// refusal or a failure names (orgRetryCommand), with p.HintStateDir.
func cleanupCommand(p CleanupParams) string {
	return orgRetryCommand("cleanup", p.Branch, p.HintStateDir)
}

// orgRetryCommand is `ralph org <verb> --branch <branch>`, followed by
// --state-dir <hintStateDir> when hintStateDir is not "" (VerbCommand): the
// command a refusal or a failure of ralph org cleanup (verb cleanup) or ralph
// org merge (verb merge) names to run again, which then reads the ledger the
// first one read (CleanupParams.HintStateDir, MergeParams.HintStateDir). The
// branch and the dir are each one shell word (ShellWord).
func orgRetryCommand(verb, branch, hintStateDir string) string {
	return VerbCommand(verb+" --branch "+ShellWord(branch), hintStateDir)
}

// VerbCommand is `ralph org <verbAndArgs>`, followed by --state-dir
// <hintStateDir> when hintStateDir is not "", the dir one shell word
// (ShellWord); verbAndArgs is written as it is. It builds every ralph org
// command that ralph org merge and ralph org cleanup print with
// MergeParams.HintStateDir or CleanupParams.HintStateDir, which the CLI sets
// only when it was given --state-dir: the ones to run again
// (orgRetryCommand), and those of the verbs that read or change the same
// ledger, such as ralph org merge-order --all, ralph org inbox, and ralph org
// status --org-id <org_id>. A ledger [org].state_dir names gets no
// --state-dir, since the bare command reads the same file.
func VerbCommand(verbAndArgs, hintStateDir string) string {
	command := "ralph org " + verbAndArgs
	if hintStateDir != "" {
		command += " --state-dir " + ShellWord(hintStateDir)
	}
	return command
}

// scriptCmd is the Manual of a record org-<org_id> on the binding's branch,
// whether it is the binding's or only its path or kind differs: the script's
// cleanup, run in the main worktree, which removes the record's worktree and
// deletes that branch.
func (c *cleanupRun) scriptCmd() string {
	return "cd " + ShellWord(c.root) + " && " + featureWorktreeCleanup(c.orgID) + " --force-branch"
}

// gitCmd is `git -C <main worktree> args...` as a shell command
// (cleanupGitCmd).
func (c *cleanupRun) gitCmd(args ...string) string {
	return cleanupGitCmd(c.root, args...)
}

// cleanupGitCmd is `git -C <dir> args...` as a shell command, for a Manual
// and for a command a refusal's reason names: each word is ShellWord,
// so the command runs as it is written whatever the path holds.
func cleanupGitCmd(dir string, args ...string) string {
	words := []string{"git", "-C", ShellWord(dir)}
	for _, a := range args {
		words = append(words, ShellWord(a))
	}
	return strings.Join(words, " ")
}

// ledgerMoveStep is the next step of a refusal of a ledger whose directory is
// inside a worktree a cleanup would remove: the binding is in this ledger, so
// no other ledger can clean the branch up, and the way on is to move this
// one, while nothing uses it, to a place where no other ledger is, which it
// would otherwise overwrite, and to clean up again with it there. Nothing
// uses it only once no org on it runs: a running org's leader and director
// name the ledger's path in their commands, and its lock and manifest
// writes make the directory again, so a ledger moved between two commands
// would come back at the old path. The new place is named the two ways
// ResolveOrgStateDir reads one: --state-dir, with <new place> for the reader
// to fill in, or [org].state_dir in the personal file with the bare command.
// Neither repeats CleanupParams.HintStateDir, which names the old place.
func (c *cleanupRun) ledgerMoveStep() string {
	command := orgRetryCommand("cleanup", c.p.Branch, "")
	return fmt.Sprintf("once no org on the ledger runs and no ralph org command runs on it, move its directory to a place "+
		"outside the worktree where no other ledger is, and run %s --state-dir <new place> again, or set [org].state_dir in %s "+
		"to the new place and run %s again", command, config.LocalConfigFileName, command)
}

// pathWithin reports whether child is dir or inside it, compared by file
// identity: with both made absolute and their symlinks resolved
// (resolvedOrClean), names that say so match without a stat, and otherwise
// child and each of its ancestors in turn are compared with dir by
// os.SameFile, so a spelling the file system does not tell apart from dir's,
// such as another case of a letter or another Unicode normalization on
// macOS's default APFS volume, matches too (pathWithinDir compares the names
// alone). A dir that cannot be stat'd, such as a worktree whose directory is
// gone, holds nothing, so that is no match. A stat of child or of an ancestor
// that fails ends the walk with no match. That is safe for the callers, whose
// child exists: the ledger's directory, whose manifest check 1 read, and the
// working directory. A stat needs search permission on the ancestors only, so
// every ancestor of a child that can be stat'd can be stat'd too, and the walk
// ends that way only for a child removed or moved meanwhile.
func pathWithin(child, dir string) bool {
	child, dir = resolvedOrClean(mustAbs(child)), resolvedOrClean(mustAbs(dir))
	if child == dir || strings.HasPrefix(child, dir+string(filepath.Separator)) {
		return true
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return false
	}
	for p := child; ; p = filepath.Dir(p) {
		info, err := os.Stat(p)
		if err != nil {
			return false
		}
		if os.SameFile(info, dirInfo) {
			return true
		}
		if filepath.Dir(p) == p {
			return false
		}
	}
}

// recordGone is the refusal when the record org-<org_id> does not exist
// (D2's resume): what is left cannot be told from what another feature
// left, so nothing is removed. Manual removes by hand what git still has of
// the binding's worktree and branch: the worktree, or its registration when
// its directory is gone (git worktree remove takes a missing directory too),
// and the branch, whose tip the reason says to check first. When git's
// registration of the worktree is not on the binding's branch with that
// branch there, the reason also says what its HEAD is on and how to keep it
// before git worktree remove drops it, as checkRegistration's does
// (registrationAtRisk). nothingLeft reports that git has neither
// (CleanupResult.NothingLeft), with no Manual.
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
	reason = gone + fmt.Sprintf(", so ralph cannot tell that what is left is this feature's and removes nothing: %s left; ",
		strings.Join(left, " and "))
	if why, keep := c.registrationAtRisk(st); why != "" {
		reason += fmt.Sprintf("git still has the feature worktree's registration, %s: %s, and ", why, keep)
	}
	reason += "remove what is left by hand"
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
// or kind differs, Manual is the script's cleanup, to run once the record,
// the branch's commits the PR did not merge, and the worktree at the
// record's path, which the script removes, have been checked. When that
// worktree holds the ledger's directory (pathWithin), which checkLedger,
// comparing with the binding's worktree, does not see, the script would
// remove the ledger with it, so there is no Manual, and the reason names the
// move of the ledger first (ledgerMoveStep), after which the next cleanup
// names the script's cleanup.
func (c *cleanupRun) checkRecord(rec WorktreeRecord) (string, string) {
	var diffs []string
	if rec.Branch != c.f.Branch {
		diffs = append(diffs, fmt.Sprintf("branch %q, not %q", rec.Branch, c.f.Branch))
	}
	if !gitroot.SamePath(rec.WorktreePath, c.f.Worktree) {
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
	if dir := c.ledgerDir(); pathWithin(dir, rec.WorktreePath) {
		return reason + fmt.Sprintf(": the worktree at the record's path %s holds the ledger's directory %s, so cleaning it up "+
			"by hand would remove the ledger with it: check the record (%s state-path %s names its file), then, %s",
			rec.WorktreePath, dir, worktreeScriptRel, c.id(), c.ledgerMoveStep()), ""
	}
	return reason + fmt.Sprintf(": check the record (%s state-path %s names its file), and clean up by hand only once it is "+
		"this feature's, the branch holds nothing that was not merged (%s), and the worktree at the record's path holds nothing "+
		"you need (%s)", worktreeScriptRel, c.id(), c.gitCmd("log", c.p.Head+".."+c.f.Branch),
		cleanupGitCmd(rec.WorktreePath, "status")), c.scriptCmd()
}

// branchMovedSteps are the steps of check 5's refusal of a branch that moved
// from the head the PR merged to tip: look at the commits the PR did not
// merge, keep what is needed on a new branch made in the main worktree's
// repository, which leaves the feature worktree where it is, so that reset
// still moves the branch and not the one kept, put the branch back at the
// head with reset, and clean up again. The new branch is keptBranch of tip.
// It is made from the branch, not from tip, so that it also keeps a commit
// made on the branch after the refusal, which reset would otherwise take off.
func (c *cleanupRun) branchMovedSteps(tip, reset string) string {
	return fmt.Sprintf("look at them (%s), keep what you need on another branch made from the main worktree, not by switching "+
		"branches in the feature worktree (%s), put the branch back at the head its PR merged (%s), and %s",
		c.gitCmd("log", c.p.Head+".."+c.f.Branch), c.gitCmd("branch", c.keptBranch(tip), c.f.Branch), reset,
		cleanupAgain(c.p))
}

// keptBranch is the branch a refusal names to keep a commit at tip on:
// <branch>-unmerged-<the first 7 characters of tip>. A branch of that name
// already there was kept from the same tip, so git branch failing on it loses
// nothing, and another tip, such as that of a branch that moved again, gets
// another name.
func (c *cleanupRun) keptBranch(tip string) string {
	return c.f.Branch + "-unmerged-" + tip[:min(len(tip), 7)]
}

// fetchHeadCmd is the command that fetches the head the PR merged from
// origin into the main worktree's repository: refs/pull/<n>/head, which
// GitHub keeps after the merge even once the branch is deleted, when the PR's
// number is known, else the head by its SHA.
func (c *cleanupRun) fetchHeadCmd() string {
	if c.p.PRNumber > 0 {
		return c.gitCmd("fetch", "origin", "refs/pull/"+strconv.Itoa(c.p.PRNumber)+"/head")
	}
	return c.gitCmd("fetch", "origin", c.p.Head)
}

// checkRegistration is D2's check 6 of git's registration of the feature
// worktree, whose directory is gone (st, readState): it is on the binding's
// branch, and that branch is there, so its HEAD is the branch's tip, which
// check 5 compares next with the head the PR merged. Otherwise nothing is
// removed, since git worktree remove would drop the registration's HEAD and
// reflog, which may be the last references to a commit made there on a
// detached HEAD or another branch: the reason says what its HEAD is on, what
// removing it would drop, and how to keep that (registrationAtRisk, whose
// sentence recordGone gives too). The registration stays as it is, with no
// directory to change it from, so the next cleanup would refuse it again:
// Manual is the removal by hand, git worktree remove of the registration and
// the script's cleanup, as remove's when that git fails.
func (c *cleanupRun) checkRegistration(st cleanupState) (reason, manual string) {
	why, keep := c.registrationAtRisk(st)
	if why == "" {
		return "", ""
	}
	reason = fmt.Sprintf("the directory of the feature worktree %s is gone, but git still has its registration, %s, so nothing "+
		"is removed: %s", c.f.Worktree, why, keep)
	if st.branchTip != "" {
		reason += fmt.Sprintf(", check that the branch %s holds nothing that was not merged (%s)", c.f.Branch,
			c.gitCmd("log", c.p.Head+".."+c.f.Branch))
	}
	reason += ", then clean up by hand: the registration stays as it is, so the next cleanup would refuse it again"
	return reason, c.gitCmd("worktree", "remove", c.f.Worktree) + " && " + c.scriptCmd()
}

// registrationAtRisk is what a refusal says of git's registration of the
// feature worktree (st, readState), with its directory there or gone, when
// the registration is not on the binding's branch with that branch there:
// checkRegistration refuses it, and recordGone's Manual removes it with git
// worktree remove. why starts with "whose HEAD" and says what that HEAD is
// on and what removing the registration would drop; keep says how to keep
// that. Both are "" when git has no registration at the worktree's path, or
// it is on the binding's branch and that branch is there.
//
// Only a detached HEAD is kept on a new branch (keptBranch). A HEAD on
// another branch is that branch's tip, which the cleanup leaves, so what goes
// is the reflog only, and a HEAD that names no commit (zeros: its branch is
// gone, or git cannot read it) cannot be kept by a command, since the commit
// is only in the reflog, which git reflog and git log -g do not read while
// the HEAD names no commit. A HEAD on no branch git can read is one of the
// last kind: git prints the HEAD of such a registration as zeros, so it never
// reaches keptBranch. For both of the last two, keep says where that reflog
// is: the directory under worktrees/ in git's common directory whose gitdir
// file holds the worktree's .git, since git names that directory after the
// path's last part and may change or number the name.
func (c *cleanupRun) registrationAtRisk(st cleanupState) (why, keep string) {
	var on string
	switch {
	case !st.registered:
		return "", ""
	case st.regDetached:
		on = "is detached, not on the branch " + c.f.Branch
	case st.regBranch == "":
		on = "is on no branch git can read"
	case st.regBranch != c.f.Branch:
		on = "is on the branch " + st.regBranch + ", not " + c.f.Branch
	case st.branchTip == "":
		on = "is on the branch " + c.f.Branch + ", which no longer exists"
	default:
		return "", ""
	}
	why = fmt.Sprintf("whose HEAD %s %s", st.regHead, on)
	reflog := fmt.Sprintf("keep what you need on another branch made from the main worktree once you have found it in that "+
		"reflog (logs/HEAD of the directory under worktrees/ whose gitdir file holds %s, in git's common directory, which %s "+
		"prints)", filepath.Join(c.f.Worktree, ".git"), c.gitCmd("rev-parse", "--path-format=absolute", "--git-common-dir"))
	switch {
	case !gitObjectPattern.MatchString(st.regHead) || strings.Trim(st.regHead, "0") == "":
		return why + ": that HEAD names no commit, and removing the registration would drop its reflog, which may be the last " +
			"record of a commit made there", reflog
	case !st.regDetached && st.regBranch != "" && st.regBranch != c.f.Branch:
		return why + fmt.Sprintf(": that HEAD is the tip of the branch %s, which the cleanup leaves, and removing the "+
			"registration would drop only its reflog, which may be the last record of a commit made there", st.regBranch), reflog
	}
	keep = fmt.Sprintf("keep what you need on another branch made from the main worktree (%s)",
		c.gitCmd("branch", c.keptBranch(st.regHead), st.regHead))
	return why + ": removing the registration would drop its HEAD and reflog, which may be the last references to a commit " +
		"made there", keep
}

// cleanupDetails is worktree_cleaned's Details (EventWorktreeCleaned): the
// worktree last and strconv.Quote'd, since a path may hold spaces.
func cleanupDetails(branch, head, via, worktree string) string {
	return "branch=" + branch + " head=" + head + " via=" + via + " worktree=" + strconv.Quote(worktree)
}

// ShellWord is s as one shell word in a printed command: as it is when
// shellPlainWord matches it, else in single quotes (shellQuote). It is the one
// function internal/org and internal/cli both use to write a value as one shell
// word in a command they print: a Manual command, a command a refusal names to
// run again, and the hints of the CLI's own messages.
func ShellWord(s string) string {
	if shellPlainWord.MatchString(s) {
		return s
	}
	return shellQuote(s)
}
