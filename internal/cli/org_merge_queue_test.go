package cli

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for the merge queue path of `ralph org merge` at the command line
// (plan docs/plans/active/2026-11-20-org-merge-queue-verb.md, S3a and S4, D6,
// AC4 and AC14; spec docs/specs/2026-11-19-org-merge-queue.md, FR-5b and
// FR-10): what printMergeResult prints for a result that leaves the PR in the
// merge queue and for a merge the queue made, exit code 7 (ErrMergeQueued),
// --requeue through the command with the fake gh of org_merge_test.go, and
// the help. The org layer's queue path is covered by
// internal/org/merge_queue_run_test.go and merge_queue_wait_test.go.

// mergeCLIQueueFakeGH is mergeCLIFakeGH that also tells the two GraphQL
// calls of the merge queue path from the mergeQueue read of the protection
// check, by their query: the read of the PR's merge queue state (api-queue-pr)
// and enqueuePullRequest (api-enqueue). Every other call is answered as
// mergeCLIFakeGH answers it.
var mergeCLIQueueFakeGH = strings.Replace(mergeCLIFakeGH, `"api "*" graphql") reply=api-graphql ;;`,
	`"api "*"enqueuePullRequest"*) reply=api-enqueue ;;`+"\n"+`"api "*"isInMergeQueue"*) reply=api-queue-pr ;;`+"\n"+
		`"api "*" graphql") reply=api-graphql ;;`, 1)

const (
	// mergeCLIQueueNodeID is PR #7's node ID in the fake gh's answers.
	mergeCLIQueueNodeID = "PR_kwDOAbc123"
	// mergeCLIQueueEnqueuedAt is when enqueuePullRequest's answer says PR #7
	// entered the queue.
	mergeCLIQueueEnqueuedAt = "2026-11-20T10:00:00Z"
	// mergeCLIQueueProtection is CheckMergeEnforcement's Detail for the fake
	// gh's merge queue and ruleset, which the PROTECTION: line prints.
	mergeCLIQueueProtection = "main, the default branch of origin's repository o/r, has a merge queue (GraphQL mergeQueue is set), " +
		"and rulesets require 1 required status check: the queue merges the PR only once its required checks pass on it with the " +
		"latest main"
)

// newMergeCLIQueueFixture is newMergeCLIFixture on a default branch with a
// merge queue whose configuration merges with SQUASH and whose ruleset
// requires the check verify: the fake gh (mergeCLIQueueFakeGH) reads PR #7
// OPEN at the gate's head and out of the queue, with removedAt, when set, as
// the time of a RemovedFromMergeQueueEvent (reason failed_checks, by octocat)
// in its timeline, and answers enqueuePullRequest with the PR at position 2.
func newMergeCLIQueueFixture(t *testing.T, removedAt string) *mergeCLIFixture {
	t.Helper()
	if !strings.Contains(mergeCLIQueueFakeGH, "reply=api-enqueue") {
		t.Fatal("mergeCLIQueueFakeGH did not get its GraphQL cases: mergeCLIFakeGH's graphql case changed")
	}
	fx := newMergeCLIFixture(t, false)
	writeStubScript(t, filepath.Join(fx.bin, "gh"), mergeCLIQueueFakeGH)
	fx.reply(t, "api-graphql", `{"data":{"repository":{"mergeQueue":{"id":"MQ_kwDOA","configuration":{"mergeMethod":"SQUASH"}}}}}`)
	nodes := ""
	if removedAt != "" {
		nodes = `{"__typename":"RemovedFromMergeQueueEvent","createdAt":"` + removedAt + `","reason":"failed_checks",` +
			`"actor":{"login":"octocat"}}`
	}
	fx.reply(t, "api-queue-pr", `{"data":{"repository":{"pullRequest":{"id":"`+mergeCLIQueueNodeID+`","state":"OPEN",`+
		`"headRefOid":"`+fx.head+`","baseRefName":"main","mergeCommit":null,"isInMergeQueue":false,"mergeQueueEntry":null,`+
		`"timelineItems":{"nodes":[`+nodes+`]}}}}}`)
	fx.reply(t, "api-enqueue", `{"data":{"enqueuePullRequest":{"mergeQueueEntry":{"state":"QUEUED","position":2,"enqueuedAt":"`+
		mergeCLIQueueEnqueuedAt+`"}}}}`)
	return fx
}

