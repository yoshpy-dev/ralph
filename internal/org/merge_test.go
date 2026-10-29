package org

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode"
)

// The checks of `ralph org merge` at the org layer (plan
// docs/plans/active/2026-10-17-org-auto-merge.md, S3a: the org-layer half of
// AC3 to AC5): checkMerge on manifest fixtures, with a fake MergeOrderExec
// standing in for git and gh (mxExec and mxFixture, merge_fake_test.go), so
// no test runs gh or reaches a network.

// TestCheckMerge_RefusesWithNothingAfterTheFailingCheck covers AC3, AC4 and
// AC5 of the plan for the checks: each refusal of the 13 steps, worded with
// what was found, that nothing was merged, and one next step, with no command
// after the failing check run, nothing written, and only read-only commands.
// Every refusal here is of a kind --wait-checks does not wait out, with
// nothing read kept; TestCheckMerge_OrgThatStillRunsIsOfAKindThatWaits and
// TestCheckMerge_ReadsTheChecks have those it waits out.
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
		// 8. The org (D5); an org that still runs the branch's feature is in
		// TestCheckMerge_OrgThatStillRunsIsOfAKindThatWaits.
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
		// 10. gh pr checks (D7); TestCheckMerge_ReadsTheChecks has the rest,
		// pending checks and none among them, whose refusal keeps what
		// checkMerge read for the wait of --wait-checks.
		{"a check failed", nil, reply(mxPRChecks, mgReply{out: mxChecks(mxPass("verify"),
			ghPRCheck{Bucket: "fail", Name: "lint", State: "FAILURE", Workflow: "ci"}), err: errors.New("exit status 1")}),
			"org: merge: 1 of the 2 checks of PR #7 are not pass or skipping: ci / lint (fail, FAILURE), so nothing was merged: " +
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
			r, refused, kind, err := fx.checkKind(context.Background(), t, p)
			if err != nil || refused != tc.want || r != (mergeReady{}) || kind.waits() {
				t.Fatalf("checkMerge = %+v, %v, kind %d\nrefused %q\nwant    %q", r, err, kind, refused, tc.want)
			}
			if got := fx.exec.keys(); !slices.Equal(got, mxAll[:tc.ran]) {
				t.Errorf("commands = %q\nwant %q", got, mxAll[:tc.ran])
			}
		})
	}
}

// TestCheckMerge_OrgThatStillRunsIsOfAKindThatWaits covers step 8 (D5) for an
// org that still runs the branch's feature, the org of its binding or, for a
// branch with a gate record only, the gate's: the refusal is of the kind
// mergeRefusalOrgRunning, which --wait-checks waits out (D13), and keeps the
// branch and the org_id, for the wait to read the org's disband in the
// ledger, with nothing after step 8 run. An org that runs another feature now
// is refused with a kind not waited out (in the table above).
func TestCheckMerge_OrgThatStillRunsIsOfAKindThatWaits(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ledger []any
		org    string
		want   string
	}{
		{"org running", []any{moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain)}, "org-a",
			`org: merge: the branch feat/a is bound to org_id "org-a", which is running, so its leader may still push to the branch ` +
				"and open or change the PR, and nothing was merged: handle the org's RESULT in the inbox (ralph org inbox), and once " +
				"the org is disbanded, " + mxAgain},
		{"gate-only branch whose gate's org runs", []any{moBind("org-b", "b"), moPassed("org-b", "feat/a", moHeadA, moMain)}, "org-b",
			`org: merge: the branch feat/a was gated by org_id "org-b", which is running, so its leader may still push to the ` +
				"branch and open or change the PR, and nothing was merged: handle the org's RESULT in the inbox (ralph org inbox), " +
				"and once the org is disbanded, " + mxAgain},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMXFixture(t, tc.ledger)
			r, refused, kind, err := fx.checkKind(context.Background(), t, fx.params())
			if err != nil || refused != tc.want || kind != mergeRefusalOrgRunning || !kind.waits() {
				t.Fatalf("checkMerge = %v, kind %d\nrefused %q\nwant    %q", err, kind, refused, tc.want)
			}
			if r.branch != "feat/a" || r.orgID != tc.org || r.repoArg != mxRepoArg {
				t.Errorf("checkMerge = %+v; want the branch feat/a, org_id %s, and origin's repository", r, tc.org)
			}
			if got := fx.exec.keys(); !slices.Equal(got, mxAll[:3]) {
				t.Errorf("commands = %q\nwant %q", got, mxAll[:3])
			}
		})
	}
}

