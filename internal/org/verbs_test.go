package org

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
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
	if result.Progress != SendProgressEnterPressed {
		t.Errorf("expected Progress SendProgressEnterPressed on success, got %v", result.Progress)
	}

	// Two AgentWait calls: the idle/done wait before send-text, and the
	// post-Enter working/blocked confirm wait -- both target the same
	// resolved seat name.
	want := herdrAgentName("org-a", "seat-1")
	if len(h.agentWaitTargets) != 2 || h.agentWaitTargets[0] != want || h.agentWaitTargets[1] != want {
		t.Fatalf("expected exactly two AgentWait calls (idle/done wait + submit confirm) for the target seat, got %v", h.agentWaitTargets)
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
	if len(h.agentWaitTargets) != 2 {
		t.Fatalf("expected the raw send to still drive the real AgentWait calls (idle/done wait + submit confirm), got %v", h.agentWaitTargets)
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
// real driver call (AgentWait/PaneSendText/PaneSendKeys) -- so DryRun never
// waits the enter delay and never runs the submit confirmation either,
// leaving SubmitConfirmed false (edge case 7: dry-run skips both).
func TestOrgSend_DryRun_ValidMessage_AppendsEventWithoutDriverCalls(t *testing.T) {
	o, h, a := testOrg(t)

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg, DryRun: true})
	if result.Err != nil {
		t.Fatalf("expected a well-formed dry-run message to pass, got %v", result.Err)
	}
	if result.SubmitConfirmed {
		t.Error("expected SubmitConfirmed false for a dry-run send (no confirmation ever attempted)")
	}
	if result.Progress != SendProgressNothingSent {
		t.Errorf("expected Progress SendProgressNothingSent for a dry-run send (no pane call is ever attempted), got %v", result.Progress)
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
	if len(h.agentWaitTargets) != 2 || h.agentWaitTargets[0] != recordedName || h.agentWaitTargets[1] != recordedName {
		t.Fatalf("expected both AgentWait calls to target the recorded herdr_agent_name %q, got %v", recordedName, h.agentWaitTargets)
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
	if len(h.agentWaitTargets) != 2 || h.agentWaitTargets[0] != want || h.agentWaitTargets[1] != want {
		t.Fatalf("expected both AgentWait calls to fall back to the derived name %q for a legacy event, got %v", want, h.agentWaitTargets)
	}
}

// TestOrgSend_Confirmed_ExactCallOrderAndUntilStates covers AC-1: the exact
// driver call order Send must follow (idle/done wait -> send-text -> Enter
// -> working/blocked confirm wait) and the exact `until` states each
// AgentWait call carries. h.calls already holds Spawn's own
// workspace_create/tab_create/agent_start entries, so only the tail added
// by Send itself is compared.
func TestOrgSend_Confirmed_ExactCallOrderAndUntilStates(t *testing.T) {
	o, h, _ := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	callsBefore := len(h.calls)

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg})
	if result.Err != nil {
		t.Fatalf("expected Send to succeed, got %v", result.Err)
	}
	if !result.SubmitConfirmed {
		t.Error("expected SubmitConfirmed true when the post-Enter AgentWait succeeds")
	}

	wantOrder := []string{"agent_wait", "pane_send_text", "pane_send_keys", "agent_wait"}
	if got := h.calls[callsBefore:]; !slices.Equal(got, wantOrder) {
		t.Fatalf("expected call order %v, got %v", wantOrder, got)
	}
	if len(h.agentWaitUntil) != 2 {
		t.Fatalf("expected exactly 2 AgentWait calls, got %d (%v)", len(h.agentWaitUntil), h.agentWaitUntil)
	}
	if !slices.Equal(h.agentWaitUntil[0], []string{"idle", "done"}) {
		t.Errorf("expected the first AgentWait to wait on idle/done, got %v", h.agentWaitUntil[0])
	}
	if !slices.Equal(h.agentWaitUntil[1], []string{"working", "blocked"}) {
		t.Errorf("expected the second (confirm) AgentWait to wait on working/blocked, got %v", h.agentWaitUntil[1])
	}

	events := mustReadEvents(t, o)
	last := events[len(events)-1]
	if strings.Contains(last.Details, "submit_unconfirmed") {
		t.Errorf("expected no submit_unconfirmed marker in Details when confirmed, got %q", last.Details)
	}
}

// TestOrgSend_UnconfirmedSubmit_NeverResendsEnter is the AC-2 regression for
// the no-resend design decision (plan "Design decisions", "Enter の自動再送
// はしない"): when the post-Enter confirm AgentWait fails, Send still
// succeeds, reports SubmitConfirmed=false, records submit_unconfirmed=true
// on the `sent` event -- and, critically, PaneSendKeys was invoked exactly
// once, with exactly ["Enter"]. A blind second Enter here could confirm an
// approval dialog the seat already advanced to after a submit Send failed
// to observe (see confirmSubmitted's doc comment) -- so this pins that no
// resend ever happens, no matter what the confirm wait reports.
func TestOrgSend_UnconfirmedSubmit_NeverResendsEnter(t *testing.T) {
	o, h, _ := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	// First AgentWait call (idle/done wait) succeeds; second (the confirm
	// wait) fails.
	h.agentWaitErrs = []error{nil, errors.New("stub: confirm wait failed")}

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg})
	if result.Err != nil {
		t.Fatalf("expected Send to succeed even when the submit cannot be confirmed, got %v", result.Err)
	}
	if result.SubmitConfirmed {
		t.Error("expected SubmitConfirmed false when the confirm AgentWait fails")
	}

	if len(h.sendKeysKeys) != 1 {
		t.Fatalf("expected PaneSendKeys to be called exactly once, got %d calls: %v", len(h.sendKeysKeys), h.sendKeysKeys)
	}
	if !slices.Equal(h.sendKeysKeys[0], []string{"Enter"}) {
		t.Fatalf("expected the single PaneSendKeys call to send exactly [\"Enter\"], got %v", h.sendKeysKeys[0])
	}

	events := mustReadEvents(t, o)
	last := events[len(events)-1]
	if !strings.HasPrefix(last.Details, "submit_unconfirmed=true ") {
		t.Fatalf("expected Details to start with submit_unconfirmed=true, got %q", last.Details)
	}
}

// TestOrgSend_IdleDoneWaitFails_ErrorsBeforeAnyTypingOrEvent covers the
// first AgentWait call (waiting for the seat to go idle/done before typing
// anything) failing -- the mirror image of
// TestOrgSend_UnconfirmedSubmit_NeverResendsEnter, which covers the second
// (confirm) AgentWait failing. Unlike the confirm wait, an idle/done wait
// failure means Send never reaches PaneSendText at all: nothing is typed,
// no Enter is sent, and no `sent` event is appended. This path had no
// direct coverage before (found via go tool cover -func on verbs.go).
func TestOrgSend_IdleDoneWaitFails_ErrorsBeforeAnyTypingOrEvent(t *testing.T) {
	o, h, _ := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	h.agentWaitErrs = []error{errors.New("stub: idle/done wait failed")}
	eventsBefore := len(mustReadEvents(t, o))
	callsBefore := len(h.calls)

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg})
	if result.Err == nil {
		t.Fatal("expected a non-nil Err when the idle/done AgentWait fails")
	}
	if !strings.Contains(result.Err.Error(), "wait for seat") {
		t.Errorf("expected the error to name the idle/done wait, got %v", result.Err)
	}
	if result.Progress != SendProgressNothingSent {
		t.Errorf("expected Progress SendProgressNothingSent: PaneSendText is never reached when the idle/done wait fails, got %v", result.Progress)
	}
	if result.PaneID != "" {
		t.Errorf("expected empty PaneID: PaneSendText was never attempted, got %q", result.PaneID)
	}
	if got, want := h.calls[callsBefore:], []string{"agent_wait"}; !slices.Equal(got, want) {
		t.Fatalf("expected only the idle/done AgentWait call after Send, got %v", got)
	}
	if got := len(mustReadEvents(t, o)); got != eventsBefore {
		t.Fatalf("expected no new manifest event when the idle/done wait fails, %d -> %d", eventsBefore, got)
	}
}

// TestSendProgress_String_Table is a direct unit pin for SendProgress's own
// String method -- it had zero test coverage before this (found via
// go tool cover -func on internal/org/verbs.go): every %v/%q formatting of
// a SendProgress value inside the existing tests' t.Errorf calls happens on
// the ARGUMENT side of an Errorf that never actually fires (the tests all
// pass before reaching that format verb), so the Stringer method itself was
// never exercised. Covers all five named states plus the out-of-range
// default branch (an out-of-range value cannot occur through Send's own
// code today -- SendProgress is only ever set to one of the five named
// constants -- but the default branch exists specifically so a value that
// somehow got out of range still prints something grep-able instead of a
// silent zero-value/panic, and the doc comment promises "SendProgress(N)"
// for it, which nothing was checking).
func TestSendProgress_String_Table(t *testing.T) {
	tests := []struct {
		name string
		p    SendProgress
		want string
	}{
		{"nothing sent", SendProgressNothingSent, "nothing-sent"},
		{"text unacknowledged", SendProgressTextUnacknowledged, "text-unacknowledged"},
		{"text typed", SendProgressTextTyped, "text-typed"},
		{"enter unacknowledged", SendProgressEnterUnacknowledged, "enter-unacknowledged"},
		{"enter pressed", SendProgressEnterPressed, "enter-pressed"},
		{"out of range (positive)", SendProgress(99), "SendProgress(99)"},
		{"out of range (negative)", SendProgress(-1), "SendProgress(-1)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.p.String(); got != tc.want {
				t.Errorf("SendProgress(%d).String() = %q, want %q", int(tc.p), got, tc.want)
			}
		})
	}
}

