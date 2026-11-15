package org

import (
	"os"
	"slices"
	"strings"
	"testing"
)

// A reseat whose org is disbanded while it runs (plan
// docs/plans/active/2026-11-12-org-reseat.md, P6; cross-review cycle 3,
// finding 1): right before each stop and each start, Reseat reads the
// manifest again and ends when the org's latest `disbanded` is no longer the
// one PlanReseat read, so it starts no seat in an org with no binding (Spawn)
// and no new life without the old one's approvals (StartFeature), and prints
// no retry, which a reseat of the disbanded org would refuse. On the fixture
// of reseat_test.go (newReseatOrg). The disband is a real Disband on the fake
// drivers, as `ralph org disband` would run it from another command, so the
// manifest holds what a disband writes (the stops, the workspace close and
// `disbanded`), not a `disbanded` appended alone.

// disbandAuthCore returns a func that disbands auth-core with Disband, and
// where the `disbanded` it recorded is put once it ran.
func disbandAuthCore(t *testing.T, st *featureStart) (disband func(), disbanded *ManifestEvent) {
	t.Helper()
	disbanded = new(ManifestEvent)
	return func() {
		if d := st.o.Disband(DisbandParams{OrgID: "auth-core"}); len(d.Errs) != 0 || !d.Disbanded {
			t.Fatalf("disband auth-core: %+v", d)
		}
		*disbanded = eventOf(t, st.o, "auth-core", "", EventDisbanded)
	}, disbanded
}

// disbandBeforeStart disbands auth-core with Disband right before the reseat
// starts seat, and returns where the `disbanded` it recorded is put once it
// ran.
func disbandBeforeStart(t *testing.T, st *featureStart, seat string) *ManifestEvent {
	t.Helper()
	disband, disbanded := disbandAuthCore(t, st)
	intrudeBeforeStart(t, seat, disband)
	return disbanded
}

// intrudeBeforeStop sets reseatTestHooks.beforeStopCheck, for the rest of t,
// to run intrusion once, right before the reseat reads the manifest again to
// check seat before its stop.
func intrudeBeforeStop(t *testing.T, seat string, intrusion func()) {
	t.Helper()
	done := false
	reseatTestHooks.beforeStopCheck = func(id string) {
		if id == seat && !done {
			done = true
			intrusion()
		}
	}
	t.Cleanup(func() { reseatTestHooks.beforeStopCheck = nil })
}

// disbandBeforeStop disbands auth-core with Disband right before the reseat
// checks seat before its stop, and returns where the `disbanded` it recorded
// is put once it ran.
func disbandBeforeStop(t *testing.T, st *featureStart, seat string) *ManifestEvent {
	t.Helper()
	disband, disbanded := disbandAuthCore(t, st)
	intrudeBeforeStop(t, seat, disband)
	return disbanded
}

// assertDisbandedResult checks what every reseat that found its org
// disbanded returns: Disbanded, the error want, no retry, DECISION or read
// of the leader's pane, and no start of the seat it ended before, nor of any
// after it.
func assertDisbandedResult(t *testing.T, res ReseatResult, want string, from int) {
	t.Helper()
	if !res.Disbanded {
		t.Errorf("Disbanded = false; want true")
	}
	if res.Err == nil || res.Err.Error() != want {
		t.Errorf("Reseat error %v;\nwant %q", res.Err, want)
	}
	if res.Retry != "" || res.Decision != "" || res.DecisionRead != "" || res.Changed != "" {
		t.Errorf("retry %q decision %q read %q changed %q; want none, the org being disbanded",
			res.Retry, res.Decision, res.DecisionRead, res.Changed)
	}
	for i, s := range res.Seats[from:] {
		if s.Spawn != nil {
			t.Errorf("seat %q (start %d) was started after the disband: %+v", s.SeatID, from+i, s.Spawn)
		}
	}
}

