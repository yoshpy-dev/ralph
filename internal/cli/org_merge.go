package cli

import (
	"cmp"
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

// newOrgMergeCmd wires `ralph org merge --branch <b>` (plan
// docs/plans/active/2026-10-17-org-auto-merge.md, S4: AC2, AC3, AC6 and AC12
// at the command line): (*org.Org).Merge checks the branch's PR, its checks,
// and GitHub's protection of origin's default branch, merges the PR with gh
// pr merge once, reads it back, and records pr_merged; this command reads
// [org.director].auto_merge as ralph org escalate reads [org.director].enabled
// (withMainWorktreeDirectorEnabled), passes it with the branch, runs git and
// gh in the current directory (Dir left empty, Exec and LookPath nil, so the
// real commands, with the default timeouts), cancels its context on SIGINT,
// SIGTERM, or SIGHUP (signal.NotifyContext), and prints the result
// (printMergeResult). Merge reads the org from the branch's binding or gate
// record (D4), so --org-id is refused before anything is read, as regate
// refuses it, and so are a missing --branch and a --wait-checks out of range
// (org.MergeWaitChecksRefusal). --wait-checks (S7, D13) reaches Merge with
// stderr for the lines of its wait. The main worktree root (mergeCleanupRoot)
// reaches Merge as MergeParams.RepoRoot, so that Merge cleans up the branch's
// feature worktree once pr_merged is recorded (plan
// docs/plans/active/2026-10-28-org-merge-cleanup.md, S2).
func newOrgMergeCmd(stateDir, configPath *string) *cobra.Command {
	var branch string
	var waitChecks time.Duration

	cmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge, with gh, the PR of a feature branch ralph org merge-order lists as ready",
		Long: "ralph org merge merges, with gh, the PR of one feature branch that\n" +
			"ralph org merge-order lists as ready, after it has checked again, live,\n" +
			"what the merge rests on. Name the branch as merge-order lists it: merge\n" +
			"reads the org from the branch's latest complete feature binding, else\n" +
			"from its latest gate record, so --org-id is refused. git and gh run in\n" +
			"the current directory's repository, as for merge-order, and the merge\n" +
			"goes into origin's repository: merge reads its host and owner/repo from\n" +
			"git remote get-url origin (an origin that names none, such as a local\n" +
			"path, is refused) and pins every gh call to it, gh pr with -R\n" +
			"<host>/<owner>/<repo>, gh repo view with that as its argument, and gh\n" +
			"api with --hostname <host>, so neither gh repo set-default nor an\n" +
			"upstream remote makes gh read another repository.\n" +
			"\n" +
			"First, merge checks, in this order, and refuses at the first check that\n" +
			"fails: that no other ralph org regate, merge, or cleanup runs on the\n" +
			"ledger (one runs at a time; it waits up to 1 s for regate.lock); that\n" +
			"origin's URL names a host and owner/repo; that gh is on PATH; that a\n" +
			"feature binding or a gate record in the ledger names the branch; that\n" +
			"git ls-remote reads origin; that gh pr list finds the branch's PR in\n" +
			"origin's repository (pr_lookup found, in merge-order's terms) and its\n" +
			"url names that repository; that merge-order classifies the branch as\n" +
			"ready; that the branch's org is not running (its leader may still push:\n" +
			"handle the org's RESULT in the inbox, and once the org is disbanded, run\n" +
			"merge again); that gh pr view reads the PR as OPEN, with the head the\n" +
			"gate passed as its head, origin's default branch as its base, and no\n" +
			"auto-merge request (the refusal names gh pr merge <n> --disable-auto -R\n" +
			"<repo>, which removes one); that every check gh pr checks lists is pass\n" +
			"or skipping, and at least one is pass; that GitHub's protection of the\n" +
			"default branch, read live with gh api and not from the cache ralph\n" +
			"doctor writes, shows no merge queue and keeps the merge up to date for\n" +
			"you: a ruleset whose up-to-date rule (Require branches to be up to date\n" +
			"before merging) has required status checks of its own and whose\n" +
			"current_user_can_bypass is never, or else a branch protection that is\n" +
			"strict with required checks and has enforce_admins enabled, whether or\n" +
			"not you are an admin of the repository, since without it an admin or a\n" +
			"custom role that may bypass branch protections can merge a head that\n" +
			"is not up to date (its strict can be read only with admin access, so\n" +
			"the refusal says to use a ruleset instead); that gh repo view gives the\n" +
			"merge method (viewerDefaultMergeMethod: MERGE, SQUASH, or REBASE); and,\n" +
			"last, that a second git ls-remote finds origin's default branch and the\n" +
			"branch where the gate left them. A git or gh call that fails is refused\n" +
			"too. A refusal merges and records nothing, says why and what to do next\n" +
			"on stderr, and exits 1.\n" +
			"\n" +
			"Then merge runs gh pr merge <n> --merge|--squash|--rebase\n" +
			"--match-head-commit <the gate's head> -R <repo> once, for up to 60 s,\n" +
			"never with --auto, --admin, or --delete-branch, so GitHub merges only\n" +
			"while the PR's head is the gate's. Whatever that returned, merge reads\n" +
			"the PR back once with gh pr view. When the PR is MERGED at the gate's\n" +
			"head into origin's default branch, merge records pr_merged for the org\n" +
			"in the ledger, prints MERGED: <PR url> (PR #<n> of <branch>), METHOD:,\n" +
			"HEAD:, MERGE COMMIT: (unknown when gh gave none), AUTO MERGE:,\n" +
			"PROTECTION: (the setting that keeps the merge up to date), and NOTE:\n" +
			"when gh pr merge failed or timed out though the PR merged, and exits 0.\n" +
			"When the PR is MERGED at another head, or with no head read back\n" +
			"(someone may have pushed a new head and merged it after the checks),\n" +
			"merge records nothing, prints no MERGED: line, names the head merged on\n" +
			"stderr, and exits 1. When the PR is MERGED at the gate's head into\n" +
			"another base, or with no base read back (someone may have retargeted the\n" +
			"PR after the checks; --match-head-commit binds the head, not the base),\n" +
			"merge records nothing, prints no MERGED: line, names the base on stderr,\n" +
			"and exits 1. When the PR is not MERGED, merge records nothing, says what\n" +
			"happened on stderr with the end of gh pr merge's error when it failed,\n" +
			"and exits 1; an auto-merge request GitHub holds instead of merging is\n" +
			"removed with gh pr merge <n> --disable-auto -R <repo>, and when that\n" +
			"fails, merge prints that command for you to run. When the read fails,\n" +
			"whether the PR merged is unknown: merge records nothing, names the gh\n" +
			"pr view that checks it, and exits 1. When the PR merged but pr_merged\n" +
			"cannot be recorded, merge prints the lines above and exits 1.\n" +
			"\n" +
			"Once pr_merged is recorded, merge cleans up the branch's feature\n" +
			"worktree, the one ralph org start --plan made, with regate.lock still\n" +
			"held: it runs ./scripts/ralph-worktree.sh cleanup --id org-<org_id>\n" +
			"--force-branch in the main worktree (found as start --plan finds it),\n" +
			"which removes the worktree, the local branch, and the script's record,\n" +
			"and records worktree_cleaned in the ledger. Before that it checks that\n" +
			"the branch's org is not running; that the ledger's directory (the org\n" +
			"state dir) is not the feature worktree or inside it, since the ledger\n" +
			"would go with the worktree; that the record is the binding's (its\n" +
			"branch, path, and kind org); that ralph does not run inside the\n" +
			"worktree; that the worktree has no uncommitted change or untracked file\n" +
			"and no rebase or merge in progress (files .gitignore names do not count\n" +
			"and go with the worktree, but for the ledger's directory); that the\n" +
			"local branch is at the head the gate passed; and that the worktree has\n" +
			"that branch checked out at that head. The ledger's directory and the\n" +
			"working directory are compared with the worktree by file identity, so\n" +
			"another spelling of the same directory counts too.\n" +
			"When a check fails, it removes nothing. For a worktree whose directory\n" +
			"is gone but which git still registers, that last check is made on the\n" +
			"registration: it must be on that branch, and the branch must still\n" +
			"exist. A cleanup that stopped part way goes on from what is left. After\n" +
			"the lines above, merge prints CLEANUP:\n" +
			"done, with (resumed: <what was left>) when it went on from such a\n" +
			"cleanup, or CLEANUP: skipped (<why>) followed, when there is one, by\n" +
			"CLEANUP MANUAL: <the command that cleans up by hand once the reason is\n" +
			"checked>. Only a cleanup ralph cannot go on with by itself has such a\n" +
			"command (the record is gone or is not the binding's, the worktree is\n" +
			"left with no local branch, the worktree's directory is gone while git's\n" +
			"registration of it has a HEAD that removing it would drop, or the\n" +
			"removal itself failed); after any other skip, fix what the reason says\n" +
			"and run ralph org cleanup --branch <branch> again. The cleanup never\n" +
			"changes the exit code. A skip because the ledger's directory is inside\n" +
			"the feature worktree has no CLEANUP MANUAL: either, since cleaning up\n" +
			"by hand would remove the ledger with it: the reason says to move the\n" +
			"ledger's directory out of the worktree, to a place where no other\n" +
			"ledger is, once no org on the ledger runs and no ralph org command\n" +
			"runs on it, and to run ralph org cleanup --branch <branch> again with\n" +
			"--state-dir or RALPH_ORG_STATE_DIR naming the new place. So does a\n" +
			"record that is not the binding's (its path or kind) when the worktree\n" +
			"at the record's path holds the ledger; after the ledger is moved, that\n" +
			"ralph org cleanup is refused with a MANUAL: line.\n" +
			"\n" +
			"With --wait-checks <duration> (Go's form, such as 60m; 0, the default,\n" +
			"waits not at all; at most 6h, and a negative value or one over 6h is\n" +
			"refused before anything is read), merge waits instead of refusing when\n" +
			"the only refusal is that the branch's org is still running (its leader\n" +
			"disbands it after its RESULT), that a check is pending, or that gh pr\n" +
			"checks reports no checks yet. It releases regate.lock, so a ralph org\n" +
			"regate, another merge, or a ralph org cleanup can run meanwhile. For\n" +
			"the org, it reads only the ledger every 30 s, with no git and no gh,\n" +
			"until the org is disbanded, runs another feature (told apart only on a\n" +
			"branch with a feature binding; on a branch with only a gate record, the\n" +
			"wait goes on until the org is disbanded or the time is up), or the\n" +
			"ledger cannot be read; for the checks, it reads only gh pr checks <n>\n" +
			"-R <repo> every 30 s, until no check is pending any more or gh pr\n" +
			"checks fails. Either wait also ends when the time is up. Then merge\n" +
			"takes regate.lock again and runs every check above from the start, so a\n" +
			"branch that moved meanwhile is refused, and an org still running or\n" +
			"checks pending again are waited for again, all within the one duration.\n" +
			"An org that runs another feature now (on a branch with a feature\n" +
			"binding), a check that failed or was cancelled, checks that are all\n" +
			"skipping, and every other\n" +
			"refusal are not waited for. When the time is up and the org still\n" +
			"runs or the checks are still pending or none, merge refuses as without\n" +
			"the flag and adds how long it waited; for the org, it also names ralph\n" +
			"org status --org-id <org>, since whether to disband the org is a\n" +
			"human's call. stderr gets one line when a wait starts and one when it\n" +
			"ends. An interrupt while merge waits merges nothing and exits 1. In a\n" +
			"repository with no CI, the checks stay at none, and merge waits until\n" +
			"the time is up.\n" +
			"\n" +
			"An interrupt (SIGINT, SIGTERM, or SIGHUP) stops the git or gh call that\n" +
			"runs, up to and including gh pr merge. Before gh pr merge, nothing is\n" +
			"merged. Once gh pr merge has started, merge still reads the PR back once\n" +
			"and then removes a request or records the merge as above, prints the\n" +
			"lines when the PR merged, and exits 1 saying it was interrupted; an\n" +
			"interrupt once pr_merged is recorded skips the cleanup or stops it, and\n" +
			"the CLEANUP: line says how it ended. The read-back and the removal run\n" +
			"in a process group of their own, so a Ctrl-C or hangup the terminal\n" +
			"sends does not stop them, and each is still bounded by gh's 15 s\n" +
			"timeout. Exit codes: 0 for a merge that was recorded, whatever the\n" +
			"cleanup did; 1 for a refusal, a merge that did not happen, is unknown,\n" +
			"is of another head than the gate's, or went into another base than the\n" +
			"default branch, an interrupt, and any other error.\n" +
			"\n" +
			"merge starts no autonomous work, so it needs no running watch and no\n" +
			"approved plan. While it runs, except while it waits, ralph org regate,\n" +
			"another ralph org merge, and ralph org cleanup are refused on the\n" +
			"ledger. ralph org spawn\n" +
			"and start of the branch's org are refused only while merge cleans up\n" +
			"its feature worktree, since the merge itself changes no worktree.\n" +
			"[org.director].auto_merge is only reported, in AUTO MERGE: and in\n" +
			"pr_merged: merge merges whatever it is. The /director skill reads it to\n" +
			"know whether it may merge without asking. Without --config, it is read\n" +
			"from the main worktree's ralph.toml when the ledger is the main\n" +
			"worktree's, as ralph org escalate reads [org.director].enabled, so the\n" +
			"ralph.toml of a feature org's worktree does not change it; a main\n" +
			"worktree ralph.toml that does not load fails the command.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("org-id") {
				return errors.New("org: merge: --org-id does not apply: merge reads the org from the binding or the gate record of " +
					"the branch --branch names")
			}
			if strings.TrimSpace(branch) == "" {
				return errors.New("org: merge: --branch is required: name the feature branch whose PR to merge (ralph org merge-order " +
					"lists the ledger's branches)")
			}
			if refused := org.MergeWaitChecksRefusal(waitChecks); refused != "" {
				return errors.New(refused)
			}
			// newOrgRuntime's steps, split so that auto_merge is read by the
			// main worktree rule of ralph org escalate.
			resolvedStateDir, stateDirSource, err := resolveOrgLedger(cmd, *stateDir, orgLedgerMutating)
			if err != nil {
				return err
			}
			rt, err := newOrgRuntimeAt(resolvedStateDir, *configPath)
			if err != nil {
				return err
			}
			if rt.Config, err = withMainWorktreeDirectorEnabled(rt.Config, *configPath, resolvedStateDir, stateDirSource); err != nil {
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
			repoRoot, cleanupSkipped := mergeCleanupRoot(resolvedStateDir, stateDirSource, branch)
			res, err := rt.Merge(ctx, org.MergeParams{Branch: branch, AutoMerge: rt.Config.Director.AutoMerge, WaitChecks: waitChecks,
				Progress: cmd.ErrOrStderr(), RepoRoot: repoRoot})
			if res.Recorded && res.Cleanup == nil && cleanupSkipped != "" {
				res.Cleanup = &org.CleanupResult{Branch: res.Branch, Head: res.Head, Refused: cleanupSkipped}
			}
			return printMergeResult(cmd.OutOrStdout(), res, err)
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "", "the feature branch whose PR to merge, as ralph org merge-order lists it (required)")
	cmd.Flags().DurationVar(&waitChecks, "wait-checks", 0, "how long to wait, with regate.lock released, for checks that are "+
		"pending or not reported yet and for the branch's org to be disbanded before refusing, such as 60m (0 waits not at all; "+
		"at most 6h)")

	return cmd
}

