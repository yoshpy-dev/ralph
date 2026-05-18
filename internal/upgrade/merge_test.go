package upgrade

import "testing"

func TestPlanMerge_TemplateOnlyChange(t *testing.T) {
	plan := PlanMerge(
		[]byte("a\nold\nz\n"),
		[]byte("a\nold\nz\n"),
		[]byte("a\nnew\nz\n"),
	)
	if got := plan.ConflictCount(); got != 1 {
		t.Fatalf("ConflictCount = %d, want 1", got)
	}
	region := plan.Regions[0]
	assertLines(t, "local", region.LocalLines, []string{"old"})
	assertLines(t, "template", region.TemplateLines, []string{"new"})
}

func TestPlanMerge_LocalOnlyChange(t *testing.T) {
	plan := PlanMerge(
		[]byte("a\nold\nz\n"),
		[]byte("a\nmine\nz\n"),
		[]byte("a\nold\nz\n"),
	)
	if got := plan.ConflictCount(); got != 1 {
		t.Fatalf("ConflictCount = %d, want 1", got)
	}
	region := plan.Regions[0]
	assertLines(t, "local", region.LocalLines, []string{"mine"})
	assertLines(t, "template", region.TemplateLines, []string{"old"})
}

func TestPlanMerge_NonOverlappingChanges(t *testing.T) {
	plan := PlanMerge(
		[]byte("a\nold-local\nmiddle\nold-template\nz\n"),
		[]byte("a\nmine\nmiddle\nold-template\nz\n"),
		[]byte("a\nold-local\nmiddle\ntheirs\nz\n"),
	)
	if got := plan.ConflictCount(); got != 2 {
		t.Fatalf("ConflictCount = %d, want 2", got)
	}
	assertLines(t, "local first", plan.Regions[0].LocalLines, []string{"mine"})
	assertLines(t, "template first", plan.Regions[0].TemplateLines, []string{"old-local"})
	assertLines(t, "local second", plan.Regions[1].LocalLines, []string{"old-template"})
	assertLines(t, "template second", plan.Regions[1].TemplateLines, []string{"theirs"})
}

func TestPlanMerge_OverlappingChanges(t *testing.T) {
	plan := PlanMerge(
		[]byte("a\nold\nz\n"),
		[]byte("a\nmine\nz\n"),
		[]byte("a\ntheirs\nz\n"),
	)
	if got := plan.ConflictCount(); got != 1 {
		t.Fatalf("ConflictCount = %d, want 1", got)
	}
	assertLines(t, "local", plan.Regions[0].LocalLines, []string{"mine"})
	assertLines(t, "template", plan.Regions[0].TemplateLines, []string{"theirs"})
}

func TestPlanMerge_SameLocalAndTemplateChangeAutoResolves(t *testing.T) {
	plan := PlanMerge(
		[]byte("a\nold\nz\n"),
		[]byte("a\nsame\nz\n"),
		[]byte("a\nsame\nz\n"),
	)
	if got := plan.ConflictCount(); got != 0 {
		t.Fatalf("ConflictCount = %d, want 0", got)
	}
}

func TestPlanMerge_InsertionAtSamePoint(t *testing.T) {
	plan := PlanMerge(
		[]byte("a\nz\n"),
		[]byte("a\nmine\nz\n"),
		[]byte("a\ntheirs\nz\n"),
	)
	if got := plan.ConflictCount(); got != 1 {
		t.Fatalf("ConflictCount = %d, want 1", got)
	}
	assertLines(t, "local", plan.Regions[0].LocalLines, []string{"mine"})
	assertLines(t, "template", plan.Regions[0].TemplateLines, []string{"theirs"})
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
