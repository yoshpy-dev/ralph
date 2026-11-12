package cli

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Tests for codex_hook_trust.go (plan 2026-11-12-codex-hook-trust-check,
// issue #310). None of them reads the real ~/.codex: each passes its own
// config path. None runs in parallel (newLegacyLedgerRepo chdirs and sets
// environment variables).

// codexHookTrustTempDir is t.TempDir with its symlinks resolved (macOS keeps
// temp dirs under /var, a symlink to /private/var), so a fixture's lexical
// and resolved roots agree unless the test adds a symlink of its own.
func codexHookTrustTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	return dir
}

// writeCodexHookTrustFile writes content to path, creating its directories.
func writeCodexHookTrustFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// mkdirCodexHookTrust creates path and its parents.
func mkdirCodexHookTrust(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

// makeCodexHookTrustCheckout makes dir look like a main checkout to codex: a
// .git directory holding HEAD. No git command runs.
func makeCodexHookTrustCheckout(t *testing.T, dir string) {
	t.Helper()
	writeCodexHookTrustFile(t, filepath.Join(dir, ".git", "HEAD"), "ref: refs/heads/main\n")
}

// symlinkCodexHookTrust makes link point at target, skipping the test where
// symlinks are not available.
func symlinkCodexHookTrust(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

// chdirCodexHookTrustAs makes dir the working directory with $PWD spelled as
// dir, as a shell that ran cd dir leaves it. os.Getwd then returns dir even
// when it goes through a symlink, so a relative path such as doctor's "." is
// made absolute in that spelling, not the real path. It fails the test when
// os.Getwd does not return dir.
func chdirCodexHookTrustAs(t *testing.T, dir string) {
	t.Helper()
	t.Chdir(dir)
	t.Setenv("PWD", dir)
	if wd, err := os.Getwd(); err != nil || wd != dir {
		t.Fatalf("os.Getwd() = %q, %v; want the spelling %s", wd, err, dir)
	}
}

// codexHookTrustStateTable renders one [hooks.state."<key>"] table.
func codexHookTrustStateTable(key, body string) string {
	return "[hooks.state." + strconv.Quote(key) + "]\n" + body + "\n"
}

// codexHookTrustTails lists the Tail of each item.
func codexHookTrustTails(items []codexHookTrustItem) []string {
	tails := []string{}
	for _, item := range items {
		tails = append(tails, item.Tail)
	}
	return tails
}

// codexHookTrustApproved lists the Tail of every slot of a classified report
// that is in neither NotApproved nor Off, the hooks the doctor check counts
// as approved; nil when the report is not classified.
func codexHookTrustApproved(report codexHookTrustReport) []string {
	if !report.Classified() {
		return nil
	}
	tails := []string{}
	for i, layer := range report.Layers {
		for _, slot := range layer.Slots {
			item := codexHookTrustItem{Layer: i, Tail: slot.Tail()}
			if !slices.Contains(report.NotApproved, item) && !slices.Contains(report.Off, item) {
				tails = append(tails, item.Tail)
			}
		}
	}
	return tails
}

// codexHookTrustCheckoutOf is codexHookCheckoutOf(dir), failing the test on
// an error.
func codexHookTrustCheckoutOf(t *testing.T, dir string) codexHookCheckout {
	t.Helper()
	c, err := codexHookCheckoutOf(dir)
	if err != nil {
		t.Fatalf("codexHookCheckoutOf(%s): %v", dir, err)
	}
	return c
}

// codexHookTrustRepos lists the distinct Repo of codexHookSpellings(dir), in
// order: the roots codex may key dir's hooks under.
func codexHookTrustRepos(t *testing.T, dir string) []string {
	t.Helper()
	spellings, err := codexHookSpellings(dir)
	if err != nil {
		t.Fatalf("codexHookSpellings(%s): %v", dir, err)
	}
	var repos []string
	for _, s := range spellings {
		if !slices.Contains(repos, s.Repo) {
			repos = append(repos, s.Repo)
		}
	}
	return repos
}

// registerCodexHookTrustWorktree writes what git writes for a linked
// worktree at checkout of the repository whose common directory is common,
// without running git: <common>/worktrees/<name> with a gitdir file naming
// <checkout>/.git and a commondir file of "../..", and the checkout's .git
// file with the gitdir line gitdirLine. It returns the registration.
func registerCodexHookTrustWorktree(t *testing.T, common, name, checkout, gitdirLine string) string {
	t.Helper()
	registration := filepath.Join(common, "worktrees", name)
	writeCodexHookTrustFile(t, filepath.Join(registration, "gitdir"), filepath.Join(checkout, ".git")+"\n")
	writeCodexHookTrustFile(t, filepath.Join(registration, "commondir"), "../..\n")
	writeCodexHookTrustFile(t, filepath.Join(checkout, ".git"), "gitdir: "+gitdirLine+"\n")
	return registration
}

func TestCodexHookTrust_RootFollowsCodexsWalk(t *testing.T) {
	tmp := codexHookTrustTempDir(t)
	mainRoot := filepath.Join(tmp, "main")
	makeCodexHookTrustCheckout(t, mainRoot)
	mkdirCodexHookTrust(t, filepath.Join(mainRoot, "a", "b"))
	writeCodexHookTrustFile(t, filepath.Join(mainRoot, "a", "file.txt"), "x")
	// A .git directory without HEAD is passed over.
	mkdirCodexHookTrust(t, filepath.Join(mainRoot, "nested", ".git"))
	mkdirCodexHookTrust(t, filepath.Join(mainRoot, "nested", "x"))
	// A .git file whose gitdir is not under worktrees (a submodule's).
	submodule := filepath.Join(tmp, "submodule")
	mkdirCodexHookTrust(t, filepath.Join(mainRoot, ".git", "modules", "sub"))
	writeCodexHookTrustFile(t, filepath.Join(submodule, ".git"), "gitdir: "+filepath.Join(tmp, "main", ".git", "modules", "sub")+"\n")
	// A .git file pointing into worktrees of a main checkout that has no .git.
	orphan := filepath.Join(tmp, "orphan")
	writeCodexHookTrustFile(t, filepath.Join(orphan, ".git"), "gitdir: "+filepath.Join(tmp, "gone", ".git", "worktrees", "orphan")+"\n")
	// A relative gitdir is taken from the checkout.
	relative := filepath.Join(tmp, "relative")
	registerCodexHookTrustWorktree(t, filepath.Join(mainRoot, ".git"), "relative", relative, "../main/.git/worktrees/relative")
	mkdirCodexHookTrust(t, filepath.Join(relative, "c"))
	// A .git file that is not a gitdir line.
	garbled := filepath.Join(tmp, "garbled")
	writeCodexHookTrustFile(t, filepath.Join(garbled, ".git"), "not a gitdir line\n")
	noGit := filepath.Join(tmp, "nogit", "sub")
	mkdirCodexHookTrust(t, noGit)

	cases := []struct {
		name, dir, checkout, repo string
	}{
		{"main checkout", mainRoot, mainRoot, mainRoot},
		{"subdirectory of the main checkout", filepath.Join(mainRoot, "a", "b"), mainRoot, mainRoot},
		{"file in the main checkout", filepath.Join(mainRoot, "a", "file.txt"), mainRoot, mainRoot},
		{"directory that does not exist", filepath.Join(mainRoot, "a", "nope"), mainRoot, mainRoot},
		{".git directory without HEAD", filepath.Join(mainRoot, "nested", "x"), mainRoot, mainRoot},
		{".git file not under worktrees", submodule, submodule, submodule},
		{".git file whose main has no .git", orphan, orphan, orphan},
		{".git file with a relative gitdir", relative, relative, mainRoot},
		{"subdirectory of a linked worktree", filepath.Join(relative, "c"), relative, mainRoot},
		{".git file without a gitdir line", garbled, garbled, garbled},
		{"no git at all", noGit, noGit, noGit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := codexHookTrustCheckoutOf(t, tc.dir)
			if got.Checkout != tc.checkout || got.Repo != tc.repo {
				t.Errorf("codexHookCheckoutOf(%s) = checkout %s, repo %s; want %s, %s", tc.dir, got.Checkout, got.Repo, tc.checkout, tc.repo)
			}
		})
	}
}

func TestCodexHookTrust_RootOfARelativeDir(t *testing.T) {
	tmp := codexHookTrustTempDir(t)
	mainRoot := filepath.Join(tmp, "main")
	makeCodexHookTrustCheckout(t, mainRoot)
	mkdirCodexHookTrust(t, filepath.Join(mainRoot, "a", "b"))
	t.Chdir(filepath.Join(mainRoot, "a"))
	got := codexHookTrustCheckoutOf(t, "b")
	want := codexHookCheckout{Base: filepath.Join(mainRoot, "a", "b"), Checkout: mainRoot, Repo: mainRoot}
	if got != want {
		t.Errorf("codexHookCheckoutOf(b) = %+v, want %+v", got, want)
	}
}

// codexHookTrustLinked is a linked worktree written without git
// (registerCodexHookTrustWorktree): checkout at <tmp>/wt, registered as wt
// in common, whose main checkout codex takes to be main.
type codexHookTrustLinked struct {
	tmp, main, common, checkout, registration string
}

// newCodexHookTrustLinked writes a repository whose common directory is
// <tmp>/<commonRel> (a directory holding HEAD) and a linked worktree of it at
// <tmp>/wt whose .git file names <common>/worktrees/wt. Its main checkout is
// the common directory's parent; nothing more is written there.
func newCodexHookTrustLinked(t *testing.T, commonRel string) codexHookTrustLinked {
	t.Helper()
	tmp := codexHookTrustTempDir(t)
	f := codexHookTrustLinked{tmp: tmp, common: filepath.Join(tmp, commonRel), checkout: filepath.Join(tmp, "wt")}
	f.main = filepath.Dir(f.common)
	writeCodexHookTrustFile(t, filepath.Join(f.common, "HEAD"), "ref: refs/heads/main\n")
	f.registration = registerCodexHookTrustWorktree(t, f.common, "wt", f.checkout, filepath.Join(f.common, "worktrees", "wt"))
	return f
}

// codex takes a linked worktree's main checkout only when its registration
// checks out (resolve_root in codex-rs/git-utils/src/trust.rs at
// rust-v0.166.0; plan D7 (c), AC12): otherwise the worktree is its own root.
func TestCodexHookTrust_RootChecksTheLinkedWorktreesOwnership(t *testing.T) {
	cases := []struct {
		name      string
		commonRel string                                     // where the common directory is, under the temp dir
		change    func(t *testing.T, f codexHookTrustLinked) // what the case changes; nil for none
		takesMain bool
	}{
		{"registered as git registers it", "main/.git", nil, true},
		{"a relative gitdir file in the registration", "main/.git", func(t *testing.T, f codexHookTrustLinked) {
			writeCodexHookTrustFile(t, filepath.Join(f.registration, "gitdir"), "../../../../wt/.git\n")
		}, true},
		{"the gitdir a symlink", "main/.git", func(t *testing.T, f codexHookTrustLinked) {
			moved := filepath.Join(f.common, "worktrees", "wt-real")
			if err := os.Rename(f.registration, moved); err != nil {
				t.Fatal(err)
			}
			symlinkCodexHookTrust(t, moved, f.registration)
		}, false},
		{"no gitdir file in the registration", "main/.git", func(t *testing.T, f codexHookTrustLinked) {
			if err := os.Remove(filepath.Join(f.registration, "gitdir")); err != nil {
				t.Fatal(err)
			}
		}, false},
		{"an empty gitdir file", "main/.git", func(t *testing.T, f codexHookTrustLinked) {
			writeCodexHookTrustFile(t, filepath.Join(f.registration, "gitdir"), " \n")
		}, false},
		{"a gitdir file naming another checkout", "main/.git", func(t *testing.T, f codexHookTrustLinked) {
			other := filepath.Join(f.tmp, "other")
			mkdirCodexHookTrust(t, other)
			writeCodexHookTrustFile(t, filepath.Join(f.registration, "gitdir"), filepath.Join(other, ".git")+"\n")
		}, false},
		{"a gitdir file naming no .git", "main/.git", func(t *testing.T, f codexHookTrustLinked) {
			writeCodexHookTrustFile(t, filepath.Join(f.registration, "gitdir"), filepath.Join(f.checkout, "dotgit")+"\n")
		}, false},
		{"no commondir file", "main/.git", func(t *testing.T, f codexHookTrustLinked) {
			if err := os.Remove(filepath.Join(f.registration, "commondir")); err != nil {
				t.Fatal(err)
			}
		}, false},
		{"a commondir file naming another directory", "main/.git", func(t *testing.T, f codexHookTrustLinked) {
			writeCodexHookTrustFile(t, filepath.Join(f.registration, "commondir"), f.tmp+"\n")
		}, false},
		{"a bare repository whose parent has no .git", "outer/cache.git", nil, false},
		{"a bare repository whose parent's .git is another repository", "outer/cache.git", func(t *testing.T, f codexHookTrustLinked) {
			makeCodexHookTrustCheckout(t, f.main)
		}, false},
		{"a main checkout whose .git file names the common directory", "outer/cache.git", func(t *testing.T, f codexHookTrustLinked) {
			writeCodexHookTrustFile(t, filepath.Join(f.main, ".git"), "gitdir: cache.git\n")
		}, true},
		{"a main checkout whose .git file names another directory", "outer/cache.git", func(t *testing.T, f codexHookTrustLinked) {
			mkdirCodexHookTrust(t, filepath.Join(f.main, "elsewhere.git"))
			writeCodexHookTrustFile(t, filepath.Join(f.main, ".git"), "gitdir: elsewhere.git\n")
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCodexHookTrustLinked(t, tc.commonRel)
			if tc.change != nil {
				tc.change(t, f)
			}
			want := f.checkout
			if tc.takesMain {
				want = f.main
			}
			got := codexHookTrustCheckoutOf(t, f.checkout)
			if got.Checkout != f.checkout || got.Repo != want {
				t.Errorf("codexHookCheckoutOf(%s) = checkout %s, repo %s; want %s, %s", f.checkout, got.Checkout, got.Repo, f.checkout, want)
			}
		})
	}
}

func TestCodexHookTrust_RootOfALinkedWorktreeMadeByGit(t *testing.T) {
	r := newLegacyLedgerRepo(t)
	sub := filepath.Join(r.worktree, "sub")
	mkdirCodexHookTrust(t, sub)
	for _, dir := range []string{r.worktree, sub} {
		got := codexHookTrustCheckoutOf(t, dir)
		if got.Checkout != r.worktree || got.Repo != r.mainRoot {
			t.Errorf("codexHookCheckoutOf(%s) = checkout %s, repo %s; want %s, the main checkout %s", dir, got.Checkout, got.Repo, r.worktree, r.mainRoot)
		}
	}
	// Through a symlink the worktree still keys under the real main
	// checkout: its .git file spells the gitdir with the real path (evidence
	// section 2).
	link := filepath.Join(codexHookTrustTempDir(t), "link")
	symlinkCodexHookTrust(t, filepath.Dir(r.worktree), link)
	if repos := codexHookTrustRepos(t, filepath.Join(link, filepath.Base(r.worktree))); !slices.Equal(repos, []string{r.mainRoot}) {
		t.Errorf("roots through the symlink = %v, want [%s]", repos, r.mainRoot)
	}
}

// A linked worktree of a bare repository has no main checkout. Its gitdir is
// <common>/worktrees/<name>, and, as in codex's resolve_root, the parent of
// <common> is the root only when its .git is <common>: a bare repository
// named store.git leaves the worktree as its own root, one at <proj>/.git
// makes proj the root, and one at <outer>/cache.git leaves the worktree as
// its own root although <outer>/.git is another repository (AC12).
func TestCodexHookTrust_RootOfABareRepositorysLinkedWorktree(t *testing.T) {
	isolateFeatureGitEnv(t)
	tmp := codexHookTrustTempDir(t)
	src := filepath.Join(tmp, "src")
	runLedgerTestGit(t, "init", "--quiet", src)
	runLedgerTestGit(t, "-C", src, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
		"commit", "--quiet", "--allow-empty", "-m", "init")

	// <tmp>/.git does not exist, so the worktree of store.git is its own root.
	store := filepath.Join(tmp, "store.git")
	runLedgerTestGit(t, "clone", "--quiet", "--bare", src, store)
	wtA := filepath.Join(tmp, "wt-a")
	runLedgerTestGit(t, "-C", store, "worktree", "add", "--quiet", wtA, "-b", "a")
	if got := codexHookTrustCheckoutOf(t, wtA); got.Repo != wtA {
		t.Errorf("codexHookCheckoutOf(%s).Repo = %s, want the worktree itself", wtA, got.Repo)
	}

	// The bare repository is proj's .git, so proj is the root.
	proj := filepath.Join(tmp, "proj")
	runLedgerTestGit(t, "clone", "--quiet", "--bare", src, filepath.Join(proj, ".git"))
	wtB := filepath.Join(tmp, "wt-b")
	runLedgerTestGit(t, "-C", filepath.Join(proj, ".git"), "worktree", "add", "--quiet", wtB, "-b", "b")
	if got := codexHookTrustCheckoutOf(t, wtB); got.Repo != proj {
		t.Errorf("codexHookCheckoutOf(%s).Repo = %s, want %s", wtB, got.Repo, proj)
	}
	if repos := codexHookTrustRepos(t, wtB); !slices.Equal(repos, []string{proj}) {
		t.Errorf("roots of %s = %v, want [%s]", wtB, repos, proj)
	}

	// The bare repository is outer/cache.git while outer/.git is another
	// repository: outer does not own cache.git, so the worktree reads and
	// keys its own hooks, not outer's.
	outer := filepath.Join(tmp, "outer")
	runLedgerTestGit(t, "init", "--quiet", outer)
	cache := filepath.Join(outer, "cache.git")
	runLedgerTestGit(t, "clone", "--quiet", "--bare", src, cache)
	wtC := filepath.Join(tmp, "wt-c")
	runLedgerTestGit(t, "-C", cache, "worktree", "add", "--quiet", wtC, "-b", "c")
	if got := codexHookTrustCheckoutOf(t, wtC); got.Checkout != wtC || got.Repo != wtC {
		t.Errorf("codexHookCheckoutOf(%s) = checkout %s, repo %s; want the worktree itself for both", wtC, got.Checkout, got.Repo)
	}
	writeCodexHookTrustFile(t, filepath.Join(outer, ".codex", "hooks.json"), codexHookTrustFixtureHooks)
	wtHooks := filepath.Join(wtC, ".codex", "hooks.json")
	writeCodexHookTrustFile(t, wtHooks, `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "stop.sh"}]}]}}`)
	report := codexHookTrustFor(wtC, codexSandboxEnv{ConfigPath: filepath.Join(tmp, "no-config.toml")})
	if len(report.Layers) != 1 || !slices.Equal(report.Layers[0].Dirs, []string{wtC}) || report.Layers[0].HooksPath() != wtHooks {
		t.Fatalf("layers = %+v, want only the worktree's own %s", report.Layers, wtHooks)
	}
	if len(report.NotApproved) != 1 || report.NotApproved[0] != (codexHookTrustItem{Layer: 0, Tail: "stop:0:0"}) {
		t.Errorf("not approved = %+v, want stop:0:0 keyed under %s", report.NotApproved, wtHooks)
	}
}

func TestCodexHookTrust_RootCandidatesOfASymlinkedMainCheckout(t *testing.T) {
	tmp := codexHookTrustTempDir(t)
	realMain := filepath.Join(tmp, "real", "main")
	makeCodexHookTrustCheckout(t, realMain)
	mkdirCodexHookTrust(t, filepath.Join(realMain, "sub"))
	symlinkCodexHookTrust(t, filepath.Join(tmp, "real"), filepath.Join(tmp, "link"))
	linkMain := filepath.Join(tmp, "link", "main")

	cases := []struct {
		name string
		dir  string
		want []string
	}{
		{"through the symlink", filepath.Join(linkMain, "sub"), []string{linkMain, realMain}},
		{"by the real path", realMain, []string{realMain}},
		{"a path that does not exist", filepath.Join(linkMain, "nope"), []string{linkMain}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if roots := codexHookTrustRepos(t, tc.dir); !slices.Equal(roots, tc.want) {
				t.Errorf("roots of %s = %v, want %v", tc.dir, roots, tc.want)
			}
		})
	}
}

