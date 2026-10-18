package org

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// The tests of CheckDirector (plan 2026-10-18-org-director-liveness, D3 and
// AC3). They run no ps and no herdr: the Org's DirectorProcessReader answers
// from dirCheckFixture.procs, and its herdr is a fakeHerdr that also answers
// agent get (checkHerdr). Every ledger is a t.TempDir.

// checkNow is the time the tests give CheckDirector, in a zone other than
// UTC, so CheckedAt shows it is written in UTC (checkNowUTC).
var checkNow = time.Date(2026, 10, 18, 18, 30, 0, 0, time.FixedZone("JST", 9*60*60))

const checkNowUTC = "2026-10-18T09:30:00Z"

// The starter and the checked-in process of the registrations below.
const (
	checkStarterPID   = 4242
	checkStarterStart = directorTestStartTime
	checkAgentPID     = 5151
	checkAgentStart   = "Thu Oct 15 21:52:00 2026"
	checkOtherStart   = "Fri Oct 16 08:00:00 2026"
)

// checkHerdr is a fakeHerdr that also runs herdr agent get: every call
// returns agentGetErr (nil: herdr has the agent), or, with agentGetBlock,
// nothing until the call's context ends. Each call is logged as "agent_get"
// in calls, next to the fakeHerdr's own calls, and its target in
// agentGetCalls.
type checkHerdr struct {
	*fakeHerdr
	agentGetErr   error
	agentGetBlock bool
	agentGetCalls []string
}

func (h *checkHerdr) AgentGet(ctx context.Context, target string) (string, error) {
	h.mu.Lock()
	h.calls = append(h.calls, "agent_get")
	h.agentGetCalls = append(h.agentGetCalls, target)
	err, block := h.agentGetErr, h.agentGetBlock
	h.mu.Unlock()
	if block {
		<-ctx.Done()
		return "", ctx.Err()
	}
	if err != nil {
		return "", err
	}
	return "status: idle", nil
}

// dirCheckFixture is one ledger for the CheckDirector tests: an Org from testOrg
// whose herdr is a checkHerdr and whose ps is procs, and the director store
// in the manifest's directory.
type dirCheckFixture struct {
	o     *Org
	h     *checkHerdr
	store *DirectorStore
	dir   string
	// procs is what the fake ps answers for a pid; a pid with no entry has
	// no process. psErr, when set, makes every read fail. psReads logs the
	// pids read.
	procs   map[int]DirectorProcess
	psErr   error
	psReads []int
	// attempts logs the tries directorCheckAfterRead saw; seam, when set,
	// runs in it, and seamLeft is director.jsonl as the last seam left it,
	// which check compares the file with afterwards.
	attempts []int
	seam     func(attempt int)
	seamLeft string
}

func newDirCheckFixture(t *testing.T) *dirCheckFixture {
	t.Helper()
	o, fh, _ := testOrg(t)
	f := &dirCheckFixture{o: o, h: &checkHerdr{fakeHerdr: fh}, dir: filepath.Dir(o.Manifest.Path()), procs: map[int]DirectorProcess{}}
	o.Herdr = f.h
	o.DirectorProcessReader = func(_ context.Context, pid int) (DirectorProcess, error) {
		f.psReads = append(f.psReads, pid)
		return f.peek(pid)
	}
	o.directorCheckAfterRead = func(attempt int) {
		f.attempts = append(f.attempts, attempt)
		if f.seam != nil {
			f.seam(attempt)
			f.seamLeft = directorFileState(t, f.store)
		}
	}
	f.store = NewDirectorStore(f.dir)
	clock := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	f.store.Now = func() time.Time { return clock }
	return f
}

// peek is the fake ps without the log, for a check-in's read of the starter.
func (f *dirCheckFixture) peek(pid int) (DirectorProcess, error) {
	if f.psErr != nil {
		return DirectorProcess{}, f.psErr
	}
	if p, ok := f.procs[pid]; ok {
		return p, nil
	}
	return DirectorProcess{State: DirectorProcessGone}, nil
}

func (f *dirCheckFixture) alive(pid int, start string) {
	f.procs[pid] = DirectorProcess{State: DirectorProcessAlive, StartTime: start}
}

// herdrHas makes the fake herdr hold paneID in a tab labelled tabLabel in
// workspace wsID labelled wsLabel.
func (f *dirCheckFixture) herdrHas(paneID, tabLabel, wsID, wsLabel string) {
	setLabel(&f.h.paneTabs, paneID, fakeTabID(paneID))
	setLabel(&f.h.paneWorkspaces, paneID, wsID)
	setLabel(&f.h.tabLabels, fakeTabID(paneID), tabLabel)
	setLabel(&f.h.workspaceLabels, wsID, wsLabel)
}

// check runs CheckDirector at checkNow and asserts it wrote nothing to
// director.jsonl (the file is as it was before, or as the last seam left it)
// and made no herdr call other than the four gets.
func (f *dirCheckFixture) check(t *testing.T) DirectorCheckResult {
	t.Helper()
	f.seamLeft = directorFileState(t, f.store)
	res, err := f.o.CheckDirector(f.store, checkNow)
	if err != nil {
		t.Fatalf("CheckDirector: %v", err)
	}
	if after := directorFileState(t, f.store); after != f.seamLeft {
		t.Fatalf("CheckDirector changed director.jsonl:\nbefore %q\nafter  %q", f.seamLeft, after)
	}
	wantOnlyHerdrGets(t, f.h.calls)
	return res
}

