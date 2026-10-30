package org

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// The ci regate at the org layer (plan
// docs/plans/active/2026-10-29-regate-ci-checks.md, S4a: D2 step 1, D4, D6,
// AC5, AC6; S4b: D2 steps 2 to 4, D3, AC2, AC5, AC7): Regate with
// RegateParams.Checks config.RegateChecksCI on the regate fixture
// (regate_test.go), whose origin is a local bare repository that Regate's
// own fetches, push, and read-back of origin's tip use, with mxExec
// (merge_fake_test.go) answering `git remote get-url origin` with a GitHub
// URL and the gh calls, so no test runs gh or reaches a network.

// rcFixture is regateFixture with mxExec answering, for feat/auth at head,
// HEAD and origin's tip of the branch, origin's URL moOriginURL
// (github.com/o/r) and PR #7 of o/r, OPEN into main at head, with the check
// verify passed. tip is origin's main.
type rcFixture struct {
	*regateFixture
	exec      *mxExec
	lookPath  func(string) (string, error)
	ghTimeout time.Duration
	head, tip string
}

// rcPRURL is the url of the fixture's PR #7.
const rcPRURL = "https://github.com/o/r/pull/7"

// rcPRList is a `gh pr list --json` answer of PR #7 at head in state into
// base, with url u.
func rcPRList(u, head, state, base string) string {
	return moPRs(ghPullRequest{Number: 7, State: state, URL: u, HeadRefOid: head, BaseRefName: base})
}

// rcPRHead is a `gh pr view --json headRefOid` answer of head.
func rcPRHead(head string) mgReply { return mgReply{out: `{"headRefOid":"` + head + `"}`} }

// rcCheck is a check of the workflow ci.
func rcCheck(bucket, name, state string) ghPRCheck {
	return ghPRCheck{Bucket: bucket, Name: name, State: state, Workflow: "ci"}
}

// The calls a ci regate makes through Exec, in order, when it reads the
// checks: the pre-checks' (rcPreCheckKeys), then the PR's head and checks
// (rcChecksKeys, all it reads while they are pending or cannot be read), and,
// when the checks decide, the PR's head once more (rcReadKeys, self-review
// L4).
var (
	rcPreCheckKeys = []string{mxGetURL, mxPRList}
	rcChecksKeys   = []string{mxGetURL, mxPRList, mxPRView, mxPRChecks}
	rcReadKeys     = []string{mxGetURL, mxPRList, mxPRView, mxPRChecks, mxPRView}
)

// Parts of the texts of the default fixture.
const (
	rcDone    = "nothing was merged, pushed, or recorded"
	rcAgain   = "run ralph org regate --branch feat/auth again"
	rcNext    = rcAgain + " once CI on PR #7 has finished"
	rcAccess  = "check that gh can reach github.com and is logged in to it (gh auth status --hostname github.com), and " + rcAgain
	rcMissing = `gh is not on PATH (exec: "gh": executable file not found in $PATH), and a regate with [org.gate].regate_checks = ` +
		`"ci" reads the checks of the branch's PR with gh, so `
	rcInstall = ": install gh, log in to github.com (gh auth login --hostname github.com), and " + rcAgain
)

func newRCFixture(t *testing.T) *rcFixture {
	t.Helper()
	fx := &rcFixture{regateFixture: newRegateFixture(t), lookPath: mgFound}
	fx.head, fx.tip = fx.tips(t)
	fx.exec = &mxExec{origin: moOriginURL + "\n", replies: map[string]mgReply{
		mxPRList:   {out: rcPRList(rcPRURL, fx.head, "OPEN", "main")},
		mxPRView:   rcPRHead(fx.head),
		mxPRChecks: {out: mxChecks(mxPass("verify"))},
	}}
	return fx
}

// params is a ci regate of feat/auth with the fixture's environment, Exec,
// LookPath, and GHTimeout.
func (fx *rcFixture) params() RegateParams {
	return RegateParams{Branch: "feat/auth", Env: fx.env, Checks: config.RegateChecksCI, Exec: fx.exec.run, LookPath: fx.lookPath,
		GHTimeout: fx.ghTimeout}
}

// regateCI is Regate with params.
func (fx *rcFixture) regateCI(ctx context.Context) (RegateResult, error) {
	return fx.o.Regate(ctx, fx.params())
}

// logPath is the gate log a ci regate of the fixture writes.
func (fx *rcFixture) logPath() string { return filepath.Join(fx.stateDir, gateLogDir, regateTestLog) }

// assertKeys fails the test unless the calls through Exec are keys and each
// ran in the feature worktree pinned as ralph org merge's are: git with
// GIT_TERMINAL_PROMPT=0, gh with GH_PROMPT_DISABLED=1 and -R github.com/o/r,
// each with a deadline.
func (fx *rcFixture) assertKeys(t *testing.T, keys []string) {
	t.Helper()
	if got := fx.exec.keys(); !slices.Equal(got, keys) {
		t.Errorf("calls = %q\nwant %q", got, keys)
	}
	for _, c := range fx.exec.snapshot() {
		env, pinned := []string{"GH_PROMPT_DISABLED=1"}, false
		if c.name == "git" {
			env, pinned = []string{"GIT_TERMINAL_PROMPT=0"}, true
		} else if i := slices.Index(c.args, "-R"); i > 0 && i+1 < len(c.args) {
			pinned = c.args[i+1] == mxRepoArg
		}
		if !pinned || c.dir != fx.worktree || !slices.Equal(c.env, env) || c.deadline.IsZero() {
			t.Errorf("%s %q in %q with %q (deadline %v) is not pinned to %s in %s with %q", c.name, c.args, c.dir, c.env, c.deadline,
				mxRepoArg, fx.worktree, env)
		}
	}
}

