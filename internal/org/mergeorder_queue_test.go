package org

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

// The merge queue of origin's default branch in merge-order's classification,
// and its read outside CheckMergeEnforcement (slice S1 of plan
// docs/plans/active/2026-11-20-org-merge-queue-order.md: the classification
// part of AC1, and the one read AC2 and AC7 rest on). The classification
// tests build mergeTarget and mergeRemote as collectMergeTargets and
// listOriginBranches would; the read tests run on mgExec through newMEExec
// (mergeenforce_test.go). No test runs gh or reaches a network.

// moqUnreadWhy is the why of a mergeQueueUnknown read in the classification
// tests: what readMergeQueue says when gh's login lapsed.
const moqUnreadWhy = "gh api --hostname github.com graphql (mergeQueue of main) failed (exit status 1: gh: HTTP 401: Bad " +
	"credentials), so whether " + mgSubject + ", has a merge queue is unknown" + meAccess

// moqQueueReads are the four reads of the merge queue the classification
// takes, by name.
var moqQueueReads = []struct {
	name string
	read mergeQueueLookup
}{
	{"not read", mergeQueueLookup{}},
	{"absent", mergeQueueLookup{state: mergeQueueAbsent}},
	{"present", mergeQueueLookup{state: mergeQueuePresent, method: "SQUASH"}},
	{"unknown", mergeQueueLookup{state: mergeQueueUnknown, why: moqUnreadWhy}},
}

// moqTarget is feat/auth, bound to org-a, with the readable gate record r of
// org-a: a pass when r has no reason, else a failure, which alice accepted
// when accepted is set. org-a runs unless disbanded is set (then the branch's
// Details name ralph org regate). pr is the branch's PR, nil for none.
func moqTarget(r gateRecord, accepted, disbanded bool, pr *MergeOrderPR) *mergeTarget {
	ev := ManifestEvent{TS: "2026-10-14T07:00:00Z", OrgID: "org-a", Event: EventGatePassed}
	if r.Reason != "" {
		ev.Event = EventGateFailed
	}
	t := &mergeTarget{
		branch: "feat/auth", binding: &MergeOrderBinding{OrgID: "org-a", Feature: "auth"}, orgRunning: !disbanded,
		gate: &mergeGateRecord{ev: ev, record: r, readable: true}, pr: pr,
	}
	if accepted {
		t.accepted = &GateAcceptance{By: "alice", TS: "2026-10-14T07:10:00Z", Note: "the failing test is outside the change"}
	}
	return t
}

// TestClassifyMergeTarget_MergeQueueKeepsABranchWhoseBaseMovedReady covers
// the classification part of AC1: with a merge queue (mergeQueuePresent), a
// pass, or an accepted failure, whose base_ref names origin's default branch
// and whose base is not that branch's tip is ready (accepted), its Detail
// saying the queue tests the PR on the latest tip before it merges it; with
// no queue or no read it is rebase with today's Detail, and with a read that
// failed it is rebase with today's Detail and a clause that says the queue
// could not be read and why. A base_ref of another branch stays rebase, a
// branch tip that is not the gate's head stays regate, a PR into another base
// stays wrong_base, and a base that is the tip stays ready, each with the
// Detail it has with no read, whatever the queue.
func TestClassifyMergeTarget_MergeQueueKeepsABranchWhoseBaseMovedReady(t *testing.T) {
	const (
		movedReady = "the gate passed on aaaaaaaaaaaa, origin's tip of the branch, with base 111111111111; origin/main moved to " +
			"222222222222 since, and its merge queue tests the PR on the latest origin/main before it merges it, so no regate is " +
			"needed"
		movedAccepted = "the gate failed (test_failed) on aaaaaaaaaaaa, origin's tip of the branch, with base 111111111111; " +
			"origin/main moved to 222222222222 since, and its merge queue tests the PR on the latest origin/main before it merges " +
			"it, so no regate is needed; alice accepted the failure at 2026-10-14T07:10:00Z: the failing test is outside the change"
		unreadClause = "; the merge queue of origin/main could not be read, and with one the branch would be ready: " + moqUnreadWhy
	)
	onMain := gateRecord{Head: moHeadA, Base: moMain, BaseRef: "refs/remotes/origin/main", Branch: "feat/auth"}
	onDevelop := onMain
	onDevelop.BaseRef = "refs/remotes/origin/develop"
	for _, sc := range []struct {
		name       string
		record     gateRecord
		disbanded  bool
		pr         *MergeOrderPR
		branchTip  string
		mainTip    string
		moved      bool   // the queue decides: ready with a queue, else rebase
		class      string // with no read, for a pass
		passed     string // a part of the Detail with no read, for a pass
		acceptedIn string // a part of the Detail with no read, for an accepted failure
	}{
		{"base is the tip", onMain, false, nil, moHeadA, moMain, false, MergeClassReady,
			"the gate passed on aaaaaaaaaaaa, origin's tip of the branch, with base 111111111111, origin/main's tip", moAcceptedDetail},
		{"base moved", onMain, false, nil, moHeadA, moMain2, true, MergeClassRebase,
			"origin/main moved: gate base 111111111111, now 222222222222: rebase on origin/main and run ralph org gate --org-id org-a " +
				"again " + mergeGateRunsWhileOrgRuns, ""},
		{"base moved, org disbanded", onMain, true, nil, moHeadA, moMain2, true, MergeClassRebase,
			"origin/main moved: gate base 111111111111, now 222222222222: run ralph org regate --branch feat/auth, which merges " +
				"origin/main in and runs the gate again", ""},
		{"base moved, PR into main", onMain, false, &MergeOrderPR{Number: 12, State: "OPEN", BaseRefName: "main"}, moHeadA, moMain2,
			true, MergeClassRebase, "origin/main moved: gate base 111111111111, now 222222222222", ""},
		{"base_ref names another branch", onDevelop, false, nil, moHeadA, moMain2, false, MergeClassRebase,
			"the gate checked against refs/remotes/origin/develop, but origin's default branch is now main", ""},
		{"base_ref names another branch at the tip", onDevelop, false, nil, moHeadA, moMain, false, MergeClassRebase,
			"the gate checked against refs/remotes/origin/develop, but origin's default branch is now main", ""},
		{"branch tip is not the gate's head", onMain, false, nil, moHeadB, moMain2, false, MergeClassRegate,
			"pushed after the gate: origin has bbbbbbbbbbbb, the gate checked aaaaaaaaaaaa", ""},
		{"PR into another base", onMain, false, &MergeOrderPR{Number: 12, State: "OPEN", BaseRefName: "develop"}, moHeadA, moMain2,
			false, MergeClassWrongBase, "PR #12 merges into develop, not origin's default branch main", ""},
	} {
		for _, accepted := range []bool{false, true} {
			record := sc.record
			if accepted {
				record.Reason = GateReasonTestFailed
			}
			remote := mergeRemote{defaultBranch: "main", defaultHead: sc.mainTip,
				heads: map[string]string{"main": sc.mainTip, "feat/auth": sc.branchTip}}
			baseClass, baseDetail := classifyMergeTarget(moqTarget(record, accepted, sc.disbanded, sc.pr), remote)
			wantClass := sc.class
			if accepted && wantClass == MergeClassReady {
				wantClass = MergeClassAccepted
			}
			want := sc.passed
			if accepted && sc.acceptedIn != "" {
				want = sc.acceptedIn
			}
			if baseClass != wantClass || !strings.Contains(baseDetail, want) || strings.Contains(baseDetail, "merge queue") {
				t.Fatalf("%s (accepted %v) with no read = %s %q\nwant %s with %q and no merge queue", sc.name, accepted, baseClass,
					baseDetail, wantClass, want)
			}
			for _, q := range moqQueueReads {
				t.Run(fmt.Sprintf("%s, accepted %v, queue %s", sc.name, accepted, q.name), func(t *testing.T) {
					remote.queue = q.read
					class, detail := classifyMergeTarget(moqTarget(record, accepted, sc.disbanded, sc.pr), remote)
					wantClass, wantDetail := baseClass, baseDetail
					switch {
					case sc.moved && q.read.state == mergeQueuePresent && accepted:
						wantClass, wantDetail = MergeClassAccepted, movedAccepted
					case sc.moved && q.read.state == mergeQueuePresent:
						wantClass, wantDetail = MergeClassReady, movedReady
					case sc.moved && q.read.state == mergeQueueUnknown:
						wantDetail = baseDetail + unreadClause
					}
					if class != wantClass || detail != wantDetail {
						t.Errorf("class %s, Detail %q\nwant %s, %q", class, detail, wantClass, wantDetail)
					}
				})
			}
		}
	}
}

