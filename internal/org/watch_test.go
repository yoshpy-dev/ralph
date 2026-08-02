package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// --- test fixtures (self-contained for this file: does not reuse
// spawn_test.go's fakeHerdr/fakeAgmsg, since those do not implement the
// AgentGet/History probes watch.go's watchHerdrProbe/watchAgmsgHistory
// consumption-side interfaces need) ---------------------------------------

// fakeWatchHerdr is a call-recording HerdrClient + watchHerdrProbe fake.
// AgentGetSeq/AgentGetErrSeq, keyed by target (herdrAgentName's convention),
// let a test script a per-cycle sequence of AgentGet outcomes for a
// specific seat/lead target; once a sequence is exhausted the last queued
// value repeats (a test that only cares about a constant value can supply
// exactly one entry).
type fakeWatchHerdr struct {
	mu sync.Mutex

	AgentGetSeq    map[string][]string
	AgentGetErrSeq map[string][]error

	sendKeysCalls    []sendKeysCall
	paneSendTextCall []paneSendTextCall
	agentGetCalls    []string
}

type sendKeysCall struct {
	paneID string
	keys   []string
}

type paneSendTextCall struct {
	paneID, text string
}

func (f *fakeWatchHerdr) WorkspaceCreate(_ context.Context, _, _ string) (string, error) {
	return "ws-1", nil
}

func (f *fakeWatchHerdr) TabCreate(_ context.Context, _, _, _ string) (string, error) {
	return "pane-1", nil
}

func (f *fakeWatchHerdr) AgentStart(_ context.Context, _, _, _ string, _ int, _ []string) (string, error) {
	return "agent-1", nil
}

func (f *fakeWatchHerdr) AgentWait(_ context.Context, _ string, _ []string, _ int) (string, error) {
	return "idle", nil
}

func (f *fakeWatchHerdr) PaneRead(_ context.Context, _ string, _ int) (string, error) {
	return "", nil
}

func (f *fakeWatchHerdr) PaneSendText(_ context.Context, paneID, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.paneSendTextCall = append(f.paneSendTextCall, paneSendTextCall{paneID: paneID, text: text})
	return nil
}

func (f *fakeWatchHerdr) PaneSendKeys(_ context.Context, paneID string, keys ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendKeysCalls = append(f.sendKeysCalls, sendKeysCall{paneID: paneID, keys: keys})
	return nil
}

func (f *fakeWatchHerdr) AgentGet(_ context.Context, target string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.agentGetCalls = append(f.agentGetCalls, target)

	if errs := f.AgentGetErrSeq[target]; len(errs) > 0 {
		err := errs[0]
		if len(errs) > 1 {
			f.AgentGetErrSeq[target] = errs[1:]
		}
		if err != nil {
			return "", err
		}
	}
	if outs := f.AgentGetSeq[target]; len(outs) > 0 {
		out := outs[0]
		if len(outs) > 1 {
			f.AgentGetSeq[target] = outs[1:]
		}
		return out, nil
	}
	return "ok", nil
}

// countKeys returns how many recorded PaneSendKeys calls contain want among
// their keys argv (e.g. "C-c" for Stop, "Enter" for Send).
func (f *fakeWatchHerdr) countKeys(want string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.sendKeysCalls {
		if slices.Contains(c.keys, want) {
			n++
		}
	}
	return n
}

// fakeWatchAgmsg is a call-recording AgmsgClient + watchAgmsgHistory fake.
type fakeWatchAgmsg struct {
	mu sync.Mutex

	HistorySeq map[string][]string
	// JoinErrSeq scripts a per-agentID sequence of Join outcomes, keyed by
	// agentID (same pop-front-then-stick-on-last-entry convention as
	// fakeWatchHerdr.AgentGetErrSeq): a nil entry means Join succeeds that
	// call. Once exhausted, the last queued entry repeats. Keyed by agentID
	// (not a flat sequence) because Spawn's own lead/seat Join calls and
	// ensureWatchdogJoined's watchdog-identity Join call all share this one
	// fake method -- a flat sequence would let an unrelated identity's Join
	// consume entries meant for "watchdog".
	JoinErrSeq map[string][]error

	joinCalls []watchJoinCall
	sendCalls []watchSendCall
	sendErr   error
	leaveErr  error
}

type watchJoinCall struct {
	team, agentID, agmsgType, projectPath string
}

// watchSendCall records one identity-level Agmsg.Send call -- this is what
// every ALERT-delivery assertion in this file now inspects (Bug 1 fix: the
// pulse layer and the on-demand watcher both send ALERTs via Agmsg.Send from
// the "watchdog" identity to "lead", not via the seat-steering Send verb's
// PaneSendText, since verb-Send silently drops the message when no lead SEAT
// was ever spawned -- the normal session-promoted-lead org shape).
type watchSendCall struct {
	team, from, to, message string
}

func (f *fakeWatchAgmsg) Send(_ context.Context, team, from, to, message string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendCalls = append(f.sendCalls, watchSendCall{team: team, from: from, to: to, message: message})
	return f.sendErr
}

func (f *fakeWatchAgmsg) Join(_ context.Context, team, agentID, agmsgType, projectPath string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.joinCalls = append(f.joinCalls, watchJoinCall{team: team, agentID: agentID, agmsgType: agmsgType, projectPath: projectPath})
	if errs := f.JoinErrSeq[agentID]; len(errs) > 0 {
		err := errs[0]
		if len(errs) > 1 {
			f.JoinErrSeq[agentID] = errs[1:]
		}
		return err
	}
	return nil
}

func (f *fakeWatchAgmsg) Leave(_ context.Context, _, _ string) error {
	return f.leaveErr
}

// watchdogAlerts filters a.sendCalls down to messages sent from the
// watchdog identity -- Spawn's own HELLO Send (Agmsg.Send from the spawning
// SeatID to lead) shares the same fake/log, so ALERT-count assertions must
// not conflate the two.
func watchdogAlerts(a *fakeWatchAgmsg) []watchSendCall {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []watchSendCall
	for _, c := range a.sendCalls {
		if c.from == watchdogIdentity {
			out = append(out, c)
		}
	}
	return out
}

// watchdogJoinCalls filters a.joinCalls down to the watchdog identity's own
// Join attempts -- Spawn's own lead/seat Join calls share the same fake/log,
// so ensureWatchdogJoined retry-count assertions must not conflate the two.
func watchdogJoinCalls(a *fakeWatchAgmsg) []watchJoinCall {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []watchJoinCall
	for _, c := range a.joinCalls {
		if c.agentID == watchdogIdentity {
			out = append(out, c)
		}
	}
	return out
}

func (f *fakeWatchAgmsg) History(_ context.Context, team, _ string, _ int) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if outs := f.HistorySeq[team]; len(outs) > 0 {
		out := outs[0]
		if len(outs) > 1 {
			f.HistorySeq[team] = outs[1:]
		}
		return out, nil
	}
	return "", nil
}

// fakeClock is an injectable, manually-advanced Clock (see spawn.go's Clock
// type) so every watch test drives condition boundaries deterministically,
// with no real sleeps.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func watchTestConfig() config.OrgConfig {
	return config.OrgConfig{
		DriverPool:     []string{"claude"},
		ModelPool:      []config.OrgModelPoolEntry{{Driver: "claude", Model: "sonnet"}},
		Roles:          map[string][]string{},
		MaxSeats:       10,
		Budget:         config.OrgBudgetConfig{SeatWallClockMinutes: 30, TotalWallClockMinutes: 120},
		DeadmanMinutes: 10,
		Watchdog: config.OrgWatchdogConfig{
			IntervalSeconds: 30, StallMinutes: 15, WatcherEnabled: true, WatcherModel: "haiku",
		},
	}
}

func testWatchOrg(t *testing.T) (*Org, *fakeWatchHerdr, *fakeWatchAgmsg, *fakeClock) {
	t.Helper()
	dir := t.TempDir()
	h := &fakeWatchHerdr{AgentGetSeq: map[string][]string{}, AgentGetErrSeq: map[string][]error{}}
	a := &fakeWatchAgmsg{HistorySeq: map[string][]string{}, JoinErrSeq: map[string][]error{}}
	clk := newFakeClock()
	o := &Org{
		Config:   watchTestConfig(),
		Manifest: NewManifestStoreAtPath(filepath.Join(dir, "manifest.jsonl")),
		Receipts: NewReceiptStoreAtPath(filepath.Join(dir, "receipts.jsonl")),
		Herdr:    h,
		Agmsg:    a,
		Now:      clk.Now,
	}
	return o, h, a, clk
}

