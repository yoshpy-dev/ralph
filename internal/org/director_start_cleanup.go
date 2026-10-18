package org

import "fmt"

// Start's cleanup of a current headless registration that its check found
// ended (plan docs/plans/active/2026-10-18-org-director-liveness.md, D6):
// checkCurrentDirector, checkDirectorStart's first look at the current
// registration, and cleanUpEndedDirector, which runs the closes StopDirector
// shares (closeHerdrDirector in director_stop.go, stopDirectorProcess in
// director_cmd.go) on the registration as that check read it and records the
// stop. StartDirector and checkDirectorStart are in director_start.go.

// checkCurrentDirector is checkDirectorStart's first look at cur, the
// current registration as a read without the lock found it (plan
// 2026-10-18-org-director-liveness, D6). A promoted session's registration
// is refused as before (*DirectorAttachedError), with no check. For a
// headless one it runs CheckDirector on the spot, not the watch's mark, and:
//   - a director.jsonl that cannot be read is that error;
//   - with nothing to check (the registration is stopped while still
//     current, or is no longer current or headless by the check's read) it is
//     refused as before, naming cur;
//   - alive or unknown is refused as before, with the registration the check
//     read and what it found (Check);
//   - ended runs o.directorStartAfterCheck (a test-only seam) and is cleaned
//     up on the registration as the check read it, version A
//     (cleanUpEndedDirector), and once that is recorded its line goes to
//     p.Notice and start goes on (nil). A cleanup that did not get as far as
//     the record is its *DirectorCleanupError, with nothing written.
//
// AttachHeadless decides again under the lock either way.
func (o *Org) checkCurrentDirector(p *DirectorStartParams, cur *DirectorRegistration) error {
	refused := &DirectorAttachedError{Current: cur, Err: directorStartAttachedErr(cur)}
	if !cur.Headless() {
		return refused
	}
	chk, err := o.CheckDirector(p.Store, o.nowTime())
	switch {
	case err != nil:
		return err
	case !chk.Checked:
		return refused
	case chk.Check.State != DirectorCheckEnded:
		reg, check := chk.Registration, chk.Check
		return &DirectorAttachedError{Current: &reg, Err: directorStartAttachedErr(&reg), Check: &check}
	}
	if o.directorStartAfterCheck != nil {
		o.directorStartAfterCheck()
	}
	notice, err := o.cleanUpEndedDirector(p.Store, chk, p.Who)
	if err != nil {
		return err
	}
	if p.Notice != nil {
		_, _ = fmt.Fprintln(p.Notice, notice)
	}
	return nil
}

