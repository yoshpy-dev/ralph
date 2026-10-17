package org

import (
	"context"
	"errors"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

// The wait of `ralph org merge --wait-checks` at the org layer (plan
// docs/plans/active/2026-10-17-org-auto-merge.md, S7: D13 and AC13): Merge on
// the fixtures of merge_run_test.go, with a fake clock (mwClock) in
// MergeParams.waitTest, so that no test sleeps for the 30 s interval, and
// the fake gh's answers changed while Merge sleeps.

// mwMaxSleeps is how many sleeps mwClock allows before it ends the wait.
const mwMaxSleeps = 50

// mwClock is a fake clock for a wait: now moves only when sleep sleeps. Each
// sleep is recorded; onSleep, when set, runs after it with the sleep's
// number (from 1), as what happens while Merge sleeps. A sleep returns ctx's
// error. After mwMaxSleeps sleeps it fails the test and stops ctx, so a wait
// that does not end ends.
type mwClock struct {
	t       *testing.T
	now     time.Time
	sleeps  []time.Duration
	onSleep func(n int)
	stop    context.CancelCauseFunc
}

func (c *mwClock) Now() time.Time { return c.now }

func (c *mwClock) sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.now = c.now.Add(d)
	c.sleeps = append(c.sleeps, d)
	if len(c.sleeps) > mwMaxSleeps {
		c.t.Errorf("more than %d sleeps: the wait does not end", mwMaxSleeps)
		c.stop(errors.New("too many sleeps"))
	}
	if c.onSleep != nil {
		c.onSleep(len(c.sleeps))
	}
	return ctx.Err()
}

// The answers of gh pr checks in these tests.
var (
	mwPending = mgReply{out: mxChecks(mxPass("verify"), ghPRCheck{Bucket: "pending", Name: "lint", State: "IN_PROGRESS",
		Workflow: "ci"}), err: errors.New("exit status 8")}
	mwPass         = mgReply{out: mxChecks(mxPass("verify"), mxPass("lint"))}
	mwNone         = mgReply{out: "[]"}
	mwNoneSentence = mgReply{err: errors.New("exit status 1: no checks reported on the 'feat/a' branch")}
	mwFail         = mgReply{out: mxChecks(mxPass("verify"), ghPRCheck{Bucket: "fail", Name: "lint", State: "FAILURE",
		Workflow: "ci"}), err: errors.New("exit status 1")}
	mwGHFails = mgReply{err: errors.New("exit status 1: HTTP 502: Bad Gateway")}
)

// The refusals of the default fixture with those answers.
const (
	mwPendingRefusal = "org: merge: 1 of the 2 checks of PR #7 are not pass or skipping: ci / lint (pending, IN_PROGRESS), so " +
		"nothing was merged: wait for the pending ones and fix the others, and once all passed, " + mxAgain
	mwFailRefusal = "org: merge: 1 of the 2 checks of PR #7 are not pass or skipping: ci / lint (fail, FAILURE), so nothing was " +
		"merged: wait for the pending ones and fix the others, and once all passed, " + mxAgain
	mwNoneRefusal = "org: merge: gh pr checks 7 -R " + mxRepoArg + " reported no checks, so nothing shows that the PR's CI " +
		"passed and nothing was merged: let CI run on the PR, and once it passed, " + mxAgain
	mwInterrupted = "org: merge: interrupted (test stopped) while waiting for the checks of PR #7, before gh pr merge ran: " +
		"nothing was merged"
)

// mwStart is the line a wait writes when it starts, why being "has checks
// pending" or "has no checks reported yet".
func mwStart(why, left, budget string) string {
	return "org: merge: PR #7 " + why + ", so ralph org merge waits for them for up to " + left + " (--wait-checks " + budget +
		"), with regate.lock released, reading gh pr checks 7 -R " + mxRepoArg + " every 30s"
}

// mwEnd is the line a wait that did not end in ctx done writes when it ends.
func mwEnd(after, why string) string {
	return "org: merge: stopped waiting for the checks of PR #7 after " + after + ": " + why + "; running every check again"
}

// mwFixture is mrFixture with gh pr checks answering checks, Merge's
// params with WaitChecks and Progress, a ctx the test stops, and mwClock,
// whose every sleep takes regate.lock and releases it (lockTaken counts
// them) before onSleep.
type mwFixture struct {
	*mrFixture
	clock     *mwClock
	progress  *strings.Builder
	ctx       context.Context
	cancel    context.CancelCauseFunc
	p         MergeParams
	onSleep   func(n int)
	lockTaken int
}

func newMWFixture(t *testing.T, checks mgReply, wait time.Duration) *mwFixture {
	t.Helper()
	return newMWFixtureOn(t, nil, checks, wait)
}

