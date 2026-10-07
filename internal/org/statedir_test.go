package org

import (
	"fmt"
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

// initGitRepo runs `git init` in dir so ResolveOrgStateDir's
// git-main-worktree tier has something to resolve against.
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
		t.Errorf("dir = %q, want %q (running from a subdirectory of the repo must still resolve to the toplevel state dir, fixing the leader/operator cwd-split)", got, want)
	}
	if source != "git-main-worktree" {
		t.Errorf("source = %q, want %q", source, "git-main-worktree")
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
	if source != "git-main-worktree" {
		t.Errorf("source = %q, want %q", source, "git-main-worktree")
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

// resolved returns p with symlinks resolved, for comparing against
// resolver output (see chdir).
func resolved(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatalf("eval symlinks %s: %v", p, err)
	}
	return r
}

// mkdirAll creates p and its parents, failing the test on error.
func mkdirAll(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", p, err)
	}
}

// newRepoWithLinkedWorktrees creates a normal repository at <tmp>/main with
// one commit and two linked worktrees: one beside it (<tmp>/sibling) and
// one nested inside the main checkout (<main>/.claude/worktrees/nested,
// ralph's own task-worktree layout). All returned paths are symlink-resolved.
func newRepoWithLinkedWorktrees(t *testing.T) (mainRoot, sibling, nested string) {
	t.Helper()
	isolateGitEnv(t)
	tmp := resolved(t, t.TempDir())
	mainRoot = filepath.Join(tmp, "main")
	runGit(t, "init", "--quiet", mainRoot)
	commitEmpty(t, mainRoot)
	sibling = filepath.Join(tmp, "sibling")
	nested = filepath.Join(mainRoot, ".claude", "worktrees", "nested")
	runGit(t, "-C", mainRoot, "worktree", "add", "--quiet", sibling, "-b", "sibling")
	runGit(t, "-C", mainRoot, "worktree", "add", "--quiet", nested, "-b", "nested")
	return mainRoot, sibling, nested
}

func TestResolveOrgStateDir_LinkedWorktreeResolvesToMainWorktree(t *testing.T) {
	mainRoot, sibling, nested := newRepoWithLinkedWorktrees(t)
	want := filepath.Join(mainRoot, defaultOrgStateDirRelPath)

	cases := []struct {
		name string
		cwd  string
	}{
		{"main root", mainRoot},
		{"main subdirectory", filepath.Join(mainRoot, "pkg", "sub")},
		{"sibling worktree root", sibling},
		{"sibling worktree subdirectory", filepath.Join(sibling, "pkg", "sub")},
		{"nested worktree root", nested},
		{"nested worktree subdirectory", filepath.Join(nested, "pkg", "sub")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mkdirAll(t, tc.cwd)
			chdir(t, tc.cwd)

			got, source := ResolveOrgStateDir("", false)
			if got != want {
				t.Errorf("dir = %q, want %q (every worktree of the repository must share the main worktree's ledger)", got, want)
			}
			if source != "git-main-worktree" {
				t.Errorf("source = %q, want %q", source, "git-main-worktree")
			}
		})
	}
}

// TestResolveOrgStateDir_BareDotGitWorktreeFallsBackToToplevel covers a
// bare repository stored as <tmp>/project/.git: git names <tmp>/project as
// its main worktree but marks the record `bare`, so a linked worktree keeps
// its own show-toplevel state dir instead of picking up <tmp>/project.
func TestResolveOrgStateDir_BareDotGitWorktreeFallsBackToToplevel(t *testing.T) {
	isolateGitEnv(t)
	tmp := resolved(t, t.TempDir())
	src := filepath.Join(tmp, "src")
	runGit(t, "init", "--quiet", src)
	commitEmpty(t, src)
	bare := filepath.Join(tmp, "project", ".git")
	runGit(t, "clone", "--quiet", "--bare", src, bare)
	wt := filepath.Join(tmp, "project", "wt")
	runGit(t, "-C", bare, "worktree", "add", "--quiet", wt, "-b", "feat")
	want := filepath.Join(wt, defaultOrgStateDirRelPath)

	for name, cwd := range map[string]string{"worktree root": wt, "worktree subdirectory": filepath.Join(wt, "pkg")} {
		t.Run(name, func(t *testing.T) {
			mkdirAll(t, cwd)
			chdir(t, cwd)

			got, source := ResolveOrgStateDir("", false)
			if got != want {
				t.Errorf("dir = %q, want %q", got, want)
			}
			if source != "git-toplevel" {
				t.Errorf("source = %q, want %q", source, "git-toplevel")
			}
		})
	}
}

