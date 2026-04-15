package panes

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/state"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/ui"
)

// DetailModel shows details for the currently selected slice.
type DetailModel struct {
	slice   *state.SliceState
	width   int
	height  int
	focused bool
}

// NewDetail creates a new DetailModel.
func NewDetail(width, height int) DetailModel {
	return DetailModel{width: width, height: height}
}

// SetSlice updates the displayed slice.
func (m *DetailModel) SetSlice(s *state.SliceState) {
	m.slice = s
}

// SetSize updates the pane dimensions.
func (m *DetailModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetFocused sets the focus state.
func (m *DetailModel) SetFocused(focused bool) {
	m.focused = focused
}

// Init implements tea.Model.
func (m DetailModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m DetailModel) Update(msg tea.Msg) (DetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.SliceSelectedMsg:
		s := msg.Slice
		m.slice = &s
	}
	return m, nil
}

// View renders the detail pane.
func (m DetailModel) View() string {
	if m.slice == nil {
		return "No slice selected"
	}

	s := m.slice
	var b strings.Builder

	// Name
	nameStyle := lipgloss.NewStyle().Bold(true)
	b.WriteString(nameStyle.Render(s.Name))
	b.WriteString("\n\n")

	// Status with color
	statusColor := ui.StatusColor(string(s.Status))
	statusStyle := lipgloss.NewStyle().Foreground(statusColor)
	icon := ui.StatusIcon(string(s.Status))
	fmt.Fprintf(&b, "Status:  %s %s\n", statusStyle.Render(icon), statusStyle.Render(string(s.Status)))

	// Phase with color
	if s.Phase != "" {
		phaseColor := ui.PhaseColor(s.Phase)
		phaseStyle := lipgloss.NewStyle().Foreground(phaseColor)
		fmt.Fprintf(&b, "Phase:   %s\n", phaseStyle.Render(s.Phase))
	}

	// Cycle
	if s.MaxCycles > 0 {
		fmt.Fprintf(&b, "Cycle:   %d/%d\n", s.Cycle, s.MaxCycles)
	} else if s.Cycle > 0 {
		fmt.Fprintf(&b, "Cycle:   %d\n", s.Cycle)
	}

	// Elapsed
	fmt.Fprintf(&b, "Elapsed: %s\n", FormatDuration(s.Elapsed))

	// Test result
	if s.TestResult != "" {
		fmt.Fprintf(&b, "Tests:   %s\n", s.TestResult)
	}

	// PR URL
	if s.PRURL != "" {
		fmt.Fprintf(&b, "PR:      %s\n", s.PRURL)
	}

	return b.String()
}

// FormatDuration formats seconds into a human-readable duration string.
func FormatDuration(seconds int) string {
	if seconds < 0 {
		return "—"
	}
	if seconds == 0 {
		return "0s"
	}

	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