// ciEvents returns the manifest's gate records of a ci regate,
// gate_ci_passed and gate_ci_failed.
func (fx *rcFixture) ciEvents(t *testing.T) []ManifestEvent {
	t.Helper()
	var out []ManifestEvent
	for _, ev := range mustReadEvents(t, fx.o) {
		if ev.Event == EventGateCIPassed || ev.Event == EventGateCIFailed {
			out = append(out, ev)
		}
	}
	return out
}

// assertNothingWritten fails the test unless the manifest is before, no gate
// log was written, no script ran, and regate.lock is released.
func (fx *rcFixture) assertNothingWritten(t *testing.T, before string) {
	t.Helper()
	if after := fileOrAbsent(t, fx.o.Manifest.Path()); after != before {
		t.Errorf("the manifest changed:\nbefore %s\nafter  %s", before, after)
	}
	if _, err := os.Stat(filepath.Join(fx.stateDir, gateLogDir)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s exists (stat: %v); want no log", gateLogDir, err)
	}
	if calls := fx.calls(t); calls != nil {
		t.Errorf("scripts run = %v; want none", calls)
	}
	fx.assertUnlocked(t)
}

// TestRegateCI_RefusedBeforeAnythingChanges covers AC6 of the plan (D2 step
// 1): with [org.gate].regate_checks = "ci", origin's URL that cannot be read,
// times out, or names no GitHub repository, gh not on PATH, and a branch whose
// PR gh cannot read or that has no PR to read the checks of (none, past a
// limit of fork PRs, another repository's, not OPEN, into another branch) are
// each refused with the reason and one next step, with nothing merged,
// pushed, or recorded, the worktree and origin as they were, no script run,
// and no call after the refusing one.
func TestRegateCI_RefusedBeforeAnythingChanges(t *testing.T) {
	block := func(key string) func(*rcFixture) {
		return func(fx *rcFixture) { fx.exec.block, fx.ghTimeout = key, 30*time.Millisecond }
	}
	reply := func(key string, r mgReply) func(*rcFixture) {
		return func(fx *rcFixture) { fx.exec.replies[key] = r }
	}
	listed := func(u, state, base string) func(*rcFixture) {
		return func(fx *rcFixture) { fx.exec.replies[mxPRList] = mgReply{out: rcPRList(u, fx.head, state, base)} }
	}
	const list = "gh pr list --head feat/auth -R " + mxRepoArg
	for _, tc := range []struct {
		name  string
		setup func(fx *rcFixture)
		keys  []string
		want  func(fx *rcFixture) []string
	}{
		{"origin's URL cannot be read", func(fx *rcFixture) { fx.exec.originErr = errors.New("exit status 2: error: No such remote 'origin'") },
			[]string{mxGetURL}, func(fx *rcFixture) []string {
				return []string{"git remote get-url origin in " + fx.worktree + " failed (exit status 2: error: No such remote 'origin'), " +
					"so origin's repository, whose PR checks a regate with [org.gate].regate_checks = \"ci\" reads, is unknown and " +
					rcDone + ": add origin (git -C " + fx.worktree + " remote add origin <URL>) and " + rcAgain}
			}},
		{"origin's get-url times out", block(mxGetURL), []string{mxGetURL}, func(fx *rcFixture) []string {
			return []string{"git remote get-url origin in " + fx.worktree + " timed out after 30ms", "check the repository's git config and " + rcAgain}
		}},
		{"origin is a local path", func(fx *rcFixture) { fx.exec.origin = fx.origin + "\n" }, []string{mxGetURL}, func(fx *rcFixture) []string {
			return []string{"origin's URL in " + fx.worktree + " names no host and owner/repo", "point origin at the GitHub repository (git -C " +
				fx.worktree + " remote set-url origin <URL>), or set [org.gate].regate_checks = \"local\" to run the gate scripts here, and " +
				rcAgain}
		}},
		{"gh is not on PATH", func(fx *rcFixture) { fx.lookPath = mgNotFound }, []string{mxGetURL}, func(*rcFixture) []string {
			return []string{"org: regate: " + rcMissing + rcDone + rcInstall}
		}},
		{"gh pr list fails", reply(mxPRList, mgReply{err: errors.New("exit status 1: HTTP 502: Bad Gateway")}), rcPreCheckKeys,
			func(*rcFixture) []string {
				return []string{"org: regate: " + list + " failed (exit status 1: HTTP 502: Bad Gateway), so the branch's PR is unknown and " +
					rcDone + ": " + rcAccess}
			}},
		{"gh pr list times out", block(mxPRList), rcPreCheckKeys, func(*rcFixture) []string {
			return []string{list + " failed (timed out after 30ms: context deadline exceeded)", rcAccess}
		}},
		{"gh is not found at gh pr list", reply(mxPRList, mxNotFound), rcPreCheckKeys, func(*rcFixture) []string {
			return []string{"org: regate: " + rcMissing + rcDone + rcInstall}
		}},
		{"no PR", reply(mxPRList, mgReply{out: "[]"}), rcPreCheckKeys, func(*rcFixture) []string {
			return []string{"org: regate: " + list + " found no PR of the branch feat/auth in origin's repository " + mxRepoArg + ", whose " +
				"checks a regate with [org.gate].regate_checks = \"ci\" reads, so " + rcDone + ": push the branch and open its PR into " +
				"main (gh pr create --head feat/auth --base main -R " + mxRepoArg + "), and once CI ran on it, " + rcAgain}
		}},
		{"a limit full of fork PRs", reply(mxPRList, mgReply{out: moPRs(moForkPRs(1, mergeOrderGHLimit)...)}), rcPreCheckKeys,
			func(*rcFixture) []string {
				return []string{list + " returned 1000 PRs, its limit, all from forks", "set [org.gate].regate_checks = \"local\" to run the " +
					"gate scripts here instead, and " + rcAgain}
			}},
		{"a PR of another repository", listed("https://github.com/x/y/pull/7", "OPEN", "main"), rcPreCheckKeys, func(fx *rcFixture) []string {
			return []string{list + " returned PR #7 with the url https://github.com/x/y/pull/7, which is not a PR of origin's " +
				"repository " + mxRepoArg + ", so " + rcDone + ": check origin (git -C " + fx.worktree + " remote get-url origin) and " + rcAgain}
		}},
		{"a closed PR", listed(rcPRURL, "CLOSED", "main"), rcPreCheckKeys, func(*rcFixture) []string {
			return []string{"PR #7 of the branch feat/auth is CLOSED, not OPEN", "reopen it (gh pr reopen 7 -R " + mxRepoArg + ") or open " +
				"another into main, and " + rcAgain}
		}},
		{"a merged PR", listed(rcPRURL, "MERGED", "main"), rcPreCheckKeys, func(*rcFixture) []string {
			return []string{"PR #7 of the branch feat/auth is MERGED, not OPEN", "when it was merged, there is nothing to regate"}
		}},
		{"a PR into another branch", listed(rcPRURL, "OPEN", "release"), rcPreCheckKeys, func(*rcFixture) []string {
			return []string{"org: regate: PR #7 of the branch feat/auth merges into release, not origin's default branch main, so " + rcDone +
				": change its base (gh pr edit 7 --base main -R " + mxRepoArg + ") and " + rcAgain}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRCFixture(t)
			tc.setup(fx)
			before := fx.snapshot(t)
			res, err := fx.regateCI(context.Background())
			fx.assertRefused(t, res, err, before, append(tc.want(fx), rcDone), nil)
			fx.assertKeys(t, tc.keys)
		})
	}
}

