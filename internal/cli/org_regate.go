package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// newOrgRegateCmd wires `ralph org regate --branch <b>` (plan
// docs/plans/active/2026-10-16-org-regate.md, AC1 to AC9 at the command
// line): (*org.Org).Regate reads the branch's binding, merges origin's default
// branch into it in the feature worktree when it lacks it, runs the gate
// scripts there, records the outcome with via=regate, and pushes a pass; this
// command passes it the branch and the process environment, cancels its
// context on SIGINT, SIGTERM, or SIGHUP (signal.NotifyContext), and prints
// the result (printRegateResult). Regate reads the org from the branch's
// binding (D1), so --org-id is refused before anything is read, as
// merge-order refuses it, and so is a missing --branch. Under
// [org.gate].regate_checks = "ci" (plan
// docs/plans/active/2026-10-29-regate-ci-checks.md, S6), it also passes the
// loaded config's Org.Gate.RegateChecks and --wait-checks, and leaves Exec,
// LookPath, and GHTimeout unset, as ralph org merge does: the real git and gh
// on PATH, with the 15 s timeout.
func newOrgRegateCmd(stateDir, configPath *string) *cobra.Command {
	var branch string
	var waitChecks time.Duration

	cmd := &cobra.Command{
		Use:   "regate",
		Short: "Merge the default branch into a disbanded feature org's branch, run the gate on it, and push it",
		Long: "ralph org regate brings the branch of a disbanded feature org back to\n" +
			"ready in ralph org merge-order after another feature's PR was merged:\n" +
			"in the org's feature worktree, it merges origin's default branch into\n" +
			"the branch when the branch lacks it, runs the gate there, records the\n" +
			"outcome, and pushes the branch. While a feature org runs, its reviewer\n" +
			"runs ralph org gate at its leader's TASK; once the org has opened its\n" +
			"PR and been disbanded, ralph org gate refuses it, and regate takes its\n" +
			"place. Name the branch as merge-order lists it: regate reads the org\n" +
			"and the feature worktree from the branch's latest complete feature\n" +
			"binding, so --org-id is refused, and it works in that worktree\n" +
			"whatever the current directory, which only picks the ledger, as for\n" +
			"every org verb.\n" +
			"\n" +
			"First, regate checks that no other ralph org regate, merge, or cleanup\n" +
			"runs on the ledger (one runs at a time; it waits up to 1 s for the\n" +
			"lock), that a complete binding names the branch and its org is not\n" +
			"running, that the feature worktree exists and has the branch checked\n" +
			"out (not detached, no rebase or merge in progress), that it is clean\n" +
			"(git status --porcelain --untracked-files=all prints nothing), that a\n" +
			"default branch is found (the one refs/remotes/origin/HEAD names, else\n" +
			"main, else master), and that ./scripts/run-static-verify.sh and\n" +
			"./scripts/run-test.sh are executable files there. Then it fetches\n" +
			"origin's default branch and the branch, each with its own refspec, with\n" +
			"GIT_TERMINAL_PROMPT=0 and up to 120 s each, and checks that HEAD is\n" +
			"origin's tip of the branch: regate gates only what origin has. When a\n" +
			"check or a fetch fails, or origin has no such branch (it was deleted\n" +
			"after its merge), regate records nothing, leaves the worktree and origin\n" +
			"as they were, says why and what to do next on stderr, and exits 1.\n" +
			"\n" +
			"When HEAD contains origin's default branch tip and the branch's latest\n" +
			"gate record is a gate_passed of HEAD on that tip, regate runs, records,\n" +
			"and pushes nothing, prints UP_TO_DATE:, HEAD:, BASE: and that gate's\n" +
			"LOG: on stdout, and exits 0. When HEAD contains the tip otherwise, it\n" +
			"runs the gate on HEAD. When HEAD does not contain the tip, it first\n" +
			"merges the tip in with a merge commit, git merge --no-ff --no-edit -m\n" +
			"'Merge origin/<default branch> <12 hex> into <branch>', with the\n" +
			"caller's git identity and hooks, and runs the gate on the merge commit.\n" +
			"It never rebases and never force-pushes. A merge that conflicts or that\n" +
			"a hook stops makes no commit: regate undoes a merge left in progress\n" +
			"(git merge --abort), records nothing, names the conflicting files, and\n" +
			"exits 1. When git has no user.name and user.email to make the merge\n" +
			"commit with, regate merges nothing, records nothing, and exits 1. So it\n" +
			"does when an ignored file or directory is on disk where the merge\n" +
			"would write (git merge overwrites those without a word): it names the\n" +
			"paths to move out of the worktree.\n" +
			"\n" +
			"The gate runs the two scripts as ralph org gate does (see ralph org\n" +
			"gate --help for their environment, the log under <state dir>/gates/,\n" +
			"and the reasons of a failure) and records gate_passed or gate_failed\n" +
			"for the binding's org, with via=regate. When the branch's binding or\n" +
			"its org changed while the scripts ran, regate records nothing and is\n" +
			"refused. A pass of a merge commit is pushed to origin's branch without\n" +
			"force: regate prints GATE: pass, MERGED:, HEAD:, BASE:, LOG: and\n" +
			"PUSHED: and exits 0. A pass of HEAD as origin has it needs no push:\n" +
			"GATE: pass, HEAD:, BASE: and LOG:, exit 0.\n" +
			"\n" +
			"On a failure regate prints GATE: fail, REASON:, MERGED: after a merge,\n" +
			"HEAD:, BASE: and LOG:, then the failing script's last 20 lines or the\n" +
			"files the scripts left, pushes nothing, and exits 1. After static_failed\n" +
			"or test_failed it undoes its merge commit (git reset --keep to origin's\n" +
			"tip of the branch; then only the reflog has the merge commit). After\n" +
			"head_moved or tree_changed it keeps it, since the reset could drop a\n" +
			"commit made meanwhile or carry the files along, and prints the command\n" +
			"that undoes it. When nothing is recorded (the record is refused, or an\n" +
			"interrupt comes before it), the merge commit is undone too. It is undone\n" +
			"only while HEAD is still the merge commit and the worktree is clean.\n" +
			"\n" +
			"Once a gate_passed is recorded, the record and the merge commit are kept\n" +
			"whatever the push does. A push that fails, runs past 120 s, or is\n" +
			"interrupted prints the lines before PUSHED: and exits 1 with the reason.\n" +
			"Run regate again then: when the push did not land, it pushes HEAD and\n" +
			"does nothing else (PUSH_ONLY:, HEAD:, BASE:, LOG: and PUSHED:, exit 0);\n" +
			"when it did, regate is UP_TO_DATE. It pushes only when the branch's\n" +
			"latest gate record is a gate_passed of HEAD on origin's default branch\n" +
			"tip, origin's branch is under HEAD, and, for a record regate wrote, HEAD\n" +
			"is regate's own merge commit: its parents are origin's tip of the branch\n" +
			"and the record's base, and its subject is the one regate gave it. When a\n" +
			"hook changed that subject, regate refuses and asks you to push HEAD\n" +
			"yourself. When origin's default branch or origin's branch moved in\n" +
			"between, regate refuses and says what changed.\n" +
			"\n" +
			"All of the above is [org.gate].regate_checks = \"local\", the default.\n" +
			"With \"ci\" (read from the ralph.toml --config names, else from\n" +
			"./ralph.toml in the directory regate runs in), regate runs no gate\n" +
			"script: the checks of the branch's PR on GitHub decide the record.\n" +
			"Right after the fetches it\n" +
			"checks that origin's URL names a GitHub repository, that gh is on PATH,\n" +
			"and that gh pr list finds the branch's PR in origin's repository, OPEN\n" +
			"and into the default branch, as ralph org merge finds it, and refuses\n" +
			"with nothing merged, pushed, or recorded when one fails; an UP_TO_DATE:\n" +
			"or PUSH_ONLY: regate needs them too. After a merge it pushes the merge\n" +
			"commit without force before anything is recorded, then reads the PR's\n" +
			"head (gh pr view) and its checks (gh pr checks) for the merge commit,\n" +
			"or for HEAD when it merged nothing. When the PR's head is that commit\n" +
			"and every check is pass or skipping with at least one pass, it records\n" +
			"gate_ci_passed with via=regate_ci and prints GATE: pass and CHECKS:\n" +
			"passed (PR #<n>), then MERGED:, HEAD:, BASE:, LOG: (a log with the\n" +
			"checks' JSON) and PUSHED: as above, and exits 0. When a check ended\n" +
			"and is neither pass nor skipping and none is pending, it records\n" +
			"gate_ci_failed with reason ci_failed and via=regate_ci, prints GATE:\n" +
			"fail, REASON: ci_failed, CHECKS: failed (PR #<n>) and the same lines,\n" +
			"then the failed checks, and exits 1; a pushed merge commit stays on\n" +
			"origin, so fix the branch, push the fix, and run regate again. When the\n" +
			"PR's head is not that commit yet, a check is pending (also with others\n" +
			"failed), or gh reports no checks, regate records nothing, prints\n" +
			"CHECKS: pending (<why>), MERGED:, HEAD:, BASE:, PUSHED: and NEXT: <the\n" +
			"next step>, and exits 6; the merge commit stays on origin, and the next\n" +
			"regate finds HEAD on origin's tip and reads the checks again. Checks\n" +
			"that cannot be read, or that are all skipping, are refused with nothing\n" +
			"recorded (exit 1). After a push that fails, regate reads origin's tip\n" +
			"of the branch back: it undoes the merge commit only when that tip is\n" +
			"still the one the merge commit was made on, goes on to the checks when\n" +
			"it is the merge commit, and otherwise keeps the merge commit as HEAD\n" +
			"and refuses with how to check origin. A ci pass is never pushed again\n" +
			"(no PUSH_ONLY:): when origin's tip is no longer its head, regate\n" +
			"refuses.\n" +
			"\n" +
			"With --wait-checks <duration> (Go's form, such as 60m; 0, the default,\n" +
			"reads the checks once; at most 6h; only with \"ci\": a negative value,\n" +
			"one over 6h, and one above 0 with \"local\" are refused before the\n" +
			"ledger is read), a ci regate waits for checks that are not done\n" +
			"instead of exiting 6. It releases regate.lock while it waits, so\n" +
			"another regate, merge, or cleanup can run meanwhile, and every 30 s it\n" +
			"only reads the PR's head and checks again. Once they are done (or gh\n" +
			"fails), it takes the lock again, once, checks again that the branch's\n" +
			"binding and its stopped org, HEAD, and origin's tip of the branch are\n" +
			"as they were, and reads the PR again to record from it; checks pending\n" +
			"again then exit 6 with no more wait. When another regate, merge, or\n" +
			"cleanup holds the lock at that moment, regate refuses (exit 1), as\n" +
			"merge does after its wait. stderr gets one line when the wait starts\n" +
			"and one when it ends. When another regate recorded a pass of HEAD\n" +
			"meanwhile, it prints UP_TO_DATE: and exits 0; when the time is up, it\n" +
			"exits 6 as above, without taking the lock again, with how long it\n" +
			"waited in CHECKS: pending.\n" +
			"\n" +
			"An interrupt (SIGINT, SIGTERM, or SIGHUP) stops a running fetch, script,\n" +
			"or push. A merge, its abort, and its undo finish first: they run in a\n" +
			"process group of their own, which the Ctrl-C or the hangup of the\n" +
			"terminal does not reach either, and regate says on stderr that it\n" +
			"waits for one (at most 120 s). Exit codes: 0\n" +
			"for UP_TO_DATE, a pass, and a push that landed; 6 for a ci regate\n" +
			"whose checks are not done; 1 for a refusal, a failure, a push that\n" +
			"failed, an interrupt, and any other error. regate\n" +
			"starts no autonomous work, so it needs no running watch and no approved\n" +
			"plan. While it runs, ralph org spawn (not a --dry-run) and ralph org\n" +
			"start (either form) of the org it works on are refused; other orgs are\n" +
			"not affected.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("org-id") {
				return errors.New("org: regate: --org-id does not apply: regate reads the org from the binding of the branch --branch names")
			}
			if strings.TrimSpace(branch) == "" {
				return errors.New("org: regate: --branch is required: name the feature branch to regate (ralph org merge-order lists " +
					"the ledger's branches)")
			}
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath, orgLedgerMutating)
			if err != nil {
				return err
			}
			parent := cmd.Context()
			if parent == nil {
				parent = context.Background()
			}
			// As for ralph org gate (newOrgGateCmd): the first SIGINT,
			// SIGTERM, or SIGHUP cancels ctx with a cause that names it, and
			// later ones are caught until stop runs.
			ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
			defer stop()
			// Regate refuses a --wait-checks out of range, and one above 0
			// with local checks, before it takes regate.lock or reads the
			// ledger, with one text for either; the second needs
			// [org.gate].regate_checks, so the flag is checked there, once
			// the config is loaded, and not here.
			res, err := rt.Regate(ctx, org.RegateParams{Branch: branch, Env: os.Environ(), Stderr: cmd.ErrOrStderr(),
				Checks: rt.Config.Gate.RegateChecks, WaitChecks: waitChecks})
			return printRegateResult(cmd.OutOrStdout(), res, err)
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "", "the feature branch to regate, as ralph org merge-order lists it (required)")
	cmd.Flags().DurationVar(&waitChecks, "wait-checks", 0, "with [org.gate].regate_checks = \"ci\" only, how long to wait, with "+
		"regate.lock released, for the checks of the branch's PR to be done, such as 60m (0 reads them once; at most 6h)")

	return cmd
}

