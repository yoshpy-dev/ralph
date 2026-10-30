package org

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
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

// retiredOrgStateDirEnv is the variable that named the ledger before plan
// 2026-10-30-org-state-dir-local (D5). ResolveOrgStateDir no longer reads
// it; the tests set it to show that.
const retiredOrgStateDirEnv = "RALPH_ORG_STATE_DIR"

// resolveOrgStateDir is ResolveOrgStateDir for a test that expects it to
// resolve: an error fails the test.
func resolveOrgStateDir(t *testing.T, explicit string, explicitSet bool) (dir, source string) {
	t.Helper()
	dir, source, err := ResolveOrgStateDir(explicit, explicitSet)
	if err != nil {
		t.Fatalf("ResolveOrgStateDir(%q, %t): %v", explicit, explicitSet, err)
	}
	return dir, source
}

func TestResolveOrgStateDir_ExplicitFlagWins(t *testing.T) {
	rawDir := t.TempDir()
	dir := chdir(t, rawDir)

	// A relative explicit flag resolves to an absolute path against cwd,
	// not against a git toplevel, even when one is present -- explicit flag
	// is the top precedence tier.
	initGitRepo(t, rawDir)

	got, source := resolveOrgStateDir(t, "relative-state", true)
	want := filepath.Join(dir, "relative-state")
	if got != want {
		t.Errorf("dir = %q, want %q", got, want)
	}
	if source != "flag" {
		t.Errorf("source = %q, want %q", source, "flag")
	}
}

// TestResolveOrgStateDir_RetiredEnvIsIgnored covers AC6 of plan
// 2026-10-30-org-state-dir-local at the resolver: RALPH_ORG_STATE_DIR, which
// named the ledger before the flag's defaults, no longer does. In a git
// repository the main worktree's ledger is returned, and outside git the
// cwd's, with the variable set to another directory in both.
func TestResolveOrgStateDir_RetiredEnvIsIgnored(t *testing.T) {
	rawDir := t.TempDir()
	dir := chdir(t, rawDir)
	initGitRepo(t, rawDir)
	t.Setenv(retiredOrgStateDirEnv, filepath.Join(dir, "env-state"))

	got, source := resolveOrgStateDir(t, "", false)
	if want := filepath.Join(dir, defaultOrgStateDirRelPath); got != want || source != "git-main-worktree" {
		t.Errorf("in git: ResolveOrgStateDir = %q, %q; want %q, git-main-worktree", got, source, want)
	}

	outside := chdir(t, t.TempDir())
	got, source = resolveOrgStateDir(t, "", false)
	if want := filepath.Join(outside, defaultOrgStateDirRelPath); got != want || source != "cwd" {
		t.Errorf("outside git: ResolveOrgStateDir = %q, %q; want %q, cwd", got, source, want)
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

	got, source := resolveOrgStateDir(t, "", false)
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

	got, source := resolveOrgStateDir(t, "", false)
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

	got, source := resolveOrgStateDir(t, "", false)
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

			got, source := resolveOrgStateDir(t, "", false)
			if got != want {
				t.Errorf("dir = %q, want %q (every worktree of the repository must share the main worktree's ledger)", got, want)
			}
			if source != "git-main-worktree" {
				t.Errorf("source = %q, want %q", source, "git-main-worktree")
			}
		})
	}
}

