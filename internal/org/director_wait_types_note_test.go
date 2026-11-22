package org

// Issue #374: two director waits with --types QUESTION,BLOCKED,RESULT timed
// out while the watch's ALERT items were open, with the very text a wait
// with nothing open gives, so nothing showed why they returned nothing. The
// tests below fix the note the timeout of a --types wait now adds (plan
// 2026-11-22-director-wait-types-note, AC1) and that two overlapping director
// waits both return the items the watch escalates while they run (AC2).
// Every ledger is a t.TempDir; no test runs herdr, osascript, or a terminal.

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// AC1: the list of the open items a --types wait did not count. No types
// (every TYPE counts) and no item of another TYPE give "", one item gives
// "<id> <TYPE>", and past inboxWaitOtherTypesShown items the rest are counted.
func TestInboxWaitOtherTypesText(t *testing.T) {
	item := func(id, typ string) InboxItem { return InboxItem{ID: id, Type: typ, State: InboxStateOpen} }
	alerts := func(from, to int) []InboxItem {
		var items []InboxItem
		for n := from; n <= to; n++ {
			items = append(items, item(fmt.Sprintf("e%d", n), protocol.TypeAlert))
		}
		return items
	}
	for _, tc := range []struct {
		name  string
		types []string
		open  []InboxItem
		want  string
	}{
		{"no types", nil, alerts(1, 2), ""},
		{"an empty list", []string{}, alerts(1, 2), ""},
		{"no open item", leaderInboxTypes, nil, ""},
		{"only items of the listed TYPEs", leaderInboxTypes, []InboxItem{item("e1", protocol.TypeQuestion), item("e2", protocol.TypeResult)}, ""},
		{"one item of another TYPE", leaderInboxTypes, []InboxItem{item("e1", protocol.TypeQuestion), item("e2", protocol.TypeAlert)}, "e2 ALERT"},
		{"two items of another TYPE", leaderInboxTypes, alerts(144, 145), "e144 ALERT, e145 ALERT"},
		{"a QUESTION left out by an ALERT wait", []string{protocol.TypeAlert}, []InboxItem{item("e1", protocol.TypeAlert), item("e2", protocol.TypeQuestion)}, "e2 QUESTION"},
		{"as many as are shown", leaderInboxTypes, alerts(1, inboxWaitOtherTypesShown), "e1 ALERT, e2 ALERT, e3 ALERT, e4 ALERT, e5 ALERT"},
		{"more than are shown", leaderInboxTypes, append(alerts(1, 3), append([]InboxItem{item("e4", protocol.TypeBlocked)}, alerts(5, 9)...)...),
			"e1 ALERT, e2 ALERT, e3 ALERT, e5 ALERT, e6 ALERT, and 3 more"},
		{"a TYPE no writer would write", leaderInboxTypes, []InboxItem{item("e1", "TASK")}, "e1 TASK"},
		{"a TYPE with a control character is escaped", leaderInboxTypes, []InboxItem{item("e1", "ALE\x1b[2JRT")}, `e1 ALE\x1b[2JRT`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := inboxWaitOtherTypesText(tc.types, tc.open); got != tc.want {
				t.Errorf("inboxWaitOtherTypesText(%v, %d items) = %q; want %q", tc.types, len(tc.open), got, tc.want)
			}
		})
	}
}

// AC1: the note after the timeout's text names the TYPEs counted, in the
// order given, and the list; with an empty list there is no note.
func TestInboxWaitTimeoutNote(t *testing.T) {
	for _, tc := range []struct {
		name   string
		types  []string
		others string
		want   string
	}{
		{"no types", nil, "", ""},
		{"no item of another TYPE", leaderInboxTypes, "", ""},
		{"the leader's TYPEs", leaderInboxTypes, "e144 ALERT, e145 ALERT",
			" (counted only QUESTION,BLOCKED,RESULT; open items of other TYPEs: e144 ALERT, e145 ALERT)"},
		{"TYPEs in the order given", []string{protocol.TypeResult, protocol.TypeQuestion}, "e1 ALERT, and 2 more",
			" (counted only RESULT,QUESTION; open items of other TYPEs: e1 ALERT, and 2 more)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := inboxWaitTimeoutNote(tc.types, tc.others); got != tc.want {
				t.Errorf("inboxWaitTimeoutNote(%v, %q) = %q; want %q", tc.types, tc.others, got, tc.want)
			}
		})
	}
}