// newMWFixtureOn is newMWFixture on the ledger lines (mxReadyLedger when
// nil).
func newMWFixtureOn(t *testing.T, lines []any, checks mgReply, wait time.Duration) *mwFixture {
	t.Helper()
	fx := &mwFixture{mrFixture: newMRFixture(t, lines), progress: &strings.Builder{}}
	fx.ctx, fx.cancel = context.WithCancelCause(context.Background())
	t.Cleanup(func() { fx.cancel(nil) })
	fx.clock = &mwClock{t: t, now: time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC), stop: fx.cancel, onSleep: func(n int) {
		lock, err := acquireHeldFileLock(fx.mo.stateDir, regateLockFile, "test", 0)
		if err != nil {
			t.Errorf("regate.lock while Merge waits: %v; want it free", err)
		} else {
			fx.lockTaken++
			_ = lock.release()
		}
		if fx.onSleep != nil {
			fx.onSleep(n)
		}
	}}
	fx.exec.replies[mxPRChecks] = checks
	fx.p = fx.params()
	fx.p.WaitChecks, fx.p.Progress = wait, fx.progress
	fx.p.waitTest = mergeWaitTest{now: fx.clock.Now, sleep: fx.clock.sleep}
	return fx
}

// setChecks makes gh pr checks answer r from now on.
func (fx *mwFixture) setChecks(r mgReply) {
	fx.exec.mu.Lock()
	defer fx.exec.mu.Unlock()
	fx.exec.replies[mxPRChecks] = r
}

// afterCall runs f once the call with index n (from 0) has its answer.
func (fx *mwFixture) afterCall(n int, f func()) {
	fx.exec.after = func(i int) {
		if i == n {
			f()
		}
	}
}

// lines are the lines Merge wrote to Progress.
func (fx *mwFixture) lines() []string {
	var out []string
	for line := range strings.Lines(fx.progress.String()) {
		out = append(out, strings.TrimSuffix(line, "\n"))
	}
	return out
}

// check fails the test unless the commands, the sleeps, and the lines are
// keys, sleeps, and lines, and regate.lock was free in every sleep and is
// free now.
func (fx *mwFixture) check(t *testing.T, keys []string, sleeps []time.Duration, lines []string) {
	t.Helper()
	if got := fx.exec.keys(); !slices.Equal(got, keys) {
		t.Errorf("commands = %q\nwant %q", got, keys)
	}
	if !slices.Equal(fx.clock.sleeps, sleeps) {
		t.Errorf("sleeps = %v, want %v", fx.clock.sleeps, sleeps)
	}
	if fx.lockTaken != len(fx.clock.sleeps) {
		t.Errorf("regate.lock was taken in %d of the %d sleeps; want every one", fx.lockTaken, len(fx.clock.sleeps))
	}
	if got := fx.lines(); !slices.Equal(got, lines) {
		t.Errorf("Progress =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(lines, "\n"))
	}
	mrAssertLockFree(t, fx.mo.stateDir)
}

// mwRounds are the commands of rounds of checkMerge that refused at gh pr
// checks, each followed by the reads of the wait after it (polls[i]), and
// then tail.
func mwRounds(polls []int, tail ...string) []string {
	var keys []string
	for _, n := range polls {
		keys = append(keys, mxAll[:5]...)
		for range n {
			keys = append(keys, mxPRChecks)
		}
	}
	return append(keys, tail...)
}

// mwMerged is the record of the default fixture's merge.
func mwMerged(t *testing.T) string {
	return mrRecord(t, "branch=feat/a pr=7 head="+moHeadA+" method=merge merge_commit="+mrOID+" auto_merge=false")
}

// TestMerge_WaitChecksWaitsOutPendingOrNoChecks covers AC13's wait that
// ends in a merge: checks pending, none as [], and none as gh's sentence of
// no checks (which then turns pending), each with --wait-checks 1h, are
// waited out with regate.lock released, reading only gh pr checks every 30 s;
// once they passed, every check runs again and the PR is merged and
// recorded once, with one line when the wait starts and one when it ends.
func TestMerge_WaitChecksWaitsOutPendingOrNoChecks(t *testing.T) {
	for _, tc := range []struct {
		name   string
		first  mgReply
		after  []mgReply // gh pr checks' answer after each sleep
		why    string
		waited string
	}{
		{"pending, then passed", mwPending, []mgReply{mwPass}, "has checks pending", "30s"},
		{"none as [], then passed", mwNone, []mgReply{mwPass}, "has no checks reported yet", "30s"},
		{"none as gh's sentence, then pending, then passed", mwNoneSentence, []mgReply{mwPending, mwPass},
			"has no checks reported yet", "1m0s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMWFixture(t, tc.first, time.Hour)
			fx.onSleep = func(n int) { fx.setChecks(tc.after[n-1]) }
			res, err := fx.run(t)
			want := mrResult("merge")
			want.Merged, want.Recorded, want.MergeCommit = true, true, mrOID
			if err != nil || res != want {
				t.Fatalf("Merge = %+v, %v\nwant    %+v", res, err, want)
			}
			sleeps := slices.Repeat([]time.Duration{30 * time.Second}, len(tc.after))
			fx.check(t, mwRounds([]int{len(tc.after)}, slices.Concat(mxAll, []string{mrMerge, mrReread})...), sleeps,
				[]string{mwStart(tc.why, "1h0m0s", "1h0m0s"), mwEnd(tc.waited, "the checks passed")})
			if got := fx.appended(t); got != mwMerged(t) {
				t.Errorf("appended %q\nwant     %q", got, mwMerged(t))
			}
		})
	}
}

