package cli

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org approve` and the CLI side of the approval check (plan
// docs/plans/active/2026-10-12-org-plan-approval.md, AC2, AC3, AC4 and
// AC13, and AC10 of plan 2026-10-14-org-director-attach, which makes approve
// record only and print the resolve after the send: the tests that checked
// 7a's resolve now check that the item stays open). Most of them seed the
// ledger as `ralph org start --plan` leaves it
// (seedApprovalOrg) instead of running start, whose recording of the mode is
// covered by TestOrgStartPlan_ApprovalModeFromMainWorktree; the AC13 test
// runs the real start in a temp git repo. None of them runs in parallel:
// they set environment variables, and some chdir.

// approveTestPlanRel is where the tests' feature plan lives in a feature
// worktree.
const approveTestPlanRel = "docs/plans/active/2026-10-13-a.md"

// approveTestPlanDraft is a feature plan as its leader asks for the
// approval: the two header lines not yet approved, a - Branch: line, and a
// Progress checklist.
const approveTestPlanDraft = `# a

- Status: Draft
- Approved: TBD
- Branch: feat/a

## Objective

Build a.

## Progress checklist

- [ ] Implementation started
`

// markPlanApproved is plan with the two header lines its leader writes after
// the approval of digest.
func markPlanApproved(plan, digest string) string {
	plan = strings.Replace(plan, "- Status: Draft", "- Status: Approved", 1)
	return strings.Replace(plan, "- Approved: TBD", "- Approved: 2026-10-13 sha256:"+digest, 1)
}

// approveEnv pins what approve records about who ran it: USER is tester and
// HERDR_PANE_ID is empty, so no test depends on whether `go test` runs in a
// herdr pane.
func approveEnv(t *testing.T) {
	t.Helper()
	t.Setenv("USER", "tester")
	t.Setenv("HERDR_PANE_ID", "")
}

// seedApprovalOrg records orgID in the ledger at stateDir the way `ralph org
// start --plan` leaves it, without running start: a scope_reserved bound to
// feature orgID of split plan demo, with `approval=<mode>` when mode is not
// "" (no token: a start from before approval modes), in a new feature
// worktree under a temp dir (returned, symlinks resolved), and its leader
// seat spawned on pane pane-<orgID>.
func seedApprovalOrg(t *testing.T, stateDir, orgID, mode string) string {
	t.Helper()
	wt, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	details := "paths=internal/" + orgID + "/ split=demo feature=" + orgID + " digest=0123456789ab branch=feat/" + orgID
	if mode != "" {
		details += " approval=" + mode
	}
	appendSeedEvent(t, stateDir, org.ManifestEvent{OrgID: orgID, Event: org.EventScopeReserved, Worktree: wt, Details: details})
	appendSeedEvent(t, stateDir, org.ManifestEvent{
		OrgID: orgID, SeatID: org.LeaderIdentity, Event: org.EventSpawned, Role: org.LeaderIdentity,
		Driver: "claude", Model: "sonnet", PaneID: "pane-" + orgID, AgmsgTeam: "ralph-" + orgID, Worktree: wt,
	})
	return wt
}

// writeWorktreePlan writes content as the plan at rel in worktree and
// returns its ref (rel and org.PlanDigest of content).
func writeWorktreePlan(t *testing.T, worktree, rel, content string) org.PlanRef {
	t.Helper()
	path := filepath.Join(worktree, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, content)
	return org.PlanRef{Path: rel, Digest: org.PlanDigest([]byte(content))}
}

// planQuestion is the inbox QUESTION a leader escalates to ask for the
// approval of ref.
func planQuestion(ref org.PlanRef) string {
	return "TYPE: QUESTION\nPLAN: " + ref.String() + "\n\nplease approve the plan"
}

// planApprovedEvents returns the plan_approved events in the manifest under
// stateDir.
func planApprovedEvents(t *testing.T, stateDir string) []org.ManifestEvent {
	t.Helper()
	var approvals []org.ManifestEvent
	for _, ev := range readManifestEvents(t, org.ManifestPathIn(stateDir)) {
		if ev.Event == org.EventPlanApproved {
			approvals = append(approvals, ev)
		}
	}
	return approvals
}

// wantApprovedBlock is what approve prints on stdout for the item id of
// orgID approving ref: its approved line, then the send command whose
// DECISION tells the leader, the message spelled out here, and then the
// resolve command to run once that send succeeded. stateDirFlag is
// " --state-dir '<dir>'" when the commands repeat the ledger, else "".
func wantApprovedBlock(id, orgID string, ref org.PlanRef, stateDirFlag string) string {
	return "approved " + id + " (org=" + orgID + " plan=" + ref.Path + " sha256:" + ref.Digest + ")\n" +
		"approve sends no message and resolves nothing; tell the leader of org " + orgID + " with:\n" +
		"ralph org send --org-id " + orgID + " --to leader" + stateDirFlag + " --text 'TYPE: DECISION\n" +
		"PLAN: " + ref.Path + " sha256:" + ref.Digest + "\n" +
		"\n" +
		"approved'\n" +
		"once that send succeeded, resolve the item with:\n" +
		"ralph org inbox resolve " + id + stateDirFlag + " --note 'approved " + ref.Path + " sha256:" + ref.Digest + "'\n"
}

