package cli

import (
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// newOrgFollowupCmd wires `ralph org followup` and its verbs add, list, and
// done (issue #270; plan docs/plans/active/2026-10-30-org-followups.md, D3
// and D5). They call the follow-up store (internal/org/followup.go)
// directly, built by newOrgFollowupStore. list only reads the ledger; add
// and done change it. None of them reads ralph.toml.
func newOrgFollowupCmd(stateDir *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "followup",
		Short: "Record issues to ask the human about once another branch's PR is merged",
		Long: "ralph org followup keeps the follow-ups of the ledger in\n" +
			"<state-dir>/followups.jsonl. A follow-up is an issue the director asks\n" +
			"the human about only once the PR of another branch, its after-branch,\n" +
			"is merged. add records one, list shows them, and done marks one done\n" +
			"once the human was asked or it is no longer needed. A follow-up belongs\n" +
			"to the whole ledger, not to one org, so these verbs take no --org-id.\n" +
			"`ralph status` (from the ledger's records of merges) and\n" +
			"`ralph org merge-order` (from the PRs, with gh) show which open\n" +
			"follow-ups are ready; when one says ready and the other waiting, use\n" +
			"merge-order's answer, and when merge-order says unknown, go by\n" +
			"status's answer and give merge-order's reason.",
		Args: cobra.NoArgs,
		// Runnable, so a mistyped verb is an unknown command (exit 1), as for
		// `ralph org director`, instead of this help.
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(
		newOrgFollowupAddCmd(stateDir),
		newOrgFollowupListCmd(stateDir),
		newOrgFollowupDoneCmd(stateDir),
	)

	return cmd
}

func newOrgFollowupAddCmd(stateDir *string) *cobra.Command {
	var (
		issue       int
		afterBranch string
		note        string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Record a follow-up: an issue to ask about once a branch's PR is merged",
		Long: fmt.Sprintf("ralph org followup add records a follow-up in the ledger and prints its\n"+
			"ID (f1, f2, ...): issue --issue, which the director asks the human\n"+
			"about only once the PR of --after-branch is merged, with an optional\n"+
			"--note. --issue is a positive number. --after-branch is not empty,\n"+
			"holds no whitespace or control character, and does not start with -;\n"+
			"it is matched as a string against the branch the ledger records and\n"+
			"the head branch of the PRs. --note is one line, at most %d characters,\n"+
			"with no control characters. Input that fails these checks is refused\n"+
			"and nothing is written.\n"+
			"\n"+
			"`ralph status` (from the ledger's pr_merged and worktree_cleaned\n"+
			"records) and `ralph org merge-order` (from the PR, with gh) show the\n"+
			"follow-up as ready once the after-branch's PR is merged after the\n"+
			"follow-up was recorded. A merge from before it does not count, since a\n"+
			"branch name can be used again. So a branch whose PR is already merged\n"+
			"needs no follow-up: ask the human about the issue now instead. One case\n"+
			"still differs: a PR merged on GitHub before the follow-up was recorded\n"+
			"and cleaned up with `ralph org cleanup` after it shows as ready in\n"+
			"`ralph status` (from its worktree_cleaned) and as waiting in `ralph org\n"+
			"merge-order`; when one says ready and the other waiting, use\n"+
			"merge-order's answer, and when merge-order says unknown, go by status's\n"+
			"answer and give merge-order's reason.",
			org.InboxNoteMaxRunes),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, flag := range []string{"issue", "after-branch"} {
				if !cmd.Flags().Changed(flag) {
					return fmt.Errorf("org: followup add: --%s is required", flag)
				}
			}
			in := org.FollowupAdd{Issue: issue, AfterBranch: afterBranch, Note: note}
			// Checked before the ledger is resolved, so refused input is
			// reported as such before any note or refusal about the ledger;
			// Add checks it again.
			if err := org.ValidateFollowupAdd(in); err != nil {
				return err
			}
			store, err := newOrgFollowupStore(cmd, "followup add", *stateDir, orgLedgerMutating)
			if err != nil {
				return err
			}
			id, err := store.Add(in)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "added %s: issue #%d after %s\n", id, in.Issue, in.AfterBranch)
			return nil
		},
	}

	cmd.Flags().IntVar(&issue, "issue", 0, "number of the issue to ask the human about: a positive number (required)")
	cmd.Flags().StringVar(&afterBranch, "after-branch", "",
		"branch whose PR is merged first: not empty, no whitespace or control character, not starting with - (required)")
	cmd.Flags().StringVar(&note, "note", "", fmt.Sprintf(
		"optional note: one line, at most %d characters, no control characters", org.InboxNoteMaxRunes))

	return cmd
}

func newOrgFollowupListCmd(stateDir *string) *cobra.Command {
	var all, jsonOut bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List the follow-ups of the ledger",
		Long: "ralph org followup list lists the follow-ups recorded with\n" +
			"`ralph org followup add`: open ones by default, done ones too with\n" +
			"--all, as JSON with --json. NOTE is the note given to add; ADDED_AT is\n" +
			"when add recorded it. With --all, DONE_AT (when done recorded it) and\n" +
			"DONE_NOTE (done's note) follow, both - for an open follow-up; --json\n" +
			"has them as done_at and done_note on a done one. Control characters in\n" +
			"the values are printed as escapes (\\x1b). list shows only the record,\n" +
			"not whether a follow-up is ready: `ralph status` says that from the\n" +
			"ledger's records of merges, and `ralph org merge-order` from the\n" +
			"after-branch's PR, with gh.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := newOrgFollowupStore(cmd, "followup list", *stateDir, orgLedgerReadOnly)
			if err != nil {
				return err
			}
			res, err := store.Read()
			if err != nil {
				return err
			}
			warnFollowupDamage(cmd.ErrOrStderr(), store.Path(), res)
			items := make([]org.Followup, 0, len(res.Items))
			for _, item := range res.Items {
				if all || item.State != org.FollowupStateDone {
					items = append(items, item)
				}
			}
			if jsonOut {
				return writeIndentedJSON(cmd.OutOrStdout(), orgFollowupJSON{
					Items: items, CorruptLines: res.CorruptLines, IgnoredEvents: res.IgnoredEvents,
				})
			}
			printFollowupTable(cmd.OutOrStdout(), items, all)
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "include done follow-ups")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "machine-readable JSON output")

	return cmd
}

