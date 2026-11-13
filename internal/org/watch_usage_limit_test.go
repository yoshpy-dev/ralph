package org

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// The usage limit lines UsageLimitScreen reads from the four usage limit
// screens of testdata/resume (usage_limit_screen_test.go).
const (
	ulClaudeSessionLine = "⏺ You've hit your session limit · resets 3:45pm"
	ulClaudeWeeklyLine  = "⎿ You've hit your weekly limit · resets 11pm (Asia/Tokyo) /usage-credits to finish what you’re working on."
	ulCodexUsageLine    = "■ You've hit your usage limit. Upgrade to Pro or purchase more credits, or try again at 3:05 PM."
	ulClaude5hWaitLine  = "⚠ Usage limit reached · limit resets 3:50pm Continuing automatically at 3:50pm · Esc to cancel 🤖 Haiku 5.5 | ⚡ low | 🌿 main ⏵⏵ bypass permissions on (shift+tab to cycle)"
)

// usageLimitHerdr is fakeWatchHerdr with the screens of the seats' panes:
// PaneRead answers the screen set for a pane (screens), the error set for
// it instead (readErr), or, for a pane in hang, waits until its ctx is done
// and answers ctx's error. It records each call (reads).
type usageLimitHerdr struct {
	*fakeWatchHerdr

	readMu  sync.Mutex
	screens map[string]string
	readErr map[string]error
	hang    map[string]bool
	reads   []usageLimitRead
}

// usageLimitRead is one PaneRead call: the pane, the lines asked for, and
// whether its ctx had a deadline.
type usageLimitRead struct {
	paneID      string
	lines       int
	hasDeadline bool
}

func (h *usageLimitHerdr) PaneRead(ctx context.Context, paneID string, lines int) (string, error) {
	_, hasDeadline := ctx.Deadline()
	h.readMu.Lock()
	h.reads = append(h.reads, usageLimitRead{paneID: paneID, lines: lines, hasDeadline: hasDeadline})
	screen, err, hang := h.screens[paneID], h.readErr[paneID], h.hang[paneID]
	h.readMu.Unlock()
	if hang {
		<-ctx.Done()
		return "", ctx.Err()
	}
	return screen, err
}

// readsSoFar returns the PaneRead calls so far.
func (h *usageLimitHerdr) readsSoFar() []usageLimitRead {
	h.readMu.Lock()
	defer h.readMu.Unlock()
	return slices.Clone(h.reads)
}

// usageLimitSeat is a seat of org-a: its seat_id and driver.
type usageLimitSeat struct{ id, driver string }

// usageLimitFixture is a watchInboxFixture with stall_minutes 5, a herdr
// whose panes answer screens (usageLimitHerdr), the given seats of org-a in
// the manifest, each in a pane of its own (usageLimitPane), and the clock 10
// minutes past their spawn, so from the second cycle on a seat whose agent
// get answers the same text is stalled unless something else moves.
type usageLimitFixture struct {
	*watchInboxFixture
	uh     *usageLimitHerdr
	a      *fakeWatchAgmsg
	run    *watchRun
	status *watchStatusFile
}

func newUsageLimitFixture(t *testing.T, seats ...usageLimitSeat) *usageLimitFixture {
	t.Helper()
	f := newWatchInboxFixture(t)
	f.o.Config.Watchdog.StallMinutes = 5
	uh := &usageLimitHerdr{fakeWatchHerdr: f.h, screens: map[string]string{}, readErr: map[string]error{}, hang: map[string]bool{}}
	f.o.Herdr = uh
	for _, s := range seats {
		appendUsageLimitSeat(t, f.o, f.clk.Now(), s, usageLimitPane(s.id))
	}
	f.clk.Advance(10 * time.Minute)
	return &usageLimitFixture{watchInboxFixture: f, uh: uh, a: f.o.Agmsg.(*fakeWatchAgmsg), run: f.run(nil), status: &watchStatusFile{OrgID: "org-a"}}
}

// appendUsageLimitSeat records s as spawned in org-a at ts, in paneID (""
// records no pane).
func appendUsageLimitSeat(t *testing.T, o *Org, ts time.Time, s usageLimitSeat, paneID string) {
	t.Helper()
	if err := o.Manifest.Append(ManifestEvent{
		TS: ts.UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: s.id, Event: EventSpawned,
		Role: s.id, Driver: s.driver, Model: "m", PaneID: paneID,
	}); err != nil {
		t.Fatal(err)
	}
}

// usageLimitPane is the pane of seat in a usageLimitFixture.
func usageLimitPane(seat string) string {
	return "pane-" + seat
}

// rerun replaces f's watchRun with one built from the Config as it is now
// (set the director switch first) and acceptsWork.
func (f *usageLimitFixture) rerun(acceptsWork func() bool) {
	f.run = f.watchInboxFixture.run(acceptsWork)
}

// reply makes the herdr agent get of seat answer out from now on.
func (f *usageLimitFixture) reply(seat, out string) {
	f.h.mu.Lock()
	defer f.h.mu.Unlock()
	f.h.AgentGetSeq[herdrAgentName("org-a", seat)] = []string{out}
}

// screen makes seat's pane show file of testdata/resume from now on.
func (f *usageLimitFixture) screen(t *testing.T, seat, file string) {
	t.Helper()
	f.screenText(seat, readResumeScreen(t, file))
}

// screenText makes seat's pane show text from now on.
func (f *usageLimitFixture) screenText(seat, text string) {
	f.uh.readMu.Lock()
	defer f.uh.readMu.Unlock()
	f.uh.screens[usageLimitPane(seat)] = text
}

// cycle runs one cycle of org-a and advances the clock a minute.
func (f *usageLimitFixture) cycle(t *testing.T) {
	t.Helper()
	if err := f.run.evaluateCycle("org-a", f.status); err != nil {
		t.Fatalf("evaluateCycle: %v", err)
	}
	f.clk.Advance(time.Minute)
}

// alerts returns the messages of the watch's ALERTs to the leader whose
// CONDITION is cond.
func (f *usageLimitFixture) alerts(cond string) []string {
	var out []string
	for _, c := range watchdogAlerts(f.a) {
		if strings.Contains(c.message, "\nCONDITION: "+cond+"\n") {
			out = append(out, c.message)
		}
	}
	return out
}

// usageLimitedItems returns the ALERT items of org-a's usage_limited.
func (f *usageLimitFixture) usageLimitedItems(t *testing.T) []InboxItem {
	t.Helper()
	key := conditionKey("org-a", LeaderIdentity, condUsageLimited)
	var out []InboxItem
	for _, item := range mustReadInbox(t, f.o.Inbox).Items {
		if item.Type == protocol.TypeAlert && alertIDKey(item.AlertID) == key {
			out = append(out, item)
		}
	}
	return out
}

