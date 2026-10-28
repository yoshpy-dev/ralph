package org

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// The cleanup of a merged feature org's worktree (cleanup.go), S1 of plan
// docs/plans/active/2026-10-28-org-merge-cleanup.md: AC3, AC3b and AC4 at the
// org layer. worktreeCleanupFixture runs the real scripts/ralph-worktree.sh and git;
// worktreeCleanupFakeFixture runs neither, for the refusals at the ledger and the
// record, which come before any git.

const (
	cleanupTestBranch = "feat/auth"
	cleanupTestID     = "org-auth"
)

// worktreeCleanupFixture is a repo of worktreeScriptRepo with a bare origin, the
// feature worktree org-auth on feat/auth that the script's ensure made, with
// one commit pushed to origin's feat/auth (head, the head its PR merges) and
// a .gitignore that names ignored/, and a ledger whose binding of feat/auth
// to the org auth was disbanded the way Disband leaves it. The Org reads the
// worktrees through spyWorktrees.
type worktreeCleanupFixture struct {
	o        *Org
	spy      *spyWorktrees
	root     string // the main worktree, symlinks resolved
	origin   string
	stateDir string
	worktree string
	head     string
}

func newWorktreeCleanupFixture(t *testing.T) *worktreeCleanupFixture {
	t.Helper()
	root := worktreeScriptRepo(t)
	tmp := resolvedOrClean(t.TempDir())
	fx := &worktreeCleanupFixture{root: root, origin: filepath.Join(tmp, "origin.git"), stateDir: filepath.Join(tmp, "state")}
	runGit(t, "init", "--quiet", "--bare", "-b", "main", fx.origin)
	runGit(t, "-C", root, "remote", "add", "origin", fx.origin)
	runGit(t, "-C", root, "push", "--quiet", "-u", "origin", "main")
	out, err := runWorktreeScript(root, "ensure", "--id", cleanupTestID, "--kind", featureWorktreeKind, "--branch", cleanupTestBranch,
		"--path", featureWorktreeRelPath("auth"), "--canonical-ref", "split:/plans/auth-split.md#auth", "--cleanup-policy", "manual")
	if err != nil {
		t.Fatal(err)
	}
	fx.worktree = strings.TrimSpace(out)
	if want := filepath.Join(root, ".claude", "worktrees", cleanupTestID); fx.worktree != want {
		t.Fatalf("ensure made %q, want %s", fx.worktree, want)
	}
	writeGateFile(t, fx.worktree, ".gitignore", ".claude/worktrees/\nignored/\n", 0o644)
	writeGateFile(t, fx.worktree, "internal/auth/auth.go", "package auth\n", 0o644)
	runGit(t, "-C", fx.worktree, "add", ".")
	gateCommit(t, fx.worktree, "the feature")
	runGit(t, "-C", fx.worktree, "push", "--quiet", "-u", "origin", cleanupTestBranch)
	fx.head = gitOut(t, fx.worktree, "rev-parse", "HEAD")

	f := testFeature("split-1", "auth", "0123456789ab")
	f.Worktree = fx.worktree
	fx.o = &Org{Manifest: NewManifestStoreAtPath(ManifestPathIn(fx.stateDir)), Now: func() time.Time { return regateTestClock }}
	fx.spy = &spyWorktrees{}
	fx.o.Worktrees = fx.spy
	appendCleanupEvents(t, fx.o, boundReserveEvent("auth", f, "internal/auth/"))
	seedSeat(t, fx.o, "auth", LeaderIdentity, LeaderIdentity, "pane-1")
	appendCleanupEvents(t, fx.o,
		ManifestEvent{TS: fx.o.now(), OrgID: "auth", SeatID: LeaderIdentity, Event: EventStopped},
		ManifestEvent{TS: fx.o.now(), OrgID: "auth", Event: EventDisbanded})
	return fx
}

// appendCleanupEvents appends events to o's manifest.
func appendCleanupEvents(t *testing.T, o *Org, events ...ManifestEvent) {
	t.Helper()
	for _, ev := range events {
		if err := o.Manifest.Append(ev); err != nil {
			t.Fatal(err)
		}
	}
}

// params is the cleanup of feat/auth at the fixture's head by ralph org
// cleanup, run in the main worktree.
func (fx *worktreeCleanupFixture) params() CleanupParams {
	return CleanupParams{Branch: cleanupTestBranch, Head: fx.head, RepoRoot: fx.root, Cwd: fx.root, Via: CleanupViaCleanup}
}

// cleanup is CleanupFeatureWorktree with p.
func (fx *worktreeCleanupFixture) cleanup(p CleanupParams) (CleanupResult, error) {
	return fx.o.CleanupFeatureWorktree(context.Background(), p)
}

// scriptCmd is the Manual of a refusal whose record is the binding's.
func (fx *worktreeCleanupFixture) scriptCmd() string {
	return "cd " + fx.root + " && ./scripts/ralph-worktree.sh cleanup --id org-auth --force-branch"
}

// recordFile is the path of the record org-auth, which the script prints.
func (fx *worktreeCleanupFixture) recordFile(t *testing.T) string {
	t.Helper()
	out, err := runWorktreeScript(fx.root, "state-path", cleanupTestID)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(out)
}

// branchTip is feat/auth's tip in the main worktree, "" when there is no such
// branch.
func (fx *worktreeCleanupFixture) branchTip() string {
	out, err := exec.Command("git", "-C", fx.root, "rev-parse", "--verify", "--quiet", "refs/heads/"+cleanupTestBranch).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// snapshot is what the cleanup may remove or record: the record's file,
// feat/auth's tip, whether the worktree's directory is there, git worktree
// list, and the manifest.
func (fx *worktreeCleanupFixture) snapshot(t *testing.T) string {
	t.Helper()
	_, statErr := os.Stat(fx.worktree)
	return strings.Join([]string{fileOrAbsent(t, fx.recordFile(t)), fx.branchTip(), fmt.Sprint(statErr == nil),
		gitOut(t, fx.root, "worktree", "list", "--porcelain"), fileOrAbsent(t, fx.o.Manifest.Path())}, "\n--\n")
}

// assertRefused checks that res is a refusal with nothing removed or
// recorded since before was taken: Refused holds want, on one line and with
// no prefix, Manual is manual, the script's cleanup was not run, and
// regate.lock is released.
func (fx *worktreeCleanupFixture) assertRefused(t *testing.T, res CleanupResult, err error, before, want, manual string) {
	t.Helper()
	if err != nil || res.Refused == "" || res.Failed != "" || res.Cleaned || res.Recorded {
		t.Fatalf("cleanup = %+v, %v; want a refusal", res, err)
	}
	assertCleanupRefusal(t, res, want, manual)
	if after := fx.snapshot(t); after != before {
		t.Errorf("a refused cleanup changed something:\n%s\n->\n%s", before, after)
	}
	if slices.Contains(fx.spy.calls, "cleanup:"+cleanupTestID) {
		t.Errorf("a refused cleanup ran the script's cleanup: calls %q", fx.spy.calls)
	}
	assertCleanupUnlocked(t, fx.stateDir)
}

// assertCleanupRefusal checks res's Refused and Manual: Refused holds want,
// holds no line break, does not start with cleanupPrefix, and suggests no gc
// --prune; Manual is manual.
func assertCleanupRefusal(t *testing.T, res CleanupResult, want, manual string) {
	t.Helper()
	if !strings.Contains(res.Refused, want) {
		t.Errorf("refusal %q does not say %q", res.Refused, want)
	}
	if strings.Contains(res.Refused, "\n") || strings.HasPrefix(res.Refused, cleanupPrefix) {
		t.Errorf("refusal %q is not one line without the prefix %q", res.Refused, cleanupPrefix)
	}
	if strings.Contains(res.Refused+res.Manual, "gc --prune") {
		t.Errorf("refusal %q or Manual %q suggests gc --prune", res.Refused, res.Manual)
	}
	if res.Manual != manual {
		t.Errorf("Manual = %q, want %q", res.Manual, manual)
	}
}

// assertCleanupUnlocked checks that regate.lock of stateDir is not held.
func assertCleanupUnlocked(t *testing.T, stateDir string) {
	t.Helper()
	if held, err := fileLockHeld(regateLockPathIn(stateDir)); err != nil || held {
		t.Errorf("regate.lock held = %v (%v) after the cleanup returned; want it released", held, err)
	}
}

// assertCleaned checks that res cleaned up feat/auth and recorded it,
// resumed from resumed: the worktree's directory, git's registration of it,
// the branch, and the record are gone, the script's cleanup ran once, and
// the manifest's last line is the one worktree_cleaned with via.
func (fx *worktreeCleanupFixture) assertCleaned(t *testing.T, res CleanupResult, err error, resumed, via string) {
	t.Helper()
	if err != nil || !res.Cleaned || !res.Recorded || res.Refused != "" || res.Failed != "" || res.Manual != "" ||
		res.Resumed != resumed || res.OrgID != "auth" || res.Worktree != fx.worktree || res.Branch != cleanupTestBranch ||
		res.Head != fx.head {
		t.Fatalf("cleanup = %+v, %v; want it cleaned (resumed %q) and recorded", res, err, resumed)
	}
	if _, err := os.Stat(fx.worktree); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the worktree %s is still there (%v)", fx.worktree, err)
	}
	if list := gitOut(t, fx.root, "worktree", "list", "--porcelain"); strings.Contains(list, fx.worktree) {
		t.Errorf("git still lists the worktree:\n%s", list)
	}
	if tip := fx.branchTip(); tip != "" {
		t.Errorf("the branch %s is still at %s", cleanupTestBranch, tip)
	}
	if record := fileOrAbsent(t, fx.recordFile(t)); record != "<absent>" {
		t.Errorf("the record is still there:\n%s", record)
	}
	if n := countCalls(fx.spy.calls, "cleanup:"+cleanupTestID); n != 1 {
		t.Errorf("the script's cleanup ran %d times (calls %q); want once", n, fx.spy.calls)
	}
	events := mustReadEvents(t, fx.o)
	want := ManifestEvent{TS: fx.o.now(), OrgID: "auth", Event: EventWorktreeCleaned,
		Details: "branch=feat/auth head=" + fx.head + " via=" + via + ` worktree="` + fx.worktree + `"`}
	if last := events[len(events)-1]; last != want || len(cleanedEvents(events)) != 1 {
		t.Errorf("manifest ends with %+v (%d worktree_cleaned); want the one %+v", last, len(cleanedEvents(events)), want)
	}
	assertCleanupUnlocked(t, fx.stateDir)
}

