package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// codexSandboxEnv is the part of the process environment
// checkCodexAgmsgWritableRoot depends on. It is a struct (rather than
// reading CODEX_HOME/AGMSG_STORAGE_PATH/TMPDIR directly) so callers --
// production and test alike -- can substitute values through a plain
// function instead of mutating real environment variables (mirrors
// shellAliasEnv's shape).
type codexSandboxEnv struct {
	Home             string // the real user home directory, for "~" display only -- never used to build ConfigPath itself
	ConfigPath       string // the user's codex config.toml, as this process would read it
	AgmsgStoragePath string // $AGMSG_STORAGE_PATH as seen by this process ("" when unset)
	TmpDir           string // $TMPDIR as seen by this process ("" when unset) -- see codexImplicitWritableRoots
	SlashTmpDir      string // the directory codex's workspace-write sandbox keeps writable as its fixed temp root (unix: /tmp; unset where there is no such default) -- empty disables that implicit root; see codexImplicitWritableRoots
}

// codexSandboxEnvFromOS resolves codexSandboxEnv from CODEX_HOME,
// os.UserHomeDir, AGMSG_STORAGE_PATH, TMPDIR, and runtime.GOOS. This is the
// only function in this file allowed to call os.UserHomeDir/os.Getenv, and
// the only place the fixed temp root's literal path appears at all -- every
// other function that needs one of these values takes it as a parameter, so
// TestMain can pin it and no test accidentally reads the developer's real
// home, the machine's real TMPDIR, or that hard-coded path (cross-review
// WC-1's follow-up: every t.TempDir() fixture lives under the real TMPDIR,
// and on a machine or CI runner whose real TMPDIR happens to equal codex's
// own fixed temp root -- Go's own default when TMPDIR is unset -- a check
// that hard-coded that path instead of routing it through this seam could
// never be tested hermetically; SlashTmpDir exists so tests can inject a
// fixture directory here instead). The config path follows the exact rule
// codexModelsCachePath uses for codex's own model cache (see its doc
// comment): CODEX_HOME is used literally when non-empty, else $HOME/.codex.
// Returns an error only when CODEX_HOME is empty and the home directory
// cannot be resolved -- in that case there is no way to build ConfigPath at
// all. When CODEX_HOME IS set but the home directory still can't be
// resolved, Home is left "" (display falls back to the raw path) rather
// than failing the whole check over a value it doesn't strictly need.
// SlashTmpDir is set on every platform except windows, where codex's
// documented fixed temp root has no equivalent, so it stays unset there and
// codexImplicitWritableRoots simply skips that implicit root.
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

	var slashTmpDir string
	if runtime.GOOS != "windows" {
		slashTmpDir = "/tmp"
	}

	return codexSandboxEnv{
		Home:             home,
		ConfigPath:       configPath,
		AgmsgStoragePath: os.Getenv("AGMSG_STORAGE_PATH"),
		TmpDir:           os.Getenv("TMPDIR"),
		SlashTmpDir:      slashTmpDir,
	}, nil
}

// doctorCodexSandboxEnv is the environment resolver runDoctorFull hands to
// checkCodexAgmsgWritableRoot. It is a package variable (rather than a
// direct call to codexSandboxEnvFromOS) so internal/cli's TestMain
// (main_test.go) can pin it to a config path that does not exist, keeping
// every runDoctor*-based test hermetic against the developer's real
// ~/.codex/config.toml (same seam shape as doctorShellAliasEnv). TestMain's
// pinned literal leaves TmpDir and SlashTmpDir unset (Go zero value ""),
// which is already hermetic: codexImplicitWritableRoots only treats a
// non-empty value as a candidate for either.
var doctorCodexSandboxEnv = codexSandboxEnvFromOS

