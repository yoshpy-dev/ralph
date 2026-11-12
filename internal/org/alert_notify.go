package org

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// The desktop notification of the TYPE ALERT inbox items that `ralph org
// watch --all` sends to the human (plan 2026-11-12-alert-notify-batch, D2,
// D2b, D3, and D4). The watch sends them in two kinds of step: one deadline
// sweep (SweepInboxDeadlines, step 1 of RunWatchAll's cycle) and one org's
// deadman step (checkDeadman, at the end of that org's pulse). Each step
// collects the ALERT items it sends in one alertNotifyBatch and, at its end,
// sends one desktop notification for all of them (flushAlertNotifyBatch),
// or none with [org.watchdog].alert_notify = "off". The escalations.jsonl
// line and the banner of each item are still written at once, item by item.
// A leader's items, `ralph org inbox notify`, the escalation at the start of
// a cycle of an alert the inbox already holds
// (reconcilePendingAlertsWithInbox), and the path of an escalation the inbox
// could not take pass no batch, so they are notified item by item and do not
// read alert_notify.

// alertNotifyBatch collects the TYPE ALERT items one sending step of the
// watch sends to the human. sendInboxItemToHuman adds an item to it after it
// has written the item's escalations.jsonl line and banner, and sends no
// desktop notification and appends no notified event for it;
// flushAlertNotifyBatch does both at the end of the step. A nil batch takes
// nothing (takes). A watch that ends between the two leaves the items it
// added with no notified event, so the next watch sends them at their ack
// deadline (D4). A batch is not safe for concurrent use.
type alertNotifyBatch struct {
	items []alertNotifyItem
}

// alertNotifyItem is one ALERT item of a batch: the values of its
// notification text (inboxHumanTextFor) and of its notified event.
type alertNotifyItem struct {
	orgID, id, alertID, reason string
	// lineWritten reports that the item's escalations.jsonl line was
	// written. An item whose line was not is still in the notification, as
	// a send item by item notifies it, but gets no notified event, so it
	// stays unnotified in the inbox.
	lineWritten bool
}

// alertNotifyResult is the result of appending the notified event of one
// item of a batch: err is AppendNotified's error, nil when it was appended.
type alertNotifyResult struct {
	orgID, id string
	err       error
}

// takes reports whether the batch takes the send of an item of TYPE typ: b
// is not nil and typ is ALERT.
func (b *alertNotifyBatch) takes(typ string) bool {
	return b != nil && typ == protocol.TypeAlert
}

func (b *alertNotifyBatch) add(item alertNotifyItem) {
	b.items = append(b.items, item)
}

// flushAlertNotifyBatch ends the sending step of batch: it sends one desktop
// notification for the items batch holds (alertNotifyText) and then appends
// the notified event of each item whose escalations.jsonl line was written,
// with the item's own reason and the notification's result as osascript.
// With Config.Watchdog.AlertNotify set to config.AlertNotifyOff it sends no
// notification, and the result recorded is inboxNotifyOff, so the ack
// deadline does not send the items again (D3). Any other value, the empty
// string of a Config built without Default included, sends the one
// notification. It returns the result of each append, in the order the
// items were added, for the caller to act on, and empties batch. A nil or
// empty batch sends and appends nothing.
func (o *Org) flushAlertNotifyBatch(batch *alertNotifyBatch) []alertNotifyResult {
	if batch == nil || len(batch.items) == 0 {
		return nil
	}
	items := batch.items
	batch.items = nil
	osascript := inboxNotifyOff
	if o.Config.Watchdog.AlertNotify != config.AlertNotifyOff {
		title, message := o.alertNotifyText(items)
		osascript = o.inboxDesktopNotify(title, message)
	}
	var results []alertNotifyResult
	for _, item := range items {
		if !item.lineWritten {
			continue
		}
		results = append(results, alertNotifyResult{
			orgID: item.orgID, id: item.id,
			err: o.Inbox.AppendNotified(item.id, item.reason, osascript),
		})
	}
	return results
}

// alertNotifyText is the title and the text of the one desktop notification
// of items (not empty). For one item they are those of a send of that item
// alone (inboxHumanTextFor). For two or more the title is
// inboxNotifyTitleWatch and the text names how many items there are and the
// oldest one, the item with the smallest e<N>, and its org, guarded as
// inboxHumanTextFor guards it (notifiableOrgID), never a body. The IDs come
// from the inbox, which writes and reads only IDs of the form e<N>.
func (o *Org) alertNotifyText(items []alertNotifyItem) (title, message string) {
	if len(items) == 1 {
		item := items[0]
		text := o.inboxHumanTextFor(item.orgID, item.id, protocol.TypeAlert, item.alertID, item.reason)
		return text.title, text.desktop
	}
	oldest := slices.MinFunc(items, func(a, b alertNotifyItem) int {
		na, _ := inboxIDNumber(a.id)
		nb, _ := inboxIDNumber(b.id)
		return cmp.Compare(na, nb)
	})
	return inboxNotifyTitleWatch, fmt.Sprintf("%d ALERT items; the oldest is %s of org %s", len(items), oldest.id, notifiableOrgID(oldest.orgID))
}
