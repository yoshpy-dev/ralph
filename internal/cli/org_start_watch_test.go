package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// Tests for `ralph org start` launching the cross-org watch (plan
// 2026-10-12-org-watch-gate, AC3, AC4, AC5, AC6, AC13, AC14 and AC15 on the
// start side). Each test sets TestMain's gate that always passes back to nil
// (useRealOrgWatchGate), so start goes through ensureOrgWatch and the real
// gate, and sets its own launcher (TestMain's panics): a fake child that
// holds watch.lock (lateMarkLauncher), or the in-process watch
// (inProcessLauncher); none execs the test binary. None of these tests runs
// in parallel: they swap package-level vars, and the --plan ones chdir.

// writeWatchMarkLater writes mark as the liveness mark of stateDir after
// delay, by a rename as a watch writes it, and sends the write's error on
// the returned channel.
func writeWatchMarkLater(t *testing.T, stateDir string, mark *org.WatchAlive, delay time.Duration) <-chan error {
	t.Helper()
	data, err := json.Marshal(mark)
	if err != nil {
		t.Fatal(err)
	}
	written := make(chan error, 1)
	go func() {
		time.Sleep(delay)
		tmp := org.WatchAlivePathIn(stateDir) + ".tmp"
		err := os.WriteFile(tmp, data, 0o644)
		if err == nil {
			err = os.Rename(tmp, org.WatchAlivePathIn(stateDir))
		}
		written <- err
	}()
	return written
}

// lateMarkLauncher is an orgWatchLaunch whose child, with pid, takes
// watch.lock at once and writes its running mark only after delay, as a
// watch does between its lock and its first mark; it then runs until stop
// (or the test's cleanup) frees the lock. It appends each request it was
// given to got.
func lateMarkLauncher(t *testing.T, stateDir string, pid int, delay time.Duration, got *[]orgWatchLaunchRequest) func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
	return func(req orgWatchLaunchRequest) (*orgWatchProcess, error) {
		*got = append(*got, req)
		release := holdWatchLock(t, stateDir, nil)
		written := writeWatchMarkLater(t, stateDir, runningMark(pid, time.Now()), delay)
		exited := make(chan struct{})
		var once sync.Once
		stop := func() {
			once.Do(func() {
				if err := <-written; err != nil {
					t.Error(err)
				}
				release()
				close(exited)
			})
		}
		t.Cleanup(stop)
		return &orgWatchProcess{pid: pid, exited: exited, stop: stop}, nil
	}
}

// orgStartForm is one form of `ralph org start` on a ledger: setup makes the
// ledger (and, for --plan, the repo it runs in) and returns its state dir
// and a function that runs the start with extra flags.
type orgStartForm struct {
	name  string
	setup func(t *testing.T) (stateDir string, run func(extra ...string) (stdout, stderr string, err error))
	// noWorktree checks, for --plan, that the start made no feature
	// worktree and no branch.
	noWorktree func(t *testing.T)
}

// orgStartForms are `ralph org start <task>` and `ralph org start --plan`,
// each on a ledger outside a repository (a --state-dir under a temp dir),
// so the watch they launch keeps this process's working directory.
func orgStartForms() []orgStartForm {
	var root string
	return []orgStartForm{
		{
			name: "<task>",
			setup: func(t *testing.T) (string, func(...string) (string, string, error)) {
				stateDir := t.TempDir()
				cwd := t.TempDir()
				return stateDir, func(extra ...string) (string, string, error) {
					args := append([]string{"start", "--org-id", "org-a", "--driver", "claude", "--model", "sonnet",
						"--cwd", cwd, "--scope", "org-wide", "--state-dir", stateDir, "do the task"}, extra...)
					return runOrgCmdStreams(t, args...)
				}
			},
			noWorktree: func(*testing.T) {},
		},
		{
			name: "--plan",
			setup: func(t *testing.T) (string, func(...string) (string, string, error)) {
				root = newFeatureRepo(t, ".claude/worktrees/\n")
				t.Chdir(root)
				tmp, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				stateDir := filepath.Join(tmp, "state")
				plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
				return stateDir, func(extra ...string) (string, string, error) {
					return startOrgFeature(t, plan, "a", append([]string{"--state-dir", stateDir}, extra...)...)
				}
			},
			noWorktree: func(t *testing.T) {
				t.Helper()
				wt := filepath.Join(root, ".claude", "worktrees", "org-a")
				if _, err := os.Stat(wt); !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("stat %s = %v; want no worktree", wt, err)
				}
				out, err := exec.Command("git", "-C", root, "branch", "--list", "feat/a").Output()
				if err != nil {
					t.Fatalf("git branch --list: %v", err)
				}
				if branches := strings.TrimSpace(string(out)); branches != "" {
					t.Errorf("branches named feat/a = %q; want none", branches)
				}
			},
		},
	}
}

