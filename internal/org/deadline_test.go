package org

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// deadlineT0 is the fixed clock of newEscalateFixture (inboxTestNow): the
// time the deadline tests' items are escalated unless a test says
// otherwise.
var deadlineT0 = time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

// newDeadlineFixture is newEscalateFixture with the default deadlines:
// ack_minutes 10, resolve_minutes 60.
func newDeadlineFixture(t *testing.T) *escalateFixture {
	t.Helper()
	f := newEscalateFixture(t)
	f.o.Config.Director = config.OrgDirectorConfig{AckMinutes: 10, ResolveMinutes: 60}
	return f
}

// sweep runs SweepInboxDeadlines at deadlineT0+at, with banner lines going
// to f.banner.
func (f *escalateFixture) sweep(at time.Duration, mem *InboxDeadlineMemory) InboxDeadlineSweep {
	return f.o.SweepInboxDeadlines(deadlineT0.Add(at), mem, &f.banner)
}

// escalateInboxAt records a QUESTION from orgID with the inbox clock set to
// at, then puts the clock back.
func escalateInboxAt(t *testing.T, s *InboxStore, at time.Time, orgID string) string {
	t.Helper()
	saved := s.Now
	s.Now = func() time.Time { return at }
	defer func() { s.Now = saved }()
	return mustEscalate(t, s, orgID)
}

// deadlineSends returns "<id> <reason>" for each send, so a list compares
// with slices.Equal.
func deadlineSends(sends []InboxDeadlineSend) []string {
	out := make([]string, 0, len(sends))
	for _, s := range sends {
		out = append(out, s.ID+" "+s.Reason)
	}
	return out
}

// assertSweep fails the test unless sweep has no error and exactly the
// given sends (each "<id> <reason>").
func assertSweep(t *testing.T, label string, sweep InboxDeadlineSweep, sent, failed, heldBack []string) {
	t.Helper()
	if sweep.Err != nil {
		t.Fatalf("%s: sweep error %v", label, sweep.Err)
	}
	for _, c := range []struct {
		name      string
		got, want []string
	}{
		{"Sent", deadlineSends(sweep.Sent), sent},
		{"Failed", deadlineSends(sweep.Failed), failed},
		{"HeldBack", deadlineSends(sweep.HeldBack), heldBack},
	} {
		if !slices.Equal(c.got, c.want) {
			t.Errorf("%s: %s = %v; want %v", label, c.name, c.got, c.want)
		}
	}
	for _, s := range sweep.Failed {
		if s.Err == nil {
			t.Errorf("%s: failed send %s %s has no error", label, s.ID, s.Reason)
		}
	}
	for _, s := range slices.Concat(sweep.Sent, sweep.HeldBack) {
		if s.Err != nil {
			t.Errorf("%s: send %s %s carries an error %v outside Failed", label, s.ID, s.Reason, s.Err)
		}
	}
}

// escalationReasons returns "<inbox_id> <reason>" for each escalations.jsonl
// line.
func escalationReasons(t *testing.T, path string) []string {
	t.Helper()
	var out []string
	for _, line := range escalationLines(t, path) {
		id, _ := line["inbox_id"].(string)
		reason, _ := line["reason"].(string)
		out = append(out, id+" "+reason)
	}
	return out
}

// deadlineTestItem is an item e1 escalated at inboxTestNow in state, with a
// notified event for each of notifiedReasons.
func deadlineTestItem(state string, notifiedReasons ...string) InboxItem {
	item := InboxItem{
		ID: "e1", OrgID: "org-a", Type: "QUESTION", State: state, EscalatedAt: inboxTestNow,
		Events: []InboxEvent{{TS: inboxTestNow, ID: "e1", Event: InboxEventEscalated, OrgID: "org-a", Type: "QUESTION"}},
	}
	for _, reason := range notifiedReasons {
		item.Events = append(item.Events, InboxEvent{TS: inboxTestNow, ID: "e1", Event: InboxEventNotified, Reason: reason, Osascript: "ok"})
		item.Notified, item.NotifiedAt = true, inboxTestNow
	}
	return item
}

