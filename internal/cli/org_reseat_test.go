package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// Tests for `ralph org reseat` through the CLI (plan
// docs/plans/active/2026-11-12-org-reseat.md, S3: the CLI side of AC-3,
// AC-5, AC-9, AC-11, AC-16's help and AC-19; S6: AC-20's signal hold and
// help, AC-21's help, and the refusal with no main worktree). The org layer's
// checks, stops and starts are tested in internal/org/reseat_test.go; these
// tests run the command against the stub herdr/agmsg (setupOrgStubPATH) and
// the repo's real scripts/ralph-worktree.sh, and check what it prints, its
// exit, and what it leaves in the ledger. They chdir and swap package-level
// vars, so none runs in parallel.

// reseatCLIConfig is the ralph.toml of the reseat fixture's main checkout:
// approval mode split, so the implementer spawns with no plan approval, and a
// reviewer role that runs the claude CLI's own default model.
const reseatCLIConfig = "[org.approval]\nmode = \"split\"\n\n[org.role_models]\nreviewer = { driver = \"claude\", model = \"\" }\n"

// reseatCLIOrg is the feature org a of the split plan demo, started with
// `ralph org start --plan` in a temp repo whose main checkout is the working
// directory, with its ledger named by --state-dir.
type reseatCLIOrg struct {
	stateDir string
	herdrLog string
}

