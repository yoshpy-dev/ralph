package org

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// fakeHerdr is a call-recording, in-memory HerdrClient used by every spawn
// saga unit test in this file -- no exec.Command, no PATH, no real herdr
// binary needed. Per-method Err fields let a test inject a failure at
// exactly one saga boundary. mu guards every field below so fakeHerdr is
// safe to share across goroutines (TestOrgSpawn_ConcurrentSpawns_MaxSeatsNeverExceeded,
// AC-9) -- every other test in this file drives fakeHerdr from a single
// goroutine, where an uncontended mutex is a no-op cost.
type fakeHerdr struct {
	mu    sync.Mutex
	calls []string

	workspaceCreateErr error
	tabCreateErr       error
	agentStartErr      error
	// agentStartErrs, when non-empty, is dequeued one entry per AgentStart
	// call (nil entries count as a successful call) and takes priority over
	// agentStartErr -- lets a test script a specific sequence of outcomes
	// (e.g. busy, busy, success) for the agentStartWithRetry retry-path
	// tests, while every other test keeps using the simpler single-error
	// agentStartErr field unmodified.
	agentStartErrs  []error
	paneSendKeysErr error
	// paneSendTextErr, when non-nil, makes PaneSendText fail (after the
	// optional paneSendTextDelay below).
	paneSendTextErr error
	// agentWaitErrs, when non-empty, is dequeued one entry per AgentWait
	// call (nil entries count as a successful call) -- lets a test script a
	// specific outcome for one AgentWait call in a sequence (e.g. the send
	// verb's idle/done wait succeeding but its post-Enter confirm wait
	// failing) without affecting any other AgentWait call. Empty queue
	// means every call succeeds, exactly like before this field existed.
	agentWaitErrs []error
	// paneSendTextDelay, when > 0, makes PaneSendText sleep that long before
	// returning -- simulating a real herdr round trip that eats into the
	// ctx budget between Send's fail-closed pre-Enter-pause budget check
	// (which measures ctx time remaining right before PaneSendText) and the
	// actual pre-Enter wait, so a test can deterministically drive ctx
	// expiry inside that wait without shaving its margin thin (see
	// TestOrgSend_CtxExpiresDuringEnterDelay_AfterBudgetCheckPasses in
	// verbs_test.go). Zero (the default) keeps PaneSendText instantaneous,
	// exactly as before this field existed.
	paneSendTextDelay time.Duration
	// paneCloseErrs / workspaceCloseErrs, keyed by pane / workspace id, make
	// PaneClose / WorkspaceClose return that error for that id; ids without
	// an entry (and a nil map) succeed. For an "already closed" reply, use
	// driver.NewHerdrError(driver.HerdrCodePaneNotFound, ...) (or
	// HerdrCodeWorkspaceNotFound) so driver.IsNotFound recognises it. A test
	// can delete an entry between calls to model herdr coming back.
	paneCloseErrs      map[string]error
	workspaceCloseErrs map[string]error
	// paneSendKeysBlock / paneCloseBlock / workspaceCloseBlock, when true,
	// make PaneSendKeys / PaneClose / WorkspaceClose record the call and then
	// not return until ctx is done, returning ctx.Err() -- a herdr that never
	// answers, for the per-call timeout tests (plan AC11). The call is still
	// logged first.
	paneSendKeysBlock   bool
	paneCloseBlock      bool
	workspaceCloseBlock bool

	// herdr's view of what this fake created, which PaneGet / TabGet /
	// WorkspaceGet answer from: WorkspaceCreate labels the workspace it hands
	// out, and TabCreate puts the pane it hands out in a tab of its own
	// (fakeTabID) labelled with the label it was given (the seat id), inside
	// the workspace it was given. A later TabCreate handing out the same pane
	// id relabels that tab, so seats meant to be stopped need pane ids of
	// their own. A test edits these maps to model a recorded id that herdr
	// now uses for something else (plan AC14), or deletes an entry to model
	// one herdr no longer knows; an id with no entry is not found.
	paneTabs        map[string]string // pane id -> tab id
	paneWorkspaces  map[string]string // pane id -> workspace id
	tabLabels       map[string]string // tab id -> label
	workspaceLabels map[string]string // workspace id -> label
	// getErrs, keyed by pane, tab or workspace id, makes PaneGet / TabGet /
	// WorkspaceGet return that error for that id. getBlock makes the three
	// record the call and then not return until ctx is done (see
	// paneCloseBlock).
	getErrs  map[string]error
	getBlock bool

	workspaceID string
	paneID      string

	sendKeysCalls     []string   // paneIDs PaneSendKeys was invoked with, in order
	sendKeysKeys      [][]string // keys PaneSendKeys was invoked with, in order (e.g. asserting exactly one "Enter")
	agentStartNames   []string   // agent names AgentStart was invoked with, in order
	agentStartArgs    [][]string // agentArgs AgentStart was invoked with, in order (AC-4 argv assertions)
	agentWaitTargets  []string   // targets AgentWait was invoked with, in order
	agentWaitUntil    [][]string // until states AgentWait was invoked with, in order
	agentWaitTimeouts []int      // timeoutMS AgentWait was invoked with, in order

	paneCloseCalls      []string // paneIDs PaneClose was invoked with, in order
	workspaceCloseCalls []string // workspaceIDs WorkspaceClose was invoked with, in order
	tabCreateWorkspaces []string // workspaceIDs TabCreate was invoked with, in order
	paneGetCalls        []string // paneIDs PaneGet was invoked with, in order
	tabGetCalls         []string // tabIDs TabGet was invoked with, in order
	workspaceGetCalls   []string // workspaceIDs WorkspaceGet was invoked with, in order
}

// fakeTabID is the tab id fakeHerdr's TabCreate gives the tab holding
// paneID.
func fakeTabID(paneID string) string { return "tab-" + paneID }

// setLabel records id -> label in *m, making the map on first use.
func setLabel(m *map[string]string, id, label string) {
	if *m == nil {
		*m = make(map[string]string)
	}
	(*m)[id] = label
}

func (f *fakeHerdr) WorkspaceCreate(_ context.Context, _, label string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "workspace_create")
	if f.workspaceCreateErr != nil {
		return "", f.workspaceCreateErr
	}
	if f.workspaceID == "" {
		f.workspaceID = "ws-1"
	}
	setLabel(&f.workspaceLabels, f.workspaceID, label)
	return f.workspaceID, nil
}

func (f *fakeHerdr) TabCreate(_ context.Context, workspaceID, _, label string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "tab_create")
	f.tabCreateWorkspaces = append(f.tabCreateWorkspaces, workspaceID)
	if f.tabCreateErr != nil {
		return "", f.tabCreateErr
	}
	if f.paneID == "" {
		f.paneID = "pane-1"
	}
	setLabel(&f.paneTabs, f.paneID, fakeTabID(f.paneID))
	setLabel(&f.paneWorkspaces, f.paneID, workspaceID)
	setLabel(&f.tabLabels, fakeTabID(f.paneID), label)
	return f.paneID, nil
}

// fakeGet is the shared body of PaneGet / TabGet / WorkspaceGet: record id
// in *calls and as call name in f.calls, then block (getBlock), return the
// injected error (getErrs), or answer lookup(id), run with mu held, with
// herdr's not-found error (notFoundCode) when it reports no entry.
func (f *fakeHerdr) fakeGet(ctx context.Context, name string, calls *[]string, id string, lookup func(string) (string, bool), notFoundCode string) (string, error) {
	f.mu.Lock()
	f.calls = append(f.calls, name)
	*calls = append(*calls, id)
	block := f.getBlock
	injected := f.getErrs[id]
	value, ok := lookup(id)
	f.mu.Unlock()
	switch {
	case block:
		<-ctx.Done()
		return "", ctx.Err()
	case injected != nil:
		return "", injected
	case !ok:
		return "", driver.NewHerdrError(notFoundCode, id+" not found")
	}
	return value, nil
}

// PaneGet answers the pane's tab and workspace as TabCreate recorded them.
func (f *fakeHerdr) PaneGet(ctx context.Context, paneID string) (string, string, error) {
	var workspaceID string
	tabID, err := f.fakeGet(ctx, "pane_get", &f.paneGetCalls, paneID, func(id string) (string, bool) {
		tabID, ok := f.paneTabs[id]
		workspaceID = f.paneWorkspaces[id]
		return tabID, ok
	}, driver.HerdrCodePaneNotFound)
	if err != nil {
		return "", "", err
	}
	return tabID, workspaceID, nil
}

func (f *fakeHerdr) TabGet(ctx context.Context, tabID string) (string, error) {
	return f.fakeGet(ctx, "tab_get", &f.tabGetCalls, tabID, func(id string) (string, bool) {
		label, ok := f.tabLabels[id]
		return label, ok
	}, driver.HerdrCodeTabNotFound)
}

func (f *fakeHerdr) WorkspaceGet(ctx context.Context, workspaceID string) (string, error) {
	return f.fakeGet(ctx, "workspace_get", &f.workspaceGetCalls, workspaceID, func(id string) (string, bool) {
		label, ok := f.workspaceLabels[id]
		return label, ok
	}, driver.HerdrCodeWorkspaceNotFound)
}

func (f *fakeHerdr) AgentStart(_ context.Context, name, _, _ string, _ int, agentArgs []string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "agent_start")
	f.agentStartNames = append(f.agentStartNames, name)
	f.agentStartArgs = append(f.agentStartArgs, agentArgs)
	if len(f.agentStartErrs) > 0 {
		err := f.agentStartErrs[0]
		f.agentStartErrs = f.agentStartErrs[1:]
		if err != nil {
			return "", err
		}
		return "agent-1", nil
	}
	if f.agentStartErr != nil {
		return "", f.agentStartErr
	}
	return "agent-1", nil
}

func (f *fakeHerdr) AgentWait(_ context.Context, target string, until []string, timeoutMS int) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "agent_wait")
	f.agentWaitTargets = append(f.agentWaitTargets, target)
	f.agentWaitUntil = append(f.agentWaitUntil, until)
	f.agentWaitTimeouts = append(f.agentWaitTimeouts, timeoutMS)
	if len(f.agentWaitErrs) > 0 {
		err := f.agentWaitErrs[0]
		f.agentWaitErrs = f.agentWaitErrs[1:]
		if err != nil {
			return "", err
		}
	}
	return "idle", nil
}

func (f *fakeHerdr) PaneRead(_ context.Context, _ string, _ int) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "pane_read")
	return "pane output", nil
}

func (f *fakeHerdr) PaneSendText(_ context.Context, _, _ string) error {
	f.mu.Lock()
	f.calls = append(f.calls, "pane_send_text")
	delay := f.paneSendTextDelay
	err := f.paneSendTextErr
	f.mu.Unlock()
	if delay > 0 {
		time.Sleep(delay)
	}
	return err
}

func (f *fakeHerdr) PaneSendKeys(ctx context.Context, paneID string, keys ...string) error {
	f.mu.Lock()
	f.calls = append(f.calls, "pane_send_keys")
	f.sendKeysCalls = append(f.sendKeysCalls, paneID)
	f.sendKeysKeys = append(f.sendKeysKeys, keys)
	block := f.paneSendKeysBlock
	err := f.paneSendKeysErr
	f.mu.Unlock()
	if block {
		<-ctx.Done()
		return ctx.Err()
	}
	return err
}

func (f *fakeHerdr) PaneClose(ctx context.Context, paneID string) error {
	f.mu.Lock()
	f.calls = append(f.calls, "pane_close")
	f.paneCloseCalls = append(f.paneCloseCalls, paneID)
	block := f.paneCloseBlock
	err := f.paneCloseErrs[paneID]
	f.mu.Unlock()
	if block {
		<-ctx.Done()
		return ctx.Err()
	}
	return err
}

func (f *fakeHerdr) WorkspaceClose(ctx context.Context, workspaceID string) error {
	f.mu.Lock()
	f.calls = append(f.calls, "workspace_close")
	f.workspaceCloseCalls = append(f.workspaceCloseCalls, workspaceID)
	block := f.workspaceCloseBlock
	err := f.workspaceCloseErrs[workspaceID]
	f.mu.Unlock()
	if block {
		<-ctx.Done()
		return ctx.Err()
	}
	return err
}

// fakeAgmsg is a call-recording, in-memory AgmsgClient. joinErrs, keyed by
// agentID (e.g. "leader" or a seat id), lets a test inject a Join failure at
// exactly one identity while leaving the other Join call (leader vs seat)
// unaffected -- needed to test ensureLeaderJoined's best-effort semantics
// independently of the seat Join's hard-failure gate. mu guards every field
// below -- see fakeHerdr's doc comment for why.
type fakeAgmsg struct {
	mu      sync.Mutex
	calls   []string
	sendErr error

	joinErrs   map[string]error
	joinCalls  []joinCall
	leaveErr   error
	leaveCalls []leaveCall
	// leaveBlock, when true, makes Leave record the call and then not
	// return until ctx is done, returning ctx.Err() -- an agmsg that never
	// answers (see fakeHerdr's paneCloseBlock).
	leaveBlock bool
	// leaveHook, when non-nil, runs inside Leave (after the call is
	// recorded, without mu held), so a test can snapshot what had already
	// happened -- herdr calls, manifest events -- at the moment of the Leave.
	leaveHook func()
}

type joinCall struct {
	team, agentID, agmsgType, projectPath string
}

type leaveCall struct {
	team, agentID string
}

func (f *fakeAgmsg) Send(_ context.Context, _, _, _, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "send")
	return f.sendErr
}

func (f *fakeAgmsg) Join(_ context.Context, team, agentID, agmsgType, projectPath string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "join:"+agentID)
	f.joinCalls = append(f.joinCalls, joinCall{team: team, agentID: agentID, agmsgType: agmsgType, projectPath: projectPath})
	if f.joinErrs != nil {
		if err, ok := f.joinErrs[agentID]; ok {
			return err
		}
	}
	return nil
}

func (f *fakeAgmsg) Leave(ctx context.Context, team, agentID string) error {
	f.mu.Lock()
	f.calls = append(f.calls, "leave")
	f.leaveCalls = append(f.leaveCalls, leaveCall{team: team, agentID: agentID})
	block := f.leaveBlock
	err := f.leaveErr
	hook := f.leaveHook
	f.mu.Unlock()
	if hook != nil {
		hook()
	}
	if block {
		<-ctx.Done()
		return ctx.Err()
	}
	return err
}

// testCodexObserveGenerousBudget is the scan budget a test sets (in place
// of testOrg's tiny default below) whenever Spawn's poll or Stop's single
// check must actually FIND a real, matching session record: a found or
// ambiguous result returns at once, so this budget is only an upper
// bound, never something the test waits out -- it costs nothing on the
// normal (fast) path and only matters as headroom against ReadDir/Lstat/
// open/line-read latency under CI-level scheduling contention (the ctx
// checked inside a single scan pass, not merely between polls). A test
// that instead wants the budget itself to run out (not-found, cut-short,
// budget-exhausted-mid-pass) keeps testOrg's own tiny default or sets its
// own small, explicit value -- never this constant.
//
// A third category needs neither: a test whose terminal state depends on
// at least one pass COMPLETING (reaching os.Open and recording a read
// error, say) before the budget expires, but that budget must still run
// out quickly because nothing it can find will ever satisfy it. testOrg's
// 1ms default risks cutting that first pass short (degrading the result
// to not-found, since a cut-short pass never sets lastErr); this
// constant's 30s would make the test wait out the full 30s for nothing.
// Such a test sets its own modest, explicit budget instead (e.g. 50ms) --
// enough margin for one pass to complete, small enough to keep the suite
// fast.
const testCodexObserveGenerousBudget = 30 * time.Second

// raiseCodexObserveBudgetForStop sets o.CodexModelObserveTimeout to
// testCodexObserveGenerousBudget, for the common shape across the Stop
// tests below: Spawn runs first under testOrg's own tiny default (so a
// spawn with no fixture on disk yet still returns quickly, honored
// unknown), then a fixture appears, then Stop's own separate observation
// must actually find it. Every call site places this AFTER the Spawn call
// it follows and BEFORE the Stop call it precedes -- never before Spawn,
// or Spawn's own poll would wait out the generous budget instead of
// returning its intended "nothing here yet" result.
func raiseCodexObserveBudgetForStop(o *Org) {
	o.CodexModelObserveTimeout = testCodexObserveGenerousBudget
}

// testOrg builds an Org backed by temp-file manifest/receipt stores and
// fresh fake driver clients, using the shared testOrgConfig() from
// envelope_test.go (max_seats=3, claude+codex pools).
//
// SendEnterDelay is pinned to a tiny value here (rather than left at the
// zero value, which would fall back to the real 750ms
// defaultSendEnterDelay) so every Send test that doesn't care about timing
// itself stays fast -- fakeHerdr's AgentWait/PaneSendText/PaneSendKeys
// return instantly, so the pre-Enter wait would otherwise be the only real
// sleep in the whole suite, once per Send call. Tests that exercise timing
// directly (the delay actually elapsing, ctx expiring mid-delay) override
// this field on the returned *Org after construction.
//
// CodexSessionsDir is pinned to an empty "codex-sessions" subdirectory of
// dir (created lazily -- nothing writes to it unless a test asks it to) so
// no test in this package ever falls through to resolving CODEX_HOME/HOME
// and reading a developer's real ~/.codex/sessions -- codexSessionsDir
// treats a non-empty override literally, with no further
// CodexSessionsDir(codexHome, home) join on top. A test that wants a
// qualifying fixture writes into this same directory under its own
// YYYY/MM/DD layout (codex_session_test.go's fixture helpers). The
// observe timeout/interval are pinned to near-zero so a codex seat's
// spawn-time poll never adds real wall-clock time to a test that doesn't
// care about it -- tests that do (the poll actually retrying, or a pass
// that must genuinely find a record) override CodexModelObserveTimeout
// (testCodexObserveGenerousBudget, above, for the found case) and/or
// CodexModelObserveInterval on the returned *Org after construction, the
// same pattern SendEnterDelay already uses above.
//
// Getenv is pinned to report every variable unset, so Stop's own-pane check
// (HERDR_PANE_ID) never depends on whether `go test` itself runs inside a
// herdr pane. Self-pane tests replace it on the returned *Org.
func testOrg(t *testing.T) (*Org, *fakeHerdr, *fakeAgmsg) {
	t.Helper()
	dir := t.TempDir()
	h := &fakeHerdr{}
	a := &fakeAgmsg{}
	o := &Org{
		Config:                    testOrgConfig(),
		Manifest:                  NewManifestStoreAtPath(ManifestPathIn(dir)),
		Receipts:                  NewReceiptStoreAtPath(filepath.Join(dir, "receipts.jsonl")),
		Herdr:                     h,
		Agmsg:                     a,
		SendEnterDelay:            time.Millisecond,
		CodexSessionsDir:          filepath.Join(dir, "codex-sessions"),
		CodexModelObserveTimeout:  time.Millisecond,
		CodexModelObserveInterval: time.Millisecond,
		Getenv:                    func(string) string { return "" },
	}
	return o, h, a
}

func mustSpawnParams(orgID, seatID string) SpawnParams {
	return SpawnParams{
		OrgID: orgID, SeatID: seatID, Role: "worker", Driver: "claude", Model: "sonnet",
		Cwd: "/tmp/seat", TimeoutMS: 5000,
		// Scope is set by default so every test that doesn't care about the
		// AC-2b minimum control gate (--scope required for autonomous mode,
		// the config default -- see testOrgConfig()/config.Default()) still
		// clears it; tests that specifically exercise Scope/the gate itself
		// override this field explicitly.
		Scope: "test-scope",
	}
}

func eventNames(t *testing.T, o *Org) []string {
	t.Helper()
	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	names := make([]string, len(rr.Events))
	for i, ev := range rr.Events {
		names[i] = ev.Event
	}
	return names
}

func TestOrgSpawn_HappyPath_EventSequenceAndReceipt(t *testing.T) {
	o, h, a := testOrg(t)

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected SpawnOutcomeSpawned, got %v (err=%v)", result.Outcome, result.Err)
	}
	if result.Err != nil {
		t.Fatalf("expected nil Err on success, got %v", result.Err)
	}
	if result.Seat.PaneID != "pane-1" {
		t.Fatalf("expected pane_id pane-1, got %q", result.Seat.PaneID)
	}

	// tab_created, agent_started, agmsg_leader_joined, agmsg_joined, agmsg_announced.
	want := []string{EventSpawnStarted, EventOrgWorkspaceCreated, EventSpawnStep, EventSpawnStep, EventSpawnStep, EventSpawnStep, EventSpawnStep, EventSpawned}
	got := eventNames(t, o)
	if len(got) != len(want) {
		t.Fatalf("expected %d events %v, got %d: %v", len(want), want, len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event[%d]: want %q, got %q (full sequence: %v)", i, want[i], got[i], got)
		}
	}

	wantCalls := []string{"workspace_create", "tab_create", "agent_start"}
	if len(h.calls) != len(wantCalls) {
		t.Fatalf("expected herdr calls %v, got %v", wantCalls, h.calls)
	}
	wantAgmsgCalls := []string{"join:leader", "join:seat-1", "send"}
	if len(a.calls) != len(wantAgmsgCalls) {
		t.Fatalf("expected agmsg calls %v, got %v", wantAgmsgCalls, a.calls)
	}
	for i := range wantAgmsgCalls {
		if a.calls[i] != wantAgmsgCalls[i] {
			t.Errorf("agmsg call[%d]: want %q, got %q (full: %v)", i, wantAgmsgCalls[i], a.calls[i], a.calls)
		}
	}
	if len(a.joinCalls) != 2 {
		t.Fatalf("expected 2 Join calls (leader then seat), got %+v", a.joinCalls)
	}
	leaderJoin, seatJoin := a.joinCalls[0], a.joinCalls[1]
	if leaderJoin.agentID != "leader" || leaderJoin.agmsgType != "claude-code" || leaderJoin.projectPath != "/tmp/seat" {
		t.Errorf("expected leader Join(team, leader, claude-code, /tmp/seat), got %+v", leaderJoin)
	}
	if seatJoin.agentID != "seat-1" || seatJoin.agmsgType != "claude-code" || seatJoin.projectPath != "/tmp/seat" {
		t.Errorf("expected seat Join(team, seat-1, claude-code, /tmp/seat) for a claude driver seat, got %+v", seatJoin)
	}
	if leaderJoin.team != seatJoin.team {
		t.Errorf("expected leader and seat Join calls to target the same team, got %q vs %q", leaderJoin.team, seatJoin.team)
	}

	rr, err := o.Receipts.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	if len(rr.Receipts) != 1 {
		t.Fatalf("expected 1 receipt, got %d", len(rr.Receipts))
	}
	if rr.Receipts[0].Honored != HonoredUnknown {
		t.Errorf("expected honored=unknown for an interactive spawn, got %q", rr.Receipts[0].Honored)
	}
	if rr.Receipts[0].CommandedModel != "sonnet" {
		t.Errorf("expected commanded_model=sonnet, got %q", rr.Receipts[0].CommandedModel)
	}
}

