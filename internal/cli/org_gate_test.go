package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org gate` (plan docs/plans/active/2026-10-13-org-gate.md,
// the CLI side of AC1, AC2, AC3 and AC5, and the Integration line of its test
// plan). The org layer's checks, base resolution and records are covered by
// internal/org/gate_test.go; these run the command on real git repositories
// with stub gate scripts and check what it prints, its exit (the error
// Execute returns), the ledger it picks, and its interrupt. None of them runs
// in parallel: they set environment variables and chdir, and the interrupt
// test signals the test process.

// gateCLIStub is a stub gate script (fmt verb: its name), the org layer
// tests' gateStub: it appends its name to $GATE_TEST_OUT/calls, saves its
// environment and working directory there, prints 25 numbered lines and one
// line on stderr, sources $GATE_TEST_OUT/hook-<name> when it exists, and
// exits with the number in $GATE_TEST_OUT/exit-<name> (0 when there is none).
const gateCLIStub = `#!/bin/sh
name=%s
printf '%%s\n' "$name" >> "$GATE_TEST_OUT/calls"
env > "$GATE_TEST_OUT/env-$name"
pwd -P > "$GATE_TEST_OUT/pwd-$name"
i=1
while [ "$i" -le 25 ]; do
  echo "$name line $i"
  i=$((i + 1))
done
echo "$name to stderr" >&2
if [ -f "$GATE_TEST_OUT/hook-$name" ]; then
  . "$GATE_TEST_OUT/hook-$name"
fi
exit "$(cat "$GATE_TEST_OUT/exit-$name" 2>/dev/null || echo 0)"
`

// gateCLIStubTail is what gate prints, indented, after a failure of the stub
// called name with no hook: its last 19 numbered lines and its stderr line.
func gateCLIStubTail(name string) string {
	var b strings.Builder
	for i := 7; i <= 25; i++ {
		fmt.Fprintf(&b, "  %s line %d\n", name, i)
	}
	return b.String() + "  " + name + " to stderr\n"
}

// gateCLILogPattern is the shape of a gate log's name for org a: the org,
// the UTC time, and an optional -<n> suffix.
var gateCLILogPattern = regexp.MustCompile(`^a-[0-9]{8}T[0-9]{6}Z(-[0-9]+)?\.log$`)

// orgGateFixture is org a seeded in the ledger at stateDir the way `ralph org
// start --plan` leaves it (seedGateOrg), bound to the git repository
// worktree, which is the feature worktree itself: one commit on main, with a
// .gitignore for .harness/ and the two stub gate scripts (writeGateCLIStubs),
// refs/remotes/origin/main set to main (there is no remote to contact: the
// gate reads only local refs), and feat/a checked out one commit ahead of it.
// The stubs report to out, outside the repository, through GATE_TEST_OUT.
// USER is tester and HERDR_PANE_ID empty (approveEnv), and the current
// directory is the worktree.
type orgGateFixture struct {
	stateDir string
	worktree string
	out      string
}

func newOrgGateFixture(t *testing.T) *orgGateFixture {
	t.Helper()
	isolateFeatureGitEnv(t)
	approveEnv(t)
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	fx := &orgGateFixture{stateDir: filepath.Join(tmp, "state"), worktree: filepath.Join(tmp, "repo"), out: filepath.Join(tmp, "out")}
	runLedgerTestGit(t, "init", "--quiet", "-b", "main", fx.worktree)
	writeTestFile(t, filepath.Join(fx.worktree, ".gitignore"), ".harness/\n")
	writeTestFile(t, filepath.Join(fx.worktree, "README.md"), "gate fixture\n")
	writeGateCLIStubs(t, fx.worktree)
	runLedgerTestGit(t, "-C", fx.worktree, "add", ".")
	gateCLICommit(t, fx.worktree, "init")
	runLedgerTestGit(t, "-C", fx.worktree, "update-ref", "refs/remotes/origin/main", "main")
	runLedgerTestGit(t, "-C", fx.worktree, "switch", "--quiet", "-c", "feat/a")
	gateCLIAddCommit(t, fx.worktree, "internal/a/a.txt")

	if err := os.MkdirAll(fx.out, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GATE_TEST_OUT", fx.out)
	seedGateOrg(t, fx.stateDir, "a", fx.worktree)
	t.Chdir(fx.worktree)
	return fx
}

// seedGateOrg records orgID in the ledger at stateDir as `ralph org start
// --plan` leaves it: a scope_reserved bound to feature orgID of split plan
// demo on feat/<orgID> in worktree, in approval mode per-plan, and its leader
// seat spawned there.
func seedGateOrg(t *testing.T, stateDir, orgID, worktree string) {
	t.Helper()
	appendSeedEvent(t, stateDir, org.ManifestEvent{
		OrgID: orgID, Event: org.EventScopeReserved, Worktree: worktree,
		Details: "paths=internal/" + orgID + "/ split=demo feature=" + orgID + " digest=0123456789ab branch=feat/" + orgID + " approval=per-plan",
	})
	appendSeedEvent(t, stateDir, org.ManifestEvent{
		OrgID: orgID, SeatID: org.LeaderIdentity, Event: org.EventSpawned, Role: org.LeaderIdentity,
		Driver: "claude", Model: "sonnet", PaneID: "pane-" + orgID, AgmsgTeam: "ralph-" + orgID, Worktree: worktree,
	})
}

// writeGateCLIStubs writes the two stub gate scripts (gateCLIStub) in dir's
// scripts/, executable.
func writeGateCLIStubs(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, rel := range map[string]string{"static": "run-static-verify.sh", "test": "run-test.sh"} {
		writeStubScript(t, filepath.Join(dir, "scripts", rel), fmt.Sprintf(gateCLIStub, name))
	}
}

// gateCLICommit commits what is staged in dir, or an empty commit.
func gateCLICommit(t *testing.T, dir, msg string) {
	t.Helper()
	runLedgerTestGit(t, "-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
		"commit", "--quiet", "--allow-empty", "-m", msg)
}

// gateCLIAddCommit writes the file rel in dir and commits it.
func gateCLIAddCommit(t *testing.T, dir, rel string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, rel+"\n")
	runLedgerTestGit(t, "-C", dir, "add", rel)
	gateCLICommit(t, dir, "add "+rel)
}

// gateCLIGitOut runs git -C dir args and returns its stdout, trimmed.
func gateCLIGitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		t.Fatalf("git -C %s %v: %v", dir, args, err)
	}
	return strings.TrimSpace(string(out))
}

