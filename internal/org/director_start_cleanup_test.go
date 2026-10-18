package org

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// The tests of start's cleanup of a current headless registration that its
// check finds ended (plan 2026-10-18-org-director-liveness, D6 and AC6), and
// of MarkStoppedAtVersion. herdr is a fake (cleanupHerdr), ps is the fake of
// cleanupFixture.procs, which answers a process the test started itself
// where a wrong SIGTERM could reach it, and every ledger is a t.TempDir.

// cleanupHerdr is a directorHerdr that also runs herdr agent get, as
// checkHerdr does for CheckDirector's tests: every call returns agentGetErr
// (nil: herdr has the agent) and is logged as "agent_get".
type cleanupHerdr struct {
	*directorHerdr
	agentGetErr error
}

func (h *cleanupHerdr) AgentGet(_ context.Context, _ string) (string, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls = append(h.calls, "agent_get")
	if h.agentGetErr != nil {
		return "", h.agentGetErr
	}
	return "status: idle", nil
}

// cleanupNextToken is the launch token of the registration the start under
// test records, d2.
const cleanupNextToken = "ba0000000000"

// cleanupFixture is a dirStartFixture for a start that finds d1, written by
// hand, current: its herdr runs agent get and hands d2 workspace ws-2 and
// pane pane-2; ps answers from procs (a pid with no entry has no process)
// and logs the pids read; the watch gate passes unless gateErr is set and
// counts its calls; the clock is checkNow; and start's notice line goes to
// notice.
type cleanupFixture struct {
	*dirStartFixture
	ch        *cleanupHerdr
	procs     map[int]DirectorProcess
	psReads   []int
	gateErr   error
	gateCalls int
	notice    bytes.Buffer
}

func newCleanupFixture(t *testing.T, events ...DirectorEvent) *cleanupFixture {
	t.Helper()
	f := &cleanupFixture{dirStartFixture: newDirStartFixture(t, cleanupNextToken), procs: map[int]DirectorProcess{}}
	f.ch = &cleanupHerdr{directorHerdr: f.h}
	f.o.Herdr = f.ch
	f.h.workspaceID, f.h.paneID = "ws-2", "pane-2"
	f.o.Now = func() time.Time { return checkNow }
	f.o.DirectorProcessReader = func(_ context.Context, pid int) (DirectorProcess, error) {
		f.psReads = append(f.psReads, pid)
		if p, ok := f.procs[pid]; ok {
			return p, nil
		}
		return DirectorProcess{State: DirectorProcessGone}, nil
	}
	f.o.WatchGate = func() error {
		f.gateCalls++
		return f.gateErr
	}
	if len(events) > 0 {
		writeDirectorEvents(t, f.store, events...)
	}
	return f
}

// herdrHas makes the fake herdr hold paneID in a tab labelled tabLabel in
// workspace wsID labelled wsLabel.
func (f *cleanupFixture) herdrHas(paneID, tabLabel, wsID, wsLabel string) {
	setLabel(&f.h.paneTabs, paneID, fakeTabID(paneID))
	setLabel(&f.h.paneWorkspaces, paneID, wsID)
	setLabel(&f.h.tabLabels, fakeTabID(paneID), tabLabel)
	setLabel(&f.h.workspaceLabels, wsID, wsLabel)
}

// d1HerdrHasItsOwn makes the fake herdr hold d1's pane-1 and ws-1 with the
// labels d1's start gave them.
func (f *cleanupFixture) d1HerdrHasItsOwn() {
	f.herdrHas("pane-1", DirectorTabLabel(directorTestToken), "ws-1", DirectorWorkspaceLabel(directorTestToken))
}

// d1WorkspaceOnly makes the fake herdr hold d1's ws-1, labelled as d1's
// start labelled it, and no pane of d1.
func (f *cleanupFixture) d1WorkspaceOnly() {
	setLabel(&f.h.workspaceLabels, "ws-1", DirectorWorkspaceLabel(directorTestToken))
}

func (f *cleanupFixture) start() (DirectorStartResult, error) {
	p := f.params(DirectorDriverClaude, "opus")
	p.Notice = &f.notice
	return f.o.StartDirector(p)
}

// wantNoClose fails t when herdr was sent C-c or asked to close anything.
func (f *cleanupFixture) wantNoClose(t *testing.T) {
	t.Helper()
	if len(f.h.sendKeysCalls) != 0 || len(f.h.paneCloseCalls) != 0 || len(f.h.workspaceCloseCalls) != 0 {
		t.Fatalf("herdr was sent keys to %q and closes of panes %q and workspaces %q, want none (calls %q)",
			f.h.sendKeysCalls, f.h.paneCloseCalls, f.h.workspaceCloseCalls, f.h.calls)
	}
}