func TestOrgSpawn_HerdrAgentNameNamespacedByOrgID(t *testing.T) {
	// Two different orgs spawning a seat with the same seat_id must not
	// collide in herdr's global agent namespace: AgentStart's agent name
	// must differ per org_id even though SeatID is identical.
	o, h, _ := testOrg(t)

	if r := o.Spawn(mustSpawnParams("org-a", "reviewer")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("org-a reviewer spawn failed: %+v", r)
	}
	if r := o.Spawn(mustSpawnParams("org-b", "reviewer")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("org-b reviewer spawn failed: %+v", r)
	}

	if len(h.agentStartNames) != 2 {
		t.Fatalf("expected 2 AgentStart calls, got %d: %v", len(h.agentStartNames), h.agentStartNames)
	}
	nameA, nameB := h.agentStartNames[0], h.agentStartNames[1]
	if nameA == nameB {
		t.Fatalf("expected distinct herdr agent names for the same seat_id in different org_ids, got %q for both", nameA)
	}
	wantA, wantB := herdrAgentName("org-a", "reviewer"), herdrAgentName("org-b", "reviewer")
	if nameA != wantA || nameB != wantB {
		t.Fatalf("expected agent names %q and %q, got %q and %q", wantA, wantB, nameA, nameB)
	}
}

func TestHerdrAgentName_NamespacesBySeatAndOrg(t *testing.T) {
	if got := herdrAgentName("org-a", "reviewer"); got != "org-a_reviewer" {
		t.Fatalf("herdrAgentName(org-a, reviewer) = %q, want org-a_reviewer", got)
	}
	if herdrAgentName("org-a", "reviewer") == herdrAgentName("org-b", "reviewer") {
		t.Fatalf("expected herdrAgentName to differ across org_ids for the same seat_id")
	}
}

func TestOrgSpawn_WorkspaceReusedForSecondSeat(t *testing.T) {
	o, h, _ := testOrg(t)

	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("seat-1 spawn failed: %+v", r)
	}
	if r := o.Spawn(mustSpawnParams("org-a", "seat-2")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("seat-2 spawn failed: %+v", r)
	}

	workspaceCreates := 0
	for _, c := range h.calls {
		if c == "workspace_create" {
			workspaceCreates++
		}
	}
	if workspaceCreates != 1 {
		t.Fatalf("expected workspace_create called exactly once across 2 seats in the same org, got %d (calls=%v)", workspaceCreates, h.calls)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	orgLevelWorkspaceEvents := 0
	for _, ev := range rr.Events {
		if ev.Event == EventOrgWorkspaceCreated {
			orgLevelWorkspaceEvents++
		}
	}
	if orgLevelWorkspaceEvents != 1 {
		t.Fatalf("expected exactly 1 org_workspace_created event, got %d", orgLevelWorkspaceEvents)
	}
}

// TestOrgSpawn_AfterDisbandClosedWorkspace_CreatesNewWorkspace covers plan
// AC13: once disband closed the org's workspace and recorded
// org_workspace_closed, the next spawn in the same org_id creates a new
// workspace and puts its tab there instead of in the closed one, and the
// spawn after that reuses the new workspace.
func TestOrgSpawn_AfterDisbandClosedWorkspace_CreatesNewWorkspace(t *testing.T) {
	o, h, _ := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("first spawn failed: %+v", r)
	}
	if d := o.Disband(DisbandParams{OrgID: "org-a"}); !d.Disbanded || len(d.Errs) != 0 {
		t.Fatalf("disband failed: %+v", d)
	}
	if !slices.Equal(h.workspaceCloseCalls, []string{"ws-1"}) {
		t.Fatalf("expected disband to close ws-1, got %v", h.workspaceCloseCalls)
	}

	h.workspaceID = "ws-2"
	h.paneID = "pane-2"
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn after disband failed: %+v", r)
	}
	h.paneID = "pane-3"
	if r := o.Spawn(mustSpawnParams("org-a", "seat-2")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("second spawn after disband failed: %+v", r)
	}

	if got := countString(h.calls, "workspace_create"); got != 2 {
		t.Fatalf("expected workspace_create once before and once after the disband, got %d (calls=%v)", got, h.calls)
	}
	if want := []string{"ws-1", "ws-2", "ws-2"}; !slices.Equal(h.tabCreateWorkspaces, want) {
		t.Fatalf("TabCreate workspaces = %v, want %v (no tab in the closed ws-1)", h.tabCreateWorkspaces, want)
	}
	var created []string
	for _, ev := range mustReadEvents(t, o) {
		if ev.Event == EventOrgWorkspaceCreated {
			created = append(created, ev.PaneID)
		}
	}
	if !slices.Equal(created, []string{"ws-1", "ws-2"}) {
		t.Fatalf("org_workspace_created ids = %v, want [ws-1 ws-2]", created)
	}
}

func countString(list []string, s string) int {
	n := 0
	for _, v := range list {
		if v == s {
			n++
		}
	}
	return n
}

// TestOpenOrgWorkspaces pins which recorded workspaces count as open: a
// created id stays open until a later real org-level org_workspace_closed
// for the same org_id and id, and a closed id that is created again is open
// again.
func TestOpenOrgWorkspaces(t *testing.T) {
	created := func(org, id string) ManifestEvent {
		return ManifestEvent{OrgID: org, Event: EventOrgWorkspaceCreated, PaneID: id}
	}
	closed := func(org, id string) ManifestEvent {
		return ManifestEvent{OrgID: org, Event: EventOrgWorkspaceClosed, PaneID: id}
	}
	dryRunClosed := closed("org-a", "ws-1")
	dryRunClosed.DryRun = true
	seatLevelClosed := closed("org-a", "ws-1")
	seatLevelClosed.SeatID = "seat-1"

	tests := []struct {
		name   string
		events []ManifestEvent
		want   []string
	}{
		{name: "none recorded", want: nil},
		{name: "created", events: []ManifestEvent{created("org-a", "ws-1")}, want: []string{"ws-1"}},
		{name: "created then closed", events: []ManifestEvent{created("org-a", "ws-1"), closed("org-a", "ws-1")}, want: nil},
		{name: "closed then a new one created", events: []ManifestEvent{created("org-a", "ws-1"), closed("org-a", "ws-1"), created("org-a", "ws-2")}, want: []string{"ws-2"}},
		{name: "same id created again after close", events: []ManifestEvent{created("org-a", "ws-1"), closed("org-a", "ws-1"), created("org-a", "ws-1")}, want: []string{"ws-1"}},
		{name: "two created, oldest first", events: []ManifestEvent{created("org-a", "ws-1"), created("org-a", "ws-2")}, want: []string{"ws-1", "ws-2"}},
		{name: "close of another id", events: []ManifestEvent{created("org-a", "ws-1"), closed("org-a", "ws-9")}, want: []string{"ws-1"}},
		{name: "close in another org", events: []ManifestEvent{created("org-a", "ws-1"), closed("org-b", "ws-1")}, want: []string{"ws-1"}},
		{name: "workspace of another org", events: []ManifestEvent{created("org-b", "ws-1")}, want: nil},
		{name: "dry-run close ignored", events: []ManifestEvent{created("org-a", "ws-1"), dryRunClosed}, want: []string{"ws-1"}},
		{name: "seat-level close ignored", events: []ManifestEvent{created("org-a", "ws-1"), seatLevelClosed}, want: []string{"ws-1"}},
		{name: "empty id skipped", events: []ManifestEvent{created("org-a", "")}, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := openOrgWorkspaces(tt.events, "org-a"); !slices.Equal(got, tt.want) {
				t.Fatalf("openOrgWorkspaces = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrgSpawn_Rejected_OutOfPoolModel(t *testing.T) {
	o, h, a := testOrg(t)

	p := mustSpawnParams("org-a", "seat-1")
	p.Model = "not-a-real-model"
	result := o.Spawn(p)

	if result.Outcome != SpawnOutcomeRejected {
		t.Fatalf("expected SpawnOutcomeRejected, got %v", result.Outcome)
	}
	if result.Err == nil {
		t.Fatal("expected non-nil Err for a rejection (CLI must exit non-zero)")
	}
	if len(h.calls) != 0 || len(a.calls) != 0 {
		t.Fatalf("expected no driver calls on rejection, got herdr=%v agmsg=%v", h.calls, a.calls)
	}

	got := eventNames(t, o)
	if len(got) != 1 || got[0] != EventRejected {
		t.Fatalf("expected exactly one rejected event, got %v", got)
	}

	rr, err := o.Receipts.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	if len(rr.Receipts) != 1 || rr.Receipts[0].Honored != HonoredFalse {
		t.Fatalf("expected 1 receipt with honored=false, got %+v", rr.Receipts)
	}
}

func TestOrgSpawn_Rejected_MaxSeatsReached_OrgIsolated(t *testing.T) {
	o, _, _ := testOrg(t) // testOrgConfig(): MaxSeats: 3

	for i := range 3 {
		seatID := "seat-" + string(rune('a'+i))
		if r := o.Spawn(mustSpawnParams("org-a", seatID)); r.Outcome != SpawnOutcomeSpawned {
			t.Fatalf("expected seat %s to spawn while under max_seats, got %+v", seatID, r)
		}
	}

	// A 4th seat in org-a must be rejected: max_seats(3) reached.
	result := o.Spawn(mustSpawnParams("org-a", "seat-overflow"))
	if result.Outcome != SpawnOutcomeRejected {
		t.Fatalf("expected 4th seat in org-a to be rejected (max_seats reached), got %v", result.Outcome)
	}

	// A seat in a *different* org_id must not be blocked by org-a's count
	// (AC-2: no cross-namespace bleed).
	otherOrgResult := o.Spawn(mustSpawnParams("org-b", "seat-1"))
	if otherOrgResult.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected org-b's first seat to spawn despite org-a being at max_seats, got %v (err=%v)", otherOrgResult.Outcome, otherOrgResult.Err)
	}
}

func TestOrgSpawn_Idempotent_AlreadySpawned_NoNewDriverCalls(t *testing.T) {
	o, h, a := testOrg(t)

	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("initial spawn failed: %+v", r)
	}
	callsBefore := len(h.calls)
	sendsBefore := len(a.calls)

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeIdempotent {
		t.Fatalf("expected SpawnOutcomeIdempotent on respawn of an already-spawned seat, got %v", result.Outcome)
	}
	if result.Err != nil {
		t.Fatalf("expected nil Err for idempotent respawn (CLI must exit 0), got %v", result.Err)
	}
	if len(h.calls) != callsBefore || len(a.calls) != sendsBefore {
		t.Fatalf("expected no new driver calls on idempotent respawn, herdr %d->%d agmsg %d->%d", callsBefore, len(h.calls), sendsBefore, len(a.calls))
	}
}

// TestOrgSpawn_Idempotent_NoScopeRetry_ReturnsExistingSeat is the regression
// test for cross-review-triage-org-runtime-lead.md ACTION_REQUIRED #1: under
// the default (autonomous) permission mode, retrying `spawn` for an
// already-spawned seat *without* --scope must return the existing seat
// idempotently, not be rejected by the AC-2b minimum control gate. The gate
// only exists to fail-close a *new* unscoped autonomous seat; a no-op retry
// of an existing active seat creates no new seat at all, so it must never
// reach the gate -- exactly the same idempotent-vs-validation ordering
// TestOrgSpawn_Idempotent_AtMaxSeats_RespawnSucceedsInsteadOfRejected already
// covers for envelope/capacity validation.
func TestOrgSpawn_Idempotent_NoScopeRetry_ReturnsExistingSeat(t *testing.T) {
	o, h, a := testOrg(t)
	// testOrgConfig() leaves Permissions unset, which resolves to the
	// default autonomous mode (ResolvePermissionMode) -- exactly the
	// condition the AC-2b gate targets.

	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("initial spawn failed: %+v", r)
	}

	rrBefore, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	eventsBefore := len(rrBefore.Events)
	callsBefore := len(h.calls)
	sendsBefore := len(a.calls)

	p := mustSpawnParams("org-a", "seat-1")
	p.Scope = ""
	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeIdempotent {
		t.Fatalf("expected SpawnOutcomeIdempotent for a scope-less retry of an already-spawned seat under autonomous default, got %v (err=%v)", result.Outcome, result.Err)
	}
	if result.Err != nil {
		t.Fatalf("expected nil Err for idempotent respawn (CLI must exit 0), got %v", result.Err)
	}

	rrAfter, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if len(rrAfter.Events) != eventsBefore {
		t.Fatalf("expected no new manifest events on idempotent no-scope retry, before=%d after=%d (events=%+v)", eventsBefore, len(rrAfter.Events), rrAfter.Events)
	}
	if len(h.calls) != callsBefore || len(a.calls) != sendsBefore {
		t.Fatalf("expected no new driver calls on idempotent no-scope retry, herdr %d->%d agmsg %d->%d", callsBefore, len(h.calls), sendsBefore, len(a.calls))
	}
}

func TestOrgSpawn_Idempotent_AtMaxSeats_RespawnSucceedsInsteadOfRejected(t *testing.T) {
	// Regression for cross-review-triage-org-runtime-mechanism.md ACTION_REQUIRED #1:
	// with max_seats=1, respawning the org's only (already-spawned) seat must
	// return idempotently -- envelope validation (including max_seats) must
	// never even run for an already-spawned seat, so an at-cap org cannot
	// turn a legitimate respawn retry into a rejection.
	o, h, a := testOrg(t)
	o.Config.MaxSeats = 1

	if r := o.Spawn(mustSpawnParams("org-a", "seat-a")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("initial spawn failed: %+v", r)
	}

	rrBefore, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	eventsBefore := len(rrBefore.Events)
	callsBefore := len(h.calls)
	sendsBefore := len(a.calls)

	result := o.Spawn(mustSpawnParams("org-a", "seat-a"))
	if result.Outcome != SpawnOutcomeIdempotent {
		t.Fatalf("expected SpawnOutcomeIdempotent for respawn of the org's only seat at max_seats=1, got %v (err=%v)", result.Outcome, result.Err)
	}
	if result.Err != nil {
		t.Fatalf("expected nil Err for idempotent respawn, got %v", result.Err)
	}

	rrAfter, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if len(rrAfter.Events) != eventsBefore {
		t.Fatalf("expected no new manifest events on idempotent respawn at max_seats, %d -> %d (events=%v)", eventsBefore, len(rrAfter.Events), rrAfter.Events)
	}
	if len(h.calls) != callsBefore || len(a.calls) != sendsBefore {
		t.Fatalf("expected no driver calls on idempotent respawn at max_seats, herdr %d->%d agmsg %d->%d", callsBefore, len(h.calls), sendsBefore, len(a.calls))
	}
}

func TestOrgSpawn_Rejected_AtMaxSeats_NewSeatStillRejected(t *testing.T) {
	// Unchanged-behavior guard alongside the fix above: a *new* seat_id at
	// max_seats=1 must still be rejected, not treated as idempotent.
	o, h, a := testOrg(t)
	o.Config.MaxSeats = 1

	if r := o.Spawn(mustSpawnParams("org-a", "seat-a")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("initial spawn failed: %+v", r)
	}
	callsBefore := len(h.calls)
	sendsBefore := len(a.calls)

	result := o.Spawn(mustSpawnParams("org-a", "seat-b"))
	if result.Outcome != SpawnOutcomeRejected {
		t.Fatalf("expected SpawnOutcomeRejected for a new seat at max_seats=1, got %v", result.Outcome)
	}
	if result.Err == nil {
		t.Fatal("expected non-nil Err for a rejection (CLI must exit non-zero)")
	}
	if len(h.calls) != callsBefore || len(a.calls) != sendsBefore {
		t.Fatalf("expected no driver calls on rejection, herdr %d->%d agmsg %d->%d", callsBefore, len(h.calls), sendsBefore, len(a.calls))
	}
}

func TestOrgSpawn_StaleInFlight_AtMaxSeats_CompensationFreesCapForFreshSaga(t *testing.T) {
	// Regression companion: a stale in-flight seat at max_seats=1 must be
	// compensated (spawn_failed) *before* ValidateSpawn runs, so the stale
	// seat no longer counts toward activeSeats and the fresh saga succeeds
	// instead of being rejected for being "at cap".
	o, h, _ := testOrg(t)
	o.Config.MaxSeats = 1

	if err := o.Manifest.Append(ManifestEvent{
		TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-a", Event: EventSpawnStarted,
		Role: "worker", Driver: "claude", Model: "sonnet", PaneID: "stale-pane-1",
	}); err != nil {
		t.Fatalf("seed stale spawn_started: %v", err)
	}

	result := o.Spawn(mustSpawnParams("org-a", "seat-a"))
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected fresh spawn to succeed at max_seats=1 after compensating the stale seat that no longer counts toward the cap, got %+v", result)
	}
	if len(h.sendKeysCalls) == 0 || h.sendKeysCalls[0] != "stale-pane-1" {
		t.Fatalf("expected compensation C-c sent to the stale pane stale-pane-1, got %v", h.sendKeysCalls)
	}

	got := eventNames(t, o)
	want := []string{EventSpawnStarted, EventSpawnFailed, EventSpawnStarted, EventOrgWorkspaceCreated, EventSpawnStep, EventSpawnStep, EventSpawnStep, EventSpawnStep, EventSpawnStep, EventSpawned}
	if len(got) != len(want) {
		t.Fatalf("expected event sequence %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event[%d]: want %q, got %q (full: %v)", i, want[i], got[i], got)
		}
	}
}

func TestOrgSpawn_StaleInFlight_StatelessEnvelopeViolation_RejectedBeforeCompensation(t *testing.T) {
	// Regression for cycle-2 self-review MEDIUM 1
	// (docs/reports/self-review-2026-08-01-org-runtime-mechanism.md): a
	// stateless envelope violation (out-of-pool model, here) must be
	// rejected before any stale-in-flight compensation is attempted, even
	// when a stale spawn_started event already exists for the same seat.
	// Compensation is a destructive external side effect (PaneSendKeys
	// C-c) plus a spawn_failed manifest write -- neither may happen on a
	// request that was always going to be rejected on stateless grounds.
	o, h, a := testOrg(t)

	if err := o.Manifest.Append(ManifestEvent{
		TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-a", Event: EventSpawnStarted,
		Role: "worker", Driver: "claude", Model: "sonnet", PaneID: "stale-pane-1",
	}); err != nil {
		t.Fatalf("seed stale spawn_started: %v", err)
	}

	p := mustSpawnParams("org-a", "seat-a")
	p.Model = "not-a-real-model"
	result := o.Spawn(p)

	if result.Outcome != SpawnOutcomeRejected {
		t.Fatalf("expected SpawnOutcomeRejected for a stateless envelope violation against a stale seat, got %v (err=%v)", result.Outcome, result.Err)
	}
	if result.Err == nil {
		t.Fatal("expected non-nil Err for a rejection (CLI must exit non-zero)")
	}
	if len(h.calls) != 0 || len(a.calls) != 0 {
		t.Fatalf("expected zero driver calls (no C-c compensation, no saga steps), got herdr=%v agmsg=%v", h.calls, a.calls)
	}
	if len(h.sendKeysCalls) != 0 {
		t.Fatalf("expected no PaneSendKeys compensation calls, got %v", h.sendKeysCalls)
	}

	got := eventNames(t, o)
	want := []string{EventSpawnStarted, EventRejected}
	if len(got) != len(want) {
		t.Fatalf("expected event sequence %v (the seeded spawn_started plus one rejected -- no spawn_failed), got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event[%d]: want %q, got %q (full: %v)", i, want[i], got[i], got)
		}
	}
}

func TestOrgSpawn_StaleInFlight_CompensatesThenRespawnsFresh(t *testing.T) {
	o, h, _ := testOrg(t)

	// Simulate a crashed spawn: spawn_started with a persisted pane id, but
	// no terminal event.
	if err := o.Manifest.Append(ManifestEvent{
		TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawnStarted,
		Role: "worker", Driver: "claude", Model: "sonnet", PaneID: "stale-pane-1",
	}); err != nil {
		t.Fatalf("seed stale spawn_started: %v", err)
	}

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected fresh spawn to succeed after compensating a stale in-flight saga, got %+v", result)
	}

	if len(h.sendKeysCalls) == 0 || h.sendKeysCalls[0] != "stale-pane-1" {
		t.Fatalf("expected compensation C-c sent to the stale pane stale-pane-1, got %v", h.sendKeysCalls)
	}

	got := eventNames(t, o)
	// stale spawn_started (seeded) -> spawn_failed (compensation) -> fresh saga.
	want := []string{EventSpawnStarted, EventSpawnFailed, EventSpawnStarted, EventOrgWorkspaceCreated, EventSpawnStep, EventSpawnStep, EventSpawnStep, EventSpawnStep, EventSpawnStep, EventSpawned}
	if len(got) != len(want) {
		t.Fatalf("expected event sequence %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event[%d]: want %q, got %q (full: %v)", i, want[i], got[i], got)
		}
	}
}

// TestOrgSpawn_StaleInFlight_RacerCompletesDuringCompensationWindow_Phase2ReturnsIdempotent
// is a regression for self-review Cycle-2 M-2: Phase 2 re-reads the manifest
// under a re-acquired lock but, before this fix, only re-ran the capacity
// check against that fresh snapshot -- not the idempotent check. A
// concurrent racer's Spawn call for this exact seat can complete a full
// saga during the lock-free window between Phase 1's release (right after
// staleExisting is detected) and Phase 2's re-acquire, since compensateStale's
// only driver call (PaneSendKeys) runs in that window. Phase 2 must notice
// the fresh EventSpawned terminus and return SpawnOutcomeIdempotent instead
// of appending a second spawn_started on top of an already-spawned seat.
func TestOrgSpawn_StaleInFlight_RacerCompletesDuringCompensationWindow_Phase2ReturnsIdempotent(t *testing.T) {
	o, h, _ := testOrg(t)

	if err := o.Manifest.Append(ManifestEvent{
		TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-a", Event: EventSpawnStarted,
		Role: "worker", Driver: "claude", Model: "sonnet", PaneID: "stale-pane-1",
	}); err != nil {
		t.Fatalf("seed stale spawn_started: %v", err)
	}

	// Fires right after compensateStale's own spawn_failed write -- i.e. in
	// the lock-free window between Phase 1's release and Phase 2's
	// re-acquire -- to simulate a concurrent racer's Spawn call completing a
	// full saga for the same seat before this call's Phase 2 re-reads the
	// manifest. This is the seam spawn.go documents at afterStaleCompensation's
	// declaration.
	orig := afterStaleCompensation
	afterStaleCompensation = func() {
		if err := o.Manifest.Append(ManifestEvent{
			TS: "2026-08-01T00:00:05Z", OrgID: "org-a", SeatID: "seat-a", Event: EventSpawned,
			Role: "worker", Driver: "claude", Model: "sonnet", PaneID: "racer-pane-1", AgmsgTeam: "team-racer",
		}); err != nil {
			t.Fatalf("seed racer spawned event: %v", err)
		}
	}
	defer func() { afterStaleCompensation = orig }()

	result := o.Spawn(mustSpawnParams("org-a", "seat-a"))
	if result.Outcome != SpawnOutcomeIdempotent {
		t.Fatalf("expected SpawnOutcomeIdempotent once Phase 2's fresh read shows the seat already spawned by a racer, got %+v", result)
	}
	if result.Seat.PaneID != "racer-pane-1" {
		t.Fatalf("expected the idempotent result to reflect the racer's spawned seat (pane racer-pane-1), got %+v", result.Seat)
	}

	// The only driver call is the compensation C-c: no second saga's worth
	// of workspace/tab/agent/agmsg calls were attempted on top of the
	// racer's already-spawned seat.
	if len(h.calls) != 1 || h.calls[0] != "pane_send_keys" {
		t.Fatalf("expected exactly one driver call (the compensation C-c), got %v", h.calls)
	}

	got := eventNames(t, o)
	// stale spawn_started (seeded) -> spawn_failed (compensation) -> racer's spawned -- no second spawn_started.
	want := []string{EventSpawnStarted, EventSpawnFailed, EventSpawned}
	if len(got) != len(want) {
		t.Fatalf("expected event sequence %v (no duplicate spawn_started appended over the racer's spawned seat), got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("event[%d]: want %q, got %q (full: %v)", i, want[i], got[i], got)
		}
	}
}

