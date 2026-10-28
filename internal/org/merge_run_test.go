package org

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// The merge of `ralph org merge` at the org layer (plan
// docs/plans/active/2026-10-17-org-auto-merge.md, S3b: the org-layer half of
// AC6 to AC10): Merge on the manifest fixtures of merge_fake_test.go, with
// mrExec answering the calls Merge adds after checkMerge's, so no test runs
// gh or reaches a network.

// TestMerge_MergesWithTheMethodAndRecordsPRMerged covers AC6: for each
// method, gh pr merge runs once as `gh pr merge 7 <flag> --match-head-commit
// <the gate's head> -R github.com/o/r`, the PR is read back once, pr_merged
// is appended once with its exact Details, MergeResult holds the merge, and
// regate.lock is released. AutoMerge, given both ways for one method, only
// shows in the result and the record.
func TestMerge_MergesWithTheMethodAndRecordsPRMerged(t *testing.T) {
	for _, tc := range []struct {
		name, value, method, flag string
		autoMerge                 bool
	}{
		{"MERGE", "MERGE", "merge", "--merge", false},
		{"MERGE with auto_merge", "MERGE", "merge", "--merge", true},
		{"SQUASH", "SQUASH", "squash", "--squash", false},
		{"REBASE with auto_merge", "REBASE", "rebase", "--rebase", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMRFixture(t, nil)
			fx.exec.replies[mxRepoView] = mgReply{out: `{"viewerDefaultMergeMethod":"` + tc.value + `"}`}
			p := fx.params()
			p.AutoMerge = tc.autoMerge
			res, err := fx.merge(context.Background(), t, p)
			want := mrResult(tc.method)
			want.Merged, want.Recorded, want.MergeCommit, want.AutoMerge = true, true, mrOID, tc.autoMerge
			if err != nil || res != want {
				t.Fatalf("Merge = %+v, %v\nwant    %+v", res, err, want)
			}
			if got := fx.exec.keys(); !slices.Equal(got, mrMerged) {
				t.Errorf("commands = %q\nwant %q", got, mrMerged)
			}
			for key, argv := range map[string][]string{
				mrMerge:  {"gh", "pr", "merge", "7", tc.flag, "--match-head-commit", moHeadA, "-R", mxRepoArg},
				mrReread: {"gh", "pr", "view", "7", "-R", mxRepoArg, "--json", "state,headRefOid,baseRefName,mergeCommit,autoMergeRequest"},
			} {
				if c := fx.call(t, key); !slices.Equal(append([]string{c.name}, c.args...), argv) {
					t.Errorf("%s = %q\nwant %q", key, append([]string{c.name}, c.args...), argv)
				}
			}
			details := fmt.Sprintf("branch=feat/a pr=7 head=%s method=%s merge_commit=%s auto_merge=%t", moHeadA, tc.method, mrOID,
				tc.autoMerge)
			if got, want := fx.appended(t), mrRecord(t, details); got != want {
				t.Errorf("appended %q\nwant     %q", got, want)
			}
			mrAssertLockFree(t, fx.mo.stateDir)
		})
	}
}

// TestMerge_RecordsAnUnknownMergeCommit: a read-back that gives no merge
// commit, or one that is not a commit SHA, is recorded as
// merge_commit=unknown, with MergeCommit empty.
func TestMerge_RecordsAnUnknownMergeCommit(t *testing.T) {
	for name, oid := range map[string]string{"null": "", "not a SHA": "c0ffee x=1"} {
		t.Run(name, func(t *testing.T) {
			fx := newMRFixture(t, nil)
			fx.ex.reread = mgReply{out: mrReadBack("MERGED", oid, "null")}
			res, err := fx.merge(context.Background(), t, fx.params())
			if err != nil || !res.Merged || !res.Recorded || res.MergeCommit != "" {
				t.Fatalf("Merge = %+v, %v; want a recorded merge with no merge commit", res, err)
			}
			want := mrRecord(t, "branch=feat/a pr=7 head="+moHeadA+" method=merge merge_commit=unknown auto_merge=false")
			if got := fx.appended(t); got != want {
				t.Errorf("appended %q\nwant     %q", got, want)
			}
		})
	}
}

// TestMerge_NotMergedRecordsNothing covers AC7: gh pr merge exits 0 and the
// read-back finds the PR not MERGED. A request GitHub held is removed with gh
// pr merge --disable-auto; a removal that fails names that command; with no
// request the PR's state is named; a read-back that fails or gives no usable
// JSON names the command that checks the PR. Each is Failed with the other
// fields of the checks set, and nothing is recorded.
func TestMerge_NotMergedRecordsNothing(t *testing.T) {
	removed := "org: merge: " + mrMergeCmd + " exited 0 and did not merge PR #7, which is OPEN: GitHub held an auto-merge request " +
		"for it instead, which ralph org merge removed (" + mrDisableCmd + "), so nothing was merged or recorded: " + mxAgain +
		", which checks the PR and the protection again"
	unknown := func(why string) string {
		return "org: merge: " + mrViewCmd + " " + why + " after " + mrMergeCmd + " exited 0, so whether PR #7 was merged is unknown " +
			"and nothing was recorded: check it with " + mrViewCmd + " --json state,headRefOid,baseRefName,mergeCommit (ralph org " +
			"merge-order lists the branch as merged once it is)"
	}
	notMerged := func(state string) string {
		return "org: merge: " + mrMergeCmd + " exited 0 but PR #7 is " + state + ", not merged, so nothing was recorded: check the " +
			"PR with " + mrViewCmd + " and run ralph org merge-order again"
	}
	const notJSON = "printed output that is not the PR's JSON with state,headRefOid,baseRefName,mergeCommit,autoMergeRequest"
	for _, tc := range []struct {
		name            string
		reread, disable mgReply
		want            string
		keys            []string
	}{
		{"a request is removed", mgReply{out: mrReadBack("OPEN", "", mrRequest)}, mgReply{}, removed, mrDisabled},
		{"a request that cannot be removed", mgReply{out: mrReadBack("OPEN", "", mrRequest)},
			mgReply{err: errors.New("exit status 1: HTTP 502: Bad Gateway")}, "org: merge: " + mrMergeCmd + " exited 0 and did not " +
				"merge PR #7, which is OPEN: GitHub held an auto-merge request for it instead, and " + mrDisableCmd + " failed (exit " +
				"status 1: HTTP 502: Bad Gateway), so the request may remain and GitHub may still merge the PR apart from ralph org " +
				"merge; nothing was recorded: remove the request with " + mrDisableCmd, mrDisabled},
		{"open with no request", mgReply{out: mrReadBack("OPEN", "", "null")}, mgReply{}, notMerged("OPEN"), mrMerged},
		{"closed with no request", mgReply{out: mrReadBack("CLOSED", "", "null")}, mgReply{}, notMerged("CLOSED"), mrMerged},
		{"the read-back fails", mgReply{err: errors.New("exit status 1: HTTP 502: Bad Gateway")}, mgReply{},
			unknown("failed (exit status 1: HTTP 502: Bad Gateway)"), mrMerged},
		{"the read-back is not JSON", mgReply{out: "<html>"}, mgReply{}, unknown(notJSON), mrMerged},
		{"the read-back has no state", mgReply{out: `{"autoMergeRequest":null,"mergeCommit":null}`}, mgReply{}, unknown(notJSON), mrMerged},
		{"the read-back of an open PR leaves autoMergeRequest out", mgReply{out: `{"mergeCommit":null,"state":"OPEN"}`}, mgReply{},
			unknown(notJSON), mrMerged},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMRFixture(t, nil)
			fx.ex.reread, fx.ex.disable = tc.reread, tc.disable
			res, err := fx.merge(context.Background(), t, fx.params())
			want := mrResult("merge")
			want.Failed = tc.want
			if err != nil || res != want {
				t.Fatalf("Merge = %+v, %v\nFailed %q\nwant   %q", res, err, res.Failed, tc.want)
			}
			if got := fx.exec.keys(); !slices.Equal(got, tc.keys) {
				t.Errorf("commands = %q\nwant %q", got, tc.keys)
			}
			if slices.Equal(tc.keys, mrDisabled) {
				if c, argv := fx.call(t, mrDisable), []string{"pr", "merge", "7", "--disable-auto", "-R", mxRepoArg}; !slices.Equal(c.args, argv) {
					t.Errorf("the removal = %q, want gh %q", c.args, argv)
				}
			}
			if got := fx.appended(t); got != "" {
				t.Errorf("appended %q; want nothing recorded", got)
			}
			mrAssertLockFree(t, fx.mo.stateDir)
		})
	}
}

