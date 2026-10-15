package org

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// StopDirector of stage 8a-2 (plan
// docs/plans/active/2026-10-15-org-director-start.md, D4 and D9) and the
// closing it shares with StartDirector's cleanup: the checks of herdr's
// labels before anything is sent, the C-c and the closes, and the
// *DirectorCleanupError of a start or a stop that could not finish them.
// StartDirector and the DirectorStore writers of a headless registration
// are in director_start.go.

// The refusals of StopDirector that the CLI tells apart, each wrapped by an
// error that names the case.
var (
	// ErrDirectorNotHeadless is wrapped, with ErrDirectorNoOpenHeadless, by
	// StopDirector's refusal of a registration that a session promoted with
	// `ralph org director attach`: it has no herdr pane of ralph's to close,
	// and `ralph org director detach --id` ends it.
	ErrDirectorNotHeadless = errors.New("the registration was not started in herdr by ralph org director start")
	// ErrDirectorStopSelfPane: stop ran inside the director's own pane
	// (HERDR_PANE_ID is its recorded pane), and closing it would end the
	// command before it records the stop.
	ErrDirectorStopSelfPane = errors.New("run ralph org director stop from a terminal outside the director's pane")
	// ErrDirectorStopSelfWorkspace: stop ran inside the director's own
	// workspace (HERDR_WORKSPACE_ID is its recorded workspace, which herdr's
	// label confirmed as this launch's), and closing it would end the command
	// before it records the stop.
	ErrDirectorStopSelfWorkspace = errors.New("run ralph org director stop from a terminal outside the director's workspace")
	// ErrDirectorChangedDuringStop: a start still running recorded a
	// workspace or pane of the registration after stop read it, so stop did
	// not close it and recorded nothing (plan D14,
	// MarkStoppedIfUnchanged). Running stop again closes what start recorded.
	ErrDirectorChangedDuringStop = errors.New("a start in progress recorded a workspace or pane after stop read the registration")
)

// DirectorStopParams describes one StopDirector call: Store is the
// director.jsonl of the ledger, ID the registration to stop, and Who who
// writes the stop. Force records the stop even when the recorded pane and
// workspace cannot be confirmed as this launch's or closed (plan D11).
type DirectorStopParams struct {
	Store *DirectorStore
	ID    string
	Force bool
	Who   DirectorWho
}

// DirectorStopResult is what StopDirector did. Registration is the open
// headless registration as read before the stop. WasCurrent says it was the
// current registration, so detached was written with stopped and no
// director is attached now. CtrlCNote, PaneNote, and WorkspaceNote say what
// happened to the pane and the workspace. UnrecordedWorkspaceLabel is set
// when no workspace id was recorded: herdr may still hold a workspace with
// this label that start made before it could record the id.
//
// Forced says Force recorded the stop although the pane or the workspace
// may still be open: stopped carries force, the notes say what was left
// open, CloseErr says why (herdr's labels did not confirm a recorded id, a
// get failed or timed out, or a close failed), and HerdrCommands find this
// launch's workspace by its label to close it by hand. They name no recorded
// id: nothing checks it again once the stop is recorded, and a herdr that
// renumbered its ids can give it to another launch. All three are unset when
// Force changed nothing.
type DirectorStopResult struct {
	Registration             DirectorRegistration
	WasCurrent               bool
	CtrlCNote                string
	PaneNote                 string
	WorkspaceNote            string
	UnrecordedWorkspaceLabel string
	Forced                   bool
	CloseErr                 error
	HerdrCommands            []string
}

