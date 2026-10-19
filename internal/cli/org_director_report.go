package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// The director report in the director verbs: the director switch and
// registration of a ledger as `ralph status` and `ralph org director status`
// report them (orgDirectorReport, read by readOrgDirectorReport), the
// director: line (orgDirectorLine) and its parts, the text form of director
// status (printOrgDirectorReport), and the warnings about damage in
// director.jsonl (warnDirectorDamage). newOrgDirectorStatusCmd
// (org_director.go) and ralph status (status.go) call them.

// The states of orgDirectorReport.State.
const (
	orgDirectorStateAttached    = "attached"
	orgDirectorStateNotAttached = "not_attached"
	orgDirectorStateDisabled    = "disabled"
	orgDirectorStateUnreadable  = "unreadable"
)

// orgDirectorReport is the director switch and registration of a ledger as
// `ralph status` (its director: line and director key) and `ralph org
// director status` report them, and their --json wire shape. AutoMerge is
// [org.director].auto_merge, read from the same ralph.toml as Enabled and
// always present in the JSON, false included; it does not change State.
// State is disabled when Enabled is false, whatever the registration;
// otherwise unreadable when director.jsonl could not be read, attached when
// a registration is current, and not_attached. Registration is the current
// registration (null when there is none or the file could not be read); a
// headless one carries its launch, state, token, model, permission mode,
// workspace, and pane (org.DirectorRegistration), and one with launch cmd
// its pid, start time, and terminal name instead of the workspace and the
// pane (orgDirectorReportRegistration). OpenHeadless lists the other open
// headless registrations, detached but with their pane or process not yet
// closed, which `ralph org director stop --id` closes (omitted when none).
// Error is the read error, set whenever the read failed (with disabled too),
// and CorruptLines and IgnoredEvents the counts of org.DirectorReadResult.
// Check is what the watch's check last found for the current registration
// when it is headless and not stopped (plan 2026-10-18-org-director-liveness,
// D8: watchChecks), read from the watch's mark when the mark is that
// registration's (readOrgDirectorCheck); it is omitted when the watch has not
// checked it yet, when the mark is another registration's or cannot be read
// (markErr, which only the text form shows), and for a promoted session.
// Wait is the mark of the director's wait, `ralph org wait --inbox
// --director` (plan 2026-10-19-org-director-wait, D5), for any current
// registration, headless or not, enabled false included, when the mark is
// that registration's (readOrgDirectorWait); it is omitted when there is no
// such mark or the mark cannot be read (waitErr, which only the text form
// shows). waitStale is set for a running wait that has not rewritten its mark
// for two of its intervals.
type orgDirectorReport struct {
	Enabled       bool                            `json:"enabled"`
	AutoMerge     bool                            `json:"auto_merge"`
	State         string                          `json:"state"`
	Registration  *orgDirectorReportRegistration  `json:"registration"`
	Check         *orgDirectorReportCheck         `json:"check,omitempty"`
	Wait          *orgDirectorReportWait          `json:"wait,omitempty"`
	OpenHeadless  []orgDirectorReportRegistration `json:"open_headless,omitempty"`
	Error         string                          `json:"error,omitempty"`
	CorruptLines  int                             `json:"corrupt_lines"`
	IgnoredEvents int                             `json:"ignored_events"`
	watchChecks   bool
	markErr       string
	waitStale     bool
	waitErr       string
}

// orgDirectorReportRegistration is a registration as status reports it: the
// org.DirectorRegistration the fold made, its fields inline in the JSON, and
// for a launch cmd TerminalName, the name launch_cmd gave the terminal
// (org.DirectorAgentName of the token; plan
// 2026-10-16-org-director-launch-cmd, AC8). The name follows from the token
// and is not recorded, so it is added here, not to the fold's registration,
// and a herdr or promoted registration's JSON is what it was before.
type orgDirectorReportRegistration struct {
	org.DirectorRegistration
	TerminalName string `json:"terminal_name,omitempty"`
}

// newOrgDirectorReportRegistration is reg as status reports it.
func newOrgDirectorReportRegistration(reg org.DirectorRegistration) orgDirectorReportRegistration {
	r := orgDirectorReportRegistration{DirectorRegistration: reg}
	if reg.CmdLaunch() {
		r.TerminalName = org.DirectorAgentName(reg.Token)
	}
	return r
}

