package org

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"
)

// The queue path of `ralph org merge` once the checks passed, up to the
// enqueue (merge_queue_run.go; plan
// docs/plans/active/2026-11-20-org-merge-queue-verb.md, S3a: AC4 for a budget
// of 0, AC5, AC10, AC13 and AC16; spec docs/specs/2026-11-19-org-merge-queue.md,
// FR-4, FR-5, FR-5b, FR-7 and FR-10) with mqrExec, mrExec of
// merge_fake_test.go with the two GraphQL calls of the queue path answered
// apart from the protection's: the read of the PR's merge queue state and
// enqueuePullRequest. The wait for the queue that a budget above 0 starts is
// in merge_queue_wait_test.go. No test runs gh or reaches a network.

// The keys mqrExec records the queue path's GraphQL calls under.
const (
	mqrKeyRead    = "gh api graphql (merge queue state)"
	mqrKeyEnqueue = "gh api graphql (enqueuePullRequest)"
)

const (
	// mqrGateTS is the ts moFixture.add gives the gate record of
	// mxReadyLedger, its second line: 118 minutes before mergeOrderTestClock.
	mqrGateTS = "2026-10-14T07:02:00Z"
	// mqrEnqueuedAt is when the fixture's enqueuePullRequest answer says the
	// PR was enqueued.
	mqrEnqueuedAt = "2026-10-14T09:00:30Z"
	// mqrNext is the next step of an outcome that leaves PR #7 in the merge
	// queue, or may (mergeRun.queueNext).
	mqrNext = "run ralph org merge --branch feat/a again, which reads the merge queue first and adds the PR only when it is not " +
		"there, and once GitHub merged the PR, run ralph org cleanup --branch feat/a to clean up the branch's feature worktree"
	// mqrRequeueNext is the next step once GitHub removed PR #7 from the merge
	// queue after the gate (mergeRun.requeueNext).
	mqrRequeueNext = "push a fix to the branch from its feature worktree and run ralph org regate --branch feat/a to gate the " +
		"new head, or, after a flaky check, run ralph org merge --branch feat/a --requeue to add it again with the same gate on " +
		"purpose"
	// mqrReadCmd and mqrEnqueueCmd name the read and enqueuePullRequest of
	// PR #7 (node ID mqID) in the texts.
	mqrReadCmd    = "gh api --hostname github.com graphql (the merge queue state of o/r#7)"
	mqrEnqueueCmd = "gh api --hostname github.com graphql (enqueuePullRequest of " + mqID + " at aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa)"
)

// mqrReadOnly are the commands of a queue path that reads the PR's merge
// queue state and sends no enqueuePullRequest, and mqrEnqueued those of one
// that sends it.
var (
	mqrReadOnly = slices.Concat(mxQueuePath, []string{mqrKeyRead})
	mqrEnqueued = slices.Concat(mqrReadOnly, []string{mqrKeyEnqueue})
)

// mqrReadArgv and mqrEnqueueArgv are the argv of the read and of
// enqueuePullRequest for PR #7 of o/r on github.com at moHeadA.
var (
	mqrReadArgv = []string{"gh", "api", "--hostname", "github.com", "-f", "query=" + mergeQueuePRQuery, "-f", "owner=o", "-f",
		"name=r", "-F", "number=7", "graphql"}
	mqrEnqueueArgv = []string{"gh", "api", "--hostname", "github.com", "-f", "query=" + mergeQueueEnqueueMutation, "-f",
		"pullRequestId=" + mqID, "-f", "expectedHeadOid=" + moHeadA, "graphql"}
)

// mqrExec is mrExec with the queue path's GraphQL calls, told from the
// protection's GraphQL read (mxQueue) by their query, answered by replies of
// their own: read, the read of the PR's merge queue state
// (mergeQueuePRQuery), and enqueue, enqueuePullRequest
// (mergeQueueEnqueueMutation). They are recorded with mxExec's calls, under
// mqrKeyRead and mqrKeyEnqueue. In each, inRead or inEnqueue runs with its
// ctx first, when set; a call whose ctx is done then fails with ctx's error,
// as a gh that is killed does. answered, when set, runs with the call's key
// once the call has its answer, as it returns.
type mqrExec struct {
	*mrExec
	read, enqueue     mgReply
	inRead, inEnqueue func(ctx context.Context)
	answered          func(key string)
}

