package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// Tests for `ralph config get` (plan 2026-10-29-review-switches-config, D2,
// D3, D6 and AC4). Those that run the verb on a key first chdir into a temp
// directory with t.Chdir, so that ./ralph.toml is the test's own. t.Chdir
// cannot be called from a parallel test, so none of these tests runs in
// parallel.

// runConfigGet runs `ralph config get <key>` with extra flags in-process and
// returns its stdout, its stderr, and the exit code main would use.
func runConfigGet(t *testing.T, key string, flags ...string) (stdout, stderr string, code int) {
	t.Helper()
	args := append([]string{"config", "get"}, flags...)
	stdout, stderr, err := runRootCmdSplit(t, append(args, key)...)
	return stdout, stderr, ExitCode(err)
}

// With no ralph.toml, every key prints its built-in default: the review
// switches are off and the cross-review cap is 2. An integer and a string
// key from outside this plan's tables print config.Default()'s values. The
// pipeline cap and the [reviewer] keys print their defaults too (plan
// 2026-10-30-standard-flow-config, AC1), and so do the five local-only keys
// the scripts read, an empty line or false (plan
// 2026-10-30-script-local-config, AC1).
func TestConfigGet_NoRalphTomlPrintsTheDefaults(t *testing.T) {
	t.Chdir(t.TempDir())
	defaults := config.Default()
	for _, tc := range []struct{ key, want string }{
		{"standard.plan_advisory", "false\n"},
		{"standard.cross_review", "false\n"},
		{"org.review.plan_advisory", "false\n"},
		{"org.review.cross_review", "false\n"},
		{"org.review.max_cross_review_cycles", "2\n"},
		{"org.max_seats", strconv.Itoa(defaults.Org.MaxSeats) + "\n"},
		{"org.approval.mode", defaults.Org.Approval.Mode + "\n"},
		{"org.gate.regate_checks", "local\n"},
		{"standard.max_pipeline_cycles", "2\n"},
		{"reviewer.claude_model", "fable\n"},
		{"reviewer.claude_effort", "xhigh\n"},
		{"reviewer.codex_model", "gpt-6-astra\n"},
		{"reviewer.codex_effort", "xhigh\n"},
		{"plan_visual.browser", "\n"},
		{"plan_visual.opener", "\n"},
		{"standard.base_branch", "\n"},
		{"verify.base", "\n"},
		{"verify.skip_branch_secret_scan", "false\n"},
	} {
		stdout, stderr, code := runConfigGet(t, tc.key)
		if code != 0 || stdout != tc.want || stderr != "" {
			t.Errorf("config get %s = (stdout %q, stderr %q, exit %d); want (%q, \"\", 0)", tc.key, stdout, stderr, code, tc.want)
		}
	}
}

// A key ralph.toml sets prints that value; a key it does not set still prints
// the built-in default.
func TestConfigGet_PrintsTheValueRalphTomlSets(t *testing.T) {
	t.Chdir(t.TempDir())
	writeTestFile(t, "ralph.toml", "[standard]\nplan_advisory = true\n\n[org.review]\nmax_cross_review_cycles = 5\n\n[org.approval]\nmode = \"split\"\n\n[org.gate]\nregate_checks = \"ci\"\n")
	for _, tc := range []struct{ key, want string }{
		{"standard.plan_advisory", "true\n"},
		{"org.review.max_cross_review_cycles", "5\n"},
		{"org.approval.mode", "split\n"},
		{"org.gate.regate_checks", "ci\n"},
		{"standard.cross_review", "false\n"},
		{"org.review.cross_review", "false\n"},
	} {
		stdout, stderr, code := runConfigGet(t, tc.key)
		if code != 0 || stdout != tc.want || stderr != "" {
			t.Errorf("config get %s = (stdout %q, stderr %q, exit %d); want (%q, \"\", 0)", tc.key, stdout, stderr, code, tc.want)
		}
	}
}

