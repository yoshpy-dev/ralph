package cli

import (
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
			toml:        config.LoopConfig{Driver: "codex", CodexSandbox: "read-only", CodexApprovalPolicy: "on-failure"},
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
			r := checkLoopDriver(cfg, getenv)
			if r.Status != "pass" {
				t.Errorf("status = %q, want pass", r.Status)
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
