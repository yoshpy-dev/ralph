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
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// Escalate, NotifyInboxItem, and WaitInbox are the org layer of `ralph org
// escalate`, `ralph org inbox notify`, and `ralph org wait --inbox` (FR-5 of
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

// defaultInboxPollInterval is how often WaitInbox reads the inbox again
// when Org.InboxPollInterval is unset.
const defaultInboxPollInterval = time.Second

// inboxNotifyInvalid stands for an org_id or a TYPE in the desktop
// notification that Escalate would have refused, which only a hand-edited
// inbox.jsonl line can hold: the notification text is built into an
// AppleScript string, so such a value is not passed on.
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
	// HintStateDir is the stateDir of the InboxCommand commands that the
	// banner and the error name: the CLI passes the resolved state dir
	// when --state-dir was given, and "" otherwise.
	HintStateDir string
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
	res := EscalateResult{ID: id, Type: m.Type, Recorded: true}
	if err := o.sendInboxItemToHuman(p.OrgID, id, m.Type, p.HintStateDir, banner); err != nil {
		res.Err = fmt.Errorf("org: escalate: the item is recorded as %s, but the notification was not completed (%w): run `%s` to send it again (not escalate, which records another item)", id, err, InboxCommand("notify", id, p.HintStateDir))
		return res
	}
	res.Notified = true
	return res
}

// EscalateNotRecorded is Escalate for a caller that could not set up the
// org runtime, such as `ralph org escalate` when ralph.toml does not load
// (cause). It checks p the same way, so a refused message still writes
// nothing and notifies no one. Otherwise it tells the human that the item
// was not recorded, the way Escalate does when the inbox cannot record it
// (the banner, the desktop notification, and a best-effort
// escalations.jsonl line), and returns an error. It needs only Inbox (for
// the path in the banner), EscalationsPath, and DesktopNotify.
func (o *Org) EscalateNotRecorded(p EscalateParams, cause error) EscalateResult {
	m, err := validateEscalation(p.OrgID, p.Text)
	if err != nil {
		return EscalateResult{Err: err}
	}
	if o.Inbox == nil {
		return EscalateResult{Err: errors.New("org: escalate: no inbox store is configured")}
	}
	o.alertUnrecordedEscalation(p.OrgID, m.Type, cause, bannerOrStderr(p.Banner))
	return EscalateResult{Err: fmt.Errorf("org: escalate: the item could not be recorded: %w", cause)}
}

// NotifyInboxItem sends inbox item id to the human path again and records
// another notified event (`ralph org inbox notify`), for an item whose
// escalate could not complete its notification. hintStateDir is
// EscalateParams.HintStateDir for the commands the banner and the error
// name. It returns ErrInboxUnknownID (wrapped) for an unknown ID and
// refuses a resolved item, which has nothing left to notify.
func (o *Org) NotifyInboxItem(id, hintStateDir string, banner io.Writer) error {
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
	if err := o.sendInboxItemToHuman(item.OrgID, id, item.Type, hintStateDir, bannerOrStderr(banner)); err != nil {
		return fmt.Errorf("org: inbox notify %s: the notification was not completed (%w): run `%s` again", id, err, InboxCommand("notify", id, hintStateDir))
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
		inbox, err := o.Inbox.Read()
		if err != nil {
			return nil, fmt.Errorf("org: wait --inbox: %w", err)
		}
		var open []InboxItem
		for _, item := range inbox.Items {
			if item.State == InboxStateOpen {
				open = append(open, item)
			}
		}
		if len(open) > 0 {
			return open, nil
		}
		if expired {
			return nil, fmt.Errorf("org: wait --inbox: no open inbox item arrived within %d ms", timeout.Milliseconds())
		}
		select {
		case <-deadline:
			expired = true
		case <-ticker.C:
		}
	}
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
//
// orgID and typ come from inbox.jsonl when NotifyInboxItem calls, so a
// hand-edited line can hold anything there: the banner escapes them with
// PrintableInboxText, and the desktop notification gets them only when
// they are values Escalate accepts (inboxNotifyInvalid otherwise). The ID
// is one the inbox fold accepted (e<N>).
func (o *Org) sendInboxItemToHuman(orgID, id, typ, hintStateDir string, banner io.Writer) error {
	recordErr := errors.New("no escalations.jsonl path is configured")
	if o.EscalationsPath != "" {
		recordErr = appendJSONLine(o.EscalationsPath, escalationRecord{
			TS: o.now(), OrgID: orgID, InboxID: id, Subject: LeaderIdentity, Reason: inboxReasonNoDirector,
		})
	}
	_, _ = fmt.Fprintf(banner, "ORG ESCALATION: org=%s item=%s type=%s -- recorded in %s; read it with: %s\n",
		PrintableInboxText(orgID, false), id, PrintableInboxText(typ, false), o.Inbox.Path(), InboxCommand("show", id, hintStateDir))
	osascript := o.inboxDesktopNotify(fmt.Sprintf("org %s raised %s (%s)", notifiableOrgID(orgID), id, notifiableType(typ)))
	if recordErr != nil {
		return recordErr
	}
	return o.Inbox.AppendNotified(id, inboxReasonNoDirector, osascript)
}

// notifiableOrgID returns orgID when it is an identifier
// (ValidateIdentifier), and inboxNotifyInvalid otherwise.
func notifiableOrgID(orgID string) string {
	if ValidateIdentifier("org_id", orgID) != nil {
		return inboxNotifyInvalid
	}
	return orgID
}

// notifiableType returns typ when it is one of escalateTypes, and
// inboxNotifyInvalid otherwise.
func notifiableType(typ string) string {
	if !slices.Contains(escalateTypes, typ) {
		return inboxNotifyInvalid
	}
	return typ
}

// PrintableInboxText returns s with each control character written as its
// Go escape (\x1b, \r, \u0085), keeping newlines and tabs when
// keepLineBreaks is set. Inbox text is written by a leader (or by hand in
// inbox.jsonl) and printed to a terminal, so an escape sequence in it must
// not reach the terminal as one. The human-path banner and every `ralph
// org inbox` text output use it.
func PrintableInboxText(s string, keepLineBreaks bool) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case keepLineBreaks && (r == '\n' || r == '\t'):
			b.WriteRune(r)
		case unicode.IsControl(r):
			quoted := strconv.QuoteRune(r)
			b.WriteString(quoted[1 : len(quoted)-1])
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// alertUnrecordedEscalation tells the human about an escalate that was not
// recorded: the banner (marked not recorded, with the cause), the desktop
// notification, and a best-effort escalations.jsonl line, which is in the
// same ledger and may fail for the same cause. orgID and typ have passed
// validateEscalation.
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
