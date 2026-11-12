package cli

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

// Tests for codex_hook_trust_parse.go (plan 2026-11-12-codex-hook-trust-check,
// issue #310): the event labels and the hooks.json reader. The helpers they
// share with the rest of the codex hook trust tests are in
// codex_hook_trust_test.go.

func TestCodexHookTrust_EventLabelsAreCodexs14(t *testing.T) {
	want := map[string]string{
		"PreToolUse": "pre_tool_use", "PermissionRequest": "permission_request",
		"PostToolUse": "post_tool_use", "PreCompact": "pre_compact", "PostCompact": "post_compact",
		"PreModelRequest": "pre_model_request", "PostModelRequest": "post_model_request",
		"SessionStart": "session_start", "SessionEnd": "session_end",
		"UserPromptSubmit": "user_prompt_submit", "SubagentStart": "subagent_start",
		"SubagentStop": "subagent_stop", "Stop": "stop", "Interrupt": "interrupt",
	}
	if len(want) != 14 {
		t.Fatalf("the expectation itself must list 14 events, has %d", len(want))
	}
	if !maps.Equal(codexHookEventLabels, want) {
		t.Errorf("codexHookEventLabels = %v, want %v", codexHookEventLabels, want)
	}
}

func TestCodexHookTrust_SlotsCountPositionsLikeCodex(t *testing.T) {
	data := `{
  "description": "ignored",
  "hooks": {
    "SessionStart": [{"hooks": [{"type": "command", "command": "echo start"}]}],
    "PreToolUse": [
      {"matcher": "Bash", "hooks": [
        {"type": "command", "command": "  \t"},
        {"type": "command", "command": "guard.sh", "timeout": 30}
      ]},
      {"matcher": "Edit", "hooks": [
        {"type": "command", "command": "a"},
        {"type": "command", "command": "b"}
      ]}
    ],
    "PostToolUse": [{"hooks": [{"type": "command", "command": ""}]}],
    "UserPromptSubmit": [{"hooks": [{"type": "mcp_tool", "server": "s", "tool": "t"}]}],
    "NotACodexEvent": [{"hooks": [{"type": "command", "command": "x"}]}],
    "Stop": [{"hooks": []}]
  }
}`
	slots, err := parseCodexHookSlots([]byte(data))
	if err != nil {
		t.Fatalf("parseCodexHookSlots: %v", err)
	}
	var tails []string
	for _, slot := range slots {
		tails = append(tails, slot.Tail())
	}
	// The blank command at PreToolUse 0:0 gets no slot, but the next handler
	// keeps position 1; the empty PostToolUse command, the unknown event, and
	// the empty Stop group get none. A non-command handler is counted.
	want := []string{"pre_tool_use:0:1", "pre_tool_use:1:0", "pre_tool_use:1:1", "session_start:0:0", "user_prompt_submit:0:0"}
	if !slices.Equal(tails, want) {
		t.Fatalf("slot tails = %v, want %v", tails, want)
	}
	if got := codexHookKey("/r", slots[0]); got != "/r/.codex/hooks.json:pre_tool_use:0:1" {
		t.Errorf("codexHookKey = %q", got)
	}
}

// A file with no handler codex asks approval for has no slots. A group
// without "hooks", an unknown event of any value, and a null description are
// what serde defaults or skips; "hooks": null is not (see
// TestCodexHookTrust_SlotsRejectNullAndMissingFieldsLikeCodex).
func TestCodexHookTrust_SlotsOfAFileWithNoHandlers(t *testing.T) {
	for _, data := range []string{
		`{}`, `{"hooks": {}}`, `{"hooks": {"Stop": []}}`, `{"hooks": {"Stop": [{"hooks": []}]}}`,
		`{"hooks": {"Stop": [{}]}}`, `{"hooks": {"Stop": [{"matcher": null}]}}`,
		`{"hooks": {"NotACodexEvent": null, "Other": 5}}`, `{"description": null, "hooks": {}}`,
	} {
		slots, err := parseCodexHookSlots([]byte(data))
		if err != nil {
			t.Errorf("%s: unexpected error %v", data, err)
		}
		if len(slots) != 0 {
			t.Errorf("%s: slots = %v, want none", data, slots)
		}
	}
}

