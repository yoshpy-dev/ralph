package org

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// `ralph org reseat --restart-herdr` at the org layer (spec
// docs/specs/2026-11-20-org-reseat-restart-herdr.md, FR-1 to FR-5b; plan
// docs/plans/archive/2026-11-20-org-reseat-restart-herdr.md, S2: AC1 to AC5's
// retry, D1 to D4). On the fixture of reseat_test.go (newReseatOrg: auth-core
// with its leader, implementer and reviewer on the fake herdr), with a fake
// herdr server (restartHerdrServer) whose status reads and gate runs are
// written into the fake herdr's call log, so a test reads one order of every
// call.

// restartProbeAnswer is one answer of the fake herdr server's status read.
type restartProbeAnswer struct {
	running bool
	err     error
}

var (
	serverRunning = restartProbeAnswer{running: true}
	serverStopped = restartProbeAnswer{}
)

// restartHerdrServer is the herdr server of a --restart-herdr test: its
// probe (ReseatParams.HerdrServerProbe) answers probes in order and its gate
// (Org.HerdrServerGate) gates in order, each the last answer again once they
// run out. Each read and run is appended to the fake herdr's call log, as
// status:running, status:stopped or status:error, and herdr_server_gate.
type restartHerdrServer struct {
	t       *testing.T
	h       *fakeHerdr
	probes  []restartProbeAnswer
	gates   []error
	nProbes int
	nGates  int
}

func (s *restartHerdrServer) probe() (bool, error) {
	s.h.mu.Lock()
	defer s.h.mu.Unlock()
	if len(s.probes) == 0 {
		s.t.Fatal("the reseat read the herdr server's status, and the test gave no answer")
	}
	a := s.probes[min(s.nProbes, len(s.probes)-1)]
	s.nProbes++
	switch {
	case a.err != nil:
		s.h.calls = append(s.h.calls, "status:error")
	case a.running:
		s.h.calls = append(s.h.calls, "status:running")
	default:
		s.h.calls = append(s.h.calls, "status:stopped")
	}
	return a.running, a.err
}

func (s *restartHerdrServer) gate() error {
	s.h.mu.Lock()
	defer s.h.mu.Unlock()
	err := s.gates[min(s.nGates, len(s.gates)-1)]
	s.nGates++
	s.h.calls = append(s.h.calls, "herdr_server_gate")
	return err
}

// newRestartReseat is newReseatOrg with panes that close as herdr closes
// them, the fake herdr server as the Org's herdr server gate (gates; one
// that passes when none is given), and the wait for the server to stop
// shortened to 200ms, read every millisecond.
func newRestartReseat(t *testing.T, probes []restartProbeAnswer, gates ...error) (*featureStart, *restartHerdrServer) {
	t.Helper()
	st := newReseatOrg(t)
	st.h.closeRemoves = true
	if len(gates) == 0 {
		gates = []error{nil}
	}
	srv := &restartHerdrServer{t: t, h: st.h, probes: probes, gates: gates}
	st.o.HerdrServerGate = srv.gate
	setReseatHerdrStopWait(t, 200*time.Millisecond, time.Millisecond)
	return st, srv
}

// setReseatHerdrStopWait sets the wait for the server to stop and its read
// interval for the rest of t.
func setReseatHerdrStopWait(t *testing.T, wait, interval time.Duration) {
	t.Helper()
	oldWait, oldInterval := reseatHerdrStopWait, reseatHerdrStopPollInterval
	t.Cleanup(func() { reseatHerdrStopWait, reseatHerdrStopPollInterval = oldWait, oldInterval })
	reseatHerdrStopWait, reseatHerdrStopPollInterval = wait, interval
}

// restartParams is a reseat of auth-core naming seats, with --restart-herdr
// and srv's probe.
func (st *featureStart) restartParams(srv *restartHerdrServer, seats ...string) ReseatParams {
	p := st.reseatParams(seats...)
	p.RestartHerdr, p.HerdrServerProbe = true, srv.probe
	return p
}

// addOtherHerdrPane puts a pane that is not a seat of auth-core in the fake
// herdr's view, in a workspace of its own, with label and cwd when each is
// not "".
func addOtherHerdrPane(h *fakeHerdr, paneID, label, cwd string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	setLabel(&h.paneTabs, paneID, fakeTabID(paneID))
	setLabel(&h.paneWorkspaces, paneID, "ws-other")
	if label != "" {
		setLabel(&h.paneLabels, paneID, label)
	}
	if cwd != "" {
		setLabel(&h.paneCwds, paneID, cwd)
	}
}

// herdrCallsFrom is a copy of the fake herdr's call log from index from.
func herdrCallsFrom(h *fakeHerdr, from int) []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.calls[from:])
}