// TestRegateCI_ChecksOriginGHAndThePRBeforeEveryPath covers AC6's "before
// the merge": the ci pre-checks run right after the fetches, so a gh not on
// PATH is refused before a HEAD that needs the merge, before a HEAD that the
// retry of a push would push (the same setup, run with local checks, pushes
// it), and before a HEAD whose gate_ci_passed would let Regate stop.
func TestRegateCI_ChecksOriginGHAndThePRBeforeEveryPath(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, fx *rcFixture)
		local RegateResult // with local checks, when set: what Regate does instead
	}{
		{"a HEAD that needs the merge", func(t *testing.T, fx *rcFixture) { fx.pushToOrigin(t, "main") }, RegateResult{}},
		{"a HEAD that the retry of a push would push", func(t *testing.T, fx *rcFixture) {
			writeGateFile(t, fx.worktree, "internal/auth/more.go", "package auth\n", 0o644)
			runGit(t, "-C", fx.worktree, "add", ".")
			gateCommit(t, fx.worktree, "not pushed")
			head := gitOut(t, fx.worktree, "rev-parse", "HEAD")
			fx.appendEvents(t, moGate("auth", "feat/auth", head, fx.tip, "refs/remotes/origin/main", ""))
		}, RegateResult{Outcome: RegateOutcomePushOnly, Pushed: true}},
		{"a HEAD with a gate_ci_passed", func(t *testing.T, fx *rcFixture) {
			fx.appendEvents(t, moCIGate("auth", "feat/auth", fx.head, fx.tip, ""))
		}, RegateResult{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRCFixture(t)
			tc.setup(t, fx)
			fx.lookPath = mgNotFound
			before := fx.snapshot(t)
			res, err := fx.regateCI(context.Background())
			fx.assertRefused(t, res, err, before, []string{"org: regate: " + rcMissing + rcDone + rcInstall}, nil)
			fx.assertKeys(t, []string{mxGetURL})
			if tc.local.Outcome == "" {
				return
			}
			res, err = fx.regate(context.Background())
			if err != nil || res.Outcome != tc.local.Outcome || res.Pushed != tc.local.Pushed {
				t.Fatalf("with local checks, regate = %+v, %v; want %s with Pushed %t", res, err, tc.local.Outcome, tc.local.Pushed)
			}
		})
	}
}