// TestOrgSend_RawAndUnconfirmed_DetailsPrefixOrder pins the exact prefix
// order when both markers apply: "raw=true " comes first, then
// "submit_unconfirmed=true ", then the (possibly truncated) text -- the
// order this plan's Scope 3 chose so a human skimming Details sees the
// escape-hatch marker before the confirmation-state marker.
func TestOrgSend_RawAndUnconfirmed_DetailsPrefixOrder(t *testing.T) {
	o, h, _ := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	h.agentWaitErrs = []error{nil, errors.New("stub: confirm wait failed")}

	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: "not a valid protocol message", Raw: true})
	if result.Err != nil {
		t.Fatalf("expected --raw send to succeed, got %v", result.Err)
	}
	if result.SubmitConfirmed {
		t.Error("expected SubmitConfirmed false when the confirm AgentWait fails")
	}

	events := mustReadEvents(t, o)
	last := events[len(events)-1]
	wantPrefix := "raw=true submit_unconfirmed=true "
	if !strings.HasPrefix(last.Details, wantPrefix) {
		t.Fatalf("expected Details to start with %q, got %q", wantPrefix, last.Details)
	}
}

// TestOrgSend_EnterDelay_WaitsAtLeastConfiguredDuration asserts the wait
// between PaneSendText and PaneSendKeys is real: with Org.SendEnterDelay
// set to a small-but-measurable value, elapsed wall time around Send is at
// least that long. Only a lower bound is asserted (no upper bound) so this
// stays non-flaky under CI scheduling jitter.
func TestOrgSend_EnterDelay_WaitsAtLeastConfiguredDuration(t *testing.T) {
	o, _, _ := testOrg(t)
	o.SendEnterDelay = 30 * time.Millisecond
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	start := time.Now()
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg})
	elapsed := time.Since(start)
	if result.Err != nil {
		t.Fatalf("expected Send to succeed, got %v", result.Err)
	}
	if elapsed < o.SendEnterDelay {
		t.Errorf("expected Send to take at least the configured SendEnterDelay (%v), took %v", o.SendEnterDelay, elapsed)
	}
}

// TestOrgSend_EnterDelayMS_OverridesOrgSendEnterDelay asserts
// SendParams.EnterDelayMS, when > 0, wins over Org.SendEnterDelay for that
// one call -- the CLI's --enter-delay-ms escape hatch.
func TestOrgSend_EnterDelayMS_OverridesOrgSendEnterDelay(t *testing.T) {
	o, _, _ := testOrg(t)
	o.SendEnterDelay = time.Millisecond // would be far too short to observe below
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	const overrideDelay = 30 * time.Millisecond
	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	start := time.Now()
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg, EnterDelayMS: int(overrideDelay / time.Millisecond)})
	elapsed := time.Since(start)
	if result.Err != nil {
		t.Fatalf("expected Send to succeed, got %v", result.Err)
	}
	if elapsed < overrideDelay {
		t.Errorf("expected EnterDelayMS to override Org.SendEnterDelay and take at least %v, took %v", overrideDelay, elapsed)
	}
}

// TestOrgSend_BudgetTooSmallForEnterDelay_NothingTyped is the M2 fail-closed
// regression (self-review MEDIUM-2,
// docs/reports/self-review-2026-09-19-org-send-enter-timing.md): a
// --timeout-ms budget that (after the idle/done wait) cannot even fund the
// pre-Enter pause must be refused before anything is typed -- no
// PaneSendText call, no PaneSendKeys call, no `sent` event, Progress stays
// SendProgressNothingSent. Before M2, this exact TimeoutMS=5 /
// SendEnterDelay=200ms pairing used to type the text and only then hit ctx
// expiry inside waitOrCtxDone -- see
// TestOrgSend_CtxExpiresDuringEnterDelay_AfterBudgetCheckPasses below for
// how that branch is exercised deterministically today.
func TestOrgSend_BudgetTooSmallForEnterDelay_NothingTyped(t *testing.T) {
	o, h, _ := testOrg(t)
	o.SendEnterDelay = 200 * time.Millisecond
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	callsBefore := len(h.calls)
	eventsBefore := len(mustReadEvents(t, o))

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg, TimeoutMS: 5})
	if result.Err == nil {
		t.Fatal("expected a non-nil Err when the remaining budget cannot fund the pre-Enter pause")
	}
	if !strings.Contains(result.Err.Error(), "nothing was typed") {
		t.Errorf("expected the error to say nothing was typed, got %v", result.Err)
	}
	if result.Progress != SendProgressNothingSent {
		t.Errorf("expected Progress SendProgressNothingSent: the budget check must run before PaneSendText, got %v", result.Progress)
	}
	if result.PaneID != "" {
		t.Errorf("expected no PaneID on a return before PaneSendText was ever attempted, got %q", result.PaneID)
	}
	// Only the idle/done AgentWait call happened -- no PaneSendText, no
	// PaneSendKeys, no second (confirm) AgentWait.
	if got := h.calls[callsBefore:]; !slices.Equal(got, []string{"agent_wait"}) {
		t.Fatalf("expected only the idle/done AgentWait call, got %v", got)
	}
	if got := len(mustReadEvents(t, o)); got != eventsBefore {
		t.Fatalf("expected no new sent event, %d -> %d", eventsBefore, got)
	}
}

// TestOrgSend_CtxExpiresDuringEnterDelay_AfterBudgetCheckPasses is the
// self-review revalidation LOW-3 fix (docs/reports/self-review-2026-09-19-org-send-enter-timing.md,
// "Answer to the explicit question"): a deterministic reproduction of ctx
// expiring *inside* waitOrCtxDone, after the fail-closed budget check
// above has already passed -- the branch that check's own passing
// (`remaining > enterDelay`) would otherwise make unreachable through this
// package's fake driver alone, since fakeHerdr's calls used to be
// synchronous. fakeHerdr.paneSendTextDelay closes that gap: it simulates a
// real herdr round trip eating into the ctx budget between the budget
// check (which measures ctx time remaining right before PaneSendText) and
// the actual pre-Enter wait.
//
// TimeoutMS: 100, SendEnterDelay: 50ms, paneSendTextDelay: 80ms. The
// budget check passes (100ms > 50ms). PaneSendText then consumes 80ms of
// real time, leaving roughly 20ms of ctx budget when waitOrCtxDone's
// select races a 50ms timer against ctx.Done() -- a ~30ms margin in the
// direction that must win, run at -count=50 to confirm it does not flake
// (widen these three values, keeping their relative shape, if it ever
// does).
func TestOrgSend_CtxExpiresDuringEnterDelay_AfterBudgetCheckPasses(t *testing.T) {
	o, h, _ := testOrg(t)
	o.SendEnterDelay = 50 * time.Millisecond
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	h.paneSendTextDelay = 80 * time.Millisecond
	eventsBefore := len(mustReadEvents(t, o))

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg, TimeoutMS: 100})
	if result.Err == nil {
		t.Fatal("expected a non-nil Err when ctx expires inside waitOrCtxDone after the budget check passed")
	}
	if !strings.Contains(result.Err.Error(), "text typed but not submitted") {
		t.Errorf("expected the error to mention the text was typed but not submitted, got %v", result.Err)
	}
	if result.Progress != SendProgressTextTyped {
		t.Errorf("expected Progress SendProgressTextTyped: PaneSendText already succeeded and ctx expired before PaneSendKeys was ever called, got %v", result.Progress)
	}
	if result.PaneID == "" {
		t.Error("expected a non-empty PaneID on the error return so the operator knows which pane to check")
	}
	if len(h.sendKeysKeys) != 0 {
		t.Fatalf("expected zero PaneSendKeys calls, got %v", h.sendKeysKeys)
	}
	if got := len(mustReadEvents(t, o)); got != eventsBefore {
		t.Fatalf("expected no new sent event, %d -> %d", eventsBefore, got)
	}
}

// TestOrgSend_PaneSendKeysFails_ReportsEnterUnacknowledged is the AR-1
// cross-review fix (docs/reports/cross-review-triage-org-send-enter-timing.md):
// a PaneSendKeys failure no longer asserts "not submitted" -- herdr's CLI
// can be killed by ctx deadline expiry mid-call
// (driver.ExecRunner.Run uses exec.CommandContext), so an error here does
// NOT mean Enter failed to reach the pane; it means Send does not know
// either way. Progress must be SendProgressEnterUnacknowledged, and the
// error text must say the outcome is unknown rather than assert it never
// happened.
func TestOrgSend_PaneSendKeysFails_ReportsEnterUnacknowledged(t *testing.T) {
	o, h, _ := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	h.paneSendKeysErr = errors.New("stub: PaneSendKeys failed")
	eventsBefore := len(mustReadEvents(t, o))

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg})
	if result.Err == nil {
		t.Fatal("expected a non-nil Err when PaneSendKeys fails")
	}
	if strings.Contains(result.Err.Error(), "not submitted") {
		t.Errorf("expected the error to NOT assert the message was not submitted (the outcome is unknown), got %v", result.Err)
	}
	if !strings.Contains(result.Err.Error(), "may or may not have been submitted") {
		t.Errorf("expected the error to say the submit outcome is unknown, got %v", result.Err)
	}
	if result.Progress != SendProgressEnterUnacknowledged {
		t.Errorf("expected Progress SendProgressEnterUnacknowledged, got %v", result.Progress)
	}
	if result.PaneID == "" {
		t.Error("expected a non-empty PaneID on the error return so the operator knows which pane to check")
	}
	if got := len(mustReadEvents(t, o)); got != eventsBefore {
		t.Fatalf("expected no new sent event when PaneSendKeys fails, %d -> %d", eventsBefore, got)
	}
}