// doctor passes "." (relative), and a shell opened through a symlink keeps
// the symlink's spelling in $PWD. The real root must still come back second:
// a seat started under the real path keys its hooks there (AC9).
func TestCodexHookTrust_RootCandidatesOfARelativeDirInASymlinkedMainCheckout(t *testing.T) {
	tmp := codexHookTrustTempDir(t)
	realMain := filepath.Join(tmp, "real", "main")
	makeCodexHookTrustCheckout(t, realMain)
	mkdirCodexHookTrust(t, filepath.Join(realMain, "sub"))
	symlinkCodexHookTrust(t, filepath.Join(tmp, "real"), filepath.Join(tmp, "link"))
	linkMain := filepath.Join(tmp, "link", "main")

	cases := []struct {
		name string
		cwd  string // the working directory, with $PWD spelled the same way
		dir  string
		want []string
	}{
		{"dot through the symlink", linkMain, ".", []string{linkMain, realMain}},
		{"subdirectory through the symlink", linkMain, "sub", []string{linkMain, realMain}},
		{"path that does not exist through the symlink", linkMain, "nope", []string{linkMain}},
		{"dot by the real path", realMain, ".", []string{realMain}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			chdirCodexHookTrustAs(t, tc.cwd)
			if roots := codexHookTrustRepos(t, tc.dir); !slices.Equal(roots, tc.want) {
				t.Errorf("roots of %s in %s = %v, want %v", tc.dir, tc.cwd, roots, tc.want)
			}
		})
	}
}