// TestMergeEnforcement_TakesAQueueReadAlreadyMade covers the one read of AC2
// and AC7 (FR-12): CheckMergeEnforcement given a read lookupMergeQueue made
// (MergeEnforcementParams.queue) runs no gh api graphql of its own, after
// step 1 as before, and goes on as that read says, whatever GraphQL would
// answer now: a queue is the queue path with the read's method, no queue
// reads the rules (where a merge_queue rule is still the queue path), and a
// read that failed is not ok with the read's Detail. With no read it reads
// the queue itself, as before.
func TestMergeEnforcement_TakesAQueueReadAlreadyMade(t *testing.T) {
	queueOK := func(how string) string {
		return mgSubject + ", has a merge queue (" + how + "), and rulesets require 1 required status check: the queue merges " +
			"the PR only once its required checks pass on it with the latest main"
	}
	never := `the ruleset "protect main" (id 1) of ` + mgSubject + `, requires branches to be up to date before merging with its ` +
		`1 required status check, and you cannot bypass it (current_user_can_bypass: never)`
	const otherDefault = `gh api --hostname github.com repos/acme/app returned the default branch "trunk", but origin's HEAD names ` +
		"main, so which branch's protection to read is unknown: run ralph org merge-order again"
	aQueue := mgReply{out: meQueueJSON(`{"id":"MQ_kwDOA","configuration":{"mergeMethod":"MERGE"}}`)}
	for _, tc := range []struct {
		name      string
		read      mergeQueueLookup
		setup     func(e *mgExec)
		ok, queue bool
		method    string
		detail    string
		endpoints []string
	}{
		{"present with a method", mergeQueueLookup{state: mergeQueuePresent, method: "SQUASH"}, func(*mgExec) {}, true, true, "SQUASH",
			queueOK("GraphQL mergeQueue is set"), []string{mgRepoEP, mgRulesEP}},
		{"present with no method", mergeQueueLookup{state: mergeQueuePresent}, func(*mgExec) {}, true, true, "",
			queueOK("GraphQL mergeQueue is set"), []string{mgRepoEP, mgRulesEP}},
		{"absent while GraphQL shows a queue now", mergeQueueLookup{state: mergeQueueAbsent},
			func(e *mgExec) { e.replies[meQueueEP] = aQueue }, true, false, "", never, []string{mgRepoEP, mgRulesEP, meRuleset1EP}},
		{"absent with a merge_queue rule", mergeQueueLookup{state: mergeQueueAbsent}, func(e *mgExec) {
			e.replies[mgRulesEP] = mgReply{out: mgRules(mgMergeQueueRule, meChecksRule(1, true, "ci"))}
		}, true, true, "", queueOK("a ruleset's merge_queue rule"), []string{mgRepoEP, mgRulesEP}},
		{"unknown", mergeQueueLookup{state: mergeQueueUnknown, why: moqUnreadWhy}, func(e *mgExec) { e.replies[meQueueEP] = aQueue },
			false, false, "", moqUnreadWhy, []string{mgRepoEP}},
		{"present, but step 1 refuses first", mergeQueueLookup{state: mergeQueuePresent, method: "SQUASH"}, func(e *mgExec) {
			e.replies[mgRepoEP] = mgReply{out: meRepoJSON("trunk", "acme/app", nil, false)}
		}, false, false, "", otherDefault, []string{mgRepoEP}},
		{"not read", mergeQueueLookup{}, func(e *mgExec) { e.replies[meQueueEP] = aQueue }, true, true, "MERGE",
			queueOK("GraphQL mergeQueue is set"), []string{mgRepoEP, meQueueEP, mgRulesEP}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newMEExec()
			e.replies[meQueueEP] = mgReply{out: meQueueJSON("null")}
			tc.setup(e)
			p := meParams(e)
			p.queue = tc.read
			got := CheckMergeEnforcement(context.Background(), p)
			if got.OK != tc.ok || got.Queue != tc.queue || got.QueueMethod != tc.method || got.Detail != tc.detail {
				t.Errorf("enforcement = %+v\nwant ok %v, queue %v, method %q, %q", got, tc.ok, tc.queue, tc.method, tc.detail)
			}
			if eps := e.endpoints(); !slices.Equal(eps, tc.endpoints) {
				t.Errorf("gh endpoints = %q, want %q", eps, tc.endpoints)
			}
			meAssertGHCalls(t, e, "github.com", "acme", "app", "main")
		})
	}
}