// enqueueCalls are the fake gh's enqueuePullRequest calls.
func (fx *mergeCLIFixture) enqueueCalls(t *testing.T) []string {
	t.Helper()
	var calls []string
	for _, c := range fx.ghCalls(t) {
		if strings.Contains(c, "enqueuePullRequest") {
			calls = append(calls, c)
		}
	}
	return calls
}

// gatePassedTS is the ts of the fixture ledger's gate_passed, which the texts
// of FR-5b name.
func (fx *mergeCLIFixture) gatePassedTS(t *testing.T) string {
	t.Helper()
	for _, ev := range readManifestEvents(t, org.ManifestPathIn(fx.stateDir)) {
		if ev.Event == org.EventGatePassed {
			return ev.TS
		}
	}
	t.Fatal("the fixture's ledger holds no gate_passed")
	return ""
}

// printMergeResult of a result that left the PR in the merge queue, not
// merged and nothing recorded (MergeResult.Queued, D6): stdout gets the
// QUEUED: block, with POSITION: and ENQUEUED AT: only when GitHub gave the
// PR's entry, GATE: for an accepted branch, and NOTE: only with a QueueNote,
// every value from GitHub escaped; the error is the Queued text itself, wraps
// ErrMergeQueued, and exits 7. With an error from Merge (an interrupt once
// enqueuePullRequest was sent), it prints nothing, returns that error, and
// exits 1.
func TestOrgMerge_PrintsAQueuedResultAndExitsSeven(t *testing.T) {
	queued := org.MergeResult{
		Queued: "org: merge: PR #7 (https://github.com/o/r/pull/7) was added to the merge queue of main at the head " +
			"aaaaaaaaaaaa, the head the gate passed, and it is at position 1, enqueued at 2026-10-14T09:00:30Z; it is not merged " +
			"yet, so nothing was recorded: run ralph org merge --branch feat/a again",
		QueuePosition: 1, EnqueuedAt: "2026-10-14T09:00:30Z", Branch: "feat/a", OrgID: "a", PRNumber: 7,
		PRURL: "https://github.com/o/r/pull/7", Method: "squash", Head: strings.Repeat("a", 40), Protection: "a merge queue",
	}
	block := "QUEUED: https://github.com/o/r/pull/7 (PR #7 of feat/a)\nMETHOD: squash\nHEAD: " + strings.Repeat("a", 40) + "\n"
	noEntry := queued
	noEntry.QueuePosition, noEntry.EnqueuedAt, noEntry.Method = 0, "", "queue"
	requeued := queued
	requeued.QueueNote = "PR #7 was removed from the merge queue at 2026-10-14T09:00:00Z by octocat (reason: failed_checks), " +
		"after the branch's latest gate record, and --requeue added it to the merge queue again"
	requeued.Accepted = &org.GateAcceptance{FailedReason: "test_failed", By: "alice", TS: "2026-11-12T10:00:00Z", Note: "outside"}
	escaped := queued
	escaped.Protection, escaped.QueueNote = "a merge queue \x1b[31m", "already \x07there"
	interrupted := errors.New("org: merge: interrupted (SIGINT) once enqueuePullRequest had been sent, and then: PR #7 was added")

	for _, tc := range []struct {
		name       string
		res        org.MergeResult
		err        error
		wantStdout string
		wantExit   int
	}{
		{"with its entry", queued, nil, block + "POSITION: 1\nENQUEUED AT: 2026-10-14T09:00:30Z\nPROTECTION: a merge queue\n", 7},
		{"with no entry", noEntry, nil, strings.Replace(block, "METHOD: squash", "METHOD: queue", 1) +
			"PROTECTION: a merge queue\n", 7},
		{"requeued, an accepted branch", requeued, nil, block + "POSITION: 1\nENQUEUED AT: 2026-10-14T09:00:30Z\n" +
			"PROTECTION: a merge queue\nGATE: accepted (test_failed) by alice at 2026-11-12T10:00:00Z: outside\nNOTE: " +
			requeued.QueueNote + "\n", 7},
		{"control characters are escaped", escaped, nil, block + "POSITION: 1\nENQUEUED AT: 2026-10-14T09:00:30Z\n" +
			`PROTECTION: a merge queue \x1b[31m` + "\n" + `NOTE: already \athere` + "\n", 7},
		{"interrupted", queued, interrupted, "", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			err := printMergeResult(&out, tc.res, tc.err, "")
			if out.String() != tc.wantStdout {
				t.Errorf("stdout:\n%q\nwant:\n%q", out.String(), tc.wantStdout)
			}
			if got := ExitCode(err); got != tc.wantExit {
				t.Errorf("ExitCode(%v) = %d; want %d", err, got, tc.wantExit)
			}
			switch {
			case tc.err != nil && err != tc.err:
				t.Errorf("err = %v; want Merge's error itself", err)
			case tc.err != nil && errors.Is(err, ErrMergeQueued):
				t.Errorf("err = %v wraps ErrMergeQueued; want only Merge's error", err)
			case tc.err == nil && (err == nil || err.Error() != tc.res.Queued || !errors.Is(err, ErrMergeQueued)):
				t.Errorf("err = %v; want the Queued text %q, wrapping ErrMergeQueued", err, tc.res.Queued)
			}
		})
	}
}

