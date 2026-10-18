package org

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"time"
)

// The inbox deadlines are the human path of the items escalate leaves to
// the director, and the second net of those it sends at once (FR-6 of
// docs/specs/2026-10-07-org-multi-org-director.md; escalate.go says which
// is which). `ralph org watch --all` runs SweepInboxDeadlines once per cycle
// and sends an item when one of its deadlines has passed:
//
//   - ack deadline: the item is open, Config.Director.AckMinutes have passed
//     since it was escalated, and it has no notified event yet (escalate
//     left it to the director, or could not complete its notification). An
//     item can go from open straight to resolved, so the rule reads the
//     open state, not "not acked".
//   - resolve deadline: the item is not resolved,
//     Config.Director.ResolveMinutes have passed since it was escalated,
//     and it has no notified event with InboxReasonResolveDeadline.
//
// Both are computed from inbox.jsonl on every sweep, so a restarted watch
// sends an item whose deadline passed while no watch ran, and the notified
// event of a send keeps the next sweep (in this process or another) from
// sending it again.

// Reasons of a deadline send, written to its escalations.jsonl line and its
// notified event.
const (
	InboxReasonAckDeadline     = "inbox_ack_deadline"
	InboxReasonResolveDeadline = "inbox_resolve_deadline"
)

// InboxDeadlineDue returns the deadline item has passed at now and is not
// yet sent for: InboxReasonResolveDeadline, InboxReasonAckDeadline, or ""
// when nothing is due. It reads only the item's state, its escalated_at,
// and its notified events, never its Delivery, so an item escalate left to
// the director (InboxDeliveryDeadline) is due exactly as one whose send at
// once was not completed. The rules, in this order:
//
//   - a resolved item has nothing due;
//   - the resolve deadline is due when more than resolveMinutes have passed
//     since escalated_at and no notified event has
//     InboxReasonResolveDeadline. It comes first: when both deadlines have
//     passed only it is sent, and its notified event also ends the ack
//     deadline;
//   - the ack deadline is due when the item is open, more than ackMinutes
//     have passed, and it has no notified event at all.
//
// A deadline is passed only strictly after it (exactly ackMinutes after
// escalated_at is not yet past). The error is set only when escalated_at
// (an RFC3339 time) does not parse, which only a hand-edited inbox.jsonl
// line can cause. It does not read Woken or WokenAt either: a push wake of
// the director (WakeDirector) does not stop the ack deadline, so an item the
// director was woken for but did not ack still reaches the human (plan
// 2026-10-18-org-director-wake, D10).
func InboxDeadlineDue(item InboxItem, now time.Time, ackMinutes, resolveMinutes int) (string, error) {
	if item.State == InboxStateResolved {
		return "", nil
	}
	escalated, err := time.Parse(time.RFC3339, item.EscalatedAt)
	if err != nil {
		return "", fmt.Errorf("org: inbox deadlines: %s has an escalated_at that is not an RFC3339 time (%q)", item.ID, item.EscalatedAt)
	}
	since := now.Sub(escalated)
	if since > time.Duration(resolveMinutes)*time.Minute && !inboxNotifiedFor(item, InboxReasonResolveDeadline) {
		return InboxReasonResolveDeadline, nil
	}
	if item.State == InboxStateOpen && since > time.Duration(ackMinutes)*time.Minute && !item.Notified {
		return InboxReasonAckDeadline, nil
	}
	return "", nil
}

// inboxNotifiedFor reports whether item has a notified event with reason.
func inboxNotifiedFor(item InboxItem, reason string) bool {
	return slices.ContainsFunc(item.Events, func(ev InboxEvent) bool {
		return ev.Event == InboxEventNotified && ev.Reason == reason
	})
}

// InboxDeadlineMemory is what one watch process remembers between
// SweepInboxDeadlines calls, in memory only, so a restarted watch starts
// with nothing remembered. It holds the sends that failed, so a ledger
// whose escalations.jsonl or inbox.jsonl cannot be written does not ring
// the desktop every cycle, and the items whose escalated_at does not parse,
// so each is warned about once. The zero value is ready to use. It is not
// safe for concurrent use.
type InboxDeadlineMemory struct {
	// failedAt is the time of the last failed send of each (ID, reason).
	failedAt map[inboxDeadlineKey]time.Time
	// warned holds the IDs whose escalated_at warning was written.
	warned map[string]bool
}

type inboxDeadlineKey struct {
	id, reason string
}

// holdsBack reports whether the send key failed less than retry ago.
func (m *InboxDeadlineMemory) holdsBack(key inboxDeadlineKey, now time.Time, retry time.Duration) bool {
	at, ok := m.failedAt[key]
	return ok && now.Sub(at) < retry
}

func (m *InboxDeadlineMemory) recordFailure(key inboxDeadlineKey, now time.Time) {
	if m.failedAt == nil {
		m.failedAt = map[inboxDeadlineKey]time.Time{}
	}
	m.failedAt[key] = now
}

// warnOnce reports whether the escalated_at warning for id is still to be
// written, and remembers that it now is.
func (m *InboxDeadlineMemory) warnOnce(id string) bool {
	if m.warned[id] {
		return false
	}
	if m.warned == nil {
		m.warned = map[string]bool{}
	}
	m.warned[id] = true
	return true
}