func watchSpawnParams(orgID, seatID, role string) SpawnParams {
	return SpawnParams{
		OrgID: orgID, SeatID: seatID, Role: role, Driver: "claude", Model: "sonnet",
		Cwd: "/tmp/watch-seat-" + seatID, TimeoutMS: 5000, Scope: "test-scope",
	}
}

// newTestWatchRun builds a watchRun wired to o/gitStatus/escalateFn, with
// its own scratch StatusDir -- the test drives evaluateCycle directly
// (rather than going through RunWatch's real select/sleep loop) so a
// multi-cycle test needs no real time to elapse; only the fakeClock moves.
func newTestWatchRun(o *Org, gitStatus GitStatusFunc, escalateFn EscalateFunc, stderr *bytes.Buffer) (*watchRun, string, string) {
	dir, err := os.MkdirTemp("", "watch-status-*")
	if err != nil {
		panic(err)
	}
	statusPath := filepath.Join(dir, WatchStatusFileName("org-a"))
	escalationsPath := filepath.Join(dir, EscalationsRelName)
	if gitStatus == nil {
		gitStatus = func(string) (string, error) { return "", nil }
	}
	if escalateFn == nil {
		escalateFn = func(context.Context, string) error { return nil }
	}
	return &watchRun{
		org: o, cfg: o.Config, hooks: WatchHooks{},
		gitStatus: gitStatus, escalateFn: escalateFn, stderr: stderr,
		statusPath: statusPath, escalationsPath: escalationsPath,
	}, statusPath, escalationsPath
}

// --- AC-3: seat wall-clock budget cutoff ------------------------------------

func TestWatch_SeatBudgetCutoff_AtBoundary_NotBeforeThenCutoffThenDeduped(t *testing.T) {
	o, h, a, clk := testWatchOrg(t)
	o.Config.Watchdog.StallMinutes = 10000 // keep the stall condition out of the picture
	// Deliberately no lead seat spawned -- ALERT delivery must not depend on
	// one (Bug 1 regression; see the ALERT assertions below via watchdogAlerts(a)).
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	run, statusPath, _ := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	// Cycle 1: 29 minutes elapsed -- must NOT cut off yet.
	clk.Advance(29 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if st, _ := o.Status("org-a", false); !findSeatStatus(t, st.Seats, "seat-1").Active {
		t.Fatalf("expected seat-1 still active after 29m, got %+v", st.Seats)
	}

	// Cycle 2: exactly 30 minutes elapsed -- boundary itself must NOT cut off
	// ("not before", i.e. strictly greater than the limit).
	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	if st, _ := o.Status("org-a", false); !findSeatStatus(t, st.Seats, "seat-1").Active {
		t.Fatalf("expected seat-1 still active at exactly the 30m boundary, got %+v", st.Seats)
	}

	// Cycle 3: 30m + 1s -- must cut off now.
	clk.Advance(1 * time.Second)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 3: %v", err)
	}
	st, err := o.Status("org-a", false)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	seat1 := findSeatStatus(t, st.Seats, "seat-1")
	if seat1.Active {
		t.Fatalf("expected seat-1 stopped after exceeding budget, got %+v", st.Seats)
	}
	if !strings.Contains(seat1.Details, "watchdog_budget_cutoff seat_wall_clock=30m") {
		t.Fatalf("expected stopped Details to record the watchdog Reason, got %q", seat1.Details)
	}
	if n := h.countKeys("C-c"); n != 1 {
		t.Fatalf("expected exactly 1 Stop attempt (C-c), got %d", n)
	}
	alerts := watchdogAlerts(a)
	if n := len(alerts); n != 1 {
		t.Fatalf("expected exactly 1 ALERT sent, got %d: %+v", n, alerts)
	}
	if alerts[0].from != watchdogIdentity || alerts[0].to != LeadIdentity {
		t.Errorf("expected the ALERT to be sent from %q to %q, got from=%q to=%q", watchdogIdentity, LeadIdentity, alerts[0].from, alerts[0].to)
	}
	if err := protocol.ValidateText(alerts[0].message, 0); err != nil {
		t.Errorf("expected the generated ALERT to be protocol-conformant, got %v (text=%q)", err, alerts[0].message)
	}
	if !strings.Contains(alerts[0].message, "TYPE: ALERT") || !strings.Contains(alerts[0].message, "watchdog_budget_cutoff") {
		t.Errorf("expected ALERT body to carry TYPE: ALERT and the cutoff reason, got %q", alerts[0].message)
	}

	// Cycle 4 and 5 (3 total cycles past the cutoff): dedupe -- no second
	// Stop attempt, no second ALERT.
	for range 2 {
		clk.Advance(1 * time.Minute)
		if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
			t.Fatalf("dedupe cycle: %v", err)
		}
	}
	if n := h.countKeys("C-c"); n != 1 {
		t.Fatalf("expected still exactly 1 Stop attempt after 2 more cycles (dedupe), got %d", n)
	}
	if n := len(watchdogAlerts(a)); n != 1 {
		t.Fatalf("expected still exactly 1 ALERT after 2 more cycles (dedupe), got %d", n)
	}
}

// --- AC-3b: org total wall-clock budget cutoff ------------------------------

func TestWatch_TotalBudgetCutoff_CutsAllActiveSeats_OneOrgLevelAlert(t *testing.T) {
	o, h, a, clk := testWatchOrg(t)
	o.Config.Budget.TotalWallClockMinutes = 5
	o.Config.Budget.SeatWallClockMinutes = 1000 // avoid seat-level cutoff firing first
	// Deliberately no lead seat spawned -- ALERT delivery must not depend on
	// one (Bug 1 regression).

	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-1 failed: %+v", r)
	}
	clk.Advance(1 * time.Minute)
	if r := o.Spawn(watchSpawnParams("org-a", "seat-2", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-2 failed: %+v", r)
	}

	run, statusPath, _ := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	clk.Advance(10 * time.Minute) // org_start (seat-1's spawn, the earliest active seat) + ~10m > 5m total budget
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("evaluateCycle: %v", err)
	}

	st, err := o.Status("org-a", false)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	for _, seatID := range []string{"seat-1", "seat-2"} {
		s := findSeatStatus(t, st.Seats, seatID)
		if s.Active {
			t.Errorf("expected seat %q stopped by total-budget cutoff, still active: %+v", seatID, s)
		}
		if !strings.Contains(s.Details, "watchdog_total_budget_cutoff") {
			t.Errorf("expected seat %q Details to record total-budget reason, got %q", seatID, s.Details)
		}
	}
	if n := h.countKeys("C-c"); n != 2 {
		t.Fatalf("expected exactly 2 Stop attempts (one per seat), got %d", n)
	}
	if n := len(watchdogAlerts(a)); n != 1 {
		t.Fatalf("expected exactly 1 org-level ALERT, got %d: %+v", n, watchdogAlerts(a))
	}
	if !strings.Contains(watchdogAlerts(a)[0].message, "CONDITION: total_budget") {
		t.Errorf("expected the org-level ALERT to carry CONDITION: total_budget, got %q", watchdogAlerts(a)[0].message)
	}

	// A further cycle must not re-cut or re-alert (AC-3c, cutoff never
	// clears).
	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("second evaluateCycle: %v", err)
	}
	if n := h.countKeys("C-c"); n != 2 {
		t.Fatalf("expected still exactly 2 Stop attempts after a further cycle, got %d", n)
	}
	if n := len(watchdogAlerts(a)); n != 1 {
		t.Fatalf("expected still exactly 1 org-level ALERT after a further cycle, got %d", n)
	}
}

