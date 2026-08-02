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

	if err := runPipeline("", 0, 0, false, false, true, false, false, false); err != nil {
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

// TestRunPipeline_ExportsPhaseModelEnvDefaults verifies that `ralph run`
// still exports the per-phase RALPH_<PHASE>_MODEL variables (now as literal
// defaults — see runDefaults in run.go — rather than values read from
// ralph.toml's removed [pipeline.phases] section), and that a pre-set env
// var still wins over the literal default.
func TestRunPipeline_ExportsPhaseModelEnvDefaults(t *testing.T) {
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

	// Env var must win over the literal default for implement.
	t.Setenv("RALPH_IMPLEMENT_MODEL", "opus")

	if err := runPipeline("docs/plans/active/example", 0, 0, false, false, true, false, false, false); err != nil {
		t.Fatalf("runPipeline: %v", err)
	}

	data, err := os.ReadFile("env.txt")
	if err != nil {
		t.Fatal(err)
	}
	envLines := strings.Split(strings.TrimSpace(string(data)), "\n")

	want := map[string]string{
		"RALPH_IMPLEMENT_MODEL":   "opus",   // env wins
		"RALPH_SELF_REVIEW_MODEL": "opus",   // literal default
		"RALPH_VERIFY_MODEL":      "sonnet", // literal default
		"RALPH_TEST_MODEL":        "sonnet", // literal default
		"RALPH_SYNC_DOCS_MODEL":   "sonnet", // literal default
		"RALPH_PR_MODEL":          "sonnet", // literal default
		"RALPH_PROBE_MODEL":       "haiku",  // literal default
		"RALPH_ESCALATION_MODEL":  "opus",   // literal default
	}
	for key, val := range want {
		if !containsKV(envLines, key, val) {
			t.Errorf("expected %s=%s in orchestrator env", key, val)
		}
	}

	// RALPH_FORCE_MODEL export was removed along with [pipeline.phases].force;
	// it must never appear.
	for _, e := range envLines {
		if strings.HasPrefix(e, "RALPH_FORCE_MODEL=") {
			t.Errorf("RALPH_FORCE_MODEL must not be exported (force knob removed), got %q", e)
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

	if err := runPipeline("docs/plans/active/example", 0, 0, true, true, true, true, false, false); err != nil {
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

// setupEnvStub creates a temp dir, changes to it, installs an env-dumping
// orchestrator stub, and returns cleanup. Callers must defer the returned func.
func setupEnvStub(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir("scripts", 0755); err != nil {
		t.Fatal(err)
	}
	stub := "#!/usr/bin/env sh\nenv > env.txt\n"
	if err := os.WriteFile(filepath.Join("scripts", "ralph-orchestrator.sh"), []byte(stub), 0755); err != nil {
		t.Fatal(err)
	}
	return func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}
}

func readEnvLines(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile("env.txt")
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

// TestRunPipeline_EnvWinsOverTomlForModelEtc asserts AC2:
// when RALPH_MODEL, RALPH_EFFORT, RALPH_PERMISSION_MODE are pre-set in the
// environment, ralph run must forward those values unchanged — not replace
// them with ralph.toml values.
func TestRunPipeline_EnvWinsOverTomlForModelEtc(t *testing.T) {
	cleanup := setupEnvStub(t)
	defer cleanup()

	// TOML sets different values for every key under test.
	tomlContent := `[pipeline]
model = "sonnet"
effort = "low"
permission_mode = "auto"
`
	if err := os.WriteFile("ralph.toml", []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Pre-set env vars that must win.
	t.Setenv("RALPH_MODEL", "haiku")
	t.Setenv("RALPH_EFFORT", "high")
	t.Setenv("RALPH_PERMISSION_MODE", "bypassPermissions")

	if err := runPipeline("docs/plans/active/example", 0, 0, false, false, true, false, false, false); err != nil {
		t.Fatalf("runPipeline: %v", err)
	}

	envLines := readEnvLines(t)
	for _, tc := range []struct{ key, want string }{
		{"RALPH_MODEL", "haiku"},
		{"RALPH_EFFORT", "high"},
		{"RALPH_PERMISSION_MODE", "bypassPermissions"},
	} {
		if !containsKV(envLines, tc.key, tc.want) {
			t.Errorf("AC2: expected %s=%s (env wins over toml), got something else in env", tc.key, tc.want)
		}
		// Also assert the TOML value is NOT present (no duplicate entry).
		count := 0
		for _, e := range envLines {
			if strings.HasPrefix(e, tc.key+"=") {
				count++
			}
		}
		if count != 1 {
			t.Errorf("AC2: expected exactly 1 entry for %s, got %d", tc.key, count)
		}
	}
}

// TestRunPipeline_MaxIterFlagBeatsEnv asserts AC3 (flag path):
// when --max-iterations is explicitly set (maxIterChanged=true), the CLI value
// must win over any pre-existing RALPH_MAX_ITERATIONS in the environment.
func TestRunPipeline_MaxIterFlagBeatsEnv(t *testing.T) {
	cleanup := setupEnvStub(t)
	defer cleanup()

	// Pre-set env var that the flag must override.
	t.Setenv("RALPH_MAX_ITERATIONS", "99")

	// maxIterChanged=true simulates --max-iterations 5 on the CLI.
	if err := runPipeline("docs/plans/active/example", 5, 0, false, false, true, false, true, false); err != nil {
		t.Fatalf("runPipeline: %v", err)
	}

	envLines := readEnvLines(t)
	// The last RALPH_MAX_ITERATIONS wins in a process env; we check the
	// exported value is the CLI value (5), not the env value (99).
	// Because exec.Cmd.Env is the complete env, containsKV is the right check.
	if !containsKV(envLines, "RALPH_MAX_ITERATIONS", "5") {
		t.Errorf("AC3: expected RALPH_MAX_ITERATIONS=5 (CLI flag wins), envLines=%v", envLines)
	}
}

// TestRunPipeline_MaxIterEnvBeatsTomlWhenNoFlag asserts AC3 (env path):
// when --max-iterations is absent (maxIterChanged=false), a pre-set env var
// must win over the ralph.toml value.
func TestRunPipeline_MaxIterEnvBeatsTomlWhenNoFlag(t *testing.T) {
	cleanup := setupEnvStub(t)
	defer cleanup()

	tomlContent := `[pipeline]
max_iterations = 7
`
	if err := os.WriteFile("ralph.toml", []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("RALPH_MAX_ITERATIONS", "42")

	// maxIterChanged=false: no CLI flag, env should win over toml.
	if err := runPipeline("docs/plans/active/example", 0, 0, false, false, true, false, false, false); err != nil {
		t.Fatalf("runPipeline: %v", err)
	}

	envLines := readEnvLines(t)
	if !containsKV(envLines, "RALPH_MAX_ITERATIONS", "42") {
		t.Errorf("AC3: expected RALPH_MAX_ITERATIONS=42 (env wins over toml when flag absent), envLines=%v", envLines)
	}
}

// TestRunPipeline_EmptyPermissionModeEnv asserts the "non-empty env wins"
// contract for RALPH_PERMISSION_MODE: when the env var is present but empty
// (RALPH_PERMISSION_MODE=), appendEnvIfMissing treats it as "present" and
// does NOT export the toml value — the downstream shell's ${VAR:-default}
// then resolves it to the shell default (bypassPermissions).
//
// This test verifies the Go side of the contract: that cmd.Env does NOT
// contain a RALPH_PERMISSION_MODE=<toml-value> entry that would override the
// empty var already in the environment.  The shell-layer resolution
// (${RALPH_PERMISSION_MODE:-bypassPermissions}) is tested by
// tests/test-ralph-config.sh which sources scripts/ralph-config.sh directly.
func TestRunPipeline_EmptyPermissionModeEnv(t *testing.T) {
	cleanup := setupEnvStub(t)
	defer cleanup()

	tomlContent := `[pipeline]
permission_mode = "auto"
`
	if err := os.WriteFile("ralph.toml", []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Present-but-empty: signals "user explicitly set this" even though value is empty.
	t.Setenv("RALPH_PERMISSION_MODE", "")

	if err := runPipeline("docs/plans/active/example", 0, 0, false, false, true, false, false, false); err != nil {
		t.Fatalf("runPipeline: %v", err)
	}

	envLines := readEnvLines(t)
	// The toml value "auto" must NOT appear — empty env wins over toml.
	if containsKV(envLines, "RALPH_PERMISSION_MODE", "auto") {
		t.Error("empty env var must win over toml: RALPH_PERMISSION_MODE=auto must not appear in orchestrator env")
	}
	// The empty entry IS present (inherited from os.Environ via t.Setenv).
	found := false
	for _, e := range envLines {
		if e == "RALPH_PERMISSION_MODE=" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected RALPH_PERMISSION_MODE= (empty) to appear in orchestrator env")
	}
}