// TestOrgSpawn_DryRun_And_Real_AgreeOnRejectionCause_EnvelopeBeforeScopeGate
// is a regression for self-review Cycle-2 M-1: dry-run and the real spawn
// path used to check the AC-2b scope gate and the envelope validation in
// opposite orders, so a request that violated both an out-of-pool model
// (ValidateSpawnEnvelope) and the unscoped-autonomous gate (AC-2b) was
// rejected for a different cause depending on --dry-run alone -- even
// though dry-run's whole purpose is to predict the real path's rejection.
// Both paths must now report the same first cause: the envelope's
// out-of-pool-model error wins in both modes.
func TestOrgSpawn_DryRun_And_Real_AgreeOnRejectionCause_EnvelopeBeforeScopeGate(t *testing.T) {
	newParams := func(dryRun bool) SpawnParams {
		p := mustSpawnParams("org-a", "seat-1")
		p.Model = "not-a-real-model" // out-of-pool -> ValidateSpawnEnvelope error
		p.Scope = ""                 // unscoped autonomous -> AC-2b gate error
		p.DryRun = dryRun
		return p
	}

	realOrg, _, _ := testOrg(t)
	realResult := realOrg.Spawn(newParams(false))
	if realResult.Outcome != SpawnOutcomeRejected || realResult.Err == nil {
		t.Fatalf("expected real spawn to reject, got %+v", realResult)
	}
	if !strings.Contains(realResult.Err.Error(), "not in [org].model_pool") {
		t.Fatalf("expected real spawn to reject for the envelope (out-of-pool model) cause, got %v", realResult.Err)
	}
	if strings.Contains(realResult.Err.Error(), "--scope") {
		t.Fatalf("expected real spawn's rejection to NOT mention --scope (envelope check must win first), got %v", realResult.Err)
	}

	dryOrg, _, _ := testOrg(t)
	dryResult := dryOrg.Spawn(newParams(true))
	if dryResult.Outcome != SpawnOutcomeRejected || dryResult.Err == nil {
		t.Fatalf("expected dry-run spawn to reject, got %+v", dryResult)
	}
	if !strings.Contains(dryResult.Err.Error(), "not in [org].model_pool") {
		t.Fatalf("expected dry-run spawn to reject for the envelope (out-of-pool model) cause, got %v", dryResult.Err)
	}
	if strings.Contains(dryResult.Err.Error(), "--scope") {
		t.Fatalf("expected dry-run spawn's rejection to NOT mention --scope (envelope check must win first), got %v", dryResult.Err)
	}

	if realResult.Err.Error() != dryResult.Err.Error() {
		t.Fatalf("expected dry-run and real spawn to record the SAME rejection cause, got real=%q dry=%q", realResult.Err.Error(), dryResult.Err.Error())
	}

	// Both paths route rejection through reject(), so both must record the
	// same Details string on the resulting rejected manifest event.
	realRR, err := realOrg.Manifest.Read()
	if err != nil {
		t.Fatalf("read real manifest: %v", err)
	}
	dryRR, err := dryOrg.Manifest.Read()
	if err != nil {
		t.Fatalf("read dry-run manifest: %v", err)
	}
	if len(realRR.Events) != 1 || len(dryRR.Events) != 1 {
		t.Fatalf("expected exactly one rejected event each, got real=%+v dry=%+v", realRR.Events, dryRR.Events)
	}
	if realRR.Events[0].Details != dryRR.Events[0].Details {
		t.Fatalf("expected the rejected event's Details to match between dry-run and real spawn, got real=%q dry=%q", realRR.Events[0].Details, dryRR.Events[0].Details)
	}
}