// fx.run runs Merge with fx.p and fx.ctx.
func (fx *mwFixture) run(t *testing.T) (MergeResult, error) {
	t.Helper()
	return fx.merge(fx.ctx, t, fx.p)
}

// TestMerge_WaitChecksEndsAndChecksAgain covers the other ends of a wait in
// AC13: a check that fails ends it and the checks again refuse, with
// nothing merged and no time added to the refusal; gh pr checks failing ends
// it and the checks run again (here they pass); a branch that moved while
// Merge waited is refused by the checks again; and checks pending again in
// the checks after a wait are waited for again, in a second wait, whose
// start line rounds what is left of the time to the second when the checks
// before it took a fraction of one.
func TestMerge_WaitChecksEndsAndChecksAgain(t *testing.T) {
	merged := mrResult("merge")
	merged.Merged, merged.Recorded, merged.MergeCommit = true, true, mrOID
	mergeTail := slices.Concat(mxAll, []string{mrMerge, mrReread})
	start := mwStart("has checks pending", "1h0m0s", "1h0m0s")
	for _, tc := range []struct {
		name  string
		setup func(fx *mwFixture)
		want  MergeResult
		keys  []string
		lines []string
	}{
		{"a check failed", func(fx *mwFixture) {
			fx.onSleep = func(int) { fx.setChecks(mwFail) }
		}, MergeResult{Refused: mwFailRefusal}, mwRounds([]int{1}, mxAll[:5]...),
			[]string{start, mwEnd("30s", "a check did not pass")}},
		{"gh pr checks failed", func(fx *mwFixture) {
			fx.onSleep = func(int) { fx.setChecks(mwGHFails) }
			fx.afterCall(5, func() { fx.setChecks(mwPass) })
		}, merged, mwRounds([]int{1}, mergeTail...), []string{start, mwEnd("30s", "gh pr checks 7 -R "+mxRepoArg+" failed")}},
		{"the branch moved", func(fx *mwFixture) {
			fx.exec.ls = []mgReply{{out: mxListing()}, {out: moListing("main", "main", moMain, "feat/a", moHeadB)}}
			fx.onSleep = func(int) { fx.setChecks(mwPass) }
		}, MergeResult{Refused: "org: merge: ralph org merge-order lists the branch feat/a as regate, not ready, so nothing was " +
			"merged: pushed after the gate, or the gate's head was not pushed: origin has bbbbbbbbbbbb, the gate checked " +
			"aaaaaaaaaaaa: run ralph org regate --branch feat/a"}, mwRounds([]int{1}, mxAll[:3]...),
			[]string{start, mwEnd("30s", "the checks passed")}},
		{"pending again after a wait", func(fx *mwFixture) {
			fx.onSleep = func(int) { fx.setChecks(mwPass) }
			fx.afterCall(5, func() { fx.setChecks(mwPending) })
		}, merged, mwRounds([]int{1, 1}, mergeTail...), []string{start, mwEnd("30s", "the checks passed"),
			mwStart("has checks pending", "59m30s", "1h0m0s"), mwEnd("30s", "the checks passed")}},
		{"pending again after checks that took 623ms", func(fx *mwFixture) {
			fx.onSleep = func(int) { fx.setChecks(mwPass) }
			fx.exec.after = func(i int) {
				switch i {
				case 5: // the read of the first wait
					fx.setChecks(mwPending)
				case 7: // the ls-remote of the checks after it
					fx.clock.now = fx.clock.now.Add(623 * time.Millisecond)
				}
			}
		}, merged, mwRounds([]int{1, 1}, mergeTail...), []string{start, mwEnd("30s", "the checks passed"),
			mwStart("has checks pending", "59m29s", "1h0m0s"), mwEnd("30s", "the checks passed")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMWFixture(t, mwPending, time.Hour)
			tc.setup(fx)
			res, err := fx.run(t)
			if err != nil || res != tc.want {
				t.Fatalf("Merge = %+v, %v\nwant    %+v", res, err, tc.want)
			}
			fx.check(t, tc.keys, slices.Repeat([]time.Duration{30 * time.Second}, len(tc.lines)/2), tc.lines)
			wantAppended := ""
			if tc.want.Recorded {
				wantAppended = mwMerged(t)
			}
			if got := fx.appended(t); got != wantAppended {
				t.Errorf("appended %q\nwant     %q", got, wantAppended)
			}
		})
	}
}

