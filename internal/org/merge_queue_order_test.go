package org

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// The merge queue in the checks of `ralph org merge` (merge_queue_order.go,
// and steps 7, 11 and 13 of checkMerge in merge.go; plan
// docs/plans/active/2026-11-20-org-merge-queue-order.md, S3: AC2, AC3, AC4
// and AC7, D2 and D3; spec docs/specs/2026-11-19-org-merge-queue.md, FR-12
// and AC11): one branch with mxExec and the queue fixtures of
// merge_queue_path_test.go, merge_queue_run_test.go and
// merge_queue_wait_test.go, and three branches with mqoExec, a fake origin of
// its own, for the director's two stages. No test runs gh or reaches a
// network.

// mqoMovedListing is origin with main moved to moMain2 since the gate
// (moMain) and feat/a still at the gate's head.
func mqoMovedListing() string { return moListing("main", "main", moMain2, "feat/a", moHeadA) }

// mqoMovedRebase is the Detail of feat/a, gated on moMain by org-a, which is
// disbanded, once origin/main moved to moMain2, with no merge queue to keep
// it ready.
const mqoMovedRebase = "origin/main moved: gate base 111111111111, now 222222222222: run ralph org regate --branch feat/a, " +
	"which merges origin/main in and runs the gate again"

// mqoMovedAtStep7 are the commands checkMerge runs when every check passes on
// the queue path with origin/main moved past the gate's base before step 5:
// mxQueuePath with the read of the merge queue at step 7, after gh pr list,
// and none at step 11.
var mqoMovedAtStep7 = []string{mxGetURL, mxLsRemote, mxPRList, mxQueue, mxPRView, mxPRChecks, mxRepoAPI, mxRules, mxLsRemote}

// mqoNotReady is the refusal of step 7 for feat/a, listed as class with
// detail.
func mqoNotReady(class, detail string) string {
	return "org: merge: ralph org merge-order lists the branch feat/a as " + class + ", not ready or accepted, so nothing was " +
		"merged: " + detail
}

// mqoMoved is the refusal of step 13 for what moved.
func mqoMoved(what string) string {
	return "org: merge: origin moved while the checks ran: " + what + ", so nothing was merged: run ralph org merge-order again " +
		"and follow the branch's Detail"
}

// mqoQueueFixture is newMXFixture on lines (mxReadyLedger when nil) with a
// merge queue whose configuration's method is SQUASH and no answer for gh
// repo view, as newMXQueueFixture is, and with ls as the answers of the
// ls-remotes, the last one again after them.
func mqoQueueFixture(t *testing.T, lines []any, ls ...string) *mxFixture {
	t.Helper()
	fx := newMXFixture(t, lines)
	fx.exec.replies[mxQueue] = mgReply{out: mxQueueJSON(`{"mergeMethod":"SQUASH"}`)}
	delete(fx.exec.replies, mxRepoView)
	fx.exec.ls = nil
	for _, out := range ls {
		fx.exec.ls = append(fx.exec.ls, mgReply{out: out})
	}
	return fx
}

