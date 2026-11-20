package org

import (
	"context"
	"errors"
	"fmt"
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
