package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// Tests for `ralph org reseat --restart-herdr` and `--claude-config-dir`
// through the CLI (spec docs/specs/2026-11-20-org-reseat-restart-herdr.md;
// plan docs/plans/archive/2026-11-20-org-reseat-restart-herdr.md, S2: the CLI
// side of AC1 to AC6). They run the command on the reseat fixture of
// org_reseat_test.go (newReseatCLIOrg: the stub herdr and agmsg), with the
// herdr server's status read, lookup and launch on the fakes of
// org_herdr_server_test.go (setOrgHerdrServerSeams) and the read of the
// server's CLAUDE_CONFIG_DIR on a fake (setOrgReseatClaudeConfigDirReader),
// so no test runs a real herdr, lsof or ps. Every CLAUDE_CONFIG_DIR a test
// sets or reads is a fake value. None runs in parallel: they swap
// package-level vars and the environment.

// setOrgReseatClaudeConfigDirReader makes the reads of the herdr server's
// CLAUDE_CONFIG_DIR answer gots in order, the last one again once they run
// out, for the rest of the test, and returns the count of its reads.
func setOrgReseatClaudeConfigDirReader(t *testing.T, gots ...orgHerdrServerClaudeConfigDir) *int {
	t.Helper()
	if len(gots) == 0 {
		t.Fatal("setOrgReseatClaudeConfigDirReader needs an answer")
	}
	orig := orgHerdrServerClaudeConfigDirReader
	t.Cleanup(func() { orgHerdrServerClaudeConfigDirReader = orig })
	reads := new(int)
	orgHerdrServerClaudeConfigDirReader = func(context.Context) orgHerdrServerClaudeConfigDir {
		got := gots[min(*reads, len(gots)-1)]
		*reads++
		return got
	}
	return reads
}

// setCallerClaudeConfigDir sets this process's CLAUDE_CONFIG_DIR to value
// for the rest of the test, or unsets it when value is "".
func setCallerClaudeConfigDir(t *testing.T, value string) {
	t.Helper()
	t.Setenv(orgClaudeConfigDirName, value)
	if value == "" {
		if err := os.Unsetenv(orgClaudeConfigDirName); err != nil {
			t.Fatal(err)
		}
	}
}

// claudeAccountDir makes a directory named name for a --claude-config-dir,
// one plain shell word, so a printed command that holds it splits on spaces.
func claudeAccountDir(t *testing.T, name string) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir = filepath.Join(dir, name)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if org.ShellWord(dir) != dir {
		t.Fatalf("the test's directory %q is not one plain shell word", dir)
	}
	return dir
}

// herdrLogLinesFrom is the lines of the stub herdr's log from index from.
func herdrLogLinesFrom(t *testing.T, fx reseatCLIOrg, from int) []string {
	t.Helper()
	return readLogLines(t, fx.herdrLog)[from:]
}

// addStubHerdrPane puts a pane that is no seat of the fixture in the stub
// herdr's view, in workspace w9, labelled label when it is not "".
func addStubHerdrPane(t *testing.T, paneID, label string) {
	t.Helper()
	state := os.Getenv("ORG_STUB_HERDR_STATE")
	writeTestFile(t, filepath.Join(state, "pane-"+paneID), "w9:t1 w9\n")
	if label != "" {
		writeTestFile(t, filepath.Join(state, "label-pane-"+paneID), label+"\n")
	}
}

// reseatStopLines are the lines of the fixture's three stops, in order.
var reseatStopLines = []string{
	"stopped: reviewer (reviewer, claude CLI default)",
	"stopped: implementer (implementer, claude opus)",
	"stopped: leader (leader, claude sonnet)",
}

// TestOrgReseat_RestartHerdrStopsTheServerBetweenTheStopsAndTheStarts is
// AC1 through the CLI: `--restart-herdr` stops the seats, lists the panes
// again, runs `herdr server stop` right after that list, reads the status
// until the server is not running, and starts the seats; stdout says when
// the server stopped and started, stderr says that a herdr TUI closes with
// the server, and the reseat exits 0 with its reseated line.
func TestOrgReseat_RestartHerdrStopsTheServerBetweenTheStopsAndTheStarts(t *testing.T) {
	fx := newReseatCLIOrg(t)
	setOrgWatchDuration(t, &orgHerdrServerPollInterval, 10*time.Millisecond)
	f := &herdrGateFake{answers: []herdrStatusAnswer{herdrRunning, herdrNotRunning}}
	setOrgHerdrServerSeams(t, f)
	from := len(readLogLines(t, fx.herdrLog))

	stdout, stderr, err := fx.reseat(t, "--restart-herdr")
	if err != nil {
		t.Fatalf("reseat --restart-herdr: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	id := reseatIDOf(t, stdout)
	assertReseatLines(t, stdout, append(slices.Clone(reseatStopLines),
		"stopped: herdr server",
		"started: herdr server",
		"started: leader (leader, claude sonnet, pane *)",
		"started: implementer (implementer, claude opus, pane *)",
		"started: reviewer (reviewer, claude CLI default, pane *)",
		"reseated org_id a (reseat="+id+")",
	))
	if stderr != orgReseatTUINote+"\n" {
		t.Errorf("stderr = %q; want the TUI note alone", stderr)
	}
	lines := herdrLogLinesFrom(t, fx, from)
	stop := slices.Index(lines, "server stop")
	count := func(want string) int {
		n := 0
		for _, l := range lines {
			if l == want {
				n++
			}
		}
		return n
	}
	if stop < 1 || lines[stop-1] != "pane list" || count("pane list") != 2 || count("server stop") != 1 {
		t.Errorf("herdr calls %q; want two full pane lists, the second right before the one server stop", lines)
	}
	if first := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(l, "workspace create") }); first < stop {
		t.Errorf("herdr calls %q; want the leader's workspace made after the server stop", lines)
	}
	if f.reads != 2 {
		t.Errorf("status reads = %d; want 2 (before the gates, then the wait)", f.reads)
	}
}