// TestReseat_OneSeatOfAnOrgDisbandedBeforeItsStart covers a reseat of the
// implementer alone (the leader is not one of its seats) whose org another
// command disbands after the implementer's stop: the reseat does not start the
// implementer, which Spawn would start with no binding and no approval
// check, says why and what starts the org again, and records `reseated`
// after the `disbanded`, the implementer stopped.
func TestReseat_OneSeatOfAnOrgDisbandedBeforeItsStart(t *testing.T) {
	st := newReseatOrg(t)
	disbanded := disbandBeforeStart(t, st, implementerSeatID)
	from := len(mustReadEvents(t, st.o))

	res := st.o.Reseat(st.reseatParams(implementerSeatID))
	if got := reseatOutcomes(res); got != "implementer:stopped" {
		t.Errorf("seats %s; want the implementer stopped and not started", got)
	}
	want := `org: reseat ` + res.Plan.ID + ` of org_id "auth-core" ended before the start of seat "implementer", and ` +
		`started no seat after it: the org was disbanded during the reseat (disbanded at ` + disbanded.TS + `); the ` +
		`feature org starts again with ralph org start --plan`
	assertDisbandedResult(t, res, want, 0)
	if res.Plan.LastDisbandedIndex != -1 {
		t.Errorf("LastDisbandedIndex = %d; want -1, the org having had no disband before the reseat", res.Plan.LastDisbandedIndex)
	}

	added := mustReadEvents(t, st.o)[from:]
	if got := eventSeats(added, EventSpawnStarted); len(got) != 0 {
		t.Errorf("spawn_started %q; want none", got)
	}
	assertReseated(t, added, "id="+res.Plan.ID+" seats=implementer:stopped")
	names := eventNamesOf(added)
	if d, r := slices.Index(names, EventDisbanded), slices.Index(names, EventReseated); d < 0 || r < d {
		t.Errorf("events %q; want the reseated after the disbanded", names)
	}
}

// TestReseat_AnOrgDisbandedAfterTheLeaderStarted covers a reseat of every
// seat whose org another command disbands after the leader's start, right
// before the implementer's: the leader is started, the implementer and the
// reviewer are left stopped, and the error names the leader as started
// before the disband, in the life it ended, without saying the disband
// stopped it.
func TestReseat_AnOrgDisbandedAfterTheLeaderStarted(t *testing.T) {
	st := newReseatOrg(t)
	disbanded := disbandBeforeStart(t, st, implementerSeatID)
	from := len(mustReadEvents(t, st.o))

	res := st.o.Reseat(st.reseatParams())
	if got := reseatOutcomes(res); got != "leader:started,implementer:stopped,reviewer:stopped" {
		t.Errorf("seats %s; want the leader started and the others left stopped", got)
	}
	want := `org: reseat ` + res.Plan.ID + ` of org_id "auth-core" ended before the start of seat "implementer", and ` +
		`started no seat after it: the org was disbanded during the reseat (disbanded at ` + disbanded.TS + `); seat ` +
		`"leader" started before the disband, in the life it ended, so ralph org status shows whether it still runs; ` +
		`the feature org starts again with ralph org start --plan`
	assertDisbandedResult(t, res, want, 1)

	added := mustReadEvents(t, st.o)[from:]
	if got := eventSeats(added, EventSpawned); !slices.Equal(got, []string{LeaderIdentity}) {
		t.Errorf("spawned %q; want the leader only", got)
	}
	assertReseated(t, added, "id="+res.Plan.ID+" seats=leader:started,implementer:stopped,reviewer:stopped")
}

