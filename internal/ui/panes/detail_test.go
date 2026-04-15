package panes

import (
	"strings"
	"testing"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/state"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/ui"
)

func TestDetailViewFields(t *testing.T) {
	m := NewDetail(60, 20)
	s := state.SliceState{
		Name:       "slice-1",
		Status:     state.StatusRunning,
		Phase:      "inner",
		Cycle:      2,
		MaxCycles:  5,
		Elapsed:    125,
		TestResult: "pass",
		PRURL:      "https://github.com/org/repo/pull/42",
	}
	m.SetSlice(&s)

	view := m.View()

	checks := []struct {
		label    string
		expected string
	}{
		{"name", "slice-1"},
		{"status label", "Status:"},
		{"status icon", "*"},
		{"status value", "running"},
		{"phase", "inner"},
		{"cycle", "2/5"},
		{"elapsed", "2m5s"},
		{"test result", "pass"},
		{"PR URL", "https://github.com/org/repo/pull/42"},
	}

	for _, c := range checks {
		if !strings.Contains(view, c.expected) {
			t.Errorf("expected %s (%q) in view, got:\n%s", c.label, c.expected, view)
		}
	}
}

func TestDetailNoSliceSelected(t *testing.T) {
	m := NewDetail(60, 20)
	view := m.View()

	if !strings.Contains(view, "No slice selected") {
		t.Fatalf("expected 'No slice selected', got %q", view)
	}
}

func TestDetailSliceSelectedMsg(t *testing.T) {
	m := NewDetail(60, 20)

	s := state.SliceState{
		Name:   "slice-3",
		Status: state.StatusComplete,
	}
	m, _ = m.Update(ui.SliceSelectedMsg{Slice: s})

	view := m.View()
	if !strings.Contains(view, "slice-3") {
		t.Fatalf("expected 'slice-3' in view after SliceSelectedMsg, got:\n%s", view)
	}
}

func TestDetailCycleWithoutMax(t *testing.T) {
	m := NewDetail(60, 20)
	s := state.SliceState{
		Name:   "test",
		Status: state.StatusRunning,
		Cycle:  3,
	}
	m.SetSlice(&s)

	view := m.View()
	if !strings.Contains(view, "Cycle:   3") {
		t.Errorf("expected cycle without max, got:\n%s", view)
	}
}

func TestDetailNoCycleNoPhasePRTestResult(t *testing.T) {
	m := NewDetail(60, 20)
	s := state.SliceState{
		Name:    "minimal",
		Status:  state.StatusPending,
		Elapsed: 0,
	}
	m.SetSlice(&s)

	view := m.View()
	if strings.Contains(view, "Phase:") {
		t.Error("expected no Phase line for empty phase")
	}
	if strings.Contains(view, "Tests:") {
		t.Error("expected no Tests line for empty test result")
	}
	if strings.Contains(view, "PR:") {
		t.Error("expected no PR line for empty PR URL")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0s"},
		{-1, "—"},
		{30, "30s"},
		{90, "1m30s"},
		{125, "2m5s"},
		{3661, "1h1m1s"},
		{7200, "2h0m0s"},
	}

	for _, tt := range tests {
		result := FormatDuration(tt.input)
		if result != tt.expected {
			t.Errorf("FormatDuration(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
