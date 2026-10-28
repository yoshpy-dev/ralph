package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	// Neither agent CLI is required by default, so a project with only
	// claude or only codex installed does not see `ralph doctor` exit
	// non-zero.
	if cfg.Doctor.RequireClaudeCLI {
		t.Error("require_claude_cli should default to false")
	}
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
	if cfg.Doctor.RequireClaudeCLI {
		t.Error("require_claude_cli should default to false")
	}
}

func TestLoad_FullRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")

	content := `[doctor]
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

	if !cfg.Doctor.RequireClaudeCLI {
		t.Error("require_claude_cli should be true")
	}
	if cfg.Doctor.RequireGo {
		t.Error("require_go should be false")
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

// TestLoad_TemplateRalphToml verifies the shipped templates/base/ralph.toml
// parses cleanly under the current Config schema — catching drift between
// the scaffolded file and Load()/Default(). Repo-root discovery follows the
// runtime.Caller pattern used by internal/scaffold/embed_test.go.
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
	// TestDefaultsLockStep checks the [org] keys orgTemplateDrift reads
	// (auto_merge and wake included) against Default(). It does not read
	// [org.roles], [org.permissions].roles, or launch_cmd; wake_cmd is below.
	if len(cfg.Org.Director.WakeCmd) != 0 {
		t.Errorf("templates/base/ralph.toml director.wake_cmd = %q, want empty (the example stays a comment)", cfg.Org.Director.WakeCmd)
	}
	// [doctor]'s require_claude_cli and require_codex_cli must be written out
	// in the template: ralph.toml is a seed that ralph upgrade does not
	// overwrite, so a scaffolded project keeps the values the template writes,
	// not Default()'s. Load keeps Default()'s value for a key the template
	// leaves out, so the raw text is checked as well.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\nrequire_claude_cli = false\n") {
		t.Error("templates/base/ralph.toml does not set require_claude_cli = false in [doctor]")
	}
	if got, want := cfg.Doctor.RequireClaudeCLI, Default().Doctor.RequireClaudeCLI; got != want {
		t.Errorf("templates/base/ralph.toml doctor.require_claude_cli = %v, want Default()'s %v", got, want)
	}
	if !strings.Contains(string(data), "\nrequire_codex_cli = false\n") {
		t.Error("templates/base/ralph.toml does not set require_codex_cli = false in [doctor]")
	}
	if got, want := cfg.Doctor.RequireCodexCLI, Default().Doctor.RequireCodexCLI; got != want {
		t.Errorf("templates/base/ralph.toml doctor.require_codex_cli = %v, want Default()'s %v", got, want)
	}
}

// TestLoad_CodexVerifiedAndAutoMergeDefaultToTrue verifies that a project
// with no ralph.toml, and one whose ralph.toml leaves out
// [org.permissions].codex_verified and [org.director].auto_merge, loads both
// as true (issue #274; plan 2026-10-28-codex-verified-auto-merge-on, AC1).
func TestLoad_CodexVerifiedAndAutoMergeDefaultToTrue(t *testing.T) {
	cases := []struct {
		name    string
		content string // "" writes no ralph.toml
	}{
		{name: "no ralph.toml"},
		{name: "a ralph.toml without [org]", content: "[doctor]\nrequire_go = false\n"},
		{name: "both tables without the keys", content: "[org.permissions]\ndefault = \"edits\"\n\n[org.director]\nack_minutes = 5\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "ralph.toml")
			if c.content != "" {
				if err := os.WriteFile(path, []byte(c.content), 0644); err != nil {
					t.Fatal(err)
				}
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if !cfg.Org.Permissions.CodexVerified {
				t.Error("permissions.codex_verified = false, want true")
			}
			if !cfg.Org.Director.AutoMerge {
				t.Error("director.auto_merge = false, want true")
			}
		})
	}
}

// TestLoad_TemplateWithBothKeysFalse verifies the opt-out, which is also how
// a project goes back to the old behavior (plan
// 2026-10-28-codex-verified-auto-merge-on, AC7 and its rollback note):
// templates/base/ralph.toml with its codex_verified = true and
// auto_merge = true lines rewritten to false loads both as false, and every
// other value as the unchanged template does.
func TestLoad_TemplateWithBothKeysFalse(t *testing.T) {
	templatePath := filepath.Join(repoRoot(t), "templates", "base", "ralph.toml")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("cannot read templates/base/ralph.toml: %v", err)
	}
	text := string(data)
	for _, key := range []string{"codex_verified", "auto_merge"} {
		old := "\n" + key + " = true\n"
		if n := strings.Count(text, old); n != 1 {
			t.Fatalf("templates/base/ralph.toml contains %q %d times, want exactly 1", old, n)
		}
		text = strings.Replace(text, old, "\n"+key+" = false\n", 1)
	}
	path := filepath.Join(t.TempDir(), "ralph.toml")
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(the template with both keys false): %v", err)
	}
	if cfg.Org.Permissions.CodexVerified {
		t.Error("permissions.codex_verified = true, want false")
	}
	if cfg.Org.Director.AutoMerge {
		t.Error("director.auto_merge = true, want false")
	}
	want, err := Load(templatePath)
	if err != nil {
		t.Fatalf("Load(templates/base/ralph.toml): %v", err)
	}
	want.Org.Permissions.CodexVerified = false
	want.Org.Director.AutoMerge = false
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("config = %+v,\nwant the template's with only the two keys false = %+v", cfg, want)
	}
}

// TestLoad_RequireAgentCLIs verifies require_claude_cli and require_codex_cli
// round-trip with true, the value that differs from Default(). The doctor
// command reads them to fail when that CLI is not usable, instead of warning
// or deferring to the other CLI.
func TestLoad_RequireAgentCLIs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")

	content := `[doctor]
require_claude_cli = true
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
	if !cfg.Doctor.RequireClaudeCLI {
		t.Error("require_claude_cli = false after loading explicit true")
	}
}