// TestCheckMerge_QueueKeepsABranchWhoseBaseMovedReady covers AC2 and AC7 for
// the checks: on a default branch with a merge queue, a ready or an accepted
// branch whose gate base origin/main moved past, before step 5 or only
// before step 13, passes every check on the queue path with the gate's base
// kept, and the GraphQL mergeQueue is read once in all: at step 7 when the
// base moved before it, which step 11 then takes instead of a read of its
// own, else at step 11. The read has the enforcement timeout's deadline,
// every command is read-only and pinned to origin's repository, and the
// refusal kind is none.
func TestCheckMerge_QueueKeepsABranchWhoseBaseMovedReady(t *testing.T) {
	acceptedLines, accepted := mxAcceptedLedger(t)
	for _, tc := range []struct {
		name     string
		lines    []any
		accepted *GateAcceptance
		gateAt   string
		ls       []string
		keys     []string
	}{
		{"ready, main moved before step 5", nil, nil, mqrGateTS, []string{mqoMovedListing()}, mqoMovedAtStep7},
		{"accepted, main moved before step 5", acceptedLines, &accepted, "2026-10-14T07:10:00Z", []string{mqoMovedListing()},
			mqoMovedAtStep7},
		{"ready, main moved only before step 13", nil, nil, mqrGateTS, []string{mxListing(), mqoMovedListing()}, mxQueuePath},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := mqoQueueFixture(t, tc.lines, tc.ls...)
			start := time.Now()
			r, refused, kind, err := fx.checkKind(context.Background(), t, fx.params())
			end := time.Now()
			if err != nil || refused != "" || kind != mergeRefusalOther {
				t.Fatalf("checkMerge = %v, kind %d, refused %q", err, kind, refused)
			}
			if (r.accepted == nil) != (tc.accepted == nil) || tc.accepted != nil && *r.accepted != *tc.accepted {
				t.Fatalf("mergeReady.accepted = %+v, want %+v", r.accepted, tc.accepted)
			}
			r.accepted = nil
			want := mergeReady{
				branch: "feat/a", orgID: "org-a", apiHost: "github.com", repo: mergeRepo{owner: "o", name: "r"}, repoArg: mxRepoArg,
				originHost: "github.com", prNumber: 7, prURL: "https://github.com/o/r/pull/7", head: moHeadA, base: moMain,
				defaultBranch: "main", queue: true, gateAt: mqTime(t, tc.gateAt), method: "squash",
				protection: mxQueueProtection("GraphQL mergeQueue is set", "rulesets require 1 required status check"),
			}
			if r != want {
				t.Fatalf("mergeReady =\n%+v\nwant\n%+v", r, want)
			}
			if got := fx.exec.keys(); !slices.Equal(got, tc.keys) {
				t.Errorf("commands = %q\nwant %q", got, tc.keys)
			}
			for _, c := range fx.exec.snapshot() {
				if c.key == mxQueue && (c.deadline.Before(start.Add(defaultMergeGuaranteeTimeout)) ||
					c.deadline.After(end.Add(defaultMergeGuaranteeTimeout))) {
					t.Errorf("the read of the merge queue has the deadline %v; want the enforcement timeout from the start", c.deadline)
				}
			}
			mxAssertPinned(t, fx.exec, "github.com", mxRepoArg)
		})
	}
}

// TestMerge_QueueEnqueuesABranchWhoseBaseMovedWithoutARegate covers AC2 for
// Merge: on a default branch with a merge queue, Merge with --wait-checks 0
// on a ready branch whose gate base origin/main moved past adds the PR to the
// queue at the gate's head as the queue path does with a base that did not
// move (mqrAdded), with one read of the queue, at step 7, no regate, and
// nothing recorded.
func TestMerge_QueueEnqueuesABranchWhoseBaseMovedWithoutARegate(t *testing.T) {
	fx := newMQRFixture(t, nil)
	fx.exec.ls = []mgReply{{out: mqoMovedListing()}}
	res, err := fx.merge(context.Background(), t, fx.queueParams(t))
	if want := mqrAdded(); err != nil || res != want {
		t.Fatalf("Merge = %+v, %v\nwant %+v", res, err, want)
	}
	fx.assertRan(t, slices.Concat(mqoMovedAtStep7, []string{mqrKeyRead, mqrKeyEnqueue}))
	if c := fx.call(t, mqrKeyEnqueue); !slices.Equal(append([]string{c.name}, c.args...), mqrEnqueueArgv) {
		t.Errorf("enqueuePullRequest = %s %q\nwant %q", c.name, c.args, mqrEnqueueArgv)
	}
}