// TestMerge_ReadsThePRBackAfterAFailedMerge covers AC8: when gh pr merge
// fails or times out, the PR is read back once all the same. Not MERGED (as
// when GitHub refused a merge whose base moved after the last check), it is
// Failed with the last 5 lines of gh's error, escaped, and nothing recorded;
// with a request, the request is removed; MERGED, it is recorded with a Note
// that carries the same tail; a read-back that fails says so after the merge's
// own failure.
func TestMerge_ReadsThePRBackAfterAFailedMerge(t *testing.T) {
	failure := errors.New("exit status 1: X Pull request o/r#7 is not mergeable\nline 2\n\n  line 3\x1b[2J\r\nline 4\nline 5\nline 6\n")
	const said = "; gh said:\n  line 2\n  line 3\\x1b[2J\n  line 4\n  line 5\n  line 6"
	const timedOut = "; gh said:\n  context deadline exceeded"
	for _, tc := range []struct {
		name    string
		block   bool // gh pr merge times out
		reread  mgReply
		failed  string
		note    string
		keys    []string
		records bool
	}{
		{"fails, the PR is open", false, mgReply{out: mrReadBack("OPEN", "", "null")}, "org: merge: " + mrMergeCmd + " failed and PR #7 " +
			"is OPEN, not merged, so nothing was merged or recorded: run ralph org merge-order again and follow the branch's Detail" +
			said, "", mrMerged, false},
		{"fails, the PR is merged", false, mgReply{out: mrReadBack("MERGED", mrOID, "null")}, "", mrMergeCmd + " failed, but " +
			mrViewCmd + " read PR #7 back as MERGED" + said, mrMerged, true},
		{"times out, the PR is merged", true, mgReply{out: mrReadBack("MERGED", mrOID, "null")}, "", mrMergeCmd + " timed out after " +
			"30ms, but " + mrViewCmd + " read PR #7 back as MERGED" + timedOut, mrMerged, true},
		{"times out, the PR is open", true, mgReply{out: mrReadBack("OPEN", "", "null")}, "org: merge: " + mrMergeCmd + " timed out " +
			"after 30ms and PR #7 is OPEN, not merged, so nothing was merged or recorded: run ralph org merge-order again and follow " +
			"the branch's Detail" + timedOut, "", mrMerged, false},
		{"fails, a request is removed", false, mgReply{out: mrReadBack("OPEN", "", mrRequest)}, "org: merge: " + mrMergeCmd + " failed " +
			"and did not merge PR #7, which is OPEN: GitHub held an auto-merge request for it instead, which ralph org merge removed (" +
			mrDisableCmd + "), so nothing was merged or recorded: " + mxAgain + ", which checks the PR and the protection again" + said,
			"", mrDisabled, false},
		{"fails, the read-back fails", false, mgReply{err: errors.New("exit status 4: HTTP 401")}, "org: merge: " + mrViewCmd +
			" failed (exit status 4: HTTP 401) after " + mrMergeCmd + " failed, so whether PR #7 was merged is unknown and nothing " +
			"was recorded: check it with " + mrViewCmd + " --json state,headRefOid,baseRefName,mergeCommit (ralph org merge-order " +
			"lists the branch as merged once it is)" + said, "", mrMerged, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMRFixture(t, nil)
			fx.ex.merge, fx.ex.blockMerge, fx.ex.reread = mgReply{err: failure}, tc.block, tc.reread
			p := fx.params()
			p.MergeTimeout = 30 * time.Millisecond
			res, err := fx.merge(context.Background(), t, p)
			want := mrResult("merge")
			want.Failed, want.Note = tc.failed, tc.note
			if tc.records {
				want.Merged, want.Recorded, want.MergeCommit = true, true, mrOID
			}
			if err != nil || res != want {
				t.Fatalf("Merge = %+v, %v\nFailed %q\nwant   %q\nNote   %q\nwant   %q", res, err, res.Failed, tc.failed, res.Note, tc.note)
			}
			mrNoRawControl(t, res.Failed+res.Note)
			if got := fx.exec.keys(); !slices.Equal(got, tc.keys) {
				t.Errorf("commands = %q\nwant %q", got, tc.keys)
			}
			wantAppended := ""
			if tc.records {
				wantAppended = mrRecord(t, "branch=feat/a pr=7 head="+moHeadA+" method=merge merge_commit="+mrOID+" auto_merge=false")
			}
			if got := fx.appended(t); got != wantAppended {
				t.Errorf("appended %q\nwant     %q", got, wantAppended)
			}
		})
	}
}

