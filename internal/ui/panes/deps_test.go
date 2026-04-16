package panes

import (
	"strings"
	"testing"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/state"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/ui"
)

func TestDepsLinearTree(t *testing.T) {
	deps := []state.SliceDependency{
		{From: "slice-1", To: "slice-2"},
		{From: "slice-2", To: "slice-3"},
	}
	slices := []state.SliceState{
		{Name: "slice-1", Status: state.StatusComplete},
		{Name: "slice-2", Status: state.StatusRunning},
		{Name: "slice-3", Status: state.StatusPending},
	}

	m := NewDeps(deps, slices, 60, 20)
	view := m.View()

	if !strings.Contains(view, "slice-1") {
		t.Error("expected slice-1 in tree")
	}
	if !strings.Contains(view, "slice-2") {
		t.Error("expected slice-2 in tree")
	}
	if !strings.Contains(view, "slice-3") {
		t.Error("expected slice-3 in tree")
	}
}

func TestDepsBranchingTree(t *testing.T) {
	deps := []state.SliceDependency{
		{From: "root", To: "child-a"},
		{From: "root", To: "child-b"},
	}
	slices := []state.SliceState{
		{Name: "root", Status: state.StatusComplete},
		{Name: "child-a", Status: state.StatusRunning},
		{Name: "child-b", Status: state.StatusPending},
	}

	m := NewDeps(deps, slices, 60, 20)
	view := m.View()

	if !strings.Contains(view, "root") {
		t.Error("expected root in tree")
	}
	if !strings.Contains(view, "child-a") {
		t.Error("expected child-a in tree")
	}
	if !strings.Contains(view, "child-b") {
		t.Error("expected child-b in tree")
	}
	// Should have tree connectors
	if !strings.Contains(view, "├") && !strings.Contains(view, "└") {
		t.Error("expected tree connectors in branching tree")
	}
}

func TestDepsNoDependencies(t *testing.T) {
	m := NewDeps(nil, nil, 60, 20)
	view := m.View()

	if view != "No dependencies" {
		t.Fatalf("expected 'No dependencies', got %q", view)
	}
}

func TestDepsSelectedHighlight(t *testing.T) {
	deps := []state.SliceDependency{
		{From: "a", To: "b"},
	}
	slices := []state.SliceState{
		{Name: "a", Status: state.StatusComplete},
		{Name: "b", Status: state.StatusRunning},
	}

	m := NewDeps(deps, slices, 60, 20)
	m.SetSelected("b")
	view := m.View()

	// Selected node should appear in view
	if !strings.Contains(view, "b") {
		t.Error("expected selected node 'b' in view")
	}
}

func TestDepsStatusColors(t *testing.T) {
	deps := []state.SliceDependency{
		{From: "done", To: "active"},
	}
	slices := []state.SliceState{
		{Name: "done", Status: state.StatusComplete},
		{Name: "active", Status: state.StatusFailed},
	}

	m := NewDeps(deps, slices, 60, 20)
	view := m.View()

	// Complete uses +, failed uses !
	if !strings.Contains(view, "+") {
		t.Error("expected + icon for complete slice")
	}
	if !strings.Contains(view, "!") {
		t.Error("expected ! icon for failed slice")
	}
}

func TestDepsSliceSelectedMsg(t *testing.T) {
	m := NewDeps(nil, nil, 60, 20)

	s := state.SliceState{Name: "test-slice", Status: state.StatusRunning}
	m, _ = m.Update(ui.SliceSelectedMsg{Slice: s})

	if m.selected != "test-slice" {
		t.Fatalf("expected selected to be 'test-slice', got %q", m.selected)
	}
}

func TestDepsIndependentSlices(t *testing.T) {
	// Slices with no dependencies between them.
	slices := []state.SliceState{
		{Name: "alpha", Status: state.StatusComplete},
		{Name: "beta", Status: state.StatusRunning},
	}

	m := NewDeps(nil, slices, 60, 20)
	view := m.View()

	if !strings.Contains(view, "alpha") {
		t.Error("expected alpha in view")
	}
	if !strings.Contains(view, "beta") {
		t.Error("expected beta in view")
	}
}
