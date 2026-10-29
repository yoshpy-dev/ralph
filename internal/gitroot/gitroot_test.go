package gitroot

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// isolateGitEnv keeps the git fixtures below independent of the developer's
// git config and of a GIT_DIR-style variable inherited from a git hook,
// which would otherwise point `git worktree add` at the real repository.
func isolateGitEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unsetenv %s: %v", key, err)
		}
	}
}

// runGit runs git with args and fails the test on a non-zero exit.
func runGit(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// commitEmpty makes the one commit `git worktree add -b` needs.
func commitEmpty(t *testing.T, dir string) {
	t.Helper()
	runGit(t, "-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
		"commit", "--quiet", "--allow-empty", "-m", "init")
}

// mkdirAll creates p and its parents, failing the test on error.
func mkdirAll(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", p, err)
	}
}

// gitFixture is a temporary repository with one commit at main, a linked
// worktree beside it (linked), a linked worktree of a bare repository stored
// as <tmp>/project/.git (bareLinked), and a directory outside git (outside).
// The paths are the raw t.TempDir() ones, not symlink-resolved, so a result
// git printed resolved only matches through SamePath.
type gitFixture struct {
	main, linked, bare, bareLinked, outside string
}

func newGitFixture(t *testing.T) gitFixture {
	t.Helper()
	isolateGitEnv(t)
	tmp := t.TempDir()
	f := gitFixture{
		main:       filepath.Join(tmp, "main"),
		linked:     filepath.Join(tmp, "linked"),
		bare:       filepath.Join(tmp, "project", ".git"),
		bareLinked: filepath.Join(tmp, "project", "wt"),
		outside:    filepath.Join(tmp, "outside"),
	}
	runGit(t, "init", "--quiet", f.main)
	commitEmpty(t, f.main)
	runGit(t, "-C", f.main, "worktree", "add", "--quiet", f.linked, "-b", "linked")
	runGit(t, "clone", "--quiet", "--bare", f.main, f.bare)
	runGit(t, "-C", f.bare, "worktree", "add", "--quiet", f.bareLinked, "-b", "feat")
	for _, dir := range []string{
		f.outside,
		filepath.Join(f.main, "pkg", "sub"),
		filepath.Join(f.linked, "pkg", "sub"),
		filepath.Join(f.bareLinked, "pkg"),
	} {
		mkdirAll(t, dir)
	}
	return f
}

// TestMainWorktree_RunsGitInDir runs every case from a working directory
// outside git, so a main root can only come from the dir argument.
func TestMainWorktree_RunsGitInDir(t *testing.T) {
	f := newGitFixture(t)
	t.Chdir(f.outside)

	for _, tc := range []struct {
		name string
		dir  string
	}{
		{"main worktree root", f.main},
		{"main worktree subdirectory", filepath.Join(f.main, "pkg", "sub")},
		{"linked worktree root", f.linked},
		{"linked worktree subdirectory", filepath.Join(f.linked, "pkg", "sub")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := MainWorktree(tc.dir)
			if !ok || !SamePath(got, f.main) {
				t.Errorf("MainWorktree(%q) = (%q, %v), want (%q, true)", tc.dir, got, ok, f.main)
			}
		})
	}
}

// TestMainWorktree_NoneToName covers the dirs that have no main worktree:
// outside git, a directory that does not exist, a bare repository, and a
// linked worktree of a bare repository stored as .git (git names
// <tmp>/project as its main worktree but marks the record `bare`). The
// working directory is the main worktree, which does have one, so ok false
// can only come from the dir argument.
func TestMainWorktree_NoneToName(t *testing.T) {
	f := newGitFixture(t)
	t.Chdir(f.main)

	for _, tc := range []struct {
		name string
		dir  string
	}{
		{"outside git", f.outside},
		{"missing directory", filepath.Join(f.outside, "missing")},
		{"bare repository", f.bare},
		{"bare repository's linked worktree root", f.bareLinked},
		{"bare repository's linked worktree subdirectory", filepath.Join(f.bareLinked, "pkg")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := MainWorktree(tc.dir); ok {
				t.Errorf("MainWorktree(%q) = (%q, true), want ok false", tc.dir, got)
			}
		})
	}
}