// TestCheckMerge_QueueStillRefusesWhatMovedBesidesTheBase covers AC2 and AC7
// for the refusals a merge queue does not lift: a branch pushed after the
// gate is refused at step 7 as regate, and a gate whose base_ref names
// another branch as rebase, each with no read of the queue there; at step 13
// the branch moved or gone and origin's default branch renamed are refused
// as with no queue, whether origin/main moved before step 5 or only before
// step 13.
func TestCheckMerge_QueueStillRefusesWhatMovedBesidesTheBase(t *testing.T) {
	onDevelop := moGate("org-a", "feat/a", moHeadA, moMain, "refs/remotes/origin/develop", "")
	for _, tc := range []struct {
		name  string
		lines []any
		ls    []string
		want  string
		keys  []string
	}{
		{"pushed after the gate, main moved (step 7)", nil, []string{moListing("main", "main", moMain2, "feat/a", moHeadB)},
			mqoNotReady("regate", "pushed after the gate, or the gate's head was not pushed: origin has bbbbbbbbbbbb, the gate "+
				"checked aaaaaaaaaaaa: run ralph org regate --branch feat/a"), []string{mxGetURL, mxLsRemote, mxPRList}},
		{"gated against another branch (step 7)", []any{moBind("org-a", "a"), onDevelop, moDisbanded("org-a")},
			[]string{mxListing()}, mqoNotReady("rebase", "the gate checked against refs/remotes/origin/develop, but origin's "+
				"default branch is now main: run ralph org regate --branch feat/a, which merges origin/main in and runs the gate "+
				"again"), []string{mxGetURL, mxLsRemote, mxPRList}},
		{"the branch moved (step 13)", nil, []string{mqoMovedListing(), moListing("main", "main", moMain2, "feat/a", moHeadB)},
			mqoMoved("origin's feat/a moved to bbbbbbbbbbbb from aaaaaaaaaaaa, the head the gate passed"), mqoMovedAtStep7},
		{"the branch moved, main moved only before step 13", nil, []string{mxListing(),
			moListing("main", "main", moMain2, "feat/a", moHeadB)},
			mqoMoved("origin's feat/a moved to bbbbbbbbbbbb from aaaaaaaaaaaa, the head the gate passed"), mxQueuePath},
		{"the branch is gone (step 13)", nil, []string{mqoMovedListing(), moListing("main", "main", moMain2)},
			mqoMoved("origin has no branch feat/a any more"), mqoMovedAtStep7},
		{"the default branch is another (step 13)", nil, []string{mqoMovedListing(),
			moListing("trunk", "trunk", moMain2, "feat/a", moHeadA)}, mqoMoved("origin's default branch is now trunk, not main"),
			mqoMovedAtStep7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := mqoQueueFixture(t, tc.lines, tc.ls...)
			r, refused, kind, err := fx.checkKind(context.Background(), t, fx.params())
			if err != nil || refused != tc.want || r != (mergeReady{}) || kind.waits() {
				t.Fatalf("checkMerge = %+v, %v, kind %d\nrefused %q\nwant    %q", r, err, kind, refused, tc.want)
			}
			if got := fx.exec.keys(); !slices.Equal(got, tc.keys) {
				t.Errorf("commands = %q\nwant %q", got, tc.keys)
			}
		})
	}
}

// TestCheckMerge_NoQueueKeepsTodaysCommands covers AC3: with no merge queue
// (GraphQL's mergeQueue null), a branch whose gate base origin/main did not
// move runs exactly today's commands, mxAll for the checks and mrMerged for
// Merge, with the queue read once, at step 11; one whose gate base origin/main
// moved past is refused at step 7 as rebase with today's Detail, after one
// read of the queue there and nothing else.
func TestCheckMerge_NoQueueKeepsTodaysCommands(t *testing.T) {
	t.Run("base where the gate left it", func(t *testing.T) {
		fx := newMXFixture(t, nil)
		r, refused, err := fx.check(context.Background(), t, fx.params())
		if err != nil || refused != "" || r.queue {
			t.Fatalf("checkMerge = %+v, %v, refused %q; want the checks to pass with no queue", r, err, refused)
		}
		if got := fx.exec.keys(); !slices.Equal(got, mxAll) {
			t.Errorf("commands = %q\nwant %q", got, mxAll)
		}
	})
	t.Run("Merge with the base where the gate left it", func(t *testing.T) {
		fx := newMRFixture(t, nil)
		res, err := fx.merge(context.Background(), t, fx.params())
		if err != nil || !res.Recorded {
			t.Fatalf("Merge = %+v, %v; want a recorded merge", res, err)
		}
		if got := fx.exec.keys(); !slices.Equal(got, mrMerged) {
			t.Errorf("commands = %q\nwant %q", got, mrMerged)
		}
	})
	t.Run("base moved", func(t *testing.T) {
		fx := newMXFixture(t, nil)
		fx.exec.ls = []mgReply{{out: mqoMovedListing()}}
		r, refused, kind, err := fx.checkKind(context.Background(), t, fx.params())
		if want := mqoNotReady("rebase", mqoMovedRebase); err != nil || refused != want || r != (mergeReady{}) || kind.waits() {
			t.Fatalf("checkMerge = %+v, %v, kind %d\nrefused %q\nwant    %q", r, err, kind, refused, want)
		}
		if got, want := fx.exec.keys(), []string{mxGetURL, mxLsRemote, mxPRList, mxQueue}; !slices.Equal(got, want) {
			t.Errorf("commands = %q\nwant %q", got, want)
		}
	})
}

