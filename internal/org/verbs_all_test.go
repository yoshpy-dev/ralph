package org

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// spawnSeatIn spawns seatID in orgID on pane. workspace is the id the fake
// hands out if this is the org's first spawn (later spawns reuse the open
// one).
func spawnSeatIn(t *testing.T, o *Org, h *fakeHerdr, orgID, seatID, workspace, pane string) {
	t.Helper()
	h.workspaceID, h.paneID = workspace, pane
	if r := o.Spawn(mustSpawnParams(orgID, seatID)); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn %s/%s failed: %+v", orgID, seatID, r)
	}
}

// failedSeatRefs returns the (org_id, seat_id) of each failure, in order.
func failedSeatRefs(fs []OrgSeatFailure) []OrgSeat {
	out := make([]OrgSeat, 0, len(fs))
	for _, f := range fs {
		out = append(out, OrgSeat{OrgID: f.OrgID, SeatID: f.SeatID})
	}
	return out
}

// failedWorkspaceRefs returns the (org_id, workspace id) of each failure, in
// order.
func failedWorkspaceRefs(fs []OrgWorkspaceFailure) []OrgWorkspace {
	out := make([]OrgWorkspace, 0, len(fs))
	for _, f := range fs {
		out = append(out, OrgWorkspace{OrgID: f.OrgID, WorkspaceID: f.WorkspaceID})
	}
	return out
}

// activeRealSeats returns the real seats Roster derives as active.
func activeRealSeats(t *testing.T, o *Org) []OrgSeat {
	t.Helper()
	var out []OrgSeat
	for _, s := range Roster(mustReadEvents(t, o), RosterOptions{}) {
		if s.Active {
			out = append(out, OrgSeat{OrgID: s.OrgID, SeatID: s.SeatID})
		}
	}
	return out
}

// hangingHerdr is a fakeHerdr whose PaneSendKeys and PaneClose for the panes
// in panes, and WorkspaceClose for the workspaces in workspaces, record the
// call and then do not return until ctx is done -- one org's herdr calls
// never answering while the other orgs' calls do (plan AC11).
type hangingHerdr struct {
	*fakeHerdr
	panes      map[string]bool
	workspaces map[string]bool
}

func (h *hangingHerdr) PaneSendKeys(ctx context.Context, paneID string, keys ...string) error {
	err := h.fakeHerdr.PaneSendKeys(ctx, paneID, keys...)
	if h.panes[paneID] {
		<-ctx.Done()
		return ctx.Err()
	}
	return err
}

func (h *hangingHerdr) PaneClose(ctx context.Context, paneID string) error {
	err := h.fakeHerdr.PaneClose(ctx, paneID)
	if h.panes[paneID] {
		<-ctx.Done()
		return ctx.Err()
	}
	return err
}

func (h *hangingHerdr) WorkspaceClose(ctx context.Context, workspaceID string) error {
	err := h.fakeHerdr.WorkspaceClose(ctx, workspaceID)
	if h.workspaces[workspaceID] {
		<-ctx.Done()
		return ctx.Err()
	}
	return err
}

// TestOrgStopAll_StopsEveryActiveSeatAcrossOrgsInOrder covers plan AC4's
// success path: StopAll stops every active real seat of every org_id, in
// org_id then seat_id order whatever the spawn order, closing each seat's
// pane and no workspace. A seat already stopped and a dry-run seat are left
// alone.
func TestOrgStopAll_StopsEveryActiveSeatAcrossOrgsInOrder(t *testing.T) {
	o, h, a := testOrg(t)
	spawnSeatIn(t, o, h, "org-c", "seat-1", "ws-c", "pane-c1")
	spawnSeatIn(t, o, h, "org-a", "seat-2", "ws-a", "pane-a2")
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-a1")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
	spawnSeatIn(t, o, h, "org-b", "seat-gone", "ws-b", "pane-b-gone")
	if r := o.Stop(StopParams{OrgID: "org-b", Seat: "seat-gone"}); r.Err != nil {
		t.Fatalf("pre-stop org-b/seat-gone: %v", r.Err)
	}
	dry := mustSpawnParams("org-d", "seat-1")
	dry.DryRun = true
	if r := o.Spawn(dry); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("dry-run spawn failed: %+v", r)
	}
	closesBefore, leavesBefore := len(h.paneCloseCalls), len(a.leaveCalls)
	eventsBefore := len(mustReadEvents(t, o))

	result := o.StopAll(StopAllParams{})
	if len(result.Errs) != 0 || len(result.FailedSeats) != 0 || result.DeferredSelfPaneID != "" {
		t.Fatalf("expected a clean stop of every seat, got %+v", result)
	}
	want := []OrgSeat{{"org-a", "seat-1"}, {"org-a", "seat-2"}, {"org-b", "seat-1"}, {"org-c", "seat-1"}}
	if !slices.Equal(result.StoppedSeats, want) {
		t.Fatalf("StoppedSeats = %v, want %v", result.StoppedSeats, want)
	}
	if got := h.paneCloseCalls[closesBefore:]; !slices.Equal(got, []string{"pane-a1", "pane-a2", "pane-b1", "pane-c1"}) {
		t.Fatalf("pane closes = %v, want the four active seats' panes in order", got)
	}
	if len(h.workspaceCloseCalls) != 0 {
		t.Fatalf("expected stop --all to close no workspace, got %v", h.workspaceCloseCalls)
	}
	if got := len(a.leaveCalls) - leavesBefore; got != 4 {
		t.Fatalf("expected 4 Leave calls, got %d", got)
	}
	if active := activeRealSeats(t, o); len(active) != 0 {
		t.Fatalf("expected no active real seat after StopAll, got %v", active)
	}
	for _, ev := range mustReadEvents(t, o)[eventsBefore:] {
		if ev.Event != EventStopped || ev.DryRun || ev.OrgID == "org-d" || ev.SeatID == "seat-gone" {
			t.Fatalf("expected only real stopped events for the four active seats, got %+v", ev)
		}
	}
}

