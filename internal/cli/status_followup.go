package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// The reservations and the follow-ups `ralph status` shows (issue #270; plan
// docs/plans/active/2026-10-30-org-followups.md, D2, D4, D5, AC4, AC5 and
// AC8). Each running org's reservation goes in its group, and an org that
// holds one with no seat gets a group of its own (withStatusReservations).
// The ledger's open follow-ups come after the groups, each ready when the
// manifest records a merge of its after-branch since it was recorded
// (org.FollowupRecordedMerge), unknown when the time it was recorded cannot
// be read, else waiting. Both read only the manifest events status already
// read and followups.jsonl, so status still runs no gh: `ralph org
// merge-order` is the command that asks GitHub.

// statusFollowupSourceRecorded is the source of every follow-up status
// shows: its state comes from the ledger's records of merges, not from gh
// (D2). `ralph org merge-order` shows org.FollowupMergeSourceGH instead.
const statusFollowupSourceRecorded = "recorded"

// statusFollowupWaitingDetail is the detail of a waiting follow-up whose
// added_at can be read. Like the details of `ralph org merge-order`, it
// names commands without backticks.
const statusFollowupWaitingDetail = "no merge recorded in the ledger since the follow-up; ralph org merge-order asks GitHub"

// statusFollowupJSON is one open follow-up as `ralph status` shows it, in
// the table and as an item of --json's followups. State is
// org.FollowupMergeReady, org.FollowupMergeWaiting or
// org.FollowupMergeUnknown, the values `ralph org merge-order` uses; Source
// is always statusFollowupSourceRecorded.
// MergedEvent and MergedAt are the event (pr_merged or worktree_cleaned) and
// the time of the recorded merge that makes it ready, "" otherwise.
// Detail says, in one sentence, why it has its state and what to do next.
type statusFollowupJSON struct {
	ID          string `json:"id"`
	Issue       int    `json:"issue"`
	AfterBranch string `json:"after_branch"`
	AddedAt     string `json:"added_at"`
	State       string `json:"state"`
	Source      string `json:"source"`
	MergedEvent string `json:"merged_event"`
	MergedAt    string `json:"merged_at"`
	Detail      string `json:"detail"`
}

// statusFollowupsJSON is --json's followups, the same keys on the empty path
// and the populated one: the open follow-ups (Items, never nil), the record's
// damage counts as `ralph org followup list --json` gives them and
// merge-order warns about them (org.FollowupReadResult), and Error, why the
// record could not be read, "" when it was read or does not exist. A reader
// tells a record that could not be read from one with no open follow-up by
// Error, without reading stderr.
type statusFollowupsJSON struct {
	Items         []statusFollowupJSON `json:"items"`
	CorruptLines  int                  `json:"corrupt_lines"`
	IgnoredEvents int                  `json:"ignored_events"`
	Error         string               `json:"error"`
}

// withStatusReservations returns groups with the reservation of each running
// org (org.RunningOrgs, org.ActiveReservation) set on its group, and a group
// with no seat added for an org that holds a reservation but has no seat
// (D4). With filterOrgID set only that org is looked at, as for the seats.
// The groups stay in OrgID order.
func withStatusReservations(events []org.ManifestEvent, groups []orgGroup, filterOrgID string) []orgGroup {
	index := make(map[string]int, len(groups))
	for i, g := range groups {
		index[g.OrgID] = i
	}
	added := false
	for _, orgID := range org.RunningOrgs(events) {
		if filterOrgID != "" && orgID != filterOrgID {
			continue
		}
		reservation := org.ActiveReservation(events, orgID)
		if reservation == nil {
			continue
		}
		if i, ok := index[orgID]; ok {
			groups[i].Reservation = reservation
			continue
		}
		groups = append(groups, orgGroup{OrgID: orgID, Reservation: reservation})
		added = true
	}
	if added {
		sort.SliceStable(groups, func(i, j int) bool { return groups[i].OrgID < groups[j].OrgID })
	}
	return groups
}