// startedWatchLine is the stderr line of a start that launched the watch
// with pid on the ledger stateDir.
func startedWatchLine(stateDir string, pid int) string {
	return fmt.Sprintf("ralph org start: started the cross-org watch (pid %d); log: %s\n", pid, filepath.Join(stateDir, org.WatchLogFileName))
}

// AC3: on a ledger with no watch, both forms of start launch the watch once
// (the --plan form runs the gate twice, in StartFeature and in Spawn), with
// the ledger's foreground watch as the child, wait for its running mark
// (written 200ms after the launch, past the gate's own settle), say so on
// stderr, and spawn the leader. The watch keeps running after start
// returns, and the same start again finds it running and launches nothing.
func TestOrgStart_LaunchesTheWatchWhenNoneRuns(t *testing.T) {
	for _, form := range orgStartForms() {
		t.Run(form.name, func(t *testing.T) {
			useRealOrgWatchGate(t)
			setupOrgStubPATH(t)
			stateDir, run := form.setup(t)
			var got []orgWatchLaunchRequest
			setOrgWatchLaunch(t, lateMarkLauncher(t, stateDir, 4401, 200*time.Millisecond, &got))

			start := time.Now()
			stdout, stderr, err := run()
			elapsed := time.Since(start)
			if err != nil {
				t.Fatalf("start with no watch: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			if len(got) != 1 {
				t.Fatalf("launches = %d; want 1", len(got))
			}
			if want := []string{"org", "watch", "--all", "--state-dir", stateDir}; !slices.Equal(got[0].args, want) {
				t.Errorf("child args = %q; want %q", got[0].args, want)
			}
			if want := filepath.Join(stateDir, org.WatchLogFileName); got[0].logPath != want {
				t.Errorf("child log = %q; want %q", got[0].logPath, want)
			}
			if got[0].dir != "" {
				t.Errorf("child dir = %q; want this process's working directory for a ledger outside the repository", got[0].dir)
			}
			if n := strings.Count(stderr, startedWatchLine(stateDir, 4401)); n != 1 {
				t.Errorf("stderr has the started line %d times; want once:\n%s", n, stderr)
			}
			if !strings.HasPrefix(stdout, `spawned seat "leader" `) {
				t.Errorf("stdout = %q; want the leader spawned", stdout)
			}
			if elapsed < 150*time.Millisecond {
				t.Errorf("start returned after %v; want it to wait for the mark written 200ms after the launch", elapsed)
			}
			if probe := org.ProbeWatch(stateDir, time.Now(), 10); !org.WatchAcceptsWork(probe) || probe.Mark.PID != 4401 {
				t.Errorf("watch after start = %+v; want the launched watch (pid 4401) still running", probe)
			}

			stdout, stderr, err = run()
			if err != nil {
				t.Fatalf("the same start again: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			if len(got) != 1 {
				t.Errorf("launches after the second start = %d; want still 1", len(got))
			}
			if strings.Contains(stderr, "started the cross-org watch") {
				t.Errorf("stderr of the second start = %q; want no started line", stderr)
			}
			if !strings.Contains(stdout, `seat "leader" already spawned`) {
				t.Errorf("stdout of the second start = %q; want the leader already spawned", stdout)
			}
		})
	}
}

// AC4: a launched watch that takes watch.lock but never writes a running
// mark is stopped and reaped once the deadline (shortened) has passed, and
// the start is refused naming watch.log, with no leader, no worktree, and
// nothing recorded or called.
func TestOrgStart_ALaunchedWatchThatNeverRuns_IsStoppedAndTheStartRefused(t *testing.T) {
	for _, form := range orgStartForms() {
		t.Run(form.name, func(t *testing.T) {
			useRealOrgWatchGate(t)
			shortenOrgWatchDeadline(t)
			setOrgWatchDuration(t, &orgWatchReleaseDeadline, 100*time.Millisecond)
			herdrLog, agmsgLog := setupOrgStubPATH(t)
			stateDir, run := form.setup(t)
			stops := 0
			setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
				release := holdWatchLock(t, stateDir, nil)
				exited := make(chan struct{})
				return &orgWatchProcess{pid: 4402, exited: exited, stop: func() {
					stops++
					release()
					close(exited)
				}}, nil
			})

			stdout, _, err := run()
			want := "org: start: the cross-org watch did not start, so no leader was started: the watch (pid 4402) was not running within 300ms, so it was stopped; see " +
				filepath.Join(stateDir, org.WatchLogFileName)
			if err == nil || err.Error() != want {
				t.Fatalf("start with a watch that never runs = %v; want %q", err, want)
			}
			if stdout != "rejected: "+want+"\n" {
				t.Errorf("stdout = %q; want only the rejected line", stdout)
			}
			if stops != 1 {
				t.Errorf("the child was stopped %d times; want once", stops)
			}
			if got := org.ProbeWatch(stateDir, time.Now(), 10).State; got != org.WatchLivenessNotRunning {
				t.Errorf("ProbeWatch after start = %q; want not_running", got)
			}
			checkNothingStarted(t, stateDir, herdrLog, agmsgLog)
			form.noWorktree(t)
		})
	}
}

