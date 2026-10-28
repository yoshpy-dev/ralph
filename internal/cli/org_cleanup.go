package cli

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// newOrgCleanupCmd wires `ralph org cleanup [--branch <b>]` (plan
// docs/plans/active/2026-10-28-org-merge-cleanup.md, S3: AC2, and AC4 at the
// command line): (*org.Org).MergeOrder reads the ledger's feature branches
// and their PRs as ralph org merge-order --all reads them, with git and gh in
// the current directory (Dir left empty, Exec nil); cleanupSteps picks the
// branches whose PR gh reads as merged; and (*org.Org).CleanupFeatureWorktree
// cleans up each one's feature worktree at its PR's head, taking regate.lock
// itself, while cleanupPrinter prints a line per branch. The main worktree
// root comes from org.FeatureRepoRoot, as for ralph org start --plan and
// merge. The list holds every org's branches, so --org-id is refused before
// anything is read, as merge-order refuses it, and the context is canceled on
// SIGINT, SIGTERM, or SIGHUP (signal.NotifyContext), as merge's is.
func newOrgCleanupCmd(stateDir, configPath *string) *cobra.Command {
	var branch string

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up the feature worktree of a feature branch whose PR was merged on GitHub",
		Long: "ralph org cleanup cleans up what ralph org start --plan made for a\n" +
			"feature org once the PR of its branch was merged on GitHub (in the\n" +
			"browser, or with gh pr merge): the feature worktree, the local branch,\n" +
			"and the record of scripts/ralph-worktree.sh. It removes them with\n" +
			"./scripts/ralph-worktree.sh cleanup --id org-<org_id> --force-branch in\n" +
			"the main worktree (found as start --plan finds it) and records\n" +
			"worktree_cleaned in the ledger with via=cleanup. ralph org merge does the\n" +
			"same cleanup right after its own merge, so cleanup is for a PR merged\n" +
			"some other way, or for a cleanup that merge skipped. Run it from the\n" +
			"main worktree, not from inside the feature worktree it removes.\n" +
			"\n" +
			"Without --branch, cleanup reads the ledger's branches as ralph org\n" +
			"merge-order --all does (one git ls-remote, and gh pr list for each\n" +
			"branch) and cleans up each branch with a feature binding whose PR gh\n" +
			"reads as MERGED, at the PR's head (headRefOid). A branch whose PR gh\n" +
			"could not read (pr_lookup failed, limit_forks, or skipped in\n" +
			"merge-order's --json) gets a SKIPPED: line, since whether it was merged\n" +
			"is not known. A branch whose PR is open, closed, or not found, one with\n" +
			"only a gate record, and one cleaned up already, with nothing of it\n" +
			"left, get no line. The old branch of an org_id that was bound again to\n" +
			"another branch later is handled only when --branch names it. With\n" +
			"--branch <branch>, cleanup cleans up that branch alone, and refuses,\n" +
			"removing nothing, when the ledger does not name it, it has no feature\n" +
			"binding, gh does not read its PR, or the PR is not MERGED.\n" +
			"\n" +
			"Before it removes anything, cleanup checks that the branch's org is not\n" +
			"running; that the record is the binding's (its branch, path, and kind\n" +
			"org); that ralph does not run inside the worktree; that the worktree has\n" +
			"no uncommitted change or untracked file and no rebase or merge in\n" +
			"progress (files .gitignore names do not count and go with the\n" +
			"worktree); that the local branch is at the head the PR merged; and that\n" +
			"the worktree has that branch checked out at that head, so that no\n" +
			"commit that was not merged is lost. When a check fails, it removes\n" +
			"nothing. A cleanup that stopped part way goes on from what is left.\n" +
			"While it cleans up a branch, ralph org regate, ralph org merge, and\n" +
			"another cleanup are refused on the ledger (one runs at a time; it waits\n" +
			"up to 1 s for regate.lock), and so are ralph org spawn and start of the\n" +
			"org whose feature worktree it removes.\n" +
			"\n" +
			"It prints one line per branch, CLEANED: <branch> org-<org_id>, with\n" +
			"(resumed: <what was left>) when it went on from a cleanup that stopped\n" +
			"part way, or SKIPPED: <branch> <why>, followed, when there is one, by\n" +
			"MANUAL: <the command that cleans up by hand once the reason is\n" +
			"checked>. With no such line, it prints nothing to clean up. Only a\n" +
			"branch ralph cannot go on with by itself gets a MANUAL: line (the\n" +
			"record is gone or is not the binding's, the worktree is left with no\n" +
			"local branch, or the removal itself failed); after any other SKIPPED:\n" +
			"line, fix what the reason says and run ralph org cleanup --branch\n" +
			"<branch> again. Exit codes: 0 when every branch was cleaned up or\n" +
			"there was nothing to clean up; 1 when a branch was skipped, gh is not\n" +
			"on PATH (whether a PR was merged cannot be told, so nothing is\n" +
			"removed), the main worktree is not found, --branch is refused, for an\n" +
			"interrupt (SIGINT, SIGTERM, or SIGHUP; what was cleaned up before it\n" +
			"stays cleaned up), and for any other error.\n" +
			"\n" +
			"cleanup starts no autonomous work, so it needs no running watch and no\n" +
			"approved plan. It reads the branches of every org in the ledger, so\n" +
			"--org-id is refused.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("org-id") {
				return errors.New("org: cleanup: --org-id does not apply: cleanup reads the org from the binding of each merged " +
					"feature branch in the ledger")
			}
			if cmd.Flags().Changed("branch") && strings.TrimSpace(branch) == "" {
				return errors.New("org: cleanup: --branch is empty: name the merged feature branch to clean up, or leave --branch " +
					"out to clean up every merged one")
			}
			resolvedStateDir, stateDirSource := org.ResolveOrgStateDir(*stateDir, cmd.Flags().Changed("state-dir"))
			if err := guardLegacyOrgStateDir(cmd, resolvedStateDir, stateDirSource, orgLedgerMutating); err != nil {
				return err
			}
			rt, err := newOrgRuntimeAt(resolvedStateDir, *configPath)
			if err != nil {
				return err
			}
			parent := cmd.Context()
			if parent == nil {
				parent = context.Background()
			}
			// As for ralph org merge (newOrgMergeCmd).
			ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
			defer stop()
			repoRoot, err := org.FeatureRepoRoot(resolvedStateDir, stateDirSource)
			if err != nil {
				// err is one of the two cases orgMainWorktreeMissing says.
				return errors.New("org: cleanup: the main worktree, where scripts/ralph-worktree.sh removes the feature worktrees, " +
					"was not found (" + orgMainWorktreeMissing + "), so nothing was read or removed: run ralph org cleanup from the " +
					"main worktree")
			}
			res, err := rt.MergeOrder(ctx, org.MergeOrderParams{All: true})
			if err != nil {
				return fmt.Errorf("org: cleanup: nothing was removed, since the feature branches could not be read as ralph org "+
					"merge-order --all reads them: %w", err)
			}
			steps, err := cleanupSteps(res, branch)
			if err != nil {
				return err
			}
			return runCleanupSteps(ctx, rt, repoRoot, steps, &cleanupPrinter{w: cmd.OutOrStdout()})
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "", "clean up only this feature branch, as ralph org merge-order --all lists it "+
		"(default: every branch whose PR is merged)")

	return cmd
}