// usageLimitedActive reports whether org-a's usage_limited is recorded as
// active.
func (f *usageLimitFixture) usageLimitedActive() bool {
	rec := f.status.Conditions[conditionKey("org-a", LeaderIdentity, condUsageLimited)]
	return rec != nil && rec.Active
}

// activeStalls returns the seats whose stall is recorded as active.
func (f *usageLimitFixture) activeStalls() []string {
	var out []string
	for key, rec := range f.status.Conditions {
		if rec.Active && strings.HasSuffix(key, "/"+condStall) {
			out = append(out, key)
		}
	}
	return out
}

// limitThreeSeats sets the screens of org-a's leader (codex), implementer
// and reviewer (claude) to the three usage limit fixtures.
func (f *usageLimitFixture) limitThreeSeats(t *testing.T) {
	t.Helper()
	f.screen(t, LeaderIdentity, "codex-usage-limit.txt")
	f.screen(t, "implementer", "claude-session-limit.txt")
	f.screen(t, "reviewer", "claude-weekly-limit-under-prompt.txt")
}

// threeSeats are a leader on codex and two claude seats.
var threeSeats = []usageLimitSeat{{LeaderIdentity, "codex"}, {"implementer", "claude"}, {"reviewer", "claude"}}

// assertUsageLimitedBody fails the test unless body is the typed message of
// a usage_limited ALERT item of org-a: the headers of the watch's ALERT
// items with SEAT leader, REASON usage_limit, alertID, ts, and seats as
// SEATS, then the reason that says the org's seats stopped on a usage limit
// and the lines.
func assertUsageLimitedBody(t *testing.T, body, alertID, ts, seats string, lines []string) {
	t.Helper()
	if err := protocol.ValidateText(body, protocol.DefaultMaxBodyChars); err != nil {
		t.Fatalf("body %q fails the protocol check: %v", body, err)
	}
	m, err := protocol.Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"ORG_ID": "org-a", "SEAT": LeaderIdentity, "CONDITION": condUsageLimited, "REASON": usageLimitReason,
		"ALERT_ID": alertID, "ALERT_TS": ts, "SEATS": seats,
	}
	if m.Type != protocol.TypeAlert || m.TaskID != "" || !reflect.DeepEqual(m.Fields, want) {
		t.Errorf("body %q parses as TYPE %q, TASK_ID %q, headers %v; want ALERT with %v", body, m.Type, m.TaskID, m.Fields, want)
	}
	wantText := append([]string{"watchdog: escalated to a person because seats of the org stopped on a usage limit, so the watch did not wait for the leader to act. " +
		"It is a condition of the org, not of the leader that SEAT names: SEATS lists the seats that stopped, " +
		"and each line below gives a seat, its driver, and the usage limit line of its screen, with the reset time it shows"}, lines...)
	if got := strings.Split(m.Body, "\n"); !slices.Equal(got, wantText) {
		t.Errorf("body text lines\n%q\nwant\n%q", got, wantText)
	}
	if strings.Contains(body, "about the leader itself") || strings.Contains(body, "leader_is_anomaly_subject") {
		t.Errorf("body %q reads as an ALERT about the leader itself", body)
	}
}

// AC1, AC2, AC3, AC7, and AC10: three seats of org-a, the leader on codex
// and two claude seats, whose agent get stops changing and whose screens
// show the three usage limits. The first cycle reads no screen (no seat has
// a previous answer yet). In the second, each seat's screen is read once,
// for 60 lines under a deadline; no stall is raised, and the org's
// usage_limited is: one ALERT to the leader that lists the seats, no
// on-demand watcher, and, in the same cycle, one ALERT item with REASON
// usage_limit, SEATS, and a line for each seat with its driver and limit
// line, the reset time included, sent to the human with the director off.
// The status holds the three seats. A third cycle with the same screens
// reads them again and records nothing more.
func TestWatchUsageLimit_SeatsStoppedOnAUsageLimitAreOneItemOfTheOrg(t *testing.T) {
	f := newUsageLimitFixture(t, threeSeats...)
	f.limitThreeSeats(t)
	var triggers []string
	f.run.hooks.OnSemanticTrigger = func(orgID, seatID, condType, _ string) {
		triggers = append(triggers, conditionKey(orgID, seatID, condType))
	}

	f.cycle(t)
	if reads := f.uh.readsSoFar(); len(reads) != 0 {
		t.Fatalf("the first cycle read %+v; want no screen read before a seat has a previous answer", reads)
	}
	ts := f.clk.Now().UTC().Format(time.RFC3339)
	f.cycle(t)

	reads := f.uh.readsSoFar()
	var panes []string
	for _, r := range reads {
		panes = append(panes, r.paneID)
		if r.lines != resumeReadLines || !r.hasDeadline {
			t.Errorf("read %+v; want %d lines under a deadline", r, resumeReadLines)
		}
	}
	slices.Sort(panes)
	if want := []string{"pane-implementer", "pane-leader", "pane-reviewer"}; !slices.Equal(panes, want) {
		t.Errorf("panes read in the second cycle %v; want each seat's once: %v", panes, want)
	}
	if stalls := f.alerts(condStall); len(stalls) != 0 || len(f.activeStalls()) != 0 {
		t.Errorf("stall ALERTs %q, active stalls %v; want none for seats stopped on a usage limit", stalls, f.activeStalls())
	}
	alerts := f.alerts(condUsageLimited)
	if len(alerts) != 1 {
		t.Fatalf("usage_limited ALERTs to the leader %q; want one", alerts)
	}
	if err := protocol.ValidateText(alerts[0], protocol.DefaultMaxBodyChars); err != nil || !strings.Contains(alerts[0], "\nSEATS: implementer, leader, reviewer\n") ||
		!strings.Contains(alerts[0], "a condition of the org, not of the leader") || !strings.Contains(alerts[0], "\nimplementer (claude): "+ulClaudeSessionLine) {
		t.Errorf("ALERT to the leader %q (protocol check %v); want the org's condition with SEATS and each seat's line", alerts[0], err)
	}
	if len(triggers) != 0 {
		t.Errorf("on-demand watcher triggers %q; want none", triggers)
	}

	items := mustReadInbox(t, f.o.Inbox).Items
	if len(items) != 1 {
		t.Fatalf("inbox = %+v; want one ALERT item", items)
	}
	item := items[0]
	key := conditionKey("org-a", LeaderIdentity, condUsageLimited)
	if item.Type != protocol.TypeAlert || item.Delivery != InboxDeliveryNow || !item.Notified || !strings.HasPrefix(item.AlertID, key+"@") {
		t.Fatalf("item = %+v; want an ALERT of %s sent to the human at once", item, key)
	}
	assertUsageLimitedBody(t, item.Body, item.AlertID, ts, "implementer, leader, reviewer", []string{
		"implementer (claude): " + ulClaudeSessionLine,
		"leader (codex): " + ulCodexUsageLine,
		"reviewer (claude): " + ulClaudeWeeklyLine,
	})
	want := map[string]any{"ts": ts, "org_id": "org-a", "inbox_id": "e1", "alert_id": item.AlertID, "subject": LeaderIdentity, "reason": InboxReasonDirectorDisabled}
	if lines := escalationLines(t, f.o.EscalationsPath); len(lines) != 1 || !reflect.DeepEqual(lines[0], want) {
		t.Errorf("escalations.jsonl = %v; want exactly one line %v", lines, want)
	}
	if !f.usageLimitedActive() || !slices.Equal(f.status.UsageLimitedSeats, []string{"implementer", LeaderIdentity, "reviewer"}) {
		t.Errorf("usage_limited active %t with seats %v; want active with the three", f.usageLimitedActive(), f.status.UsageLimitedSeats)
	}
	assertEscalatedOnce(t, f.status, item.AlertID)

	f.cycle(t)
	if n := len(f.uh.readsSoFar()); n != 6 {
		t.Errorf("%d screen reads after the third cycle; want 6, each seat's screen once more", n)
	}
	if n, alerts := len(mustReadInbox(t, f.o.Inbox).Items), f.alerts(condUsageLimited); n != 1 || len(alerts) != 1 || len(recurredEvents(f.usageLimitedItems(t)[0])) != 0 {
		t.Errorf("after the third cycle: %d items, ALERTs %q; want still one of each and no recurrence", n, alerts)
	}
}

