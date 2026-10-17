package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org regate` (plan docs/plans/active/2026-10-16-org-regate.md,
// AC1 to AC9 at the command line, and the Integration line of its test plan).
// The org layer's checks, merge, records, undo, and push are covered by
// internal/org/regate_test.go; these check what the command prints for each
// outcome (printRegateResult on constructed results), its flags, and its exit
// (the error Execute returns), and run it on real git repositories whose
// origin is a bare repository in the test's temp dir, with the stub gate
// scripts of org_gate_test.go, so no test reaches a network. None of them
// runs in parallel: they set environment variables and chdir, and the
// interrupt test signals the test process.

// orgRegateFixture is the branch feat/a of the feature org a after its PR
// was opened and the org disbanded: origin is a bare repository whose main
// holds one commit (a .gitignore for .harness/, a README.md and the stub gate
// scripts, writeGateCLIStubs) and whose feat/a holds the feature's one commit
// on top of it. main is the main checkout, on main, with origin's HEAD set as
// a clone sets it and a user to make a merge commit with; the feature
// worktree is a linked worktree of it on feat/a. The ledger at stateDir holds
// a's binding to feat/a in that worktree, its leader, and its disband. The
// stubs report to out through GATE_TEST_OUT; USER is tester and HERDR_PANE_ID
// empty (approveEnv); the current directory is the main checkout.
type orgRegateFixture struct {
	*orgGateFixture
	origin, main string
}

func newOrgRegateFixture(t *testing.T) *orgRegateFixture {
	t.Helper()
	isolateFeatureGitEnv(t)
	approveEnv(t)
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	fx := &orgRegateFixture{
		orgGateFixture: &orgGateFixture{stateDir: filepath.Join(tmp, "state"), worktree: filepath.Join(tmp, "wt-a"), out: filepath.Join(tmp, "out")},
		origin:         filepath.Join(tmp, "origin.git"), main: filepath.Join(tmp, "main"),
	}
	runLedgerTestGit(t, "init", "--quiet", "--bare", "-b", "main", fx.origin)
	runLedgerTestGit(t, "init", "--quiet", "-b", "main", fx.main)
	runLedgerTestGit(t, "-C", fx.main, "config", "user.name", "Regate Tester")
	runLedgerTestGit(t, "-C", fx.main, "config", "user.email", "regate@example.com")
	writeTestFile(t, filepath.Join(fx.main, ".gitignore"), ".harness/\n")
	writeTestFile(t, filepath.Join(fx.main, "README.md"), "regate fixture\n")
	writeGateCLIStubs(t, fx.main)
	runLedgerTestGit(t, "-C", fx.main, "add", ".")
	gateCLICommit(t, fx.main, "init")
	runLedgerTestGit(t, "-C", fx.main, "remote", "add", "origin", fx.origin)
	runLedgerTestGit(t, "-C", fx.main, "push", "--quiet", "origin", "main")
	runLedgerTestGit(t, "-C", fx.main, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	runLedgerTestGit(t, "-C", fx.main, "worktree", "add", "--quiet", "-b", "feat/a", fx.worktree)
	gateCLIAddCommit(t, fx.worktree, "internal/a/a.txt")
	runLedgerTestGit(t, "-C", fx.worktree, "push", "--quiet", "origin", "feat/a")

	if err := os.MkdirAll(fx.out, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GATE_TEST_OUT", fx.out)
	seedGateOrg(t, fx.stateDir, "a", fx.worktree)
	appendSeedEvent(t, fx.stateDir, org.ManifestEvent{OrgID: "a", Event: org.EventDisbanded})
	t.Chdir(fx.main)
	return fx
}

// regate runs `ralph org regate --branch feat/a --state-dir <the fixture's
// ledger>` with extra after it.
func (fx *orgRegateFixture) regate(t *testing.T, extra ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runOrgCmdStreams(t, append([]string{"regate", "--branch", "feat/a", "--state-dir", fx.stateDir}, extra...)...)
}

// moveMain commits a file on main in the main checkout and pushes it: another
// feature's PR merged after feat/a's gate. It returns origin's new main.
func (fx *orgRegateFixture) moveMain(t *testing.T) string {
	t.Helper()
	gateCLIAddCommit(t, fx.main, "internal/other/other.txt")
	runLedgerTestGit(t, "-C", fx.main, "push", "--quiet", "origin", "main")
	return gateCLIGitOut(t, fx.origin, "rev-parse", "refs/heads/main")
}

// regateCLIRecord is the Details regate records for org a on head and base
// with log, by tester with no herdr pane, reason first when it is set.
func regateCLIRecord(reason, head, base, log string) string {
	d := "head=" + head + " base=" + base + " base_ref=refs/remotes/origin/main branch=feat/a by=tester herdr_pane=unset via=regate log=" +
		strconv.Quote(log)
	if reason != "" {
		d = "reason=" + reason + " " + d
	}
	return d
}

