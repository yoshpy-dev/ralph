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
	"strings"
	"time"
)

// The headless director of stage 8a-2 (plan
// docs/plans/active/2026-10-15-org-director-start.md): the DirectorStore
// writers of a headless registration with its launch token and the herdr
// names built from it, and the org-layer verbs StartDirector and
// StopDirector, which launch a headless director in a herdr workspace of its
// own and close it again. Neither verb writes a manifest line or a model
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
// (RecordWorkspace, RecordPane, MarkLaunched, MarkStopped): USER, the
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
			return fmt.Errorf("org: director: %s: %w: %s", verb, ErrDirectorAttached, cur.describe())
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
// registration id are closed, which `ralph org director stop` and start's
// cleanup do once herdr has closed them (or never made them, or no longer
// knows them), and returns whether id was the current registration. For the
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
		ts := s.now()
		var events []DirectorEvent
		var cur *DirectorRegistration
		if fold.current == reg {
			cur = reg
			events = append(events, DirectorEvent{TS: ts, ID: id, Event: DirectorEventDetached, User: who.User, Cwd: who.Cwd, HerdrPane: who.HerdrPane})
		}
		events = append(events, DirectorEvent{TS: ts, ID: id, Event: DirectorEventStopped, User: who.User, Cwd: who.Cwd, HerdrPane: who.HerdrPane})
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

// The refusals and failures of StartDirector and StopDirector that the CLI
// tells apart, each wrapped by an error that names the case.
var (
	// ErrDirectorDisabled: [org.director].enabled is false, so start starts
	// nothing (D5).
	ErrDirectorDisabled = errors.New("[org.director].enabled is false")
	// ErrDirectorStartInsideHerdr: start ran inside a herdr pane
	// (HERDR_PANE_ID is set).
	ErrDirectorStartInsideHerdr = errors.New("run ralph org director start from a terminal outside herdr")
	// ErrDirectorReplacedDuringStart: by the time start's agent was up, its
	// registration was no longer the current one (a human's attach
	// --replace, detach --force, or stop came in while start ran), so start
	// closed what it made and wrote no launched (D3).
	ErrDirectorReplacedDuringStart = errors.New("the registration stopped being the current one while start ran")
	// ErrDirectorNotHeadless is wrapped, with ErrDirectorNoOpenHeadless, by
	// StopDirector's refusal of a registration that a session promoted with
	// `ralph org director attach`: it has no herdr pane of ralph's to close,
	// and `ralph org director detach --id` ends it.
	ErrDirectorNotHeadless = errors.New("the registration was not started in herdr by ralph org director start")
	// ErrDirectorStopSelfPane: stop ran inside the director's own pane
	// (HERDR_PANE_ID is its recorded pane), and closing it would end the
	// command before it records the stop.
	ErrDirectorStopSelfPane = errors.New("run ralph org director stop from a terminal outside the director's pane")
)

