package org

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"
)

// The checks of `ralph org merge` at the org layer (plan
// docs/plans/active/2026-10-17-org-auto-merge.md, S3a: the org-layer half of
// AC3 to AC5): checkMerge on manifest fixtures, with a fake MergeOrderExec
// standing in for git and gh, so no test runs gh or reaches a network.

// The keys mxExec answers by (mxKey).
const (
	mxGetURL   = "git remote get-url origin"
	mxLsRemote = "git ls-remote"
	mxPRList   = "gh pr list"
	mxPRView   = "gh pr view"
	mxPRChecks = "gh pr checks"
	mxRepoAPI  = "gh api repos/o/r"
	mxQueue    = "gh api graphql"
	mxRules    = "gh api repos/o/r/rules/branches/main"
	mxRuleset  = "gh api repos/o/r/rulesets/1"
	mxRepoView = "gh repo view"
)

// mxAll are the commands checkMerge runs when every check passes, in order.
var mxAll = []string{mxGetURL, mxLsRemote, mxPRList, mxPRView, mxPRChecks, mxRepoAPI, mxQueue, mxRules, mxRuleset, mxRepoView, mxLsRemote}

// Parts of the refusals of the default fixture.
const (
	mxRepoArg = "github.com/o/r"
	mxAgain   = "run ralph org merge --branch feat/a again"
	mxAccess  = "check that gh can reach github.com and is logged in to it (gh auth status --hostname github.com), and " + mxAgain
	mxProtect = "org: merge: GitHub's protection of origin's default branch was not confirmed, so nothing was merged: "
	mxSubject = "main, the default branch of origin's repository o/r"
)

// mxExec is a fake MergeOrderExec for checkMerge. Every call is recorded with
// its key (mxKey) and its ctx's deadline. `git remote get-url origin`
// answers originErr, else origin; the i-th ls-remote answers ls[i], the last
// one again after them; gh answers the reply under its key, and fails when it
// has none. With upstream, gh pr and gh repo view without origin's -R fail,
// as gh would read the upstream remote then. The call whose key is block
// waits for its ctx instead. after runs once the n-th call (from 0) has its
// answer, before it returns.
type mxExec struct {
	mu        sync.Mutex
	calls     []mxCall
	origin    string
	originErr error
	ls        []mgReply
	lsCalls   int
	replies   map[string]mgReply
	upstream  bool
	block     string
	after     func(n int)
}

type mxCall struct {
	key      string
	dir      string
	env      []string
	name     string
	args     []string
	deadline time.Time // ctx's, zero when it has none
}

// mxKey is the key of a call: the git command, `gh api <endpoint>`, or `gh
// <command> <subcommand>`.
func mxKey(name string, args []string) string {
	switch {
	case name == "git" && len(args) > 0 && args[0] == "ls-remote":
		return mxLsRemote
	case name == "gh" && len(args) > 0 && args[0] == "api":
		return "gh api " + args[len(args)-1]
	case name == "gh" && len(args) > 1:
		return "gh " + args[0] + " " + args[1]
	}
	return strings.Join(append([]string{name}, args...), " ")
}

func (e *mxExec) run(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	key := mxKey(name, args)
	deadline, _ := ctx.Deadline()
	e.mu.Lock()
	n := len(e.calls)
	e.calls = append(e.calls, mxCall{key: key, dir: dir, env: slices.Clone(env), name: name, args: slices.Clone(args), deadline: deadline})
	var r mgReply
	known := true
	switch key {
	case mxGetURL:
		r = mgReply{out: e.origin, err: e.originErr}
	case mxLsRemote:
		r = e.ls[min(e.lsCalls, len(e.ls)-1)]
		e.lsCalls++
	default:
		r, known = e.replies[key]
	}
	after := e.after
	e.mu.Unlock()
	if after != nil {
		defer after(n)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if key == e.block {
		<-ctx.Done()
		return "", ctx.Err()
	}
	if e.upstream && name == "gh" && args[0] != "api" && !slices.Contains(args, mxRepoArg) {
		return "", errors.New("mxExec: gh read the upstream remote")
	}
	if !known {
		return "", fmt.Errorf("mxExec: unexpected %s %q", name, args)
	}
	return r.out, r.err
}

// keys are the keys of the calls, in order.
func (e *mxExec) keys() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []string
	for _, c := range e.calls {
		out = append(out, c.key)
	}
	return out
}

func (e *mxExec) snapshot() []mxCall {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.calls)
}

// mxListing is origin with main at moMain and feat/a at moHeadA, where the
// gate left them.
func mxListing() string { return moListing("main", "main", moMain, "feat/a", moHeadA) }

// mxPRViewJSON is a `gh pr view --json` answer; autoMerge is the
// autoMergeRequest as JSON.
func mxPRViewJSON(state, head, base, url, autoMerge string) string {
	return fmt.Sprintf(`{"autoMergeRequest":%s,"baseRefName":%q,"headRefOid":%q,"mergeCommit":null,"state":%q,"url":%q}`,
		autoMerge, base, head, state, url)
}

// mxChecks is a `gh pr checks --json` answer.
func mxChecks(checks ...ghPRCheck) string { return mgJSON(append([]ghPRCheck{}, checks...)) }

// mxPass is a check of the workflow ci that passed.
func mxPass(name string) ghPRCheck {
	return ghPRCheck{Bucket: "pass", Name: name, State: "SUCCESS", Workflow: "ci"}
}

// newMXExec answers every check of the branch feat/a of o/r on github.com as
// one that passes: PR #7, open, at moHeadA into main, with the check verify
// passed, a ruleset (id 1) the caller cannot bypass, no merge queue, and the
// merge method MERGE.
func newMXExec() *mxExec {
	return &mxExec{origin: moOriginURL + "\n", ls: []mgReply{{out: mxListing()}}, replies: map[string]mgReply{
		mxPRList:   {out: moPRs(moPR(7, "OPEN", "main"))},
		mxPRView:   {out: mxPRViewJSON("OPEN", moHeadA, "main", "https://github.com/o/r/pull/7", "null")},
		mxPRChecks: {out: mxChecks(mxPass("verify"))},
		mxRepoAPI:  {out: meRepoJSON("main", "o/r", false, false)},
		mxQueue:    {out: meQueueJSON("null")},
		mxRules:    {out: mgRules(meChecksRule(1, true, "verify"))},
		mxRuleset:  {out: meRuleset(1, "protect main", "never")},
		mxRepoView: {out: `{"viewerDefaultMergeMethod":"MERGE"}`},
	}}
}