// A launch that fails (mutation survivor E8 of /test cycle 1 of
// org-watch-gate) refuses `ralph org start <task>` with start's prefix,
// which says no leader was started, before the launch's error, and the start
// records and calls nothing.
func TestOrgStart_ALaunchThatFails_IsRefusedWithThePrefixAndNothingStarted(t *testing.T) {
	useRealOrgWatchGate(t)
	failOrgWatchLaunch(t)
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	stateDir := t.TempDir()
	stdout, _, err := runOrgCmdStreams(t, "start", "--org-id", "org-a", "--driver", "claude", "--model", "sonnet",
		"--cwd", t.TempDir(), "--scope", "org-wide", "--state-dir", stateDir, "do the task")
	want := "org: start: the cross-org watch did not start, so no leader was started: start /x/ralph: boom"
	if err == nil || err.Error() != want {
		t.Fatalf("start with a launch that fails = %v; want %q", err, want)
	}
	if stdout != "rejected: "+want+"\n" {
		t.Errorf("stdout = %q; want only the rejected line", stdout)
	}
	checkNothingStarted(t, stateDir, herdrLog, agmsgLog)
}

// Self-review C2-3 of org-watch-gate for `ralph org start <task>`: the
// launched watch never runs and keeps watch.lock after its stop, and the
// refusal names `ralph status` with --state-dir, quoted, when it was given,
// and without it for a ledger from RALPH_ORG_STATE_DIR.
func TestOrgStart_AStoppedWatchThatKeepsTheLock_NamesRalphStatusWithAnExplicitStateDir(t *testing.T) {
	for _, given := range []bool{true, false} {
		t.Run(fmt.Sprintf("--state-dir given %t", given), func(t *testing.T) {
			useRealOrgWatchGate(t)
			shortenOrgWatchDeadline(t)
			setOrgWatchDuration(t, &orgWatchReleaseDeadline, 50*time.Millisecond)
			setupOrgStubPATH(t)
			stateDir := filepath.Join(t.TempDir(), "led ger")
			keepLockOrgWatchLaunch(t, stateDir, 4413)
			args := []string{"start", "--org-id", "org-a", "--driver", "claude", "--model", "sonnet",
				"--cwd", t.TempDir(), "--scope", "org-wide"}
			t.Setenv(org.EnvOrgStateDir, "")
			want := "; see " + filepath.Join(stateDir, org.WatchLogFileName) + " and ralph status"
			if given {
				args = append(args, "--state-dir", stateDir)
				want += " --state-dir '" + stateDir + "'"
			} else {
				t.Setenv(org.EnvOrgStateDir, stateDir)
			}

			_, _, err := runOrgCmdStreams(t, append(args, "do the task")...)
			if err == nil || !strings.HasPrefix(err.Error(), "org: start: the cross-org watch did not start, so no leader was started: ") ||
				!strings.HasSuffix(err.Error(), want) {
				t.Fatalf("start with a watch that keeps the lock = %v; want start's refusal ending %q", err, want)
			}
		})
	}
}

