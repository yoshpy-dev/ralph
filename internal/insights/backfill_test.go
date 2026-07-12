package insights

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const testdataReports = "testdata/reports"

// firstEvent is a helper that asserts ParseReport returns exactly one event
// and returns it. Used for single-cycle report types (self_review, verify, test)
// and single-cycle cross-review fixtures.
func firstEvent(t *testing.T, path string) BackfillEvent {
	t.Helper()
	bevs, err := ParseReport(path)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if len(bevs) == 0 {
		t.Fatal("expected at least one BackfillEvent, got none")
	}
	if len(bevs) != 1 {
		t.Fatalf("expected exactly 1 BackfillEvent, got %d", len(bevs))
	}
	return bevs[0]
}

// --- ParseReport tests ---

func TestParseReport_SelfReview_WithFindings(t *testing.T) {
	path := filepath.Join(testdataReports, "self-review-2026-07-12-worktree-gc-exit-code.md")
	bev := firstEvent(t, path)

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
	// SourceReportPath must be the absolute form of path (Fix 3: filepath.Abs).
	absPath, _ := filepath.Abs(path)
	if bev.SourceReportPath != absPath {
		t.Errorf("source_report_path = %q, want %q", bev.SourceReportPath, absPath)
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
	bev := firstEvent(t, path)

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
	bev := firstEvent(t, path)

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
	bev := firstEvent(t, path)

	if bev.ParseMiss {
		t.Errorf("ParseMiss = true: %s", bev.ParseMissReason)
	}
	if bev.Verdict != "fail" {
		t.Errorf("verdict = %q, want fail", bev.Verdict)
	}
}

func TestParseReport_Test_Pass(t *testing.T) {
	path := filepath.Join(testdataReports, "test-2026-07-12-worktree-gc-exit-code.md")
	bev := firstEvent(t, path)

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
	bev := firstEvent(t, path)

	if bev.ParseMiss {
		t.Errorf("ParseMiss = true: %s", bev.ParseMissReason)
	}
	if bev.Verdict != "fail" {
		t.Errorf("verdict = %q, want fail", bev.Verdict)
	}
}

func TestParseReport_CrossReview_ZeroCounts(t *testing.T) {
	path := filepath.Join(testdataReports, "cross-review-triage-worktree-gc-exit-code.md")
	bev := firstEvent(t, path)

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

// TestParseReport_CrossReview_MultiCycle verifies that a two-cycle cross-review
// report (cycle-1 ACTION_REQUIRED, cycle-2 pass) yields two distinct BackfillEvents
// with the correct triage counts per cycle.
//
// The fixture cross-review-triage-loop-model-routing.md has:
//   - Line 9:  "After triage: ACTION_REQUIRED=2, ..." (no preceding ## Cycle heading → cycle 1)
//   - Line 51: "## Cycle 2 (2026-07-11)"            (explicit heading → pendingCycle=2)
//   - Line 63: "After triage: ACTION_REQUIRED=0, ..." (uses pendingCycle=2)
func TestParseReport_CrossReview_MultiCycle(t *testing.T) {
	path := filepath.Join(testdataReports, "cross-review-triage-loop-model-routing.md")
	bevs, err := ParseReport(path)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if len(bevs) != 2 {
		t.Fatalf("expected 2 BackfillEvents (one per cycle), got %d", len(bevs))
	}

	// Cycle 1: ACTION_REQUIRED=2 → verdict=action_required
	c1 := bevs[0]
	if c1.ParseMiss {
		t.Errorf("cycle 1 ParseMiss = true: %s", c1.ParseMissReason)
	}
	if c1.Cycle != 1 {
		t.Errorf("cycle 1 Cycle = %d, want 1", c1.Cycle)
	}
	if c1.Triage.ActionRequired != 2 {
		t.Errorf("cycle 1 triage.action_required = %d, want 2", c1.Triage.ActionRequired)
	}
	if c1.Verdict != "action_required" {
		t.Errorf("cycle 1 verdict = %q, want action_required", c1.Verdict)
	}

	// Cycle 2: ACTION_REQUIRED=0 → verdict=pass
	c2 := bevs[1]
	if c2.ParseMiss {
		t.Errorf("cycle 2 ParseMiss = true: %s", c2.ParseMissReason)
	}
	if c2.Cycle != 2 {
		t.Errorf("cycle 2 Cycle = %d, want 2", c2.Cycle)
	}
	if c2.Triage.ActionRequired != 0 {
		t.Errorf("cycle 2 triage.action_required = %d, want 0", c2.Triage.ActionRequired)
	}
	if c2.Verdict != "pass" {
		t.Errorf("cycle 2 verdict = %q, want pass", c2.Verdict)
	}

	// Both events must share the same slug and phase.
	if c1.Slug != c2.Slug {
		t.Errorf("slug mismatch: cycle1=%q cycle2=%q", c1.Slug, c2.Slug)
	}
	if c1.Phase != "cross_review" || c2.Phase != "cross_review" {
		t.Errorf("phase mismatch: cycle1=%q cycle2=%q", c1.Phase, c2.Phase)
	}

	// DedupeKeys must be distinct (different cycle suffix).
	k1 := DedupeKey(c1.Event)
	k2 := DedupeKey(c2.Event)
	if k1 == k2 {
		t.Errorf("DedupeKey collision: both = %q", k1)
	}
}

func TestParseReport_UnrecognisedType(t *testing.T) {
	// walkthrough-* is not a recognised type
	dir := t.TempDir()
	path := filepath.Join(dir, "walkthrough-2026-07-12-something.md")
	if err := os.WriteFile(path, []byte("# Walkthrough\nsome content\n"), 0644); err != nil {
		t.Fatal(err)
	}
	bevs, err := ParseReport(path)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if bevs != nil {
		t.Errorf("expected nil for unrecognised type, got %+v", bevs)
	}
}

func TestParseReport_MissingFile(t *testing.T) {
	_, err := ParseReport("/nonexistent/self-review-something.md")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// TestParseReport_PathNormalization verifies that a relative path and the
// corresponding absolute path yield events with the same SourceReportPath
// (the absolute form), so their DedupeKeys are identical. (Fix 3 regression)
func TestParseReport_PathNormalization(t *testing.T) {
	absPath, err := filepath.Abs(filepath.Join(testdataReports, "verify-2026-07-12-worktree-gc-exit-code.md"))
	if err != nil {
		t.Fatal(err)
	}

	// Parse with relative path (cwd is package dir when running go test).
	relPath := filepath.Join(testdataReports, "verify-2026-07-12-worktree-gc-exit-code.md")
	bevsRel, err := ParseReport(relPath)
	if err != nil {
		t.Fatalf("ParseReport(relative): %v", err)
	}
	if len(bevsRel) == 0 {
		t.Fatal("no events from relative path")
	}

	// Parse with absolute path.
	bevsAbs, err := ParseReport(absPath)
	if err != nil {
		t.Fatalf("ParseReport(absolute): %v", err)
	}
	if len(bevsAbs) == 0 {
		t.Fatal("no events from absolute path")
	}

	// Both must yield the same SourceReportPath (absolute).
	if bevsRel[0].SourceReportPath != absPath {
		t.Errorf("relative parse SourceReportPath = %q, want %q", bevsRel[0].SourceReportPath, absPath)
	}
	if bevsAbs[0].SourceReportPath != absPath {
		t.Errorf("absolute parse SourceReportPath = %q, want %q", bevsAbs[0].SourceReportPath, absPath)
	}

	// DedupeKeys must be identical.
	kRel := DedupeKey(bevsRel[0].Event)
	kAbs := DedupeKey(bevsAbs[0].Event)
	if kRel != kAbs {
		t.Errorf("DedupeKey mismatch: rel=%q abs=%q", kRel, kAbs)
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

// TestRunBackfill_RelThenAbsDedupe verifies that applying backfill with a
// relative reports-dir and then again with the absolute form of the same dir
// produces zero new events on the second run. (Fix 3 regression test)
func TestRunBackfill_RelThenAbsDedupe(t *testing.T) {
	eventsDir := t.TempDir()

	// Use testdata/reports as a relative path (tests run with cwd = package dir).
	relDir := testdataReports
	absDir, err := filepath.Abs(relDir)
	if err != nil {
		t.Fatal(err)
	}

	// First run with relative path.
	stats1, err := RunBackfill(relDir, eventsDir, true)
	if err != nil {
		t.Fatalf("first RunBackfill (relative): %v", err)
	}
	if stats1.Written == 0 {
		t.Fatal("first run wrote 0 events, expected > 0")
	}
	t.Logf("first run (relative): written=%d", stats1.Written)

	// Second run with absolute path — must write zero new events.
	stats2, err := RunBackfill(absDir, eventsDir, true)
	if err != nil {
		t.Fatalf("second RunBackfill (absolute): %v", err)
	}
	if stats2.Written != 0 {
		t.Errorf("second run (absolute) wrote %d new events, want 0 (rel-then-abs dedupe)", stats2.Written)
	}
	t.Logf("second run (absolute): duplicate=%d written=%d", stats2.Duplicate, stats2.Written)
}

// TestRunBackfill_MultiCycleReport verifies that a cross-review report with
// two cycles (cycle-1 action_required, cycle-2 pass) yields two distinct
// events with correct triage counts, and that a second apply run adds zero
// new events (idempotent per cycle).
func TestRunBackfill_MultiCycleReport(t *testing.T) {
	reportsDir := t.TempDir()
	eventsDir := t.TempDir()

	// Write a two-cycle cross-review-triage fixture.
	content := `# Cross-review triage report: multi-cycle-slug

- Date: 2026-07-13
- After triage: ACTION_REQUIRED=3, WORTH_CONSIDERING=1, DISMISSED=0

## ACTION_REQUIRED

| # | Finding | Rationale | Files |
|---|---------|-----------|-------|
| 1 | Bug A   | Verified  | foo.go |

## Cycle 2 (2026-07-13)

- Fixes applied.
- After triage: ACTION_REQUIRED=0, WORTH_CONSIDERING=0, DISMISSED=0
`
	fixturePath := filepath.Join(reportsDir, "cross-review-triage-multi-cycle-slug.md")
	if err := os.WriteFile(fixturePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// First apply.
	stats1, err := RunBackfill(reportsDir, eventsDir, true)
	if err != nil {
		t.Fatalf("first RunBackfill: %v", err)
	}
	if stats1.Written != 2 {
		t.Errorf("first run written = %d, want 2 (one per cycle)", stats1.Written)
	}

	// Read back events and verify per-cycle correctness.
	events, _, err := ReadEvents(eventsDir)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}

	byC := make(map[int]Event)
	for _, ev := range events {
		byC[ev.Cycle] = ev
	}

	c1, ok1 := byC[1]
	c2, ok2 := byC[2]
	if !ok1 || !ok2 {
		t.Fatalf("missing events: cycles present = %v", func() []int {
			var cs []int
			for k := range byC {
				cs = append(cs, k)
			}
			return cs
		}())
	}

	// Cycle 1: ACTION_REQUIRED=3 → verdict=action_required
	if c1.Triage.ActionRequired != 3 {
		t.Errorf("cycle1 action_required = %d, want 3", c1.Triage.ActionRequired)
	}
	if c1.Verdict != "action_required" {
		t.Errorf("cycle1 verdict = %q, want action_required", c1.Verdict)
	}

	// Cycle 2: ACTION_REQUIRED=0 → verdict=pass
	if c2.Triage.ActionRequired != 0 {
		t.Errorf("cycle2 action_required = %d, want 0", c2.Triage.ActionRequired)
	}
	if c2.Verdict != "pass" {
		t.Errorf("cycle2 verdict = %q, want pass", c2.Verdict)
	}

	// Second apply must write zero new events (idempotent per cycle).
	stats2, err := RunBackfill(reportsDir, eventsDir, true)
	if err != nil {
		t.Fatalf("second RunBackfill: %v", err)
	}
	if stats2.Written != 0 {
		t.Errorf("second run written = %d, want 0 (idempotent)", stats2.Written)
	}
	if stats2.Duplicate != 2 {
		t.Errorf("second run duplicate = %d, want 2", stats2.Duplicate)
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
