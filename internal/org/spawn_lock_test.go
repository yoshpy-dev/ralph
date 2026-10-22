package org

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// Tests of a seat's spawn lock (spawnLockName; plan
// 2026-10-22-seat-spawn-close, AC9 and D5): a non-dry-run spawn holds it from
// before its first manifest read until it returns, so a second spawn of the
// same seat is refused at once instead of taking the first one's in-flight
// records for a stale saga.

// holdJoin makes a's first Join of agentID into team wait: entered is closed
// when that Join is reached, and the Join answers once release is closed.
// Later Joins, and Joins of other agents or teams, are not held.
func holdJoin(a *fakeAgmsg, team, agentID string) (entered <-chan struct{}, release chan<- struct{}) {
	in, out := make(chan struct{}), make(chan struct{})
	var once sync.Once
	a.mu.Lock()
	defer a.mu.Unlock()
	a.joinHook = func(gotTeam, gotAgent string) {
		if gotTeam != team || gotAgent != agentID {
			return
		}
		held := false
		once.Do(func() {
			held = true
			close(in)
		})
		if held {
			<-out
		}
	}
	return in, out
}

// waitClosed fails the test unless ch is closed within 10 s.
func waitClosed(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(10 * time.Second):
		t.Fatalf("%s was not reached within 10s", what)
	}
}

// waitSpawn returns the result results delivers within 10 s, or fails the
// test.
func waitSpawn(t *testing.T, results <-chan SpawnResult, what string) SpawnResult {
	t.Helper()
	select {
	case r := <-results:
		return r
	case <-time.After(10 * time.Second):
		t.Fatalf("%s did not return within 10s", what)
		return SpawnResult{}
	}
}

// driverCalls returns copies of the herdr and agmsg call logs, read under
// their locks, for a test whose spawn still runs in another goroutine.
func driverCalls(h *fakeHerdr, a *fakeAgmsg) (herdr, agmsg []string) {
	h.mu.Lock()
	herdr = slices.Clone(h.calls)
	h.mu.Unlock()
	a.mu.Lock()
	agmsg = slices.Clone(a.calls)
	a.mu.Unlock()
	return herdr, agmsg
}

// TestOrgSpawn_SeatLock_RefusesASecondSpawnOfTheSeat covers plan AC9's
// deterministic case: while a first spawn of org-a/seat-1 is held in its
// seat's agmsg Join, holding the seat's spawn lock, a second spawn of the
// same seat is refused at once with an error saying another spawn runs:
// nothing is written to the manifest or the receipts, no herdr or agmsg call
// is made, and the first spawn's pane gets no C-c and no close. Once the
// Join is let go, the first spawn reaches spawned, and a retry after it
// returned answers idempotently.
func TestOrgSpawn_SeatLock_RefusesASecondSpawnOfTheSeat(t *testing.T) {
	o, h, a := testOrg(t)
	entered, release := holdJoin(a, "ralph-org-a", "seat-1")
	first := make(chan SpawnResult, 1)
	go func() { first <- o.Spawn(mustSpawnParams("org-a", "seat-1")) }()
	waitClosed(t, entered, "the first spawn's seat Join")

	lockPath := filepath.Join(filepath.Dir(o.Manifest.Path()), "spawn-org-a_seat-1.lock")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("expected the seat's lock file %s: %v", lockPath, err)
	}
	eventsBefore, receiptsBefore := len(mustReadEvents(t, o)), receiptCount(t, o)
	herdrBefore, agmsgBefore := driverCalls(h, a)

	second := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if second.Outcome != SpawnOutcomeRejected || !errors.Is(second.Err, errFileLockHeld) {
		t.Fatalf("expected the second spawn refused for the held lock, got %+v", second)
	}
	for _, want := range []string{`another ralph org spawn of seat "seat-1" in org_id "org-a" is running`, "retry after it returns"} {
		if !strings.Contains(second.Err.Error(), want) {
			t.Errorf("expected the error to contain %q, got %v", want, second.Err)
		}
	}
	if second.ModelReceipt != (Receipt{}) {
		t.Fatalf("expected no receipt on the refusal, got %+v", second.ModelReceipt)
	}
	if n := len(mustReadEvents(t, o)); n != eventsBefore {
		t.Fatalf("expected the refusal to write no event, events %d -> %d", eventsBefore, n)
	}
	if n := receiptCount(t, o); n != receiptsBefore {
		t.Fatalf("expected the refusal to write no receipt, receipts %d -> %d", receiptsBefore, n)
	}
	if herdrAfter, agmsgAfter := driverCalls(h, a); !slices.Equal(herdrAfter, herdrBefore) || !slices.Equal(agmsgAfter, agmsgBefore) {
		t.Fatalf("expected the refusal to call no driver, herdr %v -> %v, agmsg %v -> %v", herdrBefore, herdrAfter, agmsgBefore, agmsgAfter)
	}

	close(release)
	r := waitSpawn(t, first, "the first spawn")
	if r.Outcome != SpawnOutcomeSpawned || r.Seat.PaneID != "pane-1" {
		t.Fatalf("expected the first spawn to reach spawned on pane-1, got %+v", r)
	}
	if len(h.sendKeysCalls) != 0 || len(h.paneCloseCalls) != 0 {
		t.Fatalf("expected no C-c and no close of the first spawn's pane, got C-c %v closes %v", h.sendKeysCalls, h.paneCloseCalls)
	}
	if retry := o.Spawn(mustSpawnParams("org-a", "seat-1")); retry.Outcome != SpawnOutcomeIdempotent || retry.Seat.PaneID != "pane-1" {
		t.Fatalf("expected a retry after the first spawn returned to answer idempotently, got %+v", retry)
	}
}

