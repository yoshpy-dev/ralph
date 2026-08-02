package org

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// chdir switches the test process's cwd to dir for the duration of the
// test and restores the original cwd on cleanup. os.Chdir is process-global,
// so tests using it must not run in parallel with each other. It returns
// dir with any symlinks resolved (t.TempDir() on macOS returns a path under
// /var/folders, itself a symlink to /private/var/folders; os.Getwd() and
// filepath.Abs() resolve symlinks, so expectations built from the raw
// t.TempDir() string would spuriously mismatch resolver output unless
// callers compare against this resolved form instead).
func chdir(t *testing.T, dir string) string {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("eval symlinks %s: %v", dir, err)
	}
	return resolved
}

// initGitRepo runs `git init` in dir so ResolveOrgStateDir's git-toplevel
// tier has something to resolve against.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init", "--quiet", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init %s: %v\n%s", dir, err, out)
	}
}

func TestResolveOrgStateDir_ExplicitFlagWins(t *testing.T) {
	rawDir := t.TempDir()
	dir := chdir(t, rawDir)

	// A relative explicit flag resolves to an absolute path against cwd,
	// not against a git toplevel or env override, even when both are also
	// present -- explicit flag is the top precedence tier.
	t.Setenv(EnvOrgStateDir, filepath.Join(dir, "env-state"))
	initGitRepo(t, rawDir)

	got, source := ResolveOrgStateDir("relative-state", true)
	want := filepath.Join(dir, "relative-state")
	if got != want {
		t.Errorf("dir = %q, want %q", got, want)
	}
	if source != "flag" {
		t.Errorf("source = %q, want %q", source, "flag")
	}
}

func TestResolveOrgStateDir_EnvWinsOverGitAndCwd(t *testing.T) {
	rawDir := t.TempDir()
	dir := chdir(t, rawDir)
	initGitRepo(t, rawDir)

	envDir := filepath.Join(dir, "env-state")
	t.Setenv(EnvOrgStateDir, envDir)

	got, source := ResolveOrgStateDir("", false)
	if got != envDir {
		t.Errorf("dir = %q, want %q", got, envDir)
	}
	if source != "env" {
		t.Errorf("source = %q, want %q", source, "env")
	}
}

func TestResolveOrgStateDir_GitSubdirResolvesToToplevel(t *testing.T) {
	rawRoot := t.TempDir()
	initGitRepo(t, rawRoot)

	rawSub := filepath.Join(rawRoot, "nested", "deeper")
	if err := os.MkdirAll(rawSub, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rawSub, err)
	}
	chdir(t, rawSub)
	root, err := filepath.EvalSymlinks(rawRoot)
	if err != nil {
		t.Fatalf("eval symlinks %s: %v", rawRoot, err)
	}

	got, source := ResolveOrgStateDir("", false)
	want := filepath.Join(root, defaultOrgStateDirRelPath)
	if got != want {
		t.Errorf("dir = %q, want %q (running from a subdirectory of the repo must still resolve to the toplevel state dir, fixing the lead/operator cwd-split)", got, want)
	}
	if source != "git-toplevel" {
		t.Errorf("source = %q, want %q", source, "git-toplevel")
	}
}

func TestResolveOrgStateDir_GitRoot(t *testing.T) {
	rawRoot := t.TempDir()
	initGitRepo(t, rawRoot)
	root := chdir(t, rawRoot)

	got, source := ResolveOrgStateDir("", false)
	want := filepath.Join(root, defaultOrgStateDirRelPath)
	if got != want {
		t.Errorf("dir = %q, want %q", got, want)
	}
	if source != "git-toplevel" {
		t.Errorf("source = %q, want %q", source, "git-toplevel")
	}
}

func TestResolveOrgStateDir_NonGitCwdFallsBackToCwd(t *testing.T) {
	rawDir := t.TempDir()
	dir := chdir(t, rawDir)

	got, source := ResolveOrgStateDir("", false)
	want := filepath.Join(dir, defaultOrgStateDirRelPath)
	if got != want {
		t.Errorf("dir = %q, want %q", got, want)
	}
	if source != "cwd" {
		t.Errorf("source = %q, want %q", source, "cwd")
	}
}
