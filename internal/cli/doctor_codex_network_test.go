package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// The four texts checkCodexSeatNetwork prints, written out here in full so a
// change to the wording shows up as a test change (plan
// 2026-11-21-codex-seat-network, AC4).
const (
	wantCodexSeatNetworkNoDriverPool = "not needed: codex is not in [org].driver_pool, so no codex seat can be spawned"
	wantCodexSeatNetworkUnverified   = "not needed: [org.permissions].codex_verified is false, so no codex seat runs under --sandbox workspace-write"
	wantCodexSeatNetworkAllGuarded   = "not needed: no role that can use a codex model resolves to edits or autonomous, " +
		"so no codex seat runs under --sandbox workspace-write"
	wantCodexSeatNetworkReseat = "a change to [org.permissions].codex_network_access reaches only the codex seats started after it: " +
		"a running seat keeps its arguments until it is started again, which ralph org reseat does for the seats of a feature org " +
		"(docs/recipes/codex-seat-permissions.md)"
	wantCodexSeatNetworkOpen = "[org.permissions].codex_network_access = true: ralph passes -c sandbox_workspace_write.network_access=true " +
		"to codex seats in edits and autonomous (--sandbox workspace-write), so a check that needs the network, " +
		"such as tflint's plugin init or a dependency download, can reach it inside them; " + wantCodexSeatNetworkReseat
	wantCodexSeatNetworkClosed = "[org.permissions].codex_network_access = false: ralph passes -c sandbox_workspace_write.network_access=false " +
		"to codex seats in edits and autonomous (--sandbox workspace-write), so a check that needs the network, " +
		"such as tflint's plugin init or a dependency download, fails inside them; " +
		"set the key to true, or run such checks outside codex seats; " + wantCodexSeatNetworkReseat
)

// allClaudeRoleModels is an [org.role_models] with every role on claude: the
// check must not read it, since --driver codex starts any role on codex once
// codex is in [org].driver_pool (plan 2026-11-05-org-role-models, spec
// FR-7).
func allClaudeRoleModels() map[string]config.OrgRoleModel {
	return map[string]config.OrgRoleModel{
		"leader":      {Driver: "claude", Model: "opus"},
		"implementer": {Driver: "claude", Model: "opus"},
		"reviewer":    {Driver: "claude", Model: "opus"},
		"director":    {Driver: "claude", Model: ""},
	}
}

// TestDoctorCheckCodexSeatNetwork_Matrix runs every combination of
// [org].driver_pool (with and without codex), codex_verified, the modes of
// [org.permissions] (every role guarded, an edits role under a guarded
// default, the autonomous default), and codex_network_access. Every case
// puts every [org.role_models] role on claude, so the with-codex cases also
// pin that a project whose roles all run on claude still gets the info or
// the warn.
func TestDoctorCheckCodexSeatNetwork_Matrix(t *testing.T) {
	pools := []struct {
		name  string
		pool  []string
		codex bool
	}{
		{"pool claude+codex", []string{"claude", "codex"}, true},
		{"pool claude only", []string{"claude"}, false},
	}
	modes := []struct {
		name           string
		defaultMode    string
		roles          map[string]string
		workspaceWrite bool
	}{
		{"every role guarded", "guarded", map[string]string{"implementer": "guarded", "reviewer": "guarded"}, false},
		{"an edits role under a guarded default", "guarded", map[string]string{"implementer": "edits"}, true},
		{"the autonomous default", "autonomous", nil, true},
	}
	for _, pool := range pools {
		for _, verified := range []bool{true, false} {
			for _, mode := range modes {
				for _, network := range []bool{true, false} {
					name := fmt.Sprintf("%s/codex_verified=%v/%s/codex_network_access=%v", pool.name, verified, mode.name, network)
					t.Run(name, func(t *testing.T) {
						orgCfg := config.OrgConfig{
							DriverPool: pool.pool,
							RoleModels: allClaudeRoleModels(),
							Permissions: config.OrgPermissionsConfig{
								Default:            mode.defaultMode,
								Roles:              mode.roles,
								CodexVerified:      verified,
								CodexNetworkAccess: network,
							},
						}
						wantStatus, wantDetail := "pass", ""
						switch {
						case !pool.codex:
							wantDetail = wantCodexSeatNetworkNoDriverPool
						case !verified:
							wantDetail = wantCodexSeatNetworkUnverified
						case !mode.workspaceWrite:
							wantDetail = wantCodexSeatNetworkAllGuarded
						case network:
							wantStatus, wantDetail = "info", wantCodexSeatNetworkOpen
						default:
							wantStatus, wantDetail = "warn", wantCodexSeatNetworkClosed
						}
						r := checkCodexSeatNetwork(orgCfg)
						if r.Name != "Codex sandbox (seat network)" {
							t.Errorf("Name = %q, want %q", r.Name, "Codex sandbox (seat network)")
						}
						if r.Status != wantStatus || r.Detail != wantDetail {
							t.Errorf("checkCodexSeatNetwork = (%s, %q)\nwant (%s, %q)", r.Status, r.Detail, wantStatus, wantDetail)
						}
					})
				}
			}
		}
	}
}

