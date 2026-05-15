package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAppendEnvIfMissing_TomlFillsWhenEnvAbsent confirms that the loop
// driver value from ralph.toml reaches the orchestrator when the user has
// not already set the corresponding env var. This is the AC-4 path: TOML
// alone must be runtime-effective.
func TestAppendEnvIfMissing_TomlFillsWhenEnvAbsent(t *testing.T) {
	env := []string{"PATH=/usr/bin", "HOME=/home/user"}
	got := appendEnvIfMissing(env, "RALPH_LOOP_DRIVER", "codex")
	if !containsKV(got, "RALPH_LOOP_DRIVER", "codex") {
		t.Errorf("expected RALPH_LOOP_DRIVER=codex in %v", got)
	}
}

// TestAppendEnvIfMissing_EnvWinsOverToml confirms the documented priority:
// an explicit env var must NOT be overwritten by the TOML value.
func TestAppendEnvIfMissing_EnvWinsOverToml(t *testing.T) {
	env := []string{"PATH=/usr/bin", "RALPH_LOOP_DRIVER=claude"}
	got := appendEnvIfMissing(env, "RALPH_LOOP_DRIVER", "codex")
	count := 0
	var lastVal string
	for _, e := range got {
		if strings.HasPrefix(e, "RALPH_LOOP_DRIVER=") {
			count++
			lastVal = strings.TrimPrefix(e, "RALPH_LOOP_DRIVER=")
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 RALPH_LOOP_DRIVER entry, got %d in %v", count, got)
	}
	if lastVal != "claude" {
		t.Errorf("env should win: got %q, want claude", lastVal)
	}
}

// TestAppendEnvIfMissing_DoesNotMatchPrefix guards against accidental matches
// when a variable's name is a prefix of another (e.g. RALPH_LOOP and
// RALPH_LOOP_DRIVER).
func TestAppendEnvIfMissing_DoesNotMatchPrefix(t *testing.T) {
	env := []string{"RALPH_LOOP_DRIVER_EXTRA=ignored"}
	got := appendEnvIfMissing(env, "RALPH_LOOP_DRIVER", "codex")
	if !containsKV(got, "RALPH_LOOP_DRIVER", "codex") {
		t.Errorf("RALPH_LOOP_DRIVER not added; prefix match leaked: %v", got)
	}
}

func containsKV(env []string, key, value string) bool {
	want := key + "=" + value
	for _, e := range env {
		if e == want {
			return true
		}
	}
	return false
}

func TestDetectLatestPlanDir_SelectsNewestManifestDirectory(t *testing.T) {
	dir := t.TempDir()
	activeDir := filepath.Join(dir, "docs", "plans", "active")
	for _, plan := range []string{
		"2026-01-01-old-plan",
		"2026-05-14-new-plan",
	} {
		manifestDir := filepath.Join(activeDir, plan)
		if err := os.MkdirAll(manifestDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(manifestDir, "_manifest.md"), []byte("# plan\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(activeDir, "2026-12-31-no-manifest"), 0755); err != nil {
		t.Fatal(err)
	}

	got, err := detectLatestPlanDir(activeDir)
	if err != nil {
		t.Fatalf("detectLatestPlanDir: %v", err)
	}
	want := filepath.Join(activeDir, "2026-05-14-new-plan")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDetectLatestPlanDir_NoManifestDirectoryFails(t *testing.T) {
	dir := t.TempDir()
	activeDir := filepath.Join(dir, "docs", "plans", "active")
	if err := os.MkdirAll(filepath.Join(activeDir, "2026-05-14-no-manifest"), 0755); err != nil {
		t.Fatal(err)
	}

	_, err := detectLatestPlanDir(activeDir)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no directory-based plan found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunPipelineAutoDetectsPlan(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	if err := os.Mkdir("scripts", 0755); err != nil {
		t.Fatal(err)
	}
	stub := "#!/usr/bin/env sh\nfor arg in \"$@\"; do printf '%s\\n' \"$arg\"; done > args.txt\n"
	if err := os.WriteFile(filepath.Join("scripts", "ralph-orchestrator.sh"), []byte(stub), 0755); err != nil {
		t.Fatal(err)
	}

	for _, plan := range []string{
		"2026-01-01-old-plan",
		"2026-05-14-new-plan",
	} {
		manifestDir := filepath.Join(activePlansDir, plan)
		if err := os.MkdirAll(manifestDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(manifestDir, "_manifest.md"), []byte("# plan\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := runPipeline("", 0, 0, false, false, true, false); err != nil {
		t.Fatalf("runPipeline: %v", err)
	}

	data, err := os.ReadFile("args.txt")
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Fields(string(data))
	wantPlan := filepath.Join(activePlansDir, "2026-05-14-new-plan")
	for _, want := range []string{"--plan", wantPlan, "--dry-run"} {
		found := false
		for _, arg := range got {
			if arg == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("forwarded args %v missing %q", got, want)
		}
	}
}

func TestRunPipeline_ForwardsRunModeFlags(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	if err := os.Mkdir("scripts", 0755); err != nil {
		t.Fatal(err)
	}
	argsFile := filepath.Join(dir, "args.txt")
	stub := "#!/usr/bin/env sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\n"
	if err := os.WriteFile(filepath.Join("scripts", "ralph-orchestrator.sh"), []byte(stub), 0755); err != nil {
		t.Fatal(err)
	}

	if err := runPipeline("docs/plans/active/example", 0, 0, true, true, true, true); err != nil {
		t.Fatalf("runPipeline: %v", err)
	}

	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Fields(string(data))
	for _, want := range []string{"--plan", "docs/plans/active/example", "--preflight", "--resume", "--dry-run", "--unified-pr"} {
		found := false
		for _, arg := range got {
			if arg == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("forwarded args %v missing %q", got, want)
		}
	}
}
