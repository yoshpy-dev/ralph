package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Pipeline.Model != "opus" {
		t.Errorf("model = %q", cfg.Pipeline.Model)
	}
	if cfg.Pipeline.Effort != "high" {
		t.Errorf("effort = %q", cfg.Pipeline.Effort)
	}
	if cfg.Pipeline.MaxIterations != 20 {
		t.Errorf("max_iterations = %d", cfg.Pipeline.MaxIterations)
	}
	if cfg.Pipeline.Prompts.Dir != ".ralph/prompts" {
		t.Errorf("prompts.dir = %q", cfg.Pipeline.Prompts.Dir)
	}
	if !cfg.Doctor.RequireClaudeCLI {
		t.Error("require_claude_cli should be true by default")
	}
	// Codex is opt-in: defaults to false so projects without Codex
	// installed do not see `ralph doctor` exit non-zero.
	if cfg.Doctor.RequireCodexCLI {
		t.Error("require_codex_cli should default to false")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/ralph.toml")
	if err != nil {
		t.Fatalf("expected nil error for missing file, got: %v", err)
	}
	// Should return defaults.
	if cfg.Pipeline.MaxIterations != 20 {
		t.Errorf("max_iterations = %d, want 20", cfg.Pipeline.MaxIterations)
	}
}

func TestLoad_PartialConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")

	content := `[pipeline]
model = "claude-opus-4-20250514"
max_parallel = 8
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Overridden values.
	if cfg.Pipeline.Model != "claude-opus-4-20250514" {
		t.Errorf("model = %q, want claude-opus-4-20250514", cfg.Pipeline.Model)
	}
	if cfg.Pipeline.MaxParallel != 8 {
		t.Errorf("max_parallel = %d, want 8", cfg.Pipeline.MaxParallel)
	}

	// Defaults for unspecified values.
	if cfg.Pipeline.MaxIterations != 20 {
		t.Errorf("max_iterations = %d, want 20", cfg.Pipeline.MaxIterations)
	}
	if cfg.Pipeline.Effort != "high" {
		t.Errorf("effort = %q, want high", cfg.Pipeline.Effort)
	}
}

func TestLoad_FullRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")

	content := `[pipeline]
model = "opus"
effort = "high"
max_iterations = 20
max_parallel = 4
slice_timeout = "30m"
permission_mode = "bypassPermissions"

[pipeline.prompts]
dir = ".ralph/prompts"

[doctor]
require_claude_cli = true
require_go = false
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Pipeline.Model != "opus" {
		t.Errorf("model = %q", cfg.Pipeline.Model)
	}
	if cfg.Pipeline.SliceTimeout != "30m" {
		t.Errorf("slice_timeout = %q", cfg.Pipeline.SliceTimeout)
	}
	if cfg.Doctor.RequireGo {
		t.Error("require_go should be false")
	}
}

// TestDefault_PermissionMode asserts that Default() returns bypassPermissions,
// matching the template toml and the shell default in scripts/ralph-config.sh.
func TestDefault_PermissionMode(t *testing.T) {
	cfg := Default()
	if cfg.Pipeline.PermissionMode != "bypassPermissions" {
		t.Errorf("Default().Pipeline.PermissionMode = %q, want bypassPermissions", cfg.Pipeline.PermissionMode)
	}
}

// TestLoad_PermissionModeBackfill verifies that a ralph.toml without
// permission_mode yields the bypassPermissions default (not the zero string).
func TestLoad_PermissionModeBackfill(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[pipeline]
model = "opus"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Pipeline.PermissionMode != "bypassPermissions" {
		t.Errorf("PermissionMode after backfill = %q, want bypassPermissions", cfg.Pipeline.PermissionMode)
	}
}

// TestDefault_Loop verifies the [loop] defaults stay claude-driven so existing
// users see no behaviour change after Phase 2 lands.
func TestDefault_Loop(t *testing.T) {
	cfg := Default()
	if cfg.Loop.Driver != "claude" {
		t.Errorf("loop.driver default = %q, want claude", cfg.Loop.Driver)
	}
	if cfg.Loop.CodexSandbox != "workspace-write" {
		t.Errorf("loop.codex_sandbox default = %q, want workspace-write", cfg.Loop.CodexSandbox)
	}
	if cfg.Loop.CodexApprovalPolicy != "on-failure" {
		t.Errorf("loop.codex_approval_policy default = %q, want on-failure", cfg.Loop.CodexApprovalPolicy)
	}
}