// TestWatch_TotalBudgetCutoff_NoActiveSeats_NoAlertNoEscalation_ThenCutsNewSeat
// pins the cross-review AR-2 fix: an org that has already aged past its
// total wall-clock budget but currently has zero active seats (e.g. every
// seat was already stopped, by a prior cutoff or normal completion) must
// produce no total-budget ALERT and register no AC-5 pending-alert deadman
// record -- there is nothing to cut off, so an ALERT would be a false
// positive, repeating every cycle forever. Once a new seat becomes active in
// that same, still-over-budget org, total-budget enforcement resumes
// normally.
func TestWatch_TotalBudgetCutoff_NoActiveSeats_NoAlertNoEscalation_ThenCutsNewSeat(t *testing.T) {
	o, _, a, clk := testWatchOrg(t)
	o.Config.Budget.TotalWallClockMinutes = 5
	o.Config.Budget.SeatWallClockMinutes = 1000 // avoid seat-level cutoff firing first
	o.Config.DeadmanMinutes = 5

	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-1 failed: %+v", r)
	}
	if r := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1", Reason: "test setup: leave the org with zero active seats"}); r.Err != nil {
		t.Fatalf("stop seat-1: %v", r.Err)
	}

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	// org_start (seat-1's spawn) + ~10m > 5m total budget, but zero active
	// seats -- must not ALERT.
	clk.Advance(10 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 0 {
		t.Fatalf("expected no total-budget ALERT for an org with zero active seats, got %d: %+v", n, watchdogAlerts(a))
	}
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected no pending deadman record for an org with zero active seats, got %+v", status.PendingAlerts)
	}

	// A second cycle, past DeadmanMinutes too (proving no pending alert was
	// ever registered to escalate): still no ALERT, no escalation.
	clk.Advance(6 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 0 {
		t.Fatalf("expected still no ALERT after a second cycle with zero active seats, got %d", n)
	}
	assertNoEscalations(t, escalationsPath)

	// A new active seat appears in the same, still-over-budget org: the
	// zero-active-seats guard must not silently disable enforcement once a
	// seat becomes active again -- the org is cut.
	if r := o.Spawn(watchSpawnParams("org-a", "seat-2", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-2 failed: %+v", r)
	}
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 3 (seat-2 active): %v", err)
	}
	st, err := o.Status("org-a", false)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if findSeatStatus(t, st.Seats, "seat-2").Active {
		t.Fatalf("expected seat-2 to be cut off by the still-exceeded total budget once it is the org's only active seat")
	}
	if n := len(watchdogAlerts(a)); n != 1 {
		t.Fatalf("expected exactly 1 total-budget ALERT once an active seat exists, got %d: %+v", n, watchdogAlerts(a))
	}
}

// --- AC-4: heartbeat stall (ALERT, no cutoff, recovers, re-fires) ----------

func TestWatch_Stall_AlertsNoCutoff_RecoversAndRefires(t *testing.T) {
	o, h, a, clk := testWatchOrg(t)
	o.Config.Budget.SeatWallClockMinutes = 1000 // keep budget out of the picture
	o.Config.Watchdog.StallMinutes = 5
	// Deliberately no lead seat spawned -- ALERT delivery must not depend on
	// one (Bug 1 regression).

	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetSeq[target] = []string{"same", "same", "same", "different", "same-again", "same-again"}

	run, statusPath, _ := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	// Cycle 1: baseline snapshot only (no previous value to compare against).
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 0 {
		t.Fatalf("expected no ALERT on the baseline cycle, got %d", n)
	}

	// Cycle 2: same raw text AND >5m elapsed since the (unchanged) manifest
	// TS -- stalled, exactly 1 ALERT.
	clk.Advance(10 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 1 {
		t.Fatalf("expected exactly 1 stall ALERT after cycle 2, got %d: %+v", n, watchdogAlerts(a))
	}
	if !strings.Contains(watchdogAlerts(a)[0].message, "CONDITION: stall") {
		t.Errorf("expected CONDITION: stall in ALERT body, got %q", watchdogAlerts(a)[0].message)
	}
	if n := h.countKeys("C-c"); n != 0 {
		t.Errorf("stall must never cut off the seat, got %d Stop attempts", n)
	}

	// Cycle 3: still stalled (dedupe) -- no second ALERT.
	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 3: %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 1 {
		t.Fatalf("expected still exactly 1 ALERT (dedupe) after cycle 3, got %d", n)
	}

	// Cycle 4: raw text changed ("different") -- recovers, clears the key.
	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 4: %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 1 {
		t.Fatalf("expected still exactly 1 ALERT after the recovery cycle, got %d", n)
	}

	// Cycle 5: baseline again ("same-again"), no previous match yet.
	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 5: %v", err)
	}

	// Cycle 6: matches cycle 5's snapshot again and still >5m stalled since
	// manifest TS -- re-fires (a second, distinct ALERT).
	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 6: %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 2 {
		t.Fatalf("expected a second ALERT after re-stalling, got %d: %+v", n, watchdogAlerts(a))
	}
}

// --- AC-4: process liveness --------------------------------------------------

func TestWatch_Liveness_AlertOnAgentGetError(t *testing.T) {
	o, h, a, _ := testWatchOrg(t)
	// Deliberately no lead seat spawned -- ALERT delivery must not depend on
	// one (Bug 1 regression).
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")}

	run, statusPath, _ := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("evaluateCycle: %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 1 {
		t.Fatalf("expected exactly 1 liveness ALERT, got %d: %+v", n, watchdogAlerts(a))
	}
	if !strings.Contains(watchdogAlerts(a)[0].message, "CONDITION: liveness") {
		t.Errorf("expected CONDITION: liveness in ALERT body, got %q", watchdogAlerts(a)[0].message)
	}
	if n := h.countKeys("C-c"); n != 0 {
		t.Errorf("liveness must never cut off the seat, got %d Stop attempts", n)
	}
}

// --- AC-4: scope change ------------------------------------------------------

func TestWatch_ScopeChange_AlertCarriesScopeText_NoCutoff(t *testing.T) {
	o, h, a, _ := testWatchOrg(t)
	// Deliberately no lead seat spawned -- ALERT delivery must not depend on
	// one (Bug 1 regression).
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	// Keyed by cwd (not a shared counter), matching the other per-seat pulse
	// condition tests' convention -- only seat-1's cwd flips from clean to
	// dirty on its second probe.
	seat1Cwd := watchSpawnParams("org-a", "seat-1", "worker").Cwd
	calls := map[string]int{}
	gitStatus := func(cwd string) (string, error) {
		calls[cwd]++
		if cwd != seat1Cwd || calls[cwd] == 1 {
			return "", nil // baseline: clean worktree
		}
		return " M internal/org/watch.go\n?? new_file.go\n", nil
	}

	run, statusPath, _ := newTestWatchRun(o, gitStatus, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1 (baseline): %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 0 {
		t.Fatalf("expected no ALERT on the baseline cycle, got %d", n)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2 (scope changed): %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 1 {
		t.Fatalf("expected exactly 1 scope-change ALERT, got %d: %+v", n, watchdogAlerts(a))
	}
	body := watchdogAlerts(a)[0].message
	if !strings.Contains(body, "CONDITION: scope_change") {
		t.Errorf("expected CONDITION: scope_change in ALERT body, got %q", body)
	}
	if !strings.Contains(body, "watch.go") || !strings.Contains(body, "new_file.go") {
		t.Errorf("expected the ALERT body to carry the git status --porcelain scope text, got %q", body)
	}
	if n := h.countKeys("C-c"); n != 0 {
		t.Errorf("scope change must never cut off the seat, got %d Stop attempts", n)
	}
}

// --- AC-5: deadman escalation -----------------------------------------------

func TestWatch_Deadman_NoActivity_EscalatesOnceAfterTimeout(t *testing.T) {
	o, h, _, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 5
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")} // liveness ALERT, sticky error

	var stderr bytes.Buffer
	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &stderr)
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises ALERT): %v", err)
	}
	assertNoEscalations(t, escalationsPath)

	clk.Advance(6 * time.Minute) // past DeadmanMinutes, no manifest/lead/history activity
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2 (deadman): %v", err)
	}
	lines := readJSONLFile(t, escalationsPath)
	if len(lines) != 1 {
		t.Fatalf("expected exactly 1 escalation line, got %d: %v", len(lines), lines)
	}
	if !strings.Contains(stderr.String(), "WATCHDOG ESCALATION") {
		t.Errorf("expected a stderr banner, got %q", stderr.String())
	}

	// A further cycle (condition still failing, already Active -- no new
	// ALERT/pending record) must not escalate a second time.
	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 3 (post-escalation): %v", err)
	}
	lines = readJSONLFile(t, escalationsPath)
	if len(lines) != 1 {
		t.Fatalf("expected escalation dedupe, still 1 line, got %d: %v", len(lines), lines)
	}
}

