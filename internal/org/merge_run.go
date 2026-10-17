package org

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// The merge of `ralph org merge` (Merge), S3b of plan
// docs/plans/active/2026-10-17-org-auto-merge.md: with regate.lock held and
// every check of checkMerge (merge.go) passed, it runs gh pr merge once, with
// the merge method gh repo view reported and --match-head-commit at the head
// the gate passed, never with --auto, --admin, or --delete-branch (AC6); reads
// the PR back whatever gh pr merge returned (D6, AC7, AC8); removes an
// auto-merge request GitHub held for the PR instead of merging it; and
// records the org-level pr_merged when the PR is MERGED (D4). Its gh calls run
// through MergeOrderExec with GH_PROMPT_DISABLED=1, pinned to origin's
// repository with -R, as checkMerge's are.

const (
	// defaultMergeTimeout bounds gh pr merge when MergeParams.MergeTimeout is
	// zero or less.
	defaultMergeTimeout = 60 * time.Second
	// mergeReadBackFields are the fields Merge reads with gh pr view once gh
	// pr merge returned (D6).
	mergeReadBackFields = "state,mergeCommit,autoMergeRequest"
	// mergeGHTailLines is how many of the last lines of gh pr merge's error a
	// Failed or a Note quotes (mergeGHSaid).
	mergeGHTailLines = 5
)

// mergeTestHooks are the points where a test stops Merge. Each is nil
// outside tests.
var mergeTestHooks struct {
	// afterChecks runs right after checkMerge passed, regate.lock held,
	// before Merge looks at ctx a last time and runs gh pr merge.
	afterChecks func()
}

// MergeResult is Merge's outcome. With an error once gh pr merge has started
// (an interruption, a pr_merged that could not be appended), Merge returns it
// too, as far as it got.
type MergeResult struct {
	// Refused is set when Merge refused before gh pr merge ran: no branch
	// named, another ralph org regate or merge holding regate.lock, or a
	// check of checkMerge that failed. It says why and one next step. Nothing
	// was merged or recorded, and every other field is empty.
	Refused string
	// Failed is set when gh pr merge ran but the PR is not MERGED as gh pr
	// view read it back, or that read failed (D6): what happened, whether an
	// auto-merge request GitHub held was removed, and one next step, then the
	// end of gh pr merge's error when it failed (mergeGHSaid). Nothing was
	// recorded. The fields from Branch on are set; MergeCommit is empty.
	Failed string
	// Merged reports that gh pr view read the PR back as MERGED after gh pr
	// merge, whatever gh pr merge returned.
	Merged bool
	// Recorded reports that pr_merged was appended (D4). With Merged it is
	// false only when the append failed, and Merge's error says so.
	Recorded bool
	// Note is set with Merged when gh pr merge failed or timed out: that gh
	// pr view read the PR back as MERGED anyway, then the end of gh pr
	// merge's error (mergeGHSaid).
	Note string
	// Branch is the branch merged, set once the checks passed, as are the
	// fields below.
	Branch string
	// OrgID is the org the merge is recorded under: the org of the branch's
	// latest binding, else of its latest gate record (D4).
	OrgID string
	// PRNumber and PRURL are the branch's PR, as gh pr view read it in the
	// checks.
	PRNumber int
	PRURL    string
	// Method is the merge method gh pr merge was run with: merge, squash, or
	// rebase (D2).
	Method string
	// Head is the head the gate passed, which --match-head-commit named.
	Head string
	// MergeCommit is the PR's merge commit as gh pr view read it back, set
	// with Merged; empty when gh gave none that is a commit SHA, and pr_merged
	// records merge_commit=unknown then.
	MergeCommit string
	// AutoMerge is MergeParams.AutoMerge, [org.director].auto_merge, which
	// Merge only reports and records (D1).
	AutoMerge bool
	// Protection is CheckMergeEnforcement's Detail: why GitHub keeps the
	// merge up to date (D11).
	Protection string
}

