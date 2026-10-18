package org

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"
)

// The `ralph org merge` verb, stage 8a-3a of plan
// docs/plans/active/2026-10-17-org-auto-merge.md (FR-8 of
// docs/specs/2026-10-07-org-multi-org-director.md): it merges, with gh, the
// PR of one branch ralph org merge-order lists as ready, once it has checked
// everything the merge rests on. This file holds the checks (checkMerge, S3a
// of the plan): they merge nothing, write nothing, and take no lock. The merge
// itself is in merge_run.go (Merge, S3b): regate.lock, gh pr merge, the read
// of the PR after it, and the record of pr_merged. The wait of --wait-checks,
// for checks that are pending or not reported yet (S7) and for an org that
// still runs the branch's feature to be disbanded (S8c; on a branch with no
// binding, also its gate's org that runs another feature now:
// runningRefusal), is in merge_wait.go.
//
// The repository is origin's (D10): checkMerge reads its host and owner/repo
// from `git remote get-url origin` and pins every gh call to it, with `-R
// <host>/<owner>/<repo>` for gh pr, the same as the repository argument of gh
// repo view, and --hostname <host> for gh api (CheckMergeEnforcement), so
// neither gh repo set-default nor an upstream remote makes gh read another
// one. A PR whose url names another host or owner/repo is refused. Every gh
// call carries GH_PROMPT_DISABLED=1, every git call GIT_TERMINAL_PROMPT=0, and
// all of them run through MergeOrderExec.

const (
	// mergePrefix starts every refusal and error of the merge verb.
	mergePrefix = "org: merge: "
	// mergePRViewFields are the fields checkMerge reads with gh pr view.
	mergePRViewFields = "state,url,headRefOid,baseRefName,autoMergeRequest"
	// mergePRChecksFields are the fields checkMerge reads with gh pr checks.
	mergePRChecksFields = "bucket,name,state,workflow"
	// mergeNoChecksSentence is what gh pr checks prints on stderr, exiting 1
	// with no JSON on stdout, not even [], for a PR with no checks, %s being
	// the PR's head branch (gh 2.102.0: populateStatusChecks in
	// pkg/cmd/pr/checks/checks.go returns it before the JSON is written).
	mergeNoChecksSentence = "no checks reported on the '%s' branch"
)

// mergeMethodFlags maps the merge methods gh repo view reports
// (viewerDefaultMergeMethod) to gh pr merge's flag (D2). Any other value is
// refused.
var mergeMethodFlags = map[string]string{"MERGE": "--merge", "SQUASH": "--squash", "REBASE": "--rebase"}

// MergeParams describes one `ralph org merge` call.
type MergeParams struct {
	// Branch is the feature branch whose PR is merged.
	Branch string
	// Dir is the repository directory git and gh run in; empty means the
	// current directory.
	Dir string
	// Exec runs git and gh; nil means realMergeOrderExec, and
	// realMergeDetachedExec for the gh calls after gh pr merge started
	// (merge_run.go).
	Exec MergeOrderExec
	// LookPath finds gh before any gh call; nil means exec.LookPath.
	LookPath func(file string) (string, error)
	// LsRemoteTimeout bounds each ls-remote, GHTimeout each gh call and the
	// `git remote get-url origin`, and EnforcementTimeout the whole of
	// CheckMergeEnforcement. Zero or less means 30 s, 15 s and 20 s.
	LsRemoteTimeout, GHTimeout, EnforcementTimeout time.Duration
	// MergeTimeout bounds gh pr merge, which Merge runs once (merge_run.go);
	// zero or less means 60 s. The gh calls after it are bounded by
	// GHTimeout.
	MergeTimeout time.Duration
	// AutoMerge is [org.director].auto_merge, which the verb only reports
	// (D1): the checks do not read it, and the verb merges whatever it is.
	AutoMerge bool
	// WaitChecks is --wait-checks (D13, merge_wait.go): how long Merge may
	// wait, with regate.lock released, for checks that are pending or not
	// reported yet and for the branch's org to be disbanded, from 0, which
	// waits not at all, to maxMergeWaitChecks.
	WaitChecks time.Duration
	// Progress gets the line a wait writes when it starts and the one it
	// writes when it ends; nil discards them.
	Progress io.Writer

	// waitTest replaces the wait's interval, clock, and sleep in a test.
	waitTest mergeWaitTest
}

