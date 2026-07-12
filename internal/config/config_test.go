package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Pipeline.Model != "opus" {
		t.Errorf("model = %q", cfg.Pipeline.Model)
	}
	if cfg.Pipeline.Effort != "high" {
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
	// Codex is opt-in: defaults to false so projects without Codex
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
	if cfg.Pipeline.Effort != "high" {
		t.Errorf("effort = %q, want high", cfg.Pipeline.Effort)
	}
}

func TestLoad_FullRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")

	content := `[pipeline]
model = "opus"
effort = "high"
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

	if cfg.Pipeline.Model != "opus" {
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

// TestDefault_Phases verifies the [pipeline.phases] defaults mirror the
// RALPH_<PHASE>_MODEL shell defaults in scripts/ralph-config.sh — the two
// must change in lock-step (see .claude/rules/model-routing.md).
func TestDefault_Phases(t *testing.T) {
	p := Default().Pipeline.Phases
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"implement", p.Implement, "sonnet"},
		{"self_review", p.SelfReview, "opus"},
		{"verify", p.Verify, "sonnet"},
		{"test", p.Test, "sonnet"},
		{"sync_docs", p.SyncDocs, "sonnet"},
		{"pr", p.PR, "sonnet"},
		{"probe", p.Probe, "haiku"},
		{"escalation", p.Escalation, "opus"},
		// force is an override knob, not a default: empty means "no override".
		{"force", p.Force, ""},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("phases.%s default = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

// TestLoad_PhasesAbsent verifies that a ralph.toml without [pipeline.phases]
// yields all phase defaults (backward compatibility for existing projects).
func TestLoad_PhasesAbsent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[pipeline]
model = "opus"
max_parallel = 8
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Pipeline.Phases != Default().Pipeline.Phases {
		t.Errorf("phases = %+v, want defaults %+v", cfg.Pipeline.Phases, Default().Pipeline.Phases)
	}
}

// TestLoad_PhasesPartialBackfill verifies that a partial [pipeline.phases]
// section keeps the explicit value and backfills every other key — except
// force, which must stay empty because empty is meaningful (no override).
func TestLoad_PhasesPartialBackfill(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[pipeline.phases]
implement = "opus"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Pipeline.Phases.Implement != "opus" {
		t.Errorf("implement = %q, want opus (explicit value)", cfg.Pipeline.Phases.Implement)
	}
	if cfg.Pipeline.Phases.SelfReview != "opus" {
		t.Errorf("self_review = %q, want opus (backfilled)", cfg.Pipeline.Phases.SelfReview)
	}
	if cfg.Pipeline.Phases.Verify != "sonnet" {
		t.Errorf("verify = %q, want sonnet (backfilled)", cfg.Pipeline.Phases.Verify)
	}
	if cfg.Pipeline.Phases.Probe != "haiku" {
		t.Errorf("probe = %q, want haiku (backfilled)", cfg.Pipeline.Phases.Probe)
	}
	if cfg.Pipeline.Phases.Force != "" {
		t.Errorf("force = %q, want empty (never backfilled)", cfg.Pipeline.Phases.Force)
	}
}

// TestLoad_PhasesFullRoundTrip verifies every [pipeline.phases] key, including
// force, survives a full parse unchanged.
func TestLoad_PhasesFullRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[pipeline.phases]
implement   = "haiku"
self_review = "sonnet"
verify      = "opus"
test        = "opus"
sync_docs   = "haiku"
pr          = "haiku"
probe       = "sonnet"
escalation  = "sonnet"
force       = "opus"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := PhaseModelConfig{
		Implement:  "haiku",
		SelfReview: "sonnet",
		Verify:     "opus",
		Test:       "opus",
		SyncDocs:   "haiku",
		PR:         "haiku",
		Probe:      "sonnet",
		Escalation: "sonnet",
		Force:      "opus",
	}
	if cfg.Pipeline.Phases != want {
		t.Errorf("phases = %+v, want %+v", cfg.Pipeline.Phases, want)
	}
}

// TestLoad_TemplateRalphToml verifies the shipped templates/base/ralph.toml
// parses cleanly and its [pipeline.phases] values match the Go defaults —
// catching drift between the scaffolded file and Default(). Repo-root
// discovery follows the runtime.Caller pattern used by
// internal/scaffold/embed_test.go.
func TestLoad_TemplateRalphToml(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	// thisFile is internal/config/config_test.go → repo root is ../../
	path := filepath.Join(filepath.Dir(thisFile), "..", "..", "templates", "base", "ralph.toml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("templates/base/ralph.toml not found: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(templates/base/ralph.toml): %v", err)
	}
	if cfg.Pipeline.Phases != Default().Pipeline.Phases {
		t.Errorf("template phases = %+v, want defaults %+v (template and Default() must stay in lock-step)",
			cfg.Pipeline.Phases, Default().Pipeline.Phases)
	}
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
