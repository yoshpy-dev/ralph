package org

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// The follow-up section of `ralph org merge-order` (issue #270; plan
// docs/plans/active/2026-10-30-org-followups.md, D2, D5, AC3, AC7 and AC8).
// With MergeOrderParams.Followups, MergeOrder reports every open follow-up of
// the ledger (followups.jsonl, followup.go, read with no lock, as the inbox's
// Read is) after its entries, with the state gh gives its after-branch. For
// each distinct after-branch it runs one gh pr list through ghBranchPRs, as
// lookupMergePR does, but asks for mergedAt too and keeps every PR that is
// not from a fork, each checked against origin's repository as
// lookupMergePRs checks the PR it uses. A branch name can be used again, so
// only a merge at or after the follow-up's added_at counts (D5). The section
// changes neither the entries, nor their lookups, nor MergeOrder's error; it
// sets Followups, GHAvailable when its lookup found gh, and a note in Notes
// when the follow-up record cannot be read.
// ralph status decides it from the ledger's records of merges instead,
// without gh (D2), so the two can differ (FollowupRecordedMerge says when);
// the help tells a reader to use merge-order's answer when one says ready
// and the other waiting, and status's, with merge-order's reason, when
// merge-order says unknown.

// The states of a MergeOrderFollowup (D5).
const (
	// FollowupMergeReady: a PR of the after-branch was merged at or after the
	// follow-up's added_at, so the human can be asked about its issue now.
	FollowupMergeReady = "ready"
	// FollowupMergeWaiting: no such PR, and the highest-numbered PR is open,
	// or the branch has no PR yet, or its only merged PRs were merged before
	// the follow-up was recorded.
	FollowupMergeWaiting = "waiting"
	// FollowupMergeClosed: no such PR, and the highest-numbered PR was closed
	// without a merge.
	FollowupMergeClosed = "closed"
	// FollowupMergeUnknown: gh's answer does not tell: gh is missing; gh pr
	// list failed or timed out for the branch; gh's PRs are not of origin's
	// repository or cannot be checked against it (checkFollowupPRs);
	// merge-order was interrupted before the branch was read; a merged PR's
	// mergedAt cannot be read; gh returned its limit of 1000 PRs with none
	// merged after the follow-up; gh gave a PR state other than OPEN, MERGED
	// and CLOSED; or the follow-up's added_at cannot be read
	// (FollowupAddedTime). Detail says which.
	FollowupMergeUnknown = "unknown"
)

// FollowupMergeSourceGH is the Source of every MergeOrderFollowup: its state
// comes from gh's PRs, not from the ledger's records of merges (D2).
const FollowupMergeSourceGH = "gh"

// mergeOrderFollowupGHFields are the PR fields a follow-up's lookup asks gh
// for: mergeOrderGHFields and the merge time.
const mergeOrderFollowupGHFields = mergeOrderGHFields + ",mergedAt"

// MergeOrderFollowup is one open follow-up as merge-order reports it. PR is
// the PR its state is read from, nil when there is none to name, and MergedAt
// that PR's merge time as gh gives it, "" unless the PR is merged: a waiting
// follow-up can carry one from before it was recorded.
type MergeOrderFollowup struct {
	ID          string                `json:"id"`
	Issue       int                   `json:"issue"`
	AfterBranch string                `json:"after_branch"`
	AddedAt     string                `json:"added_at"`
	State       string                `json:"state"`  // FollowupMerge*
	Source      string                `json:"source"` // FollowupMergeSourceGH
	PR          *MergeOrderFollowupPR `json:"pr"`
	MergedAt    string                `json:"merged_at"`
	// Detail says, in one sentence, why the follow-up has its state and what
	// to do next.
	Detail string `json:"detail"`
}

// MergeOrderFollowupPR names the PR a MergeOrderFollowup's state is read
// from.
type MergeOrderFollowupPR struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

