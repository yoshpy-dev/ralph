package upgrade

import "testing"

func TestPlanMerge_TemplateOnlyChange(t *testing.T) {
	plan := PlanMerge(
		[]byte("a\nold\nz\n"),
		[]byte("a\nold\nz\n"),
		[]byte("a\nnew\nz\n"),
	)
	if got := plan.DecisionCount(); got != 1 {
		t.Fatalf("DecisionCount = %d, want 1", got)
	}
	h := plan.Hunks[0]
	assertLines(t, "local", h.LocalLines, []string{"old"})
	assertLines(t, "template", h.TemplateLines, []string{"new"})

	got := string(plan.Render(map[int]MergeDecision{
		h.Index: {Choice: MergeUseTemplate},
	}))
	if got != "a\nnew\nz\n" {
		t.Fatalf("render apply = %q", got)
	}
}

func TestPlanMerge_LocalOnlyChange(t *testing.T) {
	plan := PlanMerge(
		[]byte("a\nold\nz\n"),
		[]byte("a\nmine\nz\n"),
		[]byte("a\nold\nz\n"),
	)
	if got := plan.DecisionCount(); got != 1 {
		t.Fatalf("DecisionCount = %d, want 1", got)
	}
	h := plan.Hunks[0]
	assertLines(t, "local", h.LocalLines, []string{"mine"})
	assertLines(t, "template", h.TemplateLines, []string{"old"})

	got := string(plan.Render(map[int]MergeDecision{
		h.Index: {Choice: MergeUseLocal},
	}))
	if got != "a\nmine\nz\n" {
		t.Fatalf("render keep = %q", got)
	}
}

func TestPlanMerge_NonOverlappingChanges(t *testing.T) {
	plan := PlanMerge(
		[]byte("a\nold-local\nmiddle\nold-template\nz\n"),
		[]byte("a\nmine\nmiddle\nold-template\nz\n"),
		[]byte("a\nold-local\nmiddle\ntheirs\nz\n"),
	)
	if got := plan.DecisionCount(); got != 2 {
		t.Fatalf("DecisionCount = %d, want 2", got)
	}
	got := string(plan.Render(map[int]MergeDecision{
		plan.Hunks[0].Index: {Choice: MergeUseLocal},
		plan.Hunks[1].Index: {Choice: MergeUseTemplate},
	}))
	if got != "a\nmine\nmiddle\ntheirs\nz\n" {
		t.Fatalf("render mixed = %q", got)
	}
}

func TestPlanMerge_OverlappingChangesAndEdit(t *testing.T) {
	plan := PlanMerge(
		[]byte("a\nold\nz\n"),
		[]byte("a\nmine\nz\n"),
		[]byte("a\ntheirs\nz\n"),
	)
	if got := plan.DecisionCount(); got != 1 {
		t.Fatalf("DecisionCount = %d, want 1", got)
	}
	got := string(plan.Render(map[int]MergeDecision{
		plan.Hunks[0].Index: {
			Choice:      MergeUseEdited,
			EditedLines: []string{"merged"},
		},
	}))
	if got != "a\nmerged\nz\n" {
		t.Fatalf("render edited = %q", got)
	}
}

func TestPlanMerge_SameLocalAndTemplateChangeAutoResolves(t *testing.T) {
	plan := PlanMerge(
		[]byte("a\nold\nz\n"),
		[]byte("a\nsame\nz\n"),
		[]byte("a\nsame\nz\n"),
	)
	if got := plan.DecisionCount(); got != 0 {
		t.Fatalf("DecisionCount = %d, want 0", got)
	}
	got := string(plan.Render(nil))
	if got != "a\nsame\nz\n" {
		t.Fatalf("render auto = %q", got)
	}
}

func TestPlanMerge_InsertionAtSamePoint(t *testing.T) {
	plan := PlanMerge(
		[]byte("a\nz\n"),
		[]byte("a\nmine\nz\n"),
		[]byte("a\ntheirs\nz\n"),
	)
	if got := plan.DecisionCount(); got != 1 {
		t.Fatalf("DecisionCount = %d, want 1", got)
	}
	got := string(plan.Render(map[int]MergeDecision{
		plan.Hunks[0].Index: {Choice: MergeUseTemplate},
	}))
	if got != "a\ntheirs\nz\n" {
		t.Fatalf("render insertion = %q", got)
	}
}

func assertLines(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s lines len = %d, want %d (%v)", label, len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s[%d] = %q, want %q", label, i, got[i], want[i])
		}
	}
}