// TestOrgReseat_RestartHerdrRefusesAnotherPane is AC2 through the CLI: a
// pane of the stub herdr that is no seat of the org refuses the reseat with
// exit 1, the pane's id, label and cwd, and the way to tell and close a
// pane herdr restored, with nothing on stdout, nothing closed or stopped,
// and the manifest as it was.
func TestOrgReseat_RestartHerdrRefusesAnotherPane(t *testing.T) {
	fx := newReseatCLIOrg(t)
	setOrgHerdrServerSeams(t, &herdrGateFake{answers: []herdrStatusAnswer{herdrRunning}})
	addStubHerdrPane(t, "w9:p1", "shell")
	from, manifest := len(readLogLines(t, fx.herdrLog)), fx.manifest(t)

	stdout, _, err := fx.reseat(t, "--restart-herdr")
	want := `org: reseat of org_id "a" is refused before anything is stopped: --restart-herdr stops the herdr server, which ` +
		`closes every pane it holds, and the server holds a pane that is not a pane of org_id "a"'s active seats: w9:p1 ` +
		`(label "shell", no cwd reported); a pane for which herdr pane process-info <pane_id> finds no process is only the ` +
		`record of a pane that herdr restored after an earlier herdr server stop, and herdr pane close <pane_id> closes it; ` +
		`the others (another org's seats, a headless director in herdr, a shell) end with the server, so run the reseat ` +
		`with --restart-herdr again once they are gone, or without it to start the seats on the server as it is`
	if err == nil || err.Error() != want {
		t.Fatalf("reseat --restart-herdr error\n%v\nwant\n%q", err, want)
	}
	if stdout != "" {
		t.Errorf("a refused reseat printed on stdout:\n%s", stdout)
	}
	for _, line := range herdrLogLinesFrom(t, fx, from) {
		if strings.HasPrefix(line, "pane close") || strings.HasPrefix(line, "workspace close") || line == "server stop" {
			t.Errorf("the refused reseat ran herdr %s", line)
		}
	}
	if fx.manifest(t) != manifest {
		t.Error("the refused reseat changed the manifest")
	}
}

// TestOrgReseat_RestartHerdrRefusals covers AC1b and AC3 through the CLI:
// --restart-herdr with --seat naming only the reviewer, and from a herdr
// pane, are refused with exit 1, nothing on stdout, and nothing recorded.
func TestOrgReseat_RestartHerdrRefusals(t *testing.T) {
	const refused = `org: reseat of org_id "a" is refused before anything is stopped: `
	for _, tc := range []struct {
		name string
		pane string
		args []string
		want string
	}{
		{"a subset of the active seats", "", []string{"--seat", "reviewer"},
			refused + `--restart-herdr stops the herdr server, which closes the panes of every seat of org_id "a", so it ` +
				"takes every active seat of the org, and the active seats leader, implementer are not named with --seat: run " +
				"it with no --seat, or name them too"},
		{"from a herdr pane", "w9:p7", nil,
			refused + "--restart-herdr stops the herdr server, which closes every pane it holds, and this command runs in " +
				"the herdr pane w9:p7: run ralph org reseat --restart-herdr from a shell outside herdr"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newReseatCLIOrg(t)
			f := &herdrGateFake{answers: []herdrStatusAnswer{herdrRunning}}
			setOrgHerdrServerSeams(t, f)
			t.Setenv("HERDR_PANE_ID", tc.pane)
			manifest := fx.manifest(t)

			stdout, _, err := fx.reseat(t, append([]string{"--restart-herdr"}, tc.args...)...)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("reseat error\n%v\nwant\n%q", err, tc.want)
			}
			if stdout != "" || fx.manifest(t) != manifest || f.reads != 0 {
				t.Errorf("stdout %q, manifest changed %v, status reads %d; want nothing printed, recorded, or read",
					stdout, fx.manifest(t) != manifest, f.reads)
			}
		})
	}
}

