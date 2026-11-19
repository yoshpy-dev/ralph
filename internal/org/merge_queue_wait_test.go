package org

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// The wait for the merge queue of `ralph org merge --wait-checks` and what
// follows it (merge_queue_wait.go; plan
// docs/plans/active/2026-11-20-org-merge-queue-verb.md, S3b: AC1, AC2, AC4 for
// a budget that runs out, AC8, AC12, AC13 and AC15, D1, D2, D5 to D7; spec
// docs/specs/2026-11-19-org-merge-queue.md, FR-6 to FR-11) with mqwFixture:
// the queue fixture of merge_queue_run_test.go with the fake clock of
// merge_wait_test.go (mwClock), so no test sleeps for the 30 s interval, and
// the read's answer changed while Merge sleeps. No test runs gh or reaches a
// network.

// The removal and the times of these tests: the fixture's enqueuePullRequest
// enqueues the PR at mqrEnqueuedAt (09:00:30), and GitHub removes it after.
const (
	mqwRemovedAt = "2026-10-14T09:01:40Z"
	mqwOID       = mqMerge
)

// mqwQueuedAt is the read of PR #7 in the merge queue at position, enqueued
// at mqrEnqueuedAt.
func mqwQueuedAt(position int) mgReply {
	return mqrPR(map[string]any{"isInMergeQueue": true, "mergeQueueEntry": mqEntry("AWAITING_CHECKS", position, mqrEnqueuedAt),
		"timelineItems": mqNodes(mqAdded(mqrEnqueuedAt))})
}

// mqwMergedAt is the read of PR #7 MERGED at head into base, out of the
// queue, with commit as its mergeCommit (nil for null).
func mqwMergedAt(head, base string, commit any) mgReply {
	return mqrPR(map[string]any{"state": "MERGED", "headRefOid": head, "baseRefName": base, "mergeCommit": commit,
		"timelineItems": mqNodes(mqAdded(mqrEnqueuedAt))})
}

// mqwMerged is the read of PR #7 MERGED at the gate's head into main with the
// merge commit mqwOID.
var mqwMerged = mqwMergedAt(moHeadA, "main", map[string]any{"oid": mqwOID})

// mqwOut is the read of PR #7 out of the queue and not merged, in state,
// with the timeline nodes.
func mqwOut(state string, nodes ...any) mgReply {
	return mqrPR(map[string]any{"state": state, "timelineItems": mqNodes(nodes...)})
}

// mqwKeys are the commands of a queue path that sent enqueuePullRequest and
// then read the PR's merge queue state reads times in the wait.
func mqwKeys(reads int) []string {
	return slices.Concat(mqrEnqueued, slices.Repeat([]string{mqrKeyRead}, reads))
}

// mqwStart is the line the wait writes when it starts with the PR at
// position 1, left being what is left of budget.
func mqwStart(left, budget string) string {
	return "org: merge: PR #7 is in the merge queue of main at position 1, so ralph org merge waits for it to merge for up to " +
		left + " (--wait-checks " + budget + "), with regate.lock released, reading the PR's merge queue state every 30s"
}

// mqwEnd is the line the wait writes when it ends after after, why being why.
func mqwEnd(after, why string) string {
	return "org: merge: stopped waiting for PR #7 in the merge queue after " + after + ": " + why
}

// mqwRecord is the manifest line of the fixture's pr_merged through the
// queue with commit, gate being " gate=accepted" or "".
func mqwRecord(t *testing.T, commit, gate string) string {
	t.Helper()
	return mrRecord(t, "branch=feat/a pr=7 head="+moHeadA+" method=squash merge_commit="+commit+" auto_merge=false"+gate+
		" via=merge_queue")
}

// mqwMergedResult is mqrResult once the queue merged the PR and Merge
// recorded it with the merge commit commit.
func mqwMergedResult(commit string) MergeResult {
	res := mqrResult()
	res.Merged, res.Recorded, res.MergeCommit = true, true, commit
	return res
}

// mqwFixture is mqrFixture with the wait of mwFixture: Merge's params with
// WaitChecks and Progress, a ctx the test stops, and mwClock, whose every
// sleep takes regate.lock and releases it (lockTaken counts them), then makes
// the read answer reads[n-1] from the sleep numbered n on (the last one again
// after them), then runs onSleep.
type mqwFixture struct {
	*mqrFixture
	clock     *mwClock
	progress  *strings.Builder
	ctx       context.Context
	cancel    context.CancelCauseFunc
	p         MergeParams
	reads     []mgReply
	onSleep   func(n int)
	lockTaken int
}

func newMQWFixture(t *testing.T, lines []any, wait time.Duration, reads ...mgReply) *mqwFixture {
	t.Helper()
	fx := &mqwFixture{mqrFixture: newMQRFixture(t, lines), progress: &strings.Builder{}, reads: reads}
	fx.ctx, fx.cancel = context.WithCancelCause(context.Background())
	t.Cleanup(func() { fx.cancel(nil) })
	fx.clock = &mwClock{t: t, now: time.Date(2026, 10, 14, 9, 0, 30, 0, time.UTC), stop: fx.cancel, onSleep: func(n int) {
		lock, err := acquireHeldFileLock(fx.mo.stateDir, regateLockFile, "test", 0)
		if err != nil {
			t.Errorf("regate.lock while Merge waits for the queue: %v; want it free", err)
		} else {
			fx.lockTaken++
			_ = lock.release()
		}
		if n <= len(fx.reads) {
			fx.q.read = fx.reads[n-1]
		}
		if fx.onSleep != nil {
			fx.onSleep(n)
		}
	}}
	fx.p = fx.params()
	fx.p.Exec = fx.q.run
	fx.p.WaitChecks, fx.p.Progress = wait, fx.progress
	fx.p.waitTest = mergeWaitTest{now: fx.clock.Now, sleep: fx.clock.sleep}
	return fx
}

