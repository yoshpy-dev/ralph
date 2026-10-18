package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// The director check in the director verbs (plan
// 2026-10-18-org-director-liveness, D7 and D8): attach's refusal of a
// current registration (attachedError) and what it adds for a headless one,
// the check in the CLI's words, which start's refusal uses too, and what
// status reads of the watch's mark and the director: line says of it.
// newOrgDirectorAttachCmd (org_director.go) calls attachedError, and
// readOrgDirectorReport and orgDirectorLine (org_director.go) call
// readOrgDirectorCheck and orgDirectorWatchCheckClause.

// attachedError is the store's ErrDirectorAttached refusal err followed by
// the commands that end or replace the current registration. The store's
// message names the registration and ends there, so the two ways out are
// said once, here (self-review L7 of org-director-attach). The current
// registration is read again here, without the lock, so it can be newer
// than the one the store refused on. A headless one (`ralph org director
// start`, starting or running) is ended with `ralph org director stop --id`,
// which closes its herdr pane, not with detach (plan
// 2026-10-15-org-director-start, AC2); one with launch cmd is stopped from a
// terminal other than the director's, and that stop ends its process (plan
// 2026-10-16-org-director-launch-cmd, D7 and D8). For a headless one the
// refusal ends with what is known of whether it is still alive
// (attachHeadlessChecks; plan 2026-10-18-org-director-liveness, D7); a
// promoted session's refusal says nothing of that.
func (rt *orgDirectorRuntime) attachedError(err error, driverName string) error {
	replace := rt.command("ralph org director attach --driver " + driverName + " --replace")
	var cur *org.DirectorRegistration
	if res, readErr := rt.store.Read(); readErr == nil {
		cur = res.Current
	}
	if cur != nil && cur.CmdLaunch() {
		return fmt.Errorf("%w: stop it with %s from a terminal other than the director's, which stops its process, "+
			"or replace it with %s, which leaves that process running until such a stop%s", err, rt.command("ralph org director stop --id "+inboxField(cur.ID)), replace,
			rt.attachHeadlessChecks(cur))
	}
	if cur != nil && cur.Headless() {
		return fmt.Errorf("%w: stop it with %s from a terminal outside herdr, which closes its herdr pane, "+
			"or replace it with %s, which leaves that pane open until such a stop%s", err, rt.command("ralph org director stop --id "+inboxField(cur.ID)), replace,
			rt.attachHeadlessChecks(cur))
	}
	detach := "ralph org director detach --id <its id>"
	if cur != nil {
		detach = "ralph org director detach --id " + inboxField(cur.ID)
	}
	return fmt.Errorf("%w: end it with %s from the session that attached it (or with --force in place of --id, when that session is gone), "+
		"or replace it with %s", err, rt.command(detach), replace)
}

// attachHeadlessChecks is what attach's refusal adds for cur, the current
// headless registration (plan 2026-10-18-org-director-liveness, D7), each
// part after "; ": when the watch's mark (director-alive.json) is cur's
// (org.DirectorAliveFor: the same id and attached_at), what the watch found
// and when; when cur is starting with a starter recorded, what ps reads of
// that `ralph org director start` process now (org.CheckDirectorStarter);
// and, when either says ended, that `ralph org director start` cleans cur up
// before it starts a new headless director. It is "" when there is neither.
// It makes no herdr call, sends nothing, and writes nothing: attach does not
// clean up. A mark that cannot be read adds nothing, as the refusal does not
// rest on it; every value read from the mark or the registration is escaped.
func (rt *orgDirectorRuntime) attachHeadlessChecks(cur *org.DirectorRegistration) string {
	var b strings.Builder
	ended := false
	if mark, err := org.ReadDirectorAlive(rt.stateDir); err == nil {
		if m := org.DirectorAliveFor(mark, cur); m != nil {
			fmt.Fprintf(&b, "; the watch checked %s at %s: %s", inboxField(cur.ID), inboxField(m.CheckedAt), orgDirectorCheckText(m.Check()))
			ended = m.State == org.DirectorCheckEnded
		}
	}
	if c, ok := org.CheckDirectorStarter(*cur, time.Now(), nil); ok {
		fmt.Fprintf(&b, "; its start (pid %d), checked just now: %s", cur.StarterPID, orgDirectorCheckText(c))
		ended = ended || c.State == org.DirectorCheckEnded
	}
	if ended {
		fmt.Fprintf(&b, "; as it has ended, %s cleans it up and then starts a new headless director (it checks %s again first)",
			rt.command("ralph org director start --driver <claude|codex>"), inboxField(cur.ID))
	}
	return b.String()
}