// wantClosedOnly fails t unless herdr was sent C-c to and asked to close
// exactly the panes and workspaces given (none: nil).
func (f *cleanupFixture) wantClosedOnly(t *testing.T, ctrlC, panes, workspaces []string) {
	t.Helper()
	if !slices.Equal(f.h.sendKeysCalls, ctrlC) || !slices.Equal(f.h.paneCloseCalls, panes) || !slices.Equal(f.h.workspaceCloseCalls, workspaces) {
		t.Fatalf("herdr was sent keys to %q and closes of panes %q and workspaces %q, want %q, %q, %q",
			f.h.sendKeysCalls, f.h.paneCloseCalls, f.h.workspaceCloseCalls, ctrlC, panes, workspaces)
	}
}

// wantStartedD2 fails t unless res is d2, running in ws-2 and pane-2 with
// cleanupNextToken.
func wantStartedD2(t *testing.T, res DirectorStartResult) {
	t.Helper()
	reg := res.Registration
	if reg.ID != "d2" || reg.State != DirectorStateRunning || reg.Token != cleanupNextToken || reg.WorkspaceID != "ws-2" || reg.PaneID != "pane-2" {
		t.Fatalf("start registered %+v, want d2 running in ws-2 and pane-2", reg)
	}
}

// d1Trail is the trail of the registration d1 that d1's events write.
func d1Trail(events []DirectorEvent) []string {
	trail := make([]string, 0, len(events))
	for _, ev := range events {
		trail = append(trail, ev.Event+" "+ev.ID)
	}
	return trail
}

// d2Trail is what a start that went through after the cleanup appends.
var d2Trail = []string{"detached d1", "stopped d1", "attached d2", "workspace_created d2", "pane_created d2", "launched d2"}

// AC6 on the herdr path: d1 running in herdr has ended, with its agent gone
// from its labelled pane (agent_ended) or its pane gone (pane_closed). start
// checks it (agent get, then the pane by its labels), closes what d1
// recorded as stop does (the label checks, C-c to the pane still there, the
// closes), records d1 stopped with detached first in one write, writes one
// line on the notice writer, and then starts d2 in a workspace of its own.
// Nothing of d2 is touched by the cleanup.
func TestStartDirector_CleansUpAnEndedHerdrDirectorAndStarts(t *testing.T) {
	for _, tc := range []struct {
		name       string
		setup      func(f *cleanupFixture)
		calls      []string // the check's and the cleanup's herdr calls, before d2's
		ctrlC      []string
		panes      []string
		notice     string
		workspaces []string
	}{
		{"agent_ended", func(f *cleanupFixture) { f.d1HerdrHasItsOwn() },
			append([]string{"agent_get", "pane_get", "tab_get", "workspace_get"}, directorCloseCalls...),
			[]string{"pane-1"}, []string{"pane-1"},
			"cleaned up d1, which had ended (agent_ended): ctrl_c=ok, pane=closed, workspace=closed, and recorded it stopped\n", []string{"ws-1"}},
		{"pane_closed", (*cleanupFixture).d1WorkspaceOnly,
			[]string{"agent_get", "pane_get", "pane_get", "workspace_get", "workspace_close"},
			nil, nil,
			"cleaned up d1, which had ended (pane_closed): ctrl_c=skipped, pane=already closed, workspace=closed, and recorded it stopped\n", []string{"ws-1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			events := checkHerdrRunning()
			f := newCleanupFixture(t, events...)
			f.ch.agentGetErr = checkAgentNotFound()
			tc.setup(f)
			res, err := f.start()
			if err != nil {
				t.Fatalf("StartDirector: %v", err)
			}
			wantStartedD2(t, res)
			want := append(slices.Clone(tc.calls), "workspace_create", "tab_create", "agent_start")
			if !slices.Equal(f.h.calls, want) {
				t.Fatalf("herdr calls = %q, want %q", f.h.calls, want)
			}
			f.wantClosedOnly(t, tc.ctrlC, tc.panes, tc.workspaces)
			if got := f.notice.String(); got != tc.notice {
				t.Fatalf("notice = %q, want %q", got, tc.notice)
			}
			wantTrail(t, f.store, append(d1Trail(events), d2Trail...)...)
			if f.gateCalls != 1 {
				t.Fatalf("watch gate asked %d times, want once, after the cleanup", f.gateCalls)
			}
		})
	}
}

// AC6 on the launch_cmd path: d1's process is gone (process_ended). The
// cleanup reads it once more, sends it nothing, records d1 stopped, says so,
// and start goes on to d2. No herdr call is made for d1.
func TestStartDirector_CleansUpAnEndedLaunchCmdDirectorAndStarts(t *testing.T) {
	events := checkCmdRunning()
	f := newCleanupFixture(t, events...)
	res, err := f.start()
	if err != nil {
		t.Fatalf("StartDirector: %v", err)
	}
	wantStartedD2(t, res)
	if want := []string{"workspace_create", "tab_create", "agent_start"}; !slices.Equal(f.h.calls, want) {
		t.Fatalf("herdr calls = %q, want only d2's %q", f.h.calls, want)
	}
	if want := []int{checkAgentPID, checkAgentPID, os.Getpid()}; !slices.Equal(f.psReads, want) {
		t.Fatalf("ps reads = %v, want the check's and the cleanup's of pid %d, then start's own %v", f.psReads, checkAgentPID, want)
	}
	want := "cleaned up d1, which had ended (process_ended): process=already ended: no process has pid 5151, and recorded it stopped; " +
		"nothing was sent to its process, and the terminal that [org.director].launch_cmd opened (name director_0123456789ab) is left open for you to close\n"
	if got := f.notice.String(); got != want {
		t.Fatalf("notice = %q, want %q", got, want)
	}
	wantTrail(t, f.store, append(d1Trail(events), d2Trail...)...)
}

