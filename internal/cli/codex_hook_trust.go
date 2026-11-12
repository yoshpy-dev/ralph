package cli

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	toml "github.com/pelletier/go-toml/v2"
)

// This file is the core of the codex hook trust check (plan
// 2026-11-12-codex-hook-trust-check, issue #310). For a directory a codex
// seat works in, it finds the project layers codex reads hooks from, lists
// the handlers of each layer's .codex/hooks.json under the keys codex records
// their approval at, reads [hooks.state] from codex's user config, and sorts
// each handler into approved, not approved, and turned off. The rules mirror
// codex-cli 0.166.0 (openai/codex, tag rust-v0.166.0), checked on this
// machine in docs/evidence/codex-hook-trust-worktree-2026-11-12.md. Nothing
// here runs git or codex, and nothing here prints: the doctor check and the
// launch notices build their own text from a codexHookTrustReport.

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

// codexHookSlot is one handler of a hooks.json that codex asks approval for.
// Group and Handler are positions in the JSON arrays, counted the way codex
// counts them: append_matcher_groups in
// codex-rs/hooks/src/engine/discovery.rs at rust-v0.166.0 enumerates every
// group and every handler, so a handler it skips still uses its position.
type codexHookSlot struct {
	Label   string // codex's event label, such as "pre_tool_use"
	Group   int    // position of the matcher group in the event's array
	Handler int    // position of the handler in the group's "hooks" array
}

// Tail is the part of the slot's key after the hooks.json path, such as
// "pre_tool_use:0:0".
func (s codexHookSlot) Tail() string {
	return fmt.Sprintf("%s:%d:%d", s.Label, s.Group, s.Handler)
}

// codexHooksJSONPath is the hooks.json of the layer whose .codex is in dir.
func codexHooksJSONPath(dir string) string {
	return filepath.Join(dir, ".codex", "hooks.json")
}