// mxReadyLedger binds feat/a to org-a, records a gate_passed of moHeadA on
// moMain, and disbands org-a.
func mxReadyLedger() []any {
	return []any{moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain), moDisbanded("org-a")}
}

// mxFixture is a ledger in a temp state dir and the fake exec checkMerge runs
// through.
type mxFixture struct {
	mo   *moFixture
	exec *mxExec
}

// newMXFixture is the ledger lines (mxReadyLedger when nil) with newMXExec.
func newMXFixture(t *testing.T, lines []any) *mxFixture {
	t.Helper()
	fx := &mxFixture{mo: newMOFixture(t), exec: newMXExec()}
	if lines == nil {
		lines = mxReadyLedger()
	}
	fx.mo.add(t, lines...)
	return fx
}

func (fx *mxFixture) params() MergeParams {
	return MergeParams{Branch: "feat/a", Dir: "/repo", Exec: fx.exec.run, LookPath: mgFound}
}

// check runs checkMerge with p and fails the test unless the temp dir, the
// manifest included, is as it was before, and every command run is read-only
// (mxAssertReadOnly).
func (fx *mxFixture) check(ctx context.Context, t *testing.T, p MergeParams) (mergeReady, string, error) {
	t.Helper()
	before := moSnapshot(t, fx.mo.root)
	r, refused, err := fx.mo.o.checkMerge(ctx, p)
	if after := moSnapshot(t, fx.mo.root); !slices.Equal(after, before) {
		t.Errorf("the temp dir changed:\nbefore %q\nafter  %q", before, after)
	}
	mxAssertReadOnly(t, fx.exec)
	return r, refused, err
}

// mxLsRemoteArgs are the ls-remote's arguments.
var mxLsRemoteArgs = []string{"ls-remote", "--symref", "origin", "HEAD", "refs/heads/*"}

// mxAssertReadOnly fails the test unless every call is one that changes
// nothing: git remote get-url origin, the ls-remote, gh pr list, view and
// checks, gh repo view, and gh api with no method and no input, with -f fields
// only on the GraphQL query; so never gh pr merge. No call names an upstream
// remote either.
func mxAssertReadOnly(t *testing.T, e *mxExec) {
	t.Helper()
	for _, c := range e.snapshot() {
		var readOnly bool
		switch {
		case c.name == "git":
			readOnly = slices.Equal(c.args, []string{"remote", "get-url", "origin"}) || slices.Equal(c.args, mxLsRemoteArgs)
		case slices.Contains([]string{mxPRList, mxPRView, mxPRChecks, mxRepoView}, c.key):
			readOnly = true
		case c.name == "gh" && c.args[0] == "api":
			writes := slices.ContainsFunc(c.args, func(a string) bool {
				return a == "-X" || a == "--input" || strings.HasPrefix(a, "--method") || strings.HasPrefix(a, "-X")
			})
			fields := slices.Contains(c.args, "-f") || slices.Contains(c.args, "-F")
			readOnly = !writes && (!fields || c.key == mxQueue && slices.Contains(c.args, "query="+mergeQueueQuery))
		}
		if !readOnly {
			t.Errorf("%s %q is not a read-only command", c.name, c.args)
		}
		if slices.ContainsFunc(c.args, func(a string) bool { return strings.Contains(a, "upstream") }) {
			t.Errorf("%s %q names the upstream remote", c.name, c.args)
		}
	}
}

// mxAssertPinned fails the test unless every call runs in /repo, git with
// GIT_TERMINAL_PROMPT=0, gh with GH_PROMPT_DISABLED=1 and pinned to origin's
// repository: `-R <repoArg>` on gh pr, repoArg as gh repo view's argument, and
// `--hostname <apiHost>` on gh api.
func mxAssertPinned(t *testing.T, e *mxExec, apiHost, repoArg string) {
	t.Helper()
	for _, c := range e.snapshot() {
		env := []string{"GH_PROMPT_DISABLED=1"}
		var pinned bool
		switch {
		case c.name == "git":
			env, pinned = []string{"GIT_TERMINAL_PROMPT=0"}, true
		case c.args[0] == "pr":
			i := slices.Index(c.args, "-R")
			pinned = i > 0 && i+1 < len(c.args) && c.args[i+1] == repoArg
		case c.args[0] == "repo":
			pinned = len(c.args) > 2 && c.args[2] == repoArg
		case c.args[0] == "api":
			pinned = slices.Equal(c.args[1:3], []string{"--hostname", apiHost})
		}
		if !pinned || c.dir != "/repo" || !slices.Equal(c.env, env) {
			t.Errorf("%s %q in %q with %q is not pinned to %s (%s) in /repo with %q", c.name, c.args, c.dir, c.env, repoArg, apiHost, env)
		}
	}
}

// mxGHMissing is the refusal of a gh that is not on PATH.
const mxGHMissing = `org: merge: gh is not on PATH (exec: "gh": executable file not found in $PATH), and ralph org merge reads ` +
	"the PR and merges it with gh, so nothing was merged: install gh, log in to github.com (gh auth login --hostname " +
	"github.com), and " + mxAgain

// mxNotFound is how a gh that is not installed fails.
var mxNotFound = mgReply{err: &exec.Error{Name: "gh", Err: exec.ErrNotFound}}

// mxForeign is the refusal of a PR whose url u names another repository than
// origin's, as cmd returned it.
func mxForeign(cmd, u string) string {
	return "org: merge: " + cmd + " returned PR #7 with the url " + u + ", which is not a PR of origin's repository " + mxRepoArg +
		", so nothing was merged: ralph org merge merges only origin's PRs; check origin (git remote get-url origin), and merge " +
		"the PR yourself if it is the one to merge"
}

