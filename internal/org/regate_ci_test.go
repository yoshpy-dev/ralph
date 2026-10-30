package org

import (
	"context"
	"errors"
	"io"
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

// reply sets mxExec's reply under key, under its lock, for a hook that runs
// while Regate runs.
func (fx *rcFixture) reply(key string, r mgReply) {
	fx.exec.mu.Lock()
	defer fx.exec.mu.Unlock()
	fx.exec.replies[key] = r
}

// replyHeadOnPush makes gh pr view answer the worktree's HEAD, the merge
// commit, once a push origin took has returned (regateTestHooks.afterPush):
// GitHub has updated PR #7 to it.
func (fx *rcFixture) replyHeadOnPush(t *testing.T) {
	t.Helper()
	setRegateHook(t, &regateTestHooks.afterPush, func() { fx.reply(mxPRView, rcPRHead(gitOut(t, fx.worktree, "rev-parse", "HEAD"))) })
}

// worktreeHead returns the feature worktree's HEAD.
func (fx *rcFixture) worktreeHead(t *testing.T) string {
	t.Helper()
	return gitOut(t, fx.worktree, "rev-parse", "HEAD")
}

// assertOnOrigin fails the test unless HEAD and origin's feat/auth are both
// merge.
func (fx *rcFixture) assertOnOrigin(t *testing.T, merge string) {
	t.Helper()
	if head, got := fx.worktreeHead(t), fx.originBranch(t); head != merge || got != merge {
		t.Fatalf("HEAD = %s, origin's feat/auth = %s; want both the pushed merge commit %s", head, got, merge)
	}
}

// rcCheckOrigin is the next step of a refusal or an error after a push whose
// landing cannot be told.
func (fx *rcFixture) rcCheckOrigin() string {
	return "check origin and HEAD (git -C " + fx.worktree + " fetch origin, then git -C " + fx.worktree + " log --oneline --graph " +
		"HEAD refs/remotes/origin/feat/auth) and " + rcAgain
}

// TestRegateCI_MergePathPushesTheMergeCommitBeforeTheChecks covers AC2 and
// AC5 of the plan on the merge path (D2 steps 2 to 4, D6): with origin's main
// past HEAD, a ci regate makes D3's merge commit, runs no script, pushes the
// merge commit to origin's feat/auth without force before anything is
// recorded, and then reads PR #7's head, which right after the push is still
// the branch's old head: it records nothing, writes no log, and returns
// RegateOutcomeChecksPending with the merge commit pushed, kept as HEAD, and
// in refs/remotes/origin/feat/auth. Run again once GitHub has updated the PR,
// it finds HEAD on origin's tip and takes the gate-only path: a
// gate_ci_passed of the merge commit on origin's main, and nothing merged or
// pushed again.
func TestRegateCI_MergePathPushesTheMergeCommitBeforeTheChecks(t *testing.T) {
	fx := newRCFixture(t)
	pre := fx.head
	tip := fx.pushToOrigin(t, "main")
	manifest := fileOrAbsent(t, fx.o.Manifest.Path())
	res, err := fx.regateCI(context.Background())
	merge := fx.worktreeHead(t)
	want := RegateResult{Outcome: RegateOutcomeChecksPending, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: merge,
		Base: tip, BaseRef: "refs/remotes/origin/main", Merge: merge, PreMergeHead: pre, Pushed: true, PR: 7,
		ChecksPending: rcHeadNotPushed(pre, merge), ChecksNext: rcNext}
	if err != nil || !reflect.DeepEqual(res, want) {
		t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
	}
	fx.assertMergeCommit(t, merge, pre, tip)
	fx.assertOnOrigin(t, merge)
	if got := gitOut(t, fx.worktree, "rev-parse", "refs/remotes/origin/feat/auth"); got != merge {
		t.Errorf("refs/remotes/origin/feat/auth = %s; want the pushed %s", got, merge)
	}
	if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Errorf("the worktree after the regate:\n%s", st)
	}
	fx.assertKeys(t, []string{mxGetURL, mxPRList, mxPRView})
	fx.assertNothingWritten(t, manifest)
	if class := fx.mergeClass(t); class != MergeClassNoGate {
		t.Errorf("merge-order class after the regate = %s; want no_gate", class)
	}

	fx.reply(mxPRView, rcPRHead(merge))
	res, err = fx.regateCI(context.Background())
	want = RegateResult{Outcome: RegateOutcomeGated, Passed: true, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: merge,
		Base: tip, BaseRef: "refs/remotes/origin/main", LogPath: fx.logPath(), PR: 7}
	if err != nil || !reflect.DeepEqual(res, want) {
		t.Fatalf("regate run again = %#v, %v\nwant           %#v", res, err, want)
	}
	if last := lastEvent(t, fx.o); last.Event != EventGateCIPassed || !strings.HasPrefix(last.Details, "head="+merge+" base="+tip+" ") {
		t.Fatalf("last event = %+v; want a gate_ci_passed of the merge commit %s on %s", last, merge, tip)
	}
	fx.assertOnOrigin(t, merge)
	if calls := fx.calls(t); calls != nil {
		t.Fatalf("scripts run = %v; want none", calls)
	}
	if class := fx.mergeClass(t); class != MergeClassReady {
		t.Errorf("merge-order class after the second regate = %s; want ready", class)
	}
}

// TestRegateCI_MergePathRecordsTheChecksOfThePushedMergeCommit covers AC2 and
// AC4 of the plan on the merge path (D4): when PR #7's head is the pushed
// merge commit once the push returned, a ci regate reads its checks and
// records a gate_ci_passed, or a gate_ci_failed with reason ci_failed, of the
// merge commit on origin's main (head the pushed SHA, base and base_ref
// origin's main), RegateOutcomeMerged, with no script run. The merge commit
// stays on origin and as HEAD after a gate_ci_failed too.
func TestRegateCI_MergePathRecordsTheChecksOfThePushedMergeCommit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		checks mgReply
		reason string
		failed []string
		class  string
	}{
		{"every check passed", mgReply{out: mxChecks(mxPass("verify"), mxPass("test"))}, "", nil, MergeClassReady},
		{"a check failed", mgReply{out: mxChecks(rcCheck("fail", "test", "FAILURE"), mxPass("verify")), err: errors.New("exit status 1")},
			GateReasonCIFailed, []string{"ci / test (fail, FAILURE)"}, MergeClassFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRCFixture(t)
			pre := fx.head
			tip := fx.pushToOrigin(t, "main")
			fx.exec.replies[mxPRChecks] = tc.checks
			fx.replyHeadOnPush(t)
			res, err := fx.regateCI(context.Background())
			merge := fx.worktreeHead(t)
			want := RegateResult{Outcome: RegateOutcomeMerged, Passed: tc.reason == "", Reason: tc.reason, OrgID: "auth", Worktree: fx.worktree,
				Branch: "feat/auth", Head: merge, Base: tip, BaseRef: "refs/remotes/origin/main", Merge: merge, PreMergeHead: pre,
				LogPath: fx.logPath(), Pushed: true, PR: 7, FailedChecks: tc.failed}
			if err != nil || !reflect.DeepEqual(res, want) {
				t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
			}
			fx.assertMergeCommit(t, merge, pre, tip)
			fx.assertOnOrigin(t, merge)
			fx.assertKeys(t, rcReadKeys)
			if args := fx.exec.snapshot()[2].args; !slices.Equal(args, []string{"pr", "view", "7", "-R", mxRepoArg, "--json", "headRefOid"}) {
				t.Errorf("gh pr view = gh %q", args)
			}
			if calls := fx.calls(t); calls != nil {
				t.Fatalf("scripts run = %v; want none", calls)
			}
			event, details := EventGateCIPassed, ""
			if tc.reason != "" {
				event, details = EventGateCIFailed, "reason="+tc.reason+" "
			}
			details += "head=" + merge + " base=" + tip + " base_ref=refs/remotes/origin/main branch=feat/auth by=alice herdr_pane=unset " +
				"via=regate_ci log=" + strconv.Quote(fx.logPath())
			if events := fx.ciEvents(t); len(events) != 1 || events[0].Event != event || events[0].OrgID != "auth" || events[0].Details != details {
				t.Fatalf("ci gate events = %+v\nwant one %s of auth with Details %q", events, event, details)
			}
			if events := fx.gateEvents(t); len(events) != 0 {
				t.Errorf("gate_passed and gate_failed events = %+v; want none", events)
			}
			if log := fileOrAbsent(t, fx.logPath()); log != tc.checks.out {
				t.Errorf("log = %q\nwant the checks' JSON %q", log, tc.checks.out)
			}
			if class := fx.mergeClass(t); class != tc.class {
				t.Errorf("merge-order class after the regate = %s; want %s", class, tc.class)
			}
			fx.assertUnlocked(t)
		})
	}
}

// TestRegateCI_ARefusalAfterThePushSaysTheMergeCommitStays: once the merge
// commit is pushed, a refusal of the PR's head (gh pr view fails) or of the
// record (the org started again while the checks were read) records nothing,
// undoes nothing, and says that the merge commit was pushed and stays on
// origin, keeping Merge, PreMergeHead, and Pushed.
func TestRegateCI_ARefusalAfterThePushSaysTheMergeCommitStays(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, fx *rcFixture)
		want  string // the refusal, before what it says of the merge commit
		log   bool
	}{
		{"gh pr view fails", func(_ *testing.T, fx *rcFixture) {
			fx.exec.replies[mxPRView] = mgReply{err: errors.New("exit status 1: HTTP 502: Bad Gateway")}
		}, "org: regate: gh pr view 7 -R " + mxRepoArg + " --json headRefOid failed (exit status 1: HTTP 502: Bad Gateway), so the PR's " +
			"head is unknown and nothing was recorded: " + rcAccess, false},
		{"the org starts again while the checks are read", func(t *testing.T, fx *rcFixture) {
			fx.replyHeadOnPush(t)
			setRegateHook(t, &regateTestHooks.beforeRecord, func() { fx.appendEvents(t, reserveEvent("auth", "internal/other/")) })
		}, "org: regate: nothing was recorded, since the branch feat/auth changed while the PR's checks were read (org_id " +
			`"auth" is running again): check it with ralph org merge-order and ralph org status, and ` + rcAgain + " once its org is " +
			"disbanded", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRCFixture(t)
			pre := fx.head
			tip := fx.pushToOrigin(t, "main")
			tc.setup(t, fx)
			res, err := fx.regateCI(context.Background())
			merge := fx.worktreeHead(t)
			want := RegateResult{Refused: tc.want + "; the merge commit " + merge + " was pushed to origin's feat/auth and stays there",
				OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: merge, Base: tip, BaseRef: "refs/remotes/origin/main",
				Merge: merge, PreMergeHead: pre, Pushed: true, PR: 7}
			if tc.log {
				want.LogPath = fx.logPath()
			}
			if err != nil || !reflect.DeepEqual(res, want) {
				t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
			}
			fx.assertOnOrigin(t, merge)
			if events := fx.ciEvents(t); len(events) != 0 {
				t.Fatalf("ci gate events = %+v; want none", events)
			}
			fx.assertUnlocked(t)
		})
	}
}

