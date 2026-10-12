package org

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// The `ralph org approve` verb (Approve). The PLAN: header, the plan path and
// hashes, the plan_approved record and the approval check it feeds are in
// approval.go.

// ApproveParams describes one `ralph org approve` call.
type ApproveParams struct {
	// IDs are the inbox items to approve (e<N>), each a leader's QUESTION
	// for its plan: at least one, none given twice.
	IDs []string
	// Cwd is the directory recorded as where approve ran; empty means
	// os.Getwd().
	Cwd string
}

// ApprovedPlan is one inbox item whose approval Approve recorded.
type ApprovedPlan struct {
	ID     string // the inbox item
	OrgID  string // the item's org
	Plan   string // the plan's path in the org's feature worktree (PlanRef.Path)
	Digest string // the approved digest (PlanRef.Digest)
	// RequiresApproval is the org binding's RequiresApproval. false means
	// the org was started in split mode or by an older start: the approval
	// is recorded, but the org's check does not read it, so the CLI says so.
	RequiresApproval bool
	// Resolved reports that the inbox item was resolved after the record,
	// with the note `approved <path> sha256:<digest>`.
	Resolved bool
}

// ApproveRefusal is one inbox item Approve refused, with the reason.
type ApproveRefusal struct {
	ID  string
	Err error
}

// ApproveResult is Approve's return value. Err is set on every refusal and
// failure and is the one error to print: on a refusal it lists every refused
// item, one per line, and then Refused has them too and Approved is empty,
// since nothing was recorded. Approved holds, in argument order, the items
// whose approval is in the manifest, which may be fewer than asked when an
// append failed midway (Err then names the items not recorded and the
// command to run again), and an item there with Resolved false has its
// `ralph org inbox resolve` command in Err.
type ApproveResult struct {
	Approved []ApprovedPlan
	Refused  []ApproveRefusal
	Err      error
}

// approveCandidate is one inbox item while Approve checks it.
type approveCandidate struct {
	id      string
	item    InboxItem
	ref     PlanRef
	binding *FeatureBinding
	content string // PlanContentHash of the plan file, read under the manifest lock
	err     error  // why the item is refused; nil while it passes
}

// Approve records a human's approval of the plans that inbox QUESTION items
// p.IDs name (`ralph org approve`, A2 and A4 of plan
// 2026-10-12-org-plan-approval). It refuses an empty list and an ID given
// twice. When HERDR_PANE_ID is the pane of an active seat (approveTripwire),
// it refuses before it reads the inbox or a plan. Every item is then checked
// before anything is written: it is in the inbox and not resolved, a
// QUESTION whose header names a plan (ParsePlanHeader), its org holds a
// complete split plan feature binding (ActiveFeature), the plan resolves to a
// regular file in that binding's worktree (resolvePlanFile), and the file's
// PlanDigest equals the header's digest. The part that reads the manifest and
// the plan is done under the manifest lock, and the plan_approved records are
// appended under the same lock hold, so what is recorded, the
// PlanContentHash included, is what was checked. If one item fails, nothing
// is recorded or resolved and every failure comes back. After the lock each
// recorded item is resolved; a resolve that fails keeps the record. An org
// whose binding does not require an approval (split mode, or an older start)
// is recorded too, and marked. Approve never calls herdr or agmsg and never
// writes the plan file (A3).
func (o *Org) Approve(p ApproveParams) ApproveResult {
	if err := checkApproveIDs(p.IDs); err != nil {
		return ApproveResult{Err: err}
	}
	if o.Inbox == nil {
		return ApproveResult{Err: errors.New("org: approve: no inbox store is configured")}
	}
	rr, err := o.Manifest.Read()
	if err != nil {
		return ApproveResult{Err: fmt.Errorf("org: approve: read manifest: %w", err)}
	}
	pane := o.getenv(herdrPaneIDEnv)
	if err := approveTripwire(rr.Events, pane); err != nil {
		return ApproveResult{Err: err}
	}
	inbox, err := o.Inbox.Read()
	if err != nil {
		return ApproveResult{Err: fmt.Errorf("org: approve: %w", err)}
	}
	candidates := approveCandidates(inbox.Items, p.IDs)
	record := planApproval{By: approverName(o.getenv("USER")), HerdrPane: pane != "", Cwd: p.Cwd}
	if record.Cwd == "" {
		// The directory is audit information, not part of the check: when
		// it cannot be read, the record holds cwd="" rather than refusing
		// the approval.
		if wd, err := os.Getwd(); err == nil {
			record.Cwd = wd
		}
	}

	var res ApproveResult
	if slices.ContainsFunc(candidates, func(c *approveCandidate) bool { return c.err == nil }) {
		var lockErr error
		if res.Approved, res.Err, lockErr = o.recordUnderLock(candidates, record); lockErr != nil {
			return ApproveResult{Err: lockErr}
		}
	}
	if refused, ok := approveRefusals(candidates); ok {
		return refused
	}
	res.Err = errors.Join(res.Err, o.resolveApproved(res.Approved))
	return res
}

