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
	a := &fakeWatchAgmsg{HistorySeq: map[string][]string{}}
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
// the self-review M-6 fix: the deadman's manifest-growth "has anything
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
	// evaluateTotalBudget produce ("reason=watchdog_..."). Before the M-6
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

// TestWatch_Stall_UsesLatestEventOfAnyType_NotOnlyStateEvents pins the
// self-review M-8 fix: the stall condition's time term must track the
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
	// this -- the pre-M-8 stall check (isStallByTime(s.TS, ...)) would still
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
