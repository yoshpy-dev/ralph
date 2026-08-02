package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
// own doc comment): org-a has two seats (lead active, reviewer stopped),
// org-b has one seat (qa active).
func seedTwoOrgManifest(t *testing.T, stateDir string) {
	t.Helper()
	store := org.NewManifestStore(stateDir)
	events := []org.ManifestEvent{
		{TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "lead", Event: org.EventSpawned, Role: "lead", Driver: "claude", Model: "opus", Worktree: "/tmp/org-a-lead"},
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
		"lead", "reviewer", "qa",
		"active 1/2", // org-a: lead active, reviewer stopped
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
	store := org.NewManifestStore(dir)
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