// cleanupMovedRefusal is check 5's refusal of feat/auth at tip, not at head,
// the head its PR merged, which git has in root, the main worktree; reset
// puts the branch back (checkBranchTip).
func cleanupMovedRefusal(root, tip, head, reset string) string {
	return "the local branch feat/auth is at " + tip + ", not at " + head + ", the head its PR merged, so it may hold commits " +
		"that were not merged, which git branch -D would delete: " + cleanupMovedSteps(root, head, reset)
}

// cleanupMovedSteps are the steps of check 5's refusal of feat/auth
// (branchMovedSteps), with root, head, and reset as cleanupMovedRefusal's:
// the branch that keeps what is needed is made from the main worktree, so
// the feature worktree stays where it is and reset moves feat/auth.
func cleanupMovedSteps(root, head, reset string) string {
	return "look at them (git -C " + root + " log " + head + "..feat/auth), keep what you need on another branch made from the " +
		"main worktree, not by switching branches in the feature worktree (git -C " + root + " branch feat/auth-unmerged " +
		"feat/auth), put the branch back at the head its PR merged (" + reset + "), and run ralph org cleanup --branch feat/auth " +
		"again"
}

// countCalls counts call in calls.
func countCalls(calls []string, call string) int {
	n := 0
	for _, c := range calls {
		if c == call {
			n++
		}
	}
	return n
}

// cleanedEvents are the worktree_cleaned events of events.
func cleanedEvents(events []ManifestEvent) []ManifestEvent {
	return slices.DeleteFunc(slices.Clone(events), func(ev ManifestEvent) bool { return ev.Event != EventWorktreeCleaned })
}

// squashMerge merges feat/auth into origin's main the way GitHub's squash
// merge does, then deletes origin's feat/auth, as a repository that deletes
// head branches does, and prunes it from the main worktree: main gets the
// feature's change in a commit of its own, so feat/auth is merged into
// neither main nor an upstream, and git branch -d would refuse it.
func (fx *worktreeCleanupFixture) squashMerge(t *testing.T) {
	t.Helper()
	runGit(t, "-C", fx.root, "merge", "--quiet", "--squash", cleanupTestBranch)
	gateCommit(t, fx.root, "the feature (#1)")
	runGit(t, "-C", fx.root, "push", "--quiet", "origin", "main")
	runGit(t, "-C", fx.root, "push", "--quiet", "origin", "--delete", cleanupTestBranch)
	runGit(t, "-C", fx.root, "fetch", "--quiet", "--prune", "origin")
	if err := exec.Command("git", "-C", fx.root, "merge-base", "--is-ancestor", cleanupTestBranch, "main").Run(); err == nil {
		t.Fatal("main contains feat/auth after the squash merge; want a branch git branch -d refuses")
	}
	if err := exec.Command("git", "-C", fx.root, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+cleanupTestBranch).Run(); err == nil {
		t.Fatal("refs/remotes/origin/feat/auth is still there after the prune")
	}
}

// TestCleanupFeatureWorktree_SquashMergedBranch covers AC1 and AC2 at the
// org layer, through the real scripts/ralph-worktree.sh: after a squash merge
// and the deletion of origin's branch, which git branch -d refuses, the
// cleanup removes the feature worktree, its registration, the branch, and the
// record with one run of the script's cleanup (--force-branch), and records
// the one worktree_cleaned, which makes no org run and changes no seat.
func TestCleanupFeatureWorktree_SquashMergedBranch(t *testing.T) {
	fx := newWorktreeCleanupFixture(t)
	fx.squashMerge(t)
	before := mustReadEvents(t, fx.o)

	res, err := fx.cleanup(fx.params())
	fx.assertCleaned(t, res, err, "", CleanupViaCleanup)
	if want := []string{"lookup:" + cleanupTestID, "cleanup:" + cleanupTestID}; !slices.Equal(fx.spy.calls, want) {
		t.Errorf("worktree calls = %q, want %q", fx.spy.calls, want)
	}
	after := mustReadEvents(t, fx.o)
	if running := RunningOrgs(after); len(running) != 0 {
		t.Errorf("RunningOrgs = %q after worktree_cleaned; want none", running)
	}
	if !slices.Equal(Roster(after, RosterOptions{}), Roster(before, RosterOptions{})) {
		t.Errorf("worktree_cleaned changed the roster: %+v -> %+v", Roster(before, RosterOptions{}), Roster(after, RosterOptions{}))
	}
	if orgs := orgsToDisband(after); len(orgs) != 0 || leaderActivityEventCount(after, "auth") != leaderActivityEventCount(before, "auth") {
		t.Errorf("worktree_cleaned made %q a target of disband --all or counted as leader activity", orgs)
	}
}

// TestCleanupFeatureWorktree_IgnoredFilesGoWithTheWorktree covers D6: files
// the worktree's .gitignore names do not stop the cleanup and are removed
// with the worktree.
func TestCleanupFeatureWorktree_IgnoredFilesGoWithTheWorktree(t *testing.T) {
	fx := newWorktreeCleanupFixture(t)
	writeGateFile(t, fx.worktree, "ignored/run.log", "a log the gate left\n", 0o644)
	if out := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); out != "" {
		t.Fatalf("status lists %q; want the ignored file left out", out)
	}
	res, err := fx.cleanup(fx.params())
	fx.assertCleaned(t, res, err, "", CleanupViaCleanup)
}

// TestCleanupFeatureWorktreeLocked_KeepsTheCallersLock covers the entry
// ralph org merge uses with regate.lock held: the cleanup writes the marker of
// the org, leaves the lock held and the marker in it, and records via=merge.
func TestCleanupFeatureWorktreeLocked_KeepsTheCallersLock(t *testing.T) {
	fx := newWorktreeCleanupFixture(t)
	lock, err := acquireHeldFileLock(fx.stateDir, regateLockFile, "regate", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.release() }()
	p := fx.params()
	p.Via = CleanupViaMerge
	res, err := fx.o.cleanupFeatureWorktreeLocked(context.Background(), lock, p)
	if held, heldErr := fileLockHeld(regateLockPathIn(fx.stateDir)); heldErr != nil || !held {
		t.Errorf("regate.lock held = %v (%v) after the cleanup returned; want the caller still holding it", held, heldErr)
	}
	if orgID, ok := readRegateMarker(regateLockPathIn(fx.stateDir)); !ok || orgID != "auth" {
		t.Errorf("marker = %q (%v); want auth's", orgID, ok)
	}
	if err := lock.release(); err != nil {
		t.Fatal(err)
	}
	fx.assertCleaned(t, res, err, "", CleanupViaMerge)
}

