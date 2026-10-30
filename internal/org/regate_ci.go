package org

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// The ci regate of plan docs/plans/active/2026-10-29-regate-ci-checks.md
// ([org.gate].regate_checks = "ci", RegateParams.Checks): in place of the
// gate scripts, Regate (regate.go) reads the checks of the branch's PR on
// origin with gh and records the gate from them. This file holds the checks
// made before anything changes (regateCIPreCheck, D2 step 1) and the
// decision from the PR's head and checks to a gate_ci_passed, a
// gate_ci_failed, or no record (regateChecks, D4), which the gate-only path
// (D6) runs in place of regateGate; and the merge path's steps after the
// merge commit regateMerged makes (regateCIMerged, D2 steps 3 and 4): the push
// before any record, with the read-back of origin's tip after a push that
// failed (regateCIPush, D3), and then the same decision on the pushed merge
// commit. Both paths decide through regateChecksWithWait, which, with
// RegateParams.WaitChecks, waits for checks that are not done yet the way
// ralph org merge --wait-checks waits (merge_wait.go, D5): it releases
// regate.lock, only reads the PR's head and checks while they are pending,
// and then takes the lock again, once, and checks again what the record
// relies on (regateCIRecheck) before it decides again. That wait is the only
// place where a Regate lets go of regate.lock before it returns.
// Every gh call goes through prchecks.go, pinned to origin's repository with
// -R, as ralph org merge's do.

// regateCIDone is what a ci regate refused by regateCIPreCheck did.
const regateCIDone = "nothing was merged, pushed, or recorded"

// regateCI is what regateCIPreCheck found: the runner and the timeout of
// the gh calls, origin's repository, and the branch's PR, which is origin's,
// OPEN, and merges into the default branch; and the wait for the PR's checks
// (RegateParams.WaitChecks), which Regate sets once the pre-checks passed
// (newRegateCIWait).
type regateCI struct {
	run     MergeOrderExec
	timeout time.Duration
	origin  MergeOrigin
	pr      MergeOrderPR
	wait    *regateCIWait
}

// regateCIPreCheck makes a ci regate's checks of D2 step 1 in t's worktree,
// once the fetches have read origin's default branch, in this order: origin's
// URL names a GitHub repository (ReadMergeOrigin), gh is on PATH
// (p.LookPath), and the branch's PR, as ralph org merge finds it
// (findOriginPR), is a PR of that repository, OPEN, and merges into the
// default branch. It returns what they found, or the refusal of the first that
// fails, worded with w, which says that nothing was merged, pushed, or
// recorded and names one next step; control characters from gh in it are
// escaped (PrintableInboxText). The error is for ctx done.
func regateCIPreCheck(ctx context.Context, p RegateParams, t gateTarget, w gateWording) (*regateCI, string, error) {
	ci := &regateCI{run: p.Exec, timeout: p.GHTimeout}
	if ci.run == nil {
		ci.run = realMergeOrderExec
	}
	if ci.timeout <= 0 {
		ci.timeout = defaultMergeOrderGHTimeout
	}
	lookPath := p.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	refuse := func(format string, args ...any) (*regateCI, string, error) {
		return nil, PrintableInboxText(w.prefix+fmt.Sprintf(format, args...), false), nil
	}

	origin, problem, err := ReadMergeOrigin(ctx, t.root, ci.run, ci.timeout)
	switch problem {
	case MergeOriginInterrupted:
		return nil, "", regateCIInterrupted(ctx, mergeGuaranteeGetURL+" ran", regateCIDone)
	case MergeOriginTimedOut:
		return refuse("%s in %s timed out after %s, so origin's repository, whose PR checks a regate with "+
			"[org.gate].regate_checks = \"ci\" reads, is unknown and %s: check the repository's git config and %s",
			mergeGuaranteeGetURL, t.root, ci.timeout, regateCIDone, w.again)
	case MergeOriginGetURLFailed:
		return refuse("%s in %s failed (%s), so origin's repository, whose PR checks a regate with "+
			"[org.gate].regate_checks = \"ci\" reads, is unknown and %s: add origin (git -C %s remote add origin <URL>) and %s",
			mergeGuaranteeGetURL, t.root, mergeGuaranteeErrText(err), regateCIDone, t.root, w.again)
	case MergeOriginNoRepository:
		return refuse("origin's URL in %s names no host and owner/repo (a local path, file://, or a scheme other than https, "+
			"http, ssh and git), so there is no GitHub repository whose PR checks a regate with [org.gate].regate_checks = "+
			"\"ci\" could read, and %s: point origin at the GitHub repository (git -C %s remote set-url origin <URL>), or set "+
			"[org.gate].regate_checks = \"local\" to run the gate scripts here, and %s", t.root, regateCIDone, t.root, w.again)
	}
	if _, err := lookPath("gh"); err != nil {
		return refuse("%s", regateCIGHMissing(origin, err, regateCIDone, w))
	}

	f := findOriginPR(ctx, t.root, ci.run, ci.timeout, origin, t.branch, t.base.branch)
	switch f.problem {
	case originPRInterrupted:
		return nil, "", regateCIInterrupted(ctx, f.cmd+" ran", regateCIDone)
	case originPRGHFailed:
		return refuse("%s", regateCIGHFailed(origin, f.cmd, f.err, "the branch's PR is unknown", regateCIDone, w))
	case originPRLimitForks:
		return refuse("%s returned %d PRs, its limit, all from forks, so the branch's own PR, whose checks a regate with "+
			"[org.gate].regate_checks = \"ci\" reads, may be past them, and %s: set [org.gate].regate_checks = \"local\" to "+
			"run the gate scripts here instead, and %s", f.cmd, mergeOrderGHLimit, regateCIDone, w.again)
	case originPRNone:
		return refuse("%s found no PR of the branch %s in origin's repository %s, whose checks a regate with "+
			"[org.gate].regate_checks = \"ci\" reads, so %s: push the branch and open its PR into %s (gh pr create --head %s "+
			"--base %s -R %s), and once CI ran on it, %s", f.cmd, t.branch, origin.Arg(), regateCIDone, t.base.branch,
			t.branch, t.base.branch, origin.Arg(), w.again)
	case originPRForeign:
		return refuse("%s returned PR #%d with the url %s, which is not a PR of origin's repository %s, so %s: check origin "+
			"(git -C %s remote get-url origin) and %s", f.cmd, f.pr.Number, f.pr.URL, origin.Arg(), regateCIDone, t.root, w.again)
	case originPRNotOpen:
		return refuse("PR #%d of the branch %s is %s, not OPEN, so its checks are not those of a PR to merge, and %s: when "+
			"it was merged, there is nothing to regate; when it was closed, reopen it (gh pr reopen %d -R %s) or open another "+
			"into %s, and %s", f.pr.Number, t.branch, f.pr.State, regateCIDone, f.pr.Number, origin.Arg(), t.base.branch, w.again)
	case originPRWrongBase:
		return refuse("PR #%d of the branch %s merges into %s, not origin's default branch %s, so %s: change its base (gh pr "+
			"edit %d --base %s -R %s) and %s", f.pr.Number, t.branch, f.pr.BaseRefName, t.base.branch, regateCIDone,
			f.pr.Number, t.base.branch, origin.Arg(), w.again)
	}
	ci.origin, ci.pr = origin, *f.pr
	return ci, "", nil
}