// writeLocalConfig writes body as the personal ralph.local.toml at root
// and turns reading it back on: TestMain sets config.LocalConfigEnv to off.
func writeLocalConfig(t *testing.T, root, body string) string {
	t.Helper()
	t.Setenv(config.LocalConfigEnv, "")
	path := filepath.Join(root, config.LocalConfigFileName)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// TestResolveOrgStateDir_PersonalFileTier covers the config tier (plan
// 2026-10-30-org-state-dir-local, AC1 and AC2): the [org].state_dir of the
// ralph.local.toml at the main worktree's root names the ledger, with
// source "config", from the main checkout, a subdirectory, and linked
// worktrees beside and inside it, whose own ralph.local.toml is not read. A
// value that starts with "~/" is under the home directory, and an absolute
// one is cleaned.
func TestResolveOrgStateDir_PersonalFileTier(t *testing.T) {
	mainRoot, sibling, nested := newRepoWithLinkedWorktrees(t)
	ledger := filepath.Join(t.TempDir(), "ledger")
	writeLocalConfig(t, mainRoot, "[org]\nstate_dir = \""+ledger+"/../ledger\"\n")
	writeLocalConfig(t, sibling, "[org]\nstate_dir = \"/the-sibling-file\"\n")

	for name, cwd := range map[string]string{
		"main root":                    mainRoot,
		"main subdirectory":            filepath.Join(mainRoot, "pkg", "sub"),
		"sibling worktree root":        sibling,
		"nested worktree subdirectory": filepath.Join(nested, "pkg"),
	} {
		t.Run(name, func(t *testing.T) {
			mkdirAll(t, cwd)
			chdir(t, cwd)
			if dir, source := resolveOrgStateDir(t, "", false); dir != ledger || source != "config" {
				t.Errorf("ResolveOrgStateDir = %q, %q; want %q, config", dir, source, ledger)
			}
		})
	}

	t.Run("a value under the home directory", func(t *testing.T) {
		home := resolved(t, t.TempDir())
		t.Setenv("HOME", home)
		writeLocalConfig(t, mainRoot, "[org]\nstate_dir = \"~/ledgers/app\"\n")
		chdir(t, sibling)
		want := filepath.Join(home, "ledgers", "app")
		if dir, source := resolveOrgStateDir(t, "", false); dir != want || source != "config" {
			t.Errorf("ResolveOrgStateDir = %q, %q; want %q, config", dir, source, want)
		}
	})

	// The returned dir is absolute also when the home directory is not
	// (self-review L6 of plan 2026-10-30-org-state-dir-local): a relative
	// ledger would be read against another cwd by a child or a hint.
	t.Run("a value under a relative home directory", func(t *testing.T) {
		t.Setenv("HOME", filepath.Join("rel", "home"))
		writeLocalConfig(t, mainRoot, "[org]\nstate_dir = \"~/ledgers/app\"\n")
		cwd := chdir(t, sibling)
		want := filepath.Join(cwd, "rel", "home", "ledgers", "app")
		if dir, source := resolveOrgStateDir(t, "", false); dir != want || source != "config" {
			t.Errorf("ResolveOrgStateDir = %q, %q; want %q, config", dir, source, want)
		}
	})
}

// TestResolveOrgStateDir_PersonalFileOrder covers where the config tier
// stands: --state-dir comes before it, RALPH_ORG_STATE_DIR (which came
// before it until D5 of plan 2026-10-30-org-state-dir-local) no longer
// does, and the git tiers come after it, also when the file is missing,
// sets no state_dir, sets an empty one, or is not read
// (config.LocalConfigEnv "off", AC9). The flag does not read the file at
// all, so a broken one does not stop it.
func TestResolveOrgStateDir_PersonalFileOrder(t *testing.T) {
	mainRoot, sibling, _ := newRepoWithLinkedWorktrees(t)
	shared := filepath.Join(mainRoot, defaultOrgStateDirRelPath)
	ledger := filepath.Join(resolved(t, t.TempDir()), "ledger")
	chdir(t, sibling)

	t.Run("the flag beats the file, and does not read it", func(t *testing.T) {
		flagDir := filepath.Join(sibling, "flag-ledger")
		for _, body := range []string{"[org]\nstate_dir = \"" + ledger + "\"\n", "[org]\nstate_dir = \"rel\"\n", "[org\n"} {
			writeLocalConfig(t, mainRoot, body)
			if dir, source := resolveOrgStateDir(t, "flag-ledger", true); dir != flagDir || source != "flag" {
				t.Errorf("with %q: ResolveOrgStateDir = %q, %q; want %q, flag", body, dir, source, flagDir)
			}
		}
	})

	t.Run("the retired env does not beat the file", func(t *testing.T) {
		writeLocalConfig(t, mainRoot, "[org]\nstate_dir = \""+ledger+"\"\n")
		t.Setenv(retiredOrgStateDirEnv, filepath.Join(t.TempDir(), "env-ledger"))
		if dir, source := resolveOrgStateDir(t, "", false); dir != ledger || source != "config" {
			t.Errorf("ResolveOrgStateDir = %q, %q; want %q, config", dir, source, ledger)
		}
	})

	for _, tc := range []struct{ name, body string }{
		{"no state_dir", "[org]\nmax_seats = 4\n"},
		{"an empty state_dir", "[org]\nstate_dir = \"\"\n"},
	} {
		t.Run(tc.name+" falls through to git", func(t *testing.T) {
			writeLocalConfig(t, mainRoot, tc.body)
			if dir, source := resolveOrgStateDir(t, "", false); dir != shared || source != "git-main-worktree" {
				t.Errorf("ResolveOrgStateDir = %q, %q; want %q, git-main-worktree", dir, source, shared)
			}
		})
	}

	t.Run("no file falls through to git", func(t *testing.T) {
		t.Setenv(config.LocalConfigEnv, "")
		if err := os.Remove(filepath.Join(mainRoot, config.LocalConfigFileName)); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if dir, source := resolveOrgStateDir(t, "", false); dir != shared || source != "git-main-worktree" {
			t.Errorf("ResolveOrgStateDir = %q, %q; want %q, git-main-worktree", dir, source, shared)
		}
	})

	t.Run("RALPH_LOCAL_CONFIG=off skips the file", func(t *testing.T) {
		for _, body := range []string{"[org]\nstate_dir = \"" + ledger + "\"\n", "[org]\nstate_dir = \"rel\"\n"} {
			writeLocalConfig(t, mainRoot, body)
			t.Setenv(config.LocalConfigEnv, "off")
			if dir, source := resolveOrgStateDir(t, "", false); dir != shared || source != "git-main-worktree" {
				t.Errorf("with %q: ResolveOrgStateDir = %q, %q; want %q, git-main-worktree", body, dir, source, shared)
			}
		}
	})
}

// TestResolveOrgStateDir_PersonalFileErrors covers D4 of plan
// 2026-10-30-org-state-dir-local: a personal file that does not parse, or
// whose state_dir is neither absolute nor starts with "~/", is an error
// naming the file, with no dir and no source, rather than a fall to the git
// tiers.
func TestResolveOrgStateDir_PersonalFileErrors(t *testing.T) {
	mainRoot, sibling, _ := newRepoWithLinkedWorktrees(t)
	chdir(t, sibling)
	for _, tc := range []struct{ name, body, want string }{
		{"a relative state_dir", "[org]\nstate_dir = \"rel/ledger\"\n", `[org].state_dir "rel/ledger" must be an absolute path or start with ~/`},
		{"~ alone", "[org]\nstate_dir = \"~\"\n", `[org].state_dir "~" must be an absolute path or start with ~/`},
		{"~user", "[org]\nstate_dir = \"~someone/ledger\"\n", `[org].state_dir "~someone/ledger" must be an absolute path or start with ~/`},
		{"not TOML", "[org\nstate_dir = \"/ledger\"\n", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeLocalConfig(t, mainRoot, tc.body)
			dir, source, err := ResolveOrgStateDir("", false)
			if err == nil || dir != "" || source != "" {
				t.Fatalf("ResolveOrgStateDir = %q, %q, %v; want \"\", \"\" and an error", dir, source, err)
			}
			if !strings.HasPrefix(err.Error(), path+": ") || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v; want it to start with %q and say %q", err, path+": ", tc.want)
			}
		})
	}
}

