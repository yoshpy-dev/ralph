package org

import (
	"errors"
	"fmt"
	"time"
)

// The watch gate (plan 2026-10-12-org-watch-gate): a spawn, a start, and a
// TASK send begin autonomous work, so they are refused while no cross-org
// watch (`ralph org watch --all`) accepts work on the ledger. Org.WatchGate
// is the check. Spawn calls it right after its input checks, StartFeature
// before it makes the worktree, and Send for a TASK only, each before it
// writes to the manifest or calls herdr or agmsg, dry-run included; a
// refusal is the error alone, with no manifest event and no receipt (F1).

// DefaultWatchGateSettle is how long the gate the CLI builds (NewWatchGate)
// waits for a stale watch to become one that accepts work.
const DefaultWatchGateSettle = 2 * time.Second

// watchGateProbeInterval is how often NewWatchGate's gate probes again while
// it waits out a stale watch.
const watchGateProbeInterval = 100 * time.Millisecond

// ErrNoRunningWatch is what every *WatchGateError is (errors.Is).
var ErrNoRunningWatch = errors.New("no ralph org watch --all accepts work on this state dir")

// WatchGateError is a refusal of the watch gate: Liveness is what ProbeWatch
// last found on StateDir. Its message names that state and what to do next.
type WatchGateError struct {
	StateDir string
	Liveness WatchLiveness
}

// Is makes errors.Is(err, ErrNoRunningWatch) true for a *WatchGateError.
func (e *WatchGateError) Is(target error) bool {
	return target == ErrNoRunningWatch
}

func (e *WatchGateError) Error() string {
	mark := e.Liveness.Mark
	const later = "run this again once it has ended and a long-running watch is up (ralph org watch --all --detach starts one)"
	switch e.Liveness.State {
	case WatchLivenessNotRunning:
		return fmt.Sprintf("org: no ralph org watch --all is running on the org state dir %s, so no new work starts; "+
			"start one with ralph org watch --all --detach, or start the org with ralph org start, which starts one", e.StateDir)
	case WatchLivenessStale:
		kill := fmt.Sprintf("find the process that holds %s and kill it", WatchLockPathIn(e.StateDir))
		if mark != nil && mark.PID > 0 {
			kill = fmt.Sprintf("check that pid %d is a ralph org watch and kill it", mark.PID)
		}
		return fmt.Sprintf("org: the ralph org watch --all that holds the lock of the org state dir %s has not refreshed its liveness mark (%s), "+
			"so no new work starts; stop it with ralph org watch --stop, or %s", e.StateDir, watchGatePIDText(mark), kill)
	case WatchLivenessStopping:
		return fmt.Sprintf("org: the ralph org watch --all on the org state dir %s (%s) is stopping and accepts no new work; %s",
			e.StateDir, watchGatePIDText(mark), later)
	case WatchLivenessRunning:
		return fmt.Sprintf("org: the ralph org watch --all on the org state dir %s (%s) is a --once run, which ends after its cycle and accepts no new work; %s",
			e.StateDir, watchGatePIDText(mark), later)
	}
	return fmt.Sprintf("org: the ralph org watch --all on the org state dir %s is %q and accepts no new work", e.StateDir, e.Liveness.State)
}

// watchGatePIDText names the pid of a liveness mark in a WatchGateError.
func watchGatePIDText(mark *WatchAlive) string {
	if mark == nil || mark.PID <= 0 {
		return "pid unknown"
	}
	return fmt.Sprintf("pid %d", mark.PID)
}

// WatchAcceptsWork reports whether the watch ProbeWatch found accepts new
// work: it is running and is not a run that ends after its cycles
// (WatchAlive.Once). A stopping watch runs no more cycles, and a stale or
// not running one watches nothing.
func WatchAcceptsWork(l WatchLiveness) bool {
	return l.State == WatchLivenessRunning && l.Mark != nil && !l.Mark.Once
}

// NewWatchGate returns an Org.WatchGate on the org state dir stateDir: it
// probes the watch (ProbeWatch at now(), with deadmanMinutes for a mark that
// carries no deadman_minutes) and passes when it accepts work
// (WatchAcceptsWork). A stale watch may be one that has taken watch.lock and
// not written its first mark yet, so for stale the gate probes again every
// watchGateProbeInterval until settle has passed, and passes once the watch
// accepts work (plan 2026-10-12-org-watch-gate, F10). Every other state is
// refused at once, as is stale after settle; the refusal is a
// *WatchGateError. The settle reads the real clock; a nil now means
// time.Now.
func NewWatchGate(stateDir string, now func() time.Time, deadmanMinutes int, settle time.Duration) func() error {
	if now == nil {
		now = time.Now
	}
	return func() error {
		deadline := time.Now().Add(settle)
		for {
			l := ProbeWatch(stateDir, now(), deadmanMinutes)
			if WatchAcceptsWork(l) {
				return nil
			}
			left := time.Until(deadline)
			if l.State != WatchLivenessStale || left <= 0 {
				return &WatchGateError{StateDir: stateDir, Liveness: l}
			}
			time.Sleep(min(watchGateProbeInterval, left))
		}
	}
}

// checkWatchGate runs o.WatchGate; a nil WatchGate checks nothing.
func (o *Org) checkWatchGate() error {
	if o.WatchGate == nil {
		return nil
	}
	return o.WatchGate()
}