// wantOnlyHerdrGets asserts calls holds no herdr call but the four gets: no
// close, no send-keys (C-c included), no send-text.
func wantOnlyHerdrGets(t *testing.T, calls []string) {
	t.Helper()
	for _, c := range calls {
		switch c {
		case "agent_get", "pane_get", "tab_get", "workspace_get":
		default:
			t.Fatalf("CheckDirector made the herdr call %q (all calls: %q); it may only get", c, calls)
		}
	}
}

// checkAttached is the attached of d1 with launch (herdr or cmd), with the
// starter when starter is set.
func checkAttached(launch string, starter bool) DirectorEvent {
	return checkAttachedAs("d1", directorTestToken, launch, starter)
}

func checkAttachedAs(id, token, launch string, starter bool) DirectorEvent {
	ev := DirectorEvent{
		TS: "2026-10-18T08:00:00Z", ID: id, Event: DirectorEventAttached, Driver: DirectorDriverCodex, Wake: []string{DirectorWakePull},
		Launch: launch, Token: token, Model: "m", PermissionMode: PermissionModeGuarded, User: "u", Cwd: "/w",
	}
	if starter {
		ev.StarterPID, ev.StarterStartTime = checkStarterPID, checkStarterStart
	}
	return ev
}

// checkEvent is a later event of kind for id; set fills in its fields.
func checkEvent(id, kind string, set func(*DirectorEvent)) DirectorEvent {
	ev := DirectorEvent{TS: "2026-10-18T08:00:01Z", ID: id, Event: kind, User: "u", Cwd: "/w"}
	if set != nil {
		set(&ev)
	}
	return ev
}

func checkWorkspace(id, ws string) DirectorEvent {
	return checkEvent(id, DirectorEventWorkspaceCreated, func(ev *DirectorEvent) { ev.WorkspaceID = ws })
}

func checkPane(id, pane string) DirectorEvent {
	return checkEvent(id, DirectorEventPaneCreated, func(ev *DirectorEvent) { ev.PaneID = pane })
}

// checkHerdrRunning is d1 running in herdr with workspace ws-1 and pane
// pane-1 recorded.
func checkHerdrRunning() []DirectorEvent {
	return []DirectorEvent{checkAttached(DirectorLaunchHerdr, true), checkWorkspace("d1", "ws-1"), checkPane("d1", "pane-1"), checkEvent("d1", DirectorEventLaunched, nil)}
}

// checkCmdRunning is d1 of a launch cmd whose launch script checked in with
// checkAgentPID, started at checkAgentStart.
func checkCmdRunning() []DirectorEvent {
	return []DirectorEvent{checkAttached(DirectorLaunchCmd, true), checkEvent("d1", DirectorEventLaunched, func(ev *DirectorEvent) {
		ev.PID, ev.StartTime = checkAgentPID, checkAgentStart
	})}
}

// checkAgentNotFound is the error the driver's AgentGet returns for herdr's
// agent_not_found reply: the Runner error with the envelope wrapped next to
// it.
func checkAgentNotFound() error {
	return fmt.Errorf("%w (%w)", errors.New("herdr: exit status 1"), driver.NewHerdrError(driver.HerdrCodeAgentNotFound, "agent "+DirectorAgentName(directorTestToken)+" not found"))
}

// checkRow is one row of D3: the registration (events), what herdr and ps
// answer (setup), the state and how CheckDirector must return, a substring
// of the detail (empty: the detail must be empty), and the herdr calls and
// ps reads it must make.
type checkRow struct {
	name       string
	events     []DirectorEvent
	setup      func(f *dirCheckFixture)
	state, how string
	detail     string
	herdrCalls []string
	psReads    []int
}

func runCheckRows(t *testing.T, rows []checkRow) {
	t.Helper()
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			f := newDirCheckFixture(t)
			writeDirectorEvents(t, f.store, row.events...)
			if row.setup != nil {
				row.setup(f)
			}
			res := f.check(t)
			got := res.Check
			switch {
			case row.detail == "" && got.Detail != "":
				t.Fatalf("detail = %q, want none", got.Detail)
			case row.detail != "" && !strings.Contains(got.Detail, row.detail):
				t.Fatalf("detail = %q, want it to hold %q", got.Detail, row.detail)
			}
			want := DirectorCheck{ID: "d1", State: row.state, How: row.how, CheckedAt: checkNowUTC, Detail: got.Detail}
			if !res.Checked || got != want {
				t.Fatalf("CheckDirector = checked %v, %+v; want %+v", res.Checked, got, want)
			}
			if cur := mustReadDirector(t, f.store).Current; cur == nil || !reflect.DeepEqual(res.Registration, *cur) {
				t.Fatalf("result registration = %+v, want the one it checked, %+v", res.Registration, cur)
			}
			if !slices.Equal(f.h.calls, row.herdrCalls) {
				t.Fatalf("herdr calls = %q, want %q", f.h.calls, row.herdrCalls)
			}
			if !slices.Equal(f.psReads, row.psReads) {
				t.Fatalf("ps reads = %v, want %v", f.psReads, row.psReads)
			}
			if !slices.Equal(f.attempts, []int{1}) {
				t.Fatalf("tries = %v, want one: nothing changed", f.attempts)
			}
		})
	}
}