func TestCodexHookTrust_ReadStatesOfAMissingConfig(t *testing.T) {
	states, state, err := readCodexHookStates(filepath.Join(t.TempDir(), "no-such", "config.toml"))
	if state != codexHookSourceMissing || err != nil || states != nil {
		t.Errorf("got states=%v state=%v err=%v, want nil, missing, nil", states, state, err)
	}
}

func TestCodexHookTrust_ReadStatesOfAnUnreadableConfig(t *testing.T) {
	dir := t.TempDir()
	asDir := filepath.Join(dir, "config-dir.toml")
	mkdirCodexHookTrust(t, asDir)
	oversized := filepath.Join(dir, "big.toml")
	writeCodexHookTrustFile(t, oversized, strings.Repeat("#", codexConfigMaxBytes+1))

	cases := []struct {
		name, path, want string
	}{
		{"a directory", asDir, "not a regular file"},
		{"larger than 1 MiB", oversized, "larger than 1 MiB"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			states, state, err := readCodexHookStates(tc.path)
			if state != codexHookSourceUnreadable || err == nil || err.Error() != tc.want || states != nil {
				t.Errorf("got states=%v state=%v err=%v, want nil, unreadable, %q", states, state, err, tc.want)
			}
		})
	}
}

func TestCodexHookTrust_ReadStatesOfAnUndecodableConfigCarriesNoText(t *testing.T) {
	const marker = "config-content-marker-7f3a"
	cases := []struct {
		name, content string
	}{
		{"invalid TOML", "[hooks.state.\"/r/.codex/hooks.json:stop:0:0\"]\ntrusted_hash = \"" + marker + "\"\n" + marker + " = = 1\n"},
		{"a table defined twice", codexHookTrustStateTable("/"+marker, "trusted_hash = \"x\"") + codexHookTrustStateTable("/"+marker, "enabled = true")},
		{"trusted_hash of the wrong type", codexHookTrustStateTable("/"+marker, "trusted_hash = 5")},
		{"hooks that is not a table", "hooks = \"" + marker + "\"\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			writeCodexHookTrustFile(t, path, tc.content)
			states, state, err := readCodexHookStates(path)
			if state != codexHookSourceInvalid || states != nil {
				t.Fatalf("got states=%v state=%v err=%v, want nil, invalid", states, state, err)
			}
			var decodeErr *codexConfigDecodeError
			if !errors.As(err, &decodeErr) {
				t.Fatalf("err = %T %v, want a *codexConfigDecodeError", err, err)
			}
			if strings.Contains(err.Error(), marker) {
				t.Errorf("error text %q repeats the config's content", err)
			}
		})
	}
}