// StopDirector closes the herdr pane and workspace of the open headless
// registration p.ID and records the stop (plan D4): current or detached,
// starting or running. It refuses, with nothing written and no herdr call,
// an empty or malformed id, an id no registration has, a registration a
// session promoted (ErrDirectorNotHeadless) or one already stopped
// (ErrDirectorStopped), both with ErrDirectorNoOpenHeadless, and a call from
// inside the director's own pane (ErrDirectorStopSelfPane).
//
// confirmDirectorResources checks herdr's labels for the recorded pane and
// workspace before anything is sent (D9): a label that is not this launch's
// sends no C-c and closes nothing. Once herdr's label confirmed the
// recorded workspace as this launch's, a call from inside it
// (HERDR_WORKSPACE_ID is its id) is refused with
// ErrDirectorStopSelfWorkspace, with no C-c, no close, and nothing written.
// The id alone does not refuse: after herdr renumbered its ids it can name
// someone else's workspace, and that is the label mismatch instead. Then it
// sends C-c to the pane, closes it, and closes the workspace
// (closeConfirmedDirectorResources); a pane or workspace herdr no longer
// knows counts as closed, and one not recorded is skipped. When any of that
// does not finish, it returns a *DirectorCleanupError and writes nothing.
// Otherwise MarkStoppedIfUnchanged writes stopped, with detached when the
// registration is current, after it checks under the lock that the
// workspace and pane ids recorded now are the ones read above (plan D14).
// The read takes no lock, so a start still running can record its workspace
// or pane after it, and this stop did not close that one: then nothing is
// written, and the *DirectorCleanupError wraps ErrDirectorChangedDuringStop,
// says what this stop closed, and names the stop to run again, which closes
// what start recorded. On a *DirectorCleanupError the result still carries
// Registration and the notes.
//
// With p.Force (plan D11), a label that is not this launch's, a get that
// fails or times out, and a close that fails do not end the stop: it sends
// no C-c and no close that it has not sent yet (a pane it closed stays
// closed), records the stop with MarkForceStopped (stopped with force, and
// detached first while the registration is current; it compares no ids),
// and returns Forced.
// That is the way out after herdr lost its state and renumbered its ids:
// the recorded ids can name another launch's pane and workspace (AC5), and
// without Force every stop refuses on them while the registration stays
// open. As with the seat's `ralph org stop --force` (stopSeatPane), nothing
// unconfirmed is closed. When the labels match and the closes go through,
// Force changes nothing: the stop is recorded without force and Forced is
// false. The refusals above, the pane and the workspace ones included, hold
// with Force too.
func (o *Org) StopDirector(p DirectorStopParams) (DirectorStopResult, error) {
	switch _, ok := directorIDNumber(p.ID); {
	case p.Store == nil:
		return DirectorStopResult{}, errors.New("org: director: stop: no director store was given")
	case p.ID == "":
		return DirectorStopResult{}, errors.New("org: director: stop needs the id of the registration (d<N>, such as d1)")
	case !ok:
		return DirectorStopResult{}, fmt.Errorf("org: director: stop: %q is not a registration id (d<N>, such as d1)", p.ID)
	}
	reg, err := p.Store.openHeadlessRegistration("stop "+p.ID, p.ID)
	if err != nil {
		return DirectorStopResult{}, err
	}
	if own := o.getenv(herdrPaneIDEnv); own != "" && own == reg.PaneID {
		return DirectorStopResult{}, fmt.Errorf("org: director: stop %s: HERDR_PANE_ID is its pane %q, and closing that pane would end this command before it records the stop: %w",
			p.ID, reg.PaneID, ErrDirectorStopSelfPane)
	}
	closed, gone := o.confirmDirectorResources(reg)
	if closed.err == nil && reg.WorkspaceID != "" && !gone.workspace && o.getenv(herdrWorkspaceIDEnv) == reg.WorkspaceID {
		return DirectorStopResult{}, fmt.Errorf("org: director: stop %s: HERDR_WORKSPACE_ID is its workspace %q, labelled %s, and closing that workspace would end this command before it records the stop: %w",
			p.ID, reg.WorkspaceID, DirectorWorkspaceLabel(reg.Token), ErrDirectorStopSelfWorkspace)
	}
	if closed.err == nil {
		closed = o.closeConfirmedDirectorResources(reg, closed, gone)
	}
	res := DirectorStopResult{Registration: reg, CtrlCNote: closed.ctrlC, PaneNote: closed.pane, WorkspaceNote: closed.workspace}
	if reg.WorkspaceID == "" {
		res.UnrecordedWorkspaceLabel = DirectorWorkspaceLabel(reg.Token)
	}
	return recordDirectorStop(p, res, closed)
}

