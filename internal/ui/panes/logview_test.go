package panes

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/ui"
)

func TestLogViewSetContent(t *testing.T) {
	m := NewLogView(60, 10)

	m.SetContent("line 1\nline 2\nline 3")
	view := m.View()

	if !strings.Contains(view, "line 1") {
		t.Error("expected 'line 1' in view")
	}
	if !strings.Contains(view, "line 3") {
		t.Error("expected 'line 3' in view")
	}
}

func TestLogViewAppendLine(t *testing.T) {
	m := NewLogView(60, 10)

	m.AppendLine("first line")
	m.AppendLine("second line")

	if m.LineCount() != 2 {
		t.Fatalf("expected 2 lines, got %d", m.LineCount())
	}
}

func TestLogViewANSIStripping(t *testing.T) {
	m := NewLogView(60, 10)

	m.SetContent("\x1b[31mred text\x1b[0m")
	view := m.View()

	if strings.Contains(view, "\x1b[") {
		t.Error("expected ANSI sequences to be stripped")
	}
	if !strings.Contains(view, "red text") {
		t.Error("expected 'red text' content to remain")
	}
}

func TestLogViewAppendLineANSI(t *testing.T) {
	m := NewLogView(60, 10)
	m.AppendLine("\x1b[32mgreen\x1b[0m")

	if m.LineCount() != 1 {
		t.Fatalf("expected 1 line, got %d", m.LineCount())
	}
}

func TestLogViewEmpty(t *testing.T) {
	m := NewLogView(60, 10)
	view := m.View()

	if view != "No logs" {
		t.Fatalf("expected 'No logs', got %q", view)
	}
}

func TestLogViewLogLineMsg(t *testing.T) {
	m := NewLogView(60, 10)
	m.SetFocused(true)

	m, _ = m.Update(ui.LogLineMsg{Line: "new log entry"})

	if m.LineCount() != 1 {
		t.Fatalf("expected 1 line after LogLineMsg, got %d", m.LineCount())
	}
}

func TestLogViewScrollJK(t *testing.T) {
	m := NewLogView(60, 5)
	m.SetFocused(true)

	// Add many lines so scrolling is meaningful.
	var lines []string
	for i := range 20 {
		lines = append(lines, strings.Repeat("x", i+1))
	}
	m.SetContent(strings.Join(lines, "\n"))

	// Scroll up should not panic
	m, _ = m.Update(tea.KeyPressMsg{Code: 'k'})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})

	view := m.View()
	if len(view) == 0 {
		t.Fatal("expected non-empty view after scrolling")
	}
}

func TestLogViewGoToTopBottom(t *testing.T) {
	m := NewLogView(60, 5)
	m.SetFocused(true)

	var lines []string
	for range 20 {
		lines = append(lines, "line")
	}
	m.SetContent(strings.Join(lines, "\n"))

	// Go to top
	m, _ = m.Update(tea.KeyPressMsg{Code: 'g'})
	// Go to bottom
	m, _ = m.Update(tea.KeyPressMsg{Code: 'G'})

	// Should not panic
	view := m.View()
	if len(view) == 0 {
		t.Fatal("expected non-empty view")
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"plain text", "plain text"},
		{"\x1b[31mred\x1b[0m", "red"},
		{"\x1b[1;32mbold green\x1b[0m", "bold green"},
		{"no escape", "no escape"},
		{"", ""},
	}

	for _, tt := range tests {
		result := StripANSI(tt.input)
		if result != tt.expected {
			t.Errorf("StripANSI(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestLogViewUnfocusedIgnoresKeys(t *testing.T) {
	m := NewLogView(60, 10)
	// Not focused
	m.SetContent("line 1\nline 2")

	m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	// Should not panic
	view := m.View()
	if len(view) == 0 {
		t.Fatal("expected non-empty view")
	}
}