func TestLoad_LoopCodexDriver(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[loop]
driver = "codex"
codex_sandbox = "read-only"
codex_approval_policy = "untrusted"
claude_reviewer_model = "claude-sonnet-4-6"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Loop.Driver != "codex" {
		t.Errorf("driver = %q, want codex", cfg.Loop.Driver)
	}
	if cfg.Loop.CodexSandbox != "read-only" {
		t.Errorf("codex_sandbox = %q, want read-only", cfg.Loop.CodexSandbox)
	}
	if cfg.Loop.CodexApprovalPolicy != "untrusted" {
		t.Errorf("codex_approval_policy = %q, want untrusted", cfg.Loop.CodexApprovalPolicy)
	}
	if cfg.Loop.ClaudeReviewerModel != "claude-sonnet-4-6" {
		t.Errorf("claude_reviewer_model = %q, want claude-sonnet-4-6", cfg.Loop.ClaudeReviewerModel)
	}
}

func TestLoad_LoopRejectsInvalidDriver(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantSub string
	}{
		{"unknown driver", `[loop]` + "\n" + `driver = "foo"` + "\n", "[loop].driver"},
		{"unknown sandbox", `[loop]` + "\n" + `codex_sandbox = "bogus"` + "\n", "[loop].codex_sandbox"},
		{"unknown approval", `[loop]` + "\n" + `codex_approval_policy = "asap"` + "\n", "[loop].codex_approval_policy"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ralph.toml")
			if err := os.WriteFile(path, []byte(tc.body), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil {
				t.Fatalf("Load: expected error for %s, got nil", tc.name)
			}
			if !contains(err.Error(), tc.wantSub) {
				t.Errorf("error %q does not mention %q", err.Error(), tc.wantSub)
			}
		})
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// TestDefault_Phases verifies the [pipeline.phases] defaults mirror the
// RALPH_<PHASE>_MODEL shell defaults in scripts/ralph-config.sh — the two
// must change in lock-step (see .claude/rules/model-routing.md).
func TestDefault_Phases(t *testing.T) {
	p := Default().Pipeline.Phases
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"implement", p.Implement, "sonnet"},
		{"self_review", p.SelfReview, "opus"},
		{"verify", p.Verify, "sonnet"},
		{"test", p.Test, "sonnet"},
		{"sync_docs", p.SyncDocs, "sonnet"},
		{"pr", p.PR, "sonnet"},
		{"probe", p.Probe, "haiku"},
		{"escalation", p.Escalation, "opus"},
		// force is an override knob, not a default: empty means "no override".
		{"force", p.Force, ""},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("phases.%s default = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

// TestLoad_PhasesAbsent verifies that a ralph.toml without [pipeline.phases]
// yields all phase defaults (backward compatibility for existing projects).
func TestLoad_PhasesAbsent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[pipeline]
model = "opus"
max_parallel = 8
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Pipeline.Phases != Default().Pipeline.Phases {
		t.Errorf("phases = %+v, want defaults %+v", cfg.Pipeline.Phases, Default().Pipeline.Phases)
	}
}

// TestLoad_PhasesPartialBackfill verifies that a partial [pipeline.phases]
// section keeps the explicit value and backfills every other key — except
// force, which must stay empty because empty is meaningful (no override).
func TestLoad_PhasesPartialBackfill(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[pipeline.phases]
implement = "opus"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Pipeline.Phases.Implement != "opus" {
		t.Errorf("implement = %q, want opus (explicit value)", cfg.Pipeline.Phases.Implement)
	}
	if cfg.Pipeline.Phases.SelfReview != "opus" {
		t.Errorf("self_review = %q, want opus (backfilled)", cfg.Pipeline.Phases.SelfReview)
	}
	if cfg.Pipeline.Phases.Verify != "sonnet" {
		t.Errorf("verify = %q, want sonnet (backfilled)", cfg.Pipeline.Phases.Verify)
	}
	if cfg.Pipeline.Phases.Probe != "haiku" {
		t.Errorf("probe = %q, want haiku (backfilled)", cfg.Pipeline.Phases.Probe)
	}
	if cfg.Pipeline.Phases.Force != "" {
		t.Errorf("force = %q, want empty (never backfilled)", cfg.Pipeline.Phases.Force)
	}
}

// TestLoad_PhasesFullRoundTrip verifies every [pipeline.phases] key, including
// force, survives a full parse unchanged.
func TestLoad_PhasesFullRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[pipeline.phases]
implement   = "haiku"
self_review = "sonnet"
verify      = "opus"
test        = "opus"
sync_docs   = "haiku"
pr          = "haiku"
probe       = "sonnet"
escalation  = "sonnet"
force       = "opus"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := PhaseModelConfig{
		Implement:  "haiku",
		SelfReview: "sonnet",
		Verify:     "opus",
		Test:       "opus",
		SyncDocs:   "haiku",
		PR:         "haiku",
		Probe:      "sonnet",
		Escalation: "sonnet",
		Force:      "opus",
	}
	if cfg.Pipeline.Phases != want {
		t.Errorf("phases = %+v, want %+v", cfg.Pipeline.Phases, want)
	}
}