// TestCleanupFeatureWorktreeLocked_RefusalKeepsTheMarkerWhileTheLockIsHeld
// covers the same entry when a check after the first one refuses (an
// untracked file): nothing is removed or recorded, and while the caller still
// holds regate.lock its marker names the org the first check found, so a
// spawn of that org is refused until the caller releases the lock.
func TestCleanupFeatureWorktreeLocked_RefusalKeepsTheMarkerWhileTheLockIsHeld(t *testing.T) {
	fx := newWorktreeCleanupFixture(t)
	writeGateFile(t, fx.worktree, "stray.txt", "x\n", 0o644)
	lock, err := acquireHeldFileLock(fx.stateDir, regateLockFile, "regate", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.release() }()
	before := fx.snapshot(t)
	p := fx.params()
	p.Via = CleanupViaMerge
	res, err := fx.o.cleanupFeatureWorktreeLocked(context.Background(), lock, p)
	if err != nil || res.Refused == "" || res.Cleaned || res.Recorded || res.OrgID != "auth" {
		t.Fatalf("cleanup = %+v, %v; want a refusal past the first check", res, err)
	}
	assertCleanupRefusal(t, res, "has 1 uncommitted change(s) or untracked file(s) (?? stray.txt)", "")
	if after := fx.snapshot(t); after != before {
		t.Errorf("a refused cleanup changed something:\n%s\n->\n%s", before, after)
	}
	if held, heldErr := fileLockHeld(regateLockPathIn(fx.stateDir)); heldErr != nil || !held {
		t.Errorf("regate.lock held = %v (%v) after the cleanup returned; want the caller still holding it", held, heldErr)
	}
	if orgID, ok := readRegateMarker(regateLockPathIn(fx.stateDir)); !ok || orgID != "auth" {
		t.Errorf("marker = %q (%v) after the refusal; want auth's while the lock is held", orgID, ok)
	}
	if err := regateSpawnErr(fx.stateDir, "auth"); err == nil {
		t.Error("regateSpawnErr = nil while the caller holds the lock; want the spawn of auth refused")
	}
	if err := lock.release(); err != nil {
		t.Fatal(err)
	}
	if err := regateSpawnErr(fx.stateDir, "auth"); err != nil {
		t.Errorf("regateSpawnErr after the release = %v; want nil", err)
	}
}

// TestCleanupFeatureWorktree_RefusedInTheWorktree covers AC3's checks of the
// worktree and the branch (D2's 3 to 6) through the real script and git:
// each refuses with its reason, removes nothing, records nothing, and
// releases regate.lock. The next cleanup goes on from each once its cause is
// fixed, so none names a command by hand (M2 of
// docs/reports/self-review-2026-10-28-org-merge-cleanup.md: the script's
// cleanup --force-branch would skip the checks after the one that refused),
// and the refusal of a branch that moved names how to put it back for where
// it is checked out. Only a worktree left with no local branch, which ralph
// cannot go on from, names the script's cleanup; a directory that is not a
// worktree's root, which the script cannot remove, names none.
func TestCleanupFeatureWorktree_RefusedInTheWorktree(t *testing.T) {
	for _, tc := range []struct {
		name   string
		setup  func(t *testing.T, fx *worktreeCleanupFixture, p *CleanupParams)
		want   func(fx *worktreeCleanupFixture) string
		manual bool // the Manual is the script's cleanup; else none
	}{
		{"run in the worktree", func(_ *testing.T, fx *worktreeCleanupFixture, p *CleanupParams) { p.Cwd = fx.worktree },
			func(fx *worktreeCleanupFixture) string {
				return "ralph runs in " + fx.worktree + ", inside the feature worktree " + fx.worktree + ", which the cleanup removes"
			}, false},
		{"run below the worktree", func(_ *testing.T, fx *worktreeCleanupFixture, p *CleanupParams) {
			p.Cwd = filepath.Join(fx.worktree, "internal", "auth")
		}, func(fx *worktreeCleanupFixture) string {
			return "ralph runs in " + filepath.Join(fx.worktree, "internal", "auth") + ", inside the feature worktree"
		}, false},
		{"an uncommitted change", func(t *testing.T, fx *worktreeCleanupFixture, _ *CleanupParams) {
			writeGateFile(t, fx.worktree, "internal/auth/auth.go", "package auth // changed\n", 0o644)
		}, func(fx *worktreeCleanupFixture) string {
			return "the feature worktree " + fx.worktree + " has 1 uncommitted change(s) or untracked file(s) (M internal/auth/auth.go)"
		}, false},
		{"untracked files", func(t *testing.T, fx *worktreeCleanupFixture, _ *CleanupParams) {
			for _, name := range []string{"a.txt", "b.txt", "c.txt", "d.txt"} {
				writeGateFile(t, fx.worktree, "notes/"+name, "x\n", 0o644)
			}
		}, func(fx *worktreeCleanupFixture) string {
			return "has 4 uncommitted change(s) or untracked file(s) (?? notes/a.txt, ?? notes/b.txt, ?? notes/c.txt, and 1 more)"
		}, false},
		{"a rebase in progress", func(t *testing.T, fx *worktreeCleanupFixture, _ *CleanupParams) {
			cmd := exec.Command("git", "-C", fx.worktree, "-c", "user.name=t", "-c", "user.email=t@example.com", "rebase", "--quiet",
				"--exec", "false", "HEAD~1")
			if out, err := cmd.CombinedOutput(); err == nil {
				t.Fatalf("git rebase --exec false succeeded; want it stopped:\n%s", out)
			}
		}, func(fx *worktreeCleanupFixture) string {
			return "a rebase is in progress in the feature worktree " + fx.worktree
		}, false},
		{"a merge in progress", func(t *testing.T, fx *worktreeCleanupFixture, _ *CleanupParams) {
			writeGateFile(t, fx.root, "docs/other.md", "another feature\n", 0o644)
			runGit(t, "-C", fx.root, "add", ".")
			gateCommit(t, fx.root, "another feature")
			runGit(t, "-C", fx.worktree, "-c", "user.name=t", "-c", "user.email=t@example.com", "merge", "--quiet", "--no-commit",
				"--no-ff", "main")
		}, func(fx *worktreeCleanupFixture) string {
			return "a merge is in progress in the feature worktree " + fx.worktree
		}, false},
		{"the branch moved past the merged head", func(t *testing.T, fx *worktreeCleanupFixture, _ *CleanupParams) {
			gateCommit(t, fx.worktree, "not merged")
		}, func(fx *worktreeCleanupFixture) string {
			return cleanupMovedRefusal(fx.root, fx.branchTip(), fx.head, "git -C "+fx.worktree+" reset --hard "+fx.head)
		}, false},
		{"the branch moved, another branch checked out", func(t *testing.T, fx *worktreeCleanupFixture, _ *CleanupParams) {
			gateCommit(t, fx.worktree, "not merged")
			runGit(t, "-C", fx.worktree, "switch", "--quiet", "-c", "other")
		}, func(fx *worktreeCleanupFixture) string {
			return "put the branch back at the head its PR merged (git -C " + fx.root + " branch -f feat/auth " + fx.head + "), and " +
				"run ralph org cleanup --branch feat/auth again"
		}, false},
		{"a detached HEAD with a commit", func(t *testing.T, fx *worktreeCleanupFixture, _ *CleanupParams) {
			runGit(t, "-C", fx.worktree, "checkout", "--quiet", "--detach")
			gateCommit(t, fx.worktree, "on a detached HEAD")
		}, func(fx *worktreeCleanupFixture) string {
			return "HEAD is detached in the feature worktree " + fx.worktree + ", not on the branch feat/auth"
		}, false},
		{"another branch", func(t *testing.T, fx *worktreeCleanupFixture, _ *CleanupParams) {
			runGit(t, "-C", fx.worktree, "switch", "--quiet", "-c", "other")
		}, func(fx *worktreeCleanupFixture) string {
			return "the feature worktree " + fx.worktree + " has the branch other checked out, not feat/auth"
		}, false},
		{"the local branch gone, the worktree detached", func(t *testing.T, fx *worktreeCleanupFixture, _ *CleanupParams) {
			runGit(t, "-C", fx.worktree, "checkout", "--quiet", "--detach")
			runGit(t, "-C", fx.root, "branch", "--quiet", "-D", cleanupTestBranch)
		}, func(fx *worktreeCleanupFixture) string {
			return "the local branch feat/auth does not exist in " + fx.root + ", though its feature worktree " + fx.worktree +
				" does: check what the worktree has checked out (git -C " + fx.worktree + " status)"
		}, true},
		{"not a worktree's root", func(t *testing.T, fx *worktreeCleanupFixture, _ *CleanupParams) {
			if err := os.Remove(filepath.Join(fx.worktree, ".git")); err != nil {
				t.Fatal(err)
			}
		}, func(fx *worktreeCleanupFixture) string {
			return "the feature worktree " + fx.worktree + " is not the root of a git worktree (git rev-parse --show-toplevel: " +
				fx.root + "), so the script cannot remove it"
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newWorktreeCleanupFixture(t)
			p := fx.params()
			tc.setup(t, fx, &p)
			before := fx.snapshot(t)
			want := tc.want(fx)
			res, err := fx.cleanup(p)
			manual := ""
			if tc.manual {
				manual = fx.scriptCmd()
			}
			fx.assertRefused(t, res, err, before, want, manual)
			if res.OrgID != "auth" || res.Worktree != fx.worktree {
				t.Errorf("result names org %q and worktree %q; want auth and %s", res.OrgID, res.Worktree, fx.worktree)
			}
		})
	}
}

