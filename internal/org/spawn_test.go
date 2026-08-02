package org

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
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
	paneSendKeysErr    error

	workspaceID string
	paneID      string

	sendKeysCalls    []string // paneIDs PaneSendKeys was invoked with, in order
	agentStartNames  []string // agent names AgentStart was invoked with, in order
	agentWaitTargets []string // targets AgentWait was invoked with, in order
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

func (f *fakeHerdr) AgentStart(_ context.Context, name, _, _ string, _ int, _ []string) (string, error) {
	f.calls = append(f.calls, "agent_start")
	f.agentStartNames = append(f.agentStartNames, name)
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

// fakeAgmsg is a call-recording, in-memory AgmsgClient.
type fakeAgmsg struct {
	calls   []string
	sendErr error
}

func (f *fakeAgmsg) Send(_ context.Context, _, _, _, _ string) error {
	f.calls = append(f.calls, "send")
	return f.sendErr
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

	want := []string{EventSpawnStarted, EventOrgWorkspaceCreated, EventSpawnStep, EventSpawnStep, EventSpawnStep, EventSpawned}
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
	if len(a.calls) != 1 || a.calls[0] != "send" {
		t.Fatalf("expected exactly one agmsg send call, got %v", a.calls)
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
	want := []string{EventSpawnStarted, EventSpawnFailed, EventSpawnStarted, EventOrgWorkspaceCreated, EventSpawnStep, EventSpawnStep, EventSpawnStep, EventSpawned}
	if len(got) != len(want) {
		t.Fatalf("expected event sequence %v, got %v", want, got)
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
	want := []string{EventSpawnStarted, EventSpawnFailed, EventSpawnStarted, EventOrgWorkspaceCreated, EventSpawnStep, EventSpawnStep, EventSpawnStep, EventSpawned}
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
	assertDetailsContains(t, last.Details, "step=agmsg_announce")
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