// AC5 (tech-debt 206): the launched child exits first, refused the lock by
// the watch of a concurrent start or --detach, whose mark says running.
// start takes that watch, says it launched nothing, and spawns the leader.
func TestOrgStart_AChildThatExitsWhileAnotherWatchRuns_StartGoesOn(t *testing.T) {
	useRealOrgWatchGate(t)
	setupOrgStubPATH(t)
	form := orgStartForms()[0]
	stateDir, run := form.setup(t)
	launches := 0
	setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
		launches++
		holdWatchLock(t, stateDir, runningMark(4403, time.Now()))
		exited := make(chan struct{})
		close(exited)
		return &orgWatchProcess{pid: 4404, exited: exited, stop: func() {}}, nil
	})

	stdout, stderr, err := run()
	if err != nil {
		t.Fatalf("start while another watch runs: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if launches != 1 {
		t.Errorf("launches = %d; want 1", launches)
	}
	if strings.Contains(stderr, "started the cross-org watch") {
		t.Errorf("stderr = %q; want no started line for a child that exited", stderr)
	}
	if !strings.HasPrefix(stdout, `spawned seat "leader" (org_id=org-a `) {
		t.Errorf("stdout = %q; want the leader spawned", stdout)
	}
}

// The finding of /test cycle 1 for start (see
// TestOrgWatchDetach_AnotherWatchRuns_ReturnsOnlyOnceItsChildHasExited):
// another watch runs, and the child this start launched, refused the lock,
// exits 500ms after its launch. Both forms of start go on only once the
// child has exited (when it exits, nothing has called herdr, and --plan has
// made no feature worktree), launch once, print no started line, and spawn
// the leader.
func TestOrgStart_AnotherWatchRuns_GoesOnOnlyOnceItsChildHasExited(t *testing.T) {
	for _, form := range orgStartForms() {
		t.Run(form.name, func(t *testing.T) {
			useRealOrgWatchGate(t)
			setOrgWatchDuration(t, &orgWatchChildExitWait, 3*time.Second)
			setOrgWatchDuration(t, &orgWatchProbeInterval, 10*time.Millisecond)
			herdrLog, _ := setupOrgStubPATH(t)
			stateDir, run := form.setup(t)
			// The --plan form runs in its repo (setup chdirs there) and makes
			// the feature worktree before it spawns the leader; no such path
			// exists for <task>.
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			worktree := filepath.Join(wd, ".claude", "worktrees", "org-a")
			launches := 0
			var exited chan struct{}
			// wentOnFirst is set before exited closes when the start had
			// already called herdr or made the worktree then.
			var wentOnFirst bool
			setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
				launches++
				holdWatchLock(t, stateDir, runningMark(4411, time.Now()))
				ch := make(chan struct{})
				go func() {
					time.Sleep(500 * time.Millisecond)
					_, herdrErr := os.Stat(herdrLog)
					_, worktreeErr := os.Stat(worktree)
					wentOnFirst = herdrErr == nil || worktreeErr == nil
					close(ch)
				}()
				exited = ch
				return &orgWatchProcess{pid: 4412, exited: ch, stop: func() { t.Error("the child that exits on its own was stopped") }}, nil
			})

			stdout, stderr, err := run()
			if exited == nil {
				t.Fatalf("start launched no watch: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			select {
			case <-exited:
			default:
				t.Error("start returned while its child was still alive; want it to go on once the child has exited (500ms)")
			}
			<-exited
			if wentOnFirst {
				t.Error("start called herdr or made the worktree before the child exited; want it to go on only once the child has")
			}
			if err != nil {
				t.Fatalf("start while another watch runs: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			if launches != 1 {
				t.Errorf("launches = %d; want 1", launches)
			}
			if strings.Contains(stderr, "started the cross-org watch") {
				t.Errorf("stderr = %q; want no started line for a child that exited", stderr)
			}
			if !strings.HasPrefix(stdout, `spawned seat "leader" `) {
				t.Errorf("stdout = %q; want the leader spawned", stdout)
			}
		})
	}
}