// TestMainWorktreeRoot covers the accessor `ralph org spawn` uses to read the
// org-wide limits from the main worktree's ralph.toml: from every worktree of
// the repository it reads back the main root from the resolved state dir, and
// no other source has a main root.
func TestMainWorktreeRoot(t *testing.T) {
	mainRoot, sibling, nested := newRepoWithLinkedWorktrees(t)
	for name, cwd := range map[string]string{
		"main subdirectory":            filepath.Join(mainRoot, "pkg"),
		"sibling worktree root":        sibling,
		"nested worktree subdirectory": filepath.Join(nested, "pkg"),
	} {
		t.Run(name, func(t *testing.T) {
			mkdirAll(t, cwd)
			chdir(t, cwd)

			dir, source := resolveOrgStateDir(t, "", false)
			if root, ok := MainWorktreeRoot(dir, source); !ok || root != mainRoot {
				t.Errorf("MainWorktreeRoot(%q, %q) = %q, %t, want %q, true", dir, source, root, ok, mainRoot)
			}
		})
	}

	stateDir := filepath.Join(mainRoot, defaultOrgStateDirRelPath)
	for _, tc := range []struct{ name, dir, source string }{
		{"flag", stateDir, "flag"},
		{"config", stateDir, "config"},
		{"git-toplevel", stateDir, "git-toplevel"},
		{"cwd", stateDir, "cwd"},
		{"git-main-worktree dir not ending with the state-dir path", filepath.Join(mainRoot, "state"), "git-main-worktree"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if root, ok := MainWorktreeRoot(tc.dir, tc.source); ok || root != "" {
				t.Errorf("MainWorktreeRoot(%q, %q) = %q, %t, want \"\", false", tc.dir, tc.source, root, ok)
			}
		})
	}

	t.Run("main worktree at the filesystem root", func(t *testing.T) {
		dir := filepath.Join(string(filepath.Separator), defaultOrgStateDirRelPath)
		if root, ok := MainWorktreeRoot(dir, "git-main-worktree"); !ok || root != string(filepath.Separator) {
			t.Errorf("MainWorktreeRoot(%q) = %q, %t, want %q, true", dir, root, ok, string(filepath.Separator))
		}
	})
}