// regateCIGHMissing is the refusal text, after its prefix, for gh not on PATH
// (err): by LookPath, or at a gh call that is not found. done says what was
// not done.
func regateCIGHMissing(origin MergeOrigin, err error, done string, w gateWording) string {
	return fmt.Sprintf("gh is not on PATH (%s), and a regate with [org.gate].regate_checks = \"ci\" reads the checks of the "+
		"branch's PR with gh, so %s: install gh, log in to %s (gh auth login --hostname %s), and %s", mergeGuaranteeErrText(err),
		done, origin.apiHost, origin.apiHost, w.again)
}

// regateCIGHFailed is the refusal text, after its prefix, for cmd, a gh call
// that returned err with ctx not done: a gh not found (regateCIGHMissing), or
// one that failed, timed out, or printed what is not its JSON, which says what
// is unknown and that done was not done.
func regateCIGHFailed(origin MergeOrigin, cmd string, err error, unknown, done string, w gateWording) string {
	if errors.Is(err, exec.ErrNotFound) {
		return regateCIGHMissing(origin, err, done, w)
	}
	return fmt.Sprintf("%s failed (%s), so %s and %s: check that gh can reach %s and is logged in to it (gh auth status "+
		"--hostname %s), and %s", cmd, mergeGuaranteeErrText(err), unknown, done, origin.apiHost, origin.apiHost, w.again)
}

// regateCIInterrupted is a ci regate's error when ctx is done while step,
// done saying what was not done.
func regateCIInterrupted(ctx context.Context, step, done string) error {
	return fmt.Errorf("org: regate: interrupted (%v) while %s: %s", context.Cause(ctx), step, done)
}

// regateCIMerged is regateMerged's path for a ci regate once it has made its
// merge commit res.Merge of t's base tip on res.PreMergeHead, origin's tip of
// the branch, with t.head the merge commit (D2 steps 3 and 4): it runs no gate
// script, pushes the merge commit before anything is recorded (regateCIPush,
// which reads origin's tip back after a push that failed, D3), and once
// origin has it, decides the gate from the PR's head and checks on it
// (regateChecksWithWait). A gate_ci_passed or gate_ci_failed it records is of
// the merge commit on t's base tip and ref, with RegateOutcomeMerged; when
// the checks are not done, as right after the push, when the PR's head is
// usually not the merge commit yet or its checks are pending, it records
// nothing, with RegateOutcomeChecksPending, and the next Regate finds HEAD on
// origin's tip of the branch and reads the checks again (D6); with
// RegateParams.WaitChecks it waits for them first, and ends
// RegateOutcomeUpToDate when another Regate recorded a pass of the merge
// commit meanwhile. Once origin has the merge commit, nothing undoes it: an
// interrupt, an error, or a refusal after the push, those of the wait
// included, records nothing and says that the merge commit stays on origin.
// stderr gets the line regateUncancelledGit writes for an undo.
func (o *Org) regateCIMerged(ctx context.Context, b regateBranch, t gateTarget, ci *regateCI, env []string, stderr io.Writer,
	w gateWording, res RegateResult) (RegateResult, error) {
	res, err := regateCIPush(ctx, t, env, stderr, w, res)
	switch {
	case err != nil && res.Pushed:
		// regateSetTrackingRef failed after the push; its error says the
		// merge commit was pushed.
		return res, fmt.Errorf("%w; nothing was recorded: %s, which reads the PR's checks of it", err, w.again)
	case err != nil, !res.Pushed:
		return res, err
	case ctx.Err() != nil:
		return res, fmt.Errorf("org: regate: interrupted (%v) after %s was pushed to origin's %s, so nothing was recorded: the merge "+
			"commit stays on origin and HEAD is kept; %s, which reads the PR's checks of it", context.Cause(ctx), t.head, t.branch, w.again)
	}
	res, err = o.regateChecksWithWait(ctx, b, t, ci, env, res)
	switch {
	case err != nil:
		return res, fmt.Errorf("%w; %s: %s, which reads the PR's checks of it", err, regateCIPushedText(t), w.again)
	case res.Refused != "":
		res.Refused += "; " + regateCIPushedText(t)
	case res.Outcome == "":
		res.Outcome = RegateOutcomeMerged
	}
	return res, nil
}

