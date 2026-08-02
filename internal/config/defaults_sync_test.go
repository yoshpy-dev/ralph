package config

// TestDefaultsLockStep verifies that the three authoritative sources for
// model/effort defaults stay in lock-step:
//
//  1. scripts/ralph-config.sh   — shell defaults (VAR="${VAR:-value}" idiom)
//  2. templates/base/ralph.toml — declarative project config
//  3. config.Default()          — Go CLI defaults exported as RALPH_* env vars
//
// It also checks that the cross-review SKILL.md fallback for
// RALPH_CLAUDE_REVIEWER_MODEL matches the shell default.
//
// Any mismatch is a test failure naming the surface and key.  The test uses
// runtime.Caller to resolve paths relative to this file so it works from any
// working directory.  Files are resolved relative to the repo root
// (two directories above internal/config/).
//
// If a file is absent (e.g. the package is vendored), the relevant sub-test
// is skipped with an explanatory message so the test does not break in
// downstream repos that embed internal/config without the meta-repo tree.

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// repoRoot resolves the repository root from this source file's location.
// internal/config/defaults_sync_test.go → repo root is ../../
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location via runtime.Caller")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

// parseShellDefaults reads a shell file and extracts VAR="${VAR:-value}"
// defaults into a map.  Only top-level assignments of the form
//
//	VARNAME="${VARNAME:-default_value}"
//
// are captured.  The idiom is documented in scripts/ralph-config.sh.
//
// Go's regexp package does not support backreferences, so we match the full
// pattern with two named groups and verify post-match that the variable name
// in the LHS matches the name inside the parameter expansion.
func parseShellDefaults(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("parseShellDefaults: cannot read %s: %v", path, err)
	}
	// Matches: VARNAME="${ANYNAME:-value}"
	// Captures: [1]=LHS name  [2]=inner name  [3]=default value
	// We then confirm [1]==[2] to enforce the self-referential idiom.
	re := regexp.MustCompile(`^([A-Z_]+)="\$\{([A-Z_]+):-([^}]*)}"`)
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		m := re.FindStringSubmatch(line)
		if len(m) == 4 && m[1] == m[2] {
			result[m[1]] = m[3]
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("parseShellDefaults: scanner error: %v", err)
	}
	return result
}

// mustShell returns the value for the given var from the shell map, failing
// loudly if missing.
func mustShell(t *testing.T, m map[string]string, varName string) string {
	t.Helper()
	v, ok := m[varName]
	if !ok {
		t.Fatalf("scripts/ralph-config.sh: %s not found by VAR=\"${VAR:-default}\" regex — update the regex or the file", varName)
	}
	return v
}

