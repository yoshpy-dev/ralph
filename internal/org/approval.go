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
	"time"
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
	approved               string // the `- Approved:` value, trimmed
	approvedDigest         string // approvedLineDigest of the `- Approved:` value ("" when it is not of that shape)
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
		h.approved = strings.TrimSpace(strings.TrimPrefix(lines[approved], "- Approved:"))
		h.approvedDigest = approvedLineDigest(h.approved)
	}
	return h
}

// approvedLineDigest returns the digest a feature plan's `- Approved:` value
// names when the value is exactly two whitespace-separated fields, a valid
// date YYYY-MM-DD and `sha256:<12 lowercase hex digits>`, and "" for any
// other value. PlanContentHash leaves this line out, so the check accepts
// nothing else on it: with approvedDigestFrom's reading (the first sha256:
// value anywhere on the line), words added to the line after the approval
// would change neither hash (self-review L2 of org-plan-approval).
func approvedLineDigest(value string) string {
	fields := strings.Fields(value)
	if len(fields) != 2 {
		return ""
	}
	if _, err := time.Parse(time.DateOnly, fields[0]); err != nil {
		return ""
	}
	digest, ok := strings.CutPrefix(fields[1], "sha256:")
	if !ok || !approvedDigestPattern.MatchString(digest) {
		return ""
	}
	return digest
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

// approvalRefusalKind is the kind of reason the approval check refuses with.
// The reason only says what the check found; its kind picks the one next
// step the refusal tells a leader (approvalNextStep), so the two never give
// different instructions (self-review M1 of org-plan-approval). The zero
// value is approvalRefusalOther, the kind a leader cannot fix itself.
type approvalRefusalKind int

const (
	// approvalRefusalOther: an incomplete binding, a plan path that resolves
	// outside the feature worktree, a plan file that is missing, not a
	// regular file, or unreadable. The leader escalates BLOCKED.
	approvalRefusalOther approvalRefusalKind = iota
	// approvalRefusalNoApproval: no approval recorded for the plan and
	// digest, or none at all, in the org's current life. The leader waits
	// for the answer to its QUESTION, or escalates BLOCKED when an approving
	// DECISION came without the record.
	approvalRefusalNoApproval
	// approvalRefusalPlanHeader: the approval is recorded, but the plan's
	// first `- Status:` or `- Approved:` line before its first `## ` heading
	// is missing or does not say the approval (checkApprovedPlanHeader). The
	// leader writes the two lines, commits them, and tries again.
	approvalRefusalPlanHeader
	// approvalRefusalPlanChanged: the plan changed after the approval, in
	// its digest or only in its PlanContentHash. The leader undoes the
	// change, or asks for a new approval.
	approvalRefusalPlanChanged
	// approvalRefusalTaskHeader: the TASK has no PLAN: header, two of them,
	// or a malformed one (ParsePlanHeader). The leader puts one on the TASK
	// and sends it again.
	approvalRefusalTaskHeader
)

// approvalCheckError is a reason the approval check refuses with: err says
// what the check found, kind picks the next step (approvalNextStep), and
// digest is the approved digest that an approvalRefusalPlanHeader step tells
// the leader to write.
type approvalCheckError struct {
	kind   approvalRefusalKind
	digest string
	err    error
}

func (e approvalCheckError) Error() string { return e.err.Error() }

func (e approvalCheckError) Unwrap() error { return e.err }

// approvalRefusalf is an approvalCheckError of kind whose err is
// fmt.Errorf(format, args...).
func approvalRefusalf(kind approvalRefusalKind, format string, args ...any) error {
	return approvalCheckError{kind: kind, err: fmt.Errorf(format, args...)}
}

// approvalCheckErrorOf returns the approvalCheckError err is or wraps, or one
// of kind approvalRefusalOther around err when it has none.
func approvalCheckErrorOf(err error) approvalCheckError {
	var e approvalCheckError
	if errors.As(err, &e) {
		return e
	}
	return approvalCheckError{kind: approvalRefusalOther, err: err}
}

// checkTaskPlanApproved is the approval check of text, a TASK to an
// implementer seat of orgID, bound to f: text names one plan in its header
// (ParsePlanHeader, whose errors become approvalRefusalTaskHeader reasons)
// and checkPlanApproved passes that plan.
func checkTaskPlanApproved(events []ManifestEvent, orgID string, f *FeatureBinding, text string) error {
	ref, err := ParsePlanHeader(text)
	if err != nil {
		return approvalCheckError{kind: approvalRefusalTaskHeader, err: err}
	}
	return checkPlanApproved(events, orgID, f, ref)
}

// checkPlanApproved returns nil when the plan ref (a TASK's PLAN: header,
// ParsePlanHeader) is approved for orgID, bound to f: orgID's current life
// has a plan_approved for the same path and digest (currentPlanApprovals),
// the file at that path in f's worktree (resolvePlanFile) has that digest
// now (PlanDigest), its header's first `- Status:` is Approved and its first
// `- Approved:` is `<YYYY-MM-DD> sha256:<that digest>`
// (checkApprovedPlanHeader), and its PlanContentHash equals the content one
// of those records holds. Each way to fail has its own approvalCheckError,
// which says what the check found and carries its approvalRefusalKind.
func checkPlanApproved(events []ManifestEvent, orgID string, f *FeatureBinding, ref PlanRef) error {
	if f == nil || !f.Complete() {
		return approvalRefusalf(approvalRefusalOther,
			"org: org_id %q has no complete split plan feature binding, so no plan approval can be checked", orgID)
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
		return approvalRefusalf(approvalRefusalNoApproval,
			"org: org_id %q has no approval recorded for plan %s since its last disband%s", orgID, ref, since)
	}
	return checkApprovedPlanFile(orgID, f, ref, matching)
}

// checkApprovedPlanFile is the file half of checkPlanApproved: approvals are
// the records of orgID's current life for ref's path and digest.
func checkApprovedPlanFile(orgID string, f *FeatureBinding, ref PlanRef, approvals []planApproval) error {
	abs, err := resolvePlanFile(f.Worktree, ref.Path)
	if err != nil {
		return approvalRefusalf(approvalRefusalOther, "org: org_id %q: %w", orgID, err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return approvalRefusalf(approvalRefusalOther, "org: org_id %q: plan %s: %w", orgID, ref.Path, pathErrorCause(err))
	}
	if d := PlanDigest(data); d != ref.Digest {
		return approvalRefusalf(approvalRefusalPlanChanged,
			"org: plan %s of org_id %q changed after its approval: approved digest %s, current digest %s", ref.Path, orgID, ref.Digest, d)
	}
	if err := checkApprovedPlanHeader(orgID, ref, readFeaturePlanHeader(data)); err != nil {
		return err
	}
	content := PlanContentHash(data)
	if !slices.ContainsFunc(approvals, func(a planApproval) bool { return a.Content == content }) {
		return approvalRefusalf(approvalRefusalPlanChanged,
			"org: plan %s of org_id %q changed after its approval outside its - Status: and - Approved: lines and checkbox marks, "+
				"which its digest %s does not show (the digest skips the %s section and every - Status:, - Approved: and - Branch: line)",
			ref.Path, orgID, ref.Digest, planProgressHeading)
	}
	return nil
}

// checkApprovedPlanHeader is the header half of checkApprovedPlanFile: h, the
// plan's header lines, has `- Status: Approved` and an `- Approved:` line of
// approvedLineDigest's shape that names ref.Digest. Its refusals are of kind
// approvalRefusalPlanHeader, with ref.Digest for the lines to write.
func checkApprovedPlanHeader(orgID string, ref PlanRef, h featurePlanHeader) error {
	var found string
	switch {
	case !h.hasStatus:
		found = " has no - Status: line before its first ## heading"
	case h.status != planApprovedStatus:
		found = fmt.Sprintf(" is not marked approved: its - Status: is %q, want %s", h.status, planApprovedStatus)
	case !h.hasApproved:
		found = " has no - Approved: line before its first ## heading"
	case h.approvedDigest == "":
		found = fmt.Sprintf(": its - Approved: line holds %q, which is not <YYYY-MM-DD> sha256:<12 hex digits>", h.approved)
	case h.approvedDigest != ref.Digest:
		found = fmt.Sprintf(": its - Approved: line names sha256:%s, not the approved sha256:%s", h.approvedDigest, ref.Digest)
	default:
		return nil
	}
	return approvalCheckError{kind: approvalRefusalPlanHeader, digest: ref.Digest,
		err: fmt.Errorf("org: plan %s of org_id %q%s", ref.Path, orgID, found)}
}

// checkAnyPlanApproved returns nil when at least one approval of orgID's
// current life passes checkPlanApproved with its own path and digest: the
// check of an implementer spawn, which names no plan. With no approval at all
// the error says no plan of the org was approved (approvalRefusalNoApproval);
// otherwise it gives the reason the latest approval does not pass, with that
// reason's kind and digest.
func checkAnyPlanApproved(events []ManifestEvent, orgID string, f *FeatureBinding) error {
	approvals := currentPlanApprovals(events, orgID)
	if len(approvals) == 0 {
		return approvalRefusalf(approvalRefusalNoApproval, "org: no plan of org_id %q has been approved since its last disband", orgID)
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
	latest := approvalCheckErrorOf(latestErr)
	return approvalCheckError{kind: latest.kind, digest: latest.digest,
		err: fmt.Errorf("org: none of the %d plan approvals of org_id %q since its last disband passes the check; the latest: %s",
			len(approvals), orgID, strings.TrimPrefix(latestErr.Error(), "org: "))}
}

// ErrPlanNotApproved is what every refusal of the approval check is
// (errors.Is): a TASK to an implementer seat (Send) and an implementer spawn
// (Spawn) in an org whose binding RequiresApproval.
var ErrPlanNotApproved = errors.New("org: no approved plan passes the approval check")

// planNotApprovedMarker is the phrase every refusal of the approval check
// carries, by which the leader template (prompts/leader.md, step 7 of the
// 機能ごとの org section) tells a leader to recognize it; the one next step
// follows it.
const planNotApprovedMarker = "refused by the approval check"

// planNotApprovedNever ends every refusal of the approval check: what a
// leader never does whatever the reason. A TASK's refusal adds
// planNotApprovedNeverRaw; a spawn's does not, since ralph org spawn has no
// --raw.
const (
	planNotApprovedNever    = "the leader never runs ralph org approve itself"
	planNotApprovedNeverRaw = " and never sends the TASK with --raw"
)

// approvalNextStep is the one next step a refusal of kind tells a leader.
// digest is the approved digest an approvalRefusalPlanHeader step names
// ("" writes <digest>).
func approvalNextStep(kind approvalRefusalKind, digest string) string {
	switch kind {
	case approvalRefusalNoApproval:
		return "if the leader has not received an approving DECISION for its plan yet, it waits for the answer to its QUESTION, " +
			"and if it has, that DECISION came without the record, so it escalates BLOCKED with ralph org escalate"
	case approvalRefusalPlanHeader:
		if digest == "" {
			digest = "<digest>"
		}
		return "the leader writes - Status: " + planApprovedStatus + " and - Approved: <YYYY-MM-DD> sha256:" + digest +
			" before the plan's first ## heading, commits them, and tries again"
	case approvalRefusalPlanChanged:
		return "the leader undoes the change, or sets the plan back to - Status: Draft and - Approved: TBD " +
			"and asks for a new approval with a new QUESTION (step 6 of its procedure)"
	case approvalRefusalTaskHeader:
		return "the leader puts one PLAN: <path> sha256:<digest> header naming the approved plan on the line right after TASK_ID: " +
			"and sends the TASK again"
	default:
		return "the leader cannot fix this itself, so it escalates BLOCKED with ralph org escalate"
	}
}

// planNotApprovedError is a refusal of the approval check: reason (an
// approvalCheckError of checkTaskPlanApproved or checkAnyPlanApproved), then
// planNotApprovedMarker, the one next step of the reason's kind
// (approvalNextStep), and planNotApprovedNever. task is true for the refusal
// of a TASK (Send), which also gets planNotApprovedNeverRaw, and false for
// that of a spawn (Spawn). errors.Is(err, ErrPlanNotApproved) holds for it,
// and it unwraps to reason.
type planNotApprovedError struct {
	reason error
	task   bool
}

func (e planNotApprovedError) Error() string {
	r := approvalCheckErrorOf(e.reason)
	never := planNotApprovedNever
	if e.task {
		never += planNotApprovedNeverRaw
	}
	return e.reason.Error() + "; " + planNotApprovedMarker + ": " + approvalNextStep(r.kind, r.digest) + "; " + never
}

func (e planNotApprovedError) Is(target error) bool { return target == ErrPlanNotApproved }

func (e planNotApprovedError) Unwrap() error { return e.reason }
