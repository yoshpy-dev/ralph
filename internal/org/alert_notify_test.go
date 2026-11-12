package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// The tests of plan 2026-11-12-alert-notify-batch: the ALERT items of one
// sending step of the watch (a deadline sweep, one org's deadman step) get
// one desktop notification, or none with [org.watchdog].alert_notify =
// "off".

// recordAlertItemAt records a TYPE ALERT item of orgID about seat-1's
// condition straight into s at time at, with delivery deadline, as the
// cross-org watch does, puts the inbox clock back, and returns its ID.
func recordAlertItemAt(t *testing.T, s *InboxStore, at time.Time, orgID, condition string) string {
	t.Helper()
	saved := s.Now
	s.Now = func() time.Time { return at }
	defer func() { s.Now = saved }()
	alertID := conditionKey(orgID, "seat-1", condition) + "@1"
	id, err := s.Escalate(InboxEscalation{OrgID: orgID, Type: protocol.TypeAlert, Body: alertItemBody, Delivery: InboxDeliveryDeadline, AlertID: alertID})
	if err != nil {
		t.Fatalf("record the ALERT item %s: %v", alertID, err)
	}
	return id
}

// ackSends returns "<id> inbox_ack_deadline" for each of ids, the sends
// assertSweep compares with.
func ackSends(ids ...string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id+" "+InboxReasonAckDeadline)
	}
	return out
}

// notifiedEvents returns "<reason> <osascript>" for each notified event of
// item.
func notifiedEvents(item InboxItem) []string {
	var out []string
	for _, ev := range item.Events {
		if ev.Event == InboxEventNotified {
			out = append(out, ev.Reason+" "+ev.Osascript)
		}
	}
	return out
}

// bannerLines returns the lines written to banner.
func bannerLines(banner string) []string {
	return strings.Split(strings.TrimSuffix(banner, "\n"), "\n")
}

// breakEscalatedLine replaces the escalated line of item id in s's
// inbox.jsonl with a line that does not parse, so the inbox no longer folds
// an item id and AppendNotified(id) fails with ErrInboxUnknownID; the other
// items stay. It returns the line it replaced, for restoreEscalatedLine.
func breakEscalatedLine(t *testing.T, s *InboxStore, id string) []byte {
	t.Helper()
	raw, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatalf("read inbox: %v", err)
	}
	lines := bytes.Split(raw, []byte("\n"))
	var replaced []byte
	for i, line := range lines {
		var ev InboxEvent
		if json.Unmarshal(line, &ev) == nil && ev.ID == id && ev.Event == InboxEventEscalated {
			replaced = bytes.Clone(line)
			lines[i] = line[:len(line)-1]
		}
	}
	if replaced == nil {
		t.Fatalf("inbox.jsonl has no escalated line of %s", id)
	}
	if err := os.WriteFile(s.Path(), bytes.Join(lines, []byte("\n")), 0o644); err != nil {
		t.Fatalf("write inbox: %v", err)
	}
	return replaced
}

// restoreEscalatedLine appends line, an escalated line breakEscalatedLine
// replaced, to s's inbox.jsonl, so the item folds again with no notified
// event.
func restoreEscalatedLine(t *testing.T, s *InboxStore, line []byte) {
	t.Helper()
	f, err := os.OpenFile(s.Path(), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open inbox: %v", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(append(line, '\n')); err != nil {
		t.Fatalf("append to inbox: %v", err)
	}
}

