package cli

import (
	"fmt"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// The director's wait in the director verbs (plan
// docs/plans/active/2026-10-19-org-director-wait.md, D5): what status reads
// of the mark director-wait.json, which `ralph org wait --inbox --director
// <id>` writes, and what the wait: line of `ralph org director status` and
// the director: line of a promoted session say of it. readOrgDirectorReport,
// orgDirectorLine, and printOrgDirectorReport (org_director.go) call
// readOrgDirectorWait, orgDirectorWaitClause, and orgDirectorWaitLine. The
// mark says what the wait wrote, so neither line says that the director is
// waiting or alive.

// orgDirectorWaitNow is the clock the stale check of the wait's mark reads;
// tests replace it.
var orgDirectorWaitNow = time.Now

// orgDirectorReportWait is the wait's mark as status reports it (D5), the
// director.wait of --json with eight keys: the wait's wait_id and pid, when
// it started and last wrote the mark, the watch_accepting it wrote, and, once
// it ended, when, how (a DirectorWaitEnd* value), and the detail ("" for a
// running wait), copied from the mark (org.DirectorWaitMark) as read
// (readOrgDirectorWait). Like orgDirectorReportCheck it leaves out the
// mark's id and attached_at, which are the registration's (only a mark of
// the current registration is reported), and keeps every key, so a key
// added to the mark is not added here. The JSON keeps the values unescaped,
// as it keeps the registration's values; the text forms escape them.
type orgDirectorReportWait struct {
	WaitID         string `json:"wait_id"`
	PID            int    `json:"pid"`
	StartedAt      string `json:"started_at"`
	SeenAt         string `json:"seen_at"`
	WatchAccepting bool   `json:"watch_accepting"`
	EndedAt        string `json:"ended_at"`
	End            string `json:"end"`
	Detail         string `json:"detail"`
}

// readOrgDirectorWait reads the wait's mark for cur, the current
// registration of the ledger at stateDir (D5), headless or not. wait is the
// mark (org.ReadDirectorWait) when it is cur's (org.DirectorWaitFor: the same
// id and attached_at), and nil when there is no mark or it is another
// registration's. stale is set for a running wait (no end) whose seen_at is
// more than two org.DirectorWaitMarkInterval before orgDirectorWaitNow; a
// seen_at that does not parse as RFC3339 is never stale. It is decided here,
// once, so the two lines of one status agree. waitErr is the error of a mark
// that cannot be read or does not parse, which leaves wait nil and never
// fails the status.
func readOrgDirectorWait(stateDir string, cur org.DirectorRegistration) (wait *orgDirectorReportWait, stale bool, waitErr string) {
	mark, err := org.ReadDirectorWait(stateDir)
	if err != nil {
		return nil, false, err.Error()
	}
	m := org.DirectorWaitFor(mark, &cur)
	if m == nil {
		return nil, false, ""
	}
	wait = &orgDirectorReportWait{WaitID: m.WaitID, PID: m.PID, StartedAt: m.StartedAt, SeenAt: m.SeenAt,
		WatchAccepting: m.WatchAccepting, EndedAt: m.EndedAt, End: m.End, Detail: m.Detail}
	if m.End == "" {
		if seen, err := time.Parse(time.RFC3339, m.SeenAt); err == nil {
			stale = orgDirectorWaitNow().Sub(seen) > 2*org.DirectorWaitMarkInterval
		}
	}
	return wait, stale, ""
}

// orgDirectorWaitFacts is what the two lines say of w, the wait's mark,
// after "wait --inbox --director": "(pid N) started at <started_at> and last
// wrote its mark at <seen_at>" for a running wait, with ", and has not
// written it since (it writes every 30s)" when stale, or "(pid N) ended at
// <ended_at> (<end>)" for one that ended; the detail is left to the JSON.
// Every value read from the mark is escaped (inboxField).
func orgDirectorWaitFacts(w *orgDirectorReportWait, stale bool) string {
	if w.End != "" {
		return fmt.Sprintf("(pid %d) ended at %s (%s)", w.PID, inboxField(w.EndedAt), inboxField(w.End))
	}
	facts := fmt.Sprintf("(pid %d) started at %s and last wrote its mark at %s", w.PID, inboxField(w.StartedAt), inboxField(w.SeenAt))
	if stale {
		facts += ", and has not written it since (it writes every " + org.DirectorWaitMarkInterval.String() + ")"
	}
	return facts
}

// orgDirectorWaitLine is the wait: line of `ralph org director status` for
// r, whose registration is current (D5): what the mark of its wait says
// (orgDirectorWaitFacts), that it has none, or why the mark cannot be read.
func orgDirectorWaitLine(r orgDirectorReport) string {
	switch {
	case r.waitErr != "":
		return "wait: " + org.DirectorWaitFileName + " cannot be read: " + inboxField(r.waitErr)
	case r.Wait == nil:
		return "wait: no wait --inbox --director mark for this registration"
	}
	return "wait: the wait --inbox --director of " + inboxField(r.Registration.ID) + " " + orgDirectorWaitFacts(r.Wait, r.waitStale)
}

// orgDirectorWaitClause is what the director: line of a promoted session adds
// after "the registration does not tell whether its session is still
// running" (D5): the facts of orgDirectorWaitLine in short form, after "; ".
// A headless registration's line does not call it.
func orgDirectorWaitClause(r orgDirectorReport) string {
	switch {
	case r.waitErr != "":
		return "; " + org.DirectorWaitFileName + " cannot be read: " + inboxField(r.waitErr)
	case r.Wait == nil:
		return "; no wait --inbox --director mark for it"
	}
	return "; its wait --inbox --director " + orgDirectorWaitFacts(r.Wait, r.waitStale)
}