// TestMergeQueueLookup_ReadsTheQueueWithTheEnforcementChecksTexts: the read
// of the merge queue outside CheckMergeEnforcement (lookupMergeQueue) runs
// one gh api graphql, the call of step 2, and nothing else, and returns
// mergeQueueAbsent for a null mergeQueue, mergeQueuePresent with the
// configuration's merge method as step 2 reads it, and mergeQueueUnknown for
// every answer step 2 refuses, with the Detail CheckMergeEnforcement gives
// that answer; it never returns mergeQueueNotRead.
func TestMergeQueueLookup_ReadsTheQueueWithTheEnforcementChecksTexts(t *testing.T) {
	const queueCmd = "gh api --hostname github.com graphql (mergeQueue of main)"
	queue := func(config string) mgReply {
		return mgReply{out: meQueueJSON(`{"id":"MQ_kwDOA","configuration":` + config + `}`)}
	}
	for _, tc := range []struct {
		name   string
		reply  mgReply
		state  mergeQueueState
		method string
		why    string // the exact why of an unknown read, "" to compare it with CheckMergeEnforcement's Detail alone
	}{
		{"null", mgReply{out: meQueueJSON("null")}, mergeQueueAbsent, "", ""},
		{"errors null", mgReply{out: `{"data":{"repository":{"mergeQueue":null}},"errors":null}`}, mergeQueueAbsent, "", ""},
		{"set with no configuration", mgReply{out: meQueueJSON(`{"id":"MQ_kwDOA"}`)}, mergeQueuePresent, "", ""},
		{"method SQUASH", queue(`{"mergeMethod":"SQUASH"}`), mergeQueuePresent, "SQUASH", ""},
		{"method MERGE", queue(`{"mergeMethod":"MERGE"}`), mergeQueuePresent, "MERGE", ""},
		{"method REBASE", queue(`{"mergeMethod":"REBASE"}`), mergeQueuePresent, "REBASE", ""},
		{"method in lower case", queue(`{"mergeMethod":"squash"}`), mergeQueuePresent, "", ""},
		{"configuration null", queue("null"), mergeQueuePresent, "", ""},
		{"gh fails", mgReply{err: errors.New("exit status 1: gh: HTTP 401: Bad credentials")}, mergeQueueUnknown, "",
			queueCmd + " failed (exit status 1: gh: HTTP 401: Bad credentials), so whether " + mgSubject + ", has a merge queue " +
				"is unknown" + meAccess},
		{"errors", mgReply{out: `{"data":{"repository":{"mergeQueue":null}},"errors":[{"message":"x"}]}`}, mergeQueueUnknown, "",
			queueCmd + ` answered with errors ([{"message":"x"}]), so whether ` + mgSubject + ", has a merge queue is unknown" +
				meAccess},
		{"not JSON", mgReply{out: "<html>"}, mergeQueueUnknown, "", ""},
		{"no repository", mgReply{out: `{"data":{"repository":null}}`}, mergeQueueUnknown, "", ""},
		{"no mergeQueue", mgReply{out: `{"data":{"repository":{}}}`}, mergeQueueUnknown, "", ""},
		{"mergeQueue not an object", mgReply{out: meQueueJSON("true")}, mergeQueueUnknown, "", ""},
		{"configuration not an object", queue(`"SQUASH"`), mergeQueueUnknown, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newMEExec()
			e.replies[meQueueEP] = tc.reply
			got := lookupMergeQueue(context.Background(), meParams(e))
			if got.state != tc.state || got.method != tc.method {
				t.Fatalf("lookup = %+v, want state %q, method %q", got, tc.state, tc.method)
			}
			if eps := e.endpoints(); !slices.Equal(eps, []string{meQueueEP}) {
				t.Errorf("gh endpoints = %q, want only %s", eps, meQueueEP)
			}
			meAssertGHCalls(t, e, "github.com", "acme", "app", "main")
			if tc.state != mergeQueueUnknown {
				if got.why != "" {
					t.Errorf("why = %q, want none", got.why)
				}
				return
			}
			if tc.why != "" && got.why != tc.why {
				t.Errorf("why = %q\nwant %q", got.why, tc.why)
			}
			enf := newMEExec()
			enf.replies[meQueueEP] = tc.reply
			if check := CheckMergeEnforcement(context.Background(), meParams(enf)); check.OK || got.why != check.Detail {
				t.Errorf("why = %q\nCheckMergeEnforcement's Detail for the same answer = %q (ok %v)", got.why, check.Detail, check.OK)
			}
			if got.why == "" || strings.ContainsAny(got.why, "\n\r") {
				t.Errorf("why %q is empty or not one line", got.why)
			}
		})
	}
}