// TestCheckMerge_RefusesWithNothingAfterTheFailingCheck covers AC3, AC4 and
// AC5 of the plan for the checks: each refusal of the 13 steps, worded with
// what was found, that nothing was merged, and one next step, with no command
// after the failing check run, nothing written, and only read-only commands.
func TestCheckMerge_RefusesWithNothingAfterTheFailingCheck(t *testing.T) {
	const listCmd = "gh pr list --head feat/a -R " + mxRepoArg
	const viewCmd = "gh pr view 7 -R " + mxRepoArg
	const lsCmd = "git ls-remote --symref origin HEAD 'refs/heads/*' in /repo"
	notReady := func(class, detail string) string {
		return "org: merge: ralph org merge-order lists the branch feat/a as " + class + ", not ready, so nothing was merged: " + detail
	}
	noHost := "org: merge: origin's URL names no host and owner/repo (a local path, file://, or a scheme other than https, http, " +
		"ssh and git), so there is no GitHub repository to merge in and nothing was merged: point origin at the GitHub repository " +
		"(git remote set-url origin <URL>) and " + mxAgain
	notListed := func(branch string) string {
		return "org: merge: no feature binding or gate record in the ledger names the branch " + branch + ", so nothing was merged: " +
			"ralph org merge merges only a branch ralph org merge-order lists as ready, so run it to see the ledger's branches"
	}
	moved := func(what string) string {
		return "org: merge: origin moved while the checks ran: " + what + ", so nothing was merged: run ralph org merge-order again " +
			"and follow the branch's Detail"
	}
	method := func(m string) string {
		return `org: merge: gh repo view github.com/o/r reported the merge method "` + m + `", which ralph org merge does not know ` +
			"(it merges with MERGE, SQUASH or REBASE), so nothing was merged: merge the PR yourself"
	}
	view := func(state, head, base, url, autoMerge string) func(fx *mxFixture, _ *MergeParams) {
		return func(fx *mxFixture, _ *MergeParams) {
			fx.exec.replies[mxPRView] = mgReply{out: mxPRViewJSON(state, head, base, url, autoMerge)}
		}
	}
	reply := func(key string, r mgReply) func(fx *mxFixture, _ *MergeParams) {
		return func(fx *mxFixture, _ *MergeParams) { fx.exec.replies[key] = r }
	}
	listing := func(ls ...string) func(fx *mxFixture, _ *MergeParams) {
		return func(fx *mxFixture, _ *MergeParams) {
			fx.exec.ls = nil
			for _, l := range ls {
				fx.exec.ls = append(fx.exec.ls, mgReply{out: l})
			}
		}
	}
	const pr7 = "https://github.com/o/r/pull/7"
	for _, tc := range []struct {
		name   string
		ledger []any // nil: mxReadyLedger
		setup  func(fx *mxFixture, p *MergeParams)
		want   string
		ran    int // how many of mxAll ran
	}{
		// 1. The branch.
		{"no branch named", nil, func(_ *mxFixture, p *MergeParams) { p.Branch = "" },
			"org: merge: no branch named, so nothing was merged: name the feature branch to merge (ralph org merge-order lists the " +
				"ledger's branches)", 0},
		// 2. origin (D10).
		{"get-url fails", nil, func(fx *mxFixture, _ *MergeParams) {
			fx.exec.originErr = errors.New("exit status 2: error: No such remote 'origin'")
		}, "org: merge: git remote get-url origin failed (exit status 2: error: No such remote 'origin'), so origin's repository is " +
			"unknown and nothing was merged: ralph org merge merges only into origin's repository, so add origin (git remote add " +
			"origin <URL>) and " + mxAgain, 1},
		{"get-url times out", nil, func(fx *mxFixture, p *MergeParams) { fx.exec.block, p.GHTimeout = mxGetURL, 30*time.Millisecond },
			"org: merge: git remote get-url origin timed out after 30ms, so origin's repository is unknown and nothing was merged: " +
				"check the repository's git config and " + mxAgain, 1},
		{"origin is a local path", nil, func(fx *mxFixture, _ *MergeParams) { fx.exec.origin = "/srv/git/r.git\n" }, noHost, 1},
		{"origin is a file URL", nil, func(fx *mxFixture, _ *MergeParams) { fx.exec.origin = "file:///srv/o/r.git\n" }, noHost, 1},
		{"origin names no owner/repo", nil, func(fx *mxFixture, _ *MergeParams) { fx.exec.origin = "https://github.com/o\n" }, noHost, 1},
		{"origin's host starts with - (scp form)", nil, func(fx *mxFixture, _ *MergeParams) { fx.exec.origin = "git@-oProxyCommand=x:o/r.git\n" },
			noHost, 1},
		{"origin's host starts with - (URL form)", nil, func(fx *mxFixture, _ *MergeParams) { fx.exec.origin = "ssh://git@-x.example.com/o/r.git\n" },
			noHost, 1},
		{"origin's owner starts with -", nil, func(fx *mxFixture, _ *MergeParams) { fx.exec.origin = "git@github.com:-o/r.git\n" }, noHost, 1},
		{"origin's repo starts with -", nil, func(fx *mxFixture, _ *MergeParams) { fx.exec.origin = "https://github.com/o/-r.git\n" }, noHost, 1},
		// 3. gh on PATH, and a gh call that is not found later.
		{"gh is not on PATH", nil, func(_ *mxFixture, p *MergeParams) { p.LookPath = mgNotFound }, mxGHMissing, 1},
		{"gh pr list is not found", nil, reply(mxPRList, mxNotFound), mxGHMissing, 3},
		{"gh pr view is not found", nil, reply(mxPRView, mxNotFound), mxGHMissing, 4},
		{"gh pr checks is not found", nil, reply(mxPRChecks, mgReply{out: "", err: mxNotFound.err}), mxGHMissing, 5},
		{"gh api is not found while the protection is read", nil, reply(mxQueue, mxNotFound), mxGHMissing, 7},
		{"gh repo view is not found", nil, reply(mxRepoView, mxNotFound), mxGHMissing, 10},
		// 4. The ledger.
		{"branch not in the ledger", nil, func(_ *mxFixture, p *MergeParams) { p.Branch = "feat/zzz" }, notListed("feat/zzz"), 1},
		{"empty ledger", []any{}, func(*mxFixture, *MergeParams) {}, notListed("feat/a"), 1},
		// 5. The ls-remote.
		{"ls-remote fails", nil, func(fx *mxFixture, _ *MergeParams) {
			fx.exec.ls = []mgReply{{err: errors.New("exit status 128: fatal: could not read from remote")}}
		}, "org: merge: " + lsCmd + " failed (exit status 128: fatal: could not read from remote): origin's state is unknown, so " +
			"nothing was merged; ralph org merge reads origin, so run it in a repository whose origin answers", 2},
		{"ls-remote names no HEAD", nil, listing(moHeadA + "\trefs/heads/feat/a\n"),
			"org: merge: " + lsCmd + " printed no HEAD symref, so origin's default branch is unknown and nothing was merged: point " +
				"origin's HEAD at the default branch and run it again", 2},
		{"ls-remote times out", nil, func(fx *mxFixture, p *MergeParams) {
			fx.exec.block, p.LsRemoteTimeout = mxLsRemote, 30*time.Millisecond
		}, "org: merge: " + lsCmd + " timed out after 30ms (context deadline exceeded): origin's state is unknown, so nothing was " +
			"merged; check the network and origin's credentials and run it again", 2},
		// 6. The PR (D10).
		{"gh pr list fails", nil, reply(mxPRList, mgReply{err: errors.New("exit status 1: HTTP 502: Bad Gateway")}),
			"org: merge: " + listCmd + " failed (exit status 1: HTTP 502: Bad Gateway), so the branch's PR is unknown and nothing " +
				"was merged: " + mxAccess, 3},
		{"gh pr list times out", nil, func(fx *mxFixture, p *MergeParams) { fx.exec.block, p.GHTimeout = mxPRList, 30*time.Millisecond },
			"org: merge: " + listCmd + " failed (timed out after 30ms: context deadline exceeded), so the branch's PR is unknown " +
				"and nothing was merged: " + mxAccess, 3},
		{"gh pr list's limit is full of forks", nil, reply(mxPRList, mgReply{out: moPRs(moForkPRs(100, mergeOrderGHLimit)...)}),
			"org: merge: " + listCmd + " returned 1000 PRs, its limit, all from forks, so the branch's own PR may be past them and " +
				"nothing was merged: merge the branch's PR yourself", 3},
		{"no PR", nil, reply(mxPRList, mgReply{out: "[]"}),
			"org: merge: " + listCmd + " found no PR of the branch feat/a in origin's repository github.com/o/r, so nothing was " +
				"merged: open one (gh pr create --head feat/a -R github.com/o/r), and once its CI passed, " + mxAgain, 3},
		{"PR of another owner/repo", nil, reply(mxPRList, mgReply{out: moPRs(moRepoPR(7, "OPEN", "main", "acme", "app"))}),
			mxForeign(listCmd, "https://github.com/acme/app/pull/7"), 3},
		{"PR of the same owner/repo on another host", nil, func(fx *mxFixture, _ *MergeParams) {
			pr := moPR(7, "OPEN", "main")
			pr.URL = "https://ghe.example.com/o/r/pull/7"
			fx.exec.replies[mxPRList] = mgReply{out: moPRs(pr)}
		}, mxForeign(listCmd, "https://ghe.example.com/o/r/pull/7"), 3},
		{"PR url that is not a PR's", nil, func(fx *mxFixture, _ *MergeParams) {
			pr := moPR(7, "OPEN", "main")
			pr.URL = "https://github.com/o/r/issues/7"
			fx.exec.replies[mxPRList] = mgReply{out: moPRs(pr)}
		}, mxForeign(listCmd, "https://github.com/o/r/issues/7"), 3},
		// 7. The class.
		{"not ready: rebase", nil, listing(moListing("main", "main", moMain2, "feat/a", moHeadA)), notReady("rebase",
			"origin/main moved: gate base 111111111111, now 222222222222: run ralph org regate --branch feat/a, which merges "+
				"origin/main in and runs the gate again"), 3},
		{"not ready: regate", nil, listing(moListing("main", "main", moMain, "feat/a", moHeadB)), notReady("regate",
			"pushed after the gate, or the gate's head was not pushed: origin has bbbbbbbbbbbb, the gate checked aaaaaaaaaaaa: "+
				"run ralph org regate --branch feat/a"), 3},
		{"not ready: merged", nil, reply(mxPRList, mgReply{out: moPRs(moPR(7, "MERGED", "main"))}), notReady("merged", "PR #7 is merged"), 3},
		{"not ready: wrong_base", nil, reply(mxPRList, mgReply{out: moPRs(moPR(7, "OPEN", "develop"))}), notReady("wrong_base",
			"PR #7 merges into develop, not origin's default branch main: change the PR's base to main"), 3},
		{"not ready: no_gate", []any{moBind("org-a", "a"), moDisbanded("org-a")}, func(*mxFixture, *MergeParams) {}, notReady("no_gate",
			"no gate record: run ralph org regate --branch feat/a before the PR is merged"), 3},
		// 8. The org (D5).
		{"org running", []any{moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain)}, func(*mxFixture, *MergeParams) {},
			`org: merge: the branch feat/a is bound to org_id "org-a", which is running, so its leader may still push to the branch ` +
				"and open or change the PR, and nothing was merged: handle the org's RESULT in the inbox (ralph org inbox), and once " +
				"the org is disbanded, " + mxAgain, 3},
		{"gate-only branch whose gate's org runs", []any{moBind("org-b", "b"), moPassed("org-b", "feat/a", moHeadA, moMain)},
			func(*mxFixture, *MergeParams) {}, `org: merge: the branch feat/a was gated by org_id "org-b", which is running, so its ` +
				"leader may still push to the branch and open or change the PR, and nothing was merged: handle the org's RESULT in " +
				"the inbox (ralph org inbox), and once the org is disbanded, " + mxAgain, 3},
		{"org runs another feature now", append(mxReadyLedger(), moBindTo("org-a", "split-1", "other", "feat/other")),
			func(*mxFixture, *MergeParams) {}, `org: merge: the branch feat/a is bound to org_id "org-a", which is running, now for ` +
				"another feature on the branch feat/other, so nothing was merged: wait for the org to be disbanded and " + mxAgain, 3},
		// 9. gh pr view (D12).
		{"gh pr view fails", nil, reply(mxPRView, mgReply{err: errors.New("exit status 1: GraphQL: Could not resolve to a PullRequest")}),
			"org: merge: " + viewCmd + " failed (exit status 1: GraphQL: Could not resolve to a PullRequest), so the PR's state is " +
				"unknown and nothing was merged: " + mxAccess, 4},
		{"gh pr view prints no JSON", nil, reply(mxPRView, mgReply{out: "<html>"}),
			"org: merge: " + viewCmd + " printed output that is not the PR's JSON with state,url,headRefOid,baseRefName," +
				"autoMergeRequest, so nothing was merged: " + mxAccess, 4},
		{"gh pr view leaves autoMergeRequest out", nil, reply(mxPRView, mgReply{out: `{"state":"OPEN","url":"` + pr7 +
			`","headRefOid":"` + moHeadA + `","baseRefName":"main"}`}), "org: merge: " + viewCmd + " printed output that is not the " +
			"PR's JSON with state,url,headRefOid,baseRefName,autoMergeRequest, so nothing was merged: " + mxAccess, 4},
		{"PR is not OPEN", nil, view("MERGED", moHeadA, "main", pr7, "null"),
			"org: merge: PR #7 is MERGED, not OPEN, so nothing was merged: run ralph org merge-order to see the branch's state", 4},
		{"PR's head is not the gate's", nil, view("OPEN", moHeadB, "main", pr7, "null"),
			"org: merge: PR #7's head is bbbbbbbbbbbb, not aaaaaaaaaaaa, the head the gate passed, so nothing was merged: run ralph " +
				"org merge-order again and follow the branch's Detail", 4},
		{"PR's base is not the default branch", nil, view("OPEN", moHeadA, "develop", pr7, "null"),
			"org: merge: PR #7 merges into develop, not origin's default branch main, so nothing was merged: change its base (gh pr " +
				"edit 7 --base main -R github.com/o/r), and once its CI passed, " + mxAgain, 4},
		{"PR view's url names another repository", nil, view("OPEN", moHeadA, "main", "https://github.com/acme/app/pull/7", "null"),
			mxForeign(viewCmd, "https://github.com/acme/app/pull/7"), 4},
		{"PR view's url names another host", nil, view("OPEN", moHeadA, "main", "https://ghe.example.com/o/r/pull/7", "null"),
			mxForeign(viewCmd, "https://ghe.example.com/o/r/pull/7"), 4},
		{"PR has an auto-merge request", nil, view("OPEN", moHeadA, "main", pr7, `{"enabledAt":"2026-10-17T00:00:00Z","mergeMethod":"MERGE"}`),
			"org: merge: PR #7 has an auto-merge request already, which GitHub would carry out apart from ralph org merge, so nothing " +
				"was merged: remove it (gh pr merge 7 --disable-auto -R github.com/o/r) and " + mxAgain, 4},
		// 10. gh pr checks (D7); TestCheckMerge_ReadsTheChecks has the rest.
		{"a check is pending", nil, reply(mxPRChecks, mgReply{out: mxChecks(mxPass("verify"),
			ghPRCheck{Bucket: "pending", Name: "lint", State: "IN_PROGRESS", Workflow: "ci"}), err: errors.New("exit status 8")}),
			"org: merge: 1 of the 2 checks of PR #7 are not pass or skipping: ci / lint (pending, IN_PROGRESS), so nothing was merged: " +
				"wait for the pending ones and fix the others, and once all passed, " + mxAgain, 5},
		// 11. The protection (D11); TestMergeEnforcement_DecidesWhetherTheMergeMayGoOn has the rest.
		{"merge queue", nil, reply(mxQueue, mgReply{out: meQueueJSON(`{"id":"MQ_1"}`)}), mxProtect + mxSubject + ", has a merge queue " +
			"(GraphQL mergeQueue is set): ralph org merge does not merge into a branch with a merge queue, so merge the PR through " +
			"the queue yourself", 7},
		{"ruleset the caller can bypass", nil, reply(mxRuleset, mgReply{out: meRuleset(1, "protect main", "always")}), mxProtect +
			"every ruleset of " + mxSubject + `, that requires branches to be up to date before merging with required status checks ` +
			`of its own can be bypassed by you (current_user_can_bypass of "protect main" (id 1): "always"), so GitHub would let you ` +
			"merge a head that is not up to date: take whoever runs ralph org merge off the bypass list of one of them, or merge " +
			"the PR yourself", 9},
		// 12. The merge method (D2).
		{"merge method empty", nil, reply(mxRepoView, mgReply{out: `{"viewerDefaultMergeMethod":""}`}), method(""), 10},
		{"merge method MERGE_QUEUE", nil, reply(mxRepoView, mgReply{out: `{"viewerDefaultMergeMethod":"MERGE_QUEUE"}`}), method("MERGE_QUEUE"), 10},
		{"merge method in lower case", nil, reply(mxRepoView, mgReply{out: `{"viewerDefaultMergeMethod":"merge"}`}), method("merge"), 10},
		{"merge method missing", nil, reply(mxRepoView, mgReply{out: `{}`}), method(""), 10},
		{"gh repo view fails", nil, reply(mxRepoView, mgReply{err: errors.New("exit status 1: HTTP 404: Not Found")}),
			"org: merge: gh repo view github.com/o/r failed (exit status 1: HTTP 404: Not Found), so the merge method is unknown and " +
				"nothing was merged: " + mxAccess, 10},
		{"gh repo view prints no JSON", nil, reply(mxRepoView, mgReply{out: "<html>"}),
			"org: merge: gh repo view github.com/o/r printed output that is not the repository's JSON (invalid character '<' looking " +
				"for beginning of value), so the merge method is unknown and nothing was merged: " + mxAccess, 10},
		// 13. origin again (D8).
		{"origin's default branch moved", nil, listing(mxListing(), moListing("main", "main", moMain2, "feat/a", moHeadA)),
			moved("origin/main moved to 222222222222 from 111111111111, the gate's base"), 11},
		{"the branch moved", nil, listing(mxListing(), moListing("main", "main", moMain, "feat/a", moHeadB)),
			moved("origin's feat/a moved to bbbbbbbbbbbb from aaaaaaaaaaaa, the head the gate passed"), 11},
		{"the branch is gone", nil, listing(mxListing(), moListing("main", "main", moMain)), moved("origin has no branch feat/a any more"), 11},
		{"the default branch is another", nil, listing(mxListing(), moListing("trunk", "trunk", moMain, "feat/a", moHeadA)),
			moved("origin's default branch is now trunk, not main"), 11},
		{"the second ls-remote fails", nil, func(fx *mxFixture, _ *MergeParams) {
			fx.exec.ls = []mgReply{{out: mxListing()}, {err: errors.New("exit status 128: fatal: unable to access")}}
		}, "org: merge: " + lsCmd + " failed (exit status 128: fatal: unable to access): origin's state is unknown, so nothing was " +
			"merged; ralph org merge reads origin, so run it in a repository whose origin answers", 11},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMXFixture(t, tc.ledger)
			p := fx.params()
			tc.setup(fx, &p)
			r, refused, err := fx.check(context.Background(), t, p)
			if err != nil || refused != tc.want || r != (mergeReady{}) {
				t.Fatalf("checkMerge = %+v, %v\nrefused %q\nwant    %q", r, err, refused, tc.want)
			}
			if got := fx.exec.keys(); !slices.Equal(got, mxAll[:tc.ran]) {
				t.Errorf("commands = %q\nwant %q", got, mxAll[:tc.ran])
			}
		})
	}
}

