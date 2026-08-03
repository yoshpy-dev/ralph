package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/insights"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// writeTestEvent writes one JSON event line to a file in dir.
func writeTestEvent(t *testing.T, dir, filename, line string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(filepath.Join(dir, filename), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.WriteString(line + "\n"); err != nil {
		t.Fatal(err)
	}
}

// runInsightsCmd runs "ralph insights" with the given args and returns stdout.
func runInsightsCmd(t *testing.T, args ...string) string {
	t.Helper()
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(append([]string{"insights"}, args...))
	if err := root.Execute(); err != nil {
		t.Fatalf("command error: %v (output: %s)", err, buf.String())
	}
	return buf.String()
}

func TestInsightsCmd_NoData(t *testing.T) {
	dir := t.TempDir()
	eventsDir := filepath.Join(dir, "events")
	receiptsPath := filepath.Join(dir, "receipts.jsonl")

	out := runInsightsCmd(t,
		"--events-dir", eventsDir,
		"--receipts", receiptsPath,
	)

	if !strings.Contains(out, "No insight data yet") {
		t.Errorf("expected 'No insight data yet', got:\n%s", out)
	}
	if !strings.Contains(out, eventsDir) {
		t.Errorf("expected events dir %q in output, got:\n%s", eventsDir, out)
	}
	if !strings.Contains(out, receiptsPath) {
		t.Errorf("expected receipts path %q in output, got:\n%s", receiptsPath, out)
	}
}

// TestInsightsCmd_JSONZeroData verifies that --json on an empty data dir
// emits valid JSON (not the human-readable "no data yet" message). Fix 4.
func TestInsightsCmd_JSONZeroData(t *testing.T) {
	dir := t.TempDir()
	eventsDir := filepath.Join(dir, "events")
	receiptsPath := filepath.Join(dir, "receipts.jsonl")

	out := runInsightsCmd(t,
		"--events-dir", eventsDir,
		"--receipts", receiptsPath,
		"--json",
	)

	// Must be parseable JSON, not a human message.
	var agg insights.AggregateResult
	if err := json.Unmarshal([]byte(out), &agg); err != nil {
		t.Fatalf("--json zero-data output is not valid JSON: %v\noutput:\n%s", err, out)
	}
	if agg.TotalEvents != 0 {
		t.Errorf("TotalEvents = %d, want 0 (empty events dir)", agg.TotalEvents)
	}
	// Must NOT contain human-mode text.
	if strings.Contains(out, "No insight data yet") {
		t.Errorf("--json mode must not emit human message, got:\n%s", out)
	}
}

func TestInsightsCmd_HumanOutput(t *testing.T) {
	dir := t.TempDir()
	eventsDir := filepath.Join(dir, "events")

	// Write a valid event.
	line := `{"schema":1,"ts":"2026-07-13T01:00:00Z","run_id":"r1","slug":"my-task","flow":"loop","phase":"verify","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"driver":"claude","requested_model":"sonnet","effective_model":"sonnet","honored":true,"source":"pipeline"}`
	writeTestEvent(t, eventsDir, "2026-07-13-my-task.jsonl", line)

	out := runInsightsCmd(t,
		"--events-dir", eventsDir,
		"--receipts", filepath.Join(dir, "nonexistent-receipts.jsonl"),
	)

	if !strings.Contains(out, "Events") {
		t.Errorf("expected 'Events' section, got:\n%s", out)
	}
	if !strings.Contains(out, "verify") {
		t.Errorf("expected 'verify' phase in output, got:\n%s", out)
	}
	// 1 event, 1 pass
	if !strings.Contains(out, "1") {
		t.Errorf("expected count '1' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Routing") {
		t.Errorf("expected 'Routing' section, got:\n%s", out)
	}
	if !strings.Contains(out, "100%") {
		t.Errorf("expected '100%%' honored-rate, got:\n%s", out)
	}
}

func TestInsightsCmd_JSONOutput(t *testing.T) {
	dir := t.TempDir()
	eventsDir := filepath.Join(dir, "events")

	line := `{"schema":1,"ts":"2026-07-13T02:00:00Z","run_id":"r2","slug":"json-task","flow":"loop","phase":"self_review","cycle":1,"verdict":"pass","findings":{"critical":0,"high":1,"medium":2,"low":3},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"driver":"claude","requested_model":"opus","effective_model":"opus","honored":true,"source":"pipeline"}`
	writeTestEvent(t, eventsDir, "2026-07-13-json-task.jsonl", line)

	out := runInsightsCmd(t,
		"--events-dir", eventsDir,
		"--receipts", filepath.Join(dir, "nonexistent.jsonl"),
		"--json",
	)

	var agg insights.AggregateResult
	if err := json.Unmarshal([]byte(out), &agg); err != nil {
		t.Fatalf("json.Unmarshal: %v\noutput:\n%s", err, out)
	}
	if agg.TotalEvents != 1 {
		t.Errorf("TotalEvents = %d, want 1", agg.TotalEvents)
	}
	sr := agg.PerPhase["self_review"]
	if sr == nil {
		t.Fatal("PerPhase[self_review] = nil")
	}
	if sr.Findings.High != 1 {
		t.Errorf("findings.high = %d, want 1", sr.Findings.High)
	}
	if sr.Findings.Medium != 2 {
		t.Errorf("findings.medium = %d, want 2", sr.Findings.Medium)
	}
	if sr.Findings.Low != 3 {
		t.Errorf("findings.low = %d, want 3", sr.Findings.Low)
	}
}

func TestInsightsCmd_EscalationShown(t *testing.T) {
	dir := t.TempDir()
	eventsDir := filepath.Join(dir, "events")

	// cycle 1 fail, cycle 2 pass
	lines := []string{
		`{"schema":1,"ts":"2026-07-13T03:00:00Z","run_id":"r3","slug":"esc-task","flow":"loop","phase":"verify","cycle":1,"verdict":"fail","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"driver":"claude","requested_model":"sonnet","effective_model":"sonnet","honored":true,"source":"pipeline"}`,
		`{"schema":1,"ts":"2026-07-13T03:10:00Z","run_id":"r4","slug":"esc-task","flow":"loop","phase":"verify","cycle":2,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"driver":"claude","requested_model":"opus","effective_model":"opus","honored":true,"source":"pipeline"}`,
	}
	for _, l := range lines {
		writeTestEvent(t, eventsDir, "2026-07-13-esc-task.jsonl", l)
	}

	out := runInsightsCmd(t,
		"--events-dir", eventsDir,
		"--receipts", filepath.Join(dir, "nonexistent.jsonl"),
	)

	if !strings.Contains(out, "esc-task") {
		t.Errorf("expected 'esc-task' in escalation section, got:\n%s", out)
	}
	if !strings.Contains(out, "cycle1=fail") {
		t.Errorf("expected 'cycle1=fail', got:\n%s", out)
	}
	if !strings.Contains(out, "final=pass") {
		t.Errorf("expected 'final=pass', got:\n%s", out)
	}
}

// TestInsightsCmd_HistoricalLoopFlowEventStillReadable is a read-compat
// regression test for org-runtime-retire-loop plan AC-6b: `ralph insights`
// must keep aggregating historical committed events recorded before Ralph
// Loop's execution scripts (scripts/ralph-orchestrator.sh,
// scripts/ralph-pipeline.sh) were retired -- flow="loop"/source="pipeline"
// is historical schema vocabulary, not an active code path. internal/insights
// has no import of any package deleted in this plan (internal/state,
// internal/action, internal/watcher, internal/ui, cmd/ralph-tui); this test
// pins that the deletion did not also break reading their historical output.
func TestInsightsCmd_HistoricalLoopFlowEventStillReadable(t *testing.T) {
	dir := t.TempDir()
	eventsDir := filepath.Join(dir, "events")

	// Shaped exactly as ralph-pipeline.sh's claude -p agents used to emit
	// it pre-retirement.
	line := `{"schema":1,"ts":"2026-06-01T00:00:00Z","run_id":"legacy-1","slug":"legacy-loop-task","flow":"loop","phase":"verify","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"driver":"claude","requested_model":"sonnet","effective_model":"sonnet","honored":true,"source":"pipeline"}`
	writeTestEvent(t, eventsDir, "2026-06-01-legacy-loop-task.jsonl", line)

	out := runInsightsCmd(t,
		"--events-dir", eventsDir,
		"--receipts", filepath.Join(dir, "nonexistent-receipts.jsonl"),
		"--json",
	)

	var agg insights.AggregateResult
	if err := json.Unmarshal([]byte(out), &agg); err != nil {
		t.Fatalf("json.Unmarshal: %v\noutput:\n%s", err, out)
	}
	if agg.TotalEvents != 1 {
		t.Errorf("TotalEvents = %d, want 1 (historical flow=loop event should still be read)", agg.TotalEvents)
	}
	verify := agg.PerPhase["verify"]
	if verify == nil {
		t.Fatal("PerPhase[verify] = nil; historical loop-flow event was not aggregated")
	}
	if verify.Verdicts.Pass != 1 {
		t.Errorf("verify.pass = %d, want 1", verify.Verdicts.Pass)
	}
}

// --- Receipts (org runtime) tests: AC-3 output contract ---

// TestInsightsCmd_ReceiptsDefaultPathFromOrgStateDir pins that omitting
// --receipts resolves the same way "ralph org" verbs resolve their state
// dir (env RALPH_ORG_STATE_DIR here), landing on
// org.ReceiptsPathIn(stateDir) -- not the retired
// .harness/state/pipeline/model-receipts.jsonl path.
func TestInsightsCmd_ReceiptsDefaultPathFromOrgStateDir(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv(org.EnvOrgStateDir, stateDir)

	receiptsPath := org.ReceiptsPathIn(stateDir)
	store := org.NewReceiptStoreAtPath(receiptsPath)
	if err := store.Append(org.Receipt{
		TS:             "2026-08-03T01:00:00Z",
		OrgID:          "demo",
		SeatID:         "lead",
		Driver:         "claude",
		CommandedModel: "opus",
		Honored:        org.HonoredTrue,
	}); err != nil {
		t.Fatalf("append receipt: %v", err)
	}

	dir := t.TempDir()
	eventsDir := filepath.Join(dir, "events")

	out := runInsightsCmd(t, "--events-dir", eventsDir)

	if !strings.Contains(out, "ORG demo") || !strings.Contains(out, "SEAT lead") {
		t.Errorf("expected default-path receipts (resolved via RALPH_ORG_STATE_DIR) to be read, got:\n%s", out)
	}
	if !strings.Contains(out, receiptsPath) {
		t.Errorf("expected resolved receipts path %q in output, got:\n%s", receiptsPath, out)
	}
}

// TestInsightsCmd_ReceiptsExplicitFlagOverridesDefault pins that an explicit
// --receipts always wins over the org-state-dir default, even when
// RALPH_ORG_STATE_DIR is set to a different (receipts-less) directory.
func TestInsightsCmd_ReceiptsExplicitFlagOverridesDefault(t *testing.T) {
	emptyStateDir := t.TempDir()
	t.Setenv(org.EnvOrgStateDir, emptyStateDir)

	dir := t.TempDir()
	explicitPath := filepath.Join(dir, "explicit-receipts.jsonl")
	store := org.NewReceiptStoreAtPath(explicitPath)
	if err := store.Append(org.Receipt{
		TS:             "2026-08-03T02:00:00Z",
		OrgID:          "acme",
		SeatID:         "reviewer",
		Driver:         "codex",
		CommandedModel: "sonnet",
		Honored:        org.HonoredFalse,
	}); err != nil {
		t.Fatalf("append receipt: %v", err)
	}

	eventsDir := filepath.Join(dir, "events")
	out := runInsightsCmd(t, "--events-dir", eventsDir, "--receipts", explicitPath)

	if !strings.Contains(out, "ORG acme") || !strings.Contains(out, "SEAT reviewer") {
		t.Errorf("expected explicit --receipts file to be read, got:\n%s", out)
	}
	defaultPath := org.ReceiptsPathIn(emptyStateDir)
	if strings.Contains(out, defaultPath) {
		t.Errorf("expected env-default path %q to be ignored when --receipts is explicit, got:\n%s", defaultPath, out)
	}
}

// TestInsightsCmd_ReceiptsTextContractExample pins the exact text-output
// contract from the plan (AC-3): "ORG demo  SEAT lead  commanded=opus
// honored: true=3 false=1 unknown=2  rate=75% (unknown 2 excluded)".
func TestInsightsCmd_ReceiptsTextContractExample(t *testing.T) {
	dir := t.TempDir()
	receiptsPath := filepath.Join(dir, "model-receipts.jsonl")
	store := org.NewReceiptStoreAtPath(receiptsPath)

	ts := []string{
		"2026-08-03T01:00:00Z", "2026-08-03T01:01:00Z", "2026-08-03T01:02:00Z",
		"2026-08-03T01:03:00Z", "2026-08-03T01:04:00Z", "2026-08-03T01:05:00Z",
	}
	honored := []string{org.HonoredTrue, org.HonoredTrue, org.HonoredTrue, org.HonoredFalse, org.HonoredUnknown, org.HonoredUnknown}
	for i := range ts {
		if err := store.Append(org.Receipt{
			TS:             ts[i],
			OrgID:          "demo",
			SeatID:         "lead",
			Driver:         "claude",
			CommandedModel: "opus",
			Honored:        honored[i],
		}); err != nil {
			t.Fatalf("append receipt %d: %v", i, err)
		}
	}

	eventsDir := filepath.Join(dir, "events")
	out := runInsightsCmd(t, "--events-dir", eventsDir, "--receipts", receiptsPath)

	want := "ORG demo  SEAT lead  commanded=opus  honored: true=3 false=1 unknown=2  rate=75% (unknown 2 excluded)"
	if !strings.Contains(out, want) {
		t.Errorf("expected AC-3 contract line:\n  %s\ngot:\n%s", want, out)
	}
}

// TestInsightsCmd_ReceiptsZeroMessage pins the "no org receipts found
// (path)" single-line message for a receipts file with zero lines.
func TestInsightsCmd_ReceiptsZeroMessage(t *testing.T) {
	dir := t.TempDir()
	eventsDir := filepath.Join(dir, "events")
	// At least one event so the top-level "No insight data yet" early
	// return does not mask the receipts-section message under test.
	writeTestEvent(t, eventsDir, "2026-08-03-task.jsonl",
		`{"schema":1,"ts":"2026-08-03T00:00:00Z","run_id":"r1","slug":"task","flow":"loop","phase":"verify","cycle":1,"verdict":"pass","findings":{"critical":0,"high":0,"medium":0,"low":0},"triage":{"action_required":0,"worth_considering":0,"dismissed":0},"driver":"claude","requested_model":"sonnet","effective_model":"sonnet","honored":true,"source":"pipeline"}`)

	receiptsPath := filepath.Join(dir, "missing-receipts.jsonl")
	out := runInsightsCmd(t, "--events-dir", eventsDir, "--receipts", receiptsPath)

	wantMsg := "no org receipts found (" + receiptsPath + ")"
	if !strings.Contains(out, wantMsg) {
		t.Errorf("expected %q, got:\n%s", wantMsg, out)
	}
}

// TestInsightsCmd_ReceiptsJSONSchema pins the AC-3 JSON schema for the
// receipts section: {"path":...,"orgs":[...],"skipped_lines":N}, identical
// shape whether or not any receipts were found.
func TestInsightsCmd_ReceiptsJSONSchema(t *testing.T) {
	dir := t.TempDir()
	eventsDir := filepath.Join(dir, "events")
	receiptsPath := filepath.Join(dir, "missing-receipts.jsonl")

	out := runInsightsCmd(t, "--events-dir", eventsDir, "--receipts", receiptsPath, "--json")

	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v\noutput:\n%s", err, out)
	}
	var receiptsSection map[string]json.RawMessage
	if err := json.Unmarshal(payload["receipts"], &receiptsSection); err != nil {
		t.Fatalf("json.Unmarshal receipts section: %v\nreceipts: %s", err, payload["receipts"])
	}
	for _, key := range []string{"path", "orgs", "skipped_lines"} {
		if _, ok := receiptsSection[key]; !ok {
			t.Errorf("receipts JSON missing key %q, got: %s", key, payload["receipts"])
		}
	}
	// Orgs must be an empty array, not null -- pretty-printed output
	// (enc.SetIndent) inserts whitespace, so parse rather than substring-match.
	rawOrgs := strings.TrimSpace(string(receiptsSection["orgs"]))
	if rawOrgs == "null" {
		t.Errorf("expected empty orgs array (not null), got: %s", payload["receipts"])
	}
	var orgs []json.RawMessage
	if err := json.Unmarshal(receiptsSection["orgs"], &orgs); err != nil {
		t.Fatalf("json.Unmarshal orgs: %v", err)
	}
	if len(orgs) != 0 {
		t.Errorf("orgs = %d entries, want 0", len(orgs))
	}
}

// runBackfillCmd runs "ralph insights backfill" with the given args and returns stdout.
func runBackfillCmd(t *testing.T, args ...string) string {
	t.Helper()
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(append([]string{"insights", "backfill"}, args...))
	if err := root.Execute(); err != nil {
		t.Fatalf("command error: %v (output: %s)", err, buf.String())
	}
	return buf.String()
}

func TestBackfillCmd_DryRun(t *testing.T) {
	// Use the package testdata/reports fixtures as the reports dir.
	reportsDir := filepath.Join("..", "insights", "testdata", "reports")
	eventsDir := t.TempDir()

	out := runBackfillCmd(t,
		"--reports-dir", reportsDir,
		"--events-dir", eventsDir,
	)

	if !strings.Contains(out, "dry-run") {
		t.Errorf("expected 'dry-run' in output, got:\n%s", out)
	}
	// At least some events should be derivable.
	if !strings.Contains(out, "verify") && !strings.Contains(out, "self_review") {
		t.Errorf("expected phase names in output, got:\n%s", out)
	}
	// Dry-run must not write any files.
	entries, err := os.ReadDir(eventsDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("dry-run wrote %d files, want 0", len(entries))
	}
}

func TestBackfillCmd_Apply(t *testing.T) {
	reportsDir := filepath.Join("..", "insights", "testdata", "reports")
	eventsDir := t.TempDir()

	out := runBackfillCmd(t,
		"--reports-dir", reportsDir,
		"--events-dir", eventsDir,
		"--apply",
	)

	if !strings.Contains(out, "applied") {
		t.Errorf("expected 'applied' in output, got:\n%s", out)
	}

	// At least one event file should exist.
	entries, err := os.ReadDir(eventsDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("apply wrote 0 files, expected at least 1")
	}
}

func TestBackfillCmd_Idempotent(t *testing.T) {
	reportsDir := filepath.Join("..", "insights", "testdata", "reports")
	eventsDir := t.TempDir()

	// First apply.
	runBackfillCmd(t, "--reports-dir", reportsDir, "--events-dir", eventsDir, "--apply")

	// Second apply: output must say 0 new events written.
	out := runBackfillCmd(t, "--reports-dir", reportsDir, "--events-dir", eventsDir, "--apply")
	if !strings.Contains(out, "0 new events") {
		t.Errorf("second apply: expected '0 new events', got:\n%s", out)
	}
}