// gateCLIAdvance points ref, in dir's repository, to a new empty commit on
// top of its tip without touching any checkout: the default branch moving on
// after the feature branch left it. It returns the new tip.
func gateCLIAdvance(t *testing.T, dir, ref string) string {
	t.Helper()
	sha := gateCLIGitOut(t, dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit-tree", "-p", ref, "-m", "moves on", ref+"^{tree}")
	runLedgerTestGit(t, "-C", dir, "update-ref", ref, sha)
	return sha
}

// head returns the worktree's HEAD and the tip of refs/remotes/origin/main.
func (fx *orgGateFixture) head(t *testing.T) (head, base string) {
	t.Helper()
	return gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD"), gateCLIGitOut(t, fx.worktree, "rev-parse", "refs/remotes/origin/main")
}

// gate runs `ralph org gate --org-id a --state-dir <the fixture's ledger>`
// with extra after it.
func (fx *orgGateFixture) gate(t *testing.T, extra ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runOrgCmdStreams(t, append([]string{"gate", "--org-id", "a", "--state-dir", fx.stateDir}, extra...)...)
}

// setExit makes the stub called name exit with code.
func (fx *orgGateFixture) setExit(t *testing.T, name string, code int) {
	t.Helper()
	writeTestFile(t, filepath.Join(fx.out, "exit-"+name), strconv.Itoa(code)+"\n")
}

// hook makes the stub called name source body before it exits.
func (fx *orgGateFixture) hook(t *testing.T, name, body string) {
	t.Helper()
	writeTestFile(t, filepath.Join(fx.out, "hook-"+name), body)
}

// calls returns the stubs that ran, in order: nil when none did.
func (fx *orgGateFixture) calls(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fx.out, "calls"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(string(data))
}

// gateCLIEvents returns the gate_passed and gate_failed events in the
// manifest under stateDir.
func gateCLIEvents(t *testing.T, stateDir string) []org.ManifestEvent {
	t.Helper()
	var out []org.ManifestEvent
	for _, ev := range readManifestEvents(t, org.ManifestPathIn(stateDir)) {
		if ev.Event == org.EventGatePassed || ev.Event == org.EventGateFailed {
			out = append(out, ev)
		}
	}
	return out
}

// gateCLILogLine checks that line is `LOG: <path>` for a new gate log of org
// a in stateDir's gates/ and returns the path.
func gateCLILogLine(t *testing.T, stateDir, line string) string {
	t.Helper()
	path, ok := strings.CutPrefix(line, "LOG: ")
	if !ok || filepath.Dir(path) != filepath.Join(stateDir, "gates") || !gateCLILogPattern.MatchString(filepath.Base(path)) {
		t.Fatalf("line %q is not LOG: %s/a-<UTC time>.log", line, filepath.Join(stateDir, "gates"))
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the gate log %s: %v", path, err)
	}
	return path
}

// gateCLIRecord is the Details gate records for org a on head and base with
// log, by tester with no herdr pane, reason first when it is set, with its
// load and tool keys as gateCLINormalize leaves them.
func gateCLIRecord(reason, head, base, log string) string {
	d := "head=" + head + " base=" + base + " base_ref=refs/remotes/origin/main branch=feat/a by=tester herdr_pane=unset" +
		gateCLILoadKeys + gateCLIToolKeys + " log=" + strconv.Quote(log)
	if reason != "" {
		d = "reason=" + reason + " " + d
	}
	return d
}

// The load average around the gate scripts (plan 2026-11-12-gate-lock-accept,
// AC5) and the tool versions measured before them (plan
// 2026-11-12-gate-tool-versions, AC2 and AC5) in the CLI's output and the
// gate record. The CLI tests read the machine's real load and run the real
// version probe: only the org package's own tests pin them. So
// gateCLINormalize checks the shape of each value (two decimals, or unknown,
// for a load; one token for a tool) and replaces the LOAD: and TOOLS: lines
// with gateCLILoadLine and gateCLIToolsLine and the record's load and tool
// keys with gateCLILoadKeys and gateCLIToolKeys, and a test compares the rest
// whole.
const (
	gateCLILoadValue = `(?:[0-9]+\.[0-9]{2}|unknown)`
	gateCLILoadLine  = "LOAD: <load>"
	gateCLILoadKeys  = " <load>"
	gateCLIToolValue = `[A-Za-z0-9._+-]+`
	gateCLIToolsLine = "TOOLS: <tools>"
	gateCLIToolKeys  = " <tools>"
)

var (
	gateCLILoadLinePattern = regexp.MustCompile(`(?m)^LOAD: start ` + gateCLILoadValue + `, highest ` + gateCLILoadValue +
		`, end ` + gateCLILoadValue + ` \([0-9]+ CPUs\)$`)
	gateCLILoadKeysPattern = regexp.MustCompile(` load_start=` + gateCLILoadValue + ` load_max=` + gateCLILoadValue +
		` load_end=` + gateCLILoadValue + ` ncpu=[0-9]+( |$)`)
	gateCLIToolsLinePattern = regexp.MustCompile(`(?m)^TOOLS: [a-z0-9]+ ` + gateCLIToolValue + `(?:, [a-z0-9]+ ` + gateCLIToolValue +
		`)*(?: \(see LOG\))?$`)
	gateCLIToolKeysPattern = regexp.MustCompile(` tool_dirs=[0-9]+(?: tool_[a-z0-9]+=` + gateCLIToolValue + `)+( |$)`)
	// gateCLIToolsLogPattern is the start of a gate log's tool lines in a
	// fixture, whose repository has no JS marker and no go.mod: the line that
	// says how they were measured, right after the load at the start, then
	// python3 at the root.
	gateCLIToolsLogPattern = regexp.MustCompile(`\n# load at start: [^\n]*\n# tools: measured before the scripts [^\n]*\n` +
		`# tool python3 \(\.\): [^\n]+\n`)
)

// gateCLINormalize is s, stdout or a Details, with its LOAD: and TOOLS: lines
// and its load and tool keys replaced (see gateCLILoadLine).
func gateCLINormalize(s string) string {
	s = gateCLILoadLinePattern.ReplaceAllString(s, gateCLILoadLine)
	s = gateCLIToolsLinePattern.ReplaceAllString(s, gateCLIToolsLine)
	s = gateCLIToolKeysPattern.ReplaceAllString(s, gateCLIToolKeys+"$1")
	return gateCLILoadKeysPattern.ReplaceAllString(s, gateCLILoadKeys+"$1")
}

// AC1 through the CLI: in a clean feature worktree whose HEAD contains
// origin/main, with both scripts passing, gate prints exactly the six lines
// GATE: pass, HEAD:, BASE:, LOG:, LOAD: and TOOLS: on stdout and nothing on
// stderr, exits 0, records one org-level gate_passed with what it printed and
// who ran it, and writes the log under <state-dir>/gates/, with the tool
// lines in its header, and nothing in the worktree. The scripts get
// RALPH_VERIFY_SCOPE=full and none of the caller's RALPH_VERIFY_* variables,
// and herdr and agmsg are never called.
func TestOrgGate_PassPrintsGateHeadBaseAndLog(t *testing.T) {
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	t.Setenv("RALPH_VERIFY_BASE", "HEAD")
	t.Setenv("RALPH_VERIFY_SCOPE", "changed")
	fx := newOrgGateFixture(t)
	head, base := fx.head(t)
	if head == base || len(head) != 40 || len(base) != 40 {
		t.Fatalf("fixture: HEAD %q and origin/main %q should be two 40-hex commits", head, base)
	}

	stdout, stderr, err := fx.gate(t)
	if err != nil {
		t.Fatalf("gate: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q; want nothing", stderr)
	}
	lines := strings.Split(stdout, "\n")
	if len(lines) != 7 || lines[6] != "" {
		t.Fatalf("stdout = %q; want six lines", stdout)
	}
	logPath := gateCLILogLine(t, fx.stateDir, lines[3])
	if want := "GATE: pass\nHEAD: " + head + "\nBASE: " + base + "\nLOG: " + logPath + "\n" + gateCLILoadLine + "\n" + gateCLIToolsLine + "\n"; gateCLINormalize(stdout) != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", stdout, want)
	}

	if calls := fx.calls(t); !slices.Equal(calls, []string{"static", "test"}) {
		t.Errorf("scripts run = %v; want static then test", calls)
	}
	env := fileBytesOrAbsent(t, filepath.Join(fx.out, "env-static"))
	if envLines := strings.Split(env, "\n"); !slices.Contains(envLines, "RALPH_VERIFY_SCOPE=full") ||
		slices.Contains(envLines, "RALPH_VERIFY_SCOPE=changed") || strings.Contains(env, "RALPH_VERIFY_BASE=") {
		t.Errorf("the static script's environment has the caller's RALPH_VERIFY_* or lacks RALPH_VERIFY_SCOPE=full:\n%s", env)
	}
	events := gateCLIEvents(t, fx.stateDir)
	if want := gateCLIRecord("", head, base, logPath); len(events) != 1 || events[0].Event != org.EventGatePassed ||
		events[0].OrgID != "a" || events[0].SeatID != "" || gateCLINormalize(events[0].Details) != want {
		t.Fatalf("gate events = %+v\nwant one gate_passed of a with Details %q", events, want)
	}
	log := fileBytesOrAbsent(t, logPath)
	if !strings.Contains(log, "==> ralph org gate: ./scripts/run-test.sh exited 0\n") {
		t.Errorf("log does not say the test script exited 0:\n%s", log)
	}
	if !gateCLIToolsLogPattern.MatchString(log) {
		t.Errorf("log does not have the tool lines after the load at the start:\n%s", log)
	}
	if st := gateCLIGitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all", "--ignored"); st != "" {
		t.Errorf("gate wrote in the worktree:\n%s", st)
	}
	if h, a := readLogLines(t, herdrLog), readLogLines(t, agmsgLog); len(h) != 0 || len(a) != 0 {
		t.Errorf("gate called herdr %v or agmsg %v; want neither", h, a)
	}
}

