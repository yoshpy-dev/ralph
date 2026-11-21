package org

import (
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// The leader's task for a feature of a split plan with shared paths (plan
// docs/plans/active/2026-11-22-split-shared-paths.md, AC8): the shared paths
// line, and the seats' --scope that covers the reservation and the shared
// paths, at a first start, a `start --plan` restart, and a reseat.
// TestFeatureLeaderTask pins the task of a plan with no shared paths, which
// is the one featureLeaderTask built before that plan.

// sharedTaskLine is the task's shared paths line for the shared paths
// shared, joined with ", ".
func sharedTaskLine(shared string) string {
	return featureSharedLead + " " + shared + "(どの機能の org も変えてよい。座席の spawn の --scope には予約したパスと合わせて渡す)"
}

// TestFeatureLeaderTask_SharedPaths pins the task of a feature whose split
// plan has shared paths: the shared paths line right after the reserved
// paths line, whose --scope is the reserved paths and then the shared paths
// as one shell word, and every other line as for a plan with none. A plan
// whose Shared is empty but not nil gives the same task as one whose Shared
// is nil, with no shared paths line.
func TestFeatureLeaderTask_SharedPaths(t *testing.T) {
	f := SplitFeature{
		Slug: "auth-docs", Type: "feat", Reserve: []string{"docs/auth/", "docs/x.md"},
		DependsOn: []string{"auth-core"}, Body: "Objective: document it.\n\n- [ ] AC1",
	}
	plan := func(shared []string) *SplitPlan {
		return &SplitPlan{ID: "auth-split", Path: "/state/splits/auth-split.md", Digest: "0123456789ab", Shared: shared}
	}
	task := func(shared []string) string {
		return featureLeaderTask(plan(shared), f, "/state", "/wt", "feat/auth-docs", config.ApprovalModePerPlan, config.ReviewerConfig{})
	}
	head := func(reserved string) string {
		return "- 分割計画: auth-split(/state/splits/auth-split.md、承認の digest 0123456789ab)\n" +
			"- 機能: auth-docs\n" +
			"- worktree: /wt(leader の cwd)\n" +
			"- ブランチ: feat/auth-docs\n" +
			"- 予約したパス: " + reserved + "\n"
	}
	tail := "- 依存する機能: auth-core\n" +
		"- 承認方式: per-plan\n" +
		"- 進め方: `/org` skill の「機能ごとの org」の手順に従う\n" +
		"- 台帳: /state(ralph org のコマンドには必ず --state-dir '/state' を付ける)\n" +
		defaultFeatureReviewerLines() +
		"\n\nObjective: document it.\n\n- [ ] AC1"

	got := task([]string{"packages/shared/", "pnpm-lock.yaml"})
	want := head("docs/auth/, docs/x.md(座席の spawn には --scope 'docs/auth/, docs/x.md, packages/shared/, pnpm-lock.yaml' を付ける)") +
		sharedTaskLine("packages/shared/, pnpm-lock.yaml") + "\n" + tail
	if got != want {
		t.Errorf("task =\n%s\nwant\n%s", got, want)
	}

	none := head("docs/auth/, docs/x.md(座席の spawn には --scope 'docs/auth/, docs/x.md' を付ける)") + tail
	for _, shared := range [][]string{nil, {}} {
		if got := task(shared); got != none {
			t.Errorf("task with Shared %#v =\n%s\nwant\n%s", shared, got, none)
		}
	}
	if strings.Contains(none, featureSharedLead) {
		t.Fatalf("the task of a plan with no shared paths names %q:\n%s", featureSharedLead, none)
	}
}

// TestFeatureLeaderTask_SharedScopeIsOneShellWord: with shared paths, the
// --scope argument of the reserved paths line, run through /bin/sh, is the
// reserved paths and then the shared paths joined with ", " as one argument,
// also for paths with a `$`, a backtick, quotes, a backslash and a glob,
// which the reservation rules let through.
func TestFeatureLeaderTask_SharedScopeIsOneShellWord(t *testing.T) {
	const line, lead, tail = "- 予約したパス: ", "(座席の spawn には --scope ", " を付ける)"
	for _, tc := range []struct{ reserve, shared []string }{
		{[]string{"a/"}, []string{"pnpm-lock.yaml"}},
		{[]string{"docs/auth/", "docs/x.md"}, []string{"go.sum", "packages/shared/"}},
		{[]string{"it's/x.md"}, []string{"app/$HOME/", `say"hi"/`, "x`id`.lock"}},
		{[]string{`g\h`}, []string{"b'c/", "internal/*.go"}},
	} {
		for _, paths := range [][]string{tc.reserve, tc.shared} {
			if got, err := NormalizeReservePaths(paths); err != nil || !slices.Equal(got, paths) {
				t.Fatalf("fixture %q is not normalized reservation paths as it is: NormalizeReservePaths = %q, %v", paths, got, err)
			}
		}
		plan := &SplitPlan{ID: "s", Path: "/state/splits/s.md", Digest: "0123456789ab", Shared: tc.shared}
		task := featureLeaderTask(plan, SplitFeature{Slug: "a", Type: "feat", Reserve: tc.reserve}, "/state", "/wt", "feat/a", config.ApprovalModePerPlan, config.ReviewerConfig{})
		reserved := strings.Join(tc.reserve, ", ")
		var reserveLine string
		for l := range strings.SplitSeq(task, "\n") {
			if strings.HasPrefix(l, line) {
				reserveLine = l
				break
			}
		}
		rest, ok := strings.CutPrefix(reserveLine, line+reserved+lead)
		arg, ok2 := strings.CutSuffix(rest, tail)
		if !ok || !ok2 || arg == "" {
			t.Fatalf("no %s%s%s<arg>%s line in the task:\n%s", line, reserved, lead, tail, task)
		}
		got, err := exec.Command("/bin/sh", "-c", "set -- "+arg+`; printf '%s|%s' "$#" "$1"`).Output()
		if err != nil {
			t.Fatalf("/bin/sh with %s: %v", arg, err)
		}
		if want := "1|" + strings.Join(slices.Concat(tc.reserve, tc.shared), ", "); string(got) != want {
			t.Errorf("--scope %s reads in /bin/sh as %q, want %q", arg, got, want)
		}
		if !strings.Contains(task, "\n"+reserveLine+"\n"+sharedTaskLine(strings.Join(tc.shared, ", "))+"\n") {
			t.Errorf("expected the shared paths line right after the reserved paths line, got:\n%s", task)
		}
	}
}

// sharedRestartLines are the reserved paths line and the shared paths line
// of auth-core's task when its split plan shares packages/shared/ and
// pnpm-lock.yaml (startSharedAuthCore).
const sharedRestartLines = "\n- 予約したパス: docs/auth.md, internal/auth/(座席の spawn には --scope " +
	"'docs/auth.md, internal/auth/, packages/shared/, pnpm-lock.yaml' を付ける)\n" +
	"- 共有のパス: packages/shared/, pnpm-lock.yaml(どの機能の org も変えてよい。座席の spawn の --scope には予約したパスと合わせて渡す)\n"

// TestStartFeature_SharedPathsLineAtStartAndRestart covers AC8 for `ralph org
// start --plan`: the leader's task at the first start is featureLeaderTask's
// with the shared paths line and the --scope of the reservation and the
// shared paths, and the leader a restart after it stopped starts with the
// same task, a progress section after it or not.
func TestStartFeature_SharedPathsLineAtStartAndRestart(t *testing.T) {
	st := newFeatureStart(t)
	st.startSharedAuthCore(t, "pnpm-lock.yaml, packages/shared/")
	fresh := st.freshLeaderTask(t)
	if !strings.Contains(fresh, sharedRestartLines) {
		t.Fatalf("featureLeaderTask of the shared plan does not hold%s\ngot:\n%s", sharedRestartLines, fresh)
	}
	if got := st.leaderTaskOf(t); got != fresh {
		t.Fatalf("the first start's task =\n%s\nwant featureLeaderTask's\n%s", got, fresh)
	}

	st.leaderRestarted(t, st.params("auth-core"))
	if got := st.leaderTaskOf(t); got != fresh && !strings.HasPrefix(got, fresh+"\n\n") {
		t.Errorf("the restarted leader's task =\n%s\nwant featureLeaderTask's\n%s", got, fresh)
	}
}

// TestReseat_SharedPathsLineInTheLeadersTask covers AC8 for `ralph org
// reseat`: a reseat of auth-core's leader starts it with the task of the
// first start, the shared paths line and the --scope of the reservation and
// the shared paths included, before the progress section.
func TestReseat_SharedPathsLineInTheLeadersTask(t *testing.T) {
	st := newFeatureStart(t)
	st.startSharedAuthCore(t, "pnpm-lock.yaml, packages/shared/")
	approveFeaturePlan(t, st.o, "auth-core", st.worktree("auth-core"))
	st.spawnSeat(t, implementerSeatID, implementerRole, "opus")
	st.spawnSeat(t, "reviewer", "reviewer", "")
	st.h.closeRemoves = true
	fresh := st.freshLeaderTask(t)
	if !strings.Contains(fresh, sharedRestartLines) {
		t.Fatalf("featureLeaderTask of the shared plan does not hold%s\ngot:\n%s", sharedRestartLines, fresh)
	}

	st.mustReseat(t, st.reseatParams(LeaderIdentity))
	if got := st.leaderTaskOf(t); !strings.HasPrefix(got, fresh+"\n\n") {
		t.Errorf("the reseated leader's task =\n%s\nwant featureLeaderTask's\n%s\nand the progress section", got, fresh)
	}
}
