package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// --- test fixtures (self-contained for this file: does not reuse
// spawn_test.go's fakeHerdr/fakeAgmsg, since those do not implement the
// AgentGet/History probes watch.go's watchHerdrProbe/watchAgmsgHistory
// consumption-side interfaces need) ---------------------------------------

// fakeWatchHerdr is a call-recording HerdrClient + watchHerdrProbe fake.
// AgentGetSeq/AgentGetErrSeq, keyed by target (herdrAgentName's convention),
// let a test script a per-cycle sequence of AgentGet outcomes for a
// specific seat/leader target; once a sequence is exhausted the last queued
// value repeats (a test that only cares about a constant value can supply
// exactly one entry).
type fakeWatchHerdr struct {
	mu sync.Mutex

	AgentGetSeq    map[string][]string
	AgentGetErrSeq map[string][]error
	// PaneGetErr, when set, is what PaneGet returns instead of the
	// not-found reply: a herdr that does not answer, so a Stop cannot
	// confirm the seat's pane and records `stop_failed`.
	PaneGetErr error

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

func (f *fakeWatchHerdr) PaneClose(_ context.Context, _ string) error {
	return nil
}

func (f *fakeWatchHerdr) WorkspaceClose(_ context.Context, _ string) error {
	return nil
}

// PaneGet, TabGet and WorkspaceGet answer herdr's not-found reply for every
// id: this fake does not model herdr's panes, so a Stop in a watch test
// records the seat's pane as already closed, with no C-c and no close. A
// set PaneGetErr replaces PaneGet's reply.
func (f *fakeWatchHerdr) PaneGet(_ context.Context, paneID string) (string, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.PaneGetErr != nil {
		return "", "", f.PaneGetErr
	}
	return "", "", driver.NewHerdrError(driver.HerdrCodePaneNotFound, "pane "+paneID+" not found")
}

func (f *fakeWatchHerdr) TabGet(_ context.Context, tabID string) (string, error) {
	return "", driver.NewHerdrError(driver.HerdrCodeTabNotFound, "tab "+tabID+" not found")
}

func (f *fakeWatchHerdr) WorkspaceGet(_ context.Context, workspaceID string) (string, error) {
	return "", driver.NewHerdrError(driver.HerdrCodeWorkspaceNotFound, "workspace "+workspaceID+" not found")
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
	// (not a flat sequence) because Spawn's own leader/seat Join calls and
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
// the "watchdog" identity to "leader", not via the seat-steering Send verb's
// PaneSendText, since verb-Send silently drops the message when no leader SEAT
// was ever spawned -- the normal session-promoted-leader org shape).
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
// SeatID to leader) shares the same fake/log, so ALERT-count assertions must
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
// Join attempts -- Spawn's own leader/seat Join calls share the same fake/log,
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
		DriverPool: []string{"claude"},
		ModelPool:  []config.OrgModelPoolEntry{{Driver: "claude", Model: "sonnet"}},
		Roles:      map[string][]string{},
		MaxSeats:   10,
		// The org-wide limits at config.Default's values: spawn refuses at
		// 0, and no watch test is about them.
		MaxOrgs:        config.Default().Org.MaxOrgs,
		MaxTotalSeats:  config.Default().Org.MaxTotalSeats,
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
		Manifest: NewManifestStoreAtPath(ManifestPathIn(dir)),
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
// (rather than going through RunWatchAll's real wait between cycles) so a
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

// --- AC-4: heartbeat stall (ALERT, no cutoff, recovers, re-fires) ----------

func TestWatch_Stall_AlertsOnlyNeverStops_RecoversAndRefires(t *testing.T) {
	o, h, a, clk := testWatchOrg(t)
	o.Config.Watchdog.StallMinutes = 5
	// Deliberately no leader seat spawned -- ALERT delivery must not depend on
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
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 0 {
		t.Fatalf("expected no ALERT on the baseline cycle, got %d", n)
	}

	// Cycle 2: same raw text AND >5m elapsed since the (unchanged) manifest
	// TS -- stalled, exactly 1 ALERT.
	clk.Advance(10 * time.Minute)
	if err := run.evaluateCycle("org-a", status); err != nil {
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
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 3: %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 1 {
		t.Fatalf("expected still exactly 1 ALERT (dedupe) after cycle 3, got %d", n)
	}

	// Cycle 4: raw text changed ("different") -- recovers, clears the key.
	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 4: %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 1 {
		t.Fatalf("expected still exactly 1 ALERT after the recovery cycle, got %d", n)
	}

	// Cycle 5: baseline again ("same-again"), no previous match yet.
	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 5: %v", err)
	}

	// Cycle 6: matches cycle 5's snapshot again and still >5m stalled since
	// manifest TS -- re-fires (a second, distinct ALERT).
	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 6: %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 2 {
		t.Fatalf("expected a second ALERT after re-stalling, got %d: %+v", n, watchdogAlerts(a))
	}
}

// --- AC-4: process liveness --------------------------------------------------

func TestWatch_Liveness_AlertOnAgentGetError(t *testing.T) {
	o, h, a, _ := testWatchOrg(t)
	// Deliberately no leader seat spawned -- ALERT delivery must not depend on
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

	if err := run.evaluateCycle("org-a", status); err != nil {
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

func TestWatch_ScopeChange_AlertCarriesScopeText_NoStop(t *testing.T) {
	o, h, a, _ := testWatchOrg(t)
	// Deliberately no leader seat spawned -- ALERT delivery must not depend on
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

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1 (baseline): %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 0 {
		t.Fatalf("expected no ALERT on the baseline cycle, got %d", n)
	}

	if err := run.evaluateCycle("org-a", status); err != nil {
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

// TestWatch_ScopeChange_TruncatesOversizedPorcelainOutput pins the
// tech-debt "watchdog deferred LOW (2)" fix: a busy seat's `git status
// --porcelain` output used to be interpolated whole into the scope-change
// ALERT body, which could push the overall message past
// protocol.DefaultMaxBodyChars and silently degrade to the content-free
// protocol-validation-failure fallback (asserted separately by
// TestSendAlert_ProtocolValidationFailureFallback_IncludesSeatHeader).
// 30 dirty files, each with a long generated-file-style path, both exceeds
// maxScopeChangeLines (20) and -- even after that line-count truncation --
// would still overflow DefaultMaxBodyChars without the byte-budget second
// pass in truncateScopeOutput.
func TestWatch_ScopeChange_TruncatesOversizedPorcelainOutput(t *testing.T) {
	o, _, a, _ := testWatchOrg(t)
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	seat1Cwd := watchSpawnParams("org-a", "seat-1", "worker").Cwd
	var dirtyLines []string
	for i := range 30 {
		dirtyLines = append(dirtyLines, fmt.Sprintf(" M internal/some/deeply/nested/generated/package/directory/file_number_%03d_with_a_long_name.go", i))
	}
	oversized := strings.Join(dirtyLines, "\n")
	calls := 0
	gitStatus := func(cwd string) (string, error) {
		calls++
		if cwd != seat1Cwd || calls == 1 {
			return "", nil // baseline: clean worktree
		}
		return oversized, nil
	}

	run, statusPath, _ := newTestWatchRun(o, gitStatus, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1 (baseline): %v", err)
	}
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2 (scope changed): %v", err)
	}

	alerts := watchdogAlerts(a)
	if n := len(alerts); n != 1 {
		t.Fatalf("expected exactly 1 scope-change ALERT, got %d: %+v", n, alerts)
	}
	body := alerts[0].message
	if n := utf8.RuneCountInString(body); n > protocol.DefaultMaxBodyChars {
		t.Errorf("expected the ALERT message to stay within DefaultMaxBodyChars (%d), got %d chars:\n%s",
			protocol.DefaultMaxBodyChars, n, body)
	}
	if !strings.Contains(body, "more lines") {
		t.Errorf("expected a \"... N more lines\" truncation summary in the ALERT body, got:\n%s", body)
	}
	if strings.Contains(body, "file_number_029") {
		t.Errorf("expected the 30th dirty-file line to be truncated out of the ALERT body, got:\n%s", body)
	}
	if !strings.Contains(body, "file_number_000") {
		t.Errorf("expected the first dirty-file line to still be present in the ALERT body, got:\n%s", body)
	}
}

// TestTruncateScopeOutput_LineCountThenByteBudget unit-tests
// truncateScopeOutput directly: a two-pass truncation (line count, then a
// hard byte budget) rather than either bound alone.
func TestTruncateScopeOutput_LineCountThenByteBudget(t *testing.T) {
	t.Run("within both bounds returns unchanged", func(t *testing.T) {
		in := " M a.go\n?? b.go"
		if got := truncateScopeOutput(in, scopeChangeBodyBudget); got != in {
			t.Errorf("expected unchanged output for small input, got %q", got)
		}
	})

	t.Run("over line count but under byte budget truncates by line only", func(t *testing.T) {
		var lines []string
		for i := range 25 {
			lines = append(lines, fmt.Sprintf(" M f%d.go", i))
		}
		in := strings.Join(lines, "\n")
		got := truncateScopeOutput(in, scopeChangeBodyBudget)
		if !strings.Contains(got, "… 5 more lines") {
			t.Errorf("expected exactly 5 hidden lines (25 - maxScopeChangeLines=20), got:\n%s", got)
		}
		if strings.Contains(got, "f24.go") {
			t.Errorf("expected the 25th line truncated out, got:\n%s", got)
		}
	})

	t.Run("under line count but over byte budget truncates by size", func(t *testing.T) {
		in := strings.Repeat("x", 500) // one line, well under maxScopeChangeLines
		got := truncateScopeOutput(in, 100)
		if utf8.RuneCountInString(got) > 100 {
			t.Errorf("expected output bounded to the byte budget, got %d chars: %q", len(got), got)
		}
		if !strings.Contains(got, "more lines") {
			t.Errorf("expected a truncation summary once even a single line is dropped for size, got %q", got)
		}
	})
}

// TestSendAlert_ProtocolValidationFailureFallback_IncludesSeatHeader pins
// the second half of the tech-debt "watchdog deferred LOW (2)" fix: the
// protocol-validation-failure fallback message itself must still carry a
// `SEAT: <seat_id>` header so Leader can identify the subject seat, instead of
// only "message failed protocol validation" with no indication of who it
// was about. Constructs an oversized body directly (rather than relying on
// scope-change's own truncation, which is designed to prevent this path
// from being reached in practice) so this fallback is exercised even if
// truncateScopeOutput's budget is later tuned.
func TestSendAlert_ProtocolValidationFailureFallback_IncludesSeatHeader(t *testing.T) {
	o, _, a, clk := testWatchOrg(t)
	run, statusPath, _ := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}
	// sendAlert is normally only reached through evaluateCycle, which
	// lazy-inits these maps first; called directly here, so mirror that
	// same lazy-init.
	status.PendingAlerts = map[string]*watchPendingAlert{}

	oversizedBody := strings.Repeat("x", protocol.DefaultMaxBodyChars+500)
	msg := fmt.Sprintf("TYPE: ALERT\nORG_ID: org-a\nSEAT: seat-1\nCONDITION: scope_change\n\n%s", oversizedBody)
	run.sendAlert(status, "org-a", "seat-1", "scope_change", msg, clk.Now())

	alerts := watchdogAlerts(a)
	if n := len(alerts); n != 1 {
		t.Fatalf("expected exactly 1 ALERT send, got %d: %+v", n, alerts)
	}
	body := alerts[0].message
	if !strings.Contains(body, "SEAT: seat-1") {
		t.Errorf("expected the fallback message to carry a SEAT header for the subject seat, got:\n%s", body)
	}
	if !strings.Contains(body, "message failed protocol validation") {
		t.Errorf("expected the fallback message text, got:\n%s", body)
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

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises ALERT): %v", err)
	}
	assertNoEscalations(t, escalationsPath)

	clk.Advance(6 * time.Minute) // past DeadmanMinutes, no manifest/leader/history activity
	if err := run.evaluateCycle("org-a", status); err != nil {
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
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 3 (post-escalation): %v", err)
	}
	lines = readJSONLFile(t, escalationsPath)
	if len(lines) != 1 {
		t.Fatalf("expected escalation dedupe, still 1 line, got %d: %v", len(lines), lines)
	}
}

func TestWatch_Deadman_LeaderActivity_PreventsEscalation(t *testing.T) {
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

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises ALERT): %v", err)
	}

	// Simulate leader activity: a new manifest event recorded between cycles.
	if err := o.Manifest.Append(ManifestEvent{TS: clk.Now().UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: LeaderIdentity, Event: EventSent, Details: "leader activity"}); err != nil {
		t.Fatalf("append leader-activity event: %v", err)
	}

	clk.Advance(6 * time.Minute) // past DeadmanMinutes, but leader was active
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	assertNoEscalations(t, escalationsPath)
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected the pending alert to clear after leader activity, got %+v", status.PendingAlerts)
	}
}