func TestWatch_Deadman_LeadActivity_PreventsEscalation(t *testing.T) {
	o, h, _, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 5
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")}

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises ALERT): %v", err)
	}

	// Simulate lead activity: a new manifest event recorded between cycles.
	if err := o.Manifest.Append(ManifestEvent{TS: clk.Now().UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: LeadIdentity, Event: EventSent, Details: "lead activity"}); err != nil {
		t.Fatalf("append lead-activity event: %v", err)
	}

	clk.Advance(6 * time.Minute) // past DeadmanMinutes, but lead was active
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	assertNoEscalations(t, escalationsPath)
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected the pending alert to clear after lead activity, got %+v", status.PendingAlerts)
	}
}

func TestWatch_Deadman_LeadIsAnomalySubject_EscalatesImmediately(t *testing.T) {
	o, _, _, _ := testWatchOrg(t)
	if r := o.Spawn(watchSpawnParams("org-a", LeadIdentity, LeadIdentity)); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn lead failed: %+v", r)
	}
	target := herdrAgentName("org-a", LeadIdentity)
	// Sticky liveness failure for the lead seat itself.
	fh := o.Herdr.(*fakeWatchHerdr)
	fh.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")}

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	// Same cycle: ALERT raised (subject=lead) then the deadman sweep
	// escalates immediately, without waiting for DeadmanMinutes.
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("evaluateCycle: %v", err)
	}
	lines := readJSONLFile(t, escalationsPath)
	if len(lines) != 1 {
		t.Fatalf("expected exactly 1 immediate escalation for a lead-subject anomaly, got %d: %v", len(lines), lines)
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("parse escalation line: %v", err)
	}
	if rec["reason"] != "lead_is_anomaly_subject" {
		t.Errorf("expected reason=lead_is_anomaly_subject, got %+v", rec)
	}
}

// --- watch-status.json heartbeat --------------------------------------------

func TestWatch_StatusFile_HeartbeatWrittenEveryCycle(t *testing.T) {
	o, _, _, clk := testWatchOrg(t)
	run, statusPath, _ := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	for i := range 3 {
		clk.Advance(time.Minute)
		if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
			t.Fatalf("cycle %d: %v", i+1, err)
		}
	}

	data, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatalf("read %s: %v", statusPath, err)
	}
	var onDisk watchStatusFile
	if err := json.Unmarshal(data, &onDisk); err != nil {
		t.Fatalf("parse %s: %v", statusPath, err)
	}
	if onDisk.OrgID != "org-a" || onDisk.Cycles != 3 {
		t.Fatalf("expected org_id=org-a cycles=3, got %+v", onDisk)
	}
	wantTS := clk.Now().UTC().Format(time.RFC3339)
	if onDisk.LastCycleTS != wantTS {
		t.Errorf("expected last_cycle_ts=%q, got %q", wantTS, onDisk.LastCycleTS)
	}
}

// --- RunWatch's public loop (Cycles-capped, no real per-interval sleep
// needed since Cycles > 0 returns before the loop ever selects on the
// ticker) --------------------------------------------------------------------

func TestRunWatch_Cycles_RunsExactlyNThenReturns(t *testing.T) {
	o, _, _, _ := testWatchOrg(t)
	dir := t.TempDir()

	var onCycleCount int
	err := o.RunWatch(context.Background(), WatchParams{
		OrgID: "org-a", Interval: time.Millisecond, Cycles: 2, StatusDir: dir,
	}, WatchHooks{OnCycle: func(int, time.Time) { onCycleCount++ }})
	if err != nil {
		t.Fatalf("RunWatch: %v", err)
	}
	if onCycleCount != 2 {
		t.Fatalf("expected exactly 2 cycles, got %d", onCycleCount)
	}

	data, err := os.ReadFile(filepath.Join(dir, WatchStatusFileName("org-a")))
	if err != nil {
		t.Fatalf("read status file: %v", err)
	}
	var status watchStatusFile
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("parse status file: %v", err)
	}
	if status.Cycles != 2 {
		t.Fatalf("expected cycles=2 in the persisted status, got %d", status.Cycles)
	}
}

func TestRunWatch_RequiresOrgIDAndStatusDir(t *testing.T) {
	o, _, _, _ := testWatchOrg(t)
	if err := o.RunWatch(context.Background(), WatchParams{StatusDir: t.TempDir()}, WatchHooks{}); err == nil {
		t.Fatal("expected an error for a blank org_id")
	}
	if err := o.RunWatch(context.Background(), WatchParams{OrgID: "org-a"}, WatchHooks{}); err == nil {
		t.Fatal("expected an error for a blank StatusDir")
	}
}

// TestRunWatch_MultipleOrgs_SeparateStatusFiles pins the self-review H-1 fix:
// two orgs watched from the same repository (one shared StatusDir, exactly
// as manifest.jsonl/model-receipts.jsonl are already shared there) must get
// distinct watch-status files, not silently clobber each other's
// OrgID/Cycles/WatchdogJoined/SeatSnapshots state via one fixed file name.
func TestRunWatch_MultipleOrgs_SeparateStatusFiles(t *testing.T) {
	o, _, _, _ := testWatchOrg(t)
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn org-a failed: %+v", r)
	}
	if r := o.Spawn(watchSpawnParams("org-b", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn org-b failed: %+v", r)
	}

	dir := t.TempDir()
	if err := o.RunWatch(context.Background(), WatchParams{OrgID: "org-a", Interval: time.Millisecond, Cycles: 1, StatusDir: dir}, WatchHooks{}); err != nil {
		t.Fatalf("RunWatch org-a: %v", err)
	}
	if err := o.RunWatch(context.Background(), WatchParams{OrgID: "org-b", Interval: time.Millisecond, Cycles: 1, StatusDir: dir}, WatchHooks{}); err != nil {
		t.Fatalf("RunWatch org-b: %v", err)
	}

	pathA := filepath.Join(dir, WatchStatusFileName("org-a"))
	pathB := filepath.Join(dir, WatchStatusFileName("org-b"))
	if pathA == pathB {
		t.Fatalf("expected distinct status file names for org-a/org-b, got %q for both", pathA)
	}
	for orgID, path := range map[string]string{"org-a": pathA, "org-b": pathB} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var status watchStatusFile
		if err := json.Unmarshal(data, &status); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		if status.OrgID != orgID {
			t.Errorf("expected org_id=%q recorded in %s, got %+v", orgID, path, status)
		}
		if status.Cycles != 1 {
			t.Errorf("expected cycles=1 recorded in %s (not clobbered by the other org's run), got %+v", path, status)
		}
	}
}