// recordUnderLock takes the manifest lock, reads the manifest, and runs
// checkApproveCandidate on every candidate that passed the inbox checks; when
// every candidate passes, it appends their records under the same lock hold
// (recordApprovals) and returns the items recorded and the error of a failed
// append. lockErr is set when the lock could not be taken or the manifest
// could not be read; nothing was checked or recorded then.
func (o *Org) recordUnderLock(candidates []*approveCandidate, record planApproval) (recorded []ApprovedPlan, appendErr, lockErr error) {
	lockErr = withManifestLock(filepath.Dir(o.Manifest.Path()), func() error {
		rr, err := o.Manifest.Read()
		if err != nil {
			return fmt.Errorf("org: approve: read manifest: %w", err)
		}
		for _, c := range candidates {
			if c.err == nil {
				c.binding, c.content, c.err = checkApproveCandidate(rr.Events, c)
			}
		}
		if slices.ContainsFunc(candidates, func(c *approveCandidate) bool { return c.err != nil }) {
			return nil
		}
		recorded, appendErr = o.recordApprovals(candidates, record)
		return nil
	})
	return recorded, appendErr, lockErr
}

// approveRefusals is Approve's result when candidates were refused: the
// refusals in argument order, and an error with a line that says nothing was
// recorded and then each refusal on its own line. ok is false when none was.
func approveRefusals(candidates []*approveCandidate) (res ApproveResult, ok bool) {
	errs := []error{nil}
	for _, c := range candidates {
		if c.err != nil {
			res.Refused = append(res.Refused, ApproveRefusal{ID: c.id, Err: c.err})
			errs = append(errs, c.err)
		}
	}
	if len(res.Refused) == 0 {
		return ApproveResult{}, false
	}
	errs[0] = fmt.Errorf("org: approve: nothing recorded, since %d of the %d inbox items cannot be approved", len(res.Refused), len(candidates))
	res.Err = errors.Join(errs...)
	return res, true
}

// checkApproveIDs refuses an empty ID list and an ID given twice.
func checkApproveIDs(ids []string) error {
	if len(ids) == 0 {
		return errors.New("org: approve: name at least one inbox item ID (e<N>), the leader's QUESTION for its plan")
	}
	for i, id := range ids {
		if slices.Contains(ids[:i], id) {
			return fmt.Errorf("org: approve: inbox item %s is given twice", PrintableInboxText(id, false))
		}
	}
	return nil
}

// approveTripwire refuses approve when pane (HERDR_PANE_ID) is the recorded
// pane of an active real seat of any org (Roster): a seat that runs approve
// from its own pane, by mistake. It is not a boundary (A2): unsetting the
// variable passes, and a leader can write the manifest. An unset pane, and
// one that is no active seat's, pass.
func approveTripwire(events []ManifestEvent, pane string) error {
	if pane == "" {
		return nil
	}
	for _, s := range Roster(events, RosterOptions{}) {
		if s.Active && s.PaneID == pane {
			return fmt.Errorf("org: approve: refused, since HERDR_PANE_ID %q is the pane of the active seat %q of org_id %q: "+
				"ralph org approve records a human's approval of a plan, so a human runs it from a terminal outside the seats' panes "+
				"(a human in a herdr pane that only shares its id with a seat of another herdr server runs it from a terminal outside herdr)",
				pane, s.SeatID, s.OrgID)
		}
	}
	return nil
}

