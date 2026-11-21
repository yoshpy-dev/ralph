package org

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// Org-wide limits and scope reservations (plan
// docs/plans/active/2026-10-08-org-limits-reserve.md). Every org_id in one org
// state dir shares one manifest, so the limits in [org].max_orgs /
// [org].max_total_seats and the reservations below are all derived from that
// manifest's events, and Spawn decides them under the same manifest lock as
// max_seats.
//
// A reservation is a list of paths relative to the repo root that an org's
// leader spawn claims for the org (SpawnParams.Reserve). It is recorded as one
// org-level EventScopeReserved and lasts until the org's next real
// `disbanded`. It is advisory: nothing stops a seat from writing outside it
// (the watchdog's scope_change alert reports such writes).
//
// An org started for one feature of an approved split plan (`ralph org start
// --plan`, plan docs/plans/active/2026-10-09-org-feature-worktree.md) records
// that feature, its FeatureBinding, in the same EventScopeReserved as the
// feature's reservation, so the binding lasts exactly as long as the
// reservation and the failed-self-close compensation restores both together.
// The binding also holds the approval mode start read ([org.approval].mode,
// plan docs/plans/active/2026-10-12-org-plan-approval.md, A8): the approval
// check reads that recorded mode, never a config.
//
// It holds the split plan's shared paths too (SplitPlan.Shared, plan
// 2026-11-22-split-shared-paths): the paths every
// feature of the plan may change, such as a lockfile. They are not part of
// the reservation's paths, so two orgs may share a path, but a shared path
// may not overlap the paths another running org reserves, in either order
// of start (reservationDecision).
//
// And it holds the number of the issue the org was started for, when
// `ralph org start --plan` was given `--issue` (plan
// 2026-11-22-org-start-issue): the leader's task names it so that the
// feature's PR closes the issue, and a reseat hands it to the leader it
// starts again.

// wholeRepoReserve is the reservation path that stands for the whole repo. It
// overlaps every other path.
const wholeRepoReserve = "."

// scopeReservedPathsKey starts an EventScopeReserved's Details: the
// normalized paths joined with commas (`paths=docs/,internal/auth/`). For a
// bound reservation the binding follows, one space-separated `key=value`
// token per field in this order: `split=<id> feature=<slug> digest=<hex>
// branch=<branch> approval=<mode> feature_digest=<hex> shared=<paths>
// issue=<number>` (the scopeReserved*Key constants below; `approval=` only
// when the binding has a mode, `feature_digest=` only when it has a feature
// digest, `shared=`, the shared paths joined with commas like the
// reservation's, only when it has shared paths, and `issue=` only when it has
// an issue); the binding's worktree goes into the event's Worktree
// field instead. Either may be followed by a space and a note (the
// failed-self-close compensation adds `restored: <why>`), which never starts
// with one of those keys. The path rules reject commas and whitespace, and
// validateFeatureBinding rejects whitespace, `=` and `,` in the binding's
// other fields, holds the shared paths to the path rules and the issue to
// decimal digits, so every token reads back unambiguously, and a reader that
// cuts the paths at the first space (as older binaries do) still reads the
// paths. `approval=`, then `feature_digest=`, `shared=` and `issue=` come last
// because an older binary stops reading the binding at the first key it does
// not know and takes the rest as the note: a binary from before the approval
// mode still reads the other four tokens, one from before the feature digest
// the other five, one from before the shared paths the other six, and one from
// before the issue the other seven, as before.
const scopeReservedPathsKey = "paths="

// The keys of a FeatureBinding's tokens in an EventScopeReserved's Details
// (see scopeReservedPathsKey), written in this order.
const (
	scopeReservedSplitKey         = "split"
	scopeReservedFeatureKey       = "feature"
	scopeReservedDigestKey        = "digest"
	scopeReservedBranchKey        = "branch"
	scopeReservedApprovalKey      = "approval"
	scopeReservedFeatureDigestKey = "feature_digest"
	scopeReservedSharedKey        = "shared"
	scopeReservedIssueKey         = "issue"
)

// maxIssueDigits is the most digits an issue number (FeatureBinding.Issue)
// may have, and issueNumberForm says the form of one for the refusals of a
// value that is not.
const (
	maxIssueDigits  = 10
	issueNumberForm = "1 to 10 decimal digits with no leading zero and no #"
)

