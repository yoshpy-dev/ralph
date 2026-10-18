package org

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// The headless director of stage 8a-2 (plan
// docs/plans/active/2026-10-15-org-director-start.md): the DirectorStore
// writers of a headless registration with its launch token and the herdr
// names built from it, and the org-layer verb StartDirector, which launches
// a headless director in a herdr workspace of its own. StopDirector, which
// closes it again, and the closing that start's cleanup shares with it are
// in director_stop.go. Neither verb writes a manifest line or a model
// receipt (D2, D8): director.jsonl is the only record. The fold that reads
// these events back is in director.go. With [org.director].launch_cmd
// (plan 2026-10-16-org-director-launch-cmd) StartDirector opens the human's
// terminal instead of herdr; that path (startCmd) and its cleanup are in
// director_cmd.go.

// DirectorHeadlessAttach is what AttachHeadless records: the launch
// (DirectorLaunchHerdr, or DirectorLaunchCmd for a start with
// [org.director].launch_cmd; empty means herdr), driver, model, permission
// mode (autonomous, edits, or guarded), and wake ([org.director].wake as
// start read it; plan 2026-10-18-org-director-wake, D3) of the headless
// director `ralph org director start` launches. StarterPID and
// StarterStartTime are the starter: the pid of the start process and that
// pid's start time as ReadDirectorProcess read it (plan
// 2026-10-18-org-director-liveness, D1), both set, or both empty when start
// could not read them. User, Cwd, and HerdrPane describe the caller.
type DirectorHeadlessAttach struct {
	Launch           string
	Driver           string
	Model            string
	PermissionMode   string
	Wake             []string
	StarterPID       int
	StarterStartTime string
	User             string
	Cwd              string
	HerdrPane        bool
}

// DirectorWho is who writes a headless registration's later events
// (RecordWorkspace, RecordPane, MarkLaunched, MarkCmdLaunched, MarkStopped,
// MarkStoppedIfUnchanged, MarkStoppedAtVersion, MarkForceStopped): USER, the
// current directory, and whether HERDR_PANE_ID is set. The store does not
// read the environment.
type DirectorWho struct {
	User      string
	Cwd       string
	HerdrPane bool
}

// directorTokenPattern is the shape of a launch token as newDirectorToken
// makes it: 12 lowercase hex digits. The fold ignores a headless attached
// whose token does not match, so a token read from director.jsonl is safe
// to put in a herdr name or label and to compare a label with.
var directorTokenPattern = regexp.MustCompile(`^[0-9a-f]{12}$`)