// TestOrgStopAll_OneSeatFails_OthersStoppedThenRetryStopsIt covers plan AC4's
// failure path and AC2's recovery through --all: a seat whose pane close
// fails is in FailedSeats (stop_failed, still active) while the seats after
// it are still stopped, and the error names <org_id>/<seat_id>. Once herdr
// answers, StopAll again stops only that seat, with no error.
func TestOrgStopAll_OneSeatFails_OthersStoppedThenRetryStopsIt(t *testing.T) {
	o, h, _ := testOrg(t)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-a1")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
	spawnSeatIn(t, o, h, "org-c", "seat-1", "ws-c", "pane-c1")
	closeErr := errors.New("herdr pane close: connect: connection refused")
	h.paneCloseErrs = map[string]error{"pane-b1": closeErr}

	result := o.StopAll(StopAllParams{})
	if !slices.Equal(result.StoppedSeats, []OrgSeat{{"org-a", "seat-1"}, {"org-c", "seat-1"}}) {
		t.Fatalf("StoppedSeats = %v, want org-a/seat-1 and org-c/seat-1", result.StoppedSeats)
	}
	if len(result.FailedSeats) != 1 || result.FailedSeats[0].OrgID != "org-b" || result.FailedSeats[0].SeatID != "seat-1" ||
		result.FailedSeats[0].Forced || !errors.Is(result.FailedSeats[0].Err, closeErr) {
		t.Fatalf("FailedSeats = %+v, want org-b/seat-1 with the close failure", result.FailedSeats)
	}
	if len(result.Errs) != 1 || !errors.Is(result.Errs[0], closeErr) || !strings.HasPrefix(result.Errs[0].Error(), "org-b/seat-1: ") {
		t.Fatalf("Errs = %v, want one error for org-b/seat-1", result.Errs)
	}
	if failed := eventsNamed(mustReadEvents(t, o), EventStopFailed); len(failed) != 1 || failed[0].OrgID != "org-b" || failed[0].SeatID != "seat-1" {
		t.Fatalf("expected one stop_failed for org-b/seat-1, got %+v", failed)
	}
	if active := activeRealSeats(t, o); !slices.Equal(active, []OrgSeat{{"org-b", "seat-1"}}) {
		t.Fatalf("active seats = %v, want only org-b/seat-1", active)
	}

	delete(h.paneCloseErrs, "pane-b1")
	closesBefore := len(h.paneCloseCalls)
	retry := o.StopAll(StopAllParams{})
	if len(retry.Errs) != 0 || len(retry.FailedSeats) != 0 || !slices.Equal(retry.StoppedSeats, []OrgSeat{{"org-b", "seat-1"}}) {
		t.Fatalf("expected the retry to stop only org-b/seat-1, got %+v", retry)
	}
	if got := h.paneCloseCalls[closesBefore:]; !slices.Equal(got, []string{"pane-b1"}) {
		t.Fatalf("retry pane closes = %v, want only pane-b1", got)
	}
	if active := activeRealSeats(t, o); len(active) != 0 {
		t.Fatalf("expected no active seat after the retry, got %v", active)
	}
}

// TestOrgStopAll_Force_RecordsPastCloseFailure covers plan AC10 through
// --all: with Force, a seat whose pane could not be closed is recorded
// stopped (forced) and comes back as a Forced failure, with no error.
func TestOrgStopAll_Force_RecordsPastCloseFailure(t *testing.T) {
	o, h, _ := testOrg(t)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-a1")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
	closeErr := errors.New("herdr pane close: connect: connection refused")
	h.paneCloseErrs = map[string]error{"pane-a1": closeErr}

	result := o.StopAll(StopAllParams{Force: true})
	if len(result.Errs) != 0 {
		t.Fatalf("expected a forced stop to return no error, got %v", result.Errs)
	}
	if !slices.Equal(result.StoppedSeats, []OrgSeat{{"org-b", "seat-1"}}) {
		t.Fatalf("StoppedSeats = %v, want org-b/seat-1", result.StoppedSeats)
	}
	if len(result.FailedSeats) != 1 || result.FailedSeats[0].OrgID != "org-a" || !result.FailedSeats[0].Forced || !errors.Is(result.FailedSeats[0].Err, closeErr) {
		t.Fatalf("FailedSeats = %+v, want org-a/seat-1 as a forced failure", result.FailedSeats)
	}
	events := mustReadEvents(t, o)
	if len(eventsNamed(events, EventStopFailed)) != 0 {
		t.Fatalf("expected no stop_failed under Force, got %v", eventNames(t, o))
	}
	for _, ev := range eventsNamed(events, EventStopped) {
		if ev.OrgID == "org-a" {
			assertDetailsContains(t, ev.Details, "pane=close failed (forced): ", "connection refused")
		}
	}
	if active := activeRealSeats(t, o); len(active) != 0 {
		t.Fatalf("expected no active seat after a forced StopAll, got %v", active)
	}
}

// TestOrgStopAll_DryRun_NoDriverCallsDryRunRecordsOnly covers plan AC7 for
// stop --all: no herdr or agmsg call, one dry-run stopped per active seat,
// the real seats still active. The caller's own pane does not reorder a dry
// run or come back deferred.
func TestOrgStopAll_DryRun_NoDriverCallsDryRunRecordsOnly(t *testing.T) {
	o, h, a := testOrg(t)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-a1")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
	ownPaneEnv(o, "pane-a1", "ws-a")
	herdrBefore, agmsgBefore := len(h.calls), len(a.calls)
	eventsBefore := len(mustReadEvents(t, o))

	result := o.StopAll(StopAllParams{DryRun: true})
	if len(result.Errs) != 0 || len(result.FailedSeats) != 0 || result.DeferredSelfPaneID != "" {
		t.Fatalf("expected a clean record-only dry run, got %+v", result)
	}
	if !slices.Equal(result.StoppedSeats, []OrgSeat{{"org-a", "seat-1"}, {"org-b", "seat-1"}}) {
		t.Fatalf("StoppedSeats = %v, want both seats in org_id order", result.StoppedSeats)
	}
	if len(h.calls) != herdrBefore || len(a.calls) != agmsgBefore {
		t.Fatalf("expected no driver call, got herdr %v agmsg %v", h.calls[herdrBefore:], a.calls[agmsgBefore:])
	}
	added := mustReadEvents(t, o)[eventsBefore:]
	if len(added) != 2 {
		t.Fatalf("expected 2 dry-run events, got %+v", added)
	}
	for _, ev := range added {
		if ev.Event != EventStopped || !ev.DryRun {
			t.Fatalf("expected only dry-run stopped events, got %+v", ev)
		}
	}
	if active := activeRealSeats(t, o); len(active) != 2 {
		t.Fatalf("expected both real seats still active after a dry run, got %v", active)
	}
}