// InboxDeadlineSend names one item and the deadline (reason) it was due
// for in a sweep.
type InboxDeadlineSend struct {
	ID     string
	Reason string
	// Err is why the send was not completed; it is set only in
	// InboxDeadlineSweep.Failed.
	Err error
}

// InboxDeadlineSweep is what one SweepInboxDeadlines call did, each list in
// item ID order. Err is set when the sweep could not start (no inbox store,
// a deadline below one minute, an inbox that cannot be read); the lists are
// empty then. The caller (the watch) logs Failed and Err and goes on.
type InboxDeadlineSweep struct {
	// Sent are the items sent to the human path: the escalations.jsonl line
	// and the notified event were both written.
	Sent []InboxDeadlineSend
	// Failed are the sends that were not completed, with the error.
	Failed []InboxDeadlineSend
	// HeldBack are the due items not tried because the same send failed
	// less than Config.Director.AckMinutes ago in this memory.
	HeldBack []InboxDeadlineSend
	// InvalidEscalatedAt are the IDs skipped because their escalated_at
	// does not parse.
	InvalidEscalatedAt []string
	Err                error
}

// SweepInboxDeadlines reads the inbox once and sends every item a deadline
// is due for (InboxDeadlineDue at now, with Config.Director's AckMinutes
// and ResolveMinutes) to the human path, with the deadline as the reason.
// The banner lines go to banner (nil means os.Stderr).
//
// mem carries what this watch process remembers between sweeps (nil means
// nothing is remembered, so every due item is tried and every bad
// escalated_at is warned about). A send that failed is not tried again for
// the same item and reason until AckMinutes have passed since that attempt.
// A send that succeeded needs no memory: its notified event keeps the item
// from being due for the same reason again. An item whose escalated_at does
// not parse is skipped, with one warning line on banner the first time mem
// sees it. Like InboxDeadlineDue, it does not read whether the director was
// woken for an item (its woken events), so a woken item is sent at its
// deadline as any other; the watch runs it before the wake (plan
// 2026-10-18-org-director-wake, D6 and D10).
func (o *Org) SweepInboxDeadlines(now time.Time, mem *InboxDeadlineMemory, banner io.Writer) InboxDeadlineSweep {
	if o.Inbox == nil {
		return InboxDeadlineSweep{Err: errors.New("org: inbox deadlines: no inbox store is configured")}
	}
	ackMinutes, resolveMinutes := o.Config.Director.AckMinutes, o.Config.Director.ResolveMinutes
	if ackMinutes < 1 || resolveMinutes < 1 {
		return InboxDeadlineSweep{Err: fmt.Errorf("org: inbox deadlines: [org.director] ack_minutes (%d) and resolve_minutes (%d) must both be >= 1", ackMinutes, resolveMinutes)}
	}
	inbox, err := o.Inbox.Read()
	if err != nil {
		return InboxDeadlineSweep{Err: fmt.Errorf("org: inbox deadlines: %w", err)}
	}
	if mem == nil {
		mem = &InboxDeadlineMemory{}
	}
	banner = bannerOrStderr(banner)
	retry := time.Duration(ackMinutes) * time.Minute
	var sweep InboxDeadlineSweep
	for _, item := range inbox.Items {
		reason, err := InboxDeadlineDue(item, now, ackMinutes, resolveMinutes)
		if err != nil {
			sweep.InvalidEscalatedAt = append(sweep.InvalidEscalatedAt, item.ID)
			if mem.warnOnce(item.ID) {
				_, _ = fmt.Fprintf(banner, "%v; its deadlines are skipped\n", err)
			}
			continue
		}
		if reason == "" {
			continue
		}
		key := inboxDeadlineKey{id: item.ID, reason: reason}
		send := InboxDeadlineSend{ID: item.ID, Reason: reason}
		if mem.holdsBack(key, now, retry) {
			sweep.HeldBack = append(sweep.HeldBack, send)
			continue
		}
		if err := o.sendInboxItemToHuman(item.OrgID, item.ID, item.Type, reason, banner); err != nil {
			mem.recordFailure(key, now)
			send.Err = err
			sweep.Failed = append(sweep.Failed, send)
			continue
		}
		sweep.Sent = append(sweep.Sent, send)
	}
	return sweep
}

// inboxDeadlineHumanText is the text of a deadline send (reason
// InboxReasonAckDeadline or InboxReasonResolveDeadline): the banner line
// names the deadline it missed and its length from Config.Director, and the
// desktop notification, titled after the watch that sends it, names the
// org, the ID, the TYPE, and the deadline, never the body. orgID and typ
// are guarded the way sendInboxItemToHuman's doc says.
func (o *Org) inboxDeadlineHumanText(orgID, id, typ, reason string) inboxHumanText {
	missed, minutes := "not acked", o.Config.Director.AckMinutes
	if reason == InboxReasonResolveDeadline {
		missed, minutes = "not resolved", o.Config.Director.ResolveMinutes
	}
	return inboxHumanText{
		banner: fmt.Sprintf("ORG INBOX DEADLINE: org=%s item=%s type=%s -- %s within %dm (%s); read it with: %s",
			PrintableInboxText(orgID, false), id, PrintableInboxText(typ, false), missed, minutes, reason, InboxCommand("show", id, o.InboxHintStateDir)),
		title:   inboxNotifyTitleWatch,
		desktop: fmt.Sprintf("org %s: %s (%s) %s within %d min", notifiableOrgID(orgID), id, notifiableType(typ), missed, minutes),
	}
}