// AC2: the due-check table, with ack_minutes 10 and resolve_minutes 60.
func TestInboxDeadlineDue(t *testing.T) {
	const (
		ack     = InboxReasonAckDeadline
		resolve = InboxReasonResolveDeadline
	)
	withEscalatedAt := func(item InboxItem, ts string) InboxItem {
		item.EscalatedAt = ts
		return item
	}
	ackedWithReason := deadlineTestItem(InboxStateAcked)
	// An acked event that carries a reason (a hand-edited line) is not a
	// notification.
	ackedWithReason.Events = append(ackedWithReason.Events, InboxEvent{TS: inboxTestNow, ID: "e1", Event: InboxEventAcked, Reason: resolve})
	cases := []struct {
		name string
		item InboxItem
		at   time.Duration
		want string
	}{
		{"open, not notified, before the ack deadline", deadlineTestItem(InboxStateOpen), 9 * time.Minute, ""},
		{"open, not notified, exactly at the ack deadline", deadlineTestItem(InboxStateOpen), 10 * time.Minute, ""},
		{"open, not notified, just past the ack deadline", deadlineTestItem(InboxStateOpen), 10*time.Minute + time.Second, ack},
		{"open, not notified, a clock behind escalated_at", deadlineTestItem(InboxStateOpen), -time.Hour, ""},
		{"acked, not notified, past the ack deadline", deadlineTestItem(InboxStateAcked), 30 * time.Minute, ""},
		{"acked, not notified, exactly at the resolve deadline", deadlineTestItem(InboxStateAcked), 60 * time.Minute, ""},
		{"acked, not notified, past the resolve deadline", deadlineTestItem(InboxStateAcked), 61 * time.Minute, resolve},
		{"notified by escalate, past the ack deadline", deadlineTestItem(InboxStateOpen, InboxReasonDirectorDisabled), 30 * time.Minute, ""},
		{"notified by escalate, exactly at the resolve deadline", deadlineTestItem(InboxStateOpen, InboxReasonDirectorDisabled), 60 * time.Minute, ""},
		{"notified by escalate, past the resolve deadline", deadlineTestItem(InboxStateOpen, InboxReasonDirectorDisabled), 61 * time.Minute, resolve},
		{"notified by escalate with no watch, past the ack deadline", deadlineTestItem(InboxStateOpen, InboxReasonNoWatch), 30 * time.Minute, ""},
		{"sent by inbox notify, past the ack deadline", deadlineTestItem(InboxStateOpen, InboxReasonManualNotify), 30 * time.Minute, ""},
		{"sent by inbox notify, past the resolve deadline", deadlineTestItem(InboxStateOpen, InboxReasonManualNotify), 61 * time.Minute, resolve},
		{"notified before stage 8a (inbox_no_director), past the ack deadline", deadlineTestItem(InboxStateOpen, "inbox_no_director"), 30 * time.Minute, ""},
		{"sent at the ack deadline, later in the ack window", deadlineTestItem(InboxStateOpen, ack), 30 * time.Minute, ""},
		{"sent at the ack deadline, past the resolve deadline", deadlineTestItem(InboxStateOpen, ack), 61 * time.Minute, resolve},
		{"open, sent at the resolve deadline", deadlineTestItem(InboxStateOpen, ack, resolve), 3 * time.Hour, ""},
		{"acked, sent at the resolve deadline", deadlineTestItem(InboxStateAcked, InboxReasonDirectorDisabled, resolve), 3 * time.Hour, ""},
		{"open, not notified, both deadlines passed: resolve only", deadlineTestItem(InboxStateOpen), 61 * time.Minute, resolve},
		{"open, not notified, exactly at the resolve deadline: ack only", deadlineTestItem(InboxStateOpen), 60 * time.Minute, ack},
		{"an acked event with a reason is not a notification", ackedWithReason, 61 * time.Minute, resolve},
		{"resolved straight from open, past the ack deadline", deadlineTestItem(InboxStateResolved), 30 * time.Minute, ""},
		{"resolved straight from open, past the resolve deadline", deadlineTestItem(InboxStateResolved), 3 * time.Hour, ""},
		{"resolved after escalate's notification", deadlineTestItem(InboxStateResolved, InboxReasonDirectorDisabled), 3 * time.Hour, ""},
		{"escalated_at with an offset is the same instant", withEscalatedAt(deadlineTestItem(InboxStateOpen), "2026-10-10T21:00:00+09:00"), 10*time.Minute + time.Second, ack},
		{"resolved, escalated_at does not parse", withEscalatedAt(deadlineTestItem(InboxStateResolved), "yesterday"), 3 * time.Hour, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// AC11: the result does not change with the item's delivery,
			// "" being an escalated line written before stage 8a.
			for _, delivery := range []string{"", InboxDeliveryNow, InboxDeliveryDeadline} {
				item := tc.item
				item.Delivery = delivery
				item.Events = slices.Clone(tc.item.Events)
				item.Events[0].Delivery = delivery
				got, err := InboxDeadlineDue(item, deadlineT0.Add(tc.at), 10, 60)
				if err != nil || got != tc.want {
					t.Errorf("delivery %q: InboxDeadlineDue = (%q, %v); want (%q, nil)", delivery, got, err, tc.want)
				}
			}
		})
	}

	for _, ts := range []string{"", "yesterday", "2026-10-10 12:00:00", "2026-10-10T12:00:00"} {
		for _, state := range []string{InboxStateOpen, InboxStateAcked} {
			t.Run("escalated_at "+ts+" "+state, func(t *testing.T) {
				got, err := InboxDeadlineDue(withEscalatedAt(deadlineTestItem(state), ts), deadlineT0.Add(3*time.Hour), 10, 60)
				if err == nil || got != "" {
					t.Fatalf("InboxDeadlineDue = (%q, %v); want an error and no reason", got, err)
				}
				for _, sub := range []string{"e1", "not an RFC3339 time"} {
					if !strings.Contains(err.Error(), sub) {
						t.Errorf("error %q does not contain %q", err, sub)
					}
				}
			})
		}
	}
}