// orgMainWorktreeMissing is why ralph org merge and ralph org cleanup find no
// main worktree when org.FeatureRepoRoot fails, said in place of its error,
// which names start --plan and where to run it: it fails only when the
// current directory is in no git repository, or in one where git names no
// main worktree (a linked worktree of a bare repository has none).
const orgMainWorktreeMissing = "the current directory is in no git repository, or in one with no main worktree, such as a " +
	"linked worktree of a bare repository"

// mergeCleanupRoot is the main worktree root merge passes as
// MergeParams.RepoRoot for the cleanup of the merged branch's feature
// worktree, found as ralph org start --plan finds it (org.FeatureRepoRoot).
// When it cannot be found, root is "", so Merge runs no cleanup, and skipped
// is the reason the CLEANUP: line gives instead (orgMainWorktreeMissing).
func mergeCleanupRoot(resolvedStateDir, stateDirSource, branch string) (root, skipped string) {
	root, err := org.FeatureRepoRoot(resolvedStateDir, stateDirSource)
	if err != nil {
		return "", org.PrintableInboxText(fmt.Sprintf("the main worktree, where scripts/ralph-worktree.sh removes the feature "+
			"worktree, was not found (%s), so the feature worktree of the branch %s was not cleaned up: run ralph org cleanup "+
			"--branch %s from the main worktree", orgMainWorktreeMissing, branch, shellQuoteIfNeeded(branch)), false)
	}
	return root, ""
}

