package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// newOrgMergeOrderCmd wires `ralph org merge-order` (plan
// docs/plans/active/2026-10-14-org-merge-order.md, the CLI half of AC1, AC2,
// AC5 to AC9 and AC11): (*org.Org).MergeOrder reads the ledger and the merge
// guarantee `ralph doctor` cached in it (plan
// docs/plans/active/2026-10-14-org-merge-guarantee.md, AC7), runs the
// ls-remote, gh, and `git remote get-url origin` in the current directory
// (Dir left empty, Exec nil, so the real commands) and classifies and orders
// the branches; this command prints the result as text (printMergeOrder) or,
// with --json, as it is. It exits 0 whatever the classes, and 1 on
// MergeOrder's error, with nothing on stdout then. The list holds every
// org's branches, so --org-id is refused before anything is read, as the
// inbox verbs refuse it (newOrgInboxRuntime).
func newOrgMergeOrderCmd(stateDir, configPath *string) *cobra.Command {
	var all, jsonOut bool

	cmd := &cobra.Command{
		Use:   "merge-order",
		Short: "List the feature branches in the order their PRs may be merged now",
		Long: "ralph org merge-order lists the feature branches in the ledger in the\n" +
			"order their PRs may be merged now. A branch is ready when its latest\n" +
			"gate record (`ralph org gate`, or `ralph org regate`) is a gate_passed,\n" +
			"or a gate_ci_passed that regate records from the checks of the\n" +
			"branch's PR under [org.gate].regate_checks = \"ci\" (via=regate_ci),\n" +
			"whose head is the branch's tip on origin and whose base is the tip of\n" +
			"origin's default branch now. The gate records are the only evidence it\n" +
			"reads: a reviewer's or a leader's word does not count.\n" +
			"\n" +
			"The branches are those of every feature binding `ralph org start\n" +
			"--plan` recorded, disbanded orgs included, and those a gate record\n" +
			"names; each shows the org and feature of its latest binding.\n" +
			"merge-order reads the ledger's bindings and gate records, then runs one\n" +
			"git ls-remote --symref origin HEAD 'refs/heads/*' in the current\n" +
			"directory's repository: origin's default branch is the branch origin's\n" +
			"HEAD names, and the output gives its tip and each branch's tip. When gh\n" +
			"is on PATH, it runs gh pr list --head <branch> --state all --limit 1000\n" +
			"for each branch, in the same directory, and takes the highest-numbered\n" +
			"PR that is not from a fork; when all 1000 are from forks, it takes none\n" +
			"and a note says so. gh's PRs are used only when their repository is\n" +
			"origin's (git remote get-url origin): when gh reads another one, such\n" +
			"as upstream in a fork's checkout, no PR is used and a note says to run\n" +
			"gh repo set-default. A repository renamed on GitHub while origin keeps\n" +
			"the old URL looks the same (git reaches it through GitHub's redirect,\n" +
			"gh's PR urls carry the new name), so the note also says to point\n" +
			"origin at the new name with git remote set-url origin if it was\n" +
			"renamed. The check has three gaps: when origin's URL names\n" +
			"no owner/repo (a local path, file://, a scheme such as git+ssh://),\n" +
			"gh's PRs are used as they are; the host is not compared; and the check\n" +
			"runs only on a PR gh returns, so no note appears when the repository\n" +
			"gh reads has no PR for the branch.\n" +
			"\n" +
			"A branch gets the first of these classes that applies:\n" +
			"  merged, closed  its PR is merged, or closed without a merge (listed\n" +
			"                  only with --all)\n" +
			"  unreadable      its latest gate record cannot be read, or a corrupt\n" +
			"                  manifest line follows it and may be a newer one\n" +
			"  no_gate         it has no gate record\n" +
			"  failed          its latest gate record is a gate_failed, or a\n" +
			"                  gate_ci_failed (reason ci_failed); the reason is\n" +
			"                  shown\n" +
			"  no_remote       origin has no such branch: not pushed, or deleted\n" +
			"                  after a merge (told apart only with gh)\n" +
			"  regate          origin's tip of the branch is not the head the gate\n" +
			"                  checked: it was pushed after the gate, or the gate's\n" +
			"                  head was not pushed (a regate whose push failed)\n" +
			"  wrong_base      its PR merges into another branch than origin's\n" +
			"                  default one\n" +
			"  rebase          origin's default branch moved since the gate, or is\n" +
			"                  another branch than the one the gate checked\n" +
			"  ready           none of the above\n" +
			"\n" +
			"For a branch whose feature org is not running, the Details name ralph\n" +
			"org regate --branch <b> instead of ralph org gate; for one whose org\n" +
			"now runs another feature, they name it for once that org is disbanded.\n" +
			"\n" +
			"The list is in this order: ready, rebase, wrong_base, regate,\n" +
			"unreadable, failed, no_remote, no_gate, and with --all merged and\n" +
			"closed. Within a class, the oldest gate record comes first, and a\n" +
			"branch with no gate record goes by its binding. Merging one ready\n" +
			"branch moves the default branch, so the other ready ones become\n" +
			"rebase. The text starts with origin's default branch and its tip, then\n" +
			"the merge guarantee line, and ends with notes; --json prints the same\n" +
			"list, in the same order, as JSON, with the guarantee as\n" +
			"merge_guarantee.\n" +
			"\n" +
			"Each --json entry also has pr_lookup, which the text does not show:\n" +
			"what gh's PR lookup gave for the branch. It is found (its PR is used),\n" +
			"none (gh returned no PR, or fewer than 1000, all from forks), failed\n" +
			"(gh failed for the branch, and a note says why), limit_forks (gh\n" +
			"returned 1000 PRs, all from forks), or skipped for every branch: gh\n" +
			"is not on PATH, or its PRs were left out because gh read another\n" +
			"repository than origin's or the check could not be made.\n" +
			"\n" +
			"Each --json entry with a binding also has bound_to in it, which the\n" +
			"text does not show either: the branch of the org's latest complete\n" +
			"binding when that is another branch (the org_id was started again for\n" +
			"another feature later), whether the org runs now or not, and no\n" +
			"bound_to when the latest binding is this one. ralph org cleanup leaves\n" +
			"such a branch out unless --branch names it.\n" +
			"\n" +
			"The merge guarantee line says what makes sure, when a PR merges into\n" +
			"origin's default branch, that its required checks passed on a head\n" +
			"with the latest default branch, as ralph doctor last found it and\n" +
			"cached it in the ledger's state directory (merge-guarantee.json):\n" +
			"github_protection (a ruleset or the branch protection requires\n" +
			"up-to-date branches, or a ruleset has a merge queue, with a required\n" +
			"check), pr_checks_only (only the gate and the CI that ran when the PR\n" +
			"was made), or unknown with the reason: no ledger or no cache, a cache\n" +
			"that cannot be read or is more than 24 hours old, an origin whose host\n" +
			"or owner/repo is not the one doctor checked or cannot be read here, or\n" +
			"a default branch other than the one the ls-remote names. merge-order\n" +
			"runs no gh for it, and runs git remote get-url origin (local) for it\n" +
			"only when the cache is fresh. With no branch in the ledger, no\n" +
			"ls-remote runs and the guarantee line comes first.\n" +
			"\n" +
			"merge-order fetches, rebases, gates and merges nothing, and writes\n" +
			"nothing, to the ledger or anywhere else. It takes no lock and needs no\n" +
			"running watch. The order is as of the ls-remote and is not a\n" +
			"guarantee: the default branch can move before a merge. The list holds\n" +
			"every org's branches, so --org-id is refused.\n" +
			"\n" +
			"It exits 0 whatever the classes, also when gh is missing or fails: the\n" +
			"PR column is then empty and a note says what is not told apart. It\n" +
			"exits 1, and classifies nothing, when the manifest cannot be read, or\n" +
			"when the ls-remote fails, times out (30 s), or names no default branch.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Flags().Changed("org-id") {
				return errors.New("org: merge-order: --org-id does not apply: merge-order lists the feature branches of every org in the ledger")
			}
			resolvedStateDir, _, err := resolveOrgLedger(cmd, *stateDir, orgLedgerReadOnly)
			if err != nil {
				return err
			}
			rt, err := newOrgRuntimeAt(resolvedStateDir, *configPath)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			// Dir is left empty (the current directory) and Exec nil (the
			// real git and gh).
			res, err := rt.MergeOrder(ctx, org.MergeOrderParams{All: all})
			if err != nil {
				return err
			}
			if jsonOut {
				return writeIndentedJSON(cmd.OutOrStdout(), res)
			}
			printMergeOrder(cmd.OutOrStdout(), res)
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "also list the branches whose PR is merged or closed (known only with gh)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "machine-readable JSON output, in the same order as the text")

	return cmd
}