func TestOrgSpawn_FailureInjection_WorkspaceCreate_NoCompensationAttempted(t *testing.T) {
	o, h, _ := testOrg(t)
	h.workspaceCreateErr = errors.New("stub failure: workspace create")

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeFailed {
		t.Fatalf("expected SpawnOutcomeFailed, got %v", result.Outcome)
	}
	if result.Err == nil {
		t.Fatal("expected non-nil Err on saga failure")
	}
	if len(h.sendKeysCalls) != 0 {
		t.Fatalf("expected no compensation attempt when no pane was ever created, got %v", h.sendKeysCalls)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	if last.Event != EventSpawnFailed {
		t.Fatalf("expected last event to be spawn_failed, got %q", last.Event)
	}
	assertDetailsContains(t, last.Details, "step=workspace_create", "no pane to compensate")
}

func TestOrgSpawn_FailureInjection_TabCreate_NoCompensationAttempted(t *testing.T) {
	o, h, _ := testOrg(t)
	h.tabCreateErr = errors.New("stub failure: tab create")

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeFailed {
		t.Fatalf("expected SpawnOutcomeFailed, got %v", result.Outcome)
	}
	if len(h.sendKeysCalls) != 0 {
		t.Fatalf("expected no compensation attempt: tab_create itself never returned a pane id, got %v", h.sendKeysCalls)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	if last.Event != EventSpawnFailed {
		t.Fatalf("expected last event to be spawn_failed, got %q", last.Event)
	}
	assertDetailsContains(t, last.Details, "step=tab_create")
}

func TestOrgSpawn_FailureInjection_AgentStart_CompensatesExistingPane(t *testing.T) {
	o, h, _ := testOrg(t)
	h.agentStartErr = errors.New("stub failure: agent start")

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeFailed {
		t.Fatalf("expected SpawnOutcomeFailed, got %v", result.Outcome)
	}
	if len(h.sendKeysCalls) != 1 || h.sendKeysCalls[0] != "pane-1" {
		t.Fatalf("expected exactly one compensation C-c to pane-1 (created by tab_create before the failure), got %v", h.sendKeysCalls)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	if last.Event != EventSpawnFailed {
		t.Fatalf("expected last event to be spawn_failed, got %q", last.Event)
	}
	if last.PaneID != "pane-1" {
		t.Fatalf("expected orphaned pane_id pane-1 to remain traceable on the spawn_failed event, got %q", last.PaneID)
	}
	assertDetailsContains(t, last.Details, "step=agent_start", "C-c sent")
}

func TestOrgSpawn_FailureInjection_AgmsgSend_CompensatesExistingPane(t *testing.T) {
	o, h, a := testOrg(t)
	a.sendErr = errors.New("stub failure: agmsg send")

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeFailed {
		t.Fatalf("expected SpawnOutcomeFailed, got %v", result.Outcome)
	}
	if len(h.sendKeysCalls) != 1 || h.sendKeysCalls[0] != "pane-1" {
		t.Fatalf("expected exactly one compensation C-c to pane-1, got %v", h.sendKeysCalls)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	if last.Event != EventSpawnFailed {
		t.Fatalf("expected last event to be spawn_failed, got %q", last.Event)
	}
	if last.PaneID != "pane-1" {
		t.Fatalf("expected orphaned pane_id pane-1 to remain traceable, got %q", last.PaneID)
	}
	assertDetailsContains(t, last.Details, "step=agmsg_announce", "leader_join=ok")
}

func TestOrgSpawn_FailureInjection_AgmsgJoin_SeatJoinFails_CompensatesExistingPane(t *testing.T) {
	// Seat Join is a hard-failure gate distinct from the leader's best-effort
	// ensureLeaderJoined: a seat Join failure must fail the saga at
	// "agmsg_join", before the HELLO Send is ever attempted.
	o, h, a := testOrg(t)
	a.joinErrs = map[string]error{"seat-1": errors.New("stub failure: seat join")}

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeFailed {
		t.Fatalf("expected SpawnOutcomeFailed, got %v", result.Outcome)
	}
	if len(h.sendKeysCalls) != 1 || h.sendKeysCalls[0] != "pane-1" {
		t.Fatalf("expected exactly one compensation C-c to pane-1, got %v", h.sendKeysCalls)
	}
	if len(a.calls) != 2 || a.calls[0] != "join:leader" || a.calls[1] != "join:seat-1" {
		t.Fatalf("expected leader Join then seat Join (no Send attempted after seat Join fails), got %v", a.calls)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	if last.Event != EventSpawnFailed {
		t.Fatalf("expected last event to be spawn_failed, got %q", last.Event)
	}
	if last.PaneID != "pane-1" {
		t.Fatalf("expected orphaned pane_id pane-1 to remain traceable, got %q", last.PaneID)
	}
	assertDetailsContains(t, last.Details, "step=agmsg_join", "C-c sent")
}

func TestOrgSpawn_EnsureLeaderJoined_ErrorDoesNotFailSaga_WhenSeatJoinAndSendSucceed(t *testing.T) {
	// ensureLeaderJoined is best-effort: an error joining "leader" (e.g. it was
	// already a member and join.sh soft-failed on the retry) must not fail
	// the saga on its own -- the seat's own Join and the HELLO Send are the
	// authoritative gates. The leader-join error is still recorded on the
	// agmsg_leader_joined spawn_step for diagnosis.
	o, _, a := testOrg(t)
	a.joinErrs = map[string]error{"leader": errors.New("stub: leader already a member")}

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected SpawnOutcomeSpawned despite a leader-join error, got %v (err=%v)", result.Outcome, result.Err)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var leaderJoinedStep *ManifestEvent
	for i := range rr.Events {
		if rr.Events[i].Details == "" {
			continue
		}
		if strings.HasPrefix(rr.Events[i].Details, "agmsg_leader_joined") {
			leaderJoinedStep = &rr.Events[i]
			break
		}
	}
	if leaderJoinedStep == nil {
		t.Fatalf("expected an agmsg_leader_joined spawn_step event, got events %+v", rr.Events)
	}
	assertDetailsContains(t, leaderJoinedStep.Details, "error=")
}

func TestOrgSpawn_FailureInjection_AgmsgSend_DetailsIncludeLeaderJoinError(t *testing.T) {
	// When HELLO Send fails, the recorded leader-join outcome must be carried
	// into the spawn_failed Details alongside the send failure itself, so an
	// operator can immediately see whether a missing "leader" roster entry is
	// the likely root cause.
	o, _, a := testOrg(t)
	a.joinErrs = map[string]error{"leader": errors.New("stub: leader join failed")}
	a.sendErr = errors.New("stub failure: agmsg send")

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeFailed {
		t.Fatalf("expected SpawnOutcomeFailed, got %v", result.Outcome)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	if last.Event != EventSpawnFailed {
		t.Fatalf("expected last event to be spawn_failed, got %q", last.Event)
	}
	assertDetailsContains(t, last.Details, "step=agmsg_announce", "leader_join=", "stub: leader join failed", "stub failure: agmsg send")
}

func TestOrgSpawn_DryRun_NoDriverCalls_EventsFlaggedAndExcludedByDefault(t *testing.T) {
	o, h, a := testOrg(t)

	p := mustSpawnParams("org-a", "seat-1")
	p.DryRun = true
	result := o.Spawn(p)

	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected SpawnOutcomeSpawned for a valid dry-run, got %v (err=%v)", result.Outcome, result.Err)
	}
	if result.Err != nil {
		t.Fatalf("expected nil Err for a successful dry-run, got %v", result.Err)
	}
	if len(h.calls) != 0 || len(a.calls) != 0 {
		t.Fatalf("expected zero driver calls for --dry-run, got herdr=%v agmsg=%v", h.calls, a.calls)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	for _, ev := range rr.Events {
		if !ev.DryRun {
			t.Errorf("expected every dry-run saga event to carry dry_run=true, got %+v", ev)
		}
	}

	statusResult, err := o.Status("org-a", false)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(statusResult.Seats) != 0 {
		t.Fatalf("expected dry-run seat excluded from default status, got %+v", statusResult.Seats)
	}
	statusAll, err := o.Status("org-a", true)
	if err != nil {
		t.Fatalf("Status --all: %v", err)
	}
	if len(statusAll.Seats) != 1 {
		t.Fatalf("expected dry-run seat included with --all, got %+v", statusAll.Seats)
	}

	receiptsResult, err := o.Receipts.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	if len(receiptsResult.Receipts) != 1 || receiptsResult.Receipts[0].Honored != HonoredUnknown || receiptsResult.Receipts[0].Reason != "dry-run" {
		t.Fatalf("expected 1 receipt honored=unknown reason=dry-run, got %+v", receiptsResult.Receipts)
	}
}

// assertDetailsContains fails the test unless details contains every want
// substring.
func assertDetailsContains(t *testing.T, details string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(details, w) {
			t.Errorf("expected details %q to contain %q", details, w)
		}
	}
}

func TestOrgSpawn_HelloMessage_IsProtocolConformant(t *testing.T) {
	// Regression for AC-11: the exact HELLO body the saga sends must itself
	// pass protocol.ValidateText -- the org's own messages must obey the
	// protocol it enforces on `ralph org send`.
	msg := fmt.Sprintf("TYPE: HELLO\nSEAT: %s\nROLE: %s\nORG_ID: %s", "seat-1", "worker", "org-a")
	if err := protocol.ValidateText(msg, 0); err != nil {
		t.Fatalf("expected the saga's HELLO message to be protocol-conformant, got %v (message=%q)", err, msg)
	}
}

func TestOrgSpawn_RoleTemplate_ExpandsIntoInitialPrompt(t *testing.T) {
	// AC-4: --role reviewer expands the embedded reviewer template,
	// substituted with the spawn's own org_id/seat_id/role/scope. Real herdr
	// rejects multi-line agent args (see the maxInlinePromptRunes/
	// needsPromptFile doc comments in spawn.go), so the rendered template is
	// written to a prompt file and the AgentStart argv carries only a
	// one-line pointer to it -- assert the pointer (no newline) via argv and
	// the full rendered content via the file.
	o, h, _ := testOrg(t)

	p := mustSpawnParams("org-a", "seat-1")
	p.Role = "reviewer"
	p.Scope = "internal/org/**"
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	if len(h.agentStartArgs) != 1 {
		t.Fatalf("expected exactly 1 AgentStart call, got %d", len(h.agentStartArgs))
	}
	args := h.agentStartArgs[0]
	// AC-2: permission-mode args (default config -> autonomous ->
	// bypassPermissions) come first, then --model, then the prompt pointer.
	if len(args) != 5 || args[0] != "--permission-mode" || args[1] != "bypassPermissions" || args[2] != "--model" {
		t.Fatalf("expected AgentStart args [--permission-mode bypassPermissions --model <model> <pointer>], got %v", args)
	}
	promptArg := args[4]
	if strings.Contains(promptArg, "\n") {
		t.Fatalf("expected the AgentStart prompt arg to be a single line, got:\n%s", promptArg)
	}
	if !strings.HasPrefix(promptArg, "役割指示を読み込んで従ってください: ") {
		t.Fatalf("expected the AgentStart prompt arg to be the file pointer, got %q", promptArg)
	}
	promptPath := strings.TrimPrefix(promptArg, "役割指示を読み込んで従ってください: ")

	data, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("expected the prompt file to exist at %q: %v", promptPath, err)
	}
	fileContent := string(data)
	for _, want := range []string{"org-a", "seat-1", "reviewer", "internal/org/**", ".claude/rules/ralph/agent-messaging.md"} {
		if !strings.Contains(fileContent, want) {
			t.Errorf("expected the prompt file content to contain %q, got:\n%s", want, fileContent)
		}
	}
}

func TestOrgSpawn_RoleTemplate_PromptFlagAppendedAfterTemplate(t *testing.T) {
	// When --role has a template AND --prompt is also given, the template
	// comes first and --prompt is appended after a blank line -- in the
	// prompt file's content, since the combined prompt is multi-line.
	o, h, _ := testOrg(t)

	p := mustSpawnParams("org-a", "seat-1")
	p.Role = "reviewer"
	p.Prompt = "focus on the protocol package first"
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	// index 4: [--permission-mode bypassPermissions --model <model> <pointer>].
	promptArg := h.agentStartArgs[0][4]
	promptPath := strings.TrimPrefix(promptArg, "役割指示を読み込んで従ってください: ")
	data, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("expected the prompt file to exist at %q: %v", promptPath, err)
	}
	fileContent := string(data)
	if !strings.Contains(fileContent, "run-static-verify.sh") {
		t.Fatalf("expected the reviewer template body in the prompt file, got:\n%s", fileContent)
	}
	if !strings.HasSuffix(fileContent, p.Prompt) {
		t.Fatalf("expected --prompt appended at the end of the prompt file, got:\n%s", fileContent)
	}
}

func TestOrgSpawn_Respawn_OverwritesPromptFile(t *testing.T) {
	// A respawn of the same org_id/seat_id after a prior terminal (failed)
	// attempt must overwrite the existing prompt file with the new render,
	// not fail because the file already exists and not leave the previous
	// attempt's content lingering behind.
	o, _, a := testOrg(t)

	// First attempt: reaches agent_start (which writes the prompt file)
	// but then fails at the agmsg_join step -- a terminal spawn_failed, not
	// a stale in-flight saga, so the next Spawn call runs a full fresh
	// attempt rather than compensate-and-retry.
	a.joinErrs = map[string]error{"seat-1": errors.New("stub failure: agmsg join")}
	p1 := mustSpawnParams("org-a", "seat-1")
	p1.Role = "reviewer"
	p1.Scope = "internal/org/**"
	if r := o.Spawn(p1); r.Outcome != SpawnOutcomeFailed {
		t.Fatalf("expected first attempt to fail at agmsg_join, got %+v", r)
	}

	promptPath, perr := o.promptFilePath("org-a", "seat-1")
	if perr != nil {
		t.Fatalf("promptFilePath: %v", perr)
	}
	firstContent, rerr := os.ReadFile(promptPath)
	if rerr != nil {
		t.Fatalf("expected prompt file written by the first attempt: %v", rerr)
	}
	if !strings.Contains(string(firstContent), "internal/org/**") {
		t.Fatalf("expected the first attempt's scope in the prompt file, got:\n%s", firstContent)
	}

	// Respawn: same org_id/seat_id, different scope, no injected failure.
	a.joinErrs = nil
	p2 := mustSpawnParams("org-a", "seat-1")
	p2.Role = "reviewer"
	p2.Scope = "internal/cli/**"
	if r := o.Spawn(p2); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected the respawn to succeed, got %+v", r)
	}

	secondContent, rerr := os.ReadFile(promptPath)
	if rerr != nil {
		t.Fatalf("expected the prompt file to still exist after the respawn: %v", rerr)
	}
	if strings.Contains(string(secondContent), "internal/org/**") {
		t.Fatalf("expected the respawn to overwrite the previous attempt's content, still found the stale scope:\n%s", secondContent)
	}
	if !strings.Contains(string(secondContent), "internal/cli/**") {
		t.Fatalf("expected the respawn's own scope in the overwritten prompt file, got:\n%s", secondContent)
	}
}

func TestOrgSpawn_PromptFileWriteFailure_FailsStepPromptFile(t *testing.T) {
	// AC-4 deviation: a write failure for the prompt file must fail the
	// saga at a dedicated "prompt_file" step, before AgentStart is ever
	// called, with the same best-effort pane compensation as any other
	// post-tab_create failure step.
	o, h, _ := testOrg(t)
	stateDir := filepath.Dir(o.Manifest.Path())

	// Make the prompts directory unusable: a regular file sits at the exact
	// path writePromptFile needs to mkdir -p, so os.MkdirAll fails.
	if err := os.WriteFile(filepath.Join(stateDir, "prompts"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("seed prompts-as-file: %v", err)
	}

	p := mustSpawnParams("org-a", "seat-1")
	p.Role = "reviewer"
	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeFailed {
		t.Fatalf("expected SpawnOutcomeFailed when the prompt file cannot be written, got %+v", result)
	}
	if len(h.agentStartArgs) != 0 {
		t.Fatalf("expected AgentStart never called when the prompt file write fails first, got %v", h.agentStartArgs)
	}
	if len(h.sendKeysCalls) != 1 || h.sendKeysCalls[0] != "pane-1" {
		t.Fatalf("expected a compensation C-c to the already-created pane, got %v", h.sendKeysCalls)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	if last.Event != EventSpawnFailed {
		t.Fatalf("expected last event to be spawn_failed, got %q", last.Event)
	}
	assertDetailsContains(t, last.Details, "step=prompt_file")
}

func TestOrgSpawn_DryRun_RoleTemplate_RecordsPromptFilePathWithoutWriting(t *testing.T) {
	// AC-4 deviation: dry-run must record the would-be prompt file path on
	// the agent_started step's Details, matching what a real spawn would
	// produce for the same params, but must never actually write the file
	// (AC-8: dry-run has zero side effects).
	o, h, a := testOrg(t)

	p := mustSpawnParams("org-a", "seat-1")
	p.Role = "reviewer"
	p.Scope = "internal/org/**"
	p.DryRun = true
	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected dry-run spawn to succeed, got %+v", result)
	}
	if len(h.calls) != 0 || len(a.calls) != 0 {
		t.Fatalf("expected zero driver calls for --dry-run, got herdr=%v agmsg=%v", h.calls, a.calls)
	}

	promptPath, perr := o.promptFilePath("org-a", "seat-1")
	if perr != nil {
		t.Fatalf("promptFilePath: %v", perr)
	}
	if _, err := os.Stat(promptPath); !os.IsNotExist(err) {
		t.Fatalf("expected dry-run to NOT write the prompt file, got stat err=%v", err)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var agentStartedEvent *ManifestEvent
	for i := range rr.Events {
		if rr.Events[i].Event == EventSpawnStep && strings.HasPrefix(rr.Events[i].Details, "agent_started") {
			agentStartedEvent = &rr.Events[i]
			break
		}
	}
	if agentStartedEvent == nil {
		t.Fatalf("expected an agent_started spawn_step event in the dry-run trail, got %+v", rr.Events)
	}
	if !strings.Contains(agentStartedEvent.Details, "prompt_file="+promptPath) {
		t.Fatalf("expected the dry-run agent_started step to record the would-be prompt_file path, got %q", agentStartedEvent.Details)
	}
}

func TestOrgSpawn_UnknownRole_PromptFlagOnly_NoTemplateNoError(t *testing.T) {
	// AC-4: an unknown role has no embedded template -- the saga must
	// proceed with --prompt verbatim (or empty), never erroring.
	o, h, _ := testOrg(t)

	p := mustSpawnParams("org-a", "seat-1")
	p.Role = "not-a-known-role"
	p.Prompt = "verbatim prompt"
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected spawn to succeed for an unknown role, got %+v", r)
	}

	args := h.agentStartArgs[0]
	// [--permission-mode bypassPermissions --model <model> verbatim prompt].
	if len(args) != 5 || args[4] != "verbatim prompt" {
		t.Fatalf("expected AgentStart args [--permission-mode bypassPermissions --model <model> verbatim prompt], got %v", args)
	}
}

func TestOrgSpawn_UnknownRole_NoPromptFlag_NoPromptArgAtAll(t *testing.T) {
	// Unchanged-behavior guard: an unknown role with no --prompt at all
	// must still omit the trailing agentArgs element entirely (matching
	// pre-role-template behavior), not pass an empty string argument.
	o, h, _ := testOrg(t)

	p := mustSpawnParams("org-a", "seat-1")
	p.Role = "not-a-known-role"
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected spawn to succeed, got %+v", r)
	}

	args := h.agentStartArgs[0]
	// [--permission-mode bypassPermissions --model <model>] with no prompt element.
	if len(args) != 4 {
		t.Fatalf("expected AgentStart args [--permission-mode bypassPermissions --model <model>] with no prompt element, got %v", args)
	}
}

func TestOrgSpawn_ScopeRecordedOnSpawnedEventDetails(t *testing.T) {
	o, _, _ := testOrg(t)

	p := mustSpawnParams("org-a", "seat-1")
	p.Scope = "internal/org/**"
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	if last.Event != EventSpawned {
		t.Fatalf("expected last event spawned, got %q", last.Event)
	}
	assertDetailsContains(t, last.Details, "scope=internal/org/**")
}

func TestOrgSpawn_NoScope_SpawnedEventDetailsOmitsScopeFragment(t *testing.T) {
	// Behavior guard: a spawn with no Scope leaves no "scope=" fragment in
	// Details. Unlike Scope, permission_mode is unconditionally recorded on
	// every spawned event (AC-2b audit requirement), so Details is no
	// longer empty by default -- an autonomous-mode spawn with no Scope
	// must also set AllowUnscoped to get past the minimum control gate
	// (AC-2b), and that bypass itself is recorded too.
	o, _, _ := testOrg(t)

	p := mustSpawnParams("org-a", "seat-1")
	p.Scope = ""
	p.AllowUnscoped = true
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	if strings.Contains(last.Details, "scope=") {
		t.Fatalf("expected no scope= fragment in Details with no Scope, got %q", last.Details)
	}
	assertDetailsContains(t, last.Details, "permission_mode=autonomous", "allow_unscoped=true")
}

// agentPaneBusyErr builds a fake AgentStart error carrying the literal
// "agent_pane_busy" marker agentStartWithRetry matches on, shaped like the
// real herdr adapter's error text (see docs/evidence/
// org-seats-smoke-2026-08-02.log).
func agentPaneBusyErr() error {
	return errors.New(`herdr: exit status 1: {"error":{"code":"agent_pane_busy","message":"agent target pane w5:p2 is not an available shell"}} (herdr: agent_pane_busy: agent target pane w5:p2 is not an available shell)`)
}

func TestOrgSpawn_AgentStart_RetriesOnAgentPaneBusyThenSucceeds(t *testing.T) {
	// Third real-herdr smoke deviation (see plan
	// docs/plans/active/2026-08-02-org-runtime-seats.md, "Implementation
	// notes (deviations)"): a freshly created tab's pane rejects `agent
	// start` with agent_pane_busy for ~1-3s while its shell initializes.
	// agentStartWithRetry must retry only on that specific error and
	// eventually succeed once the fake reports the pane ready.
	o, h, _ := testOrg(t)
	o.AgentStartRetryInterval = time.Millisecond // keep the test fast
	h.agentStartErrs = []error{agentPaneBusyErr(), agentPaneBusyErr(), nil}

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected SpawnOutcomeSpawned after busy retries resolve, got %v (err=%v)", result.Outcome, result.Err)
	}
	if len(h.agentStartNames) != 3 {
		t.Fatalf("expected exactly 3 AgentStart calls (2 busy + 1 success), got %d: %v", len(h.agentStartNames), h.agentStartNames)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var agentStartedDetails string
	for _, ev := range rr.Events {
		if ev.Event == EventSpawnStep && strings.HasPrefix(ev.Details, "agent_started") {
			agentStartedDetails = ev.Details
		}
	}
	assertDetailsContains(t, agentStartedDetails, "agent_start_retries=2")
}

func TestOrgSpawn_AgentStart_NonBusyError_FailsImmediatelyWithoutRetry(t *testing.T) {
	// Any AgentStart error other than agent_pane_busy must fail the saga on
	// the first attempt, exactly as before this retry was added -- the
	// retry is scoped narrowly to the one known-transient herdr error.
	o, h, _ := testOrg(t)
	o.AgentStartRetryInterval = time.Millisecond
	h.agentStartErr = errors.New("stub failure: agent start rejected for an unrelated reason")

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeFailed {
		t.Fatalf("expected SpawnOutcomeFailed, got %v", result.Outcome)
	}
	if len(h.agentStartNames) != 1 {
		t.Fatalf("expected exactly 1 AgentStart call (no retry on non-busy error), got %d: %v", len(h.agentStartNames), h.agentStartNames)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	assertDetailsContains(t, last.Details, "step=agent_start")
}

func TestOrgSpawn_AgentStart_AlwaysBusy_BoundedByCtxDeadline(t *testing.T) {
	// AC guard: if the pane never becomes ready, the retry loop must not
	// hang -- it is bounded by the saga's own ctx deadline (SpawnParams.
	// TimeoutMS), so the saga always terminates and reports agent_start as
	// the failing step.
	o, h, _ := testOrg(t)
	o.AgentStartRetryInterval = 5 * time.Millisecond
	h.agentStartErr = agentPaneBusyErr()

	p := mustSpawnParams("org-a", "seat-1")
	p.TimeoutMS = 30 // short deadline so the retry loop's ctx.Done() fires quickly

	done := make(chan SpawnResult, 1)
	go func() { done <- o.Spawn(p) }()

	select {
	case result := <-done:
		if result.Outcome != SpawnOutcomeFailed {
			t.Fatalf("expected SpawnOutcomeFailed once the ctx deadline is exceeded, got %v", result.Outcome)
		}
		if len(h.agentStartNames) == 0 {
			t.Fatalf("expected at least one AgentStart attempt before the ctx deadline")
		}
		if len(h.agentStartNames) >= maxAgentStartAttempts {
			t.Fatalf("expected the ctx deadline to cut the retry loop short, well under maxAgentStartAttempts=%d, got %d", maxAgentStartAttempts, len(h.agentStartNames))
		}

		rr, err := o.Manifest.Read()
		if err != nil {
			t.Fatalf("read manifest: %v", err)
		}
		last := rr.Events[len(rr.Events)-1]
		if last.Event != EventSpawnFailed {
			t.Fatalf("expected last event to be spawn_failed, got %q", last.Event)
		}
		assertDetailsContains(t, last.Details, "step=agent_start")
	case <-time.After(5 * time.Second):
		t.Fatal("Spawn did not return within 5s -- agentStartWithRetry is not bounded by ctx deadline")
	}
}

// --- AC-6: announce (HELLO Send) failure Leave compensation -----------------

func TestOrgSpawn_FailureInjection_AgmsgSend_LeavesJoinedSeat(t *testing.T) {
	// tech-debt fix (docs/tech-debt/README.md, "spawn の agmsg_announce
	// (HELLO send)失敗パスの補償..."): by the time HELLO Send fails, the
	// seat's own Join already succeeded, so the failure path must
	// best-effort Leave the seat back out of the roster and record the
	// outcome in Details.
	o, _, a := testOrg(t)
	a.sendErr = errors.New("stub failure: agmsg send")

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeFailed {
		t.Fatalf("expected SpawnOutcomeFailed, got %v", result.Outcome)
	}

	if len(a.leaveCalls) != 1 {
		t.Fatalf("expected exactly 1 Leave call, got %+v", a.leaveCalls)
	}
	wantTeam := agmsgTeam("org-a")
	if a.leaveCalls[0].team != wantTeam || a.leaveCalls[0].agentID != "seat-1" {
		t.Fatalf("expected Leave(%q, seat-1), got %+v", wantTeam, a.leaveCalls[0])
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	if last.Event != EventSpawnFailed {
		t.Fatalf("expected last event to be spawn_failed, got %q", last.Event)
	}
	assertDetailsContains(t, last.Details, "step=agmsg_announce", "leave=ok")
}

func TestOrgSpawn_FailureInjection_AgmsgSend_LeaveFailure_RecordedInDetails(t *testing.T) {
	// The Leave compensation is itself best-effort: a Leave failure must be
	// recorded in Details, not propagated as a second saga failure or
	// silently dropped.
	o, _, a := testOrg(t)
	a.sendErr = errors.New("stub failure: agmsg send")
	a.leaveErr = errors.New("stub failure: agmsg leave")

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeFailed {
		t.Fatalf("expected SpawnOutcomeFailed, got %v", result.Outcome)
	}
	if len(a.leaveCalls) != 1 {
		t.Fatalf("expected the Leave call to still be attempted despite it failing, got %+v", a.leaveCalls)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	assertDetailsContains(t, last.Details, "step=agmsg_announce", "leave=failed:", "stub failure: agmsg leave")
}

// --- AC-7: LeaderIdentity const + leader agmsg type from LeaderDriver -------------

func TestLeaderIdentity_ConstantValue(t *testing.T) {
	if LeaderIdentity != "leader" {
		t.Fatalf("LeaderIdentity = %q, want %q", LeaderIdentity, "leader")
	}
}

func TestOrgSpawn_EnsureLeaderJoined_DefaultLeaderDriver_ClaudeCodeType(t *testing.T) {
	o, _, a := testOrg(t)

	p := mustSpawnParams("org-a", "seat-1")
	// LeaderDriver left unset -- must default to "claude" -> "claude-code".
	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected SpawnOutcomeSpawned, got %v (err=%v)", result.Outcome, result.Err)
	}
	if len(a.joinCalls) < 1 {
		t.Fatalf("expected at least 1 Join call, got %+v", a.joinCalls)
	}
	leaderJoin := a.joinCalls[0]
	if leaderJoin.agentID != LeaderIdentity || leaderJoin.agmsgType != "claude-code" {
		t.Fatalf("expected leader Join(%s, claude-code, ...) by default, got %+v", LeaderIdentity, leaderJoin)
	}
}

func TestOrgSpawn_EnsureLeaderJoined_LeaderDriverCodex_UsesCodexAgmsgType(t *testing.T) {
	// The leader identity's own driver (LeaderDriver) is independent of the
	// seat's Driver: a claude-driven seat spawned under a codex-coordinated
	// org must still register "leader" with agmsg type "codex", not
	// "claude-code".
	o, _, a := testOrg(t)

	p := mustSpawnParams("org-a", "seat-1")
	p.LeaderDriver = "codex"
	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected SpawnOutcomeSpawned, got %v (err=%v)", result.Outcome, result.Err)
	}
	if len(a.joinCalls) != 2 {
		t.Fatalf("expected 2 Join calls (leader then seat), got %+v", a.joinCalls)
	}
	leaderJoin, seatJoin := a.joinCalls[0], a.joinCalls[1]
	if leaderJoin.agentID != LeaderIdentity || leaderJoin.agmsgType != "codex" {
		t.Fatalf("expected leader Join(%s, codex, ...) for LeaderDriver=codex, got %+v", LeaderIdentity, leaderJoin)
	}
	if seatJoin.agentID != "seat-1" || seatJoin.agmsgType != "claude-code" {
		t.Fatalf("expected the seat's own Join to still use its own Driver (claude -> claude-code), got %+v", seatJoin)
	}
}

// --- AC-3: `ralph org start` = leader-seat spawn sugar (SeatID == LeaderIdentity) ---

func TestOrgSpawn_LeaderSelfSpawn_SingleAgmsgJoin_NoHelloSend(t *testing.T) {
	// SeatID == LeaderIdentity ("ralph org start") must not double-join or
	// HELLO-announce: the seat's own Join call IS the leader-identity join, and
	// a HELLO from leader to leader would violate the star topology's
	// single-coordinator premise (see the leaderSelfSpawn doc comment in
	// spawn.go's Spawn).
	o, _, a := testOrg(t)

	p := mustSpawnParams("org-a", LeaderIdentity)
	p.Role = LeaderIdentity
	p.Task = "dry-run 座席を spawn し、送信・確認・disband まで行え"
	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected SpawnOutcomeSpawned, got %v (err=%v)", result.Outcome, result.Err)
	}

	if len(a.joinCalls) != 1 {
		t.Fatalf("expected exactly 1 agmsg Join call for a leader-self spawn (no separate ensureLeaderJoined), got %+v", a.joinCalls)
	}
	if a.joinCalls[0].agentID != LeaderIdentity {
		t.Fatalf("expected the single Join call to register %q, got %+v", LeaderIdentity, a.joinCalls[0])
	}
	for _, c := range a.calls {
		if c == "send" {
			t.Fatalf("expected no agmsg Send (HELLO) call for a leader-self spawn, got calls=%v", a.calls)
		}
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var joinedStep *ManifestEvent
	for i := range rr.Events {
		if strings.HasPrefix(rr.Events[i].Details, "agmsg_joined") {
			joinedStep = &rr.Events[i]
			break
		}
	}
	if joinedStep == nil {
		t.Fatalf("expected an agmsg_joined spawn_step event, got events %+v", rr.Events)
	}
	assertDetailsContains(t, joinedStep.Details, "leader_self=true")
	for _, ev := range rr.Events {
		if strings.HasPrefix(ev.Details, "agmsg_leader_joined") || ev.Details == "agmsg_announced" {
			t.Fatalf("expected no agmsg_leader_joined/agmsg_announced step for a leader-self spawn, got %+v", ev)
		}
	}
}

func TestOrgSpawn_LeaderSelfSpawn_DryRun_MirrorsSameSkip(t *testing.T) {
	o, _, _ := testOrg(t)

	p := mustSpawnParams("org-a", LeaderIdentity)
	p.Role = LeaderIdentity
	p.DryRun = true
	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected SpawnOutcomeSpawned for a valid dry-run, got %v (err=%v)", result.Outcome, result.Err)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	sawJoinedLeaderSelf := false
	for _, ev := range rr.Events {
		if strings.HasPrefix(ev.Details, "agmsg_leader_joined") || ev.Details == "agmsg_announced" {
			t.Fatalf("expected no agmsg_leader_joined/agmsg_announced step in the dry-run trail for a leader-self spawn, got %+v", ev)
		}
		if ev.Details == "agmsg_joined leader_self=true" {
			sawJoinedLeaderSelf = true
		}
	}
	if !sawJoinedLeaderSelf {
		t.Fatalf("expected an 'agmsg_joined leader_self=true' step in the dry-run trail, got events %+v", rr.Events)
	}
}

func TestOrgSpawn_LeaderRole_TaskAndEnvelopeSubstitutedIntoPromptFile(t *testing.T) {
	// `ralph org start`'s Task and the org's EnvelopeSummary must both land
	// in the leader seat's rendered prompt file (the leader.md template is long
	// enough to always need the prompt-file path, same as reviewer).
	o, h, _ := testOrg(t)

	p := mustSpawnParams("org-a", LeaderIdentity)
	p.Role = LeaderIdentity
	p.Task = "dry-run 座席を1つ spawn し、typed message を送り、status を確認して disband せよ"
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	if len(h.agentStartArgs) != 1 {
		t.Fatalf("expected exactly 1 AgentStart call, got %d", len(h.agentStartArgs))
	}
	promptArg := h.agentStartArgs[0][len(h.agentStartArgs[0])-1]
	promptPath := strings.TrimPrefix(promptArg, "役割指示を読み込んで従ってください: ")
	data, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("expected the prompt file to exist at %q: %v", promptPath, err)
	}
	fileContent := string(data)
	for _, want := range []string{p.Task, "model_pool:", "max_seats:", "permission default:"} {
		if !strings.Contains(fileContent, want) {
			t.Errorf("expected the leader prompt file content to contain %q, got:\n%s", want, fileContent)
		}
	}
}

// --- AC-9: concurrent spawn TOCTOU (manifest flock) -------------------------

func TestOrgSpawn_ConcurrentSpawns_MaxSeatsNeverExceeded(t *testing.T) {
	// Regression for the tech-debt TOCTOU race (docs/tech-debt/README.md,
	// "max_seats is enforced across an unlocked read-then-append window"):
	// N goroutines spawn N distinct seats against the same org concurrently.
	// testOrgConfig's MaxSeats=3 must never be exceeded, even though every
	// goroutine races to read-validate-append on the same manifest file.
	o, _, _ := testOrg(t)
	const n = 10

	var wg sync.WaitGroup
	wg.Add(n)
	results := make([]SpawnResult, n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			results[i] = o.Spawn(mustSpawnParams("org-a", fmt.Sprintf("seat-%d", i)))
		}(i)
	}
	wg.Wait()

	spawned := 0
	for _, r := range results {
		if r.Outcome == SpawnOutcomeSpawned {
			spawned++
		} else if r.Outcome != SpawnOutcomeRejected {
			t.Errorf("unexpected outcome %v (err=%v)", r.Outcome, r.Err)
		}
	}
	if spawned != testOrgConfig().MaxSeats {
		t.Fatalf("expected exactly max_seats=%d successful spawns out of %d concurrent attempts, got %d",
			testOrgConfig().MaxSeats, n, spawned)
	}

	active, err := o.Manifest.ActiveSeatCount("org-a", RosterOptions{})
	if err != nil {
		t.Fatalf("ActiveSeatCount: %v", err)
	}
	if active > testOrgConfig().MaxSeats {
		t.Fatalf("manifest reports %d active seats, exceeding max_seats=%d", active, testOrgConfig().MaxSeats)
	}
}

func TestWithManifestLock_SerializesConcurrentCallers(t *testing.T) {
	// Unit-level pin for lockfile.go's own contract, independent of Spawn:
	// two withManifestLock calls racing on the same dir must never run fn
	// concurrently.
	dir := t.TempDir()
	const n = 20
	var (
		mu        sync.Mutex
		inside    int
		maxInside int
	)

	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			err := withManifestLock(dir, func() error {
				mu.Lock()
				inside++
				if inside > maxInside {
					maxInside = inside
				}
				mu.Unlock()

				time.Sleep(time.Millisecond)

				mu.Lock()
				inside--
				mu.Unlock()
				return nil
			})
			if err != nil {
				t.Errorf("withManifestLock: %v", err)
			}
		}()
	}
	wg.Wait()

	if maxInside != 1 {
		t.Fatalf("expected at most 1 concurrent holder of the manifest lock, observed max concurrency %d", maxInside)
	}
}

// --- AC-10b: dryRunSpawn error propagation ----------------------------------