// TestLedgerMainWorktreeRoot covers the accessor `ralph org spawn` uses for
// the org-wide limits: a --state-dir or personal-file [org].state_dir
// (source "config", AC8 of plan 2026-10-30-org-state-dir-local) naming the
// main worktree's ledger, as a feature org's leader passes it from its
// linked worktree, has the main root (also through a symlink to the
// ledger), the default ledger has it as MainWorktreeRoot does, and any
// other ledger, source (the retired "env" of RALPH_ORG_STATE_DIR included,
// D5), or a cwd outside git has none.
func TestLedgerMainWorktreeRoot(t *testing.T) {
	mainRoot, sibling, nested := newRepoWithLinkedWorktrees(t)
	stateDir := filepath.Join(mainRoot, defaultOrgStateDirRelPath)
	mkdirAll(t, stateDir)
	link := filepath.Join(t.TempDir(), "ledger-link")
	if err := os.Symlink(stateDir, link); err != nil {
		t.Fatal(err)
	}
	for _, cwd := range []string{sibling, filepath.Join(nested, "pkg"), mainRoot} {
		mkdirAll(t, cwd)
		chdir(t, cwd)
		for _, tc := range []struct{ name, dir, source string }{
			{"flag", stateDir, "flag"},
			{"config", stateDir, "config"},
			{"flag through a symlink", link, "flag"},
			{"config through a symlink", link, "config"},
			{"default ledger", stateDir, "git-main-worktree"},
		} {
			if root, ok := LedgerMainWorktreeRoot(tc.dir, tc.source); !ok || root != mainRoot {
				t.Errorf("from %s, %s: LedgerMainWorktreeRoot(%q, %q) = %q, %t, want %q, true", cwd, tc.name, tc.dir, tc.source, root, ok, mainRoot)
			}
		}
		for _, tc := range []struct{ name, dir, source string }{
			{"flag naming the worktree's own ledger", filepath.Join(sibling, defaultOrgStateDirRelPath), "flag"},
			{"the retired env source naming the main worktree's ledger", stateDir, "env"},
			{"config naming another directory", filepath.Join(t.TempDir(), "org"), "config"},
			{"flag naming the main root", mainRoot, "flag"},
			{"git-toplevel", stateDir, "git-toplevel"},
			{"cwd", stateDir, "cwd"},
		} {
			if root, ok := LedgerMainWorktreeRoot(tc.dir, tc.source); ok || root != "" {
				t.Errorf("from %s, %s: LedgerMainWorktreeRoot(%q, %q) = %q, %t, want \"\", false", cwd, tc.name, tc.dir, tc.source, root, ok)
			}
		}
	}

	chdir(t, resolved(t, t.TempDir()))
	if root, ok := LedgerMainWorktreeRoot(stateDir, "flag"); ok || root != "" {
		t.Errorf("outside git: LedgerMainWorktreeRoot(%q, flag) = %q, %t, want \"\", false", stateDir, root, ok)
	}
}

