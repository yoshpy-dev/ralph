package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// codexSandboxEnv is the part of the process environment
// checkCodexAgmsgWritableRoot depends on. It is a struct (rather than
// reading CODEX_HOME/AGMSG_STORAGE_PATH directly) so callers -- production
// and test alike -- can substitute values through a plain function instead
// of mutating real environment variables (mirrors shellAliasEnv's shape).
type codexSandboxEnv struct {
	ConfigPath       string // the user's codex config.toml, as this process would read it
	AgmsgStoragePath string // $AGMSG_STORAGE_PATH as seen by this process ("" when unset)
}

// codexSandboxEnvFromOS resolves codexSandboxEnv from CODEX_HOME and
// AGMSG_STORAGE_PATH. The config path follows the exact rule
// codexModelsCachePath uses for codex's own model cache (see its doc
// comment): CODEX_HOME is used literally when non-empty, else
// $HOME/.codex, via os.UserHomeDir. Returns an error only when CODEX_HOME
// is empty and the home directory cannot be resolved.
func codexSandboxEnvFromOS() (codexSandboxEnv, error) {
	var configPath string
	if home := os.Getenv("CODEX_HOME"); home != "" {
		configPath = filepath.Join(home, "config.toml")
	} else {
		uh, err := os.UserHomeDir()
		if err != nil {
			return codexSandboxEnv{}, fmt.Errorf("resolve home directory: %w", err)
		}
		configPath = filepath.Join(uh, ".codex", "config.toml")
	}
	return codexSandboxEnv{
		ConfigPath:       configPath,
		AgmsgStoragePath: os.Getenv("AGMSG_STORAGE_PATH"),
	}, nil
}

// doctorCodexSandboxEnv is the environment resolver runDoctorFull hands to
// checkCodexAgmsgWritableRoot. It is a package variable (rather than a
// direct call to codexSandboxEnvFromOS) so internal/cli's TestMain
// (main_test.go) can pin it to a config path that does not exist, keeping
// every runDoctor*-based test hermetic against the developer's real
// ~/.codex/config.toml (same seam shape as doctorShellAliasEnv).
var doctorCodexSandboxEnv = codexSandboxEnvFromOS

// codexUserConfig is a minimal decode of the user-level codex config.toml:
// only the three keys this check consumes. WritableRoots is typed any
// (rather than []string) so a wrong TOML shape is detected and reported
// instead of silently failing the whole decode -- see codexWritableRoots.
type codexUserConfig struct {
	SandboxMode           string `toml:"sandbox_mode"`
	Profile               string `toml:"profile"`
	SandboxWorkspaceWrite struct {
		WritableRoots any `toml:"writable_roots"`
	} `toml:"sandbox_workspace_write"`
}

// codexConfigMaxBytes bounds how much of a candidate config.toml
// readCodexUserConfig will read. A config far beyond this size is not a
// codex config this check knows how to reason about; the check reports it
// as unreadable rather than risking an unbounded read.
const codexConfigMaxBytes = 1 << 20 // 1 MiB

// codexConfigDisplayPath renders path with the real user home directory
// shown as "~", mirroring checkShellAliases' displayPath convention. Falls
// back to path unchanged when the home directory can't be resolved or
// doesn't prefix path (e.g. every test fixture, which lives under a temp
// dir unrelated to the developer's real home).
func codexConfigDisplayPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	prefix := home + string(filepath.Separator)
	if strings.HasPrefix(path, prefix) {
		return "~" + string(filepath.Separator) + strings.TrimPrefix(path, prefix)
	}
	return path
}

// readCodexUserConfig reads and decodes path as a codex user config. exists
// reports whether path was found on disk at all; when it is false, err is
// always nil and cfg is the zero value -- callers treat an absent config as
// "no writable_roots configured", not as an error. A path that stats
// successfully but is not a regular file (a directory, a FIFO, a device) is
// reported as an error WITHOUT ever being opened -- opening a FIFO can
// block forever, and a directory can't be read as a file. Reads are capped
// at codexConfigMaxBytes; anything larger is reported as an error without
// attempting to decode a truncated document.
func readCodexUserConfig(path string) (cfg codexUserConfig, exists bool, err error) {
	info, statErr := os.Stat(path)
	if statErr != nil {
		if errors.Is(statErr, fs.ErrNotExist) {
			return cfg, false, nil
		}
		return cfg, false, statErr
	}
	if !info.Mode().IsRegular() {
		return cfg, true, errors.New("not a regular file")
	}

	f, openErr := os.Open(path)
	if openErr != nil {
		return cfg, true, openErr
	}
	defer func() { _ = f.Close() }()

	data, readErr := io.ReadAll(io.LimitReader(f, codexConfigMaxBytes+1))
	if readErr != nil {
		return cfg, true, readErr
	}
	if len(data) > codexConfigMaxBytes {
		return cfg, true, errors.New("larger than 1 MiB")
	}

	if decodeErr := toml.Unmarshal(data, &cfg); decodeErr != nil {
		return cfg, true, decodeErr
	}
	return cfg, true, nil
}

