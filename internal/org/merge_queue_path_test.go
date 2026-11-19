package org

import (
	"context"
	"slices"
	"testing"
	"time"
)

// The queue path of checkMerge's steps 11 and 12 (plan
// docs/plans/active/2026-11-20-org-merge-queue-verb.md, S2; FR-1 to FR-3 of
// spec docs/specs/2026-11-19-org-merge-queue.md) with mxExec and mrExec, the
// fakes of merge_fake_test.go: a default branch with a merge queue and a
// required status check passes the checks with mergeReady.queue set and the
// method the queue's configuration gives, step 12 not run; with no check
// required it is refused (AC6); and Merge refuses the queue path before gh
// pr merge, so nothing merges through the queue in this build. No test runs
// gh or reaches a network.

// mxBranch is the key of the read of the branch protection's required
// checks, which step 11 makes on the queue path when the rules require none.
const mxBranch = "gh api repos/o/r/branches/main"

// mxQueuePath are the commands checkMerge runs when every check passes on the
// queue path with checks the rules require: mxAll without the ruleset's
// bypass read and without gh repo view (step 12).
var mxQueuePath = []string{mxGetURL, mxLsRemote, mxPRList, mxPRView, mxPRChecks, mxRepoAPI, mxQueue, mxRules, mxLsRemote}

// mxQueueJSON is a GraphQL answer whose repository.mergeQueue is set, with
// config as its configuration's JSON.
func mxQueueJSON(config string) string {
	return meQueueJSON(`{"id":"MQ_kwDOA","configuration":` + config + `}`)
}

// mxQueueProtection is the Detail of an accepted queue path, how being what
// showed the queue and required what requires the checks.
func mxQueueProtection(how, required string) string {
	return mxSubject + ", has a merge queue (" + how + "), and " + required + ": the queue merges the PR only once its " +
		"required checks pass on it with the latest main"
}

// newMXQueueFixture is newMXFixture on mxReadyLedger with a merge queue whose
// configuration is config, and no answer for gh repo view, so a step 12 that
// ran would be refused.
func newMXQueueFixture(t *testing.T, config string) *mxFixture {
	t.Helper()
	fx := newMXFixture(t, nil)
	fx.exec.replies[mxQueue] = mgReply{out: mxQueueJSON(config)}
	delete(fx.exec.replies, mxRepoView)
	return fx
}

// TestCheckMerge_TakesTheQueuePath covers FR-1 to FR-3: a merge queue that
// GraphQL or a ruleset's merge_queue rule shows, with a status check the rules
// or the branch protection require, passes every check with mergeReady.queue
// set, method the queue's configuration's in lower case or queue when
// GraphQL gave none, no methodFlag, and the queue's Detail as the protection;
// neither the ruleset's bypass nor gh repo view (step 12) is read, every
// command is read-only and pinned to origin's repository, and the refusal kind
// is none.
func TestCheckMerge_TakesTheQueuePath(t *testing.T) {
	const graphQL, rule = "GraphQL mergeQueue is set", "a ruleset's merge_queue rule"
	const inRules = "rulesets require 1 required status check"
	for _, tc := range []struct {
		name       string
		config     string // the queue's configuration in GraphQL; "" for no queue there
		setup      func(fx *mxFixture)
		method     string
		protection string
		commands   []string
	}{
		{"GraphQL method SQUASH", `{"mergeMethod":"SQUASH"}`, func(*mxFixture) {}, "squash", mxQueueProtection(graphQL, inRules),
			mxQueuePath},
		{"GraphQL method MERGE", `{"mergeMethod":"MERGE"}`, func(*mxFixture) {}, "merge", mxQueueProtection(graphQL, inRules),
			mxQueuePath},
		{"GraphQL method REBASE", `{"mergeMethod":"REBASE"}`, func(*mxFixture) {}, "rebase", mxQueueProtection(graphQL, inRules),
			mxQueuePath},
		{"GraphQL configuration null", "null", func(*mxFixture) {}, "queue", mxQueueProtection(graphQL, inRules), mxQueuePath},
		{"GraphQL mergeMethod null", `{"mergeMethod":null}`, func(*mxFixture) {}, "queue", mxQueueProtection(graphQL, inRules),
			mxQueuePath},
		{"a ruleset's merge_queue rule only", "", func(fx *mxFixture) {
			fx.exec.replies[mxRules] = mgReply{out: mgRules(mgMergeQueueRule, meChecksRule(1, true, "verify"))}
		}, "queue", mxQueueProtection(rule, inRules), mxQueuePath},
		{"checks of the branch protection only", `{"mergeMethod":"SQUASH"}`, func(fx *mxFixture) {
			fx.exec.replies[mxRules] = mgReply{out: "[]"}
			fx.exec.replies[mxBranch] = mgReply{out: mgClassicRequires("verify")}
		}, "squash", mxQueueProtection(graphQL, "the branch protection requires 1 required status check"),
			[]string{mxGetURL, mxLsRemote, mxPRList, mxPRView, mxPRChecks, mxRepoAPI, mxQueue, mxRules, mxBranch, mxLsRemote}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMXQueueFixture(t, tc.config)
			if tc.config == "" {
				fx.exec.replies[mxQueue] = mgReply{out: meQueueJSON("null")}
			}
			tc.setup(fx)
			r, refused, kind, err := fx.checkKind(context.Background(), t, fx.params())
			if err != nil || refused != "" || kind != mergeRefusalOther {
				t.Fatalf("checkMerge = %v, kind %d, refused %q", err, kind, refused)
			}
			want := mergeReady{
				branch: "feat/a", orgID: "org-a", apiHost: "github.com", repo: mergeRepo{owner: "o", name: "r"}, repoArg: mxRepoArg,
				originHost: "github.com", prNumber: 7, prURL: "https://github.com/o/r/pull/7", head: moHeadA, base: moMain,
				defaultBranch: "main", queue: true, method: tc.method, protection: tc.protection,
			}
			if r != want {
				t.Fatalf("mergeReady =\n%+v\nwant\n%+v", r, want)
			}
			if got := fx.exec.keys(); !slices.Equal(got, tc.commands) {
				t.Errorf("commands = %q\nwant %q", got, tc.commands)
			}
			mxAssertPinned(t, fx.exec, "github.com", mxRepoArg)
		})
	}
}

