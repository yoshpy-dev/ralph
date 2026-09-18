package org

import (
	"fmt"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// Permission-mode enum constants (self-review MEDIUM-4): the single,
// grep-able definition of the three permission-mode values, mirroring how
// LeadIdentity (spawn.go) is the one place the "lead" identity literal is
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
// comment on permissionArgsForDriver below); CodexVerified is the operator's
// explicit acknowledgement that they have repeated that verification for
// their installed codex version and config.
var (
	codexAutonomousArgs = []string{"--sandbox", "workspace-write", "--ask-for-approval", "never"}
	codexEditsArgs      = []string{"--sandbox", "workspace-write"}
)

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
// codex is fail-closed by default (Codex advisory 2, plan Design decisions
// "codex は fail-closed"): anything other than guarded (no flag -- codex's
// own CLI default) is rejected outright unless cfg.Permissions.CodexVerified
// is true. The alternative -- silently emitting a stub argv that looks like
// it applied autonomous/edits but doesn't -- is worse than a loud error: a
// fail-closed codex seat is at least honest about running guarded.
//
// The mapping was live-verified on 2026-09-18 against codex-cli 0.154.0
// (docs/evidence/codex-seat-permissions-2026-09-18.md): autonomous runs
// with no approval prompt and the sandbox rejects writes outside its
// writable roots (cwd plus /tmp-style temp roots -- see evidence P4); edits
// auto-accepts in-cwd edits and prompts only when the model requests an
// escalation. The default stays false by design rather than for lack of
// verification: the flags inherit the operator's own ~/.codex/config.toml
// (approval_policy, sandbox writable_roots -- the agmsg DB directory must be
// writable or seats cannot send RESULT), so each machine opts in after
// running docs/recipes/codex-seat-permissions.md.
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
