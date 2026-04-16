package action

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestExecutor(t *testing.T) (*Executor, string) {
	t.Helper()
	dir := t.TempDir()
	ralphPath := filepath.Join(dir, "scripts", "ralph")
	if err := os.MkdirAll(filepath.Dir(ralphPath), 0o755); err != nil {
		t.Fatal(err)
	}
	// Script that prints its arguments for verification
	script := "#!/bin/sh\necho \"$@\"\n"
	if err := os.WriteFile(ralphPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	exec, err := NewExecutor(dir)
	if err != nil {
		t.Fatal(err)
	}
	return exec, dir
}

func TestRetrySlice(t *testing.T) {
	t.Run("valid slice name", func(t *testing.T) {
		exec, _ := setupTestExecutor(t)
		cmd := exec.RetrySlice("slice-1")
		if cmd == nil {
			t.Fatal("expected non-nil command")
		}
		msg := cmd()
		result, ok := msg.(RetryResultMsg)
		if !ok {
			t.Fatalf("expected RetryResultMsg, got %T", msg)
		}
		if result.SliceName != "slice-1" {
			t.Errorf("SliceName = %q, want %q", result.SliceName, "slice-1")
		}
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		// The script should have been called with "retry slice-1"
		expected := "retry slice-1"
		if !strings.Contains(result.Output, expected) {
			t.Errorf("output = %q, want to contain %q", result.Output, expected)
		}
	})

	t.Run("invalid slice name", func(t *testing.T) {
		exec, _ := setupTestExecutor(t)
		cmd := exec.RetrySlice("../../evil")
		msg := cmd()
		result, ok := msg.(RetryResultMsg)
		if !ok {
			t.Fatalf("expected RetryResultMsg, got %T", msg)
		}
		if result.Err == nil {
			t.Error("expected validation error")
		}
		if result.SliceName != "../../evil" {
			t.Errorf("SliceName = %q, want %q", result.SliceName, "../../evil")
		}
	})

	t.Run("empty slice name", func(t *testing.T) {
		exec, _ := setupTestExecutor(t)
		cmd := exec.RetrySlice("")
		msg := cmd()
		result := msg.(RetryResultMsg)
		if result.Err == nil {
			t.Error("expected error for empty name")
		}
	})
}
