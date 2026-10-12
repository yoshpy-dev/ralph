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

func setOrgWatchLaunch(t *testing.T, launch func(req orgWatchLaunchRequest) (*orgWatchProcess, error)) {
	t.Helper()
	orig := orgWatchLaunch
	orgWatchLaunch = launch
	t.Cleanup(func() { orgWatchLaunch = orig })
}

// refuseOrgWatchLaunch sets a launcher that fails the test when called.
func refuseOrgWatchLaunch(t *testing.T) {
	t.Helper()
	setOrgWatchLaunch(t, func(req orgWatchLaunchRequest) (*orgWatchProcess, error) {
		t.Errorf("a watch was launched (args %q); want nothing started", req.args)
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
// another watch holds exits 1 naming that watch's pid, prints no banner
// (plan 2026-10-12-org-watch-gate AC9, tech-debt 204 (e)), and leaves the
// mark as it was; once the lock is free, the next one runs and prints its
// banner.
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

	stdout, _, err := runOrgCmdStreams(t, "watch", "--all", "--once", "--state-dir", stateDir)
	if !errors.Is(err, org.ErrWatchRunning) || !strings.Contains(err.Error(), "pid 4242") {
		t.Fatalf("second watch = %v; want ErrWatchRunning naming pid 4242", err)
	}
	if stdout != "" {
		t.Errorf("stdout of the refused watch = %q; want nothing, not the banner", stdout)
	}
	if markAfter, err := os.ReadFile(org.WatchAlivePathIn(stateDir)); err != nil || string(markAfter) != string(markBefore) {
		t.Errorf("the refused watch changed the mark: %s -> %s (%v)", markBefore, markAfter, err)
	}

	release()
	stdout, _, err = runOrgCmdStreams(t, "watch", "--all", "--once", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("watch after the lock was freed: %v", err)
	}
	if want := "watching every org (interval="; !strings.HasPrefix(stdout, want) {
		t.Errorf("stdout of the watch that took the lock = %q; want the banner %q...", stdout, want)
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
// with the pid when the mark names one. Since plan 2026-10-12-org-watch-gate
// (F10, AC4) it first waits out the deadline for the mark to turn running,
// shortened here.
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
			setOrgWatchDuration(t, &orgWatchEnsureDeadline, 300*time.Millisecond)
			setOrgWatchDuration(t, &orgWatchProbeInterval, 10*time.Millisecond)
			stateDir := filepath.Join(t.TempDir(), "state")
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatal(err)
			}
			holdWatchLock(t, stateDir, tc.mark)

			start := time.Now()
			_, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir)
			if err == nil {
				t.Fatal("--detach with a stale watch succeeded; want exit 1")
			}
			if elapsed := time.Since(start); elapsed < 300*time.Millisecond {
				t.Errorf("--detach refused the stale watch after %v; want it to wait the 300ms deadline first", elapsed)
			}
			for _, want := range append([]string{"so no new watch runs", "ralph org watch --stop"}, tc.want...) {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
		})
	}
}

// inProcessLauncher is an orgWatchLaunch that runs RunWatchAll on stateDir
// in a goroutine of this process, so the child's mark names os.Getpid(),
// and stops it by cancelling its context. It appends each request it was
// given to got.
func inProcessLauncher(t *testing.T, stateDir string, got *[]orgWatchLaunchRequest) func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
	return func(req orgWatchLaunchRequest) (*orgWatchProcess, error) {
		*got = append(*got, req)
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
			var got []orgWatchLaunchRequest
			setOrgWatchLaunch(t, inProcessLauncher(t, stateDir, &got))

			args := append([]string{"watch", "--all", "--detach", "--state-dir", stateDir}, tc.extra...)
			stdout, _, err := runOrgCmdStreams(t, args...)
			if err != nil {
				t.Fatalf("--detach: %v", err)
			}
			logPath := filepath.Join(stateDir, org.WatchLogFileName)
			if want := fmt.Sprintf("started ralph org watch --all (pid %d); log: %s\n", os.Getpid(), logPath); stdout != want {
				t.Errorf("stdout = %q; want %q", stdout, want)
			}
			if len(got) != 1 {
				t.Fatalf("launches = %d; want 1", len(got))
			}
			if wantArgs := append([]string{"org", "watch", "--all", "--state-dir", stateDir}, tc.wantTail...); !slices.Equal(got[0].args, wantArgs) {
				t.Errorf("child args = %q; want %q", got[0].args, wantArgs)
			}
			if got[0].logPath != logPath {
				t.Errorf("child log = %q; want %q", got[0].logPath, logPath)
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
	setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
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

// failOrgWatchLaunch sets a launcher that fails with "start /x/ralph: boom",
// as launchOrgWatch does when the binary cannot be started.
func failOrgWatchLaunch(t *testing.T) {
	t.Helper()
	setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
		return nil, errors.New("start /x/ralph: boom")
	})
}

// A launch that fails (mutation survivor E8 of /test cycle 1 of
// org-watch-gate) is --detach's error after its prefix, whole.
func TestOrgWatchDetach_ALaunchThatFails_IsTheErrorAfterTheDetachPrefix(t *testing.T) {
	failOrgWatchLaunch(t)
	stateDir := t.TempDir()
	_, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir)
	if want := "org: watch --detach: start /x/ralph: boom"; err == nil || err.Error() != want {
		t.Fatalf("--detach with a launch that fails = %v; want %q", err, want)
	}
}