// TestOrgReseat_RestartHerdrDryRunPrintsTheServerStepsAndStopsNothing covers
// `--dry-run --restart-herdr`: it reads the status, lists the panes once,
// prints the server's stop and start between the seats' stops and starts,
// and the TUI note on stderr, and closes, stops, starts and records nothing.
// A herdr with no `status server --json` is refused: the restart could not
// tell when the server stops.
func TestOrgReseat_RestartHerdrDryRunPrintsTheServerStepsAndStopsNothing(t *testing.T) {
	fx := newReseatCLIOrg(t)
	f := &herdrGateFake{answers: []herdrStatusAnswer{herdrRunning}}
	setOrgHerdrServerSeams(t, f)
	from, manifest := len(readLogLines(t, fx.herdrLog)), fx.manifest(t)

	stdout, stderr, err := fx.reseat(t, "--dry-run", "--restart-herdr")
	if err != nil {
		t.Fatalf("reseat --dry-run --restart-herdr: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	assertReseatLines(t, stdout, []string{
		"stop: reviewer (reviewer, claude CLI default)",
		"stop: implementer (implementer, claude opus)",
		"stop: leader (leader, claude sonnet)",
		"stop: herdr server",
		"start: herdr server",
		"start: leader (leader, claude sonnet)",
		"start: implementer (implementer, claude opus)",
		"start: reviewer (reviewer, claude CLI default)",
		orgReseatStartsWatchNote,
	})
	if stderr != orgReseatTUINote+"\n" {
		t.Errorf("stderr = %q; want the TUI note alone", stderr)
	}
	if got := herdrLogLinesFrom(t, fx, from); !slices.Equal(got, []string{"pane list"}) {
		t.Errorf("the dry run ran herdr %q; want one full pane list", got)
	}
	if fx.manifest(t) != manifest || f.reads != 2 {
		t.Errorf("manifest changed %v, status reads %d; want no change and 2 reads", fx.manifest(t) != manifest, f.reads)
	}

	setOrgHerdrServerSeams(t, &herdrGateFake{answers: []herdrStatusAnswer{
		{err: fmt.Errorf("herdr status server: %w", driver.ErrHerdrStatusJSONUnsupported)},
	}})
	_, _, err = fx.reseat(t, "--dry-run", "--restart-herdr")
	if err == nil || !strings.Contains(err.Error(), "reseat --restart-herdr: this herdr has no 'herdr status server --json' (herdr "+
		"0.7.5 has it), so reseat cannot tell whether the herdr server runs or has stopped") {
		t.Errorf("dry run on a herdr with no status --json: %v; want the restart's refusal", err)
	}
}

// TestOrgReseat_ClaudeConfigDirIsCheckedFirst covers FR-5's check through
// the CLI: a --claude-config-dir that does not exist, or that is a file, is
// refused before the ledger is even read (the --state-dir given does not
// exist), naming the value.
func TestOrgReseat_ClaudeConfigDirIsCheckedFirst(t *testing.T) {
	setupOrgStubPATH(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "a-file")
	writeTestFile(t, file, "x\n")
	missing := filepath.Join(dir, "no-such-dir")
	const refused = `org: reseat of org_id "a" is refused before anything is stopped: --claude-config-dir `
	for _, tc := range []struct {
		value, want string
	}{
		{missing, refused + fmt.Sprintf("%q is not an existing directory: ", missing)},
		{file, refused + fmt.Sprintf("%q is not a directory", file)},
	} {
		stdout, _, err := runOrgCmdStreams(t, "reseat", "--org-id", "a", "--state-dir", filepath.Join(dir, "no-ledger"),
			"--claude-config-dir", tc.value)
		if err == nil || !strings.HasPrefix(err.Error(), tc.want) || stdout != "" {
			t.Errorf("reseat --claude-config-dir %q: %v (stdout %q); want an error that starts %q", tc.value, err, stdout, tc.want)
		}
	}
}

// TestOrgReseat_ClaudeConfigDirReachesTheHerdrServerGate is AC5 through the
// CLI, with the real gate on the fake seams and a caller whose
// CLAUDE_CONFIG_DIR is a fake old account: a --restart-herdr reseat given
// --claude-config-dir starts the stopped server with that directory, made
// absolute, as its only CLAUDE_CONFIG_DIR; a reseat without --restart-herdr
// given an empty --claude-config-dir on a server that is not running starts
// it with none.
func TestOrgReseat_ClaudeConfigDirReachesTheHerdrServerGate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		restart bool
		empty   bool // --claude-config-dir '' in place of the directory
		answers []herdrStatusAnswer
	}{
		{"restart with a directory", true, false,
			[]herdrStatusAnswer{herdrRunning, herdrRunning, herdrNotRunning, herdrNotRunning, herdrRunning}},
		{"a stopped server and ''", false, true, []herdrStatusAnswer{herdrNotRunning, herdrRunning}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newReseatCLIOrg(t)
			setCallerClaudeConfigDir(t, "/accounts/old-fake")
			setOrgHerdrServerGateOverride(t, nil)
			setOrgWatchDuration(t, &orgHerdrServerPollInterval, 10*time.Millisecond)
			proc, _ := fakeHerdrServerProcess(4730, false)
			f := &herdrGateFake{answers: tc.answers, lookPath: "/opt/herdr/bin/herdr", proc: proc}
			setOrgHerdrServerSeams(t, f)
			dir := claudeAccountDir(t, "account-b")
			args := []string{"--claude-config-dir", dir}
			want := []string{"CLAUDE_CONFIG_DIR=" + dir}
			if tc.empty {
				args, want = []string{"--claude-config-dir", ""}, nil
			}
			if tc.restart {
				args = append(args, "--restart-herdr")
			}

			stdout, stderr, err := fx.reseat(t, args...)
			if err != nil {
				t.Fatalf("reseat %q: %v (stdout: %s stderr: %s)", args, err, stdout, stderr)
			}
			reseatIDOf(t, stdout)
			if len(f.launches) != 1 {
				t.Fatalf("herdr server launches = %d; want 1", len(f.launches))
			}
			if got, _ := claudeConfigDirEntries(f.launches[0].req.env); !slices.Equal(got, want) {
				t.Errorf("CLAUDE_CONFIG_DIR entries of the started server's environment = %q; want %q", got, want)
			}
			if !strings.Contains(stderr, "ralph org reseat: started the herdr server (pid 4730, ") {
				t.Errorf("stderr = %q; want the gate's started line", stderr)
			}
		})
	}
}