// restartPhases is calls with each call that shows a step of the restart
// as its step (a pane or workspace close as close, a workspace create, tab
// create or pane split as create, the gate as gate, and the pane list, the
// server stop and the status reads as they are), every other call dropped,
// and runs of one step written once.
func restartPhases(calls []string) []string {
	var phases []string
	for _, c := range calls {
		phase := ""
		switch {
		case c == "pane_close" || c == "workspace_close":
			phase = "close"
		case c == "workspace_create" || c == "tab_create" || c == "pane_split":
			phase = "create"
		case c == "herdr_server_gate":
			phase = "gate"
		case c == "pane_list_all" || c == "server_stop" || strings.HasPrefix(c, "status:"):
			phase = c
		}
		if phase != "" && (len(phases) == 0 || phases[len(phases)-1] != phase) {
			phases = append(phases, phase)
		}
	}
	return phases
}

// assertNoRestartCall checks that calls hold none of the calls that stop a
// seat or the server: no pane or workspace close and no server stop.
func assertNoRestartCall(t *testing.T, calls []string, what string) {
	t.Helper()
	for _, c := range calls {
		if c == "pane_close" || c == "workspace_close" || c == "server_stop" {
			t.Errorf("%s called %s; want no seat and no server stopped (calls %q)", what, c, calls)
		}
	}
}

// restartRetryParams is the reseat of auth-core that the printed command
// retry runs: retryParams's words and --restart-herdr and
// --claude-config-dir, whose value is a plain word or two single quotes for
// "" (no value with a space).
func (st *featureStart) restartRetryParams(t *testing.T, retry string) ReseatParams {
	t.Helper()
	rest, ok := strings.CutPrefix(retry, "ralph org reseat --org-id auth-core")
	if !ok {
		t.Fatalf("retry %q is not a reseat of auth-core", retry)
	}
	p := st.reseatParams()
	fields := strings.Fields(rest)
	for i := 0; i < len(fields); i++ {
		if fields[i] == "--restart-herdr" {
			p.RestartHerdr = true
			continue
		}
		if i+1 == len(fields) {
			t.Fatalf("retry %q ends with the flag %q and no value", retry, fields[i])
		}
		value := fields[i+1]
		switch fields[i] {
		case "--seat":
			p.Seats = append(p.Seats, value)
		case "--driver":
			p.Driver = value
		case "--model":
			p.Model = value
		case "--config":
			p.HintConfigPath = value
		case "--claude-config-dir":
			p.ClaudeConfigDirGiven = true
			if value != "''" {
				p.ClaudeConfigDir = value
			}
		default:
			t.Fatalf("retry %q has the flag %q", retry, fields[i])
		}
		i++
	}
	return p
}

// TestReseat_RestartHerdr_StopsTheServerBetweenTheStopsAndTheStarts is AC1:
// with --restart-herdr on a server that holds only the org's seats, the
// reseat lists the panes, stops the seats, lists the panes again, stops the
// server with no call between that list and the stop, reads the status
// until it is not running, runs the gate, which starts the server again,
// and only then starts the seats, and records `reseated` with every seat
// started. Before all that, PlanReseat read the status once (the server
// runs) and ran the gate once (check 10). There is no retry and no DECISION.
func TestReseat_RestartHerdr_StopsTheServerBetweenTheStopsAndTheStarts(t *testing.T) {
	st, srv := newRestartReseat(t, []restartProbeAnswer{serverRunning, serverRunning, serverStopped})
	from, events := len(herdrCallsFrom(st.h, 0)), len(mustReadEvents(t, st.o))

	res := st.mustReseat(t, st.restartParams(srv))
	if !res.Plan.RestartsHerdr || !res.HerdrStopped || !res.HerdrRestarted || res.Retry != "" || res.Decision != "" {
		t.Errorf("restarts %v, stopped %v, restarted %v, retry %q, decision %q; want the server stopped and started again, "+
			"no retry and no DECISION", res.Plan.RestartsHerdr, res.HerdrStopped, res.HerdrRestarted, res.Retry, res.Decision)
	}
	calls := herdrCallsFrom(st.h, from)
	phases := restartPhases(calls)
	want := []string{"status:running", "gate", "pane_list_all", "close", "pane_list_all", "server_stop",
		"status:running", "status:stopped", "gate", "create"}
	if len(phases) < len(want) || !slices.Equal(phases[:len(want)], want) {
		t.Fatalf("the reseat's steps %q; want them to start %q", phases, want)
	}
	if slices.Contains(phases[len(want):], "server_stop") || slices.Contains(phases[len(want):], "close") {
		t.Errorf("the reseat's steps %q; want no stop after the starts began", phases)
	}
	stop := slices.Index(calls, "server_stop")
	if calls[stop-1] != "pane_list_all" {
		t.Errorf("the call right before herdr server stop is %q; want the second pane list (calls %q)", calls[stop-1], calls)
	}
	if srv.nProbes != 3 {
		t.Errorf("status reads = %d; want 3 (PlanReseat's, then two in the wait)", srv.nProbes)
	}
	added := mustReadEvents(t, st.o)[events:]
	assertReseated(t, added, "id="+res.Plan.ID+" seats=leader:started,implementer:started,reviewer:started")
	if last := added[len(added)-1]; last.Event != EventReseated {
		t.Errorf("the last event is %+v; want the reseated", last)
	}
}