// newReseatCLIOrg starts org a (its leader on claude sonnet) and spawns, in
// this order, its implementer (claude opus) and its reviewer (claude, the
// CLI's default model), as its leader would.
func newReseatCLIOrg(t *testing.T) reseatCLIOrg {
	t.Helper()
	herdrLog, _ := setupOrgStubPATH(t)
	root := newFeatureRepo(t, ".claude/worktrees/\n")
	writeTestFile(t, filepath.Join(root, "ralph.toml"), reseatCLIConfig)
	runLedgerTestGit(t, "-C", root, "add", "ralph.toml")
	runLedgerTestGit(t, "-C", root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
		"commit", "--quiet", "-m", "ralph.toml")
	t.Chdir(root)
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(tmp, "state")
	plan := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
	if stdout, stderr, err := startOrgFeature(t, plan, "a", "--state-dir", stateDir); err != nil {
		t.Fatalf("start --plan --feature a: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	wt := filepath.Join(root, ".claude", "worktrees", "org-a")
	for _, seat := range [][]string{
		{"--id", "implementer", "--role", "implementer", "--driver", "claude", "--model", "opus"},
		{"--id", "reviewer", "--role", "reviewer"},
	} {
		args := append([]string{"spawn", "--org-id", "a", "--cwd", wt, "--scope", "internal/a/", "--state-dir", stateDir}, seat...)
		if stdout, stderr, err := runOrgCmdStreams(t, args...); err != nil {
			t.Fatalf("spawn %v: %v (stdout: %s stderr: %s)", seat, err, stdout, stderr)
		}
	}
	return reseatCLIOrg{stateDir: stateDir, herdrLog: herdrLog}
}

// reseat runs `ralph org reseat --org-id a --state-dir <ledger>` with extra.
func (fx reseatCLIOrg) reseat(t *testing.T, extra ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runOrgCmdStreams(t, append([]string{"reseat", "--org-id", "a", "--state-dir", fx.stateDir}, extra...)...)
}

// manifest returns the bytes of the fixture's manifest.
func (fx reseatCLIOrg) manifest(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(org.ManifestPathIn(fx.stateDir))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// stateDirListing is every name in stateDir, with each file's size: what a
// dry run must leave as it is, with the manifest's bytes.
func stateDirListing(t *testing.T, stateDir string) string {
	t.Helper()
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		if e.IsDir() {
			fmt.Fprintf(&b, "%s/\n", e.Name())
			continue
		}
		fmt.Fprintf(&b, "%s %d\n", e.Name(), info.Size())
	}
	return b.String()
}

// assertReseatLines checks that the lines of stdout are want, in order. A
// `pane *` in a wanted line stands for any pane id.
func assertReseatLines(t *testing.T, stdout string, want []string) {
	t.Helper()
	got := outputLines(stdout)
	ok := len(got) == len(want)
	for i := 0; ok && i < len(want); i++ {
		pattern := strings.ReplaceAll(regexp.QuoteMeta(want[i]), `pane \*`, `pane [^ )]+`)
		ok = regexp.MustCompile("^" + pattern + "$").MatchString(got[i])
	}
	if !ok {
		t.Errorf("reseat printed:\n%s\nwant the lines:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// reseatedLine matches the line a reseat that ended well prints, and holds
// its id.
var reseatedLine = regexp.MustCompile(`(?m)^reseated org_id a \(reseat=(\d{8}T\d{6}-[0-9a-f]{4})\)$`)

// reseatIDOf returns the id on stdout's reseated line.
func reseatIDOf(t *testing.T, stdout string) string {
	t.Helper()
	m := reseatedLine.FindStringSubmatch(stdout)
	if m == nil {
		t.Fatalf("no reseated line with an id in:\n%s", stdout)
	}
	return m[1]
}

// lastEvent returns the last event of the fixture's manifest.
func (fx reseatCLIOrg) lastEvent(t *testing.T) org.ManifestEvent {
	t.Helper()
	events := readManifestEvents(t, org.ManifestPathIn(fx.stateDir))
	if len(events) == 0 {
		t.Fatal("the manifest is empty")
	}
	return events[len(events)-1]
}

// TestOrgReseat_StopsAndStartsEverySeat: with no --seat, reseat stops the
// reviewer, the implementer and the leader, in that order, starts the leader
// and then the others in their spawn order, each with the driver and model
// the ledger recorded (the reviewer's "" printed as the CLI default), prints a
// line for each stop and start and the reseated line, and exits 0. It prints
// no DECISION, since the leader started again, and records `reseated` last.
func TestOrgReseat_StopsAndStartsEverySeat(t *testing.T) {
	fx := newReseatCLIOrg(t)

	stdout, stderr, err := fx.reseat(t)
	if err != nil {
		t.Fatalf("reseat: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	id := reseatIDOf(t, stdout)
	assertReseatLines(t, stdout, []string{
		"stopped: reviewer (reviewer, claude CLI default)",
		"stopped: implementer (implementer, claude opus)",
		"stopped: leader (leader, claude sonnet)",
		"started: leader (leader, claude sonnet, pane *)",
		"started: implementer (implementer, claude opus, pane *)",
		"started: reviewer (reviewer, claude CLI default, pane *)",
		"reseated org_id a (reseat=" + id + ")",
	})
	if ev := fx.lastEvent(t); ev.Event != org.EventReseated || ev.OrgID != "a" ||
		ev.Details != "id="+id+" seats=leader:started,implementer:started,reviewer:started" {
		t.Errorf("last event %+v; want the org's reseated with every seat started", ev)
	}
}

// TestOrgReseat_OneSeatPrintsTheDecisionForTheLeader is AC-3 through the CLI:
// --seat implementer with --driver and --model stops and starts the
// implementer alone, on codex, and leaves the leader running. The output ends
// with the line that says reseat sent nothing, which names the read of the
// leader's pane, and the `ralph org send` of the DECISION, both with the
// --state-dir that was given.
func TestOrgReseat_OneSeatPrintsTheDecisionForTheLeader(t *testing.T) {
	fx := newReseatCLIOrg(t)
	leaderBefore := latestSeatEventNamed(t, fx.stateDir, "a", org.LeaderIdentity, org.EventSpawned)

	stdout, stderr, err := fx.reseat(t, "--seat", "implementer", "--driver", "codex", "--model", "gpt-x")
	if err != nil {
		t.Fatalf("reseat --seat implementer: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	id := reseatIDOf(t, stdout)
	decision := org.ReseatDecisionCommand("a", org.ReseatDecisionText(id, []string{"implementer"}), fx.stateDir)
	intro := "reseat sends the leader nothing, and the seats it started have no TASK: check the leader's pane with " +
		"ralph org read --org-id a --seat leader --state-dir " + org.ShellWord(fx.stateDir) + ", then tell the leader with:"
	head, ok := strings.CutSuffix(stdout, intro+"\n"+decision+"\n")
	if !ok {
		t.Fatalf("expected the output to end with the DECISION:\n%s\n%s\ngot:\n%s", intro, decision, stdout)
	}
	assertReseatLines(t, head, []string{
		"stopped: implementer (implementer, claude opus)",
		"started: implementer (implementer, codex gpt-x; the ledger had claude opus, pane *)",
		"reseated org_id a (reseat=" + id + ")",
	})
	if ev := fx.lastEvent(t); ev.Event != org.EventReseated || ev.Details != "id="+id+" seats=implementer:started changed=implementer:codex/gpt-x" {
		t.Errorf("last event %+v; want the reseated of the implementer on codex", ev)
	}
	if got := latestSeatEventNamed(t, fx.stateDir, "a", org.LeaderIdentity, org.EventSpawned); got != leaderBefore {
		t.Errorf("the leader's latest spawned changed: %+v -> %+v; want the leader left running", leaderBefore, got)
	}
	for _, ev := range readManifestEvents(t, org.ManifestPathIn(fx.stateDir)) {
		if ev.SeatID == org.LeaderIdentity && ev.Event == org.EventStopped {
			t.Errorf("the leader was stopped: %+v", ev)
		}
	}
}

// TestOrgReseat_AFailedStartPrintsTheRetry covers a start that fails after
// the leader's: reseat prints the stops, the leader's start, the failed
// start and the seat it left stopped, then the reseat that starts the rest,
// and exits 1 with the org layer's error. The printed reseat, run as it is,
// starts the implementer and the reviewer in the order the org first spawned
// them (FR-6), and ends with the DECISION, since the leader kept running.
func TestOrgReseat_AFailedStartPrintsTheRetry(t *testing.T) {
	fx := newReseatCLIOrg(t)
	t.Setenv("ORG_STUB_FAIL", "pane:split")

	stdout, stderr, err := fx.reseat(t)
	if err == nil || !strings.Contains(err.Error(), `seat "implementer" did not start again, so the reseat started no seat after it`) {
		t.Fatalf("expected the implementer's failed start, got: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	retry := "ralph org reseat --org-id a --seat implementer --seat reviewer --state-dir " + org.ShellWord(fx.stateDir)
	assertReseatLines(t, stdout, []string{
		"stopped: reviewer (reviewer, claude CLI default)",
		"stopped: implementer (implementer, claude opus)",
		"stopped: leader (leader, claude sonnet)",
		"started: leader (leader, claude sonnet, pane *)",
		"start failed: implementer (implementer, claude opus)",
		"left stopped: reviewer (stopped by this reseat, not started again)",
		"to start the seats left, run:",
		retry,
	})

	t.Setenv("ORG_STUB_FAIL", "")
	args := strings.Fields(strings.TrimPrefix(retry, "ralph org "))
	stdout, stderr, err = runOrgCmdStreams(t, args...)
	if err != nil {
		t.Fatalf("the printed retry: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	id := reseatIDOf(t, stdout)
	decision := org.ReseatDecisionCommand("a", org.ReseatDecisionText(id, []string{"implementer", "reviewer"}), fx.stateDir)
	head, ok := strings.CutSuffix(stdout, decision+"\n")
	if !ok {
		t.Fatalf("expected the retry's output to end with %s, got:\n%s", decision, stdout)
	}
	assertReseatLines(t, head, []string{
		"started: implementer (implementer, claude opus, pane *)",
		"started: reviewer (reviewer, claude CLI default, pane *)",
		"reseated org_id a (reseat=" + id + ")",
		"reseat sends the leader nothing, and the seats it started have no TASK: check the leader's pane with " +
			"ralph org read --org-id a --seat leader --state-dir " + org.ShellWord(fx.stateDir) + ", then tell the leader with:",
	})
}

// TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig covers the
// second and third findings of the cross-review (cycle 1) through the CLI: a
// reseat of the implementer and the reviewer, given a relative --config,
// while the leader runs, whose reviewer start is refused after the
// implementer started (another spawn of the reviewer holds its spawn lock).
// reseat prints the stops, the implementer's start and the reviewer's
// failure, then the retry with the --config made absolute and the
// --state-dir given, then the DECISION for the implementer, which started
// with no TASK, and exits 1. The ralph.toml of the working directory now
// holds another approval mode, so a reseat of the reviewer without --config
// is refused and records nothing; the printed retry, run as it is from a
// subdirectory once the lock is free, starts the reviewer and ends with the
// DECISION for the reviewer alone.
func TestOrgReseat_APartialFailurePrintsTheDecisionAndTheConfig(t *testing.T) {
	fx := newReseatCLIOrg(t)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, "reseat.toml", reseatCLIConfig)
	cfg := filepath.Join(wd, "reseat.toml")
	writeTestFile(t, "ralph.toml", "[org.approval]\nmode = \"per-plan\"\n")
	f, err := os.OpenFile(filepath.Join(fx.stateDir, "spawn-a_reviewer.lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := fx.reseat(t, "--seat", "implementer", "--seat", "reviewer", "--config", "reseat.toml")
	if err == nil || !strings.Contains(err.Error(), `seat "reviewer" did not start again, so the reseat started no seat after it`) {
		t.Fatalf("expected the reviewer's refused start, got: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	id, ok := strings.CutPrefix(strings.Fields(fx.lastEvent(t).Details + " ")[0], "id=")
	if !ok {
		t.Fatalf("the last event %+v is not the reseat's reseated", fx.lastEvent(t))
	}
	sd := org.ShellWord(fx.stateDir)
	retry := "ralph org reseat --org-id a --seat reviewer --config " + org.ShellWord(cfg) + " --state-dir " + sd
	intro := "reseat sends the leader nothing, and the seats it started have no TASK: check the leader's pane with " +
		"ralph org read --org-id a --seat leader --state-dir " + sd + ", then tell the leader with:"
	decision := org.ReseatDecisionCommand("a", org.ReseatDecisionText(id, []string{"implementer"}), fx.stateDir)
	head, ok := strings.CutSuffix(stdout, intro+"\n"+decision+"\n")
	if !ok {
		t.Fatalf("expected the output to end with the DECISION for the implementer:\n%s\n%s\ngot:\n%s", intro, decision, stdout)
	}
	assertReseatLines(t, head, []string{
		"stopped: reviewer (reviewer, claude CLI default)",
		"stopped: implementer (implementer, claude opus)",
		"started: implementer (implementer, claude opus, pane *)",
		"start failed: reviewer (reviewer, claude CLI default)",
		"to start the seats left, run:",
		retry,
	})

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	manifest := fx.manifest(t)
	if stdout, _, err := fx.reseat(t, "--seat", "reviewer"); err == nil ||
		!strings.HasPrefix(err.Error(), `org: reseat of org_id "a" is refused before anything is stopped: `) ||
		!strings.Contains(err.Error(), `approval "split") and reserves internal/a/`) ||
		!strings.Contains(err.Error(), `approval "per-plan") is refused`) || stdout != "" {
		t.Errorf("reseat --seat reviewer without --config: %v (stdout: %s); want the refusal of the working directory's approval mode", err, stdout)
	}
	if fx.manifest(t) != manifest {
		t.Error("the refused reseat without --config changed the manifest")
	}

	sub := filepath.Join(wd, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	args := strings.Fields(strings.TrimPrefix(retry, "ralph org "))
	stdout, stderr, err = runOrgCmdStreams(t, args...)
	if err != nil {
		t.Fatalf("the printed retry from %s: %v (stdout: %s stderr: %s)", sub, err, stdout, stderr)
	}
	id = reseatIDOf(t, stdout)
	decision = org.ReseatDecisionCommand("a", org.ReseatDecisionText(id, []string{"reviewer"}), fx.stateDir)
	head, ok = strings.CutSuffix(stdout, intro+"\n"+decision+"\n")
	if !ok {
		t.Fatalf("expected the retry's output to end with the DECISION for the reviewer:\n%s\ngot:\n%s", decision, stdout)
	}
	assertReseatLines(t, head, []string{
		"started: reviewer (reviewer, claude CLI default, pane *)",
		"reseated org_id a (reseat=" + id + ")",
	})
}

// TestOrgReseat_DryRunPrintsThePlanAndStartsNothing is AC-9 and AC-19
// through the CLI: --dry-run prints the seats it would stop and start, in
// order, with their drivers and models, and leaves the ledger as it was. It
// only reads whether the watch and the herdr server run: the watch launcher,
// the herdr server gate, and the start of `herdr server` are never called,
// and a note says what a real reseat would start. A herdr with no
// `status server --json` counts as running, with the gate's warning, and a
// status that cannot be read is the gate's refusal, before anything prints.
func TestOrgReseat_DryRunPrintsThePlanAndStartsNothing(t *testing.T) {
	plan := []string{
		"stop: reviewer (reviewer, claude CLI default)",
		"stop: implementer (implementer, claude opus)",
		"stop: leader (leader, claude sonnet)",
		"start: leader (leader, claude sonnet)",
		"start: implementer (implementer, claude opus)",
		"start: reviewer (reviewer, claude CLI default)",
	}
	for _, tc := range []struct {
		name       string
		watch      bool // a watch that accepts work holds the ledger's watch.lock
		status     herdrStatusAnswer
		wantNotes  []string
		wantStderr string // "" for no warning
		wantErr    string // a refusal's words; "" for none
	}{
		{"neither runs", false, herdrNotRunning, []string{orgReseatStartsWatchNote, orgReseatStartsServerNote}, "", ""},
		{"both run", true, herdrRunning, nil, "", ""},
		{"a herdr with no status --json", true,
			herdrStatusAnswer{err: fmt.Errorf("herdr status server: %w", driver.ErrHerdrStatusJSONUnsupported)}, nil,
			"warning: this herdr has no 'herdr status server --json' (herdr 0.7.5 has it), so ralph org reseat cannot check or start the herdr server", ""},
		{"a status that cannot be read", true, herdrStatusAnswer{err: errors.New("socket refused")}, nil, "",
			`org: reseat of org_id "a" is refused before anything is stopped: reseat: could not read the herdr server's status ` +
				`(socket refused), so nothing was made; start herdr in another terminal with 'herdr', then run ralph org reseat again`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newReseatCLIOrg(t)
			useRealOrgWatchGate(t)
			origLaunch, origGate := orgWatchLaunch, orgHerdrServerGateOverride
			t.Cleanup(func() { orgWatchLaunch, orgHerdrServerGateOverride = origLaunch, origGate })
			var watchLaunches, gateCalls int
			orgWatchLaunch = func(orgWatchLaunchRequest) (*orgWatchProcess, error) {
				watchLaunches++
				return nil, errors.New("a dry run launched the watch")
			}
			orgHerdrServerGateOverride = func() error { gateCalls++; return nil }
			gate := &herdrGateFake{answers: []herdrStatusAnswer{tc.status}}
			setOrgHerdrServerSeams(t, gate)
			if tc.watch {
				holdWatchLock(t, fx.stateDir, runningMark(os.Getpid(), time.Now()))
			}
			listing, manifest, herdrCalls := stateDirListing(t, fx.stateDir), fx.manifest(t), len(readLogLines(t, fx.herdrLog))

			stdout, stderr, err := fx.reseat(t, "--dry-run")
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Errorf("dry run error %v; want %q", err, tc.wantErr)
				}
				if stdout != "" {
					t.Errorf("a refused dry run printed on stdout:\n%s", stdout)
				}
			} else {
				if err != nil {
					t.Fatalf("dry run: %v (stdout: %s stderr: %s)", err, stdout, stderr)
				}
				assertReseatLines(t, stdout, append(slices.Clone(plan), tc.wantNotes...))
			}
			switch {
			case tc.wantStderr == "" && strings.Contains(stderr, "warning:"):
				t.Errorf("stderr %q; want no warning", stderr)
			case !strings.Contains(stderr, tc.wantStderr):
				t.Errorf("stderr %q; want %q", stderr, tc.wantStderr)
			}
			if got := stateDirListing(t, fx.stateDir); got != listing {
				t.Errorf("the dry run changed the state dir:\n%s\n->\n%s", listing, got)
			}
			if fx.manifest(t) != manifest {
				t.Error("the dry run changed the manifest")
			}
			if got := len(readLogLines(t, fx.herdrLog)); got != herdrCalls {
				t.Errorf("the dry run called herdr %d times; want none", got-herdrCalls)
			}
			if watchLaunches != 0 || gateCalls != 0 || len(gate.launches) != 0 || gate.lookups != 0 {
				t.Errorf("the dry run launched the watch %d times, ran the herdr server gate %d times, started herdr %d times "+
					"and looked it up %d times; want none", watchLaunches, gateCalls, len(gate.launches), gate.lookups)
			}
			if gate.reads != 1 {
				t.Errorf("the dry run read the herdr server's status %d times; want once", gate.reads)
			}
		})
	}
}

// newReseatPlainOrg starts the org plain with `ralph org start <task>` (no
// split plan, so no binding) in a ledger named by --state-dir, with a temp
// git repository as the working directory, and returns the ledger.
func newReseatPlainOrg(t *testing.T) string {
	t.Helper()
	setupOrgStubPATH(t)
	isolateFeatureGitEnv(t)
	repo, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runLedgerTestGit(t, "init", "--quiet", "-b", "main", repo)
	t.Chdir(repo)
	stateDir := filepath.Join(t.TempDir(), "state")
	if _, stderr, err := startOrgLeader(t, "plain", "--state-dir", stateDir, "--scope", "docs/"); err != nil {
		t.Fatalf("start plain: %v (stderr: %s)", err, stderr)
	}
	return stateDir
}

// TestOrgReseat_OrgWithoutBindingRefused is AC-5 through the CLI: an org
// started without --plan is refused with the stop and spawn commands (with
// the --state-dir that was given), nothing on stdout and nothing recorded.
func TestOrgReseat_OrgWithoutBindingRefused(t *testing.T) {
	stateDir := newReseatPlainOrg(t)
	manifest := readManifestEvents(t, org.ManifestPathIn(stateDir))

	stdout, _, err := runOrgCmdStreams(t, "reseat", "--org-id", "plain", "--state-dir", stateDir)
	sd := org.ShellWord(stateDir)
	for _, want := range []string{
		`org: reseat of org_id "plain" is refused before anything is stopped: org_id "plain" has no split plan binding since ` +
			`its last disband (it was not started with ralph org start --plan)`,
		"stop a seat with ralph org stop --org-id plain --seat <seat> --state-dir " + sd,
		"start it again with ralph org spawn --org-id plain --id <seat> --role <role> --state-dir " + sd,
	} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("reseat error %v; want it to contain %q", err, want)
		}
	}
	if stdout != "" {
		t.Errorf("a refused reseat printed on stdout:\n%s", stdout)
	}
	if got := readManifestEvents(t, org.ManifestPathIn(stateDir)); !slices.Equal(got, manifest) {
		t.Errorf("a refused reseat recorded %v", eventTypes(got[len(manifest):]))
	}
}

// TestOrgReseat_FlagRefusals is AC-11 and FR-1's flag half through the CLI:
// --driver or --model without exactly one --seat, one of them without the
// other, a seat named twice, no --org-id, and an argument are refused with
// exit 1, nothing on stdout and nothing recorded.
func TestOrgReseat_FlagRefusals(t *testing.T) {
	stateDir := newReseatPlainOrg(t)
	manifest := readManifestEvents(t, org.ManifestPathIn(stateDir))
	const refused = `org: reseat of org_id "plain" is refused before anything is stopped: `
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"--driver alone", []string{"--org-id", "plain", "--seat", "leader", "--driver", "codex"},
			refused + "--driver and --model go together: pass both with --seat leader"},
		{"--model with two --seat", []string{"--org-id", "plain", "--seat", "leader", "--seat", "implementer", "--model", "opus"},
			refused + "--driver and --model start one seat on another driver or model, so they need exactly one --seat (2 given)"},
		{"--driver and --model with no --seat", []string{"--org-id", "plain", "--driver", "codex", "--model", "gpt-x"},
			refused + "--driver and --model start one seat on another driver or model, so they need exactly one --seat (0 given)"},
		{"a seat named twice", []string{"--org-id", "plain", "--seat", "leader", "--seat", "leader"},
			refused + "--seat leader is given twice: name each seat once"},
		{"no --org-id", []string{"--seat", "leader"}, "org: --org-id is required"},
		{"an argument", []string{"--org-id", "plain", "leader"}, `unknown command "leader" for "ralph org reseat"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, _, err := runOrgCmdStreams(t, append(append([]string{"reseat"}, tc.args...), "--state-dir", stateDir)...)
			if err == nil || !strings.HasPrefix(err.Error(), tc.want) {
				t.Errorf("reseat %v: error %v; want one that starts with %q", tc.args, err, tc.want)
			}
			if stdout != "" {
				t.Errorf("a refused reseat printed on stdout:\n%s", stdout)
			}
			if got := readManifestEvents(t, org.ManifestPathIn(stateDir)); !slices.Equal(got, manifest) {
				t.Errorf("a refused reseat recorded %v", eventTypes(got[len(manifest):]))
			}
		})
	}
}

// countReseated is the number of `reseated` events in the ledger stateDir.
func countReseated(t *testing.T, stateDir string) int {
	t.Helper()
	n := 0
	for _, ev := range readManifestEvents(t, org.ManifestPathIn(stateDir)) {
		if ev.Event == org.EventReseated {
			n++
		}
	}
	return n
}

// TestOrgReseat_ARealReseatRunsThroughTheSignalHold covers P9's CLI half
// (AC-20): a real reseat runs, with the print of its result, inside
// orgReseatThroughSignals, which catches SIGINT, SIGTERM, SIGHUP and
// SIGPIPE, so its `reseated` is recorded and its lines are printed before
// the hold ends; a dry run, which stops nothing, does not go through it. The
// hold itself is TestOrgReseatHoldSignals_KeepsTheProcessAndSaysSoOnce's and
// TestOrgReseatHoldSignals_AClosedStderrDoesNotEndIt's.
func TestOrgReseat_ARealReseatRunsThroughTheSignalHold(t *testing.T) {
	fx := newReseatCLIOrg(t)
	setOrgHerdrServerSeams(t, &herdrGateFake{answers: []herdrStatusAnswer{herdrRunning}}) // what the dry run reads
	orig := orgReseatThroughSignals
	t.Cleanup(func() { orgReseatThroughSignals = orig })
	var holds []string // the reseated count before and after each call
	orgReseatThroughSignals = func(_ io.Writer, call func()) {
		before := countReseated(t, fx.stateDir)
		call()
		holds = append(holds, fmt.Sprintf("%d->%d", before, countReseated(t, fx.stateDir)))
	}

	if stdout, stderr, err := fx.reseat(t, "--dry-run"); err != nil || !strings.HasPrefix(stdout, "stop: reviewer") {
		t.Fatalf("dry run: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if len(holds) != 0 {
		t.Errorf("a dry run went through the signal hold %d times; want none", len(holds))
	}
	stdout, stderr, err := fx.reseat(t)
	if err != nil {
		t.Fatalf("reseat: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	reseatIDOf(t, stdout)
	if !slices.Equal(holds, []string{"0->1"}) {
		t.Errorf("signal holds %q; want one, inside which the reseat recorded its reseated", holds)
	}
}

// reseatHoldChildEnv selects the child side of
// TestOrgReseatHoldSignals_KeepsTheProcessAndSaysSoOnce.
const reseatHoldChildEnv = "RALPH_TEST_RESEAT_HOLD_CHILD"

// TestOrgReseatHoldSignals_KeepsTheProcessAndSaysSoOnce covers the hold a
// real reseat runs in (P9): while call runs, SIGINT, SIGTERM and SIGHUP do
// not end the process, the first one prints one line on stderr, and call
// returns as it would. The signals go to a child copy of the test binary,
// because a signal nothing catches ends the process that gets it: the child
// sends itself SIGINT from inside call, waits for the line, then SIGTERM and
// SIGHUP 100ms apart, and prints what stderr got once call has returned.
func TestOrgReseatHoldSignals_KeepsTheProcessAndSaysSoOnce(t *testing.T) {
	if os.Getenv(reseatHoldChildEnv) == "1" {
		stderr := &regateCLIStderr{wrote: make(chan struct{})}
		orgReseatHoldSignals(stderr, func() {
			if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
				t.Fatalf("send SIGINT to the child: %v", err)
			}
			select {
			case <-stderr.wrote:
			case <-time.After(5 * time.Second):
				t.Fatal("no line on stderr within 5s of SIGINT")
			}
			// The pauses let the hold do whatever it does after a signal
			// before the next one comes, as a second Ctrl-C would.
			for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGHUP} {
				time.Sleep(100 * time.Millisecond)
				if err := syscall.Kill(os.Getpid(), sig); err != nil {
					t.Fatalf("send %v to the child: %v", sig, err)
				}
			}
			time.Sleep(200 * time.Millisecond)
		})
		fmt.Printf("call returned; stderr: %q\n", stderr.String())
		return
	}

	// A channel of the test's own for SIGINT and SIGHUP, so that the child
	// starts with them at their default action even when this test binary
	// started with them ignored (nohup, or a shell's &), as ralph would.
	caught := make(chan os.Signal, 1)
	signal.Notify(caught, syscall.SIGINT, syscall.SIGHUP)
	t.Cleanup(func() { signal.Stop(caught) })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestOrgReseatHoldSignals_KeepsTheProcessAndSaysSoOnce$", "-test.count=1")
	child.Env = append(os.Environ(), reseatHoldChildEnv+"=1")
	out, err := child.CombinedOutput()
	if err != nil {
		t.Fatalf("the child that got SIGINT, SIGTERM and SIGHUP failed or was ended by a signal: %v\n%s", err, out)
	}
	want := fmt.Sprintf("call returned; stderr: %q\n", "note: interrupt signal received: "+orgReseatSignalNote+"\n")
	if !strings.Contains(string(out), want) {
		t.Errorf("the child printed:\n%s\nwant the line %q", out, want)
	}
}

// reseatHoldPipeChildEnv selects the child side of
// TestOrgReseatHoldSignals_AClosedStderrDoesNotEndIt.
const reseatHoldPipeChildEnv = "RALPH_TEST_RESEAT_HOLD_PIPE_CHILD"

// reseatHoldPipeResult is the exit code the child of
// TestOrgReseatHoldSignals_AClosedStderrDoesNotEndIt ends with: the result
// its call gives, as runOrgReseat ends with Reseat's.
const reseatHoldPipeResult = 3

// reseatPipeStderr is the stderr the child of
// TestOrgReseatHoldSignals_AClosedStderrDoesNotEndIt gives the hold: it keeps
// what it is given, writes it to os.Stderr (fd 2), and sends that write's
// error on wrote.
type reseatPipeStderr struct {
	mu    sync.Mutex
	given strings.Builder
	wrote chan error
}

func (w *reseatPipeStderr) Write(p []byte) (int, error) {
	w.mu.Lock()
	w.given.Write(p)
	w.mu.Unlock()
	n, err := os.Stderr.Write(p)
	select {
	case w.wrote <- err:
	default:
	}
	return n, err
}

func (w *reseatPipeStderr) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.given.String()
}

// TestOrgReseatHoldSignals_AClosedStderrDoesNotEndIt covers the hold a real
// reseat runs in when nobody reads its stderr any more (a `2>&1 | tee` whose
// tee a Ctrl-C ended): the note the hold writes at SIGINT and the output call
// writes to stderr fail with EPIPE instead of ending the process by SIGPIPE,
// so call returns and the process ends with the result call gave. A child
// copy of the test binary runs with its stderr, fd 2, on a pipe whose read
// end the test closed before the start (Go ends a process at a broken pipe
// on fd 1 or 2 unless SIGPIPE is notified). From inside call the child sends
// itself SIGINT, waits for the note's write, writes a line of its own to
// stderr, and sets the result; once the hold has returned it prints on
// stdout what the two writes returned and exits with that result.
func TestOrgReseatHoldSignals_AClosedStderrDoesNotEndIt(t *testing.T) {
	if os.Getenv(reseatHoldPipeChildEnv) == "1" {
		stderr := &reseatPipeStderr{wrote: make(chan error, 1)}
		var noteErr, outErr error
		result := 0
		orgReseatHoldSignals(stderr, func() {
			if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
				t.Fatalf("send SIGINT to the child: %v", err)
			}
			select {
			case noteErr = <-stderr.wrote:
			case <-time.After(5 * time.Second):
				t.Fatal("no write on stderr within 5s of SIGINT")
			}
			_, outErr = fmt.Fprintln(os.Stderr, "stopped: reviewer (reviewer, claude CLI default)")
			result = reseatHoldPipeResult
		})
		fmt.Printf("call returned; note %q: EPIPE %v; output: EPIPE %v\n", stderr.String(),
			errors.Is(noteErr, syscall.EPIPE), errors.Is(outErr, syscall.EPIPE))
		os.Exit(result)
	}

	// As in TestOrgReseatHoldSignals_KeepsTheProcessAndSaysSoOnce: the child
	// starts with SIGINT and SIGHUP at their default action.
	caught := make(chan os.Signal, 1)
	signal.Notify(caught, syscall.SIGINT, syscall.SIGHUP)
	t.Cleanup(func() { signal.Stop(caught) })
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestOrgReseatHoldSignals_AClosedStderrDoesNotEndIt$", "-test.count=1")
	child.Env = append(os.Environ(), reseatHoldPipeChildEnv+"=1")
	var stdout bytes.Buffer
	child.Stdout, child.Stderr = &stdout, w
	err = child.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != reseatHoldPipeResult {
		t.Fatalf("the child whose stderr nobody reads ended with %v; want exit %d, the result its call gave, not a signal\n%s",
			err, reseatHoldPipeResult, stdout.String())
	}
	want := fmt.Sprintf("call returned; note %q: EPIPE true; output: EPIPE true\n",
		"note: interrupt signal received: "+orgReseatSignalNote+"\n")
	if !strings.Contains(stdout.String(), want) {
		t.Errorf("the child printed:\n%s\nwant the line %q", stdout.String(), want)
	}
}

// TestOrgReseat_NoMainWorktreeIsAReseatRefusal covers the refusal of a
// reseat run where org.FeatureRepoRoot finds no main worktree (the ledger
// named with --state-dir, the working directory in no git repository): it
// is reseat's, not start --plan's words, for a real reseat and a dry run,
// with nothing on stdout and nothing recorded.
func TestOrgReseat_NoMainWorktreeIsAReseatRefusal(t *testing.T) {
	fx := newReseatCLIOrg(t)
	outside := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(outside))
	t.Chdir(outside)
	manifest := fx.manifest(t)
	want := `org: reseat of org_id "a" is refused before anything is stopped: reseat needs the main worktree of the repository, ` +
		"as ralph org start --plan does, and it was not found (" + orgMainWorktreeMissing + "): run ralph org reseat again " +
		"from the main worktree or one of its linked worktrees"
	for _, extra := range [][]string{nil, {"--dry-run"}} {
		stdout, _, err := fx.reseat(t, extra...)
		if err == nil || err.Error() != want {
			t.Errorf("reseat %v: error %v; want %q", extra, err, want)
		}
		if stdout != "" {
			t.Errorf("reseat %v printed on stdout:\n%s", extra, stdout)
		}
	}
	if fx.manifest(t) != manifest {
		t.Error("a refused reseat changed the manifest")
	}
}

// TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts pins the key
// sentences of `ralph org reseat --help` (FR-1, FR-3, FR-7, FR-8, FR-11 and
// AC-16's help): what reseat keeps, the orgs it takes, the targets, the
// order, the refusal that stops nothing, the retry with the --config and
// --state-dir given, --dry-run, reseat.lock and the records, the DECISION it
// never sends, also after a failed start, and the move to another account.
// It also checks that `ralph org --help` lists the verb.
func TestOrgReseatHelp_SaysWhatItDoesAndHowToSwitchAccounts(t *testing.T) {
	out, err := runOrgCmd(t, "reseat", "--help")
	if err != nil {
		t.Fatalf("reseat --help: %v", err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"ralph org reseat stops the seats of an org and starts them again in the current environment, without a disband. " +
			"The org keeps its plan approvals, its reservation, and its binding to the feature.",
		"Only an org started with `ralph org start --plan` is accepted",
		"With no --seat, reseat stops and starts every active seat of the org, and also starts the seats that a reseat cut " +
			"off before its record (killed, say) stopped.",
		"To finish a cut-off reseat, run the same command again, with the same --seat values: a reseat with no --seat in " +
			"place of one that named seats would also stop and start every active seat, the leader included.",
		"--seat names one seat, and can be repeated: a seat spawned since the org's last disband, whether it is active, " +
			"stopped, or its start failed, or a seat a reseat stopped or started since then, so the command a failed reseat " +
			"prints can be run as it is.",
		"The output then names the seats stopped, started, and left, and the reseat command that starts the rest, with " +
			"the --config and --state-dir that were given, and reseat exits 1.",
		"When the leader is not one of the seats and keeps running, the output ends with the `ralph org send` command of " +
			"a DECISION that asks the leader to send the seats reseat started their TASKs again, also after a start that " +
			"failed: they start with no TASK.",
		"With neither an active seat nor a seat a cut-off reseat stopped, reseat with no --seat is refused and names the " +
			"stopped seats.",
		"--driver and --model go together and only with exactly one --seat",
		"A seat of a role with no role prompt template (spawned with --prompt, which the ledger does not keep) is refused: " +
			"stop it and spawn it again by hand.",
		"A real reseat does not end at SIGINT, SIGTERM, or SIGHUP: it says so on stderr and goes on until it has written its " +
			"record and printed what it did.",
		"Nor does it end when nobody reads its stdout or stderr any more (a closed pipe, SIGPIPE): what it prints there is " +
			"lost, and the record is written all the same.",
		"Its exit code is the reseat's result, also after a signal: a reseat that started every target seat again and " +
			"recorded it exits 0.",
		"When a check fails, reseat stops nothing, records nothing, and exits 1. Then it stops the seats other than the " +
			"leader, then the leader. It starts the leader first",
		"then the other seats in the order the org first spawned them.",
		"reseat never sends it; check the leader's pane before you run it.",
		"each holds <state dir>/reseat.lock from its checks until it has written its record, and a second one is refused at once.",
		"Every stop and start a reseat makes records reseat=<id>, and the reseat ends with one reseated event of the org",
		"--dry-run runs the checks and prints the seats reseat would stop and start, in that order, with each seat's " +
			"driver and model. It stops, starts, and writes nothing.",
		"The panes of the seats take the herdr server's environment, not this command's. To move seats to another account " +
			"(another CLAUDE_CONFIG_DIR, say), stop the herdr server, which ends the panes of every org and of a headless " +
			"director in herdr. Then run ralph org reseat from a shell with the new environment: its herdr server check " +
			"starts the server with that environment.",
		"It is refused in the herdr pane (HERDR_PANE_ID) of an active seat of the org, which it would close.",
		"reseat reads ralph.toml as `ralph org start --plan` does. A leader it starts gets the current values of the four " +
			"[reviewer] keys in its task, and an [org.approval].mode other than the one the org was started with refuses the reseat.",
		"The reseat, stop, spawn, and start commands reseat prints for you to run, in its output or in a refusal, repeat " +
			"the --config that was given, made absolute, so they read the same file from any directory. The `ralph org read` " +
			"and `ralph org send` in its output read no ralph.toml and do not repeat it.",
		strings.Join(strings.Fields(orgScaffoldVersionNoticeHelp), " "),
	} {
		if !strings.Contains(help, want) {
			t.Errorf("reseat --help does not say %q:\n%s", want, out)
		}
	}
	out, err = runOrgCmd(t, "--help")
	if err != nil {
		t.Fatalf("org --help: %v", err)
	}
	if !regexp.MustCompile(`(?m)^\s+reseat\s+Stop a feature org's seats`).MatchString(out) {
		t.Errorf("org --help does not list reseat:\n%s", out)
	}
}