// AC3 through the CLI: for each gate_failed reason, gate prints GATE: fail,
// REASON:, HEAD: (the head from before the scripts), BASE:, LOG:, LOAD: and
// TOOLS:, then a blank line and the reason's detail (the failing script's
// last 20 lines, indented; the files the scripts left and a hint; or one line
// for head_moved), exits 1 with a one-line error naming the reason and no
// usage, and records one gate_failed with that reason.
func TestOrgGate_FailurePrintsTheReasonAndExits1(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(t *testing.T, fx *orgGateFixture)
		reason string
		calls  []string
		detail string
	}{
		{"the static script fails", func(t *testing.T, fx *orgGateFixture) { fx.setExit(t, "static", 3) },
			org.GateReasonStaticFailed, []string{"static"},
			"last 20 lines of ./scripts/run-static-verify.sh:\n" + gateCLIStubTail("static")},
		{"the test script fails", func(t *testing.T, fx *orgGateFixture) { fx.setExit(t, "test", 1) },
			org.GateReasonTestFailed, []string{"static", "test"},
			"last 20 lines of ./scripts/run-test.sh:\n" + gateCLIStubTail("test")},
		{"a script leaves a file in the worktree", func(t *testing.T, fx *orgGateFixture) {
			fx.hook(t, "static", "echo x > untracked.txt\n")
		}, org.GateReasonTreeChanged, []string{"static", "test"},
			"the scripts left 1 uncommitted change(s) or untracked file(s) in the feature worktree:\n" +
				"  ?? untracked.txt\n" +
				"hint: when the scripts write these files, add their paths to .gitignore and commit that; then run the gate again\n"},
		{"HEAD moves during the scripts", func(t *testing.T, fx *orgGateFixture) {
			fx.hook(t, "test", "git -c user.name=t -c user.email=t@example.com -c commit.gpgsign=false commit --quiet --allow-empty -m moved\n")
		}, org.GateReasonHeadMoved, []string{"static", "test"},
			"HEAD moved while the scripts ran: commit, reset, or switch nothing in the feature worktree while the gate runs, and run it again\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newOrgGateFixture(t)
			tc.setup(t, fx)
			head, base := fx.head(t)

			stdout, stderr, err := fx.gate(t)
			if want := "org: gate: recorded gate_failed (reason=" + tc.reason + "); nothing to push until the gate passes"; err == nil || err.Error() != want {
				t.Fatalf("gate = %v; want the error %q (stdout: %s)", err, want, stdout)
			}
			if stderr != "" || strings.Contains(stdout, "Usage:") {
				t.Errorf("stderr = %q, stdout = %q; want no usage and nothing on stderr", stderr, stdout)
			}
			lines := strings.Split(stdout, "\n")
			if len(lines) < 5 {
				t.Fatalf("stdout = %q; want the five header lines", stdout)
			}
			logPath := gateCLILogLine(t, fx.stateDir, lines[4])
			want := "GATE: fail\nREASON: " + tc.reason + "\nHEAD: " + head + "\nBASE: " + base + "\nLOG: " + logPath + "\n" + gateCLILoadLine +
				"\n" + gateCLIToolsLine + "\n\n" + tc.detail
			if gateCLINormalize(stdout) != want {
				t.Fatalf("stdout =\n%s\nwant\n%s", stdout, want)
			}
			if calls := fx.calls(t); !slices.Equal(calls, tc.calls) {
				t.Errorf("scripts run = %v; want %v", calls, tc.calls)
			}
			events := gateCLIEvents(t, fx.stateDir)
			if wantDetails := gateCLIRecord(tc.reason, head, base, logPath); len(events) != 1 || events[0].Event != org.EventGateFailed ||
				gateCLINormalize(events[0].Details) != wantDetails {
				t.Fatalf("gate events = %+v\nwant one gate_failed with Details %q", events, wantDetails)
			}
		})
	}
}

// printGateResult on the edges the stub scripts do not reach: a failing
// script that printed nothing or one line, and more than 20 files left in the
// worktree (the list stops at 20 and counts the rest).
func TestOrgGate_PrintGateResultEdges(t *testing.T) {
	const header = "GATE: fail\nREASON: %s\nHEAD: h\nBASE: b\nLOG: /l\n\n"
	many := make([]string, 22)
	var listed strings.Builder
	for i := range many {
		many[i] = fmt.Sprintf("?? f%02d", i)
		if i < 20 {
			listed.WriteString("  " + many[i] + "\n")
		}
	}
	cases := []struct {
		name string
		res  org.GateResult
		want string
	}{
		{"a test script that printed nothing", org.GateResult{Reason: org.GateReasonTestFailed},
			fmt.Sprintf(header, org.GateReasonTestFailed) + "./scripts/run-test.sh printed nothing\n"},
		{"a static script that printed one line", org.GateResult{Reason: org.GateReasonStaticFailed, Tail: []string{"boom"}},
			fmt.Sprintf(header, org.GateReasonStaticFailed) + "last line of ./scripts/run-static-verify.sh:\n  boom\n"},
		{"22 files left", org.GateResult{Reason: org.GateReasonTreeChanged, Dirty: many},
			fmt.Sprintf(header, org.GateReasonTreeChanged) +
				"the scripts left 22 uncommitted change(s) or untracked file(s) in the feature worktree:\n" + listed.String() +
				"  ... and 2 more\n" +
				"hint: when the scripts write these files, add their paths to .gitignore and commit that; then run the gate again\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.res.Head, tc.res.Base, tc.res.LogPath = "h", "b", "/l"
			var out bytes.Buffer
			err := printGateResult(&out, tc.res)
			if want := "org: gate: recorded gate_failed (reason=" + tc.res.Reason + "); nothing to push until the gate passes"; err == nil || err.Error() != want {
				t.Errorf("printGateResult = %v; want %q", err, want)
			}
			if out.String() != tc.want {
				t.Errorf("stdout =\n%s\nwant\n%s", out.String(), tc.want)
			}
		})
	}
}

