package insights

import (
	"os"
	"path/filepath"
	"testing"
)

// --- ReadEvents tests ---

func TestReadEvents_ValidFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join("testdata", "valid_events.jsonl")
	dst := filepath.Join(dir, "2026-07-13-test-task.jsonl")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatal(err)
	}

	events, stats, err := ReadEvents(dir)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	if len(events) != 4 {
		t.Errorf("events = %d, want 4", len(events))
	}
	if stats.FilesRead != 1 {
		t.Errorf("FilesRead = %d, want 1", stats.FilesRead)
	}
	if stats.SkippedLines != 0 {
		t.Errorf("SkippedLines = %d, want 0", stats.SkippedLines)
	}
	if stats.LinesRead != 4 {
		t.Errorf("LinesRead = %d, want 4", stats.LinesRead)
	}
	// Spot-check first event.
	ev := events[0]
	if ev.Phase != "self_review" {
		t.Errorf("phase = %q, want self_review", ev.Phase)
	}
	if ev.Verdict != "pass" {
		t.Errorf("verdict = %q, want pass", ev.Verdict)
	}
	if ev.Findings.Medium != 1 {
		t.Errorf("findings.medium = %d, want 1", ev.Findings.Medium)
	}
}

func TestReadEvents_CorruptLines(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join("testdata", "corrupt_lines.jsonl")
	dst := filepath.Join(dir, "2026-07-13-corrupt-task.jsonl")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatal(err)
	}

	events, stats, err := ReadEvents(dir)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	// 4 lines total, 2 are corrupt.
	if len(events) != 2 {
		t.Errorf("events = %d, want 2", len(events))
	}
	if stats.SkippedLines != 2 {
		t.Errorf("SkippedLines = %d, want 2", stats.SkippedLines)
	}
}

func TestReadEvents_MissingDir(t *testing.T) {
	events, stats, err := ReadEvents("/nonexistent/dir/that/does/not/exist")
	if err != nil {
		t.Fatalf("ReadEvents on missing dir: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("events = %d, want 0", len(events))
	}
	if stats.FilesRead != 0 {
		t.Errorf("FilesRead = %d, want 0", stats.FilesRead)
	}
}

func TestReadEvents_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	events, stats, err := ReadEvents(dir)
	if err != nil {
		t.Fatalf("ReadEvents on empty dir: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("events = %d, want 0", len(events))
	}
	if stats.FilesRead != 0 {
		t.Errorf("FilesRead = %d, want 0", stats.FilesRead)
	}
}

func TestReadEvents_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"valid_events.jsonl", "corrupt_lines.jsonl"} {
		src := filepath.Join("testdata", name)
		dst := filepath.Join(dir, name)
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	events, stats, err := ReadEvents(dir)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	// valid: 4 events, corrupt: 2 valid + 2 skipped
	if len(events) != 6 {
		t.Errorf("events = %d, want 6", len(events))
	}
	if stats.FilesRead != 2 {
		t.Errorf("FilesRead = %d, want 2", stats.FilesRead)
	}
	if stats.SkippedLines != 2 {
		t.Errorf("SkippedLines = %d, want 2", stats.SkippedLines)
	}
}

// --- ReadReceipts tests ---

func TestReadReceipts_ValidFile(t *testing.T) {
	path := filepath.Join("testdata", "receipts.jsonl")
	receipts, stats, err := ReadReceipts(path)
	if err != nil {
		t.Fatalf("ReadReceipts: %v", err)
	}
	// 5 lines: 4 valid, 1 corrupt
	if len(receipts) != 4 {
		t.Errorf("receipts = %d, want 4", len(receipts))
	}
	if stats.SkippedLines != 1 {
		t.Errorf("SkippedLines = %d, want 1", stats.SkippedLines)
	}
	// Spot-check first receipt.
	r := receipts[0]
	if r.Phase != "self_review" {
		t.Errorf("phase = %q, want self_review", r.Phase)
	}
	if !r.Honored {
		t.Error("honored = false, want true")
	}
}

func TestReadReceipts_MissingFile(t *testing.T) {
	receipts, stats, err := ReadReceipts("/nonexistent/receipts.jsonl")
	if err != nil {
		t.Fatalf("ReadReceipts on missing file: %v", err)
	}
	if len(receipts) != 0 {
		t.Errorf("receipts = %d, want 0", len(receipts))
	}
	if stats.LinesRead != 0 {
		t.Errorf("LinesRead = %d, want 0", stats.LinesRead)
	}
}

// --- Aggregate tests ---

func TestAggregate_BasicPhaseGrouping(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join("testdata", "valid_events.jsonl")
	dst := filepath.Join(dir, "2026-07-13-test-task.jsonl")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatal(err)
	}

	events, stats, err := ReadEvents(dir)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}

	agg := Aggregate(events, stats)

	if agg.TotalEvents != 4 {
		t.Errorf("TotalEvents = %d, want 4", agg.TotalEvents)
	}
	if len(agg.PerPhase) != 4 {
		t.Errorf("len(PerPhase) = %d, want 4", len(agg.PerPhase))
	}

	// self_review: pass=1
	sr := agg.PerPhase["self_review"]
	if sr == nil {
		t.Fatal("PerPhase[self_review] = nil")
	}
	if sr.Verdicts.Pass != 1 {
		t.Errorf("self_review.pass = %d, want 1", sr.Verdicts.Pass)
	}
	if sr.Findings.Medium != 1 {
		t.Errorf("self_review.findings.medium = %d, want 1", sr.Findings.Medium)
	}
	if sr.Findings.Low != 2 {
		t.Errorf("self_review.findings.low = %d, want 2", sr.Findings.Low)
	}

	// cross_review: action_required=1, triage AR=1, dismissed=2
	cr := agg.PerPhase["cross_review"]
	if cr == nil {
		t.Fatal("PerPhase[cross_review] = nil")
	}
	if cr.Verdicts.ActionRequired != 1 {
		t.Errorf("cross_review.action_required = %d, want 1", cr.Verdicts.ActionRequired)
	}
	if cr.Triage.ActionRequired != 1 {
		t.Errorf("cross_review.triage.action_required = %d, want 1", cr.Triage.ActionRequired)
	}
	if cr.Triage.Dismissed != 2 {
		t.Errorf("cross_review.triage.dismissed = %d, want 2", cr.Triage.Dismissed)
	}
}