// TestPlanReseat_RestartHerdr_RefusesASubsetOfTheActiveSeats is AC1b:
// --restart-herdr with --seat naming only some of the active seats is
// refused before anything is stopped, naming the active seats left out, and
// the server's status is not read. Naming every active seat passes.
func TestPlanReseat_RestartHerdr_RefusesASubsetOfTheActiveSeats(t *testing.T) {
	st, srv := newRestartReseat(t, []restartProbeAnswer{serverRunning})
	from := len(herdrCallsFrom(st.h, 0))
	manifest := fileOrAbsent(t, st.o.Manifest.Path())

	res := st.o.Reseat(st.restartParams(srv, "reviewer"))
	var refusal *reseatRefusal
	if !errors.As(res.Err, &refusal) || res.Seats != nil {
		t.Fatalf("Reseat = %v (seats %v); want a refusal before anything is stopped", res.Err, res.Seats)
	}
	want := `org: reseat of org_id "auth-core" is refused before anything is stopped: --restart-herdr stops the herdr server, ` +
		`which closes the panes of every seat of org_id "auth-core", so it takes every active seat of the org, and the active ` +
		`seats leader, implementer are not named with --seat: run it with no --seat, or name them too`
	if res.Err.Error() != want {
		t.Errorf("refusal\n%q\nwant\n%q", res.Err, want)
	}
	if srv.nProbes != 0 || srv.nGates != 0 {
		t.Errorf("status reads %d, gate runs %d; want none", srv.nProbes, srv.nGates)
	}
	assertNoRestartCall(t, herdrCallsFrom(st.h, from), "the refused reseat")
	if got := fileOrAbsent(t, st.o.Manifest.Path()); got != manifest {
		t.Error("the refused reseat changed the manifest")
	}

	plan, err := st.o.PlanReseat(st.restartParams(srv, "reviewer", implementerSeatID, LeaderIdentity))
	if err != nil || !plan.RestartsHerdr {
		t.Errorf("PlanReseat naming every active seat: restarts %v, %v; want it to pass", plan.RestartsHerdr, err)
	}
}

// TestReseat_RestartHerdr_RefusesAnotherPaneBeforeAnyStop is AC2: a herdr
// server that holds a pane other than the org's seats refuses the restart
// before anything is stopped, real or dry run: the refusal names each such
// pane with its id, label and cwd (a pane with neither says so), says how
// to tell the record of a pane herdr restored after an earlier stop and how
// to close it, and OnPlan is not called. Nothing is closed or stopped and
// the manifest is as it was.
func TestReseat_RestartHerdr_RefusesAnotherPaneBeforeAnyStop(t *testing.T) {
	const want = `org: reseat of org_id "auth-core" is refused before anything is stopped: --restart-herdr stops the ` +
		`herdr server, which closes every pane it holds, and the server holds 2 panes that are not panes of org_id ` +
		`"auth-core"'s active seats: w9:p1 (label "shell", cwd "/home/dev/other repo"); w9:p2 (no label, no cwd reported); ` +
		`a pane for which herdr pane process-info <pane_id> finds no process is only the record of a pane that herdr ` +
		`restored after an earlier herdr server stop, and herdr pane close <pane_id> closes it; the others (another org's ` +
		`seats, a headless director in herdr, a shell) end with the server, so run the reseat with --restart-herdr again ` +
		`once they are gone, or without it to start the seats on the server as it is`
	for _, dryRun := range []bool{false, true} {
		t.Run(map[bool]string{false: "real", true: "dry run"}[dryRun], func(t *testing.T) {
			st, srv := newRestartReseat(t, []restartProbeAnswer{serverRunning})
			addOtherHerdrPane(st.h, "w9:p1", "shell", "/home/dev/other repo")
			addOtherHerdrPane(st.h, "w9:p2", "", "")
			from := len(herdrCallsFrom(st.h, 0))
			manifest := fileOrAbsent(t, st.o.Manifest.Path())
			p := st.restartParams(srv)
			p.DryRun = dryRun
			onPlan := 0
			p.OnPlan = func(ReseatPlan) { onPlan++ }

			res := st.o.Reseat(p)
			var refusal *reseatRefusal
			if !errors.As(res.Err, &refusal) || res.Seats != nil {
				t.Fatalf("Reseat = %v (seats %v); want a refusal before anything is stopped", res.Err, res.Seats)
			}
			if res.Err.Error() != want {
				t.Errorf("refusal\n%q\nwant\n%q", res.Err, want)
			}
			if onPlan != 0 {
				t.Errorf("OnPlan ran %d times; want none for a refused restart", onPlan)
			}
			calls := herdrCallsFrom(st.h, from)
			assertNoRestartCall(t, calls, "the refused reseat")
			if n := countString(calls, "pane_list_all"); n != 1 {
				t.Errorf("pane lists = %d; want the first check's one", n)
			}
			if got := fileOrAbsent(t, st.o.Manifest.Path()); got != manifest {
				t.Error("the refused reseat changed the manifest")
			}
		})
	}
}

