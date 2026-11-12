package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// CLI tests of the scaffold version note (issue #313; plan
// docs/plans/active/2026-11-12-scaffold-version-notice.md, AC1 to AC5 and
// the --help half of AC8): org start in both forms, org director attach,
// org director start, and status print scaffoldVersionNotice's line once on
// stderr when the scaffold they read is older than the CLI. Every test that
// needs an older scaffold sets the package's Version and puts the previous
// value back, and none runs in parallel.

// setScaffoldVersionCLI sets the package's Version to v until the test ends,
// when the value it had comes back. Other tests leave Version at values such
// as "0.1.0-test", so a test of the note never assumes "dev".
func setScaffoldVersionCLI(t *testing.T, v string) {
	t.Helper()
	prev := Version
	t.Cleanup(func() { Version = prev })
	Version = v
}

// scaffoldVersionOutsideGit keeps git from finding a repository at or above
// cwd, and from the GIT_DIR-style variables a git hook may export, so
// orgScaffoldRoot falls through to ".", the working directory.
func scaffoldVersionOutsideGit(t *testing.T, cwd string) {
	t.Helper()
	isolateFeatureGitEnv(t)
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(cwd))
}

// TestScaffoldVersionOrgStart_TaskFormNotesTheScaffoldOnce is AC1 for
// `ralph org start <task>`: outside git, with the ledger named by
// --state-dir, the working directory's 7.0.0 scaffold under an 8.1.0 CLI
// gets the note, and nothing else, on stderr, and the leader is spawned.
func TestScaffoldVersionOrgStart_TaskFormNotesTheScaffoldOnce(t *testing.T) {
	setScaffoldVersionCLI(t, "8.1.0")
	setupOrgStubPATH(t)
	t.Chdir(t.TempDir())
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	scaffoldVersionOutsideGit(t, cwd)
	writeScaffoldVersionManifest(t, cwd, "[meta]\nversion = \"7.0.0\"\n")
	stateDir := filepath.Join(t.TempDir(), "state")

	stdout, stderr, err := runOrgCmdStreams(t, "start", "--org-id", "org-a", "--cwd", t.TempDir(), "--scope", "org-a",
		"--state-dir", stateDir, "task text")
	if err != nil {
		t.Fatalf("start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if want := scaffoldVersionNoticeWant("org start", cwd, "7.0.0", "8.1.0") + "\n"; stderr != want {
		t.Errorf("stderr =\n%q\nwant\n%q", stderr, want)
	}
	if last := lastSeatEvent(t, stateDir, "org-a", org.LeaderIdentity); last.Event != org.EventSpawned {
		t.Errorf("leader's last event = %+v; want spawned", last)
	}
}

// TestScaffoldVersionOrgStartPlan_ReadsTheMainCheckoutScaffold is AC1 and
// AC5 for `ralph org start --plan`: run from a linked worktree whose
// committed scaffold is 8.1.0 while the main checkout's is 7.0.0, start
// --plan names the main checkout once on stderr, never the linked
// worktree, and starts the feature's leader.
func TestScaffoldVersionOrgStartPlan_ReadsTheMainCheckoutScaffold(t *testing.T) {
	setScaffoldVersionCLI(t, "8.1.0")
	setupOrgStubPATH(t)
	root := newFeatureRepo(t, ".claude/worktrees/\n.harness/\n")
	writeScaffoldVersionManifest(t, root, "[meta]\nversion = \"7.0.0\"\n")
	runLedgerTestGit(t, "-C", root, "add", ".ralph/manifest.toml")
	runLedgerTestGit(t, "-C", root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
		"commit", "--quiet", "-m", "scaffold 7.0.0")
	linked := filepath.Join(root, ".claude", "worktrees", "task")
	runLedgerTestGit(t, "-C", root, "worktree", "add", "--quiet", linked, "-b", "task")
	writeScaffoldVersionManifest(t, linked, "[meta]\nversion = \"8.1.0\"\n")
	runLedgerTestGit(t, "-C", linked, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
		"commit", "--quiet", "-am", "the task's scaffold 8.1.0")
	stateDir := filepath.Join(root, ".harness", "state", "org")
	plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
	t.Chdir(linked)

	stdout, stderr, err := startOrgFeature(t, plan, "b")
	if err != nil {
		t.Fatalf("start --plan --feature b: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	want := scaffoldVersionNoticeWant("org start", root, "7.0.0", "8.1.0") + "\n"
	if stderr != want {
		t.Errorf("stderr =\n%q\nwant\n%q", stderr, want)
	}
	if strings.Contains(stderr, linked) {
		t.Errorf("stderr names the linked worktree %s: %q", linked, stderr)
	}
	if last := lastSeatEvent(t, stateDir, "b", org.LeaderIdentity); last.Event != org.EventSpawned {
		t.Errorf("b's leader = %+v; want spawned", last)
	}
}

// TestScaffoldVersionDirectorAttach_ReadsTheMainWorktreeScaffold is AC3 and
// AC5 for `ralph org director attach` from a linked worktree: with the main
// worktree's ledger, the main worktree's scaffold decides the note (an
// older main is named, an older worktree copy under a current main gives
// none), and so it does with a ledger named by --state-dir outside the
// repository. attach records the registration either way.
func TestScaffoldVersionDirectorAttach_ReadsTheMainWorktreeScaffold(t *testing.T) {
	cases := []struct {
		name                string
		main, linked        string // meta.version of each scaffold
		stateDirOutsideRepo bool
		wantNotice          bool
	}{
		{"main older, worktree current: the main root is named", "7.0.0", "8.1.0", false, true},
		{"main current, worktree older: no note", "8.1.0", "7.0.0", false, false},
		{"--state-dir outside the repository, main older: the main root is named", "7.0.0", "8.1.0", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setScaffoldVersionCLI(t, "8.1.0")
			setEscalateWatch(t, acceptingWatchLiveness())
			r := newLegacyLedgerRepo(t)
			writeScaffoldVersionManifest(t, r.mainRoot, "[meta]\nversion = \""+tc.main+"\"\n")
			writeScaffoldVersionManifest(t, r.worktree, "[meta]\nversion = \""+tc.linked+"\"\n")
			args := []string{"director", "attach", "--driver", "claude"}
			if tc.stateDirOutsideRepo {
				args = append(args, "--state-dir", filepath.Join(t.TempDir(), "org"))
			}

			stdout, stderr, err := runOrgCmdStreams(t, args...)
			if err != nil || !strings.HasPrefix(stdout, "attached d1 (driver=claude)\n") {
				t.Fatalf("attach = %q, %v (stderr: %s); want attached d1", stdout, err, stderr)
			}
			want := ""
			if tc.wantNotice {
				want = scaffoldVersionNoticeWant("org director attach", r.mainRoot, "7.0.0", "8.1.0") + "\n"
			}
			if stderr != want {
				t.Errorf("stderr =\n%q\nwant\n%q", stderr, want)
			}
			if strings.Contains(stderr, r.worktree) {
				t.Errorf("stderr names the linked worktree %s: %q", r.worktree, stderr)
			}
		})
	}
}

// TestScaffoldVersionDirectorStart_NotesTheScaffoldOnce is AC3 for `ralph
// org director start`: outside git, with the ledger named by --state-dir,
// the working directory's 7.0.0 scaffold under an 8.1.0 CLI gets the note,
// and nothing else, on stderr, and the director starts.
func TestScaffoldVersionDirectorStart_NotesTheScaffoldOnce(t *testing.T) {
	setScaffoldVersionCLI(t, "8.1.0")
	stateDir, _, cwd := directorStartLedger(t)
	scaffoldVersionOutsideGit(t, cwd)
	writeScaffoldVersionManifest(t, cwd, "[meta]\nversion = \"7.0.0\"\n")

	stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("director start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if want := scaffoldVersionNoticeWant("org director start", cwd, "7.0.0", "8.1.0") + "\n"; stderr != want {
		t.Errorf("stderr =\n%q\nwant\n%q", stderr, want)
	}
	if !strings.HasPrefix(stdout, "started d1 (") {
		t.Errorf("stdout = %q; want it to start with started d1 (", stdout)
	}
}

// TestScaffoldVersionStatus_NotesOnStderrAndLeavesStdout is AC2: with the
// current directory's scaffold at 7.0.0 and an 8.1.0 CLI, `ralph status`
// and `ralph status --json` print the note once on stderr, and stdout is
// byte for byte what the same command prints with a dev CLI, which prints
// no note; the --json stdout still parses as JSON. stdout keeps the legacy
// layout's own pointer at `ralph upgrade` in backticks, so the check for
// the note's words looks for the quoted run 'ralph upgrade'.
func TestScaffoldVersionStatus_NotesOnStderrAndLeavesStdout(t *testing.T) {
	prev := Version
	t.Cleanup(func() { Version = prev })
	t.Chdir(t.TempDir())
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	writeScaffoldVersionManifest(t, cwd, "[meta]\nversion = \"7.0.0\"\n")
	stateDir := t.TempDir()
	note := scaffoldVersionNoticeWant("status", cwd, "7.0.0", "8.1.0") + "\n"

	for _, jsonOut := range []bool{false, true} {
		args := []string{"status", "--state-dir", stateDir}
		if jsonOut {
			args = append(args, "--json")
		}
		Version = "dev"
		quietOut, quietErr, err := runRootCmdSplit(t, args...)
		if err != nil {
			t.Fatalf("%v with a dev CLI: %v (stderr: %s)", args, err, quietErr)
		}
		if strings.Contains(quietErr, "run 'ralph upgrade'") {
			t.Fatalf("%v with a dev CLI: stderr = %q; want no note", args, quietErr)
		}

		Version = "8.1.0"
		stdout, stderr, err := runRootCmdSplit(t, args...)
		if err != nil {
			t.Fatalf("%v: %v (stderr: %s)", args, err, stderr)
		}
		if n := strings.Count(stderr, note); n != 1 {
			t.Errorf("%v: stderr has the note %d times; want once:\n%s", args, n, stderr)
		}
		if stdout != quietOut {
			t.Errorf("%v: stdout changed with the note:\n%s\nwant\n%s", args, stdout, quietOut)
		}
		if strings.Contains(stdout, "run 'ralph upgrade'") {
			t.Errorf("%v: stdout has the note's words:\n%s", args, stdout)
		}
		if jsonOut {
			var payload map[string]any
			if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
				t.Errorf("status --json stdout does not parse: %v\n%s", err, stdout)
			}
		}
	}
}

// TestScaffoldVersionStatus_SilentCases is AC4 at the command level, through
// `ralph status`: no note for a dev CLI, a CLI version that is not semver,
// a scaffold as new as the CLI or newer, an empty meta.version, or no
// manifest.
func TestScaffoldVersionStatus_SilentCases(t *testing.T) {
	cases := []struct {
		name, cli, manifest string // manifest "": no .ralph/manifest.toml
	}{
		{"dev CLI", "dev", "[meta]\nversion = \"7.0.0\"\n"},
		{"CLI version not semver", "abc", "[meta]\nversion = \"7.0.0\"\n"},
		{"same version", "8.1.0", "[meta]\nversion = \"8.1.0\"\n"},
		{"newer scaffold", "8.1.0", "[meta]\nversion = \"9.0.0\"\n"},
		{"empty meta.version", "8.1.0", "[meta]\nversion = \"\"\n"},
		{"no manifest", "8.1.0", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setScaffoldVersionCLI(t, tc.cli)
			dir := t.TempDir()
			if tc.manifest != "" {
				writeScaffoldVersionManifest(t, dir, tc.manifest)
			}
			t.Chdir(dir)

			stdout, stderr, err := runRootCmdSplit(t, "status", "--state-dir", t.TempDir())
			if err != nil {
				t.Fatalf("status: %v (stderr: %s)", err, stderr)
			}
			if strings.Contains(stderr, "run 'ralph upgrade'") || strings.Contains(stdout, "run 'ralph upgrade'") {
				t.Errorf("status printed the note:\nstdout: %s\nstderr: %s", stdout, stderr)
			}
		})
	}
}

