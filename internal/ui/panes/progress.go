package panes

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/state"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/ui"
)

// ProgressModel displays an overall progress bar with ETA.
type ProgressModel struct {
	slices []state.SliceState
	width  int
}

// NewProgress creates a new ProgressModel.
func NewProgress(slices []state.SliceState, width int) ProgressModel {
	return ProgressModel{slices: slices, width: width}
}

// SetSlices updates the slice data for progress calculation.
func (m *ProgressModel) SetSlices(slices []state.SliceState) {
	m.slices = slices
}

// SetWidth updates the available width.
func (m *ProgressModel) SetWidth(width int) {
	m.width = width
}

// Init implements tea.Model.
func (m ProgressModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m ProgressModel) Update(msg tea.Msg) (ProgressModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.StateUpdatedMsg:
		m.slices = msg.Status.Slices
	}
	return m, nil
}

// Stats holds computed progress statistics.
type Stats struct {
	Total     int
	Completed int
	Percent   float64
	ETA       string
}

// ComputeStats calculates progress statistics from slices.
func ComputeStats(slices []state.SliceState) Stats {
	total := len(slices)
	if total == 0 {
		return Stats{ETA: "—"}
	}

	completed := 0
	totalElapsed := 0
	for _, s := range slices {
		if s.Status == state.StatusComplete {
			completed++
			totalElapsed += s.Elapsed
		}
	}

	pct := float64(completed) / float64(total) * 100
	eta := EstimateETA(completed, total-completed, totalElapsed)

	return Stats{
		Total:     total,
		Completed: completed,
		Percent:   pct,
		ETA:       eta,
	}
}

// EstimateETA calculates estimated time remaining.
// Matches the logic in ralph-status-helpers.sh estimate_eta().
func EstimateETA(completed, remaining, totalElapsed int) string {
	if completed == 0 || remaining == 0 {
		return "—"
	}
	avg := totalElapsed / completed
	eta := avg * remaining
	return "~" + FormatDuration(eta)
}

// View renders the progress bar.
func (m ProgressModel) View() string {
	stats := ComputeStats(m.slices)

	// Build the bar: [####.....] XX% (N/M)  ETA: Xm
	suffix := fmt.Sprintf(" %.0f%% (%d/%d)  ETA: %s", stats.Percent, stats.Completed, stats.Total, stats.ETA)

	barWidth := m.width - len(suffix) - 2 // 2 for [ and ]
	if barWidth < 5 {
		barWidth = 5
	}

	filled := 0
	if stats.Total > 0 {
		filled = int(float64(barWidth) * stats.Percent / 100)
	}
	if filled > barWidth {
		filled = barWidth
	}

	bar := "[" + strings.Repeat("#", filled) + strings.Repeat(".", barWidth-filled) + "]"

	return bar + suffix
}