// printMergeResult of a merge the merge queue made (FR-8): today's MERGED:
// block, with METHOD: the queue's method, or queue when GitHub gave none, a
// NOTE: line for the QueueNote (the PR was in the queue already, or --requeue
// added it again), escaped, and one after it when the ledger held the merge's
// pr_merged already (D7), and exit 0.
func TestOrgMerge_PrintsAMergeQueueMergeAsMerged(t *testing.T) {
	merged := org.MergeResult{
		Merged: true, Recorded: true, Branch: "feat/a", OrgID: "a", PRNumber: 7, PRURL: "https://github.com/o/r/pull/7",
		Method: "squash", Head: "aaaa", MergeCommit: "bbbb", AutoMerge: true, Protection: "a merge queue",
	}
	lines := func(method string) string {
		return "MERGED: https://github.com/o/r/pull/7 (PR #7 of feat/a)\nMETHOD: " + method + "\nHEAD: aaaa\nMERGE COMMIT: bbbb\n" +
			"AUTO MERGE: true\nPROTECTION: a merge queue\n"
	}
	noMethod := merged
	noMethod.Method = "queue"
	inQueue := merged
	inQueue.QueueNote = "PR #7 was in the merge queue already at the head the gate passed, so ralph org merge did not add it again"
	requeued := merged
	requeued.QueueNote = "PR #7 was removed from the merge queue at 2026-10-14T09:00:00Z by octocat (reason: failed_checks), " +
		"after the branch's latest gate record, and --requeue added it to the merge queue again"
	escaped := merged
	escaped.QueueNote = "PR #7 was removed by \x1b[31moctocat\x07"
	recordedAlready := inQueue
	recordedAlready.Note = "the ledger held pr_merged of PR #7 at the head aaaa already, so ralph org merge did not record it again: " +
		"another ralph org merge that waited for the same merge queue recorded it"
	for _, tc := range []struct {
		name       string
		res        org.MergeResult
		wantStdout string
	}{
		{"the queue's method", merged, lines("squash")},
		{"no method from GitHub", noMethod, lines("queue")},
		{"in the queue already", inQueue, lines("squash") + "NOTE: " + inQueue.QueueNote + "\n"},
		{"added again with --requeue", requeued, lines("squash") + "NOTE: " + requeued.QueueNote + "\n"},
		{"a QueueNote with control characters", escaped, lines("squash") + `NOTE: PR #7 was removed by \x1b[31moctocat\a` + "\n"},
		{"recorded already by another merge", recordedAlready, lines("squash") + "NOTE: " + inQueue.QueueNote + "\nNOTE: " +
			recordedAlready.Note + "\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			err := printMergeResult(&out, tc.res, nil, "")
			if out.String() != tc.wantStdout {
				t.Errorf("stdout:\n%q\nwant:\n%q", out.String(), tc.wantStdout)
			}
			if err != nil || ExitCode(err) != 0 {
				t.Errorf("err = %v (exit %d); want nil, exit 0", err, ExitCode(err))
			}
		})
	}
}