func (e *mqrExec) run(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	var key string
	var r mgReply
	var in func(context.Context)
	switch {
	case name == "gh" && slices.Contains(args, "query="+mergeQueuePRQuery):
		key, r, in = mqrKeyRead, e.read, e.inRead
	case name == "gh" && slices.Contains(args, "query="+mergeQueueEnqueueMutation):
		key, r, in = mqrKeyEnqueue, e.enqueue, e.inEnqueue
	default:
		return e.mrExec.run(ctx, dir, env, name, args...)
	}
	if e.answered != nil {
		defer e.answered(key)
	}
	deadline, _ := ctx.Deadline()
	e.mu.Lock()
	e.calls = append(e.calls, mxCall{key: key, dir: dir, env: slices.Clone(env), name: name, args: slices.Clone(args), deadline: deadline})
	e.mu.Unlock()
	if in != nil {
		in(ctx)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return r.out, r.err
}

// mqrFixture is mrFixture with mqrExec in front of its mrExec, on a default
// branch with a merge queue whose configuration's method is SQUASH and a
// check the rules require, and no answer for gh repo view, which step 12
// does not run on the queue path. The read answers PR #7 open at moHeadA
// into main, not in the queue, with no queue events (mqrPR), and
// enqueuePullRequest an entry at position 1, enqueued at mqrEnqueuedAt.
type mqrFixture struct {
	*mrFixture
	q *mqrExec
}

// newMQRFixture is the queue fixture on the ledger lines (mxReadyLedger when
// nil).
func newMQRFixture(t *testing.T, lines []any) *mqrFixture {
	t.Helper()
	fx := newMRFixture(t, lines)
	fx.exec.replies[mxQueue] = mgReply{out: mxQueueJSON(`{"mergeMethod":"SQUASH"}`)}
	delete(fx.exec.replies, mxRepoView)
	return &mqrFixture{mrFixture: fx, q: &mqrExec{mrExec: fx.ex, read: mqrPR(nil),
		enqueue: mgReply{out: mqEnqueueAnswer(mqEntry("QUEUED", 1, mqrEnqueuedAt))}}}
}

// queueParams is the merge of feat/a through mqrExec with --wait-checks 0,
// with a wait whose sleep fails the test: with no budget the queue path does
// not wait (D5).
func (fx *mqrFixture) queueParams(t *testing.T) MergeParams {
	p := fx.params()
	p.Exec = fx.q.run
	p.waitTest = mergeWaitTest{sleep: func(context.Context, time.Duration) error {
		t.Errorf("Merge slept on the queue path")
		return errors.New("mqrFixture: no sleep")
	}}
	return p
}

// assertRan fails the test unless the commands are keys, nothing was
// appended to the manifest, and regate.lock is free.
func (fx *mqrFixture) assertRan(t *testing.T, keys []string) {
	t.Helper()
	if got := fx.exec.keys(); !slices.Equal(got, keys) {
		t.Errorf("commands = %q\nwant %q", got, keys)
	}
	if got := fx.appended(t); got != "" {
		t.Errorf("appended %q, want nothing", got)
	}
	mrAssertLockFree(t, fx.mo.stateDir)
}

// mqrPR is the read's answer for PR #7, open at moHeadA into main, not in
// the merge queue, with no queue events, with the fields of set put over it.
func mqrPR(set map[string]any) mgReply {
	pr := mqPRFields()
	pr["headRefOid"] = moHeadA
	maps.Copy(pr, set)
	return mgReply{out: mqPRAnswer(pr)}
}

// mqrResult is the MergeResult fields the checks of the queue fixture set:
// the queue's method and Detail.
func mqrResult() MergeResult {
	res := mrResult("squash")
	res.Protection = mxQueueProtection("GraphQL mergeQueue is set", "rulesets require 1 required status check")
	return res
}

// mqrQueued is the Queued of PR #7 that how says is in the merge queue of
// main, at where.
func mqrQueued(how, where string) string {
	return "org: merge: PR #7 (https://github.com/o/r/pull/7) " + how + " at the head aaaaaaaaaaaa, the head the gate passed, " +
		"and " + where + "; it is not merged yet, so nothing was recorded: " + mqrNext
}

// mqrAdded is mqrResult with the Queued of an enqueuePullRequest that gave
// the fixture's entry.
func mqrAdded() MergeResult {
	res := mqrResult()
	res.Queued = mqrQueued("was added to the merge queue of main", "it is at position 1, enqueued at "+mqrEnqueuedAt)
	res.QueuePosition, res.EnqueuedAt = 1, mqrEnqueuedAt
	return res
}

// TestMerge_QueuePathEnqueuesThePRAtTheGatesHead covers FR-4 and AC4 for
// --wait-checks 0: once the checks pass on the queue path, Merge reads the
// PR's merge queue state and sends enqueuePullRequest once, with the PR's
// node ID from the read and the gate's head, each with its argv and a
// deadline of the gh timeout, and ends Queued with the answer's entry, not
// read again (D5): no gh pr merge, no read-back, no gh repo view, no wait,
// nothing appended, and regate.lock released.
func TestMerge_QueuePathEnqueuesThePRAtTheGatesHead(t *testing.T) {
	fx := newMQRFixture(t, nil)
	start := time.Now()
	res, err := fx.merge(context.Background(), t, fx.queueParams(t))
	end := time.Now()
	if want := mqrAdded(); err != nil || res != want {
		t.Fatalf("Merge = %+v, %v\nwant %+v", res, err, want)
	}
	fx.assertRan(t, mqrEnqueued)
	for key, argv := range map[string][]string{mqrKeyRead: mqrReadArgv, mqrKeyEnqueue: mqrEnqueueArgv} {
		c := fx.call(t, key)
		if got := append([]string{c.name}, c.args...); !slices.Equal(got, argv) {
			t.Errorf("%s = %q\nwant %q", key, got, argv)
		}
		if c.deadline.Before(start.Add(defaultMergeOrderGHTimeout)) || c.deadline.After(end.Add(defaultMergeOrderGHTimeout)) {
			t.Errorf("%s has the deadline %v; want the gh timeout (%s) from the call", key, c.deadline, defaultMergeOrderGHTimeout)
		}
	}
	mrNoRawControl(t, res.Queued)
}

// TestMerge_QueuePathPinsTheGraphQLCallsToTheResolvedHost covers AC13 for
// the read and the enqueue: with origin an ssh alias whose ssh -G names
// github.com, both go to gh api with --hostname github.com, not the alias,
// in /repo with GH_PROMPT_DISABLED=1 (mrFixture.merge, mxAssertPinned), with
// the PR's number as -F, which the query's $number: Int! needs, and every
// other variable as -f.
func TestMerge_QueuePathPinsTheGraphQLCallsToTheResolvedHost(t *testing.T) {
	fx := newMQRFixture(t, nil)
	fx.exec.origin = "git@github.com.emu:o/r.git\n"
	fx.exec.ssh = &mgReply{out: "host github.com.emu\nuser git\nhostname github.com\nport 22\n"}
	if res, err := fx.merge(context.Background(), t, fx.queueParams(t)); err != nil || res != mqrAdded() {
		t.Fatalf("Merge = %+v, %v; want the PR added to the queue", res, err)
	}
	if !strings.Contains(mergeQueuePRQuery, "$number: Int!") {
		t.Fatalf("the read's query does not declare $number: Int!: %s", mergeQueuePRQuery)
	}
	for key, want := range map[string]map[string]string{
		mqrKeyRead:    {"owner": "-f", "name": "-f", "number": "-F"},
		mqrKeyEnqueue: {"pullRequestId": "-f", "expectedHeadOid": "-f"},
	} {
		c := fx.call(t, key)
		if !slices.Equal(c.args[:3], []string{"api", "--hostname", "github.com"}) || c.dir != "/repo" ||
			!slices.Equal(c.env, []string{"GH_PROMPT_DISABLED=1"}) {
			t.Errorf("%s = %q in %q with %q; want gh api --hostname github.com in /repo with GH_PROMPT_DISABLED=1", key, c.args,
				c.dir, c.env)
		}
		if got := mqFieldFlags(t, c.args); !maps.Equal(got, want) {
			t.Errorf("%s variables = %v, want %v", key, got, want)
		}
	}
}

// TestMerge_QueuePathDoesNotEnqueueAPRInTheQueueAlready covers FR-5 and AC5
// with --wait-checks 0: a PR the read finds in the merge queue at the gate's
// head gets no enqueuePullRequest; Merge ends Queued with the entry the read
// gave, in UTC, or with no position when GitHub gave none, and a QueueNote,
// also with a removal after the gate in its timeline, which counts only for a
// PR out of the queue. With a budget, the wait for the queue goes on from
// there (merge_queue_wait_test.go).
func TestMerge_QueuePathDoesNotEnqueueAPRInTheQueueAlready(t *testing.T) {
	const how = "is in the merge queue of main already"
	const note = "PR #7 was in the merge queue already at the head the gate passed, so ralph org merge did not add it again"
	for _, tc := range []struct {
		name     string
		set      map[string]any
		where    string
		position int
		at       string
	}{
		{"with its entry", map[string]any{"isInMergeQueue": true, "mergeQueueEntry": mqEntry("AWAITING_CHECKS", 2,
			"2026-10-14T08:50:00Z")}, "it is at position 2, enqueued at 2026-10-14T08:50:00Z", 2, "2026-10-14T08:50:00Z"},
		{"an entry with an offset", map[string]any{"isInMergeQueue": true, "mergeQueueEntry": mqEntry("QUEUED", 3,
			"2026-10-14T17:50:00+09:00")}, "it is at position 3, enqueued at 2026-10-14T08:50:00Z", 3, "2026-10-14T08:50:00Z"},
		{"no entry", map[string]any{"isInMergeQueue": true}, "GitHub gave no entry of it, so its position is unknown", 0, ""},
		{"a removal after the gate", map[string]any{"isInMergeQueue": true, "mergeQueueEntry": mqEntry("QUEUED", 1,
			"2026-10-14T08:50:00Z"), "timelineItems": mqNodes(mqRemoved("2026-10-14T08:10:00Z", "failed_checks", "alice"))},
			"it is at position 1, enqueued at 2026-10-14T08:50:00Z", 1, "2026-10-14T08:50:00Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMQRFixture(t, nil)
			fx.q.read = mqrPR(tc.set)
			res, err := fx.merge(context.Background(), t, fx.queueParams(t))
			want := mqrResult()
			want.Queued, want.QueuePosition, want.EnqueuedAt, want.QueueNote = mqrQueued(how, tc.where), tc.position, tc.at, note
			if err != nil || res != want {
				t.Fatalf("Merge = %+v, %v\nwant %+v", res, err, want)
			}
			fx.assertRan(t, mqrReadOnly)
		})
	}
}

