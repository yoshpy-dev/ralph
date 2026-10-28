package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org cleanup` (plan
// docs/plans/active/2026-10-28-org-merge-cleanup.md, S3: AC2, AC4 and AC6 at
// the command line). The checks before the removal are covered by
// internal/org/cleanup_test.go. These unit-test cleanupSteps and
// cleanupPrinter, and run the command with the real git and
// scripts/ralph-worktree.sh against a bare origin (newCleanupCLIFixture, on
// merge's newMergeCLIWorktreeFixture) and the fake gh of merge-order
// (mergeOrderFakeGH), which answers gh pr list per branch, so no test runs a
// real gh or reaches a network. None runs in parallel: they set PATH and
// chdir.

// cleanupCLIFixture is merge's worktree fixture (the main worktree work with
// the ledger, feat/a's feature worktree org-a made by the script's ensure,
// org a's binding, gate_passed, and disband) with merge-order's fake gh, which
// prints <ghPRs>/<branch>.json for gh pr list --head <branch>.
type cleanupCLIFixture struct {
	*mergeCLIFixture
	wt string // feat/a's feature worktree
}

func newCleanupCLIFixture(t *testing.T) *cleanupCLIFixture {
	t.Helper()
	fx, wt := newMergeCLIWorktreeFixture(t)
	writeStubScript(t, filepath.Join(fx.bin, "gh"), mergeOrderFakeGH)
	return &cleanupCLIFixture{mergeCLIFixture: fx, wt: wt}
}

// squashMergeOnGitHub does to origin what GitHub's squash merge of PR #7 does,
// with head branches deleted on merge: origin's main gets one commit with
// feat/a's tree, which feat/a is not an ancestor of, and origin's feat/a is
// deleted. The fake gh then reads PR #7 of feat/a as MERGED at fx.head.
func (fx *cleanupCLIFixture) squashMergeOnGitHub(t *testing.T) {
	t.Helper()
	squash := gateCLIGitOut(t, fx.work, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit-tree", "-p",
		"refs/heads/main", "-m", "a (#7)", "refs/heads/feat/a^{tree}")
	fx.push(t, squash+":refs/heads/main", ":refs/heads/feat/a")
	fx.prs(t, "feat/a", cleanupCLIPR(7, "MERGED", fx.head))
}

// cleanupCLIPR is PR n of origin's repository o/r with state and head, into
// main.
func cleanupCLIPR(n int, state, head string) mergeOrderGHPR {
	return mergeOrderGHPR{Number: n, State: state, URL: "https://github.com/o/r/pull/" + strconv.Itoa(n), HeadRefOid: head,
		BaseRefName: "main"}
}

// cleanup runs `ralph org cleanup --state-dir <the fixture's ledger>` with
// extra after it.
func (fx *cleanupCLIFixture) cleanup(t *testing.T, extra ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runOrgCmdStreams(t, append([]string{"cleanup", "--state-dir", fx.stateDir}, extra...)...)
}

// snapshot is what the cleanup may remove or record: whether the worktree's
// directory is there, git worktree list, feat/a's tip, the record org-a, and
// the ledger.
func (fx *cleanupCLIFixture) snapshot(t *testing.T) string {
	t.Helper()
	_, statErr := os.Stat(fx.wt)
	return strings.Join([]string{fmt.Sprint(statErr == nil), gateCLIGitOut(t, fx.work, "worktree", "list", "--porcelain"),
		gateCLIGitOut(t, fx.work, "branch", "--list", "--format=%(refname) %(objectname)", "feat/a"), fx.worktreeRecord(t),
		fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))}, "\n--\n")
}

// cleanedEvents are the ledger's worktree_cleaned events.
func (fx *cleanupCLIFixture) cleanedEvents(t *testing.T) []org.ManifestEvent {
	t.Helper()
	var out []org.ManifestEvent
	for _, ev := range readManifestEvents(t, org.ManifestPathIn(fx.stateDir)) {
		if ev.Event == org.EventWorktreeCleaned {
			out = append(out, ev)
		}
	}
	return out
}

// cleanupLeftErr is the error of a cleanup that left n branches.
func cleanupLeftErr(n int) string {
	return "org: cleanup: " + strconv.Itoa(n) + " branch(es) were not cleaned up: the SKIPPED: line of each says why and what " +
		"to do next"
}

