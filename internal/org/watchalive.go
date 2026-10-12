package org

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// One `ralph org watch --all` runs per org state dir (RunWatchAll,
// watch.go). It holds watch.lock for its lifetime and rewrites
// watch-alive.json, the liveness mark, before its first cycle, after every
// cycle, and, saying stopping, every interval while its caller's work ends
// (WatchHooks.BeforeStop). ProbeWatch reads both to tell whether a watch is
// running, and WatchAcceptsWork whether it accepts new work (watchgate.go).
// `ralph org watch --stop` sends no signal: it writes watch-stop.json, which
// the watch reads between cycles (a signal could reach an unrelated process
// that reuses the pid of a mark not yet rewritten).

// File names of the watch, within the org state dir.
const (
	WatchLockFileName  = "watch.lock"
	WatchAliveFileName = "watch-alive.json"
	WatchStopFileName  = "watch-stop.json"
	// WatchLogFileName is the file `ralph org watch --all --detach` sends the
	// detached watch's stdout and stderr to.
	WatchLogFileName = "watch.log"
	// watchStopTakenFileName is where a starting watch moves the stop request
	// to read it (takeWatchStopRequest). Only the holder of watch.lock uses
	// it.
	watchStopTakenFileName = "." + WatchStopFileName + ".taken"
)

// States of the liveness mark (WatchAlive.State).
const (
	WatchAliveRunning = "running"
	// WatchAliveStopping is the mark of a watch that has run its last cycle
	// and still holds watch.lock while its caller's work ends
	// (WatchHooks.BeforeStop); it writes stopped next.
	WatchAliveStopping = "stopping"
	WatchAliveStopped  = "stopped"
)

// States ProbeWatch returns (WatchLiveness.State).
const (
	// WatchLivenessRunning: watch.lock is held and the mark says running
	// with a ts within the window of ProbeWatch.
	WatchLivenessRunning = "running"
	// WatchLivenessStopping: watch.lock is held and the mark says stopping
	// with a ts within the window of ProbeWatch. The watch runs no more
	// cycles, so it accepts no new work (WatchAcceptsWork).
	WatchLivenessStopping = "stopping"
	// WatchLivenessStale: watch.lock is held, but the mark is missing, does
	// not parse, says stopped, or says running or stopping and is older than
	// the window of ProbeWatch. The holder may be hung, or may have taken the
	// lock and not written its mark yet.
	WatchLivenessStale = "stale"
	// WatchLivenessNotRunning: no one holds watch.lock, whatever the mark
	// says (a killed watch leaves a running mark behind).
	WatchLivenessNotRunning = "not_running"
)

// DefaultWatchLockWait is how long AcquireWatchLock waits for watch.lock
// when its caller passes no wait: long enough to outlast ProbeWatch's
// momentary shared lock, short enough that a second watch fails at once. A
// watch refused the lock lives that long, which the CLI waits out after it
// launched one (launchAndAwaitOrgWatch).
const DefaultWatchLockWait = 2 * time.Second

// ErrWatchRunning is wrapped by AcquireWatchLock's error when another watch
// holds watch.lock.
var ErrWatchRunning = errors.New("another ralph org watch --all is running on this state dir")

// WatchAlive is the liveness mark, watch-alive.json. StartedAt carries
// nanoseconds because a stop request counts only when it is later than
// StartedAt, and a `--stop` right after a start falls in the same second.
// TS is the last time the watch wrote its running or stopping mark: running
// at the start and after each cycle, then stopping as it starts to drain
// its watchers (WatchHooks.BeforeStop) and every interval until they end, so
// it is not always a cycle's end; the stopped mark keeps it. TS and
// StoppedAt are whole seconds; all are UTC RFC3339.
type WatchAlive struct {
	State           string `json:"state"`
	PID             int    `json:"pid"`
	StartedAt       string `json:"started_at"`
	TS              string `json:"ts"`
	IntervalSeconds int    `json:"interval_seconds"`
	// DeadmanMinutes is the watch's own [org].deadman_minutes as configured,
	// below 1 included, so ProbeWatch judges the mark by the window of the
	// watch that wrote it rather than by its reader's ralph.toml. It is nil
	// in a mark written before the field was added; ProbeWatch then uses the
	// value its caller passes.
	DeadmanMinutes *int `json:"deadman_minutes,omitempty"`
	// Once is true for a run that ends after its cycles
	// (WatchAllParams.Cycles > 0; `--once` is 1). Such a watch accepts no
	// new work even while it runs (WatchAcceptsWork).
	Once      bool   `json:"once,omitempty"`
	StoppedAt string `json:"stopped_at,omitempty"`
}