// TestRegateCI_CommandsQuoteTheWorktreeAndTheBranch covers self-review L8:
// the commands a ci regate's texts name hold the worktree's path and the
// branches as one shell word each, so a path with a space, a $, and a ; and
// branch names git accepts with a $ and a ; are not split or expanded by a
// shell. The pre-checks' refusals are read from regateCIPreCheck with mxExec
// answering; the commands to check origin after a push (regateCICheckOrigin),
// to check origin after a ci pass whose head origin moved off
// (regateCIPassMovedText), and to fast-forward after a ci pass whose head
// origin has commits on top of (self-review L1, through
// regateHeadNotOnOrigin) are run with sh -c as written, in a repository at
// that path whose origin has the branch one commit past HEAD.
func TestRegateCI_CommandsQuoteTheWorktreeAndTheBranch(t *testing.T) {
	isolateGitEnv(t)
	tmp := resolved(t, t.TempDir())
	const branch, base = "feat/a$HOME;b", "rel$x"
	origin, root := filepath.Join(tmp, "origin.git"), filepath.Join(tmp, "wt $HOME; x")
	runGit(t, "init", "--quiet", "--bare", "-b", base, origin)
	runGit(t, "init", "--quiet", "-b", branch, root)
	gateCommit(t, root, "first")
	head := gitOut(t, root, "rev-parse", "HEAD")
	gateCommit(t, root, "on top")
	onTop := gitOut(t, root, "rev-parse", "HEAD")
	runGit(t, "-C", root, "remote", "add", "origin", origin)
	runGit(t, "-C", root, "push", "--quiet", "origin", "HEAD:refs/heads/"+branch)
	runGit(t, "-C", root, "reset", "--quiet", "--hard", head)
	tgt := gateTarget{root: root, branch: branch, head: head, base: gateBase{branch: base, ref: "refs/remotes/origin/" + base, sha: head}}
	w := regateWords(branch)

	// Written by hand, not with cleanupShellWord: the path has no '.
	inRoot := "git -C '" + root + "' "
	ref := "'refs/remotes/origin/" + branch + "'"
	checkOrigin := inRoot + "fetch origin, then " + inRoot + "log --oneline --graph HEAD " + ref
	fastForward := inRoot + "fetch origin, then " + inRoot + "merge --ff-only " + ref
	sh := func(t *testing.T, cmd string) {
		t.Helper()
		c := exec.Command("sh", "-c", cmd)
		c.Dir = tmp
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("sh -c %q: %v\n%s", cmd, err, out)
		}
	}

	for _, tc := range []struct {
		name  string
		setup func(e *mxExec)
		want  string
	}{
		{"origin's URL cannot be read", func(e *mxExec) { e.originErr = errors.New("exit status 2") },
			"add origin (" + inRoot + "remote add origin <URL>)"},
		{"origin is a local path", func(e *mxExec) { e.origin = origin + "\n" },
			"point origin at the GitHub repository (" + inRoot + "remote set-url origin <URL>)"},
		{"no PR", func(e *mxExec) { e.replies[mxPRList] = mgReply{out: "[]"} },
			"(gh pr create --head 'feat/a$HOME;b' --base 'rel$x' -R " + mxRepoArg + ")"},
		{"a PR of another repository", func(e *mxExec) {
			e.replies[mxPRList] = mgReply{out: rcPRList("https://github.com/x/y/pull/7", head, "OPEN", base)}
		}, "check origin (" + inRoot + "remote get-url origin)"},
		{"a PR into another branch", func(e *mxExec) { e.replies[mxPRList] = mgReply{out: rcPRList(rcPRURL, head, "OPEN", "main")} },
			"(gh pr edit 7 --base 'rel$x' -R " + mxRepoArg + ")"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := &mxExec{origin: moOriginURL + "\n", replies: map[string]mgReply{}}
			tc.setup(e)
			p := RegateParams{Exec: e.run, LookPath: mgFound, GHTimeout: time.Second}
			ci, refused, err := regateCIPreCheck(context.Background(), p, tgt, w)
			if err != nil || ci != nil || !strings.Contains(refused, tc.want) {
				t.Fatalf("regateCIPreCheck = %v, %q, %v; want a refusal with %q", ci, refused, err, tc.want)
			}
		})
	}
	sh(t, inRoot+"remote get-url origin")

	if got := regateCICheckOrigin(tgt, w); !strings.Contains(got, "("+checkOrigin+")") {
		t.Errorf("regateCICheckOrigin = %q; want it to name %q", got, checkOrigin)
	}
	if got := regateCIPassMovedText(tgt, onTop, w); !strings.Contains(got, "("+checkOrigin+")") {
		t.Errorf("regateCIPassMovedText = %q; want it to name %q", got, checkOrigin)
	}
	b := regateBranch{gate: &mergeGateRecord{readable: true, record: gateRecord{Head: head, Base: head, BaseRef: tgt.base.ref,
		Via: gateViaRegateCI}}}
	res, err := regateHeadNotOnOrigin(context.Background(), b, tgt, onTop, nil, w, RegateResult{})
	if err != nil || !strings.Contains(res.Refused, "("+fastForward+")") {
		t.Fatalf("regateHeadNotOnOrigin = %+v, %v; want a refusal that names %q", res, err, fastForward)
	}
	for cmd := range strings.SplitSeq(checkOrigin+", then "+fastForward, ", then ") {
		sh(t, cmd)
	}
	if got := gitOut(t, root, "rev-parse", "HEAD"); got != onTop {
		t.Fatalf("HEAD = %s after the fast-forward the refusal names; want origin's tip %s", got, onTop)
	}
}

