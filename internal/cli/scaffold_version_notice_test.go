package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/gitroot"
)

// writeScaffoldVersionManifest writes body as root's .ralph/manifest.toml.
func writeScaffoldVersionManifest(t *testing.T, root, body string) {
	t.Helper()
	dir := filepath.Join(root, ".ralph")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.toml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

// scaffoldVersionNoticeWant is the line scaffoldVersionNotice returns for
// verb when the scaffold in dir is manifestVersion and the CLI cliVersion.
func scaffoldVersionNoticeWant(verb, dir, manifestVersion, cliVersion string) string {
	return "note: " + verb + ": the scaffold in " + dir + " (.ralph/manifest.toml) is ralph " + manifestVersion +
		", older than this CLI (" + cliVersion + "), so its skills and rules may not match this CLI; run 'ralph upgrade' in " + dir
}

// TestScaffoldVersionOlder_ComparesSemver checks the ordering of AC6 and the
// versions AC4 keeps silent: a version either side has that is not semver
// is never older.
func TestScaffoldVersionOlder_ComparesSemver(t *testing.T) {
	cases := []struct {
		name     string
		manifest string
		cli      string
		want     bool
	}{
		{"older major", "7.0.0", "8.1.0", true},
		{"numbers compare as numbers", "8.1.0", "8.10.0", true},
		{"prerelease is older than its release", "8.2.0-rc.1", "8.2.0", true},
		{"release is not older than its prerelease", "8.2.0", "8.2.0-rc.1", false},
		{"release older than a later prerelease CLI", "8.1.0", "8.2.0-rc.1", true},
		{"same version", "8.1.0", "8.1.0", false},
		{"leading v on one side only", "v8.1.0", "8.1.0", false},
		{"leading v on the older side", "v7.0.0", "8.1.0", true},
		{"newer manifest", "8.2.0", "8.1.0", false},
		{"cli dev", "7.0.0", "dev", false},
		{"empty manifest version", "", "8.1.0", false},
		{"empty cli version", "7.0.0", "", false},
		{"garbage manifest version", "not-a-version", "8.1.0", false},
		{"garbage cli version", "7.0.0", "8.1.0.0", false},
		{"build metadata is ignored", "8.1.0+abc", "8.1.0", false},
		{"surrounding spaces", " 7.0.0\t", "\n8.1.0 ", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := scaffoldVersionOlder(tc.manifest, tc.cli); got != tc.want {
				t.Errorf("scaffoldVersionOlder(%q, %q) = %v, want %v", tc.manifest, tc.cli, got, tc.want)
			}
		})
	}
}

// TestScaffoldVersionNotice_SilentCases checks AC4 at the helper level: no
// line when the manifest is missing or broken, when the scaffold is the
// same version or newer, and when either version is not semver.
func TestScaffoldVersionNotice_SilentCases(t *testing.T) {
	cases := []struct {
		name     string
		manifest string // "" writes no manifest
		cli      string
	}{
		{"no manifest", "", "8.1.0"},
		{"broken toml", "[meta\nversion = \n", "8.1.0"},
		{"same version", "[meta]\nversion = \"8.1.0\"\n", "8.1.0"},
		{"newer manifest", "[meta]\nversion = \"8.2.0\"\n", "8.1.0"},
		{"cli dev", "[meta]\nversion = \"7.0.0\"\n", "dev"},
		{"empty manifest version", "[meta]\nversion = \"\"\n", "8.1.0"},
		{"no meta table", "[files]\n", "8.1.0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if tc.manifest != "" {
				writeScaffoldVersionManifest(t, root, tc.manifest)
			}
			if got := scaffoldVersionNotice("org start", root, tc.cli); got != "" {
				t.Errorf("scaffoldVersionNotice = %q, want \"\"", got)
			}
		})
	}
}

// TestScaffoldVersionNotice_OlderScaffoldNamesTheRootAndBothVersions checks
// the exact line for a scaffold older than the CLI: the verb, the absolute
// root, both versions as written, and `run 'ralph upgrade' in <root>`.
func TestScaffoldVersionNotice_OlderScaffoldNamesTheRootAndBothVersions(t *testing.T) {
	root := t.TempDir()
	writeScaffoldVersionManifest(t, root, "[meta]\nversion = \"7.0.0\"\n")

	got := scaffoldVersionNotice("org start", root, "8.1.0")
	if want := scaffoldVersionNoticeWant("org start", root, "7.0.0", "8.1.0"); got != want {
		t.Errorf("scaffoldVersionNotice =\n%q\nwant\n%q", got, want)
	}
}

// TestScaffoldVersionNotice_TrimsVersionsAndKeepsTheirV checks that the line
// shows each version trimmed of spaces but otherwise as written: no "v" is
// added to a version without one, and none is taken from one with it.
func TestScaffoldVersionNotice_TrimsVersionsAndKeepsTheirV(t *testing.T) {
	root := t.TempDir()
	writeScaffoldVersionManifest(t, root, "[meta]\nversion = \" v7.0.0 \"\n")

	got := scaffoldVersionNotice("status", root, " 8.1.0\n")
	if want := scaffoldVersionNoticeWant("status", root, "v7.0.0", "8.1.0"); got != want {
		t.Errorf("scaffoldVersionNotice =\n%q\nwant\n%q", got, want)
	}
}