// readOrgDirectorReport reads the registration of the ledger at stateDir
// (org.DirectorStore.Read, no lock) into an orgDirectorReport with
// director's Enabled as the switch and its AutoMerge, the watch's check of
// the current registration (readOrgDirectorCheck), and the mark of that
// registration's wait --inbox --director (readOrgDirectorWait).
func readOrgDirectorReport(stateDir string, director config.OrgDirectorConfig) orgDirectorReport {
	report := orgDirectorReport{Enabled: director.Enabled, AutoMerge: director.AutoMerge}
	res, err := org.NewDirectorStore(stateDir).Read()
	switch {
	case err != nil:
		report.Error, report.State = err.Error(), orgDirectorStateUnreadable
	case res.Current != nil:
		cur := newOrgDirectorReportRegistration(*res.Current)
		report.Registration, report.State = &cur, orgDirectorStateAttached
		report.watchChecks, report.Check, report.markErr = readOrgDirectorCheck(stateDir, *res.Current)
		report.Wait, report.waitStale, report.waitErr = readOrgDirectorWait(stateDir, *res.Current)
	default:
		report.State = orgDirectorStateNotAttached
	}
	for _, reg := range res.OpenHeadless {
		if res.Current == nil || reg.ID != res.Current.ID {
			report.OpenHeadless = append(report.OpenHeadless, newOrgDirectorReportRegistration(reg))
		}
	}
	report.CorruptLines, report.IgnoredEvents = res.CorruptLines, res.IgnoredEvents
	if !director.Enabled {
		report.State = orgDirectorStateDisabled
	}
	return report
}

// orgDirectorLine is the one director: line that `ralph status` and `ralph
// org director status` print for r. A headless registration adds its state,
// model, pane, and workspace (for a launch cmd, the launch, the terminal
// name, and the pid: orgDirectorLaunchDetails). A registration whose wake
// has push ends its details with the wake and where the watch sends
// (orgDirectorPushDetails); a pull-only one's line is that of stage 8a.
// After the details, a registration the watch checks (headless and not
// stopped) says what the watch's check found (orgDirectorWatchCheckClause),
// and every other one (a promoted session's, or a headless one recorded
// stopped while still current) that the registration does not tell whether
// its session is still running. A promoted session's line adds after that
// sentence what the mark of its wait --inbox --director says
// (orgDirectorWaitClause; plan 2026-10-19-org-director-wait, D5); a headless
// registration's line says nothing of that mark, and neither does the
// disabled line. Disabled adds the check after the stop command. Attached
// and not attached end with what [org.director].auto_merge means
// (orgDirectorAutoMergeClause), and disabled says that auto_merge = true has
// no effect; unreadable says nothing of it. The open headless registrations
// that are not current are named at the end (orgDirectorOpenHeadlessNote).
// The values from director.jsonl and the marks, which a human can edit, have
// their control characters escaped (inboxField), so the line stays one line.
// The stop commands it names repeat hintStateDir as --state-dir
// (org.CommandWithStateDir; the caller passes orgInboxHintStateDir's value,
// the state dir only when --state-dir was given), as the other director
// commands do.
func orgDirectorLine(r orgDirectorReport, hintStateDir string) string {
	note := orgDirectorOpenHeadlessNote(r.OpenHeadless, hintStateDir)
	switch r.State {
	case orgDirectorStateAttached:
		reg := r.Registration
		details := "driver=" + inboxField(reg.Driver)
		if reg.Headless() {
			details += ", headless " + inboxField(reg.State) + ", model=" + inboxField(reg.Model) + orgDirectorLaunchDetails(reg.DirectorRegistration)
		}
		details += ", attached at " + inboxField(reg.AttachedAt)
		if reg.User != "" {
			details += ", by " + inboxField(reg.User)
		}
		if !reg.Headless() && reg.PaneID != "" {
			details += ", pane " + inboxField(reg.PaneID)
		}
		details += orgDirectorPushDetails(reg.DirectorRegistration)
		alive := "the registration does not tell whether its session is still running"
		switch {
		case r.watchChecks:
			alive = orgDirectorWatchCheckClause(r, hintStateDir, true)
		case !reg.Headless():
			alive += orgDirectorWaitClause(r)
		}
		return "director: attached " + inboxField(reg.ID) + " (" + details + ") -- " + alive + "; " +
			"inbox items it does not ack reach the human at the ack deadline" + orgDirectorAutoMergeClause(r.AutoMerge, true) + note
	case orgDirectorStateDisabled:
		state := "[org.director].enabled = false"
		if r.AutoMerge {
			state += ", so [org.director].auto_merge = true has no effect"
		}
		switch {
		case r.Error != "":
			state += "; director.jsonl cannot be read: " + inboxField(r.Error)
		case r.Registration != nil && r.Registration.Headless():
			reg := r.Registration
			state += "; registration " + inboxField(reg.ID) + " is still recorded, headless " + inboxField(reg.State) + orgDirectorLaunchDetails(reg.DirectorRegistration) +
				": stop it with " + org.CommandWithStateDir("ralph org director stop --id "+inboxField(reg.ID), hintStateDir)
			if r.watchChecks {
				// start refuses while the switch is off, so the check names no
				// cleanup by start: the stop above is the way out.
				state += "; " + orgDirectorWatchCheckClause(r, hintStateDir, false)
			}
		case r.Registration != nil:
			state += "; registration " + inboxField(r.Registration.ID) + " is still recorded"
		}
		return "director: disabled (" + state + ") -- ralph org escalate sends every item to the human at once" + note
	case orgDirectorStateUnreadable:
		return "director: unreadable (" + inboxField(r.Error) + ")"
	}
	return "director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work" +
		orgDirectorAutoMergeClause(r.AutoMerge, false) + note
}