// TestDefault_Org verifies the [org] envelope defaults (AC-6): driver_pool,
// model_pool, roles, max_seats, max_orgs, max_total_seats, and
// deadman_minutes.
func TestDefault_Org(t *testing.T) {
	o := Default().Org
	if len(o.DriverPool) != 2 || o.DriverPool[0] != "claude" || o.DriverPool[1] != "codex" {
		t.Errorf("driver_pool = %v, want [claude codex]", o.DriverPool)
	}
	wantModelPool := []OrgModelPoolEntry{
		{Driver: "claude", Model: "fable"},
		{Driver: "claude", Model: "opus"},
		{Driver: "claude", Model: "sonnet"},
		{Driver: "claude", Model: "haiku"},
		{Driver: "codex", Model: "gpt-6-astra"},
		{Driver: "codex", Model: "gpt-5.6-sol"},
		{Driver: "codex", Model: "gpt-5.6-terra"},
		{Driver: "codex", Model: "gpt-5.6-luna"},
	}
	// gpt-5.5 was dropped from the default on 2026-10-02 (#156). This check
	// runs before the length check below, which t.Fatalf's on a mismatch, so a
	// regression that re-adds it fails here by name as well.
	for _, e := range o.ModelPool {
		if e.Model == "gpt-5.5" {
			t.Errorf("model_pool contains dropped default %+v, want it absent", e)
		}
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
	if o.MaxOrgs != 10 {
		t.Errorf("max_orgs = %d, want 10", o.MaxOrgs)
	}
	if o.MaxTotalSeats != 30 {
		t.Errorf("max_total_seats = %d, want 30", o.MaxTotalSeats)
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
	if cfg.Org.MaxOrgs != want.MaxOrgs {
		t.Errorf("max_orgs = %d, want %d", cfg.Org.MaxOrgs, want.MaxOrgs)
	}
	if cfg.Org.MaxTotalSeats != want.MaxTotalSeats {
		t.Errorf("max_total_seats = %d, want %d", cfg.Org.MaxTotalSeats, want.MaxTotalSeats)
	}
	if cfg.Org.DeadmanMinutes != want.DeadmanMinutes {
		t.Errorf("deadman_minutes = %d, want %d", cfg.Org.DeadmanMinutes, want.DeadmanMinutes)
	}
	if cfg.Org.AgmsgHome != want.AgmsgHome {
		t.Errorf("agmsg_home = %q, want %q", cfg.Org.AgmsgHome, want.AgmsgHome)
	}
	// OrgDirectorConfig holds a slice (LaunchCmd), so it is compared with
	// reflect.DeepEqual rather than !=.
	if !reflect.DeepEqual(cfg.Org.Director, want.Director) {
		t.Errorf("director = %+v, want %+v", cfg.Org.Director, want.Director)
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
	if len(cfg.Org.ModelPool) != 8 {
		t.Errorf("model_pool = %+v, want 8 default entries", cfg.Org.ModelPool)
	}
}

// TestLoad_OrgFullRoundTrip verifies a fully specified [org] section
// (including [org.roles]) round-trips through Load() unchanged.
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
max_orgs = 4
max_total_seats = 12
deadman_minutes = 15
agmsg_home = "~/custom/agmsg-home"

[org.roles]
reviewer = ["opus"]
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
	if cfg.Org.MaxOrgs != 4 {
		t.Errorf("max_orgs = %d, want 4", cfg.Org.MaxOrgs)
	}
	if cfg.Org.MaxTotalSeats != 12 {
		t.Errorf("max_total_seats = %d, want 12", cfg.Org.MaxTotalSeats)
	}
	if cfg.Org.DeadmanMinutes != 15 {
		t.Errorf("deadman_minutes = %d, want 15", cfg.Org.DeadmanMinutes)
	}
	if models := cfg.Org.Roles["reviewer"]; len(models) != 1 || models[0] != "opus" {
		t.Errorf("roles[reviewer] = %v, want [opus]", models)
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
// explicitly empty model_pool, and roles referencing an unknown model.
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

// TestLoad_IgnoresRetiredOrgBudgetTable verifies that a ralph.toml which
// still carries the retired [org.budget] table (seat_wall_clock_minutes/
// total_wall_clock_minutes/max_fix_rounds -- removed together with the org
// budget concept) loads without error: go-toml's Unmarshal ignores TOML
// keys/tables with no matching Go struct field, so an unrecognized
// [org.budget] table is silently skipped rather than rejected. This is what
// lets a downstream project's existing ralph.toml (written before this
// removal) keep working unchanged after an upgrade.
func TestLoad_IgnoresRetiredOrgBudgetTable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[org.budget]
seat_wall_clock_minutes = 30
total_wall_clock_minutes = 120
max_fix_rounds = 2
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		t.Fatalf("Load: expected a retired [org.budget] table to be silently ignored, got error: %v", err)
	}
}

// TestDefault_OrgPermissions verifies [org.permissions] defaults: every
// role uses "autonomous", no per-role override exists out of the box, and
// codex_verified is true (issue #274).
func TestDefault_OrgPermissions(t *testing.T) {
	p := Default().Org.Permissions
	if p.Default != "autonomous" {
		t.Errorf("permissions.default = %q, want autonomous", p.Default)
	}
	if len(p.Roles) != 0 {
		t.Errorf("permissions.roles = %v, want empty", p.Roles)
	}
	if !p.CodexVerified {
		t.Errorf("permissions.codex_verified = %v, want true (the default since issue #274)", p.CodexVerified)
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
leader = "autonomous"
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
	if cfg.Org.Permissions.Roles["leader"] != "autonomous" {
		t.Errorf("permissions.roles[leader] = %q, want autonomous", cfg.Org.Permissions.Roles["leader"])
	}
}

// TestLoad_OrgPermissionsMissingSectionBackfillsDefault verifies AC-1
// compatibility: a ralph.toml without [org.permissions] at all loads with
// the "autonomous" default, codex_verified true, and no error.
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
	if !cfg.Org.Permissions.CodexVerified {
		t.Errorf("permissions.codex_verified = %v, want true", cfg.Org.Permissions.CodexVerified)
	}
}

// TestLoad_OrgPermissionsCodexVerifiedRoundTrip verifies AC-8's config gate:
// an explicit [org.permissions].codex_verified round-trips through Load()
// unchanged, false (the opt-out that keeps codex seats fail-closed to
// guarded) as well as true.
func TestLoad_OrgPermissionsCodexVerifiedRoundTrip(t *testing.T) {
	for _, want := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "ralph.toml")
		content := fmt.Sprintf("[org.permissions]\ncodex_verified = %t\n", want)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.Org.Permissions.CodexVerified != want {
			t.Errorf("codex_verified = %t: permissions.codex_verified = %v, want %v", want, cfg.Org.Permissions.CodexVerified, want)
		}
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

// TestLoad_OrgFleetLimitsBoundary verifies the >= 1 boundary of the two
// limits shared by every org_id in one state dir: an explicit 0 or negative
// max_orgs / max_total_seats is rejected with an error naming that key, and 1
// is accepted. Setting one of them leaves the other at its default.
func TestLoad_OrgFleetLimitsBoundary(t *testing.T) {
	for _, key := range []string{"max_orgs", "max_total_seats"} {
		for _, bad := range []int{0, -1} {
			t.Run(fmt.Sprintf("%s=%d rejected", key, bad), func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, "ralph.toml")
				content := fmt.Sprintf("[org]\n%s = %d\n", key, bad)
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				_, err := Load(path)
				if err == nil {
					t.Fatalf("Load: expected error for %s = %d, got nil", key, bad)
				}
				if want := "[org]." + key + " must be >= 1"; !contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err.Error(), want)
				}
			})
		}
	}

	t.Run("one accepted", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "ralph.toml")
		content := `[org]
max_orgs = 1
max_total_seats = 1
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load: unexpected error for max_orgs = 1, max_total_seats = 1: %v", err)
		}
		if cfg.Org.MaxOrgs != 1 {
			t.Errorf("max_orgs = %d, want 1", cfg.Org.MaxOrgs)
		}
		if cfg.Org.MaxTotalSeats != 1 {
			t.Errorf("max_total_seats = %d, want 1", cfg.Org.MaxTotalSeats)
		}
	})

	t.Run("one key set leaves the other at default", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "ralph.toml")
		content := `[org]
max_orgs = 2
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.Org.MaxOrgs != 2 {
			t.Errorf("max_orgs = %d, want 2", cfg.Org.MaxOrgs)
		}
		if want := Default().Org.MaxTotalSeats; cfg.Org.MaxTotalSeats != want {
			t.Errorf("max_total_seats = %d, want default %d", cfg.Org.MaxTotalSeats, want)
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

// TestDefault_OrgDirector verifies the [org.director] defaults: the director
// switch enabled is true, auto_merge is true (the director merges ready PRs
// without asking, issue #274), the inbox deadlines are ack_minutes 10 and
// resolve_minutes 60, launch_cmd is empty (start uses herdr), wake is
// ["pull"] (the director waits on the inbox itself), and wake_cmd is empty.
func TestDefault_OrgDirector(t *testing.T) {
	d := Default().Org.Director
	if !d.Enabled {
		t.Error("director.enabled = false, want true")
	}
	if !d.AutoMerge {
		t.Error("director.auto_merge = false, want true")
	}
	if d.AckMinutes != 10 {
		t.Errorf("director.ack_minutes = %d, want 10", d.AckMinutes)
	}
	if d.ResolveMinutes != 60 {
		t.Errorf("director.resolve_minutes = %d, want 60", d.ResolveMinutes)
	}
	if len(d.LaunchCmd) != 0 {
		t.Errorf("director.launch_cmd = %q, want empty", d.LaunchCmd)
	}
	if want := []string{"pull"}; !slices.Equal(d.Wake, want) {
		t.Errorf("director.wake = %q, want %q", d.Wake, want)
	}
	if len(d.WakeCmd) != 0 {
		t.Errorf("director.wake_cmd = %q, want empty", d.WakeCmd)
	}
	// Each call builds its own Wake, so a caller that changes one does not
	// change the next Default().
	d.Wake[0] = "changed"
	if again := Default().Org.Director.Wake; !slices.Equal(again, []string{"pull"}) {
		t.Errorf("Default().Org.Director.Wake after changing a returned slice = %q, want [pull]", again)
	}
}

// TestLoad_OrgDirectorMissingSectionBackfillsDefault verifies a ralph.toml
// without [org.director] (but with other [org] tables) loads with the full
// Director defaults and no error, enabled = true and auto_merge = true
// included.
func TestLoad_OrgDirectorMissingSectionBackfillsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[org]
max_seats = 3

[org.watchdog]
stall_minutes = 20
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(cfg.Org.Director, Default().Org.Director) {
		t.Errorf("director = %+v, want %+v", cfg.Org.Director, Default().Org.Director)
	}
	if !cfg.Org.Director.Enabled {
		t.Error("director.enabled = false, want true for an absent [org.director]")
	}
	if !cfg.Org.Director.AutoMerge {
		t.Error("director.auto_merge = false, want true for an absent [org.director]")
	}
}