// run runs Merge with fx.p and fx.ctx.
func (fx *mqwFixture) run(t *testing.T) (MergeResult, error) {
	t.Helper()
	return fx.merge(fx.ctx, t, fx.p)
}

// withFakeWorktrees sets fakeWorktrees with the record of org-a's feature
// worktree on feat/a, as mcFakeFixture does, and RepoRoot root, so a
// recorded merge runs the cleanup.
func (fx *mqwFixture) withFakeWorktrees(root string) *fakeWorktrees {
	wt := newFakeWorktrees()
	wt.records[featureWorktreeID("org-a")] = WorktreeRecord{WorktreePath: "/repo/.claude/worktrees/org-a", Branch: "feat/a",
		Kind: featureWorktreeKind}
	fx.mo.o.Worktrees = wt
	fx.p.RepoRoot = root
	return wt
}

// lines are the lines Merge wrote to Progress.
func (fx *mqwFixture) lines() []string {
	var out []string
	for line := range strings.Lines(fx.progress.String()) {
		out = append(out, strings.TrimSuffix(line, "\n"))
	}
	return out
}

// check fails the test unless the commands, the sleeps, and the lines are
// keys, sleeps, and lines, regate.lock was free in every sleep, and it is
// free now.
func (fx *mqwFixture) check(t *testing.T, keys []string, sleeps []time.Duration, lines []string) {
	t.Helper()
	if got := fx.exec.keys(); !slices.Equal(got, keys) {
		t.Errorf("commands = %q\nwant %q", got, keys)
	}
	if !slices.Equal(fx.clock.sleeps, sleeps) {
		t.Errorf("sleeps = %v, want %v", fx.clock.sleeps, sleeps)
	}
	if fx.lockTaken != len(fx.clock.sleeps) {
		t.Errorf("regate.lock was taken in %d of the %d sleeps; want every one", fx.lockTaken, len(fx.clock.sleeps))
	}
	if got := fx.lines(); !slices.Equal(got, lines) {
		t.Errorf("Progress =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(lines, "\n"))
	}
	mrAssertLockFree(t, fx.mo.stateDir)
}

// mqwLockSpy is fakeWorktrees that notes, in Lookup, the cleanup's first call
// to it, whether regate.lock is held and which org's marker it holds.
type mqwLockSpy struct {
	*fakeWorktrees
	stateDir string
	looked   bool
	held     bool
	marker   string
}

func (s *mqwLockSpy) Lookup(repoRoot, id string) (WorktreeRecord, bool, error) {
	if !s.looked {
		s.looked = true
		s.held, _ = fileLockHeld(regateLockPathIn(s.stateDir))
		s.marker, _ = readRegateMarker(regateLockPathIn(s.stateDir))
	}
	return s.fakeWorktrees.Lookup(repoRoot, id)
}

// TestMerge_QueueWaitRecordsTheMergeOnceTheQueueMergedIt covers AC1, FR-6 to
// FR-8, D1, and D2: with --wait-checks 30m, Merge enqueues the PR, releases
// regate.lock, and reads only the PR's merge queue state every 30 s, through
// gh api with the gh timeout and no check of checkMerge again; once a read
// finds the PR MERGED at the gate's head into main, it appends pr_merged with
// via=merge_queue and merge_commit the read's mergeCommit.oid, once, then
// cleans up the feature worktree with regate.lock taken again and org-a's
// marker in it, which records worktree_cleaned; the lock is free in every
// sleep and every read of the wait, and after Merge.
func TestMerge_QueueWaitRecordsTheMergeOnceTheQueueMergedIt(t *testing.T) {
	isolateGitEnv(t)
	gitRoot := resolvedOrClean(t.TempDir())
	runGit(t, "init", "--quiet", "-b", "main", gitRoot)
	fx := newMQWFixture(t, nil, 30*time.Minute, mqwQueuedAt(1), mqwQueuedAt(1), mqwMerged)
	spy := &mqwLockSpy{fakeWorktrees: fx.withFakeWorktrees(gitRoot), stateDir: fx.mo.stateDir}
	fx.mo.o.Worktrees = spy
	var reads, freeInReads int
	fx.q.inRead = func(context.Context) {
		if reads++; reads == 1 {
			return // the read before the enqueue, under the lock
		}
		if lock, err := acquireHeldFileLock(fx.mo.stateDir, regateLockFile, "test", 0); err == nil {
			freeInReads++
			_ = lock.release()
		}
	}
	start := time.Now()
	res, err := fx.run(t)
	end := time.Now()
	if err != nil || res.Cleanup == nil {
		t.Fatalf("Merge = %+v, %v; want a recorded merge with its cleanup", res, err)
	}
	if c := *res.Cleanup; !c.Cleaned || !c.Recorded || c.Resumed != CleanupResumedRecordOnly || c.Refused != "" || c.Failed != "" ||
		res.CleanupErr != "" {
		t.Errorf("cleanup = %+v, %q; want it cleaned, resumed from the record alone", c, res.CleanupErr)
	}
	res.Cleanup = nil
	if want := mqwMergedResult(mqwOID); res != want {
		t.Fatalf("Merge = %+v\nwant %+v", res, want)
	}
	fx.check(t, mqwKeys(3), slices.Repeat([]time.Duration{30 * time.Second}, 3), []string{mqwStart("30m0s", "30m0s"),
		mqwEnd("1m30s", "it was merged into main at the head aaaaaaaaaaaa")})
	if freeInReads != 3 {
		t.Errorf("regate.lock was free in %d of the 3 reads of the wait; want every one", freeInReads)
	}
	if !spy.looked || !spy.held || spy.marker != "org-a" {
		t.Errorf("in the cleanup: looked %v, lock held %v, marker %q; want held with org-a's marker", spy.looked, spy.held, spy.marker)
	}
	want := mqwRecord(t, mqwOID, "") + mcCleanedRecord(t, "branch=feat/a head="+moHeadA+
		` via=merge worktree="/repo/.claude/worktrees/org-a"`)
	if got := fx.appended(t); got != want {
		t.Errorf("appended %q\nwant     %q", got, want)
	}
	for i, c := range fx.exec.snapshot()[len(mqrEnqueued):] {
		if c.key != mqrKeyRead || !slices.Equal(append([]string{c.name}, c.args...), mqrReadArgv) {
			t.Errorf("read %d of the wait = %s %q; want the read of the merge queue state", i+1, c.name, c.args)
		}
		if c.deadline.Before(start.Add(defaultMergeOrderGHTimeout)) || c.deadline.After(end.Add(defaultMergeOrderGHTimeout)) {
			t.Errorf("read %d of the wait has the deadline %v; want the gh timeout from the call", i+1, c.deadline)
		}
	}
}

// TestMerge_QueueWaitRecordDetails covers FR-8 for the Details and Merged of
// a merge through the queue: a read with no merge commit is recorded as
// merge_commit=unknown; an accepted branch ends with gate=accepted and then
// via=merge_queue; a PR found in the queue already (FR-5, AC5) is waited for
// from there, with no enqueuePullRequest, and keeps its QueueNote. The
// Details read back as FollowupRecordedMerge reads them.
func TestMerge_QueueWaitRecordDetails(t *testing.T) {
	inQueue := map[string]any{"isInMergeQueue": true, "mergeQueueEntry": mqEntry("QUEUED", 1, mqrEnqueuedAt),
		"timelineItems": mqNodes(mqAdded(mqrEnqueuedAt))}
	for _, tc := range []struct {
		name     string
		accepted bool
		first    mgReply // the read before the enqueue; mqrPR(nil) when empty
		merged   mgReply
		commit   string
		note     string
		keys     []string
	}{
		{"no merge commit", false, mgReply{}, mqwMergedAt(moHeadA, "main", nil), "", "", mqwKeys(1)},
		{"an accepted branch", true, mgReply{}, mqwMerged, mqwOID, "", mqwKeys(1)},
		{"in the queue already", false, mqrPR(inQueue), mqwMerged, mqwOID, "PR #7 was in the merge queue already at the head " +
			"the gate passed, so ralph org merge did not add it again", append(slices.Clone(mqrReadOnly), mqrKeyRead)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var lines []any
			var accepted GateAcceptance
			if tc.accepted {
				lines, accepted = mxAcceptedLedger(t)
			}
			fx := newMQWFixture(t, lines, 30*time.Minute, tc.merged)
			if tc.first.out != "" {
				fx.q.read = tc.first
			}
			res, err := fx.run(t)
			if err != nil || tc.accepted != (res.Accepted != nil) || tc.accepted && *res.Accepted != accepted {
				t.Fatalf("Merge = %+v, %v; want Accepted %t", res, err, tc.accepted)
			}
			res.Accepted = nil
			want := mqwMergedResult(tc.commit)
			want.QueueNote = tc.note
			if res != want {
				t.Fatalf("Merge = %+v\nwant %+v", res, want)
			}
			gate := ""
			if tc.accepted {
				gate = " gate=accepted"
			}
			commit := tc.commit
			if commit == "" {
				commit = "unknown"
			}
			record := mqwRecord(t, commit, gate)
			if got := fx.appended(t); got != record {
				t.Errorf("appended %q\nwant     %q", got, record)
			}
			fx.check(t, tc.keys, []time.Duration{30 * time.Second}, []string{mqwStart("30m0s", "30m0s"),
				mqwEnd("30s", "it was merged into main at the head aaaaaaaaaaaa")})
			events := mustReadEvents(t, fx.mo.o)
			details := events[len(events)-1].Details
			if branch, ok := recordedMergeBranch(details); !ok || branch != "feat/a" {
				t.Errorf("recordedMergeBranch(%q) = %q, %v; want feat/a", details, branch, ok)
			}
			if head, ok := recordedMergeHead(details); !ok || head != moHeadA {
				t.Errorf("recordedMergeHead(%q) = %q, %v; want %s", details, head, ok, moHeadA)
			}
		})
	}
}

