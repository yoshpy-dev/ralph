package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// runStatusCmd runs `ralph status <args...>` in-process and returns combined
// stdout/stderr plus any error from Execute().
func runStatusCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(append([]string{"status"}, args...))
	err := root.Execute()
	return buf.String(), err
}

// seedTwoOrgManifest appends a small fixture manifest directly (no
// herdr/agmsg involved -- `ralph status` is manifest-only, per org.Status's
// own doc comment): org-a has two seats (leader active, reviewer stopped),
// org-b has one seat (qa active).
func seedTwoOrgManifest(t *testing.T, stateDir string) {
	t.Helper()
	// stateDir here plays the role of an already-resolved org state
	// directory (what org.ResolveOrgStateDir returns and runStatus
	// receives), so the fixture is written to the same path runStatus
	// itself reads (org.ManifestPathIn) -- not a root-relative derivation.
	store := org.NewManifestStoreAtPath(org.ManifestPathIn(stateDir))
	events := []org.ManifestEvent{
		{TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "leader", Event: org.EventSpawned, Role: "leader", Driver: "claude", Model: "opus", Worktree: "/tmp/org-a-leader"},
		{TS: "2026-08-01T00:01:00Z", OrgID: "org-a", SeatID: "reviewer", Event: org.EventSpawned, Role: "reviewer", Driver: "codex", Model: "sonnet", Worktree: "/tmp/org-a-reviewer"},
		{TS: "2026-08-01T00:02:00Z", OrgID: "org-a", SeatID: "reviewer", Event: org.EventStopped, Role: "reviewer", Driver: "codex", Model: "sonnet", Worktree: "/tmp/org-a-reviewer"},
		{TS: "2026-08-01T00:00:00Z", OrgID: "org-b", SeatID: "qa", Event: org.EventSpawned, Role: "qa", Driver: "claude", Model: "sonnet", Worktree: "/tmp/org-b-qa"},
	}
	for _, ev := range events {
		if err := store.Append(ev); err != nil {
			t.Fatalf("seed manifest event: %v", err)
		}
	}
}

