package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/insights"
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