// TestLoad_OrgDirectorRoundTrip verifies explicit [org.director] values
// round-trip through Load() unchanged, including a resolve_minutes shorter
// than ack_minutes (the two deadlines are independent), enabled = false, and
// auto_merge = false, and that setting one key leaves the others at their
// defaults (a section without enabled or auto_merge loads it as true).
func TestLoad_OrgDirectorRoundTrip(t *testing.T) {
	t.Run("all keys", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "ralph.toml")
		content := `[org.director]
enabled = false
auto_merge = false
ack_minutes = 25
resolve_minutes = 5
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.Org.Director.Enabled {
			t.Error("director.enabled = true, want false")
		}
		if cfg.Org.Director.AutoMerge {
			t.Error("director.auto_merge = true, want false")
		}
		if cfg.Org.Director.AckMinutes != 25 {
			t.Errorf("director.ack_minutes = %d, want 25", cfg.Org.Director.AckMinutes)
		}
		if cfg.Org.Director.ResolveMinutes != 5 {
			t.Errorf("director.resolve_minutes = %d, want 5", cfg.Org.Director.ResolveMinutes)
		}
	})

	t.Run("one key set leaves the other at default", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "ralph.toml")
		content := `[org.director]
ack_minutes = 1
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load: unexpected error for ack_minutes = 1: %v", err)
		}
		if cfg.Org.Director.AckMinutes != 1 {
			t.Errorf("director.ack_minutes = %d, want 1", cfg.Org.Director.AckMinutes)
		}
		if want := Default().Org.Director.ResolveMinutes; cfg.Org.Director.ResolveMinutes != want {
			t.Errorf("director.resolve_minutes = %d, want default %d", cfg.Org.Director.ResolveMinutes, want)
		}
		if !cfg.Org.Director.Enabled {
			t.Error("director.enabled = false, want true for a section without enabled")
		}
		if !cfg.Org.Director.AutoMerge {
			t.Error("director.auto_merge = false, want true for a section without auto_merge")
		}
	})

	tests := []struct {
		name          string
		content       string
		wantEnabled   bool
		wantAutoMerge bool
	}{
		{"enabled = true", "[org.director]\nenabled = true\n", true, true},
		{"enabled = false alone", "[org.director]\nenabled = false\n", false, true},
		{"auto_merge = true alone", "[org.director]\nauto_merge = true\n", true, true},
		{"auto_merge = false alone", "[org.director]\nauto_merge = false\n", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "ralph.toml")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.Org.Director.Enabled != tt.wantEnabled {
				t.Errorf("director.enabled = %v, want %v", cfg.Org.Director.Enabled, tt.wantEnabled)
			}
			if cfg.Org.Director.AutoMerge != tt.wantAutoMerge {
				t.Errorf("director.auto_merge = %v, want %v", cfg.Org.Director.AutoMerge, tt.wantAutoMerge)
			}
			def := Default().Org.Director
			if cfg.Org.Director.AckMinutes != def.AckMinutes || cfg.Org.Director.ResolveMinutes != def.ResolveMinutes {
				t.Errorf("director deadlines = %d, %d, want defaults %d, %d",
					cfg.Org.Director.AckMinutes, cfg.Org.Director.ResolveMinutes, def.AckMinutes, def.ResolveMinutes)
			}
		})
	}
}

// TestLoad_OrgDirectorAutoMergeWrongType verifies an auto_merge that is not
// a bool (a string, "true" included, or a number) fails Load at decoding
// rather than loading as either value (plan 2026-10-17-org-auto-merge, AC2).
// The decoder's message names the Go field, not the key, so only the error
// is checked, as for launch_cmd.
func TestLoad_OrgDirectorAutoMergeWrongType(t *testing.T) {
	for _, content := range []string{
		"[org.director]\nauto_merge = \"yes\"\n",
		"[org.director]\nauto_merge = \"true\"\n",
		"[org.director]\nauto_merge = 1\n",
	} {
		path := filepath.Join(t.TempDir(), "ralph.toml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		if cfg, err := Load(path); err == nil {
			t.Errorf("Load(%q): err = nil (auto_merge = %v), want a decode error", content, cfg.Org.Director.AutoMerge)
		}
	}
}

// TestLoad_OrgDirectorRejects verifies an explicit 0 or negative
// ack_minutes / resolve_minutes is rejected with an error naming that key and
// the value it got.
func TestLoad_OrgDirectorRejects(t *testing.T) {
	for _, key := range []string{"ack_minutes", "resolve_minutes"} {
		for _, bad := range []int{0, -1} {
			t.Run(fmt.Sprintf("%s=%d rejected", key, bad), func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, "ralph.toml")
				content := fmt.Sprintf("[org.director]\n%s = %d\n", key, bad)
				if err := os.WriteFile(path, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				_, err := Load(path)
				if err == nil {
					t.Fatalf("Load: expected error for %s = %d, got nil", key, bad)
				}
				if want := fmt.Sprintf("[org.director].%s must be >= 1, got %d", key, bad); err.Error() != want {
					t.Errorf("error = %q, want %q", err.Error(), want)
				}
			})
		}
	}
}