// TestMerge_QueuePathRefusesAReadThatDisagreesWithTheChecks covers the
// agreement with step 9: a read that finds the PR not OPEN, at another head
// (in the merge queue or not), or merging into another base is refused with
// what it found, escaped, and the next step, no enqueuePullRequest sent and
// every other field empty.
func TestMerge_QueuePathRefusesAReadThatDisagreesWithTheChecks(t *testing.T) {
	const lead = "org: merge: the read of PR #7's merge queue state after the checks found "
	const notAdded = "so nothing was merged and it was not added to the merge queue: "
	const base = notAdded + "change its base (gh pr edit 7 --base main -R github.com/o/r), and once its CI passed, " + mxAgain
	for _, tc := range []struct {
		name string
		set  map[string]any
		want string
	}{
		{"MERGED", map[string]any{"state": "MERGED", "mergeCommit": map[string]any{"oid": mqMerge}}, lead + "it MERGED, not OPEN, " +
			notAdded + "run ralph org merge-order to see the branch's state"},
		{"CLOSED", map[string]any{"state": "CLOSED"}, lead + "it CLOSED, not OPEN, " + notAdded + "run ralph org merge-order to see " +
			"the branch's state"},
		{"another head", map[string]any{"headRefOid": moHeadB}, lead + "its head at bbbbbbbbbbbb, not aaaaaaaaaaaa, the head the " +
			"gate passed, " + notAdded + "run ralph org merge-order again and follow the branch's Detail"},
		{"in the queue at another head", map[string]any{"headRefOid": moHeadB, "isInMergeQueue": true,
			"mergeQueueEntry": mqEntry("QUEUED", 1, "2026-10-14T08:50:00Z")}, lead + "it in the merge queue at the head " +
			"bbbbbbbbbbbb, not aaaaaaaaaaaa, the head the gate passed, so nothing was merged and it was not added to the merge " +
			"queue again: run ralph org merge-order again and follow the branch's Detail"},
		{"another base", map[string]any{"baseRefName": "dev"}, lead + "it merging into dev, not origin's default branch main, " + base},
		{"a base with a control character", map[string]any{"baseRefName": "dev\x1b[31m"}, lead + `it merging into dev\x1b[31m, ` +
			"not origin's default branch main, " + base},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMQRFixture(t, nil)
			fx.q.read = mqrPR(tc.set)
			res, err := fx.merge(context.Background(), t, fx.queueParams(t))
			if err != nil || res != (MergeResult{Refused: tc.want}) {
				t.Fatalf("Merge = %+v, %v\nwant refused %q", res, err, tc.want)
			}
			mrNoRawControl(t, res.Refused)
			fx.assertRan(t, mqrReadOnly)
		})
	}
}

