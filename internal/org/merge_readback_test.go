package org

import (
	"context"
	"errors"
	"slices"
	"testing"
)

// The base the read-back of `ralph org merge` gives (plan
// docs/plans/active/2026-10-17-org-auto-merge.md, S10: D6, AC7 and AC8 for
// cross-review cycle 2 #1), and the default branch it is compared with when
// that is not main: Merge on the manifest fixtures of merge_fake_test.go,
// with mrExec answering the calls Merge adds after checkMerge's. Then how
// `ralph org merge-order` lists a PR merged into another base, which the
// Failed of such a merge names: MergeOrder on the fixtures of
// mergeorder_test.go. No test runs gh or reaches a network.

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

// TestMerge_ComparesTheBaseWithOriginsDefaultBranch covers D6, AC7 and AC8
// for the branch the read-back's base is compared with: origin's default
// branch as the checks read it, not main. With trunk as origin's default
// branch (ls-remote's HEAD, the gate's base, the PR's base, gh's repository
// and its rules), a PR read back as MERGED at the gate's head into trunk is
// recorded; one into main, or with no base, is Failed with Merged not set
// and nothing recorded, and the Failed names trunk as the default branch
// that got nothing, or may have got nothing, from that merge.
func TestMerge_ComparesTheBaseWithOriginsDefaultBranch(t *testing.T) {
	const trunkRules = "gh api repos/o/r/rules/branches/trunk"
	commands := slices.Clone(mrMerged)
	commands[slices.Index(commands, mxRules)] = trunkRules
	for _, tc := range []struct {
		name, base string
		failed     string // "" for a recorded merge
	}{
		{"MERGED into trunk, the default branch", "trunk", ""},
		{"MERGED into main, not the default branch", "main", mrMergedAfter("exited 0") + "into the base main, not trunk, the " +
			"default branch the checks read, so trunk got nothing from it as far as ralph knows" + mrCheckIt + " (it lists the " +
			"branch as merged, though trunk got nothing from this merge), and go on with the other branches"},
		{"MERGED with no base", "", mrMergedAfter("exited 0") + "with no base, so whether it merged into trunk is unknown" +
			mrCheckIt + " (it lists the branch as merged, though trunk may have got nothing from this merge), and go on with " +
			"the other branches"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMRFixture(t, []any{moBind("org-a", "a"), moGate("org-a", "feat/a", moHeadA, moMain,
				"refs/remotes/origin/trunk", ""), moDisbanded("org-a")})
			fx.exec.ls = []mgReply{{out: moListing("trunk", "trunk", moMain, "feat/a", moHeadA)}}
			fx.exec.replies[mxPRList] = mgReply{out: moPRs(moPR(7, "OPEN", "trunk"))}
			fx.exec.replies[mxPRView] = mgReply{out: mxPRViewJSON("OPEN", moHeadA, "trunk", "https://github.com/o/r/pull/7", "null")}
			fx.exec.replies[mxRepoAPI] = mgReply{out: meRepoJSON("trunk", "o/r", nil, false)}
			fx.exec.replies[trunkRules] = mgReply{out: mgRules(meChecksRule(1, true, "verify"))}
			fx.exec.replies[mxRuleset] = mgReply{out: meRuleset(1, "protect trunk", "never")}
			fx.ex.reread = mgReply{out: mrReadBackInto("MERGED", moHeadA, tc.base, mrOID, "null")}
			res, err := fx.merge(context.Background(), t, fx.params())
			want := mrResult("merge")
			want.Protection = `the ruleset "protect trunk" (id 1) of trunk, the default branch of origin's repository o/r, ` +
				"requires branches to be up to date before merging with its 1 required status check, and you cannot bypass it " +
				"(current_user_can_bypass: never)"
			want.Failed = tc.failed
			if tc.failed == "" {
				want.Merged, want.Recorded, want.MergeCommit = true, true, mrOID
			}
			if err != nil || res != want {
				t.Fatalf("Merge = %+v, %v\nFailed %q\nwant   %q", res, err, res.Failed, tc.failed)
			}
			if got := fx.exec.keys(); !slices.Equal(got, commands) {
				t.Errorf("commands = %q\nwant %q", got, commands)
			}
			wantAppended := ""
			if tc.failed == "" {
				wantAppended = mrRecord(t, "branch=feat/a pr=7 head="+moHeadA+" method=merge merge_commit="+mrOID+" auto_merge=false")
			}
			if got := fx.appended(t); got != wantAppended {
				t.Errorf("appended %q\nwant     %q", got, wantAppended)
			}
			mrAssertLockFree(t, fx.mo.stateDir)
		})
	}
}

// TestMergeOrder_ListsAMergeIntoAnotherBaseAsMerged pins the claim in
// otherBaseMerged's Failed that ralph org merge-order lists the branch as
// merged though the default branch got nothing, or may have got nothing,
// from that merge: merge-order classifies a branch whose PR gh reads as
// MERGED into another base than main, or with no base, as merged, hidden
// and counted without All and listed with it. That is the tech-debt row on
// merge-order classifying a MERGED PR without looking at its base
// (docs/tech-debt/README.md), so whoever changes that classification changes
// the Failed's text too.
func TestMergeOrder_ListsAMergeIntoAnotherBaseAsMerged(t *testing.T) {
	for _, tc := range []struct{ name, base string }{
		{"MERGED into release", "release"},
		{"MERGED with no base", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMOFixture(t)
			fx.add(t, moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain))
			fx.exec.lsOut = moListing("main", "main", moMain, "feat/a", moHeadA)
			fx.exec.setPR("feat/a", moPRs(moPR(7, "MERGED", tc.base)))
			res := fx.run(t, false)
			if len(res.Entries) != 0 || moNoteWith(res, "1 branch(es) whose PR is merged or closed are not shown") == "" {
				t.Fatalf("entries %v, notes %q; want feat/a hidden and counted as merged or closed", moOrder(res), res.Notes)
			}
			all := fx.run(t, true)
			if got, want := moOrder(all), []string{"1 merged feat/a"}; !slices.Equal(got, want) {
				t.Fatalf("order with All = %q, want %q", got, want)
			}
			if e := moEntry(t, all, "feat/a"); e.PR == nil || e.PR.Number != 7 || e.Detail != "PR #7 is merged" {
				t.Errorf("feat/a = %+v, want PR #7 merged", e)
			}
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
