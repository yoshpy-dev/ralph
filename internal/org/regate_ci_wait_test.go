package org

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

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