// TestPlanReseat_RestartHerdr_RefusedInAHerdrPane is AC3: --restart-herdr
// from any herdr pane, here one that is no seat of the org (which check 2
// would refuse first), is refused before anything is read or stopped. The
// same reseat without --restart-herdr passes.
func TestPlanReseat_RestartHerdr_RefusedInAHerdrPane(t *testing.T) {
	st, srv := newRestartReseat(t, []restartProbeAnswer{serverRunning})
	st.o.Getenv = func(key string) string {
		if key == herdrPaneIDEnv {
			return "w9:p7"
		}
		return ""
	}
	from := len(herdrCallsFrom(st.h, 0))

	res := st.o.Reseat(st.restartParams(srv))
	want := `org: reseat of org_id "auth-core" is refused before anything is stopped: --restart-herdr stops the herdr ` +
		`server, which closes every pane it holds, and this command runs in the herdr pane w9:p7: run ralph org reseat ` +
		`--restart-herdr from a shell outside herdr`
	if res.Err == nil || res.Err.Error() != want || res.Seats != nil {
		t.Fatalf("Reseat = %v (seats %v); want the refusal\n%q", res.Err, res.Seats, want)
	}
	if srv.nProbes != 0 || srv.nGates != 0 {
		t.Errorf("status reads %d, gate runs %d; want none", srv.nProbes, srv.nGates)
	}
	assertNoRestartCall(t, herdrCallsFrom(st.h, from), "the refused reseat")
	if _, err := st.o.PlanReseat(st.reseatParams()); err != nil {
		t.Errorf("PlanReseat without --restart-herdr from that pane: %v; want it to pass", err)
	}
}

// TestPlanReseat_RestartHerdr_RefusesAServerItCannotRestart covers FR-2's
// last case and the restart's needs: a server that is not running is
// refused with the advice to reseat without --restart-herdr, a status read
// that fails is refused with the probe's error, and neither runs the gate,
// which would start a stopped server. A reseat with no probe, and a real one
// with no herdr server gate, are refused too; a dry run needs no gate.
func TestPlanReseat_RestartHerdr_RefusesAServerItCannotRestart(t *testing.T) {
	errStatus := errors.New("org: reseat --restart-herdr: could not read the herdr server's status (socket refused)")
	for _, tc := range []struct {
		name   string
		probe  restartProbeAnswer
		change func(st *featureStart, p *ReseatParams)
		want   string
		is     error
	}{
		{name: "a server that is not running", probe: serverStopped,
			want: "--restart-herdr stops the herdr server and starts it again, and no herdr server is running: run the " +
				"same ralph org reseat without --restart-herdr, which starts the server in this command's environment (with " +
				"--claude-config-dir when it is given) before it starts the seats"},
		{name: "a status read that fails", probe: restartProbeAnswer{err: errStatus},
			want: "reseat --restart-herdr: could not read the herdr server's status (socket refused)", is: errStatus},
		{name: "no probe", probe: serverRunning, change: func(_ *featureStart, p *ReseatParams) { p.HerdrServerProbe = nil },
			want: "(ReseatParams.HerdrServerProbe is nil)"},
		{name: "no gate", probe: serverRunning, change: func(st *featureStart, _ *ReseatParams) { st.o.HerdrServerGate = nil },
			want: "(Org.HerdrServerGate is nil)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, srv := newRestartReseat(t, []restartProbeAnswer{tc.probe})
			p := st.restartParams(srv)
			if tc.change != nil {
				tc.change(st, &p)
			}
			from := len(herdrCallsFrom(st.h, 0))
			res := st.o.Reseat(p)
			const lead = `org: reseat of org_id "auth-core" is refused before anything is stopped: `
			if res.Err == nil || !strings.HasPrefix(res.Err.Error(), lead) || !strings.HasSuffix(res.Err.Error(), tc.want) {
				t.Fatalf("Reseat = %v; want a refusal ending %q", res.Err, tc.want)
			}
			if tc.is != nil && !errors.Is(res.Err, tc.is) {
				t.Errorf("refusal %v is not %v", res.Err, tc.is)
			}
			if srv.nGates != 0 {
				t.Errorf("gate runs = %d; want none", srv.nGates)
			}
			calls := herdrCallsFrom(st.h, from)
			assertNoRestartCall(t, calls, "the refused reseat")
			if slices.Contains(calls, "pane_list_all") {
				t.Errorf("the refused reseat listed the panes (calls %q); want it refused before", calls)
			}
		})
	}

	st, srv := newRestartReseat(t, []restartProbeAnswer{serverRunning})
	st.o.HerdrServerGate = nil
	p := st.restartParams(srv)
	p.DryRun = true
	if res := st.o.Reseat(p); res.Err != nil || !res.Plan.RestartsHerdr {
		t.Errorf("dry run with no gate: restarts %v, %v; want it to pass", res.Plan.RestartsHerdr, res.Err)
	}
}

