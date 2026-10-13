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
// log, by tester with no herdr pane, reason first when it is set.
func gateCLIRecord(reason, head, base, log string) string {
	d := "head=" + head + " base=" + base + " base_ref=refs/remotes/origin/main branch=feat/a by=tester herdr_pane=unset log=" + strconv.Quote(log)
	if reason != "" {
		d = "reason=" + reason + " " + d
	}
	return d
}

// AC1 through the CLI: in a clean feature worktree whose HEAD contains
// origin/main, with both scripts passing, gate prints exactly the four lines
// GATE: pass, HEAD:, BASE: and LOG: on stdout and nothing on stderr, exits 0,
// records one org-level gate_passed with what it printed and who ran it, and
// writes the log under <state-dir>/gates/ and nothing in the worktree. The
// scripts get RALPH_VERIFY_SCOPE=full and none of the caller's RALPH_VERIFY_*
// variables, and herdr and agmsg are never called.
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
	if len(lines) != 5 || lines[4] != "" {
		t.Fatalf("stdout = %q; want four lines", stdout)
	}
	logPath := gateCLILogLine(t, fx.stateDir, lines[3])
	if want := "GATE: pass\nHEAD: " + head + "\nBASE: " + base + "\nLOG: " + logPath + "\n"; stdout != want {
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
		events[0].OrgID != "a" || events[0].SeatID != "" || events[0].Details != want {
		t.Fatalf("gate events = %+v\nwant one gate_passed of a with Details %q", events, want)
	}
	if log := fileBytesOrAbsent(t, logPath); !strings.Contains(log, "==> ralph org gate: ./scripts/run-test.sh exited 0\n") {
		t.Errorf("log does not say the test script exited 0:\n%s", log)
	}
	if st := gateCLIGitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all", "--ignored"); st != "" {
		t.Errorf("gate wrote in the worktree:\n%s", st)
	}
	if h, a := readLogLines(t, herdrLog), readLogLines(t, agmsgLog); len(h) != 0 || len(a) != 0 {
		t.Errorf("gate called herdr %v or agmsg %v; want neither", h, a)
	}
}

// AC3 through the CLI: for each gate_failed reason, gate prints GATE: fail,
// REASON:, HEAD: (the head from before the scripts), BASE: and LOG:, then a
// blank line and the reason's detail (the failing script's last 20 lines,
// indented; the files the scripts left and a hint; or one line for
// head_moved), exits 1 with a one-line error naming the reason and no usage,
// and records one gate_failed with that reason.
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
			want := "GATE: fail\nREASON: " + tc.reason + "\nHEAD: " + head + "\nBASE: " + base + "\nLOG: " + logPath + "\n\n" + tc.detail
			if stdout != want {
				t.Fatalf("stdout =\n%s\nwant\n%s", stdout, want)
			}
			if calls := fx.calls(t); !slices.Equal(calls, tc.calls) {
				t.Errorf("scripts run = %v; want %v", calls, tc.calls)
			}
			events := gateCLIEvents(t, fx.stateDir)
			if wantDetails := gateCLIRecord(tc.reason, head, base, logPath); len(events) != 1 || events[0].Event != org.EventGateFailed ||
				events[0].Details != wantDetails {
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
	if err != nil || len(lines) != 4 || lines[0] != "GATE: pass" {
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
	if len(lines) != 4 {
		t.Fatalf("stdout = %q; want four lines", stdout)
	}
	logPath := gateCLILogLine(t, stateDir, lines[3])
	if want := "GATE: pass\nHEAD: " + head + "\nBASE: " + base + "\nLOG: " + logPath + "\n"; stdout != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", stdout, want)
	}
	if pwd := strings.TrimSpace(fileBytesOrAbsent(t, filepath.Join(out, "pwd-static"))); pwd != wt {
		t.Errorf("the static script ran in %s; want the feature worktree's root %s", pwd, wt)
	}
	events := gateCLIEvents(t, stateDir)
	if want := gateCLIRecord("", head, base, logPath); len(events) != 1 || events[0].Event != org.EventGatePassed || events[0].Details != want {
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

// gate --help says who runs it and where, what it checks first, how the
// scripts run, what it records and prints, and what an interrupt does;
// ralph org --help lists gate and names it in its description.
func TestOrgGateHelp(t *testing.T) {
	out, err := runOrgCmd(t, "gate", "--help")
	if err != nil {
		t.Fatalf("gate --help: %v", err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"The leader of an org started with `ralph org start --plan` runs it in the org's feature worktree (or a directory under it), on the head it pushes",
		"the org holds a complete feature binding and is running",
		"on the binding's branch (a detached HEAD is refused)",
		"git status --porcelain --untracked-files=all prints nothing (ignored files do not count)",
		"checks that HEAD contains the base ref's tip",
		"gate never fetches",
		"./scripts/run-static-verify.sh and ./scripts/run-test.sh are executable files in the worktree's root",
		"When a check fails, gate runs nothing and records nothing",
		"without the RALPH_VERIFY_* and HARNESS_VERIFY_* variables and with RALPH_VERIFY_SCOPE=full",
		"a log under <state dir>/gates/",
		"They can take minutes",
		"records gate_passed in the manifest",
		"gate_failed with a reason: static_failed, test_failed, head_moved",
		"Both records hold the head, the base, base_ref, the branch, who ran it (by=<USER>, and herdr_pane=set or unset",
		"GATE: pass, HEAD: <sha>, BASE: <sha> and LOG: <path> on stdout and exits 0",
		"GATE: fail, REASON: <reason>, HEAD:, BASE: and LOG:",
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
