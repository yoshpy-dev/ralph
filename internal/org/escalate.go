package org

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// Escalate, NotifyInboxItem, and WaitInbox are the org layer of `ralph org
// escalate`, `ralph org inbox notify`, and `ralph org wait --inbox` (FR-5 and
// FR-9 of docs/specs/2026-10-07-org-multi-org-director.md). With the
// director switch on (Config.Director.Enabled) and a cross-org watch that
// accepts work (Org.EscalateWatch), an escalated item is only recorded, with
// delivery InboxDeliveryDeadline: the director reads it from the inbox, and
// when it is still open after the ack deadline, `ralph org watch --all`
// sends it to the human (deadline.go). Otherwise it is also sent to the
// human path at once (delivery InboxDeliveryNow): one line in
// escalations.jsonl, a banner on the caller's stderr, and a best-effort
// desktop notification. The notified event in the inbox records that the
// item was sent, so the ack deadline does not send it again. Whether a
// director is registered (director.go) changes none of this (plan
// 2026-10-14-org-director-attach, D2).

// escalateTypes are the TYPEs Escalate accepts: a leader raises an item
// when it needs a decision (QUESTION), cannot proceed (BLOCKED), or has
// finished, for example opened a PR (RESULT). BLOCKED and RESULT need a
// TASK_ID; protocol.Validate enforces that.
var escalateTypes = []string{protocol.TypeQuestion, protocol.TypeBlocked, protocol.TypeResult}

// inboxItemTypes are the TYPEs an inbox item is written with: escalateTypes
// from a leader's escalate, and ALERT, which only the cross-org watch
// writes, for each of its escalations (plan
// 2026-10-23-watch-escalation-inbox, D2). Escalate still refuses ALERT
// (escalateTypes); this list is what the human path accepts as a TYPE
// (notifiableType).
var inboxItemTypes = append(slices.Clone(escalateTypes), protocol.TypeAlert)

// Reasons of the human-path records, written to the escalations.jsonl line
// and the notified event of a send. Escalate writes
// InboxReasonDirectorDisabled when Config.Director.Enabled is false and
// InboxReasonNoWatch when it is true but no cross-org watch accepts work, so
// no one would check the ack deadline; NotifyInboxItem writes
// InboxReasonManualNotify. inboxReasonNotRecorded goes only on the
// best-effort escalations.jsonl line of an escalate the inbox could not
// record (it has no inbox_id). The deadline reasons are in deadline.go.
//
// Ledgers written before stage 8a hold inbox_no_director on the notified
// events and the escalations.jsonl lines of escalate and inbox notify.
// Readers treat every notified reason alike (InboxItem.Notified); the only
// reason looked up is InboxReasonResolveDeadline, by inboxNotifiedFor.
const (
	InboxReasonDirectorDisabled = "inbox_director_disabled"
	InboxReasonNoWatch          = "inbox_no_watch"
	InboxReasonManualNotify     = "inbox_manual_notify"
	inboxReasonNotRecorded      = "inbox_not_recorded"
)

// Osascript results recorded in a notified event besides "failed: <err>".
const (
	inboxNotifyOK      = "ok"
	inboxNotifySkipped = "skipped"
)

// inboxDesktopNotifyTimeout bounds one desktop notification call
// (notifyBounded): the inbox's, and that of a watch escalation the inbox
// could not take (escalateWithoutInbox, watch.go).
const inboxDesktopNotifyTimeout = 10 * time.Second

// Titles of the inbox's desktop notifications on the osascript path, named
// after the command whose process sends them, or for an ALERT item after
// the watch that raised it (plan 2026-10-23-watch-escalation-inbox, AC6):
// inboxNotifyTitleWatch for a deadline send and for every send of an ALERT
// item, inboxNotifyTitleEscalate for the other sends.
const (
	inboxNotifyTitleEscalate = "ralph org escalate"
	inboxNotifyTitleWatch    = "ralph org watch"
)

// defaultInboxPollInterval is how often WaitInbox reads the inbox again
// when Org.InboxPollInterval is unset.
const defaultInboxPollInterval = time.Second

