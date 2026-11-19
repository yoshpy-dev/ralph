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
// then the resolve approve printed, with no --without-decision, to run once
// that send succeeded, and last a sentence that says when --without-decision
// is for and keeps approve's note, with no command that includes the flag; a
// dry-run send does not count. Once the DECISION is sent, the resolve prints
// `resolved <id>` and nothing on stderr.
func TestOrgInboxResolve_ApprovedItemNeedsTheDecisionSent(t *testing.T) {
	stateDir, id, ref := approvedPlanItem(t)
	note := "approved " + ref.String()
	send := org.ApprovalDecisionCommand("a", ref, stateDir)
	if !strings.Contains(send, " --state-dir '"+stateDir+"' ") {
		t.Fatalf("send command %q does not repeat the state dir", send)
	}
	resolve := org.ApprovalResolveCommand(id, ref, stateDir)
	if !strings.HasSuffix(resolve, " --note '"+note+"'") || strings.Contains(resolve, "--without-decision") {
		t.Fatalf("resolve command %q does not end with approve's note %q or names --without-decision", resolve, note)
	}

	refused := func(why string) {
		t.Helper()
		before := inboxFileOrAbsent(t, stateDir)
		stdout, stderr, err := runOrgCmdStreams(t, "inbox", "resolve", id, "--note", note, "--state-dir", stateDir)
		if !errors.Is(err, org.ErrApprovalDecisionNotSent) || ExitCode(err) != 1 {
			t.Fatalf("resolve %s = %v; want ErrApprovalDecisionNotSent with exit 1", why, err)
		}
		for _, want := range []string{
			"org: inbox: resolve " + id + ": refused, with nothing written: the approval of this inbox item's plan is recorded, " +
				"but the manifest holds no counted approval DECISION sent to the org's leader after it",
			"tell the leader with:\n" + send + "\nand once that send succeeded, resolve the item with:\n" + resolve + "\n",
			"Adding --without-decision to that resolve closes the item on purpose with no counted DECISION: after you told the leader another way, " +
				"or when a ralph from before this check sent the DECISION (its sent event has no text_sha256, so it is not counted). ",
			"Keep approve's note on that resolve, since a leader woken while it waits reads the note as the approval.",
		} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("resolve %s error =\n%v\nwant it to contain %q", why, err, want)
			}
		}
		// Issue #345: the last line of the error is not a command, and no
		// line is a command with --without-decision; the send comes before
		// the resolve.
		lines := strings.Split(err.Error(), "\n")
		if last := lines[len(lines)-1]; strings.HasPrefix(last, "ralph ") || !strings.Contains(last, "--without-decision") {
			t.Errorf("resolve %s error =\n%v\nwant its last line to be a sentence that names --without-decision, not a command; got %q", why, err, last)
		}
		for _, line := range lines {
			if strings.HasPrefix(line, "ralph ") && strings.Contains(line, "--without-decision") {
				t.Errorf("resolve %s error =\n%v\nwant no line to be a command with --without-decision; got %q", why, err, line)
			}
		}
		if s, r := strings.Index(err.Error(), "\n"+send+"\n"), strings.Index(err.Error(), "\n"+resolve+"\n"); s < 0 || r < 0 || s >= r {
			t.Errorf("resolve %s error =\n%v\nwant the send command (at %d) before the resolve command (at %d)", why, err, s, r)
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

// AC-3: --without-decision, added to the resolve approve printed, resolves an
// approved item whose DECISION was not sent with approve's note, prints
// `resolved <id>`, and says on stderr, in one line, that no approval DECISION
// this ralph recorded was found for the leader, so the human makes sure the
// leader was told; it does not say the leader was not told, since it may have
// been, another way or by an older ralph.
func TestOrgInboxResolve_WithoutDecisionSaysNoDecisionWasFound(t *testing.T) {
	stateDir, id, ref := approvedPlanItem(t)
	note := "approved " + ref.String()
	stdout, stderr := runInboxOK(t, "inbox", "resolve", id, "--note", note, "--without-decision", "--state-dir", stateDir)
	if stdout != "resolved "+id+"\n" {
		t.Errorf("stdout = %q; want %q", stdout, "resolved "+id+"\n")
	}
	want := "note: resolved " + id + " with --without-decision: no approval DECISION of plan " + ref.String() +
		" recorded by this ralph was found for the leader of org a (one a ralph from before this check sent is not counted), " +
		"so make sure the leader was told that the plan is approved\n"
	if stderr != want {
		t.Errorf("stderr = %q; want %q", stderr, want)
	}
	if strings.Contains(stderr, "has not been told") {
		t.Errorf("stderr = %q; want it not to claim the leader was not told", stderr)
	}
	if item := cliInboxItem(t, stateDir, id); item.State != org.InboxStateResolved || item.Note != note {
		t.Errorf("item %s = state %q note %q; want resolved with approve's note %q", id, item.State, item.Note, note)
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
// DECISION sent, that a DECISION sent by an older ralph is not counted, and
// when --without-decision is for, with approve's note kept on that resolve
// for a woken leader to read.
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
		"--without-decision resolves an approved plan QUESTION on purpose with no counted DECISION: after the leader was told another way, " +
			"or when a ralph from before this check sent the DECISION.",
		"Keep approve's note (approved <path> sha256:<digest>) on that resolve: " +
			"a leader woken while it waits for the DECISION reads that note as the approval.",
		"resolve then says on stderr that no counted DECISION was found, to make sure the leader was told.",
		"--without-decision resolve on purpose a plan QUESTION whose approval is recorded with no counted approval DECISION sent to its leader: " +
			"after the leader was told another way, or when a ralph from before this check sent the DECISION (not counted); " +
			"keep approve's note 'approved <path> sha256:<digest>'",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("inbox resolve --help does not say %q:\n%s", want, out)
		}
	}
}