// TestOrgStopAll_UnansweredHerdrForOneOrg_OthersStillStopped covers plan AC11
// for stop --all: when one org's herdr calls never answer, each call is cut
// off by DriverCallTimeout, that seat is a failure, and the seats of the
// orgs after it are still stopped, within a bounded time.
func TestOrgStopAll_UnansweredHerdrForOneOrg_OthersStillStopped(t *testing.T) {
	o, h, _ := testOrg(t)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-a1")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
	spawnSeatIn(t, o, h, "org-c", "seat-1", "ws-c", "pane-c1")
	o.Herdr = &hangingHerdr{fakeHerdr: h, panes: map[string]bool{"pane-b1": true}}
	o.DriverCallTimeout = 20 * time.Millisecond

	start := time.Now()
	result := o.StopAll(StopAllParams{})
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("StopAll took %v: the per-call timeout did not bound the unanswered calls", elapsed)
	}
	if !slices.Equal(result.StoppedSeats, []OrgSeat{{"org-a", "seat-1"}, {"org-c", "seat-1"}}) {
		t.Fatalf("StoppedSeats = %v, want org-a and org-c stopped", result.StoppedSeats)
	}
	if len(result.FailedSeats) != 1 || result.FailedSeats[0].OrgID != "org-b" || !errors.Is(result.FailedSeats[0].Err, context.DeadlineExceeded) {
		t.Fatalf("FailedSeats = %+v, want org-b/seat-1 timed out", result.FailedSeats)
	}
	if !slices.Equal(h.paneCloseCalls, []string{"pane-a1", "pane-b1", "pane-c1"}) {
		t.Fatalf("pane closes = %v, want pane-c1 attempted after the hung pane-b1", h.paneCloseCalls)
	}
}

// TestOrgStopAll_EmptyManifest_NothingToDo: no manifest at all means no seat
// to stop and no error.
func TestOrgStopAll_EmptyManifest_NothingToDo(t *testing.T) {
	o, h, a := testOrg(t)
	result := o.StopAll(StopAllParams{})
	if len(result.Errs) != 0 || len(result.StoppedSeats) != 0 || len(result.FailedSeats) != 0 {
		t.Fatalf("expected nothing to do, got %+v", result)
	}
	if len(h.calls) != 0 || len(a.calls) != 0 || len(mustReadEvents(t, o)) != 0 {
		t.Fatalf("expected no call and no event, got herdr %v agmsg %v", h.calls, a.calls)
	}
}

// TestOrgStopAll_ManifestReadFails_StopsNothing: a manifest that cannot be
// read is the only error, and no seat is touched.
func TestOrgStopAll_ManifestReadFails_StopsNothing(t *testing.T) {
	o, h, a := testOrg(t)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-a1")
	herdrBefore, agmsgBefore := len(h.calls), len(a.calls)
	o.Manifest = NewManifestStoreAtPath(t.TempDir()) // a directory: Read fails

	result := o.StopAll(StopAllParams{})
	if len(result.Errs) != 1 || !strings.Contains(result.Errs[0].Error(), "read manifest") {
		t.Fatalf("Errs = %v, want the manifest read failure", result.Errs)
	}
	if len(result.StoppedSeats) != 0 || len(result.FailedSeats) != 0 {
		t.Fatalf("expected nothing stopped or failed, got %+v", result)
	}
	if len(h.calls) != herdrBefore || len(a.calls) != agmsgBefore {
		t.Fatalf("expected no driver call, got herdr %v agmsg %v", h.calls[herdrBefore:], a.calls[agmsgBefore:])
	}
}

// TestOrgStopAll_OwnPaneSeatStoppedLast covers plan AC12 for stop --all: the
// seat in the caller's own pane is stopped after every other seat (although
// its org_id sorts first), recorded without a C-c or close, and handed back
// in DeferredSelfPaneID.
func TestOrgStopAll_OwnPaneSeatStoppedLast(t *testing.T) {
	o, h, _ := testOrg(t)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-own")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
	ownPaneEnv(o, "pane-own", "ws-a")
	sendKeysBefore := len(h.sendKeysCalls)

	result := o.StopAll(StopAllParams{})
	if len(result.Errs) != 0 || len(result.FailedSeats) != 0 {
		t.Fatalf("expected a clean stop, got %+v", result)
	}
	if !slices.Equal(result.StoppedSeats, []OrgSeat{{"org-b", "seat-1"}, {"org-a", "seat-1"}}) {
		t.Fatalf("StoppedSeats = %v, want org-b/seat-1 then the own org-a/seat-1", result.StoppedSeats)
	}
	if result.DeferredSelfPaneID != "pane-own" {
		t.Fatalf("DeferredSelfPaneID = %q, want pane-own", result.DeferredSelfPaneID)
	}
	if !slices.Equal(h.paneCloseCalls, []string{"pane-b1"}) || !slices.Equal(h.sendKeysCalls[sendKeysBefore:], []string{"pane-b1"}) {
		t.Fatalf("expected C-c and close only on pane-b1, got C-c %v closes %v", h.sendKeysCalls[sendKeysBefore:], h.paneCloseCalls)
	}
	stopped := eventsNamed(mustReadEvents(t, o), EventStopped)
	last := stopped[len(stopped)-1]
	if last.OrgID != "org-a" || last.SeatID != "seat-1" {
		t.Fatalf("expected the own seat's stopped last, got %+v", last)
	}
	assertDetailsContains(t, last.Details, "pane=self, closed last")
}

