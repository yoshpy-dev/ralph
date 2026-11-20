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
// stderr for the lines of its wait, and --requeue as MergeParams.Requeue (plan
// docs/plans/active/2026-11-20-org-merge-queue-verb.md, D6), which only the
// merge queue path reads. The main worktree root (mergeCleanupRoot)
// reaches Merge as MergeParams.RepoRoot, so that Merge cleans up the branch's
// feature worktree once pr_merged is recorded (plan
// docs/plans/active/2026-10-28-org-merge-cleanup.md, S2), and
// orgInboxHintStateDir's value reaches it as MergeParams.HintStateDir, so
// that the commands its refusals and the cleanup's name keep a --state-dir the
// command was given: the command to run again, and the commands of another
// verb such as ralph org merge-order. The refusal of a missing --branch and
// printMergeResult name ralph org merge-order the same way (orgVerbHint).
func newOrgMergeCmd(stateDir, configPath *string) *cobra.Command {
	var branch string
	var requeue bool
	var waitChecks time.Duration

	cmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge, with gh, the PR of a feature branch ralph org merge-order lists as ready or accepted",
		Long: "ralph org merge merges, with gh, the PR of one feature branch that\n" +
			"ralph org merge-order lists as ready, or as accepted (its latest gate\n" +
			"record is a failure a human accepted with ralph org gate accept), after\n" +
			"it has checked again, live, what the merge rests on, the same way for\n" +
			"both. Name the branch as merge-order lists it: merge reads the org from\n" +
			"the branch's latest complete feature binding, else from its latest\n" +
			"gate record, so --org-id is refused. git and gh run in\n" +
			"the current directory's repository, as for merge-order, and the merge\n" +
			"goes into origin's repository: merge reads its host and owner/repo from\n" +
			"git remote get-url origin (an origin that names none, such as a local\n" +
			"path, is refused) and pins every gh call to it, gh pr with -R\n" +
			"<host>/<owner>/<repo>, gh repo view with that as its argument, and gh\n" +
			"api with --hostname <host>, so neither gh repo set-default nor an\n" +
			"upstream remote makes gh read another repository. For an ssh origin\n" +
			"(ssh:// or scp-like), the host is the hostname ssh -G <host> prints, as\n" +
			"gh resolves a remote, so an ssh alias (a Host in ~/.ssh/config) is read\n" +
			"on its real host. ssh -G connects to nothing, but it runs the commands\n" +
			"of any Match exec in the ssh config.\n" +
			"\n" +
			"First, merge checks, in this order, and refuses at the first check that\n" +
			"fails: that no other ralph org regate, merge, or cleanup runs on the\n" +
			"ledger (one runs at a time; it waits up to 1 s for regate.lock); that\n" +
			"origin's URL names a host and owner/repo; that gh is on PATH; that a\n" +
			"feature binding or a gate record in the ledger names the branch; that\n" +
			"git ls-remote reads origin; that gh pr list finds the branch's PR in\n" +
			"origin's repository (pr_lookup found, in merge-order's terms) and its\n" +
			"url names that repository; that merge-order classifies the branch as\n" +
			"ready or accepted (for an accepted branch, the head and base below are\n" +
			"those of the gate failure a human accepted; a branch whose gate base\n" +
			"origin's default branch moved past is ready or accepted only when that\n" +
			"branch has a merge queue, which merge reads for this check only then);\n" +
			"that the branch's org is not running (its leader may still push: handle\n" +
			"the org's RESULT in the inbox, and once the org is disbanded, run merge\n" +
			"again); that gh pr view reads the PR as OPEN, with the head the\n" +
			"gate passed as its head, origin's default branch as its base, and no\n" +
			"auto-merge request (the refusal names gh pr merge <n> --disable-auto -R\n" +
			"<repo>, which removes one); that every check gh pr checks lists is pass\n" +
			"or skipping, and at least one is pass; that GitHub's protection of the\n" +
			"default branch, read live with gh api and not from the cache ralph\n" +
			"doctor writes, keeps the merge up to date for you: a ruleset whose\n" +
			"up-to-date rule (Require branches to be up to date before merging) has\n" +
			"required status checks of its own and whose current_user_can_bypass is\n" +
			"never, or else a branch protection that is strict with required checks\n" +
			"and has enforce_admins enabled, whether or not you are an admin of the\n" +
			"repository, since without it an admin or a custom role that may bypass\n" +
			"branch protections can merge a head that is not up to date (its strict\n" +
			"can be read only with admin access, so the refusal says to use a\n" +
			"ruleset instead), or else a merge queue with at least one required\n" +
			"status check (the merge queue path, below); that gh repo view gives the\n" +
			"merge method (viewerDefaultMergeMethod: MERGE, SQUASH, or REBASE),\n" +
			"except on the merge queue path; and,\n" +
			"last, that a second git ls-remote finds origin's default branch and the\n" +
			"branch where the gate left them (on the merge queue path, the tip of\n" +
			"origin's default branch need not be the gate's base: that ls-remote\n" +
			"checks only that origin's HEAD still names the same default branch and\n" +
			"that the branch's tip is still the gate's head). A git or gh call that\n" +
			"fails is refused too. A refusal merges and records nothing, says why\n" +
			"and what to do next on stderr, and exits 1.\n" +
			"\n" +
			"Then merge runs gh pr merge <n> --merge|--squash|--rebase\n" +
			"--match-head-commit <the gate's head> -R <repo> once, for up to 60 s,\n" +
			"never with --auto, --admin, or --delete-branch, so GitHub merges only\n" +
			"while the PR's head is the gate's. Whatever that returned, merge reads\n" +
			"the PR back once with gh pr view. When the PR is MERGED at the gate's\n" +
			"head into origin's default branch, merge records pr_merged for the org\n" +
			"in the ledger, prints MERGED: <PR url> (PR #<n> of <branch>), METHOD:,\n" +
			"HEAD:, MERGE COMMIT: (unknown when gh gave none), AUTO MERGE:,\n" +
			"PROTECTION: (the setting that keeps the merge up to date), GATE:\n" +
			"accepted (<failed reason>) by <who> at <time>: <the --reason text> for\n" +
			"an accepted branch, whose pr_merged also ends with gate=accepted, and\n" +
			"NOTE: when gh pr merge failed or timed out though the PR merged, and\n" +
			"exits 0.\n" +
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
			"When origin's default branch has a merge queue (GraphQL's mergeQueue of\n" +
			"the branch, or a merge_queue rule of a ruleset), merge takes the merge\n" +
			"queue path. The protection check then requires at least one required\n" +
			"status check, of the rules or else of the branch protection, refuses a\n" +
			"queue with none, and reads no up-to-date rule, bypass, or enforce_admins;\n" +
			"gh repo view is not run, and METHOD: is the merge method of the queue's\n" +
			"configuration (merge, squash, or rebase), or queue when GitHub gives\n" +
			"none. Once every check passed, merge reads the PR's merge queue state\n" +
			"with gh api graphql and puts the PR in the queue with GraphQL's\n" +
			"enqueuePullRequest at the gate's head, once, with no gh pr merge and no\n" +
			"--admin. A PR in the queue already at the gate's head is not added again,\n" +
			"and one GitHub removed from the queue after the branch's latest gate\n" +
			"record is refused without --requeue (below). With --wait-checks, merge\n" +
			"then releases regate.lock and waits, within what is left of that\n" +
			"duration, for the queue to merge the PR, reading only the PR's merge\n" +
			"queue state every 30 s. When a read finds the PR MERGED at the gate's\n" +
			"head into origin's default branch, merge records pr_merged under the\n" +
			"manifest lock only, its details ending with via=merge_queue, prints the\n" +
			"MERGED: lines above, with a NOTE: line when the PR was in the queue\n" +
			"already or --requeue added it again, and one when another merge that\n" +
			"waited for the same queue had recorded it already (merge then records\n" +
			"nothing more), takes regate.lock again for the cleanup below, and\n" +
			"exits 0. When --wait-checks leaves no time once the PR is in the queue\n" +
			"(0, the default, included: nothing is read after the enqueue), or the\n" +
			"time is up with the PR still in the queue, merge records nothing,\n" +
			"leaves the PR in the queue, prints QUEUED: <PR url> (PR #<n> of\n" +
			"<branch>), METHOD:, HEAD:, POSITION: and ENQUEUED AT: (when GitHub\n" +
			"gave the PR's entry in the queue), PROTECTION:, GATE: (for an accepted\n" +
			"branch), and NOTE: (when the PR was in the queue already or --requeue\n" +
			"added it again), says on stderr that the PR is in the queue, and\n" +
			"exits 7: run merge again to keep waiting, also once origin's default\n" +
			"branch moved (it reads the queue first and does not add the PR twice),\n" +
			"or ralph org cleanup --branch <branch> once GitHub merged the PR. When a\n" +
			"read finds the PR out of the queue and not merged, merge records\n" +
			"nothing, says why on stderr, with GitHub's removal of the PR and its\n" +
			"reason when one was read, and exits 1. It does the same when\n" +
			"the queue merged the PR at another head or into another base, and when\n" +
			"the last read of the wait failed, after which the PR may still be in\n" +
			"the queue. An interrupt while merge waits for the queue records nothing,\n" +
			"unless a read found the PR merged first, leaves the PR in the queue, and\n" +
			"exits 1.\n" +
			"\n" +
			"--requeue is for the merge queue path only: when GitHub removed the PR\n" +
			"from the queue after the branch's latest gate record, or less than 5\n" +
			"minutes before it (ralph's clock and GitHub's may differ), merge does\n" +
			"not put it back without the flag, since the queue already tried that\n" +
			"gate's head. Give it on purpose, for example after a flaky check, never\n" +
			"by default.\n" +
			"\n" +
			"Once pr_merged is recorded, merge cleans up the branch's feature\n" +
			"worktree, the one ralph org start --plan made, with regate.lock still\n" +
			"held (on the merge queue path, taken again): it runs\n" +
			"./scripts/ralph-worktree.sh cleanup --id org-<org_id> --force-branch in\n" +
			"the main worktree (found as start --plan finds it),\n" +
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
			"and run ralph org cleanup --branch <branch> again; the reason names that\n" +
			"command with the --state-dir merge was given, if any. The cleanup never\n" +
			"changes the exit code. A skip because the ledger's directory is inside\n" +
			"the feature worktree has no CLEANUP MANUAL: either, since cleaning up\n" +
			"by hand would remove the ledger with it: the reason says to move the\n" +
			"ledger's directory out of the worktree, to a place where no other\n" +
			"ledger is, once no org on the ledger runs and no ralph org command\n" +
			"runs on it, and to run ralph org cleanup --branch <branch> again with\n" +
			"--state-dir or [org].state_dir in ralph.local.toml naming the new\n" +
			"place. So does a record that is not the binding's (its path or kind)\n" +
			"when the worktree at the record's path holds the ledger; after the\n" +
			"ledger is moved, that ralph org cleanup is refused with a MANUAL: line.\n" +
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
			"the time is up. On the merge queue path, the wait for the queue (above)\n" +
			"runs within the same duration, counted from the first wait.\n" +
			"\n" +
			"An interrupt (SIGINT, SIGTERM, or SIGHUP) stops the git or gh call that\n" +
			"runs, up to and including gh pr merge, or on the merge queue path\n" +
			"enqueuePullRequest. Before gh pr merge, nothing is merged; before\n" +
			"enqueuePullRequest, nothing is added to the queue, and once it was sent,\n" +
			"merge records nothing, says the PR is or may be in the queue, and\n" +
			"exits 1. Once gh pr merge has started, merge still reads the PR back once\n" +
			"and then removes a request or records the merge as above, prints the\n" +
			"lines when the PR merged, and exits 1 saying it was interrupted; an\n" +
			"interrupt once pr_merged is recorded skips the cleanup or stops it, and\n" +
			"the CLEANUP: line says how it ended. The read-back and the removal run\n" +
			"in a process group of their own, so a Ctrl-C or hangup the terminal\n" +
			"sends does not stop them, and each is still bounded by gh's 15 s\n" +
			"timeout. Exit codes: 0 for a merge that was recorded, whatever the\n" +
			"cleanup did; 1 for a refusal, a merge that did not happen, is unknown,\n" +
			"is of another head than the gate's, or went into another base than the\n" +
			"default branch, an interrupt, and any other error; 7 for a PR that\n" +
			"merge left in the merge queue, not merged yet, with nothing recorded\n" +
			"(run merge again to keep waiting).\n" +
			"\n" +
			"merge starts no autonomous work, so it needs no running watch and no\n" +
			"approved plan. While it runs, except while it waits, ralph org regate,\n" +
			"another ralph org merge, and ralph org cleanup are refused on the\n" +
			"ledger. On the merge queue path, merge releases regate.lock when the\n" +
			"wait for the queue starts and takes it again only for the cleanup (not\n" +
			"at all when none runs), so none of them is refused in between. ralph\n" +
			"org spawn and start of the branch's org are refused only while merge\n" +
			"cleans up its feature worktree, since the merge itself changes no\n" +
			"worktree.\n" +
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
				// The ledger is not resolved yet, so orgInboxHintStateDir
				// cannot be used here; this repeats its rule: the value is
				// the --state-dir as org.ResolveOrgStateDir makes it
				// absolute, and only when --state-dir was passed. With
				// explicitSet true, ResolveOrgStateDir returns no error and
				// makes no git call, so the discarded error and source lose
				// nothing and nothing is read.
				var hintStateDir string
				if cmd.Flags().Changed("state-dir") {
					hintStateDir, _, _ = org.ResolveOrgStateDir(*stateDir, true)
				}
				return errors.New("org: merge: --branch is required: name the feature branch whose PR to merge (" +
					orgVerbHint("merge-order", hintStateDir) + " lists the ledger's branches)")
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
			hintStateDir := orgInboxHintStateDir(cmd, resolvedStateDir)
			repoRoot, cleanupSkipped := mergeCleanupRoot(resolvedStateDir, stateDirSource, branch, hintStateDir)
			res, err := rt.Merge(ctx, org.MergeParams{Branch: branch, AutoMerge: rt.Config.Director.AutoMerge, Requeue: requeue,
				WaitChecks: waitChecks, Progress: cmd.ErrOrStderr(), RepoRoot: repoRoot, HintStateDir: hintStateDir})
			if res.Recorded && res.Cleanup == nil && cleanupSkipped != "" {
				res.Cleanup = &org.CleanupResult{Branch: res.Branch, Head: res.Head, Refused: cleanupSkipped}
			}
			return printMergeResult(cmd.OutOrStdout(), res, err, hintStateDir)
		},
	}

	cmd.Flags().StringVar(&branch, "branch", "", "the feature branch whose PR to merge, as ralph org merge-order lists it (required)")
	cmd.Flags().DurationVar(&waitChecks, "wait-checks", 0, "how long to wait, with regate.lock released, for checks that are "+
		"pending or not reported yet and for the branch's org to be disbanded before refusing, and, on a branch with a merge "+
		"queue, for the queue to merge the PR before exiting 7, such as 60m (0 waits not at all; at most 6h)")
	cmd.Flags().BoolVar(&requeue, "requeue", false, "on a branch with a merge queue, put the PR in the queue again though GitHub "+
		"removed it from the queue after the branch's latest gate record (or less than 5 minutes before it), which merge refuses "+
		"without it; use it on purpose, such as after a flaky check, never by default")

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
// is the reason the CLEANUP: line gives instead (orgMainWorktreeMissing),
// which names the ralph org cleanup command of branch with hintStateDir,
// orgInboxHintStateDir's value (cleanupRetryCommand).
func mergeCleanupRoot(resolvedStateDir, stateDirSource, branch, hintStateDir string) (root, skipped string) {
	root, err := org.FeatureRepoRoot(resolvedStateDir, stateDirSource)
	if err != nil {
		return "", org.PrintableInboxText(fmt.Sprintf("the main worktree, where scripts/ralph-worktree.sh removes the feature "+
			"worktree, was not found (%s), so the feature worktree of the branch %s was not cleaned up: run %s from the main "+
			"worktree", orgMainWorktreeMissing, branch, cleanupRetryCommand(branch, hintStateDir)), false)
	}
	return root, ""
}

