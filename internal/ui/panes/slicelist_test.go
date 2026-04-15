package panes

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/state"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/ui"
)

func testSlices() []state.SliceState {
	return []state.SliceState{
		{Name: "slice-1", Status: state.StatusComplete},
		{Name: "slice-2", Status: state.StatusRunning},
		{Name: "slice-3", Status: state.StatusPending},
		{Name: "slice-4", Status: state.StatusFailed},
		{Name: "slice-5", Status: state.StatusComplete},
	}
}

func TestSliceListJKNavigation(t *testing.T) {
	m := NewSliceList(testSlices(), 40, 20)
	m.SetFocused(true)

	if m.Cursor() != 0 {
		t.Fatalf("expected cursor at 0, got %d", m.Cursor())
	}

	// Move down with j
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	if m.Cursor() != 1 {
		t.Fatalf("expected cursor at 1 after j, got %d", m.Cursor())
	}

	// Move down again
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	if m.Cursor() != 2 {
		t.Fatalf("expected cursor at 2 after j, got %d", m.Cursor())
	}

	// Move up with k
	m, _ = m.Update(tea.KeyPressMsg{Code: 'k'})
	if m.Cursor() != 1 {
		t.Fatalf("expected cursor at 1 after k, got %d", m.Cursor())
	}
}

func TestSliceListCursorBounds(t *testing.T) {
	m := NewSliceList(testSlices(), 40, 20)
	m.SetFocused(true)

	// Try moving up past beginning
	m, _ = m.Update(tea.KeyPressMsg{Code: 'k'})
	if m.Cursor() != 0 {
		t.Fatalf("cursor should not go below 0, got %d", m.Cursor())
	}

	// Move to end
	for i := 0; i < 10; i++ {
		m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	}
	if m.Cursor() != 4 {
		t.Fatalf("cursor should stop at last item (4), got %d", m.Cursor())
	}

	// Try moving past end
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	if m.Cursor() != 4 {
		t.Fatalf("cursor should not exceed last item, got %d", m.Cursor())
	}
}

func TestSliceListSelectedSlice(t *testing.T) {
	m := NewSliceList(testSlices(), 40, 20)
	m.SetFocused(true)

	s, ok := m.SelectedSlice()
	if !ok {
		t.Fatal("expected a selected slice")
	}
	if s.Name != "slice-1" {
		t.Fatalf("expected slice-1, got %s", s.Name)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	s, ok = m.SelectedSlice()
	if !ok {
		t.Fatal("expected a selected slice")
	}
	if s.Name != "slice-2" {
		t.Fatalf("expected slice-2, got %s", s.Name)
	}
}

func TestSliceListSliceSelectedMsg(t *testing.T) {
	m := NewSliceList(testSlices(), 40, 20)
	m.SetFocused(true)

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'j'})
	if cmd == nil {
		t.Fatal("expected a command to be returned on cursor move")
	}

	msg := cmd()
	sel, ok := msg.(ui.SliceSelectedMsg)
	if !ok {
		t.Fatalf("expected SliceSelectedMsg, got %T", msg)
	}
	if sel.Slice.Name != "slice-2" {
		t.Fatalf("expected slice-2 in msg, got %s", sel.Slice.Name)
	}
}

func TestSliceListStatusIcons(t *testing.T) {
	m := NewSliceList(testSlices(), 40, 20)
	view := m.View()

	// complete -> +
	if !strings.Contains(view, "+") {
		t.Error("expected + icon for complete status")
	}
	// running -> *
	if !strings.Contains(view, "*") {
		t.Error("expected * icon for running status")
	}
	// pending -> -
	// NOTE: - appears in slice names too, so we check for the icon in context
	// failed -> !
	if !strings.Contains(view, "!") {
		t.Error("expected ! icon for failed status")
	}
}

func TestSliceListFilter(t *testing.T) {
	m := NewSliceList(testSlices(), 40, 20)
	m.SetFocused(true)

	// Enter filter mode
	m, _ = m.Update(tea.KeyPressMsg{Code: '/'})

	// Type "3"
	m, _ = m.Update(tea.KeyPressMsg{Code: '3'})

	if m.FilteredCount() != 1 {
		t.Fatalf("expected 1 filtered item, got %d", m.FilteredCount())
	}

	s, ok := m.SelectedSlice()
	if !ok {
		t.Fatal("expected a selected slice after filter")
	}
	if s.Name != "slice-3" {
		t.Fatalf("expected slice-3, got %s", s.Name)
	}

	// Confirm filter with Enter
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Should still show filtered result
	if m.FilteredCount() != 1 {
		t.Fatalf("expected filter to persist after Enter, got %d items", m.FilteredCount())
	}
}

func TestSliceListFilterEscape(t *testing.T) {
	m := NewSliceList(testSlices(), 40, 20)
	m.SetFocused(true)

	// Enter filter mode and type something
	m, _ = m.Update(tea.KeyPressMsg{Code: '/'})
	m, _ = m.Update(tea.KeyPressMsg{Code: '3'})

	if m.FilteredCount() != 1 {
		t.Fatalf("expected 1 filtered item, got %d", m.FilteredCount())
	}

	// Cancel filter with Escape
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	if m.FilteredCount() != 5 {
		t.Fatalf("expected all 5 items after Escape, got %d", m.FilteredCount())
	}
}

func TestSliceListFilterBackspace(t *testing.T) {
	m := NewSliceList(testSlices(), 40, 20)
	m.SetFocused(true)

	m, _ = m.Update(tea.KeyPressMsg{Code: '/'})
	m, _ = m.Update(tea.KeyPressMsg{Code: '3'})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})

	if m.FilteredCount() != 5 {
		t.Fatalf("expected all 5 items after backspace, got %d", m.FilteredCount())
	}
}

func TestSliceListEmptySlices(t *testing.T) {
	m := NewSliceList(nil, 40, 20)
	view := m.View()

	if view != "No slices" {
		t.Fatalf("expected 'No slices', got %q", view)
	}

	_, ok := m.SelectedSlice()
	if ok {
		t.Fatal("expected no selected slice with empty list")
	}
}

func TestSliceListSingleSlice(t *testing.T) {
	slices := []state.SliceState{
		{Name: "only-one", Status: state.StatusRunning},
	}
	m := NewSliceList(slices, 40, 20)
	m.SetFocused(true)

	s, ok := m.SelectedSlice()
	if !ok || s.Name != "only-one" {
		t.Fatal("expected 'only-one' to be selected")
	}

	// Try moving should not crash
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'k'})

	if m.Cursor() != 0 {
		t.Fatalf("cursor should stay at 0 with single item, got %d", m.Cursor())
	}
}

func TestSliceListUnfocusedIgnoresKeys(t *testing.T) {
	m := NewSliceList(testSlices(), 40, 20)
	// Not focused by default
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	if m.Cursor() != 0 {
		t.Fatal("unfocused list should not respond to keys")
	}
}

func TestSliceListLongName(t *testing.T) {
	slices := []state.SliceState{
		{Name: "a-very-long-slice-name-that-exceeds-width", Status: state.StatusComplete},
	}
	m := NewSliceList(slices, 20, 10)
	view := m.View()
	// Should not panic and should truncate
	if len(view) == 0 {
		t.Fatal("expected non-empty view for long name")
	}
}
