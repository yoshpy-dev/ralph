package org

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// The usage_limited condition of the cross-org watch (plan
// 2026-11-13-watch-usage-limited). A seat that hits its account's usage
// limit ends its turn, and the watch's stall check holds for it: herdr's
// agent get and the manifest stop changing. Before it raises that stall,
// evaluateSeat (watch.go) reads the seat's screen (readUsageLimitScreen); a
// seat stopped on a usage limit (UsageLimitScreen) gets no stall and is
// collected for its org, and the stall ALERT of a seat whose screen could
// not be read says why (usageLimitUnreadNote). After the
// org's last seat, raiseUsageLimited raises one condition of the org,
// usage_limited, keyed (org, leader, usage_limited) (D1 and D2). Its ALERT
// is about the leader, so the deadman sweep escalates it to the inbox in
// the same cycle (D3), with REASON usage_limit and a body that lists the
// seats, each with its driver and the usage limit line of its screen
// (usageLimitedEscalationBody, D6).

// usageLimitReason is the reason of the escalation of a usage_limited ALERT
// (watchEscalationReason): the REASON of its item, and the reason of the
// escalations.jsonl line of an escalation the inbox could not take.
const usageLimitReason = "usage_limit"

// usageLimitedWhy is what the body of a usage_limited ALERT item says after
// "watchdog: escalated to a person because ". It says that the item is
// about the org, not about the leader its SEAT names (D2).
const usageLimitedWhy = "seats of the org stopped on a usage limit, so the watch did not wait for the leader to act. " +
	"It is a condition of the org, not of the leader that SEAT names: SEATS lists the seats that stopped, " +
	"and each line below gives a seat, its driver, and the usage limit line of its screen, with the reset time it shows"

// minUsageLimitTextRunes is the fewest runes of a seat's usage limit line
// fitUsageLimitedSeats keeps when it cuts the lines to fit: enough for the
// sentence and, most of the time, the reset time after it. A seat that
// cannot get that many is named in a "+N more" entry instead.
const minUsageLimitTextRunes = 60

// usageLimitUnreadMaxRunes is the most runes of the error
// usageLimitUnreadNote puts in a stall ALERT, the "…" that marks a cut
// included.
const usageLimitUnreadMaxRunes = 200

// watchUsageLimitedSeat is a seat of an org stopped on a usage limit in a
// cycle: its seat_id, its driver, and the usage limit line of its screen
// (UsageLimitScreen's line: one line of at most usageLimitLineMaxRunes runes
// with no control character).
type watchUsageLimitedSeat struct {
	Seat   string `json:"seat"`
	Driver string `json:"driver"`
	Line   string `json:"line"`
}

// watchUsageLimitedSeats is what a usage_limited pending alert keeps for its
// item's body (watchPendingAlert.UsageLimited): the seats of the org stopped
// on a usage limit, sorted by seat_id.
type watchUsageLimitedSeats struct {
	Seats []watchUsageLimitedSeat `json:"seats"`
}

// limitedSeatEntries returns the seats p keeps for the org condition
// usage_limited, each with its driver and line, or nil when it keeps none
// (p.UsageLimited is nil).
func (p *watchPendingAlert) limitedSeatEntries() []watchUsageLimitedSeat {
	if p.UsageLimited == nil {
		return nil
	}
	return p.UsageLimited.Seats
}

// isUsageLimitedAlert reports whether alertID is the alert_id of an ALERT of
// the org condition usage_limited (alertIDConditionType).
func isUsageLimitedAlert(alertID string) bool {
	return alertIDConditionType(alertID) == condUsageLimited
}

// readUsageLimitScreen reads the screen of s, a seat whose stall holds this
// cycle before the leader's exemption (evaluateSeat), and returns s's entry
// of the org's usage_limited and true when the screen shows s stopped on a
// usage limit (UsageLimitScreen with s's driver); evaluateSeat adds the
// entry to the cycle's usageLimited. It reads resumeReadLines lines of the
// pane the manifest records for s, once, under the deadline of each driver
// call of the pulse (callWithTimeout; D4 and AC7), and changes nothing.
//
// A screen that is no usage limit (a network error, a screen
// UsageLimitScreen does not know) returns false and no error, and so does
// an Org with no herdr, which reads nothing. A screen that could not be read
// returns false and why: no pane is recorded for s, or the read failed or
// was cut off by the deadline, which evaluateSeat puts in the seat's stall
// ALERT (usageLimitUnreadNote). Either way the seat's stall is decided as
// before (AC6).
func (w *watchRun) readUsageLimitScreen(s SeatStatus) (watchUsageLimitedSeat, bool, error) {
	if w.org.Herdr == nil {
		return watchUsageLimitedSeat{}, false, nil
	}
	if s.PaneID == "" {
		return watchUsageLimitedSeat{}, false, errors.New("no pane is recorded for the seat")
	}
	var screen string
	if err := w.org.callWithTimeout(func(ctx context.Context) error {
		var err error
		screen, err = w.org.Herdr.PaneRead(ctx, s.PaneID, resumeReadLines)
		return err
	}); err != nil {
		return watchUsageLimitedSeat{}, false, fmt.Errorf("herdr pane read %s failed: %w", s.PaneID, err)
	}
	line, limited := UsageLimitScreen(s.Driver, screen)
	if !limited {
		return watchUsageLimitedSeat{}, false, nil
	}
	return watchUsageLimitedSeat{Seat: s.SeatID, Driver: s.Driver, Line: line}, true, nil
}

