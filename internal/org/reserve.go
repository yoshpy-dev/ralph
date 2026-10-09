package org

import (
	"errors"
	"fmt"
	"path"
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

// wholeRepoReserve is the reservation path that stands for the whole repo. It
// overlaps every other path.
const wholeRepoReserve = "."

// scopeReservedPathsKey starts an EventScopeReserved's Details: the
// normalized paths joined with commas (`paths=docs/,internal/auth/`),
// optionally followed by a space and a note (the failed-self-close
// compensation adds `restored: <why>`). The path rules reject commas and
// whitespace, so the list reads back unambiguously.
const scopeReservedPathsKey = "paths="

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

// scopeReservedDetails is the Details of an EventScopeReserved for paths
// (already normalized), with note appended after a space when non-empty.
func scopeReservedDetails(paths []string, note string) string {
	details := scopeReservedPathsKey + strings.Join(paths, ",")
	if note != "" {
		details += " " + note
	}
	return details
}

// reservedPathsFromDetails reads the paths back from an EventScopeReserved's
// Details (see scopeReservedDetails), normalized again. ok is false when the
// Details hold no readable path list.
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

// scopeReservedEvent is the org-level EventScopeReserved recording paths as
// orgID's reservation.
func scopeReservedEvent(ts, orgID string, paths []string, note string, dryRun bool) ManifestEvent {
	return ManifestEvent{
		TS: ts, OrgID: orgID, SeatID: "", Event: EventScopeReserved,
		DryRun: dryRun, Details: scopeReservedDetails(paths, note),
	}
}

// orgLife is what an org's real events after its latest real `disbanded`
// (all of them when it has none) still hold open: the workspaces created in
// that span and not closed since, oldest first, and the latest reservation
// recorded in that span (nil: none).
type orgLife struct {
	workspaces []string
	reserved   []string
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
// after it is. A reservation whose Details cannot be read is taken as the
// whole repo, so a damaged record blocks overlapping reservations instead of
// hiding them until the org is disbanded.
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
			paths, ok := reservedPathsFromDetails(ev.Details)
			if !ok {
				paths = []string{wholeRepoReserve}
			}
			life(ev.OrgID).reserved = paths
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
		if len(l.workspaces) > 0 || l.reserved != nil {
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
// the org holds none.
func ActiveReservation(events []ManifestEvent, orgID string) []string {
	if l, ok := currentOrgLives(events)[orgID]; ok {
		return l.reserved
	}
	return nil
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
// whether orgID may hold the reservation want (normalized). When orgID
// already holds one, the same set passes with nothing to record and a
// different set is refused. Otherwise want is checked against the
// reservation of every other org that holds one (an org with a reservation
// is running, see RunningOrgs): the first such org in org_id order whose
// paths overlap want refuses it, naming the overlapping paths. record is true
// when the caller must append the EventScopeReserved.
func reservationDecision(events []ManifestEvent, orgID string, want []string) (record bool, err error) {
	lives := currentOrgLives(events)
	if own, ok := lives[orgID]; ok && own.reserved != nil {
		if slices.Equal(own.reserved, want) {
			return false, nil
		}
		return false, fmt.Errorf("org: org_id %q already reserves %s, so a different reservation (%s) is refused: "+
			"a reservation stays until ralph org disband --org-id %s",
			orgID, strings.Join(own.reserved, ","), strings.Join(want, ","), orgID)
	}
	others := make([]string, 0, len(lives))
	for other, l := range lives {
		if other != orgID && l.reserved != nil {
			others = append(others, other)
		}
	}
	slices.Sort(others)
	for _, other := range others {
		if pairs := reserveOverlaps(want, lives[other].reserved); len(pairs) > 0 {
			return false, fmt.Errorf("org: reservation overlaps the reservation of running org_id %q: %s",
				other, strings.Join(pairs, ", "))
		}
	}
	return true, nil
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