// MergeOrderFollowups is merge-order's follow-ups, --json's followups, an
// object with the same outer keys as the followups of `ralph status --json`:
// the open follow-ups (Items, never nil), the record's damage counts as
// FollowupReadResult counts them, and Error, why the record could not be
// read, "" when it was read or does not exist. The items' keys differ: an
// item here has pr, and merged_at is the PR's mergedAt as gh gives it,
// while status's item has merged_event, and its merged_at is the ts of that
// ledger event (for a worktree_cleaned, when the cleanup ran). A reader
// tells a record that could not be read from one with no open follow-up by
// Error, without reading the notes. Path is the record's path, which the CLI
// names when it warns about the damage on stderr; it is not in the JSON.
type MergeOrderFollowups struct {
	Items         []MergeOrderFollowup `json:"items"`
	CorruptLines  int                  `json:"corrupt_lines"`
	IgnoredEvents int                  `json:"ignored_events"`
	Error         string               `json:"error"`
	Path          string               `json:"-"`
}

// mergeOrderFollowups fills res.Followups when p.Followups is set: it reads
// the record in the manifest's state dir, keeps its open follow-ups, and
// looks their after-branches up with run (lookupFollowupMerges). It sets
// res.GHAvailable when that lookup found gh, and never clears it, so the
// entries' lookup (lookupMergePRs) keeps a true it gave. A record that cannot
// be read leaves Items empty, with its error in Error and a note. Nothing
// here makes MergeOrder fail.
func (o *Org) mergeOrderFollowups(ctx context.Context, p MergeOrderParams, run MergeOrderExec, res *MergeOrderResult) {
	if !p.Followups {
		return
	}
	store := NewFollowupStore(filepath.Dir(o.Manifest.Path()))
	res.Followups.Path = store.Path()
	rec, err := store.Read()
	if err != nil {
		res.Followups.Error = err.Error()
		res.Notes = append(res.Notes, fmt.Sprintf("the follow-up record cannot be read (%v): no follow-up is shown", err))
		return
	}
	res.Followups.CorruptLines, res.Followups.IgnoredEvents = rec.CorruptLines, rec.IgnoredEvents
	var open []Followup
	for _, f := range rec.Items {
		if f.State == FollowupStateOpen {
			open = append(open, f)
		}
	}
	var ghFound bool
	res.Followups.Items, ghFound = lookupFollowupMerges(ctx, p, run, open)
	res.GHAvailable = res.GHAvailable || ghFound
}

// followupLookup is what gh's lookup gave for one after-branch: its PRs that
// are not from a fork, all checked against origin's repository, and
// limitFull when gh returned mergeOrderGHLimit PRs, so older ones may be past
// the limit; or why, the reason no answer of gh is used for the branch.
type followupLookup struct {
	prs       []followupPR
	limitFull bool
	why       string
}

// followupPR is one PR of an after-branch, with its merge time as gh gives
// it.
type followupPR struct {
	MergeOrderPR
	mergedAt string
}

func (pr *followupPR) merged() bool {
	return strings.ToUpper(pr.State) == "MERGED"
}

// lookupFollowupMerges runs one lookup per distinct after-branch of open, in
// the order the branches first appear (lookupFollowupPRs), and returns each
// follow-up of open, in order, with its state (followupMergeState). A gh that
// fails for a branch makes only that branch's follow-ups unknown. gh not
// found, or a PR that is not of origin's repository or cannot be checked
// against it (checkFollowupPRs, with a mergeOriginRepo of its own, so one
// more `git remote get-url origin` than lookupMergePRs runs), makes every
// follow-up unknown with the reason and runs gh no more, as lookupMergePRs
// then leaves every PR out. ctx done makes the branches not looked up yet
// unknown and runs nothing more: the entries are classified by then, so it
// is no error. ghFound reports that gh was found, as lookupMergePRs's
// available does: gh answered for a branch, with its PRs or a failure, and
// no call found it missing; false when no branch was looked up.
func lookupFollowupMerges(ctx context.Context, p MergeOrderParams, run MergeOrderExec, open []Followup) (_ []MergeOrderFollowup, ghFound bool) {
	timeout := p.GHTimeout
	if timeout <= 0 {
		timeout = defaultMergeOrderGHTimeout
	}
	var branches []string
	for _, f := range open {
		if !slices.Contains(branches, f.AfterBranch) {
			branches = append(branches, f.AfterBranch)
		}
	}
	interrupted := func() string {
		return fmt.Sprintf("interrupted (%v) while gh ran, so the after-branch's PRs were not read", context.Cause(ctx))
	}
	lookups := make(map[string]followupLookup)
	// every is why no answer of gh is used for any branch, rest why the
	// branches not looked up have none.
	var every, rest string
	var origin mergeOriginRepo
	repoArg := p.Origin.Arg()
lookup:
	for _, branch := range branches {
		prs, limitFull, err := lookupFollowupPRs(ctx, p.Dir, run, timeout, repoArg, branch)
		switch {
		case ctx.Err() != nil:
			rest = interrupted()
			break lookup
		case errors.Is(err, exec.ErrNotFound):
			every = "gh was not found, so no PR was read: ralph status tells it from the ledger's records of merges, without gh"
			ghFound = false
			break lookup
		}
		ghFound = true
		if err != nil {
			lookups[branch] = followupLookup{why: fmt.Sprintf("gh pr list --head %s failed (%s): its PR is unknown", branch, err)}
			continue
		}
		why, err := checkFollowupPRs(ctx, p, run, timeout, &origin, branch, prs)
		switch {
		case err != nil:
			rest = interrupted()
			break lookup
		case why != "":
			every = why
			break lookup
		}
		lookups[branch] = followupLookup{prs: prs, limitFull: limitFull}
	}
	out := make([]MergeOrderFollowup, 0, len(open))
	for _, f := range open {
		l, looked := lookups[f.AfterBranch]
		switch {
		case every != "":
			l = followupLookup{why: every}
		case !looked:
			l = followupLookup{why: rest}
		}
		out = append(out, followupMergeState(f, l))
	}
	return out, ghFound
}