// AC2 through the CLI and the real scripts/ralph-worktree.sh: after a squash
// merge on GitHub and the deletion of origin's feat/a, which git branch -d
// refuses, cleanup removes the feature worktree org-a, its registration, the
// local branch, and the record, records one worktree_cleaned with
// via=cleanup, prints CLEANED:, exits 0, releases regate.lock, and runs no gh
// but gh pr list. Run again, with nothing left of feat/a, it prints nothing
// to clean up and exits 0 with the ledger as it was, with --branch feat/a too.
func TestOrgCleanup_CleansUpABranchMergedOnGitHub(t *testing.T) {
	fx := newCleanupCLIFixture(t)
	fx.squashMergeOnGitHub(t)
	if err := exec.Command("git", "-C", fx.bare, "merge-base", "--is-ancestor", fx.head, "refs/heads/main").Run(); err == nil {
		t.Fatal("origin's main contains feat/a; want a branch git branch -d refuses")
	}
	if got := gateCLIGitOut(t, fx.bare, "branch", "--list", "feat/a"); got != "" {
		t.Fatalf("origin still has feat/a: %q", got)
	}

	stdout, stderr, err := fx.cleanup(t)
	if err != nil || stderr != "" {
		t.Fatalf("cleanup: err %v, stderr %q (stdout %q)", err, stderr, stdout)
	}
	if want := "CLEANED: feat/a org-a\n"; stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
	if _, err := os.Stat(fx.wt); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the worktree %s is still there (stat: %v)", fx.wt, err)
	}
	if list := gateCLIGitOut(t, fx.work, "worktree", "list", "--porcelain"); strings.Contains(list, fx.wt) {
		t.Errorf("git still lists the worktree:\n%s", list)
	}
	if got := gateCLIGitOut(t, fx.work, "branch", "--list", "feat/a"); got != "" {
		t.Errorf("the branch feat/a is still there: %q", got)
	}
	if got := fx.worktreeRecord(t); got != "<absent>" {
		t.Errorf("the record org-a is still there:\n%s", got)
	}
	events := readManifestEvents(t, org.ManifestPathIn(fx.stateDir))
	want := "branch=feat/a head=" + fx.head + " via=cleanup worktree=" + strconv.Quote(fx.wt)
	if last := events[len(events)-1]; last.Event != org.EventWorktreeCleaned || last.OrgID != "a" || last.SeatID != "" ||
		last.Details != want || len(fx.cleanedEvents(t)) != 1 {
		t.Errorf("the ledger ends with %+v (%d worktree_cleaned); want org a's one worktree_cleaned with %q", last,
			len(fx.cleanedEvents(t)), want)
	}
	if got := fx.ghCalls(t); !slices.Equal(got, []string{mergeOrderGHCall("feat/a")}) {
		t.Errorf("gh calls = %q; want only merge-order's gh pr list of feat/a", got)
	}
	t.Run("again, with nothing left", func(t *testing.T) {
		before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))
		for _, extra := range [][]string{nil, {"--branch", "feat/a"}} {
			stdout, stderr, err := fx.cleanup(t, extra...)
			if err != nil || stderr != "" || stdout != "nothing to clean up\n" {
				t.Errorf("cleanup %q again: err %v, stderr %q, stdout %q; want exit 0 and nothing to clean up", extra, err, stderr, stdout)
			}
		}
		if got := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); got != before {
			t.Errorf("the ledger changed:\nbefore %s\nafter  %s", before, got)
		}
	})
	// regate.lock is free once cleanup has returned.
	holdMergeOrderLock(t, filepath.Join(fx.stateDir, "regate.lock"))
}

// D4's gh: with gh missing from PATH, whether a PR was merged cannot be told,
// so cleanup exits 1 saying so, with nothing on stdout and nothing removed or
// recorded.
func TestOrgCleanup_WithoutGHRemovesNothing(t *testing.T) {
	fx := newCleanupCLIFixture(t)
	fx.squashMergeOnGitHub(t)
	before := fx.snapshot(t)
	path := os.Getenv("PATH")
	if err := os.Remove(filepath.Join(fx.bin, "gh")); err != nil {
		t.Fatal(err)
	}
	// The developer's PATH after the fixture's bin may hold a real gh.
	t.Setenv("PATH", fx.bin)

	stdout, _, err := fx.cleanup(t)
	t.Setenv("PATH", path)
	const want = "org: cleanup: gh is not on PATH, so whether the PRs of the ledger's feature branches were merged cannot be " +
		"told, and nothing was removed: put gh on PATH and run ralph org cleanup again"
	if err == nil || err.Error() != want || stdout != "" {
		t.Fatalf("cleanup without gh: err %v, stdout %q; want the error %q and nothing on stdout", err, stdout, want)
	}
	if after := fx.snapshot(t); after != before {
		t.Errorf("cleanup without gh changed something:\n%s\n->\n%s", before, after)
	}
}

// D4's --branch: a branch whose PR is OPEN is refused, exit 1 with nothing
// on stdout and nothing removed; without --branch, the same branch gets no
// line, and with nothing else, cleanup prints nothing to clean up and exits 0.
func TestOrgCleanup_BranchWhosePRIsNotMergedIsRefused(t *testing.T) {
	fx := newCleanupCLIFixture(t)
	fx.prs(t, "feat/a", cleanupCLIPR(7, "OPEN", fx.head))
	before := fx.snapshot(t)

	stdout, _, err := fx.cleanup(t, "--branch", "feat/a")
	const want = "org: cleanup: PR #7 of the branch feat/a is OPEN, not MERGED, so nothing was removed: cleanup removes the " +
		"feature worktree of a branch whose PR was merged"
	if err == nil || err.Error() != want || stdout != "" {
		t.Errorf("cleanup --branch feat/a: err %v, stdout %q; want the error %q and nothing on stdout", err, stdout, want)
	}
	stdout, stderr, err := fx.cleanup(t)
	if err != nil || stderr != "" || stdout != "nothing to clean up\n" {
		t.Errorf("cleanup: err %v, stderr %q, stdout %q; want exit 0 and nothing to clean up", err, stderr, stdout)
	}
	if after := fx.snapshot(t); after != before {
		t.Errorf("cleanup of an open PR's branch changed something:\n%s\n->\n%s", before, after)
	}
}