// cleanUpEndedDirector is start's cleanup of the registration chk checked
// (plan 2026-10-18-org-director-liveness, D6): chk is the CheckDirector of
// checkDirectorStart's first look, Checked and ended, and store the ledger's
// director.jsonl. It does to chk.Registration, the registration as the check
// read it (version A), what StopDirector without Force does after its read,
// and it never reads the registration again to decide what to close, so a
// pane, a workspace, or a pid recorded after that read is left alone (the
// race of two starts in Codex's advisory on the plan). For a launch herdr
// that is closeHerdrDirector: the refusals from inside the director's pane
// or workspace, the check of herdr's labels, C-c, and the closes. For a
// launch cmd it is stopDirectorProcess, which sends SIGTERM only to the
// recorded pid while ps reads the recorded start time for it, so nothing
// goes to a process the check found gone, nor to a starting registration,
// which has no pid. The refusal of stopCmdDirector from inside the director
// (RALPH_DIRECTOR_TOKEN) is not made: the director has ended, so ending its
// process cannot end this command. Then MarkStoppedAtVersion records
// stopped, with detached first as the registration is current, only while
// director.jsonl still holds version A. There is no Force here: what is not
// confirmed or not closed is left to `ralph org director stop`.
//
// Once that record is written it returns the one line that says what it did
// (directorCleanupNotice) and a nil error. Otherwise it has written nothing,
// and the error is a *DirectorCleanupError of verb start
// (newDirectorCleanupError, or newDirectorCmdCleanupError for a launch cmd)
// whose Err wraps ErrDirectorAttached and names the registration and how it
// ended, whose CleanupErr says what stopped the cleanup (a refusal from
// inside the director's pane or workspace, a label that is not this
// launch's, a close that failed, a process whose end is not confirmed,
// ErrDirectorChangedSinceCheck, or another record that failed), and whose
// StopCommand is the stop that finishes it, with --force for a label that is
// not this launch's, as a stop's error names it.
func (o *Org) cleanUpEndedDirector(store *DirectorStore, chk DirectorCheckResult, who DirectorWho) (string, error) {
	reg := chk.Registration
	cause := fmt.Errorf("org: director: start: %w: %s, which start found ended (%s) when it checked it just now, and start could not clean it up, so it started no new director",
		ErrDirectorAttached, reg.describe(), chk.Check.How)
	if reg.CmdLaunch() {
		stop := o.stopDirectorProcess(reg)
		if stop.outcome != directorProcessStopEnded {
			return "", newDirectorCmdCleanupError(reg, cause, fmt.Errorf("%w: %w", ErrDirectorProcessNotEnded, stop.err), false)
		}
		if _, err := store.MarkStoppedAtVersion(reg, chk.version, who); err != nil {
			return "", newDirectorCmdCleanupError(reg, cause, &directorRecordStopError{id: reg.ID, err: err}, false)
		}
		return directorCmdCleanupNotice(reg, chk.Check.How, stop), nil
	}
	res, closed, err := o.closeHerdrDirector(reg)
	switch {
	case err != nil:
		return "", newDirectorCleanupError("start", reg, directorClose{}, cause, err, false)
	case closed.err != nil:
		return "", newDirectorCleanupError("start", reg, closed, cause, closed.err, false)
	}
	if _, err := store.MarkStoppedAtVersion(reg, chk.version, who); err != nil {
		return "", newDirectorCleanupError("start", reg, closed, cause, &directorRecordStopError{id: reg.ID, err: err}, false)
	}
	return directorHerdrCleanupNotice(res, chk.Check.How), nil
}

// directorHerdrCleanupNotice is the line of cleanUpEndedDirector for res, a
// launch herdr it closed and recorded stopped, which had ended by how: its
// notes on the C-c, the pane, and the workspace, escaped (a failed C-c's note
// holds herdr's error text), and, when no workspace id was recorded, the
// label of a workspace start may have made before it could record it.
func directorHerdrCleanupNotice(res DirectorStopResult, how string) string {
	line := fmt.Sprintf("cleaned up %s, which had ended (%s): %s, %s, %s, and recorded it stopped", res.Registration.ID, how,
		PrintableInboxText(res.CtrlCNote, false), PrintableInboxText(res.PaneNote, false), PrintableInboxText(res.WorkspaceNote, false))
	if label := res.UnrecordedWorkspaceLabel; label != "" {
		line += "; herdr may still have a workspace labelled " + label + " that its start made before it could record it (herdr workspace list)"
	}
	return line
}

// directorCmdCleanupNotice is the line of cleanUpEndedDirector for reg, a
// launch cmd it recorded stopped, which had ended by how, after stop: what
// was done to the process, whether anything was sent to it, and the terminal
// launch_cmd opened (or may have opened, before a check-in), which ralph
// never closes.
func directorCmdCleanupNotice(reg DirectorRegistration, how string, stop directorProcessStop) string {
	line := fmt.Sprintf("cleaned up %s, which had ended (%s): %s, and recorded it stopped; ", reg.ID, how, PrintableInboxText(stop.note, false))
	if !stop.signalled {
		line += "nothing was sent to its process, and "
	}
	name := DirectorAgentName(reg.Token)
	if reg.PID > 0 {
		return line + "the terminal that [org.director].launch_cmd opened (name " + name + ") is left open for you to close"
	}
	return line + "the terminal that [org.director].launch_cmd may have opened (name " + name + ") is left open for you to close, " +
		"and a launch script that checks in from it now is refused and starts no agent"
}