// usageLimitUnreadNote is what evaluateSeat adds to the message of a seat's
// stall ALERT when readUsageLimitScreen could not read the seat's screen
// (err), so that whoever reads the ALERT can tell that the watch could not
// see whether the seat stopped on a usage limit: a line break, then
// "screen not read for the usage-limit check: " and err as one line with no
// control character (usageLimitLine), cut to usageLimitUnreadMaxRunes runes.
// It is "" for a nil err: a screen that was read adds nothing, whatever it
// shows.
func usageLimitUnreadNote(err error) string {
	if err == nil {
		return ""
	}
	return "\nscreen not read for the usage-limit check: " + cutUsageLimitText(usageLimitLine(err.Error()), usageLimitUnreadMaxRunes)
}

// usageLimitedBannerSeats is what the stderr banner of an escalation the
// inbox could not take (escalateWithoutInbox) adds after its reason for an
// alert of the org condition usage_limited: " seats=" and the seats pending
// keeps, escaped (PrintableInboxText) and joined with ",", so the banner
// names the seats that stopped and does not read as about the leader alone.
// It is "" for an alert of any other condition, and for a usage_limited
// alert that keeps no seat, which escalateToInbox refuses for that
// (checkUsageLimitedSeats).
func usageLimitedBannerSeats(alertID string, pending *watchPendingAlert) string {
	seats := pending.limitedSeatEntries()
	if !isUsageLimitedAlert(alertID) || len(seats) == 0 {
		return ""
	}
	ids := make([]string, len(seats))
	for i, s := range seats {
		ids[i] = PrintableInboxText(s.Seat, false)
	}
	return " seats=" + strings.Join(ids, ",")
}

// raiseUsageLimited raises, keeps, or clears orgID's usage_limited condition
// from limited, the seats of the org stopped on a usage limit this cycle
// (plan 2026-11-13-watch-usage-limited, D1, D2, AC4, and AC5). Its key is
// conditionKey(orgID, LeaderIdentity, condUsageLimited), and
// status.UsageLimitedSeats holds the seats of the last cycle while it is
// active.
//
//   - No seat: the condition clears, if it was active, and the set is
//     emptied, so the next seat that stops raises it again.
//   - The condition was active and every seat is in the set (the same
//     seats, or fewer): the set becomes limited's, and nothing else
//     happens, as raiseOrClear dedupes an active key.
//   - Otherwise it rises: it was not active, or a seat not in the set
//     stopped, a new member, which is an edge too. The set becomes
//     limited's, and the key is recorded as Active from now. The edge is a
//     recurrence of the condition's earlier ALERT when
//     recordUsageLimitRecurrence records it so; otherwise a new ALERT is
//     sent to the leader (sendAlert, with usageLimitedAlertMessage), and its
//     pending alert keeps limited, which the deadman sweep escalates to a
//     new item in this cycle.
//
// No OnSemanticTrigger is fired: the screen already said why the seats
// stopped, which is what the on-demand watcher would judge.
func (w *watchRun) raiseUsageLimited(status *watchStatusFile, orgID string, limited []watchUsageLimitedSeat, now time.Time) {
	key := conditionKey(orgID, LeaderIdentity, condUsageLimited)
	rec := status.Conditions[key]
	wasActive := rec != nil && rec.Active
	if len(limited) == 0 {
		if wasActive {
			rec.Active = false
		}
		status.UsageLimitedSeats = nil
		return
	}
	slices.SortFunc(limited, func(a, b watchUsageLimitedSeat) int { return strings.Compare(a.Seat, b.Seat) })
	added := limited
	if wasActive {
		added = slices.DeleteFunc(slices.Clone(limited), func(s watchUsageLimitedSeat) bool {
			return slices.Contains(status.UsageLimitedSeats, s.Seat)
		})
	}
	status.UsageLimitedSeats = usageLimitedSeatIDs(limited)
	if len(added) == 0 {
		return // still active with no new member: dedupe
	}
	status.Conditions[key] = &watchConditionRecord{Active: true, FirstTS: now.UTC().Format(time.RFC3339)}
	if w.recordUsageLimitRecurrence(status, orgID, limited, added, wasActive, now) {
		return
	}
	pending := w.sendAlert(status, orgID, LeaderIdentity, condUsageLimited, usageLimitedAlertMessage(orgID, limited), now)
	pending.UsageLimited = &watchUsageLimitedSeats{Seats: limited}
}