func TestCodexHookTrust_ReadStatesKeepsOnlyHookStates(t *testing.T) {
	content := `model = "gpt-x"
sandbox_mode = "workspace-write"

[projects."/r"]
trust_level = "trusted"

[[hooks.PreToolUse]]
matcher = "Bash"
[[hooks.PreToolUse.hooks]]
type = "command"
command = "echo toml"

` + codexHookTrustStateTable("/r/.codex/hooks.json:pre_tool_use:0:0", `trusted_hash = "sha256:aaaa"`) +
		codexHookTrustStateTable("/r/.codex/hooks.json:session_start:0:0", "enabled = false\ntrusted_hash = \"sha256:bbbb\"") +
		codexHookTrustStateTable("/r/.codex/hooks.json:stop:0:0", "enabled = true") +
		codexHookTrustStateTable("/r/.codex/hooks.json:post_tool_use:0:0", `trusted_content_hash = "sha256:cccc"`)
	path := filepath.Join(t.TempDir(), "config.toml")
	writeCodexHookTrustFile(t, path, content)

	states, state, err := readCodexHookStates(path)
	if state != codexHookSourceRead || err != nil {
		t.Fatalf("state=%v err=%v, want read, nil", state, err)
	}
	off, on := false, true
	want := map[string]codexHookStateEntry{
		"/r/.codex/hooks.json:pre_tool_use:0:0":  {TrustedHash: "sha256:aaaa"},
		"/r/.codex/hooks.json:session_start:0:0": {TrustedHash: "sha256:bbbb", Enabled: &off},
		"/r/.codex/hooks.json:stop:0:0":          {Enabled: &on},
		"/r/.codex/hooks.json:post_tool_use:0:0": {},
	}
	if len(states) != len(want) {
		t.Fatalf("states = %v, want %d entries", states, len(want))
	}
	for key, w := range want {
		got, ok := states[key]
		if !ok {
			t.Errorf("no state for %s", key)
			continue
		}
		if got.TrustedHash != w.TrustedHash || (got.Enabled == nil) != (w.Enabled == nil) || (got.Enabled != nil && *got.Enabled != *w.Enabled) {
			t.Errorf("state of %s = %+v, want %+v", key, got, w)
		}
	}

	// A config with no [hooks] at all reads as no states.
	writeCodexHookTrustFile(t, path, "model = \"gpt-x\"\n")
	states, state, err = readCodexHookStates(path)
	if state != codexHookSourceRead || err != nil || len(states) != 0 {
		t.Errorf("no [hooks]: states=%v state=%v err=%v, want none, read, nil", states, state, err)
	}
}