// TestOrgSpawn_SeatLock_OtherSeatsOrgsAndDryRunsDoNotWait covers the rest of
// plan AC9: while a spawn of org-a/seat-1 holds its seat's lock, a spawn of
// another seat in org-a and one of the same seat id in org-b reach spawned,
// and a dry run of org-a/seat-1 itself takes no lock and goes on. The first
// spawn then reaches spawned too.
func TestOrgSpawn_SeatLock_OtherSeatsOrgsAndDryRunsDoNotWait(t *testing.T) {
	o, h, a := testOrg(t)
	entered, release := holdJoin(a, "ralph-org-a", "seat-1")
	first := make(chan SpawnResult, 1)
	go func() { first <- o.Spawn(mustSpawnParams("org-a", "seat-1")) }()
	waitClosed(t, entered, "the first spawn's seat Join")

	h.mu.Lock()
	h.paneID = "pane-2"
	h.mu.Unlock()
	if r := spawnWithin(t, o, mustSpawnParams("org-a", "seat-2"), 10*time.Second); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected org-a/seat-2 to spawn while seat-1's lock is held, got %+v", r)
	}
	h.mu.Lock()
	h.workspaceID, h.paneID = "ws-2", "pane-3"
	h.mu.Unlock()
	if r := spawnWithin(t, o, mustSpawnParams("org-b", "seat-1"), 10*time.Second); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected org-b/seat-1 to spawn while org-a/seat-1's lock is held, got %+v", r)
	}
	dry := mustSpawnParams("org-a", "seat-1")
	dry.DryRun = true
	if r := spawnWithin(t, o, dry, 10*time.Second); r.Outcome != SpawnOutcomeSpawned || !r.Seat.DryRun {
		t.Fatalf("expected a dry run of org-a/seat-1 to go on while its lock is held, got %+v", r)
	}

	close(release)
	if r := waitSpawn(t, first, "the first spawn"); r.Outcome != SpawnOutcomeSpawned || r.Seat.PaneID != "pane-1" {
		t.Fatalf("expected the first spawn to reach spawned on pane-1, got %+v", r)
	}
	if active := activeRealSeats(t, o); !slices.Equal(active, []OrgSeat{{"org-a", "seat-1"}, {"org-a", "seat-2"}, {"org-b", "seat-1"}}) {
		t.Fatalf("active seats = %v, want org-a/seat-1, org-a/seat-2 and org-b/seat-1", active)
	}
}

// TestOrgSpawn_SeatLock_LockThatCannotBeTaken_FailsWithNothingWritten: when
// the seat's lock file cannot be made (a regular file stands where the
// manifest directory goes), the spawn fails with that error, which is not
// the held-lock refusal, before it reads the manifest: no receipt and no
// driver call.
func TestOrgSpawn_SeatLock_LockThatCannotBeTaken_FailsWithNothingWritten(t *testing.T) {
	o, h, a := testOrg(t)
	blocker := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocker, []byte("a file"), 0o644); err != nil {
		t.Fatalf("seed the blocking file: %v", err)
	}
	o.Manifest = NewManifestStoreAtPath(filepath.Join(blocker, "org", "manifest.jsonl"))

	r := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if r.Outcome != SpawnOutcomeFailed || r.Err == nil || errors.Is(r.Err, errFileLockHeld) {
		t.Fatalf("expected SpawnOutcomeFailed that is not the held-lock refusal, got %+v", r)
	}
	if !strings.Contains(r.Err.Error(), "create spawn lock dir") {
		t.Fatalf("expected the error to say the spawn lock's directory could not be made, got %v", r.Err)
	}
	if len(h.calls) != 0 || len(a.calls) != 0 || receiptCount(t, o) != 0 {
		t.Fatalf("expected no driver call and no receipt, got herdr %v agmsg %v receipts %d", h.calls, a.calls, receiptCount(t, o))
	}
}