// TestMerge_RecordsOnlyAMergeOfTheGatesHead covers D6, AC7 and AC8 for the
// head the read-back gives (cross-review #3 of the plan): a PR read back as
// MERGED at another head than the gate's, or with no head (empty or left
// out), after gh pr merge failed (--match-head-commit refused it while
// someone merged a new head) or exited 0, is Failed with Merged not set, so no
// MERGED: line, and nothing recorded; the Failed names the head merged, how gh
// pr merge ended with gh's tail, and the next steps. MERGED at the gate's head
// is recorded as before.
func TestMerge_RecordsOnlyAMergeOfTheGatesHead(t *testing.T) {
	failure := errors.New("exit status 1: GraphQL: Head branch was modified. Review and try the merge again.")
	const said = "; gh said:\n  exit status 1: GraphQL: Head branch was modified. Review and try the merge again."
	otherHead := "at the head bbbbbbbbbbbb, not aaaaaaaaaaaa, the head the gate passed, so a head the gate did not pass was merged"
	noHead := "with no head, so whether the head the gate passed (aaaaaaaaaaaa) was merged is unknown"
	for _, tc := range []struct {
		name    string
		merge   mgReply
		reread  string
		failed  string // "" for a recorded merge
		records bool
	}{
		{"gh pr merge refused, another head merged", mgReply{err: failure}, mrReadBackAt("MERGED", moHeadB, mrOID, "null"),
			mrMergedAfter("failed") + otherHead + mrNext + said, false},
		{"gh pr merge exited 0, another head merged", mgReply{}, mrReadBackAt("MERGED", moHeadB, mrOID, "null"),
			mrMergedAfter("exited 0") + otherHead + mrNext, false},
		{"MERGED with an empty head", mgReply{}, mrReadBackAt("MERGED", "", mrOID, "null"),
			mrMergedAfter("exited 0") + noHead + mrNext, false},
		{"MERGED with no headRefOid", mgReply{err: failure}, `{"autoMergeRequest":null,"mergeCommit":{"oid":"` + mrOID +
			`"},"state":"MERGED"}`, mrMergedAfter("failed") + noHead + mrNext + said, false},
		{"a head with control characters is escaped", mgReply{}, mrReadBackAt("MERGED", "b\x1b[2Jb", mrOID, "null"),
			mrMergedAfter("exited 0") + `at the head b\x1b[2Jb, not aaaaaaaaaaaa, the head the gate passed, so a head the gate did ` +
				"not pass was merged" + mrNext, false},
		{"MERGED at the gate's head", mgReply{}, mrReadBack("MERGED", mrOID, "null"), "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMRFixture(t, nil)
			fx.ex.merge, fx.ex.reread = tc.merge, mgReply{out: tc.reread}
			res, err := fx.merge(context.Background(), t, fx.params())
			want := mrResult("merge")
			want.Failed = tc.failed
			if tc.records {
				want.Merged, want.Recorded, want.MergeCommit = true, true, mrOID
			}
			if err != nil || res != want {
				t.Fatalf("Merge = %+v, %v\nFailed %q\nwant   %q", res, err, res.Failed, tc.failed)
			}
			mrNoRawControl(t, res.Failed)
			if got := fx.exec.keys(); !slices.Equal(got, mrMerged) {
				t.Errorf("commands = %q\nwant %q", got, mrMerged)
			}
			wantAppended := ""
			if tc.records {
				wantAppended = mrRecord(t, "branch=feat/a pr=7 head="+moHeadA+" method=merge merge_commit="+mrOID+" auto_merge=false")
			}
			if got := fx.appended(t); got != wantAppended {
				t.Errorf("appended %q\nwant     %q", got, wantAppended)
			}
			mrAssertLockFree(t, fx.mo.stateDir)
		})
	}
}

// TestMerge_RecordThatFailsAfterTheMergeIsAnError covers AC8's last case:
// the PR is read back as MERGED, but pr_merged cannot be appended. Merge
// returns an error that says the PR was merged and not recorded, with Merged
// set and Recorded not.
func TestMerge_RecordThatFailsAfterTheMergeIsAnError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can append to a read-only manifest")
	}
	fx := newMRFixture(t, nil)
	if err := os.Chmod(fx.mo.path, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(fx.mo.path, 0o644) })
	res, err := fx.merge(context.Background(), t, fx.params())
	want := mrResult("merge")
	want.Merged, want.MergeCommit = true, mrOID
	prefix := "org: merge: PR #7 (https://github.com/o/r/pull/7) was merged (merge commit " + mrOID + ") but pr_merged could not " +
		"be recorded (org: open " + fx.mo.path + ": "
	suffix := "): the merge stands, and ralph org merge-order lists the branch as merged, so only the ledger misses it; check that " +
		fx.mo.path + " can be written"
	if res != want || err == nil || !strings.HasPrefix(err.Error(), prefix) || !strings.HasSuffix(err.Error(), suffix) ||
		!errors.Is(err, os.ErrPermission) {
		t.Fatalf("Merge = %+v, %v; want the merged-but-not-recorded error", res, err)
	}
	if got := fx.appended(t); got != "" {
		t.Errorf("appended %q", got)
	}
}

// TestMerge_InterruptedOnceGHPRMergeStarted covers the interruption half of
// AC7: ctx done while gh pr merge runs stops it, but the read-back, the
// removal of a request, and the record still run, once each, with no ctx
// done and the gh timeout; Merge then returns an error that says it was
// interrupted and how the merge ended.
func TestMerge_InterruptedOnceGHPRMergeStarted(t *testing.T) {
	const interrupted = "org: merge: interrupted (test stopped) once gh pr merge had started, and then: "
	const said = "; gh said:\n  context canceled"
	for _, tc := range []struct {
		name    string
		reread  mgReply
		outcome string
		keys    []string
		records bool
	}{
		{"the PR is merged", mgReply{out: mrReadBack("MERGED", mrOID, "null")}, "PR #7 (https://github.com/o/r/pull/7) was merged " +
			"(merge commit " + mrOID + ") and pr_merged was recorded", mrMerged, true},
		{"a request is removed", mgReply{out: mrReadBack("OPEN", "", mrRequest)}, mrMergeCmd + " failed and did not merge PR #7, " +
			"which is OPEN: GitHub held an auto-merge request for it instead, which ralph org merge removed (" + mrDisableCmd +
			"), so nothing was merged or recorded: " + mxAgain + ", which checks the PR and the protection again" + said, mrDisabled,
			false},
		{"the PR is open", mgReply{out: mrReadBack("OPEN", "", "null")}, mrMergeCmd + " failed and PR #7 is OPEN, not merged, so " +
			"nothing was merged or recorded: run ralph org merge-order again and follow the branch's Detail" + said, mrMerged, false},
		{"another head is merged", mgReply{out: mrReadBackAt("MERGED", moHeadB, mrOID, "null")}, mrViewCmd + " read PR #7 back " +
			"as MERGED after " + mrMergeCmd + " failed, but at the head bbbbbbbbbbbb, not aaaaaaaaaaaa, the head the gate passed, so " +
			"a head the gate did not pass was merged" + mrNext + said, mrMerged, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMRFixture(t, nil)
			fx.ex.reread = tc.reread
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			fx.ex.inMerge = func(context.Context) { cancel(errors.New("test stopped")) }
			start := time.Now()
			res, err := fx.merge(ctx, t, fx.params())
			end := time.Now()
			if err == nil || err.Error() != interrupted+tc.outcome || res.Merged != tc.records || res.Recorded != tc.records {
				t.Fatalf("Merge = %+v, %v\nwant the error %q", res, err, interrupted+tc.outcome)
			}
			if got := fx.exec.keys(); !slices.Equal(got, tc.keys) {
				t.Errorf("commands = %q\nwant %q", got, tc.keys)
			}
			for _, key := range tc.keys[len(mrMerged)-1:] {
				if d := fx.call(t, key).deadline; d.Before(start.Add(15*time.Second)) || d.After(end.Add(15*time.Second)) {
					t.Errorf("%s: deadline %v, want 15s after the start (%v..%v)", key, d, start, end)
				}
			}
			wantAppended := ""
			if tc.records {
				wantAppended = mrRecord(t, "branch=feat/a pr=7 head="+moHeadA+" method=merge merge_commit="+mrOID+" auto_merge=false")
			}
			if got := fx.appended(t); got != wantAppended {
				t.Errorf("appended %q\nwant     %q", got, wantAppended)
			}
		})
	}

	// The record waits for a manifest lock held when it comes, though ctx is
	// done: the wait is the lock's, not ctx's.
	t.Run("the manifest lock is held when the record comes", func(t *testing.T) {
		fx := newMRFixture(t, nil)
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		var release func()
		fx.ex.inMerge = func(context.Context) {
			cancel(errors.New("test stopped"))
			release = holdManifestLock(t, fx.mo.o)
			go func() {
				time.Sleep(300 * time.Millisecond)
				release()
			}()
		}
		res, err := fx.merge(ctx, t, fx.params())
		if release == nil {
			t.Fatal("gh pr merge did not run")
		}
		release()
		want := interrupted + "PR #7 (https://github.com/o/r/pull/7) was merged (merge commit " + mrOID + ") and pr_merged was recorded"
		if err == nil || err.Error() != want || !res.Recorded {
			t.Fatalf("Merge = %+v, %v\nwant the error %q", res, err, want)
		}
		if got := fx.appended(t); got != mrRecord(t, "branch=feat/a pr=7 head="+moHeadA+" method=merge merge_commit="+mrOID+
			" auto_merge=false") {
			t.Errorf("appended %q", got)
		}
	})
}