// approverName is the `by=` value for USER: USER itself, or approverUnknown
// when it is empty or cannot stand as one token.
func approverName(user string) string {
	if user == "" || strings.ContainsFunc(user, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		return approverUnknown
	}
	return user
}

// approveNote is the resolve note of an approved item.
func approveNote(ref PlanRef) string {
	return "approved " + ref.String()
}

// approveCandidates runs the checks that read only the inbox, in argument
// order: the item exists and is not resolved (open or acked), is a QUESTION,
// names one plan in its header (planRefFromText), and its resolve note
// (approveNote) passes ValidateInboxNote, so the resolve after the record
// does not fail on it.
func approveCandidates(items []InboxItem, ids []string) []*approveCandidate {
	candidates := make([]*approveCandidate, 0, len(ids))
	for _, id := range ids {
		c := &approveCandidate{id: id}
		candidates = append(candidates, c)
		shown := PrintableInboxText(id, false)
		i := slices.IndexFunc(items, func(item InboxItem) bool { return item.ID == id })
		if i < 0 {
			c.err = fmt.Errorf("org: approve %s: %w", shown, ErrInboxUnknownID)
			continue
		}
		c.item = items[i]
		switch {
		case c.item.State == InboxStateResolved:
			c.err = fmt.Errorf("org: approve %s: %w", id, ErrInboxResolved)
			continue
		case c.item.Type != protocol.TypeQuestion:
			c.err = fmt.Errorf("org: approve %s: the item is a %s, not a QUESTION: a leader asks for a plan's approval with a QUESTION "+
				"whose header has PLAN: <path> sha256:<digest>", id, PrintableInboxText(c.item.Type, false))
			continue
		}
		ref, err := planRefFromText(c.item.Body)
		if err != nil {
			c.err = fmt.Errorf("org: approve %s: %w", id, err)
			continue
		}
		if err := ValidateInboxNote(approveNote(ref)); err != nil {
			c.err = fmt.Errorf("org: approve %s: the item could not be resolved with the note %q: %s",
				id, approveNote(ref), strings.TrimPrefix(err.Error(), "org: inbox: "))
			continue
		}
		c.ref = ref
	}
	return candidates
}

