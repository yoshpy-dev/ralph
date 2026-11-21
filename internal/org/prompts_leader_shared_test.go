package org

import (
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// TestRenderRolePrompt_Leader_FeatureOrgSharedPaths pins AC8 of plan
// docs/plans/active/2026-11-22-split-shared-paths.md in the leader template:
// the intro of the 機能ごとの org section, after it keeps the feature's
// changes in the reservation, lets the leader change the paths of the task's
// shared paths line (featureSharedLead, the words featureLeaderTask starts
// that line with), says the seats' --scope on the reserved paths line holds
// both, and has a conflict on a shared path resolved the way the /org skill
// says, through the implementer's TASK. The implementer template keeps its
// writes to its scope only and says nothing of the reservation, so the
// --scope that holds the shared paths is all it needs.
func TestRenderRolePrompt_Leader_FeatureOrgSharedPaths(t *testing.T) {
	text := renderSeatPrompt(t, "leader")
	section, found := markdownSection(text, "## 機能ごとの org")
	if !found {
		t.Fatalf("expected the leader template to contain a '## 機能ごとの org' section, got:\n%s", text)
	}
	intro, _, ok := strings.Cut(section, "\n1. ")
	if !ok {
		t.Fatalf("expected the 機能ごとの org section to have numbered steps, got:\n%s", section)
	}
	squashed := squashSpace(intro)
	at := -1
	for _, phrase := range []string{
		"機能のコードと文書の変更は `- 予約したパス:` の中に収めて",
		"タスクに `" + featureSharedLead + "` の行があれば、そこに並ぶパスも変えてかまいません",
		"lockfile のように、分割計画のどの機能の org も変えてよいと決めたパスです",
		"座席の spawn に付ける `--scope`(`- 予約したパス:` の行の括弧にある引数)には、予約したパスと共有のパスの両方が入っています",
		"共有のパスは並んで走るほかの機能の org も変えるので、rebase がそこで衝突することがあります",
		"implementer に衝突の解消を頼むときは、`/org` skill に書かれた共有のパスの直し方に従うよう TASK に書きます",
	} {
		i := strings.Index(squashed, squashSpace(phrase))
		switch {
		case i < 0:
			t.Errorf("expected the 機能ごとの org intro to say %q, got intro:\n%s", phrase, intro)
		case i < at:
			t.Errorf("expected %q to come after the sentences before it in the 機能ごとの org intro, got intro:\n%s", phrase, intro)
		default:
			at = i
		}
	}

	// The template's words name the line featureLeaderTask writes, and the
	// --scope it points at holds the shared paths. Like the template, the
	// line says that --scope already holds them (sharedTaskLine), so the
	// leader copies it and does not build one.
	task := featureLeaderTask(&SplitPlan{ID: "s", Path: "/state/splits/s.md", Digest: "0123456789ab", Shared: []string{"pnpm-lock.yaml"}},
		SplitFeature{Slug: "a", Type: "feat", Reserve: []string{"a/"}}, "/state", "/wt", "feat/a", config.ApprovalModePerPlan, config.ReviewerConfig{})
	if !strings.Contains(task, "\n- 予約したパス: a/(座席の spawn には --scope 'a/, pnpm-lock.yaml' を付ける)\n"+sharedTaskLine("pnpm-lock.yaml")+"\n") {
		t.Fatalf("featureLeaderTask no longer writes the %q line after the reserved paths line with both in its --scope; got:\n%s", featureSharedLead, task)
	}

	if implementer := renderSeatPrompt(t, "implementer"); strings.Contains(implementer, "予約") {
		t.Errorf("expected the implementer template to bound its writes by its scope only, not by the reservation, got:\n%s", implementer)
	}
}
