package org

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/gitroot"
)

// defaultOrgStateDirRelPath is the state-dir path segment appended to a
// resolved git-main-worktree, git-toplevel, or cwd default. It mirrors the
// historical --state-dir flag default (".harness/state/org"), which is now
// applied by ResolveOrgStateDir instead of by the flag itself.
const defaultOrgStateDirRelPath = ".harness/state/org"

// ResolveOrgStateDir resolves the org state directory using the precedence
// order documented in docs/plans/archive/2026-08-02-org-runtime-watchdog.md
// (tech-debt: "state-dir の cwd 相対解決" -- the leader/operator cwd-split),
// extended to linked worktrees by plan 2026-10-07-org-state-dir-common and
// to the personal file by plan 2026-10-30-org-state-dir-local, which also
// retired the RALPH_ORG_STATE_DIR tier that came after the flag (D5: the
// variable is not read here; internal/cli reads it only to say it is still
// set, or to refuse a verb that would change another ledger than the one it
// names):
//
//  1. explicit flag ("flag") -- explicitSet is true (the caller passed
//     --state-dir; detected via cobra's cmd.Flags().Changed("state-dir")
//     rather than "explicit != default", since the flag's own default is
//     now an empty string precisely so a merely-defaulted value can never
//     be confused with an explicit one). explicit is resolved to an
//     absolute path against the current working directory if it is
//     relative. The personal file is not read.
//  2. personal file ("config") -- the [org].state_dir of the
//     ralph.local.toml at the root of the main worktree of the repository
//     containing the current working directory
//     (config.LocalOrgStateDir("")), when it sets a non-empty one: an
//     absolute path, or one that starts with "~/". It is not the file a
//     --config names (D1 of that plan). With config.LocalConfigEnv "off"
//     the file is not read. A file that cannot be read or parsed, or whose
//     state_dir is neither absolute nor starts with "~/", is err (naming
//     the file), and no lower tier is tried: falling through would put the
//     caller on another ledger without saying so (D4).
//  3. git main worktree ("git-main-worktree") -- the root of the
//     repository's main worktree (see gitroot.MainWorktree), joined with
//     ".harness/state/org". This is what fixes the leader/operator
//     cwd-split, including across worktrees: every org verb invoked from
//     anywhere inside the main checkout or any of its linked worktrees
//     resolves to the same state directory, so one org ledger (manifest,
//     receipts, watch state) is shared by all of them. In the main checkout
//     this is the same path the git-toplevel tier below would return; only
//     the tag differs.
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
// re-deriving this same logic. err is set only by tier 2, and then dir and
// source are "".
func ResolveOrgStateDir(explicit string, explicitSet bool) (dir, source string, err error) {
	if explicitSet {
		return mustAbs(explicit), "flag", nil
	}
	local, _, err := config.LocalOrgStateDir("")
	if err != nil {
		return "", "", err
	}
	if local != "" {
		return local, "config", nil
	}
	if root, ok := gitroot.MainWorktree(""); ok {
		return filepath.Join(root, defaultOrgStateDirRelPath), "git-main-worktree", nil
	}
	if top, err := gitroot.Toplevel(""); err == nil && top != "" {
		return filepath.Join(top, defaultOrgStateDirRelPath), "git-toplevel", nil
	}
	return mustAbs(defaultOrgStateDirRelPath), "cwd", nil
}

// MainWorktreeRoot returns the main worktree root that ResolveOrgStateDir
// joined with ".harness/state/org" to produce resolvedDir, when source is
// "git-main-worktree". It reads the root back from resolvedDir instead of
// running git again, so it names the root the ledger lives under. ok is
// false for every other source (flag, config, git-toplevel, cwd), and
// when resolvedDir does not end with that path.
func MainWorktreeRoot(resolvedDir, source string) (root string, ok bool) {
	if source != "git-main-worktree" {
		return "", false
	}
	root = resolvedDir
	for range strings.Split(defaultOrgStateDirRelPath, "/") {
		root = filepath.Dir(root)
	}
	if filepath.Join(root, defaultOrgStateDirRelPath) != resolvedDir {
		return "", false
	}
	return root, true
}

// LedgerMainWorktreeRoot returns the main worktree root whose ledger
// resolvedDir is: MainWorktreeRoot for source "git-main-worktree", and for
// a ledger named by --state-dir or the personal file's [org].state_dir
// (source "flag" or "config") the main worktree of the repository
// containing the current working directory
// (gitroot.MainWorktree), when resolvedDir is that root's
// .harness/state/org (compared with gitroot.SamePath). The leader of a
// feature org passes --state-dir for the ledger start used
// (featureLeaderTask), so its spawns from the feature worktree still find
// the main worktree this way. ok is false for any other directory or
// source, and when git names no main worktree.
func LedgerMainWorktreeRoot(resolvedDir, source string) (root string, ok bool) {
	switch source {
	case "git-main-worktree":
		return MainWorktreeRoot(resolvedDir, source)
	case "flag", "config":
	default:
		return "", false
	}
	root, ok = gitroot.MainWorktree("")
	if !ok || !gitroot.SamePath(resolvedDir, filepath.Join(root, defaultOrgStateDirRelPath)) {
		return "", false
	}
	return root, true
}

// FeatureRepoRoot returns the main worktree root that `ralph org start --plan`
// passes as StartFeatureParams.RepoRoot, for the ledger ResolveOrgStateDir
// returned as resolvedDir and source: MainWorktreeRoot when it names one,
// otherwise (a ledger chosen with --state-dir or the personal file's
// [org].state_dir) the main worktree of the repository containing the
// current working directory (gitroot.MainWorktree). Outside a git working
// tree, and in a repository with no main worktree to name (a linked
// worktree of a bare repository), it is an error.
func FeatureRepoRoot(resolvedDir, source string) (string, error) {
	if root, ok := MainWorktreeRoot(resolvedDir, source); ok {
		return root, nil
	}
	if root, ok := gitroot.MainWorktree(""); ok {
		return root, nil
	}
	if top, err := gitroot.Toplevel(""); err == nil && top != "" {
		return "", fmt.Errorf("org: start --plan makes the feature worktree from the main worktree, and git names none for %s "+
			"(a linked worktree of a bare repository has none): run it from a repository with a main worktree", top)
	}
	return "", errors.New("org: start --plan makes the feature worktree from the main worktree of a git repository, " +
		"and the current directory is not in one: run it from the main worktree or one of its linked worktrees")
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
// any other source, since a flag or personal-file choice is deliberate --
// and that is not an error; err is non-nil only when the
// legacy manifest exists but cannot be read.
func LegacyWorktreeStateDir(resolvedDir, source string) (legacyDir string, activeSeats int, err error) {
	if source != "git-main-worktree" {
		return "", 0, nil
	}
	top, topErr := gitroot.Toplevel("")
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
	if gitroot.SamePath(candidate, resolvedDir) {
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

// resolvedOrClean is gitroot.ResolvedOrClean (symlinks resolved where p
// exists, cleaned otherwise) under the name the rest of this package
// (feature.go, permissions.go, split.go) calls it by.
func resolvedOrClean(p string) string {
	return gitroot.ResolvedOrClean(p)
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