// Merge runs `ralph org merge --branch <p.Branch>`. It refuses
// (MergeResult.Refused, nothing merged or recorded) when p.Branch is empty,
// before any lock; when another ralph org regate or merge of the ledger holds
// regate.lock after regateLockWait (D3); and when a check of checkMerge fails.
// Once it holds the lock it empties the marker an earlier regate left there
// and writes none: a merge changes no worktree, so a spawn of the branch's org
// goes on while it runs (regateSpawnErr). It then runs gh pr merge once
// (mergeRun.merge) and, whatever that returned, reads the PR back once
// (mergeRun.readBack, D6):
//
//   - MERGED: it appends pr_merged (recordMerge, D4) and sets Merged and
//     Recorded; Note says how gh pr merge failed when it did;
//   - not MERGED with an auto-merge request, which GitHub holds instead of
//     merging a PR that cannot be merged yet: it removes the request with gh
//     pr merge --disable-auto, and Failed says whether that worked;
//   - not MERGED otherwise, or a read that failed: Failed says what happened.
//
// Failed records nothing. The read, the removal, and the record run with
// ctx's cancellation dropped, under the gh timeout and the manifest lock's
// wait, so an interruption once gh pr merge has started still gets each of
// them once; Merge then returns an error that says it was interrupted and how
// the merge ended, with MergeResult as far as it got. Before gh pr merge, an
// interruption is an error with nothing merged. The other errors are a
// manifest that cannot be read, a lock that cannot be taken or written, and a
// pr_merged that cannot be appended after the merge (Merged set, Recorded
// not). regate.lock is released when Merge returns.
func (o *Org) Merge(ctx context.Context, p MergeParams) (MergeResult, error) {
	if p.Branch == "" {
		_, refused, err := o.checkMerge(ctx, p)
		return MergeResult{Refused: refused}, err
	}
	stateDir := filepath.Dir(o.Manifest.Path())
	lock, err := acquireHeldFileLock(stateDir, regateLockFile, "regate", regateLockWait)
	switch {
	case errors.Is(err, errFileLockHeld):
		return MergeResult{Refused: PrintableInboxText(fmt.Sprintf(mergePrefix+"another ralph org regate or merge runs on this "+
			"ledger (%s is held): one runs at a time, so wait for it to end and run ralph org merge --branch %s again",
			regateLockPathIn(stateDir), p.Branch), false)}, nil
	case err != nil:
		return MergeResult{}, fmt.Errorf(mergePrefix+"nothing was merged, since %s could not be taken: %w", regateLockPathIn(stateDir), err)
	}
	defer func() { _ = lock.release() }()
	// D3: the marker an earlier regate left goes at once, and Merge writes
	// none, so no spawn of that org is refused while the merge runs.
	if err := writeRegateMarker(lock.f, nil); err != nil {
		return MergeResult{}, fmt.Errorf(mergePrefix+"nothing was merged: %w", err)
	}

	r, refused, err := o.checkMerge(ctx, p)
	switch {
	case err != nil:
		return MergeResult{}, err
	case refused != "":
		return MergeResult{Refused: refused}, nil
	}
	if hook := mergeTestHooks.afterChecks; hook != nil {
		hook()
	}
	if ctx.Err() != nil {
		return MergeResult{}, fmt.Errorf(mergePrefix+"interrupted (%v) after the checks passed, before gh pr merge ran: nothing "+
			"was merged", context.Cause(ctx))
	}
	run := mergeRun{m: newMergeCheck(p), r: r}
	res, err := o.mergeAndRecord(ctx, run, MergeResult{Branch: r.branch, OrgID: r.orgID, PRNumber: r.prNumber, PRURL: r.prURL,
		Method: r.method, Head: r.head, AutoMerge: p.AutoMerge, Protection: r.protection})
	if ctx.Err() != nil {
		return res, mergeInterruptedAfterStart(ctx, res, err)
	}
	return res, err
}