// AC3, the starting rows: for both launches the registration is read by its
// starter only, with no herdr call, even with its pane recorded.
func TestCheckDirector_StartingRows(t *testing.T) {
	for _, launch := range []string{DirectorLaunchHerdr, DirectorLaunchCmd} {
		t.Run(launch, func(t *testing.T) {
			starting := []DirectorEvent{checkAttached(launch, true)}
			runCheckRows(t, []checkRow{
				{name: "no_starter", events: []DirectorEvent{checkAttached(launch, false)},
					state: DirectorCheckUnknown, how: DirectorCheckHowNoStarter, detail: "no ralph org director start process is recorded"},
				{name: "start_ended gone", events: starting,
					state: DirectorCheckEnded, how: DirectorCheckHowStartEnded, psReads: []int{checkStarterPID}},
				{name: "start_ended zombie", events: starting,
					setup: func(f *dirCheckFixture) {
						f.procs[checkStarterPID] = DirectorProcess{State: DirectorProcessZombie, StartTime: checkStarterStart}
					},
					state: DirectorCheckEnded, how: DirectorCheckHowStartEnded, psReads: []int{checkStarterPID}},
				{name: "starter_pid_reused", events: starting, setup: func(f *dirCheckFixture) { f.alive(checkStarterPID, checkOtherStart) },
					state: DirectorCheckUnknown, how: DirectorCheckHowStarterPIDReused,
					detail: `pid 4242, the start process, started "Fri Oct 16 08:00:00 2026", not "Thu Oct 15 21:51:09 2026"`, psReads: []int{checkStarterPID}},
				{name: "start_running", events: starting, setup: func(f *dirCheckFixture) { f.alive(checkStarterPID, checkStarterStart) },
					state: DirectorCheckAlive, how: DirectorCheckHowStartRunning, psReads: []int{checkStarterPID}},
				{name: "ps fails", events: starting, setup: func(f *dirCheckFixture) { f.psErr = errors.New("ps: no answer") },
					state: DirectorCheckUnknown, how: DirectorCheckHowStarterUnreadable, detail: "pid 4242: ps: no answer", psReads: []int{checkStarterPID}},
				{name: "ps answers a state it never gives", events: starting,
					setup: func(f *dirCheckFixture) {
						f.procs[checkStarterPID] = DirectorProcess{State: "sleeping", StartTime: checkStarterStart}
					},
					state: DirectorCheckUnknown, how: DirectorCheckHowStarterUnreadable, detail: `as "sleeping"`, psReads: []int{checkStarterPID}},
			})
		})
	}
	t.Run("herdr with its pane recorded", func(t *testing.T) {
		runCheckRows(t, []checkRow{{name: "start_ended",
			events: []DirectorEvent{checkAttached(DirectorLaunchHerdr, true), checkWorkspace("d1", "ws-1"), checkPane("d1", "pane-1")},
			setup: func(f *dirCheckFixture) {
				f.herdrHas("pane-1", DirectorTabLabel(directorTestToken), "ws-1", DirectorWorkspaceLabel(directorTestToken))
			},
			state: DirectorCheckEnded, how: DirectorCheckHowStartEnded, psReads: []int{checkStarterPID}}})
	})
}

// AC3, the rows of a running launch herdr: the agent get, then, for
// agent_not_found only, the pane by its labels. The starter is not read.
func TestCheckDirector_HerdrRunningRows(t *testing.T) {
	tabLabel, wsLabel := DirectorTabLabel(directorTestToken), DirectorWorkspaceLabel(directorTestToken)
	notFound := func(f *dirCheckFixture) { f.h.agentGetErr = checkAgentNotFound() }
	paneCalls := []string{"agent_get", "pane_get", "tab_get", "workspace_get"}
	running := checkHerdrRunning()
	noPane := []DirectorEvent{checkAttached(DirectorLaunchHerdr, true), checkWorkspace("d1", "ws-1"), checkEvent("d1", DirectorEventLaunched, nil)}
	runCheckRows(t, []checkRow{
		{name: "agent_found", events: running,
			state: DirectorCheckAlive, how: DirectorCheckHowAgentFound, herdrCalls: []string{"agent_get"}},
		{name: "agent_ended", events: running, setup: func(f *dirCheckFixture) { notFound(f); f.herdrHas("pane-1", tabLabel, "ws-1", wsLabel) },
			state: DirectorCheckEnded, how: DirectorCheckHowAgentEnded, herdrCalls: paneCalls},
		{name: "pane_closed", events: running, setup: notFound,
			state: DirectorCheckEnded, how: DirectorCheckHowPaneClosed, herdrCalls: []string{"agent_get", "pane_get"}},
		{name: "no pane recorded reads as pane_closed", events: noPane, setup: notFound,
			state: DirectorCheckEnded, how: DirectorCheckHowPaneClosed, herdrCalls: []string{"agent_get"}},
		{name: "tab label differs", events: running, setup: func(f *dirCheckFixture) { notFound(f); f.herdrHas("pane-1", "someone-else", "ws-1", wsLabel) },
			state: DirectorCheckUnknown, how: DirectorCheckHowPaneUnconfirmed, detail: `herdr has it in tab "tab-pane-1" labelled "someone-else"`,
			herdrCalls: []string{"agent_get", "pane_get", "tab_get"}},
		{name: "workspace label differs", events: running, setup: func(f *dirCheckFixture) { notFound(f); f.herdrHas("pane-1", tabLabel, "ws-1", "orgA") },
			state: DirectorCheckUnknown, how: DirectorCheckHowPaneUnconfirmed, detail: `herdr labels it "orgA"`, herdrCalls: paneCalls},
		{name: "pane get fails", events: running,
			setup: func(f *dirCheckFixture) {
				notFound(f)
				f.h.getErrs = map[string]error{"pane-1": errors.New("herdr: connection refused")}
			},
			state: DirectorCheckUnknown, how: DirectorCheckHowPaneUnconfirmed, detail: "herdr: connection refused", herdrCalls: []string{"agent_get", "pane_get"}},
		{name: "agent get times out", events: running,
			setup: func(f *dirCheckFixture) { f.h.agentGetBlock, f.o.DriverCallTimeout = true, 20*time.Millisecond },
			state: DirectorCheckUnknown, how: DirectorCheckHowAgentGetFailed, detail: "context deadline exceeded", herdrCalls: []string{"agent_get"}},
		{name: "agent get fails otherwise", events: running, setup: func(f *dirCheckFixture) { f.h.agentGetErr = errors.New("herdr: no server is running") },
			state: DirectorCheckUnknown, how: DirectorCheckHowAgentGetFailed, detail: "herdr agent get director_0123456789ab failed: herdr: no server is running",
			herdrCalls: []string{"agent_get"}},
		{name: "a pane_not_found from agent get is a failure, not an ended agent", events: running,
			setup: func(f *dirCheckFixture) {
				f.h.agentGetErr = fmt.Errorf("herdr agent get: %w", driver.NewHerdrError(driver.HerdrCodePaneNotFound, "pane w1:p1 not found"))
			},
			state: DirectorCheckUnknown, how: DirectorCheckHowAgentGetFailed, detail: "pane_not_found", herdrCalls: []string{"agent_get"}},
		{name: "control characters in the detail are escaped", events: running,
			setup: func(f *dirCheckFixture) { f.h.agentGetErr = errors.New("herdr: \x1b[31mred\r") },
			state: DirectorCheckUnknown, how: DirectorCheckHowAgentGetFailed, detail: `herdr: \x1b[31mred\r`, herdrCalls: []string{"agent_get"}},
	})
}