// TestCleanupFeatureWorktree_HeadNotHereNamesItsFetch covers R2-L3 of
// docs/reports/self-review-2026-10-28-org-merge-cleanup.md: the head ralph
// org cleanup reads from gh (headRefOid) can be a commit made on GitHub that
// this repository does not have, so check 5 refuses the local branch, at that
// head's parent, with a reason that first names how to fetch the head:
// refs/pull/<n>/head, which GitHub keeps after a merge even once the branch
// is deleted, when the PR's number is known, else the head by its SHA. Origin
// has the head only under refs/pull/7/head, as after a merge that deleted the
// branch. With the reason's commands run with sh as it writes them, the next
// cleanup cleans up at that head, and the branch kept is where the local
// branch was.
func TestCleanupFeatureWorktree_HeadNotHereNamesItsFetch(t *testing.T) {
	for _, tc := range []struct {
		name  string
		pr    int
		fetch func(fx *worktreeCleanupFixture, head string) string
	}{
		{"the PR's number known", 7, func(fx *worktreeCleanupFixture, _ string) string {
			return "git -C " + fx.root + " fetch origin refs/pull/7/head"
		}},
		{"the PR's number not known", 0, func(fx *worktreeCleanupFixture, head string) string {
			return "git -C " + fx.root + " fetch origin " + head
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newWorktreeCleanupFixture(t)
			local := fx.head
			head := gitOut(t, fx.origin, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit-tree", "-p", local, "-m",
				"update the branch, on GitHub", local+"^{tree}")
			runGit(t, "-C", fx.origin, "update-ref", "refs/pull/7/head", head)
			runGit(t, "-C", fx.origin, "update-ref", "-d", "refs/heads/"+cleanupTestBranch)
			if err := exec.Command("git", "-C", fx.root, "cat-file", "-e", head+"^{commit}").Run(); err == nil {
				t.Fatalf("the main worktree has %s; want a head only origin has", head)
			}
			p := fx.params()
			p.Head, p.PRNumber = head, tc.pr
			before := fx.snapshot(t)
			res, err := fx.cleanup(p)
			fetch, reset := tc.fetch(fx, head), "git -C "+fx.worktree+" reset --hard "+head
			fx.assertRefused(t, res, err, before, "the local branch feat/auth is at "+local+", not at "+head+", the head its PR "+
				"merged, which git does not have in "+fx.root+" (a commit made on GitHub or pushed from another clone), so whether "+
				"the branch holds commits that were not merged cannot be told yet: fetch the head ("+fetch+"), then "+
				cleanupMovedSteps(fx.root, head, reset), "")
			for _, cmd := range []string{fetch, "git -C " + fx.root + " log " + head + "..feat/auth",
				"git -C " + fx.root + " branch feat/auth-unmerged feat/auth", reset} {
				if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
					t.Fatalf("%s: %v\n%s", cmd, err, out)
				}
			}
			fx.head = head
			res, err = fx.cleanup(p)
			fx.assertCleaned(t, res, err, "", CleanupViaCleanup)
			if kept := gitOut(t, fx.root, "rev-parse", "refs/heads/feat/auth-unmerged"); kept != local {
				t.Errorf("feat/auth-unmerged is at %s; want %s, where the local branch was", kept, local)
			}
		})
	}
}

// TestCleanupFeatureWorktree_ResumesWhatAnEarlierCleanupLeft covers AC3b
// (D2's resume): from what a script that stopped part way leaves, the next
// cleanup removes the rest and records it, with the checks of what is left.
// Git's registration of a missing worktree is removed alone, so another
// missing worktree's stays; a branch that moved past the merged head is not
// deleted, and once it is put back as the refusal says, the next cleanup goes
// on.
func TestCleanupFeatureWorktree_ResumesWhatAnEarlierCleanupLeft(t *testing.T) {
	t.Run("the worktree removed, the branch and the record left", func(t *testing.T) {
		fx := newWorktreeCleanupFixture(t)
		fx.squashMerge(t)
		runGit(t, "-C", fx.root, "worktree", "remove", fx.worktree)
		res, err := fx.cleanup(fx.params())
		fx.assertCleaned(t, res, err, CleanupResumedWorktreeGone, CleanupViaCleanup)
	})
	t.Run("the directory deleted, its registration left", func(t *testing.T) {
		fx := newWorktreeCleanupFixture(t)
		other := filepath.Join(resolvedOrClean(t.TempDir()), "other")
		runGit(t, "-C", fx.root, "worktree", "add", "--quiet", "-b", "other", other)
		for _, dir := range []string{other, fx.worktree} {
			if err := os.RemoveAll(dir); err != nil {
				t.Fatal(err)
			}
		}
		res, err := fx.cleanup(fx.params())
		fx.assertCleaned(t, res, err, CleanupResumedWorktreeGone, CleanupViaCleanup)
		if list := gitOut(t, fx.root, "worktree", "list", "--porcelain"); !strings.Contains(list, "worktree "+other+"\n") {
			t.Errorf("the other missing worktree's registration is gone; want only %s's removed:\n%s", fx.worktree, list)
		}
	})
	t.Run("only the record left", func(t *testing.T) {
		fx := newWorktreeCleanupFixture(t)
		runGit(t, "-C", fx.root, "worktree", "remove", fx.worktree)
		runGit(t, "-C", fx.root, "branch", "--quiet", "-D", cleanupTestBranch)
		res, err := fx.cleanup(fx.params())
		fx.assertCleaned(t, res, err, CleanupResumedRecordOnly, CleanupViaCleanup)
	})
	// A branch that moved is refused with no command by hand; the reason
	// names how to keep what it holds and how to put the branch back, which
	// differs with git's registration of the missing worktree left (git
	// branch -f refuses a branch it has checked out), and once those
	// commands run as the reason writes them, the next cleanup goes on and
	// the branch kept is still where the branch had moved.
	for _, tc := range []struct {
		name   string
		remove func(t *testing.T, fx *worktreeCleanupFixture)
		reset  func(fx *worktreeCleanupFixture) string
	}{
		{"the worktree removed, the branch moved", func(t *testing.T, fx *worktreeCleanupFixture) {
			runGit(t, "-C", fx.root, "worktree", "remove", fx.worktree)
		}, func(fx *worktreeCleanupFixture) string {
			return "git -C " + fx.root + " branch -f feat/auth " + fx.head
		}},
		{"the directory deleted, its registration left, the branch moved", func(t *testing.T, fx *worktreeCleanupFixture) {
			if err := os.RemoveAll(fx.worktree); err != nil {
				t.Fatal(err)
			}
		}, func(fx *worktreeCleanupFixture) string {
			return "git -C " + fx.root + " update-ref refs/heads/feat/auth " + fx.head
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newWorktreeCleanupFixture(t)
			tc.remove(t, fx)
			runGit(t, "-C", fx.root, "update-ref", "refs/heads/"+cleanupTestBranch, "main")
			before := fx.snapshot(t)
			res, err := fx.cleanup(fx.params())
			reset, moved := tc.reset(fx), fx.branchTip()
			fx.assertRefused(t, res, err, before, cleanupMovedRefusal(fx.root, moved, fx.head, reset), "")
			if res.Resumed != "" {
				t.Errorf("Resumed = %q on a refusal; want none", res.Resumed)
			}
			for _, cmd := range []string{"git -C " + fx.root + " branch feat/auth-unmerged feat/auth", reset} {
				if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
					t.Fatalf("%s: %v\n%s", cmd, err, out)
				}
			}
			res, err = fx.cleanup(fx.params())
			fx.assertCleaned(t, res, err, CleanupResumedWorktreeGone, CleanupViaCleanup)
			if kept := gitOut(t, fx.root, "rev-parse", "refs/heads/feat/auth-unmerged"); kept != moved {
				t.Errorf("feat/auth-unmerged is at %s after the cleanup; want %s, where feat/auth had moved", kept, moved)
			}
		})
	}
}

