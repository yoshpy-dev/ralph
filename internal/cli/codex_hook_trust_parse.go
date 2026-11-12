package cli

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// This file is the hooks.json reader of the codex hook trust check (plan
// 2026-11-12-codex-hook-trust-check, issue #310). It mirrors the serde types
// codex reads a hooks.json with at rust-v0.166.0 and lists the handlers codex
// asks approval for as codexHookSlot values. codex_hook_trust.go holds the
// rest of the check: the layers, the approval state, and the classification.

// codexHookEventLabels maps the event names of a hooks.json to the snake_case
// labels codex puts into a hook's key (hook_event_key_label in
// codex-rs/hooks/src/lib.rs at rust-v0.166.0). An event that is not one of
// these 14 is not one codex runs, so its handlers get no key.
var codexHookEventLabels = map[string]string{
	"PreToolUse":        "pre_tool_use",
	"PermissionRequest": "permission_request",
	"PostToolUse":       "post_tool_use",
	"PreCompact":        "pre_compact",
	"PostCompact":       "post_compact",
	"PreModelRequest":   "pre_model_request",
	"PostModelRequest":  "post_model_request",
	"SessionStart":      "session_start",
	"SessionEnd":        "session_end",
	"UserPromptSubmit":  "user_prompt_submit",
	"SubagentStart":     "subagent_start",
	"SubagentStop":      "subagent_stop",
	"Stop":              "stop",
	"Interrupt":         "interrupt",
}

// parseCodexHookSlots lists the handlers of a hooks.json that codex asks
// approval for, ordered by label, then group, then handler. It refuses what
// codex's serde types for the file refuse at rust-v0.166.0 (HooksFile in
// codex-rs/config/src/hook_config.rs; HookEventsToml, MatcherGroup, and
// HookHandlerConfig in codex-rs/protocol/src/hooks.rs), since codex then
// loads no hook from the file (plan D7 (b), AC11): data that is not UTF-8 or
// not JSON, a top level that is not an object or has a key other than
// "description" and "hooks", a key of the shape below given twice, and a
// value of the wrong JSON type, null included, where serde has no default
// for null ("hooks", an event's array, a group, a group's "hooks", a
// handler). A handler needs a "type" codex knows and the fields that type
// requires (codexHookHandlerTypes). An event codex does not know is skipped
// unread, as serde skips it, and so is a "command" handler whose command is
// empty or blank: codex skips it with a warning and never runs it, so it
// needs no approval (its position still counts for the next handler). serde
// would also read an object of this shape from a JSON array; that form is
// refused here, so such a file is reported as invalid although codex loads
// it.
func parseCodexHookSlots(data []byte) ([]codexHookSlot, error) {
	if !utf8.Valid(data) {
		return nil, errors.New("not valid UTF-8")
	}
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, codexHooksJSONError(err)
	}
	switch kind := codexJSONKind(raw); kind {
	case "object":
	case "null":
		return nil, errors.New("not a JSON object")
	default:
		return nil, fmt.Errorf("unexpected JSON %s at the top level", kind)
	}
	members, err := codexJSONMembers(raw, "", map[string]string{"description": "description", "hooks": "hooks"})
	if err != nil {
		return nil, err
	}
	var slots []codexHookSlot
	for _, m := range members {
		switch m.Key {
		case "description":
			err = codexJSONCheck(m.Value, m.Key, codexJSONOptionalString)
		case "hooks":
			slots, err = parseCodexHookEvents(m.Value, m.Key)
		default:
			err = fmt.Errorf("unknown key %q at the top level", m.Key)
		}
		if err != nil {
			return nil, err
		}
	}
	slices.SortFunc(slots, func(a, b codexHookSlot) int {
		return cmp.Or(cmp.Compare(a.Label, b.Label), cmp.Compare(a.Group, b.Group), cmp.Compare(a.Handler, b.Handler))
	})
	return slots, nil
}