// TestMergeQueueLookup_PinsTheHostTimeoutAndContext: the read carries
// --hostname, origin's host lowercased with ssh.github.com read as github.com,
// and the variables as they are; it runs under p.Timeout from its start (20 s
// when zero or less), and a read that outlasts it, or a caller's ctx done, is
// unknown with the texts step 2 of CheckMergeEnforcement gives them.
func TestMergeQueueLookup_PinsTheHostTimeoutAndContext(t *testing.T) {
	for _, tc := range []struct{ host, apiHost string }{
		{"github.com", "github.com"}, {"ssh.github.com", "github.com"}, {"GHE.Example.com", "ghe.example.com"},
	} {
		t.Run("host "+tc.host, func(t *testing.T) {
			e := newMEExec()
			p := meParams(e)
			p.Host, p.Owner, p.DefaultBranch = tc.host, "{owner}", "release/v1"
			if got := lookupMergeQueue(context.Background(), p); got.state != mergeQueueAbsent {
				t.Fatalf("lookup = %+v, want absent", got)
			}
			if n := len(e.callsOf("gh")); n != 1 {
				t.Errorf("%d gh calls, want 1: %q", n, e.endpoints())
			}
			meAssertGHCalls(t, e, tc.apiHost, "{owner}", "app", "release/v1")
		})
	}
	for _, tc := range []struct{ timeout, want time.Duration }{
		{5 * time.Second, 5 * time.Second}, {0, 20 * time.Second}, {-time.Second, 20 * time.Second},
	} {
		e := newMEExec()
		p := meParams(e)
		p.Timeout = tc.timeout
		start := time.Now()
		lookupMergeQueue(context.Background(), p)
		end := time.Now()
		calls := e.callsOf("gh")
		if len(calls) != 1 {
			t.Fatalf("timeout %s: %d gh calls, want 1", tc.timeout, len(calls))
		}
		if d := calls[0].deadline; d.Before(start.Add(tc.want)) || d.After(end.Add(tc.want)) {
			t.Errorf("timeout %s: deadline %v is not %s after the start (%v..%v)", tc.timeout, d, tc.want, start, end)
		}
	}

	e := newMEExec()
	e.block = meQueueEP
	p := meParams(e)
	p.Timeout = 30 * time.Millisecond
	got := lookupMergeQueue(context.Background(), p)
	want := "the check timed out after 30ms while gh api --hostname github.com graphql (mergeQueue of main) ran, so the " +
		"protection of origin's repository's default branch is unknown: check the network and gh's access"
	if got.state != mergeQueueUnknown || got.why != want {
		t.Errorf("lookup = %+v\nwant unknown with %q", got, want)
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("merge stopped"))
	got = lookupMergeQueue(ctx, meParams(newMEExec()))
	want = "the check was interrupted (merge stopped) while gh api --hostname github.com graphql (mergeQueue of main) ran, so " +
		"the protection of origin's repository's default branch is unknown"
	if got.state != mergeQueueUnknown || got.why != want {
		t.Errorf("lookup = %+v\nwant unknown with %q", got, want)
	}
}

// TestMergeQueueLookup_OneReadServesTheEnforcementCheck: a read made with
// lookupMergeQueue and handed to CheckMergeEnforcement, as ralph org merge
// hands step 7's read to step 11, leaves one gh api graphql in all.
func TestMergeQueueLookup_OneReadServesTheEnforcementCheck(t *testing.T) {
	e := newMEExec()
	e.replies[meQueueEP] = mgReply{out: meQueueJSON(`{"id":"MQ_kwDOA","configuration":{"mergeMethod":"REBASE"}}`)}
	p := meParams(e)
	p.queue = lookupMergeQueue(context.Background(), p)
	got := CheckMergeEnforcement(context.Background(), p)
	if !got.OK || !got.Queue || got.QueueMethod != "REBASE" {
		t.Fatalf("enforcement = %+v, want ok on the queue path with REBASE", got)
	}
	if eps, want := e.endpoints(), []string{meQueueEP, mgRepoEP, mgRulesEP}; !slices.Equal(eps, want) {
		t.Errorf("gh endpoints = %q, want %q (one graphql)", eps, want)
	}
}

// The read of the merge queue in MergeOrder (slice S2 of plan
// docs/plans/active/2026-11-20-org-merge-queue-order.md: AC1, and the org
// part of AC5): MergeOrder on manifest fixtures with
// MergeOrderParams.ReadMergeQueue, through moExec, whose `gh api ... graphql`
// answers queueOut or queueErr.

// moqQueueJSON is gh api graphql's answer for a merge queue whose
// configuration's merge method is SQUASH.
const moqQueueJSON = `{"data":{"repository":{"mergeQueue":{"id":"MQ_kwDOA","configuration":{"mergeMethod":"SQUASH"}}}}}`

const (
	// moqMovedReadyDetail and moqMovedAcceptedDetail are the Details of
	// moqMovedFixture's feat/b and feat/auth when origin/main has a merge
	// queue.
	moqMovedReadyDetail = "the gate passed on bbbbbbbbbbbb, origin's tip of the branch, with base 111111111111; origin/main " +
		"moved to 222222222222 since, and its merge queue tests the PR on the latest origin/main before it merges it, so no " +
		"regate is needed"
	moqMovedAcceptedDetail = "the gate failed (test_failed) on aaaaaaaaaaaa, origin's tip of the branch, with base " +
		"111111111111; origin/main moved to 222222222222 since, and its merge queue tests the PR on the latest origin/main " +
		"before it merges it, so no regate is needed; alice accepted the failure at 2026-10-14T07:10:00Z: the failing test is " +
		"outside the change"
	// moqOrderNote is the last note of every list of the fixtures.
	moqOrderNote = "the order is as of the ls-remote at " + mergeOrderTestListedAt + " and is not a guarantee: origin/main " +
		"can move before a merge"
)

// moqMovedFixture is a ledger whose two branches origin/main moved past:
// feat/b, org-b's pass on moHeadB, and feat/auth, org-a's failure on moHeadA
// that alice accepted (moAccepted), both against origin/main at moMain, with
// origin/main now at moMain2 and each branch's tip on origin the gate's head.
func moqMovedFixture(t *testing.T) *moFixture {
	t.Helper()
	fx := newMOFixture(t)
	failed, accepted := moAccepted(t)
	fx.add(t, moBind("org-a", "auth"), failed, accepted, moBind("org-b", "b"), moPassed("org-b", "feat/b", moHeadB, moMain))
	fx.exec.lsOut = moListing("main", "main", moMain2, "feat/auth", moHeadA, "feat/b", moHeadB)
	return fx
}

// moqParams is fx.params(false) with ReadMergeQueue set to read.
func moqParams(fx *moFixture, read bool) MergeOrderParams {
	p := fx.params(false)
	p.ReadMergeQueue = read
	return p
}