// AC3: without a herdr client that runs agent get, a running launch herdr
// is unknown, with no herdr call.
func TestCheckDirector_NoAgentGet(t *testing.T) {
	for name, herdr := range map[string]func(f *dirCheckFixture) HerdrClient{
		"a client without AgentGet": func(f *dirCheckFixture) HerdrClient { return f.h.fakeHerdr },
		"no client":                 func(*dirCheckFixture) HerdrClient { return nil },
	} {
		t.Run(name, func(t *testing.T) {
			runCheckRows(t, []checkRow{{name: "no_agent_get", events: checkHerdrRunning(),
				setup: func(f *dirCheckFixture) { f.o.Herdr = herdr(f) },
				state: DirectorCheckUnknown, how: DirectorCheckHowNoAgentGet, detail: "cannot run herdr agent get"}})
		})
	}
}

// AC3, the rows of a running launch cmd: the pid its launch script checked
// in with. Another start time is unknown, not ended as a stop reads it. The
// starter and herdr are not read.
func TestCheckDirector_CmdRunningRows(t *testing.T) {
	running := checkCmdRunning()
	reads := []int{checkAgentPID}
	runCheckRows(t, []checkRow{
		{name: "process_ended gone", events: running,
			state: DirectorCheckEnded, how: DirectorCheckHowProcessEnded, psReads: reads},
		{name: "process_ended zombie", events: running,
			setup: func(f *dirCheckFixture) {
				f.procs[checkAgentPID] = DirectorProcess{State: DirectorProcessZombie, StartTime: checkAgentStart}
			},
			state: DirectorCheckEnded, how: DirectorCheckHowProcessEnded, psReads: reads},
		{name: "pid_reused", events: running, setup: func(f *dirCheckFixture) { f.alive(checkAgentPID, checkOtherStart) },
			state: DirectorCheckUnknown, how: DirectorCheckHowPIDReused,
			detail: `pid 5151 started "Fri Oct 16 08:00:00 2026", not "Thu Oct 15 21:52:00 2026"`, psReads: reads},
		{name: "process_found", events: running, setup: func(f *dirCheckFixture) { f.alive(checkAgentPID, checkAgentStart) },
			state: DirectorCheckAlive, how: DirectorCheckHowProcessFound, psReads: reads},
		{name: "ps fails", events: running, setup: func(f *dirCheckFixture) { f.psErr = errors.New("ps: no answer") },
			state: DirectorCheckUnknown, how: DirectorCheckHowProcessUnreadable, detail: "pid 5151, which its launch script checked in with: ps: no answer", psReads: reads},
		{name: "ps answers a state it never gives", events: running,
			setup: func(f *dirCheckFixture) {
				f.procs[checkAgentPID] = DirectorProcess{State: "sleeping", StartTime: checkAgentStart}
			},
			state: DirectorCheckUnknown, how: DirectorCheckHowProcessUnreadable, detail: `as "sleeping"`, psReads: reads},
	})
}

// AC3: a registration that is not the current headless one is not checked:
// a promoted session's, a stopped one, one detached and still open, none,
// and no director.jsonl at all. Nothing is read with ps or herdr, nothing is
// written, and no lock is taken.
func TestCheckDirector_NothingToCheck(t *testing.T) {
	headless := checkHerdrRunning()
	rows := map[string][]DirectorEvent{
		"a promoted session's registration": {{TS: "2026-10-18T08:00:00Z", ID: "d1", Event: DirectorEventAttached, Driver: DirectorDriverClaude,
			Wake: []string{DirectorWakePull}, User: "u", Cwd: "/w"}},
		"a stopped registration":           append(slices.Clone(headless), checkEvent("d1", DirectorEventDetached, nil), checkEvent("d1", DirectorEventStopped, nil)),
		"a current registration stopped":   append(slices.Clone(headless), checkEvent("d1", DirectorEventStopped, nil)),
		"a detached registration, open":    append(slices.Clone(headless), checkEvent("d1", DirectorEventDetached, nil)),
		"an empty director.jsonl":          nil,
		"no director.jsonl (or state dir)": nil,
	}
	for name, events := range rows {
		t.Run(name, func(t *testing.T) {
			f := newDirCheckFixture(t)
			if name == "no director.jsonl (or state dir)" {
				f.store = NewDirectorStore(filepath.Join(t.TempDir(), "state"))
			} else {
				writeDirectorEvents(t, f.store, events...)
			}
			entries := dirEntriesOrMissing(t, filepath.Dir(f.store.Path()))
			res := f.check(t)
			if res.Checked || !reflect.DeepEqual(res, DirectorCheckResult{}) {
				t.Fatalf("CheckDirector = %+v, want nothing checked", res)
			}
			if len(f.h.calls) != 0 || len(f.psReads) != 0 {
				t.Fatalf("herdr calls %q and ps reads %v, want none", f.h.calls, f.psReads)
			}
			if got := dirEntriesOrMissing(t, filepath.Dir(f.store.Path())); !slices.Equal(got, entries) {
				t.Fatalf("state dir = %q after the check, %q before: it must create nothing (no director.lock)", got, entries)
			}
		})
	}
}

