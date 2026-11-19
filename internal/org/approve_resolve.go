package org

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// The resolve of an inbox item, `ralph org inbox resolve` (ResolveInboxItem;
// plan docs/plans/active/2026-11-19-approve-resolve-decision.md, issue #345).
// `ralph org approve` records an approval and never tells the leader (D7 of
// plan 2026-10-14-org-director-attach): the approval DECISION is a separate
// `ralph org send` (ApprovalDecisionCommand), and while the item is open or
// acked, the ack and resolve deadlines of `ralph org watch --all` can still
// bring it to a human. A resolve of an approved item before that DECISION was
// sent ended those deadlines and left the leader waiting with no one told,
// so ResolveInboxItem refuses it until the DECISION is sent, unless it is
// told to resolve the item without it.

// ErrApprovalDecisionNotSent is returned (wrapped) by ResolveInboxItem for an
// inbox item whose plan approval is recorded in its org's current life while
// the manifest holds no counted approval DECISION (checkApprovalDecision)
// sent to the org's leader after that approval.
var ErrApprovalDecisionNotSent = errors.New("the approval of this inbox item's plan is recorded, " +
	"but the manifest holds no counted approval DECISION sent to the org's leader after it")

// InboxResolveResult is what ResolveInboxItem reports about an item it
// resolved. For an item whose plan approval is not recorded, every field is
// empty.
type InboxResolveResult struct {
	// OrgID and Plan are the item's org and the plan and digest its approval
	// recorded, set for an item whose approval is recorded.
	OrgID string
	Plan  PlanRef
	// SubmitUnconfirmed reports that the approval DECISION was sent, but
	// every send of it recorded submit_unconfirmed=true: Send could not
	// confirm the submit, so the CLI says to read the leader's pane with
	// `ralph org read`.
	SubmitUnconfirmed bool
	// NoDecisionFound reports that the item was resolved with
	// withoutDecision while the manifest holds no counted approval DECISION
	// (none this ralph recorded; one an older ralph sent has no TextSHA256),
	// so the CLI says to make sure the leader was told.
	NoDecisionFound bool
}

// resolveAfterCheckHook, when set (tests only), runs inside ResolveInboxItem's
// hold of the manifest lock, which it takes for a plan QUESTION and for an ID
// its first read of the inbox did not find, after the check decided to
// resolve the item and before the inbox is written.
var resolveAfterCheckHook func()

