package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// newOrgRegateCmd wires `ralph org regate --branch <b>` (plan
// docs/plans/active/2026-10-16-org-regate.md, AC1 to AC9 at the command
// line): (*org.Org).Regate reads the branch's binding, merges origin's default
// branch into it in the feature worktree when it lacks it, runs the gate
// scripts there, records the outcome with via=regate, and pushes a pass; this
// command passes it the branch and the process environment, cancels its
// context on SIGINT, SIGTERM, or SIGHUP (gateSignalContext), and prints the
// result (printRegateResult). Regate reads the org from the branch's binding
// (D1), so --org-id is refused before anything is read, as merge-order
// refuses it, and so is a missing --branch.
func newOrgRegateCmd(stateDir, configPath *string) *cobra.Command {
	var branch string

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
			"First, regate checks that no other ralph org regate or merge runs on\n" +
			"the ledger (one runs at a time; it waits up to 1 s for the lock), that\n" +
			"a complete binding names the branch and its org is not running, that\n" +
			"the feature worktree exists and has the branch checked out (not\n" +
			"detached, no rebase or merge in progress), that it is clean (git status\n" +
			"--porcelain --untracked-files=all prints nothing), that a default\n" +
			"branch is found (the one refs/remotes/origin/HEAD names, else main,\n" +
			"else master), and that ./scripts/run-static-verify.sh and\n" +
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
			"An interrupt (SIGINT, SIGTERM, or SIGHUP) stops a running fetch, script,\n" +
			"or push. A merge, its abort, and its undo finish first: they run in a\n" +
			"process group of their own, which the Ctrl-C or the hangup of the\n" +
			"terminal does not reach either, and regate says on stderr that it\n" +
			"waits for one (at most 120 s). Exit codes: 0\n" +
			"for UP_TO_DATE, a pass, and a push that landed; 1 for a refusal, a\n" +
			"failure, a push that failed, an interrupt, and any other error. regate\n" +
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
			ctx, stop := gateSignalContext(parent, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
			defer stop()
			res, err := rt.Regate(ctx, org.RegateParams{Branch: branch, Env: os.Environ(), Stderr: cmd.ErrOrStderr()})
			return printRegateResult(cmd.OutOrStdout(), res, err)
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "", "the feature branch to regate, as ralph org merge-order lists it (required)")

	return cmd
}

// printRegateResult prints res, what Regate returned with err, and returns
// the command's error.
//
// An error is the command's error as Regate wrote it. When Regate had
// recorded a gate_passed or reached the push before it (res.Outcome set), the
// lines below are printed first, so stdout says what was recorded and pushed.
// A refusal prints nothing on stdout and is the error (exit 1); a refusal at
// the record, after the scripts ran, also names their log.
//
// Otherwise stdout gets one block of `KEY: value` lines, by res.Outcome:
//
//	up_to_date  UP_TO_DATE: <why>, HEAD:, BASE:, LOG: (of the earlier gate)
//	push_only   PUSH_ONLY: <why>, HEAD:, BASE:, LOG:, PUSHED:
//	gated       GATE: pass|fail, REASON: (on a fail), HEAD:, BASE:, LOG:
//	merged      GATE: pass|fail, REASON: (on a fail), MERGED:, HEAD:, BASE:,
//	            LOG:, PUSHED: (once pushed)
//
// LOG: is left out when res has no log, PUSHED: when nothing was pushed. A
// pass, up_to_date, and a push that landed return nil (exit 0). A push that
// failed returns res.PushFailed (exit 1). A recorded gate_failed is followed
// by a blank line and the detail ralph org gate prints (printGateFailDetail,
// worded for regate), then, after a merge, a blank line and what became of
// the merge commit: undone, or res.NotUndone; its error (exit 1) is one line
// naming the reason.
func printRegateResult(w io.Writer, res org.RegateResult, err error) error {
	if err != nil {
		if res.Outcome != "" {
			printRegateLines(w, res)
		}
		return err
	}
	if res.Refused != "" {
		if res.LogPath != "" {
			return fmt.Errorf("%s; the scripts' output is in %s", res.Refused, res.LogPath)
		}
		return errors.New(res.Refused)
	}
	printRegateLines(w, res)
	if res.PushFailed != "" {
		return errors.New(res.PushFailed)
	}
	if res.Outcome == org.RegateOutcomeUpToDate || res.Outcome == org.RegateOutcomePushOnly || res.Passed {
		return nil
	}

	again := "run ralph org regate --branch " + res.Branch + " again"
	printGateFailDetail(w, res.Reason, res.Tail, res.Dirty, gateDetailWords{while: "regate runs", again: again})
	merge := ""
	switch {
	case res.Merge == "":
	case res.Undone:
		_, _ = fmt.Fprintf(w, "\nthe merge commit %s was undone (git reset --keep %s), so only the reflog has it now\n", res.Merge, res.PreMergeHead)
		merge = fmt.Sprintf(" of the merge commit %s, which was undone", res.Merge)
	default:
		_, _ = fmt.Fprintf(w, "\n%s\n", res.NotUndone)
		merge = fmt.Sprintf(" of the merge commit %s, which was not undone", res.Merge)
	}
	return fmt.Errorf("org: regate: recorded gate_failed (reason=%s)%s; nothing was pushed", res.Reason, merge)
}

// printRegateLines prints printRegateResult's block of `KEY: value` lines for
// res (Outcome set).
func printRegateLines(w io.Writer, res org.RegateResult) {
	base := strings.TrimPrefix(res.BaseRef, "refs/remotes/")
	switch res.Outcome {
	case org.RegateOutcomeUpToDate:
		_, _ = fmt.Fprintf(w, "UP_TO_DATE: the latest gate record of %s is a gate_passed of HEAD, origin's tip of the branch, on %s's "+
			"tip: nothing was run, recorded, or pushed\n", res.Branch, base)
	case org.RegateOutcomePushOnly:
		_, _ = fmt.Fprintf(w, "PUSH_ONLY: the latest gate record of %s is a gate_passed of HEAD on %s's tip, and origin's %s is "+
			"behind HEAD: nothing was run or recorded\n", res.Branch, base, res.Branch)
	default:
		if res.Passed {
			_, _ = fmt.Fprintln(w, "GATE: pass")
		} else {
			_, _ = fmt.Fprintf(w, "GATE: fail\nREASON: %s\n", res.Reason)
		}
		if res.Merge != "" {
			_, _ = fmt.Fprintf(w, "MERGED: %s (%s into %s)\n", res.Merge, base, res.PreMergeHead)
		}
	}
	_, _ = fmt.Fprintf(w, "HEAD: %s\nBASE: %s\n", res.Head, res.Base)
	if res.LogPath != "" {
		_, _ = fmt.Fprintf(w, "LOG: %s\n", res.LogPath)
	}
	if res.Pushed {
		_, _ = fmt.Fprintf(w, "PUSHED: %s to origin's %s\n", res.Head, res.Branch)
	}
}