// recordUsageLimitRecurrence records an edge of orgID's usage_limited as a
// recurrence of the condition's earlier ALERT, when there is one that may
// take it, and reports whether it did. limited is the org's seats stopped on
// a usage limit this cycle, and added those of them the edge brought: all of
// them for a rise, and only the new members when the condition was already
// active (grown). It follows recordRecurrence (watch_recurrence.go), with
// two differences for the seats:
//
//   - a pending alert of the key, one the deadman sweep has not escalated
//     yet (it escalates such an alert in the cycle that raised it, so one is
//     left only by a cycle that ended before that sweep), takes one more
//     Recurrences and limited as its seats, so the item it becomes lists
//     them all;
//   - otherwise an ALERT item of the key that is not resolved
//     (openAlertItemForKey) gets a recurred event with a new alert_id of the
//     firing (watchAlertID), count 1, and a note that names the seats of
//     added (usageLimitRecurNote), and one stderr line names the item. A new
//     member is recorded so only on an item that is open, which the director
//     or the human has not acknowledged yet and reads with the note; an
//     acked item was read already, so it reports false, and the new members
//     get a new item (AC4). The state of the read is a fast path: the append
//     of a grown condition (AppendRecurredIfOpen) checks again under the
//     inbox lock that the item is open, so an ack between the read and the
//     append gives a new item too. A rise after the condition cleared is
//     recorded on an item that is open or acked (AppendRecurred), by
//     recordRecurrence's rule (AC5).
//
// It reports false, toward a new ALERT, when there is no item of the key
// that is not resolved, no inbox store, or an inbox that cannot be read (the
// deadline sweep at the start of the cycle already says so on stderr), and
// when the append fails: the item was resolved meanwhile (ErrInboxResolved)
// or, for a grown condition, acked meanwhile (ErrInboxAcked) with no line,
// any other failure with one stderr line.
func (w *watchRun) recordUsageLimitRecurrence(status *watchStatusFile, orgID string, limited, added []watchUsageLimitedSeat, grown bool, now time.Time) bool {
	key := conditionKey(orgID, LeaderIdentity, condUsageLimited)
	if _, pending := pendingAlertForKey(status, key); pending != nil {
		pending.Recurrences++
		pending.UsageLimited = &watchUsageLimitedSeats{Seats: limited}
		return true
	}
	if w.org.Inbox == nil {
		return false
	}
	inbox, err := w.org.Inbox.Read()
	if err != nil {
		return false
	}
	item, ok := openAlertItemForKey(inbox.Items, orgID, key)
	if !ok || (grown && item.State != InboxStateOpen) {
		return false
	}
	org, seats := PrintableInboxText(orgID, false), usageLimitedSeatList(added)
	recur := w.org.Inbox.AppendRecurred
	if grown {
		recur = w.org.Inbox.AppendRecurredIfOpen
	}
	beforeAppendRecurred()
	if err := recur(item.ID, watchAlertID(key, now), usageLimitRecurNote(added), 1); err != nil {
		if !errors.Is(err, ErrInboxResolved) && !errors.Is(err, ErrInboxAcked) {
			_, _ = fmt.Fprintf(w.stderr, "watchdog: org=%s condition=%s: seats %s stopped on a usage limit while inbox item %s is not resolved, "+
				"but the recurrence was not recorded (%v); sending a new ALERT\n", org, condUsageLimited, seats, item.ID, err)
		}
		return false
	}
	_, _ = fmt.Fprintf(w.stderr, "watchdog: org=%s condition=%s: seats %s stopped on a usage limit; recorded as a recurrence on inbox item %s (recurrences=%d)\n",
		org, condUsageLimited, seats, item.ID, item.Recurrences+1)
	return true
}