// TestCheckMerge_QueueReadAtStep7ThatFails covers AC2 for a read of the
// merge queue at step 7 that fails, with origin/main moved past the gate's
// base: a gh that fails, an answer with an errors list, and a read that runs
// out of the enforcement timeout are refused as rebase with today's Detail
// and, at its end, why the queue could not be read, so the refusal does not
// read as a reason to regate alone; a gh that is not found is refused as a gh
// not on PATH. Nothing after the read runs.
func TestCheckMerge_QueueReadAtStep7ThatFails(t *testing.T) {
	const cmd = "gh api --hostname github.com graphql (mergeQueue of main)"
	const unknown = ", so whether " + mxSubject + ", has a merge queue is unknown"
	unread := func(why string) string {
		return mqoNotReady("rebase", mqoMovedRebase+"; the merge queue of origin/main could not be read, and with one the branch "+
			"would be ready: "+why)
	}
	for _, tc := range []struct {
		name  string
		setup func(fx *mxFixture, p *MergeParams)
		want  string
	}{
		{"gh fails", func(fx *mxFixture, _ *MergeParams) {
			fx.exec.replies[mxQueue] = mgReply{err: errors.New("exit status 1: gh: HTTP 401: Bad credentials")}
		}, unread(cmd + " failed (exit status 1: gh: HTTP 401: Bad credentials)" + unknown + meAccess)},
		{"an errors list", func(fx *mxFixture, _ *MergeParams) {
			fx.exec.replies[mxQueue] = mgReply{out: `{"data":null,"errors":[{"message":"Resource not accessible"}]}`}
		}, unread(cmd + ` answered with errors ([{"message":"Resource not accessible"}])` + unknown + meAccess)},
		{"timed out", func(fx *mxFixture, p *MergeParams) {
			fx.exec.block, p.EnforcementTimeout = mxQueue, 30*time.Millisecond
		}, unread("the check timed out after 30ms while " + cmd + " ran, so the protection of origin's repository's default " +
			"branch is unknown: check the network and gh's access")},
		{"gh not found", func(fx *mxFixture, _ *MergeParams) { fx.exec.replies[mxQueue] = mxNotFound }, mxGHMissing},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMXFixture(t, nil)
			fx.exec.ls = []mgReply{{out: mqoMovedListing()}}
			p := fx.params()
			tc.setup(fx, &p)
			r, refused, kind, err := fx.checkKind(context.Background(), t, p)
			if err != nil || refused != tc.want || r != (mergeReady{}) || kind.waits() {
				t.Fatalf("checkMerge = %+v, %v, kind %d\nrefused %q\nwant    %q", r, err, kind, refused, tc.want)
			}
			if got, want := fx.exec.keys(), []string{mxGetURL, mxLsRemote, mxPRList, mxQueue}; !slices.Equal(got, want) {
				t.Errorf("commands = %q\nwant %q", got, want)
			}
		})
	}
}

// TestCheckMerge_QueueReadAtStep7Interrupted: ctx done while step 7 reads the
// merge queue is an error that says so and that nothing was merged, with no
// refusal and nothing after the read run.
func TestCheckMerge_QueueReadAtStep7Interrupted(t *testing.T) {
	fx := mqoQueueFixture(t, nil, mqoMovedListing())
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	fx.exec.after = func(i int) {
		if i == 3 {
			cancel(errors.New("test stopped"))
		}
	}
	r, refused, err := fx.check(ctx, t, fx.params())
	const want = "org: merge: interrupted (test stopped) while the merge queue of origin/main was read: nothing was merged"
	if err == nil || err.Error() != want || refused != "" || r != (mergeReady{}) {
		t.Fatalf("checkMerge = %+v, %v, refused %q; want %q", r, err, refused, want)
	}
	if got, want := fx.exec.keys(), []string{mxGetURL, mxLsRemote, mxPRList, mxQueue}; !slices.Equal(got, want) {
		t.Errorf("commands = %q\nwant %q", got, want)
	}
}

// TestMerge_QueueIsReadAgainInEachRunOfTheChecks covers FR-12's one read per
// run of the checks: with origin/main moved past the gate's base on a
// default branch with a merge queue and the PR's checks pending, the first
// run reads the queue at step 7 and waits at step 10; the run after the wait
// reads it again at step 7, takes that read at step 11, enqueues the PR, and
// the wait for the queue records the merge.
func TestMerge_QueueIsReadAgainInEachRunOfTheChecks(t *testing.T) {
	fx := newMQWFixture(t, nil, 30*time.Minute, mqrPR(nil), mqwMerged)
	fx.exec.ls = []mgReply{{out: mqoMovedListing()}}
	fx.exec.replies[mxPRChecks] = mwPending
	fx.onSleep = func(n int) {
		if n == 1 {
			fx.setChecks(mwPass)
		}
	}
	res, err := fx.run(t)
	if want := mqwMergedResult(mqwOID); err != nil || res != want {
		t.Fatalf("Merge = %+v, %v\nwant %+v", res, err, want)
	}
	keys := slices.Concat(mqoMovedAtStep7[:6], []string{mxPRChecks}, mqoMovedAtStep7, []string{mqrKeyRead, mqrKeyEnqueue, mqrKeyRead})
	fx.check(t, keys, []time.Duration{30 * time.Second, 30 * time.Second}, []string{
		mwStart("has checks pending", "30m0s", "30m0s"), mwEnd("30s", "the checks passed"), mqwStart("29m30s", "30m0s"),
		mqwEnd("30s", "it was merged into main at the head aaaaaaaaaaaa")})
	if got, want := fx.appended(t), mqwRecord(t, mqwOID, ""); got != want {
		t.Errorf("appended %q\nwant     %q", got, want)
	}
}

