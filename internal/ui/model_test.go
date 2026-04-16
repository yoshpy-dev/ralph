package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// --- Pane navigation tests ---

func TestNextPane(t *testing.T) {
	tests := []struct {
		input Pane
		want  Pane
	}{
		{PaneSlices, PaneDetail},
		{PaneDetail, PaneDeps},
		{PaneDeps, PaneActions},
		{PaneActions, PaneLogs},
		{PaneLogs, PaneSlices}, // wraps
	}
	for _, tt := range tests {
		got := NextPane(tt.input)
		if got != tt.want {
			t.Errorf("NextPane(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestPrevPane(t *testing.T) {
	tests := []struct {
		input Pane
		want  Pane
	}{
		{PaneSlices, PaneLogs}, // wraps
		{PaneDetail, PaneSlices},
		{PaneDeps, PaneDetail},
		{PaneActions, PaneDeps},
		{PaneLogs, PaneActions},
	}
	for _, tt := range tests {
		got := PrevPane(tt.input)
		if got != tt.want {
			t.Errorf("PrevPane(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestRightPane(t *testing.T) {
	tests := []struct {
		input Pane
		want  Pane
	}{
		{PaneSlices, PaneDetail},
		{PaneDetail, PaneDeps},
		{PaneDeps, PaneDeps}, // edge: stays
		{PaneActions, PaneLogs},
		{PaneLogs, PaneLogs}, // edge: stays
	}
	for _, tt := range tests {
		got := RightPane(tt.input)
		if got != tt.want {
			t.Errorf("RightPane(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLeftPane(t *testing.T) {
	tests := []struct {
		input Pane
		want  Pane
	}{
		{PaneSlices, PaneSlices}, // edge: stays
		{PaneDetail, PaneSlices},
		{PaneDeps, PaneDetail},
		{PaneActions, PaneActions}, // edge: stays
		{PaneLogs, PaneActions},
	}
	for _, tt := range tests {
		got := LeftPane(tt.input)
		if got != tt.want {
			t.Errorf("LeftPane(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestPaneString(t *testing.T) {
	tests := []struct {
		pane Pane
		want string
	}{
		{PaneSlices, "Slices"},
		{PaneDetail, "Detail"},
		{PaneDeps, "Dependencies"},
		{PaneActions, "Actions"},
		{PaneLogs, "Logs"},
		{Pane(99), "Unknown"},
	}
	for _, tt := range tests {
		got := tt.pane.String()
		if got != tt.want {
			t.Errorf("Pane(%d).String() = %q, want %q", tt.pane, got, tt.want)
		}
	}
}

// --- KeyMap tests ---

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()
	bindings := []struct {
		name string
		keys []string
	}{
		{"Quit", km.Quit.Keys()},
		{"Help", km.Help.Keys()},
		{"Left", km.Left.Keys()},
		{"Right", km.Right.Keys()},
		{"Up", km.Up.Keys()},
		{"Down", km.Down.Keys()},
		{"NextPane", km.NextPane.Keys()},
		{"PrevPane", km.PrevPane.Keys()},
		{"Select", km.Select.Keys()},
		{"Retry", km.Retry.Keys()},
		{"Abort", km.Abort.Keys()},
		{"AbortAll", km.AbortAll.Keys()},
		{"ViewLog", km.ViewLog.Keys()},
		{"Editor", km.Editor.Keys()},
		{"FocusDep", km.FocusDep.Keys()},
		{"Search", km.Search.Keys()},
		{"Enter", km.Enter.Keys()},
	}
	for _, b := range bindings {
		if len(b.keys) == 0 {
			t.Errorf("DefaultKeyMap().%s has no keys", b.name)
		}
	}
}

// --- Styles tests ---

func TestPaneStyle(t *testing.T) {
	focused := PaneStyle(true, 20, 10).Render("test")
	unfocused := PaneStyle(false, 20, 10).Render("test")
	if focused == "" {
		t.Error("PaneStyle(focused) rendered empty")
	}
	if unfocused == "" {
		t.Error("PaneStyle(unfocused) rendered empty")
	}
	if focused == unfocused {
		t.Error("Focused and unfocused pane styles should differ")
	}
}

func TestTitleStyle(t *testing.T) {
	r := TitleStyle().Render("Title")
	if r == "" {
		t.Error("TitleStyle rendered empty")
	}
}

func TestStatusStyle(t *testing.T) {
	for _, status := range []string{"complete", "passed", "failed", "error", "running", "pending", ""} {
		r := StatusStyle(status).Render("text")
		if r == "" {
			t.Errorf("StatusStyle(%q) rendered empty", status)
		}
	}
}

func TestHelpOverlayStyle(t *testing.T) {
	r := HelpOverlayStyle().Render("content")
	if r == "" {
		t.Error("HelpOverlayStyle rendered empty")
	}
}

// --- Layout tests ---

func testPanes() PaneContents {
	return PaneContents{
		Slices:  "slice content",
		Detail:  "detail content",
		Deps:    "deps content",
		Actions: "actions content",
		Logs:    "logs content",
	}
}

func TestRenderLayout(t *testing.T) {
	result := RenderLayout(120, 40, testPanes(), PaneSlices, "Progress: 0/0")
	if result == "" {
		t.Error("RenderLayout produced empty output")
	}
	if !strings.Contains(result, "Slices") {
		t.Error("RenderLayout missing Slices title")
	}
	if !strings.Contains(result, "Detail") {
		t.Error("RenderLayout missing Detail title")
	}
	if !strings.Contains(result, "Dependencies") {
		t.Error("RenderLayout missing Dependencies title")
	}
	if !strings.Contains(result, "Actions") {
		t.Error("RenderLayout missing Actions title")
	}
	if !strings.Contains(result, "Logs") {
		t.Error("RenderLayout missing Logs title")
	}
}

func TestRenderLayoutSmallTerminal(t *testing.T) {
	result := RenderLayout(40, 10, testPanes(), PaneSlices, "Progress: 0/0")
	if result == "" {
		t.Error("Compact layout produced empty output")
	}
	// Compact mode shows focused pane name
	if !strings.Contains(result, "Slices") {
		t.Error("Compact layout should show focused pane")
	}
}

func TestRenderLayoutLargeTerminal(t *testing.T) {
	result := RenderLayout(300, 80, testPanes(), PaneDetail, "Progress: 3/6")
	if result == "" {
		t.Error("Large layout produced empty output")
	}
}

func TestRenderLayoutFocusChangesOutput(t *testing.T) {
	panes := testPanes()
	view1 := RenderLayout(120, 40, panes, PaneSlices, "")
	view2 := RenderLayout(120, 40, panes, PaneLogs, "")
	if view1 == view2 {
		t.Error("Different focused panes should produce different output")
	}
}

func TestRenderLayoutCompactAllPanes(t *testing.T) {
	// Test compact mode for each focused pane to ensure content is shown.
	for _, p := range []Pane{PaneSlices, PaneDetail, PaneDeps, PaneActions, PaneLogs} {
		result := RenderLayout(40, 10, testPanes(), p, "")
		if result == "" {
			t.Errorf("Compact layout empty for focused=%v", p)
		}
	}
}

// --- Help tests ---

func TestRenderHelp(t *testing.T) {
	result := RenderHelp(120, 40)
	if result == "" {
		t.Error("RenderHelp produced empty output")
	}
	if !strings.Contains(result, "Keybindings") {
		t.Error("RenderHelp missing title")
	}
	if !strings.Contains(result, "Quit") {
		t.Error("RenderHelp missing quit entry")
	}
}

// --- Model tests ---

// charKey creates a KeyPressMsg for a single printable character.
func charKey(c rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: c, Text: string(c)}
}

func TestNew(t *testing.T) {
	m := New()
	if m.Focused != PaneSlices {
		t.Errorf("New().Focused = %v, want PaneSlices", m.Focused)
	}
	if m.ShowHelp {
		t.Error("New().ShowHelp should be false")
	}
	if m.Quitting {
		t.Error("New().Quitting should be false")
	}
	if m.Panes.Slices == "" {
		t.Error("New().Panes.Slices should have default text")
	}
}

func TestInit(t *testing.T) {
	m := New()
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil")
	}
}

func TestUpdateWindowSize(t *testing.T) {
	m := New()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	um := updated.(Model)
	if um.Width != 120 || um.Height != 40 {
		t.Errorf("WindowSizeMsg: got (%d, %d), want (120, 40)", um.Width, um.Height)
	}
}

func TestUpdateQuit(t *testing.T) {
	m := New()
	m.Width, m.Height = 120, 40
	updated, cmd := m.Update(charKey('q'))
	um := updated.(Model)
	if !um.Quitting {
		t.Error("'q' should set Quitting to true")
	}
	if cmd == nil {
		t.Error("'q' should return a quit command")
	}
}

func TestUpdateHelpToggle(t *testing.T) {
	m := New()
	m.Width, m.Height = 120, 40

	// Toggle on
	updated, _ := m.Update(charKey('?'))
	um := updated.(Model)
	if !um.ShowHelp {
		t.Error("'?' should toggle help on")
	}

	// Toggle off
	updated, _ = um.Update(charKey('?'))
	um = updated.(Model)
	if um.ShowHelp {
		t.Error("'?' again should toggle help off")
	}
}

func TestUpdatePaneFocusRight(t *testing.T) {
	m := New()
	m.Width, m.Height = 120, 40
	updated, _ := m.Update(charKey('l'))
	um := updated.(Model)
	if um.Focused != PaneDetail {
		t.Errorf("'l' from PaneSlices: got %v, want PaneDetail", um.Focused)
	}
}

func TestUpdatePaneFocusLeft(t *testing.T) {
	m := New()
	m.Width, m.Height = 120, 40
	m.Focused = PaneDetail
	updated, _ := m.Update(charKey('h'))
	um := updated.(Model)
	if um.Focused != PaneSlices {
		t.Errorf("'h' from PaneDetail: got %v, want PaneSlices", um.Focused)
	}
}

func TestUpdateTabCycle(t *testing.T) {
	m := New()
	m.Width, m.Height = 120, 40
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	um := updated.(Model)
	if um.Focused != PaneDetail {
		t.Errorf("Tab from PaneSlices: got %v, want PaneDetail", um.Focused)
	}
}

func TestUpdateShiftTabCycle(t *testing.T) {
	m := New()
	m.Width, m.Height = 120, 40
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	um := updated.(Model)
	if um.Focused != PaneLogs {
		t.Errorf("Shift+Tab from PaneSlices: got %v, want PaneLogs", um.Focused)
	}
}

func TestHelpBlocksOtherKeys(t *testing.T) {
	m := New()
	m.Width, m.Height = 120, 40
	m.ShowHelp = true

	// 'l' should NOT move focus when help is shown.
	updated, _ := m.Update(charKey('l'))
	um := updated.(Model)
	if um.Focused != PaneSlices {
		t.Error("Keys other than ? and q should be blocked in help mode")
	}
	if !um.ShowHelp {
		t.Error("Help should remain open after non-? key")
	}
}

func TestHelpQuitWorks(t *testing.T) {
	m := New()
	m.Width, m.Height = 120, 40
	m.ShowHelp = true

	updated, cmd := m.Update(charKey('q'))
	um := updated.(Model)
	if !um.Quitting {
		t.Error("'q' should work in help mode")
	}
	if cmd == nil {
		t.Error("'q' should return quit command in help mode")
	}
}

func TestViewInitializing(t *testing.T) {
	m := New()
	view := m.View()
	if view.Content != "Initializing..." {
		t.Errorf("View() with zero dimensions = %q, want %q", view.Content, "Initializing...")
	}
}

func TestViewQuitting(t *testing.T) {
	m := New()
	m.Quitting = true
	view := m.View()
	if view.Content != "" {
		t.Errorf("View() when quitting = %q, want empty", view.Content)
	}
}

func TestViewNormal(t *testing.T) {
	m := New()
	m.Width, m.Height = 120, 40
	view := m.View()
	if view.Content == "" {
		t.Error("View() with valid dimensions should not be empty")
	}
	if view.Content == "Initializing..." {
		t.Error("View() should not show Initializing with valid dimensions")
	}
}

func TestViewHelp(t *testing.T) {
	m := New()
	m.Width, m.Height = 120, 40
	m.ShowHelp = true
	view := m.View()
	if !strings.Contains(view.Content, "Keybindings") {
		t.Error("Help view should contain 'Keybindings'")
	}
}

func TestViewFocusHighlight(t *testing.T) {
	m := New()
	m.Width, m.Height = 120, 40

	view1 := m.View()

	m.Focused = PaneDetail
	view2 := m.View()

	if view1.Content == view2.Content {
		t.Error("Focus change should produce different view output")
	}
}

func TestFullTabCycleReturnsToStart(t *testing.T) {
	m := New()
	m.Width, m.Height = 120, 40
	tabMsg := tea.KeyPressMsg{Code: tea.KeyTab}

	// Tab through all panes and back.
	current := m
	for range PaneCount {
		updated, _ := current.Update(tabMsg)
		current = updated.(Model)
	}
	if current.Focused != PaneSlices {
		t.Errorf("Full tab cycle: got %v, want PaneSlices", current.Focused)
	}
}

func TestUnknownMessagePassthrough(t *testing.T) {
	m := New()
	m.Width, m.Height = 120, 40

	// Send an unknown message type.
	type customMsg struct{}
	updated, cmd := m.Update(customMsg{})
	um := updated.(Model)

	// Model should be unchanged.
	if um.Focused != m.Focused {
		t.Error("Unknown message should not change focus")
	}
	if cmd != nil {
		t.Error("Unknown message should return nil cmd")
	}
}