func TestWatch_Deadman_LeaderIsAnomalySubject_EscalatesImmediately(t *testing.T) {
	o, _, _, _ := testWatchOrg(t)
	if r := o.Spawn(watchSpawnParams("org-a", LeaderIdentity, LeaderIdentity)); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn leader failed: %+v", r)
	}
	target := herdrAgentName("org-a", LeaderIdentity)
	// Sticky liveness failure for the leader seat itself.
	fh := o.Herdr.(*fakeWatchHerdr)
	fh.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")}

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	// Same cycle: ALERT raised (subject=leader) then the deadman sweep
	// escalates immediately, without waiting for DeadmanMinutes.
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("evaluateCycle: %v", err)
	}
	lines := readJSONLFile(t, escalationsPath)
	if len(lines) != 1 {
		t.Fatalf("expected exactly 1 immediate escalation for a leader-subject anomaly, got %d: %v", len(lines), lines)
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("parse escalation line: %v", err)
	}
	if rec["reason"] != "leader_is_anomaly_subject" {
		t.Errorf("expected reason=leader_is_anomaly_subject, got %+v", rec)
	}
}

// The desktop notification of a watch escalation gets its own deadline,
// inboxDesktopNotifyTimeout, cut from context.Background() (self-review L2):
// an osascript that does not return cannot hold the cycle, and a watch whose
// loop context is cancelled before the cycle ends (a SIGINT or SIGTERM
// mid-cycle) still sends the notification of the escalation it records,
// which is never sent again. That holds on both paths of an escalation
// (plan 2026-10-23-watch-escalation-inbox, D9): an ALERT item sent to the
// human path at once (Org.DesktopNotify, through the inbox), and an
// escalation the inbox could not take (WatchAllParams.Escalate). Before L2
// this test pinned the opposite: a context derived from the cycle's.
func TestWatch_Deadman_EscalateFuncGetsItsOwnDeadlineEvenWhenTheLoopIsCancelled(t *testing.T) {
	for _, viaInbox := range []bool{true, false} {
		name := "an escalation without the inbox, WatchAllParams.Escalate"
		if viaInbox {
			name = "an ALERT item sent at once, Org.DesktopNotify"
		}
		t.Run(name, func(t *testing.T) {
			f := newWatchAllFixture(t)
			if r := f.o.Spawn(watchSpawnParams("org-a", LeaderIdentity, LeaderIdentity)); r.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("spawn leader failed: %+v", r)
			}
			f.h.AgentGetErrSeq[herdrAgentName("org-a", LeaderIdentity)] = []error{errors.New("herdr: agent not found")}

			type call struct {
				left        time.Duration
				hasDeadline bool
				err         error
			}
			var calls []call
			record := func(ctx context.Context, _ string) error {
				deadline, ok := ctx.Deadline()
				calls = append(calls, call{left: time.Until(deadline), hasDeadline: ok, err: ctx.Err()})
				return nil
			}
			p := f.params(1)
			if viaInbox {
				f.o.DesktopNotify = record
			} else {
				f.o.Inbox = nil
				p.Escalate = record
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // the loop's context is done before the cycle escalates
			if err := runWatchAll(t, f.o, ctx, p, WatchHooks{}); err != nil {
				t.Fatalf("RunWatchAll: %v", err)
			}
			if len(calls) != 1 {
				t.Fatalf("EscalateFunc called %d times; want 1 for the leader-subject escalation", len(calls))
			}
			c := calls[0]
			if !c.hasDeadline || c.left <= 0 || c.left > inboxDesktopNotifyTimeout {
				t.Errorf("EscalateFunc context deadline: present %t, %v left; want one within %v", c.hasDeadline, c.left, inboxDesktopNotifyTimeout)
			}
			if c.err != nil {
				t.Errorf("EscalateFunc context is already done (%v); want it not cut off by the cancelled loop context", c.err)
			}
			if lines := readJSONLFile(t, f.o.EscalationsPath); len(lines) != 1 {
				t.Errorf("escalations.jsonl = %v; want the one escalation the notification is for", lines)
			}
		})
	}
}

// --- plan 2026-10-23-watch-escalation-inbox: escalations as ALERT items ----

// watchInboxFixture is testWatchOrg's Org with an inbox, an
// escalations.jsonl, and a stub desktop notification in one scratch state
// dir, with the default deadlines (ack 10 min, resolve 60 min) and the
// director switch off. As in production, where the CLI gives the Org and
// the watch the same escalations.jsonl, the watchRun of run writes to the
// Org's. The stub DesktopNotify (the inbox's human path) records each
// message and returns notifyErr; the stub escalateFn of run (the path
// without the inbox) calls atEscalate when it is set, then records each
// message and returns escalateErr.
type watchInboxFixture struct {
	o      *Org
	h      *fakeWatchHerdr
	clk    *fakeClock
	dir    string
	stderr bytes.Buffer

	notifyErr   error
	notified    []string
	escalateErr error
	escalated   []string
	atEscalate  func()
}

func newWatchInboxFixture(t *testing.T) *watchInboxFixture {
	t.Helper()
	o, h, _, clk := testWatchOrg(t)
	f := &watchInboxFixture{o: o, h: h, clk: clk, dir: t.TempDir()}
	inbox := NewInboxStore(f.dir)
	inbox.Now = clk.Now
	o.Inbox = inbox
	o.EscalationsPath = EscalationsPathIn(f.dir)
	o.DesktopNotify = func(_ context.Context, message string) error {
		f.notified = append(f.notified, message)
		return f.notifyErr
	}
	o.Config.Director = config.OrgDirectorConfig{AckMinutes: 10, ResolveMinutes: 60}
	return f
}

// run returns a watchRun on f's Org with its Config as it is now (set the
// director switch first), acceptsWork, and the stub escalateFn.
func (f *watchInboxFixture) run(acceptsWork func() bool) *watchRun {
	return &watchRun{
		org: f.o, cfg: f.o.Config,
		gitStatus: func(string) (string, error) { return "", nil },
		escalateFn: func(_ context.Context, message string) error {
			if f.atEscalate != nil {
				f.atEscalate()
			}
			f.escalated = append(f.escalated, message)
			return f.escalateErr
		},
		stderr:          &f.stderr,
		statusPath:      filepath.Join(f.dir, WatchStatusFileName("org-a")),
		escalationsPath: f.o.EscalationsPath,
		acceptsWork:     acceptsWork,
	}
}

// pendingLeaderAlert returns a status of org-a holding one pending ALERT
// about the leader, raised at clk's time, which the next cycle's deadman
// sweep escalates at once with leader_is_anomaly_subject, and its alert_id.
// Its probe baselines are the unavailable sentinels, so no probe reads as
// leader activity.
func pendingLeaderAlert(clk *fakeClock) (*watchStatusFile, string) {
	alertID := fmt.Sprintf("%s@%d", conditionKey("org-a", LeaderIdentity, condLiveness), clk.Now().UnixNano())
	return &watchStatusFile{OrgID: "org-a", PendingAlerts: map[string]*watchPendingAlert{
		alertID: {AlertID: alertID, TS: clk.Now().UTC().Format(time.RFC3339), Subject: LeaderIdentity, HistoryLeaderLines: -1},
	}}, alertID
}

// pendingSeatAlert is pendingLeaderAlert for a liveness ALERT about seat-1
// of org-a, raised deadmanMinutes + 1 minutes before clk's time, so the
// next cycle's deadman sweep escalates it with deadman_timeout. The
// manifest of the fixtures that use it holds no leader activity, so nothing
// reads as the leader acting on the ALERT.
func pendingSeatAlert(clk *fakeClock, deadmanMinutes int) (*watchStatusFile, string) {
	raised := clk.Now().Add(-time.Duration(deadmanMinutes+1) * time.Minute)
	alertID := fmt.Sprintf("%s@%d", conditionKey("org-a", "seat-1", condLiveness), raised.UnixNano())
	return &watchStatusFile{OrgID: "org-a", PendingAlerts: map[string]*watchPendingAlert{
		alertID: {AlertID: alertID, TS: raised.UTC().Format(time.RFC3339), Subject: "seat-1", HistoryLeaderLines: -1},
	}}, alertID
}

// assertEscalatedOnce fails the test unless alertID left the pending alerts
// of status and is in its Escalated record.
func assertEscalatedOnce(t *testing.T, status *watchStatusFile, alertID string) {
	t.Helper()
	if _, pending := status.PendingAlerts[alertID]; pending || !status.Escalated[alertID] {
		t.Errorf("after the escalation, pending %v and escalated %v; want %s escalated and no longer pending", status.PendingAlerts, status.Escalated, alertID)
	}
}

// assertAlertItemBody fails the test unless body is a typed message that
// protocol.Parse and protocol.Validate accept, of TYPE ALERT with no
// TASK_ID, with exactly the headers want besides TYPE, and a body of one
// line saying why the watch escalated it.
func assertAlertItemBody(t *testing.T, body string, want map[string]string) {
	t.Helper()
	m, err := protocol.Parse(body)
	if err == nil {
		err = protocol.Validate(m, protocol.DefaultMaxBodyChars)
	}
	if err != nil {
		t.Fatalf("the ALERT item's body %q fails the protocol check: %v", body, err)
	}
	if m.Type != protocol.TypeAlert || m.TaskID != "" || !reflect.DeepEqual(m.Fields, want) {
		t.Errorf("body %q parses as TYPE %q, TASK_ID %q, headers %v; want ALERT with no TASK_ID and %v", body, m.Type, m.TaskID, m.Fields, want)
	}
	if !strings.HasPrefix(m.Body, "watchdog: escalated to a person because ") || strings.Contains(m.Body, "\n") {
		t.Errorf("body text %q; want one line saying why the watch escalated the ALERT", m.Body)
	}
}

// AC1 and AC3: a deadman escalation of a seat's ALERT and the escalation at
// once of an ALERT about the leader each become one TYPE ALERT item. Its
// escalated event and the folded item carry the ALERT's alert_id
// (<org>/<seat>/<condition>@<n>, the leader's too), and its body is a typed
// message with the ALERT's headers plus REASON, ALERT_ID, and ALERT_TS. With
// the director switch off it is sent to the human path in the same cycle:
// one escalations.jsonl line with inbox_id, alert_id, and the seat as
// subject, the watch's banner, and one desktop notification that names
// ALERT; the path without the inbox is not taken. A later cycle with the
// seat still failing records nothing more.
func TestWatch_Escalation_ASeatsAlertAndAnAlertAboutTheLeaderEachBecomeOneAlertItem(t *testing.T) {
	cases := []struct {
		name, seat, role, reason string
		// wait is how long after the ALERT the cycle that escalates it runs;
		// 0 means the ALERT's own cycle.
		wait time.Duration
	}{
		{"a seat's ALERT at the deadman timeout", "seat-1", "worker", "deadman_timeout", 6 * time.Minute},
		{"an ALERT about the leader at once", LeaderIdentity, LeaderIdentity, "leader_is_anomaly_subject", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatchInboxFixture(t)
			f.o.Config.DeadmanMinutes = 5
			if r := f.o.Spawn(watchSpawnParams("org-a", tc.seat, tc.role)); r.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("spawn %s failed: %+v", tc.seat, r)
			}
			f.h.AgentGetErrSeq[herdrAgentName("org-a", tc.seat)] = []error{errors.New("herdr: agent not found")}
			run := f.run(nil)
			status := &watchStatusFile{OrgID: "org-a"}
			alertTS := f.clk.Now().UTC().Format(time.RFC3339)
			if err := run.evaluateCycle("org-a", status); err != nil {
				t.Fatalf("cycle 1: %v", err)
			}
			if tc.wait > 0 {
				if items := mustReadInbox(t, f.o.Inbox).Items; len(items) != 0 {
					t.Fatalf("inbox before the deadman timeout = %+v; want empty", items)
				}
				f.clk.Advance(tc.wait)
				if err := run.evaluateCycle("org-a", status); err != nil {
					t.Fatalf("cycle 2: %v", err)
				}
			}
			escalatedTS := f.clk.Now().UTC().Format(time.RFC3339)

			items := mustReadInbox(t, f.o.Inbox).Items
			if len(items) != 1 {
				t.Fatalf("inbox = %+v; want one ALERT item", items)
			}
			item := items[0]
			if item.ID != "e1" || item.Type != protocol.TypeAlert || item.OrgID != "org-a" || item.Delivery != InboxDeliveryNow ||
				item.State != InboxStateOpen || !item.Notified || !strings.HasPrefix(item.AlertID, conditionKey("org-a", tc.seat, condLiveness)+"@") {
				t.Fatalf("item = %+v; want e1, an open ALERT of org-a sent at once, with the alert_id of the %s liveness ALERT", item, tc.seat)
			}
			if ev := item.Events[0]; ev.Event != InboxEventEscalated || ev.AlertID != item.AlertID {
				t.Errorf("first event = %+v; want the escalated event with alert_id %s", ev, item.AlertID)
			}
			assertAlertItemBody(t, item.Body, map[string]string{
				"ORG_ID": "org-a", "SEAT": tc.seat, "CONDITION": condLiveness, "REASON": tc.reason,
				"ALERT_ID": item.AlertID, "ALERT_TS": alertTS,
			})
			want := map[string]any{"ts": escalatedTS, "org_id": "org-a", "inbox_id": "e1", "alert_id": item.AlertID, "subject": tc.seat, "reason": InboxReasonDirectorDisabled}
			if lines := escalationLines(t, f.o.EscalationsPath); len(lines) != 1 || !reflect.DeepEqual(lines[0], want) {
				t.Errorf("escalations.jsonl = %v; want exactly one line %v", lines, want)
			}
			if !slices.Equal(f.notified, []string{"org org-a raised e1 (ALERT)"}) || len(f.escalated) != 0 {
				t.Errorf("desktop notifications %q, without-inbox notifications %q; want one inbox notification naming ALERT", f.notified, f.escalated)
			}
			if banner := "WATCHDOG ESCALATION: org=org-a item=e1 type=ALERT alert=" + item.AlertID + " -- recorded in "; !strings.Contains(f.stderr.String(), banner) {
				t.Errorf("stderr = %q; want the banner %q", f.stderr.String(), banner)
			}
			if strings.Contains(f.stderr.String(), "not recorded in the inbox") {
				t.Errorf("stderr = %q; want no escalation without the inbox", f.stderr.String())
			}
			assertEscalatedOnce(t, status, item.AlertID)

			f.clk.Advance(time.Minute)
			if err := run.evaluateCycle("org-a", status); err != nil {
				t.Fatalf("the cycle after the escalation: %v", err)
			}
			if n := len(mustReadInbox(t, f.o.Inbox).Items); n != 1 || len(f.notified) != 1 || len(readJSONLFile(t, f.o.EscalationsPath)) != 1 {
				t.Errorf("after one more cycle: %d items, %d notifications, escalations %v; want still one of each", n, len(f.notified), readJSONLFile(t, f.o.EscalationsPath))
			}
		})
	}
}

