// Package gitroot answers where a git working tree's roots are: the root of
// the repository's main worktree (MainWorktree), the toplevel of the working
// tree a directory is in (Toplevel), and whether two paths name the same
// location (SamePath). internal/org resolves its ledger with it (statedir.go)
// and internal/config finds ralph.local.toml with it, so both agree on which
// directory is the main worktree's root. Every function runs git in the
// directory it is given (`git -C <dir>`), or in the current working directory
// when that is empty.
package gitroot

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MainWorktree returns the root of the main worktree of the repository
// containing dir (the current working directory when dir is empty). ok is
// false when there is none to name (not in git, a bare repository, a git
// failure); internal/org's ResolveOrgStateDir then falls through to its
// git-toplevel tier.
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
func MainWorktree(dir string) (root string, ok bool) {
	out, err := command(dir, "rev-parse", "--path-format=absolute", "--git-dir", "--git-common-dir").Output()
	if err != nil {
		return "", false
	}
	dirs := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(dirs) != 2 {
		return "", false
	}
	gitDir, commonDir := dirs[0], dirs[1]
	if SamePath(gitDir, commonDir) {
		top, topErr := Toplevel(dir)
		if topErr != nil || top == "" {
			return "", false
		}
		return top, true
	}
	list, err := command(dir, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return "", false
	}
	root, ok = parseMainWorktreeRecord(string(list))
	if !ok || SamePath(root, commonDir) {
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

// Toplevel runs `git rev-parse --show-toplevel` in dir (the current working
// directory when dir is empty). A dir outside git (or any other git failure)
// returns a non-nil error -- internal/org's ResolveOrgStateDir treats that as
// "fall through to the cwd default", not as a fatal condition.
func Toplevel(dir string) (string, error) {
	out, err := command(dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// SamePath reports whether a and b name the same location, comparing with
// symlinks resolved where the path exists (macOS's /var is a symlink to
// /private/var, and git prints resolved paths) and cleaned otherwise.
func SamePath(a, b string) bool {
	return ResolvedOrClean(a) == ResolvedOrClean(b)
}

// ResolvedOrClean returns p with its symlinks resolved when p exists, and
// filepath.Clean(p) otherwise. It resolves a path all or nothing: a path
// that does not exist yet under a symlinked parent stays unresolved.
func ResolvedOrClean(p string) string {
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return filepath.Clean(p)
}

// command returns the git command for args, run with `git -C dir` when dir
// is not empty and in the current working directory otherwise. The empty
// dir keeps the argument list exactly as it was before dir existed: the git
// stub of internal/org's statedir_test.go matches whole argument lists, and
// TestStatusCmd_MergeGuaranteeRunsNoGH in internal/cli counts the calls that
// start with rev-parse.
func command(dir string, args ...string) *exec.Cmd {
	if dir != "" {
		args = append([]string{"-C", dir}, args...)
	}
	return exec.Command("git", args...)
}
