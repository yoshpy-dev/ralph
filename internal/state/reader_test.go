package state

import (
	"os"
	"path/filepath"
	"testing"
)

// mustWriteFile is a test helper that writes a file and fails on error.
func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

// mustMkdirAll is a test helper that creates directories and fails on error.
func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("failed to mkdir %s: %v", path, err)
	}
}

// mustReadFixture reads a testdata fixture file and fails on error.
func mustReadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}
	return data
}

func TestReadOrchestratorState(t *testing.T) {
	t.Run("valid running orchestrator", func(t *testing.T) {
		s, err := ReadOrchestratorState("testdata")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Plan != "docs/plans/active/2026-04-15-ralph-tui/_manifest.md" {
			t.Errorf("plan = %q, want docs/plans/active/2026-04-15-ralph-tui/_manifest.md", s.Plan)
		}
		if s.Status != "running" {
			t.Errorf("status = %q, want running", s.Status)
		}
		if s.MaxParallel != 3 {
			t.Errorf("max_parallel = %d, want 3", s.MaxParallel)
		}
		if !s.UnifiedPR {
			t.Error("unified_pr = false, want true")
		}
		if s.Started != "2026-04-15T06:00:00Z" {
			t.Errorf("started = %q, want 2026-04-15T06:00:00Z", s.Started)
		}
	})

	t.Run("valid complete orchestrator", func(t *testing.T) {
		dir := t.TempDir()
		data := mustReadFixture(t, "testdata/orchestrator-complete.json")
		mustWriteFile(t, filepath.Join(dir, "orchestrator.json"), data)

		s, err := ReadOrchestratorState(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Status != "complete" {
			t.Errorf("status = %q, want complete", s.Status)
		}
		if s.Ended != "2026-04-15T08:30:00Z" {
			t.Errorf("ended = %q, want 2026-04-15T08:30:00Z", s.Ended)
		}
		if s.PRUrl != "https://github.com/yoshpy-dev/harness-engineering-scaffolding-template/pull/42" {
			t.Errorf("pr_url = %q, unexpected", s.PRUrl)
		}
		if s.IntegrationBranch != "integration/ralph-tui" {
			t.Errorf("integration_branch = %q, want integration/ralph-tui", s.IntegrationBranch)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := ReadOrchestratorState("/tmp/does-not-exist-ralph-test")
		if err == nil {
			t.Fatal("expected error for nonexistent file, got nil")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		mustWriteFile(t, filepath.Join(dir, "orchestrator.json"), []byte("{invalid json}"))

		_, err := ReadOrchestratorState(dir)
		if err == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}
	})

	t.Run("empty JSON object", func(t *testing.T) {
		dir := t.TempDir()
		mustWriteFile(t, filepath.Join(dir, "orchestrator.json"), []byte("{}"))

		s, err := ReadOrchestratorState(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.Status != "" {
			t.Errorf("status = %q, want empty", s.Status)
		}
	})
}

func TestReadSliceStatus(t *testing.T) {
	t.Run("valid status file", func(t *testing.T) {
		dir := t.TempDir()
		mustWriteFile(t, filepath.Join(dir, "slice-1-ralph-tui.status"), []byte("running\n"))

		status, err := ReadSliceStatus(dir, "1-ralph-tui")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status != "running" {
			t.Errorf("status = %q, want running", status)
		}
	})

	t.Run("nonexistent status file", func(t *testing.T) {
		_, err := ReadSliceStatus("/tmp/does-not-exist-ralph-test", "foo")
		if err == nil {
			t.Fatal("expected error for nonexistent file, got nil")
		}
	})

	t.Run("status with whitespace", func(t *testing.T) {
		dir := t.TempDir()
		mustWriteFile(t, filepath.Join(dir, "slice-test.status"), []byte("  complete  \n"))

		status, err := ReadSliceStatus(dir, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if status != "complete" {
			t.Errorf("status = %q, want complete", status)
		}
	})
}

func TestReadPipelineCheckpoint(t *testing.T) {
	t.Run("valid checkpoint", func(t *testing.T) {
		dir := t.TempDir()
		sliceDir := filepath.Join(dir, "1-ralph-tui", ".harness", "state", "pipeline")
		mustMkdirAll(t, sliceDir)
		data := mustReadFixture(t, "testdata/checkpoint.json")
		mustWriteFile(t, filepath.Join(sliceDir, "checkpoint.json"), data)

		c, err := ReadPipelineCheckpoint(dir, "1-ralph-tui")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.Phase != "inner" {
			t.Errorf("phase = %q, want inner", c.Phase)
		}
		if c.InnerCycle != 2 {
			t.Errorf("inner_cycle = %d, want 2", c.InnerCycle)
		}
		if c.Iteration != 3 {
			t.Errorf("iteration = %d, want 3", c.Iteration)
		}
		if c.LastTestResult != nil {
			t.Errorf("last_test_result = %v, want nil", c.LastTestResult)
		}
		if len(c.AcceptanceCriteriaMet) != 1 || c.AcceptanceCriteriaMet[0] != "AC-1" {
			t.Errorf("acceptance_criteria_met = %v, want [AC-1]", c.AcceptanceCriteriaMet)
		}
		if len(c.PhaseTransitions) != 1 {
			t.Fatalf("phase_transitions count = %d, want 1", len(c.PhaseTransitions))
		}
		if c.PhaseTransitions[0].Timestamp != "2026-04-15T06:10:00Z" {
			t.Errorf("first transition timestamp = %q, unexpected", c.PhaseTransitions[0].Timestamp)
		}
	})

	t.Run("complete checkpoint with failure triage", func(t *testing.T) {
		dir := t.TempDir()
		sliceDir := filepath.Join(dir, "test", ".harness", "state", "pipeline")
		mustMkdirAll(t, sliceDir)
		data := mustReadFixture(t, "testdata/checkpoint-complete.json")
		mustWriteFile(t, filepath.Join(sliceDir, "checkpoint.json"), data)

		c, err := ReadPipelineCheckpoint(dir, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.Status != "complete" {
			t.Errorf("status = %q, want complete", c.Status)
		}
		if c.LastTestResult == nil || *c.LastTestResult != "pass" {
			t.Errorf("last_test_result = %v, want pass", c.LastTestResult)
		}
		if len(c.FailureTriage) != 1 {
			t.Fatalf("failure_triage count = %d, want 1", len(c.FailureTriage))
		}
		if c.FailureTriage[0].Hypothesis != "missing import in types.go" {
			t.Errorf("failure_triage[0].hypothesis = %q, unexpected", c.FailureTriage[0].Hypothesis)
		}
		if c.CodexTriage.WorthConsidering != 1 {
			t.Errorf("codex_triage.worth_considering = %d, want 1", c.CodexTriage.WorthConsidering)
		}
		if !c.PRCreated {
			t.Error("pr_created = false, want true")
		}
	})

	t.Run("nonexistent checkpoint", func(t *testing.T) {
		_, err := ReadPipelineCheckpoint("/tmp/does-not-exist-ralph-test", "foo")
		if err == nil {
			t.Fatal("expected error for nonexistent file, got nil")
		}
	})

	t.Run("invalid JSON checkpoint", func(t *testing.T) {
		dir := t.TempDir()
		sliceDir := filepath.Join(dir, "bad", ".harness", "state", "pipeline")
		mustMkdirAll(t, sliceDir)
		mustWriteFile(t, filepath.Join(sliceDir, "checkpoint.json"), []byte("not json"))

		_, err := ReadPipelineCheckpoint(dir, "bad")
		if err == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}
	})
}

func TestReadSliceDependencies(t *testing.T) {
	t.Run("valid manifest", func(t *testing.T) {
		deps, err := ReadSliceDependencies("testdata")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []SliceDependency{
			{From: "slice-1", To: "slice-2"},
			{From: "slice-1", To: "slice-3"},
			{From: "slice-3", To: "slice-4"},
			{From: "slice-3", To: "slice-5"},
			{From: "slice-2", To: "slice-6"},
			{From: "slice-4", To: "slice-6"},
			{From: "slice-5", To: "slice-6"},
		}

		if len(deps) != len(expected) {
			t.Fatalf("got %d deps, want %d: %+v", len(deps), len(expected), deps)
		}

		for i, want := range expected {
			if deps[i].From != want.From || deps[i].To != want.To {
				t.Errorf("dep[%d] = {%s->%s}, want {%s->%s}",
					i, deps[i].From, deps[i].To, want.From, want.To)
			}
		}
	})

	t.Run("nonexistent manifest", func(t *testing.T) {
		_, err := ReadSliceDependencies("/tmp/does-not-exist-ralph-test")
		if err == nil {
			t.Fatal("expected error for nonexistent file, got nil")
		}
	})

	t.Run("manifest without dependency section", func(t *testing.T) {
		dir := t.TempDir()
		mustWriteFile(t, filepath.Join(dir, "_manifest.md"), []byte("# Plan\n\n## Objective\n\nSome objective.\n"))

		deps, err := ReadSliceDependencies(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(deps) != 0 {
			t.Errorf("got %d deps, want 0", len(deps))
		}
	})

	t.Run("manifest with empty dependency section", func(t *testing.T) {
		dir := t.TempDir()
		content := "# Plan\n\n## Dependency graph\n\n```\n```\n\n## Next\n"
		mustWriteFile(t, filepath.Join(dir, "_manifest.md"), []byte(content))

		deps, err := ReadSliceDependencies(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(deps) != 0 {
			t.Errorf("got %d deps, want 0", len(deps))
		}
	})

	t.Run("resolves slice-N to real names", func(t *testing.T) {
		// When slice names are provided, "slice-1" should resolve to the real name.
		names := []string{"1-ralph-tui", "2-ralph-tui", "3-ralph-tui", "4-ralph-tui", "5-ralph-tui", "6-ralph-tui"}
		deps, err := ReadSliceDependencies("testdata", names...)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []SliceDependency{
			{From: "1-ralph-tui", To: "2-ralph-tui"},
			{From: "1-ralph-tui", To: "3-ralph-tui"},
			{From: "3-ralph-tui", To: "4-ralph-tui"},
			{From: "3-ralph-tui", To: "5-ralph-tui"},
			{From: "2-ralph-tui", To: "6-ralph-tui"},
			{From: "4-ralph-tui", To: "6-ralph-tui"},
			{From: "5-ralph-tui", To: "6-ralph-tui"},
		}

		if len(deps) != len(expected) {
			t.Fatalf("got %d deps, want %d: %+v", len(deps), len(expected), deps)
		}

		for i, want := range expected {
			if deps[i].From != want.From || deps[i].To != want.To {
				t.Errorf("dep[%d] = {%s->%s}, want {%s->%s}",
					i, deps[i].From, deps[i].To, want.From, want.To)
			}
		}
	})
}

func TestListSliceNames(t *testing.T) {
	t.Run("multiple status files", func(t *testing.T) {
		dir := t.TempDir()
		for _, name := range []string{"1-ralph-tui", "2-ralph-tui", "3-ralph-tui"} {
			mustWriteFile(t, filepath.Join(dir, "slice-"+name+".status"), []byte("running"))
		}

		names, err := ListSliceNames(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(names) != 3 {
			t.Fatalf("got %d names, want 3", len(names))
		}
	})

	t.Run("no status files", func(t *testing.T) {
		dir := t.TempDir()
		names, err := ListSliceNames(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(names) != 0 {
			t.Errorf("got %d names, want 0", len(names))
		}
	})
}

func TestReadFullStatus(t *testing.T) {
	dir := t.TempDir()
	orchDir := filepath.Join(dir, "orch")
	wtBase := filepath.Join(dir, "worktrees")
	planDir := filepath.Join(dir, "plan")
	mustMkdirAll(t, orchDir)
	mustMkdirAll(t, planDir)

	orchData := mustReadFixture(t, "testdata/orchestrator.json")
	mustWriteFile(t, filepath.Join(orchDir, "orchestrator.json"), orchData)

	mustWriteFile(t, filepath.Join(orchDir, "slice-1-ralph-tui.status"), []byte("running"))
	mustWriteFile(t, filepath.Join(orchDir, "slice-2-ralph-tui.status"), []byte("pending"))

	ckptDir := filepath.Join(wtBase, "1-ralph-tui", ".harness", "state", "pipeline")
	mustMkdirAll(t, ckptDir)
	ckptData := mustReadFixture(t, "testdata/checkpoint.json")
	mustWriteFile(t, filepath.Join(ckptDir, "checkpoint.json"), ckptData)

	manifestContent := "# Plan\n\n## Dependency graph\n\n```\nslice-1 ──→ slice-2\n```\n\n## End\n"
	mustWriteFile(t, filepath.Join(planDir, "_manifest.md"), []byte(manifestContent))

	full, err := ReadFullStatus(orchDir, wtBase, planDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if full.Orchestrator.Status != "running" {
		t.Errorf("orchestrator status = %q, want running", full.Orchestrator.Status)
	}
	if len(full.Slices) != 2 {
		t.Fatalf("slices count = %d, want 2", len(full.Slices))
	}
	if full.Progress.Completed != 0 {
		t.Errorf("progress.completed = %d, want 0", full.Progress.Completed)
	}
	if full.Progress.Total != 2 {
		t.Errorf("progress.total = %d, want 2", full.Progress.Total)
	}
	if full.Progress.Percent != 0 {
		t.Errorf("progress.percent = %d, want 0", full.Progress.Percent)
	}
	if len(full.Dependencies) != 1 {
		t.Fatalf("dependencies count = %d, want 1", len(full.Dependencies))
	}
	// With slice names ["1-ralph-tui", "2-ralph-tui"], "slice-1" resolves to "1-ralph-tui".
	if full.Dependencies[0].From != "1-ralph-tui" || full.Dependencies[0].To != "2-ralph-tui" {
		t.Errorf("dependency = {%s->%s}, want {1-ralph-tui->2-ralph-tui}",
			full.Dependencies[0].From, full.Dependencies[0].To)
	}

	var slice1 *SliceState
	for i := range full.Slices {
		if full.Slices[i].Name == "1-ralph-tui" {
			slice1 = &full.Slices[i]
			break
		}
	}
	if slice1 == nil {
		t.Fatal("slice 1-ralph-tui not found")
	}
	if slice1.Phase != "inner" {
		t.Errorf("slice1 phase = %q, want inner", slice1.Phase)
	}
	if slice1.Cycle != 2 {
		t.Errorf("slice1 inner_cycle = %d, want 2", slice1.Cycle)
	}

	var slice2 *SliceState
	for i := range full.Slices {
		if full.Slices[i].Name == "2-ralph-tui" {
			slice2 = &full.Slices[i]
			break
		}
	}
	if slice2 == nil {
		t.Fatal("slice 2-ralph-tui not found")
	}
	if slice2.Phase != "waiting" {
		t.Errorf("slice2 phase = %q, want waiting", slice2.Phase)
	}
}

func TestReadFullStatus_NoOrchestrator(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadFullStatus(dir, dir, dir)
	if err == nil {
		t.Fatal("expected error when orchestrator.json is missing")
	}
}

func TestOrchestratorState_StartedTime(t *testing.T) {
	s := &OrchestratorState{Started: "2026-04-15T06:00:00Z"}
	ts, err := s.StartedTime()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts.Year() != 2026 || ts.Month() != 4 || ts.Day() != 15 {
		t.Errorf("parsed time = %v, unexpected", ts)
	}
}

func TestOrchestratorState_StartedTime_Empty(t *testing.T) {
	s := &OrchestratorState{}
	ts, err := s.StartedTime()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ts.IsZero() {
		t.Errorf("expected zero time for empty started, got %v", ts)
	}
}

func TestPipelineCheckpoint_FirstTransitionTime(t *testing.T) {
	c := &PipelineCheckpoint{
		PhaseTransitions: []PhaseTransition{
			{Timestamp: "2026-04-15T06:10:00Z"},
		},
	}
	ts, err := c.FirstTransitionTime()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts.Hour() != 6 || ts.Minute() != 10 {
		t.Errorf("parsed time = %v, unexpected", ts)
	}
}

func TestPipelineCheckpoint_FirstTransitionTime_Empty(t *testing.T) {
	c := &PipelineCheckpoint{}
	ts, err := c.FirstTransitionTime()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ts.IsZero() {
		t.Errorf("expected zero time, got %v", ts)
	}
}

func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		wantZero bool
	}{
		{"RFC3339", "2026-04-15T06:00:00Z", false, false},
		{"RFC3339 with offset", "2026-04-15T06:00:00+09:00", false, false},
		{"without timezone", "2026-04-15T06:00:00", false, false},
		{"empty", "", false, true},
		{"invalid", "not-a-date", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := parseTimestamp(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimestamp(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && ts.IsZero() != tt.wantZero {
				t.Errorf("parseTimestamp(%q) isZero = %v, want %v", tt.input, ts.IsZero(), tt.wantZero)
			}
		})
	}
}