// AC1 for the sweep: three ALERT items and a QUESTION past the ack deadline.
// The QUESTION is notified on its own when its turn comes, and the three
// ALERT items share one notification at the end of the sweep, which names
// how many there are, the oldest (the smallest e<N>), and its org. Each item
// still gets its own escalations.jsonl line and banner, and each ALERT item
// a notified event with the one notification's result, failed: <err>
// included.
func TestSweepInboxDeadlines_SendsTheAlertItemsOfOneSweepInOneNotification(t *testing.T) {
	cases := []struct {
		name          string
		notifyErr     error
		wantOsascript string
	}{
		{"the notification is sent", nil, "ok"},
		{"the notification fails", errors.New("exit status 1: Not authorized to send Apple events"), "failed: exit status 1: Not authorized to send Apple events"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDeadlineFixture(t)
			f.notifyErr = tc.notifyErr
			mustEscalate(t, f.o.Inbox, "org-a")                             // e1, a leader's QUESTION
			recordAlertItemAt(t, f.o.Inbox, deadlineT0, "org-b", condStall) // e2
			recordAlertItemAt(t, f.o.Inbox, deadlineT0, "org-c", condLiveness)
			recordAlertItemAt(t, f.o.Inbox, deadlineT0, "org-b", condLiveness)

			assertSweep(t, "past the ack deadline", f.sweep(11*time.Minute, &InboxDeadlineMemory{}), ackSends("e1", "e2", "e3", "e4"), nil, nil)

			want := []string{"org org-a: e1 (QUESTION) not acked within 10 min", "3 ALERT items; the oldest is e2 of org org-b"}
			if got := f.notifications(); !slices.Equal(got, want) {
				t.Errorf("desktop notifications = %q; want %q", got, want)
			}
			if got := escalationReasons(t, f.o.EscalationsPath); !slices.Equal(got, ackSends("e1", "e2", "e3", "e4")) {
				t.Errorf("escalations.jsonl = %v; want one line per item", got)
			}
			lines := bannerLines(f.banner.String())
			wantPrefixes := []string{
				"ORG INBOX DEADLINE: org=org-a item=e1 type=QUESTION -- ",
				"ORG INBOX DEADLINE: org=org-b item=e2 type=ALERT -- ",
				"ORG INBOX DEADLINE: org=org-c item=e3 type=ALERT -- ",
				"ORG INBOX DEADLINE: org=org-b item=e4 type=ALERT -- ",
			}
			if len(lines) != len(wantPrefixes) {
				t.Fatalf("banner = %q; want one line per item", lines)
			}
			for i, prefix := range wantPrefixes {
				if !strings.HasPrefix(lines[i], prefix) {
					t.Errorf("banner line %d = %q; want it to start %q", i, lines[i], prefix)
				}
			}
			if strings.Contains(f.banner.String()+strings.Join(f.notifications(), ""), "did not act") {
				t.Errorf("an ALERT item's body reached the banner or the notification")
			}
			inbox := mustReadInbox(t, f.o.Inbox)
			for _, id := range []string{"e1", "e2", "e3", "e4"} {
				item := inboxItemByID(t, inbox, id)
				if got, want := notifiedEvents(item), []string{InboxReasonAckDeadline + " " + tc.wantOsascript}; item.State != InboxStateOpen || !slices.Equal(got, want) {
					t.Errorf("%s: state %s, notified events %q; want open and %q", id, item.State, got, want)
				}
			}
		})
	}
}

