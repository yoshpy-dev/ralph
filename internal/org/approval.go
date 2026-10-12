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
	// implementerRole is the role (prompts/implementer.md) whose spawn and
	// whose TASKs the approval check covers. Spawn and Send compare a seat's
	// role with it, not its seat id.
	implementerRole = "implementer"
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

// ErrPlanNotApproved is what every refusal of the approval check is
// (errors.Is): a TASK to an implementer seat (Send) and an implementer spawn
// (Spawn) in an org whose binding RequiresApproval.
var ErrPlanNotApproved = errors.New("org: no approved plan passes the approval check")

// planNotApprovedNext ends every refusal of the approval check: what a
// leader does instead of trying again (the full procedure is in
// prompts/leader.md).
const planNotApprovedNext = "; a leader does not retry this, send it with --raw, or run ralph org approve itself, " +
	"but escalates BLOCKED with ralph org escalate"

// planNotApprovedError is a refusal of the approval check: reason, an "org: "
// error of ParsePlanHeader, checkPlanApproved or checkAnyPlanApproved, then
// planNotApprovedNext. errors.Is(err, ErrPlanNotApproved) holds for it, and
// it unwraps to reason.
type planNotApprovedError struct{ reason error }

func (e planNotApprovedError) Error() string { return e.reason.Error() + planNotApprovedNext }

func (e planNotApprovedError) Is(target error) bool { return target == ErrPlanNotApproved }

func (e planNotApprovedError) Unwrap() error { return e.reason }