// orgDirectorAutoMergeClause is the part of the director: line that says,
// while the switch is on, what [org.director].auto_merge means: with false,
// merges are left to the human; with true, the director merges the PRs
// `ralph org merge-order` lists as ready with `ralph org merge` without
// asking, once one is attached when attached is false (plan
// 2026-10-17-org-auto-merge, AC2).
func orgDirectorAutoMergeClause(autoMerge, attached bool) string {
	switch {
	case !autoMerge:
		return "; merges are left to the human ([org.director].auto_merge = false)"
	case attached:
		return "; the director merges the PRs ralph org merge-order lists as ready with ralph org merge, without asking ([org.director].auto_merge = true)"
	}
	return "; once attached, a director merges the PRs ralph org merge-order lists as ready with ralph org merge, without asking ([org.director].auto_merge = true)"
}

// orgDirectorHerdrIDs is ", pane <id>" and ", workspace <id>" for the herdr
// ids recorded for a headless registration, each only when recorded.
func orgDirectorHerdrIDs(reg org.DirectorRegistration) string {
	ids := ""
	if reg.PaneID != "" {
		ids += ", pane " + inboxField(reg.PaneID)
	}
	if reg.WorkspaceID != "" {
		ids += ", workspace " + inboxField(reg.WorkspaceID)
	}
	return ids
}

// orgDirectorLaunchDetails is what the director: line and the open_headless
// lines say of where a headless registration runs: for a launch herdr its
// herdr ids (orgDirectorHerdrIDs), and for a launch cmd ", launch cmd", the
// terminal name launch_cmd gave the terminal (", name director_<token>"),
// and ", pid <pid>" once the check-in recorded one (plan
// 2026-10-16-org-director-launch-cmd, AC8).
func orgDirectorLaunchDetails(reg org.DirectorRegistration) string {
	if !reg.CmdLaunch() {
		return orgDirectorHerdrIDs(reg)
	}
	details := ", launch " + inboxField(reg.Launch) + ", name " + inboxField(org.DirectorAgentName(reg.Token))
	if reg.PID > 0 {
		details += ", pid " + strconv.Itoa(reg.PID)
	}
	return details
}

// orgDirectorOpenHeadlessNote is the end of the director: line that names
// the open headless registrations that are not current (detached, or
// replaced, with their herdr pane or launch_cmd process not stopped yet) and
// how to stop them, or "" when there are none (plan
// 2026-10-15-org-director-start, AC10). The stop command repeats
// hintStateDir as --state-dir, as orgDirectorLine's does.
func orgDirectorOpenHeadlessNote(open []orgDirectorReportRegistration, hintStateDir string) string {
	if len(open) == 0 {
		return ""
	}
	names := make([]string, 0, len(open))
	for _, reg := range open {
		names = append(names, inboxField(reg.ID)+" ("+inboxField(reg.State)+orgDirectorLaunchDetails(reg.DirectorRegistration)+")")
	}
	if len(open) == 1 {
		return "; the headless director " + names[0] + " is still open but not attached: stop it with " +
			org.CommandWithStateDir("ralph org director stop --id "+inboxField(open[0].ID), hintStateDir)
	}
	return "; the headless directors " + strings.Join(names, ", ") + " are still open but not attached: stop each with " +
		org.CommandWithStateDir("ralph org director stop --id <id>", hintStateDir)
}