// TestCheckMerge_ReadsTheChecks covers D7: gh pr checks passes only when
// every bucket is pass or skipping and one is pass; zero checks, skipping
// only, and any pending, fail, cancel or unknown bucket refuse, naming the
// checks (the first five and how many more). stdout is read whatever gh's
// exit (8 while a check is pending); a stdout that is not a JSON list refuses
// with gh's error, and a timeout refuses.
func TestCheckMerge_ReadsTheChecks(t *testing.T) {
	const cmd = "gh pr checks 7 -R " + mxRepoArg
	check := func(bucket, name, state string) ghPRCheck {
		return ghPRCheck{Bucket: bucket, Name: name, State: state, Workflow: "ci"}
	}
	notPassing := func(n, of int, named string) string {
		return fmt.Sprintf("org: merge: %d of the %d checks of PR #7 are not pass or skipping: %s, so nothing was merged: wait for "+
			"the pending ones and fix the others, and once all passed, %s", n, of, named, mxAgain)
	}
	var many []ghPRCheck
	var manyNamed []string
	for i := range 7 {
		many = append(many, check("pending", fmt.Sprintf("c%d", i+1), "QUEUED"))
		manyNamed = append(manyNamed, fmt.Sprintf("ci / c%d (pending, QUEUED)", i+1))
	}
	for _, tc := range []struct {
		name  string
		reply mgReply
		want  string // "" when the checks pass
	}{
		{"pass only", mgReply{out: mxChecks(mxPass("verify"), mxPass("test"))}, ""},
		{"pass and skipping", mgReply{out: mxChecks(mxPass("verify"), check("skipping", "deploy", "SKIPPED"))}, ""},
		{"skipping only", mgReply{out: mxChecks(check("skipping", "deploy", "SKIPPED"))},
			"org: merge: every one of the 1 checks of PR #7 is skipping and none passed, so nothing shows that the PR's CI passed " +
				"and nothing was merged: let a check run on the PR, and once it passed, " + mxAgain},
		{"one pending", mgReply{out: mxChecks(mxPass("verify"), check("pending", "lint", "IN_PROGRESS"))},
			notPassing(1, 2, "ci / lint (pending, IN_PROGRESS)")},
		{"no checks", mgReply{out: "[]"}, "org: merge: " + cmd + " reported no checks, so nothing shows that the PR's CI passed and " +
			"nothing was merged: let CI run on the PR, and once it passed, " + mxAgain},
		{"exit 8 with a pending check on stdout", mgReply{out: mxChecks(check("pending", "lint", "QUEUED"), mxPass("verify")),
			err: errors.New("exit status 8")}, notPassing(1, 2, "ci / lint (pending, QUEUED)")},
		{"a non-zero exit with checks that pass on stdout reads them", mgReply{out: mxChecks(mxPass("verify")),
			err: errors.New("exit status 8")}, ""},
		{"exit 1 with output that is not JSON", mgReply{out: "no checks reported on the 'feat/a' branch\n",
			err: errors.New("exit status 1: no checks reported on the 'feat/a' branch")},
			"org: merge: " + cmd + " failed (exit status 1: no checks reported on the 'feat/a' branch), so the PR's checks are " +
				"unknown and nothing was merged: " + mxAccess},
		{"exit 0 with output that is not JSON", mgReply{out: "<html>"}, "org: merge: " + cmd + " printed output that is not a " +
			"JSON list of checks, so the PR's checks are unknown and nothing was merged: " + mxAccess},
		{"null is not a list", mgReply{out: "null"}, "org: merge: " + cmd + " printed output that is not a JSON list of checks, so " +
			"the PR's checks are unknown and nothing was merged: " + mxAccess},
		{"an object is not a list", mgReply{out: `{"bucket":"pass"}`}, "org: merge: " + cmd + " printed output that is not a JSON " +
			"list of checks, so the PR's checks are unknown and nothing was merged: " + mxAccess},
		{"fail and cancel", mgReply{out: mxChecks(mxPass("verify"), check("fail", "test", "FAILURE"), check("cancel", "e2e", "CANCELLED")),
			err: errors.New("exit status 1")}, notPassing(2, 3, "ci / test (fail, FAILURE), ci / e2e (cancel, CANCELLED)")},
		{"a bucket not known", mgReply{out: mxChecks(mxPass("verify"), ghPRCheck{Name: "x"}, ghPRCheck{Bucket: "PASS", Name: "y",
			State: "SUCCESS", Workflow: "y"})}, notPassing(2, 3, "x (no bucket, no state), y (PASS, SUCCESS)")},
		{"more than five", mgReply{out: mxChecks(append([]ghPRCheck{mxPass("verify")}, many...)...)},
			notPassing(7, 8, strings.Join(manyNamed[:5], ", ")+" and 2 more")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMXFixture(t, nil)
			fx.exec.replies[mxPRChecks] = tc.reply
			r, refused, err := fx.check(context.Background(), t, fx.params())
			switch {
			case err != nil:
				t.Fatalf("checkMerge error: %v", err)
			case tc.want == "" && (refused != "" || r.prNumber != 7):
				t.Fatalf("checkMerge = %+v, refused %q; want it to pass", r, refused)
			case tc.want != "" && refused != tc.want:
				t.Fatalf("refused %q\nwant    %q", refused, tc.want)
			}
			want := mxAll
			if tc.want != "" {
				want = mxAll[:5]
			}
			if got := fx.exec.keys(); !slices.Equal(got, want) {
				t.Errorf("commands = %q\nwant %q", got, want)
			}
		})
	}

	t.Run("times out", func(t *testing.T) {
		fx := newMXFixture(t, nil)
		fx.exec.block = mxPRChecks
		p := fx.params()
		p.GHTimeout = 30 * time.Millisecond
		_, refused, err := fx.check(context.Background(), t, p)
		want := "org: merge: " + cmd + " failed (timed out after 30ms: context deadline exceeded), so the PR's checks are unknown " +
			"and nothing was merged: " + mxAccess
		if err != nil || refused != want {
			t.Fatalf("checkMerge = %v\nrefused %q\nwant    %q", err, refused, want)
		}
	})
}