// inboxWaitTimeoutPhrase is the part of WaitInbox's timeout error that the
// /director skill's inbox section matches in stderr to tell the timeout from
// a failure; it reruns the wait only on a match.
// TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase
// (director_skill_sync_test.go) fails when the skill and this text differ.
const inboxWaitTimeoutPhrase = "no open inbox item arrived within"

// inboxNotifyInvalid stands for an org_id or a TYPE in the desktop
// notification that no inbox writer would have written (an org_id that is
// not an identifier, a TYPE not in inboxItemTypes), which only a
// hand-edited inbox.jsonl line can hold: the notification text is built
// into an AppleScript string, so such a value is not passed on.
const inboxNotifyInvalid = "<invalid>"

// EscalationsPathIn returns the escalations.jsonl path within an
// already-resolved org state directory, mirroring InboxPathIn.
func EscalationsPathIn(stateDir string) string {
	return filepath.Join(stateDir, EscalationsRelName)
}

// InboxCommand returns the `ralph org inbox <verb> <id>` command that a
// banner or an error tells the reader to run, with --state-dir stateDir
// appended when stateDir is set. A caller sets stateDir only when the
// ledger was named with --state-dir (the rule of the CLI's
// orgReadCommandHint): without it the printed command reads the default
// ledger, which may be another ledger holding an item with the same ID. A
// ledger resolved by default resolves the same way when the command is run
// from the same place, so it is not repeated.
func InboxCommand(verb, id, stateDir string) string {
	command := "ralph org inbox " + verb + " " + id
	if stateDir != "" {
		command += " --state-dir " + shellQuote(stateDir)
	}
	return command
}

// EscalateParams describes one `ralph org escalate` call.
type EscalateParams struct {
	OrgID string
	// Text is the typed protocol message: TYPE QUESTION, BLOCKED, or
	// RESULT, a body of at most protocol.DefaultMaxBodyChars.
	Text string
	// Banner is where the one-line human-path banner is written (the CLI
	// passes stderr); nil means os.Stderr.
	Banner io.Writer
}

// EscalateResult is Escalate's return value. Err is set on every refusal
// and failure; ID and Recorded stay set when the item was recorded but its
// notification was not completed, so the caller can name the item to send
// again with `ralph org inbox notify <id>`.
type EscalateResult struct {
	// ID is the new item's ID (e<N>), set whenever Recorded.
	ID string
	// Type is the message's TYPE, set whenever Recorded.
	Type string
	// Recorded reports that the escalated event is in the inbox.
	Recorded bool
	// Notified reports that the human path was completed: the
	// escalations.jsonl line and the notified event were both written. The
	// desktop notification's own result does not count; it is recorded in
	// the notified event. It is false when Deferred.
	Notified bool
	// Deferred reports that the item was recorded only, with delivery
	// InboxDeliveryDeadline, and left to the director and to the ack
	// deadline of `ralph org watch --all`: no escalations.jsonl line, no
	// notified event, and no desktop notification were written or sent.
	// Err is nil then. It is for a caller that needs to tell this case from
	// a recorded item sent at once (Notified); the CLI does not: it prints
	// the same stdout line for both, and the banner says which. Tests read
	// it.
	Deferred bool
	Err      error
}