// TestReseat_RestartHerdr_DryRunListsThePanesAndStopsNothing covers the dry
// run of a restart: it reads the status, lists the panes, and stops, starts
// and records nothing; its plan says the server would be restarted, and
// OnPlan runs once.
func TestReseat_RestartHerdr_DryRunListsThePanesAndStopsNothing(t *testing.T) {
	st, srv := newRestartReseat(t, []restartProbeAnswer{serverRunning})
	from := len(herdrCallsFrom(st.h, 0))
	manifest := fileOrAbsent(t, st.o.Manifest.Path())
	p := st.restartParams(srv)
	p.DryRun = true
	onPlan := 0
	p.OnPlan = func(ReseatPlan) { onPlan++ }

	res := st.o.Reseat(p)
	if res.Err != nil || !res.Plan.RestartsHerdr || !res.Plan.DryRun || res.Seats != nil || onPlan != 1 {
		t.Fatalf("dry run = %v (restarts %v, dry %v, seats %v, OnPlan %d); want a plan that restarts the server and one OnPlan",
			res.Err, res.Plan.RestartsHerdr, res.Plan.DryRun, res.Seats, onPlan)
	}
	if got := herdrCallsFrom(st.h, from); !slices.Equal(got, []string{"status:running", "status:running", "pane_list_all"}) {
		t.Errorf("dry run calls %q; want the restart check's and the dry-run gate's status reads, then the pane list", got)
	}
	if got := fileOrAbsent(t, st.o.Manifest.Path()); got != manifest {
		t.Error("the dry run changed the manifest")
	}
}

// TestReseat_RestartHerdr_APaneMadeWhileTheSeatsStopKeepsTheServer is AC4:
// a pane that appears while the reseat stops the seats is found by the
// second pane check, so the reseat does not stop the server, starts no seat,
// leaves the seats stopped, records `reseated`, and prints the retry, which
// repeats --restart-herdr and --claude-config-dir, since the server was not
// stopped.
func TestReseat_RestartHerdr_APaneMadeWhileTheSeatsStopKeepsTheServer(t *testing.T) {
	st, srv := newRestartReseat(t, []restartProbeAnswer{serverRunning})
	intrudeBeforeStop(t, LeaderIdentity, func() { addOtherHerdrPane(st.h, "w9:p3", "director", "/home/dev") })
	from, events := len(herdrCallsFrom(st.h, 0)), len(mustReadEvents(t, st.o))
	p := st.restartParams(srv)
	p.ClaudeConfigDir, p.ClaudeConfigDirGiven = "/accounts/claude b", true

	res := st.o.Reseat(p)
	want := "org: reseat " + res.Plan.ID + ` of org_id "auth-core" did not stop the herdr server and started no seat, and the ` +
		`seats it stopped are left stopped: listed again before the stop of the server, the server holds a pane that is ` +
		`not a pane of the seats this reseat stopped: w9:p3 (label "director", cwd "/home/dev"); ` + reseatOtherPanesHint
	if res.Err == nil || res.Err.Error() != want {
		t.Fatalf("Reseat error\n%v\nwant\n%q", res.Err, want)
	}
	if got := reseatOutcomes(res); got != "leader:stopped,implementer:stopped,reviewer:stopped" {
		t.Errorf("seats %s; want every seat stopped and none started", got)
	}
	if res.HerdrStopped || res.HerdrRestarted {
		t.Errorf("server stopped %v, restarted %v; want neither", res.HerdrStopped, res.HerdrRestarted)
	}
	wantRetry := "ralph org reseat --org-id auth-core --seat leader --seat implementer --seat reviewer --restart-herdr " +
		"--claude-config-dir '/accounts/claude b'"
	if res.Retry != wantRetry {
		t.Errorf("retry\n%q\nwant\n%q", res.Retry, wantRetry)
	}
	calls := herdrCallsFrom(st.h, from)
	if slices.Contains(calls, "server_stop") || countString(calls, "pane_list_all") != 2 || srv.nGates != 1 {
		t.Errorf("calls %q (gate runs %d); want two pane lists, no server stop, and only PlanReseat's gate", calls, srv.nGates)
	}
	assertReseated(t, mustReadEvents(t, st.o)[events:], "id="+res.Plan.ID+" seats=leader:stopped,implementer:stopped,reviewer:stopped")
}

