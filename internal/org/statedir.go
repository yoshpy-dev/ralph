package org

import (
	"errors"
	"fmt"
	"io/fs"
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
// resolved git-main-worktree, git-toplevel, or cwd default. It mirrors the
// historical --state-dir flag default (".harness/state/org"), which is now
// applied by ResolveOrgStateDir instead of by the flag itself.
const defaultOrgStateDirRelPath = ".harness/state/org"

// ResolveOrgStateDir resolves the org state directory using the precedence
// order documented in docs/plans/archive/2026-08-02-org-runtime-watchdog.md
// (tech-debt: "state-dir の cwd 相対解決" -- the leader/operator cwd-split)
// and extended to linked worktrees by plan 2026-10-07-org-state-dir-common:
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
//  3. git main worktree ("git-main-worktree") -- the root of the
//     repository's main worktree (see gitMainWorktree), joined with
//     ".harness/state/org". This is what fixes the leader/operator
//     cwd-split, including across worktrees: every org verb invoked from
//     anywhere inside the main checkout or any of its linked worktrees
//     resolves to the same state directory, so one org ledger (manifest,
//     receipts, watch state) is shared by all of them. In the main checkout
//     this is the same path tier 4 used to return; only the tag differs.
//  4. git toplevel ("git-toplevel") -- `git rev-parse --show-toplevel` run
//     in the current working directory, joined with ".harness/state/org".
//     Used when tier 3 cannot name a main worktree, e.g. in a linked
//     worktree of a bare repository, so each such worktree keeps its own
//     state directory as before.
//  5. cwd fallback ("cwd") -- the current working directory joined with
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
	if root, ok := gitMainWorktree(); ok {
		return filepath.Join(root, defaultOrgStateDirRelPath), "git-main-worktree"
	}
	if top, err := gitToplevel(); err == nil && top != "" {
		return filepath.Join(top, defaultOrgStateDirRelPath), "git-toplevel"
	}
	return mustAbs(defaultOrgStateDirRelPath), "cwd"
}

// LegacyWorktreeStateDir reports the per-worktree org state directory that
// ResolveOrgStateDir returned from the current working directory before the
// git-main-worktree tier existed, when it still holds a ledger: cwd is
// inside a linked worktree, ResolveOrgStateDir returned resolvedDir with
// source "git-main-worktree", and
// <worktree toplevel>/.harness/state/org/manifest.jsonl exists. The ledger
// is never moved or merged; callers only refuse or warn about it.
//
// activeSeats counts the seats that ledger's manifest still records as
// active, with the same Roster(events, RosterOptions{}) derivation `ralph
// org status` and ActiveSeatCount use (dry-run seats excluded), across all
// org_ids. legacyDir is "" when there is no such ledger -- including for
// any other source, since a flag or env choice is deliberate -- and that is
// not an error; err is non-nil only when the legacy manifest exists but
// cannot be read.
func LegacyWorktreeStateDir(resolvedDir, source string) (legacyDir string, activeSeats int, err error) {
	if source != "git-main-worktree" {
		return "", 0, nil
	}
	top, topErr := gitToplevel()
	if topErr != nil || top == "" {
		return "", 0, nil
	}
	candidate := filepath.Join(top, defaultOrgStateDirRelPath)
	manifestPath := ManifestPathIn(candidate)
	if _, statErr := os.Stat(manifestPath); statErr != nil {
		if errors.Is(statErr, fs.ErrNotExist) {
			return "", 0, nil
		}
		return "", 0, fmt.Errorf("org: stat %s: %w", manifestPath, statErr)
	}
	if samePath(candidate, resolvedDir) {
		return "", 0, nil
	}
	seats, readErr := NewManifestStoreAtPath(manifestPath).Roster(RosterOptions{})
	if readErr != nil {
		return "", 0, readErr
	}
	for _, s := range seats {
		if s.Active {
			activeSeats++
		}
	}
	return candidate, activeSeats, nil
}

// gitMainWorktree returns the root of the main worktree of the repository
// containing the current working directory. ok is false when there is none
// to name (not in git, a bare repository, a git failure), and
// ResolveOrgStateDir falls through to the git-toplevel tier.
//
// In the main worktree itself (git dir == common dir) the root is
// `git rev-parse --show-toplevel`: `git worktree list` names the main
// worktree after its common dir minus "/.git", which for a repository made
// with `git init --separate-git-dir <elsewhere>/.git <worktree>` is
// <elsewhere>, not the worktree. In a linked worktree the root is the first
// record of `git worktree list --porcelain` (see parseMainWorktreeRecord),
// skipped when that path is the common dir itself (a separated git dir not
// named .git, so git has no worktree path for it) or is not an existing
// directory. A linked worktree of a separate-git-dir repository whose git
// dir is named .git still resolves under <elsewhere>: git reports that path
// and nothing in the repository records the real main worktree.
func gitMainWorktree() (string, bool) {
	out, err := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-dir", "--git-common-dir").Output()
	if err != nil {
		return "", false
	}
	dirs := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(dirs) != 2 {
		return "", false
	}
	gitDir, commonDir := dirs[0], dirs[1]
	if samePath(gitDir, commonDir) {
		top, topErr := gitToplevel()
		if topErr != nil || top == "" {
			return "", false
		}
		return top, true
	}
	list, err := exec.Command("git", "worktree", "list", "--porcelain").Output()
	if err != nil {
		return "", false
	}
	root, ok := parseMainWorktreeRecord(string(list))
	if !ok || samePath(root, commonDir) {
		return "", false
	}
	if info, statErr := os.Stat(root); statErr != nil || !info.IsDir() {
		return "", false
	}
	return root, true
}

// parseMainWorktreeRecord reads the first record of `git worktree list
// --porcelain` output (the lines up to the first blank line), which git
// always emits for the main worktree. It returns that record's
// `worktree <path>` value, with ok false when the record is marked `bare`,
// has no `worktree` line, or names a relative path.
func parseMainWorktreeRecord(porcelain string) (path string, ok bool) {
	bare := false
	for _, line := range strings.Split(porcelain, "\n") {
		if line == "" {
			break
		}
		switch {
		case strings.HasPrefix(line, "worktree "):
			path = strings.TrimPrefix(line, "worktree ")
		case line == "bare":
			bare = true
		}
	}
	if bare || path == "" || !filepath.IsAbs(path) {
		return "", false
	}
	return path, true
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

// samePath reports whether a and b name the same location, comparing with
// symlinks resolved where the path exists (macOS's /var is a symlink to
// /private/var, and git prints resolved paths) and cleaned otherwise.
func samePath(a, b string) bool {
	return resolvedOrClean(a) == resolvedOrClean(b)
}

func resolvedOrClean(p string) string {
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return filepath.Clean(p)
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