// TestMergeCheck_QueueReadForTakesOnlyTheBranchStep7Read: step 11 gets step
// 7's read of the merge queue only for the default branch step 7 read it
// for, and no read when step 7 made none.
func TestMergeCheck_QueueReadForTakesOnlyTheBranchStep7Read(t *testing.T) {
	present := mergeQueueLookup{state: mergeQueuePresent, method: "SQUASH"}
	m := &mergeCheck{queue: present, queueOf: "main"}
	if got := m.queueReadFor("main"); got != present {
		t.Errorf("queueReadFor(main) = %+v, want %+v", got, present)
	}
	if got := m.queueReadFor("trunk"); got != (mergeQueueLookup{}) {
		t.Errorf("queueReadFor(trunk) = %+v, want no read", got)
	}
	if got := (&mergeCheck{}).queueReadFor("main"); got != (mergeQueueLookup{}) {
		t.Errorf("queueReadFor(main) with no read = %+v, want no read", got)
	}
}

// setChecks makes gh pr checks answer r from now on.
func (fx *mqwFixture) setChecks(r mgReply) {
	fx.exec.mu.Lock()
	defer fx.exec.mu.Unlock()
	fx.exec.replies[mxPRChecks] = r
}

// mqoHeadC is feat/c's head in mqoExec, and mqoMergeB the commit the merge
// queue merges feat/b's PR as.
var (
	mqoHeadC  = strings.Repeat("c", 40)
	mqoMergeB = strings.Repeat("e", 40)
)

// mqoPR is a PR of mqoExec's origin: the PR of branch, number, with the node
// ID id, at head, in state (OPEN or MERGED), in the merge queue at position
// when inQueue, with mergeCommit once MERGED, and checks, the answer of its
// gh pr checks.
type mqoPR struct {
	branch, id, head, state, mergeCommit string
	number, position                     int
	inQueue                              bool
	checks                               mgReply
}

// url is the PR's url in o/r on github.com.
func (pr *mqoPR) url() string { return fmt.Sprintf("https://github.com/o/r/pull/%d", pr.number) }

// answer is the read of the PR's merge queue state (mergeQueuePRQuery).
func (pr *mqoPR) answer() string {
	f := map[string]any{"id": pr.id, "state": pr.state, "headRefOid": pr.head, "baseRefName": "main", "mergeCommit": nil,
		"isInMergeQueue": pr.inQueue, "mergeQueueEntry": nil, "timelineItems": mqNodes()}
	if pr.inQueue {
		f["mergeQueueEntry"] = mqEntry("AWAITING_CHECKS", pr.position, mqrEnqueuedAt)
	}
	if pr.mergeCommit != "" {
		f["mergeCommit"] = map[string]any{"oid": pr.mergeCommit}
	}
	return mqPRAnswer(f)
}

// mqoExec is a fake MergeOrderExec for Merge on branches of o/r on
// github.com whose default branch main is at tip and has a merge queue
// (method SQUASH), with a ruleset that requires the check verify: origin's
// URL, the ls-remote of main and of each PR's branch at its head, ssh -G of
// github.com, and per PR (by --head, by number, or by node ID) gh pr list,
// view and checks, the read of its merge queue state, and enqueuePullRequest
// at its head, which puts it at the end of the queue. queueReads counts the
// reads of main's merge queue (mergeQueueQuery), and enqueued lists the
// branches in the order enqueuePullRequest added them. Any other call fails.
type mqoExec struct {
	mu         sync.Mutex
	tip        string
	prs        []*mqoPR
	queueReads int
	enqueued   []string
}

func (e *mqoExec) run(ctx context.Context, _ string, _ []string, name string, args ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	switch {
	case name == "git" && slices.Equal(args, []string{"remote", "get-url", "origin"}):
		return moOriginURL + "\n", nil
	case name == "git" && slices.Equal(args, mxLsRemoteArgs):
		pairs := []string{"main", e.tip}
		for _, pr := range e.prs {
			pairs = append(pairs, pr.branch, pr.head)
		}
		return moListing("main", pairs...), nil
	case name == "ssh":
		return "hostname " + args[len(args)-1] + "\n", nil
	case name == "gh" && len(args) > 2 && args[0] == "pr":
		return e.ghPR(args)
	case name == "gh" && len(args) > 0 && args[0] == "api":
		return e.ghAPI(args)
	}
	return "", fmt.Errorf("mqoExec: unexpected %s %q", name, args)
}