// AC3 and F13: the watch sends its ALERT item to the human path in the
// cycle that records it unless it can leave the item to the director, with
// leader escalate's reasons: the director switch off gives
// inbox_director_disabled without asking whether the watch accepts work,
// and with the switch on, no answer or a watch that accepts no work gives
// inbox_no_watch. The notified event records the notification's result,
// failed: <err> when it fails.
func TestWatch_Escalation_SendsTheAlertItemAtOnceUnlessThisWatchCanLeaveItToTheDirector(t *testing.T) {
	yes, no := true, false
	cases := []struct {
		name          string
		enabled       bool
		accepts       *bool // nil means no acceptsWork
		notifyErr     error
		wantReason    string
		wantAsks      int
		wantOsascript string
	}{
		{name: "director switch off", accepts: &yes, wantReason: InboxReasonDirectorDisabled, wantOsascript: "ok"},
		{name: "switch on, no answer whether the watch accepts work", enabled: true, wantReason: InboxReasonNoWatch, wantOsascript: "ok"},
		{name: "switch on, a watch that accepts no work", enabled: true, accepts: &no, wantReason: InboxReasonNoWatch, wantAsks: 1, wantOsascript: "ok"},
		{name: "the desktop notification fails", notifyErr: errors.New("exit status 1: Not authorized to send Apple events"),
			wantReason: InboxReasonDirectorDisabled, wantOsascript: "failed: exit status 1: Not authorized to send Apple events"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatchInboxFixture(t)
			f.o.Config.Director.Enabled = tc.enabled
			f.notifyErr = tc.notifyErr
			asks := 0
			var acceptsWork func() bool
			if tc.accepts != nil {
				answer := *tc.accepts
				acceptsWork = func() bool {
					asks++
					return answer
				}
			}
			status, alertID := pendingLeaderAlert(f.clk)
			if err := f.run(acceptsWork).evaluateCycle("org-a", status); err != nil {
				t.Fatalf("evaluateCycle: %v", err)
			}
			item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1")
			if item.Delivery != InboxDeliveryNow || !item.Notified || item.AlertID != alertID {
				t.Fatalf("e1 = %+v; want sent at once with alert_id %s", item, alertID)
			}
			if last := item.Events[len(item.Events)-1]; last.Event != InboxEventNotified || last.Reason != tc.wantReason || last.Osascript != tc.wantOsascript {
				t.Errorf("last event = %+v; want notified with reason %s and osascript %q", last, tc.wantReason, tc.wantOsascript)
			}
			want := map[string]any{"ts": f.clk.Now().UTC().Format(time.RFC3339), "org_id": "org-a", "inbox_id": "e1", "alert_id": alertID, "subject": LeaderIdentity, "reason": tc.wantReason}
			if lines := escalationLines(t, f.o.EscalationsPath); len(lines) != 1 || !reflect.DeepEqual(lines[0], want) {
				t.Errorf("escalations.jsonl = %v; want exactly one line %v", lines, want)
			}
			if asks != tc.wantAsks {
				t.Errorf("acceptsWork asked %d times; want %d", asks, tc.wantAsks)
			}
			if len(f.notified) != 1 || len(f.escalated) != 0 {
				t.Errorf("desktop notifications %q, without-inbox notifications %q; want one inbox notification", f.notified, f.escalated)
			}
			assertEscalatedOnce(t, status, alertID)
		})
	}
}

// AC2: with the director switch on and a watch that accepts work, the ALERT
// item is only recorded, with delivery deadline, for an ALERT about the
// leader (escalated at once) and for a seat's ALERT (escalated at the
// deadman timeout): no escalations.jsonl line and no desktop notification,
// and one stderr line that starts WATCHDOG ESCALATION, names the alert, its
// subject, the reason, and the item, says that it was left to the director
// and when the watch sends it to the human, and gives the inbox show
// command with the ledger's --state-dir.
func TestWatch_Escalation_LeavesTheAlertItemToTheDirectorWhenThisWatchAcceptsWork(t *testing.T) {
	cases := []struct {
		name, subject, reason string
		// pending returns a status of org-a holding the pending ALERT, which
		// the cycle escalates with the fixture's deadman_minutes, and its
		// alert_id.
		pending func(clk *fakeClock, deadmanMinutes int) (*watchStatusFile, string)
	}{
		{"an ALERT about the leader", LeaderIdentity, "leader_is_anomaly_subject",
			func(clk *fakeClock, _ int) (*watchStatusFile, string) { return pendingLeaderAlert(clk) }},
		{"a seat's ALERT past the deadman timeout", "seat-1", "deadman_timeout", pendingSeatAlert},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatchInboxFixture(t)
			f.o.Config.Director.Enabled = true
			f.o.InboxHintStateDir = "/ledgers/feature a"
			status, alertID := tc.pending(f.clk, f.o.Config.DeadmanMinutes)
			if err := f.run(func() bool { return true }).evaluateCycle("org-a", status); err != nil {
				t.Fatalf("evaluateCycle: %v", err)
			}
			item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1")
			if item.Type != protocol.TypeAlert || item.Delivery != InboxDeliveryDeadline || item.State != InboxStateOpen || item.Notified || item.AlertID != alertID {
				t.Fatalf("e1 = %+v; want an open, unnotified ALERT item with delivery deadline and alert_id %s", item, alertID)
			}
			if m, err := protocol.Parse(item.Body); err != nil || m.Fields["SEAT"] != tc.subject || m.Fields["REASON"] != tc.reason {
				t.Errorf("e1 body %q (parse error %v); want SEAT %s and REASON %s", item.Body, err, tc.subject, tc.reason)
			}
			if got := fileOrAbsent(t, f.o.EscalationsPath); got != "<absent>" {
				t.Errorf("escalations.jsonl = %q; want absent for an item left to the director", got)
			}
			if len(f.notified) != 0 || len(f.escalated) != 0 {
				t.Errorf("desktop notifications %q, without-inbox notifications %q; want none", f.notified, f.escalated)
			}
			want := "WATCHDOG ESCALATION: org=org-a alert=" + alertID + " subject=" + tc.subject + " reason=" + tc.reason + " item=e1 -- left in " + f.o.Inbox.Path() + " for the director; " +
				"if it is still open after [org.director].ack_minutes, or not resolved after [org.director].resolve_minutes, ralph org watch --all sends it to the human; read it with: " +
				InboxCommand("show", "e1", "/ledgers/feature a") + "\n"
			if got := f.stderr.String(); got != want {
				t.Errorf("stderr = %q; want the one line %q", got, want)
			}
			assertEscalatedOnce(t, status, alertID)
		})
	}
}

// AC3: an ALERT item whose send to the human path is not completed (here
// escalations.jsonl cannot be written) is still recorded, so the escalation
// does not take the path without the inbox. It stays open with no notified
// event, one stderr line names `ralph org inbox notify <id>`, and the ack
// deadline sends it once escalations.jsonl can be written.
func TestWatch_Escalation_ASendThatIsNotCompletedNamesInboxNotifyAndLeavesTheItemToTheAckDeadline(t *testing.T) {
	f := newWatchInboxFixture(t)
	if err := os.Mkdir(f.o.EscalationsPath, 0o755); err != nil {
		t.Fatal(err)
	}
	status, alertID := pendingLeaderAlert(f.clk)
	if err := f.run(nil).evaluateCycle("org-a", status); err != nil {
		t.Fatalf("evaluateCycle: %v", err)
	}
	item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1")
	if item.State != InboxStateOpen || item.Notified || item.AlertID != alertID {
		t.Fatalf("e1 = %+v; want open and unnotified with alert_id %s", item, alertID)
	}
	stderr := f.stderr.String()
	if !strings.Contains(stderr, "WATCHDOG ESCALATION: org=org-a item=e1 -- recorded, but the send to the human was not completed (") ||
		!strings.Contains(stderr, "run `"+InboxCommand("notify", "e1", "")+"` to send it again") {
		t.Errorf("stderr = %q; want the line naming the inbox notify command for e1", stderr)
	}
	if strings.Contains(stderr, "not recorded in the inbox") || len(f.escalated) != 0 {
		t.Errorf("stderr = %q, without-inbox notifications %q; want no escalation without the inbox", stderr, f.escalated)
	}
	assertEscalatedOnce(t, status, alertID)

	if err := os.Remove(f.o.EscalationsPath); err != nil {
		t.Fatal(err)
	}
	sweep := f.o.SweepInboxDeadlines(f.clk.Now().Add(11*time.Minute), nil, &bytes.Buffer{})
	assertSweep(t, "the sweep past the ack deadline", sweep, []string{"e1 " + InboxReasonAckDeadline}, nil, nil)
}

// AC4: an alert the inbox already holds as an item (a watch that recorded
// it and ended before saving its status file) gets no second item. The
// item is sent to the human path under its own ID, with this watch's
// reason, only when it is open with no notified event and this watch
// cannot leave it to the director, whatever delivery it was recorded with.
// An open, unnotified item this watch can leave to the director is not
// sent, even one recorded to be sent at once: it stays unnotified for the
// director and the ack deadline. One stderr line names the item either way,
// and the alert leaves the pending alerts.
func TestWatch_Escalation_AnAlertAlreadyInTheInboxGetsNoSecondItem(t *testing.T) {
	yes, no := true, false
	markNotified := func(t *testing.T, s *InboxStore, id string) {
		if err := s.AppendNotified(id, InboxReasonNoWatch, "ok"); err != nil {
			t.Fatal(err)
		}
	}
	markAcked := func(t *testing.T, s *InboxStore, id string) {
		if _, err := s.Ack(id); err != nil {
			t.Fatal(err)
		}
	}
	markResolved := func(t *testing.T, s *InboxStore, id string) {
		if err := s.Resolve(id, "docs/reports/x.md"); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name     string
		delivery string
		then     func(*testing.T, *InboxStore, string)
		enabled  bool
		accepts  *bool
		// wantReason is the reason of the send; "" means nothing is sent.
		wantReason string
	}{
		{name: "already notified", delivery: InboxDeliveryNow, then: markNotified},
		{name: "open, to be sent at once, this watch accepts work, left to the director and the deadlines", delivery: InboxDeliveryNow, enabled: true, accepts: &yes},
		{name: "open, to be sent at once, this watch accepts no work", delivery: InboxDeliveryNow, enabled: true, accepts: &no, wantReason: InboxReasonNoWatch},
		{name: "open, to be sent at once, director switch off", delivery: InboxDeliveryNow, wantReason: InboxReasonDirectorDisabled},
		{name: "open, left to the director, this watch accepts work", delivery: InboxDeliveryDeadline, enabled: true, accepts: &yes},
		{name: "open, left to the director, this watch accepts no work", delivery: InboxDeliveryDeadline, enabled: true, accepts: &no, wantReason: InboxReasonNoWatch},
		{name: "open, left to the director, director switch off now", delivery: InboxDeliveryDeadline, wantReason: InboxReasonDirectorDisabled},
		{name: "acked, this watch accepts no work", delivery: InboxDeliveryDeadline, then: markAcked, enabled: true, accepts: &no},
		{name: "resolved, director switch off", delivery: InboxDeliveryNow, then: markResolved},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatchInboxFixture(t)
			f.o.Config.Director.Enabled = tc.enabled
			var acceptsWork func() bool
			if tc.accepts != nil {
				answer := *tc.accepts
				acceptsWork = func() bool { return answer }
			}
			status, alertID := pendingLeaderAlert(f.clk)
			id := escalateAlertItem(t, f.o.Inbox, alertID, tc.delivery)
			if tc.then != nil {
				tc.then(t, f.o.Inbox, id)
			}
			before := mustReadInbox(t, f.o.Inbox)

			if err := f.run(acceptsWork).evaluateCycle("org-a", status); err != nil {
				t.Fatalf("evaluateCycle: %v", err)
			}
			after := mustReadInbox(t, f.o.Inbox)
			if ids := inboxItemIDs(after); !slices.Equal(ids, []string{id}) {
				t.Fatalf("inbox items = %v; want only %s", ids, id)
			}
			lines := escalationLines(t, f.o.EscalationsPath)
			if tc.wantReason == "" {
				if !reflect.DeepEqual(after, before) || len(lines) != 0 || len(f.notified) != 0 {
					t.Errorf("inbox %+v (was %+v), escalations %v, notifications %q; want nothing sent or written", after, before, lines, f.notified)
				}
				if line := "-- already in the inbox as " + id + " ("; !strings.Contains(f.stderr.String(), line) || !strings.Contains(f.stderr.String(), "nothing new is recorded or sent") {
					t.Errorf("stderr = %q; want the line saying %s is already in the inbox and nothing was sent", f.stderr.String(), id)
				}
			} else {
				want := map[string]any{"ts": f.clk.Now().UTC().Format(time.RFC3339), "org_id": "org-a", "inbox_id": id, "alert_id": alertID, "subject": LeaderIdentity, "reason": tc.wantReason}
				if len(lines) != 1 || !reflect.DeepEqual(lines[0], want) {
					t.Errorf("escalations.jsonl = %v; want exactly one line %v", lines, want)
				}
				if item := inboxItemByID(t, after, id); !item.Notified || item.Events[len(item.Events)-1].Reason != tc.wantReason || len(f.notified) != 1 {
					t.Errorf("%s = %+v, notifications %q; want it notified once with %s", id, item, f.notified, tc.wantReason)
				}
				if line := "-- already in the inbox as " + id + " (state=open notified=false delivery=" + tc.delivery + "); sending that item to the human"; !strings.Contains(f.stderr.String(), line) {
					t.Errorf("stderr = %q; want %q", f.stderr.String(), line)
				}
			}
			if len(f.escalated) != 0 {
				t.Errorf("without-inbox notifications %q; want none", f.escalated)
			}
			assertEscalatedOnce(t, status, alertID)
		})
	}
}

