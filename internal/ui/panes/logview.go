package panes

import (
	"regexp"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/ui"
)

// ansiRegexp matches ANSI escape sequences.
var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape sequences from a string.
func StripANSI(s string) string {
	return ansiRegexp.ReplaceAllString(s, "")
}

// LogViewModel wraps a viewport for displaying log output.
type LogViewModel struct {
	viewport   viewport.Model
	lines      []string
	autoScroll bool
	focused    bool
	width      int
	height     int
}

// NewLogView creates a new LogViewModel.
func NewLogView(width, height int) LogViewModel {
	vp := viewport.New(viewport.WithWidth(width), viewport.WithHeight(height))
	return LogViewModel{
		viewport:   vp,
		autoScroll: true,
		width:      width,
		height:     height,
	}
}

// SetSize updates the pane dimensions.
func (m *LogViewModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.SetWidth(width)
	m.viewport.SetHeight(height)
}

// SetFocused sets the focus state.
func (m *LogViewModel) SetFocused(focused bool) {
	m.focused = focused
}

// SetContent replaces all log content.
func (m *LogViewModel) SetContent(content string) {
	cleaned := StripANSI(content)
	m.lines = strings.Split(cleaned, "\n")
	m.viewport.SetContent(cleaned)
	if m.autoScroll {
		m.viewport.GotoBottom()
	}
}

// AppendLine adds a log line and auto-scrolls if enabled.
func (m *LogViewModel) AppendLine(line string) {
	cleaned := StripANSI(line)
	m.lines = append(m.lines, cleaned)
	m.viewport.SetContent(strings.Join(m.lines, "\n"))
	if m.autoScroll {
		m.viewport.GotoBottom()
	}
}

// LineCount returns the number of log lines.
func (m LogViewModel) LineCount() int {
	return len(m.lines)
}

// Init implements tea.Model.
func (m LogViewModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m LogViewModel) Update(msg tea.Msg) (LogViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.LogLineMsg:
		m.AppendLine(msg.Line)
		return m, nil
	case tea.KeyPressMsg:
		if m.focused {
			return m.updateKeys(msg)
		}
	}
	return m, nil
}

func (m LogViewModel) updateKeys(msg tea.KeyPressMsg) (LogViewModel, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		m.autoScroll = false
		m.viewport.ScrollDown(1)
	case "k", "up":
		m.autoScroll = false
		m.viewport.ScrollUp(1)
	case "G":
		m.autoScroll = true
		m.viewport.GotoBottom()
	case "g":
		m.autoScroll = false
		m.viewport.GotoTop()
	}
	return m, nil
}

// View renders the log viewport.
func (m LogViewModel) View() string {
	if len(m.lines) == 0 {
		return "No logs"
	}
	return m.viewport.View()
}
