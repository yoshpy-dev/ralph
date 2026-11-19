package cli

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// `ralph org inbox resolve` of an approved plan item (plan
// docs/plans/active/2026-11-19-approve-resolve-decision.md, AC-1 to AC-3,
// AC-7's resolve help, and AC-8 through the CLI). The ledger is seeded as
// `ralph org start --plan` leaves it (seedApprovalOrg, org_approve_test.go),
// approve runs for real, and the DECISION is sent for real through the herdr
// stub (setupOrgStubPATH).

// approvedPlanItem seeds org a in per-plan mode in the ledger at a new state
// dir, escalates its leader's QUESTION for the plan, and approves it with
// `ralph org approve`. It returns the state dir, the item, and the plan.
func approvedPlanItem(t *testing.T) (stateDir, id string, ref org.PlanRef) {
	t.Helper()
	setupOrgStubPATH(t)
	approveEnv(t)
	stateDir = filepath.Join(t.TempDir(), "state")
	ref = writeWorktreePlan(t, seedApprovalOrg(t, stateDir, "a", "per-plan"), approveTestPlanRel, approveTestPlanDraft)
	id = escalateOK(t, stateDir, "a", planQuestion(ref))
	runInboxOK(t, "approve", id, "--state-dir", stateDir)
	return stateDir, id, ref
}

// sendApprovalDecision sends the approval DECISION of ref to org a's leader
// with `ralph org send`, as approve prints it.
func sendApprovalDecision(t *testing.T, stateDir string, ref org.PlanRef) (stdout, stderr string) {
	t.Helper()
	return runInboxOK(t, "send", "--org-id", "a", "--to", org.LeaderIdentity, "--text", org.ApprovalDecisionText(ref),
		"--enter-delay-ms", "1", "--state-dir", stateDir)
}

// AC-1, AC-2, and AC-8: resolve of an approved item exits 1 with nothing
// written and nothing on stdout until the DECISION is sent, and its error
// gives the send command approve printed, with the --state-dir it was given,
// and names --without-decision; a dry-run send does not count. Once the
// DECISION is sent, the resolve prints `resolved <id>` and nothing on stderr.
func TestOrgInboxResolve_ApprovedItemNeedsTheDecisionSent(t *testing.T) {
	stateDir, id, ref := approvedPlanItem(t)
	note := "approved " + ref.String()
	send := org.ApprovalDecisionCommand("a", ref, stateDir)
	if !strings.Contains(send, " --state-dir '"+stateDir+"' ") {
		t.Fatalf("send command %q does not repeat the state dir", send)
	}

	refused := func(why string) {
		t.Helper()
		before := inboxFileOrAbsent(t, stateDir)
		stdout, stderr, err := runOrgCmdStreams(t, "inbox", "resolve", id, "--note", note, "--state-dir", stateDir)
		if !errors.Is(err, org.ErrApprovalDecisionNotSent) || ExitCode(err) != 1 {
			t.Fatalf("resolve %s = %v; want ErrApprovalDecisionNotSent with exit 1", why, err)
		}
		for _, want := range []string{
			"org: inbox: resolve " + id + ": refused, with nothing written: ",
			"tell the leader with:\n" + send + "\nand once that send succeeded, run the resolve again. ",
			"run the resolve with --without-decision",
		} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("resolve %s error =\n%v\nwant it to contain %q", why, err, want)
			}
		}
		if stdout != "" || stderr != "" {
			t.Errorf("resolve %s printed stdout %q, stderr %q; want nothing", why, stdout, stderr)
		}
		if after := inboxFileOrAbsent(t, stateDir); after != before {
			t.Errorf("a refused resolve wrote to the inbox:\nbefore: %s\nafter:  %s", before, after)
		}
		if item := cliInboxItem(t, stateDir, id); item.State != org.InboxStateOpen {
			t.Errorf("item %s is %s after the refusal; want it still open", id, item.State)
		}
	}
	refused("with no send")
	runInboxOK(t, "send", "--org-id", "a", "--to", org.LeaderIdentity, "--text", org.ApprovalDecisionText(ref), "--dry-run", "--state-dir", stateDir)
	refused("after a dry-run send")

	sendApprovalDecision(t, stateDir, ref)
	stdout, stderr := runInboxOK(t, "inbox", "resolve", id, "--note", note, "--state-dir", stateDir)
	if stdout != "resolved "+id+"\n" || stderr != "" {
		t.Errorf("resolve after the send printed stdout %q, stderr %q; want %q and nothing", stdout, stderr, "resolved "+id+"\n")
	}
	if item := cliInboxItem(t, stateDir, id); item.State != org.InboxStateResolved || item.Note != note {
		t.Errorf("item %s = state %q note %q; want resolved with %q", id, item.State, item.Note, note)
	}
}