func TestOrgSpawn_DryRun_PromptFilePathError_FailsStep(t *testing.T) {
	// tech-debt fix (docs/tech-debt/README.md, "dryRunSpawn silently
	// swallows RenderRolePrompt's and promptFilePath's errors"): a
	// promptFilePath failure must fail the dry run the same way it would
	// fail a real spawn at the "prompt_file" step, not silently report
	// success.
	o, _, _ := testOrg(t)
	orig := absPath
	absPath = func(string) (string, error) { return "", errors.New("stub: abs failed") }
	defer func() { absPath = orig }()

	p := mustSpawnParams("org-a", "seat-1")
	p.Role = "reviewer" // reviewer.md is multi-line, so needsPromptFile triggers.
	p.DryRun = true
	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeFailed {
		t.Fatalf("expected dry-run spawn to fail when promptFilePath errors, got %+v", result)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	if last.Event != EventSpawnFailed {
		t.Fatalf("expected last event to be spawn_failed, got %q", last.Event)
	}
	if !last.DryRun {
		t.Fatalf("expected the dry-run failure event to carry dry_run: true, got %+v", last)
	}
	assertDetailsContains(t, last.Details, "step=prompt_file", "stub: abs failed")
}

func TestOrgSpawn_DryRun_PromptFilePathError_NoManifestEventForRealSeat(t *testing.T) {
	// The dry-run failure event must be excluded from real-seat accounting
	// (Roster with the default RosterOptions{}), exactly like every other
	// dry-run event -- confirms DryRun: true actually took effect, not just
	// that the field was set on the struct literal.
	o, _, _ := testOrg(t)
	orig := absPath
	absPath = func(string) (string, error) { return "", errors.New("stub: abs failed") }
	defer func() { absPath = orig }()

	p := mustSpawnParams("org-a", "seat-1")
	p.Role = "reviewer"
	p.DryRun = true
	if result := o.Spawn(p); result.Outcome != SpawnOutcomeFailed {
		t.Fatalf("expected SpawnOutcomeFailed, got %+v", result)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if roster := Roster(rr.Events, RosterOptions{}); len(roster) != 0 {
		t.Fatalf("expected the dry-run failure to be excluded from the default (real-seat) roster, got %+v", roster)
	}
}

// --- AC-2/AC-2b: permission-mode envelope -----------------------------------

// TestOrgSpawn_PermissionMode_Claude_ArgvMapping pins AC-2's argv contract
// for all three modes: permission args are prepended to AgentStart's
// agentArgs, before --model.
func TestOrgSpawn_PermissionMode_Claude_ArgvMapping(t *testing.T) {
	cases := []struct {
		mode     string
		wantArgs []string // nil for guarded (no flag)
	}{
		{"autonomous", []string{"--permission-mode", "bypassPermissions"}},
		{"edits", []string{"--permission-mode", "acceptEdits"}},
		{"guarded", nil},
	}
	for _, tc := range cases {
		t.Run(tc.mode, func(t *testing.T) {
			o, h, _ := testOrg(t)
			o.Config.Permissions = config.OrgPermissionsConfig{Default: tc.mode}

			p := mustSpawnParams("org-a", "seat-1")
			result := o.Spawn(p)
			if result.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("expected spawn to succeed under mode %q, got %+v", tc.mode, result)
			}

			args := h.agentStartArgs[0]
			wantLen := len(tc.wantArgs) + 2 // + "--model" <model>
			if len(args) != wantLen {
				t.Fatalf("mode %q: expected %d AgentStart args, got %d (%v)", tc.mode, wantLen, len(args), args)
			}
			for i, want := range tc.wantArgs {
				if args[i] != want {
					t.Fatalf("mode %q: args[%d] = %q, want %q (full args: %v)", tc.mode, i, args[i], want, args)
				}
			}
			if args[len(tc.wantArgs)] != "--model" {
				t.Fatalf("mode %q: expected --model to immediately follow permission args, got %v", tc.mode, args)
			}

			rr, err := o.Manifest.Read()
			if err != nil {
				t.Fatalf("read manifest: %v", err)
			}
			last := rr.Events[len(rr.Events)-1]
			assertDetailsContains(t, last.Details, "permission_mode="+tc.mode)
		})
	}
}

// TestOrgSpawn_MinimumControlGate_Autonomous_EmptyScope_RejectedWithEvent
// covers AC-2b: an autonomous-mode spawn with an empty Scope and no
// AllowUnscoped is rejected through the same reject() path as every other
// envelope-validation rejection (self-review LOW finding) -- a `rejected`
// manifest event plus an honored=false receipt, but still zero driver calls
// and no spawn_started (reject() never appends one), so the gate's
// fail-closed/no-saga-side-effects guarantee is unchanged; only the audit
// trail is no longer silent.
func TestOrgSpawn_MinimumControlGate_Autonomous_EmptyScope_RejectedWithEvent(t *testing.T) {
	o, h, a := testOrg(t)

	p := mustSpawnParams("org-a", "seat-1")
	p.Scope = ""
	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeRejected {
		t.Fatalf("expected SpawnOutcomeRejected, got %+v", result)
	}
	if result.Err == nil || !strings.Contains(result.Err.Error(), "--scope") || !strings.Contains(result.Err.Error(), "--reserve") ||
		!strings.Contains(result.Err.Error(), "--allow-unscoped") {
		t.Fatalf("expected error naming --scope, --reserve and --allow-unscoped, got %v", result.Err)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if len(rr.Events) != 1 || rr.Events[0].Event != EventRejected {
		t.Fatalf("expected exactly one rejected manifest event for the gate rejection, got %+v", rr.Events)
	}
	if len(h.calls) != 0 || len(a.calls) != 0 {
		t.Fatalf("expected zero driver calls for the gate rejection, got herdr=%v agmsg=%v", h.calls, a.calls)
	}

	receiptRR, err := o.Receipts.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	if len(receiptRR.Receipts) != 1 || receiptRR.Receipts[0].Honored != HonoredFalse {
		t.Fatalf("expected exactly one honored=false receipt for the gate rejection, got %+v", receiptRR.Receipts)
	}
}

// TestOrgSpawn_MinimumControlGate_EditsAndGuarded_EmptyScope_Proceeds
// verifies the gate only applies to autonomous mode: edits/guarded seats
// with an empty Scope spawn normally.
func TestOrgSpawn_MinimumControlGate_EditsAndGuarded_EmptyScope_Proceeds(t *testing.T) {
	for _, mode := range []string{"edits", "guarded"} {
		t.Run(mode, func(t *testing.T) {
			o, _, _ := testOrg(t)
			o.Config.Permissions = config.OrgPermissionsConfig{Default: mode}

			p := mustSpawnParams("org-a", "seat-1")
			p.Scope = ""
			result := o.Spawn(p)
			if result.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("expected spawn to succeed under mode %q with empty scope, got %+v", mode, result)
			}
		})
	}
}

// TestOrgSpawn_MinimumControlGate_DryRun_AppliesSameGate verifies the
// validate-then-record contract: a dry-run autonomous spawn with an empty
// Scope is rejected the same way a real spawn would be, recording a
// DryRun:true rejected event via reject() (self-review LOW finding) so it
// stays excluded from ActiveSeatCount/roster like every other dry-run event.
func TestOrgSpawn_MinimumControlGate_DryRun_AppliesSameGate(t *testing.T) {
	o, _, _ := testOrg(t)

	p := mustSpawnParams("org-a", "seat-1")
	p.Scope = ""
	p.DryRun = true
	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeRejected {
		t.Fatalf("expected SpawnOutcomeRejected for a dry-run gate violation, got %+v", result)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if len(rr.Events) != 1 || rr.Events[0].Event != EventRejected || !rr.Events[0].DryRun {
		t.Fatalf("expected exactly one dry-run rejected manifest event for the dry-run gate rejection, got %+v", rr.Events)
	}
}

// TestOrgSpawn_Codex_AutonomousDefault_RejectedFailClosed_WithReceipt is the
// AC-2 fail-closed end-to-end test: a codex seat under the default
// (autonomous) config is rejected with a `rejected` manifest event and an
// honored=false receipt -- unlike the AC-2b gate rejection above, this is a
// permissionArgsForDriver error, reached only after the gate itself passes
// (mustSpawnParams sets a non-empty Scope by default).
func TestOrgSpawn_Codex_AutonomousDefault_RejectedFailClosed_WithReceipt(t *testing.T) {
	o, h, a := testOrg(t)

	p := mustSpawnParams("org-a", "seat-1")
	p.Driver = "codex"
	p.Model = "gpt-5-codex"
	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeRejected {
		t.Fatalf("expected SpawnOutcomeRejected, got %+v", result)
	}
	if result.Err == nil || !strings.Contains(result.Err.Error(), "codex seat permission mode") {
		t.Fatalf("expected fail-closed codex error, got %v", result.Err)
	}
	if len(h.calls) != 0 || len(a.calls) != 0 {
		t.Fatalf("expected zero driver calls for the fail-closed rejection, got herdr=%v agmsg=%v", h.calls, a.calls)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	if last.Event != EventRejected {
		t.Fatalf("expected last event rejected, got %q", last.Event)
	}

	receiptRR, err := o.Receipts.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	if len(receiptRR.Receipts) != 1 || receiptRR.Receipts[0].Honored != HonoredFalse {
		t.Fatalf("expected 1 receipt honored=false, got %+v", receiptRR.Receipts)
	}
	// reject() sets SpawnResult.ModelReceipt to the same receipt it
	// appended, not the zero value.
	if result.ModelReceipt != receiptRR.Receipts[0] {
		t.Fatalf("expected SpawnResult.ModelReceipt to match the persisted rejection receipt, got %+v vs %+v", result.ModelReceipt, receiptRR.Receipts[0])
	}
	if result.ModelReceipt.ReportedEffectiveModel != "" {
		t.Fatalf("expected the rejection receipt to carry no reported model, got %+v", result.ModelReceipt)
	}
}

// TestOrgSpawn_Reject_ReceiptsAppendFailureLeavesModelReceiptZero covers
// cycle-2 self-review C2-3
// (docs/reports/self-review-2026-09-20-codex-effective-model-receipt.md):
// reject() must not claim ModelReceipt for a receipt that was never
// actually persisted, mirroring how Stop already
// guards its own ModelReceipt (verbs.go). Uses the same read-only-file
// technique as TestOrgSend_AppendEventFailsAfterEnter_ReportsEnterPressedButUnrecorded
// (verbs_test.go) and TestOrgStop_Codex_ReceiptsAppendFailureLeavesStopSuccessful.
func TestOrgSpawn_Reject_ReceiptsAppendFailureLeavesModelReceiptZero(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignores a read-only file's permission bit")
	}
	o, _, _ := testOrg(t)

	// A prior successful spawn creates the receipts file so it exists to
	// chmod below.
	if r := o.Spawn(mustSpawnParams("org-a", "seat-warmup")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("warmup spawn failed: %+v", r)
	}

	receiptsPath := o.Receipts.Path()
	if err := os.Chmod(receiptsPath, 0o444); err != nil {
		t.Fatalf("chmod receipts read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(receiptsPath, 0o644) })

	p := mustSpawnParams("org-a", "seat-1")
	p.Driver = "codex"
	p.Model = "gpt-5-codex" // fail-closed under the default autonomous mode, routing through reject()

	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeRejected {
		t.Fatalf("expected SpawnOutcomeRejected, got %+v", result)
	}
	if result.ModelReceipt != (Receipt{}) {
		t.Fatalf("expected zero ModelReceipt when the receipts append failed, got %+v", result.ModelReceipt)
	}
}

// TestOrgSpawn_Codex_GuardedRoleOverride_Spawns is the positive fail-closed
// counterpart: a codex seat whose role is explicitly overridden to guarded
// spawns normally, with no permission flags in AgentStart's argv.
func TestOrgSpawn_Codex_GuardedRoleOverride_Spawns(t *testing.T) {
	o, h, _ := testOrg(t)
	o.Config.Permissions = config.OrgPermissionsConfig{
		Default: "autonomous",
		Roles:   map[string]string{"worker": "guarded"},
	}

	p := mustSpawnParams("org-a", "seat-1") // Role: "worker" (see mustSpawnParams)
	p.Driver = "codex"
	p.Model = "gpt-5-codex"
	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected spawn to succeed for a guarded codex seat, got %+v", result)
	}

	args := h.agentStartArgs[0]
	if len(args) != 2 || args[0] != "--model" {
		t.Fatalf("expected no permission flags for guarded mode, got %v", args)
	}
}

// TestOrgSpawn_Codex_WorkspaceWrite_AddDirWhenStateDirOutsideCwd pins AC8
// (plan 2026-10-07-org-state-dir-common) at the AgentStart argv: once
// codex_verified unlocks workspace-write, a codex leader seat whose --cwd is
// a linked worktree gets --add-dir <state dir> right after the sandbox flags
// and before --model, so it can append to the shared ledger under the main
// worktree; a leader seat whose --cwd is the main worktree (state dir
// inside) and an implementer seat in the same linked worktree get the
// sandbox flags alone.
func TestOrgSpawn_Codex_WorkspaceWrite_AddDirWhenStateDirOutsideCwd(t *testing.T) {
	cases := []struct {
		name      string
		role      string
		linked    bool
		wantFlags bool
	}{
		{"leader/cwd is a linked worktree", LeaderIdentity, true, true},
		{"leader/cwd is the main worktree", LeaderIdentity, false, false},
		{"implementer/cwd is a linked worktree", "implementer", true, false},
	}
	for _, mode := range []string{PermissionModeAutonomous, PermissionModeEdits} {
		sandbox := codexEditsArgs
		if mode == PermissionModeAutonomous {
			sandbox = codexAutonomousArgs
		}
		for _, tc := range cases {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				// Each subtest gets its own tree and ledger, so a second
				// spawn of the same seat is not an idempotent no-op.
				root := t.TempDir()
				mainWT := filepath.Join(root, "main")
				linkedWT := filepath.Join(mainWT, ".claude", "worktrees", "slug")
				stateDir := filepath.Join(mainWT, ".harness", "state", "org")
				if err := os.MkdirAll(linkedWT, 0o755); err != nil {
					t.Fatalf("mkdir linked worktree: %v", err)
				}
				cwd := mainWT
				if tc.linked {
					cwd = linkedWT
				}
				// Both roles render an embedded template long enough to go
				// through the prompt file, so the pointer is the last arg.
				pointer := PromptFilePointer(filepath.Join(stateDir, "prompts", "org-a_"+tc.role+".md"))
				want := slices.Concat(sandbox, []string{"--model", "gpt-5-codex", pointer})
				if tc.wantFlags {
					want = slices.Concat(sandbox, []string{"--add-dir", stateDir, "--model", "gpt-5-codex", pointer})
				}

				o, h, _ := testOrg(t)
				o.Manifest = NewManifestStoreAtPath(ManifestPathIn(stateDir))
				o.Config.Permissions = config.OrgPermissionsConfig{Default: mode, CodexVerified: true}

				p := mustSpawnParams("org-a", tc.role) // seat id = role, as `ralph org start` does for leader
				p.Role = tc.role
				p.Driver = "codex"
				p.Model = "gpt-5-codex"
				p.Cwd = cwd
				if result := o.Spawn(p); result.Outcome != SpawnOutcomeSpawned {
					t.Fatalf("expected spawn to succeed, got %+v", result)
				}
				if got := h.agentStartArgs[0]; !slices.Equal(got, want) {
					t.Fatalf("AgentStart args = %v, want %v", got, want)
				}
			})
		}
	}
}

// --- codex model observation (plan Scope row 3, AC-2/AC-2b/AC-2c/AC-3/AC-4) ---

// codexGuardedOrg returns testOrg(t) with role guarded under codex's
// fail-closed autonomous default (permissionArgsForDriver rejects a codex
// seat under autonomous mode -- see
// TestOrgSpawn_Codex_AutonomousDefault_RejectedFailClosed_WithReceipt
// above), so every model-observation test below can reach
// SpawnOutcomeSpawned without also exercising that unrelated gate.
func codexGuardedOrg(t *testing.T, role string) (*Org, *fakeHerdr, *fakeAgmsg) {
	t.Helper()
	o, h, a := testOrg(t)
	o.Config.Permissions = config.OrgPermissionsConfig{Roles: map[string]string{role: PermissionModeGuarded}}
	return o, h, a
}

// mustCodexSpawnParams returns spawn params for a codex seat whose Role
// ("implementer") renders a long embedded template (RenderRolePrompt,
// prompts.go), guaranteeing needsPromptFile is true and a role-prompt file
// gets written -- every model-observation test needs that file to exist so
// there is a promptPath to correlate a fixture session record with. Model
// is testOrgConfig()'s only codex [org].model_pool entry.
func mustCodexSpawnParams(orgID, seatID string) SpawnParams {
	p := mustSpawnParams(orgID, seatID)
	p.Role = "implementer"
	p.Driver = "codex"
	p.Model = "gpt-5-codex"
	return p
}

// writeQualifyingCodexFixture writes a minimal rollout-*.jsonl record under
// sessionsDir's date directory for at that fully qualifies
// ObserveCodexEffectiveModel for promptPath: a session_meta at at, a user
// message containing promptPath, and a turn_context reporting model. Built
// on codex_session_test.go's own line-builders/writeRolloutFile/
// fixtureDateDir (same package) rather than duplicating them -- see this
// file's mustCodexSpawnParams for the promptPath every test here needs.
func writeQualifyingCodexFixture(t *testing.T, sessionsDir, promptPath string, at time.Time, model string) string {
	t.Helper()
	lines := []string{
		sessionMetaLine(t, rfc3339Milli(at)),
		userMessageLine(t, rfc3339Milli(at.Add(50*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(at.Add(20*time.Millisecond)), model),
	}
	name := fmt.Sprintf("rollout-%d.jsonl", at.UnixNano())
	return writeRolloutFile(t, fixtureDateDir(sessionsDir, at), name, lines)
}

// TestOrgSpawn_Codex_ModelObservation_Found covers AC-2's true/false split:
// a session record that already exists (before Spawn is even called, so
// the very first, no-wait poll finds it) reporting the commanded model
// gets honored=true; reporting a different model gets honored=false with
// both model names surfaced in Reason.
func TestOrgSpawn_Codex_ModelObservation_Found(t *testing.T) {
	cases := []struct {
		name          string
		reportedModel string
		wantHonored   string
	}{
		{"matches commanded model", "gpt-5-codex", HonoredTrue},
		{"differs from commanded model (retired/migrated)", "gpt-5.6-sol", HonoredFalse},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o, _, _ := codexGuardedOrg(t, "implementer")
			// The fixture already qualifies before Spawn is even called, so
			// its very first, no-wait poll must find it -- give that pass a
			// generous scan budget rather than testOrg's tiny default.
			o.CodexModelObserveTimeout = testCodexObserveGenerousBudget
			p := mustCodexSpawnParams("org-a", "seat-1")

			promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
			if err != nil {
				t.Fatalf("promptFilePath: %v", err)
			}
			writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, time.Now().Add(time.Second), tc.reportedModel)

			result := o.Spawn(p)
			if result.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("expected spawned, got %+v", result)
			}
			if result.ModelReceipt.Honored != tc.wantHonored {
				t.Fatalf("expected honored=%s, got %+v", tc.wantHonored, result.ModelReceipt)
			}
			if result.ModelReceipt.ReportedEffectiveModel != tc.reportedModel {
				t.Fatalf("expected reported model %q, got %+v", tc.reportedModel, result.ModelReceipt)
			}
			if tc.wantHonored == HonoredFalse {
				if !strings.Contains(result.ModelReceipt.Reason, tc.reportedModel) || !strings.Contains(result.ModelReceipt.Reason, "gpt-5-codex") {
					t.Fatalf("expected Reason to name both the commanded and reported models, got %q", result.ModelReceipt.Reason)
				}
			}

			// AC-4 (recorded alongside AC-2 here): the same receipt is
			// persisted, not only returned.
			rr, err := o.Receipts.Read()
			if err != nil {
				t.Fatalf("read receipts: %v", err)
			}
			if len(rr.Receipts) != 1 || rr.Receipts[0] != result.ModelReceipt {
				t.Fatalf("expected the persisted receipt to match SpawnResult.ModelReceipt, got %+v vs %+v", rr.Receipts, result.ModelReceipt)
			}
		})
	}
}

// TestOrgSpawn_Codex_ModelObservation_NotFoundAfterTimeout is AC-3: no
// matching session record exists at all, so the poll runs out its (tiny,
// testOrg-pinned) timeout and the spawn still succeeds, with an
// honored=unknown receipt.
func TestOrgSpawn_Codex_ModelObservation_NotFoundAfterTimeout(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	p := mustCodexSpawnParams("org-a", "seat-1")

	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected spawned, got %+v", result)
	}
	if result.ModelReceipt.Honored != HonoredUnknown || result.ModelReceipt.ReportedEffectiveModel != "" {
		t.Fatalf("expected an unknown receipt with no reported model, got %+v", result.ModelReceipt)
	}
	if !strings.Contains(result.ModelReceipt.Reason, "no codex session record") {
		t.Fatalf("expected the not-found reason text, got %q", result.ModelReceipt.Reason)
	}
}

// TestOrgSpawn_Codex_ModelObservation_ReadError_DistinctReason: the poll's
// timeout elapses and the LAST attempt returned a non-nil observer error
// (a candidate record existed -- it was
// already Lstat'd as regular/name-matching/recently-modified -- but could
// not be opened), so the unknown receipt's reason must say so distinctly
// from "nothing found yet", without ever naming the error text or the
// path (content-free, matching ObserveCodexEffectiveModel's own error
// contract).
func TestOrgSpawn_Codex_ModelObservation_ReadError_DistinctReason(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignores a read-only file's permission bit")
	}
	o, _, _ := codexGuardedOrg(t, "implementer")
	// This test's own claim is that the exact reason text distinguishes a
	// COMPLETED read-error pass from not-found -- that requires at least
	// one pass to reach os.Open on the permission-denied candidate and
	// return before the budget expires. testOrg's tiny (1ms) default risks
	// cutting that first pass short, degrading the reason to not-found
	// (lastErr stays nil): the third category testCodexObserveGenerousBudget's
	// own doc names -- a pass must COMPLETE before the budget runs out. A
	// modest, explicit budget gives that one open+immediate-fail pass ample
	// margin while still running out quickly (this poll never finds
	// anything to return early on, so the test takes as long as this
	// budget: 200ms buys a wide margin over one pass for 0.2s of suite
	// time).
	o.CodexModelObserveTimeout = 200 * time.Millisecond
	p := mustCodexSpawnParams("org-a", "seat-1")

	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	fixturePath := writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, time.Now().Add(time.Second), "gpt-5-codex")
	if err := os.Chmod(fixturePath, 0o000); err != nil {
		t.Fatalf("chmod fixture unreadable: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(fixturePath, 0o644) })

	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected spawned, got %+v", result)
	}
	if result.ModelReceipt.Honored != HonoredUnknown || result.ModelReceipt.ReportedEffectiveModel != "" {
		t.Fatalf("expected an unknown receipt with no reported model, got %+v", result.ModelReceipt)
	}
	if result.ModelReceipt.Reason != codexReadErrorReason {
		t.Fatalf("expected reason %q, got %q", codexReadErrorReason, result.ModelReceipt.Reason)
	}
}

// TestObserveCodexSpawnReceipt_CtxDone_DistinctReason covers the third
// unknown-reason case: the wait is cut short by ctx (Spawn's own
// --timeout-ms budget) before this function's own timeout naturally
// elapses -- a distinct reason from both "nothing found yet" and "a
// record could not be read". Unit-tested directly against
// observeCodexSpawnReceipt (the smallest setup that reaches this arm)
// rather than threading a cancelled ctx through the
// full Spawn/herdr round trip.
func TestObserveCodexSpawnReceipt_CtxDone_DistinctReason(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	// Wide enough that the poll would keep retrying for a while on its own
	// -- ctx must be what actually cuts it short here, not this timeout.
	o.CodexModelObserveTimeout = time.Second
	o.CodexModelObserveInterval = 500 * time.Millisecond

	promptPath, err := o.promptFilePath("org-a", "seat-1")
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	// No fixture at all: every poll attempt is not-found, no error -- ctx
	// expiring mid-wait is the only way this reaches its return.

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	base := Receipt{OrgID: "org-a", SeatID: "seat-1", Role: "implementer", Driver: "codex", CommandedModel: "gpt-5-codex"}
	receipt := o.observeCodexSpawnReceipt(ctx, base, promptPath, time.Now())

	if receipt.Honored != HonoredUnknown || receipt.ReportedEffectiveModel != "" {
		t.Fatalf("expected an unknown receipt with no reported model, got %+v", receipt)
	}
	if receipt.Reason != codexCutShortReason {
		t.Fatalf("expected reason %q, got %q", codexCutShortReason, receipt.Reason)
	}
}