// The pipeline cap and the [reviewer] keys ralph.toml sets print their values,
// a personal ralph.local.toml beside it overrides them key by key, and a
// [reviewer] key neither file sets prints its default (plan
// 2026-10-30-standard-flow-config, AC1). The directory is outside git, so the
// personal file is the one beside ralph.toml.
func TestConfigGet_PrintsThePipelineCapAndReviewerValues(t *testing.T) {
	t.Setenv(config.LocalConfigEnv, "")
	t.Chdir(t.TempDir())
	writeTestFile(t, "ralph.toml", "[standard]\nmax_pipeline_cycles = 3\n\n[reviewer]\nclaude_model = \"opus\"\nclaude_effort = \"high\"\ncodex_model = \"gpt-5.6-sol\"\n")
	writeTestFile(t, config.LocalConfigFileName, "[standard]\nmax_pipeline_cycles = 4\n\n[reviewer]\nclaude_effort = \"max\"\n")
	for _, tc := range []struct{ key, want string }{
		{"standard.max_pipeline_cycles", "4\n"},
		{"reviewer.claude_model", "opus\n"},
		{"reviewer.claude_effort", "max\n"},
		{"reviewer.codex_model", "gpt-5.6-sol\n"},
		{"reviewer.codex_effort", "xhigh\n"},
	} {
		stdout, stderr, code := runConfigGet(t, tc.key)
		if code != 0 || stdout != tc.want || stderr != "" {
			t.Errorf("config get %s = (stdout %q, stderr %q, exit %d); want (%q, \"\", 0)", tc.key, stdout, stderr, code, tc.want)
		}
	}
}

// A [reviewer] value or a pipeline cap that fails validation exits 3 with
// nothing on stdout, also when the key asked for is another one, and the
// error names the key at fault (plan 2026-10-30-standard-flow-config, AC1).
func TestConfigGet_InvalidPipelineCapOrReviewerValueExits3(t *testing.T) {
	for _, tc := range []struct{ name, content, key, want string }{
		{"a cap of 0", "[standard]\nmax_pipeline_cycles = 0\n", "standard.max_pipeline_cycles", "[standard].max_pipeline_cycles must be >= 1, got 0"},
		{"an unknown claude effort", "[reviewer]\nclaude_effort = \"ultra\"\n", "reviewer.claude_effort", `[reviewer].claude_effort "ultra" must be one of`},
		{"a codex model with a space", "[reviewer]\ncodex_model = \"gpt 6\"\n", "reviewer.codex_model", `[reviewer].codex_model "gpt 6" must be non-empty`},
		{"an empty claude model", "[reviewer]\nclaude_model = \"\"\n", "standard.cross_review", `[reviewer].claude_model "" must be non-empty`},
		{"a codex effort with a $", "[reviewer]\ncodex_effort = \"x$y\"\n", "reviewer.codex_model", `[reviewer].codex_effort "x$y" must be non-empty`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			writeTestFile(t, "ralph.toml", tc.content)
			stdout, _, err := runRootCmdSplit(t, "config", "get", tc.key)
			if code := ExitCode(err); code != 3 || stdout != "" {
				t.Fatalf("config get %s = (stdout %q, exit %d, err %v); want (\"\", 3)", tc.key, stdout, code, err)
			}
			if !errors.Is(err, ErrConfigLoad) || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v; want ErrConfigLoad containing %q", err, tc.want)
			}
		})
	}
}

// --config reads the file it names instead of ./ralph.toml, given after get
// or before it on the config group.
func TestConfigGet_ConfigFlagReadsThatFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeTestFile(t, "ralph.toml", "[standard]\ncross_review = false\n")
	other := filepath.Join(dir, "other.toml")
	writeTestFile(t, other, "[standard]\ncross_review = true\n")

	stdout, stderr, code := runConfigGet(t, "standard.cross_review", "--config", other)
	if code != 0 || stdout != "true\n" || stderr != "" {
		t.Errorf("config get --config %s standard.cross_review = (stdout %q, stderr %q, exit %d); want (\"true\\n\", \"\", 0)", other, stdout, stderr, code)
	}
	stdout, stderr, err := runRootCmdSplit(t, "config", "--config", other, "get", "standard.cross_review")
	if code := ExitCode(err); code != 0 || stdout != "true\n" || stderr != "" {
		t.Errorf("config --config %s get standard.cross_review = (stdout %q, stderr %q, exit %d); want (\"true\\n\", \"\", 0)", other, stdout, stderr, code)
	}
	stdout, _, code = runConfigGet(t, "standard.cross_review")
	if code != 0 || stdout != "false\n" {
		t.Errorf("config get standard.cross_review without --config = (stdout %q, exit %d); want (\"false\\n\", 0)", stdout, code)
	}
}

