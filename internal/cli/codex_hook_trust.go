package cli

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// This file is the core of the codex hook trust check (plan
// 2026-11-12-codex-hook-trust-check, issue #310). For a directory a codex
// seat works in, it finds the root codex reads project hooks from, lists the
// handlers of that root's .codex/hooks.json under the keys codex records
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

// codexHooksJSONPath is the hooks.json codex reads for root.
func codexHooksJSONPath(root string) string {
	return filepath.Join(root, ".codex", "hooks.json")
}

// codexHookKey is the [hooks.state] key codex records the approval of slot
// under when it reads hooks from root:
// "<root>/.codex/hooks.json:<label>:<group>:<handler>" (hook_key in
// codex-rs/hooks/src/lib.rs at rust-v0.166.0, with the hooks.json's absolute
// path as the key source).
func codexHookKey(root string, slot codexHookSlot) string {
	return codexHooksJSONPath(root) + ":" + slot.Tail()
}

// codexHooksFile, codexHookGroup, and codexHookHandler decode the parts of a
// hooks.json the check reads: {"hooks": {"<Event>": [{"hooks": [{"type":
// ..., "command": ...}]}]}}. Other keys (a top-level "description", a
// group's "matcher", a handler's "timeout") are ignored.
type codexHooksFile struct {
	Hooks map[string][]codexHookGroup `json:"hooks"`
}

type codexHookGroup struct {
	Hooks []codexHookHandler `json:"hooks"`
}

type codexHookHandler struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// parseCodexHookSlots lists the handlers of a hooks.json that codex asks
// approval for, ordered by label, then group, then handler. Events codex
// does not know are left out, and so is a "command" handler whose command is
// empty or blank: codex skips it with a warning and never runs it, so it
// needs no approval (its position still counts for the next handler). data
// that is not a JSON object of the shape above is an error.
func parseCodexHookSlots(data []byte) ([]codexHookSlot, error) {
	var file *codexHooksFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, codexHooksJSONError(err)
	}
	if file == nil {
		return nil, errors.New("not a JSON object")
	}
	var slots []codexHookSlot
	for event, groups := range file.Hooks {
		label, known := codexHookEventLabels[event]
		if !known {
			continue
		}
		for g, group := range groups {
			for h, handler := range group.Hooks {
				if handler.Type == "command" && strings.TrimSpace(handler.Command) == "" {
					continue
				}
				slots = append(slots, codexHookSlot{Label: label, Group: g, Handler: h})
			}
		}
	}
	slices.SortFunc(slots, func(a, b codexHookSlot) int {
		return cmp.Or(cmp.Compare(a.Label, b.Label), cmp.Compare(a.Group, b.Group), cmp.Compare(a.Handler, b.Handler))
	})
	return slots, nil
}

// codexHooksJSONError renders a hooks.json decode failure without Go's type
// names: the byte offset of a syntax error, or the JSON kind found where the
// shape codex reads expects another.
func codexHooksJSONError(err error) error {
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return fmt.Errorf("not valid JSON (at byte %d)", syntaxErr.Offset)
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		if typeErr.Field == "" {
			return fmt.Errorf("unexpected JSON %s at the top level", typeErr.Value)
		}
		return fmt.Errorf("unexpected JSON %s at %q", typeErr.Value, typeErr.Field)
	}
	return err
}

// codexGitMetadataMaxBytes bounds the .git file codexLinkedWorktreeMain
// reads, as MAX_GIT_METADATA_FILE_BYTES does in
// codex-rs/git-utils/src/trust.rs at rust-v0.166.0.
const codexGitMetadataMaxBytes = 64 * 1024