// moqRun is MergeOrder on fx with p, failing the test on an error.
func moqRun(t *testing.T, fx *moFixture, p MergeOrderParams) MergeOrderResult {
	t.Helper()
	res, err := fx.o.MergeOrder(context.Background(), p)
	if err != nil {
		t.Fatalf("MergeOrder: %v", err)
	}
	return res
}

// moqCalls are e's calls of name whose arguments start with prefix: with
// "gh" and "api", the reads of the merge queue; with "git" and "remote",
// the get-urls of origin.
func moqCalls(e *moExec, name, prefix string) []moCall {
	var out []moCall
	for _, c := range e.callsOf(name) {
		if len(c.args) > 0 && c.args[0] == prefix {
			out = append(out, c)
		}
	}
	return out
}

// moqQueueCall is the gh api graphql call that reads the merge queue of main
// in owner/name on host, in /repo.
func moqQueueCall(host, owner, name string) moCall {
	return moCall{dir: "/repo", env: []string{"GH_PROMPT_DISABLED=1"}, name: "gh", args: []string{"api", "--hostname", host,
		"-f", "query=" + mergeQueueQuery, "-f", "owner=" + owner, "-f", "name=" + name, "-f", "branch=main", "graphql"}}
}

// moqJSON is res marshaled as the CLI's --json marshals it, as a string.
func moqJSON(t *testing.T, res MergeOrderResult) string {
	t.Helper()
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestMergeOrder_MergeQueueKeepsMovedBranchesReadyAndChangesTheNote covers
// AC1 and AC5 with a merge queue: with ReadMergeQueue, MergeOrder reads the
// queue once, after the ls-remote and the PR lookup, whose calls are those of
// a run with no read (git remote get-url origin, ssh -G for the ssh origin,
// then one gh api graphql), lists a pass whose base origin/main moved past as
// ready and an accepted failure as accepted, reports the read as present with
// the queue's method (JSON merge_queue), and, for two or more ready or
// accepted branches, replaces the note that merging one makes the others
// rebase with the queue's: add them all first, then wait one at a time, with
// no ralph org regate and no voided acceptance in it. One ready branch gets
// no such note.
func TestMergeOrder_MergeQueueKeepsMovedBranchesReadyAndChangesTheNote(t *testing.T) {
	const (
		readyOrAccepted = "2 branches are ready or accepted, and origin/main has a merge queue, so merging one leaves the others " +
			"ready or accepted: add them all to the queue first (ralph org merge --branch <branch> --wait-checks 0 for each), then " +
			"wait for them one at a time with a budget (ralph org merge --branch <branch> --wait-checks <duration>, such as 60m)"
		twoReady = "2 branches are ready, and origin/main has a merge queue, so merging one leaves the others ready: add them " +
			"all to the queue first (ralph org merge --branch <branch> --wait-checks 0 for each), then wait for them one at a " +
			"time with a budget (ralph org merge --branch <branch> --wait-checks <duration>, such as 60m)"
		atTip = "the gate passed on %s, origin's tip of the branch, with base 111111111111, origin/main's tip"
	)
	for _, tc := range []struct {
		name    string
		fixture func(t *testing.T) *moFixture
		order   []string
		details map[string]string
		notes   []string
	}{
		{"a ready and an accepted branch whose base moved", moqMovedFixture,
			[]string{"1 ready feat/b", "2 accepted feat/auth"},
			map[string]string{"feat/b": moqMovedReadyDetail, "feat/auth": moqMovedAcceptedDetail},
			[]string{readyOrAccepted, moqOrderNote}},
		{"two ready branches at the tip, one org disbanded", func(t *testing.T) *moFixture {
			fx := newMOFixture(t)
			fx.add(t, moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain), moBind("org-b", "b"),
				moPassed("org-b", "feat/b", moHeadB, moMain), moDisbanded("org-b"))
			fx.exec.lsOut = moListing("main", "main", moMain, "feat/a", moHeadA, "feat/b", moHeadB)
			return fx
		}, []string{"1 ready feat/a", "2 ready feat/b"},
			map[string]string{"feat/a": fmt.Sprintf(atTip, "aaaaaaaaaaaa"), "feat/b": fmt.Sprintf(atTip, "bbbbbbbbbbbb")},
			[]string{twoReady, moqOrderNote}},
		{"one ready branch whose base moved", func(t *testing.T) *moFixture {
			fx := newMOFixture(t)
			fx.add(t, moBind("org-b", "b"), moPassed("org-b", "feat/b", moHeadB, moMain))
			fx.exec.lsOut = moListing("main", "main", moMain2, "feat/b", moHeadB)
			return fx
		}, []string{"1 ready feat/b"}, map[string]string{"feat/b": moqMovedReadyDetail}, []string{moqOrderNote}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			off := tc.fixture(t)
			moqRun(t, off, moqParams(off, false))
			fx := tc.fixture(t)
			fx.exec.queueOut = moqQueueJSON
			res := moqRun(t, fx, moqParams(fx, true))
			if got := moOrder(res); !slices.Equal(got, tc.order) {
				t.Fatalf("order = %q, want %q", got, tc.order)
			}
			for branch, want := range tc.details {
				if got := moEntry(t, res, branch).Detail; got != want {
					t.Errorf("%s's Detail = %q\nwant %q", branch, got, want)
				}
			}
			if !slices.Equal(res.Notes, tc.notes) {
				t.Errorf("notes = %q\nwant %q", res.Notes, tc.notes)
			}
			if want := (&MergeOrderQueue{State: MergeOrderQueuePresent, Method: "SQUASH"}); !reflect.DeepEqual(res.MergeQueue, want) {
				t.Errorf("MergeQueue = %+v, want %+v", res.MergeQueue, want)
			}
			if j := moqJSON(t, res); !strings.Contains(j, `"merge_queue":{"state":"present","method":"SQUASH"}`) {
				t.Errorf("JSON %s has no merge_queue present with method SQUASH", j)
			}
			before := off.exec.calls
			if len(fx.exec.calls) < len(before) || !reflect.DeepEqual(fx.exec.calls[:len(before)], before) {
				t.Fatalf("calls = %+v\nwant those of a run with no read first: %+v", fx.exec.calls, before)
			}
			want := []moCall{
				{dir: "/repo", env: []string{"GIT_TERMINAL_PROMPT=0"}, name: "git", args: []string{"remote", "get-url", "origin"}},
				{dir: "/repo", name: "ssh", args: []string{"-G", "github.com"}},
				moqQueueCall("github.com", "o", "r"),
			}
			if got := fx.exec.calls[len(before):]; !reflect.DeepEqual(got, want) {
				t.Errorf("calls after those of a run with no read = %+v\nwant %+v", got, want)
			}
		})
	}
}