// checkApproveCandidate runs Approve's checks that read the manifest (events,
// read under the manifest lock) and the plan file, and returns the binding and
// the file's PlanContentHash.
func checkApproveCandidate(events []ManifestEvent, c *approveCandidate) (*FeatureBinding, string, error) {
	orgID := c.item.OrgID
	f := ActiveFeature(events, orgID)
	switch {
	case f == nil:
		return nil, "", fmt.Errorf("org: approve %s: org_id %q holds no split plan feature binding since its last disband "+
			"(only an org that ralph org start --plan started has a feature worktree and a plan to approve)", c.id, orgID)
	case !f.Complete():
		return nil, "", fmt.Errorf("org: approve %s: org_id %q is bound to %s: disband it and start it again", c.id, orgID, f)
	}
	abs, err := resolvePlanFile(f.Worktree, c.ref.Path)
	if err != nil {
		return nil, "", fmt.Errorf("org: approve %s: org_id %q: %w", c.id, orgID, err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, "", fmt.Errorf("org: approve %s: org_id %q: plan %s: %w", c.id, orgID, c.ref.Path, pathErrorCause(err))
	}
	if d := PlanDigest(data); d != c.ref.Digest {
		return nil, "", fmt.Errorf("org: approve %s: plan %s of org_id %q has digest %s now, not the sha256:%s the QUESTION names: "+
			"the plan changed after the leader asked, so the leader asks again with the current digest", c.id, c.ref.Path, orgID, d, c.ref.Digest)
	}
	return f, PlanContentHash(data), nil
}

// recordApprovals appends one EventPlanApproved per candidate, in order,
// under the manifest lock Approve holds; record carries the fields every
// record shares (by, herdr_pane, cwd). It returns the items recorded. When an
// append fails, it stops there, and the error names the items recorded and
// not recorded and the approve command to run again for the latter.
func (o *Org) recordApprovals(candidates []*approveCandidate, record planApproval) ([]ApprovedPlan, error) {
	var recorded []ApprovedPlan
	for i, c := range candidates {
		a := record
		a.Plan, a.Digest, a.Content, a.InboxID = c.ref.Path, c.ref.Digest, c.content, c.id
		if err := o.appendEvent(ManifestEvent{
			TS: o.now(), OrgID: c.item.OrgID, SeatID: "", Event: EventPlanApproved, Details: a.details(),
		}); err != nil {
			var left []string
			for _, rest := range candidates[i:] {
				left = append(left, rest.id)
			}
			done := "nothing was recorded"
			if len(recorded) > 0 {
				ids := make([]string, 0, len(recorded))
				for _, r := range recorded {
					ids = append(ids, r.ID)
				}
				done = "recorded: " + strings.Join(ids, ", ")
			}
			return recorded, fmt.Errorf("org: approve: recording the approval of %s failed (%w); %s; not recorded: %s: run `%s` again",
				c.id, err, done, strings.Join(left, ", "), approveCommand(left, o.InboxHintStateDir))
		}
		recorded = append(recorded, ApprovedPlan{
			ID: c.id, OrgID: c.item.OrgID, Plan: c.ref.Path, Digest: c.ref.Digest,
			RequiresApproval: c.binding.RequiresApproval(),
		})
	}
	return recorded, nil
}

// approveCommand is the `ralph org approve <id>...` command an error tells
// the reader to run, with --state-dir stateDir when it is set (the rule of
// InboxCommand).
func approveCommand(ids []string, stateDir string) string {
	command := "ralph org approve " + strings.Join(ids, " ")
	if stateDir != "" {
		command += " --state-dir " + shellQuote(stateDir)
	}
	return command
}

// ApprovalDecisionText is the DECISION message that tells a feature org's
// leader a human approved the plan ref names: TYPE: DECISION, a PLAN: header
// with the plan and digest of the leader's QUESTION, and a body that starts
// with `approved`, which the leader prompt reads as the approval. `ralph org
// approve` prints it in ApprovalDecisionCommand and never sends it (A3).
func ApprovalDecisionText(ref PlanRef) string {
	return "TYPE: " + protocol.TypeDecision + "\n" + planHeaderKey + ": " + ref.String() + "\n\napproved"
}

// ApprovalDecisionCommand is the one POSIX shell command, `ralph org send`,
// that sends ApprovalDecisionText(ref) to the leader seat (LeaderIdentity)
// of orgID, with --state-dir stateDir when it is set (the rule of
// InboxCommand). The --text value is single-quoted with its line breaks kept
// (shellQuote). An orgID that is not an identifier (ValidateIdentifier),
// which only a hand-edited ledger holds, is quoted too.
func ApprovalDecisionCommand(orgID string, ref PlanRef, stateDir string) string {
	orgWord := orgID
	if ValidateIdentifier("org_id", orgID) != nil {
		orgWord = shellQuote(orgID)
	}
	command := "ralph org send --org-id " + orgWord + " --to " + LeaderIdentity
	if stateDir != "" {
		command += " --state-dir " + shellQuote(stateDir)
	}
	return command + " --text " + shellQuote(ApprovalDecisionText(ref))
}

// resolveApproved resolves each recorded item with its approveNote and sets
// Resolved on it. A resolve that fails keeps the approval record; the error
// names those items and the `ralph org inbox resolve` command (InboxCommand
// with --note) that finishes each.
func (o *Org) resolveApproved(approved []ApprovedPlan) error {
	var failed, commands []string
	for i := range approved {
		a := &approved[i]
		note := approveNote(PlanRef{Path: a.Plan, Digest: a.Digest})
		if err := o.Inbox.Resolve(a.ID, note); err != nil {
			failed = append(failed, fmt.Sprintf("%s (%v)", a.ID, err))
			commands = append(commands, "`"+InboxCommand("resolve", a.ID, o.InboxHintStateDir)+" --note "+shellQuote(note)+"`")
			continue
		}
		a.Resolved = true
	}
	if len(failed) == 0 {
		return nil
	}
	return fmt.Errorf("org: approve: the approval is recorded, but the inbox item could not be resolved: %s; run %s to finish",
		strings.Join(failed, ", "), strings.Join(commands, " and "))
}