// TestWatch_SeatBudgetCutoff_StopFails_RetriesThenSucceeds pins the
// self-review H-2 fix: a Stop call whose `stopped` manifest write fails must
// not set the Cutoff ratchet -- the failure is logged to stderr and the next
// cycle retries Stop for the still-active seat, only ratcheting Cutoff once
// a retry actually succeeds. The ALERT is still sent exactly once regardless
// of the Stop outcome.
func TestWatch_SeatBudgetCutoff_StopFails_RetriesThenSucceeds(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: a read-only manifest file is still writable, so this permission-based failure injection does not apply")
	}
	o, h, a, clk := testWatchOrg(t)
	o.Config.Watchdog.StallMinutes = 10000 // keep the stall condition out of the picture
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	var stderr bytes.Buffer
	run, statusPath, _ := newTestWatchRun(o, nil, nil, &stderr)
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	clk.Advance(31 * time.Minute) // past the 30m seat budget

	// Make the manifest file unwritable so Stop's appendEvent (the `stopped`
	// event write) fails deterministically -- Stop's own PaneSendKeys/Leave
	// failures are captured as best-effort Details notes, never as
	// StopResult.Err (see verbs.go's Stop doc comment); only findSeat's read
	// or appendEvent's write can produce a non-nil Stop error.
	manifestPath := o.Manifest.Path()
	if err := os.Chmod(manifestPath, 0o444); err != nil {
		t.Fatalf("chmod manifest read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(manifestPath, 0o644) })

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1 (Stop's manifest write fails): %v", err)
	}
	if !strings.Contains(stderr.String(), "budget cutoff Stop failed") {
		t.Fatalf("expected a stderr line reporting the failed cutoff Stop, got %q", stderr.String())
	}
	key := conditionKey("org-a", "seat-1", condSeatBudget)
	if rec := status.Conditions[key]; rec == nil || rec.Cutoff {
		t.Fatalf("expected Cutoff to remain false after a failed Stop, got %+v", rec)
	}
	if n := len(watchdogAlerts(a)); n != 1 {
		t.Fatalf("expected exactly 1 ALERT even though Stop's manifest write failed, got %d", n)
	}
	if st, serr := o.Status("org-a", false); serr != nil || !findSeatStatus(t, st.Seats, "seat-1").Active {
		t.Fatalf("expected seat-1 to remain active after the failed Stop (status err=%v)", serr)
	}

	// Restore write permission -- the next cycle must retry Stop (the seat
	// is still active per the roster) and this time set Cutoff, without a
	// second ALERT.
	if err := os.Chmod(manifestPath, 0o644); err != nil {
		t.Fatalf("chmod manifest writable: %v", err)
	}
	clk.Advance(time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2 (Stop retries and succeeds): %v", err)
	}
	if rec := status.Conditions[key]; rec == nil || !rec.Cutoff {
		t.Fatalf("expected Cutoff to be set true after the retried Stop succeeds, got %+v", rec)
	}
	if n := len(watchdogAlerts(a)); n != 1 {
		t.Fatalf("expected still exactly 1 ALERT after the retry succeeds (deduped), got %d", n)
	}
	st, err := o.Status("org-a", false)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if findSeatStatus(t, st.Seats, "seat-1").Active {
		t.Fatalf("expected seat-1 to be stopped once the retried Stop succeeds")
	}
	if n := h.countKeys("C-c"); n != 2 {
		t.Fatalf("expected 2 Stop attempts (1 failed manifest write + 1 successful retry), got %d", n)
	}
}

// TestWatch_Deadman_WatchdogsOwnCutoffEvent_DoesNotClearPendingAlert pins
// the self-review M-4 fix: the deadman's manifest-growth "has anything
// happened since the ALERT" activity source must not count the watchdog's
// own cutoff of an unrelated seat as if it were genuine lead/seat activity.
func TestWatch_Deadman_WatchdogsOwnCutoffEvent_DoesNotClearPendingAlert(t *testing.T) {
	o, h, _, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 5
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-1 failed: %+v", r)
	}
	if r := o.Spawn(watchSpawnParams("org-a", "seat-2", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-2 failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")} // sticky liveness ALERT for seat-1

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises seat-1's liveness ALERT): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected exactly 1 pending alert after cycle 1, got %+v", status.PendingAlerts)
	}

	// Simulate the watchdog cutting off a DIFFERENT seat (seat-2) between
	// cycles, via the same Details shape evaluateSeatBudget/
	// evaluateTotalBudget produce ("reason=watchdog_..."). Before the M-4
	// fix, this alone grows the manifest enough to satisfy the deadman's
	// unfiltered len(rr.Events) > ManifestLen check and wrongly clears
	// seat-1's still-unanswered pending alert.
	if r := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-2", Reason: "watchdog_budget_cutoff seat_wall_clock=30m observed=31m0s"}); r.Err != nil {
		t.Fatalf("stop seat-2: %v", r.Err)
	}

	clk.Advance(6 * time.Minute) // past DeadmanMinutes, no genuine lead/seat activity
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2 (deadman sweep): %v", err)
	}

	lines := readJSONLFile(t, escalationsPath)
	if len(lines) != 1 {
		t.Fatalf("expected the pending alert to still escalate (the watchdog's own cutoff of seat-2 must not count as lead activity), got %d escalation(s): %v", len(lines), lines)
	}
}

// TestWatch_Deadman_CrossOrgActivity_DoesNotClearPendingAlert pins the
// cross-review AR-1 fix: leadActivityEventCount must be scoped to the
// watched org. A shared manifest store (RunWatch's normal shape: one
// manifest.jsonl for every org in the harness) means an unrelated org's own
// genuine activity must never clear a different, stalled org's pending
// deadman alert -- only activity recorded under the watched org's own OrgID
// may do so.
func TestWatch_Deadman_CrossOrgActivity_DoesNotClearPendingAlert(t *testing.T) {
	o, h, _, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 5
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-1 failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")} // sticky liveness ALERT for org-a/seat-1

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises org-a's liveness ALERT): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected exactly 1 pending alert after cycle 1, got %+v", status.PendingAlerts)
	}

	// A new event in a DIFFERENT org, sharing the same manifest store, must
	// not count as activity for org-a's pending alert.
	if err := o.Manifest.Append(ManifestEvent{TS: clk.Now().UTC().Format(time.RFC3339), OrgID: "org-b", SeatID: "seat-9", Event: EventSent, Details: "org-b genuine activity"}); err != nil {
		t.Fatalf("append org-b activity event: %v", err)
	}
	clk.Advance(1 * time.Minute) // still within DeadmanMinutes
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2 (org-b activity must not clear org-a's pending alert): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected org-a's pending alert to remain after unrelated org-b activity, got %+v", status.PendingAlerts)
	}
	assertNoEscalations(t, escalationsPath)

	// A lead-authored event recorded IN org-a itself DOES clear it.
	if err := o.Manifest.Append(ManifestEvent{TS: clk.Now().UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: LeadIdentity, Event: EventSent, Details: "lead activity"}); err != nil {
		t.Fatalf("append org-a lead-activity event: %v", err)
	}
	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 3 (org-a's own lead activity clears the pending alert): %v", err)
	}
	assertNoEscalations(t, escalationsPath)
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected the pending alert to clear after org-a's own lead activity, got %+v", status.PendingAlerts)
	}
}

// TestWatch_Deadman_SeatSentEvent_ClearsPendingAlert_WatchdogCutoffDoesNot
// pins the self-review cycle-3 H3-1 fix and replaces the earlier (inverted)
// TestWatch_Deadman_UnrelatedSeatEvent_DoesNotClearPendingAlert, which
// claimed a `sent` event whose SeatID names a different seat must NOT count
// as lead activity. That claim was backwards: ev.SeatID on a `sent` event is
// the *recipient* (Send writes SeatID: p.To -- verbs.go), not the author, and
// `ralph org send` -- the only verb that appends a `sent` event -- is only
// ever driven by lead/the operator (star topology,
// .claude/rules/agent-messaging.md: a seat only ever addresses TO: lead, and
// its replies travel over the agmsg skill, never through this manifest). So
// a `sent` event naming seat-2 as SeatID IS lead activity (lead sending to
// seat-2), and must clear seat-1's pending alert. The genuinely non-clearing
// case is the watchdog's OWN cutoff `stopped` write (reason=watchdog_...,
// already pinned by
// TestWatch_Deadman_WatchdogsOwnCutoffEvent_DoesNotClearPendingAlert above)
// -- this test reuses that same non-clearing shape for a THIRD seat before
// showing the `sent` event's positive, clearing case.
func TestWatch_Deadman_SeatSentEvent_ClearsPendingAlert_WatchdogCutoffDoesNot(t *testing.T) {
	o, h, _, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 5
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-1 failed: %+v", r)
	}
	if r := o.Spawn(watchSpawnParams("org-a", "seat-2", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-2 failed: %+v", r)
	}
	if r := o.Spawn(watchSpawnParams("org-a", "seat-3", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-3 failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")} // sticky liveness ALERT for seat-1

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises seat-1's liveness ALERT): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected exactly 1 pending alert after cycle 1, got %+v", status.PendingAlerts)
	}

	// The watchdog cutting off seat-3 (its own enforcement write, carrying
	// "reason=watchdog_...") must NOT count as lead activity.
	if r := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-3", Reason: "watchdog_budget_cutoff seat_wall_clock=30m observed=31m0s"}); r.Err != nil {
		t.Fatalf("stop seat-3: %v", r.Err)
	}
	clk.Advance(3 * time.Minute) // still within DeadmanMinutes
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2 (watchdog's own cutoff of seat-3 must not clear seat-1's alert): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected seat-1's pending alert to remain after the watchdog's own cutoff of seat-3, got %+v", status.PendingAlerts)
	}
	assertNoEscalations(t, escalationsPath)

	// A `sent` event naming seat-2 as SeatID (the recipient) -- e.g.
	// `ralph org send --to seat-2`, which only lead/the operator can run --
	// IS lead activity by construction and DOES clear seat-1's pending alert.
	if err := o.Manifest.Append(ManifestEvent{TS: clk.Now().UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: "seat-2", Event: EventSent, Details: "lead sends to seat-2"}); err != nil {
		t.Fatalf("append sent event: %v", err)
	}
	clk.Advance(3 * time.Minute) // now past DeadmanMinutes overall
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 3 (sent event clears seat-1's pending alert): %v", err)
	}
	assertNoEscalations(t, escalationsPath)
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected seat-1's pending alert to clear after a sent event naming seat-2, got %+v", status.PendingAlerts)
	}
}