// recordDirectorStop is the end of StopDirector once closed is what it did
// to the pane and the workspace of res.Registration: it records the stop,
// with force when closed did not finish and p.Force is set, and fills in
// WasCurrent and, for a forced stop, Forced, CloseErr, and HerdrCommands.
// Without p.Force a close that did not finish records nothing. A stop whose
// closes finished is recorded by MarkStoppedIfUnchanged against
// res.Registration, the registration as StopDirector read it (plan D14); a
// forced one by MarkForceStopped, which compares no ids. A record that fails
// is a *DirectorCleanupError; after a forced stop its StopCommand carries
// --force, since the retry meets what this one met. For a start that
// recorded an id after the read (ErrDirectorChangedDuringStop), its
// CleanupErr says what this stop closed, and its StopCommand is the stop to
// run again.
func recordDirectorStop(p DirectorStopParams, res DirectorStopResult, closed directorClose) (DirectorStopResult, error) {
	reg := res.Registration
	forced := closed.err != nil
	if forced && !p.Force {
		return res, newDirectorCleanupError("stop", reg, closed, nil, closed.err, false)
	}
	var wasCurrent bool
	var err error
	if forced {
		wasCurrent, err = p.Store.MarkForceStopped(p.ID, p.Who)
	} else {
		wasCurrent, err = p.Store.MarkStoppedIfUnchanged(reg, p.Who)
	}
	switch {
	case err != nil && forced:
		e := newDirectorCleanupError("stop", reg, closed, nil, errors.Join(closed.err, fmt.Errorf("record %s stopped: %w", p.ID, err)), false)
		e.StopCommand = directorStopCommand(reg.ID, true)
		return res, e
	case errors.Is(err, ErrDirectorChangedDuringStop):
		return res, newDirectorCleanupError("stop", reg, closed, nil,
			fmt.Errorf("record %s stopped: %w; this stop closed only what it had read (%s, %s, %s)", p.ID, err, closed.ctrlC, closed.pane, closed.workspace), false)
	case err != nil:
		return res, newDirectorCleanupError("stop", reg, closed, nil, fmt.Errorf("record %s stopped: %w", p.ID, err), false)
	}
	res.WasCurrent = wasCurrent
	if forced {
		res.Forced, res.CloseErr = true, closed.err
		res.HerdrCommands = directorFindWorkspaceCommands(DirectorWorkspaceLabel(reg.Token))
	}
	return res, nil
}

// openHeadlessRegistration returns the open headless registration id, read
// without the lock (MarkStopped decides again under it), or verb's refusal:
// a registration a session promoted wraps ErrDirectorNotHeadless as well as
// ErrDirectorNoOpenHeadless; an unknown or already stopped id is
// openHeadless's error. A missing director.jsonl reads as no registration.
func (s *DirectorStore) openHeadlessRegistration(verb, id string) (DirectorRegistration, error) {
	raw, err := s.readRaw()
	if err != nil {
		return DirectorRegistration{}, err
	}
	fold := foldDirector(raw)
	if reg := fold.regs[id]; reg != nil && !reg.Headless() {
		return DirectorRegistration{}, fmt.Errorf("org: director: %s: %w: %w: %s", verb, ErrDirectorNoOpenHeadless, ErrDirectorNotHeadless, reg.describe())
	}
	reg, err := fold.openHeadless(verb, id)
	if err != nil {
		return DirectorRegistration{}, err
	}
	return *reg, nil
}

