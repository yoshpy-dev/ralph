package cli

import (
	"encoding/json"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org start --plan <split plan> --feature <slug>` and the
// feature line of `ralph org status` (plan
// docs/plans/active/2026-10-09-org-feature-worktree.md, the CLI side of AC3,
// AC5, AC9 and AC11). The tests that make a worktree run the repo's real
// scripts/ralph-worktree.sh in a temp git repo and chdir into it, so none of
// them runs in parallel.

// featureSplitPlan is a split plan with two features: a (no `- Type:`, so a
// feat/ branch) and b. Its approval line holds a placeholder digest;
// approvedFeatureSplitPlan fills it in.
const featureSplitPlan = `# demo split

- Status: Approved
- Approved: 2026-10-09 sha256:000000000000

## Features

### a

- Reserve: internal/a/
- Depends on: none

Objective: build a.

### b

- Type: fix
- Reserve: docs/b.md
- Depends on: a

Objective: fix b.
`

// approvedFeatureSplitPlan is featureSplitPlan with its digest on the
// `- Approved:` line (org.PlanDigest skips that line, so the digest stays the
// same after it is written).
func approvedFeatureSplitPlan(t *testing.T) string {
	t.Helper()
	return strings.Replace(featureSplitPlan, "sha256:000000000000", "sha256:"+org.PlanDigest([]byte(featureSplitPlan)), 1)
}

// writeFeatureSplitPlan writes content as the split plan demo in stateDir's
// splits/ and returns its path.
func writeFeatureSplitPlan(t *testing.T, stateDir, content string) string {
	t.Helper()
	dir := org.SplitPlansDirIn(stateDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "demo.md")
	writeTestFile(t, path, content)
	return path
}

// startOrgFeature runs `ralph org start --plan plan --feature feature` through
// the stub herdr/agmsg (setupOrgStubPATH), with extra flags after them.
func startOrgFeature(t *testing.T, plan, feature string, extra ...string) (stdout, stderr string, err error) {
	t.Helper()
	args := append([]string{"start", "--plan", plan, "--feature", feature, "--driver", "claude", "--model", "sonnet"}, extra...)
	return runOrgCmdStreams(t, args...)
}

// TestOrgStartPlan_FlagCombinationsRefused is AC5's flag half: --plan and
// --feature go together, and the --plan form refuses a task argument, the
// flags whose values the split plan supplies, and a malformed --org-id. Each
// refusal comes before the state dir is resolved, so nothing is recorded and
// herdr is not called.
func TestOrgStartPlan_FlagCombinationsRefused(t *testing.T) {
	const supplies = "with --plan the split plan supplies the leader's cwd (the feature worktree), its scope and its reservation"
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"--plan without --feature", []string{"--plan", "demo.md"}, []string{"org: start: --plan and --feature go together"}},
		{"--feature without --plan", []string{"--feature", "a"}, []string{"org: start: --plan and --feature go together"}},
		{"--feature without --plan, with a task", []string{"--feature", "a", "task text"}, []string{"--plan and --feature go together"}},
		{"a blank --plan", []string{"--plan", " ", "--feature", "a"}, []string{"org: start: --plan and --feature must not be blank"}},
		{"a blank --feature", []string{"--plan", "demo.md", "--feature", ""}, []string{"--plan and --feature must not be blank"}},
		{"a task argument", []string{"--plan", "demo.md", "--feature", "a", "task text"},
			[]string{`org: start: --plan takes no task argument (got "task text")`, supplies}},
		{"--cwd", []string{"--plan", "demo.md", "--feature", "a", "--cwd", "."}, []string{"org: start: --plan cannot be combined with --cwd: " + supplies}},
		{"--scope", []string{"--plan", "demo.md", "--feature", "a", "--scope", "s"}, []string{"--plan cannot be combined with --scope: " + supplies}},
		{"--reserve", []string{"--plan", "demo.md", "--feature", "a", "--reserve", "docs/"}, []string{"--plan cannot be combined with --reserve: " + supplies}},
		{"--allow-unscoped", []string{"--plan", "demo.md", "--feature", "a", "--allow-unscoped"},
			[]string{"--plan cannot be combined with --allow-unscoped: " + supplies}},
		{"--allow-unscoped=false", []string{"--plan", "demo.md", "--feature", "a", "--allow-unscoped=false"},
			[]string{"--plan cannot be combined with --allow-unscoped"}},
		{"a malformed --org-id", []string{"--plan", "demo.md", "--feature", "a", "--org-id", "../x"}, []string{"org_id"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, _ := setupOrgStubPATH(t)
			stateDir := filepath.Join(t.TempDir(), "state")

			stdout, _, err := runOrgCmdStreams(t, append([]string{"start", "--state-dir", stateDir}, tc.args...)...)
			if err == nil {
				t.Fatalf("expected start refused, stdout: %s", stdout)
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("expected the error to contain %q, got: %v", want, err)
				}
			}
			if _, statErr := os.Stat(stateDir); !os.IsNotExist(statErr) {
				t.Errorf("expected no state dir to be made, stat: %v", statErr)
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("expected no herdr call, got %v", lines)
			}
		})
	}
}

// TestOrgStart_WithoutPlan_StillTakesExactlyOneTask pins the old form's
// argument check, which the --plan form replaced with its own: without --plan
// or --feature, start still wants exactly one task argument, with cobra's
// message, before --org-id is looked at.
func TestOrgStart_WithoutPlan_StillTakesExactlyOneTask(t *testing.T) {
	for name, args := range map[string][]string{
		"no task":                 {"start", "--org-id", "org-a", "--cwd", "."},
		"two tasks":               {"start", "--org-id", "org-a", "--cwd", ".", "one", "two"},
		"no task and no --org-id": {"start", "--cwd", "."},
	} {
		t.Run(name, func(t *testing.T) {
			stateDir := filepath.Join(t.TempDir(), "state")
			_, _, err := runOrgCmdStreams(t, append(args, "--state-dir", stateDir)...)
			if err == nil || !strings.Contains(err.Error(), "accepts 1 arg(s), received") {
				t.Fatalf("expected cobra's argument count error, got: %v", err)
			}
		})
	}
}

// TestOrgStartPlan_OutsideGitRepositoryRefused: with the ledger chosen by
// --state-dir, start --plan takes the main worktree from the current
// directory's repository, and outside one it is refused before anything is
// recorded.
func TestOrgStartPlan_OutsideGitRepositoryRefused(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	isolateFeatureGitEnv(t)
	cwd := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(cwd)) // no repository above the temp dir either
	t.Chdir(cwd)
	stateDir := filepath.Join(t.TempDir(), "state")
	plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))

	_, _, err := startOrgFeature(t, plan, "a", "--state-dir", stateDir)
	if err == nil || !strings.Contains(err.Error(), "org: start --plan makes the feature worktree from the main worktree of a git repository, "+
		"and the current directory is not in one") {
		t.Fatalf("expected start refused outside a git repository, got: %v", err)
	}
	if events := readManifestEvents(t, org.ManifestPathIn(stateDir)); len(events) != 0 {
		t.Errorf("expected nothing recorded, got %v", eventTypes(events))
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("expected no herdr call, got %v", lines)
	}
}

// isolateFeatureGitEnv keeps git away from the developer's config and from
// GIT_DIR-style variables a git hook may export, and clears
// RALPH_ORG_STATE_DIR so the ledger is the one each test picks.
func isolateFeatureGitEnv(t *testing.T) {
	t.Helper()
	isolateGitConfig(t)
	for _, key := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unsetenv %s: %v", key, err)
		}
	}
	t.Setenv(org.EnvOrgStateDir, "")
	if err := os.Unsetenv(org.EnvOrgStateDir); err != nil {
		t.Fatalf("unsetenv %s: %v", org.EnvOrgStateDir, err)
	}
}

// newFeatureRepo makes a git repo on main (symlinks resolved, as git prints
// it) whose one commit holds this repo's scripts/ralph-worktree.sh and
// scripts/ralph-common.sh and a .gitignore with gitignore, and returns its
// root. bash, git and jq must be on PATH: CI has all three, so a missing one
// fails the test instead of skipping it.
func newFeatureRepo(t *testing.T, gitignore string) string {
	t.Helper()
	for _, tool := range []string{"bash", "git", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Fatalf("%s is needed to run scripts/ralph-worktree.sh: %v", tool, err)
		}
	}
	isolateFeatureGitEnv(t)
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location via runtime.Caller")
	}
	source := filepath.Join(filepath.Dir(thisFile), "..", "..", "scripts")

	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(tmp, "repo")
	runLedgerTestGit(t, "init", "--quiet", "-b", "main", root)
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ralph-worktree.sh", "ralph-common.sh"} {
		data, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "scripts", name), data, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeTestFile(t, filepath.Join(root, ".gitignore"), gitignore)
	runLedgerTestGit(t, "-C", root, "add", ".")
	runLedgerTestGit(t, "-C", root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
		"commit", "--quiet", "-m", "init")
	return root
}

// gitCurrentBranch returns the branch checked out in dir.
func gitCurrentBranch(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "branch", "--show-current").Output()
	if err != nil {
		t.Fatalf("git -C %s branch --show-current: %v", dir, err)
	}
	return strings.TrimSpace(string(out))
}

// orgEventCount returns how many manifest events under stateDir belong to
// orgID.
func orgEventCount(t *testing.T, stateDir, orgID string) int {
	t.Helper()
	n := 0
	for _, ev := range readManifestEvents(t, org.ManifestPathIn(stateDir)) {
		if ev.OrgID == orgID {
			n++
		}
	}
	return n
}

// TestOrgStartPlan_RealWorktreeScript is AC3, AC9 and AC11 end to end through
// the CLI and the real scripts/ralph-worktree.sh, with the ledger chosen by
// --state-dir (so the main worktree comes from the current directory): start
// makes .claude/worktrees/org-a on feat/a from main and spawns the leader
// there, status shows the reservation and the binding, and the same start
// again records nothing. A body edited after the approval and a dirty main
// checkout are refused, with no worktree for the feature and nothing in the
// manifest.
func TestOrgStartPlan_RealWorktreeScript(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	root := newFeatureRepo(t, ".claude/worktrees/\n")
	t.Chdir(root)
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(tmp, "state") // symlinks resolved, as the split plan path in errors is
	approved := approvedFeatureSplitPlan(t)
	plan := writeFeatureSplitPlan(t, stateDir, approved)
	digest := org.PlanDigest([]byte(approved))
	wt := filepath.Join(root, ".claude", "worktrees", "org-a")

	stdout, stderr, err := startOrgFeature(t, plan, "a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("start --plan --feature a: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	lines := outputLines(stdout)
	for _, want := range []string{"worktree: " + wt, "branch: feat/a", "hint: ralph org status --org-id a ; attach with herdr to observe the leader pane"} {
		if !slices.Contains(lines, want) {
			t.Errorf("expected the line %q in start's output, got:\n%s", want, stdout)
		}
	}
	if !strings.HasPrefix(stdout, `spawned seat "leader" (org_id=a `) {
		t.Errorf("expected the spawn line first, got:\n%s", stdout)
	}
	if info, err := os.Stat(wt); err != nil || !info.IsDir() {
		t.Fatalf("expected the worktree directory %s, stat: %v", wt, err)
	}
	if got := gitCurrentBranch(t, wt); got != "feat/a" {
		t.Errorf("expected %s on feat/a, got %q", wt, got)
	}
	if got := gitCurrentBranch(t, root); got != "main" {
		t.Errorf("expected the main checkout left on main, got %q", got)
	}
	for _, name := range []string{org.EventSpawnStarted, org.EventSpawned} {
		ev := latestSeatEventNamed(t, stateDir, "a", org.LeaderIdentity, name)
		if ev.Worktree != wt {
			t.Errorf("expected the leader's %s in %s, got %+v", name, wt, ev)
		}
	}
	if !slices.ContainsFunc(readLogLines(t, herdrLog), func(l string) bool {
		return strings.HasPrefix(l, "workspace create --cwd "+wt+" ")
	}) {
		t.Errorf("expected herdr's workspace create in %s, log: %v", wt, readLogLines(t, herdrLog))
	}

	stdout, _, err = runOrgCmdStreams(t, "status", "--org-id", "a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status a: %v", err)
	}
	lines = outputLines(stdout)
	wantFeature := "feature: demo/a branch feat/a worktree " + wt
	if i := slices.Index(lines, "reserved: internal/a/"); i < 0 || i+1 >= len(lines) || lines[i+1] != wantFeature {
		t.Errorf("expected %q right after %q in status, got:\n%s", wantFeature, "reserved: internal/a/", stdout)
	}
	stdout, _, err = runOrgCmdStreams(t, "status", "--org-id", "a", "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status a --json: %v", err)
	}
	payload := orgStatusPayload(t, stdout)
	var feature map[string]string
	if err := json.Unmarshal(payload["feature"], &feature); err != nil {
		t.Fatalf("status --json feature: %v\n%s", err, stdout)
	}
	if want := map[string]string{"split": "demo", "feature": "a", "digest": digest, "branch": "feat/a", "worktree": wt}; !maps.Equal(feature, want) {
		t.Errorf("feature = %v, want %v", feature, want)
	}
	if got := string(payload["reservation"]); strings.Join(strings.Fields(got), "") != `["internal/a/"]` {
		t.Errorf("reservation = %s, want [\"internal/a/\"]", got)
	}

	before := len(readManifestEvents(t, org.ManifestPathIn(stateDir)))
	stdout, stderr, err = startOrgFeature(t, plan, "a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("expected the same start to pass again: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if !strings.Contains(stdout, `seat "leader" already spawned`) || !slices.Contains(outputLines(stdout), "worktree: "+wt) {
		t.Errorf("expected the existing leader and the same worktree, stdout:\n%s", stdout)
	}
	if got := len(readManifestEvents(t, org.ManifestPathIn(stateDir))); got != before {
		t.Errorf("expected the same start to record nothing, %d -> %d events", before, got)
	}

	wtB := filepath.Join(root, ".claude", "worktrees", "org-b")
	assertFeatureStartRefused := func(t *testing.T, want ...string) {
		t.Helper()
		stdout, _, err := startOrgFeature(t, plan, "b", "--state-dir", stateDir)
		if err == nil {
			t.Fatalf("expected start --feature b refused, stdout: %s", stdout)
		}
		for _, w := range want {
			if !strings.Contains(err.Error(), w) {
				t.Errorf("expected the error to contain %q, got: %v", w, err)
			}
		}
		if !strings.HasPrefix(stdout, "rejected: ") {
			t.Errorf("expected the rejected line on stdout, got: %s", stdout)
		}
		if _, statErr := os.Stat(wtB); !os.IsNotExist(statErr) {
			t.Errorf("expected no worktree %s, stat: %v", wtB, statErr)
		}
		if got := len(readManifestEvents(t, org.ManifestPathIn(stateDir))); got != before || orgEventCount(t, stateDir, "b") != 0 {
			t.Errorf("expected nothing recorded, %d -> %d events, %d for b", before, got, orgEventCount(t, stateDir, "b"))
		}
	}

	t.Run("a body edited after the approval", func(t *testing.T) {
		writeTestFile(t, plan, strings.Replace(approved, "Objective: fix b.", "Objective: fix b and c.", 1))
		t.Cleanup(func() { writeTestFile(t, plan, approved) })
		assertFeatureStartRefused(t, "org: split plan "+plan+": changed after its approval: approved digest "+digest)
	})

	t.Run("a dirty main checkout", func(t *testing.T) {
		untracked := filepath.Join(root, "untracked.txt")
		writeTestFile(t, untracked, "x\n")
		t.Cleanup(func() { _ = os.Remove(untracked) })
		assertFeatureStartRefused(t,
			"org: scripts/ralph-worktree.sh ensure: ralph-worktree: base branch 'main' has uncommitted changes",
			"; make the main worktree "+root+" a clean checkout of the default branch and run start again")
	})
}

// TestOrgStartPlan_DefaultLedgerFromSubdirectory runs start --plan from a
// subdirectory of the main checkout without --state-dir: the ledger is the
// main worktree's .harness/state/org (ignored by git, so the checkout stays
// clean), and the worktree is made under the main worktree root, not under
// the subdirectory.
func TestOrgStartPlan_DefaultLedgerFromSubdirectory(t *testing.T) {
	setupOrgStubPATH(t)
	root := newFeatureRepo(t, ".claude/worktrees/\n.harness/\n")
	sub := filepath.Join(root, "pkg", "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	stateDir := filepath.Join(root, ".harness", "state", "org")
	plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))

	stdout, stderr, err := startOrgFeature(t, plan, "b")
	if err != nil {
		t.Fatalf("start --plan --feature b: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	wt := filepath.Join(root, ".claude", "worktrees", "org-b")
	if !slices.Contains(outputLines(stdout), "worktree: "+wt) || !slices.Contains(outputLines(stdout), "branch: fix/b") {
		t.Errorf("expected the worktree %s on fix/b, stdout:\n%s", wt, stdout)
	}
	if got := gitCurrentBranch(t, wt); got != "fix/b" {
		t.Errorf("expected %s on fix/b, got %q", wt, got)
	}
	if ev := latestSeatEventNamed(t, stateDir, "b", org.LeaderIdentity, org.EventSpawned); ev.Worktree != wt {
		t.Errorf("expected the leader spawned in %s in the main worktree's ledger, got %+v", wt, ev)
	}
}

// TestOrgStatus_FeatureLineOnlyForBoundReservation covers AC11's edges: a
// bound reservation record shows the feature line and the JSON feature
// object, also when the org has no seat left, while a plain --reserve
// reservation shows its reserved line with neither.
func TestOrgStatus_FeatureLineOnlyForBoundReservation(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	appendSeedEvent(t, stateDir, org.ManifestEvent{
		OrgID: "bound", Event: org.EventScopeReserved, Worktree: "/repo/.claude/worktrees/org-bound",
		Details: "paths=docs/b.md split=demo feature=b digest=0123456789ab branch=fix/b",
	})
	if _, stderr, err := startOrgLeader(t, "plain", "--state-dir", stateDir, "--reserve", "internal/"); err != nil {
		t.Fatalf("start plain: %v (stderr: %s)", err, stderr)
	}

	stdout, _, err := runOrgCmdStreams(t, "status", "--org-id", "bound", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status bound: %v", err)
	}
	if got, want := outputLines(stdout), []string{"no seats", "reserved: docs/b.md", "feature: demo/b branch fix/b worktree /repo/.claude/worktrees/org-bound"}; !slices.Equal(got, want) {
		t.Errorf("bound's status = %q, want %q", got, want)
	}
	stdout, _, err = runOrgCmdStreams(t, "status", "--org-id", "bound", "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status bound --json: %v", err)
	}
	payload := orgStatusPayload(t, stdout)
	if keys := payloadKeys(payload); !slices.Equal(keys, []string{"corrupt_lines", "feature", "reservation", "seats"}) {
		t.Errorf("bound's status --json keys = %v, want [corrupt_lines feature reservation seats]", keys)
	}
	var feature map[string]string
	if err := json.Unmarshal(payload["feature"], &feature); err != nil {
		t.Fatalf("status --json feature: %v\n%s", err, stdout)
	}
	if want := map[string]string{"split": "demo", "feature": "b", "digest": "0123456789ab", "branch": "fix/b", "worktree": "/repo/.claude/worktrees/org-bound"}; !maps.Equal(feature, want) {
		t.Errorf("feature = %v, want %v", feature, want)
	}

	stdout, _, err = runOrgCmdStreams(t, "status", "--org-id", "plain", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status plain: %v", err)
	}
	if !slices.Contains(outputLines(stdout), "reserved: internal/") || strings.Contains(stdout, "feature:") {
		t.Errorf("expected plain's reserved line and no feature line, got:\n%s", stdout)
	}
	stdout, _, err = runOrgCmdStreams(t, "status", "--org-id", "plain", "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status plain --json: %v", err)
	}
	if keys := payloadKeys(orgStatusPayload(t, stdout)); !slices.Equal(keys, []string{"corrupt_lines", "reservation", "seats"}) {
		t.Errorf("plain's status --json keys = %v, want [corrupt_lines reservation seats]", keys)
	}
}

// TestOrgStatus_IncompleteFeatureBinding: a binding read from a damaged
// reservation record (a key missing, a key given twice, no worktree) shows
// the fields read on the feature line, which ends with (incomplete record),
// and in the JSON feature object, which adds "incomplete": true.
func TestOrgStatus_IncompleteFeatureBinding(t *testing.T) {
	setupOrgStubPATH(t)
	const wt = "/repo/.claude/worktrees/org-x"
	for _, tc := range []struct {
		name, details, worktree string
		wantLine                string
		wantJSON                map[string]any
	}{
		{"no digest", "paths=docs/b.md split=demo feature=b branch=fix/b", wt,
			"feature: demo/b branch fix/b worktree " + wt + " (incomplete record)",
			map[string]any{"split": "demo", "feature": "b", "digest": "", "branch": "fix/b", "worktree": wt, "incomplete": true}},
		{"split twice", "paths=docs/b.md split=demo split=other feature=b digest=0123456789ab branch=fix/b", wt,
			"feature: /b branch fix/b worktree " + wt + " (incomplete record)",
			map[string]any{"split": "", "feature": "b", "digest": "0123456789ab", "branch": "fix/b", "worktree": wt, "incomplete": true}},
		{"no worktree", "paths=docs/b.md split=demo feature=b digest=0123456789ab branch=fix/b", "",
			"feature: demo/b branch fix/b worktree  (incomplete record)",
			map[string]any{"split": "demo", "feature": "b", "digest": "0123456789ab", "branch": "fix/b", "worktree": "", "incomplete": true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := filepath.Join(t.TempDir(), "state")
			appendSeedEvent(t, stateDir, org.ManifestEvent{OrgID: "x", Event: org.EventScopeReserved, Worktree: tc.worktree, Details: tc.details})

			stdout, _, err := runOrgCmdStreams(t, "status", "--org-id", "x", "--state-dir", stateDir)
			if err != nil {
				t.Fatalf("status x: %v", err)
			}
			if got, want := outputLines(stdout), []string{"no seats", "reserved: docs/b.md", tc.wantLine}; !slices.Equal(got, want) {
				t.Errorf("status = %q, want %q", got, want)
			}
			stdout, _, err = runOrgCmdStreams(t, "status", "--org-id", "x", "--state-dir", stateDir, "--json")
			if err != nil {
				t.Fatalf("status x --json: %v", err)
			}
			var feature map[string]any
			if err := json.Unmarshal(orgStatusPayload(t, stdout)["feature"], &feature); err != nil {
				t.Fatalf("status --json feature: %v\n%s", err, stdout)
			}
			if !maps.Equal(feature, tc.wantJSON) {
				t.Errorf("feature = %v, want %v", feature, tc.wantJSON)
			}
		})
	}
}

// latestSeatEventNamed returns the latest manifest event named name of
// orgID/seatID under stateDir.
func latestSeatEventNamed(t *testing.T, stateDir, orgID, seatID, name string) org.ManifestEvent {
	t.Helper()
	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	for i := len(events) - 1; i >= 0; i-- {
		if ev := events[i]; ev.OrgID == orgID && ev.SeatID == seatID && ev.Event == name {
			return ev
		}
	}
	t.Fatalf("no %s event for %s/%s", name, orgID, seatID)
	return org.ManifestEvent{}
}