// printStatusReservation prints g's reservation as `ralph org status`
// prints it (printStatusTable), indented under the group: `  reserved:
// <path>, <path>`, or nothing when g holds none. The paths were normalized
// when they were read, so they hold no whitespace or control character.
func printStatusReservation(w io.Writer, g orgGroup) {
	if len(g.Reservation) > 0 {
		_, _ = fmt.Fprintf(w, "  reserved: %s\n", strings.Join(g.Reservation, ", "))
	}
}

// statusReservationJSON is g's reservation for --json: never nil, so an org
// that holds none carries an empty array.
func statusReservationJSON(g orgGroup) []string {
	return append([]string{}, g.Reservation...)
}

// readStatusFollowups reads the ledger's follow-up record in stateDir with no
// lock, as `ralph org followup list` reads it (a missing file is no
// follow-up, and nothing is created), and returns its open follow-ups in ID
// order, each with the state events give it (statusFollowup), and the
// record's damage counts. Damage in the record is warned about on stderr as
// the followup verbs warn about it (warnFollowupDamage). status only reads,
// so a record that cannot be read is warned about too and shows no
// follow-up, with the error in Error, instead of failing the command. Items
// is never nil, so --json always carries the items array.
func readStatusFollowups(stderr io.Writer, stateDir string, events []org.ManifestEvent) statusFollowupsJSON {
	followups := statusFollowupsJSON{Items: []statusFollowupJSON{}}
	store := org.NewFollowupStore(stateDir)
	res, err := store.Read()
	if err != nil {
		followups.Error = err.Error()
		_, _ = fmt.Fprintf(stderr, "warning: status: %v; no follow-up is shown\n", err)
		return followups
	}
	followups.CorruptLines, followups.IgnoredEvents = res.CorruptLines, res.IgnoredEvents
	warnFollowupDamage(stderr, store.Path(), res)
	for _, f := range res.Items {
		if f.State == org.FollowupStateOpen {
			followups.Items = append(followups.Items, statusFollowup(events, f))
		}
	}
	return followups
}

// statusFollowup is f with the state the manifest events give it (D2, D5):
// unknown, with the detail `ralph org merge-order` gives it
// (org.FollowupAddedTime), when f's added_at is not an RFC3339 time (a hand
// edit), so no merge can be compared with it; ready, naming the event and
// its time in merge-order's words (org.FollowupReadyDetail), when
// org.FollowupRecordedMerge finds a pr_merged or a worktree_cleaned of f's
// after-branch at or after f was recorded; else waiting, which says that
// `ralph org merge-order` asks GitHub.
func statusFollowup(events []org.ManifestEvent, f org.Followup) statusFollowupJSON {
	out := statusFollowupJSON{
		ID: f.ID, Issue: f.Issue, AfterBranch: f.AfterBranch, AddedAt: f.AddedAt, Source: statusFollowupSourceRecorded,
	}
	if _, unreadable := org.FollowupAddedTime(f); unreadable != "" {
		out.State, out.Detail = org.FollowupMergeUnknown, unreadable
		return out
	}
	if merged, at, event := org.FollowupRecordedMerge(events, f); merged {
		out.State, out.MergedEvent, out.MergedAt = org.FollowupMergeReady, event, at
		out.Detail = org.FollowupReadyDetail(f, "the ledger records "+event, at)
		return out
	}
	out.State, out.Detail = org.FollowupMergeWaiting, statusFollowupWaitingDetail
	return out
}

// printStatusFollowups prints, when there is at least one open follow-up, a
// blank line, `follow-ups:`, and a table of them:
//
//	ID	ISSUE	AFTER_BRANCH	STATE	SOURCE	DETAIL
//	<one tab-separated row per follow-up>
//
// as printMergeOrderFollowups prints merge-order's, with SOURCE in place of
// PR. Every value goes through inboxField. Nothing is printed with no
// follow-up.
func printStatusFollowups(w io.Writer, followups []statusFollowupJSON) {
	if len(followups) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w, "\nfollow-ups:")
	_, _ = fmt.Fprintln(w, "ID\tISSUE\tAFTER_BRANCH\tSTATE\tSOURCE\tDETAIL")
	for _, f := range followups {
		_, _ = fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\t%s\n",
			inboxField(f.ID), f.Issue, inboxField(f.AfterBranch), inboxField(f.State), inboxField(f.Source), inboxField(f.Detail))
	}
}
