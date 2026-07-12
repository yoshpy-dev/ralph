package insights

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const testdataReports = "testdata/reports"

// --- ParseReport tests ---

func TestParseReport_SelfReview_WithFindings(t *testing.T) {
	path := filepath.Join(testdataReports, "self-review-2026-07-12-worktree-gc-exit-code.md")
	bev, err := ParseReport(path)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if bev == nil {
		t.Fatal("expected non-nil BackfillEvent")
	}
	if bev.ParseMiss {
		t.Errorf("ParseMiss = true: %s", bev.ParseMissReason)
	}
	if bev.Phase != "self_review" {
		t.Errorf("phase = %q, want self_review", bev.Phase)
	}
	if bev.Slug != "worktree-gc-exit-code" {
		t.Errorf("slug = %q, want worktree-gc-exit-code", bev.Slug)
	}
	if bev.Cycle != 1 {
		t.Errorf("cycle = %d, want 1", bev.Cycle)
	}
	if bev.Source != "backfill" {
		t.Errorf("source = %q, want backfill", bev.Source)
	}
	if bev.SourceReportPath != path {
		t.Errorf("source_report_path = %q, want %q", bev.SourceReportPath, path)
	}
	// Fixture has 1 LOW finding → verdict=pass (no CRITICAL/HIGH)
	if bev.Findings.Low != 1 {
		t.Errorf("findings.low = %d, want 1", bev.Findings.Low)
	}
	if bev.Verdict != "pass" {
		t.Errorf("verdict = %q, want pass (no critical/high)", bev.Verdict)
	}
}

func TestParseReport_SelfReview_MultipleFindings(t *testing.T) {
	// self-review-2026-07-11-loop-model-routing.md has 3 LOW findings
	path := filepath.Join(testdataReports, "self-review-2026-07-11-loop-model-routing.md")
	bev, err := ParseReport(path)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if bev == nil {
		t.Fatal("nil BackfillEvent")
	}
	if bev.ParseMiss {
		t.Errorf("ParseMiss = true: %s", bev.ParseMissReason)
	}
	if bev.Findings.Low != 3 {
		t.Errorf("findings.low = %d, want 3", bev.Findings.Low)
	}
	if bev.Verdict != "pass" {
		t.Errorf("verdict = %q, want pass", bev.Verdict)
	}
}

func TestParseReport_Verify_Pass(t *testing.T) {
	path := filepath.Join(testdataReports, "verify-2026-07-12-worktree-gc-exit-code.md")
	bev, err := ParseReport(path)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if bev == nil {
		t.Fatal("nil BackfillEvent")
	}
	if bev.ParseMiss {
		t.Errorf("ParseMiss = true: %s", bev.ParseMissReason)
	}
	if bev.Phase != "verify" {
		t.Errorf("phase = %q, want verify", bev.Phase)
	}
	if bev.Slug != "worktree-gc-exit-code" {
		t.Errorf("slug = %q, want worktree-gc-exit-code", bev.Slug)
	}
	if bev.Verdict != "pass" {
		t.Errorf("verdict = %q, want pass", bev.Verdict)
	}
}

func TestParseReport_Verify_Fail(t *testing.T) {
	path := filepath.Join(testdataReports, "verify-fail-example.md")
	bev, err := ParseReport(path)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if bev == nil {
		t.Fatal("nil BackfillEvent")
	}
	if bev.ParseMiss {
		t.Errorf("ParseMiss = true: %s", bev.ParseMissReason)
	}
	if bev.Verdict != "fail" {
		t.Errorf("verdict = %q, want fail", bev.Verdict)
	}
}

func TestParseReport_Test_Pass(t *testing.T) {
	path := filepath.Join(testdataReports, "test-2026-07-12-worktree-gc-exit-code.md")
	bev, err := ParseReport(path)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if bev == nil {
		t.Fatal("nil BackfillEvent")
	}
	if bev.ParseMiss {
		t.Errorf("ParseMiss = true: %s", bev.ParseMissReason)
	}
	if bev.Phase != "test" {
		t.Errorf("phase = %q, want test", bev.Phase)
	}
	if bev.Verdict != "pass" {
		t.Errorf("verdict = %q, want pass", bev.Verdict)
	}
}