// TestMerge_QueuePathRefusesAReadThatFails: a read whose gh fails, is not
// found, or times out is refused as a gh call of the checks is (ghFailed),
// and an answer that cannot be read with the command and the next step,
// whether the PR is in the queue being unknown; no enqueuePullRequest is
// sent.
func TestMerge_QueuePathRefusesAReadThatFails(t *testing.T) {
	const unknown = ", so whether PR #7 is in the merge queue is unknown and nothing was merged: " + mxAccess
	for _, tc := range []struct {
		name    string
		read    mgReply
		timeout time.Duration
		want    string
	}{
		{"gh fails", mgReply{err: errors.New("exit status 1: HTTP 502")}, 0, "org: merge: " + mqrReadCmd +
			" failed (exit status 1: HTTP 502)" + unknown},
		{"gh is not found", mxNotFound, 0, mxGHMissing},
		{"gh times out", mgReply{}, 50 * time.Millisecond, "org: merge: " + mqrReadCmd + " failed (timed out after 50ms: " +
			"context deadline exceeded)" + unknown},
		{"no data", mgReply{out: `{"data":null}`}, 0, "org: merge: " + mqrReadCmd + " returned no data" + unknown},
		{"errors", mgReply{out: `{"errors":[{"message":"x"}]}`}, 0, "org: merge: " + mqrReadCmd +
			` answered with errors ([{"message":"x"}])` + unknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMQRFixture(t, nil)
			fx.q.read = tc.read
			p := fx.queueParams(t)
			if tc.timeout > 0 {
				p.GHTimeout = tc.timeout
				fx.q.inRead = func(ctx context.Context) { <-ctx.Done() }
			}
			res, err := fx.merge(context.Background(), t, p)
			if err != nil || res != (MergeResult{Refused: tc.want}) {
				t.Fatalf("Merge = %+v, %v\nwant refused %q", res, err, tc.want)
			}
			fx.assertRan(t, mqrReadOnly)
		})
	}
}