// AC1: the read at the timeout takes the items that are open, of every TYPE
// (an acked item is not open), and an inbox that cannot be read gives no
// note, so the timeout stays a timeout with its text.
func TestInboxWaitOtherTypes_ReadsTheOpenItemsOrNothing(t *testing.T) {
	f := newEscalateFixture(t)
	// e1 and e2 are ALERT items, e3 a QUESTION; e2 is acked.
	escalateAlertItem(t, f.o.Inbox, "org-a/seat-1/stall@1", InboxDeliveryDeadline)
	escalateAlertItem(t, f.o.Inbox, "org-a/seat-2/stall@2", InboxDeliveryDeadline)
	if res := f.escalate("org-b", "TYPE: QUESTION\n\nwhich?"); res.Err != nil {
		t.Fatal(res.Err)
	}
	if _, err := f.o.Inbox.Ack("e2"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		types []string
		want  string
	}{
		{"the leader's TYPEs", leaderInboxTypes, "e1 ALERT"},
		{"ALERT", []string{protocol.TypeAlert}, "e3 QUESTION"},
		{"no types", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := f.o.inboxWaitOtherTypes(tc.types); got != tc.want {
				t.Errorf("inboxWaitOtherTypes(%v) = %q; want %q", tc.types, got, tc.want)
			}
		})
	}

	t.Run("an inbox that cannot be read", func(t *testing.T) {
		g := newEscalateFixture(t)
		// A directory where inbox.jsonl should be: the read fails.
		if err := os.MkdirAll(g.o.Inbox.Path(), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := g.o.Inbox.Read(); err == nil {
			t.Fatal("the inbox read did not fail; the fixture does not test a failed read")
		}
		if got := g.o.inboxWaitOtherTypes(leaderInboxTypes); got != "" {
			t.Errorf("inboxWaitOtherTypes with an unreadable inbox = %q; want \"\"", got)
		}
	})
}

// AC1: WaitInbox with the leader's TYPEs times out while only ALERT items are
// open, and its error names them after the text the /director skill matches.
func TestWaitInbox_TypesTimeoutNamesTheOpenItemsOfOtherTypes(t *testing.T) {
	f := newEscalateFixture(t)
	f.o.InboxPollInterval = 10 * time.Millisecond
	escalateAlertItem(t, f.o.Inbox, "org-a/seat-1/stall@1", InboxDeliveryDeadline)
	escalateAlertItem(t, f.o.Inbox, "org-a/seat-2/stall@2", InboxDeliveryDeadline)

	out := runWaitInboxTypes(t, f.o, leaderInboxTypes, 200*time.Millisecond, 5*time.Second)
	want := "org: wait --inbox: " + inboxWaitTimeoutPhrase + " 200 ms (counted only QUESTION,BLOCKED,RESULT; open items of other TYPEs: e1 ALERT, e2 ALERT)"
	if out.err == nil || out.err.Error() != want {
		t.Fatalf("WaitInbox(%v) = %v, %v; want the error %q", leaderInboxTypes, waitInboxItemIDs(out.items), out.err, want)
	}
	if out.items != nil {
		t.Errorf("a timeout returned items %v", waitInboxItemIDs(out.items))
	}
}

// AC1: the timeout keeps its text when there is nothing to name: no --types
// (an item of any TYPE would have ended the wait), only an acked item of
// another TYPE, or an item of another TYPE resolved while the wait runs,
// before its timeout reads the inbox again.
func TestWaitInbox_TimeoutKeepsItsTextWithNoOpenItemOfAnotherType(t *testing.T) {
	for _, tc := range []struct {
		name  string
		types []string
		seed  func(t *testing.T, f *escalateFixture)
		// during runs while the wait runs, from another goroutine.
		during func(f *escalateFixture) error
	}{
		{"no types, nothing open", nil, func(*testing.T, *escalateFixture) {}, nil},
		{"no types, an acked ALERT", nil, seedAckedAlert, nil},
		{"the leader's TYPEs, an acked ALERT", leaderInboxTypes, seedAckedAlert, nil},
		{"the leader's TYPEs, an ALERT resolved during the wait", leaderInboxTypes,
			func(t *testing.T, f *escalateFixture) {
				escalateAlertItem(t, f.o.Inbox, "org-a/seat-1/stall@1", InboxDeliveryDeadline)
			},
			func(f *escalateFixture) error { return f.o.Inbox.Resolve("e1", "docs/reports/x.md") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newEscalateFixture(t)
			f.o.InboxPollInterval = 10 * time.Millisecond
			tc.seed(t, f)
			if tc.during != nil {
				done := make(chan error, 1)
				go func() {
					time.Sleep(50 * time.Millisecond)
					done <- tc.during(f)
				}()
				t.Cleanup(func() {
					if err := <-done; err != nil {
						t.Errorf("during the wait: %v", err)
					}
				})
			}
			out := runWaitInboxTypes(t, f.o, tc.types, 300*time.Millisecond, 5*time.Second)
			if want := "org: wait --inbox: " + inboxWaitTimeoutPhrase + " 300 ms"; out.err == nil || out.err.Error() != want {
				t.Fatalf("WaitInbox(%v) = %v, %v; want the error %q", tc.types, waitInboxItemIDs(out.items), out.err, want)
			}
		})
	}
}