// Escalate records p.Text as a new inbox item. A refused message (an org_id
// that is not an identifier, a message protocol.Validate rejects, a TYPE
// other than QUESTION, BLOCKED, or RESULT) writes nothing, notifies no one,
// and calls no EscalateWatch. Before it records the item, Escalate decides
// how the item reaches a person (escalateDelivery):
//
//   - Config.Director.Enabled is false: the item is also sent to the human
//     path at once, with InboxReasonDirectorDisabled. EscalateWatch is not
//     called.
//   - Enabled, and EscalateWatch is nil or reports a watch that does not
//     accept work (WatchAcceptsWork: not running, stale, stopping, or a
//     --once run): no one would check the ack deadline, so the item is sent
//     at once, with InboxReasonNoWatch. A stale watch is not waited for; the
//     human gets the item at once.
//   - Enabled, and the watch accepts work: the item is only recorded, with
//     delivery InboxDeliveryDeadline, and one banner line on p.Banner says
//     that it was left to the director and that `ralph org watch --all`
//     sends it to the human after the ack deadline. The result is Deferred.
//
// Whether a director is registered does not change these cases (FR-9 of
// the spec). When the inbox cannot record the item, Escalate still writes
// the banner (marked not recorded) and sends the desktop notification, so
// the item reaches the human even from a broken ledger, then returns an
// error. When the item is recorded but a send at once is not completed, the
// error names the ID and `ralph org inbox notify <id>`, not another
// escalate, which would record a second item.
func (o *Org) Escalate(p EscalateParams) EscalateResult {
	m, err := validateEscalation(p.OrgID, p.Text)
	if err != nil {
		return EscalateResult{Err: err}
	}
	if o.Inbox == nil {
		return EscalateResult{Err: errors.New("org: escalate: no inbox store is configured")}
	}
	banner := bannerOrStderr(p.Banner)
	delivery, reason := o.escalateDelivery()
	id, err := o.Inbox.Escalate(InboxEscalation{OrgID: p.OrgID, Type: m.Type, TaskID: m.TaskID, Body: p.Text, Delivery: delivery})
	if err != nil {
		o.alertUnrecordedEscalation(p.OrgID, m.Type, err, banner)
		return EscalateResult{Err: fmt.Errorf("org: escalate: the item could not be recorded in the inbox: %w", err)}
	}
	res := EscalateResult{ID: id, Type: m.Type, Recorded: true}
	if delivery == InboxDeliveryDeadline {
		_, _ = fmt.Fprintln(banner, o.inboxDeferredBanner(p.OrgID, id, m.Type))
		res.Deferred = true
		return res
	}
	if err := o.sendInboxItemToHuman(p.OrgID, id, m.Type, "", reason, banner); err != nil {
		res.Err = fmt.Errorf("org: escalate: the item is recorded as %s, but the notification was not completed (%w): run `%s` to send it again (not escalate, which records another item)", id, err, InboxCommand("notify", id, o.InboxHintStateDir))
		return res
	}
	res.Notified = true
	return res
}

// EscalateNotRecorded is Escalate for a caller that could not set up the
// org runtime, such as `ralph org escalate` when ralph.toml does not load or
// when the ledger cannot be resolved (cause). It checks p the same way, so a
// refused message still writes nothing and notifies no one. Otherwise it
// tells the human that the item was not recorded, the way Escalate does when
// the inbox cannot record it (the banner, the desktop notification, and a
// best-effort escalations.jsonl line), and returns an error. It uses only
// Inbox (for the path in the banner), EscalationsPath, and DesktopNotify,
// and Inbox and EscalationsPath may be unset: an Org with neither (the
// CLI's Org for a ledger it cannot resolve) writes nothing and names only
// cause in the banner (alertUnrecordedEscalation).
func (o *Org) EscalateNotRecorded(p EscalateParams, cause error) EscalateResult {
	m, err := validateEscalation(p.OrgID, p.Text)
	if err != nil {
		return EscalateResult{Err: err}
	}
	o.alertUnrecordedEscalation(p.OrgID, m.Type, cause, bannerOrStderr(p.Banner))
	return EscalateResult{Err: fmt.Errorf("org: escalate: the item could not be recorded: %w", cause)}
}

// NotifyInboxItem sends inbox item id to the human path now and records
// another notified event, with InboxReasonManualNotify (`ralph org inbox
// notify`): for an item whose escalate could not complete its
// notification, and for one escalate left to the director and the ack
// deadline (delivery InboxDeliveryDeadline) that should reach the human
// without waiting. The commands the banner and the error name repeat
// Org.InboxHintStateDir, as Escalate's do. It returns ErrInboxUnknownID
// (wrapped) for an unknown ID and refuses a resolved item, which has
// nothing left to notify.
func (o *Org) NotifyInboxItem(id string, banner io.Writer) error {
	if o.Inbox == nil {
		return errors.New("org: inbox notify: no inbox store is configured")
	}
	inbox, err := o.Inbox.Read()
	if err != nil {
		return fmt.Errorf("org: inbox notify %s: %w", id, err)
	}
	i := slices.IndexFunc(inbox.Items, func(item InboxItem) bool { return item.ID == id })
	if i < 0 {
		return fmt.Errorf("org: inbox notify %s: %w", id, ErrInboxUnknownID)
	}
	item := inbox.Items[i]
	if item.State == InboxStateResolved {
		return fmt.Errorf("org: inbox notify %s: %w, so nothing is left to notify", id, ErrInboxResolved)
	}
	if err := o.sendInboxItemToHuman(item.OrgID, id, item.Type, item.AlertID, InboxReasonManualNotify, bannerOrStderr(banner)); err != nil {
		return fmt.Errorf("org: inbox notify %s: the notification was not completed (%w): run `%s` again", id, err, InboxCommand("notify", id, o.InboxHintStateDir))
	}
	return nil
}