// TestMergeOrder_NoMergeQueueKeepsTodaysListAndNotes covers AC1 and AC5 with
// no queue: a read whose mergeQueue is null lists every branch with the
// class, Detail and notes of a run with no read, and reports absent in
// merge_queue with no method or why; it reads once. Two ready branches keep
// the note that merging one makes the others rebase.
func TestMergeOrder_NoMergeQueueKeepsTodaysListAndNotes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fixture func(t *testing.T) *moFixture
	}{
		{"bases moved", moqMovedFixture},
		{"two ready branches", func(t *testing.T) *moFixture {
			fx := newMOFixture(t)
			fx.add(t, moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain), moBind("org-b", "b"),
				moPassed("org-b", "feat/b", moHeadB, moMain))
			fx.exec.lsOut = moListing("main", "main", moMain, "feat/a", moHeadA, "feat/b", moHeadB)
			return fx
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			off := tc.fixture(t)
			want := moqRun(t, off, moqParams(off, false))
			fx := tc.fixture(t)
			res := moqRun(t, fx, moqParams(fx, true))
			if !reflect.DeepEqual(res.Entries, want.Entries) || !slices.Equal(res.Notes, want.Notes) {
				t.Errorf("entries %+v, notes %q\nwant those with no read: %+v, %q", res.Entries, res.Notes, want.Entries, want.Notes)
			}
			if w := (&MergeOrderQueue{State: MergeOrderQueueAbsent}); !reflect.DeepEqual(res.MergeQueue, w) {
				t.Errorf("MergeQueue = %+v, want %+v", res.MergeQueue, w)
			}
			if j := moqJSON(t, res); !strings.Contains(j, `"merge_queue":{"state":"absent"}`) {
				t.Errorf("JSON %s has no merge_queue absent with no method or why", j)
			}
			if n := len(moqCalls(fx.exec, "gh", "api")); n != 1 {
				t.Errorf("%d reads of the merge queue, want 1", n)
			}
		})
	}
	fx := newMOFixture(t)
	fx.add(t, moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain), moBind("org-b", "b"),
		moPassed("org-b", "feat/b", moHeadB, moMain))
	fx.exec.lsOut = moListing("main", "main", moMain, "feat/a", moHeadA, "feat/b", moHeadB)
	if moNoteWith(moqRun(t, fx, moqParams(fx, true)), "2 branches are ready, but merging one moves origin/main") == "" {
		t.Errorf("no note that merging one of two ready branches moves origin/main with no queue")
	}
}

// TestMergeOrder_MergeQueueThatCannotBeReadIsUnknown covers AC1 and AC5 for
// a read that fails: gh failing or timing out after p.GHTimeout, and origin
// that cannot be read (its get-url failing or timing out, or its URL naming no
// repository; then no gh api runs). Each lists the branches whose base
// origin/main moved past as rebase, their Details those of a run with no read
// with a clause that says the queue could not be read and why, adds one note
// before the others that says so and that such a branch is listed as
// rebase, as with no queue, and reports unknown with why in merge_queue.
func TestMergeOrder_MergeQueueThatCannotBeReadIsUnknown(t *testing.T) {
	const subject = "main, the default branch of origin's repository o/r"
	access := ": check that gh can reach github.com and is logged in to it (gh auth status --hostname github.com)"
	blockOn := func(fx *moFixture, name, arg string) MergeOrderExec {
		return func(c context.Context, dir string, env []string, n string, args ...string) (string, error) {
			out, err := fx.exec.run(c, dir, env, n, args...)
			if n == name && args[0] == arg {
				<-c.Done()
				return "", c.Err()
			}
			return out, err
		}
	}
	for _, tc := range []struct {
		name  string
		setup func(fx *moFixture, p *MergeOrderParams)
		why   string
		reads int // gh api calls
	}{
		{"gh fails", func(fx *moFixture, _ *MergeOrderParams) {
			fx.exec.queueErr = errors.New("exit status 1: gh: HTTP 401: Bad credentials")
		}, "gh api --hostname github.com graphql (mergeQueue of main) failed (exit status 1: gh: HTTP 401: Bad credentials), so " +
			"whether " + subject + ", has a merge queue is unknown" + access, 1},
		{"gh times out", func(fx *moFixture, p *MergeOrderParams) {
			p.Exec, p.GHTimeout = blockOn(fx, "gh", "api"), 20*time.Millisecond
		}, "the check timed out after 20ms while gh api --hostname github.com graphql (mergeQueue of main) ran, so the " +
			"protection of origin's repository's default branch is unknown: check the network and gh's access", 1},
		{"origin's get-url fails", func(fx *moFixture, _ *MergeOrderParams) {
			fx.exec.originErr = errors.New("exit status 2: error: No such remote 'origin'")
		}, "git remote get-url origin failed (exit status 2: error: No such remote 'origin'), so origin's repository is unknown", 0},
		{"origin's get-url times out", func(fx *moFixture, p *MergeOrderParams) {
			p.Exec, p.GHTimeout = blockOn(fx, "git", "remote"), 20*time.Millisecond
		}, "git remote get-url origin failed (timed out after 20ms: context deadline exceeded), so origin's repository is unknown", 0},
		{"origin is a local path", func(fx *moFixture, _ *MergeOrderParams) { fx.exec.origin = "/srv/git/r.git" },
			"origin's URL names no host and owner/repo (a local path, file://, or a scheme other than https, http, ssh and git), so " +
				"there is no GitHub repository to read", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			off := moqMovedFixture(t)
			want := moqRun(t, off, moqParams(off, false))
			fx := moqMovedFixture(t)
			fx.exec.queueOut = moqQueueJSON
			p := moqParams(fx, true)
			tc.setup(fx, &p)
			res := moqRun(t, fx, p)
			if got := moOrder(res); !slices.Equal(got, []string{"1 rebase feat/auth", "2 rebase feat/b"}) {
				t.Fatalf("order = %q, want both rebase", got)
			}
			clause := "; the merge queue of origin/main could not be read, and with one the branch would be ready: " + tc.why
			for i, e := range res.Entries {
				if w := want.Entries[i].Detail + clause; e.Detail != w {
					t.Errorf("%s's Detail = %q\nwant %q", e.Branch, e.Detail, w)
				}
			}
			note := "the merge queue of origin/main could not be read, so a branch whose gate base origin/main moved past is " +
				"listed as rebase, as with no queue: " + tc.why
			if w := append([]string{note}, want.Notes...); !slices.Equal(res.Notes, w) {
				t.Errorf("notes = %q\nwant %q", res.Notes, w)
			}
			if w := (&MergeOrderQueue{State: MergeOrderQueueUnknown, Why: tc.why}); !reflect.DeepEqual(res.MergeQueue, w) {
				t.Errorf("MergeQueue = %+v\nwant %+v", res.MergeQueue, w)
			}
			if j, w := moqJSON(t, res), `"merge_queue":{"state":"unknown","why":`; !strings.Contains(j, w) {
				t.Errorf("JSON %s has no %s", j, w)
			}
			if n := len(moqCalls(fx.exec, "gh", "api")); n != tc.reads {
				t.Errorf("%d reads of the merge queue, want %d", n, tc.reads)
			}
		})
	}
}