// TestRegateCI_AFailedPushIsDecidedByOriginsTipReadBack covers AC7 of the
// plan (D3): when the ci regate's push of its merge commit fails, Regate
// reads origin's tip of feat/auth back from origin, and
//
//   - when it is still HEAD before the merge, the push did not land: Regate
//     undoes the merge commit, records nothing, and refuses with what git
//     said and to fix it, for a push origin refused and one a signal ended;
//   - when it is the merge commit (the push landed and its answer was lost;
//     here the test pushes it right before the read-back), Regate goes on to
//     the PR's checks and records a gate_ci_passed of the merge commit;
//   - when it is a descendant of the merge commit or another commit, when
//     origin has no such branch, or when it cannot be read, whether origin
//     has the merge commit cannot be told: Regate keeps HEAD, the merge
//     commit, records nothing, and refuses naming what it read and how to
//     check origin and HEAD.
//
// It reads the PR's head and checks only once origin has the merge commit,
// and runs no script.
func TestRegateCI_AFailedPushIsDecidedByOriginsTipReadBack(t *testing.T) {
	const (
		undone = "undone"
		landed = "landed"
		kept   = "kept"
	)
	for _, tc := range []struct {
		name   string
		signal bool // the push's pre-push hook kills git, instead of origin refusing the push
		// readBack runs right before the read-back, with origin taking pushes
		// again, and returns what the refusal says was read (kept).
		readBack func(t *testing.T, fx *rcFixture, merge, pre string) string
		outcome  string
	}{
		{"origin's tip is still the head merged onto", false, nil, undone},
		{"a push ended by a signal, origin's tip still the head merged onto", true, nil, undone},
		{"origin's tip is the pushed merge commit", false, func(t *testing.T, fx *rcFixture, merge, _ string) string {
			runGit(t, "-C", fx.worktree, "push", "--quiet", "origin", "HEAD:refs/heads/feat/auth")
			fx.reply(mxPRView, rcPRHead(merge))
			return ""
		}, landed},
		{"origin's tip is a descendant of the merge commit", false, func(t *testing.T, fx *rcFixture, merge, pre string) string {
			runGit(t, "-C", fx.worktree, "push", "--quiet", "origin", "HEAD:refs/heads/feat/auth")
			newer := fx.pushToOrigin(t, "feat/auth")
			return "origin's feat/auth is now " + newer + ", neither the tip " + pre + " the merge commit was made on nor the merge commit"
		}, kept},
		{"origin's tip is another commit", false, func(t *testing.T, fx *rcFixture, _, pre string) string {
			under := gitOut(t, fx.worktree, "rev-parse", pre+"^")
			runGit(t, "-C", fx.origin, "update-ref", "refs/heads/feat/auth", under)
			return "origin's feat/auth is now " + under + ", neither the tip " + pre + " the merge commit was made on nor the merge commit"
		}, kept},
		{"origin has no branch", false, func(t *testing.T, fx *rcFixture, _, _ string) string {
			runGit(t, "-C", fx.origin, "update-ref", "-d", "refs/heads/feat/auth")
			return "origin has no branch feat/auth now"
		}, kept},
		{"origin's tip cannot be read", false, func(t *testing.T, fx *rcFixture, _, _ string) string {
			runGit(t, "-C", fx.main, "remote", "set-url", "origin", filepath.Join(fx.out, "missing.git"))
			return "origin's tip of feat/auth could not be read back (git ls-remote origin refs/heads/feat/auth in " + fx.worktree +
				" failed (exit 128))"
		}, kept},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRCFixture(t)
			pre := fx.head
			tip := fx.pushToOrigin(t, "main")
			allow := func() {}
			if tc.signal {
				fx.setHooks(t, "pre-push", "kill -KILL $PPID\n")
			} else {
				allow = fx.rejectPushes(t)
			}
			var merge, read string
			readBacks := 0
			setRegateHook(t, &regateTestHooks.beforeReadBack, func() {
				readBacks++
				merge = fx.worktreeHead(t)
				if tc.readBack != nil {
					allow()
					read = tc.readBack(t, fx, merge, pre)
				}
			})
			manifest := fileOrAbsent(t, fx.o.Manifest.Path())
			res, err := fx.regateCI(context.Background())
			if err != nil || readBacks != 1 || merge == "" {
				t.Fatalf("regate = %+v, %v after %d read-backs; want no error and one read-back of a merge commit", res, err, readBacks)
			}
			failed := "org: regate: git push origin " + merge + ":refs/heads/feat/auth in " + fx.worktree + " failed (exit 1), and "
			if tc.signal {
				failed = "org: regate: git push origin " + merge + ":refs/heads/feat/auth in " + fx.worktree + " was ended by a signal " +
					"(killed), and "
			}
			want := RegateResult{Refused: res.Refused, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: merge, Base: tip,
				BaseRef: "refs/remotes/origin/main", Merge: merge, PreMergeHead: pre}
			var prefix, suffix string
			switch tc.outcome {
			case undone:
				want.Undone = true
				prefix = failed + "origin's feat/auth is still " + pre + ", the tip the merge commit was made on, so the push did not land " +
					"and nothing was recorded: fix it and " + rcAgain
				suffix = "; " + regateUndoneTextFor(merge, pre)
			case kept:
				prefix = failed + read + ", so whether origin has " + merge + " cannot be told, and nothing was recorded: HEAD, the merge " +
					"commit, is kept; " + fx.rcCheckOrigin()
			case landed:
				want = RegateResult{Outcome: RegateOutcomeMerged, Passed: true, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth",
					Head: merge, Base: tip, BaseRef: "refs/remotes/origin/main", Merge: merge, PreMergeHead: pre, LogPath: fx.logPath(),
					Pushed: true, PR: 7}
			}
			if !reflect.DeepEqual(res, want) {
				t.Fatalf("regate = %#v\nwant     %#v", res, want)
			}
			if !strings.HasPrefix(res.Refused, prefix) || !strings.HasSuffix(res.Refused, suffix) {
				t.Errorf("refusal = %q\nwant it to start with %q and end with %q", res.Refused, prefix, suffix)
			}
			if tc.outcome != landed && !tc.signal && (!strings.Contains(res.Refused, "; git said:\n  ") ||
				!strings.Contains(res.Refused, "pushes are closed")) {
				t.Errorf("refusal %q does not quote what git said of the push", res.Refused)
			}
			if calls := fx.calls(t); calls != nil {
				t.Fatalf("scripts run = %v; want none", calls)
			}
			switch tc.outcome {
			case undone:
				if head, got := fx.worktreeHead(t), fx.originBranch(t); head != pre || got != pre || fx.undoneMerge(t) != merge {
					t.Fatalf("HEAD = %s, origin's feat/auth = %s; want both %s, the merge commit %s undone", head, got, pre, merge)
				}
				fx.assertKeys(t, rcPreCheckKeys)
				fx.assertNothingWritten(t, manifest)
			case kept:
				if head := fx.worktreeHead(t); head != merge {
					t.Fatalf("HEAD = %s; want the merge commit %s kept", head, merge)
				}
				fx.assertKeys(t, rcPreCheckKeys)
				fx.assertNothingWritten(t, manifest)
			case landed:
				fx.assertOnOrigin(t, merge)
				fx.assertKeys(t, rcReadKeys)
				if events := fx.ciEvents(t); len(events) != 1 || events[0].Event != EventGateCIPassed ||
					!strings.HasPrefix(events[0].Details, "head="+merge+" base="+tip+" ") {
					t.Fatalf("ci gate events = %+v; want a gate_ci_passed of the merge commit %s on %s", events, merge, tip)
				}
			}
			if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
				t.Errorf("the worktree after the regate:\n%s", st)
			}
			fx.assertUnlocked(t)
		})
	}
}

