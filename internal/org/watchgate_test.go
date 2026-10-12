package org

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// The watch gate (plan 2026-10-12-org-watch-gate, AC1, AC2, AC13 to AC15
// at the org layer): WatchAcceptsWork and the gate NewWatchGate builds.
// The gate positions in Spawn, StartFeature and Send are in
// watchgate_positions_test.go.

// holdWatchWithMark takes watch.lock in dir for the rest of the test and
// writes mark, as a watch in that state would.
func holdWatchWithMark(t *testing.T, dir string, mark WatchAlive) {
	t.Helper()
	mustAcquireWatchLock(t, dir)
	mustWriteWatchAlive(t, dir, mark)
}

// onceMark is the running mark of a --once watch.
func onceMark(pid int, at time.Time) WatchAlive {
	mark := runningMark(pid, at)
	mark.Once = true
	return mark
}

// gateErr checks that err is a refusal of the gate on dir in state, and
// that its message holds every string of want.
func gateErr(t *testing.T, err error, dir, state string, want ...string) {
	t.Helper()
	if !errors.Is(err, ErrNoRunningWatch) {
		t.Fatalf("gate = %v; want an error that is ErrNoRunningWatch", err)
	}
	var gerr *WatchGateError
	if !errors.As(err, &gerr) || gerr.Liveness.State != state || gerr.StateDir != dir {
		t.Fatalf("gate = %#v; want a *WatchGateError on %s in state %q", err, dir, state)
	}
	for _, w := range append([]string{dir}, want...) {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("gate error %q does not say %q", err, w)
		}
	}
}

func TestWatchAcceptsWork_OnlyARunningWatchThatIsNotOnce(t *testing.T) {
	running := runningMark(7, watchAliveT0)
	once := onceMark(7, watchAliveT0)
	stopping := stoppingMark(7, watchAliveT0)
	cases := []struct {
		name string
		l    WatchLiveness
		want bool
	}{
		{name: "running", l: WatchLiveness{State: WatchLivenessRunning, Mark: &running}, want: true},
		{name: "running --once", l: WatchLiveness{State: WatchLivenessRunning, Mark: &once}},
		{name: "running with no mark", l: WatchLiveness{State: WatchLivenessRunning}},
		{name: "stopping", l: WatchLiveness{State: WatchLivenessStopping, Mark: &stopping}},
		{name: "stale", l: WatchLiveness{State: WatchLivenessStale, Mark: &running}},
		{name: "not running", l: WatchLiveness{State: WatchLivenessNotRunning, Mark: &running}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := WatchAcceptsWork(tc.l); got != tc.want {
				t.Errorf("WatchAcceptsWork(%+v) = %t; want %t", tc.l, got, tc.want)
			}
		})
	}
}

// The gate passes a long-running watch and refuses every other state, each
// with the next step: start a watch (not_running), stop or kill the one that
// holds the lock (stale), or wait for the watch to end (stopping, --once).
// Only stale is waited out, for settle; the other refusals come at once.
func TestNewWatchGate_PassesOnlyAWatchThatAcceptsWork(t *testing.T) {
	const settle = 300 * time.Millisecond
	cases := []struct {
		name     string
		setup    func(t *testing.T, dir string)
		wantPass bool
		state    string
		want     []string
		waits    bool
	}{
		{
			name:     "running",
			setup:    func(t *testing.T, dir string) { holdWatchWithMark(t, dir, runningMark(4242, time.Now())) },
			wantPass: true,
		},
		{
			name:  "running --once",
			setup: func(t *testing.T, dir string) { holdWatchWithMark(t, dir, onceMark(4242, time.Now())) },
			state: WatchLivenessRunning,
			want:  []string{"pid 4242", "is a --once run", "accepts no new work", "run this again once it has ended", "ralph org start then starts a long-running watch itself", "ralph org watch --all --detach"},
		},
		{
			name:  "stopping",
			setup: func(t *testing.T, dir string) { holdWatchWithMark(t, dir, stoppingMark(4242, time.Now())) },
			state: WatchLivenessStopping,
			want:  []string{"pid 4242", "is stopping", "accepts no new work", "run this again once it has ended", "ralph org start then starts a long-running watch itself", "ralph org watch --all --detach"},
		},
		{
			name: "stale with an old mark",
			setup: func(t *testing.T, dir string) {
				holdWatchWithMark(t, dir, runningMark(4242, time.Now().Add(-time.Hour)))
			},
			state: WatchLivenessStale,
			want:  []string{"has not refreshed its liveness mark (pid 4242)", "ralph org watch --stop", "kill 4242 (the pid in ", WatchAliveFileName + "; check that it is still a ralph org watch first)"},
			waits: true,
		},
		{
			name:  "stale with no mark",
			setup: func(t *testing.T, dir string) { mustAcquireWatchLock(t, dir) },
			state: WatchLivenessStale,
			want:  []string{"(pid unknown)", "ralph org watch --stop", "find the process that holds ", WatchLockFileName + " (", WatchAliveFileName + " names no pid) and kill it"},
			waits: true,
		},
		{
			name:  "not running",
			setup: func(t *testing.T, dir string) { mustWriteWatchAlive(t, dir, runningMark(4242, time.Now())) },
			state: WatchLivenessNotRunning,
			want:  []string{"no ralph org watch --all is running", "ralph org watch --all --detach", "or, for a new org, start it with ralph org start, which starts one"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tc.setup(t, dir)
			start := time.Now()
			err := NewWatchGate(dir, "", nil, 10, settle)()
			elapsed := time.Since(start)
			if tc.wantPass {
				if err != nil {
					t.Fatalf("gate = %v; want it to pass", err)
				}
				return
			}
			gateErr(t, err, dir, tc.state, tc.want...)
			if tc.waits && elapsed < settle {
				t.Errorf("a stale watch was refused after %v; want the gate to wait out the settle of %v first", elapsed, settle)
			}
			if !tc.waits && elapsed >= settle {
				t.Errorf("a %s watch was refused after %v; want at once, without the settle of %v", tc.state, elapsed, settle)
			}
		})
	}
}