// lookupFollowupPRs runs gh's PR list for branch as lookupMergePR does
// (ghBranchPRs, with -R repo when repo is not empty), but asks for
// mergeOrderFollowupGHFields, and returns every PR of its output that is not
// from a fork (isCrossRepository false). limitFull reports that gh returned
// mergeOrderGHLimit PRs. The error is ghBranchPRs's.
func lookupFollowupPRs(ctx context.Context, dir string, run MergeOrderExec, timeout time.Duration, repo, branch string) (prs []followupPR, limitFull bool, _ error) {
	all, err := ghBranchPRs(ctx, dir, run, timeout, repo, branch, mergeOrderFollowupGHFields)
	if err != nil {
		return nil, false, err
	}
	for _, pr := range all {
		if !pr.IsCrossRepository {
			prs = append(prs, followupPR{
				MergeOrderPR: MergeOrderPR{Number: pr.Number, State: pr.State, URL: pr.URL, HeadRefOid: pr.HeadRefOid, BaseRefName: pr.BaseRefName},
				mergedAt:     pr.MergedAt,
			})
		}
	}
	return prs, len(all) >= mergeOrderGHLimit, nil
}

// checkFollowupPRs checks each of prs, the PRs gh returned for branch,
// against origin's repository as lookupMergePRs checks the PR it uses: with
// p.Origin by MergeOrigin.foreignPRWhy, else by origin.check, which runs `git
// remote get-url origin` the first time it runs. It returns why the PRs may
// not be used, "" when all of them may. The error is for ctx done.
func checkFollowupPRs(ctx context.Context, p MergeOrderParams, run MergeOrderExec, timeout time.Duration, origin *mergeOriginRepo, branch string, prs []followupPR) (string, error) {
	for i := range prs {
		pr := &prs[i].MergeOrderPR
		if p.Origin.Arg() != "" {
			if why := p.Origin.foreignPRWhy(branch, pr); why != "" {
				return why, nil
			}
			continue
		}
		if stop, err := origin.check(ctx, p.Dir, run, timeout, branch, pr); err != nil || stop != "" {
			return stop, err
		}
	}
	return "", nil
}