// regateCIPushedText says that t's HEAD, a ci regate's merge commit, is on
// origin, for an error or a refusal after its push.
func regateCIPushedText(t gateTarget) string {
	return fmt.Sprintf("the merge commit %s was pushed to origin's %s and stays there", t.head, t.branch)
}

// regateCIPush pushes t's HEAD, the merge commit res.Merge on res.PreMergeHead,
// to origin's branch without force for a ci regate, with nothing recorded yet
// (D2 step 3): `git push origin <sha>:refs/heads/<branch>` with env and
// GIT_TERMINAL_PROMPT=0 (regateRemoteEnv) in ralph's process group
// (regateRemoteGit, D8), as regatePush does. Once origin has the merge commit,
// it sets res.Pushed and points refs/remotes/origin/<branch> at it
// (regateSetTrackingRef, from the value it read right before the push); its
// error is then that ref's update. Origin has it when the push succeeds, or
// when the push failed (an exit, a push past regateGitTimeout or ended by a
// signal, a git that cannot run) and origin's tip read back from origin
// (regateReadOriginTip) is the merge commit: the push landed and its answer
// was lost (D3). Otherwise res.Pushed stays false and nothing more is done:
//
//   - ctx done before the push, or the remote-tracking ref that cannot be
//     read before it: nothing was pushed, so the merge commit is undone
//     (regateUndoMerge), and the error says so;
//   - a push that failed with origin's tip read back as res.PreMergeHead,
//     the tip the merge commit was made on: the push did not land, so the
//     merge commit is undone, and res.Refused says how the push failed, with
//     what git said, and to run Regate again once that is fixed;
//   - a push that failed with origin's tip read back as another commit (a
//     descendant of the merge commit, or any other), with no such branch on
//     origin, or not read: whether origin has the merge commit cannot be
//     told, so HEAD is kept, and res.Refused names what was read and how to
//     check origin and HEAD before running Regate again;
//   - ctx done while the push or the read-back runs: the same cannot be
//     told, HEAD is kept, and the error says so.
//
// The undo writes its line to stderr (regateUncancelledGit).
func regateCIPush(ctx context.Context, t gateTarget, env []string, stderr io.Writer, w gateWording, res RegateResult) (RegateResult,
	error) {
	if ctx.Err() != nil {
		res = regateUndoMerge(ctx, t, env, stderr, res)
		return res, fmt.Errorf("org: regate: interrupted (%v) before %s was pushed to origin's %s, so nothing was pushed or recorded; %s",
			context.Cause(ctx), t.head, t.branch, regateUndoneText(res))
	}
	spec := t.head + ":refs/heads/" + t.branch
	command := "git push origin " + spec + " in " + t.root
	before, err := regateTrackingRef(ctx, t)
	if err != nil {
		res = regateUndoMerge(ctx, t, env, stderr, res)
		if ctx.Err() != nil {
			return res, fmt.Errorf("org: regate: interrupted (%v) before %s was pushed to origin's %s, so nothing was pushed or "+
				"recorded; %s", context.Cause(ctx), t.head, t.branch, regateUndoneText(res))
		}
		return res, fmt.Errorf("%w, so %s was not pushed to origin's %s and nothing was recorded; %s; %s", err, t.head, t.branch,
			regateUndoneText(res), w.again)
	}
	unknown := func(while string) error {
		return fmt.Errorf("org: regate: interrupted (%v) while %s, so origin may or may not have %s, and nothing was recorded: HEAD, "+
			"the merge commit, is kept; %s", context.Cause(ctx), while, t.head, regateCICheckOrigin(t, w))
	}

	r, timedOut, err := regateRemoteGit(ctx, t.root, regateRemoteEnv(env), "push", "origin", spec)
	var failed string
	switch {
	case err != nil && ctx.Err() != nil:
		return res, unknown(fmt.Sprintf("%s was pushed to origin's %s", t.head, t.branch))
	case err != nil:
		// A signal (regateGitSignaled) or a git that cannot run: err says
		// which, after the prefix and the command.
		failed = strings.TrimPrefix(err.Error(), regatePrefix)
	case timedOut:
		failed = fmt.Sprintf("%s ran past %s and was stopped", command, regateGitTimeout)
	case r.code != 0:
		failed = fmt.Sprintf("%s failed (exit %d)", command, r.code)
	default:
		return regateCILanded(ctx, t, before, res)
	}

	if hook := regateTestHooks.beforeReadBack; hook != nil {
		hook()
	}
	tip, unread, err := regateReadOriginTip(ctx, t, env)
	if err != nil {
		return res, unknown(fmt.Sprintf("origin's tip of %s was read back after %s", t.branch, failed))
	}
	said := regateStderrTail(r.stderr)
	var read string
	switch {
	case unread == "" && tip == res.PreMergeHead:
		res = regateUndoMerge(ctx, t, env, stderr, res)
		res.Refused = w.prefix + fmt.Sprintf("%s, and origin's %s is still %s, the tip the merge commit was made on, so the push did "+
			"not land and nothing was recorded: fix it and %s%s; %s", failed, t.branch, tip, w.again, said,
			regateUndoneText(res))
		return res, nil
	case unread == "" && tip == t.head:
		return regateCILanded(ctx, t, before, res)
	case unread != "":
		read = fmt.Sprintf("origin's tip of %s could not be read back (%s)", t.branch, unread)
	case tip == "":
		read = fmt.Sprintf("origin has no branch %s now", t.branch)
	default:
		read = fmt.Sprintf("origin's %s is now %s, neither the tip %s the merge commit was made on nor the merge commit", t.branch, tip,
			res.PreMergeHead)
	}
	res.Refused = w.prefix + fmt.Sprintf("%s, and %s, so whether origin has %s cannot be told, and nothing was recorded: HEAD, the "+
		"merge commit, is kept; %s%s", failed, read, t.head, regateCICheckOrigin(t, w), said)
	return res, nil
}