// DirectorCleanupError is a start or a stop (Verb) that could not close the
// herdr pane and workspace of registration ID, or closed them and could not
// record the stop. Err is why StartDirector failed (nil for StopDirector);
// CleanupErr is why the cleanup did not finish. Unconfirmed means herdr's
// labels did not confirm the recorded ids as this launch's (TabLabel and
// WorkspaceLabel), or could not be read, so nothing was sent to them and
// they may name another launch's pane. Detached means start wrote detached
// for ID (the registration was still current), so ID is no longer current
// but stays open; a stop writes nothing when it fails. HerdrCommands are what
// a human can run to close what was left open (for Unconfirmed, finding the
// workspace by its label, never closing a recorded id), and StopCommand
// records the stop afterwards; the CLI adds its --state-dir to it. For
// Unconfirmed it carries --force (plan D11): the recorded ids may go on
// naming another launch's pane and workspace, and a stop without it would
// refuse on them again. A stop that closed what it read and then found that a
// start recorded a workspace or pane meanwhile (CleanupErr wraps
// ErrDirectorChangedDuringStop, plan D14) has nothing left open of what it
// read, so HerdrCommands is empty, and StopCommand is the stop to run again,
// which closes what start recorded.
type DirectorCleanupError struct {
	Verb           string
	ID             string
	WorkspaceID    string
	PaneID         string
	WorkspaceLabel string
	TabLabel       string
	Err            error
	CleanupErr     error
	Unconfirmed    bool
	Detached       bool
	HerdrCommands  []string
	StopCommand    string
}

func (e *DirectorCleanupError) Error() string {
	var b strings.Builder
	if e.Err != nil {
		fmt.Fprintf(&b, "%v; its cleanup did not finish: %v", e.Err, e.CleanupErr)
	} else {
		fmt.Fprintf(&b, "org: director: %s %s: %v", e.Verb, e.ID, e.CleanupErr)
	}
	switch {
	case errors.Is(e.CleanupErr, ErrDirectorWriteNotRestored):
		b.WriteString("; director.jsonl may hold part of the write")
	case e.Detached:
		fmt.Fprintf(&b, "; %s was recorded detached only, so it is no longer the current registration but stays open", e.ID)
	case e.Err != nil:
		b.WriteString("; nothing more was recorded")
	default:
		b.WriteString("; nothing was recorded")
	}
	then := "record the stop with "
	if len(e.HerdrCommands) > 0 {
		fmt.Fprintf(&b, "; close it by hand: %s", strings.Join(e.HerdrCommands, ", then "))
		then = "afterwards record the stop with "
	}
	switch {
	case e.StopCommand == "":
	case errors.Is(e.CleanupErr, ErrDirectorChangedDuringStop):
		b.WriteString("; run " + e.StopCommand + " again to close what start recorded")
	default:
		b.WriteString("; " + then + e.StopCommand)
	}
	return b.String()
}

func (e *DirectorCleanupError) Unwrap() []error {
	if e.Err == nil {
		return []error{e.CleanupErr}
	}
	return []error{e.Err, e.CleanupErr}
}

// directorClose is what closeDirectorResources did: the notes for the C-c,
// the pane, and the workspace; err, why it did not finish (nil when both are
// closed or were never recorded); unconfirmed, when herdr's labels did not
// confirm a recorded id; and which recorded ids may still be open.
type directorClose struct {
	ctrlC, pane, workspace  string
	err                     error
	unconfirmed             bool
	paneOpen, workspaceOpen bool
}

// closeDirectorResources closes the herdr pane and workspace recorded for
// reg, the closing shared by StopDirector and StartDirector's cleanup. It
// first checks both (confirmDirectorResources); when either is not
// confirmed, or its get fails, nothing is sent to either. Then it sends C-c
// to the pane, closes it, and closes the workspace
// (closeConfirmedDirectorResources). StopDirector calls the two halves
// itself, to refuse a call from inside the confirmed workspace in between.
func (o *Org) closeDirectorResources(reg DirectorRegistration) directorClose {
	c, gone := o.confirmDirectorResources(reg)
	if c.err != nil {
		return c
	}
	return o.closeConfirmedDirectorResources(reg, c, gone)
}

// directorGone is what confirmDirectorResources found herdr no longer has:
// the recorded pane, the recorded workspace.
type directorGone struct {
	pane, workspace bool
}