// TestMergeQueueRun_ReadRefusesValuesThatFailTheirCheck: a read whose values
// fail their check (errMergeQueueInput), which the checks keep from
// happening, is refused with no gh run.
func TestMergeQueueRun_ReadRefusesValuesThatFailTheirCheck(t *testing.T) {
	e := &mxExec{}
	run := mergeRun{m: newMergeCheck(MergeParams{Branch: "feat/a", Dir: "/repo", Exec: e.run}), r: mergeReady{branch: "feat/a",
		apiHost: "github.com", repo: mergeRepo{owner: "o", name: "r"}, prNumber: 0}}
	_, refused, err := run.readQueue(context.Background())
	const want = "org: merge: the merge queue state of PR #0 was not read (the PR number 0 is not 1 to 2147483647: not passed " +
		"to gh), so nothing was merged: merge the PR through the queue yourself"
	if err != nil || refused != want {
		t.Fatalf("readQueue = %q, %v\nwant %q", refused, err, want)
	}
	if got := e.keys(); len(got) != 0 {
		t.Errorf("commands = %q, want none", got)
	}
}

// TestMerge_QueuePathRefusesARemovalAfterTheGateWithoutRequeue covers FR-5b,
// AC10 and AC16: a PR out of the queue that GitHub removed from it after the
// branch's latest gate record, at it, or less than mergeQueueClockSkew (5m)
// before it, is refused without --requeue with the latest such removal,
// escaped, and no enqueuePullRequest; with --requeue it is added again with a
// QueueNote. A removal 5m or more before the gate does not count, so the PR
// is added with no note, --requeue or not.
func TestMerge_QueuePathRefusesARemovalAfterTheGateWithoutRequeue(t *testing.T) {
	const gate = "the branch's latest gate record (at " + mqrGateTS + "; a removal less than 5m0s before it counts as after it)"
	for _, tc := range []struct {
		name    string
		nodes   []any
		removal string // the removal's text when it counts, "" when none counts
	}{
		{"after the gate", []any{mqRemoved("2026-10-14T07:30:00Z", "failed_checks", "alice")},
			"removed from the merge queue at 2026-10-14T07:30:00Z by alice (reason: failed_checks)"},
		{"at the gate's ts", []any{mqRemoved(mqrGateTS, "manual", "bob")},
			"removed from the merge queue at " + mqrGateTS + " by bob (reason: manual)"},
		{"4m59s before the gate, inside the skew", []any{mqRemoved("2026-10-14T06:57:01Z", nil, nil)},
			"removed from the merge queue at 2026-10-14T06:57:01Z by an actor GitHub does not name (reason: none given)"},
		{"5m before the gate, the edge of the skew", []any{mqRemoved("2026-10-14T06:57:00Z", "manual", "bob")}, ""},
		{"10m before the gate", []any{mqRemoved("2026-10-14T06:52:00Z", "failed_checks", "alice")}, ""},
		{"an offset after the gate", []any{mqRemoved("2026-10-14T16:10:00+09:00", "failed_checks", "alice")},
			"removed from the merge queue at 2026-10-14T07:10:00Z by alice (reason: failed_checks)"},
		{"the latest of two counts", []any{mqRemoved("2026-10-14T06:00:00Z", "manual", "bob"),
			mqRemoved("2026-10-14T07:40:00Z", "failed_checks", "alice"), mqRemoved("2026-10-14T07:35:00Z", "manual", "carol")},
			"removed from the merge queue at 2026-10-14T07:40:00Z by alice (reason: failed_checks)"},
		{"a reason with a control character", []any{mqRemoved("2026-10-14T07:30:00Z", "failed\x1b[31m\nchecks", "alice")},
			`removed from the merge queue at 2026-10-14T07:30:00Z by alice (reason: failed\x1b[31m\nchecks)`},
	} {
		for _, requeue := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s requeue=%t", tc.name, requeue), func(t *testing.T) {
				fx := newMQRFixture(t, nil)
				fx.q.read = mqrPR(map[string]any{"timelineItems": mqNodes(tc.nodes...)})
				p := fx.queueParams(t)
				p.Requeue = requeue
				res, err := fx.merge(context.Background(), t, p)
				want, keys := mqrAdded(), mqrEnqueued
				switch {
				case tc.removal != "" && !requeue:
					want, keys = MergeResult{Refused: "org: merge: PR #7 was " + tc.removal + ", after " + gate + ", so ralph org " +
						"merge did not add it to the merge queue again and nothing was merged: " + mqrRequeueNext}, mqrReadOnly
				case tc.removal != "":
					want.QueueNote = "PR #7 was " + tc.removal + ", after " + gate + ", and --requeue added it to the merge queue again"
				}
				if err != nil || res != want {
					t.Fatalf("Merge = %+v, %v\nwant %+v", res, err, want)
				}
				mrNoRawControl(t, res.Refused+res.QueueNote)
				fx.assertRan(t, keys)
			})
		}
	}
}