// TestWatch_Deadman_LeadSpawnedEvent_ClearsPendingAlert pins that a
// `spawned` event -- part of leadActivityEventCount's lead-driven lifecycle
// set (see its doc comment, case (b)) -- counts as lead activity even when
// it names lead itself as SeatID, the concrete case a session-promoted-lead
// org produces when lead self-registers via `ralph org spawn --id lead`.
func TestWatch_Deadman_LeadSpawnedEvent_ClearsPendingAlert(t *testing.T) {
	o, h, _, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 5
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-1 failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")}

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises ALERT): %v", err)
	}

	// Lead spawns itself between cycles.
	if r := o.Spawn(watchSpawnParams("org-a", LeadIdentity, LeadIdentity)); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn lead failed: %+v", r)
	}

	clk.Advance(6 * time.Minute) // past DeadmanMinutes, but lead spawned in between
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	assertNoEscalations(t, escalationsPath)
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected the pending alert to clear after lead's own spawned event, got %+v", status.PendingAlerts)
	}
}

// TestWatch_Deadman_LeadSpawnsReplacementSeat_ClearsPendingAlert pins the
// self-review cycle-3 M3-1 fix: a `spawned` event for a DIFFERENT seat (not
// lead itself) still counts as lead activity -- the concrete scenario is
// lead responding to a stall ALERT by spawning a replacement seat, which
// only lead/the operator can do via `ralph org spawn`. Before this fix,
// leadActivityEventCount only special-cased `stopped`/`disbanded` for
// non-lead-named events, so this spawn would have counted as nothing and
// the deadman would escalate against a demonstrably responsive lead.
func TestWatch_Deadman_LeadSpawnsReplacementSeat_ClearsPendingAlert(t *testing.T) {
	o, h, _, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 5
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-1 failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")}

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises seat-1's ALERT): %v", err)
	}

	// Lead spawns a REPLACEMENT seat (seat-2), not itself, between cycles.
	if r := o.Spawn(watchSpawnParams("org-a", "seat-2", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-2 failed: %+v", r)
	}

	clk.Advance(6 * time.Minute) // past DeadmanMinutes, but lead spawned seat-2 in between
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	assertNoEscalations(t, escalationsPath)
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected the pending alert to clear after lead spawns a replacement seat, got %+v", status.PendingAlerts)
	}
}

// TestWatch_Deadman_ManualStopOfOtherSeat_ClearsPendingAlert pins that a
// manual (operator-issued, non-watchdog) `stopped` event for a DIFFERENT
// seat still counts as lead activity: someone had to run `ralph org stop`
// for it to exist, and issuing that command is itself a lead-driven action
// -- unlike the watchdog's own cutoff `stopped` events
// (reason=watchdog_..., excluded by
// TestWatch_Deadman_WatchdogsOwnCutoffEvent_DoesNotClearPendingAlert above),
// which are self-inflicted and prove nothing about lead responsiveness.
func TestWatch_Deadman_ManualStopOfOtherSeat_ClearsPendingAlert(t *testing.T) {
	o, h, _, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 5
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-1 failed: %+v", r)
	}
	if r := o.Spawn(watchSpawnParams("org-a", "seat-2", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-2 failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")}

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises seat-1's ALERT): %v", err)
	}

	// A human operator manually stops seat-2 (no watchdog_ reason prefix).
	if r := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-2", Reason: "operator: no longer needed"}); r.Err != nil {
		t.Fatalf("stop seat-2: %v", r.Err)
	}

	clk.Advance(6 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	assertNoEscalations(t, escalationsPath)
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected the pending alert to clear after a manual stop of another seat, got %+v", status.PendingAlerts)
	}
}

// TestWatch_Stall_UsesLatestEventOfAnyType_NotOnlyStateEvents pins the
// self-review M-6 fix: the stall condition's time term must track the
// seat's latest manifest event of ANY type, not Roster's SeatStatus.TS
// (which only advances on *state* events and so stays pinned at a healthy
// active seat's own `spawned` TS forever).
func TestWatch_Stall_UsesLatestEventOfAnyType_NotOnlyStateEvents(t *testing.T) {
	o, h, a, clk := testWatchOrg(t)
	o.Config.Budget.SeatWallClockMinutes = 1000 // keep budget out of the picture
	o.Config.Watchdog.StallMinutes = 10

	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetSeq[target] = []string{"same", "same", "same"}

	run, statusPath, _ := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	// Cycle 1: baseline snapshot only (no previous herdr value to compare
	// against yet).
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}

	// Advance well past stall_minutes, but record a fresh non-state `sent`
	// event for the seat in between cycles. Roster's SeatStatus.TS (a
	// spawned-and-still-active seat's last *state* event) never moves for
	// this -- the pre-M-6 stall check (isStallByTime(s.TS, ...)) would still
	// fire on the seat's now-stale spawn TS; the fixed check
	// (latestSeatEventTS, any event type) must see this recent event and
	// not treat the seat as stalled.
	clk.Advance(11 * time.Minute)
	if err := o.Manifest.Append(ManifestEvent{
		TS: clk.Now().UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: "seat-1",
		Event: EventSent, Details: "genuine seat activity",
	}); err != nil {
		t.Fatalf("append sent event: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 0 {
		t.Fatalf("expected no stall ALERT: the seat's own latest event (any type) is recent, got %d: %+v", n, watchdogAlerts(a))
	}
}

// --- cross-review-triage cycle-2 #3: deadman history source is lead-only ---

// TestFilterLeadHistoryLines pins leadHistoryFromField/filterLeadHistoryLines'
// parsing contract directly: only lines whose "from" field (the segment
// between "] " and " → ") is exactly LeadIdentity survive; anything that
// does not parse into that shape is excluded, not conservatively kept.
func TestFilterLeadHistoryLines(t *testing.T) {
	raw := strings.Join([]string{
		"  ● [2026-01-01T00:00:00Z] lead → seat-1: kickoff task",
		"  ○ [2026-01-01T00:01:00Z] seat-1 → lead: ack",
		"  ● [2026-01-01T00:02:00Z] watchdog → lead: TYPE: ALERT",
		"garbled line with no timestamp or arrow at all",
		"  ● [2026-01-01T00:03:00Z] lead with no arrow marker here",
		"  ● [2026-01-01T00:04:00Z] lead → seat-2: another lead message",
	}, "\n")

	got := filterLeadHistoryLines(raw)
	want := strings.Join([]string{
		"  ● [2026-01-01T00:00:00Z] lead → seat-1: kickoff task",
		"  ● [2026-01-01T00:04:00Z] lead → seat-2: another lead message",
	}, "\n")
	if got != want {
		t.Errorf("filterLeadHistoryLines:\n got:  %q\n want: %q", got, want)
	}
}

// TestWatch_Deadman_WatchdogAlertHistoryLine_DoesNotClearPendingAlert pins
// cross-review-triage cycle-2 #3: a new agmsg history line produced by the
// watchdog's OWN alert traffic (watchdog -> lead) must not count as lead
// activity and clear a pending deadman alert -- only lead-authored lines may.
func TestWatch_Deadman_WatchdogAlertHistoryLine_DoesNotClearPendingAlert(t *testing.T) {
	o, h, a, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 100 // keep the timeout out of the picture
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")} // sticky liveness ALERT

	team := agmsgTeam("org-a")
	leadLine := "  ● [2026-01-01T00:00:00Z] lead → seat-1: kickoff task"
	base := leadLine
	withWatchdogAlert := leadLine + "\n  ● [2026-01-01T00:05:00Z] watchdog → lead: TYPE: ALERT\nCONDITION: liveness"
	// call 1: sendAlert's own historyLeadLineCount capture (cycle 1); call 2:
	// checkDeadman's comparison in that same cycle 1; call 3: checkDeadman's
	// comparison in cycle 2, where a new non-lead line has been appended.
	a.HistorySeq[team] = []string{base, base, withWatchdogAlert}

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected 1 pending alert after cycle 1, got %d: %+v", len(status.PendingAlerts), status.PendingAlerts)
	}

	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Errorf("expected the pending alert to remain: a new watchdog->lead history line must not count as lead activity, got %d pending: %+v",
			len(status.PendingAlerts), status.PendingAlerts)
	}
	assertNoEscalations(t, escalationsPath)
}

