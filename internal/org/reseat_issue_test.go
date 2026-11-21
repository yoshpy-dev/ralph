package org

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// `ralph org reseat` of a feature org started for an issue (plan
// 2026-11-22-org-start-issue, AC4): the leader reseat starts again gets the
// issue of the binding, and the start reseat prints for an org with no seat
// names it. The featureStart fixture lives in feature_test.go and
// startAuthCoreFor in feature_issue_test.go.

// newIssueReseatOrg is newReseatOrg for auth-core started for issue: its
// implementation plan approved, and its implementer (claude opus) and
// reviewer (claude, the CLI's default model) spawned.
func newIssueReseatOrg(t *testing.T, issue string) *featureStart {
	t.Helper()
	st := newFeatureStart(t)
	st.startAuthCoreFor(t, issue)
	approveFeaturePlan(t, st.o, "auth-core", st.worktree("auth-core"))
	st.spawnSeat(t, implementerSeatID, implementerRole, "opus")
	st.spawnSeat(t, "reviewer", "reviewer", "")
	st.h.closeRemoves = true
	return st
}

// TestReseat_IssueLineInTheLeadersTask covers AC4: the plan of a reseat of
// the leader of an org started for an issue starts the leader with that
// issue, and the reseat starts it with the task of the first start, the
// issue line included, before the progress section, recording no second
// scope_reserved, so the org keeps its binding and its issue. A reseat of
// the other seats alone, which checks the binding the split plan gives now
// against the org's, passes too.
func TestReseat_IssueLineInTheLeadersTask(t *testing.T) {
	st := newIssueReseatOrg(t, "365")
	first := ActiveFeature(mustReadEvents(t, st.o), "auth-core")
	fresh := st.freshIssueLeaderTask(t, "365")
	if !strings.Contains(fresh, "\n"+issueTaskLine("365")+"\n") {
		t.Fatalf("featureLeaderTask for issue 365 does not hold its issue line:\n%s", fresh)
	}

	plan := st.mustPlanReseat(t, st.reseatParams(LeaderIdentity))
	if leader := plan.Start[0]; !leader.Leader || leader.Start == nil || leader.Start.Issue != "365" {
		t.Fatalf("the reseat's leader start = %+v, want the leader started with issue 365", leader.Start)
	}
	st.mustPlanReseat(t, st.reseatParams(implementerSeatID, "reviewer"))

	st.mustReseat(t, st.reseatParams(LeaderIdentity))
	if got := st.leaderTaskOf(t); !strings.HasPrefix(got, fresh+"\n\n") {
		t.Errorf("the reseated leader's task =\n%s\nwant featureLeaderTask's with the issue\n%s\nand the progress section", got, fresh)
	}
	if n := countString(eventNames(t, st.o), EventScopeReserved); n != 1 {
		t.Fatalf("expected one scope_reserved, got %d", n)
	}
	assertActiveFeature(t, st.o, "auth-core", first)
}

// TestReseat_NoIssueLeavesTheLeadersTaskWithoutIt covers AC2 for a reseat:
// the leader of an org started with no issue starts again with no issue
// line.
func TestReseat_NoIssueLeavesTheLeadersTaskWithoutIt(t *testing.T) {
	st := newIssueReseatOrg(t, "")
	st.mustReseat(t, st.reseatParams(LeaderIdentity))
	if got, fresh := st.leaderTaskOf(t), st.freshIssueLeaderTask(t, ""); !strings.HasPrefix(got, fresh+"\n\n") || strings.Contains(got, featureIssueLead) {
		t.Errorf("the reseated leader's task =\n%s\nwant featureLeaderTask's with no issue\n%s\nand the progress section", got, fresh)
	}
}

// TestPlanReseat_StartHintNamesTheIssue: the refusal of a reseat of an org
// with no seat spawned since its last disband names the start of its
// leader with the --issue of its binding, which such a start must name, and
// without --issue for a binding with none.
func TestPlanReseat_StartHintNamesTheIssue(t *testing.T) {
	for _, tc := range []struct{ issue, tail string }{
		{"12", " --issue 12"},
		{"", ""},
	} {
		st := newFeatureStart(t)
		f := &FeatureBinding{Split: "auth-split", Feature: "auth-core", Digest: planDigestOf(t, st.plan), Branch: "fix/auth-core",
			Worktree: st.worktree("auth-core"), Approval: config.ApprovalModePerPlan, FeatureDigest: featureDigestOf(t, st.plan, "auth-core"),
			Issue: tc.issue}
		if err := st.o.Manifest.Append(boundReserveEvent("auth-core", f, "docs/auth.md", "internal/auth/")); err != nil {
			t.Fatal(err)
		}
		start := "no seat has been spawned since then: start its leader with ralph org start --plan " +
			ShellWord(filepath.Join(SplitPlansDirIn(st.stateDir), "auth-split.md")) + " --feature auth-core --org-id auth-core" + tc.tail
		if err := st.checkReseatRefused(t, st.reseatParams(), start); !strings.HasSuffix(err.Error(), start) {
			t.Errorf("issue %q: refusal %q does not end with %q", tc.issue, err, start)
		}
	}
}
