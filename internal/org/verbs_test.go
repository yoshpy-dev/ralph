package org

import (
	"errors"
	"strings"
	"testing"
)

// TestOrgStop_UnknownSeat_ErrorsWithoutAppendingEvent is the AC-10 regression
// this slice closes: a seat that was never spawned (never appears in the
// manifest roster at all) must return an error and append NO manifest
// event -- so `stop --seat <unknown>` can never fabricate a phantom
// `stopped` state event.
func TestOrgStop_UnknownSeat_ErrorsWithoutAppendingEvent(t *testing.T) {
	o, h, a := testOrg(t)

	rrBefore, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	eventsBefore := len(rrBefore.Events)

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "never-spawned"})
	if result.Err == nil {
		t.Fatal("expected non-nil Err for stop on an unknown seat (CLI must exit non-zero)")
	}
	if len(h.calls) != 0 || len(a.calls) != 0 {
		t.Fatalf("expected no driver calls for an unknown seat, got herdr=%v agmsg=%v", h.calls, a.calls)
	}

	rrAfter, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if len(rrAfter.Events) != eventsBefore {
		t.Fatalf("expected no new manifest event for an unknown-seat stop, %d -> %d (events=%v)", eventsBefore, len(rrAfter.Events), rrAfter.Events)
	}
}

// TestOrgStop_UnknownSeat_DryRun_AlsoErrorsWithoutAppendingEvent asserts the
// existing-seat precondition applies identically to --dry-run stop: the
// manifest lookup must happen before the (skipped) real driver calls, not
// only in the non-dry-run path.
func TestOrgStop_UnknownSeat_DryRun_AlsoErrorsWithoutAppendingEvent(t *testing.T) {
	o, _, _ := testOrg(t)

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "never-spawned", DryRun: true})
	if result.Err == nil {
		t.Fatal("expected non-nil Err for a dry-run stop on an unknown seat")
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if len(rr.Events) != 0 {
		t.Fatalf("expected no manifest event for an unknown-seat dry-run stop, got %v", rr.Events)
	}
}

// TestOrgStop_ExistingSeat_RecordsPaneAndLeaveOutcomes covers AC-5: Stop on
// a real, existing seat best-effort-calls both PaneSendKeys(C-c) and
// Agmsg.Leave, and records both outcomes in the stopped event's Details --
// including when Leave itself fails, so status stays truthful without the
// verb itself failing. It also covers the live-smoke follow-up fix: the
// stopped event must carry the seat's Role/Driver/Model forward so `status`
// after stop does not show blank columns for a stopped seat.
func TestOrgStop_ExistingSeat_RecordsPaneAndLeaveOutcomes(t *testing.T) {
	o, _, a := testOrg(t)
	a.leaveErr = errors.New("stub failure: leave")

	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected nil Err: leave failure is best-effort and must not fail Stop, got %v", result.Err)
	}

	if len(a.leaveCalls) != 1 {
		t.Fatalf("expected exactly one Leave call, got %+v", a.leaveCalls)
	}
	if a.leaveCalls[0].agentID != "seat-1" {
		t.Errorf("expected Leave(team, seat-1), got %+v", a.leaveCalls[0])
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	if last.Event != EventStopped {
		t.Fatalf("expected last event stopped, got %q", last.Event)
	}
	assertDetailsContains(t, last.Details, "pane=ok", "leave=failed", "stub failure: leave")
	if last.Role != "worker" || last.Driver != "claude" || last.Model != "sonnet" {
		t.Errorf("expected stopped event to carry seat role/driver/model, got role=%q driver=%q model=%q", last.Role, last.Driver, last.Model)
	}

	statusResult, err := o.Status("org-a", false)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(statusResult.Seats) != 1 || statusResult.Seats[0].Active {
		t.Fatalf("expected seat-1 inactive after stop despite leave failure, got %+v", statusResult.Seats)
	}
	if got := statusResult.Seats[0]; got.Role != "worker" || got.Driver != "claude" || got.Model != "sonnet" {
		t.Errorf("expected status to still show seat role/driver/model after stop, got %+v", got)
	}
}