// TestWatch_Deadman_LeadHistoryLine_ClearsPendingAlert is the positive
// counterpart: a new agmsg history line genuinely authored BY lead (lead ->
// seat) must still clear a pending deadman alert, same as before this fix.
func TestWatch_Deadman_LeadHistoryLine_ClearsPendingAlert(t *testing.T) {
	o, h, a, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 100 // keep the timeout out of the picture
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")} // sticky liveness ALERT

	team := agmsgTeam("org-a")
	// A malformed line is deliberately mixed in alongside the genuine lead
	// line -- it must not itself be mistaken for lead activity (it is simply
	// excluded), the real lead -> seat line is what clears the alert.
	withLeadLine := "garbled line with no timestamp or arrow\n" +
		"  ● [2026-01-01T00:05:00Z] lead → seat-1: are you there?"
	a.HistorySeq[team] = []string{"", "", withLeadLine}

	run, statusPath, _ := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected 1 pending alert after cycle 1, got %d: %+v", len(status.PendingAlerts), status.PendingAlerts)
	}

	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected the pending alert to clear: a new lead->seat history line is genuine lead activity, got %d pending: %+v",
			len(status.PendingAlerts), status.PendingAlerts)
	}
}

// TestWatch_Deadman_HistoryWindowEviction_DoesNotFalselyClearPendingAlert
// pins the self-review cycle-3 M3-2 fix: historyLeadLineCount's window is
// the last-20-of-EVERYONE window (see its doc comment), so as other seats
// chat, older lead lines fall out of that window and the visible lead-line
// set can shrink with NO new lead activity at all. The pre-fix exact-string
// comparison (`cur != "" && cur != pending.History`) treated any change --
// including a pure shrinkage -- as activity and wrongly cleared the pending
// alert; the count-based comparison (`cur > pending.HistoryLeadLines`) does
// not, because eviction can only ever decrease the count.
func TestWatch_Deadman_HistoryWindowEviction_DoesNotFalselyClearPendingAlert(t *testing.T) {
	o, h, a, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 100 // keep the timeout out of the picture
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")} // sticky liveness ALERT

	team := agmsgTeam("org-a")
	threeLeadLines := strings.Join([]string{
		"  ● [2026-01-01T00:00:00Z] lead → seat-1: msg 1",
		"  ● [2026-01-01T00:01:00Z] lead → seat-1: msg 2",
		"  ● [2026-01-01T00:02:00Z] lead → seat-1: msg 3",
	}, "\n")
	// Simulates the underlying last-20-of-everyone window evicting the two
	// older lead lines as other seats chat -- fewer lead lines are visible,
	// but none of them is new.
	shrunkWindow := "  ● [2026-01-01T00:02:00Z] lead → seat-1: msg 3"
	// call 1: sendAlert's own historyLeadLineCount capture (cycle 1,
	// count=3); call 2: checkDeadman's comparison in that same cycle 1
	// (count=3, no growth); call 3: checkDeadman's comparison in cycle 2,
	// where eviction has shrunk the visible window to 1 lead line.
	a.HistorySeq[team] = []string{threeLeadLines, threeLeadLines, shrunkWindow}

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected 1 pending alert after cycle 1, got %d: %+v", len(status.PendingAlerts), status.PendingAlerts)
	}

	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Errorf("expected the pending alert to remain: a shrunk (evicted) history window with no new lead line must not count as activity, got %d pending: %+v",
			len(status.PendingAlerts), status.PendingAlerts)
	}
	assertNoEscalations(t, escalationsPath)
}

// --- cross-review-triage cycle-2 #4: skip just-cut seats within the cycle --

// TestWatch_TotalBudgetCutoff_SkipsCutSeatsForRestOfSameCycle pins the fix:
// once evaluateTotalBudget stops every active seat in a cycle, that same
// cycle's per-seat loop must not still probe those seats for
// stall/liveness/scope-change -- doing so would call herdr AgentGet against a
// seat that was just stopped and could raise a spurious ALERT (and deadman
// pending-alert record) purely because activeSeats was snapshotted before
// the cutoff ran.
func TestWatch_TotalBudgetCutoff_SkipsCutSeatsForRestOfSameCycle(t *testing.T) {
	o, h, a, clk := testWatchOrg(t)
	o.Config.Budget.TotalWallClockMinutes = 5
	o.Config.Budget.SeatWallClockMinutes = 1000 // avoid seat-level cutoff firing first

	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-1 failed: %+v", r)
	}
	if r := o.Spawn(watchSpawnParams("org-a", "seat-2", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-2 failed: %+v", r)
	}

	target1 := herdrAgentName("org-a", "seat-1")
	target2 := herdrAgentName("org-a", "seat-2")
	// Sticky liveness failure for both seats -- if evaluateSeat's liveness
	// probe still ran against a just-cut seat this cycle, it would both call
	// AgentGet against its target and raise a spurious liveness ALERT.
	h.AgentGetErrSeq[target1] = []error{errors.New("herdr: agent not found")}
	h.AgentGetErrSeq[target2] = []error{errors.New("herdr: agent not found")}

	run, statusPath, _ := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	clk.Advance(10 * time.Minute) // past total budget
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("evaluateCycle: %v", err)
	}

	st, err := o.Status("org-a", false)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	for _, seatID := range []string{"seat-1", "seat-2"} {
		if findSeatStatus(t, st.Seats, seatID).Active {
			t.Fatalf("expected seat %q cut off by total-budget", seatID)
		}
	}

	for _, c := range h.agentGetCalls {
		if c == target1 || c == target2 {
			t.Errorf("expected no AgentGet probe for just-cut seat target %q in the cutoff cycle, but one was made (calls: %v)", c, h.agentGetCalls)
		}
	}

	alerts := watchdogAlerts(a)
	if len(alerts) != 1 {
		t.Fatalf("expected exactly 1 ALERT (org-level total-budget only, no per-seat liveness ALERTs), got %d: %+v", len(alerts), alerts)
	}
	if !strings.Contains(alerts[0].message, "CONDITION: total_budget") {
		t.Errorf("expected the sole ALERT to be total_budget, got %q", alerts[0].message)
	}
}

// TestWatch_SeatBudgetCutoff_SkipsFurtherProbesForCutSeatSameCycle is the
// single-seat analogue: once evaluateSeatBudget stops a seat, evaluateSeat
// must return before its stall/liveness/scope-change checks run against that
// same, now-stale seat this cycle.
func TestWatch_SeatBudgetCutoff_SkipsFurtherProbesForCutSeatSameCycle(t *testing.T) {
	o, h, a, clk := testWatchOrg(t)
	o.Config.Budget.SeatWallClockMinutes = 5
	o.Config.Budget.TotalWallClockMinutes = 1000 // avoid total-budget firing first

	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-1 failed: %+v", r)
	}

	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")}

	run, statusPath, _ := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	clk.Advance(10 * time.Minute) // past seat budget
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("evaluateCycle: %v", err)
	}

	st, err := o.Status("org-a", false)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if findSeatStatus(t, st.Seats, "seat-1").Active {
		t.Fatalf("expected seat-1 cut off by seat-budget")
	}

	for _, c := range h.agentGetCalls {
		if c == target {
			t.Errorf("expected no AgentGet probe for the just-cut seat in the cutoff cycle, but one was made (calls: %v)", h.agentGetCalls)
		}
	}

	alerts := watchdogAlerts(a)
	if len(alerts) != 1 {
		t.Fatalf("expected exactly 1 ALERT (seat_budget only, no liveness ALERT), got %d: %+v", len(alerts), alerts)
	}
	if !strings.Contains(alerts[0].message, "CONDITION: seat_budget") {
		t.Errorf("expected the sole ALERT to be seat_budget, got %q", alerts[0].message)
	}
}

// --- PR④ known gap #5: deadman probe-recovery false-clear -------------------