// AC3: with alert_notify off, the sweep sends no desktop notification for
// its ALERT items, and their notified events record osascript off, so
// neither the next sweep nor a restarted watch sends them again at the ack
// deadline. Their banners and escalations.jsonl lines are written, and the
// QUESTION of the same sweep is still notified.
func TestSweepInboxDeadlines_AlertNotifyOffSendsNoNotificationForTheAlertItems(t *testing.T) {
	f := newDeadlineFixture(t)
	f.o.Config.Watchdog.AlertNotify = config.AlertNotifyOff
	recordAlertItemAt(t, f.o.Inbox, deadlineT0, "org-a", condStall) // e1
	mustEscalate(t, f.o.Inbox, "org-b")                             // e2
	recordAlertItemAt(t, f.o.Inbox, deadlineT0, "org-c", condStall) // e3
	mem := &InboxDeadlineMemory{}

	assertSweep(t, "past the ack deadline", f.sweep(11*time.Minute, mem), ackSends("e1", "e2", "e3"), nil, nil)
	if got, want := f.notifications(), []string{"org org-b: e2 (QUESTION) not acked within 10 min"}; !slices.Equal(got, want) {
		t.Errorf("desktop notifications = %q; want only the QUESTION's %q", got, want)
	}
	if got := escalationReasons(t, f.o.EscalationsPath); !slices.Equal(got, ackSends("e1", "e2", "e3")) {
		t.Errorf("escalations.jsonl = %v; want one line per item", got)
	}
	if lines := bannerLines(f.banner.String()); len(lines) != 3 {
		t.Errorf("banner = %q; want one line per item", lines)
	}
	inbox := mustReadInbox(t, f.o.Inbox)
	for id, osascript := range map[string]string{"e1": inboxNotifyOff, "e2": inboxNotifyOK, "e3": inboxNotifyOff} {
		if got, want := notifiedEvents(inboxItemByID(t, inbox, id)), []string{InboxReasonAckDeadline + " " + osascript}; !slices.Equal(got, want) {
			t.Errorf("%s notified events = %q; want %q", id, got, want)
		}
	}

	assertSweep(t, "the next sweep", f.sweep(12*time.Minute, mem), nil, nil, nil)
	assertSweep(t, "a restarted watch", f.sweep(30*time.Minute, &InboxDeadlineMemory{}), nil, nil, nil)
	if n := len(f.notifications()); n != 1 {
		t.Errorf("desktop notified %d times; want only the QUESTION's 1", n)
	}
}

// AC1c (D2b): the notified event of one ALERT item of the sweep cannot be
// appended at the end of the sweep. That send is Failed and remembered like
// any failed send: the next sweep within ack_minutes holds it back, and the
// sweep ack_minutes after the attempt sends it again, on its own. The other
// ALERT items of the batch are Sent and never sent again.
func TestSweepInboxDeadlines_AnAlertItemWhoseNotifiedEventIsNotWrittenIsHeldBack(t *testing.T) {
	f := newDeadlineFixture(t)
	recordAlertItemAt(t, f.o.Inbox, deadlineT0, "org-a", condStall)    // e1
	recordAlertItemAt(t, f.o.Inbox, deadlineT0, "org-b", condStall)    // e2
	recordAlertItemAt(t, f.o.Inbox, deadlineT0, "org-c", condLiveness) // e3
	var e2Line []byte
	f.onNotify = func() {
		// Once, inside the one notification of the sweep: after every line
		// and banner and before the notified events.
		f.onNotify = nil
		e2Line = breakEscalatedLine(t, f.o.Inbox, "e2")
	}
	mem := &InboxDeadlineMemory{}
	sweep := f.sweep(11*time.Minute, mem)
	assertSweep(t, "the sweep whose append of e2 fails", sweep, ackSends("e1", "e3"), ackSends("e2"), nil)
	if err := sweep.Failed[0].Err; !errors.Is(err, ErrInboxUnknownID) {
		t.Errorf("e2's error = %v; want the failed append's ErrInboxUnknownID", err)
	}
	restoreEscalatedLine(t, f.o.Inbox, e2Line)
	if item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e2"); item.Notified || item.State != InboxStateOpen {
		t.Fatalf("e2 = %+v; want open and not notified", item)
	}

	assertSweep(t, "within ack_minutes", f.sweep(12*time.Minute, mem), nil, nil, ackSends("e2"))
	assertSweep(t, "ack_minutes after the attempt", f.sweep(21*time.Minute, mem), ackSends("e2"), nil, nil)
	assertSweep(t, "after the retry", f.sweep(22*time.Minute, mem), nil, nil, nil)

	want := []string{"3 ALERT items; the oldest is e1 of org org-a", "org org-b: e2 (ALERT) not acked within 10 min"}
	if got := f.notifications(); !slices.Equal(got, want) {
		t.Errorf("desktop notifications = %q; want %q", got, want)
	}
	if got, want := escalationReasons(t, f.o.EscalationsPath), ackSends("e1", "e2", "e3", "e2"); !slices.Equal(got, want) {
		t.Errorf("escalations.jsonl = %v; want %v (e2's line of the failed send and of the retry)", got, want)
	}
	inbox := mustReadInbox(t, f.o.Inbox)
	for _, id := range []string{"e1", "e2", "e3"} {
		if got, want := notifiedEvents(inboxItemByID(t, inbox, id)), []string{InboxReasonAckDeadline + " ok"}; !slices.Equal(got, want) {
			t.Errorf("%s notified events = %q; want %q", id, got, want)
		}
	}
}

