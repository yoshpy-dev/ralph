package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeOrchestratorJSON creates .harness/state/orchestrator/orchestrator.json
// in dir with the given status and started values, then sets its mtime to mtime.
func writeOrchestratorJSON(t *testing.T, dir, status, started string, mtime time.Time) {
	t.Helper()
	orchDir := filepath.Join(dir, ".harness", "state", "orchestrator")
	if err := os.MkdirAll(orchDir, 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(map[string]string{
		"status":  status,
		"started": started,
	})
	if err != nil {
		t.Fatal(err)
	}
	orchPath := filepath.Join(orchDir, "orchestrator.json")
	if err := os.WriteFile(orchPath, data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(orchPath, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

// TestCheckStaleOrchestratorState_StaleRunning verifies that a running
// orchestrator state whose file has not been touched for more than 24 hours
// produces a WARN result with a message mentioning the started time.
func TestCheckStaleOrchestratorState_StaleRunning(t *testing.T) {
	dir := t.TempDir()
	started := "2026-05-13T10:00:00Z"
	staleTime := time.Now().Add(-25 * time.Hour)
	writeOrchestratorJSON(t, dir, "running", started, staleTime)

	r := checkStaleOrchestratorState(dir)

	if r.Status != "warn" {
		t.Errorf("status = %q, want warn (stale running state)", r.Status)
	}
	if !strings.Contains(r.Detail, started) {
		t.Errorf("detail %q should mention started time %q", r.Detail, started)
	}
	if !strings.Contains(r.Detail, "stale orchestrator state") {
		t.Errorf("detail %q should mention 'stale orchestrator state'", r.Detail)
	}
}

// TestCheckStaleOrchestratorState_FreshRunning verifies that a running
// orchestrator state whose file was updated within the last 24 hours
// produces a pass result (live run in progress).
func TestCheckStaleOrchestratorState_FreshRunning(t *testing.T) {
	dir := t.TempDir()
	freshTime := time.Now().Add(-1 * time.Hour)
	writeOrchestratorJSON(t, dir, "running", "2026-07-13T10:00:00Z", freshTime)

	r := checkStaleOrchestratorState(dir)

	if r.Status != "pass" {
		t.Errorf("status = %q, want pass (fresh running state, likely live run)", r.Status)
	}
}

// TestCheckStaleOrchestratorState_Completed verifies that a completed
// orchestrator state (status != "running") does not trigger a warning even
// if the file is very old.
func TestCheckStaleOrchestratorState_Completed(t *testing.T) {
	dir := t.TempDir()
	staleTime := time.Now().Add(-72 * time.Hour)
	writeOrchestratorJSON(t, dir, "complete", "2026-05-01T08:00:00Z", staleTime)

	r := checkStaleOrchestratorState(dir)

	if r.Status != "pass" {
		t.Errorf("status = %q, want pass (non-running status should not warn)", r.Status)
	}
}

// TestCheckStaleOrchestratorState_MissingFile verifies that the absence of
// orchestrator.json produces a pass result (nothing to report).
func TestCheckStaleOrchestratorState_MissingFile(t *testing.T) {
	dir := t.TempDir()
	// No orchestrator.json created.

	r := checkStaleOrchestratorState(dir)

	if r.Status != "pass" {
		t.Errorf("status = %q, want pass (missing file means no active run)", r.Status)
	}
}
