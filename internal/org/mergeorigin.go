package org

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Origin's GitHub repository, which ralph org merge pins every gh call to
// (D10 of docs/plans/active/2026-10-17-org-auto-merge.md, merge.go) and ralph
// org cleanup its PR lookup (MergeOrderParams.Origin, S-X1 of
// docs/plans/active/2026-10-28-org-merge-cleanup.md): ReadMergeOrigin reads it
// from `git remote get-url origin`, and MergeOrigin.hasPR tells a PR of it by
// the PR's url. The host of an ssh origin (ssh:// or scp-like) is resolved
// with `ssh -G <host>` the way gh resolves a remote's (resolveMergeAPIHost,
// plan docs/plans/active/2026-11-19-org-ssh-alias-host.md), so an ssh alias
// (a Host in ~/.ssh/config) is read on the host it names. The host passed to
// ssh -G is origin's as its URL writes it, case kept (mergeSSHHost), as git
// and gh pass it, since OpenSSH matches a Host with case. ssh -G connects to
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
// the repository on with `ssh -G <host>` in dir, <host> as origin's URL
// writes it (mergeSSHHost), through sshRun, bounded by
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
// origin's URL originURL and its host as mergeOriginFromURL read it
// (lowercased), the way gh resolves a remote's host (go-gh's pkg/ssh
// Translator): for an ssh URL, the scp-like form or the scheme ssh, it runs
// `ssh -G <host>` in dir through sshRun (nil means realMergeDetachedExec),
// bounded by timeout (zero or less means mergeSSHResolveTimeout) under ctx,
// and takes the value of the last line of its output whose first word is
// "hostname", lowercased; for https, http and git:// it runs nothing and
// takes host. The <host> of ssh -G is origin's host as originURL writes it,
// case kept (mergeSSHHost): OpenSSH matches a Host of the ssh config with
// case, and git and gh look the ssh config up with the host as origin writes
// it, so a lowercased host could read another Host than the one git connects
// with, or none. ssh.github.com is then read as github.com
// (mergeGuaranteeAPIHost), after the resolution as gh reads it, since an
// alias may name ssh.github.com.
//
// unresolvedWhy is "" when apiHost is resolved or the URL is not an ssh one.
// Otherwise apiHost is host (mergeGuaranteeAPIHost), lowercased, and
// unresolvedWhy a clause that names the command, with the host as ssh -G
// gets it, and says why its answer was not used: a host that is no plain host
// name (mergeSSHHostName: no ssh runs), an ssh -G that failed (ssh not
// installed included), timed out, or was stopped by ctx, output with no
// hostname line, an empty hostname, or a hostname that is no plain host name,
// which gh's -R <host>/<owner>/<repo> could misread. For a host or a hostname
// that is no plain host name the clause quotes it and ends with
// mergeSSHHostNameWhy, the same words for both. ssh -G runs exactly when
// mergeSSHResolveRuns says so, which doctor's readOrigin reads too. With ctx
// done the caller tells the interruption (ReadMergeOrigin).
func resolveMergeAPIHost(ctx context.Context, dir string, sshRun MergeOrderExec, timeout time.Duration, originURL, host string) (apiHost, unresolvedWhy string) {
	fallback := mergeGuaranteeAPIHost(host)
	if !mergeOriginIsSSH(originURL) {
		return fallback, ""
	}
	sshHost := mergeSSHHost(originURL, host)
	if !mergeSSHResolveRuns(originURL, host) {
		return fallback, fmt.Sprintf("ssh -G was not run for the host %q, %s", sshHost, mergeSSHHostNameWhy(sshHost))
	}
	if sshRun == nil {
		sshRun = realMergeDetachedExec
	}
	if timeout <= 0 {
		timeout = mergeSSHResolveTimeout
	}
	cmd := "ssh -G " + sshHost
	sshCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out, err := sshRun(sshCtx, dir, nil, "ssh", "-G", sshHost)
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
	case !mergeSSHHostName(resolved):
		return fallback, fmt.Sprintf("%s printed the hostname %q, %s", cmd, resolved, mergeSSHHostNameWhy(resolved))
	}
	return mergeGuaranteeAPIHost(strings.ToLower(resolved)), ""
}

// mergeSSHResolveRuns reports whether resolveMergeAPIHost runs ssh -G for
// origin's URL originURL and its host as mergeOriginFromURL read it: the URL
// is an ssh one (mergeOriginIsSSH) and the host ssh -G would get
// (mergeSSHHost) a plain host name (mergeSSHHostName). doctor's readOrigin
// asks it whether an ssh -G ran before it names one as what ran when the
// check's ctx is done, so the two cannot disagree.
func mergeSSHResolveRuns(originURL, host string) bool {
	return mergeOriginIsSSH(originURL) && mergeSSHHostName(mergeSSHHost(originURL, host))
}

// mergeSSHHost is the host resolveMergeAPIHost passes to ssh -G for origin's
// URL originURL and host, its host as mergeOriginFromURL read it
// (lowercased): origin's host as originURL writes it, case kept
// (mergeOriginFromURLAsWritten), since OpenSSH matches a Host pattern of the
// ssh config with case (`ssh -G Work-GH` reads `Host Work-GH`, `ssh -G
// work-gh` does not) and git and gh (go-gh's Translator takes the URL's
// Hostname as written) look the ssh config up with the host as origin writes
// it. It is host when originURL names no host that equals host without case,
// which a caller that read host from originURL never sees.
func mergeSSHHost(originURL, host string) string {
	written, _, ok := mergeOriginFromURLAsWritten(originURL)
	if !ok || !strings.EqualFold(written, host) {
		return host
	}
	return written
}

// mergeSSHHostName reports whether s is a plain host name, the only host
// ralph runs ssh -G for and takes from the hostname line ssh -G prints
// (resolveMergeAPIHost): not empty, not starting with "-", which ssh would
// read as an option and gh as a flag, and made only of the ASCII letters and
// digits, ".", "_", ":" (an IPv6 address) and "-". Every other byte is
// refused: origin's URL can name a host such as $(id) or a;touch, which a
// `Match exec` line of the ssh config with %h would hand to a shell, and gh's
// -R <host>/<owner>/<repo> would misread white space, a control character,
// "/", or invalid UTF-8. A host that is not ASCII is refused too.
func mergeSSHHostName(s string) bool {
	if s == "" || strings.HasPrefix(s, "-") {
		return false
	}
	for _, c := range []byte(s) {
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9', c == '.', c == '_', c == ':', c == '-':
		default:
			return false
		}
	}
	return true
}

// mergeSSHHostNameWhy ends the clause resolveMergeAPIHost gives for s, a host
// it ran no ssh -G for or a hostname ssh -G printed, that mergeSSHHostName
// refuses: it says what ralph refuses s for, with the same words for the
// host and the hostname. It names ralph's rule, not gh's, since gh does not
// check the characters itself.
func mergeSSHHostNameWhy(s string) string {
	if strings.HasPrefix(s, "-") {
		return `which starts with "-"`
	}
	return `which is not a plain host name (only ASCII letters, digits, ".", "_", ":" and "-")`
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