// cleanupStep is one branch ralph org cleanup deals with: cleaned up at head,
// the head its PR merged, or, when unknown is set, only reported with that
// reason, since gh did not tell whether its PR was merged.
type cleanupStep struct {
	branch, head, unknown string
}

// cleanupSteps picks from res, merge-order's result with All, the branches
// ralph org cleanup deals with, in merge-order's order. Without branch, they
// are the entries with a feature binding that is still its org's latest
// (BoundTo empty): one whose PR gh reads as merged is cleaned up at the PR's
// head, and one whose PR lookup is neither found nor none is reported as
// unknown; any other entry is left out. With branch, it is that entry alone,
// whatever its BoundTo (the cleanup refuses a record that is not the
// binding's), and it is refused unless it has a feature binding and gh reads
// its PR as merged. No entry at all is nothing to clean up without branch.
// The error is a refusal, nothing removed: a branch the ledger does not name,
// gh not found, and the refusals of branch (cleanupNamedStep).
func cleanupSteps(res org.MergeOrderResult, branch string) ([]cleanupStep, error) {
	switch {
	case len(res.Entries) == 0 && branch == "":
		return nil, nil
	case len(res.Entries) == 0:
		return nil, cleanupNotInLedger(branch)
	case !res.GHAvailable:
		return nil, errors.New("org: cleanup: gh is not on PATH, so whether the PRs of the ledger's feature branches were " +
			"merged cannot be told, and nothing was removed: put gh on PATH and run ralph org cleanup again")
	case branch != "":
		step, err := cleanupNamedStep(res, branch)
		if err != nil {
			return nil, err
		}
		return []cleanupStep{step}, nil
	}
	var steps []cleanupStep
	for _, e := range res.Entries {
		if e.Binding == nil || e.Binding.BoundTo != "" {
			continue
		}
		switch {
		case e.Class == org.MergeClassMerged && e.PR != nil:
			steps = append(steps, cleanupStep{branch: e.Branch, head: e.PR.HeadRefOid})
		case e.PRLookup != org.MergeOrderPRLookupFound && e.PRLookup != org.MergeOrderPRLookupNone:
			steps = append(steps, cleanupStep{branch: e.Branch, unknown: "whether its PR was merged is not known (" +
				cleanupLookupWhy(e.PRLookup) + "; the notes of ralph org merge-order --all say more), so nothing was removed: " +
				"once gh reads its PR, run ralph org cleanup again"})
		}
	}
	return steps, nil
}