// seedAckedAlert records the ALERT item e1 and acks it.
func seedAckedAlert(t *testing.T, f *escalateFixture) {
	t.Helper()
	escalateAlertItem(t, f.o.Inbox, "org-a/seat-1/stall@1", InboxDeliveryDeadline)
	if _, err := f.o.Inbox.Ack("e1"); err != nil {
		t.Fatal(err)
	}
}

// AC1: the director's wait with the leader's TYPEs times out while only ALERT
// items are open: its error names them as WaitInbox's does, and the mark's
// detail names them after the TYPEs it counted.
func TestWaitInboxForDirector_TypesTimeoutNamesTheOpenItemsOfOtherTypes(t *testing.T) {
	f := newDirWaitFixture(t)
	f.attach(t)
	holdRunningWatch(t, f.dir)
	escalateAlertItem(t, f.o.Inbox, "org-a/seat-1/stall@1", InboxDeliveryDeadline)
	escalateAlertItem(t, f.o.Inbox, "org-a/seat-2/stall@2", InboxDeliveryDeadline)
	p := f.params("d1")
	p.Timeout, p.Types = 200*time.Millisecond, leaderInboxTypes

	out := f.runDirectorWait(t, p)
	want := directorWaitErrPrefix + inboxWaitTimeoutPhrase + " 200 ms (counted only QUESTION,BLOCKED,RESULT; open items of other TYPEs: e1 ALERT, e2 ALERT)"
	if out.err == nil || out.err.Error() != want {
		t.Fatalf("wait = %v, %v; want the error %q", waitInboxItemIDs(out.items), out.err, want)
	}
	if out.items != nil {
		t.Errorf("a timeout returned items %v", waitInboxItemIDs(out.items))
	}
	m := f.mark(t)
	if wantDetail := "no open inbox item of type QUESTION,BLOCKED,RESULT within 200 ms (open items of other TYPEs: e1 ALERT, e2 ALERT)"; m.End != DirectorWaitEndTimeout || m.Detail != wantDetail {
		t.Errorf("mark = %+v; want end %q with the detail %q", m, DirectorWaitEndTimeout, wantDetail)
	}
}

// AC1: the director's timeout keeps its error and its detail when there is
// nothing to name: no --types with nothing open, and the leader's TYPEs with
// only an acked ALERT.
func TestWaitInboxForDirector_TimeoutKeepsItsTextWithNoOpenItemOfAnotherType(t *testing.T) {
	for _, tc := range []struct {
		name       string
		types      []string
		ackedAlert bool
		wantDetail string
	}{
		{"no types, nothing open", nil, false, "no open inbox item within 200 ms"},
		{"the leader's TYPEs, an acked ALERT", leaderInboxTypes, true, "no open inbox item of type QUESTION,BLOCKED,RESULT within 200 ms"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDirWaitFixture(t)
			f.attach(t)
			holdRunningWatch(t, f.dir)
			if tc.ackedAlert {
				escalateAlertItem(t, f.o.Inbox, "org-a/seat-1/stall@1", InboxDeliveryDeadline)
				if _, err := f.o.Inbox.Ack("e1"); err != nil {
					t.Fatal(err)
				}
			}
			p := f.params("d1")
			p.Timeout, p.Types = 200*time.Millisecond, tc.types
			out := f.runDirectorWait(t, p)
			if want := directorWaitErrPrefix + inboxWaitTimeoutPhrase + " 200 ms"; out.err == nil || out.err.Error() != want {
				t.Fatalf("wait = %v, %v; want the error %q", waitInboxItemIDs(out.items), out.err, want)
			}
			if m := f.mark(t); m.End != DirectorWaitEndTimeout || m.Detail != tc.wantDetail {
				t.Errorf("mark = %+v; want end %q with the detail %q", m, DirectorWaitEndTimeout, tc.wantDetail)
			}
		})
	}
}