// TestReseat_RestartHerdr_TheRetryAfterTheServerStartedLeavesOutRestartHerdr
// is AC4b: the server is stopped and started again, the leader starts, and
// the implementer's start fails. The printed retry names the implementer and
// the reviewer with --claude-config-dir and without --restart-herdr. Run as
// it is, it reads no status, lists no pane, stops no seat and not the
// server, and starts the two seats.
func TestReseat_RestartHerdr_TheRetryAfterTheServerStartedLeavesOutRestartHerdr(t *testing.T) {
	st, srv := newRestartReseat(t, []restartProbeAnswer{serverRunning, serverStopped})
	sentinel := errors.New("herdr: split refused")
	st.h.paneSplitErr = sentinel
	p := st.restartParams(srv)
	p.ClaudeConfigDir, p.ClaudeConfigDirGiven = "/accounts/b", true

	res := st.o.Reseat(p)
	if !errors.Is(res.Err, sentinel) || !res.HerdrStopped || !res.HerdrRestarted {
		t.Fatalf("Reseat = %v (server stopped %v, restarted %v); want the implementer's failed start after the restart",
			res.Err, res.HerdrStopped, res.HerdrRestarted)
	}
	if got := reseatOutcomes(res); got != "leader:started,implementer:start_failed,reviewer:stopped" {
		t.Errorf("seats %s; want the leader started, the implementer's start failed and the reviewer left stopped", got)
	}
	const wantRetry = "ralph org reseat --org-id auth-core --seat implementer --seat reviewer --claude-config-dir /accounts/b"
	if res.Retry != wantRetry {
		t.Fatalf("retry\n%q\nwant\n%q", res.Retry, wantRetry)
	}

	st.h.paneSplitErr = nil
	from, probes := len(herdrCallsFrom(st.h, 0)), srv.nProbes
	retry := st.restartRetryParams(t, res.Retry)
	if retry.RestartHerdr || !retry.ClaudeConfigDirGiven || retry.ClaudeConfigDir != "/accounts/b" {
		t.Fatalf("the retry's params %+v; want --claude-config-dir and no --restart-herdr", retry)
	}
	retry.HerdrServerProbe = srv.probe
	again := st.mustReseat(t, retry)
	if len(again.Plan.Stop) != 0 || reseatOutcomes(again) != "implementer:started,reviewer:started" || again.Plan.RestartsHerdr {
		t.Errorf("the retry stopped %q, left %s, restarts %v; want none stopped, both started, no restart",
			seatIDs(again.Plan.Stop), reseatOutcomes(again), again.Plan.RestartsHerdr)
	}
	calls := herdrCallsFrom(st.h, from)
	assertNoRestartCall(t, calls, "the retry")
	if slices.Contains(calls, "pane_list_all") || srv.nProbes != probes {
		t.Errorf("the retry listed the panes or read the status (calls %q, %d reads); want neither", calls, srv.nProbes-probes)
	}
}

// TestReseat_RestartHerdr_AServerThatDoesNotStop covers FR-4's wait: a server
// still reported running, or whose status cannot be read, when the wait ends
// leaves the seats stopped, starts none, does not run the gate again, and
// prints the retry with --restart-herdr, since the server may still run.
func TestReseat_RestartHerdr_AServerThatDoesNotStop(t *testing.T) {
	errStatus := errors.New("status: socket busy")
	for _, tc := range []struct {
		name string
		last restartProbeAnswer
		want string
	}{
		{"still running", serverRunning, "(the last read of its status: it reports the server running)"},
		{"status unreadable", restartProbeAnswer{err: errStatus}, "(the last read of its status: it could not be read: status: socket busy)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, srv := newRestartReseat(t, []restartProbeAnswer{serverRunning, tc.last})
			setReseatHerdrStopWait(t, 30*time.Millisecond, 5*time.Millisecond)

			res := st.o.Reseat(st.restartParams(srv))
			want := "org: reseat " + res.Plan.ID + ` of org_id "auth-core": the herdr server was not reported stopped within ` +
				"30ms of herdr server stop " + tc.want + ", so the reseat did not start it again and started no seat; the " +
				"server may still stop: herdr status server --json says whether it runs"
			if res.Err == nil || res.Err.Error() != want {
				t.Fatalf("Reseat error\n%v\nwant\n%q", res.Err, want)
			}
			if res.HerdrStopped || res.HerdrRestarted || srv.nGates != 1 || srv.nProbes < 3 {
				t.Errorf("stopped %v, restarted %v, gate runs %d, status reads %d; want neither, PlanReseat's gate only, "+
					"and the wait's reads", res.HerdrStopped, res.HerdrRestarted, srv.nGates, srv.nProbes)
			}
			if got := reseatOutcomes(res); got != "leader:stopped,implementer:stopped,reviewer:stopped" {
				t.Errorf("seats %s; want every seat left stopped", got)
			}
			if want := "ralph org reseat --org-id auth-core --seat leader --seat implementer --seat reviewer --restart-herdr"; res.Retry != want {
				t.Errorf("retry %q; want %q", res.Retry, want)
			}
		})
	}
}

// TestReseat_RestartHerdr_AStopThatFails covers a `herdr server stop` that
// fails: the reseat reads the status once, says what it read, and starts no
// seat. The retry repeats --restart-herdr unless that read found the server
// stopped; then the retry's gate starts it.
func TestReseat_RestartHerdr_AStopThatFails(t *testing.T) {
	errStop := errors.New("herdr server stop: refused")
	for _, tc := range []struct {
		name        string
		after       restartProbeAnswer
		wantStatus  string
		wantStopped bool
	}{
		{"the server runs", serverRunning, "it reports the server running", false},
		{"the server stopped anyway", serverStopped, "it reports the server not running", true},
		{"the status cannot be read", restartProbeAnswer{err: errors.New("socket busy")}, "it could not be read: socket busy", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, srv := newRestartReseat(t, []restartProbeAnswer{serverRunning, tc.after})
			st.h.serverStopErr = errStop

			res := st.o.Reseat(st.restartParams(srv))
			want := "org: reseat " + res.Plan.ID + ` of org_id "auth-core": herdr server stop failed (herdr server stop: refused), ` +
				"so the reseat started no seat; the herdr server's status read after it: " + tc.wantStatus
			if !errors.Is(res.Err, errStop) || res.Err.Error() != want {
				t.Fatalf("Reseat error\n%v\nwant\n%q", res.Err, want)
			}
			if res.HerdrStopped != tc.wantStopped || res.HerdrRestarted || srv.nProbes != 2 || srv.nGates != 1 {
				t.Errorf("stopped %v, restarted %v, status reads %d, gate runs %d; want %v, false, 2 and 1",
					res.HerdrStopped, res.HerdrRestarted, srv.nProbes, srv.nGates, tc.wantStopped)
			}
			wantRetry := "ralph org reseat --org-id auth-core --seat leader --seat implementer --seat reviewer"
			if !tc.wantStopped {
				wantRetry += " --restart-herdr"
			}
			if res.Retry != wantRetry {
				t.Errorf("retry %q; want %q", res.Retry, wantRetry)
			}
		})
	}
}