// dirEntriesOrMissing is dirEntryNames, or ["<missing>"] for a dir that does
// not exist.
func dirEntriesOrMissing(t *testing.T, dir string) []string {
	t.Helper()
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return []string{"<missing>"}
	}
	return dirEntryNames(t, dir)
}

// A director.jsonl that cannot be read, at the first read or under the lock
// after the probes, is CheckDirector's only error.
func TestCheckDirector_UnreadableDirectorFileIsAnError(t *testing.T) {
	unreadable := func(t *testing.T, f *dirCheckFixture) {
		if err := os.Remove(f.store.Path()); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(f.store.Path(), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("at the first read", func(t *testing.T) {
		f := newDirCheckFixture(t)
		writeDirectorEvents(t, f.store, checkHerdrRunning()...)
		unreadable(t, f)
		res, err := f.o.CheckDirector(f.store, checkNow)
		if err == nil || !strings.Contains(err.Error(), "org: director: read") || res.Checked {
			t.Fatalf("CheckDirector = %+v, %v; want the read error and nothing checked", res, err)
		}
		if len(f.h.calls) != 0 {
			t.Fatalf("herdr calls = %q, want none", f.h.calls)
		}
	})
	t.Run("under the lock", func(t *testing.T) {
		f := newDirCheckFixture(t)
		writeDirectorEvents(t, f.store, checkHerdrRunning()...)
		f.seam = func(int) { unreadable(t, f) }
		res, err := f.o.CheckDirector(f.store, checkNow)
		if err == nil || !strings.Contains(err.Error(), "org: director: read") || res.Checked {
			t.Fatalf("CheckDirector = %+v, %v; want the read error and nothing checked", res, err)
		}
		if !slices.Equal(f.h.calls, []string{"agent_get"}) {
			t.Fatalf("herdr calls = %q, want the agent get of the try that could not be confirmed", f.h.calls)
		}
	})
	t.Run("no store", func(t *testing.T) {
		f := newDirCheckFixture(t)
		if _, err := f.o.CheckDirector(nil, checkNow); err == nil {
			t.Fatal("CheckDirector(nil): want an error")
		}
	})
}

// AC3, the barrier (Codex's HIGH on the plan): the start of a starting
// registration lands its launched after the check read the registration and
// then ends, so ps finds its start process gone. The check must not keep
// start_ended: it reads the registration again under the lock, finds the
// launched, and checks it again as running, for herdr (agent_found) and for
// a launch cmd (process_found). The result carries the registration it was
// computed from, with its version.
func TestCheckDirector_LaunchedAfterTheReadIsNotReadAsStartEnded(t *testing.T) {
	t.Run(DirectorLaunchHerdr, func(t *testing.T) {
		f := newDirCheckFixture(t)
		writeDirectorEvents(t, f.store, checkAttached(DirectorLaunchHerdr, true), checkWorkspace("d1", "ws-1"), checkPane("d1", "pane-1"))
		f.alive(checkStarterPID, checkStarterStart)
		f.seam = func(attempt int) {
			if attempt != 1 {
				return
			}
			if still, err := f.store.MarkLaunched("d1", testDirectorWho); err != nil || !still {
				t.Fatalf("MarkLaunched = %v, %v", still, err)
			}
			delete(f.procs, checkStarterPID)
		}
		res := f.check(t)
		wantBarrierResult(t, f, res, DirectorCheckHowAgentFound)
		if !slices.Equal(f.h.calls, []string{"agent_get"}) || !slices.Equal(f.psReads, []int{checkStarterPID}) {
			t.Fatalf("herdr calls %q, ps reads %v; want the starter read by the first try and the agent get by the second", f.h.calls, f.psReads)
		}
	})
	t.Run(DirectorLaunchCmd, func(t *testing.T) {
		f := newDirCheckFixture(t)
		writeDirectorEvents(t, f.store, checkAttached(DirectorLaunchCmd, true))
		f.alive(checkStarterPID, checkStarterStart)
		f.seam = func(attempt int) {
			if attempt != 1 {
				return
			}
			checkIn := DirectorCmdCheckIn{ID: "d1", Token: directorTestToken, PID: checkAgentPID, StartTime: checkAgentStart, Who: testDirectorWho,
				ReadStarter: func(_ context.Context, pid int) (DirectorProcess, error) { return f.peek(pid) }}
			if err := f.store.MarkCmdLaunched(checkIn); err != nil {
				t.Fatalf("MarkCmdLaunched: %v", err)
			}
			f.alive(checkAgentPID, checkAgentStart)
			delete(f.procs, checkStarterPID)
		}
		res := f.check(t)
		wantBarrierResult(t, f, res, DirectorCheckHowProcessFound)
		if res.Registration.PID != checkAgentPID || len(f.h.calls) != 0 || !slices.Equal(f.psReads, []int{checkStarterPID, checkAgentPID}) {
			t.Fatalf("registration pid %d, herdr calls %q, ps reads %v; want the starter read by the first try and the checked-in pid by the second",
				res.Registration.PID, f.h.calls, f.psReads)
		}
	})
}

// wantBarrierResult asserts res is alive by how, from the second try, on the
// registration as it is now (running), with the version of that read.
func wantBarrierResult(t *testing.T, f *dirCheckFixture, res DirectorCheckResult, how string) {
	t.Helper()
	want := DirectorCheck{ID: "d1", State: DirectorCheckAlive, How: how, CheckedAt: checkNowUTC}
	if !res.Checked || res.Check != want {
		t.Fatalf("CheckDirector = checked %v, %+v; want %+v, not start_ended", res.Checked, res.Check, want)
	}
	if !slices.Equal(f.attempts, []int{1, 2}) {
		t.Fatalf("tries = %v, want 2: the first one's result must be thrown away", f.attempts)
	}
	cur := mustReadDirector(t, f.store).Current
	if cur == nil || cur.State != DirectorStateRunning || !reflect.DeepEqual(res.Registration, *cur) {
		t.Fatalf("result registration = %+v, want the running one it checked, %+v", res.Registration, cur)
	}
	wantVersionNow(t, f.store, res)
}

// wantVersionNow asserts res's version is the version director.jsonl holds
// now for its registration, as start's cleanup will compare it (D6).
func wantVersionNow(t *testing.T, s *DirectorStore, res DirectorCheckResult) {
	t.Helper()
	raw, err := s.readRaw()
	if err != nil {
		t.Fatal(err)
	}
	if now := directorVersionOf(raw, foldDirector(raw), res.Registration.ID); res.version != now {
		t.Fatalf("result version = %+v, want the version on disk, %+v", res.version, now)
	}
}

// AC3, changed: an event lands for the registration after the read on both
// tries, so the check gives up with unknown changed and the registration the
// second try read (its workspace recorded, its pane not yet).
func TestCheckDirector_ChangedOnBothTriesIsUnknown(t *testing.T) {
	f := newDirCheckFixture(t)
	writeDirectorEvents(t, f.store, checkAttached(DirectorLaunchHerdr, true))
	f.alive(checkStarterPID, checkStarterStart)
	f.seam = func(attempt int) {
		var err error
		switch attempt {
		case 1:
			err = f.store.RecordWorkspace("d1", "ws-1", testDirectorWho)
		case 2:
			err = f.store.RecordPane("d1", "pane-1", testDirectorWho)
		}
		if err != nil {
			t.Fatalf("record on try %d: %v", attempt, err)
		}
	}
	res := f.check(t)
	if !res.Checked || res.Check.State != DirectorCheckUnknown || res.Check.How != DirectorCheckHowChanged || res.Check.ID != "d1" ||
		res.Check.CheckedAt != checkNowUTC || !strings.Contains(res.Check.Detail, "on each of 2 tries") {
		t.Fatalf("CheckDirector = checked %v, %+v; want unknown changed", res.Checked, res.Check)
	}
	if !slices.Equal(f.attempts, []int{1, 2}) || !slices.Equal(f.psReads, []int{checkStarterPID, checkStarterPID}) || len(f.h.calls) != 0 {
		t.Fatalf("tries %v, ps reads %v, herdr calls %q; want two tries that each read the starter", f.attempts, f.psReads, f.h.calls)
	}
	if reg := res.Registration; reg.WorkspaceID != "ws-1" || reg.PaneID != "" || reg.State != DirectorStateStarting {
		t.Fatalf("result registration = %s, want the second try's read (ws-1, no pane, starting)", directorRegSummary(reg))
	}
}

// The version counts events for the checked registration and which one is
// current, nothing else: a stopped for an older registration, detached and
// still open, does not make the check try again.
func TestCheckDirector_AnotherRegistrationsEventIsNoChange(t *testing.T) {
	f := newDirCheckFixture(t)
	const token2 = "ba9876543210"
	writeDirectorEvents(t, f.store, append(append(checkHerdrRunning(), checkEvent("d1", DirectorEventDetached, nil)),
		checkAttachedAs("d2", token2, DirectorLaunchHerdr, true), checkWorkspace("d2", "ws-2"), checkPane("d2", "pane-2"),
		checkEvent("d2", DirectorEventLaunched, nil))...)
	f.seam = func(int) {
		if _, err := f.store.MarkStopped("d1", testDirectorWho); err != nil {
			t.Fatalf("MarkStopped(d1): %v", err)
		}
	}
	res := f.check(t)
	want := DirectorCheck{ID: "d2", State: DirectorCheckAlive, How: DirectorCheckHowAgentFound, CheckedAt: checkNowUTC}
	if !res.Checked || res.Check != want || !slices.Equal(f.attempts, []int{1}) {
		t.Fatalf("CheckDirector = %+v after tries %v; want %+v from one try", res.Check, f.attempts, want)
	}
	if !slices.Equal(f.h.agentGetCalls, []string{DirectorAgentName(token2)}) {
		t.Fatalf("agent gets = %q, want d2's only", f.h.agentGetCalls)
	}
	wantVersionNow(t, f.store, res)
}

// A registration replaced after the read (attach --replace by a promoted
// session) is not the current one at the second read, so the check throws
// its result away and finds nothing to check: the second try stops at its
// read, before the seam and the probes.
func TestCheckDirector_ReplacedAfterTheReadIsNothingToCheck(t *testing.T) {
	f := newDirCheckFixture(t)
	writeDirectorEvents(t, f.store, checkHerdrRunning()...)
	f.seam = func(attempt int) {
		if attempt == 1 {
			mustAttachDirector(t, f.store, DirectorAttach{Driver: DirectorDriverClaude, Replace: true, User: "carol", Cwd: "/w"})
		}
	}
	res := f.check(t)
	if res.Checked || !slices.Equal(f.attempts, []int{1}) || !slices.Equal(f.h.calls, []string{"agent_get"}) {
		t.Fatalf("CheckDirector = %+v after tries %v and herdr calls %q; want the first try's agent_found thrown away and nothing checked",
			res, f.attempts, f.h.calls)
	}
}

// directorVersionOf, which start's cleanup will compare under the lock (D6):
// equal for two reads with nothing new, and different once an event for the
// registration lands, or once another registration is current, but not for
// an event of another registration that leaves the current one as it is.
func TestDirectorVersionOf(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	writeDirectorEvents(t, s, checkAttachedAs("d1", directorTestToken, DirectorLaunchHerdr, false), checkEvent("d1", DirectorEventDetached, nil),
		checkAttachedAs("d2", "ba9876543210", DirectorLaunchHerdr, false))
	version := func(id string) directorVersion {
		raw, err := s.readRaw()
		if err != nil {
			t.Fatal(err)
		}
		return directorVersionOf(raw, foldDirector(raw), id)
	}
	v := version("d2")
	if again := version("d2"); again != v || v.currentID != "d2" || v.events != 1 || v.folded == "" {
		t.Fatalf("version of d2 = %+v, then %+v; want one event, current, folded, and equal reads", v, again)
	}
	if _, err := s.MarkStopped("d1", testDirectorWho); err != nil {
		t.Fatal(err)
	}
	if now := version("d2"); now != v {
		t.Fatalf("version of d2 = %+v after d1's stopped, want %+v", now, v)
	}
	if err := s.RecordWorkspace("d2", "ws-2", testDirectorWho); err != nil {
		t.Fatal(err)
	}
	afterWorkspace := version("d2")
	if afterWorkspace == v || afterWorkspace.events != 2 {
		t.Fatalf("version of d2 = %+v after its workspace_created, want two events and a change from %+v", afterWorkspace, v)
	}
	if _, err := s.Detach(DirectorDetach{Force: true, User: "u", Cwd: "/w"}); err != nil {
		t.Fatal(err)
	}
	if now := version("d2"); now == afterWorkspace || now.currentID != "" {
		t.Fatalf("version of d2 = %+v after its detach, want no current registration and a change", now)
	}
}

// DirectorCheck's JSON tags give five keys, id, state, how, checked_at, and
// detail, detail included when empty. Only this test marshals a
// DirectorCheck: director-alive.json is DirectorAlive (six keys,
// TestDirectorAlive_WriteAndReadBack), and status's director.check is the
// CLI's orgDirectorReportCheck (four keys).
func TestDirectorCheck_JSON(t *testing.T) {
	raw, err := json.Marshal(DirectorCheck{ID: "d3", State: DirectorCheckEnded, How: DirectorCheckHowAgentEnded, CheckedAt: checkNowUTC})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":"d3","state":"ended","how":"agent_ended","checked_at":"2026-10-18T09:30:00Z","detail":""}`
	if string(raw) != want {
		t.Fatalf("JSON = %s, want %s", raw, want)
	}
}

// The mark (D4): director-alive.json holds the check and the attached_at of
// the registration it checked, under the keys id, attached_at, state, how,
// checked_at, and detail, mode 0644. A second write replaces the first, and
// the reader reads back what was written; Check gives the check without the
// attached_at.
func TestDirectorAlive_WriteAndReadBack(t *testing.T) {
	dir := t.TempDir()
	res := DirectorCheckResult{
		Checked:      true,
		Check:        DirectorCheck{ID: "d2", State: DirectorCheckEnded, How: DirectorCheckHowAgentEnded, CheckedAt: checkNowUTC},
		Registration: DirectorRegistration{ID: "d2", AttachedAt: "2026-10-18T08:00:00Z", Launch: DirectorLaunchHerdr},
	}
	mark := directorAliveOf(res)
	want := DirectorAlive{ID: "d2", AttachedAt: "2026-10-18T08:00:00Z", State: DirectorCheckEnded, How: DirectorCheckHowAgentEnded, CheckedAt: checkNowUTC}
	if mark != want || mark.Check() != res.Check {
		t.Fatalf("directorAliveOf = %+v (check %+v), want %+v (check %+v)", mark, mark.Check(), want, res.Check)
	}
	if err := writeDirectorAlive(dir, mark); err != nil {
		t.Fatalf("writeDirectorAlive: %v", err)
	}
	raw, err := os.ReadFile(DirectorAlivePathIn(dir))
	if err != nil {
		t.Fatal(err)
	}
	wantFile := "{\n" +
		`  "id": "d2",` + "\n" +
		`  "attached_at": "2026-10-18T08:00:00Z",` + "\n" +
		`  "state": "ended",` + "\n" +
		`  "how": "agent_ended",` + "\n" +
		`  "checked_at": "2026-10-18T09:30:00Z",` + "\n" +
		`  "detail": ""` + "\n" +
		"}\n"
	if string(raw) != wantFile {
		t.Fatalf("director-alive.json =\n%s\nwant\n%s", raw, wantFile)
	}
	if DirectorAlivePathIn(dir) != filepath.Join(dir, "director-alive.json") {
		t.Fatalf("DirectorAlivePathIn = %s", DirectorAlivePathIn(dir))
	}
	info, err := os.Stat(DirectorAlivePathIn(dir))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o644 {
		t.Errorf("mark mode = %v, want 0644", perm)
	}
	got, err := ReadDirectorAlive(dir)
	if err != nil || got == nil || *got != want {
		t.Fatalf("ReadDirectorAlive = %+v, %v; want %+v", got, err, want)
	}

	next := DirectorAlive{ID: "d3", AttachedAt: "2026-10-18T09:00:00Z", State: DirectorCheckUnknown, How: DirectorCheckHowAgentGetFailed,
		CheckedAt: checkNowUTC, Detail: "herdr agent get director_x failed"}
	if err := writeDirectorAlive(dir, next); err != nil {
		t.Fatalf("writeDirectorAlive: %v", err)
	}
	if got, err := ReadDirectorAlive(dir); err != nil || got == nil || *got != next {
		t.Fatalf("ReadDirectorAlive after a second write = %+v, %v; want %+v", got, err, next)
	}
	if entries := dirEntryNames(t, dir); !slices.Equal(entries, []string{DirectorAliveFileName}) {
		t.Fatalf("state dir = %q, want only the mark: no temporary file left", entries)
	}
}

// The reader: no mark, in a state dir or none at all, is nil with no error;
// a mark that does not parse, is not an object, or cannot be read (a
// directory in its place) is an error and no mark. Values are kept as read,
// a control character in the detail of a hand-edited mark included, for the
// printers to escape.
func TestReadDirectorAlive_MissingMalformedAndAsRead(t *testing.T) {
	for _, dir := range []string{t.TempDir(), filepath.Join(t.TempDir(), "absent")} {
		if mark, err := ReadDirectorAlive(dir); mark != nil || err != nil {
			t.Fatalf("ReadDirectorAlive(%s) with no mark = %+v, %v; want nil, nil", dir, mark, err)
		}
	}
	for name, content := range map[string]string{"not JSON": "{", "an array": "[]", "a number for the id": `{"id": 2}`} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(DirectorAlivePathIn(dir), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			mark, err := ReadDirectorAlive(dir)
			if mark != nil || err == nil || !strings.HasPrefix(err.Error(), "org: director: parse "+DirectorAlivePathIn(dir)+": ") {
				t.Fatalf("ReadDirectorAlive of %q = %+v, %v; want nil and the parse error", content, mark, err)
			}
		})
	}
	t.Run("a directory in its place", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(DirectorAlivePathIn(dir), 0o755); err != nil {
			t.Fatal(err)
		}
		mark, err := ReadDirectorAlive(dir)
		if mark != nil || err == nil || !strings.HasPrefix(err.Error(), "org: director: read "+DirectorAlivePathIn(dir)+": ") {
			t.Fatalf("ReadDirectorAlive of a directory = %+v, %v; want nil and the read error", mark, err)
		}
	})
	t.Run("values as read", func(t *testing.T) {
		dir := t.TempDir()
		hand := `{"id":"d1","attached_at":"t","state":"weird","how":"h","checked_at":"c","detail":"a\u001b[2Jb","extra":1}`
		if err := os.WriteFile(DirectorAlivePathIn(dir), []byte(hand), 0o644); err != nil {
			t.Fatal(err)
		}
		mark, err := ReadDirectorAlive(dir)
		want := DirectorAlive{ID: "d1", AttachedAt: "t", State: "weird", How: "h", CheckedAt: "c", Detail: "a\x1b[2Jb"}
		if err != nil || mark == nil || *mark != want {
			t.Fatalf("ReadDirectorAlive = %+v, %v; want %+v", mark, err, want)
		}
	})
}

