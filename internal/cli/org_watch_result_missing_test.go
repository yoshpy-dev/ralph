package cli

import (
	"strings"
	"testing"
)

// The help of watch lists result_missing among the conditions of the pulse and
// says what it is: a seat other than the leader that waits for input without
// the RESULT or REVIEW of its last TASK, with its stall held while the clock
// runs, counted per TASK, unlike the recurrence rule before it (a rise for
// another TASK is a new ALERT and a new item, only a rise for the same TASK
// is a recurrence of its own ALERT or item). The sentences follow the ones
// TestOrgWatch_HelpDescribesAlertRecurrencesLeaderStallAndReservedScope
// (org_watch_test.go) pins, which stay as they are (plan
// 2026-11-22-org-result-missing, verify V1 and V2). The test sits in its own
// file because org_watch_test.go is far past the 800-line guide already.
func TestOrgWatch_HelpDescribesResultMissingAndItsRecurrencePerTask(t *testing.T) {
	stdout, _, err := runOrgCmdStreams(t, "watch", "--help")
	if err != nil {
		t.Fatal(err)
	}
	help := strings.Join(strings.Fields(stdout), " ")
	for _, want := range []string{
		"heartbeat-stall / process-liveness / worktree-scope-change / usage_limited / result_missing ALERTs sent to the org's leader",
		"A seat other than the leader that waits for input for [org.watchdog].stall_minutes " +
			"without having sent the leader the RESULT or REVIEW of its last TASK",
		"gets a result_missing ALERT with that TASK's TASK_ID header, and its heartbeat-stall is held while that clock runs",
		"result_missing is counted per TASK, unlike the rule above: a rise for another TASK is a new ALERT and a new item, " +
			"even while the earlier ALERT or its item is open, and only a rise for the same TASK is a recurrence of its own ALERT or item",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("watch --help does not say %q:\n%s", want, stdout)
		}
	}
	// The recurrence rule it departs from is still the one stated before it.
	rule := strings.Index(help, "a re-firing is recorded as a recurrence of it")
	perTask := strings.Index(help, "result_missing is counted per TASK")
	if rule < 0 || perTask < rule {
		t.Errorf("watch --help should state result_missing's per-TASK recurrence after the general rule (rule at %d, per-TASK at %d):\n%s", rule, perTask, stdout)
	}
}