// The plan's edge case: an ALERT item resolved between its turn in the
// sweep and the end of the sweep still gets its notified event and is Sent,
// as a notified event is recorded for an item in any state.
func TestSweepInboxDeadlines_AnAlertItemResolvedBeforeTheEndOfTheSweepIsStillSent(t *testing.T) {
	f := newDeadlineFixture(t)
	recordAlertItemAt(t, f.o.Inbox, deadlineT0, "org-a", condStall)
	recordAlertItemAt(t, f.o.Inbox, deadlineT0, "org-b", condStall)
	f.onNotify = func() {
		f.onNotify = nil
		if err := f.o.Inbox.Resolve("e2", "docs/reports/x.md"); err != nil {
			t.Errorf("resolve e2: %v", err)
		}
	}
	assertSweep(t, "past the ack deadline", f.sweep(11*time.Minute, &InboxDeadlineMemory{}), ackSends("e1", "e2"), nil, nil)
	item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e2")
	if got, want := notifiedEvents(item), []string{InboxReasonAckDeadline + " ok"}; item.State != InboxStateResolved || !slices.Equal(got, want) {
		t.Errorf("e2: state %s, notified events %q; want resolved and %q", item.State, got, want)
	}
}

// AC1b: in a RunWatchAll cycle, the one notification of the sweep's ALERT
// items is sent, and their notified events written, before the pulse makes
// its first herdr call, so an org whose herdr does not answer cannot hold
// it back.
func TestRunWatchAll_SendsTheSweepsAlertNotificationBeforeThePulse(t *testing.T) {
	f := newWatchAllFixture(t)
	f.spawn(t, "org-a", "seat-a")
	past := f.clk.Now().Add(-11 * time.Minute)
	recordAlertItemAt(t, f.o.Inbox, past, "org-a", condStall)
	recordAlertItemAt(t, f.o.Inbox, past, "org-b", condStall)
	var herdrCalls int
	var notifiedAtFirstCall []string
	var unnotifiedAtFirstCall []string
	f.o.Herdr = &beforeAgentGetHerdr{fakeWatchHerdr: f.h, before: func() {
		herdrCalls++
		if herdrCalls > 1 {
			return
		}
		notifiedAtFirstCall = f.notifications()
		for _, item := range f.alertItems(t) {
			if !item.Notified {
				unnotifiedAtFirstCall = append(unnotifiedAtFirstCall, item.ID)
			}
		}
	}}
	if err := runWatchAll(t, f.o, context.Background(), f.params(1), WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	if herdrCalls == 0 {
		t.Fatal("the cycle made no herdr call")
	}
	if want := []string{"2 ALERT items; the oldest is e1 of org org-a"}; !slices.Equal(notifiedAtFirstCall, want) || len(unnotifiedAtFirstCall) != 0 {
		t.Errorf("at the first herdr call: notifications %q, unnotified items %v; want %q and none", notifiedAtFirstCall, unnotifiedAtFirstCall, want)
	}
	if n := len(f.notifications()); n != 1 {
		t.Errorf("desktop notified %d times in the cycle; want 1", n)
	}
}

// pendingLeaderAlerts returns a status of orgID holding one pending ALERT
// about the leader for each of conditions, raised at clk's time, which the
// deadman step escalates at once with leader_is_anomaly_subject, and their
// alert_ids in the order of conditions.
func pendingLeaderAlerts(clk *fakeClock, orgID string, conditions ...string) (*watchStatusFile, []string) {
	status := &watchStatusFile{OrgID: orgID, PendingAlerts: map[string]*watchPendingAlert{}, Escalated: map[string]bool{}}
	var ids []string
	for i, condition := range conditions {
		alertID := fmt.Sprintf("%s@%d", conditionKey(orgID, LeaderIdentity, condition), clk.Now().UnixNano()+int64(i))
		status.PendingAlerts[alertID] = &watchPendingAlert{AlertID: alertID, TS: clk.Now().UTC().Format(time.RFC3339), Subject: LeaderIdentity, HistoryLeaderLines: -1}
		ids = append(ids, alertID)
	}
	return status, ids
}

// AC1 for the deadman step: with the director switch off, the two ALERTs
// about org-a's leader that one deadman step escalates are sent to the
// human at once, one of them as the item the inbox already holds
// (escalateRecordedItem) and one as a new item, and share one desktop
// notification at the end of the step. Each has its escalations.jsonl line,
// its banner, and a notified event with the one notification's result. The
// deadman step of a second org is a step of its own, with its own
// notification.
func TestWatch_Deadman_SendsTheAlertItemsOfOneOrgsStepInOneNotification(t *testing.T) {
	f := newWatchInboxFixture(t)
	statusA, alertsA := pendingLeaderAlerts(f.clk, "org-a", condLiveness, condStall)
	// A watch that recorded the first alert as e1 and ended before it sent
	// it or saved its status file.
	escalateAlertItem(t, f.o.Inbox, alertsA[0], InboxDeliveryNow)
	run := f.run(nil)
	run.checkDeadman(statusA, ManifestReadResult{}, f.clk.Now())

	if want := []string{"2 ALERT items; the oldest is e1 of org org-a"}; !slices.Equal(f.notified, want) {
		t.Fatalf("desktop notifications after org-a's step = %q; want %q", f.notified, want)
	}
	inbox := mustReadInbox(t, f.o.Inbox)
	if ids := inboxItemIDs(inbox); !slices.Equal(ids, []string{"e1", "e2"}) {
		t.Fatalf("inbox items = %v; want e1 and the new e2", ids)
	}
	for i, id := range []string{"e1", "e2"} {
		item := inboxItemByID(t, inbox, id)
		if got, want := notifiedEvents(item), []string{InboxReasonDirectorDisabled + " ok"}; item.AlertID != alertsA[i] || !slices.Equal(got, want) {
			t.Errorf("%s: alert_id %s, notified events %q; want %s and %q", id, item.AlertID, got, alertsA[i], want)
		}
		if !strings.Contains(f.stderr.String(), "WATCHDOG ESCALATION: org=org-a item="+id+" type=ALERT alert="+alertsA[i]+" -- recorded in ") {
			t.Errorf("stderr = %q; want the banner of %s", f.stderr.String(), id)
		}
	}
	// checkDeadman ranges over a map, so the lines come in either order.
	got := escalationReasons(t, f.o.EscalationsPath)
	slices.Sort(got)
	if want := []string{"e1 " + InboxReasonDirectorDisabled, "e2 " + InboxReasonDirectorDisabled}; !slices.Equal(got, want) {
		t.Errorf("escalations.jsonl = %v; want one line per item", got)
	}
	assertEscalatedOnce(t, statusA, alertsA[0])
	assertEscalatedOnce(t, statusA, alertsA[1])

	statusB, _ := pendingLeaderAlerts(f.clk, "org-b", condLiveness, condStall)
	run.checkDeadman(statusB, ManifestReadResult{}, f.clk.Now())
	if want := []string{"2 ALERT items; the oldest is e1 of org org-a", "2 ALERT items; the oldest is e3 of org org-b"}; !slices.Equal(f.notified, want) {
		t.Errorf("desktop notifications after org-b's step = %q; want %q", f.notified, want)
	}
}

// AC3 for the deadman step: with alert_notify off, the ALERT items the step
// sends at once get no desktop notification, and their notified events
// record off; their banners and escalations.jsonl lines are written.
func TestWatch_Deadman_AlertNotifyOffSendsNoNotification(t *testing.T) {
	f := newWatchInboxFixture(t)
	f.o.Config.Watchdog.AlertNotify = config.AlertNotifyOff
	status, _ := pendingLeaderAlerts(f.clk, "org-a", condLiveness, condStall)
	f.run(nil).checkDeadman(status, ManifestReadResult{}, f.clk.Now())
	if len(f.notified) != 0 || len(f.escalated) != 0 {
		t.Errorf("desktop notifications %q, without-inbox notifications %q; want none", f.notified, f.escalated)
	}
	inbox := mustReadInbox(t, f.o.Inbox)
	for _, id := range []string{"e1", "e2"} {
		if got, want := notifiedEvents(inboxItemByID(t, inbox, id)), []string{InboxReasonDirectorDisabled + " " + inboxNotifyOff}; !slices.Equal(got, want) {
			t.Errorf("%s notified events = %q; want %q", id, got, want)
		}
	}
	if n := len(escalationReasons(t, f.o.EscalationsPath)); n != 2 || strings.Count(f.stderr.String(), "WATCHDOG ESCALATION: org=org-a item=") != 2 {
		t.Errorf("%d escalations.jsonl lines, stderr %q; want two lines and two banners", n, f.stderr.String())
	}
}

// AC1c for the deadman step: a notified event that cannot be appended at the
// end of the step gets the stderr line of a send that was not completed,
// naming `ralph org inbox notify <id>`; the other item of the step is
// notified.
func TestWatch_Deadman_ANotifiedEventNotWrittenAtTheEndOfTheStepNamesInboxNotify(t *testing.T) {
	f := newWatchInboxFixture(t)
	status, _ := pendingLeaderAlerts(f.clk, "org-a", condLiveness, condStall)
	f.o.DesktopNotify = func(_ context.Context, message string) error {
		f.notified = append(f.notified, message)
		breakEscalatedLine(t, f.o.Inbox, "e2")
		return nil
	}
	f.run(nil).checkDeadman(status, ManifestReadResult{}, f.clk.Now())
	stderr := f.stderr.String()
	if !strings.Contains(stderr, "WATCHDOG ESCALATION: org=org-a item=e2 -- recorded, but the send to the human was not completed (") ||
		!strings.Contains(stderr, "run `"+InboxCommand("notify", "e2", "")+"` to send it again") {
		t.Errorf("stderr = %q; want the line naming the inbox notify command for e2", stderr)
	}
	if strings.Contains(stderr, "item=e1 -- recorded, but") {
		t.Errorf("stderr = %q; want no such line for e1", stderr)
	}
	if got := notifiedEvents(inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1")); len(got) != 1 {
		t.Errorf("e1 notified events = %q; want one", got)
	}
	if len(f.notified) != 1 {
		t.Errorf("desktop notifications = %q; want one for the step", f.notified)
	}
}

// AC5 (D4): a watch that ends after a send of an ALERT item went into its
// step's batch and before the end of the step leaves the item's
// escalations.jsonl line and banner but no notification and no notified
// event, so the next watch's sweep sends it at the ack deadline, and not
// before.
func TestSendInboxItemToHuman_AnAlertItemOfABatchNeverFlushedIsSentAtTheAckDeadline(t *testing.T) {
	f := newDeadlineFixture(t)
	const alertID = "org-a/leader/liveness@1"
	id := escalateAlertItem(t, f.o.Inbox, alertID, InboxDeliveryNow)
	batch := &alertNotifyBatch{}
	if err := f.o.sendInboxItemToHuman("org-a", id, protocol.TypeAlert, alertID, InboxReasonNoWatch, &f.banner, batch); err != nil {
		t.Fatalf("send into the batch: %v", err)
	}
	// The watch ends here: the batch is never flushed.
	if got := escalationReasons(t, f.o.EscalationsPath); !slices.Equal(got, []string{id + " " + InboxReasonNoWatch}) {
		t.Errorf("escalations.jsonl = %v; want the send's line", got)
	}
	if lines := bannerLines(f.banner.String()); len(lines) != 1 || !strings.HasPrefix(lines[0], "WATCHDOG ESCALATION: org=org-a item="+id+" type=ALERT ") {
		t.Errorf("banner = %q; want the send's banner", lines)
	}
	if item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), id); item.Notified || len(f.notifications()) != 0 {
		t.Fatalf("%s = %+v, notifications %q; want no notified event and no notification", id, item, f.notifications())
	}

	assertSweep(t, "the next watch, at the ack deadline", f.sweep(10*time.Minute, &InboxDeadlineMemory{}), nil, nil, nil)
	assertSweep(t, "the next watch, past the ack deadline", f.sweep(11*time.Minute, &InboxDeadlineMemory{}), ackSends(id), nil, nil)
	if got, want := f.notifications(), []string{"org org-a: " + id + " (ALERT) not acked within 10 min"}; !slices.Equal(got, want) {
		t.Errorf("desktop notifications = %q; want %q", got, want)
	}
}

