package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org watch --all`, `--detach`, and `--stop` (plan
// 2026-10-11-org-cross-watch, AC8, AC9, AC10, AC12, AC12b, AC13, AC13b on
// the CLI side). TestMain sets orgWatchLaunch to one that panics; a test of
// --detach sets its own. None of these tests runs in parallel: they swap
// package-level vars, and some send a signal to this process.

func setOrgWatchDuration(t *testing.T, p *time.Duration, d time.Duration) {
	t.Helper()
	orig := *p
	*p = d
	t.Cleanup(func() { *p = orig })
}

func setOrgWatchLaunch(t *testing.T, launch func(args []string, logPath string) (*orgWatchProcess, error)) {
	t.Helper()
	orig := orgWatchLaunch
	orgWatchLaunch = launch
	t.Cleanup(func() { orgWatchLaunch = orig })
}

// refuseOrgWatchLaunch sets a launcher that fails the test when called.
func refuseOrgWatchLaunch(t *testing.T) {
	t.Helper()
	setOrgWatchLaunch(t, func(args []string, _ string) (*orgWatchProcess, error) {
		t.Errorf("a watch was launched (args %q); want nothing started", args)
		return nil, errors.New("no launch in this test")
	})
}

// runningMark is the liveness mark of a running watch with pid whose last
// cycle ended at ts.
func runningMark(pid int, ts time.Time) *org.WatchAlive {
	return &org.WatchAlive{
		State: org.WatchAliveRunning, PID: pid,
		StartedAt: ts.UTC().Format(time.RFC3339Nano), TS: ts.UTC().Format(time.RFC3339),
		IntervalSeconds: 30,
	}
}

// holdWatchLock holds watch.lock in stateDir for the rest of the test, as
// another watch would, and writes mark as the liveness mark when it is not
// nil. Release (or the cleanup) frees the lock.
func holdWatchLock(t *testing.T, stateDir string, mark *org.WatchAlive) (release func()) {
	t.Helper()
	lock, err := org.AcquireWatchLock(stateDir, 0)
	if err != nil {
		t.Fatalf("hold watch.lock: %v", err)
	}
	var once sync.Once
	release = func() { once.Do(func() { _ = lock.Release() }) }
	t.Cleanup(release)
	if mark != nil {
		data, err := json.Marshal(mark)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(org.WatchAlivePathIn(stateDir), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return release
}

// waitForWatchState waits up to 10s until ProbeWatch on stateDir says want
// and, for running, names pid.
func waitForWatchState(t *testing.T, stateDir, want string, pid int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		probe := org.ProbeWatch(stateDir, time.Now(), 10)
		if probe.State == want && (want != org.WatchLivenessRunning || probe.Mark.PID == pid) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the watch on %s is %q (mark %+v); want %q", stateDir, probe.State, probe.Mark, want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// inProcessWatch is `ralph org watch --all` running in a goroutine of this
// test process (startInProcessWatch).
type inProcessWatch struct {
	done   chan error
	once   sync.Once
	result error
}

// wait returns the command's error once it has ended, and fails the test
// when it has not ended within 10s.
func (w *inProcessWatch) wait(t *testing.T) error {
	t.Helper()
	w.once.Do(func() {
		select {
		case w.result = <-w.done:
		case <-time.After(10 * time.Second):
			w.result = errors.New("the in-process watch did not end within 10s")
			t.Error(w.result)
		}
	})
	return w.result
}

// startInProcessWatch runs `ralph org watch --all` on stateDir, with an hour
// between cycles, in a goroutine, and returns once its mark says running
// with this process's pid. A cleanup stops a watch the test left running
// and waits for it.
func startInProcessWatch(t *testing.T, stateDir string) *inProcessWatch {
	t.Helper()
	w := &inProcessWatch{done: make(chan error, 1)}
	go func() {
		_, _, err := runOrgCmdStreams(t, "watch", "--all", "--interval-seconds", "3600", "--state-dir", stateDir)
		w.done <- err
	}()
	t.Cleanup(func() {
		if org.ProbeWatch(stateDir, time.Now(), 10).State != org.WatchLivenessNotRunning {
			_ = org.WriteWatchStopRequest(stateDir, time.Now())
		}
		_ = w.wait(t)
	})
	waitForWatchState(t, stateDir, org.WatchLivenessRunning, os.Getpid())
	return w
}

// AC8 and the flag rules: --org-id is refused, --all is required except with
// --stop, --stop takes none of --detach, --once, --interval-seconds, and
// --detach does not take --once. Each refusal exits 1 before the state dir
// is touched.
func TestOrgWatch_RefusesFlagCombinationsAndWritesNothing(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "--org-id", args: []string{"--all", "--once", "--org-id", "org-a"}, want: "--org-id does not apply: ralph org watch watches every org of the ledger; drop --org-id and pass --all"},
		{name: "--org-id with --stop", args: []string{"--stop", "--org-id", "org-a"}, want: "--org-id does not apply"},
		{name: "no --all", args: []string{"--once"}, want: "pass --all"},
		{name: "no flag at all", args: nil, want: "pass --all"},
		{name: "--detach without --all", args: []string{"--detach"}, want: "pass --all"},
		{name: "--stop --detach", args: []string{"--all", "--stop", "--detach"}, want: "--stop does not take --detach"},
		{name: "--stop --once", args: []string{"--stop", "--once"}, want: "--stop does not take --once"},
		{name: "--stop --interval-seconds", args: []string{"--stop", "--interval-seconds", "5"}, want: "--stop does not take --interval-seconds"},
		{name: "--detach --once", args: []string{"--all", "--detach", "--once"}, want: "--detach does not take --once"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refuseOrgWatchLaunch(t)
			stateDir := filepath.Join(t.TempDir(), "state")
			args := append([]string{"watch"}, tc.args...)
			stdout, _, err := runOrgCmdStreams(t, append(args, "--state-dir", stateDir)...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("watch %v = %v; want an error saying %q", tc.args, err, tc.want)
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			if _, err := os.Stat(stateDir); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("the state dir after the refusal: %v; want it not created", err)
			}
		})
	}
}

// AC9 on the CLI: a `watch --all --once` on a ledger whose watch.lock
// another watch holds exits 1 naming that watch's pid and leaves the mark
// as it was; once the lock is free, the next one runs.
func TestOrgWatch_ASecondWatchIsRefusedWhileAnotherHoldsTheLock(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	release := holdWatchLock(t, stateDir, runningMark(4242, time.Now()))
	markBefore, err := os.ReadFile(org.WatchAlivePathIn(stateDir))
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = runOrgCmdStreams(t, "watch", "--all", "--once", "--state-dir", stateDir)
	if !errors.Is(err, org.ErrWatchRunning) || !strings.Contains(err.Error(), "pid 4242") {
		t.Fatalf("second watch = %v; want ErrWatchRunning naming pid 4242", err)
	}
	if markAfter, err := os.ReadFile(org.WatchAlivePathIn(stateDir)); err != nil || string(markAfter) != string(markBefore) {
		t.Errorf("the refused watch changed the mark: %s -> %s (%v)", markBefore, markAfter, err)
	}

	release()
	if _, _, err := runOrgCmdStreams(t, "watch", "--all", "--once", "--state-dir", stateDir); err != nil {
		t.Fatalf("watch after the lock was freed: %v", err)
	}
}

// AC10 on the CLI: SIGINT and SIGTERM end `ralph org watch --all` after its
// cycle, through the context signal.NotifyContext cancels, with the stopped
// mark written and watch.lock free. The signal is sent only once the mark
// says running, which RunWatchAll writes after the command installed its
// handler.
func TestOrgWatch_SIGINTOrSIGTERMEndsTheWatchWithTheStoppedMark(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			stateDir := filepath.Join(t.TempDir(), "state")
			w := startInProcessWatch(t, stateDir)
			if err := syscall.Kill(os.Getpid(), sig); err != nil {
				t.Fatalf("send %v: %v", sig, err)
			}
			if err := w.wait(t); err != nil {
				t.Fatalf("watch after %v: %v", sig, err)
			}
			mark, err := org.ReadWatchAlive(stateDir)
			if err != nil || mark.State != org.WatchAliveStopped || mark.StoppedAt == "" || mark.PID != os.Getpid() {
				t.Errorf("mark after %v = %+v (%v); want stopped with stopped_at", sig, mark, err)
			}
			if got := org.ProbeWatch(stateDir, time.Now(), 10).State; got != org.WatchLivenessNotRunning {
				t.Errorf("ProbeWatch after %v = %q; want not_running", sig, got)
			}
		})
	}
}

// The CLI wires the inbox into the watch (newOrgRuntimeAt): an item past
// its ack deadline is sent in the `--once` cycle, through escalations.jsonl,
// the stderr banner (naming the inbox command with the --state-dir this
// watch was given), and the desktop notification.
func TestOrgWatch_Once_SendsAnInboxItemPastItsDeadline(t *testing.T) {
	rec := recordDesktopNotify(t, nil)
	stateDir := filepath.Join(t.TempDir(), "state")
	inbox := org.NewInboxStore(stateDir)
	inbox.Now = func() time.Time { return time.Now().Add(-11 * time.Minute) }
	id, err := inbox.Escalate(org.InboxEscalation{OrgID: "org-a", Type: "QUESTION", Body: inboxQuestion("which one?")})
	if err != nil {
		t.Fatal(err)
	}

	_, stderr, err := runOrgCmdStreams(t, "watch", "--all", "--once", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("watch --all --once: %v (stderr %s)", err, stderr)
	}
	for _, want := range []string{"ORG INBOX DEADLINE: org=org-a item=" + id, org.InboxCommand("show", id, stateDir)} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr %q does not contain %q", stderr, want)
		}
	}
	escLines := readLogLines(t, org.EscalationsPathIn(stateDir))
	if len(escLines) != 1 || !strings.Contains(escLines[0], `"inbox_id":"`+id+`"`) || !strings.Contains(escLines[0], org.InboxReasonAckDeadline) {
		t.Errorf("escalations.jsonl = %v; want one %s line for %s", escLines, org.InboxReasonAckDeadline, id)
	}
	if calls := rec.calls(); len(calls) != 1 || !strings.Contains(calls[0], id) {
		t.Errorf("desktop notifications = %q; want one for %s", calls, id)
	}
}

// AC12: --detach starts nothing when a watch is already running and says
// so with its pid, exit 0.
func TestOrgWatchDetach_AlreadyRunning_StartsNothing(t *testing.T) {
	refuseOrgWatchLaunch(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	holdWatchLock(t, stateDir, runningMark(4242, time.Now()))

	stdout, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("--detach with a running watch: %v", err)
	}
	if want := fmt.Sprintf("ralph org watch --all is already running (pid 4242); log: %s\n", filepath.Join(stateDir, org.WatchLogFileName)); stdout != want {
		t.Errorf("stdout = %q; want %q", stdout, want)
	}
}

// AC12: --detach starts nothing when watch.lock is held with a stale mark
// (none, an old one, a stopped one), and exits 1 saying how to stop it,
// with the pid when the mark names one.
func TestOrgWatchDetach_Stale_StartsNothingAndSaysHowToStopIt(t *testing.T) {
	stopped := runningMark(4242, time.Now())
	stopped.State = org.WatchAliveStopped
	cases := []struct {
		name string
		mark *org.WatchAlive
		want []string
	}{
		{name: "no mark", want: []string{"pid unknown", "find the process that holds"}},
		{name: "a mark older than deadman_minutes", mark: runningMark(4242, time.Now().Add(-time.Hour)), want: []string{"pid 4242", "kill 4242 (the pid in"}},
		{name: "a stopped mark", mark: stopped, want: []string{"pid 4242", "kill 4242 (the pid in"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refuseOrgWatchLaunch(t)
			stateDir := filepath.Join(t.TempDir(), "state")
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatal(err)
			}
			holdWatchLock(t, stateDir, tc.mark)

			_, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir)
			if err == nil {
				t.Fatal("--detach with a stale watch succeeded; want exit 1")
			}
			for _, want := range append([]string{"nothing was started", "ralph org watch --stop"}, tc.want...) {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
		})
	}
}

// inProcessLauncher is an orgWatchLaunch that runs RunWatchAll on stateDir
// in a goroutine of this process, so the child's mark names os.Getpid(),
// and stops it by cancelling its context. It records the args and the log
// path it was given.
func inProcessLauncher(t *testing.T, stateDir string, gotArgs *[]string, gotLog *string) func([]string, string) (*orgWatchProcess, error) {
	return func(args []string, logPath string) (*orgWatchProcess, error) {
		*gotArgs, *gotLog = args, logPath
		rt, err := newOrgRuntimeAt(stateDir, "")
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithCancel(context.Background())
		exited := make(chan struct{})
		go func() {
			defer close(exited)
			_ = rt.RunWatchAll(ctx, org.WatchAllParams{
				StatusDir: stateDir, Interval: time.Hour, Stderr: io.Discard,
				Escalate: func(context.Context, string) error { return nil }, StopPollInterval: 5 * time.Millisecond,
			}, org.WatchHooks{})
		}()
		stop := func() {
			cancel()
			<-exited
		}
		t.Cleanup(stop)
		return &orgWatchProcess{pid: os.Getpid(), exited: exited, stop: stop}, nil
	}
}

// AC12: --detach starts the child with the resolved state dir, plus
// --config and --interval-seconds when they were given, and exits 0 once
// the child's mark says running with the child's pid.
func TestOrgWatchDetach_StartsTheChildAndWaitsForItsRunningMark(t *testing.T) {
	configPath := writeOrgConfig(t, t.TempDir(), 5)
	cases := []struct {
		name     string
		extra    []string
		wantTail []string
	}{
		{name: "no --config or --interval-seconds"},
		{name: "--config and --interval-seconds", extra: []string{"--config", configPath, "--interval-seconds", "7"}, wantTail: []string{"--config", configPath, "--interval-seconds", "7"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := filepath.Join(t.TempDir(), "state")
			var gotArgs []string
			var gotLog string
			setOrgWatchLaunch(t, inProcessLauncher(t, stateDir, &gotArgs, &gotLog))

			args := append([]string{"watch", "--all", "--detach", "--state-dir", stateDir}, tc.extra...)
			stdout, _, err := runOrgCmdStreams(t, args...)
			if err != nil {
				t.Fatalf("--detach: %v", err)
			}
			logPath := filepath.Join(stateDir, org.WatchLogFileName)
			if want := fmt.Sprintf("started ralph org watch --all (pid %d); log: %s\n", os.Getpid(), logPath); stdout != want {
				t.Errorf("stdout = %q; want %q", stdout, want)
			}
			if wantArgs := append([]string{"org", "watch", "--all", "--state-dir", stateDir}, tc.wantTail...); !slices.Equal(gotArgs, wantArgs) {
				t.Errorf("child args = %q; want %q", gotArgs, wantArgs)
			}
			if gotLog != logPath {
				t.Errorf("child log = %q; want %q", gotLog, logPath)
			}
			if got := org.ProbeWatch(stateDir, time.Now(), 10); got.State != org.WatchLivenessRunning {
				t.Errorf("ProbeWatch after --detach = %+v; want the child running", got)
			}
		})
	}
}

// AC12: a child that exits before its mark says running makes --detach
// exit 1 naming the log.
func TestOrgWatchDetach_ChildExitsFirst_ExitsOneNamingTheLog(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	setOrgWatchLaunch(t, func([]string, string) (*orgWatchProcess, error) {
		exited := make(chan struct{})
		close(exited)
		return &orgWatchProcess{pid: 4242, exited: exited, stop: func() {}}, nil
	})
	_, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir)
	want := fmt.Sprintf("the watch (pid 4242) exited before it was running; see %s", filepath.Join(stateDir, org.WatchLogFileName))
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("--detach with a child that exits = %v; want %q", err, want)
	}
}

// AC12b: a child that takes watch.lock but never writes a running mark is
// stopped by its handle when the deadline passes, and --detach exits 1,
// naming the log, only once watch.lock is free. When the lock stays held
// after the stop, the error says so.
func TestOrgWatchDetach_NotRunningInTime_StopsTheChildAndWaitsForTheLock(t *testing.T) {
	cases := []struct {
		name string
		// releaseOnStop: the child's stop frees watch.lock.
		releaseOnStop bool
		want          string
	}{
		{name: "the stop frees the lock", releaseOnStop: true, want: "was not running within 300ms, so it was stopped; see "},
		{name: "the lock stays held", want: ".lock is still held; see "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setOrgWatchDuration(t, &orgWatchDetachDeadline, 300*time.Millisecond)
			setOrgWatchDuration(t, &orgWatchReleaseDeadline, 100*time.Millisecond)
			setOrgWatchDuration(t, &orgWatchProbeInterval, 10*time.Millisecond)
			stateDir := filepath.Join(t.TempDir(), "state")
			var stops int
			setOrgWatchLaunch(t, func([]string, string) (*orgWatchProcess, error) {
				release := holdWatchLock(t, stateDir, nil)
				exited := make(chan struct{})
				stop := func() {
					stops++
					if tc.releaseOnStop {
						release()
					}
					close(exited)
				}
				return &orgWatchProcess{pid: 4242, exited: exited, stop: stop}, nil
			})

			start := time.Now()
			_, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir)
			elapsed := time.Since(start)
			logPath := filepath.Join(stateDir, org.WatchLogFileName)
			if err == nil || !strings.Contains(err.Error(), tc.want+logPath) {
				t.Fatalf("--detach with a child that never runs = %v; want %q", err, tc.want+logPath)
			}
			if stops != 1 {
				t.Errorf("the child was stopped %d times; want once", stops)
			}
			if elapsed < 300*time.Millisecond {
				t.Errorf("--detach gave up after %v; want the 300ms deadline", elapsed)
			}
			wantState := org.WatchLivenessNotRunning
			if !tc.releaseOnStop {
				wantState = org.WatchLivenessStale
			}
			if got := org.ProbeWatch(stateDir, time.Now(), 10).State; got != wantState {
				t.Errorf("ProbeWatch after --detach = %q; want %q", got, wantState)
			}
		})
	}
}

// AC13: --stop on a ledger with no watch says so, exits 0, and writes no
// stop request.
func TestOrgWatchStop_NotRunning_SaysSoAndWritesNoRequest(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	stdout, _, err := runOrgCmdStreams(t, "watch", "--stop", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("--stop with no watch: %v", err)
	}
	if want := "ralph org watch --all is not running on " + stateDir + "\n"; stdout != want {
		t.Errorf("stdout = %q; want %q", stdout, want)
	}
	if _, err := org.ReadWatchStopRequest(stateDir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("stop request: %v; want none", err)
	}
}

// AC13: --stop (with or without --all) writes the stop request, and exits 0
// once the running watch has written its stopped mark and freed the lock.
func TestOrgWatchStop_StopsARunningWatch(t *testing.T) {
	for _, args := range [][]string{{"--stop"}, {"--all", "--stop"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			stateDir := filepath.Join(t.TempDir(), "state")
			w := startInProcessWatch(t, stateDir)

			stdout, _, err := runOrgCmdStreams(t, append(append([]string{"watch"}, args...), "--state-dir", stateDir)...)
			if err != nil {
				t.Fatalf("watch %v: %v", args, err)
			}
			if want := fmt.Sprintf("stopped ralph org watch --all (pid %d)\n", os.Getpid()); stdout != want {
				t.Errorf("stdout = %q; want %q", stdout, want)
			}
			if err := w.wait(t); err != nil {
				t.Errorf("the stopped watch returned %v; want nil", err)
			}
			if mark, err := org.ReadWatchAlive(stateDir); err != nil || mark.State != org.WatchAliveStopped {
				t.Errorf("mark = %+v (%v); want stopped", mark, err)
			}
			if _, err := org.ReadWatchStopRequest(stateDir); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("stop request after the stop: %v; want it removed by the watch", err)
			}
		})
	}
}

// AC13 and AC13b: a holder of watch.lock that never follows the request
// (here: the lock held without a watch loop) makes --stop exit 1 after its
// deadline, naming the pid of the mark, with the request left in place. The
// mark names another live process, as the mark of a watch that has just
// taken the lock still names the watch before it: --stop sends it no
// signal.
func TestOrgWatchStop_AWatchThatDoesNotStopInTime_ExitsOneSendsNoSignal(t *testing.T) {
	setOrgWatchDuration(t, &orgWatchStopDeadline, 300*time.Millisecond)
	setOrgWatchDuration(t, &orgWatchProbeInterval, 10*time.Millisecond)
	bystander := exec.Command("sleep", "30")
	if err := bystander.Start(); err != nil {
		t.Fatal(err)
	}
	bystanderDone := make(chan struct{})
	go func() {
		_ = bystander.Wait()
		close(bystanderDone)
	}()
	t.Cleanup(func() {
		_ = bystander.Process.Kill()
		<-bystanderDone
	})
	stateDir := filepath.Join(t.TempDir(), "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pid := bystander.Process.Pid
	holdWatchLock(t, stateDir, runningMark(pid, time.Now()))

	_, _, err := runOrgCmdStreams(t, "watch", "--stop", "--state-dir", stateDir)
	if err == nil {
		t.Fatal("--stop of a watch that does not stop succeeded; want exit 1")
	}
	for _, want := range []string{
		"did not stop within 300ms",
		"the stop request stays in " + org.WatchStopPathIn(stateDir),
		fmt.Sprintf("kill %d (the pid in %s", pid, org.WatchAlivePathIn(stateDir)),
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}
	if _, err := org.ReadWatchStopRequest(stateDir); err != nil {
		t.Errorf("stop request after the deadline: %v; want it left in place", err)
	}
	select {
	case <-bystanderDone:
		t.Error("the process the mark names has exited; --stop must send no signal")
	default:
	}
}

// waitForFileContains waits up to 10s until path holds substr count times.
func waitForFileContains(t *testing.T, path, substr string, count int) string {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		data, _ := os.ReadFile(path)
		if strings.Count(string(data), substr) >= count {
			return string(data)
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s = %q; want %q %d times", path, data, substr, count)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// startDetachedProcess, the launch of the production orgWatchLaunch, run on
// a shell script instead of ralph: the child leads its own session's
// process group, reads EOF from stdin, appends stdout and stderr to the log
// (created 0644 with its directory), and stop ends it, with SIGKILL after
// the grace when it ignores SIGTERM.
func TestOrgWatchDetach_StartDetachedProcess_NewSessionDevNullStdinAppendedLog(t *testing.T) {
	setOrgWatchDuration(t, &orgWatchChildStopGrace, 200*time.Millisecond)
	dir := t.TempDir()
	script := filepath.Join(dir, "child.sh")
	body := "echo \"out $1\"\necho err >&2\nif read -r line; then echo \"stdin: $line\"; else echo 'stdin: eof'; fi\n" +
		"if [ \"$2\" = ignore-term ]; then trap '' TERM; fi\nexec sleep 30\n"
	if err := os.WriteFile(script, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(dir, "state", org.WatchLogFileName)

	for i, mode := range []string{"", "ignore-term"} {
		p, err := startDetachedProcess("/bin/sh", []string{script, fmt.Sprintf("run-%d", i+1), mode}, logPath)
		if err != nil {
			t.Fatalf("startDetachedProcess: %v", err)
		}
		waitForFileContains(t, logPath, "stdin: eof", i+1)
		if pgid, err := syscall.Getpgid(p.pid); err != nil || pgid != p.pid {
			t.Errorf("child %d process group = %d (%v); want its own (%d) from setsid", i+1, pgid, err, p.pid)
		}
		start := time.Now()
		p.stop()
		select {
		case <-p.exited:
		default:
			t.Errorf("child %d: exited is not closed after stop", i+1)
		}
		if elapsed := time.Since(start); mode == "ignore-term" && elapsed < 200*time.Millisecond {
			t.Errorf("a child that ignores SIGTERM ended after %v; want SIGKILL after the 200ms grace", elapsed)
		}
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if want := "out run-1\nerr\nstdin: eof\nout run-2\nerr\nstdin: eof\n"; string(data) != want {
		t.Errorf("log = %q; want both runs appended: %q", data, want)
	}
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o644 {
		t.Errorf("log mode = %v; want 0644", perm)
	}
}