// DirectorStartParams describes one StartDirector call. Store is the
// director.jsonl of the ledger: its directory, made absolute, is the org
// state dir the director is given (--state-dir in its prompt, and codex's
// --add-dir), and the prompt file goes under <state dir>/prompts/. Model is
// resolved by the caller. Cwd is the pane's directory, made absolute against
// the caller's. TimeoutMS bounds the herdr agent start; 0 or less means
// defaultSpawnTimeoutMS. Who is who writes the events.
type DirectorStartParams struct {
	Store     *DirectorStore
	Driver    string
	Model     string
	Cwd       string
	TimeoutMS int
	Who       DirectorWho
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
// current. Err is AttachHeadless's error, which wraps ErrDirectorAttached and
// names the registration it refused on. Current is the current registration
// read again after the refusal, without the lock, so it can be newer; nil
// when the read failed or none is current by then. Its Headless tells the
// CLI which way out to print: `ralph org director stop --id` for a headless
// one, `ralph org director detach --id` for a promoted session's.
type DirectorAttachedError struct {
	Current *DirectorRegistration
	Err     error
}

func (e *DirectorAttachedError) Error() string { return e.Err.Error() }

func (e *DirectorAttachedError) Unwrap() error { return e.Err }

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
// records the stop afterwards; the CLI adds its --state-dir to it.
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
	if e.StopCommand != "" {
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

// StartDirector launches a headless director in a herdr workspace of its own
// (plan D3, D6, and D9) and returns it, in state running. It writes only
// p.Store's director.jsonl (and director.lock) and the prompt file: no
// manifest line and no model receipt (D2, D8).
//
// It refuses, before anything is written and before any herdr call, when
// [org.director].enabled is false (ErrDirectorDisabled), when HERDR_PANE_ID
// is set (ErrDirectorStartInsideHerdr), for a driver other than claude and
// codex or an empty model, when the permission mode of role director
// ([org.permissions.roles].director, else [org.permissions].default) does not
// map to the driver's flags (permissionArgsForDriver: codex is fail-closed
// until codex_verified), when o.WatchGate refuses, and while a registration
// is current (*DirectorAttachedError). A seat's scope gate
// (autonomousScopeGateErr) does not apply: the director is not a seat.
//
// Then, in this order, with no herdr call under director.lock (D3):
//  1. AttachHeadless records the registration, state starting, with a new
//     id and launch token
//  2. WorkspaceCreate labelled DirectorWorkspaceLabel(token), then
//     RecordWorkspace
//  3. TabCreate labelled DirectorTabLabel(token), then RecordPane
//  4. RenderDirectorPrompt, written to <state dir>/prompts/director_<token>.md
//  5. the agent start of DirectorAgentName(token) on that pane with the
//     permission flags, codexWritableRootArgs, --model, and the prompt
//     file's PromptFilePointer, in this order
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
	agentArgs := append(append([]string{}, permArgs...), codexWritableRootArgs(DirectorRole, p.Driver, mode, p.Cwd, stateDir)...)
	agentArgs = append(agentArgs, "--model", p.Model, PromptFilePointer(promptPath))
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
// no herdr call, in the order its doc comment lists them (all but the one
// on a current registration, which AttachHeadless decides under the lock).
// It returns the absolute state dir, the permission flags, and the
// permission mode, and leaves p with Cwd made absolute and TimeoutMS
// defaulted.
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
	if permArgs, err = permissionArgsForDriver(o.Config, p.Driver, mode); err != nil {
		return "", nil, "", fmt.Errorf("org: director: start: permission mode %q of role %s ([org.permissions.roles].%s, else [org.permissions].default): %w",
			mode, DirectorRole, DirectorRole, err)
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
	text, err := RenderDirectorPrompt(DirectorPromptVars{ID: r.reg.ID, Driver: r.p.Driver, StateDir: r.stateDir, Cwd: r.p.Cwd})
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

// DirectorStopParams describes one StopDirector call: Store is the
// director.jsonl of the ledger, ID the registration to stop, and Who who
// writes the stop.
type DirectorStopParams struct {
	Store *DirectorStore
	ID    string
	Who   DirectorWho
}

// DirectorStopResult is what StopDirector did. Registration is the open
// headless registration as read before the stop. WasCurrent says it was the
// current registration, so detached was written with stopped and no
// director is attached now. CtrlCNote, PaneNote, and WorkspaceNote say what
// happened to the pane and the workspace. UnrecordedWorkspaceLabel is set
// when no workspace id was recorded: herdr may still hold a workspace with
// this label that start made before it could record the id.
type DirectorStopResult struct {
	Registration             DirectorRegistration
	WasCurrent               bool
	CtrlCNote                string
	PaneNote                 string
	WorkspaceNote            string
	UnrecordedWorkspaceLabel string
}

// StopDirector closes the herdr pane and workspace of the open headless
// registration p.ID and records the stop (plan D4): current or detached,
// starting or running. It refuses, with nothing written and no herdr call,
// an empty or malformed id, an id no registration has, a registration a
// session promoted (ErrDirectorNotHeadless) or one already stopped
// (ErrDirectorStopped), both with ErrDirectorNoOpenHeadless, and a call from
// inside the director's own pane (ErrDirectorStopSelfPane).
//
// closeDirectorResources checks herdr's labels for the recorded pane and
// workspace before anything is sent (D9): a label that is not this launch's
// sends no C-c and closes nothing. Then it sends C-c to the pane, closes it,
// and closes the workspace; a pane or workspace herdr no longer knows counts
// as closed, and one not recorded is skipped. When any of that does not
// finish, it returns a *DirectorCleanupError and writes nothing. Otherwise
// MarkStopped writes stopped, with detached when the registration is
// current. On a *DirectorCleanupError the result still carries Registration
// and the notes.
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
	res := DirectorStopResult{Registration: reg}
	if reg.WorkspaceID == "" {
		res.UnrecordedWorkspaceLabel = DirectorWorkspaceLabel(reg.Token)
	}
	closed := o.closeDirectorResources(reg)
	res.CtrlCNote, res.PaneNote, res.WorkspaceNote = closed.ctrlC, closed.pane, closed.workspace
	if closed.err != nil {
		return res, newDirectorCleanupError("stop", reg, closed, nil, closed.err, false)
	}
	wasCurrent, err := p.Store.MarkStopped(p.ID, p.Who)
	if err != nil {
		return res, newDirectorCleanupError("stop", reg, closed, nil, fmt.Errorf("record %s stopped: %w", p.ID, err), false)
	}
	res.WasCurrent = wasCurrent
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
// first checks both: the pane with confirmSeatPane (its tab labelled
// DirectorTabLabel and its workspace DirectorWorkspaceLabel of reg's token),
// the workspace with confirmOrgWorkspace (labelled DirectorWorkspaceLabel).
// When either is not confirmed, or its get fails, nothing is sent to either.
// Then it sends C-c to the pane (a failed C-c is a note), closes it, and
// closes the workspace, stopping at the first close that fails. A recorded
// id herdr no longer knows counts as closed, and one not recorded is skipped.
// Every herdr call runs under its own driverCallTimeout.
func (o *Org) closeDirectorResources(reg DirectorRegistration) directorClose {
	tabLabel, wsLabel := DirectorTabLabel(reg.Token), DirectorWorkspaceLabel(reg.Token)
	c := directorClose{ctrlC: "ctrl_c=skipped", pane: "pane=none recorded", workspace: "workspace=none recorded",
		paneOpen: reg.PaneID != "", workspaceOpen: reg.WorkspaceID != ""}
	var paneGone, wsGone bool
	if reg.PaneID != "" {
		gone, err := o.confirmSeatPane(wsLabel, tabLabel, reg.PaneID)
		if err != nil {
			return c.notConfirmed(fmt.Errorf("pane %q is not confirmed as %s's (a tab labelled %s in a workspace labelled %s), so nothing was sent to it: %w",
				reg.PaneID, reg.ID, tabLabel, wsLabel, err))
		}
		paneGone = gone
	}
	if reg.WorkspaceID != "" {
		gone, err := o.confirmOrgWorkspace(wsLabel, reg.WorkspaceID)
		if err != nil {
			return c.notConfirmed(fmt.Errorf("workspace %q is not confirmed as %s's (labelled %s), so nothing was sent to it: %w",
				reg.WorkspaceID, reg.ID, wsLabel, err))
		}
		wsGone = gone
	}
	if reg.PaneID != "" && !o.closeDirectorPane(&c, reg.PaneID, paneGone) {
		return c
	}
	if reg.WorkspaceID != "" {
		o.closeDirectorWorkspace(&c, reg.WorkspaceID, wsGone)
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
		StopCommand: "ralph org director stop --id " + reg.ID,
	}
	switch {
	case closed.unconfirmed:
		// The recorded ids may name another launch's pane and workspace, so
		// the human finds this launch's workspace by its label instead.
		e.HerdrCommands = []string{"herdr workspace list", "herdr workspace close <the id of the workspace labelled " + e.WorkspaceLabel + ">"}
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