// TestRegateCI_GateOnlyRecordsFromThePRsChecks covers AC2 and AC4 of the
// plan on the gate-only path (D4, D6): with HEAD origin's tip of feat/auth,
// containing origin's main, and no pass of it as the latest record, a ci
// regate runs no script, reads PR #7's head and then its checks, and records
// a gate_ci_passed when every check is pass or skipping and one passed (also
// after a gate_ci_failed of HEAD whose checks were run again), or a
// gate_ci_failed with reason ci_failed when one ended and is not pass or
// skipping and none is pending: via=regate_ci, head HEAD, base origin's main
// and its ref, by= and herdr_pane= from the caller's environment, and log= a
// gate log holding the JSON gh pr checks printed. It pushes nothing, and
// merge-order classifies the branch by the record.
func TestRegateCI_GateOnlyRecordsFromThePRsChecks(t *testing.T) {
	const (
		test = "ci / test (fail, FAILURE)"
		e2e  = "ci / e2e (cancel, CANCELLED)"
	)
	for _, tc := range []struct {
		name   string
		setup  func(t *testing.T, fx *rcFixture)
		checks mgReply
		reason string
		failed []string
		by     string // by= and herdr_pane=
		class  string // merge-order's class of feat/auth after the regate
	}{
		{"every check passed", nil, mgReply{out: mxChecks(mxPass("verify"), mxPass("test"))}, "", nil, "by=alice herdr_pane=unset",
			MergeClassReady},
		{"a pass and a skipping", nil, mgReply{out: mxChecks(mxPass("verify"), rcCheck("skipping", "deploy", "SKIPPED"))}, "", nil,
			"by=alice herdr_pane=unset", MergeClassReady},
		{"a pass after a gate_ci_failed of HEAD", func(t *testing.T, fx *rcFixture) {
			fx.appendEvents(t, moCIGate("auth", "feat/auth", fx.head, fx.tip, GateReasonCIFailed))
		}, mgReply{out: mxChecks(mxPass("verify"))}, "", nil, "by=alice herdr_pane=unset", MergeClassReady},
		{"by and herdr_pane from the caller's environment", func(_ *testing.T, fx *rcFixture) {
			fx.env = append(fx.env, "USER=bob", herdrPaneIDEnv+"=pane-9")
		}, mgReply{out: mxChecks(mxPass("verify"))}, "", nil, "by=bob herdr_pane=set", MergeClassReady},
		{"a failed and a cancelled check", nil, mgReply{out: mxChecks(rcCheck("fail", "test", "FAILURE"), mxPass("verify"),
			rcCheck("cancel", "e2e", "CANCELLED")), err: errors.New("exit status 1")}, GateReasonCIFailed, []string{test, e2e},
			"by=alice herdr_pane=unset", MergeClassFailed},
		{"a check of a bucket ralph does not know", nil, mgReply{out: mxChecks(mxPass("verify"), ghPRCheck{Name: "x"})},
			GateReasonCIFailed, []string{"x (no bucket, no state)"}, "by=alice herdr_pane=unset", MergeClassFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRCFixture(t)
			if tc.setup != nil {
				tc.setup(t, fx)
			}
			fx.exec.replies[mxPRChecks] = tc.checks
			before := fx.snapshot(t)
			res, err := fx.regateCI(context.Background())
			want := RegateResult{Outcome: RegateOutcomeGated, Passed: tc.reason == "", Reason: tc.reason, OrgID: "auth", Worktree: fx.worktree,
				Branch: "feat/auth", Head: fx.head, Base: fx.tip, BaseRef: "refs/remotes/origin/main", LogPath: fx.logPath(), PR: 7,
				FailedChecks: tc.failed}
			if err != nil || !reflect.DeepEqual(res, want) {
				t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
			}
			if calls := fx.calls(t); calls != nil {
				t.Fatalf("scripts run = %v; want none", calls)
			}
			fx.assertKeys(t, rcReadKeys)
			for _, c := range fx.exec.snapshot()[2:] {
				argv := map[string][]string{
					mxPRView:   {"pr", "view", "7", "-R", mxRepoArg, "--json", "headRefOid"},
					mxPRChecks: {"pr", "checks", "7", "-R", mxRepoArg, "--json", mergePRChecksFields},
				}[c.key]
				if !slices.Equal(c.args, argv) {
					t.Errorf("%s = gh %q; want gh %q", c.key, c.args, argv)
				}
			}

			event, details := EventGateCIPassed, ""
			if tc.reason != "" {
				event, details = EventGateCIFailed, "reason="+tc.reason+" "
			}
			details += "head=" + fx.head + " base=" + fx.tip + " base_ref=refs/remotes/origin/main branch=feat/auth " + tc.by +
				" via=regate_ci log=" + strconv.Quote(fx.logPath())
			last := lastEvent(t, fx.o)
			if last.Event != event || last.OrgID != "auth" || last.SeatID != "" || last.TS != "2026-10-16T12:00:00Z" || last.Details != details {
				t.Fatalf("last event = %+v\nwant an %s of auth with Details %q", last, event, details)
			}
			if events := fx.gateEvents(t); len(events) != 0 {
				t.Errorf("gate_passed and gate_failed events = %+v; want none", events)
			}
			if log := fileOrAbsent(t, fx.logPath()); log != tc.checks.out {
				t.Errorf("log = %q\nwant the checks' JSON %q", log, tc.checks.out)
			}
			if after := fx.snapshot(t); after.head != before.head || after.status != before.status || after.originRefs != before.originRefs {
				t.Fatalf("a gate-only ci regate changed the worktree or origin:\nbefore %+v\nafter  %+v", before, after)
			}
			if class := fx.mergeClass(t); class != tc.class {
				t.Errorf("merge-order class after the regate = %s; want %s", class, tc.class)
			}
			fx.assertUnlocked(t)
		})
	}
}