// codexUserConfig is a minimal decode of the user-level codex config.toml:
// only the keys this check consumes. WritableRoots is typed any (rather
// than []string) so a wrong TOML shape is detected and reported instead of
// silently failing the whole decode -- see codexWritableRoots.
// ExcludeSlashTmp/ExcludeTmpdirEnvVar gate the two implicit writable roots
// workspace-write grants by default -- see codexImplicitWritableRoots
// (cross-review WC-1).
type codexUserConfig struct {
	SandboxMode           string `toml:"sandbox_mode"`
	Profile               string `toml:"profile"`
	SandboxWorkspaceWrite struct {
		WritableRoots       any  `toml:"writable_roots"`
		ExcludeSlashTmp     bool `toml:"exclude_slash_tmp"`
		ExcludeTmpdirEnvVar bool `toml:"exclude_tmpdir_env_var"`
	} `toml:"sandbox_workspace_write"`
}

// codexConfigMaxBytes bounds how much of a candidate config.toml
// readCodexUserConfig will read. A config far beyond this size is not a
// codex config this check knows how to reason about; the check reports it
// as unreadable rather than risking an unbounded read.
const codexConfigMaxBytes = 1 << 20 // 1 MiB

// codexConfigDisplayPath renders path with home shown as "~", mirroring
// checkShellAliases' displayPath convention. home is supplied by the
// caller (codexSandboxEnv.Home, set by codexSandboxEnvFromOS in production
// and by the seam -- TestMain, or a test's own resolveEnv closure -- in
// tests) rather than read here via os.UserHomeDir, so the substitution is
// observable and controllable from the same seam as the rest of the check
// (self-review MEDIUM-2: reading the real home directly here made every
// Detail assertion depend on where the test process's TMPDIR happened to
// be). Falls back to path unchanged when home is empty or doesn't prefix
// path.
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

// codexModelPoolModels returns the codex-driver entries of
// orgCfg.ModelPool, in list order.
func codexModelPoolModels(orgCfg config.OrgConfig) []string {
	var models []string
	for _, entry := range orgCfg.ModelPool {
		if entry.Driver == "codex" {
			models = append(models, entry.Model)
		}
	}
	return models
}

// codexModelPermittedForRole mirrors internal/org/envelope.go's
// modelAllowedForRole -- the actual gate ValidateSpawnEnvelope applies at
// spawn time -- so this check's role-mode gating agrees with what could
// really be spawned (cross-review WC-2: without this, a permission-mode
// override for a role whose [org.roles] entry excludes every codex model
// was counted as a possible codex seat, although ValidateSpawnEnvelope
// would reject every codex model for that role). A role absent from
// orgCfg.Roles, or mapped to an empty list, means "no restriction": any
// codex model in the pool is enough. Otherwise the role's explicit
// allowlist must contain at least one of codexModels. codexModels being
// empty always returns false -- there is nothing to permit, matching the
// "no codex model in the pool" gate checkCodexAgmsgWritableRoot already
// applies one level up.
func codexModelPermittedForRole(orgCfg config.OrgConfig, role string, codexModels []string) bool {
	if len(codexModels) == 0 {
		return false
	}
	allowed, ok := orgCfg.Roles[role]
	if !ok || len(allowed) == 0 {
		return true
	}
	return slices.ContainsFunc(allowed, func(m string) bool { return slices.Contains(codexModels, m) })
}