// TestMergeOrder_MergeQueueReadNamesTheSSHKeyFixForAnUnresolvedHost covers
// self-review finding L5 of plan
// docs/plans/active/2026-11-20-org-merge-queue-order.md: when ssh -G did not
// resolve an ssh origin's host and gh's read of the merge queue on that host
// fails, the why of the unknown read (MergeQueue, the note, and each rebase
// Detail) ends with the ssh key fix, as doctor's and ralph org merge's gh
// failures do; when ssh -G resolved the host, when gh read github.com, and
// for an https origin, which runs no ssh, the why is gh's failure alone.
func TestMergeOrder_MergeQueueReadNamesTheSSHKeyFixForAnUnresolvedHost(t *testing.T) {
	failed := func(host string) string {
		return "gh api --hostname " + host + " graphql (mergeQueue of main) failed (exit status 1: HTTP 404: Not Found), so " +
			"whether main, the default branch of origin's repository o/r, has a merge queue is unknown: check that gh can " +
			"reach " + host + " and is logged in to it (gh auth status --hostname " + host + ")"
	}
	const hint = "; ssh -G github.com.emu failed (exit status 255: no such identity), so gh read the host github.com.emu as " +
		"origin names it: if github.com.emu is an ssh alias (a Host in ~/.ssh/config), point origin at the real host (git " +
		"remote set-url origin <URL>) and choose the ssh key for this repository with git config core.sshCommand 'ssh -i <key> " +
		"-o IdentitiesOnly=yes'"
	sshFails := errors.New("exit status 255: no such identity")
	for _, tc := range []struct {
		name, origin, sshOut string
		sshErr               error
		why                  string
	}{
		{"unresolved alias", "git@github.com.emu:o/r.git", "", sshFails, failed("github.com.emu") + hint},
		{"resolved alias", "git@github.com.emu:o/r.git", "hostname github.com\n", nil, failed("github.com")},
		{"github.com origin whose ssh -G fails", "", "", sshFails, failed("github.com")},
		{"https origin", "https://github.com.emu/o/r.git", "", sshFails, failed("github.com.emu")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := moqMovedFixture(t)
			fx.exec.origin, fx.exec.sshOut, fx.exec.sshErr = tc.origin, tc.sshOut, tc.sshErr
			fx.exec.queueErr = errors.New("exit status 1: HTTP 404: Not Found")
			res := moqRun(t, fx, moqParams(fx, true))
			if w := (&MergeOrderQueue{State: MergeOrderQueueUnknown, Why: tc.why}); !reflect.DeepEqual(res.MergeQueue, w) {
				t.Errorf("MergeQueue = %+v\nwant %+v", res.MergeQueue, w)
			}
			note := "the merge queue of origin/main could not be read, so a branch whose gate base origin/main moved past is " +
				"listed as rebase, as with no queue: " + tc.why
			if len(res.Notes) == 0 || res.Notes[0] != note {
				t.Errorf("notes = %q\nwant first %q", res.Notes, note)
			}
			clause := "; the merge queue of origin/main could not be read, and with one the branch would be ready: " + tc.why
			for _, e := range res.Entries {
				if !strings.HasSuffix(e.Detail, clause) {
					t.Errorf("%s's Detail = %q\nwant it to end with %q", e.Branch, e.Detail, clause)
				}
			}
			if len(res.Entries) != 2 {
				t.Errorf("entries = %q, want feat/auth and feat/b", moOrder(res))
			}
		})
	}
}

// TestMergeOrder_ReadsNoMergeQueueUnlessAskedAndOnlyWithATarget covers AC5's
// conditions: without ReadMergeQueue (ralph org cleanup's MergeOrder) no gh
// api, ssh -G or get-url of origin runs, a queue makes no branch ready, and
// the result and its JSON carry no merge_queue; with it, a ledger with no
// target and an ls-remote that fails read nothing.
func TestMergeOrder_ReadsNoMergeQueueUnlessAskedAndOnlyWithATarget(t *testing.T) {
	queueRan := func(t *testing.T, fx *moFixture) {
		t.Helper()
		if n := len(fx.exec.callsOf("ssh")) + len(moqCalls(fx.exec, "gh", "api")) + len(moqCalls(fx.exec, "git", "remote")); n != 0 {
			t.Errorf("%d calls of ssh -G, gh api or git remote get-url origin, want none", n)
		}
	}
	t.Run("not asked", func(t *testing.T) {
		fx := moqMovedFixture(t)
		fx.exec.queueOut = moqQueueJSON
		res := moqRun(t, fx, moqParams(fx, false))
		if got := moOrder(res); !slices.Equal(got, []string{"1 rebase feat/auth", "2 rebase feat/b"}) {
			t.Errorf("order = %q, want both rebase", got)
		}
		if j := moqJSON(t, res); res.MergeQueue != nil || strings.Contains(j, "merge_queue") {
			t.Errorf("MergeQueue = %+v, JSON %s; want none", res.MergeQueue, j)
		}
		queueRan(t, fx)
	})
	t.Run("no target", func(t *testing.T) {
		fx := newMOFixture(t)
		fx.exec.queueOut = moqQueueJSON
		if res := moqRun(t, fx, moqParams(fx, true)); res.MergeQueue != nil {
			t.Errorf("MergeQueue = %+v, want none with no target", res.MergeQueue)
		}
		queueRan(t, fx)
	})
	t.Run("the ls-remote fails", func(t *testing.T) {
		fx := moqMovedFixture(t)
		fx.exec.lsOut, fx.exec.lsErr = "", errors.New("exit status 128: fatal: could not read from remote repository")
		if res, err := fx.o.MergeOrder(context.Background(), moqParams(fx, true)); err == nil || res.MergeQueue != nil {
			t.Errorf("MergeOrder = %+v, %v; want the ls-remote's error and no read", res, err)
		}
		queueRan(t, fx)
	})
}