// AC4 and AC3 through the CLI, and merge-order after them: with origin's
// main moved on after feat/a's commit, regate merges it in with a merge
// commit, runs both scripts on it, records one gate_passed via=regate, pushes
// the merge commit to origin's feat/a, prints exactly GATE: pass, MERGED:,
// HEAD:, BASE:, LOG: and PUSHED:, and exits 0 with nothing on stderr, with no
// watch running and no plan approved. Run again, it runs, records, and pushes
// nothing and prints UP_TO_DATE: with the same HEAD:, BASE: and LOG:, and
// ralph org merge-order then lists feat/a as ready with the record via
// regate.
func TestOrgRegate_MergesPushesAndThenIsUpToDate(t *testing.T) {
	fx := newOrgRegateFixture(t)
	pre := gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD")
	tip := fx.moveMain(t)

	stdout, stderr, err := fx.regate(t)
	if err != nil || stderr != "" {
		t.Fatalf("regate: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	merge := gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD")
	if parents := strings.Fields(gateCLIGitOut(t, fx.worktree, "rev-list", "--parents", "-n", "1", merge))[1:]; !slices.Equal(parents, []string{pre, tip}) {
		t.Fatalf("HEAD %s has the parents %v; want the merge commit of origin/main %s into feat/a at %s", merge, parents, tip, pre)
	}
	lines := strings.Split(stdout, "\n")
	if len(lines) != 7 {
		t.Fatalf("stdout = %q; want six lines", stdout)
	}
	logPath := gateCLILogLine(t, fx.stateDir, lines[4])
	want := "GATE: pass\nMERGED: " + merge + " (origin/main into " + pre + ")\nHEAD: " + merge + "\nBASE: " + tip + "\nLOG: " + logPath +
		"\nPUSHED: " + merge + " to origin's feat/a\n"
	if stdout != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", stdout, want)
	}
	if got := gateCLIGitOut(t, fx.origin, "rev-parse", "refs/heads/feat/a"); got != merge {
		t.Errorf("origin's feat/a = %s; want the merge commit %s", got, merge)
	}
	if calls := fx.calls(t); !slices.Equal(calls, []string{"static", "test"}) {
		t.Errorf("scripts run = %v; want static then test", calls)
	}
	events := gateCLIEvents(t, fx.stateDir)
	if wantDetails := regateCLIRecord("", merge, tip, logPath); len(events) != 1 || events[0].Event != org.EventGatePassed ||
		events[0].OrgID != "a" || events[0].SeatID != "" || events[0].Details != wantDetails {
		t.Fatalf("gate events = %+v\nwant one gate_passed of a with Details %q", events, wantDetails)
	}

	before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))
	stdout, stderr, err = fx.regate(t)
	if err != nil || stderr != "" {
		t.Fatalf("regate again: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	want = "UP_TO_DATE: the latest gate record of feat/a is a gate_passed of HEAD, origin's tip of the branch, on origin/main's tip: " +
		"nothing was run, recorded, or pushed\nHEAD: " + merge + "\nBASE: " + tip + "\nLOG: " + logPath + "\n"
	if stdout != want {
		t.Fatalf("stdout of the second regate =\n%s\nwant\n%s", stdout, want)
	}
	if after := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); after != before {
		t.Errorf("an up-to-date regate changed the manifest:\nbefore %s\nafter  %s", before, after)
	}
	if calls := fx.calls(t); len(calls) != 2 {
		t.Errorf("scripts run = %v; want none run by the second regate", calls)
	}

	// merge-order runs no gh here: PATH holds only a git that execs the
	// real one (setMergeOrderPATH).
	setMergeOrderPATH(t, filepath.Join(filepath.Dir(fx.main), "bin"), false)
	stdout, stderr, err = runOrgCmdStreams(t, "merge-order", "--json", "--state-dir", fx.stateDir)
	if err != nil {
		t.Fatalf("merge-order: %v (stderr %s)", err, stderr)
	}
	if !strings.Contains(stdout, `"class": "ready"`) || !strings.Contains(stdout, `"branch": "feat/a"`) || !strings.Contains(stdout, `"via": "regate"`) {
		t.Fatalf("merge-order --json does not list feat/a ready with the record via regate:\n%s", stdout)
	}
}

// AC9 through the CLI: with HEAD origin's tip of feat/a and containing
// origin's main, but no gate record, regate merges nothing, runs both
// scripts, records a gate_passed via=regate, pushes nothing, prints GATE:
// pass, HEAD:, BASE: and LOG:, and exits 0.
func TestOrgRegate_GateOnlyPassPrintsNoMergeAndNoPush(t *testing.T) {
	fx := newOrgRegateFixture(t)
	head := gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD")
	tip := gateCLIGitOut(t, fx.origin, "rev-parse", "refs/heads/main")
	originBefore := gateCLIGitOut(t, fx.origin, "for-each-ref")

	stdout, stderr, err := fx.regate(t)
	if err != nil || stderr != "" {
		t.Fatalf("regate: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	lines := strings.Split(stdout, "\n")
	if len(lines) != 5 {
		t.Fatalf("stdout = %q; want four lines", stdout)
	}
	logPath := gateCLILogLine(t, fx.stateDir, lines[3])
	if want := "GATE: pass\nHEAD: " + head + "\nBASE: " + tip + "\nLOG: " + logPath + "\n"; stdout != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", stdout, want)
	}
	if got := gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD"); got != head {
		t.Errorf("HEAD = %s; want %s, as it was", got, head)
	}
	if after := gateCLIGitOut(t, fx.origin, "for-each-ref"); after != originBefore {
		t.Errorf("a gate-only regate changed origin:\nbefore %s\nafter  %s", originBefore, after)
	}
	if events := gateCLIEvents(t, fx.stateDir); len(events) != 1 || events[0].Details != regateCLIRecord("", head, tip, logPath) {
		t.Fatalf("gate events = %+v; want one gate_passed of %s via regate", events, head)
	}
}