// ghPR answers gh pr list, view and checks.
func (e *mqoExec) ghPR(args []string) (string, error) {
	var pr *mqoPR
	if args[1] == "list" {
		pr = e.find(func(pr *mqoPR) bool { return pr.branch == args[slices.Index(args, "--head")+1] })
	} else {
		pr = e.find(func(pr *mqoPR) bool { return strconv.Itoa(pr.number) == args[2] })
	}
	switch {
	case pr == nil:
	case args[1] == "list":
		return moPRs(ghPullRequest{Number: pr.number, State: pr.state, URL: pr.url(), HeadRefOid: pr.head, BaseRefName: "main"}), nil
	case args[1] == "view":
		return mxPRViewJSON(pr.state, pr.head, "main", pr.url(), "null"), nil
	case args[1] == "checks":
		return pr.checks.out, pr.checks.err
	}
	return "", fmt.Errorf("mqoExec: unexpected gh %q", args)
}

// ghAPI answers gh api: the repository, the rules of main, and the three
// GraphQL calls by their query.
func (e *mqoExec) ghAPI(args []string) (string, error) {
	field := func(name string) string {
		for _, a := range args {
			if v, ok := strings.CutPrefix(a, name+"="); ok {
				return v
			}
		}
		return ""
	}
	switch path := args[len(args)-1]; {
	case path == "repos/o/r":
		return meRepoJSON("main", "o/r", nil, false), nil
	case path == "repos/o/r/rules/branches/main":
		return mgRules(meChecksRule(1, true, "verify")), nil
	case field("query") == mergeQueueQuery:
		e.queueReads++
		return mxQueueJSON(`{"mergeMethod":"SQUASH"}`), nil
	case field("query") == mergeQueuePRQuery:
		if pr := e.find(func(pr *mqoPR) bool { return strconv.Itoa(pr.number) == field("number") }); pr != nil {
			return pr.answer(), nil
		}
	case field("query") == mergeQueueEnqueueMutation:
		pr := e.find(func(pr *mqoPR) bool { return pr.id == field("pullRequestId") })
		if pr == nil || pr.head != field("expectedHeadOid") || pr.state != "OPEN" || pr.inQueue {
			break
		}
		pr.position = 1 + len(slices.DeleteFunc(slices.Clone(e.prs), func(pr *mqoPR) bool { return !pr.inQueue }))
		pr.inQueue = true
		e.enqueued = append(e.enqueued, pr.branch)
		return mqEnqueueAnswer(mqEntry("QUEUED", pr.position, mqrEnqueuedAt)), nil
	}
	return "", fmt.Errorf("mqoExec: unexpected gh %q", args)
}

// find is the first PR f holds for, nil for none.
func (e *mqoExec) find(f func(pr *mqoPR) bool) *mqoPR {
	if i := slices.IndexFunc(e.prs, f); i >= 0 {
		return e.prs[i]
	}
	return nil
}

// mergeByQueue is the merge queue merging branch's PR as commit: the PR is
// MERGED and out of the queue, those behind it move up, and main's tip is
// commit.
func (e *mqoExec) mergeByQueue(branch, commit string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	pr := e.find(func(pr *mqoPR) bool { return pr.branch == branch })
	for _, other := range e.prs {
		if other.inQueue && other.position > pr.position {
			other.position--
		}
	}
	pr.state, pr.mergeCommit, pr.inQueue, pr.position = "MERGED", commit, false, 0
	e.tip = commit
}

// setChecks makes gh pr checks of branch's PR answer r from now on.
func (e *mqoExec) setChecks(branch string, r mgReply) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.find(func(pr *mqoPR) bool { return pr.branch == branch }).checks = r
}

// mqoFixture is a ledger with three ready branches on mqoExec: feat/a,
// feat/b and feat/c, each bound to its org (org-a, org-b, org-c), gated on
// moMain at its head (moHeadA, moHeadB, mqoHeadC), and disbanded, whose PRs
// #7, #8 and #9 are open into main with the check verify passed and none in
// the queue; main is at moMain.
type mqoFixture struct {
	mo *moFixture
	ex *mqoExec
}