// setDesktopNotifyGOOS sets desktopNotifyGOOS to goos for the test, so a
// nil escalateFn or Org.DesktopNotify reads as that GOOS (notifyBounded)
// and, off darwin, never reaches osascript.
func setDesktopNotifyGOOS(t *testing.T, goos string) {
	t.Helper()
	saved := desktopNotifyGOOS
	desktopNotifyGOOS = goos
	t.Cleanup(func() { desktopNotifyGOOS = saved })
}

// fakeOsascriptOnPath puts a fake osascript alone on PATH for the test, so
// a nil escalateFn or Org.DesktopNotify with desktopNotifyGOOS set to
// darwin runs it, never the real one, on every GOOS. It returns the file
// the fake appends each call's arguments to, one line per call. The script
// runs builtins only, as PATH holds nothing else, and exits with the
// status of that append, 0 when the call was recorded.
func fakeOsascriptOnPath(t *testing.T) string {
	t.Helper()
	bin := t.TempDir()
	argv := filepath.Join(bin, "argv")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> " + shellQuote(argv) + "\n"
	if err := os.WriteFile(filepath.Join(bin, "osascript"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	return argv
}

// AC5 and F13: when the inbox cannot take the escalation (no inbox store,
// an inbox that cannot be read, one that cannot be written), it takes the
// path without the inbox, in this order: the stderr banner, which names the
// cause; the desktop notification (WatchAllParams.Escalate); and one
// escalations.jsonl line with alert_id, subject, reason, and the
// notification's result as osascript: ok, failed: <err>, or skipped for a
// nil Escalate off darwin. No inbox notification is sent.
func TestWatch_Escalation_WithoutTheInboxWritesTheBannerThenNotifiesThenWritesOneLine(t *testing.T) {
	noInbox := func(t *testing.T, f *watchInboxFixture) { f.o.Inbox = nil }
	unreadable := func(t *testing.T, f *watchInboxFixture) {
		if err := os.Mkdir(f.o.Inbox.Path(), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	unwritable := func(t *testing.T, f *watchInboxFixture) {
		if err := os.Mkdir(filepath.Join(f.dir, inboxLockFile), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cases := []struct {
		name       string
		breakInbox func(*testing.T, *watchInboxFixture)
		wantCause  string
		// readable is set when the inbox can still be read, to check that it
		// holds no item afterwards.
		readable      bool
		escalateErr   error
		nilEscalate   bool
		wantOsascript string
	}{
		{name: "no inbox store", breakInbox: noInbox, wantCause: "no inbox store is configured", wantOsascript: "ok"},
		{name: "an inbox that cannot be read", breakInbox: unreadable, wantCause: "org: inbox: read ", wantOsascript: "ok"},
		{name: "an inbox that cannot be written", breakInbox: unwritable, wantCause: "org: open inbox lock file ", readable: true, wantOsascript: "ok"},
		{name: "the notification fails", breakInbox: noInbox, wantCause: "no inbox store is configured",
			escalateErr: errors.New("exit status 1: Not authorized to send Apple events"), wantOsascript: "failed: exit status 1: Not authorized to send Apple events"},
		{name: "no notification off darwin", breakInbox: noInbox, wantCause: "no inbox store is configured", nilEscalate: true, wantOsascript: "skipped"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatchInboxFixture(t)
			tc.breakInbox(t, f)
			f.escalateErr = tc.escalateErr
			banner := ""
			var escalationsAtNotify string
			f.atEscalate = func() {
				banner = f.stderr.String()
				escalationsAtNotify = fileOrAbsent(t, f.o.EscalationsPath)
			}
			status, alertID := pendingLeaderAlert(f.clk)
			run := f.run(nil)
			if tc.nilEscalate {
				setDesktopNotifyGOOS(t, "linux")
				run.escalateFn = nil
			}
			if err := run.evaluateCycle("org-a", status); err != nil {
				t.Fatalf("evaluateCycle: %v", err)
			}

			wantBanner := "WATCHDOG ESCALATION: org=org-a alert=" + alertID + " subject=leader reason=leader_is_anomaly_subject -- see " + f.o.EscalationsPath + " (not recorded in the inbox: "
			if !strings.HasPrefix(f.stderr.String(), wantBanner) || !strings.Contains(f.stderr.String(), tc.wantCause) || strings.Count(f.stderr.String(), "\n") != 1 {
				t.Errorf("stderr = %q; want the one banner %q... naming %q", f.stderr.String(), wantBanner, tc.wantCause)
			}
			wantNotify := []string{"ralph org watch: escalation for org org-a (leader_is_anomaly_subject)"}
			if tc.nilEscalate {
				wantNotify = nil
			}
			if !slices.Equal(f.escalated, wantNotify) || len(f.notified) != 0 {
				t.Errorf("without-inbox notifications %q, inbox notifications %q; want %q and none", f.escalated, f.notified, wantNotify)
			}
			if !tc.nilEscalate && (!strings.HasPrefix(banner, wantBanner) || escalationsAtNotify != "<absent>") {
				t.Errorf("at the notification, stderr %q and escalations.jsonl %q; want the banner written and no line yet", banner, escalationsAtNotify)
			}
			want := map[string]any{"ts": f.clk.Now().UTC().Format(time.RFC3339), "org_id": "org-a", "alert_id": alertID, "subject": LeaderIdentity,
				"reason": "leader_is_anomaly_subject", "osascript": tc.wantOsascript}
			if lines := escalationLines(t, f.o.EscalationsPath); len(lines) != 1 || !reflect.DeepEqual(lines[0], want) {
				t.Errorf("escalations.jsonl = %v; want exactly one line %v", lines, want)
			}
			if tc.readable {
				if items := mustReadInbox(t, f.o.Inbox).Items; len(items) != 0 {
					t.Errorf("inbox = %+v; want no item", items)
				}
			}
			assertEscalatedOnce(t, status, alertID)
		})
	}
}

// Self-review F1: on the path without the inbox, an escalations.jsonl line
// that cannot be written is one more stderr line after the banner, and that
// line carries the notification's result as notification=<result> (ok,
// failed: <err>, or skipped), so the result is not lost with the line.
func TestWatch_Escalation_WithoutTheInboxALineThatCannotBeWrittenStillNamesTheNotificationsResult(t *testing.T) {
	cases := []struct {
		name          string
		escalateErr   error
		nilEscalate   bool
		wantOsascript string
	}{
		{name: "the notification is sent", wantOsascript: "ok"},
		{name: "the notification fails", escalateErr: errors.New("exit status 1: Not authorized to send Apple events"),
			wantOsascript: "failed: exit status 1: Not authorized to send Apple events"},
		{name: "no notification off darwin", nilEscalate: true, wantOsascript: "skipped"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatchInboxFixture(t)
			f.o.Inbox = nil
			if err := os.Mkdir(f.o.EscalationsPath, 0o755); err != nil {
				t.Fatal(err)
			}
			f.escalateErr = tc.escalateErr
			status, alertID := pendingLeaderAlert(f.clk)
			run := f.run(nil)
			if tc.nilEscalate {
				setDesktopNotifyGOOS(t, "linux")
				run.escalateFn = nil
			}
			if err := run.evaluateCycle("org-a", status); err != nil {
				t.Fatalf("evaluateCycle: %v", err)
			}

			lines := strings.Split(strings.TrimSuffix(f.stderr.String(), "\n"), "\n")
			if len(lines) != 2 || !strings.HasPrefix(lines[0], "WATCHDOG ESCALATION: org=org-a alert="+alertID+" ") {
				t.Fatalf("stderr = %q; want the banner and one more line", f.stderr.String())
			}
			wantStart, wantEnd := "watchdog: escalation of alert "+alertID+" not recorded: ", "; notification="+tc.wantOsascript
			if !strings.HasPrefix(lines[1], wantStart) || !strings.HasSuffix(lines[1], wantEnd) || !strings.Contains(lines[1], f.o.EscalationsPath) {
				t.Errorf("second stderr line = %q; want %q, the error naming %s, then %q", lines[1], wantStart, f.o.EscalationsPath, wantEnd)
			}
			assertEscalatedOnce(t, status, alertID)
		})
	}
}

// Self-review F4: a pending ALERT whose values in the watch-status file are
// not ones the watch writes (a subject with a line break, an alert_id whose
// condition holds one, an alert_id of another shape, a ts with a line
// break) records no item: a line break in a header value would add a
// header to the body the director reads. The escalation takes the path
// without the inbox instead: one stderr line, the banner with the values
// escaped and the cause, the notification, and the escalations.jsonl line
// with the values as they are.
func TestWatch_Escalation_AnAlertWithValuesTheWatchDoesNotWriteRecordsNoItem(t *testing.T) {
	const inject = "\nINJECTED: yes"
	cases := []struct {
		name string
		// alert returns the pending ALERT of org-a and its alert_id, raised
		// before clk's time; the cycle at clk's time escalates it.
		alert      func(clk *fakeClock) (string, *watchPendingAlert)
		wantReason string
		wantCause  string
	}{
		{name: "a subject with a line break", alert: func(clk *fakeClock) (string, *watchPendingAlert) {
			id := fmt.Sprintf("%s@%d", conditionKey("org-a", "seat-1", condLiveness), clk.Now().UnixNano())
			return id, &watchPendingAlert{AlertID: id, TS: clk.Now().Add(-6 * time.Minute).UTC().Format(time.RFC3339), Subject: "seat-1" + inject, HistoryLeaderLines: -1}
		}, wantReason: "deadman_timeout", wantCause: `the ALERT's subject in the watch-status file is not a seat: org: invalid seat_id "seat-1\nINJECTED: yes"`},
		{name: "an alert_id whose condition holds a line break", alert: func(clk *fakeClock) (string, *watchPendingAlert) {
			id := fmt.Sprintf("%s%s@%d", conditionKey("org-a", LeaderIdentity, condLiveness), inject, clk.Now().UnixNano())
			return id, &watchPendingAlert{AlertID: id, TS: clk.Now().UTC().Format(time.RFC3339), Subject: LeaderIdentity, HistoryLeaderLines: -1}
		}, wantReason: "leader_is_anomaly_subject", wantCause: "the alert_id in the watch-status file does not have the shape"},
		{name: "an alert_id of another shape", alert: func(clk *fakeClock) (string, *watchPendingAlert) {
			id := fmt.Sprintf("org-a/%s@%d", LeaderIdentity, clk.Now().UnixNano())
			return id, &watchPendingAlert{AlertID: id, TS: clk.Now().UTC().Format(time.RFC3339), Subject: LeaderIdentity, HistoryLeaderLines: -1}
		}, wantReason: "leader_is_anomaly_subject", wantCause: "the alert_id in the watch-status file does not have the shape"},
		{name: "a ts with a line break", alert: func(clk *fakeClock) (string, *watchPendingAlert) {
			id := fmt.Sprintf("%s@%d", conditionKey("org-a", LeaderIdentity, condLiveness), clk.Now().UnixNano())
			return id, &watchPendingAlert{AlertID: id, TS: clk.Now().UTC().Format(time.RFC3339) + inject, Subject: LeaderIdentity, HistoryLeaderLines: -1}
		}, wantReason: "leader_is_anomaly_subject", wantCause: "the ALERT's ts in the watch-status file holds a control character"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newWatchInboxFixture(t)
			f.o.Config.DeadmanMinutes = 5
			alertID, pending := tc.alert(f.clk)
			status := &watchStatusFile{OrgID: "org-a", PendingAlerts: map[string]*watchPendingAlert{alertID: pending}}
			if err := f.run(nil).evaluateCycle("org-a", status); err != nil {
				t.Fatalf("evaluateCycle: %v", err)
			}

			if items := mustReadInbox(t, f.o.Inbox).Items; len(items) != 0 {
				t.Errorf("inbox = %+v; want no item", items)
			}
			wantBanner := "WATCHDOG ESCALATION: org=org-a alert=" + PrintableInboxText(alertID, false) + " subject=" + PrintableInboxText(pending.Subject, false) +
				" reason=" + tc.wantReason + " -- see " + f.o.EscalationsPath + " (not recorded in the inbox: "
			if got := f.stderr.String(); !strings.HasPrefix(got, wantBanner) || !strings.Contains(got, tc.wantCause) || strings.Count(got, "\n") != 1 {
				t.Errorf("stderr = %q; want the one banner %q... naming %q", got, wantBanner, tc.wantCause)
			}
			if len(f.escalated) != 1 || len(f.notified) != 0 {
				t.Errorf("without-inbox notifications %q, inbox notifications %q; want one and none", f.escalated, f.notified)
			}
			want := map[string]any{"ts": f.clk.Now().UTC().Format(time.RFC3339), "org_id": "org-a", "alert_id": alertID, "subject": pending.Subject,
				"reason": tc.wantReason, "osascript": "ok"}
			if lines := escalationLines(t, f.o.EscalationsPath); len(lines) != 1 || !reflect.DeepEqual(lines[0], want) {
				t.Errorf("escalations.jsonl = %v; want exactly one line %v", lines, want)
			}
			assertEscalatedOnce(t, status, alertID)
		})
	}
}

// AC6 and D9: on the osascript path, which a nil Org.DesktopNotify and a nil
// WatchAllParams.Escalate take on darwin, the desktop notification of a
// watch escalation is titled ralph org watch: an ALERT item sent at once
// and at its ack deadline (the inbox's human path, inboxDesktopNotify), and
// an escalation the inbox could not take (realEscalate). A leader's
// escalate is still titled ralph org escalate, so the title osascript gets
// is the send's, not a fixed one. Each send is one osascript call, and the
// ledger records its result as ok.
func TestWatch_Escalation_TheOsascriptNotificationIsTitledAfterTheWatch(t *testing.T) {
	// cycle runs one cycle of run that escalates a pending ALERT about the
	// leader of org-a.
	cycle := func(t *testing.T, f *watchInboxFixture, run *watchRun) {
		t.Helper()
		status, _ := pendingLeaderAlert(f.clk)
		if err := run.evaluateCycle("org-a", status); err != nil {
			t.Fatalf("evaluateCycle: %v", err)
		}
	}
	// notifiedResult returns the osascript result of e1's last event, which
	// must be a notified event.
	notifiedResult := func(t *testing.T, f *watchInboxFixture) string {
		t.Helper()
		item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1")
		last := item.Events[len(item.Events)-1]
		if last.Event != InboxEventNotified {
			t.Fatalf("e1's last event = %+v; want a notified event", last)
		}
		return last.Osascript
	}
	cases := []struct {
		name string
		// send makes one send to the human on f's Org, whose DesktopNotify is
		// nil, and returns the result the ledger recorded for its
		// notification.
		send     func(t *testing.T, f *watchInboxFixture) string
		wantArgv string
	}{
		{name: "an ALERT item sent at once", send: func(t *testing.T, f *watchInboxFixture) string {
			cycle(t, f, f.run(nil))
			return notifiedResult(t, f)
		}, wantArgv: `-e display notification "org org-a raised e1 (ALERT)" with title "ralph org watch"`},
		{name: "an ALERT item sent at its ack deadline", send: func(t *testing.T, f *watchInboxFixture) string {
			f.o.Config.Director.Enabled = true
			cycle(t, f, f.run(func() bool { return true }))
			sweep := f.o.SweepInboxDeadlines(f.clk.Now().Add(11*time.Minute), nil, &f.stderr)
			assertSweep(t, "the sweep past the ack deadline", sweep, []string{"e1 " + InboxReasonAckDeadline}, nil, nil)
			return notifiedResult(t, f)
		}, wantArgv: `-e display notification "org org-a: e1 (ALERT) not acked within 10 min" with title "ralph org watch"`},
		{name: "an escalation the inbox could not take", send: func(t *testing.T, f *watchInboxFixture) string {
			f.o.Inbox = nil
			run := f.run(nil)
			run.escalateFn = nil
			cycle(t, f, run)
			lines := escalationLines(t, f.o.EscalationsPath)
			if len(lines) != 1 {
				t.Fatalf("escalations.jsonl = %v; want one line", lines)
			}
			got, _ := lines[0]["osascript"].(string)
			return got
		}, wantArgv: `-e display notification "ralph org watch: escalation for org org-a (leader_is_anomaly_subject)" with title "ralph org watch"`},
		{name: "a leader's escalate, for comparison", send: func(t *testing.T, f *watchInboxFixture) string {
			if res := f.o.Escalate(EscalateParams{OrgID: "org-a", Text: "TYPE: QUESTION\n\nwhich?", Banner: &f.stderr}); res.Err != nil || !res.Notified {
				t.Fatalf("Escalate = %+v; want e1 sent at once", res)
			}
			return notifiedResult(t, f)
		}, wantArgv: `-e display notification "org org-a raised e1 (QUESTION)" with title "ralph org escalate"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setDesktopNotifyGOOS(t, "darwin")
			argv := fakeOsascriptOnPath(t)
			f := newWatchInboxFixture(t)
			f.o.DesktopNotify = nil
			if got := tc.send(t, f); got != "ok" {
				t.Errorf("recorded notification result = %q; want ok", got)
			}
			if got := fileOrAbsent(t, argv); got != tc.wantArgv+"\n" {
				t.Errorf("osascript calls = %q; want the one call %q", got, tc.wantArgv)
			}
			if len(f.escalated) != 0 {
				t.Errorf("notifications of the stub escalateFn = %q; want none", f.escalated)
			}
		})
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
		if err := run.evaluateCycle("org-a", status); err != nil {
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

// --- RunWatchAll's public loop (Cycles-capped: Cycles > 0 returns before
// the wait between cycles) ---------------------------------------------------

func TestRunWatchAll_Cycles_RunsExactlyNThenReturns(t *testing.T) {
	f := newWatchAllFixture(t)
	f.spawn(t, "org-a", "seat-1")

	var onCycleCount int
	err := runWatchAll(t, f.o, context.Background(), f.params(2), WatchHooks{OnCycle: func(int, time.Time) { onCycleCount++ }})
	if err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if onCycleCount != 2 {
		t.Fatalf("expected exactly 2 cycles, got %d", onCycleCount)
	}
	if status := readWatchStatusFile(t, f.dir, "org-a"); status.Cycles != 2 {
		t.Fatalf("expected cycles=2 in the persisted status, got %d", status.Cycles)
	}
}

// RunWatchAll without a StatusDir is refused before it takes any lock or
// writes anything (a blank dir would put watch.lock in the working
// directory); org_id is no parameter of it.
func TestRunWatchAll_RequiresStatusDir(t *testing.T) {
	f := newWatchAllFixture(t)
	cwd := t.TempDir()
	t.Chdir(cwd)
	for _, dir := range []string{"", "  "} {
		p := f.params(1)
		p.StatusDir = dir
		err := f.o.RunWatchAll(context.Background(), p, WatchHooks{OnCycle: func(int, time.Time) {
			t.Errorf("a watch with StatusDir %q ran a cycle", dir)
		}})
		if err == nil || !strings.Contains(err.Error(), "state dir is required") {
			t.Errorf("RunWatchAll with StatusDir %q = %v; want the state dir required", dir, err)
		}
	}
	if entries, err := os.ReadDir(cwd); err != nil || len(entries) != 0 {
		t.Errorf("the working directory after the refusals holds %v (%v); want nothing written", entries, err)
	}
}

// TestRunWatchAll_MultipleOrgs_SeparateStatusFiles pins the self-review H-1
// fix: two orgs watched from the same repository (one shared StatusDir,
// exactly as manifest.jsonl/model-receipts.jsonl are already shared there),
// with the same seat id, must get distinct watch-status files, not silently
// clobber each other's OrgID/Cycles/WatchdogJoined/SeatSnapshots state via
// one fixed file name.
func TestRunWatchAll_MultipleOrgs_SeparateStatusFiles(t *testing.T) {
	f := newWatchAllFixture(t)
	f.spawn(t, "org-a", "seat-1")
	f.spawn(t, "org-b", "seat-1")

	if err := runWatchAll(t, f.o, context.Background(), f.params(1), WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}

	dir := f.dir
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

// TestWatch_Deadman_LegacyWatchdogStopEvent_DoesNotClearPendingAlert pins
// the legacy-manifest compatibility guard in leaderActivityEventCount (see
// its doc comment, case (b)): a `stopped` event carrying the pre-#152
// watchdog's "reason=watchdog_..." suffix must not count as leader activity,
// even though it names a real (unrelated) seat. The event is appended
// directly via Manifest.Append rather than through Stop, because no
// current code path produces this shape any more -- PR #152 removed the
// org budget concept that used to call Stop with a watchdog Reason.
func TestWatch_Deadman_LegacyWatchdogStopEvent_DoesNotClearPendingAlert(t *testing.T) {
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

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises seat-1's liveness ALERT): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected exactly 1 pending alert after cycle 1, got %+v", status.PendingAlerts)
	}

	// Reproduce a pre-#152 watchdog cutoff's Details string: `pane=ok
	// leave=ok` (what Stop wrote) plus its " reason=watchdog_..." suffix
	// -- the only part of the old shape the exclusion predicate reads.
	// The other fields the old producer recorded onto the event (driver,
	// model, worktree, pane_id, agmsg_team) are omitted, since
	// nothing in leaderActivityEventCount consults them; no current code
	// path emits this shape at all any more.
	if err := o.Manifest.Append(ManifestEvent{TS: clk.Now().UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: "seat-2", Event: EventStopped, Role: "worker", Details: "pane=ok leave=ok reason=watchdog_cutoff seat_wall_clock=30m observed=31m0s"}); err != nil {
		t.Fatalf("append legacy watchdog stopped event: %v", err)
	}

	clk.Advance(6 * time.Minute) // past DeadmanMinutes, no genuine leader/seat activity
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2 (deadman sweep): %v", err)
	}

	lines := readJSONLFile(t, escalationsPath)
	if len(lines) != 1 {
		t.Fatalf("expected the pending alert to still escalate (a legacy watchdog cutoff of seat-2 must not count as leader activity), got %d escalation(s): %v", len(lines), lines)
	}
}

// TestWatch_Deadman_PersistedAlertBaseline_SurvivesLegacyWatchdogStop pins
// the upgrade-boundary contract behind keeping leaderActivityEventCount's
// "reason=watchdog_" exclusion (plan org-stop-reason-removal, Codex plan
// advisory HIGH-1): a pending alert persisted by a pre-#152 watchdog stored
// a ManifestLen baseline that excluded the watchdog's own cutoff `stopped`
// events, and checkDeadman compares that baseline against a full recount.
// If the exclusion were removed, an unchanged manifest containing one legacy
// cutoff would recount one higher than the baseline and the alert would be
// cleared as "leader activity" -- silently, with dedupe suppressing any
// replacement. Two cases share a fixture: (i) no new events -> the alert
// survives and escalates exactly once at its deadline; (ii) a genuine leader
// `sent` event -> the alert clears with no escalation.
//
// Relation to TestWatch_Deadman_LegacyWatchdogStopEvent_DoesNotClearPendingAlert:
// that test's alert baseline is recorded live by sendAlert (with the guard
// already applied) before the legacy cutoff exists, so it pins the
// "cutoff after baseline" case entirely in-process. What this test adds is
// a baseline restored from persisted status JSON instead of held in
// memory -- the actual upgrade boundary a pre-#152 binary's on-disk state
// crosses. Only subtest (i) discriminates the guard; subtest (ii) is a
// sanity check that passes with or without it, since it repeats the
// positive `sent`-clears-the-alert case already covered by
// TestWatch_Deadman_SeatSentEvent_ClearsPendingAlert_LegacyWatchdogStopDoesNot.
//
// Discrimination note: the fixture captures baseline from the manifest
// BEFORE the legacy cutoff event below is appended, mirroring the real
// chronology (an ALERT is raised and its ManifestLen baseline recorded;
// only afterwards does the watchdog cut a different seat off). Capturing
// baseline AFTER the cutoff was already present would make baseline and
// case (i)'s recount two calls to the identical leaderActivityEventCount on
// the identical, unchanged event set -- always equal regardless of whether
// the exclusion exists, so that ordering could never actually exercise the
// guard. With baseline captured first, case (i)'s recount (which does see
// the cutoff) equals baseline only because the exclusion keeps the cutoff
// from counting; remove the exclusion and the same recount would read one
// higher than the unchanged baseline and wrongly clear the alert instead of
// escalating. The fixture's "lead_agent_get": "" and "history_lead_lines":
// -1 are also load-bearing: they make checkDeadman skip its herdr-probe and
// agmsg-history activity sources (pending.LeaderAgentGet != "" and
// pending.HistoryLeaderLines >= 0 both stay false), so the manifest recount
// above is the only signal deciding the outcome.
func TestWatch_Deadman_PersistedAlertBaseline_SurvivesLegacyWatchdogStop(t *testing.T) {
	// baselineFixture holds one fixture instance -- returned by pointer so
	// each subtest gets an independent org/status/clock and reads the alert
	// id the fixture actually wrote into the status JSON.
	type baselineFixture struct {
		o               *Org
		run             *watchRun
		status          *watchStatusFile
		escalationsPath string
		alertID         string
		clk             *fakeClock
		escalateCalls   int
	}
	newFixture := func(t *testing.T) *baselineFixture {
		t.Helper()
		f := &baselineFixture{}
		var h *fakeWatchHerdr
		f.o, h, _, f.clk = testWatchOrg(t)
		f.o.Config.DeadmanMinutes = 5
		if r := f.o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
			t.Fatalf("spawn seat-1 failed: %+v", r)
		}
		if r := f.o.Spawn(watchSpawnParams("org-a", "seat-2", "worker")); r.Outcome != SpawnOutcomeSpawned {
			t.Fatalf("spawn seat-2 failed: %+v", r)
		}
		target := herdrAgentName("org-a", "seat-1")
		h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")} // sticky liveness ALERT for seat-1

		// baseline is captured BEFORE the legacy cutoff below is appended --
		// see the discrimination note above.
		rr, err := f.o.Manifest.Read()
		if err != nil {
			t.Fatalf("read manifest: %v", err)
		}
		baseline := leaderActivityEventCount(rr.Events, "org-a")

		// The watchdog's own legacy (pre-#152) cutoff of seat-2, appended
		// directly since no current code path produces this shape.
		if err := f.o.Manifest.Append(ManifestEvent{TS: f.clk.Now().UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: "seat-2", Event: EventStopped, Role: "worker", Details: "pane=ok leave=ok reason=watchdog_cutoff observed=31m0s"}); err != nil {
			t.Fatalf("append legacy watchdog stopped event: %v", err)
		}

		condKey := conditionKey("org-a", "seat-1", condLiveness)
		f.alertID = condKey + "@1000000000"
		alertTS := f.clk.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339)

		fixture := fmt.Sprintf(`{
  "org_id": "org-a",
  "conditions": {
    %q: {"active": true, "first_ts": %q}
  },
  "pending_alerts": {
    %q: {
      "alert_id": %q,
      "ts": %q,
      "subject": "seat-1",
      "manifest_len": %d,
      "lead_agent_get": "",
      "history_lead_lines": -1
    }
  }
}`, condKey, alertTS, f.alertID, f.alertID, alertTS, baseline)

		escalateFn := func(context.Context, string) error {
			f.escalateCalls++
			return nil
		}
		var statusPath string
		f.run, statusPath, f.escalationsPath = newTestWatchRun(f.o, nil, escalateFn, &bytes.Buffer{})
		if err := os.WriteFile(statusPath, []byte(fixture), 0o644); err != nil {
			t.Fatalf("write status fixture: %v", err)
		}
		f.status, err = loadWatchStatus(statusPath, "org-a")
		if err != nil {
			t.Fatalf("loadWatchStatus: %v", err)
		}
		if len(f.status.PendingAlerts) != 1 || len(f.status.Conditions) != 1 {
			t.Fatalf("setup: expected the fixture to load with 1 pending alert and 1 condition, got pending=%+v conditions=%+v",
				f.status.PendingAlerts, f.status.Conditions)
		}
		return f
	}

	t.Run("no new events: alert survives and escalates once at deadline", func(t *testing.T) {
		f := newFixture(t)

		if err := f.run.evaluateCycle("org-a", f.status); err != nil {
			t.Fatalf("evaluateCycle: %v", err)
		}

		if f.escalateCalls != 1 {
			t.Fatalf("expected exactly 1 escalation (the persisted baseline must survive the legacy watchdog cutoff), got %d", f.escalateCalls)
		}
		if _, ok := f.status.PendingAlerts[f.alertID]; ok {
			t.Errorf("expected the pending alert to be removed after escalation, got %+v", f.status.PendingAlerts)
		}
		if !f.status.Escalated[f.alertID] {
			t.Errorf("expected alert %q to be recorded in Escalated, got %+v", f.alertID, f.status.Escalated)
		}
		lines := readJSONLFile(t, f.escalationsPath)
		if len(lines) != 1 {
			t.Fatalf("expected exactly 1 escalation record, got %d: %v", len(lines), lines)
		}
	})

	t.Run("genuine leader sent event clears the alert without escalation", func(t *testing.T) {
		f := newFixture(t)

		if err := f.o.Manifest.Append(ManifestEvent{TS: f.clk.Now().UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: "seat-2", Event: EventSent, Details: "leader sends to seat-2"}); err != nil {
			t.Fatalf("append sent event: %v", err)
		}

		if err := f.run.evaluateCycle("org-a", f.status); err != nil {
			t.Fatalf("evaluateCycle: %v", err)
		}

		if f.escalateCalls != 0 {
			t.Errorf("expected no escalation once genuine leader activity is recorded, got %d", f.escalateCalls)
		}
		if len(f.status.PendingAlerts) != 0 {
			t.Errorf("expected the pending alert to clear after a genuine sent event, got %+v", f.status.PendingAlerts)
		}
		assertNoEscalations(t, f.escalationsPath)
	})
}

// TestWatch_Deadman_CrossOrgActivity_DoesNotClearPendingAlert pins the
// cross-review AR-1 fix: leaderActivityEventCount must be scoped to the
// watched org. A shared manifest store (the watch's normal shape: one
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

	if err := run.evaluateCycle("org-a", status); err != nil {
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
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2 (org-b activity must not clear org-a's pending alert): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected org-a's pending alert to remain after unrelated org-b activity, got %+v", status.PendingAlerts)
	}
	assertNoEscalations(t, escalationsPath)

	// A leader-authored event recorded IN org-a itself DOES clear it.
	if err := o.Manifest.Append(ManifestEvent{TS: clk.Now().UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: LeaderIdentity, Event: EventSent, Details: "leader activity"}); err != nil {
		t.Fatalf("append org-a leader-activity event: %v", err)
	}
	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 3 (org-a's own leader activity clears the pending alert): %v", err)
	}
	assertNoEscalations(t, escalationsPath)
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected the pending alert to clear after org-a's own leader activity, got %+v", status.PendingAlerts)
	}
}

// TestWatch_Deadman_SeatSentEvent_ClearsPendingAlert_LegacyWatchdogStopDoesNot
// pins the self-review cycle-3 H3-1 fix and replaces the earlier (inverted)
// TestWatch_Deadman_UnrelatedSeatEvent_DoesNotClearPendingAlert, which
// claimed a `sent` event whose SeatID names a different seat must NOT count
// as leader activity. That claim was backwards: ev.SeatID on a `sent` event is
// the *recipient* (Send writes SeatID: p.To -- verbs.go), not the author, and
// `ralph org send` -- the only verb that appends a `sent` event -- is only
// ever driven by leader/the operator (star topology,
// .claude/rules/ralph/agent-messaging.md: a seat only ever addresses TO: leader, and
// its replies travel over the agmsg skill, never through this manifest). So
// a `sent` event naming seat-2 as SeatID IS leader activity (leader sending to
// seat-2), and must clear seat-1's pending alert. The genuinely non-clearing
// case is a legacy (pre-#152) watchdog cutoff `stopped` event
// (reason=watchdog_..., already pinned by
// TestWatch_Deadman_LegacyWatchdogStopEvent_DoesNotClearPendingAlert above)
// -- this test reuses that same non-clearing shape for a THIRD seat before
// showing the `sent` event's positive, clearing case.
func TestWatch_Deadman_SeatSentEvent_ClearsPendingAlert_LegacyWatchdogStopDoesNot(t *testing.T) {
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

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises seat-1's liveness ALERT): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected exactly 1 pending alert after cycle 1, got %+v", status.PendingAlerts)
	}

	// A legacy (pre-#152) watchdog cutoff event for seat-3, appended
	// directly since no current code path produces this shape -- carrying
	// "reason=watchdog_..." -- must NOT count as leader activity.
	if err := o.Manifest.Append(ManifestEvent{TS: clk.Now().UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: "seat-3", Event: EventStopped, Role: "worker", Details: "pane=ok leave=ok reason=watchdog_cutoff observed=31m0s"}); err != nil {
		t.Fatalf("append legacy watchdog stopped event for seat-3: %v", err)
	}
	clk.Advance(3 * time.Minute) // still within DeadmanMinutes
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2 (the legacy watchdog cutoff of seat-3 must not clear seat-1's alert): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected seat-1's pending alert to remain after the legacy watchdog cutoff of seat-3, got %+v", status.PendingAlerts)
	}
	assertNoEscalations(t, escalationsPath)

	// A `sent` event naming seat-2 as SeatID (the recipient) -- e.g.
	// `ralph org send --to seat-2`, which only leader/the operator can run --
	// IS leader activity by construction and DOES clear seat-1's pending alert.
	if err := o.Manifest.Append(ManifestEvent{TS: clk.Now().UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: "seat-2", Event: EventSent, Details: "leader sends to seat-2"}); err != nil {
		t.Fatalf("append sent event: %v", err)
	}
	clk.Advance(3 * time.Minute) // now past DeadmanMinutes overall
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 3 (sent event clears seat-1's pending alert): %v", err)
	}
	assertNoEscalations(t, escalationsPath)
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected seat-1's pending alert to clear after a sent event naming seat-2, got %+v", status.PendingAlerts)
	}
}