// A key that is not a key of one value in ralph.toml exits 2 and prints
// nothing on stdout: an unknown key, a table, an array, a map, a key under a
// map or under a value, and an empty or misspelled one.
func TestConfigGet_KeyThatIsNotOneValueExits2(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, tc := range []struct{ key, want string }{
		{"nope.key", "not a key of one value in ralph.toml: no such key"},
		{"standard.nope", "not a key of one value in ralph.toml: no such key"},
		{"Standard.plan_advisory", "not a key of one value in ralph.toml: no such key"},
		{"", "not a key of one value in ralph.toml: no such key"},
		{"standard.", "not a key of one value in ralph.toml: no such key"},
		{"standard", "it is a table"},
		{"plan_visual", "it is a table"},
		{"verify", "it is a table"},
		{"org.review", "it is a table"},
		{"org.roles", "it is a table"},
		{"org.driver_pool", "it is an array"},
		{"org.model_pool", "it is an array"},
		{"org.roles.leader", "org.roles is a table"},
		{"org.driver_pool.claude", "org.driver_pool is an array"},
		{"standard.plan_advisory.x", "standard.plan_advisory is a boolean"},
		{"org.max_seats.x", "org.max_seats is an integer"},
		{"org.approval.mode.x", "org.approval.mode is a string"},
	} {
		stdout, _, err := runRootCmdSplit(t, "config", "get", tc.key)
		if code := ExitCode(err); code != 2 || stdout != "" {
			t.Errorf("config get %q = (stdout %q, exit %d, err %v); want (\"\", 2)", tc.key, stdout, code, err)
			continue
		}
		if !errors.Is(err, ErrConfigKey) || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("config get %q: err = %v; want ErrConfigKey naming %q", tc.key, err, tc.want)
		}
	}
}

// A ralph.toml that cannot be parsed, that fails validation, that cannot be
// read, or that is a directory exits 3 and prints nothing on stdout, also
// when --config names it.
func TestConfigGet_RalphTomlThatCannotBeReadOrIsInvalidExits3(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T) []string
	}{
		{"unparsable", func(t *testing.T) []string {
			writeTestFile(t, "ralph.toml", "[standard\nplan_advisory = true\n")
			return nil
		}},
		{"fails validation", func(t *testing.T) []string {
			writeTestFile(t, "ralph.toml", "[org.review]\nmax_cross_review_cycles = 0\n")
			return nil
		}},
		{"a directory", func(t *testing.T) []string {
			if err := os.Mkdir("ralph.toml", 0o755); err != nil {
				t.Fatal(err)
			}
			return nil
		}},
		{"unreadable", func(t *testing.T) []string {
			if os.Geteuid() == 0 {
				t.Skip("root ignores the permission bits")
			}
			writeRalphTomlBehindLockedDir(t, "[standard]\nplan_advisory = false\n")
			return nil
		}},
		{"unparsable --config", func(t *testing.T) []string {
			path := filepath.Join(t.TempDir(), "other.toml")
			writeTestFile(t, path, "plan_advisory = = true\n")
			return []string{"--config", path}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			flags := tc.setup(t)
			stdout, _, err := runRootCmdSplit(t, append(append([]string{"config", "get"}, flags...), "standard.plan_advisory")...)
			if code := ExitCode(err); code != 3 || stdout != "" {
				t.Fatalf("config get standard.plan_advisory = (stdout %q, exit %d, err %v); want (\"\", 3)", stdout, code, err)
			}
			if !errors.Is(err, ErrConfigLoad) {
				t.Errorf("err = %v; want ErrConfigLoad", err)
			}
		})
	}
}

// Regression for the Codex plan advisory (D6, AC4): a ralph.toml that turns
// the cross-review on but holds an invalid [org] setting exits 3, so a skill
// stops and reports it instead of reading the default false, or an exit 1 it
// would take for an older ralph, as off.
func TestConfigGet_InvalidOrgSettingDoesNotTurnCrossReviewOff(t *testing.T) {
	t.Chdir(t.TempDir())
	writeTestFile(t, "ralph.toml", "[standard]\ncross_review = true\n\n[org]\nmax_seats = 0\n")
	stdout, _, err := runRootCmdSplit(t, "config", "get", "standard.cross_review")
	if code := ExitCode(err); code != 3 || stdout != "" {
		t.Fatalf("config get standard.cross_review = (stdout %q, exit %d, err %v); want (\"\", 3)", stdout, code, err)
	}
	if !strings.Contains(err.Error(), "[org].max_seats") {
		t.Errorf("err = %v; want it to name [org].max_seats", err)
	}
}

