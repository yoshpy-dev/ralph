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

// permissionArgsForDriver maps driver + a resolved permission mode to the
// driver-native CLI flags Spawn prepends to AgentStart's agentArgs (before
// the "--model" flags -- see Spawn's agentArgs construction). A nil, nil
// return means "no flag needed" (the driver CLI's own interactive default
// already matches the requested mode).
//
// claude accepts --permission-mode directly: autonomous maps to
// bypassPermissions (no interactive permission dialog at all -- this is the
// mode that resolves PR②'s observed "seat blocked on a permission dialog"
// problem), edits maps to acceptEdits (auto-accept file edits, still
// prompts for other tool calls), guarded needs no flag (claude's own
// interactive default already behaves like "guarded").
//
// codex is fail-closed (Codex advisory 2, plan Design decisions
// "codex は fail-closed"): codex's interactive-mode permission/sandbox
// flags have not been live-verified against a real codex seat, so anything
// other than guarded (no flag -- codex's own CLI default) is rejected
// outright. The alternative -- silently emitting a stub argv that looks
// like it applied autonomous/edits but doesn't -- is worse than a loud
// error: a fail-closed codex seat is at least honest about running
// guarded. See docs/tech-debt/README.md for the live-verification
// follow-up that lifts this restriction.
func permissionArgsForDriver(driver, mode string) ([]string, error) {
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
		if mode == PermissionModeGuarded {
			return nil, nil
		}
		return nil, fmt.Errorf("org: codex seat permission mode %q not yet live-verified; only guarded is allowed (fail-closed)", mode)
	default:
		return nil, fmt.Errorf("org: unknown driver %q for permission mode mapping", driver)
	}
}
