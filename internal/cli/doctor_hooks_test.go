package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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
// the codex-hooks-multi-event plan (docs/plans/).
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
		want := "hooks.json " + eventName + " has no handler routed through ralph-dispatch.sh with the matching event argument"
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
// through ralph-dispatch.sh with the matching event argument" findings for
// any of them.
func TestValidateCodexHooksJSON_AllFourEventsWired_NoMissingEventFindings(t *testing.T) {
	findings := validateCodexHooksJSON([]byte(validHooksJSON))

	for _, f := range findings {
		if strings.Contains(f, "has no handler routed through ralph-dispatch.sh") {
			t.Errorf("findings = %v, want no missing-dispatcher-routing findings when all four shipped events are wired", findings)
		}
	}
}

// misroutedPreToolUseHooksJSON wires PostToolUse correctly but gives the
// PreToolUse entry a command that ends in "ralph-dispatch.sh PostToolUse" —
// the dispatcher is referenced, but with the wrong event argument. This pins
// the cross-review AR#1 fix: dispatcherRoutedByEvent must require the
// MATCHING event argument, not just a bare "ralph-dispatch.sh" substring, or
// a miswired PreToolUse entry would be silently treated as "routed" because
// some other event's command happened to mention the dispatcher (cycle 1,
// docs/reports/cross-review-triage-codex-hooks-multi-event.md).
const misroutedPreToolUseHooksJSON = `{
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
    ],
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "cd \"$(git rev-parse --show-toplevel)\" && ./.claude/hooks/ralph-dispatch.sh PostToolUse"
          }
        ]
      }
    ],
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "cd \"$(git rev-parse --show-toplevel)\" && ./.claude/hooks/ralph-dispatch.sh SessionStart"
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "cd \"$(git rev-parse --show-toplevel)\" && ./.claude/hooks/ralph-dispatch.sh UserPromptSubmit"
          }
        ]
      }
    ]
  }
}`