// codexSeatModesPossible reports whether any role in orgCfg could resolve
// to a workspace-write-capable permission mode (edits or autonomous) and/or
// to guarded, across orgCfg's effective default plus every per-role
// override in orgCfg.Permissions.Roles. A seat's permission mode is decided
// entirely by [org.permissions] -- org.ResolvePermissionMode's precedence
// (a role's own entry, else Default, else the built-in default) -- there is
// no per-spawn override, so this fully characterizes which modes a codex
// seat in this project could ever actually run under (self-review LOW-6).
//
// The effective default is resolved from a copy of orgCfg with Roles nilled
// out, not from orgCfg itself: org.ResolvePermissionMode(orgCfg, "") checks
// Permissions.Roles[""] first, so a [org.permissions.roles] entry keyed by
// the empty string would silently replace the real default instead of
// being counted as just another role (self-review NEW-2 -- config.Load
// validates a roles entry's mode value but never its role-name key, so such
// a document loads without error even though "" is never a real role name:
// internal/cli/org.go's --role flag is required and non-blank at spawn
// time). The loop below still applies every Roles value, but only when
// codexModelPermittedForRole says that role could actually be assigned a
// codex model (cross-review WC-2) -- the default itself counts iff the
// model pool has any codex model at all (codexModelPoolModels non-empty):
// role names are open-ended (chosen at `ralph org spawn --role <name>`), so
// there is no fixed role name to check the default's own eligibility
// against; the default applies to whichever unlisted role a seat spawns
// under, and any such role could hold a codex model unless the pool itself
// has none.
func codexSeatModesPossible(orgCfg config.OrgConfig) (workspaceWriteRole, guardedRole bool) {
	apply := func(mode string) {
		switch mode {
		case org.PermissionModeEdits, org.PermissionModeAutonomous:
			workspaceWriteRole = true
		case org.PermissionModeGuarded:
			guardedRole = true
		}
	}

	codexModels := codexModelPoolModels(orgCfg)
	if len(codexModels) > 0 {
		defaultsOnly := orgCfg
		defaultsOnly.Permissions.Roles = nil
		apply(org.ResolvePermissionMode(defaultsOnly, ""))
	}
	for role, mode := range orgCfg.Permissions.Roles {
		if codexModelPermittedForRole(orgCfg, role, codexModels) {
			apply(mode)
		}
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
			"[org.permissions].codex_verified = true and a role that can use a codex model resolves to edits or autonomous "+
				"(ralph passes --sandbox workspace-write to those codex seats)")
	}
	if cfg.SandboxMode == "workspace-write" && guardedRole {
		reasons = append(reasons,
			fmt.Sprintf("sandbox_mode = \"workspace-write\" in %s and a role that can use a codex model resolves to guarded (guarded codex seats inherit it)", cfgDisplay))
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
	return "no role that can use a codex model resolves to edits or autonomous"
}

// codexNotNeededWhyB is codexNotNeededWhyA's counterpart for reason (b)
// (sandbox_mode = workspace-write plus a guarded-capable role), with three
// alternatives tried in order: no role could ever be guarded at all; a
// guarded role exists but there is no config to set sandbox_mode in; a
// guarded role exists and the config exists but doesn't set it.
func codexNotNeededWhyB(guardedRole, exists bool, cfgDisplay string) string {
	switch {
	case !guardedRole:
		return "no role that can use a codex model resolves to guarded"
	case !exists:
		return fmt.Sprintf("%s does not exist", cfgDisplay)
	default:
		return fmt.Sprintf("%s does not set sandbox_mode = \"workspace-write\"", cfgDisplay)
	}
}