// TestCheckMerge_EscapesControlCharactersInTheRefusal: a check name gh
// returns with control characters reaches the refusal written as Go escapes,
// so a terminal that prints it runs none.
func TestCheckMerge_EscapesControlCharactersInTheRefusal(t *testing.T) {
	fx := newMXFixture(t, nil)
	fx.exec.replies[mxPRChecks] = mgReply{out: mxChecks(mxPass("verify"), ghPRCheck{Bucket: "fail", Name: "lint\x1b[2J\a",
		State: "FAILURE"})}
	_, refused, err := fx.check(context.Background(), t, fx.params())
	if err != nil || !strings.Contains(refused, `lint\x1b[2J\a (fail, FAILURE)`) {
		t.Fatalf("checkMerge = %v, refused %q", err, refused)
	}
	if strings.ContainsFunc(refused, unicode.IsControl) {
		t.Errorf("refused %q holds a raw control character", refused)
	}
}

// TestCheckMerge_PassesWithEverythingTheMergeNeeds covers the passing path:
// mergeReady holds the branch, the org (D4), origin's repository, the PR, the
// gate's head and base, the method and the protection; the commands are
// mxAll, each with its exact argv, pinned to origin's repository (AC4); with
// an upstream remote that gh would read without -R, nothing reads it.
func TestCheckMerge_PassesWithEverythingTheMergeNeeds(t *testing.T) {
	fx := newMXFixture(t, nil)
	fx.exec.upstream = true
	r, refused, err := fx.check(context.Background(), t, fx.params())
	if err != nil || refused != "" {
		t.Fatalf("checkMerge = %v, refused %q", err, refused)
	}
	want := mergeReady{
		branch: "feat/a", orgID: "org-a", apiHost: "github.com", repo: mergeRepo{owner: "o", name: "r"}, repoArg: mxRepoArg,
		prNumber: 7, prURL: "https://github.com/o/r/pull/7", head: moHeadA, base: moMain, defaultBranch: "main",
		method: "merge", methodFlag: "--merge",
		protection: `the ruleset "protect main" (id 1) of ` + mxSubject + ", requires branches to be up to date before merging " +
			"with its 1 required status check, and you cannot bypass it (current_user_can_bypass: never)",
	}
	if r != want {
		t.Fatalf("mergeReady =\n%+v\nwant\n%+v", r, want)
	}
	argv := [][]string{
		{"git", "remote", "get-url", "origin"},
		append([]string{"git"}, mxLsRemoteArgs...),
		{"gh", "pr", "list", "--head", "feat/a", "--state", "all", "--limit", "1000", "--json", mergeOrderGHFields, "-R", mxRepoArg},
		{"gh", "pr", "view", "7", "-R", mxRepoArg, "--json", "state,url,headRefOid,baseRefName,autoMergeRequest"},
		{"gh", "pr", "checks", "7", "-R", mxRepoArg, "--json", "bucket,name,state,workflow"},
		{"gh", "api", "--hostname", "github.com", "repos/o/r"},
		{"gh", "api", "--hostname", "github.com", "-f", "query=" + mergeQueueQuery, "-f", "owner=o", "-f", "name=r", "-f",
			"branch=main", "graphql"},
		{"gh", "api", "--hostname", "github.com", "--paginate", "repos/o/r/rules/branches/main"},
		{"gh", "api", "--hostname", "github.com", "repos/o/r/rulesets/1"},
		{"gh", "repo", "view", mxRepoArg, "--json", "viewerDefaultMergeMethod"},
		append([]string{"git"}, mxLsRemoteArgs...),
	}
	calls := fx.exec.snapshot()
	if len(calls) != len(argv) {
		t.Fatalf("commands = %q, want %d", fx.exec.keys(), len(argv))
	}
	for i, c := range calls {
		if got := append([]string{c.name}, c.args...); !slices.Equal(got, argv[i]) {
			t.Errorf("command %d = %q\nwant        %q", i, got, argv[i])
		}
	}
	mxAssertPinned(t, fx.exec, "github.com", mxRepoArg)
}