// TestMerge_InterruptedBeforeGHPRMergeMergesNothing: ctx done during the
// checks, or after they passed and before gh pr merge, is an error that says
// nothing was merged, with gh pr merge never run, nothing recorded, and
// regate.lock released.
func TestMerge_InterruptedBeforeGHPRMergeMergesNothing(t *testing.T) {
	t.Run("during the checks", func(t *testing.T) {
		fx := newMRFixture(t, nil)
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		fx.exec.after = func(n int) {
			if n == 4 { // gh pr checks
				cancel(errors.New("test stopped"))
			}
		}
		res, err := fx.merge(ctx, t, fx.params())
		if err == nil || !strings.HasPrefix(err.Error(), "org: merge: interrupted (test stopped) while ") ||
			!strings.HasSuffix(err.Error(), ": nothing was merged") || res != (MergeResult{}) {
			t.Fatalf("Merge = %+v, %v; want the interruption", res, err)
		}
		if got := fx.exec.keys(); slices.Contains(got, mrMerge) || !slices.Equal(got, mxAll[:len(got)]) {
			t.Errorf("commands = %q, want some of the checks' and no gh pr merge", got)
		}
		if got := fx.appended(t); got != "" {
			t.Errorf("appended %q", got)
		}
		mrAssertLockFree(t, fx.mo.stateDir)
	})
	t.Run("after the checks", func(t *testing.T) {
		fx := newMRFixture(t, nil)
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		mergeTestHooks.afterChecks = func() { cancel(errors.New("test stopped")) }
		t.Cleanup(func() { mergeTestHooks.afterChecks = nil })
		res, err := fx.merge(ctx, t, fx.params())
		const want = "org: merge: interrupted (test stopped) after the checks passed, before gh pr merge ran: nothing was merged"
		if err == nil || err.Error() != want || res != (MergeResult{}) {
			t.Fatalf("Merge = %+v, %v\nwant the error %q", res, err, want)
		}
		if got := fx.exec.keys(); !slices.Equal(got, mxAll) {
			t.Errorf("commands = %q\nwant %q", got, mxAll)
		}
		if got := fx.appended(t); got != "" {
			t.Errorf("appended %q", got)
		}
		mrAssertLockFree(t, fx.mo.stateDir)
	})
}

// TestMerge_RefusalsBeforeTheMerge: no branch is refused before any lock,
// with checkMerge's words and no regate.lock made; a check that fails is
// refused with checkMerge's refusal. Neither runs gh pr merge or records
// anything, and regate.lock is released.
func TestMerge_RefusalsBeforeTheMerge(t *testing.T) {
	t.Run("no branch", func(t *testing.T) {
		fx := newMRFixture(t, nil)
		p := fx.params()
		p.Branch = ""
		res, err := fx.merge(context.Background(), t, p)
		want := MergeResult{Refused: "org: merge: no branch named, so nothing was merged: name the feature branch to merge (ralph " +
			"org merge-order lists the ledger's branches)"}
		if err != nil || res != want {
			t.Fatalf("Merge = %+v, %v\nwant %+v", res, err, want)
		}
		if got := fx.exec.keys(); len(got) != 0 {
			t.Errorf("commands = %q, want none", got)
		}
		if got := fileOrAbsent(t, regateLockPathIn(fx.mo.stateDir)); got != "<absent>" {
			t.Errorf("regate.lock holds %q; want no lock taken", got)
		}
	})
	t.Run("a check refuses", func(t *testing.T) {
		fx := newMRFixture(t, nil)
		fx.exec.replies[mxPRList] = mgReply{out: "[]"}
		res, err := fx.merge(context.Background(), t, fx.params())
		want := MergeResult{Refused: "org: merge: gh pr list --head feat/a -R " + mxRepoArg + " found no PR of the branch feat/a in " +
			"origin's repository github.com/o/r, so nothing was merged: open one (gh pr create --head feat/a -R github.com/o/r), and " +
			"once its CI passed, " + mxAgain}
		if err != nil || res != want {
			t.Fatalf("Merge = %+v, %v\nwant %+v", res, err, want)
		}
		if got := fx.exec.keys(); !slices.Equal(got, mxAll[:3]) {
			t.Errorf("commands = %q\nwant %q", got, mxAll[:3])
		}
		if got := fx.appended(t); got != "" {
			t.Errorf("appended %q", got)
		}
		mrAssertLockFree(t, fx.mo.stateDir)
	})
}

// TestMerge_TimeoutsOfTheMergeAndTheCallsAfterIt: gh pr merge runs under
// MergeTimeout, 60 s when it is zero or less; the read-back and the removal
// of a request under GHTimeout, 15 s when it is zero or less.
func TestMerge_TimeoutsOfTheMergeAndTheCallsAfterIt(t *testing.T) {
	for _, tc := range []struct {
		name              string
		merge, gh         time.Duration // MergeParams'
		wantMerge, wantGH time.Duration
	}{
		{"defaults", 0, 0, 60 * time.Second, 15 * time.Second},
		{"less than zero", -time.Second, -time.Second, 60 * time.Second, 15 * time.Second},
		{"set", 5 * time.Second, 3 * time.Second, 5 * time.Second, 3 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMRFixture(t, nil)
			fx.ex.reread = mgReply{out: mrReadBack("OPEN", "", mrRequest)}
			p := fx.params()
			p.MergeTimeout, p.GHTimeout = tc.merge, tc.gh
			start := time.Now()
			if res, err := fx.merge(context.Background(), t, p); err != nil || res.Failed == "" {
				t.Fatalf("Merge = %+v, %v; want the removed request", res, err)
			}
			end := time.Now()
			for key, want := range map[string]time.Duration{mrMerge: tc.wantMerge, mrReread: tc.wantGH, mrDisable: tc.wantGH} {
				if d := fx.call(t, key).deadline; d.Before(start.Add(want)) || d.After(end.Add(want)) {
					t.Errorf("%s: deadline %v is not %s after the start (%v..%v)", key, d, want, start, end)
				}
			}
		})
	}
}

// TestMergeGHSaid: the tail is gh's last 5 lines that are not blank, each
// with its whitespace runs made one space, its control characters escaped,
// and cut at 300 bytes; no error or an empty one gives none.
func TestMergeGHSaid(t *testing.T) {
	long := strings.Repeat("x", 400)
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"empty", errors.New(""), ""},
		{"blank lines only", errors.New("\n \n"), ""},
		{"one line", errors.New("exit status 1"), "; gh said:\n  exit status 1"},
		{"the last five", errors.New("1\n2\n\n3\n4\t\tfour\n5\n6\r\n"), "; gh said:\n  2\n  3\n  4 four\n  5\n  6"},
		{"escaped", errors.New("a\x1b]0;x\a b"), `; gh said:` + "\n  " + `a\x1b]0;x\a b`},
		{"cut", errors.New(long), "; gh said:\n  " + long[:300] + "..."},
	} {
		if got := mergeGHSaid(tc.err); got != tc.want {
			t.Errorf("%s: mergeGHSaid = %q\nwant %q", tc.name, got, tc.want)
		}
	}
}