// TestFeatureRepoRoot covers the main worktree root `ralph org start --plan`
// makes the feature worktree from: the default ledger's root, and for a
// ledger chosen with --state-dir or the personal file's [org].state_dir
// (source "config", AC8 of plan 2026-10-30-org-state-dir-local), wherever it
// is, the main worktree of the repository the command runs in. Outside git
// it is an error.
func TestFeatureRepoRoot(t *testing.T) {
	mainRoot, sibling, _ := newRepoWithLinkedWorktrees(t)
	elsewhere := filepath.Join(t.TempDir(), "ledger")
	chdir(t, sibling)
	for _, tc := range []struct{ name, dir, source string }{
		{"default ledger", filepath.Join(mainRoot, defaultOrgStateDirRelPath), "git-main-worktree"},
		{"flag", elsewhere, "flag"},
		{"config", elsewhere, "config"},
	} {
		if root, err := FeatureRepoRoot(tc.dir, tc.source); err != nil || root != mainRoot {
			t.Errorf("%s: FeatureRepoRoot(%q, %q) = %q, %v; want %q", tc.name, tc.dir, tc.source, root, err, mainRoot)
		}
	}

	chdir(t, resolved(t, t.TempDir()))
	if root, err := FeatureRepoRoot(elsewhere, "config"); err == nil || root != "" {
		t.Errorf("outside git: FeatureRepoRoot(%q, config) = %q, %v; want an error", elsewhere, root, err)
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

			got, source := resolveOrgStateDir(t, "", false)
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

		got, source := resolveOrgStateDir(t, "", false)
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

		got, source := resolveOrgStateDir(t, "", false)
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

	got, source := resolveOrgStateDir(t, "", false)
	want := filepath.Join(top, defaultOrgStateDirRelPath)
	if got != want {
		t.Errorf("dir = %q, want %q", got, want)
	}
	if source != "git-toplevel" {
		t.Errorf("source = %q, want %q", source, "git-toplevel")
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

		dir, source := resolveOrgStateDir(t, "", false)
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

		dir, source := resolveOrgStateDir(t, "", false)
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

		dir, source := resolveOrgStateDir(t, "", false)
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

		dir, source := resolveOrgStateDir(t, "", false)
		gotDir, active, err := LegacyWorktreeStateDir(dir, source)
		if err != nil || gotDir != "" || active != 0 {
			t.Errorf("LegacyWorktreeStateDir = (%q, %d, %v), want (\"\", 0, nil)", gotDir, active, err)
		}
	})

	t.Run("flag or config choice is never second-guessed, and the retired env hides nothing", func(t *testing.T) {
		mainRoot, sibling, _ := newRepoWithLinkedWorktrees(t)
		appendLedgerEvents(t, filepath.Join(sibling, defaultOrgStateDirRelPath),
			ManifestEvent{OrgID: "org-a", SeatID: "leader", Event: EventSpawned},
		)
		chdir(t, sibling)

		flagDir, flagSource := resolveOrgStateDir(t, filepath.Join(t.TempDir(), "state"), true)
		if gotDir, active, err := LegacyWorktreeStateDir(flagDir, flagSource); err != nil || gotDir != "" || active != 0 {
			t.Errorf("source %q: LegacyWorktreeStateDir = (%q, %d, %v), want (\"\", 0, nil)", flagSource, gotDir, active, err)
		}

		writeLocalConfig(t, mainRoot, "[org]\nstate_dir = \""+filepath.Join(t.TempDir(), "state")+"\"\n")
		configDir, configSource := resolveOrgStateDir(t, "", false)
		if gotDir, active, err := LegacyWorktreeStateDir(configDir, configSource); configSource != "config" || err != nil || gotDir != "" || active != 0 {
			t.Errorf("source %q: LegacyWorktreeStateDir = (%q, %d, %v), want source config and (\"\", 0, nil)", configSource, gotDir, active, err)
		}

		// RALPH_ORG_STATE_DIR was a deliberate choice too, until ralph
		// stopped reading it (plan 2026-10-30-org-state-dir-local, D5): set
		// now, with no state_dir in the personal file, it leaves the default
		// ledger in place, and the linked worktree's older ledger is found.
		writeLocalConfig(t, mainRoot, "")
		t.Setenv(retiredOrgStateDirEnv, filepath.Join(t.TempDir(), "state"))
		envDir, envSource := resolveOrgStateDir(t, "", false)
		wantLegacy := filepath.Join(sibling, defaultOrgStateDirRelPath)
		if gotDir, active, err := LegacyWorktreeStateDir(envDir, envSource); envSource != "git-main-worktree" || err != nil || gotDir != wantLegacy || active != 1 {
			t.Errorf("source %q: LegacyWorktreeStateDir = (%q, %d, %v), want source git-main-worktree and (%q, 1, nil)", envSource, gotDir, active, err, wantLegacy)
		}
	})

	t.Run("unreadable ledger is an error", func(t *testing.T) {
		_, sibling, _ := newRepoWithLinkedWorktrees(t)
		// A directory where manifest.jsonl should be: stat succeeds, the
		// read fails.
		mkdirAll(t, ManifestPathIn(filepath.Join(sibling, defaultOrgStateDirRelPath)))
		chdir(t, sibling)

		dir, source := resolveOrgStateDir(t, "", false)
		if _, _, err := LegacyWorktreeStateDir(dir, source); err == nil {
			t.Error("LegacyWorktreeStateDir: want an error for an unreadable legacy manifest, got nil")
		}
	})
}
