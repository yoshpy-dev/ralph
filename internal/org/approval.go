package org

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// Plan approval (plan docs/plans/active/2026-10-12-org-plan-approval.md,
// FR-7 of docs/specs/2026-10-07-org-multi-org-director.md). In an org that
// `ralph org start --plan` started in per-plan mode (the binding's
// RequiresApproval), the leader writes the feature's plan in the feature
// worktree and asks a human to approve it with an inbox QUESTION whose header
// names the plan and its digest:
//
//	TYPE: QUESTION
//	PLAN: docs/plans/active/<date>-<slug>.md sha256:<what scripts/plan-visual.sh digest prints>
//
// The human runs `ralph org approve <inbox ID>...` (Approve). It records one
// org-level EventPlanApproved per item in the manifest, resolves the items,
// and never writes the plan file (A3): the leader writes `- Status:
// Approved` and `- Approved: <date> sha256:<digest>` in the plan's header
// and commits them. The check reads both the record and the file
// (checkPlanApproved for a TASK to an implementer, which names the plan with
// the same PLAN: header, and checkAnyPlanApproved for an implementer spawn),
// since the leader writes the file and could otherwise approve its own plan
// (A2). The record also holds the file's PlanContentHash, so a change the
// digest does not show is refused too (A9).
//
// None of this is a security boundary: a leader can write the manifest
// itself, and Approve's pane check (approveTripwire) only stops a seat that
// runs approve by mistake. The record keeps who ran approve and where, for a
// later audit.

const (
	// planHeaderKey is the protocol header that names a plan:
	// `PLAN: <path> sha256:<digest>`.
	planHeaderKey = "PLAN"
	// planApprovedStatus is the `- Status:` value of an approved plan.
	planApprovedStatus = "Approved"
	// approverUnknown is the `by=` value when USER is empty or cannot stand
	// as one token (it has whitespace or a control character).
	approverUnknown = "unknown"
)

// contentHashPattern is the shape of PlanContentHash: 64 lowercase hex digits.
var contentHashPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// The keys of an EventPlanApproved's Details, written in this order and each
// as one space-separated `key=value` token. cwd comes last because its value
// is strconv.Quote'd and may hold spaces.
const (
	planApprovedPlanKey      = "plan"
	planApprovedDigestKey    = "digest"
	planApprovedContentKey   = "content"
	planApprovedInboxKey     = "inbox"
	planApprovedByKey        = "by"
	planApprovedHerdrPaneKey = "herdr_pane"
	planApprovedCwdKey       = "cwd"
)

// PlanRef is the plan a `PLAN: <path> sha256:<digest>` header names. Path is
// relative to the org's feature worktree and cleaned, with forward slashes
// (normalizePlanPath); Digest is the 12 hex digits PlanDigest gives.
type PlanRef struct {
	Path   string
	Digest string
}

// String writes r as a PLAN: header value: `<path> sha256:<digest>`.
func (r PlanRef) String() string {
	return r.Path + " sha256:" + r.Digest
}

// ParsePlanHeader reads the plan a typed protocol message names in its
// header block: exactly one PLAN: header (protocol.HeaderValues, the header
// rules of protocol.Parse), whose value is two words, a path and
// `sha256:<12 lowercase hex digits>`. The path must be relative to the
// feature worktree, with no `..` element, and is returned cleaned
// (normalizePlanPath). No PLAN: header, two or more of them (A7: Parse keeps
// the last, so the check and the seat could read different plans), and any
// other shape are errors.
func ParsePlanHeader(text string) (PlanRef, error) {
	ref, err := planRefFromText(text)
	if err != nil {
		return PlanRef{}, fmt.Errorf("org: %w", err)
	}
	return ref, nil
}