// codexSandboxReasonClause joins codexSandboxReasons' output into the
// "needed because ..." clause. Each individual reason string already
// contains its own " and " (the role condition -- see codexSandboxReasons),
// so a second, plain " and " between two reasons used to read as one flat,
// undifferentiated four-clause chain with no visible boundary between the
// two reasons (self-review NEW-1). With more than one reason, each is
// numbered so the boundary is visible: "(1) <reason a> and (2) <reason
// b>". A single reason is rendered unchanged -- numbering it would only add
// noise, since there is nothing to distinguish it from. Zero reasons
// renders as "" -- checkCodexAgmsgWritableRoot never actually calls this in
// that case (it takes the "not needed" branch instead), but the case is
// defined here rather than left unspecified, since this is otherwise a
// pure, directly testable function.
func codexSandboxReasonClause(reasons []string) string {
	switch len(reasons) {
	case 0:
		return ""
	case 1:
		return reasons[0]
	default:
		numbered := make([]string, len(reasons))
		for i, reason := range reasons {
			numbered[i] = fmt.Sprintf("(%d) %s", i+1, reason)
		}
		return strings.Join(numbered, " and ")
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
// match would be a false pass. pathCovers alone does NOT decide whether
// codex would actually leave the target writable -- see
// pathCrossesCodexProtectedDir for the directories codex protects even
// inside a covering root.
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

// codexProtectedDirNames are directory names codex protects as read-only
// even inside an otherwise-writable root, per codex's own documentation
// ("Agent approvals & security"): .git, .agents, and .codex are excluded
// from writes regardless of sandbox mode, and the protection is described
// as recursive. The default agmsg home is ~/.agents/skills/agmsg, so a
// broad writable root like the user's home directory does NOT actually make
// the agmsg store writable (cross-review AR-1). Whether codex protects only
// a writable root's own top-level entries or every nested occurrence is not
// documented precisely, so pathCrossesCodexProtectedDir treats any
// occurrence between root and target as protected -- the warn-biased
// reading, consistent with this check's stated bias (a false pass hides the
// exact failure this check exists to catch; a false warn costs one look at
// the recipe).
var codexProtectedDirNames = map[string]bool{
	".git":    true,
	".agents": true,
	".codex":  true,
}

// pathCrossesCodexProtectedDir reports whether any path element of
// filepath.Rel(root, target) -- excluding "." (root and target are the same
// path) -- is one of codexProtectedDirNames. It applies the same ancestor
// test pathCovers does (root must be absolute, and target must not require
// walking "up" via ".." from root), so a root that does not actually cover
// target at all always reports false here too, rather than an
// off-the-rel-string false positive; callers still call pathCovers
// separately to tell "does not cover" apart from "covers and does not
// cross" (both false here). A path element that only superficially
// resembles a protected name (".agentsx", "agents" without the leading
// dot) does not match -- comparison is by exact path element, never
// substring. Callers evaluate this on BOTH the resolved and the
// as-configured spelling of a root/target pair (cross-review C2-1): codex
// applies its protection to the path it is actually given, and whether
// that is the literal path or its resolved form is not documented, so a
// symlink that leads back under the same writable root -- but through a
// differently-named directory -- must not be allowed to remove a protected
// element from consideration just because the RESOLVED spelling no longer
// contains it.
func pathCrossesCodexProtectedDir(root, target string) bool {
	if !filepath.IsAbs(root) {
		return false
	}
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == "." {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false // root does not actually cover target
	}
	for _, elem := range strings.Split(rel, string(filepath.Separator)) {
		if codexProtectedDirNames[elem] {
			return true
		}
	}
	return false
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

// codexRootCoverage tests one candidate writable root (explicit or
// implicit) against target, given resolvedTarget (target already resolved
// through resolveNearestExisting -- computed once by the caller, since it
// does not change across candidates). covers is false when root is not
// absolute, or does not genuinely cover target -- decided on the RESOLVED
// pair via pathCovers, exactly as before this function existed. When
// covers is true, blocked reports whether a codex-protected directory
// element sits between root and target on EITHER the resolved pair or the
// pair exactly as configured (cross-review C2-1): a writable_roots entry
// or an agmsg home that reaches the store through a symlink whose target
// lands back under the same root, but through a differently-named
// directory, would otherwise let the RESOLVED spelling hide a protected
// element that the CONFIGURED spelling still shows -- and which spelling
// codex itself evaluates its protection against is not documented, so
// pathCrossesCodexProtectedDir is checked on both (see its own doc
// comment).
func codexRootCoverage(root, target, resolvedTarget string) (covers, blocked bool) {
	if !filepath.IsAbs(root) {
		return false, false // a relative or ~-prefixed root never matches (see pathCovers)
	}
	resolvedRoot := resolveNearestExisting(root)
	if !pathCovers(resolvedRoot, resolvedTarget) {
		return false, false
	}
	blocked = pathCrossesCodexProtectedDir(resolvedRoot, resolvedTarget) || pathCrossesCodexProtectedDir(root, target)
	return true, blocked
}

// coveringWritableRoot returns the first entry in roots that covers target
// without being blocked (codexRootCoverage). A root that covers target
// only by crossing a protected directory (cross-review AR-1 -- e.g. the
// home directory, when the store lives under its .agents subdirectory) is
// recorded as blockedAncestor (the first such root seen) instead of being
// accepted: codex would still leave the store itself read-only, so this
// must not be reported as a real covering root. blockedAncestor is "" when
// either some root genuinely covers target, or no root is an ancestor at
// all. The comparison itself is the ONLY pass over roots: an earlier
// version tried an unresolved textual comparison first, which returned
// pass for a store directory that was itself a symlink pointing OUTSIDE
// every configured root -- exactly the misconfiguration this check exists
// to catch (self-review MEDIUM-3). resolveNearestExisting is strictly at
// least as permissive as a plain textual comparison (it falls back to the
// cleaned path when nothing resolves), so dropping the textual pass loses
// no legitimate match. The returned string is always the ORIGINAL
// (unresolved) root text as written in the config, so the Detail names
// what the operator actually configured.
func coveringWritableRoot(roots []string, target string) (covering, blockedAncestor string) {
	resolvedTarget := resolveNearestExisting(target)
	for _, root := range roots {
		covers, blocked := codexRootCoverage(root, target, resolvedTarget)
		if !covers {
			continue
		}
		if blocked {
			if blockedAncestor == "" {
				blockedAncestor = root
			}
			continue
		}
		return root, ""
	}
	return "", blockedAncestor
}

// codexImplicitRoot pairs one of workspace-write's implicit writable
// directories with the [sandbox_workspace_write] key that disables it and
// whether it came from $TMPDIR -- computed once, alongside the directory
// itself, by codexImplicitWritableRoots. A later match is never re-derived
// by comparing the matched directory's STRING VALUE against slashTmpDir
// (cross-review C2-2): TmpDir can legitimately equal slashTmpDir -- e.g.
// the seat's own $TMPDIR really is codex's fixed temp root too -- in which
// case a post-hoc identity guess cannot tell which
// [sandbox_workspace_write] key actually gated that particular entry, and
// used to name the wrong one whenever the fixed-root entry was excluded
// and the $TMPDIR entry (holding the same directory) matched instead.
type codexImplicitRoot struct {
	Dir           string
	ExcludeKey    string
	FromTmpdirEnv bool
}

// codexImplicitWritableRoots returns the directories workspace-write keeps
// writable by default, regardless of writable_roots (cross-review WC-1):
// slashTmpDir (codex's fixed temp root -- injected by the caller rather
// than hard-coded here, see codexSandboxEnv.SlashTmpDir for why) unless
// [sandbox_workspace_write].exclude_slash_tmp is true
// (docs/evidence/codex-seat-permissions-2026-09-18.md P4 confirms this
// fixed root is writable under workspace-write), and tmpDir (the seat's own
// $TMPDIR) when it is non-empty, absolute, and
// [sandbox_workspace_write].exclude_tmpdir_env_var is not true. Both values
// come from the codexSandboxEnv seam -- never os.Getenv or a hard-coded
// path inside this function -- specifically because every t.TempDir()
// fixture in this package's own tests lives under the real TMPDIR, and on a
// machine or CI runner whose real TMPDIR happens to equal codex's own fixed
// temp root (Go's own default when TMPDIR is unset), a hard-coded path here
// would make those fixtures collide with this exact implicit root and
// silently turn dozens of "should warn" tests into "pass" (the bug this
// seam fixes: cross-review WC-1 follow-up). Order matters:
// codexCoveringImplicitRoot returns the first match.
func codexImplicitWritableRoots(cfg codexUserConfig, slashTmpDir, tmpDir string) []codexImplicitRoot {
	var roots []codexImplicitRoot
	if slashTmpDir != "" && filepath.IsAbs(slashTmpDir) && !cfg.SandboxWorkspaceWrite.ExcludeSlashTmp {
		roots = append(roots, codexImplicitRoot{Dir: slashTmpDir, ExcludeKey: "exclude_slash_tmp"})
	}
	if tmpDir != "" && filepath.IsAbs(tmpDir) && !cfg.SandboxWorkspaceWrite.ExcludeTmpdirEnvVar {
		roots = append(roots, codexImplicitRoot{Dir: tmpDir, ExcludeKey: "exclude_tmpdir_env_var", FromTmpdirEnv: true})
	}
	return roots
}

// codexCoveringImplicitRoot is coveringWritableRoot's counterpart for the
// implicit roots list: the same coverage/blocked semantics
// (codexRootCoverage), but it returns the matched codexImplicitRoot itself
// -- not just its directory -- so the caller can name the exact
// [sandbox_workspace_write] key and pane-environment note that actually
// apply (see codexImplicitRoot's doc comment for why this must not be
// re-derived from the directory string alone). ok is false when no root
// covers target without being blocked; blockedAncestor mirrors
// coveringWritableRoot's.
func codexCoveringImplicitRoot(roots []codexImplicitRoot, target string) (matched codexImplicitRoot, ok bool, blockedAncestor string) {
	resolvedTarget := resolveNearestExisting(target)
	for _, r := range roots {
		covers, blocked := codexRootCoverage(r.Dir, target, resolvedTarget)
		if !covers {
			continue
		}
		if blocked {
			if blockedAncestor == "" {
				blockedAncestor = r.Dir
			}
			continue
		}
		return r, true, ""
	}
	return codexImplicitRoot{}, false, blockedAncestor
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

// codexPaneEnvironmentMayDifferClause is the exact sentence fragment both
// codexWritableRootSuffix's AGMSG_STORAGE_PATH note and
// codexImplicitRootDetail's $TMPDIR note use to say the seat's pane
// environment may not match this process's -- kept as one literal string so
// codexImplicitRootDetail's substring check can reliably tell whether the
// note is already present (cross-review WC-1: don't say it twice).
const codexPaneEnvironmentMayDifferClause = "the seat's pane environment may differ"

// codexImplicitRootDetail renders the Detail for a pass where the agmsg
// store is covered by a directory workspace-write keeps writable by
// default (codexCoveringImplicitRoot's matched entry), not by an explicit
// writable_roots entry (cross-review WC-1). exists distinguishes "the
// config exists but doesn't turn this default off" from "there is no
// config to turn it off in" (the same convention codexNotNeededWhyB and
// codexWritableRootDetail's missing-config form use). matched.ExcludeKey
// names the exact key that gated this entry (never re-derived from the
// directory string -- see codexImplicitRoot); when matched.FromTmpdirEnv,
// the pane-environment note is appended to suffix unless it's already
// there from an AGMSG_STORAGE_PATH override -- the two notes say the same
// thing, so this never repeats it.
func codexImplicitRootDetail(matched codexImplicitRoot, cfgDisplay, store string, exists bool, reasonClause, suffix string) string {
	var keyClause string
	if !exists {
		keyClause = fmt.Sprintf("%s does not exist", cfgDisplay)
	} else {
		keyClause = fmt.Sprintf("%s is not set in %s", matched.ExcludeKey, cfgDisplay)
	}
	if matched.FromTmpdirEnv && !strings.Contains(suffix, codexPaneEnvironmentMayDifferClause) {
		suffix += fmt.Sprintf(" (%s)", codexPaneEnvironmentMayDifferClause)
	}
	return fmt.Sprintf("the agmsg store %s is under %s, which workspace-write keeps writable by default (%s); needed because %s",
		store, matched.Dir, keyClause, reasonClause) + suffix
}

// codexBlockedAncestorClause renders the note codexWritableRootDetail
// inserts when a writable root would cover the store but for a
// codex-protected directory element between them (cross-review AR-1).
// Returns "" when blockedAncestor is "".
func codexBlockedAncestorClause(blockedAncestor string) string {
	if blockedAncestor == "" {
		return ""
	}
	return fmt.Sprintf(" (%s contains it, but codex keeps .git, .agents, and .codex directories under a writable root read-only)", blockedAncestor)
}

// codexWritableRootDetail renders the Detail for the check's last two
// outcomes: root is coveringWritableRoot's (or the implicit-root check's)
// result (non-empty on a pass, "" on a warn). When there is no covering
// root, blockedAncestor -- when non-"" -- names a root that covers the
// store only by crossing a codex-protected directory (cross-review AR-1),
// which codexBlockedAncestorClause renders as a parenthetical right after
// the store path; this can happen in BOTH the absent-config and the
// config-exists warn (cross-review C2-5 -- an absent config still leaves
// [sandbox_workspace_write]'s two exclude keys at their false default, so
// codexImplicitWritableRoots still returns candidates for
// codexCoveringImplicitRoot to test, and one of those can be blocked just
// as an explicit root can). The wording also distinguishes a config that
// exists but simply omits a covering entry from one that does not exist at
// all (self-review LOW-4) -- the latter also names the config path in the
// "add ... in <cfg>" clause, since there is no existing
// [sandbox_workspace_write] table to point the operator at otherwise. The
// absent-config sentence leads with "<cfg> does not exist" rather than
// parenthesizing it after the store path, so it reads as a statement about
// the config, not about the store (self-review NEW-3).
func codexWritableRootDetail(root, blockedAncestor, cfgDisplay, store string, exists bool, reasonClause, suffix string) string {
	if root != "" {
		return fmt.Sprintf("writable root %s in %s covers the agmsg store %s; needed because %s",
			root, cfgDisplay, store, reasonClause) + suffix
	}
	if !exists {
		return fmt.Sprintf("%s does not exist, so no writable root covers the agmsg store %s%s and a codex seat under workspace-write "+
			"cannot send RESULT to lead (\"attempt to write a readonly database\"); needed because %s; add %s to "+
			"[sandbox_workspace_write].writable_roots in %s (docs/recipes/codex-seat-permissions.md)",
			cfgDisplay, store, codexBlockedAncestorClause(blockedAncestor), reasonClause, store, cfgDisplay) + suffix
	}
	return fmt.Sprintf("no writable root in %s covers the agmsg store %s%s, so a codex seat under workspace-write cannot send RESULT to lead "+
		"(\"attempt to write a readonly database\"); needed because %s; add %s to [sandbox_workspace_write].writable_roots "+
		"(docs/recipes/codex-seat-permissions.md)",
		cfgDisplay, store, codexBlockedAncestorClause(blockedAncestor), reasonClause, store) + suffix
}

// checkCodexAgmsgWritableRoot is `ralph doctor`'s "Codex sandbox (agmsg
// writable root)" check (#164): a static read of the user's codex
// config.toml plus the project's [org] envelope, never spawning any
// process. It warns when a codex seat could actually run under
// workspace-write -- via [org.permissions].codex_verified = true with a
// role that resolves to edits or autonomous, or via the config's own
// sandbox_mode = workspace-write with a role that resolves to guarded --
// but no writable_roots entry, nor either of workspace-write's own implicit
// defaults (/tmp, $TMPDIR), actually leaves the agmsg store writable: the
// exact failure that made a codex seat under workspace-write unable to send
// RESULT to lead ("attempt to write a readonly database", docs/evidence/
// codex-seat-permissions-2026-09-18.md P3).
//
// Deterministic outcomes, in order:
//  1. agmsg not installed at agmsgHome (driver.AgmsgAvailable) -> info --
//     org seats aren't usable at all, so there is nothing to check. This is
//     decided by AgmsgAvailable itself, never by the separate "agmsg"
//     check's Status, which can be info for a merely differently-versioned
//     install (see checkAgmsgAvailable).
//  2. codex is not in [org].driver_pool -> pass -- no codex seat can ever
//     be spawned, so nothing downstream is even read (self-review LOW-6).
//  3. [org].model_pool has no codex model -> pass -- same reasoning as
//     outcome 2, one level down (cross-review WC-2).
//  4. resolveEnv fails -> info, no config path to read.
//  5. the config can't be read (permission error, not a regular file, over
//     codexConfigMaxBytes) -> info naming the reason (codexConfigReadReason).
//     The config can't be decoded -- invalid TOML syntax, or syntactically
//     valid TOML with the wrong shape (a duplicate key, a table
//     redefinition, a type mismatch) -- -> info naming only the line/column
//     when one is available; its own message text, which can embed a key or
//     table name taken from the user's config, is never surfaced
//     (codexConfigDecodeError, self-review MEDIUM-1). writable_roots is
//     present but not an array of strings -> info.
//  6. no role could ever make a writable root necessary
//     (codexSeatModesPossible plus codexSandboxReasons finds nothing) ->
//     pass, naming which of the two preconditions failed on each side
//     (codexNotNeededWhyA/codexNotNeededWhyB).
//  7. an explicit writable_roots entry covers the agmsg store directory
//     without crossing a codex-protected directory (coveringWritableRoot)
//     -> pass.
//  8. no explicit root covers it, but one of workspace-write's own
//     implicit defaults does (codexImplicitWritableRoots) -> pass, worded
//     differently to say the store is covered by default rather than by
//     configuration (codexImplicitRootDetail).
//  9. otherwise -> warn, distinguishing a config that exists but omits a
//     covering root from one that does not exist at all, and naming a
//     blocked ancestor root when one exists
//     (codexWritableRootDetail/codexBlockedAncestorClause).
//
// The check never returns "fail" -- a missing root is a real gap, but not
// one that should flip doctor's exit code (see docs/recipes/
// codex-seat-permissions.md for the fix).
//
// This check trusts orgCfg as loaded and does not itself detect a broken
// ralph.toml: runDoctorFull still calls it with config.Load's returned
// cfg.Org even when config.Load itself returned an error (e.g. an invalid
// [org.permissions].default) -- the separate "ralph.toml" check reports
// that load failure on its own line. In that case this check's verdict
// describes the partially-populated defaults Load returned alongside the
// error, not the document the operator actually wrote (self-review NEW-5).
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
	if len(codexModelPoolModels(orgCfg)) == 0 {
		r.Status = "pass"
		r.Detail = "not needed: [org].model_pool has no codex model, so no codex seat can be spawned"
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
	reasonClause := codexSandboxReasonClause(reasons)

	store, fromOverride := agmsgStoreDir(agmsgHome, env.AgmsgStoragePath)
	if abs, absErr := filepath.Abs(store); absErr == nil {
		store = abs
	}
	suffix := codexWritableRootSuffix(fromOverride, cfg.Profile, cfgDisplay)

	root, blockedAncestor := coveringWritableRoot(roots, store)
	if root != "" {
		r.Status = "pass"
		r.Detail = codexWritableRootDetail(root, "", cfgDisplay, store, exists, reasonClause, suffix)
		return r
	}

	implicitRoots := codexImplicitWritableRoots(cfg, env.SlashTmpDir, env.TmpDir)
	matchedImplicit, matchedOK, implicitBlockedAncestor := codexCoveringImplicitRoot(implicitRoots, store)
	if matchedOK {
		r.Status = "pass"
		r.Detail = codexImplicitRootDetail(matchedImplicit, cfgDisplay, store, exists, reasonClause, suffix)
		return r
	}
	if blockedAncestor == "" {
		blockedAncestor = implicitBlockedAncestor
	}

	r.Status = "warn"
	r.Detail = codexWritableRootDetail("", blockedAncestor, cfgDisplay, store, exists, reasonClause, suffix)
	return r
}