// printMergeResult prints res, what Merge returned with err, and returns the
// command's error.
//
// When gh pr view read the PR back as MERGED at the gate's head into the
// default branch (res.Merged; a merge of another head, or into another base
// someone may have retargeted the PR to after the checks, is res.Failed,
// with no lines), stdout gets these lines first, also when err is set
// (pr_merged could not be recorded, or an interrupt came once gh pr merge had
// started):
//
//	MERGED: <PR url> (PR #<n> of <branch>)
//	METHOD: merge|squash|rebase
//	HEAD: <the head the gate passed>
//	MERGE COMMIT: <its SHA, or unknown>
//	AUTO MERGE: true|false
//	PROTECTION: <what keeps the merge up to date>
//	NOTE: <how gh pr merge failed though the PR merged> (only with res.Note)
//	CLEANUP: done|skipped (<reason>) (only with res.Cleanup)
//	CLEANUP MANUAL: <the command by hand> (only after a skipped one with it)
//
// The CLEANUP: lines say how the cleanup of the branch's feature worktree
// went once pr_merged was recorded (printMergeCleanup). The URL, branch,
// head, merge commit, protection, and the cleanup's reason and command go
// through PrintableInboxText, as merge-order's values from gh do; res.Note,
// escaped by Merge, keeps the line breaks of gh's tail and is printed as it
// is.
//
// The error is err when set; else res.Failed or res.Refused, with nothing on
// stdout (exit 1); else nil once a recorded merge was printed (exit 0),
// whatever the cleanup did. A result with none of these, which Merge does not
// return, is an error too, so that exit 0 always means a recorded merge.
func printMergeResult(w io.Writer, res org.MergeResult, err error) error {
	if res.Merged {
		printMergeLines(w, res)
	}
	switch {
	case err != nil:
		return err
	case res.Failed != "":
		return errors.New(res.Failed)
	case res.Refused != "":
		return errors.New(res.Refused)
	case !res.Merged || !res.Recorded:
		return fmt.Errorf("org: merge: ralph org merge ended with no outcome for the branch %s (merged=%t, recorded=%t), so whether "+
			"its PR was merged is unknown: run ralph org merge-order to see the branch", org.PrintableInboxText(res.Branch, false),
			res.Merged, res.Recorded)
	}
	return nil
}

