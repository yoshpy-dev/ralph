package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org gate accept` (plan
// docs/plans/active/2026-11-12-gate-lock-accept.md, AC7 and AC8 at the
// command line). The org layer's refusals, the record's Details, and the
// classes are covered by internal/org/gate_accept_test.go and
// mergeorder_test.go; these run the command and check what it prints, its
// exit, what it records, and that merge-order then lists the branch as
// accepted. None runs in parallel: they set environment variables and chdir.

// gateAcceptCLILog is the log of the failures these tests seed
// (mergeOrderGateDetails).
const gateAcceptCLILog = "/state/gates/x.log"

// gateAcceptCLIHead and gateAcceptCLIBase are the head and base of the
// failure seedGateAcceptFailure records.
var (
	gateAcceptCLIHead = strings.Repeat("a", 40)
	gateAcceptCLIBase = strings.Repeat("1", 40)
)

// seedGateAcceptFailure records org a bound to feat/a and its gate_failed
// (test_failed) of gateAcceptCLIHead on gateAcceptCLIBase, as ralph org gate
// records it, in the ledger at stateDir, and returns the failure's ts.
func seedGateAcceptFailure(t *testing.T, stateDir string) string {
	t.Helper()
	appendSeedEvent(t, stateDir, org.ManifestEvent{OrgID: "a", Event: org.EventScopeReserved, Worktree: t.TempDir(),
		Details: "paths=internal/a/ split=demo feature=a digest=0123456789ab branch=feat/a approval=per-plan"})
	appendSeedEvent(t, stateDir, org.ManifestEvent{OrgID: "a", Event: org.EventGateFailed,
		Details: mergeOrderGateDetails(org.GateReasonTestFailed, gateAcceptCLIHead, gateAcceptCLIBase, "refs/remotes/origin/main", "feat/a")})
	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	return events[len(events)-1].TS
}

// gateAcceptCLIArgs are the arguments of a gate accept of feat/a's failure
// in the ledger at stateDir, with log and reason.
func gateAcceptCLIArgs(stateDir, log, reason string) []string {
	return []string{"gate", "accept", "--state-dir", stateDir, "--branch", "feat/a", "--log", log, "--reason", reason}
}

// gateAcceptedEvents returns the gate_accepted events in the ledger at
// stateDir.
func gateAcceptedEvents(t *testing.T, stateDir string) []org.ManifestEvent {
	t.Helper()
	var out []org.ManifestEvent
	for _, ev := range readManifestEvents(t, org.ManifestPathIn(stateDir)) {
		if ev.Event == org.EventGateAccepted {
			out = append(out, ev)
		}
	}
	return out
}

// gateAcceptCLIChdir moves the test into a new temp dir, outside any
// repository and with no ralph.toml.
func gateAcceptCLIChdir(t *testing.T) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
}

// AC7 at the command line: gate accept records one gate_accepted under org
// a, by USER with no herdr pane, and prints the acceptance, one line each,
// with the --reason trimmed, nothing on stderr, and exit 0.
func TestOrgGateAccept_RecordsAndPrintsTheAcceptance(t *testing.T) {
	approveEnv(t)
	gateAcceptCLIChdir(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	failedTS := seedGateAcceptFailure(t, stateDir)
	before := len(readManifestEvents(t, org.ManifestPathIn(stateDir)))

	stdout, stderr, err := runOrgCmdStreams(t, gateAcceptCLIArgs(stateDir, gateAcceptCLILog, "  the test fails only under load ")...)
	if err != nil || stderr != "" {
		t.Fatalf("gate accept: err %v, stderr %q (stdout %q)", err, stderr, stdout)
	}
	want := "ACCEPTED: feat/a\nFAILED: gate_failed (test_failed) at " + failedTS + "\nHEAD: " + gateAcceptCLIHead + "\nBASE: " +
		gateAcceptCLIBase + "\nLOG: " + gateAcceptCLILog + "\nBY: tester\nNOTE: the test fails only under load\n"
	if stdout != want {
		t.Errorf("stdout =\n%s\nwant\n%s", stdout, want)
	}
	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	if len(events) != before+1 {
		t.Fatalf("the ledger got %d events; want one", len(events)-before)
	}
	wantDetails := "head=" + gateAcceptCLIHead + " base=" + gateAcceptCLIBase + " base_ref=refs/remotes/origin/main branch=feat/a " +
		"failed_event=gate_failed failed_reason=test_failed failed_ts=" + failedTS + " by=tester herdr_pane=unset " +
		`failed_log="` + gateAcceptCLILog + `" note="the test fails only under load"`
	if ev := events[before]; ev.Event != org.EventGateAccepted || ev.OrgID != "a" || ev.SeatID != "" || ev.Details != wantDetails {
		t.Errorf("recorded %+v; want gate_accepted of org a with %q", ev, wantDetails)
	}
}

// AC7's refusals at the command line: a log that is not the latest
// failure's, the failure accepted already, and HERDR_PANE_ID set to an
// active seat's pane each exit 1 with the refusal as the error, nothing on
// stdout or stderr, and nothing recorded.
func TestOrgGateAccept_RefusalExitsOneAndRecordsNothing(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, stateDir string)
		log   string
		want  string
	}{
		{"another log", func(*testing.T, string) {}, "/state/gates/y.log", "its log is " + gateAcceptCLILog + ", not " +
			"/state/gates/y.log, so nothing was recorded: a newer gate may have replaced the failure that was read; read " +
			gateAcceptCLILog + " and ask the human again before accepting that failure"},
		{"accepted already", func(t *testing.T, stateDir string) {
			if _, _, err := runOrgCmdStreams(t, gateAcceptCLIArgs(stateDir, gateAcceptCLILog, "first")...); err != nil {
				t.Fatalf("the first gate accept: %v", err)
			}
		}, gateAcceptCLILog, "and tester accepted that failure at "},
		{"a seat's pane", func(t *testing.T, stateDir string) {
			appendSeedEvent(t, stateDir, org.ManifestEvent{OrgID: "a", SeatID: "reviewer", Event: org.EventSpawned, Role: "reviewer",
				PaneID: "p-1"})
			t.Setenv("HERDR_PANE_ID", "p-1")
		}, gateAcceptCLILog, `org: gate accept: refused, since HERDR_PANE_ID "p-1" is the pane of the active seat "reviewer" of ` +
			`org_id "a": ralph org gate accept records a human's acceptance of a gate failure`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			approveEnv(t)
			gateAcceptCLIChdir(t)
			stateDir := filepath.Join(t.TempDir(), "state")
			seedGateAcceptFailure(t, stateDir)
			tc.setup(t, stateDir)
			before := fileBytesOrAbsent(t, org.ManifestPathIn(stateDir))
			stdout, stderr, err := runOrgCmdStreams(t, gateAcceptCLIArgs(stateDir, tc.log, "again")...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("gate accept = %v; want an error containing %q", err, tc.want)
			}
			if stdout != "" || stderr != "" {
				t.Errorf("stdout = %q, stderr = %q; want nothing", stdout, stderr)
			}
			if after := fileBytesOrAbsent(t, org.ManifestPathIn(stateDir)); after != before {
				t.Errorf("the refusal changed the ledger:\nbefore %s\nafter  %s", before, after)
			}
		})
	}
}