// AC6, a start that died (start_ended): d1 is still starting and its start
// process is gone. For herdr the cleanup closes the workspace and the pane
// d1 recorded; for launch_cmd, with no check-in, it sends nothing, and the
// line says a check-in is refused from now on. start then starts d2.
func TestStartDirector_CleansUpARegistrationWhoseStartDied(t *testing.T) {
	t.Run(DirectorLaunchHerdr, func(t *testing.T) {
		events := []DirectorEvent{checkAttached(DirectorLaunchHerdr, true), checkWorkspace("d1", "ws-1"), checkPane("d1", "pane-1")}
		f := newCleanupFixture(t, events...)
		f.d1HerdrHasItsOwn()
		res, err := f.start()
		if err != nil {
			t.Fatalf("StartDirector: %v", err)
		}
		wantStartedD2(t, res)
		if want := append(slices.Clone(directorCloseCalls), "workspace_create", "tab_create", "agent_start"); !slices.Equal(f.h.calls, want) {
			t.Fatalf("herdr calls = %q, want the cleanup's closes (the check reads only ps), then d2's: %q", f.h.calls, want)
		}
		f.wantClosedOnly(t, []string{"pane-1"}, []string{"pane-1"}, []string{"ws-1"})
		if want := "cleaned up d1, which had ended (start_ended): ctrl_c=ok, pane=closed, workspace=closed, and recorded it stopped\n"; f.notice.String() != want {
			t.Fatalf("notice = %q, want %q", f.notice.String(), want)
		}
		wantTrail(t, f.store, append(d1Trail(events), d2Trail...)...)
	})
	t.Run(DirectorLaunchCmd, func(t *testing.T) {
		events := []DirectorEvent{checkAttached(DirectorLaunchCmd, true)}
		f := newCleanupFixture(t, events...)
		res, err := f.start()
		if err != nil {
			t.Fatalf("StartDirector: %v", err)
		}
		wantStartedD2(t, res)
		if want := []int{checkStarterPID, os.Getpid()}; !slices.Equal(f.psReads, want) {
			t.Fatalf("ps reads = %v, want the check's read of the starter, then start's own: %v", f.psReads, want)
		}
		want := "cleaned up d1, which had ended (start_ended): process=none recorded, and recorded it stopped; nothing was sent to its process, and " +
			"the terminal that [org.director].launch_cmd may have opened (name director_0123456789ab) is left open for you to close, " +
			"and a launch script that checks in from it now is refused and starts no agent\n"
		if got := f.notice.String(); got != want {
			t.Fatalf("notice = %q, want %q", got, want)
		}
		wantTrail(t, f.store, append(d1Trail(events), d2Trail...)...)
	})
}

// AC6: the line of a cleanup that was recorded stays when start is refused
// afterwards, here by the watch gate: d1 is recorded stopped, the line is
// written, and no d2 is attached.
func TestStartDirector_ACleanupIsReportedWhenTheWatchGateRefusesAfterwards(t *testing.T) {
	events := checkHerdrRunning()
	f := newCleanupFixture(t, events...)
	f.ch.agentGetErr = checkAgentNotFound()
	f.d1HerdrHasItsOwn()
	f.gateErr = &WatchGateError{StateDir: f.dir, Liveness: WatchLiveness{State: WatchLivenessNotRunning}}
	if _, err := f.start(); !errors.Is(err, ErrNoRunningWatch) {
		t.Fatalf("StartDirector err = %v, want the watch gate's refusal", err)
	}
	if !strings.HasPrefix(f.notice.String(), "cleaned up d1, which had ended (agent_ended): ") || strings.Count(f.notice.String(), "\n") != 1 {
		t.Fatalf("notice = %q, want the one line of the cleanup", f.notice.String())
	}
	wantTrail(t, f.store, append(d1Trail(events), "detached d1", "stopped d1")...)
	f.wantClosedOnly(t, []string{"pane-1"}, []string{"pane-1"}, []string{"ws-1"})
	if _, err := os.Stat(filepath.Join(f.dir, "prompts")); err == nil {
		t.Fatal("a start the watch gate refused wrote a prompt file")
	}
}