// TestLoad_OrgDirectorLaunchCmd covers [org.director].launch_cmd through
// Load() (plan 2026-10-16-org-director-launch-cmd, AC1). An absent key or an
// empty array loads as empty (start uses herdr), and a value with {script}
// or {script_sh} loads as written. A non-empty value whose first element is
// empty or whitespace only, that has a {...} outside the four placeholders
// (in any element), or that has neither {script} nor {script_sh}, fails with
// an error that starts with the key and says the reason. Other keys of the
// section keep their defaults either way.
func TestLoad_OrgDirectorLaunchCmd(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string // the loaded launch_cmd; read only when wantErr is ""
		wantErr string   // a substring of the error; "" means Load must succeed
	}{
		{"no file content", "", nil, ""},
		{"section without the key", "[org.director]\nack_minutes = 5\n", nil, ""},
		{"empty array keeps herdr", "[org.director]\nlaunch_cmd = []\n", nil, ""},
		{
			"{script} as an argument",
			`[org.director]
launch_cmd = ["sh", "-c", "nohup sh \"$1\" >/dev/null 2>&1 &", "sh", "{script}"]
`,
			[]string{"sh", "-c", `nohup sh "$1" >/dev/null 2>&1 &`, "sh", "{script}"},
			"",
		},
		{
			"{script_sh} in a shell string",
			`[org.director]
launch_cmd = ["tmux", "-L", "ralph", "new-session", "-d", "-s", "{name}", "-c", "{cwd}", "sh {script_sh}"]
`,
			[]string{"tmux", "-L", "ralph", "new-session", "-d", "-s", "{name}", "-c", "{cwd}", "sh {script_sh}"},
			"",
		},
		{
			"all four placeholders, two in one element",
			`[org.director]
launch_cmd = ["orca", "terminal", "create", "--name={name}", "--command", "cd {cwd} && sh {script_sh}", "{script}"]
`,
			[]string{"orca", "terminal", "create", "--name={name}", "--command", "cd {cwd} && sh {script_sh}", "{script}"},
			"",
		},
		{"{script} as the command itself", "[org.director]\nlaunch_cmd = [\"{script}\"]\n", []string{"{script}"}, ""},
		{
			"unmatched braces pass through",
			`[org.director]
launch_cmd = ["sh", "-c", "echo }{ x{ {{script_sh}}", "{"]
`,
			[]string{"sh", "-c", "echo }{ x{ {{script_sh}}", "{"},
			"",
		},
		{"empty first element", "[org.director]\nlaunch_cmd = [\"\", \"{script}\"]\n", nil,
			`the first element is the command to run and must not be empty or whitespace only, got ""`},
		{"whitespace-only first element", "[org.director]\nlaunch_cmd = [\" \\t\", \"{script}\"]\n", nil,
			`must not be empty or whitespace only, got " \t"`},
		{"unknown placeholder in a later element", "[org.director]\nlaunch_cmd = [\"tmux\", \"new-session\", \"-s\", \"{title}\", \"sh {script_sh}\"]\n", nil,
			`element 3 "{title}" has the unknown placeholder {title}; the placeholders are {script}, {script_sh}, {name}, {cwd}`},
		{"unknown placeholder in the first element", "[org.director]\nlaunch_cmd = [\"{terminal}\", \"{script}\"]\n", nil,
			`element 0 "{terminal}" has the unknown placeholder {terminal}`},
		{"misspelt placeholder next to a valid one", "[org.director]\nlaunch_cmd = [\"sh\", \"-c\", \"cd {cwd} && sh {scirpt}\"]\n", nil,
			`element 2 "cd {cwd} && sh {scirpt}" has the unknown placeholder {scirpt}`},
		{"placeholder in another case", "[org.director]\nlaunch_cmd = [\"sh\", \"{SCRIPT}\"]\n", nil,
			`has the unknown placeholder {SCRIPT}`},
		{"placeholder with spaces inside", "[org.director]\nlaunch_cmd = [\"sh\", \"{ script }\"]\n", nil,
			`has the unknown placeholder { script }`},
		{"empty braces", "[org.director]\nlaunch_cmd = [\"xargs\", \"-I\", \"{}\", \"sh\", \"{script}\"]\n", nil,
			`element 2 "{}" has the unknown placeholder {}`},
		{"shell brace group", "[org.director]\nlaunch_cmd = [\"sh\", \"-c\", \"{ sh \\\"$0\\\"; }\", \"{script}\"]\n", nil,
			`has the unknown placeholder { sh "$0"; }`},
		{"neither {script} nor {script_sh}", "[org.director]\nlaunch_cmd = [\"tmux\", \"new-session\", \"-d\", \"-s\", \"{name}\", \"-c\", \"{cwd}\"]\n", nil,
			`no element has {script} or {script_sh}, so the terminal would never run the launch script`},
		{"no placeholder at all", "[org.director]\nlaunch_cmd = [\"open\", \"-a\", \"Terminal\"]\n", nil,
			`no element has {script} or {script_sh}`},
		{"the first-element check comes first", "[org.director]\nlaunch_cmd = [\" \", \"{title}\"]\n", nil,
			`the first element is the command to run`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "ralph.toml")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Load: err = nil (launch_cmd = %q), want an error containing %q", cfg.Org.Director.LaunchCmd, tt.wantErr)
				}
				if !strings.HasPrefix(err.Error(), "[org.director].launch_cmd: ") {
					t.Errorf("error = %q, want it to start with %q", err.Error(), "[org.director].launch_cmd: ")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load: unexpected error: %v", err)
			}
			if !slices.Equal(cfg.Org.Director.LaunchCmd, tt.want) {
				t.Errorf("director.launch_cmd = %q, want %q", cfg.Org.Director.LaunchCmd, tt.want)
			}
			def := Default().Org.Director
			if !cfg.Org.Director.Enabled || cfg.Org.Director.ResolveMinutes != def.ResolveMinutes {
				t.Errorf("director.enabled, resolve_minutes = %v, %d, want the defaults %v, %d",
					cfg.Org.Director.Enabled, cfg.Org.Director.ResolveMinutes, def.Enabled, def.ResolveMinutes)
			}
		})
	}
}

// TestLoad_OrgDirectorLaunchCmdWrongType verifies a launch_cmd that is not
// an array of strings (a single string, or an array with a number) fails
// Load at decoding, before ValidateDirectorLaunchCmd runs.
func TestLoad_OrgDirectorLaunchCmdWrongType(t *testing.T) {
	for _, content := range []string{
		"[org.director]\nlaunch_cmd = \"tmux new-session {script_sh}\"\n",
		"[org.director]\nlaunch_cmd = [\"sh\", 1, \"{script}\"]\n",
	} {
		path := filepath.Join(t.TempDir(), "ralph.toml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		if cfg, err := Load(path); err == nil {
			t.Errorf("Load(%q): err = nil (launch_cmd = %q), want a decode error", content, cfg.Org.Director.LaunchCmd)
		}
	}
}

// TestValidateDirectorLaunchCmd_EmptyIsValid verifies a nil and an empty
// launch_cmd (start uses herdr) pass ValidateDirectorLaunchCmd, which Load
// and `ralph org director start` share.
func TestValidateDirectorLaunchCmd_EmptyIsValid(t *testing.T) {
	for _, cmd := range [][]string{nil, {}} {
		if err := ValidateDirectorLaunchCmd(cmd); err != nil {
			t.Errorf("ValidateDirectorLaunchCmd(%q) = %v, want nil", cmd, err)
		}
	}
}

// TestDirectorLaunchPlaceholders verifies the placeholder constants and that
// DirectorLaunchPlaceholders lists exactly those four, in order, as a slice
// a caller may change without changing the next call's.
func TestDirectorLaunchPlaceholders(t *testing.T) {
	want := []string{"{script}", "{script_sh}", "{name}", "{cwd}"}
	consts := []string{DirectorLaunchPlaceholderScript, DirectorLaunchPlaceholderScriptSh, DirectorLaunchPlaceholderName, DirectorLaunchPlaceholderCwd}
	if !slices.Equal(consts, want) {
		t.Errorf("placeholder constants = %q, want %q", consts, want)
	}
	got := DirectorLaunchPlaceholders()
	if !slices.Equal(got, want) {
		t.Fatalf("DirectorLaunchPlaceholders() = %q, want %q", got, want)
	}
	got[0] = "{changed}"
	if again := DirectorLaunchPlaceholders(); !slices.Equal(again, want) {
		t.Errorf("DirectorLaunchPlaceholders() after changing a returned slice = %q, want %q", again, want)
	}
}

// TestExpandDirectorLaunchArg verifies the replacement start applies to each
// launch_cmd element: every placeholder is replaced by its value in one
// pass (a value holding a placeholder is not replaced again), braces that
// are not a placeholder are kept as written, and a placeholder outside the
// values (unknown, or known but given no value) is an error that returns the
// element unchanged.
func TestExpandDirectorLaunchArg(t *testing.T) {
	values := map[string]string{
		DirectorLaunchPlaceholderScript:   "/repo dir/.harness/state/org/launch/director_t1.sh",
		DirectorLaunchPlaceholderScriptSh: `'/repo dir/.harness/state/org/launch/director_t1.sh'`,
		DirectorLaunchPlaceholderName:     "director_t1",
		DirectorLaunchPlaceholderCwd:      "/repo {name}",
	}
	tests := []struct {
		arg     string
		want    string
		wantErr string
	}{
		{"{script}", "/repo dir/.harness/state/org/launch/director_t1.sh", ""},
		{"sh {script_sh}", `sh '/repo dir/.harness/state/org/launch/director_t1.sh'`, ""},
		{"--name={name}", "--name=director_t1", ""},
		{"{name}-{name}", "director_t1-director_t1", ""},
		{"{cwd}", "/repo {name}", ""},
		{"cd {cwd} && sh {script_sh}", `cd /repo {name} && sh '/repo dir/.harness/state/org/launch/director_t1.sh'`, ""},
		{"no placeholder", "no placeholder", ""},
		{"", "", ""},
		{"echo }{ x{ {{name}}", "echo }{ x{ {director_t1}", ""},
		{"{title} {name}", "", `"{title} {name}" has the placeholder {title}, which has no value`},
		{"{}", "", `has the placeholder {}, which has no value`},
	}
	for _, tt := range tests {
		got, err := ExpandDirectorLaunchArg(tt.arg, values)
		if tt.wantErr != "" {
			if err == nil || !strings.HasPrefix(err.Error(), "[org.director].launch_cmd: ") || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ExpandDirectorLaunchArg(%q) err = %v, want an error naming [org.director].launch_cmd and containing %q", tt.arg, err, tt.wantErr)
			}
			if got != tt.arg {
				t.Errorf("ExpandDirectorLaunchArg(%q) on error = %q, want the element unchanged", tt.arg, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ExpandDirectorLaunchArg(%q): unexpected error: %v", tt.arg, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ExpandDirectorLaunchArg(%q) = %q, want %q", tt.arg, got, tt.want)
		}
	}

	// A known placeholder with no entry in values is an error too.
	partial := map[string]string{DirectorLaunchPlaceholderScript: "/s.sh"}
	if got, err := ExpandDirectorLaunchArg("sh {script} {cwd}", partial); err == nil || !strings.Contains(err.Error(), "has the placeholder {cwd}, which has no value") || got != "sh {script} {cwd}" {
		t.Errorf("ExpandDirectorLaunchArg with no {cwd} value = %q, %v, want the element unchanged and an error naming {cwd}", got, err)
	}
}

