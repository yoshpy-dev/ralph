package org

import (
	"crypto/sha256"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// `ralph org start --plan --issue` at the org layer (plan
// 2026-11-22-org-start-issue, AC1 to AC6): the issue line of the leader's
// task, the issue in the binding, the restart of an org started for an
// issue, and the refusal of a value that is not an issue number. The
// featureStart fixture lives in feature_test.go.

// issueTaskLine is the task's issue line for issue.
func issueTaskLine(issue string) string {
	return "- issue: #" + issue + "(PR の本文に Closes #" + issue + " を書く)"
}

// issueTaskFixture is a split plan and a feature with a preamble, shared
// paths, a dependency and a body, so that the task has every optional part.
func issueTaskFixture() (*SplitPlan, SplitFeature) {
	plan := &SplitPlan{ID: "auth-split", Path: "/state/splits/auth-split.md", Digest: "0123456789ab",
		Preamble: "## 目的\n\n- 決まり", Shared: []string{"packages/shared/", "pnpm-lock.yaml"}}
	f := SplitFeature{Slug: "auth-docs", Type: "feat", Reserve: []string{"docs/auth/", "docs/x.md"},
		DependsOn: []string{"auth-core"}, Body: "Objective: document it.\n\n- [ ] AC1"}
	return plan, f
}

// TestFeatureLeaderTask_IssueLine covers AC1 and AC2 for the task: with an
// issue, the task is the one with no issue and the issue line right after
// the line of the features it depends on (with dependencies and with none,
// with shared paths and with none); with no issue, it holds no issue line
// and is byte for byte the task featureLeaderTask built before the issue
// (its sha256 printed at 81530fb0, the commit before plan
// 2026-11-22-org-start-issue, for issueTaskFixture).
func TestFeatureLeaderTask_IssueLine(t *testing.T) {
	plan, f := issueTaskFixture()
	none := featureLeaderTask(plan, f, "/state", "/wt", "feat/auth-docs", config.ApprovalModePerPlan, "", config.ReviewerConfig{})
	const before = "f6426197684a840b2c33d39c70a2f26a3bc20335de70f28a5d778c84d2fb3d08"
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(none))); got != before {
		t.Fatalf("the task with no issue has sha256 %s, want %s, the one of before:\n%s", got, before, none)
	}
	if strings.Contains(none, featureIssueLead) || strings.Contains(none, "Closes") {
		t.Fatalf("the task with no issue names an issue:\n%s", none)
	}

	bare := SplitFeature{Slug: "auth-core", Type: "fix", Reserve: []string{"internal/auth/"}}
	for _, tc := range []struct {
		name  string
		plan  *SplitPlan
		f     SplitFeature
		after string
	}{
		{"a dependency, shared paths, a preamble and a body", plan, f, "- 依存する機能: auth-core\n"},
		{"no dependency, no shared paths", &SplitPlan{ID: "s", Path: "/state/splits/s.md", Digest: "0123456789ab"}, bare, "- 依存する機能: なし\n"},
	} {
		for _, issue := range []string{"12", "1", "9999999999"} {
			without := featureLeaderTask(tc.plan, tc.f, "/state", "/wt", "feat/x", config.ApprovalModeSplit, "", config.ReviewerConfig{})
			with := featureLeaderTask(tc.plan, tc.f, "/state", "/wt", "feat/x", config.ApprovalModeSplit, issue, config.ReviewerConfig{})
			want := strings.Replace(without, "\n"+tc.after, "\n"+tc.after+issueTaskLine(issue)+"\n", 1)
			if want == without {
				t.Fatalf("%s: no %q line in the task:\n%s", tc.name, tc.after, without)
			}
			if with != want {
				t.Errorf("%s, issue %s: task =\n%s\nwant\n%s", tc.name, issue, with, want)
			}
			if n := strings.Count(with, featureIssueLead); n != 1 {
				t.Errorf("%s, issue %s: %d issue lines, want 1:\n%s", tc.name, issue, n, with)
			}
			if !strings.Contains(with, "\n"+issueTaskLine(issue)+"\n- 承認方式: split\n") {
				t.Errorf("%s, issue %s: want the approval mode line right after the issue line:\n%s", tc.name, issue, with)
			}
		}
	}
}

// startAuthCoreFor starts auth-core of the fixture's split plan with issue
// ("" for none) and returns the binding the org reads back.
func (st *featureStart) startAuthCoreFor(t *testing.T, issue string) *FeatureBinding {
	t.Helper()
	p := st.params("auth-core")
	p.Issue = issue
	st.mustStart(t, p, SpawnOutcomeSpawned)
	return ActiveFeature(mustReadEvents(t, st.o), "auth-core")
}