// --detach says `started` only for a running mark of its own child's pid
// (mutation survivor X3). Two --detach on a fresh ledger: the other one's
// watch takes watch.lock and writes its running mark, and this one's child,
// refused the lock, exits. This --detach must not take that mark for its
// child: it says the other watch is already running, with the other pid,
// and the other watch keeps running. Plan 2026-10-12-org-watch-gate AC5
// (tech-debt 206) changed the exit code from 1 to 0: a watch accepts work.
func TestOrgWatchDetach_AWatchStartedByAnotherProcessIsNotTakenForTheChild(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
		holdWatchLock(t, stateDir, runningMark(4343, time.Now()))
		exited := make(chan struct{})
		close(exited)
		return &orgWatchProcess{pid: 4242, exited: exited, stop: func() {}}, nil
	})
	stdout, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("--detach while another watch runs = %v (stdout %q); want exit 0", err, stdout)
	}
	if want := fmt.Sprintf("ralph org watch --all is already running (pid 4343); log: %s\n", filepath.Join(stateDir, org.WatchLogFileName)); stdout != want {
		t.Errorf("stdout = %q; want %q, not a started line for the child that exited", stdout, want)
	}
	if got := org.ProbeWatch(stateDir, time.Now(), 10); got.State != org.WatchLivenessRunning || got.Mark == nil || got.Mark.PID != 4343 {
		t.Errorf("ProbeWatch after --detach = %+v; want the other watch (pid 4343) running", got)
	}
}

// Tech-debt 206, the other order: this --detach's child exits, refused the
// lock, before the watch that won watch.lock has written its first mark, so
// the ledger is stale for 200ms after the exit. --detach waits on for that
// mark instead of failing for the exited child, and says the other watch is
// already running.
func TestOrgWatchDetach_AChildThatExitsBeforeTheOtherWatchsFirstMark_IsAlreadyRunning(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	var written <-chan error
	setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
		holdWatchLock(t, stateDir, nil)
		written = writeWatchMarkLater(t, stateDir, runningMark(4344, time.Now()), 200*time.Millisecond)
		exited := make(chan struct{})
		close(exited)
		return &orgWatchProcess{pid: 4242, exited: exited, stop: func() { t.Error("the child that exited was stopped") }}, nil
	})
	stdout, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir)
	if written == nil {
		t.Fatalf("--detach launched no watch: %v (stdout %q)", err, stdout)
	}
	if werr := <-written; werr != nil {
		t.Fatal(werr)
	}
	if err != nil {
		t.Fatalf("--detach whose child exited before the other watch's first mark = %v (stdout %q); want exit 0", err, stdout)
	}
	if want := fmt.Sprintf("ralph org watch --all is already running (pid 4344); log: %s\n", filepath.Join(stateDir, org.WatchLogFileName)); stdout != want {
		t.Errorf("stdout = %q; want %q", stdout, want)
	}
}

// This --detach's child exits, refused watch.lock by a watch that took it
// first and accepts no work: one that is stopping, a --once run, or stale
// until the deadline (shortened). --detach exits 1 with its refusal for that
// watch, as when the first probe finds it, not with the child's exit
// (self-review L1 of org-watch-gate).
func TestOrgWatchDetach_AChildThatExitsWhileAWatchThatAcceptsNoWorkHoldsTheLock_IsRefused(t *testing.T) {
	cases := []struct {
		name string
		mark func() *org.WatchAlive
		want string
	}{
		{name: "stopping", mark: func() *org.WatchAlive { return stoppingWatchMark(4305, time.Now()) },
			want: "(pid 4305) is stopping, so no new watch runs; run ralph org watch --all --detach --state-dir "},
		{name: "running --once", mark: func() *org.WatchAlive { return onceWatchMark(4306, time.Now()) },
			want: "(pid 4306) is a --once run, which ends after its cycle, so no new watch runs"},
		{name: "stale until the deadline", mark: func() *org.WatchAlive { return nil },
			want: "but its liveness mark is not fresh (pid unknown), so no new watch runs; stop it with ralph org watch --stop"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setOrgWatchDuration(t, &orgWatchEnsureDeadline, 300*time.Millisecond)
			setOrgWatchDuration(t, &orgWatchProbeInterval, 10*time.Millisecond)
			stateDir := t.TempDir()
			launches := 0
			setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
				launches++
				holdWatchLock(t, stateDir, tc.mark())
				exited := make(chan struct{})
				close(exited)
				return &orgWatchProcess{pid: 4242, exited: exited, stop: func() { t.Error("the child that exited was stopped") }}, nil
			})

			stdout, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir)
			if err == nil {
				t.Fatalf("--detach with a %s watch that holds the lock succeeded (stdout %q); want exit 1", tc.name, stdout)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err, tc.want)
			}
			if strings.Contains(err.Error(), "exited before it was running") {
				t.Errorf("error = %q; want the refusal of the watch that holds the lock, not the child's exit", err)
			}
			if launches != 1 {
				t.Errorf("launches = %d; want 1", launches)
			}
		})
	}
}

