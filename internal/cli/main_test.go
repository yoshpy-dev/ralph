package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// TestMain pins the shell-alias check's environment resolver
// (doctorShellAliasEnv) to a home directory that does not exist, so no
// runDoctor*-based test in this package reads the developer's real shell rc
// files (self-review L7, docs/reports/self-review-2026-09-18-doctor-shell-alias-check.md):
// without this seam, every pre-existing runDoctor* call site would silently
// start reading every candidate rc file under the real $HOME.
//
// It also pins the codex-sandbox check's environment resolver
// (doctorCodexSandboxEnv) to a config path that does not exist, for the same
// reason: without this seam, every runDoctor*-based test would silently
// start reading the developer's real ~/.codex/config.toml.
//
// It also pins the merge-guarantee check's resolver (doctorMergeGuaranteeEnv)
// to an Exec that runs nothing and fails, a gh lookup that always fails, and
// a state dir that does not exist, for the same reason: without this seam,
// every runDoctor*-based test would silently start running the real `git
// remote get-url origin` and `gh api` against GitHub, and writing
// merge-guarantee.json into the repository's own org ledger whenever that
// ledger exists. A test of the check sets its own resolver and restores
// this one.
//
// It also raises probeBinary's bound (doctorProbeTimeout) to a minute: the
// stub binaries' `--version` can take more than 5 s on a heavily loaded
// machine. Four runDoctor*-based tests failed that way under load: the three
// in the doctor-test row of docs/tech-debt/README.md (2026-10-19 and
// 2026-10-21) and TestRunDoctorOpts_HerdrAgmsgAbsent_ExitCodeUnaffected
// (docs/reports/test-2026-10-13-org-plan-approval.md); three of them printed
// "claude --version timed out after 5s". 5 s stays doctor's bound for users;
// the test of the timeout sets its own bound and restores this one.
//
// It also pins org.go's codex model-observation overrides
// (orgCodexSessionsDirOverride/orgCodexModelObserveTimeoutOverride/
// orgCodexModelObserveIntervalOverride) to a sessions directory that does
// not exist and near-zero durations, for the same reason: without this
// seam, every runOrg*-based test would silently start reading the
// developer's real ~/.codex/sessions and waiting out the real 8s poll
// timeout on every codex spawn/stop.
//
// It also pins orgDesktopNotifyOverride to a stub that does nothing, so no
// runOrg*-based `ralph org escalate`, `ralph org inbox notify`, or `ralph
// org watch --all` (its inbox deadlines and its deadman) shows a real
// desktop notification (osascript on darwin). A test that checks the
// notification swaps in its own stub and restores this one.
//
// It also sets orgWatchLaunch, the start of the detached watch of `ralph
// org watch --all --detach` and `ralph org start`, to one that panics: in a
// test, os.Executable is the test binary, so a real launch would exec it. A
// test that launches a watch sets its own launcher. And it shortens how long
// a watch waits for watch.lock and how often it looks for a stop request
// (orgWatchLockWaitOverride, orgWatchStopPollIntervalOverride), so a refused
// second watch and a `--stop` take milliseconds instead of seconds.
//
// It also sets orgWatchGateOverride to a watch gate that always passes, so
// the tests of spawn, start, and send run as they did before the gate,
// with no watch on their ledger, and start launches none; a test of the
// gate sets it back to nil (useRealOrgWatchGate). And it shortens the
// gate's settle (orgWatchGateSettleOverride), so a refusal of a stale watch
// comes in milliseconds instead of 2 s.
//
// It also points how director start with [org.director].launch_cmd finds
// the agent and the ralph executable (orgDirectorAgentLookPath,
// orgDirectorRalphExecutable) at a failure, so such a start is refused
// before it writes anything: PATH may hold a real claude or codex, and
// os.Executable is the test binary, which a launch script would run as its
// check-in. A test of launch_cmd sets its own (setDirectorLaunchSeams).
//
// It also turns off git's automatic maintenance for every git command the
// tests run (disableGitAutoMaintenance). `git commit` starts `git
// maintenance run --auto --quiet --detach` in the background (seen with git
// 2.49 through GIT_TRACE2_EVENT), and that
// process creates and removes .git/objects/maintenance.lock in the test's
// repository while the test walks it: snapshotDirHashes then fails with
// lstat ... no such file or directory (TestRunMigrateLegacy_ConfirmUX in
// PR #220's CI, TestRunEjectIO_LegacyManifest_FailsClosed in PR #232's).
//
// It also sets RALPH_LOCAL_CONFIG (config.LocalConfigEnv) to off, so
// config.Load reads no personal ralph.local.toml, for two reasons. A test
// whose working directory or config path is inside this repository would
// otherwise read the developer's ralph.local.toml at the main worktree's
// root (config.LocalPath), and its result would depend on that file. And
// with off, Load does not look for the file and runs no git for it, so the
// tests that count the git calls of a command
// (TestStatusCmd_MergeGuaranteeRunsNoGH,
// TestOrgMergeOrder_NoLedgerPrintsTheNoBranchesNoteWithoutRunningGit) see
// the calls they did before the personal file existed. A test of the
// personal file sets it to "" (t.Setenv), which reads the file.
//
// It also pins driver.AgmsgHomeOverride to an agmsg home that does not
// exist, so no test reaches the developer's real agmsg (the default
// ~/.agents/skills/agmsg, or the one the environment or a ralph.toml
// names): every agmsg call of a runOrg*-based test and the agmsg checks of
// runDoctor* go through driver.ResolveAgmsgHome, which returns the
// override first. setupOrgStubPATH points it at its stub agmsg, a test that
// needs another home sets its own (setAgmsgHomeOverride), and a test of how
// ResolveAgmsgHome reads the environment or [org].agmsg_home clears it.
func TestMain(m *testing.M) {
	if err := disableGitAutoMaintenance(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.Setenv(config.LocalConfigEnv, "off"); err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: set %s: %v\n", config.LocalConfigEnv, err)
		os.Exit(1)
	}
	driver.AgmsgHomeOverride = filepath.Join(os.TempDir(), "ralph-cli-test-no-such-home", "agmsg")
	doctorShellAliasEnv = func() (shellAliasEnv, error) {
		return shellAliasEnv{Home: filepath.Join(os.TempDir(), "ralph-cli-test-no-such-home")}, nil
	}
	doctorCodexSandboxEnv = func() (codexSandboxEnv, error) {
		noSuchHome := filepath.Join(os.TempDir(), "ralph-cli-test-no-such-home")
		return codexSandboxEnv{Home: noSuchHome, ConfigPath: filepath.Join(noSuchHome, ".codex", "config.toml")}, nil
	}
	doctorMergeGuaranteeEnv = func(targetDir string) (org.MergeGuaranteeParams, string) {
		return org.MergeGuaranteeParams{
			Dir: targetDir,
			Exec: func(_ context.Context, _ string, _ []string, name string, args ...string) (string, error) {
				return "", fmt.Errorf("%s %s not run: commands are disabled in tests", name, strings.Join(args, " "))
			},
			LookPath: func(string) (string, error) { return "", errors.New("gh lookup disabled in tests") },
		}, filepath.Join(os.TempDir(), "ralph-cli-test-no-such-home", "org-state-do-not-exist")
	}
	doctorProbeTimeout = time.Minute
	orgCodexSessionsDirOverride = filepath.Join(os.TempDir(), "ralph-cli-test-no-such-home", ".codex-sessions-do-not-exist")
	orgCodexModelObserveTimeoutOverride = 50 * time.Millisecond
	orgCodexModelObserveIntervalOverride = 5 * time.Millisecond
	orgDesktopNotifyOverride = func(context.Context, string) error { return nil }
	orgWatchLaunch = func(req orgWatchLaunchRequest) (*orgWatchProcess, error) {
		panic(fmt.Sprintf("a test tried to launch a real detached ralph org watch (args %q, log %s); set orgWatchLaunch to a fake", req.args, req.logPath))
	}
	orgWatchLockWaitOverride = 50 * time.Millisecond
	orgWatchStopPollIntervalOverride = 5 * time.Millisecond
	orgWatchGateOverride = func() error { return nil }
	orgWatchGateSettleOverride = 50 * time.Millisecond
	orgDirectorAgentLookPath = func(file string) (string, error) {
		return "", fmt.Errorf("a test looked up the %s agent for a launch_cmd director; set orgDirectorAgentLookPath to a fake (setDirectorLaunchSeams)", file)
	}
	orgDirectorRalphExecutable = func() (string, error) {
		return "", errors.New("a test asked for the ralph executable of a launch_cmd director; set orgDirectorRalphExecutable to a wrapper (setDirectorLaunchSeams)")
	}
	os.Exit(m.Run())
}