// TestScaffoldVersionNotice_RelativeRootIsShownAbsolute checks that the "."
// `ralph status` passes is named as the absolute working directory.
func TestScaffoldVersionNotice_RelativeRootIsShownAbsolute(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	writeScaffoldVersionManifest(t, root, "[meta]\nversion = \"7.0.0\"\n")
	t.Chdir(root)
	abs, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if !filepath.IsAbs(abs) || !gitroot.SamePath(abs, root) {
		t.Fatalf("filepath.Abs(\".\") = %q, want an absolute path naming %q", abs, root)
	}

	got := scaffoldVersionNotice("status", ".", "8.1.0")
	if want := scaffoldVersionNoticeWant("status", abs, "7.0.0", "8.1.0"); got != want {
		t.Errorf("scaffoldVersionNotice =\n%q\nwant\n%q", got, want)
	}
}

// TestScaffoldVersionOrgRoot_LedgerPathNamesTheRootWithoutGit checks the
// first tier of orgScaffoldRoot: for a ledger the git-main-worktree tier
// found, the root is read back from the ledger's path. With no git on PATH,
// any later tier would give ".".
func TestScaffoldVersionOrgRoot_LedgerPathNamesTheRootWithoutGit(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PATH", t.TempDir())
	stateDir := filepath.Join(root, ".harness", "state", "org")

	if got := orgScaffoldRoot(stateDir, "git-main-worktree"); got != root {
		t.Errorf("orgScaffoldRoot(%q, git-main-worktree) = %q, want %q", stateDir, got, root)
	}
}

// TestScaffoldVersionOrgRoot_LinkedWorktreeReadsTheMainWorktree checks the
// second tier and AC5 at the helper level: from a linked worktree, with a
// ledger named by --state-dir outside the repository, the root is the main
// worktree's, so its manifest decides the line and the worktree's copy does
// not.
func TestScaffoldVersionOrgRoot_LinkedWorktreeReadsTheMainWorktree(t *testing.T) {
	r := newLegacyLedgerRepo(t)
	stateDir := filepath.Join(t.TempDir(), "ledger")

	got := orgScaffoldRoot(stateDir, "flag")
	if got != r.mainRoot {
		t.Fatalf("orgScaffoldRoot(%q, flag) = %q, want the main worktree %q", stateDir, got, r.mainRoot)
	}

	// The worktree's copy is older and the main worktree's is current: no line.
	writeScaffoldVersionManifest(t, r.worktree, "[meta]\nversion = \"7.0.0\"\n")
	writeScaffoldVersionManifest(t, r.mainRoot, "[meta]\nversion = \"8.1.0\"\n")
	if notice := scaffoldVersionNotice("org start", got, "8.1.0"); notice != "" {
		t.Errorf("stale worktree copy, current main: scaffoldVersionNotice = %q, want \"\"", notice)
	}

	// The main worktree's is older and the worktree's is current: the line names the main worktree.
	writeScaffoldVersionManifest(t, r.worktree, "[meta]\nversion = \"8.1.0\"\n")
	writeScaffoldVersionManifest(t, r.mainRoot, "[meta]\nversion = \"7.0.0\"\n")
	want := scaffoldVersionNoticeWant("org start", r.mainRoot, "7.0.0", "8.1.0")
	if notice := scaffoldVersionNotice("org start", got, "8.1.0"); notice != want {
		t.Errorf("old main: scaffoldVersionNotice =\n%q\nwant\n%q", notice, want)
	}
}

// TestScaffoldVersionOrgRoot_OutsideGitIsTheWorkingDirectory checks the last
// tier: outside git, with a ledger named by --state-dir, the root is ".".
func TestScaffoldVersionOrgRoot_OutsideGitIsTheWorkingDirectory(t *testing.T) {
	isolateFeatureGitEnv(t)
	cwd, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(cwd)) // no repository above the temp dir either
	t.Chdir(cwd)
	stateDir := filepath.Join(t.TempDir(), "ledger")

	if got := orgScaffoldRoot(stateDir, "flag"); got != "." {
		t.Errorf("orgScaffoldRoot(%q, flag) outside git = %q, want \".\"", stateDir, got)
	}
}

// TestScaffoldVersionPrint_NoManifestWritesNothing checks that
// printScaffoldVersionNotice writes nothing when there is no line.
func TestScaffoldVersionPrint_NoManifestWritesNothing(t *testing.T) {
	var buf bytes.Buffer
	printScaffoldVersionNotice(&buf, "org start", t.TempDir())
	if buf.Len() != 0 {
		t.Errorf("printScaffoldVersionNotice wrote %q, want nothing", buf.String())
	}
}

// TestScaffoldVersionPrint_OlderScaffoldWritesTheLine checks that
// printScaffoldVersionNotice compares with the package's Version and ends
// the line with a newline. It sets Version and puts the previous value back.
func TestScaffoldVersionPrint_OlderScaffoldWritesTheLine(t *testing.T) {
	prev := Version
	t.Cleanup(func() { Version = prev })
	Version = "8.1.0"
	root := t.TempDir()
	writeScaffoldVersionManifest(t, root, "[meta]\nversion = \"7.0.0\"\n")

	var buf bytes.Buffer
	printScaffoldVersionNotice(&buf, "director start", root)
	if want := scaffoldVersionNoticeWant("director start", root, "7.0.0", "8.1.0") + "\n"; buf.String() != want {
		t.Errorf("printScaffoldVersionNotice wrote\n%q\nwant\n%q", buf.String(), want)
	}
}