// AC4: the sends that are not a sending step of the watch do not read
// alert_notify: `ralph org inbox notify` of an ALERT item and a leader's
// escalate sent at once are notified on their own with alert_notify off,
// and so is an escalation the inbox could not take.
func TestNotifyInboxItem_AlertNotifyOffDoesNotStopTheSendsOutsideTheWatchsSteps(t *testing.T) {
	t.Run("inbox notify and escalate", func(t *testing.T) {
		f := newEscalateFixture(t)
		f.o.Config.Watchdog.AlertNotify = config.AlertNotifyOff
		id := escalateAlertItem(t, f.o.Inbox, "org-a/seat-1/stall@1", InboxDeliveryDeadline)
		if err := f.o.NotifyInboxItem(id, &f.banner); err != nil {
			t.Fatalf("NotifyInboxItem: %v", err)
		}
		if res := f.escalate("org-b", "TYPE: QUESTION\n\nwhich base?"); res.Err != nil || !res.Notified {
			t.Fatalf("Escalate = %+v; want it sent at once", res)
		}
		want := []string{"org org-a raised " + id + " (ALERT)", "org org-b raised e2 (QUESTION)"}
		if got := f.notifications(); !slices.Equal(got, want) {
			t.Errorf("desktop notifications = %q; want %q", got, want)
		}
		if got, want := notifiedEvents(inboxItemByID(t, mustReadInbox(t, f.o.Inbox), id)), []string{InboxReasonManualNotify + " ok"}; !slices.Equal(got, want) {
			t.Errorf("%s notified events = %q; want %q", id, got, want)
		}
	})
	t.Run("an escalation the inbox could not take", func(t *testing.T) {
		f := newWatchInboxFixture(t)
		f.o.Config.Watchdog.AlertNotify = config.AlertNotifyOff
		f.o.Inbox = nil
		status, _ := pendingLeaderAlerts(f.clk, "org-a", condLiveness)
		f.run(nil).checkDeadman(status, ManifestReadResult{}, f.clk.Now())
		if len(f.escalated) != 1 {
			t.Errorf("without-inbox notifications = %q; want one", f.escalated)
		}
	})
}