// AC3: the usage_limited item is recorded in the cycle the seats stop,
// with deadman_minutes far off, and reaches a person like the watch's other
// ALERT items: left to the director with the switch on and a watch that
// accepts work, sent to the human otherwise, with leader escalate's
// reasons. Without an inbox store the escalation takes the path without
// the inbox, whose escalations.jsonl line has REASON usage_limit too and
// whose stderr banner names the seats that stopped.
func TestWatchUsageLimit_TheItemIsDeliveredLikeTheWatchsOtherAlertItems(t *testing.T) {
	yes, no := true, false
	cases := []struct {
		name         string
		enabled      bool
		accepts      *bool
		noInbox      bool
		wantDelivery string
		wantReason   string // the reason of the send to the human; "" for none
	}{
		{name: "director switch off", wantDelivery: InboxDeliveryNow, wantReason: InboxReasonDirectorDisabled},
		{name: "switch on, a watch that accepts work", enabled: true, accepts: &yes, wantDelivery: InboxDeliveryDeadline},
		{name: "switch on, a watch that accepts no work", enabled: true, accepts: &no, wantDelivery: InboxDeliveryNow, wantReason: InboxReasonNoWatch},
		{name: "no inbox store", noInbox: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newUsageLimitFixture(t, usageLimitSeat{LeaderIdentity, "codex"}, usageLimitSeat{"implementer", "claude"})
			f.o.Config.DeadmanMinutes = 60
			f.o.Config.Director.Enabled = tc.enabled
			var acceptsWork func() bool
			if tc.accepts != nil {
				answer := *tc.accepts
				acceptsWork = func() bool { return answer }
			}
			f.rerun(acceptsWork)
			if tc.noInbox {
				f.o.Inbox = nil
			}
			f.screen(t, LeaderIdentity, "codex-usage-limit.txt")
			f.screen(t, "implementer", "claude-session-limit.txt")
			f.cycle(t)
			ts := f.clk.Now().UTC().Format(time.RFC3339)
			f.cycle(t)

			if len(f.status.PendingAlerts) != 0 {
				t.Errorf("pending alerts %+v; want the usage_limited ALERT escalated in its own cycle", f.status.PendingAlerts)
			}
			if tc.noInbox {
				lines := escalationLines(t, f.o.EscalationsPath)
				if len(lines) != 1 || lines[0]["reason"] != usageLimitReason || lines[0]["subject"] != LeaderIdentity ||
					!strings.HasPrefix(fmt.Sprint(lines[0]["alert_id"]), conditionKey("org-a", LeaderIdentity, condUsageLimited)+"@") {
					t.Errorf("escalations.jsonl = %v; want one line of the usage_limited alert with reason %s", lines, usageLimitReason)
				}
				banner := fmt.Sprintf("WATCHDOG ESCALATION: org=org-a alert=%s subject=leader reason=%s seats=implementer,leader -- see %s (not recorded in the inbox: ",
					lines[0]["alert_id"], usageLimitReason, f.o.EscalationsPath)
				if !strings.Contains(f.stderr.String(), banner) || len(f.escalated) != 1 {
					t.Errorf("stderr %q, notifications %q; want the banner %q, naming the seats, and the notification of the path without the inbox", f.stderr.String(), f.escalated, banner)
				}
				return
			}
			items := f.usageLimitedItems(t)
			if len(items) != 1 {
				t.Fatalf("usage_limited items %+v; want one", items)
			}
			item := items[0]
			if item.Delivery != tc.wantDelivery || item.Notified != (tc.wantReason != "") {
				t.Errorf("item = %+v; want delivery %s, notified %t", item, tc.wantDelivery, tc.wantReason != "")
			}
			if m, err := protocol.Parse(item.Body); err != nil || m.Fields["REASON"] != usageLimitReason || m.Fields["ALERT_TS"] != ts {
				t.Errorf("body %q (parse error %v); want REASON %s and ALERT_TS %s", item.Body, err, usageLimitReason, ts)
			}
			if tc.wantReason == "" {
				head := "WATCHDOG ESCALATION: org=org-a alert=" + item.AlertID + " subject=leader reason=" + usageLimitReason + " item=e1 -- left in "
				if !strings.Contains(f.stderr.String(), head) || fileOrAbsent(t, f.o.EscalationsPath) != "<absent>" || len(f.notified) != 0 {
					t.Errorf("stderr %q, notifications %q; want the item left to the director with no send", f.stderr.String(), f.notified)
				}
				return
			}
			if lines := escalationLines(t, f.o.EscalationsPath); len(lines) != 1 || lines[0]["reason"] != tc.wantReason || lines[0]["subject"] != LeaderIdentity {
				t.Errorf("escalations.jsonl = %v; want one send with reason %s about the leader", lines, tc.wantReason)
			}
		})
	}
}