// setAgmsgHomeOverride points driver.AgmsgHomeOverride at home until the
// test ends, when the value it had comes back. An empty home clears the
// override, so driver.ResolveAgmsgHome reads the environment and
// [org].agmsg_home as it does outside the tests.
func setAgmsgHomeOverride(t *testing.T, home string) {
	t.Helper()
	old := driver.AgmsgHomeOverride
	t.Cleanup(func() { driver.AgmsgHomeOverride = old })
	driver.AgmsgHomeOverride = home
}

// disableGitAutoMaintenance sets maintenance.auto=false and gc.auto=0 for
// every git command this test process and its children run, through the
// GIT_CONFIG_COUNT, GIT_CONFIG_KEY_<n>, and GIT_CONFIG_VALUE_<n> variables.
// It adds the two after any that the caller's environment already sets.
func disableGitAutoMaintenance() error {
	n := 0
	if v := os.Getenv("GIT_CONFIG_COUNT"); v != "" {
		var err error
		if n, err = strconv.Atoi(v); err != nil || n < 0 {
			return fmt.Errorf("TestMain: GIT_CONFIG_COUNT=%q is not a count of git config variables", v)
		}
	}
	for _, kv := range [][2]string{{"maintenance.auto", "false"}, {"gc.auto", "0"}} {
		if err := os.Setenv(fmt.Sprintf("GIT_CONFIG_KEY_%d", n), kv[0]); err != nil {
			return fmt.Errorf("TestMain: set GIT_CONFIG_KEY_%d: %w", n, err)
		}
		if err := os.Setenv(fmt.Sprintf("GIT_CONFIG_VALUE_%d", n), kv[1]); err != nil {
			return fmt.Errorf("TestMain: set GIT_CONFIG_VALUE_%d: %w", n, err)
		}
		n++
	}
	if err := os.Setenv("GIT_CONFIG_COUNT", strconv.Itoa(n)); err != nil {
		return fmt.Errorf("TestMain: set GIT_CONFIG_COUNT: %w", err)
	}
	return nil
}

