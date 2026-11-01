package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// TestOrgStartPlan_LeaderTaskCarriesTheReviewerSettings is AC4 of plan
// docs/plans/active/2026-10-30-standard-flow-config.md (D5) through the CLI:
// start --plan writes the [reviewer] values of the config it loads on the
// leader task's two reviewer lines, the model and then the effort before
// each line's parenthesis: the defaults with no ralph.toml, the values of
// ralph.toml, and, with the personal file read, the keys ralph.local.toml
// sets over them. ralph.toml is committed and ralph.local.toml ignored, so
// the main checkout stays clean for ralph-worktree.sh.
func TestOrgStartPlan_LeaderTaskCarriesTheReviewerSettings(t *testing.T) {
	const tomlReviewer = "[reviewer]\nclaude_model = \"opus\"\nclaude_effort = \"high\"\ncodex_model = \"gpt-6-sol\"\ncodex_effort = \"medium\"\n"
	def := config.Default().Reviewer
	cases := []struct {
		name, toml, local string // "" for either: no such file
		codex, claude     string // the "<model> <effort>" of each reviewer line
	}{
		{"no ralph.toml", "", "", def.CodexModel + " " + def.CodexEffort, def.ClaudeModel + " " + def.ClaudeEffort},
		{"ralph.toml's [reviewer]", tomlReviewer, "", "gpt-6-sol medium", "opus high"},
		{"ralph.local.toml over ralph.toml", tomlReviewer, "[reviewer]\nclaude_model = \"sonnet\"\ncodex_effort = \"low\"\n",
			"gpt-6-sol low", "sonnet high"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupOrgStubPATH(t)
			t.Setenv(config.LocalConfigEnv, "")
			root := newFeatureRepo(t, ".claude/worktrees/\n.harness/\n"+config.LocalConfigFileName+"\n")
			if tc.toml != "" {
				writeTestFile(t, filepath.Join(root, "ralph.toml"), tc.toml)
				runLedgerTestGit(t, "-C", root, "add", "ralph.toml")
				runLedgerTestGit(t, "-C", root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
					"commit", "--quiet", "-m", "ralph.toml")
			}
			if tc.local != "" {
				writeTestFile(t, filepath.Join(root, config.LocalConfigFileName), tc.local)
			}
			t.Chdir(root)
			stateDir := filepath.Join(root, ".harness", "state", "org")
			plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))

			stdout, stderr, err := startOrgFeature(t, plan, "b")
			if err != nil {
				t.Fatalf("start --plan --feature b: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			prompt, err := os.ReadFile(codexPromptPathFor(stateDir, "b", org.LeaderIdentity))
			if err != nil {
				t.Fatalf("read the leader's prompt: %v", err)
			}
			for _, want := range []string{"\n- reviewer の codex: " + tc.codex + "(", "\n- reviewer の claude: " + tc.claude + "("} {
				if !strings.Contains(string(prompt), want) {
					t.Errorf("expected the leader's task to have %q, got:\n%s", want, prompt)
				}
			}
		})
	}
}