// AC6: a check that finds d1 alive or cannot tell refuses start as before
// (*DirectorAttachedError naming d1), now with what the check found (Check).
// Nothing is written, herdr is only asked (no C-c, no close), ps is only
// read, no line is written, and the watch gate is not asked. The launch_cmd
// row whose process is alive uses a process the test started, and checks
// that it is still running afterwards.
func TestStartDirector_RefusesADirectorItsCheckFindsAliveOrCannotTell(t *testing.T) {
	for _, tc := range []struct {
		name       string
		setup      func(t *testing.T, f *cleanupFixture) (after func())
		state, how string
	}{
		{"agent_found", func(t *testing.T, f *cleanupFixture) func() {
			writeDirectorEvents(t, f.store, checkHerdrRunning()...)
			return nil
		}, DirectorCheckAlive, DirectorCheckHowAgentFound},
		{"start_running", func(t *testing.T, f *cleanupFixture) func() {
			writeDirectorEvents(t, f.store, checkAttached(DirectorLaunchHerdr, true), checkWorkspace("d1", "ws-1"), checkPane("d1", "pane-1"))
			f.d1HerdrHasItsOwn()
			f.procs[checkStarterPID] = DirectorProcess{State: DirectorProcessAlive, StartTime: checkStarterStart}
			return nil
		}, DirectorCheckAlive, DirectorCheckHowStartRunning},
		{"process_found", func(t *testing.T, f *cleanupFixture) func() {
			p := startTestProcess(t, sleepPath(t), "30")
			proc := mustReadProcess(t, p.pid())
			writeDirectorEvents(t, f.store, checkAttached(DirectorLaunchCmd, true), checkEvent("d1", DirectorEventLaunched, func(ev *DirectorEvent) {
				ev.PID, ev.StartTime = p.pid(), proc.StartTime
			}))
			f.procs[p.pid()] = proc
			return func() { wantAliveWith(t, p.pid(), proc.StartTime, "a refused start must send it nothing") }
		}, DirectorCheckAlive, DirectorCheckHowProcessFound},
		{"pid_reused", func(t *testing.T, f *cleanupFixture) func() {
			writeDirectorEvents(t, f.store, checkCmdRunning()...)
			f.procs[checkAgentPID] = DirectorProcess{State: DirectorProcessAlive, StartTime: checkOtherStart}
			return nil
		}, DirectorCheckUnknown, DirectorCheckHowPIDReused},
		{"agent_get_failed", func(t *testing.T, f *cleanupFixture) func() {
			writeDirectorEvents(t, f.store, checkHerdrRunning()...)
			f.d1HerdrHasItsOwn()
			f.ch.agentGetErr = errors.New("herdr: no server is running")
			return nil
		}, DirectorCheckUnknown, DirectorCheckHowAgentGetFailed},
		{"no_starter", func(t *testing.T, f *cleanupFixture) func() {
			writeDirectorEvents(t, f.store, checkAttached(DirectorLaunchHerdr, false), checkWorkspace("d1", "ws-1"))
			return nil
		}, DirectorCheckUnknown, DirectorCheckHowNoStarter},
		{"pane_unconfirmed", func(t *testing.T, f *cleanupFixture) func() {
			writeDirectorEvents(t, f.store, checkHerdrRunning()...)
			f.herdrHas("pane-1", "someone-else", "ws-1", DirectorWorkspaceLabel(directorTestToken))
			f.ch.agentGetErr = checkAgentNotFound()
			return nil
		}, DirectorCheckUnknown, DirectorCheckHowPaneUnconfirmed},
		{"no_agent_get", func(t *testing.T, f *cleanupFixture) func() {
			writeDirectorEvents(t, f.store, checkHerdrRunning()...)
			f.o.Herdr = f.h
			return nil
		}, DirectorCheckUnknown, DirectorCheckHowNoAgentGet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCleanupFixture(t)
			afterRefusal := tc.setup(t, f)
			before := directorFileState(t, f.store)
			_, err := f.start()
			var ae *DirectorAttachedError
			if !errors.As(err, &ae) || !errors.Is(err, ErrDirectorAttached) || ae.Current == nil || ae.Current.ID != "d1" {
				t.Fatalf("err = %#v, want a *DirectorAttachedError naming d1", err)
			}
			if ae.Check == nil || ae.Check.ID != "d1" || ae.Check.State != tc.state || ae.Check.How != tc.how || ae.Check.CheckedAt != checkNowUTC {
				t.Fatalf("Check = %+v, want d1 %s (%s) at %s", ae.Check, tc.state, tc.how, checkNowUTC)
			}
			if (ae.Check.Detail != "") != (tc.state == DirectorCheckUnknown) {
				t.Fatalf("Check.Detail = %q, want one for unknown only", ae.Check.Detail)
			}
			if want := directorStartAttachedErr(ae.Current).Error(); err.Error() != want {
				t.Fatalf("err = %q, want the refusal as before, %q", err, want)
			}
			if after := directorFileState(t, f.store); after != before {
				t.Fatalf("director.jsonl changed from %q to %q", before, after)
			}
			wantOnlyHerdrGets(t, f.h.calls)
			f.wantNoClose(t)
			if f.notice.Len() != 0 || f.gateCalls != 0 {
				t.Fatalf("notice %q and %d watch gate calls, want none", f.notice.String(), f.gateCalls)
			}
			if afterRefusal != nil {
				afterRefusal()
			}
		})
	}
}