// TestResolveOrgStateDir_SeparateGitDir covers `git init --separate-git-dir`.
// git's own `worktree list` names the separated git dir's parent (or the git
// dir itself when it is not named .git) as the main worktree, so neither may
// leak into the result.
func TestResolveOrgStateDir_SeparateGitDir(t *testing.T) {
	t.Run("main checkout, git dir named .git", func(t *testing.T) {
		isolateGitEnv(t)
		tmp := resolved(t, t.TempDir())
		mkdirAll(t, filepath.Join(tmp, "meta"))
		work := filepath.Join(tmp, "work")
		runGit(t, "init", "--quiet", "--separate-git-dir", filepath.Join(tmp, "meta", ".git"), work)
		commitEmpty(t, work)
		chdir(t, work)

		got, source := ResolveOrgStateDir("", false)
		want := filepath.Join(work, defaultOrgStateDirRelPath)
		if got != want {
			t.Errorf("dir = %q, want %q (not under the separated git dir's parent %q)", got, want, filepath.Join(tmp, "meta"))
		}
		if source != "git-main-worktree" {
			t.Errorf("source = %q, want %q", source, "git-main-worktree")
		}
	})

	t.Run("linked worktree, git dir not named .git", func(t *testing.T) {
		isolateGitEnv(t)
		tmp := resolved(t, t.TempDir())
		mkdirAll(t, filepath.Join(tmp, "meta"))
		work := filepath.Join(tmp, "work")
		runGit(t, "init", "--quiet", "--separate-git-dir", filepath.Join(tmp, "meta", "repo.git"), work)
		commitEmpty(t, work)
		wt := filepath.Join(tmp, "wt")
		runGit(t, "-C", work, "worktree", "add", "--quiet", wt, "-b", "feat")
		chdir(t, wt)

		got, source := ResolveOrgStateDir("", false)
		want := filepath.Join(wt, defaultOrgStateDirRelPath)
		if got != want {
			t.Errorf("dir = %q, want %q (not inside the git dir)", got, want)
		}
		if source != "git-toplevel" {
			t.Errorf("source = %q, want %q", source, "git-toplevel")
		}
	})
}