// TestWatch_Deadman_ProbeOutageRecoveryAlone_DoesNotClearPendingAlert pins the
// checkDeadman fix for cross-review-triage cycle-3 #5: an ALERT recorded
// while the lead herdr probe (leadProbeSnapshot) was unavailable persists
// LeadAgentGet == "" as its baseline. A later cycle where the probe merely
// recovers -- with no other genuine lead activity -- must NOT by itself
// clear the pending alert: pre-fix, `cur != "" && cur != pending.LeadAgentGet`
// is trivially satisfied by any recovered value compared against a ""
// baseline, treating "the probe came back" as if it were "lead did
// something new".
func TestWatch_Deadman_ProbeOutageRecoveryAlone_DoesNotClearPendingAlert(t *testing.T) {
	o, h, _, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 100 // keep the timeout out of the picture
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	// seat-1's own probe fails throughout -- the sticky liveness ALERT this
	// test exercises the deadman bookkeeping for.
	seatTarget := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[seatTarget] = []error{errors.New("herdr: agent not found")}

	// The lead's own probe (checkDeadman's source #2) is down for exactly the
	// 2 calls made during cycle 1 (sendAlert's baseline capture, then
	// checkDeadman's own same-cycle comparison), then recovers from cycle 2
	// onward.
	leadTarget := herdrAgentName("org-a", LeadIdentity)
	h.AgentGetErrSeq[leadTarget] = []error{
		errors.New("herdr: unreachable"),
		errors.New("herdr: unreachable"),
		nil, // recovered: falls through to the default "ok" AgentGet response
	}

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises ALERT, probe baseline unavailable): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected 1 pending alert after cycle 1, got %d: %+v", len(status.PendingAlerts), status.PendingAlerts)
	}
	for _, pending := range status.PendingAlerts {
		if pending.LeadAgentGet != "" {
			t.Fatalf("expected the pending alert's probe baseline to be the unavailable sentinel (\"\"), got %q", pending.LeadAgentGet)
		}
	}

	clk.Advance(1 * time.Minute) // well under DeadmanMinutes
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2 (probe recovers, no other activity): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Errorf("expected the pending alert to remain: probe recovery alone (unavailable baseline) must not count as lead activity, got %d pending: %+v",
			len(status.PendingAlerts), status.PendingAlerts)
	}
	assertNoEscalations(t, escalationsPath)
}

// TestWatch_Deadman_ProbeOutageThenGenuineManifestActivity_ClearsPendingAlert
// is the positive counterpart: once a pending alert's probe baseline is the
// unavailable sentinel, that source can never independently clear it (its
// baseline is frozen at ALERT time), but genuine lead activity via a
// different, unaffected source -- a new lead-attributable manifest event --
// must still clear it, exactly as before this fix.
func TestWatch_Deadman_ProbeOutageThenGenuineManifestActivity_ClearsPendingAlert(t *testing.T) {
	o, h, _, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 100 // keep the timeout out of the picture
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	seatTarget := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[seatTarget] = []error{errors.New("herdr: agent not found")}

	leadTarget := herdrAgentName("org-a", LeadIdentity)
	h.AgentGetErrSeq[leadTarget] = []error{
		errors.New("herdr: unreachable"),
		errors.New("herdr: unreachable"),
		nil, // recovered from cycle 2 onward, same as the negative test above
	}

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises ALERT, probe baseline unavailable): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected 1 pending alert after cycle 1, got %d: %+v", len(status.PendingAlerts), status.PendingAlerts)
	}

	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2 (probe recovers, no other activity): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected the pending alert to remain after probe recovery alone, got %d pending: %+v", len(status.PendingAlerts), status.PendingAlerts)
	}

	// Genuine lead activity: a real `sent` event, lead-authored by
	// construction (see leadActivityEventCount's doc comment).
	if err := o.Manifest.Append(ManifestEvent{TS: clk.Now().UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: "seat-1", Event: EventSent, Details: "lead activity"}); err != nil {
		t.Fatalf("append lead-activity event: %v", err)
	}

	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 3 (genuine manifest activity): %v", err)
	}
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected the pending alert to clear: a genuine manifest sent event is lead activity via an unaffected source, got %d pending: %+v",
			len(status.PendingAlerts), status.PendingAlerts)
	}
	assertNoEscalations(t, escalationsPath)
}

// --- PR④ known gap #6: WatchdogJoined only set on successful Join -----------

// TestWatch_EnsureWatchdogJoined_TransientFailure_RetriesUntilSuccess pins
// the ensureWatchdogJoined fix for cross-review-triage cycle-3 #6: a
// transient Join failure must NOT persist WatchdogJoined = true (which would
// permanently skip every future Join retry for the org, per the const's own
// doc comment), so the next cycle retries and, once Join actually succeeds,
// status.WatchdogJoined is finally set and ALERT delivery is unaffected
// (best-effort Send is independent of Join's own outcome).
func TestWatch_EnsureWatchdogJoined_TransientFailure_RetriesUntilSuccess(t *testing.T) {
	o, h, a, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 100          // keep the deadman sweep out of the picture
	o.Config.Watchdog.StallMinutes = 10000 // keep the stall condition out of the picture

	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	// sendAlert (and therefore ensureWatchdogJoined) only runs on an Active
	// *transition* (raiseOrClear dedupes a still-Active condition, see its
	// own doc comment) -- so this test's liveness condition must clear and
	// re-raise across 3 cycles to call ensureWatchdogJoined twice: cycle 1
	// fails (raises), cycle 2 recovers (clears, no ALERT), cycle 3 fails
	// again (re-raises, calling ensureWatchdogJoined a second time).
	seatTarget := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[seatTarget] = []error{
		errors.New("herdr: agent not found"), // cycle 1: fails
		nil,                                  // cycle 2: recovers
		errors.New("herdr: agent not found"), // cycle 3: fails again
	}

	// cycle 1's Join attempt fails; cycle 3's Join attempt (the only other
	// one -- cycle 2 raises no ALERT) succeeds. Keyed by watchdogIdentity so
	// Spawn's own lead/seat Join calls (made before evaluateCycle even runs)
	// do not consume these entries.
	a.JoinErrSeq[watchdogIdentity] = []error{errors.New("agmsg: team unreachable"), nil}

	run, statusPath, _ := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises ALERT, Join fails): %v", err)
	}
	if status.WatchdogJoined {
		t.Fatalf("expected WatchdogJoined to remain false after a failed Join")
	}
	if calls := watchdogJoinCalls(a); len(calls) != 1 {
		t.Fatalf("expected exactly 1 watchdog Join attempt after cycle 1, got %d: %+v", len(calls), calls)
	}

	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 2 (recovers, no ALERT): %v", err)
	}
	if status.WatchdogJoined {
		t.Fatalf("expected WatchdogJoined to still be false: no ALERT was raised in cycle 2, so Join is not retried")
	}
	if calls := watchdogJoinCalls(a); len(calls) != 1 {
		t.Fatalf("expected still exactly 1 watchdog Join attempt after cycle 2 (no ALERT raised), got %d: %+v", len(calls), calls)
	}

	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle(context.Background(), "org-a", status); err != nil {
		t.Fatalf("cycle 3 (re-raises ALERT, Join retried and succeeds): %v", err)
	}
	if !status.WatchdogJoined {
		t.Fatalf("expected WatchdogJoined to be true after cycle 3's successful Join retry")
	}
	if calls := watchdogJoinCalls(a); len(calls) != 2 {
		t.Fatalf("expected cycle 3 to retry the watchdog Join (2 total attempts), got %d: %+v", len(calls), calls)
	}

	// ALERT delivery itself is best-effort and independent of Join's outcome
	// (SendWatchdogAlert's own doc comment) -- both the cycle-1 and cycle-3
	// liveness ALERTs must still have been sent despite the cycle-1 Join
	// failure.
	alerts := watchdogAlerts(a)
	if len(alerts) != 2 {
		t.Fatalf("expected 2 ALERTs sent (cycle 1 and cycle 3) despite the cycle-1 Join failure, got %d: %+v", len(alerts), alerts)
	}
}

// --- helpers -----------------------------------------------------------------

func findSeatStatus(t *testing.T, seats []SeatStatus, seatID string) SeatStatus {
	t.Helper()
	for _, s := range seats {
		if s.SeatID == seatID {
			return s
		}
	}
	t.Fatalf("seat %q not found in %+v", seatID, seats)
	return SeatStatus{}
}

func assertNoEscalations(t *testing.T, path string) {
	t.Helper()
	if lines := readJSONLFile(t, path); len(lines) != 0 {
		t.Fatalf("expected no escalations yet, got %d: %v", len(lines), lines)
	}
}

func readJSONLFile(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read %s: %v", path, err)
	}
	var lines []string
	for l := range strings.SplitSeq(strings.TrimRight(string(data), "\n"), "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	return lines
}
