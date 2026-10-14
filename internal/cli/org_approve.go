package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// newOrgApproveCmd wires `ralph org approve <inbox ID>...` (plan
// docs/plans/active/2026-10-12-org-plan-approval.md, A2, A3, A4 and A9):
// (*org.Org).Approve checks every item and records the approvals all or
// nothing, and refuses an item whose approval is already recorded. It
// records only and resolves no item (D7 of plan
// 2026-10-14-org-director-attach), so an item stays open or acked, and under
// the inbox deadlines, until the leader has been told. This command prints,
// for each approved item, its line, the `ralph org send` DECISION that tells
// the item's org's leader (org.ApprovalDecisionCommand), which approve never
// sends itself, and then the resolve to run once that send succeeded
// (org.ApprovalResolveCommand). Each item names its org, so --org-id is
// refused as the inbox verbs refuse it (newOrgInboxRuntime), which also sets
// InboxHintStateDir for the printed commands.
func newOrgApproveCmd(stateDir, configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "approve <inbox ID>...",
		Short: "Record a human's approval of the plans feature org leaders asked for in the inbox",
		Long: "ralph org approve records a human's approval of a feature plan. The leader\n" +
			"of an org started with `ralph org start --plan` writes the plan in the\n" +
			"org's feature worktree and asks for the approval with an inbox QUESTION\n" +
			"(`ralph org escalate`) whose header names the plan as\n" +
			"PLAN: <path in the feature worktree> sha256:<digest>. Pass the IDs of\n" +
			"those inbox items (e1, e2, ...).\n" +
			"\n" +
			"For each item, approve checks that it is an open or acked QUESTION with\n" +
			"exactly one PLAN: header of that shape; that its org holds the binding\n" +
			"`ralph org start --plan` recorded and was not disbanded since; that the\n" +
			"path is relative, stays inside the org's feature worktree (symlinks\n" +
			"resolved), and is a regular file; and that the digest equals the plan\n" +
			"file's digest now. Several IDs are all or nothing: when one item fails,\n" +
			"nothing is recorded, every failure is listed, and approve exits 1.\n" +
			"\n" +
			"When every item passes, approve records a plan_approved event per item\n" +
			"in the manifest, with who ran it (USER, whether HERDR_PANE_ID was set,\n" +
			"and the current directory). It records the plan file as it is now:\n" +
			"from then on, the leader may change only the plan's - Status: and\n" +
			"- Approved: lines and its checkbox marks, and any other change refuses\n" +
			"the implementer's spawn and TASKs until the plan is approved again.\n" +
			"\n" +
			"approve only records: it never edits the plan file, never messages the\n" +
			"leader, and never resolves the inbox item. For each item it prints the\n" +
			"`ralph org send` command whose DECISION tells that org's leader, and\n" +
			"then the `ralph org inbox resolve` command to run once that send\n" +
			"succeeded. Run them in that order: until the item is resolved, the ack\n" +
			"and resolve deadlines of `ralph org watch --all` can still bring it to\n" +
			"a human, so an approval whose DECISION was never sent does not leave\n" +
			"the leader waiting unseen. A second approve of an item whose approval\n" +
			"is recorded is refused, with the two commands printed again.\n" +
			"\n" +
			"approve is refused when HERDR_PANE_ID is the pane of an active seat, as\n" +
			"when a seat runs it from its own herdr pane: a human runs it from a\n" +
			"terminal outside the seats' panes. The inbox is shared by every org in\n" +
			"the ledger and each item names its org, so --org-id is refused. An org\n" +
			"started in approval mode split, or by a ralph from before approval\n" +
			"modes, does not wait for the approval; approve records it all the same\n" +
			"and says so on stderr.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := newOrgInboxRuntime(cmd, "approve", *stateDir, *configPath, orgLedgerMutating)
			if err != nil {
				return err
			}
			// Cwd is left empty: Approve records os.Getwd(), and cwd="" when
			// that fails (the directory is for an audit only).
			res := rt.Approve(org.ApproveParams{IDs: args})
			// res.Approved is what the manifest holds, also when res.Err is
			// set (an append that failed after a record, or the inbox read
			// again after the records), so those items are printed before
			// the error. A refusal records nothing and prints nothing on
			// stdout. An item someone else resolved in the meantime is
			// printed with its DECISION command too, since its approval is
			// recorded, but with no resolve command; res.Err, which goes to
			// stderr after it, says not to send that DECISION before reading
			// the note the item was resolved with, and names the
			// `ralph org inbox show` command for it (org's
			// checkResolvedMeanwhile).
			printApprovedPlans(cmd.OutOrStdout(), cmd.ErrOrStderr(), res.Approved, rt.InboxHintStateDir)
			return res.Err
		},
	}
}

// approveRecordedNote follows the approved items on stdout, once (A9 of plan
// 2026-10-12-org-plan-approval): the record holds a hash of the whole plan
// file, so the human, who tells the leader, knows which edits the approval
// survives. It goes to stdout with the approved lines, since it says what
// was just recorded; stderr carries the notes on an item that differs from
// that result (an org that does not wait for the approval) and the errors.
const approveRecordedNote = "note: approve recorded each plan file as it is now. From here on the leader may change only " +
	"the plan's - Status: and - Approved: lines and its checkbox marks: any other change makes a per-plan org " +
	"refuse its implementer's spawn and TASKs until the plan is approved again."

// printApprovedPlans prints, for each approved item, a block on stdout: the
// line `approved <ID> (org=<org> plan=<path> sha256:<digest>)`, a line that
// says approve sends no message and resolves nothing, the `ralph org send`
// command whose DECISION tells the org's leader
// (org.ApprovalDecisionCommand), a line that says to resolve the item once
// that send succeeded, and the `ralph org inbox resolve` command that does
// (org.ApprovalResolveCommand), both commands repeating --state-dir
// hintStateDir when it is set. Each command starts its own line and the
// send's --text spans several, so blocks are separated by a blank line and
// each starts with its approved line. An item someone else resolved in the
// meantime (ResolvedElsewhere) gets no resolve lines: nothing is left to
// resolve, and the error after the block says what to do. approveRecordedNote
// follows the last block. An item whose org does not wait for the approval
// (RequiresApproval false) also gets a note on stderr. With no item, it
// prints nothing.
func printApprovedPlans(stdout, stderr io.Writer, approved []org.ApprovedPlan, hintStateDir string) {
	for i, a := range approved {
		if i > 0 {
			_, _ = fmt.Fprintln(stdout)
		}
		ref := org.PlanRef{Path: a.Plan, Digest: a.Digest}
		id, orgID := inboxField(a.ID), inboxField(a.OrgID)
		_, _ = fmt.Fprintf(stdout, "approved %s (org=%s plan=%s)\n", id, orgID, ref)
		_, _ = fmt.Fprintf(stdout, "approve sends no message and resolves nothing; tell the leader of org %s with:\n%s\n",
			orgID, org.ApprovalDecisionCommand(a.OrgID, ref, hintStateDir))
		if !a.ResolvedElsewhere {
			_, _ = fmt.Fprintf(stdout, "once that send succeeded, resolve the item with:\n%s\n",
				org.ApprovalResolveCommand(a.ID, ref, hintStateDir))
		}
		if !a.RequiresApproval {
			_, _ = fmt.Fprintf(stderr, "note: org %s was started with approval mode split (or by a ralph from before approval modes), "+
				"so its implementer does not wait for the approval of %s; it is recorded all the same\n", orgID, id)
		}
	}
	if len(approved) > 0 {
		_, _ = fmt.Fprintf(stdout, "\n%s\n", approveRecordedNote)
	}
}