// orgDirectorCheckText is c, a check of a headless registration (plan
// 2026-10-18-org-director-liveness, D3), in the CLI's words: its state and
// its how, then its detail when it has one, each escaped (inboxField), as
// "ended (pane_closed)" or "unknown (agent_get_failed: <detail>)".
func orgDirectorCheckText(c org.DirectorCheck) string {
	text := inboxField(c.State) + " (" + inboxField(c.How)
	if c.Detail != "" {
		text += ": " + inboxField(c.Detail)
	}
	return text + ")"
}

// orgDirectorReportCheck is the check of a headless registration as status
// reports it (plan 2026-10-18-org-director-liveness, D8), the director.check
// of --json with four keys: the state (alive, ended, or unknown), how the
// check found it, when, and the detail of an unknown one ("" for alive and
// ended), copied from the watch's mark (org.DirectorAlive,
// director-alive.json, whose id and attached_at it leaves out) as read
// (readOrgDirectorCheck). It is not org.DirectorCheck, which nothing
// marshals, and a key added to the mark is not added here. The JSON keeps
// the values unescaped, as it keeps the registration's values; the text form
// escapes them.
type orgDirectorReportCheck struct {
	State     string `json:"state"`
	How       string `json:"how"`
	CheckedAt string `json:"checked_at"`
	Detail    string `json:"detail"`
}

// readOrgDirectorCheck reads what the watch's check found for cur, the
// current registration of the ledger at stateDir (plan
// 2026-10-18-org-director-liveness, D8). watchChecks is set when cur is
// headless and not stopped, the registrations `ralph org watch --all` checks
// each cycle; a promoted session's, and a stopped one's, are never checked,
// and nothing is read for them. For one the watch checks, check is the mark
// (org.ReadDirectorAlive) when it is cur's (org.DirectorAliveFor: the same id
// and attached_at), and nil when there is no mark or it is another
// registration's; markErr is the error of a mark that cannot be read or does
// not parse, which leaves check nil and never fails the status.
func readOrgDirectorCheck(stateDir string, cur org.DirectorRegistration) (watchChecks bool, check *orgDirectorReportCheck, markErr string) {
	if !cur.Headless() || cur.State == org.DirectorStateStopped {
		return false, nil, ""
	}
	mark, err := org.ReadDirectorAlive(stateDir)
	if err != nil {
		return true, nil, err.Error()
	}
	if m := org.DirectorAliveFor(mark, &cur); m != nil {
		return true, &orgDirectorReportCheck{State: m.State, How: m.How, CheckedAt: m.CheckedAt, Detail: m.Detail}, ""
	}
	return true, nil, ""
}

// orgDirectorWatchCheckClause is what the director: line says of r's current
// registration in place of "the registration does not tell whether its
// session is still running", when the watch checks it (r.watchChecks; plan
// 2026-10-18-org-director-liveness, D8): "checked by the watch at <time>: "
// and the check (orgDirectorCheckText), or "not checked yet" with when the
// watch checks it and, for a mark that cannot be read, why. When the check
// says ended and cleanup is set (the switch is on, so start can run), it adds
// the two ways out: `ralph org director start`, which cleans the registration
// up and then starts a new headless director, and `ralph org director stop
// --id`, each with hintStateDir as --state-dir (org.CommandWithStateDir).
// Every value read from the mark or the registration is escaped (inboxField).
func orgDirectorWatchCheckClause(r orgDirectorReport, hintStateDir string, cleanup bool) string {
	c := r.Check
	if c == nil {
		clause := "not checked yet: ralph org watch --all checks it each cycle"
		if r.markErr != "" {
			clause += ", but its mark cannot be read (" + inboxField(r.markErr) + ")"
		}
		return clause
	}
	clause := "checked by the watch at " + inboxField(c.CheckedAt) + ": " + orgDirectorCheckText(org.DirectorCheck{State: c.State, How: c.How, Detail: c.Detail})
	if cleanup && c.State == org.DirectorCheckEnded {
		id := inboxField(r.Registration.ID)
		clause += "; as it has ended, " + org.CommandWithStateDir("ralph org director start --driver <claude|codex>", hintStateDir) +
			" cleans it up and then starts a new headless director, or " + org.CommandWithStateDir("ralph org director stop --id "+id, hintStateDir) + " stops it"
	}
	return clause
}
