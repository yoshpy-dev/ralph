package org

import (
	"os"
	"strings"
	"sync"
	"testing"
)

func newTestManifestStore(t *testing.T) *ManifestStore {
	t.Helper()
	dir := t.TempDir()
	return NewManifestStoreAtPath(ManifestPathIn(dir))
}

func TestManifestStore_AppendReadRoundTrip(t *testing.T) {
	store := newTestManifestStore(t)

	events := []ManifestEvent{
		{TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawnStarted, Driver: "claude", Model: "sonnet"},
		{TS: "2026-08-01T00:00:01Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawned, Driver: "claude", Model: "sonnet", PaneID: "pane-123"},
	}
	for _, ev := range events {
		if err := store.Append(ev); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	result, err := store.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if result.CorruptLines != 0 {
		t.Fatalf("expected 0 corrupt lines, got %d", result.CorruptLines)
	}
	if len(result.Events) != len(events) {
		t.Fatalf("expected %d events, got %d", len(events), len(result.Events))
	}
	for i, ev := range events {
		if result.Events[i] != ev {
			t.Errorf("event %d mismatch: want %+v, got %+v", i, ev, result.Events[i])
		}
	}
}

func TestManifestStore_Read_MissingFileIsEmpty(t *testing.T) {
	store := newTestManifestStore(t)
	result, err := store.Read()
	if err != nil {
		t.Fatalf("expected no error reading missing manifest, got: %v", err)
	}
	if len(result.Events) != 0 || result.CorruptLines != 0 {
		t.Fatalf("expected empty result for missing manifest, got: %+v", result)
	}
}

func TestManifestStore_Read_SkipsAndCountsCorruptLines(t *testing.T) {
	store := newTestManifestStore(t)

	good1 := ManifestEvent{TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawnStarted}
	good2 := ManifestEvent{TS: "2026-08-01T00:00:01Z", OrgID: "org-a", SeatID: "seat-2", Event: EventSpawnStarted}
	if err := store.Append(good1); err != nil {
		t.Fatalf("Append good1 failed: %v", err)
	}

	// Inject a corrupt line directly (not through Append, which always
	// writes valid JSON).
	f, err := os.OpenFile(store.Path(), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("failed to open manifest for corrupt-line injection: %v", err)
	}
	if _, err := f.WriteString("{not valid json\n"); err != nil {
		t.Fatalf("failed to write corrupt line: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close manifest after corrupt-line injection: %v", err)
	}

	if err := store.Append(good2); err != nil {
		t.Fatalf("Append good2 failed: %v", err)
	}

	result, err := store.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if result.CorruptLines != 1 {
		t.Fatalf("expected 1 corrupt line, got %d", result.CorruptLines)
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected 2 good events despite corrupt line, got %d", len(result.Events))
	}
}

func TestRoster_DryRunExclusionAndInclusion(t *testing.T) {
	events := []ManifestEvent{
		{TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawned, DryRun: false},
		{TS: "2026-08-01T00:00:01Z", OrgID: "org-a", SeatID: "seat-2", Event: EventSpawned, DryRun: true},
	}

	activeDefault := ActiveSeatCount(events, "org-a", RosterOptions{})
	if activeDefault != 1 {
		t.Fatalf("expected dry-run seat excluded by default, active count = 1, got %d", activeDefault)
	}

	activeWithDryRun := ActiveSeatCount(events, "org-a", RosterOptions{IncludeDryRun: true})
	if activeWithDryRun != 2 {
		t.Fatalf("expected dry-run seat included with IncludeDryRun, active count = 2, got %d", activeWithDryRun)
	}

	roster := Roster(events, RosterOptions{})
	if len(roster) != 1 {
		t.Fatalf("expected roster to exclude dry-run seat by default, got %d seats", len(roster))
	}

	rosterAll := Roster(events, RosterOptions{IncludeDryRun: true})
	if len(rosterAll) != 2 {
		t.Fatalf("expected roster --all to include dry-run seat, got %d seats", len(rosterAll))
	}
}

func TestRoster_SagaProgressionToStoppedIsInactive(t *testing.T) {
	events := []ManifestEvent{
		{TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawnStarted},
		{TS: "2026-08-01T00:00:01Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawned},
		{TS: "2026-08-01T00:00:02Z", OrgID: "org-a", SeatID: "seat-1", Event: EventStopped},
	}

	roster := Roster(events, RosterOptions{})
	if len(roster) != 1 {
		t.Fatalf("expected 1 seat in roster, got %d", len(roster))
	}
	if roster[0].Active {
		t.Fatalf("expected seat-1 to be inactive after stopped, got active status")
	}
	if roster[0].Event != EventStopped {
		t.Fatalf("expected latest event to be %q, got %q", EventStopped, roster[0].Event)
	}
}

func TestRoster_SpawnStartedAloneCountsAsActive(t *testing.T) {
	events := []ManifestEvent{
		{TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawnStarted},
	}

	count := ActiveSeatCount(events, "org-a", RosterOptions{})
	if count != 1 {
		t.Fatalf("expected stale spawn_started-only seat to count as active (visible in status), got %d", count)
	}
}

func TestRoster_UnknownEventTypeIgnoredForActivity(t *testing.T) {
	// A non-state event (unknown, or a known-but-non-state type such as
	// `sent`) must not overwrite or clear a seat's derived status -- only
	// STATE events (see seat.go stateEvents) participate in "latest
	// applicable event" derivation. Here the seat's real state is `spawned`
	// (active); a later unknown event must leave that untouched.
	events := []ManifestEvent{
		{TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawned},
		{TS: "2026-08-01T00:00:01Z", OrgID: "org-a", SeatID: "seat-1", Event: "some_future_event_v2"},
	}

	roster := Roster(events, RosterOptions{})
	if len(roster) != 1 {
		t.Fatalf("expected roster derivation to tolerate unknown event type, got %d seats", len(roster))
	}
	if !roster[0].Active {
		t.Fatalf("expected seat activity to remain unaffected by a non-state event, got inactive")
	}
	if roster[0].Event != EventSpawned {
		t.Fatalf("expected latest STATE event to remain %q, got %q", EventSpawned, roster[0].Event)
	}
}

func TestRoster_UnknownEventTypeAloneProducesNoSeat(t *testing.T) {
	// If a seat has never had a STATE event, a non-state event alone must
	// not create a roster entry for it -- ignored for activity means
	// ignored entirely from the "latest applicable event" pointer.
	events := []ManifestEvent{
		{TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: "some_future_event_v2"},
	}

	roster := Roster(events, RosterOptions{})
	if len(roster) != 0 {
		t.Fatalf("expected no roster entry for a seat with only non-state events, got %d", len(roster))
	}
}

func TestRoster_DisbandedOrgLevelEventDeactivatesAllSeats(t *testing.T) {
	events := []ManifestEvent{
		{TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawned},
		{TS: "2026-08-01T00:00:01Z", OrgID: "org-a", SeatID: "seat-2", Event: EventSpawned},
		{TS: "2026-08-01T00:00:02Z", OrgID: "org-a", SeatID: "", Event: EventDisbanded},
	}

	roster := Roster(events, RosterOptions{})
	for _, s := range roster {
		if s.Active {
			t.Errorf("expected seat %s to be inactive after org-level disbanded, got active", s.SeatID)
		}
	}
}

func TestRoster_DryRunDisbandLeavesRealSeatsActive(t *testing.T) {
	events := []ManifestEvent{
		{TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawned},
		{TS: "2026-08-01T00:00:01Z", OrgID: "org-a", SeatID: "", Event: EventDisbanded, DryRun: true},
	}

	roster := Roster(events, RosterOptions{IncludeDryRun: true})
	found := false
	for _, s := range roster {
		if s.SeatID == "seat-1" {
			found = true
			if !s.Active {
				t.Errorf("expected real seat-1 to remain active after a dry-run disbanded event, got inactive: %+v", s)
			}
		}
	}
	if !found {
		t.Fatalf("expected seat-1 present in roster, got %+v", roster)
	}
}

func TestRoster_RealDisbandLeavesDryRunSeatsUntouched(t *testing.T) {
	events := []ManifestEvent{
		{TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawned, DryRun: true},
		{TS: "2026-08-01T00:00:01Z", OrgID: "org-a", SeatID: "", Event: EventDisbanded},
	}

	roster := Roster(events, RosterOptions{IncludeDryRun: true})
	found := false
	for _, s := range roster {
		if s.SeatID == "seat-1" && s.DryRun {
			found = true
			if !s.Active {
				t.Errorf("expected dry-run seat-1 to remain active after a real disbanded event, got inactive: %+v", s)
			}
		}
	}
	if !found {
		t.Fatalf("expected dry-run seat-1 present in roster, got %+v", roster)
	}
}

func TestRoster_DryRunStoppedDoesNotDeactivateRealSeatWithSameID(t *testing.T) {
	events := []ManifestEvent{
		{TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawned},
		{TS: "2026-08-01T00:00:01Z", OrgID: "org-a", SeatID: "seat-1", Event: EventStopped, DryRun: true},
	}

	roster := Roster(events, RosterOptions{IncludeDryRun: true})
	var real, dryRun *SeatStatus
	for i := range roster {
		s := &roster[i]
		if s.SeatID != "seat-1" {
			continue
		}
		if s.DryRun {
			dryRun = s
		} else {
			real = s
		}
	}
	if real == nil {
		t.Fatalf("expected a real seat-1 entry in the roster, got %+v", roster)
	}
	if !real.Active || real.Event != EventSpawned {
		t.Errorf("expected the real seat-1 to remain active with event %q untouched by a dry-run stopped event, got %+v", EventSpawned, real)
	}
	if dryRun == nil {
		t.Fatalf("expected a distinct dry-run seat-1 entry in the roster, got %+v", roster)
	}
	if dryRun.Active || dryRun.Event != EventStopped {
		t.Errorf("expected the dry-run seat-1 entry to be inactive with event %q, got %+v", EventStopped, dryRun)
	}
}

func TestManifestStore_ConcurrentAppendsAllLinesIntact(t *testing.T) {
	store := newTestManifestStore(t)

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	errCh := make(chan error, n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			ev := ManifestEvent{
				TS:      "2026-08-01T00:00:00Z",
				OrgID:   "org-a",
				SeatID:  "seat-concurrent",
				Event:   EventSpawnStarted,
				Details: repeatDigit(i),
			}
			if err := store.Append(ev); err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent Append failed: %v", err)
	}

	result, err := store.Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if result.CorruptLines != 0 {
		t.Fatalf("expected 0 corrupt lines from concurrent appends, got %d", result.CorruptLines)
	}
	if len(result.Events) != n {
		t.Fatalf("expected %d intact events from concurrent appends, got %d", n, len(result.Events))
	}
}

// repeatDigit gives each concurrent goroutine's event distinguishable
// content so a corrupted/interleaved write is detectable via failed
// unmarshal, not just a wrong count.
func repeatDigit(i int) string {
	digits := "0123456789"
	d := digits[i%10]
	out := make([]byte, 32)
	for j := range out {
		out[j] = d
	}
	return string(out)
}

// TestManifestEvent_HerdrAgentName_JSONRoundTrip pins the wire shape (AC-8):
// the field must serialize under the "herdr_agent_name" JSON key when set,
// and be entirely absent from the JSON line when empty (omitempty) -- the
// latter is what makes a pre-AC-8 manifest line decode with an empty
// HerdrAgentName instead of an unmarshal error.
func TestManifestEvent_HerdrAgentName_JSONRoundTrip(t *testing.T) {
	store := newTestManifestStore(t)

	withName := ManifestEvent{TS: "2026-08-02T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: EventSpawned, HerdrAgentName: "org-a_seat-1"}
	withoutName := ManifestEvent{TS: "2026-08-02T00:00:01Z", OrgID: "org-a", SeatID: "seat-2", Event: EventSpawned}
	for _, ev := range []ManifestEvent{withName, withoutName} {
		if err := store.Append(ev); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	raw, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatalf("read raw manifest: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 raw JSONL lines, got %d:\n%s", len(lines), raw)
	}
	if !strings.Contains(lines[0], `"herdr_agent_name":"org-a_seat-1"`) {
		t.Errorf("expected the first event's raw JSON line to contain the herdr_agent_name key, got:\n%s", lines[0])
	}
	if strings.Contains(lines[1], "herdr_agent_name") {
		t.Errorf("expected the second (unset) event's raw JSON line to omit herdr_agent_name entirely, got:\n%s", lines[1])
	}

	result, err := store.Read()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(result.Events))
	}
	if result.Events[0].HerdrAgentName != "org-a_seat-1" {
		t.Errorf("expected the first event's HerdrAgentName to round-trip, got %q", result.Events[0].HerdrAgentName)
	}
	if result.Events[1].HerdrAgentName != "" {
		t.Errorf("expected the second event's HerdrAgentName to decode as empty, got %q", result.Events[1].HerdrAgentName)
	}

	roster := Roster(result.Events, RosterOptions{})
	if len(roster) != 2 {
		t.Fatalf("expected 2 roster entries, got %+v", roster)
	}
	for _, s := range roster {
		want := ""
		if s.SeatID == "seat-1" {
			want = "org-a_seat-1"
		}
		if s.HerdrAgentName != want {
			t.Errorf("roster entry %s: HerdrAgentName = %q, want %q", s.SeatID, s.HerdrAgentName, want)
		}
	}
}