// regateCILanded is regateCIPush once origin has t's HEAD: it sets
// res.Pushed and points refs/remotes/origin/<branch> at HEAD from before, its
// value right before the push (regateSetTrackingRef), whose error it returns.
func regateCILanded(ctx context.Context, t gateTarget, before string, res RegateResult) (RegateResult, error) {
	res.Pushed = true
	if hook := regateTestHooks.afterPush; hook != nil {
		hook()
	}
	return res, regateSetTrackingRef(context.WithoutCancel(ctx), t, before)
}

// regateCICheckOrigin is the next step after a ci regate's push whose landing
// cannot be told: check origin and t's HEAD, then run Regate again.
func regateCICheckOrigin(t gateTarget, w gateWording) string {
	return fmt.Sprintf("check origin and HEAD (git -C %s fetch origin, then git -C %s log --oneline --graph HEAD "+
		"refs/remotes/origin/%s) and %s", t.root, t.root, t.branch, w.again)
}

// regateChecksRead is what readRegateChecks found of a ci regate's PR, one of
// three: why its head and checks are not done yet (pending), why they tell
// nothing (refused, a refusal with its prefix), or the checks that decide the
// record (checks, with reason GateReasonCIFailed, or "" for a pass).
type regateChecksRead struct {
	pending, refused string
	checks           prChecks
	reason           string
}