// TestTemplateRalphToml_LaunchCmdExampleIsValid verifies the commented-out
// launch_cmd example in templates/base/ralph.toml, with its "# " removed,
// loads under [org.director], so the example a user uncomments passes the
// validation.
func TestTemplateRalphToml_LaunchCmdExampleIsValid(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "..", "..", "templates", "base", "ralph.toml"))
	if err != nil {
		t.Fatalf("read templates/base/ralph.toml: %v", err)
	}
	var examples []string
	for line := range strings.SplitSeq(string(data), "\n") {
		if example, found := strings.CutPrefix(line, "# launch_cmd = "); found {
			examples = append(examples, "launch_cmd = "+example)
		}
	}
	if len(examples) != 1 {
		t.Fatalf("templates/base/ralph.toml has %d lines starting with %q, want 1", len(examples), "# launch_cmd = ")
	}
	path := filepath.Join(t.TempDir(), "ralph.toml")
	if err := os.WriteFile(path, []byte("[org.director]\n"+examples[0]+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(the uncommented template example %q): %v", examples[0], err)
	}
	if len(cfg.Org.Director.LaunchCmd) == 0 {
		t.Errorf("the uncommented template example %q loaded an empty launch_cmd", examples[0])
	}
}

// TestLoad_OrgDirectorWake covers [org.director].wake through Load() (plan
// 2026-10-18-org-director-wake, AC1). An absent [org.director], an absent
// key, or no file content loads as ["pull"]; pull, push, and both in either
// order load as written. An empty array, a value other than pull and push
// (another case and the empty string included), and a value listed twice
// fail with an error that starts with the key and says the reason. Other
// keys of the section keep their defaults either way.
func TestLoad_OrgDirectorWake(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string // the loaded wake; read only when wantErr is ""
		wantErr string   // a substring of the error; "" means Load must succeed
	}{
		{"no file content", "", []string{"pull"}, ""},
		{"no [org.director]", "[org.watchdog]\nstall_minutes = 20\n", []string{"pull"}, ""},
		{"section without the key", "[org.director]\nack_minutes = 5\n", []string{"pull"}, ""},
		{"pull", "[org.director]\nwake = [\"pull\"]\n", []string{"pull"}, ""},
		{"push", "[org.director]\nwake = [\"push\"]\n", []string{"push"}, ""},
		{"pull and push", "[org.director]\nwake = [\"pull\", \"push\"]\n", []string{"pull", "push"}, ""},
		{"push and pull", "[org.director]\nwake = [\"push\", \"pull\"]\n", []string{"push", "pull"}, ""},
		{"empty array", "[org.director]\nwake = []\n", nil,
			`the array must not be empty; the values are "pull" and "push", one or both`},
		{"unknown value", "[org.director]\nwake = [\"poll\"]\n", nil,
			`element 0 "poll" is not a wake value; the values are "pull" and "push"`},
		{"unknown value after a valid one", "[org.director]\nwake = [\"pull\", \"tmux\"]\n", nil,
			`element 1 "tmux" is not a wake value`},
		{"value in another case", "[org.director]\nwake = [\"Push\"]\n", nil,
			`element 0 "Push" is not a wake value`},
		{"empty string", "[org.director]\nwake = [\"\"]\n", nil,
			`element 0 "" is not a wake value`},
		{"value with spaces", "[org.director]\nwake = [\" pull\"]\n", nil,
			`element 0 " pull" is not a wake value`},
		{"comma-joined value", "[org.director]\nwake = [\"pull,push\"]\n", nil,
			`element 0 "pull,push" is not a wake value`},
		{"pull twice", "[org.director]\nwake = [\"pull\", \"pull\"]\n", nil,
			`element 1 "pull" repeats element 0; list each value once`},
		{"push twice around pull", "[org.director]\nwake = [\"push\", \"pull\", \"push\"]\n", nil,
			`element 2 "push" repeats element 0; list each value once`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "ralph.toml")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Load: err = nil (wake = %q), want an error containing %q", cfg.Org.Director.Wake, tt.wantErr)
				}
				if !strings.HasPrefix(err.Error(), "[org.director].wake: ") {
					t.Errorf("error = %q, want it to start with %q", err.Error(), "[org.director].wake: ")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load: unexpected error: %v", err)
			}
			if !slices.Equal(cfg.Org.Director.Wake, tt.want) {
				t.Errorf("director.wake = %q, want %q", cfg.Org.Director.Wake, tt.want)
			}
			if len(cfg.Org.Director.WakeCmd) != 0 {
				t.Errorf("director.wake_cmd = %q, want empty", cfg.Org.Director.WakeCmd)
			}
			def := Default().Org.Director
			if !cfg.Org.Director.Enabled || cfg.Org.Director.ResolveMinutes != def.ResolveMinutes || len(cfg.Org.Director.LaunchCmd) != 0 {
				t.Errorf("director.enabled, resolve_minutes, launch_cmd = %v, %d, %q, want the defaults %v, %d, empty",
					cfg.Org.Director.Enabled, cfg.Org.Director.ResolveMinutes, cfg.Org.Director.LaunchCmd, def.Enabled, def.ResolveMinutes)
			}
		})
	}
}