// warnDirectorDamage writes to w (stderr, so --json output stays clean) the
// lines of director.jsonl at path that did not parse and the events the fold
// ignored, as warnInboxDamage does for the inbox.
func warnDirectorDamage(w io.Writer, path string, r orgDirectorReport) {
	if r.CorruptLines > 0 {
		_, _ = fmt.Fprintf(w, "warning: %d unreadable director line(s) skipped in %s\n", r.CorruptLines, path)
	}
	if r.IgnoredEvents > 0 {
		_, _ = fmt.Fprintf(w, "warning: %d director event(s) ignored in %s (unknown kind or id, an attached with an unknown driver or a malformed token, a reused id, "+
			"or while another registration is current, a detached of an id that is not the current one, a workspace_created or pane_created "+
			"that names no open headless registration or repeats one, a launched of an id that is not the current one starting, "+
			"or a stopped that names no open headless registration)\n", r.IgnoredEvents, path)
	}
}

// printOrgDirectorReport prints `ralph org director status`'s text form: the
// director: line (orgDirectorLine), enabled, auto_merge, and, when a
// registration is current, one line per field of it, with control
// characters escaped. The fields of a headless registration (launch to
// workspace_id) and pane_id are left out when not set, so a promoted
// session's lines are those of stage 8a-1 (with auto_merge after enabled),
// and so are pid, start_time, and terminal_name, which only a launch cmd
// has (plan 2026-10-16-org-director-launch-cmd, AC8). wake_target follows
// wake only when a promoted session was attached with --wake-target (plan
// 2026-10-18-org-director-wake, AC2): a headless registration's target is
// on its pane_id or terminal_name line. After the fields, a current
// registration, headless or not, gets the wait: line (orgDirectorWaitLine;
// plan 2026-10-19-org-director-wait, D5). Then one open_headless line per
// open headless registration that is not current. hintStateDir is
// orgDirectorLine's.
func printOrgDirectorReport(w io.Writer, r orgDirectorReport, hintStateDir string) {
	_, _ = fmt.Fprintln(w, orgDirectorLine(r, hintStateDir))
	_, _ = fmt.Fprintf(w, "enabled: %t\n", r.Enabled)
	_, _ = fmt.Fprintf(w, "auto_merge: %t\n", r.AutoMerge)
	if reg := r.Registration; reg != nil {
		for _, f := range []struct {
			name, value string
			omitEmpty   bool
		}{
			{"id", reg.ID, false}, {"driver", reg.Driver, false}, {"wake", strings.Join(reg.Wake, ","), false},
			{"wake_target", reg.WakeTarget, true},
			{"attached_at", reg.AttachedAt, false}, {"user", reg.User, false}, {"cwd", reg.Cwd, false},
			{"herdr_pane", yesNo(reg.HerdrPane), false},
		} {
			if f.omitEmpty && f.value == "" {
				continue
			}
			_, _ = fmt.Fprintf(w, "%s: %s\n", f.name, inboxField(f.value))
		}
		pid := ""
		if reg.PID > 0 {
			pid = strconv.Itoa(reg.PID)
		}
		for _, f := range []struct{ name, value string }{
			{"launch", reg.Launch}, {"state", reg.State}, {"model", reg.Model}, {"permission_mode", reg.PermissionMode},
			{"token", reg.Token}, {"workspace_id", reg.WorkspaceID}, {"pane_id", reg.PaneID},
			{"pid", pid}, {"start_time", reg.StartTime}, {"terminal_name", reg.TerminalName},
		} {
			if f.value != "" {
				_, _ = fmt.Fprintf(w, "%s: %s\n", f.name, inboxField(f.value))
			}
		}
		_, _ = fmt.Fprintln(w, orgDirectorWaitLine(r))
	}
	for _, reg := range r.OpenHeadless {
		_, _ = fmt.Fprintf(w, "open_headless: %s (%s%s)\n", inboxField(reg.ID), inboxField(reg.State), orgDirectorLaunchDetails(reg.DirectorRegistration))
	}
}
