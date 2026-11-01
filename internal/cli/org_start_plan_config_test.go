package cli

import (
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for how `ralph org start --plan` reads its config (plan
// docs/plans/active/2026-11-02-standard-config-gaps.md, AC2 and AC3): each
// file once, nothing started when one cannot be read, and the runtime it
// builds from that read still repeats --state-dir in its stop hints. Like the
// other start --plan tests, they run the real scripts/ralph-worktree.sh in a
// temp git repo and chdir into it, so none of them runs in parallel.

// countConfigReads replaces loadConfigFile, for the rest of the test, with
// one that counts the calls for each file by its absolute path and then
// loads it as config.Load does.
func countConfigReads(t *testing.T) map[string]int {
	t.Helper()
	reads := map[string]int{}
	orig := loadConfigFile
	loadConfigFile = func(path string) (config.Config, error) {
		abs, err := filepath.Abs(path)
		if err != nil {
			t.Fatalf("make the config path %s absolute: %v", path, err)
		}
		reads[abs]++
		return orig(path)
	}
	t.Cleanup(func() { loadConfigFile = orig })
	return reads
}

// newFeatureRepoSub is newFeatureRepo with an empty subdirectory pkg/sub of
// the main checkout (git keeps no empty directory, so the checkout stays
// clean). It returns the root and the subdirectory.
func newFeatureRepoSub(t *testing.T, gitignore string) (root, sub string) {
	t.Helper()
	root = newFeatureRepo(t, gitignore)
	sub = filepath.Join(root, "pkg", "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	return root, sub
}

// TestOrgStartPlan_ReadsEachConfigFileOnce is AC2's count: start --plan, run
// from a subdirectory of the main checkout, reads the ./ralph.toml there (or
// the --config file) once, and the main worktree's ralph.toml once more, as
// the override of the main worktree, only when there is no --config and the
// ledger is the main worktree's. The code before this plan read the main
// worktree's ralph.toml twice, and with --config or another ledger the cwd's
// file or the --config file twice.
func TestOrgStartPlan_ReadsEachConfigFileOnce(t *testing.T) {
	cases := []struct {
		name   string
		ledger string // "main" (the default ledger, the main worktree's) or "outside" (--state-dir in a temp dir)
		config bool   // pass --config <root>/.harness/custom.toml
	}{
		{"the main worktree's ledger", "main", false},
		{"--config", "main", true},
		{"a ledger outside the main worktree", "outside", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupOrgStubPATH(t)
			root, sub := newFeatureRepoSub(t, ".claude/worktrees/\n.harness/\n")
			stateDir := filepath.Join(root, ".harness", "state", "org")
			var extra []string
			if tc.ledger == "outside" {
				tmp, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				stateDir = filepath.Join(tmp, "state")
				extra = append(extra, "--state-dir", stateDir)
			}
			plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
			want := map[string]int{filepath.Join(sub, "ralph.toml"): 1}
			if tc.config {
				path := filepath.Join(root, ".harness", "custom.toml") // .harness/ exists: the split plan is under it
				writeTestFile(t, path, "[org]\nmax_seats = 5\n")
				extra = append(extra, "--config", path)
				want = map[string]int{path: 1}
			} else if tc.ledger == "main" {
				want[filepath.Join(root, "ralph.toml")] = 1
			}
			t.Chdir(sub)
			reads := countConfigReads(t)

			stdout, stderr, err := startOrgFeature(t, plan, "b", extra...)
			if err != nil {
				t.Fatalf("start --plan --feature b: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			if !maps.Equal(reads, want) {
				t.Errorf("config reads = %v, want %v", reads, want)
			}
		})
	}
}

// TestOrgStartPlan_UnreadableConfigStartsNothing is AC3: when the main
// worktree's ralph.toml (committed, so the checkout is clean) or the --config
// file is not valid TOML, start --plan stops with an error, the first naming
// the file and every value start reads from it, and makes no feature
// worktree, no branch, no record of ralph-worktree.sh, no manifest event (a
// feature binding is one), and no herdr call.
func TestOrgStartPlan_UnreadableConfigStartsNothing(t *testing.T) {
	const broken = "[org\nmax_seats = 5\n"
	cases := []struct {
		name string
		main bool // the main worktree's ralph.toml is broken; otherwise the --config file is
	}{
		{"the main worktree's ralph.toml", true},
		{"the --config file", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, _ := setupOrgStubPATH(t)
			root, sub := newFeatureRepoSub(t, ".claude/worktrees/\n.harness/\n")
			stateDir := filepath.Join(root, ".harness", "state", "org")
			plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
			want := "org: load config: "
			var extra []string
			if tc.main {
				writeTestFile(t, filepath.Join(root, "ralph.toml"), broken)
				runLedgerTestGit(t, "-C", root, "add", "ralph.toml")
				runLedgerTestGit(t, "-C", root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
					"commit", "--quiet", "-m", "ralph.toml")
				want = "org: load max_orgs, max_total_seats, [org.approval].mode and [reviewer] from the main worktree's " +
					filepath.Join(root, "ralph.toml") + ": "
			} else {
				path := filepath.Join(root, ".harness", "custom.toml") // .harness/ exists: the split plan is under it
				writeTestFile(t, path, broken)
				extra = []string{"--config", path}
			}
			t.Chdir(sub)

			stdout, stderr, err := startOrgFeature(t, plan, "b", extra...)
			if err == nil || !strings.HasPrefix(err.Error(), want) {
				t.Fatalf("expected start refused with %q..., got %v (stdout: %s stderr: %s)", want, err, stdout, stderr)
			}
			if wt := filepath.Join(root, ".claude", "worktrees", "org-b"); !pathMissing(t, wt) {
				t.Errorf("expected no feature worktree %s", wt)
			}
			if exec.Command("git", "-C", root, "rev-parse", "--verify", "--quiet", "refs/heads/fix/b").Run() == nil {
				t.Errorf("expected no branch fix/b")
			}
			if records := filepath.Join(root, ".git", "ralph", "worktrees"); !pathMissing(t, records) {
				t.Errorf("expected no ralph-worktree.sh record under %s", records)
			}
			if events := readManifestEvents(t, org.ManifestPathIn(stateDir)); len(events) != 0 {
				t.Errorf("expected nothing recorded, got %v", eventTypes(events))
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("expected no herdr call, got %v", lines)
			}
		})
	}
}

// pathMissing reports whether path does not exist, failing the test when it
// cannot tell.
func pathMissing(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
	return os.IsNotExist(err)
}

// TestOrgStartPlan_InFlightLeaderRefusal_StopHintRepeatsStateDir is
// TestOrgSpawn_InFlightSeatRefusal_StopHintRepeatsStateDir for start --plan,
// which builds its runtime from the config it read (AC2): with the ledger
// named by --state-dir and the feature's leader left with an unfinished
// spawn, the refusal of a model outside the pool ends with the stop command
// and that --state-dir, quoted. The refusal comes before the worktree is
// made, writes nothing, and calls no herdr.
func TestOrgStartPlan_InFlightLeaderRefusal_StopHintRepeatsStateDir(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	root := newFeatureRepo(t, ".claude/worktrees/\n")
	t.Chdir(root)
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(tmp, "state dir")
	plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
	store := org.NewManifestStoreAtPath(org.ManifestPathIn(stateDir))
	for _, ev := range []org.ManifestEvent{
		{TS: "2026-08-01T00:00:00Z", OrgID: "b", SeatID: org.LeaderIdentity, Event: org.EventSpawnStarted, Role: org.LeaderIdentity, Driver: "claude", Model: "sonnet"},
		{TS: "2026-08-01T00:00:01Z", OrgID: "b", SeatID: org.LeaderIdentity, Event: org.EventSpawnStep, PaneID: "pane-7", Details: "tab_created"},
	} {
		if err := store.Append(ev); err != nil {
			t.Fatalf("seed %s: %v", ev.Event, err)
		}
	}

	stdout, stderr, err := startOrgFeature(t, plan, "b", "--state-dir", stateDir, "--model", "not-a-real-model")
	want := "; seat \"leader\" in org_id \"b\" has an unfinished spawn on pane \"pane-7\", left as it is: " +
		"end it with ralph org stop --org-id b --seat leader --state-dir '" + stateDir + "'"
	if err == nil || !strings.HasSuffix(err.Error(), want) {
		t.Fatalf("expected the refusal to end with %q, got %v (stdout: %s stderr: %s)", want, err, stdout, stderr)
	}
	if wt := filepath.Join(root, ".claude", "worktrees", "org-b"); !pathMissing(t, wt) {
		t.Errorf("expected no feature worktree %s", wt)
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("expected no herdr call for the refusal, got %v", lines)
	}
	if events := readManifestEvents(t, org.ManifestPathIn(stateDir)); len(events) != 2 {
		t.Errorf("expected the two seeded events only, got %v", eventTypes(events))
	}
}
