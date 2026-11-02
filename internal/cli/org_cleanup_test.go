package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org cleanup` (plan
// docs/plans/active/2026-10-28-org-merge-cleanup.md, S3: AC2, AC4 and AC6 at
// the command line). The checks before the removal are covered by
// internal/org/cleanup_test.go. These unit-test cleanupSteps,
// cleanupOriginRefusal, and cleanupPrinter, and run the command with the real
// git and scripts/ralph-worktree.sh against a bare origin
// (newCleanupCLIFixture, on merge's newMergeCLIWorktreeFixture) and the fake
// gh of merge-order (mergeOrderFakeGH), which answers gh pr list per branch,
// so no test runs a real gh or reaches a network. None runs in parallel: they
// set PATH and chdir.

// cleanupCLIFixture is merge's worktree fixture (the main worktree work with
// the ledger, feat/a's feature worktree org-a made by the script's ensure,
// org a's binding, gate_passed, and disband) with merge-order's fake gh, which
// prints <ghPRs>/<branch>.json for gh pr list --head <branch> whatever its -R.
// git remote get-url origin prints https://github.com/o/r.git
// ($MERGE_ORDER_ORIGIN_URL), so cleanup reads the PRs with -R github.com/o/r.
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

// cleanupGHCall is the fake gh's log line for cleanup's PR lookup of branch:
// merge-order's, in origin's repository o/r (S-X1 of the plan).
func cleanupGHCall(branch string) string {
	return mergeOrderGHCall(branch) + " -R github.com/o/r"
}

// AC2 through the CLI and the real scripts/ralph-worktree.sh: after a squash
// merge on GitHub and the deletion of origin's feat/a, which git branch -d
// refuses, cleanup removes the feature worktree org-a, its registration, the
// local branch, and the record, records one worktree_cleaned with
// via=cleanup, prints CLEANED:, exits 0, releases regate.lock, and runs no gh
// but gh pr list, in origin's repository. Run again, with nothing left of
// feat/a, it prints nothing to clean up and exits 0 with the ledger as it
// was, with --branch feat/a too.
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
	if got := fx.ghCalls(t); !slices.Equal(got, []string{cleanupGHCall("feat/a")}) {
		t.Errorf("gh calls = %q; want only merge-order's gh pr list of feat/a, with -R github.com/o/r", got)
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
	want := "org: cleanup: gh is not on PATH, so whether the PRs of the ledger's feature branches were merged cannot be " +
		"told, and nothing was removed: put gh on PATH and run ralph org cleanup --state-dir " + shellQuoteIfNeeded(fx.stateDir) +
		" again"
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

// Finding 1 (cross-review cycle 1,
// docs/reports/cross-review-triage-org-merge-cleanup.md; S-X1 of the plan):
// gh's PR lookup is pinned to origin's repository o/r, as ralph org merge
// pins its own, so a MERGED PR at feat/a's head that gh returns with the url
// of another owner/name, or of o/r on another host, is not used: cleanup
// skips feat/a with a SKIPPED: line that names the PR gh returned and says
// to check origin (C2-L2 of
// docs/reports/self-review-2026-10-28-org-merge-cleanup.md), exit 1, and
// --branch feat/a is refused with the same reason and nothing on stdout. Both
// gh calls carry -R github.com/o/r, and nothing is removed or recorded.
func TestOrgCleanup_MergedPROfAnotherRepositoryRemovesNothing(t *testing.T) {
	for _, tc := range []struct{ name, url string }{
		{"another owner/name", "https://github.com/x/r/pull/7"},
		{"o/r on another host", "https://ghe.example.com/o/r/pull/7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newCleanupCLIFixture(t)
			fx.squashMergeOnGitHub(t)
			pr := cleanupCLIPR(7, "MERGED", fx.head)
			pr.URL = tc.url
			fx.prs(t, "feat/a", pr)
			before := fx.snapshot(t)

			why := "no answer of gh was used: gh pr list --head feat/a -R github.com/o/r returned PR #7 with the url " +
				strconv.Quote(tc.url) + ", which is not a PR of origin's repository github.com/o/r; check origin (git remote get-url " +
				"origin)"
			stdout, stderr, err := fx.cleanup(t)
			stateDir := shellQuoteIfNeeded(fx.stateDir)
			want := "SKIPPED: feat/a whether its PR was merged is not known (" + why + "), so nothing was removed: once gh reads " +
				"its PR, run ralph org cleanup --state-dir " + stateDir + " again\n"
			if err == nil || err.Error() != cleanupLeftErr(1) || stderr != "" || stdout != want {
				t.Errorf("cleanup: err %v, stderr %q, stdout:\n%s\nwant exit 1 with %q and:\n%s", err, stderr, stdout,
					cleanupLeftErr(1), want)
			}
			stdout, _, err = fx.cleanup(t, "--branch", "feat/a")
			wantErr := "org: cleanup: whether the PR of the branch feat/a was merged is not known (" + why + "), so nothing was " +
				"removed: once gh reads its PR, run ralph org cleanup --branch feat/a --state-dir " + stateDir + " again"
			if err == nil || err.Error() != wantErr || stdout != "" {
				t.Errorf("cleanup --branch feat/a: err %v, stdout %q; want the error %q and nothing on stdout", err, stdout, wantErr)
			}
			if got := fx.ghCalls(t); !slices.Equal(got, []string{cleanupGHCall("feat/a"), cleanupGHCall("feat/a")}) {
				t.Errorf("gh calls = %q; want gh pr list of feat/a with -R github.com/o/r, twice", got)
			}
			if after := fx.snapshot(t); after != before {
				t.Errorf("cleanup on another repository's PR changed something:\n%s\n->\n%s", before, after)
			}
		})
	}
}

// S-X1's origin: when origin's URL names no GitHub repository (the bare
// origin's local path, a file URL) or git remote get-url origin fails,
// cleanup is refused before it runs gh, without --branch and with it, as
// ralph org merge refuses such an origin: exit 1 with the reason and the
// command to run again, nothing on stdout, and nothing removed or recorded.
// The other origins that name no repository are
// TestMergeOrigin_ReadsOriginsRepositoryOrSaysWhyNot's (internal/org), and
// the wording of each problem TestOrgCleanup_OriginRefusalForEachProblem's.
func TestOrgCleanup_OriginThatNamesNoGitHubRepositoryIsRefused(t *testing.T) {
	const only = "ralph org cleanup removes a feature worktree only once the PR of its branch in origin's repository is merged"
	noRepo := "org: cleanup: origin's URL names no host and owner/repo (a local path, file://, or a scheme other than https, " +
		"http, ssh and git), so there is no GitHub repository to read the PRs in and nothing was removed: " + only + ", so " +
		"point origin at the GitHub repository (git remote set-url origin <URL>) and "
	for _, tc := range []struct {
		name   string
		origin string // $MERGE_ORDER_ORIGIN_URL; "" leaves git remote get-url origin to the real git
		// noRemote removes origin, so git remote get-url origin fails.
		noRemote bool
		// With suffix empty, the refusal is prefix and then the command to run
		// again; else prefix, git's error, suffix, and then that command.
		prefix, suffix string
	}{
		{"the bare origin's local path", "", false, noRepo, ""},
		{"a file URL", "file:///srv/o/r.git", false, noRepo, ""},
		{"no remote origin", "", true, "org: cleanup: git remote get-url origin failed (exit status ",
			"), so origin's repository is unknown and nothing was removed: " + only + ", so add origin (git remote add origin " +
				"<URL>) and "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newCleanupCLIFixture(t)
			fx.squashMergeOnGitHub(t)
			t.Setenv("MERGE_ORDER_ORIGIN_URL", tc.origin)
			if tc.noRemote {
				runLedgerTestGit(t, "-C", fx.work, "remote", "remove", "origin")
			}
			before := fx.snapshot(t)

			stateDir := shellQuoteIfNeeded(fx.stateDir)
			for _, run := range []struct {
				extra []string
				again string
			}{
				{nil, "run ralph org cleanup --state-dir " + stateDir + " again"},
				{[]string{"--branch", "feat/a"}, "run ralph org cleanup --branch feat/a --state-dir " + stateDir + " again"},
			} {
				stdout, _, err := fx.cleanup(t, run.extra...)
				refused := err != nil && err.Error() == tc.prefix+run.again
				if tc.suffix != "" {
					refused = err != nil && strings.HasPrefix(err.Error(), tc.prefix) && strings.HasSuffix(err.Error(), tc.suffix+run.again)
				}
				if !refused || stdout != "" {
					t.Errorf("cleanup %q: err %v, stdout %q; want the error %q...%q and nothing on stdout", run.extra, err, stdout,
						tc.prefix, tc.suffix+run.again)
				}
			}
			if got := fx.ghCalls(t); len(got) != 0 {
				t.Errorf("gh ran: %q", got)
			}
			if after := fx.snapshot(t); after != before {
				t.Errorf("a refused cleanup changed something:\n%s\n->\n%s", before, after)
			}
		})
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
				"org cleanup --branch feat/a --state-dir " + orgLayerShellWord(fx.stateDir) + " again\n"
		}},
		{"regate.lock held", func(t *testing.T, fx *cleanupCLIFixture) {
			holdMergeOrderLock(t, filepath.Join(fx.stateDir, "regate.lock"))
		}, func(fx *cleanupCLIFixture) string {
			return "SKIPPED: feat/a another ralph org regate, merge, or cleanup runs on this ledger (" +
				filepath.Join(fx.stateDir, "regate.lock") + " is held): one runs at a time, so wait for it to end and run ralph " +
				"org cleanup --branch feat/a --state-dir " + orgLayerShellWord(fx.stateDir) + " again\n"
		}},
		{"the org runs", func(t *testing.T, fx *cleanupCLIFixture) {
			appendSeedEvent(t, fx.stateDir, org.ManifestEvent{
				OrgID: "a", Event: org.EventScopeReserved, Worktree: fx.wt,
				Details: "paths=internal/a/ split=demo feature=a digest=0123456789ab branch=feat/a approval=per-plan",
			})
		}, func(fx *cleanupCLIFixture) string {
			return `SKIPPED: feat/a the branch feat/a is bound to org_id "a", which is running and may still use its feature ` +
				"worktree " + fx.wt + ": wait for the org to be disbanded and run ralph org cleanup --branch feat/a --state-dir " +
				orgLayerShellWord(fx.stateDir) + " again\n"
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

// R2-L3 of docs/reports/self-review-2026-10-28-org-merge-cleanup.md through
// the CLI: the head gh reads (headRefOid) is a commit made on GitHub, which
// origin has only under refs/pull/7/head (the branch deleted on merge) and
// work does not have, so cleanup, with --branch and without it, skips feat/a,
// exit 1 with nothing removed, with a reason that names the fetch of the head
// of PR #7, the number merge-order read for the branch. With the reason's
// commands run with sh as it writes them, the next cleanup cleans feat/a up
// at that head, and the branch kept is where feat/a was.
func TestOrgCleanup_HeadNotHereNamesTheFetchOfThePRsHead(t *testing.T) {
	fx := newCleanupCLIFixture(t)
	head := gateCLIGitOut(t, fx.bare, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit-tree", "-p", fx.head, "-m",
		"update the branch, on GitHub", fx.head+"^{tree}")
	runLedgerTestGit(t, "-C", fx.bare, "update-ref", "refs/pull/7/head", head)
	runLedgerTestGit(t, "-C", fx.bare, "update-ref", "-d", "refs/heads/feat/a")
	fx.prs(t, "feat/a", cleanupCLIPR(7, "MERGED", head))
	before := fx.snapshot(t)

	kept, reset := "feat/a-unmerged-"+fx.head[:7], "git -C "+fx.wt+" reset --hard "+head
	fetch, keep := "git -C "+fx.work+" fetch origin refs/pull/7/head", "git -C "+fx.work+" branch "+kept+" feat/a"
	want := "SKIPPED: feat/a the local branch feat/a is at " + fx.head + ", not at " + head + ", the head its PR merged, which git " +
		"does not have in " + fx.work + " (a commit made on GitHub or pushed from another clone), so whether the branch holds " +
		"commits that were not merged cannot be told yet: fetch the head (" + fetch + "), then look at them (git -C " + fx.work +
		" log " + head + "..feat/a), keep what you need on another branch made from the main worktree, not by switching branches " +
		"in the feature worktree (" + keep + "), put the branch back at the head its PR merged (" + reset + "), and run ralph org " +
		"cleanup --branch feat/a --state-dir " + orgLayerShellWord(fx.stateDir) + " again\n"
	for _, extra := range [][]string{{"--branch", "feat/a"}, nil} {
		stdout, stderr, err := fx.cleanup(t, extra...)
		if err == nil || err.Error() != cleanupLeftErr(1) || stderr != "" || stdout != want {
			t.Errorf("cleanup %q: err %v, stderr %q, stdout:\n%s\nwant exit 1 with %q and:\n%s", extra, err, stderr, stdout,
				cleanupLeftErr(1), want)
		}
	}
	if after := fx.snapshot(t); after != before {
		t.Errorf("a skipped cleanup changed something:\n%s\n->\n%s", before, after)
	}

	for _, cmd := range []string{fetch, keep, reset} {
		if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
			t.Fatalf("%s: %v\n%s", cmd, err, out)
		}
	}
	stdout, stderr, err := fx.cleanup(t)
	if err != nil || stderr != "" || stdout != "CLEANED: feat/a org-a\n" {
		t.Fatalf("cleanup after the fetch: err %v, stderr %q, stdout %q; want feat/a cleaned", err, stderr, stdout)
	}
	if got := fx.tip(t, kept); got != fx.head {
		t.Errorf("%s is at %s, want %s, where feat/a was", kept, got, fx.head)
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
	want := "org: cleanup: the main worktree, where scripts/ralph-worktree.sh removes the feature worktrees, was not found " +
		"(the current directory is in no git repository, or in one with no main worktree, such as a linked worktree of a bare " +
		"repository), so nothing was read or removed: run ralph org cleanup --state-dir " + shellQuoteIfNeeded(fx.stateDir) +
		" from the main worktree"
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

// cleanup changes the ledger, so with RALPH_ORG_STATE_DIR, which ralph no
// longer reads, naming another directory than the default ledger, it is
// refused as the other verbs that change the ledger are (plan
// 2026-10-30-org-state-dir-local, D5 and AC6; checkRetiredOrgStateDirEnv):
// exit 1, an error marked errOrgLedgerRefusedByRetiredEnv, not
// errOrgLedgerUnresolved, that starts as a refusal and names both
// directories and the ways out, nothing on stdout and no note, no gh call,
// and nothing removed or recorded, the merged feat/a's worktree, its
// registration, its branch, its record, and the ledger all as they were.
// With --state-dir naming the ledger, the same cleanup cleans up feat/a with
// one note, so the refusal is the variable's alone.
func TestOrgCleanup_RefusedWhileTheRetiredStateDirEnvNamesAnotherLedger(t *testing.T) {
	fx := newCleanupCLIFixture(t)
	fx.squashMergeOnGitHub(t)
	before := fx.snapshot(t)
	envDir := filepath.Join(mergeOrderTempDir(t), "old ledger")
	t.Setenv(retiredOrgStateDirEnv, envDir)

	stdout, stderr, err := runOrgCmdStreams(t, "cleanup")
	want := "ralph org cleanup: refusing to change the org ledger: RALPH_ORG_STATE_DIR names " + envDir + ", which ralph no " +
		"longer reads, and the ledger ralph finds by default is " + fx.stateDir + ", so nothing was changed; to keep using " +
		envDir + ", write [org].state_dir = " + strconv.Quote(envDir) + " in " + filepath.Join(fx.work, config.LocalConfigFileName) +
		" or pass --state-dir " + shellQuoteIfNeeded(envDir) + ", and unset RALPH_ORG_STATE_DIR; to use " + fx.stateDir +
		", unset RALPH_ORG_STATE_DIR"
	if code := ExitCode(err); code != 1 || !errors.Is(err, errOrgLedgerRefusedByRetiredEnv) || errors.Is(err, errOrgLedgerUnresolved) ||
		err.Error() != want {
		t.Errorf("cleanup = exit %d, %v\nwant exit 1 with\n%s", code, err, want)
	}
	if stdout != "" || strings.Contains(stderr, retiredEnvNote) {
		t.Errorf("cleanup printed stdout %q, stderr %q; want nothing beside the error", stdout, stderr)
	}
	if got := fx.ghCalls(t); len(got) != 0 {
		t.Errorf("gh ran: %q", got)
	}
	if after := fx.snapshot(t); after != before {
		t.Errorf("a refused cleanup changed something:\n%s\n->\n%s", before, after)
	}
	if _, err := os.Stat(envDir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a refused cleanup made the variable's directory %s (stat: %v)", envDir, err)
	}

	stdout, stderr, err = fx.cleanup(t)
	if err != nil || stdout != "CLEANED: feat/a org-a\n" {
		t.Fatalf("cleanup --state-dir with the variable set: err %v, stdout %q, stderr %q; want feat/a cleaned up", err, stdout, stderr)
	}
	checkRetiredEnvNote(t, "ralph org cleanup --state-dir", stderr, envDir, fx.stateDir, false)
	if got := len(fx.cleanedEvents(t)); got != 1 {
		t.Errorf("the ledger has %d worktree_cleaned; want 1", got)
	}
}

// Without --state-dir, cleanup reads the ledger [org].state_dir names in the
// personal file at the main worktree's root (source config; plan
// 2026-10-30-org-state-dir-local, AC1, AC5, and AC8). The fixture's ledger is
// moved out of the repository, so the default ledger holds nothing. A
// relative value is refused first: exit 1, an error marked
// errOrgLedgerUnresolved that names the file, no gh call, and nothing removed
// or recorded. With the moved ledger's absolute path, cleanup finds feat/a's
// binding there, cleans up its feature worktree in the main worktree
// (org.FeatureRepoRoot for source config), and records worktree_cleaned in
// that ledger, none in the default one.
func TestOrgCleanup_PersonalFileNamesTheLedger(t *testing.T) {
	fx := newCleanupCLIFixture(t)
	fx.squashMergeOnGitHub(t)
	defaultLedger := fx.stateDir
	moved := filepath.Join(mergeOrderTempDir(t), "ledger")
	if err := os.Rename(defaultLedger, moved); err != nil {
		t.Fatal(err)
	}
	fx.stateDir = moved
	before := fx.snapshot(t)
	t.Setenv(config.LocalConfigEnv, "")
	path := filepath.Join(fx.work, config.LocalConfigFileName)

	writeTestFile(t, path, "[org]\nstate_dir = \"ledger\"\n")
	stdout, _, err := runOrgCmdStreams(t, "cleanup")
	want := "ralph org cleanup: cannot resolve the org ledger: " + path + `: [org].state_dir "ledger" must be an absolute path ` +
		"or start with ~/"
	if code := ExitCode(err); code != 1 || !errors.Is(err, errOrgLedgerUnresolved) || !strings.HasPrefix(err.Error(), want) {
		t.Errorf("cleanup with a relative [org].state_dir = exit %d, %v\nwant exit 1 with an error that starts\n%s", code, err, want)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if got := fx.ghCalls(t); len(got) != 0 {
		t.Errorf("gh ran: %q", got)
	}
	if after := fx.snapshot(t); after != before {
		t.Errorf("a refused cleanup changed something:\n%s\n->\n%s", before, after)
	}

	writeTestFile(t, path, fmt.Sprintf("[org]\nstate_dir = %q\n", moved))
	stdout, stderr, err := runOrgCmdStreams(t, "cleanup")
	if err != nil || stderr != "" || stdout != "CLEANED: feat/a org-a\n" {
		t.Fatalf("cleanup with [org].state_dir = %s: err %v, stdout %q, stderr %q; want feat/a cleaned up", moved, err, stdout, stderr)
	}
	if _, err := os.Stat(fx.wt); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the worktree %s is still there (stat: %v)", fx.wt, err)
	}
	if got := fx.worktreeRecord(t); got != "<absent>" {
		t.Errorf("the record org-a is still there:\n%s", got)
	}
	wantDetails := "branch=feat/a head=" + fx.head + " via=cleanup worktree=" + strconv.Quote(fx.wt)
	if cleaned := fx.cleanedEvents(t); len(cleaned) != 1 || cleaned[0].OrgID != "a" || cleaned[0].Details != wantDetails {
		t.Errorf("worktree_cleaned in %s = %+v; want org a's one with %q", moved, cleaned, wantDetails)
	}
	if _, err := os.Stat(org.ManifestPathIn(defaultLedger)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the default ledger %s has a manifest (stat: %v); want the cleanup recorded in %s alone", defaultLedger, err, moved)
	}
	if got := fx.ghCalls(t); !slices.Equal(got, []string{cleanupGHCall("feat/a")}) {
		t.Errorf("gh calls = %q; want only merge-order's gh pr list of feat/a, with -R github.com/o/r", got)
	}
}