// A usage_limited pending alert in the watch-status file whose seats the
// watch would not write (none, a seat or a driver that is not an
// identifier) is escalated by the path without the inbox, with REASON
// usage_limit, and records no item. Its banner names the seats it keeps,
// escaped, and no seats for none.
func TestWatchUsageLimit_APendingAlertWithSeatsTheWatchDoesNotWriteRecordsNoItem(t *testing.T) {
	cases := []struct {
		name  string
		seats *watchUsageLimitedSeats
		cause string
		after string // what the banner holds after the reason
	}{
		{"no seat", nil, "names no seat", " -- see "},
		{"a seat with a line break", &watchUsageLimitedSeats{Seats: []watchUsageLimitedSeat{{Seat: "bad\nSEAT: x", Driver: "claude", Line: "l"}}}, "is not a seat", ` seats=bad\nSEAT: x -- see `},
		{"a driver that is not one", &watchUsageLimitedSeats{Seats: []watchUsageLimitedSeat{{Seat: "implementer", Driver: "Claude Code", Line: "l"}}}, "has a driver that is not one", " seats=implementer -- see "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newUsageLimitFixture(t)
			alertID := watchAlertID(conditionKey("org-a", LeaderIdentity, condUsageLimited), f.clk.Now())
			f.status.PendingAlerts = map[string]*watchPendingAlert{alertID: {
				AlertID: alertID, TS: f.clk.Now().UTC().Format(time.RFC3339), Subject: LeaderIdentity, HistoryLeaderLines: -1, UsageLimited: tc.seats,
			}}
			f.cycle(t)
			if items := mustReadInbox(t, f.o.Inbox).Items; len(items) != 0 {
				t.Errorf("inbox = %+v; want no item", items)
			}
			if lines := escalationLines(t, f.o.EscalationsPath); len(lines) != 1 || lines[0]["reason"] != usageLimitReason || lines[0]["alert_id"] != alertID {
				t.Errorf("escalations.jsonl = %v; want one line of %s with reason %s", lines, alertID, usageLimitReason)
			}
			if !strings.Contains(f.stderr.String(), "not recorded in the inbox: ") || !strings.Contains(f.stderr.String(), tc.cause) {
				t.Errorf("stderr = %q; want the banner of the path without the inbox naming %q", f.stderr.String(), tc.cause)
			}
			if head := escalationHead("org-a", alertID, LeaderIdentity, usageLimitReason) + tc.after; !strings.Contains(f.stderr.String(), head) {
				t.Errorf("stderr = %q; want the banner %q", f.stderr.String(), head)
			}
		})
	}
}

