package org

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// The pane lock around [org.director].wake_cmd (cross-review cycle 2 #3 of
// org-seat-resume; pane_lock.go): the push wake of a headless director in
// herdr runs wake_cmd holding the lock of the director's pane, the one ralph
// org send and ralph org resume take, since wake_cmd can type into that pane
// too, and the wake_cmd of any other registration runs with no lock.
// TestWakeDirector_TakesThePaneLockOfTheDirectorAroundItsTyping in
// director_wake_test.go covers the wake with no wake_cmd, which types into
// the pane itself.

// A headless director in herdr with a wake_cmd: while another holder keeps
// the pane's lock for the whole wait (driverCallTimeout), wake_cmd does not
// run and the wake fails as a failed send does: one banner line, no woken
// event, and the registration held back. Once the lock is free, wake_cmd runs
// with the lock held for the whole run, gets the pane as {target}, and the
// lock is released when it ends. herdr is not called either way.
func TestWakeDirector_AWakeCmdOfADirectorInHerdrRunsUnderItsPaneLock(t *testing.T) {
	f, _, reg := wakeInHerdr(t)
	f.o.Config.Director.WakeCmd = []string{"./wake.sh", "{target}", "{text}"}
	f.o.DriverCallTimeout = 50 * time.Millisecond
	holder, err := f.o.acquirePaneLock(reg.PaneID, 0)
	if err != nil {
		t.Fatalf("hold the pane lock of %s: %v", reg.PaneID, err)
	}
	release := sync.OnceFunc(func() { _ = holder.release() })
	t.Cleanup(release)

	e1 := f.escalate(t, "x")
	res := f.wake()
	reason := fmt.Sprintf("another send or resume to pane %q still held the pane's lock when the wait for it ended, so wake_cmd was not run: ", reg.PaneID)
	if res.Sent || res.Failure == nil || !errors.Is(res.Failure, errFileLockHeld) || !strings.HasPrefix(res.Failure.Error(), reason) {
		t.Fatalf("WakeDirector = %+v, want a failure starting %q", res, reason)
	}
	if runs := f.runs(t); runs != nil {
		t.Fatalf("wake_cmd runs = %+v, want none while another holder keeps the lock", runs)
	}
	if lines := f.bannerLines(); len(lines) != 1 || !strings.HasPrefix(lines[0], "watchdog: director "+reg.ID+" not woken for "+e1+": "+reason) {
		t.Fatalf("banner lines = %q, want one failure line with %q", lines, reason)
	}
	if woken := f.wokenIDs(t); woken != nil {
		t.Fatalf("woken = %q, want none", woken)
	}
	if res := f.wake(); !res.HeldBack || res.Sent {
		t.Fatalf("WakeDirector right after the failure = %+v, want it held back", res)
	}

	release()
	f.now = f.now.Add(directorWakeRetry)
	f.o.DriverCallTimeout = 0
	gate := t.TempDir()
	started, goOn := filepath.Join(gate, "started"), filepath.Join(gate, "go")
	f.script(t, ": > "+shellQuote(started)+"\nwhile [ ! -e "+shellQuote(goOn)+" ]; do "+shellQuote(sleepPath(t))+" 0.02; done")
	letGo := sync.OnceFunc(func() {
		if err := os.WriteFile(goOn, nil, 0o644); err != nil {
			t.Errorf("let wake_cmd go on: %v", err)
		}
	})
	t.Cleanup(letGo)
	woke := make(chan DirectorWakeResult, 1)
	go func() { woke <- f.wake() }()
	waitForResumeTest(t, "the run of wake_cmd", func() bool {
		_, err := os.Stat(started)
		return err == nil
	})
	if held, err := fileLockHeld(paneLockPath(f.o, reg.PaneID)); err != nil || !held {
		t.Fatalf("pane lock of %s held while wake_cmd runs = %v (%v); want it held", reg.PaneID, held, err)
	}
	letGo()
	res = <-woke
	if !res.Sent || !res.Recorded || res.Failure != nil || res.Via != InboxWokenViaWakeCmd || !slices.Equal(res.IDs, []string{e1}) {
		t.Fatalf("WakeDirector once the lock is free = %+v (banner %q), want %s woken via wake_cmd", res, f.banner.String(), e1)
	}
	if runs := f.runs(t); len(runs) != 1 || !slices.Equal(runs[0].args, []string{reg.PaneID, DirectorWakeText([]string{e1})}) {
		t.Fatalf("wake_cmd runs = %+v, want one with the pane %s as {target}", runs, reg.PaneID)
	}
	mustPaneLockFree(t, f.o, reg.PaneID)
	if len(f.h.calls) != 0 {
		t.Fatalf("herdr calls = %q, want none", f.h.calls)
	}
}