// AC3: an item past a deadline reaches the human path once, with the
// deadline as the reason, and its state does not change.
func TestSweepInboxDeadlines_SendsAnItemPastItsDeadlineToTheHuman(t *testing.T) {
	const secret = "the body must not reach the notification"
	cases := []struct {
		name          string
		ack           bool
		at            time.Duration
		hint          string
		notifyErr     error
		wantReason    string
		wantState     string
		wantBanner    string
		wantDesktop   string
		wantOsascript string
	}{
		{
			name: "ack deadline of an open item", at: 11 * time.Minute,
			wantReason: InboxReasonAckDeadline, wantState: InboxStateOpen,
			wantBanner:    "ORG INBOX DEADLINE: org=org-a item=e1 type=QUESTION -- not acked within 10m (inbox_ack_deadline); read it with: ralph org inbox show e1\n",
			wantDesktop:   "org org-a: e1 (QUESTION) not acked within 10 min",
			wantOsascript: "ok",
		},
		{
			name: "resolve deadline of an acked item", ack: true, at: 61 * time.Minute,
			hint: "/repo/.harness/state/org", notifyErr: errors.New("exit status 1: Not authorized to send Apple events"),
			wantReason: InboxReasonResolveDeadline, wantState: InboxStateAcked,
			wantBanner:    "ORG INBOX DEADLINE: org=org-a item=e1 type=QUESTION -- not resolved within 60m (inbox_resolve_deadline); read it with: ralph org inbox show e1 --state-dir '/repo/.harness/state/org'\n",
			wantDesktop:   "org org-a: e1 (QUESTION) not resolved within 60 min",
			wantOsascript: "failed: exit status 1: Not authorized to send Apple events",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDeadlineFixture(t)
			f.o.InboxHintStateDir = tc.hint
			f.notifyErr = tc.notifyErr
			if _, err := f.o.Inbox.Escalate(InboxEscalation{OrgID: "org-a", Type: "QUESTION", Body: "TYPE: QUESTION\n\n" + secret}); err != nil {
				t.Fatal(err)
			}
			wantKinds := []string{InboxEventEscalated}
			if tc.ack {
				if _, err := f.o.Inbox.Ack("e1"); err != nil {
					t.Fatal(err)
				}
				wantKinds = append(wantKinds, InboxEventAcked)
			}
			wantKinds = append(wantKinds, InboxEventNotified)

			assertSweep(t, "sweep", f.sweep(tc.at, &InboxDeadlineMemory{}), []string{"e1 " + tc.wantReason}, nil, nil)

			lines := escalationLines(t, f.o.EscalationsPath)
			if len(lines) != 1 {
				t.Fatalf("escalations.jsonl has %d lines; want 1: %v", len(lines), lines)
			}
			want := map[string]any{"ts": inboxTestNow, "org_id": "org-a", "inbox_id": "e1", "subject": "leader", "reason": tc.wantReason}
			if len(lines[0]) != len(want) {
				t.Errorf("escalations line = %v; want exactly %v", lines[0], want)
			}
			for k, v := range want {
				if lines[0][k] != v {
					t.Errorf("escalations line %s = %v; want %v", k, lines[0][k], v)
				}
			}
			if got := f.banner.String(); got != tc.wantBanner {
				t.Errorf("banner = %q; want %q", got, tc.wantBanner)
			}
			if calls := f.notifications(); !slices.Equal(calls, []string{tc.wantDesktop}) {
				t.Errorf("desktop notifications = %q; want [%q]", calls, tc.wantDesktop)
			}
			if strings.Contains(f.banner.String()+strings.Join(f.notifications(), ""), secret) {
				t.Errorf("the message body reached the banner or the notification")
			}

			inbox := mustReadInbox(t, f.o.Inbox)
			if got := inboxItemIDs(inbox); !slices.Equal(got, []string{"e1"}) {
				t.Fatalf("items = %v; a deadline send must not add an item", got)
			}
			item := inbox.Items[0]
			if item.State != tc.wantState || !item.Notified {
				t.Errorf("e1 = %+v; want state %s, notified", item, tc.wantState)
			}
			if got := inboxEventKinds(item); !slices.Equal(got, wantKinds) {
				t.Fatalf("e1 events = %v; want %v", got, wantKinds)
			}
			if ev := item.Events[len(item.Events)-1]; ev.Reason != tc.wantReason || ev.Osascript != tc.wantOsascript {
				t.Errorf("notified event = %+v; want reason %s, osascript %q", ev, tc.wantReason, tc.wantOsascript)
			}
		})
	}
}

// A hand-edited inbox.jsonl line can put anything in org_id and type: the
// deadline banner escapes them, and the desktop notification gets them only
// when escalate would have accepted them.
func TestSweepInboxDeadlines_HandEditedLineEscapesTheBannerAndKeepsTheNotificationNeutral(t *testing.T) {
	f := newDeadlineFixture(t)
	line, err := json.Marshal(InboxEvent{
		TS: inboxTestNow, ID: "e1", Event: InboxEventEscalated,
		OrgID: "org-a\x1b[2J\"; beep", Type: "BLOCKED\x07\"", TaskID: "org-a", Body: "TYPE: BLOCKED\nTASK_ID: org-a\n\nstuck",
	})
	if err != nil {
		t.Fatal(err)
	}
	writeInboxFile(t, f.o.Inbox, string(line)+"\n")

	assertSweep(t, "sweep", f.sweep(11*time.Minute, nil), []string{"e1 " + InboxReasonAckDeadline}, nil, nil)
	banner := f.banner.String()
	for _, sub := range []string{`org=org-a\x1b[2J"; beep`, `type=BLOCKED\a"`, "not acked within 10m"} {
		if !strings.Contains(banner, sub) {
			t.Errorf("banner %q does not contain %q", banner, sub)
		}
	}
	for _, r := range strings.TrimSuffix(banner, "\n") {
		if r < 0x20 || r == 0x7f {
			t.Errorf("banner %q carries the raw control character %U", banner, r)
		}
	}
	if calls, want := f.notifications(), "org <invalid>: e1 (<invalid>) not acked within 10 min"; !slices.Equal(calls, []string{want}) {
		t.Errorf("desktop notifications = %q; want [%q]", calls, want)
	}
}

