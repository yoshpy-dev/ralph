package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
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
			"updated_at": m.TS, "mark_state": m.State,
		}
		if m.StoppedAt != "" {
			want["stopped_at"] = m.StoppedAt
		}
		return want
	}
	const staleTail = " -- a watch holds watch.lock, but its liveness mark is missing, stopped, or older than the larger of deadman_minutes (10 when below 1) and two of its intervals"

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
			wantLine: "org watch: not running (a running mark is left from pid 4243, mark updated 2026-10-11T04:40:00Z)",
			wantJSON: markJSON("not_running", killed),
		},
		{
			name:     "running",
			setup:    func(t *testing.T, stateDir string) { holdWatchLock(t, stateDir, fresh) },
			wantLine: "org watch: running (pid 4244, mark updated " + fresh.TS + ")",
			wantJSON: markJSON("running", fresh),
		},
		{
			name:     "stale with an old mark",
			setup:    func(t *testing.T, stateDir string) { holdWatchLock(t, stateDir, old) },
			wantLine: "org watch: stale (pid 4245, mark updated " + old.TS + ")" + staleTail,
			wantJSON: markJSON("stale", old),
		},
		{
			name:     "stale with a stopped mark",
			setup:    func(t *testing.T, stateDir string) { holdWatchLock(t, stateDir, stopped) },
			wantLine: "org watch: stale (pid 4242, mark updated 2026-10-11T04:40:00Z, stopped at 2026-10-11T04:40:05Z)" + staleTail,
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
// on the populated path: right below the state-dir line, above the director
// line (not attached: the state dir has no registration), the merge
// guarantee line (not checked: the state dir has no cache) and the rosters,
// whose per-org watch heartbeat stays as it was.
func TestStatusCmd_OrgWatchLineFollowsTheStateDirLine(t *testing.T) {
	dir := t.TempDir()
	seedTwoOrgManifest(t, dir)

	out, err := runStatusCmd(t, "--state-dir", dir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	lines := strings.Split(out, "\n")
	if len(lines) < 5 || !strings.HasPrefix(lines[0], "state-dir: ") || lines[1] != "org watch: not running" || lines[2] != directorNotAttachedLine ||
		lines[3] != "merge guarantee: unknown (not checked: run ralph doctor)" || !strings.HasPrefix(lines[4], "org_id: ") {
		t.Errorf("want the state-dir line, the org watch line, the director line, the merge guarantee line, then the first roster, got:\n%s", out)
	}
}

// Tests for the director: line and the director object of `ralph status`
// (plan 2026-10-14-org-director-attach, AC6). The registration is written
// into temp state dirs only, through org.DirectorStore or by hand.

// statusDirectorJSONOf decodes the director object of `ralph status --json`
// output; a missing object fails the test.
func statusDirectorJSONOf(t *testing.T, out string) orgDirectorReport {
	t.Helper()
	var payload struct {
		Director *orgDirectorReport `json:"director"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v\noutput:\n%s", err, out)
	}
	if payload.Director == nil {
		t.Fatalf("--json carries no director object:\n%s", out)
	}
	return *payload.Director
}

// AC6: `ralph status` prints one director: line for each state of the
// registration and the switch, on the empty roster and with orgs, right
// after the org watch line, and --json carries the same state in director.
// status has no --config, so the switch comes from ./ralph.toml here: the
// state dirs are not a main worktree's ledger.
func TestStatusCmd_DirectorLinePerState(t *testing.T) {
	const ts = "2026-10-15T09:00:00Z"
	clock := func() time.Time { return time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC) }
	attach := func(t *testing.T, stateDir string) {
		t.Helper()
		store := org.NewDirectorStore(stateDir)
		store.Now = clock
		if _, _, err := store.Attach(org.DirectorAttach{Driver: org.DirectorDriverClaude, User: "alice", Cwd: "/work"}); err != nil {
			t.Fatalf("attach: %v", err)
		}
	}
	unreadable := func(t *testing.T, stateDir string) {
		t.Helper()
		if err := os.Mkdir(org.DirectorPathIn(stateDir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	readErrPrefix := func(stateDir string) string {
		return "org: director: read " + org.DirectorPathIn(stateDir) + ": "
	}
	cases := []struct {
		name     string
		disabled bool
		setup    func(t *testing.T, stateDir string)
		// check checks the line; wantState and wantID the JSON object.
		check     func(t *testing.T, stateDir, line string)
		wantState string
		wantID    string
	}{
		{
			name: "not attached", setup: func(*testing.T, string) {},
			check: func(t *testing.T, _ string, line string) {
				if line != directorNotAttachedLine {
					t.Errorf("director line = %q; want %q", line, directorNotAttachedLine)
				}
			},
			wantState: "not_attached",
		},
		{
			name: "attached", setup: attach,
			check: func(t *testing.T, _ string, line string) {
				if want := directorAttachedLine("d1", "claude", ts, "alice"); line != want {
					t.Errorf("director line:\n got %q\nwant %q", line, want)
				}
			},
			wantState: "attached", wantID: "d1",
		},
		{
			name: "attached with a pane and no user",
			setup: func(t *testing.T, stateDir string) {
				writeDirectorLines(t, stateDir, directorEventLine(t, org.DirectorEvent{
					TS: ts, ID: "d4", Event: org.DirectorEventAttached, Driver: "codex", Wake: []string{"pull"}, PaneID: "%7",
				}))
			},
			check: func(t *testing.T, _ string, line string) {
				want := "director: attached d4 (driver=codex, attached at " + ts + ", pane %7) -- " +
					"the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline" +
					directorAutoMergeOff
				if line != want {
					t.Errorf("director line:\n got %q\nwant %q", line, want)
				}
			},
			wantState: "attached", wantID: "d4",
		},
		{
			name: "disabled", disabled: true, setup: func(*testing.T, string) {},
			check: func(t *testing.T, _ string, line string) {
				if line != directorDisabledLine {
					t.Errorf("director line = %q; want %q", line, directorDisabledLine)
				}
			},
			wantState: "disabled",
		},
		{
			name: "disabled with a registration left", disabled: true, setup: attach,
			check: func(t *testing.T, _ string, line string) {
				want := "director: disabled ([org.director].enabled = false; registration d1 is still recorded) -- ralph org escalate sends every item to the human at once"
				if line != want {
					t.Errorf("director line:\n got %q\nwant %q", line, want)
				}
			},
			wantState: "disabled", wantID: "d1",
		},
		{
			name: "unreadable", setup: unreadable,
			check: func(t *testing.T, stateDir, line string) {
				if !strings.HasPrefix(line, "director: unreadable ("+readErrPrefix(stateDir)) || !strings.HasSuffix(line, ")") {
					t.Errorf("director line = %q; want unreadable with the read error", line)
				}
			},
			wantState: "unreadable",
		},
		{
			name: "disabled and unreadable", disabled: true, setup: unreadable,
			check: func(t *testing.T, stateDir, line string) {
				if !strings.HasPrefix(line, "director: disabled ([org.director].enabled = false; director.jsonl cannot be read: "+readErrPrefix(stateDir)) ||
					!strings.HasSuffix(line, ") -- ralph org escalate sends every item to the human at once") {
					t.Errorf("director line = %q; want disabled naming the read error", line)
				}
			},
			wantState: "disabled",
		},
	}
	for _, tc := range cases {
		for _, seed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s, orgs %v", tc.name, seed), func(t *testing.T) {
				t.Chdir(t.TempDir())
				if tc.disabled {
					writeTestFile(t, "ralph.toml", "[org.director]\nenabled = false\n")
				}
				stateDir := t.TempDir()
				tc.setup(t, stateDir)
				if seed {
					seedTwoOrgManifest(t, stateDir)
				}

				out, err := runStatusCmd(t, "--state-dir", stateDir)
				if err != nil {
					t.Fatalf("status: %v (output: %s)", err, out)
				}
				line := statusDirectorLine(t, out)
				tc.check(t, stateDir, line)
				lines := strings.Split(out, "\n")
				if i := slices.Index(lines, line); i < 1 || !strings.HasPrefix(lines[i-1], "org watch: ") {
					t.Errorf("the director line does not come right after the org watch line:\n%s", out)
				}

				jsonOut, err := runStatusCmd(t, "--state-dir", stateDir, "--json")
				if err != nil {
					t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
				}
				got := statusDirectorJSONOf(t, jsonOut)
				gotID := ""
				if got.Registration != nil {
					gotID = got.Registration.ID
				}
				if got.State != tc.wantState || got.Enabled == tc.disabled || gotID != tc.wantID {
					t.Errorf("director = %+v; want state %s, enabled %v, registration %q", got, tc.wantState, !tc.disabled, tc.wantID)
				}
				if (tc.name == "unreadable" || tc.name == "disabled and unreadable") != (got.Error != "") {
					t.Errorf("director error = %q for %s", got.Error, tc.name)
				}
			})
		}
	}
}

// AC2 of plan 2026-10-17-org-auto-merge for `ralph status`, with no orgs and
// with two: the director: line of attached and not attached ends with what
// [org.director].auto_merge means, before the note on open headless
// directors; disabled says that auto_merge = true has no effect, and
// unreadable says nothing of it. --json carries director.auto_merge, false
// included. The settings come from ./ralph.toml: the state dirs are not a
// main worktree's ledger.
func TestStatusCmd_DirectorLineShowsAutoMerge(t *testing.T) {
	const (
		ts           = "2026-10-15T09:00:00Z"
		on           = "[org.director]\nauto_merge = true\n"
		attachedTail = " -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline"
		disabledEnd  = ") -- ralph org escalate sends every item to the human at once"
		noEffect     = "[org.director].enabled = false, so [org.director].auto_merge = true has no effect"
	)
	none := func(*testing.T, string) {}
	attach := func(t *testing.T, stateDir string) {
		t.Helper()
		store := org.NewDirectorStore(stateDir)
		store.Now = func() time.Time { return time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC) }
		if _, _, err := store.Attach(org.DirectorAttach{Driver: org.DirectorDriverClaude, User: "alice", Cwd: "/work"}); err != nil {
			t.Fatalf("attach: %v", err)
		}
	}
	openHeadless := func(t *testing.T, stateDir string) {
		t.Helper()
		seedHeadless(t, stateDir, "codex", "gpt-6-astra", "w-1", "p-1", true)
		if _, err := org.NewDirectorStore(stateDir).Detach(org.DirectorDetach{ID: "d1", User: "alice"}); err != nil {
			t.Fatal(err)
		}
	}
	openNote := func(stateDir string) string {
		return "; the headless director d1 (running, pane p-1, workspace w-1) is still open but not attached: stop it with ralph org director stop --id d1" +
			" --state-dir '" + stateDir + "'"
	}
	cases := []struct {
		name          string
		toml          string // ./ralph.toml ("" for none)
		setup         func(t *testing.T, stateDir string)
		wantLine      func(stateDir string) string // "" checks only that the line is unreadable
		wantAutoMerge bool
	}{
		{"not attached, no ralph.toml", "", none, func(string) string { return directorNotAttachedLine }, false},
		{"not attached, auto_merge = false", "[org.director]\nauto_merge = false\n", none, func(string) string { return directorNotAttachedLine }, false},
		{"not attached, auto_merge = true", on, none, func(string) string { return directorNotAttachedBase + directorAutoMergeOnNotAttached }, true},
		{"not attached with an open headless director, auto_merge = true", on, openHeadless, func(stateDir string) string {
			return directorNotAttachedBase + directorAutoMergeOnNotAttached + openNote(stateDir)
		}, true},
		{"attached, auto_merge = false", "", attach, func(string) string { return directorAttachedLine("d1", "claude", ts, "alice") }, false},
		{"attached, auto_merge = true", on, attach, func(string) string {
			return "director: attached d1 (driver=claude, attached at " + ts + ", by alice)" + attachedTail + directorAutoMergeOnAttached
		}, true},
		{"disabled, auto_merge = false", "[org.director]\nenabled = false\n", none, func(string) string { return directorDisabledLine }, false},
		{"disabled, auto_merge = true", "[org.director]\nenabled = false\nauto_merge = true\n", none, func(string) string {
			return "director: disabled (" + noEffect + disabledEnd
		}, true},
		{"disabled with a registration left, auto_merge = true", "[org.director]\nenabled = false\nauto_merge = true\n", attach, func(string) string {
			return "director: disabled (" + noEffect + "; registration d1 is still recorded" + disabledEnd
		}, true},
		{"unreadable, auto_merge = true", on, func(t *testing.T, stateDir string) {
			t.Helper()
			if err := os.Mkdir(org.DirectorPathIn(stateDir), 0o755); err != nil {
				t.Fatal(err)
			}
		}, func(string) string { return "" }, true},
	}
	for _, tc := range cases {
		for _, seed := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s, orgs %v", tc.name, seed), func(t *testing.T) {
				t.Chdir(t.TempDir())
				if tc.toml != "" {
					writeTestFile(t, "ralph.toml", tc.toml)
				}
				stateDir := t.TempDir()
				tc.setup(t, stateDir)
				if seed {
					seedTwoOrgManifest(t, stateDir)
				}

				out, err := runStatusCmd(t, "--state-dir", stateDir)
				if err != nil {
					t.Fatalf("status: %v (output: %s)", err, out)
				}
				line := statusDirectorLine(t, out)
				if want := tc.wantLine(stateDir); want != "" && line != want {
					t.Errorf("director line:\n got %q\nwant %q", line, want)
				} else if want == "" && (!strings.HasPrefix(line, "director: unreadable (") || strings.Contains(line, "[org.director].auto_merge") ||
					strings.Contains(line, "merges")) {
					// The read error holds the TempDir path, which holds this
					// subtest's name, auto_merge included, so only the clause's
					// own words are looked for.
					t.Errorf("director line = %q; want unreadable, saying nothing of auto_merge", line)
				}

				jsonOut, err := runStatusCmd(t, "--state-dir", stateDir, "--json")
				if err != nil {
					t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
				}
				if got := statusDirectorJSONOf(t, jsonOut); got.AutoMerge != tc.wantAutoMerge {
					t.Errorf("director.auto_merge = %v; want %v", got.AutoMerge, tc.wantAutoMerge)
				}
				if key := fmt.Sprintf(`"auto_merge": %t,`, tc.wantAutoMerge); !strings.Contains(jsonOut, key) {
					t.Errorf("status --json does not carry %s:\n%s", key, jsonOut)
				}
			})
		}
	}
}

// AC10 of plan 2026-10-15-org-director-start for `ralph status`, with no
// orgs and with two: a current headless registration shows headless, its
// state, model, and the herdr ids recorded on the director: line, and an
// open headless registration that is not current is named at the end of it
// with the stop command, attached, disabled, and not attached; --json carries
// the headless fields and director.open_headless.
func TestStatusCmd_DirectorLineShowsHeadlessDirectors(t *testing.T) {
	const tail = " -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline" +
		directorAutoMergeOff
	for _, seed := range []bool{false, true} {
		t.Run(fmt.Sprintf("orgs %v", seed), func(t *testing.T) {
			t.Chdir(t.TempDir())
			stateDir := t.TempDir()
			if seed {
				seedTwoOrgManifest(t, stateDir)
			}
			store := org.NewDirectorStore(stateDir)
			seedHeadless(t, stateDir, "codex", "gpt-6-astra", "w-1", "p-1", true)
			if _, err := store.Detach(org.DirectorDetach{ID: "d1", User: "alice"}); err != nil {
				t.Fatal(err)
			}
			d2 := seedHeadless(t, stateDir, "claude", "opus", "w-2", "", false)
			suffix := " --state-dir '" + stateDir + "'"
			d1Note := "; the headless director d1 (running, pane p-1, workspace w-1) is still open but not attached: stop it with ralph org director stop --id d1" + suffix
			line := func() string {
				t.Helper()
				out, err := runStatusCmd(t, "--state-dir", stateDir)
				if err != nil {
					t.Fatalf("status: %v (output: %s)", err, out)
				}
				return statusDirectorLine(t, out)
			}

			want := "director: attached d2 (driver=claude, headless starting, model=opus, workspace w-2, attached at " + d2.AttachedAt + ", by alice)" + tail + d1Note
			if got := line(); got != want {
				t.Errorf("director line:\n got %q\nwant %q", got, want)
			}
			jsonOut, err := runStatusCmd(t, "--state-dir", stateDir, "--json")
			if err != nil {
				t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
			}
			got := statusDirectorJSONOf(t, jsonOut)
			if reg := got.Registration; got.State != "attached" || reg == nil || reg.ID != "d2" || reg.Launch != org.DirectorLaunchHerdr ||
				reg.State != org.DirectorStateStarting || reg.Model != "opus" || reg.WorkspaceID != "w-2" || reg.PaneID != "" || reg.Token != d2.Token ||
				len(got.OpenHeadless) != 1 || got.OpenHeadless[0].ID != "d1" || got.OpenHeadless[0].State != org.DirectorStateRunning ||
				got.OpenHeadless[0].PaneID != "p-1" || got.OpenHeadless[0].WorkspaceID != "w-1" {
				t.Errorf("director = %+v; want d2 headless starting with w-2, and d1 running in open_headless", got)
			}
			for _, key := range []string{`"launch": "herdr"`, `"state": "starting"`, `"model": "opus"`, `"workspace_id": "w-2"`, `"open_headless": [`, `"pane_id": "p-1"`} {
				if !strings.Contains(jsonOut, key) {
					t.Errorf("status --json does not carry %s:\n%s", key, jsonOut)
				}
			}

			writeTestFile(t, "ralph.toml", "[org.director]\nenabled = false\n")
			want = "director: disabled ([org.director].enabled = false; registration d2 is still recorded, headless starting, workspace w-2: " +
				"stop it with ralph org director stop --id d2" + suffix + ") -- ralph org escalate sends every item to the human at once" + d1Note
			if got := line(); got != want {
				t.Errorf("director line with the switch off:\n got %q\nwant %q", got, want)
			}

			if err := os.Remove("ralph.toml"); err != nil {
				t.Fatal(err)
			}
			if _, err := store.Detach(org.DirectorDetach{ID: "d2", User: "alice"}); err != nil {
				t.Fatal(err)
			}
			want = directorNotAttachedLine + "; the headless directors d1 (running, pane p-1, workspace w-1), d2 (starting, workspace w-2) are still open but not attached: " +
				"stop each with ralph org director stop --id <id>" + suffix
			if got := line(); got != want {
				t.Errorf("director line with no registration current:\n got %q\nwant %q", got, want)
			}
		})
	}
}

// AC8 of plan 2026-10-16-org-director-launch-cmd for `ralph status`: a
// launch cmd's registration shows launch cmd, the name of its terminal, and
// its pid once checked in, in place of the herdr ids, attached, open but not
// attached, and disabled; --json carries launch, state, terminal_name, pid,
// start_time, and model. The pids are made up: status reads or signals no
// process.
func TestStatusCmd_DirectorLineShowsLaunchCmdDirectors(t *testing.T) {
	const tail = " -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline" +
		directorAutoMergeOff
	t.Chdir(t.TempDir())
	stateDir := t.TempDir()
	store := org.NewDirectorStore(stateDir)
	who := org.DirectorWho{User: "alice", Cwd: "/work"}
	d1 := seedCmdHeadless(t, stateDir)
	if err := store.MarkCmdLaunched(org.DirectorCmdCheckIn{ID: "d1", Token: d1.Token, PID: 4242, StartTime: directorTestStartTime, Who: who}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Detach(org.DirectorDetach{ID: "d1", User: "alice"}); err != nil {
		t.Fatal(err)
	}
	d2 := seedCmdHeadless(t, stateDir)
	name1, name2 := "director_"+d1.Token, "director_"+d2.Token
	suffix := " --state-dir '" + stateDir + "'"
	d1Open := "d1 (running, launch cmd, name " + name1 + ", pid 4242)"
	d1Note := "; the headless director " + d1Open + " is still open but not attached: stop it with ralph org director stop --id d1" + suffix
	line := func() string {
		t.Helper()
		out, err := runStatusCmd(t, "--state-dir", stateDir)
		if err != nil {
			t.Fatalf("status: %v (output: %s)", err, out)
		}
		return statusDirectorLine(t, out)
	}

	want := "director: attached d2 (driver=claude, headless starting, model=opus, launch cmd, name " + name2 + ", attached at " + d2.AttachedAt + ", by alice)" + tail + d1Note
	if got := line(); got != want {
		t.Errorf("director line:\n got %q\nwant %q", got, want)
	}
	jsonOut, err := runStatusCmd(t, "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
	}
	got := statusDirectorJSONOf(t, jsonOut)
	if reg := got.Registration; got.State != "attached" || reg == nil || reg.ID != "d2" || reg.Launch != org.DirectorLaunchCmd || reg.State != org.DirectorStateStarting ||
		reg.Model != "opus" || reg.TerminalName != name2 || reg.PID != 0 || len(got.OpenHeadless) != 1 || got.OpenHeadless[0].ID != "d1" ||
		got.OpenHeadless[0].PID != 4242 || got.OpenHeadless[0].StartTime != directorTestStartTime || got.OpenHeadless[0].TerminalName != name1 {
		t.Errorf("director = %+v; want d2 starting with launch cmd and %s, and d1 with pid 4242 in open_headless", got, name2)
	}
	for _, key := range []string{`"launch": "cmd"`, `"state": "starting"`, `"terminal_name": "` + name2 + `"`, `"pid": 4242`, `"start_time": "` + directorTestStartTime + `"`} {
		if !strings.Contains(jsonOut, key) {
			t.Errorf("status --json does not carry %s:\n%s", key, jsonOut)
		}
	}

	if err := store.MarkCmdLaunched(org.DirectorCmdCheckIn{ID: "d2", Token: d2.Token, PID: 4343, StartTime: directorTestStartTime, Who: who}); err != nil {
		t.Fatal(err)
	}
	want = "director: attached d2 (driver=claude, headless running, model=opus, launch cmd, name " + name2 + ", pid 4343, attached at " + d2.AttachedAt + ", by alice)" +
		tail + d1Note
	if got := line(); got != want {
		t.Errorf("director line of a running d2:\n got %q\nwant %q", got, want)
	}

	writeTestFile(t, "ralph.toml", "[org.director]\nenabled = false\n")
	want = "director: disabled ([org.director].enabled = false; registration d2 is still recorded, headless running, launch cmd, name " + name2 + ", pid 4343: " +
		"stop it with ralph org director stop --id d2" + suffix + ") -- ralph org escalate sends every item to the human at once" + d1Note
	if got := line(); got != want {
		t.Errorf("director line with the switch off:\n got %q\nwant %q", got, want)
	}

	if err := os.Remove("ralph.toml"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Detach(org.DirectorDetach{ID: "d2", User: "alice"}); err != nil {
		t.Fatal(err)
	}
	want = directorNotAttachedLine + "; the headless directors " + d1Open + ", d2 (running, launch cmd, name " + name2 + ", pid 4343) are still open but not attached: " +
		"stop each with ralph org director stop --id <id>" + suffix
	if got := line(); got != want {
		t.Errorf("director line with no registration current:\n got %q\nwant %q", got, want)
	}
}

// Self-review L3 of org-director-start: the stop command on the director:
// line repeats --state-dir exactly when it was given, through `ralph status`
// and `ralph org director status` alike, shell-quoted so a state dir with a
// space stays one argument, and names none when RALPH_ORG_STATE_DIR names
// the ledger instead, as the other director commands do
// (org.CommandWithStateDir).
func TestStatusCmd_DirectorStopCommandRepeatsTheStateDir(t *testing.T) {
	t.Chdir(t.TempDir())
	stateDir := filepath.Join(t.TempDir(), "a ledger")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	seedHeadless(t, stateDir, "codex", "gpt-6-astra", "w-1", "p-1", true)
	if _, err := org.NewDirectorStore(stateDir).Detach(org.DirectorDetach{ID: "d1", User: "alice"}); err != nil {
		t.Fatal(err)
	}
	const stop = ": stop it with ralph org director stop --id d1"
	statusLine := func(args ...string) string {
		t.Helper()
		out, err := runStatusCmd(t, args...)
		if err != nil {
			t.Fatalf("status %q: %v (output: %s)", args, err, out)
		}
		return statusDirectorLine(t, out)
	}

	withFlag := stop + " --state-dir '" + stateDir + "'"
	for name, line := range map[string]string{
		"ralph status --state-dir":              statusLine("--state-dir", stateDir),
		"ralph org director status --state-dir": directorStatusLine(t, "--state-dir", stateDir),
	} {
		if !strings.HasSuffix(line, withFlag) {
			t.Errorf("%s: director line %q does not end with %q", name, line, withFlag)
		}
	}

	t.Setenv("RALPH_ORG_STATE_DIR", stateDir)
	for name, line := range map[string]string{
		"ralph status":              statusLine(),
		"ralph org director status": directorStatusLine(t),
	} {
		if !strings.HasSuffix(line, stop) || strings.Contains(line, "--state-dir") {
			t.Errorf("%s with RALPH_ORG_STATE_DIR: director line %q; want it to end with %q and name no --state-dir", name, line, stop)
		}
	}
}

// AC5, AC6: with no --state-dir in a linked worktree, `ralph status` reads
// the switch from the main worktree's ralph.toml, as escalate does, whatever
// the worktree's ralph.toml says. A main worktree ralph.toml that does not
// load warns on stderr and shows the built-in default (true); status still
// exits 0 and --json stays one JSON document.
func TestStatusCmd_DirectorEnabledFromTheMainWorktree(t *testing.T) {
	r := newLegacyLedgerRepo(t)
	writeTestFile(t, filepath.Join(r.mainRoot, "ralph.toml"), "[org.director]\nenabled = false\n")
	writeTestFile(t, filepath.Join(r.worktree, "ralph.toml"), "[org.director]\nenabled = true\n")
	stdout, stderr, err := runRootCmdSplit(t, "status")
	if err != nil {
		t.Fatalf("status: %v (stderr %s)", err, stderr)
	}
	if got := statusDirectorLine(t, stdout); got != directorDisabledLine {
		t.Errorf("director line = %q; want %q from the main worktree's ralph.toml", got, directorDisabledLine)
	}

	mainToml := filepath.Join(r.mainRoot, "ralph.toml")
	writeTestFile(t, mainToml, "[org]\nmax_orgs = 0\n")
	for _, extra := range [][]string{nil, {"--json"}} {
		stdout, stderr, err = runRootCmdSplit(t, append([]string{"status"}, extra...)...)
		if err != nil {
			t.Fatalf("status %v with a broken main ralph.toml: %v (stderr %s)", extra, err, stderr)
		}
		for _, want := range []string{"warning: status: org: load [org.director].enabled and auto_merge from the main worktree's " + mainToml,
			"showing the director by the default [org.director].enabled (true) and auto_merge (false)\n"} {
			if !strings.Contains(stderr, want) {
				t.Errorf("status %v stderr does not contain %q:\n%s", extra, want, stderr)
			}
		}
		if extra == nil {
			if got := statusDirectorLine(t, stdout); got != directorNotAttachedLine {
				t.Errorf("director line with a broken main ralph.toml = %q; want %q", got, directorNotAttachedLine)
			}
		} else if got := statusDirectorJSONOf(t, stdout); got.State != "not_attached" || !got.Enabled {
			t.Errorf("director with a broken main ralph.toml = %+v; want enabled, not_attached", got)
		}
	}
}

// status only reads, so a ./ralph.toml that does not load warns and shows
// the built-in default of the switch instead of failing. status reads the
// file once, for the watch's deadman_minutes and the switch, so stderr holds
// exactly one warning line about it (self-review L1 of
// org-director-attach), also in a main checkout, where ./ralph.toml is the
// main worktree's file the switch would be read from, and with --json.
func TestStatusCmd_DirectorFallsBackToTheDefaultWhenRalphTomlDoesNotLoad(t *testing.T) {
	const wantSuffix = "; judging the org watch by the default deadman_minutes (10) and showing the director by the default [org.director].enabled (true) and auto_merge (false)\n"
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T) []string // chdirs and returns status's extra args
	}{
		{"a --state-dir outside git", func(t *testing.T) []string {
			stateDir := t.TempDir()
			t.Chdir(t.TempDir())
			return []string{"--state-dir", stateDir}
		}},
		{"the main checkout's own ralph.toml", func(t *testing.T) []string {
			r := newLegacyLedgerRepo(t)
			t.Chdir(r.mainRoot)
			return nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"status"}, tc.setup(t)...)
			writeTestFile(t, "ralph.toml", "[org\n")
			for _, extra := range [][]string{nil, {"--json"}} {
				stdout, stderr, err := runRootCmdSplit(t, append(args, extra...)...)
				if err != nil {
					t.Fatalf("status %v: %v (stderr %s)", extra, err, stderr)
				}
				if extra == nil {
					if got := statusDirectorLine(t, stdout); got != directorNotAttachedLine {
						t.Errorf("director line = %q; want %q", got, directorNotAttachedLine)
					}
				} else if got := statusDirectorJSONOf(t, stdout); got.State != "not_attached" || !got.Enabled {
					t.Errorf("director = %+v; want enabled, not_attached", got)
				}
				if !strings.HasPrefix(stderr, "warning: status: load ralph.toml: ") || !strings.HasSuffix(stderr, wantSuffix) || strings.Count(stderr, "\n") != 1 {
					t.Errorf("status %v stderr = %q; want exactly one load warning ending %q", extra, stderr, wantSuffix)
				}
			}
		})
	}
}

// status warns on stderr about damage in director.jsonl, as director status
// does, and leaves --json stdout one JSON document.
func TestStatusCmd_DirectorDamageWarnsOnStderr(t *testing.T) {
	stateDir := t.TempDir()
	writeDirectorLines(t, stateDir, `{"id":"d1","event":"atta`)
	stdout, stderr, err := runRootCmdSplit(t, "status", "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (stderr %s)", err, stderr)
	}
	if got := statusDirectorJSONOf(t, stdout); got.CorruptLines != 1 || got.State != "not_attached" {
		t.Errorf("director = %+v; want not_attached with 1 corrupt line", got)
	}
	if want := "warning: 1 unreadable director line(s) skipped in " + org.DirectorPathIn(stateDir) + "\n"; !strings.Contains(stderr, want) {
		t.Errorf("stderr = %q; want %q", stderr, want)
	}
}

// The status help says what the director: line shows and that a
// registration does not mean its session is alive, and names the git that
// finds the main worktree for the switch besides the guarantee's get-url
// (self-review M1 of org-director-attach).
func TestStatusCmd_HelpDescribesTheDirectorLine(t *testing.T) {
	out, err := runStatusCmd(t, "--help")
	if err != nil {
		t.Fatalf("status --help: %v", err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"A `director:` line follows it", "read as `ralph org escalate` reads it without --config",
		"It is attached,", "not attached;", "disabled (enabled = false: escalate sends every item to the human at once)",
		"or unreadable, when director.jsonl cannot be read", "A registration does not tell whether its session is still running",
		"A `merge guarantee:` line follows it",
		"for this line git runs only the local `git remote get-url origin` in the current directory, and only when the cache is fresh",
		"status runs no gh and does not use the network.",
		"the local git rev-parse (and git worktree list --porcelain in a linked worktree) that finds the main worktree",
		"when one of them does, to tell whether it names the main worktree's ledger, whose ralph.toml then holds [org.director].enabled, whether or not the cache is fresh",
		// Plan 2026-10-16-org-director-launch-cmd, AC8.
		"(for one launched with [org.director].launch_cmd, `launch cmd`, the name of its terminal, director_<token>, and the pid its launch script checked in with, " +
			"in place of the pane and workspace; the JSON adds terminal_name)",
		// Plan 2026-10-17-org-auto-merge, AC2.
		"Attached and not attached end with [org.director].auto_merge, read by the same rule as enabled: merges are left to the human (false), " +
			"or a director merges the PRs `ralph org merge-order` lists as ready with `ralph org merge`, without asking (true); " +
			"disabled says that auto_merge = true has no effect. The JSON has director.auto_merge, false included.",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("status --help does not say %q:\n%s", want, out)
		}
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
	if got, want := statusOrgWatchLine(t, out), "org watch: running (pid 4246, mark updated "+mark.TS+")"; got != want {
		t.Errorf("org watch line on the empty roster:\n got %q\nwant %q", got, want)
	}

	jsonOut, err := runStatusCmd(t, "--state-dir", dir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
	}
	got := statusOrgWatchJSONOf(t, jsonOut)
	if got["state"] != "running" || got["pid"] != float64(4246) || got["updated_at"] != mark.TS {
		t.Errorf("org_watch on the empty roster = %v, want state running, pid 4246, updated_at %s", got, mark.TS)
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
	if got, want := statusOrgWatchLine(t, out), "org watch: running (pid 4247, mark updated "+mark.TS+")"; got != want {
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
	if got, want := statusOrgWatchLine(t, out), "org watch: running (pid 4248, mark updated "+mark.TS+")"; got != want {
		t.Errorf("org watch line with a broken ralph.toml:\n got %q\nwant %q", got, want)
	}
}

// Plan 2026-10-12-org-watch-gate, AC13 and AC14 on `ralph status`: a watch
// that has run its last cycle and still holds watch.lock is stopping, not
// "not running" or stale, and a running --once watch says (--once), since
// neither accepts new work; --json carries state stopping and once. A
// stopping mark left by a killed watch, and one held but old, read as a
// running mark would.
func TestStatusCmd_OrgWatchShowsStoppingAndOnce(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	stopping := func(pid int, ts time.Time) *org.WatchAlive {
		mark := runningMark(pid, ts)
		mark.State = org.WatchAliveStopping
		return mark
	}
	fresh := stopping(4250, now)
	old := stopping(4251, now.Add(-time.Hour))
	killed := stopping(4252, now)
	once := runningMark(4253, now)
	once.Once = true
	markJSON := func(state string, m *org.WatchAlive) map[string]any {
		want := map[string]any{
			"state": state, "pid": float64(m.PID), "started_at": m.StartedAt,
			"updated_at": m.TS, "mark_state": m.State,
		}
		if m.Once {
			want["once"] = true
		}
		return want
	}
	const staleTail = " -- a watch holds watch.lock, but its liveness mark is missing, stopped, or older than the larger of deadman_minutes (10 when below 1) and two of its intervals"

	cases := []struct {
		name     string
		setup    func(t *testing.T, stateDir string)
		wantLine string
		wantJSON map[string]any
	}{
		{
			name:     "stopping",
			setup:    func(t *testing.T, stateDir string) { holdWatchLock(t, stateDir, fresh) },
			wantLine: "org watch: stopping (pid 4250, mark updated " + fresh.TS + ") -- it has run its last cycle and accepts no new work; it releases watch.lock once its in-flight work ends",
			wantJSON: markJSON("stopping", fresh),
		},
		{
			name:     "running --once",
			setup:    func(t *testing.T, stateDir string) { holdWatchLock(t, stateDir, once) },
			wantLine: "org watch: running (--once) (pid 4253, mark updated " + once.TS + ") -- it ends after its cycle and accepts no new work",
			wantJSON: markJSON("running", once),
		},
		{
			name:     "stale with an old stopping mark",
			setup:    func(t *testing.T, stateDir string) { holdWatchLock(t, stateDir, old) },
			wantLine: "org watch: stale (pid 4251, mark updated " + old.TS + ")" + staleTail,
			wantJSON: markJSON("stale", old),
		},
		{
			name:     "not running with the stopping mark of a killed watch",
			setup:    func(t *testing.T, stateDir string) { writeWatchMark(t, stateDir, killed) },
			wantLine: "org watch: not running (a stopping mark is left from pid 4252, mark updated " + killed.TS + ")",
			wantJSON: markJSON("not_running", killed),
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

// Plan 2026-10-12-org-watch-gate, AC7 on `ralph status`: a mark that carries
// the watch's own deadman_minutes is judged by it, not by ./ralph.toml, so a
// watch started with another --config reads right. A 30-minute-old mark of a
// watch with 60 is running with no ralph.toml (10), and one of a watch with 0
// (the default 10) is stale under a ralph.toml with 60.
func TestStatusCmd_OrgWatchJudgesTheMarkByTheWatchsOwnDeadmanMinutes(t *testing.T) {
	for _, tc := range []struct {
		name      string
		watchDM   int
		ralphToml string
		want      string
	}{
		{name: "watch 60, no ralph.toml", watchDM: 60, want: "org watch: running (pid 4254, "},
		{name: "watch 0, ralph.toml 60", watchDM: 0, ralphToml: "[org]\ndeadman_minutes = 60\n", want: "org watch: stale (pid 4254, "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if tc.ralphToml != "" {
				if err := os.WriteFile("ralph.toml", []byte(tc.ralphToml), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			stateDir := t.TempDir()
			mark := runningMark(4254, time.Now().Add(-30*time.Minute))
			mark.DeadmanMinutes = &tc.watchDM
			holdWatchLock(t, stateDir, mark)
			out, err := runStatusCmd(t, "--state-dir", stateDir)
			if err != nil {
				t.Fatalf("status: %v (output: %s)", err, out)
			}
			if got := statusOrgWatchLine(t, out); !strings.HasPrefix(got, tc.want) {
				t.Errorf("org watch line = %q; want it to start %q", got, tc.want)
			}
		})
	}
}

// Plan 2026-10-12-org-watch-gate, AC7 on `ralph status`, the case of a watch
// run with a --config whose deadman_minutes differs from ./ralph.toml's: a
// 20-minute-old mark of a watch with 30 reads running under a ralph.toml with
// 10, in the text and in --json, while the same mark without
// deadman_minutes, as an older version wrote it, reads stale by the 10.
func TestStatusCmd_OrgWatchOfAWatchWithAnotherConfigReadsByItsMark(t *testing.T) {
	thirty := 30
	for _, tc := range []struct {
		name    string
		watchDM *int
		state   string
	}{
		{name: "mark with deadman_minutes 30", watchDM: &thirty, state: org.WatchLivenessRunning},
		{name: "mark without deadman_minutes", state: org.WatchLivenessStale},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if err := os.WriteFile("ralph.toml", []byte("[org]\ndeadman_minutes = 10\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			stateDir := t.TempDir()
			mark := runningMark(4255, time.Now().Add(-20*time.Minute))
			mark.DeadmanMinutes = tc.watchDM
			holdWatchLock(t, stateDir, mark)

			out, err := runStatusCmd(t, "--state-dir", stateDir)
			if err != nil {
				t.Fatalf("status: %v (output: %s)", err, out)
			}
			if want := "org watch: " + tc.state + " (pid 4255, "; !strings.HasPrefix(statusOrgWatchLine(t, out), want) {
				t.Errorf("org watch line = %q; want it to start %q", statusOrgWatchLine(t, out), want)
			}
			jsonOut, err := runStatusCmd(t, "--state-dir", stateDir, "--json")
			if err != nil {
				t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
			}
			if got := statusOrgWatchJSONOf(t, jsonOut)["state"]; got != tc.state {
				t.Errorf("org_watch state = %v; want %s", got, tc.state)
			}
		})
	}
}