// AC4: with org-a's usage_limited item e1 of the leader and the
// implementer, the reviewer, which was working, stops on a usage limit in
// the next cycle. With e1 open, e1 gets one recurred event whose note names
// the reviewer and its limit line, and nothing else is recorded or sent.
// With e1 acked or resolved, or resolved between the read and the append,
// the three seats get a new item e2 and a second ALERT to the leader, and
// e1 gets no recurred event.
func TestWatchUsageLimit_ASeatThatStopsLaterIsARecurrenceOfAnOpenItemAndANewItemOtherwise(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(t *testing.T, f *usageLimitFixture)
		wantNew bool
	}{
		{name: "e1 open", prepare: func(*testing.T, *usageLimitFixture) {}},
		{name: "e1 acked", wantNew: true, prepare: func(t *testing.T, f *usageLimitFixture) {
			if _, err := f.o.Inbox.Ack("e1"); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "e1 resolved", wantNew: true, prepare: func(t *testing.T, f *usageLimitFixture) {
			if err := f.o.Inbox.Resolve("e1", "docs/reports/x.md"); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "e1 resolved between the read and the append", wantNew: true, prepare: func(t *testing.T, f *usageLimitFixture) {
			setBeforeAppendRecurred(t, func() {
				if err := f.o.Inbox.Resolve("e1", "docs/reports/x.md"); err != nil && !errors.Is(err, ErrInboxResolved) {
					t.Error(err)
				}
			})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newUsageLimitFixture(t, threeSeats...)
			f.limitThreeSeats(t)
			f.reply("reviewer", watchAgentReply("working", 1))
			f.cycle(t)
			f.reply("reviewer", watchAgentReply("working", 2))
			f.cycle(t)
			if items := f.usageLimitedItems(t); len(items) != 1 || !strings.Contains(items[0].Body, "\nSEATS: implementer, leader\n") {
				t.Fatalf("usage_limited items %+v; want e1 of the leader and the implementer", items)
			}
			tc.prepare(t, f)

			f.cycle(t) // the reviewer's answer stops changing: it is stalled, on its weekly limit
			if !slices.Equal(f.status.UsageLimitedSeats, []string{"implementer", LeaderIdentity, "reviewer"}) || len(f.alerts(condStall)) != 0 {
				t.Errorf("usage_limited seats %v, stall ALERTs %q; want the three seats and no stall", f.status.UsageLimitedSeats, f.alerts(condStall))
			}
			items := f.usageLimitedItems(t)
			e1 := items[0]
			if !tc.wantNew {
				recurred := recurredEvents(e1)
				if len(items) != 1 || len(recurred) != 1 || len(f.alerts(condUsageLimited)) != 1 {
					t.Fatalf("items %+v, ALERTs %q; want only e1, with one recurred event, and the one ALERT", items, f.alerts(condUsageLimited))
				}
				if ev := recurred[0]; ev.Count != 1 || ev.Note != "stopped on a usage limit: reviewer (claude): "+ulClaudeWeeklyLine ||
					!strings.HasPrefix(ev.AlertID, conditionKey("org-a", LeaderIdentity, condUsageLimited)+"@") || ev.AlertID == e1.AlertID {
					t.Errorf("recurred event %+v; want count 1, a new alert_id of the key, and a note naming the reviewer and its line", ev)
				}
				line := "watchdog: org=org-a condition=usage_limited: seats reviewer stopped on a usage limit; recorded as a recurrence on inbox item e1 (recurrences=1)\n"
				if !strings.Contains(f.stderr.String(), line) {
					t.Errorf("stderr = %q; want %q", f.stderr.String(), line)
				}
				return
			}
			if len(items) != 2 || items[1].ID != "e2" || len(recurredEvents(e1)) != 0 || len(f.alerts(condUsageLimited)) != 2 {
				t.Fatalf("items %+v, ALERTs %q; want e1 untouched, a new e2, and a second ALERT", items, f.alerts(condUsageLimited))
			}
			if !strings.Contains(items[1].Body, "\nSEATS: implementer, leader, reviewer\n") || !strings.Contains(items[1].Body, "\nreviewer (claude): "+ulClaudeWeeklyLine) {
				t.Errorf("e2 body %q; want the three seats, the reviewer's line among them", items[1].Body)
			}
		})
	}
}

// AC5: once no seat of org-a shows a usage limit (the screens go back to
// the prompt and the seats move), the condition clears and the set of
// seats is emptied, in memory and in the status file. When the seats stop
// on a usage limit again, an item that is not resolved, open or acked,
// takes it as one recurred event naming the seats; a resolved one gives a
// new item.
func TestWatchUsageLimit_TheConditionClearsAndALaterLimitFollowsTheEpisodeRule(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(t *testing.T, s *InboxStore)
		wantNew bool
	}{
		{name: "e1 open", prepare: func(*testing.T, *InboxStore) {}},
		{name: "e1 acked", prepare: func(t *testing.T, s *InboxStore) {
			if _, err := s.Ack("e1"); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "e1 resolved", wantNew: true, prepare: func(t *testing.T, s *InboxStore) {
			if err := s.Resolve("e1", "docs/reports/x.md"); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newUsageLimitFixture(t, usageLimitSeat{LeaderIdentity, "codex"}, usageLimitSeat{"implementer", "claude"})
			f.screen(t, LeaderIdentity, "codex-usage-limit.txt")
			f.screen(t, "implementer", "claude-session-limit.txt")
			f.cycle(t)
			f.cycle(t)
			if items := f.usageLimitedItems(t); len(items) != 1 {
				t.Fatalf("usage_limited items %+v; want e1", items)
			}

			// The limit is over: the screens show the prompt, the seats answer
			// anew, and the leader sends each seat a TASK.
			f.screen(t, LeaderIdentity, "codex-idle-after-reply.txt")
			f.screen(t, "implementer", "claude-idle-after-reply.txt")
			for _, seat := range []string{LeaderIdentity, "implementer"} {
				f.reply(seat, "after-the-limit")
				if err := f.o.Manifest.Append(ManifestEvent{TS: f.clk.Now().UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: seat, Event: EventSent}); err != nil {
					t.Fatal(err)
				}
			}
			f.cycle(t)
			if f.usageLimitedActive() || f.status.UsageLimitedSeats != nil || len(f.alerts(condStall)) != 0 {
				t.Fatalf("usage_limited active %t with seats %v, stall ALERTs %q; want it cleared with no seat and no stall", f.usageLimitedActive(), f.status.UsageLimitedSeats, f.alerts(condStall))
			}
			if raw, err := os.ReadFile(f.run.statusPath); err != nil || strings.Contains(string(raw), "usage_limited_seats") {
				t.Errorf("status file %s (%v); want no usage_limited_seats once the condition cleared", raw, err)
			}
			tc.prepare(t, f.o.Inbox)

			// The seats stop on a usage limit again.
			f.screen(t, LeaderIdentity, "codex-usage-limit.txt")
			f.screen(t, "implementer", "claude-session-limit.txt")
			f.clk.Advance(10 * time.Minute)
			f.cycle(t)
			items := f.usageLimitedItems(t)
			if !f.usageLimitedActive() || !slices.Equal(f.status.UsageLimitedSeats, []string{"implementer", LeaderIdentity}) {
				t.Errorf("usage_limited active %t with seats %v; want active with both", f.usageLimitedActive(), f.status.UsageLimitedSeats)
			}
			if tc.wantNew {
				if len(items) != 2 || items[1].ID != "e2" || len(recurredEvents(items[0])) != 0 {
					t.Errorf("items %+v; want e1 untouched and a new e2", items)
				}
				return
			}
			recurred := recurredEvents(items[0])
			wantNote := "stopped on a usage limit: implementer (claude): " + ulClaudeSessionLine + "; leader (codex): " + ulCodexUsageLine
			if len(items) != 1 || len(recurred) != 1 || recurred[0].Note != wantNote || recurred[0].Count != 1 {
				t.Errorf("items %+v; want only e1 with one recurred event, count 1, note %q", items, wantNote)
			}
		})
	}
}

// AC6: a seat whose stall holds and whose screen is no usage limit stalls
// as before: one stall ALERT about it, and no usage_limited. That covers a
// screen that cannot be read, a network error, a screen no table knows, a
// claude limit on a codex seat (the seat's own driver decides), a seat with
// no driver recorded, and a seat with no pane recorded, whose screen is not
// read at all. When the screen could not be read (the read failed, or no
// pane is recorded), the stall ALERT ends with a line that says why, one
// line with no control character and the error cut to
// usageLimitUnreadMaxRunes runes; a screen that was read adds no line.
func TestWatchUsageLimit_ASeatWhoseScreenIsNoUsageLimitStallsAsBefore(t *testing.T) {
	const unread = "screen not read for the usage-limit check: "
	longErr := "herdr:\x1b[31m pane\r\nread\x07 failed " + strings.Repeat("y", 400)
	longClean := "herdr pane read pane-implementer failed: herdr:[31m pane read failed " + strings.Repeat("y", 400)
	cases := []struct {
		name      string
		seat      usageLimitSeat
		noPane    bool
		screen    string // a fixture of testdata/resume; "" leaves the pane empty
		readErr   error
		wantReads int
		wantNote  string // the last line of the stall ALERT; "" for none
	}{
		{name: "a read that fails", seat: usageLimitSeat{"implementer", "claude"}, readErr: errors.New("herdr: pane read failed"), wantReads: 1,
			wantNote: unread + "herdr pane read pane-implementer failed: herdr: pane read failed"},
		{name: "a read that fails with a long error holding control characters", seat: usageLimitSeat{"implementer", "claude"}, readErr: errors.New(longErr), wantReads: 1,
			wantNote: unread + string([]rune(longClean)[:usageLimitUnreadMaxRunes-1]) + "…"},
		{name: "a network error", seat: usageLimitSeat{"implementer", "claude"}, screen: "claude-enotfound.txt", wantReads: 1},
		{name: "a screen no table knows", seat: usageLimitSeat{"implementer", "claude"}, wantReads: 1},
		{name: "a claude limit on a codex seat", seat: usageLimitSeat{"implementer", "codex"}, screen: "claude-session-limit.txt", wantReads: 1},
		{name: "no driver recorded", seat: usageLimitSeat{"implementer", ""}, screen: "claude-session-limit.txt", wantReads: 1},
		{name: "no pane recorded", seat: usageLimitSeat{"implementer", "claude"}, noPane: true, screen: "claude-session-limit.txt",
			wantNote: unread + "no pane is recorded for the seat"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newUsageLimitFixture(t)
			pane := usageLimitPane(tc.seat.id)
			if tc.noPane {
				pane = ""
			}
			appendUsageLimitSeat(t, f.o, f.clk.Now().Add(-10*time.Minute), tc.seat, pane)
			if tc.screen != "" {
				f.screen(t, tc.seat.id, tc.screen)
			}
			f.uh.readErr[usageLimitPane(tc.seat.id)] = tc.readErr
			f.cycle(t)
			f.cycle(t)
			stalls := f.alerts(condStall)
			if len(stalls) != 1 || !strings.Contains(stalls[0], "\nSEAT: implementer\n") {
				t.Fatalf("stall ALERTs %q; want one about the implementer", stalls)
			}
			assertStallNote(t, stalls[0], tc.wantNote)
			if len(f.alerts(condUsageLimited)) != 0 || f.usageLimitedActive() || len(f.usageLimitedItems(t)) != 0 || f.status.UsageLimitedSeats != nil {
				t.Errorf("usage_limited ALERTs %q, active %t, seats %v; want none", f.alerts(condUsageLimited), f.usageLimitedActive(), f.status.UsageLimitedSeats)
			}
			if reads := f.uh.readsSoFar(); len(reads) != tc.wantReads {
				t.Errorf("screen reads %+v; want %d", reads, tc.wantReads)
			}
		})
	}
}