func TestAggregate_HonoredRate(t *testing.T) {
	// All events in valid_events.jsonl have honored=true.
	dir := t.TempDir()
	src := filepath.Join("testdata", "valid_events.jsonl")
	dst := filepath.Join(dir, "events.jsonl")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatal(err)
	}

	events, stats, err := ReadEvents(dir)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	agg := Aggregate(events, stats)

	for phase, ps := range agg.PerPhase {
		if ps.HonoredRate != 1.0 {
			t.Errorf("phase %s: honored_rate = %f, want 1.0", phase, ps.HonoredRate)
		}
	}
}

func TestAggregate_Escalation(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join("testdata", "multi_cycle.jsonl")
	dst := filepath.Join(dir, "2026-07-13-escalation-task.jsonl")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatal(err)
	}

	events, stats, err := ReadEvents(dir)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	agg := Aggregate(events, stats)

	if len(agg.Escalations) != 1 {
		t.Fatalf("escalations = %d, want 1", len(agg.Escalations))
	}
	esc := agg.Escalations[0]
	if esc.Slug != "escalation-task" {
		t.Errorf("slug = %q, want escalation-task", esc.Slug)
	}
	if esc.MaxCycle != 2 {
		t.Errorf("MaxCycle = %d, want 2", esc.MaxCycle)
	}
	// Cycle 1: verify=fail, test=fail
	if esc.Cycle1Verdicts["verify"] != "fail" {
		t.Errorf("cycle1 verify = %q, want fail", esc.Cycle1Verdicts["verify"])
	}
	if esc.Cycle1Verdicts["test"] != "fail" {
		t.Errorf("cycle1 test = %q, want fail", esc.Cycle1Verdicts["test"])
	}
	// Cycle 2 (final): verify=pass, test=pass
	if esc.FinalVerdicts["verify"] != "pass" {
		t.Errorf("final verify = %q, want pass", esc.FinalVerdicts["verify"])
	}
	if esc.FinalVerdicts["test"] != "pass" {
		t.Errorf("final test = %q, want pass", esc.FinalVerdicts["test"])
	}
}

func TestAggregate_NoEscalationWhenSingleCycle(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join("testdata", "valid_events.jsonl")
	dst := filepath.Join(dir, "events.jsonl")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatal(err)
	}

	events, stats, err := ReadEvents(dir)
	if err != nil {
		t.Fatalf("ReadEvents: %v", err)
	}
	agg := Aggregate(events, stats)

	if len(agg.Escalations) != 0 {
		t.Errorf("escalations = %d, want 0 (all events at cycle 1)", len(agg.Escalations))
	}
}

func TestAggregateWithReceipts(t *testing.T) {
	// Start with empty aggregate result.
	agg := Aggregate(nil, ReadStats{})

	receipts, rStats, err := ReadReceipts(filepath.Join("testdata", "receipts.jsonl"))
	if err != nil {
		t.Fatalf("ReadReceipts: %v", err)
	}

	AggregateWithReceipts(agg, receipts, rStats)

	diag := agg.Receipts
	if !diag.Present {
		t.Error("Receipts.Present = false, want true")
	}
	if diag.TotalCount != 4 {
		t.Errorf("TotalCount = %d, want 4", diag.TotalCount)
	}
	if diag.SkippedLines != 1 {
		t.Errorf("SkippedLines = %d, want 1", diag.SkippedLines)
	}
	// 3 out of 4 are honored.
	want := 3.0 / 4.0
	if diag.HonoredRate != want {
		t.Errorf("HonoredRate = %f, want %f", diag.HonoredRate, want)
	}
	// self_review: honored
	if diag.PerPhase["self_review"] != 1.0 {
		t.Errorf("PerPhase[self_review] = %f, want 1.0", diag.PerPhase["self_review"])
	}
	// test: not honored (codex driver)
	if diag.PerPhase["test"] != 0.0 {
		t.Errorf("PerPhase[test] = %f, want 0.0", diag.PerPhase["test"])
	}
}

func TestAggregateWithReceipts_Absent(t *testing.T) {
	agg := Aggregate(nil, ReadStats{})
	// Do NOT call AggregateWithReceipts — simulate absent receipts file.
	if agg.Receipts.Present {
		t.Error("Receipts.Present = true for zero aggregate, want false")
	}
}

func TestFindings_Total(t *testing.T) {
	f := Findings{Critical: 1, High: 2, Medium: 3, Low: 4}
	if f.Total() != 10 {
		t.Errorf("Total() = %d, want 10", f.Total())
	}
}