// TestMerge_QueueWaitRecordsNothingWhenThePRLeavesTheQueue covers AC2 and
// FR-9: a read that finds the PR out of the queue and not MERGED ends the
// wait with Failed and nothing recorded or cleaned up: with GitHub's latest
// removal strictly after the entry's enqueuedAt, its reason and actor,
// escaped, and the next step FR-5b leaves; with none after it (one before
// it, or one at that very time), that none was read.
func TestMerge_QueueWaitRecordsNothingWhenThePRLeavesTheQueue(t *testing.T) {
	const left = "org: merge: PR #7 (https://github.com/o/r/pull/7) left the merge queue of main without being merged and is "
	const since = " after it entered the queue at " + mqrEnqueuedAt
	const requeue = ", so nothing was recorded: fix what made the queue remove it, then get a new gate record of the branch " +
		"(ralph org regate --branch feat/a, or a gate of a new push), or run ralph org merge --branch feat/a --requeue to add it " +
		"again with the same gate on purpose"
	const none = ", and no removal from the queue" + since + " was read, so nothing was recorded: check the PR (" + mrViewCmd +
		" --json state,headRefOid,baseRefName,mergeCommit) and run ralph org merge-order to see the branch's state"
	for _, tc := range []struct {
		name  string
		read  mgReply
		state string
		want  string
	}{
		{"removed for failed checks", mqwOut("OPEN", mqAdded(mqrEnqueuedAt), mqRemoved(mqwRemovedAt, "failed_checks",
			"github-merge-queue")), "OPEN", left + "OPEN now: it was removed from the merge queue at " + mqwRemovedAt +
			" by github-merge-queue (reason: failed_checks)" + since + requeue},
		{"the latest of two removals after it", mqwOut("OPEN", mqAdded(mqrEnqueuedAt), mqRemoved("2026-10-14T09:02:00Z", "manual",
			"bob"), mqRemoved(mqwRemovedAt, "failed_checks", "alice")), "OPEN", left + "OPEN now: it was removed from the merge " +
			"queue at 2026-10-14T09:02:00Z by bob (reason: manual)" + since + requeue},
		{"closed, with a reason that holds a control character", mqwOut("CLOSED", mqAdded(mqrEnqueuedAt), mqRemoved(mqwRemovedAt,
			"manual\x1b[31m", nil)), "CLOSED", left + "CLOSED now: it was removed from the merge queue at " + mqwRemovedAt +
			` by an actor GitHub does not name (reason: manual\x1b[31m)` + since + requeue},
		{"a removal before it entered the queue", mqwOut("OPEN", mqRemoved("2026-10-14T08:10:00Z", "failed_checks", "alice"),
			mqAdded(mqrEnqueuedAt)), "OPEN", left + "OPEN now" + none},
		{"a removal at the time it entered the queue", mqwOut("OPEN", mqAdded(mqrEnqueuedAt), mqRemoved(mqrEnqueuedAt, "manual",
			"bob")), "OPEN", left + "OPEN now" + none},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMQWFixture(t, nil, 30*time.Minute, mqwQueuedAt(1), tc.read)
			wt := fx.withFakeWorktrees("/repo")
			res, err := fx.run(t)
			want := mqrResult()
			want.Failed = tc.want
			if err != nil || res != want {
				t.Fatalf("Merge = %+v, %v\nwant %+v", res, err, want)
			}
			mrNoRawControl(t, res.Failed)
			fx.check(t, mqwKeys(2), []time.Duration{30 * time.Second, 30 * time.Second}, []string{mqwStart("30m0s", "30m0s"),
				mqwEnd("1m0s", "it left the merge queue without being merged (it is "+tc.state+")")})
			if got := fx.appended(t); got != "" {
				t.Errorf("appended %q, want nothing", got)
			}
			if len(wt.calls) != 0 {
				t.Errorf("worktree calls = %q, want none", wt.calls)
			}
		})
	}
}