// TestRegateCI_GateOnlyRecordsNothingWhileTheChecksAreNotDone covers AC5 of
// the plan on the gate-only path (D4): while PR #7's head is not HEAD yet
// (its checks are then not read), a check is pending, also with another that
// failed, gh reports no checks, as [] or as its sentence, or the PR's head
// moved while its checks were read (self-review L4: read again once checks
// that decide were read), a ci regate records nothing, writes no log, runs no
// script, and returns RegateOutcomeChecksPending with ChecksPending saying
// what it saw and ChecksNext the next step, running it again once CI has
// finished.
func TestRegateCI_GateOnlyRecordsNothingWhileTheChecksAreNotDone(t *testing.T) {
	other := strings.Repeat("e", 40)
	for _, tc := range []struct {
		name  string
		setup func(fx *rcFixture)
		keys  []string
		want  func(fx *rcFixture) string
	}{
		{"a check is pending", func(fx *rcFixture) {
			fx.exec.replies[mxPRChecks] = mgReply{out: mxChecks(mxPass("verify"), rcCheck("pending", "lint", "QUEUED")),
				err: errors.New("exit status 8")}
		}, rcChecksKeys, func(*rcFixture) string { return "1 of the 2 checks of PR #7 are pending: ci / lint (pending, QUEUED)" }},
		{"a check failed and another is pending", func(fx *rcFixture) {
			fx.exec.replies[mxPRChecks] = mgReply{out: mxChecks(rcCheck("fail", "test", "FAILURE"), mxPass("verify"),
				rcCheck("pending", "lint", "IN_PROGRESS")), err: errors.New("exit status 1")}
		}, rcChecksKeys, func(*rcFixture) string {
			return "1 of the 3 checks of PR #7 are pending: ci / lint (pending, IN_PROGRESS); 1 failed: ci / test (fail, FAILURE), " +
				"and regate records the outcome once none is pending"
		}},
		{"no checks, as []", func(fx *rcFixture) { fx.exec.replies[mxPRChecks] = mgReply{out: "[]"} }, rcChecksKeys, func(*rcFixture) string {
			return "gh pr checks 7 -R " + mxRepoArg + " reported no checks yet"
		}},
		{"no checks, as gh's sentence", func(fx *rcFixture) {
			fx.exec.replies[mxPRChecks] = mgReply{err: errors.New("exit status 1: no checks reported on the 'feat/auth' branch")}
		}, rcChecksKeys, func(*rcFixture) string { return "gh pr checks 7 -R " + mxRepoArg + " reported no checks yet" }},
		{"the PR's head is not HEAD yet", func(fx *rcFixture) { fx.exec.replies[mxPRView] = rcPRHead(other) },
			[]string{mxGetURL, mxPRList, mxPRView}, func(fx *rcFixture) string { return rcHeadNotYet(other, fx.head) }},
		{"the PR's head moved while its checks were read", func(fx *rcFixture) {
			fx.exec.after = func(n int) {
				if n == 3 { // gh pr checks, which passed
					fx.reply(mxPRView, rcPRHead(other))
				}
			}
		}, rcReadKeys, func(fx *rcFixture) string {
			return "PR #7's head moved to " + other[:12] + " while its checks were read, so they may not be those of " + fx.head[:12] +
				", origin's tip of feat/auth"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRCFixture(t)
			tc.setup(fx)
			before, manifest := fx.snapshot(t), fileOrAbsent(t, fx.o.Manifest.Path())
			res, err := fx.regateCI(context.Background())
			want := RegateResult{Outcome: RegateOutcomeChecksPending, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth",
				Head: fx.head, Base: fx.tip, BaseRef: "refs/remotes/origin/main", PR: 7, ChecksPending: tc.want(fx), ChecksNext: rcNext}
			if err != nil || !reflect.DeepEqual(res, want) {
				t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
			}
			fx.assertKeys(t, tc.keys)
			fx.assertNothingWritten(t, manifest)
			if after := fx.snapshot(t); after != before {
				t.Fatalf("a ci regate whose checks are not done changed %+v\nto %+v", before, after)
			}
			if class := fx.mergeClass(t); class != MergeClassNoGate {
				t.Errorf("merge-order class after the regate = %s; want no_gate", class)
			}
		})
	}
}