// The launched child exits, refused watch.lock by a watch that took it
// first and accepts no work: one that is stopping, a --once run, or stale
// until the deadline (shortened). Both forms of start are refused with the
// gate's refusal for that watch, as when the first probe finds it, not with
// the child's exit (self-review L1 of org-watch-gate), and launch once,
// print no started line, and record or make nothing.
func TestOrgStart_AChildThatExitsWhileAWatchThatAcceptsNoWorkHoldsTheLock_IsRefusedAsTheGateRefuses(t *testing.T) {
	watches := []struct {
		name string
		mark func() *org.WatchAlive
		want []string
	}{
		{name: "stopping", mark: func() *org.WatchAlive { return stoppingWatchMark(4408, time.Now()) },
			want: []string{"(pid 4408) is stopping", "run this again once it has ended"}},
		{name: "running --once", mark: func() *org.WatchAlive { return onceWatchMark(4409, time.Now()) },
			want: []string{"(pid 4409) is a --once run", "run this again once it has ended"}},
		{name: "stale until the deadline", mark: func() *org.WatchAlive { return nil },
			want: []string{"has not refreshed its liveness mark (pid unknown)", "stop it with ralph org watch --stop"}},
	}
	for _, form := range orgStartForms() {
		for _, w := range watches {
			t.Run(form.name+" "+w.name, func(t *testing.T) {
				useRealOrgWatchGate(t)
				shortenOrgWatchDeadline(t)
				herdrLog, agmsgLog := setupOrgStubPATH(t)
				stateDir, run := form.setup(t)
				launches := 0
				setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
					launches++
					holdWatchLock(t, stateDir, w.mark())
					exited := make(chan struct{})
					close(exited)
					return &orgWatchProcess{pid: 4410, exited: exited, stop: func() { t.Error("the child that exited was stopped") }}, nil
				})

				stdout, stderr, err := run()
				checkGateRefusal(t, err, stateDir, w.want...)
				if strings.Contains(err.Error(), "exited before it was running") {
					t.Errorf("error = %q; want the refusal of the watch that holds the lock, not the child's exit", err)
				}
				if launches != 1 {
					t.Errorf("launches = %d; want 1", launches)
				}
				if want := "rejected: " + err.Error() + "\n"; stdout != want {
					t.Errorf("stdout = %q; want only %q", stdout, want)
				}
				if strings.Contains(stderr, "started the cross-org watch") {
					t.Errorf("stderr = %q; want no started line for a child that exited", stderr)
				}
				checkNothingStarted(t, stateDir, herdrLog, agmsgLog)
				form.noWorktree(t)
			})
		}
	}
}

// AC13 and AC14: a watch that holds watch.lock and is stopping, or is a
// --once run, accepts no work and ends soon: both forms of start are
// refused at once with the gate's refusal, launch nothing, and record or
// make nothing.
func TestOrgStart_AStoppingOrOnceWatch_IsRefusedAndNothingLaunched(t *testing.T) {
	watches := []struct {
		name string
		mark *org.WatchAlive
		want string
	}{
		{name: "stopping", mark: stoppingWatchMark(4405, time.Now()), want: "(pid 4405) is stopping"},
		{name: "running --once", mark: onceWatchMark(4406, time.Now()), want: "(pid 4406) is a --once run"},
	}
	for _, form := range orgStartForms() {
		for _, w := range watches {
			t.Run(form.name+" "+w.name, func(t *testing.T) {
				useRealOrgWatchGate(t)
				refuseOrgWatchLaunch(t)
				setOrgWatchDuration(t, &orgWatchEnsureDeadline, 3*time.Second)
				herdrLog, agmsgLog := setupOrgStubPATH(t)
				stateDir, run := form.setup(t)
				if err := os.MkdirAll(stateDir, 0o755); err != nil {
					t.Fatal(err)
				}
				holdWatchLock(t, stateDir, w.mark)

				start := time.Now()
				stdout, _, err := run()
				checkGateRefusal(t, err, stateDir, w.want, "run this again once it has ended")
				if elapsed := time.Since(start); elapsed >= time.Second {
					t.Errorf("start refused the %s watch after %v; want at once, not after a wait", w.name, elapsed)
				}
				if want := "rejected: " + err.Error() + "\n"; stdout != want {
					t.Errorf("stdout = %q; want only %q", stdout, want)
				}
				checkNothingStarted(t, stateDir, herdrLog, agmsgLog)
				form.noWorktree(t)
			})
		}
	}
}