// TestMerge_WaitChecksBudget covers AC13's time limit: the sleeps never go
// past --wait-checks, the last one shortened to what is left; once it is
// used up, the checks run again and, still pending or still with none
// reported (as [] or as gh's sentence of no checks, a repository with no CI),
// refuse with how long Merge waited, counted from the start of the first
// wait, also over two waits. While it waits it reads only gh pr checks, and
// it runs no gh pr merge.
func TestMerge_WaitChecksBudget(t *testing.T) {
	for _, tc := range []struct {
		name    string
		checks  mgReply // gh pr checks' answer every time
		refusal string
		why     string
	}{
		{"pending until the time is up", mwPending, mwPendingRefusal, "has checks pending"},
		{"none as [] until the time is up", mwNone, mwNoneRefusal, "has no checks reported yet"},
		{"none as gh's sentence until the time is up", mwNoneSentence, mwNoneRefusal, "has no checks reported yet"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMWFixture(t, tc.checks, 70*time.Second)
			res, err := fx.run(t)
			want := MergeResult{Refused: tc.refusal + "; ralph org merge waited 1m10s for the checks (--wait-checks 1m10s)"}
			if err != nil || res != want {
				t.Fatalf("Merge = %+v, %v\nwant    %+v", res, err, want)
			}
			fx.check(t, mwRounds([]int{3}, mxAll[:5]...), []time.Duration{30 * time.Second, 30 * time.Second, 10 * time.Second},
				[]string{mwStart(tc.why, "1m10s", "1m10s"), mwEnd("1m10s", "--wait-checks 1m10s is used up")})
			if got := fx.appended(t); got != "" {
				t.Errorf("appended %q", got)
			}
		})
	}
	t.Run("over two waits", func(t *testing.T) {
		fx := newMWFixture(t, mwPending, 45*time.Second)
		fx.onSleep = func(n int) {
			if n == 1 {
				fx.setChecks(mwPass)
			}
		}
		fx.afterCall(5, func() { fx.setChecks(mwPending) })
		res, err := fx.run(t)
		want := MergeResult{Refused: mwPendingRefusal + "; ralph org merge waited 45s for the checks (--wait-checks 45s)"}
		if err != nil || res != want {
			t.Fatalf("Merge = %+v, %v\nwant    %+v", res, err, want)
		}
		fx.check(t, mwRounds([]int{1, 1}, mxAll[:5]...), []time.Duration{30 * time.Second, 15 * time.Second}, []string{
			mwStart("has checks pending", "45s", "45s"), mwEnd("30s", "the checks passed"),
			mwStart("has checks pending", "15s", "45s"), mwEnd("15s", "--wait-checks 45s is used up"),
		})
	})
}

// TestMerge_WaitChecksInterrupted covers AC13's interrupt: ctx done while
// Merge sleeps, or while it reads the checks, ends the wait with an error
// that says nothing was merged, no gh pr merge run, nothing recorded, and
// regate.lock free, and the end line says so.
func TestMerge_WaitChecksInterrupted(t *testing.T) {
	end := "org: merge: stopped waiting for the checks of PR #7 after 30s: interrupted (test stopped); nothing was merged"
	for _, tc := range []struct {
		name  string
		setup func(fx *mwFixture)
		keys  []string
	}{
		{"while it sleeps", func(fx *mwFixture) {
			fx.onSleep = func(int) { fx.cancel(errors.New("test stopped")) }
		}, mxAll[:5]},
		{"while it reads the checks", func(fx *mwFixture) {
			fx.afterCall(5, func() { fx.cancel(errors.New("test stopped")) })
		}, mwRounds([]int{1})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMWFixture(t, mwPending, time.Hour)
			tc.setup(fx)
			res, err := fx.run(t)
			if err == nil || err.Error() != mwInterrupted || res != (MergeResult{}) {
				t.Fatalf("Merge = %+v, %v\nwant the error %q", res, err, mwInterrupted)
			}
			fx.check(t, tc.keys, []time.Duration{30 * time.Second}, []string{mwStart("has checks pending", "1h0m0s", "1h0m0s"), end})
			if got := fx.appended(t); got != "" {
				t.Errorf("appended %q", got)
			}
		})
	}
}

// TestMerge_WaitChecksReleasesRegateLock covers D3 with AC13: a regate that
// takes regate.lock while Merge waits and leaves its marker there does not
// hold off the merge after the wait, which takes the lock again and empties
// the marker before it merges; a lock still held when the wait ends is
// refused as it is without a wait.
func TestMerge_WaitChecksReleasesRegateLock(t *testing.T) {
	t.Run("a regate ran meanwhile", func(t *testing.T) {
		fx := newMWFixture(t, mwPending, time.Hour)
		path := regateLockPathIn(fx.mo.stateDir)
		fx.onSleep = func(int) {
			other, err := acquireHeldFileLock(fx.mo.stateDir, regateLockFile, "regate", 0)
			if err != nil {
				t.Fatalf("regate.lock while Merge waits: %v", err)
			}
			if err := writeRegateMarker(other.f, &regateMarker{OrgID: "org-a", Branch: "feat/a", Worktree: "/w", PID: 1}); err != nil {
				t.Fatal(err)
			}
			_ = other.release()
			fx.setChecks(mwPass)
		}
		var held bool
		var raw string
		var spawnErr error
		fx.ex.inMerge = func(context.Context) {
			held, _ = fileLockHeld(path)
			raw, spawnErr = fileOrAbsent(t, path), regateSpawnErr(fx.mo.stateDir, "org-a")
		}
		if res, err := fx.run(t); err != nil || !res.Recorded {
			t.Fatalf("Merge = %+v, %v", res, err)
		}
		if !held || raw != "" || spawnErr != nil {
			t.Errorf("during the merge: lock held %v, regate.lock holds %q, regateSpawnErr = %v; want held, empty, nil", held, raw,
				spawnErr)
		}
	})
	t.Run("the lock is held when the wait ends", func(t *testing.T) {
		fx := newMWFixture(t, mwPending, time.Hour)
		var other *heldFileLock
		fx.onSleep = func(int) {
			var err error
			if other, err = acquireHeldFileLock(fx.mo.stateDir, regateLockFile, "regate", 0); err != nil {
				t.Fatalf("regate.lock while Merge waits: %v", err)
			}
			fx.setChecks(mwPass)
		}
		res, err := fx.run(t)
		if other == nil {
			t.Fatal("Merge did not sleep")
		}
		defer func() { _ = other.release() }()
		want := MergeResult{Refused: "org: merge: another ralph org regate or merge runs on this ledger (" +
			regateLockPathIn(fx.mo.stateDir) + " is held): one runs at a time, so wait for it to end and run ralph org merge " +
			"--branch feat/a again"}
		if err != nil || res != want {
			t.Fatalf("Merge = %+v, %v\nwant    %+v", res, err, want)
		}
		if got := fx.exec.keys(); !slices.Equal(got, mwRounds([]int{1})) {
			t.Errorf("commands = %q\nwant %q", got, mwRounds([]int{1}))
		}
	})
}