// TestMergeOrder_MergeQueueReadsOriginsRepository covers where AC5's read
// looks: origin's repository as `git remote get-url origin` names it, an ssh
// alias read on the host ssh -G names (through p.Exec), an https origin with
// no ssh -G, and p.Origin with no get-url and no ssh -G at all. The get-url
// and the read are each bounded by 15 s with p.GHTimeout zero.
func TestMergeOrder_MergeQueueReadsOriginsRepository(t *testing.T) {
	for _, tc := range []struct {
		name, origin, sshOut string
		pinned               MergeOrigin
		ssh                  []string // the ssh -G call's arguments, nil for none
		getURL               bool
		query                moCall
	}{
		{"the default ssh origin", "", "", MergeOrigin{}, []string{"-G", "github.com"}, true,
			moqQueueCall("github.com", "o", "r")},
		{"an ssh alias", "git@work-gh:o/r.git", "hostname github.com\n", MergeOrigin{}, []string{"-G", "work-gh"}, true,
			moqQueueCall("github.com", "o", "r")},
		{"an https origin", "https://GHE.example.com/acme/app.git", "", MergeOrigin{}, nil, true,
			moqQueueCall("ghe.example.com", "acme", "app")},
		{"p.Origin", "", "", MergeOrigin{apiHost: "ghe.example.com", repo: mergeRepo{owner: "acme", name: "app"}}, nil, false,
			moqQueueCall("ghe.example.com", "acme", "app")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := moqMovedFixture(t)
			fx.exec.origin, fx.exec.sshOut, fx.exec.queueOut = tc.origin, tc.sshOut, moqQueueJSON
			deadlines := map[string]time.Time{}
			p := moqParams(fx, true)
			p.Origin = tc.pinned
			p.Exec = func(c context.Context, dir string, env []string, name string, args ...string) (string, error) {
				if d, ok := c.Deadline(); ok && (name == "gh" || name == "git") {
					deadlines[name+" "+args[0]] = d
				}
				return fx.exec.run(c, dir, env, name, args...)
			}
			start := time.Now()
			res := moqRun(t, fx, p)
			end := time.Now()
			if res.MergeQueue == nil || res.MergeQueue.State != MergeOrderQueuePresent {
				t.Errorf("MergeQueue = %+v, want present", res.MergeQueue)
			}
			if reads := moqCalls(fx.exec, "gh", "api"); len(reads) != 1 || !reflect.DeepEqual(reads[0], tc.query) {
				t.Errorf("reads of the merge queue = %+v\nwant one: %+v", reads, tc.query)
			}
			var ssh, wantSSH [][]string
			for _, c := range fx.exec.callsOf("ssh") {
				ssh = append(ssh, c.args)
			}
			if tc.ssh != nil {
				wantSSH = [][]string{tc.ssh}
			}
			if !reflect.DeepEqual(ssh, wantSSH) {
				t.Errorf("ssh calls = %q, want %q", ssh, wantSSH)
			}
			steps, getURLs := []string{"gh api"}, 0
			if tc.getURL {
				steps, getURLs = append(steps, "git remote"), 1
			}
			if n := len(moqCalls(fx.exec, "git", "remote")); n != getURLs {
				t.Errorf("%d get-urls of origin, want %d", n, getURLs)
			}
			for _, step := range steps {
				if d, ok := deadlines[step]; !ok || d.Before(start.Add(15*time.Second)) || d.After(end.Add(15*time.Second)) {
					t.Errorf("%s's deadline %v is not 15 s after the start (%v..%v)", step, d, start, end)
				}
			}
		})
	}
}

// TestMergeOrder_InterruptedWhileTheMergeQueueIsReadIsAnError covers AC5's
// ctx done: ctx done while origin is read (its get-url or ssh -G) or while gh
// reads the queue is MergeOrder's interrupted error, with nothing classified.
func TestMergeOrder_InterruptedWhileTheMergeQueueIsReadIsAnError(t *testing.T) {
	const originStep = "origin was read (git remote get-url origin, and ssh -G for an ssh origin)"
	for _, tc := range []struct{ name, command, arg, step string }{
		{"the get-url", "git", "remote", originStep},
		{"ssh -G", "ssh", "-G", originStep},
		{"gh api graphql", "gh", "api", "gh read the merge queue of origin/main"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := moqMovedFixture(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			p := moqParams(fx, true)
			p.Exec = func(c context.Context, dir string, env []string, name string, args ...string) (string, error) {
				if name == tc.command && args[0] == tc.arg {
					cancel(errors.New("test stopped"))
					return "", c.Err()
				}
				return fx.exec.run(c, dir, env, name, args...)
			}
			res, err := fx.o.MergeOrder(ctx, p)
			want := "org: merge-order: interrupted (test stopped) while " + tc.step + ": nothing was classified"
			if err == nil || err.Error() != want || res.MergeQueue != nil || len(res.Entries) != 0 {
				t.Fatalf("MergeOrder = %+v, %v\nwant %q", res, err, want)
			}
		})
	}
}