// AC15: a watch that holds watch.lock and writes its first running mark
// 200ms later (stale until then, longer than the gate's settle) is waited
// for: both forms of start launch nothing, print no started line, and
// spawn the leader once the mark is there.
func TestOrgStart_WaitsForAWatchThatWritesItsFirstMarkLate(t *testing.T) {
	for _, form := range orgStartForms() {
		t.Run(form.name, func(t *testing.T) {
			useRealOrgWatchGate(t)
			refuseOrgWatchLaunch(t)
			setupOrgStubPATH(t)
			stateDir, run := form.setup(t)
			holdWatchLock(t, stateDir, nil)
			written := writeWatchMarkLater(t, stateDir, runningMark(4407, time.Now()), 200*time.Millisecond)

			start := time.Now()
			stdout, stderr, err := run()
			elapsed := time.Since(start)
			if werr := <-written; werr != nil {
				t.Fatal(werr)
			}
			if err != nil {
				t.Fatalf("start while a watch writes its first mark: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			if strings.Contains(stderr, "started the cross-org watch") {
				t.Errorf("stderr = %q; want no started line", stderr)
			}
			if !strings.HasPrefix(stdout, `spawned seat "leader" `) {
				t.Errorf("stdout = %q; want the leader spawned", stdout)
			}
			if elapsed < 150*time.Millisecond {
				t.Errorf("start returned after %v; want it to wait for the mark written after 200ms", elapsed)
			}
		})
	}
}

// F10 for start (mutation survivor E4 of /test cycle 1 of org-watch-gate):
// the holder of watch.lock is stale (a mark an hour old) and releases the
// lock 200ms later, during ensureOrgWatch's wait. Both forms of start then
// launch the watch once, say so on stderr, and spawn the leader, rather than
// refusing a watch that has ended.
func TestOrgStart_AStaleHolderThatEndsDuringTheWait_LaunchesTheWatch(t *testing.T) {
	for _, form := range orgStartForms() {
		t.Run(form.name, func(t *testing.T) {
			useRealOrgWatchGate(t)
			setOrgWatchDuration(t, &orgWatchEnsureDeadline, 3*time.Second)
			setOrgWatchDuration(t, &orgWatchProbeInterval, 10*time.Millisecond)
			setupOrgStubPATH(t)
			stateDir, run := form.setup(t)
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatal(err)
			}
			release := holdWatchLock(t, stateDir, runningMark(4501, time.Now().Add(-time.Hour)))
			go func() { time.Sleep(200 * time.Millisecond); release() }()
			var got []orgWatchLaunchRequest
			setOrgWatchLaunch(t, lateMarkLauncher(t, stateDir, 4502, 50*time.Millisecond, &got))

			stdout, stderr, err := run()
			if err != nil {
				t.Fatalf("start: %v (stdout %q stderr %q)", err, stdout, stderr)
			}
			if len(got) != 1 {
				t.Fatalf("launches = %d; want 1", len(got))
			}
			if !strings.Contains(stderr, startedWatchLine(stateDir, 4502)) {
				t.Errorf("stderr = %q; want the started line", stderr)
			}
			if !strings.HasPrefix(stdout, `spawned seat "leader" `) {
				t.Errorf("stdout = %q; want the leader spawned", stdout)
			}
		})
	}
}

// AC7 for `ralph org start <task>` (mutation survivor E7d of /test cycle 1):
// ensureOrgWatch judges a mark that carries no deadman_minutes by the
// caller's config, as the gate does. With deadman_minutes 30, a
// 20-minute-old mark is running, so start launches nothing and spawns the
// leader.
func TestOrgStart_AMarkWithoutDeadmanMinutes_IsJudgedByTheCallersConfig(t *testing.T) {
	useRealOrgWatchGate(t)
	refuseOrgWatchLaunch(t)
	shortenOrgWatchDeadline(t)
	setupOrgStubPATH(t)
	stateDir, configPath := ledgerWithAMarkWithoutDeadman(t, 4520, 30, 20*time.Minute)
	stdout, stderr, err := runOrgCmdStreams(t, "start", "--org-id", "org-a", "--driver", "claude", "--model", "sonnet",
		"--cwd", t.TempDir(), "--scope", "org-wide", "--state-dir", stateDir, "--config", configPath, "do the task")
	if err != nil {
		t.Fatalf("start: %v (stdout %q stderr %q)", err, stdout, stderr)
	}
	if !strings.HasPrefix(stdout, `spawned seat "leader" `) {
		t.Errorf("stdout = %q; want the leader spawned", stdout)
	}
}