// TestOrgStartPlan_ReviewerFromMainWorktree is D5 of plan
// docs/plans/active/2026-10-30-standard-flow-config.md through the CLI, by
// the rule TestOrgStartPlan_ApprovalModeFromMainWorktree checks for the
// approval mode: without --config and with the main worktree's ledger,
// start --plan writes the [reviewer] values of the main checkout's
// ralph.toml in the leader's task, also when it runs from a subdirectory or
// a linked worktree whose own ralph.toml says others, and a main checkout
// ralph.toml without [reviewer] gives the defaults. With --config, the
// --config file's values are written. The main worktree root's
// ralph.local.toml applies on top either way; the --config file lies in the
// repository's ignored .harness/, so the personal file Load reads for it is
// that root's. The ralph.toml files are committed and ralph.local.toml
// ignored, so the main checkout stays clean for ralph-worktree.sh.
func TestOrgStartPlan_ReviewerFromMainWorktree(t *testing.T) {
	reviewerToml := func(claudeModel, claudeEffort, codexModel, codexEffort string) string {
		return "[reviewer]\nclaude_model = \"" + claudeModel + "\"\nclaude_effort = \"" + claudeEffort + "\"\n" +
			"codex_model = \"" + codexModel + "\"\ncodex_effort = \"" + codexEffort + "\"\n"
	}
	mainToml := reviewerToml("opus", "high", "gpt-6-sol", "medium")
	hereToml := reviewerToml("sonnet", "low", "gpt-6-here", "low")
	cfgToml := reviewerToml("haiku", "max", "gpt-6-cfg", "high")
	const localToml = "[reviewer]\nclaude_effort = \"medium\"\ncodex_model = \"gpt-6-local\"\n"
	def := config.Default().Reviewer
	cases := []struct {
		name, from       string // from: "sub" (a subdirectory of the main checkout) or "linked" (a linked worktree)
		main, cfg, local string // cfg: "" for no --config; local: "" for no ralph.local.toml
		codex, claude    string // the "<model> <effort>" of each reviewer line
	}{
		{"subdirectory, the main checkout's [reviewer]", "sub", mainToml, "", "", "gpt-6-sol medium", "opus high"},
		{"linked worktree, the main checkout's [reviewer]", "linked", mainToml, "", "", "gpt-6-sol medium", "opus high"},
		{"subdirectory, main checkout without [reviewer]", "sub", "[org]\nmax_seats = 5\n", "", "",
			def.CodexModel + " " + def.CodexEffort, def.ClaudeModel + " " + def.ClaudeEffort},
		{"subdirectory, ralph.local.toml over the main checkout's", "sub", mainToml, "", localToml, "gpt-6-local medium", "opus medium"},
		{"--config, its [reviewer]", "sub", mainToml, cfgToml, "", "gpt-6-cfg high", "haiku max"},
		{"--config, ralph.local.toml over it", "sub", mainToml, cfgToml, localToml, "gpt-6-local high", "haiku medium"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupOrgStubPATH(t)
			t.Setenv(config.LocalConfigEnv, "")
			root := newFeatureRepo(t, ".claude/worktrees/\n.harness/\n"+config.LocalConfigFileName+"\n")
			sub := filepath.Join(root, "pkg", "sub")
			if err := os.MkdirAll(sub, 0o755); err != nil {
				t.Fatal(err)
			}
			writeTestFile(t, filepath.Join(root, "ralph.toml"), tc.main)
			writeTestFile(t, filepath.Join(sub, "ralph.toml"), hereToml)
			runLedgerTestGit(t, "-C", root, "add", ".")
			runLedgerTestGit(t, "-C", root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
				"commit", "--quiet", "-m", "ralph.toml")
			dir := sub
			if tc.from == "linked" {
				dir = filepath.Join(root, ".claude", "worktrees", "task")
				runLedgerTestGit(t, "-C", root, "worktree", "add", "--quiet", dir, "-b", "task")
				writeTestFile(t, filepath.Join(dir, "ralph.toml"), hereToml)
				runLedgerTestGit(t, "-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
					"commit", "--quiet", "-am", "the task's ralph.toml")
			}
			if tc.local != "" {
				writeTestFile(t, filepath.Join(root, config.LocalConfigFileName), tc.local)
			}
			stateDir := filepath.Join(root, ".harness", "state", "org")
			plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
			var extra []string
			if tc.cfg != "" {
				path := filepath.Join(root, ".harness", "custom.toml")
				writeTestFile(t, path, tc.cfg)
				extra = []string{"--config", path}
			}
			t.Chdir(dir)

			stdout, stderr, err := startOrgFeature(t, plan, "b", extra...)
			if err != nil {
				t.Fatalf("start --plan --feature b: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			prompt, err := os.ReadFile(codexPromptPathFor(stateDir, "b", org.LeaderIdentity))
			if err != nil {
				t.Fatalf("read the leader's prompt: %v", err)
			}
			for _, want := range []string{"\n- reviewer の codex: " + tc.codex + "(", "\n- reviewer の claude: " + tc.claude + "("} {
				if !strings.Contains(string(prompt), want) {
					t.Errorf("expected the leader's task to have %q, got:\n%s", want, prompt)
				}
			}
		})
	}
}

// TestOrgStartHelp_ReviewerSource: `ralph org start --help` says that the
// [reviewer] values the leader's task carries come from the ralph.toml the
// approval mode and the org-wide limits come from, and that the leader does
// not read them itself (plan 2026-10-30-standard-flow-config, D5). The
// sentence that named the --config file or the ./ralph.toml of the directory
// start runs in, which stopped being true for a start without --config on
// the main worktree's ledger, is gone.
func TestOrgStartHelp_ReviewerSource(t *testing.T) {
	out, err := runOrgCmd(t, "start", "--help")
	if err != nil {
		t.Fatalf("start --help: %v", err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"The four keys of [reviewer], the model and effort of the reviewers the leader's plan advisory calls, are read from " +
			"the same ralph.toml as max_orgs and max_total_seats below",
		"and written in the leader's task as two lines, one for codex and one for claude: the leader uses them as they " +
			"were when the org started and does not read them itself.",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("start --help does not say %q:\n%s", want, out)
		}
	}
	if stale := "the ./ralph.toml of the directory start runs in"; strings.Contains(help, stale) {
		t.Errorf("start --help still says %q:\n%s", stale, out)
	}
}