// AC6 through the CLI: when the test script fails on the merge commit,
// regate records a gate_failed of it via=regate, undoes it, pushes nothing,
// prints GATE: fail, REASON:, MERGED:, HEAD:, BASE: and LOG:, the script's
// tail, and that the merge commit was undone, and exits 1 with a one-line
// error that names the reason and the merge commit.
func TestOrgRegate_FailureAfterAMergeUndoesItAndExits1(t *testing.T) {
	fx := newOrgRegateFixture(t)
	pre := gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD")
	tip := fx.moveMain(t)
	fx.setExit(t, "test", 1)

	stdout, stderr, err := fx.regate(t)
	merge := gateCLIGitOut(t, fx.worktree, "rev-parse", "feat/a@{1}")
	if want := "org: regate: recorded gate_failed (reason=test_failed) of the merge commit " + merge + ", which was undone; nothing was pushed"; err == nil ||
		err.Error() != want {
		t.Fatalf("regate = %v; want the error %q (stdout: %s)", err, want, stdout)
	}
	if stderr != "" || strings.Contains(stdout, "Usage:") {
		t.Errorf("stderr = %q, stdout = %q; want no usage and nothing on stderr", stderr, stdout)
	}
	lines := strings.Split(stdout, "\n")
	if len(lines) < 6 {
		t.Fatalf("stdout = %q; want the six header lines", stdout)
	}
	logPath := gateCLILogLine(t, fx.stateDir, lines[5])
	want := "GATE: fail\nREASON: test_failed\nMERGED: " + merge + " (origin/main into " + pre + ")\nHEAD: " + merge + "\nBASE: " + tip +
		"\nLOG: " + logPath + "\n\nlast 20 lines of ./scripts/run-test.sh:\n" + gateCLIStubTail("test") +
		"\nthe merge commit " + merge + " was undone (git reset --keep " + pre + "), so only the reflog has it now\n"
	if stdout != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", stdout, want)
	}
	if head := gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD"); head != pre {
		t.Errorf("HEAD = %s; want %s, the merge commit undone", head, pre)
	}
	if got := gateCLIGitOut(t, fx.origin, "rev-parse", "refs/heads/feat/a"); got != pre {
		t.Errorf("origin's feat/a = %s; want %s, nothing pushed", got, pre)
	}
	if events := gateCLIEvents(t, fx.stateDir); len(events) != 1 || events[0].Details != regateCLIRecord(org.GateReasonTestFailed, merge, tip, logPath) {
		t.Fatalf("gate events = %+v; want one gate_failed of the merge commit %s via regate", events, merge)
	}
}

// D5 and D4 through the CLI: when origin refuses the push of a merge commit
// whose gate_passed was recorded, regate keeps the record and the merge
// commit, prints GATE: pass, MERGED:, HEAD:, BASE: and LOG: with no PUSHED:,
// and exits 1 with the push's error. Run again once origin takes pushes, it
// runs and records nothing, pushes the merge commit, prints exactly
// PUSH_ONLY:, HEAD:, BASE:, LOG: and PUSHED: (no MERGED: line, since the
// push-only path made no merge), and exits 0.
func TestOrgRegate_AFailedPushIsPushedOnlyNextTime(t *testing.T) {
	fx := newOrgRegateFixture(t)
	pre := gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD")
	tip := fx.moveMain(t)
	hook := filepath.Join(fx.origin, "hooks", "pre-receive")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\necho 'pushes are closed' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := fx.regate(t)
	merge := gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD")
	if want := "org: regate: git push origin " + merge + ":refs/heads/feat/a in " + fx.worktree + " failed (exit 1)"; err == nil ||
		!strings.HasPrefix(err.Error(), want) || !strings.Contains(err.Error(), "pushes are closed") {
		t.Fatalf("regate = %v; want an error starting with %q that quotes the hook (stdout: %s)", err, want, stdout)
	}
	if stderr != "" {
		t.Errorf("stderr = %q; want nothing", stderr)
	}
	lines := strings.Split(stdout, "\n")
	if len(lines) != 6 {
		t.Fatalf("stdout = %q; want five lines", stdout)
	}
	logPath := gateCLILogLine(t, fx.stateDir, lines[4])
	want := "GATE: pass\nMERGED: " + merge + " (origin/main into " + pre + ")\nHEAD: " + merge + "\nBASE: " + tip + "\nLOG: " + logPath + "\n"
	if stdout != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", stdout, want)
	}
	if got := gateCLIGitOut(t, fx.origin, "rev-parse", "refs/heads/feat/a"); got != pre {
		t.Fatalf("origin's feat/a = %s; want %s, the push refused", got, pre)
	}

	if err := os.Remove(hook); err != nil {
		t.Fatal(err)
	}
	before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))
	stdout, stderr, err = fx.regate(t)
	if err != nil || stderr != "" {
		t.Fatalf("regate again: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	want = "PUSH_ONLY: the latest gate record of feat/a is a gate_passed of HEAD on origin/main's tip, and origin's feat/a is behind " +
		"HEAD: nothing was run or recorded\nHEAD: " + merge + "\nBASE: " + tip + "\nLOG: " + logPath + "\nPUSHED: " + merge +
		" to origin's feat/a\n"
	if stdout != want {
		t.Fatalf("stdout of the second regate =\n%s\nwant\n%s", stdout, want)
	}
	if got := gateCLIGitOut(t, fx.origin, "rev-parse", "refs/heads/feat/a"); got != merge {
		t.Errorf("origin's feat/a = %s; want the merge commit %s pushed", got, merge)
	}
	if after := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); after != before {
		t.Errorf("a push-only regate changed the manifest:\nbefore %s\nafter  %s", before, after)
	}
	if calls := fx.calls(t); len(calls) != 2 {
		t.Errorf("scripts run = %v; want none run by the second regate", calls)
	}
}