// TestCleanupFeatureWorktree_RecordGoneRemovesNothing covers AC3b's missing
// record: nothing is removed, and Manual removes by hand what git still has
// of the worktree and the branch, the reason saying to check the branch
// first. Only with nothing left at all (no record, no worktree or
// registration of it, no branch) is NothingLeft set, with the refusal and no
// Manual (plan docs/plans/active/2026-10-28-org-merge-cleanup.md, S3).
func TestCleanupFeatureWorktree_RecordGoneRemovesNothing(t *testing.T) {
	for _, tc := range []struct {
		name        string
		setup       func(t *testing.T, fx *worktreeCleanupFixture)
		want        func(fx *worktreeCleanupFixture) string
		manual      func(fx *worktreeCleanupFixture) string
		nothingLeft bool
	}{
		{"the branch left", func(t *testing.T, fx *worktreeCleanupFixture) {
			runGit(t, "-C", fx.root, "worktree", "remove", fx.worktree)
		}, func(fx *worktreeCleanupFixture) string {
			return "the worktree record org-auth of the branch feat/auth does not exist, so ralph cannot tell that what is left " +
				"is this feature's and removes nothing: the branch feat/auth at " + fx.head + " left; remove it by hand once you " +
				"have checked that the branch holds nothing that was not merged (git -C " + fx.root + " log " + fx.head + "..feat/auth"
		}, func(fx *worktreeCleanupFixture) string { return "git -C " + fx.root + " branch -D feat/auth" }, false},
		{"the worktree and the branch left", nil, func(fx *worktreeCleanupFixture) string {
			return "removes nothing: the feature worktree " + fx.worktree + " and the branch feat/auth at " + fx.head + " left"
		}, func(fx *worktreeCleanupFixture) string {
			return "git -C " + fx.root + " worktree remove " + fx.worktree + " && git -C " + fx.root + " branch -D feat/auth"
		}, false},
		{"only the registration left", func(t *testing.T, fx *worktreeCleanupFixture) {
			if err := os.RemoveAll(fx.worktree); err != nil {
				t.Fatal(err)
			}
			// git branch -D refuses a branch a registered worktree has checked
			// out, even one whose directory is gone.
			runGit(t, "-C", fx.root, "update-ref", "-d", "refs/heads/"+cleanupTestBranch)
		}, func(fx *worktreeCleanupFixture) string {
			return "removes nothing: the feature worktree " + fx.worktree + " left; remove it by hand"
		}, func(fx *worktreeCleanupFixture) string {
			return "git -C " + fx.root + " worktree remove " + fx.worktree
		}, false},
		{"nothing left", func(t *testing.T, fx *worktreeCleanupFixture) {
			runGit(t, "-C", fx.root, "worktree", "remove", fx.worktree)
			runGit(t, "-C", fx.root, "branch", "--quiet", "-D", cleanupTestBranch)
		}, func(fx *worktreeCleanupFixture) string {
			return "the worktree record org-auth of the branch feat/auth does not exist, and neither its feature worktree " +
				fx.worktree + " nor the branch is left: nothing is left to clean up"
		}, func(*worktreeCleanupFixture) string { return "" }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newWorktreeCleanupFixture(t)
			if tc.setup != nil {
				tc.setup(t, fx)
			}
			if err := os.Remove(fx.recordFile(t)); err != nil {
				t.Fatal(err)
			}
			before := fx.snapshot(t)
			res, err := fx.cleanup(fx.params())
			fx.assertRefused(t, res, err, before, tc.want(fx), tc.manual(fx))
			if res.NothingLeft != tc.nothingLeft {
				t.Errorf("NothingLeft = %v, want %v", res.NothingLeft, tc.nothingLeft)
			}
		})
	}
}

// TestCleanupFeatureWorktree_ScriptFailureRecordsNothing covers a removal
// that fails once the checks passed: with the worktree locked, the script's
// git worktree remove refuses it, so the worktree, the branch, and the record
// stay, nothing is recorded, and Failed says so with the script's cleanup as
// the command by hand.
func TestCleanupFeatureWorktree_ScriptFailureRecordsNothing(t *testing.T) {
	fx := newWorktreeCleanupFixture(t)
	runGit(t, "-C", fx.root, "worktree", "lock", "--reason", "in use", fx.worktree)
	before := fx.snapshot(t)
	res, err := fx.cleanup(fx.params())
	if err != nil || res.Cleaned || res.Recorded || res.Refused != "" ||
		!strings.Contains(res.Failed, "scripts/ralph-worktree.sh cleanup --id org-auth --force-branch failed (") ||
		!strings.Contains(res.Failed, "locked") || res.Manual != fx.scriptCmd() {
		t.Fatalf("cleanup = %+v, %v; want Failed with the script's error and the script's cleanup as Manual", res, err)
	}
	if after := fx.snapshot(t); after != before {
		t.Errorf("a failed removal changed something:\n%s\n->\n%s", before, after)
	}
	assertCleanupUnlocked(t, fx.stateDir)
}

// TestCleanupFeatureWorktree_InterruptedBeforeTheRemoval covers ctx done once
// the checks passed: an error that says nothing was removed, with nothing
// removed or recorded.
func TestCleanupFeatureWorktree_InterruptedBeforeTheRemoval(t *testing.T) {
	fx := newWorktreeCleanupFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cleanupTestHooks.beforeScript = cancel
	t.Cleanup(func() { cleanupTestHooks.beforeScript = nil })
	before := fx.snapshot(t)
	res, err := fx.o.CleanupFeatureWorktree(ctx, fx.params())
	if err == nil || !strings.Contains(err.Error(), "org: cleanup: interrupted (context canceled) after the checks passed, before "+
		"anything was removed") || res.Cleaned || res.Recorded {
		t.Fatalf("cleanup = %+v, %v; want the interruption before the removal", res, err)
	}
	if after := fx.snapshot(t); after != before {
		t.Errorf("an interrupted cleanup changed something:\n%s\n->\n%s", before, after)
	}
}