// readRegateChecks reads ci's PR for t's HEAD, origin's tip of the branch or
// the merge commit regateCIMerged pushed on preMerge (empty on the gate-only
// path), and decides from it (D4, D6), writing nothing and taking no lock: it
// reads the PR's head (readPRHead) and, when that is HEAD, its checks
// (readPRChecks), and, when those would decide, the PR's head once more,
// since gh pr checks names no commit:
//
//   - pending, when the PR's head is not HEAD (still preMerge, as GitHub may
//     not have updated the PR after the push yet; or another commit, as when
//     GitHub has not updated it after another push yet or origin's branch
//     moved after Regate read it), when a check is pending, others that
//     failed included, when gh reported no checks, and when the PR's head
//     read again is not HEAD any more (it moved while the checks were read);
//   - checks with reason GateReasonCIFailed, when a check ended and is not
//     pass or skipping and none is pending;
//   - checks with reason "", when every check is pass or skipping and one
//     passed;
//   - refused, when every check is skipping, or the head or the checks cannot
//     be read (gh failed, timed out, was not found, or printed no JSON).
//
// Texts from gh are escaped (PrintableInboxText). The error is for ctx done.
func readRegateChecks(ctx context.Context, t gateTarget, ci *regateCI, preMerge string) (regateChecksRead, error) {
	w := regateWords(t.branch)
	n, repoArg := ci.pr.Number, ci.origin.Arg()
	refusal := func(format string, args ...any) string {
		return PrintableInboxText(w.prefix+fmt.Sprintf(format, args...), false)
	}
	refuse := func(format string, args ...any) (regateChecksRead, error) {
		return regateChecksRead{refused: refusal(format, args...)}, nil
	}
	pending := func(format string, args ...any) (regateChecksRead, error) {
		return regateChecksRead{pending: PrintableInboxText(fmt.Sprintf(format, args...), false)}, nil
	}
	// readHead reads the PR's head, or the refusal when it cannot be read.
	viewCmd := fmt.Sprintf("gh pr view %d -R %s --json headRefOid", n, repoArg)
	readHead := func() (head, refused string, err error) {
		head, err = readPRHead(ctx, t.root, ci.run, ci.timeout, repoArg, n)
		switch {
		case ctx.Err() != nil:
			return "", "", regateCIInterrupted(ctx, viewCmd+" ran", "nothing was recorded")
		case err != nil:
			return "", refusal("%s", regateCIGHFailed(ci.origin, viewCmd, err, "the PR's head is unknown", "nothing was recorded", w)), nil
		}
		return head, "", nil
	}

	head, refused, err := readHead()
	switch {
	case err != nil:
		return regateChecksRead{}, err
	case refused != "":
		return regateChecksRead{refused: refused}, nil
	case head != t.head && head == preMerge:
		return pending("PR #%d's head is %s, not %s, the merge commit regate pushed to origin's %s: GitHub has not updated "+
			"the PR to the push yet", n, shortMergeSHA(head), shortMergeSHA(t.head), t.branch)
	case head != t.head:
		return pending("PR #%d's head is %s, not %s, origin's tip of %s: either GitHub has not updated the PR yet, or "+
			"origin's %s moved after regate read it", n, shortMergeSHA(head), shortMergeSHA(t.head), t.branch, t.branch)
	}

	c := readPRChecks(ctx, t.root, ci.run, ci.timeout, repoArg, n, t.branch)
	switch c.read {
	case prChecksInterrupted:
		return regateChecksRead{}, regateCIInterrupted(ctx, c.cmd+" ran", "nothing was recorded")
	case prChecksGHFailed:
		return refuse("%s", regateCIGHFailed(ci.origin, c.cmd, c.err, "the PR's checks are unknown", "nothing was recorded", w))
	case prChecksNotList:
		return refuse("%s printed output that is not a JSON list of checks, so the PR's checks are unknown and nothing was "+
			"recorded: check that gh can reach %s and is logged in to it (gh auth status --hostname %s), and %s", c.cmd,
			ci.origin.apiHost, ci.origin.apiHost, w.again)
	}
	switch {
	case c.total == 0:
		return pending("%s reported no checks yet", c.cmd)
	case len(c.pending) > 0 && len(c.failed) > 0:
		return pending("%d of the %d checks of PR #%d are pending: %s; %d failed: %s, and regate records the outcome once "+
			"none is pending", len(c.pending), c.total, n, mergeNamedBranches(c.pending), len(c.failed), mergeNamedBranches(c.failed))
	case len(c.pending) > 0:
		return pending("%d of the %d checks of PR #%d are pending: %s", len(c.pending), c.total, n, mergeNamedBranches(c.pending))
	}

	// The checks would decide: they are HEAD's only if the PR's head is still
	// HEAD once they were read (L4 of
	// docs/reports/self-review-2026-10-29-regate-ci-checks.md).
	switch again, refused, err := readHead(); {
	case err != nil:
		return regateChecksRead{}, err
	case refused != "":
		return regateChecksRead{refused: refused}, nil
	case again != t.head:
		return pending("PR #%d's head moved to %s while its checks were read, so they may not be those of %s, origin's tip "+
			"of %s", n, shortMergeSHA(again), shortMergeSHA(t.head), t.branch)
	}
	switch {
	case len(c.failed) > 0:
		return regateChecksRead{checks: c, reason: GateReasonCIFailed}, nil
	case c.passed == 0:
		return refuse("every one of the %d checks of PR #%d is skipping and none passed, so nothing shows that the PR's CI "+
			"passed and nothing was recorded: let a check run on the PR, and once it passed, %s", c.total, n, w.again)
	}
	return regateChecksRead{checks: c}, nil
}

// regateChecksPending is res once a ci regate found the checks of its PR
// (res.PR) not done, with nothing recorded: RegateOutcomeChecksPending,
// ChecksPending why, and ChecksNext running Regate again once CI on the PR
// has finished.
func regateChecksPending(res RegateResult, branch, why string) RegateResult {
	res.Outcome, res.ChecksPending = RegateOutcomeChecksPending, why
	res.ChecksNext = fmt.Sprintf("%s once CI on PR #%d has finished", regateWords(branch).again, res.PR)
	return res
}

// regateChecks is the gate of a ci regate on t's HEAD, origin's tip of the
// branch or the merge commit regateCIMerged pushed, decided from ci's PR in
// place of the scripts (D4, D6), as readRegateChecks reads it:
//
//   - while the PR's head and checks are not done: it records nothing and
//     returns res as regateChecksPending sets it, ChecksPending saying why;
//   - with checks that decide: it records a gate_ci_failed with reason
//     ci_failed, with res.FailedChecks naming the checks that ended and are
//     not pass or skipping, or a gate_ci_passed;
//   - with a refusal: it records nothing and sets res.Refused to it.
//
// A record is gateRecord with via=regate_ci, head HEAD, base and base_ref t's
// base tip and ref, by= and herdr_pane= from env as regateGate's, and log= a
// gate log of b's org (createGateLog) that holds the JSON gh pr checks
// printed, appended by recordRegate after its check that the branch's binding
// and its org are as the first check found them. res keeps what it held, with
// res.PR set to the PR's number and, after a record, LogPath, Passed, and
// Reason, and Outcome left to the caller (RegateOutcomeGated, or
// RegateOutcomeMerged after regateCIMerged's push). regateChecksWithWait runs
// it on both paths, with regate.lock held, and once more when its wait ends
// with the lock taken again. The error is for ctx done, a log that cannot be
// written, and a record that cannot be appended; nothing is recorded then.
func (o *Org) regateChecks(ctx context.Context, b regateBranch, t gateTarget, ci *regateCI, env []string, res RegateResult) (RegateResult,
	error) {
	w := regateWords(t.branch)
	res.PR = ci.pr.Number
	read, err := readRegateChecks(ctx, t, ci, res.PreMergeHead)
	switch {
	case err != nil:
		return res, err
	case read.refused != "":
		res.Refused = read.refused
		return res, nil
	case read.pending != "":
		return regateChecksPending(res, t.branch, read.pending), nil
	}
	c, reason := read.checks, read.reason

	logPath, err := o.writeGateChecksLog(b.orgID, w, c.stdout)
	if err != nil {
		return res, err
	}
	res.LogPath = logPath
	record := gateRecord{
		Reason: reason, Head: t.head, Base: t.base.sha, BaseRef: t.base.ref, Branch: t.branch,
		By: approverName(gateEnvValue(env, "USER")), HerdrPane: gateEnvValue(env, herdrPaneIDEnv) != "", Via: gateViaRegateCI, Log: logPath,
	}
	if hook := regateTestHooks.beforeRecord; hook != nil {
		hook()
	}
	refused, err := o.recordRegate(ctx, b, t.branch, record, "the PR's checks were read")
	switch {
	case err != nil && ctx.Err() != nil:
		return res, fmt.Errorf("org: regate: interrupted (%v) before the record of the PR's checks: nothing was recorded; the "+
			"checks it read are in %s", context.Cause(ctx), logPath)
	case err != nil:
		return res, err
	case refused != "":
		res.Refused = refused
		return res, nil
	}
	if hook := regateTestHooks.afterRecord; hook != nil {
		hook()
	}
	res.Passed, res.Reason = reason == "", reason
	if reason != "" {
		for _, label := range c.failed {
			res.FailedChecks = append(res.FailedChecks, PrintableInboxText(label, false))
		}
	}
	return res, nil
}