func TestParseReport_Test_Fail(t *testing.T) {
	path := filepath.Join(testdataReports, "test-fail-example.md")
	bev, err := ParseReport(path)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if bev == nil {
		t.Fatal("nil BackfillEvent")
	}
	if bev.ParseMiss {
		t.Errorf("ParseMiss = true: %s", bev.ParseMissReason)
	}
	if bev.Verdict != "fail" {
		t.Errorf("verdict = %q, want fail", bev.Verdict)
	}
}

func TestParseReport_CrossReview_ZeroCounts(t *testing.T) {
	path := filepath.Join(testdataReports, "cross-review-triage-worktree-gc-exit-code.md")
	bev, err := ParseReport(path)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if bev == nil {
		t.Fatal("nil BackfillEvent")
	}
	if bev.ParseMiss {
		t.Errorf("ParseMiss = true: %s", bev.ParseMissReason)
	}
	if bev.Phase != "cross_review" {
		t.Errorf("phase = %q, want cross_review", bev.Phase)
	}
	if bev.Triage.ActionRequired != 0 {
		t.Errorf("triage.action_required = %d, want 0", bev.Triage.ActionRequired)
	}
	if bev.Verdict != "pass" {
		t.Errorf("verdict = %q, want pass", bev.Verdict)
	}
}

func TestParseReport_CrossReview_ActionRequired(t *testing.T) {
	path := filepath.Join(testdataReports, "cross-review-triage-loop-model-routing.md")
	bev, err := ParseReport(path)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if bev == nil {
		t.Fatal("nil BackfillEvent")
	}
	if bev.ParseMiss {
		t.Errorf("ParseMiss = true: %s", bev.ParseMissReason)
	}
	if bev.Triage.ActionRequired != 2 {
		t.Errorf("triage.action_required = %d, want 2", bev.Triage.ActionRequired)
	}
	if bev.Verdict != "action_required" {
		t.Errorf("verdict = %q, want action_required", bev.Verdict)
	}
}