// cobra's own errors stay exit 1, and none of them says unknown command
// "config", the one exit 1 a skill treats as an older ralph. A mistyped verb
// is an error, not the group's help with exit 0.
func TestConfigGet_CommandLineErrorsExit1(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{
		{"config", "get"},
		{"config", "get", "standard.plan_advisory", "standard.cross_review"},
		{"config", "gte", "standard.plan_advisory"},
		{"config", "get", "--nope", "standard.plan_advisory"},
	} {
		stdout, _, err := runRootCmdSplit(t, args...)
		if code := ExitCode(err); code != 1 || stdout != "" {
			t.Errorf("ralph %s = (stdout %q, exit %d, err %v); want (\"\", 1)", strings.Join(args, " "), stdout, code, err)
			continue
		}
		if strings.Contains(err.Error(), `unknown command "config"`) {
			t.Errorf("ralph %s: err = %v; must not read as an older ralph", strings.Join(args, " "), err)
		}
	}
}

// The help of get states the contract of D6 for the skills and seats: exit
// 0 is the value, exit 1 with unknown command "config" (and only that) is
// off, and exits 2 and 3 stop. It also says that ralph_setting, which reads
// the standard flow's values (plan 2026-10-30-standard-flow-config, D2),
// goes on with the default instead, after one line on stderr for exits 2
// and 3, and that the scripts read their local-only keys with the same
// helper, the two bases with ralph_setting_or_fail, which fails instead
// of giving the default for those exits (plan 2026-10-30-script-local-config,
// D7). Each phrase is matched with runs of white space, line breaks
// included, read as one space, so re-wrapping the help keeps the test green.
func TestConfigGet_HelpStatesTheExitCodeContract(t *testing.T) {
	stdout, _, err := runRootCmdSplit(t, "config", "get", "--help")
	if err != nil {
		t.Fatalf("config get --help: %v", err)
	}
	help := strings.Join(strings.Fields(stdout), " ")
	for _, want := range []string{
		"A key the file does not set prints its built-in default",
		"0 the value was printed: use it.",
		"unknown command \"config\" on stderr. Treat that, and only that, as off, and record that the setting was treated as off.",
		"2 the key is not a key of ralph.toml, or names a table or an array rather than one value: stop and report it, do not treat it as off.",
		"3 ralph.toml cannot be read, cannot be parsed, or fails validation: stop and report it, do not treat it as off.",
		"These are for a skill or seat that runs ralph config get itself, as for the review switches of [standard] and [org.review].",
		"The skills read standard.max_pipeline_cycles and the four keys of [reviewer] with ralph_setting of " +
			"scripts/ralph-config.sh instead, which does not stop: when it gets no value it goes on with the key's built-in default, " +
			"silently when ralph is not on PATH or is an older ralph without this verb, and after one line on stderr otherwise, " +
			"exits 2 and 3 included.",
		"The scripts plan-visual.sh, xreview-helpers.sh, detect-changed-languages.sh, and run-verify.sh read their keys with the same helper: " +
			"ralph_setting for plan_visual.browser, plan_visual.opener, and verify.skip_branch_secret_scan, " +
			"and ralph_setting_or_fail for standard.base_branch and verify.base.",
		"ralph_setting_or_fail gives the default only when ralph is not on PATH or is an older ralph without this verb. " +
			"On any other failure, exits 2 and 3 included, it writes one line on stderr, prints nothing, and fails, " +
			"so the script stops or takes the safe side (the full verify scope) instead of using the default base.",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("config get --help lacks %q:\n%s", want, stdout)
		}
	}
}