// printGateResult's LOAD: line (plan 2026-11-12-gate-lock-accept, AC5 and
// AC6): after LOG:, on a pass and on a recorded failure, with the wait for
// max_load when the gate waited, the CPU count unknown when it is 0, and no
// line when no script ran (Load nil).
func TestOrgGate_PrintsTheLoadLine(t *testing.T) {
	cases := []struct {
		name string
		res  org.GateResult
		want string
	}{
		{"a pass", org.GateResult{Passed: true, Load: &org.GateLoad{Start: "16.24", Highest: "157.03", End: "80.10", NCPU: 12}},
			"GATE: pass\nHEAD: h\nBASE: b\nLOG: /l\nLOAD: start 16.24, highest 157.03, end 80.10 (12 CPUs)\n"},
		{"a pass after a wait for max_load", org.GateResult{Passed: true, Load: &org.GateLoad{Start: "30.00", Highest: "31.00",
			End: "29.00", NCPU: 12, MaxLoad: 24, Waited: 3*time.Minute + 400*time.Millisecond}},
			"GATE: pass\nHEAD: h\nBASE: b\nLOG: /l\nLOAD: start 30.00, highest 31.00, end 29.00 (12 CPUs), waited 3m0s for max_load 24\n"},
		{"max_load with no wait", org.GateResult{Passed: true, Load: &org.GateLoad{Start: "1.00", Highest: "2.00", End: "1.50",
			NCPU: 4, MaxLoad: 24}},
			"GATE: pass\nHEAD: h\nBASE: b\nLOG: /l\nLOAD: start 1.00, highest 2.00, end 1.50 (4 CPUs)\n"},
		{"an unknown load", org.GateResult{Passed: true, Load: &org.GateLoad{Start: "unknown", Highest: "unknown", End: "unknown"}},
			"GATE: pass\nHEAD: h\nBASE: b\nLOG: /l\nLOAD: start unknown, highest unknown, end unknown (CPU count unknown)\n"},
		{"no load", org.GateResult{Passed: true}, "GATE: pass\nHEAD: h\nBASE: b\nLOG: /l\n"},
		{"a recorded failure", org.GateResult{Reason: org.GateReasonTestFailed, Tail: []string{"boom"},
			Load: &org.GateLoad{Start: "2.00", Highest: "9.00", End: "3.00", NCPU: 8}},
			"GATE: fail\nREASON: test_failed\nHEAD: h\nBASE: b\nLOG: /l\nLOAD: start 2.00, highest 9.00, end 3.00 (8 CPUs)\n\n" +
				"last line of ./scripts/run-test.sh:\n  boom\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.res.Head, tc.res.Base, tc.res.LogPath = "h", "b", "/l"
			var out bytes.Buffer
			err := printGateResult(&out, tc.res)
			if (err == nil) != tc.res.Passed {
				t.Errorf("printGateResult = %v; want an error only for the failure", err)
			}
			if out.String() != tc.want {
				t.Errorf("stdout =\n%s\nwant\n%s", out.String(), tc.want)
			}
		})
	}
}

// printGateResult's TOOLS: line (plan 2026-11-12-gate-tool-versions, AC5):
// right after LOAD:, on a pass and on a recorded failure, each tool with its
// value in GateTools' order, mixed as it is with "(see LOG)" at the end, and
// no line when no script ran (Tools nil).
func TestOrgGate_PrintsTheToolsLine(t *testing.T) {
	load := &org.GateLoad{Start: "1.00", Highest: "2.00", End: "1.50", NCPU: 8}
	const loadLine = "LOAD: start 1.00, highest 2.00, end 1.50 (8 CPUs)\n"
	cases := []struct {
		name string
		res  org.GateResult
		want string
	}{
		{"a pass", org.GateResult{Passed: true, Load: load, Tools: &org.GateTools{Dirs: 1, Tools: []org.GateTool{
			{Name: "node", Value: "v24.22.1"}, {Name: "pnpm", Value: "10.10.0"}, {Name: "python3", Value: "3.12.1"}}}},
			"GATE: pass\nHEAD: h\nBASE: b\nLOG: /l\n" + loadLine + "TOOLS: node v24.22.1, pnpm 10.10.0, python3 3.12.1\n"},
		{"a mixed tool", org.GateResult{Passed: true, Load: load, Tools: &org.GateTools{Dirs: 2, Tools: []org.GateTool{
			{Name: "node", Value: "mixed"}, {Name: "yarn", Value: "missing"}, {Name: "python3", Value: "unknown"}}}},
			"GATE: pass\nHEAD: h\nBASE: b\nLOG: /l\n" + loadLine + "TOOLS: node mixed, yarn missing, python3 unknown (see LOG)\n"},
		{"no tool", org.GateResult{Passed: true, Load: load, Tools: &org.GateTools{Dirs: 1}},
			"GATE: pass\nHEAD: h\nBASE: b\nLOG: /l\n" + loadLine + "TOOLS: none measured\n"},
		{"no tools", org.GateResult{Passed: true}, "GATE: pass\nHEAD: h\nBASE: b\nLOG: /l\n"},
		{"a recorded failure", org.GateResult{Reason: org.GateReasonTestFailed, Tail: []string{"boom"}, Load: load,
			Tools: &org.GateTools{Dirs: 1, Tools: []org.GateTool{{Name: "python3", Value: "3.12.1"}}}},
			"GATE: fail\nREASON: test_failed\nHEAD: h\nBASE: b\nLOG: /l\n" + loadLine + "TOOLS: python3 3.12.1\n\n" +
				"last line of ./scripts/run-test.sh:\n  boom\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.res.Head, tc.res.Base, tc.res.LogPath = "h", "b", "/l"
			var out bytes.Buffer
			err := printGateResult(&out, tc.res)
			if (err == nil) != tc.res.Passed {
				t.Errorf("printGateResult = %v; want an error only for the failure", err)
			}
			if out.String() != tc.want {
				t.Errorf("stdout =\n%s\nwant\n%s", out.String(), tc.want)
			}
		})
	}
}

