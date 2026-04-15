package panes

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/action"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/state"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/ui"
)

// ActionsModel is the actions pane that shows available actions for the selected slice.
type ActionsModel struct {
	selectedSlice *state.SliceState
	executor      *action.Executor
	statusText    string
	statusIsError bool
	width         int
	height        int
	focused       bool
}

// NewActionsModel creates a new actions pane.
func NewActionsModel(executor *action.Executor) ActionsModel {
	return ActionsModel{
		executor: executor,
	}
}

// SetSize sets the available rendering area.
func (m ActionsModel) SetSize(w, h int) ActionsModel {
	m.width = w
	m.height = h
	return m
}

// SetFocused sets whether this pane has focus.
func (m ActionsModel) SetFocused(focused bool) ActionsModel {
	m.focused = focused
	return m
}

// Update handles messages for the actions pane.
func (m ActionsModel) Update(msg tea.Msg) (ActionsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.SliceSelectedMsg:
		s := msg.Slice
		m.selectedSlice = &s
		m.statusText = ""
		m.statusIsError = false
		return m, nil

	case action.RetryResultMsg:
		if msg.Err != nil {
			m.statusText = fmt.Sprintf("Retry failed: %v", msg.Err)
			m.statusIsError = true
		} else {
			m.statusText = fmt.Sprintf("Retry started: %s", msg.SliceName)
			m.statusIsError = false
		}
		return m, nil

	case action.AbortResultMsg:
		target := msg.SliceName
		if target == "" {
			target = "all slices"
		}
		if msg.Err != nil {
			m.statusText = fmt.Sprintf("Abort failed (%s): %v", target, msg.Err)
			m.statusIsError = true
		} else {
			m.statusText = fmt.Sprintf("Abort sent: %s", target)
			m.statusIsError = false
		}
		return m, nil

	case action.ExternalDoneMsg:
		if msg.Err != nil {
			m.statusText = fmt.Sprintf("%s error: %v", msg.Action, msg.Err)
			m.statusIsError = true
		} else {
			m.statusText = ""
			m.statusIsError = false
		}
		return m, nil

	case ui.StatusMsg:
		m.statusText = msg.Text
		m.statusIsError = msg.IsError
		return m, nil
	}

	return m, nil
}

// HandleKey processes a key press and returns a command if the key triggers an action.
// The second return value is a confirmation request (message, tag) if the action needs confirmation.
// The third return value indicates whether the key was consumed.
func (m ActionsModel) HandleKey(msg tea.KeyPressMsg) (tea.Cmd, *ConfirmRequest, bool) {
	if m.selectedSlice == nil {
		return nil, nil, false
	}

	s := m.selectedSlice
	switch msg.String() {
	case "r":
		if !s.CanRetry() {
			return nil, nil, true // consumed but no-op
		}
		req := &ConfirmRequest{
			Message: fmt.Sprintf("Retry slice %q?", s.Name),
			Tag:     "retry:" + s.Name,
		}
		return nil, req, true

	case "a":
		if !s.CanAbort() {
			return nil, nil, true
		}
		req := &ConfirmRequest{
			Message: fmt.Sprintf("Abort slice %q?", s.Name),
			Tag:     "abort:" + s.Name,
		}
		return nil, req, true

	case "A":
		req := &ConfirmRequest{
			Message: "Abort ALL slices?",
			Tag:     "abort-all",
		}
		return nil, req, true

	case "L":
		if !s.HasLogs() {
			return nil, nil, true
		}
		return m.executor.OpenPager(s.LogPath), nil, true

	case "e":
		if !s.HasWorktree() {
			return nil, nil, true
		}
		return m.executor.OpenEditor(s.WorktreePath), nil, true
	}

	return nil, nil, false
}

// ExecuteConfirmed runs the action after confirmation is received.
func (m ActionsModel) ExecuteConfirmed(tag string) tea.Cmd {
	if m.executor == nil {
		return nil
	}

	parts := strings.SplitN(tag, ":", 2)
	switch parts[0] {
	case "retry":
		if len(parts) < 2 {
			return nil
		}
		return m.executor.RetrySlice(parts[1])
	case "abort":
		if len(parts) < 2 {
			return nil
		}
		return m.executor.AbortSlice(parts[1])
	case "abort-all":
		return m.executor.AbortAll()
	}
	return nil
}

// ConfirmRequest represents a request to show a confirmation dialog.
type ConfirmRequest struct {
	Message string
	Tag     string
}

// View renders the actions pane.
func (m ActionsModel) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	b.WriteString(titleStyle.Render("Actions"))
	b.WriteString("\n\n")

	if m.selectedSlice == nil {
		dimStyle := lipgloss.NewStyle().Faint(true)
		b.WriteString(dimStyle.Render("(no slice selected)"))
	} else {
		s := m.selectedSlice
		b.WriteString(m.renderActions(s))
	}

	if m.statusText != "" {
		b.WriteString("\n\n")
		style := lipgloss.NewStyle()
		if m.statusIsError {
			style = style.Foreground(lipgloss.Color("196"))
		} else {
			style = style.Foreground(lipgloss.Color("82"))
		}
		b.WriteString(style.Render(m.statusText))
	}

	return b.String()
}

// renderActions builds the action list based on slice status.
func (m ActionsModel) renderActions(s *state.SliceState) string {
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("45"))
	dimStyle := lipgloss.NewStyle().Faint(true)

	type actionItem struct {
		key     string
		label   string
		enabled bool
	}

	items := []actionItem{
		{"r", "Retry", s.CanRetry()},
		{"a", "Abort", s.CanAbort()},
		{"A", "Abort All", true},
		{"L", "Logs", s.HasLogs()},
		{"e", "Editor", s.HasWorktree()},
	}

	var lines []string
	for _, item := range items {
		if item.enabled {
			line := fmt.Sprintf("%s %s",
				keyStyle.Render("["+item.key+"]"),
				item.label,
			)
			lines = append(lines, line)
		} else {
			line := dimStyle.Render(fmt.Sprintf("[%s] %s", item.key, item.label))
			lines = append(lines, line)
		}
	}

	if len(lines) == 0 {
		return dimStyle.Render("(no actions available)")
	}

	return strings.Join(lines, "\n")
}