// TestCleanupFeatureWorktree_MarkerRefusesSpawnOfTheOrg covers D2's 0 and
// AC4: while the cleanup runs, regate.lock holds the marker of the org, so
// regateSpawnErr, which spawn and start --plan run, refuses that org_id,
// and a spawn of it is a plain rejection with nothing recorded, while
// another org goes on. Once the cleanup returns, the org can be spawned
// again.
func TestCleanupFeatureWorktree_MarkerRefusesSpawnOfTheOrg(t *testing.T) {
	fx := newWorktreeCleanupFixture(t)
	o, h, a := testOrgIn(t, fx.stateDir)
	var sameErr, otherErr error
	var other SpawnResult
	reached := false
	cleanupTestHooks.beforeScript = func() {
		reached = true
		sameErr, otherErr = regateSpawnErr(fx.stateDir, "auth"), regateSpawnErr(fx.stateDir, "other")
		before, manifest := takeSpawnSnapshot(t, o, h, a), fileOrAbsent(t, o.Manifest.Path())
		assertRegateRefusedSpawn(t, o.Spawn(mustSpawnParams("auth", "worker-1")), "auth", fx.stateDir, o, h, a, before, manifest)
		other = o.Spawn(mustSpawnParams("other", "seat-1"))
	}
	t.Cleanup(func() { cleanupTestHooks.beforeScript = nil })

	res, err := fx.cleanup(fx.params())
	if err != nil || !res.Cleaned || !reached {
		t.Fatalf("cleanup = %+v, %v (hook reached %v); want it cleaned through the hook", res, err, reached)
	}
	want := `org: org_id "auth" is not spawned while ralph org regate runs on its branch or ralph org merge or cleanup removes ` +
		`its merged feature worktree (` + regateLockPathIn(fx.stateDir) + ` holds the marker of that org): wait for it to end and ` +
		`run the same command again`
	if sameErr == nil || sameErr.Error() != want || otherErr != nil {
		t.Errorf("regateSpawnErr during the cleanup = %v for auth, %v for other; want %q and nil", sameErr, otherErr, want)
	}
	if other.Outcome != SpawnOutcomeSpawned || other.Err != nil {
		t.Errorf("spawn of another org during the cleanup = %+v; want it spawned", other)
	}
	if err := regateSpawnErr(fx.stateDir, "auth"); err != nil {
		t.Errorf("regateSpawnErr after the cleanup = %v; want nil", err)
	}
}

// worktreeCleanupFakeFixture is the cleanup on a ledger whose binding of feat/auth to
// the org auth was disbanded, with fakeWorktrees holding the binding's record
// and a repo root that is no git repository: for the refusals at the ledger
// (D2's 1) and the record (D2's 2), which run no git.
type worktreeCleanupFakeFixture struct {
	o        *Org
	wt       *fakeWorktrees
	f        *FeatureBinding
	root     string
	stateDir string
}

func newWorktreeCleanupFakeFixture(t *testing.T) *worktreeCleanupFakeFixture {
	t.Helper()
	return newWorktreeCleanupFakeFixtureAt(t, resolvedOrClean(t.TempDir()))
}

// newWorktreeCleanupFakeFixtureAt is newWorktreeCleanupFakeFixture with the
// repo root root, whose feature worktree is root/.claude/worktrees/org-auth.
func newWorktreeCleanupFakeFixtureAt(t *testing.T, root string) *worktreeCleanupFakeFixture {
	t.Helper()
	fx := &worktreeCleanupFakeFixture{root: root, stateDir: filepath.Join(resolvedOrClean(t.TempDir()), "state")}
	fx.f = testFeature("split-1", "auth", "0123456789ab")
	fx.f.Worktree = filepath.Join(fx.root, ".claude", "worktrees", cleanupTestID)
	fx.o = &Org{Manifest: NewManifestStoreAtPath(ManifestPathIn(fx.stateDir)), Now: func() time.Time { return regateTestClock }}
	fx.wt = newFakeWorktrees()
	fx.wt.records[cleanupTestID] = WorktreeRecord{WorktreePath: fx.f.Worktree, Branch: cleanupTestBranch, Kind: featureWorktreeKind,
		CanonicalRef: "split:/plans/auth-split.md#auth"}
	fx.o.Worktrees = fx.wt
	appendCleanupEvents(t, fx.o, boundReserveEvent("auth", fx.f, "internal/auth/"))
	seedSeat(t, fx.o, "auth", LeaderIdentity, LeaderIdentity, "pane-1")
	appendCleanupEvents(t, fx.o,
		ManifestEvent{TS: fx.o.now(), OrgID: "auth", SeatID: LeaderIdentity, Event: EventStopped},
		ManifestEvent{TS: fx.o.now(), OrgID: "auth", Event: EventDisbanded})
	return fx
}

// cleanupTestHead is the head the fake fixture's PR merged.
const cleanupTestHead = "0123456789abcdef0123456789abcdef01234567"

func (fx *worktreeCleanupFakeFixture) params() CleanupParams {
	return CleanupParams{Branch: cleanupTestBranch, Head: cleanupTestHead, RepoRoot: fx.root, Cwd: fx.root, Via: CleanupViaCleanup}
}

