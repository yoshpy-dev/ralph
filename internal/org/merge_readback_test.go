package org

import (
	"context"
	"errors"
	"slices"
	"testing"
)

// The base the read-back of `ralph org merge` gives (plan
// docs/plans/active/2026-10-17-org-auto-merge.md, S10: D6, AC7 and AC8 for
// cross-review cycle 2 #1): Merge on the manifest fixtures of
// merge_fake_test.go, with mrExec answering the calls Merge adds after
// checkMerge's, so no test runs gh or reaches a network.

// TestMerge_RecordsOnlyAMergeIntoTheDefaultBranch covers D6, AC7 and AC8 for
// the base the read-back gives (cross-review cycle 2 #1 of the plan): a PR
// read back as MERGED at the gate's head into another base than main, the
// fixture's default branch, or with no base (empty or left out), as when
// someone retargeted the PR after the checks, after gh pr merge exited 0 or
// failed, is Failed with Merged not set, so no MERGED: line, and nothing
// recorded; the Failed names the base, how gh pr merge ended with gh's tail,
// and the next steps, which say that merge-order lists the branch as merged
// though main got nothing from this merge (S11). When the head is not the
// gate's either, the Failed is the head's (otherHeadMerged), base or no base.
// MERGED at the gate's head into main is recorded.
func TestMerge_RecordsOnlyAMergeIntoTheDefaultBranch(t *testing.T) {
	failure := errors.New("exit status 1: HTTP 502: Bad Gateway")
	const said = "; gh said:\n  exit status 1: HTTP 502: Bad Gateway"
	otherBase := "into the base release, not main, the default branch the checks read, so main got nothing from it as far as " +
		"ralph knows"
	noBase := "with no base, so whether it merged into main is unknown"
	otherHead := "at the head bbbbbbbbbbbb, not aaaaaaaaaaaa, the head the gate passed, so a head the gate did not pass was merged"
	for _, tc := range []struct {
		name    string
		merge   mgReply
		reread  string
		failed  string // "" for a recorded merge
		records bool
	}{
		{"gh pr merge exited 0, merged into another base", mgReply{}, mrReadBackInto("MERGED", moHeadA, "release", mrOID, "null"),
			mrMergedAfter("exited 0") + otherBase + mrNextBase, false},
		{"gh pr merge failed, merged into another base", mgReply{err: failure}, mrReadBackInto("MERGED", moHeadA, "release", mrOID,
			"null"), mrMergedAfter("failed") + otherBase + mrNextBase + said, false},
		{"MERGED with an empty base", mgReply{}, mrReadBackInto("MERGED", moHeadA, "", mrOID, "null"),
			mrMergedAfter("exited 0") + noBase + mrNextNoBase, false},
		{"MERGED with no baseRefName", mgReply{err: failure}, `{"autoMergeRequest":null,"headRefOid":"` + moHeadA +
			`","mergeCommit":{"oid":"` + mrOID + `"},"state":"MERGED"}`, mrMergedAfter("failed") + noBase + mrNextNoBase + said, false},
		{"a base with control characters is escaped", mgReply{}, mrReadBackInto("MERGED", moHeadA, "rel\x1b[2Jease", mrOID, "null"),
			mrMergedAfter("exited 0") + `into the base rel\x1b[2Jease, not main, the default branch the checks read, so main got ` +
				"nothing from it as far as ralph knows" + mrNextBase, false},
		{"another head into another base names the head", mgReply{}, mrReadBackInto("MERGED", moHeadB, "release", mrOID, "null"),
			mrMergedAfter("exited 0") + otherHead + mrNext, false},
		{"another head with no base names the head", mgReply{err: failure}, mrReadBackInto("MERGED", moHeadB, "", mrOID, "null"),
			mrMergedAfter("failed") + otherHead + mrNext + said, false},
		{"MERGED at the gate's head into main", mgReply{}, mrReadBack("MERGED", mrOID, "null"), "", true},
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

// TestMerge_InterruptedThenMergedIntoAnotherBase covers the interruption
// half of AC7 for the base: ctx done while gh pr merge runs, and the
// read-back, which still runs once, finds the PR MERGED at the gate's head
// into another base. Merge returns the error that says it was interrupted,
// then the Failed of that merge, with Merged and Recorded not set and
// nothing recorded.
func TestMerge_InterruptedThenMergedIntoAnotherBase(t *testing.T) {
	fx := newMRFixture(t, nil)
	fx.ex.reread = mgReply{out: mrReadBackInto("MERGED", moHeadA, "release", mrOID, "null")}
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	fx.ex.inMerge = func(context.Context) { cancel(errors.New("test stopped")) }
	res, err := fx.merge(ctx, t, fx.params())
	want := "org: merge: interrupted (test stopped) once gh pr merge had started, and then: " + mrViewCmd + " read PR #7 back as " +
		"MERGED after " + mrMergeCmd + " failed, but into the base release, not main, the default branch the checks read, so main " +
		"got nothing from it as far as ralph knows" + mrNextBase + "; gh said:\n  context canceled"
	if err == nil || err.Error() != want || res.Merged || res.Recorded {
		t.Fatalf("Merge = %+v, %v\nwant the error %q", res, err, want)
	}
	if got := fx.exec.keys(); !slices.Equal(got, mrMerged) {
		t.Errorf("commands = %q\nwant %q", got, mrMerged)
	}
	if got := fx.appended(t); got != "" {
		t.Errorf("appended %q; want nothing recorded", got)
	}
	mrAssertLockFree(t, fx.mo.stateDir)
}