// TestOrgSend_PaneSendTextFails_ReportsTextUnacknowledged is the AR-1
// cross-review fix (docs/reports/cross-review-triage-org-send-enter-timing.md),
// the twin of TestOrgSend_PaneSendKeysFails_ReportsEnterUnacknowledged for
// the step before it: when herdr rejects the send-text call, Send still
// does not press Enter (a keystroke into a pane whose input box ralph
// does not know it filled is exactly the blind-Enter hazard this whole
// change rules out) and still appends no `sent` event -- but it does not
// claim nothing happened either. exec.CommandContext can kill herdr's CLI
// mid-call, so the paste may already have landed before the error came
// back: Progress reports SendProgressTextUnacknowledged and PaneID IS set,
// so the CLI can still point the operator at the pane to check.
func TestOrgSend_PaneSendTextFails_ReportsTextUnacknowledged(t *testing.T) {
	o, h, _ := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	h.paneSendTextErr = errors.New("stub: PaneSendText failed")
	eventsBefore := len(mustReadEvents(t, o))

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg})
	if result.Err == nil {
		t.Fatal("expected a non-nil Err when PaneSendText fails")
	}
	if !strings.Contains(result.Err.Error(), "send text to seat") {
		t.Errorf("expected the error to name the send-text step, got %v", result.Err)
	}
	if !strings.Contains(result.Err.Error(), "may or may not have reached the pane") {
		t.Errorf("expected the error to say the outcome is unknown, got %v", result.Err)
	}
	if result.Progress != SendProgressTextUnacknowledged {
		t.Errorf("expected Progress SendProgressTextUnacknowledged, got %v", result.Progress)
	}
	if result.PaneID == "" {
		t.Error("expected a non-empty PaneID on the error return (Send does not know whether the text reached the pane, so it still names the pane to check)")
	}
	if result.SubmitConfirmed {
		t.Error("expected SubmitConfirmed false: confirmSubmitted is never reached")
	}
	if len(h.sendKeysKeys) != 0 {
		t.Fatalf("expected no PaneSendKeys call after a failed PaneSendText, got %v", h.sendKeysKeys)
	}
	if got := len(mustReadEvents(t, o)); got != eventsBefore {
		t.Fatalf("expected no new sent event when PaneSendText fails, %d -> %d", eventsBefore, got)
	}
}

// TestOrgSend_AppendEventFailsAfterEnter_ReportsEnterPressedButUnrecorded is
// the self-review revalidation NEW-1 fix, renamed in cycle-2 (C2-3,
// docs/reports/self-review-2026-09-19-org-send-enter-timing.md) to say what
// Send actually observed: "Enter was pressed", not "submitted" -- Send
// cannot know the message was submitted, only that PaneSendKeys succeeded.
// When the manifest write for the `sent` event fails AFTER Enter has
// already succeeded (and confirmSubmitted has already run), Send must
// report Progress=SendProgressEnterPressed and wrap the error to say Enter
// was pressed -- this is what lets the CLI tell "Enter was pressed, only
// the history record was lost" apart from the two typed-but-unsubmitted/
// unacknowledged residues (see Send's and SendProgress's doc comments).
//
// Unix-only, no build tag needed: this whole package already is one
// (internal/org/lockfile.go uses syscall.Flock unconditionally). The
// manifest file is chmod'd read-only AFTER Spawn's own writes have already
// landed, so Send's idle/done wait, PaneSendText, PaneSendKeys and
// confirmSubmitted all still run normally against the fake driver, and
// only the final o.appendEvent call -- a real file write -- fails with a
// real permission error.
func TestOrgSend_AppendEventFailsAfterEnter_ReportsEnterPressedButUnrecorded(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores a read-only file's permission bit")
	}
	o, h, _ := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	manifestPath := o.Manifest.Path()
	if err := os.Chmod(manifestPath, 0o444); err != nil {
		t.Fatalf("chmod manifest read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(manifestPath, 0o644) })

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg})
	if result.Err == nil {
		t.Fatal("expected a non-nil Err when the sent event cannot be appended")
	}
	if !strings.Contains(result.Err.Error(), "Enter was pressed for seat") || !strings.Contains(result.Err.Error(), "could not be recorded") {
		t.Errorf("expected the error to say Enter was pressed but the sent event could not be recorded, got %v", result.Err)
	}
	if result.Progress != SendProgressEnterPressed {
		t.Errorf("expected Progress SendProgressEnterPressed: PaneSendKeys succeeded before appendEvent failed, got %v", result.Progress)
	}
	if result.PaneID == "" {
		t.Error("expected a non-empty PaneID on the error return")
	}
	if len(h.sendKeysKeys) != 1 || !slices.Equal(h.sendKeysKeys[0], []string{"Enter"}) {
		t.Fatalf("expected exactly one PaneSendKeys call with exactly [\"Enter\"] (no resend on this path either), got %v", h.sendKeysKeys)
	}
}

// TestOrgSend_ConfirmTimeout_CappedByRemainingCtxBudget covers the ms cap
// in capMSToContext: with a short TimeoutMS budget almost entirely
// consumed by the pre-Enter delay, the confirm AgentWait's timeoutMS must
// be capped at (roughly) whatever ctx budget remains, not at the full
// configured SendSubmitConfirmTimeout, and must never be less than 1.
//
// TimeoutMS is 400 (not e.g. 40) so this test's margin over SendEnterDelay
// (~380ms) is comparable to the neighbouring
// TestOrgSend_BudgetTooSmallForEnterDelay_NothingTyped's margin in the
// opposite direction, rather than the ~20ms this test used to leave itself
// (self-review LOW-6): a loaded CI runner or -race can add tens of
// milliseconds of scheduling jitter between the fail-closed budget check
// and PaneSendText/waitOrCtxDone actually running, and a 20ms margin was
// close enough to that jitter to risk turning this test itself into the
// fail-closed-budget case instead of the case it means to cover.
func TestOrgSend_ConfirmTimeout_CappedByRemainingCtxBudget(t *testing.T) {
	o, h, _ := testOrg(t)
	o.SendEnterDelay = 20 * time.Millisecond
	o.SendSubmitConfirmTimeout = 10 * time.Second // deliberately far larger than the ctx budget
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	msg := "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg, TimeoutMS: 400})
	if result.Err != nil {
		t.Fatalf("expected Send to succeed, got %v", result.Err)
	}

	if len(h.agentWaitTimeouts) != 2 {
		t.Fatalf("expected exactly 2 AgentWait calls, got %d (%v)", len(h.agentWaitTimeouts), h.agentWaitTimeouts)
	}
	confirmMS := h.agentWaitTimeouts[1]
	if confirmMS < 1 {
		t.Errorf("expected the confirm timeout to never go below 1ms, got %d", confirmMS)
	}
	if confirmMS >= int(o.SendSubmitConfirmTimeout/time.Millisecond) {
		t.Errorf("expected the confirm timeout (%dms) to be capped well below the configured SendSubmitConfirmTimeout (%v), it was not", confirmMS, o.SendSubmitConfirmTimeout)
	}
}

// TestCapMSToContext_FloorsAtOneMillisecond is a direct unit pin for
// capMSToContext's own two "at least 1" floors (self-review M1,
// docs/reports/self-review-2026-09-19-org-send-enter-timing.md): herdr
// (v0.7.5) treats a 0 --timeout as "wait indefinitely", so this helper must
// never return less than 1, whether it is want itself that rounds down to
// zero/negative milliseconds, or ctx's remaining budget that has already
// gone negative. TestOrgSend_ConfirmTimeout_CappedByRemainingCtxBudget only
// exercises the "cap to whatever remains" branch with a comfortably
// positive remaining (~380ms); it never drives either floor down to where
// it actually clamps, so both are pinned here directly against the pure
// function instead of through Send's timing-sensitive plumbing.
func TestCapMSToContext_FloorsAtOneMillisecond(t *testing.T) {
	t.Run("want itself rounds below 1ms, no deadline", func(t *testing.T) {
		if got := capMSToContext(context.Background(), 0); got != 1 {
			t.Errorf("capMSToContext(no deadline, want=0) = %d, want 1", got)
		}
	})

	t.Run("want is negative, no deadline", func(t *testing.T) {
		if got := capMSToContext(context.Background(), -5*time.Second); got != 1 {
			t.Errorf("capMSToContext(no deadline, want=-5s) = %d, want 1", got)
		}
	})

	t.Run("ctx deadline already passed, want comfortably positive", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
		defer cancel()
		if got := capMSToContext(ctx, 10*time.Second); got != 1 {
			t.Errorf("capMSToContext(expired deadline, want=10s) = %d, want 1", got)
		}
	})

	t.Run("ctx deadline already passed, want itself also rounds below 1ms", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
		defer cancel()
		if got := capMSToContext(ctx, 0); got != 1 {
			t.Errorf("capMSToContext(expired deadline, want=0) = %d, want 1", got)
		}
	})
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