func newMQOFixture(t *testing.T) *mqoFixture {
	t.Helper()
	fx := &mqoFixture{mo: newMOFixture(t), ex: &mqoExec{tip: moMain}}
	for i, b := range []struct{ slug, head string }{{"a", moHeadA}, {"b", moHeadB}, {"c", mqoHeadC}} {
		org, branch := "org-"+b.slug, "feat/"+b.slug
		fx.mo.add(t, moBind(org, b.slug), moPassed(org, branch, b.head, moMain), moDisbanded(org))
		fx.ex.prs = append(fx.ex.prs, &mqoPR{branch: branch, id: "PR_kwDO" + b.slug, head: b.head, state: "OPEN", number: 7 + i,
			checks: mgReply{out: mxChecks(mxPass("verify"))}})
	}
	return fx
}

// merge runs Merge on branch with --wait-checks wait and a fake clock whose
// sleeps run onSleep (mwClock), and returns its result, the reads of the
// merge queue it made (mqoExec.queueReads), and its error. A Merge with no
// budget that sleeps fails the test.
func (fx *mqoFixture) merge(t *testing.T, branch string, wait time.Duration, onSleep func(n int)) (MergeResult, int, error) {
	t.Helper()
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	clock := &mwClock{t: t, now: time.Date(2026, 10, 14, 9, 0, 30, 0, time.UTC), stop: cancel, onSleep: onSleep}
	fx.ex.mu.Lock()
	before := fx.ex.queueReads
	fx.ex.mu.Unlock()
	res, err := fx.mo.o.Merge(ctx, MergeParams{Branch: branch, Dir: "/repo", Exec: fx.ex.run, LookPath: mgFound, WaitChecks: wait,
		waitTest: mergeWaitTest{now: clock.Now, sleep: clock.sleep}})
	if wait == 0 && len(clock.sleeps) > 0 {
		t.Errorf("Merge of %s with --wait-checks 0 slept %v", branch, clock.sleeps)
	}
	fx.ex.mu.Lock()
	defer fx.ex.mu.Unlock()
	return res, fx.ex.queueReads - before, err
}

// pr is the fixture's PR of branch.
func (fx *mqoFixture) pr(branch string) *mqoPR {
	return fx.ex.find(func(pr *mqoPR) bool { return pr.branch == branch })
}

// mqoResult is the MergeResult fields the checks set for pr's branch on the
// queue path.
func mqoResult(pr *mqoPR) MergeResult {
	return MergeResult{Branch: pr.branch, OrgID: "org-" + strings.TrimPrefix(pr.branch, "feat/"), PRNumber: pr.number,
		PRURL: pr.url(), Method: "squash", Head: pr.head,
		Protection: mxQueueProtection("GraphQL mergeQueue is set", "rulesets require 1 required status check")}
}

// mqoAdded is mqoResult for pr once Merge added it to the merge queue at
// position.
func mqoAdded(pr *mqoPR, position int) MergeResult {
	res := mqoResult(pr)
	res.Queued = fmt.Sprintf("org: merge: PR #%d (%s) was added to the merge queue of main at the head %s, the head the gate "+
		"passed, and it is at position %d, enqueued at %s; it is not merged yet, so nothing was recorded: run ralph org merge "+
		"--branch %s again, which reads the merge queue first and adds the PR only when it is not there, and once GitHub merged "+
		"the PR, run ralph org cleanup --branch %s to clean up the branch's feature worktree", pr.number, pr.url(),
		shortMergeSHA(pr.head), position, mqrEnqueuedAt, pr.branch, pr.branch)
	res.QueuePosition, res.EnqueuedAt = position, mqrEnqueuedAt
	return res
}