// cleanupNamedStep is cleanupSteps' step for --branch branch, or its refusal:
// the ledger does not name the branch, it has no feature binding (gate
// records only), gh found no PR for it or did not tell, or its PR is not
// merged.
func cleanupNamedStep(res org.MergeOrderResult, branch string) (cleanupStep, error) {
	i := slices.IndexFunc(res.Entries, func(e org.MergeOrderEntry) bool { return e.Branch == branch })
	if i < 0 {
		return cleanupStep{}, cleanupNotInLedger(branch)
	}
	e, b := res.Entries[i], org.PrintableInboxText(branch, false)
	switch {
	case e.Binding == nil:
		return cleanupStep{}, fmt.Errorf("org: cleanup: no complete feature binding in the ledger names the branch %s, only a gate "+
			"record, so it has no feature worktree to clean up and nothing was removed: cleanup removes only the worktree ralph "+
			"org start --plan made for a feature org", b)
	case e.PRLookup == org.MergeOrderPRLookupNone:
		return cleanupStep{}, fmt.Errorf("org: cleanup: gh found no PR for the branch %s, so nothing was removed: cleanup "+
			"removes the feature worktree of a branch whose PR was merged", b)
	case e.PR == nil:
		return cleanupStep{}, fmt.Errorf("org: cleanup: whether the PR of the branch %s was merged is not known (%s; the notes "+
			"of ralph org merge-order --all say more), so nothing was removed: once gh reads its PR, run ralph org cleanup "+
			"--branch %s again", b, cleanupLookupWhy(e.PRLookup), b)
	case e.Class != org.MergeClassMerged:
		return cleanupStep{}, fmt.Errorf("org: cleanup: PR #%d of the branch %s is %s, not MERGED, so nothing was removed: "+
			"cleanup removes the feature worktree of a branch whose PR was merged", e.PR.Number, b,
			org.PrintableInboxText(e.PR.State, false))
	}
	return cleanupStep{branch: e.Branch, head: e.PR.HeadRefOid}, nil
}

// cleanupNotInLedger is the refusal of a --branch the ledger does not name.
func cleanupNotInLedger(branch string) error {
	return fmt.Errorf("org: cleanup: no feature binding or gate record in the ledger names the branch %s, so nothing was "+
		"removed: ralph org merge-order --all lists the ledger's branches", org.PrintableInboxText(branch, false))
}

// cleanupLookupWhy says what gh's PR lookup (MergeOrderEntry.PRLookup) gave
// for a branch whose PR is not used.
func cleanupLookupWhy(lookup string) string {
	switch lookup {
	case org.MergeOrderPRLookupFailed:
		return "gh pr list failed for the branch"
	case org.MergeOrderPRLookupLimitForks:
		return "gh returned its limit of PRs for the branch, all from forks"
	case org.MergeOrderPRLookupSkipped:
		return "no answer of gh was used"
	}
	return "gh's PR lookup gave " + strconv.Quote(lookup)
}