// assertStallNote fails the test unless msg, a stall ALERT of the
// implementer, passes the protocol check and its last line is wantNote, the
// line that says why the screen was not read, with no control character and
// the error in it at most usageLimitUnreadMaxRunes runes; for a wantNote of
// "", unless msg ends with the stall line and holds no such line.
func assertStallNote(t *testing.T, msg, wantNote string) {
	t.Helper()
	if err := protocol.ValidateText(msg, protocol.DefaultMaxBodyChars); err != nil {
		t.Errorf("stall ALERT %q fails the protocol check: %v", msg, err)
	}
	lines := strings.Split(msg, "\n")
	last := lines[len(lines)-1]
	if wantNote == "" {
		if last != "seat implementer heartbeat stalled for over 5m" || strings.Contains(msg, "screen not read") {
			t.Errorf("stall ALERT %q; want it to end with the stall line and say nothing of the screen", msg)
		}
		return
	}
	if last != wantNote || lines[len(lines)-2] != "seat implementer heartbeat stalled for over 5m" {
		t.Errorf("stall ALERT ends with\n%q\nwant the stall line, then\n%q", lines[len(lines)-2:], wantNote)
	}
	if i := strings.IndexFunc(last, unicode.IsControl); i >= 0 {
		t.Errorf("note %q holds a control character at %d", last, i)
	}
	if n := utf8.RuneCountInString(strings.TrimPrefix(last, "screen not read for the usage-limit check: ")); n > usageLimitUnreadMaxRunes {
		t.Errorf("the error in note %q is %d runes; want at most %d", last, n, usageLimitUnreadMaxRunes)
	}
}

// Plan 2026-11-13-claude-5h-usage-limit, AC6: a claude seat whose pane shows
// the real screen of Claude Code 2.1.316 waiting for its 5-hour limit to
// reset (claude-5h-limit-continuing.txt) goes into the org's usage_limited
// in the cycle its stall holds, the second, with the footer block as its
// line, and raises no stall.
func TestWatchUsageLimit_AClaudeSeatWaitingForItsLimitToResetIsUsageLimited(t *testing.T) {
	f := newUsageLimitFixture(t, usageLimitSeat{"implementer", "claude"})
	f.screen(t, "implementer", "claude-5h-limit-continuing.txt")
	f.cycle(t)
	ts := f.clk.Now().UTC().Format(time.RFC3339)
	f.cycle(t)

	if reads := f.uh.readsSoFar(); len(reads) != 1 {
		t.Errorf("screen reads %+v; want one, in the cycle the stall holds", reads)
	}
	if stalls := f.alerts(condStall); len(stalls) != 0 || len(f.activeStalls()) != 0 {
		t.Errorf("stall ALERTs %q, active stalls %v; want none for a seat waiting for its limit to reset", stalls, f.activeStalls())
	}
	if !f.usageLimitedActive() || !slices.Equal(f.status.UsageLimitedSeats, []string{"implementer"}) {
		t.Errorf("usage_limited active %t with seats %v; want active with the implementer", f.usageLimitedActive(), f.status.UsageLimitedSeats)
	}
	items := f.usageLimitedItems(t)
	if len(items) != 1 {
		t.Fatalf("usage_limited items %+v; want one", items)
	}
	assertUsageLimitedBody(t, items[0].Body, items[0].AlertID, ts, "implementer", []string{"implementer (claude): " + ulClaude5hWaitLine})
}

// Plan 2026-11-13-claude-5h-usage-limit, AC6 and D2: a claude seat whose last
// reply quotes the wait for a usage limit to reset above the input box, with
// nothing of it in the footer (usageLimitQuotedWaitScreen), is no usage
// limit. Its stall is raised in the cycle it holds and stays active in the
// cycles after it, its screen read in each, and the org never goes into
// usage_limited.
func TestWatchUsageLimit_ASeatWhoseReplyQuotesTheWaitKeepsItsStall(t *testing.T) {
	f := newUsageLimitFixture(t, usageLimitSeat{"implementer", "claude"})
	f.screenText("implementer", usageLimitQuotedWaitScreen)
	stall := conditionKey("org-a", "implementer", condStall)
	f.cycle(t)
	for n := 2; n <= 4; n++ {
		f.cycle(t)
		if active := f.activeStalls(); !slices.Equal(active, []string{stall}) {
			t.Errorf("after cycle %d: active stalls %v; want the implementer's", n, active)
		}
		if f.usageLimitedActive() || len(f.alerts(condUsageLimited)) != 0 || len(f.usageLimitedItems(t)) != 0 || f.status.UsageLimitedSeats != nil {
			t.Errorf("after cycle %d: usage_limited active %t, ALERTs %q, seats %v; want none", n, f.usageLimitedActive(), f.alerts(condUsageLimited), f.status.UsageLimitedSeats)
		}
		if reads := f.uh.readsSoFar(); len(reads) != n-1 {
			t.Errorf("after cycle %d: screen reads %+v; want %d, one in each cycle the stall holds", n, reads, n-1)
		}
	}
	stalls := f.alerts(condStall)
	if len(stalls) != 1 || !strings.Contains(stalls[0], "\nSEAT: implementer\n") {
		t.Fatalf("stall ALERTs %q; want one about the implementer", stalls)
	}
	assertStallNote(t, stalls[0], "")
}

// An Org with no herdr reads no screen and reports neither a usage limit
// nor an error, so the seat's stall ALERT gets no line about the screen.
func TestWatchUsageLimit_AnOrgWithNoHerdrReadsNoScreenAndReportsNoError(t *testing.T) {
	w := &watchRun{org: &Org{}}
	seat, limited, err := w.readUsageLimitScreen(SeatStatus{SeatID: "implementer", Driver: "claude", PaneID: "pane-implementer"})
	if limited || err != nil || seat != (watchUsageLimitedSeat{}) || usageLimitUnreadNote(err) != "" {
		t.Errorf("readUsageLimitScreen = %+v, %t, %v; want no entry, false, and no error", seat, limited, err)
	}
}

