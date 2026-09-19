package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// codexSandboxEnv is the part of the process environment
// checkCodexAgmsgWritableRoot depends on. It is a struct (rather than
// reading CODEX_HOME/AGMSG_STORAGE_PATH directly) so callers -- production
// and test alike -- can substitute values through a plain function instead
// of mutating real environment variables (mirrors shellAliasEnv's shape).
type codexSandboxEnv struct {
	Home             string // the real user home directory, for "~" display only -- never used to build ConfigPath itself
	ConfigPath       string // the user's codex config.toml, as this process would read it
	AgmsgStoragePath string // $AGMSG_STORAGE_PATH as seen by this process ("" when unset)
}

// codexSandboxEnvFromOS resolves codexSandboxEnv from CODEX_HOME,
// os.UserHomeDir, and AGMSG_STORAGE_PATH. This is the only function in this
// file allowed to call os.UserHomeDir -- every other function that needs to
// know the home directory (for "~" display) takes it as a parameter, so
// TestMain can pin it and no test accidentally reads the developer's real
// home. The config path follows the exact rule codexModelsCachePath uses
// for codex's own model cache (see its doc comment): CODEX_HOME is used
// literally when non-empty, else $HOME/.codex. Returns an error only when
// CODEX_HOME is empty and the home directory cannot be resolved -- in that
// case there is no way to build ConfigPath at all. When CODEX_HOME IS set
// but the home directory still can't be resolved, Home is left "" (display
// falls back to the raw path) rather than failing the whole check over a
// value it doesn't strictly need.
func codexSandboxEnvFromOS() (codexSandboxEnv, error) {
	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		home = ""
	}

	var configPath string
	if codexHome := os.Getenv("CODEX_HOME"); codexHome != "" {
		configPath = filepath.Join(codexHome, "config.toml")
	} else {
		if homeErr != nil {
			return codexSandboxEnv{}, fmt.Errorf("resolve home directory: %w", homeErr)
		}
		configPath = filepath.Join(home, ".codex", "config.toml")
	}

	return codexSandboxEnv{
		Home:             home,
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

// codexConfigDisplayPath renders path with home shown as "~", mirroring
// checkShellAliases' displayPath convention. home is supplied by the
// caller (codexSandboxEnv.Home, itself only ever set by
// codexSandboxEnvFromOS) rather than read here via os.UserHomeDir, so the
// substitution is observable and controllable from the same seam as the
// rest of the check (self-review MEDIUM-2: reading the real home directly
// here made every Detail assertion depend on where the test process's
// TMPDIR happened to be). Falls back to path unchanged when home is empty
// or doesn't prefix path.
func codexConfigDisplayPath(home, path string) string {
	if home == "" {
		return path
	}
	prefix := home + string(filepath.Separator)
	if strings.HasPrefix(path, prefix) {
		return "~" + string(filepath.Separator) + strings.TrimPrefix(path, prefix)
	}
	return path
}

// codexConfigDecodeError marks a config that was read successfully but
// could not be decoded as a codexUserConfig -- either invalid TOML syntax,
// or syntactically valid TOML with the wrong shape (a duplicate key, a
// table redefinition, a type mismatch). It deliberately carries no message
// text of its own: go-toml v2.3.0 raises several of these (duplicate-key
// and table-conflict errors from its internal/tracker package, in
// particular) as plain fmt.Errorf values whose text repeats the offending
// key or table name verbatim -- exactly the config content this check
// promises never to surface (self-review MEDIUM-1). HasPosition is true
// only when the underlying error was a *toml.DecodeError, in which case
// Line/Column were read through its Position() method -- never through its
// Error() or String(), both of which include surrounding config text.
type codexConfigDecodeError struct {
	Line, Column int
	HasPosition  bool
}

func (e *codexConfigDecodeError) Error() string {
	if e.HasPosition {
		return fmt.Sprintf("codex config: could not be decoded (line %d, column %d)", e.Line, e.Column)
	}
	return "codex config: could not be decoded"
}

// codexConfigDecodeDetail renders the Detail for a config that was read but
// not decodable (a *codexConfigDecodeError from readCodexUserConfig) --
// "not valid TOML" is deliberately avoided, since a type mismatch (e.g.
// sandbox_mode = 5) is valid TOML that merely has the wrong shape for this
// check's struct.
func codexConfigDecodeDetail(cfgDisplay string, err *codexConfigDecodeError) string {
	if err.HasPosition {
		return fmt.Sprintf("%s could not be decoded as a codex config (line %d, column %d) — writable roots not checked",
			cfgDisplay, err.Line, err.Column)
	}
	return fmt.Sprintf("%s could not be decoded as a codex config — writable roots not checked", cfgDisplay)
}

// readCodexUserConfig reads and decodes path as a codex user config. exists
// reports whether path was found on disk at all; when it is false, err is
// always nil and cfg is the zero value -- callers treat an absent config as
// "no writable_roots configured", not as an error. A path that stats
// successfully but is not a regular file (a directory, a FIFO, a device) is
// reported as an error WITHOUT ever being opened -- opening a FIFO can
// block forever, and a directory can't be read as a file. Reads are capped
// at codexConfigMaxBytes; anything larger is reported as an error without
// attempting to decode a truncated document. A decode failure is always
// wrapped as *codexConfigDecodeError (never returned as go-toml's own
// error), so callers can distinguish "couldn't read the bytes" from
// "read the bytes fine, couldn't decode them" without ever being tempted to
// print go-toml's message text.
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
		wrapped := &codexConfigDecodeError{}
		var tomlErr *toml.DecodeError
		if errors.As(decodeErr, &tomlErr) {
			wrapped.HasPosition = true
			wrapped.Line, wrapped.Column = tomlErr.Position()
		}
		return cfg, true, wrapped
	}
	return cfg, true, nil
}

