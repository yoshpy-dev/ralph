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
// key from outside this plan's tables print config.Default()'s values.
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
	writeTestFile(t, "ralph.toml", "[standard]\nplan_advisory = true\n\n[org.review]\nmax_cross_review_cycles = 5\n\n[org.approval]\nmode = \"split\"\n")
	for _, tc := range []struct{ key, want string }{
		{"standard.plan_advisory", "true\n"},
		{"org.review.max_cross_review_cycles", "5\n"},
		{"org.approval.mode", "split\n"},
		{"standard.cross_review", "false\n"},
		{"org.review.cross_review", "false\n"},
	} {
		stdout, stderr, code := runConfigGet(t, tc.key)
		if code != 0 || stdout != tc.want || stderr != "" {
			t.Errorf("config get %s = (stdout %q, stderr %q, exit %d); want (%q, \"\", 0)", tc.key, stdout, stderr, code, tc.want)
		}
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
// off, and exits 2 and 3 stop. Each phrase is matched with runs of white
// space, line breaks included, read as one space, so re-wrapping the help
// keeps the test green.
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
