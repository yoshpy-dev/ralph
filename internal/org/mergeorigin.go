package org

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Origin's GitHub repository, which ralph org merge pins every gh call to
// (D10 of docs/plans/active/2026-10-17-org-auto-merge.md, merge.go) and ralph
// org cleanup its PR lookup (MergeOrderParams.Origin, S-X1 of
// docs/plans/active/2026-10-28-org-merge-cleanup.md): ReadMergeOrigin reads it
// from `git remote get-url origin`, and MergeOrigin.hasPR tells a PR of it by
// the PR's url. The host of an ssh origin (ssh:// or scp-like) is resolved
// with `ssh -G <host>` the way gh resolves a remote's (resolveMergeAPIHost,
// plan docs/plans/active/2026-11-19-org-ssh-alias-host.md), so an ssh alias
// (a Host in ~/.ssh/config) is read on the host it names. ssh -G connects to
// nothing, but it runs the commands of any `Match exec` in the ssh config, as
// gh's own resolution does.

// mergeSSHResolveTimeout bounds the `ssh -G <host>` that resolves an ssh
// origin's host (resolveMergeAPIHost) apart from the get-url and the gh
// calls, so a `Match exec` command that does not return costs at most this.
const mergeSSHResolveTimeout = 5 * time.Second

// MergeOrigin is origin's GitHub repository: the host gh reads it on
// (apiHost: for an ssh origin the hostname `ssh -G` gives for origin's host,
// else origin's host, with ssh.github.com read as github.com:
// resolveMergeAPIHost) and its owner/repo. originHost is origin's host as
// its URL names it, lowercased, and unresolvedWhy why ssh -G did not resolve
// it, "" when it did or origin's URL is not an ssh one: the refusal of a gh
// that fails names the ssh key fix from them (mergeSSHAliasHint). The zero
// MergeOrigin names none. Outside this package only ReadMergeOrigin makes
// one.
type MergeOrigin struct {
	apiHost       string
	repo          mergeRepo
	originHost    string
	unresolvedWhy string
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
	// MergeOriginInterrupted: ctx was done when the get-url or the ssh -G
	// that resolves an ssh origin's host returned.
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
// (mergeOriginFromURL). For an ssh origin it then resolves the host gh reads
// the repository on with `ssh -G <host>` in dir through sshRun, bounded by
// mergeSSHResolveTimeout (resolveMergeAPIHost); nil means
// realMergeDetachedExec, whose ctx done kills ssh's whole process group, the
// commands of a `Match exec` with it. An ssh -G that fails leaves origin's
// host as it is, and the MergeOrigin keeps why. It returns the repository
// with MergeOriginRead, or the zero MergeOrigin with the first problem that
// applies: ctx done (after the get-url or the ssh -G), the get-url timed out,
// it failed, or the URL names no repository. getURLErr is the get-url's error
// with MergeOriginGetURLFailed, and with MergeOriginTimedOut the same after
// "timed out after <timeout>: "; nil otherwise.
//
// A host, owner or repo that starts with "-" names no repository: gh repo
// view takes <host>/<owner>/<repo> as an argument, where gh would read a
// leading "-" as a flag, and ssh would read such a host as an option, so no
// ssh -G runs for it. merge-order's own reading of origin (mergeOriginRepo)
// does not check this.
func ReadMergeOrigin(ctx context.Context, dir string, run, sshRun MergeOrderExec, timeout time.Duration) (o MergeOrigin, problem MergeOriginProblem, getURLErr error) {
	if run == nil {
		run = realMergeOrderExec
	}
	if sshRun == nil {
		sshRun = realMergeDetachedExec
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
	apiHost, why := resolveMergeAPIHost(ctx, dir, sshRun, mergeSSHResolveTimeout, line, host)
	if ctx.Err() != nil {
		return MergeOrigin{}, MergeOriginInterrupted, nil
	}
	return MergeOrigin{apiHost: apiHost, repo: repo, originHost: host, unresolvedWhy: why}, MergeOriginRead, nil
}

// resolveMergeAPIHost is the host gh reads origin's repository on, for
// origin's URL originURL and its host as mergeOriginFromURL read it, the way
// gh resolves a remote's host (go-gh's pkg/ssh Translator): for an ssh URL,
// the scp-like form or the scheme ssh, it runs `ssh -G <host>` in dir
// through sshRun (nil means realMergeDetachedExec), bounded by timeout (zero
// or less means mergeSSHResolveTimeout) under ctx, and takes the value of the
// last line of its output whose first word is "hostname", lowercased; for
// https, http and git:// it runs nothing and takes host. ssh.github.com is
// then read as github.com (mergeGuaranteeAPIHost), after the resolution as gh
// reads it, since an alias may name ssh.github.com.
//
// unresolvedWhy is "" when apiHost is resolved or the URL is not an ssh one.
// Otherwise apiHost is host (mergeGuaranteeAPIHost) and unresolvedWhy a
// clause that names the command and says why its answer was not used: a host
// that starts with "-" (no ssh runs, as ssh would read it as an option), an
// ssh -G that failed (ssh not installed included), timed out, or was stopped
// by ctx, output with no hostname line, or a hostname that is empty, starts
// with "-", or holds white space, a control character, "/" or invalid UTF-8,
// which gh's -R <host>/<owner>/<repo> would misread. gh itself does not check
// the hostname's characters. With ctx done the caller tells the interruption
// (ReadMergeOrigin).
func resolveMergeAPIHost(ctx context.Context, dir string, sshRun MergeOrderExec, timeout time.Duration, originURL, host string) (apiHost, unresolvedWhy string) {
	fallback := mergeGuaranteeAPIHost(host)
	if !mergeOriginIsSSH(originURL) {
		return fallback, ""
	}
	if strings.HasPrefix(host, "-") {
		return fallback, fmt.Sprintf("ssh -G was not run for the host %s, which starts with \"-\"", host)
	}
	if sshRun == nil {
		sshRun = realMergeDetachedExec
	}
	if timeout <= 0 {
		timeout = mergeSSHResolveTimeout
	}
	cmd := "ssh -G " + host
	sshCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out, err := sshRun(sshCtx, dir, nil, "ssh", "-G", host)
	switch {
	case err != nil && ctx.Err() != nil:
		return fallback, fmt.Sprintf("%s was interrupted (%v)", cmd, context.Cause(ctx))
	case err != nil && sshCtx.Err() != nil:
		return fallback, fmt.Sprintf("%s timed out after %s", cmd, timeout)
	case err != nil:
		return fallback, fmt.Sprintf("%s failed (%s)", cmd, mergeGuaranteeErrText(err))
	}
	resolved, found := "", false
	for line := range strings.Lines(out) {
		if key, value, ok := strings.Cut(line, " "); ok && key == "hostname" {
			resolved, found = strings.TrimSpace(value), true
		}
	}
	switch {
	case !found:
		return fallback, cmd + " printed no hostname line"
	case resolved == "":
		return fallback, cmd + " printed an empty hostname"
	case strings.HasPrefix(resolved, "-") || !utf8.ValidString(resolved) || strings.ContainsFunc(resolved, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r) || r == '/'
	}):
		return fallback, fmt.Sprintf("%s printed the hostname %q, which gh cannot take as a host", cmd, resolved)
	}
	return mergeGuaranteeAPIHost(strings.ToLower(resolved)), ""
}