// TestCleanupFeatureWorktree_RefusedAtTheLedgerAndTheRecord covers AC3's and
// AC4's refusals before any git: a branch not named or a head that is no
// commit SHA (before any lock), a branch no complete binding names (gate
// records only included), a binding whose org_id is not one ralph makes, an
// org that runs (for this feature or another), a record that cannot be read
// or that the script cannot be run for, and a record that is not the
// binding's. Each removes nothing, records nothing, runs no script, releases
// regate.lock, and leaves its marker naming the org only once the ledger's
// check passed; Manual is the command by hand when one applies.
func TestCleanupFeatureWorktree_RefusedAtTheLedgerAndTheRecord(t *testing.T) {
	gateOnly := ManifestEvent{OrgID: "gated", Event: EventGatePassed, Details: gateRecord{Head: cleanupTestHead, Base: cleanupTestHead,
		BaseRef: "refs/remotes/origin/main", Branch: "feat/gated", By: "alice", Log: "/state/gates/gated.log"}.details()}
	for _, tc := range []struct {
		name   string
		setup  func(t *testing.T, fx *worktreeCleanupFakeFixture, p *CleanupParams)
		want   string
		manual func(fx *worktreeCleanupFakeFixture) string // nil for none
		calls  []string
		marker bool // the ledger's check passed and wrote auth's marker
	}{
		{"no branch named", func(_ *testing.T, _ *worktreeCleanupFakeFixture, p *CleanupParams) { p.Branch = "" },
			"no branch named: name the merged feature branch to clean up", nil, nil, false},
		{"a head that is no commit SHA", func(_ *testing.T, _ *worktreeCleanupFakeFixture, p *CleanupParams) { p.Head = "0123abc" },
			`the head the PR of feat/auth merged is not known ("0123abc" is not a commit SHA)`, nil, nil, false},
		{"a branch no binding names", func(_ *testing.T, _ *worktreeCleanupFakeFixture, p *CleanupParams) { p.Branch = "feat/none" },
			"no complete feature binding in the ledger names the branch feat/none, so it has no feature worktree to clean up", nil, nil, false},
		{"a branch with gate records only", func(t *testing.T, fx *worktreeCleanupFakeFixture, p *CleanupParams) {
			appendCleanupEvents(t, fx.o, gateOnly)
			p.Branch = "feat/gated"
		}, "no complete feature binding in the ledger names the branch feat/gated", nil, nil, false},
		// L1 of docs/reports/self-review-2026-10-28-org-merge-cleanup.md: the
		// script would read --id org-Auth;x as org-auth-x, another org's
		// record.
		{"a binding whose org_id ralph does not make", func(t *testing.T, fx *worktreeCleanupFakeFixture, p *CleanupParams) {
			g := testFeature("split-1", "x", "0123456789ab")
			g.Worktree = filepath.Join(fx.root, ".claude", "worktrees", "org-Auth;x")
			appendCleanupEvents(t, fx.o, boundReserveEvent("Auth;x", g, "internal/x/"),
				ManifestEvent{TS: fx.o.now(), OrgID: "Auth;x", Event: EventDisbanded})
			p.Branch = g.Branch
		}, `the ledger's binding of the branch feat/x names org_id "Auth;x", which is not of the form ralph makes ` +
			`(^[a-z][a-z0-9-]{0,29}$), so which worktree record and org it names cannot be told and nothing is removed: check the ` +
			"binding in ", nil, nil, false},
		{"the org runs", func(t *testing.T, fx *worktreeCleanupFakeFixture, _ *CleanupParams) {
			seedSeat(t, fx.o, "auth", "worker", "worker", "pane-2")
		}, `the branch feat/auth is bound to org_id "auth", which is running and may still use its feature worktree`, nil, nil, false},
		{"the org runs another feature", func(t *testing.T, fx *worktreeCleanupFakeFixture, _ *CleanupParams) {
			g := *fx.f
			g.Feature, g.Branch = "auth-2", "feat/auth-2"
			appendCleanupEvents(t, fx.o, boundReserveEvent("auth", &g, "internal/auth/"))
		}, `the branch feat/auth is bound to org_id "auth", which is running, now for another feature, on the branch feat/auth-2`,
			nil, nil, false},
		{"a record that cannot be read", func(_ *testing.T, fx *worktreeCleanupFakeFixture, _ *CleanupParams) {
			fx.wt.lookupErr = errors.New("org: /x/org-auth.json is not a JSON record")
		}, "the worktree record org-auth cannot be read (org: /x/org-auth.json is not a JSON record), so nothing is removed", nil,
			[]string{"lookup:" + cleanupTestID}, true},
		{"no script to run", func(_ *testing.T, fx *worktreeCleanupFakeFixture, _ *CleanupParams) { fx.o.Worktrees = nil },
			" (ralph init ships it into the repo), so the worktree record org-auth cannot be read and nothing is removed: restore " +
				"the script and run ralph org cleanup --branch feat/auth again", nil, nil, true},
		{"a record on another branch", func(_ *testing.T, fx *worktreeCleanupFakeFixture, _ *CleanupParams) {
			rec := fx.wt.records[cleanupTestID]
			rec.Branch = "feat/auth-2"
			fx.wt.records[cleanupTestID] = rec
		}, `the worktree record org-auth records branch "feat/auth-2", not "feat/auth": it is not the record of the feature worktree ` +
			`org_id "auth" had for the branch feat/auth and may be another feature's on the same org_id, so nothing is removed: if ` +
			"the branch feat/auth is left in ",
			func(fx *worktreeCleanupFakeFixture) string {
				return "git -C " + cleanupShellWord(fx.root) + " branch -D feat/auth"
			},
			[]string{"lookup:" + cleanupTestID}, true},
		{"a record at another path", func(_ *testing.T, fx *worktreeCleanupFakeFixture, _ *CleanupParams) {
			rec := fx.wt.records[cleanupTestID]
			rec.WorktreePath = "/elsewhere/org-auth"
			fx.wt.records[cleanupTestID] = rec
		}, `the worktree record org-auth records worktree_path "/elsewhere/org-auth", not "`,
			func(fx *worktreeCleanupFakeFixture) string {
				return "cd " + cleanupShellWord(fx.root) + " && ./scripts/ralph-worktree.sh cleanup --id org-auth --force-branch"
			}, []string{"lookup:" + cleanupTestID}, true},
		{"a record of another kind", func(_ *testing.T, fx *worktreeCleanupFakeFixture, _ *CleanupParams) {
			rec := fx.wt.records[cleanupTestID]
			rec.Kind = "task"
			fx.wt.records[cleanupTestID] = rec
		}, `the worktree record org-auth records kind "task", not "org"`,
			func(fx *worktreeCleanupFakeFixture) string {
				return "cd " + cleanupShellWord(fx.root) + " && ./scripts/ralph-worktree.sh cleanup --id org-auth --force-branch"
			}, []string{"lookup:" + cleanupTestID}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newWorktreeCleanupFakeFixture(t)
			p := fx.params()
			tc.setup(t, fx, &p)
			manifest := fileOrAbsent(t, fx.o.Manifest.Path())
			res, err := fx.o.CleanupFeatureWorktree(context.Background(), p)
			if err != nil || res.Refused == "" || res.Failed != "" || res.Cleaned || res.Recorded || res.NothingLeft {
				t.Fatalf("cleanup = %+v, %v; want a refusal with something left", res, err)
			}
			manual := ""
			if tc.manual != nil {
				manual = tc.manual(fx)
			}
			assertCleanupRefusal(t, res, tc.want, manual)
			// The error of a missing script ends with start's next step,
			// which no refusal of the cleanup names.
			if strings.Contains(res.Refused, "start again") {
				t.Errorf("refusal %q names start's next step", res.Refused)
			}
			if !slices.Equal(fx.wt.calls, tc.calls) {
				t.Errorf("worktree calls = %q, want %q", fx.wt.calls, tc.calls)
			}
			if now := fileOrAbsent(t, fx.o.Manifest.Path()); now != manifest {
				t.Errorf("a refused cleanup changed the manifest:\n%s\n->\n%s", manifest, now)
			}
			assertCleanupUnlocked(t, fx.stateDir)
			if orgID, ok := readRegateMarker(regateLockPathIn(fx.stateDir)); ok != tc.marker || ok && orgID != "auth" {
				t.Errorf("marker = %q (%v); want auth's: %v", orgID, ok, tc.marker)
			}
		})
	}
}

// TestCleanupFeatureWorktree_RecordOfAnotherPathOrKindNamesTheChecks covers
// R2-L1 of docs/reports/self-review-2026-10-28-org-merge-cleanup.md: a record
// on the binding's branch whose path or kind is not the binding's is refused
// with the script's cleanup as Manual, which removes the worktree at the
// record's path and deletes the branch with git branch -D (the script refuses
// only an uncommitted change), so the reason names, as the refusals of a
// record that is gone or on another branch do, what to check before it runs:
// the commits the PR did not merge and the worktree at the record's path.
func TestCleanupFeatureWorktree_RecordOfAnotherPathOrKindNamesTheChecks(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(rec *WorktreeRecord)
		diff   func(fx *worktreeCleanupFakeFixture) string
		path   func(fx *worktreeCleanupFakeFixture) string // the record's worktree_path
	}{
		{"another path", func(rec *WorktreeRecord) { rec.WorktreePath = "/elsewhere/org auth" },
			func(fx *worktreeCleanupFakeFixture) string {
				return `worktree_path "/elsewhere/org auth", not "` + fx.f.Worktree + `"`
			}, func(*worktreeCleanupFakeFixture) string { return "'/elsewhere/org auth'" }},
		{"another kind", func(rec *WorktreeRecord) { rec.Kind = "task" },
			func(*worktreeCleanupFakeFixture) string { return `kind "task", not "org"` },
			func(fx *worktreeCleanupFakeFixture) string { return cleanupShellWord(fx.f.Worktree) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newWorktreeCleanupFakeFixture(t)
			rec := fx.wt.records[cleanupTestID]
			tc.change(&rec)
			fx.wt.records[cleanupTestID] = rec
			res, err := fx.o.CleanupFeatureWorktree(context.Background(), fx.params())
			root := cleanupShellWord(fx.root)
			want := "the worktree record org-auth records " + tc.diff(fx) + `: it is not the record of the feature worktree org_id ` +
				`"auth" had for the branch feat/auth and may be another feature's on the same org_id, so nothing is removed: check ` +
				"the record (scripts/ralph-worktree.sh state-path org-auth names its file), and clean up by hand only once it is " +
				"this feature's, the branch holds nothing that was not merged (git -C " + root + " log " + cleanupTestHead +
				"..feat/auth), and the worktree at the record's path holds nothing you need (git -C " + tc.path(fx) + " status)"
			if err != nil || res.Cleaned || res.Refused != want {
				t.Fatalf("cleanup = %+v, %v\nwant the refusal %q", res, err, want)
			}
			if manual := "cd " + root + " && ./scripts/ralph-worktree.sh cleanup --id org-auth --force-branch"; res.Manual != manual {
				t.Errorf("Manual = %q, want %q", res.Manual, manual)
			}
		})
	}
}

// TestCleanupFeatureWorktree_RefusedWhileRegateLockIsHeld covers AC4's
// regate or merge that runs: with regate.lock held by another, the cleanup
// that takes the lock itself refuses after regateLockWait, reads nothing, and
// leaves the lock to its holder.
func TestCleanupFeatureWorktree_RefusedWhileRegateLockIsHeld(t *testing.T) {
	fx := newWorktreeCleanupFakeFixture(t)
	other, err := acquireHeldFileLock(fx.stateDir, regateLockFile, "regate", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = other.release() }()
	res, err := fx.o.CleanupFeatureWorktree(context.Background(), fx.params())
	if err != nil || res.Cleaned {
		t.Fatalf("cleanup = %+v, %v; want a refusal", res, err)
	}
	assertCleanupRefusal(t, res, "another ralph org regate, merge, or cleanup runs on this ledger ("+regateLockPathIn(fx.stateDir)+
		" is held): one runs at a time, so wait for it to end and run ralph org cleanup --branch feat/auth again", "")
	if len(fx.wt.calls) != 0 {
		t.Errorf("worktree calls = %q while the lock is held; want none", fx.wt.calls)
	}
}