// TestLoad_OrgDirectorWakePushWithoutWakeCmd verifies that Load accepts a
// wake with push and no wake_cmd (absent or empty): whether a director can be
// pushed depends on its registration, so `ralph org director attach` and
// `start` check it, not Load (plan 2026-10-18-org-director-wake, D4).
func TestLoad_OrgDirectorWakePushWithoutWakeCmd(t *testing.T) {
	for _, content := range []string{
		"[org.director]\nwake = [\"push\"]\n",
		"[org.director]\nwake = [\"push\"]\nwake_cmd = []\n",
		"[org.director]\nwake = [\"pull\", \"push\"]\nwake_cmd = []\n",
	} {
		path := filepath.Join(t.TempDir(), "ralph.toml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(path)
		if err != nil {
			t.Errorf("Load(%q): unexpected error: %v", content, err)
			continue
		}
		if !slices.Contains(cfg.Org.Director.Wake, "push") || len(cfg.Org.Director.WakeCmd) != 0 {
			t.Errorf("Load(%q): wake, wake_cmd = %q, %q, want push and an empty wake_cmd", content, cfg.Org.Director.Wake, cfg.Org.Director.WakeCmd)
		}
	}
}

// TestLoad_OrgDirectorWakeCmd covers [org.director].wake_cmd through Load()
// (plan 2026-10-18-org-director-wake, AC1). An absent key or an empty array
// loads as empty, and a value whose placeholders are {target} and {text}
// outside the first element loads as written, braces that form no pair
// included. A non-empty value whose first element is empty or whitespace
// only, whose first element has any placeholder (a known one included), or
// that has a {...} other than the two in any element fails with an error that
// starts with the key and says the reason. Other keys of the section keep
// their defaults either way.
func TestLoad_OrgDirectorWakeCmd(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string // the loaded wake_cmd; read only when wantErr is ""
		wantErr string   // a substring of the error; "" means Load must succeed
	}{
		{"no file content", "", nil, ""},
		{"section without the key", "[org.director]\nack_minutes = 5\n", nil, ""},
		{"empty array", "[org.director]\nwake_cmd = []\n", nil, ""},
		{
			"tmux send-keys",
			`[org.director]
wake_cmd = ["tmux", "-L", "ralph", "send-keys", "-t", "{target}", "-l", "{text}", ";", "send-keys", "-t", "{target}", "Enter"]
`,
			[]string{"tmux", "-L", "ralph", "send-keys", "-t", "{target}", "-l", "{text}", ";", "send-keys", "-t", "{target}", "Enter"},
			"",
		},
		{
			"orca terminal send",
			`[org.director]
wake_cmd = ["orca", "terminal", "send", "--terminal", "{target}", "--text", "{text}", "--enter"]
`,
			[]string{"orca", "terminal", "send", "--terminal", "{target}", "--text", "{text}", "--enter"},
			"",
		},
		{"both placeholders in one element", "[org.director]\nwake_cmd = [\"./wake.sh\", \"{target}{text}\"]\n",
			[]string{"./wake.sh", "{target}{text}"}, ""},
		{"a placeholder inside a longer element", "[org.director]\nwake_cmd = [\"wake\", \"--to={target}\", \"--line={text}\"]\n",
			[]string{"wake", "--to={target}", "--line={text}"}, ""},
		{"no placeholder at all", "[org.director]\nwake_cmd = [\"/usr/local/bin/wake-director\"]\n",
			[]string{"/usr/local/bin/wake-director"}, ""},
		{
			"unmatched braces pass through",
			`[org.director]
wake_cmd = ["sh", "-c", "echo }{ x{ {{text}}", "{"]
`,
			[]string{"sh", "-c", "echo }{ x{ {{text}}", "{"},
			"",
		},
		{"an unpaired brace in the first element", "[org.director]\nwake_cmd = [\"wake{\", \"{text}\"]\n",
			[]string{"wake{", "{text}"}, ""},
		{"empty first element", "[org.director]\nwake_cmd = [\"\", \"{text}\"]\n", nil,
			`the first element is the command to run and must not be empty or whitespace only, got ""`},
		{"whitespace-only first element", "[org.director]\nwake_cmd = [\" \\t\", \"{text}\"]\n", nil,
			`must not be empty or whitespace only, got " \t"`},
		{"{target} as the first element", "[org.director]\nwake_cmd = [\"{target}\", \"{text}\"]\n", nil,
			`the first element is the command to run and must not contain a placeholder (a director's registration must never choose the program), got "{target}"`},
		{"{text} as the first element", "[org.director]\nwake_cmd = [\"{text}\"]\n", nil,
			`must not contain a placeholder (a director's registration must never choose the program), got "{text}"`},
		{"a placeholder inside the first element", "[org.director]\nwake_cmd = [\"./bin/{target}.sh\", \"{text}\"]\n", nil,
			`must not contain a placeholder (a director's registration must never choose the program), got "./bin/{target}.sh"`},
		{"an unknown placeholder in the first element", "[org.director]\nwake_cmd = [\"{prog}\", \"{text}\"]\n", nil,
			`must not contain a placeholder (a director's registration must never choose the program), got "{prog}"`},
		{"placeholder in another case", "[org.director]\nwake_cmd = [\"tmux\", \"send-keys\", \"-t\", \"{TARGET}\", \"-l\", \"{text}\"]\n", nil,
			`element 3 "{TARGET}" has the unknown placeholder {TARGET}; the placeholders are {target}, {text}`},
		{"a launch_cmd placeholder", "[org.director]\nwake_cmd = [\"tmux\", \"send-keys\", \"-t\", \"{name}\", \"{text}\"]\n", nil,
			`element 3 "{name}" has the unknown placeholder {name}`},
		{"misspelt placeholder next to a valid one", "[org.director]\nwake_cmd = [\"wake\", \"{target} {txet}\"]\n", nil,
			`element 1 "{target} {txet}" has the unknown placeholder {txet}`},
		{"placeholder with spaces inside", "[org.director]\nwake_cmd = [\"wake\", \"{ text }\"]\n", nil,
			`has the unknown placeholder { text }`},
		{"empty braces", "[org.director]\nwake_cmd = [\"xargs\", \"-I\", \"{}\", \"wake\", \"{text}\"]\n", nil,
			`element 2 "{}" has the unknown placeholder {}`},
		{"shell brace group", "[org.director]\nwake_cmd = [\"sh\", \"-c\", \"{ echo \\\"$0\\\"; }\", \"{text}\"]\n", nil,
			`has the unknown placeholder { echo "$0"; }`},
		{"the first-element empty check comes first", "[org.director]\nwake_cmd = [\" \", \"{title}\"]\n", nil,
			`must not be empty or whitespace only`},
		{"the first-element placeholder check comes before the unknown one", "[org.director]\nwake_cmd = [\"{target}\", \"{title}\"]\n", nil,
			`must not contain a placeholder`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "ralph.toml")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("Load: err = nil (wake_cmd = %q), want an error containing %q", cfg.Org.Director.WakeCmd, tt.wantErr)
				}
				if !strings.HasPrefix(err.Error(), "[org.director].wake_cmd: ") {
					t.Errorf("error = %q, want it to start with %q", err.Error(), "[org.director].wake_cmd: ")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load: unexpected error: %v", err)
			}
			if !slices.Equal(cfg.Org.Director.WakeCmd, tt.want) {
				t.Errorf("director.wake_cmd = %q, want %q", cfg.Org.Director.WakeCmd, tt.want)
			}
			def := Default().Org.Director
			if !slices.Equal(cfg.Org.Director.Wake, def.Wake) || !cfg.Org.Director.Enabled || len(cfg.Org.Director.LaunchCmd) != 0 {
				t.Errorf("director.wake, enabled, launch_cmd = %q, %v, %q, want the defaults %q, %v, empty",
					cfg.Org.Director.Wake, cfg.Org.Director.Enabled, cfg.Org.Director.LaunchCmd, def.Wake, def.Enabled)
			}
		})
	}
}

// TestLoad_OrgDirectorWakeWrongType verifies a wake or wake_cmd that is not
// an array of strings (a single string, or an array with a number) fails
// Load at decoding, before ValidateDirectorWake and ValidateDirectorWakeCmd
// run.
func TestLoad_OrgDirectorWakeWrongType(t *testing.T) {
	for _, content := range []string{
		"[org.director]\nwake = \"pull\"\n",
		"[org.director]\nwake = [\"pull\", 1]\n",
		"[org.director]\nwake_cmd = \"tmux send-keys -t {target} {text}\"\n",
		"[org.director]\nwake_cmd = [\"wake\", 1, \"{text}\"]\n",
	} {
		path := filepath.Join(t.TempDir(), "ralph.toml")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		if cfg, err := Load(path); err == nil {
			t.Errorf("Load(%q): err = nil (wake = %q, wake_cmd = %q), want a decode error", content, cfg.Org.Director.Wake, cfg.Org.Director.WakeCmd)
		}
	}
}

// TestValidateDirectorWake checks ValidateDirectorWake directly, as attach
// and start may call it on a value that did not come through Load: the four
// valid values pass, and nil fails like an empty array.
func TestValidateDirectorWake(t *testing.T) {
	if DirectorWakePull != "pull" || DirectorWakePush != "push" {
		t.Errorf("DirectorWakePull, DirectorWakePush = %q, %q, want pull, push", DirectorWakePull, DirectorWakePush)
	}
	for _, wake := range [][]string{{"pull"}, {"push"}, {"pull", "push"}, {"push", "pull"}} {
		if err := ValidateDirectorWake(wake); err != nil {
			t.Errorf("ValidateDirectorWake(%q) = %v, want nil", wake, err)
		}
	}
	for _, wake := range [][]string{nil, {}} {
		err := ValidateDirectorWake(wake)
		if err == nil || err.Error() != `[org.director].wake: the array must not be empty; the values are "pull" and "push", one or both` {
			t.Errorf("ValidateDirectorWake(%#v) = %v, want the empty-array error", wake, err)
		}
	}
}