// Resolved items are never swept, and the sends come in item ID order.
func TestSweepInboxDeadlines_SkipsResolvedItemsAndKeepsIDOrder(t *testing.T) {
	f := newDeadlineFixture(t)
	for _, orgID := range []string{"org-a", "org-b", "org-c", "org-d"} {
		mustEscalate(t, f.o.Inbox, orgID)
	}
	if err := f.o.Inbox.Resolve("e1", "docs/reports/x.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.o.Inbox.Ack("e3"); err != nil {
		t.Fatal(err)
	}

	assertSweep(t, "past the ack deadline", f.sweep(11*time.Minute, &InboxDeadlineMemory{}),
		[]string{"e2 " + InboxReasonAckDeadline, "e4 " + InboxReasonAckDeadline}, nil, nil)
	assertSweep(t, "past the resolve deadline", f.sweep(61*time.Minute, &InboxDeadlineMemory{}),
		[]string{"e2 " + InboxReasonResolveDeadline, "e3 " + InboxReasonResolveDeadline, "e4 " + InboxReasonResolveDeadline}, nil, nil)
	if got := escalationReasons(t, f.o.EscalationsPath); slices.ContainsFunc(got, func(s string) bool { return strings.HasPrefix(s, "e1 ") }) {
		t.Errorf("escalations.jsonl = %v; the resolved e1 was sent", got)
	}
}

// AC4: a send that cannot write escalations.jsonl writes no notified
// event, and the same process tries it again only once ack_minutes have
// passed since the attempt. A fresh memory (a restarted watch) tries at once.
func TestSweepInboxDeadlines_FailedSendIsRetriedOncePerAckMinutes(t *testing.T) {
	f := newDeadlineFixture(t)
	mustEscalate(t, f.o.Inbox, "org-a")
	// A directory where escalations.jsonl should be: the line cannot be
	// written.
	if err := os.Mkdir(f.o.EscalationsPath, 0o755); err != nil {
		t.Fatal(err)
	}
	ack := []string{"e1 " + InboxReasonAckDeadline}
	mem := &InboxDeadlineMemory{}

	assertSweep(t, "first attempt", f.sweep(11*time.Minute, mem), nil, ack, nil)
	if n := len(f.notifications()); n != 1 {
		t.Fatalf("desktop notified %d times after the first attempt; want 1", n)
	}
	if item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1"); item.Notified || item.State != InboxStateOpen {
		t.Fatalf("e1 = %+v; want open and not notified after a failed send", item)
	}

	assertSweep(t, "within ack_minutes", f.sweep(21*time.Minute-time.Second, mem), nil, nil, ack)
	if n := len(f.notifications()); n != 1 {
		t.Errorf("desktop notified %d times; a held-back send must not ring", n)
	}

	assertSweep(t, "ack_minutes after the attempt", f.sweep(21*time.Minute, mem), nil, ack, nil)
	if n := len(f.notifications()); n != 2 {
		t.Errorf("desktop notified %d times; want the retry to ring (2)", n)
	}

	assertSweep(t, "a fresh memory", f.sweep(21*time.Minute+30*time.Second, &InboxDeadlineMemory{}), nil, ack, nil)
	assertSweep(t, "nil memory", f.sweep(21*time.Minute+30*time.Second, nil), nil, ack, nil)
	assertSweep(t, "the first memory still holds it", f.sweep(21*time.Minute+30*time.Second, mem), nil, nil, ack)
	if n := len(f.notifications()); n != 4 {
		t.Errorf("desktop notified %d times; want 4 (the first attempt, the retry, the fresh memory's, and nil memory's)", n)
	}
	if inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1").Notified {
		t.Fatalf("e1 is notified although no escalations.jsonl line was written")
	}

	// The ledger is writable again: the memory still holds the send back
	// until ack_minutes have passed, then the retry completes it.
	if err := os.Remove(f.o.EscalationsPath); err != nil {
		t.Fatal(err)
	}
	assertSweep(t, "writable, still within ack_minutes", f.sweep(30*time.Minute, mem), nil, nil, ack)
	assertSweep(t, "writable, ack_minutes after the last attempt", f.sweep(31*time.Minute+30*time.Second, mem), ack, nil, nil)
	assertSweep(t, "after the send", f.sweep(32*time.Minute, mem), nil, nil, nil)
	if got, want := escalationReasons(t, f.o.EscalationsPath), ack; !slices.Equal(got, want) {
		t.Errorf("escalations.jsonl = %v; want %v", got, want)
	}
	item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1")
	if got := inboxEventKinds(item); !slices.Equal(got, []string{InboxEventEscalated, InboxEventNotified}) {
		t.Errorf("e1 events = %v; want escalated, notified", got)
	}
}