// TestRegateCI_AnInterruptOnTheMergePath covers AC5's interrupt on the merge
// path (D3): an interrupt while the merge runs, before the push, undoes the
// merge commit and records nothing; one while the push runs keeps HEAD, the
// merge commit, and records nothing, since origin may or may not have it; one
// after the push landed, right after it or while the PR's head is read,
// records nothing and says the merge commit stays on origin and the next
// regate reads its checks, which, run again, it does on the gate-only path.
func TestRegateCI_AnInterruptOnTheMergePath(t *testing.T) {
	type outcome struct {
		res RegateResult
		err error
	}
	run := func(ctx context.Context, fx *rcFixture) <-chan outcome {
		done := make(chan outcome, 1)
		go func() {
			p := fx.params()
			p.Stderr = io.Discard
			res, err := fx.o.Regate(ctx, p)
			done <- outcome{res, err}
		}()
		return done
	}
	wait := func(t *testing.T, done <-chan outcome) outcome {
		t.Helper()
		select {
		case got := <-done:
			return got
		case <-time.After(10 * time.Second):
			t.Fatal("regate did not return within 10s of the interrupt")
		}
		return outcome{}
	}
	// blocking is a hook body that waits for release, or for git, its parent,
	// to be gone, after it made started.
	blocking := func(fx *rcFixture) (body, started, release string) {
		started, release = filepath.Join(fx.out, "started"), filepath.Join(fx.out, "release")
		return "touch '" + started + "'\nwhile [ ! -f '" + release + "' ] && kill -0 $PPID 2>/dev/null; do sleep 0.05; done\n", started, release
	}

	t.Run("while the merge runs, before the push", func(t *testing.T) {
		fx := newRCFixture(t)
		pre := fx.head
		fx.pushToOrigin(t, "main")
		body, started, release := blocking(fx)
		fx.setHooks(t, "pre-merge-commit", body)
		t.Cleanup(func() { _ = os.WriteFile(release, nil, 0o644) })
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := run(ctx, fx)
		waitForFile(t, started)
		cancel()
		if err := os.WriteFile(release, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		got := wait(t, done)
		merge := fx.undoneMerge(t)
		if want := "org: regate: interrupted (context canceled) before " + merge + " was pushed to origin's feat/auth, so nothing was " +
			"pushed or recorded; " + regateUndoneTextFor(merge, pre); got.err == nil || got.err.Error() != want {
			t.Fatalf("regate = %v\nwant     %s", got.err, want)
		}
		if !got.res.Undone || got.res.Merge != merge || got.res.Pushed {
			t.Fatalf("regate = %+v; want the merge commit %s undone, not pushed", got.res, merge)
		}
		if head, origin := fx.worktreeHead(t), fx.originBranch(t); head != pre || origin != pre {
			t.Fatalf("HEAD = %s, origin's feat/auth = %s; want both %s", head, origin, pre)
		}
		fx.assertKeys(t, rcPreCheckKeys)
		fx.assertNothingWritten(t, manifest)
	})

	t.Run("while the push runs", func(t *testing.T) {
		fx := newRCFixture(t)
		pre := fx.head
		fx.pushToOrigin(t, "main")
		body, started, release := blocking(fx)
		fx.setHooks(t, "pre-push", body+"exit 1\n")
		t.Cleanup(func() { _ = os.WriteFile(release, nil, 0o644) })
		setRegateHook(t, &regateTestHooks.beforeReadBack, func() { t.Error("regate read origin's tip back after an interrupt") })
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := run(ctx, fx)
		waitForFile(t, started)
		cancel()
		got := wait(t, done)
		merge := fx.worktreeHead(t)
		if want := "org: regate: interrupted (context canceled) while " + merge + " was pushed to origin's feat/auth, so origin may or " +
			"may not have " + merge + ", and nothing was recorded: HEAD, the merge commit, is kept; " + fx.rcCheckOrigin(); got.err == nil ||
			got.err.Error() != want {
			t.Fatalf("regate = %v\nwant     %s", got.err, want)
		}
		if got.res.Merge != merge || got.res.PreMergeHead != pre || got.res.Undone || got.res.Pushed {
			t.Fatalf("regate = %+v; want the merge commit %s kept, not undone", got.res, merge)
		}
		fx.assertMergeCommit(t, merge, pre, got.res.Base)
		if origin := fx.originBranch(t); origin != pre {
			t.Fatalf("origin's feat/auth = %s; want %s, which the hook kept the push from changing", origin, pre)
		}
		fx.assertKeys(t, rcPreCheckKeys)
		fx.assertNothingWritten(t, manifest)
	})

	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, fx *rcFixture, cancel func())
		keys  []string
		want  func(merge string) string
	}{
		{"right after the push landed", func(t *testing.T, fx *rcFixture, cancel func()) {
			setRegateHook(t, &regateTestHooks.afterPush, cancel)
		}, rcPreCheckKeys, func(merge string) string {
			return "org: regate: interrupted (context canceled) after " + merge + " was pushed to origin's feat/auth, so nothing was " +
				"recorded: the merge commit stays on origin and HEAD is kept; " + rcAgain + ", which reads the PR's checks of it"
		}},
		{"once gh pr view returned after the push", func(_ *testing.T, fx *rcFixture, cancel func()) {
			fx.exec.after = func(n int) {
				if n == 2 {
					cancel()
				}
			}
		}, []string{mxGetURL, mxPRList, mxPRView}, func(merge string) string {
			return "org: regate: interrupted (context canceled) while gh pr view 7 -R " + mxRepoArg + " --json headRefOid ran: nothing " +
				"was recorded; the merge commit " + merge + " was pushed to origin's feat/auth and stays there: " + rcAgain + ", which " +
				"reads the PR's checks of it"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRCFixture(t)
			pre := fx.head
			tip := fx.pushToOrigin(t, "main")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			tc.setup(t, fx, cancel)
			manifest := fileOrAbsent(t, fx.o.Manifest.Path())
			res, err := fx.regateCI(ctx)
			merge := fx.worktreeHead(t)
			if want := tc.want(merge); err == nil || err.Error() != want {
				t.Fatalf("regate = %v\nwant     %s", err, want)
			}
			if res.Merge != merge || res.PreMergeHead != pre || !res.Pushed || res.Undone || res.Outcome != "" {
				t.Fatalf("regate = %+v; want the merge commit %s pushed and kept, with no outcome", res, merge)
			}
			fx.assertOnOrigin(t, merge)
			fx.assertKeys(t, tc.keys)
			fx.assertNothingWritten(t, manifest)

			regateTestHooks.afterPush, fx.exec.after = nil, nil
			fx.reply(mxPRView, rcPRHead(merge))
			res, err = fx.regateCI(context.Background())
			if err != nil || res.Outcome != RegateOutcomeGated || !res.Passed || res.Head != merge || res.Merge != "" || res.Pushed {
				t.Fatalf("regate run again = %+v, %v; want a gate-only pass of the merge commit %s", res, err, merge)
			}
			if last := lastEvent(t, fx.o); last.Event != EventGateCIPassed || !strings.HasPrefix(last.Details, "head="+merge+" base="+tip+" ") {
				t.Fatalf("last event = %+v; want a gate_ci_passed of the merge commit %s on %s", last, merge, tip)
			}
		})
	}
}