// TestOrgReseat_HerdrServerLineAndNotes is AC6 through the CLI, on dry
// runs: the line of the server's CLAUDE_CONFIG_DIR in its three states, the
// note when the value is not the wanted one (--claude-config-dir's, else
// this command's), the note of a --claude-config-dir that does not reach a
// server that runs when the value could not be read, and, with
// --restart-herdr, the TUI note and no mismatch note. A value that cannot
// be read is not a refusal. Each case's whole stderr is pinned.
func TestOrgReseat_HerdrServerLineAndNotes(t *testing.T) {
	const (
		mismatchEnv = "note: the herdr server runs with %s, and the seats start with its environment, so they will not run " +
			"with %s from this command's environment; add --restart-herdr to start the server again with this command's " +
			"environment"
		mismatchFlag = "note: the herdr server runs with %s, and the seats start with its environment, so they will not run " +
			"with %s, which --claude-config-dir asks for: it applies only when ralph starts a stopped herdr server; add " +
			"--restart-herdr to start the server again with it"
	)
	set := func(v string) orgHerdrServerClaudeConfigDir {
		return orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvSet, value: v}
	}
	unset := orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvUnset}
	unreadable := orgHerdrServerEnvCannotRead("lsof -t on the socket failed")
	for _, tc := range []struct {
		name    string
		got     orgHerdrServerClaudeConfigDir
		caller  string // this command's CLAUDE_CONFIG_DIR; "" for none
		flag    string // "dir" for --claude-config-dir <account dir>, "''" for '', "" for none
		restart bool
		status  herdrStatusAnswer
		want    []string // %DIR% stands for the account dir
	}{
		{name: "set, the same as this command's", got: set("/accounts/a"), caller: "/accounts/a",
			want: []string{"herdr server: CLAUDE_CONFIG_DIR=/accounts/a"}},
		{name: "set, not this command's", got: set("/accounts/a"), caller: "/accounts/b",
			want: []string{"herdr server: CLAUDE_CONFIG_DIR=/accounts/a",
				fmt.Sprintf(mismatchEnv, "CLAUDE_CONFIG_DIR=/accounts/a", "CLAUDE_CONFIG_DIR=/accounts/b")}},
		{name: "set, none here", got: set("/accounts/a"),
			want: []string{"herdr server: CLAUDE_CONFIG_DIR=/accounts/a",
				fmt.Sprintf(mismatchEnv, "CLAUDE_CONFIG_DIR=/accounts/a", "no CLAUDE_CONFIG_DIR")}},
		{name: "unset, none here either", got: unset, want: []string{"herdr server: CLAUDE_CONFIG_DIR=unset"}},
		{name: "unset, one here", got: unset, caller: "/accounts/b",
			want: []string{"herdr server: CLAUDE_CONFIG_DIR=unset",
				fmt.Sprintf(mismatchEnv, "no CLAUDE_CONFIG_DIR", "CLAUDE_CONFIG_DIR=/accounts/b")}},
		{name: "set, not the flag's", got: set("/accounts/a"), caller: "/accounts/a", flag: "dir",
			want: []string{"herdr server: CLAUDE_CONFIG_DIR=/accounts/a",
				fmt.Sprintf(mismatchFlag, "CLAUDE_CONFIG_DIR=/accounts/a", "CLAUDE_CONFIG_DIR=%DIR%")}},
		{name: "set, the flag's", got: set("%DIR%"), caller: "/accounts/b", flag: "dir",
			want: []string{"herdr server: CLAUDE_CONFIG_DIR=%DIR%"}},
		{name: "set, the flag removes it", got: set("/accounts/a"), flag: "''",
			want: []string{"herdr server: CLAUDE_CONFIG_DIR=/accounts/a",
				fmt.Sprintf(mismatchFlag, "CLAUDE_CONFIG_DIR=/accounts/a", "no CLAUDE_CONFIG_DIR")}},
		{name: "unset, not the flag's", got: unset, flag: "dir",
			want: []string{"herdr server: CLAUDE_CONFIG_DIR=unset",
				fmt.Sprintf(mismatchFlag, "no CLAUDE_CONFIG_DIR", "CLAUDE_CONFIG_DIR=%DIR%")}},
		{name: "unset, the flag removes it", got: unset, caller: "/accounts/b", flag: "''",
			want: []string{"herdr server: CLAUDE_CONFIG_DIR=unset"}},
		{name: "unreadable", got: unreadable, caller: "/accounts/b",
			want: []string{"herdr server: CLAUDE_CONFIG_DIR cannot be read: lsof -t on the socket failed"}},
		{name: "unreadable, the flag on a server that runs", got: unreadable, flag: "dir", status: herdrRunning,
			want: []string{"herdr server: CLAUDE_CONFIG_DIR cannot be read: lsof -t on the socket failed",
				orgReseatClaudeConfigDirRunningNote}},
		{name: "unreadable, the flag on a server that does not run", got: unreadable, flag: "dir", status: herdrNotRunning,
			want: []string{"herdr server: CLAUDE_CONFIG_DIR cannot be read: lsof -t on the socket failed"}},
		{name: "restart, not this command's", got: set("/accounts/a"), caller: "/accounts/b", restart: true,
			want: []string{orgReseatTUINote, "herdr server: CLAUDE_CONFIG_DIR=/accounts/a"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newReseatCLIOrg(t)
			dir := claudeAccountDir(t, "account-b")
			got := tc.got
			got.value = strings.ReplaceAll(got.value, "%DIR%", dir)
			reads := setOrgReseatClaudeConfigDirReader(t, got)
			setCallerClaudeConfigDir(t, tc.caller)
			status := tc.status
			if status == (herdrStatusAnswer{}) {
				status = herdrRunning
			}
			setOrgHerdrServerSeams(t, &herdrGateFake{answers: []herdrStatusAnswer{status, herdrRunning}})
			args := []string{"--dry-run"}
			switch tc.flag {
			case "dir":
				args = append(args, "--claude-config-dir", dir)
			case "''":
				args = append(args, "--claude-config-dir", "")
			}
			if tc.restart {
				args = append(args, "--restart-herdr")
			}

			stdout, stderr, err := fx.reseat(t, args...)
			if err != nil {
				t.Fatalf("reseat %q: %v (stdout: %s stderr: %s)", args, err, stdout, stderr)
			}
			want := strings.ReplaceAll(strings.Join(tc.want, "\n")+"\n", "%DIR%", dir)
			if stderr != want {
				t.Errorf("stderr\n%s\nwant\n%s", stderr, want)
			}
			if *reads != 1 || !strings.HasPrefix(stdout, "stop: reviewer") {
				t.Errorf("reads %d, stdout %q; want one read and the plan", *reads, stdout)
			}
		})
	}
}