// mergeReady is what checkMerge found when every check passed: what the
// merge of the branch's PR needs. With a refusal whose kind waits
// (mergeRefusalKind.waits), only the fields read up to the refusing step are
// filled: for step 8 branch, origin's fields, and orgID, and for step 10 all
// but protection, method, and methodFlag.
type mergeReady struct {
	branch string
	// orgID is the org the merge is recorded under (D4): the org of the
	// branch's latest binding, else of its latest gate record.
	orgID string
	// apiHost is the --hostname of gh api (origin's host, ssh.github.com read
	// as github.com), repo origin's owner/repo, and repoArg the -R of gh pr,
	// <apiHost>/<owner>/<repo>.
	apiHost string
	repo    mergeRepo
	repoArg string
	// prNumber and prURL are the branch's PR, as gh pr view read it.
	prNumber int
	prURL    string
	// head and base are the gate record's: the head the gate passed, which is
	// the PR's head and origin's tip of the branch, and the base it passed on,
	// origin's tip of defaultBranch.
	head, base    string
	defaultBranch string
	// method is the merge method (merge, squash or rebase) and methodFlag
	// gh pr merge's flag for it (--merge, --squash or --rebase).
	method, methodFlag string
	// protection is CheckMergeEnforcement's Detail: why GitHub keeps the
	// merge up to date.
	protection string
}

// isOriginsPR reports whether u is the url of a PR of r's repository: its
// host is r.apiHost and its owner/repo is r.repo, without case
// (mergeHostRepoFromPRURL).
func (r mergeReady) isOriginsPR(u string) bool {
	host, repo, ok := mergeHostRepoFromPRURL(u)
	return ok && strings.EqualFold(host, r.apiHost) && repo.same(r.repo)
}

// mergeCheck is one checkMerge while it runs: its MergeParams with the
// defaults filled in.
type mergeCheck struct {
	branch                                   string
	dir                                      string
	run                                      MergeOrderExec
	lookPath                                 func(string) (string, error)
	lsTimeout, ghTimeout, enforcementTimeout time.Duration
	// mergeTimeout bounds gh pr merge (Merge).
	mergeTimeout time.Duration
	// runDetached runs the gh calls Merge makes once gh pr merge has started
	// (mergeRun.ghAfterStart): realMergeDetachedExec, a process group of
	// their own, when MergeParams.Exec is nil, else MergeParams.Exec.
	runDetached MergeOrderExec
}

func newMergeCheck(p MergeParams) *mergeCheck {
	m := &mergeCheck{branch: p.Branch, dir: p.Dir, run: p.Exec, lookPath: p.LookPath, lsTimeout: p.LsRemoteTimeout,
		ghTimeout: p.GHTimeout, enforcementTimeout: p.EnforcementTimeout, mergeTimeout: p.MergeTimeout, runDetached: p.Exec}
	if m.run == nil {
		m.run, m.runDetached = realMergeOrderExec, realMergeDetachedExec
	}
	if m.lookPath == nil {
		m.lookPath = exec.LookPath
	}
	if m.lsTimeout <= 0 {
		m.lsTimeout = defaultMergeOrderLsRemoteTimeout
	}
	if m.ghTimeout <= 0 {
		m.ghTimeout = defaultMergeOrderGHTimeout
	}
	if m.enforcementTimeout <= 0 {
		m.enforcementTimeout = defaultMergeGuaranteeTimeout
	}
	if m.mergeTimeout <= 0 {
		m.mergeTimeout = defaultMergeTimeout
	}
	return m
}

// checkMerge makes the checks `ralph org merge --branch <p.Branch>` makes
// before it merges, in this order, and returns what the merge needs, or the
// refusal of the first that fails, with nothing after it run:
//
//  1. p.Branch is not empty.
//  2. origin's URL names a host and owner/repo (D10, readOrigin).
//  3. gh is on PATH (p.LookPath); a later gh call that is not found is
//     refused the same way.
//  4. The ledger names the branch: a feature binding or a gate record
//     (readMergeLedger and collectMergeTargets, without the manifest lock).
//  5. The ls-remote reads origin (listOriginBranches).
//  6. gh finds the branch's PR in origin's repository, whose url names that
//     host and owner/repo (findPR).
//  7. merge-order classifies the branch as ready (classifyMergeTarget).
//  8. Its org, the binding's or else the gate record's (D4), is not running
//     (D5).
//  9. gh pr view: the PR is OPEN, its head is the gate's head, its base is
//     origin's default branch, its url names origin's repository, and it has
//     no auto-merge request (D12, viewPR).
//  10. gh pr checks: every check is pass or skipping, and one passed (D7,
//     readChecks).
//  11. CheckMergeEnforcement says GitHub keeps the merge up to date (D11).
//  12. gh repo view gives a merge method it knows (D2, readMethod).
//  13. A second ls-remote finds origin's default branch, its tip and the
//     branch's tip still where the gate left them (D8, checkOriginUnmoved).
//
// A refusal starts with "org: merge: ", says what was found, that nothing was
// merged, and one next step; control characters in it, from the ledger or
// gh, are escaped (PrintableInboxText). kind says which refusal it is
// (mergeRefusalKind): of step 8 for an org that still runs the branch's
// feature (on a branch with no binding, also for its gate's org that runs
// another feature now: runningRefusal), of step 10, and mergeRefusalOther for
// a refusal of any other step and when every check passed. With a refusal
// whose kind waits, r holds what the steps before it read, for the wait of
// --wait-checks (merge_wait.go): for step 8 the branch and its org_id, whose
// disband the wait reads in the ledger, and for step 10 what steps 2 to 9
// read, the PR whose checks the wait reads; r is empty with any other
// refusal. The error is for a manifest that cannot be read and for ctx done;
// nothing was merged then either. p.AutoMerge, p.WaitChecks, and p.Progress
// are not read.
func (o *Org) checkMerge(ctx context.Context, p MergeParams) (r mergeReady, refused string, kind mergeRefusalKind, err error) {
	if p.Branch == "" {
		return mergeReady{}, mergePrefix + "no branch named, so nothing was merged: name the feature branch to merge (ralph " +
			"org merge-order lists the ledger's branches)", mergeRefusalOther, nil
	}
	m := newMergeCheck(p)
	r, refused, kind, err = o.checkMergeBranch(ctx, m)
	if err == nil && refused == "" {
		refused, kind, err = m.checkMergeOnGitHub(ctx, &r)
	}
	switch {
	case err != nil:
		return mergeReady{}, "", mergeRefusalOther, err
	case refused != "" && !kind.waits():
		return mergeReady{}, PrintableInboxText(refused, false), kind, nil
	case refused != "":
		return r, PrintableInboxText(refused, false), kind, nil
	}
	return r, "", mergeRefusalOther, nil
}