// printMergeLines prints printMergeResult's lines for res (Merged set).
func printMergeLines(w io.Writer, res org.MergeResult) {
	_, _ = fmt.Fprintf(w, "MERGED: %s (PR #%d of %s)\n", org.PrintableInboxText(res.PRURL, false), res.PRNumber,
		org.PrintableInboxText(res.Branch, false))
	_, _ = fmt.Fprintf(w, "METHOD: %s\n", res.Method)
	_, _ = fmt.Fprintf(w, "HEAD: %s\n", org.PrintableInboxText(res.Head, false))
	_, _ = fmt.Fprintf(w, "MERGE COMMIT: %s\n", cmp.Or(org.PrintableInboxText(res.MergeCommit, false), "unknown"))
	_, _ = fmt.Fprintf(w, "AUTO MERGE: %t\n", res.AutoMerge)
	_, _ = fmt.Fprintf(w, "PROTECTION: %s\n", org.PrintableInboxText(res.Protection, false))
	if res.Note != "" {
		_, _ = fmt.Fprintf(w, "NOTE: %s\n", res.Note)
	}
	printMergeCleanup(w, res)
}

// printMergeCleanup prints the CLEANUP: line of res.Cleanup, nothing when it
// is nil:
//
//   - CLEANUP: done when the cleanup removed the feature worktree, its branch,
//     and its record, followed by "(resumed: <what an earlier cleanup left>)"
//     when it went on from a cleanup that stopped part way, and by
//     res.CleanupErr when worktree_cleaned could not be recorded, both in one
//     pair of parentheses, separated by "; ";
//   - CLEANUP: skipped (<reason>) otherwise: its Refused, else its Failed,
//     else res.CleanupErr, then CLEANUP MANUAL: <command> when it names the
//     command that cleans up by hand.
func printMergeCleanup(w io.Writer, res org.MergeResult) {
	c := res.Cleanup
	if c == nil {
		return
	}
	if c.Cleaned {
		var notes []string
		if c.Resumed != "" {
			notes = append(notes, "resumed: "+c.Resumed)
		}
		if res.CleanupErr != "" {
			notes = append(notes, res.CleanupErr)
		}
		line := "CLEANUP: done"
		if len(notes) > 0 {
			line += " (" + org.PrintableInboxText(strings.Join(notes, "; "), false) + ")"
		}
		_, _ = fmt.Fprintln(w, line)
		return
	}
	reason := cmp.Or(c.Refused, c.Failed, res.CleanupErr, "the cleanup ended with no outcome")
	_, _ = fmt.Fprintf(w, "CLEANUP: skipped (%s)\n", org.PrintableInboxText(reason, false))
	if c.Manual != "" {
		_, _ = fmt.Fprintf(w, "CLEANUP MANUAL: %s\n", org.PrintableInboxText(c.Manual, false))
	}
}