// mwOrgRunning is a ledger where org-a still runs the feature of feat/a: its
// binding and its gate record, and no disband.
func mwOrgRunning() []any {
	return []any{moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain)}
}

// mwOrgRefusal is the refusal of step 8 on mwOrgRunning.
const mwOrgRefusal = `org: merge: the branch feat/a is bound to org_id "org-a", which is running, so its leader may still push ` +
	"to the branch and open or change the PR, and nothing was merged: handle the org's RESULT in the inbox (ralph org inbox), and " +
	"once the org is disbanded, " + mxAgain

// mwOrgStart is the line a wait for org-a to be disbanded writes when it
// starts.
func mwOrgStart(left, budget string) string {
	return `org: merge: org_id "org-a" of the branch is still running (its leader disbands it after its RESULT), so ralph org ` +
		"merge waits for it to be disbanded for up to " + left + " (--wait-checks " + budget + "), with regate.lock released, " +
		"reading the ledger every 30s"
}

// mwOrgEnd is the line a wait for org-a that did not end in ctx done writes
// when it ends.
func mwOrgEnd(after, why string) string {
	return `org: merge: stopped waiting for org_id "org-a" to be disbanded after ` + after + ": " + why + "; running every check again"
}

// addDuring appends lines to the ledger in the sleep numbered n (from 1)
// and returns a pointer to what that appended, as the manifest holds it.
func (fx *mwFixture) addDuring(t *testing.T, n int, lines ...any) *string {
	var added string
	fx.onSleep = func(i int) {
		if i == n {
			before := mrRead(t, fx.mo.path)
			fx.mo.add(t, lines...)
			added = strings.TrimPrefix(mrRead(t, fx.mo.path), before)
		}
	}
	return &added
}

// TestMerge_WaitChecksWaitsForTheOrgToBeDisbanded covers the wait for the
// org (D5, D13, AC11, AC13): with --wait-checks 1h, step 8's refusal for an
// org that still runs the branch's feature is waited out with regate.lock
// released, reading only the ledger every 30 s, with no git or gh between the
// checks before and after it; once the org is disbanded, every check runs
// again and the PR is merged and recorded once. An org that runs another
// feature by the next read ends the wait; the checks refuse it without a wait.
func TestMerge_WaitChecksWaitsForTheOrgToBeDisbanded(t *testing.T) {
	merged := mrResult("merge")
	merged.Merged, merged.Recorded, merged.MergeCommit = true, true, mrOID
	for _, tc := range []struct {
		name   string
		sleep  int   // the sleep in which lines are appended
		lines  []any // what is appended then
		want   MergeResult
		keys   []string
		waited string
		ended  string
	}{
		{"disbanded in the first sleep", 1, []any{moDisbanded("org-a")}, merged,
			slices.Concat(mxAll[:3], mxAll, []string{mrMerge, mrReread}), "30s", "the org was disbanded"},
		{"disbanded in the third sleep", 3, []any{moDisbanded("org-a")}, merged,
			slices.Concat(mxAll[:3], mxAll, []string{mrMerge, mrReread}), "1m30s", "the org was disbanded"},
		{"runs another feature now", 1, []any{moDisbanded("org-a"), moBindTo("org-a", "split-1", "other", "feat/other")},
			MergeResult{Refused: `org: merge: the branch feat/a is bound to org_id "org-a", which is running, now for another ` +
				"feature on the branch feat/other, so nothing was merged: wait for the org to be disbanded and " + mxAgain},
			slices.Concat(mxAll[:3], mxAll[:3]), "30s", "the org runs another feature now"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMWFixtureOn(t, mwOrgRunning(), mwPass, time.Hour)
			added := fx.addDuring(t, tc.sleep, tc.lines...)
			res, err := fx.run(t)
			if err != nil || res != tc.want {
				t.Fatalf("Merge = %+v, %v\nwant    %+v", res, err, tc.want)
			}
			fx.check(t, tc.keys, slices.Repeat([]time.Duration{30 * time.Second}, tc.sleep),
				[]string{mwOrgStart("1h0m0s", "1h0m0s"), mwOrgEnd(tc.waited, tc.ended)})
			want := *added
			if tc.want.Recorded {
				want += mwMerged(t)
			}
			if got := fx.appended(t); got != want {
				t.Errorf("appended %q\nwant     %q", got, want)
			}
		})
	}
}