// TestOrgDisband_OnlyStopsExistingActiveSeats_UnknownNeverAppears is a
// structural regression check: Disband iterates the manifest roster (never
// an externally-supplied seat list), so it inherently cannot process a
// phantom seat_id that was never spawned. An org with no seats at all still
// gets a terminal `disbanded` event (unchanged, harmless no-op behavior).
func TestOrgDisband_OnlyStopsExistingActiveSeats_UnknownNeverAppears(t *testing.T) {
	o, h, a := testOrg(t)

	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-1 failed: %+v", r)
	}
	// seat-2 is spawned then already stopped -- Disband must not attempt to
	// stop it a second time (it's no longer Active).
	if r := o.Spawn(mustSpawnParams("org-a", "seat-2")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn seat-2 failed: %+v", r)
	}
	if r := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-2"}); r.Err != nil {
		t.Fatalf("pre-stop seat-2 failed: %v", r.Err)
	}

	sendKeysBefore := len(h.sendKeysCalls)
	leaveBefore := len(a.leaveCalls)

	result := o.Disband(DisbandParams{OrgID: "org-a"})
	if len(result.Errs) != 0 {
		t.Fatalf("expected no errors from disband, got %v", result.Errs)
	}
	if len(result.StoppedSeats) != 1 || result.StoppedSeats[0] != "seat-1" {
		t.Fatalf("expected disband to stop only the still-active seat-1, got %v", result.StoppedSeats)
	}
	if got := len(h.sendKeysCalls) - sendKeysBefore; got != 1 {
		t.Fatalf("expected exactly 1 new PaneSendKeys call (only seat-1 was active), got %d", got)
	}
	if got := len(a.leaveCalls) - leaveBefore; got != 1 {
		t.Fatalf("expected exactly 1 new Leave call (only seat-1 was active), got %d", got)
	}

	statusResult, err := o.Status("org-a", false)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	for _, s := range statusResult.Seats {
		if s.Active {
			t.Errorf("expected no seat to remain active after disband, got %+v", s)
		}
	}
}

// TestOrgDisband_EmptyOrg_StillAppendsDisbandedEvent documents the current,
// unchanged choice: an org_id with no seats at all still gets a terminal
// `disbanded` event -- a harmless no-op that keeps Disband idempotent-safe
// to call even before any spawn.
func TestOrgDisband_EmptyOrg_StillAppendsDisbandedEvent(t *testing.T) {
	o, _, _ := testOrg(t)

	result := o.Disband(DisbandParams{OrgID: "org-empty"})
	if len(result.Errs) != 0 {
		t.Fatalf("expected no errors disbanding an org with no seats, got %v", result.Errs)
	}
	if len(result.StoppedSeats) != 0 {
		t.Fatalf("expected no stopped seats for an empty org, got %v", result.StoppedSeats)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if len(rr.Events) != 1 || rr.Events[0].Event != EventDisbanded {
		t.Fatalf("expected exactly one disbanded event, got %v", rr.Events)
	}
}

// TestOrgSend_Malformed_RejectedNoManifestEventNoDriverCall covers AC-11:
// a message that fails protocol.ValidateText is rejected before any
// manifest event is appended and before any driver call (AgentWait/
// PaneSendText/PaneSendKeys) is attempted.
func TestOrgSend_Malformed_RejectedNoManifestEventNoDriverCall(t *testing.T) {
	o, h, a := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	eventsBefore := len(mustReadEvents(t, o))
	herdrCallsBefore := len(h.calls)
	agmsgCallsBefore := len(a.calls)

	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: "not a valid protocol message"})
	if result.Err == nil {
		t.Fatal("expected a non-nil Err for a malformed message (CLI must exit non-zero)")
	}

	if got := len(mustReadEvents(t, o)); got != eventsBefore {
		t.Fatalf("expected no new manifest event for a rejected send, %d -> %d", eventsBefore, got)
	}
	if len(h.calls) != herdrCallsBefore || len(a.calls) != agmsgCallsBefore {
		t.Fatalf("expected no new driver calls for a rejected send, herdr %d->%d agmsg %d->%d",
			herdrCallsBefore, len(h.calls), agmsgCallsBefore, len(a.calls))
	}
}