// checkMergeBranch is steps 2 to 8 of checkMerge: origin, gh, the ledger,
// the ls-remote, the PR, the class, and the org. It returns what they read;
// kind is runningRefusal's when step 8 refused, else mergeRefusalOther.
func (o *Org) checkMergeBranch(ctx context.Context, m *mergeCheck) (mergeReady, string, mergeRefusalKind, error) {
	r := mergeReady{branch: m.branch}
	if refused, err := m.readOrigin(ctx, &r); err != nil || refused != "" {
		return mergeReady{}, refused, mergeRefusalOther, err
	}
	if _, err := m.lookPath("gh"); err != nil {
		return mergeReady{}, m.ghMissing(r, err), mergeRefusalOther, nil
	}
	ledger, err := readMergeLedger(o.Manifest.Path())
	if err != nil {
		return mergeReady{}, "", mergeRefusalOther, fmt.Errorf("org: merge: read manifest: %w", err)
	}
	targets, _ := collectMergeTargets(ledger)
	i := slices.IndexFunc(targets, func(t *mergeTarget) bool { return t.branch == m.branch })
	if i < 0 {
		return mergeReady{}, fmt.Sprintf(mergePrefix+"no feature binding or gate record in the ledger names the branch %s, so "+
			"nothing was merged: ralph org merge merges only a branch ralph org merge-order lists as ready, so run it to see the "+
			"ledger's branches", m.branch), mergeRefusalOther, nil
	}
	t := targets[i]
	remote, refused, err := m.listOrigin(ctx)
	if err != nil || refused != "" {
		return mergeReady{}, refused, mergeRefusalOther, err
	}
	if refused, err := m.findPR(ctx, r, t); err != nil || refused != "" {
		return mergeReady{}, refused, mergeRefusalOther, err
	}
	if class, detail := classifyMergeTarget(t, remote); class != MergeClassReady {
		return mergeReady{}, fmt.Sprintf(mergePrefix+"ralph org merge-order lists the branch %s as %s, not ready, so nothing was "+
			"merged: %s", m.branch, class, detail), mergeRefusalOther, nil
	}
	r.orgID = mergeTargetOrg(t)
	if refused, kind := m.runningRefusal(t, r.orgID, RunningOrgs(ledger.events)); refused != "" {
		return r, refused, kind, nil
	}
	r.prNumber, r.prURL = t.pr.Number, t.pr.URL
	r.head, r.base, r.defaultBranch = t.gate.record.Head, t.gate.record.Base, remote.defaultBranch
	return r, "", mergeRefusalOther, nil
}

// checkMergeOnGitHub is steps 9 to 13 of checkMerge: the PR as gh pr view
// reads it, its checks, the protection, the merge method, and origin again.
// It fills in r. kind is readChecks' when step 10 refused, else
// mergeRefusalOther.
func (m *mergeCheck) checkMergeOnGitHub(ctx context.Context, r *mergeReady) (string, mergeRefusalKind, error) {
	if refused, err := m.viewPR(ctx, r); err != nil || refused != "" {
		return refused, mergeRefusalOther, err
	}
	if refused, kind, err := m.readChecks(ctx, *r); err != nil || refused != "" {
		return refused, kind, err
	}
	refused, err := m.checkMergeAfterChecks(ctx, r)
	return refused, mergeRefusalOther, err
}