// TestCheckMerge_PassesForOtherOriginsAndLedgers: an ssh.github.com origin
// is github.com; a GitHub Enterprise origin pins every call to its host; a
// PR url whose owner/repo differs from origin's only in case is origin's; a
// branch only a gate record names is recorded under the gate's org; and
// AutoMerge changes nothing.
func TestCheckMerge_PassesForOtherOriginsAndLedgers(t *testing.T) {
	for _, tc := range []struct {
		name             string
		ledger           []any
		setup            func(fx *mxFixture, p *MergeParams)
		apiHost, repoArg string
		orgID            string
	}{
		{"ssh.github.com origin", nil, func(fx *mxFixture, _ *MergeParams) {
			fx.exec.origin = "ssh://git@ssh.github.com:443/o/r.git\n"
		}, "github.com", mxRepoArg, "org-a"},
		{"GitHub Enterprise origin", nil, func(fx *mxFixture, _ *MergeParams) {
			fx.exec.origin = "git@GHE.example.com:o/r.git\n"
			pr := moPR(7, "OPEN", "main")
			pr.URL = "https://ghe.example.com/o/r/pull/7"
			fx.exec.replies[mxPRList] = mgReply{out: moPRs(pr)}
			fx.exec.replies[mxPRView] = mgReply{out: mxPRViewJSON("OPEN", moHeadA, "main", pr.URL, "null")}
		}, "ghe.example.com", "ghe.example.com/o/r", "org-a"},
		{"PR url in other case", nil, func(fx *mxFixture, _ *MergeParams) {
			pr := moPR(7, "OPEN", "main")
			pr.URL = "https://GitHub.com/O/R/pull/7"
			fx.exec.replies[mxPRList] = mgReply{out: moPRs(pr)}
			fx.exec.replies[mxPRView] = mgReply{out: mxPRViewJSON("OPEN", moHeadA, "main", pr.URL, "null")}
		}, "github.com", mxRepoArg, "org-a"},
		{"gate-only branch", []any{moPassed("org-x", "feat/a", moHeadA, moMain)}, func(*mxFixture, *MergeParams) {}, "github.com",
			mxRepoArg, "org-x"},
		{"AutoMerge set", nil, func(_ *mxFixture, p *MergeParams) { p.AutoMerge = true }, "github.com", mxRepoArg, "org-a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMXFixture(t, tc.ledger)
			p := fx.params()
			tc.setup(fx, &p)
			r, refused, err := fx.check(context.Background(), t, p)
			if err != nil || refused != "" {
				t.Fatalf("checkMerge = %v, refused %q", err, refused)
			}
			if r.apiHost != tc.apiHost || r.repoArg != tc.repoArg || r.orgID != tc.orgID || r.prNumber != 7 || r.methodFlag != "--merge" {
				t.Errorf("mergeReady = %+v, want host %s, repo %s, org %s", r, tc.apiHost, tc.repoArg, tc.orgID)
			}
			if got := fx.exec.keys(); !slices.Equal(got, mxAll) {
				t.Errorf("commands = %q\nwant %q", got, mxAll)
			}
			mxAssertPinned(t, fx.exec, tc.apiHost, tc.repoArg)
		})
	}
}