// TestReseat_RestartHerdr_AGateThatFailsAfterTheStop covers D2's second
// half and FR-5b: the server stopped, and the gate could not start it again.
// The reseat starts no seat, and its retry leaves out --restart-herdr and
// repeats --claude-config-dir (here ""), so the retry's gate starts the
// stopped server with it. Run once the gate passes, the retry stops nothing,
// not the server either, and starts every seat.
func TestReseat_RestartHerdr_AGateThatFailsAfterTheStop(t *testing.T) {
	errGate := errors.New("herdr server did not answer within 10s")
	st, srv := newRestartReseat(t, []restartProbeAnswer{serverRunning, serverStopped}, nil, errGate)
	p := st.restartParams(srv)
	p.ClaudeConfigDirGiven = true

	res := st.o.Reseat(p)
	want := "org: reseat " + res.Plan.ID + ` of org_id "auth-core": the herdr server stopped, and its start failed, so the ` +
		"reseat started no seat: herdr server did not answer within 10s"
	if !errors.Is(res.Err, errGate) || res.Err.Error() != want {
		t.Fatalf("Reseat error\n%v\nwant\n%q", res.Err, want)
	}
	if !res.HerdrStopped || res.HerdrRestarted {
		t.Errorf("stopped %v, restarted %v; want the server stopped and not started", res.HerdrStopped, res.HerdrRestarted)
	}
	const wantRetry = "ralph org reseat --org-id auth-core --seat leader --seat implementer --seat reviewer --claude-config-dir ''"
	if res.Retry != wantRetry {
		t.Fatalf("retry %q; want %q", res.Retry, wantRetry)
	}
	if got := reseatOutcomes(res); got != "leader:stopped,implementer:stopped,reviewer:stopped" {
		t.Errorf("seats %s; want every seat left stopped", got)
	}

	srv.gates = append(srv.gates, nil)
	from := len(herdrCallsFrom(st.h, 0))
	retry := st.restartRetryParams(t, res.Retry)
	if retry.RestartHerdr || !retry.ClaudeConfigDirGiven || retry.ClaudeConfigDir != "" {
		t.Fatalf("the retry's params %+v; want --claude-config-dir '' and no --restart-herdr", retry)
	}
	again := st.mustReseat(t, retry)
	if len(again.Plan.Stop) != 0 || reseatOutcomes(again) != "leader:started,implementer:started,reviewer:started" {
		t.Errorf("the retry stopped %q and left %s; want none stopped and all three started", seatIDs(again.Plan.Stop), reseatOutcomes(again))
	}
	calls := herdrCallsFrom(st.h, from)
	assertNoRestartCall(t, calls, "the retry")
	if phases := restartPhases(calls); len(phases) == 0 || phases[0] != "gate" {
		t.Errorf("the retry's steps %q; want its gate first, which starts the stopped server", phases)
	}
}

// TestReseat_RestartHerdr_APaneListThatCannotBeRead covers a pane list that
// fails: before the stops it is a refusal with nothing stopped; right before
// the server's stop it ends the reseat with the seats stopped, the server
// not stopped, and the retry with --restart-herdr.
func TestReseat_RestartHerdr_APaneListThatCannotBeRead(t *testing.T) {
	errList := errors.New("herdr pane list: socket refused")
	t.Run("before the stops", func(t *testing.T) {
		st, srv := newRestartReseat(t, []restartProbeAnswer{serverRunning})
		st.h.paneListAllErr = errList
		from := len(herdrCallsFrom(st.h, 0))
		res := st.o.Reseat(st.restartParams(srv))
		want := `org: reseat of org_id "auth-core" is refused before anything is stopped: --restart-herdr lists the herdr ` +
			"server's panes before it stops anything, and the list could not be read (herdr pane list: socket refused): the " +
			"server is stopped only once it holds no pane but the org's active seats"
		if !errors.Is(res.Err, errList) || res.Err.Error() != want || res.Seats != nil {
			t.Fatalf("Reseat = %v (seats %v); want the refusal\n%q", res.Err, res.Seats, want)
		}
		assertNoRestartCall(t, herdrCallsFrom(st.h, from), "the refused reseat")
	})
	t.Run("before the server's stop", func(t *testing.T) {
		st, srv := newRestartReseat(t, []restartProbeAnswer{serverRunning})
		intrudeBeforeStop(t, LeaderIdentity, func() {
			st.h.mu.Lock()
			st.h.paneListAllErr = errList
			st.h.mu.Unlock()
		})
		res := st.o.Reseat(st.restartParams(srv))
		want := "org: reseat " + res.Plan.ID + ` of org_id "auth-core" could not list the herdr server's panes again before ` +
			"it stops the server (herdr pane list: socket refused), so it did not stop the server and started no seat, and " +
			"the seats it stopped are left stopped"
		if !errors.Is(res.Err, errList) || res.Err.Error() != want {
			t.Fatalf("Reseat error\n%v\nwant\n%q", res.Err, want)
		}
		if slices.Contains(herdrCallsFrom(st.h, 0), "server_stop") || res.HerdrStopped {
			t.Error("the reseat stopped the server after a pane list it could not read")
		}
		if want := "ralph org reseat --org-id auth-core --seat leader --seat implementer --seat reviewer --restart-herdr"; res.Retry != want {
			t.Errorf("retry %q; want %q", res.Retry, want)
		}
	})
}

