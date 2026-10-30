package org

import (
	"errors"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

// The wait of a ci regate and regate.lock, and the branch moving under it
// (rcWaitFixture and its helpers are in regate_ci_wait_test.go).

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

// TestRegateCI_WaitChecksPendingUnderTheLockAfterGHFailed covers self-review
// R2-4: when the wait's read after a sleep stops the poll because gh could
// not read PR #7's head or its checks (it fails, or prints no JSON list), the
// head read again once the checks were read included, and the read with
// regate.lock taken again finds the checks pending, ChecksPending and the
// line say that gh failed, not that the checks were not done again, since
// they never were. A read that stops the poll at every check skipping, which
// is no check pending, says "again" as after checks that passed.
func TestRegateCI_WaitChecksPendingUnderTheLockAfterGHFailed(t *testing.T) {
	const ghFailed = "gh failed to read the PR's head or its checks after a sleep, and the checks were not done when regate read " +
		"them with regate.lock taken again"
	bad := mgReply{err: errors.New("exit status 1: HTTP 502: Bad Gateway")}
	for _, tc := range []struct {
		name string
		// sleep sets the answers of the read after the sleep; afterCall,
		// when set, changes them after the call with index n; at is the
		// index of that read's last call, after which the head is HEAD and
		// the checks are pending again.
		sleep     func(fx *rcWaitFixture)
		afterCall func(fx *rcWaitFixture, n int)
		at        int
		keys      []string
		reads     string
		ghFailed  bool
	}{
		{"gh pr view fails", func(fx *rcWaitFixture) { fx.reply(mxPRView, bad) }, nil, 4,
			slices.Concat(rcChecksKeys, []string{mxPRView}, rcPendingRead), "LL" + "." + "LL", true},
		{"gh pr checks fails", func(fx *rcWaitFixture) { fx.reply(mxPRChecks, bad) }, nil, 5,
			slices.Concat(rcChecksKeys, rcPendingRead, rcPendingRead), "LL" + ".." + "LL", true},
		{"gh pr checks prints no JSON list", func(fx *rcWaitFixture) { fx.reply(mxPRChecks, mgReply{out: "<html>"}) }, nil, 5,
			slices.Concat(rcChecksKeys, rcPendingRead, rcPendingRead), "LL" + ".." + "LL", true},
		{"gh pr view fails once the checks were read", func(fx *rcWaitFixture) {
			fx.reply(mxPRChecks, mgReply{out: mxChecks(mxPass("verify"))})
		}, func(fx *rcWaitFixture, n int) {
			if n == 5 { // gh pr checks, which passed
				fx.reply(mxPRView, bad)
			}
		}, 6, slices.Concat(rcChecksKeys, rcDoneRead, rcPendingRead), "LL" + "..." + "LL", true},
		{"every check is skipping", func(fx *rcWaitFixture) {
			fx.reply(mxPRChecks, mgReply{out: mxChecks(rcCheck("skipping", "deploy", "SKIPPED"))})
		}, nil, 6, slices.Concat(rcChecksKeys, rcDoneRead, rcPendingRead), "LL" + "..." + "LL", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRCWaitFixture(t, time.Hour)
			fx.exec.replies[mxPRChecks] = rcPending
			fx.onSleep = func(int) { tc.sleep(fx) }
			fx.afterCall = func(n int) {
				if tc.afterCall != nil {
					tc.afterCall(fx, n)
				}
				if n == tc.at {
					fx.reply(mxPRView, rcPRHead(fx.head))
					fx.reply(mxPRChecks, rcPending)
				}
			}
			found, line := "they were not done again when it read them with regate.lock taken again",
				"the checks were not done again when it read them with regate.lock taken again"
			if tc.ghFailed {
				found, line = "it stopped once "+ghFailed, ghFailed
			}
			manifest := fileOrAbsent(t, fx.o.Manifest.Path())
			res, err := fx.run()
			want := RegateResult{Outcome: RegateOutcomeChecksPending, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth",
				Head: fx.head, Base: fx.tip, BaseRef: "refs/remotes/origin/main", PR: 7, ChecksNext: rcNext,
				ChecksPending: rcPendingWhy + "; ralph org regate waited 30s for the checks of PR #7 (--wait-checks 1h0m0s), and " + found}
			if err != nil || !reflect.DeepEqual(res, want) {
				t.Fatalf("regate = %#v, %v\nwant     %#v", res, err, want)
			}
			fx.assertKeys(t, tc.keys)
			fx.assertReads(t, tc.reads)
			fx.assertWaited(t, []time.Duration{30 * time.Second}, []string{rcWaitStart(rcPendingWhy, "1h0m0s"),
				rcWaitEnd("30s", line+", so nothing was recorded")})
			fx.assertNothingWritten(t, manifest)
		})
	}
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