// reseatCombined runs `ralph org reseat --org-id a --state-dir <ledger>` with
// extra, with stdout and stderr in one buffer, so a test sees the order of
// the lines of both.
func (fx reseatCLIOrg) reseatCombined(t *testing.T, extra ...string) (string, error) {
	t.Helper()
	return runOrgCmd(t, append([]string{"reseat", "--org-id", "a", "--state-dir", fx.stateDir}, extra...)...)
}

// restartedClaudeConfigDirNote is the note after the server's line read
// again, with %s for the server's value, then for "--claude-config-dir asks
// for <wanted>" or "this command's environment has <wanted>".
const restartedClaudeConfigDirNote = "note: after the restart the herdr server runs with %s, but %s: another process may " +
	"have started the server after reseat stopped it and before reseat could start it, and the seats started with the " +
	"server's environment"

// TestOrgReseat_RestartHerdrPrintsTheServersClaudeConfigDirAgain covers the
// line a real reseat prints once it started the herdr server again (the
// verify report's I6): the server's CLAUDE_CONFIG_DIR read a second time, in
// the form of the line printed before the reseat, after every line of the
// result (stdout and stderr share one buffer here, so the order is pinned),
// and, when it is not the wanted value (--claude-config-dir's, else this
// command's), a note that another process may have started the server and
// the seats started with its environment. A value that cannot be read is
// only its line: the reseat still exits 0. Each case's whole output is
// pinned.
func TestOrgReseat_RestartHerdrPrintsTheServersClaudeConfigDirAgain(t *testing.T) {
	set := func(v string) orgHerdrServerClaudeConfigDir {
		return orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvSet, value: v}
	}
	unset := orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvUnset}
	for _, tc := range []struct {
		name   string
		after  orgHerdrServerClaudeConfigDir // the second read; %DIR% stands for the account dir
		caller string                        // this command's CLAUDE_CONFIG_DIR; "" for none
		flag   string                        // "dir" for --claude-config-dir <account dir>, "''" for '', "" for none
		want   []string                      // the lines after the result's; %DIR% stands for the account dir
	}{
		{name: "the flag's", after: set("%DIR%"), caller: "/accounts/old-fake", flag: "dir",
			want: []string{"herdr server: CLAUDE_CONFIG_DIR=%DIR%"}},
		{name: "this command's", after: set("/accounts/b"), caller: "/accounts/b",
			want: []string{"herdr server: CLAUDE_CONFIG_DIR=/accounts/b"}},
		{name: "unset, as the flag asks", after: unset, caller: "/accounts/b", flag: "''",
			want: []string{"herdr server: CLAUDE_CONFIG_DIR=unset"}},
		{name: "not the flag's", after: set("/accounts/other"), caller: "/accounts/old-fake", flag: "dir",
			want: []string{"herdr server: CLAUDE_CONFIG_DIR=/accounts/other", fmt.Sprintf(restartedClaudeConfigDirNote,
				"CLAUDE_CONFIG_DIR=/accounts/other", "--claude-config-dir asks for CLAUDE_CONFIG_DIR=%DIR%")}},
		{name: "set, and the flag removes it", after: set("/accounts/other"), flag: "''",
			want: []string{"herdr server: CLAUDE_CONFIG_DIR=/accounts/other", fmt.Sprintf(restartedClaudeConfigDirNote,
				"CLAUDE_CONFIG_DIR=/accounts/other", "--claude-config-dir asks for no CLAUDE_CONFIG_DIR")}},
		{name: "not this command's", after: unset, caller: "/accounts/b",
			want: []string{"herdr server: CLAUDE_CONFIG_DIR=unset", fmt.Sprintf(restartedClaudeConfigDirNote,
				"no CLAUDE_CONFIG_DIR", "this command's environment has CLAUDE_CONFIG_DIR=/accounts/b")}},
		{name: "cannot be read", after: orgHerdrServerEnvCannotRead("lsof -t on the socket failed"), caller: "/accounts/b",
			want: []string{"herdr server: CLAUDE_CONFIG_DIR cannot be read: lsof -t on the socket failed"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newReseatCLIOrg(t)
			setOrgWatchDuration(t, &orgHerdrServerPollInterval, 10*time.Millisecond)
			setOrgHerdrServerSeams(t, &herdrGateFake{answers: []herdrStatusAnswer{herdrRunning, herdrNotRunning}})
			dir := claudeAccountDir(t, "account-b")
			after := tc.after
			after.value = strings.ReplaceAll(after.value, "%DIR%", dir)
			reads := setOrgReseatClaudeConfigDirReader(t,
				orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvSet, value: "/accounts/old-fake"}, after)
			setCallerClaudeConfigDir(t, tc.caller)
			args := []string{"--restart-herdr"}
			switch tc.flag {
			case "dir":
				args = append(args, "--claude-config-dir", dir)
			case "''":
				args = append(args, "--claude-config-dir", "")
			}

			out, err := fx.reseatCombined(t, args...)
			if err != nil {
				t.Fatalf("reseat %q: %v\n%s", args, err, out)
			}
			want := append([]string{orgReseatTUINote, "herdr server: CLAUDE_CONFIG_DIR=/accounts/old-fake"}, reseatStopLines...)
			want = append(want,
				"stopped: herdr server",
				"started: herdr server",
				"started: leader (leader, claude sonnet, pane *)",
				"started: implementer (implementer, claude opus, pane *)",
				"started: reviewer (reviewer, claude CLI default, pane *)",
				"reseated org_id a (reseat="+reseatIDOf(t, out)+")",
			)
			for _, line := range tc.want {
				want = append(want, strings.ReplaceAll(line, "%DIR%", dir))
			}
			assertReseatLines(t, out, want)
			if *reads != 2 {
				t.Errorf("reads of the server's CLAUDE_CONFIG_DIR = %d; want 2, before the reseat and after it", *reads)
			}
		})
	}
}