// writeGateChecksLog writes out, the JSON gh pr checks printed, to a new gate
// log of orgID (createGateLog) and returns the log's absolute path. An error
// starts with w.prefix.
func (o *Org) writeGateChecksLog(orgID string, w gateWording, out string) (string, error) {
	path, f, err := o.createGateLog(orgID, w)
	if err != nil {
		return "", err
	}
	_, err = io.WriteString(f, out)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", fmt.Errorf("%swrite the log %s: %w", w.prefix, path, err)
	}
	return path, nil
}

// regateWaitChecksRefusal is Regate's refusal of p.WaitChecks, made before
// anything is done, or "" for one it takes: a value ralph org merge
// --wait-checks does not take either (MergeWaitChecksRefusal: below 0 or
// above maxMergeWaitChecks), and one above 0 with p.Checks other than
// config.RegateChecksCI, since a local regate runs the gate scripts and has
// no checks of the PR to wait for.
func regateWaitChecksRefusal(p RegateParams) string {
	switch {
	case MergeWaitChecksRefusal(p.WaitChecks) != "":
		return fmt.Sprintf(regatePrefix+"--wait-checks %s is out of range, so nothing was done: give a duration from 0 (no wait, "+
			"the default) to %s, such as 60m", p.WaitChecks, maxMergeWaitChecks)
	case p.WaitChecks > 0 && p.Checks != config.RegateChecksCI:
		return fmt.Sprintf(regatePrefix+"--wait-checks %s waits for the checks of the branch's PR, which regate reads only with "+
			"[org.gate].regate_checks = %q, not %q, which runs the gate scripts instead, so nothing was done: run ralph org "+
			"regate --branch %s again without --wait-checks, or set [org.gate].regate_checks = %q", p.WaitChecks,
			config.RegateChecksCI, config.RegateChecksLocal, p.Branch, config.RegateChecksCI)
	}
	return ""
}

// regateCIWait is the wait of a ci regate for the PR's checks (D5,
// RegateParams.WaitChecks): how long it may wait in all (budget), how long
// each sleep is at most (interval, mergeWaitInterval), the clock and the
// sleep it uses (time.Now and sleepContext, or what RegateParams.waitTest
// sets, as for ralph org merge's wait), where its lines go, and regate.lock
// as Regate holds it (held), which it releases when it starts waiting and
// takes again once, when it stops.
type regateCIWait struct {
	budget, interval time.Duration
	now              func() time.Time
	sleep            func(ctx context.Context, d time.Duration) error
	progress         io.Writer
	held             *regateHeldLock
}

func newRegateCIWait(p RegateParams, held *regateHeldLock, progress io.Writer) *regateCIWait {
	w := &regateCIWait{budget: p.WaitChecks, interval: mergeWaitInterval, now: time.Now, sleep: sleepContext, progress: progress,
		held: held}
	if p.waitTest.interval > 0 {
		w.interval = p.waitTest.interval
	}
	if p.waitTest.now != nil {
		w.now = p.waitTest.now
	}
	if p.waitTest.sleep != nil {
		w.sleep = p.waitTest.sleep
	}
	return w
}