// mergeRun is one Merge once its checks passed: the settings of its checks
// (newMergeCheck) and what checkMerge found.
type mergeRun struct {
	m *mergeCheck
	r mergeReady
}

// pr is the PR's number as gh takes it.
func (run mergeRun) pr() string { return strconv.Itoa(run.r.prNumber) }

// mergeCmd names gh pr merge of the PR in the texts.
func (run mergeRun) mergeCmd() string {
	return "gh pr merge " + run.pr() + " -R " + run.r.repoArg
}

// viewCmd names gh pr view of the PR in the texts.
func (run mergeRun) viewCmd() string {
	return "gh pr view " + run.pr() + " -R " + run.r.repoArg
}

// disableCmd is the command that removes the PR's auto-merge request.
func (run mergeRun) disableCmd() string {
	return "gh pr merge " + run.pr() + " --disable-auto -R " + run.r.repoArg
}

// mergeAttempt is how gh pr merge ended: err is its error, nil on exit 0, and
// timedOut reports that MergeParams.MergeTimeout stopped it, not ctx.
type mergeAttempt struct {
	err      error
	timedOut bool
}

// merge is the gh pr merge of Merge (AC6): `gh pr merge <n> <method flag>
// --match-head-commit <the gate's head> -R <repo>`, once, in the repository
// directory with GH_PROMPT_DISABLED=1, bounded by MergeParams.MergeTimeout.
// --match-head-commit makes GitHub merge only while the PR's head is the
// gate's. gh's stdout is not read: the read-back says what happened.
func (run mergeRun) merge(ctx context.Context) mergeAttempt {
	mergeCtx, cancel := context.WithTimeout(ctx, run.m.mergeTimeout)
	defer cancel()
	_, err := run.m.run(mergeCtx, run.m.dir, []string{"GH_PROMPT_DISABLED=1"}, "gh", "pr", "merge", run.pr(), run.r.methodFlag,
		"--match-head-commit", run.r.head, "-R", run.r.repoArg)
	return mergeAttempt{err: err, timedOut: err != nil && mergeCtx.Err() != nil && ctx.Err() == nil}
}

// how says how gh pr merge ended (a), for the texts.
func (run mergeRun) how(a mergeAttempt) string {
	switch {
	case a.timedOut:
		return "timed out after " + run.m.mergeTimeout.String()
	case a.err != nil:
		return "failed"
	}
	return "exited 0"
}

// mergeReadBack is gh pr view's answer with mergeReadBackFields.
// AutoMergeRequest is the JSON as gh printed it, as in ghPRView: null when the
// PR has no auto-merge request, nil when the field is missing. MergeCommit is
// nil when gh printed null.
type mergeReadBack struct {
	State       string `json:"state"`
	MergeCommit *struct {
		OID string `json:"oid"`
	} `json:"mergeCommit"`
	AutoMergeRequest json.RawMessage `json:"autoMergeRequest"`
}

// mergeCommit is the merge commit gh read back, "" when it gave none that is
// a commit SHA (gitObjectPattern): a value with a space in it would split
// pr_merged's Details.
func (b mergeReadBack) mergeCommit() string {
	if b.MergeCommit == nil || !gitObjectPattern.MatchString(b.MergeCommit.OID) {
		return ""
	}
	return b.MergeCommit.OID
}