// mergeOrderSHALen is how many characters of a SHA the text shows, as the
// org layer's details do.
const mergeOrderSHALen = 12

// printMergeOrder prints res as `ralph org merge-order`'s text:
//
//	origin/<default branch> <its tip, 12 characters> (listed <ListedAt>)
//	merge guarantee: <res.MergeGuarantee.Text()>
//	RANK	CLASS	BRANCH	PR	GATE_HEAD	GATE_BASE	ORG	DETAIL
//	<one tab-separated row per entry>
//	note: <one line per note>
//
// The first line is left out when there is no default branch (no target
// branch, so no ls-remote ran), so the merge guarantee line, always printed,
// comes first then. The table is left out when there is no entry; with a
// default branch and no entry, the line `no branch to list` takes its place.
// A row's PR is `#<number> <url>`, its gate head and base 12 characters, its
// ORG `<org_id>/<feature>` of the latest binding, "-" for each one the entry
// lacks. Every value from the ledger, the guarantee's cache, git or gh goes
// through inboxField, so a crafted branch name, URL or detail cannot break a
// line or a column.
func printMergeOrder(w io.Writer, res org.MergeOrderResult) {
	if res.DefaultBranch != "" {
		_, _ = fmt.Fprintf(w, "origin/%s %s (listed %s)\n",
			inboxField(res.DefaultBranch), inboxField(mergeOrderShortSHA(res.DefaultHead)), inboxField(res.ListedAt))
	}
	_, _ = fmt.Fprintln(w, mergeGuaranteeLine(res.MergeGuarantee))
	if res.DefaultBranch != "" && len(res.Entries) == 0 {
		_, _ = fmt.Fprintln(w, "no branch to list")
	}
	if len(res.Entries) > 0 {
		_, _ = fmt.Fprintln(w, "RANK\tCLASS\tBRANCH\tPR\tGATE_HEAD\tGATE_BASE\tORG\tDETAIL")
	}
	for _, e := range res.Entries {
		pr, gateHead, gateBase, orgName := "-", "-", "-", "-"
		if e.PR != nil {
			pr = "#" + strconv.Itoa(e.PR.Number) + " " + inboxField(e.PR.URL)
		}
		if e.Gate != nil {
			gateHead, gateBase = inboxField(mergeOrderShortSHA(e.Gate.Head)), inboxField(mergeOrderShortSHA(e.Gate.Base))
		}
		if e.Binding != nil {
			orgName = inboxField(e.Binding.OrgID) + "/" + inboxField(e.Binding.Feature)
		}
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			e.Rank, inboxField(e.Class), inboxField(e.Branch), pr, gateHead, gateBase, orgName, inboxField(e.Detail))
	}
	for _, note := range res.Notes {
		_, _ = fmt.Fprintf(w, "note: %s\n", inboxField(note))
	}
}

// mergeOrderShortSHA is sha's first mergeOrderSHALen characters, cut at a
// character boundary.
func mergeOrderShortSHA(sha string) string {
	if r := []rune(sha); len(r) > mergeOrderSHALen {
		return string(r[:mergeOrderSHALen])
	}
	return sha
}