// TestOrgStop_LegacySpawnedEvent_NoSpawnStarted_NoModelObservedToken is a
// /test cycle-1 addition (issue #173 handoff item 10: "Stop for a seat with
// NO spawn_started at all in the manifest -- legacy or hand-built events").
// TestOrgStop_ExistingSeat_NoPaneOrAgmsgTeam_RecordsSkippedNotes above
// already exercises codexSpawnCorrelation's ok=false path (its seat's only
// manifest event is a legacy EventSpawned, never EventSpawnStarted) but
// never asserts the specific claim this item names: no model_observed=
// TOKEN AT ALL on the stopped event -- not even "none" -- since
// isCodexSpawn is false for a seat with nothing to correlate
// (observeStopModelReceipt's own doc comment), distinct from a codex seat
// whose correlated spawn just could not be observed (which DOES get
// "model_observed=none", per TestOrgStop_Codex_NoPromptFileSeat_NothingAppended
// above). This test names and pins that distinction directly, plus
// StopResult.ModelReceipt staying the zero Receipt and Stop still
// succeeding.
func TestOrgStop_LegacySpawnedEvent_NoSpawnStarted_NoModelObservedToken(t *testing.T) {
	o, _, _ := testOrg(t)
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: "org-a", SeatID: "seat-legacy", Event: EventSpawned,
	}); err != nil {
		t.Fatalf("seed manifest event: %v", err)
	}

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-legacy"})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt != (Receipt{}) {
		t.Fatalf("expected no receipt for a seat with no correlated spawn_started, got %+v", result.ModelReceipt)
	}

	events := mustReadEvents(t, o)
	last := events[len(events)-1]
	if strings.Contains(last.Details, "model_observed=") {
		t.Fatalf("expected no model_observed= token at all (not even \"none\") for a seat with no spawn_started to correlate, got Details=%q", last.Details)
	}
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

// --- codex model observation at stop (plan Scope row 4, AC-5/AC-6) ---

// appendInterruptedCodexSpawnTrail directly seeds the manifest trail a real
// Spawn call for a codex seat with a role-prompt file produces up through
// `spawned` -- WITHOUT ever appending a receipt -- simulating a process
// killed partway through observeCodexSpawnReceipt's poll (spawn_started ->
// spawn_step(agent_started prompt_file=...) -> spawned all durably
// written; the receipt append that follows in Spawn never ran). Used by
// the "no receipt at all for this spawn attempt" test below, which needs
// that exact on-disk shape without going through the full Spawn saga
// (which always eventually appends some receipt itself).
func appendInterruptedCodexSpawnTrail(t *testing.T, o *Org, orgID, seatID, promptPath string, spawnStartedAt time.Time) {
	t.Helper()
	base := ManifestEvent{OrgID: orgID, SeatID: seatID, Role: "implementer", Driver: "codex", Model: "gpt-5-codex"}

	started := base
	started.TS = spawnStartedAt.UTC().Format(time.RFC3339)
	started.Event = EventSpawnStarted
	if err := o.appendEvent(started); err != nil {
		t.Fatalf("seed spawn_started: %v", err)
	}

	step := base
	step.TS = spawnStartedAt.Add(time.Second).UTC().Format(time.RFC3339)
	step.Event = EventSpawnStep
	step.Details = codexPromptFileDetailsPrefix + promptPath
	if err := o.appendEvent(step); err != nil {
		t.Fatalf("seed spawn_step: %v", err)
	}

	spawned := base
	spawned.TS = spawnStartedAt.Add(2 * time.Second).UTC().Format(time.RFC3339)
	spawned.Event = EventSpawned
	if err := o.appendEvent(spawned); err != nil {
		t.Fatalf("seed spawned: %v", err)
	}
}

// appendInterruptedAgentStartedTrail seeds only spawn_started and
// spawn_step(agent_started prompt_file=...) -- WITHOUT ever reaching
// `spawned` -- simulating a process killed between AgentStart succeeding
// and the spawned event being written. Roster derives this seat as Active
// (spawn_started/spawn_step both count, seat.go's activeEvents) but NOT
// idempotent-eligible (Spawn's own idempotent early return only fires on
// EventSpawned): a second Spawn call for the same org/seat still reaches
// ValidateSpawnEnvelope, which runs BEFORE Spawn's own stale-in-flight
// compensation branch -- so an invalid retry gets rejected via reject()
// while these two events survive untouched on the manifest. That lets a
// test set up exactly the scenario codexSpawnCorrelation must handle
// correctly (issue #173): it still finds this ORIGINAL spawn_started,
// while the roster's latest state becomes the REJECTED retry's own
// Driver/Model.
// base carries whichever Role/Driver/Model the caller needs (AC-1 uses
// codex/codex; AC-2's reverse case uses a claude-launched seat).
func appendInterruptedAgentStartedTrail(t *testing.T, o *Org, base ManifestEvent, promptPath string, spawnStartedAt time.Time) {
	t.Helper()
	started := base
	started.TS = spawnStartedAt.UTC().Format(time.RFC3339)
	started.Event = EventSpawnStarted
	if err := o.appendEvent(started); err != nil {
		t.Fatalf("seed spawn_started: %v", err)
	}

	step := base
	step.TS = spawnStartedAt.Add(time.Second).UTC().Format(time.RFC3339)
	step.Event = EventSpawnStep
	step.Details = codexPromptFileDetailsPrefix + promptPath
	if err := o.appendEvent(step); err != nil {
		t.Fatalf("seed spawn_step: %v", err)
	}
}

// TestOrgStop_Codex_AppendsWhenSpawnReceiptWasUnknown is AC-5's main path:
// Spawn's own poll found nothing (testOrg's tiny timeout, no fixture yet),
// so its receipt is honored=unknown with no reported model -- that does
// not suppress Stop's own, single, non-waiting observation. Once a
// qualifying record exists by the time Stop runs, Stop appends a second
// receipt and records model_observed=true on the stopped event.
func TestOrgStop_Codex_AppendsWhenSpawnReceiptWasUnknown(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	p := mustCodexSpawnParams("org-a", "seat-1")

	spawnResult := o.Spawn(p)
	if spawnResult.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", spawnResult)
	}
	if spawnResult.ModelReceipt.Honored != HonoredUnknown {
		t.Fatalf("expected spawn's own receipt to be unknown (no fixture yet), got %+v", spawnResult.ModelReceipt)
	}

	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, time.Now().Add(time.Second), "gpt-5-codex")

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt.Honored != HonoredTrue || result.ModelReceipt.ReportedEffectiveModel != "gpt-5-codex" {
		t.Fatalf("expected an honored=true receipt appended at stop, got %+v", result.ModelReceipt)
	}
	if !strings.Contains(result.ModelReceipt.Reason, "(observed at stop)") {
		t.Fatalf("expected the stop-time Reason suffix, got %q", result.ModelReceipt.Reason)
	}

	rr, err := o.Receipts.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	if len(rr.Receipts) != 2 {
		t.Fatalf("expected 2 receipts (spawn's unknown + stop's observed), got %+v", rr.Receipts)
	}

	mrr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := mrr.Events[len(mrr.Events)-1]
	assertDetailsContains(t, last.Details, "model_observed=true")
}

// TestOrgStop_Codex_HonoredFalse_ObservedAtStop mirrors the above for the
// mismatch case, through Stop instead of Spawn's own poll.
func TestOrgStop_Codex_HonoredFalse_ObservedAtStop(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	p := mustCodexSpawnParams("org-a", "seat-1")

	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, time.Now().Add(time.Second), "gpt-5.6-sol")

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt.Honored != HonoredFalse || result.ModelReceipt.ReportedEffectiveModel != "gpt-5.6-sol" {
		t.Fatalf("expected an honored=false receipt appended at stop, got %+v", result.ModelReceipt)
	}
	if !strings.Contains(result.ModelReceipt.Reason, "gpt-5.6-sol") || !strings.Contains(result.ModelReceipt.Reason, "gpt-5-codex") {
		t.Fatalf("expected Reason to name both models, got %q", result.ModelReceipt.Reason)
	}

	mrr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := mrr.Events[len(mrr.Events)-1]
	assertDetailsContains(t, last.Details, "model_observed=false")
}

// TestOrgStop_Codex_DoesNotAppendWhenAlreadyObserved: Spawn's own poll
// already found and recorded the model, so Stop must not observe again.
//
// A deterministic, strictly-increasing fake clock (the same pattern
// TestOrgStop_Codex_DryRunRespawnDoesNotDisplaceRealSpawnCorrelation
// uses) guarantees Spawn's own found receipt lands strictly after its own
// spawn_started -- matching hasObservedCodexReceiptAfter's own strict
// "after, not at-or-after" rule -- rather than depending on whether a
// fast test run happens to keep both timestamps in the same wall-clock
// second, which a real `o.now()` clock cannot guarantee either way.
//
// Only ONE qualifying fixture exists on disk (deliberately not a second,
// contradicting one, unlike an earlier version of this test): if Stop
// wrongly re-observed instead of skipping, it would still find this same
// single fixture and append a SECOND receipt to the store, so "exactly
// one receipt persisted" is what actually proves Stop never called the
// observer a second time. Two qualifying fixtures would make "ambiguous,
// nothing appended" indistinguishable from "correctly skipped" -- both
// produce the same zero ModelReceipt and the same receipts-file count.
func TestOrgStop_Codex_DoesNotAppendWhenAlreadyObserved(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	p := mustCodexSpawnParams("org-a", "seat-1")

	base := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	var tick int64
	o.Now = func() time.Time {
		tick++
		return base.Add(time.Duration(tick) * time.Second)
	}

	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	// Safely after any plausible spawn_started tick (a saga is at most a
	// few dozen o.now() calls), so the age gate passes regardless of the
	// exact count, without needing to read spawn_started back first.
	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, base.Add(1000*time.Second), "gpt-5-codex")

	spawnResult := o.Spawn(p)
	if spawnResult.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", spawnResult)
	}
	if spawnResult.ModelReceipt.Honored != HonoredTrue {
		t.Fatalf("expected spawn's own poll to have already observed the model, got %+v", spawnResult.ModelReceipt)
	}

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt != (Receipt{}) {
		t.Fatalf("expected no receipt appended at stop, got %+v", result.ModelReceipt)
	}

	rr, err := o.Receipts.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	if len(rr.Receipts) != 1 {
		t.Fatalf("expected only spawn's own receipt (a second receipt here would mean Stop re-observed), got %+v", rr.Receipts)
	}

	mrr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := mrr.Events[len(mrr.Events)-1]
	assertDetailsContains(t, last.Details, "model_observed=none")
}