// readBack is the read of the PR once gh pr merge returned, whatever it
// returned (D6): gh pr view <n> -R <repo> --json mergeReadBackFields, once,
// with ctx's cancellation dropped and the gh timeout. unknown is set when the
// PR's state stays unknown, saying why: gh failed, or printed no JSON with a
// state and, for a PR that is not MERGED, its autoMergeRequest, without which
// Merge cannot tell whether to remove a request.
func (run mergeRun) readBack(ctx context.Context) (back mergeReadBack, unknown string) {
	out, _, err := run.m.gh(context.WithoutCancel(ctx), "pr", "view", run.pr(), "-R", run.r.repoArg, "--json", mergeReadBackFields)
	if err != nil {
		return mergeReadBack{}, "failed (" + mergeGuaranteeErrText(err) + ")"
	}
	if err := json.Unmarshal([]byte(out), &back); err != nil || back.State == "" ||
		back.State != "MERGED" && back.AutoMergeRequest == nil {
		return mergeReadBack{}, "printed output that is not the PR's JSON with " + mergeReadBackFields
	}
	return back, ""
}

// mergeAndRecord is what Merge does once its checks passed: gh pr merge, the
// read-back, and what follows from the PR's state, as Merge lists them. res
// holds what the checks found; the outcome is added to it. The error is a
// pr_merged that could not be appended (recordMerge).
func (o *Org) mergeAndRecord(ctx context.Context, run mergeRun, res MergeResult) (MergeResult, error) {
	a := run.merge(ctx)
	back, unknown := run.readBack(ctx)
	switch {
	case unknown != "":
		res.Failed = run.failed(fmt.Sprintf("%s %s after %s %s, so whether PR #%d was merged is unknown and nothing was "+
			"recorded: check it with %s --json state,mergeCommit (ralph org merge-order lists the branch as merged once it is)",
			run.viewCmd(), unknown, run.mergeCmd(), run.how(a), run.r.prNumber, run.viewCmd()), a)
	case back.State == "MERGED":
		res.Merged, res.MergeCommit = true, back.mergeCommit()
		if a.err != nil {
			res.Note = PrintableInboxText(fmt.Sprintf("%s %s, but %s read PR #%d back as MERGED", run.mergeCmd(), run.how(a),
				run.viewCmd(), run.r.prNumber), false) + mergeGHSaid(a.err)
		}
		if err := o.recordMerge(ctx, res); err != nil {
			return res, err
		}
		res.Recorded = true
	case string(bytes.TrimSpace(back.AutoMergeRequest)) != "null":
		res.Failed = run.removeAutoMerge(ctx, back.State, a)
	case a.err != nil:
		res.Failed = run.failed(fmt.Sprintf("%s %s and PR #%d is %s, not merged, so nothing was merged or recorded: run ralph org "+
			"merge-order again and follow the branch's Detail", run.mergeCmd(), run.how(a), run.r.prNumber, back.State), a)
	default:
		res.Failed = run.failed(fmt.Sprintf("%s exited 0 but PR #%d is %s, not merged, so nothing was recorded: check the PR with "+
			"%s and run ralph org merge-order again", run.mergeCmd(), run.r.prNumber, back.State, run.viewCmd()), a)
	}
	return res, nil
}

// removeAutoMerge handles a PR that is not MERGED and has an auto-merge
// request after gh pr merge (a): GitHub held the request instead of merging,
// as it does for a PR that cannot be merged yet (a check that still runs, a
// merge queue the checks did not find). It removes the request with gh pr
// merge --disable-auto, once, with ctx's cancellation dropped and the gh
// timeout, and returns the Failed that says whether that worked.
func (run mergeRun) removeAutoMerge(ctx context.Context, state string, a mergeAttempt) string {
	_, _, err := run.m.gh(context.WithoutCancel(ctx), "pr", "merge", run.pr(), "--disable-auto", "-R", run.r.repoArg)
	if err != nil {
		return run.failed(fmt.Sprintf("%s %s and did not merge PR #%d, which is %s: GitHub held an auto-merge request for it "+
			"instead, and %s failed (%s), so the request may remain and GitHub may still merge the PR apart from ralph org "+
			"merge; nothing was recorded: remove the request with %s", run.mergeCmd(), run.how(a), run.r.prNumber, state,
			run.disableCmd(), mergeGuaranteeErrText(err), run.disableCmd()), a)
	}
	return run.failed(fmt.Sprintf("%s %s and did not merge PR #%d, which is %s: GitHub held an auto-merge request for it "+
		"instead, which ralph org merge removed (%s), so nothing was merged or recorded: %s, which checks the PR and the "+
		"protection again", run.mergeCmd(), run.how(a), run.r.prNumber, state, run.disableCmd(), run.m.again()), a)
}