// A write that fails (a non-empty directory where the mark goes, so the
// rename fails) is an error and leaves no temporary file.
func TestWriteDirectorAlive_AFailedWriteLeavesNoTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(DirectorAlivePathIn(dir), "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := writeDirectorAlive(dir, DirectorAlive{ID: "d1", State: DirectorCheckAlive})
	if err == nil || !strings.HasPrefix(err.Error(), "org: write "+DirectorAlivePathIn(dir)+": ") {
		t.Fatalf("writeDirectorAlive over a non-empty directory = %v, want the write error", err)
	}
	if entries := dirEntryNames(t, dir); !slices.Equal(entries, []string{DirectorAliveFileName}) {
		t.Fatalf("state dir after a failed write = %q, want no temporary file left", entries)
	}
}

// DirectorAliveFor (D8): a mark counts only for the registration whose id
// and attached_at it holds; one of another id, of the same id attached at
// another time (director.jsonl removed and the id given again), no mark, and
// no registration are nil.
func TestDirectorAliveFor(t *testing.T) {
	reg := &DirectorRegistration{ID: "d2", AttachedAt: "2026-10-18T08:00:00Z"}
	mark := &DirectorAlive{ID: "d2", AttachedAt: "2026-10-18T08:00:00Z", State: DirectorCheckAlive, How: DirectorCheckHowAgentFound}
	if got := DirectorAliveFor(mark, reg); got != mark {
		t.Fatalf("DirectorAliveFor of its own registration = %+v, want the mark", got)
	}
	for name, tc := range map[string]struct {
		mark *DirectorAlive
		reg  *DirectorRegistration
	}{
		"another id":               {&DirectorAlive{ID: "d1", AttachedAt: mark.AttachedAt}, reg},
		"another attached_at":      {&DirectorAlive{ID: "d2", AttachedAt: "2026-10-17T08:00:00Z"}, reg},
		"no attached_at in a mark": {&DirectorAlive{ID: "d2"}, reg},
		"no mark":                  {nil, reg},
		"no registration":          {mark, nil},
	} {
		if got := DirectorAliveFor(tc.mark, tc.reg); got != nil {
			t.Errorf("%s: DirectorAliveFor = %+v, want nil", name, got)
		}
	}
}