// WaitInbox returns the open inbox items (`ralph org wait --inbox`), in ID
// order. When none is open it reads the inbox again every
// Org.InboxPollInterval (defaultInboxPollInterval when unset) until one is,
// or until timeout passes (timeout <= 0 waits without a bound), which is an
// error. Acked items do not count: an ack records that the item was read,
// and an acked item that stays unresolved is left to a resolve deadline
// (stage 6 of the spec's rollout), not to a waiting director. A failed read
// returns at once.
func (o *Org) WaitInbox(timeout time.Duration) ([]InboxItem, error) {
	if o.Inbox == nil {
		return nil, errors.New("org: wait --inbox: no inbox store is configured")
	}
	interval := o.InboxPollInterval
	if interval <= 0 {
		interval = defaultInboxPollInterval
	}
	var deadline <-chan time.Time
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		deadline = timer.C
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	// expired is set when the deadline fires; the loop then reads once more,
	// so an item that arrived just before the deadline is still returned.
	expired := false
	for {
		open, err := o.openInboxItems()
		if err != nil {
			return nil, fmt.Errorf("org: wait --inbox: %w", err)
		}
		if len(open) > 0 {
			return open, nil
		}
		if expired {
			return nil, fmt.Errorf("org: wait --inbox: %s %d ms", inboxWaitTimeoutPhrase, timeout.Milliseconds())
		}
		select {
		case <-deadline:
			expired = true
		case <-ticker.C:
		}
	}
}

// openInboxItems is one read of the inbox for a wait: the open items in ID
// order (nil when none), or the read's error as Inbox.Read returned it. It
// is the read of every round of WaitInbox and of the director's wait
// (WaitInboxForDirector, director_wait.go), so both count the same items as
// open; each wraps the error in its own prefix.
func (o *Org) openInboxItems() ([]InboxItem, error) {
	inbox, err := o.Inbox.Read()
	if err != nil {
		return nil, err
	}
	var open []InboxItem
	for _, item := range inbox.Items {
		if item.State == InboxStateOpen {
			open = append(open, item)
		}
	}
	return open, nil
}

// escalateDelivery decides how Escalate delivers a new item (the cases in
// Escalate's doc) with inboxDelivery: InboxDeliveryDeadline with no reason
// when Config.Director.Enabled is set and EscalateWatch reports a watch that
// accepts work, and InboxDeliveryNow with the reason of the send at once
// otherwise. EscalateWatch is called only when Enabled is set, and a nil
// EscalateWatch counts as no watch.
func (o *Org) escalateDelivery() (delivery, reason string) {
	var acceptsWork func() bool
	if o.EscalateWatch != nil {
		acceptsWork = func() bool { return WatchAcceptsWork(o.EscalateWatch()) }
	}
	return inboxDelivery(o.Config.Director.Enabled, acceptsWork)
}

// inboxDelivery decides how a new inbox item reaches a person, the same way
// for a leader's escalate (escalateDelivery) and for an escalation of the
// cross-org watch (plan 2026-10-23-watch-escalation-inbox, D3). enabled is
// Config.Director.Enabled, and acceptsWork reports whether a cross-org watch
// that will check the ack deadline accepts work:
//
//   - enabled is false: InboxDeliveryNow with InboxReasonDirectorDisabled.
//     acceptsWork is not called.
//   - acceptsWork is nil or returns false: no one would check the ack
//     deadline, so InboxDeliveryNow with InboxReasonNoWatch.
//   - otherwise: InboxDeliveryDeadline with no reason; the item is left to
//     the director and to the ack deadline.
func inboxDelivery(enabled bool, acceptsWork func() bool) (delivery, reason string) {
	if !enabled {
		return InboxDeliveryNow, InboxReasonDirectorDisabled
	}
	if acceptsWork == nil || !acceptsWork() {
		return InboxDeliveryNow, InboxReasonNoWatch
	}
	return InboxDeliveryDeadline, ""
}