// AC6 through the CLI: gate reads [org.gate].max_load from its --config file
// and passes it on, so a value no machine's load reaches does not make it
// wait and the gate_passed records max_load=; a negative value fails the
// config's load, before any script runs or anything is recorded.
func TestOrgGate_PassesMaxLoadFromTheConfig(t *testing.T) {
	fx := newOrgGateFixture(t)
	head, base := fx.head(t)
	cfg := filepath.Join(t.TempDir(), "ralph.toml")
	writeTestFile(t, cfg, "[org.gate]\nmax_load = 100000\n")
	stdout, stderr, err := fx.gate(t, "--config", cfg)
	if err != nil || stderr != "" {
		t.Fatalf("gate --config %s = %v (stdout: %s stderr: %s); want a pass and nothing on stderr", cfg, err, stdout, stderr)
	}
	lines := outputLines(stdout)
	if len(lines) != 6 {
		t.Fatalf("stdout = %q; want six lines", stdout)
	}
	logPath := gateCLILogLine(t, fx.stateDir, lines[3])
	if want := "GATE: pass\nHEAD: " + head + "\nBASE: " + base + "\nLOG: " + logPath + "\n" + gateCLILoadLine + "\n" + gateCLIToolsLine + "\n"; gateCLINormalize(stdout) != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", stdout, want)
	}
	events := gateCLIEvents(t, fx.stateDir)
	want := strings.Replace(gateCLIRecord("", head, base, logPath), gateCLILoadKeys+gateCLIToolKeys,
		gateCLILoadKeys+" max_load=100000"+gateCLIToolKeys, 1)
	if len(events) != 1 || gateCLINormalize(events[0].Details) != want {
		t.Fatalf("gate events = %+v\nwant one gate_passed with Details %q", events, want)
	}
	if log := fileBytesOrAbsent(t, logPath); !strings.Contains(log, "\n# max_load: 100000; did not wait: the 1-minute load average was not above it\n") {
		t.Errorf("log does not have the max_load line:\n%s", log)
	}

	before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))
	writeTestFile(t, cfg, "[org.gate]\nmax_load = -1\n")
	stdout, _, err = fx.gate(t, "--config", cfg)
	if err == nil || !strings.Contains(err.Error(), "[org.gate].max_load must be >= 0, got -1") || stdout != "" {
		t.Fatalf("gate with max_load = -1: %v (stdout %q); want the config error and nothing on stdout", err, stdout)
	}
	if after := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); after != before {
		t.Errorf("gate with max_load = -1 changed the manifest:\nbefore %s\nafter  %s", before, after)
	}
	if calls := fx.calls(t); !slices.Equal(calls, []string{"static", "test"}) {
		t.Errorf("scripts run = %v; want only the first gate's static then test", calls)
	}
}

// AC2 through the CLI: a refusal before the scripts prints nothing on stdout
// or stderr (the refusal is the error, which main prints on stderr), exits 1,
// runs no script, and leaves the manifest as it was with no gate log.
func TestOrgGate_RefusalPrintsNothingAndRecordsNothing(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, fx *orgGateFixture) []string // the args after `gate`
		want  []string
	}{
		{"an untracked file", func(t *testing.T, fx *orgGateFixture) []string {
			writeTestFile(t, filepath.Join(fx.worktree, "untracked.txt"), "x\n")
			return []string{"--org-id", "a", "--state-dir", fx.stateDir}
		}, []string{"has 1 uncommitted change(s) or untracked file(s)", "\n  ?? untracked.txt"}},
		{"an org with no binding in the ledger", func(t *testing.T, fx *orgGateFixture) []string {
			return []string{"--org-id", "b", "--state-dir", fx.stateDir}
		}, []string{`org_id "b" is not running and holds no split plan feature binding`}},
		{"another git worktree", func(t *testing.T, fx *orgGateFixture) []string {
			other := filepath.Join(filepath.Dir(fx.worktree), "other")
			runLedgerTestGit(t, "init", "--quiet", "-b", "main", other)
			t.Chdir(other)
			return []string{"--org-id", "a", "--state-dir", fx.stateDir}
		}, []string{"the gate ran in the git worktree ", `not in org_id "a"'s feature worktree`}},
		{"a HEAD that does not contain origin/main", func(t *testing.T, fx *orgGateFixture) []string {
			gateCLIAdvance(t, fx.worktree, "refs/remotes/origin/main")
			return []string{"--org-id", "a", "--state-dir", fx.stateDir}
		}, []string{"does not contain refs/remotes/origin/main's tip", "run `git fetch origin main` and `git rebase refs/remotes/origin/main`"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newOrgGateFixture(t)
			args := tc.setup(t, fx)
			before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))

			stdout, stderr, err := runOrgCmdStreams(t, append([]string{"gate"}, args...)...)
			if err == nil || !strings.HasPrefix(err.Error(), "org: gate: ") {
				t.Fatalf("gate = %v; want a refusal starting \"org: gate: \" (stdout: %s)", err, stdout)
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("refusal %q does not say %q", err, w)
				}
			}
			if stdout != "" || stderr != "" {
				t.Errorf("stdout = %q, stderr = %q; want nothing", stdout, stderr)
			}
			if after := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); after != before {
				t.Errorf("a refused gate changed the manifest:\nbefore %s\nafter  %s", before, after)
			}
			if _, err := os.Stat(filepath.Join(fx.stateDir, "gates")); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("a refused gate made gates/ (stat: %v)", err)
			}
			if calls := fx.calls(t); calls != nil {
				t.Errorf("a refused gate ran %v", calls)
			}
		})
	}
}