// TestWatch_Deadman_LeaderSpawnedEvent_ClearsPendingAlert pins that a
// `spawned` event -- part of leaderActivityEventCount's leader-driven lifecycle
// set (see its doc comment, case (b)) -- counts as leader activity even when
// it names leader itself as SeatID, the concrete case a session-promoted-leader
// org produces when leader self-registers via `ralph org spawn --id leader`.
func TestWatch_Deadman_LeaderSpawnedEvent_ClearsPendingAlert(t *testing.T) {
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

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises ALERT): %v", err)
	}

	// Leader spawns itself between cycles.
	if r := o.Spawn(watchSpawnParams("org-a", LeaderIdentity, LeaderIdentity)); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn leader failed: %+v", r)
	}

	clk.Advance(6 * time.Minute) // past DeadmanMinutes, but leader spawned in between
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	assertNoEscalations(t, escalationsPath)
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected the pending alert to clear after leader's own spawned event, got %+v", status.PendingAlerts)
	}
}

// TestWatch_Deadman_LeaderSpawnsReplacementSeat_ClearsPendingAlert pins the
// self-review cycle-3 M3-1 fix: a `spawned` event for a DIFFERENT seat (not
// leader itself) still counts as leader activity -- the concrete scenario is
// leader responding to a stall ALERT by spawning a replacement seat, which
// only leader/the operator can do via `ralph org spawn`. Before this fix,
// leaderActivityEventCount only special-cased `stopped`/`disbanded` for
// non-leader-named events, so this spawn would have counted as nothing and
// the deadman would escalate against a demonstrably responsive leader.
func TestWatch_Deadman_LeaderSpawnsReplacementSeat_ClearsPendingAlert(t *testing.T) {
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

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises seat-1's ALERT): %v", err)
	}

	// Leader spawns a REPLACEMENT seat (seat-2), not itself, between cycles.
	if r := o.Spawn(watchSpawnParams("org-a", "seat-2", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-2 failed: %+v", r)
	}

	clk.Advance(6 * time.Minute) // past DeadmanMinutes, but leader spawned seat-2 in between
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	assertNoEscalations(t, escalationsPath)
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected the pending alert to clear after leader spawns a replacement seat, got %+v", status.PendingAlerts)
	}
}