// TestOrgSend_DryRun_Malformed_AlsoRejectedBeforeManifestEvent asserts
// protocol validation runs for --dry-run sends too (it is a pure,
// side-effect-free check that should not be skipped just because no real
// driver call would happen anyway).
func TestOrgSend_DryRun_Malformed_AlsoRejectedBeforeManifestEvent(t *testing.T) {
	o, _, _ := testOrg(t)

	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: "not a valid protocol message", DryRun: true})
	if result.Err == nil {
		t.Fatal("expected a non-nil Err for a malformed dry-run message")
	}
	if len(mustReadEvents(t, o)) != 0 {
		t.Fatalf("expected no manifest event for a rejected dry-run send, got %v", mustReadEvents(t, o))
	}
}

// TestOrgSend_ValidTypedMessage_PassesAndDrivesRealCalls asserts a
// well-formed typed message is accepted: Send proceeds to the real
// AgentWait/PaneSendText/PaneSendKeys sequence and appends a `sent` event.
func TestOrgSend_ValidTypedMessage_PassesAndDrivesRealCalls(t *testing.T) {
	o, h, _ := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg})
	if result.Err != nil {
		t.Fatalf("expected a well-formed typed message to pass, got %v", result.Err)
	}

	if len(h.agentWaitTargets) != 1 || h.agentWaitTargets[0] != herdrAgentName("org-a", "seat-1") {
		t.Fatalf("expected exactly one AgentWait call for the target seat, got %v", h.agentWaitTargets)
	}

	events := mustReadEvents(t, o)
	last := events[len(events)-1]
	if last.Event != EventSent {
		t.Fatalf("expected last event sent, got %q", last.Event)
	}
	if strings.HasPrefix(last.Details, "raw=true") {
		t.Fatalf("expected a non-raw send to not carry the raw=true marker, got %q", last.Details)
	}
}

// TestOrgSend_Raw_BypassesValidation_RecordsRawInDetails covers the --raw
// escape hatch: an otherwise-malformed message is accepted, and the `sent`
// event's Details records that the bypass was used.
func TestOrgSend_Raw_BypassesValidation_RecordsRawInDetails(t *testing.T) {
	o, h, _ := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: "not a valid protocol message", Raw: true})
	if result.Err != nil {
		t.Fatalf("expected --raw to bypass validation, got %v", result.Err)
	}
	if len(h.agentWaitTargets) != 1 {
		t.Fatalf("expected the raw send to still drive the real AgentWait call, got %v", h.agentWaitTargets)
	}

	events := mustReadEvents(t, o)
	last := events[len(events)-1]
	if last.Event != EventSent {
		t.Fatalf("expected last event sent, got %q", last.Event)
	}
	assertDetailsContains(t, last.Details, "raw=true")
}

// mustReadEvents is a small helper shared by the Send tests above (distinct
// from eventNames in spawn_test.go, which returns only event type names --
// these tests need the full events for Details assertions in some cases).
func mustReadEvents(t *testing.T, o *Org) []ManifestEvent {
	t.Helper()
	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	return rr.Events
}

// TestOrgSend_DryRun_ValidMessage_AppendsEventWithoutDriverCalls is the
// positive counterpart of TestOrgSend_DryRun_Malformed...: a well-formed
// typed message with DryRun set still runs protocol validation (and
// passes), appends the (dry_run: true) `sent` event, and never reaches any
// real driver call (AgentWait/PaneSendText/PaneSendKeys).
func TestOrgSend_DryRun_ValidMessage_AppendsEventWithoutDriverCalls(t *testing.T) {
	o, h, a := testOrg(t)

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg, DryRun: true})
	if result.Err != nil {
		t.Fatalf("expected a well-formed dry-run message to pass, got %v", result.Err)
	}
	if len(h.calls) != 0 || len(a.calls) != 0 {
		t.Fatalf("expected no driver calls for a dry-run send, got herdr=%v agmsg=%v", h.calls, a.calls)
	}

	events := mustReadEvents(t, o)
	last := events[len(events)-1]
	if last.Event != EventSent || !last.DryRun {
		t.Fatalf("expected a dry-run sent event, got %+v", last)
	}
}

