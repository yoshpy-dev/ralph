package cli

import (
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
// seat works in, it finds the project layers codex reads hooks from, lists
// the handlers of each layer's .codex/hooks.json under the keys codex records
// their approval at, reads [hooks.state] from codex's user config, and sorts
// each handler into approved, not approved, and turned off. The rules mirror
// codex-cli 0.166.0 (openai/codex, tag rust-v0.166.0), checked on this
// machine in docs/evidence/codex-hook-trust-worktree-2026-11-12.md. Nothing
// here runs git or codex, and nothing here prints: the doctor check and the
// launch notices build their own text from a codexHookTrustReport.

// codexHookSlot is one handler of a hooks.json that the check asks an
// approval for (parseCodexHookSlots, which counts some handlers codex asks
// none for). Group and Handler are positions in the JSON arrays, counted the
// way codex counts them: append_matcher_groups in
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
	Slots []codexHookSlot      // every handler parseCodexHookSlots counts, once HooksPath was read
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

// codexHookTrustItem is one hook handler of a codexHookTrustReport. Its key
// under each of its layer's Dirs is codexHookKey of that directory and the
// slot whose Tail it holds.
type codexHookTrustItem struct {
	Layer int    // the index of its layer in codexHookTrustReport.Layers
	Tail  string // the key after the hooks.json path, such as "pre_tool_use:0:0"
}

// codexHookTrustReport is what codexHookTrustFor found. Each input's outcome
// is reported apart, so a caller can tell a missing file from one it could
// not read or decode. When Classified, every slot of Layers that is in
// neither NotApproved nor Off is approved.
type codexHookTrustReport struct {
	Layers      []codexHookLayer     // the root layer first, then each nested layer down to the directory; never empty
	ConfigPath  string               // codex's user config (codexSandboxEnv.ConfigPath)
	ConfigState codexHookSourceState // how reading ConfigPath went
	ConfigErr   error                // why, for codexHookSourceUnreadable; a *codexConfigDecodeError for codexHookSourceInvalid
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

// Classified reports whether every slot was classified, so that NotApproved
// and Off are complete: no layer has a problem (HooksProblems), and codex's
// user config was read or does not exist (then no hook is approved).
func (r codexHookTrustReport) Classified() bool {
	return len(r.HooksProblems()) == 0 &&
		(r.ConfigState == codexHookSourceRead || r.ConfigState == codexHookSourceMissing)
}

// SlotCount is the number of slots (the handlers parseCodexHookSlots counts),
// over every layer read.
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

// SlotPaths lists the hooks.json of every layer that holds a slot, root
// first.
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
// env.ConfigPath, and lists the handlers not approved and those turned off
// over all spellings; the rest are approved. It prints nothing and never
// fails: a missing, unreadable, or invalid file is reported in the result for
// the caller to word.
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
		report.NotApproved, report.Off = classifyCodexHookSlots(states, report.Layers)
	}
	return report
}

// classifyCodexHookSlots lists the slots of every layer that
// classifyCodexHookSlot finds not approved and those it finds turned off,
// keeping the layers' and the slots' order. An approved slot is in neither.
func classifyCodexHookSlots(states map[string]codexHookStateEntry, layers []codexHookLayer) (notApproved, off []codexHookTrustItem) {
	for i, layer := range layers {
		for _, slot := range layer.Slots {
			item := codexHookTrustItem{Layer: i, Tail: slot.Tail()}
			switch classifyCodexHookSlot(states, layer.Dirs, slot) {
			case codexHookNotApproved:
				notApproved = append(notApproved, item)
			case codexHookOff:
				off = append(off, item)
			}
		}
	}
	return notApproved, off
}
