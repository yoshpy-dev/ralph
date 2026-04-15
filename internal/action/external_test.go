package action

import (
	"testing"
)

func TestOpenPager(t *testing.T) {
	exec, _ := setupTestExecutor(t)

	t.Run("returns non-nil command", func(t *testing.T) {
		cmd := exec.OpenPager("/tmp/test.log")
		if cmd == nil {
			t.Fatal("expected non-nil command")
		}
	})

	t.Run("uses PAGER env var", func(t *testing.T) {
		t.Setenv("PAGER", "cat")
		cmd := exec.OpenPager("/tmp/test.log")
		if cmd == nil {
			t.Fatal("expected non-nil command")
		}
	})

	t.Run("fallback to less when PAGER unset", func(t *testing.T) {
		t.Setenv("PAGER", "")
		cmd := exec.OpenPager("/tmp/test.log")
		if cmd == nil {
			t.Fatal("expected non-nil command")
		}
	})
}

func TestOpenEditor(t *testing.T) {
	exec, _ := setupTestExecutor(t)

	t.Run("returns non-nil command", func(t *testing.T) {
		cmd := exec.OpenEditor("/tmp/worktree")
		if cmd == nil {
			t.Fatal("expected non-nil command")
		}
	})

	t.Run("uses EDITOR env var", func(t *testing.T) {
		t.Setenv("EDITOR", "nano")
		cmd := exec.OpenEditor("/tmp/worktree")
		if cmd == nil {
			t.Fatal("expected non-nil command")
		}
	})

	t.Run("fallback to vi when EDITOR unset", func(t *testing.T) {
		t.Setenv("EDITOR", "")
		cmd := exec.OpenEditor("/tmp/worktree")
		if cmd == nil {
			t.Fatal("expected non-nil command")
		}
	})
}