// codexHookRoot returns the directory whose .codex/hooks.json codex reads
// for a session in dir, mirroring resolve_root in
// codex-rs/git-utils/src/trust.rs at rust-v0.166.0 without running git (plan
// 2026-11-12-codex-hook-trust-check, D1 and AC5; issue #310). dir is made
// absolute without resolving symlinks (its parent when dir is not a
// directory), and its ancestors are walked lexically, nearest first, until
// codexHookRootAt finds the root. With no such ancestor the root is that
// absolute directory itself. The error is filepath.Abs's, for a relative dir
// when the working directory cannot be read.
func codexHookRoot(dir string) (string, error) {
	base, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if info, statErr := os.Stat(base); statErr != nil || !info.IsDir() {
		base = filepath.Dir(base)
	}
	for ancestor := base; ; ancestor = filepath.Dir(ancestor) {
		if root, found := codexHookRootAt(ancestor); found {
			return root, nil
		}
		if filepath.Dir(ancestor) == ancestor {
			return base, nil
		}
	}
}

// codexHookRootAt looks at ancestor's .git for codexHookRoot. found is false
// when the walk goes on: there is no .git, or it is a directory without HEAD,
// which codex passes over. A .git directory holding HEAD makes ancestor the
// root. Any other .git ends the walk as well: when it is the .git file of a
// linked worktree of a main checkout, the root is that main checkout
// (codexLinkedWorktreeMain); otherwise codex keeps the hooks of the checkout
// itself, so the root is ancestor.
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

// codexLinkedWorktreeMain reads dotGit, the .git file of the checkout at
// checkout: a regular file of at most codexGitMetadataMaxBytes holding
// "gitdir: <path>", a relative path being taken from checkout. When that
// path is <main>/.git/worktrees/<name> -- its parent is named "worktrees" --
// and <main>/.git exists, it returns <main> spelled as the gitdir line
// spells it, since codex keys a linked worktree's hooks under its main
// checkout's path. codex also checks that the worktree's registration under
// the gitdir points back at checkout and that <main>/.git is that common
// directory; a worktree made by git passes those, so they are not repeated
// here.
func codexLinkedWorktreeMain(checkout, dotGit string) (string, bool) {
	info, err := os.Lstat(dotGit)
	if err != nil || !info.Mode().IsRegular() || info.Size() > codexGitMetadataMaxBytes {
		return "", false
	}
	data, err := os.ReadFile(dotGit)
	if err != nil || len(data) > codexGitMetadataMaxBytes {
		return "", false
	}
	target, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir:")
	target = strings.TrimSpace(target)
	if !ok || target == "" {
		return "", false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(checkout, target)
	}
	worktrees := filepath.Dir(filepath.Clean(target))
	if filepath.Base(worktrees) != "worktrees" {
		return "", false
	}
	mainRoot := filepath.Dir(filepath.Dir(worktrees))
	if _, err := os.Stat(filepath.Join(mainRoot, ".git")); err != nil {
		return "", false
	}
	return mainRoot, true
}