// TestOrgSend_UnknownSeat_ErrorsWithoutDriverCall covers Send's seat-lookup
// gate: a target seat that never appears in the manifest roster is
// rejected before any driver call, distinct from the protocol-validation
// rejection tests above.
func TestOrgSend_UnknownSeat_ErrorsWithoutDriverCall(t *testing.T) {
	o, h, a := testOrg(t)

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	result := o.Send(SendParams{OrgID: "org-a", To: "never-spawned", Text: msg})
	if result.Err == nil {
		t.Fatal("expected a non-nil Err sending to an unknown seat")
	}
	if len(h.calls) != 0 || len(a.calls) != 0 {
		t.Fatalf("expected no driver calls sending to an unknown seat, got herdr=%v agmsg=%v", h.calls, a.calls)
	}
	if len(mustReadEvents(t, o)) != 0 {
		t.Fatalf("expected no manifest event sending to an unknown seat, got %v", mustReadEvents(t, o))
	}
}

// TestOrgSend_SeatWithoutPaneID_Errors covers the "seat exists in the
// manifest but has no pane_id recorded" branch. Constructed directly via
// appendEvent (a state event with PaneID left blank) rather than through
// Spawn, which always records a pane_id by the time it reaches the spawned
// state -- this is the same seeding technique TestOrgRead_SeatWithoutPaneID_Errors
// uses below.
func TestOrgSend_SeatWithoutPaneID_Errors(t *testing.T) {
	o, h, a := testOrg(t)
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: "org-a", SeatID: "seat-nopane", Event: EventSpawned,
	}); err != nil {
		t.Fatalf("seed manifest event: %v", err)
	}

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-nopane", Text: msg})
	if result.Err == nil {
		t.Fatal("expected a non-nil Err sending to a seat with no pane_id recorded")
	}
	if !strings.Contains(result.Err.Error(), "no pane_id recorded") {
		t.Errorf("expected the error to mention no pane_id recorded, got %v", result.Err)
	}
	if len(h.calls) != 0 || len(a.calls) != 0 {
		t.Fatalf("expected no driver calls for a paneless seat, got herdr=%v agmsg=%v", h.calls, a.calls)
	}
}

// TestOrgSend_PrefersRecordedHerdrAgentName_OverDerivedConvention pins AC-8:
// Send must target the seat's persisted HerdrAgentName (from the `spawned`
// event) rather than re-deriving it via herdrAgentName, so a future change
// to herdrAgentName's own naming convention cannot orphan an already-spawned
// seat. Seeded with a deliberately different recorded name than
// herdrAgentName("org-a", "seat-1") would produce, so the assertion cannot
// pass by coincidence.
func TestOrgSend_PrefersRecordedHerdrAgentName_OverDerivedConvention(t *testing.T) {
	o, h, _ := testOrg(t)
	const recordedName = "org-a--seat-1-under-a-hypothetical-future-convention"
	if recordedName == herdrAgentName("org-a", "seat-1") {
		t.Fatal("test fixture bug: recordedName must differ from the derived convention")
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: "org-a", SeatID: "seat-1", Event: EventSpawned,
		PaneID: "pane-1", HerdrAgentName: recordedName,
	}); err != nil {
		t.Fatalf("seed manifest event: %v", err)
	}

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg})
	if result.Err != nil {
		t.Fatalf("expected Send to succeed, got %v", result.Err)
	}
	if len(h.agentWaitTargets) != 1 || h.agentWaitTargets[0] != recordedName {
		t.Fatalf("expected AgentWait to target the recorded herdr_agent_name %q, got %v", recordedName, h.agentWaitTargets)
	}
}

