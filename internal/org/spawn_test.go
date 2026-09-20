package org

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
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

	workspaceID string
	paneID      string

	sendKeysCalls     []string   // paneIDs PaneSendKeys was invoked with, in order
	sendKeysKeys      [][]string // keys PaneSendKeys was invoked with, in order (e.g. asserting exactly one "Enter")
	agentStartNames   []string   // agent names AgentStart was invoked with, in order
	agentStartArgs    [][]string // agentArgs AgentStart was invoked with, in order (AC-4 argv assertions)
	agentWaitTargets  []string   // targets AgentWait was invoked with, in order
	agentWaitUntil    [][]string // until states AgentWait was invoked with, in order
	agentWaitTimeouts []int      // timeoutMS AgentWait was invoked with, in order
}

func (f *fakeHerdr) WorkspaceCreate(_ context.Context, _, _ string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "workspace_create")
	if f.workspaceCreateErr != nil {
		return "", f.workspaceCreateErr
	}
	if f.workspaceID == "" {
		f.workspaceID = "ws-1"
	}
	return f.workspaceID, nil
}

func (f *fakeHerdr) TabCreate(_ context.Context, _, _, _ string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "tab_create")
	if f.tabCreateErr != nil {
		return "", f.tabCreateErr
	}
	if f.paneID == "" {
		f.paneID = "pane-1"
	}
	return f.paneID, nil
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

func (f *fakeHerdr) PaneSendKeys(_ context.Context, paneID string, keys ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "pane_send_keys")
	f.sendKeysCalls = append(f.sendKeysCalls, paneID)
	f.sendKeysKeys = append(f.sendKeysKeys, keys)
	if f.paneSendKeysErr != nil {
		return f.paneSendKeysErr
	}
	return nil
}

// fakeAgmsg is a call-recording, in-memory AgmsgClient. joinErrs, keyed by
// agentID (e.g. "lead" or a seat id), lets a test inject a Join failure at
// exactly one identity while leaving the other Join call (lead vs seat)
// unaffected -- needed to test ensureLeadJoined's best-effort semantics
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

func (f *fakeAgmsg) Leave(_ context.Context, team, agentID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "leave")
	f.leaveCalls = append(f.leaveCalls, leaveCall{team: team, agentID: agentID})
	return f.leaveErr
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

	// tab_created, agent_started, agmsg_lead_joined, agmsg_joined, agmsg_announced.
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
	wantAgmsgCalls := []string{"join:lead", "join:seat-1", "send"}
	if len(a.calls) != len(wantAgmsgCalls) {
		t.Fatalf("expected agmsg calls %v, got %v", wantAgmsgCalls, a.calls)
	}
	for i := range wantAgmsgCalls {
		if a.calls[i] != wantAgmsgCalls[i] {
			t.Errorf("agmsg call[%d]: want %q, got %q (full: %v)", i, wantAgmsgCalls[i], a.calls[i], a.calls)
		}
	}
	if len(a.joinCalls) != 2 {
		t.Fatalf("expected 2 Join calls (lead then seat), got %+v", a.joinCalls)
	}
	leadJoin, seatJoin := a.joinCalls[0], a.joinCalls[1]
	if leadJoin.agentID != "lead" || leadJoin.agmsgType != "claude-code" || leadJoin.projectPath != "/tmp/seat" {
		t.Errorf("expected lead Join(team, lead, claude-code, /tmp/seat), got %+v", leadJoin)
	}
	if seatJoin.agentID != "seat-1" || seatJoin.agmsgType != "claude-code" || seatJoin.projectPath != "/tmp/seat" {
		t.Errorf("expected seat Join(team, seat-1, claude-code, /tmp/seat) for a claude driver seat, got %+v", seatJoin)
	}
	if leadJoin.team != seatJoin.team {
		t.Errorf("expected lead and seat Join calls to target the same team, got %q vs %q", leadJoin.team, seatJoin.team)
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
	assertDetailsContains(t, last.Details, "step=agmsg_announce", "lead_join=ok")
}