// The finding of /test cycle 1 of org-watch-gate (eight concurrent --detach,
// then a --stop that timed out): another watch took watch.lock and runs,
// and this --detach's child, refused the lock, is still alive in its wait
// for it (org.DefaultWatchLockWait); a fake child here exits 300ms after its
// launch. --detach says the other watch is already running, exit 0, only
// once its child has exited, so a --stop right after it returns leaves no
// child that takes the lock after the stop. It does not stop a child that
// exits on its own.
func TestOrgWatchDetach_AnotherWatchRuns_ReturnsOnlyOnceItsChildHasExited(t *testing.T) {
	setOrgWatchDuration(t, &orgWatchChildExitWait, 3*time.Second)
	setOrgWatchDuration(t, &orgWatchProbeInterval, 10*time.Millisecond)
	stateDir := t.TempDir()
	var exited chan struct{}
	setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
		holdWatchLock(t, stateDir, runningMark(4345, time.Now()))
		ch := make(chan struct{})
		go func() { time.Sleep(300 * time.Millisecond); close(ch) }()
		exited = ch
		return &orgWatchProcess{pid: 4242, exited: ch, stop: func() { t.Error("the child that exits on its own was stopped") }}, nil
	})

	start := time.Now()
	stdout, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir)
	elapsed := time.Since(start)
	if exited == nil {
		t.Fatalf("--detach launched no watch: %v (stdout %q)", err, stdout)
	}
	select {
	case <-exited:
	default:
		t.Errorf("--detach returned after %v while its child was still alive; want it to return once the child has exited (300ms)", elapsed)
	}
	if err != nil {
		t.Fatalf("--detach while another watch runs = %v (stdout %q); want exit 0", err, stdout)
	}
	if want := fmt.Sprintf("ralph org watch --all is already running (pid 4345); log: %s\n", filepath.Join(stateDir, org.WatchLogFileName)); stdout != want {
		t.Errorf("stdout = %q; want %q", stdout, want)
	}
	if elapsed >= 3*time.Second {
		t.Errorf("--detach returned after %v; want soon after the child exited, before orgWatchChildExitWait (3s)", elapsed)
	}
}

// The wait of the test above is bounded: a child that has not exited
// orgWatchChildExitWait (shortened) after another watch was found accepting
// work is stopped and reaped, once, and --detach says the other watch is
// already running, exit 0. The other watch keeps running.
func TestOrgWatchDetach_AnotherWatchRuns_AChildThatDoesNotExitInTimeIsStopped(t *testing.T) {
	setOrgWatchDuration(t, &orgWatchChildExitWait, 200*time.Millisecond)
	setOrgWatchDuration(t, &orgWatchProbeInterval, 10*time.Millisecond)
	stateDir := t.TempDir()
	stops := 0
	setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
		holdWatchLock(t, stateDir, runningMark(4346, time.Now()))
		exited := make(chan struct{})
		return &orgWatchProcess{pid: 4242, exited: exited, stop: func() {
			stops++
			if stops == 1 {
				close(exited)
			}
		}}, nil
	})

	start := time.Now()
	stdout, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("--detach while another watch runs = %v (stdout %q); want exit 0", err, stdout)
	}
	if want := fmt.Sprintf("ralph org watch --all is already running (pid 4346); log: %s\n", filepath.Join(stateDir, org.WatchLogFileName)); stdout != want {
		t.Errorf("stdout = %q; want %q", stdout, want)
	}
	if stops != 1 {
		t.Errorf("the child was stopped %d times; want once", stops)
	}
	if elapsed < 200*time.Millisecond {
		t.Errorf("--detach stopped the child after %v; want it to wait orgWatchChildExitWait (200ms) first", elapsed)
	}
	if got := org.ProbeWatch(stateDir, time.Now(), 10); got.State != org.WatchLivenessRunning || got.Mark == nil || got.Mark.PID != 4346 {
		t.Errorf("ProbeWatch after --detach = %+v; want the other watch (pid 4346) running", got)
	}
}

// Self-review C2-1 of org-watch-gate: the stop of a child still alive at the
// end of orgWatchChildExitWait can take the rest of its lock wait, and the
// watch that was running may change meanwhile, so --detach judges by a
// probe after the stop, not by the one before it. Here the child's stop
// changes the ledger before the child exits: the other watch (pid 4348)
// ends, turns stopping, or hands the lock to another watch (pid 4349).
// --detach says `already running` only for a watch that accepts work after
// the stop, with that watch's pid; otherwise it exits 1.
func TestOrgWatchDetach_AnotherWatchRuns_AStoppedChildIsJudgedByAProbeAfterTheStop(t *testing.T) {
	cases := []struct {
		name string
		// during is what happens to the other watch while the child stops;
		// release frees that watch's lock.
		during     func(t *testing.T, stateDir string, release func())
		wantStdout string
		wantErr    []string
	}{
		{name: "the other watch ended",
			during:  func(t *testing.T, stateDir string, release func()) { release() },
			wantErr: []string{"org: watch --detach: the watch (pid 4242) was stopped while another watch held ", "and that one has ended too, so no watch runs; see "}},
		{name: "the other watch is stopping",
			during: func(t *testing.T, stateDir string, release func()) {
				writeWatchMark(t, stateDir, stoppingWatchMark(4348, time.Now()))
			},
			wantErr: []string{"(pid 4348) is stopping, so no new watch runs"}},
		{name: "another watch took the lock",
			during: func(t *testing.T, stateDir string, release func()) {
				release()
				holdWatchLock(t, stateDir, runningMark(4349, time.Now()))
			},
			wantStdout: "ralph org watch --all is already running (pid 4349); log: "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setOrgWatchDuration(t, &orgWatchChildExitWait, 100*time.Millisecond)
			setOrgWatchDuration(t, &orgWatchProbeInterval, 10*time.Millisecond)
			stateDir := t.TempDir()
			stops := 0
			setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
				release := holdWatchLock(t, stateDir, runningMark(4348, time.Now()))
				exited := make(chan struct{})
				return &orgWatchProcess{pid: 4242, exited: exited, stop: func() {
					stops++
					if stops == 1 {
						tc.during(t, stateDir, release)
						close(exited)
					}
				}}, nil
			})

			stdout, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir)
			if stops != 1 {
				t.Errorf("the child was stopped %d times; want once", stops)
			}
			if tc.wantStdout != "" {
				if err != nil {
					t.Fatalf("--detach = %v (stdout %q); want exit 0", err, stdout)
				}
				if want := tc.wantStdout + filepath.Join(stateDir, org.WatchLogFileName) + "\n"; stdout != want {
					t.Errorf("stdout = %q; want %q", stdout, want)
				}
				return
			}
			if err == nil {
				t.Fatalf("--detach succeeded (stdout %q); want exit 1, not already running for the watch seen before the stop", stdout)
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			for _, want := range tc.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
		})
	}
}

