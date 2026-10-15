package org

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// The headless director of stage 8a-2 (plan
// docs/plans/active/2026-10-15-org-director-start.md): the DirectorStore
// writers of a headless registration with its launch token and the herdr
// names built from it, and the org-layer verb StartDirector, which launches
// a headless director in a herdr workspace of its own. StopDirector, which
// closes it again, and the closing that start's cleanup shares with it are
// in director_stop.go. Neither verb writes a manifest line or a model
// receipt (D2, D8): director.jsonl is the only record. The fold that reads
// these events back is in director.go.

// DirectorHeadlessAttach is what AttachHeadless records: the driver, model,
// and permission mode (autonomous, edits, or guarded) of the headless
// director `ralph org director start` launches. User, Cwd, and HerdrPane
// describe the caller.
type DirectorHeadlessAttach struct {
	Driver         string
	Model          string
	PermissionMode string
	User           string
	Cwd            string
	HerdrPane      bool
}

// DirectorWho is who writes a headless registration's later events
// (RecordWorkspace, RecordPane, MarkLaunched, MarkStopped,
// MarkForceStopped): USER, the current directory, and whether HERDR_PANE_ID
// is set. The store does not read the environment.
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
// one attached with launch herdr, the token, h.Model, h.PermissionMode, and
// wake pull. The new id is chosen as Attach chooses it, and a failed write
// ends one of the two ways Attach's does. A driver other than claude and
// codex, an empty model, and a permission mode other than autonomous, edits,
// and guarded are refused before anything is read or written.
func (s *DirectorStore) AttachHeadless(h DirectorHeadlessAttach) (DirectorRegistration, error) {
	const verb = "start"
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
			Driver: h.Driver, Wake: []string{DirectorWakePull},
			Launch: DirectorLaunchHerdr, Token: token, Model: h.Model, PermissionMode: h.PermissionMode,
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
// open headless registration id it returns ErrDirectorNoOpenHeadless; when
// id already has a workspace recorded it refuses too, since the fold keeps
// the first one. Neither writes anything. An id not of the form d<N> and an
// empty workspaceID are refused before anything is read, and a missing
// director.jsonl is refused without creating the state dir or the lock. A
// failed write ends one of the two ways Attach's does.
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
		reg, err := foldDirector(raw).openHeadless(verb, id)
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
// current registration, a headless one in state starting; `ralph org
// director start` calls it once its agent start went through. When the
// current registration is another one, or none (a human's attach --replace
// or detach --force came in while start ran), or id is not headless or
// already stopped, it writes nothing and returns false with a nil error, and
// start closes its own pane and workspace (D3). A current id already running
// returns true and writes nothing. An id not of the form d<N> is refused
// before anything is read; a missing director.jsonl returns false without
// creating the state dir or the lock. A failed write ends one of the two ways
// Attach's does.
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
		if cur == nil || cur.ID != id || !cur.Headless() || cur.State == DirectorStateStopped {
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
	return s.markStopped(id, who, false, nil)
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
// ids mean nothing was recorded since. launched is not compared: it changes
// nothing stop closes. Every other refusal is MarkStopped's.
func (s *DirectorStore) MarkStoppedIfUnchanged(read DirectorRegistration, who DirectorWho) (wasCurrent bool, err error) {
	return s.markStopped(read.ID, who, false, &read)
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
	return s.markStopped(id, who, true, nil)
}

// markStopped is MarkStopped, with force MarkForceStopped, and with a
// non-nil read MarkStoppedIfUnchanged: read is the registration as stop read
// it, whose workspace and pane ids must still be the recorded ones.
func (s *DirectorStore) markStopped(id string, who DirectorWho, force bool, read *DirectorRegistration) (wasCurrent bool, err error) {
	if _, ok := directorIDNumber(id); !ok {
		return false, fmt.Errorf("org: director: stop: %q is not a registration id (d<N>, such as d1)", id)
	}
	verb := "stop " + id
	if _, err := os.Stat(s.path); errors.Is(err, fs.ErrNotExist) {
		return false, directorNoRegistration(verb, id)
	}
	err = s.withLock(func() error {
		raw, err := s.readRaw()
		if err != nil {
			return err
		}
		fold := foldDirector(raw)
		reg, err := fold.openHeadless(verb, id)
		if err != nil {
			return err
		}
		if read != nil && (reg.WorkspaceID != read.WorkspaceID || reg.PaneID != read.PaneID) {
			return fmt.Errorf("org: director: %s: %w: %s had %s when stop read it, and has %s now",
				verb, ErrDirectorChangedDuringStop, id, directorRecordedIDs(*read), directorRecordedIDs(*reg))
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
	// closed what it made and wrote no launched (D3).
	ErrDirectorReplacedDuringStart = errors.New("the registration stopped being the current one while start ran")
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
// read. TimeoutMS bounds the herdr agent start; 0 or less means
// defaultSpawnTimeoutMS. Who is who writes the events.
type DirectorStartParams struct {
	Store      *DirectorStore
	Driver     string
	Model      string
	Cwd        string
	ConfigPath string
	TimeoutMS  int
	Who        DirectorWho
}

// DirectorStartResult is what StartDirector launched: the registration
// (state running, with its workspace and pane), the prompt file, the herdr
// agent name, the labels of its workspace and tab, and how many
// agent_pane_busy retries the agent start took.
type DirectorStartResult struct {
	Registration      DirectorRegistration
	PromptPath        string
	AgentName         string
	WorkspaceLabel    string
	TabLabel          string
	AgentStartRetries int
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
type DirectorAttachedError struct {
	Current *DirectorRegistration
	Err     error
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
// codex or an empty model, for codex when the permission mode of role
// director ([org.permissions.roles].director, else
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
// Then, in this order, with no herdr call under director.lock (D3):
//  1. AttachHeadless records the registration, state starting, with a new
//     id and launch token
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
func (o *Org) StartDirector(p DirectorStartParams) (DirectorStartResult, error) {
	stateDir, permArgs, mode, err := o.checkDirectorStart(&p)
	if err != nil {
		return DirectorStartResult{}, err
	}
	reg, err := p.Store.AttachHeadless(DirectorHeadlessAttach{
		Driver: p.Driver, Model: p.Model, PermissionMode: mode,
		User: p.Who.User, Cwd: p.Who.Cwd, HerdrPane: p.Who.HerdrPane,
	})
	if errors.Is(err, ErrDirectorAttached) {
		return DirectorStartResult{}, newDirectorAttachedError(p.Store, err)
	}
	if err != nil {
		return DirectorStartResult{}, err
	}
	r := &directorStartRun{o: o, p: p, stateDir: stateDir, reg: reg}
	if err := r.createWorkspaceAndTab(); err != nil {
		return DirectorStartResult{}, r.abort(err)
	}
	promptPath, err := r.writePrompt()
	if err != nil {
		return DirectorStartResult{}, r.abort(err)
	}
	agentArgs := append(append([]string{}, permArgs...), "--model", p.Model, PromptFilePointer(promptPath))
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

// checkDirectorStart runs StartDirector's refusals that need no write and
// no herdr call, in the order its doc comment lists them. Its look at a
// current registration reads without the lock and only refuses early;
// AttachHeadless decides again under the lock. It returns the absolute
// state dir, the permission flags, and the permission mode, and leaves p
// with Cwd and a non-empty ConfigPath made absolute and TimeoutMS defaulted.
func (o *Org) checkDirectorStart(p *DirectorStartParams) (stateDir string, permArgs []string, mode string, err error) {
	switch {
	case p.Store == nil:
		return "", nil, "", errors.New("org: director: start: no director store was given")
	case p.Cwd == "":
		return "", nil, "", errors.New("org: director: start: the cwd is empty")
	case !o.Config.Director.Enabled:
		return "", nil, "", fmt.Errorf("org: director: start: %w, so no director is started", ErrDirectorDisabled)
	case o.getenv(herdrPaneIDEnv) != "":
		return "", nil, "", fmt.Errorf("org: director: start: HERDR_PANE_ID is set, so this runs inside a herdr pane: %w", ErrDirectorStartInsideHerdr)
	case p.Driver != DirectorDriverClaude && p.Driver != DirectorDriverCodex:
		return "", nil, "", fmt.Errorf("org: director: start: unknown driver %q: use %s or %s", p.Driver, DirectorDriverClaude, DirectorDriverCodex)
	case p.Model == "":
		return "", nil, "", errors.New("org: director: start: the model is empty")
	}
	mode = ResolvePermissionMode(o.Config, DirectorRole)
	// Before permissionArgsForDriver, so with codex_verified false this
	// refusal, not the codex_verified one, says why (plan D12).
	if p.Driver == DirectorDriverCodex && CodexModeRunsUnderWorkspaceWrite(mode) {
		return "", nil, "", fmt.Errorf("org: director: start: permission mode %q of role %s ([org.permissions.roles].%s, else [org.permissions].default): "+
			"%w, whatever [org.permissions].codex_verified says: autonomous and edits run codex under --sandbox workspace-write, which keeps .git read-only, "+
			"so the director could not start a feature org (ralph org start --plan runs git worktree add); "+
			"set [org.permissions.roles].%s = %q and let your codex config (~/.codex/config.toml) allow writing .git and reaching the network, "+
			"or start a claude director (--driver %s); a guarded codex director follows that config, so under sandbox_mode = \"workspace-write\" it meets the same wall",
			mode, DirectorRole, DirectorRole, ErrDirectorCodexWorkspaceWrite, DirectorRole, PermissionModeGuarded, DirectorDriverClaude)
	}
	if permArgs, err = permissionArgsForDriver(o.Config, p.Driver, mode); err != nil {
		return "", nil, "", fmt.Errorf("org: director: start: permission mode %q of role %s ([org.permissions.roles].%s, else [org.permissions].default): %w",
			mode, DirectorRole, DirectorRole, err)
	}
	// Before the watch gate, which may launch a watch, so a start refused
	// while a director is attached starts nothing (self-review L1 of
	// org-director-start).
	res, err := p.Store.Read()
	if err != nil {
		return "", nil, "", err
	}
	if cur := res.Current; cur != nil {
		return "", nil, "", &DirectorAttachedError{Current: cur, Err: directorStartAttachedErr(cur)}
	}
	if err := o.checkWatchGate(); err != nil {
		return "", nil, "", err
	}
	if stateDir, err = absPath(filepath.Dir(p.Store.Path())); err != nil {
		return "", nil, "", fmt.Errorf("org: director: start: resolve the state dir: %w", err)
	}
	if p.Cwd, err = absPath(p.Cwd); err != nil {
		return "", nil, "", fmt.Errorf("org: director: start: resolve the cwd: %w", err)
	}
	if p.ConfigPath != "" {
		if p.ConfigPath, err = absPath(p.ConfigPath); err != nil {
			return "", nil, "", fmt.Errorf("org: director: start: resolve the --config path: %w", err)
		}
	}
	if p.TimeoutMS <= 0 {
		p.TimeoutMS = defaultSpawnTimeoutMS
	}
	return stateDir, permArgs, mode, nil
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

// writePrompt is step 4 of StartDirector: it returns the prompt file's path.
func (r *directorStartRun) writePrompt() (string, error) {
	text, err := RenderDirectorPrompt(DirectorPromptVars{ID: r.reg.ID, Driver: r.p.Driver, StateDir: r.stateDir, ConfigPath: r.p.ConfigPath, Cwd: r.p.Cwd})
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
// record the stop, or, when the close did not finish, detached only.
func (r *directorStartRun) abort(cause error) error {
	closed := r.o.closeDirectorResources(r.reg)
	if closed.err == nil {
		_, err := r.p.Store.MarkStopped(r.reg.ID, r.p.Who)
		if err == nil || errors.Is(err, ErrDirectorStopped) {
			return fmt.Errorf("%w; start closed what it had made in herdr, and %s is recorded stopped", cause, r.reg.ID)
		}
		return newDirectorCleanupError("start", r.reg, closed, cause, fmt.Errorf("record %s stopped: %w", r.reg.ID, err), false)
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
