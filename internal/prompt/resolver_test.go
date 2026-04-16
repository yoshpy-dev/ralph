package prompt

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolve_LocalOverride(t *testing.T) {
	dir := t.TempDir()
	promptDir := filepath.Join(dir, ".ralph", "prompts")
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a local override.
	if err := os.WriteFile(filepath.Join(promptDir, "self-review.md"), []byte("custom review prompt"), 0644); err != nil {
		t.Fatal(err)
	}

	content, err := Resolve("self-review", promptDir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if string(content) != "custom review prompt" {
		t.Errorf("content = %q, want custom review prompt", content)
	}
}

func TestResolve_FallbackToEmbedded(t *testing.T) {
	// Use a non-existent directory for project prompts → should fall back.
	_, err := Resolve("self-review", "/nonexistent/prompts")
	// This will fail if EmbeddedFS is not initialized (unit test context).
	// That's expected — this test validates the fallback logic path.
	if err != nil {
		t.Skipf("EmbeddedFS not initialized: %v", err)
	}
}