// TestOrgStopAll_OwnPaneSeatLeftRunningWhenAnotherFails: when another seat
// could not be stopped, the seat in the caller's own pane is left running
// (still active, no Leave, nothing recorded) and listed as a failure, and
// nothing is handed back, so the caller survives to report and retry.
func TestOrgStopAll_OwnPaneSeatLeftRunningWhenAnotherFails(t *testing.T) {
	o, h, a := testOrg(t)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-own")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
	ownPaneEnv(o, "pane-own", "ws-a")
	h.paneCloseErrs = map[string]error{"pane-b1": errors.New("herdr pane close: connect: connection refused")}
	leavesBefore := len(a.leaveCalls)

	result := o.StopAll(StopAllParams{})
	if len(result.StoppedSeats) != 0 || result.DeferredSelfPaneID != "" {
		t.Fatalf("expected nothing stopped and nothing deferred, got %+v", result)
	}
	if got := failedSeatRefs(result.FailedSeats); !slices.Equal(got, []OrgSeat{{"org-b", "seat-1"}, {"org-a", "seat-1"}}) {
		t.Fatalf("FailedSeats = %v, want org-b/seat-1 then the own org-a/seat-1", got)
	}
	if !strings.Contains(result.FailedSeats[1].Err.Error(), "caller's own pane") || len(result.Errs) != 2 {
		t.Fatalf("expected the own seat's reason to name the own pane and 2 errors, got %v / %v", result.FailedSeats[1].Err, result.Errs)
	}
	if got := len(a.leaveCalls) - leavesBefore; got != 0 {
		t.Fatalf("expected no Leave (the failed seat skips it, the own seat is untouched), got %d", got)
	}
	if len(eventsNamed(mustReadEvents(t, o), EventStopped)) != 0 {
		t.Fatalf("expected no stopped event, got %v", eventNames(t, o))
	}
	if active := activeRealSeats(t, o); len(active) != 2 {
		t.Fatalf("expected both seats still active, got %v", active)
	}
}