// TestReseat_NoDecisionAfterADisbandOnceASeatOtherThanTheLeaderStarted
// covers the one shape in which a reseat that found its org disbanded would
// otherwise print a DECISION: the leader is not one of its seats and a seat
// started before the disband (the implementer, before the reviewer's start).
// The DECISION and the read of the leader's pane would go to a leader the
// disband ended, so finish leaves both out, as it leaves out the retry
// (self-review L23: the other disband tests start no seat but the leader).
func TestReseat_NoDecisionAfterADisbandOnceASeatOtherThanTheLeaderStarted(t *testing.T) {
	st := newReseatOrg(t)
	disbanded := disbandBeforeStart(t, st, "reviewer")

	res := st.o.Reseat(st.reseatParams(implementerSeatID, "reviewer"))
	if got := reseatOutcomes(res); got != "implementer:started,reviewer:stopped" {
		t.Errorf("seats %s; want the implementer started and the reviewer left stopped", got)
	}
	want := `org: reseat ` + res.Plan.ID + ` of org_id "auth-core" ended before the start of seat "reviewer", and ` +
		`started no seat after it: the org was disbanded during the reseat (disbanded at ` + disbanded.TS + `); seat ` +
		`"implementer" started before the disband, in the life it ended, so ralph org status shows whether it still ` +
		`runs; the feature org starts again with ralph org start --plan`
	assertDisbandedResult(t, res, want, 1)
}

// TestReseat_AnOrgDisbandedBeforeItsSecondStop covers a reseat of every seat
// whose org another command disbands after the reseat stopped the reviewer,
// right before the implementer's stop: the reseat stops neither the
// implementer nor the leader, which the disband stopped, and starts no seat.
// It says the org was disbanded instead of naming the implementer as a seat
// another command changed, and prints no retry, which a reseat of the
// disbanded org would refuse (reseatNoBindingErr). The reviewer it stopped
// stays stopped, and `reseated` records that after the `disbanded`.
func TestReseat_AnOrgDisbandedBeforeItsSecondStop(t *testing.T) {
	st := newReseatOrg(t)
	disbanded := disbandBeforeStop(t, st, implementerSeatID)
	from := len(mustReadEvents(t, st.o))

	res := st.o.Reseat(st.reseatParams())
	if got := reseatOutcomes(res); got != "leader:not_stopped,implementer:not_stopped,reviewer:stopped" {
		t.Errorf("seats %s; want the reviewer stopped and the others not", got)
	}
	want := `org: reseat ` + res.Plan.ID + ` of org_id "auth-core" ended before the stop of seat "implementer", and ` +
		`started no seat: the org was disbanded during the reseat (disbanded at ` + disbanded.TS + `); the feature ` +
		`org starts again with ralph org start --plan`
	assertDisbandedResult(t, res, want, 0)
	if res.Seats[0].Stop != nil || res.Seats[1].Stop != nil {
		t.Errorf("stops of the leader %+v and the implementer %+v; want none", res.Seats[0].Stop, res.Seats[1].Stop)
	}

	added := mustReadEvents(t, st.o)[from:]
	if got := eventSeats(added, EventSpawnStarted); len(got) != 0 {
		t.Errorf("spawn_started %q; want none", got)
	}
	var byReseat []string
	for _, ev := range eventsNamed(added, EventStopped) {
		if reseatNoteID(ev.Details) == res.Plan.ID {
			byReseat = append(byReseat, ev.SeatID)
		}
	}
	if !slices.Equal(byReseat, []string{"reviewer"}) {
		t.Errorf("seats the reseat stopped %q; want the reviewer only", byReseat)
	}
	assertReseated(t, added, "id="+res.Plan.ID+" seats=leader:not_stopped,implementer:not_stopped,reviewer:stopped")
	names := eventNamesOf(added)
	if d, r := slices.Index(names, EventDisbanded), slices.Index(names, EventReseated); d < 0 || r < d {
		t.Errorf("events %q; want the reseated after the disbanded", names)
	}
}