// TestOrgStop_Codex_AppendsWhenNoReceiptAtAllForThisSpawn is AC-5's
// explicit edge case: a spawn attempt that reached `spawned` but never got
// as far as appending any receipt at all (process killed mid-poll) must
// still be observed at stop -- there being nothing to compare against is
// not the same as "already observed".
func TestOrgStop_Codex_AppendsWhenNoReceiptAtAllForThisSpawn(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	promptPath, err := o.promptFilePath("org-a", "seat-1")
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	spawnStartedAt := time.Now()
	appendInterruptedCodexSpawnTrail(t, o, "org-a", "seat-1", promptPath, spawnStartedAt)

	rr, err := o.Receipts.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	if len(rr.Receipts) != 0 {
		t.Fatalf("test setup invariant broken: expected zero receipts before stop, got %+v", rr.Receipts)
	}

	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, spawnStartedAt.Add(time.Second), "gpt-5-codex")

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt.Honored != HonoredTrue {
		t.Fatalf("expected an honored=true receipt appended at stop, got %+v", result.ModelReceipt)
	}
}

// TestOrgStop_Codex_AppendsWhenOnlyRejectionReceiptExistsSince covers the
// Codex-advisory fix (plan Design decisions): a rejection or dry-run
// receipt for this seat, sitting after this spawn attempt's
// spawn_started, never carries a reported model -- it must not be
// mistaken for "already observed".
func TestOrgStop_Codex_AppendsWhenOnlyRejectionReceiptExistsSince(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	promptPath, err := o.promptFilePath("org-a", "seat-1")
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	spawnStartedAt := time.Now()
	appendInterruptedCodexSpawnTrail(t, o, "org-a", "seat-1", promptPath, spawnStartedAt)

	if err := o.Receipts.Append(Receipt{
		TS:    spawnStartedAt.Add(3 * time.Second).UTC().Format(time.RFC3339),
		OrgID: "org-a", SeatID: "seat-1", Role: "implementer", Driver: "codex",
		CommandedModel: "gpt-5-codex", Honored: HonoredFalse, Reason: "unrelated rejection receipt",
	}); err != nil {
		t.Fatalf("seed rejection receipt: %v", err)
	}

	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, spawnStartedAt.Add(4*time.Second), "gpt-5-codex")

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt.Honored != HonoredTrue {
		t.Fatalf("expected the rejection receipt (no reported model) to not suppress observation, got %+v", result.ModelReceipt)
	}
}

// TestOrgStop_Codex_DryRunRespawnDoesNotDisplaceRealSpawnCorrelation: a
// `ralph org spawn --dry-run` issued for an already-spawned codex seat
// appends its own dry-run spawn_started/
// spawn_step trail (dryRunSpawn, spawn.go) for the SAME org/seat -- with
// DryRun: true. codexSpawnCorrelation must skip those dry-run events in
// both of its loops, so this later dry-run trail never becomes "the
// latest spawn_started" and displaces the real one; Stop must still
// correlate against the real spawn and observe successfully.
func TestOrgStop_Codex_DryRunRespawnDoesNotDisplaceRealSpawnCorrelation(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	p := mustCodexSpawnParams("org-a", "seat-1")

	// A deterministic, strictly-increasing fake clock (1 tick per o.now()/
	// o.nowTime() call) replaces the real wall clock for this test: it
	// needs the real spawn's own spawn_started to be provably,
	// unambiguously earlier than the dry-run respawn's own (later)
	// spawn_started -- RFC3339's whole-second TS precision means a short
	// real sleep could land both in the same second and fail to
	// distinguish the bug from the fix; a monotonic fake clock gets the
	// same guarantee instantly and deterministically.
	base := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	var tick int64
	o.Now = func() time.Time {
		tick++
		return base.Add(time.Duration(tick) * time.Second)
	}

	spawnResult := o.Spawn(p)
	if spawnResult.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("real spawn failed: %+v", spawnResult)
	}
	if spawnResult.ModelReceipt.Honored != HonoredUnknown {
		t.Fatalf("expected the real spawn's own receipt to be unknown (no fixture yet), got %+v", spawnResult.ModelReceipt)
	}

	mrr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var realSpawnStartedAt time.Time
	for _, ev := range mrr.Events {
		if ev.OrgID == "org-a" && ev.SeatID == "seat-1" && ev.Event == EventSpawnStarted {
			realSpawnStartedAt, err = time.Parse(time.RFC3339, ev.TS)
			if err != nil {
				t.Fatalf("parse spawn_started TS: %v", err)
			}
		}
	}
	if realSpawnStartedAt.IsZero() {
		t.Fatalf("test setup invariant broken: no spawn_started event found for the real spawn")
	}

	// This record's session_meta is exactly the real spawn's own
	// spawn_started instant -- guaranteed, by the strictly-increasing fake
	// clock, to be chronologically before the dry-run respawn's own
	// spawn_started below (appended by a later call to the same clock).
	// It is exactly the record that must be picked when correlation
	// (correctly) uses the REAL spawn_started as its age cutoff, and
	// exactly the record that would be wrongly excluded by age if
	// correlation instead used the dry-run's (later) spawn_started as
	// cutoff -- the displacement bug this test guards against.
	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, realSpawnStartedAt, "gpt-5-codex")

	dryRunResult := o.Spawn(SpawnParams{
		OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver, Model: p.Model,
		Cwd: p.Cwd, Scope: p.Scope, DryRun: true,
	})
	if dryRunResult.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("dry-run respawn of the same seat failed: %+v", dryRunResult)
	}

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt.Honored != HonoredTrue {
		t.Fatalf("expected the observed receipt for the real spawn's own (earlier) session, got %+v", result.ModelReceipt)
	}

	mrr2, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := mrr2.Events[len(mrr2.Events)-1]
	assertDetailsContains(t, last.Details, "model_observed=true")
}

// --- issue #173: Stop compares against the spawn that actually launched
// the seat, never the roster ---

// TestOrgStop_Codex_RecoversCommandedModelAfterInterruptedRetryRejected is
// AC-1: a codex spawn interrupted right after agent_started (no `spawned`
// yet -- a stale in-flight saga), followed by a retry for the SAME
// org/seat with a DIFFERENT, out-of-pool model, which Spawn rejects via
// ValidateSpawnEnvelope BEFORE it ever reaches the stale-in-flight
// compensation branch (Spawn's own doc comment: step 2 runs before step
// 4). The `rejected` event this appends carries the RETRY's own
// Model/Driver, becoming the roster's latest SeatStatus for this seat --
// but codexSpawnCorrelation still finds the ORIGINAL spawn_started
// (rejected events are never scanned by it), so Stop's receipt must
// compare the session record against the ORIGINAL commanded model, never
// the rejected one. Before the fix, this test fails: the old code
// compared against seat.Model (the roster's "gpt-9-nonexistent"), giving
// a false honored=false.
func TestOrgStop_Codex_RecoversCommandedModelAfterInterruptedRetryRejected(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	const orgID, seatID = "org-a", "seat-1"

	promptPath, err := o.promptFilePath(orgID, seatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	spawnStartedAt := time.Now()
	appendInterruptedAgentStartedTrail(t, o, ManifestEvent{
		OrgID: orgID, SeatID: seatID, Role: "implementer", Driver: "codex", Model: "gpt-5-codex",
	}, promptPath, spawnStartedAt)

	retryResult := o.Spawn(SpawnParams{
		OrgID: orgID, SeatID: seatID, Role: "implementer", Driver: "codex", Model: "gpt-9-nonexistent",
		Cwd: "/tmp/seat", TimeoutMS: 5000, Scope: "test-scope",
	})
	if retryResult.Outcome != SpawnOutcomeRejected {
		t.Fatalf("expected the retry to be rejected (out-of-pool model), got %+v", retryResult)
	}

	// Test setup invariant: after the rejected retry, the roster reflects
	// that retry's own model, not the original spawn's -- exactly the
	// stale data Stop must not compare against.
	seat, ok, err := o.findSeat(orgID, seatID)
	if err != nil || !ok {
		t.Fatalf("findSeat: ok=%v err=%v", ok, err)
	}
	if seat.Model != "gpt-9-nonexistent" {
		t.Fatalf("test setup invariant broken: expected the roster to reflect the rejected retry's model, got %q", seat.Model)
	}

	// The session record for the ORIGINAL spawn reports the ORIGINAL
	// commanded model.
	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, spawnStartedAt.Add(time.Second), "gpt-5-codex")

	result := o.Stop(StopParams{OrgID: orgID, Seat: seatID})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt.CommandedModel != "gpt-5-codex" {
		t.Fatalf("expected the receipt's commanded model to be the ORIGINAL spawn's model, not the rejected retry's, got %+v", result.ModelReceipt)
	}
	if result.ModelReceipt.Honored != HonoredTrue || result.ModelReceipt.ReportedEffectiveModel != "gpt-5-codex" {
		t.Fatalf("expected honored=true (before the fix: false, wrongly compared against the rejected retry's model), got %+v", result.ModelReceipt)
	}

	mrr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := mrr.Events[len(mrr.Events)-1]
	assertDetailsContains(t, last.Details, "model_observed=true")
}