func TestStatusCmd_ListsAllOrgsWithRosterAndActiveCounts(t *testing.T) {
	dir := t.TempDir()
	seedTwoOrgManifest(t, dir)

	out, err := runStatusCmd(t, "--state-dir", dir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}

	for _, want := range []string{
		"org_id: org-a", "org_id: org-b",
		"leader", "reviewer", "qa",
		"active 1/2", // org-a: leader active, reviewer stopped
		"active 1/1", // org-b: qa active
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestStatusCmd_OrgIDFilterShowsOnlyThatOrg(t *testing.T) {
	dir := t.TempDir()
	seedTwoOrgManifest(t, dir)

	out, err := runStatusCmd(t, "--state-dir", dir, "--org-id", "org-a")
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}

	if !strings.Contains(out, "org_id: org-a") {
		t.Errorf("expected org-a in output, got:\n%s", out)
	}
	if strings.Contains(out, "org_id: org-b") {
		t.Errorf("--org-id filter leaked org-b into output:\n%s", out)
	}
	if strings.Contains(out, "qa") {
		t.Errorf("--org-id filter leaked org-b's seat into output:\n%s", out)
	}
}

// seedOrgWithDryRunSeat appends one real active seat ("leader") and one
// dry-run active seat ("shadow") to the same org_id, directly to the
// manifest (no herdr/agmsg involved). Used to assert that a dry-run seat
// shows up as a roster row (IncludeDryRun: true, internal/cli/status.go)
// while never inflating the active/total aggregates that gate `ralph org
// spawn`'s max_seats check (C2-1: those aggregates must derive from
// org.RosterOptions{}, not from counting the dry-run-inclusive rows).
func seedOrgWithDryRunSeat(t *testing.T, stateDir string) {
	t.Helper()
	store := org.NewManifestStoreAtPath(org.ManifestPathIn(stateDir))
	events := []org.ManifestEvent{
		{TS: "2026-08-01T00:00:00Z", OrgID: "org-c", SeatID: "leader", Event: org.EventSpawned, Role: "leader", Driver: "claude", Model: "opus", Worktree: "/tmp/org-c-leader"},
		{TS: "2026-08-01T00:01:00Z", OrgID: "org-c", SeatID: "shadow", Event: org.EventSpawned, Role: "qa", Driver: "codex", Model: "sonnet", Worktree: "/tmp/org-c-shadow", DryRun: true},
	}
	for _, ev := range events {
		if err := store.Append(ev); err != nil {
			t.Fatalf("seed manifest event: %v", err)
		}
	}
}

// TestStatusCmd_DryRunSeatIsARowButNotCountedInAggregates is the C2-1
// regression: a dry-run seat must render as a roster row (with its
// "[dry-run]" marker) but must not move the table's "active N/M" or the
// --json path's active_count/total_count, since those numbers must match
// what org.ActiveSeatCount (RosterOptions{}) reports -- the same
// derivation the max_seats spawn gate uses (internal/org/spawn.go:333,788,
// internal/org/report.go:201).
func TestStatusCmd_DryRunSeatIsARowButNotCountedInAggregates(t *testing.T) {
	dir := t.TempDir()
	seedOrgWithDryRunSeat(t, dir)

	out, err := runStatusCmd(t, "--state-dir", dir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "active 1/1") {
		t.Errorf("expected the dry-run seat excluded from the aggregate (want \"active 1/1\"), got:\n%s", out)
	}
	if strings.Contains(out, "active 2/2") {
		t.Errorf("dry-run seat leaked into the aggregate count:\n%s", out)
	}
	if !strings.Contains(out, "shadow") || !strings.Contains(out, "[dry-run]") {
		t.Errorf("expected the dry-run seat to still render as a row with its [dry-run] marker:\n%s", out)
	}

	jsonOut, err := runStatusCmd(t, "--state-dir", dir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
	}
	var payload struct {
		Orgs []struct {
			OrgID       string `json:"org_id"`
			ActiveCount int    `json:"active_count"`
			TotalCount  int    `json:"total_count"`
			Seats       []struct {
				SeatID string `json:"seat_id"`
				DryRun bool   `json:"dry_run"`
			} `json:"seats"`
		} `json:"orgs"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v\noutput:\n%s", err, jsonOut)
	}
	if len(payload.Orgs) != 1 {
		t.Fatalf("orgs = %d, want 1\noutput:\n%s", len(payload.Orgs), jsonOut)
	}
	orgC := payload.Orgs[0]
	if orgC.ActiveCount != 1 {
		t.Errorf("org-c active_count = %d, want 1 (dry-run seat must not count)", orgC.ActiveCount)
	}
	if orgC.TotalCount != 1 {
		t.Errorf("org-c total_count = %d, want 1 (dry-run seat must not count)", orgC.TotalCount)
	}
	if len(orgC.Seats) != 2 {
		t.Fatalf("org-c seats = %d, want 2 (both real and dry-run rows must still render)", len(orgC.Seats))
	}
	var sawDryRun bool
	for _, s := range orgC.Seats {
		if s.SeatID == "shadow" && s.DryRun {
			sawDryRun = true
		}
	}
	if !sawDryRun {
		t.Errorf("expected the shadow seat to render with dry_run=true, got: %+v", orgC.Seats)
	}
}

// TestStatusCmd_SeesSeatWrittenByRealOrgSpawn pins live-write/status-read
// agreement end to end: a real `ralph org spawn --dry-run` (the write path,
// newOrgRuntimeAt -> org.ManifestPathIn) followed by a real top-level `ralph
// status` (the read path, runStatus -> org.ManifestPathIn) against the same
// --state-dir. This is the AR-1 regression case
// (docs/reports/cross-review-triage-org-runtime-retire-loop.md): before the
// fix, `ralph status` used the package's then-exported root-relative
// constructor, which
// re-appended a package-level root-relative fragment onto an
// already-resolved directory and always reported "no org runtime state
// found" for a manifest a real spawn had just written. That root-relative
// constructor and its relative-path constant were later removed entirely
// (C2-3, docs/tech-debt/README.md) in favor of org.ManifestPathIn as the
// single derivation. Deliberately does not touch a shared fixture helper
// directly (unlike seedTwoOrgManifest) so a future refactor that makes only
// one of the two call sites use org.ManifestPathIn is caught by this test
// even if a hand-seeded fixture would not catch it.
//
// Uses --dry-run (no herdr/agmsg on PATH needed) and asserts the seat shows
// up in the top-level `ralph status` output with no --all flag: unlike
// `ralph org status` (which gates dry-run visibility behind --all), this
// command has no --all flag and always includes dry-run seats (see
// runStatus's RosterOptions{IncludeDryRun: true}, status.go) -- a second,
// related gap found while confirming this exact repro against the AR-1 fix.
func TestStatusCmd_SeesSeatWrittenByRealOrgSpawn(t *testing.T) {
	t.Setenv("PATH", "") // dry-run spawn needs no herdr/agmsg on PATH
	stateDir := filepath.Join(t.TempDir(), "state")

	spawnOut, err := runOrgCmd(t,
		"spawn", "--org-id", "demo", "--id", "leader", "--role", "leader",
		"--driver", "claude", "--model", "opus", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--dry-run",
	)
	if err != nil {
		t.Fatalf("org spawn --dry-run: %v (output: %s)", err, spawnOut)
	}

	out, err := runStatusCmd(t, "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	if strings.Contains(out, "no org runtime state found") {
		t.Fatalf("ralph status did not see the seat written by a real `ralph org spawn` at the same --state-dir:\n%s", out)
	}
	if !strings.Contains(out, "org_id: demo") || !strings.Contains(out, "leader") {
		t.Errorf("expected demo org's leader seat in status output, got:\n%s", out)
	}
}

// TestStatusCmd_ShowsStateDirSource pins the tech-debt "watchdog deferred
// LOW (1)" fix on the `ralph status` side: org.ResolveOrgStateDir's second
// return value used to be discarded (`_`) at this call site too. Both the
// text table header (populated roster) and the --json state_dir_source
// field must carry the resolved precedence tier -- "flag" here, since
// --state-dir is passed explicitly.
func TestStatusCmd_ShowsStateDirSource(t *testing.T) {
	dir := t.TempDir()
	seedTwoOrgManifest(t, dir)

	out, err := runStatusCmd(t, "--state-dir", dir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "(source: flag)") {
		t.Errorf("expected the text table to show the state-dir source, got:\n%s", out)
	}

	jsonOut, err := runStatusCmd(t, "--state-dir", dir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
	}
	var payload struct {
		StateDirSource string `json:"state_dir_source"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v\noutput:\n%s", err, jsonOut)
	}
	if payload.StateDirSource != "flag" {
		t.Errorf("expected state_dir_source=%q, got %q\noutput:\n%s", "flag", payload.StateDirSource, jsonOut)
	}
}

// TestStatusCmd_EmptyStateDirShowsSourceToo is the same state_dir_source
// assertion against the empty-roster path (printStatusEmpty), which folds
// the annotation into its existing message rather than a separate header
// line (see printStatusEmpty's doc comment).
func TestStatusCmd_EmptyStateDirShowsSourceToo(t *testing.T) {
	dir := t.TempDir()

	out, err := runStatusCmd(t, "--state-dir", dir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "(source: flag)") {
		t.Errorf("expected the empty-roster message to show the state-dir source, got:\n%s", out)
	}

	jsonOut, err := runStatusCmd(t, "--state-dir", dir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
	}
	var payload struct {
		StateDirSource string `json:"state_dir_source"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v\noutput:\n%s", err, jsonOut)
	}
	if payload.StateDirSource != "flag" {
		t.Errorf("expected state_dir_source=%q on the empty-roster JSON path too, got %q\noutput:\n%s", "flag", payload.StateDirSource, jsonOut)
	}
}

func TestStatusCmd_EmptyStateDirShowsFriendlyNoteAndDoctorHint(t *testing.T) {
	dir := t.TempDir() // no manifest ever written here

	out, err := runStatusCmd(t, "--state-dir", dir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}

	if !strings.Contains(out, "no org runtime state found") {
		t.Errorf("expected empty-state note in English (matching the rest of the CLI's output), got:\n%s", out)
	}
	if !strings.Contains(out, "ralph org spawn") {
		t.Errorf("expected a hint to run `ralph org spawn`, got:\n%s", out)
	}
	if !strings.Contains(out, "ralph doctor") {
		t.Errorf("expected a doctor hint, got:\n%s", out)
	}
}

// TestStatusCmd_EmptyRosterFromFullyCorruptManifestStillWarns seeds a
// manifest containing only corrupt lines (so Roster derives zero seats,
// exercising the same len(groups)==0 path as the "nothing has ever run"
// case) and asserts the corrupt-line warning still surfaces -- a
// fully-corrupt manifest is a data-integrity signal, not "nothing here yet"
// (M5: printStatusEmpty must not silently discard rr.CorruptLines).
func TestStatusCmd_EmptyRosterFromFullyCorruptManifestStillWarns(t *testing.T) {
	dir := t.TempDir()
	store := org.NewManifestStoreAtPath(org.ManifestPathIn(dir))
	if err := os.MkdirAll(filepath.Dir(store.Path()), 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	if err := os.WriteFile(store.Path(), []byte("{not valid json\nalso not valid\n"), 0o644); err != nil {
		t.Fatalf("seed fully-corrupt manifest: %v", err)
	}

	out, err := runStatusCmd(t, "--state-dir", dir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}

	if !strings.Contains(out, "no org runtime state found") {
		t.Errorf("expected the empty-state note, got:\n%s", out)
	}
	if !strings.Contains(out, "2 corrupt manifest line") {
		t.Errorf("expected a corrupt-line warning even with zero roster seats, got:\n%s", out)
	}
}

// TestStatusCmd_JSONSchemaIsIdenticalEmptyVsPopulated asserts `ralph status
// --json` never requires a consumer to branch on which shape it received:
// the empty-roster path must carry the same `orgs` (empty array, not
// omitted) and `corrupt_lines` (present when nonzero) fields the populated
// path does (M6: unify the --json schema across both paths).
func TestStatusCmd_JSONSchemaIsIdenticalEmptyVsPopulated(t *testing.T) {
	type payload struct {
		StateDir     string `json:"state_dir"`
		OrgID        string `json:"org_id,omitempty"`
		Orgs         []any  `json:"orgs"`
		CorruptLines int    `json:"corrupt_lines,omitempty"`
	}

	emptyDir := t.TempDir()
	emptyOut, err := runStatusCmd(t, "--state-dir", emptyDir, "--json")
	if err != nil {
		t.Fatalf("status (empty): %v (output: %s)", err, emptyOut)
	}
	var emptyPayload payload
	if err := json.Unmarshal([]byte(emptyOut), &emptyPayload); err != nil {
		t.Fatalf("json.Unmarshal (empty): %v\noutput:\n%s", err, emptyOut)
	}
	if emptyPayload.Orgs == nil {
		t.Errorf("empty-path JSON must carry `orgs: []`, not an omitted/null field:\n%s", emptyOut)
	}

	populatedDir := t.TempDir()
	seedTwoOrgManifest(t, populatedDir)
	populatedOut, err := runStatusCmd(t, "--state-dir", populatedDir, "--json")
	if err != nil {
		t.Fatalf("status (populated): %v (output: %s)", err, populatedOut)
	}
	var populatedPayload payload
	if err := json.Unmarshal([]byte(populatedOut), &populatedPayload); err != nil {
		t.Fatalf("json.Unmarshal (populated): %v\noutput:\n%s", err, populatedOut)
	}
	if len(populatedPayload.Orgs) != 2 {
		t.Fatalf("populated-path orgs = %d, want 2:\n%s", len(populatedPayload.Orgs), populatedOut)
	}

	// Both payloads decode into the same schema with no divergent fields
	// (verified above by sharing the `payload` struct for both Unmarshal
	// calls); additionally confirm state_dir is present in both.
	if emptyPayload.StateDir == "" {
		t.Errorf("empty-path JSON missing state_dir:\n%s", emptyOut)
	}
	if populatedPayload.StateDir == "" {
		t.Errorf("populated-path JSON missing state_dir:\n%s", populatedOut)
	}
}

func TestStatusCmd_JSONOutputIsValidAndGroupsByOrg(t *testing.T) {
	dir := t.TempDir()
	seedTwoOrgManifest(t, dir)

	out, err := runStatusCmd(t, "--state-dir", dir, "--json")
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}

	var payload struct {
		Orgs []struct {
			OrgID       string `json:"org_id"`
			ActiveCount int    `json:"active_count"`
			TotalCount  int    `json:"total_count"`
			Seats       []struct {
				SeatID string `json:"seat_id"`
			} `json:"seats"`
		} `json:"orgs"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v\noutput:\n%s", err, out)
	}
	if len(payload.Orgs) != 2 {
		t.Fatalf("orgs = %d, want 2\noutput:\n%s", len(payload.Orgs), out)
	}
	byID := make(map[string]int)
	for _, o := range payload.Orgs {
		byID[o.OrgID] = o.ActiveCount
	}
	if byID["org-a"] != 1 {
		t.Errorf("org-a active_count = %d, want 1", byID["org-a"])
	}
	if byID["org-b"] != 1 {
		t.Errorf("org-b active_count = %d, want 1", byID["org-b"])
	}
}

// TestStatusCmd_ShowsWatchHeartbeatAndPendingAlertCount seeds a
// watch-status-<org_id>.json file matching the shape `ralph org watch`
// persists (org.WatchStatusFileName) and asserts `ralph status` surfaces the
// last heartbeat cycle and pending-alert count for that org.
func TestStatusCmd_ShowsWatchHeartbeatAndPendingAlertCount(t *testing.T) {
	dir := t.TempDir()
	seedTwoOrgManifest(t, dir)

	watchStatus := map[string]any{
		"org_id":        "org-a",
		"last_cycle_ts": "2026-08-01T00:05:00Z",
		"cycles":        7,
		"pending_alerts": map[string]any{
			"org-a//deadman": map[string]any{"alert_id": "a1", "ts": "2026-08-01T00:05:00Z", "reason": "test"},
		},
	}
	data, err := json.Marshal(watchStatus)
	if err != nil {
		t.Fatalf("marshal watch status fixture: %v", err)
	}
	watchPath := filepath.Join(dir, org.WatchStatusFileName("org-a"))
	if err := os.WriteFile(watchPath, data, 0o644); err != nil {
		t.Fatalf("write watch status fixture: %v", err)
	}

	out, err := runStatusCmd(t, "--state-dir", dir, "--org-id", "org-a")
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}

	if !strings.Contains(out, "last heartbeat 2026-08-01T00:05:00Z") {
		t.Errorf("expected last heartbeat in output, got:\n%s", out)
	}
	if !strings.Contains(out, "cycle 7") {
		t.Errorf("expected cycle count in output, got:\n%s", out)
	}
	if !strings.Contains(out, "pending alerts: 1") {
		t.Errorf("expected pending alert count in output, got:\n%s", out)
	}
}

// Tests for the `org watch:` line and the org_watch object of `ralph status`
// (plan 2026-10-11-org-cross-watch, AC14). holdWatchLock and runningMark
// come from org_watch_test.go.

// writeWatchMark writes mark as the liveness mark in stateDir without taking
// watch.lock, as a watch that has ended leaves it.
func writeWatchMark(t *testing.T, stateDir string, mark *org.WatchAlive) {
	t.Helper()
	data, err := json.Marshal(mark)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(org.WatchAlivePathIn(stateDir), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// statusOrgWatchLine returns the one `org watch:` line of `ralph status`
// text output, failing the test when there is not exactly one.
func statusOrgWatchLine(t *testing.T, out string) string {
	t.Helper()
	var found []string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "org watch:") {
			found = append(found, line)
		}
	}
	if len(found) != 1 {
		t.Fatalf("want exactly one `org watch:` line, got %d:\n%s", len(found), out)
	}
	return found[0]
}

// statusOrgWatchJSONOf decodes the org_watch object of `ralph status --json`
// output into a map, so a key left out differs from a zero value.
func statusOrgWatchJSONOf(t *testing.T, out string) map[string]any {
	t.Helper()
	var payload struct {
		OrgWatch map[string]any `json:"org_watch"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v\noutput:\n%s", err, out)
	}
	if payload.OrgWatch == nil {
		t.Fatalf("--json carries no org_watch object:\n%s", out)
	}
	return payload.OrgWatch
}

// TestStatusCmd_OrgWatchLineAndJSONPerWatchState runs `ralph status` and
// `ralph status --json` on a populated roster for each state the cross-org
// watch can be in, by watch.lock (held by this test or not) and the
// liveness mark, and pins the whole `org watch:` line and org_watch object.
func TestStatusCmd_OrgWatchLineAndJSONPerWatchState(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	fresh := runningMark(4244, now)
	old := runningMark(4245, now.Add(-time.Hour))
	killed := runningMark(4243, time.Date(2026, 10, 11, 4, 40, 0, 0, time.UTC))
	stopped := &org.WatchAlive{
		State: org.WatchAliveStopped, PID: 4242,
		StartedAt: "2026-10-11T04:00:00.123456789Z", TS: "2026-10-11T04:40:00Z",
		IntervalSeconds: 30, StoppedAt: "2026-10-11T04:40:05Z",
	}
	// markJSON is the org_watch object for state with every field of m.
	markJSON := func(state string, m *org.WatchAlive) map[string]any {
		want := map[string]any{
			"state": state, "pid": float64(m.PID), "started_at": m.StartedAt,
			"last_cycle_ts": m.TS, "mark_state": m.State,
		}
		if m.StoppedAt != "" {
			want["stopped_at"] = m.StoppedAt
		}
		return want
	}
	const staleTail = " -- a watch holds watch.lock, but its liveness mark is missing, stopped, or older than deadman_minutes"

	cases := []struct {
		name     string
		setup    func(t *testing.T, stateDir string)
		wantLine string
		wantJSON map[string]any
	}{
		{
			name:     "not running with no mark",
			setup:    func(*testing.T, string) {},
			wantLine: "org watch: not running",
			wantJSON: map[string]any{"state": "not_running"},
		},
		{
			name:     "not running with a stopped mark",
			setup:    func(t *testing.T, stateDir string) { writeWatchMark(t, stateDir, stopped) },
			wantLine: "org watch: not running (stopped at 2026-10-11T04:40:05Z)",
			wantJSON: markJSON("not_running", stopped),
		},
		{
			name:     "not running with the running mark of a killed watch",
			setup:    func(t *testing.T, stateDir string) { writeWatchMark(t, stateDir, killed) },
			wantLine: "org watch: not running (a running mark is left from pid 4243, last cycle 2026-10-11T04:40:00Z)",
			wantJSON: markJSON("not_running", killed),
		},
		{
			name:     "running",
			setup:    func(t *testing.T, stateDir string) { holdWatchLock(t, stateDir, fresh) },
			wantLine: "org watch: running (pid 4244, last cycle " + fresh.TS + ")",
			wantJSON: markJSON("running", fresh),
		},
		{
			name:     "stale with an old mark",
			setup:    func(t *testing.T, stateDir string) { holdWatchLock(t, stateDir, old) },
			wantLine: "org watch: stale (pid 4245, last cycle " + old.TS + ")" + staleTail,
			wantJSON: markJSON("stale", old),
		},
		{
			name:     "stale with a stopped mark",
			setup:    func(t *testing.T, stateDir string) { holdWatchLock(t, stateDir, stopped) },
			wantLine: "org watch: stale (pid 4242, last cycle 2026-10-11T04:40:00Z, stopped at 2026-10-11T04:40:05Z)" + staleTail,
			wantJSON: markJSON("stale", stopped),
		},
		{
			name:     "stale with no mark",
			setup:    func(t *testing.T, stateDir string) { holdWatchLock(t, stateDir, nil) },
			wantLine: "org watch: stale" + staleTail,
			wantJSON: map[string]any{"state": "stale"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			seedTwoOrgManifest(t, dir)
			tc.setup(t, dir)

			out, err := runStatusCmd(t, "--state-dir", dir)
			if err != nil {
				t.Fatalf("status: %v (output: %s)", err, out)
			}
			if got := statusOrgWatchLine(t, out); got != tc.wantLine {
				t.Errorf("org watch line:\n got %q\nwant %q", got, tc.wantLine)
			}

			jsonOut, err := runStatusCmd(t, "--state-dir", dir, "--json")
			if err != nil {
				t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
			}
			if got := statusOrgWatchJSONOf(t, jsonOut); !reflect.DeepEqual(got, tc.wantJSON) {
				t.Errorf("org_watch:\n got %v\nwant %v", got, tc.wantJSON)
			}
		})
	}
}

// TestStatusCmd_OrgWatchLineFollowsTheStateDirLine pins where the line goes
// on the populated path: right below the state-dir line, above the rosters,
// whose per-org watch heartbeat stays as it was.
func TestStatusCmd_OrgWatchLineFollowsTheStateDirLine(t *testing.T) {
	dir := t.TempDir()
	seedTwoOrgManifest(t, dir)

	out, err := runStatusCmd(t, "--state-dir", dir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	lines := strings.Split(out, "\n")
	if len(lines) < 3 || !strings.HasPrefix(lines[0], "state-dir: ") || lines[1] != "org watch: not running" || !strings.HasPrefix(lines[2], "org_id: ") {
		t.Errorf("want the state-dir line, then the org watch line, then the first roster, got:\n%s", out)
	}
}

// TestStatusCmd_EmptyRosterStillShowsTheOrgWatch: a watch can run before
// any org starts, so the empty-roster path carries the line and org_watch
// too.
func TestStatusCmd_EmptyRosterStillShowsTheOrgWatch(t *testing.T) {
	dir := t.TempDir() // no manifest ever written here
	mark := runningMark(4246, time.Now())
	holdWatchLock(t, dir, mark)

	out, err := runStatusCmd(t, "--state-dir", dir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "no org runtime state found") {
		t.Errorf("expected the empty-state note, got:\n%s", out)
	}
	if got, want := statusOrgWatchLine(t, out), "org watch: running (pid 4246, last cycle "+mark.TS+")"; got != want {
		t.Errorf("org watch line on the empty roster:\n got %q\nwant %q", got, want)
	}

	jsonOut, err := runStatusCmd(t, "--state-dir", dir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
	}
	got := statusOrgWatchJSONOf(t, jsonOut)
	if got["state"] != "running" || got["pid"] != float64(4246) || got["last_cycle_ts"] != mark.TS {
		t.Errorf("org_watch on the empty roster = %v, want state running, pid 4246, last_cycle_ts %s", got, mark.TS)
	}
}

// TestStatusCmd_OrgWatchJudgesTheMarkByDeadmanMinutesOfRalphToml: status has
// no --config, so it reads deadman_minutes from ./ralph.toml as the ralph org
// verbs do without --config. A 30-minute-old mark is stale under the
// default 10 and running under 60.
func TestStatusCmd_OrgWatchJudgesTheMarkByDeadmanMinutesOfRalphToml(t *testing.T) {
	stateDir := t.TempDir()
	mark := runningMark(4247, time.Now().Add(-30*time.Minute))
	holdWatchLock(t, stateDir, mark)
	t.Chdir(t.TempDir())

	out, err := runStatusCmd(t, "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status without ralph.toml: %v (output: %s)", err, out)
	}
	if got := statusOrgWatchLine(t, out); !strings.HasPrefix(got, "org watch: stale (pid 4247, ") {
		t.Errorf("without ralph.toml (deadman_minutes 10), a 30-minute-old mark should be stale, got %q", got)
	}

	if err := os.WriteFile("ralph.toml", []byte("[org]\ndeadman_minutes = 60\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = runStatusCmd(t, "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status with ralph.toml: %v (output: %s)", err, out)
	}
	if got, want := statusOrgWatchLine(t, out), "org watch: running (pid 4247, last cycle "+mark.TS+")"; got != want {
		t.Errorf("with deadman_minutes = 60:\n got %q\nwant %q", got, want)
	}
}

// TestStatusCmd_OrgWatchWithDeadmanMinutesZeroJudgesByTheDefault (self-review
// M1): deadman_minutes = 0 in ./ralph.toml means the deadman never escalates
// by timeout, so status judges the mark by the default 10 minutes: a fresh
// mark is running (it used to be stale under any deadman below 1), and a
// 30-minute-old one is stale.
func TestStatusCmd_OrgWatchWithDeadmanMinutesZeroJudgesByTheDefault(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.WriteFile("ralph.toml", []byte("[org]\ndeadman_minutes = 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		age  time.Duration
		want string
	}{
		{name: "fresh mark", want: "org watch: running (pid 4249, "},
		{name: "30-minute-old mark", age: 30 * time.Minute, want: "org watch: stale (pid 4249, "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := t.TempDir()
			holdWatchLock(t, stateDir, runningMark(4249, time.Now().Add(-tc.age)))
			out, err := runStatusCmd(t, "--state-dir", stateDir)
			if err != nil {
				t.Fatalf("status: %v (output: %s)", err, out)
			}
			if got := statusOrgWatchLine(t, out); !strings.HasPrefix(got, tc.want) {
				t.Errorf("with deadman_minutes = 0, org watch line = %q; want it to start %q", got, tc.want)
			}
		})
	}
}

// TestStatusCmd_OrgWatchFallsBackToTheDefaultDeadmanWhenRalphTomlDoesNotLoad:
// status only reads, so a ./ralph.toml that does not load warns on stderr
// and judges the mark by the default deadman_minutes (10) instead of
// failing. A 5-minute-old mark is running under it (a zero fallback would
// make it stale), and --json stdout stays one JSON document.
func TestStatusCmd_OrgWatchFallsBackToTheDefaultDeadmanWhenRalphTomlDoesNotLoad(t *testing.T) {
	stateDir := t.TempDir()
	mark := runningMark(4248, time.Now().Add(-5*time.Minute))
	holdWatchLock(t, stateDir, mark)
	t.Chdir(t.TempDir())
	if err := os.WriteFile("ralph.toml", []byte("[org\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	root := NewRootCmd()
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"status", "--state-dir", stateDir, "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("status --json with a broken ralph.toml: %v (stdout: %s, stderr: %s)", err, stdout.String(), stderr.String())
	}
	got := statusOrgWatchJSONOf(t, stdout.String())
	if got["state"] != "running" {
		t.Errorf("org_watch state = %v, want running under the default deadman_minutes (10); org_watch = %v", got["state"], got)
	}
	for _, want := range []string{"warning: status: load ralph.toml: ", "the default deadman_minutes (10)"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("stderr should contain %q, got:\n%s", want, stderr.String())
		}
	}

	out, err := runStatusCmd(t, "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status with a broken ralph.toml: %v (output: %s)", err, out)
	}
	if got, want := statusOrgWatchLine(t, out), "org watch: running (pid 4248, last cycle "+mark.TS+")"; got != want {
		t.Errorf("org watch line with a broken ralph.toml:\n got %q\nwant %q", got, want)
	}
}