// TestLoad_TemplateRalphToml verifies the shipped templates/base/ralph.toml
// parses cleanly and its [pipeline.phases] values match the Go defaults —
// catching drift between the scaffolded file and Default(). Repo-root
// discovery follows the runtime.Caller pattern used by
// internal/scaffold/embed_test.go.
//
// AC4b: also asserts that [pipeline] permission_mode equals Default()'s value
// so the template, the Go default, and the shell default stay in lock-step.
// Upgrade-path note: managed ralph.toml files auto-update on `ralph upgrade`;
// locally-edited ones surface as conflict/skip per the upgrade engine.
func TestLoad_TemplateRalphToml(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	// thisFile is internal/config/config_test.go → repo root is ../../
	path := filepath.Join(filepath.Dir(thisFile), "..", "..", "templates", "base", "ralph.toml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("templates/base/ralph.toml not found: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(templates/base/ralph.toml): %v", err)
	}
	if cfg.Pipeline.Phases != Default().Pipeline.Phases {
		t.Errorf("template phases = %+v, want defaults %+v (template and Default() must stay in lock-step)",
			cfg.Pipeline.Phases, Default().Pipeline.Phases)
	}
	// AC4b: template permission_mode must equal Default() so all entry points agree.
	if cfg.Pipeline.PermissionMode != Default().Pipeline.PermissionMode {
		t.Errorf("template permission_mode = %q, want %q (template and Default() must stay in lock-step)",
			cfg.Pipeline.PermissionMode, Default().Pipeline.PermissionMode)
	}
}