// TestReseat_AnOrgDisbandedBeforeItsFirstStop covers the org disbanded
// between the reseat's checks and its first stop: the reseat stops and
// starts nothing, so it records nothing either, not even `reseated`, says
// the org was disbanded, and prints no retry.
func TestReseat_AnOrgDisbandedBeforeItsFirstStop(t *testing.T) {
	st := newReseatOrg(t)
	disbanded := disbandBeforeStop(t, st, "reviewer")
	from := len(mustReadEvents(t, st.o))

	res := st.o.Reseat(st.reseatParams())
	if got := reseatOutcomes(res); got != "leader:not_stopped,implementer:not_stopped,reviewer:not_stopped" {
		t.Errorf("seats %s; want none stopped", got)
	}
	want := `org: reseat ` + res.Plan.ID + ` of org_id "auth-core" ended before the stop of seat "reviewer", and ` +
		`started no seat: the org was disbanded during the reseat (disbanded at ` + disbanded.TS + `); the feature ` +
		`org starts again with ralph org start --plan`
	assertDisbandedResult(t, res, want, 0)
	for _, s := range res.Seats {
		if s.Stop != nil {
			t.Errorf("seat %q was stopped by the reseat: %+v", s.SeatID, s.Stop)
		}
	}

	added := mustReadEvents(t, st.o)[from:]
	for _, ev := range added {
		if reseatNoteID(ev.Details) != "" || ev.Event == EventReseated {
			t.Errorf("event %+v; want the disband's events only", ev)
		}
	}
	if got := eventsNamed(added, EventDisbanded); len(got) != 1 {
		t.Errorf("disbanded %+v; want the one of the disband", got)
	}
}

// TestReseat_AManifestUnreadableBeforeAStop covers the manifest that cannot
// be read again right before a stop (a directory in its place): the reseat
// ends there, stopping and starting nothing, without Disbanded or Changed,
// and the retry names every seat.
func TestReseat_AManifestUnreadableBeforeAStop(t *testing.T) {
	st := newReseatOrg(t)
	path := st.o.Manifest.Path()
	intrudeBeforeStop(t, "reviewer", func() {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
	})

	res := st.o.Reseat(st.reseatParams())
	prefix := `org: reseat ` + res.Plan.ID + ` of org_id "auth-core" ended before the stop of seat "reviewer", and ` +
		`started no seat: read the manifest again: `
	if res.Err == nil || !strings.HasPrefix(res.Err.Error(), prefix) {
		t.Errorf("Reseat error %v; want it to start with %q", res.Err, prefix)
	}
	if res.Disbanded || res.Changed != "" {
		t.Errorf("disbanded %v changed %q; want neither, the org not being known to be disbanded", res.Disbanded, res.Changed)
	}
	if got := reseatOutcomes(res); got != "leader:not_stopped,implementer:not_stopped,reviewer:not_stopped" {
		t.Errorf("seats %s; want none stopped", got)
	}
	if want := "ralph org reseat --org-id auth-core --seat leader --seat implementer --seat reviewer"; res.Retry != want {
		t.Errorf("retry %q; want %q", res.Retry, want)
	}
}

// TestReseat_AnOrgDisbandedBeforeTheReseatStartsEverySeat covers the ledger
// that does not change: an org disbanded and started again before the
// reseat has a `disbanded`, the one PlanReseat read, so the reseat of its
// current life starts every seat (the leader, its only seat). Without a
// disband, TestReseat_StopsAndStartsEverySeatInOrder covers it.
func TestReseat_AnOrgDisbandedBeforeTheReseatStartsEverySeat(t *testing.T) {
	st := newReseatOrg(t)
	if d := st.o.Disband(DisbandParams{OrgID: "auth-core"}); len(d.Errs) != 0 || !d.Disbanded {
		t.Fatalf("disband auth-core: %+v", d)
	}
	st.h.workspaceID, st.h.nextPaneID = "ws-2", "pane-2"
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	events := mustReadEvents(t, st.o)
	last := slices.IndexFunc(events, func(ev ManifestEvent) bool { return ev.Event == EventDisbanded })

	res := st.mustReseat(t, st.reseatParams())
	if res.Plan.LastDisbandedIndex != last || last < 0 {
		t.Errorf("LastDisbandedIndex = %d; want %d, the org's disbanded", res.Plan.LastDisbandedIndex, last)
	}
	if got := reseatOutcomes(res); got != "leader:started" || res.Disbanded {
		t.Errorf("seats %s, disbanded %v; want the leader started", got, res.Disbanded)
	}
}

