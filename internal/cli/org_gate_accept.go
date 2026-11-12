package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// newOrgGateAcceptCmd wires `ralph org gate accept --branch <b> --log <path>
// --reason <text>` (plan docs/plans/active/2026-11-12-gate-lock-accept.md,
// AC7, D3, D5): (*org.Org).GateAccept records, under the manifest lock, a
// human's acceptance of the failure the branch's latest gate record holds, as
// gate_accepted, and ralph org merge-order then lists the branch as accepted
// where a pass would be ready, and ralph org merge merges it. The acceptance
// names its org from the gate record, so --org-id, a persistent flag of
// `ralph org`, is refused before anything is read, as ralph org merge-order
// refuses it; so are an empty --branch, --log, or --reason
// (org.GateAcceptParamsRefusal). This command prints the recorded acceptance
// (printGateAcceptance), and a refusal is its error (exit 1).
func newOrgGateAcceptCmd(stateDir, configPath *string) *cobra.Command {
	var p org.GateAcceptParams

	cmd := &cobra.Command{
		Use:   "accept",
		Short: "Record a human's acceptance of a feature branch's latest gate failure",
		Long: "ralph org gate accept records that a human accepted the failure of a\n" +
			"feature branch's latest gate record: a gate_failed or gate_ci_failed\n" +
			"that ralph org gate or ralph org regate recorded with the reason\n" +
			"static_failed, test_failed, or ci_failed, where a script or a check\n" +
			"failed, for example a test that failed because the machine was\n" +
			"loaded, outside the branch's change. It records gate_accepted in the\n" +
			"manifest, under the org of that gate record, with the record's head,\n" +
			"base, base_ref, branch, event, reason, time, and log, who ran it\n" +
			"(by=<USER>, and herdr_pane=set or unset for whether HERDR_PANE_ID was\n" +
			"set), and the --reason text.\n" +
			"\n" +
			"All three flags are required. --branch names the feature branch as\n" +
			"ralph org merge-order lists it. --log names the gate log of the\n" +
			"failure the human read: the LOG: line ralph org gate printed, or the\n" +
			"gate's log in ralph org merge-order --json (each gate run writes a log\n" +
			"of its own, so the log names one failure). --reason says in one line,\n" +
			"of at most 500 characters and with no control characters, why the\n" +
			"human accepts the failure.\n" +
			"\n" +
			"A human runs it, or a director the human told to: it records a human's\n" +
			"decision, and the director does not accept a failure on its own. It is\n" +
			"refused when HERDR_PANE_ID is the pane of an active seat, as when a seat\n" +
			"runs it from its own herdr pane, as ralph org approve is.\n" +
			"\n" +
			"Under the manifest lock, gate accept reads the ledger, finds the\n" +
			"branch's latest gate record as merge-order does, and records the\n" +
			"acceptance in the same lock hold, so two accepts of the same failure\n" +
			"record one. It records nothing, says why on stderr, and exits 1 when:\n" +
			"no gate record names the branch; the latest gate record cannot be read,\n" +
			"or a corrupt manifest line follows it; it is a pass; its reason is\n" +
			"head_moved or tree_changed (the scripts passed: run the gate again);\n" +
			"its log is not --log (a newer gate replaced the failure the human read:\n" +
			"read the latest failure's log, which the refusal names, and ask the\n" +
			"human again); that failure is already accepted; a flag is empty;\n" +
			"--org-id is given; or HERDR_PANE_ID is the pane of an active seat.\n" +
			"\n" +
			"On success it prints ACCEPTED: <branch>, FAILED: <event> (<reason>) at\n" +
			"<time>, HEAD:, BASE:, LOG:, BY:, and NOTE: <the --reason text> on\n" +
			"stdout and exits 0. Then ralph org merge-order classifies the branch as\n" +
			"it would classify a pass, and lists it as accepted, right after the\n" +
			"ready ones, where a pass would be ready (rebase, wrong_base, regate, and\n" +
			"no_remote stay as they are), with the acceptance in its --json gate;\n" +
			"and ralph org merge merges an accepted branch with the same checks as a\n" +
			"ready one, prints GATE: accepted, and records gate=accepted in\n" +
			"pr_merged. Any gate record of the branch after the acceptance voids it,\n" +
			"so a gate or regate run again, on the same head or another, needs a new\n" +
			"acceptance when it fails again. The acceptance names the head the gate\n" +
			"ran on, so a failure regate recorded on the merge commit it made and\n" +
			"undid stays regate in merge-order.\n" +
			"\n" +
			"gate accept starts no autonomous work, so it needs no running watch and\n" +
			"no approved plan.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("org-id") {
				return errors.New("org: gate accept: --org-id does not apply: gate accept records the acceptance under the org " +
					"of the branch's latest gate record")
			}
			if refused := org.GateAcceptParamsRefusal(p); refused != "" {
				return errors.New(refused)
			}
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath, orgLedgerMutating)
			if err != nil {
				return err
			}
			res, err := rt.GateAccept(p)
			switch {
			case err != nil:
				return err
			case res.Refused != "":
				return errors.New(res.Refused)
			case res.Accepted == nil:
				return errors.New("org: gate accept: ralph org gate accept ended with no outcome, so whether the acceptance " +
					"was recorded is unknown: ralph org merge-order --json shows the branch's gate and its acceptance")
			}
			printGateAcceptance(cmd.OutOrStdout(), *res.Accepted)
			return nil
		},
	}

	cmd.Flags().StringVar(&p.Branch, "branch", "", "the feature branch whose latest gate failure the human accepts, as ralph "+
		"org merge-order lists it (required)")
	cmd.Flags().StringVar(&p.Log, "log", "", "the gate log of that failure, which the human read: the LOG: line of ralph org "+
		"gate, or the gate's log in ralph org merge-order --json (required)")
	cmd.Flags().StringVar(&p.Reason, "reason", "", "why the human accepts the failure, in one line of at most 500 characters "+
		"(required)")

	return cmd
}

// printGateAcceptance prints the acceptance a ralph org gate accept
// recorded, one line each, every value escaped (inboxField):
//
//	ACCEPTED: <branch>
//	FAILED: <event> (<reason>) at <the failure's ts>
//	HEAD: <sha>
//	BASE: <sha>
//	LOG: <path>
//	BY: <by>
//	NOTE: <the --reason text>
func printGateAcceptance(w io.Writer, a org.GateAcceptance) {
	_, _ = fmt.Fprintf(w, "ACCEPTED: %s\n", inboxField(a.Branch))
	_, _ = fmt.Fprintf(w, "FAILED: %s (%s) at %s\n", inboxField(a.FailedEvent), inboxField(a.FailedReason), inboxField(a.FailedTS))
	_, _ = fmt.Fprintf(w, "HEAD: %s\n", inboxField(a.Head))
	_, _ = fmt.Fprintf(w, "BASE: %s\n", inboxField(a.Base))
	_, _ = fmt.Fprintf(w, "LOG: %s\n", inboxField(a.FailedLog))
	_, _ = fmt.Fprintf(w, "BY: %s\n", inboxField(a.By))
	_, _ = fmt.Fprintf(w, "NOTE: %s\n", inboxField(a.Note))
}