// parseCodexHookEvents reads the "hooks" object of a hooks.json at at: an
// array of matcher groups for each event codex knows.
func parseCodexHookEvents(raw json.RawMessage, at string) ([]codexHookSlot, error) {
	if codexJSONKind(raw) != "object" {
		return nil, codexJSONUnexpected(raw, at)
	}
	events, err := codexJSONMembers(raw, at, codexHookEventLabels)
	if err != nil {
		return nil, err
	}
	var slots []codexHookSlot
	for _, event := range events {
		label, known := codexHookEventLabels[event.Key]
		if !known {
			continue
		}
		eventAt := codexJSONPath(at, event.Key)
		groups, err := codexJSONArray(event.Value, eventAt)
		if err != nil {
			return nil, err
		}
		for g, group := range groups {
			groupSlots, err := parseCodexHookGroup(group, codexJSONPath(eventAt, strconv.Itoa(g)), label, g)
			if err != nil {
				return nil, err
			}
			slots = append(slots, groupSlots...)
		}
	}
	return slots, nil
}

// parseCodexHookGroup reads the matcher group at position g of the event
// labelled label, at at. A group without "hooks" has no handler, as serde
// defaults it.
func parseCodexHookGroup(raw json.RawMessage, at, label string, g int) ([]codexHookSlot, error) {
	if codexJSONKind(raw) != "object" {
		return nil, codexJSONUnexpected(raw, at)
	}
	members, err := codexJSONMembers(raw, at, map[string]string{"matcher": "matcher", "hooks": "hooks"})
	if err != nil {
		return nil, err
	}
	var handlers []json.RawMessage
	for _, m := range members {
		switch m.Key {
		case "matcher":
			err = codexJSONCheck(m.Value, codexJSONPath(at, m.Key), codexJSONOptionalString)
		case "hooks":
			handlers, err = codexJSONArray(m.Value, codexJSONPath(at, m.Key))
		}
		if err != nil {
			return nil, err
		}
	}
	var slots []codexHookSlot
	for h, handler := range handlers {
		needsApproval, err := parseCodexHookHandler(handler, codexJSONPath(codexJSONPath(at, "hooks"), strconv.Itoa(h)))
		if err != nil {
			return nil, err
		}
		if needsApproval {
			slots = append(slots, codexHookSlot{Label: label, Group: g, Handler: h})
		}
	}
	return slots, nil
}

// codexHookHandlerField is one field of a variant of codex's
// HookHandlerConfig: the field a key fills (an alias fills the same field,
// so giving both is a duplicate), whether the variant requires it, and the
// JSON type serde accepts for it.
type codexHookHandlerField struct {
	Field    string
	Required bool
	Type     codexJSONType
}

// codexHookHandlerTypes are the variants of HookHandlerConfig in
// codex-rs/protocol/src/hooks.rs at rust-v0.166.0, by their "type" tag, with
// the fields each declares. Keys a variant does not declare are ignored, as
// serde ignores them.
var codexHookHandlerTypes = map[string]map[string]codexHookHandlerField{
	"command": {
		"command":                {Field: "command", Required: true, Type: codexJSONString},
		"commandWindows":         {Field: "command_windows", Type: codexJSONOptionalString},
		"command_windows":        {Field: "command_windows", Type: codexJSONOptionalString},
		"timeout":                {Field: "timeout_sec", Type: codexJSONOptionalUint},
		"async":                  {Field: "async", Type: codexJSONBool},
		"statusMessage":          {Field: "status_message", Type: codexJSONOptionalString},
		"additionalContextLimit": {Field: "additional_context_limit", Type: codexJSONOptionalUint},
	},
	"mcp_tool": {
		"server":        {Field: "server", Required: true, Type: codexJSONString},
		"tool":          {Field: "tool", Required: true, Type: codexJSONString},
		"input":         {Field: "input", Type: codexJSONTOMLObject},
		"timeout":       {Field: "timeout_sec", Type: codexJSONOptionalUint},
		"statusMessage": {Field: "status_message", Type: codexJSONOptionalString},
	},
	"prompt": {},
	"agent":  {},
}