// TestToplevel_RunsGitInDir checks Toplevel against each kind of dir from a
// working directory outside git: the toplevel of the working tree dir is in,
// including a bare repository's linked worktree, and an error outside git.
func TestToplevel_RunsGitInDir(t *testing.T) {
	f := newGitFixture(t)
	t.Chdir(f.outside)

	for _, tc := range []struct {
		name, dir, want string
	}{
		{"main worktree subdirectory", filepath.Join(f.main, "pkg", "sub"), f.main},
		{"linked worktree subdirectory", filepath.Join(f.linked, "pkg", "sub"), f.linked},
		{"bare repository's linked worktree subdirectory", filepath.Join(f.bareLinked, "pkg"), f.bareLinked},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Toplevel(tc.dir)
			if err != nil || !SamePath(got, tc.want) {
				t.Errorf("Toplevel(%q) = (%q, %v), want (%q, nil)", tc.dir, got, err, tc.want)
			}
		})
	}
	t.Run("outside git", func(t *testing.T) {
		if got, err := Toplevel(f.outside); err == nil {
			t.Errorf("Toplevel(%q) = (%q, nil), want an error", f.outside, got)
		}
	})
}

// TestEmptyDirIsTheWorkingDirectory checks that an empty dir runs git in the
// current working directory, the way internal/org's ResolveOrgStateDir calls
// both functions.
func TestEmptyDirIsTheWorkingDirectory(t *testing.T) {
	f := newGitFixture(t)

	t.Run("linked worktree subdirectory", func(t *testing.T) {
		t.Chdir(filepath.Join(f.linked, "pkg", "sub"))
		if got, ok := MainWorktree(""); !ok || !SamePath(got, f.main) {
			t.Errorf("MainWorktree(\"\") = (%q, %v), want (%q, true)", got, ok, f.main)
		}
		if got, err := Toplevel(""); err != nil || !SamePath(got, f.linked) {
			t.Errorf("Toplevel(\"\") = (%q, %v), want (%q, nil)", got, err, f.linked)
		}
	})
	t.Run("outside git", func(t *testing.T) {
		t.Chdir(f.outside)
		if got, ok := MainWorktree(""); ok {
			t.Errorf("MainWorktree(\"\") = (%q, true), want ok false", got)
		}
		if got, err := Toplevel(""); err == nil {
			t.Errorf("Toplevel(\"\") = (%q, nil), want an error", got)
		}
	})
}

func TestSamePath(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "target")
	mkdirAll(t, filepath.Join(target, "sub"))
	link := filepath.Join(tmp, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	for _, tc := range []struct {
		name string
		a, b string
		want bool
	}{
		{"a symlink and its target", link, target, true},
		{"a subdirectory through the symlink and through the target", filepath.Join(link, "sub"), filepath.Join(target, "sub"), true},
		{"missing paths that clean to the same path", filepath.Join(tmp, "gone", "x", ".."), filepath.Join(tmp, "gone"), true},
		{"two different directories", target, tmp, false},
		{"a missing path and an existing one", filepath.Join(tmp, "gone"), target, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := SamePath(tc.a, tc.b); got != tc.want {
				t.Errorf("SamePath(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestParseMainWorktreeRecord(t *testing.T) {
	cases := []struct {
		name      string
		porcelain string
		wantPath  string
		wantOK    bool
	}{
		{
			name:      "main record followed by a linked record",
			porcelain: "worktree /repo/main\nHEAD 1111\nbranch refs/heads/main\n\nworktree /repo/linked\nHEAD 2222\nbranch refs/heads/feat\n\n",
			wantPath:  "/repo/main",
			wantOK:    true,
		},
		{
			name:      "bare main record",
			porcelain: "worktree /repo/project\nbare\n\nworktree /repo/project/wt\nHEAD 2222\nbranch refs/heads/feat\n\n",
		},
		{
			name:      "empty output",
			porcelain: "",
		},
		{
			name:      "garbled output",
			porcelain: "fatal: not a git repository\n",
		},
		{
			name:      "first record without a worktree line",
			porcelain: "HEAD 1111\nbranch refs/heads/main\n\nworktree /repo/linked\n\n",
		},
		{
			name:      "leading blank line ends the first record",
			porcelain: "\nworktree /repo/main\n\n",
		},
		{
			name:      "relative path",
			porcelain: "worktree repo/main\nHEAD 1111\n\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, gotOK := parseMainWorktreeRecord(tc.porcelain)
			if gotPath != tc.wantPath || gotOK != tc.wantOK {
				t.Errorf("parseMainWorktreeRecord(%q) = (%q, %v), want (%q, %v)", tc.porcelain, gotPath, gotOK, tc.wantPath, tc.wantOK)
			}
		})
	}
}