// checkMergeAfterChecks is steps 11 to 13 of checkMerge: the protection, the
// merge method, and origin again. It fills in r.
func (m *mergeCheck) checkMergeAfterChecks(ctx context.Context, r *mergeReady) (string, error) {
	// CheckMergeEnforcement runs its gh calls one after another; the first
	// that is not found is refused as any other gh call of checkMerge is.
	var ghNotFound error
	run := func(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
		out, err := m.run(ctx, dir, env, name, args...)
		if errors.Is(err, exec.ErrNotFound) && ghNotFound == nil {
			ghNotFound = err
		}
		return out, err
	}
	enf := CheckMergeEnforcement(ctx, MergeEnforcementParams{
		Dir: m.dir, Exec: run, Timeout: m.enforcementTimeout, Host: r.apiHost, Owner: r.repo.owner, Name: r.repo.name,
		DefaultBranch: r.defaultBranch,
	})
	switch {
	case ctx.Err() != nil:
		return "", mergeInterrupted(ctx, "the protection of origin's default branch was read")
	case ghNotFound != nil:
		return m.ghMissing(*r, ghNotFound), nil
	case !enf.OK:
		return mergePrefix + "GitHub's protection of origin's default branch was not confirmed, so nothing was merged: " +
			enf.Detail, nil
	}
	r.protection = enf.Detail
	if refused, err := m.readMethod(ctx, r); err != nil || refused != "" {
		return refused, err
	}
	return m.checkOriginUnmoved(ctx, *r)
}

// again is the next step of a refusal that a retry may clear.
func (m *mergeCheck) again() string {
	return "run ralph org merge --branch " + m.branch + " again"
}

// ghAccess is the next step of a refusal for a gh call that failed.
func (m *mergeCheck) ghAccess(r mergeReady) string {
	return fmt.Sprintf("check that gh can reach %s and is logged in to it (gh auth status --hostname %s), and %s", r.apiHost,
		r.apiHost, m.again())
}

// ghMissing is the refusal for gh not on PATH (err): step 3, and any later gh
// call that is not found.
func (m *mergeCheck) ghMissing(r mergeReady, err error) string {
	return fmt.Sprintf(mergePrefix+"gh is not on PATH (%s), and ralph org merge reads the PR and merges it with gh, so nothing "+
		"was merged: install gh, log in to %s (gh auth login --hostname %s), and %s", mergeGuaranteeErrText(err), r.apiHost,
		r.apiHost, m.again())
}

// mergeInterrupted is checkMerge's error when ctx is done while step.
func mergeInterrupted(ctx context.Context, step string) error {
	return fmt.Errorf("org: merge: interrupted (%v) while %s: nothing was merged", context.Cause(ctx), step)
}

// readOrigin is step 2 of checkMerge (D10): origin's host and owner/repo from
// `git remote get-url origin`, bounded by the gh timeout, into r, or the
// refusal of an origin that cannot be read, names none, or names a host,
// owner or repo that starts with "-".
func (m *mergeCheck) readOrigin(ctx context.Context, r *mergeReady) (string, error) {
	getCtx, cancel := context.WithTimeout(ctx, m.ghTimeout)
	defer cancel()
	line, err := getMergeOriginURL(getCtx, m.dir, m.run)
	switch {
	case ctx.Err() != nil:
		return "", mergeInterrupted(ctx, mergeGuaranteeGetURL+" ran")
	case err != nil && getCtx.Err() != nil:
		return fmt.Sprintf(mergePrefix+"%s timed out after %s, so origin's repository is unknown and nothing was merged: check "+
			"the repository's git config and %s", mergeGuaranteeGetURL, m.ghTimeout, m.again()), nil
	case err != nil:
		return fmt.Sprintf(mergePrefix+"%s failed (%s), so origin's repository is unknown and nothing was merged: ralph org "+
			"merge merges only into origin's repository, so add origin (git remote add origin <URL>) and %s",
			mergeGuaranteeGetURL, mergeGuaranteeErrText(err), m.again()), nil
	}
	host, repo, ok := mergeOriginFromURL(line)
	// A host, owner or repo that starts with "-" is refused as no host and
	// owner/repo: gh repo view takes <host>/<owner>/<repo> as an argument,
	// where gh would read a leading "-" as a flag. merge-order's reading of
	// origin does not check this.
	dashed := strings.HasPrefix(host, "-") || strings.HasPrefix(repo.owner, "-") || strings.HasPrefix(repo.name, "-")
	if !ok || dashed {
		return fmt.Sprintf(mergePrefix+"origin's URL names no host and owner/repo (a local path, file://, or a scheme other "+
			"than https, http, ssh and git), so there is no GitHub repository to merge in and nothing was merged: point origin "+
			"at the GitHub repository (git remote set-url origin <URL>) and %s", m.again()), nil
	}
	r.apiHost, r.repo = mergeGuaranteeAPIHost(host), repo
	r.repoArg = r.apiHost + "/" + repo.String()
	return "", nil
}

// listOrigin runs the ls-remote of steps 5 and 13 (listOriginBranches). Its
// failure is the refusal mergeLsRemoteRefusal words.
func (m *mergeCheck) listOrigin(ctx context.Context) (mergeRemote, string, error) {
	remote, err := listOriginBranches(ctx, MergeOrderParams{Dir: m.dir, LsRemoteTimeout: m.lsTimeout}, m.run)
	switch {
	case ctx.Err() != nil:
		return mergeRemote{}, "", mergeInterrupted(ctx, "the ls-remote ran")
	case err != nil:
		return mergeRemote{}, mergeLsRemoteRefusal(err), nil
	}
	return remote, "", nil
}

