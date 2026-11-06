package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// newOrgResumeCmd is `ralph org resume` (plan 2026-11-07-org-seat-resume,
// S6, D6 and D7; spec docs/specs/2026-11-07-org-seat-resume.md, FR-1 and
// FR-7): (*org.Org).Resume, then printResumeResult. --org-id is optional
// (D6): without it Resume looks at the seats of every org of the ledger and
// at the headless directors. A run that types into panes and records
// `resumed` is a mutating verb of the ledger; --dry-run types and records
// nothing, so it resolves the ledger as a read-only verb does.
func newOrgResumeCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "resume",
		Short: "Continue the seats whose CLI stopped on an API error (a connection failure or a usage limit)",
		Long: "ralph org resume continues the seats whose CLI ended its turn on an API\n" +
			"error: a connection failure or a usage limit. It looks at every running\n" +
			"seat with a pane, only those of --org-id when it is given, and without\n" +
			"--org-id also at the headless directors in herdr (ralph org director\n" +
			"start). For each it reads herdr's agent status (herdr agent get) and the\n" +
			"last 60 lines of the pane. Only when herdr reports the agent idle or done\n" +
			"and the screen ends with a known API error sentence above an empty input\n" +
			"box does it type this sentence and press Enter once:\n" +
			"\n" +
			"  " + org.ResumeText + "\n" +
			"\n" +
			"When the screen asks for Enter after a usage limit reset (Press enter to\n" +
			"continue), it presses Enter once and types nothing. It never types into a\n" +
			"seat that is working, retrying or waiting on its own, or shows an\n" +
			"approval, a choice, or any screen it does not know: those are listed as\n" +
			"waiting or skipped. It holds the pane's lock, the one ralph org send\n" +
			"takes, while it reads the screen again and types, and presses Enter only\n" +
			"once the input box holds the sentence. It goes through the seats that\n" +
			"are not leaders first, then the leaders, then the headless directors, and\n" +
			"skips the pane it runs in (HERDR_PANE_ID). Each seat it typed into gets a\n" +
			"resumed event in the manifest.\n" +
			"\n" +
			"It prints one line per seat: what it did (resumed, entered, waiting,\n" +
			"skipped, or failed), the seat (<org_id>/<seat_id>, or director <id>), its\n" +
			"driver, and the screen line it matched or the reason. A usage limit's\n" +
			"reset time is printed as the screen shows it, not read. A summary line\n" +
			"follows. When herdr does not confirm that the seat started working after\n" +
			"Enter, a note: line on stderr names the command that reads its pane.\n" +
			"--dry-run looks at every seat the same way but types and records nothing\n" +
			"(would_resume, would_enter). It exits 1 when typing into a seat failed or\n" +
			"the manifest could not be read, and 0 otherwise, also when every seat was\n" +
			"waiting or skipped.\n" +
			"\n" +
			"The director runs it when the human asks to continue after a connection\n" +
			"failure or a usage limit; a human can run it too.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkResumeOrgID(cmd, *orgID); err != nil {
				return err
			}
			access := orgLedgerMutating
			if dryRun {
				access = orgLedgerReadOnly
			}
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath, access)
			if err != nil {
				return err
			}
			result := rt.Resume(org.ResumeParams{OrgID: *orgID, DryRun: dryRun})
			hintStateDir := orgInboxHintStateDir(cmd, filepath.Dir(rt.Manifest.Path()))
			return printResumeResult(cmd.OutOrStdout(), cmd.ErrOrStderr(), result, dryRun, hintStateDir)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "look at every seat as a real run does, but type and record nothing")

	return cmd
}

// checkResumeOrgID checks --org-id when it was given, before the ledger is
// resolved: resume runs without it (D6), so an empty value is refused rather
// than read as every org, and any other value must be an identifier, as
// requireOrgID checks it for the verbs that require it.
func checkResumeOrgID(cmd *cobra.Command, orgID string) error {
	if !cmd.Flags().Changed("org-id") {
		return nil
	}
	if strings.TrimSpace(orgID) == "" {
		return errors.New("org: resume: --org-id is empty; leave it out to resume the seats of every org")
	}
	return org.ValidateIdentifier("org_id", orgID)
}

// printResumeResult prints Resume's result r and returns the error that
// makes the command exit 1 (D7: FR-7), nil when r.Errs is empty, so a run
// whose targets were all waiting or skipped exits 0.
//
// stdout: one line per target in the order Resume handled them, in aligned
// columns (outcome, resumeTargetLabel, driver, resumeTargetText), then the
// summary (resumeSummary). With no target and no error it prints
// `resume: no running seats` instead; a manifest that could not be read
// leaves stdout empty.
//
// stderr: the notes of each target (printResumeTargetNotes), then a warning
// when director.jsonl could not be read (the seats were looked at all the
// same, and the exit code does not change), then `error: <err>` for each
// error of r.Errs.
//
// Every text that comes from a pane, the manifest, director.jsonl, or herdr
// goes through org.PrintableInboxText, so a control character in it is
// printed escaped and never reaches the terminal as one.
func printResumeResult(out, stderr io.Writer, r org.ResumeResult, dryRun bool, hintStateDir string) error {
	if len(r.Targets) == 0 && len(r.Errs) == 0 {
		_, _ = fmt.Fprintf(out, "%s: no running seats\n", resumeSummaryPrefix(dryRun))
	}
	rows := make([][4]string, len(r.Targets))
	var widths [3]int
	for i, t := range r.Targets {
		rows[i] = [4]string{string(t.Outcome), resumeTargetLabel(t), inboxField(t.Driver), inboxField(resumeTargetText(t))}
		for c := range widths {
			widths[c] = max(widths[c], utf8.RuneCountInString(rows[i][c]))
		}
	}
	for i, t := range r.Targets {
		row := rows[i]
		_, _ = fmt.Fprintf(out, "%-*s  %-*s  %-*s  %s\n", widths[0], row[0], widths[1], row[1], widths[2], row[2], row[3])
		printResumeTargetNotes(stderr, t, hintStateDir)
	}
	if len(r.Targets) > 0 {
		_, _ = fmt.Fprintln(out, resumeSummary(r.Targets, dryRun))
	}
	if r.DirectorReadErr != nil {
		_, _ = fmt.Fprintf(stderr, "warning: resume: the headless directors of the ledger could not be read, so none was looked at: %s\n",
			inboxField(r.DirectorReadErr.Error()))
	}
	for _, err := range r.Errs {
		_, _ = fmt.Fprintf(stderr, "error: %s\n", org.PrintableInboxText(err.Error(), false))
	}
	if len(r.Errs) == 0 {
		return nil
	}
	return fmt.Errorf("org: resume: %d error(s), listed above", len(r.Errs))
}

