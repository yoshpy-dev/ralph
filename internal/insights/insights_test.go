package insights

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
//
// testdata/receipts.jsonl holds the org runtime's receipt schema: 8 valid
// lines across two orgs (demo, acme) and one corrupt line ("not valid
// json"). demo/lead has 3 true, 1 false, 2 unknown (matching the AC-3
// output-contract example: true=3 false=1 unknown=2, rate=75%).

func TestReadReceipts_ValidFile(t *testing.T) {
	path := filepath.Join("testdata", "receipts.jsonl")
	receipts, stats, err := ReadReceipts(path)
	if err != nil {
		t.Fatalf("ReadReceipts: %v", err)
	}
	// 9 lines: 8 valid, 1 corrupt.
	if len(receipts) != 8 {
		t.Errorf("receipts = %d, want 8", len(receipts))
	}
	if stats.SkippedLines != 1 {
		t.Errorf("SkippedLines = %d, want 1", stats.SkippedLines)
	}
	if stats.LinesRead != 9 {
		t.Errorf("LinesRead = %d, want 9", stats.LinesRead)
	}
	// Spot-check first receipt: org runtime tri-state schema preserved verbatim.
	r := receipts[0]
	if r.OrgID != "demo" {
		t.Errorf("org_id = %q, want demo", r.OrgID)
	}
	if r.SeatID != "lead" {
		t.Errorf("seat_id = %q, want lead", r.SeatID)
	}
	if r.CommandedModel != "opus" {
		t.Errorf("commanded_model = %q, want opus", r.CommandedModel)
	}
	if r.Honored != "true" {
		t.Errorf("honored = %q, want %q (tri-state string, not bool)", r.Honored, "true")
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

func TestReadReceipts_TriStateValuesPreserved(t *testing.T) {
	path := filepath.Join("testdata", "receipts.jsonl")
	receipts, _, err := ReadReceipts(path)
	if err != nil {
		t.Fatalf("ReadReceipts: %v", err)
	}

	var trueCount, falseCount, unknownCount int
	for _, r := range receipts {
		switch r.Honored {
		case "true":
			trueCount++
		case "false":
			falseCount++
		case "unknown":
			unknownCount++
		default:
			t.Errorf("unexpected honored value %q", r.Honored)
		}
	}
	if trueCount != 4 {
		t.Errorf("true count = %d, want 4", trueCount)
	}
	if falseCount != 1 {
		t.Errorf("false count = %d, want 1", falseCount)
	}
	if unknownCount != 3 {
		t.Errorf("unknown count = %d, want 3", unknownCount)
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

func TestAggregateReceipts_TriStateCountsAndRate(t *testing.T) {
	receipts, stats, err := ReadReceipts(filepath.Join("testdata", "receipts.jsonl"))
	if err != nil {
		t.Fatalf("ReadReceipts: %v", err)
	}

	summary := AggregateReceipts(receipts, stats, "testdata/receipts.jsonl")

	if summary.Path != "testdata/receipts.jsonl" {
		t.Errorf("Path = %q, want testdata/receipts.jsonl", summary.Path)
	}
	if summary.SkippedLines != 1 {
		t.Errorf("SkippedLines = %d, want 1", summary.SkippedLines)
	}

	var demoLead *ReceiptSeatStats
	for oi := range summary.Orgs {
		if summary.Orgs[oi].OrgID != "demo" {
			continue
		}
		for si := range summary.Orgs[oi].Seats {
			if summary.Orgs[oi].Seats[si].SeatID == "lead" {
				demoLead = &summary.Orgs[oi].Seats[si]
			}
		}
	}
	if demoLead == nil {
		t.Fatal("demo/lead seat not found in aggregated receipts")
	}

	// AC-3 output-contract example: true=3 false=1 unknown=2, rate=75%.
	if demoLead.HonoredTrue != 3 {
		t.Errorf("demo/lead HonoredTrue = %d, want 3", demoLead.HonoredTrue)
	}
	if demoLead.HonoredFalse != 1 {
		t.Errorf("demo/lead HonoredFalse = %d, want 1", demoLead.HonoredFalse)
	}
	if demoLead.HonoredUnknown != 2 {
		t.Errorf("demo/lead HonoredUnknown = %d, want 2", demoLead.HonoredUnknown)
	}
	if len(demoLead.CommandedModels) != 1 || demoLead.CommandedModels[0] != "opus" {
		t.Errorf("demo/lead CommandedModels = %v, want [opus]", demoLead.CommandedModels)
	}
	rate, ok := demoLead.HonoredRate()
	if !ok {
		t.Fatal("demo/lead HonoredRate() ok = false, want true")
	}
	if rate != 0.75 {
		t.Errorf("demo/lead HonoredRate() = %f, want 0.75 (unknown excluded from denominator)", rate)
	}
}

func TestAggregateReceipts_MultiOrgMultiSeatDeterministicOrder(t *testing.T) {
	receipts, stats, err := ReadReceipts(filepath.Join("testdata", "receipts.jsonl"))
	if err != nil {
		t.Fatalf("ReadReceipts: %v", err)
	}

	summary := AggregateReceipts(receipts, stats, "testdata/receipts.jsonl")

	if len(summary.Orgs) != 2 {
		t.Fatalf("len(Orgs) = %d, want 2 (acme, demo)", len(summary.Orgs))
	}
	// Orgs sorted by org_id: "acme" < "demo".
	if summary.Orgs[0].OrgID != "acme" {
		t.Errorf("Orgs[0].OrgID = %q, want acme", summary.Orgs[0].OrgID)
	}
	if summary.Orgs[1].OrgID != "demo" {
		t.Errorf("Orgs[1].OrgID = %q, want demo", summary.Orgs[1].OrgID)
	}
	// demo has two seats, sorted by seat_id: "lead" < "reviewer".
	demoSeats := summary.Orgs[1].Seats
	if len(demoSeats) != 2 {
		t.Fatalf("len(demo seats) = %d, want 2", len(demoSeats))
	}
	if demoSeats[0].SeatID != "lead" {
		t.Errorf("demoSeats[0].SeatID = %q, want lead", demoSeats[0].SeatID)
	}
	if demoSeats[1].SeatID != "reviewer" {
		t.Errorf("demoSeats[1].SeatID = %q, want reviewer", demoSeats[1].SeatID)
	}
}

func TestAggregateReceipts_UnknownOnlySeatHasNoRate(t *testing.T) {
	// Edge case: a seat whose every receipt is "unknown" has no true/false
	// data to compute a rate from — HonoredRate() must report ok=false
	// rather than a misleading 0%.
	receipts, stats, err := ReadReceipts(filepath.Join("testdata", "receipts.jsonl"))
	if err != nil {
		t.Fatalf("ReadReceipts: %v", err)
	}

	summary := AggregateReceipts(receipts, stats, "testdata/receipts.jsonl")

	var acmeLead *ReceiptSeatStats
	for oi := range summary.Orgs {
		if summary.Orgs[oi].OrgID != "acme" {
			continue
		}
		for si := range summary.Orgs[oi].Seats {
			if summary.Orgs[oi].Seats[si].SeatID == "lead" {
				acmeLead = &summary.Orgs[oi].Seats[si]
			}
		}
	}
	if acmeLead == nil {
		t.Fatal("acme/lead seat not found")
	}
	if acmeLead.HonoredTrue != 0 || acmeLead.HonoredFalse != 0 || acmeLead.HonoredUnknown != 1 {
		t.Errorf("acme/lead counts = true=%d false=%d unknown=%d, want 0/0/1",
			acmeLead.HonoredTrue, acmeLead.HonoredFalse, acmeLead.HonoredUnknown)
	}
	if _, ok := acmeLead.HonoredRate(); ok {
		t.Error("acme/lead HonoredRate() ok = true, want false (all receipts unknown)")
	}
}

func TestAggregateReceipts_ZeroReceipts(t *testing.T) {
	summary := AggregateReceipts(nil, ReceiptStats{}, "testdata/nonexistent-receipts.jsonl")

	if summary.Path != "testdata/nonexistent-receipts.jsonl" {
		t.Errorf("Path = %q, want testdata/nonexistent-receipts.jsonl", summary.Path)
	}
	if summary.Orgs == nil {
		t.Error("Orgs = nil, want non-nil empty slice (identical JSON shape empty vs populated)")
	}
	if len(summary.Orgs) != 0 {
		t.Errorf("len(Orgs) = %d, want 0", len(summary.Orgs))
	}
	if summary.SkippedLines != 0 {
		t.Errorf("SkippedLines = %d, want 0", summary.SkippedLines)
	}
}

func TestAggregateReceipts_JSONSchemaIdenticalEmptyVsPopulated(t *testing.T) {
	empty := AggregateReceipts(nil, ReceiptStats{}, "path/a.jsonl")
	emptyJSON, err := json.Marshal(empty)
	if err != nil {
		t.Fatalf("marshal empty: %v", err)
	}
	if !strings.Contains(string(emptyJSON), `"orgs":[]`) {
		t.Errorf("empty JSON = %s, want \"orgs\":[] (not null)", emptyJSON)
	}

	receipts, stats, err := ReadReceipts(filepath.Join("testdata", "receipts.jsonl"))
	if err != nil {
		t.Fatalf("ReadReceipts: %v", err)
	}
	populated := AggregateReceipts(receipts, stats, "path/b.jsonl")
	populatedJSON, err := json.Marshal(populated)
	if err != nil {
		t.Fatalf("marshal populated: %v", err)
	}

	// Both must have the same top-level keys.
	var emptyMap, populatedMap map[string]any
	if err := json.Unmarshal(emptyJSON, &emptyMap); err != nil {
		t.Fatalf("unmarshal empty: %v", err)
	}
	if err := json.Unmarshal(populatedJSON, &populatedMap); err != nil {
		t.Fatalf("unmarshal populated: %v", err)
	}
	for _, key := range []string{"path", "orgs", "skipped_lines"} {
		if _, ok := emptyMap[key]; !ok {
			t.Errorf("empty JSON missing key %q", key)
		}
		if _, ok := populatedMap[key]; !ok {
			t.Errorf("populated JSON missing key %q", key)
		}
	}
}

func TestFindings_Total(t *testing.T) {
	f := Findings{Critical: 1, High: 2, Medium: 3, Low: 4}
	if f.Total() != 10 {
		t.Errorf("Total() = %d, want 10", f.Total())
	}
}