// mergeLsRemoteRefusal is the refusal for err, an error of
// listOriginBranches with ctx not done. Its text is written for merge-order,
// so the refusal swaps its prefix for merge's, "nothing was classified" for
// "nothing was merged", and names ralph org merge where it names merge-order
// as what reads origin. The rest, what failed and the next step, stays.
func mergeLsRemoteRefusal(err error) string {
	s := strings.TrimPrefix(err.Error(), "org: merge-order: ")
	s = strings.ReplaceAll(s, "nothing was classified", "nothing was merged")
	s = strings.ReplaceAll(s, "merge-order reads origin", "ralph org merge reads origin")
	return mergePrefix + s
}

// findPR is step 6 of checkMerge (D10): the branch's PR from gh pr list in
// r's repository (lookupMergePR with -R), set on t as merge-order sets a PR it
// uses, or the refusal of a gh that failed, of no PR, of a limit full of fork
// PRs, and of a PR whose url names another host or owner/repo than origin's.
func (m *mergeCheck) findPR(ctx context.Context, r mergeReady, t *mergeTarget) (string, error) {
	pr, limitFullOfForks, err := lookupMergePR(ctx, m.dir, m.run, m.ghTimeout, r.repoArg, m.branch)
	cmd := "gh pr list --head " + m.branch + " -R " + r.repoArg
	if refused, err := m.ghFailed(ctx, r, cmd, err, "the branch's PR is unknown"); err != nil || refused != "" {
		return refused, err
	}
	switch {
	case limitFullOfForks:
		return fmt.Sprintf(mergePrefix+"%s returned %d PRs, its limit, all from forks, so the branch's own PR may be past them "+
			"and nothing was merged: merge the branch's PR yourself", cmd, mergeOrderGHLimit), nil
	case pr == nil:
		return fmt.Sprintf(mergePrefix+"%s found no PR of the branch %s in origin's repository %s, so nothing was merged: open "+
			"one (gh pr create --head %s -R %s), and once its CI passed, %s", cmd, m.branch, r.repoArg, m.branch, r.repoArg,
			m.again()), nil
	case !r.isOriginsPR(pr.URL):
		return mergeForeignPRRefusal(cmd, r, pr.Number, pr.URL), nil
	}
	t.pr, t.prLookup = pr, MergeOrderPRLookupFound
	return "", nil
}

// mergeForeignPRRefusal is the refusal for PR number n with the url u, which
// cmd returned and which names another host or owner/repo than r's.
func mergeForeignPRRefusal(cmd string, r mergeReady, n int, u string) string {
	return fmt.Sprintf(mergePrefix+"%s returned PR #%d with the url %s, which is not a PR of origin's repository %s, so nothing "+
		"was merged: ralph org merge merges only origin's PRs; check origin (git remote get-url origin), and merge the PR "+
		"yourself if it is the one to merge", cmd, n, u, r.repoArg)
}

// ghFailed is the outcome of cmd, a gh call of checkMerge, when it returned
// err: the interruption when ctx is done; "" when err is nil; the refusal of a
// gh not on PATH (ghMissing); else the refusal of a gh that failed, saying what
// is unknown.
func (m *mergeCheck) ghFailed(ctx context.Context, r mergeReady, cmd string, err error, unknown string) (string, error) {
	switch {
	case ctx.Err() != nil:
		return "", mergeInterrupted(ctx, cmd+" ran")
	case err == nil:
		return "", nil
	case errors.Is(err, exec.ErrNotFound):
		return m.ghMissing(r, err), nil
	}
	return fmt.Sprintf(mergePrefix+"%s failed (%s), so %s and nothing was merged: %s", cmd, mergeGuaranteeErrText(err), unknown,
		m.ghAccess(r)), nil
}

// gh runs gh with args in m.dir with GH_PROMPT_DISABLED=1, bounded by
// m.ghTimeout, and returns its stdout. timedOut reports that the timeout
// stopped it, and err then says so.
func (m *mergeCheck) gh(ctx context.Context, args ...string) (out string, timedOut bool, err error) {
	return m.ghWith(ctx, m.run, args...)
}

// ghWith is gh run through run instead of m.run.
func (m *mergeCheck) ghWith(ctx context.Context, run MergeOrderExec, args ...string) (out string, timedOut bool, err error) {
	ghCtx, cancel := context.WithTimeout(ctx, m.ghTimeout)
	defer cancel()
	out, err = run(ghCtx, m.dir, []string{"GH_PROMPT_DISABLED=1"}, "gh", args...)
	if err != nil && ghCtx.Err() != nil && ctx.Err() == nil {
		return out, true, fmt.Errorf("timed out after %s: %w", m.ghTimeout, err)
	}
	return out, false, err
}