// usageLimitRecurNote is the note of the recurred event of a usage_limited
// edge: "stopped on a usage limit: " and an entry for each seat of added,
// `<seat> (<driver>): <line>`, joined with "; ", within InboxNoteMaxRunes
// (fitUsageLimitedSeats), so AppendRecurred takes it: one line with no
// control character.
func usageLimitRecurNote(added []watchUsageLimitedSeat) string {
	const lead = "stopped on a usage limit: "
	return lead + fitUsageLimitedSeats(added, "; ", InboxNoteMaxRunes-utf8.RuneCountInString(lead))
}

// usageLimitedAlertMessage is the ALERT the watch sends the leader over agmsg
// when orgID's usage_limited rises (raiseUsageLimited): the headers of the
// watch's ALERT, with SEAT the leader (the seat of the condition's key) and
// SEATS the seats that stopped, then a line saying that the seats of the org
// stopped on a usage limit, and a line for each seat
// (withUsageLimitedLines), so it passes sendAlert's protocol check.
func usageLimitedAlertMessage(orgID string, limited []watchUsageLimitedSeat) string {
	head := fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: %s\nSEATS: %s\n\n"+
		"seats of org %s stopped on a usage limit; it is a condition of the org, not of the leader that SEAT names, "+
		"and each line below gives a seat, its driver, and the usage limit line of its screen:",
		orgID, LeaderIdentity, condUsageLimited, usageLimitedSeatList(limited), orgID)
	return withUsageLimitedLines(head, limited)
}

// usageLimitedEscalationBody is watchEscalationBody for an alert of the org
// condition usage_limited (plan 2026-11-13-watch-usage-limited, D2, D3, D6,
// and AC10): the headers of every watch ALERT item (ORG_ID, SEAT,
// CONDITION, REASON, ALERT_ID, ALERT_TS), with SEAT the leader and REASON
// usage_limit, then SEATS, the seats pending keeps joined with ", "; after
// a blank line, "watchdog: escalated to a person because " and
// usageLimitedWhy, then a line for each seat (withUsageLimitedLines).
// escalateToInbox checks the seats first (checkUsageLimitedSeats).
func usageLimitedEscalationBody(orgID, alertID string, pending *watchPendingAlert) string {
	seats := pending.limitedSeatEntries()
	head := fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: %s\nREASON: %s\nALERT_ID: %s\nALERT_TS: %s\nSEATS: %s\n\n"+
		"watchdog: escalated to a person because %s",
		orgID, pending.Subject, condUsageLimited, usageLimitReason, alertID, pending.TS, usageLimitedSeatList(seats), usageLimitedWhy)
	return withUsageLimitedLines(head, seats)
}

// withUsageLimitedLines returns head and, on the lines after it, an entry
// for each seat of seats (fitUsageLimitedSeats with "\n"), the whole text
// within protocol.DefaultMaxBodyChars runes, the cap of a message's body
// (protocol.Validate counts only the part after the headers, so the text
// passes it too). A head that leaves no room is returned alone; its SEATS
// header names the seats.
func withUsageLimitedLines(head string, seats []watchUsageLimitedSeat) string {
	lines := fitUsageLimitedSeats(seats, "\n", protocol.DefaultMaxBodyChars-utf8.RuneCountInString(head)-1)
	if lines == "" {
		return head
	}
	return head + "\n" + lines
}

// fitUsageLimitedSeats returns an entry for each seat of seats,
// `<seat> (<driver>): <line>`, joined with sep, in at most budget runes. The
// seat and the driver are escaped (PrintableInboxText) and the line made
// one line with no control character again (usageLimitLine), since a
// pending alert's seats come from the watch-status file. When the entries
// do not fit whole, every line is cut to the longest length at which they
// fit (usageLimitTextCap), with "…" where it was cut; when that length would
// be below minUsageLimitTextRunes, the last seats get no entry of their own
// and are named after the others in one "+N more: a, b" entry, as few of
// them as can be. When not even the "+N more" entry of every seat fits, that
// entry is cut to budget. It returns "" for no seats.
func fitUsageLimitedSeats(seats []watchUsageLimitedSeat, sep string, budget int) string {
	for shown := len(seats); shown > 0; shown-- {
		if text, ok := usageLimitedEntriesWithin(seats[:shown], usageLimitedMore(seats[shown:]), sep, budget); ok {
			return text
		}
	}
	return cutUsageLimitText(usageLimitedMore(seats), budget)
}