// ResolveInboxItem resolves inbox item id with note, as InboxStore.Resolve
// does, after one more check. The note is checked first (ValidateInboxNote),
// and with no inbox file the ID is unknown (ErrInboxUnknownID), as
// InboxStore.Resolve says, so neither writes or creates anything. Then it
// reads the inbox: an item found there that is not a plan QUESTION (a
// QUESTION whose header names a plan, isPlanQuestion) is resolved as before,
// without the manifest lock, since an item's type and body do not change
// after its escalate. For a plan QUESTION, and for an ID the read did not
// find, it takes the manifest lock, the lock Approve records under, and reads
// the inbox and the manifest again: when the item is an open or acked plan
// QUESTION and its org's current life holds the approval of this item, the
// item is resolved only when an approval DECISION was sent to the org's
// leader after that approval (checkApprovalDecision). Otherwise it is refused
// with ErrApprovalDecisionNotSent and nothing written, and the error gives
// the `ralph org send` of the DECISION (ApprovalDecisionCommand, with
// --state-dir InboxHintStateDir when it is set), the resolve to run once that
// send succeeded (ApprovalResolveCommand), and in a last sentence what
// --without-decision is for; with withoutDecision set, the item is resolved
// all the same and the result's NoDecisionFound says so. A plan QUESTION with
// no approval recorded, or only one from before the org's latest disband, is
// resolved as before, and an unknown or resolved ID gets InboxStore.Resolve's
// error.
//
// For a plan QUESTION the inbox is written (InboxStore.Resolve, which takes
// the inbox lock) while the manifest lock is still held, so an approve cannot
// record the approval between the check and the write: an approve that
// records first makes the resolve refuse, and an approve that waits for the
// lock finds the item resolved afterwards (checkResolvedMeanwhile). The locks
// are taken in the order manifest, then inbox; no code takes the manifest
// lock while it holds the inbox lock (withFileLock). An ID the first read did
// not find takes the lock too, so an item escalated and approved after that
// read is not resolved past the check.
func (o *Org) ResolveInboxItem(id, note string, withoutDecision bool) (InboxResolveResult, error) {
	if err := ValidateInboxNote(note); err != nil {
		return InboxResolveResult{}, err
	}
	if o.Inbox == nil {
		return InboxResolveResult{}, errors.New("org: inbox: resolve: no inbox store is configured")
	}
	if _, err := os.Stat(o.Inbox.Path()); errors.Is(err, fs.ErrNotExist) {
		return InboxResolveResult{}, fmt.Errorf("org: inbox: resolve %s: %w", id, ErrInboxUnknownID)
	}
	inbox, err := o.Inbox.Read()
	if err != nil {
		return InboxResolveResult{}, err
	}
	if item, found := findInboxItem(inbox.Items, id); found && !isPlanQuestion(item) {
		return InboxResolveResult{}, o.Inbox.Resolve(id, note)
	}
	var res InboxResolveResult
	err = withManifestLock(filepath.Dir(o.Manifest.Path()), func() error {
		c, err := o.readApprovalDecisionCheck(id)
		if err != nil {
			return err
		}
		// With no approval of the item in its org's current life, res stays
		// empty and the item is resolved as before.
		if c.approved {
			if !c.sent && !withoutDecision {
				return approvalDecisionNotSentError(id, c, o.InboxHintStateDir)
			}
			res = InboxResolveResult{OrgID: c.orgID, Plan: c.ref, SubmitUnconfirmed: c.submitUnconfirmed, NoDecisionFound: !c.sent}
		}
		if resolveAfterCheckHook != nil {
			resolveAfterCheckHook()
		}
		return o.Inbox.Resolve(id, note)
	})
	if err != nil {
		return InboxResolveResult{}, err
	}
	return res, nil
}

// findInboxItem returns inbox item id of items, and whether it is there.
func findInboxItem(items []InboxItem, id string) (InboxItem, bool) {
	i := slices.IndexFunc(items, func(item InboxItem) bool { return item.ID == id })
	if i < 0 {
		return InboxItem{}, false
	}
	return items[i], true
}

// isPlanQuestion reports whether item is a QUESTION whose header names a plan
// (planRefFromText), the item a leader raises for a plan's approval. It reads
// only the item's type and body, which do not change after its escalate.
func isPlanQuestion(item InboxItem) bool {
	if item.Type != protocol.TypeQuestion {
		return false
	}
	_, err := planRefFromText(item.Body)
	return err == nil
}

// approvalDecisionCheck is what checkApprovalDecision found for one inbox
// item.
type approvalDecisionCheck struct {
	// approved: the org's current life holds an approval of the item.
	approved bool
	orgID    string
	// ref is the plan and digest of the item's first approval there.
	ref PlanRef
	// sent: an approval DECISION was sent to the org's leader after one of
	// those approvals.
	sent bool
	// submitUnconfirmed: sent, and every such send recorded
	// submit_unconfirmed=true.
	submitUnconfirmed bool
}

// readApprovalDecisionCheck reads the inbox and, for an open or acked plan
// QUESTION item id (isPlanQuestion), the manifest, and returns
// checkApprovalDecision for it. An item that is unknown, resolved, or not a
// plan QUESTION gives a check with approved false, which leaves the item to
// InboxStore.Resolve. ResolveInboxItem calls it holding the manifest lock.
func (o *Org) readApprovalDecisionCheck(id string) (approvalDecisionCheck, error) {
	inbox, err := o.Inbox.Read()
	if err != nil {
		return approvalDecisionCheck{}, err
	}
	item, found := findInboxItem(inbox.Items, id)
	if !found || item.State == InboxStateResolved || !isPlanQuestion(item) {
		return approvalDecisionCheck{}, nil
	}
	rr, err := o.Manifest.Read()
	if err != nil {
		return approvalDecisionCheck{}, fmt.Errorf("org: inbox: resolve %s: read manifest: %w", id, err)
	}
	return checkApprovalDecision(rr.Events, item.OrgID, id), nil
}