// D2: the text of the one notification of a step. One item keeps the text
// of a send of that item alone (the deadline text, or the watch's text of
// an ALERT item sent at once). Two or more name how many there are and the
// item with the smallest e<N> (e9 before e10, not by string order), with
// its org guarded as a single item's is.
func TestAlertNotifyText_NamesTheCountTheOldestAndItsOrg(t *testing.T) {
	o := newDeadlineFixture(t).o
	item := func(orgID, id, reason string) alertNotifyItem {
		return alertNotifyItem{orgID: orgID, id: id, alertID: orgID + "/seat-1/stall@1", reason: reason, lineWritten: true}
	}
	cases := []struct {
		name                   string
		items                  []alertNotifyItem
		wantTitle, wantMessage string
	}{
		{"one item at its ack deadline", []alertNotifyItem{item("org-a", "e7", InboxReasonAckDeadline)},
			inboxNotifyTitleWatch, "org org-a: e7 (ALERT) not acked within 10 min"},
		{"one item sent at once", []alertNotifyItem{item("org-a", "e7", InboxReasonDirectorDisabled)},
			inboxNotifyTitleWatch, "org org-a raised e7 (ALERT)"},
		{"three items, the oldest not first", []alertNotifyItem{
			item("org-c", "e10", InboxReasonAckDeadline), item("org-b", "e9", InboxReasonResolveDeadline), item("org-a", "e12", InboxReasonAckDeadline),
		}, inboxNotifyTitleWatch, "3 ALERT items; the oldest is e9 of org org-b"},
		{"the oldest item's org is not an identifier", []alertNotifyItem{
			item("org-a", "e3", InboxReasonNoWatch), item("org-b\x07\"", "e2", InboxReasonNoWatch),
		}, inboxNotifyTitleWatch, "2 ALERT items; the oldest is e2 of org " + inboxNotifyInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			title, message := o.alertNotifyText(tc.items)
			if title != tc.wantTitle || message != tc.wantMessage {
				t.Errorf("alertNotifyText = (%q, %q); want (%q, %q)", title, message, tc.wantTitle, tc.wantMessage)
			}
		})
	}
}