// WatchLiveness is what ProbeWatch found: State is one of the
// WatchLiveness* constants, and Mark is the mark it read (nil when the mark
// is missing or does not parse).
type WatchLiveness struct {
	State string
	Mark  *WatchAlive
}

// WatchStopRequest is the stop request, watch-stop.json. RequestedAt is
// UTC RFC3339 with nanoseconds (see WatchAlive).
type WatchStopRequest struct {
	RequestedAt string `json:"requested_at"`
}

// WatchLockPathIn returns the watch.lock path within an already-resolved
// org state dir, mirroring EscalationsPathIn; WatchAlivePathIn and
// WatchStopPathIn do the same for the mark and the stop request.
func WatchLockPathIn(stateDir string) string {
	return filepath.Join(stateDir, WatchLockFileName)
}

// WatchAlivePathIn returns the watch-alive.json path (see WatchLockPathIn).
func WatchAlivePathIn(stateDir string) string {
	return filepath.Join(stateDir, WatchAliveFileName)
}

// WatchStopPathIn returns the watch-stop.json path (see WatchLockPathIn).
func WatchStopPathIn(stateDir string) string {
	return filepath.Join(stateDir, WatchStopFileName)
}

// WatchLock is watch.lock held by this process (AcquireWatchLock) until
// Release.
type WatchLock struct {
	held *heldFileLock
}

// AcquireWatchLock takes watch.lock in stateDir and keeps it until Release,
// waiting up to wait for it (DefaultWatchLockWait when wait <= 0). When
// another watch keeps holding it, the error wraps ErrWatchRunning and names
// the pid of the mark, or says the pid is unknown when the mark cannot be
// read.
func AcquireWatchLock(stateDir string, wait time.Duration) (*WatchLock, error) {
	if wait <= 0 {
		wait = DefaultWatchLockWait
	}
	held, err := acquireHeldFileLock(stateDir, WatchLockFileName, "watch", wait)
	if errors.Is(err, errFileLockHeld) {
		alivePath := WatchAlivePathIn(stateDir)
		mark, readErr := ReadWatchAlive(stateDir)
		if readErr != nil {
			return nil, fmt.Errorf("org: watch: %w (its pid is unknown: %v)", ErrWatchRunning, readErr)
		}
		return nil, fmt.Errorf("org: watch: %w (pid %d, from %s)", ErrWatchRunning, mark.PID, alivePath)
	}
	if err != nil {
		return nil, err
	}
	return &WatchLock{held: held}, nil
}

// Release drops watch.lock.
func (l *WatchLock) Release() error {
	return l.held.release()
}

// ReadWatchAlive reads the mark in stateDir. A missing mark is an error
// that wraps fs.ErrNotExist.
func ReadWatchAlive(stateDir string) (*WatchAlive, error) {
	path := WatchAlivePathIn(stateDir)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("org: watch: read %s: %w", path, err)
	}
	var mark WatchAlive
	if err := json.Unmarshal(data, &mark); err != nil {
		return nil, fmt.Errorf("org: watch: parse %s: %w", path, err)
	}
	return &mark, nil
}

// writeWatchAlive replaces the mark in stateDir (writeJSONFileAtomic).
func writeWatchAlive(stateDir string, mark WatchAlive) error {
	return writeJSONFileAtomic(WatchAlivePathIn(stateDir), mark)
}

// ProbeWatch tells whether a watch is running on stateDir at now (see the
// WatchLiveness* constants). It checks watch.lock with fileLockHeld, then
// reads the mark. A lock that cannot be checked counts as held: not_running
// says no watch holds the lock, which this probe could not tell, and
// running still needs a fresh running mark.
//
// A running or stopping mark is fresh when its ts is no older than the
// window watchAliveWindow returns: the mark's own deadman_minutes when it
// carries one, else deadmanMinutes, or the default deadman_minutes
// (config.Default, 10) when that value is below 1, and never shorter than
// two of the mark's own interval_seconds. A deadman_minutes of 0 means the
// deadman never escalates by timeout (checkDeadman), not that no watch is
// running, and a watch whose interval is longer than deadman_minutes
// rewrites its mark only once per interval, so it is not stale between two
// cycles. A fresh running mark is running and a fresh stopping mark is
// stopping.
func ProbeWatch(stateDir string, now time.Time, deadmanMinutes int) WatchLiveness {
	held, err := fileLockHeld(WatchLockPathIn(stateDir))
	mark, _ := ReadWatchAlive(stateDir)
	if err == nil && !held {
		return WatchLiveness{State: WatchLivenessNotRunning, Mark: mark}
	}
	if mark == nil || (mark.State != WatchAliveRunning && mark.State != WatchAliveStopping) {
		return WatchLiveness{State: WatchLivenessStale, Mark: mark}
	}
	if mark.DeadmanMinutes != nil {
		deadmanMinutes = *mark.DeadmanMinutes
	}
	ts, err := time.Parse(time.RFC3339, mark.TS)
	if err != nil || now.Sub(ts) > watchAliveWindow(deadmanMinutes, mark.IntervalSeconds) {
		return WatchLiveness{State: WatchLivenessStale, Mark: mark}
	}
	if mark.State == WatchAliveStopping {
		return WatchLiveness{State: WatchLivenessStopping, Mark: mark}
	}
	return WatchLiveness{State: WatchLivenessRunning, Mark: mark}
}