// TestMerge_QueueWaitLeavesThePRQueuedWhenTheTimeRunsOut covers AC4 for a
// budget that runs out and FR-10: reads that keep finding the PR in the
// queue run until --wait-checks is used up, the last sleep cut to what is
// left; Merge ends Queued with the last read's entry and how long it waited,
// records nothing, cleans up nothing, and makes no call that would remove the
// PR from the queue.
func TestMerge_QueueWaitLeavesThePRQueuedWhenTheTimeRunsOut(t *testing.T) {
	const still = "org: merge: PR #7 (https://github.com/o/r/pull/7) is still in the merge queue of main and not merged yet, and "
	const left = ", so nothing was recorded and ralph org merge left it in the queue: " + mqrNext
	noEntry := mqrPR(map[string]any{"isInMergeQueue": true})
	for _, tc := range []struct {
		name     string
		budget   time.Duration
		reads    []mgReply
		sleeps   []time.Duration
		where    string
		position int
		at       string
	}{
		{"2m, the position moving", 2 * time.Minute, []mgReply{mqwQueuedAt(3), mqwQueuedAt(2)},
			slices.Repeat([]time.Duration{30 * time.Second}, 4), "it is at position 2, enqueued at " + mqrEnqueuedAt, 2, mqrEnqueuedAt},
		{"45s, the last sleep cut", 45 * time.Second, []mgReply{mqwQueuedAt(1)}, []time.Duration{30 * time.Second, 15 * time.Second},
			"it is at position 1, enqueued at " + mqrEnqueuedAt, 1, mqrEnqueuedAt},
		{"the last read with no entry", time.Minute, []mgReply{mqwQueuedAt(1), noEntry},
			[]time.Duration{30 * time.Second, 30 * time.Second}, "GitHub gave no entry of it, so its position is unknown", 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMQWFixture(t, nil, tc.budget, tc.reads...)
			wt := fx.withFakeWorktrees("/repo")
			res, err := fx.run(t)
			budget := tc.budget.String()
			want := mqrResult()
			want.Queued = still + tc.where + left + "; ralph org merge waited " + budget + " for the merge queue (--wait-checks " +
				budget + ")"
			want.QueuePosition, want.EnqueuedAt = tc.position, tc.at
			if err != nil || res != want {
				t.Fatalf("Merge = %+v, %v\nwant %+v", res, err, want)
			}
			fx.check(t, mqwKeys(len(tc.sleeps)), tc.sleeps, []string{mqwStart(budget, budget),
				mqwEnd(budget, "--wait-checks "+budget+" is used up; it is still in the merge queue")})
			if got := fx.appended(t); got != "" {
				t.Errorf("appended %q, want nothing", got)
			}
			if len(wt.calls) != 0 {
				t.Errorf("worktree calls = %q, want none", wt.calls)
			}
		})
	}
}