// mergeOriginIsSSH reports whether originURL, a URL mergeOriginFromURL reads
// a repository from, is an ssh one: the scp-like form (no "://") or a URL
// whose scheme is ssh.
func mergeOriginIsSSH(originURL string) bool {
	if !strings.Contains(originURL, "://") {
		return true
	}
	u, err := url.Parse(originURL)
	return err == nil && u.Scheme == "ssh"
}

// mergeSSHAliasHint is the clause doctor and ralph org merge add to the next
// step of a gh call that failed when ssh -G did not resolve origin's host
// (why, resolveMergeAPIHost's unresolvedWhy) and gh read originHost as it is
// (apiHost): if originHost is an ssh alias, gh cannot reach it, so the clause
// says to point origin at the real host and to choose the ssh key for the
// repository with core.sshCommand. It is "" when why is "" or apiHost is
// github.com, which is no alias.
func mergeSSHAliasHint(originHost, apiHost, why string) string {
	if why == "" || apiHost == "github.com" {
		return ""
	}
	return fmt.Sprintf("; %s, so gh read the host %s as origin names it: if %s is an ssh alias (a Host in ~/.ssh/config), "+
		"point origin at the real host (git remote set-url origin <URL>) and choose the ssh key for this repository with "+
		"git config core.sshCommand 'ssh -i <key> -o IdentitiesOnly=yes'", why, apiHost, originHost)
}
