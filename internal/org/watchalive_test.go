package org

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

// watchAliveT0 is the clock of the watchalive tests.
var watchAliveT0 = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

func mustAcquireWatchLock(t *testing.T, dir string) *WatchLock {
	t.Helper()
	lock, err := AcquireWatchLock(dir, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("AcquireWatchLock: %v", err)
	}
	t.Cleanup(func() { _ = lock.Release() })
	return lock
}

func mustWriteWatchAlive(t *testing.T, dir string, mark WatchAlive) {
	t.Helper()
	if err := writeWatchAlive(dir, mark); err != nil {
		t.Fatalf("writeWatchAlive: %v", err)
	}
}

// runningMark is a running mark of pid with ts at.
func runningMark(pid int, at time.Time) WatchAlive {
	return WatchAlive{
		State: WatchAliveRunning, PID: pid,
		StartedAt: watchAliveT0.Add(-time.Hour).Format(time.RFC3339Nano),
		TS:        at.UTC().Format(time.RFC3339), IntervalSeconds: 30,
	}
}

// dirEntries returns the names in dir, sorted.
func dirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// AC9 at the lock layer: watch.lock excludes a second holder in the same
// process, whether through AcquireWatchLock or a bare flock on another open
// of the file, and the refusal names the pid of the mark.
func TestAcquireWatchLock_ExcludesASecondHolderAndNamesThePIDOfTheMark(t *testing.T) {
	dir := t.TempDir()
	first, err := AcquireWatchLock(dir, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("first AcquireWatchLock: %v", err)
	}

	f, err := os.Open(WatchLockPathIn(dir))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); !errors.Is(err, syscall.EWOULDBLOCK) {
		t.Fatalf("flock on a second open of watch.lock = %v; want EWOULDBLOCK while the first holds it", err)
	}

	_, err = AcquireWatchLock(dir, 50*time.Millisecond)
	if !errors.Is(err, ErrWatchRunning) || !strings.Contains(err.Error(), "its pid is unknown") {
		t.Fatalf("second AcquireWatchLock with no mark = %v; want ErrWatchRunning, pid unknown", err)
	}
	mustWriteWatchAlive(t, dir, runningMark(4242, watchAliveT0))
	_, err = AcquireWatchLock(dir, 50*time.Millisecond)
	if !errors.Is(err, ErrWatchRunning) || !strings.Contains(err.Error(), "pid 4242") {
		t.Fatalf("second AcquireWatchLock = %v; want ErrWatchRunning naming pid 4242", err)
	}

	if err := first.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	again, err := AcquireWatchLock(dir, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("AcquireWatchLock after Release: %v", err)
	}
	if err := again.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
}

