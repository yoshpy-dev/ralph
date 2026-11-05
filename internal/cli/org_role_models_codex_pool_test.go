package cli

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// TestOrgSpawnAndDirectorStart_ClaudeEntryOutsideACodexPool_Refused is the
// edge case of the plan's test plan (docs/plans/active/2026-11-05-org-role-models.md,
// "driver_pool = [\"codex\"] で claude のエントリを明示する") in the direction
// the other tests do not take: driver_pool = ["codex"] with a role's entry
// on claude. config.Load reads it (D9), and a launch without --driver and
// --model is refused by org.ResolveRoleModel naming the role, its entry, and
// driver_pool, with nothing written: for spawn (no manifest event, receipt,
// herdr, or agmsg call) and for director start, whose own path checks no
// driver_pool, so this refusal is the only thing that keeps a claude director
// out of a codex-only pool.
func TestOrgSpawnAndDirectorStart_ClaudeEntryOutsideACodexPool_Refused(t *testing.T) {
	const codexPool = "[org]\ndriver_pool = [\"codex\"]\n\n[org.permissions]\ndefault = \"guarded\"\n\n"

	t.Run("spawn", func(t *testing.T) {
		herdrLog, agmsgLog := setupOrgStubPATH(t)
		t.Chdir(t.TempDir())
		stateDir := filepath.Join(t.TempDir(), "state")
		configPath := roleModelsConfig(t, codexPool+"[org.role_models]\nreviewer = { driver = \"claude\", model = \"opus\" }\n")

		stdout, stderr, err := runOrgCmdStreams(t, "spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "reviewer",
			"--cwd", t.TempDir(), "--scope", "test-scope", "--state-dir", stateDir, "--config", configPath)
		if err == nil {
			t.Fatalf("spawn succeeded; want refused (stdout: %s)", stdout)
		}
		want := `org: role "reviewer" runs on driver "claude" in [org.role_models].reviewer, which is not in [org].driver_pool [codex]`
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
		assertNoModelFallbackWarning(t, stderr)
		if events := readManifestEvents(t, org.ManifestPathIn(stateDir)); len(events) != 0 {
			t.Errorf("manifest = %v; want nothing recorded", eventTypes(events))
		}
		if _, err := os.Stat(org.ReceiptsPathIn(stateDir)); !os.IsNotExist(err) {
			t.Errorf("stat receipts = %v; want no receipt written", err)
		}
		if lines := slices.Concat(readLogLines(t, herdrLog), readLogLines(t, agmsgLog)); len(lines) != 0 {
			t.Errorf("herdr and agmsg calls = %v; want none", lines)
		}
	})

	t.Run("director start", func(t *testing.T) {
		stateDir, herdrLog, _ := directorStartLedger(t)
		configPath := writeDirectorConfig(t, codexPool+"[org.role_models]\ndirector = { driver = \"claude\", model = \"\" }\n")
		before := directorFileOrAbsent(t, stateDir)

		stdout, _, err := runOrgCmdStreams(t, "director", "start", "--state-dir", stateDir, "--config", configPath)
		if err == nil {
			t.Fatalf("director start succeeded; want refused (stdout: %s)", stdout)
		}
		want := `org: role "director" runs on driver "claude" in [org.role_models].director, which is not in [org].driver_pool [codex]`
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
		if stdout != "" {
			t.Errorf("stdout = %q; want nothing", stdout)
		}
		if after := directorFileOrAbsent(t, stateDir); after != before {
			t.Errorf("a refused start changed director.jsonl:\nbefore: %s\nafter:  %s", before, after)
		}
		if lines := readLogLines(t, herdrLog); len(lines) != 0 {
			t.Errorf("herdr calls = %v; want none", lines)
		}
		if _, err := os.Stat(filepath.Join(stateDir, "prompts")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("stat prompts/ = %v; want no prompt file written", err)
		}
	})
}
