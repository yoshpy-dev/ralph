package org

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// The episodes of the cross-org watch's alerts (plan
// 2026-11-12-watch-alert-dedupe, D1 and D2): a condition that fires again
// while its earlier ALERT is pending, or while its ALERT item is not
// resolved, is recorded as a recurrence of it instead of a new ALERT.
// raiseOrClear and escalateToInbox (watch.go) call these.

// alertIDKey returns the conditionKey part of an alert_id of watchAlertID's
// shape, everything before its last "@", or "" when alertID has no "@". The
// watch finds the open episode of a key by it (pendingAlertForKey,
// openAlertItemForKey; plan 2026-11-12-watch-alert-dedupe, D1 and D2), so
// it must stay in step with watchAlertID, as alertIDConditionType does.
func alertIDKey(alertID string) string {
	at := strings.LastIndex(alertID, "@")
	if at < 0 {
		return ""
	}
	return alertID[:at]
}

// pendingAlertForKey returns the pending alert of status whose alert_id has
// key as its conditionKey part (alertIDKey), or nil when there is none. A
// status file written before plan 2026-11-12-watch-alert-dedupe can hold
// several for one key; then it returns the oldest by the time in its
// alert_id (escalatedTimestamp, ties by alert_id), the one the deadman sweep
// escalates first, so the result does not depend on map order.
func pendingAlertForKey(status *watchStatusFile, key string) *watchPendingAlert {
	var bestID string
	var best *watchPendingAlert
	for alertID, pending := range status.PendingAlerts {
		if pending == nil || alertIDKey(alertID) != key {
			continue
		}
		if best == nil || escalatedTimestamp(alertID) < escalatedTimestamp(bestID) ||
			(escalatedTimestamp(alertID) == escalatedTimestamp(bestID) && alertID < bestID) {
			bestID, best = alertID, pending
		}
	}
	return best
}

// openAlertItemForKey returns the TYPE ALERT item of orgID in items whose
// alert_id has key as its conditionKey part (alertIDKey) and that is not
// resolved (open or acked), and whether there is one. items are in ID order
// (InboxStore.Read); an inbox written before plan
// 2026-11-12-watch-alert-dedupe can hold several such items, and then it
// returns the latest.
func openAlertItemForKey(items []InboxItem, orgID, key string) (InboxItem, bool) {
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if item.Type == protocol.TypeAlert && item.OrgID == orgID && item.State != InboxStateResolved && alertIDKey(item.AlertID) == key {
			return item, true
		}
	}
	return InboxItem{}, false
}

// beforeAppendRecurred is a no-op by default; tests reassign it (the
// package-level seam of afterStopRequestTaken) to change the inbox between
// the read that found an open ALERT item of a key and the append of its
// recurred event, which is the window in which the director can resolve
// the item.
var beforeAppendRecurred = func() {}

// recordRecurrence records a rising edge of (orgID, seatID, condType) as a
// recurrence of that key's open episode, if there is one (plan
// 2026-11-12-watch-alert-dedupe, D1), and reports whether it did:
//
//   - a pending alert of the key (pendingAlertForKey) gets one more
//     Recurrences, which escalateToInbox carries to the alert's item in one
//     recurred event recorded at escalation time, so the item's
//     last_recurred_at for that event is the escalation time, not now;
//   - otherwise an ALERT item of orgID for the key that is not resolved
//     (openAlertItemForKey, in the inbox read once here) gets a recurred
//     event with a new alert_id of the firing (watchAlertID), count 1, and
//     note, and one stderr line names the item.
//
// It reports false, and the caller sends a new ALERT, when there is
// neither, when there is no inbox store or the inbox cannot be read (the
// deadline sweep at the start of the cycle already says so on stderr), and
// when the append fails: the director resolved the item meanwhile
// (ErrInboxResolved) with no line, any other failure with one stderr line.
// A failure goes toward a new ALERT, never toward silence.
func (w *watchRun) recordRecurrence(status *watchStatusFile, orgID, seatID, condType string, now time.Time, note string) bool {
	key := conditionKey(orgID, seatID, condType)
	if pending := pendingAlertForKey(status, key); pending != nil {
		pending.Recurrences++
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
	if !ok {
		return false
	}
	org, seat := PrintableInboxText(orgID, false), PrintableInboxText(seatID, false)
	beforeAppendRecurred()
	if err := w.org.Inbox.AppendRecurred(item.ID, watchAlertID(key, now), note, 1); err != nil {
		if !errors.Is(err, ErrInboxResolved) {
			_, _ = fmt.Fprintf(w.stderr, "watchdog: org=%s seat=%s condition=%s fired again while inbox item %s is not resolved, "+
				"but the recurrence was not recorded (%v); sending a new ALERT\n", org, seat, condType, item.ID, err)
		}
		return false
	}
	_, _ = fmt.Fprintf(w.stderr, "watchdog: org=%s seat=%s condition=%s fired again; recorded as a recurrence on inbox item %s (recurrences=%d)\n",
		org, seat, condType, item.ID, item.Recurrences+1)
	return true
}

// escalateAsRecurrence is escalateToInbox for an alert whose (org, seat,
// condition) has item, an ALERT item that is not resolved and holds another
// alert_id (plan 2026-11-12-watch-alert-dedupe, D2). It appends one recurred
// event to item with alertID and a count of 1 plus the alert's own
// Recurrences (a negative one, which only a hand edit writes, counts as 0),
// writes one stderr line naming item, and reports true; it sends nothing to
// the human, so item keeps its own deadlines. It reports false, so the
// caller records a new item, when the append fails: with no line when the
// director resolved item meanwhile (ErrInboxResolved), with one line
// otherwise.
func (w *watchRun) escalateAsRecurrence(alertID string, pending *watchPendingAlert, item InboxItem, head string) bool {
	beforeAppendRecurred()
	if err := w.org.Inbox.AppendRecurred(item.ID, alertID, "", 1+max(pending.Recurrences, 0)); err != nil {
		if !errors.Is(err, ErrInboxResolved) {
			_, _ = fmt.Fprintf(w.stderr, "%s -- not recorded as a recurrence on inbox item %s, which is not resolved (%v); recording a new item\n",
				head, item.ID, err)
		}
		return false
	}
	_, _ = fmt.Fprintf(w.stderr, "%s -- recorded as a recurrence on inbox item %s (state=%s), an ALERT of the same seat and condition that is not resolved; "+
		"no new item is recorded or sent; read it with: %s\n", head, item.ID, item.State, InboxCommand("show", item.ID, w.org.InboxHintStateDir))
	return true
}