// freshIssueLeaderTask is freshLeaderTask (reseat_progress_start_test.go)
// with the issue issue: auth-core's task as featureLeaderTask builds it from
// the split plan now.
func (st *featureStart) freshIssueLeaderTask(t *testing.T, issue string) string {
	t.Helper()
	path, _, err := ResolveSplitPlanPath(st.stateDir, st.plan)
	if err != nil {
		t.Fatalf("ResolveSplitPlanPath: %v", err)
	}
	plan, err := LoadSplitPlan(path)
	if err != nil {
		t.Fatalf("LoadSplitPlan: %v", err)
	}
	feature, ok := plan.Feature("auth-core")
	if !ok {
		t.Fatal("the split plan has no feature auth-core")
	}
	return featureLeaderTask(plan, feature, st.stateDir, st.worktree("auth-core"), "fix/auth-core", config.ApprovalModePerPlan, issue, config.ReviewerConfig{})
}

// TestStartFeature_RecordsTheIssue covers AC1 at the org layer: a start
// with an issue records it as the last token of the feature's
// scope_reserved, right after its feature digest, and that is the binding
// the org reads back; the feature digest is the one a start with no issue
// records; and the leader's task is featureLeaderTask's with the issue
// line after the line of the features it depends on.
func TestStartFeature_RecordsTheIssue(t *testing.T) {
	st := newFeatureStart(t)
	got := st.startAuthCoreFor(t, "365")
	wt, digest, featureDigest := st.worktree("auth-core"), planDigestOf(t, st.plan), featureDigestOf(t, st.plan, "auth-core")
	want := "paths=docs/auth.md,internal/auth/ split=auth-split feature=auth-core digest=" + digest + " branch=fix/auth-core approval=per-plan" +
		" feature_digest=" + featureDigest + " issue=365"
	if ev := eventOf(t, st.o, "auth-core", "", EventScopeReserved); ev.Details != want || ev.Worktree != wt {
		t.Fatalf("scope_reserved = %+v, want Details %q and Worktree %q", ev, want, wt)
	}
	wantBinding := &FeatureBinding{
		Split: "auth-split", Feature: "auth-core", Digest: digest, Branch: "fix/auth-core", Worktree: wt, Approval: config.ApprovalModePerPlan,
		FeatureDigest: featureDigest, Issue: "365",
	}
	assertActiveFeature(t, st.o, "auth-core", wantBinding)
	if got == nil || *got != *wantBinding {
		t.Fatalf("binding = %+v, want %+v", got, wantBinding)
	}

	task := st.leaderTaskOf(t)
	if fresh := st.freshIssueLeaderTask(t, "365"); task != fresh {
		t.Fatalf("the leader's task =\n%s\nwant featureLeaderTask's\n%s", task, fresh)
	}
	if !strings.Contains(task, "\n- 依存する機能: なし\n"+issueTaskLine("365")+"\n- 承認方式: per-plan\n") {
		t.Fatalf("the leader's task does not hold the issue line after the dependency line:\n%s", task)
	}
}

// TestStartFeature_NoIssueIsTheStartOfBefore covers AC2 at the org layer: a
// start with no issue records no `issue=` token, its binding has no issue,
// and its leader's task is featureLeaderTask's with no issue, which
// TestFeatureLeaderTask_IssueLine pins to the task of before.
func TestStartFeature_NoIssueIsTheStartOfBefore(t *testing.T) {
	st := newFeatureStart(t)
	got := st.startAuthCoreFor(t, "")
	if ev := eventOf(t, st.o, "auth-core", "", EventScopeReserved); strings.Contains(ev.Details, "issue") {
		t.Fatalf("scope_reserved Details = %q, want no issue token", ev.Details)
	}
	if got == nil || got.Issue != "" || !got.Complete() {
		t.Fatalf("binding = %+v, want a complete one with no issue", got)
	}
	task := st.leaderTaskOf(t)
	if fresh := st.freshIssueLeaderTask(t, ""); task != fresh || strings.Contains(task, featureIssueLead) {
		t.Fatalf("the leader's task =\n%s\nwant featureLeaderTask's with no issue\n%s", task, fresh)
	}
}