// AC4: a send whose notified event cannot be written fails the same way and
// is held back the same way, although its escalations.jsonl line is there.
func TestSweepInboxDeadlines_NotifiedUnwritableIsHeldBack(t *testing.T) {
	f := newDeadlineFixture(t)
	mustEscalate(t, f.o.Inbox, "org-a")
	inboxPath := f.o.Inbox.Path()
	var saved []byte
	f.onNotify = func() {
		// Swap inbox.jsonl for a directory once, after the sweep's read and
		// before the notified event.
		f.onNotify = nil
		raw, err := os.ReadFile(inboxPath)
		if err != nil {
			t.Errorf("read inbox: %v", err)
			return
		}
		saved = raw
		if err := os.Remove(inboxPath); err != nil {
			t.Errorf("remove inbox: %v", err)
		}
		if err := os.Mkdir(inboxPath, 0o755); err != nil {
			t.Errorf("mkdir inbox: %v", err)
		}
	}
	ack := []string{"e1 " + InboxReasonAckDeadline}
	mem := &InboxDeadlineMemory{}
	assertSweep(t, "notified unwritable", f.sweep(11*time.Minute, mem), nil, ack, nil)

	if err := os.Remove(inboxPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inboxPath, saved, 0o644); err != nil {
		t.Fatal(err)
	}
	if item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1"); item.Notified {
		t.Fatalf("e1 = %+v; want not notified", item)
	}
	assertSweep(t, "within ack_minutes", f.sweep(12*time.Minute, mem), nil, nil, ack)
	assertSweep(t, "a fresh memory", f.sweep(12*time.Minute, &InboxDeadlineMemory{}), ack, nil, nil)
	if got, want := escalationReasons(t, f.o.EscalationsPath), []string{"e1 " + InboxReasonAckDeadline, "e1 " + InboxReasonAckDeadline}; !slices.Equal(got, want) {
		t.Errorf("escalations.jsonl = %v; want %v (the failed send's line and the retry's)", got, want)
	}
}

// AC4: the memory holds back the same item and reason only: an item whose
// ack deadline send failed is tried at once when its resolve deadline
// passes.
func TestSweepInboxDeadlines_HoldBackIsPerReason(t *testing.T) {
	f := newDeadlineFixture(t)
	mustEscalate(t, f.o.Inbox, "org-a")
	if err := os.Mkdir(f.o.EscalationsPath, 0o755); err != nil {
		t.Fatal(err)
	}
	mem := &InboxDeadlineMemory{}
	assertSweep(t, "ack deadline", f.sweep(55*time.Minute, mem), nil, []string{"e1 " + InboxReasonAckDeadline}, nil)
	assertSweep(t, "resolve deadline 6 minutes later", f.sweep(61*time.Minute, mem), nil, []string{"e1 " + InboxReasonResolveDeadline}, nil)
	assertSweep(t, "resolve deadline again", f.sweep(62*time.Minute, mem), nil, nil, []string{"e1 " + InboxReasonResolveDeadline})
	if n := len(f.notifications()); n != 2 {
		t.Errorf("desktop notified %d times; want 2", n)
	}
}

// AC5: a watch started on a ledger with an item past its deadlines sends it
// in its first sweep, and a restarted watch does not send what the notified
// events already record.
func TestSweepInboxDeadlines_ARestartedWatchSendsWhatIsDueAndNothingTwice(t *testing.T) {
	f := newDeadlineFixture(t)
	escalateInboxAt(t, f.o.Inbox, deadlineT0.Add(-3*time.Hour), "org-a") // e1, long ago, never notified
	mustEscalate(t, f.o.Inbox, "org-b")                                  // e2, at deadlineT0

	assertSweep(t, "first watch", f.sweep(11*time.Minute, &InboxDeadlineMemory{}),
		[]string{"e1 " + InboxReasonResolveDeadline, "e2 " + InboxReasonAckDeadline}, nil, nil)
	assertSweep(t, "restarted watch", f.sweep(12*time.Minute, &InboxDeadlineMemory{}), nil, nil, nil)
	assertSweep(t, "restarted past e2's resolve deadline", f.sweep(61*time.Minute, &InboxDeadlineMemory{}),
		[]string{"e2 " + InboxReasonResolveDeadline}, nil, nil)
	assertSweep(t, "restarted again", f.sweep(3*time.Hour, &InboxDeadlineMemory{}), nil, nil, nil)

	want := []string{"e1 " + InboxReasonResolveDeadline, "e2 " + InboxReasonAckDeadline, "e2 " + InboxReasonResolveDeadline}
	if got := escalationReasons(t, f.o.EscalationsPath); !slices.Equal(got, want) {
		t.Errorf("escalations.jsonl = %v; want %v", got, want)
	}
	if n := len(f.notifications()); n != 3 {
		t.Errorf("desktop notified %d times; want 3", n)
	}
	e1 := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1")
	if got := inboxEventKinds(e1); !slices.Equal(got, []string{InboxEventEscalated, InboxEventNotified}) {
		t.Errorf("e1 events = %v; want one notified (the resolve deadline only)", got)
	}
}