// While --detach waits for its child to exit, the other watch ends and the
// child takes watch.lock and writes its running mark (here the mark of the
// held lock is rewritten with the child's pid 200ms after the launch).
// --detach says it started the child, exit 0, and leaves it running.
func TestOrgWatchDetach_AnotherWatchRuns_AChildThatTakesTheLockMeanwhileIsStarted(t *testing.T) {
	setOrgWatchDuration(t, &orgWatchChildExitWait, 3*time.Second)
	setOrgWatchDuration(t, &orgWatchProbeInterval, 10*time.Millisecond)
	stateDir := t.TempDir()
	var written <-chan error
	setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
		holdWatchLock(t, stateDir, runningMark(4347, time.Now()))
		written = writeWatchMarkLater(t, stateDir, runningMark(4242, time.Now()), 200*time.Millisecond)
		return &orgWatchProcess{pid: 4242, exited: make(chan struct{}), stop: func() { t.Error("the child that took the lock was stopped") }}, nil
	})

	start := time.Now()
	stdout, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir)
	elapsed := time.Since(start)
	if written == nil {
		t.Fatalf("--detach launched no watch: %v (stdout %q)", err, stdout)
	}
	if werr := <-written; werr != nil {
		t.Fatal(werr)
	}
	if err != nil {
		t.Fatalf("--detach whose child took the lock = %v (stdout %q); want exit 0", err, stdout)
	}
	if want := fmt.Sprintf("started ralph org watch --all (pid 4242); log: %s\n", filepath.Join(stateDir, org.WatchLogFileName)); stdout != want {
		t.Errorf("stdout = %q; want %q", stdout, want)
	}
	if elapsed < 150*time.Millisecond {
		t.Errorf("--detach returned after %v; want it to wait for its child rather than return at the other watch's mark", elapsed)
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
			setOrgWatchDuration(t, &orgWatchEnsureDeadline, 300*time.Millisecond)
			setOrgWatchDuration(t, &orgWatchReleaseDeadline, 100*time.Millisecond)
			setOrgWatchDuration(t, &orgWatchProbeInterval, 10*time.Millisecond)
			stateDir := filepath.Join(t.TempDir(), "state")
			var stops int
			setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
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

// Plan 2026-10-12-org-watch-gate AC13 and AC14: a watch that holds
// watch.lock and accepts no work because it is stopping or a --once run
// ends soon, so --detach launches nothing, does not wait for it, and exits 1
// saying to run it again once that watch has ended.
func TestOrgWatchDetach_AStoppingOrOnceWatch_StartsNothing(t *testing.T) {
	cases := []struct {
		name string
		mark *org.WatchAlive
		want string
	}{
		{name: "stopping", mark: stoppingWatchMark(4303, time.Now()), want: "(pid 4303) is stopping, so no new watch runs"},
		{name: "running --once", mark: onceWatchMark(4304, time.Now()), want: "(pid 4304) is a --once run, which ends after its cycle, so no new watch runs"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refuseOrgWatchLaunch(t)
			setOrgWatchDuration(t, &orgWatchEnsureDeadline, 3*time.Second)
			stateDir := t.TempDir()
			holdWatchLock(t, stateDir, tc.mark)

			start := time.Now()
			stdout, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir)
			if err == nil {
				t.Fatalf("--detach with a %s watch succeeded (stdout %q); want exit 1", tc.name, stdout)
			}
			for _, want := range []string{tc.want, "the watch that holds " + org.WatchLockPathIn(stateDir), "run ralph org watch --all --detach --state-dir '" + stateDir + "' again once it has ended"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
			if elapsed := time.Since(start); elapsed >= time.Second {
				t.Errorf("--detach refused the %s watch after %v; want at once, not after a wait", tc.name, elapsed)
			}
		})
	}
}

// keepLockOrgWatchLaunch sets a launcher whose child, with pid, takes
// watch.lock, never writes a running mark, and keeps the lock after its
// stop (until the test's cleanup), as a watch that ignores its SIGTERM would.
func keepLockOrgWatchLaunch(t *testing.T, stateDir string, pid int) {
	t.Helper()
	setOrgWatchLaunch(t, func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
		holdWatchLock(t, stateDir, nil)
		exited := make(chan struct{})
		var once sync.Once
		return &orgWatchProcess{pid: pid, exited: exited, stop: func() { once.Do(func() { close(exited) }) }}, nil
	})
}

