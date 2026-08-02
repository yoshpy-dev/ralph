package org

import (
	"errors"
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

// TestOrgStop_ExistingSeat_RecordsPaneAndDespawnOutcomes covers AC-5: Stop on
// a real, existing seat best-effort-calls both PaneSendKeys(C-c) and
// Agmsg.Despawn, and records both outcomes in the stopped event's Details --
// including when Despawn itself fails, so status stays truthful without the
// verb itself failing.
func TestOrgStop_ExistingSeat_RecordsPaneAndDespawnOutcomes(t *testing.T) {
	o, _, a := testOrg(t)
	a.despawnErr = errors.New("stub failure: despawn")

	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}

	result := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if result.Err != nil {
		t.Fatalf("expected nil Err: despawn failure is best-effort and must not fail Stop, got %v", result.Err)
	}

	if len(a.despawnCalls) != 1 {
		t.Fatalf("expected exactly one Despawn call, got %+v", a.despawnCalls)
	}
	if a.despawnCalls[0].name != "seat-1" || a.despawnCalls[0].from != "lead" {
		t.Errorf("expected Despawn(team, lead, seat-1), got %+v", a.despawnCalls[0])
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	last := rr.Events[len(rr.Events)-1]
	if last.Event != EventStopped {
		t.Fatalf("expected last event stopped, got %q", last.Event)
	}
	assertDetailsContains(t, last.Details, "pane=ok", "despawn=failed", "stub failure: despawn")

	statusResult, err := o.Status("org-a", false)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(statusResult.Seats) != 1 || statusResult.Seats[0].Active {
		t.Fatalf("expected seat-1 inactive after stop despite despawn failure, got %+v", statusResult.Seats)
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
	despawnBefore := len(a.despawnCalls)

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
	if got := len(a.despawnCalls) - despawnBefore; got != 1 {
		t.Fatalf("expected exactly 1 new Despawn call (only seat-1 was active), got %d", got)
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