// --org-id is required and must be a safe identifier, and gate takes no
// argument: each is refused before the ledger is read, with nothing printed,
// run, or written.
func TestOrgGate_OrgIDRequiredAndNoArguments(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"no --org-id", []string{"gate"}, "org: --org-id is required"},
		{"a blank --org-id", []string{"gate", "--org-id", " "}, "org: --org-id is required"},
		{"a malformed --org-id", []string{"gate", "--org-id", "../x"}, "org_id"},
		{"an argument", []string{"gate", "--org-id", "a", "extra"}, `unknown command "extra" for "ralph org gate"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newOrgGateFixture(t)
			before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))
			stdout, stderr, err := runOrgCmdStreams(t, append(tc.args, "--state-dir", fx.stateDir)...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("gate %v = %v; want an error containing %q", tc.args, err, tc.want)
			}
			if stdout != "" || stderr != "" {
				t.Errorf("stdout = %q, stderr = %q; want nothing", stdout, stderr)
			}
			if after := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); after != before || fx.calls(t) != nil {
				t.Errorf("a refused gate changed the manifest or ran %v", fx.calls(t))
			}
		})
	}
}

// --state-dir picks the ledger: without it, gate in the fixture's repository
// reads that repository's own .harness/state/org, which holds no org a, and
// is refused; with it, gate finds the seeded org, passes, and writes its log
// under that ledger's gates/.
func TestOrgGate_StateDirPicksTheLedger(t *testing.T) {
	fx := newOrgGateFixture(t)
	stdout, _, err := runOrgCmdStreams(t, "gate", "--org-id", "a")
	if err == nil || !strings.Contains(err.Error(), `org_id "a" is not running and holds no split plan feature binding`) {
		t.Fatalf("gate without --state-dir = %v; want the refusal of an org the default ledger does not hold", err)
	}
	if stdout != "" || fx.calls(t) != nil {
		t.Fatalf("gate without --state-dir printed %q or ran %v", stdout, fx.calls(t))
	}

	stdout, _, err = fx.gate(t)
	lines := outputLines(stdout)
	if err != nil || len(lines) != 6 || lines[0] != "GATE: pass" {
		t.Fatalf("gate --state-dir %s = %v, %q; want a pass", fx.stateDir, err, stdout)
	}
	gateCLILogLine(t, fx.stateDir, lines[3])
	if events := gateCLIEvents(t, fx.stateDir); len(events) != 1 {
		t.Fatalf("gate events in %s = %+v; want one", fx.stateDir, events)
	}
}

// The round trip of the test plan's Integration line: `ralph org start
// --plan` makes the feature worktree and the ledger (the main worktree's
// .harness/state/org, as neither command gets --state-dir), the feature gets
// a commit, and gate run from a directory under the feature worktree finds
// the ledger, runs the scripts in the worktree's root, prints the pass, and
// records it, with no watch running and no plan approved (AC7). Once
// origin/main moves on, the same gate is refused, printing and recording
// nothing.
func TestOrgGate_StartPlanThenGateInTheFeatureWorktree(t *testing.T) {
	setupOrgStubPATH(t)
	approveEnv(t)
	root := newFeatureRepo(t, ".claude/worktrees/\n.harness/\n")
	writeGateCLIStubs(t, root)
	runLedgerTestGit(t, "-C", root, "add", "scripts")
	gateCLICommit(t, root, "gate scripts")
	runLedgerTestGit(t, "-C", root, "update-ref", "refs/remotes/origin/main", "main")
	out := filepath.Join(t.TempDir(), "out")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GATE_TEST_OUT", out)
	t.Chdir(root)
	stateDir := filepath.Join(root, ".harness", "state", "org")
	plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))

	if stdout, stderr, err := startOrgFeature(t, plan, "a"); err != nil {
		t.Fatalf("start --plan --feature a: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	wt := filepath.Join(root, ".claude", "worktrees", "org-a")
	gateCLIAddCommit(t, wt, "internal/a/a.txt")
	t.Chdir(filepath.Join(wt, "internal", "a"))
	head, base := gateCLIGitOut(t, wt, "rev-parse", "HEAD"), gateCLIGitOut(t, wt, "rev-parse", "refs/remotes/origin/main")

	stdout, stderr, err := runOrgCmdStreams(t, "gate", "--org-id", "a")
	if err != nil {
		t.Fatalf("gate from %s: %v (stdout: %s stderr: %s)", filepath.Join(wt, "internal", "a"), err, stdout, stderr)
	}
	lines := outputLines(stdout)
	if len(lines) != 6 {
		t.Fatalf("stdout = %q; want six lines", stdout)
	}
	logPath := gateCLILogLine(t, stateDir, lines[3])
	if want := "GATE: pass\nHEAD: " + head + "\nBASE: " + base + "\nLOG: " + logPath + "\n" + gateCLILoadLine + "\n" + gateCLIToolsLine + "\n"; gateCLINormalize(stdout) != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", stdout, want)
	}
	if pwd := strings.TrimSpace(fileBytesOrAbsent(t, filepath.Join(out, "pwd-static"))); pwd != wt {
		t.Errorf("the static script ran in %s; want the feature worktree's root %s", pwd, wt)
	}
	events := gateCLIEvents(t, stateDir)
	if want := gateCLIRecord("", head, base, logPath); len(events) != 1 || events[0].Event != org.EventGatePassed ||
		gateCLINormalize(events[0].Details) != want {
		t.Fatalf("gate events = %+v\nwant one gate_passed with Details %q", events, want)
	}

	moved := gateCLIAdvance(t, root, "refs/remotes/origin/main")
	before := fileBytesOrAbsent(t, org.ManifestPathIn(stateDir))
	stdout, _, err = runOrgCmdStreams(t, "gate", "--org-id", "a")
	if err == nil || !strings.Contains(err.Error(), "does not contain refs/remotes/origin/main's tip "+moved) {
		t.Fatalf("gate after origin/main moved = %v; want the rebase refusal", err)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if after := fileBytesOrAbsent(t, org.ManifestPathIn(stateDir)); after != before {
		t.Errorf("a refused gate changed the manifest:\nbefore %s\nafter  %s", before, after)
	}
}

// runOrgCmdStreamsContext is runOrgCmdStreams with ctx as the command's
// context (ExecuteContext).
func runOrgCmdStreamsContext(ctx context.Context, args ...string) (stdout, stderr string, err error) {
	root := NewRootCmd()
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetArgs(append([]string{"org"}, args...))
	err = root.ExecuteContext(ctx)
	return out.String(), errBuf.String(), err
}

// waitForGateCLIFile waits up to 10s for path to exist.
func waitForGateCLIFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s did not appear within 10s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// gateCLIResult is one run of the command in a goroutine.
type gateCLIResult struct {
	stdout, stderr string
	err            error
}

// signalSelf returns an interrupt that sends sig to the test process. A
// channel of the test's own is registered for sig first, so a signal that
// came while gate's handler was not registered would not end the test binary
// (gate would then not return, and the test fails on its timeout).
func signalSelf(sig syscall.Signal) func(t *testing.T, cancel context.CancelFunc) {
	return func(t *testing.T, _ context.CancelFunc) {
		t.Helper()
		caught := make(chan os.Signal, 1)
		signal.Notify(caught, sig)
		t.Cleanup(func() { signal.Stop(caught) })
		if err := syscall.Kill(os.Getpid(), sig); err != nil {
			t.Fatalf("send %v to the test process: %v", sig, err)
		}
	}
}

// AC5 through the CLI: SIGINT, SIGTERM, or SIGHUP to the ralph process, or
// the command's context ending, while a script runs stops the script's
// process group, runs no test script, prints nothing on stdout, records
// nothing, and exits 1 with the org layer's interrupt error, which names the
// signal and the log. SIGHUP is what a closed terminal sends ralph; it does
// not reach the script's group, which is not ralph's.
func TestOrgGate_InterruptStopsTheScriptAndRecordsNothing(t *testing.T) {
	cases := []struct {
		name      string
		cause     string
		interrupt func(t *testing.T, cancel context.CancelFunc)
	}{
		{"SIGINT", "interrupt signal received", signalSelf(syscall.SIGINT)},
		{"SIGTERM", "terminated signal received", signalSelf(syscall.SIGTERM)},
		{"SIGHUP", "hangup signal received", signalSelf(syscall.SIGHUP)},
		{"the command's context is cancelled", "context canceled", func(_ *testing.T, cancel context.CancelFunc) { cancel() }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newOrgGateFixture(t)
			fx.hook(t, "static", `echo $$ > "$GATE_TEST_OUT/pid"`+"\n"+`touch "$GATE_TEST_OUT/started"`+"\nsleep 30\n")
			before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan gateCLIResult, 1)
			go func() {
				stdout, stderr, err := runOrgCmdStreamsContext(ctx, "gate", "--org-id", "a", "--state-dir", fx.stateDir)
				done <- gateCLIResult{stdout, stderr, err}
			}()
			waitForGateCLIFile(t, filepath.Join(fx.out, "started"))
			pid, err := strconv.Atoi(strings.TrimSpace(fileBytesOrAbsent(t, filepath.Join(fx.out, "pid"))))
			if err != nil {
				t.Fatalf("read the static script's pid: %v", err)
			}
			// When gate did not stop the script's group (its own, Setpgid),
			// the test does, so it does not outlive the test. Only on a
			// failure: after a pass the group is gone and its id may be
			// reused.
			t.Cleanup(func() {
				if t.Failed() {
					_ = syscall.Kill(-pid, syscall.SIGKILL)
				}
			})

			tc.interrupt(t, cancel)
			var r gateCLIResult
			select {
			case r = <-done:
			case <-time.After(15 * time.Second):
				t.Fatal("gate did not return within 15s of the interrupt")
			}
			if r.err == nil || !strings.HasPrefix(r.err.Error(), "org: gate: interrupted ("+tc.cause+"): the scripts were stopped and nothing was recorded; the output so far is in "+filepath.Join(fx.stateDir, "gates")+string(filepath.Separator)) {
				t.Fatalf("gate = %v; want the interrupt error with the cause %q and the log", r.err, tc.cause)
			}
			if r.stdout != "" || r.stderr != "" {
				t.Errorf("stdout = %q, stderr = %q; want nothing", r.stdout, r.stderr)
			}
			if after := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); after != before {
				t.Errorf("an interrupted gate changed the manifest:\nbefore %s\nafter  %s", before, after)
			}
			if calls := fx.calls(t); !slices.Equal(calls, []string{"static"}) {
				t.Errorf("scripts run = %v; want only static", calls)
			}
			deadline := time.Now().Add(5 * time.Second)
			for !errors.Is(syscall.Kill(-pid, 0), syscall.ESRCH) {
				if time.Now().After(deadline) {
					t.Fatalf("the static script's process group %d is still there 5s after gate returned", pid)
				}
				time.Sleep(20 * time.Millisecond)
			}
		})
	}
}