// codexConfigReadReason renders a read (not decode) failure from
// readCodexUserConfig without repeating the path (the caller already names
// the config file next to this text): a *fs.PathError's inner Err (e.g.
// "permission denied") when errors.As matches, otherwise the error's own
// text -- which is exactly "not a regular file" or "larger than 1 MiB" for
// the two synthetic errors readCodexUserConfig itself produces. Mirrors
// shellAliasUnreadableReason's reduction.
func codexConfigReadReason(err error) string {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err.Error()
	}
	return err.Error()
}

// codexWritableRoots converts the decoded [sandbox_workspace_write].
// writable_roots value into a plain string slice. raw is nil when the key
// is absent (a missing key is not an error -- it just means no roots are
// configured); raw is []any when TOML decoded an array -- every element
// must be a string, or the whole value is rejected as malformed rather than
// silently dropping the bad entries. Any other shape (a bare string, a
// table, a number) is also rejected.
func codexWritableRoots(raw any) ([]string, error) {
	if raw == nil {
		return nil, nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, errors.New("writable_roots is not an array of strings")
	}
	roots := make([]string, 0, len(arr))
	for _, v := range arr {
		s, ok := v.(string)
		if !ok {
			return nil, errors.New("writable_roots is not an array of strings")
		}
		roots = append(roots, s)
	}
	return roots, nil
}

// codexSandboxReasons collects why a writable root would be needed at all:
// (a) [org.permissions].codex_verified = true, under which ralph itself
// passes --sandbox workspace-write to edits/autonomous codex seats, and (b)
// the user config's own sandbox_mode = workspace-write, which a guarded
// codex seat (one ralph passes no sandbox flag to) inherits unopposed. Both,
// either, or neither may apply; an empty result means the check needs no
// writable root at all.
func codexSandboxReasons(codexVerified bool, cfg codexUserConfig, cfgDisplay string) []string {
	var reasons []string
	if codexVerified {
		reasons = append(reasons,
			"[org.permissions].codex_verified = true (ralph passes --sandbox workspace-write to edits and autonomous codex seats)")
	}
	if cfg.SandboxMode == "workspace-write" {
		reasons = append(reasons,
			fmt.Sprintf("sandbox_mode = \"workspace-write\" in %s (guarded codex seats inherit it)", cfgDisplay))
	}
	return reasons
}

// agmsgStoreDir picks the directory agmsg itself would use for its SQLite
// message store, in agmsg's own priority order (docs/evidence/
// codex-seat-permissions-2026-09-18.md, agmsg 1.1.13's storage.sh):
// AGMSG_STORAGE_PATH when non-empty (with exactly one trailing "/"
// stripped, matching agmsg's own trim), else "<agmsgHome>/db". fromOverride
// reports which branch was taken, so the caller can note that the seat's
// pane environment may not match this process's.
func agmsgStoreDir(agmsgHome, storageOverride string) (dir string, fromOverride bool) {
	if storageOverride != "" {
		dir = strings.TrimSuffix(storageOverride, "/")
		return dir, true
	}
	return filepath.Join(agmsgHome, "db"), false
}

// pathCovers reports whether root -- an absolute path -- is target itself
// or an ancestor of target, one full path element at a time (so /a/b does
// NOT cover /a/bc, only /a/b and everything under it). A relative or
// "~"-prefixed root is never treated as covering anything: codex is not
// known to expand either form in writable_roots, so treating them as a
// match would be a false pass.
func pathCovers(root, target string) bool {
	if !filepath.IsAbs(root) {
		return false
	}
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// resolveNearestExisting resolves symlinks in a path that may not exist yet
// (the agmsg store directory before agmsg has ever run, or a writable root
// the operator configured ahead of time): it walks up to the nearest
// existing ancestor, resolves that ancestor through filepath.EvalSymlinks,
// and re-appends the non-existent suffix components under it. For a path
// that exists this is plain EvalSymlinks. If no ancestor at all can be
// resolved (in practice: the whole path is unreachable), it gives up and
// returns the cleaned original path. Both sides of the coverage comparison
// go through this one function so a not-yet-created store under a
// symlinked parent compares equal to a root spelled through the same
// symlink.
func resolveNearestExisting(path string) string {
	clean := filepath.Clean(path)
	dir := clean
	var suffix []string
	for {
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			return filepath.Join(append([]string{resolved}, suffix...)...)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return clean
		}
		suffix = append([]string{filepath.Base(dir)}, suffix...)
		dir = parent
	}
}

// coveringWritableRoot returns the first entry in roots that covers target
// (pathCovers), trying every root in its cleaned, as-configured form first.
// If none match, a second pass resolves symlinks on both sides
// (resolveNearestExisting) before comparing again. The returned string is always the ORIGINAL
// (unresolved) root text as written in the config, so the Detail names what
// the operator actually configured. Returns "" when no root covers target
// either way.
func coveringWritableRoot(roots []string, target string) string {
	for _, root := range roots {
		if pathCovers(root, target) {
			return root
		}
	}
	resolvedTarget := resolveNearestExisting(target)
	for _, root := range roots {
		if !filepath.IsAbs(root) {
			continue // a relative or ~-prefixed root never matches (see pathCovers)
		}
		if pathCovers(resolveNearestExisting(root), resolvedTarget) {
			return root
		}
	}
	return ""
}