// TestMerge_QueuePathComparesRemovalsWithTheAcceptanceWhenLater covers FR-5b
// for a branch merge-order lists as accepted: a removal is compared with the
// acceptance (07:10), which is later than the gate failure (07:00), so one
// at 07:04, which the failure's time would count, is not counted, and one at
// 07:06 is, and the refusal names the gate record and its acceptance.
func TestMerge_QueuePathComparesRemovalsWithTheAcceptanceWhenLater(t *testing.T) {
	lines, accepted := mxAcceptedLedger(t)
	for _, tc := range []struct {
		name    string
		removed string
		refused bool
	}{
		{"before the acceptance's skew", "2026-10-14T07:04:00Z", false},
		{"inside the acceptance's skew", "2026-10-14T07:06:00Z", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMQRFixture(t, lines)
			fx.q.read = mqrPR(map[string]any{"timelineItems": mqNodes(mqRemoved(tc.removed, "failed_checks", "alice"))})
			res, err := fx.merge(context.Background(), t, fx.queueParams(t))
			want, keys := mqrAdded(), mqrEnqueued
			want.Accepted = &accepted
			if tc.refused {
				want, keys = MergeResult{Refused: "org: merge: PR #7 was removed from the merge queue at " + tc.removed + " by alice " +
					"(reason: failed_checks), after the branch's latest gate record and its acceptance (the later at " +
					"2026-10-14T07:10:00Z; a removal less than 5m0s before it counts as after it), so ralph org merge did not add it " +
					"to the merge queue again and nothing was merged: " + mqrRequeueNext}, mqrReadOnly
			}
			if err != nil || !tc.refused && (res.Accepted == nil || *res.Accepted != accepted) {
				t.Fatalf("Merge = %+v, %v; want Accepted %+v", res, err, accepted)
			}
			res.Accepted, want.Accepted = nil, nil
			if res != want {
				t.Fatalf("Merge = %+v\nwant %+v", res, want)
			}
			fx.assertRan(t, keys)
		})
	}
}