// AC6: escalate sends the item at once (the director switch is off, so
// inbox_director_disabled); the ack deadline does not send it again, and
// the resolve deadline sends it once more while it is not resolved.
func TestSweepInboxDeadlines_AnEscalatedItemIsSentAgainOnlyAtTheResolveDeadline(t *testing.T) {
	f := newDeadlineFixture(t)
	res := f.escalate("org-a", "TYPE: RESULT\nTASK_ID: org-a\n\nPR https://example.invalid/pull/1")
	if res.Err != nil || !res.Notified {
		t.Fatalf("Escalate = %+v; want notified", res)
	}
	mem := &InboxDeadlineMemory{}
	for _, at := range []time.Duration{11 * time.Minute, 30 * time.Minute, 60 * time.Minute} {
		assertSweep(t, "before the resolve deadline", f.sweep(at, mem), nil, nil, nil)
	}
	if n := len(f.notifications()); n != 1 {
		t.Errorf("desktop notified %d times before the resolve deadline; want escalate's 1", n)
	}

	assertSweep(t, "past the resolve deadline", f.sweep(61*time.Minute, mem), []string{"e1 " + InboxReasonResolveDeadline}, nil, nil)
	assertSweep(t, "next sweep", f.sweep(62*time.Minute, mem), nil, nil, nil)
	assertSweep(t, "a restarted watch", f.sweep(3*time.Hour, &InboxDeadlineMemory{}), nil, nil, nil)

	calls := f.notifications()
	if want := "org org-a: e1 (RESULT) not resolved within 60 min"; len(calls) != 2 || calls[1] != want {
		t.Errorf("desktop notifications = %q; want escalate's and %q", calls, want)
	}
	if got, want := escalationReasons(t, f.o.EscalationsPath), []string{"e1 " + InboxReasonDirectorDisabled, "e1 " + InboxReasonResolveDeadline}; !slices.Equal(got, want) {
		t.Errorf("escalations.jsonl = %v; want %v", got, want)
	}
	item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1")
	if item.State != InboxStateOpen {
		t.Errorf("e1 state = %s; want open", item.State)
	}
	var reasons []string
	for _, ev := range item.Events {
		if ev.Event == InboxEventNotified {
			reasons = append(reasons, ev.Reason)
		}
	}
	if want := []string{InboxReasonDirectorDisabled, InboxReasonResolveDeadline}; !slices.Equal(reasons, want) {
		t.Errorf("notified reasons = %v; want %v", reasons, want)
	}

	// Resolved now: nothing more.
	if err := f.o.Inbox.Resolve("e1", "https://example.invalid/pull/1"); err != nil {
		t.Fatal(err)
	}
	assertSweep(t, "resolved", f.sweep(5*time.Hour, &InboxDeadlineMemory{}), nil, nil, nil)
}

// escalateDeferred escalates a QUESTION from org-a while a watch accepts
// work, so escalate only records it (delivery deadline), and resets the
// banner.
func escalateDeferred(t *testing.T, f *escalateFixture) {
	t.Helper()
	f.useWatch(acceptingWatch())
	if res := f.escalate("org-a", "TYPE: QUESTION\n\nplan A or plan B?"); res.Err != nil || res.ID != "e1" || !res.Deferred || res.Notified {
		t.Fatalf("Escalate = %+v; want e1 deferred, not notified", res)
	}
	f.banner.Reset()
}

// AC3: an item escalate left to the director reaches the human only by the
// deadlines: the ack deadline sends it with inbox_ack_deadline once
// ack_minutes have passed and not before, and the resolve deadline once
// more while it is not resolved.
func TestSweepInboxDeadlines_ADeferredItemIsSentAtTheAckDeadline(t *testing.T) {
	f := newDeadlineFixture(t)
	escalateDeferred(t, f)
	mem := &InboxDeadlineMemory{}
	for _, at := range []time.Duration{0, time.Minute, 10 * time.Minute} {
		assertSweep(t, fmt.Sprintf("%v after the escalate", at), f.sweep(at, mem), nil, nil, nil)
	}
	if calls := f.notifications(); len(calls) != 0 || fileOrAbsent(t, f.o.EscalationsPath) != "<absent>" {
		t.Fatalf("the deferred item reached the human before its ack deadline: notifications %q", calls)
	}

	assertSweep(t, "past the ack deadline", f.sweep(10*time.Minute+time.Second, mem), []string{"e1 " + InboxReasonAckDeadline}, nil, nil)
	assertSweep(t, "next sweep", f.sweep(11*time.Minute, mem), nil, nil, nil)
	assertSweep(t, "a restarted watch", f.sweep(30*time.Minute, &InboxDeadlineMemory{}), nil, nil, nil)
	if calls, want := f.notifications(), "org org-a: e1 (QUESTION) not acked within 10 min"; !slices.Equal(calls, []string{want}) {
		t.Errorf("desktop notifications = %q; want [%q]", calls, want)
	}
	item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1")
	if item.Delivery != InboxDeliveryDeadline || !item.Notified || item.State != InboxStateOpen {
		t.Errorf("e1 = %+v; want open, delivery deadline, notified", item)
	}

	assertSweep(t, "past the resolve deadline", f.sweep(61*time.Minute, mem), []string{"e1 " + InboxReasonResolveDeadline}, nil, nil)
	if got, want := escalationReasons(t, f.o.EscalationsPath), []string{"e1 " + InboxReasonAckDeadline, "e1 " + InboxReasonResolveDeadline}; !slices.Equal(got, want) {
		t.Errorf("escalations.jsonl = %v; want %v", got, want)
	}
}