func TestCodexHookTrust_ClassifyUnderOneSpelling(t *testing.T) {
	slot := codexHookSlot{Label: "pre_tool_use", Group: 0, Handler: 0}
	key := codexHookKey("/r", slot)
	off, on := false, true
	cases := []struct {
		name   string
		states map[string]codexHookStateEntry
		want   codexHookTrust
	}{
		{"approved", map[string]codexHookStateEntry{key: {TrustedHash: "sha256:a"}}, codexHookApproved},
		{"approved and enabled", map[string]codexHookStateEntry{key: {TrustedHash: "sha256:a", Enabled: &on}}, codexHookApproved},
		{"no config entries", nil, codexHookNotApproved},
		{"key missing", map[string]codexHookStateEntry{codexHookKey("/other", slot): {TrustedHash: "sha256:a"}}, codexHookNotApproved},
		{"no trusted_hash", map[string]codexHookStateEntry{key: {Enabled: &on}}, codexHookNotApproved},
		{"turned off with a hash", map[string]codexHookStateEntry{key: {TrustedHash: "sha256:a", Enabled: &off}}, codexHookOff},
		{"turned off without a hash", map[string]codexHookStateEntry{key: {Enabled: &off}}, codexHookOff},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyCodexHookSlot(tc.states, []string{"/r"}, slot); got != tc.want {
				t.Errorf("classifyCodexHookSlot = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCodexHookTrust_ClassifyNeedsEverySpelling(t *testing.T) {
	slot := codexHookSlot{Label: "session_start", Group: 0, Handler: 0}
	roots := []string{"/link/main", "/real/main"}
	linkKey, realKey := codexHookKey(roots[0], slot), codexHookKey(roots[1], slot)
	approved := codexHookStateEntry{TrustedHash: "sha256:a"}
	off := false
	turnedOff := codexHookStateEntry{TrustedHash: "sha256:a", Enabled: &off}
	cases := []struct {
		name   string
		states map[string]codexHookStateEntry
		want   codexHookTrust
	}{
		{"approved only under the real path", map[string]codexHookStateEntry{realKey: approved}, codexHookNotApproved},
		{"approved only under the symlink path", map[string]codexHookStateEntry{linkKey: approved}, codexHookNotApproved},
		{"approved under both", map[string]codexHookStateEntry{linkKey: approved, realKey: approved}, codexHookApproved},
		{"approved under one, turned off under the other", map[string]codexHookStateEntry{linkKey: approved, realKey: turnedOff}, codexHookOff},
		{"turned off under one, missing under the other", map[string]codexHookStateEntry{linkKey: turnedOff}, codexHookNotApproved},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyCodexHookSlot(tc.states, roots, slot); got != tc.want {
				t.Errorf("classifyCodexHookSlot = %v, want %v", got, tc.want)
			}
		})
	}
}

// codexHookTrustFixtureHooks is a hooks.json with four handlers codex asks
// approval for (pre_tool_use 0:0 and 0:1, post_tool_use 0:0, session_start
// 0:0) and a Stop handler with a blank command, which needs none.
const codexHookTrustFixtureHooks = `{
  "hooks": {
    "PreToolUse": [{"matcher": "Bash", "hooks": [
      {"type": "command", "command": "guard.sh"},
      {"type": "command", "command": "other.sh"}
    ]}],
    "PostToolUse": [{"hooks": [{"type": "command", "command": "post.sh"}]}],
    "SessionStart": [{"hooks": [{"type": "command", "command": "start.sh"}]}],
    "Stop": [{"hooks": [{"type": "command", "command": " "}]}]
  }
}`

func TestCodexHookTrustFor_SortsEveryHandlerOfTheRepo(t *testing.T) {
	tmp := codexHookTrustTempDir(t)
	repo := filepath.Join(tmp, "repo")
	makeCodexHookTrustCheckout(t, repo)
	mkdirCodexHookTrust(t, filepath.Join(repo, "sub"))
	hooksPath := filepath.Join(repo, ".codex", "hooks.json")
	writeCodexHookTrustFile(t, hooksPath, codexHookTrustFixtureHooks)
	configPath := filepath.Join(tmp, "codex-home", "config.toml")
	writeCodexHookTrustFile(t, configPath,
		codexHookTrustStateTable(hooksPath+":pre_tool_use:0:0", `trusted_hash = "sha256:a"`)+
			codexHookTrustStateTable(hooksPath+":pre_tool_use:0:1", "trusted_hash = \"sha256:b\"\nenabled = false")+
			codexHookTrustStateTable(hooksPath+":session_start:0:0", "enabled = true"))

	report := codexHookTrustFor(filepath.Join(repo, "sub"), codexSandboxEnv{ConfigPath: configPath})

	// sub has no .codex, so the root is the only layer.
	if len(report.Layers) != 1 || !slices.Equal(report.Layers[0].Dirs, []string{repo}) || report.Layers[0].HooksPath() != hooksPath || report.ConfigPath != configPath {
		t.Fatalf("layers=%+v config=%s, want the root layer [%s] %s and %s", report.Layers, report.ConfigPath, repo, hooksPath, configPath)
	}
	root := report.Layers[0]
	if root.State != codexHookSourceRead || root.Err != nil || report.ConfigState != codexHookSourceRead || report.ConfigErr != nil || !report.Classified() {
		t.Fatalf("hooks=%v/%v config=%v/%v classified=%v, want both read", root.State, root.Err, report.ConfigState, report.ConfigErr, report.Classified())
	}
	if len(root.Slots) != 4 || report.SlotCount() != 4 {
		t.Errorf("slots = %v (count %d), want 4", root.Slots, report.SlotCount())
	}
	if got := codexHookTrustApproved(report); !slices.Equal(got, []string{"pre_tool_use:0:0"}) {
		t.Errorf("approved = %v", got)
	}
	if got := codexHookTrustTails(report.NotApproved); !slices.Equal(got, []string{"post_tool_use:0:0", "session_start:0:0"}) {
		t.Errorf("not approved = %v", got)
	}
	if got := codexHookTrustTails(report.Off); !slices.Equal(got, []string{"pre_tool_use:0:1"}) {
		t.Errorf("off = %v", got)
	}
}

func TestCodexHookTrustFor_LinkedWorktreeReadsTheMainCheckoutsHooks(t *testing.T) {
	r := newLegacyLedgerRepo(t)
	mainHooks := filepath.Join(r.mainRoot, ".codex", "hooks.json")
	writeCodexHookTrustFile(t, mainHooks, `{"hooks": {"PreToolUse": [{"hooks": [{"type": "command", "command": "guard.sh"}]}]}}`)
	// The worktree's own hooks.json is not the one codex reads.
	writeCodexHookTrustFile(t, filepath.Join(r.worktree, ".codex", "hooks.json"), codexHookTrustFixtureHooks)
	configPath := filepath.Join(t.TempDir(), "config.toml")
	writeCodexHookTrustFile(t, configPath, codexHookTrustStateTable(mainHooks+":pre_tool_use:0:0", `trusted_hash = "sha256:a"`))

	report := codexHookTrustFor(r.worktree, codexSandboxEnv{ConfigPath: configPath})

	if len(report.Layers) != 1 || !slices.Equal(report.Layers[0].Dirs, []string{r.mainRoot}) || report.Layers[0].HooksPath() != mainHooks {
		t.Fatalf("layers=%+v, want the root layer [%s] %s", report.Layers, r.mainRoot, mainHooks)
	}
	if got := codexHookTrustApproved(report); !slices.Equal(got, []string{"pre_tool_use:0:0"}) || len(report.NotApproved) != 0 || len(report.Off) != 0 {
		t.Errorf("approved=%v not approved=%v off=%v, want only pre_tool_use:0:0 approved", got, report.NotApproved, report.Off)
	}
}

// codexHookTrustStopHooks is a hooks.json with one Stop handler, and
// codexHookTrustPreHooks one with one PreToolUse handler.
const (
	codexHookTrustStopHooks = `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "stop.sh"}]}]}}`
	codexHookTrustPreHooks  = `{"hooks": {"PreToolUse": [{"hooks": [{"type": "command", "command": "guard.sh"}]}]}}`
)

// codex reads the .codex/hooks.json of every directory from the checkout
// root down to the seat's directory that has a .codex directory, each keyed
// under its own path (plan D7 (a), AC10). A directory without .codex is no
// layer, a .codex without hooks.json is a silent one, and a .codex beside
// the path is not read.
func TestCodexHookTrustFor_ReadsEveryLayerDownToTheDirectory(t *testing.T) {
	tmp := codexHookTrustTempDir(t)
	repo := filepath.Join(tmp, "repo")
	makeCodexHookTrustCheckout(t, repo)
	rootHooks := filepath.Join(repo, ".codex", "hooks.json")
	writeCodexHookTrustFile(t, rootHooks, codexHookTrustStopHooks)
	sub := filepath.Join(repo, "sub")
	subHooks := filepath.Join(sub, ".codex", "hooks.json")
	writeCodexHookTrustFile(t, subHooks, codexHookTrustPreHooks)
	deeper := filepath.Join(sub, "mid", "deeper")
	mkdirCodexHookTrust(t, filepath.Join(deeper, ".codex"))
	writeCodexHookTrustFile(t, filepath.Join(repo, "beside", ".codex", "hooks.json"), codexHookTrustFixtureHooks)
	configPath := filepath.Join(tmp, "config.toml")
	writeCodexHookTrustFile(t, configPath, codexHookTrustStateTable(rootHooks+":stop:0:0", `trusted_hash = "sha256:a"`))

	report := codexHookTrustFor(deeper, codexSandboxEnv{ConfigPath: configPath})

	var rels []string
	for _, layer := range report.Layers {
		rels = append(rels, layer.Rel)
	}
	if want := []string{".", "sub", filepath.Join("sub", "mid", "deeper")}; !slices.Equal(rels, want) {
		t.Fatalf("layers %v, want %v", rels, want)
	}
	if got := report.Layers[1]; !slices.Equal(got.Dirs, []string{sub}) || got.HooksPath() != subHooks || got.State != codexHookSourceRead {
		t.Errorf("sub's layer = %+v, want read from %s", got, subHooks)
	}
	if got := report.Layers[2]; got.State != codexHookSourceMissing || len(report.HooksProblems()) != 0 || !report.Classified() {
		t.Errorf("deeper's layer = %+v, problems %v, classified %v; want a silent missing hooks.json", got, report.HooksProblems(), report.Classified())
	}
	if report.SlotCount() != 2 || !slices.Equal(report.SlotPaths(), []string{rootHooks, subHooks}) {
		t.Errorf("slot count %d, paths %v; want 2 in %s and %s", report.SlotCount(), report.SlotPaths(), rootHooks, subHooks)
	}
	if got := codexHookTrustApproved(report); !slices.Equal(got, []string{"stop:0:0"}) {
		t.Errorf("approved = %v, want the root's stop:0:0", got)
	}
	if len(report.NotApproved) != 1 || report.NotApproved[0] != (codexHookTrustItem{Layer: 1, Tail: "pre_tool_use:0:0"}) {
		t.Errorf("not approved = %+v, want sub's pre_tool_use:0:0 keyed under %s", report.NotApproved, subHooks)
	}
	if got := report.ApprovalDirs(); !slices.Equal(got, []string{sub}) {
		t.Errorf("approval dirs = %v, want [%s]", got, sub)
	}
	if got := report.ItemName(report.NotApproved[0]); got != filepath.Join("sub", ".codex", "hooks.json")+":pre_tool_use:0:0" {
		t.Errorf("item name = %q", got)
	}
}

// From a linked worktree a nested layer exists when its .codex is in the
// worktree, and codex reads and keys its hooks.json at the same place in the
// main checkout (root_checkout_hooks_folder_for_dir; AC10).
func TestCodexHookTrustFor_LinkedWorktreeReadsNestedLayersFromTheMainCheckout(t *testing.T) {
	r := newLegacyLedgerRepo(t)
	mainHooks := filepath.Join(r.mainRoot, ".codex", "hooks.json")
	mainSubHooks := filepath.Join(r.mainRoot, "sub", ".codex", "hooks.json")
	writeCodexHookTrustFile(t, mainHooks, codexHookTrustStopHooks)
	writeCodexHookTrustFile(t, mainSubHooks, codexHookTrustPreHooks)
	// The worktree's own files are not the ones codex reads.
	writeCodexHookTrustFile(t, filepath.Join(r.worktree, ".codex", "hooks.json"), codexHookTrustFixtureHooks)
	sub := filepath.Join(r.worktree, "sub")
	mkdirCodexHookTrust(t, sub)
	configPath := filepath.Join(t.TempDir(), "config.toml")
	env := codexSandboxEnv{ConfigPath: configPath}

	// Without sub/.codex in the worktree, codex has no layer there, although
	// the main checkout has one.
	report := codexHookTrustFor(sub, env)
	if len(report.Layers) != 1 || report.SlotCount() != 1 || !slices.Equal(report.SlotPaths(), []string{mainHooks}) {
		t.Errorf("without sub/.codex: layers %+v, slots %d in %v; want the root layer alone", report.Layers, report.SlotCount(), report.SlotPaths())
	}

	writeCodexHookTrustFile(t, filepath.Join(sub, ".codex", "hooks.json"), codexHookTrustFixtureHooks)
	report = codexHookTrustFor(sub, env)
	if len(report.Layers) != 2 || !slices.Equal(report.Layers[1].Dirs, []string{filepath.Join(r.mainRoot, "sub")}) || report.Layers[1].HooksPath() != mainSubHooks {
		t.Fatalf("layers = %+v, want the root and sub, read from %s", report.Layers, mainSubHooks)
	}
	if got := codexHookTrustTails(report.NotApproved); !slices.Equal(got, []string{"stop:0:0", "pre_tool_use:0:0"}) || report.NotApproved[1].Layer != 1 {
		t.Errorf("not approved = %+v, want the main checkout's stop:0:0 and sub's pre_tool_use:0:0", report.NotApproved)
	}
	if got := report.ApprovalDirs(); !slices.Equal(got, []string{filepath.Join(r.mainRoot, "sub")}) {
		t.Errorf("approval dirs = %v, want the main checkout's sub", got)
	}
}

// A nested hooks.json that cannot be read or is invalid is a problem, like
// the root's, and the report is then not classified (AC10, AC11).
func TestCodexHookTrustFor_NestedLayerProblems(t *testing.T) {
	for _, tc := range []struct{ name, hooks string }{{"invalid", `{"hooks": null}`}, {"a directory", "<dir>"}} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := codexHookTrustTempDir(t)
			repo := filepath.Join(tmp, "repo")
			makeCodexHookTrustCheckout(t, repo)
			writeCodexHookTrustFile(t, filepath.Join(repo, ".codex", "hooks.json"), codexHookTrustStopHooks)
			subHooks := filepath.Join(repo, "sub", ".codex", "hooks.json")
			if tc.hooks == "<dir>" {
				mkdirCodexHookTrust(t, subHooks)
			} else {
				writeCodexHookTrustFile(t, subHooks, tc.hooks)
			}

			report := codexHookTrustFor(filepath.Join(repo, "sub"), codexSandboxEnv{ConfigPath: filepath.Join(tmp, "config.toml")})

			problems := report.HooksProblems()
			if len(problems) != 1 || problems[0].HooksPath() != subHooks || problems[0].Err == nil || report.Classified() {
				t.Errorf("problems = %+v, classified %v; want sub's hooks.json alone, not classified", problems, report.Classified())
			}
		})
	}
}