// TestMerge_QueueWaitSharesTheBudgetWithTheChecksWait covers FR-7: one
// --wait-checks bounds the wait for pending checks and the wait for the
// queue after it, so the queue's wait gets what is left (1m30s of 2m), and
// the Queued once it is used up says Merge waited for both.
func TestMerge_QueueWaitSharesTheBudgetWithTheChecksWait(t *testing.T) {
	fx := newMQWFixture(t, nil, 2*time.Minute, mqrPR(nil), mqwQueuedAt(1))
	fx.exec.replies[mxPRChecks] = mwPending
	fx.onSleep = func(n int) {
		if n == 1 {
			fx.exec.mu.Lock()
			fx.exec.replies[mxPRChecks] = mwPass
			fx.exec.mu.Unlock()
		}
	}
	res, err := fx.run(t)
	want := mqrResult()
	want.Queued = "org: merge: PR #7 (https://github.com/o/r/pull/7) is still in the merge queue of main and not merged yet, and " +
		"it is at position 1, enqueued at " + mqrEnqueuedAt + ", so nothing was recorded and ralph org merge left it in the queue: " +
		mqrNext + "; ralph org merge waited 2m0s for the checks and for the merge queue (--wait-checks 2m0s)"
	want.QueuePosition, want.EnqueuedAt = 1, mqrEnqueuedAt
	if err != nil || res != want {
		t.Fatalf("Merge = %+v, %v\nwant %+v", res, err, want)
	}
	keys := slices.Concat(mxQueuePath[:5], []string{mxPRChecks}, mqwKeys(3))
	fx.check(t, keys, slices.Repeat([]time.Duration{30 * time.Second}, 4), []string{mwStart("has checks pending", "2m0s", "2m0s"),
		mwEnd("30s", "the checks passed"), mqwStart("1m30s", "2m0s"),
		mqwEnd("1m30s", "--wait-checks 2m0s is used up; it is still in the merge queue")})
}

// TestMerge_QueueWaitRecordsNothingForAMergeOfAnotherHeadOrBase covers AC8
// and NFR-3: a read that finds the PR MERGED at a head that is not the
// gate's, or into a base that is not origin's default branch, ends the wait
// with Failed, which names what was merged, and nothing recorded or cleaned
// up.
func TestMerge_QueueWaitRecordsNothingForAMergeOfAnotherHeadOrBase(t *testing.T) {
	const check = ", and nothing was recorded: check that merge yourself (" + mrViewCmd + " --json state,headRefOid,baseRefName," +
		"mergeCommit), run ralph org merge-order again"
	for _, tc := range []struct {
		name  string
		read  mgReply
		ended string
		want  string
	}{
		{"another head", mqwMergedAt(moHeadB, "main", map[string]any{"oid": mqwOID}), "it was merged into main at the head " +
			"bbbbbbbbbbbb", "org: merge: the merge queue merged PR #7 at the head bbbbbbbbbbbb, not aaaaaaaaaaaa, the head the gate " +
			"passed, so a head the gate did not pass was merged" + check + ", and go on with the other branches"},
		{"another base", mqwMergedAt(moHeadA, "dev", map[string]any{"oid": mqwOID}), "it was merged into dev at the head " +
			"aaaaaaaaaaaa", "org: merge: the merge queue merged PR #7 into the base dev, not main, the default branch the checks " +
			"read, so main got nothing from it as far as ralph knows" + check + " (it lists the branch as merged, though main got " +
			"nothing from this merge), and go on with the other branches"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMQWFixture(t, nil, 30*time.Minute, tc.read)
			wt := fx.withFakeWorktrees("/repo")
			res, err := fx.run(t)
			want := mqrResult()
			want.Failed = tc.want
			if err != nil || res != want {
				t.Fatalf("Merge = %+v, %v\nwant %+v", res, err, want)
			}
			fx.check(t, mqwKeys(1), []time.Duration{30 * time.Second}, []string{mqwStart("30m0s", "30m0s"), mqwEnd("30s", tc.ended)})
			if got := fx.appended(t); got != "" {
				t.Errorf("appended %q, want nothing", got)
			}
			if len(wt.calls) != 0 {
				t.Errorf("worktree calls = %q, want none", wt.calls)
			}
		})
	}
}

// TestMerge_QueueWaitReadsAgainAfterAFailedRead covers AC12 and FR-11: a
// read of the wait that fails (gh fails, times out, or answers what cannot be
// read) lets the wait go on; when the last read before the time is used up
// failed, Merge ends Failed, the PR maybe still in the queue, with the read's
// error, the next step a queued outcome gives, and how long it waited, and
// records nothing; when a read after a failed one finds the PR in the queue,
// it ends Queued.
func TestMerge_QueueWaitReadsAgainAfterAFailedRead(t *testing.T) {
	fails := mgReply{err: errors.New("exit status 1: HTTP 502")}
	const unknown = " at the last read of the wait, so whether PR #7 is still in the merge queue is unknown and nothing was " +
		"recorded: it may still be in the merge queue, so " + mqrNext + "; ralph org merge waited 1m0s for the merge queue " +
		"(--wait-checks 1m0s)"
	queued := mqrResult()
	queued.Queued = "org: merge: PR #7 (https://github.com/o/r/pull/7) is still in the merge queue of main and not merged yet, " +
		"and it is at position 1, enqueued at " + mqrEnqueuedAt + ", so nothing was recorded and ralph org merge left it in the " +
		"queue: " + mqrNext + "; ralph org merge waited 1m0s for the merge queue (--wait-checks 1m0s)"
	queued.QueuePosition, queued.EnqueuedAt = 1, mqrEnqueuedAt
	for _, tc := range []struct {
		name    string
		reads   []mgReply
		timeout bool // the reads of the wait wait for their ctx, which the gh timeout of 50ms ends
		failed  string
		end     string
	}{
		{"every read fails", []mgReply{fails}, false, "org: merge: " + mqrReadCmd + " failed (exit status 1: HTTP 502)" + unknown,
			"its last read failed"},
		{"in the queue, then a read that fails", []mgReply{mqwQueuedAt(1), fails}, false, "org: merge: " + mqrReadCmd +
			" failed (exit status 1: HTTP 502)" + unknown, "its last read failed"},
		{"every read times out", []mgReply{{}}, true, "org: merge: " + mqrReadCmd + " timed out after 50ms" + unknown,
			"its last read failed"},
		{"an answer with no data", []mgReply{{out: `{"data":null}`}}, false, "org: merge: " + mqrReadCmd + " returned no data" +
			unknown, "its last read failed"},
		{"a read that fails, then in the queue", []mgReply{fails, mqwQueuedAt(1)}, false, "", "it is still in the merge queue"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMQWFixture(t, nil, time.Minute, tc.reads...)
			if tc.timeout {
				fx.p.GHTimeout = 50 * time.Millisecond
				var reads int
				fx.q.inRead = func(ctx context.Context) {
					if reads++; reads > 1 {
						<-ctx.Done()
					}
				}
			}
			res, err := fx.run(t)
			want := queued
			if tc.failed != "" {
				want = mqrResult()
				want.Failed = tc.failed
			}
			if err != nil || res != want {
				t.Fatalf("Merge = %+v, %v\nwant %+v", res, err, want)
			}
			fx.check(t, mqwKeys(2), []time.Duration{30 * time.Second, 30 * time.Second}, []string{mqwStart("1m0s", "1m0s"),
				mqwEnd("1m0s", "--wait-checks 1m0s is used up; "+tc.end)})
			if got := fx.appended(t); got != "" {
				t.Errorf("appended %q, want nothing", got)
			}
		})
	}
}