// codexHookRootCandidates returns the roots codex may key dir's hooks under:
// the root of dir as spelled, then, when it differs, the root of dir with its
// symlinks resolved. A codex seat's working directory may be either spelling,
// and codex keys the hooks of a main checkout opened through a symlink under
// the symlink's path, so the check asks for an approval under both (plan
// 2026-11-12-codex-hook-trust-check, D1 and AC9). dir is made absolute
// first, with filepath.Abs as codexHookRoot does, and the symlinks of that
// absolute path are resolved: filepath.EvalSymlinks leaves a relative dir
// relative (doctor's "." stays "."), and making it absolute afterwards takes
// os.Getwd, which returns $PWD when it names the same directory, so with $PWD
// in the symlink's spelling both candidates would come out in that spelling
// and the real root would be lost. When dir's symlinks cannot be resolved
// (dir does not exist), only the first root is returned.
func codexHookRootCandidates(dir string) ([]string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	lexical, err := codexHookRoot(absDir)
	if err != nil {
		return nil, err
	}
	roots := []string{lexical}
	resolvedDir, err := filepath.EvalSymlinks(absDir)
	if err != nil {
		return roots, nil
	}
	if resolved, err := codexHookRoot(resolvedDir); err == nil && resolved != lexical {
		roots = append(roots, resolved)
	}
	return roots, nil
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

// codexHookSourceState is how reading one input of the check went: the
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

// classifyCodexHookSlot classifies slot over every root spelling (plan D1,
// AC9): approved only when it is approved under each; otherwise not approved
// when one spelling has no approval, and turned off when every spelling
// without an approval turns it off.
func classifyCodexHookSlot(states map[string]codexHookStateEntry, roots []string, slot codexHookSlot) codexHookTrust {
	result := codexHookApproved
	for _, root := range roots {
		switch codexHookTrustUnder(states, codexHookKey(root, slot)) {
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
	Tail string   // the key after the hooks.json path, such as "pre_tool_use:0:0"
	Keys []string // the full key under each root, in the order of codexHookTrustReport.Roots
}

// codexHookTrustReport is what codexHookTrustFor found. Each input's outcome
// is reported apart, so a caller can tell a missing file from one it could
// not read or decode.
type codexHookTrustReport struct {
	Roots       []string             // codexHookRootCandidates of dir; empty only when dir could not be made absolute
	HooksPath   string               // the hooks.json read: <Roots[0]>/.codex/hooks.json
	HooksState  codexHookSourceState // how reading HooksPath went
	HooksErr    error                // why, for codexHookSourceUnreadable and codexHookSourceInvalid
	ConfigPath  string               // codex's user config (codexSandboxEnv.ConfigPath)
	ConfigState codexHookSourceState // how reading ConfigPath went
	ConfigErr   error                // why, for codexHookSourceUnreadable; a *codexConfigDecodeError for codexHookSourceInvalid
	Slots       []codexHookSlot      // every handler codex asks approval for, once HooksPath was read
	Approved    []codexHookTrustItem // filled only when Classified
	NotApproved []codexHookTrustItem // filled only when Classified; every slot when the config is missing
	Off         []codexHookTrustItem // filled only when Classified
}

// Classified reports whether every slot is in one of Approved, NotApproved,
// and Off: the hooks.json was read, and codex's user config was read or does
// not exist (then no hook is approved).
func (r codexHookTrustReport) Classified() bool {
	return r.HooksState == codexHookSourceRead &&
		(r.ConfigState == codexHookSourceRead || r.ConfigState == codexHookSourceMissing)
}

// codexHookTrustFor checks the hook approvals a codex seat working in dir
// would see (plan 2026-11-12-codex-hook-trust-check, issue #310). It reads
// the hooks.json of dir's first root candidate and [hooks.state] of codex's
// user config at env.ConfigPath, and sorts every handler into approved, not
// approved, and turned off over all root candidates. It prints nothing and
// never fails: a missing, unreadable, or invalid file is reported in the
// result for the caller to word.
func codexHookTrustFor(dir string, env codexSandboxEnv) codexHookTrustReport {
	report := codexHookTrustReport{ConfigPath: env.ConfigPath}
	roots, err := codexHookRootCandidates(dir)
	if err != nil {
		report.HooksPath = codexHooksJSONPath(dir)
		report.HooksState, report.HooksErr = codexHookSourceUnreadable, err
	} else {
		report.Roots = roots
		report.HooksPath = codexHooksJSONPath(roots[0])
		report.Slots, report.HooksState, report.HooksErr = readCodexHookSlots(report.HooksPath)
	}
	var states map[string]codexHookStateEntry
	states, report.ConfigState, report.ConfigErr = readCodexHookStates(env.ConfigPath)
	if report.Classified() {
		report.Approved, report.NotApproved, report.Off = classifyCodexHookSlots(states, report.Roots, report.Slots)
	}
	return report
}

// classifyCodexHookSlots splits slots by classifyCodexHookSlot, keeping
// their order.
func classifyCodexHookSlots(states map[string]codexHookStateEntry, roots []string, slots []codexHookSlot) (approved, notApproved, off []codexHookTrustItem) {
	for _, slot := range slots {
		item := codexHookTrustItem{Tail: slot.Tail()}
		for _, root := range roots {
			item.Keys = append(item.Keys, codexHookKey(root, slot))
		}
		switch classifyCodexHookSlot(states, roots, slot) {
		case codexHookApproved:
			approved = append(approved, item)
		case codexHookNotApproved:
			notApproved = append(notApproved, item)
		case codexHookOff:
			off = append(off, item)
		}
	}
	return approved, notApproved, off
}