// TestOrgStop_Codex_ObservesWhenRejectedRetryUsedADifferentDriver is AC-2's
// forward case: the seat's ORIGINAL (launched) spawn was codex, but a
// later retry for a DIFFERENT DRIVER (claude) gets rejected -- becoming
// the roster's latest SeatStatus with Driver="claude". Stop must still
// attempt observation: "is this a codex seat" is decided by the
// CORRELATED spawn's own Driver, never the roster's.
func TestOrgStop_Codex_ObservesWhenRejectedRetryUsedADifferentDriver(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	const orgID, seatID = "org-a", "seat-1"

	promptPath, err := o.promptFilePath(orgID, seatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	spawnStartedAt := time.Now()
	appendInterruptedAgentStartedTrail(t, o, ManifestEvent{
		OrgID: orgID, SeatID: seatID, Role: "implementer", Driver: "codex", Model: "gpt-5-codex",
	}, promptPath, spawnStartedAt)

	retryResult := o.Spawn(SpawnParams{
		OrgID: orgID, SeatID: seatID, Role: "implementer", Driver: "claude", Model: "not-a-real-model",
		Cwd: "/tmp/seat", TimeoutMS: 5000, Scope: "test-scope",
	})
	if retryResult.Outcome != SpawnOutcomeRejected {
		t.Fatalf("expected the retry to be rejected (out-of-pool model), got %+v", retryResult)
	}

	seat, ok, err := o.findSeat(orgID, seatID)
	if err != nil || !ok {
		t.Fatalf("findSeat: ok=%v err=%v", ok, err)
	}
	if seat.Driver != "claude" {
		t.Fatalf("test setup invariant broken: expected the roster to reflect the rejected retry's driver, got %q", seat.Driver)
	}

	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, spawnStartedAt.Add(time.Second), "gpt-5-codex")

	result := o.Stop(StopParams{OrgID: orgID, Seat: seatID})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt.Honored != HonoredTrue || result.ModelReceipt.ReportedEffectiveModel != "gpt-5-codex" {
		t.Fatalf("expected Stop to observe despite the roster showing a claude driver, got %+v", result.ModelReceipt)
	}

	mrr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := mrr.Events[len(mrr.Events)-1]
	assertDetailsContains(t, last.Details, "model_observed=true")
}

// TestOrgStop_Codex_NoObservationWhenLaunchedSpawnWasClaude_DespiteRejectedCodexRetry
// is AC-2's reverse case: the seat's ORIGINAL (launched) spawn was claude,
// and a later codex retry gets rejected -- the roster's latest SeatStatus
// shows Driver="codex", but Stop must not observe at all (not even
// model_observed=none): the CORRELATED spawn decides "is this a codex
// seat", and it never was.
func TestOrgStop_Codex_NoObservationWhenLaunchedSpawnWasClaude_DespiteRejectedCodexRetry(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	const orgID, seatID = "org-a", "seat-1"

	spawnStartedAt := time.Now()
	appendInterruptedAgentStartedTrail(t, o, ManifestEvent{
		OrgID: orgID, SeatID: seatID, Role: "implementer", Driver: "claude", Model: "sonnet",
	}, "", spawnStartedAt)

	retryResult := o.Spawn(SpawnParams{
		OrgID: orgID, SeatID: seatID, Role: "implementer", Driver: "codex", Model: "gpt-9-nonexistent",
		Cwd: "/tmp/seat", TimeoutMS: 5000, Scope: "test-scope",
	})
	if retryResult.Outcome != SpawnOutcomeRejected {
		t.Fatalf("expected the retry to be rejected (out-of-pool model), got %+v", retryResult)
	}

	seat, ok, err := o.findSeat(orgID, seatID)
	if err != nil || !ok {
		t.Fatalf("findSeat: ok=%v err=%v", ok, err)
	}
	if seat.Driver != "codex" {
		t.Fatalf("test setup invariant broken: expected the roster to reflect the rejected retry's driver, got %q", seat.Driver)
	}

	result := o.Stop(StopParams{OrgID: orgID, Seat: seatID})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt != (Receipt{}) {
		t.Fatalf("expected no receipt appended, got %+v", result.ModelReceipt)
	}

	mrr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := mrr.Events[len(mrr.Events)-1]
	if strings.Contains(last.Details, "model_observed=") {
		t.Fatalf("expected no model_observed token at all (the launched spawn was claude, not codex), got %q", last.Details)
	}
}

// TestOrgStop_Codex_EmptyModelOnCorrelatedSpawn_NothingObserved covers the
// plan's own edge case: a spawn_started event with no Model at all (an old
// manifest recorded before that field existed -- a real spawn_started
// always carries it via checkCapacityAndStart). codexSpawnCorrelation
// still returns ok=true with an empty Model, but observeStopModelReceipt
// must treat that as "nothing to compare against": isCodexSpawn stays
// true (so the stopped event still gets a model_observed= token), but
// observed stays false, same as an empty PromptPath.
func TestOrgStop_Codex_EmptyModelOnCorrelatedSpawn_NothingObserved(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	const orgID, seatID = "org-a", "seat-1"

	promptPath, err := o.promptFilePath(orgID, seatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	spawnStartedAt := time.Now()
	appendInterruptedAgentStartedTrail(t, o, ManifestEvent{
		OrgID: orgID, SeatID: seatID, Role: "implementer", Driver: "codex", Model: "",
	}, promptPath, spawnStartedAt)

	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, spawnStartedAt.Add(time.Second), "gpt-5-codex")

	result := o.Stop(StopParams{OrgID: orgID, Seat: seatID})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt != (Receipt{}) {
		t.Fatalf("expected no receipt appended when the correlated spawn has no commanded model, got %+v", result.ModelReceipt)
	}

	mrr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := mrr.Events[len(mrr.Events)-1]
	assertDetailsContains(t, last.Details, "model_observed=none")
}

// TestHasObservedCodexReceiptAfter_OnlyThisSpawnsObservedReceiptCounts pins
// which receipts stop Stop from observing again: only one for the same org
// and seat, written STRICTLY AFTER this spawn started, that carries a
// reported model. An observed receipt left by an EARLIER spawn of the same
// seat must not count (the respawned seat may run a different model), and
// neither do receipts without a reported model (unknown, rejection,
// dry-run) or another seat's receipts. The same-second case is its own
// case: a same-second receipt cannot be proven to belong to THIS spawn
// rather than a previous one that happened to stop and respawn within the
// same real second, so it must not count either.
func TestHasObservedCodexReceiptAfter_OnlyThisSpawnsObservedReceiptCounts(t *testing.T) {
	const spawnTS = "2026-09-20T01:00:00Z"
	observed := func(org, seat, ts string) Receipt {
		return Receipt{TS: ts, OrgID: org, SeatID: seat, Honored: HonoredTrue, ReportedEffectiveModel: "gpt-5.5"}
	}
	cases := []struct {
		name     string
		receipts []Receipt
		want     bool
	}{
		{"no receipts at all", nil, false},
		{"observed receipt from an earlier spawn of the same seat", []Receipt{observed("org-a", "seat-1", "2026-09-20T00:59:59Z")}, false},
		{"same-second receipt may belong to the previous spawn, not this one", []Receipt{observed("org-a", "seat-1", spawnTS)}, false},
		{"observed receipt after the spawn", []Receipt{observed("org-a", "seat-1", "2026-09-20T01:00:05Z")}, true},
		{"unknown receipt after the spawn", []Receipt{{TS: "2026-09-20T01:00:05Z", OrgID: "org-a", SeatID: "seat-1", Honored: HonoredUnknown}}, false},
		{"rejection receipt after the spawn", []Receipt{{TS: "2026-09-20T01:00:05Z", OrgID: "org-a", SeatID: "seat-1", Honored: HonoredFalse, Reason: "rejected"}}, false},
		{"another seat's observed receipt", []Receipt{observed("org-a", "seat-2", "2026-09-20T01:00:05Z")}, false},
		{"another org's observed receipt", []Receipt{observed("org-b", "seat-1", "2026-09-20T01:00:05Z")}, false},
		{"earlier observed plus later unknown", []Receipt{
			observed("org-a", "seat-1", "2026-09-19T23:00:00Z"),
			{TS: "2026-09-20T01:00:05Z", OrgID: "org-a", SeatID: "seat-1", Honored: HonoredUnknown},
		}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := hasObservedCodexReceiptAfter(c.receipts, "org-a", "seat-1", spawnTS); got != c.want {
				t.Errorf("hasObservedCodexReceiptAfter = %v, want %v", got, c.want)
			}
		})
	}
}