// TestOrgReseat_RestartHerdrPrintsTheServerLineAfterAFailedStart covers a
// reseat that started the herdr server again and then failed to start a
// seat: the server's line read again comes after the printed retry, with its
// note, and the reseat still fails as before.
func TestOrgReseat_RestartHerdrPrintsTheServerLineAfterAFailedStart(t *testing.T) {
	fx := newReseatCLIOrg(t)
	setOrgWatchDuration(t, &orgHerdrServerPollInterval, 10*time.Millisecond)
	setOrgHerdrServerSeams(t, &herdrGateFake{answers: []herdrStatusAnswer{herdrRunning, herdrNotRunning}})
	reads := setOrgReseatClaudeConfigDirReader(t, orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvSet, value: "/accounts/b"},
		orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvSet, value: "/accounts/other"})
	setCallerClaudeConfigDir(t, "/accounts/b")
	t.Setenv("ORG_STUB_FAIL", "pane:split")

	out, err := fx.reseatCombined(t, "--restart-herdr")
	if err == nil || !strings.Contains(err.Error(), `seat "implementer" did not start again, so the reseat started no seat after it`) {
		t.Fatalf("expected the implementer's failed start, got: %v\n%s", err, out)
	}
	want := append([]string{orgReseatTUINote, "herdr server: CLAUDE_CONFIG_DIR=/accounts/b"}, reseatStopLines...)
	assertReseatLines(t, out, append(want,
		"stopped: herdr server",
		"started: herdr server",
		"started: leader (leader, claude sonnet, pane *)",
		"start failed: implementer (implementer, claude opus)",
		"left stopped: reviewer (stopped by this reseat, not started again)",
		"to start the seats left, run:",
		"ralph org reseat --org-id a --seat implementer --seat reviewer --state-dir "+org.ShellWord(fx.stateDir),
		"herdr server: CLAUDE_CONFIG_DIR=/accounts/other",
		fmt.Sprintf(restartedClaudeConfigDirNote, "CLAUDE_CONFIG_DIR=/accounts/other",
			"this command's environment has CLAUDE_CONFIG_DIR=/accounts/b"),
	))
	if *reads != 2 {
		t.Errorf("reads of the server's CLAUDE_CONFIG_DIR = %d; want 2", *reads)
	}
}