// TestOrgSend_LegacySpawnedEvent_FallsBackToDerivedHerdrAgentName is the
// compat counterpart: a `spawned` event recorded before HerdrAgentName
// existed (the JSON field simply absent, decoding to "") must still resolve
// via the herdrAgentName derivation, exactly as it did before this field was
// added.
func TestOrgSend_LegacySpawnedEvent_FallsBackToDerivedHerdrAgentName(t *testing.T) {
	o, h, _ := testOrg(t)
	// No HerdrAgentName field set -- simulates an event written by a
	// pre-AC-8 build (the JSON key is simply absent on disk; Go's zero
	// value for an unset string field already models this).
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: "org-a", SeatID: "seat-1", Event: EventSpawned,
		PaneID: "pane-1",
	}); err != nil {
		t.Fatalf("seed legacy manifest event: %v", err)
	}

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg})
	if result.Err != nil {
		t.Fatalf("expected Send to succeed against a legacy event, got %v", result.Err)
	}
	want := herdrAgentName("org-a", "seat-1")
	if len(h.agentWaitTargets) != 1 || h.agentWaitTargets[0] != want {
		t.Fatalf("expected AgentWait to fall back to the derived name %q for a legacy event, got %v", want, h.agentWaitTargets)
	}
}

// TestResolvedHerdrAgentName_EmptyVsSet is a direct unit pin for the helper
// itself, independent of Send's plumbing.
func TestResolvedHerdrAgentName_EmptyVsSet(t *testing.T) {
	recorded := SeatStatus{OrgID: "org-a", SeatID: "seat-1", HerdrAgentName: "explicit-name"}
	if got := resolvedHerdrAgentName(recorded); got != "explicit-name" {
		t.Errorf("resolvedHerdrAgentName with HerdrAgentName set = %q, want %q", got, "explicit-name")
	}

	legacy := SeatStatus{OrgID: "org-a", SeatID: "seat-1"}
	if got, want := resolvedHerdrAgentName(legacy), herdrAgentName("org-a", "seat-1"); got != want {
		t.Errorf("resolvedHerdrAgentName with HerdrAgentName unset = %q, want derived %q", got, want)
	}
}

// TestOrgSpawn_SpawnedEventRecordsHerdrAgentName pins the write side of
// AC-8: a real Spawn's `spawned` event must persist HerdrAgentName, matching
// herdrAgentName's derivation.
func TestOrgSpawn_SpawnedEventRecordsHerdrAgentName(t *testing.T) {
	o, _, _ := testOrg(t)
	result := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if result.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected SpawnOutcomeSpawned, got %+v", result)
	}
	want := herdrAgentName("org-a", "seat-1")
	if result.Seat.HerdrAgentName != want {
		t.Errorf("SpawnResult.Seat.HerdrAgentName = %q, want %q", result.Seat.HerdrAgentName, want)
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var spawnedEvent *ManifestEvent
	for i := range rr.Events {
		if rr.Events[i].Event == EventSpawned {
			spawnedEvent = &rr.Events[i]
		}
	}
	if spawnedEvent == nil {
		t.Fatalf("expected a spawned event, got %+v", rr.Events)
	}
	if spawnedEvent.HerdrAgentName != want {
		t.Errorf("spawned event HerdrAgentName = %q, want %q", spawnedEvent.HerdrAgentName, want)
	}
}

// TestOrgStop_DryRun_ExistingSeat_RecordsDryRunDetails covers the
// DryRun-on-an-existing-seat branch of Stop, distinct from the
// DryRun-on-an-unknown-seat tests above: with a real, spawned seat on
// record, --dry-run must still skip both driver calls (pane C-c and agmsg
// Leave) and record the dry-run marker in Details.
func TestOrgStop_DryRun_ExistingSeat_RecordsDryRunDetails(t *testing.T) {
	o, h, a := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	sendKeysBefore := len(h.sendKeysCalls)
	leaveBefore := len(a.leaveCalls)

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1", DryRun: true})
	if result.Err != nil {
		t.Fatalf("expected nil Err for a dry-run stop on an existing seat, got %v", result.Err)
	}
	if len(h.sendKeysCalls) != sendKeysBefore || len(a.leaveCalls) != leaveBefore {
		t.Fatalf("expected no driver calls for a dry-run stop, herdr sendKeys %d->%d agmsg leave %d->%d",
			sendKeysBefore, len(h.sendKeysCalls), leaveBefore, len(a.leaveCalls))
	}

	events := mustReadEvents(t, o)
	last := events[len(events)-1]
	if last.Event != EventStopped || !last.DryRun {
		t.Fatalf("expected a dry-run stopped event, got %+v", last)
	}
	if last.Details != "dry-run: no driver call" {
		t.Errorf("expected the dry-run details marker, got %q", last.Details)
	}
}