// TestCheckMerge_MapsTheMergeMethodToItsFlag covers D2: MERGE, SQUASH and
// REBASE map to gh pr merge's --merge, --squash and --rebase.
func TestCheckMerge_MapsTheMergeMethodToItsFlag(t *testing.T) {
	for value, want := range map[string][2]string{
		"MERGE": {"merge", "--merge"}, "SQUASH": {"squash", "--squash"}, "REBASE": {"rebase", "--rebase"},
	} {
		t.Run(value, func(t *testing.T) {
			fx := newMXFixture(t, nil)
			fx.exec.replies[mxRepoView] = mgReply{out: `{"viewerDefaultMergeMethod":"` + value + `"}`}
			r, refused, err := fx.check(context.Background(), t, fx.params())
			if err != nil || refused != "" || r.method != want[0] || r.methodFlag != want[1] {
				t.Fatalf("checkMerge = %+v, %v, %q; want %s %s", r, err, refused, want[0], want[1])
			}
		})
	}
}

// TestCheckMerge_TimeoutsDefaultAndBound: with the timeouts left zero, the
// get-url and each gh call run under 15 s, each ls-remote under 30 s, and
// the protection's calls under one deadline of 20 s; set, they are used.
func TestCheckMerge_TimeoutsDefaultAndBound(t *testing.T) {
	for _, tc := range []struct {
		name                           string
		ls, gh, protection             time.Duration // MergeParams'
		wantLs, wantGH, wantProtection time.Duration
	}{
		{"defaults", 0, 0, 0, 30 * time.Second, 15 * time.Second, 20 * time.Second},
		{"less than zero", -time.Second, -time.Second, -time.Second, 30 * time.Second, 15 * time.Second, 20 * time.Second},
		{"set", 7 * time.Second, 3 * time.Second, 5 * time.Second, 7 * time.Second, 3 * time.Second, 5 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMXFixture(t, nil)
			p := fx.params()
			p.LsRemoteTimeout, p.GHTimeout, p.EnforcementTimeout = tc.ls, tc.gh, tc.protection
			start := time.Now()
			if _, refused, err := fx.check(context.Background(), t, p); err != nil || refused != "" {
				t.Fatalf("checkMerge = %v, refused %q", err, refused)
			}
			end := time.Now()
			wantFor := map[string]time.Duration{mxLsRemote: tc.wantLs}
			for _, k := range []string{mxGetURL, mxPRList, mxPRView, mxPRChecks, mxRepoView} {
				wantFor[k] = tc.wantGH
			}
			for _, k := range []string{mxRepoAPI, mxQueue, mxRules, mxRuleset} {
				wantFor[k] = tc.wantProtection
			}
			var protection time.Time
			for _, c := range fx.exec.snapshot() {
				want := wantFor[c.key]
				if c.deadline.Before(start.Add(want)) || c.deadline.After(end.Add(want)) {
					t.Errorf("%s: deadline %v is not %s after the start (%v..%v)", c.key, c.deadline, want, start, end)
				}
				if c.name == "gh" && c.args[0] == "api" {
					if protection.IsZero() {
						protection = c.deadline
					} else if !c.deadline.Equal(protection) {
						t.Errorf("%s: deadline %v, the protection's first call had %v", c.key, c.deadline, protection)
					}
				}
			}
		})
	}
}