// confirmDirectorResources checks the herdr pane and workspace recorded for
// reg, with nothing sent: the pane with confirmSeatPane (its tab labelled
// DirectorTabLabel and its workspace DirectorWorkspaceLabel of reg's token),
// the workspace with confirmOrgWorkspace (labelled DirectorWorkspaceLabel),
// both with directorLabelWords, so their errors name the labels and not a
// seat or an org_id. It returns the directorClose to go on with, whose err
// is set (notConfirmed) when a recorded id is not confirmed or its get
// fails, and which recorded ids herdr no longer knows. One not recorded is
// skipped. Every get runs under its own driverCallTimeout.
func (o *Org) confirmDirectorResources(reg DirectorRegistration) (directorClose, directorGone) {
	tabLabel, wsLabel := DirectorTabLabel(reg.Token), DirectorWorkspaceLabel(reg.Token)
	c := directorClose{ctrlC: "ctrl_c=skipped", pane: "pane=none recorded", workspace: "workspace=none recorded",
		paneOpen: reg.PaneID != "", workspaceOpen: reg.WorkspaceID != ""}
	var gone directorGone
	if reg.PaneID != "" {
		paneGone, err := o.confirmSeatPane(wsLabel, tabLabel, reg.PaneID, directorLabelWords)
		if err != nil {
			return c.notConfirmed(fmt.Errorf("pane %q is not confirmed as %s's (a tab labelled %s in a workspace labelled %s), so nothing was sent to it: %w",
				reg.PaneID, reg.ID, tabLabel, wsLabel, err)), gone
		}
		gone.pane = paneGone
	}
	if reg.WorkspaceID != "" {
		wsGone, err := o.confirmOrgWorkspace(wsLabel, reg.WorkspaceID, directorLabelWords)
		if err != nil {
			return c.notConfirmed(fmt.Errorf("workspace %q is not confirmed as %s's (labelled %s), so nothing was sent to it: %w",
				reg.WorkspaceID, reg.ID, wsLabel, err)), gone
		}
		gone.workspace = wsGone
	}
	return c, gone
}

// closeConfirmedDirectorResources closes what confirmDirectorResources
// confirmed for reg, going on from its c and gone: it sends C-c to the pane
// (a failed C-c is a note), closes it, and closes the workspace, stopping at
// the first close that fails. A recorded id herdr no longer knows counts as
// closed, and one not recorded is skipped. Every herdr call runs under its
// own driverCallTimeout.
func (o *Org) closeConfirmedDirectorResources(reg DirectorRegistration, c directorClose, gone directorGone) directorClose {
	if reg.PaneID != "" && !o.closeDirectorPane(&c, reg.PaneID, gone.pane) {
		return c
	}
	if reg.WorkspaceID != "" {
		o.closeDirectorWorkspace(&c, reg.WorkspaceID, gone.workspace)
	}
	return c
}

// notConfirmed is the directorClose of a recorded id herdr's labels did not
// confirm: nothing sent, both left as they are.
func (c directorClose) notConfirmed(err error) directorClose {
	c.err, c.unconfirmed = err, true
	c.ctrlC = "ctrl_c=skipped: not confirmed"
	if c.paneOpen {
		c.pane = "pane=not closed: not confirmed"
	}
	if c.workspaceOpen {
		c.workspace = "workspace=not closed: not confirmed"
	}
	return c
}

// closeDirectorPane sends C-c to paneID and closes it (gone: herdr already
// had no such pane), and reports whether it is closed; when not, c.err says
// why and the workspace is left as it is.
func (o *Org) closeDirectorPane(c *directorClose, paneID string, gone bool) bool {
	if gone {
		c.pane, c.paneOpen = "pane=already closed", false
		return true
	}
	c.ctrlC = "ctrl_c=ok"
	if err := o.callWithTimeout(func(ctx context.Context) error {
		return o.Herdr.PaneSendKeys(ctx, paneID, "C-c")
	}); err != nil {
		c.ctrlC = fmt.Sprintf("ctrl_c=failed: %v", err)
	}
	err := o.callWithTimeout(func(ctx context.Context) error {
		return o.Herdr.PaneClose(ctx, paneID)
	})
	switch {
	case err == nil:
		c.pane, c.paneOpen = "pane=closed", false
		return true
	case isNotFound(err):
		c.pane, c.paneOpen = "pane=already closed", false
		return true
	}
	c.pane, c.err = fmt.Sprintf("pane=close failed: %v", err), fmt.Errorf("close pane %q: %w", paneID, err)
	if c.workspaceOpen {
		c.workspace = "workspace=not closed: the pane was left open"
	}
	return false
}

