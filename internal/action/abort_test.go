package action

import (
	"strings"
	"testing"
)

func TestAbortSlice(t *testing.T) {
	t.Run("valid slice name", func(t *testing.T) {
		exec, _ := setupTestExecutor(t)
		cmd := exec.AbortSlice("slice-2")
		msg := cmd()
		result, ok := msg.(AbortResultMsg)
		if !ok {
			t.Fatalf("expected AbortResultMsg, got %T", msg)
		}
		if result.SliceName != "slice-2" {
			t.Errorf("SliceName = %q, want %q", result.SliceName, "slice-2")
		}
		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		expected := "abort --slice slice-2"
		if !strings.Contains(result.Output, expected) {
			t.Errorf("output = %q, want to contain %q", result.Output, expected)
		}
	})

	t.Run("invalid slice name", func(t *testing.T) {
		exec, _ := setupTestExecutor(t)
		cmd := exec.AbortSlice("slice;rm")
		msg := cmd()
		result := msg.(AbortResultMsg)
		if result.Err == nil {
			t.Error("expected validation error for malicious name")
		}
	})
}

func TestAbortAll(t *testing.T) {
	exec, _ := setupTestExecutor(t)
	cmd := exec.AbortAll()
	msg := cmd()
	result, ok := msg.(AbortResultMsg)
	if !ok {
		t.Fatalf("expected AbortResultMsg, got %T", msg)
	}
	if result.SliceName != "" {
		t.Errorf("SliceName = %q, want empty for abort-all", result.SliceName)
	}
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
	expected := "abort"
	trimmed := strings.TrimSpace(result.Output)
	if trimmed != expected {
		t.Errorf("output = %q, want %q", trimmed, expected)
	}
}
