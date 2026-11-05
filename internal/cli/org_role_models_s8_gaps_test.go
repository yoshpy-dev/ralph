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

// Gaps the /test run of cycle 2 of plan 2026-11-05-org-role-models found in
// the CLI tests of S8 (cross-review cycle 1's fix), next to
// org_role_models_retry_test.go and org_role_models_codex_pool_test.go:
//
//   - start --plan with a --model that is not one word: a new leader is
//     refused before anything is made (only org.StartFeature's own test had
//     it), and a leader already started is returned by the same request.
//   - start --plan and director start with --driver and --model for a driver
//     outside driver_pool: the refusal takes the plain wording, which names
//     no [org.role_models] entry. The other CLI tests of these two verbs give
//     only refusals of a driver taken from the role's entry, so a call site
//     that always passed DriverFromRoleEntry = true went unseen.
//   - director start with --model only: the driver comes from the entry, so
//     the refusal names the entry, and with a --model that is not one word
//     director start refuses before it writes or calls anything.

// TestOrgStartPlan_RoleModels_NewLeaderRefusalsAndARetryWithABadModel: with
// the leader's entry on codex, start --plan for a new leader is refused
// before anything is made (no feature worktree, branch, or ralph-worktree.sh
// record, no manifest event, no herdr or agmsg call) for --driver codex
// --model with driver_pool = ["claude"], in the plain wording that names no
// entry, and for a --model that is not one word with codex in driver_pool,
// naming the model. Once the leader is started, the same start --plan with
// a --model that is not one word returns it, with driver_pool narrowed or
// not, and records and calls nothing.
func TestOrgStartPlan_RoleModels_NewLeaderRefusalsAndARetryWithABadModel(t *testing.T) {
	const leaderOnCodex = "\n[org.permissions]\ndefault = \"guarded\"\n\n[org.role_models]\nleader = { driver = \"codex\", model = \"gpt-6.1-sol\" }\n"
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	root, sub := newFeatureRepoSub(t, ".claude/worktrees/\n.harness/\n")
	stateDir := filepath.Join(root, ".harness", "state", "org")
	plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
	wide := filepath.Join(root, ".harness", "wide.toml")
	writeTestFile(t, wide, "[org]\ndriver_pool = [\"claude\", \"codex\"]\n"+leaderOnCodex)
	narrowed := filepath.Join(root, ".harness", "narrowed.toml")
	writeTestFile(t, narrowed, "[org]\ndriver_pool = [\"claude\"]\n"+leaderOnCodex)
	t.Chdir(sub)
	startPlan := func(configPath string, flags []string) (stdout, stderr string, err error) {
		return runOrgCmdStreams(t, append([]string{"start", "--plan", plan, "--feature", "a", "--config", configPath}, flags...)...)
	}

	for _, tc := range []struct {
		name       string
		configPath string
		flags      []string
		want       string
		exact      bool
	}{
		{"--driver codex --model outside driver_pool", narrowed, retryCodexFlags, `org: driver "codex" not in [org].driver_pool [claude]`, true},
		{"--model only, not one word", wide, []string{"--model", "not a real model"}, `org: model "not a real model" must be ""`, false},
		{"--driver claude --model starting with '-'", wide, []string{"--driver", "claude", "--model=-x"}, `org: model "-x" must be ""`, false},
	} {
		t.Run("new leader: "+tc.name, func(t *testing.T) {
			stdout, _, err := startPlan(tc.configPath, tc.flags)
			switch {
			case err == nil:
				t.Fatalf("start --plan %v succeeded; want refused (stdout: %s)", tc.flags, stdout)
			case tc.exact && err.Error() != tc.want:
				t.Fatalf("start --plan %v = %q; want exactly %q", tc.flags, err, tc.want)
			case !tc.exact && !strings.Contains(err.Error(), tc.want):
				t.Fatalf("start --plan %v = %q; want it to contain %q", tc.flags, err, tc.want)
			}
			if strings.Contains(err.Error(), "[org.role_models]") {
				t.Errorf("start --plan %v = %q; a driver or a model from the flags must not name the entry", tc.flags, err)
			}
			assertNoFeatureStart(t, root, "a")
			if events := readManifestEvents(t, org.ManifestPathIn(stateDir)); len(events) != 0 {
				t.Errorf("manifest = %v; want nothing recorded", eventTypes(events))
			}
			if lines := slices.Concat(readLogLines(t, herdrLog), readLogLines(t, agmsgLog)); len(lines) != 0 {
				t.Errorf("herdr and agmsg calls = %v; want none", lines)
			}
		})
	}

	stdout, stderr, err := startPlan(wide, nil)
	if err != nil {
		t.Fatalf("start --plan with codex in driver_pool: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if want := `spawned seat "leader" (org_id=a driver=codex model=gpt-6.1-sol `; !strings.Contains(stdout, want) {
		t.Fatalf("stdout = %q; want it to contain %q", stdout, want)
	}
	before := takeOrgRetrySnapshot(t, stateDir, herdrLog, agmsgLog)
	for _, tc := range []struct {
		name       string
		configPath string
		flags      []string
	}{
		{"--model not one word, codex in driver_pool", wide, []string{"--model", "not a real model"}},
		{"--model not one word, driver_pool narrowed", narrowed, []string{"--model", "not a real model"}},
		{"--driver codex --model not one word, driver_pool narrowed", narrowed, []string{"--driver", "codex", "--model", "not a real model"}},
	} {
		t.Run("re-run: "+tc.name, func(t *testing.T) {
			stdout, stderr, err := startPlan(tc.configPath, tc.flags)
			if err != nil {
				t.Fatalf("start --plan again %v: %v (stdout: %s stderr: %s)", tc.flags, err, stdout, stderr)
			}
			if want := `seat "leader" already spawned (org_id=a driver=codex model=gpt-6.1-sol pane_id=`; !strings.Contains(stdout, want) {
				t.Errorf("stdout = %q; want it to contain %q", stdout, want)
			}
			if after := takeOrgRetrySnapshot(t, stateDir, herdrLog, agmsgLog); after != before {
				t.Errorf("after starting again %+v; want it as before %+v", after, before)
			}
		})
	}
}

// TestOrgDirectorStart_RoleModels_DriverPoolAndModelRefusalsThroughTheCLI:
// director start refuses with exit 1, nothing on stdout, director.jsonl as
// it was, no herdr call, and no prompt file, for --driver claude --model with
// driver_pool = ["codex"] (the plain wording, naming no entry), for --model
// only with the director's entry on claude outside that pool (the driver
// comes from the entry, so the refusal names it), and for a --model that is
// not one word, with or without --driver.
func TestOrgDirectorStart_RoleModels_DriverPoolAndModelRefusalsThroughTheCLI(t *testing.T) {
	const codexPool = "[org]\ndriver_pool = [\"codex\"]\n\n[org.permissions]\ndefault = \"guarded\"\n\n[org.role_models]\ndirector = { driver = \"claude\", model = \"\" }\n"
	for _, tc := range []struct {
		name     string
		config   string // "" for no --config
		flags    []string
		want     string
		exact    bool
		forEntry bool // whether the refusal names [org.role_models].director
	}{
		{"--driver claude --model outside driver_pool", codexPool, []string{"--driver", "claude", "--model", "sonnet"},
			`org: driver "claude" not in [org].driver_pool [codex]`, true, false},
		{"--model only, the entry's driver outside driver_pool", codexPool, []string{"--model", "sonnet"},
			`org: role "director" runs on driver "claude" in [org.role_models].director, which is not in [org].driver_pool [codex]: ` +
				"add claude to driver_pool, change the entry, or pass both --driver and --model", true, true},
		{"--model only, not one word", "", []string{"--model", "not a real model"}, `org: director: start: model "not a real model" must be ""`, false, false},
		{"--driver claude --model starting with '-'", "", []string{"--driver", "claude", "--model=-x"}, `org: director: start: model "-x" must be ""`, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir, herdrLog, _ := directorStartLedger(t)
			args := append([]string{"director", "start", "--state-dir", stateDir}, tc.flags...)
			if tc.config != "" {
				args = append(args, "--config", writeDirectorConfig(t, tc.config))
			}
			before := directorFileOrAbsent(t, stateDir)

			stdout, _, err := runOrgCmdStreams(t, args...)
			switch {
			case err == nil:
				t.Fatalf("director start %v succeeded; want refused (stdout: %s)", tc.flags, stdout)
			case tc.exact && err.Error() != tc.want:
				t.Fatalf("director start %v = %q; want exactly %q", tc.flags, err, tc.want)
			case !tc.exact && !strings.Contains(err.Error(), tc.want):
				t.Fatalf("director start %v = %q; want it to contain %q", tc.flags, err, tc.want)
			}
			if got := strings.Contains(err.Error(), "[org.role_models]"); got != tc.forEntry {
				t.Errorf("director start %v = %q; names [org.role_models]: %t, want %t", tc.flags, err, got, tc.forEntry)
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
}