// TestOrgsToDisband_Definition pins which org_ids DisbandAll targets: an org
// with a real seat state event or workspace event after its latest real
// disbanded (or no disbanded at all), which includes every org with an
// active seat, or with a recorded workspace still open. Dry-run events and
// non-state seat events do not count.
func TestOrgsToDisband_Definition(t *testing.T) {
	seat := func(orgID, seatID, event string) ManifestEvent {
		return ManifestEvent{OrgID: orgID, SeatID: seatID, Event: event}
	}
	org := func(orgID, event, workspace string) ManifestEvent {
		return ManifestEvent{OrgID: orgID, Event: event, PaneID: workspace}
	}
	dry := func(ev ManifestEvent) ManifestEvent {
		ev.DryRun = true
		return ev
	}
	tests := []struct {
		name   string
		events []ManifestEvent
		want   []string
	}{
		{"active seat, never disbanded", []ManifestEvent{
			seat("x", "s1", EventSpawned),
		}, []string{"x"}},
		{"stopped seat, no workspace, never disbanded", []ManifestEvent{
			seat("x", "s1", EventSpawned), seat("x", "s1", EventStopped),
		}, []string{"x"}},
		{"cleanly disbanded, nothing after", []ManifestEvent{
			org("x", EventOrgWorkspaceCreated, "ws"), seat("x", "s1", EventSpawned), seat("x", "s1", EventStopped),
			org("x", EventOrgWorkspaceClosed, "ws"), org("x", EventDisbanded, ""),
		}, nil},
		{"workspace left open by an older disband", []ManifestEvent{
			org("x", EventOrgWorkspaceCreated, "ws"), seat("x", "s1", EventSpawned), seat("x", "s1", EventStopped),
			org("x", EventDisbanded, ""),
		}, []string{"x"}},
		{"seat spawned again after disbanded", []ManifestEvent{
			seat("x", "s1", EventSpawned), org("x", EventDisbanded, ""), seat("x", "s1", EventSpawned),
		}, []string{"x"}},
		{"failed spawn after disbanded", []ManifestEvent{
			seat("x", "s1", EventSpawned), org("x", EventDisbanded, ""),
			seat("x", "s2", EventSpawnStarted), seat("x", "s2", EventSpawnFailed),
		}, []string{"x"}},
		{"workspace closed after disbanded", []ManifestEvent{
			org("x", EventOrgWorkspaceCreated, "ws"), org("x", EventDisbanded, ""), org("x", EventOrgWorkspaceClosed, "ws"),
		}, []string{"x"}},
		{"only non-state seat events after disbanded", []ManifestEvent{
			seat("x", "s1", EventSpawned), seat("x", "s1", EventStopped), org("x", EventDisbanded, ""),
			seat("x", "leader", EventSent), seat("x", "s1", EventStopFailed),
		}, nil},
		{"dry-run only org", []ManifestEvent{
			dry(seat("x", "s1", EventSpawned)),
		}, nil},
		{"dry-run spawn after a real disbanded", []ManifestEvent{
			seat("x", "s1", EventSpawned), seat("x", "s1", EventStopped), org("x", EventDisbanded, ""),
			dry(seat("x", "s2", EventSpawned)), dry(org("x", EventOrgWorkspaceCreated, "ws")),
		}, nil},
		{"a dry-run disbanded does not count", []ManifestEvent{
			seat("x", "s1", EventSpawned), seat("x", "s1", EventStopped), dry(org("x", EventDisbanded, "")),
		}, []string{"x"}},
		{"several orgs come back sorted", []ManifestEvent{
			seat("org-c", "s1", EventSpawned), seat("org-a", "s1", EventSpawned),
			seat("org-b", "s1", EventSpawned), org("org-b", EventDisbanded, ""),
		}, []string{"org-a", "org-c"}},
		{"empty manifest", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := orgsToDisband(tt.events)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("orgsToDisband = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestOrgDisbandAll_TargetsOrgsThatNeedDisbanding covers plan AC5's success
// path: an org with active seats has them stopped and its workspace closed;
// an org whose seats were already stopped has its open workspace closed; an
// org already disbanded with nothing open is skipped without any call or
// event. A second DisbandAll finds nothing to do.
func TestOrgDisbandAll_TargetsOrgsThatNeedDisbanding(t *testing.T) {
	o, h, _ := testOrg(t)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-a1")
	spawnSeatIn(t, o, h, "org-a", "seat-2", "ws-a", "pane-a2")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
	if r := o.Stop(StopParams{OrgID: "org-b", Seat: "seat-1"}); r.Err != nil {
		t.Fatalf("pre-stop org-b/seat-1: %v", r.Err)
	}
	spawnSeatIn(t, o, h, "org-c", "seat-1", "ws-c", "pane-c1")
	if r := o.Disband(DisbandParams{OrgID: "org-c"}); !r.Disbanded {
		t.Fatalf("pre-disband org-c: %+v", r)
	}
	closesBefore, wsBefore := len(h.paneCloseCalls), len(h.workspaceCloseCalls)
	eventsBefore := len(mustReadEvents(t, o))

	result := o.DisbandAll(DisbandAllParams{})
	if len(result.Errs) != 0 || len(result.FailedSeats) != 0 || len(result.FailedWorkspaces) != 0 {
		t.Fatalf("expected a clean disband of every org, got %+v", result)
	}
	if !slices.Equal(result.Orgs, []string{"org-a", "org-b"}) || !slices.Equal(result.DisbandedOrgs, []string{"org-a", "org-b"}) {
		t.Fatalf("Orgs = %v, DisbandedOrgs = %v; want org-a and org-b, org-c skipped", result.Orgs, result.DisbandedOrgs)
	}
	if !slices.Equal(result.StoppedSeats, []OrgSeat{{"org-a", "seat-1"}, {"org-a", "seat-2"}}) {
		t.Fatalf("StoppedSeats = %v, want org-a's two seats", result.StoppedSeats)
	}
	if !slices.Equal(result.ClosedWorkspaces, []OrgWorkspace{{"org-a", "ws-a"}, {"org-b", "ws-b"}}) {
		t.Fatalf("ClosedWorkspaces = %v, want ws-a and ws-b", result.ClosedWorkspaces)
	}
	if got := h.paneCloseCalls[closesBefore:]; !slices.Equal(got, []string{"pane-a1", "pane-a2"}) {
		t.Fatalf("pane closes = %v, want org-a's panes only", got)
	}
	if got := h.workspaceCloseCalls[wsBefore:]; !slices.Equal(got, []string{"ws-a", "ws-b"}) {
		t.Fatalf("workspace closes = %v, want ws-a then ws-b", got)
	}
	added := mustReadEvents(t, o)[eventsBefore:]
	var disbanded []string
	for _, ev := range added {
		if ev.OrgID == "org-c" {
			t.Fatalf("expected no event for the already disbanded org-c, got %+v", ev)
		}
		if ev.Event == EventDisbanded {
			disbanded = append(disbanded, ev.OrgID)
		}
	}
	if !slices.Equal(disbanded, []string{"org-a", "org-b"}) {
		t.Fatalf("disbanded events for %v, want org-a and org-b", disbanded)
	}

	eventsBefore = len(mustReadEvents(t, o))
	again := o.DisbandAll(DisbandAllParams{})
	if len(again.Orgs) != 0 || len(again.Errs) != 0 {
		t.Fatalf("expected a second DisbandAll to find nothing, got %+v", again)
	}
	if got := len(mustReadEvents(t, o)); got != eventsBefore {
		t.Fatalf("expected no event from the second DisbandAll, %d -> %d", eventsBefore, got)
	}
}

// TestOrgDisbandAll_FailedOrgNotDisbandedThenRetried covers plan AC5's
// failure path: an org whose seat cannot be closed gets no disbanded (its
// workspace left open, errors prefixed with its org_id) while the orgs
// around it are disbanded; the next DisbandAll, once herdr answers, targets
// only that org and finishes it.
func TestOrgDisbandAll_FailedOrgNotDisbandedThenRetried(t *testing.T) {
	o, h, _ := testOrg(t)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-a1")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
	spawnSeatIn(t, o, h, "org-b", "seat-2", "ws-b", "pane-b2")
	spawnSeatIn(t, o, h, "org-c", "seat-1", "ws-c", "pane-c1")
	closeErr := errors.New("herdr pane close: connect: connection refused")
	h.paneCloseErrs = map[string]error{"pane-b2": closeErr}

	result := o.DisbandAll(DisbandAllParams{})
	if !slices.Equal(result.Orgs, []string{"org-a", "org-b", "org-c"}) || !slices.Equal(result.DisbandedOrgs, []string{"org-a", "org-c"}) {
		t.Fatalf("Orgs = %v, DisbandedOrgs = %v; want org-b not disbanded", result.Orgs, result.DisbandedOrgs)
	}
	if !slices.Equal(result.StoppedSeats, []OrgSeat{{"org-a", "seat-1"}, {"org-b", "seat-1"}, {"org-c", "seat-1"}}) {
		t.Fatalf("StoppedSeats = %v", result.StoppedSeats)
	}
	if len(result.FailedSeats) != 1 || result.FailedSeats[0].OrgID != "org-b" || result.FailedSeats[0].SeatID != "seat-2" ||
		result.FailedSeats[0].Forced || !errors.Is(result.FailedSeats[0].Err, closeErr) {
		t.Fatalf("FailedSeats = %+v, want org-b/seat-2 with the close failure", result.FailedSeats)
	}
	if got := failedWorkspaceRefs(result.FailedWorkspaces); !slices.Equal(got, []OrgWorkspace{{"org-b", "ws-b"}}) || result.FailedWorkspaces[0].Forced {
		t.Fatalf("FailedWorkspaces = %+v, want ws-b left open", result.FailedWorkspaces)
	}
	if !slices.Equal(result.ClosedWorkspaces, []OrgWorkspace{{"org-a", "ws-a"}, {"org-c", "ws-c"}}) {
		t.Fatalf("ClosedWorkspaces = %v, want ws-a and ws-c", result.ClosedWorkspaces)
	}
	if len(result.Errs) != 2 || !errors.Is(result.Errs[0], closeErr) {
		t.Fatalf("Errs = %v, want org-b's seat failure and its workspace left open", result.Errs)
	}
	for _, err := range result.Errs {
		if !strings.HasPrefix(err.Error(), `org_id "org-b": `) {
			t.Errorf("expected every error prefixed with org-b's org_id, got %v", err)
		}
	}

	delete(h.paneCloseErrs, "pane-b2")
	retry := o.DisbandAll(DisbandAllParams{})
	if len(retry.Errs) != 0 || !slices.Equal(retry.Orgs, []string{"org-b"}) || !slices.Equal(retry.DisbandedOrgs, []string{"org-b"}) {
		t.Fatalf("expected the retry to target and disband only org-b, got %+v", retry)
	}
	if !slices.Equal(retry.StoppedSeats, []OrgSeat{{"org-b", "seat-2"}}) || !slices.Equal(retry.ClosedWorkspaces, []OrgWorkspace{{"org-b", "ws-b"}}) {
		t.Fatalf("retry stopped %v and closed %v, want org-b/seat-2 and ws-b", retry.StoppedSeats, retry.ClosedWorkspaces)
	}
	if active := activeRealSeats(t, o); len(active) != 0 {
		t.Fatalf("expected no active seat after the retry, got %v", active)
	}
}

// TestOrgDisbandAll_Force_DisbandsPastCloseFailures covers plan AC10 through
// disband --all: Force passes through, so a seat and a workspace that could
// not be closed come back as Forced failures, every org is disbanded, and
// there is no error.
func TestOrgDisbandAll_Force_DisbandsPastCloseFailures(t *testing.T) {
	o, h, _ := testOrg(t)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-a1")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
	paneErr := errors.New("herdr pane close: connect: connection refused")
	wsErr := errors.New("herdr workspace close: connect: connection refused")
	h.paneCloseErrs = map[string]error{"pane-a1": paneErr}
	h.workspaceCloseErrs = map[string]error{"ws-b": wsErr}

	result := o.DisbandAll(DisbandAllParams{Force: true})
	if len(result.Errs) != 0 || !slices.Equal(result.DisbandedOrgs, []string{"org-a", "org-b"}) {
		t.Fatalf("expected a forced disband of both orgs, got %+v", result)
	}
	if len(result.FailedSeats) != 1 || result.FailedSeats[0].OrgID != "org-a" || !result.FailedSeats[0].Forced || !errors.Is(result.FailedSeats[0].Err, paneErr) {
		t.Fatalf("FailedSeats = %+v, want org-a/seat-1 forced", result.FailedSeats)
	}
	if len(result.FailedWorkspaces) != 1 || result.FailedWorkspaces[0].OrgID != "org-b" || !result.FailedWorkspaces[0].Forced || !errors.Is(result.FailedWorkspaces[0].Err, wsErr) {
		t.Fatalf("FailedWorkspaces = %+v, want ws-b forced", result.FailedWorkspaces)
	}
	if !slices.Equal(result.StoppedSeats, []OrgSeat{{"org-b", "seat-1"}}) || !slices.Equal(result.ClosedWorkspaces, []OrgWorkspace{{"org-a", "ws-a"}}) {
		t.Fatalf("stopped %v closed %v, want org-b/seat-1 and ws-a", result.StoppedSeats, result.ClosedWorkspaces)
	}
	if active := activeRealSeats(t, o); len(active) != 0 {
		t.Fatalf("expected no active seat after a forced DisbandAll, got %v", active)
	}
}

// TestOrgDisbandAll_DryRun_NoDriverCallsDryRunRecordsOnly covers plan AC7 for
// disband --all: the dry run targets the orgs a real run would (not an org
// already disbanded), makes no herdr or agmsg call, appends one dry-run
// disbanded per org and nothing else, ignores the caller's own pane, and
// changes nothing a later run targets.
func TestOrgDisbandAll_DryRun_NoDriverCallsDryRunRecordsOnly(t *testing.T) {
	o, h, a := testOrg(t)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-a1")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
	spawnSeatIn(t, o, h, "org-c", "seat-1", "ws-c", "pane-c1")
	if r := o.Disband(DisbandParams{OrgID: "org-c"}); !r.Disbanded {
		t.Fatalf("pre-disband org-c: %+v", r)
	}
	ownPaneEnv(o, "pane-a1", "ws-a")
	herdrBefore, agmsgBefore := len(h.calls), len(a.calls)
	eventsBefore := len(mustReadEvents(t, o))

	result := o.DisbandAll(DisbandAllParams{DryRun: true})
	if len(result.Errs) != 0 || len(result.StoppedSeats) != 0 || len(result.ClosedWorkspaces) != 0 ||
		result.DeferredSelfPaneID != "" || result.DeferredSelfWorkspaceID != "" {
		t.Fatalf("expected a record-only dry run, got %+v", result)
	}
	if !slices.Equal(result.Orgs, []string{"org-a", "org-b"}) || !slices.Equal(result.DisbandedOrgs, []string{"org-a", "org-b"}) {
		t.Fatalf("Orgs = %v, DisbandedOrgs = %v; want org-a and org-b in org_id order", result.Orgs, result.DisbandedOrgs)
	}
	if len(h.calls) != herdrBefore || len(a.calls) != agmsgBefore {
		t.Fatalf("expected no driver call, got herdr %v agmsg %v", h.calls[herdrBefore:], a.calls[agmsgBefore:])
	}
	added := mustReadEvents(t, o)[eventsBefore:]
	if len(added) != 2 {
		t.Fatalf("expected 2 dry-run events, got %+v", added)
	}
	for _, ev := range added {
		if ev.Event != EventDisbanded || !ev.DryRun {
			t.Fatalf("expected only dry-run disbanded events, got %+v", ev)
		}
	}
	if active := activeRealSeats(t, o); len(active) != 2 {
		t.Fatalf("expected org-a and org-b still active after a dry run, got %v", active)
	}

	again := o.DisbandAll(DisbandAllParams{DryRun: true})
	if !slices.Equal(again.Orgs, []string{"org-a", "org-b"}) {
		t.Fatalf("expected the dry run to leave the targets unchanged, got %v", again.Orgs)
	}
}

// TestOrgDisbandAll_UnansweredHerdrForOneOrg_OthersStillDisbanded covers plan
// AC11 for disband --all: one org's pane calls and another org's workspace
// close never answer; each is cut off by DriverCallTimeout, those two orgs
// are not disbanded, and the third org is still disbanded, within a bounded
// time.
func TestOrgDisbandAll_UnansweredHerdrForOneOrg_OthersStillDisbanded(t *testing.T) {
	o, h, _ := testOrg(t)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-a1")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
	spawnSeatIn(t, o, h, "org-c", "seat-1", "ws-c", "pane-c1")
	o.Herdr = &hangingHerdr{fakeHerdr: h, panes: map[string]bool{"pane-a1": true}, workspaces: map[string]bool{"ws-b": true}}
	o.DriverCallTimeout = 20 * time.Millisecond

	start := time.Now()
	result := o.DisbandAll(DisbandAllParams{})
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("DisbandAll took %v: the per-call timeout did not bound the unanswered calls", elapsed)
	}
	if !slices.Equal(result.DisbandedOrgs, []string{"org-c"}) {
		t.Fatalf("DisbandedOrgs = %v, want only org-c", result.DisbandedOrgs)
	}
	if len(result.FailedSeats) != 1 || result.FailedSeats[0].OrgID != "org-a" || !errors.Is(result.FailedSeats[0].Err, context.DeadlineExceeded) {
		t.Fatalf("FailedSeats = %+v, want org-a/seat-1 timed out", result.FailedSeats)
	}
	if got := failedWorkspaceRefs(result.FailedWorkspaces); !slices.Equal(got, []OrgWorkspace{{"org-a", "ws-a"}, {"org-b", "ws-b"}}) {
		t.Fatalf("FailedWorkspaces = %v, want ws-a left open and ws-b timed out", got)
	}
	if !errors.Is(result.FailedWorkspaces[1].Err, context.DeadlineExceeded) {
		t.Fatalf("expected ws-b's close to time out, got %v", result.FailedWorkspaces[1].Err)
	}
	if !slices.Equal(result.StoppedSeats, []OrgSeat{{"org-b", "seat-1"}, {"org-c", "seat-1"}}) {
		t.Fatalf("StoppedSeats = %v, want org-b and org-c stopped", result.StoppedSeats)
	}
	if !slices.Equal(h.workspaceCloseCalls, []string{"ws-b", "ws-c"}) {
		t.Fatalf("workspace closes = %v, want ws-b then ws-c (ws-a left open)", h.workspaceCloseCalls)
	}
}

// TestOrgDisbandAll_EmptyManifest_NothingToDo: no manifest at all means no
// org to disband, no event and no error.
func TestOrgDisbandAll_EmptyManifest_NothingToDo(t *testing.T) {
	o, h, a := testOrg(t)
	result := o.DisbandAll(DisbandAllParams{})
	if len(result.Errs) != 0 || len(result.Orgs) != 0 {
		t.Fatalf("expected nothing to do, got %+v", result)
	}
	if len(h.calls) != 0 || len(a.calls) != 0 || len(mustReadEvents(t, o)) != 0 {
		t.Fatalf("expected no call and no event, got herdr %v agmsg %v", h.calls, a.calls)
	}
}

// TestOrgDisbandAll_ManifestReadFails_DoesNothing: a manifest that cannot be
// read is the only error, and no org is touched.
func TestOrgDisbandAll_ManifestReadFails_DoesNothing(t *testing.T) {
	o, h, a := testOrg(t)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-a1")
	herdrBefore, agmsgBefore := len(h.calls), len(a.calls)
	o.Manifest = NewManifestStoreAtPath(t.TempDir()) // a directory: Read fails

	result := o.DisbandAll(DisbandAllParams{})
	if len(result.Errs) != 1 || !strings.Contains(result.Errs[0].Error(), "read manifest") || len(result.Orgs) != 0 {
		t.Fatalf("expected only the manifest read failure, got %+v", result)
	}
	if len(h.calls) != herdrBefore || len(a.calls) != agmsgBefore {
		t.Fatalf("expected no driver call, got herdr %v agmsg %v", h.calls[herdrBefore:], a.calls[agmsgBefore:])
	}
}

// TestOrgDisbandAll_OwnOrgLast_WorkspaceDeferred covers plan AC12 for
// disband --all: the org holding the caller's own pane and workspace is
// disbanded after every other org (although its org_id sorts first), its
// own workspace is recorded closed but not closed, and it comes back in
// DeferredSelfWorkspaceID with no pane (the pane is inside it).
func TestOrgDisbandAll_OwnOrgLast_WorkspaceDeferred(t *testing.T) {
	o, h, _ := testOrg(t)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-own")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
	ownPaneEnv(o, "pane-own", "ws-a")

	result := o.DisbandAll(DisbandAllParams{})
	if len(result.Errs) != 0 || !slices.Equal(result.DisbandedOrgs, []string{"org-b", "org-a"}) {
		t.Fatalf("expected org-b then the own org-a disbanded, got %+v", result)
	}
	if !slices.Equal(result.Orgs, []string{"org-b", "org-a"}) {
		t.Fatalf("Orgs = %v, want the own org-a last", result.Orgs)
	}
	if result.DeferredSelfWorkspaceID != "ws-a" || result.DeferredSelfPaneID != "" {
		t.Fatalf("deferred workspace %q pane %q, want ws-a and no pane", result.DeferredSelfWorkspaceID, result.DeferredSelfPaneID)
	}
	if !slices.Equal(h.paneCloseCalls, []string{"pane-b1"}) || !slices.Equal(h.workspaceCloseCalls, []string{"ws-b"}) {
		t.Fatalf("expected only org-b's pane and workspace closed inline, got panes %v workspaces %v", h.paneCloseCalls, h.workspaceCloseCalls)
	}
	events := mustReadEvents(t, o)
	n := len(events)
	if events[n-1].Event != EventDisbanded || events[n-1].OrgID != "org-a" {
		t.Fatalf("expected org-a's disbanded last, got %+v", events[n-1])
	}
	if events[n-2].Event != EventOrgWorkspaceClosed || events[n-2].PaneID != "ws-a" || events[n-2].Details != "workspace=self, closed last" {
		t.Fatalf("expected ws-a recorded closed-last before it, got %+v", events[n-2])
	}
}

// TestOrgDisbandAll_OwnOrgLeftAsIsWhenAnotherOrgFails: when another org
// could not be disbanded, the org holding the caller's own pane and
// workspace is not touched at all (no call, no event, its seat still
// active), its seat and workspace are listed as failures, and nothing is
// handed back, so the caller survives to report and retry.
func TestOrgDisbandAll_OwnOrgLeftAsIsWhenAnotherOrgFails(t *testing.T) {
	o, h, a := testOrg(t)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-own")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
	ownPaneEnv(o, "pane-own", "ws-a")
	h.paneCloseErrs = map[string]error{"pane-b1": errors.New("herdr pane close: connect: connection refused")}
	sendKeysBefore, leavesBefore := len(h.sendKeysCalls), len(a.leaveCalls)
	eventsBefore := len(mustReadEvents(t, o))

	result := o.DisbandAll(DisbandAllParams{})
	if len(result.DisbandedOrgs) != 0 || result.DeferredSelfPaneID != "" || result.DeferredSelfWorkspaceID != "" {
		t.Fatalf("expected nothing disbanded and nothing deferred, got %+v", result)
	}
	if !slices.Equal(result.Orgs, []string{"org-b", "org-a"}) {
		t.Fatalf("Orgs = %v, want org-b then the own org-a", result.Orgs)
	}
	if got := failedSeatRefs(result.FailedSeats); !slices.Equal(got, []OrgSeat{{"org-b", "seat-1"}, {"org-a", "seat-1"}}) {
		t.Fatalf("FailedSeats = %v", got)
	}
	if got := failedWorkspaceRefs(result.FailedWorkspaces); !slices.Equal(got, []OrgWorkspace{{"org-b", "ws-b"}, {"org-a", "ws-a"}}) {
		t.Fatalf("FailedWorkspaces = %v", got)
	}
	if !strings.Contains(result.FailedSeats[1].Err.Error(), "caller's own pane or workspace") || len(result.Errs) != 3 {
		t.Fatalf("expected the own org's reason and 3 errors, got %v / %v", result.FailedSeats[1].Err, result.Errs)
	}
	if got := h.sendKeysCalls[sendKeysBefore:]; !slices.Equal(got, []string{"pane-b1"}) || len(h.workspaceCloseCalls) != 0 || len(a.leaveCalls) != leavesBefore {
		t.Fatalf("expected no call on the own org, got C-c %v workspaces %v leaves %+v", got, h.workspaceCloseCalls, a.leaveCalls[leavesBefore:])
	}
	for _, ev := range mustReadEvents(t, o)[eventsBefore:] {
		if ev.OrgID == "org-a" {
			t.Fatalf("expected no event for the own org-a, got %+v", ev)
		}
	}
	if active := activeRealSeats(t, o); !slices.Contains(active, OrgSeat{"org-a", "seat-1"}) {
		t.Fatalf("expected the own seat still active, got %v", active)
	}
}

// TestOrgDisbandAll_OwnPaneAndWorkspaceInDifferentOrgs_WorkspaceSupersedes:
// when one org's Disband hands back the own pane and another org's the own
// workspace, the combined result hands back only the workspace, which holds
// the pane.
func TestOrgDisbandAll_OwnPaneAndWorkspaceInDifferentOrgs_WorkspaceSupersedes(t *testing.T) {
	o, h, _ := testOrg(t)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-own")
	spawnSeatIn(t, o, h, "org-b", "seat-1", "ws-b", "pane-b1")
	ownPaneEnv(o, "pane-own", "ws-b")

	result := o.DisbandAll(DisbandAllParams{})
	if len(result.Errs) != 0 || !slices.Equal(result.DisbandedOrgs, []string{"org-a", "org-b"}) {
		t.Fatalf("expected both orgs disbanded, got %+v", result)
	}
	if result.DeferredSelfWorkspaceID != "ws-b" || result.DeferredSelfPaneID != "" {
		t.Fatalf("deferred workspace %q pane %q, want ws-b and no pane", result.DeferredSelfWorkspaceID, result.DeferredSelfPaneID)
	}
	if !slices.Equal(h.paneCloseCalls, []string{"pane-b1"}) || !slices.Equal(h.workspaceCloseCalls, []string{"ws-a"}) {
		t.Fatalf("expected pane-b1 and ws-a closed inline, got panes %v workspaces %v", h.paneCloseCalls, h.workspaceCloseCalls)
	}
}