// escalateLeaderStallAsTheWatch records one stall ALERT about the leader of
// orgID in the inbox of dir the way the cross-org watch does
// (watchRun.escalateAlert, reason leader_is_anomaly_subject), on an Org of
// its own with the director switch on. accepts is whether that watch accepts
// work: with it the item is left to the director (delivery deadline), and
// without it the item is sent to the human at once (notified), through a
// DesktopNotify stub.
func escalateLeaderStallAsTheWatch(t *testing.T, dir, orgID string, accepts bool, at time.Time) {
	t.Helper()
	inbox := NewInboxStore(dir)
	inbox.Now = func() time.Time { return at }
	o := &Org{
		Inbox: inbox, EscalationsPath: EscalationsPathIn(dir), Now: func() time.Time { return at },
		DesktopNotify: func(context.Context, string) error { return nil },
	}
	o.Config.Director = config.OrgDirectorConfig{Enabled: true, AckMinutes: 10, ResolveMinutes: 60}
	var stderr strings.Builder
	w := &watchRun{org: o, cfg: o.Config, stderr: &stderr, acceptsWork: func() bool { return accepts }}
	alertID := fmt.Sprintf("%s@%d", conditionKey(orgID, LeaderIdentity, condStall), at.UnixNano())
	status := &watchStatusFile{OrgID: orgID, Escalated: map[string]bool{}, PendingAlerts: map[string]*watchPendingAlert{
		alertID: {AlertID: alertID, TS: at.UTC().Format(time.RFC3339), Subject: LeaderIdentity, HistoryLeaderLines: -1},
	}}
	w.escalateAlert(status, alertID, status.PendingAlerts[alertID], at, nil)
	if !status.Escalated[alertID] {
		t.Fatalf("the watch did not escalate %s: %s", alertID, stderr.String())
	}
}

// overlappingDirectorWaits are two director waits for d1 on one ledger, each
// on an Org of its own as two `ralph org wait --inbox --director d1`
// processes are: A started first, and B took director-wait.json over from it.
type overlappingDirectorWaits struct {
	f            *dirWaitFixture
	aDone, bDone <-chan dirWaitOutcome
	aErr, bErr   *lockedBuffer
}

// startOverlappingDirectorWaits starts A (pid 4101), then B (pid 4102) once
// A's mark is there, and returns once A has said that B took the mark over.
// Both have timeout and types, and a watch that accepts work.
func startOverlappingDirectorWaits(t *testing.T, timeout time.Duration, types []string) *overlappingDirectorWaits {
	t.Helper()
	f := newDirWaitFixture(t)
	f.attach(t)
	holdRunningWatch(t, f.dir)
	w := &overlappingDirectorWaits{f: f, aErr: &lockedBuffer{}, bErr: &lockedBuffer{}}
	start := func(pid int, stderr *lockedBuffer) <-chan dirWaitOutcome {
		p := f.params("d1")
		p.PID, p.Stderr, p.Timeout, p.Types = pid, stderr, timeout, types
		o := &Org{Inbox: NewInboxStore(f.dir), InboxPollInterval: 5 * time.Millisecond}
		done := make(chan dirWaitOutcome, 1)
		begin := time.Now()
		go func() {
			items, err := o.WaitInboxForDirector(context.Background(), p)
			done <- dirWaitOutcome{items: items, err: err, elapsed: time.Since(begin)}
		}()
		return done
	}
	w.aDone = start(4101, w.aErr)
	f.waitForMark(t, "A's start", func(m DirectorWaitMark) bool { return m.PID == 4101 })
	w.bDone = start(4102, w.bErr)
	f.waitForMark(t, "B's start", func(m DirectorWaitMark) bool { return m.PID == 4102 })
	waitForStderr(t, w.aErr, "took over")
	return w
}

// escalateTwoLeaderStalls records the leader stall ALERTs of org-a (e1) and
// org-b (e2) as the watch does.
func (w *overlappingDirectorWaits) escalateTwoLeaderStalls(t *testing.T, accepts bool) {
	t.Helper()
	at := time.Date(2026, 11, 21, 19, 6, 28, 0, time.UTC)
	escalateLeaderStallAsTheWatch(t, w.f.dir, "org-a", accepts, at)
	escalateLeaderStallAsTheWatch(t, w.f.dir, "org-b", accepts, at.Add(3*time.Second))
}