// closeDirectorWorkspace closes workspaceID (gone: herdr already had no such
// workspace); when it cannot, c.err says why.
func (o *Org) closeDirectorWorkspace(c *directorClose, workspaceID string, gone bool) {
	if gone {
		c.workspace, c.workspaceOpen = "workspace=already closed", false
		return
	}
	err := o.callWithTimeout(func(ctx context.Context) error {
		return o.Herdr.WorkspaceClose(ctx, workspaceID)
	})
	switch {
	case err == nil:
		c.workspace, c.workspaceOpen = "workspace=closed", false
	case isNotFound(err):
		c.workspace, c.workspaceOpen = "workspace=already closed", false
	default:
		c.workspace, c.err = fmt.Sprintf("workspace=close failed: %v", err), fmt.Errorf("close workspace %q: %w", workspaceID, err)
	}
}

// newDirectorCleanupError is the *DirectorCleanupError of verb (start or
// stop) for reg after closed, with the herdr commands for what closed left
// open.
func newDirectorCleanupError(verb string, reg DirectorRegistration, closed directorClose, cause, cleanupErr error, detached bool) *DirectorCleanupError {
	e := &DirectorCleanupError{
		Verb: verb, ID: reg.ID, WorkspaceID: reg.WorkspaceID, PaneID: reg.PaneID,
		WorkspaceLabel: DirectorWorkspaceLabel(reg.Token), TabLabel: DirectorTabLabel(reg.Token),
		Err: cause, CleanupErr: cleanupErr, Unconfirmed: closed.unconfirmed, Detached: detached,
		StopCommand: directorStopCommand(reg.ID, closed.unconfirmed),
	}
	switch {
	case closed.unconfirmed:
		// The recorded ids may name another launch's pane and workspace, so
		// the human finds this launch's workspace by its label instead.
		e.HerdrCommands = directorFindWorkspaceCommands(e.WorkspaceLabel)
	default:
		if closed.paneOpen {
			e.HerdrCommands = append(e.HerdrCommands, "herdr pane close "+directorShellWord(reg.PaneID))
		}
		if closed.workspaceOpen {
			e.HerdrCommands = append(e.HerdrCommands, "herdr workspace close "+directorShellWord(reg.WorkspaceID))
		}
	}
	return e
}

// directorFindWorkspaceCommands are the herdr commands that find the
// workspace labelled label and close it, for a pane and workspace whose
// recorded ids herdr's labels did not confirm or that nothing will check
// again: they name the label, never a recorded id.
func directorFindWorkspaceCommands(label string) []string {
	return []string{"herdr workspace list", "herdr workspace close <the id of the workspace labelled " + label + ">"}
}

// directorStopCommand is the `ralph org director stop` command that records
// the stop of id, with --force when force (the CLI adds its --state-dir).
func directorStopCommand(id string, force bool) string {
	cmd := "ralph org director stop --id " + id
	if force {
		cmd += " --force"
	}
	return cmd
}

// directorShellWordPattern is a herdr id that needs no quoting in a shell
// command.
var directorShellWordPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]+$`)

// directorShellWord is s as one shell word for a printed herdr command: as
// it is when it needs no quoting, else shellQuote. A recorded id comes from
// director.jsonl, which a hand edit can fill with anything.
func directorShellWord(s string) string {
	if directorShellWordPattern.MatchString(s) {
		return s
	}
	return shellQuote(s)
}