// watchAliveWindow is how old a running or stopping mark may be and still
// count as fresh (see ProbeWatch).
func watchAliveWindow(deadmanMinutes, intervalSeconds int) time.Duration {
	if deadmanMinutes < 1 {
		deadmanMinutes = config.Default().Org.DeadmanMinutes
	}
	return max(time.Duration(deadmanMinutes)*time.Minute, 2*time.Duration(intervalSeconds)*time.Second)
}

// WriteWatchStopRequest replaces the stop request in stateDir with one
// requested at at.
func WriteWatchStopRequest(stateDir string, at time.Time) error {
	return writeJSONFileAtomic(WatchStopPathIn(stateDir), WatchStopRequest{RequestedAt: at.UTC().Format(time.RFC3339Nano)})
}

// ReadWatchStopRequest reads the stop request in stateDir. A missing
// request is an error that wraps fs.ErrNotExist.
func ReadWatchStopRequest(stateDir string) (*WatchStopRequest, error) {
	return readWatchStopRequestAt(WatchStopPathIn(stateDir))
}

// takeWatchStopRequest moves the stop request in stateDir to
// watchStopTakenFileName with one rename and reads it there, so a request a
// `--stop` writes after the rename is a new watch-stop.json that the caller
// does not touch when it removes what it took. taken is the path the
// request was moved to, which the caller removes or renames back; it is ""
// when nothing was moved, and then err wraps fs.ErrNotExist when there was
// no request. An err with taken set is the read of the moved request (it
// cannot be read or parsed).
func takeWatchStopRequest(stateDir string) (req *WatchStopRequest, taken string, err error) {
	taken = filepath.Join(stateDir, watchStopTakenFileName)
	if err := os.Rename(WatchStopPathIn(stateDir), taken); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, "", err
		}
		return nil, "", fmt.Errorf("org: watch: take the stop request: %w", err)
	}
	req, err = readWatchStopRequestAt(taken)
	return req, taken, err
}

// readWatchStopRequestAt reads a stop request at path (ReadWatchStopRequest).
func readWatchStopRequestAt(path string) (*WatchStopRequest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("org: watch: read %s: %w", path, err)
	}
	var req WatchStopRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("org: watch: parse %s: %w", path, err)
	}
	return &req, nil
}

// RemoveWatchStopRequest removes the stop request in stateDir. A missing
// request is not an error.
func RemoveWatchStopRequest(stateDir string) error {
	err := os.Remove(WatchStopPathIn(stateDir))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("org: watch: remove the stop request: %w", err)
	}
	return nil
}

// AppliesTo reports whether the request is for a watch that started at
// startedAt: it was requested strictly later. A request whose requested_at
// does not parse applies to no watch.
func (r *WatchStopRequest) AppliesTo(startedAt time.Time) bool {
	at, err := time.Parse(time.RFC3339, r.RequestedAt)
	return err == nil && at.After(startedAt)
}

// writeJSONFileAtomic writes v as JSON to a temporary file in path's
// directory and renames it over path, so a reader sees the old content or
// the new, never a part. The temporary file is removed when any step
// fails.
func writeJSONFileAtomic(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("org: marshal %T: %w", v, err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("org: create dir for %s: %w", path, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("org: write %s: %w", path, err)
	}
	tmpPath := tmp.Name()
	_, err = tmp.Write(append(data, '\n'))
	if err == nil {
		err = tmp.Chmod(0o644)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmpPath, path)
	}
	if err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("org: write %s: %w", path, err)
	}
	return nil
}