// TestRegateCI_GateOnlyRefusesWhatTheChecksDoNotTell covers the refusals of
// D4 on the gate-only path: every check skipping with none passed, and a PR
// head or checks that cannot be read (gh fails, times out, is not found, or
// prints no JSON), the head read again once the checks were read included,
// record nothing, write no log, and refuse with the reason and the next step,
// keeping the fields read so far with PR set.
func TestRegateCI_GateOnlyRefusesWhatTheChecksDoNotTell(t *testing.T) {
	const view, checks = "gh pr view 7 -R " + mxRepoArg + " --json headRefOid", "gh pr checks 7 -R " + mxRepoArg
	reply := func(key string, r mgReply) func(*rcFixture) {
		return func(fx *rcFixture) { fx.exec.replies[key] = r }
	}
	for _, tc := range []struct {
		name  string
		setup func(fx *rcFixture)
		keys  []string
		want  string
	}{
		{"every check is skipping", reply(mxPRChecks, mgReply{out: mxChecks(rcCheck("skipping", "deploy", "SKIPPED"),
			rcCheck("skipping", "docs", "SKIPPED"))}), rcReadKeys,
			"org: regate: every one of the 2 checks of PR #7 is skipping and none passed, so nothing shows that the PR's CI passed " +
				"and nothing was recorded: let a check run on the PR, and once it passed, " + rcAgain},
		{"gh pr checks fails", reply(mxPRChecks, mgReply{err: errors.New("exit status 1: HTTP 502: Bad Gateway")}), rcChecksKeys,
			"org: regate: " + checks + " failed (exit status 1: HTTP 502: Bad Gateway), so the PR's checks are unknown and nothing " +
				"was recorded: " + rcAccess},
		{"gh pr checks times out", func(fx *rcFixture) { fx.exec.block, fx.ghTimeout = mxPRChecks, 30*time.Millisecond }, rcChecksKeys,
			"org: regate: " + checks + " failed (timed out after 30ms: context deadline exceeded), so the PR's checks are unknown " +
				"and nothing was recorded: " + rcAccess},
		{"gh is not found at gh pr checks", reply(mxPRChecks, mxNotFound), rcChecksKeys,
			"org: regate: " + rcMissing + "nothing was recorded" + rcInstall},
		{"gh pr checks prints no JSON list", reply(mxPRChecks, mgReply{out: "<html>"}), rcChecksKeys,
			"org: regate: " + checks + " printed output that is not a JSON list of checks, so the PR's checks are unknown and " +
				"nothing was recorded: " + rcAccess},
		{"gh pr view fails", reply(mxPRView, mgReply{err: errors.New("exit status 1: HTTP 502: Bad Gateway")}),
			[]string{mxGetURL, mxPRList, mxPRView},
			"org: regate: " + view + " failed (exit status 1: HTTP 502: Bad Gateway), so the PR's head is unknown and nothing was " +
				"recorded: " + rcAccess},
		{"gh pr view prints no headRefOid", reply(mxPRView, mgReply{out: "{}"}), []string{mxGetURL, mxPRList, mxPRView},
			"org: regate: " + view + " failed (its output has no headRefOid), so the PR's head is unknown and nothing was recorded: " +
				rcAccess},
		{"gh pr view fails once the checks were read", func(fx *rcFixture) {
			fx.exec.after = func(n int) {
				if n == 3 { // gh pr checks, which passed
					fx.reply(mxPRView, mgReply{err: errors.New("exit status 1: HTTP 502: Bad Gateway")})
				}
			}
		}, rcReadKeys,
			"org: regate: " + view + " failed (exit status 1: HTTP 502: Bad Gateway), so the PR's head is unknown and nothing was " +
				"recorded: " + rcAccess},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRCFixture(t)
			tc.setup(fx)
			before, manifest := fx.snapshot(t), fileOrAbsent(t, fx.o.Manifest.Path())
			res, err := fx.regateCI(context.Background())
			want := RegateResult{Refused: tc.want, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: fx.head, Base: fx.tip,
				BaseRef: "refs/remotes/origin/main", PR: 7}
			if err != nil || !reflect.DeepEqual(res, want) {
				t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
			}
			fx.assertKeys(t, tc.keys)
			fx.assertNothingWritten(t, manifest)
			if after := fx.snapshot(t); after != before {
				t.Fatalf("a refused ci regate changed %+v\nto %+v", before, after)
			}
		})
	}
}

// TestRegateCI_UpToDateReadsNoChecks covers AC4's up-to-date: with a
// gate_ci_passed or a gate_passed of HEAD on origin's main as the branch's
// latest record, a ci regate stops after the pre-checks, as a local one
// does: RegateOutcomeUpToDate with the record's log, no PR head or checks
// read, nothing run or written.
func TestRegateCI_UpToDateReadsNoChecks(t *testing.T) {
	for _, tc := range []struct {
		name   string
		record func(fx *rcFixture) ManifestEvent
		log    string
	}{
		{"a gate_ci_passed", func(fx *rcFixture) ManifestEvent { return moCIGate("auth", "feat/auth", fx.head, fx.tip, "") }, moRegateCILog},
		{"a gate_passed", func(fx *rcFixture) ManifestEvent {
			return moGate("auth", "feat/auth", fx.head, fx.tip, "refs/remotes/origin/main", "")
		}, "/state/gates/auth.log"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRCFixture(t)
			fx.appendEvents(t, tc.record(fx))
			before, manifest := fx.snapshot(t), fileOrAbsent(t, fx.o.Manifest.Path())
			res, err := fx.regateCI(context.Background())
			want := RegateResult{Outcome: RegateOutcomeUpToDate, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: fx.head,
				Base: fx.tip, BaseRef: "refs/remotes/origin/main", LogPath: tc.log}
			if err != nil || !reflect.DeepEqual(res, want) {
				t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
			}
			fx.assertKeys(t, rcPreCheckKeys)
			fx.assertNothingWritten(t, manifest)
			if after := fx.snapshot(t); after != before {
				t.Fatalf("an up-to-date ci regate changed %+v\nto %+v", before, after)
			}
		})
	}
}

