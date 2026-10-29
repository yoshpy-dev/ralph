package org

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"
)

// The ci regate of plan docs/plans/active/2026-10-29-regate-ci-checks.md
// ([org.gate].regate_checks = "ci", RegateParams.Checks): in place of the
// gate scripts, Regate (regate.go) reads the checks of the branch's PR on
// origin with gh and records the gate from them. This file holds the checks
// made before anything changes (regateCIPreCheck, D2 step 1) and the
// decision from the PR's head and checks to a gate_ci_passed, a
// gate_ci_failed, or no record (regateChecks, D4), which the gate-only path
// (D6) runs in place of regateGate. A ci regate whose HEAD does not contain
// origin's default branch tip is refused (regateCIMergeRefusal): its merge
// and push are not here yet. Every gh call goes through prchecks.go, pinned
// to origin's repository with -R, as ralph org merge's do.

// regateCIDone is what a ci regate refused by regateCIPreCheck did.
const regateCIDone = "nothing was merged, pushed, or recorded"

// regateCI is what regateCIPreCheck found: the runner and the timeout of
// the gh calls, origin's repository, and the branch's PR, which is origin's,
// OPEN, and merges into the default branch.
type regateCI struct {
	run     MergeOrderExec
	timeout time.Duration
	origin  MergeOrigin
	pr      MergeOrderPR
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

// regateCIMergeRefusal is the refusal of a ci regate whose HEAD, origin's
// tip of the branch, does not contain t's base tip: the merge of that tip
// with the push before the checks (D2 steps 2 to 4, D3) is not in this
// regate, which does not fall back to the gate scripts either (the plan's
// first critical fork). It is refused before anything changed.
func regateCIMergeRefusal(t gateTarget, w gateWording) string {
	return w.prefix + fmt.Sprintf("HEAD %s of %s in %s does not contain origin's tip %s of %s, and this ralph does not merge it "+
		"in with [org.gate].regate_checks = \"ci\", so %s: merge it yourself (git -C %s merge %s), push, and %s, which then "+
		"reads the PR's checks; or set [org.gate].regate_checks = \"local\" and %s, which merges and runs the gate scripts "+
		"here", t.head, t.branch, t.root, t.base.sha, t.base.branch, regateCIDone, t.root, t.base.ref, w.again, w.again)
}

// regateChecks is the gate of a ci regate on t's HEAD, origin's tip of the
// branch, decided from ci's PR in place of the scripts (D4, D6): it reads the
// PR's head (readPRHead) and, when that is HEAD, its checks (readPRChecks),
// and then
//
//   - when the PR's head is not HEAD yet (GitHub may not have updated the PR
//     after a push), a check is pending, others that failed included, or gh
//     reported no checks: it records nothing and sets res.ChecksPending to
//     why and res.ChecksNext to running Regate again once CI has finished;
//   - when a check ended and is not pass or skipping, and none is pending: it
//     records a gate_ci_failed with reason ci_failed, and res.FailedChecks
//     names those checks;
//   - when every check is pass or skipping and one passed: it records a
//     gate_ci_passed;
//   - when every check is skipping, or the head or the checks cannot be read
//     (gh failed, timed out, was not found, or printed no JSON): it records
//     nothing and sets res.Refused to why.
//
// A record is gateRecord with via=regate_ci, head HEAD, base and base_ref t's
// base tip and ref, by= and herdr_pane= from env as regateGate's, and log= a
// gate log of b's org (createGateLog) that holds the JSON gh pr checks
// printed, appended by recordRegate after its check that the branch's binding
// and its org are as the first check found them. res keeps what it held, with
// res.PR set to the PR's number and, after a record, LogPath, Passed, and
// Reason. Texts from gh are escaped (PrintableInboxText). The error is for ctx
// done, a log that cannot be written, and a record that cannot be appended;
// nothing is recorded then.
func (o *Org) regateChecks(ctx context.Context, b regateBranch, t gateTarget, ci *regateCI, env []string, res RegateResult) (RegateResult,
	error) {
	w := regateWords(t.branch)
	n, repoArg := ci.pr.Number, ci.origin.Arg()
	res.PR = n
	refuse := func(format string, args ...any) (RegateResult, error) {
		res.Refused = PrintableInboxText(w.prefix+fmt.Sprintf(format, args...), false)
		return res, nil
	}
	pending := func(format string, args ...any) (RegateResult, error) {
		res.ChecksPending = PrintableInboxText(fmt.Sprintf(format, args...), false)
		res.ChecksNext = fmt.Sprintf("%s once CI on PR #%d has finished", w.again, n)
		return res, nil
	}

	viewCmd := fmt.Sprintf("gh pr view %d -R %s --json headRefOid", n, repoArg)
	head, err := readPRHead(ctx, t.root, ci.run, ci.timeout, repoArg, n)
	switch {
	case ctx.Err() != nil:
		return res, regateCIInterrupted(ctx, viewCmd+" ran", "nothing was recorded")
	case err != nil:
		return refuse("%s", regateCIGHFailed(ci.origin, viewCmd, err, "the PR's head is unknown", "nothing was recorded", w))
	case head != t.head:
		return pending("PR #%d's head is %s, not %s, origin's tip of %s: GitHub has not updated the PR yet", n,
			shortMergeSHA(head), shortMergeSHA(t.head), t.branch)
	}

	c := readPRChecks(ctx, t.root, ci.run, ci.timeout, repoArg, n, t.branch)
	switch c.read {
	case prChecksInterrupted:
		return res, regateCIInterrupted(ctx, c.cmd+" ran", "nothing was recorded")
	case prChecksGHFailed:
		return refuse("%s", regateCIGHFailed(ci.origin, c.cmd, c.err, "the PR's checks are unknown", "nothing was recorded", w))
	case prChecksNotList:
		return refuse("%s printed output that is not a JSON list of checks, so the PR's checks are unknown and nothing was "+
			"recorded: check that gh can reach %s and is logged in to it (gh auth status --hostname %s), and %s", c.cmd,
			ci.origin.apiHost, ci.origin.apiHost, w.again)
	}
	var reason string
	switch {
	case c.total == 0:
		return pending("%s reported no checks yet", c.cmd)
	case len(c.pending) > 0 && len(c.failed) > 0:
		return pending("%d of the %d checks of PR #%d are pending: %s; %d failed: %s, and regate records the outcome once "+
			"none is pending", len(c.pending), c.total, n, mergeNamedBranches(c.pending), len(c.failed), mergeNamedBranches(c.failed))
	case len(c.pending) > 0:
		return pending("%d of the %d checks of PR #%d are pending: %s", len(c.pending), c.total, n, mergeNamedBranches(c.pending))
	case len(c.failed) > 0:
		reason = GateReasonCIFailed
	case c.passed == 0:
		return refuse("every one of the %d checks of PR #%d is skipping and none passed, so nothing shows that the PR's CI "+
			"passed and nothing was recorded: let a check run on the PR, and once it passed, %s", c.total, n, w.again)
	}

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