// wantOnlyBsMark checks that director-wait.json is B's with end and a
// detail holding detail, and that A said once that B took it over: A, whose
// mark was taken over, wrote nothing more, its end included.
func (w *overlappingDirectorWaits) wantOnlyBsMark(t *testing.T, end, detail string) {
	t.Helper()
	if m := w.f.mark(t); m.PID != 4102 || m.End != end || !strings.Contains(m.Detail, detail) {
		t.Errorf("mark = %+v; want B's (pid 4102) with end %q and a detail holding %q", m, end, detail)
	}
	if s := w.aErr.String(); strings.Count(s, "took over") != 1 {
		t.Errorf("A's stderr = %q; want one line saying B took over", s)
	}
}

// AC2 (issue #374's second expectation): two overlapping director waits, B
// having taken director-wait.json over from A, and then the watch escalates
// two leader stall ALERTs: both waits return the items, the one taken over
// as well as the owner, whether the watch left the items to the director or
// sent them to the human at once (notified changes no state).
func TestWaitInboxForDirector_TwoOverlappingWaitsBothReturnTheWatchsItems(t *testing.T) {
	for _, tc := range []struct {
		name    string
		accepts bool
	}{
		{"left to the director", true},
		{"sent to the human at once", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := startOverlappingDirectorWaits(t, 5*time.Second, nil)
			w.escalateTwoLeaderStalls(t, tc.accepts)
			a, b := awaitDirectorWait(t, w.aDone), awaitDirectorWait(t, w.bDone)
			// Each returns the open items of the read that found the first one,
			// so a wait that read between the two escalations returns e1 only.
			for _, out := range []struct {
				who string
				out dirWaitOutcome
			}{{"A (taken over)", a}, {"B (owner)", b}} {
				ids := waitInboxItemIDs(out.out.items)
				if out.out.err != nil || (!slices.Equal(ids, []string{"e1"}) && !slices.Equal(ids, []string{"e1", "e2"})) {
					t.Errorf("%s = %v, %v; want the items, starting with e1", out.who, ids, out.out.err)
				}
			}
			w.wantOnlyBsMark(t, DirectorWaitEndItems, "open inbox item")
			inbox, err := w.f.o.Inbox.Read()
			if err != nil {
				t.Fatal(err)
			}
			for _, it := range inbox.Items {
				if it.Type != protocol.TypeAlert || it.State != InboxStateOpen || it.Notified == tc.accepts {
					t.Errorf("item %s: type %s, state %s, notified %t; want an open ALERT, notified %t", it.ID, it.Type, it.State, it.Notified, !tc.accepts)
				}
			}
		})
	}
}

// AC1 and AC2 together, the symptoms of issue #374: the same two overlapping
// waits with the leader's TYPEs both time out while the watch's ALERT items
// are open, and both errors name the items; the mark, which only B writes,
// names them in its detail.
func TestWaitInboxForDirector_TwoOverlappingTypesWaitsBothNameTheOpenAlerts(t *testing.T) {
	timeout := time.Second
	w := startOverlappingDirectorWaits(t, timeout, leaderInboxTypes)
	w.escalateTwoLeaderStalls(t, true)
	if len(w.aDone) > 0 || len(w.bDone) > 0 {
		t.Fatalf("a wait ended before the watch recorded its items (A %d, B %d): this machine is too slow for the %v timeout", len(w.aDone), len(w.bDone), timeout)
	}
	a, b := awaitDirectorWait(t, w.aDone), awaitDirectorWait(t, w.bDone)
	want := directorWaitErrPrefix + inboxWaitTimeoutPhrase + " 1000 ms (counted only QUESTION,BLOCKED,RESULT; open items of other TYPEs: e1 ALERT, e2 ALERT)"
	for _, out := range []struct {
		who string
		out dirWaitOutcome
	}{{"A (taken over)", a}, {"B (owner)", b}} {
		if out.out.err == nil || out.out.err.Error() != want || out.out.items != nil {
			t.Errorf("%s = %v, %v; want the error %q", out.who, waitInboxItemIDs(out.out.items), out.out.err, want)
		}
	}
	w.wantOnlyBsMark(t, DirectorWaitEndTimeout, "within 1000 ms (open items of other TYPEs: e1 ALERT, e2 ALERT)")
}