// codexConfigReadReason renders a read failure from readCodexUserConfig --
// never a decode failure, which is always a *codexConfigDecodeError handled
// separately by codexConfigDecodeDetail -- without repeating the path (the
// caller already names the config file next to this text): a
// *fs.PathError's inner Err (e.g. "permission denied") when errors.As
// matches, otherwise the error's own text, which is exactly
// "not a regular file" or "larger than 1 MiB" for the two synthetic errors
// readCodexUserConfig itself produces on this path. Mirrors
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

// codexSeatModesPossible reports whether any role in orgCfg could resolve
// to a workspace-write-capable permission mode (edits or autonomous) and/or
// to guarded, across orgCfg's effective default plus every per-role
// override in orgCfg.Permissions.Roles. A seat's permission mode is decided
// entirely by [org.permissions] -- org.ResolvePermissionMode's precedence
// (a role's own entry, else Default, else the built-in default) -- there is
// no per-spawn override, so this fully characterizes which modes a codex
// seat in this project could ever actually run under (self-review LOW-6).
func codexSeatModesPossible(orgCfg config.OrgConfig) (workspaceWriteRole, guardedRole bool) {
	apply := func(mode string) {
		switch mode {
		case org.PermissionModeEdits, org.PermissionModeAutonomous:
			workspaceWriteRole = true
		case org.PermissionModeGuarded:
			guardedRole = true
		}
	}
	apply(org.ResolvePermissionMode(orgCfg, ""))
	for _, mode := range orgCfg.Permissions.Roles {
		apply(mode)
	}
	return workspaceWriteRole, guardedRole
}

// codexSandboxReasons collects why a writable root would be needed at all:
// (a) codexVerified plus a role that could resolve to edits or autonomous,
// under which ralph itself passes --sandbox workspace-write to those codex
// seats, and (b) the user config's own sandbox_mode = workspace-write plus
// a role that could resolve to guarded, which a guarded codex seat (one
// ralph passes no sandbox flag to) inherits unopposed. Both, either, or
// neither may apply; an empty result means the check needs no writable root
// at all -- see codexNotNeededWhyA/codexNotNeededWhyB for why, in that
// case.
func codexSandboxReasons(codexVerified, workspaceWriteRole, guardedRole bool, cfg codexUserConfig, cfgDisplay string) []string {
	var reasons []string
	if codexVerified && workspaceWriteRole {
		reasons = append(reasons,
			"[org.permissions].codex_verified = true and a role resolves to edits or autonomous "+
				"(ralph passes --sandbox workspace-write to those codex seats)")
	}
	if cfg.SandboxMode == "workspace-write" && guardedRole {
		reasons = append(reasons,
			fmt.Sprintf("sandbox_mode = \"workspace-write\" in %s and a role resolves to guarded (guarded codex seats inherit it)", cfgDisplay))
	}
	return reasons
}

// codexNotNeededWhyA explains why reason (a) (codexVerified plus a
// workspace-write-capable role) does not apply. It is only called when
// reason (a) is absent, so codexVerified alone decides the wording:
// codexVerified being false is the simpler, more directly actionable fact,
// and when it is true the missing piece can only be the role condition.
func codexNotNeededWhyA(codexVerified bool) string {
	if !codexVerified {
		return "[org.permissions].codex_verified is false"
	}
	return "no role resolves to edits or autonomous"
}