// Plan 2026-10-12-org-watch-gate F10 and AC15: another watch may hold the
// lock and not have written its first mark yet. A gate that finds it stale
// probes again, and passes once the mark says running within settle; when
// the mark says stopping instead, it refuses at once, without waiting out
// the rest of settle.
func TestNewWatchGate_AStaleWatchThatWritesItsMarkWithinTheSettle(t *testing.T) {
	const settle = 5 * time.Second
	cases := []struct {
		name     string
		mark     func() WatchAlive
		wantPass bool
	}{
		{name: "running", mark: func() WatchAlive { return runningMark(4242, time.Now()) }, wantPass: true},
		{name: "stopping", mark: func() WatchAlive { return stoppingMark(4242, time.Now()) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			mustAcquireWatchLock(t, dir)
			written := make(chan error, 1)
			go func() {
				time.Sleep(200 * time.Millisecond)
				written <- writeWatchAlive(dir, tc.mark())
			}()
			start := time.Now()
			err := NewWatchGate(dir, "", nil, 10, settle)()
			elapsed := time.Since(start)
			if werr := <-written; werr != nil {
				t.Fatal(werr)
			}
			if elapsed < 150*time.Millisecond || elapsed >= settle {
				t.Errorf("the gate returned after %v; want it to wait for the mark written after 200ms, and not the whole settle of %v", elapsed, settle)
			}
			if tc.wantPass {
				if err != nil {
					t.Fatalf("gate = %v; want it to pass once the mark says running", err)
				}
				return
			}
			gateErr(t, err, dir, WatchLivenessStopping, "is stopping")
		})
	}
}

// The gate reads the time from now: a running mark is fresh for a clock at
// its ts and stale for one an hour on, whatever the real time is.
func TestNewWatchGate_JudgesTheMarkAtTheTimeOfNow(t *testing.T) {
	dir := t.TempDir()
	holdWatchWithMark(t, dir, runningMark(4242, watchAliveT0))
	if err := NewWatchGate(dir, "", func() time.Time { return watchAliveT0 }, 10, 0)(); err != nil {
		t.Fatalf("gate at the mark's ts = %v; want it to pass", err)
	}
	later := func() time.Time { return watchAliveT0.Add(time.Hour) }
	gateErr(t, NewWatchGate(dir, "", later, 10, 0)(), dir, WatchLivenessStale)
}

// The `ralph org watch` commands a refusal names repeat HintStateDir as
// --state-dir, quoted as InboxCommand quotes it, in every state, and an empty
// HintStateDir repeats none. NewWatchGate passes its hintStateDir on to the
// refusal.
func TestWatchGateError_TheCommandsRepeatHintStateDir(t *testing.T) {
	const hint = "/led ger"
	mark := runningMark(4242, watchAliveT0)
	once := onceMark(4242, watchAliveT0)
	stopping := stoppingMark(4242, watchAliveT0)
	detach := "ralph org watch --all --detach --state-dir '/led ger'"
	cases := []struct {
		name string
		l    WatchLiveness
		want []string
	}{
		{name: "not running", l: WatchLiveness{State: WatchLivenessNotRunning}, want: []string{"start one with " + detach + ", "}},
		{name: "stale", l: WatchLiveness{State: WatchLivenessStale, Mark: &mark}, want: []string{"stop it with ralph org watch --stop --state-dir '/led ger', "}},
		{name: "stopping", l: WatchLiveness{State: WatchLivenessStopping, Mark: &stopping}, want: []string{"start one with " + detach + ")"}},
		{name: "running --once", l: WatchLiveness{State: WatchLivenessRunning, Mark: &once}, want: []string{"start one with " + detach + ")"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			with := (&WatchGateError{StateDir: "/ledger", HintStateDir: hint, Liveness: tc.l}).Error()
			for _, w := range tc.want {
				if !strings.Contains(with, w) {
					t.Errorf("error with HintStateDir %q = %q; want it to say %q", hint, with, w)
				}
			}
			without := (&WatchGateError{StateDir: "/ledger", Liveness: tc.l}).Error()
			if strings.Contains(without, "--state-dir") {
				t.Errorf("error with no HintStateDir = %q; want no --state-dir", without)
			}
			if want := strings.ReplaceAll(with, " --state-dir '/led ger'", ""); without != want {
				t.Errorf("error with no HintStateDir:\n got %q\nwant %q", without, want)
			}
		})
	}

	dir := t.TempDir()
	err := NewWatchGate(dir, hint, nil, 10, 0)()
	gateErr(t, err, dir, WatchLivenessNotRunning, detach)
	var gerr *WatchGateError
	if errors.As(err, &gerr) && gerr.HintStateDir != hint {
		t.Errorf("HintStateDir = %q; want %q", gerr.HintStateDir, hint)
	}
}
