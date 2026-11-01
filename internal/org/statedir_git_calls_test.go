package org

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/gitroot"
)

// Tests of the git ResolveOrgStateDir runs on its default tiers, and of
// LedgerLocalConfigPath, which reads the personal file's path back from the
// ledger (plan 2026-11-01-org-ledger-gaps, AC7). Every test chdirs, so none
// of them runs in parallel.

// logGitCalls puts a git first on PATH that appends its arguments to a log,
// one line per start, and then runs the real git. The function it returns
// gives the lines logged since its last call and empties the log.
func logGitCalls(t *testing.T) func() []string {
	t.Helper()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("git is needed: %v", err)
	}
	if realGit, err = filepath.Abs(realGit); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	gitLog := filepath.Join(t.TempDir(), "git.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$ORG_TEST_GIT_LOG\"\nexec " + shellQuote(realGit) + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatalf("write the git wrapper: %v", err)
	}
	t.Setenv("ORG_TEST_GIT_LOG", gitLog)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return func() []string {
		t.Helper()
		data, err := os.ReadFile(gitLog)
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("read %s: %v", gitLog, err)
		}
		if err := os.Remove(gitLog); err != nil && !os.IsNotExist(err) {
			t.Fatalf("remove %s: %v", gitLog, err)
		}
		text := strings.TrimSuffix(string(data), "\n")
		if text == "" {
			return nil
		}
		return strings.Split(text, "\n")
	}
}

// defaultTierDirs holds a working directory for each default tier of
// ResolveOrgStateDir: the main checkout and a linked worktree of a normal
// repository (git-main-worktree), a linked worktree of a bare repository
// stored as <tmp>/project/.git (git-toplevel), and a directory outside git
// (cwd). All paths are symlink-resolved.
type defaultTierDirs struct {
	mainRoot, linked, bareLinked, outside string
}

func newDefaultTierDirs(t *testing.T) defaultTierDirs {
	t.Helper()
	mainRoot, linked, _ := newRepoWithLinkedWorktrees(t)
	tmp := resolved(t, t.TempDir())
	src := filepath.Join(tmp, "src")
	runGit(t, "init", "--quiet", src)
	commitEmpty(t, src)
	bare := filepath.Join(tmp, "project", ".git")
	runGit(t, "clone", "--quiet", "--bare", src, bare)
	d := defaultTierDirs{
		mainRoot:   mainRoot,
		linked:     linked,
		bareLinked: filepath.Join(tmp, "project", "wt"),
		outside:    filepath.Join(tmp, "outside"),
	}
	runGit(t, "-C", bare, "worktree", "add", "--quiet", d.bareLinked, "-b", "feat")
	mkdirAll(t, d.outside)
	return d
}

// TestResolveOrgStateDir_LooksForItsDirectoryOnce covers AC7: with the
// personal file read (config.LocalConfigEnv ""), ResolveOrgStateDir runs the
// git commands of one lookup of the directory of its default tiers
// (gitroot.MainWorktree(""), then gitroot.Toplevel("") when that names
// none), in the same order, whether the personal file is missing or names
// the ledger, and returns the same dir and source as before. Before, it
// looked once with `git -C <working directory>` to find the personal file
// (config.LocalOrgStateDir) and, unless that file named the ledger, once
// more for its own tiers.
func TestResolveOrgStateDir_LooksForItsDirectoryOnce(t *testing.T) {
	d := newDefaultTierDirs(t)
	ledger := filepath.Join(resolved(t, t.TempDir()), "ledger")
	t.Setenv(config.LocalConfigEnv, "")
	gitCalls := logGitCalls(t)

	for _, tc := range []struct {
		name, cwd, local, wantDir, wantSource string
	}{
		{"main checkout, no personal file", d.mainRoot, "", filepath.Join(d.mainRoot, defaultOrgStateDirRelPath), "git-main-worktree"},
		{"linked worktree, no personal file", d.linked, "", filepath.Join(d.mainRoot, defaultOrgStateDirRelPath), "git-main-worktree"},
		{"linked worktree, a personal file naming the ledger", d.linked, "[org]\nstate_dir = \"" + ledger + "\"\n", ledger, "config"},
		{"a bare repository's linked worktree", d.bareLinked, "", filepath.Join(d.bareLinked, defaultOrgStateDirRelPath), "git-toplevel"},
		{"outside git", d.outside, "", filepath.Join(d.outside, defaultOrgStateDirRelPath), "cwd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.local != "" {
				path := writeLocalConfig(t, d.mainRoot, tc.local)
				t.Cleanup(func() { _ = os.Remove(path) })
			}
			chdir(t, tc.cwd)
			gitCalls()
			if _, ok := gitroot.MainWorktree(""); !ok {
				_, _ = gitroot.Toplevel("")
			}
			oneLookup := gitCalls()
			if len(oneLookup) == 0 {
				t.Fatal("the git wrapper logged no call for one lookup; it is not the git on PATH")
			}

			dir, source := resolveOrgStateDir(t, "", false)
			got := gitCalls()
			if dir != tc.wantDir || source != tc.wantSource {
				t.Errorf("ResolveOrgStateDir = %q, %q; want %q, %q", dir, source, tc.wantDir, tc.wantSource)
			}
			if !slices.Equal(got, oneLookup) {
				t.Errorf("ResolveOrgStateDir started git %d times:\n%s\nwant the %d starts of one lookup:\n%s",
					len(got), strings.Join(got, "\n"), len(oneLookup), strings.Join(oneLookup, "\n"))
			}
		})
	}
}