// checkCodexAgmsgWritableRoot is `ralph doctor`'s "Codex sandbox (agmsg
// writable root)" check (#164): a static read of the user's codex
// config.toml, never spawning any process. It warns when a codex seat could
// run under workspace-write (codexVerified, or the config's own sandbox_mode
// = workspace-write) but no writable_roots entry covers the directory
// agmsg's SQLite message store lives in -- the exact failure that made a
// codex seat under workspace-write unable to send RESULT to lead ("attempt
// to write a readonly database", docs/evidence/
// codex-seat-permissions-2026-09-18.md P3).
//
// Six deterministic outcomes, in order:
//  1. agmsg not installed at agmsgHome (driver.AgmsgAvailable) -> info --
//     org seats aren't usable at all, so there is nothing to check. This is
//     decided by AgmsgAvailable itself, never by the separate "agmsg"
//     check's Status, which can be info for a merely differently-versioned
//     install (see checkAgmsgAvailable).
//  2. resolveEnv fails -> info, no config path to read.
//  3. the config can't be read (permission error, not a regular file, over
//     codexConfigMaxBytes), can't be parsed as TOML, or its writable_roots
//     key is not an array of strings -> info. A TOML parse/decode error
//     reports only the *toml.DecodeError's line/column -- never its message
//     text or any other part of the config, which can hold secrets.
//  4. no reason a writable root would even be needed (codexSandboxReasons
//     is empty) -> pass.
//  5. a writable_roots entry covers the agmsg store directory -> pass.
//  6. otherwise -> warn.
//
// The check never returns "fail" -- a missing root is a real gap, but not
// one that should flip doctor's exit code (see docs/recipes/
// codex-seat-permissions.md for the fix).
func checkCodexAgmsgWritableRoot(codexVerified bool, agmsgHome string, resolveEnv func() (codexSandboxEnv, error)) checkResult {
	r := checkResult{Name: "Codex sandbox (agmsg writable root)"}

	if err := driver.AgmsgAvailable(agmsgHome); err != nil {
		r.Status = "info"
		r.Detail = fmt.Sprintf("agmsg not installed at %s — nothing to check", agmsgHome)
		return r
	}

	env, err := resolveEnv()
	if err != nil {
		r.Status = "info"
		r.Detail = fmt.Sprintf("could not resolve the codex config path: %v — writable roots not checked", err)
		return r
	}
	cfgDisplay := codexConfigDisplayPath(env.ConfigPath)

	cfg, _, readErr := readCodexUserConfig(env.ConfigPath)
	if readErr != nil {
		var decodeErr *toml.DecodeError
		if errors.As(readErr, &decodeErr) {
			row, col := decodeErr.Position()
			r.Status = "info"
			r.Detail = fmt.Sprintf("%s is not valid TOML (line %d, column %d) — writable roots not checked", cfgDisplay, row, col)
			return r
		}
		r.Status = "info"
		r.Detail = fmt.Sprintf("could not read %s (%s) — writable roots not checked", cfgDisplay, codexConfigReadReason(readErr))
		return r
	}

	roots, rootsErr := codexWritableRoots(cfg.SandboxWorkspaceWrite.WritableRoots)
	if rootsErr != nil {
		r.Status = "info"
		r.Detail = fmt.Sprintf("%s: [sandbox_workspace_write].writable_roots is not an array of strings — writable roots not checked", cfgDisplay)
		return r
	}

	reasons := codexSandboxReasons(codexVerified, cfg, cfgDisplay)
	if len(reasons) == 0 {
		r.Status = "pass"
		r.Detail = fmt.Sprintf("not needed: [org.permissions].codex_verified is false and %s does not set sandbox_mode = \"workspace-write\"", cfgDisplay)
		return r
	}
	reasonClause := strings.Join(reasons, " and ")

	store, fromOverride := agmsgStoreDir(agmsgHome, env.AgmsgStoragePath)
	if abs, absErr := filepath.Abs(store); absErr == nil {
		store = abs
	}

	var suffix string
	if fromOverride {
		suffix += " (store taken from AGMSG_STORAGE_PATH; the seat's pane environment may differ)"
	}
	if cfg.Profile != "" {
		suffix += fmt.Sprintf(" (profile %q in %s is not evaluated)", cfg.Profile, cfgDisplay)
	}

	if root := coveringWritableRoot(roots, store); root != "" {
		r.Status = "pass"
		r.Detail = fmt.Sprintf("writable root %s in %s covers the agmsg store %s; needed because %s",
			root, cfgDisplay, store, reasonClause) + suffix
		return r
	}

	r.Status = "warn"
	r.Detail = fmt.Sprintf("no writable root in %s covers the agmsg store %s, so a codex seat under workspace-write cannot send RESULT to lead "+
		"(\"attempt to write a readonly database\"); needed because %s; add %s to [sandbox_workspace_write].writable_roots "+
		"(docs/recipes/codex-seat-permissions.md)", cfgDisplay, store, reasonClause, store) + suffix
	return r
}