// TestOrgReseat_NoServerLineAgainWithoutARestart covers the reseats that do
// not start the herdr server again, after which nothing reads its
// CLAUDE_CONFIG_DIR again: a reseat without --restart-herdr, a restart
// refused before anything stopped (another pane), and a restart whose server
// start failed after the stop. Each prints the line before the reseat and no
// other.
func TestOrgReseat_NoServerLineAgainWithoutARestart(t *testing.T) {
	for _, tc := range []struct {
		name    string
		restart bool
		setup   func(t *testing.T)
		wantErr string // "" for a reseat that ends well
	}{
		{name: "without --restart-herdr", setup: func(*testing.T) {}},
		{name: "refused, another pane", restart: true,
			setup:   func(t *testing.T) { addStubHerdrPane(t, "w9:p1", "shell") },
			wantErr: "and the server holds a pane that is not a pane of"},
		{name: "the server's start failed after the stop", restart: true,
			setup: func(t *testing.T) {
				calls := 0
				setOrgHerdrServerGateOverride(t, func() error {
					calls++
					if calls > 1 {
						return errors.New("herdr server did not answer within 10s")
					}
					return nil
				})
			},
			wantErr: "the herdr server stopped, and its start failed, so the reseat started no seat: herdr server did not answer within 10s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newReseatCLIOrg(t)
			setOrgWatchDuration(t, &orgHerdrServerPollInterval, 10*time.Millisecond)
			setOrgHerdrServerSeams(t, &herdrGateFake{answers: []herdrStatusAnswer{herdrRunning, herdrNotRunning}})
			reads := setOrgReseatClaudeConfigDirReader(t, orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvSet, value: "/accounts/b"})
			setCallerClaudeConfigDir(t, "/accounts/b")
			tc.setup(t)
			var args []string
			if tc.restart {
				args = append(args, "--restart-herdr")
			}

			out, err := fx.reseatCombined(t, args...)
			if (tc.wantErr == "") != (err == nil) || (err != nil && !strings.Contains(err.Error(), tc.wantErr)) {
				t.Fatalf("reseat %q: %v; want an error with %q (none for \"\")\n%s", args, err, tc.wantErr, out)
			}
			var lines []string
			for _, line := range outputLines(out) {
				if strings.HasPrefix(line, "herdr server: ") || strings.HasPrefix(line, "note: after the restart") {
					lines = append(lines, line)
				}
			}
			if !slices.Equal(lines, []string{"herdr server: CLAUDE_CONFIG_DIR=/accounts/b"}) || *reads != 1 {
				t.Errorf("the server's lines %q after %d reads; want the line before the reseat alone, from one read\n%s",
					lines, *reads, out)
			}
			if strings.Contains(out, "started: herdr server") {
				t.Errorf("a reseat that did not start the server again says it did:\n%s", out)
			}
		})
	}
}