// validIssueNumber reports whether s is an issue number in the form
// FeatureBinding.Issue holds it: 1 to maxIssueDigits ASCII decimal digits,
// the first not 0.
func validIssueNumber(s string) bool {
	if s == "" || len(s) > maxIssueDigits || s[0] == '0' {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// approvalModeUnreadable is the Approval featureBindingFromTokens gives a
// binding whose record has the `approval=` token twice or with an empty
// value. It is neither config.ApprovalModePerPlan nor
// config.ApprovalModeSplit, so RequiresApproval takes it as per-plan, and it
// holds nothing a Details token cannot, so a compensation that records the
// binding again writes it back as it is.
const approvalModeUnreadable = "unreadable"

// FeatureBinding ties an org to one feature of an approved split plan
// (split.go): `ralph org start --plan` passes it to the leader spawn
// (SpawnParams.Feature), which records it in the org's EventScopeReserved
// together with the feature's reservation.
type FeatureBinding struct {
	Split    string // split plan id (SplitPlan.ID)
	Feature  string // feature slug (SplitFeature.Slug)
	Digest   string // the split plan's approval digest at start
	Branch   string // the feature branch, <type>/<slug>
	Worktree string // absolute path of the feature worktree
	// Approval is the [org.approval].mode start read (config.ApprovalModePerPlan
	// or config.ApprovalModeSplit), recorded so the approval check reads this
	// value and not the config of whoever runs spawn or send. "" is a record
	// an older start wrote, with no mode; see RequiresApproval.
	Approval string
	// FeatureDigest is the feature's own digest at start (SplitFeature.Digest:
	// the split plan's preamble, its shared paths when it has a `- Shared:`
	// line, and the feature's section), so that
	// sameFeature lets a start pass after only other features of the split
	// plan changed. "" is a record an older start wrote, or one whose
	// `feature_digest=` token is given twice or empty; sameFeature then
	// compares Digest, the whole plan's digest, as before.
	FeatureDigest string
	// Shared is the split plan's shared paths at start (SplitPlan.Shared):
	// NormalizeReservePaths output, `.` never among them, joined with commas
	// as the `shared=` token holds them, so that a binding stays comparable
	// with ==. "" is a plan with no `- Shared:` line, a record an older start
	// wrote, or one whose shared paths cannot be read
	// (featureBindingFromTokens). SharedPaths returns them as a list.
	Shared string
	// Issue is the number of the issue the org was started for
	// (`ralph org start --plan --issue`, StartFeatureParams.Issue), in
	// decimal digits with no leading zero and no `#` (validIssueNumber), so
	// that the leader's task can tell the leader to close it from the
	// feature's PR. "" is a start with no --issue, a record an older start
	// wrote, or one whose `issue=` token cannot be read
	// (featureBindingFromTokens).
	Issue string
}

// SharedPaths returns f's shared paths (Shared) as a list, nil when f is nil
// or has none, so that it can be called on the Feature of any Reservation.
func (f *FeatureBinding) SharedPaths() []string {
	if f == nil || f.Shared == "" {
		return nil
	}
	return strings.Split(f.Shared, ",")
}

// IssueNumber returns f's issue (Issue) as a number, 0 when f is nil or has
// none, as the JSON of `ralph org status` (its feature's `issue`) and of
// `ralph org merge-order` (MergeOrderBinding.Issue) gives it, so that the two
// read it the same way. A binding read from the ledger holds "" or an issue
// number of at most maxIssueDigits digits (featureBindingFromTokens leaves
// anything else ""), which an int64 holds, so the parse fails only for a
// value no record gives, and that also reads as none.
func (f *FeatureBinding) IssueNumber() int64 {
	if f == nil || f.Issue == "" {
		return 0
	}
	n, err := strconv.ParseInt(f.Issue, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// Reservation is what one EventScopeReserved records: the normalized paths
// and, for an org started for a split plan feature, its binding (all its
// fields: the five that name the feature, the approval mode, the feature
// digest, the split plan's shared paths and the issue). Feature is nil for a
// plain --reserve reservation, which has no shared paths. The shared paths
// are not part of Paths: Paths are what the org holds against every other
// running org, while a shared path is one every org of the split plan may
// change, and Feature.SharedPaths returns them.
type Reservation struct {
	Paths   []string
	Feature *FeatureBinding
}

// Complete reports whether the five fields that name the feature (split,
// feature, digest, branch and worktree) are set, as Spawn requires of a
// binding it records (validateFeatureBinding). Approval and FeatureDigest are
// not among them: a record an older start wrote has no mode or no feature
// digest and is still complete, so its org is still a feature org to the
// gate, its branch still a merge-order target, and it still reads as bound
// in `ralph org status`. A binding read back from a damaged record
// (featureBindingFromTokens) may not be complete; ActiveFeature returns it
// as read, and `ralph org status` uses this to say that the record is
// incomplete.
func (f *FeatureBinding) Complete() bool {
	return f.Split != "" && f.Feature != "" && f.Digest != "" && f.Branch != "" && f.Worktree != ""
}

// RequiresApproval reports whether the org bound to f needs a plan a human
// approved before its implementer starts, by the mode recorded at start (A8
// of plan 2026-10-12-org-plan-approval): false for split, and for a record
// with no mode, which an older start wrote and whose leader prompt has no
// approval step, so it is read as split; true for per-plan, and for any
// other value (the token given twice or empty, approvalModeUnreadable, or an
// unknown mode), so a mode that cannot be read asks for the approval.
func (f *FeatureBinding) RequiresApproval() bool {
	return f.Approval != "" && f.Approval != config.ApprovalModeSplit
}

// String names f for error text: the feature and split plan, then the
// digest, the feature digest, branch, worktree and approval mode, the shared
// paths when it has any, the issue when it has one, and a note when f was
// read from a damaged record.
func (f *FeatureBinding) String() string {
	approval := fmt.Sprintf("approval %q", f.Approval)
	if f.Approval == "" {
		approval = "no approval mode, read as split"
	}
	featureDigest := fmt.Sprintf("feature digest %q", f.FeatureDigest)
	if f.FeatureDigest == "" {
		featureDigest = "no feature digest"
	}
	shared := ""
	if f.Shared != "" {
		shared = fmt.Sprintf(", shared paths %q", f.Shared)
	}
	issue := ""
	if f.Issue != "" {
		issue = ", " + describeIssue(f.Issue)
	}
	s := fmt.Sprintf("feature %q of split plan %q (digest %q, %s, branch %q, worktree %q, %s%s%s)",
		f.Feature, f.Split, f.Digest, featureDigest, f.Branch, f.Worktree, approval, shared, issue)
	if !f.Complete() {
		s += " read from an incomplete record"
	}
	return s
}

// sameFeature reports whether two bindings are the same: both nil, or both
// complete with the same split plan, feature, branch, worktree, approval
// mode and issue (so the same feature started again with another mode, or
// for another issue or with no issue when it had one, or the other way
// round, is another binding; the org is disbanded and started again to
// change it) and the same digest. Two bindings with no issue have the same
// issue. When both have a feature digest, the digest compared is that
// one and not the whole plan's (Digest), so a start after only other
// features of the split plan changed, and the plan was approved again, is
// the same binding (plan 2026-11-12-split-feature-digest, D5); when either
// has none (a record an older start wrote, or a damaged token), it is the
// whole plan's digest, as before. That fallback is never looser than the
// feature digest: a split plan refuses every line the whole digest skips,
// but for one `- Status:` and one `- Approved:` line in its header
// (rejectDigestSkippedLine and parseSplitPlan), and the only header line
// that goes into a feature digest, the `- Shared:` line, goes into the whole
// digest too, so two copies of a plan with the same whole digest have the
// same preamble, sections and shared paths, and so the same feature
// digests. The shared paths are not compared on their own: either digest
// covers them (a feature digest holds the plan's shared paths when it has
// any, and the whole plan's digest holds its `- Shared:` line), so a plan
// whose shared paths changed is another binding, and a start with the same
// plan matches the record. An incomplete binding (a damaged record) equals
// nothing, not even itself, so a damaged record never passes as the org's
// binding. Every field but the digests and the issue is compared by
// sameFeatureExceptDigestsAndIssue, which featureDigestRefusal and
// featureIssueRefusal share, so a field added to the binding is compared in
// one place.
func sameFeature(a, b *FeatureBinding) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if !b.Complete() || !sameFeatureExceptDigestsAndIssue(a, b) || a.Issue != b.Issue {
		return false
	}
	if a.FeatureDigest != "" && b.FeatureDigest != "" {
		return a.FeatureDigest == b.FeatureDigest
	}
	return a.Digest == b.Digest
}

// sameFeatureExceptDigestsAndIssue reports whether want names the feature
// held records in every field but the digests (Digest and FeatureDigest) and
// the issue: held is complete, and both have the same split plan, feature,
// branch, worktree and approval mode. It is false when either is nil.
// sameFeature also needs want complete and the issue and a digest the same;
// featureDigestRefusal and featureIssueRefusal explain, for such a want, a
// refusal that one of those differences caused.
func sameFeatureExceptDigestsAndIssue(held, want *FeatureBinding) bool {
	return held != nil && want != nil && held.Complete() && held.Split == want.Split && held.Feature == want.Feature &&
		held.Branch == want.Branch && held.Worktree == want.Worktree && held.Approval == want.Approval
}

// validateFeatureBinding checks a binding Spawn is asked to record: split,
// feature, digest and branch must each be non-empty with no whitespace, `=`,
// `,` or control character (the Details tokens cannot hold them), worktree
// must be an absolute path, approval must be config.ApprovalModePerPlan or
// config.ApprovalModeSplit (start normalizes an empty config to per-plan),
// the feature digest must be 12 lowercase hex digits, the shape of
// SplitFeature.Digest (start always sets it; a binding without one would
// quietly compare by the whole plan's digest), and the shared paths, when
// there are any, must be in the form SplitPlan.Shared gives them:
// NormalizeReservePaths output with no `.`, joined with commas, and the
// issue, when there is one, must be an issue number (validIssueNumber).
func validateFeatureBinding(f FeatureBinding) error {
	for _, field := range []struct{ key, value string }{
		{scopeReservedSplitKey, f.Split},
		{scopeReservedFeatureKey, f.Feature},
		{scopeReservedDigestKey, f.Digest},
		{scopeReservedBranchKey, f.Branch},
	} {
		switch {
		case field.value == "":
			return fmt.Errorf("org: the split plan feature's %s is empty", field.key)
		case strings.ContainsFunc(field.value, func(r rune) bool {
			return r == '=' || r == ',' || unicode.IsSpace(r) || unicode.IsControl(r)
		}):
			return fmt.Errorf("org: the split plan feature's %s %q has whitespace, a comma, an = or a control character, which a reservation cannot hold",
				field.key, field.value)
		}
	}
	if !filepath.IsAbs(f.Worktree) {
		return fmt.Errorf("org: the split plan feature's worktree %q is not an absolute path", f.Worktree)
	}
	if f.Approval != config.ApprovalModePerPlan && f.Approval != config.ApprovalModeSplit {
		return fmt.Errorf("org: the split plan feature's approval mode %q is not %q or %q",
			f.Approval, config.ApprovalModePerPlan, config.ApprovalModeSplit)
	}
	if !approvedDigestPattern.MatchString(f.FeatureDigest) {
		return fmt.Errorf("org: the split plan feature's %s %q is not 12 lowercase hex digits",
			scopeReservedFeatureDigestKey, f.FeatureDigest)
	}
	if f.Shared != "" && sharedFromToken(f.Shared) != f.Shared {
		return fmt.Errorf("org: the split plan's shared paths %q are not normalized reservation paths joined with commas, "+
			"with no %q (the whole repo) among them", f.Shared, wholeRepoReserve)
	}
	if f.Issue != "" && !validIssueNumber(f.Issue) {
		return fmt.Errorf("org: the split plan feature's %s %q is not an issue number (%s)", scopeReservedIssueKey, f.Issue, issueNumberForm)
	}
	return nil
}

// NormalizeReservePaths checks and normalizes the paths of a reservation so
// that the same set always compares equal: each path is relative to the repo
// root; one ending in `/` (or `/.`) is a directory and covers everything
// under it, any other is a file, and `.` (or `./`) is the whole repo.
// Redundant `./` segments and repeated slashes are removed, and the result is
// sorted with duplicates dropped. An empty path, an absolute path, a path with
// a `..` segment, and a path with a comma, whitespace or a control character
// (which the manifest record cannot hold) are errors. An empty list returns
// nil: no reservation.
func NormalizeReservePaths(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		p, err := normalizeReservePath(r)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}

func normalizeReservePath(raw string) (string, error) {
	switch {
	case raw == "":
		return "", errors.New(`org: reserve path is empty: write a path relative to the repo root ("." for the whole repo)`)
	case strings.HasPrefix(raw, "/"):
		return "", fmt.Errorf("org: reserve path %q is absolute: write it relative to the repo root", raw)
	case strings.ContainsFunc(raw, func(r rune) bool { return r == ',' || unicode.IsSpace(r) || unicode.IsControl(r) }):
		return "", fmt.Errorf("org: reserve path %q has a comma, whitespace or a control character, which a reservation cannot hold", raw)
	case slices.Contains(strings.Split(raw, "/"), ".."):
		return "", fmt.Errorf("org: reserve path %q has a .. segment: write it relative to the repo root, inside it", raw)
	}
	dir := raw == "." || strings.HasSuffix(raw, "/") || strings.HasSuffix(raw, "/.")
	p := path.Clean(raw)
	switch {
	case p == ".":
		return wholeRepoReserve, nil
	case dir:
		return p + "/", nil
	default:
		return p, nil
	}
}

// reservePathsOverlap reports whether two normalized reservation paths cover
// a common path: the whole repo overlaps everything, two directories overlap
// when one contains the other, a directory and a file when the file is under
// the directory, and two files when they are the same. Directories end with
// `/`, so the prefix test compares whole segments (`internal/auth/` does not
// contain `internal/authz/`).
func reservePathsOverlap(a, b string) bool {
	aDir, bDir := strings.HasSuffix(a, "/"), strings.HasSuffix(b, "/")
	switch {
	case a == wholeRepoReserve || b == wholeRepoReserve:
		return true
	case aDir && bDir:
		return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
	case aDir:
		return strings.HasPrefix(b, a)
	case bDir:
		return strings.HasPrefix(a, b)
	default:
		return a == b
	}
}

// reserveOverlaps returns `<want path> overlaps <held path>` for every pair of
// overlapping paths between want and held, in want then held order.
func reserveOverlaps(want, held []string) []string {
	var pairs []string
	for _, w := range want {
		for _, h := range held {
			if reservePathsOverlap(w, h) {
				pairs = append(pairs, w+" overlaps "+h)
			}
		}
	}
	return pairs
}

// scopeReservedDetails is the Details of an EventScopeReserved for r (its
// paths already normalized): the paths, then r's binding tokens when it has
// one (`approval=` only when the binding has a mode, `feature_digest=` only
// when it has a feature digest, `shared=` only when it has shared paths, and
// `issue=` last, only when it has an issue), then note after a space when
// non-empty (see scopeReservedPathsKey). The binding's worktree is not part
// of the Details (scopeReservedEvent).
func scopeReservedDetails(r Reservation, note string) string {
	details := scopeReservedPathsKey + strings.Join(r.Paths, ",")
	if f := r.Feature; f != nil {
		details += " " + scopeReservedSplitKey + "=" + f.Split +
			" " + scopeReservedFeatureKey + "=" + f.Feature +
			" " + scopeReservedDigestKey + "=" + f.Digest +
			" " + scopeReservedBranchKey + "=" + f.Branch
		if f.Approval != "" {
			details += " " + scopeReservedApprovalKey + "=" + f.Approval
		}
		if f.FeatureDigest != "" {
			details += " " + scopeReservedFeatureDigestKey + "=" + f.FeatureDigest
		}
		if f.Shared != "" {
			details += " " + scopeReservedSharedKey + "=" + f.Shared
		}
		if f.Issue != "" {
			details += " " + scopeReservedIssueKey + "=" + f.Issue
		}
	}
	if note != "" {
		details += " " + note
	}
	return details
}

// reservedPathsFromDetails reads the paths back from an EventScopeReserved's
// Details (see scopeReservedDetails), normalized again. It cuts the paths at
// the first space, so the binding tokens and the note that may follow are
// not part of them. ok is false when the Details hold no readable path list.
func reservedPathsFromDetails(details string) (paths []string, ok bool) {
	rest, found := strings.CutPrefix(details, scopeReservedPathsKey)
	if !found {
		return nil, false
	}
	list, _, _ := strings.Cut(rest, " ")
	if list == "" {
		return nil, false
	}
	paths, err := NormalizeReservePaths(strings.Split(list, ","))
	if err != nil {
		return nil, false
	}
	return paths, true
}

// reservationFromEvent reads the Reservation an EventScopeReserved records.
// Paths that cannot be read are taken as the whole repo, so a damaged record
// blocks overlapping reservations instead of hiding them until the org is
// disbanded. The binding comes from the tokens after the paths and from the
// event's Worktree (featureBindingFromTokens): a record an older binary
// wrote, with neither, has none.
func reservationFromEvent(ev ManifestEvent) Reservation {
	paths, ok := reservedPathsFromDetails(ev.Details)
	if !ok {
		paths = []string{wholeRepoReserve}
	}
	var tokens []string
	if fields := strings.Fields(ev.Details); len(fields) > 1 {
		tokens = fields[1:]
	}
	return Reservation{Paths: paths, Feature: featureBindingFromTokens(tokens, ev.Worktree)}
}

// featureBindingFromTokens reads a binding from the Details tokens that
// follow the paths, up to the first token that is not `key=value` with one
// of the scopeReserved*Key keys (the note, when there is one), and from
// worktree, the event's Worktree. It returns nil when there is no such token
// and no worktree: the reservation is not bound. A record with some of them
// but not all, an empty value, or a key given twice (that field is then left
// empty) must not read as unbound, since that would let a spawn with a plain
// --reserve, or one for another feature of a running org, pass. It is
// returned with whatever it holds instead, and is incomplete, so sameFeature
// takes it as equal to nothing and reservationDecision refuses every other
// reservation for the org until it is disbanded. The approval mode is read
// the same way but does not make the binding incomplete: no `approval=`
// token leaves Approval "" (a record an older start wrote), an unknown value
// is kept as read, and the token given twice or with an empty value gives
// approvalModeUnreadable, so RequiresApproval asks for the approval in every
// case but the first. The feature digest does not make the binding
// incomplete either (a record an older start wrote has none, and its org
// must still read as bound): no `feature_digest=` token, the token given
// twice, and an empty value all leave FeatureDigest "", with no sentinel,
// so sameFeature compares the whole plan's digest instead, which is never
// looser. The shared paths do not make the binding incomplete either, and a
// record whose shared paths cannot be read has none: no `shared=` token (a
// record an older start wrote, or a plan with no `- Shared:` line), the
// token given twice, an empty value, and a value whose paths the
// reservation rules refuse or that holds `.` all leave Shared "" (the paths
// that can be read are normalized again, as reservedPathsFromDetails does).
// Taking an unreadable value as no shared paths keeps what the org may
// change without an alert to its reserved paths, so the watchdog's
// scope_change still reports a change to the paths it meant to share; taken
// as the whole repo, as unreadable reserved paths are, it would hide every
// change of the org from the watchdog. The cost is that the admission check
// (reservationDecision) does not hold another org's reservation off those
// paths. Nor does the binding keep a trace of the damage: Shared is "" as
// for a record with no `shared=` token, so the two cannot be told apart, and
// the --scope reseat gives the seats other than the leader, built from the
// record (FeatureBinding.SharedPaths), then lacks the shared paths that the
// leader's task, built from the split plan (featureLeaderTask), still names.
// The issue is read the same way as the shared paths: it does not make the
// binding incomplete, and no `issue=` token (a start with no --issue, or a
// record an older start wrote), the token given twice, and a value that is
// not an issue number (validIssueNumber: empty, not decimal digits, a
// leading zero, a `#`, more than maxIssueDigits digits) all leave Issue "".
// A reseat of such an org then gives its leader no issue line, and the
// feature's PR closes no issue.
func featureBindingFromTokens(tokens []string, worktree string) *FeatureBinding {
	f := &FeatureBinding{Worktree: worktree}
	fields := map[string]*string{
		scopeReservedSplitKey:         &f.Split,
		scopeReservedFeatureKey:       &f.Feature,
		scopeReservedDigestKey:        &f.Digest,
		scopeReservedBranchKey:        &f.Branch,
		scopeReservedApprovalKey:      &f.Approval,
		scopeReservedFeatureDigestKey: &f.FeatureDigest,
		scopeReservedSharedKey:        &f.Shared,
		scopeReservedIssueKey:         &f.Issue,
	}
	read := make(map[string]int)
	for _, token := range tokens {
		key, value, ok := strings.Cut(token, "=")
		field, known := fields[key]
		if !ok || !known {
			break
		}
		read[key]++
		*field = value
	}
	if len(read) == 0 && worktree == "" {
		return nil
	}
	for key, n := range read {
		if n > 1 {
			*fields[key] = ""
		}
	}
	if read[scopeReservedApprovalKey] > 0 && f.Approval == "" {
		f.Approval = approvalModeUnreadable
	}
	f.Shared = sharedFromToken(f.Shared)
	if !validIssueNumber(f.Issue) {
		f.Issue = ""
	}
	return f
}

// sharedFromToken returns the shared paths of a `shared=` token's value
// (paths joined with commas) in the form FeatureBinding.Shared holds them:
// normalized by NormalizeReservePaths and joined with commas again, or ""
// when value is empty, when a path is one the reservation rules refuse, and
// when `.` (the whole repo) is among them (see featureBindingFromTokens).
func sharedFromToken(value string) string {
	if value == "" {
		return ""
	}
	paths, err := NormalizeReservePaths(strings.Split(value, ","))
	if err != nil || slices.Contains(paths, wholeRepoReserve) {
		return ""
	}
	return strings.Join(paths, ",")
}

// scopeReservedEvent is the org-level EventScopeReserved recording r as
// orgID's reservation, with note at the end of its Details
// (scopeReservedDetails). A bound reservation's worktree goes into the
// event's Worktree field, which org-level events do not use otherwise.
func scopeReservedEvent(ts, orgID string, r Reservation, note string, dryRun bool) ManifestEvent {
	ev := ManifestEvent{
		TS: ts, OrgID: orgID, SeatID: "", Event: EventScopeReserved,
		DryRun: dryRun, Details: scopeReservedDetails(r, note),
	}
	if r.Feature != nil {
		ev.Worktree = r.Feature.Worktree
	}
	return ev
}

// orgLife is what an org's real events after its latest real `disbanded`
// (all of them when it has none) still hold open: the workspaces created in
// that span and not closed since, oldest first, and the latest reservation
// recorded in that span with its binding (reservation.Paths nil: none).
type orgLife struct {
	workspaces  []string
	reservation Reservation
}

// lastDisbandedIndexes maps each org_id to the index in events of its latest
// real (non-dry-run) org-level `disbanded`. An org that has none is absent.
func lastDisbandedIndexes(events []ManifestEvent) map[string]int {
	last := make(map[string]int)
	for i, ev := range events {
		if !ev.DryRun && ev.OrgID != "" && ev.SeatID == "" && ev.Event == EventDisbanded {
			last[ev.OrgID] = i
		}
	}
	return last
}

// currentOrgLives returns the orgLife of every org_id that has a real
// org-level workspace or reservation event after its latest real
// `disbanded`. Only events after that `disbanded` count, so a workspace an
// older ralph's disband left open (created before it) is not part of the
// org's current life, while one the failed-self-close compensation reopened
// after it is. Each reservation is read with reservationFromEvent: paths
// that cannot be read are taken as the whole repo, and the binding, when the
// record has one, comes with the paths, so the org's binding is always that
// of its latest reservation.
func currentOrgLives(events []ManifestEvent) map[string]*orgLife {
	lastDisbanded := lastDisbandedIndexes(events)
	lives := make(map[string]*orgLife)
	life := func(orgID string) *orgLife {
		l, ok := lives[orgID]
		if !ok {
			l = &orgLife{}
			lives[orgID] = l
		}
		return l
	}
	for i, ev := range events {
		if ev.DryRun || ev.OrgID == "" || ev.SeatID != "" {
			continue
		}
		if d, ok := lastDisbanded[ev.OrgID]; ok && i <= d {
			continue
		}
		switch ev.Event {
		case EventOrgWorkspaceCreated, EventOrgWorkspaceClosed:
			if ev.PaneID == "" {
				continue
			}
			l := life(ev.OrgID)
			l.workspaces = slices.DeleteFunc(l.workspaces, func(id string) bool { return id == ev.PaneID })
			if ev.Event == EventOrgWorkspaceCreated {
				l.workspaces = append(l.workspaces, ev.PaneID)
			}
		case EventScopeReserved:
			life(ev.OrgID).reservation = reservationFromEvent(ev)
		}
	}
	return lives
}

// RunningOrgs returns, sorted, the org_ids that count as running for
// [org].max_orgs. Only real (non-dry-run) events after an org's latest real
// `disbanded` count, and an org runs when, in that span, it has an active
// seat (Roster), a workspace created and not closed, or a reservation. An org
// with only `rejected` records, one whose seats all stopped with no workspace
// open and no reservation, and one an older ralph disbanded without closing
// its workspace do not run.
func RunningOrgs(events []ManifestEvent) []string {
	running := make(map[string]bool)
	for _, s := range Roster(events, RosterOptions{}) {
		if s.Active {
			running[s.OrgID] = true
		}
	}
	for orgID, l := range currentOrgLives(events) {
		if len(l.workspaces) > 0 || l.reservation.Paths != nil {
			running[orgID] = true
		}
	}
	orgs := make([]string, 0, len(running))
	for orgID := range running {
		orgs = append(orgs, orgID)
	}
	slices.Sort(orgs)
	return orgs
}

// ActiveReservation returns orgID's reservation: the paths of its latest real
// EventScopeReserved after its latest real `disbanded`, normalized. nil means
// the org holds none. ActiveFeature returns the binding of the same record.
func ActiveReservation(events []ManifestEvent, orgID string) []string {
	return activeReservation(events, orgID).Paths
}

// ActiveFeature returns the split plan feature orgID was started for: the
// binding of the same record ActiveReservation reads. nil means the org holds
// no reservation or a plain one. A binding read from a damaged record is
// returned as read, with some fields empty.
func ActiveFeature(events []ManifestEvent, orgID string) *FeatureBinding {
	return activeReservation(events, orgID).Feature
}

// activeReservation returns orgID's current reservation with its binding
// (see ActiveReservation and ActiveFeature); the zero Reservation when it
// holds none.
func activeReservation(events []ManifestEvent, orgID string) Reservation {
	if l, ok := currentOrgLives(events)[orgID]; ok {
		return l.reservation
	}
	return Reservation{}
}

// TotalActiveSeats returns the number of active real seats across every
// org_id, the count [org].max_total_seats limits.
func TotalActiveSeats(events []ManifestEvent) int {
	n := 0
	for _, s := range Roster(events, RosterOptions{}) {
		if s.Active {
			n++
		}
	}
	return n
}

// reservationDecision decides, from events read under the manifest lock,
// whether orgID may hold the reservation want (paths normalized, binding
// checked by validateFeatureBinding when there is one). When orgID already
// holds one, the same paths with the same binding (sameFeature: both none,
// or the same split plan, feature, branch, worktree, approval mode and
// issue, and the same feature digest, or the same whole plan's digest when
// either has no feature digest) pass with nothing to record, so the record
// keeps the first start's whole digest, and anything else is refused: an
// org bound to a split plan feature refuses a reservation without a binding
// or with another one (another feature, the same one whose section or the
// plan's preamble changed, the same one of an older record whose plan was
// approved again with another digest, or started with another approval
// mode or another issue), even for the same paths, and says which digest
// changed when the digests differ and which issue each side has when the
// issues differ (featureRestartRefusal); an org with a plain reservation
// refuses one with a binding; and a plain reservation with other paths is
// refused as before. When orgID holds none, a want with a binding is refused
// if the org is running (RunningOrgs: an active seat or an open workspace, as
// for a promoted session's leader or an org from `ralph org start <task>`),
// since that org was not started for the feature. Otherwise want is checked
// against the reservation of every other org that holds one (an org with a
// reservation is running, see RunningOrgs), in org_id order, and the first
// such org refuses it when want's paths overlap its paths, when want's
// shared paths (Feature.SharedPaths) overlap its paths, or when want's paths
// overlap its shared paths; the error names that org and the overlapping
// pairs, and says which side shares the path. Shared paths that overlap
// another org's shared paths are allowed: every org of a split plan may
// change them, and so may those of another split plan that shares them. The
// two checks with shared paths keep an org from changing a path another org
// holds as a shared path, a change the watchdog does not report, whichever
// of the two started first. record is true when the caller must append the
// EventScopeReserved.
func reservationDecision(events []ManifestEvent, orgID string, want Reservation) (record bool, err error) {
	lives := currentOrgLives(events)
	if own, ok := lives[orgID]; ok && own.reservation.Paths != nil {
		held := own.reservation
		heldPaths, wantPaths := strings.Join(held.Paths, ","), strings.Join(want.Paths, ",")
		switch {
		case slices.Equal(held.Paths, want.Paths) && sameFeature(held.Feature, want.Feature):
			return false, nil
		case held.Feature != nil:
			return false, fmt.Errorf("org: org_id %q is bound to %s and reserves %s, so %s is refused%s: "+
				"ralph org disband --org-id %s first, or use another org_id",
				orgID, held.Feature, heldPaths, describeWantedReservation(want), featureRestartRefusal(orgID, held.Feature, want.Feature), orgID)
		case want.Feature != nil:
			return false, fmt.Errorf("org: org_id %q was started without a split plan (it reserves %s with a plain --reserve), "+
				"so %s is refused: use another org_id, or ralph org disband --org-id %s first",
				orgID, heldPaths, describeWantedReservation(want), orgID)
		default:
			return false, fmt.Errorf("org: org_id %q already reserves %s, so a different reservation (%s) is refused: "+
				"a reservation stays until ralph org disband --org-id %s",
				orgID, heldPaths, wantPaths, orgID)
		}
	}
	if want.Feature != nil && slices.Contains(RunningOrgs(events), orgID) {
		return false, fmt.Errorf("org: org_id %q is running without a split plan (it has an active seat or an open workspace, "+
			"as a promoted session's leader or ralph org start <task> leaves it), so %s is refused: "+
			"use another org_id, or ralph org disband --org-id %s first",
			orgID, describeWantedReservation(want), orgID)
	}
	others := make([]string, 0, len(lives))
	for other, l := range lives {
		if other != orgID && l.reservation.Paths != nil {
			others = append(others, other)
		}
	}
	slices.Sort(others)
	for _, other := range others {
		if err := reservationOverlapErr(other, want, lives[other].reservation); err != nil {
			return false, err
		}
	}
	return true, nil
}

// reservationOverlapErr is reservationDecision's refusal of want for the
// reservation held of the running org other, or nil when none of these
// overlap, checked in this order: want's paths and held's paths, want's
// shared paths and held's paths, and want's paths and held's shared paths.
// Shared paths are never checked against shared paths. The error names
// other, every overlapping pair of the first check that has one, and which
// side shares the path.
func reservationOverlapErr(other string, want, held Reservation) error {
	if pairs := reserveOverlaps(want.Paths, held.Paths); len(pairs) > 0 {
		return fmt.Errorf("org: reservation overlaps the reservation of running org_id %q: %s",
			other, strings.Join(pairs, ", "))
	}
	if pairs := reserveOverlaps(want.Feature.SharedPaths(), held.Paths); len(pairs) > 0 {
		return fmt.Errorf("org: the shared paths of split plan %q overlap the reservation of running org_id %q "+
			"(each pair names the shared path first, then the reserved one): %s: "+
			"a path one org reserves cannot be shared by another; take it out of the split plan's %s line, "+
			"or wait until org_id %q is disbanded",
			want.Feature.Split, other, strings.Join(pairs, ", "), splitSharedPrefix, other)
	}
	if pairs := reserveOverlaps(want.Paths, held.Feature.SharedPaths()); len(pairs) > 0 {
		return fmt.Errorf("org: reservation overlaps the shared paths of running org_id %q "+
			"(each pair names the reserved path first, then the path split plan %q shares, which every org of it may change): %s: "+
			"a path another org shares cannot be reserved; reserve other paths, or wait until org_id %q is disbanded",
			other, held.Feature.Split, strings.Join(pairs, ", "), other)
	}
	return nil
}

// featureDigestRefusal is the clause reservationDecision adds to its refusal
// of want when orgID is bound to held and want names the same feature
// (sameFeatureExceptDigestsAndIssue: the same split plan, feature, branch,
// worktree and approval mode, held complete) with another digest: with a
// feature digest on both sides, that the split plan's shared paths changed
// after the org was started when they differ, with both lists, and otherwise
// that the feature's section or the split plan's preamble changed, with both
// feature digests in either case; with none in held's record, that such a
// record refuses every change to the split plan and that disbanding the org
// and starting it again records one. It is "" for every other refusal, which
// the bindings in the error already explain.
func featureDigestRefusal(orgID string, held, want *FeatureBinding) string {
	if !sameFeatureExceptDigestsAndIssue(held, want) {
		return ""
	}
	switch {
	case held.FeatureDigest != "" && want.FeatureDigest != "" && held.FeatureDigest != want.FeatureDigest && held.Shared != want.Shared:
		return fmt.Sprintf(", since the shared paths of split plan %q (its %s line) changed after org_id %q was started "+
			"(%s, now %s; feature digest %q, now %q)", held.Split, splitSharedPrefix, orgID,
			describeSharedPaths(held.Shared), describeSharedPaths(want.Shared), held.FeatureDigest, want.FeatureDigest)
	case held.FeatureDigest != "" && want.FeatureDigest != "" && held.FeatureDigest != want.FeatureDigest:
		return fmt.Sprintf(", since the feature's section (### %s) or the preamble of split plan %q changed after org_id %q was started "+
			"(feature digest %q, now %q)", held.Feature, held.Split, orgID, held.FeatureDigest, want.FeatureDigest)
	case held.FeatureDigest == "" && held.Digest != want.Digest:
		return ", and since its record has no feature digest (an older ralph started it), every change to the split plan is refused; " +
			"disbanding the org and starting it again records one"
	}
	return ""
}

// featureRestartRefusal is the clause reservationDecision adds to its refusal
// of want when orgID is bound to held: featureDigestRefusal's, then, joined
// to it with ", and", featureIssueRefusal's. It is "" when both are.
func featureRestartRefusal(orgID string, held, want *FeatureBinding) string {
	clause := featureDigestRefusal(orgID, held, want)
	issue := featureIssueRefusal(orgID, held, want)
	switch {
	case issue == "":
		return clause
	case clause == "":
		return ", " + issue
	default:
		return clause + ", and " + issue
	}
}

// featureIssueRefusal is the part of reservationDecision's refusal of want
// that names the issues, when orgID is bound to held and want names the same
// feature (sameFeatureExceptDigestsAndIssue: the same split plan, feature,
// branch, worktree and approval mode, held complete) for another issue, or
// with no issue when held has one, or the other way round: the issue the org
// was started with, the one want has, and the --issue that starts the org
// again as it was. It is "" when the issues are the same, and for every
// other refusal, which the bindings in the error already explain.
func featureIssueRefusal(orgID string, held, want *FeatureBinding) string {
	if !sameFeatureExceptDigestsAndIssue(held, want) || held.Issue == want.Issue {
		return ""
	}
	again := "with no --issue"
	if held.Issue != "" {
		again = "with --issue " + held.Issue
	}
	return fmt.Sprintf("since org_id %q was started with %s and this start has %s (a start of the org that is not disbanded "+
		"names the issue it was started with: run it %s)", orgID, describeIssue(held.Issue), describeIssue(want.Issue), again)
}

// describeIssue names a binding's issue (FeatureBinding.Issue) for String and
// featureIssueRefusal: `issue #<number>`, or "no issue".
func describeIssue(issue string) string {
	if issue == "" {
		return "no issue"
	}
	return "issue #" + issue
}

// describeSharedPaths names a binding's shared paths (FeatureBinding.Shared)
// for featureDigestRefusal: the list quoted, or "none".
func describeSharedPaths(shared string) string {
	if shared == "" {
		return "none"
	}
	return strconv.Quote(shared)
}

// describeWantedReservation names want for reservationDecision's refusals:
// its paths, and its binding or that it has none.
func describeWantedReservation(want Reservation) string {
	paths := strings.Join(want.Paths, ",")
	if want.Feature == nil {
		return "a reservation without a split plan (" + paths + ")"
	}
	return "the reservation " + paths + " for " + want.Feature.String()
}

// reservationBeforeLastDisband returns the reservation orgID held right before
// its latest real `disbanded` (nil when it held none, or has no `disbanded`).
func reservationBeforeLastDisband(events []ManifestEvent, orgID string) []string {
	d, ok := lastDisbandedIndexes(events)[orgID]
	if !ok {
		return nil
	}
	return ActiveReservation(events[:d], orgID)
}
