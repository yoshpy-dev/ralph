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
// no approval DECISION was sent to the org's leader after that approval
// (approvalDecisionSent).
var ErrApprovalDecisionNotSent = errors.New("the approval of this inbox item's plan is recorded, " +
	"but no approval DECISION was sent to the org's leader after it")

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
	// LeaderNotTold reports that the item was resolved with withoutDecision
	// while no approval DECISION was sent, so the CLI says the leader was
	// not told.
	LeaderNotTold bool
}

// resolveAfterCheckHook, when set (tests only), runs inside ResolveInboxItem's
// hold of the manifest lock, after the check decided to resolve the item and
// before the inbox is written.
var resolveAfterCheckHook func()

// ResolveInboxItem resolves inbox item id with note, as InboxStore.Resolve
// does, after one more check. The note is checked first (ValidateInboxNote),
// and with no inbox file the ID is unknown (ErrInboxUnknownID), as
// InboxStore.Resolve says, so neither writes or creates anything. Then, under
// the manifest lock, the lock Approve records under, it reads the inbox and
// the manifest: when the item is an open or acked QUESTION whose header names
// a plan (planRefFromText) and its org's current life holds the approval of
// this item, the item is resolved only when an approval DECISION was sent to
// the org's leader after that approval (approvalDecisionSent). Otherwise it
// is refused with ErrApprovalDecisionNotSent and nothing written, and the
// error gives the `ralph org send` of the DECISION (ApprovalDecisionCommand,
// with --state-dir InboxHintStateDir when it is set) and names
// --without-decision; with withoutDecision set, the item is resolved all the
// same and the result's LeaderNotTold says so. Every other item (no PLAN:
// header, not a QUESTION, no approval recorded, an approval from before the
// org's latest disband) is resolved as before, and an unknown or resolved ID
// gets InboxStore.Resolve's error.
//
// The inbox is written (InboxStore.Resolve, which takes the inbox lock) while
// the manifest lock is still held, so an approve cannot record the approval
// between the check and the write: an approve that records first makes the
// resolve refuse, and an approve that waits for the lock finds the item
// resolved afterwards (checkResolvedMeanwhile). The locks are taken in the
// order manifest, then inbox; no code takes the manifest lock while it holds
// the inbox lock.
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
	var res InboxResolveResult
	err := withManifestLock(filepath.Dir(o.Manifest.Path()), func() error {
		c, err := o.readApprovalDecisionCheck(id)
		if err != nil {
			return err
		}
		switch {
		case !c.approved:
		case c.sent:
			res = InboxResolveResult{OrgID: c.orgID, Plan: c.ref, SubmitUnconfirmed: c.submitUnconfirmed}
		case withoutDecision:
			res = InboxResolveResult{OrgID: c.orgID, Plan: c.ref, LeaderNotTold: true}
		default:
			return approvalDecisionNotSentError(id, c, o.InboxHintStateDir)
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

// approvalDecisionCheck is what approvalDecisionSent found for one inbox
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

// readApprovalDecisionCheck reads the inbox and, for an open or acked
// QUESTION item id whose header names a plan, the manifest, and returns
// approvalDecisionSent for it. An item that is unknown, resolved, not a
// QUESTION, or names no plan gives a check with approved false, which leaves
// the item to InboxStore.Resolve. ResolveInboxItem calls it holding the
// manifest lock.
func (o *Org) readApprovalDecisionCheck(id string) (approvalDecisionCheck, error) {
	inbox, err := o.Inbox.Read()
	if err != nil {
		return approvalDecisionCheck{}, err
	}
	i := slices.IndexFunc(inbox.Items, func(item InboxItem) bool { return item.ID == id })
	if i < 0 {
		return approvalDecisionCheck{}, nil
	}
	item := inbox.Items[i]
	if item.State == InboxStateResolved || item.Type != protocol.TypeQuestion {
		return approvalDecisionCheck{}, nil
	}
	if _, err := planRefFromText(item.Body); err != nil {
		return approvalDecisionCheck{}, nil
	}
	rr, err := o.Manifest.Read()
	if err != nil {
		return approvalDecisionCheck{}, fmt.Errorf("org: inbox: resolve %s: read manifest: %w", id, err)
	}
	return approvalDecisionSent(rr.Events, item.OrgID, id), nil
}

// approvalDecisionSent reads events for the approvals of inbox item id in
// orgID's current life and the approval DECISIONs sent after them. An
// approval is a real (non-dry-run) org-level plan_approved of orgID after its
// latest real `disbanded`, whose Details read back (planApprovalFromDetails)
// and name id, as currentPlanApprovals reads them. An approval DECISION is a
// real `sent` of orgID to LeaderIdentity, after one of those approvals in
// file order, whose TextSHA256 is sentTextSHA256 of ApprovalDecisionText of
// that approval's plan and digest: the message approve prints, sent as it is.
// A `sent` with no TextSHA256 (one an older ralph recorded) is not counted,
// nor is one whose Details only start with that message, since Details hold
// the first sendDetailsMaxLen bytes of the text.
func approvalDecisionSent(events []ManifestEvent, orgID, id string) approvalDecisionCheck {
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
// whose approval c found with no approval DECISION sent after it: it wraps
// ErrApprovalDecisionNotSent, gives the `ralph org send` of the DECISION on
// its own lines (ApprovalDecisionCommand, with --state-dir stateDir when it
// is set), says to run the resolve again once that send succeeded, and names
// --without-decision.
func approvalDecisionNotSentError(id string, c approvalDecisionCheck, stateDir string) error {
	return fmt.Errorf("org: inbox: resolve %s: refused, with nothing written: %w (org_id %q, plan %s); "+
		"tell the leader with:\n%s\nand once that send succeeded, run the resolve again. "+
		"To resolve the item without the DECISION on purpose, run the resolve with --without-decision: "+
		"for example after you told the leader another way, or when a ralph from before this check sent the DECISION "+
		"(its sent event has no text_sha256, so it is not counted)",
		id, ErrApprovalDecisionNotSent, c.orgID, c.ref, ApprovalDecisionCommand(c.orgID, c.ref, stateDir))
}
