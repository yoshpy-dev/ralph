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

// TestRunPipeline_ExportsPhaseModelEnv verifies that `ralph run` exports the
// per-phase RALPH_<PHASE>_MODEL variables to the orchestrator with the
// documented priority env > TOML > default, and that RALPH_FORCE_MODEL is
// NOT exported when [pipeline.phases] force is empty/absent — an empty force
// knob must never mask a user's env var or look like an explicit override.
func TestRunPipeline_ExportsPhaseModelEnv(t *testing.T) {
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
	stub := "#!/usr/bin/env sh\nenv > env.txt\n"
	if err := os.WriteFile(filepath.Join("scripts", "ralph-orchestrator.sh"), []byte(stub), 0755); err != nil {
		t.Fatal(err)
	}

	// TOML overrides one phase; force stays absent (default empty).
	toml := `[pipeline.phases]
verify = "opus"
`
	if err := os.WriteFile("ralph.toml", []byte(toml), 0644); err != nil {
		t.Fatal(err)
	}

	// Env var must win over the TOML/default value for implement.
	t.Setenv("RALPH_IMPLEMENT_MODEL", "opus")

	if err := runPipeline("docs/plans/active/example", 0, 0, false, false, true, false); err != nil {
		t.Fatalf("runPipeline: %v", err)
	}

	data, err := os.ReadFile("env.txt")
	if err != nil {
		t.Fatal(err)
	}
	envLines := strings.Split(strings.TrimSpace(string(data)), "\n")

	want := map[string]string{
		"RALPH_IMPLEMENT_MODEL":   "opus",   // env wins
		"RALPH_SELF_REVIEW_MODEL": "opus",   // default
		"RALPH_VERIFY_MODEL":      "opus",   // toml wins over default sonnet
		"RALPH_TEST_MODEL":        "sonnet", // default
		"RALPH_SYNC_DOCS_MODEL":   "sonnet", // default
		"RALPH_PR_MODEL":          "sonnet", // default
		"RALPH_PROBE_MODEL":       "haiku",  // default
		"RALPH_ESCALATION_MODEL":  "opus",   // default
	}
	for key, val := range want {
		if !containsKV(envLines, key, val) {
			t.Errorf("expected %s=%s in orchestrator env", key, val)
		}
	}

	// Empty force must not be exported at all.
	for _, e := range envLines {
		if strings.HasPrefix(e, "RALPH_FORCE_MODEL=") {
			t.Errorf("RALPH_FORCE_MODEL must not be exported when force is empty, got %q", e)
		}
	}
}

// TestRunPipeline_ExportsForceModelWhenSet verifies that a non-empty
// [pipeline.phases] force value IS exported as RALPH_FORCE_MODEL.
func TestRunPipeline_ExportsForceModelWhenSet(t *testing.T) {
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
	stub := "#!/usr/bin/env sh\nenv > env.txt\n"
	if err := os.WriteFile(filepath.Join("scripts", "ralph-orchestrator.sh"), []byte(stub), 0755); err != nil {
		t.Fatal(err)
	}

	toml := `[pipeline.phases]
force = "opus"
`
	if err := os.WriteFile("ralph.toml", []byte(toml), 0644); err != nil {
		t.Fatal(err)
	}

	if err := runPipeline("docs/plans/active/example", 0, 0, false, false, true, false); err != nil {
		t.Fatalf("runPipeline: %v", err)
	}

	data, err := os.ReadFile("env.txt")
	if err != nil {
		t.Fatal(err)
	}
	envLines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if !containsKV(envLines, "RALPH_FORCE_MODEL", "opus") {
		t.Error("expected RALPH_FORCE_MODEL=opus in orchestrator env when [pipeline.phases] force is set")
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