// ledgerFiles is the manifest's and the inbox's bytes under stateDir, which a
// refused approve must leave as they were.
func ledgerFiles(t *testing.T, stateDir string) string {
	t.Helper()
	return fileBytesOrAbsent(t, org.ManifestPathIn(stateDir)) + "\n--- inbox ---\n" + inboxFileOrAbsent(t, stateDir)
}

func fileBytesOrAbsent(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "<absent>"
	}
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

// AC2 and AC10: approve records one plan_approved for the item, with the
// plan, its digest and whole-file hash, the item, and who ran it; leaves the
// inbox as it was, so the item stays open; prints the approved line, the
// DECISION command, and then the resolve command (both repeating the
// --state-dir it was given), and the note that the file was recorded as it
// is now; and leaves the plan file as it was.
func TestOrgApprove_RecordsAndPrintsTheSendThenTheResolve(t *testing.T) {
	approveEnv(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	wt := seedApprovalOrg(t, stateDir, "a", "per-plan")
	ref := writeWorktreePlan(t, wt, approveTestPlanRel, approveTestPlanDraft)
	id := escalateOK(t, stateDir, "a", planQuestion(ref))
	planPath := filepath.Join(wt, filepath.FromSlash(approveTestPlanRel))
	eventsBefore := len(readManifestEvents(t, org.ManifestPathIn(stateDir)))
	inboxBefore := inboxFileOrAbsent(t, stateDir)
	cwd, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runOrgCmdStreams(t, "approve", id, "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("approve %s: %v (stdout: %s stderr: %s)", id, err, stdout, stderr)
	}
	if want := wantApprovedBlock(id, "a", ref, " --state-dir '"+stateDir+"'") + "\n" + approveRecordedNote + "\n"; stdout != want {
		t.Errorf("stdout =\n%s\nwant\n%s", stdout, want)
	}
	if stderr != "" {
		t.Errorf("stderr = %q; want nothing for a per-plan org", stderr)
	}

	approvals := planApprovedEvents(t, stateDir)
	if len(approvals) != 1 {
		t.Fatalf("plan_approved events = %+v; want one", approvals)
	}
	wantDetails := "plan=" + ref.Path + " digest=" + ref.Digest + " content=" + org.PlanContentHash([]byte(approveTestPlanDraft)) +
		" inbox=" + id + " by=tester herdr_pane=unset cwd=" + strconv.Quote(wd)
	if a := approvals[0]; a.OrgID != "a" || a.SeatID != "" || a.DryRun || a.Details != wantDetails {
		t.Errorf("plan_approved = %+v; want org a, no seat, not dry-run, Details %q", a, wantDetails)
	}
	if got := len(readManifestEvents(t, org.ManifestPathIn(stateDir))); got != eventsBefore+1 {
		t.Errorf("manifest events %d -> %d; want only the plan_approved added", eventsBefore, got)
	}
	if item := cliInboxItem(t, stateDir, id); item.State != org.InboxStateOpen || item.Note != "" {
		t.Errorf("item %s = state %q note %q; want it still open with no note", id, item.State, item.Note)
	}
	if after := inboxFileOrAbsent(t, stateDir); after != inboxBefore {
		t.Errorf("approve wrote to the inbox:\nbefore: %s\nafter:  %s", inboxBefore, after)
	}
	if got := string(readFileBytes(t, planPath)); got != approveTestPlanDraft {
		t.Errorf("approve changed the plan file:\n%s", got)
	}
	for _, want := range []string{"approve recorded each plan file as it is now",
		"the leader may change only the plan's - Status: and - Approved: lines and its checkbox marks"} {
		if !strings.Contains(approveRecordedNote, want) {
			t.Errorf("approveRecordedNote %q does not say %q", approveRecordedNote, want)
		}
	}
}

// The DECISION command approve prints is one POSIX shell command: run by sh,
// with a stub ralph on PATH that prints its arguments, it gives `org send`
// to the org's leader, the --state-dir, and the message, line breaks and
// single quotes kept; and those arguments, with --dry-run, are a send the CLI
// accepts. The resolve command after it is one too: it gives `org inbox
// resolve` of the item with the --state-dir and the note, single quotes
// kept, and those arguments resolve the item with that note.
func TestOrgApprove_TheDecisionAndResolveCommandsRunInAPOSIXShell(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Fatalf("sh is needed to run the printed command: %v", err)
	}
	approveEnv(t)
	stateDir := filepath.Join(t.TempDir(), "it's a state dir")
	wt := seedApprovalOrg(t, stateDir, "a", "per-plan")
	ref := writeWorktreePlan(t, wt, "docs/plans/active/it's-a.md", approveTestPlanDraft)
	id := escalateOK(t, stateDir, "a", planQuestion(ref))

	stdout, _ := runInboxOK(t, "approve", id, "--state-dir", stateDir)
	start := strings.Index(stdout, "ralph org send ")
	end := strings.Index(stdout, "\napproved'\n")
	if start < 0 || end < start {
		t.Fatalf("no send command in stdout:\n%s", stdout)
	}
	command := stdout[start : end+len("\napproved'")]
	var resolveCommand string
	for _, line := range strings.Split(stdout[end:], "\n") {
		if strings.HasPrefix(line, "ralph org inbox resolve ") {
			resolveCommand = line
			break
		}
	}
	if resolveCommand == "" {
		t.Fatalf("no resolve command after the send command in stdout:\n%s", stdout)
	}

	stubDir := t.TempDir()
	writeStubScript(t, filepath.Join(stubDir, "ralph"), "#!/bin/sh\nfor a in \"$@\"; do printf '%s\\000' \"$a\"; done\n")
	stubArgv := func(command string) []string {
		t.Helper()
		run := exec.Command(sh, "-c", command)
		run.Env = append(os.Environ(), "PATH="+stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		out, err := run.Output()
		if err != nil {
			t.Fatalf("sh -c %q: %v", command, err)
		}
		return strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	}
	argv := stubArgv(command)
	message := "TYPE: DECISION\nPLAN: docs/plans/active/it's-a.md sha256:" + ref.Digest + "\n\napproved"
	if want := []string{"org", "send", "--org-id", "a", "--to", "leader", "--state-dir", stateDir, "--text", message}; !slices.Equal(argv, want) {
		t.Fatalf("the command's arguments = %q\nwant %q", argv, want)
	}

	sendOut, _ := runInboxOK(t, append(argv[1:], "--dry-run")...)
	if sendOut != "sent message to seat \"leader\"\n" {
		t.Errorf("send --dry-run stdout = %q; want the message sent to the leader", sendOut)
	}
	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	if last := events[len(events)-1]; last.Event != org.EventSent || !last.DryRun || last.OrgID != "a" || last.SeatID != org.LeaderIdentity ||
		!strings.Contains(last.Details, "TYPE: DECISION") {
		t.Errorf("last event = %+v; want a dry-run sent to a/leader with the DECISION", last)
	}

	argv = stubArgv(resolveCommand)
	note := "approved docs/plans/active/it's-a.md sha256:" + ref.Digest
	if want := []string{"org", "inbox", "resolve", id, "--state-dir", stateDir, "--note", note}; !slices.Equal(argv, want) {
		t.Fatalf("the resolve command's arguments = %q\nwant %q", argv, want)
	}
	runInboxOK(t, argv[1:]...)
	if item := cliInboxItem(t, stateDir, id); item.State != org.InboxStateResolved || item.Note != note {
		t.Errorf("item %s = state %q note %q; want resolved with the note %q", id, item.State, item.Note, note)
	}
}

// AC2 and AC4 with two IDs: both plans are recorded, in argument order,
// both items stay open, a block per item in that order, and the note once.
func TestOrgApprove_TwoItemsAtOnce(t *testing.T) {
	approveEnv(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	refs := map[string]org.PlanRef{}
	for _, orgID := range []string{"a", "b"} {
		refs[orgID] = writeWorktreePlan(t, seedApprovalOrg(t, stateDir, orgID, "per-plan"), approveTestPlanRel,
			strings.Replace(approveTestPlanDraft, "Build a.", "Build "+orgID+".", 1))
	}
	e1 := escalateOK(t, stateDir, "a", planQuestion(refs["a"]))
	e2 := escalateOK(t, stateDir, "b", planQuestion(refs["b"]))

	stdout, _ := runInboxOK(t, "approve", e2, e1, "--state-dir", stateDir)
	flag := " --state-dir '" + stateDir + "'"
	if want := wantApprovedBlock(e2, "b", refs["b"], flag) + "\n" + wantApprovedBlock(e1, "a", refs["a"], flag) + "\n" + approveRecordedNote + "\n"; stdout != want {
		t.Errorf("stdout =\n%s\nwant\n%s", stdout, want)
	}
	approvals := planApprovedEvents(t, stateDir)
	if len(approvals) != 2 || approvals[0].OrgID != "b" || !strings.Contains(approvals[0].Details, " inbox="+e2+" ") ||
		approvals[1].OrgID != "a" || !strings.Contains(approvals[1].Details, " inbox="+e1+" ") {
		t.Errorf("plan_approved events = %+v; want b's (%s) then a's (%s)", approvals, e2, e1)
	}
	for _, id := range []string{e1, e2} {
		if item := cliInboxItem(t, stateDir, id); item.State != org.InboxStateOpen {
			t.Errorf("item %s is %s; want it still open", id, item.State)
		}
	}
}

// AC3: when one of the IDs cannot be approved, approve exits 1 with every
// reason, records nothing, resolves nothing, and prints nothing on stdout;
// an unknown, a resolved, or a repeated ID, a non-QUESTION, and a plan whose
// digest changed after the QUESTION are each refused.
func TestOrgApprove_OneBadItemRecordsNothing(t *testing.T) {
	approveEnv(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	refA := writeWorktreePlan(t, seedApprovalOrg(t, stateDir, "a", "per-plan"), approveTestPlanRel, approveTestPlanDraft)
	wtB := seedApprovalOrg(t, stateDir, "b", "per-plan")
	refB := writeWorktreePlan(t, wtB, approveTestPlanRel, approveTestPlanDraft)
	e1 := escalateOK(t, stateDir, "a", planQuestion(refA))
	e2 := escalateOK(t, stateDir, "b", planQuestion(refB))
	e3 := escalateOK(t, stateDir, "a", "TYPE: BLOCKED\nTASK_ID: a\n\ngh auth is missing")
	e4 := escalateOK(t, stateDir, "a", planQuestion(refA))
	runInboxOK(t, "inbox", "resolve", e4, "--note", "docs/reports/x.md", "--state-dir", stateDir)
	// b's plan changes after its leader asked, outside the Progress
	// checklist, which the digest skips.
	changedB := writeWorktreePlan(t, wtB, approveTestPlanRel, strings.Replace(approveTestPlanDraft, "Build a.", "Build a and c.", 1))
	if changedB.Digest == refB.Digest {
		t.Fatalf("the edit left b's digest %s as it was", refB.Digest)
	}
	before := ledgerFiles(t, stateDir)

	cases := []struct {
		name string
		ids  []string
		want []string
		is   error
	}{
		{"a plan changed after the QUESTION, next to a good item", []string{e1, e2}, []string{
			"org: approve: nothing recorded, since 1 of the 2 inbox items cannot be approved\n",
			"org: approve " + e2 + `: plan ` + refB.Path + ` of org_id "b" has digest ` + changedB.Digest + " now, not the sha256:" + refB.Digest + " the QUESTION names",
		}, nil},
		{"an unknown ID next to a good item", []string{e1, "e9"}, []string{"org: approve: nothing recorded, since 1 of the 2 inbox items cannot be approved"}, org.ErrInboxUnknownID},
		{"an unknown ID alone", []string{"e9"}, []string{"nothing recorded, since 1 of the 1 inbox items"}, org.ErrInboxUnknownID},
		{"a BLOCKED item", []string{e3}, []string{"org: approve " + e3 + ": the item is a BLOCKED, not a QUESTION"}, nil},
		{"a resolved item", []string{e1, e4}, []string{"nothing recorded, since 1 of the 2"}, org.ErrInboxResolved},
		{"an ID given twice", []string{e1, e1}, []string{"org: approve: inbox item " + e1 + " is given twice"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, _, err := runOrgCmdStreams(t, append(append([]string{"approve"}, tc.ids...), "--state-dir", stateDir)...)
			if err == nil {
				t.Fatalf("approve %v passed; want it refused (stdout: %s)", tc.ids, stdout)
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error does not contain %q:\n%v", want, err)
				}
			}
			if tc.is != nil && !errors.Is(err, tc.is) {
				t.Errorf("error %v is not %v", err, tc.is)
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			if after := ledgerFiles(t, stateDir); after != before {
				t.Errorf("a refused approve wrote to the ledger:\nbefore: %s\nafter:  %s", before, after)
			}
		})
	}
	for _, id := range []string{e1, e2} {
		if item := cliInboxItem(t, stateDir, id); item.State != org.InboxStateOpen {
			t.Errorf("item %s is %s; want it still open", id, item.State)
		}
	}

	if _, _, err := runOrgCmdStreams(t, "approve", "--state-dir", stateDir); err == nil || !strings.Contains(err.Error(), "requires at least 1 arg(s)") {
		t.Errorf("approve with no ID = %v; want cobra's argument count error", err)
	}
}

// approve refuses --org-id as the inbox verbs do, before it reads or writes
// the ledger: each item names its org.
func TestOrgApprove_RefusesOrgID(t *testing.T) {
	approveEnv(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	ref := writeWorktreePlan(t, seedApprovalOrg(t, stateDir, "a", "per-plan"), approveTestPlanRel, approveTestPlanDraft)
	id := escalateOK(t, stateDir, "a", planQuestion(ref))
	before := ledgerFiles(t, stateDir)

	stdout, _, err := runOrgCmdStreams(t, "approve", id, "--org-id", "a", "--state-dir", stateDir)
	if want := "org: approve: --org-id does not apply: the inbox holds the items of every org in the ledger"; err == nil || err.Error() != want {
		t.Errorf("approve --org-id = %v; want %q", err, want)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if after := ledgerFiles(t, stateDir); after != before {
		t.Errorf("a refused approve wrote to the ledger:\nbefore: %s\nafter:  %s", before, after)
	}
}

// The DECISION and resolve commands repeat --state-dir when approve was
// given it, and not when the ledger came from the personal file's
// [org].state_dir (the rule of orgInboxHintStateDir).
func TestOrgApprove_TheDecisionCommandRepeatsAnExplicitStateDir(t *testing.T) {
	for _, explicit := range []bool{true, false} {
		t.Run(map[bool]string{true: "flag", false: "config"}[explicit], func(t *testing.T) {
			approveEnv(t)
			stateDir := filepath.Join(t.TempDir(), "state")
			ref := writeWorktreePlan(t, seedApprovalOrg(t, stateDir, "a", "per-plan"), approveTestPlanRel, approveTestPlanDraft)
			id := escalateOK(t, stateDir, "a", planQuestion(ref))
			ledger, flag := []string{"--state-dir", stateDir}, " --state-dir '"+stateDir+"'"
			if !explicit {
				usePersonalStateDir(t, stateDir)
				ledger, flag = nil, ""
			}

			stdout, _ := runInboxOK(t, append([]string{"approve", id}, ledger...)...)
			if want := wantApprovedBlock(id, "a", ref, flag); !strings.HasPrefix(stdout, want) {
				t.Errorf("stdout =\n%s\nwant it to start with\n%s", stdout, want)
			}
		})
	}
}

// approve writes nothing to the inbox, so with an inbox file it cannot write
// it still records the approval, exits 0, and prints the whole block; 7a's
// approve failed here on its resolve.
func TestOrgApprove_PassesWithAnInboxItCannotWrite(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes a read-only inbox file, so the inbox cannot be made unwritable")
	}
	approveEnv(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	ref := writeWorktreePlan(t, seedApprovalOrg(t, stateDir, "a", "per-plan"), approveTestPlanRel, approveTestPlanDraft)
	id := escalateOK(t, stateDir, "a", planQuestion(ref))
	inbox := org.InboxPathIn(stateDir)
	if err := os.Chmod(inbox, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(inbox, 0o644) })

	stdout, stderr, err := runOrgCmdStreams(t, "approve", id, "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("approve with an inbox it cannot write: %v (stdout: %s stderr: %s)", err, stdout, stderr)
	}
	if want := wantApprovedBlock(id, "a", ref, " --state-dir '"+stateDir+"'") + "\n" + approveRecordedNote + "\n"; stdout != want {
		t.Errorf("stdout =\n%s\nwant\n%s", stdout, want)
	}
	if approvals := planApprovedEvents(t, stateDir); len(approvals) != 1 {
		t.Errorf("plan_approved events = %+v; want one", approvals)
	}
	if item := cliInboxItem(t, stateDir, id); item.State != org.InboxStateOpen {
		t.Errorf("item %s is %s; want it still open", id, item.State)
	}
}

// AC10: a second approve of an item whose approval is recorded is refused
// with exit 1, records nothing, prints nothing on stdout, and repeats the
// send command and then the resolve command, with the --state-dir it was
// given. Once the item is resolved, approve refuses it as a resolved item.
func TestOrgApprove_ASecondApproveOfTheItemIsRefused(t *testing.T) {
	approveEnv(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	ref := writeWorktreePlan(t, seedApprovalOrg(t, stateDir, "a", "per-plan"), approveTestPlanRel, approveTestPlanDraft)
	id := escalateOK(t, stateDir, "a", planQuestion(ref))
	runInboxOK(t, "approve", id, "--state-dir", stateDir)
	before := ledgerFiles(t, stateDir)

	stdout, _, err := runOrgCmdStreams(t, "approve", id, "--state-dir", stateDir)
	if !errors.Is(err, org.ErrPlanAlreadyApproved) {
		t.Fatalf("the second approve = %v; want ErrPlanAlreadyApproved (stdout: %s)", err, stdout)
	}
	flag := " --state-dir '" + stateDir + "'"
	block := wantApprovedBlock(id, "a", ref, flag)
	send := block[strings.Index(block, "ralph org send ") : strings.Index(block, "approved'\n")+len("approved'")]
	resolve := "ralph org inbox resolve " + id + flag + " --note 'approved " + ref.String() + "'"
	if want := "org: approve " + id + `: the approval of this inbox item is already recorded (org_id "a", plan ` + ref.String() + "), " +
		"so approve records nothing more for it; tell the leader with:\n" + send + "\n" +
		"and once that send succeeded, resolve the item with:\n" + resolve; !strings.HasSuffix(err.Error(), "\n"+want) {
		t.Errorf("error =\n%v\nwant it to end with\n%s", err, want)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if after := ledgerFiles(t, stateDir); after != before {
		t.Errorf("a refused approve wrote to the ledger:\nbefore: %s\nafter:  %s", before, after)
	}

	runInboxOK(t, "inbox", "resolve", id, "--note", "approved "+ref.String(), "--state-dir", stateDir)
	if _, _, err := runOrgCmdStreams(t, "approve", id, "--state-dir", stateDir); !errors.Is(err, org.ErrInboxResolved) {
		t.Errorf("approve of the resolved item = %v; want ErrInboxResolved", err)
	}
}

// An item someone else resolved in the meantime (ResolvedElsewhere) gets its
// approved line and send command but no resolve lines; the other items get
// theirs.
func TestPrintApprovedPlans_NoResolveCommandForAnItemResolvedElsewhere(t *testing.T) {
	refA := org.PlanRef{Path: approveTestPlanRel, Digest: "0123456789ab"}
	refB := org.PlanRef{Path: approveTestPlanRel, Digest: "ba9876543210"}
	var stdout, stderr bytes.Buffer
	printApprovedPlans(&stdout, &stderr, []org.ApprovedPlan{
		{ID: "e1", OrgID: "a", Plan: refA.Path, Digest: refA.Digest, RequiresApproval: true, ResolvedElsewhere: true},
		{ID: "e2", OrgID: "b", Plan: refB.Path, Digest: refB.Digest, RequiresApproval: true},
	}, "")
	full := wantApprovedBlock("e1", "a", refA, "")
	withoutResolve := full[:strings.Index(full, "once that send succeeded")]
	if want := withoutResolve + "\n" + wantApprovedBlock("e2", "b", refB, "") + "\n" + approveRecordedNote + "\n"; stdout.String() != want {
		t.Errorf("stdout =\n%s\nwant\n%s", stdout.String(), want)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q; want nothing", stderr.String())
	}
}

// AC4: with HERDR_PANE_ID set to the pane of an active seat, approve is
// refused and writes nothing; set to a pane no seat has, it passes and
// records herdr_pane=set.
func TestOrgApprove_RefusedFromAnActiveSeatsPane(t *testing.T) {
	approveEnv(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	ref := writeWorktreePlan(t, seedApprovalOrg(t, stateDir, "a", "per-plan"), approveTestPlanRel, approveTestPlanDraft)
	id := escalateOK(t, stateDir, "a", planQuestion(ref))
	before := ledgerFiles(t, stateDir)

	t.Setenv("HERDR_PANE_ID", "pane-a")
	stdout, _, err := runOrgCmdStreams(t, "approve", id, "--state-dir", stateDir)
	if want := `org: approve: refused, since HERDR_PANE_ID "pane-a" is the pane of the active seat "leader" of org_id "a"`; err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Errorf("approve from the leader's pane = %v; want it to start with %q", err, want)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if after := ledgerFiles(t, stateDir); after != before {
		t.Errorf("a refused approve wrote to the ledger:\nbefore: %s\nafter:  %s", before, after)
	}

	t.Setenv("HERDR_PANE_ID", "pane-of-a-human")
	stdout, _ = runInboxOK(t, "approve", id, "--state-dir", stateDir)
	if !strings.HasPrefix(stdout, "approved "+id+" (org=a ") {
		t.Errorf("stdout = %q; want the approved line", stdout)
	}
	if approvals := planApprovedEvents(t, stateDir); len(approvals) != 1 || !strings.Contains(approvals[0].Details, " herdr_pane=set ") {
		t.Errorf("plan_approved events = %+v; want one with herdr_pane=set", approvals)
	}
}

// An org started in split mode, or with no mode recorded (a start from before
// approval modes), does not wait for the approval: approve records it, exits
// 0, and says so on stderr, one note per such item.
func TestOrgApprove_AnOrgThatDoesNotWaitGetsANoteOnStderr(t *testing.T) {
	approveEnv(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	refS := writeWorktreePlan(t, seedApprovalOrg(t, stateDir, "s", "split"), approveTestPlanRel, approveTestPlanDraft)
	refOld := writeWorktreePlan(t, seedApprovalOrg(t, stateDir, "old", ""), approveTestPlanRel, approveTestPlanDraft)
	e1 := escalateOK(t, stateDir, "s", planQuestion(refS))
	e2 := escalateOK(t, stateDir, "old", planQuestion(refOld))

	stdout, stderr := runInboxOK(t, "approve", e1, e2, "--state-dir", stateDir)
	note := func(orgID, id string) string {
		return "note: org " + orgID + " was started with approval mode split (or by a ralph from before approval modes), " +
			"so its implementer does not wait for the approval of " + id + "; it is recorded all the same"
	}
	if got, want := outputLines(stderr), []string{note("s", e1), note("old", e2)}; !slices.Equal(got, want) {
		t.Errorf("stderr lines = %q\nwant %q", got, want)
	}
	for _, line := range []string{"approved " + e1 + " (org=s ", "approved " + e2 + " (org=old "} {
		if !strings.Contains(stdout, line) {
			t.Errorf("stdout does not have %q:\n%s", line, stdout)
		}
	}
	if approvals := planApprovedEvents(t, stateDir); len(approvals) != 2 {
		t.Errorf("plan_approved events = %+v; want both recorded", approvals)
	}
}

// approve changes the ledger, so it is refused while seats of an older,
// per-worktree ledger are active (guardLegacyOrgStateDir).
func TestOrgApprove_LegacyLedgerGuard(t *testing.T) {
	setupOrgStubPATH(t)
	r := newLegacyLedgerRepo(t)
	approveEnv(t)
	id := escalateOK(t, r.sharedDir, "org-a", inboxQuestion("seeded"))
	spawnLedgerSeat(t, r.legacyDir, "seat-1")
	before := readFileBytes(t, org.InboxPathIn(r.sharedDir))

	stdout, _, err := runOrgCmdStreams(t, "approve", id)
	if err == nil || !strings.Contains(err.Error(), "refusing to change the org ledger") {
		t.Errorf("approve error = %v; want the legacy-ledger refusal", err)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if after := readFileBytes(t, org.InboxPathIn(r.sharedDir)); !bytes.Equal(before, after) {
		t.Errorf("a refused approve wrote to the inbox:\nbefore: %s\nafter:  %s", before, after)
	}
}

// approve --help says what approve checks and records and what it never
// does; ralph org --help names approve, and --org-id's help lists it among
// the verbs that refuse the flag.
func TestOrgApproveHelp(t *testing.T) {
	out, err := runOrgCmd(t, "approve", "--help")
	if err != nil {
		t.Fatalf("approve --help: %v", err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"asks for the approval with an inbox QUESTION (`ralph org escalate`) whose header names the plan as PLAN: <path in the feature worktree> sha256:<digest>",
		"that its org holds the binding `ralph org start --plan` recorded and was not disbanded since",
		"stays inside the org's feature worktree (symlinks resolved), and is a regular file",
		"the digest equals the plan file's digest now",
		"Several IDs are all or nothing: when one item fails, nothing is recorded, every failure is listed",
		"with who ran it (USER, whether HERDR_PANE_ID was set, and the current directory)",
		"It records the plan file as it is now",
		"approve only records: it never edits the plan file, never messages the leader, and never resolves the inbox item",
		"it prints the `ralph org send` command whose DECISION tells that org's leader, and then the `ralph org inbox resolve` command to run once that send succeeded. Run them in that order",
		"the ack and resolve deadlines of `ralph org watch --all` can still bring it to a human",
		"A second approve of an item whose approval is recorded is refused, with the two commands printed again",
		"approve is refused when HERDR_PANE_ID is the pane of an active seat",
		"so --org-id is refused",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("approve --help does not say %q:\n%s", want, out)
		}
	}

	out, err = runOrgCmd(t, "--help")
	if err != nil {
		t.Fatalf("org --help: %v", err)
	}
	help = strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"`ralph org approve` records a human's approval of the plan a feature org's leader asked for there",
		"wait --inbox, watch, approve, the director verbs, and the inbox verbs, which refuse it",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("org --help does not say %q:\n%s", want, out)
		}
	}
}

// TestOrgSendSpawn_ApprovalModeIsTheOneRecordedAtStart is AC13 through the
// CLI, with the real start: the main checkout's ralph.toml gives the mode
// start records, and then the feature worktree's ralph.toml (the leader's
// copy), a subdirectory's, and a --config file all say the other mode. In an
// org started per-plan, a dry-run TASK to the implementer and a dry-run
// implementer spawn are refused from each of them until a human approves the
// plan with `ralph org approve` and the leader writes the two header lines,
// and pass from each after; in an org started split, they pass from each with
// no approval. The implementer seat is seeded, since its spawn is what the
// check refuses.
func TestOrgSendSpawn_ApprovalModeIsTheOneRecordedAtStart(t *testing.T) {
	modeToml := func(mode string) string { return "[org.approval]\nmode = \"" + mode + "\"\n" }
	for _, tc := range []struct{ started, caller string }{{"per-plan", "split"}, {"split", "per-plan"}} {
		t.Run("started "+tc.started, func(t *testing.T) {
			setupOrgStubPATH(t)
			approveEnv(t)
			root := newFeatureRepo(t, ".claude/worktrees/\n.harness/\n")
			writeTestFile(t, filepath.Join(root, "ralph.toml"), modeToml(tc.started))
			runLedgerTestGit(t, "-C", root, "add", "ralph.toml")
			runLedgerTestGit(t, "-C", root, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false",
				"commit", "--quiet", "-m", "ralph.toml")
			t.Chdir(root)
			stateDir := filepath.Join(root, ".harness", "state", "org")
			split := writeFeatureSplitPlan(t, stateDir, approvedFeatureSplitPlan(t))
			if stdout, stderr, err := startOrgFeature(t, split, "a"); err != nil {
				t.Fatalf("start --plan --feature a: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			if r := scopeReservedEvents(t, stateDir, "a"); len(r) != 1 || !strings.HasSuffix(r[0].Details, " approval="+tc.started) {
				t.Fatalf("a's scope_reserved = %+v; want one ending with approval=%s", r, tc.started)
			}

			wt := filepath.Join(root, ".claude", "worktrees", "org-a")
			writeTestFile(t, filepath.Join(wt, "ralph.toml"), modeToml(tc.caller))
			sub := filepath.Join(root, "pkg")
			if err := os.MkdirAll(sub, 0o755); err != nil {
				t.Fatal(err)
			}
			writeTestFile(t, filepath.Join(sub, "ralph.toml"), modeToml(tc.caller))
			cfg := filepath.Join(t.TempDir(), "ralph.toml")
			writeTestFile(t, cfg, modeToml(tc.caller))
			callers := []struct {
				name, dir string
				extra     []string
			}{
				{"from the feature worktree", wt, nil},
				{"from a subdirectory of the main checkout", sub, nil},
				{"with --config", root, []string{"--config", cfg}},
			}

			appendSeedEvent(t, stateDir, org.ManifestEvent{
				OrgID: "a", SeatID: "implementer", Event: org.EventSpawned, Role: "implementer",
				Driver: "claude", Model: "sonnet", PaneID: "pane-implementer", AgmsgTeam: "ralph-a", Worktree: wt,
			})
			ref := writeWorktreePlan(t, wt, approveTestPlanRel, approveTestPlanDraft)
			task := "TYPE: TASK\nTASK_ID: t-1\nPLAN: " + ref.String() + "\n\nimplement the plan"
			send := func(t *testing.T, dir string, extra []string) error {
				t.Helper()
				t.Chdir(dir)
				_, _, err := runOrgCmdStreams(t, append([]string{"send", "--org-id", "a", "--to", "implementer", "--text", task, "--dry-run"}, extra...)...)
				return err
			}
			spawn := func(t *testing.T, dir string, extra []string) (string, error) {
				t.Helper()
				t.Chdir(dir)
				stdout, _, err := runOrgCmdStreams(t, append([]string{"spawn", "--org-id", "a", "--id", "implementer", "--role", "implementer",
					"--driver", "claude", "--model", "sonnet", "--cwd", wt, "--scope", "the approved plan", "--dry-run"}, extra...)...)
				return stdout, err
			}
			expect := func(t *testing.T, refused bool, when string) {
				t.Helper()
				for _, c := range callers {
					err := send(t, c.dir, c.extra)
					if refused && !errors.Is(err, org.ErrPlanNotApproved) {
						t.Errorf("%s, %s: send = %v; want the approval check's refusal", when, c.name, err)
					} else if !refused && err != nil {
						t.Errorf("%s, %s: send = %v; want it sent", when, c.name, err)
					}
					stdout, err := spawn(t, c.dir, c.extra)
					if refused && (!errors.Is(err, org.ErrPlanNotApproved) || !strings.HasPrefix(stdout, "rejected: ")) {
						t.Errorf("%s, %s: spawn = %v (stdout %q); want rejected by the approval check", when, c.name, err, stdout)
					} else if !refused && err != nil {
						t.Errorf("%s, %s: spawn = %v (stdout %q); want it to pass", when, c.name, err, stdout)
					}
				}
			}
			if tc.started == "split" {
				expect(t, false, "with no approval")
				return
			}

			expect(t, true, "before the approval")
			t.Chdir(root)
			id := escalateOK(t, stateDir, "a", planQuestion(ref))
			stdout, _ := runInboxOK(t, "approve", id)
			if want := wantApprovedBlock(id, "a", ref, ""); !strings.HasPrefix(stdout, want) {
				t.Errorf("approve stdout =\n%s\nwant it to start with\n%s", stdout, want)
			}
			if err := send(t, wt, nil); !errors.Is(err, org.ErrPlanNotApproved) || !strings.Contains(err.Error(), "is not marked approved") {
				t.Errorf("send before the leader marks the plan = %v; want the refusal for its - Status: line", err)
			}
			writeTestFile(t, filepath.Join(wt, filepath.FromSlash(approveTestPlanRel)), markPlanApproved(approveTestPlanDraft, ref.Digest))
			expect(t, false, "after the approval")
		})
	}
}