// A refusal at the record: when the org is disbanded while the scripts run,
// gate records nothing, prints nothing on stdout, and exits 1 with the org
// layer's refusal followed by the log the scripts wrote.
func TestOrgGate_OrgDisbandedWhileTheScriptsRanNamesTheLog(t *testing.T) {
	fx := newOrgGateFixture(t)
	// The stub waits for the release at most about 20s, and a test that
	// fails before it writes the release writes it on cleanup, so no stub
	// outlives the test.
	fx.hook(t, "static", `touch "$GATE_TEST_OUT/started"`+"\n"+
		`n=0; while [ ! -f "$GATE_TEST_OUT/release" ] && [ "$n" -lt 400 ]; do sleep 0.05; n=$((n + 1)); done`+"\n")
	t.Cleanup(func() { _ = os.WriteFile(filepath.Join(fx.out, "release"), nil, 0o644) })
	done := make(chan gateCLIResult, 1)
	go func() {
		stdout, stderr, err := runOrgCmdStreamsContext(context.Background(), "gate", "--org-id", "a", "--state-dir", fx.stateDir)
		done <- gateCLIResult{stdout, stderr, err}
	}()
	waitForGateCLIFile(t, filepath.Join(fx.out, "started"))
	appendSeedEvent(t, fx.stateDir, org.ManifestEvent{OrgID: "a", Event: org.EventDisbanded})
	writeTestFile(t, filepath.Join(fx.out, "release"), "")
	var r gateCLIResult
	select {
	case r = <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("gate did not return within 15s of the release")
	}

	if r.err == nil || !strings.HasPrefix(r.err.Error(), `org: gate: nothing was recorded, since org_id "a" changed while the scripts ran (`) {
		t.Fatalf("gate = %v; want the refusal at the record", r.err)
	}
	const logNote = "; the scripts' output is in "
	i := strings.LastIndex(r.err.Error(), logNote)
	if i < 0 {
		t.Fatalf("refusal %q does not name the log", r.err)
	}
	gateCLILogLine(t, fx.stateDir, "LOG: "+r.err.Error()[i+len(logNote):])
	if r.stdout != "" || r.stderr != "" {
		t.Errorf("stdout = %q, stderr = %q; want nothing", r.stdout, r.stderr)
	}
	if events := gateCLIEvents(t, fx.stateDir); len(events) != 0 {
		t.Errorf("gate events = %+v; want none", events)
	}
}

// gate writes the manifest, so it is refused while seats of an older,
// per-worktree ledger are active (guardLegacyOrgStateDir), before it reads
// the ledger or runs anything.
func TestOrgGate_LegacyLedgerGuard(t *testing.T) {
	setupOrgStubPATH(t)
	r := newLegacyLedgerRepo(t)
	spawnLedgerSeat(t, r.legacyDir, "seat-1")

	stdout, _, err := runOrgCmdStreams(t, "gate", "--org-id", "org-a")
	if err == nil || !strings.Contains(err.Error(), "refusing to change the org ledger") {
		t.Errorf("gate error = %v; want the legacy-ledger refusal", err)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if _, statErr := os.Stat(r.sharedDir); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("expected the shared ledger dir %s not to be created, stat err: %v", r.sharedDir, statErr)
	}
}