// TestReseat_AManifestUnreadableBeforeAStart covers the manifest that cannot
// be read again right before a start (a directory in its place): the reseat
// ends there, as it does before a stop, without the start and without
// Disbanded.
func TestReseat_AManifestUnreadableBeforeAStart(t *testing.T) {
	st := newReseatOrg(t)
	path := st.o.Manifest.Path()
	intrudeBeforeStart(t, implementerSeatID, func() {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
	})

	res := st.o.Reseat(st.reseatParams(implementerSeatID))
	prefix := `org: reseat ` + res.Plan.ID + ` of org_id "auth-core" ended before the start of seat "implementer", and ` +
		`started no seat after it: read the manifest again: `
	if res.Err == nil || !strings.HasPrefix(res.Err.Error(), prefix) {
		t.Errorf("Reseat error %v; want it to start with %q", res.Err, prefix)
	}
	if res.Disbanded {
		t.Error("Disbanded = true; want false, the org not being known to be disbanded")
	}
	if got := reseatOutcomes(res); got != "implementer:stopped" || res.Seats[0].Spawn != nil {
		t.Errorf("seats %s, spawn %+v; want the implementer stopped and not started", got, res.Seats[0].Spawn)
	}
}

// TestReseatDisbandedErr pins the words of reseatDisbandedErr before a start
// for no seat, one seat and two seats started before the disband, and for a
// `disbanded` with no time, and its words before a stop.
func TestReseatDisbandedErr(t *testing.T) {
	plan := ReseatPlan{ID: "20261113T010203-abcd", OrgID: "auth-core"}
	const head = `org: reseat 20261113T010203-abcd of org_id "auth-core" ended before the start of seat "reviewer", and ` +
		`started no seat after it: the org was disbanded during the reseat`
	const tail = "; the feature org starts again with ralph org start --plan"
	at := ManifestEvent{TS: "2026-11-13T01:02:03Z", Event: EventDisbanded}
	for _, tc := range []struct {
		name      string
		disbanded ManifestEvent
		started   []string
		want      string // between head and tail
	}{
		{"none started", at, nil, " (disbanded at 2026-11-13T01:02:03Z)"},
		{"no time", ManifestEvent{Event: EventDisbanded}, nil, ""},
		{"one started", at, []string{LeaderIdentity}, ` (disbanded at 2026-11-13T01:02:03Z); seat "leader" started ` +
			`before the disband, in the life it ended, so ralph org status shows whether it still runs`},
		{"two started", at, []string{LeaderIdentity, implementerSeatID}, ` (disbanded at 2026-11-13T01:02:03Z); seats ` +
			`"leader", "implementer" started before the disband, in the life it ended, so ralph org status shows whether ` +
			`they still run`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := reseatDisbandedErr(plan, "start", "reviewer", tc.disbanded, tc.started).Error(), head+tc.want+tail; got != want {
				t.Errorf("reseatDisbandedErr =\n%q\nwant\n%q", got, want)
			}
		})
	}
	t.Run("before a stop", func(t *testing.T) {
		want := `org: reseat 20261113T010203-abcd of org_id "auth-core" ended before the stop of seat "reviewer", and ` +
			`started no seat: the org was disbanded during the reseat (disbanded at 2026-11-13T01:02:03Z)` + tail
		if got := reseatDisbandedErr(plan, "stop", "reviewer", at, nil).Error(); got != want {
			t.Errorf("reseatDisbandedErr =\n%q\nwant\n%q", got, want)
		}
	})
}

// eventNamesOf is the name of each of events, in order.
func eventNamesOf(events []ManifestEvent) []string {
	names := make([]string, len(events))
	for i, ev := range events {
		names[i] = ev.Event
	}
	return names
}