// A nil or empty batch sends and appends nothing, and a batch is empty
// after its flush, so a second flush sends nothing again.
func TestFlushAlertNotifyBatch_SendsNothingForAnEmptyBatchAndEmptiesTheBatch(t *testing.T) {
	f := newDeadlineFixture(t)
	if r := f.o.flushAlertNotifyBatch(nil); r != nil || len(f.notifications()) != 0 {
		t.Fatalf("flush of a nil batch = %v, notifications %q; want nothing", r, f.notifications())
	}
	batch := &alertNotifyBatch{}
	if r := f.o.flushAlertNotifyBatch(batch); r != nil || len(f.notifications()) != 0 {
		t.Fatalf("flush of an empty batch = %v, notifications %q; want nothing", r, f.notifications())
	}
	id := recordAlertItemAt(t, f.o.Inbox, deadlineT0, "org-a", condStall)
	if err := f.o.sendInboxItemToHuman("org-a", id, protocol.TypeAlert, "", InboxReasonNoWatch, &f.banner, batch); err != nil {
		t.Fatal(err)
	}
	if r := f.o.flushAlertNotifyBatch(batch); len(r) != 1 || r[0].id != id || r[0].err != nil {
		t.Fatalf("flush = %+v; want %s appended", r, id)
	}
	if r := f.o.flushAlertNotifyBatch(batch); r != nil || len(f.notifications()) != 1 {
		t.Errorf("second flush = %v, notifications %q; want nothing more", r, f.notifications())
	}
	if got := notifiedEvents(inboxItemByID(t, mustReadInbox(t, f.o.Inbox), id)); len(got) != 1 {
		t.Errorf("%s notified events = %q; want one", id, got)
	}
}