// codexNotNeededWhyB is codexNotNeededWhyA's counterpart for reason (b)
// (sandbox_mode = workspace-write plus a guarded-capable role), with three
// alternatives tried in order: no role could ever be guarded at all; a
// guarded role exists but there is no config to set sandbox_mode in; a
// guarded role exists and the config exists but doesn't set it.
func codexNotNeededWhyB(guardedRole, exists bool, cfgDisplay string) string {
	switch {
	case !guardedRole:
		return "no role resolves to guarded"
	case !exists:
		return fmt.Sprintf("%s does not exist", cfgDisplay)
	default:
		return fmt.Sprintf("%s does not set sandbox_mode = \"workspace-write\"", cfgDisplay)
	}
}

// agmsgStoreDir picks the directory agmsg itself would use for its SQLite
// message store, in agmsg's own priority order: AGMSG_STORAGE_PATH when
// non-empty (with exactly one trailing "/" stripped -- agmsg's own
// scripts/lib/storage.sh does `${AGMSG_STORAGE_PATH%/}`, verified against
// agmsg 1.1.13), else "<agmsgHome>/db" (the default location documented in
// docs/evidence/codex-seat-permissions-2026-09-18.md; that file does not
// itself describe AGMSG_STORAGE_PATH -- storage.sh is the source for the
// override and its trim, self-review LOW-9). fromOverride reports which
// branch was taken, so the caller can note that the seat's pane environment
// may not match this process's.
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
// (pathCovers) once both sides are resolved through resolveNearestExisting
// -- so a writable_roots entry spelled through a symlink, or an agmsg store
// that lives behind one, still compares correctly. This is now the ONLY
// comparison pass: an earlier version tried an unresolved textual
// comparison first, which returned pass for a store directory that was
// itself a symlink pointing OUTSIDE every configured root -- exactly the
// misconfiguration this check exists to catch (self-review MEDIUM-3).
// resolveNearestExisting is strictly at least as permissive as a plain
// textual comparison (it falls back to the cleaned path when nothing
// resolves), so dropping the textual pass loses no legitimate match. A
// relative or "~"-prefixed root is skipped before any resolution is
// attempted -- checked on the root as CONFIGURED, not on its resolved form,
// so a relative root can never become absolute by accident of where
// symlink resolution happens to land. The returned string is always the
// ORIGINAL (unresolved) root text as written in the config, so the Detail
// names what the operator actually configured. Returns "" when no root
// covers target.
func coveringWritableRoot(roots []string, target string) string {
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

// codexWritableRootSuffix builds the trailing notes appended to a covered
// (pass) or not-covered (warn) Detail: a note that the store came from
// AGMSG_STORAGE_PATH (the seat's pane environment may differ from this
// process's), and/or a note that a configured profile is not evaluated by
// this check. Either, both, or neither may be present; order is fixed
// (override note before profile note).
func codexWritableRootSuffix(fromOverride bool, profile, cfgDisplay string) string {
	var suffix string
	if fromOverride {
		suffix += " (store taken from AGMSG_STORAGE_PATH; the seat's pane environment may differ)"
	}
	if profile != "" {
		suffix += fmt.Sprintf(" (profile %q in %s is not evaluated)", profile, cfgDisplay)
	}
	return suffix
}

// codexWritableRootDetail renders the Detail for the check's last two
// outcomes: root is coveringWritableRoot's result (non-empty on a pass,
// "" on a warn). When there is no covering root, the wording distinguishes
// a config that exists but simply omits a covering entry from one that
// does not exist at all (self-review LOW-4) -- the latter also names the
// config path in the "add ... in <cfg>" clause, since there is no existing
// [sandbox_workspace_write] table to point the operator at otherwise.
func codexWritableRootDetail(root, cfgDisplay, store string, exists bool, reasonClause, suffix string) string {
	if root != "" {
		return fmt.Sprintf("writable root %s in %s covers the agmsg store %s; needed because %s",
			root, cfgDisplay, store, reasonClause) + suffix
	}
	if !exists {
		return fmt.Sprintf("no writable root covers the agmsg store %s (%s does not exist), so a codex seat under workspace-write "+
			"cannot send RESULT to lead (\"attempt to write a readonly database\"); needed because %s; add %s to "+
			"[sandbox_workspace_write].writable_roots in %s (docs/recipes/codex-seat-permissions.md)",
			store, cfgDisplay, reasonClause, store, cfgDisplay) + suffix
	}
	return fmt.Sprintf("no writable root in %s covers the agmsg store %s, so a codex seat under workspace-write cannot send RESULT to lead "+
		"(\"attempt to write a readonly database\"); needed because %s; add %s to [sandbox_workspace_write].writable_roots "+
		"(docs/recipes/codex-seat-permissions.md)", cfgDisplay, store, reasonClause, store) + suffix
}

// checkCodexAgmsgWritableRoot is `ralph doctor`'s "Codex sandbox (agmsg
// writable root)" check (#164): a static read of the user's codex
// config.toml plus the project's [org] envelope, never spawning any
// process. It warns when a codex seat could actually run under
// workspace-write -- via [org.permissions].codex_verified = true with a
// role that resolves to edits or autonomous, or via the config's own
// sandbox_mode = workspace-write with a role that resolves to guarded --
// but no writable_roots entry covers the directory agmsg's SQLite message
// store lives in: the exact failure that made a codex seat under
// workspace-write unable to send RESULT to lead ("attempt to write a
// readonly database", docs/evidence/codex-seat-permissions-2026-09-18.md
// P3).
//
// Deterministic outcomes, in order:
//  1. agmsg not installed at agmsgHome (driver.AgmsgAvailable) -> info --
//     org seats aren't usable at all, so there is nothing to check. This is
//     decided by AgmsgAvailable itself, never by the separate "agmsg"
//     check's Status, which can be info for a merely differently-versioned
//     install (see checkAgmsgAvailable).
//  2. codex is not in [org].driver_pool -> pass -- no codex seat can ever
//     be spawned, so nothing downstream is even read (self-review LOW-6).
//  3. resolveEnv fails -> info, no config path to read.
//  4. the config can't be read (permission error, not a regular file, over
//     codexConfigMaxBytes) -> info naming the reason (codexConfigReadReason).
//     The config can't be decoded -- invalid TOML syntax, or syntactically
//     valid TOML with the wrong shape (a duplicate key, a table
//     redefinition, a type mismatch) -- -> info naming only the line/column
//     when one is available; its own message text, which can embed a key or
//     table name taken from the user's config, is never surfaced
//     (codexConfigDecodeError, self-review MEDIUM-1). writable_roots is
//     present but not an array of strings -> info.
//  5. no role could ever make a writable root necessary
//     (codexSeatModesPossible plus codexSandboxReasons finds nothing) ->
//     pass, naming which of the two preconditions failed on each side
//     (codexNotNeededWhyA/codexNotNeededWhyB).
//  6. a writable_roots entry covers the agmsg store directory
//     (coveringWritableRoot) -> pass.
//  7. otherwise -> warn, distinguishing a config that exists but omits a
//     covering root from one that does not exist at all
//     (codexWritableRootDetail).
//
// The check never returns "fail" -- a missing root is a real gap, but not
// one that should flip doctor's exit code (see docs/recipes/
// codex-seat-permissions.md for the fix).
func checkCodexAgmsgWritableRoot(orgCfg config.OrgConfig, agmsgHome string, resolveEnv func() (codexSandboxEnv, error)) checkResult {
	r := checkResult{Name: "Codex sandbox (agmsg writable root)"}

	if err := driver.AgmsgAvailable(agmsgHome); err != nil {
		r.Status = "info"
		r.Detail = fmt.Sprintf("agmsg not installed at %s — nothing to check", agmsgHome)
		return r
	}
	if !slices.Contains(orgCfg.DriverPool, "codex") {
		r.Status = "pass"
		r.Detail = "not needed: codex is not in [org].driver_pool, so no codex seat can be spawned"
		return r
	}

	env, err := resolveEnv()
	if err != nil {
		r.Status = "info"
		r.Detail = fmt.Sprintf("could not resolve the codex config path: %v — writable roots not checked", err)
		return r
	}
	cfgDisplay := codexConfigDisplayPath(env.Home, env.ConfigPath)

	cfg, exists, readErr := readCodexUserConfig(env.ConfigPath)
	if readErr != nil {
		var decodeErr *codexConfigDecodeError
		if errors.As(readErr, &decodeErr) {
			r.Status = "info"
			r.Detail = codexConfigDecodeDetail(cfgDisplay, decodeErr)
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

	workspaceWriteRole, guardedRole := codexSeatModesPossible(orgCfg)
	reasons := codexSandboxReasons(orgCfg.Permissions.CodexVerified, workspaceWriteRole, guardedRole, cfg, cfgDisplay)
	if len(reasons) == 0 {
		r.Status = "pass"
		r.Detail = fmt.Sprintf("not needed: %s and %s",
			codexNotNeededWhyA(orgCfg.Permissions.CodexVerified),
			codexNotNeededWhyB(guardedRole, exists, cfgDisplay))
		return r
	}
	reasonClause := strings.Join(reasons, " and ")

	store, fromOverride := agmsgStoreDir(agmsgHome, env.AgmsgStoragePath)
	if abs, absErr := filepath.Abs(store); absErr == nil {
		store = abs
	}
	suffix := codexWritableRootSuffix(fromOverride, cfg.Profile, cfgDisplay)

	root := coveringWritableRoot(roots, store)
	if root != "" {
		r.Status = "pass"
	} else {
		r.Status = "warn"
	}
	r.Detail = codexWritableRootDetail(root, cfgDisplay, store, exists, reasonClause, suffix)
	return r
}
