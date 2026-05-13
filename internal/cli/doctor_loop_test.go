package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// TestCheckLoopDriver_PriorityAndSource verifies the env > TOML > default
// priority documented in AGENTS.md and the plan's Design decisions, and
// confirms the doctor result names the source so users can see which knob
// is actually in effect.
func TestCheckLoopDriver_PriorityAndSource(t *testing.T) {
	cases := []struct {
		name        string
		env         map[string]string
		toml        config.LoopConfig
		wantValue   string
		wantSource  string
		wantSandbox string // checked only when wantValue == "codex"
	}{
		{
			name:       "default when nothing set (TOML defaults match)",
			env:        nil,
			toml:       config.Default().Loop,
			wantValue:  "claude",
			wantSource: "toml", // TOML loaded from Default() produces "claude" — still toml-sourced
		},
		{
			name:       "env wins over TOML",
			env:        map[string]string{"RALPH_LOOP_DRIVER": "codex"},
			toml:       config.LoopConfig{Driver: "claude", CodexSandbox: "workspace-write", CodexApprovalPolicy: "on-failure"},
			wantValue:  "codex",
			wantSource: "env",
		},
		{
			name:        "TOML alone selects codex",
			env:         nil,
			toml:        config.LoopConfig{Driver: "codex", CodexSandbox: "read-only", CodexApprovalPolicy: "on-failure", ClaudeReviewerModel: "claude-opus-4-7"},
			wantValue:   "codex",
			wantSource:  "toml",
			wantSandbox: "read-only",
		},
		{
			name:       "totally empty TOML falls back to default",
			env:        nil,
			toml:       config.LoopConfig{},
			wantValue:  "claude",
			wantSource: "default",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{Loop: tc.toml}
			getenv := func(k string) string {
				if tc.env == nil {
					return ""
				}
				return tc.env[k]
			}
			// Provide a sham `codex` binary on PATH so the codex-driver path
			// reaches the detail string instead of short-circuiting to fail
			// (the missing-binary case has its own focused test below).
			if tc.wantValue == "codex" {
				dir := t.TempDir()
				stub := filepath.Join(dir, "codex")
				if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			}
			r := checkLoopDriver(cfg, getenv)
			if r.Status != "pass" {
				t.Errorf("status = %q, want pass (detail=%q)", r.Status, r.Detail)
			}
			if !strings.Contains(r.Detail, tc.wantValue) {
				t.Errorf("detail %q missing value %q", r.Detail, tc.wantValue)
			}
			if !strings.Contains(r.Detail, "source: "+tc.wantSource) {
				t.Errorf("detail %q missing source %q", r.Detail, tc.wantSource)
			}
			if tc.wantSandbox != "" && !strings.Contains(r.Detail, "sandbox: "+tc.wantSandbox) {
				t.Errorf("detail %q missing sandbox %q (codex driver should expose it)", r.Detail, tc.wantSandbox)
			}
		})
	}
}

// TestCheckLoopDriver_FailsWhenCodexMissing pins the cycle-3 cross-review
// fix: doctor must surface the mismatch when driver=codex is effective but
// the codex binary is not installed, instead of reporting pass and letting
// the next `ralph run` preflight block.
func TestCheckLoopDriver_FailsWhenCodexMissing(t *testing.T) {
	// Empty PATH directory → no codex binary discoverable.
	t.Setenv("PATH", t.TempDir())

	cfg := config.Config{Loop: config.LoopConfig{Driver: "codex", CodexSandbox: "workspace-write", CodexApprovalPolicy: "on-failure", ClaudeReviewerModel: "claude-opus-4-7"}}
	getenv := func(string) string { return "" }

	r := checkLoopDriver(cfg, getenv)
	if r.Status != "fail" {
		t.Errorf("status = %q, want fail (codex absent + driver=codex)", r.Status)
	}
	if !strings.Contains(r.Detail, "codex binary not found") {
		t.Errorf("detail %q should explain the mismatch", r.Detail)
	}
}

// TestCheckLoopDriver_EnvOverridesShownInDetail covers cycle-3 cross-review
// finding #3: when RALPH_CODEX_SANDBOX or RALPH_CODEX_APPROVAL_POLICY is
// set in the environment, doctor's detail line must reflect the env value
// instead of silently showing the TOML/default.
func TestCheckLoopDriver_EnvOverridesShownInDetail(t *testing.T) {
	// Sham codex on PATH so the function does not short-circuit to fail.
	dir := t.TempDir()
	stub := filepath.Join(dir, "codex")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := config.Config{Loop: config.LoopConfig{
		Driver:              "codex",
		CodexSandbox:        "workspace-write",
		CodexApprovalPolicy: "on-failure",
		ClaudeReviewerModel: "claude-opus-4-7",
	}}
	env := map[string]string{
		"RALPH_CODEX_SANDBOX":         "danger-full-access",
		"RALPH_CODEX_APPROVAL_POLICY": "never",
	}
	getenv := func(k string) string { return env[k] }

	r := checkLoopDriver(cfg, getenv)
	if r.Status != "pass" {
		t.Fatalf("status = %q, want pass (detail=%q)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "sandbox: danger-full-access") {
		t.Errorf("detail should reflect env-overridden sandbox; got %q", r.Detail)
	}
	if !strings.Contains(r.Detail, "approval: never") {
		t.Errorf("detail should reflect env-overridden approval; got %q", r.Detail)
	}
}