// parseCodexHookHandler reads the handler at at and reports whether codex
// asks approval for it: every handler of a known type does, except a
// "command" handler whose command is empty or blank.
func parseCodexHookHandler(raw json.RawMessage, at string) (needsApproval bool, err error) {
	if codexJSONKind(raw) != "object" {
		return false, codexJSONUnexpected(raw, at)
	}
	members, err := codexJSONMembers(raw, at, map[string]string{"type": "type"})
	if err != nil {
		return false, err
	}
	typeName, err := codexHookHandlerType(members, at)
	if err != nil {
		return false, err
	}
	fields := codexHookHandlerTypes[typeName]
	seen := map[string]bool{}
	var command string
	for _, m := range members {
		field, declared := fields[m.Key]
		if !declared {
			continue
		}
		if seen[field.Field] {
			return false, fmt.Errorf("duplicate key %q %s", m.Key, codexJSONWhere(at))
		}
		seen[field.Field] = true
		if err := codexJSONCheck(m.Value, codexJSONPath(at, m.Key), field.Type); err != nil {
			return false, err
		}
		if m.Key == "command" {
			if err := json.Unmarshal(m.Value, &command); err != nil {
				return false, err
			}
		}
	}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		if fields[key].Required && !seen[fields[key].Field] {
			return false, fmt.Errorf("missing key %q %s", key, codexJSONWhere(at))
		}
	}
	return typeName != "command" || strings.TrimSpace(command) != "", nil
}

// codexHookHandlerType returns the "type" tag of the handler whose members
// are members, at at: a string naming one of codexHookHandlerTypes.
func codexHookHandlerType(members []codexJSONMember, at string) (string, error) {
	typeAt := codexJSONPath(at, "type")
	for _, m := range members {
		if m.Key != "type" {
			continue
		}
		if err := codexJSONCheck(m.Value, typeAt, codexJSONString); err != nil {
			return "", err
		}
		var typeName string
		if err := json.Unmarshal(m.Value, &typeName); err != nil {
			return "", err
		}
		if _, known := codexHookHandlerTypes[typeName]; !known {
			return "", fmt.Errorf("unknown handler type %q %s", typeName, codexJSONWhere(typeAt))
		}
		return typeName, nil
	}
	return "", fmt.Errorf("missing key %q %s", "type", codexJSONWhere(at))
}

// codexHooksJSONError renders a hooks.json decode failure without Go's type
// names: the byte offset of a syntax error.
func codexHooksJSONError(err error) error {
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return fmt.Errorf("not valid JSON (at byte %d)", syntaxErr.Offset)
	}
	return err
}

// codexJSONMember is one key and value of a JSON object, in file order.
type codexJSONMember struct {
	Key   string
	Value json.RawMessage
}

// codexJSONMembers returns the members of raw, a valid JSON object at at, in
// file order. fields maps each key serde reads into a field to that field;
// a field given twice is an error, as serde refuses a duplicate field. Other
// keys may repeat, since serde skips them.
func codexJSONMembers(raw json.RawMessage, at string, fields map[string]string) ([]codexJSONMember, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	var members []codexJSONMember
	seen := map[string]bool{}
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, _ := token.(string)
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return nil, err
		}
		if field, read := fields[key]; read {
			if seen[field] {
				return nil, fmt.Errorf("duplicate key %q %s", key, codexJSONWhere(at))
			}
			seen[field] = true
		}
		members = append(members, codexJSONMember{Key: key, Value: value})
	}
	return members, nil
}