// The flags are checked before the ledger is read: --org-id (a flag of ralph
// org, which this grandchild command sees), a missing --branch, --log or
// --reason, and an argument are refused, with nothing on stdout or stderr
// and the ledger's directory never made.
func TestOrgGateAccept_FlagsAreRefusedBeforeTheLedgerIsRead(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"--org-id", []string{"--org-id", "a", "--branch", "feat/a", "--log", gateAcceptCLILog, "--reason", "r"},
			"org: gate accept: --org-id does not apply: gate accept records the acceptance under the org of the branch's latest " +
				"gate record"},
		{"no --branch", []string{"--log", gateAcceptCLILog, "--reason", "r"}, "org: gate accept: --branch is required"},
		{"no --log", []string{"--branch", "feat/a", "--reason", "r"}, "org: gate accept: --log is required"},
		{"no --reason", []string{"--branch", "feat/a", "--log", gateAcceptCLILog}, "org: gate accept: --reason is required"},
		{"a --reason of two lines", []string{"--branch", "feat/a", "--log", gateAcceptCLILog, "--reason", "a\nb"},
			"org: gate accept: --reason must be a single line"},
		{"an argument", []string{"extra", "--branch", "feat/a", "--log", gateAcceptCLILog, "--reason", "r"},
			`unknown command "extra" for "ralph org gate accept"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			approveEnv(t)
			gateAcceptCLIChdir(t)
			stateDir := filepath.Join(t.TempDir(), "state")
			stdout, stderr, err := runOrgCmdStreams(t, append([]string{"gate", "accept", "--state-dir", stateDir}, tc.args...)...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("gate accept %q = %v; want an error containing %q", tc.args, err, tc.want)
			}
			if stdout != "" || stderr != "" {
				t.Errorf("stdout = %q, stderr = %q; want nothing", stdout, stderr)
			}
			if _, err := os.Stat(stateDir); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("the ledger's directory %s exists (%v); want nothing made", stateDir, err)
			}
		})
	}
}

// AC8 through the CLI: once gate accept records the acceptance of a branch's
// failure on origin's tips, merge-order lists it as accepted, its Detail
// naming who accepted it, and its --json gate holds the log and the
// acceptance.
func TestOrgGateAccept_MergeOrderListsTheBranchAsAccepted(t *testing.T) {
	approveEnv(t)
	fx := newMergeOrderFixture(t, false)
	head := fx.branch(t, "feat/a")
	fx.push(t, "feat/a")
	fx.bind(t, "a", "a", "feat/a")
	fx.gate(t, "a", "feat/a", org.GateReasonTestFailed, head, fx.tip(t, "main"))
	if stdout, stderr, err := runOrgCmdStreams(t, gateAcceptCLIArgs(fx.stateDir, gateAcceptCLILog, "flaky under load")...); err != nil {
		t.Fatalf("gate accept: %v (stdout %q, stderr %q)", err, stdout, stderr)
	}
	accepted := gateAcceptedEvents(t, fx.stateDir)
	if len(accepted) != 1 {
		t.Fatalf("gate_accepted events = %+v; want one", accepted)
	}

	res, _ := fx.runJSON(t)
	if len(res.Entries) != 1 {
		t.Fatalf("entries = %+v; want feat/a only", res.Entries)
	}
	e := res.Entries[0]
	want := org.MergeOrderAcceptance{By: "tester", TS: accepted[0].TS, Reason: "flaky under load"}
	if e.Class != org.MergeClassAccepted || e.Gate == nil || e.Gate.Log != gateAcceptCLILog || e.Gate.Accepted == nil ||
		*e.Gate.Accepted != want {
		t.Fatalf("feat/a = %+v (gate %+v); want accepted with the log %s and %+v", e, e.Gate, gateAcceptCLILog, want)
	}
	stdout, _, err := fx.run(t)
	if err != nil {
		t.Fatal(err)
	}
	rows := parseMergeOrderText(t, stdout).rows
	if len(rows) != 1 || rows[0][1] != org.MergeClassAccepted || !strings.Contains(rows[0][7], "tester accepted the failure at "+
		accepted[0].TS+": flaky under load") {
		t.Errorf("rows = %q; want feat/a accepted, naming tester's acceptance", rows)
	}
}

// AC13's help for gate accept (S4 extends it): what it records, the three
// required flags, who runs it, the seat pane, what merge-order and merge do
// then, that a later gate record voids it, and the refusals; gate --help
// lists it as a subcommand and keeps its own text.
func TestOrgGateAccept_Help(t *testing.T) {
	out, err := runOrgCmd(t, "gate", "accept", "--help")
	if err != nil {
		t.Fatal(err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"ralph org gate accept records that a human accepted the failure of a feature branch's latest gate record: a gate_failed " +
			"or gate_ci_failed that ralph org gate or ralph org regate recorded with the reason static_failed, test_failed, or " +
			"ci_failed",
		"It records gate_accepted in the manifest, under the org of that gate record",
		"All three flags are required.",
		"--log names the gate log of the failure the human read: the LOG: line ralph org gate printed, or the gate's log in " +
			"ralph org merge-order --json",
		"A human runs it, or a director the human told to",
		"It is refused when HERDR_PANE_ID is the pane of an active seat",
		"so two accepts of the same failure record one",
		"its reason is head_moved or tree_changed (the scripts passed: run the gate again); its log is not --log (a newer gate " +
			"replaced the failure the human read: read the latest failure's log, which the refusal names, and ask the human again)",
		"--org-id is given",
		"lists it as accepted, right after the ready ones, where a pass would be ready (rebase, wrong_base, regate, and " +
			"no_remote stay as they are)",
		"ralph org merge merges an accepted branch with the same checks as a ready one, prints GATE: accepted, and records " +
			"gate=accepted in pr_merged",
		"Any gate record of the branch after the acceptance voids it",
		"--branch string", "--log string", "--reason string",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("gate accept --help does not say %q:\n%s", want, out)
		}
	}
	out, err = runOrgCmd(t, "gate", "--help")
	if err != nil {
		t.Fatal(err)
	}
	help = strings.Join(strings.Fields(out), " ")
	for _, want := range []string{"Available Commands: accept Record a human's acceptance of a feature branch's latest gate failure",
		"ralph org gate records which head of a feature org passed the verification scripts on which base."} {
		if !strings.Contains(help, want) {
			t.Errorf("gate --help does not say %q:\n%s", want, out)
		}
	}
}