// inboxDeferredBanner is the banner line (without its newline) of an item
// Escalate left to the director and the ack deadline. orgID and typ have
// passed validateEscalation; they are escaped like the other banners'. The
// `inbox show` command repeats Org.InboxHintStateDir. The banner names the
// ack deadline by its key, not its minutes: the watch reads ack_minutes from
// the ralph.toml it runs with (the main worktree's), and the escalating
// process's Config, for a feature org's leader its worktree's ralph.toml,
// can hold another value (self-review L3 of org-director-attach).
func (o *Org) inboxDeferredBanner(orgID, id, typ string) string {
	return fmt.Sprintf("ORG ESCALATION RECORDED: org=%s item=%s type=%s -- left in %s for the director; "+
		"if it is still open after the ack deadline of ralph org watch --all ([org.director].ack_minutes), the watch sends it to the human; read it with: %s",
		PrintableInboxText(orgID, false), id, PrintableInboxText(typ, false), o.Inbox.Path(),
		InboxCommand("show", id, o.InboxHintStateDir))
}

// validateEscalation checks orgID and text before Escalate writes anything
// and returns the parsed message.
func validateEscalation(orgID, text string) (protocol.Message, error) {
	if err := ValidateIdentifier("org_id", orgID); err != nil {
		return protocol.Message{}, err
	}
	m, err := protocol.Parse(text)
	if err == nil {
		err = protocol.Validate(m, protocol.DefaultMaxBodyChars)
	}
	if err != nil {
		return protocol.Message{}, fmt.Errorf("org: escalate: message rejected by protocol validation: %w", err)
	}
	if !slices.Contains(escalateTypes, m.Type) {
		return protocol.Message{}, fmt.Errorf("org: escalate: TYPE %s cannot be escalated: use one of %s", m.Type, strings.Join(escalateTypes, ", "))
	}
	return m, nil
}

// sendInboxItemToHuman is the human path shared by Escalate,
// NotifyInboxItem, and SweepInboxDeadlines (deadline.go), and by the
// cross-org watch's ALERT items (plan 2026-10-23-watch-escalation-inbox):
// the escalations.jsonl line, the banner, the desktop notification (org,
// ID, and TYPE only, never the message body), and then the notified event
// with the notification's result. reason goes on the escalations.jsonl line
// and the notified event, and picks the banner and the notification text
// (inboxHumanTextFor). The banner and the notification are best-effort.
// When the escalations.jsonl line cannot be written, the notified event is
// not written either, so the item stays unnotified in the inbox and `ralph
// org inbox notify` can complete it. The item's state never changes.
//
// alertID is the item's alert_id (InboxItem.AlertID). When it is "" (a
// leader's item) the escalations.jsonl line names the leader as its
// subject, as it always has. When it is set (an ALERT item of the watch),
// the line carries alert_id and, as its subject, the seat the watch's ALERT
// was about (alertIDSeat; no subject when alertID does not have the shape
// sendAlert writes), so the line reads like the watch's own escalation
// lines.
//
// orgID, typ, and alertID come from inbox.jsonl when NotifyInboxItem or
// SweepInboxDeadlines calls, so a hand-edited line can hold anything there:
// the banner escapes them with PrintableInboxText, and the desktop
// notification gets orgID and typ only when they are values an inbox
// writer writes (inboxNotifyInvalid otherwise) and never gets alertID. The
// ID is one the inbox fold accepted (e<N>). The banner's `inbox show`
// command repeats Org.InboxHintStateDir.
func (o *Org) sendInboxItemToHuman(orgID, id, typ, alertID, reason string, banner io.Writer) error {
	recordErr := errors.New("no escalations.jsonl path is configured")
	if o.EscalationsPath != "" {
		rec := escalationRecord{TS: o.now(), OrgID: orgID, InboxID: id, Subject: LeaderIdentity, Reason: reason}
		if alertID != "" {
			rec.AlertID, rec.Subject = alertID, alertIDSeat(alertID)
		}
		recordErr = appendJSONLine(o.EscalationsPath, rec)
	}
	text := o.inboxHumanTextFor(orgID, id, typ, alertID, reason)
	_, _ = fmt.Fprintln(banner, text.banner)
	osascript := o.inboxDesktopNotify(text.title, text.desktop)
	if recordErr != nil {
		return recordErr
	}
	return o.Inbox.AppendNotified(id, reason, osascript)
}

