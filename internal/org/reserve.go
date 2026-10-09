package org

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
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

// wholeRepoReserve is the reservation path that stands for the whole repo. It
// overlaps every other path.
const wholeRepoReserve = "."

// scopeReservedPathsKey starts an EventScopeReserved's Details: the
// normalized paths joined with commas (`paths=docs/,internal/auth/`). For a
// bound reservation the binding follows, one space-separated `key=value`
// token per field in this order: `split=<id> feature=<slug> digest=<hex>
// branch=<branch>` (the scopeReserved*Key constants below); the binding's
// worktree goes into the event's Worktree field instead. Either may be
// followed by a space and a note (the failed-self-close compensation adds
// `restored: <why>`), which never starts with one of those keys. The path
// rules reject commas and whitespace, and validateFeatureBinding rejects
// whitespace, `=` and `,` in the binding's fields, so every token reads back
// unambiguously, and a reader that cuts the paths at the first space (as
// older binaries do) still reads the paths.
const scopeReservedPathsKey = "paths="

// The keys of a FeatureBinding's tokens in an EventScopeReserved's Details
// (see scopeReservedPathsKey), written in this order.
const (
	scopeReservedSplitKey   = "split"
	scopeReservedFeatureKey = "feature"
	scopeReservedDigestKey  = "digest"
	scopeReservedBranchKey  = "branch"
)

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
}

// Reservation is what one EventScopeReserved records: the normalized paths
// and, for an org started for a split plan feature, its binding. Feature is
// nil for a plain --reserve reservation.
type Reservation struct {
	Paths   []string
	Feature *FeatureBinding
}

// Complete reports whether every field of f is set, as Spawn requires of a
// binding it records (validateFeatureBinding). A binding read back from a
// damaged record (featureBindingFromTokens) may not be; ActiveFeature
// returns it as read, and `ralph org status` uses this to say that the
// record is incomplete.
func (f *FeatureBinding) Complete() bool {
	return f.Split != "" && f.Feature != "" && f.Digest != "" && f.Branch != "" && f.Worktree != ""
}

// String names f for error text: the feature and split plan, then the
// digest, branch and worktree, and a note when f was read from a damaged
// record.
func (f *FeatureBinding) String() string {
	s := fmt.Sprintf("feature %q of split plan %q (digest %q, branch %q, worktree %q)", f.Feature, f.Split, f.Digest, f.Branch, f.Worktree)
	if !f.Complete() {
		s += " read from an incomplete record"
	}
	return s
}

// sameFeature reports whether two bindings are the same: both nil, or both
// complete with all five fields equal. An incomplete binding (a damaged
// record) equals nothing, not even itself, so a damaged record never passes
// as the org's binding.
func sameFeature(a, b *FeatureBinding) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Complete() && *a == *b
}

// validateFeatureBinding checks a binding Spawn is asked to record: split,
// feature, digest and branch must each be non-empty with no whitespace, `=`,
// `,` or control character (the Details tokens cannot hold them), and
// worktree must be an absolute path.
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
// one, then note after a space when non-empty (see scopeReservedPathsKey).
// The binding's worktree is not part of the Details (scopeReservedEvent).
func scopeReservedDetails(r Reservation, note string) string {
	details := scopeReservedPathsKey + strings.Join(r.Paths, ",")
	if f := r.Feature; f != nil {
		details += " " + scopeReservedSplitKey + "=" + f.Split +
			" " + scopeReservedFeatureKey + "=" + f.Feature +
			" " + scopeReservedDigestKey + "=" + f.Digest +
			" " + scopeReservedBranchKey + "=" + f.Branch
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
// reservation for the org until it is disbanded.
func featureBindingFromTokens(tokens []string, worktree string) *FeatureBinding {
	f := &FeatureBinding{Worktree: worktree}
	fields := map[string]*string{
		scopeReservedSplitKey:   &f.Split,
		scopeReservedFeatureKey: &f.Feature,
		scopeReservedDigestKey:  &f.Digest,
		scopeReservedBranchKey:  &f.Branch,
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
	return f
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
// or all five fields equal) pass with nothing to record, and anything else
// is refused: an org bound to a split plan feature refuses a reservation
// without a binding or with another one (another feature, or the same one
// approved again with another digest), even for the same paths; an org with
// a plain reservation refuses one with a binding; and a plain reservation
// with other paths is refused as before. When orgID holds none, a want with
// a binding is refused if the org is running (RunningOrgs: an active seat or
// an open workspace, as for a promoted session's leader or an org from
// `ralph org start <task>`), since that org was not started for the feature.
// Otherwise want's paths are checked against the reservation of every other
// org that holds one (an org with a reservation is running, see
// RunningOrgs): the first such org in org_id order whose paths overlap want
// refuses it, naming the overlapping paths. record is true when the caller
// must append the EventScopeReserved.
func reservationDecision(events []ManifestEvent, orgID string, want Reservation) (record bool, err error) {
	lives := currentOrgLives(events)
	if own, ok := lives[orgID]; ok && own.reservation.Paths != nil {
		held := own.reservation
		heldPaths, wantPaths := strings.Join(held.Paths, ","), strings.Join(want.Paths, ",")
		switch {
		case slices.Equal(held.Paths, want.Paths) && sameFeature(held.Feature, want.Feature):
			return false, nil
		case held.Feature != nil:
			return false, fmt.Errorf("org: org_id %q is bound to %s and reserves %s, so %s is refused: "+
				"ralph org disband --org-id %s first, or use another org_id",
				orgID, held.Feature, heldPaths, describeWantedReservation(want), orgID)
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
		if pairs := reserveOverlaps(want.Paths, lives[other].reservation.Paths); len(pairs) > 0 {
			return false, fmt.Errorf("org: reservation overlaps the reservation of running org_id %q: %s",
				other, strings.Join(pairs, ", "))
		}
	}
	return true, nil
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