// planRefFromText is ParsePlanHeader without the "org: " prefix on its
// errors, for callers that add their own.
func planRefFromText(text string) (PlanRef, error) {
	values := protocol.HeaderValues(text, planHeaderKey)
	switch len(values) {
	case 0:
		return PlanRef{}, errors.New("no PLAN: header: the message's header must name the plan as PLAN: <path in the feature worktree> sha256:<digest>")
	case 1:
		return parsePlanRef(values[0])
	default:
		return PlanRef{}, fmt.Errorf("%d PLAN: headers: write exactly one (a header given twice keeps only its last value, "+
			"so the check and the seat could read different plans)", len(values))
	}
}

// parsePlanRef reads a PLAN: header value: `<path> sha256:<digest>`.
func parsePlanRef(value string) (PlanRef, error) {
	words := strings.Fields(value)
	if len(words) != 2 {
		return PlanRef{}, fmt.Errorf("PLAN: %q is not the two words <path> sha256:<digest> (a plan path cannot hold whitespace)", value)
	}
	digest, ok := strings.CutPrefix(words[1], "sha256:")
	if !ok || !approvedDigestPattern.MatchString(digest) {
		return PlanRef{}, fmt.Errorf("PLAN: %q: the second word %q is not sha256: and the 12 lowercase hex digits scripts/plan-visual.sh digest prints",
			value, words[1])
	}
	path, err := normalizePlanPath(words[0])
	if err != nil {
		return PlanRef{}, fmt.Errorf("PLAN: %q: %w", value, err)
	}
	return PlanRef{Path: path, Digest: digest}, nil
}

// normalizePlanPath checks the path of a PLAN: header and returns it cleaned
// with forward slashes (filepath.ToSlash(filepath.Clean(raw))). An empty or
// absolute path, a path with a `..` element or a control character, and a
// path that cleans to `.` (the worktree itself) are errors.
func normalizePlanPath(raw string) (string, error) {
	switch {
	case raw == "":
		return "", errors.New("the plan path is empty: write it relative to the feature worktree")
	case filepath.IsAbs(raw):
		return "", fmt.Errorf("the plan path %q is absolute: write it relative to the feature worktree", raw)
	case strings.ContainsFunc(raw, unicode.IsControl):
		return "", fmt.Errorf("the plan path %q has a control character", raw)
	case slices.Contains(strings.Split(filepath.ToSlash(raw), "/"), ".."):
		return "", fmt.Errorf("the plan path %q has a .. element: write it relative to the feature worktree, inside it", raw)
	}
	p := filepath.ToSlash(filepath.Clean(raw))
	if p == "." {
		return "", fmt.Errorf("the plan path %q names the feature worktree itself, not a plan file in it", raw)
	}
	return p, nil
}

// resolvePlanFile returns the absolute path, with symlinks resolved, of the
// plan at rel (a normalizePlanPath result) in the feature worktree. Both the
// worktree and the file are resolved with filepath.EvalSymlinks, and the file
// must be under the resolved worktree (a symlink that points outside it is
// refused) and a regular file. The errors name the worktree and have no
// "org: " prefix; the callers add theirs.
func resolvePlanFile(worktree, rel string) (string, error) {
	root, err := filepath.EvalSymlinks(worktree)
	if err != nil {
		return "", fmt.Errorf("the feature worktree %s cannot be read: %w", worktree, pathErrorCause(err))
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return "", fmt.Errorf("plan %s in the feature worktree %s: %w", rel, worktree, pathErrorCause(err))
	}
	inside, err := filepath.Rel(root, resolved)
	if err != nil || inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("plan %s resolves to %s, outside the feature worktree %s", rel, resolved, worktree)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("plan %s in the feature worktree %s: %w", rel, worktree, pathErrorCause(err))
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("plan %s in the feature worktree %s is not a regular file", rel, worktree)
	}
	return resolved, nil
}

// featurePlanHeaderLines returns the indexes in lines of a feature plan's
// first `- Status:` line and first `- Approved:` line before its first line
// starting `## ` (the `# <title>` line is not one), -1 for one that is not
// there. These are the two lines the leader writes after the approval, read
// the way /implement's approval check reads a plan.
func featurePlanHeaderLines(lines []string) (status, approved int) {
	status, approved = -1, -1
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "## "):
			return status, approved
		case status < 0 && strings.HasPrefix(line, "- Status:"):
			status = i
		case approved < 0 && strings.HasPrefix(line, "- Approved:"):
			approved = i
		}
	}
	return status, approved
}