// TestPRMerged_IsNotASeatAndDoesNotRun covers AC9: a pr_merged that Merge
// wrote leaves alone what the ledger's readers make of it: the running orgs
// and so max_orgs, disband --all's targets, the roster and so stop --all's
// seats, and the watch's leader activity of the org; the org report's event
// timeline lists it.
func TestPRMerged_IsNotASeatAndDoesNotRun(t *testing.T) {
	fx := newMRFixture(t, nil)
	if res, err := fx.merge(context.Background(), t, fx.params()); err != nil || !res.Recorded {
		t.Fatalf("Merge = %+v, %v", res, err)
	}
	rr, err := fx.mo.o.Manifest.Read()
	if err != nil || len(rr.Events) == 0 || rr.Events[len(rr.Events)-1].Event != EventPRMerged {
		t.Fatalf("manifest = %+v, %v; want pr_merged last", rr.Events, err)
	}
	merged := rr.Events[len(rr.Events)-1]
	leaderB := ManifestEvent{TS: "2026-10-14T08:00:00Z", OrgID: "org-b", SeatID: LeaderIdentity, Event: EventSpawned, Role: "leader", PaneID: "p-1"}
	leaderA := ManifestEvent{TS: "2026-10-14T08:00:00Z", OrgID: "org-a", SeatID: LeaderIdentity, Event: EventSpawned, Role: "leader", PaneID: "p-2"}
	for _, tc := range []struct {
		name    string
		without []ManifestEvent
	}{
		{"after the org was disbanded", rr.Events[:len(rr.Events)-1]},
		{"with another org running", append(slices.Clone(rr.Events[:len(rr.Events)-1]), leaderB)},
		{"with the org running again", append(slices.Clone(rr.Events[:len(rr.Events)-1]), leaderA)},
		{"alone", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			with := append(slices.Clone(tc.without), merged)
			if got, want := RunningOrgs(with), RunningOrgs(tc.without); !slices.Equal(got, want) {
				t.Errorf("RunningOrgs = %q, want %q", got, want)
			}
			if got, want := orgsToDisband(with), orgsToDisband(tc.without); !slices.Equal(got, want) {
				t.Errorf("orgsToDisband = %q, want %q", got, want)
			}
			cfg := config.OrgConfig{MaxOrgs: 1}
			if got, want := fmt.Sprint(validateMaxOrgs(cfg, "org-c", RunningOrgs(with))), fmt.Sprint(validateMaxOrgs(cfg, "org-c",
				RunningOrgs(tc.without))); got != want {
				t.Errorf("validateMaxOrgs = %s, want %s", got, want)
			}
			if got, want := leaderActivityEventCount(with, "org-a"), leaderActivityEventCount(tc.without, "org-a"); got != want {
				t.Errorf("leaderActivityEventCount = %d, want %d", got, want)
			}
			for _, opts := range []RosterOptions{{}, {IncludeDryRun: true}} {
				if got, want := Roster(with, opts), Roster(tc.without, opts); !reflect.DeepEqual(got, want) {
					t.Errorf("Roster(%+v) = %+v, want %+v", opts, got, want)
				}
			}
		})
	}
	report := BuildOrgReport(rr.Events, nil, "org-a", "2026-10-14", 0)
	timeline := report[strings.Index(report, "## Event timeline"):]
	row := "| " + mergeOrderTestListedAt + " | - | pr_merged | branch=feat/a pr=7 head=" + moHeadA + " method=merge merge_commit=" +
		mrOID + " auto_merge=false |\n"
	if !strings.Contains(timeline, row) {
		t.Errorf("the report's event timeline has no row %q:\n%s", row, timeline)
	}
	if roster := report[:strings.Index(report, "## Event timeline")]; strings.Contains(roster, EventPRMerged) {
		t.Errorf("the report before its event timeline names pr_merged:\n%s", roster)
	}
}

// TestMerge_SharesRegateLockWithRegate covers AC10 and D3: while a merge
// holds regate.lock, ralph org regate of another branch is refused for the
// lock; while another holds it, Merge is refused after regateLockWait with no
// command run and the holder's marker left as it is; and during the merge a
// spawn of the branch's org goes on, since Merge emptied the marker an
// earlier regate of that org left and writes none.
func TestMerge_SharesRegateLockWithRegate(t *testing.T) {
	t.Run("regate is refused during a merge", func(t *testing.T) {
		fx := newMRFixture(t, nil)
		var regate RegateResult
		var regateErr error
		fx.ex.inMerge = func(context.Context) {
			regate, regateErr = fx.mo.o.Regate(context.Background(), RegateParams{Branch: "feat/other", Env: []string{}})
		}
		if res, err := fx.merge(context.Background(), t, fx.params()); err != nil || !res.Recorded {
			t.Fatalf("Merge = %+v, %v", res, err)
		}
		if regateErr != nil || !strings.HasPrefix(regate.Refused, "org: regate: ") ||
			!strings.Contains(regate.Refused, regateLockPathIn(fx.mo.stateDir)) || !reflect.DeepEqual(regate, RegateResult{Refused: regate.Refused}) {
			t.Fatalf("Regate during the merge = %+v, %v; want a refusal that names regate.lock", regate, regateErr)
		}
	})

	t.Run("a merge is refused while another holds the lock", func(t *testing.T) {
		fx := newMRFixture(t, nil)
		path := regateLockPathIn(fx.mo.stateDir)
		other, err := acquireHeldFileLock(fx.mo.stateDir, regateLockFile, "regate", 0)
		if err != nil {
			t.Fatal(err)
		}
		if err := writeRegateMarker(other.f, &regateMarker{OrgID: "org-x", Branch: "feat/x", Worktree: "/w", PID: 1}); err != nil {
			t.Fatal(err)
		}
		begin := time.Now()
		res, err := fx.merge(context.Background(), t, fx.params())
		if took := time.Since(begin); took < regateLockWait {
			t.Errorf("Merge refused after %s; want it to retry the lock for %s", took, regateLockWait)
		}
		want := MergeResult{Refused: "org: merge: another ralph org regate, merge, or cleanup runs on this ledger (" + path + " is held): one " +
			"runs at a time, so wait for it to end and run ralph org merge --branch feat/a again"}
		if err != nil || res != want {
			t.Fatalf("Merge = %+v, %v\nwant %+v", res, err, want)
		}
		if got := fx.exec.keys(); len(got) != 0 {
			t.Errorf("commands = %q, want none", got)
		}
		if got := fx.appended(t); got != "" {
			t.Errorf("appended %q", got)
		}
		if orgID, ok := readRegateMarker(path); !ok || orgID != "org-x" {
			t.Errorf("readRegateMarker = %q, %v; want the holder's marker, org-x", orgID, ok)
		}
		if err := other.release(); err != nil {
			t.Fatal(err)
		}
		if res, err := fx.merge(context.Background(), t, fx.params()); err != nil || !res.Recorded {
			t.Fatalf("Merge once the lock was released = %+v, %v", res, err)
		}
	})

	t.Run("a spawn of the org goes on during the merge", func(t *testing.T) {
		fx := newMRFixture(t, nil)
		path := regateLockPathIn(fx.mo.stateDir)
		const marker = `{"org_id":"org-a","branch":"feat/a","worktree":"/w","pid":1}` + "\n"
		// The control: with the lock held and that marker in it, a spawn of
		// org-a is refused.
		holder, err := acquireHeldFileLock(fx.mo.stateDir, regateLockFile, "regate", 0)
		if err != nil {
			t.Fatal(err)
		}
		writeGateFile(t, fx.mo.stateDir, regateLockFile, marker, 0o644)
		if err := regateSpawnErr(fx.mo.stateDir, "org-a"); err == nil {
			t.Fatal("regateSpawnErr = nil with org-a's marker held; the control does not refuse")
		}
		if err := holder.release(); err != nil {
			t.Fatal(err)
		}
		var held bool
		var raw string
		var spawnErr error
		fx.ex.inMerge = func(context.Context) {
			held, _ = fileLockHeld(path)
			raw = fileOrAbsent(t, path)
			spawnErr = regateSpawnErr(fx.mo.stateDir, "org-a")
		}
		if res, err := fx.merge(context.Background(), t, fx.params()); err != nil || !res.Recorded || res.OrgID != "org-a" {
			t.Fatalf("Merge = %+v, %v", res, err)
		}
		if !held || raw != "" || spawnErr != nil {
			t.Fatalf("during the merge: lock held %v, regate.lock holds %q, regateSpawnErr = %v; want held, empty, nil", held, raw, spawnErr)
		}
		if got := fileOrAbsent(t, path); got != "" {
			t.Errorf("regate.lock holds %q after the merge; want no marker", got)
		}
		mrAssertLockFree(t, fx.mo.stateDir)
	})
}