// TestMerge_WaitChecksForTheOrgEnds covers the other ends of the wait for
// the org: once --wait-checks is used up with the org still running, the
// checks run again and refuse with how long Merge waited and how to see what
// the org does (whether to disband it is a human's call), with no gh pr
// merge and nothing recorded; with no --wait-checks the refusal is as it was,
// with no wait; a ledger that cannot be read ends the wait, and the checks
// after it fail with the read's error; and ctx done while Merge sleeps is an
// error that says what it waited for and that nothing was merged.
func TestMerge_WaitChecksForTheOrgEnds(t *testing.T) {
	t.Run("the time is up", func(t *testing.T) {
		fx := newMWFixtureOn(t, mwOrgRunning(), mwPass, 70*time.Second)
		res, err := fx.run(t)
		want := MergeResult{Refused: mwOrgRefusal + `; ralph org merge waited 1m10s for org_id "org-a" to be disbanded ` +
			"(--wait-checks 1m10s); the org still runs: see what it does with ralph org status --org-id org-a, and a human " +
			"decides whether to disband it (ralph org disband --org-id org-a)"}
		if err != nil || res != want {
			t.Fatalf("Merge = %+v, %v\nwant    %+v", res, err, want)
		}
		fx.check(t, slices.Concat(mxAll[:3], mxAll[:3]), []time.Duration{30 * time.Second, 30 * time.Second, 10 * time.Second},
			[]string{mwOrgStart("1m10s", "1m10s"), mwOrgEnd("1m10s", "--wait-checks 1m10s is used up")})
		if got := fx.appended(t); got != "" {
			t.Errorf("appended %q", got)
		}
	})
	t.Run("no --wait-checks", func(t *testing.T) {
		fx := newMWFixtureOn(t, mwOrgRunning(), mwPass, 0)
		if res, err := fx.run(t); err != nil || res != (MergeResult{Refused: mwOrgRefusal}) {
			t.Fatalf("Merge = %+v, %v\nwant the refusal %q", res, err, mwOrgRefusal)
		}
		fx.check(t, mxAll[:3], nil, nil)
	})
	t.Run("the ledger cannot be read", func(t *testing.T) {
		fx := newMWFixtureOn(t, mwOrgRunning(), mwPass, time.Hour)
		fx.onSleep = func(int) {
			if err := os.Remove(fx.mo.path); err != nil {
				t.Fatal(err)
			}
			mkdirAll(t, fx.mo.path)
		}
		res, err := fx.run(t)
		if err == nil || !strings.HasPrefix(err.Error(), "org: merge: read manifest: ") || res != (MergeResult{}) {
			t.Fatalf("Merge = %+v, %v; want the error of the manifest that cannot be read", res, err)
		}
		lines := fx.lines()
		if len(lines) != 2 || lines[0] != mwOrgStart("1h0m0s", "1h0m0s") || !strings.HasPrefix(lines[1], `org: merge: stopped `+
			`waiting for org_id "org-a" to be disbanded after 30s: the ledger could not be read (`) ||
			!strings.HasSuffix(lines[1], "; running every check again") {
			t.Errorf("Progress =\n%s\nwant the start and an end for a ledger that could not be read", strings.Join(lines, "\n"))
		}
		if got := fx.exec.keys(); !slices.Equal(got, slices.Concat(mxAll[:3], mxAll[:1])) {
			t.Errorf("commands = %q\nwant %q", got, slices.Concat(mxAll[:3], mxAll[:1]))
		}
		mrAssertLockFree(t, fx.mo.stateDir)
	})
	t.Run("interrupted while it sleeps", func(t *testing.T) {
		fx := newMWFixtureOn(t, mwOrgRunning(), mwPass, time.Hour)
		fx.onSleep = func(int) { fx.cancel(errors.New("test stopped")) }
		res, err := fx.run(t)
		const want = `org: merge: interrupted (test stopped) while waiting for org_id "org-a" to be disbanded, before gh pr ` +
			"merge ran: nothing was merged"
		if err == nil || err.Error() != want || res != (MergeResult{}) {
			t.Fatalf("Merge = %+v, %v\nwant the error %q", res, err, want)
		}
		fx.check(t, mxAll[:3], []time.Duration{30 * time.Second}, []string{mwOrgStart("1h0m0s", "1h0m0s"),
			`org: merge: stopped waiting for org_id "org-a" to be disbanded after 30s: interrupted (test stopped); nothing was merged`})
		if got := fx.appended(t); got != "" {
			t.Errorf("appended %q", got)
		}
	})
}