// TestOrgStop_ExistingSeat_NoPaneOrAgmsgTeam_RecordsSkippedNotes covers the
// paneID=="" and agmsgTeam=="" branches of Stop's non-dry-run path -- a
// seat recorded without either external id (seeded directly via
// appendEvent, since Spawn always records both once it reaches spawned)
// must record "skipped" notes for both instead of attempting a driver call
// for either.
func TestOrgStop_ExistingSeat_NoPaneOrAgmsgTeam_RecordsSkippedNotes(t *testing.T) {
	o, h, a := testOrg(t)
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: "org-a", SeatID: "seat-bare", Event: EventSpawned,
	}); err != nil {
		t.Fatalf("seed manifest event: %v", err)
	}

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-bare"})
	if result.Err != nil {
		t.Fatalf("expected nil Err, got %v", result.Err)
	}
	if len(h.sendKeysCalls) != 0 || len(a.leaveCalls) != 0 {
		t.Fatalf("expected no driver calls for a seat with no pane_id/agmsg_team, got herdr=%v agmsg=%v", h.sendKeysCalls, a.leaveCalls)
	}

	events := mustReadEvents(t, o)
	last := events[len(events)-1]
	assertDetailsContains(t, last.Details, "pane=no pane_id on record", "leave=skipped: no agmsg_team on record")
}

// TestOrgWait_HappyPath_ReturnsHerdrOutputAndTargetsNamespacedAgent covers
// the happy path for Wait: it targets the org_id-namespaced herdr agent
// name (herdrAgentName), returns herdr's raw output verbatim, and never
// writes a manifest event -- Wait is a pure passthrough to
// Herdr.AgentWait, per its doc comment.
func TestOrgWait_HappyPath_ReturnsHerdrOutputAndTargetsNamespacedAgent(t *testing.T) {
	o, h, _ := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	eventsBefore := len(mustReadEvents(t, o))

	result := o.Wait(WaitParams{OrgID: "org-a", Seat: "seat-1", Until: []string{"idle"}})
	if result.Err != nil {
		t.Fatalf("expected nil Err, got %v", result.Err)
	}
	if result.Output != "idle" {
		t.Fatalf("expected herdr's raw output %q, got %q", "idle", result.Output)
	}
	if len(h.agentWaitTargets) != 1 || h.agentWaitTargets[0] != herdrAgentName("org-a", "seat-1") {
		t.Fatalf("expected AgentWait to target the namespaced agent name, got %v", h.agentWaitTargets)
	}
	if got := len(mustReadEvents(t, o)); got != eventsBefore {
		t.Fatalf("expected Wait to never write a manifest event, %d -> %d", eventsBefore, got)
	}
}

// TestOrgWait_WithTimeoutMS_UsesTimeoutContext exercises the p.TimeoutMS >
// 0 branch (context.WithTimeout), distinct from the zero-timeout branch
// exercised above.
func TestOrgWait_WithTimeoutMS_UsesTimeoutContext(t *testing.T) {
	o, h, _ := testOrg(t)

	result := o.Wait(WaitParams{OrgID: "org-a", Seat: "seat-1", Until: []string{"idle"}, TimeoutMS: 5000})
	if result.Err != nil {
		t.Fatalf("expected nil Err, got %v", result.Err)
	}
	if len(h.agentWaitTargets) != 1 {
		t.Fatalf("expected exactly one AgentWait call, got %v", h.agentWaitTargets)
	}
}

