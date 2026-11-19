package cli

import (
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for the merge queue path of `ralph org merge` at the command line
// (plan docs/plans/active/2026-11-20-org-merge-queue-verb.md, S3a; spec
// docs/specs/2026-11-19-org-merge-queue.md, FR-10): what printMergeResult
// does with a result that leaves the PR in the merge queue. The org layer's
// queue path is covered by internal/org/merge_queue_run_test.go.

// TestOrgMerge_PrintsAQueuedResultAsAnErrorWithNoLines: a result that left
// the PR in the merge queue, not merged and nothing recorded
// (MergeResult.Queued), prints no MERGED: lines and returns its Queued text
// as the error, so the command does not exit 0, which means a recorded
// merge, and does not say the outcome is unknown either.
func TestOrgMerge_PrintsAQueuedResultAsAnErrorWithNoLines(t *testing.T) {
	res := org.MergeResult{
		Queued: "org: merge: PR #7 (https://github.com/o/r/pull/7) was added to the merge queue of main at the head " +
			"aaaaaaaaaaaa, the head the gate passed, and it is at position 1, enqueued at 2026-10-14T09:00:30Z; it is not merged " +
			"yet, so nothing was recorded: run ralph org merge --branch feat/a again",
		QueuePosition: 1, EnqueuedAt: "2026-10-14T09:00:30Z", Branch: "feat/a", OrgID: "a", PRNumber: 7,
		PRURL: "https://github.com/o/r/pull/7", Method: "squash", Head: strings.Repeat("a", 40), Protection: "a merge queue",
	}
	var out strings.Builder
	err := printMergeResult(&out, res, nil, "")
	if out.String() != "" {
		t.Errorf("stdout = %q, want nothing", out.String())
	}
	if err == nil || err.Error() != res.Queued {
		t.Errorf("err = %v; want %q", err, res.Queued)
	}
}