// TestMerge_WaitChecksOrgAndChecksShareTheBudget: a wait for the org and,
// once it was disbanded, a wait for checks still pending share one
// --wait-checks, counted from the start of the first wait, and the refusal
// once it is used up says Merge waited for both.
func TestMerge_WaitChecksOrgAndChecksShareTheBudget(t *testing.T) {
	fx := newMWFixtureOn(t, mwOrgRunning(), mwPending, 70*time.Second)
	added := fx.addDuring(t, 1, moDisbanded("org-a"))
	res, err := fx.run(t)
	want := MergeResult{Refused: mwPendingRefusal + `; ralph org merge waited 1m10s for org_id "org-a" to be disbanded and for ` +
		"the checks (--wait-checks 1m10s)"}
	if err != nil || res != want {
		t.Fatalf("Merge = %+v, %v\nwant    %+v", res, err, want)
	}
	fx.check(t, slices.Concat(mxAll[:3], mwRounds([]int{2}, mxAll[:5]...)),
		[]time.Duration{30 * time.Second, 30 * time.Second, 10 * time.Second}, []string{
			mwOrgStart("1m10s", "1m10s"), mwOrgEnd("30s", "the org was disbanded"),
			mwStart("has checks pending", "40s", "1m10s"), mwEnd("40s", "--wait-checks 1m10s is used up"),
		})
	if got := fx.appended(t); got != *added {
		t.Errorf("appended %q\nwant     %q", got, *added)
	}
}

// TestMerge_WaitChecksDoesNotWaitOutOtherRefusals covers AC13's refusals
// that are not waited out: with --wait-checks 1h, checks that failed, were
// cancelled, are all skipping, have a bucket not known, or are pending beside
// one that failed, gh pr checks failing or printing no list, the branch's org
// running another feature now (D5), and a refusal of another step, each
// refuse at once as with no --wait-checks, with no sleep and no line; and
// with --wait-checks 0, pending checks refuse at once.
func TestMerge_WaitChecksDoesNotWaitOutOtherRefusals(t *testing.T) {
	check := func(bucket string) ghPRCheck {
		return ghPRCheck{Bucket: bucket, Name: "lint", State: "X", Workflow: "ci"}
	}
	for _, tc := range []struct {
		name  string
		setup func(fx *mwFixture)
	}{
		{"a check failed", func(fx *mwFixture) { fx.setChecks(mwFail) }},
		{"a check was cancelled", func(fx *mwFixture) { fx.setChecks(mgReply{out: mxChecks(mxPass("verify"), check("cancel"))}) }},
		{"skipping only", func(fx *mwFixture) { fx.setChecks(mgReply{out: mxChecks(check("skipping"))}) }},
		{"a bucket not known", func(fx *mwFixture) { fx.setChecks(mgReply{out: mxChecks(mxPass("verify"), check("PASS"))}) }},
		{"pending beside a failed check", func(fx *mwFixture) {
			fx.setChecks(mgReply{out: mxChecks(check("pending"), check("fail"))})
		}},
		{"gh pr checks fails", func(fx *mwFixture) { fx.setChecks(mwGHFails) }},
		{"gh's sentence for another branch", func(fx *mwFixture) {
			fx.setChecks(mgReply{err: errors.New("exit status 1: no checks reported on the 'feat/b' branch")})
		}},
		{"gh pr checks prints no list", func(fx *mwFixture) { fx.setChecks(mgReply{out: "<html>"}) }},
		{"the org runs another feature now", func(fx *mwFixture) {
			fx.mo.add(fx.clock.t, moBindTo("org-a", "split-1", "other", "feat/other"))
		}},
		{"no PR", func(fx *mwFixture) { fx.exec.replies[mxPRList] = mgReply{out: "[]"} }},
		{"PR not OPEN", func(fx *mwFixture) {
			fx.exec.replies[mxPRView] = mgReply{out: mxPRViewJSON("CLOSED", moHeadA, "main", "https://github.com/o/r/pull/7", "null")}
		}},
		{"no --wait-checks, checks pending", func(fx *mwFixture) { fx.p.WaitChecks = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The control: the refusal Merge gives with no --wait-checks.
			control := newMWFixture(t, mwPending, 0)
			tc.setup(control)
			control.p.WaitChecks = 0
			want, err := control.run(t)
			if err != nil || want.Refused == "" {
				t.Fatalf("Merge with no --wait-checks = %+v, %v; want a refusal", want, err)
			}
			fx := newMWFixture(t, mwPending, time.Hour)
			tc.setup(fx)
			res, err := fx.run(t)
			if err != nil || res != want {
				t.Fatalf("Merge = %+v, %v\nwant    %+v", res, err, want)
			}
			fx.check(t, control.exec.keys(), nil, nil)
		})
	}
}