// AC3 and AC4 through the CLI: a refusal of the cleanup is a SKIPPED: line
// and exit 1 with nothing removed or recorded: an untracked file in the
// worktree, regate.lock held by a regate, a merge, or another cleanup, and
// the branch's org running. The next cleanup goes on from each once its
// cause is gone, so none is followed by a MANUAL: line.
func TestOrgCleanup_RefusalsAreSkippedLines(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, fx *cleanupCLIFixture)
		want  func(fx *cleanupCLIFixture) string
	}{
		{"an untracked file", func(t *testing.T, fx *cleanupCLIFixture) {
			writeTestFile(t, filepath.Join(fx.wt, "stray.txt"), "x\n")
		}, func(fx *cleanupCLIFixture) string {
			return "SKIPPED: feat/a the feature worktree " + fx.wt + " has 1 uncommitted change(s) or untracked file(s) (?? " +
				"stray.txt), which removing it would lose: move what you need out of the worktree, remove the rest, and run ralph " +
				"org cleanup --branch feat/a again\n"
		}},
		{"regate.lock held", func(t *testing.T, fx *cleanupCLIFixture) {
			holdMergeOrderLock(t, filepath.Join(fx.stateDir, "regate.lock"))
		}, func(fx *cleanupCLIFixture) string {
			return "SKIPPED: feat/a another ralph org regate, merge, or cleanup runs on this ledger (" +
				filepath.Join(fx.stateDir, "regate.lock") + " is held): one runs at a time, so wait for it to end and run ralph " +
				"org cleanup --branch feat/a again\n"
		}},
		{"the org runs", func(t *testing.T, fx *cleanupCLIFixture) {
			appendSeedEvent(t, fx.stateDir, org.ManifestEvent{
				OrgID: "a", Event: org.EventScopeReserved, Worktree: fx.wt,
				Details: "paths=internal/a/ split=demo feature=a digest=0123456789ab branch=feat/a approval=per-plan",
			})
		}, func(fx *cleanupCLIFixture) string {
			return `SKIPPED: feat/a the branch feat/a is bound to org_id "a", which is running and may still use its feature ` +
				"worktree " + fx.wt + ": wait for the org to be disbanded and run ralph org cleanup --branch feat/a again\n"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newCleanupCLIFixture(t)
			fx.squashMergeOnGitHub(t)
			tc.setup(t, fx)
			before := fx.snapshot(t)

			stdout, stderr, err := fx.cleanup(t)
			if err == nil || err.Error() != cleanupLeftErr(1) || stderr != "" {
				t.Errorf("cleanup: err %v, stderr %q; want exit 1 with %q", err, stderr, cleanupLeftErr(1))
			}
			if want := tc.want(fx); stdout != want {
				t.Errorf("stdout:\n%s\nwant:\n%s", stdout, want)
			}
			if after := fx.snapshot(t); after != before {
				t.Errorf("a skipped cleanup changed something:\n%s\n->\n%s", before, after)
			}
		})
	}
}

// D4's branches left out without --branch: a branch cleaned up already, with
// nothing left of it (NothingLeft), and the old branch of an org_id bound to
// another branch since (BoundTo), whose local branch is left, get no line,
// and with nothing else cleanup prints nothing to clean up and exits 0 with
// the ledger and the old branch as they were. The new branch of that org_id,
// merged with nothing of it ever made, is NothingLeft too. --branch on the
// old branch handles it: the record is gone, so the cleanup removes nothing
// and names git branch -D, exit 1.
func TestOrgCleanup_LeavesOutCleanedAndReboundBranches(t *testing.T) {
	fx := newCleanupCLIFixture(t)
	fx.squashMergeOnGitHub(t)
	if stdout, _, err := fx.cleanup(t); err != nil || stdout != "CLEANED: feat/a org-a\n" {
		t.Fatalf("first cleanup: err %v, stdout %q; want feat/a cleaned", err, stdout)
	}
	orgB := filepath.Join(fx.work, ".claude", "worktrees", "org-b")
	for _, feature := range []string{"old", "new"} {
		appendSeedEvent(t, fx.stateDir, org.ManifestEvent{
			OrgID: "b", Event: org.EventScopeReserved, Worktree: orgB,
			Details: "paths=internal/b/ split=demo feature=" + feature + " digest=0123456789ab branch=feat/" + feature + " approval=per-plan",
		})
		appendSeedEvent(t, fx.stateDir, org.ManifestEvent{OrgID: "b", Event: org.EventDisbanded})
	}
	old := fx.branch(t, "feat/old")
	fx.prs(t, "feat/old", cleanupCLIPR(8, "MERGED", old))
	fx.prs(t, "feat/new", cleanupCLIPR(9, "MERGED", fx.main))
	before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))

	stdout, stderr, err := fx.cleanup(t)
	if err != nil || stderr != "" || stdout != "nothing to clean up\n" {
		t.Errorf("cleanup: err %v, stderr %q, stdout %q; want exit 0 and nothing to clean up", err, stderr, stdout)
	}
	if got := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); got != before {
		t.Errorf("the ledger changed:\nbefore %s\nafter  %s", before, got)
	}
	if got := fx.tip(t, "feat/old"); got != old {
		t.Errorf("feat/old is at %s, want %s", got, old)
	}

	stdout, _, err = fx.cleanup(t, "--branch", "feat/old")
	wantPrefix := "SKIPPED: feat/old the worktree record org-b of the branch feat/old does not exist, so ralph cannot tell that " +
		"what is left is this feature's and removes nothing: the branch feat/old at " + old + " left;"
	wantManual := "\nMANUAL: git -C " + fx.work + " branch -D feat/old\n"
	if err == nil || err.Error() != cleanupLeftErr(1) || !strings.HasPrefix(stdout, wantPrefix) || !strings.HasSuffix(stdout, wantManual) {
		t.Errorf("cleanup --branch feat/old: err %v, stdout %q; want %q ... %q and exit 1", err, stdout, wantPrefix, wantManual)
	}
	if got := fx.tip(t, "feat/old"); got != old {
		t.Errorf("feat/old is at %s after --branch, want %s", got, old)
	}
}