// TestWatch_Deadman_ManualStopOfOtherSeat_ClearsPendingAlert pins that a
// manual (operator-issued, non-watchdog) `stopped` event for a DIFFERENT
// seat still counts as leader activity: someone had to run `ralph org stop`
// for it to exist, and issuing that command is itself a leader-driven action
// -- unlike a legacy (pre-#152) watchdog cutoff `stopped` event
// (reason=watchdog_..., excluded by
// TestWatch_Deadman_LegacyWatchdogStopEvent_DoesNotClearPendingAlert above),
// which is self-inflicted and proves nothing about leader responsiveness.
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

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises seat-1's ALERT): %v", err)
	}

	// A human operator manually stops seat-2. A manual `ralph org stop`
	// writes a `stopped` event with no `reason=` suffix at all, unlike the
	// legacy watchdog cutoff events.
	if r := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-2"}); r.Err != nil {
		t.Fatalf("stop seat-2: %v", r.Err)
	}

	clk.Advance(6 * time.Minute)
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	assertNoEscalations(t, escalationsPath)
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected the pending alert to clear after a manual stop of another seat, got %+v", status.PendingAlerts)
	}
}

// TestWatch_Deadman_LeaderStopFailedEvent_ClearsPendingAlert pins that a
// `stop_failed` event counts as leader activity (leaderActivityEventCount's
// lifecycle set, case (b) of its doc comment). Since PR #208 a stop that
// could not close the seat's pane writes `stop_failed` instead of
// `stopped`, so a leader that answers an ALERT by stopping the stuck seat
// while herdr does not answer leaves only `stop_failed` behind. The event
// comes from a real Stop: PaneGetErr makes the ownership check fail.
func TestWatch_Deadman_LeaderStopFailedEvent_ClearsPendingAlert(t *testing.T) {
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

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises seat-1's ALERT): %v", err)
	}

	// herdr stops answering, and leader tries to stop seat-1.
	h.PaneGetErr = errors.New("herdr: no response")
	if r := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"}); r.Err == nil {
		t.Fatalf("stop seat-1: expected an error while herdr does not answer, got %+v", r)
	}
	if failed := eventsNamed(mustReadEvents(t, o), EventStopFailed); len(failed) != 1 || failed[0].OrgID != "org-a" || failed[0].SeatID != "seat-1" {
		t.Fatalf("setup: expected exactly one stop_failed for org-a/seat-1, got %+v", failed)
	}

	clk.Advance(6 * time.Minute) // past DeadmanMinutes, but leader tried to stop seat-1 in between
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	assertNoEscalations(t, escalationsPath)
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected the pending alert to clear after leader's stop_failed, got %+v", status.PendingAlerts)
	}
}