// TestCheckMerge_QueueWithNoRequiredCheckIsRefused covers AC6: a merge queue
// with no status check required, by the rules or the branch protection, is
// refused at step 11 with a refusal that says the queue would merge the PR
// with no check run, of a kind --wait-checks does not wait out, with nothing
// read kept, and neither step 12 nor step 13 run.
func TestCheckMerge_QueueWithNoRequiredCheckIsRefused(t *testing.T) {
	refusal := func(how string) string {
		return mxProtect + mxSubject + ", has a merge queue (" + how + "), but no status check is required, so the queue would " +
			"merge the PR with no check run on it: require at least one status check on the branch, in a ruleset or in the branch " +
			"protection, or merge the PR through the queue yourself"
	}
	for _, tc := range []struct {
		name   string
		config string // the queue's configuration in GraphQL; "" for no queue there
		rules  string
		want   string
	}{
		{"merge queue in GraphQL", `{"mergeMethod":"SQUASH"}`, "[]", refusal("GraphQL mergeQueue is set")},
		{"a ruleset's merge_queue rule", "", mgRules(mgMergeQueueRule), refusal("a ruleset's merge_queue rule")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMXQueueFixture(t, tc.config)
			if tc.config == "" {
				fx.exec.replies[mxQueue] = mgReply{out: meQueueJSON("null")}
			}
			fx.exec.replies[mxRules] = mgReply{out: tc.rules}
			fx.exec.replies[mxBranch] = mgReply{out: mgClassicRequires()}
			r, refused, kind, err := fx.checkKind(context.Background(), t, fx.params())
			if err != nil || refused != tc.want || r != (mergeReady{}) || kind.waits() {
				t.Fatalf("checkMerge = %+v, %v, kind %d\nrefused %q\nwant    %q", r, err, kind, refused, tc.want)
			}
			want := []string{mxGetURL, mxLsRemote, mxPRList, mxPRView, mxPRChecks, mxRepoAPI, mxQueue, mxRules, mxBranch}
			if got := fx.exec.keys(); !slices.Equal(got, want) {
				t.Errorf("commands = %q\nwant %q", got, want)
			}
		})
	}
}

// TestMerge_QueuePathMergesNothingInThisBuild: once the checks pass on the
// queue path, Merge refuses before gh pr merge, which on a branch with a merge
// queue adds the PR to the queue or makes an auto-merge request apart from
// ralph: no gh pr merge, no read-back, no gh repo view, nothing appended to
// the manifest, and regate.lock released; with --wait-checks too, which waits
// for nothing then.
func TestMerge_QueuePathMergesNothingInThisBuild(t *testing.T) {
	const want = "org: merge: main, the default branch of origin's repository o/r, has a merge queue, and this build of ralph " +
		"org merge does not merge through a merge queue, so nothing was merged: merge PR #7 through the queue yourself"
	for _, wait := range []time.Duration{0, 30 * time.Minute} {
		t.Run("wait-checks "+wait.String(), func(t *testing.T) {
			fx := newMRFixture(t, nil)
			fx.exec.replies[mxQueue] = mgReply{out: mxQueueJSON(`{"mergeMethod":"SQUASH"}`)}
			delete(fx.exec.replies, mxRepoView)
			p := fx.params()
			p.WaitChecks = wait
			res, err := fx.merge(context.Background(), t, p)
			if err != nil || res != (MergeResult{Refused: want}) {
				t.Fatalf("Merge = %+v, %v\nwant refused %q", res, err, want)
			}
			if got := fx.exec.keys(); !slices.Equal(got, mxQueuePath) {
				t.Errorf("commands = %q\nwant %q", got, mxQueuePath)
			}
			if got := fx.appended(t); got != "" {
				t.Errorf("appended %q, want nothing", got)
			}
			mxAssertReadOnly(t, fx.exec)
			mrAssertLockFree(t, fx.mo.stateDir)
		})
	}
}