// When the main worktree cannot be found (the ledger is chosen with
// --state-dir and the current directory is in no repository), cleanup exits
// 1 before merge-order runs: no gh call, nothing on stdout, the ledger as it
// was.
func TestOrgCleanup_NoMainWorktreeReadsNothing(t *testing.T) {
	fx := newMergeCLIFixture(t, false)
	outside := filepath.Join(filepath.Dir(fx.work), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(outside))
	t.Chdir(outside)
	before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))

	stdout, _, err := runOrgCmdStreams(t, "cleanup", "--state-dir", fx.stateDir)
	const want = "org: cleanup: the main worktree, where scripts/ralph-worktree.sh removes the feature worktrees, was not found " +
		"(the current directory is in no git repository, or in one with no main worktree, such as a linked worktree of a bare " +
		"repository), so nothing was read or removed: run ralph org cleanup from the main worktree"
	if err == nil || err.Error() != want || stdout != "" {
		t.Errorf("cleanup outside a repository: err %v, stdout %q; want the error %q", err, stdout, want)
	}
	if got := fx.ghCalls(t); len(got) != 0 {
		t.Errorf("gh ran: %q", got)
	}
	if got := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); got != before {
		t.Errorf("the ledger changed:\nbefore %s\nafter  %s", before, got)
	}
}

// --org-id and an empty --branch are refused before anything is read: exit 1
// with the reason, nothing on stdout, no git or gh run (not even to resolve
// the state dir), and no state dir made.
func TestOrgCleanup_FlagRefusalsReadNothing(t *testing.T) {
	isolateFeatureGitEnv(t)
	calls := writeMergeOrderCallLoggers(t)
	tmp := mergeOrderTempDir(t)
	t.Chdir(tmp)
	stateDir := filepath.Join(tmp, "state")
	const (
		orgIDRefusal  = "org: cleanup: --org-id does not apply: cleanup reads the org from the binding of each merged feature branch in the ledger"
		branchRefusal = "org: cleanup: --branch is empty: name the merged feature branch to clean up, or leave --branch out to clean up every merged one"
	)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"cleanup", "--org-id", "a", "--state-dir", stateDir}, orgIDRefusal},
		{[]string{"cleanup", "--org-id", "a", "--branch", "feat/a"}, orgIDRefusal},
		{[]string{"cleanup", "--branch", "", "--state-dir", stateDir}, branchRefusal},
		{[]string{"cleanup", "--branch", "  "}, branchRefusal},
	} {
		stdout, _, err := runOrgCmdStreams(t, tc.args...)
		if err == nil || err.Error() != tc.want || stdout != "" {
			t.Errorf("%q: err %v, stdout %q; want the error %q and nothing on stdout", tc.args, err, stdout, tc.want)
		}
	}
	if got := readLogLines(t, calls); len(got) != 0 {
		t.Errorf("git or gh ran before the refusal: %q", got)
	}
	for _, dir := range []string{stateDir, filepath.Join(tmp, ".harness")} {
		if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("a refused cleanup made %s (stat: %v)", dir, err)
		}
	}
}

// cleanup changes the ledger, so guardLegacyOrgStateDir refuses it while the
// linked worktree's older ledger still has an active seat, before origin, gh,
// or the shared ledger is touched.
func TestOrgCleanup_RefusedWhileALegacyLedgerHasActiveSeats(t *testing.T) {
	setupOrgStubPATH(t)
	r := newLegacyLedgerRepo(t)
	spawnLedgerSeat(t, r.legacyDir, "seat-1")
	before := readFileBytes(t, org.ManifestPathIn(r.legacyDir))

	stdout, _, err := runOrgCmdStreams(t, "cleanup")
	if err == nil {
		t.Fatalf("cleanup ran while the legacy ledger has an active seat (stdout %q)", stdout)
	}
	for _, want := range []string{"org: refusing to change the org ledger", r.legacyDir, "1 active seat(s)"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err = %v; want it to say %q", err, want)
		}
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if after := readFileBytes(t, org.ManifestPathIn(r.legacyDir)); string(after) != string(before) {
		t.Errorf("the legacy ledger changed:\nbefore %s\nafter  %s", before, after)
	}
	if _, err := os.Stat(r.sharedDir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a refused cleanup made the shared ledger %s (stat: %v)", r.sharedDir, err)
	}
}