// TestWatch_Deadman_CrossOrgStopFailedEvent_DoesNotClearPendingAlert is the
// cross-org counterpart of the test above (the orgID filter of
// leaderActivityEventCount, cross-review AR-1): a `stop_failed` written in
// another org of the same manifest is not activity of the watched org, so
// its pending alert still escalates at the deadline.
func TestWatch_Deadman_CrossOrgStopFailedEvent_DoesNotClearPendingAlert(t *testing.T) {
	o, h, _, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 5
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn org-a/seat-1 failed: %+v", r)
	}
	if r := o.Spawn(watchSpawnParams("org-b", "seat-9", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn org-b/seat-9 failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")} // sticky liveness ALERT for org-a/seat-1

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises org-a's liveness ALERT): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected exactly 1 pending alert after cycle 1, got %+v", status.PendingAlerts)
	}

	// herdr stops answering, and org-b's leader tries to stop seat-9.
	h.PaneGetErr = errors.New("herdr: no response")
	if r := o.Stop(StopParams{OrgID: "org-b", Seat: "seat-9"}); r.Err == nil {
		t.Fatalf("stop org-b/seat-9: expected an error while herdr does not answer, got %+v", r)
	}
	if failed := eventsNamed(mustReadEvents(t, o), EventStopFailed); len(failed) != 1 || failed[0].OrgID != "org-b" {
		t.Fatalf("setup: expected exactly one stop_failed, in org-b, got %+v", failed)
	}

	clk.Advance(6 * time.Minute) // past DeadmanMinutes, no activity in org-a
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2 (deadman sweep): %v", err)
	}
	lines := readJSONLFile(t, escalationsPath)
	if len(lines) != 1 {
		t.Fatalf("expected org-a's pending alert to still escalate (org-b's stop_failed is not org-a's activity), got %d escalation(s): %v", len(lines), lines)
	}
}

// TestWatch_Deadman_StopFailedBeforeAlert_DoesNotClearPendingAlert pins the
// plan's edge case for `stop_failed` (Test plan of
// docs/plans/active/2026-10-08-org-watch-stop-failed.md): a `stop_failed`
// written before the ALERT is part of the baseline sendAlert records, so
// the recount in checkDeadman does not read it as new activity and the
// alert still escalates at the deadline. The two tests above only write
// `stop_failed` after the ALERT, so a baseline that left `stop_failed` out
// while the recount counted it would pass them.
func TestWatch_Deadman_StopFailedBeforeAlert_DoesNotClearPendingAlert(t *testing.T) {
	o, h, _, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 5
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-1 failed: %+v", r)
	}
	if r := o.Spawn(watchSpawnParams("org-a", "seat-2", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-2 failed: %+v", r)
	}

	// herdr does not answer while leader tries to stop seat-2, before any
	// ALERT exists.
	h.PaneGetErr = errors.New("herdr: no response")
	if r := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-2"}); r.Err == nil {
		t.Fatalf("stop seat-2: expected an error while herdr does not answer, got %+v", r)
	}
	h.PaneGetErr = nil
	if failed := eventsNamed(mustReadEvents(t, o), EventStopFailed); len(failed) != 1 || failed[0].OrgID != "org-a" || failed[0].SeatID != "seat-2" {
		t.Fatalf("setup: expected exactly one stop_failed for org-a/seat-2, got %+v", failed)
	}

	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")} // sticky liveness ALERT for seat-1

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises seat-1's ALERT): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected exactly 1 pending alert after cycle 1 (the earlier stop_failed is in the baseline, so it must not clear the alert), got %+v", status.PendingAlerts)
	}

	clk.Advance(6 * time.Minute) // past DeadmanMinutes, no activity since the ALERT
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2 (deadman sweep): %v", err)
	}
	lines := readJSONLFile(t, escalationsPath)
	if len(lines) != 1 {
		t.Fatalf("expected the pending alert to escalate (the stop_failed predates the ALERT, so it is in the baseline), got %d escalation(s): %v", len(lines), lines)
	}
}

// TestLeaderActivityEventCount_StopFailed pins how leaderActivityEventCount
// counts `stop_failed` on its own: one for the watched org, none for a
// record whose Details carry "reason=watchdog_" (the exclusion that covers
// the whole lifecycle set), and none for another org's record.
func TestLeaderActivityEventCount_StopFailed(t *testing.T) {
	stopFailed := func(orgID, details string) ManifestEvent {
		return ManifestEvent{TS: "2026-01-01T00:00:00Z", OrgID: orgID, SeatID: "seat-1", Event: EventStopFailed, Role: "worker", Details: details}
	}
	tests := []struct {
		name string
		ev   ManifestEvent
		want int
	}{
		{"stop_failed in the watched org counts", stopFailed("org-a", "ctrl_c=skipped: pane not confirmed pane=not closed: check pane \"pane-1\": herdr: no response leave=skipped: pane not closed"), 1},
		{"stop_failed carrying reason=watchdog_ does not count", stopFailed("org-a", "pane=not closed leave=skipped: pane not closed reason=watchdog_cutoff"), 0},
		{"stop_failed in another org does not count", stopFailed("org-b", "pane=not closed leave=skipped: pane not closed"), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := leaderActivityEventCount([]ManifestEvent{tt.ev}, "org-a"); got != tt.want {
				t.Errorf("leaderActivityEventCount = %d, want %d (event %+v)", got, tt.want, tt.ev)
			}
		})
	}
}

// TestWatch_Stall_UsesLatestEventOfAnyType_NotOnlyStateEvents pins the
// self-review M-6 fix: the stall condition's time term must track the
// seat's latest manifest event of ANY type, not Roster's SeatStatus.TS
// (which only advances on *state* events and so stays pinned at a healthy
// active seat's own `spawned` TS forever).
func TestWatch_Stall_UsesLatestEventOfAnyType_NotOnlyStateEvents(t *testing.T) {
	o, h, a, clk := testWatchOrg(t)
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
	if err := run.evaluateCycle("org-a", status); err != nil {
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

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	if n := len(watchdogAlerts(a)); n != 0 {
		t.Fatalf("expected no stall ALERT: the seat's own latest event (any type) is recent, got %d: %+v", n, watchdogAlerts(a))
	}
}

// --- cross-review-triage cycle-2 #3: deadman history source is leader-only ---

// TestFilterLeaderHistoryLines pins leaderHistoryFromField/filterLeaderHistoryLines'
// parsing contract directly: only lines whose "from" field (the segment
// between "] " and " → ") is exactly LeaderIdentity survive; anything that
// does not parse into that shape is excluded, not conservatively kept.
func TestFilterLeaderHistoryLines(t *testing.T) {
	raw := strings.Join([]string{
		"  ● [2026-01-01T00:00:00Z] leader → seat-1: kickoff task",
		"  ○ [2026-01-01T00:01:00Z] seat-1 → leader: ack",
		"  ● [2026-01-01T00:02:00Z] watchdog → leader: TYPE: ALERT",
		"garbled line with no timestamp or arrow at all",
		"  ● [2026-01-01T00:03:00Z] leader with no arrow marker here",
		"  ● [2026-01-01T00:04:00Z] leader → seat-2: another leader message",
	}, "\n")

	got := filterLeaderHistoryLines(raw)
	want := strings.Join([]string{
		"  ● [2026-01-01T00:00:00Z] leader → seat-1: kickoff task",
		"  ● [2026-01-01T00:04:00Z] leader → seat-2: another leader message",
	}, "\n")
	if got != want {
		t.Errorf("filterLeaderHistoryLines:\n got:  %q\n want: %q", got, want)
	}
}

// TestWatch_Deadman_WatchdogAlertHistoryLine_DoesNotClearPendingAlert pins
// cross-review-triage cycle-2 #3: a new agmsg history line produced by the
// watchdog's OWN alert traffic (watchdog -> leader) must not count as leader
// activity and clear a pending deadman alert -- only leader-authored lines may.
func TestWatch_Deadman_WatchdogAlertHistoryLine_DoesNotClearPendingAlert(t *testing.T) {
	o, h, a, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 100 // keep the timeout out of the picture
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")} // sticky liveness ALERT

	team := agmsgTeam("org-a")
	leaderLine := "  ● [2026-01-01T00:00:00Z] leader → seat-1: kickoff task"
	base := leaderLine
	withWatchdogAlert := leaderLine + "\n  ● [2026-01-01T00:05:00Z] watchdog → leader: TYPE: ALERT\nCONDITION: liveness"
	// call 1: sendAlert's own historyLeaderLineCount capture (cycle 1); call 2:
	// checkDeadman's comparison in that same cycle 1; call 3: checkDeadman's
	// comparison in cycle 2, where a new non-leader line has been appended.
	a.HistorySeq[team] = []string{base, base, withWatchdogAlert}

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected 1 pending alert after cycle 1, got %d: %+v", len(status.PendingAlerts), status.PendingAlerts)
	}

	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Errorf("expected the pending alert to remain: a new watchdog->leader history line must not count as leader activity, got %d pending: %+v",
			len(status.PendingAlerts), status.PendingAlerts)
	}
	assertNoEscalations(t, escalationsPath)
}

// TestWatch_Deadman_LeaderHistoryLine_ClearsPendingAlert is the positive
// counterpart: a new agmsg history line genuinely authored BY leader (leader ->
// seat) must still clear a pending deadman alert, same as before this fix.
func TestWatch_Deadman_LeaderHistoryLine_ClearsPendingAlert(t *testing.T) {
	o, h, a, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 100 // keep the timeout out of the picture
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	target := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[target] = []error{errors.New("herdr: agent not found")} // sticky liveness ALERT

	team := agmsgTeam("org-a")
	// A malformed line is deliberately mixed in alongside the genuine leader
	// line -- it must not itself be mistaken for leader activity (it is simply
	// excluded), the real leader -> seat line is what clears the alert.
	withLeaderLine := "garbled line with no timestamp or arrow\n" +
		"  ● [2026-01-01T00:05:00Z] leader → seat-1: are you there?"
	a.HistorySeq[team] = []string{"", "", withLeaderLine}

	run, statusPath, _ := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected 1 pending alert after cycle 1, got %d: %+v", len(status.PendingAlerts), status.PendingAlerts)
	}

	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected the pending alert to clear: a new leader->seat history line is genuine leader activity, got %d pending: %+v",
			len(status.PendingAlerts), status.PendingAlerts)
	}
}

// TestWatch_Deadman_HistoryWindowEviction_DoesNotFalselyClearPendingAlert
// pins the self-review cycle-3 M3-2 fix: historyLeaderLineCount's window is
// the last-20-of-EVERYONE window (see its doc comment), so as other seats
// chat, older leader lines fall out of that window and the visible leader-line
// set can shrink with NO new leader activity at all. The pre-fix exact-string
// comparison (`cur != "" && cur != pending.History`) treated any change --
// including a pure shrinkage -- as activity and wrongly cleared the pending
// alert; the count-based comparison (`cur > pending.HistoryLeaderLines`) does
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
	threeLeaderLines := strings.Join([]string{
		"  ● [2026-01-01T00:00:00Z] leader → seat-1: msg 1",
		"  ● [2026-01-01T00:01:00Z] leader → seat-1: msg 2",
		"  ● [2026-01-01T00:02:00Z] leader → seat-1: msg 3",
	}, "\n")
	// Simulates the underlying last-20-of-everyone window evicting the two
	// older leader lines as other seats chat -- fewer leader lines are visible,
	// but none of them is new.
	shrunkWindow := "  ● [2026-01-01T00:02:00Z] leader → seat-1: msg 3"
	// call 1: sendAlert's own historyLeaderLineCount capture (cycle 1,
	// count=3); call 2: checkDeadman's comparison in that same cycle 1
	// (count=3, no growth); call 3: checkDeadman's comparison in cycle 2,
	// where eviction has shrunk the visible window to 1 leader line.
	a.HistorySeq[team] = []string{threeLeaderLines, threeLeaderLines, shrunkWindow}

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1: %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected 1 pending alert after cycle 1, got %d: %+v", len(status.PendingAlerts), status.PendingAlerts)
	}

	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2: %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Errorf("expected the pending alert to remain: a shrunk (evicted) history window with no new leader line must not count as activity, got %d pending: %+v",
			len(status.PendingAlerts), status.PendingAlerts)
	}
	assertNoEscalations(t, escalationsPath)
}

// --- PR④ known gap #5: deadman probe-recovery false-clear -------------------