// TestOrgWait_UnknownSeat_StillDrivesHerdr_NoManifestCheck documents the
// deliberate design captured in Wait's doc comment: Wait never touches the
// manifest, so it does not distinguish an unrecorded seat from a known one
// -- it just asks herdr directly for the namespaced agent name. This is
// not a defect: Send/Read (which act on manifest-recorded pane state) are
// the verbs that reject unknown seats; Wait intentionally does not.
func TestOrgWait_UnknownSeat_StillDrivesHerdr_NoManifestCheck(t *testing.T) {
	o, h, _ := testOrg(t)

	result := o.Wait(WaitParams{OrgID: "org-a", Seat: "never-spawned", Until: []string{"idle"}})
	if result.Err != nil {
		t.Fatalf("expected Wait to pass through to herdr even for an unrecorded seat, got %v", result.Err)
	}
	if len(h.agentWaitTargets) != 1 || h.agentWaitTargets[0] != herdrAgentName("org-a", "never-spawned") {
		t.Fatalf("expected AgentWait called with the namespaced agent name regardless of manifest state, got %v", h.agentWaitTargets)
	}
}

// TestOrgRead_HappyPath_ReturnsHerdrPaneOutput covers Read's happy path:
// resolve the seat's pane_id from the manifest, then return herdr's raw
// PaneRead output verbatim.
func TestOrgRead_HappyPath_ReturnsHerdrPaneOutput(t *testing.T) {
	o, h, _ := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	result := o.Read(ReadParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected nil Err, got %v", result.Err)
	}
	if result.Output != "pane output" {
		t.Fatalf("expected herdr's raw pane output, got %q", result.Output)
	}
	if len(h.calls) == 0 || h.calls[len(h.calls)-1] != "pane_read" {
		t.Fatalf("expected the last herdr call to be pane_read, got %v", h.calls)
	}
}

// TestOrgRead_DefaultLines_UsesDefaultReadLinesWhenUnset covers the
// lines<=0 branch, which substitutes defaultReadLines.
func TestOrgRead_DefaultLines_UsesDefaultReadLinesWhenUnset(t *testing.T) {
	o, _, _ := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	result := o.Read(ReadParams{OrgID: "org-a", Seat: "seat-1", Lines: 0})
	if result.Err != nil {
		t.Fatalf("expected nil Err for the default-lines branch, got %v", result.Err)
	}
}

// TestOrgRead_UnknownSeat_ErrorsWithoutDriverCall covers Read's seat-lookup
// gate: a seat that never appears in the manifest roster is rejected
// before any herdr call.
func TestOrgRead_UnknownSeat_ErrorsWithoutDriverCall(t *testing.T) {
	o, h, _ := testOrg(t)

	result := o.Read(ReadParams{OrgID: "org-a", Seat: "never-spawned"})
	if result.Err == nil {
		t.Fatal("expected a non-nil Err reading an unknown seat")
	}
	if len(h.calls) != 0 {
		t.Fatalf("expected no herdr calls for an unknown seat, got %v", h.calls)
	}
}

// TestOrgRead_SeatWithoutPaneID_Errors covers the "seat exists in the
// manifest but has no pane_id recorded" branch -- constructed directly via
// appendEvent (a state event with PaneID left blank) rather than through
// Spawn, which always records a pane_id once it reaches the spawned state.
func TestOrgRead_SeatWithoutPaneID_Errors(t *testing.T) {
	o, h, _ := testOrg(t)
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: "org-a", SeatID: "seat-nopane", Event: EventSpawned,
	}); err != nil {
		t.Fatalf("seed manifest event: %v", err)
	}

	result := o.Read(ReadParams{OrgID: "org-a", Seat: "seat-nopane"})
	if result.Err == nil {
		t.Fatal("expected a non-nil Err for a seat with no pane_id recorded")
	}
	if !strings.Contains(result.Err.Error(), "no pane_id recorded") {
		t.Errorf("expected the error to mention no pane_id recorded, got %v", result.Err)
	}
	if len(h.calls) != 0 {
		t.Fatalf("expected no herdr calls for a paneless seat, got %v", h.calls)
	}
}