// usageLimitedEntriesWithin returns the entries of seats, which is not
// empty, and then more, when it is not empty, joined with sep, and true when
// they fit in budget runes with each line cut to no fewer than
// minUsageLimitTextRunes runes (usageLimitTextCap).
func usageLimitedEntriesWithin(seats []watchUsageLimitedSeat, more, sep string, budget int) (string, bool) {
	prefixes := make([]string, len(seats))
	lines := make([]string, len(seats))
	lens := make([]int, len(seats))
	fixed := utf8.RuneCountInString(more) + utf8.RuneCountInString(sep)*(len(seats)-1)
	if more != "" {
		fixed += utf8.RuneCountInString(sep)
	}
	for i, s := range seats {
		prefixes[i] = fmt.Sprintf("%s (%s): ", PrintableInboxText(s.Seat, false), PrintableInboxText(s.Driver, false))
		lines[i] = usageLimitLine(s.Line)
		lens[i] = utf8.RuneCountInString(lines[i])
		fixed += utf8.RuneCountInString(prefixes[i])
	}
	textCap, ok := usageLimitTextCap(lens, budget-fixed)
	if !ok {
		return "", false
	}
	entries := make([]string, 0, len(seats)+1)
	for i := range seats {
		entries = append(entries, prefixes[i]+cutUsageLimitText(lines[i], textCap))
	}
	if more != "" {
		entries = append(entries, more)
	}
	return strings.Join(entries, sep), true
}

// usageLimitTextCap returns the longest length to which cutting each line
// whose length in runes is in lens, which is not empty, makes them fit in
// avail runes together, and false when they do not fit even when cut to
// minUsageLimitTextRunes (a line shorter than that is never cut).
func usageLimitTextCap(lens []int, avail int) (int, bool) {
	total := func(textCap int) int {
		n := 0
		for _, l := range lens {
			n += min(l, textCap)
		}
		return n
	}
	textCap := slices.Max(lens)
	for textCap > minUsageLimitTextRunes && total(textCap) > avail {
		textCap--
	}
	return textCap, total(textCap) <= avail
}

// cutUsageLimitText returns text cut to n runes, the last of them "…", as
// usageLimitLine cuts a line: text itself when it is not longer than n,
// and "" for n below 1.
func cutUsageLimitText(text string, n int) string {
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	if n < 1 {
		return ""
	}
	return strings.TrimRight(string(runes[:n-1]), " ") + "…"
}

// usageLimitedMore is the entry that names rest, the seats that get no entry
// of their own: "+N more: a, b", or "" when rest is empty.
func usageLimitedMore(rest []watchUsageLimitedSeat) string {
	if len(rest) == 0 {
		return ""
	}
	return fmt.Sprintf("+%d more: %s", len(rest), usageLimitedSeatList(rest))
}

// usageLimitedSeatIDs returns the seat_ids of seats, in their order:
// status.UsageLimitedSeats.
func usageLimitedSeatIDs(seats []watchUsageLimitedSeat) []string {
	ids := make([]string, len(seats))
	for i, s := range seats {
		ids[i] = s.Seat
	}
	return ids
}

// usageLimitedSeatList returns the seat_ids of seats, escaped
// (PrintableInboxText), joined with ", ": the SEATS header, a "+N more"
// entry, and the stderr lines of recordUsageLimitRecurrence.
func usageLimitedSeatList(seats []watchUsageLimitedSeat) string {
	ids := make([]string, len(seats))
	for i, s := range seats {
		ids[i] = PrintableInboxText(s.Seat, false)
	}
	return strings.Join(ids, ", ")
}

// checkUsageLimitedSeats returns an error when the seats of a usage_limited
// pending alert from the watch-status file are not ones raiseUsageLimited
// writes: none at all, or a seat or a driver that is not an identifier
// (ValidateIdentifier, whose error quotes the value with %q). The seats go
// into the item's SEATS header and each seat and its driver start a line,
// so a line break in one must not add a header or a line
// (checkWatchEscalationValues). The usage limit lines are made one line
// again when the body is built (fitUsageLimitedSeats).
func checkUsageLimitedSeats(seats []watchUsageLimitedSeat) error {
	if len(seats) == 0 {
		return errors.New("the usage_limited ALERT in the watch-status file names no seat")
	}
	for _, s := range seats {
		if err := ValidateIdentifier("seat_id", s.Seat); err != nil {
			return fmt.Errorf("a seat of the usage_limited ALERT in the watch-status file is not a seat: %w", err)
		}
		if err := ValidateIdentifier("driver", s.Driver); err != nil {
			return fmt.Errorf("a seat of the usage_limited ALERT in the watch-status file has a driver that is not one: %w", err)
		}
	}
	return nil
}