// TestMergeThroughQueue_ReadsThroughRunDetached covers AC13 and NFR-4 for
// the wait: every read of the PR's merge queue state after the enqueue goes
// through runDetached, which newMergeCheck sets to realMergeDetachedExec for
// the real commands, and none through the runner of the checks.
func TestMergeThroughQueue_ReadsThroughRunDetached(t *testing.T) {
	fx := newMQWFixture(t, nil, 30*time.Minute)
	r, refused, _, err := fx.mo.o.checkMerge(context.Background(), fx.p)
	if err != nil || refused != "" || !r.queue {
		t.Fatalf("checkMerge = %+v, %v, refused %q; want the queue path", r, err, refused)
	}
	detached := &mqrExec{mrExec: &mrExec{mxExec: &mxExec{}}, read: mqwQueuedAt(1)}
	run := mergeRun{m: newMergeCheck(fx.p), r: r}
	run.m.runDetached = detached.run
	fx.onSleep = func(n int) {
		if n == 2 {
			detached.read = mqwMerged
		}
	}
	entry := &mergeQueueEntry{State: "QUEUED", Position: 1, EnqueuedAt: mqTime(t, mqrEnqueuedAt)}
	res, err := fx.mo.o.mergeThroughQueue(context.Background(), fx.p, newMergeWait(fx.p), mergeQueueWaitFor{run: run,
		res: mqrAdded(), entry: entry})
	if want := mqwMergedResult(mqwOID); err != nil || res != want {
		t.Fatalf("mergeThroughQueue = %+v, %v\nwant %+v", res, err, want)
	}
	if got := fx.exec.keys(); !slices.Equal(got, mxQueuePath) {
		t.Errorf("through the runner of the checks: %q; want the checks only", got)
	}
	if got := detached.keys(); !slices.Equal(got, []string{mqrKeyRead, mqrKeyRead}) {
		t.Errorf("through runDetached: %q; want the two reads of the wait", got)
	}
	mxAssertPinned(t, detached.mxExec, "github.com", mxRepoArg)
}

// TestMerge_QueueWaitRecordsAMergeOnlyOnce covers AC15 and D7: a pr_merged of
// the same branch, PR, and head after the branch's latest gate record, which
// another ralph org merge that waited for the same queue appended while this
// one waited, makes Merge append nothing and report the merge as recorded
// with a Note; a pr_merged before that gate record, of another head, PR, or
// branch, or a dry run, is not that record, and Merge appends its own.
func TestMerge_QueueWaitRecordsAMergeOnlyOnce(t *testing.T) {
	recordOf := func(branch string, pr, head string) ManifestEvent {
		return ManifestEvent{OrgID: "org-a", Event: EventPRMerged, Details: "branch=" + branch + " pr=" + pr + " head=" + head +
			" method=squash merge_commit=" + mqwOID + " auto_merge=false via=merge_queue"}
	}
	t.Run("another merge recorded it during the wait", func(t *testing.T) {
		fx := newMQWFixture(t, nil, 30*time.Minute, mqwQueuedAt(1), mqwMerged)
		var added string
		fx.onSleep = func(n int) {
			if n == 2 {
				before := mrRead(t, fx.mo.path)
				fx.mo.add(t, recordOf("feat/a", "7", moHeadA))
				added = strings.TrimPrefix(mrRead(t, fx.mo.path), before)
			}
		}
		res, err := fx.run(t)
		want := mqwMergedResult(mqwOID)
		want.Note = "the ledger held pr_merged of PR #7 at the head aaaaaaaaaaaa after the branch's latest gate record already, so " +
			"ralph org merge did not record it again: another ralph org merge that waited for the same merge queue recorded it"
		if err != nil || res != want {
			t.Fatalf("Merge = %+v, %v\nwant %+v", res, err, want)
		}
		if got := fx.appended(t); added == "" || got != added {
			t.Errorf("appended %q\nwant only the other merge's %q", got, added)
		}
		mrAssertLockFree(t, fx.mo.stateDir)
	})
	dryRun := recordOf("feat/a", "7", moHeadA)
	dryRun.DryRun = true
	for _, tc := range []struct {
		name  string
		lines []any
	}{
		{"before the latest gate record", []any{moBind("org-a", "a"), recordOf("feat/a", "7", moHeadA),
			moPassed("org-a", "feat/a", moHeadA, moMain), moDisbanded("org-a")}},
		{"of another head", append(mxReadyLedger(), recordOf("feat/a", "7", moHeadB))},
		{"of another PR", append(mxReadyLedger(), recordOf("feat/a", "8", moHeadA))},
		{"of another branch", append(mxReadyLedger(), recordOf("feat/b", "7", moHeadA))},
		{"a dry run", append(mxReadyLedger(), dryRun)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMQWFixture(t, tc.lines, 30*time.Minute, mqwMerged)
			res, err := fx.run(t)
			if want := mqwMergedResult(mqwOID); err != nil || res != want {
				t.Fatalf("Merge = %+v, %v\nwant %+v", res, err, want)
			}
			if got, want := fx.appended(t), mqwRecord(t, mqwOID, ""); got != want {
				t.Errorf("appended %q\nwant     %q", got, want)
			}
		})
	}
}