// TestObserveCodexSpawnReceipt_ReadErrorClearedByLaterCleanPass is a /test
// cycle-1 addition (issue #173 handoff item 9g): a /test-cycle mutation run
// confirmed that reverting the 6b63b69 fix -- so a completed pass's lastErr
// is only ever SET on a genuine error and never CLEARED by a later clean
// pass ("if err != nil && !isCtxDoneErr(err) { lastErr = err }" instead of
// "if !isCtxDoneErr(err) { lastErr = err }") -- makes the whole
// internal/org suite pass unchanged; no existing test pinned this specific
// behavior. This test does: a candidate file that exists but is
// permission-denied on the first poll (a genuine read error, matched by
// TestOrgSpawn_Codex_ModelObservation_ReadError_DistinctReason's own
// technique), fixed to readable by a background goroutine shortly after
// (comfortably before the observation budget elapses, given the margins
// below) -- but its content never matches promptPath, so every pass after
// the fix completes cleanly (no error, no match), never
// CodexObservationFound. The receipt's reason must be codexNotFoundReason
// (the later clean pass cleared the earlier read error), never
// codexReadErrorReason. Confirmed to fail (red) against the 9g mutation and
// pass (green) at HEAD.
func TestObserveCodexSpawnReceipt_ReadErrorClearedByLaterCleanPass(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignores a read-only file's permission bit")
	}
	o, _, _ := codexGuardedOrg(t, "implementer")
	o.CodexModelObserveTimeout = 300 * time.Millisecond
	o.CodexModelObserveInterval = 5 * time.Millisecond

	promptPath, err := o.promptFilePath("org-a", "seat-1")
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	spawnStartedAt := time.Now()
	// A candidate whose embedded pointer sentence names a wholly unrelated
	// path -- NOT a suffix/prefix variant of promptPath (which would still
	// contain promptPath as a substring and match anyway, per
	// codexResponseItemMentionsPrompt's Contains check) -- so no pass this
	// test drives can ever match it, and the loop keeps polling (never
	// CodexObservationFound) until the observation budget above elapses.
	nonMatching := writeQualifyingCodexFixture(t, o.CodexSessionsDir, "/unrelated/other-org_other-seat.md", spawnStartedAt.Add(time.Second), "gpt-5-codex")
	if err := os.Chmod(nonMatching, 0o000); err != nil {
		t.Fatalf("chmod fixture unreadable: %v", err)
	}
	fixed := make(chan struct{})
	go func() {
		time.Sleep(15 * time.Millisecond)
		_ = os.Chmod(nonMatching, 0o644)
		close(fixed)
	}()
	t.Cleanup(func() {
		<-fixed
		_ = os.Chmod(nonMatching, 0o644)
	})

	base := Receipt{OrgID: "org-a", SeatID: "seat-1", Role: "implementer", Driver: "codex", CommandedModel: "gpt-5-codex"}
	receipt := o.observeCodexSpawnReceipt(context.Background(), base, promptPath, spawnStartedAt)

	if receipt.Honored != HonoredUnknown || receipt.ReportedEffectiveModel != "" {
		t.Fatalf("expected an unknown receipt with no reported model, got %+v", receipt)
	}
	if receipt.Reason != codexNotFoundReason {
		t.Fatalf("expected reason %q (a later CLEAN pass must clear an earlier pass's read error), got %q", codexNotFoundReason, receipt.Reason)
	}
}

// TestOrgSpawn_Codex_ObservationBudgetExhaustedMidPass_UnknownNotFoundReason
// is AC-5's first clause, driven through the full Spawn saga: a
// genuinely qualifying fixture exists on disk, but the observation's own
// budget (o.codexModelObserveTimeout(), not Spawn's --timeout-ms) is
// exhausted before any pass can complete -- o.CodexModelObserveTimeout set
// to a single nanosecond means obsCtx is already done by the time the
// observer's own first ctx.Err() check runs (inside candidate collection),
// so this is deterministic without any real sleeping. The result must
// still be codexNotFoundReason (this function's own budget, not the
// parent ctx) despite the fixture's existence.
func TestOrgSpawn_Codex_ObservationBudgetExhaustedMidPass_UnknownNotFoundReason(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	p := mustCodexSpawnParams("org-a", "seat-1")

	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, time.Now().Add(time.Second), "gpt-5-codex")
	o.CodexModelObserveTimeout = time.Nanosecond

	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected spawned, got %+v", result)
	}
	if result.ModelReceipt.Honored != HonoredUnknown || result.ModelReceipt.ReportedEffectiveModel != "" {
		t.Fatalf("expected an unknown receipt with no reported model despite a qualifying fixture existing, got %+v", result.ModelReceipt)
	}
	if !strings.Contains(result.ModelReceipt.Reason, "no codex session record") {
		t.Fatalf("expected the not-found reason text (the observation budget ran out, not the parent ctx), got %q", result.ModelReceipt.Reason)
	}
}

// TestOrgSpawn_Codex_ParentCtxCancelledFirst_UnknownCutShortReason is AC-5's
// second clause, driven through the full Spawn saga: this function's own
// observation budget is comparatively generous, but Spawn's own
// --timeout-ms (p.TimeoutMS, 1ms here) is what has already elapsed by the
// time the observation step runs -- deterministic given the manifest
// writes and locking every earlier saga step performs, without any real
// sleeping in this test itself. codexCutShortReason (the parent ctx),
// never codexNotFoundReason (this function's own budget), must be
// reported.
func TestOrgSpawn_Codex_ParentCtxCancelledFirst_UnknownCutShortReason(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	o.CodexModelObserveTimeout = time.Second
	o.CodexModelObserveInterval = 500 * time.Millisecond
	p := mustCodexSpawnParams("org-a", "seat-1")
	p.TimeoutMS = 1

	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected spawned, got %+v", result)
	}
	if result.ModelReceipt.Honored != HonoredUnknown || result.ModelReceipt.ReportedEffectiveModel != "" {
		t.Fatalf("expected an unknown receipt with no reported model, got %+v", result.ModelReceipt)
	}
	if result.ModelReceipt.Reason != codexCutShortReason {
		t.Fatalf("expected reason %q (the PARENT ctx, Spawn's own --timeout-ms, cut this short), got %q", codexCutShortReason, result.ModelReceipt.Reason)
	}
}

// TestOrgSpawn_Codex_ParentCtxCancelledAfterReadError_CutShortStillWins is a
// /test cycle-1 addition (issue #173 handoff item 9h): a /test-cycle
// mutation run confirmed that swapping observeCodexSpawnReceipt's priority
// order -- checking lastErr before ctx.Err(), so a genuine read error from
// an earlier completed pass would win over a parent ctx that is ALSO done
// by the time the wait fails -- makes the whole internal/org suite pass
// unchanged; TestOrgSpawn_Codex_ParentCtxCancelledFirst_UnknownCutShortReason
// above never actually exercises a non-nil lastErr (it writes no fixture at
// all), so it cannot discriminate the swap. This test combines both: a
// permission-denied fixture makes the very first poll pass record a
// genuine read error (same technique as
// TestOrgSpawn_Codex_ModelObservation_ReadError_DistinctReason), and a
// short-but-not-immediate p.TimeoutMS (long enough to survive the saga and
// that first pass, short enough to expire during the generous
// CodexModelObserveInterval wait that follows) makes the parent ctx also
// done by the time waitOrCtxDone fails. codexCutShortReason (the parent
// ctx) must still win, per this function's own documented priority order
// ("checked first, so it wins over the two reasons below whenever both are
// true"). Confirmed to fail (red) against the 9h swap and pass (green) at
// HEAD.
func TestOrgSpawn_Codex_ParentCtxCancelledAfterReadError_CutShortStillWins(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignores a read-only file's permission bit")
	}
	o, _, _ := codexGuardedOrg(t, "implementer")
	o.CodexModelObserveTimeout = time.Second
	o.CodexModelObserveInterval = 500 * time.Millisecond
	p := mustCodexSpawnParams("org-a", "seat-1")
	p.TimeoutMS = 50

	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	fixturePath := writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, time.Now().Add(time.Second), "gpt-5-codex")
	if err := os.Chmod(fixturePath, 0o000); err != nil {
		t.Fatalf("chmod fixture unreadable: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(fixturePath, 0o644) })

	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected spawned, got %+v", result)
	}
	if result.ModelReceipt.Honored != HonoredUnknown || result.ModelReceipt.ReportedEffectiveModel != "" {
		t.Fatalf("expected an unknown receipt with no reported model, got %+v", result.ModelReceipt)
	}
	if result.ModelReceipt.Reason != codexCutShortReason {
		t.Fatalf("expected reason %q (the parent ctx must win even though an earlier pass also recorded a genuine read error), got %q", codexCutShortReason, result.ModelReceipt.Reason)
	}
}

// TestOrgSpawn_Codex_ModelObservation_Ambiguous is AC-2b's sibling: two
// records both qualify (same promptPath, both age-eligible), so the poll
// ends at once with an unknown receipt naming neither model -- never a
// guess.
func TestOrgSpawn_Codex_ModelObservation_Ambiguous(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	// Reaching "ambiguous" requires reading BOTH qualifying candidates in
	// full before the first, no-wait poll can conclude -- give that pass a
	// generous scan budget rather than testOrg's tiny default.
	o.CodexModelObserveTimeout = testCodexObserveGenerousBudget
	p := mustCodexSpawnParams("org-a", "seat-1")

	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, time.Now().Add(time.Second), "gpt-5-codex")
	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, time.Now().Add(2*time.Second), "gpt-5.6-sol")

	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected spawned, got %+v", result)
	}
	if result.ModelReceipt.Honored != HonoredUnknown || result.ModelReceipt.ReportedEffectiveModel != "" {
		t.Fatalf("expected an unknown receipt with no reported model, got %+v", result.ModelReceipt)
	}
	if !strings.Contains(result.ModelReceipt.Reason, "more than one") {
		t.Fatalf("expected the ambiguous reason text, got %q", result.ModelReceipt.Reason)
	}
}

// TestOrgSpawn_Codex_ModelObservation_FoundOnLaterPoll proves the poll
// actually retries: no record exists at spawn time, one appears from a
// background goroutine partway through the (widened, for this test only)
// observe window, and Spawn's own receipt still picks it up.
func TestOrgSpawn_Codex_ModelObservation_FoundOnLaterPoll(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	o.CodexModelObserveInterval = 10 * time.Millisecond
	o.CodexModelObserveTimeout = testCodexObserveGenerousBudget
	p := mustCodexSpawnParams("org-a", "seat-1")

	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}

	at := time.Now().Add(time.Second)
	dateDir := fixtureDateDir(o.CodexSessionsDir, at)
	lines := []string{
		sessionMetaLine(t, rfc3339Milli(at)),
		userMessageLine(t, rfc3339Milli(at.Add(5*time.Millisecond)), PromptFilePointer(promptPath)),
		turnContextLine(t, rfc3339Milli(at.Add(3*time.Millisecond)), "gpt-5-codex"),
	}
	content := strings.Join(lines, "\n") + "\n"
	fixturePath := filepath.Join(dateDir, "rollout-late.jsonl")

	// The write happens on a background goroutine (not via t.Fatalf-using
	// helpers -- testing.T's FailNow must only be called from the test's
	// own goroutine) partway through the poll window, so the first several
	// ObserveCodexEffectiveModel calls must come back not-found before a
	// later one finds it.
	writeErrCh := make(chan error, 1)
	go func() {
		time.Sleep(30 * time.Millisecond)
		if err := os.MkdirAll(dateDir, 0o755); err != nil {
			writeErrCh <- err
			return
		}
		writeErrCh <- os.WriteFile(fixturePath, []byte(content), 0o644)
	}()

	result := o.Spawn(p)
	if err := <-writeErrCh; err != nil {
		t.Fatalf("write delayed fixture: %v", err)
	}
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected spawned, got %+v", result)
	}
	if result.ModelReceipt.Honored != HonoredTrue {
		t.Fatalf("expected honored=true once the record appears on a later poll, got %+v", result.ModelReceipt)
	}
}

// TestOrgSpawn_Codex_OldSessionRecordNotPickedOnRespawn is AC-2b through
// Spawn's own wiring (not only through ObserveCodexEffectiveModel's unit
// tests in codex_session_test.go): a stale record at this exact seat's
// promptPath (promptFilePath is deterministic on org_id/seat_id, so a
// prior attempt's record shares the same path a fresh spawn's role-prompt
// file gets written to) that started long before this Spawn call, but
// whose file is touched again afterward -- simulating a still-running old
// codex process appending to its own log -- must not be picked. Spawn's
// own spawnStartedAt (not the fixture's ModTime) is what has to exclude
// it.
func TestOrgSpawn_Codex_OldSessionRecordNotPickedOnRespawn(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	// This test's own claim is that the stale candidate is examined and
	// excluded BY AGE -- not merely that Spawn ends up unknown some other
	// way. With testOrg's tiny (1ms) default, a pass cut short mid-scan
	// would reach the same "unknown" outcome without ever actually reading
	// the stale record's session_meta timestamp, letting this test pass
	// for the wrong reason under load. Unlike the FOUND tests, this one
	// must NOT use the generous budget: nothing here will ever be found,
	// so the poll always waits out the full budget before returning --
	// testCodexObserveGenerousBudget would make this one test take 30s. A
	// modest, explicit budget instead gives ample margin over the single
	// fast (one-line) read this test's single candidate needs, without
	// slowing the suite. Under a slow enough scan even this 50ms can still
	// be cut short, and the test still passes without ever reading the
	// record's age -- it cannot fail from an unlucky schedule, only pass
	// for the wrong reason; the age-exclusion rule itself is pinned
	// deterministically, independent of any real clock, by
	// TestObserveCodexEffectiveModel_OldSessionUpdatedLaterNotPicked
	// (codex_session_test.go), which runs on context.Background() and so
	// can never be cut short at all.
	o.CodexModelObserveTimeout = 50 * time.Millisecond
	p := mustCodexSpawnParams("org-a", "seat-1")

	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}

	staleAt := time.Now().Add(-time.Hour)
	staleFixture := writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, staleAt, "gpt-5.5-stale")
	touchedAt := time.Now().Add(time.Hour)
	if err := os.Chtimes(staleFixture, touchedAt, touchedAt); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected spawned, got %+v", result)
	}
	if result.ModelReceipt.Honored != HonoredUnknown || result.ModelReceipt.ReportedEffectiveModel != "" {
		t.Fatalf("expected the stale record to be excluded (unknown, no reported model), got %+v", result.ModelReceipt)
	}
}

// TestOrgSpawn_Codex_InlinePromptSkipsObservation_NoWaiting is AC-2c: a
// codex seat whose initial prompt is short enough to pass inline (no
// role-prompt file ever gets written) must return its "no role-prompt
// file" unknown receipt without ever polling -- proven here by pinning a
// deliberately large observe timeout/interval and asserting Spawn still
// returns almost immediately, well under that timeout.
func TestOrgSpawn_Codex_InlinePromptSkipsObservation_NoWaiting(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "worker") // "worker" has no embedded template (see RenderRolePrompt/mustSpawnParams)
	o.CodexModelObserveTimeout = 2 * time.Second
	o.CodexModelObserveInterval = 50 * time.Millisecond

	p := mustSpawnParams("org-a", "seat-1")
	p.Driver = "codex"
	p.Model = "gpt-5-codex"
	p.Prompt = "short inline prompt" // no newline, well under maxInlinePromptRunes -> passed inline, no file written

	start := time.Now()
	result := o.Spawn(p)
	elapsed := time.Since(start)

	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected spawned, got %+v", result)
	}
	if result.ModelReceipt.Honored != HonoredUnknown || result.ModelReceipt.ReportedEffectiveModel != "" {
		t.Fatalf("expected an unknown receipt with no reported model, got %+v", result.ModelReceipt)
	}
	if !strings.Contains(result.ModelReceipt.Reason, "no role-prompt file") {
		t.Fatalf("expected the no-role-prompt-file reason text, got %q", result.ModelReceipt.Reason)
	}
	if elapsed >= o.CodexModelObserveTimeout {
		t.Fatalf("expected no waiting for a seat with no role-prompt file: spawn took %s (>= the %s observe timeout)", elapsed, o.CodexModelObserveTimeout)
	}
}

// TestOrgSpawn_Claude_ModelReceiptTextUnchanged is the AC-4 regression
// guard: a claude seat's receipt text must stay exactly what it was before
// this plan (no observation is ever attempted for a non-codex driver).
func TestOrgSpawn_Claude_ModelReceiptTextUnchanged(t *testing.T) {
	o, _, _ := testOrg(t)
	p := mustSpawnParams("org-a", "seat-1") // Driver: "claude" (mustSpawnParams default)

	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected spawned, got %+v", result)
	}
	if result.ModelReceipt.Honored != HonoredUnknown || result.ModelReceipt.Reason != "interactive session; effective model not yet observable" {
		t.Fatalf("expected the unchanged claude receipt text, got %+v", result.ModelReceipt)
	}
}

// TestOrgSpawn_Codex_DryRun_ModelReceiptUnchanged is the AC-4 regression
// guard for the dry-run path: dryRunSpawn's own receipt (Reason "dry-run")
// must stay exactly what it was before this plan, for codex the same as
// any other driver -- dry-run never calls the observer at all. It also
// asserts through SpawnResult.ModelReceipt, not only the receipts store:
// that field must equal the persisted receipt here too, never stay the
// zero value while a receipt was actually appended.
func TestOrgSpawn_Codex_DryRun_ModelReceiptUnchanged(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	p := mustCodexSpawnParams("org-a", "seat-1")
	p.DryRun = true

	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected spawned (dry-run), got %+v", result)
	}
	if result.ModelReceipt.Reason != "dry-run" || result.ModelReceipt.Honored != HonoredUnknown {
		t.Fatalf("expected SpawnResult.ModelReceipt to carry the dry-run receipt, got %+v", result.ModelReceipt)
	}
	rr, err := o.Receipts.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	if len(rr.Receipts) != 1 || rr.Receipts[0] != result.ModelReceipt {
		t.Fatalf("expected the persisted receipt to match SpawnResult.ModelReceipt, got %+v vs %+v", rr.Receipts, result.ModelReceipt)
	}
}

// TestOrgSpawn_RetiredLeaderName_RejectedBeforeAnyManifestWrite covers the
// plain rejection of the coordinator's retired name: --role, --id, and the
// two ralph.toml keys. For every case, in both dry-run and real mode, the
// outcome is Rejected with an error that names the replacement, and the
// guard sits ahead of any manifest event or receipt, so neither gets one
// (the normal reject() path would have written one of each).
func TestOrgSpawn_RetiredLeaderName_RejectedBeforeAnyManifestWrite(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(p *SpawnParams, cfg *config.OrgConfig)
		wantErr []string
	}{
		{
			name:    "role",
			mutate:  func(p *SpawnParams, _ *config.OrgConfig) { p.Role = oldLeaderName },
			wantErr: []string{LeaderIdentity, "--role " + LeaderIdentity},
		},
		{
			name: "role with --prompt is still rejected",
			mutate: func(p *SpawnParams, _ *config.OrgConfig) {
				p.Role = oldLeaderName
				p.Prompt = "a prompt does not rescue the old name"
			},
			wantErr: []string{LeaderIdentity, "--role " + LeaderIdentity},
		},
		{
			name:    "seat id",
			mutate:  func(p *SpawnParams, _ *config.OrgConfig) { p.SeatID = oldLeaderName },
			wantErr: []string{LeaderIdentity, "agmsg identity"},
		},
		{
			name: "seat id with role leader and --prompt is still rejected",
			mutate: func(p *SpawnParams, _ *config.OrgConfig) {
				p.SeatID = oldLeaderName
				p.Role = LeaderIdentity
				p.Prompt = "custom"
			},
			wantErr: []string{LeaderIdentity, "agmsg identity"},
		},
		{
			name:    "[org.roles] key",
			mutate:  func(_ *SpawnParams, cfg *config.OrgConfig) { cfg.Roles[oldLeaderName] = []string{"sonnet"} },
			wantErr: []string{"[org.roles]." + oldLeaderName, "[org.roles]." + LeaderIdentity},
		},
		{
			name: "[org.permissions.roles] key",
			mutate: func(_ *SpawnParams, cfg *config.OrgConfig) {
				cfg.Permissions.Roles = map[string]string{oldLeaderName: "guarded"}
			},
			wantErr: []string{"[org.permissions.roles]." + oldLeaderName, "[org.permissions.roles]." + LeaderIdentity},
		},
	}
	for _, tc := range cases {
		for _, dryRun := range []bool{false, true} {
			mode := "real"
			if dryRun {
				mode = "dry-run"
			}
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				o, h, a := testOrg(t)
				p := mustSpawnParams("org-a", "seat-1")
				p.DryRun = dryRun
				tc.mutate(&p, &o.Config)

				result := o.Spawn(p)

				if result.Outcome != SpawnOutcomeRejected {
					t.Fatalf("Outcome = %v, want SpawnOutcomeRejected (err=%v)", result.Outcome, result.Err)
				}
				if result.Err == nil {
					t.Fatal("expected a non-nil Err so the CLI exits non-zero")
				}
				for _, want := range tc.wantErr {
					if !strings.Contains(result.Err.Error(), want) {
						t.Errorf("error %q should contain %q", result.Err.Error(), want)
					}
				}
				if len(h.calls) != 0 || len(a.calls) != 0 {
					t.Errorf("expected no driver calls, got herdr=%v agmsg=%v", h.calls, a.calls)
				}
				if got := eventNames(t, o); len(got) != 0 {
					t.Errorf("expected no manifest event for the retired-name rejection, got %v", got)
				}
				rr, err := o.Receipts.Read()
				if err != nil {
					t.Fatalf("read receipts: %v", err)
				}
				if len(rr.Receipts) != 0 {
					t.Errorf("expected no receipt for the retired-name rejection, got %+v", rr.Receipts)
				}
				if result.ModelReceipt != (Receipt{}) {
					t.Errorf("expected a zero ModelReceipt, got %+v", result.ModelReceipt)
				}
			})
		}
	}
}

// TestOrgSpawn_RetiredLeaderName_ConfigKeyErrorNamesEveryOffendingKey pins
// that a ralph.toml with the old key in both tables lists both, so the
// operator renames them in one pass instead of one rejection per table.
func TestOrgSpawn_RetiredLeaderName_ConfigKeyErrorNamesEveryOffendingKey(t *testing.T) {
	o, _, _ := testOrg(t)
	o.Config.Roles[oldLeaderName] = []string{"sonnet"}
	o.Config.Permissions.Roles = map[string]string{oldLeaderName: "guarded"}

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))

	if result.Outcome != SpawnOutcomeRejected || result.Err == nil {
		t.Fatalf("expected a rejection, got %+v", result)
	}
	for _, want := range []string{"[org.roles]." + oldLeaderName, "[org.permissions.roles]." + oldLeaderName} {
		if !strings.Contains(result.Err.Error(), want) {
			t.Errorf("error %q should name %q", result.Err.Error(), want)
		}
	}
}

// TestOrgSpawn_RetiredLeaderName_CaseSensitive pins that the guard compares
// exactly, like the rest of the role handling: an upper-case spelling is a
// different (unknown) role and spawns normally.
func TestOrgSpawn_RetiredLeaderName_CaseSensitive(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		mode := "real"
		if dryRun {
			mode = "dry-run"
		}
		t.Run(mode, func(t *testing.T) {
			o, _, _ := testOrg(t)
			p := mustSpawnParams("org-a", "seat-1")
			p.DryRun = dryRun
			p.Role = strings.ToUpper(oldLeaderName[:1]) + oldLeaderName[1:]

			result := o.Spawn(p)

			if result.Outcome != SpawnOutcomeSpawned || result.Err != nil {
				t.Fatalf("expected the upper-case spelling %q to spawn, got %+v", p.Role, result)
			}
		})
	}
}