// AC6: with nothing to check, start refuses as before, with no Check, no
// herdr call, and no ps read: a promoted session's registration (never
// checked), and a headless registration recorded stopped while it is still
// current (CheckDirector has nothing to check in it).
func TestStartDirector_RefusesWithNoCheckWhenThereIsNothingToCheck(t *testing.T) {
	for name, events := range map[string][]DirectorEvent{
		"a promoted session's registration": {{TS: "2026-10-18T08:00:00Z", ID: "d1", Event: DirectorEventAttached, Driver: DirectorDriverClaude,
			Wake: []string{DirectorWakePull}, User: "u", Cwd: "/w"}},
		"a current registration recorded stopped": append(checkHerdrRunning(), checkEvent("d1", DirectorEventStopped, nil)),
	} {
		t.Run(name, func(t *testing.T) {
			f := newCleanupFixture(t, events...)
			before := directorFileState(t, f.store)
			_, err := f.start()
			var ae *DirectorAttachedError
			if !errors.As(err, &ae) || ae.Current == nil || ae.Current.ID != "d1" || ae.Check != nil {
				t.Fatalf("err = %#v, want a *DirectorAttachedError naming d1 with no Check", err)
			}
			if want := directorStartAttachedErr(ae.Current).Error(); err.Error() != want {
				t.Fatalf("err = %q, want %q", err, want)
			}
			if len(f.h.calls) != 0 || len(f.psReads) != 0 {
				t.Fatalf("herdr calls %q and ps reads %v, want none", f.h.calls, f.psReads)
			}
			if after := directorFileState(t, f.store); after != before || f.notice.Len() != 0 {
				t.Fatalf("director.jsonl changed from %q to %q, notice %q", before, after, f.notice.String())
			}
		})
	}
}

// wantCleanupRefused asserts err is start's *DirectorCleanupError for d1 of
// launch, refused by ErrDirectorAttached with its cleanup error wrapping
// cleanupIs, stop command stop, nothing detached; that director.jsonl is
// still trail and no line was written; and returns a copy of the error (a
// value, which is not an error itself, so a caller may leave it unused).
func wantCleanupRefused(t *testing.T, f *cleanupFixture, err error, launch string, cleanupIs error, stop string, trail []string) DirectorCleanupError {
	t.Helper()
	var ce *DirectorCleanupError
	if !errors.As(err, &ce) || ce.Verb != "start" || ce.ID != "d1" || ce.Launch != launch || ce.Detached || !errors.Is(err, ErrDirectorAttached) {
		t.Fatalf("err = %#v, want start's *DirectorCleanupError of d1 (%s), nothing detached", err, launch)
	}
	if cleanupIs != nil && !errors.Is(ce.CleanupErr, cleanupIs) {
		t.Fatalf("CleanupErr = %v, want it to wrap %v", ce.CleanupErr, cleanupIs)
	}
	if ce.StopCommand != stop || !strings.Contains(err.Error(), stop) {
		t.Fatalf("StopCommand = %q in %q, want %q", ce.StopCommand, err, stop)
	}
	if !strings.Contains(err.Error(), "which start found ended (") || !strings.Contains(err.Error(), "so it started no new director") {
		t.Fatalf("err = %q, want it to say start found d1 ended and started none", err)
	}
	wantTrail(t, f.store, trail...)
	if f.notice.Len() != 0 || f.gateCalls != 0 {
		t.Fatalf("notice %q and %d watch gate calls, want none: nothing was recorded", f.notice.String(), f.gateCalls)
	}
	return *ce
}

// AC6: a cleanup that cannot finish refuses start with nothing written and
// no d2, and names the stop that finishes it: a workspace whose label is
// another launch's (nothing sent, the stop with --force, the herdr commands
// that find the workspace by its label), a pane close that fails (the stop
// without --force, the commands that close what is open), and a call from
// inside d1's workspace (HERDR_WORKSPACE_ID; nothing sent).
func TestStartDirector_ACleanupThatCannotFinishRefusesStart(t *testing.T) {
	events := checkHerdrRunning()
	t.Run("the workspace is another launch's", func(t *testing.T) {
		f := newCleanupFixture(t, events...)
		f.ch.agentGetErr = checkAgentNotFound()
		setLabel(&f.h.workspaceLabels, "ws-1", "orgA")
		_, err := f.start()
		ce := wantCleanupRefused(t, f, err, DirectorLaunchHerdr, nil, "ralph org director stop --id d1 --force", d1Trail(events))
		if !ce.Unconfirmed || !slices.Equal(ce.HerdrCommands, directorFindWorkspaceCommands(DirectorWorkspaceLabel(directorTestToken))) {
			t.Fatalf("cleanup error = %+v, want it unconfirmed with the commands that find the workspace by its label", ce)
		}
		f.wantNoClose(t)
	})
	t.Run("the pane close fails", func(t *testing.T) {
		f := newCleanupFixture(t, events...)
		f.ch.agentGetErr = checkAgentNotFound()
		f.d1HerdrHasItsOwn()
		f.h.paneCloseErrs = map[string]error{"pane-1": errors.New("herdr: connection refused")}
		_, err := f.start()
		ce := wantCleanupRefused(t, f, err, DirectorLaunchHerdr, nil, "ralph org director stop --id d1", d1Trail(events))
		if ce.Unconfirmed || !slices.Equal(ce.HerdrCommands, []string{"herdr pane close pane-1", "herdr workspace close ws-1"}) {
			t.Fatalf("cleanup error = %+v, want the commands that close pane-1 and ws-1", ce)
		}
		f.wantClosedOnly(t, []string{"pane-1"}, []string{"pane-1"}, nil)
	})
	t.Run("inside d1's workspace", func(t *testing.T) {
		f := newCleanupFixture(t, events...)
		f.ch.agentGetErr = checkAgentNotFound()
		f.d1HerdrHasItsOwn()
		f.env[herdrWorkspaceIDEnv] = "ws-1"
		_, err := f.start()
		wantCleanupRefused(t, f, err, DirectorLaunchHerdr, ErrDirectorStopSelfWorkspace, "ralph org director stop --id d1", d1Trail(events))
		f.wantNoClose(t)
	})
}