// AC1 through the CLI: a refusal (here, the branch's org still runs, or no
// binding names the branch) prints nothing on stdout or stderr (the refusal
// is the error, which main prints on stderr), exits 1, runs no script, and
// leaves the manifest, the worktree, and origin as they were.
func TestOrgRegate_RefusalPrintsNothingAndChangesNothing(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, fx *orgRegateFixture) []string // the args after `regate`
		want  string
	}{
		{"the org runs again", func(t *testing.T, fx *orgRegateFixture) []string {
			seedGateOrg(t, fx.stateDir, "a", fx.worktree)
			return []string{"--branch", "feat/a", "--state-dir", fx.stateDir}
		}, `org: regate: the branch feat/a is bound to org_id "a", which is running`},
		{"no binding names the branch", func(_ *testing.T, fx *orgRegateFixture) []string {
			return []string{"--branch", "feat/b", "--state-dir", fx.stateDir}
		}, "org: regate: no complete feature binding in the ledger names the branch feat/b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newOrgRegateFixture(t)
			fx.moveMain(t)
			args := tc.setup(t, fx)
			manifest := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))
			head, origin := gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD"), gateCLIGitOut(t, fx.origin, "for-each-ref")

			stdout, stderr, err := runOrgCmdStreams(t, append([]string{"regate"}, args...)...)
			if err == nil || !strings.HasPrefix(err.Error(), tc.want) {
				t.Fatalf("regate = %v; want a refusal starting %q (stdout: %s)", err, tc.want, stdout)
			}
			if stdout != "" || stderr != "" {
				t.Errorf("stdout = %q, stderr = %q; want nothing", stdout, stderr)
			}
			if after := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); after != manifest {
				t.Errorf("a refused regate changed the manifest:\nbefore %s\nafter  %s", manifest, after)
			}
			if gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD") != head || gateCLIGitOut(t, fx.origin, "for-each-ref") != origin {
				t.Errorf("a refused regate changed the worktree's HEAD or origin")
			}
			if calls := fx.calls(t); calls != nil {
				t.Errorf("a refused regate ran %v", calls)
			}
		})
	}
}

