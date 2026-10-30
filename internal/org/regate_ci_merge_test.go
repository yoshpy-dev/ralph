package org

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The merge path of a ci regate (the fixture and helpers are in
// regate_ci_test.go): the push of the merge commit before the record, the
// read-back of origin's tip after a failed push, and the interrupts on that
// path.

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
				// Self-review L2: the undo is ralph's sentence, before what
				// git said, not a line of git's indented output.
				want.Undone = true
				prefix = failed + "origin's feat/auth is still " + pre + ", the tip the merge commit was made on, so the push did not land " +
					"and nothing was recorded: fix it and " + rcAgain + "; " + regateUndoneTextFor(merge, pre)
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
			if rest := strings.TrimPrefix(res.Refused, prefix); tc.outcome == undone && rest != "" &&
				(!strings.HasPrefix(rest, "; git said:\n  ") || strings.Contains(rest, "undone")) {
				t.Errorf("refusal = %q\nwant only what git said after %q", res.Refused, prefix)
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