// Self-review C2-3 of org-watch-gate: the commands the errors of --detach
// name (`ralph org watch --stop` for a stale watch, --detach again for a
// stopping one, and `ralph status` after a stopped child that keeps
// watch.lock) repeat --state-dir, quoted, when it was given, as the gate's
// refusal does (TestOrgWatchGate_TheCommandsRepeatAnExplicitStateDir), and
// name none for a ledger found without it (here RALPH_ORG_STATE_DIR).
func TestOrgWatchDetach_TheCommandsRepeatAnExplicitStateDir(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, stateDir string)
		// want is the command as the error names it, with %[1]s for the
		// --state-dir part (" --state-dir '<dir>'", or "") and %[2]s for
		// watch.log.
		want string
	}{
		{name: "stale", want: "stop it with ralph org watch --stop%[1]s, or kill 4360",
			setup: func(t *testing.T, stateDir string) {
				refuseOrgWatchLaunch(t)
				holdWatchLock(t, stateDir, runningMark(4360, time.Now().Add(-time.Hour)))
			}},
		{name: "stopping", want: "run ralph org watch --all --detach%[1]s again once it has ended",
			setup: func(t *testing.T, stateDir string) {
				refuseOrgWatchLaunch(t)
				holdWatchLock(t, stateDir, stoppingWatchMark(4361, time.Now()))
			}},
		{name: "a stopped child that keeps the lock", want: ".lock is still held; see %[2]s and ralph status%[1]s",
			setup: func(t *testing.T, stateDir string) {
				setOrgWatchDuration(t, &orgWatchReleaseDeadline, 50*time.Millisecond)
				keepLockOrgWatchLaunch(t, stateDir, 4362)
			}},
	}
	for _, tc := range cases {
		for _, given := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s, --state-dir given %t", tc.name, given), func(t *testing.T) {
				shortenOrgWatchDeadline(t)
				stateDir := filepath.Join(t.TempDir(), "led ger")
				tc.setup(t, stateDir)
				args, part := []string{"watch", "--all", "--detach"}, ""
				t.Setenv(org.EnvOrgStateDir, "")
				if given {
					args = append(args, "--state-dir", stateDir)
					part = " --state-dir '" + stateDir + "'"
				} else {
					t.Setenv(org.EnvOrgStateDir, stateDir)
				}

				stdout, _, err := runOrgCmdStreams(t, args...)
				want := fmt.Sprintf(tc.want, part, filepath.Join(stateDir, org.WatchLogFileName))
				if err == nil || !strings.Contains(err.Error(), want) {
					t.Fatalf("--detach = %v (stdout %q); want an error naming %q", err, stdout, want)
				}
				if !given && strings.Contains(err.Error(), "--state-dir") {
					t.Errorf("error with the ledger from %s = %q; want no --state-dir", org.EnvOrgStateDir, err)
				}
			})
		}
	}
}

// Plan 2026-10-12-org-watch-gate AC15 for --detach: a watch that holds
// watch.lock and writes its first running mark 200ms later (stale until
// then) is waited for, and --detach says it is already running, launches
// nothing, and exits 0.
func TestOrgWatchDetach_AWatchThatWritesItsFirstMarkLate_IsAlreadyRunning(t *testing.T) {
	refuseOrgWatchLaunch(t)
	stateDir := t.TempDir()
	holdWatchLock(t, stateDir, nil)
	written := writeWatchMarkLater(t, stateDir, runningMark(4350, time.Now()), 200*time.Millisecond)

	start := time.Now()
	stdout, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir)
	elapsed := time.Since(start)
	if werr := <-written; werr != nil {
		t.Fatal(werr)
	}
	if err != nil {
		t.Fatalf("--detach while a watch writes its first mark: %v; want exit 0", err)
	}
	if want := fmt.Sprintf("ralph org watch --all is already running (pid 4350); log: %s\n", filepath.Join(stateDir, org.WatchLogFileName)); stdout != want {
		t.Errorf("stdout = %q; want %q", stdout, want)
	}
	if elapsed < 150*time.Millisecond {
		t.Errorf("--detach returned after %v; want it to wait for the mark written after 200ms", elapsed)
	}
}

// Plan 2026-10-12-org-watch-gate F10 for --detach (mutation survivor E4 of
// /test cycle 1): the holder of watch.lock is stale (no mark) and releases
// the lock 200ms later, during ensureOrgWatch's wait. --detach then launches
// the watch once and says it started, rather than refusing a watch that has
// ended.
func TestOrgWatchDetach_AStaleHolderThatEndsDuringTheWait_LaunchesTheWatch(t *testing.T) {
	setOrgWatchDuration(t, &orgWatchEnsureDeadline, 3*time.Second)
	setOrgWatchDuration(t, &orgWatchProbeInterval, 10*time.Millisecond)
	stateDir := t.TempDir()
	release := holdWatchLock(t, stateDir, nil)
	go func() { time.Sleep(200 * time.Millisecond); release() }()
	var got []orgWatchLaunchRequest
	setOrgWatchLaunch(t, inProcessLauncher(t, stateDir, &got))

	start := time.Now()
	stdout, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("--detach: %v (stdout %q)", err, stdout)
	}
	if len(got) != 1 {
		t.Fatalf("launches = %d; want 1", len(got))
	}
	if want := fmt.Sprintf("started ralph org watch --all (pid %d); log: %s\n", os.Getpid(), filepath.Join(stateDir, org.WatchLogFileName)); stdout != want {
		t.Errorf("stdout = %q; want %q", stdout, want)
	}
	if elapsed < 150*time.Millisecond || elapsed >= 3*time.Second {
		t.Errorf("--detach returned after %v; want after the holder left (200ms), well before the deadline", elapsed)
	}
}

// Plan 2026-10-12-org-watch-gate AC6 and F6 for --detach: the child runs in
// the main worktree root when the ledger is that root's .harness/state/org
// (found by default or named with --state-dir), and in this process's
// working directory for a ledger outside the repository. A relative
// --config is passed made absolute. The child's environment has none of the
// herdr pane variables, keeps the others, and has PWD set to its working
// directory.
func TestOrgWatchDetach_TheChildRunsInTheMainWorktreeWithoutTheHerdrPaneEnv(t *testing.T) {
	root := newFeatureRepo(t, ".claude/worktrees/\n.harness/\n")
	sub := filepath.Join(root, "pkg", "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(sub, "ralph.toml"), "[org]\n")
	ledger := filepath.Join(root, ".harness", "state", "org")
	outside := filepath.Join(t.TempDir(), "state")
	for _, name := range orgWatchChildDroppedEnv {
		t.Setenv(name, "from-the-leader-pane")
	}
	t.Setenv("RALPH_TEST_CHILD_KEEP", "kept")
	t.Chdir(sub)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		stateDir  string
		extra     []string
		wantDir   string
		wantState string
	}{
		{name: "the default ledger", wantDir: root, wantState: ledger},
		{name: "--state-dir naming the main worktree's ledger", extra: []string{"--state-dir", ledger}, wantDir: root, wantState: ledger},
		{name: "--state-dir outside the repository", extra: []string{"--state-dir", outside}, wantState: outside},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []orgWatchLaunchRequest
			setOrgWatchLaunch(t, inProcessLauncher(t, tc.wantState, &got))
			args := append([]string{"watch", "--all", "--detach", "--config", "ralph.toml"}, tc.extra...)
			if _, _, err := runOrgCmdStreams(t, args...); err != nil {
				t.Fatalf("--detach: %v", err)
			}
			if len(got) != 1 {
				t.Fatalf("launches = %d; want 1", len(got))
			}
			checkOrgWatchChild(t, got[0], tc.wantState, filepath.Join(wd, "ralph.toml"), tc.wantDir)
		})
	}
}