// ErrMergeQueued is wrapped by the error of ralph org merge when Merge ended
// with org.MergeResult.Queued: on a default branch with a merge queue, the PR
// is in the queue and not merged yet, merge left it there, and nothing was
// recorded, because --wait-checks left no time once the PR was in the queue
// (0 included) or the time was used up with the PR still there. ExitCode maps
// it to 7, so a caller can tell a PR to wait for from a refusal or a failure
// (plan docs/plans/active/2026-11-20-org-merge-queue-verb.md, D6): run ralph
// org merge again to keep waiting, also once origin's default branch moved
// (FR-12 of spec docs/specs/2026-11-19-org-merge-queue.md), or ralph org
// cleanup once GitHub merged the PR.
var ErrMergeQueued = errors.New("the PR is in the merge queue, not merged yet")

// mergeQueuedError is the error printMergeResult returns for a queued
// result: its text is the result's Queued as Merge wrote it (escaped, with
// the next step), and it unwraps to ErrMergeQueued.
type mergeQueuedError struct{ queued string }

func (e mergeQueuedError) Error() string { return e.queued }

func (e mergeQueuedError) Unwrap() error { return ErrMergeQueued }

// printMergeResult prints res, what Merge returned with err, and returns the
// command's error.
//
// When gh pr view read the PR back as MERGED at the gate's head into the
// default branch (res.Merged; a merge of another head, or into another base
// someone may have retargeted the PR to after the checks, is res.Failed,
// with no lines), or on the merge queue path a read of the wait for the queue
// found it so, stdout gets these lines first, also when err is set
// (pr_merged could not be recorded, or an interrupt came once gh pr merge had
// started or once that read found the PR merged):
//
//	MERGED: <PR url> (PR #<n> of <branch>)
//	METHOD: merge|squash|rebase (on the merge queue path, also queue)
//	HEAD: <the head the gate passed>
//	MERGE COMMIT: <its SHA, or unknown>
//	AUTO MERGE: true|false
//	PROTECTION: <what keeps the merge up to date>
//	GATE: accepted (<failed reason>) by <by> at <ts>: <reason text> (only with res.Accepted)
//	NOTE: <res.QueueNote> (only with it)
//	NOTE: <res.Note> (only with it)
//	CLEANUP: done|skipped (<reason>) (only with res.Cleanup)
//	CLEANUP MANUAL: <the command by hand> (only after a skipped one with it)
//
// res.QueueNote, set only on the merge queue path, says that the PR was in
// the queue already or that --requeue added it again. res.Note says how gh
// pr merge failed though the PR merged, or on the merge queue path that the
// ledger held the merge's pr_merged already, so merge did not record it
// again. The CLEANUP: lines say how the cleanup of the branch's feature
// worktree went once pr_merged was recorded (printMergeCleanup). The URL,
// branch, head, merge commit, protection, the acceptance's values,
// res.QueueNote, and the cleanup's reason and command go through
// PrintableInboxText, as merge-order's values from gh do; res.Note, escaped
// by Merge, keeps the line breaks of gh's tail and is printed as it is.
//
// When, on the merge queue path, merge left the PR in the queue, not merged
// and nothing recorded (res.Queued, with err nil), stdout gets these lines,
// escaped the same way (printMergeQueuedLines):
//
//	QUEUED: <PR url> (PR #<n> of <branch>)
//	METHOD: merge|squash|rebase|queue
//	HEAD: <the head the gate passed>
//	POSITION: <the PR's position in the queue> (only when GitHub gave its entry)
//	ENQUEUED AT: <when it entered the queue, UTC> (only when GitHub gave its entry)
//	PROTECTION: <the queue and the checks it requires>
//	GATE: accepted (<failed reason>) by <by> at <ts>: <reason text> (only with res.Accepted)
//	NOTE: <res.QueueNote> (only with it)
//
// The error is err when set; else res.Failed or res.Refused, with nothing on
// stdout (exit 1); else, for res.Queued, an error whose text is res.Queued
// and that wraps ErrMergeQueued (exit 7); else nil once a recorded merge was
// printed (exit 0), whatever the cleanup did. A result with none of these,
// which Merge does not return, is an error too, so that exit 0 always means a
// recorded merge. That error names ralph org merge-order with hintStateDir,
// orgInboxHintStateDir's value (orgVerbHint). An interrupt with res.Queued
// set returns err with no QUEUED: lines (exit 1): whether the PR is still in
// the queue is what err says.
func printMergeResult(w io.Writer, res org.MergeResult, err error, hintStateDir string) error {
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
	case res.Queued != "":
		printMergeQueuedLines(w, res)
		return mergeQueuedError{queued: res.Queued}
	case !res.Merged || !res.Recorded:
		return fmt.Errorf("org: merge: ralph org merge ended with no outcome for the branch %s (merged=%t, recorded=%t), so whether "+
			"its PR was merged is unknown: run %s to see the branch", org.PrintableInboxText(res.Branch, false), res.Merged,
			res.Recorded, orgVerbHint("merge-order", hintStateDir))
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
	printMergeGate(w, res.Accepted)
	if res.QueueNote != "" {
		_, _ = fmt.Fprintf(w, "NOTE: %s\n", org.PrintableInboxText(res.QueueNote, false))
	}
	if res.Note != "" {
		_, _ = fmt.Fprintf(w, "NOTE: %s\n", res.Note)
	}
	printMergeCleanup(w, res)
}