// AC2 of plan 2026-10-29-local-config-org, and the regression test of its
// Codex plan advisory 1: `ralph config get org.agmsg_home` prints the value
// of the personal file at the main worktree's root when run from that root,
// from a subdirectory with no ralph.toml, and from a linked worktree's root
// and subdirectory. The main root's ralph.toml sets another value, and the
// same-named personal files in the linked worktree and the subdirectories
// are not read.
func TestConfigGet_ReadsTheMainWorktreesLocalFileFromAnyWorktree(t *testing.T) {
	t.Setenv(config.LocalConfigEnv, "")
	isolateFeatureGitEnv(t)
	tmp := t.TempDir()
	main, linked := filepath.Join(tmp, "main"), filepath.Join(tmp, "linked")
	runLedgerTestGit(t, "init", "--quiet", main)
	runLedgerTestGit(t, "-C", main, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
		"commit", "--quiet", "--allow-empty", "-m", "init")
	runLedgerTestGit(t, "-C", main, "worktree", "add", "--quiet", linked, "-b", "linked")
	want := filepath.Join(tmp, "agmsg-of-the-main-root")
	writeTestFile(t, filepath.Join(main, config.LocalConfigFileName), "[org]\nagmsg_home = \""+want+"\"\n")
	writeTestFile(t, filepath.Join(main, "ralph.toml"), "[org]\nagmsg_home = \"/agmsg-of-ralph-toml\"\n")

	for _, dir := range []string{
		main,
		filepath.Join(main, "pkg", "sub"),
		linked,
		filepath.Join(linked, "pkg", "sub"),
	} {
		if dir != main {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			writeTestFile(t, filepath.Join(dir, config.LocalConfigFileName), "[org]\nagmsg_home = \"/agmsg-of-"+filepath.Base(dir)+"\"\n")
		}
		t.Chdir(dir)
		stdout, stderr, code := runConfigGet(t, "org.agmsg_home")
		if code != 0 || stdout != want+"\n" || stderr != "" {
			t.Errorf("in %s: config get org.agmsg_home = (stdout %q, stderr %q, exit %d); want (%q, \"\", 0)", dir, stdout, stderr, code, want+"\n")
		}
	}
}