// TestCheckMerge_InterruptedIsAnErrorAndNothingIsMerged: ctx done after any
// command, or before the first, is an error that says nothing was merged,
// with at most the next command run, nothing written, and no refusal.
func TestCheckMerge_InterruptedIsAnErrorAndNothingIsMerged(t *testing.T) {
	for n := range len(mxAll) {
		t.Run(fmt.Sprintf("after %d %s", n, mxAll[n]), func(t *testing.T) {
			fx := newMXFixture(t, nil)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			fx.exec.after = func(i int) {
				if i == n {
					cancel(errors.New("test stopped"))
				}
			}
			r, refused, err := fx.check(ctx, t, fx.params())
			if err == nil || !strings.HasPrefix(err.Error(), "org: merge: interrupted (test stopped) while ") ||
				!strings.HasSuffix(err.Error(), ": nothing was merged") || refused != "" || r != (mergeReady{}) {
				t.Fatalf("checkMerge = %+v, %v, refused %q; want the interruption", r, err, refused)
			}
			got := fx.exec.keys()
			if len(got) > n+2 || !slices.Equal(got, mxAll[:len(got)]) {
				t.Errorf("commands = %q, want mxAll up to the one after %d", got, n)
			}
		})
	}

	fx := newMXFixture(t, nil)
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("test stopped"))
	_, refused, err := fx.check(ctx, t, fx.params())
	if err == nil || err.Error() != "org: merge: interrupted (test stopped) while git remote get-url origin ran: nothing was merged" ||
		refused != "" {
		t.Fatalf("checkMerge with ctx done = %v, refused %q", err, refused)
	}
}

// TestCheckMerge_ManifestThatCannotBeReadIsAnError: a manifest that cannot
// be read is an error, after origin and gh were found and before any other
// command.
func TestCheckMerge_ManifestThatCannotBeReadIsAnError(t *testing.T) {
	fx := newMXFixture(t, []any{})
	mkdirAll(t, fx.mo.path) // a directory where the manifest should be
	_, refused, err := fx.check(context.Background(), t, fx.params())
	if err == nil || !strings.HasPrefix(err.Error(), "org: merge: read manifest: ") || refused != "" {
		t.Fatalf("checkMerge = %v, refused %q; want a read manifest error", err, refused)
	}
	if got := fx.exec.keys(); !slices.Equal(got, []string{mxGetURL}) {
		t.Errorf("commands = %q, want the get-url only", got)
	}
}