// TestMerge_CallsAfterGHPRMergeRunDetached covers self-review L3 (S6 of the
// plan): gh pr merge runs through the runner of the checks, in ralph's
// process group, where an interruption stops it; the read-back and the
// removal of an auto-merge request run through runDetached, which
// newMergeCheck sets to realMergeDetachedExec for the real commands (no
// Exec) and to Exec otherwise.
func TestMerge_CallsAfterGHPRMergeRunDetached(t *testing.T) {
	m := newMergeCheck(MergeParams{})
	if reflect.ValueOf(m.run).Pointer() != reflect.ValueOf(realMergeOrderExec).Pointer() ||
		reflect.ValueOf(m.runDetached).Pointer() != reflect.ValueOf(realMergeDetachedExec).Pointer() {
		t.Error("newMergeCheck with no Exec: the checks and gh pr merge do not run through realMergeOrderExec, or the calls " +
			"after gh pr merge not through realMergeDetachedExec")
	}

	fx := newMRFixture(t, nil)
	r, refused, _, err := fx.mo.o.checkMerge(context.Background(), fx.params())
	if err != nil || refused != "" {
		t.Fatalf("checkMerge = %v, refused %q", err, refused)
	}
	detached := &mrExec{mxExec: &mxExec{}, reread: mgReply{out: mrReadBack("OPEN", "", mrRequest)}}
	run := mergeRun{m: newMergeCheck(fx.params()), r: r}
	run.m.runDetached = detached.run
	res, err := fx.mo.o.mergeAndRecord(context.Background(), run, MergeResult{})
	if err != nil || !strings.Contains(res.Failed, "which ralph org merge removed ("+mrDisableCmd+")") {
		t.Fatalf("mergeAndRecord = %+v, %v; want the request removed", res, err)
	}
	if got := fx.exec.keys()[len(mxAll):]; !slices.Equal(got, []string{mrMerge}) {
		t.Errorf("through the runner of the checks once they passed: %q; want gh pr merge only", got)
	}
	if got := detached.keys(); !slices.Equal(got, []string{mrReread, mrDisable}) {
		t.Errorf("through runDetached: %q; want the read-back and the removal", got)
	}
}

// TestRealMergeDetachedExec_RunsInAProcessGroupOfItsOwn covers self-review
// L3: realMergeDetachedExec starts its command in a process group of its
// own, led by the command, so a Ctrl-C or hangup the terminal sends the
// test's (ralph's) foreground group does not reach it; realMergeOrderExec,
// which runs gh pr merge, the checks, and merge-order, leaves its command in
// the test's group. sh prints its pid and, with ps, its process group.
func TestRealMergeDetachedExec_RunsInAProcessGroupOfItsOwn(t *testing.T) {
	test := syscall.Getpgrp()
	for _, tc := range []struct {
		name     string
		run      MergeOrderExec
		ownGroup bool
	}{{"realMergeDetachedExec", realMergeDetachedExec, true}, {"realMergeOrderExec", realMergeOrderExec, false}} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := tc.run(context.Background(), t.TempDir(), nil, "sh", "-c", `echo "$$ $(ps -o pgid= -p $$)"`)
			var pid, pgid int
			if _, scanErr := fmt.Sscan(out, &pid, &pgid); err != nil || scanErr != nil {
				t.Fatalf("sh printed %q, %v (%v)", out, err, scanErr)
			}
			want := test
			if tc.ownGroup {
				want = pid
			}
			if pgid != want {
				t.Errorf("sh (pid %d) ran in the process group %d; want %d (the test's is %d)", pid, pgid, want, test)
			}
		})
	}
}

// TestRealMergeDetachedExec_CtxDoneKillsTheGroup: ctx done kills the whole
// process group of realMergeDetachedExec's command, so a child the command
// started ends with it, as the terminal's interrupt would have ended it in
// ralph's group. The child would write its file 500 ms after it started.
func TestRealMergeDetachedExec_CtxDoneKillsTheGroup(t *testing.T) {
	alive := filepath.Join(t.TempDir(), "alive")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := realMergeDetachedExec(ctx, t.TempDir(), nil, "sh", "-c", `(sleep 0.5; echo alive > "$1") & wait`, "sh",
		alive); err == nil {
		t.Fatal("realMergeDetachedExec = nil error once ctx is done")
	}
	time.Sleep(time.Second)
	if _, err := os.Stat(alive); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the command's child still ran after ctx was done (%s: %v); want the whole group killed", alive, err)
	}
}

// The cleanup after a recorded merge (plan
// docs/plans/active/2026-10-28-org-merge-cleanup.md, S2: AC1, AC3, and AC4 for
// ralph org merge at the org layer).

// mcSpawnRefusal is regateSpawnErr's refusal of a spawn of orgID while the
// ledger in stateDir holds orgID's marker in a held regate.lock.
func mcSpawnRefusal(stateDir, orgID string) string {
	return `org: org_id "` + orgID + `" is not spawned while ralph org regate runs on its branch or ralph org merge or cleanup ` +
		`removes its merged feature worktree (` + regateLockPathIn(stateDir) + ` holds the marker of that org): wait for it to end ` +
		`and run the same command again`
}