// TestValidateDirectorWakeNamed checks the rule of ValidateDirectorWake
// under another name, as `ralph org director attach --wake` calls it (plan
// 2026-10-18-org-director-wake, AC2): each error starts with that name, not
// the key, and keeps the rest of the key's text; the four valid values pass.
func TestValidateDirectorWakeNamed(t *testing.T) {
	for _, wake := range [][]string{{"pull"}, {"push"}, {"pull", "push"}, {"push", "pull"}} {
		if err := ValidateDirectorWakeNamed("--wake", wake); err != nil {
			t.Errorf("ValidateDirectorWakeNamed(--wake, %q) = %v, want nil", wake, err)
		}
	}
	for _, tc := range []struct {
		wake []string
		want string
	}{
		{nil, `--wake: the array must not be empty; the values are "pull" and "push", one or both`},
		{[]string{""}, `--wake: element 0 "" is not a wake value; the values are "pull" and "push"`},
		{[]string{"pull", " push"}, `--wake: element 1 " push" is not a wake value; the values are "pull" and "push"`},
		{[]string{"pull", "pull"}, `--wake: element 1 "pull" repeats element 0; list each value once`},
	} {
		err := ValidateDirectorWakeNamed("--wake", tc.wake)
		if err == nil || err.Error() != tc.want {
			t.Errorf("ValidateDirectorWakeNamed(--wake, %q) = %v, want %q", tc.wake, err, tc.want)
		}
		keyed := ValidateDirectorWake(tc.wake)
		if want := "[org.director].wake" + strings.TrimPrefix(tc.want, "--wake"); keyed == nil || keyed.Error() != want {
			t.Errorf("ValidateDirectorWake(%q) = %v, want %q", tc.wake, keyed, want)
		}
	}
}

// TestValidateDirectorWakeCmd_EmptyIsValid verifies a nil and an empty
// wake_cmd pass ValidateDirectorWakeCmd, which Load and later the watch
// share.
func TestValidateDirectorWakeCmd_EmptyIsValid(t *testing.T) {
	for _, cmd := range [][]string{nil, {}} {
		if err := ValidateDirectorWakeCmd(cmd); err != nil {
			t.Errorf("ValidateDirectorWakeCmd(%q) = %v, want nil", cmd, err)
		}
	}
}

// TestDirectorWakePlaceholders verifies the wake_cmd placeholder constants
// and that DirectorWakePlaceholders lists exactly those two, in order, as a
// slice a caller may change without changing the next call's.
func TestDirectorWakePlaceholders(t *testing.T) {
	want := []string{"{target}", "{text}"}
	consts := []string{DirectorWakePlaceholderTarget, DirectorWakePlaceholderText}
	if !slices.Equal(consts, want) {
		t.Errorf("placeholder constants = %q, want %q", consts, want)
	}
	got := DirectorWakePlaceholders()
	if !slices.Equal(got, want) {
		t.Fatalf("DirectorWakePlaceholders() = %q, want %q", got, want)
	}
	got[0] = "{changed}"
	if again := DirectorWakePlaceholders(); !slices.Equal(again, want) {
		t.Errorf("DirectorWakePlaceholders() after changing a returned slice = %q, want %q", again, want)
	}
}

// TestExpandDirectorWakeArg verifies the replacement the watch applies to
// each wake_cmd element: every placeholder is replaced by its value in one
// pass (a value holding a placeholder is not replaced again), braces that
// are not a placeholder are kept as written, and a placeholder outside the
// values (unknown, or known but given no value) is an error that names
// [org.director].wake_cmd and returns the element unchanged.
func TestExpandDirectorWakeArg(t *testing.T) {
	values := map[string]string{
		DirectorWakePlaceholderTarget: "director_t1 {text}",
		DirectorWakePlaceholderText:   "ralph inbox: new items e3 e4 {target}",
	}
	tests := []struct {
		arg     string
		want    string
		wantErr string
	}{
		{"{target}", "director_t1 {text}", ""},
		{"{text}", "ralph inbox: new items e3 e4 {target}", ""},
		{"{target}{text}", "director_t1 {text}ralph inbox: new items e3 e4 {target}", ""},
		{"--to={target}", "--to=director_t1 {text}", ""},
		{"{target}-{target}", "director_t1 {text}-director_t1 {text}", ""},
		{"no placeholder", "no placeholder", ""},
		{"", "", ""},
		{"echo }{ x{ {{text}}", "echo }{ x{ {ralph inbox: new items e3 e4 {target}}", ""},
		{"{TARGET} {text}", "", `"{TARGET} {text}" has the placeholder {TARGET}, which has no value`},
		{"{name}", "", `has the placeholder {name}, which has no value`},
		{"{}", "", `has the placeholder {}, which has no value`},
	}
	for _, tt := range tests {
		got, err := ExpandDirectorWakeArg(tt.arg, values)
		if tt.wantErr != "" {
			if err == nil || !strings.HasPrefix(err.Error(), "[org.director].wake_cmd: ") || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ExpandDirectorWakeArg(%q) err = %v, want an error naming [org.director].wake_cmd and containing %q", tt.arg, err, tt.wantErr)
			}
			if got != tt.arg {
				t.Errorf("ExpandDirectorWakeArg(%q) on error = %q, want the element unchanged", tt.arg, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ExpandDirectorWakeArg(%q): unexpected error: %v", tt.arg, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ExpandDirectorWakeArg(%q) = %q, want %q", tt.arg, got, tt.want)
		}
	}

	// A known placeholder with no entry in values is an error too.
	partial := map[string]string{DirectorWakePlaceholderText: "wake line"}
	if got, err := ExpandDirectorWakeArg("{target} {text}", partial); err == nil ||
		err.Error() != `[org.director].wake_cmd: "{target} {text}" has the placeholder {target}, which has no value` || got != "{target} {text}" {
		t.Errorf("ExpandDirectorWakeArg with no {target} value = %q, %v, want the element unchanged and an error naming {target}", got, err)
	}
}

// TestTemplateRalphToml_WakeCmdExampleIsValid verifies the commented-out
// wake_cmd example in templates/base/ralph.toml, with its "# " removed,
// loads under [org.director] with both placeholders outside the first
// element, so the example a user uncomments passes the validation.
func TestTemplateRalphToml_WakeCmdExampleIsValid(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "..", "..", "templates", "base", "ralph.toml"))
	if err != nil {
		t.Fatalf("read templates/base/ralph.toml: %v", err)
	}
	var examples []string
	for line := range strings.SplitSeq(string(data), "\n") {
		if example, found := strings.CutPrefix(line, "# wake_cmd = "); found {
			examples = append(examples, "wake_cmd = "+example)
		}
	}
	if len(examples) != 1 {
		t.Fatalf("templates/base/ralph.toml has %d lines starting with %q, want 1", len(examples), "# wake_cmd = ")
	}
	path := filepath.Join(t.TempDir(), "ralph.toml")
	if err := os.WriteFile(path, []byte("[org.director]\nwake = [\"push\"]\n"+examples[0]+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(the uncommented template example %q): %v", examples[0], err)
	}
	cmd := cfg.Org.Director.WakeCmd
	if len(cmd) < 2 || !slices.Contains(cmd[1:], DirectorWakePlaceholderTarget) || !slices.Contains(cmd[1:], DirectorWakePlaceholderText) {
		t.Errorf("the uncommented template example %q loaded wake_cmd %q, want {target} and {text} as arguments", examples[0], cmd)
	}
}

// TestDefault_OrgApproval verifies the [org.approval] default: per-plan.
func TestDefault_OrgApproval(t *testing.T) {
	if got := Default().Org.Approval.Mode; got != "per-plan" {
		t.Errorf("approval.mode = %q, want %q", got, "per-plan")
	}
	if ApprovalModePerPlan != "per-plan" || ApprovalModeSplit != "split" {
		t.Errorf("ApprovalModePerPlan, ApprovalModeSplit = %q, %q, want per-plan, split", ApprovalModePerPlan, ApprovalModeSplit)
	}
}