// TestStartFeature_IssueRestart covers AC5: an org not disbanded starts
// again, after its leader stopped, only with the issue it was started with
// (both none included): the same start while the leader runs records and
// calls nothing, and after the leader stopped it spawns a new leader with no
// second scope_reserved and the issue line in its task. A start with
// another issue, with none when the org has one, or with one when it has
// none is refused, naming the issue of each side and the --issue that
// starts it as it was, with nothing recorded or called.
func TestStartFeature_IssueRestart(t *testing.T) {
	for _, issue := range []string{"", "12"} {
		t.Run("the same issue: "+describeIssue(issue), func(t *testing.T) {
			st := newFeatureStart(t)
			first := st.startAuthCoreFor(t, issue)
			p := st.params("auth-core")
			p.Issue = issue
			before := takeSpawnSnapshot(t, st.o, st.h, st.a)
			st.mustStart(t, p, SpawnOutcomeIdempotent)
			if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
				t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
			}

			st.stopLeader(t, "auth-core")
			st.h.nextPaneID = "pane-2"
			st.mustStart(t, p, SpawnOutcomeSpawned)
			if n := countString(eventNames(t, st.o), EventScopeReserved); n != 1 {
				t.Fatalf("expected one scope_reserved, got %d", n)
			}
			if ev := eventOf(t, st.o, "auth-core", LeaderIdentity, EventSpawned); ev.PaneID != "pane-2" {
				t.Fatalf("expected a new leader, got %+v", ev)
			}
			assertActiveFeature(t, st.o, "auth-core", first)
			if task, fresh := st.leaderTaskOf(t), st.freshIssueLeaderTask(t, issue); task != fresh && !strings.HasPrefix(task, fresh+"\n\n") {
				t.Fatalf("the restarted leader's task =\n%s\nwant featureLeaderTask's\n%s", task, fresh)
			}
		})
	}

	for _, tc := range []struct {
		name, held, want, clause string
	}{
		{"another issue", "12", "13", `started with issue #12 and this start has issue #13 ` +
			`(a start of the org that is not disbanded names the issue it was started with: run it with --issue 12)`},
		{"no issue now", "12", "", `started with issue #12 and this start has no issue ` +
			`(a start of the org that is not disbanded names the issue it was started with: run it with --issue 12)`},
		{"no issue before", "", "13", `started with no issue and this start has issue #13 ` +
			`(a start of the org that is not disbanded names the issue it was started with: run it with no --issue)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newFeatureStart(t)
			first := st.startAuthCoreFor(t, tc.held)
			st.stopLeader(t, "auth-core")
			p := st.params("auth-core")
			p.Issue = tc.want
			before := takeSpawnSnapshot(t, st.o, st.h, st.a)
			calls := slices.Clone(st.wt.calls)
			assertStartRefused(t, st.o.StartFeature(p),
				`org_id "auth-core" is bound to feature "auth-core" of split plan "auth-split"`,
				`is refused, since org_id "auth-core" was `+tc.clause+`: ralph org disband --org-id auth-core first, or use another org_id`)
			if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
				t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
			}
			st.wantCalls(t, calls...)
			assertActiveFeature(t, st.o, "auth-core", first)
		})
	}
}

// TestStartFeature_IssueFormRefused covers AC3 at the org layer: a value
// that is not an issue number (zero, a leading zero, a `#`, a sign, a
// letter, white space, a non-ASCII digit, eleven digits) is refused before
// the split plan is read, with an error that names the value and the form,
// and nothing is recorded, called or looked up. "" is no issue here: the
// CLI refuses an empty --issue. One and ten digits start.
func TestStartFeature_IssueFormRefused(t *testing.T) {
	st := newFeatureStart(t)
	before := takeSpawnSnapshot(t, st.o, st.h, st.a)
	for _, bad := range []string{"0", "00", "012", "#12", "+12", "-12", "12a", "abc", " 12", "12 ", "1 2", "1.0", "１２", "12345678901"} {
		p := st.params("auth-core")
		p.Issue = bad
		res := st.o.StartFeature(p)
		assertStartRefused(t, res, fmt.Sprintf(`org: --issue %q is not an issue number: give the number of the issue alone, `+
			`1 to 10 decimal digits with no leading zero and no # (--issue 12 for issue #12)`, bad))
		if res.Split != nil || res.OrgID != "" {
			t.Errorf("issue %q: result names split plan %v and org_id %q, want the refusal before the split plan is read", bad, res.Split, res.OrgID)
		}
	}
	if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
		t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
	}
	st.wantCalls(t)
	if _, err := os.Stat(st.worktree("auth-core")); !os.IsNotExist(err) {
		t.Fatalf("expected no worktree, stat: %v", err)
	}

	for _, ok := range []string{"1", "9999999999"} {
		st := newFeatureStart(t)
		if got := st.startAuthCoreFor(t, ok); got == nil || got.Issue != ok {
			t.Fatalf("issue %s: binding = %+v, want it recorded", ok, got)
		}
	}
}
