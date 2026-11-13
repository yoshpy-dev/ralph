package org

import (
	"slices"
	"strings"
	"testing"
)

// AC4 (cross-review cycle 1 of feat/watch-usage-limited, finding 1): with
// org-a's usage_limited item e1 of the leader and the implementer open, the
// reviewer stops on a usage limit in the next cycle, and the director acks
// e1 after the watch read the inbox and found e1 open, before the append of
// the recurred event. The append checks under the inbox lock that e1 is
// still open, so e1 stays acked with no recurred event, the three seats get
// a new item e2 and a second ALERT to the leader, and no stderr line says a
// recurrence was recorded or failed. In the next cycle no seat is new, so
// nothing more is recorded.
func TestWatchUsageLimit_ASeatAddedWhileTheItemIsAckedBeforeTheAppendGetsANewItem(t *testing.T) {
	f := newUsageLimitFixture(t, threeSeats...)
	f.limitThreeSeats(t)
	f.reply("reviewer", watchAgentReply("working", 1))
	f.cycle(t)
	f.reply("reviewer", watchAgentReply("working", 2))
	f.cycle(t)
	if items := f.usageLimitedItems(t); len(items) != 1 || items[0].State != InboxStateOpen || !strings.Contains(items[0].Body, "\nSEATS: implementer, leader\n") {
		t.Fatalf("usage_limited items %+v; want e1 of the leader and the implementer, open", items)
	}
	acks := 0
	setBeforeAppendRecurred(t, func() {
		acks++
		if _, err := f.o.Inbox.Ack("e1"); err != nil {
			t.Error(err)
		}
	})

	f.cycle(t) // the reviewer's answer stops changing: it is stalled, on its weekly limit
	if acks != 1 {
		t.Fatalf("the seam ran %d times; want once, between the read that found e1 open and the append", acks)
	}
	if !slices.Equal(f.status.UsageLimitedSeats, []string{"implementer", LeaderIdentity, "reviewer"}) || len(f.alerts(condStall)) != 0 {
		t.Errorf("usage_limited seats %v, stall ALERTs %q; want the three seats and no stall", f.status.UsageLimitedSeats, f.alerts(condStall))
	}
	items := f.usageLimitedItems(t)
	if len(items) != 2 || items[1].ID != "e2" || len(f.alerts(condUsageLimited)) != 2 {
		t.Fatalf("items %+v, ALERTs %q; want e1 and a new e2, and a second ALERT", items, f.alerts(condUsageLimited))
	}
	if e1 := items[0]; e1.State != InboxStateAcked || len(recurredEvents(e1)) != 0 {
		t.Errorf("e1 = %+v; want acked with no recurred event", e1)
	}
	if !strings.Contains(items[1].Body, "\nSEATS: implementer, leader, reviewer\n") || !strings.Contains(items[1].Body, "\nreviewer (claude): "+ulClaudeWeeklyLine) {
		t.Errorf("e2 body %q; want the three seats, the reviewer's line among them", items[1].Body)
	}
	for _, line := range []string{"recorded as a recurrence on inbox item e1", "but the recurrence was not recorded"} {
		if strings.Contains(f.stderr.String(), line) {
			t.Errorf("stderr = %q; want no line with %q", f.stderr.String(), line)
		}
	}

	f.cycle(t)
	if items := f.usageLimitedItems(t); len(items) != 2 || len(recurredEvents(items[0])) != 0 || len(recurredEvents(items[1])) != 0 || len(f.alerts(condUsageLimited)) != 2 {
		t.Errorf("after the next cycle: items %+v, ALERTs %q; want still e1 and e2 with no recurred event, and two ALERTs", items, f.alerts(condUsageLimited))
	}
}