// TestOrgSpawn_RetiredLeaderName_UnrelatedConfigKeysAndRolesStillSpawn is the
// negative control for the guard: other roles, an empty role, and role keys
// in the config tables that are not the old name must not be touched.
func TestOrgSpawn_RetiredLeaderName_UnrelatedConfigKeysAndRolesStillSpawn(t *testing.T) {
	for _, role := range []string{"", "worker", LeaderIdentity, "reviewer"} {
		t.Run("role="+role, func(t *testing.T) {
			o, _, _ := testOrg(t)
			o.Config.Roles["worker"] = []string{"sonnet"}
			o.Config.Permissions.Roles = map[string]string{"reviewer": "edits"}
			p := mustSpawnParams("org-a", "seat-1")
			p.Role = role

			result := o.Spawn(p)

			if result.Outcome != SpawnOutcomeSpawned || result.Err != nil {
				t.Fatalf("expected role %q to spawn, got %+v", role, result)
			}
		})
	}
}

// TestOrgSpawn_RemovedRole_WithoutPrompt_RejectedBeforeAnyManifestWrite covers
// the guard for a role whose template was removed outright: with no --prompt
// there is nothing to tell the seat what to do, so the spawn is refused with
// guidance to the successor role and to --prompt. Like the renamed-name
// rejection it sits ahead of the manifest and the receipts, in dry-run and
// real mode alike.
func TestOrgSpawn_RemovedRole_WithoutPrompt_RejectedBeforeAnyManifestWrite(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		mode := "real"
		if dryRun {
			mode = "dry-run"
		}
		t.Run(mode, func(t *testing.T) {
			o, h, a := testOrg(t)
			p := mustSpawnParams("org-a", "seat-1")
			p.DryRun = dryRun
			p.Role = removedRoleName

			result := o.Spawn(p)

			if result.Outcome != SpawnOutcomeRejected {
				t.Fatalf("Outcome = %v, want SpawnOutcomeRejected (err=%v)", result.Outcome, result.Err)
			}
			if result.Err == nil {
				t.Fatal("expected a non-nil Err so the CLI exits non-zero")
			}
			for _, want := range []string{"reviewer", "--prompt", removedRoleName} {
				if !strings.Contains(result.Err.Error(), want) {
					t.Errorf("error %q should contain %q", result.Err.Error(), want)
				}
			}
			if len(h.calls) != 0 || len(a.calls) != 0 {
				t.Errorf("expected no driver calls, got herdr=%v agmsg=%v", h.calls, a.calls)
			}
			if got := eventNames(t, o); len(got) != 0 {
				t.Errorf("expected no manifest event for the removed-role rejection, got %v", got)
			}
			rr, err := o.Receipts.Read()
			if err != nil {
				t.Fatalf("read receipts: %v", err)
			}
			if len(rr.Receipts) != 0 {
				t.Errorf("expected no receipt for the removed-role rejection, got %+v", rr.Receipts)
			}
			if result.ModelReceipt != (Receipt{}) {
				t.Errorf("expected a zero ModelReceipt, got %+v", result.ModelReceipt)
			}
		})
	}
}

// TestOrgSpawn_RemovedRole_WithPrompt_StartsWithPromptOnly pins the other half
// of the guard: a --prompt gives the seat its purpose, so the removed name is
// then an ordinary custom role. No template exists for it, so the initial
// prompt is exactly the --prompt text -- inline when it is short and
// single-line, through the prompt file otherwise.
func TestOrgSpawn_RemovedRole_WithPrompt_StartsWithPromptOnly(t *testing.T) {
	t.Run("inline prompt", func(t *testing.T) {
		o, h, _ := testOrg(t)
		p := mustSpawnParams("org-a", "seat-1")
		p.Role = removedRoleName
		p.Prompt = "custom qa instructions"

		result := o.Spawn(p)

		if result.Outcome != SpawnOutcomeSpawned || result.Err != nil {
			t.Fatalf("expected the removed role with --prompt to spawn, got %+v", result)
		}
		args := h.agentStartArgs[0]
		// [--permission-mode bypassPermissions --model <model> <prompt>].
		if len(args) != 5 || args[4] != p.Prompt {
			t.Fatalf("expected the initial prompt to be exactly %q as the last AgentStart arg, got %v", p.Prompt, args)
		}
	})

	t.Run("multi-line prompt goes through the prompt file unchanged", func(t *testing.T) {
		o, h, _ := testOrg(t)
		p := mustSpawnParams("org-a", "seat-1")
		p.Role = removedRoleName
		p.Prompt = "custom qa instructions\nsecond line"

		result := o.Spawn(p)

		if result.Outcome != SpawnOutcomeSpawned || result.Err != nil {
			t.Fatalf("expected the removed role with --prompt to spawn, got %+v", result)
		}
		args := h.agentStartArgs[0]
		if len(args) != 5 || !strings.HasPrefix(args[4], "役割指示を読み込んで従ってください: ") {
			t.Fatalf("expected a prompt-file pointer as the last AgentStart arg, got %v", args)
		}
		promptPath := strings.TrimPrefix(args[4], "役割指示を読み込んで従ってください: ")
		data, err := os.ReadFile(promptPath)
		if err != nil {
			t.Fatalf("expected the prompt file to exist at %q: %v", promptPath, err)
		}
		if string(data) != p.Prompt {
			t.Fatalf("expected the prompt file to hold exactly the --prompt text %q, got %q", p.Prompt, string(data))
		}
	})

	t.Run("dry-run", func(t *testing.T) {
		o, h, _ := testOrg(t)
		p := mustSpawnParams("org-a", "seat-1")
		p.DryRun = true
		p.Role = removedRoleName
		p.Prompt = "custom qa instructions"

		result := o.Spawn(p)

		if result.Outcome != SpawnOutcomeSpawned || result.Err != nil {
			t.Fatalf("expected the removed role with --prompt to pass a dry-run, got %+v", result)
		}
		if len(h.calls) != 0 {
			t.Errorf("expected a dry-run to make no herdr calls, got %v", h.calls)
		}
	})
}

// TestOrgSpawn_RemovedRole_OnlyTheRoleIsGuarded pins what the removed-kind
// guard leaves alone: the name as a seat id (only a renamed name's old
// agmsg identity is rejected there), and ralph.toml keys under the name
// (a legitimate custom-role key for a seat that brings its own --prompt).
func TestOrgSpawn_RemovedRole_OnlyTheRoleIsGuarded(t *testing.T) {
	t.Run("seat id with role reviewer", func(t *testing.T) {
		o, _, _ := testOrg(t)
		p := mustSpawnParams("org-a", removedRoleName)
		p.Role = "reviewer"

		result := o.Spawn(p)

		if result.Outcome != SpawnOutcomeSpawned || result.Err != nil {
			t.Fatalf("expected seat id %q with role reviewer to spawn, got %+v", removedRoleName, result)
		}
	})

	t.Run("seat id with an empty role and no prompt", func(t *testing.T) {
		o, _, _ := testOrg(t)
		p := mustSpawnParams("org-a", removedRoleName)
		p.Role = ""

		result := o.Spawn(p)

		if result.Outcome != SpawnOutcomeSpawned || result.Err != nil {
			t.Fatalf("expected seat id %q with an empty role to spawn, got %+v", removedRoleName, result)
		}
	})

	t.Run("config keys under the name", func(t *testing.T) {
		o, _, _ := testOrg(t)
		o.Config.Roles[removedRoleName] = []string{"sonnet"}
		o.Config.Permissions.Roles = map[string]string{removedRoleName: "guarded"}
		p := mustSpawnParams("org-a", "seat-1")
		p.Role = removedRoleName
		p.Prompt = "custom qa instructions"

		result := o.Spawn(p)

		if result.Outcome != SpawnOutcomeSpawned || result.Err != nil {
			t.Fatalf("expected [org.roles].%s and [org.permissions.roles].%s not to block a spawn, got %+v",
				removedRoleName, removedRoleName, result)
		}
	})

	t.Run("upper-case spelling is an unknown role", func(t *testing.T) {
		o, _, _ := testOrg(t)
		p := mustSpawnParams("org-a", "seat-1")
		p.Role = strings.ToUpper(removedRoleName)

		result := o.Spawn(p)

		if result.Outcome != SpawnOutcomeSpawned || result.Err != nil {
			t.Fatalf("expected the upper-case spelling %q to spawn, got %+v", p.Role, result)
		}
	})
}

// TestOrgSpawn_RetiredLeaderName_ConfigKeyErrorDescribesTheRealFallback pins
// the wording of the retired-key rejection: the role falls back to the full
// model_pool and to [org.permissions].default, which is only "autonomous" when
// the operator has not changed it, so the message must not assert autonomous.
func TestOrgSpawn_RetiredLeaderName_ConfigKeyErrorDescribesTheRealFallback(t *testing.T) {
	o, _, _ := testOrg(t)
	o.Config.Permissions.Roles = map[string]string{oldLeaderName: "guarded"}

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))

	if result.Outcome != SpawnOutcomeRejected || result.Err == nil {
		t.Fatalf("expected a rejection, got %+v", result)
	}
	msg := result.Err.Error()
	for _, want := range []string{"model_pool", "[org.permissions].default"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q should contain %q", msg, want)
		}
	}
	if strings.Contains(msg, "autonomous") {
		t.Errorf("error %q must not claim the fallback is autonomous: [org.permissions].default decides it", msg)
	}
}

// TestOrgSpawn_RetiredLeaderName_ConfigKeyAddedAfterSpawn_RespawnStaysIdempotent
// pins where the ralph.toml retired-key check sits on the real path: after the
// idempotent early return. A seat spawned before the config gained the old key
// (an org started by an older binary, or ralph.toml edited mid-org) must
// re-run as a no-op, while a new seat under the same config is still refused
// with the key-rename message and nothing written to the manifest or the
// receipts.
func TestOrgSpawn_RetiredLeaderName_ConfigKeyAddedAfterSpawn_RespawnStaysIdempotent(t *testing.T) {
	o, h, a := testOrg(t)

	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("initial spawn failed: %+v", r)
	}
	o.Config.Permissions.Roles = map[string]string{oldLeaderName: "guarded"}

	eventsBefore := eventNames(t, o)
	callsBefore, sendsBefore := len(h.calls), len(a.calls)
	rrBefore, err := o.Receipts.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	assertNothingWritten := func(t *testing.T) {
		t.Helper()
		if got := eventNames(t, o); len(got) != len(eventsBefore) {
			t.Errorf("expected no new manifest event, before=%v after=%v", eventsBefore, got)
		}
		if len(h.calls) != callsBefore || len(a.calls) != sendsBefore {
			t.Errorf("expected no new driver calls, herdr %d->%d agmsg %d->%d", callsBefore, len(h.calls), sendsBefore, len(a.calls))
		}
		rr, err := o.Receipts.Read()
		if err != nil {
			t.Fatalf("read receipts: %v", err)
		}
		if len(rr.Receipts) != len(rrBefore.Receipts) {
			t.Errorf("expected no new receipt, before=%d after=%d", len(rrBefore.Receipts), len(rr.Receipts))
		}
	}

	t.Run("re-running the spawned seat is idempotent", func(t *testing.T) {
		result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
		if result.Outcome != SpawnOutcomeIdempotent || result.Err != nil {
			t.Fatalf("expected SpawnOutcomeIdempotent with nil Err, got %v (err=%v)", result.Outcome, result.Err)
		}
		assertNothingWritten(t)
	})

	t.Run("a new seat is still rejected", func(t *testing.T) {
		result := o.Spawn(mustSpawnParams("org-a", "seat-2"))
		if result.Outcome != SpawnOutcomeRejected || result.Err == nil {
			t.Fatalf("expected a rejection, got %v (err=%v)", result.Outcome, result.Err)
		}
		for _, want := range []string{"[org.permissions.roles]." + oldLeaderName, "[org.permissions.roles]." + LeaderIdentity} {
			if !strings.Contains(result.Err.Error(), want) {
				t.Errorf("error %q should contain %q", result.Err.Error(), want)
			}
		}
		if result.ModelReceipt != (Receipt{}) {
			t.Errorf("expected a zero ModelReceipt, got %+v", result.ModelReceipt)
		}
		assertNothingWritten(t)
	})
}

// TestOrgSpawn_RetiredLeaderName_ConfigKey_RejectedBeforeStaleCompensation pins
// the other side of that position: the check runs before stale-in-flight
// compensation, so a seat with an unresolved spawn_started is neither sent
// C-c nor marked spawn_failed when the request is refused for an old key.
func TestOrgSpawn_RetiredLeaderName_ConfigKey_RejectedBeforeStaleCompensation(t *testing.T) {
	o, h, a := testOrg(t)
	if err := o.Manifest.Append(ManifestEvent{
		TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-a", Event: EventSpawnStarted,
		Role: "worker", Driver: "claude", Model: "sonnet", PaneID: "stale-pane-1",
	}); err != nil {
		t.Fatalf("seed stale spawn_started: %v", err)
	}
	o.Config.Roles[oldLeaderName] = []string{"sonnet"}

	result := o.Spawn(mustSpawnParams("org-a", "seat-a"))

	if result.Outcome != SpawnOutcomeRejected || result.Err == nil {
		t.Fatalf("expected a rejection, got %v (err=%v)", result.Outcome, result.Err)
	}
	if !strings.Contains(result.Err.Error(), "[org.roles]."+oldLeaderName) {
		t.Errorf("error %q should name [org.roles].%s", result.Err.Error(), oldLeaderName)
	}
	if len(h.calls) != 0 || len(a.calls) != 0 || len(h.sendKeysCalls) != 0 {
		t.Errorf("expected no driver calls and no compensation, got herdr=%v agmsg=%v sendKeys=%v", h.calls, a.calls, h.sendKeysCalls)
	}
	if got := eventNames(t, o); len(got) != 1 || got[0] != EventSpawnStarted {
		t.Errorf("expected only the seeded spawn_started (no spawn_failed, no rejected), got %v", got)
	}
}

// --- org-wide limits and scope reservations (plan 2026-10-08-org-limits-reserve) ---

// leaderParams is mustSpawnParams for orgID's leader seat, reserving reserve
// (none when empty).
func leaderParams(orgID string, reserve ...string) SpawnParams {
	p := mustSpawnParams(orgID, LeaderIdentity)
	p.Role = LeaderIdentity
	p.Reserve = reserve
	return p
}

// spawnLeaderIn is spawnSeatIn (verbs_all_test.go) for orgID's leader seat,
// reserving reserve.
func spawnLeaderIn(t *testing.T, o *Org, h *fakeHerdr, orgID, workspace, pane string, reserve ...string) {
	t.Helper()
	h.workspaceID, h.paneID = workspace, pane
	if r := o.Spawn(leaderParams(orgID, reserve...)); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn %s/%s failed: %+v", orgID, LeaderIdentity, r)
	}
}

// lastEvent returns the latest event of o's manifest.
func lastEvent(t *testing.T, o *Org) ManifestEvent {
	t.Helper()
	events := mustReadEvents(t, o)
	if len(events) == 0 {
		t.Fatal("expected a manifest event, got none")
	}
	return events[len(events)-1]
}

// receiptCount returns how many receipts o's receipt store holds.
func receiptCount(t *testing.T, o *Org) int {
	t.Helper()
	rr, err := o.Receipts.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	return len(rr.Receipts)
}

// assertRejectedRecorded checks that r is a rejection whose error contains
// every string in want, recorded through reject(): the latest manifest event
// is `rejected` for orgID/seatID carrying the error, and r carries the
// honored=false receipt it appended.
func assertRejectedRecorded(t *testing.T, o *Org, r SpawnResult, orgID, seatID string, want ...string) {
	t.Helper()
	if r.Outcome != SpawnOutcomeRejected || r.Err == nil {
		t.Fatalf("expected a rejection, got %+v", r)
	}
	for _, w := range want {
		if !strings.Contains(r.Err.Error(), w) {
			t.Errorf("error %q does not contain %q", r.Err, w)
		}
	}
	if ev := lastEvent(t, o); ev.Event != EventRejected || ev.OrgID != orgID || ev.SeatID != seatID || ev.Details != r.Err.Error() {
		t.Fatalf("expected a rejected event for %s/%s carrying the error, got %+v", orgID, seatID, ev)
	}
	if r.ModelReceipt.Honored != HonoredFalse || r.ModelReceipt.Reason != r.Err.Error() {
		t.Fatalf("expected an honored=false receipt carrying the error, got %+v", r.ModelReceipt)
	}
}

// assertSeatUnchanged checks that orgID's seatID is still the active
// `spawned` seat it was.
func assertSeatUnchanged(t *testing.T, o *Org, orgID, seatID string) {
	t.Helper()
	seat, ok := seatFromEvents(mustReadEvents(t, o), orgID, seatID)
	if !ok || !seat.Active || seat.Event != EventSpawned {
		t.Fatalf("expected %s/%s still active and spawned, got %+v (found=%v)", orgID, seatID, seat, ok)
	}
}

// raceSpawns runs o.Spawn(params(i)) for every i in [0, n) concurrently and
// returns how many spawned and the rejections, failing the test on any other
// outcome.
func raceSpawns(t *testing.T, o *Org, n int, params func(i int) SpawnParams) (spawned int, rejected []SpawnResult) {
	t.Helper()
	var wg sync.WaitGroup
	wg.Add(n)
	results := make([]SpawnResult, n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			results[i] = o.Spawn(params(i))
		}(i)
	}
	wg.Wait()
	for _, r := range results {
		switch r.Outcome {
		case SpawnOutcomeSpawned:
			spawned++
		case SpawnOutcomeRejected:
			rejected = append(rejected, r)
		default:
			t.Errorf("unexpected outcome %v (err=%v)", r.Outcome, r.Err)
		}
	}
	return spawned, rejected
}

// TestOrgSpawn_MaxOrgs_NewOrgRejectedAtLimit covers plan AC1: with max_orgs
// orgs running, a spawn into an org_id that is not running is rejected with a
// `rejected` record and no driver call, and the error points at disband; a
// spawn into a running org is not limited by max_orgs. The rejected org_id
// takes no slot: once another org is disbanded, it spawns.
func TestOrgSpawn_MaxOrgs_NewOrgRejectedAtLimit(t *testing.T) {
	o, h, a := testOrg(t)
	o.Config.MaxOrgs = 2
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-a1")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
	herdrBefore, agmsgBefore := len(h.calls), len(a.calls)

	assertRejectedRecorded(t, o, o.Spawn(mustSpawnParams("org-c", "seat-1")), "org-c", "seat-1",
		"max_orgs 2 reached", `org_id "org-c" is not running`, "(org-a, org-b)",
		"ralph org disband --org-id <id>", "ralph org disband --all")
	if len(h.calls) != herdrBefore || len(a.calls) != agmsgBefore {
		t.Fatalf("expected no driver call for the rejection, herdr %v agmsg %v", h.calls[herdrBefore:], a.calls[agmsgBefore:])
	}

	spawnSeatIn(t, o, h, "org-a", "seat-2", "ws-a", "pane-a2")
	if got := RunningOrgs(mustReadEvents(t, o)); !slices.Equal(got, []string{"org-a", "org-b"}) {
		t.Fatalf("RunningOrgs = %v, want org-a and org-b (the rejected org-c holds no slot)", got)
	}

	if d := o.Disband(DisbandParams{OrgID: "org-b"}); !d.Disbanded {
		t.Fatalf("disband org-b: %+v", d)
	}
	spawnSeatIn(t, o, h, "org-c", "seat-1", "ws-c", "pane-c1")
}

// TestOrgSpawn_MaxOrgs_StoppedOrgHoldsItsSlotUntilDisbanded: an org whose
// seats are all stopped still has its workspace open, so it keeps its slot
// (plan Design decisions) until disband closes the workspace.
func TestOrgSpawn_MaxOrgs_StoppedOrgHoldsItsSlotUntilDisbanded(t *testing.T) {
	o, h, _ := testOrg(t)
	o.Config.MaxOrgs = 1
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-a1")
	if r := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"}); r.Err != nil {
		t.Fatalf("stop org-a/seat-1: %v", r.Err)
	}
	assertRejectedRecorded(t, o, o.Spawn(mustSpawnParams("org-b", "seat-1")), "org-b", "seat-1", "max_orgs 1 reached")

	if d := o.Disband(DisbandParams{OrgID: "org-a"}); !d.Disbanded {
		t.Fatalf("disband org-a: %+v", d)
	}
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
}

// TestOrgSpawn_MaxTotalSeats_Boundary covers plan AC2: with max_total_seats
// seats active across all orgs, a new seat is rejected with a `rejected`
// record, in a running org and in a new one, while a respawn of a spawned
// seat still returns it with no record; a stopped seat frees its place.
func TestOrgSpawn_MaxTotalSeats_Boundary(t *testing.T) {
	o, h, _ := testOrg(t)
	o.Config.MaxTotalSeats = 3
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-a1")
	spawnSeatIn(t, o, h, "org-a", "seat-2", "ws-a", "pane-a2")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")

	assertRejectedRecorded(t, o, o.Spawn(mustSpawnParams("org-b", "seat-2")), "org-b", "seat-2",
		"max_total_seats 3 reached", "3 seats are active across all orgs", "ralph org disband --org-id <id>", "ralph org disband --all")
	assertRejectedRecorded(t, o, o.Spawn(mustSpawnParams("org-c", "seat-1")), "org-c", "seat-1", "max_total_seats 3 reached")

	before := len(mustReadEvents(t, o))
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeIdempotent || r.Err != nil {
		t.Fatalf("expected the respawn of org-a/seat-1 to return it at max_total_seats, got %+v", r)
	}
	if got := len(mustReadEvents(t, o)); got != before {
		t.Fatalf("expected no event for the idempotent respawn, %d -> %d", before, got)
	}

	if r := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"}); r.Err != nil {
		t.Fatalf("stop org-a/seat-1: %v", r.Err)
	}
	spawnSeatIn(t, o, h, "org-b", "seat-2", "ws-b", "pane-b2")
}

// TestOrgSpawn_OrgWideLimits_DryRunPredictsTheSameRejection: a dry run checks
// both org-wide limits against the real orgs and seats, as it does max_seats,
// and records its rejection as a dry-run `rejected`; dry-run seats never
// count toward either limit.
func TestOrgSpawn_OrgWideLimits_DryRunPredictsTheSameRejection(t *testing.T) {
	o, h, _ := testOrg(t)
	o.Config.MaxOrgs, o.Config.MaxTotalSeats = 1, 2
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-a1")
	dryRun := func(orgID, seatID string) SpawnResult {
		p := mustSpawnParams(orgID, seatID)
		p.DryRun = true
		return o.Spawn(p)
	}

	if r := dryRun("org-b", "seat-1"); r.Outcome != SpawnOutcomeRejected || r.Err == nil || !strings.Contains(r.Err.Error(), "max_orgs 1 reached") {
		t.Fatalf("expected the dry run into a new org rejected for max_orgs, got %+v", r)
	}
	if ev := lastEvent(t, o); ev.Event != EventRejected || !ev.DryRun || ev.OrgID != "org-b" {
		t.Fatalf("expected a dry-run rejected event for org-b, got %+v", ev)
	}
	for _, seatID := range []string{"seat-2", "seat-3"} {
		if r := dryRun("org-a", seatID); r.Outcome != SpawnOutcomeSpawned {
			t.Fatalf("expected the dry run of org-a/%s to pass below max_total_seats, got %+v", seatID, r)
		}
	}
	spawnSeatIn(t, o, h, "org-a", "seat-2", "ws-a", "pane-a2")
	if r := dryRun("org-a", "seat-3"); r.Outcome != SpawnOutcomeRejected || r.Err == nil || !strings.Contains(r.Err.Error(), "max_total_seats 2 reached") {
		t.Fatalf("expected the dry run rejected for max_total_seats once two real seats are active, got %+v", r)
	}
}