// AC3 of plan 2026-10-29-local-config-org: a personal file that is a
// directory, is not valid TOML, fails validation, or sets a relative
// [org].agmsg_home exits 3 like ralph.toml, with its path in the error, also
// when no ralph.toml exists.
func TestConfigGet_LocalFileThatCannotBeReadOrIsInvalidExits3(t *testing.T) {
	t.Setenv(config.LocalConfigEnv, "")
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T)
	}{
		{"a directory", func(t *testing.T) {
			if err := os.Mkdir(config.LocalConfigFileName, 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"unparsable", func(t *testing.T) {
			writeTestFile(t, config.LocalConfigFileName, "[org\nmax_seats = 2\n")
		}},
		{"fails validation", func(t *testing.T) {
			writeTestFile(t, config.LocalConfigFileName, "[org]\nmax_seats = 0\n")
		}},
		{"relative agmsg_home", func(t *testing.T) {
			writeTestFile(t, config.LocalConfigFileName, "[org]\nagmsg_home = \"agmsg\"\n")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			tc.setup(t)
			stdout, _, err := runRootCmdSplit(t, "config", "get", "standard.plan_advisory")
			if code := ExitCode(err); code != 3 || stdout != "" {
				t.Fatalf("config get standard.plan_advisory = (stdout %q, exit %d, err %v); want (\"\", 3)", stdout, code, err)
			}
			if !errors.Is(err, ErrConfigLoad) || !strings.Contains(err.Error(), config.LocalConfigFileName) {
				t.Errorf("err = %v; want ErrConfigLoad naming %s", err, config.LocalConfigFileName)
			}
		})
	}
}

// `ralph config get org.state_dir` prints the personal file's value as
// written: not cleaned, not expanded, and a relative one too, since Load does
// not check it (plan 2026-10-30-org-state-dir-local, D2). With no personal
// file it prints the default, an empty line.
func TestConfigGet_OrgStateDirPrintsThePersonalFilesValueAsWritten(t *testing.T) {
	t.Setenv(config.LocalConfigEnv, "")
	for _, value := range []string{"/abs/../ledger", "~/ledger", "rel/ledger", ""} {
		t.Chdir(t.TempDir())
		if value != "" {
			writeTestFile(t, config.LocalConfigFileName, "[org]\nstate_dir = \""+value+"\"\n")
		}
		stdout, stderr, code := runConfigGet(t, "org.state_dir")
		if code != 0 || stdout != value+"\n" || stderr != "" {
			t.Errorf("config get org.state_dir with state_dir %q = (stdout %q, stderr %q, exit %d); want (%q, \"\", 0)", value, stdout, stderr, code, value+"\n")
		}
	}
}

// AC4 of plan 2026-10-30-org-state-dir-local: a ralph.toml that sets
// [org].state_dir exits 3, naming the file and saying to set the key in
// ralph.local.toml, also with the personal file off (TestMain's setting).
func TestConfigGet_RalphTomlThatSetsALocalOnlyKeyExits3(t *testing.T) {
	t.Chdir(t.TempDir())
	writeTestFile(t, "ralph.toml", "[org]\nstate_dir = \"/ledger\"\n")
	stdout, _, err := runRootCmdSplit(t, "config", "get", "standard.plan_advisory")
	if code := ExitCode(err); code != 3 || stdout != "" {
		t.Fatalf("config get standard.plan_advisory = (stdout %q, exit %d, err %v); want (\"\", 3)", stdout, code, err)
	}
	if want := "ralph.toml: [org].state_dir can be set only in ralph.local.toml"; !errors.Is(err, ErrConfigLoad) || !strings.Contains(err.Error(), want) {
		t.Errorf("err = %v; want ErrConfigLoad containing %q", err, want)
	}
}

// AC1 of plan 2026-10-30-script-local-config: the five local-only keys the
// scripts read print the values the personal file gives them, an absolute
// browser path with spaces and a revision such as HEAD~1 as written, with
// exit 0. With RALPH_LOCAL_CONFIG=off, which `ralph org gate` gives its
// scripts, the same file is not read and each key prints its default (AC4):
// no personal [verify].base or skip_branch_secret_scan reaches the gate.
func TestConfigGet_PrintsTheScriptLocalOnlyValues(t *testing.T) {
	t.Setenv(config.LocalConfigEnv, "")
	t.Chdir(t.TempDir())
	const chrome = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	writeTestFile(t, config.LocalConfigFileName, "[plan_visual]\nbrowser = \""+chrome+"\"\nopener = \"xdg-open\"\n\n"+
		"[standard]\nbase_branch = \"release/1.2\"\n\n[verify]\nbase = \"HEAD~1\"\nskip_branch_secret_scan = true\n")
	cases := []struct{ key, want, def string }{
		{"plan_visual.browser", chrome + "\n", "\n"},
		{"plan_visual.opener", "xdg-open\n", "\n"},
		{"standard.base_branch", "release/1.2\n", "\n"},
		{"verify.base", "HEAD~1\n", "\n"},
		{"verify.skip_branch_secret_scan", "true\n", "false\n"},
	}
	for _, tc := range cases {
		stdout, stderr, code := runConfigGet(t, tc.key)
		if code != 0 || stdout != tc.want || stderr != "" {
			t.Errorf("config get %s = (stdout %q, stderr %q, exit %d); want (%q, \"\", 0)", tc.key, stdout, stderr, code, tc.want)
		}
	}

	t.Setenv(config.LocalConfigEnv, "off")
	for _, tc := range cases {
		stdout, stderr, code := runConfigGet(t, tc.key)
		if code != 0 || stdout != tc.def || stderr != "" {
			t.Errorf("with %s=off, config get %s = (stdout %q, stderr %q, exit %d); want the default (%q, \"\", 0)",
				config.LocalConfigEnv, tc.key, stdout, stderr, code, tc.def)
		}
	}
}

// AC1 and D7 of plan 2026-10-30-script-local-config: a value of the personal
// file that D3 refuses exits 3 with nothing on stdout, and the error names
// the key, the value, and ralph.local.toml. Load validates the whole config,
// so asking for another key exits 3 as well: a wrong browser makes
// `ralph config get verify.base` fail, not print the default.
func TestConfigGet_InvalidScriptLocalOnlyValueExits3(t *testing.T) {
	t.Setenv(config.LocalConfigEnv, "")
	for _, tc := range []struct{ name, content, key, want string }{
		{"a browser under ~/", "[plan_visual]\nbrowser = \"~/chrome\"\n", "plan_visual.browser", `[plan_visual].browser "~/chrome" must be`},
		{"a relative browser, asking for verify.base", "[plan_visual]\nbrowser = \"chrome\"\n", "verify.base", `[plan_visual].browser "chrome" must be`},
		{"an opener with an option", "[plan_visual]\nopener = \"open -a Safari\"\n", "plan_visual.opener", `[plan_visual].opener "open -a Safari" must be`},
		{"a base branch with ..", "[standard]\nbase_branch = \"a..b\"\n", "standard.base_branch", `[standard].base_branch "a..b" must be`},
		{"a base branch that starts with -", "[standard]\nbase_branch = \"-main\"\n", "standard.cross_review", `[standard].base_branch "-main" must be`},
		{"a verify base that starts with -", "[verify]\nbase = \"--all\"\n", "verify.base", `[verify].base "--all" must be`},
		{"a verify base with a line break", "[verify]\nbase = \"HEAD\\n\"\n", "verify.skip_branch_secret_scan", `[verify].base "HEAD\n" must be`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			writeTestFile(t, config.LocalConfigFileName, tc.content)
			stdout, _, err := runRootCmdSplit(t, "config", "get", tc.key)
			if code := ExitCode(err); code != 3 || stdout != "" {
				t.Fatalf("config get %s = (stdout %q, exit %d, err %v); want (\"\", 3)", tc.key, stdout, code, err)
			}
			if !errors.Is(err, ErrConfigLoad) || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), config.LocalConfigFileName) {
				t.Errorf("err = %v; want ErrConfigLoad containing %q and naming %s", err, tc.want, config.LocalConfigFileName)
			}
		})
	}
}