func TestCodexHookTrustFor_SymlinkedMainCheckoutNeedsBothSpellings(t *testing.T) {
	tmp := codexHookTrustTempDir(t)
	realRepo := filepath.Join(tmp, "real", "repo")
	makeCodexHookTrustCheckout(t, realRepo)
	writeCodexHookTrustFile(t, filepath.Join(realRepo, ".codex", "hooks.json"), `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "stop.sh"}]}]}}`)
	symlinkCodexHookTrust(t, filepath.Join(tmp, "real"), filepath.Join(tmp, "link"))
	linkRepo := filepath.Join(tmp, "link", "repo")
	linkKey := filepath.Join(linkRepo, ".codex", "hooks.json") + ":stop:0:0"
	realKey := filepath.Join(realRepo, ".codex", "hooks.json") + ":stop:0:0"
	configPath := filepath.Join(tmp, "config.toml")
	approval := `trusted_hash = "sha256:a"`

	cases := []struct {
		name     string
		config   string
		approved bool
	}{
		{"approved under the real path only", codexHookTrustStateTable(realKey, approval), false},
		{"approved under the symlink path only", codexHookTrustStateTable(linkKey, approval), false},
		{"approved under both", codexHookTrustStateTable(linkKey, approval) + codexHookTrustStateTable(realKey, approval), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			writeCodexHookTrustFile(t, configPath, tc.config)
			report := codexHookTrustFor(linkRepo, codexSandboxEnv{ConfigPath: configPath})
			if len(report.Layers) != 1 || !slices.Equal(report.Layers[0].Dirs, []string{linkRepo, realRepo}) {
				t.Fatalf("layers = %+v, want the root layer under [%s %s]", report.Layers, linkRepo, realRepo)
			}
			if got := slices.Equal(codexHookTrustApproved(report), []string{"stop:0:0"}); got != tc.approved {
				t.Errorf("approved=%v not approved=%v, want approved %v", codexHookTrustApproved(report), report.NotApproved, tc.approved)
			}
			if !tc.approved && (len(report.NotApproved) != 1 || report.NotApproved[0] != (codexHookTrustItem{Layer: 0, Tail: "stop:0:0"})) {
				t.Errorf("not approved = %v, want stop:0:0 of the layer keyed under [%s %s]", report.NotApproved, linkKey, realKey)
			}
		})
	}
}