// TestOrgSpawn_ConcurrentSpawns_MaxOrgsNeverExceeded covers plan AC4 for
// max_orgs, in the shape of TestOrgSpawn_ConcurrentSpawns_MaxSeatsNeverExceeded:
// first spawns into n distinct orgs race, and exactly max_orgs of them spawn.
func TestOrgSpawn_ConcurrentSpawns_MaxOrgsNeverExceeded(t *testing.T) {
	o, _, _ := testOrg(t)
	o.Config.MaxOrgs = 3
	spawned, rejected := raceSpawns(t, o, 10, func(i int) SpawnParams {
		return mustSpawnParams(fmt.Sprintf("org-%d", i), "seat-1")
	})
	if spawned != o.Config.MaxOrgs {
		t.Fatalf("expected exactly max_orgs=%d spawns out of 10 racing new orgs, got %d", o.Config.MaxOrgs, spawned)
	}
	for _, r := range rejected {
		if !strings.Contains(r.Err.Error(), "max_orgs 3 reached") {
			t.Errorf("expected every rejection to be for max_orgs, got %v", r.Err)
		}
	}
	if got := RunningOrgs(mustReadEvents(t, o)); len(got) != o.Config.MaxOrgs {
		t.Fatalf("RunningOrgs = %v, want exactly %d", got, o.Config.MaxOrgs)
	}
}

// TestOrgSpawn_ConcurrentSpawns_MaxTotalSeatsNeverExceeded covers plan AC4
// for max_total_seats: spawns of three seats in each of four orgs race
// (max_seats 3, so only the total limits them), and exactly max_total_seats
// spawn.
func TestOrgSpawn_ConcurrentSpawns_MaxTotalSeatsNeverExceeded(t *testing.T) {
	o, _, _ := testOrg(t)
	o.Config.MaxTotalSeats = 4
	spawned, rejected := raceSpawns(t, o, 12, func(i int) SpawnParams {
		return mustSpawnParams(fmt.Sprintf("org-%d", i%4), fmt.Sprintf("seat-%d", i/4))
	})
	if spawned != o.Config.MaxTotalSeats {
		t.Fatalf("expected exactly max_total_seats=%d spawns out of 12 racing seats, got %d", o.Config.MaxTotalSeats, spawned)
	}
	for _, r := range rejected {
		if !strings.Contains(r.Err.Error(), "max_total_seats 4 reached") {
			t.Errorf("expected every rejection to be for max_total_seats, got %v", r.Err)
		}
	}
	if got := TotalActiveSeats(mustReadEvents(t, o)); got != o.Config.MaxTotalSeats {
		t.Fatalf("TotalActiveSeats = %d, want exactly %d", got, o.Config.MaxTotalSeats)
	}
}

// TestOrgSpawn_ConcurrentReservations_OnlyOneOfOverlappingWins covers plan
// AC4 for reservations: leader spawns of eight orgs race with reservations
// that all overlap one another, and exactly one holds a reservation.
func TestOrgSpawn_ConcurrentReservations_OnlyOneOfOverlappingWins(t *testing.T) {
	o, _, _ := testOrg(t)
	nested := []string{".", "internal/", "internal/auth/", "internal/auth/token.go"}
	spawned, rejected := raceSpawns(t, o, 8, func(i int) SpawnParams {
		return leaderParams(fmt.Sprintf("org-%d", i), nested[i%len(nested)])
	})
	if spawned != 1 {
		t.Fatalf("expected exactly one of 8 overlapping reservations to win, got %d", spawned)
	}
	for _, r := range rejected {
		if !strings.Contains(r.Err.Error(), "reservation overlaps the reservation of running org_id") {
			t.Errorf("expected every rejection to be an overlap, got %v", r.Err)
		}
	}
	if n := countString(eventNames(t, o), EventScopeReserved); n != 1 {
		t.Fatalf("expected exactly one scope_reserved, got %d", n)
	}
}

// TestOrgSpawn_Reserve_RecordedBeforeSpawnStarted: a leader spawn with
// Reserve appends one org-level scope_reserved with the normalized paths
// right before its spawn_started, and that is the org's reservation.
func TestOrgSpawn_Reserve_RecordedBeforeSpawnStarted(t *testing.T) {
	o, _, _ := testOrg(t)
	if r := o.Spawn(leaderParams("org-a", "./internal//auth/", "docs/", "docs/")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	events := mustReadEvents(t, o)
	if ev := events[0]; ev.Event != EventScopeReserved || ev.OrgID != "org-a" || ev.SeatID != "" || ev.DryRun || ev.Details != "paths=docs/,internal/auth/" {
		t.Fatalf("expected the org-level scope_reserved first, got %+v", ev)
	}
	if ev := events[1]; ev.Event != EventSpawnStarted || ev.SeatID != LeaderIdentity {
		t.Fatalf("expected the leader's spawn_started right after it, got %+v", ev)
	}
	if n := countString(eventNames(t, o), EventScopeReserved); n != 1 {
		t.Fatalf("expected one scope_reserved, got %d", n)
	}
	if got := ActiveReservation(events, "org-a"); !slices.Equal(got, []string{"docs/", "internal/auth/"}) {
		t.Fatalf("ActiveReservation = %q, want docs/ and internal/auth/", got)
	}
	assertSeatUnchanged(t, o, "org-a", LeaderIdentity)
}

// TestOrgSpawn_Reserve_InputRejectedBeforeAnyRecord covers plan AC6 and the
// leader-only rule of AC7: Reserve on a seat other than the leader, or with
// a path the rules reject, is a plain rejection before the manifest is read
// (no event, no receipt, no driver call), in real and dry-run mode alike.
func TestOrgSpawn_Reserve_InputRejectedBeforeAnyRecord(t *testing.T) {
	worker := mustSpawnParams("org-a", "seat-1")
	worker.Reserve = []string{"internal/"}
	for _, tc := range []struct {
		name string
		p    SpawnParams
		want string
	}{
		{"a worker seat", worker, `only the leader seat reserves paths for its org; seat_id "seat-1" cannot`},
		{"an absolute path", leaderParams("org-a", "/etc/"), "is absolute"},
		{"a dot dot segment", leaderParams("org-a", "internal/../../x"), "has a .. segment"},
		{"an empty path", leaderParams("org-a", "internal/", ""), "is empty"},
	} {
		for _, dryRun := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s dry-run=%v", tc.name, dryRun), func(t *testing.T) {
				o, h, a := testOrg(t)
				p := tc.p
				p.DryRun = dryRun
				r := o.Spawn(p)
				if r.Outcome != SpawnOutcomeRejected || r.Err == nil || !strings.Contains(r.Err.Error(), tc.want) {
					t.Fatalf("expected a rejection containing %q, got %+v", tc.want, r)
				}
				if events := mustReadEvents(t, o); len(events) != 0 {
					t.Fatalf("expected no manifest event, got %+v", events)
				}
				if n := receiptCount(t, o); n != 0 {
					t.Fatalf("expected no receipt, got %d", n)
				}
				if len(h.calls) != 0 || len(a.calls) != 0 {
					t.Fatalf("expected no driver call, herdr %v agmsg %v", h.calls, a.calls)
				}
			})
		}
	}
}

// TestOrgSpawn_Reserve_OverlapWithRunningOrgRejected covers plan AC5: while
// org-a reserves internal/auth/, org-b's start reserving a file under it, a
// parent directory, or the whole repo is rejected with a `rejected` record
// naming org-a and the overlapping paths, and holds nothing; a sibling
// directory with a shared prefix (internal/authz/) spawns.
func TestOrgSpawn_Reserve_OverlapWithRunningOrgRejected(t *testing.T) {
	o, h, _ := testOrg(t)
	spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-a", "internal/auth/")
	for _, path := range []string{"internal/auth/token.go", "internal/", "."} {
		assertRejectedRecorded(t, o, o.Spawn(leaderParams("org-b", path)), "org-b", LeaderIdentity,
			`reservation overlaps the reservation of running org_id "org-a"`, path+" overlaps internal/auth/")
	}
	events := mustReadEvents(t, o)
	if got := ActiveReservation(events, "org-b"); got != nil {
		t.Fatalf("expected the rejected org-b to hold no reservation, got %q", got)
	}
	if got := RunningOrgs(events); !slices.Equal(got, []string{"org-a"}) {
		t.Fatalf("RunningOrgs = %v, want only org-a", got)
	}

	spawnLeaderIn(t, o, h, "org-b", "ws-b", "pane-b", "internal/authz/")
	if got := ActiveReservation(mustReadEvents(t, o), "org-b"); !slices.Equal(got, []string{"internal/authz/"}) {
		t.Fatalf("ActiveReservation(org-b) = %q, want internal/authz/", got)
	}
}

// TestOrgSpawn_Reserve_SameSetPassesDifferentSetRejected covers plan AC7 for
// a new leader spawn into an org that holds a reservation (its earlier leader
// stopped): the same set, however written, passes with no new record, a
// different set is rejected with a `rejected` record, and a seat spawned
// without Reserve is not affected by the reservation.
func TestOrgSpawn_Reserve_SameSetPassesDifferentSetRejected(t *testing.T) {
	o, h, _ := testOrg(t)
	stopLeader := func() {
		t.Helper()
		if r := o.Stop(StopParams{OrgID: "org-a", Seat: LeaderIdentity}); r.Err != nil {
			t.Fatalf("stop org-a's leader: %v", r.Err)
		}
	}
	spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-1", "internal/", "docs/")
	stopLeader()

	spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-2", "docs//", "./internal/")
	if n := countString(eventNames(t, o), EventScopeReserved); n != 1 {
		t.Fatalf("expected the same set to add no scope_reserved, got %d", n)
	}
	stopLeader()

	assertRejectedRecorded(t, o, o.Spawn(leaderParams("org-a", "internal/")), "org-a", LeaderIdentity,
		`org_id "org-a" already reserves docs/,internal/`, "a different reservation (internal/) is refused", "ralph org disband --org-id org-a")

	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-3")
	if got := ActiveReservation(mustReadEvents(t, o), "org-a"); !slices.Equal(got, []string{"docs/", "internal/"}) {
		t.Fatalf("ActiveReservation = %q, want the first reservation unchanged", got)
	}
}

// TestOrgSpawn_Reserve_ExistingLeader covers plan AC14: a spawn with Reserve
// of a leader that is already spawned decides the reservation before
// returning the seat. Without a reservation yet it reserves (one
// scope_reserved, no driver call); the same set passes with no record; a
// different set, or one overlapping another running org, is refused with no
// record at all and the seat unchanged; a retry without Reserve returns the
// seat as before.
func TestOrgSpawn_Reserve_ExistingLeader(t *testing.T) {
	type check struct {
		events, receipts, herdr, agmsg int
	}
	snapshot := func(t *testing.T, o *Org, h *fakeHerdr, a *fakeAgmsg) check {
		return check{len(mustReadEvents(t, o)), receiptCount(t, o), len(h.calls), len(a.calls)}
	}
	t.Run("no reservation yet: reserved, seat returned", func(t *testing.T) {
		o, h, a := testOrg(t)
		spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-a")
		before := snapshot(t, o, h, a)

		r := o.Spawn(leaderParams("org-a", "internal/"))
		if r.Outcome != SpawnOutcomeIdempotent || r.Err != nil || r.Seat.SeatID != LeaderIdentity || !r.Seat.Active {
			t.Fatalf("expected the existing leader returned, got %+v", r)
		}
		after := snapshot(t, o, h, a)
		if after.events != before.events+1 || after.receipts != before.receipts || after.herdr != before.herdr || after.agmsg != before.agmsg {
			t.Fatalf("expected one new event and nothing else, %+v -> %+v", before, after)
		}
		if ev := lastEvent(t, o); ev.Event != EventScopeReserved || ev.OrgID != "org-a" || ev.Details != "paths=internal/" {
			t.Fatalf("expected org-a's scope_reserved, got %+v", ev)
		}
		assertSeatUnchanged(t, o, "org-a", LeaderIdentity)
	})
	t.Run("same set: passes with no record", func(t *testing.T) {
		o, h, a := testOrg(t)
		spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-a", "internal/")
		before := snapshot(t, o, h, a)
		if r := o.Spawn(leaderParams("org-a", "./internal//")); r.Outcome != SpawnOutcomeIdempotent || r.Err != nil {
			t.Fatalf("expected the existing leader returned, got %+v", r)
		}
		if after := snapshot(t, o, h, a); after != before {
			t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
		}
	})
	for _, tc := range []struct {
		name    string
		setup   func(t *testing.T, o *Org, h *fakeHerdr)
		reserve string
		want    string
	}{
		{"different set: refused, seat unchanged", func(t *testing.T, o *Org, h *fakeHerdr) {
			spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-a", "internal/")
		}, "docs/", `org_id "org-a" already reserves internal/`},
		{"overlap with another running org: refused, seat unchanged", func(t *testing.T, o *Org, h *fakeHerdr) {
			spawnLeaderIn(t, o, h, "org-b", "ws-b", "pane-b", "docs/")
			spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-a")
		}, "docs/api/", `running org_id "org-b": docs/api/ overlaps docs/`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h, a := testOrg(t)
			tc.setup(t, o, h)
			reservedBefore := ActiveReservation(mustReadEvents(t, o), "org-a")
			before := snapshot(t, o, h, a)

			r := o.Spawn(leaderParams("org-a", tc.reserve))
			if r.Outcome != SpawnOutcomeRejected || r.Err == nil || !strings.Contains(r.Err.Error(), tc.want) {
				t.Fatalf("expected a rejection containing %q, got %+v", tc.want, r)
			}
			if after := snapshot(t, o, h, a); after != before {
				t.Fatalf("expected no event, receipt or driver call, %+v -> %+v", before, after)
			}
			assertSeatUnchanged(t, o, "org-a", LeaderIdentity)
			if got := ActiveReservation(mustReadEvents(t, o), "org-a"); !slices.Equal(got, reservedBefore) {
				t.Fatalf("ActiveReservation(org-a) = %q, want unchanged %q", got, reservedBefore)
			}
		})
	}
	t.Run("no reserve: returned as before", func(t *testing.T) {
		o, h, a := testOrg(t)
		spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-a", "internal/")
		before := snapshot(t, o, h, a)
		if r := o.Spawn(leaderParams("org-a")); r.Outcome != SpawnOutcomeIdempotent || r.Err != nil {
			t.Fatalf("expected the existing leader returned, got %+v", r)
		}
		if after := snapshot(t, o, h, a); after != before {
			t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
		}
	})
}

// TestOrgSpawn_Reserve_ExistingLeader_Phase2 covers plan AC14 on Spawn's
// second idempotent return: a leader spawn with Reserve finds a stale
// in-flight saga, and a racer spawns the leader while the stale seat is
// compensated (the seam of
// TestOrgSpawn_StaleInFlight_RacerCompletesDuringCompensationWindow_Phase2ReturnsIdempotent),
// so Phase 2's fresh read finds the leader spawned. The reservation is
// decided there too: the org had none, so it is recorded after the racer's
// `spawned`, and the racer's leader is returned with no second saga.
func TestOrgSpawn_Reserve_ExistingLeader_Phase2(t *testing.T) {
	o, h, _ := testOrg(t)
	if err := o.Manifest.Append(ManifestEvent{
		TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: LeaderIdentity, Event: EventSpawnStarted,
		Role: LeaderIdentity, Driver: "claude", Model: "sonnet", PaneID: "stale-pane",
	}); err != nil {
		t.Fatalf("seed stale spawn_started: %v", err)
	}
	orig := afterStaleCompensation
	afterStaleCompensation = func() {
		if err := o.Manifest.Append(ManifestEvent{
			TS: "2026-08-01T00:00:05Z", OrgID: "org-a", SeatID: LeaderIdentity, Event: EventSpawned,
			Role: LeaderIdentity, Driver: "claude", Model: "sonnet", PaneID: "racer-pane", AgmsgTeam: "team-racer",
		}); err != nil {
			t.Fatalf("seed racer spawned event: %v", err)
		}
	}
	defer func() { afterStaleCompensation = orig }()

	r := o.Spawn(leaderParams("org-a", "internal/"))
	if r.Outcome != SpawnOutcomeIdempotent || r.Err != nil || r.Seat.PaneID != "racer-pane" {
		t.Fatalf("expected the racer's leader returned by Phase 2, got %+v", r)
	}
	if got := eventNames(t, o); !slices.Equal(got, []string{EventSpawnStarted, EventSpawnFailed, EventSpawned, EventScopeReserved}) {
		t.Fatalf("expected the reservation recorded after the racer's spawned and no second saga, got %v", got)
	}
	if got := ActiveReservation(mustReadEvents(t, o), "org-a"); !slices.Equal(got, []string{"internal/"}) {
		t.Fatalf("ActiveReservation(org-a) = %q, want internal/", got)
	}
	if len(h.calls) != 1 || h.calls[0] != "pane_send_keys" {
		t.Fatalf("expected only the compensation C-c, got %v", h.calls)
	}
}

// TestOrgSpawn_Reserve_DisbandReleasesIt covers the first half of plan AC8:
// once org-a is disbanded, another org reserves the same paths, and org-a
// starts again with a reservation of its own.
func TestOrgSpawn_Reserve_DisbandReleasesIt(t *testing.T) {
	o, h, _ := testOrg(t)
	spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-a", "internal/")
	assertRejectedRecorded(t, o, o.Spawn(leaderParams("org-b", "internal/")), "org-b", LeaderIdentity, `running org_id "org-a"`)

	if d := o.Disband(DisbandParams{OrgID: "org-a"}); !d.Disbanded {
		t.Fatalf("disband org-a: %+v", d)
	}
	if got := ActiveReservation(mustReadEvents(t, o), "org-a"); got != nil {
		t.Fatalf("expected disband to release org-a's reservation, got %q", got)
	}
	spawnLeaderIn(t, o, h, "org-b", "ws-b", "pane-b", "internal/")
	spawnLeaderIn(t, o, h, "org-a", "ws-a2", "pane-a2", "docs/")
}

// TestOrgSpawn_Reserve_SatisfiesAutonomousScopeGate covers plan AC9: under
// the default autonomous mode, a spawn with Reserve and no Scope passes the
// AC-2b gate, in real and dry-run mode; without Reserve it is still refused.
func TestOrgSpawn_Reserve_SatisfiesAutonomousScopeGate(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(fmt.Sprintf("dry-run=%v", dryRun), func(t *testing.T) {
			o, _, _ := testOrg(t)
			p := leaderParams("org-a", "internal/")
			p.Scope, p.DryRun = "", dryRun
			if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("expected a spawn with Reserve and no Scope to pass the gate, got %+v", r)
			}
			p = leaderParams("org-b")
			p.Scope, p.DryRun = "", dryRun
			if r := o.Spawn(p); r.Outcome != SpawnOutcomeRejected || r.Err == nil || !strings.Contains(r.Err.Error(), "requires --scope or --reserve") {
				t.Fatalf("expected a spawn with neither to be refused by the gate, naming both, got %+v", r)
			}
		})
	}
}

// TestOrgSpawn_ZeroOrgWideLimits_Reject: with max_orgs or max_total_seats at
// 0 (only a hand-built config can carry it; config.Load rejects it), spawn
// refuses with a `rejected` record, as max_seats 0 does, instead of treating
// 0 as no limit.
func TestOrgSpawn_ZeroOrgWideLimits_Reject(t *testing.T) {
	t.Run("max_orgs 0", func(t *testing.T) {
		o, _, _ := testOrg(t)
		o.Config.MaxOrgs = 0
		assertRejectedRecorded(t, o, o.Spawn(mustSpawnParams("org-a", "seat-1")), "org-a", "seat-1",
			"max_orgs 0 reached", "0 orgs are (none)")
	})
	t.Run("max_total_seats 0", func(t *testing.T) {
		o, _, _ := testOrg(t)
		o.Config.MaxTotalSeats = 0
		assertRejectedRecorded(t, o, o.Spawn(mustSpawnParams("org-a", "seat-1")), "org-a", "seat-1",
			"max_total_seats 0 reached")
	})
}

// TestOrgSpawn_Reserve_DryRunPreviewsAndHoldsNothing: a dry run decides the
// reservation against the real ones (an overlap is a dry-run `rejected`),
// starts a passing trail with a dry-run scope_reserved, and that record
// holds nothing: the org does not run and a real org reserves the same paths.
func TestOrgSpawn_Reserve_DryRunPreviewsAndHoldsNothing(t *testing.T) {
	o, h, _ := testOrg(t)
	spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-a", "internal/auth/")
	dryRun := func(orgID string, reserve ...string) SpawnResult {
		p := leaderParams(orgID, reserve...)
		p.DryRun = true
		return o.Spawn(p)
	}

	if r := dryRun("org-b", "internal/"); r.Outcome != SpawnOutcomeRejected || r.Err == nil || !strings.Contains(r.Err.Error(), "internal/ overlaps internal/auth/") {
		t.Fatalf("expected the dry run rejected for the overlap, got %+v", r)
	}
	if ev := lastEvent(t, o); ev.Event != EventRejected || !ev.DryRun || ev.OrgID != "org-b" {
		t.Fatalf("expected a dry-run rejected event for org-b, got %+v", ev)
	}

	before := len(mustReadEvents(t, o))
	if r := dryRun("org-b", "docs/"); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected the dry run to pass, got %+v", r)
	}
	events := mustReadEvents(t, o)
	if ev := events[before]; ev.Event != EventScopeReserved || !ev.DryRun || ev.OrgID != "org-b" || ev.SeatID != "" || ev.Details != "paths=docs/" {
		t.Fatalf("expected the dry-run trail to start with a dry-run scope_reserved, got %+v", ev)
	}
	if ev := events[before+1]; ev.Event != EventSpawnStarted || !ev.DryRun {
		t.Fatalf("expected the dry-run spawn_started next, got %+v", ev)
	}
	if got := RunningOrgs(events); !slices.Equal(got, []string{"org-a"}) {
		t.Fatalf("RunningOrgs = %v, want only org-a", got)
	}
	spawnLeaderIn(t, o, h, "org-c", "ws-c", "pane-c", "docs/")
}