// TestMerge_QueueWaitCleanupWithTheLockHeldByAnother covers D1: when another
// ralph org regate, merge, or cleanup holds regate.lock once the merge is
// recorded, the cleanup does not run, and Cleanup holds only a Refused that
// names the lock and ralph org cleanup of the branch; the merge stays
// recorded and Merge returns no error.
func TestMerge_QueueWaitCleanupWithTheLockHeldByAnother(t *testing.T) {
	fx := newMQWFixture(t, nil, 30*time.Minute, mqwQueuedAt(1), mqwMerged)
	wt := fx.withFakeWorktrees("/repo")
	var other *heldFileLock
	fx.onSleep = func(n int) {
		if n == 2 {
			var err error
			if other, err = acquireHeldFileLock(fx.mo.stateDir, regateLockFile, "regate", 0); err != nil {
				t.Fatal(err)
			}
		}
	}
	begin := time.Now()
	res, err := fx.run(t)
	took := time.Since(begin)
	if other == nil {
		t.Fatal("the other holder never took regate.lock")
	}
	if err := other.release(); err != nil {
		t.Fatal(err)
	}
	if took < regateLockWait {
		t.Errorf("Merge skipped the cleanup after %s; want it to retry the lock for %s", took, regateLockWait)
	}
	cleanup := CleanupResult{Branch: "feat/a", Head: moHeadA, Refused: "another ralph org regate, merge, or cleanup runs on this " +
		"ledger (" + regateLockPathIn(fx.mo.stateDir) + " is held), so the feature worktree of the branch feat/a was not cleaned up " +
		"once pr_merged was recorded: once it ends, run ralph org cleanup --branch feat/a"}
	if err != nil || res.Cleanup == nil || *res.Cleanup != cleanup || res.CleanupErr != "" {
		t.Fatalf("Merge = %+v, %v (cleanup %+v)\nwant the cleanup %+v", res, err, res.Cleanup, cleanup)
	}
	res.Cleanup = nil
	if want := mqwMergedResult(mqwOID); res != want {
		t.Fatalf("Merge = %+v\nwant %+v", res, want)
	}
	if got, want := fx.appended(t), mqwRecord(t, mqwOID, ""); got != want {
		t.Errorf("appended %q\nwant     %q", got, want)
	}
	if len(wt.calls) != 0 {
		t.Errorf("worktree calls = %q, want none", wt.calls)
	}
	mrAssertLockFree(t, fx.mo.stateDir)
}

// TestMerge_QueueWaitCleanupEmptiesTheMarkerAndKeepsTheMerge covers D1 and
// D2: an earlier regate's marker left in regate.lock while Merge waited is
// emptied when the cleanup takes the lock again, as mergeLocked empties it,
// so it is gone after Merge though the cleanup writes none of its own here;
// the checks do not run again after the wait, so org-a spawned again while
// Merge waited does not undo the recorded merge, and the cleanup's own first
// check refuses that org's worktree, which keeps the merge and Merge's
// outcome.
func TestMerge_QueueWaitCleanupEmptiesTheMarkerAndKeepsTheMerge(t *testing.T) {
	fx := newMQWFixture(t, nil, 30*time.Minute, mqwQueuedAt(1), mqwMerged)
	wt := fx.withFakeWorktrees("/repo")
	path := regateLockPathIn(fx.mo.stateDir)
	var seeded string
	fx.onSleep = func(n int) {
		if n == 2 {
			writeGateFile(t, fx.mo.stateDir, regateLockFile, `{"org_id":"org-x","branch":"feat/x","worktree":"/w","pid":1}`+"\n",
				0o644)
			before := mrRead(t, fx.mo.path)
			seedSeat(t, fx.mo.o, "org-a", "worker", "worker", "pane-2")
			seeded = strings.TrimPrefix(mrRead(t, fx.mo.path), before)
		}
	}
	res, err := fx.run(t)
	cleanup := CleanupResult{Branch: "feat/a", Head: moHeadA, Refused: `the branch feat/a is bound to org_id "org-a", which is ` +
		"running and may still use its feature worktree /repo/.claude/worktrees/org-a: wait for the org to be disbanded and run " +
		"ralph org cleanup --branch feat/a again"}
	if err != nil || res.Cleanup == nil || *res.Cleanup != cleanup || res.CleanupErr != "" {
		t.Fatalf("Merge = %+v, %v (cleanup %+v)\nwant the cleanup %+v", res, err, res.Cleanup, cleanup)
	}
	res.Cleanup = nil
	if want := mqwMergedResult(mqwOID); res != want {
		t.Fatalf("Merge = %+v\nwant %+v", res, want)
	}
	if got := fileOrAbsent(t, path); got != "" {
		t.Errorf("regate.lock holds %q after the merge; want the marker emptied", got)
	}
	if got, want := fx.appended(t), seeded+mqwRecord(t, mqwOID, ""); seeded == "" || got != want {
		t.Errorf("appended %q\nwant     %q", got, want)
	}
	if len(wt.calls) != 0 {
		t.Errorf("worktree calls = %q, want none", wt.calls)
	}
	mrAssertLockFree(t, fx.mo.stateDir)
}