// ProbeWatch's momentary shared lock must not make a starting watch fail:
// AcquireWatchLock waits it out. With a wait shorter than the shared hold,
// it is refused, which shows the wait is what lets it through.
func TestAcquireWatchLock_WaitsOutAShortSharedLock(t *testing.T) {
	dir := t.TempDir()
	lockPath := WatchLockPathIn(dir)
	if err := os.WriteFile(lockPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	shared, err := os.Open(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = shared.Close() }()
	if err := syscall.Flock(int(shared.Fd()), syscall.LOCK_SH|syscall.LOCK_NB); err != nil {
		t.Fatalf("take the shared lock: %v", err)
	}

	if _, err := AcquireWatchLock(dir, 30*time.Millisecond); !errors.Is(err, ErrWatchRunning) {
		t.Fatalf("AcquireWatchLock under a shared lock with a 30ms wait = %v; want ErrWatchRunning", err)
	}

	released := make(chan struct{})
	go func() {
		time.Sleep(150 * time.Millisecond)
		_ = syscall.Flock(int(shared.Fd()), syscall.LOCK_UN)
		close(released)
	}()
	lock, err := AcquireWatchLock(dir, 2*time.Second)
	if err != nil {
		t.Fatalf("AcquireWatchLock while a shared lock is dropped within its wait = %v; want it to wait and succeed", err)
	}
	<-released
	if err := lock.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
}

// AC11: the three states, and every reason for stale.
func TestProbeWatch_TellsRunningStaleAndNotRunningApart(t *testing.T) {
	const deadman = 10
	now := watchAliveT0
	cases := []struct {
		name   string
		locked bool
		// released leaves watch.lock behind from a lock taken and released.
		released bool
		mark     string // raw watch-alive.json; "" writes none
		deadman  int
		want     string
		wantMark bool
	}{
		{name: "locked, fresh running mark", locked: true, mark: markJSON(t, runningMark(7, now.Add(-time.Minute))), deadman: deadman, want: WatchLivenessRunning, wantMark: true},
		{name: "locked, running mark exactly deadman_minutes old", locked: true, mark: markJSON(t, runningMark(7, now.Add(-deadman*time.Minute))), deadman: deadman, want: WatchLivenessRunning, wantMark: true},
		{name: "locked, no mark", locked: true, deadman: deadman, want: WatchLivenessStale},
		{name: "locked, mark is not JSON", locked: true, mark: "{not json", deadman: deadman, want: WatchLivenessStale},
		{name: "locked, stopped mark", locked: true, mark: markJSON(t, WatchAlive{State: WatchAliveStopped, PID: 7, TS: now.Format(time.RFC3339), StoppedAt: now.Format(time.RFC3339)}), deadman: deadman, want: WatchLivenessStale, wantMark: true},
		{name: "locked, running mark older than deadman_minutes", locked: true, mark: markJSON(t, runningMark(7, now.Add(-deadman*time.Minute-time.Second))), deadman: deadman, want: WatchLivenessStale, wantMark: true},
		{name: "locked, running mark with a ts that does not parse", locked: true, mark: `{"state":"running","pid":7,"ts":"yesterday"}`, deadman: deadman, want: WatchLivenessStale, wantMark: true},
		{name: "locked, deadman_minutes 0", locked: true, mark: markJSON(t, runningMark(7, now)), deadman: 0, want: WatchLivenessStale, wantMark: true},
		{name: "locked, deadman_minutes negative", locked: true, mark: markJSON(t, runningMark(7, now)), deadman: -5, want: WatchLivenessStale, wantMark: true},
		{name: "not locked, fresh running mark (a killed watch)", released: true, mark: markJSON(t, runningMark(7, now)), deadman: deadman, want: WatchLivenessNotRunning, wantMark: true},
		{name: "not locked, no mark", released: true, deadman: deadman, want: WatchLivenessNotRunning},
		{name: "no watch.lock, fresh running mark", mark: markJSON(t, runningMark(7, now)), deadman: deadman, want: WatchLivenessNotRunning, wantMark: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.locked || tc.released {
				lock := mustAcquireWatchLock(t, dir)
				if tc.released {
					if err := lock.Release(); err != nil {
						t.Fatal(err)
					}
				}
			}
			if tc.mark != "" {
				if err := os.WriteFile(WatchAlivePathIn(dir), []byte(tc.mark), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got := ProbeWatch(dir, now, tc.deadman)
			if got.State != tc.want {
				t.Errorf("State = %q; want %q", got.State, tc.want)
			}
			if (got.Mark != nil) != tc.wantMark {
				t.Errorf("Mark = %+v; want a mark: %t", got.Mark, tc.wantMark)
			}
		})
	}
}

// A probe on a state dir with no watch.lock does not create one, and leaves
// a held lock held.
func TestProbeWatch_CreatesNoLockAndLeavesAHolderHolding(t *testing.T) {
	dir := t.TempDir()
	if got := ProbeWatch(dir, watchAliveT0, 10).State; got != WatchLivenessNotRunning {
		t.Fatalf("State = %q; want not_running", got)
	}
	if _, err := os.Stat(WatchLockPathIn(dir)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("watch.lock after a probe: %v; want it not created", err)
	}

	mustAcquireWatchLock(t, dir)
	mustWriteWatchAlive(t, dir, runningMark(7, watchAliveT0))
	for i := range 3 {
		if got := ProbeWatch(dir, watchAliveT0, 10).State; got != WatchLivenessRunning {
			t.Fatalf("probe %d: State = %q; want running", i+1, got)
		}
	}
	if _, err := AcquireWatchLock(dir, 20*time.Millisecond); !errors.Is(err, ErrWatchRunning) {
		t.Fatalf("AcquireWatchLock after the probes = %v; want the first holder to still hold it", err)
	}
}

func markJSON(t *testing.T, mark WatchAlive) string {
	t.Helper()
	dir := t.TempDir()
	mustWriteWatchAlive(t, dir, mark)
	data, err := os.ReadFile(WatchAlivePathIn(dir))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// The mark is replaced through a temporary file and a rename: after many
// writes only watch-alive.json is left, holding the last mark, and a write
// that cannot rename leaves no temporary file either.
func TestWriteWatchAlive_ReplacesThroughARenameAndLeavesNoTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	for i := range 5 {
		mustWriteWatchAlive(t, dir, runningMark(100+i, watchAliveT0.Add(time.Duration(i)*time.Second)))
	}
	if got := dirEntries(t, dir); !slices.Equal(got, []string{WatchAliveFileName}) {
		t.Fatalf("state dir = %v; want only %s", got, WatchAliveFileName)
	}
	mark, err := ReadWatchAlive(dir)
	if err != nil {
		t.Fatalf("ReadWatchAlive: %v", err)
	}
	if want := runningMark(104, watchAliveT0.Add(4*time.Second)); *mark != want {
		t.Fatalf("mark = %+v; want %+v", *mark, want)
	}
	info, err := os.Stat(WatchAlivePathIn(dir))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o644 {
		t.Errorf("mark mode = %v; want 0644", perm)
	}

	blocked := t.TempDir()
	if err := os.MkdirAll(filepath.Join(WatchAlivePathIn(blocked), "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeWatchAlive(blocked, runningMark(1, watchAliveT0)); err == nil {
		t.Fatal("writeWatchAlive over a non-empty directory succeeded; want the rename to fail")
	}
	if got := dirEntries(t, blocked); !slices.Equal(got, []string{WatchAliveFileName}) {
		t.Fatalf("state dir after a failed write = %v; want no temporary file left", got)
	}
}

func TestReadWatchAlive_MissingAndCorrupt(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadWatchAlive(dir); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadWatchAlive with no mark = %v; want fs.ErrNotExist", err)
	}
	if err := os.WriteFile(WatchAlivePathIn(dir), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if mark, err := ReadWatchAlive(dir); err == nil || mark != nil {
		t.Fatalf("ReadWatchAlive of a corrupt mark = %+v, %v; want nil and an error", mark, err)
	}
}

// AC13b at the request layer: a stop request counts only for a watch that
// started strictly before it, to the nanosecond.
func TestWatchStopRequest_AppliesOnlyToAWatchStartedBeforeIt(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadWatchStopRequest(dir); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadWatchStopRequest with none = %v; want fs.ErrNotExist", err)
	}
	if err := RemoveWatchStopRequest(dir); err != nil {
		t.Fatalf("RemoveWatchStopRequest with none = %v; want nil", err)
	}

	at := watchAliveT0.Add(500 * time.Millisecond)
	if err := WriteWatchStopRequest(dir, at); err != nil {
		t.Fatalf("WriteWatchStopRequest: %v", err)
	}
	req, err := ReadWatchStopRequest(dir)
	if err != nil {
		t.Fatalf("ReadWatchStopRequest: %v", err)
	}
	for _, c := range []struct {
		name    string
		started time.Time
		want    bool
	}{
		{"a watch started earlier in the same second", watchAliveT0, true},
		{"a watch started a nanosecond earlier", at.Add(-time.Nanosecond), true},
		{"a watch started at the same instant", at, false},
		{"a watch started later", at.Add(time.Second), false},
	} {
		if got := req.AppliesTo(c.started); got != c.want {
			t.Errorf("%s: AppliesTo = %t; want %t", c.name, got, c.want)
		}
	}
	if (&WatchStopRequest{RequestedAt: "soon"}).AppliesTo(watchAliveT0) {
		t.Error("a request whose requested_at does not parse applies to a watch")
	}

	if got := dirEntries(t, dir); !slices.Equal(got, []string{WatchStopFileName}) {
		t.Fatalf("state dir = %v; want only %s", got, WatchStopFileName)
	}
	if err := RemoveWatchStopRequest(dir); err != nil {
		t.Fatalf("RemoveWatchStopRequest: %v", err)
	}
	if _, err := ReadWatchStopRequest(dir); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadWatchStopRequest after remove = %v; want fs.ErrNotExist", err)
	}
}