// --requeue through the command (FR-5b, AC10 at the command line): on a
// default branch with a merge queue, with the default --wait-checks 0, a PR
// out of the queue with no removal after the gate is added to the queue with
// one enqueuePullRequest at the gate's head and the QUEUED: block on stdout,
// nothing on stderr, nothing recorded, and exit 7; a PR GitHub removed from
// the queue after the branch's latest gate record is refused without
// --requeue (exit 1, nothing on stdout, no enqueuePullRequest), and with it
// is added as above, with NOTE: saying --requeue added it again. No gh pr
// merge and no gh repo view run on the queue path.
func TestOrgMerge_RequeueReachesTheMergeQueuePath(t *testing.T) {
	removedAt := time.Now().UTC().Add(time.Minute).Format(time.RFC3339)
	removal := "removed from the merge queue at " + removedAt + " by octocat (reason: failed_checks)"
	for _, tc := range []struct {
		name      string
		removedAt string
		requeue   bool
		refused   bool
	}{
		{"no removal after the gate", "", false, false},
		{"a removal after the gate, without --requeue", removedAt, false, true},
		{"a removal after the gate, with --requeue", removedAt, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMergeCLIQueueFixture(t, tc.removedAt)
			before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))
			gateWhen := "the branch's latest gate record (at " + fx.gatePassedTS(t) + "; a removal less than 5m0s before it counts as " +
				"after it)"
			var extra []string
			if tc.requeue {
				extra = append(extra, "--requeue")
			}

			stdout, stderr, err := fx.merge(t, extra...)
			if stderr != "" {
				t.Errorf("stderr = %q; want nothing (no wait with --wait-checks 0)", stderr)
			}
			if tc.refused {
				refusal := "org: merge: PR #7 was " + removal + ", after " + gateWhen + ", so ralph org merge did not add it to the " +
					"merge queue again and nothing was merged: "
				if err == nil || !strings.HasPrefix(err.Error(), refusal) || !strings.Contains(err.Error(), " --requeue to add it again") ||
					ExitCode(err) != 1 || errors.Is(err, ErrMergeQueued) || stdout != "" {
					t.Fatalf("merge: err %v (exit %d), stdout %q; want the refusal %q... naming --requeue, exit 1, nothing on stdout",
						err, ExitCode(err), stdout, refusal)
				}
				if got := fx.enqueueCalls(t); len(got) != 0 {
					t.Errorf("enqueuePullRequest ran: %q", got)
				}
			} else {
				queued := "org: merge: PR #7 (https://github.com/o/r/pull/7) was added to the merge queue of main at the head " +
					fx.head[:12] + ", the head the gate passed, and it is at position 2, enqueued at " + mergeCLIQueueEnqueuedAt
				if err == nil || !strings.HasPrefix(err.Error(), queued) || !errors.Is(err, ErrMergeQueued) || ExitCode(err) != 7 {
					t.Fatalf("merge: err %v (exit %d); want the Queued text %q..., wrapping ErrMergeQueued, exit 7", err, ExitCode(err),
						queued)
				}
				want := "QUEUED: https://github.com/o/r/pull/7 (PR #7 of feat/a)\nMETHOD: squash\nHEAD: " + fx.head + "\nPOSITION: 2\n" +
					"ENQUEUED AT: " + mergeCLIQueueEnqueuedAt + "\nPROTECTION: " + mergeCLIQueueProtection + "\n"
				if tc.requeue {
					want += "NOTE: PR #7 was " + removal + ", after " + gateWhen + ", and --requeue added it to the merge queue again\n"
				}
				if stdout != want {
					t.Errorf("stdout:\n%s\nwant:\n%s", stdout, want)
				}
				wantCall := "GH_PROMPT_DISABLED=1 api --hostname github.com -f query="
				wantTail := " -f pullRequestId=" + mergeCLIQueueNodeID + " -f expectedHeadOid=" + fx.head + " graphql"
				if got := fx.enqueueCalls(t); len(got) != 1 || !strings.HasPrefix(got[0], wantCall) || !strings.HasSuffix(got[0], wantTail) {
					t.Errorf("enqueuePullRequest calls = %q; want one that ends %q", got, wantTail)
				}
			}
			if got := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); got != before {
				t.Errorf("the ledger changed:\nbefore %s\nafter  %s", before, got)
			}
			if got := fx.mergeCalls(t); len(got) != 0 {
				t.Errorf("gh pr merge ran on the merge queue path: %q", got)
			}
			if slices.ContainsFunc(fx.ghCalls(t), func(c string) bool { return strings.Contains(c, " repo view ") }) {
				t.Errorf("gh repo view ran on the merge queue path: %q", fx.ghCalls(t))
			}
			holdMergeOrderLock(t, filepath.Join(fx.stateDir, "regate.lock"))
		})
	}
}

