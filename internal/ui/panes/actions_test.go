package panes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/action"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/state"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/ui"
)

func makeKeyPress(key string) tea.KeyPressMsg {
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

func TestActionsModel_NoSliceSelected(t *testing.T) {
	m := NewActionsModel(nil)
	view := m.View()
	if !strings.Contains(view, "no slice selected") {
		t.Errorf("expected 'no slice selected' in view, got: %s", view)
	}
}

func TestActionsModel_HandleKey_NoSlice(t *testing.T) {
	m := NewActionsModel(nil)
	_, req, consumed := m.HandleKey(makeKeyPress("r"))
	if consumed {
		t.Error("key should not be consumed when no slice is selected")
	}
	if req != nil {
		t.Error("no confirm request should be generated")
	}
}

func TestActionsModel_FailedSliceActions(t *testing.T) {
	m := NewActionsModel(nil)
	failedSlice := state.SliceState{
		Name:         "slice-1",
		Status:       state.StatusFailed,
		LogPath:      "/tmp/logs/slice-1.log",
		WorktreePath: "/tmp/worktrees/slice-1",
	}
	m, _ = m.Update(ui.SliceSelectedMsg{Slice: failedSlice})

	t.Run("retry available", func(t *testing.T) {
		_, req, consumed := m.HandleKey(makeKeyPress("r"))
		if !consumed {
			t.Error("r should be consumed for failed slice")
		}
		if req == nil {
			t.Fatal("expected confirmation request for retry")
		}
		if !strings.Contains(req.Message, "Retry") {
			t.Errorf("confirm message should mention Retry, got: %s", req.Message)
		}
		if req.Tag != "retry:slice-1" {
			t.Errorf("tag = %q, want %q", req.Tag, "retry:slice-1")
		}
	})

	t.Run("abort available", func(t *testing.T) {
		_, req, consumed := m.HandleKey(makeKeyPress("a"))
		if !consumed {
			t.Error("a should be consumed for failed slice")
		}
		if req == nil {
			t.Fatal("expected confirmation request for abort")
		}
		if !strings.Contains(req.Tag, "abort:") {
			t.Errorf("tag should start with abort:, got: %s", req.Tag)
		}
	})

	t.Run("abort all available", func(t *testing.T) {
		_, req, consumed := m.HandleKey(makeKeyPress("A"))
		if !consumed {
			t.Error("A should be consumed")
		}
		if req == nil {
			t.Fatal("expected confirmation request for abort all")
		}
		if req.Tag != "abort-all" {
			t.Errorf("tag = %q, want %q", req.Tag, "abort-all")
		}
	})

	t.Run("logs available", func(t *testing.T) {
		cmd, req, consumed := m.HandleKey(makeKeyPress("L"))
		if !consumed {
			t.Error("L should be consumed")
		}
		if req != nil {
			t.Error("logs should not need confirmation")
		}
		if cmd == nil {
			t.Error("expected a command for opening pager")
		}
	})

	t.Run("editor available", func(t *testing.T) {
		cmd, req, consumed := m.HandleKey(makeKeyPress("e"))
		if !consumed {
			t.Error("e should be consumed")
		}
		if req != nil {
			t.Error("editor should not need confirmation")
		}
		if cmd == nil {
			t.Error("expected a command for opening editor")
		}
	})
}

func TestActionsModel_RunningSliceActions(t *testing.T) {
	m := NewActionsModel(nil)
	runningSlice := state.SliceState{
		Name:         "slice-2",
		Status:       state.StatusRunning,
		LogPath:      "/tmp/logs/slice-2.log",
		WorktreePath: "/tmp/worktrees/slice-2",
	}
	m, _ = m.Update(ui.SliceSelectedMsg{Slice: runningSlice})

	t.Run("retry disabled for running", func(t *testing.T) {
		_, req, consumed := m.HandleKey(makeKeyPress("r"))
		if !consumed {
			t.Error("r should be consumed (disabled but still consumed)")
		}
		if req != nil {
			t.Error("retry should be disabled for running slice")
		}
	})

	t.Run("abort available for running", func(t *testing.T) {
		_, req, consumed := m.HandleKey(makeKeyPress("a"))
		if !consumed {
			t.Error("a should be consumed")
		}
		if req == nil {
			t.Error("abort should be available for running slice")
		}
	})
}

func TestActionsModel_CompleteSliceActions(t *testing.T) {
	m := NewActionsModel(nil)
	completeSlice := &state.SliceState{
		Name:         "slice-3",
		Status:       state.StatusComplete,
		LogPath:      "/tmp/logs/slice-3.log",
		WorktreePath: "/tmp/worktrees/slice-3",
	}
	m, _ = m.Update(ui.SliceSelectedMsg{Slice: completeSlice})

	t.Run("retry disabled for complete", func(t *testing.T) {
		_, req, consumed := m.HandleKey(makeKeyPress("r"))
		if !consumed {
			t.Error("r should be consumed")
		}
		if req != nil {
			t.Error("retry should be disabled for complete slice")
		}
	})

	t.Run("abort disabled for complete", func(t *testing.T) {
		_, req, consumed := m.HandleKey(makeKeyPress("a"))
		if !consumed {
			t.Error("a should be consumed")
		}
		if req != nil {
			t.Error("abort should be disabled for complete slice")
		}
	})
}

func TestActionsModel_PendingSliceActions(t *testing.T) {
	m := NewActionsModel(nil)
	pendingSlice := &state.SliceState{
		Name:   "slice-4",
		Status: state.StatusPending,
	}
	m, _ = m.Update(ui.SliceSelectedMsg{Slice: pendingSlice})

	t.Run("retry disabled for pending", func(t *testing.T) {
		_, req, consumed := m.HandleKey(makeKeyPress("r"))
		if !consumed {
			t.Error("r should be consumed")
		}
		if req != nil {
			t.Error("retry should be disabled for pending slice")
		}
	})

	t.Run("logs unavailable when no log path", func(t *testing.T) {
		cmd, req, consumed := m.HandleKey(makeKeyPress("L"))
		if !consumed {
			t.Error("L should be consumed")
		}
		if req != nil {
			t.Error("no confirm request expected")
		}
		if cmd != nil {
			t.Error("no command expected when no log path")
		}
	})

	t.Run("editor unavailable when no worktree path", func(t *testing.T) {
		cmd, req, consumed := m.HandleKey(makeKeyPress("e"))
		if !consumed {
			t.Error("e should be consumed")
		}
		if req != nil {
			t.Error("no confirm request expected")
		}
		if cmd != nil {
			t.Error("no command expected when no worktree path")
		}
	})
}

func TestActionsModel_StuckSliceActions(t *testing.T) {
	m := NewActionsModel(nil)
	stuckSlice := &state.SliceState{
		Name:         "slice-5",
		Status:       state.StatusStuck,
		LogPath:      "/tmp/logs/slice-5.log",
		WorktreePath: "/tmp/worktrees/slice-5",
	}
	m, _ = m.Update(ui.SliceSelectedMsg{Slice: stuckSlice})

	t.Run("retry available for stuck", func(t *testing.T) {
		_, req, consumed := m.HandleKey(makeKeyPress("r"))
		if !consumed {
			t.Error("r should be consumed")
		}
		if req == nil {
			t.Error("retry should be available for stuck slice")
		}
	})
}

func TestActionsModel_UnknownKey(t *testing.T) {
	m := NewActionsModel(nil)
	m, _ = m.Update(ui.SliceSelectedMsg{Slice: state.SliceState{Name: "s", Status: state.StatusFailed}})
	_, _, consumed := m.HandleKey(makeKeyPress("x"))
	if consumed {
		t.Error("unknown key should not be consumed")
	}
}

func TestActionsModel_StatusMessages(t *testing.T) {
	m := NewActionsModel(nil)

	t.Run("retry success", func(t *testing.T) {
		m, _ = m.Update(action.RetryResultMsg{SliceName: "s1"})
		if !strings.Contains(m.View(), "Retry started") {
			t.Error("view should show retry success")
		}
	})

	t.Run("retry failure", func(t *testing.T) {
		m, _ = m.Update(action.RetryResultMsg{SliceName: "s1", Err: errTest})
		if !strings.Contains(m.View(), "Retry failed") {
			t.Error("view should show retry failure")
		}
	})

	t.Run("abort success", func(t *testing.T) {
		m, _ = m.Update(action.AbortResultMsg{SliceName: "s2"})
		if !strings.Contains(m.View(), "Abort sent") {
			t.Error("view should show abort success")
		}
	})

	t.Run("abort all success", func(t *testing.T) {
		m, _ = m.Update(action.AbortResultMsg{})
		if !strings.Contains(m.View(), "all slices") {
			t.Error("view should mention all slices")
		}
	})

	t.Run("abort failure", func(t *testing.T) {
		m, _ = m.Update(action.AbortResultMsg{SliceName: "s3", Err: errTest})
		if !strings.Contains(m.View(), "Abort failed") {
			t.Error("view should show abort failure")
		}
	})

	t.Run("external done error", func(t *testing.T) {
		m, _ = m.Update(action.ExternalDoneMsg{Action: "pager", Err: errTest})
		if !strings.Contains(m.View(), "pager error") {
			t.Error("view should show pager error")
		}
	})

	t.Run("external done success clears status", func(t *testing.T) {
		m, _ = m.Update(action.ExternalDoneMsg{Action: "editor"})
		view := m.View()
		if strings.Contains(view, "error") {
			t.Errorf("view should not show error after success, got: %s", view)
		}
	})
}

func TestActionsModel_ExecuteConfirmed(t *testing.T) {
	m := NewActionsModel(nil)

	t.Run("no executor", func(t *testing.T) {
		cmd := m.ExecuteConfirmed("retry:slice-1")
		if cmd != nil {
			t.Error("should return nil with no executor")
		}
	})

	t.Run("unknown tag", func(t *testing.T) {
		cmd := m.ExecuteConfirmed("unknown:thing")
		if cmd != nil {
			t.Error("should return nil for unknown tag prefix")
		}
	})

	t.Run("retry tag missing slice name", func(t *testing.T) {
		cmd := m.ExecuteConfirmed("retry")
		if cmd != nil {
			t.Error("should return nil for retry without slice name")
		}
	})

	t.Run("abort tag missing slice name", func(t *testing.T) {
		cmd := m.ExecuteConfirmed("abort")
		if cmd != nil {
			t.Error("should return nil for abort without slice name")
		}
	})
}

func TestActionsModel_View_StyledActions(t *testing.T) {
	m := NewActionsModel(nil)
	failedSlice := state.SliceState{
		Name:         "slice-1",
		Status:       state.StatusFailed,
		LogPath:      "/path/to/log",
		WorktreePath: "/path/to/worktree",
	}
	m, _ = m.Update(ui.SliceSelectedMsg{Slice: failedSlice})
	view := m.View()

	// All action keys should be present
	for _, key := range []string{"r", "a", "A", "L", "e"} {
		if !strings.Contains(view, "["+key+"]") {
			t.Errorf("view missing action key [%s], got: %s", key, view)
		}
	}
	if !strings.Contains(view, "Retry") {
		t.Error("view missing Retry label")
	}
	if !strings.Contains(view, "Abort") {
		t.Error("view missing Abort label")
	}
}

func TestActionsModel_SetSize(t *testing.T) {
	m := NewActionsModel(nil)
	m = m.SetSize(80, 24)
	// Just verify it doesn't panic and returns a model
	if m.width != 80 || m.height != 24 {
		t.Errorf("size = (%d, %d), want (80, 24)", m.width, m.height)
	}
}

func TestActionsModel_SetFocused(t *testing.T) {
	m := NewActionsModel(nil)
	m = m.SetFocused(true)
	if !m.focused {
		t.Error("expected focused=true")
	}
	m = m.SetFocused(false)
	if m.focused {
		t.Error("expected focused=false")
	}
}

func TestActionsModel_ExecuteConfirmed_WithExecutor(t *testing.T) {
	exec, _ := setupTestExecutor(t)
	m := NewActionsModel(exec)

	t.Run("retry executes", func(t *testing.T) {
		cmd := m.ExecuteConfirmed("retry:slice-1")
		if cmd == nil {
			t.Fatal("expected non-nil command for retry")
		}
		msg := cmd()
		result, ok := msg.(action.RetryResultMsg)
		if !ok {
			t.Fatalf("expected RetryResultMsg, got %T", msg)
		}
		if result.SliceName != "slice-1" {
			t.Errorf("SliceName = %q, want %q", result.SliceName, "slice-1")
		}
	})

	t.Run("abort slice executes", func(t *testing.T) {
		cmd := m.ExecuteConfirmed("abort:slice-2")
		if cmd == nil {
			t.Fatal("expected non-nil command for abort")
		}
		msg := cmd()
		result, ok := msg.(action.AbortResultMsg)
		if !ok {
			t.Fatalf("expected AbortResultMsg, got %T", msg)
		}
		if result.SliceName != "slice-2" {
			t.Errorf("SliceName = %q, want %q", result.SliceName, "slice-2")
		}
	})

	t.Run("abort-all executes", func(t *testing.T) {
		cmd := m.ExecuteConfirmed("abort-all")
		if cmd == nil {
			t.Fatal("expected non-nil command for abort-all")
		}
		msg := cmd()
		result, ok := msg.(action.AbortResultMsg)
		if !ok {
			t.Fatalf("expected AbortResultMsg, got %T", msg)
		}
		if result.SliceName != "" {
			t.Errorf("SliceName = %q, want empty", result.SliceName)
		}
	})
}

func TestActionsModel_StatusMsg(t *testing.T) {
	m := NewActionsModel(nil)
	m, _ = m.Update(ui.StatusMsg{Text: "custom status", IsError: false})
	if m.statusText != "custom status" {
		t.Errorf("statusText = %q, want %q", m.statusText, "custom status")
	}
	if m.statusIsError {
		t.Error("expected statusIsError=false")
	}
}

func setupTestExecutor(t *testing.T) (*action.Executor, string) {
	t.Helper()
	dir := t.TempDir()
	ralphPath := filepath.Join(dir, "scripts", "ralph")
	if err := os.MkdirAll(filepath.Dir(ralphPath), 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\necho \"$@\"\n"
	if err := os.WriteFile(ralphPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	exec, err := action.NewExecutor(dir)
	if err != nil {
		t.Fatal(err)
	}
	return exec, dir
}

var errTest = errForTest("test error")

type errForTest string

func (e errForTest) Error() string { return string(e) }