// AC7: the screen of a seat whose stall does not hold is never read: one
// whose agent get answers anew every cycle, and one whose agent get stays
// the same while its manifest events move, both on a usage limit screen.
func TestWatchUsageLimit_TheScreenOfASeatThatIsNotStalledIsNotRead(t *testing.T) {
	f := newUsageLimitFixture(t, usageLimitSeat{LeaderIdentity, "codex"}, usageLimitSeat{"implementer", "claude"})
	f.screen(t, LeaderIdentity, "codex-usage-limit.txt")
	f.screen(t, "implementer", "claude-session-limit.txt")
	for rev := 1; rev <= 4; rev++ {
		f.reply("implementer", watchAgentReply("working", rev))
		if err := f.o.Manifest.Append(ManifestEvent{TS: f.clk.Now().UTC().Format(time.RFC3339), OrgID: "org-a", SeatID: LeaderIdentity, Event: EventSent}); err != nil {
			t.Fatal(err)
		}
		f.cycle(t)
	}
	if reads := f.uh.readsSoFar(); len(reads) != 0 {
		t.Errorf("screen reads %+v; want none for seats that are not stalled", reads)
	}
	if len(f.alerts(condStall)) != 0 || len(f.alerts(condUsageLimited)) != 0 || len(mustReadInbox(t, f.o.Inbox).Items) != 0 {
		t.Errorf("ALERTs %+v, inbox %+v; want nothing", watchdogAlerts(f.a), mustReadInbox(t, f.o.Inbox).Items)
	}
}

// AC7: a screen read that does not answer is cut off at the deadline of
// each driver call of the pulse (Org.DriverCallTimeout), and the seat
// stalls as before, its stall ALERT saying that the read was cut off.
func TestWatchUsageLimit_AScreenReadThatDoesNotAnswerIsCutOffAndTheSeatStalls(t *testing.T) {
	f := newUsageLimitFixture(t, usageLimitSeat{"implementer", "claude"})
	f.o.DriverCallTimeout = 50 * time.Millisecond
	f.screen(t, "implementer", "claude-session-limit.txt")
	f.uh.hang[usageLimitPane("implementer")] = true
	f.cycle(t)
	start := time.Now()
	f.cycle(t)
	if took := time.Since(start); took > 5*time.Second {
		t.Errorf("the cycle took %v; want the read cut off at the deadline", took)
	}
	if reads := f.uh.readsSoFar(); len(reads) != 1 || !reads[0].hasDeadline {
		t.Errorf("screen reads %+v; want one under a deadline", reads)
	}
	stalls := f.alerts(condStall)
	if len(stalls) != 1 || f.usageLimitedActive() {
		t.Fatalf("stall ALERTs %q, usage_limited active %t; want the implementer's stall", stalls, f.usageLimitedActive())
	}
	assertStallNote(t, stalls[0], "screen not read for the usage-limit check: herdr pane read pane-implementer failed: "+context.DeadlineExceeded.Error())
}

// AC11: a leader stopped on a usage limit goes into usage_limited while
// another seat is working and while org-a has a QUESTION or a BLOCKED its
// leader raised that is not resolved, when its stall would not be raised. A
// leader whose screen is no usage limit is read too (its stall holds before
// the exemption) but, under the same conditions, is still not stalled, as
// before.
func TestWatchUsageLimit_ALeaderStoppedOnAUsageLimitIsReportedWhileItWouldBeWaiting(t *testing.T) {
	escalate := func(typ string, ack bool) func(*testing.T, *usageLimitFixture) {
		return func(t *testing.T, f *usageLimitFixture) {
			id, err := f.o.Inbox.Escalate(InboxEscalation{OrgID: "org-a", Type: typ, TaskID: "t1", Body: "TYPE: " + typ + "\nTASK_ID: t1\n\nwaiting"})
			if err != nil {
				t.Fatal(err)
			}
			if ack {
				if _, err := f.o.Inbox.Ack(id); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	waits := []struct {
		name  string
		setup func(*testing.T, *usageLimitFixture)
		// working makes the implementer answer anew, working, every cycle.
		working bool
	}{
		{name: "another seat is working", setup: func(*testing.T, *usageLimitFixture) {}, working: true},
		{name: "an open QUESTION", setup: escalate(protocol.TypeQuestion, false)},
		{name: "an acked BLOCKED", setup: escalate(protocol.TypeBlocked, true)},
	}
	for _, wait := range waits {
		for _, limited := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s, leader on a usage limit %t", wait.name, limited), func(t *testing.T) {
				f := newUsageLimitFixture(t, usageLimitSeat{LeaderIdentity, "codex"}, usageLimitSeat{"implementer", "claude"})
				wait.setup(t, f)
				f.screen(t, LeaderIdentity, "codex-idle-after-reply.txt")
				if limited {
					f.screen(t, LeaderIdentity, "codex-usage-limit.txt")
				}
				for rev := 1; rev <= 2; rev++ {
					if wait.working {
						f.reply("implementer", watchAgentReply("working", rev))
					} else {
						f.reply("implementer", watchAgentReply("idle", rev))
					}
					f.cycle(t)
				}
				leaderReads := 0
				for _, r := range f.uh.readsSoFar() {
					if r.paneID == usageLimitPane(LeaderIdentity) {
						leaderReads++
					}
				}
				if leaderReads != 1 || len(f.alerts(condStall)) != 0 {
					t.Errorf("%d reads of the leader's screen, stall ALERTs %q; want one read and no stall", leaderReads, f.alerts(condStall))
				}
				items := f.usageLimitedItems(t)
				if !limited {
					if len(items) != 0 || f.usageLimitedActive() {
						t.Errorf("usage_limited items %+v, active %t; want none for a leader that is waiting", items, f.usageLimitedActive())
					}
					return
				}
				if len(items) != 1 || !strings.Contains(items[0].Body, "\nSEATS: leader\n") || !strings.Contains(items[0].Body, "\nleader (codex): "+ulCodexUsageLine) {
					t.Errorf("usage_limited items %+v; want one naming the leader and its limit line", items)
				}
			})
		}
	}
}