// TestMain_AGitCommitInATestStartsNoAutoMaintenance pins what TestMain's
// disableGitAutoMaintenance is for: a commit that a test makes, the way
// runMigrationTestGit does, starts no `git maintenance run --auto`. Git
// writes every child process it starts to the GIT_TRACE2_EVENT file, so the
// trace names maintenance when the setting does not reach the commit.
func TestMain_AGitCommitInATestStartsNoAutoMaintenance(t *testing.T) {
	dir := t.TempDir()
	trace := filepath.Join(t.TempDir(), "trace2.json")
	runMigrationTestGit(t, dir, "init", "-q")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runMigrationTestGit(t, dir, "add", "a.txt")
	cmd := exec.Command("git", "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-q", "-m", "fixture")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TRACE2_EVENT="+trace)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	got, err := os.ReadFile(trace)
	if err != nil {
		t.Fatalf("read the trace2 file: %v", err)
	}
	if !strings.Contains(string(got), `"cmd_name"`) {
		t.Fatalf("the trace2 file has no cmd_name event, so it says nothing about maintenance:\n%s", got)
	}
	if strings.Contains(string(got), "maintenance") {
		t.Errorf("git commit started git maintenance with GIT_CONFIG_COUNT=%s:\n%s", os.Getenv("GIT_CONFIG_COUNT"), got)
	}
}

// TestDisableGitAutoMaintenance_RefusesAValueThatIsNotACount covers the
// error path: a GIT_CONFIG_COUNT that is not a non-negative number is
// refused before any variable is set, since git would refuse it too.
func TestDisableGitAutoMaintenance_RefusesAValueThatIsNotACount(t *testing.T) {
	for _, v := range []string{"two", "-1"} {
		t.Setenv("GIT_CONFIG_COUNT", v)
		err := disableGitAutoMaintenance()
		if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("GIT_CONFIG_COUNT=%q", v)) {
			t.Errorf("GIT_CONFIG_COUNT=%q: err = %v, want the refusal that names the value", v, err)
		}
		if got := os.Getenv("GIT_CONFIG_COUNT"); got != v {
			t.Errorf("GIT_CONFIG_COUNT=%q: it is %q after the refusal, want it unchanged", v, got)
		}
	}
}