// AC1 of plan 2026-10-30-script-local-config: a ralph.toml that sets one of
// the five local-only keys the scripts read, in lower or upper case, exits 3,
// naming the file and the key and saying to set it in ralph.local.toml, also
// with the personal file off (TestMain's setting).
func TestConfigGet_RalphTomlThatSetsAScriptLocalOnlyKeyExits3(t *testing.T) {
	for _, k := range []struct{ table, key, value string }{
		{"plan_visual", "browser", `"/usr/bin/chromium"`},
		{"plan_visual", "opener", `"open"`},
		{"standard", "base_branch", `"main"`},
		{"verify", "base", `"HEAD~1"`},
		{"verify", "skip_branch_secret_scan", "true"},
	} {
		for _, tc := range []struct{ name, doc string }{
			{"lower case", "[" + k.table + "]\n" + k.key + " = " + k.value + "\n"},
			{"upper case", "[" + strings.ToUpper(k.table) + "]\n" + strings.ToUpper(k.key) + " = " + k.value + "\n"},
		} {
			t.Run(k.table+"."+k.key+" in "+tc.name, func(t *testing.T) {
				t.Chdir(t.TempDir())
				writeTestFile(t, "ralph.toml", tc.doc)
				stdout, _, err := runRootCmdSplit(t, "config", "get", k.table+"."+k.key)
				if code := ExitCode(err); code != 3 || stdout != "" {
					t.Fatalf("config get %s.%s = (stdout %q, exit %d, err %v); want (\"\", 3)", k.table, k.key, stdout, code, err)
				}
				if want := "ralph.toml: [" + k.table + "]." + k.key + " can be set only in ralph.local.toml"; !errors.Is(err, ErrConfigLoad) || !strings.Contains(err.Error(), want) {
					t.Errorf("err = %v; want ErrConfigLoad containing %q", err, want)
				}
			})
		}
	}
}

// The help of get says where the values come from besides ralph.toml: the
// personal file applied on top, read from the main worktree's root, and
// skipped with RALPH_LOCAL_CONFIG=off (plan 2026-10-29-local-config-org).
func TestConfigGet_HelpNamesThePersonalFile(t *testing.T) {
	stdout, _, err := runRootCmdSplit(t, "config", "get", "--help")
	if err != nil {
		t.Fatalf("config get --help: %v", err)
	}
	help := strings.Join(strings.Fields(stdout), " ")
	for _, want := range []string{
		"ralph.local.toml, the git-ignored personal file, is applied on top of ralph.toml key by key",
		"It is read from the main worktree root of the git repository the config file's directory is in",
		"RALPH_LOCAL_CONFIG=off skips it.",
		"A ralph.local.toml that cannot be read, cannot be parsed, or fails validation exits 3, like ralph.toml.",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("config get --help lacks %q:\n%s", want, stdout)
		}
	}
}

// ExitCode maps the two sentinels of `ralph config get` to 2 and 3, also
// when wrapped.
func TestExitCode_ConfigGet(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{"key", ErrConfigKey, 2},
		{"key wrapped", fmt.Errorf("config get: org.review: %w: it is a table", ErrConfigKey), 2},
		{"load", ErrConfigLoad, 3},
		{"load wrapped", fmt.Errorf("config get: ralph.toml: %w: %w", ErrConfigLoad, errors.New("toml: expected character ]")), 3},
	} {
		if got := ExitCode(tc.err); got != tc.want {
			t.Errorf("%s: ExitCode(%v) = %d; want %d", tc.name, tc.err, got, tc.want)
		}
	}
}