func TestOrgSpawn_FailureInjection_AgmsgJoin_SeatJoinFails_CompensatesExistingPane(t *testing.T) {
	// Seat Join is a hard-failure gate distinct from the lead's best-effort
	// ensureLeadJoined: a seat Join failure must fail the saga at
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
	if len(a.calls) != 2 || a.calls[0] != "join:lead" || a.calls[1] != "join:seat-1" {
		t.Fatalf("expected lead Join then seat Join (no Send attempted after seat Join fails), got %v", a.calls)
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

func TestOrgSpawn_EnsureLeadJoined_ErrorDoesNotFailSaga_WhenSeatJoinAndSendSucceed(t *testing.T) {
	// ensureLeadJoined is best-effort: an error joining "lead" (e.g. it was
	// already a member and join.sh soft-failed on the retry) must not fail
	// the saga on its own -- the seat's own Join and the HELLO Send are the
	// authoritative gates. The lead-join error is still recorded on the
	// agmsg_lead_joined spawn_step for diagnosis.
	o, _, a := testOrg(t)
	a.joinErrs = map[string]error{"lead": errors.New("stub: lead already a member")}

	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected SpawnOutcomeSpawned despite a lead-join error, got %v (err=%v)", result.Outcome, result.Err)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var leadJoinedStep *ManifestEvent
	for i := range rr.Events {
		if rr.Events[i].Details == "" {
			continue
		}
		if strings.HasPrefix(rr.Events[i].Details, "agmsg_lead_joined") {
			leadJoinedStep = &rr.Events[i]
			break
		}
	}
	if leadJoinedStep == nil {
		t.Fatalf("expected an agmsg_lead_joined spawn_step event, got events %+v", rr.Events)
	}
	assertDetailsContains(t, leadJoinedStep.Details, "error=")
}

func TestOrgSpawn_FailureInjection_AgmsgSend_DetailsIncludeLeadJoinError(t *testing.T) {
	// When HELLO Send fails, the recorded lead-join outcome must be carried
	// into the spawn_failed Details alongside the send failure itself, so an
	// operator can immediately see whether a missing "lead" roster entry is
	// the likely root cause.
	o, _, a := testOrg(t)
	a.joinErrs = map[string]error{"lead": errors.New("stub: lead join failed")}
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
	assertDetailsContains(t, last.Details, "step=agmsg_announce", "lead_join=", "stub: lead join failed", "stub failure: agmsg send")
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
	p.Role = "qa"
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
		t.Fatalf("expected the qa template body in the prompt file, got:\n%s", fileContent)
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

// --- AC-7: LeadIdentity const + lead agmsg type from LeadDriver -------------

func TestLeadIdentity_ConstantValue(t *testing.T) {
	if LeadIdentity != "lead" {
		t.Fatalf("LeadIdentity = %q, want %q", LeadIdentity, "lead")
	}
}

func TestOrgSpawn_EnsureLeadJoined_DefaultLeadDriver_ClaudeCodeType(t *testing.T) {
	o, _, a := testOrg(t)

	p := mustSpawnParams("org-a", "seat-1")
	// LeadDriver left unset -- must default to "claude" -> "claude-code".
	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected SpawnOutcomeSpawned, got %v (err=%v)", result.Outcome, result.Err)
	}
	if len(a.joinCalls) < 1 {
		t.Fatalf("expected at least 1 Join call, got %+v", a.joinCalls)
	}
	leadJoin := a.joinCalls[0]
	if leadJoin.agentID != LeadIdentity || leadJoin.agmsgType != "claude-code" {
		t.Fatalf("expected lead Join(%s, claude-code, ...) by default, got %+v", LeadIdentity, leadJoin)
	}
}

func TestOrgSpawn_EnsureLeadJoined_LeadDriverCodex_UsesCodexAgmsgType(t *testing.T) {
	// The lead identity's own driver (LeadDriver) is independent of the
	// seat's Driver: a claude-driven seat spawned under a codex-coordinated
	// org must still register "lead" with agmsg type "codex", not
	// "claude-code".
	o, _, a := testOrg(t)

	p := mustSpawnParams("org-a", "seat-1")
	p.LeadDriver = "codex"
	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected SpawnOutcomeSpawned, got %v (err=%v)", result.Outcome, result.Err)
	}
	if len(a.joinCalls) != 2 {
		t.Fatalf("expected 2 Join calls (lead then seat), got %+v", a.joinCalls)
	}
	leadJoin, seatJoin := a.joinCalls[0], a.joinCalls[1]
	if leadJoin.agentID != LeadIdentity || leadJoin.agmsgType != "codex" {
		t.Fatalf("expected lead Join(%s, codex, ...) for LeadDriver=codex, got %+v", LeadIdentity, leadJoin)
	}
	if seatJoin.agentID != "seat-1" || seatJoin.agmsgType != "claude-code" {
		t.Fatalf("expected the seat's own Join to still use its own Driver (claude -> claude-code), got %+v", seatJoin)
	}
}

// --- AC-3: `ralph org start` = lead-seat spawn sugar (SeatID == LeadIdentity) ---

func TestOrgSpawn_LeadSelfSpawn_SingleAgmsgJoin_NoHelloSend(t *testing.T) {
	// SeatID == LeadIdentity ("ralph org start") must not double-join or
	// HELLO-announce: the seat's own Join call IS the lead-identity join, and
	// a HELLO from lead to lead would violate the star topology's
	// single-coordinator premise (see the leadSelfSpawn doc comment in
	// spawn.go's Spawn).
	o, _, a := testOrg(t)

	p := mustSpawnParams("org-a", LeadIdentity)
	p.Role = LeadIdentity
	p.Task = "dry-run 座席を spawn し、送信・確認・disband まで行え"
	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected SpawnOutcomeSpawned, got %v (err=%v)", result.Outcome, result.Err)
	}

	if len(a.joinCalls) != 1 {
		t.Fatalf("expected exactly 1 agmsg Join call for a lead-self spawn (no separate ensureLeadJoined), got %+v", a.joinCalls)
	}
	if a.joinCalls[0].agentID != LeadIdentity {
		t.Fatalf("expected the single Join call to register %q, got %+v", LeadIdentity, a.joinCalls[0])
	}
	for _, c := range a.calls {
		if c == "send" {
			t.Fatalf("expected no agmsg Send (HELLO) call for a lead-self spawn, got calls=%v", a.calls)
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
	assertDetailsContains(t, joinedStep.Details, "lead_self=true")
	for _, ev := range rr.Events {
		if strings.HasPrefix(ev.Details, "agmsg_lead_joined") || ev.Details == "agmsg_announced" {
			t.Fatalf("expected no agmsg_lead_joined/agmsg_announced step for a lead-self spawn, got %+v", ev)
		}
	}
}

func TestOrgSpawn_LeadSelfSpawn_DryRun_MirrorsSameSkip(t *testing.T) {
	o, _, _ := testOrg(t)

	p := mustSpawnParams("org-a", LeadIdentity)
	p.Role = LeadIdentity
	p.DryRun = true
	result := o.Spawn(p)
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected SpawnOutcomeSpawned for a valid dry-run, got %v (err=%v)", result.Outcome, result.Err)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	sawJoinedLeadSelf := false
	for _, ev := range rr.Events {
		if strings.HasPrefix(ev.Details, "agmsg_lead_joined") || ev.Details == "agmsg_announced" {
			t.Fatalf("expected no agmsg_lead_joined/agmsg_announced step in the dry-run trail for a lead-self spawn, got %+v", ev)
		}
		if ev.Details == "agmsg_joined lead_self=true" {
			sawJoinedLeadSelf = true
		}
	}
	if !sawJoinedLeadSelf {
		t.Fatalf("expected an 'agmsg_joined lead_self=true' step in the dry-run trail, got events %+v", rr.Events)
	}
}

func TestOrgSpawn_LeadRole_TaskAndEnvelopeSubstitutedIntoPromptFile(t *testing.T) {
	// `ralph org start`'s Task and the org's EnvelopeSummary must both land
	// in the lead seat's rendered prompt file (the lead.md template is long
	// enough to always need the prompt-file path, same as reviewer/qa).
	o, h, _ := testOrg(t)

	p := mustSpawnParams("org-a", LeadIdentity)
	p.Role = LeadIdentity
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
			t.Errorf("expected the lead prompt file content to contain %q, got:\n%s", want, fileContent)
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
	if result.Err == nil || !strings.Contains(result.Err.Error(), "--scope") || !strings.Contains(result.Err.Error(), "--allow-unscoped") {
		t.Fatalf("expected error naming --scope and --allow-unscoped, got %v", result.Err)
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