// The wake_cmd of a registration with no herdr pane takes no pane lock: a
// promoted session, a headless director in a terminal launch_cmd opened, and
// a headless director in herdr whose pane was never recorded (a hand-edited
// director.jsonl, with a wake_cmd that has no {target}) are each woken by one
// run of wake_cmd, and no pane lock file is made next to the manifest.
func TestWakeDirector_AWakeCmdOfARegistrationWithNoHerdrPaneTakesNoPaneLock(t *testing.T) {
	for _, tc := range []struct {
		name    string
		wakeCmd []string // nil: the fixture's
		setup   func(t *testing.T, f *wakeFixture)
		target  []string // the {target} wake_cmd gets, if any
	}{
		{"a promoted session", nil, func(t *testing.T, f *wakeFixture) { f.attachPush(t, "term-1") }, []string{"term-1"}},
		{"a headless director in a terminal launch_cmd opened", nil, func(t *testing.T, f *wakeFixture) {
			writeDirectorEvents(t, f.store,
				DirectorEvent{TS: directorTestNow, ID: "d1", Event: DirectorEventAttached, Driver: DirectorDriverCodex, Wake: []string{DirectorWakePush},
					Launch: DirectorLaunchCmd, Token: directorTestToken, Model: "m", PermissionMode: PermissionModeGuarded, User: "u", Cwd: "/w"},
				DirectorEvent{TS: directorTestNow, ID: "d1", Event: DirectorEventLaunched, PID: 4242, StartTime: "Thu Oct 15 21:51:09 2026", User: "u", Cwd: "/w"},
			)
		}, []string{"director_" + directorTestToken}},
		{"a headless director in herdr with no pane recorded", []string{"./wake.sh", "{text}"}, func(t *testing.T, f *wakeFixture) {
			writeDirectorEvents(t, f.store,
				DirectorEvent{TS: directorTestNow, ID: "d1", Event: DirectorEventAttached, Driver: DirectorDriverCodex, Wake: []string{DirectorWakePush},
					Launch: DirectorLaunchHerdr, Token: directorTestToken, Model: "m", PermissionMode: PermissionModeGuarded, User: "u", Cwd: "/w"},
				DirectorEvent{TS: directorTestNow, ID: "d1", Event: DirectorEventLaunched, User: "u", Cwd: "/w"},
			)
			if cur := mustReadDirector(t, f.store).Current; cur == nil || !cur.HerdrLaunch() || cur.State != DirectorStateRunning || cur.PaneID != "" {
				t.Fatalf("registration = %+v, want d1 in herdr, running, with no pane", cur)
			}
		}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newWakeFixture(t)
			if tc.wakeCmd != nil {
				f.o.Config.Director.WakeCmd = tc.wakeCmd
			}
			tc.setup(t, f)
			e1 := f.escalate(t, "x")
			res := f.wake()
			if !res.Sent || !res.Recorded || res.Via != InboxWokenViaWakeCmd || res.Failure != nil {
				t.Fatalf("WakeDirector = %+v (banner %q), want sent and recorded via wake_cmd", res, f.banner.String())
			}
			if runs, want := f.runs(t), append(slices.Clone(tc.target), DirectorWakeText([]string{e1})); len(runs) != 1 || !slices.Equal(runs[0].args, want) {
				t.Fatalf("wake_cmd runs = %+v, want one with the args %q", runs, want)
			}
			locks, err := filepath.Glob(filepath.Join(filepath.Dir(f.o.Manifest.Path()), "pane-*.lock"))
			if err != nil || len(locks) != 0 {
				t.Fatalf("pane lock files = %q (%v), want none", locks, err)
			}
		})
	}
}