// TestCodexSpawnCorrelation_TwoRealSpawns_LatestWins is a /test cycle-1
// addition (plan's Test plan edge cases: "同じ org・seat id を stop 後に再
// spawn した場合"). TestOrgStop_Codex_DryRunRespawnDoesNotDisplaceRealSpawnCorrelation
// above only exercises one real spawn_started plus a later dry-run one; this
// covers the genuinely untested case -- TWO REAL (non-dry-run) spawn_started
// events for the same org_id/seat_id, as a real Stop-then-respawn produces.
// codexSpawnCorrelation's forward scan keeps overwriting startedIdx on every
// matching spawn_started, so it must land on the later attempt's own index,
// and its second loop (which only starts scanning from startedIdx) must
// therefore also return that later attempt's own promptPath, never the
// first attempt's -- both asserted directly here since this is the smallest
// setup that reaches the exact behavior (no herdr/agmsg round trip needed).
// The struct's other fields (Model/Driver/Role) are asserted the same
// way: the second spawn_started's own values win, and a trailing
// `rejected` event for a THIRD, different model/driver -- the roster's
// own latest state, were codexSpawnCorrelation ever to read it -- must
// have no effect at all, since this function only ever reads
// spawn_started/spawn_step.
func TestCodexSpawnCorrelation_TwoRealSpawns_LatestWins(t *testing.T) {
	events := []ManifestEvent{
		{TS: "2026-09-18T07:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawnStarted, Model: "gpt-5-codex", Driver: "codex", Role: "implementer"},
		{TS: "2026-09-18T07:00:01Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawnStep, Details: codexPromptFileDetailsPrefix + "/state/prompts/org-a_seat-1-first.md"},
		{TS: "2026-09-18T07:00:05Z", OrgID: "org-a", SeatID: "seat-1", Event: "stopped"},
		{TS: "2026-09-18T08:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawnStarted, Model: "gpt-5-codex", Driver: "codex", Role: "worker"},
		{TS: "2026-09-18T08:00:01Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawnStep, Details: codexPromptFileDetailsPrefix + "/state/prompts/org-a_seat-1-second.md"},
		// A later dry-run spawn_started/spawn_step for a third model/driver
		// must not displace the second (real) spawn_started either -- the
		// dry-run filter in both of codexSpawnCorrelation's loops.
		{TS: "2026-09-18T08:30:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawnStarted, Model: "gpt-9-dryrun", Driver: "claude", DryRun: true},
		// A trailing rejected retry (a different model/driver) is a state
		// event that would become the roster's own latest SeatStatus, but
		// codexSpawnCorrelation never reads EventRejected at all -- it
		// must have zero effect here.
		{TS: "2026-09-18T09:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventRejected, Model: "gpt-9-nonexistent", Driver: "claude", Role: "worker"},
	}

	corr, ok := codexSpawnCorrelation(events, "org-a", "seat-1")
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if corr.StartedTS != "2026-09-18T08:00:00Z" {
		t.Fatalf("expected the LATER real spawn_started's TS, got %q", corr.StartedTS)
	}
	wantAt, err := time.Parse(time.RFC3339, "2026-09-18T08:00:00Z")
	if err != nil {
		t.Fatalf("parse want TS: %v", err)
	}
	if !corr.StartedAt.Equal(wantAt) {
		t.Fatalf("expected StartedAt %v, got %v", wantAt, corr.StartedAt)
	}
	if corr.PromptPath != "/state/prompts/org-a_seat-1-second.md" {
		t.Fatalf("expected the second spawn's own promptPath, not the first attempt's, got %q", corr.PromptPath)
	}
	if corr.Model != "gpt-5-codex" || corr.Driver != "codex" || corr.Role != "worker" {
		t.Fatalf("expected the second (real, non-dry-run) spawn_started's own Model/Driver/Role, unaffected by the later dry-run or rejected events, got %+v", corr)
	}
}

// TestPromptPathFromAgentStartedDetails is promptPathFromAgentStartedDetails's
// own pure-function table test: it must strip a trailing
// " agent_start_retries=<digits>" suffix anchored at the very end of
// Details, never a path that merely contains that text (or spaces)
// somewhere in the middle.
func TestPromptPathFromAgentStartedDetails(t *testing.T) {
	cases := []struct {
		name     string
		details  string
		wantPath string
		wantOK   bool
	}{
		{
			name:     "no retries suffix",
			details:  codexPromptFileDetailsPrefix + "/state/prompts/org-a_seat-1.md",
			wantPath: "/state/prompts/org-a_seat-1.md",
			wantOK:   true,
		},
		{
			name:     "single-digit retries suffix",
			details:  codexPromptFileDetailsPrefix + "/state/prompts/org-a_seat-1.md agent_start_retries=1",
			wantPath: "/state/prompts/org-a_seat-1.md",
			wantOK:   true,
		},
		{
			name:     "two-digit retries suffix",
			details:  codexPromptFileDetailsPrefix + "/state/prompts/org-a_seat-1.md agent_start_retries=12",
			wantPath: "/state/prompts/org-a_seat-1.md",
			wantOK:   true,
		},
		{
			// /test cycle-2 addition: isAllDigits/CutPrefix never parse the
			// digits into a number (agentStartWithRetry's real retry count
			// is bounded by maxAgentStartAttempts, far smaller than this),
			// so an implausibly large digit string must still strip
			// cleanly -- no int overflow or truncation risk, since nothing
			// here ever converts it.
			name:     "retries suffix with an implausibly large digit string",
			details:  codexPromptFileDetailsPrefix + "/state/prompts/org-a_seat-1.md agent_start_retries=99999999999999999999999999999999",
			wantPath: "/state/prompts/org-a_seat-1.md",
			wantOK:   true,
		},
		{
			name:     "path with spaces plus a retries suffix",
			details:  codexPromptFileDetailsPrefix + "/state dir/with spaces/prompts/org-a_seat-1.md agent_start_retries=3",
			wantPath: "/state dir/with spaces/prompts/org-a_seat-1.md",
			wantOK:   true,
		},
		{
			name:     "path contains the suffix key in the middle, plus a real trailing suffix",
			details:  codexPromptFileDetailsPrefix + "/state/agent_start_retries=3 more/org-a_seat-1.md agent_start_retries=5",
			wantPath: "/state/agent_start_retries=3 more/org-a_seat-1.md",
			wantOK:   true,
		},
		{
			name:     "path contains the suffix key in the middle, no real trailing suffix",
			details:  codexPromptFileDetailsPrefix + "/state/agent_start_retries=3 more/org-a_seat-1.md",
			wantPath: "/state/agent_start_retries=3 more/org-a_seat-1.md",
			wantOK:   true,
		},
		{
			name:    "Details without the prefix at all",
			details: "agent_started",
			wantOK:  false,
		},
		{
			name:     "prefix with an empty path",
			details:  codexPromptFileDetailsPrefix,
			wantPath: "",
			wantOK:   true,
		},
		{
			// The retries suffix is only stripped when it is the LAST
			// field -- a field appended after it (the shape a future edit
			// to agentStartedDetails could introduce) is not stripped and
			// becomes part of the "path" instead, pinning the contract
			// the constant's own doc comment now states.
			name:     "retries suffix is not the last field, so it is not stripped",
			details:  codexPromptFileDetailsPrefix + "/state/prompts/org-a_seat-1.md agent_start_retries=2 some_later_field=x",
			wantPath: "/state/prompts/org-a_seat-1.md agent_start_retries=2 some_later_field=x",
			wantOK:   true,
		},
		{
			// /test cycle-2 addition: unlike the two "suffix key in the
			// middle" cases above, this path contains a SECOND real
			// (space-prefixed) " agent_start_retries=<digits>" match, not
			// just the bare key text -- the exact shape needed to tell
			// "the LAST occurrence" apart from "the FIRST occurrence"
			// (strings.LastIndex vs. strings.Index). A first-occurrence
			// implementation would stop at the middle match, see
			// "42/y.md agent_start_retries=7" as its "digits" (fails
			// isAllDigits), and give up stripping anything at all --
			// proven via red/green (see /test's report).
			name:     "two real retries-suffix matches: only the last is stripped",
			details:  codexPromptFileDetailsPrefix + "/state/x agent_start_retries=42/y.md agent_start_retries=7",
			wantPath: "/state/x agent_start_retries=42/y.md",
			wantOK:   true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path, ok := promptPathFromAgentStartedDetails(c.details)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if ok && path != c.wantPath {
				t.Fatalf("path = %q, want %q", path, c.wantPath)
			}
		})
	}
}

// TestOrgStop_Codex_RecoversPromptPathAfterAgentStartRetry is an
// integration test: when AgentStart needed a retry (agent_pane_busy, the
// same transient error agentStartWithRetry's other tests drive via
// fakeHerdr.agentStartErrs), Spawn's agent_started Details carries a
// trailing " agent_start_retries=N" after the prompt path.
// codexSpawnCorrelation must still recover the real path so Stop's
// second-chance observation still works -- exactly for the slow-starting
// seats it exists to help, since a busy pane is the same slow-start
// situation in which the spawn-time observation is most likely to have
// come back unknown.
func TestOrgStop_Codex_RecoversPromptPathAfterAgentStartRetry(t *testing.T) {
	o, h, _ := codexGuardedOrg(t, "implementer")
	o.AgentStartRetryInterval = time.Millisecond
	h.agentStartErrs = []error{agentPaneBusyErr(), nil}
	p := mustCodexSpawnParams("org-a", "seat-1")

	spawnResult := o.Spawn(p)
	if spawnResult.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", spawnResult)
	}
	if spawnResult.ModelReceipt.Honored != HonoredUnknown {
		t.Fatalf("expected spawn's own receipt to be unknown (no fixture yet), got %+v", spawnResult.ModelReceipt)
	}

	// Test setup invariant: the retry actually happened, and the persisted
	// Details carries the exact suffix this fix must see through.
	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var agentStartedDetails string
	for _, ev := range rr.Events {
		if ev.OrgID == "org-a" && ev.SeatID == "seat-1" && ev.Event == EventSpawnStep && strings.HasPrefix(ev.Details, "agent_started") {
			agentStartedDetails = ev.Details
		}
	}
	assertDetailsContains(t, agentStartedDetails, "agent_start_retries=1")

	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, time.Now().Add(time.Second), "gpt-5-codex")

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt.Honored != HonoredTrue || result.ModelReceipt.ReportedEffectiveModel != "gpt-5-codex" {
		t.Fatalf("expected Stop to recover the prompt path and observe successfully despite the retry suffix, got %+v", result.ModelReceipt)
	}

	mrr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := mrr.Events[len(mrr.Events)-1]
	assertDetailsContains(t, last.Details, "model_observed=true")
}

// TestOrgStop_Codex_SameSecondRespawn_PreviousReceiptDoesNotSuppressObservation
// is an integration test: a previous spawn attempt (A) of this exact seat
// was already observed, its receipt sitting at exactly
// the same whole-second timestamp a respawn (B) of the same seat gets for
// its own spawn_started -- reproducing "stop followed by spawn of the
// same seat within one second", what a lead does when it replaces a seat
// and what a scripted flow can do in milliseconds. A's receipt must not
// suppress B's own stop-time observation.
func TestOrgStop_Codex_SameSecondRespawn_PreviousReceiptDoesNotSuppressObservation(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	p := mustCodexSpawnParams("org-a", "seat-1")

	const sameSecond = "2026-09-10T01:00:00Z" // safely in the past: real wall-clock "now" during this test is always after it
	fixedNow, err := time.Parse(time.RFC3339, sameSecond)
	if err != nil {
		t.Fatalf("parse fixed clock time: %v", err)
	}
	if err := o.Receipts.Append(Receipt{
		TS: sameSecond, OrgID: "org-a", SeatID: "seat-1", Role: "implementer", Driver: "codex",
		CommandedModel: "gpt-5.5", Honored: HonoredTrue, ReportedEffectiveModel: "gpt-5.5",
	}); err != nil {
		t.Fatalf("seed spawn A's observed receipt: %v", err)
	}

	// A constant fake clock: every o.now()/o.nowTime() call during B's own
	// spawn (below) and stop returns exactly the same instant as A's
	// seeded receipt above -- the exact same-second tie
	// hasObservedCodexReceiptAfter must not be fooled by.
	o.Now = func() time.Time { return fixedNow }

	spawnResult := o.Spawn(p)
	if spawnResult.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("respawn B failed: %+v", spawnResult)
	}
	if spawnResult.ModelReceipt.Honored != HonoredUnknown {
		t.Fatalf("expected B's own spawn-time receipt to be unknown (no fixture yet), got %+v", spawnResult.ModelReceipt)
	}

	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	// Relative to the fake clock's own date (not real wall-clock "now"),
	// so the fixture lands in the same date-directory window B's own
	// spawnStartedAt derives.
	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, fixedNow.Add(time.Second), "gpt-5-codex")

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt.Honored != HonoredTrue || result.ModelReceipt.ReportedEffectiveModel != "gpt-5-codex" {
		t.Fatalf("expected Stop to append B's own observed receipt despite A's same-second receipt, got %+v", result.ModelReceipt)
	}

	rr, err := o.Receipts.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	if len(rr.Receipts) != 3 {
		t.Fatalf("expected 3 receipts (A's seeded observed + B's own unknown + B's stop-time observed), got %+v", rr.Receipts)
	}
}

// TestOrgStop_Codex_ObservesSessionThatStartedDaysAfterSpawn is the
// end-to-end case for cycle-2 self-review C2-1
// (docs/reports/self-review-2026-09-20-codex-effective-model-receipt.md):
// the seat's spawn finds no session record yet (a codex model-retirement
// dialog is open),
// so its own receipt stays unknown; three days later the dialog is
// answered and the session record appears; Stop's second-chance
// observation -- now reaching until = o.nowTime(), not just spawnStartedAt
// -- must still find it.
func TestOrgStop_Codex_ObservesSessionThatStartedDaysAfterSpawn(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	p := mustCodexSpawnParams("org-a", "seat-1")

	base := time.Date(2026, 9, 18, 7, 20, 0, 0, time.UTC)
	o.Now = func() time.Time { return base }

	spawnResult := o.Spawn(p)
	if spawnResult.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", spawnResult)
	}
	if spawnResult.ModelReceipt.Honored != HonoredUnknown {
		t.Fatalf("expected spawn's own receipt to be unknown (no session record yet), got %+v", spawnResult.ModelReceipt)
	}

	// The dialog is answered three days later -- advance the fake clock so
	// Stop's own until (o.nowTime()) reaches that day's directory.
	dayThree := base.AddDate(0, 0, 3)
	o.Now = func() time.Time { return dayThree }

	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, dayThree, "gpt-5-codex")

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt.Honored != HonoredTrue || result.ModelReceipt.ReportedEffectiveModel != "gpt-5-codex" {
		t.Fatalf("expected Stop to observe the session that started 3 days after the spawn, got %+v", result.ModelReceipt)
	}

	mrr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := mrr.Events[len(mrr.Events)-1]
	assertDetailsContains(t, last.Details, "model_observed=true")
}

