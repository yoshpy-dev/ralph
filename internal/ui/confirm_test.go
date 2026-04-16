package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestConfirmModel_Show(t *testing.T) {
	m := NewConfirmModel()
	if m.Visible {
		t.Error("new model should not be visible")
	}

	m = m.Show("Delete everything?", "delete-all")
	if !m.Visible {
		t.Error("model should be visible after Show")
	}
	if m.Message != "Delete everything?" {
		t.Errorf("Message = %q, want %q", m.Message, "Delete everything?")
	}
	if m.Tag != "delete-all" {
		t.Errorf("Tag = %q, want %q", m.Tag, "delete-all")
	}
}

func TestConfirmModel_Hide(t *testing.T) {
	m := NewConfirmModel().Show("msg", "tag")
	m = m.Hide()
	if m.Visible {
		t.Error("model should not be visible after Hide")
	}
	if m.Message != "" {
		t.Errorf("Message should be empty after Hide, got %q", m.Message)
	}
	if m.Tag != "" {
		t.Errorf("Tag should be empty after Hide, got %q", m.Tag)
	}
}

func TestConfirmModel_UpdateYes(t *testing.T) {
	keys := []string{"y", "Y", "enter"}

	for _, key := range keys {
		t.Run("key_"+key, func(t *testing.T) {
			m := NewConfirmModel().Show("Confirm?", "test-tag")
			keyMsg := makeKeyPressMsg(key)
			updated, cmd := m.Update(keyMsg)
			if updated.Visible {
				t.Error("dialog should be hidden after yes")
			}
			if cmd == nil {
				t.Fatal("expected a command for yes response")
			}
			msg := cmd()
			yes, ok := msg.(ConfirmYesMsg)
			if !ok {
				t.Fatalf("expected ConfirmYesMsg, got %T", msg)
			}
			if yes.Tag != "test-tag" {
				t.Errorf("Tag = %q, want %q", yes.Tag, "test-tag")
			}
		})
	}
}

func TestConfirmModel_UpdateNo(t *testing.T) {
	keys := []string{"n", "N", "esc"}

	for _, key := range keys {
		t.Run("key_"+key, func(t *testing.T) {
			m := NewConfirmModel().Show("Confirm?", "test-tag")
			keyMsg := makeKeyPressMsg(key)
			updated, cmd := m.Update(keyMsg)
			if updated.Visible {
				t.Error("dialog should be hidden after no")
			}
			if cmd == nil {
				t.Fatal("expected a command for no response")
			}
			msg := cmd()
			no, ok := msg.(ConfirmNoMsg)
			if !ok {
				t.Fatalf("expected ConfirmNoMsg, got %T", msg)
			}
			if no.Tag != "test-tag" {
				t.Errorf("Tag = %q, want %q", no.Tag, "test-tag")
			}
		})
	}
}

func TestConfirmModel_UpdateIgnoredWhenHidden(t *testing.T) {
	m := NewConfirmModel() // hidden
	keyMsg := makeKeyPressMsg("y")
	updated, cmd := m.Update(keyMsg)
	if updated.Visible {
		t.Error("hidden dialog should stay hidden")
	}
	if cmd != nil {
		t.Error("hidden dialog should not produce commands")
	}
}

func TestConfirmModel_UpdateUnknownKey(t *testing.T) {
	m := NewConfirmModel().Show("Confirm?", "tag")
	keyMsg := makeKeyPressMsg("x")
	updated, cmd := m.Update(keyMsg)
	if !updated.Visible {
		t.Error("unknown key should not dismiss dialog")
	}
	if cmd != nil {
		t.Error("unknown key should not produce commands")
	}
}

func TestConfirmModel_View(t *testing.T) {
	t.Run("hidden", func(t *testing.T) {
		m := NewConfirmModel()
		view := m.View()
		if view != "" {
			t.Errorf("hidden dialog View should be empty, got %q", view)
		}
	})

	t.Run("visible", func(t *testing.T) {
		m := NewConfirmModel().Show("Delete everything?", "del")
		view := m.View()
		if !strings.Contains(view, "Delete everything?") {
			t.Errorf("View should contain message, got %q", view)
		}
		if !strings.Contains(view, "Yes") {
			t.Errorf("View should contain Yes hint, got %q", view)
		}
		if !strings.Contains(view, "Cancel") {
			t.Errorf("View should contain Cancel hint, got %q", view)
		}
	})
}

// makeKeyPressMsg creates a KeyPressMsg for testing.
func makeKeyPressMsg(key string) tea.KeyPressMsg {
	switch key {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	default:
		r := []rune(key)
		if len(r) == 1 {
			return tea.KeyPressMsg{Code: r[0], Text: key}
		}
		return tea.KeyPressMsg{}
	}
}