// featurePlanHeader is what a feature plan's header lines
// (featurePlanHeaderLines) say about its approval.
type featurePlanHeader struct {
	hasStatus, hasApproved bool   // the plan has the `- Status:` / `- Approved:` line
	status                 string // the `- Status:` value, trimmed
	approvedDigest         string // approvedDigestFrom the `- Approved:` value ("" without sha256:<12 hex>)
}

// readFeaturePlanHeader reads a feature plan's header lines from its bytes.
func readFeaturePlanHeader(data []byte) featurePlanHeader {
	lines := planLines(data)
	status, approved := featurePlanHeaderLines(lines)
	var h featurePlanHeader
	if status >= 0 {
		h.hasStatus = true
		h.status = strings.TrimSpace(strings.TrimPrefix(lines[status], "- Status:"))
	}
	if approved >= 0 {
		h.hasApproved = true
		h.approvedDigest = approvedDigestFrom(strings.TrimPrefix(lines[approved], "- Approved:"))
	}
	return h
}

// PlanContentHash returns the lowercase hex SHA-256 (64 digits) of a feature
// plan that Approve records and the check compares (A9 of plan
// 2026-10-12-org-plan-approval). It reads the file the way PlanDigest does
// (planLines, each kept line followed by "\n", a line starting, after
// whitespace, `- [x]` or `- [X]` read as `- [ ]`), but leaves out only the
// first `- Status:` line and the first `- Approved:` line before the first
// `## ` line (featurePlanHeaderLines), the two lines the leader writes after
// the approval. Unlike PlanDigest it reads the `## Progress checklist`
// section, every `- Branch:` line, and every other `- Status:` or
// `- Approved:` line: PlanDigest skips those wherever they stand, so an
// instruction added there after the approval would leave the approved digest
// as it was. The digest shown to a human (the QUESTION's PLAN:, the plan's
// `- Approved:` line) stays PlanDigest, the one /plan and
// scripts/plan-visual.sh digest use; this hash is only compared inside
// ralph. After the approval the leader may write those two lines and tick
// checkboxes, and nothing else.
func PlanContentHash(data []byte) string {
	lines := planLines(data)
	status, approved := featurePlanHeaderLines(lines)
	h := sha256.New()
	for i, line := range lines {
		if i == status || i == approved {
			continue
		}
		if j := checkedBoxIndex(line); j >= 0 {
			line = line[:j] + " " + line[j+1:]
		}
		_, _ = io.WriteString(h, line+"\n")
	}
	return hex.EncodeToString(h.Sum(nil))
}

// planApproval is one EventPlanApproved: what Approve checked and recorded,
// and who ran it.
type planApproval struct {
	Plan    string // PlanRef.Path, relative to the feature worktree
	Digest  string // PlanRef.Digest, the file's PlanDigest when approved
	Content string // the file's PlanContentHash when approved
	InboxID string // the QUESTION's inbox item
	By      string // USER, or approverUnknown
	// HerdrPane: HERDR_PANE_ID was set, to no active seat's pane (approveTripwire).
	HerdrPane bool
	Cwd       string // the directory approve ran in
}

// details is the Details of a's EventPlanApproved (see the planApproved*Key
// constants).
func (a planApproval) details() string {
	pane := "unset"
	if a.HerdrPane {
		pane = "set"
	}
	return planApprovedPlanKey + "=" + a.Plan +
		" " + planApprovedDigestKey + "=" + a.Digest +
		" " + planApprovedContentKey + "=" + a.Content +
		" " + planApprovedInboxKey + "=" + a.InboxID +
		" " + planApprovedByKey + "=" + a.By +
		" " + planApprovedHerdrPaneKey + "=" + pane +
		" " + planApprovedCwdKey + "=" + strconv.Quote(a.Cwd)
}