// TestMerge_CleansUpTheFeatureWorktreeOnceRecorded covers AC1 through the
// real scripts/ralph-worktree.sh: once pr_merged is recorded, Merge cleans up
// feat/auth's feature worktree with regate.lock still held and the org's
// marker in it, so a spawn of that org is refused meanwhile and one of
// another org is not; the worktree, its registration, the branch, and the
// record go with one run of the script's cleanup, worktree_cleaned with
// via=merge follows pr_merged in the ledger, the cleanup runs nothing through
// MergeOrderExec, and Merge returns the merge with no error.
func TestMerge_CleansUpTheFeatureWorktreeOnceRecorded(t *testing.T) {
	fx := newMCFixture(t)
	path := regateLockPathIn(fx.stateDir)
	var held, reached bool
	var marked string
	var sameErr, otherErr error
	cleanupTestHooks.beforeScript = func() {
		reached = true
		held, _ = fileLockHeld(path)
		marked, _ = readRegateMarker(path)
		sameErr, otherErr = regateSpawnErr(fx.stateDir, "auth"), regateSpawnErr(fx.stateDir, "other")
	}
	t.Cleanup(func() { cleanupTestHooks.beforeScript = nil })

	res, err := fx.merge(t)
	if err != nil || !res.Merged || !res.Recorded || res.Failed != "" || res.Refused != "" || res.Cleanup == nil ||
		res.CleanupErr != "" || !reached {
		t.Fatalf("Merge = %+v, %v (hook reached %v); want a recorded merge cleaned up through the hook", res, err, reached)
	}
	fx.assertCleaned(t, *res.Cleanup, nil, "", CleanupViaMerge)
	events := mustReadEvents(t, fx.o)
	if prev := events[len(events)-2]; prev.Event != EventPRMerged || prev.OrgID != "auth" ||
		!strings.HasPrefix(prev.Details, "branch=feat/auth pr=7 head="+fx.head+" ") {
		t.Errorf("the event before worktree_cleaned is %+v; want feat/auth's pr_merged", prev)
	}
	if want := mcSpawnRefusal(fx.stateDir, "auth"); !held || marked != "auth" || sameErr == nil || sameErr.Error() != want ||
		otherErr != nil {
		t.Errorf("during the cleanup: lock held %v, marker %q, regateSpawnErr = %v for auth, %v for other; want held, auth, %q, "+
			"and nil", held, marked, sameErr, otherErr, want)
	}
	if err := regateSpawnErr(fx.stateDir, "auth"); err != nil {
		t.Errorf("regateSpawnErr after the merge = %v; want nil", err)
	}
}

// TestMerge_RefusedCleanupKeepsTheMerge covers AC3 and AC4 for ralph org
// merge, through the real script: a cleanup that refuses, for an untracked
// file in the worktree or for the branch's org spawned again while gh pr
// merge ran (no marker kept that spawn out), removes nothing and records no
// worktree_cleaned, and Merge still returns the recorded merge with no
// error and the refusal in Cleanup. Neither names a command by hand: the
// next cleanup goes on from each once its cause is gone.
func TestMerge_RefusedCleanupKeepsTheMerge(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, fx *mcFixture)
		want  string
		calls []string // the worktree calls
	}{
		{"an untracked file", func(t *testing.T, fx *mcFixture) {
			writeGateFile(t, fx.worktree, "stray.txt", "x\n", 0o644)
		}, "has 1 uncommitted change(s) or untracked file(s) (?? stray.txt), which removing it would lose",
			[]string{"lookup:" + cleanupTestID}},
		{"the org spawned again during gh pr merge", func(t *testing.T, fx *mcFixture) {
			fx.ex.inMerge = func(context.Context) { seedSeat(t, fx.o, "auth", "worker", "worker", "pane-2") }
		}, `the branch feat/auth is bound to org_id "auth", which is running and may still use its feature worktree ` +
			"%WORKTREE%: wait for the org to be disbanded and run ralph org cleanup --branch feat/auth again", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMCFixture(t)
			tc.setup(t, fx)
			record := fileOrAbsent(t, fx.recordFile(t))
			res, err := fx.merge(t)
			if err != nil || !res.Merged || !res.Recorded || res.Failed != "" || res.Cleanup == nil || res.CleanupErr != "" ||
				res.Cleanup.Cleaned || res.Cleanup.Recorded || res.Cleanup.Failed != "" {
				t.Fatalf("Merge = %+v, %v (cleanup %+v); want a recorded merge whose cleanup refused", res, err, res.Cleanup)
			}
			assertCleanupRefusal(t, *res.Cleanup, strings.ReplaceAll(tc.want, "%WORKTREE%", fx.worktree), "")
			if !slices.Equal(fx.spy.calls, tc.calls) {
				t.Errorf("worktree calls = %q, want %q", fx.spy.calls, tc.calls)
			}
			if info, err := os.Stat(fx.worktree); err != nil || !info.IsDir() {
				t.Errorf("the worktree %s is gone (%v)", fx.worktree, err)
			}
			if tip := fx.branchTip(); tip != fx.head {
				t.Errorf("the branch %s is at %q, want %s", cleanupTestBranch, tip, fx.head)
			}
			if now := fileOrAbsent(t, fx.recordFile(t)); now != record || now == "<absent>" {
				t.Errorf("the record changed:\n%s\n->\n%s", record, now)
			}
			events := mustReadEvents(t, fx.o)
			if last := events[len(events)-1]; last.Event != EventPRMerged || len(cleanedEvents(events)) != 0 {
				t.Errorf("manifest ends with %+v (%d worktree_cleaned); want pr_merged and no worktree_cleaned", last,
					len(cleanedEvents(events)))
			}
			assertCleanupUnlocked(t, fx.stateDir)
		})
	}
}

// mcFakeFixture is newMRFixture's ledger, whose binding of feat/a to org-a
// names the worktree /repo/.claude/worktrees/org-a, with fakeWorktrees
// holding that binding's record.
func mcFakeFixture(t *testing.T) (*mrFixture, *fakeWorktrees) {
	t.Helper()
	fx := newMRFixture(t, nil)
	wt := newFakeWorktrees()
	wt.records[featureWorktreeID("org-a")] = WorktreeRecord{WorktreePath: "/repo/.claude/worktrees/org-a", Branch: "feat/a",
		Kind: featureWorktreeKind}
	fx.mo.o.Worktrees = wt
	return fx, wt
}