// A start refused for its own input, before the gate, launches no watch on
// a ledger with none: an unknown feature of the split plan, and an org_id
// too long for herdr's agent name next to the leader seat.
func TestOrgStart_ARefusedInputLaunchesNoWatch(t *testing.T) {
	t.Run("--plan with an unknown feature", func(t *testing.T) {
		useRealOrgWatchGate(t)
		refuseOrgWatchLaunch(t)
		setupOrgStubPATH(t)
		form := orgStartForms()[1]
		stateDir, _ := form.setup(t)
		plan := filepath.Join(org.SplitPlansDirIn(stateDir), "demo.md")

		_, _, err := startOrgFeature(t, plan, "zzz", "--state-dir", stateDir)
		if err == nil || errors.Is(err, org.ErrNoRunningWatch) || !strings.Contains(err.Error(), "zzz") {
			t.Fatalf("start --plan of an unknown feature = %v; want refused for the feature", err)
		}
		form.noWorktree(t)
	})
	t.Run("<task> with an org_id too long", func(t *testing.T) {
		useRealOrgWatchGate(t)
		refuseOrgWatchLaunch(t)
		herdrLog, agmsgLog := setupOrgStubPATH(t)
		stateDir := t.TempDir()

		_, _, err := runOrgCmdStreams(t, "start", "--org-id", "abcdefghijklmnopqrstuvwxyz", "--driver", "claude", "--model", "sonnet",
			"--cwd", t.TempDir(), "--scope", "org-wide", "--state-dir", stateDir, "do the task")
		if err == nil || errors.Is(err, org.ErrNoRunningWatch) || !strings.Contains(err.Error(), "agent-name limit") {
			t.Fatalf("start with a long org_id = %v; want refused for the agent name", err)
		}
		checkNothingStarted(t, stateDir, herdrLog, agmsgLog)
	})
}

