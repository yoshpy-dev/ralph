package org

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// Permission-mode enum constants (self-review MEDIUM-4): the single,
// grep-able definition of the three permission-mode values, mirroring how
// LeaderIdentity (spawn.go) is the one place the "leader" identity literal is
// spelled. Every internal/org call site that names a permission-mode value
// (ResolvePermissionMode's fallback, permissionArgsForDriver's switch,
// Spawn's AC-2b gate check) must use these constants, not a bare string.
//
// internal/config cannot import internal/org (internal/org already imports
// internal/config -- see this file's own import -- so the reverse would be
// a cycle), so config.go's orgPermissionModeAllowed set necessarily keeps
// its own string literals; a comment there points back at these constants
// as the canonical enum definition.
const (
	PermissionModeAutonomous = "autonomous"
	PermissionModeEdits      = "edits"
	PermissionModeGuarded    = "guarded"
)

// defaultPermissionMode is ResolvePermissionMode's fallback when neither a
// role override nor cfg.Permissions.Default is set. config.Load() already
// backfills an absent [org.permissions].default to "autonomous", so this
// fallback only matters for a config.OrgConfig built by hand (e.g. the
// testOrgConfig() literal most internal/org tests use) rather than through
// Load() -- it mirrors [org.permissions].default's own documented default
// (see config.Default()) so both paths agree.
const defaultPermissionMode = PermissionModeAutonomous

// ResolvePermissionMode returns the driver-independent permission-mode enum
// value (autonomous|edits|guarded) that applies to role: a role-specific
// entry in cfg.Permissions.Roles wins over cfg.Permissions.Default, which in
// turn wins over defaultPermissionMode (see its doc comment for when that
// last fallback triggers). An unknown role simply has no entry in
// cfg.Permissions.Roles, so it falls through to cfg.Permissions.Default like
// every other unmapped role -- there is no separate "unknown role" case to
// handle.
func ResolvePermissionMode(cfg config.OrgConfig, role string) string {
	if mode, ok := cfg.Permissions.Roles[role]; ok && mode != "" {
		return mode
	}
	if cfg.Permissions.Default != "" {
		return cfg.Permissions.Default
	}
	return defaultPermissionMode
}

// codexAutonomousArgs/codexEditsArgs are the codex-native CLI flags applied
// once cfg.Permissions.CodexVerified is true (AC-8): autonomous runs with no
// interactive approval prompt at all (`--ask-for-approval never`, mirroring
// claude's bypassPermissions), edits keeps the default approval policy but
// still grants workspace-write so file edits do not need per-tool-call
// confirmation. These flag shapes were first stated as an assumption in
// docs/plans/archive/2026-08-02-org-runtime-watchdog.md ("codex 権限
// fail-closed の実機検証") and live-verified on 2026-09-18 (see the doc
// comment on permissionArgsForDriver below). CodexVerified is true by
// default; an operator whose installed codex version or config does not pass
// the check in docs/recipes/codex-seat-permissions.md sets it to false, which
// keeps codex seats fail-closed to guarded. For a leader seat, and for a
// reviewer seat of a feature org, Spawn follows either set with
// codexWritableRootArgs (below) when the org state dir is outside the seat's
// cwd. StartDirector refuses a codex headless director in either mode (plan
// 2026-10-15-org-director-start, D12).
var (
	codexAutonomousArgs = []string{"--sandbox", "workspace-write", "--ask-for-approval", "never"}
	codexEditsArgs      = []string{"--sandbox", "workspace-write"}
)

// CodexModeRunsUnderWorkspaceWrite reports whether permission mode mode runs
// a codex agent under `--sandbox workspace-write`: autonomous and edits,
// whose flags (codexAutonomousArgs, codexEditsArgs) both carry it. guarded
// passes no sandbox flag, so codex follows the human's own config. This is
// the one place that set of modes is spelled: checkDirectorStart refuses a
// codex director in them (plan 2026-10-15-org-director-start, D12),
// codexWritableRootArgs gives a codex leader, and the codex reviewer of a
// feature org, in them --add-dir, and ralph doctor's agmsg writable-root
// check counts the seat roles that resolve to them. permissionArgsForDriver
// keeps its own switch, as it picks different flags per mode.
func CodexModeRunsUnderWorkspaceWrite(mode string) bool {
	return mode == PermissionModeAutonomous || mode == PermissionModeEdits
}