// checkApprovalDecision reads events for the approvals of inbox item id in
// orgID's current life and the approval DECISIONs sent after them, and
// returns what it found. An approval is a real (non-dry-run) org-level
// plan_approved of orgID after its latest real `disbanded`, whose Details
// read back (planApprovalFromDetails) and name id, as currentPlanApprovals
// reads them. An approval DECISION is a real `sent` of orgID to
// LeaderIdentity, after one of those approvals in file order, whose
// TextSHA256 is sentTextSHA256 of ApprovalDecisionText of that approval's
// plan and digest: the message approve prints, sent as it is. A `sent` with
// no TextSHA256 (one an older ralph recorded) is not counted, nor is one
// whose Details only start with that message, since Details hold the first
// sendDetailsMaxLen bytes of the text.
func checkApprovalDecision(events []ManifestEvent, orgID, id string) approvalDecisionCheck {
	c := approvalDecisionCheck{orgID: orgID}
	start := 0
	if d, ok := lastDisbandedIndexes(events)[orgID]; ok {
		start = d + 1
	}
	var wants []string // the TextSHA256 of the DECISION of each approval so far
	confirmed := false
	for _, ev := range events[start:] {
		if ev.DryRun || ev.OrgID != orgID {
			continue
		}
		switch {
		case ev.SeatID == "" && ev.Event == EventPlanApproved:
			a, ok := planApprovalFromDetails(ev.Details)
			if !ok || a.InboxID != id {
				continue
			}
			ref := PlanRef{Path: a.Plan, Digest: a.Digest}
			if !c.approved {
				c.approved, c.ref = true, ref
			}
			wants = append(wants, sentTextSHA256(ApprovalDecisionText(ref)))
		case ev.Event == EventSent && ev.SeatID == LeaderIdentity && ev.TextSHA256 != "" && slices.Contains(wants, ev.TextSHA256):
			c.sent = true
			if !sentSubmitUnconfirmed(ev.Details) {
				confirmed = true
			}
		}
	}
	c.submitUnconfirmed = c.sent && !confirmed
	return c
}

// sentSubmitUnconfirmed reports whether a `sent` event's Details carry
// submit_unconfirmed=true among the `key=value ` words Send writes before the
// text (`raw=true `, `via=screen herdr_state=<state> `,
// `submit_unconfirmed=true `), read up to the first word with no `=`, as
// reseatProgressSentTask reads them.
func sentSubmitUnconfirmed(details string) bool {
	text := details
	for {
		word, rest, found := strings.Cut(text, " ")
		if !found || !strings.Contains(word, "=") || strings.Contains(word, "\n") {
			return false
		}
		if word == "submit_unconfirmed=true" {
			return true
		}
		text = rest
	}
}

// approvalDecisionNotSentError is ResolveInboxItem's refusal of item id,
// whose approval c found with no counted approval DECISION sent after it: it
// wraps ErrApprovalDecisionNotSent, gives the `ralph org send` of the
// DECISION on its own lines (ApprovalDecisionCommand, with --state-dir
// stateDir when it is set), then the resolve approve printed
// (ApprovalResolveCommand) on its own lines to run once that send succeeded,
// and last a sentence that names --without-decision: what it is for, and that
// approve's note stays on the resolve it is added to, so a leader woken while
// it waits reads the note as the approval. No line of the error is a complete
// command with --without-decision, and the last line is that sentence, so a
// reader who copies the last command of the error runs a resolve that the
// check refuses again until the send is made, not one that skips the check
// (issue #345).
func approvalDecisionNotSentError(id string, c approvalDecisionCheck, stateDir string) error {
	return fmt.Errorf("org: inbox: resolve %s: refused, with nothing written: %w (org_id %q, plan %s); "+
		"tell the leader with:\n%s\nand once that send succeeded, resolve the item with:\n%s\n"+
		"Adding --without-decision to that resolve closes the item on purpose with no counted DECISION: after you told the leader another way, "+
		"or when a ralph from before this check sent the DECISION (its sent event has no text_sha256, so it is not counted). "+
		"Keep approve's note on that resolve, since a leader woken while it waits reads the note as the approval.",
		id, ErrApprovalDecisionNotSent, c.orgID, c.ref, ApprovalDecisionCommand(c.orgID, c.ref, stateDir),
		ApprovalResolveCommand(id, c.ref, stateDir))
}