// newDirectorToken returns a new launch token: 6 bytes from crypto/rand in
// lowercase hex.
func newDirectorToken() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("make a launch token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// DirectorAgentName is the herdr agent name of the headless director
// launched with token: "director_" and the token, 21 characters, within
// herdr's agent-name rule `^[a-z][a-z0-9_-]{0,31}$` (maxHerdrAgentNameLen).
// herdr's agent names are unique across its server, so the name holds the
// token of the launch, not the ledger's d<N>, which the first director of
// every repo shares (D9).
func DirectorAgentName(token string) string {
	return "director_" + token
}

// DirectorTabLabel is the label of the herdr tab that holds the headless
// director launched with token; it is the agent name.
func DirectorTabLabel(token string) string {
	return DirectorAgentName(token)
}

// DirectorWorkspaceLabel is the label of the herdr workspace made for the
// headless director launched with token. An org's workspace label and an org
// seat's tab label hold no "_" (identifierPattern), so neither label of a
// director can be an org's.
func DirectorWorkspaceLabel(token string) string {
	return "ralph_director_" + token
}

// AttachHeadless records the registration of a headless director, which
// `ralph org director start` does before any herdr call (D3), and returns
// it: state starting, a new id, and a new launch token. Under the director
// lock it reads the file again; while any registration is current (a
// promoted session's, or a headless one in any state) it returns
// ErrDirectorAttached, naming that registration, and writes nothing. It has
// no replace: start never ends another registration. Otherwise it appends
// one attached with h.Launch (herdr when empty), the token, h.Model,
// h.PermissionMode, and h.Wake, and no wake_target: the target of a
// headless registration follows from its launch (ResolvedWakeTarget). The
// new id is chosen as Attach chooses it, and a failed write ends one of the
// two ways Attach's does. A launch other than herdr and cmd, a driver other
// than claude and codex, an empty model, a permission mode other than
// autonomous, edits, and guarded, and a wake that config.ValidateDirectorWake
// refuses (an empty one included: start always passes the wake it read) are
// refused before anything is read or written.
//
// The attached also carries the starter, h.StarterPID and h.StarterStartTime
// (trimmed), when both are given (plan 2026-10-18-org-director-liveness,
// D1), and neither when neither is. A starter pid without a start time, a
// start time without a pid, and a pid that is not positive are refused
// before anything is read or written, like the input above.
func (s *DirectorStore) AttachHeadless(h DirectorHeadlessAttach) (DirectorRegistration, error) {
	const verb = "start"
	launch := h.Launch
	if launch == "" {
		launch = DirectorLaunchHerdr
	}
	if launch != DirectorLaunchHerdr && launch != DirectorLaunchCmd {
		return DirectorRegistration{}, fmt.Errorf("org: director: %s: unknown launch %q: use %s or %s", verb, h.Launch, DirectorLaunchHerdr, DirectorLaunchCmd)
	}
	if h.Driver != DirectorDriverClaude && h.Driver != DirectorDriverCodex {
		return DirectorRegistration{}, fmt.Errorf("org: director: %s: unknown driver %q: use %s or %s", verb, h.Driver, DirectorDriverClaude, DirectorDriverCodex)
	}
	if h.Model == "" {
		return DirectorRegistration{}, fmt.Errorf("org: director: %s: the model is empty", verb)
	}
	switch h.PermissionMode {
	case PermissionModeAutonomous, PermissionModeEdits, PermissionModeGuarded:
	default:
		return DirectorRegistration{}, fmt.Errorf("org: director: %s: unknown permission mode %q: use %s, %s, or %s",
			verb, h.PermissionMode, PermissionModeAutonomous, PermissionModeEdits, PermissionModeGuarded)
	}
	if err := config.ValidateDirectorWake(h.Wake); err != nil {
		return DirectorRegistration{}, fmt.Errorf("org: director: %s: %w", verb, err)
	}
	starterStartTime := strings.TrimSpace(h.StarterStartTime)
	switch {
	case h.StarterPID == 0 && starterStartTime == "":
		// No starter: start could not read its own start time (D1).
	case h.StarterPID < 0:
		return DirectorRegistration{}, fmt.Errorf("org: director: %s: the starter pid %d is not a positive number", verb, h.StarterPID)
	case h.StarterPID == 0:
		return DirectorRegistration{}, fmt.Errorf("org: director: %s: the starter start time %q is given without its pid", verb, starterStartTime)
	case starterStartTime == "":
		return DirectorRegistration{}, fmt.Errorf("org: director: %s: the starter pid %d is given without its start time", verb, h.StarterPID)
	}
	newToken := newDirectorToken
	if s.token != nil {
		newToken = s.token
	}
	token, err := newToken()
	if err != nil {
		return DirectorRegistration{}, fmt.Errorf("org: director: %s: %w", verb, err)
	}
	if !directorTokenPattern.MatchString(token) {
		return DirectorRegistration{}, fmt.Errorf("org: director: %s: launch token %q is not 12 lowercase hex digits", verb, token)
	}
	var reg DirectorRegistration
	err = s.withLock(func() error {
		raw, err := s.readRaw()
		if err != nil {
			return err
		}
		fold := foldDirector(raw)
		if cur := fold.current; cur != nil {
			return directorStartAttachedErr(cur)
		}
		next, err := fold.nextID()
		if err != nil {
			return err
		}
		ev := DirectorEvent{
			TS: s.now(), ID: next, Event: DirectorEventAttached,
			Driver: h.Driver, Wake: slices.Clone(h.Wake),
			Launch: launch, Token: token, Model: h.Model, PermissionMode: h.PermissionMode,
			StarterPID: h.StarterPID, StarterStartTime: starterStartTime,
			User: h.User, Cwd: h.Cwd, HerdrPane: h.HerdrPane,
		}
		if err := s.appendLocked(raw, ev); err != nil {
			return directorAppendFailed(verb, nil, err)
		}
		reg = registrationOf(ev)
		return nil
	})
	if err != nil {
		return DirectorRegistration{}, err
	}
	return reg, nil
}

// directorStartAttachedErr is the ErrDirectorAttached refusal of start while
// cur is the current registration: AttachHeadless's under the lock, and the
// same words from checkDirectorStart's first look.
func directorStartAttachedErr(cur *DirectorRegistration) error {
	return fmt.Errorf("org: director: start: %w: %s", ErrDirectorAttached, cur.describe())
}

// RecordWorkspace appends workspace_created with workspaceID, the herdr
// workspace `ralph org director start` made, for the open headless
// registration id, whether it is still current or was detached meanwhile (a
// human's attach --replace or detach --force while start runs). With no
// open headless registration id it returns ErrDirectorNoOpenHeadless, and
// for one with launch cmd, which has no herdr workspace,
// ErrDirectorNotHerdrLaunch; when id already has a workspace recorded it
// refuses too, since the fold keeps the first one. None of them writes
// anything. An id not of the form d<N> and an empty workspaceID are refused
// before anything is read, and a missing director.jsonl is refused without
// creating the state dir or the lock. A failed write ends one of the two
// ways Attach's does.
func (s *DirectorStore) RecordWorkspace(id, workspaceID string, who DirectorWho) error {
	return s.recordHerdrID(id, DirectorEventWorkspaceCreated, workspaceID, who)
}

// RecordPane is RecordWorkspace for the pane: it appends pane_created with
// paneID, the pane of the herdr tab `ralph org director start` made.
func (s *DirectorStore) RecordPane(id, paneID string, who DirectorWho) error {
	return s.recordHerdrID(id, DirectorEventPaneCreated, paneID, who)
}

// recordHerdrID is RecordWorkspace (kind workspace_created) and RecordPane
// (kind pane_created) with value as the herdr id.
func (s *DirectorStore) recordHerdrID(id, kind, value string, who DirectorWho) error {
	what := "workspace"
	if kind == DirectorEventPaneCreated {
		what = "pane"
	}
	if _, ok := directorIDNumber(id); !ok {
		return fmt.Errorf("org: director: record the %s: %q is not a registration id (d<N>, such as d1)", what, id)
	}
	verb := "record the " + what + " of " + id
	if value == "" {
		return fmt.Errorf("org: director: %s: the %s id is empty", verb, what)
	}
	if _, err := os.Stat(s.path); errors.Is(err, fs.ErrNotExist) {
		return directorNoRegistration(verb, id)
	}
	return s.withLock(func() error {
		raw, err := s.readRaw()
		if err != nil {
			return err
		}
		reg, err := foldDirector(raw).openHerdrHeadless(verb, id)
		if err != nil {
			return err
		}
		ev := DirectorEvent{TS: s.now(), ID: id, Event: kind, User: who.User, Cwd: who.Cwd, HerdrPane: who.HerdrPane}
		recorded := reg.WorkspaceID
		if kind == DirectorEventPaneCreated {
			recorded, ev.PaneID = reg.PaneID, value
		} else {
			ev.WorkspaceID = value
		}
		if recorded != "" {
			return fmt.Errorf("org: director: %s: %s already has %s %s recorded", verb, id, what, PrintableInboxText(recorded, false))
		}
		if err := s.appendLocked(raw, ev); err != nil {
			return directorAppendFailed(verb, nil, err)
		}
		return nil
	})
}

// MarkLaunched appends launched for id and returns true when id is still the
// current registration, a headless one with launch herdr in state starting;
// `ralph org director start` calls it once its herdr agent start went
// through. When the current registration is another one, or none (a human's
// attach --replace or detach --force came in while start ran), or id is not
// a launch herdr (a promoted session, or a launch cmd, whose launched is the
// check-in of its launch script, MarkCmdLaunched) or already stopped, it
// writes nothing and returns false with a nil error, and start closes its
// own pane and workspace (D3). A current id already running returns true and
// writes nothing. An id not of the form d<N> is refused before anything is
// read; a missing director.jsonl returns false without creating the state
// dir or the lock. A failed write ends one of the two ways Attach's does.
func (s *DirectorStore) MarkLaunched(id string, who DirectorWho) (stillCurrent bool, err error) {
	if _, ok := directorIDNumber(id); !ok {
		return false, fmt.Errorf("org: director: mark launched: %q is not a registration id (d<N>, such as d1)", id)
	}
	if _, err := os.Stat(s.path); errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	verb := "mark " + id + " launched"
	err = s.withLock(func() error {
		raw, err := s.readRaw()
		if err != nil {
			return err
		}
		cur := foldDirector(raw).current
		if cur == nil || cur.ID != id || !cur.HerdrLaunch() || cur.State == DirectorStateStopped {
			return nil
		}
		stillCurrent = true
		if cur.State == DirectorStateRunning {
			return nil
		}
		ev := DirectorEvent{TS: s.now(), ID: id, Event: DirectorEventLaunched, User: who.User, Cwd: who.Cwd, HerdrPane: who.HerdrPane}
		if err := s.appendLocked(raw, ev); err != nil {
			stillCurrent = false
			return directorAppendFailed(verb, nil, err)
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return stillCurrent, nil
}

// DirectorCmdCheckIn is what MarkCmdLaunched records: the registration ID
// and launch Token the launch script was written with, the PID it gives as
// its own ($$, which stays the agent's pid across its exec), the StartTime
// of that pid as ReadDirectorProcess read it, and Who runs the check-in.
// ReadStarter is how MarkCmdLaunched reads the start process that recorded
// the registration (plan 2026-10-18-org-director-liveness, D2); nil means
// ReadDirectorProcess. The CLI passes the reader it reads PID with.
type DirectorCmdCheckIn struct {
	ID          string
	Token       string
	PID         int
	StartTime   string
	Who         DirectorWho
	ReadStarter func(ctx context.Context, pid int) (DirectorProcess, error)
}

// directorCheckInStarterTimeout bounds MarkCmdLaunched's read of the start
// process that recorded the registration (plan
// 2026-10-18-org-director-liveness, D2): 10 s, the driver call timeout that
// bounds every other ps read of a director process (readDirectorProcess).
const directorCheckInStarterTimeout = defaultDriverCallTimeout

// MarkCmdLaunched is the check-in of a launch script (`ralph org director
// launched`, plan 2026-10-16-org-director-launch-cmd, D4): it appends
// launched with c.PID and c.StartTime (trimmed) for registration c.ID only
// when, under the director lock, c.ID is the current registration, has
// launch cmd, is still starting, and has the launch token c.Token. Then the
// script execs the agent. Every other case writes nothing and returns an
// error that wraps ErrDirectorCheckInRefused and says why, so the script
// exits without starting the agent: no registration has c.ID, c.Token is
// not its token, it has another launch, it is already stopped (start's
// cleanup or `ralph org director stop` came first, so the terminal opened
// too late), it is no longer the current registration (an attach --replace
// or a detach came in), or it is already running (a check-in came first).
// An id not of the form d<N>, a token not of 12 lowercase hex digits, a pid
// that is not positive, and an empty start time are refused the same way
// before anything is read, and a missing director.jsonl without creating the
// state dir or the lock. Whether c.PID is the caller's parent and alive is
// the CLI's check, not the store's. A failed write ends one of the two ways
// Attach's does.
//
// Once all of those pass, a registration with a starter recorded (the
// `ralph org director start` process that wrote its attached; plan
// 2026-10-18-org-director-liveness, D2) has that pid read with c.ReadStarter
// (nil: ReadDirectorProcess) under directorCheckInStarterTimeout, still
// under the lock, so the lock is held for as long as ps takes, within that
// bound. When the pid has no process, is a zombie, or has a start time other
// than the recorded one, that start has ended: nothing waits for the
// check-in or cleans up after it, and the agent of a terminal that opened
// that late must not start, so the check-in is refused the same way, with
// that reason. A read that fails, and a registration with no starter, accept
// the check-in as before, so a ps that cannot be read never keeps a director
// from starting.
func (s *DirectorStore) MarkCmdLaunched(c DirectorCmdCheckIn) error {
	verb := "check in " + c.ID
	refuse := func(format string, args ...any) error {
		return fmt.Errorf("org: director: %s: %w: %s", verb, ErrDirectorCheckInRefused, fmt.Sprintf(format, args...))
	}
	startTime := strings.TrimSpace(c.StartTime)
	switch _, ok := directorIDNumber(c.ID); {
	case !ok:
		verb = "check in"
		return refuse("%q is not a registration id (d<N>, such as d1)", c.ID)
	case !directorTokenPattern.MatchString(c.Token):
		return refuse("the token %q is not 12 lowercase hex digits", c.Token)
	case c.PID <= 0:
		return refuse("the pid %d is not a positive number", c.PID)
	case startTime == "":
		return refuse("the start time of pid %d is empty", c.PID)
	}
	if _, err := os.Stat(s.path); errors.Is(err, fs.ErrNotExist) {
		return refuse("no registration %s is recorded", c.ID)
	}
	return s.withLock(func() error {
		raw, err := s.readRaw()
		if err != nil {
			return err
		}
		fold := foldDirector(raw)
		reg := fold.regs[c.ID]
		// The token comes before the launch and the state, so a caller
		// without the token learns nothing more of the registration. A
		// promoted session's registration has no token, so no token (12 hex
		// digits, checked above) matches it.
		switch {
		case reg == nil:
			return refuse("no registration %s is recorded", c.ID)
		case reg.Token != c.Token:
			return refuse("the token is not the one %s was started with", c.ID)
		case !reg.CmdLaunch():
			return refuse("%s was not started with [org.director].launch_cmd", reg.describe())
		case reg.State == DirectorStateStopped:
			return refuse("%s is already recorded stopped (start's cleanup or a stop came first)", reg.describe())
		case reg != fold.current:
			return refuse("%s is no longer the current registration (an attach --replace or a detach came in)", reg.describe())
		case reg.State != DirectorStateStarting:
			return refuse("%s has already checked in", reg.describe())
		}
		if how := directorCheckInStarterEnded(*reg, c.ReadStarter); how != "" {
			return refuse("the ralph org director start that recorded %s is no longer running (%s), so no start waits for this check-in: "+
				"run ralph org director start again", reg.describe(), how)
		}
		ev := DirectorEvent{
			TS: s.now(), ID: c.ID, Event: DirectorEventLaunched, PID: c.PID, StartTime: startTime,
			User: c.Who.User, Cwd: c.Who.Cwd, HerdrPane: c.Who.HerdrPane,
		}
		if err := s.appendLocked(raw, ev); err != nil {
			return directorAppendFailed(verb, nil, err)
		}
		return nil
	})
}

// directorCheckInStarterEnded is MarkCmdLaunched's look at the starter of
// reg (plan 2026-10-18-org-director-liveness, D2): it returns how the start
// process that recorded reg has ended (directorProcessEnded's words: no
// process has the pid, it is a zombie, or it is another process now), or ""
// when the check-in may go on. It reads nothing for a registration with no
// starter recorded. It reads the starter pid with read (nil:
// ReadDirectorProcess) under directorCheckInStarterTimeout, and a read that
// fails, or that answers a state ReadDirectorProcess never gives, counts as
// unreadable and returns "".
func directorCheckInStarterEnded(reg DirectorRegistration, read func(context.Context, int) (DirectorProcess, error)) string {
	if reg.StarterPID <= 0 || reg.StarterStartTime == "" {
		return ""
	}
	if read == nil {
		read = ReadDirectorProcess
	}
	ctx, cancel := context.WithTimeout(context.Background(), directorCheckInStarterTimeout)
	defer cancel()
	proc, err := read(ctx, reg.StarterPID)
	if err != nil {
		return ""
	}
	switch proc.State {
	case DirectorProcessAlive, DirectorProcessZombie, DirectorProcessGone:
	default:
		return ""
	}
	ended, how := directorProcessEnded(DirectorRegistration{PID: reg.StarterPID, StartTime: reg.StarterStartTime}, proc)
	if !ended {
		return ""
	}
	return how
}

// MarkStopped records that the pane and workspace of the open headless
// registration id are closed, which start's cleanup does once herdr has
// closed them (or never made them, or no longer knows them), and returns
// whether id was the current registration. It compares no recorded id:
// start's cleanup closes by the ids herdr handed it, which can include one
// it failed to record, and only start itself records ids for its
// registration. `ralph org director stop`, which read the registration
// before it closed anything, calls MarkStoppedIfUnchanged instead. For the
// current registration it appends detached and stopped as two lines of one
// write (appendLocked), so the registration ends with its stop; for one that
// is no longer current it appends stopped only. With no open headless
// registration id (none has it, a promoted session attached it, or it was
// already stopped, which also wraps ErrDirectorStopped) it returns
// ErrDirectorNoOpenHeadless and writes nothing. An id not of the form d<N> is
// refused before anything is read, and a missing director.jsonl is refused
// without creating the state dir or the lock. A failed write ends one of the
// two ways Attach's does: cut back, with an error that says nothing was
// recorded, or not, wrapping ErrDirectorWriteNotRestored.
//
// The detached comes first. A reader without the lock (Read) can find the
// file between the two lines: one that reads while the write lands, or
// every reader once a failed write's cut back failed too and left only the
// first line. Detached alone leaves no registration current and id open:
// start and attach go on, which is right with its pane closed, and `ralph
// org director stop --id` still finds id, finds herdr no longer knows its
// pane, and writes its stopped. Stopped alone would leave id current but
// stopped: start and attach would refuse on it, stop would refuse it as
// already stopped, and only a detach the human must think of would end it.
func (s *DirectorStore) MarkStopped(id string, who DirectorWho) (wasCurrent bool, err error) {
	return s.markStopped(id, who, false, nil, nil)
}

// MarkStoppedIfUnchanged is MarkStopped for `ralph org director stop` (plan
// D14): read is the registration as stop read it, without the lock, before
// it closed anything, and read.ID is the id to stop. Under the lock, before
// anything is written, it compares the workspace and pane ids recorded now
// with read's. When either differs, a start still running recorded it after
// stop's read, so stop did not close it, and recording the stop would mark a
// live director stopped: it writes nothing and returns an error that wraps
// ErrDirectorChangedDuringStop and names both reads. An id only goes from
// empty to a value (recordHerdrID refuses one already recorded), so equal
// ids mean nothing was recorded since. launched of a launch herdr is not
// compared: it changes nothing stop closes. For a launch cmd (plan
// 2026-10-16-org-director-launch-cmd, D7) it also compares the pid and the
// start time the check-in recorded: when they differ, the launch script
// checked in after stop's read, so stop did not stop that process, and it
// writes nothing and returns an error that wraps
// ErrDirectorCheckedInDuringStop and names both reads. A pid too only goes
// from none to a value (MarkCmdLaunched records one while starting only).
// Every other refusal is MarkStopped's.
func (s *DirectorStore) MarkStoppedIfUnchanged(read DirectorRegistration, who DirectorWho) (wasCurrent bool, err error) {
	return s.markStopped(read.ID, who, false, &read, nil)
}

// MarkForceStopped is MarkStopped for `ralph org director stop --force`
// (plan D11): its stopped carries force, which says the pane and the
// workspace were not closed but left as they were, since herdr's labels did
// not confirm them as this launch's or a close failed. The detached it
// writes first for the current registration carries no force: unlike detach
// --force's, it names the id it ends. Like MarkStopped it compares no
// recorded id (plan D14): a forced stop records that nothing more was
// closed, which stays true when a start recorded an id meanwhile.
func (s *DirectorStore) MarkForceStopped(id string, who DirectorWho) (wasCurrent bool, err error) {
	return s.markStopped(id, who, true, nil, nil)
}

// MarkStoppedAtVersion is MarkStoppedIfUnchanged for start's cleanup of a
// current registration that its check found ended (plan
// 2026-10-18-org-director-liveness, D6, cleanUpEndedDirector): read is the
// registration as the check read it (version A, DirectorCheckResult's
// Registration), read.ID the id to stop, and v that read's version
// (DirectorCheckResult's version). Under the lock, before anything else, it
// computes the version of read.ID in director.jsonl as it is now
// (directorVersionOf) and, when that is not v, writes nothing and returns an
// error that wraps ErrDirectorChangedSinceCheck and names both versions: an
// event for read.ID was recorded after the check read it (any event:
// launched, workspace_created, pane_created, detached, or stopped), or
// another registration is current, so the cleanup, which closed only what
// version A recorded, must not record the stop. A zero v never matches.
// Otherwise it writes what MarkStoppedIfUnchanged writes for read (stopped,
// with detached first while read.ID is current, in one write), whose
// comparisons then pass, and every other refusal is MarkStopped's.
func (s *DirectorStore) MarkStoppedAtVersion(read DirectorRegistration, v directorVersion, who DirectorWho) (wasCurrent bool, err error) {
	return s.markStopped(read.ID, who, false, &read, &v)
}

// markStopped is MarkStopped, with force MarkForceStopped, with a non-nil
// read MarkStoppedIfUnchanged, and with a non-nil version too
// MarkStoppedAtVersion: read is the registration as stop (or start's check)
// read it, whose workspace and pane ids, pid, and start time must still be
// the recorded ones, and version the directorVersion of that read, which the
// file must still hold for id.
func (s *DirectorStore) markStopped(id string, who DirectorWho, force bool, read *DirectorRegistration, version *directorVersion) (wasCurrent bool, err error) {
	if _, ok := directorIDNumber(id); !ok {
		return false, fmt.Errorf("org: director: stop: %q is not a registration id (d<N>, such as d1)", id)
	}
	verb := "stop " + id
	if _, err := os.Stat(s.path); errors.Is(err, fs.ErrNotExist) {
		return false, directorNoRegistration(verb, id)
	}
	if s.beforeMarkStopped != nil {
		s.beforeMarkStopped()
	}
	err = s.withLock(func() error {
		raw, err := s.readRaw()
		if err != nil {
			return err
		}
		fold := foldDirector(raw)
		if version != nil {
			if now := directorVersionOf(raw, fold, id); now != *version {
				return fmt.Errorf("org: director: %s: %w: %s had %s when the check read it, and has %s now", verb,
					ErrDirectorChangedSinceCheck, id, directorVersionText(*version), directorVersionText(now))
			}
		}
		reg, err := fold.openHeadless(verb, id)
		if err != nil {
			return err
		}
		switch {
		case read == nil:
		case reg.WorkspaceID != read.WorkspaceID || reg.PaneID != read.PaneID:
			return fmt.Errorf("org: director: %s: %w: %s had %s when stop read it, and has %s now", verb,
				ErrDirectorChangedDuringStop, id, directorRecordedIDs(*read), directorRecordedIDs(*reg))
		case reg.PID != read.PID || reg.StartTime != read.StartTime:
			return fmt.Errorf("org: director: %s: %w: %s had %s when stop read it, and has %s now", verb,
				ErrDirectorCheckedInDuringStop, id, directorRecordedProcess(*read), directorRecordedProcess(*reg))
		}
		ts := s.now()
		var events []DirectorEvent
		var cur *DirectorRegistration
		if fold.current == reg {
			cur = reg
			events = append(events, DirectorEvent{TS: ts, ID: id, Event: DirectorEventDetached, User: who.User, Cwd: who.Cwd, HerdrPane: who.HerdrPane})
		}
		events = append(events, DirectorEvent{TS: ts, ID: id, Event: DirectorEventStopped, User: who.User, Cwd: who.Cwd, HerdrPane: who.HerdrPane, Force: force})
		if err := s.appendLocked(raw, events...); err != nil {
			return directorAppendFailed(verb, cur, err)
		}
		wasCurrent = cur != nil
		return nil
	})
	if err != nil {
		return false, err
	}
	return wasCurrent, nil
}

// directorRecordedIDs names the workspace and pane ids recorded for reg in
// an error, "no workspace" and "no pane" for one not recorded.
func directorRecordedIDs(reg DirectorRegistration) string {
	ws, pane := "no workspace", "no pane"
	if reg.WorkspaceID != "" {
		ws = fmt.Sprintf("workspace %q", reg.WorkspaceID)
	}
	if reg.PaneID != "" {
		pane = fmt.Sprintf("pane %q", reg.PaneID)
	}
	return ws + " and " + pane
}

// directorVersionText names a directorVersion in MarkStoppedAtVersion's
// error: how many events its id had and which registration was current
// ("no registration" for none). Both come from the fold, which reads only an
// id of the form d<N>, so neither needs escaping.
func directorVersionText(v directorVersion) string {
	current := "no registration"
	if v.currentID != "" {
		current = v.currentID
	}
	return fmt.Sprintf("%d events, with %s current", v.events, current)
}

// directorRecordedProcess names the pid and start time recorded for reg, a
// launch cmd, in an error: "no pid" before its check-in.
func directorRecordedProcess(reg DirectorRegistration) string {
	if reg.PID == 0 && reg.StartTime == "" {
		return "no pid"
	}
	return fmt.Sprintf("pid %d started %q", reg.PID, reg.StartTime)
}

// The refusals and failures of StartDirector that the CLI tells apart, each
// wrapped by an error that names the case. StopDirector's are in
// director_stop.go.
var (
	// ErrDirectorDisabled: [org.director].enabled is false, so start starts
	// nothing (D5).
	ErrDirectorDisabled = errors.New("[org.director].enabled is false")
	// ErrDirectorStartInsideHerdr: start ran inside a herdr pane
	// (HERDR_PANE_ID is set).
	ErrDirectorStartInsideHerdr = errors.New("run ralph org director start from a terminal outside herdr")
	// ErrDirectorCodexWorkspaceWrite: the driver is codex and role
	// director's permission mode is autonomous or edits, refused whatever
	// [org.permissions].codex_verified says (plan D12). Both modes run codex
	// under --sandbox workspace-write, which keeps .git read-only, so the
	// director could not start a feature org (ralph org start --plan runs
	// git worktree add).
	ErrDirectorCodexWorkspaceWrite = errors.New("a codex director runs only in permission mode guarded")
	// ErrDirectorReplacedDuringStart: by the time start's agent was up, its
	// registration was no longer the current one (a human's attach
	// --replace, detach --force, or stop came in while start ran), so start
	// closed what it made and wrote no launched (D3). On the launch_cmd path
	// it is the same while start waits for the launch script's check-in, and
	// start stops what checked in instead of closing herdr.
	ErrDirectorReplacedDuringStart = errors.New("the registration stopped being the current one while start ran")
	// ErrDirectorNoCheckIn: on the launch_cmd path, launch_cmd returned but
	// no launch script checked in before the timeout (no terminal ran it, or
	// its check-in was refused or failed), so start recorded the stop (plan
	// 2026-10-16-org-director-launch-cmd, D5 and D6).
	ErrDirectorNoCheckIn = errors.New("the launch script did not check in before the timeout")
	// ErrDirectorChangedSinceCheck: start's cleanup of a current headless
	// registration that its check found ended (plan
	// 2026-10-18-org-director-liveness, D6) found under director.lock that
	// director.jsonl recorded something for it after the check read it, or
	// that another registration is current (MarkStoppedAtVersion), so it
	// recorded nothing and started nothing: what it closed is only what the
	// check read, and the registration may no longer be the one that ended.
	ErrDirectorChangedSinceCheck = errors.New("the registration changed after start checked it")
)

// DirectorStartParams describes one StartDirector call. Store is the
// director.jsonl of the ledger: its directory, made absolute, is the org
// state dir the director is given (--state-dir in its prompt), and the
// prompt file goes under <state dir>/prompts/. Model is resolved by the
// caller. Cwd is the pane's directory, made absolute against the caller's.
// ConfigPath is the ralph.toml the caller was given with --config, "" when
// none was: made absolute against the caller's directory, it goes into the
// prompt, whose ralph org commands for the director to run carry it as
// --config (its stop command, for the human, does not: stop reads no
// ralph.toml), so the director, which runs in Cwd, reads the settings start
// read. TimeoutMS bounds the herdr agent start, and on the launch_cmd path
// the wait for the launch script's check-in; 0 or less means
// defaultSpawnTimeoutMS. AgmsgHome is used on the launch_cmd path only:
// when it is not empty (the CLI gives start's RALPH_ORG_AGMSG_HOME), the
// launch script exports it as RALPH_ORG_AGMSG_HOME (plan
// 2026-10-16-org-director-launch-cmd, D3). Who is who writes the events.
// Notice gets the one line start writes when it cleaned up a current
// headless registration that its check found ended (plan
// 2026-10-18-org-director-liveness, D6), as soon as that cleanup is
// recorded, so the line is there even when start is refused afterwards; nil
// means the line goes nowhere.
type DirectorStartParams struct {
	Store      *DirectorStore
	Driver     string
	Model      string
	Cwd        string
	ConfigPath string
	TimeoutMS  int
	AgmsgHome  string
	Who        DirectorWho
	Notice     io.Writer
}

// DirectorStartResult is what StartDirector launched: the registration
// (state running, with its workspace and pane, or, for a launch cmd, with
// the pid and start time its launch script checked in with), whose Launch
// says which (its HerdrLaunch or CmdLaunch), and the prompt file. For a
// launch herdr: the herdr agent name, the labels of its workspace and tab,
// and how many agent_pane_busy retries the agent start took. For a launch
// cmd: TerminalName, director_<token>, which launch_cmd's {name} gave the
// terminal, and LaunchScriptPath, the script that terminal ran.
type DirectorStartResult struct {
	Registration      DirectorRegistration
	PromptPath        string
	AgentName         string
	WorkspaceLabel    string
	TabLabel          string
	AgentStartRetries int
	TerminalName      string
	LaunchScriptPath  string
}

// DirectorAttachedError is StartDirector's refusal while a registration is
// current. Err wraps ErrDirectorAttached and names the registration refused
// on, in AttachHeadless's words, whether checkDirectorStart's first look
// found it or AttachHeadless did under the lock. Current is the current
// registration: the one the first look found, or, after AttachHeadless's
// refusal, the one read again without the lock, which can be newer, and nil
// when that read failed or none is current by then. Its Headless tells the
// CLI which way out to print: `ralph org director stop --id` for a headless
// one, `ralph org director detach --id` for a promoted session's.
//
// Check is set when the first look checked the current headless
// registration on the spot (CheckDirector; plan
// 2026-10-18-org-director-liveness, D6) and found it alive or could not tell
// (unknown): what the check found, which the CLI adds to the refusal.
// Current is then the registration as that check read it. Check is nil for a
// promoted session's registration, which is not checked, for a headless one
// the check had nothing to check in (stopped while still current, say), and
// for AttachHeadless's refusal under the lock. Error is Err's text either
// way.
type DirectorAttachedError struct {
	Current *DirectorRegistration
	Err     error
	Check   *DirectorCheck
}

func (e *DirectorAttachedError) Error() string { return e.Err.Error() }

func (e *DirectorAttachedError) Unwrap() error { return e.Err }

// StartDirector launches a headless director in a herdr workspace of its own
// (plan D3, D6, and D9) and returns it, in state running. It writes only
// p.Store's director.jsonl (and director.lock) and the prompt file: no
// manifest line and no model receipt (D2, D8).
//
// It refuses, before anything is written and before any herdr call, when
// [org.director].enabled is false (ErrDirectorDisabled), when HERDR_PANE_ID
// is set (ErrDirectorStartInsideHerdr), for a driver other than claude and
// codex or an empty model, for a [org.director].wake that
// config.ValidateDirectorWake refuses (an Org can be built without
// config.Load; plan 2026-10-18-org-director-wake, AC3), for codex when the
// permission mode of role director ([org.permissions.roles].director, else
// [org.permissions].default) is autonomous or edits, whatever codex_verified
// says (ErrDirectorCodexWorkspaceWrite, D12), when that mode does not map to
// the driver's flags (permissionArgsForDriver), while a registration is current
// (*DirectorAttachedError), and when o.WatchGate refuses. The current
// registration is checked twice: first by a read without the lock before
// the watch gate, so a start it refuses launches no watch (the gate the CLI
// wires may launch one), then by AttachHeadless under the lock, which
// decides. A director.jsonl that cannot be read is refused at the first
// read. A seat's scope gate (autonomousScopeGateErr) does not apply: the
// director is not a seat.
//
// A current registration that `ralph org director start` wrote is not
// simply refused at that first read (plan 2026-10-18-org-director-liveness,
// D6, checkCurrentDirector): start checks it on the spot (CheckDirector).
// Alive or unknown is refused as before, with Check set; ended is cleaned up
// as `ralph org director stop` without --force cleans it up, but only on the
// registration as the check read it (cleanUpEndedDirector), recorded stopped
// with MarkStoppedAtVersion, reported in one line on p.Notice, and start then
// goes on to the watch gate. That cleanup makes herdr calls (the label checks,
// C-c, and the closes) and writes director.jsonl before the watch gate, so a
// start the gate refuses afterwards has still cleaned up. A cleanup that did
// not get as far as its record returns its *DirectorCleanupError, and start
// writes nothing.
//
// Then, in this order, with no herdr call under director.lock (D3):
//  1. AttachHeadless records the registration, state starting, with a new
//     id and launch token, the wake of [org.director].wake, and the
//     starter, this process's pid and its start time (directorStarter; plan
//     2026-10-18-org-director-liveness, D1), or no starter when ps does not
//     read them
//  2. WorkspaceCreate labelled DirectorWorkspaceLabel(token), then
//     RecordWorkspace
//  3. TabCreate labelled DirectorTabLabel(token), then RecordPane
//  4. RenderDirectorPrompt, with p.ConfigPath (when set) as the --config
//     of its ralph org commands, written to
//     <state dir>/prompts/director_<token>.md
//  5. the agent start of DirectorAgentName(token) on that pane with the
//     permission flags, --model, and the prompt file's PromptFilePointer,
//     in this order (no --add-dir: a codex director runs guarded only)
//  6. MarkLaunched
//
// WorkspaceCreate and TabCreate each run under driverCallTimeout. The agent
// start, with its agent_pane_busy retries (agentStartWithRetry), runs under
// one context of p.TimeoutMS, which is also herdr's own --timeout, as in
// Spawn.
//
// When a step from 2 on fails, or step 6 finds the registration no longer
// current (ErrDirectorReplacedDuringStart), start closes what it made
// (closeDirectorResources, by the ids it got from herdr even when recording
// one failed) and records the registration stopped (MarkStopped: with
// detached while it is current), and returns the step's error. When it
// cannot close them, it writes detached only, and only while the
// registration is current (Detach), and returns a *DirectorCleanupError.
//
// With [org.director].launch_cmd set (plan
// 2026-10-16-org-director-launch-cmd, D2), start makes no herdr call at
// all: it records the registration with launch cmd and opens the human's
// terminal with launch_cmd instead (startCmd, in director_cmd.go).
// HERDR_PANE_ID does not refuse it (D9), and it also refuses, before
// anything is written and with the refusals above in their order, a
// [org.director].wake with push while [org.director].wake_cmd is empty
// (ErrDirectorWakeCmdNeeded: launch_cmd opens a terminal ralph cannot type
// into, so nothing could wake the director; plan
// 2026-10-18-org-director-wake, D4 and AC3), a launch_cmd that
// config.ValidateDirectorLaunchCmd refuses, a driver's agent that is not on
// PATH, and a ralph executable it cannot find (checkDirectorLaunchCmd). A
// start in herdr with push and an empty wake_cmd is not refused: its herdr
// pane is the target, and input to that pane is how the watch sends.
func (o *Org) StartDirector(p DirectorStartParams) (DirectorStartResult, error) {
	c, err := o.checkDirectorStart(&p)
	if err != nil {
		return DirectorStartResult{}, err
	}
	launch := DirectorLaunchHerdr
	if c.cmdLaunch {
		launch = DirectorLaunchCmd
	}
	starterPID, starterStartTime := o.directorStarter()
	reg, err := p.Store.AttachHeadless(DirectorHeadlessAttach{
		Launch: launch, Driver: p.Driver, Model: p.Model, PermissionMode: c.mode, Wake: o.Config.Director.Wake,
		StarterPID: starterPID, StarterStartTime: starterStartTime,
		User: p.Who.User, Cwd: p.Who.Cwd, HerdrPane: p.Who.HerdrPane,
	})
	if errors.Is(err, ErrDirectorAttached) {
		return DirectorStartResult{}, newDirectorAttachedError(p.Store, err)
	}
	if err != nil {
		return DirectorStartResult{}, err
	}
	r := &directorStartRun{o: o, p: p, stateDir: c.stateDir, reg: reg}
	if c.cmdLaunch {
		return r.startCmd(c)
	}
	if err := r.createWorkspaceAndTab(); err != nil {
		return DirectorStartResult{}, r.abort(err)
	}
	promptPath, err := r.writePrompt()
	if err != nil {
		return DirectorStartResult{}, r.abort(err)
	}
	agentArgs := append(append([]string{}, c.permArgs...), "--model", p.Model, PromptFilePointer(promptPath))
	retries, err := r.startAgent(agentArgs)
	if err != nil {
		return DirectorStartResult{}, r.abort(err)
	}
	stillCurrent, err := p.Store.MarkLaunched(reg.ID, p.Who)
	switch {
	case err != nil:
		return DirectorStartResult{}, r.abort(err)
	case !stillCurrent:
		return DirectorStartResult{}, r.abort(r.replacedError())
	}
	r.reg.State = DirectorStateRunning
	return DirectorStartResult{
		Registration: r.reg, PromptPath: promptPath, AgentName: DirectorAgentName(reg.Token),
		WorkspaceLabel: DirectorWorkspaceLabel(reg.Token), TabLabel: DirectorTabLabel(reg.Token),
		AgentStartRetries: retries,
	}, nil
}

// directorStarter is the starter StartDirector records on its attached (plan
// 2026-10-18-org-director-liveness, D1): the pid of this process, the start
// process, and that pid's start time, read with readDirectorProcess
// (o.DirectorProcessReader when set, under one driverCallTimeout) after
// checkDirectorStart, so a refused start runs no ps. Only a read that
// answers alive with a start time gives both. A read that fails, or that
// answers anything else, gives neither, and start goes on with no starter
// recorded, so its launch script's check-in is not refused for one
// (MarkCmdLaunched).
func (o *Org) directorStarter() (pid int, startTime string) {
	pid = os.Getpid()
	proc, err := o.readDirectorProcess(pid)
	startTime = strings.TrimSpace(proc.StartTime)
	if err != nil || proc.State != DirectorProcessAlive || startTime == "" {
		return 0, ""
	}
	return pid, startTime
}

// directorStartChecks is what checkDirectorStart resolved: the absolute
// state dir, the permission flags and mode of role director, whether
// [org.director].launch_cmd is set, and, when it is, the absolute paths of
// the driver's agent and of the ralph the launch script runs.
type directorStartChecks struct {
	stateDir             string
	permArgs             []string
	mode                 string
	cmdLaunch            bool
	agentPath, ralphPath string
}

// checkDirectorStart runs StartDirector's refusals that need no write and
// no herdr call, in the order its doc comment lists them. Its look at a
// current registration reads without the lock and only refuses early, or,
// for a headless one its check finds ended, cleans it up and goes on
// (checkCurrentDirector, the one step here that may call herdr and write);
// AttachHeadless decides again under the lock. It returns what it resolved
// (directorStartChecks), and leaves p with Cwd and a non-empty ConfigPath
// made absolute and TimeoutMS defaulted.
func (o *Org) checkDirectorStart(p *DirectorStartParams) (directorStartChecks, error) {
	c := directorStartChecks{cmdLaunch: len(o.Config.Director.LaunchCmd) > 0}
	switch {
	case p.Store == nil:
		return c, errors.New("org: director: start: no director store was given")
	case p.Cwd == "":
		return c, errors.New("org: director: start: the cwd is empty")
	case !o.Config.Director.Enabled:
		return c, fmt.Errorf("org: director: start: %w, so no director is started", ErrDirectorDisabled)
	// The launch_cmd path runs no herdr, so a herdr pane is no reason to
	// refuse it (plan 2026-10-16-org-director-launch-cmd, D9).
	case !c.cmdLaunch && o.getenv(herdrPaneIDEnv) != "":
		return c, fmt.Errorf("org: director: start: HERDR_PANE_ID is set, so this runs inside a herdr pane: %w", ErrDirectorStartInsideHerdr)
	case p.Driver != DirectorDriverClaude && p.Driver != DirectorDriverCodex:
		return c, fmt.Errorf("org: director: start: unknown driver %q: use %s or %s", p.Driver, DirectorDriverClaude, DirectorDriverCodex)
	case p.Model == "":
		return c, errors.New("org: director: start: the model is empty")
	}
	if err := o.checkDirectorStartWake(c.cmdLaunch); err != nil {
		return c, err
	}
	mode := ResolvePermissionMode(o.Config, DirectorRole)
	c.mode = mode
	// Before permissionArgsForDriver, so with codex_verified false this
	// refusal, not the codex_verified one, says why (plan D12).
	if p.Driver == DirectorDriverCodex && CodexModeRunsUnderWorkspaceWrite(mode) {
		return c, fmt.Errorf("org: director: start: permission mode %q of role %s ([org.permissions.roles].%s, else [org.permissions].default): "+
			"%w, whatever [org.permissions].codex_verified says: autonomous and edits run codex under --sandbox workspace-write, which keeps .git read-only, "+
			"so the director could not start a feature org (ralph org start --plan runs git worktree add); "+
			"set [org.permissions.roles].%s = %q and let your codex config (~/.codex/config.toml) allow writing .git and reaching the network, "+
			"or start a claude director (--driver %s); a guarded codex director follows that config, so under sandbox_mode = \"workspace-write\" it meets the same wall",
			mode, DirectorRole, DirectorRole, ErrDirectorCodexWorkspaceWrite, DirectorRole, PermissionModeGuarded, DirectorDriverClaude)
	}
	var err error
	if c.permArgs, err = permissionArgsForDriver(o.Config, p.Driver, mode); err != nil {
		return c, fmt.Errorf("org: director: start: permission mode %q of role %s ([org.permissions.roles].%s, else [org.permissions].default): %w",
			mode, DirectorRole, DirectorRole, err)
	}
	// Before the first look and the watch gate, like the input checks
	// above: a launch_cmd that cannot run its agent writes nothing and
	// launches no watch (plan 2026-10-16-org-director-launch-cmd, AC5).
	if c.cmdLaunch {
		if c.agentPath, c.ralphPath, err = o.checkDirectorLaunchCmd(p.Driver); err != nil {
			return c, err
		}
	}
	// Before the watch gate, which may launch a watch, so a start refused
	// while a director is attached starts nothing (self-review L1 of
	// org-director-start).
	res, err := p.Store.Read()
	if err != nil {
		return c, err
	}
	if cur := res.Current; cur != nil {
		if err := o.checkCurrentDirector(p, cur); err != nil {
			return c, err
		}
	}
	if err := o.checkWatchGate(); err != nil {
		return c, err
	}
	if c.stateDir, err = absPath(filepath.Dir(p.Store.Path())); err != nil {
		return c, fmt.Errorf("org: director: start: resolve the state dir: %w", err)
	}
	if p.Cwd, err = absPath(p.Cwd); err != nil {
		return c, fmt.Errorf("org: director: start: resolve the cwd: %w", err)
	}
	if p.ConfigPath != "" {
		if p.ConfigPath, err = absPath(p.ConfigPath); err != nil {
			return c, fmt.Errorf("org: director: start: resolve the --config path: %w", err)
		}
	}
	if p.TimeoutMS <= 0 {
		p.TimeoutMS = defaultSpawnTimeoutMS
	}
	return c, nil
}

// checkDirectorStartWake is checkDirectorStart's check of [org.director].wake
// (plan 2026-10-18-org-director-wake, D4 and AC3), one of its input checks,
// so it comes before checkDirectorLaunchCmd, the first look, and the watch
// gate, and a refusal writes nothing, runs neither launch_cmd nor herdr,
// and launches no watch. The wake must pass config.ValidateDirectorWake,
// since an Org can be built without config.Load. With cmdLaunch (a start
// with [org.director].launch_cmd), a wake with push needs a non-empty
// [org.director].wake_cmd: the terminal launch_cmd opens is one ralph cannot
// type into, so without wake_cmd nothing could send the wake line. A start
// in herdr needs none: its pane is the target, and input to it is how the
// watch sends when wake_cmd is empty.
func (o *Org) checkDirectorStartWake(cmdLaunch bool) error {
	wake := o.Config.Director.Wake
	if err := config.ValidateDirectorWake(wake); err != nil {
		return fmt.Errorf("org: director: start: %w", err)
	}
	if cmdLaunch && slices.Contains(wake, DirectorWakePush) && len(o.Config.Director.WakeCmd) == 0 {
		return fmt.Errorf("org: director: start: %w (wake %s), and [org.director].launch_cmd is set: launch_cmd opens a terminal ralph cannot type into, "+
			"so ralph org watch --all could not wake the director; set [org.director].wake_cmd (its {target} is director_<token>, the name launch_cmd "+
			"gives the terminal) or use wake = [%q]", ErrDirectorWakeCmdNeeded, strings.Join(wake, ","), DirectorWakePull)
	}
	return nil
}

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

// newDirectorAttachedError is the *DirectorAttachedError for AttachHeadless's
// refusal err, with the current registration read again from store.
func newDirectorAttachedError(store *DirectorStore, err error) error {
	e := &DirectorAttachedError{Err: err}
	if res, readErr := store.Read(); readErr == nil {
		e.Current = res.Current
	}
	return e
}

// directorStartRun is one StartDirector past AttachHeadless: reg is the
// registration it made, with WorkspaceID and PaneID filled in as soon as
// herdr hands them out, before they are recorded, so the cleanup closes them
// even when recording one fails.
type directorStartRun struct {
	o        *Org
	p        DirectorStartParams
	stateDir string
	reg      DirectorRegistration
}

// createWorkspaceAndTab is steps 2 and 3 of StartDirector.
func (r *directorStartRun) createWorkspaceAndTab() error {
	o, p, token := r.o, r.p, r.reg.Token
	err := o.callWithTimeout(func(ctx context.Context) error {
		var err error
		r.reg.WorkspaceID, err = o.Herdr.WorkspaceCreate(ctx, p.Cwd, DirectorWorkspaceLabel(token))
		return err
	})
	if err != nil {
		r.reg.WorkspaceID = ""
		return fmt.Errorf("org: director: start %s: create the herdr workspace (one herdr made anyway is labelled %s): %w",
			r.reg.ID, DirectorWorkspaceLabel(token), err)
	}
	if err := p.Store.RecordWorkspace(r.reg.ID, r.reg.WorkspaceID, p.Who); err != nil {
		return err
	}
	err = o.callWithTimeout(func(ctx context.Context) error {
		var err error
		r.reg.PaneID, err = o.Herdr.TabCreate(ctx, r.reg.WorkspaceID, p.Cwd, DirectorTabLabel(token))
		return err
	})
	if err != nil {
		r.reg.PaneID = ""
		return fmt.Errorf("org: director: start %s: create the herdr tab in workspace %s: %w", r.reg.ID, r.reg.WorkspaceID, err)
	}
	return p.Store.RecordPane(r.reg.ID, r.reg.PaneID, p.Who)
}

// writePrompt is step 4 of StartDirector, and the first step of startCmd:
// it renders the prompt for the registration's launch and returns the
// prompt file's path.
func (r *directorStartRun) writePrompt() (string, error) {
	text, err := RenderDirectorPrompt(DirectorPromptVars{
		ID: r.reg.ID, Driver: r.p.Driver, StateDir: r.stateDir, ConfigPath: r.p.ConfigPath, Cwd: r.p.Cwd, Launch: r.reg.Launch,
	})
	if err != nil {
		return "", fmt.Errorf("org: director: start %s: %w", r.reg.ID, err)
	}
	path := filepath.Join(r.stateDir, "prompts", DirectorAgentName(r.reg.Token)+".md")
	if err := writePromptFile(path, text); err != nil {
		return "", fmt.Errorf("org: director: start %s: %s: %w", r.reg.ID, path, err)
	}
	return path, nil
}

// startAgent is step 5 of StartDirector: it returns the agent_pane_busy
// retry count.
func (r *directorStartRun) startAgent(agentArgs []string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.p.TimeoutMS)*time.Millisecond)
	defer cancel()
	name := DirectorAgentName(r.reg.Token)
	retries, err := r.o.agentStartWithRetry(ctx, name, r.p.Driver, r.reg.PaneID, r.p.TimeoutMS, agentArgs)
	if err != nil {
		return retries, fmt.Errorf("org: director: start %s: herdr agent start %s (agent_start_retries=%d): %w", r.reg.ID, name, retries, err)
	}
	return retries, nil
}

// replacedError is the ErrDirectorReplacedDuringStart error of step 6,
// naming the registration that is current now, read without the lock.
func (r *directorStartRun) replacedError() error {
	now := "no registration is current"
	res, err := r.p.Store.Read()
	switch {
	case err != nil:
		now = fmt.Sprintf("director.jsonl could not be read again (%v)", err)
	case res.Current != nil:
		now = "the current registration is " + res.Current.describe()
	}
	return fmt.Errorf("org: director: start %s: %w (an attach --replace, a detach --force, or a stop came in): %s",
		r.reg.ID, ErrDirectorReplacedDuringStart, now)
}

// abort is StartDirector's cleanup after cause: close what it made, then
// record the stop, or, when the close did not finish, detached only. A
// registration a stop recorded stopped first (ErrDirectorStopped) counts as
// recorded. A record that fails otherwise is a *DirectorCleanupError whose
// CleanupErr is directorRecordStopError, so its text has "org: director: "
// once, in cause.
func (r *directorStartRun) abort(cause error) error {
	closed := r.o.closeDirectorResources(r.reg)
	if closed.err == nil {
		_, err := r.p.Store.MarkStopped(r.reg.ID, r.p.Who)
		if err == nil || errors.Is(err, ErrDirectorStopped) {
			return fmt.Errorf("%w; start closed what it had made in herdr, and %s is recorded stopped", cause, r.reg.ID)
		}
		return newDirectorCleanupError("start", r.reg, closed, cause, &directorRecordStopError{id: r.reg.ID, err: err}, false)
	}
	w := r.p.Who
	_, err := r.p.Store.Detach(DirectorDetach{ID: r.reg.ID, User: w.User, Cwd: w.Cwd, HerdrPane: w.HerdrPane})
	switch {
	case err == nil:
		return newDirectorCleanupError("start", r.reg, closed, cause, closed.err, true)
	case errors.Is(err, ErrDirectorIDMismatch), errors.Is(err, ErrDirectorNotAttached):
		return newDirectorCleanupError("start", r.reg, closed, cause, closed.err, false)
	default:
		return newDirectorCleanupError("start", r.reg, closed, cause, errors.Join(closed.err, fmt.Errorf("detach %s: %w", r.reg.ID, err)), false)
	}
}