// resumeSummaryPrefix starts the summary line and the line of a run with no
// target: `resume`, or `resume (dry run)` for --dry-run.
func resumeSummaryPrefix(dryRun bool) string {
	if dryRun {
		return "resume (dry run)"
	}
	return "resume"
}

// resumeSummary is the line after the targets' lines: how many targets got
// each outcome. A dry run has no resumed, entered, or failed target, and a
// real run no would_resume or would_enter one, so each counts only the
// outcomes it can have.
func resumeSummary(targets []org.ResumeTargetResult, dryRun bool) string {
	n := map[org.ResumeOutcome]int{}
	for _, t := range targets {
		n[t.Outcome]++
	}
	if dryRun {
		return fmt.Sprintf("%s: %d would resume, %d would enter, %d waiting, %d skipped", resumeSummaryPrefix(dryRun),
			n[org.ResumeOutcomeWouldResume], n[org.ResumeOutcomeWouldEnter], n[org.ResumeOutcomeWaiting], n[org.ResumeOutcomeSkipped])
	}
	return fmt.Sprintf("%s: %d resumed, %d entered, %d waiting, %d skipped, %d failed", resumeSummaryPrefix(dryRun),
		n[org.ResumeOutcomeResumed], n[org.ResumeOutcomeEntered], n[org.ResumeOutcomeWaiting], n[org.ResumeOutcomeSkipped],
		n[org.ResumeOutcomeFailed])
}

// resumeTargetLabel names t in its line and its notes: `<org_id>/<seat_id>`
// for a seat and `director <id>` for a director, each value escaped
// (inboxField).
func resumeTargetLabel(t org.ResumeTargetResult) string {
	if t.Kind == org.ResumeTargetDirector {
		return "director " + inboxField(t.ID)
	}
	return inboxField(t.OrgID) + "/" + inboxField(t.ID)
}

// resumeTargetText is the last column of t's line: the screen line Resume
// matched when there is one, else the reason. A skipped or failed target
// shows its reason even when it has a line, since the line alone would read
// as a screen to type into: a target skipped because its screen changed
// under the lock, or one whose typing failed.
func resumeTargetText(t org.ResumeTargetResult) string {
	switch {
	case (t.Outcome == org.ResumeOutcomeSkipped || t.Outcome == org.ResumeOutcomeFailed) && t.Reason != "":
		return t.Reason
	case t.Line != "":
		return t.Line
	}
	return t.Reason
}

// printResumeTargetNotes writes to w (stderr) the lines a target Resume
// typed into (resumed or entered) may need. When herdr did not report it
// working or blocked after Enter (SubmitConfirmed false), a note names the
// command that reads its pane (resumeReadCommand). When a seat's `resumed`
// event could not be recorded, a warning says so; its error is also in
// ResumeResult.Errs, so the command exits 1.
func printResumeTargetNotes(w io.Writer, t org.ResumeTargetResult, hintStateDir string) {
	if t.Outcome != org.ResumeOutcomeResumed && t.Outcome != org.ResumeOutcomeEntered {
		return
	}
	label := resumeTargetLabel(t)
	if !t.SubmitConfirmed {
		_, _ = fmt.Fprintf(w, "note: %s: the submit was not confirmed; read the pane with %s\n", label, resumeReadCommand(t, hintStateDir))
	}
	if t.Kind == org.ResumeTargetSeat && !t.Recorded {
		_, _ = fmt.Fprintf(w, "warning: %s: Enter was pressed, but the resumed event could not be recorded in the manifest\n", label)
	}
}

// resumeReadCommand is the command that reads the pane of t, escaped: for a
// seat `ralph org read --org-id <org_id> --seat <seat_id>`, with --state-dir
// only when it was given (orgVerbHint, hintStateDir from
// orgInboxHintStateDir); for a director, which is not a seat of the manifest,
// `herdr pane read <pane_id>` of the 60 lines Resume read.
func resumeReadCommand(t org.ResumeTargetResult, hintStateDir string) string {
	if t.Kind == org.ResumeTargetDirector {
		return org.PrintableInboxText("herdr pane read "+org.ShellWord(t.PaneID)+" --source recent --lines 60 --format text", false)
	}
	return org.PrintableInboxText(orgVerbHint("read --org-id "+org.ShellWord(t.OrgID)+" --seat "+org.ShellWord(t.ID), hintStateDir), false)
}