func TestCodexHookTrust_SlotsRejectWhatIsNotCodexsShape(t *testing.T) {
	cases := []struct {
		data string
		want string // the start of the error text
	}{
		{`not json`, "syntax error at byte "},
		{`{"hooks": {}`, "syntax error at byte "},
		{`null`, "unexpected JSON null at the top level"},
		{`[]`, "unexpected JSON array at the top level"},
		{`"hooks"`, "unexpected JSON string at the top level"},
		{`{"hooks": []}`, `unexpected JSON array at "hooks"`},
		{`{"hooks": "x"}`, `unexpected JSON string at "hooks"`},
		{`{"hooks": {"PreToolUse": {}}}`, `unexpected JSON object at "hooks.PreToolUse"`},
		{`{"hooks": {"PreToolUse": [{"hooks": [{"type": "command", "command": 5}]}]}}`, `unexpected JSON number at "hooks.PreToolUse.0.hooks.0.command"`},
	}
	for _, tc := range cases {
		slots, err := parseCodexHookSlots([]byte(tc.data))
		if err == nil {
			t.Errorf("%s: no error, slots %v", tc.data, slots)
			continue
		}
		if !strings.HasPrefix(err.Error(), tc.want) {
			t.Errorf("%s: error %q, want it to start with %q", tc.data, err, tc.want)
		}
		if strings.Contains(err.Error(), "Go ") || strings.Contains(err.Error(), "cli.") {
			t.Errorf("%s: error %q names Go types", tc.data, err)
		}
	}
}

// codexHookTrustInvalidHooks are hooks.json files codex 0.166.0's serde types
// refuse, so codex loads no hook from them (plan D7 (b), AC11), with the
// error the check gives for each.
var codexHookTrustInvalidHooks = []struct {
	name, data, want string
}{
	{"hooks null", `{"description": "x", "hooks": null}`, `unexpected JSON null at "hooks"`},
	{"an event's value null", `{"hooks": {"PreToolUse": null}}`, `unexpected JSON null at "hooks.PreToolUse"`},
	{"a group null", `{"hooks": {"PreToolUse": [null]}}`, `unexpected JSON null at "hooks.PreToolUse.0"`},
	{"a group's hooks null", `{"hooks": {"PreToolUse": [{"hooks": null}]}}`, `unexpected JSON null at "hooks.PreToolUse.0.hooks"`},
	{"a handler null", `{"hooks": {"PreToolUse": [{"hooks": [null]}]}}`, `unexpected JSON null at "hooks.PreToolUse.0.hooks.0"`},
	{"a handler without type", `{"hooks": {"PreToolUse": [{"hooks": [{"command": "guard.sh"}]}]}}`, `missing key "type" at "hooks.PreToolUse.0.hooks.0"`},
	{"a command handler without command", `{"hooks": {"PreToolUse": [{"hooks": [{"type": "command"}]}]}}`, `missing key "command" at "hooks.PreToolUse.0.hooks.0"`},
}

