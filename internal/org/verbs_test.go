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
// expiry inside waitBeforeEnter -- see
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
// expiring *inside* waitBeforeEnter, after the fail-closed budget check
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
// real time, leaving roughly 20ms of ctx budget when waitBeforeEnter's
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
		t.Fatal("expected a non-nil Err when ctx expires inside waitBeforeEnter after the budget check passed")
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
// either way. Progress must be SendProgressEnterUnacknowledged (not the
// old, over-confident TextTyped=true/EnterPressed=false), and the error
// text must say the outcome is unknown rather than assert it never
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
// change rules out) and still appends no `sent` event -- but, unlike the
// pre-AR-1 design, Send no longer claims nothing happened.
// exec.CommandContext can kill herdr's CLI mid-call, so the paste may
// already have landed before the error came back: Progress reports
// SendProgressTextUnacknowledged (not the old TextTyped=false) and PaneID
// IS set, so the CLI can still point the operator at the pane to check.
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

// TestOrgSend_AppendEventFailsAfterEnter_ReportsSubmittedButUnrecorded is the
// self-review revalidation NEW-1 fix
// (docs/reports/self-review-2026-09-19-org-send-enter-timing.md): when the
// manifest write for the `sent` event fails AFTER Enter has already
// succeeded (and confirmSubmitted has already run), Send must report
// Progress=SendProgressEnterPressed and wrap the error to say Enter was
// pressed -- this is what lets the CLI tell "very likely delivered, only
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
func TestOrgSend_AppendEventFailsAfterEnter_ReportsSubmittedButUnrecorded(t *testing.T) {
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
// and PaneSendText/waitBeforeEnter actually running, and a 20ms margin was
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