// TestMergeGateTime_TakesTheLaterOfTheGateAndItsAcceptance covers D4: the
// gate's time is its record's ts, or the acceptance's when that is later,
// and a ts that is not RFC 3339, the gate's or the acceptance's, gives the
// zero time, which counts every removal.
func TestMergeGateTime_TakesTheLaterOfTheGateAndItsAcceptance(t *testing.T) {
	at := func(ts string) *GateAcceptance { return &GateAcceptance{TS: ts} }
	for _, tc := range []struct {
		name     string
		gate     string
		accepted *GateAcceptance
		want     string // "" for the zero time
	}{
		{"a gate record only", "2026-10-14T07:00:00Z", nil, "2026-10-14T07:00:00Z"},
		{"a gate record with an offset", "2026-10-14T16:00:00+09:00", nil, "2026-10-14T07:00:00Z"},
		{"an acceptance later", "2026-10-14T07:00:00Z", at("2026-10-14T07:10:00Z"), "2026-10-14T07:10:00Z"},
		{"an acceptance earlier", "2026-10-14T07:00:00Z", at("2026-10-14T06:50:00Z"), "2026-10-14T07:00:00Z"},
		{"a gate ts that is empty", "", nil, ""},
		{"a gate ts that is not RFC 3339", "2026-10-14 07:00:00", at("2026-10-14T07:10:00Z"), ""},
		{"an acceptance ts that is not RFC 3339", "2026-10-14T07:00:00Z", at("yesterday"), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeGateTime(tc.gate, tc.accepted)
			switch {
			case tc.want == "" && !got.IsZero():
				t.Errorf("mergeGateTime = %v, want the zero time", got)
			case tc.want != "" && !got.Equal(mqTime(t, tc.want)):
				t.Errorf("mergeGateTime = %v, want %s", got, tc.want)
			}
		})
	}
	if _, ok := mergeQueueRemovalAfterGate([]mergeQueueRemoval{{At: mqTime(t, "2000-01-01T00:00:00Z")}}, time.Time{}); !ok {
		t.Errorf("a zero gate time does not count a removal; want every removal counted")
	}
}

// TestMerge_QueuePathEnqueueWhoseOutcomeIsUnknownIsFailed covers D5: an
// enqueuePullRequest that fails, times out, or answers with no entry of the
// PR, with errors, or with no data is Failed, the PR maybe in the queue, with
// the command and the next step; nothing is recorded, and nothing is read
// again.
func TestMerge_QueuePathEnqueueWhoseOutcomeIsUnknownIsFailed(t *testing.T) {
	const unknown = ", so whether PR #7 was added to the merge queue is unknown, and nothing was recorded: " + mqrNext
	for _, tc := range []struct {
		name    string
		enqueue mgReply
		timeout time.Duration
		want    string
	}{
		{"gh fails", mgReply{err: errors.New("exit status 1: HTTP 502")}, 0, "org: merge: " + mqrEnqueueCmd +
			" failed (exit status 1: HTTP 502)" + unknown},
		{"gh times out", mgReply{}, 50 * time.Millisecond, "org: merge: " + mqrEnqueueCmd + " timed out after 50ms" + unknown},
		{"no entry", mgReply{out: mqEnqueueAnswer(nil)}, 0, "org: merge: " + mqrEnqueueCmd + " returned no " +
			"enqueuePullRequest.mergeQueueEntry" + unknown},
		{"no data", mgReply{out: `{"data":null}`}, 0, "org: merge: " + mqrEnqueueCmd + " returned no " +
			"enqueuePullRequest.mergeQueueEntry" + unknown},
		{"errors", mgReply{out: `{"errors":[{"message":"x"}]}`}, 0, "org: merge: " + mqrEnqueueCmd +
			` answered with errors ([{"message":"x"}])` + unknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMQRFixture(t, nil)
			fx.q.enqueue = tc.enqueue
			p := fx.queueParams(t)
			if tc.timeout > 0 {
				p.GHTimeout = tc.timeout
				fx.q.inEnqueue = func(ctx context.Context) { <-ctx.Done() }
			}
			res, err := fx.merge(context.Background(), t, p)
			want := mqrResult()
			want.Failed = tc.want
			if err != nil || res != want {
				t.Fatalf("Merge = %+v, %v\nwant %+v", res, err, want)
			}
			fx.assertRan(t, mqrEnqueued)
		})
	}
}

