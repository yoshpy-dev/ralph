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
	// Codex CLI is opt-in: defaults to false so projects without Codex
	// installed do not see `ralph doctor` exit non-zero.
	if cfg.Doctor.RequireCodexCLI {
		t.Error("require_codex_cli should default to false")
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

// TestDefault_Loop verifies the [loop] defaults stay claude-driven so existing
// users see no behaviour change after Phase 2 lands.
func TestDefault_Loop(t *testing.T) {
	cfg := Default()
	if cfg.Loop.Driver != "claude" {
		t.Errorf("loop.driver default = %q, want claude", cfg.Loop.Driver)
	}
	if cfg.Loop.CodexSandbox != "workspace-write" {
		t.Errorf("loop.codex_sandbox default = %q, want workspace-write", cfg.Loop.CodexSandbox)
	}
	if cfg.Loop.CodexApprovalPolicy != "on-failure" {
		t.Errorf("loop.codex_approval_policy default = %q, want on-failure", cfg.Loop.CodexApprovalPolicy)
	}
}

func TestLoad_LoopCodexDriver(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[loop]
driver = "codex"
codex_sandbox = "read-only"
codex_approval_policy = "untrusted"
claude_reviewer_model = "claude-sonnet-4-6"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Loop.Driver != "codex" {
		t.Errorf("driver = %q, want codex", cfg.Loop.Driver)
	}
	if cfg.Loop.CodexSandbox != "read-only" {
		t.Errorf("codex_sandbox = %q, want read-only", cfg.Loop.CodexSandbox)
	}
	if cfg.Loop.CodexApprovalPolicy != "untrusted" {
		t.Errorf("codex_approval_policy = %q, want untrusted", cfg.Loop.CodexApprovalPolicy)
	}
	if cfg.Loop.ClaudeReviewerModel != "claude-sonnet-4-6" {
		t.Errorf("claude_reviewer_model = %q, want claude-sonnet-4-6", cfg.Loop.ClaudeReviewerModel)
	}
}

func TestLoad_LoopRejectsInvalidDriver(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantSub string
	}{
		{"unknown driver", `[loop]` + "\n" + `driver = "foo"` + "\n", "[loop].driver"},
		{"unknown sandbox", `[loop]` + "\n" + `codex_sandbox = "bogus"` + "\n", "[loop].codex_sandbox"},
		{"unknown approval", `[loop]` + "\n" + `codex_approval_policy = "asap"` + "\n", "[loop].codex_approval_policy"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ralph.toml")
			if err := os.WriteFile(path, []byte(tc.body), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil {
				t.Fatalf("Load: expected error for %s, got nil", tc.name)
			}
			if !contains(err.Error(), tc.wantSub) {
				t.Errorf("error %q does not mention %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// TestLoad_RequireCodexCLI verifies the new toml field round-trips. The doctor
// command relies on this knob to switch between warn (default) and fail when
// the codex binary is missing.
func TestLoad_RequireCodexCLI(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")

	content := `[doctor]
require_claude_cli = false
require_codex_cli = true
require_go = false
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Doctor.RequireCodexCLI {
		t.Error("require_codex_cli = false after loading explicit true")
	}
	if cfg.Doctor.RequireClaudeCLI {
		t.Error("require_claude_cli = true after loading explicit false")
	}
}
