package org

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// fakeHerdr is a call-recording, in-memory HerdrClient used by every spawn
// saga unit test in this file -- no exec.Command, no PATH, no real herdr
// binary needed. Per-method Err fields let a test inject a failure at
// exactly one saga boundary.
type fakeHerdr struct {
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

	workspaceID string
	paneID      string

	sendKeysCalls    []string   // paneIDs PaneSendKeys was invoked with, in order
	agentStartNames  []string   // agent names AgentStart was invoked with, in order
	agentStartArgs   [][]string // agentArgs AgentStart was invoked with, in order (AC-4 argv assertions)
	agentWaitTargets []string   // targets AgentWait was invoked with, in order
}

func (f *fakeHerdr) WorkspaceCreate(_ context.Context, _, _ string) (string, error) {
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

func (f *fakeHerdr) AgentWait(_ context.Context, target string, _ []string, _ int) (string, error) {
	f.calls = append(f.calls, "agent_wait")
	f.agentWaitTargets = append(f.agentWaitTargets, target)
	return "idle", nil
}

func (f *fakeHerdr) PaneRead(_ context.Context, _ string, _ int) (string, error) {
	f.calls = append(f.calls, "pane_read")
	return "pane output", nil
}

func (f *fakeHerdr) PaneSendText(_ context.Context, _, _ string) error {
	f.calls = append(f.calls, "pane_send_text")
	return nil
}

func (f *fakeHerdr) PaneSendKeys(_ context.Context, paneID string, _ ...string) error {
	f.calls = append(f.calls, "pane_send_keys")
	f.sendKeysCalls = append(f.sendKeysCalls, paneID)
	if f.paneSendKeysErr != nil {
		return f.paneSendKeysErr
	}
	return nil
}

// fakeAgmsg is a call-recording, in-memory AgmsgClient. joinErrs, keyed by
// agentID (e.g. "lead" or a seat id), lets a test inject a Join failure at
// exactly one identity while leaving the other Join call (lead vs seat)
// unaffected -- needed to test ensureLeadJoined's best-effort semantics
// independently of the seat Join's hard-failure gate.
type fakeAgmsg struct {
	calls   []string
	sendErr error

	joinErrs     map[string]error
	joinCalls    []joinCall
	despawnErr   error
	despawnCalls []despawnCall
}

type joinCall struct {
	team, agentID, agmsgType, projectPath string
}

type despawnCall struct {
	team, from, name string
}

func (f *fakeAgmsg) Send(_ context.Context, _, _, _, _ string) error {
	f.calls = append(f.calls, "send")
	return f.sendErr
}

func (f *fakeAgmsg) Join(_ context.Context, team, agentID, agmsgType, projectPath string) error {
	f.calls = append(f.calls, "join:"+agentID)
	f.joinCalls = append(f.joinCalls, joinCall{team: team, agentID: agentID, agmsgType: agmsgType, projectPath: projectPath})
	if f.joinErrs != nil {
		if err, ok := f.joinErrs[agentID]; ok {
			return err
		}
	}
	return nil
}

func (f *fakeAgmsg) Despawn(_ context.Context, team, from, name string) error {
	f.calls = append(f.calls, "despawn")
	f.despawnCalls = append(f.despawnCalls, despawnCall{team: team, from: from, name: name})
	return f.despawnErr
}

// testOrg builds an Org backed by temp-file manifest/receipt stores and
// fresh fake driver clients, using the shared testOrgConfig() from
// envelope_test.go (max_seats=3, claude+codex pools).
func testOrg(t *testing.T) (*Org, *fakeHerdr, *fakeAgmsg) {
	t.Helper()
	dir := t.TempDir()
	h := &fakeHerdr{}
	a := &fakeAgmsg{}
	o := &Org{
		Config:   testOrgConfig(),
		Manifest: NewManifestStoreAtPath(filepath.Join(dir, "manifest.jsonl")),
		Receipts: NewReceiptStoreAtPath(filepath.Join(dir, "receipts.jsonl")),
		Herdr:    h,
		Agmsg:    a,
	}
	return o, h, a
}

func mustSpawnParams(orgID, seatID string) SpawnParams {
	return SpawnParams{
		OrgID: orgID, SeatID: seatID, Role: "worker", Driver: "claude", Model: "sonnet",
		Cwd: "/tmp/seat", TimeoutMS: 5000,
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
	if got := herdrAgentName("org-a", "reviewer"); got != "org-a-reviewer" {
		t.Fatalf("herdrAgentName(org-a, reviewer) = %q, want org-a-reviewer", got)
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
	if len(args) != 3 || args[0] != "--model" {
		t.Fatalf("expected AgentStart args [--model <model> <pointer>], got %v", args)
	}
	promptArg := args[2]
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
	for _, want := range []string{"org-a", "seat-1", "reviewer", "internal/org/**", ".claude/rules/agent-messaging.md"} {
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

	promptArg := h.agentStartArgs[0][2]
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
	if len(args) != 3 || args[2] != "verbatim prompt" {
		t.Fatalf("expected AgentStart args [--model <model> verbatim prompt], got %v", args)
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
	if len(args) != 2 {
		t.Fatalf("expected AgentStart args [--model <model>] with no prompt element, got %v", args)
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

func TestOrgSpawn_NoScope_SpawnedEventDetailsEmpty(t *testing.T) {
	// Unchanged-behavior guard: a spawn with no Scope leaves Details empty
	// on the spawned event, exactly as before Scope existed.
	o, _, _ := testOrg(t)

	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	if last.Details != "" {
		t.Fatalf("expected empty Details on the spawned event with no Scope, got %q", last.Details)
	}
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