// planApprovalFromDetails reads an EventPlanApproved's Details back. ok is
// false, and the record approves nothing, when a key before cwd is given
// twice or a token is not `key=value`, or when plan is not a clean plan path
// (normalizePlanPath), digest is not 12 lowercase hex digits, content is not
// 64, or inbox is empty. Keys it does not know are skipped. by, herdr_pane
// and cwd are read on a best-effort basis: they are for an audit, not the
// check.
func planApprovalFromDetails(details string) (a planApproval, ok bool) {
	head, cwd, hasCwd := strings.Cut(details, " "+planApprovedCwdKey+"=")
	values := make(map[string]string)
	for _, token := range strings.Fields(head) {
		key, value, found := strings.Cut(token, "=")
		if _, dup := values[key]; !found || dup {
			return planApproval{}, false
		}
		values[key] = value
	}
	a = planApproval{
		Plan: values[planApprovedPlanKey], Digest: values[planApprovedDigestKey],
		Content: values[planApprovedContentKey], InboxID: values[planApprovedInboxKey],
		By: values[planApprovedByKey], HerdrPane: values[planApprovedHerdrPaneKey] == "set",
	}
	if clean, err := normalizePlanPath(a.Plan); err != nil || clean != a.Plan ||
		!approvedDigestPattern.MatchString(a.Digest) || !contentHashPattern.MatchString(a.Content) || a.InboxID == "" {
		return planApproval{}, false
	}
	if hasCwd {
		if s, err := strconv.Unquote(cwd); err == nil {
			a.Cwd = s
		}
	}
	return a, true
}

// currentPlanApprovals returns, in file order, the approvals recorded for
// orgID in its current life: its real (non-dry-run) org-level
// EventPlanApproved after its latest real `disbanded` (all of them when it
// has none), so an approval from an earlier life of the same org_id never
// counts (AC9). A record whose Details do not read back
// (planApprovalFromDetails) is left out.
func currentPlanApprovals(events []ManifestEvent, orgID string) []planApproval {
	start := 0
	if d, ok := lastDisbandedIndexes(events)[orgID]; ok {
		start = d + 1
	}
	var approvals []planApproval
	for _, ev := range events[start:] {
		if ev.DryRun || ev.OrgID != orgID || ev.SeatID != "" || ev.Event != EventPlanApproved {
			continue
		}
		if a, ok := planApprovalFromDetails(ev.Details); ok {
			approvals = append(approvals, a)
		}
	}
	return approvals
}

// checkPlanApproved returns nil when the plan ref (a TASK's PLAN: header,
// ParsePlanHeader) is approved for orgID, bound to f: orgID's current life
// has a plan_approved for the same path and digest (currentPlanApprovals),
// the file at that path in f's worktree (resolvePlanFile) has that digest
// now (PlanDigest), its header's first `- Status:` is Approved and its first
// `- Approved:` names that digest (featurePlanHeaderLines), and its
// PlanContentHash equals the content one of those records holds. Each way to
// fail has its own error, naming what to do.
func checkPlanApproved(events []ManifestEvent, orgID string, f *FeatureBinding, ref PlanRef) error {
	if f == nil || !f.Complete() {
		return fmt.Errorf("org: org_id %q has no complete split plan feature binding, so no plan approval can be checked", orgID)
	}
	var matching []planApproval
	latest := ""
	for _, a := range currentPlanApprovals(events, orgID) {
		if a.Plan != ref.Path {
			continue
		}
		latest = a.Digest
		if a.Digest == ref.Digest {
			matching = append(matching, a)
		}
	}
	if len(matching) == 0 {
		since := ""
		if latest != "" {
			since = fmt.Sprintf(" (its latest approval is of sha256:%s)", latest)
		}
		return fmt.Errorf("org: org_id %q has no approval recorded for plan %s since its last disband%s: "+
			"the leader asks for it with an inbox QUESTION whose header has PLAN: %s, and a human runs ralph org approve <inbox ID>",
			orgID, ref, since, ref)
	}
	return checkApprovedPlanFile(orgID, f, ref, matching)
}