// runCleanupSteps deals with steps in order, printing with p: a step with
// unknown set gets its SKIPPED: line, and every other one is cleaned up with
// CleanupFeatureWorktree (Via cleanup, Cwd the working directory). It stops
// at the first error of a cleanup, and before the next cleanup when ctx is
// done; otherwise it returns p.finish().
func runCleanupSteps(ctx context.Context, rt *org.Org, repoRoot string, steps []cleanupStep, p *cleanupPrinter) error {
	for _, s := range steps {
		if s.unknown != "" {
			p.unknown(s.branch, s.unknown)
			continue
		}
		if ctx.Err() != nil {
			return fmt.Errorf("org: cleanup: interrupted (%v) before the cleanup of the branch %s: it and the branches after it "+
				"were not cleaned up, and the lines above say what was; run ralph org cleanup again", context.Cause(ctx),
				org.PrintableInboxText(s.branch, false))
		}
		res, err := rt.CleanupFeatureWorktree(ctx, org.CleanupParams{Branch: s.branch, Head: s.head, RepoRoot: repoRoot,
			Via: org.CleanupViaCleanup})
		if err := p.result(s.branch, res, err); err != nil {
			return err
		}
	}
	return p.finish()
}

// cleanupPrinter prints ralph org cleanup's lines on w as each branch is
// dealt with, and counts the branches left:
//
//	CLEANED: <branch> org-<org_id>                (the cleanup removed it)
//	CLEANED: <branch> org-<org_id> (<notes>)      (resumed: <kind>, the error, or both, "; " between)
//	SKIPPED: <branch> <why>                       (refused, failed, or whether its PR merged is unknown)
//	MANUAL: <the command by hand>                 (only after a SKIPPED: line with one)
//	nothing to clean up                           (only when no line above was printed)
//
// Every value goes through PrintableInboxText, as printMergeCleanup's do.
type cleanupPrinter struct {
	w       io.Writer
	printed int // CLEANED: and SKIPPED: lines
	left    int // branches with a SKIPPED: line
}

// unknown prints the SKIPPED: line of branch, which was not cleaned up since
// whether its PR was merged is not known (why).
func (p *cleanupPrinter) unknown(branch, why string) {
	p.skipped(branch, why, "")
}

// result prints the line of branch's cleanup, which returned res and err,
// and returns err, which stops ralph org cleanup: CLEANED: when res.Cleaned,
// with err in its parentheses when set (worktree_cleaned could not be
// recorded); no line for another err, or for res.NothingLeft (cleaned up
// already, not counted as left); else SKIPPED: with res.Refused or
// res.Failed, and MANUAL: with res.Manual when set.
func (p *cleanupPrinter) result(branch string, res org.CleanupResult, err error) error {
	switch {
	case res.Cleaned:
		var notes []string
		if res.Resumed != "" {
			notes = append(notes, "resumed: "+res.Resumed)
		}
		if err != nil {
			notes = append(notes, err.Error())
		}
		line := "CLEANED: " + org.PrintableInboxText(branch, false) + " org-" + org.PrintableInboxText(res.OrgID, false)
		if len(notes) > 0 {
			line += " (" + org.PrintableInboxText(strings.Join(notes, "; "), false) + ")"
		}
		_, _ = fmt.Fprintln(p.w, line)
		p.printed++
		return err
	case err != nil:
		return err
	case res.NothingLeft:
		return nil
	}
	p.skipped(branch, cmp.Or(res.Refused, res.Failed, "the cleanup ended with no outcome"), res.Manual)
	return nil
}

// skipped prints SKIPPED: <branch> <why>, then MANUAL: <manual> when manual
// is set, and counts branch as left.
func (p *cleanupPrinter) skipped(branch, why, manual string) {
	_, _ = fmt.Fprintf(p.w, "SKIPPED: %s %s\n", org.PrintableInboxText(branch, false), org.PrintableInboxText(why, false))
	if manual != "" {
		_, _ = fmt.Fprintf(p.w, "MANUAL: %s\n", org.PrintableInboxText(manual, false))
	}
	p.printed++
	p.left++
}

// finish prints nothing to clean up when no line was printed, and returns
// ralph org cleanup's error: nil when no branch was left (exit 0), else one
// that counts the branches left (exit 1).
func (p *cleanupPrinter) finish() error {
	if p.printed == 0 {
		_, _ = fmt.Fprintln(p.w, "nothing to clean up")
	}
	if p.left > 0 {
		return fmt.Errorf("org: cleanup: %d branch(es) were not cleaned up: the SKIPPED: line of each says why and what to do next",
			p.left)
	}
	return nil
}