// ErrRegateChecksPending is wrapped by the error of ralph org regate when
// Regate ended org.RegateOutcomeChecksPending: under [org.gate].regate_checks
// = "ci", the checks of the branch's PR are not done (the PR's head is not
// HEAD yet, a check is pending, or gh reports no checks), also once
// --wait-checks ran out, and nothing was recorded. ExitCode maps it to 6, so
// a caller can tell checks to wait for from a refusal or a failure (plan
// docs/plans/active/2026-10-29-regate-ci-checks.md, D4).
var ErrRegateChecksPending = errors.New("the checks of the branch's PR are not done")

// printRegateResult prints res, what Regate returned with err, and returns
// the command's error.
//
// An error is the command's error as Regate wrote it. When Regate had
// recorded a gate_passed or reached the push before it (res.Outcome set), the
// lines below are printed first, so stdout says what was recorded and pushed.
// A refusal prints nothing on stdout and is the error (exit 1); a refusal at
// the record also names the log: the scripts' output, or, for a ci regate
// (res.PR set), the JSON of the PR's checks it read.
//
// Otherwise stdout gets one block of `KEY: value` lines, by res.Outcome:
//
//	up_to_date      UP_TO_DATE: <why>, MERGED: (after a ci regate's merge),
//	                HEAD:, BASE:, LOG: (of the earlier gate), PUSHED: (after
//	                a ci regate's merge)
//	push_only       PUSH_ONLY: <why>, HEAD:, BASE:, LOG:, PUSHED:
//	gated           GATE: pass|fail, REASON: (on a fail), CHECKS: (ci),
//	                HEAD:, BASE:, LOG:
//	merged          GATE: pass|fail, REASON: (on a fail), CHECKS: (ci),
//	                MERGED:, HEAD:, BASE:, LOG:, PUSHED: (once pushed)
//	checks_pending  CHECKS: pending (<why>), MERGED: (after a merge), HEAD:,
//	                BASE:, PUSHED: (after a merge), NEXT: <the next step>
//
// CHECKS: is `passed (PR #<n>)` or `failed (PR #<n>)` when the record came
// from the PR's checks (res.PR set). The <why> of up_to_date says so when
// another regate recorded the pass while a ci regate waited for the checks
// (res.PR set). LOG: is left out when res has no log, PUSHED: when nothing was
// pushed. Another outcome gets only HEAD:, BASE:, LOG: and PUSHED:, and an
// error that names it (exit 1). A pass, up_to_date, and a push that landed
// return nil (exit 0). A push that failed returns res.PushFailed (exit 1).
// checks_pending returns an error that wraps ErrRegateChecksPending (exit 6)
// and repeats the next step. A recorded gate_failed or gate_ci_failed is
// followed by a blank line and the detail ralph org gate prints
// (printGateFailDetail, worded for regate; for ci_failed, the failed checks
// and the log), then, after a merge, a blank line and what became of the
// merge commit: undone, res.NotUndone, or, once a ci regate pushed it, that
// it stays on origin; its error (exit 1) is one line naming the record and
// the reason.
func printRegateResult(w io.Writer, res org.RegateResult, err error) error {
	if err != nil {
		if res.Outcome != "" {
			printRegateLines(w, res)
		}
		return err
	}
	if res.Refused != "" {
		switch {
		case res.LogPath == "":
			return errors.New(res.Refused)
		case res.PR != 0:
			return fmt.Errorf("%s; the JSON of the checks of PR #%d it read is in %s", res.Refused, res.PR, res.LogPath)
		}
		return fmt.Errorf("%s; the scripts' output is in %s", res.Refused, res.LogPath)
	}
	printRegateLines(w, res)
	if res.PushFailed != "" {
		return errors.New(res.PushFailed)
	}
	switch res.Outcome {
	case org.RegateOutcomeUpToDate, org.RegateOutcomePushOnly:
		return nil
	case org.RegateOutcomeChecksPending:
		return fmt.Errorf("org: regate: %w (PR #%d), so nothing was recorded: %s", ErrRegateChecksPending, res.PR, res.ChecksNext)
	case org.RegateOutcomeGated, org.RegateOutcomeMerged:
	default:
		// An outcome this command does not know gets no GATE: line, since
		// whether anything was recorded is not known here.
		return fmt.Errorf("org: regate: the regate of %s ended with the outcome %s, which this command does not print", res.Branch,
			res.Outcome)
	}
	if res.Passed {
		return nil
	}

	again := "run ralph org regate --branch " + res.Branch + " again"
	printGateFailDetail(w, res.Reason, res.Tail, res.Dirty, gateCIFailure{pr: res.PR, failed: res.FailedChecks, log: res.LogPath},
		gateDetailWords{while: "regate runs", again: again})
	event, merge, pushed := org.EventGateFailed, "", "; nothing was pushed"
	if res.PR != 0 {
		event = org.EventGateCIFailed
	}
	switch {
	case res.Merge == "":
	case res.Pushed:
		// A ci regate pushes its merge commit before it reads the checks
		// that recorded the failure, and nothing undoes a pushed commit.
		_, _ = fmt.Fprintf(w, "\nthe merge commit %s stays on origin's %s, where regate pushed it before it read the PR's checks: "+
			"regate never force-pushes, so the fix goes on top of it\n", res.Merge, res.Branch)
		merge, pushed = fmt.Sprintf(" of the merge commit %s, which stays on origin's %s", res.Merge, res.Branch), ""
	case res.Undone:
		_, _ = fmt.Fprintf(w, "\nthe merge commit %s was undone (git reset --keep %s), so only the reflog has it now\n", res.Merge, res.PreMergeHead)
		merge = fmt.Sprintf(" of the merge commit %s, which was undone", res.Merge)
	default:
		_, _ = fmt.Fprintf(w, "\n%s\n", res.NotUndone)
		merge = fmt.Sprintf(" of the merge commit %s, which was not undone", res.Merge)
	}
	return fmt.Errorf("org: regate: recorded %s (reason=%s)%s%s", event, res.Reason, merge, pushed)
}

