package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCheckLoopDriver_PriorityAndSource verifies the env > default priority
// and confirms the doctor result names the source so users can see which
// knob is actually in effect. The [loop] ralph.toml section this used to also
// read from was removed along with the rest of the Ralph Loop execution
// system (checkLoopDriver itself is a stub pending full removal — see its
// doc comment in doctor.go).
func TestCheckLoopDriver_PriorityAndSource(t *testing.T) {
	cases := []struct {
		name        string
		env         map[string]string
		wantValue   string
		wantSource  string
		wantSandbox string // checked only when wantValue == "codex"
	}{
		{
			name:       "default when nothing set",
			env:        nil,
			wantValue:  "claude",
			wantSource: "default",
		},
		{
			name:       "env wins over default",
			env:        map[string]string{"RALPH_LOOP_DRIVER": "codex"},
			wantValue:  "codex",
			wantSource: "env",
		},
		{
			name:        "env selects codex with an env-overridden sandbox",
			env:         map[string]string{"RALPH_LOOP_DRIVER": "codex", "RALPH_CODEX_SANDBOX": "read-only"},
			wantValue:   "codex",
			wantSource:  "env",
			wantSandbox: "read-only",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
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
			r := checkLoopDriver(getenv)
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

	getenv := func(k string) string {
		if k == "RALPH_LOOP_DRIVER" {
			return "codex"
		}
		return ""
	}

	r := checkLoopDriver(getenv)
	if r.Status != "fail" {
		t.Errorf("status = %q, want fail (codex absent + driver=codex)", r.Status)
	}
	if !strings.Contains(r.Detail, "codex binary not found") {
		t.Errorf("detail %q should explain the mismatch", r.Detail)
	}
}

// TestCheckLoopDriver_EnvOverridesShownInDetail covers cycle-3 cross-review
// finding #3: when RALPH_CODEX_SANDBOX or RALPH_CODEX_APPROVAL_POLICY is set
// in the environment, doctor's detail line must reflect the env value.
func TestCheckLoopDriver_EnvOverridesShownInDetail(t *testing.T) {
	// Sham codex on PATH so the function does not short-circuit to fail.
	dir := t.TempDir()
	stub := filepath.Join(dir, "codex")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	env := map[string]string{
		"RALPH_LOOP_DRIVER":           "codex",
		"RALPH_CODEX_SANDBOX":         "danger-full-access",
		"RALPH_CODEX_APPROVAL_POLICY": "never",
	}
	getenv := func(k string) string { return env[k] }

	r := checkLoopDriver(getenv)
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