// TestCheckMerge_ReadsTheChecks covers D7: gh pr checks passes only when
// every bucket is pass or skipping and one is pass; zero checks, skipping
// only, and any pending, fail, cancel or unknown bucket refuse, naming the
// checks (the first five and how many more). stdout is read whatever gh's
// exit (8 while a check is pending); a stdout that is not a JSON list refuses
// with gh's error, and a timeout refuses, except that gh's exit 1 with
// mergeNoChecksSentence for the branch, as gh 2.102.0 answers a PR with no
// checks, is zero checks. The kind of each refusal is the one --wait-checks
// decides on (D13): only pending checks and none wait, and their refusal
// keeps what checkMerge read; every other refusal leaves it empty.
func TestCheckMerge_ReadsTheChecks(t *testing.T) {
	const cmd = "gh pr checks 7 -R " + mxRepoArg
	check := func(bucket, name, state string) ghPRCheck {
		return ghPRCheck{Bucket: bucket, Name: name, State: state, Workflow: "ci"}
	}
	notPassing := func(n, of int, named string) string {
		return fmt.Sprintf("org: merge: %d of the %d checks of PR #7 are not pass or skipping: %s, so nothing was merged: wait for "+
			"the pending ones and fix the others, and once all passed, %s", n, of, named, mxAgain)
	}
	ghFailed := func(why string) string {
		return "org: merge: " + cmd + " failed (" + why + "), so the PR's checks are unknown and nothing was merged: " + mxAccess
	}
	noChecks := "org: merge: " + cmd + " reported no checks, so nothing shows that the PR's CI passed and nothing was merged: let " +
		"CI run on the PR, and once it passed, " + mxAgain
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
		kind  mergeRefusalKind
	}{
		{"pass only", mgReply{out: mxChecks(mxPass("verify"), mxPass("test"))}, "", mergeRefusalOther},
		{"pass and skipping", mgReply{out: mxChecks(mxPass("verify"), check("skipping", "deploy", "SKIPPED"))}, "", mergeRefusalOther},
		{"skipping only", mgReply{out: mxChecks(check("skipping", "deploy", "SKIPPED"))},
			"org: merge: every one of the 1 checks of PR #7 is skipping and none passed, so nothing shows that the PR's CI passed " +
				"and nothing was merged: let a check run on the PR, and once it passed, " + mxAgain, mergeRefusalChecksNotPassed},
		{"one pending", mgReply{out: mxChecks(mxPass("verify"), check("pending", "lint", "IN_PROGRESS"))},
			notPassing(1, 2, "ci / lint (pending, IN_PROGRESS)"), mergeRefusalChecksPending},
		{"no checks", mgReply{out: "[]"}, noChecks, mergeRefusalChecksZero},
		{"exit 8 with a pending check on stdout", mgReply{out: mxChecks(check("pending", "lint", "QUEUED"), mxPass("verify")),
			err: errors.New("exit status 8")}, notPassing(1, 2, "ci / lint (pending, QUEUED)"), mergeRefusalChecksPending},
		{"a non-zero exit with checks that pass on stdout reads them", mgReply{out: mxChecks(mxPass("verify")),
			err: errors.New("exit status 8")}, "", mergeRefusalOther},
		{"exit 1 with gh's sentence of no checks for the branch", mgReply{
			err: errors.New("exit status 1: no checks reported on the 'feat/a' branch")}, noChecks, mergeRefusalChecksZero},
		{"exit 1 with gh's sentence of no checks for another branch", mgReply{
			err: errors.New("exit status 1: no checks reported on the 'feat/a2' branch")},
			ghFailed("exit status 1: no checks reported on the 'feat/a2' branch"), mergeRefusalChecksUnread},
		{"exit 1 with another error", mgReply{err: errors.New("exit status 1: HTTP 502: Bad Gateway")},
			ghFailed("exit status 1: HTTP 502: Bad Gateway"), mergeRefusalChecksUnread},
		{"exit 0 with gh's sentence on stdout", mgReply{out: "no checks reported on the 'feat/a' branch\n"}, "org: merge: " + cmd +
			" printed output that is not a JSON list of checks, so the PR's checks are unknown and nothing was merged: " + mxAccess,
			mergeRefusalChecksUnread},
		{"exit 0 with output that is not JSON", mgReply{out: "<html>"}, "org: merge: " + cmd + " printed output that is not a " +
			"JSON list of checks, so the PR's checks are unknown and nothing was merged: " + mxAccess, mergeRefusalChecksUnread},
		{"null is not a list", mgReply{out: "null"}, "org: merge: " + cmd + " printed output that is not a JSON list of checks, so " +
			"the PR's checks are unknown and nothing was merged: " + mxAccess, mergeRefusalChecksUnread},
		{"an object is not a list", mgReply{out: `{"bucket":"pass"}`}, "org: merge: " + cmd + " printed output that is not a JSON " +
			"list of checks, so the PR's checks are unknown and nothing was merged: " + mxAccess, mergeRefusalChecksUnread},
		{"fail and cancel", mgReply{out: mxChecks(mxPass("verify"), check("fail", "test", "FAILURE"), check("cancel", "e2e", "CANCELLED")),
			err: errors.New("exit status 1")}, notPassing(2, 3, "ci / test (fail, FAILURE), ci / e2e (cancel, CANCELLED)"),
			mergeRefusalChecksNotPassed},
		{"cancel only", mgReply{out: mxChecks(check("cancel", "e2e", "CANCELLED"))}, notPassing(1, 1, "ci / e2e (cancel, CANCELLED)"),
			mergeRefusalChecksNotPassed},
		{"pending and fail", mgReply{out: mxChecks(check("pending", "lint", "QUEUED"), check("fail", "test", "FAILURE")),
			err: errors.New("exit status 1")}, notPassing(2, 2, "ci / lint (pending, QUEUED), ci / test (fail, FAILURE)"),
			mergeRefusalChecksNotPassed},
		{"a bucket not known", mgReply{out: mxChecks(mxPass("verify"), ghPRCheck{Name: "x"}, ghPRCheck{Bucket: "PASS", Name: "y",
			State: "SUCCESS", Workflow: "y"})}, notPassing(2, 3, "x (no bucket, no state), y (PASS, SUCCESS)"), mergeRefusalChecksNotPassed},
		{"more than five", mgReply{out: mxChecks(append([]ghPRCheck{mxPass("verify")}, many...)...)},
			notPassing(7, 8, strings.Join(manyNamed[:5], ", ")+" and 2 more"), mergeRefusalChecksPending},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMXFixture(t, nil)
			fx.exec.replies[mxPRChecks] = tc.reply
			r, refused, kind, err := fx.checkKind(context.Background(), t, fx.params())
			switch {
			case err != nil:
				t.Fatalf("checkMerge error: %v", err)
			case tc.want == "" && (refused != "" || r.prNumber != 7):
				t.Fatalf("checkMerge = %+v, refused %q; want it to pass", r, refused)
			case tc.want != "" && refused != tc.want:
				t.Fatalf("refused %q\nwant    %q", refused, tc.want)
			case kind != tc.kind:
				t.Errorf("kind = %d, want %d", kind, tc.kind)
			case tc.want != "" && kind.waits() && (r.branch != "feat/a" || r.prNumber != 7 || r.repoArg != mxRepoArg || r.head != moHeadA):
				t.Errorf("checkMerge = %+v with a refusal --wait-checks waits out; want what it read of PR #7", r)
			case tc.want != "" && !kind.waits() && r != (mergeReady{}):
				t.Errorf("checkMerge = %+v with a refusal --wait-checks does not wait out; want it empty", r)
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
		_, refused, kind, err := fx.checkKind(context.Background(), t, p)
		want := "org: merge: " + cmd + " failed (timed out after 30ms: context deadline exceeded), so the PR's checks are unknown " +
			"and nothing was merged: " + mxAccess
		if err != nil || refused != want || kind != mergeRefusalChecksUnread {
			t.Fatalf("checkMerge = %v, kind %d\nrefused %q\nwant    %q", err, kind, refused, want)
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

// TestMergeOrigin_ReadsOriginsRepositoryOrSaysWhyNot covers the reading
// of origin that ralph org merge (readOrigin) and ralph org cleanup share
// (S-X1 of plan docs/plans/active/2026-10-28-org-merge-cleanup.md): an https,
// scp-like, or ssh URL gives its host, lowercased, with ssh.github.com read as
// github.com, and its owner/repo, as Arg <host>/<owner>/<repo>, from a
// get-url run in dir with GIT_TERMINAL_PROMPT=0; a local path, a file URL, a
// path with no owner, and a host, owner or repo that starts with "-" are
// MergeOriginNoRepository, with the zero MergeOrigin, whose Arg is "".
func TestMergeOrigin_ReadsOriginsRepositoryOrSaysWhyNot(t *testing.T) {
	for _, tc := range []struct {
		name, url, wantArg string
		wantProblem        MergeOriginProblem
	}{
		{"https", "https://github.com/o/r.git", "github.com/o/r", MergeOriginRead},
		{"scp-like", "git@github.com:o/r.git", "github.com/o/r", MergeOriginRead},
		{"ssh.github.com", "ssh://git@ssh.github.com:443/o/r.git", "github.com/o/r", MergeOriginRead},
		{"GitHub Enterprise in other case", "git@GHE.example.com:Team/App.git", "ghe.example.com/Team/App", MergeOriginRead},
		{"a local path", "/srv/git/r.git", "", MergeOriginNoRepository},
		{"a file URL", "file:///srv/o/r.git", "", MergeOriginNoRepository},
		{"no owner", "https://github.com/o", "", MergeOriginNoRepository},
		{"a host that starts with -", "ssh://git@-x.example.com/o/r.git", "", MergeOriginNoRepository},
		{"an owner that starts with -", "git@github.com:-o/r.git", "", MergeOriginNoRepository},
		{"a repo that starts with -", "https://github.com/o/-r.git", "", MergeOriginNoRepository},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := &moExec{origin: tc.url}
			o, problem, err := ReadMergeOrigin(context.Background(), "/repo", e.run, time.Second)
			if problem != tc.wantProblem || o.Arg() != tc.wantArg || err != nil {
				t.Errorf("ReadMergeOrigin = %q, %d, %v; want %q, %d, no error", o.Arg(), problem, err, tc.wantArg, tc.wantProblem)
			}
			if problem != MergeOriginRead && o != (MergeOrigin{}) {
				t.Errorf("origin = %+v with problem %d, want the zero MergeOrigin", o, problem)
			}
			calls := e.callsOf("git")
			if len(calls) != 1 || !slices.Equal(calls[0].args, []string{"remote", "get-url", "origin"}) || calls[0].dir != "/repo" ||
				!slices.Equal(calls[0].env, []string{"GIT_TERMINAL_PROMPT=0"}) {
				t.Errorf("git calls = %+v, want one git remote get-url origin in /repo with GIT_TERMINAL_PROMPT=0", calls)
			}
		})
	}
}

// TestMergeOrigin_GetURLThatFailsTimesOutOrIsInterrupted: a get-url that
// fails is MergeOriginGetURLFailed with its error; one that runs past timeout
// is MergeOriginTimedOut with its error after "timed out after <timeout>: ";
// ctx done is MergeOriginInterrupted with no error; each with the zero
// MergeOrigin. A timeout of zero or less is 15 s.
func TestMergeOrigin_GetURLThatFailsTimesOutOrIsInterrupted(t *testing.T) {
	block := func(ctx context.Context, _ string, _ []string, _ string, _ ...string) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	}
	t.Run("fails", func(t *testing.T) {
		gitErr := errors.New("exit status 2: error: No such remote 'origin'")
		e := &moExec{originErr: gitErr}
		o, problem, err := ReadMergeOrigin(context.Background(), "/repo", e.run, time.Second)
		if problem != MergeOriginGetURLFailed || !errors.Is(err, gitErr) || o != (MergeOrigin{}) {
			t.Errorf("ReadMergeOrigin = %+v, %d, %v; want the zero MergeOrigin, MergeOriginGetURLFailed, %v", o, problem, err, gitErr)
		}
	})
	t.Run("times out", func(t *testing.T) {
		o, problem, err := ReadMergeOrigin(context.Background(), "/repo", block, 20*time.Millisecond)
		if problem != MergeOriginTimedOut || err == nil || err.Error() != "timed out after 20ms: context deadline exceeded" ||
			!errors.Is(err, context.DeadlineExceeded) || o != (MergeOrigin{}) {
			t.Errorf("ReadMergeOrigin = %+v, %d, %v; want the zero MergeOrigin, MergeOriginTimedOut, and the timeout", o, problem, err)
		}
	})
	t.Run("interrupted", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		o, problem, err := ReadMergeOrigin(ctx, "/repo", block, time.Second)
		if problem != MergeOriginInterrupted || err != nil || o != (MergeOrigin{}) {
			t.Errorf("ReadMergeOrigin = %+v, %d, %v; want the zero MergeOrigin, MergeOriginInterrupted, no error", o, problem, err)
		}
	})
	for _, timeout := range []time.Duration{0, -time.Second} {
		t.Run(fmt.Sprintf("timeout %s", timeout), func(t *testing.T) {
			var deadline time.Time
			run := func(ctx context.Context, _ string, _ []string, _ string, _ ...string) (string, error) {
				deadline, _ = ctx.Deadline()
				return moOriginURL + "\n", nil
			}
			start := time.Now()
			if _, problem, err := ReadMergeOrigin(context.Background(), "/repo", run, timeout); problem != MergeOriginRead || err != nil {
				t.Fatalf("ReadMergeOrigin = %d, %v; want origin read", problem, err)
			}
			if end := time.Now(); deadline.Before(start.Add(15*time.Second)) || deadline.After(end.Add(15*time.Second)) {
				t.Errorf("deadline %v is not 15 s after the start (%v..%v)", deadline, start, end)
			}
		})
	}
}