// printRegateLines prints printRegateResult's block of `KEY: value` lines for
// res (Outcome set).
func printRegateLines(w io.Writer, res org.RegateResult) {
	base := strings.TrimPrefix(res.BaseRef, "refs/remotes/")
	merged := func() {
		if res.Merge != "" {
			_, _ = fmt.Fprintf(w, "MERGED: %s (%s into %s)\n", res.Merge, base, res.PreMergeHead)
		}
	}
	switch res.Outcome {
	case org.RegateOutcomeUpToDate:
		if res.PR != 0 {
			// Only a ci regate's wait for the PR's checks ends so once it
			// read them: another regate recorded the pass meanwhile.
			_, _ = fmt.Fprintf(w, "UP_TO_DATE: while regate waited for the checks of PR #%d, another regate recorded a pass of HEAD "+
				"on %s's tip: this one recorded nothing\n", res.PR, base)
		} else {
			_, _ = fmt.Fprintf(w, "UP_TO_DATE: the latest gate record of %s is a pass of HEAD, origin's tip of the branch, on "+
				"%s's tip: nothing was run, recorded, or pushed\n", res.Branch, base)
		}
		merged()
	case org.RegateOutcomePushOnly:
		_, _ = fmt.Fprintf(w, "PUSH_ONLY: the latest gate record of %s is a gate_passed of HEAD on %s's tip, and origin's %s is "+
			"behind HEAD: nothing was run or recorded\n", res.Branch, base, res.Branch)
	case org.RegateOutcomeChecksPending:
		_, _ = fmt.Fprintf(w, "CHECKS: pending (%s)\n", res.ChecksPending)
		merged()
	case org.RegateOutcomeGated, org.RegateOutcomeMerged:
		if res.Passed {
			_, _ = fmt.Fprintln(w, "GATE: pass")
		} else {
			_, _ = fmt.Fprintf(w, "GATE: fail\nREASON: %s\n", res.Reason)
		}
		switch {
		case res.PR == 0:
		case res.Passed:
			_, _ = fmt.Fprintf(w, "CHECKS: passed (PR #%d)\n", res.PR)
		default:
			_, _ = fmt.Fprintf(w, "CHECKS: failed (PR #%d)\n", res.PR)
		}
		merged()
	}
	_, _ = fmt.Fprintf(w, "HEAD: %s\nBASE: %s\n", res.Head, res.Base)
	if res.LogPath != "" {
		_, _ = fmt.Fprintf(w, "LOG: %s\n", res.LogPath)
	}
	if res.Pushed {
		_, _ = fmt.Fprintf(w, "PUSHED: %s to origin's %s\n", res.Head, res.Branch)
	}
	if res.Outcome == org.RegateOutcomeChecksPending {
		_, _ = fmt.Fprintf(w, "NEXT: %s\n", res.ChecksNext)
	}
}