// The body of a usage_limited item stays within protocol.DefaultMaxBodyChars
// and passes the protocol check, with control characters, a carriage
// return, and a line separator in the stored lines left out of it: six
// seats with long lines each get a line, cut to the same length after the
// limit sentence; with twenty seats the last ones are named in one
// "+N more" line. The ALERT to the leader stays within the cap too, and the
// note of a recurrence within InboxNoteMaxRunes, one line AppendRecurred
// takes.
func TestWatchUsageLimit_TheItemBodyStaysWithinTheCap(t *testing.T) {
	long := "■ You've\x1b hit your usage limit.\x07 Upgrade\r to Pro  or purchase more credits, or try again at 3:05 PM. " + strings.Repeat("x", 400)
	for _, n := range []int{6, 20} {
		t.Run(fmt.Sprintf("%d seats", n), func(t *testing.T) {
			var seats []watchUsageLimitedSeat
			var ids []string
			for i := range n {
				id := fmt.Sprintf("seat-%02d-%s", i, strings.Repeat("a", 20))
				seats = append(seats, watchUsageLimitedSeat{Seat: id, Driver: "codex", Line: long})
				ids = append(ids, id)
			}
			alertID := watchAlertID(conditionKey("org-a", LeaderIdentity, condUsageLimited), time.Unix(1, 0))
			pending := &watchPendingAlert{AlertID: alertID, TS: "2026-01-01T00:00:00Z", Subject: LeaderIdentity, UsageLimited: &watchUsageLimitedSeats{Seats: seats}}
			if err := checkWatchEscalationValues(alertID, pending); err != nil {
				t.Fatalf("checkWatchEscalationValues: %v", err)
			}
			body := watchEscalationBody("org-a", alertID, pending, usageLimitReason, 10)
			for name, text := range map[string]string{"item body": body, "ALERT to the leader": usageLimitedAlertMessage("org-a", seats)} {
				if got := utf8.RuneCountInString(text); got > protocol.DefaultMaxBodyChars {
					t.Errorf("%s is %d runes; want at most %d", name, got, protocol.DefaultMaxBodyChars)
				}
				if err := protocol.ValidateText(text, protocol.DefaultMaxBodyChars); err != nil {
					t.Errorf("%s fails the protocol check: %v", name, err)
				}
				if i := strings.IndexFunc(text, func(r rune) bool { return r != '\n' && (unicode.IsControl(r) || r == ' ') }); i >= 0 {
					t.Errorf("%s holds a control character at %d: %q", name, i, text)
				}
			}
			m, err := protocol.Parse(body)
			if err != nil || m.Fields["SEATS"] != strings.Join(ids, ", ") {
				t.Fatalf("SEATS %q (parse error %v); want every seat", m.Fields["SEATS"], err)
			}
			lines := strings.Split(m.Body, "\n")[1:]
			var more string
			if last := lines[len(lines)-1]; strings.HasPrefix(last, "+") {
				more, lines = last, lines[:len(lines)-1]
			}
			for i, line := range lines {
				if want := ids[i] + " (codex): ■ You've hit your usage limit. Upgrade to Pro or purchase more credits"; !strings.HasPrefix(line, want) || !strings.HasSuffix(line, "…") {
					t.Errorf("line %d = %q; want %q... cut with …", i, line, want)
				}
			}
			if n == 6 && (len(lines) != 6 || more != "") {
				t.Errorf("%d lines and more %q; want a line for each of the six seats", len(lines), more)
			}
			if rest := ids[len(lines):]; len(rest) > 0 && more != fmt.Sprintf("+%d more: %s", len(rest), strings.Join(rest, ", ")) {
				t.Errorf("more = %q; want the %d seats with no line: %v", more, len(rest), rest)
			}
			if n == 20 && (len(lines) == 0 || more == "") {
				t.Errorf("%d lines and more %q; want some seats with a line and the rest in +N more", len(lines), more)
			}

			note := usageLimitRecurNote(seats)
			if err := validateInboxNoteLine("recurred", note); err != nil {
				t.Errorf("note %q: %v", note, err)
			}
		})
	}
}

// A status file written before usage_limited (no usage_limited_seats, a
// pending alert with no usage_limited) loads with neither and works: the
// watch's first cycle, through RunWatchAll, reads the seats it stalls and
// records one usage_limited item, and the status file it writes holds the
// seats. A pending alert of another condition is written with no
// usage_limited key, as before.
func TestWatchUsageLimit_AStatusFileWrittenBeforeUsageLimitedLoadsAndWorks(t *testing.T) {
	f := newWatchAllFixture(t)
	uh := &usageLimitHerdr{fakeWatchHerdr: f.h, screens: map[string]string{
		usageLimitPane(LeaderIdentity): readResumeScreen(t, "codex-usage-limit.txt"),
		usageLimitPane("implementer"):  readResumeScreen(t, "claude-session-limit.txt"),
	}}
	f.o.Herdr = uh
	f.o.Config.Watchdog.StallMinutes = 5
	f.o.Config.DeadmanMinutes = 60
	for _, s := range []usageLimitSeat{{LeaderIdentity, "codex"}, {"implementer", "claude"}} {
		appendUsageLimitSeat(t, f.o, f.clk.Now(), s, usageLimitPane(s.id))
	}
	f.clk.Advance(10 * time.Minute)
	seatAlert := conditionKey("org-a", "seat-9", condLiveness) + "@1"
	old := fmt.Sprintf(`{"org_id":"org-a","last_cycle_ts":"2026-01-01T00:09:00Z","cycles":3,"watchdog_joined":true,`+
		`"pending_alerts":{%q:{"alert_id":%q,"ts":%q,"subject":"seat-9","manifest_len":0,"lead_agent_get":"","history_lead_lines":-1}},`+
		`"seat_snapshots":{"leader":{"agent_get":"ok","agent_get_seen":true},"implementer":{"agent_get":"ok","agent_get_seen":true}}}`,
		seatAlert, seatAlert, f.clk.Now().UTC().Format(time.RFC3339))
	statusPath := filepath.Join(f.dir, WatchStatusFileName("org-a"))
	if err := os.WriteFile(statusPath, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadWatchStatus(statusPath, "org-a")
	if err != nil || loaded.UsageLimitedSeats != nil || loaded.PendingAlerts[seatAlert].UsageLimited != nil {
		t.Fatalf("loaded %+v (%v); want no usage-limited seats anywhere", loaded, err)
	}

	if err := runWatchAll(t, f.o, context.Background(), f.params(1), WatchHooks{}); err != nil {
		t.Fatalf("RunWatchAll: %v", err)
	}
	var items []InboxItem
	for _, item := range mustReadInbox(t, f.o.Inbox).Items {
		if alertIDKey(item.AlertID) == conditionKey("org-a", LeaderIdentity, condUsageLimited) {
			items = append(items, item)
		}
	}
	if len(items) != 1 || !strings.Contains(items[0].Body, "\nSEATS: implementer, leader\n") {
		t.Fatalf("usage_limited items %+v; want one of the two seats", items)
	}
	if status := readWatchStatusFile(t, f.dir, "org-a"); !slices.Equal(status.UsageLimitedSeats, []string{"implementer", LeaderIdentity}) || status.Cycles != 4 {
		t.Errorf("status file %+v; want 4 cycles and the two seats", status)
	}
	raw, err := json.Marshal(&watchStatusFile{OrgID: "org-a", PendingAlerts: map[string]*watchPendingAlert{seatAlert: {AlertID: seatAlert, Subject: "seat-9"}}})
	if err != nil || strings.Contains(string(raw), "usage_limited") {
		t.Errorf("a status with no usage limit is written as %s (%v); want no usage_limited key", raw, err)
	}
}