// TestMerge_WaitChecksOutOfRangeIsRefusedBeforeAnything covers AC13's
// range: --wait-checks below 0 or above 6h is refused before any command,
// with no regate.lock made and nothing recorded, an empty branch included;
// 6h itself is taken.
func TestMerge_WaitChecksOutOfRangeIsRefusedBeforeAnything(t *testing.T) {
	for _, tc := range []struct {
		d      time.Duration
		branch string
		shown  string
	}{
		{-time.Nanosecond, "feat/a", "-1ns"},
		{-time.Second, "feat/a", "-1s"},
		{6*time.Hour + time.Nanosecond, "feat/a", "6h0m0.000000001s"},
		{7 * time.Hour, "", "7h0m0s"},
	} {
		t.Run(tc.shown, func(t *testing.T) {
			fx := newMWFixture(t, mwPending, tc.d)
			fx.p.Branch = tc.branch
			res, err := fx.run(t)
			want := MergeResult{Refused: "org: merge: --wait-checks " + tc.shown + " is out of range, so nothing was merged: give a " +
				"duration from 0 (no wait, the default) to 6h0m0s, such as 60m"}
			if err != nil || res != want || MergeWaitChecksRefusal(tc.d) != want.Refused {
				t.Fatalf("Merge = %+v, %v\nwant    %+v", res, err, want)
			}
			if got := fx.exec.keys(); len(got) != 0 {
				t.Errorf("commands = %q, want none", got)
			}
			if got := fileOrAbsent(t, regateLockPathIn(fx.mo.stateDir)); got != "<absent>" {
				t.Errorf("regate.lock holds %q; want no lock taken", got)
			}
			if got := fx.appended(t); got != "" || len(fx.lines()) != 0 {
				t.Errorf("appended %q, Progress %q", got, fx.progress)
			}
		})
	}
	for _, d := range []time.Duration{0, 6 * time.Hour} {
		if got := MergeWaitChecksRefusal(d); got != "" {
			t.Errorf("MergeWaitChecksRefusal(%s) = %q, want none", d, got)
		}
	}
	fx := newMWFixture(t, mwPass, 6*time.Hour)
	if res, err := fx.run(t); err != nil || !res.Recorded {
		t.Fatalf("Merge with --wait-checks 6h = %+v, %v; want the merge", res, err)
	}
	fx.check(t, slices.Concat(mxAll, []string{mrMerge, mrReread}), nil, nil)
}

// TestMergeWait_Defaults: the interval is 30 s, the longest wait 6 h, and a
// wait with no test seams sleeps on a timer by the real clock and writes its
// lines nowhere when Progress is nil. The fake clock tests above leave the
// interval as it is, so their 30 s sleeps are the default's.
func TestMergeWait_Defaults(t *testing.T) {
	if mergeWaitInterval != 30*time.Second || maxMergeWaitChecks != 6*time.Hour {
		t.Fatalf("mergeWaitInterval = %s, maxMergeWaitChecks = %s; want 30s and 6h", mergeWaitInterval, maxMergeWaitChecks)
	}
	w := newMergeWait(MergeParams{WaitChecks: time.Hour})
	if w.budget != time.Hour || w.interval != mergeWaitInterval || w.progress != io.Discard ||
		reflect.ValueOf(w.now).Pointer() != reflect.ValueOf(time.Now).Pointer() ||
		reflect.ValueOf(w.sleep).Pointer() != reflect.ValueOf(sleepContext).Pointer() {
		t.Errorf("newMergeWait = %+v; want the budget, 30s, io.Discard, time.Now and sleepContext", w)
	}
}

// TestMerge_WaitChecksWithTheRealClock runs a wait on the real timer, with
// the interval made short: the time is used up after the budget and before
// long, and ctx done ends a sleep of an hour at once.
func TestMerge_WaitChecksWithTheRealClock(t *testing.T) {
	t.Run("the time is used up", func(t *testing.T) {
		fx := newMWFixture(t, mwPending, 60*time.Millisecond)
		fx.p.waitTest = mergeWaitTest{interval: 10 * time.Millisecond}
		begin := time.Now()
		res, err := fx.run(t)
		took := time.Since(begin)
		if err != nil || !strings.HasPrefix(res.Refused, mwPendingRefusal+"; ralph org merge waited ") ||
			!strings.HasSuffix(res.Refused, " for the checks (--wait-checks 60ms)") {
			t.Fatalf("Merge = %+v, %v; want the refusal with the time waited", res, err)
		}
		// One read or more, not two: a first sleep that load delays by 50ms or
		// more leaves no time for a second; the mwClock tests count the reads.
		if polls := strings.Count(strings.Join(fx.exec.keys(), "\n"), mxPRChecks) - 2; took < 60*time.Millisecond ||
			took > 10*time.Second || polls < 1 {
			t.Errorf("Merge took %s with %d reads of the checks while it waited; want 60ms or more, and one read or more", took,
				polls)
		}
	})
	t.Run("ctx done while it sleeps", func(t *testing.T) {
		fx := newMWFixture(t, mwPending, time.Hour)
		fx.p.waitTest = mergeWaitTest{interval: time.Hour}
		go func() {
			time.Sleep(50 * time.Millisecond)
			fx.cancel(errors.New("test stopped"))
		}()
		begin := time.Now()
		res, err := fx.run(t)
		if took := time.Since(begin); err == nil || err.Error() != mwInterrupted || res != (MergeResult{}) || took > 10*time.Second {
			t.Fatalf("Merge = %+v, %v after %s; want the interruption at once", res, err, took)
		}
		mrAssertLockFree(t, fx.mo.stateDir)
	})
}