// gate --help says who runs it and where, what it checks first, the waits
// for gate.lock and [org.gate].max_load and the checks made again after
// them, how the scripts run, the tool versions it measures before them, what
// it records and prints (the load average and the tool versions included),
// that a human may accept a failure, and what an interrupt does;
// ralph org --help lists gate and names it in its description.
func TestOrgGateHelp(t *testing.T) {
	out, err := runOrgCmd(t, "gate", "--help")
	if err != nil {
		t.Fatalf("gate --help: %v", err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"In an org started with `ralph org start --plan`, the reviewer seat runs it at the leader's TASK, in the org's feature " +
			"worktree (or a directory under it), on the head the leader pushes",
		"A person may run it too.",
		"the org holds a complete feature binding and is running",
		"on the binding's branch (a detached HEAD is refused)",
		"git status --porcelain --untracked-files=all prints nothing (ignored files do not count)",
		"checks that HEAD contains the base ref's tip",
		"gate never fetches",
		"./scripts/run-static-verify.sh and ./scripts/run-test.sh are executable files in the worktree's root",
		"When a check fails, gate runs nothing and records nothing",
		"without the RALPH_VERIFY_* and HARNESS_VERIFY_* variables and with RALPH_VERIFY_SCOPE=full",
		"and with RALPH_LOCAL_CONFIG=off whatever the caller set, so no ralph.local.toml changes it either.",
		"a log under <state dir>/gates/",
		"They can take minutes",
		"records gate_passed in the manifest",
		"gate_failed with a reason: static_failed, test_failed, head_moved",
		"Both records hold the head, the base, base_ref, the branch, who ran it (by=<USER>, and herdr_pane=set or unset",
		// Plan 2026-11-12-gate-lock-accept, AC13: the wait for gate.lock,
		// the checks made again after it, the load average, max_load, and
		// the acceptance of a failure.
		"A ledger runs one gate's scripts at a time. Once its checks pass, gate takes gate.lock in the state dir.",
		"While another ralph org gate, or a ralph org regate that runs the scripts, holds it, gate waits for it with no time limit " +
			"and runs nothing: stderr gets a line that starts org: gate: waiting for <state dir>/gate.lock and names the holder " +
			"(its verb, its org_id or branch, when it took the lock, and its pid) and how long gate has waited, when the wait " +
			"starts, when the holder changes, and every 5 minutes, and one more line once gate took the lock.",
		"With [org.gate].max_load above 0 (read from the ralph.toml --config names, else from ./ralph.toml in the directory gate " +
			"runs in; 0, the default, turns it off), gate then waits, with the lock held, while the machine's 1-minute load " +
			"average is above it",
		"reads the load again every 30 s, and starts once it is not above it, or after 30 minutes anyway, saying so on stderr " +
			"and in the log. A load average that cannot be read starts the gate at once.",
		"An interrupt during either wait records nothing and writes no log.",
		"After the waits, with the lock held, gate reads the ledger and makes the checks above again, and refuses the same way " +
			"when one fails now: an org disbanded during the wait runs no script, and the scripts run on the HEAD those checks " +
			"find, so a commit made during the wait is what the gate checks.",
		"They get gate.lock's open file too, so a script that runs on after gate was killed keeps the next gate waiting until it ends.",
		"The log's header says how long gate waited for gate.lock and who held it, how the max_load wait went, and the load " +
			"average (1, 5, and 15 minutes) and the CPU count before the first script; its last line, the load average after " +
			"the last script and the highest 1-minute load average read every 30 s while they ran.",
		"the load average around the scripts (load_start, load_max, load_end, and ncpu, each unknown when it could not be read), " +
			"max_load when it was above 0,",
		"GATE: pass, HEAD: <sha>, BASE: <sha>, LOG: <path>, LOAD: start <1m>, highest <1m>, end <1m> (<n> CPUs), with how " +
			"long it waited for max_load when it did, and TOOLS: <tool> <version>,",
		"The reviewer writes the output to the file the leader's TASK names (.harness/state/gate-<org_id>.out in the worktree) " +
			"and copies the HEAD: and BASE: lines into its RESULT, and the leader checks them against that file and HEAD before it pushes",
		"The TASK's command sends stderr to that file too, so it also holds the lines of a wait for gate.lock or for max_load, " +
			"and the line of a probe that changed the worktree.",
		"GATE: fail, REASON: <reason>, HEAD:, BASE:, LOG:, LOAD: and TOOLS:",
		"a human may accept that failure with ralph org gate accept (see its --help), and ralph org merge-order then lists the " +
			"branch as accepted where a pass would be ready",
		// Plan 2026-11-12-gate-tool-versions, AC9: the tool versions the
		// gate measures before the scripts, in the log, the record, and the
		// TOOLS: line; the probe's environment and budget; the stderr line of
		// a probe that changed the worktree; and a failure on other versions.
		"Right after the load average and before the first script, gate measures the tools the scripts are likely to run, " +
			"as /bin/sh resolves them with the scripts' environment: python3, and go when a go.mod is tracked, at the root; " +
			"and, when a package.json, a lockfile, .node-version, or .nvmrc is tracked, node and the package manager (pnpm, " +
			"yarn, or npm) at the root and in each directory with a lockfile or a version file (at most 10 places; the log " +
			"counts the rest).",
		"The log's header gets one line per tool and place, # tool <tool> (<place>): <version> (<path>), with missing for a " +
			"tool not on PATH, unknown (<why>) for a version that cannot be read, and (mise shim) or (asdf shim) when the path " +
			"is one, then the # mise current: lines when mise is on PATH.",
		"A child a script starts through such a shim can see other versions, which gate does not measure.",
		"The probe runs with mise's auto-install, corepack's auto-pin and downloads, and npm's update check turned off and " +
			"with GOTOOLCHAIN=local, so it installs and writes nothing, and a version that is not installed shows as unknown; " +
			"the scripts get their environment unchanged.",
		"It has 15 seconds in all: a tool it did not reach in time is unknown, and the verdict never depends on it.",
		"When git status of the worktree differs after the probe, the log lists the lines, and stderr gets a line that starts " +
			"org: gate: the version probe changed the worktree; the scripts run as before, and a pass is recorded as " +
			"tree_changed while the change is there.",
		"the tools measured (tool_dirs, how many places, and tool_<tool> per tool, such as tool_node=v24.22.1: its version, " +
			"missing, unknown, or mixed when the places differ; each place's is in the log only), and the log.",
		"(such as TOOLS: node v24.22.1, pnpm 10.10.0, python3 3.12.1; a value is missing, unknown, or mixed as in the record, " +
			"and (see LOG) ends the line when one is mixed) on stdout and exits 0",
		"A failure on tool versions the repository does not ask for (pnpm's ERR_PNPM_UNSUPPORTED_ENGINE naming a node " +
			"version, say) is the seat's environment to fix, not one to accept: ralph doctor's \"Node toolchain\" check " +
			"compares the two, and docs/recipes/isolated-herdr-server.md (Version managers) says how to pin the versions.",
		"An interrupt (SIGINT, SIGTERM, or SIGHUP) stops the running script and records nothing",
		"Processes a script leaves running when it exits are stopped too, and the log says so",
		"it needs no running watch and no approved plan",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("gate --help does not say %q:\n%s", want, out)
		}
	}

	out, err = runOrgCmd(t, "--help")
	if err != nil {
		t.Fatalf("org --help: %v", err)
	}
	if !regexp.MustCompile(`(?m)^\s+gate\s+Run the verification scripts on a feature org's head`).MatchString(out) {
		t.Errorf("org --help does not list gate:\n%s", out)
	}
	if help := strings.Join(strings.Fields(out), " "); !strings.Contains(help,
		"`ralph org gate` runs the verification scripts on the head a feature org's leader pushes and records which head passed on which base") {
		t.Errorf("org --help does not describe gate:\n%s", out)
	}
}

// The wait for gate.lock through the CLI (plan
// docs/plans/active/2026-11-12-gate-lock-accept.md, AC1): while another
// process holds the ledger's gate.lock, gate runs no script and says on the
// command's stderr which lock it waits for and whose gate holds it; once the
// lock is free it passes, and stderr says it took the lock.
func TestOrgGate_WaitsForGateLockOnTheCommandsStderr(t *testing.T) {
	fx := newOrgGateFixture(t)
	path := filepath.Join(fx.stateDir, "gate.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	unlocked := false
	unlock := func() {
		if !unlocked {
			unlocked = true
			_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		}
	}
	t.Cleanup(unlock)
	if _, err := f.WriteString(`{"verb":"ralph org gate","org_id":"b","started_at":"2026-11-12T09:00:00Z","pid":` + strconv.Itoa(os.Getpid()) + "}\n"); err != nil {
		t.Fatal(err)
	}

	stderr := &regateCLIStderr{wrote: make(chan struct{})}
	done := make(chan gateCLIResult, 1)
	go func() {
		root := NewRootCmd()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(stderr)
		root.SetArgs([]string{"org", "gate", "--org-id", "a", "--state-dir", fx.stateDir})
		err := root.ExecuteContext(context.Background())
		done <- gateCLIResult{stdout: out.String(), err: err}
	}()
	want := fmt.Sprintf("org: gate: waiting for %s (waited 0s): held by ralph org gate of org_id \"b\", started 2026-11-12T09:00:00Z by pid %d; ",
		path, os.Getpid())
	deadline := time.Now().Add(30 * time.Second)
	for !strings.Contains(stderr.String(), want) {
		select {
		case got := <-done:
			t.Fatalf("gate returned %+v while gate.lock was held; want it waiting (stderr %q)", got, stderr.String())
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("stderr never said %q within 30s: %q", want, stderr.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
	if calls := fx.calls(t); calls != nil {
		t.Fatalf("scripts run while gate.lock was held = %v; want none", calls)
	}

	unlock()
	var got gateCLIResult
	select {
	case got = <-done:
	case <-time.After(60 * time.Second):
		t.Fatal("gate did not return within 60s of gate.lock being freed")
	}
	if got.err != nil || !strings.HasPrefix(got.stdout, "GATE: pass\n") {
		t.Fatalf("gate = %v, stdout %q; want a pass once gate.lock is free", got.err, got.stdout)
	}
	if !strings.Contains(stderr.String(), "org: gate: took "+path+" after waiting ") {
		t.Fatalf("stderr = %q; want the line that gate took gate.lock", stderr.String())
	}
}
