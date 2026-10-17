package cli

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
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
// SIGTERM, or SIGHUP (gateSignalContext), and prints the result
// (printMergeResult). Merge reads the org from the branch's binding or gate
// record (D4), so --org-id is refused before anything is read, as regate
// refuses it, and so are a missing --branch and a --wait-checks out of range
// (org.MergeWaitChecksRefusal). --wait-checks (S7, D13) reaches Merge with
// stderr for the lines of its wait.
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
			"fails: that no other ralph org regate or merge runs on the ledger (one\n" +
			"runs at a time; it waits up to 1 s for regate.lock); that origin's URL\n" +
			"names a host and owner/repo; that gh is on PATH; that a feature binding\n" +
			"or a gate record in the ledger names the branch; that git ls-remote\n" +
			"reads origin; that gh pr list finds the branch's PR in origin's\n" +
			"repository (pr_lookup found, in merge-order's terms) and its url names\n" +
			"that repository; that merge-order classifies the branch as ready; that\n" +
			"the branch's org is not running (its leader may still push: handle the\n" +
			"org's RESULT in the inbox, and once the org is disbanded, run merge\n" +
			"again); that gh pr view reads the PR as OPEN, with the head the gate\n" +
			"passed as its head, origin's default branch as its base, and no\n" +
			"auto-merge request (the refusal names gh pr merge <n> --disable-auto -R\n" +
			"<repo>, which removes one); that every check gh pr checks lists is pass\n" +
			"or skipping, and at least one is pass; that GitHub's protection of the\n" +
			"default branch, read live with gh api and not from the cache ralph\n" +
			"doctor writes, shows no merge queue and keeps the merge up to date for\n" +
			"you: a ruleset whose up-to-date rule (Require branches to be up to date\n" +
			"before merging) has required status checks of its own and whose\n" +
			"current_user_can_bypass is never, or else a branch protection that is\n" +
			"strict with required checks while you are not an admin of the\n" +
			"repository or enforce_admins is enabled (its strict can be read only\n" +
			"with admin access, so the refusal says to use a ruleset instead); that\n" +
			"gh repo view gives the merge method (viewerDefaultMergeMethod: MERGE,\n" +
			"SQUASH, or REBASE); and, last, that a second git ls-remote finds\n" +
			"origin's default branch and the branch where the gate left them. A git\n" +
			"or gh call that fails is refused too. A refusal merges and records\n" +
			"nothing, says why and what to do next on stderr, and exits 1.\n" +
			"\n" +
			"Then merge runs gh pr merge <n> --merge|--squash|--rebase\n" +
			"--match-head-commit <the gate's head> -R <repo> once, for up to 60 s,\n" +
			"never with --auto, --admin, or --delete-branch, so GitHub merges only\n" +
			"while the PR's head is the gate's. Whatever that returned, merge reads\n" +
			"the PR back once with gh pr view. When the PR is MERGED, merge records\n" +
			"pr_merged for the org in the ledger, prints MERGED: <PR url> (PR #<n> of\n" +
			"<branch>), METHOD:, HEAD:, MERGE COMMIT: (unknown when gh gave none),\n" +
			"AUTO MERGE:, PROTECTION: (the setting that keeps the merge up to date),\n" +
			"and NOTE: when gh pr merge failed or timed out though the PR merged, and\n" +
			"exits 0. When the PR is not MERGED, merge records nothing, says what\n" +
			"happened on stderr with the end of gh pr merge's error when it failed,\n" +
			"and exits 1; an auto-merge request GitHub holds instead of merging is\n" +
			"removed with gh pr merge <n> --disable-auto -R <repo>, and when that\n" +
			"fails, merge prints that command for you to run. When the read fails,\n" +
			"whether the PR merged is unknown: merge records nothing, names the gh\n" +
			"pr view that checks it, and exits 1. When the PR merged but pr_merged\n" +
			"cannot be recorded, merge prints the lines above and exits 1.\n" +
			"\n" +
			"With --wait-checks <duration> (Go's form, such as 60m; 0, the default,\n" +
			"waits not at all; at most 6h, and a negative value or one over 6h is\n" +
			"refused before anything is read), merge waits for the checks instead of\n" +
			"refusing when the only refusal is that a check is pending, or that gh pr\n" +
			"checks reports no checks yet. It releases regate.lock, so a ralph org\n" +
			"regate or another merge can run meanwhile, and reads only gh pr checks\n" +
			"<n> -R <repo> every 30 s, until no check is pending any more, gh pr\n" +
			"checks fails, or the time is up; then it takes regate.lock again and runs\n" +
			"every check above from the start, so a branch that moved meanwhile is\n" +
			"refused, and checks pending again are waited for again. A check that\n" +
			"failed or was cancelled, checks that are all skipping, and every other\n" +
			"refusal are not waited for. When the time is up and the checks are still\n" +
			"pending or none, merge refuses as without the flag and adds how long it\n" +
			"waited. stderr gets one line when a wait starts and one when it ends. An\n" +
			"interrupt while merge waits merges nothing and exits 1. In a repository\n" +
			"with no CI, the checks stay at none, and merge waits until the time is up.\n" +
			"\n" +
			"An interrupt (SIGINT, SIGTERM, or SIGHUP) stops the git or gh call that\n" +
			"runs, up to and including gh pr merge. Before gh pr merge, nothing is\n" +
			"merged. Once gh pr merge has started, merge still reads the PR back once\n" +
			"and then removes a request or records the merge as above, prints the\n" +
			"lines when the PR merged, and exits 1 saying it was interrupted. The\n" +
			"read-back and the removal run in a process group of their own, so a\n" +
			"Ctrl-C or hangup the terminal sends does not stop them, and each is\n" +
			"still bounded by gh's 15 s timeout. Exit codes: 0 for a merge that was\n" +
			"recorded; 1 for a refusal, a merge that did not happen or is unknown, an\n" +
			"interrupt, and any other error.\n" +
			"\n" +
			"merge starts no autonomous work, so it needs no running watch and no\n" +
			"approved plan. While it runs, except while it waits for the checks,\n" +
			"ralph org regate and another ralph org merge are refused on the ledger,\n" +
			"but ralph org spawn and start of the branch's org are not, since merge\n" +
			"changes no worktree.\n" +
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
			resolvedStateDir, stateDirSource := org.ResolveOrgStateDir(*stateDir, cmd.Flags().Changed("state-dir"))
			if err := guardLegacyOrgStateDir(cmd, resolvedStateDir, stateDirSource, orgLedgerMutating); err != nil {
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
			ctx, stop := gateSignalContext(parent, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
			defer stop()
			res, err := rt.Merge(ctx, org.MergeParams{Branch: branch, AutoMerge: rt.Config.Director.AutoMerge, WaitChecks: waitChecks,
				Progress: cmd.ErrOrStderr()})
			return printMergeResult(cmd.OutOrStdout(), res, err)
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "", "the feature branch whose PR to merge, as ralph org merge-order lists it (required)")
	cmd.Flags().DurationVar(&waitChecks, "wait-checks", 0, "how long to wait, with regate.lock released, for checks that are "+
		"pending or not reported yet before refusing, such as 60m (0 waits not at all; at most 6h)")

	return cmd
}

// printMergeResult prints res, what Merge returned with err, and returns the
// command's error.
//
// When gh pr view read the PR back as MERGED (res.Merged), stdout gets these
// lines first, also when err is set (pr_merged could not be recorded, or an
// interrupt came once gh pr merge had started):
//
//	MERGED: <PR url> (PR #<n> of <branch>)
//	METHOD: merge|squash|rebase
//	HEAD: <the head the gate passed>
//	MERGE COMMIT: <its SHA, or unknown>
//	AUTO MERGE: true|false
//	PROTECTION: <what keeps the merge up to date>
//	NOTE: <how gh pr merge failed though the PR merged> (only with res.Note)
//
// The URL, branch, head, merge commit, and protection go through
// PrintableInboxText, as merge-order's values from gh do; res.Note, escaped
// by Merge, keeps the line breaks of gh's tail and is printed as it is.
//
// The error is err when set; else res.Failed or res.Refused, with nothing on
// stdout (exit 1); else nil once a recorded merge was printed (exit 0). A
// result with none of these, which Merge does not return, is an error too, so
// that exit 0 always means a recorded merge.
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
}