// TestMerge_RunsTheCleanupOnlyAfterARecordedMerge covers D3's opt-in and the
// merges that are not followed by a cleanup, with fakeWorktrees: with no
// RepoRoot a recorded merge runs none and leaves regate.lock with no marker,
// as before; with RepoRoot, a merge that is refused, is not MERGED, or whose
// pr_merged cannot be appended runs none either. Each has Cleanup nil and no
// worktree call. An interruption once gh pr merge started, with the merge
// recorded, runs none and says so in Cleanup's Refused.
func TestMerge_RunsTheCleanupOnlyAfterARecordedMerge(t *testing.T) {
	recorded := mrResult("merge")
	recorded.Merged, recorded.Recorded, recorded.MergeCommit = true, true, mrOID
	t.Run("no repo root", func(t *testing.T) {
		fx, wt := mcFakeFixture(t)
		res, err := fx.merge(context.Background(), t, fx.params())
		if err != nil || res != recorded {
			t.Fatalf("Merge = %+v, %v\nwant    %+v", res, err, recorded)
		}
		if len(wt.calls) != 0 {
			t.Errorf("worktree calls = %q, want none", wt.calls)
		}
		if got := fileOrAbsent(t, regateLockPathIn(fx.mo.stateDir)); got != "" {
			t.Errorf("regate.lock holds %q after the merge; want no marker", got)
		}
	})
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, fx *mrFixture)
		check func(res MergeResult, err error) bool
	}{
		{"a refused merge", func(_ *testing.T, fx *mrFixture) { fx.exec.replies[mxPRList] = mgReply{out: "[]"} },
			func(res MergeResult, err error) bool { return err == nil && res.Refused != "" }},
		{"a merge that is not MERGED", func(_ *testing.T, fx *mrFixture) {
			fx.ex.reread = mgReply{out: mrReadBack("OPEN", "", "null")}
		}, func(res MergeResult, err error) bool { return err == nil && res.Failed != "" && !res.Merged }},
		{"a pr_merged that cannot be appended", func(t *testing.T, fx *mrFixture) {
			if os.Geteuid() == 0 {
				t.Skip("root can append to a read-only manifest")
			}
			if err := os.Chmod(fx.mo.path, 0o444); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(fx.mo.path, 0o644) })
		}, func(res MergeResult, err error) bool { return err != nil && res.Merged && !res.Recorded }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx, wt := mcFakeFixture(t)
			tc.setup(t, fx)
			p := fx.params()
			p.RepoRoot = "/repo"
			res, err := fx.merge(context.Background(), t, p)
			if !tc.check(res, err) || res.Cleanup != nil || res.CleanupErr != "" {
				t.Fatalf("Merge = %+v, %v; want no cleanup", res, err)
			}
			if len(wt.calls) != 0 {
				t.Errorf("worktree calls = %q, want none", wt.calls)
			}
		})
	}
	t.Run("interrupted once gh pr merge started", func(t *testing.T) {
		fx, wt := mcFakeFixture(t)
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		fx.ex.inMerge = func(context.Context) { cancel(errors.New("test stopped")) }
		p := fx.params()
		p.RepoRoot = "/repo"
		res, err := fx.merge(ctx, t, p)
		want := "org: merge: interrupted (test stopped) once gh pr merge had started, and then: PR #7 " +
			"(https://github.com/o/r/pull/7) was merged (merge commit " + mrOID + ") and pr_merged was recorded"
		cleanup := CleanupResult{Branch: "feat/a", Head: moHeadA, Refused: "ralph org merge was interrupted (test stopped) once " +
			"pr_merged was recorded, so the feature worktree of the branch feat/a was not cleaned up: run ralph org cleanup --branch feat/a"}
		if err == nil || err.Error() != want || !res.Recorded || res.Cleanup == nil || *res.Cleanup != cleanup || res.CleanupErr != "" {
			t.Fatalf("Merge = %+v, %v (cleanup %+v)\nwant the error %q and the cleanup %+v", res, err, res.Cleanup, want, cleanup)
		}
		if len(wt.calls) != 0 {
			t.Errorf("worktree calls = %q, want none", wt.calls)
		}
	})
}

// TestMerge_CleanupOutcomesWithFakeWorktrees covers the cleanup's outcomes
// that reach MergeResult, with fakeWorktrees, the merge recorded and no error
// from Merge in each: an earlier cleanup left the record alone (no worktree,
// no branch in a repo root that git reads), which goes on and records
// worktree_cleaned (Resumed record_only); a repo root git cannot run in,
// whose error is CleanupErr with nothing removed or recorded; and a record on
// another branch, a refusal whose Manual deletes only the binding's branch.
func TestMerge_CleanupOutcomesWithFakeWorktrees(t *testing.T) {
	isolateGitEnv(t)
	gitRoot := resolvedOrClean(t.TempDir())
	runGit(t, "init", "--quiet", "-b", "main", gitRoot)
	missing := filepath.Join(resolvedOrClean(t.TempDir()), "missing")
	id := featureWorktreeID("org-a")
	for _, tc := range []struct {
		name    string
		root    string
		setup   func(wt *fakeWorktrees)
		check   func(t *testing.T, c CleanupResult, cleanupErr string)
		calls   []string
		cleaned string // the Details of the worktree_cleaned appended after pr_merged; "" for none
	}{
		{"the record alone is left", gitRoot, func(*fakeWorktrees) {}, func(t *testing.T, c CleanupResult, cleanupErr string) {
			if !c.Cleaned || !c.Recorded || c.Resumed != CleanupResumedRecordOnly || c.Refused != "" || c.Failed != "" || cleanupErr != "" {
				t.Errorf("cleanup = %+v, %q; want it cleaned, resumed from the record alone", c, cleanupErr)
			}
		}, []string{"lookup:" + id, "cleanup:" + id},
			"branch=feat/a head=" + moHeadA + ` via=merge worktree="/repo/.claude/worktrees/org-a"`},
		{"a repo root git cannot run in", missing, func(*fakeWorktrees) {}, func(t *testing.T, c CleanupResult, cleanupErr string) {
			if c.Cleaned || c.Refused != "" || c.Failed != "" ||
				!strings.HasPrefix(cleanupErr, "org: cleanup: git worktree list --porcelain in "+missing+": exit ") {
				t.Errorf("cleanup = %+v, %q; want the error of git worktree list in %s", c, cleanupErr, missing)
			}
		}, []string{"lookup:" + id}, ""},
		{"a record on another branch", gitRoot, func(wt *fakeWorktrees) {
			rec := wt.records[id]
			rec.Branch = "feat/a-2"
			wt.records[id] = rec
		}, func(t *testing.T, c CleanupResult, cleanupErr string) {
			if c.Cleaned || cleanupErr != "" {
				t.Errorf("cleanup = %+v, %q; want a refusal", c, cleanupErr)
			}
			assertCleanupRefusal(t, c, `the worktree record org-org-a records branch "feat/a-2", not "feat/a"`,
				"git -C "+cleanupShellWord(gitRoot)+" branch -D feat/a")
		}, []string{"lookup:" + id}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx, wt := mcFakeFixture(t)
			tc.setup(wt)
			p := fx.params()
			p.RepoRoot = tc.root
			res, err := fx.merge(context.Background(), t, p)
			if err != nil || !res.Merged || !res.Recorded || res.Failed != "" || res.Cleanup == nil {
				t.Fatalf("Merge = %+v, %v; want a recorded merge with its cleanup", res, err)
			}
			tc.check(t, *res.Cleanup, res.CleanupErr)
			if !slices.Equal(wt.calls, tc.calls) {
				t.Errorf("worktree calls = %q, want %q", wt.calls, tc.calls)
			}
			want := mrRecord(t, "branch=feat/a pr=7 head="+moHeadA+" method=merge merge_commit="+mrOID+" auto_merge=false")
			if tc.cleaned != "" {
				want += mcCleanedRecord(t, tc.cleaned)
			}
			if got := fx.appended(t); got != want {
				t.Errorf("appended %q\nwant     %q", got, want)
			}
			mrAssertLockFree(t, fx.mo.stateDir)
		})
	}
}

// mcCleanedRecord is the manifest line of org-a's worktree_cleaned with
// details, at mrFixture's clock.
func mcCleanedRecord(t *testing.T, details string) string {
	t.Helper()
	b, err := json.Marshal(ManifestEvent{TS: mergeOrderTestListedAt, OrgID: "org-a", Event: EventWorktreeCleaned, Details: details})
	if err != nil {
		t.Fatal(err)
	}
	return string(b) + "\n"
}