// TestWatch_Deadman_ProbeOutageRecoveryAlone_DoesNotClearPendingAlert pins the
// checkDeadman fix for cross-review-triage cycle-3 #5: an ALERT recorded
// while the leader herdr probe (leaderProbeSnapshot) was unavailable persists
// LeaderAgentGet == "" as its baseline. A later cycle where the probe merely
// recovers -- with no other genuine leader activity -- must NOT by itself
// clear the pending alert: pre-fix, `cur != "" && cur != pending.LeaderAgentGet`
// is trivially satisfied by any recovered value compared against a ""
// baseline, treating "the probe came back" as if it were "leader did
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

	// The leader's own probe (checkDeadman's source #2) is down for exactly the
	// 2 calls made during cycle 1 (sendAlert's baseline capture, then
	// checkDeadman's own same-cycle comparison), then recovers from cycle 2
	// onward.
	leaderTarget := herdrAgentName("org-a", LeaderIdentity)
	h.AgentGetErrSeq[leaderTarget] = []error{
		errors.New("herdr: unreachable"),
		errors.New("herdr: unreachable"),
		nil, // recovered: falls through to the default "ok" AgentGet response
	}

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises ALERT, probe baseline unavailable): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected 1 pending alert after cycle 1, got %d: %+v", len(status.PendingAlerts), status.PendingAlerts)
	}
	for _, pending := range status.PendingAlerts {
		if pending.LeaderAgentGet != "" {
			t.Fatalf("expected the pending alert's probe baseline to be the unavailable sentinel (\"\"), got %q", pending.LeaderAgentGet)
		}
	}

	clk.Advance(1 * time.Minute) // well under DeadmanMinutes
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2 (probe recovers, no other activity): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Errorf("expected the pending alert to remain: probe recovery alone (unavailable baseline) must not count as leader activity, got %d pending: %+v",
			len(status.PendingAlerts), status.PendingAlerts)
	}
	assertNoEscalations(t, escalationsPath)
}

// TestWatch_Deadman_ProbeOutageThenGenuineManifestActivity_ClearsPendingAlert
// is the positive counterpart: once a pending alert's probe baseline is the
// unavailable sentinel, that source can never independently clear it (its
// baseline is frozen at ALERT time), but genuine leader activity via a
// different, unaffected source -- a new leader-attributable manifest event --
// must still clear it, exactly as before this fix.
func TestWatch_Deadman_ProbeOutageThenGenuineManifestActivity_ClearsPendingAlert(t *testing.T) {
	o, h, _, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 100 // keep the timeout out of the picture
	if r := o.Spawn(watchSpawnParams("org-a", "seat-1", "worker")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	seatTarget := herdrAgentName("org-a", "seat-1")
	h.AgentGetErrSeq[seatTarget] = []error{errors.New("herdr: agent not found")}

	leaderTarget := herdrAgentName("org-a", LeaderIdentity)
	h.AgentGetErrSeq[leaderTarget] = []error{
		errors.New("herdr: unreachable"),
		errors.New("herdr: unreachable"),
		nil, // recovered from cycle 2 onward, same as the negative test above
	}

	run, statusPath, escalationsPath := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises ALERT, probe baseline unavailable): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected 1 pending alert after cycle 1, got %d: %+v", len(status.PendingAlerts), status.PendingAlerts)
	}

	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2 (probe recovers, no other activity): %v", err)
	}
	if len(status.PendingAlerts) != 1 {
		t.Fatalf("expected the pending alert to remain after probe recovery alone, got %d pending: %+v", len(status.PendingAlerts), status.PendingAlerts)
	}

	// Genuine leader activity: a real `sent` event, leader-authored by
	// construction (see leaderActivityEventCount's doc comment).
	if err := o.Manifest.Append(ManifestEvent{TS: clk.Now().UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: "seat-1", Event: EventSent, Details: "leader activity"}); err != nil {
		t.Fatalf("append leader-activity event: %v", err)
	}

	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 3 (genuine manifest activity): %v", err)
	}
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected the pending alert to clear: a genuine manifest sent event is leader activity via an unaffected source, got %d pending: %+v",
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
	// Spawn's own leader/seat Join calls (made before evaluateCycle even runs)
	// do not consume these entries.
	a.JoinErrSeq[watchdogIdentity] = []error{errors.New("agmsg: team unreachable"), nil}

	run, statusPath, _ := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}

	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 1 (raises ALERT, Join fails): %v", err)
	}
	if status.WatchdogJoined {
		t.Fatalf("expected WatchdogJoined to remain false after a failed Join")
	}
	if calls := watchdogJoinCalls(a); len(calls) != 1 {
		t.Fatalf("expected exactly 1 watchdog Join attempt after cycle 1, got %d: %+v", len(calls), calls)
	}

	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("cycle 2 (recovers, no ALERT): %v", err)
	}
	if status.WatchdogJoined {
		t.Fatalf("expected WatchdogJoined to still be false: no ALERT was raised in cycle 2, so Join is not retried")
	}
	if calls := watchdogJoinCalls(a); len(calls) != 1 {
		t.Fatalf("expected still exactly 1 watchdog Join attempt after cycle 2 (no ALERT raised), got %d: %+v", len(calls), calls)
	}

	clk.Advance(1 * time.Minute)
	if err := run.evaluateCycle("org-a", status); err != nil {
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

// --- AC-5: status.Escalated bookkeeping (tech-debt: "watchdog deferred LOW
// (3)": unreachable checkDeadman guard removed, pruning added) -------------

// TestPruneEscalated_CapsAtMaxEntries_DropsOldestByEmbeddedTimestamp is a
// direct unit test of pruneEscalated's deterministic-survivor contract: 101
// entries whose alertIDs embed strictly increasing nanosecond timestamps
// (0..100, sendAlert's "<conditionKey>@<unixnano>" format) prune down to
// exactly maxEscalatedEntries (100), dropping the single oldest (timestamp
// 0) and keeping every other entry -- not an arbitrary/random map-iteration
// eviction.
func TestPruneEscalated_CapsAtMaxEntries_DropsOldestByEmbeddedTimestamp(t *testing.T) {
	escalated := map[string]bool{}
	for i := 0; i <= maxEscalatedEntries; i++ { // 101 entries: timestamps 0..100
		escalated[fmt.Sprintf("org-a//deadman@%d", i)] = true
	}
	if len(escalated) != maxEscalatedEntries+1 {
		t.Fatalf("setup: expected %d entries, got %d", maxEscalatedEntries+1, len(escalated))
	}

	pruneEscalated(escalated)

	if len(escalated) != maxEscalatedEntries {
		t.Fatalf("expected pruneEscalated to cap at %d entries, got %d", maxEscalatedEntries, len(escalated))
	}
	if escalated["org-a//deadman@0"] {
		t.Errorf("expected the oldest entry (embedded timestamp 0) to be pruned, but it survived")
	}
	for i := 1; i <= maxEscalatedEntries; i++ {
		id := fmt.Sprintf("org-a//deadman@%d", i)
		if !escalated[id] {
			t.Errorf("expected entry %q to survive pruning (deterministic survivor set), but it was dropped", id)
		}
	}
}

// TestPruneEscalated_AtExactlyMaxEntries_NoPruning confirms the cap is
// "prune once *over* maxEscalatedEntries", not "keep strictly fewer than"
// -- exactly maxEscalatedEntries entries must be left untouched.
func TestPruneEscalated_AtExactlyMaxEntries_NoPruning(t *testing.T) {
	escalated := map[string]bool{}
	for i := range maxEscalatedEntries {
		escalated[fmt.Sprintf("org-a//deadman@%d", i)] = true
	}
	pruneEscalated(escalated)
	if len(escalated) != maxEscalatedEntries {
		t.Errorf("expected no pruning at exactly maxEscalatedEntries entries, got %d", len(escalated))
	}
}

// TestWatch_EscalateAlert_PrunesAt101stEscalation is the integration-level
// regression: escalateAlert itself (not just the pruneEscalated helper in
// isolation) must keep status.Escalated bounded as a long-running `ralph org
// watch` process accumulates escalations across many cycles, and the
// oldest-first eviction must hold end to end.
func TestWatch_EscalateAlert_PrunesAt101stEscalation(t *testing.T) {
	o, _, _, clk := testWatchOrg(t)
	run, statusPath, _ := newTestWatchRun(o, nil, nil, &bytes.Buffer{})
	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}
	status.Escalated = map[string]bool{}

	for i := 0; i <= maxEscalatedEntries; i++ { // 101 escalations
		alertID := fmt.Sprintf("org-a/seat-x/deadman@%d", i)
		pending := &watchPendingAlert{AlertID: alertID, Subject: "seat-x", TS: clk.Now().UTC().Format(time.RFC3339)}
		run.escalateAlert(status, alertID, pending, clk.Now())
	}

	if len(status.Escalated) != maxEscalatedEntries {
		t.Fatalf("expected status.Escalated capped at %d entries after %d escalations, got %d",
			maxEscalatedEntries, maxEscalatedEntries+1, len(status.Escalated))
	}
	if status.Escalated["org-a/seat-x/deadman@0"] {
		t.Errorf("expected the oldest escalation (timestamp 0) to be pruned from status.Escalated")
	}
	newest := fmt.Sprintf("org-a/seat-x/deadman@%d", maxEscalatedEntries)
	if !status.Escalated[newest] {
		t.Errorf("expected the newest escalation %q to survive pruning", newest)
	}
}

// --- org budget removal: retired condition pruning --------------------------

// TestWatch_PrunesRetiredBudgetEntriesFromStatus_NoEscalation pins the
// pruneRetiredConditions fix (AC-7b): a watch-status file persisted before
// the org budget concept was removed can still carry seat_budget/
// total_budget entries in Conditions, PendingAlerts, and Escalated. Without
// pruning, a stale seat_budget PendingAlerts entry old enough to exceed
// DeadmanMinutes would escalate on the very first cycle after upgrading --
// a false-positive deadman escalation for a condition type that no longer
// exists. This test hand-writes exactly such a legacy fixture (including
// the retired "cutoff" field the current watchConditionRecord no longer
// declares, which json.Unmarshal silently ignores), then asserts one cycle
// prunes every retired-condition entry and produces no escalation and no
// ALERT.
func TestWatch_PrunesRetiredBudgetEntriesFromStatus_NoEscalation(t *testing.T) {
	o, _, a, clk := testWatchOrg(t)
	o.Config.DeadmanMinutes = 5

	var escalateCalls int
	escalateFn := func(context.Context, string) error {
		escalateCalls++
		return nil
	}
	run, statusPath, escalationsPath := newTestWatchRun(o, nil, escalateFn, &bytes.Buffer{})

	// A legacy fixture: seat_budget/total_budget entries in all three
	// dedupe/deadman maps, alongside a live scope_change entry that must
	// survive pruning untouched. The pending seat_budget alert's ts is well
	// over an hour before the fakeClock's start (2026-01-01T00:00:00Z, see
	// newFakeClock), so -- absent pruning -- checkDeadman's
	// now.Sub(ts) > DeadmanMinutes check would escalate it on cycle 1.
	fixture := `{
  "org_id": "org-a",
  "last_cycle_ts": "2025-12-31T23:00:00Z",
  "cycles": 3,
  "conditions": {
    "org-a/seat-z/seat_budget": {"active": true, "cutoff": true, "first_ts": "2025-12-31T22:00:00Z"},
    "org-a//total_budget": {"active": true, "cutoff": false, "first_ts": "2025-12-31T22:00:00Z"},
    "org-a/seat-w/scope_change": {"active": true, "first_ts": "2025-12-31T22:30:00Z"}
  },
  "pending_alerts": {
    "org-a/seat-z/seat_budget@1000000000": {
      "alert_id": "org-a/seat-z/seat_budget@1000000000",
      "ts": "2025-12-31T22:00:00Z",
      "subject": "seat-z",
      "manifest_len": 0,
      "lead_agent_get": "",
      "history_lead_lines": -1
    }
  },
  "escalated": {
    "org-a//total_budget@2000000000": true
  }
}`
	if err := os.WriteFile(statusPath, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write legacy status fixture: %v", err)
	}

	status, err := loadWatchStatus(statusPath, "org-a")
	if err != nil {
		t.Fatalf("loadWatchStatus: %v", err)
	}
	if len(status.Conditions) != 3 || len(status.PendingAlerts) != 1 || len(status.Escalated) != 1 {
		t.Fatalf("setup: expected the fixture to load unpruned, got conditions=%+v pending=%+v escalated=%+v",
			status.Conditions, status.PendingAlerts, status.Escalated)
	}

	clk.Advance(time.Minute) // now well past the pending alert's ts + DeadmanMinutes
	if err := run.evaluateCycle("org-a", status); err != nil {
		t.Fatalf("evaluateCycle: %v", err)
	}

	for key := range status.Conditions {
		if strings.HasSuffix(key, "/seat_budget") || strings.HasSuffix(key, "/total_budget") {
			t.Errorf("expected retired condition %q to be pruned, but it survived: %+v", key, status.Conditions)
		}
	}
	if _, ok := status.Conditions["org-a/seat-w/scope_change"]; !ok {
		t.Errorf("expected the live scope_change condition to survive pruning, got %+v", status.Conditions)
	}
	if len(status.PendingAlerts) != 0 {
		t.Errorf("expected the retired seat_budget pending alert to be pruned before the deadman sweep, got %+v", status.PendingAlerts)
	}
	if status.Escalated["org-a//total_budget@2000000000"] {
		t.Errorf("expected the retired total_budget escalated entry to be pruned, got %+v", status.Escalated)
	}

	assertNoEscalations(t, escalationsPath)
	if n := len(watchdogAlerts(a)); n != 0 {
		t.Errorf("expected no ALERT to fire for a pruned retired condition, got %d: %+v", n, watchdogAlerts(a))
	}
	if escalateCalls != 0 {
		t.Errorf("expected the EscalateFunc hook to never fire for a pruned retired condition, got %d call(s)", escalateCalls)
	}
}

// --- helpers -----------------------------------------------------------------

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