// TestMerge_QueueTwoStagesForThreeReadyBranches covers AC4 (AC11 of the
// spec) for the director's two stages on a default branch with a merge
// queue. Stage 1: Merge with --wait-checks 0 on each of three ready branches
// sends enqueuePullRequest once for each, in that order, and leaves each PR
// in the queue (Queued, at positions 1 to 3, which the CLI exits 7 for),
// each run of the checks reading the queue once, at step 11. Then the queue
// merges feat/a and origin/main moves past every gate's base. Stage 2: Merge
// with a budget on feat/b passes steps 7 and 13 with no regate, reading the
// queue once, at step 7; finds the PR in the queue (FR-5) and sends no
// enqueuePullRequest; waits; reads it MERGED; and records pr_merged with
// via=merge_queue.
func TestMerge_QueueTwoStagesForThreeReadyBranches(t *testing.T) {
	fx := newMQOFixture(t)
	for i, branch := range []string{"feat/a", "feat/b", "feat/c"} {
		res, reads, err := fx.merge(t, branch, 0, nil)
		if want := mqoAdded(fx.pr(branch), i+1); err != nil || res != want || reads != 1 {
			t.Fatalf("stage 1, Merge of %s = %+v, %v, %d reads of the queue\nwant %+v, 1 read", branch, res, err, reads, want)
		}
	}
	if want := []string{"feat/a", "feat/b", "feat/c"}; !slices.Equal(fx.ex.enqueued, want) {
		t.Fatalf("stage 1 enqueued %q, want %q", fx.ex.enqueued, want)
	}
	if got := mrRead(t, fx.mo.path); strings.Contains(got, EventPRMerged) {
		t.Fatalf("stage 1 recorded a merge: %q", got)
	}

	fx.ex.mergeByQueue("feat/a", moMain2)
	before := mrRead(t, fx.mo.path)
	res, reads, err := fx.merge(t, "feat/b", 30*time.Minute, func(n int) {
		if n == 1 {
			fx.ex.mergeByQueue("feat/b", mqoMergeB)
		}
	})
	want := mqoResult(fx.pr("feat/b"))
	want.Merged, want.Recorded, want.MergeCommit = true, true, mqoMergeB
	want.QueueNote = "PR #8 was in the merge queue already at the head the gate passed, so ralph org merge did not add it again"
	if err != nil || res != want || reads != 1 {
		t.Fatalf("stage 2, Merge of feat/b = %+v, %v, %d reads of the queue\nwant %+v, 1 read", res, err, reads, want)
	}
	if want := []string{"feat/a", "feat/b", "feat/c"}; !slices.Equal(fx.ex.enqueued, want) {
		t.Errorf("after stage 2 enqueued %q, want %q: no second enqueuePullRequest", fx.ex.enqueued, want)
	}
	record, err := json.Marshal(ManifestEvent{TS: mergeOrderTestListedAt, OrgID: "org-b", Event: EventPRMerged,
		Details: "branch=feat/b pr=8 head=" + moHeadB + " method=squash merge_commit=" + mqoMergeB + " auto_merge=false via=merge_queue"})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimPrefix(mrRead(t, fx.mo.path), before); got != string(record)+"\n" {
		t.Errorf("stage 2 appended %q\nwant %q", got, string(record)+"\n")
	}
	mrAssertLockFree(t, fx.mo.stateDir)
}

// TestMerge_QueueDeferredBranchGoesInOnceItsChecksPass covers the deferred
// re-run of AC4 (a finding of the plan advisory): with feat/a's checks
// pending and feat/b's passed, Merge with --wait-checks 0 refuses feat/a with
// nothing enqueued and no read of the queue, and enqueues feat/b; once the
// queue merged feat/b, so origin/main moved past feat/a's gate base, and
// feat/a's checks passed, Merge with --wait-checks 0 on feat/a reads the
// queue once, at step 7, and enqueues it. enqueuePullRequest ran for feat/b,
// then feat/a.
func TestMerge_QueueDeferredBranchGoesInOnceItsChecksPass(t *testing.T) {
	fx := newMQOFixture(t)
	fx.ex.setChecks("feat/a", mwPending)
	res, reads, err := fx.merge(t, "feat/a", 0, nil)
	if want := (MergeResult{Refused: mwPendingRefusal}); err != nil || res != want || reads != 0 || len(fx.ex.enqueued) != 0 {
		t.Fatalf("Merge of feat/a, checks pending = %+v, %v, %d reads, enqueued %q\nwant %+v, no read, nothing enqueued", res, err,
			reads, fx.ex.enqueued, want)
	}
	res, reads, err = fx.merge(t, "feat/b", 0, nil)
	if want := mqoAdded(fx.pr("feat/b"), 1); err != nil || res != want || reads != 1 {
		t.Fatalf("Merge of feat/b = %+v, %v, %d reads\nwant %+v, 1 read", res, err, reads, want)
	}

	fx.ex.mergeByQueue("feat/b", moMain2)
	fx.ex.setChecks("feat/a", mwPass)
	res, reads, err = fx.merge(t, "feat/a", 0, nil)
	if want := mqoAdded(fx.pr("feat/a"), 1); err != nil || res != want || reads != 1 {
		t.Fatalf("Merge of feat/a again = %+v, %v, %d reads\nwant %+v, 1 read", res, err, reads, want)
	}
	if want := []string{"feat/b", "feat/a"}; !slices.Equal(fx.ex.enqueued, want) {
		t.Errorf("enqueued %q, want %q", fx.ex.enqueued, want)
	}
	if got := mrRead(t, fx.mo.path); strings.Contains(got, EventPRMerged) {
		t.Errorf("Merge recorded a merge: %q", got)
	}
}