func newOrgFollowupDoneCmd(stateDir *string) *cobra.Command {
	var note string

	cmd := &cobra.Command{
		Use:   "done <id>",
		Short: "Mark an open follow-up done",
		Long: fmt.Sprintf("ralph org followup done marks an open follow-up done: the human was\n"+
			"asked about its issue, or it is no longer needed. --note is optional: the\n"+
			"outcome, such as the decision or a PR URL, on one line, at most %d\n"+
			"characters, with no control characters. On a follow-up that is already\n"+
			"done it changes nothing and exits 0, and the first done's note stays; an\n"+
			"unknown ID exits 1.",
			org.InboxNoteMaxRunes),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Checked before the ledger is resolved, as add checks its input;
			// Done checks it again.
			if err := org.ValidateFollowupNote(note); err != nil {
				return err
			}
			store, err := newOrgFollowupStore(cmd, "followup done", *stateDir, orgLedgerMutating)
			if err != nil {
				return err
			}
			changed, err := store.Done(args[0], note)
			if err != nil {
				return err
			}
			if changed {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "done %s\n", args[0])
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s is already done; nothing changed\n", args[0])
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&note, "note", "", fmt.Sprintf(
		"optional outcome, such as the decision or a PR URL: one line, at most %d characters", org.InboxNoteMaxRunes))

	return cmd
}

// newOrgFollowupStore returns the follow-up store of the ledger for the
// `ralph org followup` verbs, as newOrgInboxRuntime builds the inbox's: a
// follow-up belongs to the whole ledger, so an --org-id is refused rather
// than ignored, and the ledger is resolved and guarded by resolveOrgLedger
// with the verb's access. The verbs read no ralph.toml, so a ralph.toml that
// does not load does not block them, and the store is returned alone, as
// the director verbs keep theirs in orgDirectorRuntime, not in an org.Org.
func newOrgFollowupStore(cmd *cobra.Command, verb, stateDir string, access orgLedgerAccess) (*org.FollowupStore, error) {
	if cmd.Flags().Changed("org-id") {
		return nil, fmt.Errorf("org: %s: --org-id does not apply: a follow-up belongs to the whole ledger, not to one org", verb)
	}
	resolvedStateDir, _, err := resolveOrgLedger(cmd, stateDir, access)
	if err != nil {
		return nil, err
	}
	return org.NewFollowupStore(resolvedStateDir), nil
}

// orgFollowupJSON is the --json wire shape for `ralph org followup list`:
// the listed follow-ups (org.Followup carries its own json tags) and the
// record's damage counts.
type orgFollowupJSON struct {
	Items         []org.Followup `json:"items"`
	CorruptLines  int            `json:"corrupt_lines"`
	IgnoredEvents int            `json:"ignored_events"`
}

// warnFollowupDamage writes to w (stderr, so --json output stays clean) the
// lines of the record at path that did not parse and the events the fold
// ignored (org.FollowupReadResult).
func warnFollowupDamage(w io.Writer, path string, res org.FollowupReadResult) {
	if res.CorruptLines > 0 {
		_, _ = fmt.Fprintf(w, "warning: %d unreadable follow-up line(s) skipped in %s\n", res.CorruptLines, path)
	}
	if res.IgnoredEvents > 0 {
		_, _ = fmt.Fprintf(w, "warning: %d follow-up event(s) ignored in %s (unknown kind or ID, a second added, an added event that "+
			"add would refuse, or a done with no open follow-up)\n", res.IgnoredEvents, path)
	}
}

// printFollowupTable prints `ralph org followup list`'s tab-separated table,
// or one line saying the list is empty. With all, the done follow-ups are
// listed too, so two columns follow NOTE: DONE_AT and DONE_NOTE, the done's
// time and note. They are two columns, not one, so each stays a value of its
// own that a reader can split on the tab: a note can hold any separator but
// a control character. Without all every item is open, so they are left out.
// Each value is escaped as inboxField escapes it, "-" when empty.
func printFollowupTable(w io.Writer, items []org.Followup, all bool) {
	if len(items) == 0 {
		if all {
			_, _ = fmt.Fprintln(w, "no follow-ups")
		} else {
			_, _ = fmt.Fprintln(w, "no open follow-ups")
		}
		return
	}
	header := "ID\tSTATE\tISSUE\tAFTER_BRANCH\tADDED_AT\tNOTE"
	if all {
		header += "\tDONE_AT\tDONE_NOTE"
	}
	_, _ = fmt.Fprintln(w, header)
	for _, item := range items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s",
			inboxField(item.ID), inboxField(item.State), strconv.Itoa(item.Issue), inboxField(item.AfterBranch),
			inboxField(item.AddedAt), inboxField(item.Note))
		if all {
			_, _ = fmt.Fprintf(w, "\t%s\t%s", inboxField(item.DoneAt), inboxField(item.DoneNote))
		}
		_, _ = fmt.Fprintln(w)
	}
}