// TestLoad_RequireCodexCLI verifies the new toml field round-trips. The doctor
// command relies on this knob to switch between warn (default) and fail when
// the codex binary is missing.
func TestLoad_RequireCodexCLI(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")

	content := `[doctor]
require_claude_cli = false
require_codex_cli = true
require_go = false
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Doctor.RequireCodexCLI {
		t.Error("require_codex_cli = false after loading explicit true")
	}
	if cfg.Doctor.RequireClaudeCLI {
		t.Error("require_claude_cli = true after loading explicit false")
	}
}

// TestDefault_Org verifies the [org] envelope defaults (AC-6): driver_pool,
// model_pool, roles, max_seats, budget, and deadman_minutes.
func TestDefault_Org(t *testing.T) {
	o := Default().Org
	if len(o.DriverPool) != 2 || o.DriverPool[0] != "claude" || o.DriverPool[1] != "codex" {
		t.Errorf("driver_pool = %v, want [claude codex]", o.DriverPool)
	}
	wantModelPool := []OrgModelPoolEntry{
		{Driver: "claude", Model: "opus"},
		{Driver: "claude", Model: "sonnet"},
		{Driver: "claude", Model: "haiku"},
	}
	if len(o.ModelPool) != len(wantModelPool) {
		t.Fatalf("model_pool = %+v, want %+v", o.ModelPool, wantModelPool)
	}
	for i, e := range wantModelPool {
		if o.ModelPool[i] != e {
			t.Errorf("model_pool[%d] = %+v, want %+v", i, o.ModelPool[i], e)
		}
	}
	if len(o.Roles) != 0 {
		t.Errorf("roles = %v, want empty", o.Roles)
	}
	if o.MaxSeats != 5 {
		t.Errorf("max_seats = %d, want 5", o.MaxSeats)
	}
	if o.Budget.SeatWallClockMinutes != 30 {
		t.Errorf("budget.seat_wall_clock_minutes = %d, want 30", o.Budget.SeatWallClockMinutes)
	}
	if o.Budget.TotalWallClockMinutes != 120 {
		t.Errorf("budget.total_wall_clock_minutes = %d, want 120", o.Budget.TotalWallClockMinutes)
	}
	if o.Budget.MaxFixRounds != 2 {
		t.Errorf("budget.max_fix_rounds = %d, want 2", o.Budget.MaxFixRounds)
	}
	if o.DeadmanMinutes != 10 {
		t.Errorf("deadman_minutes = %d, want 10", o.DeadmanMinutes)
	}
	if o.AgmsgHome != "~/.agents/skills/agmsg" {
		t.Errorf("agmsg_home = %q, want %q", o.AgmsgHome, "~/.agents/skills/agmsg")
	}
}

// TestLoad_OrgMissingSection verifies AC-9 compatibility: a ralph.toml
// without [org] at all loads unchanged with the full Org defaults and no
// error/warning.
func TestLoad_OrgMissingSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[pipeline]
model = "opus"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// OrgConfig embeds slices/maps and is therefore not comparable with ==/!=;
	// compare the fields that matter instead of the whole struct.
	want := Default().Org
	if len(cfg.Org.ModelPool) != len(want.ModelPool) {
		t.Fatalf("model_pool = %+v, want %+v", cfg.Org.ModelPool, want.ModelPool)
	}
	for i := range want.ModelPool {
		if cfg.Org.ModelPool[i] != want.ModelPool[i] {
			t.Errorf("model_pool[%d] = %+v, want %+v", i, cfg.Org.ModelPool[i], want.ModelPool[i])
		}
	}
	if strings.Join(cfg.Org.DriverPool, ",") != strings.Join(want.DriverPool, ",") {
		t.Errorf("driver_pool = %v, want %v", cfg.Org.DriverPool, want.DriverPool)
	}
	if len(cfg.Org.Roles) != len(want.Roles) {
		t.Errorf("roles = %v, want %v", cfg.Org.Roles, want.Roles)
	}
	if cfg.Org.MaxSeats != want.MaxSeats {
		t.Errorf("max_seats = %d, want %d", cfg.Org.MaxSeats, want.MaxSeats)
	}
	if cfg.Org.Budget != want.Budget {
		t.Errorf("budget = %+v, want %+v", cfg.Org.Budget, want.Budget)
	}
	if cfg.Org.DeadmanMinutes != want.DeadmanMinutes {
		t.Errorf("deadman_minutes = %d, want %d", cfg.Org.DeadmanMinutes, want.DeadmanMinutes)
	}
	if cfg.Org.AgmsgHome != want.AgmsgHome {
		t.Errorf("agmsg_home = %q, want %q", cfg.Org.AgmsgHome, want.AgmsgHome)
	}
}

// TestLoad_OrgRolesEmpty verifies an explicit, empty [org.roles] table is
// allowed (means "no role restriction" — full model_pool usable by every role).
func TestLoad_OrgRolesEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[org]
[org.roles]
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Org.Roles) != 0 {
		t.Errorf("roles = %v, want empty", cfg.Org.Roles)
	}
	// model_pool must still fall back to the default pool.
	if len(cfg.Org.ModelPool) != 3 {
		t.Errorf("model_pool = %+v, want 3 default entries", cfg.Org.ModelPool)
	}
}

// TestLoad_OrgFullRoundTrip verifies a fully specified [org] section
// (including [org.roles] and [org.budget]) round-trips through Load()
// unchanged.
func TestLoad_OrgFullRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[org]
driver_pool = ["claude"]
model_pool = [
  { driver = "claude", model = "opus" },
  { driver = "claude", model = "haiku" },
]
max_seats = 3
deadman_minutes = 15
agmsg_home = "~/custom/agmsg-home"

[org.roles]
reviewer = ["opus"]

[org.budget]
seat_wall_clock_minutes = 45
total_wall_clock_minutes = 90
max_fix_rounds = 1
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Org.DriverPool) != 1 || cfg.Org.DriverPool[0] != "claude" {
		t.Errorf("driver_pool = %v, want [claude]", cfg.Org.DriverPool)
	}
	if len(cfg.Org.ModelPool) != 2 {
		t.Fatalf("model_pool = %+v, want 2 entries", cfg.Org.ModelPool)
	}
	if cfg.Org.MaxSeats != 3 {
		t.Errorf("max_seats = %d, want 3", cfg.Org.MaxSeats)
	}
	if cfg.Org.DeadmanMinutes != 15 {
		t.Errorf("deadman_minutes = %d, want 15", cfg.Org.DeadmanMinutes)
	}
	if models := cfg.Org.Roles["reviewer"]; len(models) != 1 || models[0] != "opus" {
		t.Errorf("roles[reviewer] = %v, want [opus]", models)
	}
	if cfg.Org.Budget.SeatWallClockMinutes != 45 {
		t.Errorf("budget.seat_wall_clock_minutes = %d, want 45", cfg.Org.Budget.SeatWallClockMinutes)
	}
	if cfg.Org.Budget.TotalWallClockMinutes != 90 {
		t.Errorf("budget.total_wall_clock_minutes = %d, want 90", cfg.Org.Budget.TotalWallClockMinutes)
	}
	if cfg.Org.Budget.MaxFixRounds != 1 {
		t.Errorf("budget.max_fix_rounds = %d, want 1", cfg.Org.Budget.MaxFixRounds)
	}
	if cfg.Org.AgmsgHome != "~/custom/agmsg-home" {
		t.Errorf("agmsg_home = %q, want %q", cfg.Org.AgmsgHome, "~/custom/agmsg-home")
	}
}

// TestLoad_OrgAgmsgHomeExplicitEmptyBackfillsDefault verifies that an
// explicit `agmsg_home = ""` in the source document falls back to the
// default rather than being treated as a validation error -- AgmsgHome
// follows PipelineConfig's zero-value backfill pattern, not the strict
// "[org] fields must be explicitly valid" pattern used by max_seats etc.
func TestLoad_OrgAgmsgHomeExplicitEmptyBackfillsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[org]
agmsg_home = ""
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Org.AgmsgHome != Default().Org.AgmsgHome {
		t.Errorf("agmsg_home = %q, want default %q", cfg.Org.AgmsgHome, Default().Org.AgmsgHome)
	}
}

// TestLoad_OrgRejects is a table-driven check of every [org] validation
// rejection: driver absent from driver_pool, duplicate model_pool entries,
// explicitly empty model_pool, roles referencing an unknown model, and
// out-of-range budget values.
func TestLoad_OrgRejects(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantSub string
	}{
		{
			name: "driver not in driver_pool",
			body: `[org]
driver_pool = ["claude"]
model_pool = [{ driver = "codex", model = "gpt-5" }]
`,
			wantSub: "driver_pool",
		},
		{
			name: "duplicate model_pool entry",
			body: `[org]
model_pool = [
  { driver = "claude", model = "opus" },
  { driver = "claude", model = "opus" },
]
`,
			wantSub: "duplicate",
		},
		{
			name: "explicitly empty model_pool",
			body: `[org]
model_pool = []
`,
			wantSub: "model_pool",
		},
		{
			name: "roles reference unknown model",
			body: `[org]
model_pool = [{ driver = "claude", model = "opus" }]

[org.roles]
reviewer = ["sonnet"]
`,
			wantSub: "reviewer",
		},
		{
			name: "seat_wall_clock_minutes below 1",
			body: `[org.budget]
seat_wall_clock_minutes = 0
`,
			wantSub: "seat_wall_clock_minutes",
		},
		{
			name: "total_wall_clock_minutes below 1",
			body: `[org.budget]
total_wall_clock_minutes = 0
`,
			wantSub: "total_wall_clock_minutes",
		},
		{
			name: "max_fix_rounds below 1",
			body: `[org.budget]
max_fix_rounds = 0
`,
			wantSub: "max_fix_rounds",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ralph.toml")
			if err := os.WriteFile(path, []byte(tc.body), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil {
				t.Fatalf("Load: expected error for %s, got nil", tc.name)
			}
			if !contains(err.Error(), tc.wantSub) {
				t.Errorf("error %q does not mention %q", err.Error(), tc.wantSub)
			}
		})
	}
}

// TestDefault_OrgPermissions verifies [org.permissions] defaults: every
// role uses "autonomous" and no per-role override exists out of the box.
func TestDefault_OrgPermissions(t *testing.T) {
	p := Default().Org.Permissions
	if p.Default != "autonomous" {
		t.Errorf("permissions.default = %q, want autonomous", p.Default)
	}
	if len(p.Roles) != 0 {
		t.Errorf("permissions.roles = %v, want empty", p.Roles)
	}
}

// TestLoad_OrgPermissionsRoleOverrideRoundTrip verifies an explicit
// [org.permissions] section (default override plus a per-role override)
// round-trips through Load() unchanged.
func TestLoad_OrgPermissionsRoleOverrideRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[org.permissions]
default = "edits"

[org.permissions.roles]
reviewer = "guarded"
lead = "autonomous"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Org.Permissions.Default != "edits" {
		t.Errorf("permissions.default = %q, want edits", cfg.Org.Permissions.Default)
	}
	if cfg.Org.Permissions.Roles["reviewer"] != "guarded" {
		t.Errorf("permissions.roles[reviewer] = %q, want guarded", cfg.Org.Permissions.Roles["reviewer"])
	}
	if cfg.Org.Permissions.Roles["lead"] != "autonomous" {
		t.Errorf("permissions.roles[lead] = %q, want autonomous", cfg.Org.Permissions.Roles["lead"])
	}
}

// TestLoad_OrgPermissionsMissingSectionBackfillsDefault verifies AC-1
// compatibility: a ralph.toml without [org.permissions] at all loads with
// the "autonomous" default and no error.
func TestLoad_OrgPermissionsMissingSectionBackfillsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[pipeline]
model = "opus"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Org.Permissions.Default != "autonomous" {
		t.Errorf("permissions.default = %q, want autonomous", cfg.Org.Permissions.Default)
	}
}

// TestLoad_OrgPermissionsRejectsInvalidMode is a table-driven check that an
// invalid mode is rejected both on [org.permissions].default and on a
// [org.permissions.roles] entry.
func TestLoad_OrgPermissionsRejectsInvalidMode(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantSub string
	}{
		{
			name:    "invalid default",
			body:    "[org.permissions]\ndefault = \"yolo\"\n",
			wantSub: "[org.permissions].default",
		},
		{
			name:    "invalid role override",
			body:    "[org.permissions.roles]\nreviewer = \"yolo\"\n",
			wantSub: "[org.permissions.roles].reviewer",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ralph.toml")
			if err := os.WriteFile(path, []byte(tc.body), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil {
				t.Fatalf("Load: expected error for %s, got nil", tc.name)
			}
			if !contains(err.Error(), tc.wantSub) {
				t.Errorf("error %q does not mention %q", err.Error(), tc.wantSub)
			}
		})
	}
}

// TestLoad_OrgMaxSeatsBoundary verifies the max_seats >= 1 boundary: 0 is
// rejected, 1 is accepted.
func TestLoad_OrgMaxSeatsBoundary(t *testing.T) {
	t.Run("zero rejected", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "ralph.toml")
		content := `[org]
max_seats = 0
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := Load(path)
		if err == nil {
			t.Fatal("Load: expected error for max_seats = 0, got nil")
		}
		if !contains(err.Error(), "max_seats") {
			t.Errorf("error %q does not mention max_seats", err.Error())
		}
	})
	t.Run("one accepted", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "ralph.toml")
		content := `[org]
max_seats = 1
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load: unexpected error for max_seats = 1: %v", err)
		}
		if cfg.Org.MaxSeats != 1 {
			t.Errorf("max_seats = %d, want 1", cfg.Org.MaxSeats)
		}
	})
}

// TestDefault_OrgWatchdog verifies the [org.watchdog] envelope defaults:
// interval_seconds, stall_minutes, watcher_enabled, watcher_model.
func TestDefault_OrgWatchdog(t *testing.T) {
	w := Default().Org.Watchdog
	if w.IntervalSeconds != 30 {
		t.Errorf("watchdog.interval_seconds = %d, want 30", w.IntervalSeconds)
	}
	if w.StallMinutes != 15 {
		t.Errorf("watchdog.stall_minutes = %d, want 15", w.StallMinutes)
	}
	if !w.WatcherEnabled {
		t.Error("watchdog.watcher_enabled = false, want true")
	}
	if w.WatcherModel != "haiku" {
		t.Errorf("watchdog.watcher_model = %q, want haiku", w.WatcherModel)
	}
}

// TestLoad_OrgWatchdogMissingSectionBackfillsDefault verifies a ralph.toml
// without [org.watchdog] at all loads unchanged with the full Watchdog
// defaults and no error.
func TestLoad_OrgWatchdogMissingSectionBackfillsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[pipeline]
model = "opus"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Org.Watchdog != Default().Org.Watchdog {
		t.Errorf("watchdog = %+v, want %+v", cfg.Org.Watchdog, Default().Org.Watchdog)
	}
}

// TestLoad_OrgWatchdogRoundTrip verifies a fully specified [org.watchdog]
// section round-trips through Load() unchanged.
func TestLoad_OrgWatchdogRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[org.watchdog]
interval_seconds = 45
stall_minutes = 20
watcher_enabled = false
watcher_model = "sonnet"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Org.Watchdog.IntervalSeconds != 45 {
		t.Errorf("watchdog.interval_seconds = %d, want 45", cfg.Org.Watchdog.IntervalSeconds)
	}
	if cfg.Org.Watchdog.StallMinutes != 20 {
		t.Errorf("watchdog.stall_minutes = %d, want 20", cfg.Org.Watchdog.StallMinutes)
	}
	if cfg.Org.Watchdog.WatcherEnabled {
		t.Error("watchdog.watcher_enabled = true, want false (explicit override)")
	}
	if cfg.Org.Watchdog.WatcherModel != "sonnet" {
		t.Errorf("watchdog.watcher_model = %q, want sonnet", cfg.Org.Watchdog.WatcherModel)
	}
}

// TestLoad_OrgWatchdogRejects is a table-driven check of every
// [org.watchdog] validation rejection: interval_seconds/stall_minutes below
// 1, and an empty watcher_model while watcher_enabled is true.
func TestLoad_OrgWatchdogRejects(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantSub string
	}{
		{
			name: "interval_seconds below 1",
			body: `[org.watchdog]