// TestReseat_WithoutRestartHerdrTheServerIsLeftAlone covers the plan's
// "without the restart request nothing changes": a reseat with a probe and
// no --restart-herdr reads no status, lists no pane and stops no server, and
// a failed start's retry repeats --claude-config-dir when it was given, with
// no --restart-herdr.
func TestReseat_WithoutRestartHerdrTheServerIsLeftAlone(t *testing.T) {
	st, srv := newRestartReseat(t, []restartProbeAnswer{serverRunning})
	p := st.reseatParams()
	p.HerdrServerProbe = srv.probe
	from := len(herdrCallsFrom(st.h, 0))

	res := st.mustReseat(t, p)
	calls := herdrCallsFrom(st.h, from)
	if res.Plan.RestartsHerdr || res.HerdrStopped || res.HerdrRestarted || srv.nProbes != 0 ||
		slices.Contains(calls, "pane_list_all") || slices.Contains(calls, "server_stop") {
		t.Errorf("restarts %v, stopped %v, restarted %v, status reads %d, calls %q; want no restart and no read",
			res.Plan.RestartsHerdr, res.HerdrStopped, res.HerdrRestarted, srv.nProbes, calls)
	}

	st.h.paneSplitErr = errors.New("herdr: split refused")
	p.ClaudeConfigDir, p.ClaudeConfigDirGiven = "/accounts/b", true
	res = st.o.Reseat(p)
	if want := "ralph org reseat --org-id auth-core --seat implementer --seat reviewer --claude-config-dir /accounts/b"; res.Retry != want {
		t.Errorf("retry %q; want %q", res.Retry, want)
	}
}

// TestReseatRetryHerdrFlags pins the herdr server words of a retry and
// where they go: after the seats and --driver and --model, before --config
// and --state-dir; two single quotes for a --claude-config-dir of "", and a
// value with a space quoted.
func TestReseatRetryHerdrFlags(t *testing.T) {
	plan := ReseatPlan{OrgID: "auth-core", Start: []ReseatSeat{{SeatID: implementerSeatID, Driver: "codex", Model: "gpt-x", Overridden: true}}}
	seats := []ReseatSeatResult{{SeatID: implementerSeatID, Outcome: ReseatSeatStartFailed}}
	for _, tc := range []struct {
		name  string
		herdr reseatRetryHerdr
		want  string
	}{
		{"none", reseatRetryHerdr{}, "ralph org reseat --org-id auth-core --seat implementer --driver codex --model gpt-x --config /c --state-dir /s"},
		{"restart", reseatRetryHerdr{restartHerdr: true},
			"ralph org reseat --org-id auth-core --seat implementer --driver codex --model gpt-x --restart-herdr --config /c --state-dir /s"},
		{"restart and a dir", reseatRetryHerdr{restartHerdr: true, claudeConfigDir: "/a b", claudeConfigDirGiven: true},
			"ralph org reseat --org-id auth-core --seat implementer --driver codex --model gpt-x --restart-herdr --claude-config-dir '/a b' --config /c --state-dir /s"},
		{"no dir, given", reseatRetryHerdr{claudeConfigDirGiven: true},
			"ralph org reseat --org-id auth-core --seat implementer --driver codex --model gpt-x --claude-config-dir '' --config /c --state-dir /s"},
		{"a dir not given is left out", reseatRetryHerdr{claudeConfigDir: "/a"},
			"ralph org reseat --org-id auth-core --seat implementer --driver codex --model gpt-x --config /c --state-dir /s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := reseatRetryCommandWithHerdr(plan, seats, "", tc.herdr, "/c", "/s"); got != tc.want {
				t.Errorf("retry\n%q\nwant\n%q", got, tc.want)
			}
		})
	}
	if got := reseatRetryCommandWithHerdr(plan, []ReseatSeatResult{{SeatID: implementerSeatID, Outcome: ReseatSeatStarted}}, "",
		reseatRetryHerdr{restartHerdr: true, claudeConfigDirGiven: true}, "", ""); got != "" {
		t.Errorf("retry with every seat started = %q; want none", got)
	}
}