// AC6, the barrier of Codex's HIGH on the plan, for both launches: an
// earlier start finishes d1 (launched, or its launch script's check-in) and
// ends between the later start's read of d1 as starting and its probes, so
// ps finds d1's start process gone. The later start must not read that as
// start_ended: its check reads d1 again, finds it running and alive, and
// start refuses, with no C-c, no close, no signal, and no d2.
func TestStartDirector_AStartThatFinishedDuringTheCheckIsNotCleanedUp(t *testing.T) {
	t.Run(DirectorLaunchHerdr, func(t *testing.T) {
		f := newCleanupFixture(t, checkAttached(DirectorLaunchHerdr, true), checkWorkspace("d1", "ws-1"), checkPane("d1", "pane-1"))
		f.d1HerdrHasItsOwn()
		f.procs[checkStarterPID] = DirectorProcess{State: DirectorProcessAlive, StartTime: checkStarterStart}
		f.o.directorCheckAfterRead = func(attempt int) {
			if attempt != 1 {
				return
			}
			if still, err := f.store.MarkLaunched("d1", testDirectorWho); err != nil || !still {
				t.Fatalf("MarkLaunched = %v, %v", still, err)
			}
			delete(f.procs, checkStarterPID)
		}
		_, err := f.start()
		wantBarrierRefusal(t, f, err, DirectorCheckHowAgentFound)
		wantTrail(t, f.store, "attached d1", "workspace_created d1", "pane_created d1", "launched d1")
		f.wantNoClose(t)
	})
	t.Run(DirectorLaunchCmd, func(t *testing.T) {
		agent := startTestProcess(t, sleepPath(t), "30")
		proc := mustReadProcess(t, agent.pid())
		f := newCleanupFixture(t, checkAttached(DirectorLaunchCmd, true))
		f.procs[checkStarterPID] = DirectorProcess{State: DirectorProcessAlive, StartTime: checkStarterStart}
		f.o.directorCheckAfterRead = func(attempt int) {
			if attempt != 1 {
				return
			}
			checkIn := DirectorCmdCheckIn{ID: "d1", Token: directorTestToken, PID: agent.pid(), StartTime: proc.StartTime, Who: testDirectorWho,
				ReadStarter: func(context.Context, int) (DirectorProcess, error) {
					return DirectorProcess{State: DirectorProcessAlive, StartTime: checkStarterStart}, nil
				}}
			if err := f.store.MarkCmdLaunched(checkIn); err != nil {
				t.Fatalf("MarkCmdLaunched: %v", err)
			}
			f.procs[agent.pid()] = proc
			delete(f.procs, checkStarterPID)
		}
		_, err := f.start()
		wantBarrierRefusal(t, f, err, DirectorCheckHowProcessFound)
		wantTrail(t, f.store, "attached d1", "launched d1")
		if len(f.h.calls) != 0 {
			t.Fatalf("herdr calls = %q, want none", f.h.calls)
		}
		wantAliveWith(t, agent.pid(), proc.StartTime, "the director the earlier start launched must not be signalled")
	})
}

// wantBarrierRefusal asserts err refuses start on d1 found running and
// alive by how, from the check's second try, with no line written.
func wantBarrierRefusal(t *testing.T, f *cleanupFixture, err error, how string) {
	t.Helper()
	var ae *DirectorAttachedError
	if !errors.As(err, &ae) || ae.Check == nil || ae.Check.State != DirectorCheckAlive || ae.Check.How != how {
		t.Fatalf("err = %#v, want a refusal with d1 alive (%s), not a cleanup of start_ended", err, how)
	}
	if ae.Current == nil || ae.Current.State != DirectorStateRunning {
		t.Fatalf("Current = %+v, want d1 as the second try read it, running", ae.Current)
	}
	if f.notice.Len() != 0 || f.gateCalls != 0 {
		t.Fatalf("notice %q and %d watch gate calls, want none", f.notice.String(), f.gateCalls)
	}
}