// TestDefaultsLockStep is the main tripwire.
func TestDefaultsLockStep(t *testing.T) {
	root := repoRoot(t)

	// ── 1. Shell defaults ─────────────────────────────────────────────────────
	shellPath := filepath.Join(root, "scripts", "ralph-config.sh")
	if _, err := os.Stat(shellPath); err != nil {
		t.Skipf("scripts/ralph-config.sh not found (%v) — skipping lock-step check (vendored repo?)", err)
	}
	shell := parseShellDefaults(t, shellPath)

	// ── 2. ralph.toml defaults (via Go parser) ────────────────────────────────
	tomlPath := filepath.Join(root, "templates", "base", "ralph.toml")
	if _, err := os.Stat(tomlPath); err != nil {
		t.Skipf("templates/base/ralph.toml not found (%v) — skipping lock-step check", err)
	}
	tomlCfg, err := Load(tomlPath)
	if err != nil {
		t.Fatalf("Load(templates/base/ralph.toml): %v", err)
	}

	// ── 3. Go Default() ───────────────────────────────────────────────────────
	goCfg := Default()

	// Helper: compare shell, toml, and Go values for one key.
	check := func(label, shellVar, tomlVal, goVal string) {
		t.Helper()
		shellVal := mustShell(t, shell, shellVar)
		if shellVal != goVal {
			t.Errorf("lock-step mismatch for %s:\n  scripts/ralph-config.sh %s default = %q\n  config.Default() = %q",
				label, shellVar, shellVal, goVal)
		}
		if tomlVal != goVal {
			t.Errorf("lock-step mismatch for %s:\n  templates/base/ralph.toml = %q\n  config.Default() = %q",
				label, tomlVal, goVal)
		}
		if shellVal != tomlVal {
			t.Errorf("lock-step mismatch for %s:\n  scripts/ralph-config.sh %s default = %q\n  templates/base/ralph.toml = %q",
				label, shellVar, shellVal, tomlVal)
		}
	}

	// ── [org] envelope defaults ────────────────────────────────────────────────
	// Shell vars are comma-joined strings (RALPH_ORG_DRIVER_POOL,
	// RALPH_ORG_MODEL_POOL as "driver:model,driver:model,...") since shell has
	// no native array type; the Go/TOML sides are canonicalized to the same
	// joined form before comparing so the three surfaces line up textually.
	check("org.driver_pool", "RALPH_ORG_DRIVER_POOL",
		strings.Join(tomlCfg.Org.DriverPool, ","), strings.Join(goCfg.Org.DriverPool, ","))

	formatOrgModelPool := func(entries []OrgModelPoolEntry) string {
		parts := make([]string, len(entries))
		for i, e := range entries {
			parts[i] = e.Driver + ":" + e.Model
		}
		return strings.Join(parts, ",")
	}
	check("org.model_pool", "RALPH_ORG_MODEL_POOL",
		formatOrgModelPool(tomlCfg.Org.ModelPool), formatOrgModelPool(goCfg.Org.ModelPool))

	check("org.max_seats", "RALPH_ORG_MAX_SEATS",
		strconv.Itoa(tomlCfg.Org.MaxSeats), strconv.Itoa(goCfg.Org.MaxSeats))
	check("org.budget.seat_wall_clock_minutes", "RALPH_ORG_SEAT_BUDGET_MINUTES",
		strconv.Itoa(tomlCfg.Org.Budget.SeatWallClockMinutes), strconv.Itoa(goCfg.Org.Budget.SeatWallClockMinutes))
	check("org.budget.total_wall_clock_minutes", "RALPH_ORG_TOTAL_BUDGET_MINUTES",
		strconv.Itoa(tomlCfg.Org.Budget.TotalWallClockMinutes), strconv.Itoa(goCfg.Org.Budget.TotalWallClockMinutes))
	check("org.budget.max_fix_rounds", "RALPH_ORG_MAX_FIX_ROUNDS",
		strconv.Itoa(tomlCfg.Org.Budget.MaxFixRounds), strconv.Itoa(goCfg.Org.Budget.MaxFixRounds))
	check("org.deadman_minutes", "RALPH_ORG_DEADMAN_MINUTES",
		strconv.Itoa(tomlCfg.Org.DeadmanMinutes), strconv.Itoa(goCfg.Org.DeadmanMinutes))
	check("org.agmsg_home", "RALPH_ORG_AGMSG_HOME",
		tomlCfg.Org.AgmsgHome, goCfg.Org.AgmsgHome)
	check("org.permissions.default", "RALPH_ORG_PERMISSION_DEFAULT",
		tomlCfg.Org.Permissions.Default, goCfg.Org.Permissions.Default)
	check("org.permissions.codex_verified", "RALPH_ORG_PERMISSIONS_CODEX_VERIFIED",
		strconv.FormatBool(tomlCfg.Org.Permissions.CodexVerified), strconv.FormatBool(goCfg.Org.Permissions.CodexVerified))
	check("org.watchdog.interval_seconds", "RALPH_ORG_WATCHDOG_INTERVAL_SECONDS",
		strconv.Itoa(tomlCfg.Org.Watchdog.IntervalSeconds), strconv.Itoa(goCfg.Org.Watchdog.IntervalSeconds))
	check("org.watchdog.stall_minutes", "RALPH_ORG_WATCHDOG_STALL_MINUTES",
		strconv.Itoa(tomlCfg.Org.Watchdog.StallMinutes), strconv.Itoa(goCfg.Org.Watchdog.StallMinutes))
	check("org.watchdog.watcher_enabled", "RALPH_ORG_WATCHDOG_WATCHER_ENABLED",
		strconv.FormatBool(tomlCfg.Org.Watchdog.WatcherEnabled), strconv.FormatBool(goCfg.Org.Watchdog.WatcherEnabled))
	check("org.watchdog.watcher_model", "RALPH_ORG_WATCHDOG_WATCHER_MODEL",
		tomlCfg.Org.Watchdog.WatcherModel, goCfg.Org.Watchdog.WatcherModel)

	// ── cross-review SKILL.md fallback matches shell ──────────────────────────
	// The SKILL.md documents: ${RALPH_CLAUDE_REVIEWER_MODEL:-opus}
	// We grep for the documented fallback token and assert it equals the shell
	// default for RALPH_CLAUDE_REVIEWER_MODEL.
	skillPath := filepath.Join(root, ".claude", "skills", "cross-review", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Skipf(".claude/skills/cross-review/SKILL.md not found (%v) — skipping reviewer fallback check", err)
	}
	{
		data, err := os.ReadFile(skillPath)
		if err != nil {
			t.Fatalf("cannot read cross-review/SKILL.md: %v", err)
		}
		// Pattern: ${RALPH_CLAUDE_REVIEWER_MODEL:-<fallback>}
		re := regexp.MustCompile(`\$\{RALPH_CLAUDE_REVIEWER_MODEL:-([^}]+)\}`)
		matches := re.FindAllSubmatch(data, -1)
		if len(matches) == 0 {
			t.Fatal(".claude/skills/cross-review/SKILL.md: no ${RALPH_CLAUDE_REVIEWER_MODEL:-<fallback>} pattern found — update the regex if the wording changed")
		}
		// All occurrences must agree and match the shell default.
		shellVal := mustShell(t, shell, "RALPH_CLAUDE_REVIEWER_MODEL")
		for _, m := range matches {
			skillFallback := string(m[1])
			if skillFallback != shellVal {
				t.Errorf("cross-review SKILL.md reviewer fallback = %q, want %q (scripts/ralph-config.sh RALPH_CLAUDE_REVIEWER_MODEL default)",
					skillFallback, shellVal)
			}
		}
	}
}