// regateChecksWithWait is the gate of a ci regate on both of its paths, the
// gate-only path and the merge path once regateCIMerged pushed: regateChecks,
// and then, while that ends in RegateOutcomeChecksPending and ci.wait has
// time left (RegateParams.WaitChecks), D5's wait, the way ralph org merge
// --wait-checks waits (merge_wait.go). It writes a line when the wait starts
// and releases regate.lock; while it waits, it only reads (poll: no write, no
// lock). Once the PR's head and checks are no longer pending, it takes the
// lock again, once, and checks what the record relies on (regateCIRecheck),
// and then decides and records under the lock with regateChecks, so that a
// record comes from a read made with the lock held, and the poll's read
// counts for nothing but ending the wait. The wait ends:
//
//   - when the budget, counted from the start of the wait, is used up with
//     them still pending: in RegateOutcomeChecksPending, why the last read
//     found them pending, and how long it waited, with the lock not taken
//     again, since there is nothing to record;
//   - when ctx is done while it sleeps or reads, with the lock not taken
//     again either;
//   - when regateCIRecheck refuses, the lock held by another included, or
//     another Regate recorded a pass of HEAD on t's base tip meanwhile
//     (RegateOutcomeUpToDate, LogPath the log of that pass);
//   - otherwise as regateChecks under the lock ends: a record or a refusal,
//     or, when the checks are pending again (CI ran again), in
//     RegateOutcomeChecksPending with ChecksPending saying so; it does not
//     wait again with the lock held.
//
// The record is still checked against b, as the first check found the branch
// (recordRegate). A line says why the wait ended. The error is
// regateChecks', or one that says the wait was interrupted and nothing was
// recorded; with an error or a refusal, Outcome, ChecksPending, and
// ChecksNext are empty.
func (o *Org) regateChecksWithWait(ctx context.Context, b regateBranch, t gateTarget, ci *regateCI, env []string,
	res RegateResult) (RegateResult, error) {
	res, err := o.regateChecks(ctx, b, t, ci, env, res)
	w := ci.wait
	if err != nil || res.Outcome != RegateOutcomeChecksPending || w == nil || w.budget <= 0 {
		return res, err
	}
	start := w.now()
	w.line(fmt.Sprintf("%s, so ralph org regate waits for the checks of PR #%d for up to %s (--wait-checks %s), with regate.lock "+
		"released, reading the PR's head and its checks every %s", res.ChecksPending, res.PR, w.budget, w.budget, w.interval))
	w.held.release()
	pending := res.ChecksPending
	res.Outcome, res.ChecksPending, res.ChecksNext = "", "", ""
	pending, ranOut, err := w.poll(ctx, start.Add(w.budget), t, ci, res.PreMergeHead, pending)
	waited := func() string {
		return fmt.Sprintf("ralph org regate waited %s for the checks of PR #%d (--wait-checks %s)", mergeWaitDuration(w.now().Sub(start)),
			res.PR, w.budget)
	}
	switch {
	case err != nil:
		return w.ended(ctx, start, res, err)
	case ranOut:
		return w.ended(ctx, start, regateChecksPending(res, t.branch, pending+"; "+waited()+", and that ran out"), nil)
	}

	var now regateBranch
	now, res.Refused, err = o.regateCIRecheck(ctx, b, t, env, w, res.PR)
	switch {
	case ctx.Err() != nil:
		res.Refused = ""
		return w.ended(ctx, start, res, regateCIInterrupted(ctx, fmt.Sprintf("waiting for the checks of PR #%d", res.PR),
			"nothing was recorded"))
	case err != nil, res.Refused != "":
		return w.ended(ctx, start, res, err)
	case now.passedOn(t):
		res.Outcome, res.LogPath = RegateOutcomeUpToDate, now.gate.record.Log
		return w.ended(ctx, start, res, nil)
	}
	if res, err = o.regateChecks(ctx, b, t, ci, env, res); err != nil || res.Outcome != RegateOutcomeChecksPending {
		return w.ended(ctx, start, res, err)
	}
	res.ChecksPending += "; " + waited() + ", and they were not done again when it read them with regate.lock taken again"
	w.stopped(start, res.PR, "the checks were not done again when it read them with regate.lock taken again, so nothing was recorded")
	return res, nil
}

// poll is the loop of regateChecksWithWait's wait, with regate.lock
// released: again and again, it sleeps w's interval, or what is left until
// deadline when that is less, and reads ci's PR for t's HEAD
// (readRegateChecks, on preMerge), which writes nothing and takes no lock,
// until that read is not pending (done, or a refusal: regateChecks decides it
// again under the lock), the deadline is reached, or ctx is done. It returns
// why the last read found the PR's head and checks pending (pending, the
// first read's why until a read after a sleep), and ranOut when the deadline
// was reached with them pending. The error says the wait was interrupted and
// nothing was recorded.
func (w *regateCIWait) poll(ctx context.Context, deadline time.Time, t gateTarget, ci *regateCI, preMerge, pending string) (string, bool,
	error) {
	interrupted := func() error {
		return regateCIInterrupted(ctx, fmt.Sprintf("waiting for the checks of PR #%d", ci.pr.Number), "nothing was recorded")
	}
	for {
		left := deadline.Sub(w.now())
		if left <= 0 {
			return pending, true, nil
		}
		if err := w.sleep(ctx, min(w.interval, left)); err != nil || ctx.Err() != nil {
			return "", false, interrupted()
		}
		read, err := readRegateChecks(ctx, t, ci, preMerge)
		switch {
		case err != nil || ctx.Err() != nil:
			return "", false, interrupted()
		case read.pending == "":
			return "", false, nil
		}
		pending = read.pending
	}
}