// A deferred item the director acks gets no ack deadline, only the resolve
// deadline; one sent with inbox notify gets no ack deadline either.
func TestSweepInboxDeadlines_ADeferredItemAckedOrNotifiedByHandSkipsTheAckDeadline(t *testing.T) {
	t.Run("acked", func(t *testing.T) {
		f := newDeadlineFixture(t)
		escalateDeferred(t, f)
		if _, err := f.o.Inbox.Ack("e1"); err != nil {
			t.Fatal(err)
		}
		assertSweep(t, "past the ack deadline", f.sweep(30*time.Minute, &InboxDeadlineMemory{}), nil, nil, nil)
		assertSweep(t, "past the resolve deadline", f.sweep(61*time.Minute, &InboxDeadlineMemory{}), []string{"e1 " + InboxReasonResolveDeadline}, nil, nil)
	})
	t.Run("inbox notify", func(t *testing.T) {
		f := newDeadlineFixture(t)
		escalateDeferred(t, f)
		if err := f.o.NotifyInboxItem("e1", &f.banner); err != nil {
			t.Fatal(err)
		}
		assertSweep(t, "past the ack deadline", f.sweep(30*time.Minute, &InboxDeadlineMemory{}), nil, nil, nil)
		assertSweep(t, "past the resolve deadline", f.sweep(61*time.Minute, &InboxDeadlineMemory{}), []string{"e1 " + InboxReasonResolveDeadline}, nil, nil)
		if got, want := escalationReasons(t, f.o.EscalationsPath), []string{"e1 " + InboxReasonManualNotify, "e1 " + InboxReasonResolveDeadline}; !slices.Equal(got, want) {
			t.Errorf("escalations.jsonl = %v; want %v", got, want)
		}
	})
}

// Plan 2026-10-18-org-director-wake, D10 and AC7: a wake does not stop the
// ack deadline. An item the director was woken for is due at the same
// times, for the same reasons, as the same item not woken for, and the
// sweep sends a woken deferred item to the human at its ack deadline and
// not before.
func TestInboxDeadlines_AWokenItemComesDueAsOneNotWoken(t *testing.T) {
	t.Run("InboxDeadlineDue", func(t *testing.T) {
		for _, state := range []string{InboxStateOpen, InboxStateAcked, InboxStateResolved} {
			for _, notified := range [][]string{nil, {InboxReasonDirectorDisabled}, {InboxReasonAckDeadline}} {
				for _, at := range []time.Duration{9 * time.Minute, 10 * time.Minute, 10*time.Minute + time.Second, 60 * time.Minute, 61 * time.Minute} {
					plain := deadlineTestItem(state, notified...)
					woken := deadlineTestItem(state, notified...)
					woken.Events = append(woken.Events, InboxEvent{TS: inboxTestNow, ID: "e1", Event: InboxEventWoken, DirectorID: "d1", Via: InboxWokenViaWakeCmd})
					woken.Woken, woken.WokenAt = true, inboxTestNow
					want, wantErr := InboxDeadlineDue(plain, deadlineT0.Add(at), 10, 60)
					got, err := InboxDeadlineDue(woken, deadlineT0.Add(at), 10, 60)
					if got != want || err != nil || wantErr != nil {
						t.Errorf("%s, notified %v, at %v: woken = (%q, %v); not woken = (%q, %v)", state, notified, at, got, err, want, wantErr)
					}
				}
			}
		}
		woken := deadlineTestItem(InboxStateOpen)
		woken.Woken, woken.WokenAt = true, inboxTestNow
		if got, err := InboxDeadlineDue(woken, deadlineT0.Add(10*time.Minute+time.Second), 10, 60); got != InboxReasonAckDeadline || err != nil {
			t.Errorf("an open woken item just past the ack deadline: InboxDeadlineDue = (%q, %v); want (%q, nil)", got, err, InboxReasonAckDeadline)
		}
	})
	t.Run("sweep", func(t *testing.T) {
		f := newDeadlineFixture(t)
		escalateDeferred(t, f)
		if err := f.o.Inbox.AppendWoken([]string{"e1"}, "d1", InboxWokenViaHerdr); err != nil {
			t.Fatalf("AppendWoken: %v", err)
		}
		if item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1"); !item.Woken || item.Notified || item.State != InboxStateOpen {
			t.Fatalf("e1 = %+v; want open, woken, not notified", item)
		}
		mem := &InboxDeadlineMemory{}
		assertSweep(t, "exactly at the ack deadline", f.sweep(10*time.Minute, mem), nil, nil, nil)
		assertSweep(t, "past the ack deadline", f.sweep(10*time.Minute+time.Second, mem), []string{"e1 " + InboxReasonAckDeadline}, nil, nil)
		if calls, want := f.notifications(), "org org-a: e1 (QUESTION) not acked within 10 min"; !slices.Equal(calls, []string{want}) {
			t.Errorf("desktop notifications = %q; want [%q]", calls, want)
		}
		if got, want := escalationReasons(t, f.o.EscalationsPath), []string{"e1 " + InboxReasonAckDeadline}; !slices.Equal(got, want) {
			t.Errorf("escalations.jsonl = %v; want %v", got, want)
		}
		item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1")
		if got, want := inboxEventKinds(item), []string{InboxEventEscalated, InboxEventWoken, InboxEventNotified}; !slices.Equal(got, want) {
			t.Errorf("e1 events = %v; want %v", got, want)
		}
	})
}

