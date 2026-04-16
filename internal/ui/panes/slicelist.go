package panes

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/state"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/ui"
)

// SliceListModel displays a navigable list of slices with status icons.
type SliceListModel struct {
	slices   []state.SliceState
	filtered []int // indices into slices
	cursor   int
	width    int
	height   int
	focused  bool

	filtering  bool
	filterText string
}

// NewSliceList creates a new SliceListModel.
func NewSliceList(slices []state.SliceState, width, height int) SliceListModel {
	m := SliceListModel{
		slices: slices,
		width:  width,
		height: height,
	}
	m.rebuildFiltered()
	return m
}

// SetSlices updates the slice list.
func (m *SliceListModel) SetSlices(slices []state.SliceState) {
	m.slices = slices
	m.rebuildFiltered()
	if m.cursor >= len(m.filtered) && len(m.filtered) > 0 {
		m.cursor = len(m.filtered) - 1
	}
}

// SetSize updates the pane dimensions.
func (m *SliceListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetFocused sets the focus state.
func (m *SliceListModel) SetFocused(focused bool) {
	m.focused = focused
	if !focused && m.filtering {
		m.filtering = false
	}
}

// SelectedSlice returns the currently selected slice, if any.
func (m SliceListModel) SelectedSlice() (state.SliceState, bool) {
	if len(m.filtered) == 0 {
		return state.SliceState{}, false
	}
	idx := m.filtered[m.cursor]
	return m.slices[idx], true
}

// Cursor returns the current cursor position.
func (m SliceListModel) Cursor() int {
	return m.cursor
}

// FilteredCount returns the number of visible (filtered) items.
func (m SliceListModel) FilteredCount() int {
	return len(m.filtered)
}

// Init implements tea.Model.
func (m SliceListModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m SliceListModel) Update(msg tea.Msg) (SliceListModel, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.filtering {
			return m.updateFilter(msg)
		}
		return m.updateNormal(msg)
	}

	return m, nil
}

func (m SliceListModel) updateNormal(msg tea.KeyPressMsg) (SliceListModel, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			return m, m.selectedCmd()
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			return m, m.selectedCmd()
		}
	case "/":
		m.filtering = true
		m.filterText = ""
		return m, nil
	}
	return m, nil
}

func (m SliceListModel) updateFilter(msg tea.KeyPressMsg) (SliceListModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.filtering = false
		return m, m.selectedCmd()
	case "esc":
		m.filtering = false
		m.filterText = ""
		m.rebuildFiltered()
		m.cursor = 0
		return m, m.selectedCmd()
	case "backspace":
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
			m.rebuildFiltered()
			m.cursor = 0
		}
		return m, nil
	default:
		// Only add printable single characters.
		key := msg.String()
		if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
			m.filterText += key
			m.rebuildFiltered()
			m.cursor = 0
		}
		return m, nil
	}
}

func (m *SliceListModel) rebuildFiltered() {
	m.filtered = m.filtered[:0]
	query := strings.ToLower(m.filterText)
	for i, s := range m.slices {
		if query == "" || strings.Contains(strings.ToLower(s.Name), query) {
			m.filtered = append(m.filtered, i)
		}
	}
}

func (m SliceListModel) selectedCmd() tea.Cmd {
	if s, ok := m.SelectedSlice(); ok {
		return func() tea.Msg {
			return ui.SliceSelectedMsg{Slice: s}
		}
	}
	return nil
}

// View renders the slice list.
func (m SliceListModel) View() string {
	if len(m.slices) == 0 {
		return "No slices"
	}

	var b strings.Builder

	if m.filtering {
		fmt.Fprintf(&b, "/%s\n", m.filterText)
	}

	// Calculate visible window.
	contentHeight := m.height
	if m.filtering {
		contentHeight--
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	start := 0
	if m.cursor >= contentHeight {
		start = m.cursor - contentHeight + 1
	}
	end := start + contentHeight
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := start; i < end; i++ {
		idx := m.filtered[i]
		s := m.slices[idx]

		icon := ui.StatusIcon(string(s.Status))
		clr := ui.StatusColor(string(s.Status))
		iconStyled := lipgloss.NewStyle().Foreground(clr).Render(icon)

		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		name := s.Name
		maxNameLen := m.width - 6 // cursor(2) + icon(1) + space(1) + padding(2)
		if maxNameLen > 0 && len(name) > maxNameLen {
			name = name[:maxNameLen-1] + "~"
		}

		line := fmt.Sprintf("%s%s %s", cursor, iconStyled, name)
		if i < end-1 {
			b.WriteString(line + "\n")
		} else {
			b.WriteString(line)
		}
	}

	return b.String()
}
