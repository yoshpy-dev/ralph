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

// Escalate and NotifyInboxItem are the org layer of `ralph org escalate`
// and `ralph org inbox notify` (FR-5 of
// docs/specs/2026-10-07-org-multi-org-director.md). There is no way to
// register a director yet, so every escalated item is also sent at once to
// the human path, the way the spec treats an org whose director is
// disabled: one line in escalations.jsonl, a banner on the caller's stderr,
// and a best-effort desktop notification. The notified event in the inbox
// records that the item was sent, so a later inbox watcher does not send it
// again.

// escalateTypes are the TYPEs Escalate accepts: a leader raises an item
// when it needs a decision (QUESTION), cannot proceed (BLOCKED), or has
// finished, for example opened a PR (RESULT). BLOCKED and RESULT need a
// TASK_ID; protocol.Validate enforces that.
var escalateTypes = []string{protocol.TypeQuestion, protocol.TypeBlocked, protocol.TypeResult}

// Reasons of the human-path records. inboxReasonNoDirector is written to
// the escalations.jsonl line and the notified event of an item sent to the
// human path; inboxReasonNotRecorded to the best-effort escalations.jsonl
// line of an escalate the inbox could not record (it has no inbox_id).
const (
	inboxReasonNoDirector  = "inbox_no_director"
	inboxReasonNotRecorded = "inbox_not_recorded"
)

// Osascript results recorded in a notified event besides "failed: <err>".
const (
	inboxNotifyOK      = "ok"
	inboxNotifySkipped = "skipped"
)

// inboxDesktopNotifyTimeout bounds one desktop notification call.
const inboxDesktopNotifyTimeout = 10 * time.Second

// EscalationsPathIn returns the escalations.jsonl path within an
// already-resolved org state directory, mirroring InboxPathIn.
func EscalationsPathIn(stateDir string) string {
	return filepath.Join(stateDir, EscalationsRelName)
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
	// Recorded reports that the escalated event is in the inbox.
	Recorded bool
	// Notified reports that the human path was completed: the
	// escalations.jsonl line and the notified event were both written. The
	// desktop notification's own result does not count; it is recorded in
	// the notified event.
	Notified bool
	Err      error
}

// Escalate records p.Text as a new inbox item and sends it to the human
// path. A refused message (an org_id that is not an identifier, a message
// protocol.Validate rejects, a TYPE other than QUESTION, BLOCKED, or
// RESULT) writes nothing and notifies no one. When the inbox cannot record
// the item, Escalate still writes the banner (marked not recorded) and
// sends the desktop notification, so the item reaches the human even from
// a broken ledger, then returns an error. When the item is recorded but the
// human path is not completed, the error names the ID and `ralph org inbox
// notify <id>`, not another escalate, which would record a second item.
func (o *Org) Escalate(p EscalateParams) EscalateResult {
	m, err := validateEscalation(p.OrgID, p.Text)
	if err != nil {
		return EscalateResult{Err: err}
	}
	if o.Inbox == nil {
		return EscalateResult{Err: errors.New("org: escalate: no inbox store is configured")}
	}
	banner := bannerOrStderr(p.Banner)
	id, err := o.Inbox.Escalate(InboxEscalation{OrgID: p.OrgID, Type: m.Type, TaskID: m.TaskID, Body: p.Text})
	if err != nil {
		o.alertUnrecordedEscalation(p.OrgID, m.Type, err, banner)
		return EscalateResult{Err: fmt.Errorf("org: escalate: the item could not be recorded in the inbox: %w", err)}
	}
	res := EscalateResult{ID: id, Recorded: true}
	if err := o.sendInboxItemToHuman(p.OrgID, id, m.Type, banner); err != nil {
		res.Err = fmt.Errorf("org: escalate: the item is recorded as %s, but the notification was not completed (%w): run `ralph org inbox notify %s` to send it again (not escalate, which records another item)", id, err, id)
		return res
	}
	res.Notified = true
	return res
}

// NotifyInboxItem sends inbox item id to the human path again and records
// another notified event (`ralph org inbox notify`), for an item whose
// escalate could not complete its notification. It returns
// ErrInboxUnknownID (wrapped) for an unknown ID and refuses a resolved
// item, which has nothing left to notify.
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
	if err := o.sendInboxItemToHuman(item.OrgID, id, item.Type, bannerOrStderr(banner)); err != nil {
		return fmt.Errorf("org: inbox notify %s: the notification was not completed (%w): run `ralph org inbox notify %s` again", id, err, id)
	}
	return nil
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

// sendInboxItemToHuman is the human path shared by Escalate and
// NotifyInboxItem: the escalations.jsonl line, the banner, the desktop
// notification (org, ID, and TYPE only, never the message body), and then
// the notified event with the notification's result. The banner and the
// notification are best-effort. When the escalations.jsonl line cannot be
// written, the notified event is not written either, so the item stays
// unnotified in the inbox and `ralph org inbox notify` can complete it.
func (o *Org) sendInboxItemToHuman(orgID, id, typ string, banner io.Writer) error {
	recordErr := errors.New("no escalations.jsonl path is configured")
	if o.EscalationsPath != "" {
		recordErr = appendJSONLine(o.EscalationsPath, escalationRecord{
			TS: o.now(), OrgID: orgID, InboxID: id, Subject: LeaderIdentity, Reason: inboxReasonNoDirector,
		})
	}
	_, _ = fmt.Fprintf(banner, "ORG ESCALATION: org=%s item=%s type=%s -- recorded in %s; read it with: ralph org inbox show %s\n",
		orgID, id, typ, o.Inbox.Path(), id)
	osascript := o.inboxDesktopNotify(fmt.Sprintf("org %s raised %s (%s)", orgID, id, typ))
	if recordErr != nil {
		return recordErr
	}
	return o.Inbox.AppendNotified(id, inboxReasonNoDirector, osascript)
}

// alertUnrecordedEscalation tells the human about an escalate the inbox
// could not record: the banner (marked not recorded, with the cause), the
// desktop notification, and a best-effort escalations.jsonl line, which is
// in the same ledger and may fail for the same cause.
func (o *Org) alertUnrecordedEscalation(orgID, typ string, cause error, banner io.Writer) {
	_, _ = fmt.Fprintf(banner, "ORG ESCALATION (NOT RECORDED): org=%s type=%s -- the item is not in %s: %v\n",
		orgID, typ, o.Inbox.Path(), cause)
	_ = o.inboxDesktopNotify(fmt.Sprintf("org %s could not record a %s escalation; see the leader's pane", orgID, typ))
	if o.EscalationsPath != "" {
		_ = appendJSONLine(o.EscalationsPath, escalationRecord{
			TS: o.now(), OrgID: orgID, Subject: LeaderIdentity, Reason: inboxReasonNotRecorded,
		})
	}
}

// inboxDesktopNotify sends the desktop notification and returns the result
// a notified event records: "ok", "failed: <err>", or "skipped" when
// DesktopNotify is nil off darwin.
func (o *Org) inboxDesktopNotify(message string) string {
	notify := o.DesktopNotify
	if notify == nil {
		if runtime.GOOS != "darwin" {
			return inboxNotifySkipped
		}
		notify = func(ctx context.Context, message string) error {
			return osascriptNotify(ctx, "ralph org escalate", message)
		}
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