// TestDoctorCheckCodexSeatNetwork_DefaultConfigAllClaudeRolesReports pins
// the project ralph scaffolds: config.Default()'s [org] puts every
// [org.role_models] role on claude, yet codex is in driver_pool, so the
// check reports the network (info), and with the key turned off it warns.
func TestDoctorCheckCodexSeatNetwork_DefaultConfigAllClaudeRolesReports(t *testing.T) {
	orgCfg := config.Default().Org
	for role, entry := range orgCfg.RoleModels {
		if entry.Driver != "claude" {
			t.Fatalf("precondition: config.Default() puts role %s on %q, want every role on claude", role, entry.Driver)
		}
	}
	if r := checkCodexSeatNetwork(orgCfg); r.Status != "info" || r.Detail != wantCodexSeatNetworkOpen {
		t.Errorf("default config: checkCodexSeatNetwork = (%s, %q), want (info, %q)", r.Status, r.Detail, wantCodexSeatNetworkOpen)
	}
	orgCfg.Permissions.CodexNetworkAccess = false
	if r := checkCodexSeatNetwork(orgCfg); r.Status != "warn" || r.Detail != wantCodexSeatNetworkClosed {
		t.Errorf("default config, key false: checkCodexSeatNetwork = (%s, %q), want (warn, %q)", r.Status, r.Detail, wantCodexSeatNetworkClosed)
	}
}

// TestDoctorCheckCodexSeatNetwork_DirectorOnlyWorkspaceWriteIsNotCounted:
// ralph org director start refuses a codex director in edits and
// autonomous, so a director entry in those modes under a guarded default
// starts no codex agent under workspace-write (codexSeatModesPossible, as
// in Check 11b), and the key, false here, raises nothing.
func TestDoctorCheckCodexSeatNetwork_DirectorOnlyWorkspaceWriteIsNotCounted(t *testing.T) {
	for _, mode := range []string{"edits", "autonomous"} {
		t.Run(mode, func(t *testing.T) {
			orgCfg := config.OrgConfig{
				DriverPool: []string{"claude", "codex"},
				Permissions: config.OrgPermissionsConfig{
					Default:       "guarded",
					Roles:         map[string]string{"director": mode},
					CodexVerified: true,
				},
			}
			if r := checkCodexSeatNetwork(orgCfg); r.Status != "pass" || r.Detail != wantCodexSeatNetworkAllGuarded {
				t.Errorf("checkCodexSeatNetwork = (%s, %q), want (pass, %q)", r.Status, r.Detail, wantCodexSeatNetworkAllGuarded)
			}
		})
	}
}

// initDoctorProjectWithStubs scaffolds a v2 project (initV2Project, whose
// scaffold checks pass, so --strict fails only on what the test adds),
// appends extraToml to its ralph.toml, and puts stub claude, codex, and go
// on PATH, with agmsg pointed at a missing directory.
func initDoctorProjectWithStubs(t *testing.T, extraToml string) string {
	t.Helper()
	dir := initV2Project(t, gen1(), "1.0.0-test")
	if extraToml != "" {
		tomlPath := filepath.Join(dir, "ralph.toml")
		data, err := os.ReadFile(tomlPath)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "[org") {
			t.Fatalf("precondition: the test scaffold's ralph.toml has an [org] table already:\n%s", data)
		}
		if err := os.WriteFile(tomlPath, append(data, []byte(extraToml)...), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, bin := range []string{"claude", "codex", "go"} {
		writeStubBin(t, binDir, bin, "")
	}
	t.Setenv("PATH", binDir)
	setAgmsgHomeOverride(t, filepath.Join(dir, "no-such-agmsg-home"))
	return dir
}

// TestRunDoctorFull_CodexSeatNetworkLine is the wiring pin through
// runDoctorFull: the scaffolded project (codex in driver_pool, the
// autonomous default, codex_verified and codex_network_access true) prints
// the info line, and codex_network_access = false in ralph.toml prints the
// warn line, which leaves doctor's exit code alone without --strict and
// with it: --strict raises only the scaffold checks.
func TestRunDoctorFull_CodexSeatNetworkLine(t *testing.T) {
	t.Run("default config prints the info", func(t *testing.T) {
		dir := initDoctorProjectWithStubs(t, "")
		var out string
		if err := captureStdoutText(t, &out, func() error { return runDoctorFull(dir, false, false) }); err != nil {
			t.Fatalf("runDoctorFull: %v\noutput:\n%s", err, out)
		}
		if want := "  ℹ Codex sandbox (seat network): info — " + wantCodexSeatNetworkOpen + "\n"; !strings.Contains(out, want) {
			t.Errorf("doctor output does not contain %q:\n%s", want, out)
		}
	})
	t.Run("codex_network_access = false prints the warn, also under --strict", func(t *testing.T) {
		dir := initDoctorProjectWithStubs(t, "\n[org.permissions]\ncodex_network_access = false\n")
		want := "  ⚠ Codex sandbox (seat network): warn — " + wantCodexSeatNetworkClosed + "\n"
		for _, strict := range []bool{false, true} {
			var out string
			err := captureStdoutText(t, &out, func() error { return runDoctorFull(dir, false, strict) })
			if err != nil {
				t.Errorf("strict=%v: runDoctorFull = %v, want nil (the warn must not change the exit code)\noutput:\n%s", strict, err, out)
			}
			if !strings.Contains(out, want) {
				t.Errorf("strict=%v: doctor output does not contain %q:\n%s", strict, want, out)
			}
		}
	})
}