// TestValidateCodexHooksJSON_PreToolUseCallsWrongEvent_FlagsPreToolUseAsUnrouted
// asserts that a PreToolUse entry whose command invokes the dispatcher with
// PostToolUse's event argument still produces the "PreToolUse has no handler
// routed through ralph-dispatch.sh with the matching event argument" finding
// (the dispatcher call does not count for the event it's wired under), while
// PostToolUse — correctly wired in its own entry — must NOT be flagged.
func TestValidateCodexHooksJSON_PreToolUseCallsWrongEvent_FlagsPreToolUseAsUnrouted(t *testing.T) {
	findings := validateCodexHooksJSON([]byte(misroutedPreToolUseHooksJSON))

	want := "hooks.json PreToolUse has no handler routed through ralph-dispatch.sh with the matching event argument"
	found := false
	for _, f := range findings {
		if f == want {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("findings = %v, want %q (PreToolUse's command invokes the dispatcher with PostToolUse's event argument, which must not count as routed)", findings, want)
	}

	for _, f := range findings {
		if strings.Contains(f, "PostToolUse") && strings.Contains(f, "no handler routed") {
			t.Errorf("findings = %v, PostToolUse is correctly wired in its own entry and must not be flagged as unrouted", findings)
		}
	}
}

// TestValidateCodexHooksJSON_DispatchEventArgRe_PrefixEventNameDoesNotMatch
// is a boundary regression guard: a command ending in
// "ralph-dispatch.sh PostToolUseExtra" must not satisfy the PostToolUse
// routing check merely because "PostToolUseExtra" has "PostToolUse" as a
// prefix. It also pins the cycle-2 cross-review C2-3 fix: a double-quoted or
// single-quoted event argument (legal shell, semantically identical to the
// unquoted form) must still count as routed.
func TestValidateCodexHooksJSON_DispatchEventArgRe_PrefixEventNameDoesNotMatch(t *testing.T) {
	re, ok := dispatchEventArgRes["PostToolUse"]
	if !ok {
		t.Fatal("dispatchEventArgRes missing an entry for PostToolUse")
	}
	if re.MatchString("cd \"$(git rev-parse --show-toplevel)\" && ./.claude/hooks/ralph-dispatch.sh PostToolUseExtra") {
		t.Error("dispatchEventArgRes[\"PostToolUse\"] matched a command invoking \"ralph-dispatch.sh PostToolUseExtra\" — the event-name boundary check must reject a longer event name that merely has PostToolUse as a prefix")
	}
	if !re.MatchString("cd \"$(git rev-parse --show-toplevel)\" && ./.claude/hooks/ralph-dispatch.sh PostToolUse") {
		t.Error("dispatchEventArgRes[\"PostToolUse\"] failed to match a correctly routed command ending in \"ralph-dispatch.sh PostToolUse\"")
	}
	if !re.MatchString("cd \"$(git rev-parse --show-toplevel)\" && ./.claude/hooks/ralph-dispatch.sh \"PostToolUse\"") {
		t.Error("dispatchEventArgRes[\"PostToolUse\"] failed to match a command invoking \"ralph-dispatch.sh \\\"PostToolUse\\\"\" (double-quoted event argument, C2-3)")
	}
	if !re.MatchString("cd \"$(git rev-parse --show-toplevel)\" && ./.claude/hooks/ralph-dispatch.sh 'PostToolUse'") {
		t.Error("dispatchEventArgRes[\"PostToolUse\"] failed to match a command invoking \"ralph-dispatch.sh 'PostToolUse'\" (single-quoted event argument, C2-3)")
	}
}

// TestCodexShippedHookEventsMatchesShippedHooksJSON binds codexShippedHookEvents
// (the Go list doctor.go enforces in every downstream project) to the actual
// event keys present in the tracked templates/base/.codex/hooks.json (the
// hooks.json ralph init scaffolds). Without this test the two lists are only
// linked by a doc comment ("Keep in sync with the shipped hooks.json event
// set"), so adding an event to hooks.json without also adding it to
// codexShippedHookEvents (or vice versa) previously went undetected — see
// docs/reports/self-review-2026-08-24-codex-hooks-multi-event.md (M2).
//
// Paths are resolved relative to this source file via runtime.Caller, the
// same idiom internal/config/defaults_sync_test.go uses, so the test works
// regardless of the working directory it is invoked from.
func TestCodexShippedHookEventsMatchesShippedHooksJSON(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location via runtime.Caller")
	}
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..", "..")

	hooksJSONPath := filepath.Join(repoRoot, "templates", "base", ".codex", "hooks.json")
	data, err := os.ReadFile(hooksJSONPath)
	if err != nil {
		t.Skipf("templates/base/.codex/hooks.json not found (%v) — skipping sync check (vendored repo?)", err)
	}

	var doc struct {
		Hooks map[string]json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("templates/base/.codex/hooks.json: invalid JSON: %v", err)
	}

	shipped := append([]string(nil), codexShippedHookEvents...)
	sort.Strings(shipped)

	var fileEvents []string
	for eventName := range doc.Hooks {
		fileEvents = append(fileEvents, eventName)
	}
	sort.Strings(fileEvents)

	if len(shipped) != len(fileEvents) {
		t.Fatalf("codexShippedHookEvents (Go) = %v\ntemplates/base/.codex/hooks.json events = %v\nwant the same event set (doctor.go's codexShippedHookEvents must be kept in sync with the shipped hooks.json)",
			shipped, fileEvents)
	}
	for i := range shipped {
		if shipped[i] != fileEvents[i] {
			t.Fatalf("codexShippedHookEvents (Go) = %v\ntemplates/base/.codex/hooks.json events = %v\nwant the same event set (doctor.go's codexShippedHookEvents must be kept in sync with the shipped hooks.json)",
				shipped, fileEvents)
		}
	}
}