// The command's context reaches merge-order: with it done before the
// command runs, cleanup exits 1 with merge-order's interruption, nothing on
// stdout, and nothing removed or recorded.
func TestOrgCleanup_InterruptedBeforeItRunsRemovesNothing(t *testing.T) {
	fx := newCleanupCLIFixture(t)
	fx.squashMergeOnGitHub(t)
	before := fx.snapshot(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stdout, _, err := runOrgCmdStreamsContext(ctx, "cleanup", "--state-dir", fx.stateDir)
	const want = "org: cleanup: nothing was removed, since the feature branches could not be read as ralph org merge-order " +
		"--all reads them: org: merge-order: interrupted ("
	if err == nil || !strings.HasPrefix(err.Error(), want) || stdout != "" {
		t.Errorf("cleanup with its context done: err %v, stdout %q; want it to start with %q and nothing on stdout", err, stdout, want)
	}
	if after := fx.snapshot(t); after != before {
		t.Errorf("an interrupted cleanup changed something:\n%s\n->\n%s", before, after)
	}
}

// cleanupSteps over merge-order's results: without --branch, only a branch
// with a binding that is its org's latest is dealt with, cleaned up at its PR's
// head when the PR is merged and reported when gh did not tell (failed,
// limit_forks, skipped); with --branch, that branch alone, its BoundTo
// ignored, or the refusal of a branch not in the ledger, with no binding, no
// PR, a PR gh did not read, or a PR that is not merged. No entry is nothing
// to do without --branch, and gh not found is refused either way.
func TestCleanupSteps_PicksTheMergedBranches(t *testing.T) {
	const head = "0123456789abcdef0123456789abcdef01234567"
	bind := func(boundTo string) *org.MergeOrderBinding {
		return &org.MergeOrderBinding{OrgID: "a", Feature: "f", Split: "s", TS: "t", BoundTo: boundTo}
	}
	pr := func(state string) *org.MergeOrderPR {
		return &org.MergeOrderPR{Number: 7, State: state, URL: "https://github.com/o/r/pull/7", HeadRefOid: head, BaseRefName: "main"}
	}
	entry := func(branch, class string, p *org.MergeOrderPR, lookup string, b *org.MergeOrderBinding) org.MergeOrderEntry {
		return org.MergeOrderEntry{Branch: branch, Class: class, PR: p, PRLookup: lookup, Binding: b}
	}
	found := org.MergeOrderPRLookupFound
	res := org.MergeOrderResult{GHAvailable: true, Entries: []org.MergeOrderEntry{
		entry("feat/open", org.MergeClassReady, pr("OPEN"), found, bind("")),
		entry("feat/merged", org.MergeClassMerged, pr("MERGED"), found, bind("")),
		entry("feat/rebound", org.MergeClassMerged, pr("MERGED"), found, bind("feat/merged")),
		entry("feat/gate-only", org.MergeClassMerged, pr("MERGED"), found, nil),
		entry("feat/closed", org.MergeClassClosed, pr("CLOSED"), found, bind("")),
		entry("feat/none", org.MergeClassNoGate, nil, org.MergeOrderPRLookupNone, bind("")),
		entry("feat/failed", org.MergeClassNoGate, nil, org.MergeOrderPRLookupFailed, bind("")),
		entry("feat/forks", org.MergeClassNoGate, nil, org.MergeOrderPRLookupLimitForks, bind("")),
		entry("feat/skipped", org.MergeClassNoGate, nil, org.MergeOrderPRLookupSkipped, bind("")),
		entry("feat/failed-rebound", org.MergeClassNoGate, nil, org.MergeOrderPRLookupFailed, bind("feat/merged")),
		entry("feat/failed-gate-only", org.MergeClassNoGate, nil, org.MergeOrderPRLookupFailed, nil),
	}}
	unknown := func(why string) string {
		return "whether its PR was merged is not known (" + why + "; the notes of ralph org merge-order --all say more), so " +
			"nothing was removed: once gh reads its PR, run ralph org cleanup again"
	}
	steps, err := cleanupSteps(res, "")
	want := []cleanupStep{
		{branch: "feat/merged", head: head},
		{branch: "feat/failed", unknown: unknown("gh pr list failed for the branch")},
		{branch: "feat/forks", unknown: unknown("gh returned its limit of PRs for the branch, all from forks")},
		{branch: "feat/skipped", unknown: unknown("no answer of gh was used")},
	}
	if err != nil || !reflect.DeepEqual(steps, want) {
		t.Errorf("cleanupSteps without --branch = %+v, %v; want %+v", steps, err, want)
	}

	const gone = "org: cleanup: no feature binding or gate record in the ledger names the branch feat/nope, so nothing was " +
		"removed: ralph org merge-order --all lists the ledger's branches"
	const noGH = "org: cleanup: gh is not on PATH, so whether the PRs of the ledger's feature branches were merged cannot be " +
		"told, and nothing was removed: put gh on PATH and run ralph org cleanup again"
	noGHRes := res
	noGHRes.GHAvailable = false
	for _, tc := range []struct {
		name    string
		res     org.MergeOrderResult
		branch  string
		want    []cleanupStep
		wantErr string
	}{
		{"a merged branch", res, "feat/merged", []cleanupStep{{branch: "feat/merged", head: head}}, ""},
		{"an old binding named", res, "feat/rebound", []cleanupStep{{branch: "feat/rebound", head: head}}, ""},
		{"a branch not in the ledger", res, "feat/nope", nil, gone},
		{"gate records only", res, "feat/gate-only", nil, "org: cleanup: no complete feature binding in the ledger names the " +
			"branch feat/gate-only, only a gate record, so it has no feature worktree to clean up and nothing was removed: " +
			"cleanup removes only the worktree ralph org start --plan made for a feature org"},
		{"no PR", res, "feat/none", nil, "org: cleanup: gh found no PR for the branch feat/none, so nothing was removed: cleanup " +
			"removes the feature worktree of a branch whose PR was merged"},
		{"a PR gh did not read", res, "feat/failed", nil, "org: cleanup: whether the PR of the branch feat/failed was merged is " +
			"not known (gh pr list failed for the branch; the notes of ralph org merge-order --all say more), so nothing was " +
			"removed: once gh reads its PR, run ralph org cleanup --branch feat/failed again"},
		{"an open PR", res, "feat/open", nil, "org: cleanup: PR #7 of the branch feat/open is OPEN, not MERGED, so nothing was " +
			"removed: cleanup removes the feature worktree of a branch whose PR was merged"},
		{"a closed PR", res, "feat/closed", nil, "org: cleanup: PR #7 of the branch feat/closed is CLOSED, not MERGED, so " +
			"nothing was removed: cleanup removes the feature worktree of a branch whose PR was merged"},
		{"no entry", org.MergeOrderResult{}, "", nil, ""},
		{"no entry, a branch named", org.MergeOrderResult{}, "feat/nope", nil, gone},
		{"gh not found", noGHRes, "", nil, noGH},
		{"gh not found, a branch named", noGHRes, "feat/merged", nil, noGH},
	} {
		t.Run(tc.name, func(t *testing.T) {
			steps, err := cleanupSteps(tc.res, tc.branch)
			switch {
			case tc.wantErr == "" && err != nil:
				t.Errorf("err = %v; want none", err)
			case tc.wantErr != "" && (err == nil || err.Error() != tc.wantErr):
				t.Errorf("err = %v; want %q", err, tc.wantErr)
			}
			if !reflect.DeepEqual(steps, tc.want) {
				t.Errorf("steps = %+v; want %+v", steps, tc.want)
			}
		})
	}
}

// The refusal for a branch whose PR gh did not read ends with the command to
// run again, and a person pastes it: a branch name may hold ;, &, |, $, ', and
// (, so the command has the branch as one shell word, while the sentence
// before it names the branch as it is. A plain branch is as it was.
func TestOrgCleanup_NamedStepQuotesTheBranchInTheCommand(t *testing.T) {
	for _, tc := range []struct{ name, branch, word string }{
		{"a plain branch", "feat/a", "feat/a"},
		{"a semicolon", "feat/a;b", "'feat/a;b'"},
		{"a command substitution", "feat/$(x)", "'feat/$(x)'"},
		{"a single quote", "feat/a'b", `'feat/a'\''b'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := org.MergeOrderResult{GHAvailable: true, Entries: []org.MergeOrderEntry{{
				Branch: tc.branch, Class: org.MergeClassNoGate, PRLookup: org.MergeOrderPRLookupFailed,
				Binding: &org.MergeOrderBinding{OrgID: "a", Feature: "f", Split: "s", TS: "t"},
			}}}
			_, err := cleanupNamedStep(res, tc.branch)
			want := "org: cleanup: whether the PR of the branch " + tc.branch + " was merged is not known (gh pr list failed " +
				"for the branch; the notes of ralph org merge-order --all say more), so nothing was removed: once gh reads " +
				"its PR, run ralph org cleanup --branch " + tc.word + " again"
			if err == nil || err.Error() != want {
				t.Errorf("cleanupNamedStep error = %v; want %q", err, want)
			}
		})
	}
}

// cleanupPrinter over the outcomes of a run, fed as runCleanupSteps feeds
// them: CLEANED: with what it resumed from and an error of its record (which
// stops the run with that error), SKIPPED: with the refusal, the failure, or
// why the PR is unknown, MANUAL: with a command by hand, no line for a branch
// with nothing left or an error without a cleanup, nothing to clean up when
// no line was printed, values escaped, and exit 1 counting the branches left.
func TestCleanupPrinter_EachOutcome(t *testing.T) {
	type outcome struct {
		branch  string
		unknown string
		res     org.CleanupResult
		err     error
	}
	// A command by hand comes only with a refusal ralph cannot go on from,
	// such as a missing record, and with a failure.
	const manual = "cd /r && ./scripts/ralph-worktree.sh cleanup --id org-a --force-branch"
	const branchDelete = "git -C /r branch -D feat/a"
	cleaned := org.CleanupResult{Cleaned: true, Recorded: true, OrgID: "a"}
	resumed := cleaned
	resumed.Resumed = org.CleanupResumedWorktreeGone
	notRecorded := org.CleanupResult{Cleaned: true, Resumed: org.CleanupResumedRecordOnly, OrgID: "a"}
	recordErr := errors.New("org: cleanup: the feature worktree /w of the branch feat/a and its record org-a were removed, but " +
		"worktree_cleaned could not be recorded (denied)")
	gitErr := errors.New("org: cleanup: git worktree list --porcelain in /r: exit 128: fatal: not a git repository")
	nothingLeft := org.CleanupResult{Refused: "the worktree record org-a of the branch feat/a does not exist, and neither its " +
		"feature worktree /w nor the branch is left: nothing is left to clean up", NothingLeft: true, OrgID: "a"}
	const unknown = "whether its PR was merged is not known (gh pr list failed for the branch)"

	for _, tc := range []struct {
		name       string
		outcomes   []outcome
		wantStdout string
		wantErr    string
	}{
		{"no branch", nil, "nothing to clean up\n", ""},
		{"cleaned up", []outcome{{branch: "feat/a", res: cleaned}}, "CLEANED: feat/a org-a\n", ""},
		{"cleaned up from what an earlier cleanup left", []outcome{{branch: "feat/a", res: resumed}},
			"CLEANED: feat/a org-a (resumed: worktree_gone)\n", ""},
		{"cleaned up, worktree_cleaned not recorded", []outcome{{branch: "feat/a", res: notRecorded, err: recordErr},
			{branch: "feat/b", res: cleaned}}, "CLEANED: feat/a org-a (resumed: record_only; " + recordErr.Error() + ")\n",
			recordErr.Error()},
		{"refused, with the command by hand", []outcome{{branch: "feat/a", res: org.CleanupResult{Refused: "the worktree record " +
			"org-a of the branch feat/a does not exist, so ralph removes nothing: the branch feat/a at \x1b left",
			Manual: branchDelete}}}, `SKIPPED: feat/a the worktree record org-a of the branch feat/a does not exist, so ralph ` +
			`removes nothing: the branch feat/a at \x1b left` + "\nMANUAL: " + branchDelete + "\n", cleanupLeftErr(1)},
		{"refused, no command by hand", []outcome{{branch: "feat/a", res: org.CleanupResult{Refused: "the org runs"}}},
			"SKIPPED: feat/a the org runs\n", cleanupLeftErr(1)},
		{"failed", []outcome{{branch: "feat/a", res: org.CleanupResult{Failed: "scripts/ralph-worktree.sh cleanup --id org-a " +
			"--force-branch failed", Manual: manual}}}, "SKIPPED: feat/a scripts/ralph-worktree.sh cleanup --id org-a " +
			"--force-branch failed\nMANUAL: " + manual + "\n", cleanupLeftErr(1)},
		{"no outcome", []outcome{{branch: "feat/a"}}, "SKIPPED: feat/a the cleanup ended with no outcome\n", cleanupLeftErr(1)},
		{"nothing left", []outcome{{branch: "feat/a", res: nothingLeft}}, "nothing to clean up\n", ""},
		{"an error before the removal", []outcome{{branch: "feat/a", err: gitErr}, {branch: "feat/b", res: cleaned}}, "",
			gitErr.Error()},
		{"a PR not known to be merged", []outcome{{branch: "feat/a", unknown: unknown}}, "SKIPPED: feat/a " + unknown + "\n",
			cleanupLeftErr(1)},
		{"several branches", []outcome{{branch: "feat/a", res: cleaned}, {branch: "feat/b", res: nothingLeft},
			{branch: "feat/c", res: org.CleanupResult{Refused: "the org runs"}}, {branch: "feat/d", unknown: unknown}},
			"CLEANED: feat/a org-a\nSKIPPED: feat/c the org runs\nSKIPPED: feat/d " + unknown + "\n", cleanupLeftErr(2)},
		{"control characters are escaped", []outcome{{branch: "feat/\x1b[31m", res: org.CleanupResult{Cleaned: true,
			Recorded: true, OrgID: "a\x1b"}}}, `CLEANED: feat/\x1b[31m org-a\x1b` + "\n", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			p := &cleanupPrinter{w: &out}
			err := func() error {
				for _, o := range tc.outcomes {
					if o.unknown != "" {
						p.unknown(o.branch, o.unknown)
						continue
					}
					if err := p.result(o.branch, o.res, o.err); err != nil {
						return err
					}
				}
				return p.finish()
			}()
			if out.String() != tc.wantStdout {
				t.Errorf("stdout:\n%q\nwant:\n%q", out.String(), tc.wantStdout)
			}
			switch {
			case tc.wantErr == "" && err != nil:
				t.Errorf("err = %v; want nil (exit 0)", err)
			case tc.wantErr != "" && (err == nil || err.Error() != tc.wantErr):
				t.Errorf("err = %v; want %q", err, tc.wantErr)
			}
		})
	}
}

// cleanup's help says what it removes, when it removes it, what it refuses,
// where to run it, how it relates to merge, its lines, its exit codes, and
// that it refuses --org-id; org --help names the verb, says that merge cleans
// up after its merge, and lists cleanup among the verbs that refuse --org-id.
func TestOrgCleanup_Help(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		wants []string
	}{
		{[]string{"cleanup", "--help"}, []string{
			"the feature worktree, the local branch, and the record of scripts/ralph-worktree.sh",
			"./scripts/ralph-worktree.sh cleanup --id org-<org_id> --force-branch in the main worktree",
			"records worktree_cleaned in the ledger with via=cleanup",
			"ralph org merge does the same cleanup right after its own merge",
			"Run it from the main worktree, not from inside the feature worktree it removes.",
			"cleans up each branch with a feature binding whose PR gh reads as MERGED, at the PR's head (headRefOid)",
			"A branch whose PR gh could not read (pr_lookup failed, limit_forks, or skipped in merge-order's --json) gets a SKIPPED: line",
			"The old branch of an org_id that was bound again to another branch later is handled only when --branch names it.",
			"refuses, removing nothing, when the ledger does not name it, it has no feature binding, gh does not read its PR, or the PR is not MERGED",
			"that the worktree has no uncommitted change or untracked file and no rebase or merge in progress (files .gitignore " +
				"names do not count and go with the worktree); that the local branch is at the head the PR merged; and that the " +
				"worktree has that branch checked out at that head",
			"ralph org regate, ralph org merge, and another cleanup are refused on the ledger (one runs at a time; it waits up to " +
				"1 s for regate.lock), and so are ralph org spawn and start of the org whose feature worktree it removes",
			"CLEANED: <branch> org-<org_id>, with (resumed: <what was left>)",
			"SKIPPED: <branch> <why>, followed, when there is one, by MANUAL: <the command that cleans up by hand once the reason is checked>",
			"With no such line, it prints nothing to clean up.",
			"Only a branch ralph cannot go on with by itself gets a MANUAL: line (the record is gone or is not the binding's, " +
				"the worktree is left with no local branch, or the removal itself failed); after any other SKIPPED: line, fix " +
				"what the reason says and run ralph org cleanup --branch <branch> again.",
			"Exit codes: 0 when every branch was cleaned up or there was nothing to clean up; 1 when a branch was skipped, gh is " +
				"not on PATH (whether a PR was merged cannot be told, so nothing is removed), the main worktree is not found, " +
				"--branch is refused,",
			"so --org-id is refused",
			"--branch string",
		}},
		{[]string{"--help"}, []string{
			"cleanup Clean up the feature worktree of a feature branch whose PR was merged on GitHub",
			"records the merge, and then cleans up the branch's feature worktree.",
			"`ralph org cleanup` cleans up the feature worktree of a branch whose PR was merged on GitHub.",
			"stop --all, disband --all, merge-order, merge, cleanup, regate, wait --inbox,",
		}},
	} {
		out, err := runOrgCmd(t, tc.args...)
		if err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		help := strings.Join(strings.Fields(out), " ")
		for _, want := range tc.wants {
			if !strings.Contains(help, want) {
				t.Errorf("%v does not say %q:\n%s", tc.args, want, out)
			}
		}
	}
}

// The help of start, spawn, and merge-order follows the cleanup after a
// merge (plan docs/plans/active/2026-10-28-org-merge-cleanup.md, S4: AC5):
// start says that merge and cleanup remove a merged feature org's worktree
// and that the script removes a closed PR's by hand, no longer that the
// script removes it after a merge; start and spawn say they are refused while
// merge or cleanup removes the org's worktree, as regateSpawnErr refuses
// them; and merge-order says what bound_to in --json is.
func TestOrgStartSpawnAndMergeOrder_HelpFollowTheCleanupAfterAMerge(t *testing.T) {
	for _, tc := range []struct {
		args            []string
		wants, notWants []string
	}{
		{[]string{"start", "--help"}, []string{
			"`ralph org disband` leaves the worktree and the branch, which `ralph org regate` works in until the PR is merged.",
			"Once the PR is merged, `ralph org merge` removes the worktree, the branch, and the record of " +
				"scripts/ralph-worktree.sh right after its merge, and `ralph org cleanup` does the same for a PR merged on GitHub",
			"For a PR closed without a merge, run `./scripts/ralph-worktree.sh cleanup --id org-<org_id> --force-branch` in the " +
				"main worktree once the branch is not needed.",
			"Both forms are refused, with nothing recorded, while `ralph org regate` works on the org's branch, or while " +
				"`ralph org merge` or `ralph org cleanup` removes the org's merged feature worktree; run start again once it ends.",
		}, []string{"`./scripts/ralph-worktree.sh cleanup --id org-<org_id>` removes them", "run start again once the regate ends"}},
		{[]string{"spawn", "--help"}, []string{
			"A spawn (not a --dry-run) into an org is refused, with nothing recorded, while `ralph org regate` works on that " +
				"org's branch, or while `ralph org merge` or `ralph org cleanup` removes that org's merged feature worktree; run " +
				"the spawn again once it ends.",
		}, []string{"run the spawn again once the regate ends"}},
		{[]string{"merge-order", "--help"}, []string{
			"Each --json entry with a binding also has bound_to in it, which the text does not show either: the branch of the " +
				"org's latest complete binding when that is another branch",
			"ralph org cleanup leaves such a branch out unless --branch names it.",
		}, nil},
	} {
		out, err := runOrgCmd(t, tc.args...)
		if err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		help := strings.Join(strings.Fields(out), " ")
		for _, want := range tc.wants {
			if !strings.Contains(help, want) {
				t.Errorf("%v does not say %q:\n%s", tc.args, want, out)
			}
		}
		for _, old := range tc.notWants {
			if strings.Contains(help, old) {
				t.Errorf("%v still says %q:\n%s", tc.args, old, out)
			}
		}
	}
}
