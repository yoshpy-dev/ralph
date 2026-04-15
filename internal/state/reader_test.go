package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadOrchestratorState(t *testing.T) {
	orchDir := filepath.Join("testdata", "orchestrator")

	st, err := ReadOrchestratorState(orchDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Plan != "docs/plans/active/2026-04-15-ralph-tui/_manifest.md" {
		t.Errorf("plan = %q, want docs/plans/active/2026-04-15-ralph-tui/_manifest.md", st.Plan)
	}
	if st.Status != "running" {
		t.Errorf("status = %q, want running", st.Status)
	}
	if st.Started != "2026-04-15T05:30:00Z" {
		t.Errorf("started = %q, want 2026-04-15T05:30:00Z", st.Started)
	}
	if st.MaxParallel != 2 {
		t.Errorf("max_parallel = %d, want 2", st.MaxParallel)
	}
	if st.MaxIterations != 10 {
		t.Errorf("max_iterations = %d, want 10", st.MaxIterations)
	}
	if !st.UnifiedPR {
		t.Error("unified_pr = false, want true")
	}
}

func TestReadOrchestratorState_MissingFile(t *testing.T) {
	_, err := ReadOrchestratorState("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestReadOrchestratorState_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "orchestrator.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadOrchestratorState(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestReadSliceStatuses(t *testing.T) {
	orchDir := filepath.Join("testdata", "orchestrator")

	statuses, err := ReadSliceStatuses(orchDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("got %d statuses, want 2", len(statuses))
	}
	if statuses["1-slice-a"] != "complete" {
		t.Errorf("slice-a status = %q, want complete", statuses["1-slice-a"])
	}
	if statuses["2-slice-b"] != "running" {
		t.Errorf("slice-b status = %q, want running", statuses["2-slice-b"])
	}
}

func TestReadSliceStatuses_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	statuses, err := ReadSliceStatuses(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 0 {
		t.Errorf("got %d statuses, want 0", len(statuses))
	}
}

func TestReadPipelineCheckpoint(t *testing.T) {
	wtBase := filepath.Join("testdata", "worktrees")

	t.Run("complete slice", func(t *testing.T) {
		cp, err := ReadPipelineCheckpoint(wtBase, "1-slice-a")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cp.Phase != "done" {
			t.Errorf("phase = %q, want done", cp.Phase)
		}
		if cp.Status != "complete" {
			t.Errorf("status = %q, want complete", cp.Status)
		}
		if cp.InnerCycle != 3 {
			t.Errorf("inner_cycle = %d, want 3", cp.InnerCycle)
		}
		if cp.OuterCycle != 1 {
			t.Errorf("outer_cycle = %d, want 1", cp.OuterCycle)
		}
		if cp.LastTestResult == nil || *cp.LastTestResult != "pass" {
			t.Errorf("last_test_result = %v, want pass", cp.LastTestResult)
		}
		if cp.CodexTriage.WorthConsidering != 1 {
			t.Errorf("codex_triage.worth_considering = %d, want 1", cp.CodexTriage.WorthConsidering)
		}
		if !cp.PRCreated {
			t.Error("pr_created = false, want true")
		}
		if cp.PRUrl == nil || *cp.PRUrl != "https://github.com/example/repo/pull/42" {
			t.Errorf("pr_url = %v, want https://github.com/example/repo/pull/42", cp.PRUrl)
		}
		if len(cp.PhaseTransitions) != 2 {
			t.Errorf("phase_transitions count = %d, want 2", len(cp.PhaseTransitions))
		}
		if len(cp.AcceptanceCriteriaMet) != 2 {
			t.Errorf("acceptance_criteria_met count = %d, want 2", len(cp.AcceptanceCriteriaMet))
		}
	})

	t.Run("running slice with nulls", func(t *testing.T) {
		cp, err := ReadPipelineCheckpoint(wtBase, "2-slice-b")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cp.Phase != "inner" {
			t.Errorf("phase = %q, want inner", cp.Phase)
		}
		if cp.LastTestResult != nil {
			t.Errorf("last_test_result = %v, want nil", cp.LastTestResult)
		}
		if cp.SelfReviewResult != nil {
			t.Errorf("self_review_result = %v, want nil", cp.SelfReviewResult)
		}
		if cp.SessionID != nil {
			t.Errorf("session_id = %v, want nil", cp.SessionID)
		}
		if cp.PRUrl != nil {
			t.Errorf("pr_url = %v, want nil", cp.PRUrl)
		}
	})
}

func TestReadPipelineCheckpoint_MissingFile(t *testing.T) {
	_, err := ReadPipelineCheckpoint("/nonexistent", "slice-x")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestReadPipelineCheckpoint_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	cpDir := filepath.Join(dir, "bad-slice", ".harness", "state", "pipeline")
	if err := os.MkdirAll(cpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cpDir, "checkpoint.json"), []byte("not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadPipelineCheckpoint(dir, "bad-slice")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestReadSliceDependencies(t *testing.T) {
	planDir := filepath.Join("testdata", "plan")

	deps, err := ReadSliceDependencies(planDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expected deps from the test manifest:
	// slice-1 → slice-2
	// slice-1 → slice-3
	// slice-3 → slice-4
	// slice-2 → slice-5
	// slice-4 → slice-5
	if len(deps) != 5 {
		t.Fatalf("got %d dependencies, want 5; deps = %+v", len(deps), deps)
	}

	// Verify at least the first two
	found12 := false
	found13 := false
	for _, d := range deps {
		if d.From == "1" && d.To == "2" {
			found12 = true
		}
		if d.From == "1" && d.To == "3" {
			found13 = true
		}
	}
	if !found12 {
		t.Error("missing dependency 1 → 2")
	}
	if !found13 {
		t.Error("missing dependency 1 → 3")
	}
}

func TestReadSliceDependencies_MissingManifest(t *testing.T) {
	_, err := ReadSliceDependencies("/nonexistent/plan")
	if err == nil {
		t.Fatal("expected error for missing manifest, got nil")
	}
}

func TestReadFullStatus(t *testing.T) {
	orchDir := filepath.Join("testdata", "orchestrator")
	wtBase := filepath.Join("testdata", "worktrees")
	planDir := filepath.Join("testdata", "plan")

	fs, err := ReadFullStatus(orchDir, wtBase, planDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fs.Status != "running" {
		t.Errorf("status = %q, want running", fs.Status)
	}
	if fs.Progress.Total != 2 {
		t.Errorf("total = %d, want 2", fs.Progress.Total)
	}
	if fs.Progress.Completed != 1 {
		t.Errorf("completed = %d, want 1", fs.Progress.Completed)
	}
	if fs.Progress.Percent != 50 {
		t.Errorf("percent = %d, want 50", fs.Progress.Percent)
	}
	if len(fs.Slices) != 2 {
		t.Errorf("slices count = %d, want 2", len(fs.Slices))
	}
	if len(fs.Dependencies) != 5 {
		t.Errorf("dependencies count = %d, want 5", len(fs.Dependencies))
	}
	if len(fs.Checkpoints) != 2 {
		t.Errorf("checkpoints count = %d, want 2", len(fs.Checkpoints))
	}
}

func TestReadFullStatus_MissingOrchestrator(t *testing.T) {
	_, err := ReadFullStatus("/nonexistent", "/nonexistent", "/nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExtractSliceName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"slice-1-foo.status", "1-foo"},
		{"slice-2-bar-baz.status", "2-bar-baz"},
		{"slice-abc.status", "abc"},
	}
	for _, tc := range tests {
		got := extractSliceName(tc.input)
		if got != tc.want {
			t.Errorf("extractSliceName(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSplitDependencyLine(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"slice-1 (foundation) ──→ slice-2 (watcher)", 1},
		{"slice-2, slice-4 ──→ slice-5 (integration)", 2},
		{"no arrow here", 1},
		{"", 1},
	}
	for _, tc := range tests {
		got := splitDependencyLine(tc.input)
		if len(got) != tc.want {
			t.Errorf("splitDependencyLine(%q) = %d parts, want %d; parts = %v", tc.input, len(got), tc.want, got)
		}
	}
}

func TestOrchestratorState_Elapsed(t *testing.T) {
	t.Run("empty started", func(t *testing.T) {
		st := OrchestratorState{}
		if d := st.Elapsed(); d != 0 {
			t.Errorf("Elapsed() = %v, want 0", d)
		}
	})

	t.Run("invalid timestamp", func(t *testing.T) {
		st := OrchestratorState{Started: "not-a-timestamp"}
		if d := st.Elapsed(); d != 0 {
			t.Errorf("Elapsed() = %v, want 0", d)
		}
	})

	t.Run("valid timestamp", func(t *testing.T) {
		st := OrchestratorState{Started: "2020-01-01T00:00:00Z"}
		d := st.Elapsed()
		if d <= 0 {
			t.Errorf("Elapsed() = %v, want > 0", d)
		}
	})
}

func TestReadFullStatus_MissingDependenciesGraceful(t *testing.T) {
	// FullStatus should succeed even if plan manifest is missing
	// (dependencies are optional)
	orchDir := filepath.Join("testdata", "orchestrator")
	wtBase := filepath.Join("testdata", "worktrees")

	fs, err := ReadFullStatus(orchDir, wtBase, "/nonexistent/plan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fs.Dependencies != nil {
		t.Errorf("dependencies = %v, want nil for missing plan", fs.Dependencies)
	}
}

func TestReadSliceStatuses_ReadError(t *testing.T) {
	dir := t.TempDir()
	statusFile := filepath.Join(dir, "slice-bad.status")
	// Create a directory where a file is expected to cause read error
	if err := os.Mkdir(statusFile, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := ReadSliceStatuses(dir)
	if err == nil {
		t.Fatal("expected error reading directory as file, got nil")
	}
}
