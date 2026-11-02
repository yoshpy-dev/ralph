package org

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// TestFeatureRepoRoot_LinkedWorktreeOfBareRepositoryHasNoMainWorktree covers
// AC9 of plan 2026-11-01-org-ledger-gaps for the branch of FeatureRepoRoot
// where git names no main worktree but a toplevel exists: a linked worktree
// of a bare repository (the fixture of
// TestResolveOrgStateDir_BareDotGitWorktreeFallsBackToToplevel). The result
// is an error that names the toplevel and says such a worktree has no main
// worktree, for the ledger the default tiers found (git-toplevel) and for the
// ones --state-dir and the personal file name, from the worktree's root and
// from a subdirectory.
func TestFeatureRepoRoot_LinkedWorktreeOfBareRepositoryHasNoMainWorktree(t *testing.T) {
	isolateGitEnv(t)
	tmp := resolved(t, t.TempDir())
	src := filepath.Join(tmp, "src")
	runGit(t, "init", "--quiet", src)
	commitEmpty(t, src)
	bare := filepath.Join(tmp, "project", ".git")
	runGit(t, "clone", "--quiet", "--bare", src, bare)
	wt := filepath.Join(tmp, "project", "wt")
	runGit(t, "-C", bare, "worktree", "add", "--quiet", wt, "-b", "feat")
	elsewhere := filepath.Join(tmp, "elsewhere", defaultOrgStateDirRelPath)

	for name, cwd := range map[string]string{"worktree root": wt, "worktree subdirectory": filepath.Join(wt, "pkg")} {
		t.Run(name, func(t *testing.T) {
			mkdirAll(t, cwd)
			chdir(t, cwd)

			defaultLedger, defaultSource := resolveOrgStateDir(t, "", false)
			if defaultSource != "git-toplevel" {
				t.Fatalf("source = %q, want %q (the fixture should leave git naming no main worktree)", defaultSource, "git-toplevel")
			}
			for _, tc := range []struct{ dir, source string }{
				{defaultLedger, defaultSource},
				{elsewhere, "flag"},
				{elsewhere, "config"},
			} {
				root, err := FeatureRepoRoot(tc.dir, tc.source)
				if err == nil {
					t.Errorf("FeatureRepoRoot(%q, %q) = %q, nil; want an error", tc.dir, tc.source, root)
					continue
				}
				if root != "" {
					t.Errorf("FeatureRepoRoot(%q, %q) root = %q with an error; want none", tc.dir, tc.source, root)
				}
				for _, want := range []string{"a linked worktree of a bare repository has none", "git names none for " + wt + " "} {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("FeatureRepoRoot(%q, %q) error = %q; want it to contain %q", tc.dir, tc.source, err, want)
					}
				}
				if strings.Contains(err.Error(), "is not in one") {
					t.Errorf("FeatureRepoRoot(%q, %q) error = %q; want the bare repository sentence, not the outside-git one", tc.dir, tc.source, err)
				}
			}
		})
	}
}

// TestMustAbs_KeepsThePathAsGivenWhenAbsFails covers AC9 of plan
// 2026-11-01-org-ledger-gaps for the filepath.Abs failure branch of mustAbs,
// which filepath.Abs takes only when os.Getwd fails. Removing the working
// directory does not make os.Getwd fail on macOS (syscall.Getwd still
// returns the old path), so the test reassigns absPath, the seam of
// spawn.go, to a failing one. With it, a relative path comes back as it was
// given, from mustAbs and from ResolveOrgStateDir's --state-dir tier, and
// without it mustAbs returns an absolute path.
func TestMustAbs_KeepsThePathAsGivenWhenAbsFails(t *testing.T) {
	const rel = "rel/x"
	if got := mustAbs(rel); !filepath.IsAbs(got) || !strings.HasSuffix(got, string(filepath.Separator)+rel) {
		t.Fatalf("mustAbs(%q) = %q with filepath.Abs working; want an absolute path ending in it", rel, got)
	}

	orig := absPath
	absPath = func(string) (string, error) { return "", errors.New("stub: getwd failed") }
	t.Cleanup(func() { absPath = orig })

	if got := mustAbs(rel); got != rel {
		t.Errorf("mustAbs(%q) = %q when Abs fails; want %q as given", rel, got, rel)
	}
	dir, source, err := ResolveOrgStateDir(rel, true)
	if err != nil || dir != rel || source != "flag" {
		t.Errorf("ResolveOrgStateDir(%q, true) = %q, %q, %v when Abs fails; want %q, %q, nil", rel, dir, source, err, rel, "flag")
	}
}
