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
// file once, nothing started when one cannot be read, the org-wide limits
// taken from the main worktree's ralph.toml, and the runtime it builds from
// that read still repeats --state-dir in its stop hints. Like the other
// start --plan tests, they run the real scripts/ralph-worktree.sh in a temp
// git repo and chdir into it, so none of them runs in parallel.

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
// ledger is the main worktree's. Run from the main worktree's root, those two
// are the same file, which is read twice, once for each role, and no more
// often. The code before this plan read the main worktree's ralph.toml twice
// from the subdirectory, and with --config or another ledger the cwd's file or
// the --config file twice.
func TestOrgStartPlan_ReadsEachConfigFileOnce(t *testing.T) {
	cases := []struct {
		name          string
		atRoot        bool // run from the main worktree's root, not from pkg/sub
		outsideLedger bool // --state-dir in a temp dir; otherwise the default ledger, the main worktree's
		config        bool // pass --config <root>/.harness/custom.toml
	}{
		{"the main worktree's ledger", false, false, false},
		{"the main worktree's root", true, false, false},
		{"--config", false, false, true},
		{"a ledger outside the main worktree", false, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupOrgStubPATH(t)
			root, sub := newFeatureRepoSub(t, ".claude/worktrees/\n.harness/\n")
			cwd := sub
			if tc.atRoot {
				cwd = root
			}
			stateDir := filepath.Join(root, ".harness", "state", "org")
			var extra []string
			if tc.outsideLedger {
				stateDir = outsideLedgerDir(t, "state")
				extra = append(extra, "--state-dir", stateDir)
			}
			plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
			want := map[string]int{filepath.Join(cwd, "ralph.toml"): 1}
			switch {
			case tc.config:
				path := filepath.Join(root, ".harness", "custom.toml") // .harness/ exists: the split plan is under it
				writeTestFile(t, path, "[org]\nmax_seats = 5\n")
				extra = append(extra, "--config", path)
				want = map[string]int{path: 1}
			case !tc.outsideLedger:
				want[filepath.Join(root, "ralph.toml")]++ // 2 when cwd is root
			}
			t.Chdir(cwd)
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
// file cannot be read, start --plan stops with an error, the first naming
// the file and every value start reads from it, and makes no feature
// worktree, no branch, no record of ralph-worktree.sh, no manifest event (a
// feature binding is one), and no herdr call. The file that cannot be read is
// invalid TOML, not one without permission: config.Load fails on it the same
// way for every user, root included, and the test checks that the cause after
// the prefix is a TOML error, so a failure for another reason does not pass.
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
			// The path in the first message ends in ".toml", so look at what follows the prefix.
			if cause := strings.TrimPrefix(err.Error(), want); !strings.HasPrefix(cause, "toml: ") {
				t.Errorf("expected the cause after %q to be a TOML error, got %q", want, cause)
			}
			assertNoFeatureStart(t, root, "b")
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

// assertNoFeatureStart checks that nothing of a start of feature is left in
// the repository at root: no feature worktree org-<feature>, no branch
// fix/<feature>, and no record of ralph-worktree.sh. The manifest and the
// herdr log are not checked here: what they must hold differs between the
// tests.
func assertNoFeatureStart(t *testing.T, root, feature string) {
	t.Helper()
	if wt := filepath.Join(root, ".claude", "worktrees", "org-"+feature); !pathMissing(t, wt) {
		t.Errorf("expected no feature worktree %s", wt)
	}
	if exec.Command("git", "-C", root, "rev-parse", "--verify", "--quiet", "refs/heads/fix/"+feature).Run() == nil {
		t.Errorf("expected no branch fix/%s", feature)
	}
	if records := filepath.Join(root, ".git", "ralph", "worktrees"); !pathMissing(t, records) {
		t.Errorf("expected no ralph-worktree.sh record under %s", records)
	}
}

// outsideLedgerDir returns a path called name in a new temp directory, outside
// the main worktree, for a ledger named with --state-dir. The directory is
// symlink-resolved, as the paths the commands print are.
func outsideLedgerDir(t *testing.T, name string) string {
	t.Helper()
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(tmp, name)
}

// TestOrgStartPlan_InFlightLeaderRefusal_StopHintRepeatsStateDir is
// TestOrgSpawn_InFlightSeatRefusal_StopHintRepeatsStateDir for start --plan,
// which builds its runtime from the config it read (AC2): with the feature's
// leader left with an unfinished spawn, the refusal of a model outside the
// pool ends with the stop command and, when the ledger was named with
// --state-dir, that --state-dir, quoted; with the same ledger named by the
// personal file's [org].state_dir, it ends with the plain command. The
// refusal comes before the worktree is made, writes nothing, and calls no
// herdr.
func TestOrgStartPlan_InFlightLeaderRefusal_StopHintRepeatsStateDir(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	root := newFeatureRepo(t, ".claude/worktrees/\n")
	t.Chdir(root)
	stateDir := outsideLedgerDir(t, "state dir")
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

	const stop = "; seat \"leader\" in org_id \"b\" has an unfinished spawn on pane \"pane-7\", left as it is: " +
		"end it with ralph org stop --org-id b --seat leader"

	t.Run("--state-dir", func(t *testing.T) {
		stdout, stderr, err := startOrgFeature(t, plan, "b", "--state-dir", stateDir, "--model", "not-a-real-model")
		if want := stop + " --state-dir '" + stateDir + "'"; err == nil || !strings.HasSuffix(err.Error(), want) {
			t.Fatalf("expected the refusal to end with %q, got %v (stdout: %s stderr: %s)", want, err, stdout, stderr)
		}
		if wt := filepath.Join(root, ".claude", "worktrees", "org-b"); !pathMissing(t, wt) {
			t.Errorf("expected no feature worktree %s", wt)
		}
	})
	t.Run("[org].state_dir", func(t *testing.T) {
		r := usePersonalStateDir(t, stateDir) // chdirs into a linked worktree of its own repository
		stdout, stderr, err := startOrgFeature(t, plan, "b", "--model", "not-a-real-model")
		if err == nil || !strings.HasSuffix(err.Error(), stop) || strings.Contains(err.Error(), "--state-dir") {
			t.Fatalf("expected the refusal to end with %q and name no --state-dir, got %v (stdout: %s stderr: %s)", stop, err, stdout, stderr)
		}
		if wt := filepath.Join(r.mainRoot, ".claude", "worktrees", "org-b"); !pathMissing(t, wt) {
			t.Errorf("expected no feature worktree %s", wt)
		}
	})

	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("expected no herdr call for the refusal, got %v", lines)
	}
	if events := readManifestEvents(t, org.ManifestPathIn(stateDir)); len(events) != 2 {
		t.Errorf("expected the two seeded events only, got %v", eventTypes(events))
	}
}

// TestOrgStartPlan_OrgWideLimits_ReadFromMainWorktreeRalphToml is
// TestOrgStart_OrgWideLimits_ReadFromMainWorktreeRalphToml for start --plan,
// which reads the main worktree's ralph.toml itself (resolveOrgStartConfig,
// AC2): with org-a running in the main worktree's ledger, max_orgs = 1 or
// max_total_seats = 1 in the main worktree's ralph.toml refuses feature b
// started from pkg/sub, whose own ralph.toml allows 99, and the refused start
// makes no worktree, branch or ralph-worktree.sh record and calls no herdr.
// Unlike the twin, which finds a rejected event for the refused org, it
// checks that no event of b exists at all: start --plan refuses in
// spawnPrecheckErr (feature.go), before the worktree and the manifest, so no
// rejected event is written either. The other two cases are controls: with
// --config, and with a --state-dir naming a ledger outside the main
// worktree where org-a runs, the main worktree's ralph.toml does not apply
// (its max_orgs = 1 would refuse b), so b starts. That is all they fix, not
// which values b gets instead. The ralph.toml files a case writes are
// committed, so the checkout stays clean for the starts that go through.
func TestOrgStartPlan_OrgWideLimits_ReadFromMainWorktreeRalphToml(t *testing.T) {
	cases := []struct {
		name          string
		mainToml      string
		subToml       string // pkg/sub/ralph.toml; "" writes none
		config        bool   // pass --config <root>/.harness/custom.toml with max_orgs = 99
		outsideLedger bool   // --state-dir in a temp dir; otherwise the default ledger, the main worktree's
		refusal       string // "" when b starts
	}{
		{name: "max_orgs", mainToml: "[org]\nmax_orgs = 1\n", subToml: "[org]\nmax_orgs = 99\n",
			refusal: `max_orgs 1 reached: org_id "b" is not running and 1 orgs are (org-a)`},
		{name: "max_total_seats", mainToml: "[org]\nmax_total_seats = 1\n", subToml: "[org]\nmax_total_seats = 99\n",
			refusal: "max_total_seats 1 reached: 1 seats are active across all orgs"},
		{name: "--config", mainToml: "[org]\nmax_orgs = 1\n", config: true},
		{name: "a ledger outside the main worktree", mainToml: "[org]\nmax_orgs = 1\n", subToml: "[org]\nmax_orgs = 99\n", outsideLedger: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, _ := setupOrgStubPATH(t)
			root, sub := newFeatureRepoSub(t, ".claude/worktrees/\n.harness/\n")
			writeTestFile(t, filepath.Join(root, "ralph.toml"), tc.mainToml)
			files := []string{"ralph.toml"}
			if tc.subToml != "" {
				writeTestFile(t, filepath.Join(sub, "ralph.toml"), tc.subToml)
				files = append(files, filepath.Join("pkg", "sub", "ralph.toml"))
			}
			runLedgerTestGit(t, append([]string{"-C", root, "add"}, files...)...)
			runLedgerTestGit(t, "-C", root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
				"commit", "--quiet", "-m", "ralph.toml")

			stateDir := filepath.Join(root, ".harness", "state", "org")
			var extra []string
			if tc.outsideLedger {
				stateDir = outsideLedgerDir(t, "state")
				extra = append(extra, "--state-dir", stateDir)
			}
			t.Chdir(root)
			if _, stderr, err := startOrgLeader(t, "org-a", "--scope", "org-a", "--state-dir", stateDir); err != nil {
				t.Fatalf("start org-a in %s: %v (stderr: %s)", stateDir, err, stderr)
			}
			plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
			if tc.config {
				path := filepath.Join(root, ".harness", "custom.toml") // .harness/ exists: the split plan is under it
				writeTestFile(t, path, "[org]\nmax_orgs = 99\n")
				extra = append(extra, "--config", path)
			}
			herdrCalls := len(readLogLines(t, herdrLog))
			t.Chdir(sub)

			stdout, stderr, err := startOrgFeature(t, plan, "b", extra...)
			if tc.refusal == "" {
				if err != nil {
					t.Fatalf("expected b started under the caller's config, got %v (stdout: %s stderr: %s)", err, stdout, stderr)
				}
				if last := lastSeatEvent(t, stateDir, "b", org.LeaderIdentity); last.Event != org.EventSpawned {
					t.Errorf("expected b's leader spawned, last event %+v", last)
				}
				if wt := filepath.Join(root, ".claude", "worktrees", "org-b"); pathMissing(t, wt) {
					t.Errorf("expected the feature worktree %s", wt)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.refusal) {
				t.Fatalf("expected b refused with %q from the main worktree's ralph.toml, got %v (stdout: %s stderr: %s)", tc.refusal, err, stdout, stderr)
			}
			assertNoFeatureStart(t, root, "b")
			for _, ev := range readManifestEvents(t, org.ManifestPathIn(stateDir)) {
				if ev.OrgID == "b" {
					t.Errorf("expected nothing recorded for b, got %+v", ev)
				}
			}
			if lines := readLogLines(t, herdrLog); len(lines) != herdrCalls {
				t.Errorf("expected no herdr call for the refusal, got %v after the first %d", lines[herdrCalls:], herdrCalls)
			}
		})
	}
}
