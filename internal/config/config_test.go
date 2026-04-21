package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Pipeline.Model != "claude-opus-4-7" {
		t.Errorf("model = %q", cfg.Pipeline.Model)
	}
	if cfg.Pipeline.Effort != "xhigh" {
		t.Errorf("effort = %q", cfg.Pipeline.Effort)
	}
	if cfg.Pipeline.MaxIterations != 20 {
		t.Errorf("max_iterations = %d", cfg.Pipeline.MaxIterations)
	}
	if cfg.Pipeline.Prompts.Dir != ".ralph/prompts" {
		t.Errorf("prompts.dir = %q", cfg.Pipeline.Prompts.Dir)
	}
	if !cfg.Doctor.RequireClaudeCLI {
		t.Error("require_claude_cli should be true by default")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/ralph.toml")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	// Should return defaults.
	if cfg.Pipeline.MaxIterations != 20 {
		t.Errorf("max_iterations = %d, want 20", cfg.Pipeline.MaxIterations)
	}
}

func TestLoad_PartialConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")

	content := `[pipeline]
model = "claude-opus-4-20250514"
max_parallel = 8
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Overridden values.
	if cfg.Pipeline.Model != "claude-opus-4-20250514" {
		t.Errorf("model = %q, want claude-opus-4-20250514", cfg.Pipeline.Model)
	}
	if cfg.Pipeline.MaxParallel != 8 {
		t.Errorf("max_parallel = %d, want 8", cfg.Pipeline.MaxParallel)
	}

	// Defaults for unspecified values.
	if cfg.Pipeline.MaxIterations != 20 {
		t.Errorf("max_iterations = %d, want 20", cfg.Pipeline.MaxIterations)
	}
	if cfg.Pipeline.Effort != "xhigh" {
		t.Errorf("effort = %q, want xhigh", cfg.Pipeline.Effort)
	}
}

func TestLoad_FullRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")

	content := `[pipeline]
model = "claude-opus-4-7"
effort = "xhigh"
max_iterations = 20
max_parallel = 4
slice_timeout = "30m"
permission_mode = "auto"

[pipeline.prompts]
dir = ".ralph/prompts"

[doctor]
require_claude_cli = true
require_go = false
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Pipeline.Model != "claude-opus-4-7" {
		t.Errorf("model = %q", cfg.Pipeline.Model)
	}
	if cfg.Pipeline.SliceTimeout != "30m" {
		t.Errorf("slice_timeout = %q", cfg.Pipeline.SliceTimeout)
	}
	if cfg.Doctor.RequireGo {
		t.Error("require_go should be false")
	}
}