func TestCodexHookTrustFor_ReportsEachInputApart(t *testing.T) {
	const marker = "config-content-marker-91c2"
	cases := []struct {
		name            string
		hooks           string // "" leaves hooks.json missing; "<dir>" makes it a directory
		config          string // "" leaves config.toml missing; "<dir>" makes it a directory
		hooksState      codexHookSourceState
		configState     codexHookSourceState
		classified      bool
		notApprovedTail []string
	}{
		{"hooks.json missing", "", `model = "x"`, codexHookSourceMissing, codexHookSourceRead, false, nil},
		{"hooks.json not JSON", "{", `model = "x"`, codexHookSourceInvalid, codexHookSourceRead, false, nil},
		{"hooks.json a directory", "<dir>", `model = "x"`, codexHookSourceUnreadable, codexHookSourceRead, false, nil},
		{"config missing", codexHookTrustFixtureHooks, "", codexHookSourceRead, codexHookSourceMissing, true,
			[]string{"post_tool_use:0:0", "pre_tool_use:0:0", "pre_tool_use:0:1", "session_start:0:0"}},
		{"config a directory", codexHookTrustFixtureHooks, "<dir>", codexHookSourceRead, codexHookSourceUnreadable, false, nil},
		{"config not decodable", codexHookTrustFixtureHooks, marker + " = = 1", codexHookSourceRead, codexHookSourceInvalid, false, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := codexHookTrustTempDir(t)
			repo := filepath.Join(tmp, "repo")
			makeCodexHookTrustCheckout(t, repo)
			hooksPath := filepath.Join(repo, ".codex", "hooks.json")
			configPath := filepath.Join(tmp, "codex-home", "config.toml")
			for path, content := range map[string]string{hooksPath: tc.hooks, configPath: tc.config} {
				switch content {
				case "":
				case "<dir>":
					mkdirCodexHookTrust(t, path)
				default:
					writeCodexHookTrustFile(t, path, content)
				}
			}

			report := codexHookTrustFor(repo, codexSandboxEnv{ConfigPath: configPath})

			if len(report.Layers) != 1 {
				t.Fatalf("layers = %+v, want the root layer alone", report.Layers)
			}
			root := report.Layers[0]
			if root.HooksPath() != hooksPath || root.State != tc.hooksState || (root.Err != nil) != (tc.hooksState == codexHookSourceUnreadable || tc.hooksState == codexHookSourceInvalid) {
				t.Errorf("hooks: %s %v %v, want %s %v", root.HooksPath(), root.State, root.Err, hooksPath, tc.hooksState)
			}
			if report.ConfigState != tc.configState || (report.ConfigErr != nil) != (tc.configState == codexHookSourceUnreadable || tc.configState == codexHookSourceInvalid) {
				t.Errorf("config: %v %v, want %v", report.ConfigState, report.ConfigErr, tc.configState)
			}
			if report.ConfigErr != nil && strings.Contains(report.ConfigErr.Error(), marker) {
				t.Errorf("config error %q repeats the config's content", report.ConfigErr)
			}
			if report.Classified() != tc.classified {
				t.Errorf("classified = %v, want %v", report.Classified(), tc.classified)
			}
			if got := codexHookTrustTails(report.NotApproved); !slices.Equal(got, tc.notApprovedTail) {
				t.Errorf("not approved = %v, want %v", got, tc.notApprovedTail)
			}
			if !tc.classified && len(report.Off) != 0 {
				t.Errorf("off=%v, want none when not classified", report.Off)
			}
		})
	}
}