// mergeTargetOrg is the org a merge of t is recorded under (D4): the org of
// its latest binding, else of its latest gate record.
func mergeTargetOrg(t *mergeTarget) string {
	if t.binding != nil {
		return t.binding.OrgID
	}
	return t.gate.ev.OrgID
}

// runningRefusal is step 8 of checkMerge (D5): the refusal when orgID, t's
// org, is in running, "" otherwise. While the org runs, its leader may still
// push to the branch and open or change the PR, and it raises a RESULT when it
// is done and then disbands the org: kind is mergeRefusalOrgRunning, which
// --wait-checks waits out (D13). An org that runs another feature now
// (t.orgRunsOn) is refused too, with mergeRefusalOther: its leader works on
// another branch, and nothing says when that ends. t.orgRunsOn is set only
// when t has a binding (collectMergeTargets), so on a branch with a gate
// record and no binding, an org that runs another feature now is refused as
// one that still runs the branch's, with mergeRefusalOrgRunning, and
// --wait-checks waits for it (orgWaitEnd, docs/tech-debt/README.md).
func (m *mergeCheck) runningRefusal(t *mergeTarget, orgID string, running []string) (string, mergeRefusalKind) {
	switch {
	case !slices.Contains(running, orgID):
		return "", mergeRefusalOther
	case t.orgRunsOn != "":
		return fmt.Sprintf(mergePrefix+"the branch %s is bound to org_id %q, which is running, now for another feature on the "+
			"branch %s, so nothing was merged: wait for the org to be disbanded and %s", m.branch, orgID, t.orgRunsOn,
			m.again()), mergeRefusalOther
	}
	whose := "is bound to"
	if t.binding == nil {
		whose = "was gated by"
	}
	return fmt.Sprintf(mergePrefix+"the branch %s %s org_id %q, which is running, so its leader may still push to the branch "+
		"and open or change the PR, and nothing was merged: handle the org's RESULT in the inbox (ralph org inbox), and once "+
		"the org is disbanded, %s", m.branch, whose, orgID, m.again()), mergeRefusalOrgRunning
}

// ghPRView is the `gh pr view --json` output checkMerge reads.
// AutoMergeRequest is the JSON as gh printed it: null when the PR has no
// auto-merge request, nil when the field is missing.
type ghPRView struct {
	State            string          `json:"state"`
	URL              string          `json:"url"`
	HeadRefOid       string          `json:"headRefOid"`
	BaseRefName      string          `json:"baseRefName"`
	AutoMergeRequest json.RawMessage `json:"autoMergeRequest"`
}

// viewPR is step 9 of checkMerge: the PR as `gh pr view` reads it in r's
// repository. It refuses a gh that fails, an answer that is not the PR's JSON
// (a field empty or missing), and a PR that is not OPEN, whose head is not the
// gate's, whose base is not origin's default branch, whose url names another
// repository than origin's, or that has an auto-merge request (D12). It sets
// r.prURL.
func (m *mergeCheck) viewPR(ctx context.Context, r *mergeReady) (string, error) {
	n := strconv.Itoa(r.prNumber)
	cmd := "gh pr view " + n + " -R " + r.repoArg
	out, _, err := m.gh(ctx, "pr", "view", n, "-R", r.repoArg, "--json", mergePRViewFields)
	if refused, err := m.ghFailed(ctx, *r, cmd, err, "the PR's state is unknown"); err != nil || refused != "" {
		return refused, err
	}
	var pr ghPRView
	if err := json.Unmarshal([]byte(out), &pr); err != nil || pr.State == "" || pr.URL == "" || pr.HeadRefOid == "" ||
		pr.BaseRefName == "" || pr.AutoMergeRequest == nil {
		return fmt.Sprintf(mergePrefix+"%s printed output that is not the PR's JSON with %s, so nothing was merged: %s", cmd,
			mergePRViewFields, m.ghAccess(*r)), nil
	}
	switch {
	case pr.State != "OPEN":
		return fmt.Sprintf(mergePrefix+"PR #%d is %s, not OPEN, so nothing was merged: run ralph org merge-order to see the "+
			"branch's state", r.prNumber, pr.State), nil
	case pr.HeadRefOid != r.head:
		return fmt.Sprintf(mergePrefix+"PR #%d's head is %s, not %s, the head the gate passed, so nothing was merged: run ralph "+
			"org merge-order again and follow the branch's Detail", r.prNumber, shortMergeSHA(pr.HeadRefOid), shortMergeSHA(r.head)), nil
	case pr.BaseRefName != r.defaultBranch:
		return fmt.Sprintf(mergePrefix+"PR #%d merges into %s, not origin's default branch %s, so nothing was merged: change "+
			"its base (gh pr edit %d --base %s -R %s), and once its CI passed, %s", r.prNumber, pr.BaseRefName, r.defaultBranch,
			r.prNumber, r.defaultBranch, r.repoArg, m.again()), nil
	case !r.isOriginsPR(pr.URL):
		return mergeForeignPRRefusal(cmd, *r, r.prNumber, pr.URL), nil
	case string(bytes.TrimSpace(pr.AutoMergeRequest)) != "null":
		return fmt.Sprintf(mergePrefix+"PR #%d has an auto-merge request already, which GitHub would carry out apart from ralph "+
			"org merge, so nothing was merged: remove it (gh pr merge %d --disable-auto -R %s) and %s", r.prNumber, r.prNumber,
			r.repoArg, m.again()), nil
	}
	r.prURL = pr.URL
	return "", nil
}