// TestResolveOrgStateDir_MissingMainWorktreeFallsBackToToplevel stubs git on
// PATH: real git derives the main worktree path from the common dir, so a
// missing main worktree directory cannot be built with a real repository.
func TestResolveOrgStateDir_MissingMainWorktreeFallsBackToToplevel(t *testing.T) {
	tmp := chdir(t, t.TempDir())
	top := filepath.Join(tmp, "linked")
	stubDir := filepath.Join(tmp, "bin")
	mkdirAll(t, stubDir)
	script := fmt.Sprintf(`#!/bin/sh
case "$*" in
"rev-parse --path-format=absolute --git-dir --git-common-dir") printf '%%s\n%%s\n' '%s' '%s' ;;
"worktree list --porcelain") printf 'worktree %%s\nHEAD 0000\nbranch refs/heads/main\n\n' '%s' ;;
"rev-parse --show-toplevel") printf '%%s\n' '%s' ;;
*) exit 1 ;;
esac
`, filepath.Join(tmp, "gone", ".git", "worktrees", "linked"), filepath.Join(tmp, "gone", ".git"), filepath.Join(tmp, "gone"), top)
	if err := os.WriteFile(filepath.Join(stubDir, "git"), []byte(script), 0o755); err != nil {
		t.Fatalf("write git stub: %v", err)
	}
	t.Setenv("PATH", stubDir)

	got, source := ResolveOrgStateDir("", false)
	want := filepath.Join(top, defaultOrgStateDirRelPath)
	if got != want {
		t.Errorf("dir = %q, want %q", got, want)
	}
	if source != "git-toplevel" {
		t.Errorf("source = %q, want %q", source, "git-toplevel")
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

// appendLedgerEvents writes events to the manifest under stateDir with the
// same ManifestStore the org verbs use.
func appendLedgerEvents(t *testing.T, stateDir string, events ...ManifestEvent) {
	t.Helper()
	store := NewManifestStoreAtPath(ManifestPathIn(stateDir))
	for _, ev := range events {
		ev.TS = "2026-10-07T00:00:00Z"
		if err := store.Append(ev); err != nil {
			t.Fatalf("append %+v: %v", ev, err)
		}
	}
}

func TestLegacyWorktreeStateDir(t *testing.T) {
	t.Run("ledger with one active seat in a linked worktree", func(t *testing.T) {
		_, sibling, _ := newRepoWithLinkedWorktrees(t)
		legacy := filepath.Join(sibling, defaultOrgStateDirRelPath)
		appendLedgerEvents(t, legacy,
			ManifestEvent{OrgID: "org-a", SeatID: "leader", Event: EventSpawned},
			ManifestEvent{OrgID: "org-b", SeatID: "worker", Event: EventSpawned},
			ManifestEvent{OrgID: "org-b", SeatID: "worker", Event: EventStopped},
			ManifestEvent{OrgID: "org-c", SeatID: "probe", Event: EventSpawned, DryRun: true},
		)
		cwd := filepath.Join(sibling, "pkg")
		mkdirAll(t, cwd)
		chdir(t, cwd)

		dir, source := ResolveOrgStateDir("", false)
		gotDir, active, err := LegacyWorktreeStateDir(dir, source)
		if err != nil {
			t.Fatalf("LegacyWorktreeStateDir: %v", err)
		}
		if gotDir != legacy {
			t.Errorf("legacy dir = %q, want %q", gotDir, legacy)
		}
		if active != 1 {
			t.Errorf("active seats = %d, want 1 (stopped and dry-run seats are not active)", active)
		}
	})

	t.Run("ledger with no active seat is still reported", func(t *testing.T) {
		_, _, nested := newRepoWithLinkedWorktrees(t)
		legacy := filepath.Join(nested, defaultOrgStateDirRelPath)
		appendLedgerEvents(t, legacy,
			ManifestEvent{OrgID: "org-a", SeatID: "leader", Event: EventSpawned},
			ManifestEvent{OrgID: "org-a", SeatID: "implementer", Event: EventSpawned},
			ManifestEvent{OrgID: "org-a", Event: EventDisbanded},
		)
		chdir(t, nested)

		dir, source := ResolveOrgStateDir("", false)
		gotDir, active, err := LegacyWorktreeStateDir(dir, source)
		if err != nil {
			t.Fatalf("LegacyWorktreeStateDir: %v", err)
		}
		if gotDir != legacy {
			t.Errorf("legacy dir = %q, want %q", gotDir, legacy)
		}
		if active != 0 {
			t.Errorf("active seats = %d, want 0 (the org was disbanded)", active)
		}
	})

	t.Run("no ledger in the linked worktree", func(t *testing.T) {
		mainRoot, sibling, _ := newRepoWithLinkedWorktrees(t)
		appendLedgerEvents(t, filepath.Join(mainRoot, defaultOrgStateDirRelPath),
			ManifestEvent{OrgID: "org-a", SeatID: "leader", Event: EventSpawned},
		)
		chdir(t, sibling)

		dir, source := ResolveOrgStateDir("", false)
		gotDir, active, err := LegacyWorktreeStateDir(dir, source)
		if err != nil || gotDir != "" || active != 0 {
			t.Errorf("LegacyWorktreeStateDir = (%q, %d, %v), want (\"\", 0, nil)", gotDir, active, err)
		}
	})

	t.Run("main checkout ledger is not legacy", func(t *testing.T) {
		mainRoot, _, _ := newRepoWithLinkedWorktrees(t)
		appendLedgerEvents(t, filepath.Join(mainRoot, defaultOrgStateDirRelPath),
			ManifestEvent{OrgID: "org-a", SeatID: "leader", Event: EventSpawned},
		)
		cwd := filepath.Join(mainRoot, "pkg")
		mkdirAll(t, cwd)
		chdir(t, cwd)

		dir, source := ResolveOrgStateDir("", false)
		gotDir, active, err := LegacyWorktreeStateDir(dir, source)
		if err != nil || gotDir != "" || active != 0 {
			t.Errorf("LegacyWorktreeStateDir = (%q, %d, %v), want (\"\", 0, nil)", gotDir, active, err)
		}
	})

	t.Run("flag or env choice is never second-guessed", func(t *testing.T) {
		_, sibling, _ := newRepoWithLinkedWorktrees(t)
		appendLedgerEvents(t, filepath.Join(sibling, defaultOrgStateDirRelPath),
			ManifestEvent{OrgID: "org-a", SeatID: "leader", Event: EventSpawned},
		)
		chdir(t, sibling)

		flagDir, flagSource := ResolveOrgStateDir(filepath.Join(t.TempDir(), "state"), true)
		if gotDir, active, err := LegacyWorktreeStateDir(flagDir, flagSource); err != nil || gotDir != "" || active != 0 {
			t.Errorf("source %q: LegacyWorktreeStateDir = (%q, %d, %v), want (\"\", 0, nil)", flagSource, gotDir, active, err)
		}

		t.Setenv(EnvOrgStateDir, filepath.Join(t.TempDir(), "state"))
		envDir, envSource := ResolveOrgStateDir("", false)
		if gotDir, active, err := LegacyWorktreeStateDir(envDir, envSource); err != nil || gotDir != "" || active != 0 {
			t.Errorf("source %q: LegacyWorktreeStateDir = (%q, %d, %v), want (\"\", 0, nil)", envSource, gotDir, active, err)
		}
	})

	t.Run("unreadable ledger is an error", func(t *testing.T) {
		_, sibling, _ := newRepoWithLinkedWorktrees(t)
		// A directory where manifest.jsonl should be: stat succeeds, the
		// read fails.
		mkdirAll(t, ManifestPathIn(filepath.Join(sibling, defaultOrgStateDirRelPath)))
		chdir(t, sibling)

		dir, source := ResolveOrgStateDir("", false)
		if _, _, err := LegacyWorktreeStateDir(dir, source); err == nil {
			t.Error("LegacyWorktreeStateDir: want an error for an unreadable legacy manifest, got nil")
		}
	})
}
