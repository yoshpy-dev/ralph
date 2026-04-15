package panes

import (
	"strings"
	"testing"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/state"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/ui"
)

func TestProgressBarView(t *testing.T) {
	slices := []state.SliceState{
		{Name: "s1", Status: state.StatusComplete, ElapsedSecs: 120},
		{Name: "s2", Status: state.StatusComplete, ElapsedSecs: 180},
		{Name: "s3", Status: state.StatusRunning, ElapsedSecs: 60},
		{Name: "s4", Status: state.StatusPending},
		{Name: "s5", Status: state.StatusPending},
	}

	m := NewProgress(slices, 60)
	view := m.View()

	// Should contain bar characters
	if !strings.Contains(view, "[") || !strings.Contains(view, "]") {
		t.Error("expected progress bar brackets")
	}

	// Should show percentage
	if !strings.Contains(view, "40%") {
		t.Errorf("expected 40%% in view, got: %s", view)
	}

	// Should show count
	if !strings.Contains(view, "2/5") {
		t.Errorf("expected 2/5 in view, got: %s", view)
	}

	// Should show ETA
	if !strings.Contains(view, "ETA:") {
		t.Errorf("expected ETA in view, got: %s", view)
	}
}

func TestProgressBarAllComplete(t *testing.T) {
	slices := []state.SliceState{
		{Name: "s1", Status: state.StatusComplete, ElapsedSecs: 60},
		{Name: "s2", Status: state.StatusComplete, ElapsedSecs: 90},
	}

	m := NewProgress(slices, 60)
	view := m.View()

	if !strings.Contains(view, "100%") {
		t.Errorf("expected 100%%, got: %s", view)
	}
	if !strings.Contains(view, "2/2") {
		t.Errorf("expected 2/2, got: %s", view)
	}
	// ETA should be — when all complete
	if !strings.Contains(view, "ETA: —") {
		t.Errorf("expected ETA: — when all complete, got: %s", view)
	}
}

func TestProgressBarEmpty(t *testing.T) {
	m := NewProgress(nil, 60)
	view := m.View()

	if !strings.Contains(view, "0%") {
		t.Errorf("expected 0%%, got: %s", view)
	}
	if !strings.Contains(view, "0/0") {
		t.Errorf("expected 0/0, got: %s", view)
	}
}

func TestProgressBarSingleSlice(t *testing.T) {
	slices := []state.SliceState{
		{Name: "only", Status: state.StatusRunning, ElapsedSecs: 30},
	}

	m := NewProgress(slices, 60)
	view := m.View()

	if !strings.Contains(view, "0%") {
		t.Errorf("expected 0%%, got: %s", view)
	}
	if !strings.Contains(view, "0/1") {
		t.Errorf("expected 0/1, got: %s", view)
	}
}

func TestComputeStats(t *testing.T) {
	tests := []struct {
		name      string
		slices    []state.SliceState
		wantTotal int
		wantComp  int
		wantPct   float64
	}{
		{
			name:      "empty",
			slices:    nil,
			wantTotal: 0,
			wantComp:  0,
			wantPct:   0,
		},
		{
			name: "half done",
			slices: []state.SliceState{
				{Status: state.StatusComplete, ElapsedSecs: 100},
				{Status: state.StatusRunning},
			},
			wantTotal: 2,
			wantComp:  1,
			wantPct:   50,
		},
		{
			name: "all done",
			slices: []state.SliceState{
				{Status: state.StatusComplete, ElapsedSecs: 60},
				{Status: state.StatusComplete, ElapsedSecs: 60},
			},
			wantTotal: 2,
			wantComp:  2,
			wantPct:   100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := ComputeStats(tt.slices)
			if stats.Total != tt.wantTotal {
				t.Errorf("Total = %d, want %d", stats.Total, tt.wantTotal)
			}
			if stats.Completed != tt.wantComp {
				t.Errorf("Completed = %d, want %d", stats.Completed, tt.wantComp)
			}
			if stats.Percent != tt.wantPct {
				t.Errorf("Percent = %f, want %f", stats.Percent, tt.wantPct)
			}
		})
	}
}

func TestEstimateETA(t *testing.T) {
	tests := []struct {
		completed    int
		remaining    int
		totalElapsed int
		expected     string
	}{
		{0, 3, 0, "—"},
		{2, 0, 200, "—"},
		{2, 3, 300, "~7m30s"},
		{1, 1, 60, "~1m0s"},
	}

	for _, tt := range tests {
		result := EstimateETA(tt.completed, tt.remaining, tt.totalElapsed)
		if result != tt.expected {
			t.Errorf("EstimateETA(%d, %d, %d) = %q, want %q",
				tt.completed, tt.remaining, tt.totalElapsed, result, tt.expected)
		}
	}
}

func TestProgressStateUpdatedMsg(t *testing.T) {
	m := NewProgress(nil, 60)

	newSlices := []state.SliceState{
		{Name: "s1", Status: state.StatusComplete, ElapsedSecs: 60},
	}
	m, _ = m.Update(ui.StateUpdatedMsg{
		Status: state.FullStatus{
			Slices: newSlices,
		},
	})

	stats := ComputeStats(m.slices)
	if stats.Completed != 1 {
		t.Fatalf("expected 1 completed after StateUpdatedMsg, got %d", stats.Completed)
	}
}