// followupMergeState is f with the state l gives it (D5): unknown with l.why
// when there is one, or when f.AddedAt is not an RFC3339 time
// (FollowupAddedTime); else ready with the highest-numbered PR merged at or
// after f.AddedAt (FollowupReadyDetail); else unknown for a merged PR whose
// merge time cannot be read, or when gh returned its limit of PRs; else, by
// the highest-numbered PR, waiting with no PR, waiting for an open one,
// closed for one closed without a merge, waiting for one merged before f was
// recorded, and unknown for a state gh has no other of.
func followupMergeState(f Followup, l followupLookup) MergeOrderFollowup {
	out := MergeOrderFollowup{ID: f.ID, Issue: f.Issue, AfterBranch: f.AfterBranch, AddedAt: f.AddedAt, Source: FollowupMergeSourceGH}
	set := func(state string, pr *followupPR, detail string) MergeOrderFollowup {
		out.State, out.Detail = state, detail
		if pr != nil {
			out.PR = &MergeOrderFollowupPR{Number: pr.Number, URL: pr.URL}
			if pr.merged() {
				out.MergedAt = pr.mergedAt
			}
		}
		return out
	}
	if l.why != "" {
		return set(FollowupMergeUnknown, nil, l.why)
	}
	added, unreadableAdded := FollowupAddedTime(f)
	if unreadableAdded != "" {
		return set(FollowupMergeUnknown, nil, unreadableAdded)
	}
	top, after, before, unreadable := pickFollowupPRs(l.prs, added)
	switch {
	case after != nil:
		return set(FollowupMergeReady, after, FollowupReadyDetail(f, fmt.Sprintf("PR #%d was merged", after.Number), after.mergedAt))
	case unreadable != nil:
		return set(FollowupMergeUnknown, unreadable, fmt.Sprintf("PR #%d is merged, but gh gave its merge time as %q, which cannot "+
			"be compared with the follow-up's added_at %s", unreadable.Number, unreadable.mergedAt, f.AddedAt))
	case l.limitFull:
		return set(FollowupMergeUnknown, nil, fmt.Sprintf("gh returned %d PRs for %s, its limit, and none of them was merged after "+
			"the follow-up was recorded; the PR that was may be past them", mergeOrderGHLimit, f.AfterBranch))
	case top == nil:
		return set(FollowupMergeWaiting, nil, fmt.Sprintf("%s has no PR yet: the follow-up waits for one to be opened and merged",
			f.AfterBranch))
	}
	var older string
	if before != nil && before != top {
		older = fmt.Sprintf("; PR #%d was merged before the follow-up was recorded (at %s) and does not count", before.Number,
			before.mergedAt)
	}
	switch strings.ToUpper(top.State) {
	case "OPEN":
		return set(FollowupMergeWaiting, top, fmt.Sprintf("PR #%d is open: the follow-up waits for its merge%s", top.Number, older))
	case "CLOSED":
		return set(FollowupMergeClosed, top, fmt.Sprintf("PR #%d was closed without a merge, so the merge the follow-up waits for "+
			"may not come%s: reopen PR #%d or open a new PR of %s, or ask the human about issue #%d and run ralph org followup "+
			"done %s", top.Number, older, top.Number, f.AfterBranch, f.Issue, f.ID))
	case "MERGED":
		return set(FollowupMergeWaiting, top, fmt.Sprintf("PR #%d was merged before the follow-up was recorded (merged at %s, "+
			"recorded at %s), so it does not count: the follow-up waits for a new PR of %s to be merged; if none is coming, ask "+
			"the human about issue #%d now and run ralph org followup done %s", top.Number, top.mergedAt, f.AddedAt,
			f.AfterBranch, f.Issue, f.ID))
	}
	return set(FollowupMergeUnknown, top, fmt.Sprintf("gh gave PR #%d the state %q, which is none of OPEN, MERGED and CLOSED",
		top.Number, top.State))
}

// pickFollowupPRs returns, each nil when there is none, the highest-numbered
// of prs (top), and of its merged ones the highest-numbered merged at or after
// added (after), merged before it (before), and with a merge time that does
// not parse as RFC3339 (unreadable).
func pickFollowupPRs(prs []followupPR, added time.Time) (top, after, before, unreadable *followupPR) {
	for i := range prs {
		pr := &prs[i]
		top = higherFollowupPR(top, pr)
		if !pr.merged() {
			continue
		}
		at, err := time.Parse(time.RFC3339, pr.mergedAt)
		switch {
		case err != nil:
			unreadable = higherFollowupPR(unreadable, pr)
		case at.Before(added):
			before = higherFollowupPR(before, pr)
		default:
			after = higherFollowupPR(after, pr)
		}
	}
	return top, after, before, unreadable
}

// higherFollowupPR is the one of best and pr with the higher number, pr when
// best is nil.
func higherFollowupPR(best, pr *followupPR) *followupPR {
	if best == nil || pr.Number > best.Number {
		return pr
	}
	return best
}