// inboxHumanText is what one send of an inbox item to the human path
// shows: the banner line (without its newline), and the title and the text
// of the desktop notification.
type inboxHumanText struct {
	banner  string
	title   string
	desktop string
}

// inboxHumanTextFor returns the text of a send for reason: the deadline
// text (inboxDeadlineHumanText) for the two deadline reasons, and the text
// of an escalated item otherwise. For a TYPE ALERT item, the cross-org
// watch's (plan 2026-10-23-watch-escalation-inbox, AC6), the escalated
// item's text is the watch's: a banner that starts with WATCHDOG
// ESCALATION, as the watch's own escalation banner does, and names the
// alert_id when there is one, under the title inboxNotifyTitleWatch; the
// desktop text has the same shape as a leader's item. It takes its strings
// in sendInboxItemToHuman's order (orgID, id, typ, alertID, reason), and
// inboxDeadlineHumanText the same without alertID.
func (o *Org) inboxHumanTextFor(orgID, id, typ, alertID, reason string) inboxHumanText {
	if reason == InboxReasonAckDeadline || reason == InboxReasonResolveDeadline {
		return o.inboxDeadlineHumanText(orgID, id, typ, reason)
	}
	if typ == protocol.TypeAlert {
		alert := ""
		if alertID != "" {
			alert = " alert=" + PrintableInboxText(alertID, false)
		}
		return inboxHumanText{
			banner: fmt.Sprintf("WATCHDOG ESCALATION: org=%s item=%s type=%s%s -- recorded in %s; read it with: %s",
				PrintableInboxText(orgID, false), id, typ, alert, o.Inbox.Path(), InboxCommand("show", id, o.InboxHintStateDir)),
			title:   inboxNotifyTitleWatch,
			desktop: fmt.Sprintf("org %s raised %s (%s)", notifiableOrgID(orgID), id, notifiableType(typ)),
		}
	}
	return inboxHumanText{
		banner: fmt.Sprintf("ORG ESCALATION: org=%s item=%s type=%s -- recorded in %s; read it with: %s",
			PrintableInboxText(orgID, false), id, PrintableInboxText(typ, false), o.Inbox.Path(), InboxCommand("show", id, o.InboxHintStateDir)),
		title:   inboxNotifyTitleEscalate,
		desktop: fmt.Sprintf("org %s raised %s (%s)", notifiableOrgID(orgID), id, notifiableType(typ)),
	}
}

// notifiableOrgID returns orgID when it is an identifier
// (ValidateIdentifier), and inboxNotifyInvalid otherwise.
func notifiableOrgID(orgID string) string {
	if ValidateIdentifier("org_id", orgID) != nil {
		return inboxNotifyInvalid
	}
	return orgID
}

// notifiableType returns typ when it is one of inboxItemTypes (ALERT
// included, plan 2026-10-23-watch-escalation-inbox, AC6), and
// inboxNotifyInvalid otherwise.
func notifiableType(typ string) string {
	if !slices.Contains(inboxItemTypes, typ) {
		return inboxNotifyInvalid
	}
	return typ
}