// ghPRCheck is one element of `gh pr checks --json` output.
type ghPRCheck struct {
	Bucket   string `json:"bucket"`
	Name     string `json:"name"`
	State    string `json:"state"`
	Workflow string `json:"workflow"`
}

// label names c in a refusal: `<workflow> / <name>`, or the name alone, with
// its bucket and state.
func (c ghPRCheck) label() string {
	name := cmp.Or(c.Name, "(no name)")
	if c.Workflow != "" && c.Workflow != c.Name {
		name = c.Workflow + " / " + name
	}
	return fmt.Sprintf("%s (%s, %s)", name, cmp.Or(c.Bucket, "no bucket"), cmp.Or(c.State, "no state"))
}

// mergeRefusalKind says which refusal of checkMerge a refusal is, as far as
// the wait of --wait-checks (D5, D13, merge_wait.go) needs to know: the one
// of step 8 for an org that still runs the branch's feature (on a branch with
// no binding, also for its gate's org that runs another feature now:
// runningRefusal), and those of readChecks (step 10), so that the wait is
// decided on the kind and not on the refusal's text.
type mergeRefusalKind int

const (
	// mergeRefusalOther: a refusal of another step, of step 8 for an org
	// that runs another feature now (mergeTarget.orgRunsOn), or no refusal.
	mergeRefusalOther mergeRefusalKind = iota
	// mergeRefusalOrgRunning: step 8 refused because the branch's org runs,
	// and not another feature now as far as the ledger tells, which it does
	// only for a branch with a binding (runningRefusal): its leader disbands
	// it after its RESULT.
	mergeRefusalOrgRunning
	// mergeRefusalChecksPending: every check is pass, skipping, or pending,
	// and one is pending.
	mergeRefusalChecksPending
	// mergeRefusalChecksZero: gh pr checks reported no checks, as [] or as
	// mergeNoChecksSentence.
	mergeRefusalChecksZero
	// mergeRefusalChecksNotPassed: a check is fail, cancel, or a bucket
	// ralph does not know, or every check is skipping.
	mergeRefusalChecksNotPassed
	// mergeRefusalChecksUnread: gh pr checks failed, timed out, was not
	// found, or printed no JSON list.
	mergeRefusalChecksUnread
)

// waits reports whether --wait-checks waits out a refusal of kind k: only an
// org that still runs the branch's feature (on a branch with no binding, also
// its gate's org that runs another feature now: runningRefusal), which its
// leader disbands after its RESULT, and checks that are pending, or none
// reported yet, which a CI that runs may turn into checks that passed.
// mergeWait.wait has a case for each kind it reports.
func (k mergeRefusalKind) waits() bool {
	return k == mergeRefusalOrgRunning || k == mergeRefusalChecksPending || k == mergeRefusalChecksZero
}