// AC-3: --without-decision resolves an approved item whose DECISION was not
// sent, prints `resolved <id>`, and says on stderr, in one line, that the
// leader was not told.
func TestOrgInboxResolve_WithoutDecisionSaysTheLeaderWasNotTold(t *testing.T) {
	stateDir, id, ref := approvedPlanItem(t)
	stdout, stderr := runInboxOK(t, "inbox", "resolve", id, "--note", "told the leader by phone", "--without-decision", "--state-dir", stateDir)
	if stdout != "resolved "+id+"\n" {
		t.Errorf("stdout = %q; want %q", stdout, "resolved "+id+"\n")
	}
	want := "note: resolved " + id + " without the approval DECISION: the leader of org a has not been told that plan " +
		ref.String() + " is approved, so tell it another way\n"
	if stderr != want {
		t.Errorf("stderr = %q; want %q", stderr, want)
	}
	if item := cliInboxItem(t, stateDir, id); item.State != org.InboxStateResolved {
		t.Errorf("item %s is %s; want resolved", id, item.State)
	}
}

// AC-4 through the CLI: an approval DECISION whose send could not confirm the
// submit counts, and resolve says on stderr, in one line, to check the
// leader's pane with `ralph org read`, repeating --state-dir.
func TestOrgInboxResolve_UnconfirmedSubmitSaysToReadTheLeadersPane(t *testing.T) {
	stateDir, id, ref := approvedPlanItem(t)
	t.Setenv("ORG_STUB_AGENT_WAIT_CONFIRM_FAIL", "1")
	if _, sendErr := sendApprovalDecision(t, stateDir, ref); !strings.Contains(sendErr, "could not confirm") {
		t.Fatalf("send stderr = %q; want the unconfirmed-submit warning", sendErr)
	}
	stdout, stderr := runInboxOK(t, "inbox", "resolve", id, "--note", "approved", "--state-dir", stateDir)
	if stdout != "resolved "+id+"\n" {
		t.Errorf("stdout = %q; want %q", stdout, "resolved "+id+"\n")
	}
	read := "ralph org read --org-id a --seat leader --state-dir " + org.ShellWord(stateDir)
	want := "note: the approval DECISION for " + id + " was sent to the leader of org a, but the send could not confirm its submit " +
		"(submit_unconfirmed=true): check with " + read + " that the leader received it\n"
	if stderr != want {
		t.Errorf("stderr = %q; want %q", stderr, want)
	}
}

// AC-7 for the resolve: its help says what is refused, what counts as the
// DECISION sent, --without-decision, and that a DECISION sent by an older
// ralph is not counted.
func TestOrgInboxResolveHelp_SaysTheApprovalDecisionRule(t *testing.T) {
	out, err := runOrgCmd(t, "inbox", "resolve", "--help")
	if err != nil {
		t.Fatalf("inbox resolve --help: %v", err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"A leader's QUESTION for its plan (a PLAN: header) whose approval `ralph org approve` recorded is refused, with exit 1 and nothing written, " +
			"until the approval DECISION has been sent to the leader of the item's org",
		"a `ralph org send` of the DECISION approve printed, made after the approval was recorded and not a dry run",
		"The refusal prints that send command; run it, then run the resolve again.",
		"A DECISION sent by a ralph from before this check is not counted",
		"check the leader's pane with `ralph org read`",
		"--without-decision resolves such an item anyway, on purpose",
		"--without-decision resolve a plan QUESTION whose approval is recorded although no approval DECISION was sent to its leader",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("inbox resolve --help does not say %q:\n%s", want, out)
		}
	}
}