// AC6 and F6 for start: `ralph org start <task>` from a subdirectory of the
// main checkout, on the default ledger (the main worktree's
// .harness/state/org) and with a relative --config, launches the watch in
// the main worktree root with the absolute --config and without the herdr
// pane variables, the same child --detach launches (checkOrgWatchChild).
// The in-process watch then runs, and the leader is spawned.
func TestOrgStart_TheWatchRunsInTheMainWorktreeWithoutTheHerdrPaneEnv(t *testing.T) {
	useRealOrgWatchGate(t)
	setupOrgStubPATH(t)
	root := newFeatureRepo(t, ".claude/worktrees/\n.harness/\n")
	sub := filepath.Join(root, "pkg", "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeOrgConfig(t, sub, 5)
	ledger := filepath.Join(root, ".harness", "state", "org")
	for _, name := range orgWatchChildDroppedEnv {
		t.Setenv(name, "from-the-leader-pane")
	}
	t.Setenv("RALPH_TEST_CHILD_KEEP", "kept")
	t.Chdir(sub)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	var got []orgWatchLaunchRequest
	setOrgWatchLaunch(t, inProcessLauncher(t, ledger, &got))

	stdout, stderr, err := runOrgCmdStreams(t, "start", "--org-id", "org-a", "--driver", "claude", "--model", "sonnet",
		"--cwd", sub, "--scope", "org-wide", "--config", "ralph.toml", "do the task")
	if err != nil {
		t.Fatalf("start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if len(got) != 1 {
		t.Fatalf("launches = %d; want 1", len(got))
	}
	checkOrgWatchChild(t, got[0], ledger, filepath.Join(wd, "ralph.toml"), root)
	if !strings.Contains(stderr, startedWatchLine(ledger, os.Getpid())) {
		t.Errorf("stderr = %q; want %q", stderr, startedWatchLine(ledger, os.Getpid()))
	}
	if ev := lastSeatEvent(t, ledger, "org-a", org.LeaderIdentity); ev.Event != org.EventSpawned {
		t.Errorf("last leader event = %s; want %s", ev.Event, org.EventSpawned)
	}
}

// Plan 2026-10-29-local-config-org AC7 (D7) for start: both forms of `ralph
// org start`, on a ledger with no watch, with a --config whose
// [org].agmsg_home is relative, are refused before the watch is launched,
// with start's prefix and the sentence of director start's refusal, and
// launch, record, call, or make nothing. RALPH_ORG_AGMSG_HOME, which ralph
// no longer reads, is set to the stub agmsg home and changes nothing. These
// ledgers are outside the repository, so the watch would keep this working
// directory: the refusal does not depend on where start runs.
func TestOrgStart_ARelativeAgmsgHomeInConfigIsRefusedAndNothingLaunched(t *testing.T) {
	for _, form := range orgStartForms() {
		t.Run(form.name, func(t *testing.T) {
			useRealOrgWatchGate(t)
			refuseOrgWatchLaunch(t)
			herdrLog, agmsgLog := setupOrgStubPATH(t)
			t.Setenv(legacyAgmsgHomeEnv, driver.AgmsgHomeOverride)
			stateDir, run := form.setup(t)
			configPath := writeOrgConfig(t, t.TempDir(), 5)
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			writeTestFile(t, configPath, strings.Replace(string(data), "[org]\n", "[org]\nagmsg_home = \"rel/agmsg\"\n", 1))

			stdout, _, err := run("--config", configPath)
			want := relativeAgmsgHomeRefusal("org: start: the cross-org watch did not start, so no leader was started: ", "the watch runs", "rel/agmsg", configPath)
			if err == nil || err.Error() != want {
				t.Fatalf("start with a relative agmsg_home in --config = %v; want %q", err, want)
			}
			if stdout != "rejected: "+want+"\n" {
				t.Errorf("stdout = %q; want only the rejected line", stdout)
			}
			checkNothingStarted(t, stateDir, herdrLog, agmsgLog)
			form.noWorktree(t)
		})
	}
}

// AC7 of plan 2026-10-29-local-config-org, the other side: `ralph org start
// <task>` from a subdirectory of the main checkout, with a --config whose
// [org].agmsg_home is absolute (the stub agmsg home) or starts with "~/",
// launches the watch in the main worktree root and spawns the leader. The
// watch's environment keeps RALPH_ORG_AGMSG_HOME as it was inherited,
// relative and not made absolute, since nothing reads it (AC6).
func TestOrgStart_AnAbsoluteOrHomeAgmsgHomeInConfigLaunchesTheWatch(t *testing.T) {
	for _, home := range []string{"<stub>", "~/agmsg"} {
		t.Run(home, func(t *testing.T) {
			useRealOrgWatchGate(t)
			setupOrgStubPATH(t)
			if home == "<stub>" {
				home = driver.AgmsgHomeOverride
				setAgmsgHomeOverride(t, "")
			}
			t.Setenv(legacyAgmsgHomeEnv, "rel/agmsg")
			root := newFeatureRepo(t, ".claude/worktrees/\n.harness/\n")
			sub := filepath.Join(root, "pkg", "sub")
			if err := os.MkdirAll(sub, 0o755); err != nil {
				t.Fatal(err)
			}
			configPath := writeOrgConfig(t, sub, 5)
			data, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			writeTestFile(t, configPath, strings.Replace(string(data), "[org]\n", fmt.Sprintf("[org]\nagmsg_home = %q\n", home), 1))
			ledger := filepath.Join(root, ".harness", "state", "org")
			t.Chdir(sub)
			var got []orgWatchLaunchRequest
			setOrgWatchLaunch(t, inProcessLauncher(t, ledger, &got))

			stdout, stderr, err := runOrgCmdStreams(t, "start", "--org-id", "org-a", "--driver", "claude", "--model", "sonnet",
				"--cwd", sub, "--scope", "org-wide", "--config", "ralph.toml", "do the task")
			if err != nil {
				t.Fatalf("start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			if len(got) != 1 {
				t.Fatalf("launches = %d; want 1", len(got))
			}
			if got[0].dir != root {
				t.Errorf("child dir = %q; want %q", got[0].dir, root)
			}
			if homes := legacyAgmsgHomeEnvValues(got[0].env); !slices.Equal(homes, []string{"rel/agmsg"}) {
				t.Errorf("child %s = %q; want only the inherited %q", legacyAgmsgHomeEnv, homes, "rel/agmsg")
			}
			if ev := lastSeatEvent(t, ledger, "org-a", org.LeaderIdentity); ev.Event != org.EventSpawned {
				t.Errorf("last leader event = %s; want %s", ev.Event, org.EventSpawned)
			}
		})
	}
}

// Help (plan 2026-10-12-org-watch-gate, Scope): start says it launches the
// watch and when it is refused, spawn and send say when they are refused.
func TestOrgStartSpawnSendHelp_SayWhatTheWatchGateDoes(t *testing.T) {
	cases := map[string][]string{
		"start": {"when none is running, start launches one as `ralph org watch --all --detach` does",
			"it is refused when a watch holds the lock but is stale, stopping, or a --once run"},
		"spawn": {"A spawn, --dry-run included, is refused while no long-running `ralph org watch --all` (not stopping, not a --once run) runs on the ledger"},
		"send": {"A TASK (TYPE: TASK, also on the first line of --raw text), --dry-run included, is refused while no long-running " +
			"`ralph org watch --all` (not stopping, not a --once run) runs on the ledger"},
	}
	for verb, wants := range cases {
		out, err := runOrgCmd(t, verb, "--help")
		if err != nil {
			t.Fatalf("%s --help: %v", verb, err)
		}
		help := strings.Join(strings.Fields(out), " ")
		for _, want := range wants {
			if !strings.Contains(help, want) {
				t.Errorf("%s --help does not say %q:\n%s", verb, want, out)
			}
		}
	}
}