// printMergeQueuedLines prints printMergeResult's lines for res (Queued set):
// POSITION: and ENQUEUED AT: only when GitHub gave the PR's entry in the
// queue (res.EnqueuedAt set, as Merge sets both from it), and NOTE: only with
// res.QueueNote.
func printMergeQueuedLines(w io.Writer, res org.MergeResult) {
	_, _ = fmt.Fprintf(w, "QUEUED: %s (PR #%d of %s)\n", org.PrintableInboxText(res.PRURL, false), res.PRNumber,
		org.PrintableInboxText(res.Branch, false))
	_, _ = fmt.Fprintf(w, "METHOD: %s\n", res.Method)
	_, _ = fmt.Fprintf(w, "HEAD: %s\n", org.PrintableInboxText(res.Head, false))
	if res.EnqueuedAt != "" {
		_, _ = fmt.Fprintf(w, "POSITION: %d\n", res.QueuePosition)
		_, _ = fmt.Fprintf(w, "ENQUEUED AT: %s\n", org.PrintableInboxText(res.EnqueuedAt, false))
	}
	_, _ = fmt.Fprintf(w, "PROTECTION: %s\n", org.PrintableInboxText(res.Protection, false))
	printMergeGate(w, res.Accepted)
	if res.QueueNote != "" {
		_, _ = fmt.Fprintf(w, "NOTE: %s\n", org.PrintableInboxText(res.QueueNote, false))
	}
}

// printMergeGate prints the GATE: line of a, the acceptance of the gate
// failure an accepted branch's merge rests on; nothing when a is nil.
func printMergeGate(w io.Writer, a *org.GateAcceptance) {
	if a == nil {
		return
	}
	_, _ = fmt.Fprintf(w, "GATE: accepted (%s) by %s at %s: %s\n", org.PrintableInboxText(a.FailedReason, false),
		org.PrintableInboxText(a.By, false), org.PrintableInboxText(a.TS, false), org.PrintableInboxText(a.Note, false))
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