// checkApprovedPlanFile is the file half of checkPlanApproved: approvals are
// the records of orgID's current life for ref's path and digest.
func checkApprovedPlanFile(orgID string, f *FeatureBinding, ref PlanRef, approvals []planApproval) error {
	abs, err := resolvePlanFile(f.Worktree, ref.Path)
	if err != nil {
		return fmt.Errorf("org: org_id %q: %w", orgID, err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return fmt.Errorf("org: org_id %q: plan %s: %w", orgID, ref.Path, pathErrorCause(err))
	}
	if d := PlanDigest(data); d != ref.Digest {
		return fmt.Errorf("org: plan %s of org_id %q changed after its approval: approved digest %s, current digest %s: "+
			"ask for the approval again with a new QUESTION that names the current digest", ref.Path, orgID, ref.Digest, d)
	}
	h := readFeaturePlanHeader(data)
	write := fmt.Sprintf("write - Status: %s and - Approved: <date> sha256:%s in its header, before the first ## heading, and commit them",
		planApprovedStatus, ref.Digest)
	switch {
	case !h.hasStatus:
		return fmt.Errorf("org: plan %s of org_id %q has no - Status: line before its first ## heading: %s", ref.Path, orgID, write)
	case h.status != planApprovedStatus:
		return fmt.Errorf("org: plan %s of org_id %q is not marked approved: its - Status: is %q, want %s: %s",
			ref.Path, orgID, h.status, planApprovedStatus, write)
	case !h.hasApproved:
		return fmt.Errorf("org: plan %s of org_id %q has no - Approved: line before its first ## heading: %s", ref.Path, orgID, write)
	case h.approvedDigest != ref.Digest:
		named := "no sha256:<12 hex digits>"
		if h.approvedDigest != "" {
			named = "sha256:" + h.approvedDigest
		}
		return fmt.Errorf("org: plan %s of org_id %q: its - Approved: line names %s, not the approved sha256:%s: %s",
			ref.Path, orgID, named, ref.Digest, write)
	}
	content := PlanContentHash(data)
	if !slices.ContainsFunc(approvals, func(a planApproval) bool { return a.Content == content }) {
		return fmt.Errorf("org: plan %s of org_id %q changed after its approval outside its - Status: and - Approved: lines and checkbox marks, "+
			"which its digest %s does not show (the digest skips the %s section and every - Status:, - Approved: and - Branch: line): "+
			"undo the change, or ask for the approval again", ref.Path, orgID, ref.Digest, planProgressHeading)
	}
	return nil
}

// checkAnyPlanApproved returns nil when at least one approval of orgID's
// current life passes checkPlanApproved with its own path and digest: the
// check of an implementer spawn, which names no plan. With no approval at all
// the error says no plan of the org was approved; otherwise it gives the
// reason the latest approval does not pass.
func checkAnyPlanApproved(events []ManifestEvent, orgID string, f *FeatureBinding) error {
	approvals := currentPlanApprovals(events, orgID)
	if len(approvals) == 0 {
		return fmt.Errorf("org: no plan of org_id %q has been approved since its last disband: "+
			"the leader asks for the approval with an inbox QUESTION whose header has PLAN: <path> sha256:<digest>, "+
			"and a human runs ralph org approve <inbox ID>", orgID)
	}
	var latestErr error
	for i := len(approvals) - 1; i >= 0; i-- {
		a := approvals[i]
		err := checkPlanApproved(events, orgID, f, PlanRef{Path: a.Plan, Digest: a.Digest})
		if err == nil {
			return nil
		}
		if latestErr == nil {
			latestErr = err
		}
	}
	return fmt.Errorf("org: none of the %d plan approvals of org_id %q since its last disband passes the check; the latest: %s",
		len(approvals), orgID, strings.TrimPrefix(latestErr.Error(), "org: "))
}

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