func TestCodexHookTrust_SlotsRejectNullAndMissingFieldsLikeCodex(t *testing.T) {
	cases := append(append([]struct{ name, data, want string }{}, codexHookTrustInvalidHooks...), []struct{ name, data, want string }{
		{"not UTF-8", "{\"description\": \"\xff\"}", "not UTF-8"},
		{"an unknown top-level key", `{"hooks": {}, "version": 1}`, `unknown key "version" at the top level`},
		{"hooks given twice", `{"hooks": {}, "hooks": {}}`, `duplicate key "hooks" at the top level`},
		{"a description that is not a string", `{"description": 5}`, `unexpected JSON number at "description"`},
		{"an event given twice", `{"hooks": {"Stop": [], "Stop": []}}`, `duplicate key "Stop" at "hooks"`},
		{"an event's value an object", `{"hooks": {"Stop": {"hooks": []}}}`, `unexpected JSON object at "hooks.Stop"`},
		{"a group a string", `{"hooks": {"Stop": ["x"]}}`, `unexpected JSON string at "hooks.Stop.0"`},
		{"a matcher that is not a string", `{"hooks": {"Stop": [{"matcher": 5, "hooks": []}]}}`, `unexpected JSON number at "hooks.Stop.0.matcher"`},
		{"a group's hooks an object", `{"hooks": {"Stop": [{"hooks": {}}]}}`, `unexpected JSON object at "hooks.Stop.0.hooks"`},
		{"a handler an array", `{"hooks": {"Stop": [{"hooks": [[]]}]}}`, `unexpected JSON array at "hooks.Stop.0.hooks.0"`},
		{"a type that is not a string", `{"hooks": {"Stop": [{"hooks": [{"type": 1}]}]}}`, `unexpected JSON number at "hooks.Stop.0.hooks.0.type"`},
		{"an unknown type", `{"hooks": {"Stop": [{"hooks": [{"type": "script", "command": "x"}]}]}}`, `unknown handler type "script" at "hooks.Stop.0.hooks.0.type"`},
		{"type given twice", `{"hooks": {"Stop": [{"hooks": [{"type": "command", "type": "command", "command": "x"}]}]}}`, `duplicate key "type" at "hooks.Stop.0.hooks.0"`},
		{"a command null", `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": null}]}]}}`, `unexpected JSON null at "hooks.Stop.0.hooks.0.command"`},
		{"both spellings of commandWindows", `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "x", "commandWindows": "y", "command_windows": "y"}]}]}}`, `duplicate key "command_windows" at "hooks.Stop.0.hooks.0"`},
		{"a negative timeout", `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "x", "timeout": -1}]}]}}`, `not an integer from 0 to 2^64-1 at "hooks.Stop.0.hooks.0.timeout"`},
		{"a fractional timeout", `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "x", "timeout": 1.5}]}]}}`, `not an integer from 0 to 2^64-1 at "hooks.Stop.0.hooks.0.timeout"`},
		{"a timeout as a string", `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "x", "timeout": "30"}]}]}}`, `unexpected JSON string at "hooks.Stop.0.hooks.0.timeout"`},
		{"async null", `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "x", "async": null}]}]}}`, `unexpected JSON null at "hooks.Stop.0.hooks.0.async"`},
		{"an mcp_tool handler without tool", `{"hooks": {"Stop": [{"hooks": [{"type": "mcp_tool", "server": "s"}]}]}}`, `missing key "tool" at "hooks.Stop.0.hooks.0"`},
		{"an mcp_tool input with a null", `{"hooks": {"Stop": [{"hooks": [{"type": "mcp_tool", "server": "s", "tool": "t", "input": {"a": [1, null]}}]}]}}`, `unexpected JSON null at "hooks.Stop.0.hooks.0.input.a.1"`},
		{"an mcp_tool input above 2^63-1", `{"hooks": {"Stop": [{"hooks": [{"type": "mcp_tool", "server": "s", "tool": "t", "input": {"n": 9223372036854775808}}]}]}}`, `an integer above 2^63-1 at "hooks.Stop.0.hooks.0.input.n"`},
		{"an mcp_tool input null", `{"hooks": {"Stop": [{"hooks": [{"type": "mcp_tool", "server": "s", "tool": "t", "input": null}]}]}}`, `unexpected JSON null at "hooks.Stop.0.hooks.0.input"`},
	}...)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			slots, err := parseCodexHookSlots([]byte(tc.data))
			if err == nil {
				t.Fatalf("%s: no error, slots %v; want %q", tc.data, slots, tc.want)
			}
			if err.Error() != tc.want {
				t.Errorf("%s: error %q, want %q", tc.data, err, tc.want)
			}
		})
	}
}