// TestScaffoldVersionHelp_EachVerbDescribesTheNote is the --help half of
// AC8: org start, org director attach, and org director start end their
// Long with orgScaffoldVersionNoticeHelp as a paragraph of its own, and
// status with statusScaffoldVersionNoticeHelp; both name run 'ralph
// upgrade', and their lines are wrapped like the rest of the help.
func TestScaffoldVersionHelp_EachVerbDescribesTheNote(t *testing.T) {
	for _, para := range []string{orgScaffoldVersionNoticeHelp, statusScaffoldVersionNoticeHelp} {
		if !strings.Contains(para, "run 'ralph upgrade' in <that directory>") {
			t.Errorf("paragraph does not name run 'ralph upgrade' in <that directory>:\n%s", para)
		}
		for _, line := range strings.Split(para, "\n") {
			if len(line) > 76 {
				t.Errorf("help line of %d characters; want at most 76: %q", len(line), line)
			}
		}
	}
	root := NewRootCmd()
	for _, tc := range []struct {
		path []string
		para string
	}{
		{[]string{"org", "start"}, orgScaffoldVersionNoticeHelp},
		{[]string{"org", "director", "attach"}, orgScaffoldVersionNoticeHelp},
		{[]string{"org", "director", "start"}, orgScaffoldVersionNoticeHelp},
		{[]string{"status"}, statusScaffoldVersionNoticeHelp},
	} {
		cmd, rest, err := root.Find(tc.path)
		if err != nil || len(rest) != 0 || cmd.Name() != tc.path[len(tc.path)-1] {
			t.Fatalf("find %v = %v, %v, %v; want that command", tc.path, cmd, rest, err)
		}
		if !strings.Contains(cmd.Long, "\n\n"+tc.para) {
			t.Errorf("%v --help does not have the scaffold version paragraph:\n%s", tc.path, cmd.Long)
		}
	}
}