// AC1's flags: --org-id is refused (regate reads the org from the branch,
// D1), --branch is required and may not be blank, and regate takes no
// argument; each is refused before the ledger is read, with nothing printed
// or written: the state dir --state-dir names is not even made.
func TestOrgRegate_OrgIDRefusedBranchRequiredNoArguments(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"--org-id", []string{"regate", "--branch", "feat/a", "--org-id", "a"},
			"org: regate: --org-id does not apply: regate reads the org from the binding of the branch --branch names"},
		{"--org-id before regate", []string{"--org-id", "a", "regate", "--branch", "feat/a"},
			"org: regate: --org-id does not apply: regate reads the org from the binding of the branch --branch names"},
		{"no --branch", []string{"regate"}, "org: regate: --branch is required: name the feature branch to regate " +
			"(ralph org merge-order lists the ledger's branches)"},
		{"a blank --branch", []string{"regate", "--branch", " "}, "org: regate: --branch is required"},
		{"an argument", []string{"regate", "--branch", "feat/a", "extra"}, `unknown command "extra" for "ralph org regate"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := filepath.Join(t.TempDir(), "state")
			stdout, stderr, err := runOrgCmdStreams(t, append(tc.args, "--state-dir", stateDir)...)
			if err == nil || !strings.HasPrefix(err.Error(), tc.want) {
				t.Fatalf("regate %v = %v; want an error starting %q", tc.args, err, tc.want)
			}
			if stdout != "" || stderr != "" {
				t.Errorf("stdout = %q, stderr = %q; want nothing", stdout, stderr)
			}
			if _, err := os.Stat(stateDir); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("a refused regate made the state dir %s (stat: %v)", stateDir, err)
			}
		})
	}
}

// printRegateResult on every outcome, from constructed results: the exact
// stdout and the error (nil for exit 0), so a missing or extra line, or the
// wrong exit, shows.
func TestOrgRegate_PrintRegateResult(t *testing.T) {
	const (
		upToDate = "UP_TO_DATE: the latest gate record of feat/a is a gate_passed of HEAD, origin's tip of the branch, on origin/main's " +
			"tip: nothing was run, recorded, or pushed\n"
		pushOnly = "PUSH_ONLY: the latest gate record of feat/a is a gate_passed of HEAD on origin/main's tip, and origin's feat/a is " +
			"behind HEAD: nothing was run or recorded\n"
		pushFailed = "org: regate: git push origin m:refs/heads/feat/a in /wt failed (exit 1), so origin's feat/a is not m: ..."
		notUndone  = "the merge commit m was not undone, since HEAD moved off it to x while the scripts ran (head_moved), and git reset " +
			"--keep would drop what came after it: check the worktree"
	)
	base := org.RegateResult{OrgID: "a", Worktree: "/wt", Branch: "feat/a", Head: "h", Base: "b", BaseRef: "refs/remotes/origin/main", LogPath: "/l"}
	with := func(f func(r *org.RegateResult)) org.RegateResult {
		r := base
		f(&r)
		return r
	}
	merged := func(f func(r *org.RegateResult)) org.RegateResult {
		return with(func(r *org.RegateResult) {
			r.Outcome, r.Head, r.Merge, r.PreMergeHead = org.RegateOutcomeMerged, "m", "m", "p"
			f(r)
		})
	}
	errInterrupted := errors.New("org: regate: interrupted (terminated signal received) before m was pushed to origin's feat/a")
	cases := []struct {
		name    string
		res     org.RegateResult
		err     error // what Regate returned with res
		stdout  string
		wantErr string // "" for exit 0
	}{
		{"up_to_date", with(func(r *org.RegateResult) { r.Outcome = org.RegateOutcomeUpToDate }), nil,
			upToDate + "HEAD: h\nBASE: b\nLOG: /l\n", ""},
		{"up_to_date with no log", with(func(r *org.RegateResult) { r.Outcome, r.LogPath = org.RegateOutcomeUpToDate, "" }), nil,
			upToDate + "HEAD: h\nBASE: b\n", ""},
		{"gated: a pass", with(func(r *org.RegateResult) { r.Outcome, r.Passed = org.RegateOutcomeGated, true }), nil,
			"GATE: pass\nHEAD: h\nBASE: b\nLOG: /l\n", ""},
		{"merged: a pass, pushed", merged(func(r *org.RegateResult) { r.Passed, r.Pushed = true, true }), nil,
			"GATE: pass\nMERGED: m (origin/main into p)\nHEAD: m\nBASE: b\nLOG: /l\nPUSHED: m to origin's feat/a\n", ""},
		{"merged: a pass whose push failed", merged(func(r *org.RegateResult) { r.Passed, r.PushFailed = true, pushFailed }), nil,
			"GATE: pass\nMERGED: m (origin/main into p)\nHEAD: m\nBASE: b\nLOG: /l\n", pushFailed},
		{"push_only, pushed", with(func(r *org.RegateResult) { r.Outcome, r.Passed, r.Pushed = org.RegateOutcomePushOnly, true, true }), nil,
			pushOnly + "HEAD: h\nBASE: b\nLOG: /l\nPUSHED: h to origin's feat/a\n", ""},
		{"push_only whose push failed", with(func(r *org.RegateResult) {
			r.Outcome, r.Passed, r.PushFailed = org.RegateOutcomePushOnly, true, pushFailed
		}), nil, pushOnly + "HEAD: h\nBASE: b\nLOG: /l\n", pushFailed},
		{"gated: test_failed", with(func(r *org.RegateResult) {
			r.Outcome, r.Reason, r.Tail = org.RegateOutcomeGated, org.GateReasonTestFailed, []string{"boom"}
		}), nil, "GATE: fail\nREASON: test_failed\nHEAD: h\nBASE: b\nLOG: /l\n\nlast line of ./scripts/run-test.sh:\n  boom\n",
			"org: regate: recorded gate_failed (reason=test_failed); nothing was pushed"},
		{"gated: tree_changed", with(func(r *org.RegateResult) {
			r.Outcome, r.Reason, r.Dirty = org.RegateOutcomeGated, org.GateReasonTreeChanged, []string{"?? x"}
		}), nil, "GATE: fail\nREASON: tree_changed\nHEAD: h\nBASE: b\nLOG: /l\n\nthe scripts left 1 uncommitted change(s) or untracked " +
			"file(s) in the feature worktree:\n  ?? x\nhint: when the scripts write these files, add their paths to .gitignore and commit " +
			"that; then run ralph org regate --branch feat/a again\n",
			"org: regate: recorded gate_failed (reason=tree_changed); nothing was pushed"},
		{"gated: head_moved", with(func(r *org.RegateResult) { r.Outcome, r.Reason = org.RegateOutcomeGated, org.GateReasonHeadMoved }), nil,
			"GATE: fail\nREASON: head_moved\nHEAD: h\nBASE: b\nLOG: /l\n\nHEAD moved while the scripts ran: commit, reset, or switch " +
				"nothing in the feature worktree while regate runs, and run it again\n",
			"org: regate: recorded gate_failed (reason=head_moved); nothing was pushed"},
		{"merged: static_failed, undone", merged(func(r *org.RegateResult) { r.Reason, r.Undone = org.GateReasonStaticFailed, true }), nil,
			"GATE: fail\nREASON: static_failed\nMERGED: m (origin/main into p)\nHEAD: m\nBASE: b\nLOG: /l\n\n./scripts/run-static-verify.sh " +
				"printed nothing\n\nthe merge commit m was undone (git reset --keep p), so only the reflog has it now\n",
			"org: regate: recorded gate_failed (reason=static_failed) of the merge commit m, which was undone; nothing was pushed"},
		{"merged: head_moved, kept", merged(func(r *org.RegateResult) { r.Reason, r.NotUndone = org.GateReasonHeadMoved, notUndone }), nil,
			"GATE: fail\nREASON: head_moved\nMERGED: m (origin/main into p)\nHEAD: m\nBASE: b\nLOG: /l\n\nHEAD moved while the scripts " +
				"ran: commit, reset, or switch nothing in the feature worktree while regate runs, and run it again\n\n" + notUndone + "\n",
			"org: regate: recorded gate_failed (reason=head_moved) of the merge commit m, which was not undone; nothing was pushed"},
		{"a refusal before the scripts", org.RegateResult{Refused: "org: regate: no branch named"}, nil, "", "org: regate: no branch named"},
		{"a refusal at the record, after a merge", merged(func(r *org.RegateResult) {
			r.Outcome, r.Refused, r.Undone = "", "org: regate: nothing was recorded, since ...; the merge commit m was undone", true
		}), nil, "", "org: regate: nothing was recorded, since ...; the merge commit m was undone; the scripts' output is in /l"},
		{"an interrupt after the record", merged(func(r *org.RegateResult) { r.Passed = true }), errInterrupted,
			"GATE: pass\nMERGED: m (origin/main into p)\nHEAD: m\nBASE: b\nLOG: /l\n", errInterrupted.Error()},
		{"an error after the push", merged(func(r *org.RegateResult) { r.Passed, r.Pushed = true, true }), errors.New("org: gate: git update-ref"),
			"GATE: pass\nMERGED: m (origin/main into p)\nHEAD: m\nBASE: b\nLOG: /l\nPUSHED: m to origin's feat/a\n", "org: gate: git update-ref"},
		{"an error before the record", org.RegateResult{}, errors.New("org: regate: interrupted"), "", "org: regate: interrupted"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := printRegateResult(&out, tc.res, tc.err)
			switch {
			case tc.wantErr == "" && err != nil:
				t.Errorf("printRegateResult = %v; want nil (exit 0)", err)
			case tc.wantErr != "" && (err == nil || err.Error() != tc.wantErr):
				t.Errorf("printRegateResult = %v; want %q", err, tc.wantErr)
			}
			if out.String() != tc.stdout {
				t.Errorf("stdout =\n%s\nwant\n%s", out.String(), tc.stdout)
			}
		})
	}
}

// AC6 through the CLI for an interrupt: SIGINT, SIGTERM, or SIGHUP to the
// ralph process while a script runs on the merge commit stops the script,
// records nothing, undoes the merge commit, prints nothing on stdout, and
// exits 1 with the org layer's error, which names the signal, the log, and
// the undone merge commit.
func TestOrgRegate_InterruptUndoesTheMergeAndRecordsNothing(t *testing.T) {
	cases := []struct {
		name  string
		sig   syscall.Signal
		cause string
	}{
		{"SIGINT", syscall.SIGINT, "interrupt signal received"},
		{"SIGTERM", syscall.SIGTERM, "terminated signal received"},
		{"SIGHUP", syscall.SIGHUP, "hangup signal received"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newOrgRegateFixture(t)
			pre := gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD")
			fx.moveMain(t)
			fx.hook(t, "static", `echo $$ > "$GATE_TEST_OUT/pid"`+"\n"+`touch "$GATE_TEST_OUT/started"`+"\nsleep 30\n")
			before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))
			done := make(chan gateCLIResult, 1)
			go func() {
				stdout, stderr, err := runOrgCmdStreamsContext(context.Background(), "regate", "--branch", "feat/a", "--state-dir", fx.stateDir)
				done <- gateCLIResult{stdout, stderr, err}
			}()
			waitForGateCLIFile(t, filepath.Join(fx.out, "started"))
			pid, err := strconv.Atoi(strings.TrimSpace(fileBytesOrAbsent(t, filepath.Join(fx.out, "pid"))))
			if err != nil {
				t.Fatalf("read the static script's pid: %v", err)
			}
			t.Cleanup(func() {
				if t.Failed() {
					_ = syscall.Kill(-pid, syscall.SIGKILL)
				}
			})

			signalSelf(tc.sig)(t, nil)
			var r gateCLIResult
			select {
			case r = <-done:
			case <-time.After(15 * time.Second):
				t.Fatal("regate did not return within 15s of the interrupt")
			}
			merge := gateCLIGitOut(t, fx.worktree, "rev-parse", "feat/a@{1}")
			if r.err == nil || !strings.HasPrefix(r.err.Error(), "org: regate: interrupted ("+tc.cause+"): the scripts were stopped and nothing "+
				"was recorded; the output so far is in "+filepath.Join(fx.stateDir, "gates")+string(filepath.Separator)) ||
				!strings.HasSuffix(r.err.Error(), "; the merge commit "+merge+" was undone (git reset --keep "+pre+"), so only the reflog has it now") {
				t.Fatalf("regate = %v; want the interrupt error with the cause %q, the log, and the undone merge commit %s", r.err, tc.cause, merge)
			}
			if r.stdout != "" || r.stderr != "" {
				t.Errorf("stdout = %q, stderr = %q; want nothing", r.stdout, r.stderr)
			}
			if after := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); after != before {
				t.Errorf("an interrupted regate changed the manifest:\nbefore %s\nafter  %s", before, after)
			}
			if head := gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD"); head != pre {
				t.Errorf("HEAD = %s; want %s, the merge commit undone", head, pre)
			}
		})
	}
}

// regateCLIStderr is the stderr of a command run in a goroutine: a buffer
// behind a mutex, and wrote, closed on the first write, so that a test can
// wait for a line before the command returns.
type regateCLIStderr struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	once  sync.Once
	wrote chan struct{}
}

func (w *regateCLIStderr) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.once.Do(func() { close(w.wrote) })
	return w.buf.Write(p)
}

func (w *regateCLIStderr) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// AC6 through the CLI for an interrupt while the merge runs: SIGINT to the
// ralph process while git merge waits in its pre-merge-commit hook does not
// stop the merge, and stderr gets one line that says regate waits for it.
// Once the hook lets the merge finish, regate undoes the merge commit,
// records nothing, prints nothing on stdout, and exits 1; the undo that
// starts after the interrupt adds no line.
func TestOrgRegate_AnInterruptDuringTheMergeSaysOnStderrThatItWaits(t *testing.T) {
	fx := newOrgRegateFixture(t)
	pre := gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD")
	fx.moveMain(t)
	started, release, hooks := filepath.Join(fx.out, "started"), filepath.Join(fx.out, "release"), filepath.Join(fx.out, "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	hook := "#!/bin/sh\ntouch '" + started + "'\nwhile [ ! -f '" + release + "' ]; do sleep 0.05; done\n"
	if err := os.WriteFile(filepath.Join(hooks, "pre-merge-commit"), []byte(hook), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.WriteFile(release, nil, 0o644) })
	runLedgerTestGit(t, "-C", fx.main, "config", "core.hooksPath", hooks)
	before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))

	stderr := &regateCLIStderr{wrote: make(chan struct{})}
	done := make(chan gateCLIResult, 1)
	go func() {
		root := NewRootCmd()
		var stdout bytes.Buffer
		root.SetOut(&stdout)
		root.SetErr(stderr)
		root.SetArgs([]string{"org", "regate", "--branch", "feat/a", "--state-dir", fx.stateDir})
		err := root.ExecuteContext(context.Background())
		done <- gateCLIResult{stdout.String(), stderr.String(), err}
	}()
	waitForGateCLIFile(t, started)
	signalSelf(syscall.SIGINT)(t, nil)
	select {
	case <-stderr.wrote:
	case r := <-done:
		t.Fatalf("regate returned %+v while its merge waited for the hook; want the merge left to finish", r)
	case <-time.After(10 * time.Second):
		t.Fatal("stderr got nothing within 10s of the interrupt")
	}
	if err := os.WriteFile(release, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	var r gateCLIResult
	select {
	case r = <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("regate did not return within 15s of the release of the merge")
	}

	want := "org: regate: interrupted (interrupt signal received): waiting for git merge --no-ff in " + fx.worktree + " to finish " +
		"(at most 2m0s) before it stops, since a git stopped half-way would leave the worktree in neither state\n"
	if r.stderr != want {
		t.Errorf("stderr = %q\nwant      %q", r.stderr, want)
	}
	merge := gateCLIGitOut(t, fx.worktree, "rev-parse", "feat/a@{1}")
	if want := "org: regate: interrupted (interrupt signal received) before the scripts ran: nothing was recorded; the merge commit " +
		merge + " was undone (git reset --keep " + pre + "), so only the reflog has it now"; r.err == nil || r.err.Error() != want {
		t.Fatalf("regate = %v\nwant     %s", r.err, want)
	}
	if r.stdout != "" {
		t.Errorf("stdout = %q; want nothing", r.stdout)
	}
	if after := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); after != before {
		t.Errorf("an interrupted regate changed the manifest:\nbefore %s\nafter  %s", before, after)
	}
	if head := gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD"); head != pre {
		t.Errorf("HEAD = %s; want %s, the merge commit undone", head, pre)
	}
}

// regate writes the manifest, so it is refused while seats of an older,
// per-worktree ledger are active (guardLegacyOrgStateDir), before it reads
// the ledger or runs anything: run in the feature worktree with no
// --state-dir, where the ledger shared by every worktree (the main
// checkout's) binds feat/a to the disbanded org a and origin's main moved on
// in origin itself, regate fetches nothing, merges nothing, runs no script,
// prints nothing, and leaves both ledgers, the worktree, and origin as they
// were.
func TestOrgRegate_LegacyLedgerGuard(t *testing.T) {
	setupOrgStubPATH(t)
	fx := newOrgRegateFixture(t)
	shared := filepath.Join(fx.main, ".harness", "state", "org")
	legacy := filepath.Join(fx.worktree, ".harness", "state", "org")
	seedGateOrg(t, shared, "a", fx.worktree)
	appendSeedEvent(t, shared, org.ManifestEvent{OrgID: "a", Event: org.EventDisbanded})
	spawnLedgerSeat(t, legacy, "seat-1")
	tip := gateCLIGitOut(t, fx.origin, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit-tree", "-p", "refs/heads/main",
		"-m", "pushed", "refs/heads/main^{tree}")
	runLedgerTestGit(t, "-C", fx.origin, "update-ref", "refs/heads/main", tip)
	sharedManifest, legacyManifest := fileBytesOrAbsent(t, org.ManifestPathIn(shared)), fileBytesOrAbsent(t, org.ManifestPathIn(legacy))
	head, tracking := gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD"), gateCLIGitOut(t, fx.main, "rev-parse", "refs/remotes/origin/main")
	origin := gateCLIGitOut(t, fx.origin, "for-each-ref")
	t.Chdir(fx.worktree)

	stdout, stderr, err := runOrgCmdStreams(t, "regate", "--branch", "feat/a")
	if want := "org: refusing to change the org ledger: this linked worktree still has an older ledger at " + legacy +
		" with 1 active seat(s)"; err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("regate = %v; want the legacy-ledger refusal starting %q (stdout: %s)", err, want, stdout)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("stdout = %q, stderr = %q; want nothing", stdout, stderr)
	}
	if after := fileBytesOrAbsent(t, org.ManifestPathIn(shared)); after != sharedManifest {
		t.Errorf("a refused regate changed the shared manifest:\nbefore %s\nafter  %s", sharedManifest, after)
	}
	if after := fileBytesOrAbsent(t, org.ManifestPathIn(legacy)); after != legacyManifest {
		t.Errorf("a refused regate changed the older manifest:\nbefore %s\nafter  %s", legacyManifest, after)
	}
	if got := gateCLIGitOut(t, fx.main, "rev-parse", "refs/remotes/origin/main"); got != tracking {
		t.Errorf("refs/remotes/origin/main = %s; want %s, nothing fetched", got, tracking)
	}
	if got := gateCLIGitOut(t, fx.worktree, "rev-parse", "HEAD"); got != head {
		t.Errorf("HEAD = %s; want %s, nothing merged", got, head)
	}
	if st := gateCLIGitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Errorf("the worktree after a refused regate:\n%s", st)
	}
	if got := gateCLIGitOut(t, fx.origin, "for-each-ref"); got != origin {
		t.Errorf("origin's refs after a refused regate:\n%s\nwant them as before:\n%s", got, origin)
	}
	if calls := fx.calls(t); calls != nil {
		t.Errorf("a refused regate ran %v", calls)
	}
	for _, name := range []string{"gates", "regate.lock"} {
		if _, err := os.Stat(filepath.Join(shared, name)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("a refused regate made %s in the shared ledger (stat: %v)", name, err)
		}
	}
}

// regate's help names what it does and refuses, and org --help lists regate
// among the verbs that refuse --org-id.
func TestOrgRegate_Help(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		wants []string
	}{
		{[]string{"regate", "--help"}, []string{
			"ralph org regate brings the branch of a disbanded feature org back to ready in ralph org merge-order",
			"so --org-id is refused", "git merge --no-ff --no-edit -m 'Merge origin/<default branch> <12 hex> into <branch>'",
			// Plan 2026-10-17-org-auto-merge, AC10: merge holds the same lock.
			"First, regate checks that no other ralph org regate or merge runs on the ledger (one runs at a time; it waits up to 1 s for the lock)",
			"It never rebases and never force-pushes", "with via=regate", "GATE: pass, MERGED:, HEAD:, BASE:, LOG: and PUSHED: and exits 0",
			"After static_failed or test_failed it undoes its merge commit", "After head_moved or tree_changed it keeps it",
			"(PUSH_ONLY:, HEAD:, BASE:, LOG: and PUSHED:, exit 0)", "An interrupt (SIGINT, SIGTERM, or SIGHUP)",
			"and its subject is the one regate gave it. When a hook changed that subject, regate refuses and asks you to push HEAD yourself",
			"A merge, its abort, and its undo finish first: they run in a process group of their own, which the Ctrl-C or the " +
				"hangup of the terminal does not reach either, and regate says on stderr that it waits for one (at most 120 s)",
			"When git has no user.name and user.email to make the merge commit with, regate merges nothing, records nothing, and exits 1",
			"it needs no running watch and no approved plan",
			"While it runs, ralph org spawn (not a --dry-run) and ralph org start (either form) of the org it works on are refused",
			"--branch string",
		}},
		{[]string{"--help"}, []string{
			"`ralph org regate` merges the default branch into a disbanded feature org's branch",
			"stop --all, disband --all, merge-order, merge, regate, wait --inbox, watch, approve, the director verbs, and the inbox verbs, which refuse it",
		}},
	} {
		out, err := runOrgCmd(t, tc.args...)
		if err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		help := strings.Join(strings.Fields(out), " ")
		for _, want := range tc.wants {
			if !strings.Contains(help, want) {
				t.Errorf("%v does not say %q:\n%s", tc.args, want, out)
			}
		}
	}
}
