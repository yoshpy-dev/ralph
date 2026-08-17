package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// writeSettingsWithHooksCommand writes a minimal .claude/settings.json under
// dir whose SessionStart hook command is cmd, mirroring the shape ralph's
// own templates ship (templates/base/.claude/settings.json): a "hooks" key
// per Claude Code's settings schema.
func writeSettingsWithHooksCommand(t *testing.T, dir, cmd string) {
	t.Helper()
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	settings := `{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup|resume|clear|compact",
        "hooks": [
          {
            "type": "command",
            "command": "` + cmd + `"
          }
        ]
      }
    ]
  }
}
`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(settings), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestCheckHooks_DispatcherCommandWithArgs_ScriptPresentReportsPass pins the
// cycle-2 cross-review fix: settings.json hook commands are "executable arg"
// strings (e.g. "./.claude/hooks/ralph-dispatch.sh SessionStart"), the shape
// every v2 scaffold ships. checkHooks must stat only the executable token,
// not the whole command string, or every fresh v2 init falsely reports
// "hook script(s) missing". This is the reviewer's repro: `ralph init --yes`
// followed by `ralph doctor` previously reported hooks integrity as failed
// even though the dispatcher script was present on disk.
func TestCheckHooks_DispatcherCommandWithArgs_ScriptPresentReportsPass(t *testing.T) {
	dir := t.TempDir()
	writeSettingsWithHooksCommand(t, dir, "./.claude/hooks/ralph-dispatch.sh SessionStart")

	hooksDir := filepath.Join(dir, ".claude", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "ralph-dispatch.sh"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := checkHooks(dir)
	if r.Status != "pass" {
		t.Fatalf("checkHooks status = %q, detail = %q, want pass (dispatcher script exists at the first whitespace-separated token of the command)", r.Status, r.Detail)
	}
}

// TestCheckHooks_DispatcherCommandWithArgs_ScriptMissingReportsFail ensures
// the whitespace-splitting fix does not mask a genuinely missing hook
// script: only the executable token is stat'd, but that token's absence
// must still surface as a failure.
func TestCheckHooks_DispatcherCommandWithArgs_ScriptMissingReportsFail(t *testing.T) {
	dir := t.TempDir()
	writeSettingsWithHooksCommand(t, dir, "./.claude/hooks/ralph-dispatch.sh SessionStart")
	// Deliberately do not create .claude/hooks/ralph-dispatch.sh.

	r := checkHooks(dir)
	if r.Status != "fail" {
		t.Fatalf("checkHooks status = %q, want fail when the dispatcher script is genuinely absent", r.Status)
	}
	if r.Detail != "1 hook script(s) missing" {
		t.Errorf("checkHooks detail = %q, want %q", r.Detail, "1 hook script(s) missing")
	}
}

// TestCheckHooks_ArgLessCommand_ScriptPresentReportsPass is a regression
// guard for the pre-existing (non-dispatcher) shape: a bare command with no
// arguments must keep working exactly as before the whitespace-split fix.
func TestCheckHooks_ArgLessCommand_ScriptPresentReportsPass(t *testing.T) {
	dir := t.TempDir()
	writeSettingsWithHooksCommand(t, dir, "./.claude/hooks/session-start.sh")

	hooksDir := filepath.Join(dir, ".claude", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooksDir, "session-start.sh"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := checkHooks(dir)
	if r.Status != "pass" {
		t.Fatalf("checkHooks status = %q, detail = %q, want pass", r.Status, r.Detail)
	}
}