// codexJSONArray returns the elements of raw, which must be a JSON array (not
// null) at at.
func codexJSONArray(raw json.RawMessage, at string) ([]json.RawMessage, error) {
	if codexJSONKind(raw) != "array" {
		return nil, codexJSONUnexpected(raw, at)
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// codexJSONType is a type serde accepts for a field of a hooks.json handler.
type codexJSONType int

const (
	codexJSONString         codexJSONType = iota // String: a JSON string
	codexJSONOptionalString                      // Option<String>: a string or null
	codexJSONOptionalUint                        // Option<u64>, Option<usize>: an integer from 0 to 2^64-1, or null
	codexJSONBool                                // bool: true or false
	codexJSONTOMLObject                          // an MCP tool's input: an object whose values TOML can hold (deserialize_mcp_tool_input)
)

// codexJSONCheck checks that raw, at at, has the JSON type want.
func codexJSONCheck(raw json.RawMessage, at string, want codexJSONType) error {
	kind := codexJSONKind(raw)
	switch {
	case kind == "null" && (want == codexJSONOptionalString || want == codexJSONOptionalUint):
		return nil
	case kind == "string" && (want == codexJSONString || want == codexJSONOptionalString):
		return nil
	case kind == "boolean" && want == codexJSONBool:
		return nil
	case kind == "number" && want == codexJSONOptionalUint:
		return codexJSONCheckUint(raw, at)
	case kind == "object" && want == codexJSONTOMLObject:
		return codexJSONCheckTOMLObject(raw, at)
	}
	return codexJSONUnexpected(raw, at)
}

// codexJSONCheckUint checks that the JSON number raw, at at, is one serde
// reads as a u64: an integer from 0 to 2^64-1 written without a fraction,
// an exponent, or a sign (serde_json reads -0 as a float).
func codexJSONCheckUint(raw json.RawMessage, at string) error {
	number := string(bytes.TrimSpace(raw))
	if !strings.ContainsAny(number, ".eE-") {
		if _, err := strconv.ParseUint(number, 10, 64); err == nil {
			return nil
		}
	}
	return fmt.Errorf("not an integer from 0 to 2^64-1 %s", codexJSONWhere(at))
}

// codexJSONCheckTOMLObject checks an MCP tool handler's input, raw at at, as
// deserialize_mcp_tool_input in codex-rs/protocol/src/hooks.rs does: no
// null anywhere in it, and no integer above 2^63-1.
func codexJSONCheckTOMLObject(raw json.RawMessage, at string) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var input map[string]any
	if err := dec.Decode(&input); err != nil {
		return err
	}
	return codexJSONCheckTOMLValue(input, at)
}

// codexJSONCheckTOMLValue walks a value of codexJSONCheckTOMLObject at at.
func codexJSONCheckTOMLValue(value any, at string) error {
	switch v := value.(type) {
	case nil:
		return fmt.Errorf("unexpected JSON null %s", codexJSONWhere(at))
	case json.Number:
		if n, err := strconv.ParseUint(v.String(), 10, 64); err == nil && n > math.MaxInt64 {
			return fmt.Errorf("an integer above 2^63-1 %s", codexJSONWhere(at))
		}
	case []any:
		for i, item := range v {
			if err := codexJSONCheckTOMLValue(item, codexJSONPath(at, strconv.Itoa(i))); err != nil {
				return err
			}
		}
	case map[string]any:
		for _, key := range slices.Sorted(maps.Keys(v)) {
			if err := codexJSONCheckTOMLValue(v[key], codexJSONPath(at, key)); err != nil {
				return err
			}
		}
	}
	return nil
}

// codexJSONKind names the JSON type of raw, a valid JSON value: "object",
// "array", "string", "number", "boolean", or "null".
func codexJSONKind(raw json.RawMessage) string {
	trimmed := bytes.TrimLeft(raw, " \t\r\n")
	if len(trimmed) == 0 {
		return "nothing"
	}
	switch trimmed[0] {
	case '{':
		return "object"
	case '[':
		return "array"
	case '"':
		return "string"
	case 't', 'f':
		return "boolean"
	case 'n':
		return "null"
	default:
		return "number"
	}
}

// codexJSONUnexpected is the error for raw, of the wrong JSON type, at at.
func codexJSONUnexpected(raw json.RawMessage, at string) error {
	return fmt.Errorf("unexpected JSON %s %s", codexJSONKind(raw), codexJSONWhere(at))
}

// codexJSONWhere places an error: `at "<at>"`, or "at the top level".
func codexJSONWhere(at string) string {
	if at == "" {
		return "at the top level"
	}
	return fmt.Sprintf("at %q", at)
}

// codexJSONPath is the place of key (a member's key or an element's index)
// inside at, such as "hooks.PreToolUse.0".
func codexJSONPath(at, key string) string {
	if at == "" {
		return key
	}
	return at + "." + key
}