// regateCIRecheck is what a ci regate's wait checks once it stopped polling,
// before it reads the PR's head and checks again to decide and record (D5):
// it takes regate.lock again into w.held, refusing when another regate,
// merge, or cleanup holds it after regateLockWait, as ralph org merge refuses
// when its wait ends, empties its marker, and reads the ledger again under the
// manifest lock as the first check does (regateFirstCheck, which writes the
// marker again when the branch passes it); then it refuses when the branch is
// not as the first check found it (was) any more (regateBranchChanged, which
// words every refusal of the first check too, since was passed it), when
// HEAD in t's worktree cannot be read or is not t's HEAD, and when origin's
// tip of the branch, read from origin itself (regateReadOriginTip, as after a
// ci regate's push), cannot be read, is gone, or is not t's HEAD. Each
// refusal says what it found and that nothing was recorded, with pr, the PR
// whose checks the wait waits for, in its text. It returns the branch as it
// read it, whose latest gate record may be a pass another Regate recorded
// meanwhile. The error is for ctx done, a lock that cannot be taken or a
// marker that cannot be written, a manifest that cannot be read, and a git
// that cannot run.
func (o *Org) regateCIRecheck(ctx context.Context, was regateBranch, t gateTarget, env []string, w *regateCIWait, pr int) (regateBranch,
	string, error) {
	words := regateWords(t.branch)
	waited := fmt.Sprintf("regate waited for the checks of PR #%d", pr)
	lock, err := acquireHeldFileLock(w.held.stateDir, regateLockFile, "regate", regateLockWait)
	switch {
	case errors.Is(err, errFileLockHeld):
		return regateBranch{}, fmt.Sprintf(regatePrefix+"another ralph org regate, merge, or cleanup took %s while %s, so nothing "+
			"was recorded: wait for it to end and %s", regateLockPathIn(w.held.stateDir), waited, words.again), nil
	case err != nil:
		return regateBranch{}, "", err
	}
	w.held.lock = lock
	if err := writeRegateMarker(lock.f, nil); err != nil {
		return regateBranch{}, "", err
	}
	now, refused, err := o.regateFirstCheck(ctx, t.branch, lock)
	switch changed := regateBranchChanged(was, now); {
	case err != nil:
		return regateBranch{}, "", err
	case changed != "":
		return regateBranch{}, regateChangedRefusal(t.branch, waited, changed), nil
	case refused != "":
		// Not reached while every refusal of the first check is a change
		// from was; one that is not stays a refusal.
		return regateBranch{}, refused, nil
	}

	var found string
	r, err := gateGit(ctx, regatePrefix, t.root, "rev-parse", "--verify", "HEAD^{commit}")
	switch head := strings.TrimSpace(r.stdout); {
	case err != nil:
		return regateBranch{}, "", err
	case r.code != 0:
		found = fmt.Sprintf("could not read HEAD in %s (git rev-parse --verify HEAD^{commit}: exit %d: %s)", t.root, r.code, r.stderr)
	case head != t.head:
		found = fmt.Sprintf("found HEAD of %s in %s at %s, no longer %s", t.branch, t.root, head, t.head)
	}
	if found == "" {
		tip, unread, err := regateReadOriginTip(ctx, t, env)
		switch {
		case err != nil:
			return regateBranch{}, "", err
		case unread != "":
			found = fmt.Sprintf("could not read origin's tip of %s back (%s)", t.branch, unread)
		case tip == "":
			found = fmt.Sprintf("found no branch %s on origin", t.branch)
		case tip != t.head:
			found = fmt.Sprintf("found origin's %s at %s, no longer HEAD %s", t.branch, tip, t.head)
		}
	}
	if found != "" {
		return regateBranch{}, PrintableInboxText(fmt.Sprintf(regatePrefix+"after it waited for the checks of PR #%d, regate %s, so "+
			"nothing was recorded: %s", pr, found, regateCICheckOrigin(t, words)), false), nil
	}
	return now, "", nil
}

// line writes text after regatePrefix, escaped (PrintableInboxText), as one
// line of w.progress.
func (w *regateCIWait) line(text string) {
	_, _ = fmt.Fprintln(w.progress, PrintableInboxText(regatePrefix+text, false))
}

// ended writes the line that says why the wait that started at start ended,
// read from what regateChecksWithWait returns then (res and err), and
// returns them. RegateOutcomeChecksPending here is the budget used up;
// checks found pending again under the lock write their own line.
func (w *regateCIWait) ended(ctx context.Context, start time.Time, res RegateResult, err error) (RegateResult, error) {
	var why string
	switch {
	case err != nil && ctx.Err() != nil:
		why = fmt.Sprintf("interrupted (%v), so nothing was recorded", context.Cause(ctx))
	case err != nil:
		why = "it ended in an error, so nothing was recorded"
	case res.Refused != "":
		why = "it was refused, so nothing was recorded"
	case res.Outcome == RegateOutcomeUpToDate:
		why = "another regate recorded a pass of HEAD meanwhile"
	case res.Outcome == RegateOutcomeChecksPending:
		why = fmt.Sprintf("--wait-checks %s is used up, so nothing was recorded", w.budget)
	case res.Passed:
		why = "the checks passed, and " + EventGateCIPassed + " was recorded"
	default:
		why = "a check did not pass, and " + EventGateCIFailed + " was recorded"
	}
	w.stopped(start, res.PR, why)
	return res, err
}

// stopped writes the line that says the wait for the checks of PR #pr that
// started at start ended, and why.
func (w *regateCIWait) stopped(start time.Time, pr int, why string) {
	w.line(fmt.Sprintf("stopped waiting for the checks of PR #%d after %s: %s", pr, mergeWaitDuration(w.now().Sub(start)), why))
}