// TestMerge_QueuePathRefusesAnEnqueueThatSendsNothing: a node ID from the
// read that fails its check is refused with no enqueuePullRequest run, and a
// gh that is not found for it, which sends nothing, is refused as gh not on
// PATH is.
func TestMerge_QueuePathRefusesAnEnqueueThatSendsNothing(t *testing.T) {
	t.Run("a node ID that fails its check", func(t *testing.T) {
		fx := newMQRFixture(t, nil)
		fx.q.read = mqrPR(map[string]any{"id": "PR id\x1b"})
		res, err := fx.merge(context.Background(), t, fx.queueParams(t))
		const want = `org: merge: PR #7 was not added to the merge queue (the PR node ID "PR id\x1b" is not 1 to 128 ` +
			"characters of A-Z a-z 0-9 _ - =: not passed to gh), so nothing was merged: merge the PR through the queue yourself"
		if err != nil || res != (MergeResult{Refused: want}) {
			t.Fatalf("Merge = %+v, %v\nwant refused %q", res, err, want)
		}
		mrNoRawControl(t, res.Refused)
		fx.assertRan(t, mqrReadOnly)
	})
	t.Run("gh is not found", func(t *testing.T) {
		fx := newMQRFixture(t, nil)
		fx.q.enqueue = mxNotFound
		res, err := fx.merge(context.Background(), t, fx.queueParams(t))
		if err != nil || res != (MergeResult{Refused: mxGHMissing}) {
			t.Fatalf("Merge = %+v, %v\nwant refused %q", res, err, mxGHMissing)
		}
		fx.assertRan(t, mqrEnqueued)
	})
}

// TestMerge_QueuePathInterrupted covers the interruptions of the queue path
// (D6): before enqueuePullRequest was sent, an error that says nothing was
// merged and the PR was not added to the queue, with an empty result; once it
// was sent, an error that says the PR may be in the merge queue, with the
// next step a queued outcome gives, and the result as far as it got, or, once
// its answer came, the Queued outcome itself. Nothing is recorded, and
// regate.lock is released.
func TestMerge_QueuePathInterrupted(t *testing.T) {
	const before = ": nothing was merged, and PR #7 was not added to the merge queue"
	const after = "org: merge: interrupted (test stopped) once enqueuePullRequest had been sent, and then: "
	for _, tc := range []struct {
		name  string
		setup func(fx *mqrFixture, stop func())
		res   MergeResult
		err   string
		keys  []string
	}{
		{"after the checks", func(_ *mqrFixture, stop func()) {
			mergeTestHooks.afterChecks = stop
		}, MergeResult{}, "org: merge: interrupted (test stopped) after the checks passed, before the PR's merge queue state was " +
			"read" + before, mxQueuePath},
		{"during the read", func(fx *mqrFixture, stop func()) {
			fx.q.inRead = func(context.Context) { stop() }
		}, MergeResult{}, "org: merge: interrupted (test stopped) while the PR's merge queue state was read" + before, mqrReadOnly},
		{"once the read answered", func(fx *mqrFixture, stop func()) {
			fx.q.answered = func(key string) {
				if key == mqrKeyRead {
					stop()
				}
			}
		}, MergeResult{}, "org: merge: interrupted (test stopped) while the PR's merge queue state was read" + before, mqrReadOnly},
		{"during enqueuePullRequest", func(fx *mqrFixture, stop func()) {
			fx.q.inEnqueue = func(context.Context) { stop() }
		}, mqrResult(), after + "PR #7 may be in the merge queue, and nothing was recorded: " + mqrNext, mqrEnqueued},
		{"once enqueuePullRequest answered", func(fx *mqrFixture, stop func()) {
			fx.q.answered = func(key string) {
				if key == mqrKeyEnqueue {
					stop()
				}
			}
		}, mqrAdded(), after + strings.TrimPrefix(mqrAdded().Queued, "org: merge: "), mqrEnqueued},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMQRFixture(t, nil)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			t.Cleanup(func() { mergeTestHooks.afterChecks = nil })
			tc.setup(fx, func() { cancel(errors.New("test stopped")) })
			res, err := fx.merge(ctx, t, fx.queueParams(t))
			if err == nil || err.Error() != tc.err || res != tc.res {
				t.Fatalf("Merge = %+v, %v\nwant %+v, %q", res, err, tc.res, tc.err)
			}
			fx.assertRan(t, tc.keys)
		})
	}
}