// TestLoad_OrgApproval covers [org.approval].mode through Load(): an absent
// section or an absent key keeps the default per-plan, per-plan and split
// load as written, and any other value, the empty string and another case
// included, is an error naming the key and the two modes.
func TestLoad_OrgApproval(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantMode string // "" means Load must fail
		wantErr  string
	}{
		{"no file content", "", ApprovalModePerPlan, ""},
		{"absent section with other [org] tables", "[org]\nmax_seats = 3\n\n[org.director]\nack_minutes = 5\n", ApprovalModePerPlan, ""},
		{"section without the key", "[org.approval]\n", ApprovalModePerPlan, ""},
		{"per-plan", "[org.approval]\nmode = \"per-plan\"\n", ApprovalModePerPlan, ""},
		{"split", "[org.approval]\nmode = \"split\"\n", ApprovalModeSplit, ""},
		{"empty string", "[org.approval]\nmode = \"\"\n", "", `[org.approval].mode "" must be "per-plan" or "split"`},
		{"unknown value", "[org.approval]\nmode = \"auto\"\n", "", `[org.approval].mode "auto" must be "per-plan" or "split"`},
		{"another case", "[org.approval]\nmode = \"Split\"\n", "", `[org.approval].mode "Split" must be`},
		{"surrounding space", "[org.approval]\nmode = \" split\"\n", "", `[org.approval].mode " split" must be`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "ralph.toml")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if tt.wantMode == "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Load: err = %v, want an error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load: unexpected error: %v", err)
			}
			if cfg.Org.Approval.Mode != tt.wantMode {
				t.Errorf("approval.mode = %q, want %q", cfg.Org.Approval.Mode, tt.wantMode)
			}
		})
	}
}

// TestLoad_DriverPoolOnlyOverride_FiltersInheritedDefaultModelPool verifies
// that a ralph.toml overriding only [org].driver_pool (leaving model_pool
// unset) inherits the default model_pool filtered down to the declared
// driver_pool, rather than the raw (unfiltered) default pool -- a claude-only
// driver_pool must not inherit the default pool's codex entries and fail the
// driver-membership check (cross-review ACTION_REQUIRED #1).
func TestLoad_DriverPoolOnlyOverride_FiltersInheritedDefaultModelPool(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[org]
driver_pool = ["claude"]
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	wantModels := []string{"fable", "opus", "sonnet", "haiku"}
	if len(cfg.Org.ModelPool) != len(wantModels) {
		t.Fatalf("model_pool = %+v, want %d claude entries %v", cfg.Org.ModelPool, len(wantModels), wantModels)
	}
	for i, entry := range cfg.Org.ModelPool {
		if entry.Driver != "claude" {
			t.Errorf("model_pool[%d].driver = %q, want claude", i, entry.Driver)
		}
		if entry.Model != wantModels[i] {
			t.Errorf("model_pool[%d].model = %q, want %q", i, entry.Model, wantModels[i])
		}
	}
}

// TestLoad_DriverPoolOnlyOverride_Codex_KeepsOnlyCodexDefaultEntriesInOrder
// verifies that `driver_pool = ["codex"]` alone inherits exactly Default()'s
// codex model_pool entries -- every one of them, in Default()'s declared
// order, and no claude entry -- rather than the whole default pool or an
// empty one.
func TestLoad_DriverPoolOnlyOverride_Codex_KeepsOnlyCodexDefaultEntriesInOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[org]
driver_pool = ["codex"]
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: unexpected error: %v", err)
	}
	wantModels := []string{"gpt-6-astra", "gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"}
	if len(cfg.Org.ModelPool) != len(wantModels) {
		t.Fatalf("model_pool = %+v, want %d codex entries %v", cfg.Org.ModelPool, len(wantModels), wantModels)
	}
	for i, entry := range cfg.Org.ModelPool {
		if entry.Driver != "codex" {
			t.Errorf("model_pool[%d].driver = %q, want codex", i, entry.Driver)
		}
		if entry.Model != wantModels[i] {
			t.Errorf("model_pool[%d].model = %q, want %q", i, entry.Model, wantModels[i])
		}
	}
}

// TestLoad_DriverPoolOnlyOverride_NoDefaultModels_Errors verifies that when
// the declared driver_pool names only drivers with no entries in Default()'s
// model_pool (e.g. a hypothetical "gemini" driver), filtering the inherited
// default pool down to that driver_pool leaves it empty, and Load() still
// returns an error that names [org].driver_pool (the key the document actually wrote) rather than
// silently succeeding with zero usable models.
func TestLoad_DriverPoolOnlyOverride_NoDefaultModels_Errors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[org]
driver_pool = ["gemini"]
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	if !contains(err.Error(), "driver_pool [gemini] has no default [org].model_pool entries") {
		t.Errorf("error %q does not mention %q", err.Error(), "driver_pool [gemini] has no default [org].model_pool entries")
	}
}

// TestLoad_ExplicitModelPoolStillStrict verifies that a document which sets
// model_pool explicitly is not subject to the driver_pool-only inheritance
// filter: an explicit codex entry alongside a claude-only driver_pool must
// still fail the driver-membership check exactly as before this change.
func TestLoad_ExplicitModelPoolStillStrict(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[org]
driver_pool = ["claude"]
model_pool = [
  { driver = "claude", model = "opus" },
  { driver = "codex", model = "gpt-5.5" },
]
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	if !contains(err.Error(), "not present in [org].driver_pool") {
		t.Errorf("error %q does not mention %q", err.Error(), "not present in [org].driver_pool")
	}
}

// TestLoad_DriverPoolOnlyOverride_RolesReferencingFilteredModelErrors
// verifies that [org.roles] validation still runs against the filtered
// (inherited) model_pool: a role referencing a model that driver_pool
// filtering dropped must still error via the existing roles check. The model
// is a codex slug that IS in Default()'s pool, so only validation against the
// filtered pool rejects it; a model absent from the default pool altogether
// would also be rejected by validation against the unfiltered pool and could
// not tell the two apart.
func TestLoad_DriverPoolOnlyOverride_RolesReferencingFilteredModelErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[org]
driver_pool = ["claude"]

[org.roles]
reviewer = ["gpt-6-astra"]
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	// The distinguishing phrase from config.go's roles validation: the role,
	// the filtered-out model, and the reason. A "reviewer"-only check would
	// also accept an unrelated error that merely names the role.
	want := `[org.roles].reviewer references model "gpt-6-astra" not present in [org].model_pool`
	if !contains(err.Error(), want) {
		t.Errorf("error %q does not contain %q", err.Error(), want)
	}
}

// TestLoad_OmittedModelPoolWithRolesNamingDroppedDefault_ErrorsUntilModelPoolIsExplicit
// verifies the fail-closed consequence of dropping gpt-5.5 from the default
// model_pool (#156): a ralph.toml that omits model_pool (and driver_pool) but
// names gpt-5.5 under [org.roles] fails validation against the 8-entry default
// pool, with the role/model wording the org skill's recovery paragraph quotes;
// the same roles with an explicit model_pool that lists gpt-5.5 load cleanly.
func TestLoad_OmittedModelPoolWithRolesNamingDroppedDefault_ErrorsUntilModelPoolIsExplicit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.toml")
	content := `[org.roles]
reviewer = ["gpt-5.5"]
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error for a role naming gpt-5.5 with the default model_pool, got nil")
	}
	want := `[org.roles].reviewer references model "gpt-5.5" not present in [org].model_pool`
	if !contains(err.Error(), want) {
		t.Errorf("error %q does not contain %q", err.Error(), want)
	}

	fixed := filepath.Join(dir, "fixed.toml")
	fixedContent := `[org]
model_pool = [
  { driver = "claude", model = "opus" },
  { driver = "codex", model = "gpt-5.5" },
]

[org.roles]
reviewer = ["gpt-5.5"]
`
	if err := os.WriteFile(fixed, []byte(fixedContent), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(fixed); err != nil {
		t.Errorf("Load of the migrated config (explicit model_pool including gpt-5.5): unexpected error: %v", err)
	}
}