// AC6, the second barrier: director.jsonl records something for d1 after
// the cleanup's closes and before its record (MarkStoppedAtVersion, reached
// through the store's beforeMarkStopped seam). The cleanup records nothing,
// start writes no d2, and the refusal names start again and the stop. For
// herdr, d1's start, read as ended, records a pane after all: the cleanup
// closed only the workspace d1 had when the check read it and never touched
// that pane. For launch_cmd a human detaches d1 meanwhile; nothing was sent
// to any process.
func TestStartDirector_ARegistrationThatChangesBeforeTheRecordIsNotRecordedStopped(t *testing.T) {
	t.Run(DirectorLaunchHerdr, func(t *testing.T) {
		f := newCleanupFixture(t, checkAttached(DirectorLaunchHerdr, true), checkWorkspace("d1", "ws-1"))
		f.d1WorkspaceOnly()
		f.store.beforeMarkStopped = func() {
			f.store.beforeMarkStopped = nil
			if err := f.store.RecordPane("d1", "pane-9", testDirectorWho); err != nil {
				t.Fatalf("RecordPane: %v", err)
			}
		}
		_, err := f.start()
		ce := wantCleanupRefused(t, f, err, DirectorLaunchHerdr, ErrDirectorChangedSinceCheck, "ralph org director stop --id d1",
			[]string{"attached d1", "workspace_created d1", "pane_created d1"})
		if len(ce.HerdrCommands) != 0 || !strings.Contains(err.Error(), "run start again, which checks the registration as it is now") {
			t.Fatalf("err = %q (herdr commands %q), want no command and the start to run again", err, ce.HerdrCommands)
		}
		f.wantClosedOnly(t, nil, nil, []string{"ws-1"})
		if want := []string{"workspace_get", "workspace_close"}; !slices.Equal(f.h.calls, want) {
			t.Fatalf("herdr calls = %q, want only version A's workspace checked and closed: %q", f.h.calls, want)
		}
	})
	// The pane lands after the check and before the cleanup's first herdr
	// call: once the check has read the start process (the fake ps arms it),
	// the cleanup's look at HERDR_PANE_ID records it, labelled as d1's own.
	// The cleanup still closes only the workspace of version A.
	t.Run(DirectorLaunchHerdr+", before the closes", func(t *testing.T) {
		f := newCleanupFixture(t, checkAttached(DirectorLaunchHerdr, true), checkWorkspace("d1", "ws-1"))
		f.d1WorkspaceOnly()
		armed, read := false, f.o.DirectorProcessReader
		f.o.DirectorProcessReader = func(ctx context.Context, pid int) (DirectorProcess, error) {
			armed = armed || pid == checkStarterPID
			return read(ctx, pid)
		}
		getenv := f.o.Getenv
		f.o.Getenv = func(key string) string {
			if key == herdrPaneIDEnv && armed {
				armed = false
				if err := f.store.RecordPane("d1", "pane-9", testDirectorWho); err != nil {
					t.Fatalf("RecordPane: %v", err)
				}
				f.herdrHas("pane-9", DirectorTabLabel(directorTestToken), "ws-1", DirectorWorkspaceLabel(directorTestToken))
			}
			return getenv(key)
		}
		_, err := f.start()
		wantCleanupRefused(t, f, err, DirectorLaunchHerdr, ErrDirectorChangedSinceCheck, "ralph org director stop --id d1",
			[]string{"attached d1", "workspace_created d1", "pane_created d1"})
		f.wantClosedOnly(t, nil, nil, []string{"ws-1"})
		if want := []string{"workspace_get", "workspace_close"}; !slices.Equal(f.h.calls, want) {
			t.Fatalf("herdr calls = %q, want only version A's workspace checked and closed: %q", f.h.calls, want)
		}
	})
	t.Run(DirectorLaunchCmd, func(t *testing.T) {
		f := newCleanupFixture(t, checkCmdRunning()...)
		f.store.beforeMarkStopped = func() {
			f.store.beforeMarkStopped = nil
			if _, err := f.store.Detach(DirectorDetach{ID: "d1", User: "u", Cwd: "/w"}); err != nil {
				t.Fatalf("Detach: %v", err)
			}
		}
		_, err := f.start()
		wantCleanupRefused(t, f, err, DirectorLaunchCmd, ErrDirectorChangedSinceCheck, "ralph org director stop --id d1",
			[]string{"attached d1", "launched d1", "detached d1"})
		if len(f.h.calls) != 0 {
			t.Fatalf("herdr calls = %q, want none", f.h.calls)
		}
	})
}

// mustDirectorVersion is the directorVersion of id in s's director.jsonl as
// it is now.
func mustDirectorVersion(t *testing.T, s *DirectorStore, id string) directorVersion {
	t.Helper()
	raw, err := s.readRaw()
	if err != nil {
		t.Fatal(err)
	}
	return directorVersionOf(raw, foldDirector(raw), id)
}