// readChecks is step 10 of checkMerge (D7): the PR's checks from `gh pr
// checks`, which exits 8 while a check is pending and may exit non-zero in
// other cases with the JSON on stdout, so stdout is read whatever the exit;
// only a timeout, a gh not found, and a stdout that is not a JSON list make
// the error count, except that a gh whose error has mergeNoChecksSentence for
// the branch reported no checks. It refuses unless every check's bucket is
// pass or skipping and one is pass, and kind says why (mergeRefusalKind). The
// wait of --wait-checks reads the checks with it too.
func (m *mergeCheck) readChecks(ctx context.Context, r mergeReady) (refused string, kind mergeRefusalKind, err error) {
	n := strconv.Itoa(r.prNumber)
	cmd := "gh pr checks " + n + " -R " + r.repoArg
	out, timedOut, err := m.gh(ctx, "pr", "checks", n, "-R", r.repoArg, "--json", mergePRChecksFields)
	var checks []ghPRCheck
	listed := strings.HasPrefix(strings.TrimSpace(out), "[") && json.Unmarshal([]byte(out), &checks) == nil
	switch {
	case ctx.Err() != nil || errors.Is(err, exec.ErrNotFound) || timedOut:
		refused, err = m.ghFailed(ctx, r, cmd, err, "the PR's checks are unknown")
		return refused, mergeRefusalChecksUnread, err
	case err != nil && !listed && strings.Contains(err.Error(), fmt.Sprintf(mergeNoChecksSentence, r.branch)):
		checks = nil
	case err != nil && !listed:
		refused, err = m.ghFailed(ctx, r, cmd, err, "the PR's checks are unknown")
		return refused, mergeRefusalChecksUnread, err
	case !listed:
		return fmt.Sprintf(mergePrefix+"%s printed output that is not a JSON list of checks, so the PR's checks are unknown and "+
			"nothing was merged: %s", cmd, m.ghAccess(r)), mergeRefusalChecksUnread, nil
	}
	var passed int
	var others []string // every check that is not pass or skipping
	var ended bool      // one of them is not pending either
	for _, c := range checks {
		switch c.Bucket {
		case "pass":
			passed++
		case "skipping":
		case "pending":
			others = append(others, c.label())
		default:
			ended = true
			others = append(others, c.label())
		}
	}
	switch {
	case len(checks) == 0:
		return fmt.Sprintf(mergePrefix+"%s reported no checks, so nothing shows that the PR's CI passed and nothing was merged: "+
			"let CI run on the PR, and once it passed, %s", cmd, m.again()), mergeRefusalChecksZero, nil
	case len(others) > 0:
		kind = mergeRefusalChecksPending
		if ended {
			kind = mergeRefusalChecksNotPassed
		}
		return fmt.Sprintf(mergePrefix+"%d of the %d checks of PR #%d are not pass or skipping: %s, so nothing was merged: wait "+
			"for the pending ones and fix the others, and once all passed, %s", len(others), len(checks), r.prNumber,
			mergeNamedBranches(others), m.again()), kind, nil
	case passed == 0:
		return fmt.Sprintf(mergePrefix+"every one of the %d checks of PR #%d is skipping and none passed, so nothing shows that "+
			"the PR's CI passed and nothing was merged: let a check run on the PR, and once it passed, %s", len(checks),
			r.prNumber, m.again()), mergeRefusalChecksNotPassed, nil
	}
	return "", mergeRefusalOther, nil
}

// readMethod is step 12 of checkMerge (D2): the merge method `gh repo view`
// reports for the caller (viewerDefaultMergeMethod) in r's repository, which
// it names as its argument (gh repo view has no -R), into r. It refuses a gh
// that fails, an answer that is not that JSON, and a method not in
// mergeMethodFlags, empty included.
func (m *mergeCheck) readMethod(ctx context.Context, r *mergeReady) (string, error) {
	cmd := "gh repo view " + r.repoArg
	out, _, err := m.gh(ctx, "repo", "view", r.repoArg, "--json", "viewerDefaultMergeMethod")
	if refused, err := m.ghFailed(ctx, *r, cmd, err, "the merge method is unknown"); err != nil || refused != "" {
		return refused, err
	}
	var repo struct {
		Method string `json:"viewerDefaultMergeMethod"`
	}
	if err := json.Unmarshal([]byte(out), &repo); err != nil {
		return fmt.Sprintf(mergePrefix+"%s printed output that is not the repository's JSON (%s), so the merge method is unknown "+
			"and nothing was merged: %s", cmd, mergeGuaranteeErrText(err), m.ghAccess(*r)), nil
	}
	flag, known := mergeMethodFlags[repo.Method]
	if !known {
		return fmt.Sprintf(mergePrefix+"%s reported the merge method %q, which ralph org merge does not know (it merges with "+
			"MERGE, SQUASH or REBASE), so nothing was merged: merge the PR yourself", cmd, repo.Method), nil
	}
	r.method, r.methodFlag = strings.ToLower(repo.Method), flag
	return "", nil
}

// checkOriginUnmoved is step 13 of checkMerge (D8): a second ls-remote,
// right before the merge, must find origin's default branch with the same
// name, its tip still the gate's base, and the branch's tip still the gate's
// head. It is an early refusal: GitHub's up-to-date rule (D11) refuses a
// merge whose base moved after it.
func (m *mergeCheck) checkOriginUnmoved(ctx context.Context, r mergeReady) (string, error) {
	remote, refused, err := m.listOrigin(ctx)
	if err != nil || refused != "" {
		return refused, err
	}
	tip, onOrigin := remote.heads[m.branch]
	var moved string
	switch {
	case remote.defaultBranch != r.defaultBranch:
		moved = fmt.Sprintf("origin's default branch is now %s, not %s", remote.defaultBranch, r.defaultBranch)
	case remote.defaultHead != r.base:
		moved = fmt.Sprintf("origin/%s moved to %s from %s, the gate's base", r.defaultBranch, shortMergeSHA(remote.defaultHead),
			shortMergeSHA(r.base))
	case !onOrigin:
		moved = fmt.Sprintf("origin has no branch %s any more", m.branch)
	case tip != r.head:
		moved = fmt.Sprintf("origin's %s moved to %s from %s, the head the gate passed", m.branch, shortMergeSHA(tip),
			shortMergeSHA(r.head))
	default:
		return "", nil
	}
	return fmt.Sprintf(mergePrefix+"origin moved while the checks ran: %s, so nothing was merged: run ralph org merge-order "+
		"again and follow the branch's Detail", moved), nil
}