// TestRegateReadOriginTip covers D3's read-back of origin's tip of the
// branch from origin itself: the branch, not a ref whose tail merely matches
// git ls-remote's pattern; none when origin has no such branch; why when
// origin cannot be read; and an error for ctx done.
func TestRegateReadOriginTip(t *testing.T) {
	fx := newRegateFixture(t)
	head, tip := fx.tips(t)
	target := gateTarget{root: fx.worktree, branch: "feat/auth"}
	runGit(t, "-C", fx.origin, "update-ref", "refs/heads/x/refs/heads/feat/auth", tip)
	read := func(ctx context.Context) (string, string, error) { return regateReadOriginTip(ctx, target, fx.env) }

	if got, unread, err := read(context.Background()); err != nil || unread != "" || got != head {
		t.Fatalf("read-back = %q, %q, %v; want feat/auth's %s", got, unread, err, head)
	}
	runGit(t, "-C", fx.origin, "update-ref", "-d", "refs/heads/feat/auth")
	if got, unread, err := read(context.Background()); err != nil || unread != "" || got != "" {
		t.Fatalf("read-back with only refs/heads/x/refs/heads/feat/auth on origin = %q, %q, %v; want no tip", got, unread, err)
	}
	runGit(t, "-C", fx.main, "remote", "set-url", "origin", filepath.Join(fx.out, "missing.git"))
	want := "git ls-remote origin refs/heads/feat/auth in " + fx.worktree + " failed (exit 128)"
	if got, unread, err := read(context.Background()); err != nil || unread != want || got != "" {
		t.Fatalf("read-back of a missing origin = %q, %q, %v; want %q", got, unread, err, want)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, unread, err := read(ctx); err == nil || unread != "" || got != "" {
		t.Fatalf("read-back with ctx done = %q, %q, %v; want an error", got, unread, err)
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

// The wait of a ci regate for the PR's checks (plan
// docs/plans/active/2026-10-29-regate-ci-checks.md, S5: D5, AC5):
// RegateParams.WaitChecks on rcFixture, with mwClock (merge_fake_test.go) in
// RegateParams.waitTest, so that no test sleeps for the 30 s interval, and
// mxExec's answers changed while Regate sleeps.

// rcWaitFixture is rcFixture for a ci regate with WaitChecks wait: a fake
// clock whose every sleep takes regate.lock, empties its marker, as a merge
// that takes the lock does, and releases it (lockTaken counts them) before
// onSleep; a ctx the test stops; Stderr kept; and, for every gh pr view and
// gh pr checks, whether regate.lock was held while it ran, in reads ("L"
// held, "." free), before afterCall of that call's index.
type rcWaitFixture struct {
	*rcFixture
	clock     *mwClock
	ctx       context.Context
	cancel    context.CancelCauseFunc
	stderr    *strings.Builder
	wait      time.Duration
	onSleep   func(n int)
	afterCall func(n int)
	lockTaken int
	reads     string
}

func newRCWaitFixture(t *testing.T, wait time.Duration) *rcWaitFixture {
	t.Helper()
	fx := &rcWaitFixture{rcFixture: newRCFixture(t), stderr: &strings.Builder{}, wait: wait}
	fx.ctx, fx.cancel = context.WithCancelCause(context.Background())
	t.Cleanup(func() { fx.cancel(nil) })
	fx.clock = &mwClock{t: t, now: time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC), stop: fx.cancel, onSleep: func(n int) {
		lock, err := acquireHeldFileLock(fx.stateDir, regateLockFile, "test", 0)
		if err != nil {
			t.Errorf("regate.lock while Regate waits: %v; want it free", err)
		} else {
			fx.lockTaken++
			if err := writeRegateMarker(lock.f, nil); err != nil {
				t.Error(err)
			}
			_ = lock.release()
		}
		if fx.onSleep != nil {
			fx.onSleep(n)
		}
	}}
	fx.exec.after = func(n int) {
		if key := fx.exec.snapshot()[n].key; key == mxPRView || key == mxPRChecks {
			held, err := fileLockHeld(regateLockPathIn(fx.stateDir))
			switch {
			case err != nil:
				t.Error(err)
			case held:
				fx.reads += "L"
			default:
				fx.reads += "."
			}
		}
		if fx.afterCall != nil {
			fx.afterCall(n)
		}
	}
	return fx
}

// assertReads fails the test unless reads is want: one character per gh pr
// view and gh pr checks, "L" while regate.lock was held, "." while it was
// free.
func (fx *rcWaitFixture) assertReads(t *testing.T, want string) {
	t.Helper()
	if fx.reads != want {
		t.Errorf("regate.lock at the PR's reads = %q; want %q (L held, . free)", fx.reads, want)
	}
}

// assertNotTakenAgain fails the test unless regate.lock's marker is empty, as
// the fixture's sleeps leave it: Regate did not take the lock again after its
// last sleep.
func (fx *rcWaitFixture) assertNotTakenAgain(t *testing.T) {
	t.Helper()
	if org, ok := readRegateMarker(regateLockPathIn(fx.stateDir)); ok {
		t.Errorf("the marker after the regate names %q; want it empty, the lock not taken again", org)
	}
}

// run is the fixture's ci regate with WaitChecks, the fake clock, and Stderr.
func (fx *rcWaitFixture) run() (RegateResult, error) {
	p := fx.params()
	p.WaitChecks, p.Stderr, p.waitTest = fx.wait, fx.stderr, mergeWaitTest{now: fx.clock.Now, sleep: fx.clock.sleep}
	return fx.o.Regate(fx.ctx, p)
}

// assertWaited fails the test unless the sleeps were sleeps, regate.lock was
// free in every one and is free now, and Stderr is lines.
func (fx *rcWaitFixture) assertWaited(t *testing.T, sleeps []time.Duration, lines []string) {
	t.Helper()
	if !slices.Equal(fx.clock.sleeps, sleeps) {
		t.Errorf("sleeps = %v, want %v", fx.clock.sleeps, sleeps)
	}
	if fx.lockTaken != len(fx.clock.sleeps) {
		t.Errorf("regate.lock was taken in %d of the %d sleeps; want every one", fx.lockTaken, len(fx.clock.sleeps))
	}
	var got []string
	for line := range strings.Lines(fx.stderr.String()) {
		got = append(got, strings.TrimSuffix(line, "\n"))
	}
	if !slices.Equal(got, lines) {
		t.Errorf("Stderr =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(lines, "\n"))
	}
	fx.assertUnlocked(t)
}

// assertLockedAtTheRecord fails the test unless, right before the record,
// regate.lock is held again and its marker names auth again, which the
// fixture's sleeps emptied.
func (fx *rcWaitFixture) assertLockedAtTheRecord(t *testing.T) {
	t.Helper()
	records := 0
	setRegateHook(t, &regateTestHooks.beforeRecord, func() {
		records++
		path := regateLockPathIn(fx.stateDir)
		if held, err := fileLockHeld(path); err != nil || !held {
			t.Errorf("regate.lock held = %v (%v) right before the record; want it taken again", held, err)
		}
		if org, ok := readRegateMarker(path); !ok || org != "auth" {
			t.Errorf("the marker right before the record = %q, %v; want auth written again", org, ok)
		}
	})
	t.Cleanup(func() {
		if records != 1 {
			t.Errorf("records = %d; want 1", records)
		}
	})
}

// The answers and texts of the waits in these tests.
var rcPending = mgReply{out: mxChecks(mxPass("verify"), rcCheck("pending", "lint", "QUEUED")), err: errors.New("exit status 8")}

const rcPendingWhy = "1 of the 2 checks of PR #7 are pending: ci / lint (pending, QUEUED)"

// rcWaitStart is the line a wait writes when it starts, why being
// RegateResult.ChecksPending then.
func rcWaitStart(why, budget string) string {
	return "org: regate: " + why + ", so ralph org regate waits for the checks of PR #7 for up to " + budget + " (--wait-checks " +
		budget + "), with regate.lock released, reading the PR's head and its checks every 30s"
}

// rcWaitEnd is the line a wait writes when it ends.
func rcWaitEnd(after, why string) string {
	return "org: regate: stopped waiting for the checks of PR #7 after " + after + ": " + why
}

// rcWaitInterrupted is the error of a wait that ctx stopped.
const rcWaitInterrupted = "org: regate: interrupted (test stopped) while waiting for the checks of PR #7: nothing was recorded"

// rcHeadNotYet is RegateResult.ChecksPending on the gate-only path while PR
// #7's head is was, not HEAD head (self-review L4: what was seen, with both
// reasons it can have).
func rcHeadNotYet(was, head string) string {
	return "PR #7's head is " + was[:12] + ", not " + head[:12] + ", origin's tip of feat/auth: either GitHub has not updated the PR " +
		"yet, or origin's feat/auth moved after regate read it"
}

// rcHeadNotPushed is RegateResult.ChecksPending on the merge path while PR
// #7's head is still pre, the head the pushed merge commit merge was made on.
func rcHeadNotPushed(pre, merge string) string {
	return "PR #7's head is " + pre[:12] + ", not " + merge[:12] + ", the merge commit regate pushed to origin's feat/auth: GitHub has " +
		"not updated the PR to the push yet"
}

// The PR's reads of a wait: one that finds the checks pending, and one that
// finds them done and reads the PR's head again.
var (
	rcPendingRead = []string{mxPRView, mxPRChecks}
	rcDoneRead    = []string{mxPRView, mxPRChecks, mxPRView}
)

// TestRegateCI_WaitChecksWaitsUntilTheChecksAreDone covers D5 and AC5 of the
// plan on the gate-only path: with --wait-checks 1h, checks that are pending,
// or a PR whose head is not HEAD yet while origin's tip of the branch, read
// from origin at a read after a sleep, is still HEAD, are waited out, with
// regate.lock released for every 30 s sleep (another regate or merge takes it
// then) and for every read of the PR's head and checks after a sleep
// (self-review L3: the wait only reads), and taken again once they are done,
// before the read it records from and the record, whose marker names the org
// again; a gate_ci_passed or gate_ci_failed of HEAD is then recorded once,
// with one line when the wait starts and one when it ends.
func TestRegateCI_WaitChecksWaitsUntilTheChecksAreDone(t *testing.T) {
	other := strings.Repeat("e", 40)
	for _, tc := range []struct {
		name    string
		setup   func(fx *rcWaitFixture)
		reason  string
		failed  []string
		keys    []string
		reads   string
		why     func(fx *rcWaitFixture) string
		waited  string
		endWhy  string
		sleeps  int
		checks  string // the JSON of the checks recorded in the log
		eventOf string
	}{
		{"pending twice, then passed", func(fx *rcWaitFixture) {
			fx.exec.replies[mxPRChecks] = rcPending
			fx.onSleep = func(n int) {
				if n == 2 {
					fx.reply(mxPRChecks, mgReply{out: mxChecks(mxPass("verify"), mxPass("lint"))})
				}
			}
		}, "", nil, slices.Concat(rcChecksKeys, rcPendingRead, rcDoneRead, rcDoneRead), "LL" + ".." + "..." + "LLL",
			func(*rcWaitFixture) string { return rcPendingWhy }, "1m0s", "the checks passed, and gate_ci_passed was recorded", 2,
			mxChecks(mxPass("verify"), mxPass("lint")), EventGateCIPassed},
		{"pending, then a check failed", func(fx *rcWaitFixture) {
			fx.exec.replies[mxPRChecks] = rcPending
			fx.onSleep = func(int) {
				fx.reply(mxPRChecks, mgReply{out: mxChecks(mxPass("verify"), rcCheck("fail", "lint", "FAILURE")),
					err: errors.New("exit status 1")})
			}
		}, GateReasonCIFailed, []string{"ci / lint (fail, FAILURE)"}, slices.Concat(rcChecksKeys, rcDoneRead, rcDoneRead),
			"LL" + "..." + "LLL", func(*rcWaitFixture) string { return rcPendingWhy }, "30s",
			"a check did not pass, and gate_ci_failed was recorded", 1, mxChecks(mxPass("verify"), rcCheck("fail", "lint", "FAILURE")),
			EventGateCIFailed},
		{"the PR's head not HEAD yet, then its checks passed", func(fx *rcWaitFixture) {
			fx.exec.replies[mxPRView] = rcPRHead(other)
			fx.onSleep = func(int) { fx.reply(mxPRView, rcPRHead(fx.head)) }
		}, "", nil, slices.Concat([]string{mxGetURL, mxPRList, mxPRView}, rcDoneRead, rcDoneRead), "L" + "..." + "LLL",
			func(fx *rcWaitFixture) string { return rcHeadNotYet(other, fx.head) }, "30s",
			"the checks passed, and gate_ci_passed was recorded", 1, mxChecks(mxPass("verify")), EventGateCIPassed},
		// GitHub has not updated the PR after a read that finds its head is
		// another commit, and origin's tip, read then, is still HEAD.
		{"the PR's head not HEAD at a read after a sleep, origin's tip still HEAD, then its checks passed", func(fx *rcWaitFixture) {
			fx.exec.replies[mxPRView] = rcPRHead(other)
			fx.onSleep = func(n int) {
				if n == 2 {
					fx.reply(mxPRView, rcPRHead(fx.head))
				}
			}
		}, "", nil, slices.Concat([]string{mxGetURL, mxPRList, mxPRView, mxPRView}, rcDoneRead, rcDoneRead), "L" + "." + "..." + "LLL",
			func(fx *rcWaitFixture) string { return rcHeadNotYet(other, fx.head) }, "1m0s",
			"the checks passed, and gate_ci_passed was recorded", 2, mxChecks(mxPass("verify")), EventGateCIPassed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRCWaitFixture(t, time.Hour)
			tc.setup(fx)
			fx.assertLockedAtTheRecord(t)
			res, err := fx.run()
			want := RegateResult{Outcome: RegateOutcomeGated, Passed: tc.reason == "", Reason: tc.reason, OrgID: "auth", Worktree: fx.worktree,
				Branch: "feat/auth", Head: fx.head, Base: fx.tip, BaseRef: "refs/remotes/origin/main", LogPath: fx.logPath(), PR: 7,
				FailedChecks: tc.failed}
			if err != nil || !reflect.DeepEqual(res, want) {
				t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
			}
			fx.assertKeys(t, tc.keys)
			fx.assertReads(t, tc.reads)
			fx.assertWaited(t, slices.Repeat([]time.Duration{30 * time.Second}, tc.sleeps),
				[]string{rcWaitStart(tc.why(fx), "1h0m0s"), rcWaitEnd(tc.waited, tc.endWhy)})
			if events := fx.ciEvents(t); len(events) != 1 || events[0].Event != tc.eventOf ||
				!strings.Contains(events[0].Details, "head="+fx.head+" base="+fx.tip+" ") {
				t.Fatalf("ci gate events = %+v; want one %s of %s on %s", events, tc.eventOf, fx.head, fx.tip)
			}
			if log := fileOrAbsent(t, fx.logPath()); log != tc.checks {
				t.Errorf("log = %q\nwant the checks' JSON %q", log, tc.checks)
			}
			if calls := fx.calls(t); calls != nil {
				t.Fatalf("scripts run = %v; want none", calls)
			}
		})
	}
}

// TestRegateCI_WaitChecksOnTheMergePath covers D5 and AC5 on the merge path:
// right after the push of the merge commit, PR #7's head is still the
// branch's old head, so with --wait-checks 1h the regate waits, with
// regate.lock released for every sleep and every read after one, until
// GitHub updated the PR and its checks, pending first, passed, and then takes
// the lock again and records a gate_ci_passed of the merge commit, which
// stays pushed: RegateOutcomeMerged.
func TestRegateCI_WaitChecksOnTheMergePath(t *testing.T) {
	fx := newRCWaitFixture(t, time.Hour)
	pre := fx.head
	tip := fx.pushToOrigin(t, "main")
	fx.exec.replies[mxPRChecks] = rcPending
	fx.onSleep = func(n int) {
		switch n {
		case 1:
			fx.reply(mxPRView, rcPRHead(fx.worktreeHead(t)))
		case 2:
			fx.reply(mxPRChecks, mgReply{out: mxChecks(mxPass("verify"), mxPass("lint"))})
		}
	}
	fx.assertLockedAtTheRecord(t)
	res, err := fx.run()
	merge := fx.worktreeHead(t)
	want := RegateResult{Outcome: RegateOutcomeMerged, Passed: true, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth",
		Head: merge, Base: tip, BaseRef: "refs/remotes/origin/main", Merge: merge, PreMergeHead: pre, LogPath: fx.logPath(), Pushed: true,
		PR: 7}
	if err != nil || !reflect.DeepEqual(res, want) {
		t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
	}
	fx.assertMergeCommit(t, merge, pre, tip)
	fx.assertOnOrigin(t, merge)
	fx.assertKeys(t, slices.Concat([]string{mxGetURL, mxPRList, mxPRView}, rcPendingRead, rcDoneRead, rcDoneRead))
	fx.assertReads(t, "L"+".."+"..."+"LLL")
	fx.assertWaited(t, []time.Duration{30 * time.Second, 30 * time.Second}, []string{rcWaitStart(rcHeadNotPushed(pre, merge), "1h0m0s"),
		rcWaitEnd("1m0s", "the checks passed, and gate_ci_passed was recorded")})
	if events := fx.ciEvents(t); len(events) != 1 || events[0].Event != EventGateCIPassed ||
		!strings.HasPrefix(events[0].Details, "head="+merge+" base="+tip+" ") {
		t.Fatalf("ci gate events = %+v; want one gate_ci_passed of the merge commit %s on %s", events, merge, tip)
	}
	if class := fx.mergeClass(t); class != MergeClassReady {
		t.Errorf("merge-order class after the regate = %s; want ready", class)
	}
}

// TestRegateCI_WaitChecksRunsOut covers the end of D5's wait at its limit: a
// PR whose checks stay pending (gate-only, --wait-checks 45s) or whose head
// stays the old one after the push (merge path, --wait-checks 1m) is read
// again after every sleep, with regate.lock released, the last sleep cut to
// what is left of the limit, and then ends as RegateOutcomeChecksPending with
// ChecksPending saying why the last read found them pending and that the wait
// ran out, nothing recorded, no log, and the lock not taken again, since
// there is nothing to record (self-review L3); the merge path's merge commit
// stays on origin.
func TestRegateCI_WaitChecksRunsOut(t *testing.T) {
	t.Run("gate-only", func(t *testing.T) {
		fx := newRCWaitFixture(t, 45*time.Second)
		fx.exec.replies[mxPRChecks] = rcPending
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		res, err := fx.run()
		want := RegateResult{Outcome: RegateOutcomeChecksPending, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: fx.head,
			Base: fx.tip, BaseRef: "refs/remotes/origin/main", PR: 7, ChecksNext: rcNext,
			ChecksPending: rcPendingWhy + "; ralph org regate waited 45s for the checks of PR #7 (--wait-checks 45s), and that ran out"}
		if err != nil || !reflect.DeepEqual(res, want) {
			t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
		}
		fx.assertKeys(t, slices.Concat(rcChecksKeys, rcPendingRead, rcPendingRead))
		fx.assertReads(t, "LL"+".."+"..")
		fx.assertWaited(t, []time.Duration{30 * time.Second, 15 * time.Second}, []string{rcWaitStart(rcPendingWhy, "45s"),
			rcWaitEnd("45s", "--wait-checks 45s is used up, so nothing was recorded")})
		fx.assertNothingWritten(t, manifest)
		fx.assertNotTakenAgain(t)
	})
	t.Run("merge path", func(t *testing.T) {
		fx := newRCWaitFixture(t, time.Minute)
		pre := fx.head
		tip := fx.pushToOrigin(t, "main")
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		res, err := fx.run()
		merge := fx.worktreeHead(t)
		want := RegateResult{Outcome: RegateOutcomeChecksPending, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: merge,
			Base: tip, BaseRef: "refs/remotes/origin/main", Merge: merge, PreMergeHead: pre, Pushed: true, PR: 7, ChecksNext: rcNext,
			ChecksPending: rcHeadNotPushed(pre, merge) + "; ralph org regate waited 1m0s for the checks of PR #7 (--wait-checks 1m0s), and " +
				"that ran out"}
		if err != nil || !reflect.DeepEqual(res, want) {
			t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
		}
		fx.assertOnOrigin(t, merge)
		fx.assertKeys(t, []string{mxGetURL, mxPRList, mxPRView, mxPRView, mxPRView})
		fx.assertReads(t, "L"+"..")
		fx.assertWaited(t, []time.Duration{30 * time.Second, 30 * time.Second}, []string{rcWaitStart(rcHeadNotPushed(pre, merge), "1m0s"),
			rcWaitEnd("1m0s", "--wait-checks 1m0s is used up, so nothing was recorded")})
		fx.assertNothingWritten(t, manifest)
		fx.assertNotTakenAgain(t)
	})
}

// TestRegateCI_WaitChecksChecksAgainAfterASleep covers D5's checks once the
// wait's read after a sleep found the checks done, with regate.lock taken
// again, once: another regate, merge, or cleanup that holds the lock then,
// the branch's org that started again or a new binding of the branch (the
// first check), HEAD that moved in the worktree, and origin's tip of the
// branch that moved, is gone, or cannot be read (read from origin itself) are
// each refused with what was found and nothing recorded, the PR read no more
// after that read; after the merge path's push, the refusal says the merge
// commit stays on origin.
func TestRegateCI_WaitChecksChecksAgainAfterASleep(t *testing.T) {
	const waited = "after it waited for the checks of PR #7, regate "
	// other is regate.lock as another regate takes it in the first sleep of
	// the case that needs it, until the regate returned.
	var other *heldFileLock
	for _, tc := range []struct {
		name string
		// during runs in the first sleep, in which the checks pass; it
		// returns the refusal, or the parts of it.
		during func(t *testing.T, fx *rcWaitFixture) []string
	}{
		{"another regate holds regate.lock once the checks are done", func(t *testing.T, fx *rcWaitFixture) []string {
			lock, err := acquireHeldFileLock(fx.stateDir, regateLockFile, "test", 0)
			if err != nil {
				t.Fatal(err)
			}
			other = lock
			return []string{"org: regate: another ralph org regate, merge, or cleanup took " + regateLockPathIn(fx.stateDir) + " while " +
				"regate waited for the checks of PR #7, so nothing was recorded: wait for it to end and " + rcAgain}
		}},
		{"the org started again", func(t *testing.T, fx *rcWaitFixture) []string {
			fx.appendEvents(t, reserveEvent("auth", "internal/other/"))
			return []string{regateChangedRefusal("feat/auth", "regate waited for the checks of PR #7", `org_id "auth" is running again`)}
		}},
		{"the branch was bound again", func(t *testing.T, fx *rcWaitFixture) []string {
			fx.rebind(t, fx.worktree)
			return []string{"org: regate: nothing was recorded, since the branch feat/auth changed while regate waited for the checks of " +
				`PR #7 (it is now bound to org_id "auth-2"'s feature `, "check it with ralph org merge-order and ralph org status"}
		}},
		{"HEAD moved", func(t *testing.T, fx *rcWaitFixture) []string {
			writeGateFile(t, fx.worktree, "internal/auth/more.go", "package auth\n", 0o644)
			runGit(t, "-C", fx.worktree, "add", ".")
			gateCommit(t, fx.worktree, "not pushed")
			return []string{"org: regate: " + waited + "found HEAD of feat/auth in " + fx.worktree + " at " + fx.worktreeHead(t) + ", no " +
				"longer " + fx.head + ", so nothing was recorded: " + fx.rcCheckOrigin()}
		}},
		{"origin's tip moved", func(t *testing.T, fx *rcWaitFixture) []string {
			newer := fx.pushToOrigin(t, "feat/auth")
			return []string{"org: regate: " + waited + "found origin's feat/auth at " + newer + ", no longer HEAD " + fx.head + ", so " +
				"nothing was recorded: " + fx.rcCheckOrigin()}
		}},
		{"origin has no branch", func(t *testing.T, fx *rcWaitFixture) []string {
			runGit(t, "-C", fx.origin, "update-ref", "-d", "refs/heads/feat/auth")
			return []string{"org: regate: " + waited + "found no branch feat/auth on origin, so nothing was recorded: " + fx.rcCheckOrigin()}
		}},
		{"origin's tip cannot be read", func(t *testing.T, fx *rcWaitFixture) []string {
			runGit(t, "-C", fx.main, "remote", "set-url", "origin", filepath.Join(fx.out, "missing.git"))
			return []string{"org: regate: " + waited + "could not read origin's tip of feat/auth back (git ls-remote origin " +
				"refs/heads/feat/auth in " + fx.worktree + " failed (exit 128)), so nothing was recorded: " + fx.rcCheckOrigin()}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRCWaitFixture(t, time.Hour)
			fx.exec.replies[mxPRChecks] = rcPending
			var want []string
			fx.onSleep = func(int) {
				fx.reply(mxPRChecks, mgReply{out: mxChecks(mxPass("verify"))})
				want = tc.during(t, fx)
			}
			res, err := fx.run()
			if other != nil {
				_ = other.release()
				other = nil
			}
			if err != nil || res.Refused == "" {
				t.Fatalf("regate = %+v, %v; want a refusal", res, err)
			}
			if len(want) == 1 && res.Refused != want[0] {
				t.Errorf("refusal = %q\nwant      %q", res.Refused, want[0])
			}
			for _, w := range want {
				if !strings.Contains(res.Refused, w) {
					t.Errorf("refusal %q does not say %q", res.Refused, w)
				}
			}
			wantRes := RegateResult{Refused: res.Refused, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: fx.head, Base: fx.tip,
				BaseRef: "refs/remotes/origin/main", PR: 7}
			if !reflect.DeepEqual(res, wantRes) {
				t.Fatalf("regate = %#v\nwant     %#v", res, wantRes)
			}
			fx.assertKeys(t, slices.Concat(rcChecksKeys, rcDoneRead))
			if events := fx.ciEvents(t); len(events) != 0 {
				t.Fatalf("ci gate events = %+v; want none", events)
			}
			if _, err := os.Stat(filepath.Join(fx.stateDir, gateLogDir)); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("%s exists (stat: %v); want no log", gateLogDir, err)
			}
			fx.assertWaited(t, []time.Duration{30 * time.Second}, []string{rcWaitStart(rcPendingWhy, "1h0m0s"),
				rcWaitEnd("30s", "it was refused, so nothing was recorded")})
		})
	}

	t.Run("origin's tip moved after the merge path's push", func(t *testing.T) {
		fx := newRCWaitFixture(t, time.Hour)
		pre := fx.head
		tip := fx.pushToOrigin(t, "main")
		var newer string
		fx.onSleep = func(int) {
			fx.reply(mxPRView, rcPRHead(fx.worktreeHead(t)))
			newer = fx.pushToOrigin(t, "feat/auth")
		}
		res, err := fx.run()
		merge := fx.worktreeHead(t)
		want := RegateResult{Refused: "org: regate: " + waited + "found origin's feat/auth at " + newer + ", no longer HEAD " + merge +
			", so nothing was recorded: " + fx.rcCheckOrigin() + "; the merge commit " + merge + " was pushed to origin's feat/auth and " +
			"stays there", OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: merge, Base: tip,
			BaseRef: "refs/remotes/origin/main", Merge: merge, PreMergeHead: pre, Pushed: true, PR: 7}
		if err != nil || !reflect.DeepEqual(res, want) {
			t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
		}
		fx.assertKeys(t, slices.Concat([]string{mxGetURL, mxPRList, mxPRView}, rcDoneRead))
		fx.assertWaited(t, []time.Duration{30 * time.Second}, []string{rcWaitStart(rcHeadNotPushed(pre, merge), "1h0m0s"),
			rcWaitEnd("30s", "it was refused, so nothing was recorded")})
		if events := fx.ciEvents(t); len(events) != 0 {
			t.Fatalf("ci gate events = %+v; want none", events)
		}
	})
}

// TestRegateCI_WaitChecksEndsWhenAnotherRegatePassedHEAD covers D5's end
// when, while the regate waited with regate.lock released, another regate
// recorded a gate_ci_passed of HEAD on origin's main: once the wait's read
// finds the checks done and the lock is taken again, the wait ends as
// RegateOutcomeUpToDate, with that record's log, without reading the PR again
// or recording anything; on the merge path the merge commit stays pushed.
func TestRegateCI_WaitChecksEndsWhenAnotherRegatePassedHEAD(t *testing.T) {
	end := rcWaitEnd("30s", "another regate recorded a pass of HEAD meanwhile")
	t.Run("gate-only", func(t *testing.T) {
		fx := newRCWaitFixture(t, time.Hour)
		fx.exec.replies[mxPRChecks] = rcPending
		fx.onSleep = func(int) {
			fx.reply(mxPRChecks, mgReply{out: mxChecks(mxPass("verify"))})
			fx.appendEvents(t, moCIGate("auth", "feat/auth", fx.head, fx.tip, ""))
		}
		res, err := fx.run()
		want := RegateResult{Outcome: RegateOutcomeUpToDate, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: fx.head,
			Base: fx.tip, BaseRef: "refs/remotes/origin/main", LogPath: moRegateCILog, PR: 7}
		if err != nil || !reflect.DeepEqual(res, want) {
			t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
		}
		fx.assertKeys(t, slices.Concat(rcChecksKeys, rcDoneRead))
		fx.assertWaited(t, []time.Duration{30 * time.Second}, []string{rcWaitStart(rcPendingWhy, "1h0m0s"), end})
		if events := fx.ciEvents(t); len(events) != 1 || events[0].Details != moCIGate("auth", "feat/auth", fx.head, fx.tip, "").Details {
			t.Fatalf("ci gate events = %+v; want only the other regate's", events)
		}
		if _, err := os.Stat(filepath.Join(fx.stateDir, gateLogDir)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s exists (stat: %v); want no log", gateLogDir, err)
		}
	})
	t.Run("merge path", func(t *testing.T) {
		fx := newRCWaitFixture(t, time.Hour)
		pre := fx.head
		tip := fx.pushToOrigin(t, "main")
		fx.onSleep = func(int) {
			fx.reply(mxPRView, rcPRHead(fx.worktreeHead(t)))
			fx.appendEvents(t, moCIGate("auth", "feat/auth", fx.worktreeHead(t), tip, ""))
		}
		res, err := fx.run()
		merge := fx.worktreeHead(t)
		want := RegateResult{Outcome: RegateOutcomeUpToDate, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: merge, Base: tip,
			BaseRef: "refs/remotes/origin/main", Merge: merge, PreMergeHead: pre, LogPath: moRegateCILog, Pushed: true, PR: 7}
		if err != nil || !reflect.DeepEqual(res, want) {
			t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
		}
		fx.assertOnOrigin(t, merge)
		fx.assertKeys(t, slices.Concat([]string{mxGetURL, mxPRList, mxPRView}, rcDoneRead))
		fx.assertWaited(t, []time.Duration{30 * time.Second}, []string{rcWaitStart(rcHeadNotPushed(pre, merge), "1h0m0s"), end})
		if events := fx.ciEvents(t); len(events) != 1 {
			t.Fatalf("ci gate events = %+v; want only the other regate's", events)
		}
	})
}

// TestRegateCI_AnInterruptWhileWaitingForTheChecks covers AC5's interrupt of
// D5's wait: ctx done while the regate sleeps, or while it reads the PR's
// head after a sleep, ends the wait at once, with an error that says so,
// nothing recorded, and regate.lock released, not taken again (its marker,
// which the sleep emptied, stays empty); ctx done once the read found the
// checks done, while the ledger is read again under the manifest lock, ends
// it with the same error. After the merge path's push, the error says the
// merge commit stays on origin, as S4b's interrupt after the push does.
func TestRegateCI_AnInterruptWhileWaitingForTheChecks(t *testing.T) {
	end := rcWaitEnd("30s", "interrupted (test stopped), so nothing was recorded")
	stop := errors.New("test stopped")
	t.Run("gate-only", func(t *testing.T) {
		fx := newRCWaitFixture(t, time.Hour)
		fx.exec.replies[mxPRChecks] = rcPending
		fx.onSleep = func(int) { fx.cancel(stop) }
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		res, err := fx.run()
		if err == nil || err.Error() != rcWaitInterrupted {
			t.Fatalf("regate = %v\nwant     %s", err, rcWaitInterrupted)
		}
		want := RegateResult{OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: fx.head, Base: fx.tip,
			BaseRef: "refs/remotes/origin/main", PR: 7}
		if !reflect.DeepEqual(res, want) {
			t.Fatalf("regate = %#v\nwant     %#v", res, want)
		}
		fx.assertKeys(t, rcChecksKeys)
		fx.assertWaited(t, []time.Duration{30 * time.Second}, []string{rcWaitStart(rcPendingWhy, "1h0m0s"), end})
		fx.assertNothingWritten(t, manifest)
		fx.assertNotTakenAgain(t)
	})
	t.Run("while a read after the sleep runs", func(t *testing.T) {
		fx := newRCWaitFixture(t, time.Hour)
		fx.exec.replies[mxPRChecks] = rcPending
		fx.afterCall = func(n int) {
			if n == 4 { // gh pr view, the first read after the sleep
				fx.cancel(stop)
			}
		}
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		res, err := fx.run()
		if err == nil || err.Error() != rcWaitInterrupted {
			t.Fatalf("regate = %+v, %v\nwant     %s", res, err, rcWaitInterrupted)
		}
		want := RegateResult{OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: fx.head, Base: fx.tip,
			BaseRef: "refs/remotes/origin/main", PR: 7}
		if !reflect.DeepEqual(res, want) {
			t.Fatalf("regate = %#v\nwant     %#v", res, want)
		}
		fx.assertKeys(t, slices.Concat(rcChecksKeys, []string{mxPRView}))
		fx.assertReads(t, "LL"+".")
		fx.assertWaited(t, []time.Duration{30 * time.Second}, []string{rcWaitStart(rcPendingWhy, "1h0m0s"), end})
		fx.assertNothingWritten(t, manifest)
		fx.assertNotTakenAgain(t)
	})
	t.Run("merge path", func(t *testing.T) {
		fx := newRCWaitFixture(t, time.Hour)
		pre := fx.head
		tip := fx.pushToOrigin(t, "main")
		fx.onSleep = func(int) { fx.cancel(stop) }
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		res, err := fx.run()
		merge := fx.worktreeHead(t)
		if want := rcWaitInterrupted + "; the merge commit " + merge + " was pushed to origin's feat/auth and stays there: " + rcAgain +
			", which reads the PR's checks of it"; err == nil || err.Error() != want {
			t.Fatalf("regate = %v\nwant     %s", err, want)
		}
		want := RegateResult{OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: merge, Base: tip,
			BaseRef: "refs/remotes/origin/main", Merge: merge, PreMergeHead: pre, Pushed: true, PR: 7}
		if !reflect.DeepEqual(res, want) {
			t.Fatalf("regate = %#v\nwant     %#v", res, want)
		}
		fx.assertOnOrigin(t, merge)
		fx.assertKeys(t, []string{mxGetURL, mxPRList, mxPRView})
		fx.assertWaited(t, []time.Duration{30 * time.Second}, []string{rcWaitStart(rcHeadNotPushed(pre, merge), "1h0m0s"), end})
		fx.assertNothingWritten(t, manifest)
		fx.assertNotTakenAgain(t)
	})
	t.Run("while the ledger is read again once the checks are done", func(t *testing.T) {
		fx := newRCWaitFixture(t, time.Hour)
		fx.exec.replies[mxPRChecks] = rcPending
		locked, release, held := make(chan struct{}), make(chan struct{}), make(chan error, 1)
		fx.onSleep = func(int) {
			fx.reply(mxPRChecks, mgReply{out: mxChecks(mxPass("verify"))})
			// The manifest lock, held until the regate returned, keeps its
			// first check waiting once the read after the sleep found the
			// checks done; ctx is stopped once the sleep is over.
			go func() {
				held <- withManifestLock(fx.stateDir, func() error {
					close(locked)
					<-release
					return nil
				})
			}()
			<-locked
			time.AfterFunc(100*time.Millisecond, func() { fx.cancel(stop) })
		}
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		res, err := fx.run()
		close(release)
		if err := <-held; err != nil {
			t.Fatal(err)
		}
		if err == nil || err.Error() != rcWaitInterrupted {
			t.Fatalf("regate = %+v, %v\nwant     %s", res, err, rcWaitInterrupted)
		}
		if res.Refused != "" || res.Outcome != "" || res.ChecksPending != "" {
			t.Fatalf("regate = %+v; want no refusal and no outcome", res)
		}
		fx.assertKeys(t, slices.Concat(rcChecksKeys, rcDoneRead))
		fx.assertWaited(t, []time.Duration{30 * time.Second}, []string{rcWaitStart(rcPendingWhy, "1h0m0s"), end})
		fx.assertNothingWritten(t, manifest)
	})
}

// TestRegateCI_WaitChecksGoesOnWhileAnotherHoldsTheLock covers self-review L3:
// two waits overlap, or a merge or another regate runs, while this regate
// waits for the checks of PR #7. Another holds regate.lock through each of
// the wait's reads after a sleep (taken in the sleep, released after the
// read's last call), and the wait, which only reads, goes on; once the checks
// are done and the lock is free again, it takes the lock and records the
// gate_ci_passed, as it does with nobody else on the ledger. Retaking the
// lock before each read, as the wait of S5 did, refuses at the first one.
func TestRegateCI_WaitChecksGoesOnWhileAnotherHoldsTheLock(t *testing.T) {
	fx := newRCWaitFixture(t, time.Hour)
	fx.exec.replies[mxPRChecks] = rcPending
	var other *heldFileLock
	release := func() {
		if other != nil {
			_ = other.release()
			other = nil
		}
	}
	t.Cleanup(release)
	fx.onSleep = func(n int) {
		lock, err := acquireHeldFileLock(fx.stateDir, regateLockFile, "test", 0)
		if err != nil {
			t.Fatal(err)
		}
		other = lock
		if n == 2 {
			fx.reply(mxPRChecks, mgReply{out: mxChecks(mxPass("verify"), mxPass("lint"))})
		}
	}
	// The calls from 0: git remote get-url, gh pr list, the first read (2,
	// 3), the read after sleep 1 (4, 5: pending), and the one after sleep 2
	// (6 to 8: done). Another holds the lock through the last two.
	fx.afterCall = func(n int) {
		if n == 5 || n == 8 {
			release()
		}
	}
	fx.assertLockedAtTheRecord(t)
	res, err := fx.run()
	want := RegateResult{Outcome: RegateOutcomeGated, Passed: true, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: fx.head,
		Base: fx.tip, BaseRef: "refs/remotes/origin/main", LogPath: fx.logPath(), PR: 7}
	if err != nil || !reflect.DeepEqual(res, want) {
		t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
	}
	fx.assertKeys(t, slices.Concat(rcChecksKeys, rcPendingRead, rcDoneRead, rcDoneRead))
	// L: this regate's lock in the first and the last read, the other's in
	// the two between.
	fx.assertReads(t, "LL"+"LL"+"LLL"+"LLL")
	fx.assertWaited(t, []time.Duration{30 * time.Second, 30 * time.Second}, []string{rcWaitStart(rcPendingWhy, "1h0m0s"),
		rcWaitEnd("1m0s", "the checks passed, and gate_ci_passed was recorded")})
	if events := fx.ciEvents(t); len(events) != 1 || events[0].Event != EventGateCIPassed {
		t.Fatalf("ci gate events = %+v; want one gate_ci_passed", events)
	}
}

// TestRegateCI_WaitChecksPendingAgainUnderTheLock covers self-review L3's end
// of the wait when the read with regate.lock taken again finds the checks
// pending again (CI ran again between the wait's read and that one): it ends
// as RegateOutcomeChecksPending with why and that they were not done again,
// records nothing, writes no log, and does not wait again with the lock held
// (one sleep), with a line that says so.
func TestRegateCI_WaitChecksPendingAgainUnderTheLock(t *testing.T) {
	fx := newRCWaitFixture(t, time.Hour)
	fx.exec.replies[mxPRChecks] = rcPending
	fx.onSleep = func(int) { fx.reply(mxPRChecks, mgReply{out: mxChecks(mxPass("verify"), mxPass("lint"))}) }
	fx.afterCall = func(n int) {
		if n == 6 { // gh pr view, the last call of the read after the sleep, which found the checks done
			fx.reply(mxPRChecks, rcPending)
		}
	}
	manifest := fileOrAbsent(t, fx.o.Manifest.Path())
	res, err := fx.run()
	want := RegateResult{Outcome: RegateOutcomeChecksPending, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: fx.head,
		Base: fx.tip, BaseRef: "refs/remotes/origin/main", PR: 7, ChecksNext: rcNext,
		ChecksPending: rcPendingWhy + "; ralph org regate waited 30s for the checks of PR #7 (--wait-checks 1h0m0s), and they were not " +
			"done again when it read them with regate.lock taken again"}
	if err != nil || !reflect.DeepEqual(res, want) {
		t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
	}
	fx.assertKeys(t, slices.Concat(rcChecksKeys, rcDoneRead, rcPendingRead))
	fx.assertReads(t, "LL"+"..."+"LL")
	fx.assertWaited(t, []time.Duration{30 * time.Second}, []string{rcWaitStart(rcPendingWhy, "1h0m0s"),
		rcWaitEnd("30s", "the checks were not done again when it read them with regate.lock taken again, so nothing was recorded")})
	fx.assertNothingWritten(t, manifest)
}

// TestRegateCI_WaitChecksStopsOnceThePRsHeadShowsTheBranchMoved covers the
// wait's reads after a sleep that find PR #7's head not HEAD (D5): on the
// merge path, a head still at the one the merge commit was made on, which
// GitHub shows until it has updated the PR to the push, keeps the wait going
// without reading origin, which may not even be readable then; a head at
// another commit, as when someone pushed to origin's
// feat/auth, and on the gate-only path any head other than HEAD, one that
// moved while the checks were read included, makes the wait read origin's tip
// of feat/auth from origin with no lock (with it still HEAD, the wait goes on:
// TestRegateCI_WaitChecksWaitsUntilTheChecksAreDone). A tip that is not HEAD
// stops the wait at that read, with no sleep after it, and the single retake
// of regate.lock reads origin's tip again and refuses with its new tip,
// recording nothing. A tip that cannot be read stops it too, and the retake
// reads it again: still not readable, it refuses; HEAD, the read under the
// lock ends the wait as RegateOutcomeChecksPending, saying why it stopped.
// The read-only wait of self-review L3 slept on until the checks were done or
// the limit.
func TestRegateCI_WaitChecksStopsOnceThePRsHeadShowsTheBranchMoved(t *testing.T) {
	const found = "org: regate: after it waited for the checks of PR #7, regate found origin's feat/auth at "
	refused := rcWaitEnd("1m0s", "it was refused, so nothing was recorded")
	t.Run("merge path: the head the merge commit was made on, then a push", func(t *testing.T) {
		fx := newRCWaitFixture(t, time.Hour)
		pre := fx.head
		tip := fx.pushToOrigin(t, "main")
		var newer string
		fx.onSleep = func(n int) {
			if n == 2 {
				newer = fx.pushToOrigin(t, "feat/auth")
				fx.reply(mxPRView, rcPRHead(newer))
			}
		}
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		res, err := fx.run()
		merge := fx.worktreeHead(t)
		want := RegateResult{Refused: found + newer + ", no longer HEAD " + merge + ", so nothing was recorded: " + fx.rcCheckOrigin() +
			"; the merge commit " + merge + " was pushed to origin's feat/auth and stays there", OrgID: "auth", Worktree: fx.worktree,
			Branch: "feat/auth", Head: merge, Base: tip, BaseRef: "refs/remotes/origin/main", Merge: merge, PreMergeHead: pre, Pushed: true,
			PR: 7}
		if err != nil || !reflect.DeepEqual(res, want) {
			t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
		}
		fx.assertKeys(t, []string{mxGetURL, mxPRList, mxPRView, mxPRView, mxPRView})
		fx.assertReads(t, "L"+"."+".")
		fx.assertWaited(t, []time.Duration{30 * time.Second, 30 * time.Second}, []string{rcWaitStart(rcHeadNotPushed(pre, merge), "1h0m0s"),
			refused})
		fx.assertNothingWritten(t, manifest)
	})
	t.Run("merge path: the head the merge commit was made on, with origin not readable", func(t *testing.T) {
		fx := newRCWaitFixture(t, time.Hour)
		pre := fx.head
		tip := fx.pushToOrigin(t, "main")
		fx.onSleep = func(n int) {
			switch n {
			case 1:
				runGit(t, "-C", fx.main, "remote", "set-url", "origin", filepath.Join(fx.out, "missing.git"))
			case 2:
				runGit(t, "-C", fx.main, "remote", "set-url", "origin", fx.origin)
				fx.reply(mxPRView, rcPRHead(fx.worktreeHead(t)))
			}
		}
		fx.assertLockedAtTheRecord(t)
		res, err := fx.run()
		merge := fx.worktreeHead(t)
		want := RegateResult{Outcome: RegateOutcomeMerged, Passed: true, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth",
			Head: merge, Base: tip, BaseRef: "refs/remotes/origin/main", Merge: merge, PreMergeHead: pre, LogPath: fx.logPath(), Pushed: true,
			PR: 7}
		if err != nil || !reflect.DeepEqual(res, want) {
			t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
		}
		fx.assertKeys(t, slices.Concat([]string{mxGetURL, mxPRList, mxPRView, mxPRView}, rcDoneRead, rcDoneRead))
		fx.assertReads(t, "L"+"."+"..."+"LLL")
		fx.assertWaited(t, []time.Duration{30 * time.Second, 30 * time.Second}, []string{rcWaitStart(rcHeadNotPushed(pre, merge), "1h0m0s"),
			rcWaitEnd("1m0s", "the checks passed, and gate_ci_passed was recorded")})
		if events := fx.ciEvents(t); len(events) != 1 || events[0].Event != EventGateCIPassed {
			t.Fatalf("ci gate events = %+v; want one gate_ci_passed", events)
		}
	})
	t.Run("gate-only: a push while the checks are pending", func(t *testing.T) {
		fx := newRCWaitFixture(t, time.Hour)
		fx.exec.replies[mxPRChecks] = rcPending
		var newer string
		fx.onSleep = func(n int) {
			if n == 2 {
				newer = fx.pushToOrigin(t, "feat/auth")
				fx.reply(mxPRView, rcPRHead(newer))
			}
		}
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		res, err := fx.run()
		want := RegateResult{Refused: found + newer + ", no longer HEAD " + fx.head + ", so nothing was recorded: " + fx.rcCheckOrigin(),
			OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: fx.head, Base: fx.tip, BaseRef: "refs/remotes/origin/main", PR: 7}
		if err != nil || !reflect.DeepEqual(res, want) {
			t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
		}
		fx.assertKeys(t, slices.Concat(rcChecksKeys, rcPendingRead, []string{mxPRView}))
		fx.assertReads(t, "LL"+".."+".")
		fx.assertWaited(t, []time.Duration{30 * time.Second, 30 * time.Second}, []string{rcWaitStart(rcPendingWhy, "1h0m0s"), refused})
		fx.assertNothingWritten(t, manifest)
	})
	t.Run("gate-only: a push while the checks were read", func(t *testing.T) {
		fx := newRCWaitFixture(t, time.Hour)
		fx.exec.replies[mxPRChecks] = rcPending
		fx.onSleep = func(int) { fx.reply(mxPRChecks, mgReply{out: mxChecks(mxPass("verify"))}) }
		var newer string
		fx.afterCall = func(n int) {
			if n == 5 { // gh pr checks of the read after the sleep, which passed
				newer = fx.pushToOrigin(t, "feat/auth")
				fx.reply(mxPRView, rcPRHead(newer))
			}
		}
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		res, err := fx.run()
		want := RegateResult{Refused: found + newer + ", no longer HEAD " + fx.head + ", so nothing was recorded: " + fx.rcCheckOrigin(),
			OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: fx.head, Base: fx.tip, BaseRef: "refs/remotes/origin/main", PR: 7}
		if err != nil || !reflect.DeepEqual(res, want) {
			t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
		}
		fx.assertKeys(t, slices.Concat(rcChecksKeys, rcDoneRead))
		fx.assertReads(t, "LL"+"...")
		fx.assertWaited(t, []time.Duration{30 * time.Second}, []string{rcWaitStart(rcPendingWhy, "1h0m0s"),
			rcWaitEnd("30s", "it was refused, so nothing was recorded")})
		fx.assertNothingWritten(t, manifest)
	})
	t.Run("gate-only: origin's tip cannot be read", func(t *testing.T) {
		fx := newRCWaitFixture(t, time.Hour)
		fx.exec.replies[mxPRChecks] = rcPending
		other := strings.Repeat("e", 40)
		fx.onSleep = func(int) {
			runGit(t, "-C", fx.main, "remote", "set-url", "origin", filepath.Join(fx.out, "missing.git"))
			fx.reply(mxPRView, rcPRHead(other))
		}
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		res, err := fx.run()
		want := RegateResult{Refused: "org: regate: after it waited for the checks of PR #7, regate could not read origin's tip of " +
			"feat/auth back (git ls-remote origin refs/heads/feat/auth in " + fx.worktree + " failed (exit 128)), so nothing was recorded: " +
			fx.rcCheckOrigin(), OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: fx.head, Base: fx.tip,
			BaseRef: "refs/remotes/origin/main", PR: 7}
		if err != nil || !reflect.DeepEqual(res, want) {
			t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
		}
		fx.assertKeys(t, slices.Concat(rcChecksKeys, []string{mxPRView}))
		fx.assertReads(t, "LL"+".")
		fx.assertWaited(t, []time.Duration{30 * time.Second}, []string{rcWaitStart(rcPendingWhy, "1h0m0s"),
			rcWaitEnd("30s", "it was refused, so nothing was recorded")})
		fx.assertNothingWritten(t, manifest)
	})
	t.Run("gate-only: origin's tip cannot be read, and is HEAD under the lock", func(t *testing.T) {
		fx := newRCWaitFixture(t, time.Hour)
		fx.exec.replies[mxPRChecks] = rcPending
		other := strings.Repeat("e", 40)
		locked, held := make(chan struct{}), make(chan error, 1)
		fx.onSleep = func(int) {
			runGit(t, "-C", fx.main, "remote", "set-url", "origin", filepath.Join(fx.out, "missing.git"))
			fx.reply(mxPRView, rcPRHead(other))
			// The manifest lock, held from here, keeps the retake's first
			// check waiting; once regate.lock is held again (the poll ended
			// and regateCIRecheck took it), origin's URL is set back, so the
			// retake reads origin's tip as HEAD.
			go func() {
				held <- withManifestLock(fx.stateDir, func() error {
					close(locked)
					for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(5 * time.Millisecond) {
						if taken, err := fileLockHeld(regateLockPathIn(fx.stateDir)); err != nil || taken {
							if err != nil {
								return err
							}
							break
						}
						if time.Now().After(deadline) {
							return errors.New("regate.lock was not taken again")
						}
					}
					return exec.Command("git", "-C", fx.main, "remote", "set-url", "origin", fx.origin).Run()
				})
			}()
			<-locked
		}
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		res, err := fx.run()
		if err := <-held; err != nil {
			t.Fatal(err)
		}
		const stopped = "the PR's head was a commit other than HEAD and origin's feat/auth, read then, was not HEAD or could not be " +
			"read, but it was HEAD when regate read it again with regate.lock taken again, and the checks were not done then"
		want := RegateResult{Outcome: RegateOutcomeChecksPending, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: fx.head,
			Base: fx.tip, BaseRef: "refs/remotes/origin/main", PR: 7, ChecksNext: rcNext,
			ChecksPending: rcHeadNotYet(other, fx.head) + "; ralph org regate waited 30s for the checks of PR #7 (--wait-checks 1h0m0s), " +
				"and it stopped once " + stopped}
		if err != nil || !reflect.DeepEqual(res, want) {
			t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
		}
		fx.assertKeys(t, slices.Concat(rcChecksKeys, []string{mxPRView, mxPRView}))
		fx.assertReads(t, "LL"+"."+"L")
		fx.assertWaited(t, []time.Duration{30 * time.Second}, []string{rcWaitStart(rcPendingWhy, "1h0m0s"),
			rcWaitEnd("30s", stopped+", so nothing was recorded")})
		fx.assertNothingWritten(t, manifest)
	})
}

// TestRegateCI_WaitChecksIsCheckedBeforeAnythingIsDone covers the validation
// of RegateParams.WaitChecks: below 0 or above 6 h, the range of ralph org
// merge --wait-checks, and above 0 with local checks, which run the gate
// scripts and have no PR checks to wait for, are refused with nothing done
// and no call through Exec; 6 h is taken, and checks done at the first read
// record at once, with no sleep and no line.
func TestRegateCI_WaitChecksIsCheckedBeforeAnythingIsDone(t *testing.T) {
	const local = "org: regate: --wait-checks 1m0s waits for the checks of the branch's PR, which regate reads only with " +
		`[org.gate].regate_checks = "ci", not "local", which runs the gate scripts instead, so nothing was done: run ralph org ` +
		`regate --branch feat/auth again without --wait-checks, or set [org.gate].regate_checks = "ci"`
	for _, tc := range []struct {
		name   string
		wait   time.Duration
		checks string
		want   string
	}{
		{"below 0", -time.Second, config.RegateChecksCI, "org: regate: --wait-checks -1s is out of range, so nothing was done: give a " +
			"duration from 0 (no wait, the default) to 6h0m0s, such as 60m"},
		{"above 6h", 6*time.Hour + time.Second, config.RegateChecksCI, "org: regate: --wait-checks 6h0m1s is out of range, so nothing " +
			"was done: give a duration from 0 (no wait, the default) to 6h0m0s, such as 60m"},
		{"with local checks", time.Minute, config.RegateChecksLocal, local},
		{"with the default checks", time.Minute, "", local},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRCWaitFixture(t, tc.wait)
			p := fx.params()
			p.Checks, p.WaitChecks, p.Stderr = tc.checks, tc.wait, fx.stderr
			p.waitTest = mergeWaitTest{now: fx.clock.Now, sleep: fx.clock.sleep}
			before := fx.snapshot(t)
			res, err := fx.o.Regate(fx.ctx, p)
			fx.assertRefused(t, res, err, before, []string{tc.want}, nil)
			if res.Refused != tc.want {
				t.Errorf("refusal = %q\nwant      %q", res.Refused, tc.want)
			}
			fx.assertKeys(t, nil)
			fx.assertWaited(t, nil, nil)
		})
	}
	t.Run("6h, with the checks done at the first read", func(t *testing.T) {
		fx := newRCWaitFixture(t, 6*time.Hour)
		res, err := fx.run()
		if err != nil || res.Outcome != RegateOutcomeGated || !res.Passed {
			t.Fatalf("regate = %+v, %v; want a gate_ci_passed at once", res, err)
		}
		fx.assertKeys(t, rcReadKeys)
		fx.assertWaited(t, nil, nil)
	})
}