// What an error quotes from a hooks.json (a key, a handler type, or a place
// that holds the keys of an MCP tool's input) is cut to 64 runes followed by
// "…", and a line break in it is escaped, so the line doctor prints stays
// short and one line however long the key in the file is (self-review C2-2).
func TestCodexHookTrust_SlotErrorsQuoteAtMost64Runes(t *testing.T) {
	long := strings.Repeat("k", 200)
	handler := func(fields string) string {
		return `{"hooks": {"Stop": [{"hooks": [{` + fields + `}]}]}}`
	}
	// The place of the MCP input key is "hooks.Stop.0.hooks.0.input." (27
	// runes) and the key, so 37 runes of the key are kept.
	inputAt := "hooks.Stop.0.hooks.0.input."
	cases := []struct{ name, data, want string }{
		{"a key of 64 runes is kept whole", `{"` + strings.Repeat("k", 64) + `": 1}`,
			`unknown key "` + strings.Repeat("k", 64) + `" at the top level`},
		{"a longer key is cut", `{"hooks": {}, "` + long + `": 1}`,
			`unknown key "` + strings.Repeat("k", 64) + `…" at the top level`},
		{"runes are counted, not bytes", `{"` + strings.Repeat("é", 65) + `": 1}`,
			`unknown key "` + strings.Repeat("é", 64) + `…" at the top level`},
		{"a line break is escaped", `{"a\nb": 1}`, `unknown key "a\nb" at the top level`},
		{"a long handler type is cut", handler(`"type": "` + long + `"`),
			`unknown handler type "` + strings.Repeat("k", 64) + `…" at "hooks.Stop.0.hooks.0.type"`},
		{"a place with a long MCP input key is cut", handler(`"type": "mcp_tool", "server": "s", "tool": "t", "input": {"` + long + `": null}`),
			`unexpected JSON null at "` + inputAt + strings.Repeat("k", 64-len(inputAt)) + `…"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseCodexHookSlots([]byte(tc.data))
			if err == nil || err.Error() != tc.want {
				t.Fatalf("error %v, want %q", err, tc.want)
			}
			if strings.ContainsAny(err.Error(), "\r\n") {
				t.Errorf("error %q spans more than one line", err)
			}
		})
	}
}

// What codex's serde types accept is accepted, with the slots codex keys:
// optional fields given as null or of their type, the alias of
// commandWindows, keys a handler type does not declare, and an mcp_tool
// input whose values TOML can hold. A blank command still takes no slot.
func TestCodexHookTrust_SlotsAcceptWhatCodexAccepts(t *testing.T) {
	data := `{
  "description": null,
  "hooks": {
    "PreToolUse": [{"matcher": null, "hooks": [
      {"type": "command", "command": "a", "commandWindows": null, "timeout": 0, "async": true, "statusMessage": null, "additionalContextLimit": 18446744073709551615, "unknown": [null]},
      {"type": "command", "command": "b", "command_windows": "b.cmd", "timeout": null, "additionalContextLimit": null},
      {"type": "command", "command": " "}
    ]}],
    "Stop": [{"hooks": [
      {"type": "mcp_tool", "server": "s", "tool": "t", "input": {"n": 9223372036854775807, "f": -1.5, "big": 1e30, "list": [1, "x", {"k": true}]}}
    ]}]
  }
}`
	slots, err := parseCodexHookSlots([]byte(data))
	if err != nil {
		t.Fatalf("parseCodexHookSlots: %v", err)
	}
	var tails []string
	for _, slot := range slots {
		tails = append(tails, slot.Tail())
	}
	want := []string{"pre_tool_use:0:0", "pre_tool_use:0:1", "stop:0:0"}
	if !slices.Equal(tails, want) {
		t.Errorf("slot tails = %v, want %v", tails, want)
	}

	// prompt and agent handlers declare no field, so any other key is
	// accepted (whether they take a slot is the known over-count in the row
	// "Where the Codex hook trust check ... reads differently from codex" of
	// docs/tech-debt/README.md, (a), not checked here).
	for _, handler := range []string{`{"type": "prompt", "prompt": null}`, `{"type": "agent", "anything": {"x": [null]}}`} {
		if _, err := parseCodexHookSlots([]byte(`{"hooks": {"Stop": [{"hooks": [` + handler + `]}]}}`)); err != nil {
			t.Errorf("%s: unexpected error %v", handler, err)
		}
	}
}