// mustOpenHeadless is the open headless registration id of s.
func mustOpenHeadless(t *testing.T, s *DirectorStore, id string) DirectorRegistration {
	t.Helper()
	reg, err := s.openHeadlessRegistration("test", id)
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

// MarkStoppedAtVersion with the version unchanged writes what
// MarkStoppedIfUnchanged writes for the same read, byte for byte: detached
// and stopped while the registration is current, stopped alone once it was
// detached.
func TestMarkStoppedAtVersion_UnchangedWritesWhatMarkStoppedIfUnchangedWrites(t *testing.T) {
	for name, events := range map[string][]DirectorEvent{
		"current":           checkHerdrRunning(),
		"detached but open": append(checkHerdrRunning(), checkEvent("d1", DirectorEventDetached, nil)),
	} {
		t.Run(name, func(t *testing.T) {
			stores := [2]*DirectorStore{}
			for i := range stores {
				stores[i] = NewDirectorStore(t.TempDir())
				stores[i].Now = func() time.Time { return checkNow }
				writeDirectorEvents(t, stores[i], events...)
			}
			read := mustOpenHeadless(t, stores[0], "d1")
			wasA, errA := stores[0].MarkStoppedAtVersion(read, mustDirectorVersion(t, stores[0], "d1"), testDirectorWho)
			wasB, errB := stores[1].MarkStoppedIfUnchanged(read, testDirectorWho)
			if errA != nil || errB != nil || wasA != wasB || wasA != (name == "current") {
				t.Fatalf("MarkStoppedAtVersion = %v, %v; MarkStoppedIfUnchanged = %v, %v", wasA, errA, wasB, errB)
			}
			if a, b := readDirectorFile(t, stores[0]), readDirectorFile(t, stores[1]); a != b {
				t.Fatalf("MarkStoppedAtVersion wrote\n%s\nMarkStoppedIfUnchanged wrote\n%s", a, b)
			}
		})
	}
}

// MarkStoppedAtVersion refuses, writing nothing, with an error that wraps
// ErrDirectorChangedSinceCheck and names both versions, when the version
// read is not the file's: an event recorded for the id since (here a
// detached, and a pane_created that MarkStoppedIfUnchanged's own comparison
// would also catch), another registration current since (d1 detached and
// open at the read, then a session attached as d2, with no event for d1),
// and a zero version.
func TestMarkStoppedAtVersion_RefusesAVersionThatChanged(t *testing.T) {
	for _, tc := range []struct {
		name   string
		events []DirectorEvent
		change func(t *testing.T, s *DirectorStore)
		zero   bool
		want   string
	}{
		{"an event for the id", checkHerdrRunning(), func(t *testing.T, s *DirectorStore) {
			if _, err := s.Detach(DirectorDetach{ID: "d1", User: "u", Cwd: "/w"}); err != nil {
				t.Fatal(err)
			}
		}, false, "d1 had 4 events, with d1 current when the check read it, and has 5 events, with no registration current now"},
		{"a pane recorded", []DirectorEvent{checkAttached(DirectorLaunchHerdr, true), checkWorkspace("d1", "ws-1")}, func(t *testing.T, s *DirectorStore) {
			if err := s.RecordPane("d1", "pane-9", testDirectorWho); err != nil {
				t.Fatal(err)
			}
		}, false, "d1 had 2 events, with d1 current when the check read it, and has 3 events, with d1 current now"},
		{"another registration current", append(checkHerdrRunning(), checkEvent("d1", DirectorEventDetached, nil)), func(t *testing.T, s *DirectorStore) {
			mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude, User: "u", Cwd: "/w"})
		}, false, "d1 had 5 events, with no registration current when the check read it, and has 5 events, with d2 current now"},
		{"a zero version", checkHerdrRunning(), nil, true, "d1 had 0 events, with no registration current when the check read it"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewDirectorStore(t.TempDir())
			s.Now = func() time.Time { return checkNow }
			writeDirectorEvents(t, s, tc.events...)
			read := mustOpenHeadless(t, s, "d1")
			var v directorVersion
			if !tc.zero {
				v = mustDirectorVersion(t, s, "d1")
			}
			if tc.change != nil {
				tc.change(t, s)
			}
			before := readDirectorFile(t, s)
			was, err := s.MarkStoppedAtVersion(read, v, testDirectorWho)
			if !errors.Is(err, ErrDirectorChangedSinceCheck) || was {
				t.Fatalf("MarkStoppedAtVersion = %v, %v; want a refusal wrapping ErrDirectorChangedSinceCheck", was, err)
			}
			if !strings.HasPrefix(err.Error(), "org: director: stop d1: the registration changed after start checked it: ") || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %q, want it to name the versions: %q", err, tc.want)
			}
			if after := readDirectorFile(t, s); after != before {
				t.Fatalf("director.jsonl changed from\n%s\nto\n%s", before, after)
			}
		})
	}
}
