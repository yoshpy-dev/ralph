package org

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Origin's GitHub repository, which ralph org merge pins every gh call to
// (D10 of docs/plans/active/2026-10-17-org-auto-merge.md, merge.go) and ralph
// org cleanup its PR lookup (MergeOrderParams.Origin, S-X1 of
// docs/plans/active/2026-10-28-org-merge-cleanup.md): ReadMergeOrigin reads it
// from `git remote get-url origin`, and MergeOrigin.hasPR tells a PR of it by
// the PR's url.

// MergeOrigin is origin's GitHub repository: the host gh reads it on (origin's
// host, ssh.github.com read as github.com: mergeGuaranteeAPIHost) and its
// owner/repo. The zero MergeOrigin names none. Outside this package only
// ReadMergeOrigin makes one.
type MergeOrigin struct {
	apiHost string
	repo    mergeRepo
}

// Arg is o as gh pr's -R and gh repo view's repository argument take it,
// <host>/<owner>/<repo>; "" for the zero MergeOrigin.
func (o MergeOrigin) Arg() string {
	if o == (MergeOrigin{}) {
		return ""
	}
	return o.apiHost + "/" + o.repo.String()
}

// hasPR reports whether u is the url of a PR of o's repository: its host is
// o's and its owner/repo is o's, without case (mergeHostRepoFromPRURL).
func (o MergeOrigin) hasPR(u string) bool {
	host, repo, ok := mergeHostRepoFromPRURL(u)
	return ok && strings.EqualFold(host, o.apiHost) && repo.same(o.repo)
}

// foreignPRWhy is "" when pr, the PR `gh pr list --head <branch> -R
// <o.Arg()>` returned, is a PR of o's repository (hasPR), else why it is not
// used, which says to check origin, as ralph org merge's refusal of such a PR
// does (mergeForeignPRRefusal): gh answered, so the same lookup run again
// would return the same PR.
func (o MergeOrigin) foreignPRWhy(branch string, pr *MergeOrderPR) string {
	if o.hasPR(pr.URL) {
		return ""
	}
	return fmt.Sprintf("gh pr list --head %s -R %s returned PR #%d with the url %q, which is not a PR of origin's repository %s; "+
		"check origin (git remote get-url origin)", branch, o.Arg(), pr.Number, pr.URL, o.Arg())
}

// MergeOriginProblem is why ReadMergeOrigin read no repository of origin.
type MergeOriginProblem int

const (
	// MergeOriginRead: origin's URL names a repository, which ReadMergeOrigin
	// returns.
	MergeOriginRead MergeOriginProblem = iota
	// MergeOriginInterrupted: ctx was done when the get-url returned.
	MergeOriginInterrupted
	// MergeOriginTimedOut: the get-url timed out.
	MergeOriginTimedOut
	// MergeOriginGetURLFailed: the get-url failed, for example with no remote
	// origin.
	MergeOriginGetURLFailed
	// MergeOriginNoRepository: origin's URL names no host and owner/repo (a
	// local path, file://, or a scheme other than https, http, ssh and git),
	// or a host, owner or repo that starts with "-".
	MergeOriginNoRepository
)

// ReadMergeOrigin runs `git remote get-url origin` in dir (empty means the
// current directory) through run (nil means realMergeOrderExec), with
// GIT_TERMINAL_PROMPT=0 and bounded by timeout (zero or less means 15 s), and
// reads origin's repository from the first line it prints
// (mergeOriginFromURL). It returns the repository with MergeOriginRead, or the
// zero MergeOrigin with the first problem that applies: ctx done, the get-url
// timed out, it failed, or the URL names no repository. getURLErr is the
// get-url's error with MergeOriginGetURLFailed, and with MergeOriginTimedOut
// the same after "timed out after <timeout>: "; nil otherwise.
//
// A host, owner or repo that starts with "-" names no repository: gh repo
// view takes <host>/<owner>/<repo> as an argument, where gh would read a
// leading "-" as a flag. merge-order's own reading of origin
// (mergeOriginRepo) does not check this.
func ReadMergeOrigin(ctx context.Context, dir string, run MergeOrderExec, timeout time.Duration) (o MergeOrigin, problem MergeOriginProblem, getURLErr error) {
	if run == nil {
		run = realMergeOrderExec
	}
	if timeout <= 0 {
		timeout = defaultMergeOrderGHTimeout
	}
	getCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	line, err := getMergeOriginURL(getCtx, dir, run)
	switch {
	case ctx.Err() != nil:
		return MergeOrigin{}, MergeOriginInterrupted, nil
	case err != nil && getCtx.Err() != nil:
		return MergeOrigin{}, MergeOriginTimedOut, fmt.Errorf("timed out after %s: %w", timeout, err)
	case err != nil:
		return MergeOrigin{}, MergeOriginGetURLFailed, err
	}
	host, repo, ok := mergeOriginFromURL(line)
	if !ok || strings.HasPrefix(host, "-") || strings.HasPrefix(repo.owner, "-") || strings.HasPrefix(repo.name, "-") {
		return MergeOrigin{}, MergeOriginNoRepository, nil
	}
	return MergeOrigin{apiHost: mergeGuaranteeAPIHost(host), repo: repo}, MergeOriginRead, nil
}