// alertIDSeat returns the seat part of a watch alert_id, which sendAlert
// (watch.go) writes as conditionKey(org, seat, condition) + "@" + the
// ALERT's time in Unix nanoseconds: "<org>/<seat>/<condition>@<n>". It
// returns "" when alertID does not have that shape (an org and a seat that
// are identifiers, a non-empty condition, and a decimal n), which only a
// hand-edited inbox.jsonl line can hold. The seat is the subject of the
// watch's ALERT, so it also tells why the watch escalated it
// (escalateAlert): LeaderIdentity means leader_is_anomaly_subject, and any
// other seat deadman_timeout (plan 2026-10-23-watch-escalation-inbox, AC3).
// alertIDConditionType (watch.go) reads the condition of the same shape and
// checks only its separators; a change to the shape changes both.
func alertIDSeat(alertID string) string {
	at := strings.LastIndex(alertID, "@")
	if at < 0 {
		return ""
	}
	nano := alertID[at+1:]
	parts := strings.Split(alertID[:at], "/")
	if nano == "" || strings.Trim(nano, "0123456789") != "" || len(parts) != 3 ||
		ValidateIdentifier("org_id", parts[0]) != nil || ValidateIdentifier("seat_id", parts[1]) != nil || parts[2] == "" {
		return ""
	}
	return parts[1]
}

// alertUnrecordedEscalation tells the human about an escalate that was not
// recorded: the banner (marked not recorded, with the cause), the desktop
// notification, and a best-effort escalations.jsonl line, which is in the
// same ledger and may fail for the same cause. With no Inbox
// (EscalateNotRecorded with no ledger) the banner says the item is not in
// any inbox instead of naming the inbox's path, and with no EscalationsPath
// no line is written. orgID and typ have passed validateEscalation.
func (o *Org) alertUnrecordedEscalation(orgID, typ string, cause error, banner io.Writer) {
	where := "any inbox"
	if o.Inbox != nil {
		where = o.Inbox.Path()
	}
	_, _ = fmt.Fprintf(banner, "ORG ESCALATION (NOT RECORDED): org=%s type=%s -- the item is not in %s: %v\n",
		orgID, typ, where, cause)
	_ = o.inboxDesktopNotify(inboxNotifyTitleEscalate, fmt.Sprintf("org %s could not record a %s escalation; see the leader's pane", orgID, typ))
	if o.EscalationsPath != "" {
		_ = appendJSONLine(o.EscalationsPath, escalationRecord{
			TS: o.now(), OrgID: orgID, Subject: LeaderIdentity, Reason: inboxReasonNotRecorded,
		})
	}
}

// inboxDesktopNotify sends the desktop notification with DesktopNotify
// (notifyBounded) and returns the result a notified event records: "ok",
// "failed: <err>", or "skipped" when DesktopNotify is nil off darwin. title
// is used on the osascript path only; DesktopNotify gets the message alone.
func (o *Org) inboxDesktopNotify(title, message string) string {
	return notifyBounded(o.DesktopNotify, func(ctx context.Context, message string) error {
		return osascriptNotify(ctx, title, message)
	}, message)
}

// desktopNotifyGOOS is the GOOS notifyBounded reads when it has no notify
// function: on darwin it notifies with osascript, and elsewhere it sends
// nothing and returns skipped. It is runtime.GOOS; a test sets another value
// so that a nil Org.DesktopNotify or WatchAllParams.Escalate never reaches
// osascript.
var desktopNotifyGOOS = runtime.GOOS

// notifyBounded sends one desktop notification of message and returns its
// result with the values of the inbox's notified event: ok, failed: <err>,
// or skipped. It is the notification of the inbox's human path
// (inboxDesktopNotify, with Org.DesktopNotify) and of a watch escalation the
// inbox could not take (notifyWithoutInbox, with WatchAllParams.Escalate,
// watch.go). It calls notify, or when notify is nil, onDarwin (osascript) on
// darwin and nothing elsewhere (skipped), by desktopNotifyGOOS. The call gets
// its own bound, inboxDesktopNotifyTimeout from context.Background(): an
// osascript that does not return cannot hold the caller, and a SIGINT or
// SIGTERM that cancels the caller's context does not cut off a notification
// that is never sent again (self-review L2).
func notifyBounded(notify, onDarwin EscalateFunc, message string) string {
	if notify == nil {
		if desktopNotifyGOOS != "darwin" {
			return inboxNotifySkipped
		}
		notify = onDarwin
	}
	ctx, cancel := context.WithTimeout(context.Background(), inboxDesktopNotifyTimeout)
	defer cancel()
	if err := notify(ctx, message); err != nil {
		return "failed: " + err.Error()
	}
	return inboxNotifyOK
}

func bannerOrStderr(w io.Writer) io.Writer {
	if w == nil {
		return os.Stderr
	}
	return w
}
