package cli

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// TestOrgCleanupVerbs_LegacyModelKeys_StillWork covers the half of AC8
// (plan docs/plans/active/2026-11-05-org-role-models.md; spec
// 2026-11-05-org-role-models, acceptance 11) that the CLI tests did not
// run: a ralph.toml that still sets [org.roles] or [org].model_pool loads,
// so `ralph org status`, `ralph org stop`, and `ralph org disband` keep
// working under it, on the real path (herdr stub, no --dry-run). Only the
// verbs that start seats refuse [org.roles] (TestOrgSpawn_LegacyOrgRolesRefused).
// The seats are spawned under a clean config first, as an org started
// before the operator upgraded would have them.
func TestOrgCleanupVerbs_LegacyModelKeys_StillWork(t *testing.T) {
	const orgTable = "[org]\nmax_seats = 5\ndriver_pool = [\"claude\"]\n"
	const modelPool = "model_pool = [{ driver = \"claude\", model = \"sonnet\" }]\n"
	const rolesTable = "\n[org.roles]\nimplementer = [\"sonnet\"]\n"
	cases := []struct {
		name   string
		config string
	}{
		{"[org.roles]", orgTable + rolesTable},
		{"[org].model_pool", orgTable + modelPool},
		{"both keys", orgTable + modelPool + rolesTable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, _ := setupOrgStubPATH(t)
			stateDir := filepath.Join(t.TempDir(), "state")
			cleanConfig := writeOrgConfig(t, t.TempDir(), 5)
			legacyConfig := roleModelsConfig(t, tc.config)

			for _, id := range []string{"seat-1", "seat-2"} {
				out, err := runOrgCmd(t,
					"spawn", "--org-id", "org-a", "--id", id, "--role", "worker",
					"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
					"--scope", "test-scope", "--state-dir", stateDir, "--config", cleanConfig)
				if err != nil {
					t.Fatalf("spawn %s with the clean config failed: %v (output: %s)", id, err, out)
				}
			}
			seat1Pane := lastSeatEvent(t, stateDir, "org-a", "seat-1").PaneID
			if seat1Pane == "" {
				t.Fatalf("seat-1 has no pane id after spawn")
			}

			out, err := runOrgCmd(t, "status", "--org-id", "org-a", "--state-dir", stateDir, "--config", legacyConfig)
			if err != nil {
				t.Fatalf("status under %s failed: %v (output: %s)", tc.name, err, out)
			}
			if !strings.Contains(out, "seat-1") || !strings.Contains(out, "seat-2") {
				t.Errorf("expected both seats in status output, got: %s", out)
			}

			out, err = runOrgCmd(t, "stop", "--org-id", "org-a", "--seat", "seat-1", "--state-dir", stateDir, "--config", legacyConfig)
			if err != nil {
				t.Fatalf("stop under %s failed: %v (output: %s)", tc.name, err, out)
			}
			if last := lastSeatEvent(t, stateDir, "org-a", "seat-1"); last.Event != org.EventStopped || last.DryRun {
				t.Errorf("seat-1's last event = %+v; want a real %q", last, org.EventStopped)
			}
			if lines := readLogLines(t, herdrLog); !slices.Contains(lines, "pane close "+seat1Pane) {
				t.Errorf("herdr calls = %v; want %q from the real stop", lines, "pane close "+seat1Pane)
			}

			out, err = runOrgCmd(t, "disband", "--org-id", "org-a", "--state-dir", stateDir, "--config", legacyConfig)
			if err != nil {
				t.Fatalf("disband under %s failed: %v (output: %s)", tc.name, err, out)
			}
			if !strings.Contains(out, `disbanded org "org-a"`) {
				t.Errorf("expected the disband confirmation, got: %s", out)
			}
			if got := realDisbandedOrgs(t, stateDir); !slices.Equal(got, []string{"org-a"}) {
				t.Errorf("real disbanded orgs = %v; want [org-a]", got)
			}
			if last := lastSeatEvent(t, stateDir, "org-a", "seat-2"); last.Event != org.EventStopped || last.DryRun {
				t.Errorf("seat-2's last event = %+v; want a real %q from disband", last, org.EventStopped)
			}
		})
	}
}