// checkOrgWatchChild checks that req is the watch of the ledger stateDir
// with the absolute --config config, run in dir ("" for this process's
// working directory) with the herdr pane variables dropped,
// RALPH_TEST_CHILD_KEEP kept, and PWD set to dir when there is one.
func checkOrgWatchChild(t *testing.T, req orgWatchLaunchRequest, stateDir, config, dir string) {
	t.Helper()
	if want := []string{"org", "watch", "--all", "--state-dir", stateDir, "--config", config}; !slices.Equal(req.args, want) {
		t.Errorf("child args = %q; want %q", req.args, want)
	}
	if req.dir != dir {
		t.Errorf("child dir = %q; want %q", req.dir, dir)
	}
	var pwd []string
	for _, kv := range req.env {
		name, value, _ := strings.Cut(kv, "=")
		if slices.Contains(orgWatchChildDroppedEnv, name) {
			t.Errorf("child env has %s; want it dropped", kv)
		}
		if name == "PWD" {
			pwd = append(pwd, value)
		}
	}
	if !slices.Contains(req.env, "RALPH_TEST_CHILD_KEEP=kept") {
		t.Errorf("child env does not keep RALPH_TEST_CHILD_KEEP=kept: %q", req.env)
	}
	if dir != "" && !slices.Equal(pwd, []string{dir}) {
		t.Errorf("child PWD = %q; want only %q", pwd, dir)
	}
}