// AC14 for the help: ralph org merge --help describes the merge queue path
// (how the protection check and the method change, enqueuePullRequest at the
// gate's head with no gh pr merge, the wait, the record with via=merge_queue,
// the QUEUED: block and exit 7, and the PR out of the queue), --requeue (the
// flag and its paragraph), and exit 7 among the exit codes; it no longer says
// that merge requires no merge queue. With FR-12 of spec
// docs/specs/2026-11-19-org-merge-queue.md (plan
// docs/plans/active/2026-11-20-org-merge-queue-order.md, S4), a merge queue
// lets the check of merge-order's class and the second ls-remote pass a
// default branch that moved past the gate's base, so exit 7 says to run merge
// again with no condition on that branch.
func TestOrgMerge_HelpDescribesTheMergeQueuePath(t *testing.T) {
	out, err := runOrgCmd(t, "merge", "--help")
	if err != nil {
		t.Fatalf("merge --help: %v", err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"or else a merge queue with at least one required status check (the merge queue path, below)",
		"(viewerDefaultMergeMethod: MERGE, SQUASH, or REBASE), except on the merge queue path",
		"When origin's default branch has a merge queue (GraphQL's mergeQueue of the branch, or a merge_queue rule of a ruleset), " +
			"merge takes the merge queue path.",
		"The protection check then requires at least one required status check, of the rules or else of the branch protection, " +
			"refuses a queue with none",
		"METHOD: is the merge method of the queue's configuration (merge, squash, or rebase), or queue when GitHub gives none.",
		"puts the PR in the queue with GraphQL's enqueuePullRequest at the gate's head, once, with no gh pr merge and no --admin.",
		"one GitHub removed from the queue after the branch's latest gate record is refused without --requeue (below).",
		"With --wait-checks, merge then releases regate.lock and waits, within what is left of that duration, for the queue to " +
			"merge the PR, reading only the PR's merge queue state every 30 s.",
		"merge records pr_merged under the manifest lock only, its details ending with via=merge_queue",
		"prints QUEUED: <PR url> (PR #<n> of <branch>), METHOD:, HEAD:, POSITION: and ENQUEUED AT: (when GitHub gave the PR's " +
			"entry in the queue), PROTECTION:, GATE: (for an accepted branch), and NOTE:",
		"prints the MERGED: lines above, with a NOTE: line when the PR was in the queue already or --requeue added it again, " +
			"and one when another merge that waited for the same queue had recorded it already",
		"says on stderr that the PR is in the queue, and exits 7: run merge again to keep waiting, also once origin's default " +
			"branch moved (it reads the queue first and does not add the PR twice), or ralph org cleanup",
		"a branch whose gate base origin's default branch moved past is ready or accepted only when that branch has a merge " +
			"queue, which merge reads for this check only then",
		"a second git ls-remote finds origin's default branch and the branch where the gate left them (on the merge queue " +
			"path, the tip of origin's default branch need not be the gate's base: that ls-remote checks only that origin's " +
			"HEAD still names the same default branch and that the branch's tip is still the gate's head)",
		"or ralph org cleanup --branch <branch> once GitHub merged the PR.",
		"When a read finds the PR out of the queue and not merged, merge records nothing, says why on stderr, with GitHub's " +
			"removal of the PR and its reason when one was read, and exits 1.",
		"--requeue is for the merge queue path only: when GitHub removed the PR from the queue after the branch's latest gate " +
			"record, or less than 5 minutes before it",
		"Give it on purpose, for example after a flaky check, never by default.",
		"On the merge queue path, the wait for the queue (above) runs within the same duration, counted from the first wait.",
		"7 for a PR that merge left in the merge queue, not merged yet, with nothing recorded (run merge again to keep waiting).",
		"--requeue on a branch with a merge queue, put the PR in the queue again though GitHub removed it from the queue after " +
			"the branch's latest gate record (or less than 5 minutes before it), which merge refuses without it; use it on " +
			"purpose, such as after a flaky check, never by default",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("merge --help does not say %q:\n%s", want, out)
		}
	}
	if strings.Contains(help, "shows no merge queue") {
		t.Errorf("merge --help still says the protection shows no merge queue:\n%s", out)
	}
	for _, old := range []string{"once that branch moved, merge refuses before it reads the queue",
		"while origin's default branch is where the gate left it", "while origin's default branch has not moved"} {
		if strings.Contains(help, old) {
			t.Errorf("merge --help still says %q:\n%s", old, out)
		}
	}
}