// TestOrgReseat_RestartHerdrRetryAfterTheServerStarted is AC4b through the
// CLI: the server stops and starts again, the leader starts, and the
// implementer's start fails. The printed retry has --claude-config-dir and no
// --restart-herdr; run as it is, it lists no pane, stops no seat and not the
// server, and starts the implementer and the reviewer.
func TestOrgReseat_RestartHerdrRetryAfterTheServerStarted(t *testing.T) {
	fx := newReseatCLIOrg(t)
	setOrgWatchDuration(t, &orgHerdrServerPollInterval, 10*time.Millisecond)
	setOrgHerdrServerSeams(t, &herdrGateFake{answers: []herdrStatusAnswer{herdrRunning, herdrNotRunning}})
	dir := claudeAccountDir(t, "account-b")
	t.Setenv("ORG_STUB_FAIL", "pane:split")

	stdout, stderr, err := fx.reseat(t, "--restart-herdr", "--claude-config-dir", dir)
	if err == nil || !strings.Contains(err.Error(), `seat "implementer" did not start again, so the reseat started no seat after it`) {
		t.Fatalf("expected the implementer's failed start, got: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	retry := "ralph org reseat --org-id a --seat implementer --seat reviewer --claude-config-dir " + dir +
		" --state-dir " + org.ShellWord(fx.stateDir)
	assertReseatLines(t, stdout, append(slices.Clone(reseatStopLines),
		"stopped: herdr server",
		"started: herdr server",
		"started: leader (leader, claude sonnet, pane *)",
		"start failed: implementer (implementer, claude opus)",
		"left stopped: reviewer (stopped by this reseat, not started again)",
		"to start the seats left, run:",
		retry,
	))

	t.Setenv("ORG_STUB_FAIL", "")
	from := len(readLogLines(t, fx.herdrLog))
	stdout, stderr, err = runOrgCmdStreams(t, strings.Fields(strings.TrimPrefix(retry, "ralph org "))...)
	if err != nil {
		t.Fatalf("the printed retry: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "started: implementer (implementer, claude opus, pane ") ||
		!strings.Contains(stdout, "started: reviewer (reviewer, claude CLI default, pane ") || strings.Contains(stdout, "stopped:") {
		t.Errorf("the retry printed:\n%s\nwant the implementer and the reviewer started and nothing stopped", stdout)
	}
	for _, line := range herdrLogLinesFrom(t, fx, from) {
		if line == "pane list" || line == "server stop" || strings.HasPrefix(line, "pane close") {
			t.Errorf("the retry ran herdr %s", line)
		}
	}
}

// TestOrgReseatRestartHerdrServerProbe pins the probe of a restart: the
// status's running, and an error for a herdr with no `status server
// --json`, for no herdr on PATH, and for a status that cannot be read.
func TestOrgReseatRestartHerdrServerProbe(t *testing.T) {
	for _, tc := range []struct {
		name    string
		answer  herdrStatusAnswer
		running bool
		want    string // the error's words; "" for none
	}{
		{"running", herdrRunning, true, ""},
		{"not running", herdrNotRunning, false, ""},
		{"no status --json", herdrStatusAnswer{err: fmt.Errorf("herdr status server: %w", driver.ErrHerdrStatusJSONUnsupported)}, false,
			"org: reseat --restart-herdr: this herdr has no 'herdr status server --json' (herdr 0.7.5 has it), so reseat " +
				"cannot tell whether the herdr server runs or has stopped ("},
		{"no herdr", herdrStatusAnswer{err: fmt.Errorf("herdr: %w", exec.ErrNotFound)}, false,
			"org: reseat --restart-herdr: herdr is not on PATH ("},
		{"unreadable", herdrStatusAnswer{err: errors.New("socket refused")}, false,
			"org: reseat --restart-herdr: could not read the herdr server's status (socket refused)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setOrgHerdrServerSeams(t, &herdrGateFake{answers: []herdrStatusAnswer{tc.answer}})
			running, err := orgReseatRestartHerdrServerProbe()()
			if running != tc.running || (tc.want == "") != (err == nil) || (err != nil && !strings.HasPrefix(err.Error(), tc.want)) {
				t.Errorf("probe = %v, %v; want %v and an error that starts %q", running, err, tc.running, tc.want)
			}
			if tc.answer.err != nil && !errors.Is(err, tc.answer.err) {
				t.Errorf("probe error %v does not wrap %v", err, tc.answer.err)
			}
		})
	}
}

// TestOrgReseatHelp_SaysHowToRestartTheHerdrServer pins the key sentences of
// the --restart-herdr and --claude-config-dir paragraphs of `ralph org
// reseat --help` and the two flags.
func TestOrgReseatHelp_SaysHowToRestartTheHerdrServer(t *testing.T) {
	out, err := runOrgCmd(t, "reseat", "--help")
	if err != nil {
		t.Fatalf("reseat --help: %v", err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"--restart-herdr does that in one reseat: once the seats are stopped, it stops the herdr server with `herdr server " +
			"stop`, waits up to 10s for `herdr status server --json` to say it is not running, starts it again with this " +
			"command's environment, as the check above does, and then starts the seats.",
		"Stopping the server also closes any `herdr` TUI attached to it, which reseat says on stderr.",
		"--restart-herdr takes every active seat of the org (no --seat, or a --seat for each), and it is refused, with " +
			"nothing stopped, from any herdr pane (HERDR_PANE_ID), when the herdr server is not running (a reseat without it " +
			"starts the server), and when the server holds a pane that is not a pane of the org's active seats",
		"A pane for which `herdr pane process-info <pane_id>` finds no process is only the record of a pane herdr " +
			"restored after an earlier `herdr server stop`, and `herdr pane close <pane_id>` closes it.",
		"a pane that appeared while it stopped the seats ends the reseat there, with the seats stopped and the server running.",
		"--claude-config-dir '' removes it, so the seats use the default account. It applies only when reseat starts the " +
			"server: after --restart-herdr stopped it, or when the server was not running.",
		"repeats --restart-herdr only when the server was not seen stopped",
		"Before it stops anything, reseat prints on stderr the CLAUDE_CONFIG_DIR of the herdr server that runs",
		"With --restart-herdr it also runs the checks of the restart (see below), listing the herdr server's panes, and " +
			"prints where the server would stop and start.",
		"--restart-herdr once the seats are stopped, stop the herdr server",
		"--claude-config-dir string CLAUDE_CONFIG_DIR of the herdr server reseat starts",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("reseat --help does not say %q:\n%s", want, out)
		}
	}
}

// TestOrgHerdrServerHelp_SaysReseatCanSetClaudeConfigDir pins the sentence of
// the herdr server gate's help (orgHerdrServerHelp) about the override,
// which spawn, start, director start, and reseat print.
func TestOrgHerdrServerHelp_SaysReseatCanSetClaudeConfigDir(t *testing.T) {
	want := "CLAUDE_CONFIG_DIR, which picks the account of the claude seats, stays as this command has it, unless " +
		"`ralph org reseat --claude-config-dir <dir>` sets it to <dir> or, given '', removes it."
	for _, args := range [][]string{{"spawn", "--help"}, {"start", "--help"}, {"director", "start", "--help"}, {"reseat", "--help"}} {
		out, err := runOrgCmd(t, args...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if !strings.Contains(strings.Join(strings.Fields(out), " "), want) {
			t.Errorf("%v does not say %q", args, want)
		}
	}
}
