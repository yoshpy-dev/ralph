package org

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// EnvOrgStateDir is the environment variable that overrides the org
// state-dir default when the caller did not explicitly pass --state-dir.
// See ResolveOrgStateDir for the full precedence order.
const EnvOrgStateDir = "RALPH_ORG_STATE_DIR"

// defaultOrgStateDirRelPath is the state-dir path segment appended to a
// resolved git-toplevel or cwd default. It mirrors the historical
// --state-dir flag default (".harness/state/org"), which is now applied by
// ResolveOrgStateDir instead of by the flag itself.
const defaultOrgStateDirRelPath = ".harness/state/org"

// ResolveOrgStateDir resolves the org state directory using the precedence
// order documented in docs/plans/active/2026-08-02-org-runtime-watchdog.md
// (tech-debt: "state-dir の cwd 相対解決" -- the lead/operator cwd-split):
//
//  1. explicit flag ("flag") -- explicitSet is true (the caller passed
//     --state-dir; detected via cobra's cmd.Flags().Changed("state-dir")
//     rather than "explicit != default", since the flag's own default is
//     now an empty string precisely so a merely-defaulted value can never
//     be confused with an explicit one). explicit is resolved to an
//     absolute path against the current working directory if it is
//     relative.
//  2. env RALPH_ORG_STATE_DIR ("env") -- consulted only when the flag was
//     not explicitly set.
//  3. git toplevel ("git-toplevel") -- `git rev-parse --show-toplevel` run
//     in the current working directory, joined with
//     ".harness/state/org". This is what fixes the lead/operator
//     cwd-split: every org verb invoked from anywhere inside the same
//     repository resolves to the same state directory, regardless of
//     which subdirectory the caller's shell happens to be in.
//  4. cwd fallback ("cwd") -- the current working directory joined with
//     ".harness/state/org", used when cwd is not inside a git working
//     tree (e.g. a scratch directory, or a non-git project).
//
// The returned dir is always an absolute path; source is a grep-able tag
// naming which precedence tier produced it, useful for diagnostics without
// re-deriving this same logic.
func ResolveOrgStateDir(explicit string, explicitSet bool) (dir string, source string) {
	if explicitSet {
		return mustAbs(explicit), "flag"
	}
	if env := strings.TrimSpace(os.Getenv(EnvOrgStateDir)); env != "" {
		return mustAbs(env), "env"
	}
	if top, err := gitToplevel(); err == nil && top != "" {
		return filepath.Join(top, defaultOrgStateDirRelPath), "git-toplevel"
	}
	return mustAbs(defaultOrgStateDirRelPath), "cwd"
}

// gitToplevel runs `git rev-parse --show-toplevel` in the current working
// directory. A non-git cwd (or any other git failure) returns a non-nil
// error -- ResolveOrgStateDir treats that as "fall through to the cwd
// default", not as a fatal condition.
func gitToplevel() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// mustAbs resolves p to an absolute path against the current working
// directory. filepath.Abs only fails when os.Getwd fails, a condition none
// of ResolveOrgStateDir's callers can meaningfully recover from -- fall back
// to the original (relative) value on that practically-unreachable failure
// rather than panicking.
func mustAbs(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}