func TestParseReport_UnrecognisedType(t *testing.T) {
	// walkthrough-* is not a recognised type
	dir := t.TempDir()
	path := filepath.Join(dir, "walkthrough-2026-07-12-something.md")
	if err := os.WriteFile(path, []byte("# Walkthrough\nsome content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bev, err := ParseReport(path)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if bev != nil {
		t.Errorf("expected nil for unrecognised type, got %+v", bev)
	}
}

func TestParseReport_MissingFile(t *testing.T) {
	_, err := ParseReport("/nonexistent/self-review-something.md")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// --- DedupeKey tests ---

func TestDedupeKey(t *testing.T) {
	ev := Event{
		SourceReportPath: "/docs/reports/verify-2026-07-12-foo.md",
		Phase:            "verify",
		Cycle:            1,
	}
	key := DedupeKey(ev)
	want := "/docs/reports/verify-2026-07-12-foo.md:verify:1"
	if key != want {
		t.Errorf("DedupeKey = %q, want %q", key, want)
	}
}

// --- Idempotency test (AC5) ---

func TestRunBackfill_Idempotency(t *testing.T) {
	eventsDir := t.TempDir()

	// First run with --apply.
	stats1, err := RunBackfill(testdataReports, eventsDir, true)
	if err != nil {
		t.Fatalf("first RunBackfill: %v", err)
	}
	if stats1.Written == 0 {
		t.Fatal("first run wrote 0 events, expected > 0")
	}
	t.Logf("first run: parsed=%d written=%d miss=%d dupe=%d",
		stats1.Parsed, stats1.Written, stats1.ParseMiss, stats1.Duplicate)

	// Second run: must write zero new events.
	stats2, err := RunBackfill(testdataReports, eventsDir, true)
	if err != nil {
		t.Fatalf("second RunBackfill: %v", err)
	}
	if stats2.Written != 0 {
		t.Errorf("second run wrote %d events, want 0 (idempotency)", stats2.Written)
	}
	if stats2.Duplicate != stats1.Written {
		t.Errorf("second run duplicate=%d, want %d (all first-run events should be dupes)",
			stats2.Duplicate, stats1.Written)
	}
}

// --- Multi-cycle distinct events test (AC5) ---

func TestRunBackfill_MultiCycleDistinctEvents(t *testing.T) {
	// Write two report files for the same slug but different cycle markers.
	// Since cycle is always 1 from filename, we simulate multi-cycle by using
	// two different report *types* for the same slug (each gets its own dedup key).
	reportsDir := t.TempDir()
	eventsDir := t.TempDir()

	slug := "multi-slug"

	// self-review report (phase=self_review, cycle=1)
	srContent := "# Self-review report: multi-slug\n\n## Findings\n\n| Severity | Area | Finding | Evidence | Recommendation |\n| --- | --- | --- | --- | --- |\n| HIGH | x | y | z | w |\n\n## Recommendation\n- Merge: no.\n"
	if err := os.WriteFile(filepath.Join(reportsDir, "self-review-2026-07-12-"+slug+".md"), []byte(srContent), 0644); err != nil {
		t.Fatal(err)
	}

	// verify report (phase=verify, cycle=1) — different phase → different dedup key
	vrContent := "# Verify report: multi-slug\n\n## Verdict: PASS\n\nAll criteria met.\n"
	if err := os.WriteFile(filepath.Join(reportsDir, "verify-2026-07-12-"+slug+".md"), []byte(vrContent), 0644); err != nil {
		t.Fatal(err)
	}

	stats, err := RunBackfill(reportsDir, eventsDir, true)
	if err != nil {
		t.Fatalf("RunBackfill: %v", err)
	}
	if stats.Written != 2 {
		t.Errorf("written = %d, want 2 (two distinct phase events for same slug)", stats.Written)
	}

	// Read back and verify both events are present.
	events, _, err := ReadEvents(eventsDir)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("events = %d, want 2", len(events))
	}
	phases := make(map[string]bool)
	for _, ev := range events {
		phases[ev.Phase] = true
	}
	if !phases["self_review"] {
		t.Error("missing self_review event")
	}
	if !phases["verify"] {
		t.Error("missing verify event")
	}
}

// --- AppendBackfillEvent: verify JSON structure ---

func TestAppendBackfillEvent_JSONStructure(t *testing.T) {
	dir := t.TempDir()
	ev := Event{
		Schema:           1,
		TS:               "2026-07-13T00:00:00Z",
		RunID:            "",
		Slug:             "my-task",
		Flow:             "",
		Phase:            "verify",
		Cycle:            1,
		Verdict:          "pass",
		Findings:         Findings{},
		Triage:           Triage{},
		Source:           "backfill",
		SourceReportPath: "/docs/reports/verify-2026-07-13-my-task.md",
	}

	if err := AppendBackfillEvent(dir, ev); err != nil {
		t.Fatalf("AppendBackfillEvent: %v", err)
	}

	events, _, err := ReadEvents(dir)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	got := events[0]
	if got.Source != "backfill" {
		t.Errorf("source = %q, want backfill", got.Source)
	}
	if got.SourceReportPath != ev.SourceReportPath {
		t.Errorf("source_report_path = %q, want %q", got.SourceReportPath, ev.SourceReportPath)
	}

	// Also verify the raw JSON contains source_report_path.
	filename := "2026-07-13-my-task.jsonl"
	raw, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw[:len(raw)-1], &m); err != nil { // strip trailing \n
		t.Fatalf("unmarshal raw: %v", err)
	}
	if _, ok := m["source_report_path"]; !ok {
		t.Error("JSON missing source_report_path field")
	}
}