// TestRegateCI_NothingRecordedWhenTheBranchChangedWhileTheChecksWereRead
// covers D6's re-check on a ci regate: when the branch's org starts again
// before the record, nothing is recorded, and the refusal says the branch
// changed while the PR's checks were read and keeps the log of the checks.
func TestRegateCI_NothingRecordedWhenTheBranchChangedWhileTheChecksWereRead(t *testing.T) {
	fx := newRCFixture(t)
	setRegateHook(t, &regateTestHooks.beforeRecord, func() { fx.appendEvents(t, reserveEvent("auth", "internal/other/")) })
	res, err := fx.regateCI(context.Background())
	want := "org: regate: nothing was recorded, since the branch feat/auth changed while the PR's checks were read (org_id " +
		`"auth" is running again): check it with ralph org merge-order and ralph org status, and ` + rcAgain + " once its org is disbanded"
	if err != nil || res.Refused != want || res.Outcome != "" || res.Passed || res.PR != 7 || res.LogPath != fx.logPath() {
		t.Fatalf("regate = %+v, %v; want the record refused with %q and the log kept", res, err, want)
	}
	if log := fileOrAbsent(t, fx.logPath()); log != mxChecks(mxPass("verify")) {
		t.Errorf("log = %q; want the checks' JSON", log)
	}
	if events := fx.ciEvents(t); len(events) != 0 {
		t.Fatalf("ci gate events = %+v; want none", events)
	}
	fx.assertUnlocked(t)
}

// TestRegateCI_Interrupted: a ci regate whose ctx is done once a call
// through Exec returned, or right before the record, returns an error that
// says it was interrupted and what was not done, and records nothing.
func TestRegateCI_Interrupted(t *testing.T) {
	const pre = "org: regate: interrupted (context canceled) while "
	for _, tc := range []struct {
		name  string
		after int // the call (from 0) after which ctx is done; -1 for right before the record
		want  func(fx *rcFixture) string
	}{
		{"once git remote get-url origin returned", 0, func(*rcFixture) string {
			return pre + "git remote get-url origin ran: " + rcDone
		}},
		{"once gh pr list returned", 1, func(*rcFixture) string {
			return pre + "gh pr list --head feat/auth -R " + mxRepoArg + " ran: " + rcDone
		}},
		{"once gh pr view returned", 2, func(*rcFixture) string {
			return pre + "gh pr view 7 -R " + mxRepoArg + " --json headRefOid ran: nothing was recorded"
		}},
		{"once gh pr checks returned", 3, func(*rcFixture) string {
			return pre + "gh pr checks 7 -R " + mxRepoArg + " ran: nothing was recorded"
		}},
		{"once gh pr view returned after the checks", 4, func(*rcFixture) string {
			return pre + "gh pr view 7 -R " + mxRepoArg + " --json headRefOid ran: nothing was recorded"
		}},
		{"right before the record", -1, func(fx *rcFixture) string {
			return "org: regate: interrupted (context canceled) before the record of the PR's checks: nothing was recorded; the " +
				"checks it read are in " + fx.logPath()
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRCFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.after < 0 {
				setRegateHook(t, &regateTestHooks.beforeRecord, cancel)
			} else {
				fx.exec.after = func(n int) {
					if n == tc.after {
						cancel()
					}
				}
			}
			manifest := fileOrAbsent(t, fx.o.Manifest.Path())
			_, err := fx.regateCI(ctx)
			if want := tc.want(fx); err == nil || err.Error() != want {
				t.Fatalf("regate = %v\nwant     %s", err, want)
			}
			if after := fileOrAbsent(t, fx.o.Manifest.Path()); after != manifest {
				t.Fatalf("an interrupted ci regate changed the manifest:\nbefore %s\nafter  %s", manifest, after)
			}
			if got := fx.exec.keys(); tc.after >= 0 && len(got) != tc.after+1 {
				t.Errorf("calls = %q; want none after call %d", got, tc.after)
			}
			fx.assertUnlocked(t)
		})
	}
}

// TestRegateCI_RefusesAnUnknownChecksValue: a Checks that is neither local
// nor ci is refused before anything else, with no call through Exec.
func TestRegateCI_RefusesAnUnknownChecksValue(t *testing.T) {
	fx := newRCFixture(t)
	p := fx.params()
	p.Checks = "remote"
	before := fx.snapshot(t)
	res, err := fx.o.Regate(context.Background(), p)
	fx.assertRefused(t, res, err, before, []string{`org: regate: [org.gate].regate_checks is "remote", not "local" or "ci", so ` +
		"nothing was done: set it to one of them and " + rcAgain}, nil)
	fx.assertKeys(t, nil)
}

// TestRegateCI_LocalChecksRunTheScriptsAndCallNoExec covers AC1 of the plan
// with Exec set: with Checks "" or local, Regate runs the gate scripts and
// records a gate_passed via=regate as before, and calls nothing through Exec.
func TestRegateCI_LocalChecksRunTheScriptsAndCallNoExec(t *testing.T) {
	for _, checks := range []string{"", config.RegateChecksLocal} {
		t.Run(strconv.Quote(checks), func(t *testing.T) {
			fx := newRCFixture(t)
			p := fx.params()
			p.Checks = checks
			res, err := fx.o.Regate(context.Background(), p)
			want := RegateResult{Outcome: RegateOutcomeGated, Passed: true, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth",
				Head: fx.head, Base: fx.tip, BaseRef: "refs/remotes/origin/main", LogPath: fx.logPath()}
			if err != nil || !reflect.DeepEqual(res, want) {
				t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
			}
			if calls := fx.calls(t); !slices.Equal(calls, []string{"static", "test"}) {
				t.Fatalf("scripts run = %v; want static then test", calls)
			}
			if last := lastEvent(t, fx.o); last.Event != EventGatePassed || !strings.Contains(last.Details, " via=regate log=") {
				t.Fatalf("last event = %+v; want a gate_passed via regate", last)
			}
			fx.assertKeys(t, nil)
		})
	}
}