// permissionArgsForDriver maps driver + a resolved permission mode to the
// driver-native CLI flags Spawn prepends to AgentStart's agentArgs (before
// the "--model" flags -- see Spawn's agentArgs construction). A nil, nil
// return means "no flag needed" (the driver CLI's own interactive default
// already matches the requested mode). cfg gates the codex fail-closed
// restriction below via cfg.Permissions.CodexVerified (AC-8).
//
// claude accepts --permission-mode directly: autonomous maps to
// bypassPermissions (no interactive permission dialog at all -- this is the
// mode that resolves PR②'s observed "seat blocked on a permission dialog"
// problem), edits maps to acceptEdits (auto-accept file edits, still
// prompts for other tool calls), guarded needs no flag (claude's own
// interactive default already behaves like "guarded").
//
// codex is fail-closed when cfg.Permissions.CodexVerified is false (Codex
// advisory 2, plan Design decisions "codex は fail-closed"): anything other
// than guarded (no flag -- codex's own CLI default) is then rejected
// outright. CodexVerified is true by default, so a project that does not set
// it gets the mapping below. The alternative -- silently emitting a stub
// argv that looks like it applied autonomous/edits but doesn't -- is worse
// than a loud error: a fail-closed codex seat is at least honest about
// running guarded.
//
// The mapping was live-verified on 2026-09-18 against codex-cli 0.154.0
// (docs/evidence/codex-seat-permissions-2026-09-18.md): autonomous runs
// with no approval prompt and the sandbox rejects writes outside its
// writable roots (cwd plus /tmp-style temp roots -- see evidence P4); edits
// auto-accepts in-cwd edits and prompts only when the model requests an
// escalation. That verification, on one machine, is the evidence for the
// mapping, not the reason for the default: the default is true by the human
// decision of issue #274. The flags inherit the operator's own
// ~/.codex/config.toml (approval_policy, sandbox writable_roots -- the
// agmsg DB directory must be writable or seats cannot send RESULT), so a
// machine where docs/recipes/codex-seat-permissions.md does not pass sets
// it to false.
func permissionArgsForDriver(cfg config.OrgConfig, driver, mode string) ([]string, error) {
	switch driver {
	case "claude":
		switch mode {
		case PermissionModeAutonomous:
			return []string{"--permission-mode", "bypassPermissions"}, nil
		case PermissionModeEdits:
			return []string{"--permission-mode", "acceptEdits"}, nil
		case PermissionModeGuarded:
			return nil, nil
		default:
			return nil, fmt.Errorf("org: unknown permission mode %q for driver %q", mode, driver)
		}
	case "codex":
		switch mode {
		case PermissionModeGuarded:
			return nil, nil
		case PermissionModeAutonomous:
			if cfg.Permissions.CodexVerified {
				return codexAutonomousArgs, nil
			}
			return nil, fmt.Errorf("org: codex seat permission mode %q requires [org.permissions].codex_verified=true; only guarded is allowed until then (fail-closed; verify the codex flag mapping on this machine first: docs/recipes/codex-seat-permissions.md)", mode)
		case PermissionModeEdits:
			if cfg.Permissions.CodexVerified {
				return codexEditsArgs, nil
			}
			return nil, fmt.Errorf("org: codex seat permission mode %q requires [org.permissions].codex_verified=true; only guarded is allowed until then (fail-closed; verify the codex flag mapping on this machine first: docs/recipes/codex-seat-permissions.md)", mode)
		default:
			return nil, fmt.Errorf("org: unknown permission mode %q for driver %q", mode, driver)
		}
	default:
		return nil, fmt.Errorf("org: unknown driver %q for permission mode mapping", driver)
	}
}

// codexWritableRootArgs returns `--add-dir <stateDir>` for a codex leader
// seat (role == LeaderIdentity), or a codex reviewer seat (role ==
// reviewerRole) of a feature org (featureOrg: the org has an active binding,
// ActiveFeature), whose permission mode runs it under
// `--sandbox workspace-write` (CodexModeRunsUnderWorkspaceWrite: autonomous
// or edits; Spawn only reaches this after permissionArgsForDriver has
// accepted that mode, i.e. once cfg.Permissions.CodexVerified is true) when
// the org state directory is not inside the seat's cwd, and nil otherwise.
// workspace-write leaves only cwd (plus temp roots) writable, so a codex
// seat whose cwd is a linked worktree could not append to the shared
// ledger under the main worktree's .harness/state/org without it (plan
// 2026-10-07-org-state-dir-common, AC8). guarded passes no sandbox flag at
// all and claude has no such sandbox, so both get nil.
//
// The leader gets the flag because it runs `ralph org` verbs and so writes
// the ledger. In a feature org the reviewer gets it too: it runs
// `ralph org gate` at the leader's TASK and so writes the gate record and
// the gate log to the ledger (plan 2026-10-26-reviewer-runs-gate, D5). Like
// the leader, that reviewer can then write the whole ledger directly (the
// manifest with its plan_approved and gate_passed records, the inbox, the
// seats' prompt files), and the gate runs the repo's test code as its child
// inside that sandbox; only the role prompt keeps it to the records
// `ralph org gate` writes. The implementer, the reviewer of an org with no
// feature binding, and every other role (including one with no embedded
// template that runs from --prompt) report to the leader over agmsg and get
// nil. Making the state dir writable for them would let any seat rewrite
// the manifest, the model receipts, and other seats' prompt files under
// <state-dir>/prompts/. The headless director also writes the ledger, but
// StartDirector refuses a codex director in autonomous and edits (plan
// 2026-10-15-org-director-start, D12), so it never runs under
// workspace-write and never needs the flag.
//
// The containment test compares both paths made absolute and resolved
// through symlinks (resolvedOrClean, statedir.go: macOS's /var is a symlink
// to /private/var), so a state dir under cwd spelled through a different
// symlink still counts as inside. resolvedOrClean resolves a path all or
// nothing, so this relies on stateDir already existing: Spawn calls it
// after withManifestLock has created the directory (lockfile.go). A state
// dir that did not exist yet would compare unresolved against a resolved
// cwd (/var/... against /private/var/...) and count as outside. The flag
// carries the absolute, unresolved state dir -- the same path the runtime
// itself writes to.
func codexWritableRootArgs(role, driver, mode, cwd, stateDir string, featureOrg bool) []string {
	writesLedger := role == LeaderIdentity || (featureOrg && role == reviewerRole)
	if !writesLedger || driver != "codex" || !CodexModeRunsUnderWorkspaceWrite(mode) {
		return nil
	}
	stateDir = mustAbs(stateDir)
	if pathWithinDir(resolvedOrClean(stateDir), resolvedOrClean(mustAbs(cwd))) {
		return nil
	}
	return []string{"--add-dir", stateDir}
}

// pathWithinDir reports whether path is dir itself or a descendant of it,
// both already cleaned absolute paths. It compares path elements via
// filepath.Rel, so /x/wt-other is not inside /x/wt.
func pathWithinDir(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