// An item whose escalated_at does not parse is skipped, with one warning
// per memory; the other items are swept as usual.
func TestSweepInboxDeadlines_WarnsOnceAboutAnEscalatedAtThatDoesNotParse(t *testing.T) {
	f := newDeadlineFixture(t)
	line, err := json.Marshal(InboxEvent{TS: "yesterday noon", ID: "e1", Event: InboxEventEscalated, OrgID: "org-a", Type: "QUESTION", Body: "TYPE: QUESTION\n\nwhich?"})
	if err != nil {
		t.Fatal(err)
	}
	writeInboxFile(t, f.o.Inbox, string(line)+"\n")
	mustEscalate(t, f.o.Inbox, "org-b") // e2

	const warning = `org: inbox deadlines: e1 has an escalated_at that is not an RFC3339 time ("yesterday noon"); its deadlines are skipped` + "\n"
	mem := &InboxDeadlineMemory{}
	first := f.sweep(11*time.Minute, mem)
	assertSweep(t, "first sweep", first, []string{"e2 " + InboxReasonAckDeadline}, nil, nil)
	second := f.sweep(12*time.Minute, mem)
	assertSweep(t, "second sweep", second, nil, nil, nil)
	for _, s := range []InboxDeadlineSweep{first, second} {
		if !slices.Equal(s.InvalidEscalatedAt, []string{"e1"}) {
			t.Errorf("InvalidEscalatedAt = %v; want [e1]", s.InvalidEscalatedAt)
		}
	}
	if n := strings.Count(f.banner.String(), warning); n != 1 {
		t.Errorf("banner has the warning %d times; want once:\n%s", n, f.banner.String())
	}
	f.sweep(13*time.Minute, &InboxDeadlineMemory{})
	if n := strings.Count(f.banner.String(), warning); n != 2 {
		t.Errorf("banner has the warning %d times after a fresh memory; want twice", n)
	}
	if got := escalationReasons(t, f.o.EscalationsPath); !slices.Equal(got, []string{"e2 " + InboxReasonAckDeadline}) {
		t.Errorf("escalations.jsonl = %v; want only e2", got)
	}
}

// A sweep that cannot start returns an error and sends nothing.
func TestSweepInboxDeadlines_ReturnsAnErrorWhenItCannotStart(t *testing.T) {
	t.Run("no inbox store", func(t *testing.T) {
		o := &Org{Config: config.OrgConfig{Director: config.OrgDirectorConfig{AckMinutes: 10, ResolveMinutes: 60}}}
		sweep := o.SweepInboxDeadlines(deadlineT0, &InboxDeadlineMemory{}, &bytes.Buffer{})
		if sweep.Err == nil || !strings.Contains(sweep.Err.Error(), "no inbox store") {
			t.Errorf("sweep error = %v; want no inbox store", sweep.Err)
		}
	})
	for _, d := range []config.OrgDirectorConfig{{AckMinutes: 0, ResolveMinutes: 60}, {AckMinutes: 10, ResolveMinutes: 0}, {AckMinutes: -1, ResolveMinutes: -1}} {
		t.Run(fmt.Sprintf("ack_minutes %d resolve_minutes %d", d.AckMinutes, d.ResolveMinutes), func(t *testing.T) {
			f := newDeadlineFixture(t)
			f.o.Config.Director = d
			mustEscalate(t, f.o.Inbox, "org-a")
			sweep := f.sweep(3*time.Hour, &InboxDeadlineMemory{})
			if sweep.Err == nil || !strings.Contains(sweep.Err.Error(), "must both be >= 1") {
				t.Errorf("sweep error = %v; want the >= 1 refusal", sweep.Err)
			}
			if len(sweep.Sent)+len(sweep.Failed)+len(f.notifications()) != 0 || fileOrAbsent(t, f.o.EscalationsPath) != "<absent>" {
				t.Errorf("a refused sweep sent something: %+v, calls %v", sweep, f.notifications())
			}
		})
	}
	t.Run("the inbox cannot be read", func(t *testing.T) {
		f := newDeadlineFixture(t)
		if err := os.Mkdir(f.o.Inbox.Path(), 0o755); err != nil {
			t.Fatal(err)
		}
		sweep := f.sweep(3*time.Hour, &InboxDeadlineMemory{})
		if sweep.Err == nil || !strings.HasPrefix(sweep.Err.Error(), "org: inbox deadlines: ") {
			t.Errorf("sweep error = %v; want a read failure prefixed org: inbox deadlines:", sweep.Err)
		}
		if len(f.notifications()) != 0 || f.banner.Len() != 0 {
			t.Errorf("a failed read notified: banner %q, calls %v", f.banner.String(), f.notifications())
		}
	})
}