// TestOrgStop_Codex_NothingAppended_NotFound: no qualifying record exists
// at stop time either -- Stop must append nothing and record
// model_observed=none.
func TestOrgStop_Codex_NothingAppended_NotFound(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	p := mustCodexSpawnParams("org-a", "seat-1")
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt != (Receipt{}) {
		t.Fatalf("expected no receipt appended at stop, got %+v", result.ModelReceipt)
	}

	rr, err := o.Receipts.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	if len(rr.Receipts) != 1 {
		t.Fatalf("expected only spawn's own unknown receipt, got %+v", rr.Receipts)
	}

	mrr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := mrr.Events[len(mrr.Events)-1]
	assertDetailsContains(t, last.Details, "model_observed=none")
}

// TestOrgStop_Codex_DryRun_NoObservationAttempted: a dry-run stop of a
// codex seat never observes at all -- no model_observed token of any kind
// on the stopped event's Details, not even "none".
func TestOrgStop_Codex_DryRun_NoObservationAttempted(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	p := mustCodexSpawnParams("org-a", "seat-1")
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1", DryRun: true})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt != (Receipt{}) {
		t.Fatalf("expected no receipt appended for a dry-run stop, got %+v", result.ModelReceipt)
	}

	mrr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := mrr.Events[len(mrr.Events)-1]
	if strings.Contains(last.Details, "model_observed=") {
		t.Fatalf("expected no model_observed token at all for a dry-run stop, got %q", last.Details)
	}
}

// TestOrgStop_Claude_NoObservationAttempted: a claude seat's stopped event
// never carries a model_observed token -- observation is codex-only.
func TestOrgStop_Claude_NoObservationAttempted(t *testing.T) {
	o, _, _ := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt != (Receipt{}) {
		t.Fatalf("expected no receipt appended for a claude seat, got %+v", result.ModelReceipt)
	}

	mrr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := mrr.Events[len(mrr.Events)-1]
	if strings.Contains(last.Details, "model_observed=") {
		t.Fatalf("expected no model_observed token at all for a claude seat, got %q", last.Details)
	}
}

// TestOrgStop_Codex_NoPromptFileSeat_NothingAppended: a codex seat whose
// initial prompt was short enough to pass inline (no role-prompt file, so
// codexSpawnCorrelation's promptPath comes back empty) has nothing to
// correlate a session record with -- Stop appends nothing and records
// model_observed=none.
func TestOrgStop_Codex_NoPromptFileSeat_NothingAppended(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "worker") // "worker" has no embedded template
	p := mustSpawnParams("org-a", "seat-1")
	p.Driver = "codex"
	p.Model = "gpt-5-codex"
	p.Prompt = "short inline prompt"
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed, got %v", result.Err)
	}
	if result.ModelReceipt != (Receipt{}) {
		t.Fatalf("expected no receipt appended for a seat with no role-prompt file, got %+v", result.ModelReceipt)
	}

	mrr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := mrr.Events[len(mrr.Events)-1]
	assertDetailsContains(t, last.Details, "model_observed=none")
}

// TestOrgStop_Codex_ReceiptsAppendFailureLeavesStopSuccessful uses the same
// read-only-file technique as
// TestOrgSend_AppendEventFailsAfterEnter_ReportsEnterPressedButUnrecorded
// (above): a receipts append failure at stop time must never turn a
// successful stop into an error, and must record model_observed=none since
// nothing was actually persisted.
func TestOrgStop_Codex_ReceiptsAppendFailureLeavesStopSuccessful(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores a read-only file's permission bit")
	}
	o, _, _ := codexGuardedOrg(t, "implementer")
	p := mustCodexSpawnParams("org-a", "seat-1")
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, time.Now().Add(time.Second), "gpt-5-codex")

	receiptsPath := o.Receipts.Path()
	if err := os.Chmod(receiptsPath, 0o444); err != nil {
		t.Fatalf("chmod receipts read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(receiptsPath, 0o644) })

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed despite a receipts append failure, got %v", result.Err)
	}
	if result.ModelReceipt != (Receipt{}) {
		t.Fatalf("expected zero ModelReceipt when the append failed, got %+v", result.ModelReceipt)
	}

	mrr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := mrr.Events[len(mrr.Events)-1]
	assertDetailsContains(t, last.Details, "model_observed=none")
}

// TestOrgStop_Codex_ObservationCutShort_NothingAppendedStopStillSucceeds is
// AC-6: Stop's own single observation is bounded by the same
// o.codexModelObserveTimeout() budget Spawn's poll uses. A genuinely
// qualifying fixture exists on disk, but the budget is exhausted (pinned
// to a single nanosecond, so obsCtx is already done by the observer's own
// first ctx.Err() check -- deterministic, no real sleeping) before any
// pass can complete: Stop must append no receipt, record
// model_observed=none (the same token an ordinary not-found gets), and
// still succeed.
func TestOrgStop_Codex_ObservationCutShort_NothingAppendedStopStillSucceeds(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	p := mustCodexSpawnParams("org-a", "seat-1")
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, time.Now().Add(time.Second), "gpt-5-codex")

	// Only affects Stop's own observation below -- Spawn's poll above
	// already ran (and found nothing, honored=unknown) under testOrg's own
	// tiny-but-workable default timeout.
	o.CodexModelObserveTimeout = time.Nanosecond

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected Stop to succeed despite the observation being cut short, got %v", result.Err)
	}
	if result.ModelReceipt != (Receipt{}) {
		t.Fatalf("expected no receipt appended when the observation is cut short, got %+v", result.ModelReceipt)
	}

	mrr2, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last2 := mrr2.Events[len(mrr2.Events)-1]
	assertDetailsContains(t, last2.Details, "model_observed=none")
}