// TestLedgerLocalConfigPath covers the path the refusal of
// checkRetiredOrgStateDirEnv (internal/cli) names (AC7): for each default
// tier, it is the ralph.local.toml config.LocalPath("ralph.toml") gives from
// the same working directory, read back from the ledger with no git call.
// The sources flag and config, and a dir that does not end with
// .harness/state/org, have none.
func TestLedgerLocalConfigPath(t *testing.T) {
	d := newDefaultTierDirs(t)
	gitCalls := logGitCalls(t)

	for _, tc := range []struct{ name, cwd, wantSource string }{
		{"main checkout", d.mainRoot, "git-main-worktree"},
		{"linked worktree subdirectory", filepath.Join(d.linked, "pkg"), "git-main-worktree"},
		{"a bare repository's linked worktree", d.bareLinked, "git-toplevel"},
		{"outside git", d.outside, "cwd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mkdirAll(t, tc.cwd)
			chdir(t, tc.cwd)
			dir, source := resolveOrgStateDir(t, "", false)
			if source != tc.wantSource {
				t.Fatalf("ResolveOrgStateDir source = %q, want %q", source, tc.wantSource)
			}
			want := config.LocalPath("ralph.toml")
			if calls := gitCalls(); len(calls) == 0 {
				t.Fatal("the git wrapper logged no call for ResolveOrgStateDir and config.LocalPath; it is not the git on PATH")
			}

			got, ok := LedgerLocalConfigPath(dir, source)
			if calls := gitCalls(); len(calls) != 0 {
				t.Errorf("LedgerLocalConfigPath started git %d times: %q; want none", len(calls), calls)
			}
			if !ok || got != want {
				t.Errorf("LedgerLocalConfigPath(%q, %q) = %q, %t; want %q, true", dir, source, got, ok, want)
			}
		})
	}

	t.Run("a ledger at the filesystem root", func(t *testing.T) {
		dir := filepath.Join(string(filepath.Separator), defaultOrgStateDirRelPath)
		want := filepath.Join(string(filepath.Separator), config.LocalConfigFileName)
		if got, ok := LedgerLocalConfigPath(dir, "cwd"); !ok || got != want {
			t.Errorf("LedgerLocalConfigPath(%q, cwd) = %q, %t; want %q, true", dir, got, ok, want)
		}
	})

	stateDir := filepath.Join(d.mainRoot, defaultOrgStateDirRelPath)
	for _, tc := range []struct{ name, dir, source string }{
		{"flag", stateDir, "flag"},
		{"config", stateDir, "config"},
		{"the retired env source", stateDir, "env"},
		{"git-main-worktree dir not ending with the state-dir path", filepath.Join(d.mainRoot, "state"), "git-main-worktree"},
		{"git-toplevel dir not ending with the state-dir path", filepath.Join(d.bareLinked, "state"), "git-toplevel"},
		{"cwd dir not ending with the state-dir path", filepath.Join(d.outside, "state"), "cwd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := LedgerLocalConfigPath(tc.dir, tc.source); ok || got != "" {
				t.Errorf("LedgerLocalConfigPath(%q, %q) = %q, %t; want \"\", false", tc.dir, tc.source, got, ok)
			}
		})
	}
}