// TestCleanupFeatureWorktree_CallerMistakes covers the parameters only a
// caller's bug gets wrong: an empty repo root and an unknown via are errors,
// before any lock.
func TestCleanupFeatureWorktree_CallerMistakes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(p *CleanupParams)
		want   string
	}{
		{"no repo root", func(p *CleanupParams) { p.RepoRoot = "" }, "org: cleanup: no repo root given"},
		{"an unknown via", func(p *CleanupParams) { p.Via = "disband" }, `org: cleanup: via "disband" is neither merge nor cleanup`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newWorktreeCleanupFakeFixture(t)
			p := fx.params()
			tc.change(&p)
			res, err := fx.o.CleanupFeatureWorktree(context.Background(), p)
			if err == nil || !strings.Contains(err.Error(), tc.want) || res.Refused != "" || res.Cleaned {
				t.Fatalf("cleanup = %+v, %v; want the error %q", res, err, tc.want)
			}
			if _, statErr := os.Stat(regateLockPathIn(fx.stateDir)); !errors.Is(statErr, os.ErrNotExist) {
				t.Errorf("regate.lock was made (%v); want no lock taken", statErr)
			}
		})
	}
}

// TestCleanupFeatureWorktree_FakeScriptFailure covers Failed through
// fakeWorktrees, whose record names the fixture's real worktree, so the
// checks of git pass: the fake's Cleanup error is Failed, with nothing
// recorded.
func TestCleanupFeatureWorktree_FakeScriptFailure(t *testing.T) {
	fx := newWorktreeCleanupFixture(t)
	fake := newFakeWorktrees()
	fake.records[cleanupTestID] = WorktreeRecord{WorktreePath: fx.worktree, Branch: cleanupTestBranch, Kind: featureWorktreeKind}
	fake.cleanupErr = errors.New("ralph-worktree: refusing cleanup; worktree has uncommitted changes")
	fx.o.Worktrees = fake
	manifest := fileOrAbsent(t, fx.o.Manifest.Path())
	res, err := fx.cleanup(fx.params())
	if err != nil || res.Cleaned || res.Refused != "" || res.Failed != "scripts/ralph-worktree.sh cleanup --id org-auth "+
		"--force-branch failed (ralph-worktree: refusing cleanup; worktree has uncommitted changes): what it removed before it "+
		"stopped stays removed, and once its cause is fixed, ralph org cleanup --branch feat/auth goes on from what is left" ||
		res.Manual != fx.scriptCmd() {
		t.Fatalf("cleanup = %+v, %v; want Failed with the fake's error", res, err)
	}
	if want := []string{"lookup:" + cleanupTestID, "cleanup:" + cleanupTestID}; !slices.Equal(fake.calls, want) {
		t.Errorf("worktree calls = %q, want %q", fake.calls, want)
	}
	if now := fileOrAbsent(t, fx.o.Manifest.Path()); now != manifest {
		t.Errorf("a failed removal recorded something:\n%s\n->\n%s", manifest, now)
	}
}

// TestCleanupFeatureWorktree_RefusalCommandsRunAsWritten covers the commands
// a refusal's reason names (L2 of
// docs/reports/self-review-2026-10-28-org-merge-cleanup.md), in a repo whose
// path holds a space: each path is one shell word, so each command, run with
// sh as the reason writes it, does what the reason says, and with the steps
// of each refusal in turn done, the next cleanup passes every check. The
// branch a refusal of a moved branch has made to keep what it holds, from the
// main worktree, still holds the commit the PR did not merge (R2-L2): putting
// the branch back moved the branch, not the one kept. The feature worktree is
// git's own and fakeWorktrees holds its record, so the script never runs: the
// cleanup stops right before the removal.
func TestCleanupFeatureWorktree_RefusalCommandsRunAsWritten(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, wt string)
		// steps are the commands each refusal in turn names, for root, wt
		// (both shell-quoted), and head.
		steps func(root, wt, head string) [][]string
	}{
		{"a detached HEAD", func(t *testing.T, wt string) {
			runGit(t, "-C", wt, "checkout", "--quiet", "--detach")
		}, func(_, wt, _ string) [][]string { return [][]string{{"git -C " + wt + " switch feat/auth"}} }},
		{"the branch moved past the merged head", func(t *testing.T, wt string) {
			gateCommit(t, wt, "not merged")
		}, func(root, wt, head string) [][]string {
			return [][]string{{"git -C " + root + " log " + head + "..feat/auth", "git -C " + root + " branch feat/auth-unmerged feat/auth",
				"git -C " + wt + " reset --hard " + head}}
		}},
		{"the branch moved, another branch checked out", func(t *testing.T, wt string) {
			gateCommit(t, wt, "not merged")
			runGit(t, "-C", wt, "switch", "--quiet", "-c", "other")
		}, func(root, wt, head string) [][]string {
			return [][]string{{"git -C " + root + " log " + head + "..feat/auth", "git -C " + root + " branch feat/auth-unmerged feat/auth",
				"git -C " + root + " branch -f feat/auth " + head}, {"git -C " + wt + " switch feat/auth"}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateGitEnv(t)
			fx := newWorktreeCleanupFakeFixtureAt(t, filepath.Join(resolvedOrClean(t.TempDir()), "my repo"))
			wt := fx.f.Worktree
			runGit(t, "init", "--quiet", "-b", "main", fx.root)
			gateCommit(t, fx.root, "init")
			runGit(t, "-C", fx.root, "worktree", "add", "--quiet", "-b", cleanupTestBranch, wt)
			gateCommit(t, wt, "the feature")
			p := fx.params()
			p.Head = gitOut(t, wt, "rev-parse", "HEAD")
			tc.setup(t, wt)
			moved := gitOut(t, fx.root, "rev-parse", "refs/heads/"+cleanupTestBranch)
			for _, cmds := range tc.steps(shellQuote(fx.root), shellQuote(wt), p.Head) {
				res, err := fx.o.CleanupFeatureWorktree(context.Background(), p)
				if err != nil || res.Refused == "" || res.Manual != "" {
					t.Fatalf("cleanup = %+v, %v; want a refusal with no command by hand", res, err)
				}
				for _, cmd := range cmds {
					if !strings.Contains(res.Refused, "("+cmd+")") && !strings.Contains(res.Refused, "run "+cmd+",") {
						t.Fatalf("refusal %q does not name %s", res.Refused, cmd)
					}
					if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
						t.Fatalf("%s: %v\n%s", cmd, err, out)
					}
				}
			}
			// The commit the PR did not merge is on the branch kept, which
			// putting feat/auth back did not move.
			if moved != p.Head {
				if kept := gitOut(t, fx.root, "rev-parse", "refs/heads/feat/auth-unmerged"); kept != moved {
					t.Errorf("feat/auth-unmerged is at %s; want %s, the commit that was not merged", kept, moved)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			passed := false
			cleanupTestHooks.beforeScript = func() { passed = true; cancel() }
			t.Cleanup(func() { cleanupTestHooks.beforeScript = nil })
			res, err := fx.o.CleanupFeatureWorktree(ctx, p)
			if !passed || err == nil || !strings.Contains(err.Error(), "after the checks passed, before anything was removed") {
				t.Fatalf("cleanup = %+v, %v (checks passed %v); want every check passed", res, err, passed)
			}
		})
	}
}

// TestCleanupShellWord covers the words of a Manual command: a plain path or
// branch stays as it is, and one with a space, a quote, or a character a
// shell reads is single-quoted.
func TestCleanupShellWord(t *testing.T) {
	for in, want := range map[string]string{
		"/repo/.claude/worktrees/org-auth": "/repo/.claude/worktrees/org-auth",
		"feat/auth":                        "feat/auth",
		"/My Repos/ralph":                  `'/My Repos/ralph'`,
		"/repo/it's":                       `'/repo/it'\''s'`,
		"feat/$x":                          `'feat/$x'`,
		"=main":                            `'=main'`,
		"feat/a~1":                         `'feat/a~1'`,
	} {
		if got := cleanupShellWord(in); got != want {
			t.Errorf("cleanupShellWord(%q) = %q, want %q", in, got, want)
		}
	}
}