// codexHookKey is the [hooks.state] key codex records the approval of slot
// under when it reads the hooks of the layer whose .codex is in dir:
// "<dir>/.codex/hooks.json:<label>:<group>:<handler>" (hook_key in
// codex-rs/hooks/src/lib.rs at rust-v0.166.0, with the hooks.json's absolute
// path as the key source).
func codexHookKey(dir string, slot codexHookSlot) string {
	return codexHooksJSONPath(dir) + ":" + slot.Tail()
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

// codexGitMetadataMaxBytes bounds the git metadata files codex reads to find
// the root of a linked worktree, as MAX_GIT_METADATA_FILE_BYTES does in
// codex-rs/git-utils/src/trust.rs at rust-v0.166.0.
const codexGitMetadataMaxBytes = 64 * 1024

// codexHookCheckout is where codex reads project hooks from for a directory
// (plan 2026-11-12-codex-hook-trust-check, D1 and D7; issue #310), found the
// way codex-rs/config/src/loader/mod.rs at rust-v0.166.0 finds them
// (find_git_checkout_root, and resolve_root in
// codex-rs/git-utils/src/trust.rs), without running git.
type codexHookCheckout struct {
	Base     string // the directory, made absolute without resolving symlinks; its parent when it is not a directory
	Checkout string // the checkout root: the nearest ancestor of Base whose .git codex stops at; Base when there is none
	Repo     string // the root codex keys the checkout's hooks under: the main checkout of a linked worktree that passes codex's checks, else Checkout
}

// codexHookCheckoutOf finds the codexHookCheckout of dir: dir is made
// absolute without resolving symlinks (its parent when dir is not a
// directory), and its ancestors are walked lexically, nearest first, until
// codexHookRootAt finds the checkout root. With no such ancestor, Base is
// also the checkout and the root, so codex reads only Base's own .codex. The
// error is filepath.Abs's, for a relative dir when the working directory
// cannot be read.
func codexHookCheckoutOf(dir string) (codexHookCheckout, error) {
	base, err := filepath.Abs(dir)
	if err != nil {
		return codexHookCheckout{}, err
	}
	if info, statErr := os.Stat(base); statErr != nil || !info.IsDir() {
		base = filepath.Dir(base)
	}
	for ancestor := base; ; ancestor = filepath.Dir(ancestor) {
		if repo, found := codexHookRootAt(ancestor); found {
			return codexHookCheckout{Base: base, Checkout: ancestor, Repo: repo}, nil
		}
		if filepath.Dir(ancestor) == ancestor {
			return codexHookCheckout{Base: base, Checkout: base, Repo: base}, nil
		}
	}
}

// codexHookRootAt looks at ancestor's .git for codexHookCheckoutOf. found is
// false when the walk goes on: there is no .git, or it is a directory
// without HEAD, which codex passes over. Otherwise ancestor is the checkout
// root, and root is where codex keys its hooks: ancestor itself for a .git
// directory, the main checkout for the .git file of a linked worktree that
// passes codex's checks (codexLinkedWorktreeMain), and ancestor again for
// any other .git, since codex then keeps the hooks of the checkout itself.
func codexHookRootAt(ancestor string) (root string, found bool) {
	dotGit := filepath.Join(ancestor, ".git")
	info, err := os.Stat(dotGit)
	if err != nil {
		return "", false
	}
	if info.IsDir() {
		if _, headErr := os.Stat(filepath.Join(dotGit, "HEAD")); headErr != nil {
			return "", false
		}
		return ancestor, true
	}
	if mainRoot, ok := codexLinkedWorktreeMain(ancestor, dotGit); ok {
		return mainRoot, true
	}
	return ancestor, true
}

// codexLinkedWorktreeMain returns the main checkout codex keys the hooks of
// the linked worktree at checkout under, whose .git file is dotGit, with the
// checks resolve_root in codex-rs/git-utils/src/trust.rs at rust-v0.166.0
// makes (plan D7 (c), AC12). The .git file is a regular file, not a symlink,
// of at most codexGitMetadataMaxBytes, holding "gitdir: <path>" (a relative
// path is taken from checkout). That gitdir is a directory, not a symlink,
// and the parent of its real path is named "worktrees"; the directory above
// is the common directory. The worktree's registration in the gitdir points
// back at checkout (codexWorktreeRegisteredAt). The main checkout is three
// levels above the gitdir as spelled, the spelling codex keys hooks under,
// and it owns the common directory (codexMainOwnsCommonDir). When any check
// fails, ok is false and codex keeps the hooks of the worktree itself.
func codexLinkedWorktreeMain(checkout, dotGit string) (mainRoot string, ok bool) {
	gitDir, ok := codexReadGitdirFile(dotGit)
	if !ok {
		return "", false
	}
	if info, err := os.Lstat(gitDir); err != nil || !info.IsDir() {
		return "", false
	}
	realGitDir, err := filepath.EvalSymlinks(gitDir)
	if err != nil {
		return "", false
	}
	worktrees := filepath.Dir(realGitDir)
	if filepath.Base(worktrees) != "worktrees" {
		return "", false
	}
	commonDir := filepath.Dir(worktrees)
	if !codexWorktreeRegisteredAt(realGitDir, checkout, commonDir) {
		return "", false
	}
	mainRoot = filepath.Dir(filepath.Dir(filepath.Dir(gitDir)))
	if !codexMainOwnsCommonDir(mainRoot, commonDir) {
		return "", false
	}
	return mainRoot, true
}

// codexWorktreeRegisteredAt reports whether the worktree registration in
// realGitDir (the real path of a linked worktree's gitdir) belongs to the
// checkout at checkout and to commonDir, as resolve_root checks: its
// "gitdir" file names a path ending in .git whose directory is checkout, and
// its "commondir" file names commonDir, each compared by real path. Both
// files are read as codexReadGitMetadata reads them; a relative path in
// either is taken from realGitDir.
func codexWorktreeRegisteredAt(realGitDir, checkout, commonDir string) bool {
	backData, ok := codexReadGitMetadata(filepath.Join(realGitDir, "gitdir"))
	if !ok {
		return false
	}
	back := codexTrimASCIISpace(string(backData))
	if back == "" {
		return false
	}
	worktreeDotGit := codexJoinGitPath(realGitDir, back)
	if filepath.Base(worktreeDotGit) != ".git" {
		return false
	}
	commonData, ok := codexReadGitMetadata(filepath.Join(realGitDir, "commondir"))
	if !ok {
		return false
	}
	common := codexTrimASCIISpace(string(commonData))
	if common == "" {
		return false
	}
	registered, registeredErr := filepath.EvalSymlinks(filepath.Dir(worktreeDotGit))
	actual, actualErr := filepath.EvalSymlinks(checkout)
	linkedCommon, commonErr := filepath.EvalSymlinks(codexJoinGitPath(realGitDir, common))
	return registeredErr == nil && actualErr == nil && commonErr == nil &&
		registered == actual && linkedCommon == commonDir
}

// codexMainOwnsCommonDir reports whether mainRoot's .git is commonDir by real
// path, as resolve_root checks before it keys a worktree's hooks under
// mainRoot: a .git directory, or a .git file whose gitdir line names it (a
// main checkout made with --separate-git-dir).
func codexMainOwnsCommonDir(mainRoot, commonDir string) bool {
	mainDotGit := filepath.Join(mainRoot, ".git")
	info, err := os.Stat(mainDotGit)
	if err != nil {
		return false
	}
	mainGitDir := mainDotGit
	if !info.IsDir() {
		var ok bool
		if mainGitDir, ok = codexReadGitdirFile(mainDotGit); !ok {
			return false
		}
	}
	realMainGitDir, err := filepath.EvalSymlinks(mainGitDir)
	return err == nil && realMainGitDir == commonDir
}

// codexReadGitdirFile reads the path of a "gitdir: <path>" file at path (a
// .git file), as read_gitdir_file in codex-rs/git-utils/src/trust.rs does: a
// relative path is taken from path's directory. ok is false when the file
// cannot be read as codexReadGitMetadata reads it, has no gitdir line, or
// names no path.
func codexReadGitdirFile(path string) (gitDir string, ok bool) {
	data, ok := codexReadGitMetadata(path)
	if !ok {
		return "", false
	}
	target, found := strings.CutPrefix(codexTrimASCIISpace(string(data)), "gitdir:")
	target = codexTrimASCIISpace(target)
	if !found || target == "" {
		return "", false
	}
	return codexJoinGitPath(filepath.Dir(path), target), true
}

// codexReadGitMetadata reads a git metadata file as read_metadata_file in
// codex-rs/git-utils/src/trust.rs does: a regular file, not a symlink, of at
// most codexGitMetadataMaxBytes.
func codexReadGitMetadata(path string) ([]byte, bool) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > codexGitMetadataMaxBytes {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > codexGitMetadataMaxBytes {
		return nil, false
	}
	return data, true
}

// codexTrimASCIISpace trims what Rust's trim_ascii trims: space, tab, line
// feed, form feed, and carriage return.
func codexTrimASCIISpace(s string) string {
	return strings.Trim(s, " \t\n\f\r")
}

// codexJoinGitPath is p taken from base: p itself when it is absolute.
func codexJoinGitPath(base, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(base, p)
}

// codexHookSpellings returns the checkouts codex may see for dir: the one of
// dir as spelled, then, when it differs, the one of dir with its symlinks
// resolved. A codex seat's working directory may be either spelling, and
// codex keys the hooks of a main checkout opened through a symlink under the
// symlink's path, so the check asks for an approval under both (plan
// 2026-11-12-codex-hook-trust-check, D1 and AC9). dir is made absolute
// first, and the symlinks of that absolute path are resolved:
// filepath.EvalSymlinks leaves a relative dir relative (doctor's "." stays
// "."), and making it absolute afterwards takes os.Getwd, which returns $PWD
// when it names the same directory, so with $PWD in the symlink's spelling
// both spellings would come out alike and the real one would be lost. When
// dir's symlinks cannot be resolved (dir does not exist), only the first is
// returned.
func codexHookSpellings(dir string) ([]codexHookCheckout, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	lexical, err := codexHookCheckoutOf(absDir)
	if err != nil {
		return nil, err
	}
	spellings := []codexHookCheckout{lexical}
	resolvedDir, err := filepath.EvalSymlinks(absDir)
	if err != nil {
		return spellings, nil
	}
	if resolved, err := codexHookCheckoutOf(resolvedDir); err == nil && resolved != lexical {
		spellings = append(spellings, resolved)
	}
	return spellings, nil
}

// codexHookLayer is one project layer codex reads hooks from: a directory
// from the checkout root down to the seat's directory that holds a .codex
// directory in the checkout (discover_project_layers in
// codex-rs/config/src/loader/mod.rs at rust-v0.166.0). codex reads the
// layer's hooks.json from the same place under the root it keys hooks under
// (root_checkout_hooks_folder_for_dir: the main checkout's, for a linked
// worktree) and keys its hooks under that hooks.json's path (plan D7 (a),
// AC10).
type codexHookLayer struct {
	Rel   string               // the layer's directory relative to the checkout root; "." for the root layer
	Dirs  []string             // <root>/<Rel> under each spelling, without repeats: where its .codex/hooks.json is and where to open codex to approve its hooks
	State codexHookSourceState // how reading HooksPath went
	Err   error                // why, for codexHookSourceUnreadable and codexHookSourceInvalid
	Slots []codexHookSlot      // every handler codex asks approval for, once HooksPath was read
}

// HooksPath is the hooks.json the layer was read from: the one in Dirs[0].
func (l codexHookLayer) HooksPath() string {
	return codexHooksJSONPath(l.Dirs[0])
}

// layerRels lists the directories from c's checkout root down to its base,
// relative to the checkout root, root first.
func (c codexHookCheckout) layerRels() []string {
	rels := []string{"."}
	rel, err := filepath.Rel(c.Checkout, c.Base)
	if err != nil || rel == "." {
		return rels
	}
	parts := strings.Split(rel, string(filepath.Separator))
	for i := range parts {
		rels = append(rels, filepath.Join(parts[:i+1]...))
	}
	return rels
}

// layers lists c's project layers, root first, each with the one directory
// of c's spelling. The root layer is always listed: without a .codex
// directory in the checkout root codex reads no hooks.json there, so its
// directory is then the checkout root itself, which has no hooks.json to
// read, and the check reports that file as missing. A nested directory is a
// layer only when it has a .codex directory in the checkout.
func (c codexHookCheckout) layers() []codexHookLayer {
	var layers []codexHookLayer
	for _, rel := range c.layerRels() {
		dotCodex, err := os.Stat(filepath.Join(c.Checkout, rel, ".codex"))
		switch {
		case err == nil && dotCodex.IsDir():
			layers = append(layers, codexHookLayer{Rel: rel, Dirs: []string{filepath.Join(c.Repo, rel)}})
		case rel == ".":
			layers = append(layers, codexHookLayer{Rel: rel, Dirs: []string{c.Checkout}})
		}
	}
	return layers
}

// codexHookLayersOf merges the layers of every spelling by their place under
// the checkout root, keeping the first spelling's order: the spellings name
// the same directories, so a layer is read once, from its first directory,
// and its hooks are keyed under each.
func codexHookLayersOf(spellings []codexHookCheckout) []codexHookLayer {
	var merged []codexHookLayer
	for _, spelling := range spellings {
		for _, layer := range spelling.layers() {
			i := slices.IndexFunc(merged, func(m codexHookLayer) bool { return m.Rel == layer.Rel })
			if i < 0 {
				merged = append(merged, layer)
				continue
			}
			if !slices.Contains(merged[i].Dirs, layer.Dirs[0]) {
				merged[i].Dirs = append(merged[i].Dirs, layer.Dirs[0])
			}
		}
	}
	for i := range merged {
		merged[i].Slots, merged[i].State, merged[i].Err = readCodexHookSlots(merged[i].HooksPath())
	}
	return merged
}

// codexHookStateEntry is one [hooks.state."<key>"] table of codex's user
// config. codex writes trusted_hash when a hook is approved and enabled =
// false when it is turned off (hook_trust_status and hook_enabled in
// codex-rs/hooks/src/engine/discovery.rs at rust-v0.166.0); other keys are
// ignored.
type codexHookStateEntry struct {
	TrustedHash string `toml:"trusted_hash"`
	Enabled     *bool  `toml:"enabled"`
}

// codexHookStatesConfig decodes only [hooks.state] of codex's user config. It
// is separate from codexUserConfig so neither decode depends on what the
// other reads (plan D4): nothing under [hooks] reaches Check 11b's decode,
// and a key Check 11b reads cannot break this one.
type codexHookStatesConfig struct {
	Hooks struct {
		State map[string]codexHookStateEntry `toml:"state"`
	} `toml:"hooks"`
}

// codexHookSourceState is how reading one input of the check went: a
// hooks.json codex reads, or codex's user config.
type codexHookSourceState int

const (
	codexHookSourceRead       codexHookSourceState = iota // read and decoded
	codexHookSourceMissing                                // the file does not exist
	codexHookSourceUnreadable                             // not a regular file, larger than 1 MiB, or a stat, open, or read error
	codexHookSourceInvalid                                // hooks.json not of the shape codex reads; config.toml not decodable
)

// readCodexHookSlots reads the hooks.json at path through
// readCodexFileBounded (codex itself sets no size limit; a hooks.json over 1
// MiB is reported as unreadable) and lists its slots.
func readCodexHookSlots(path string) ([]codexHookSlot, codexHookSourceState, error) {
	data, exists, err := readCodexFileBounded(path)
	if err != nil {
		return nil, codexHookSourceUnreadable, err
	}
	if !exists {
		return nil, codexHookSourceMissing, nil
	}
	slots, err := parseCodexHookSlots(data)
	if err != nil {
		return nil, codexHookSourceInvalid, err
	}
	return slots, codexHookSourceRead, nil
}

// readCodexHookStates reads [hooks.state] from codex's user config at path,
// through readCodexFileBounded as Check 11b reads the same file
// (readCodexUserConfig). A missing file is codexHookSourceMissing with no
// error and no states. A decode failure is a *codexConfigDecodeError, which
// never carries the config's text.
func readCodexHookStates(path string) (map[string]codexHookStateEntry, codexHookSourceState, error) {
	data, exists, err := readCodexFileBounded(path)
	if err != nil {
		return nil, codexHookSourceUnreadable, err
	}
	if !exists {
		return nil, codexHookSourceMissing, nil
	}
	var cfg codexHookStatesConfig
	if decodeErr := toml.Unmarshal(data, &cfg); decodeErr != nil {
		return nil, codexHookSourceInvalid, codexConfigDecodeErrorOf(decodeErr)
	}
	return cfg.Hooks.State, codexHookSourceRead, nil
}

// codexConfigDecodeErrorOf wraps a go-toml decode failure as
// readCodexUserConfig does: the position when go-toml reports one, never its
// message, which can repeat the config's keys and values.
func codexConfigDecodeErrorOf(err error) *codexConfigDecodeError {
	wrapped := &codexConfigDecodeError{}
	var tomlErr *toml.DecodeError
	if errors.As(err, &tomlErr) {
		wrapped.HasPosition = true
		wrapped.Line, wrapped.Column = tomlErr.Position()
	}
	return wrapped
}

// codexHookTrust is what codex does with one hook handler in a seat.
type codexHookTrust int

const (
	codexHookApproved    codexHookTrust = iota // runs: a trusted_hash is recorded and the hook is not turned off
	codexHookNotApproved                       // does not run: no [hooks.state] entry, or one without a trusted_hash
	codexHookOff                               // does not run: turned off with enabled = false
)

// codexHookTrustUnder classifies the hook whose key is key from its
// [hooks.state] entry (plan D2). enabled = false wins over the hash, since
// codex does not run a turned-off hook whether or not it is approved. The
// hash itself is not compared with the hook's: codex computes it over its own
// TOML normalization, so a hook changed after its approval still counts as
// approved here, and codex shows "Hooks need review" for it in the seat.
func codexHookTrustUnder(states map[string]codexHookStateEntry, key string) codexHookTrust {
	entry, ok := states[key]
	switch {
	case !ok:
		return codexHookNotApproved
	case entry.Enabled != nil && !*entry.Enabled:
		return codexHookOff
	case entry.TrustedHash == "":
		return codexHookNotApproved
	default:
		return codexHookApproved
	}
}

// classifyCodexHookSlot classifies slot of the layer in dirs over every
// spelling (plan D1, AC9): approved only when it is approved under each;
// otherwise not approved when one spelling has no approval, and turned off
// when every spelling without an approval turns it off.
func classifyCodexHookSlot(states map[string]codexHookStateEntry, dirs []string, slot codexHookSlot) codexHookTrust {
	result := codexHookApproved
	for _, dir := range dirs {
		switch codexHookTrustUnder(states, codexHookKey(dir, slot)) {
		case codexHookNotApproved:
			return codexHookNotApproved
		case codexHookOff:
			result = codexHookOff
		}
	}
	return result
}

// codexHookTrustItem is one hook handler of a codexHookTrustReport.
type codexHookTrustItem struct {
	Layer int      // the index of its layer in codexHookTrustReport.Layers
	Tail  string   // the key after the hooks.json path, such as "pre_tool_use:0:0"
	Keys  []string // the full key under each of the layer's Dirs, in order
}

// codexHookTrustReport is what codexHookTrustFor found. Each input's outcome
// is reported apart, so a caller can tell a missing file from one it could
// not read or decode.
type codexHookTrustReport struct {
	Layers      []codexHookLayer     // the root layer first, then each nested layer down to the directory; never empty
	ConfigPath  string               // codex's user config (codexSandboxEnv.ConfigPath)
	ConfigState codexHookSourceState // how reading ConfigPath went
	ConfigErr   error                // why, for codexHookSourceUnreadable; a *codexConfigDecodeError for codexHookSourceInvalid
	Approved    []codexHookTrustItem // filled only when Classified
	NotApproved []codexHookTrustItem // filled only when Classified; every slot when the config is missing
	Off         []codexHookTrustItem // filled only when Classified
}

// HooksProblems lists the layers whose hooks.json the check could not read
// as codex reads one: the root layer's when it was not read, and a nested
// layer's when it could not be read or is invalid. A nested layer without a
// hooks.json is normal (its .codex may hold only a config.toml), so it is
// not one.
func (r codexHookTrustReport) HooksProblems() []codexHookLayer {
	var problems []codexHookLayer
	for _, layer := range r.Layers {
		if layer.State == codexHookSourceRead || (layer.State == codexHookSourceMissing && layer.Rel != ".") {
			continue
		}
		problems = append(problems, layer)
	}
	return problems
}

// Classified reports whether every slot is in one of Approved, NotApproved,
// and Off: no layer has a problem (HooksProblems), and codex's user config
// was read or does not exist (then no hook is approved).
func (r codexHookTrustReport) Classified() bool {
	return len(r.HooksProblems()) == 0 &&
		(r.ConfigState == codexHookSourceRead || r.ConfigState == codexHookSourceMissing)
}

// SlotCount is the number of handlers codex asks approval for, over every
// layer read.
func (r codexHookTrustReport) SlotCount() int {
	n := 0
	for _, layer := range r.Layers {
		n += len(layer.Slots)
	}
	return n
}

// ReadPaths lists the hooks.json of every layer that was read, root first.
func (r codexHookTrustReport) ReadPaths() []string {
	var paths []string
	for _, layer := range r.Layers {
		if layer.State == codexHookSourceRead {
			paths = append(paths, layer.HooksPath())
		}
	}
	return paths
}

// SlotPaths lists the hooks.json of every layer that holds a handler codex
// asks approval for, root first.
func (r codexHookTrustReport) SlotPaths() []string {
	var paths []string
	for _, layer := range r.Layers {
		if len(layer.Slots) > 0 {
			paths = append(paths, layer.HooksPath())
		}
	}
	return paths
}

// ItemName names item in a list of hooks: its tail when one hooks.json holds
// every slot, else its hooks.json relative to the root it is keyed under
// followed by the tail, such as "sub/.codex/hooks.json:pre_tool_use:0:0".
func (r codexHookTrustReport) ItemName(item codexHookTrustItem) string {
	if len(r.SlotPaths()) < 2 {
		return item.Tail
	}
	return filepath.Join(r.Layers[item.Layer].Rel, ".codex", "hooks.json") + ":" + item.Tail
}

// ApprovalDirs is where to open codex to approve the hooks in NotApproved:
// the directories, one per spelling, of the deepest layer holding one, since
// codex opened there reads every layer from the checkout root down to it.
// It is nil when NotApproved is empty.
func (r codexHookTrustReport) ApprovalDirs() []string {
	if len(r.NotApproved) == 0 {
		return nil
	}
	deepest := 0
	for _, item := range r.NotApproved {
		deepest = max(deepest, item.Layer)
	}
	return r.Layers[deepest].Dirs
}

// codexHookTrustFor checks the hook approvals a codex seat working in dir
// would see (plan 2026-11-12-codex-hook-trust-check, issue #310). It reads
// the hooks.json of every layer codex reads for dir (codexHookLayersOf, over
// every spelling of dir) and [hooks.state] of codex's user config at
// env.ConfigPath, and sorts every handler into approved, not approved, and
// turned off over all spellings. It prints nothing and never fails: a
// missing, unreadable, or invalid file is reported in the result for the
// caller to word.
func codexHookTrustFor(dir string, env codexSandboxEnv) codexHookTrustReport {
	report := codexHookTrustReport{ConfigPath: env.ConfigPath}
	spellings, err := codexHookSpellings(dir)
	if err != nil {
		report.Layers = []codexHookLayer{{Rel: ".", Dirs: []string{dir}, State: codexHookSourceUnreadable, Err: err}}
	} else {
		report.Layers = codexHookLayersOf(spellings)
	}
	var states map[string]codexHookStateEntry
	states, report.ConfigState, report.ConfigErr = readCodexHookStates(env.ConfigPath)
	if report.Classified() {
		report.Approved, report.NotApproved, report.Off = classifyCodexHookSlots(states, report.Layers)
	}
	return report
}

// classifyCodexHookSlots splits the slots of every layer by
// classifyCodexHookSlot, keeping the layers' and the slots' order.
func classifyCodexHookSlots(states map[string]codexHookStateEntry, layers []codexHookLayer) (approved, notApproved, off []codexHookTrustItem) {
	for i, layer := range layers {
		for _, slot := range layer.Slots {
			item := codexHookTrustItem{Layer: i, Tail: slot.Tail()}
			for _, dir := range layer.Dirs {
				item.Keys = append(item.Keys, codexHookKey(dir, slot))
			}
			switch classifyCodexHookSlot(states, layer.Dirs, slot) {
			case codexHookApproved:
				approved = append(approved, item)
			case codexHookNotApproved:
				notApproved = append(notApproved, item)
			case codexHookOff:
				off = append(off, item)
			}
		}
	}
	return approved, notApproved, off
}