// TestMerge_QueueWaitInterrupted covers the interruptions of the wait for the
// queue (D6): in a sleep or a read, an error that says nothing was recorded
// and the PR may still be in the queue, with the next step a queued outcome
// gives, and the Queued result of the enqueue; once a read found the PR
// merged, the merge is recorded, the cleanup is skipped with a Refused that
// says so, and the error says how the merge ended; once a read found it out
// of the queue, the error carries the Failed. regate.lock is free after each.
func TestMerge_QueueWaitInterrupted(t *testing.T) {
	const during = "org: merge: interrupted (test stopped) while it waited for PR #7 in the merge queue: nothing was recorded, " +
		"and PR #7 may still be in the merge queue: " + mqrNext
	const then = "org: merge: interrupted (test stopped) while it waited for the merge queue, and then: "
	removed := mqwOut("OPEN", mqAdded(mqrEnqueuedAt), mqRemoved(mqwRemovedAt, "failed_checks", "alice"))
	removedFailed := "org: merge: PR #7 (https://github.com/o/r/pull/7) left the merge queue of main without being merged and is " +
		"OPEN now: it was removed from the merge queue at " + mqwRemovedAt + " by alice (reason: failed_checks) after it entered " +
		"the queue at " + mqrEnqueuedAt + ", so nothing was recorded: fix what made the queue remove it, then get a new gate " +
		"record of the branch (ralph org regate --branch feat/a, or a gate of a new push), or run ralph org merge --branch " +
		"feat/a --requeue to add it again with the same gate on purpose"
	merged := mqwMergedResult(mqwOID)
	merged.Cleanup = &CleanupResult{Branch: "feat/a", Head: moHeadA, Refused: "ralph org merge was interrupted (test stopped) " +
		"once pr_merged was recorded, so the feature worktree of the branch feat/a was not cleaned up: run ralph org cleanup " +
		"--branch feat/a"}
	failed := mqrResult()
	failed.Failed = removedFailed
	for _, tc := range []struct {
		name     string
		setup    func(fx *mqwFixture, stop func())
		res      MergeResult
		err      string
		reads    int
		sleeps   int
		end      string
		appended func(t *testing.T) string
	}{
		{"in a sleep", func(fx *mqwFixture, stop func()) {
			fx.onSleep = func(n int) {
				if n == 2 {
					stop()
				}
			}
		}, mqrAdded(), during, 1, 2, "after 1m0s: interrupted (test stopped); nothing was recorded", nil},
		{"in a read", func(fx *mqwFixture, stop func()) {
			var reads int
			fx.q.inRead = func(context.Context) {
				if reads++; reads == 2 {
					stop()
				}
			}
		}, mqrAdded(), during, 1, 1, "after 30s: interrupted (test stopped); nothing was recorded", nil},
		{"once a read found it merged", func(fx *mqwFixture, stop func()) {
			fx.reads = []mgReply{mqwMerged}
			fx.q.answered = mqwOnRead(2, stop)
		}, merged, then + "PR #7 (https://github.com/o/r/pull/7) was merged (merge commit " + mqwOID + ") and pr_merged was " +
			"recorded", 1, 1, "after 30s: it was merged into main at the head aaaaaaaaaaaa",
			func(t *testing.T) string { return mqwRecord(t, mqwOID, "") }},
		{"once a read found it out of the queue", func(fx *mqwFixture, stop func()) {
			fx.reads = []mgReply{removed}
			fx.q.answered = mqwOnRead(2, stop)
		}, failed, then + strings.TrimPrefix(removedFailed, "org: merge: "), 1, 1, "after 30s: it left the merge queue without " +
			"being merged (it is OPEN)", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMQWFixture(t, nil, 30*time.Minute, mqwQueuedAt(1))
			wt := fx.withFakeWorktrees("/repo")
			tc.setup(fx, func() { fx.cancel(errors.New("test stopped")) })
			res, err := fx.run(t)
			if err == nil || err.Error() != tc.err {
				t.Fatalf("Merge error = %v\nwant %q", err, tc.err)
			}
			if tc.res.Cleanup != nil && (res.Cleanup == nil || *res.Cleanup != *tc.res.Cleanup) {
				t.Errorf("cleanup = %+v, want %+v", res.Cleanup, tc.res.Cleanup)
			}
			want := tc.res
			res.Cleanup, want.Cleanup = nil, nil
			if res != want {
				t.Fatalf("Merge = %+v\nwant %+v", res, want)
			}
			fx.check(t, mqwKeys(tc.reads), slices.Repeat([]time.Duration{30 * time.Second}, tc.sleeps),
				[]string{mqwStart("30m0s", "30m0s"), "org: merge: stopped waiting for PR #7 in the merge queue " + tc.end})
			var appended string
			if tc.appended != nil {
				appended = tc.appended(t)
			}
			if got := fx.appended(t); got != appended {
				t.Errorf("appended %q\nwant     %q", got, appended)
			}
			if len(wt.calls) != 0 {
				t.Errorf("worktree calls = %q, want none", wt.calls)
			}
		})
	}
}

// mqwOnRead is an mqrExec.answered that runs f once the n-th read of the
// PR's merge queue state (from 1, the read before the enqueue) has its
// answer.
func mqwOnRead(n int, f func()) func(key string) {
	var reads int
	return func(key string) {
		if key != mqrKeyRead {
			return
		}
		if reads++; reads == n {
			f()
		}
	}
}