// startDetachedProcess runs the child in the request's dir with exactly the
// request's environment.
func TestOrgWatchDetach_StartDetachedProcess_RunsInDirWithEnv(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(t.TempDir(), org.WatchLogFileName)
	t.Setenv("RALPH_TEST_CHILD_DROPPED", "inherited")
	p, err := startDetachedProcess("/bin/sh", orgWatchLaunchRequest{
		args:    []string{"-c", `echo "dir: $(pwd -P)"; echo "keep: ${RALPH_TEST_CHILD_KEEP-unset}"; echo "dropped: ${RALPH_TEST_CHILD_DROPPED-unset}"`},
		logPath: logPath, dir: dir, env: []string{"RALPH_TEST_CHILD_KEEP=kept"},
	})
	if err != nil {
		t.Fatalf("startDetachedProcess: %v", err)
	}
	select {
	case <-p.exited:
	case <-time.After(10 * time.Second):
		p.stop()
		t.Fatal("the child did not exit within 10s")
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if want := "dir: " + dir + "\nkeep: kept\ndropped: unset\n"; string(data) != want {
		t.Errorf("log = %q; want %q", data, want)
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
		"it stops when its current cycle, or its wait for an in-flight on-demand watcher call, ends",
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
// (created with 0644 less the umask, with its directory), and stop ends it,
// with SIGKILL after the grace when it ignores SIGTERM.
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
		p, err := startDetachedProcess("/bin/sh", orgWatchLaunchRequest{args: []string{script, fmt.Sprintf("run-%d", i+1), mode}, logPath: logPath})
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
	// The log is opened with 0644, which the umask can narrow (self-review
	// L6): no bit beyond 0644, and the owner can read and write it.
	if perm := info.Mode().Perm(); perm&^0o644 != 0 || perm&0o600 != 0o600 {
		t.Errorf("log mode = %v; want no bit beyond 0644 and owner read and write", perm)
	}
}

// Self-review M1: --detach on a ledger whose ralph.toml sets
// deadman_minutes = 0 (the deadman never escalates by timeout) starts a
// healthy child and exits 0 once its mark says running. ProbeWatch used to
// call every mark stale under a deadman below 1, so --detach stopped the
// child after its deadline and failed.
func TestOrgWatchDetach_DeadmanMinutesZero_StartsTheChild(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "ralph.toml")
	if err := os.WriteFile(configPath, []byte("[org]\ndeadman_minutes = 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(t.TempDir(), "state")
	var got []orgWatchLaunchRequest
	setOrgWatchLaunch(t, inProcessLauncher(t, stateDir, &got))

	stdout, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--config", configPath, "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("--detach with deadman_minutes = 0: %v", err)
	}
	if want := fmt.Sprintf("started ralph org watch --all (pid %d); log: %s\n", os.Getpid(), filepath.Join(stateDir, org.WatchLogFileName)); stdout != want {
		t.Errorf("stdout = %q; want %q", stdout, want)
	}
	if got := org.ProbeWatch(stateDir, time.Now(), 0).State; got != org.WatchLivenessRunning {
		t.Errorf("ProbeWatch with deadman 0 after --detach = %q; want running", got)
	}
}

// ledgerWithAMarkWithoutDeadman returns a ledger whose watch.lock is held
// with a running mark of pid, age old, that carries no deadman_minutes (one
// a watch of an older version writes), and a ralph.toml whose [org]
// deadman_minutes is deadman.
func ledgerWithAMarkWithoutDeadman(t *testing.T, pid, deadman int, age time.Duration) (stateDir, configPath string) {
	t.Helper()
	stateDir = t.TempDir()
	configPath = filepath.Join(t.TempDir(), "ralph.toml")
	writeTestFile(t, configPath, fmt.Sprintf("[org]\ndeadman_minutes = %d\n", deadman))
	holdWatchLock(t, stateDir, runningMark(pid, time.Now().Add(-age)))
	return stateDir, configPath
}

// Plan 2026-10-12-org-watch-gate AC7 for --detach (mutation survivor E7c of
// /test cycle 1): ensureOrgWatch judges a mark that carries no
// deadman_minutes by the caller's config, as the gate does. With
// deadman_minutes 30, a 20-minute-old mark is running, so --detach says it
// is already running and launches nothing.
func TestOrgWatchDetach_AMarkWithoutDeadmanMinutes_IsJudgedByTheCallersConfig(t *testing.T) {
	refuseOrgWatchLaunch(t)
	shortenOrgWatchDeadline(t)
	stateDir, configPath := ledgerWithAMarkWithoutDeadman(t, 4520, 30, 20*time.Minute)
	stdout, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir, "--config", configPath)
	if err != nil {
		t.Fatalf("--detach: %v", err)
	}
	if want := fmt.Sprintf("ralph org watch --all is already running (pid 4520); log: %s\n", filepath.Join(stateDir, org.WatchLogFileName)); stdout != want {
		t.Errorf("stdout = %q; want %q", stdout, want)
	}
}

// AC7 for the first probe of ensureOrgWatch alone (mutation survivor E7 of
// /test cycle 1, which judged that probe by 0, so by the default 10): with
// deadman_minutes 5, a 7-minute-old mark without deadman_minutes is stale,
// so --detach waits out the (shortened) deadline and refuses it instead of
// taking it as running.
func TestOrgWatchDetach_TheFirstProbeJudgesAMarkWithoutDeadmanMinutesByTheCallersConfig(t *testing.T) {
	refuseOrgWatchLaunch(t)
	shortenOrgWatchDeadline(t)
	stateDir, configPath := ledgerWithAMarkWithoutDeadman(t, 4530, 5, 7*time.Minute)
	stdout, _, err := runOrgCmdStreams(t, "watch", "--all", "--detach", "--state-dir", stateDir, "--config", configPath)
	if err == nil || !strings.Contains(err.Error(), "but its liveness mark is not fresh (pid 4530)") {
		t.Fatalf("--detach with a mark stale by the caller's deadman = %v (stdout %q); want the stale refusal", err, stdout)
	}
}

// Self-review L1: a long-running watch that a --stop request ends while an
// on-demand watcher call is in flight (a claude that sleeps 30s) cancels
// that call and returns within seconds, instead of keeping the process
// until the call ends. The call starts on the scope change of the seat's
// worktree from the second cycle on.
func TestOrgWatch_AStopRequestCancelsAnInFlightWatcherCall(t *testing.T) {
	setupOrgStubPATH(t)
	started := filepath.Join(t.TempDir(), "claude-started")
	writeClaudeStub(t, "#!/bin/sh\n: > '"+started+"'\nexec sleep 30\n")
	configPath := writeOrgConfigWithExtra(t, t.TempDir(), "[org.watchdog]\nwatcher_enabled = true\nwatcher_model = \"haiku\"\n")
	worktree := t.TempDir()
	runLedgerTestGit(t, "init", "--quiet", worktree)
	stateDir := filepath.Join(t.TempDir(), "state")
	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", worktree,
		"--scope", "test-scope", "--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn: %v", err)
	}

	type result struct {
		stderr string
		err    error
	}
	done := make(chan result, 1)
	go func() {
		_, stderr, err := runOrgCmdStreams(t, "watch", "--all", "--interval-seconds", "1", "--config", configPath, "--state-dir", stateDir)
		done <- result{stderr, err}
	}()
	var res *result
	t.Cleanup(func() {
		if res == nil {
			_ = org.WriteWatchStopRequest(stateDir, time.Now())
			select {
			case <-done:
			case <-time.After(60 * time.Second):
			}
		}
	})

	waitForFileContains(t, filepath.Join(stateDir, "watch-status-org-a.json"), `"cycles": 1`, 1)
	if err := os.WriteFile(filepath.Join(worktree, "new.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(started); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the watcher's claude did not start within 10s of the scope change")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := org.WriteWatchStopRequest(stateDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	stopAt := time.Now()
	select {
	case r := <-done:
		res = &r
	case <-time.After(5 * time.Second):
		t.Fatal("the watch did not return within 5s of the stop request; want the in-flight watcher call cancelled")
	}
	if res.err != nil {
		t.Fatalf("the stopped watch returned %v (stderr %s)", res.err, res.stderr)
	}
	t.Logf("the watch returned %v after the stop request", time.Since(stopAt))
	if want := `watchdog: watcher error for org "org-a" seat "seat-1" condition "scope_change"`; !strings.Contains(res.stderr, want) {
		t.Errorf("stderr = %q; want %q for the cancelled call", res.stderr, want)
	}
	if mark, err := org.ReadWatchAlive(stateDir); err != nil || mark.State != org.WatchAliveStopped {
		t.Errorf("mark = %+v (%v); want stopped", mark, err)
	}
}

// Cross-review cycle 1 #1 on the CLI: a `watch --all --once` whose
// on-demand watcher call is still in flight after its one cycle (self-review
// M-5 lets `--once` wait for it) keeps watch.lock until the call has ended,
// with a fresh mark that says stopping (plan 2026-10-12-org-watch-gate, F9
// and AC13: it said running before) and once. Meanwhile a second
// `watch --all --once` is refused,
// and a `--stop` waits for the watch instead of saying it is not running,
// then reports it stopped. A first `--once` records the seat's worktree
// status, and a new file in the worktree makes the next one see a scope
// change and start the watcher: a claude that waits until the test releases
// it.
func TestOrgWatch_Once_HoldsTheLockUntilItsInFlightWatcherCallEnds(t *testing.T) {
	setupOrgStubPATH(t)
	marks := t.TempDir()
	started := filepath.Join(marks, "claude-started")
	release := filepath.Join(marks, "claude-release")
	writeClaudeStub(t, "#!/bin/sh\n: > '"+started+"'\nwhile [ ! -f '"+release+"' ]; do sleep 0.05; done\nexit 1\n")
	configPath := writeOrgConfigWithExtra(t, t.TempDir(), "[org.watchdog]\nwatcher_enabled = true\nwatcher_model = \"haiku\"\n")
	worktree := t.TempDir()
	runLedgerTestGit(t, "init", "--quiet", worktree)
	stateDir := filepath.Join(t.TempDir(), "state")
	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", worktree,
		"--scope", "test-scope", "--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn: %v", err)
	}
	watchOnce := []string{"watch", "--all", "--once", "--config", configPath, "--state-dir", stateDir}
	if _, stderr, err := runOrgCmdStreams(t, watchOnce...); err != nil {
		t.Fatalf("the first watch --all --once: %v (stderr %s)", err, stderr)
	}
	if _, err := os.Stat(started); err == nil {
		t.Fatal("the first watch --all --once started the watcher; want only the worktree status recorded")
	}
	if err := os.WriteFile(filepath.Join(worktree, "new.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	type result struct {
		stdout, stderr string
		err            error
	}
	run := func(args ...string) <-chan result {
		ch := make(chan result, 1)
		go func() {
			stdout, stderr, err := runOrgCmdStreams(t, args...)
			ch <- result{stdout, stderr, err}
		}()
		return ch
	}
	releaseClaude := func() {
		if err := os.WriteFile(release, nil, 0o644); err != nil {
			t.Error(err)
		}
	}
	// await returns what ch got, failing the test when nothing came within
	// 10s, and drops ch from pending, which the cleanup waits for.
	pending := map[<-chan result]bool{}
	await := func(ch <-chan result, what string) result {
		t.Helper()
		select {
		case r := <-ch:
			delete(pending, ch)
			return r
		case <-time.After(10 * time.Second):
			t.Fatalf("%s did not end within 10s", what)
			return result{}
		}
	}
	t.Cleanup(func() {
		releaseClaude()
		for ch := range pending {
			select {
			case <-ch:
			case <-time.After(60 * time.Second):
			}
		}
	})
	watch := run(watchOnce...)
	pending[watch] = true
	for deadline := time.Now().Add(10 * time.Second); ; {
		if _, err := os.Stat(started); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the watcher's claude did not start within 10s of the scope change")
		}
		time.Sleep(5 * time.Millisecond)
	}

	// The claude may start before the cycle has ended, so wait for the
	// drain's stopping mark rather than probing once.
	waitForWatchState(t, stateDir, org.WatchLivenessStopping, 0)
	if mark, err := org.ReadWatchAlive(stateDir); err != nil || !mark.Once {
		t.Errorf("mark while the watcher call is in flight = %+v (%v); want once", mark, err)
	}
	if _, _, err := runOrgCmdStreams(t, watchOnce...); !errors.Is(err, org.ErrWatchRunning) {
		t.Errorf("a second watch --all --once while the first waits for its watcher call = %v; want ErrWatchRunning", err)
	}
	stop := run("watch", "--stop", "--state-dir", stateDir)
	pending[stop] = true
	// --stop writes its request only after its probe found the watch
	// holding the lock (here stopping).
	for deadline := time.Now().Add(10 * time.Second); ; {
		if _, err := org.ReadWatchStopRequest(stateDir); err == nil {
			break
		}
		if len(stop) > 0 {
			t.Fatalf("--stop while the watcher call is in flight returned %+v before writing a request; want it to wait for the watch", await(stop, "--stop"))
		}
		if time.Now().After(deadline) {
			t.Fatal("--stop wrote no stop request within 10s")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if len(watch) > 0 {
		t.Fatalf("the watch ended while its watcher call was in flight: %+v", await(watch, "the watch"))
	}

	releaseClaude()
	watchRes := await(watch, "the watch")
	stopRes := await(stop, "--stop")
	if watchRes.err != nil {
		t.Fatalf("the watch: %v (stderr %s)", watchRes.err, watchRes.stderr)
	}
	if want := `watchdog: watcher error for org "org-a" seat "seat-1" condition "scope_change"`; !strings.Contains(watchRes.stderr, want) {
		t.Errorf("the watch's stderr = %q; want %q: the call ended before the watch did", watchRes.stderr, want)
	}
	if want := fmt.Sprintf("stopped ralph org watch --all (pid %d)\n", os.Getpid()); stopRes.err != nil || stopRes.stdout != want {
		t.Errorf("--stop = %q, %v; want %q", stopRes.stdout, stopRes.err, want)
	}
	if mark, err := org.ReadWatchAlive(stateDir); err != nil || mark.State != org.WatchAliveStopped {
		t.Errorf("mark = %+v (%v); want stopped", mark, err)
	}
	if got := org.ProbeWatch(stateDir, time.Now(), 10).State; got != org.WatchLivenessNotRunning {
		t.Errorf("ProbeWatch after the watch = %q; want not_running", got)
	}
}