// failed is a Failed: mergePrefix and text, escaped (PrintableInboxText),
// then the end of gh pr merge's error when it failed (mergeGHSaid).
func (run mergeRun) failed(text string, a mergeAttempt) string {
	return PrintableInboxText(mergePrefix+text, false) + mergeGHSaid(a.err)
}

// mergeGHSaid is the end of err, gh pr merge's error, for a Failed or a Note:
// "" when err is nil or empty, else "; gh said:" and its last
// mergeGHTailLines lines that are not blank, each on a line of its own,
// escaped and cut by mergeGuaranteeErrText. realMergeOrderExec puts gh's
// stderr after the exit status in err, so these are gh's last lines.
func mergeGHSaid(err error) string {
	if err == nil {
		return ""
	}
	var lines []string
	for line := range strings.SplitSeq(err.Error(), "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	lines = lines[max(0, len(lines)-mergeGHTailLines):]
	for i, line := range lines {
		lines[i] = mergeGuaranteeErrText(errors.New(line))
	}
	return "; gh said:\n  " + strings.Join(lines, "\n  ")
}

// recordMerge appends res's pr_merged as an org-level event of res.OrgID
// under the manifest lock (D4), with ctx's cancellation dropped: the PR is
// merged on GitHub whatever ctx says. manifestLockTimeout still bounds the
// wait for the lock. The ledger is not read again, since nothing in it can
// undo the merge. The error says the PR was merged and pr_merged was not
// recorded.
func (o *Org) recordMerge(ctx context.Context, res MergeResult) error {
	commit := cmp.Or(res.MergeCommit, "unknown")
	err := withManifestLockContext(context.WithoutCancel(ctx), filepath.Dir(o.Manifest.Path()), func() error {
		return o.appendEvent(ManifestEvent{TS: o.now(), OrgID: res.OrgID, SeatID: "", Event: EventPRMerged, Details: fmt.Sprintf(
			"branch=%s pr=%d head=%s method=%s merge_commit=%s auto_merge=%t", res.Branch, res.PRNumber, res.Head, res.Method,
			commit, res.AutoMerge)})
	})
	if err != nil {
		return fmt.Errorf(mergePrefix+"PR #%d (%s) was merged (merge commit %s) but %s could not be recorded (%w): the merge "+
			"stands, and ralph org merge-order lists the branch as merged, so only the ledger misses it; check that %s can be "+
			"written", res.PRNumber, PrintableInboxText(res.PRURL, false), commit, EventPRMerged, err, o.Manifest.Path())
	}
	return nil
}

// mergeInterruptedAfterStart is Merge's error when ctx was done once gh pr
// merge had started: that Merge was interrupted, then how the merge ended,
// from err (a pr_merged that could not be appended), res.Failed, or the
// merge res records.
func mergeInterruptedAfterStart(ctx context.Context, res MergeResult, err error) error {
	outcome := fmt.Sprintf("PR #%d (%s) was merged (merge commit %s) and %s was recorded", res.PRNumber,
		PrintableInboxText(res.PRURL, false), cmp.Or(res.MergeCommit, "unknown"), EventPRMerged)
	switch {
	case err != nil:
		outcome = strings.TrimPrefix(err.Error(), mergePrefix)
	case res.Failed != "":
		outcome = strings.TrimPrefix(res.Failed, mergePrefix)
	}
	return fmt.Errorf(mergePrefix+"interrupted (%v) once gh pr merge had started, and then: %s", context.Cause(ctx), outcome)
}