interval_seconds = 0
`,
			wantSub: "interval_seconds",
		},
		{
			name: "stall_minutes below 1",
			body: `[org.watchdog]
stall_minutes = 0
`,
			wantSub: "stall_minutes",
		},
		{
			name: "watcher_model empty while watcher_enabled true",
			body: `[org.watchdog]
watcher_enabled = true
watcher_model = ""
`,
			wantSub: "watcher_model",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "ralph.toml")
			if err := os.WriteFile(path, []byte(tc.body), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil {
				t.Fatalf("Load: expected error for %s, got nil", tc.name)
			}
			if !contains(err.Error(), tc.wantSub) {
				t.Errorf("error %q does not mention %q", err.Error(), tc.wantSub)
			}
		})
	}
}

// TestLoad_OrgWatchdogWatcherDisabledAllowsEmptyModel verifies that
// watcher_model may be empty when watcher_enabled is explicitly false (the
// non-empty requirement only applies when the watcher layer is active).
func TestLoad_OrgWatchdogWatcherDisabledAllowsEmptyModel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[org.watchdog]
watcher_enabled = false
watcher_model = ""
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	if cfg.Org.Watchdog.WatcherEnabled {
		t.Error("watchdog.watcher_enabled = true, want false")
	}
	if cfg.Org.Watchdog.WatcherModel != "" {
		t.Errorf("watchdog.watcher_model = %q, want empty", cfg.Org.Watchdog.WatcherModel)
	}
}
