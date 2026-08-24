package cli

import (
	"os"
	"path/filepath"
	"strings"
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

// postToolUseOnlyHooksJSON is a schema-valid hooks.json whose only wired
// event is PostToolUse — the pre-multi-event shape ralph shipped before
// docs/plans/active/2026-08-24-codex-hooks-multi-event.md.
const postToolUseOnlyHooksJSON = `{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit|Write|MultiEdit|apply_patch",
        "hooks": [
          {
            "type": "command",
            "command": "cd \"$(git rev-parse --show-toplevel)\" && ./.claude/hooks/ralph-dispatch.sh PostToolUse"
          }
        ]
      }
    ]
  }
}`

// TestValidateCodexHooksJSON_PostToolUseOnly_FlagsMissingShippedEvents
// asserts that once PreToolUse/SessionStart/UserPromptSubmit join the
// shipped event set (codexShippedHookEvents), a hooks.json that only wires
// PostToolUse must produce one distinct finding per missing event — not just
// the legacy PostToolUse-only check.
func TestValidateCodexHooksJSON_PostToolUseOnly_FlagsMissingShippedEvents(t *testing.T) {
	findings := validateCodexHooksJSON([]byte(postToolUseOnlyHooksJSON))

	for _, eventName := range []string{"PreToolUse", "SessionStart", "UserPromptSubmit"} {
		want := "hooks.json " + eventName + " has no handler routed through ralph-dispatch.sh"
		found := false
		for _, f := range findings {
			if f == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("findings = %v, want a finding %q for missing event %s", findings, want, eventName)
		}
	}

	for _, f := range findings {
		if strings.Contains(f, "PostToolUse") && strings.Contains(f, "no handler routed") {
			t.Errorf("findings = %v, PostToolUse is wired in the fixture and must not be flagged as missing", findings)
		}
	}
}

// TestValidateCodexHooksJSON_AllFourEventsWired_NoMissingEventFindings
// asserts that a hooks.json wiring all four shipped events (mirroring the
// real tracked .codex/hooks.json) produces no "has no handler routed
// through ralph-dispatch.sh" findings for any of them.
func TestValidateCodexHooksJSON_AllFourEventsWired_NoMissingEventFindings(t *testing.T) {
	findings := validateCodexHooksJSON([]byte(validHooksJSON))

	for _, f := range findings {
		if strings.Contains(f, "has no handler routed through ralph-dispatch.sh") {
			t.Errorf("findings = %v, want no missing-dispatcher-routing findings when all four shipped events are wired", findings)
		}
	}
}
