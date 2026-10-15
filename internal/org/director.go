package org

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The director registration (FR-9 of
// docs/specs/2026-10-07-org-multi-org-director.md; plan
// docs/plans/archive/2026-10-14-org-director-attach.md, D3 and D8):
// director.jsonl in the shared org state dir, next to the manifest and the
// inbox. It belongs to the whole ledger, not to one org, so it is not a
// manifest line (every manifest line needs an org_id). It is append-only;
// the current registration is folded from its attached and detached events.
// At most one registration is current: the human talks to one director.
// Writes take director.lock, a lock of its own, so no path holds it together
// with the manifest's or the inbox's lock. A registration says only that a
// director was attached; it is no sign that the director is alive (D4).
//
// Stage 8a-2 (plan docs/plans/active/2026-10-15-org-director-start.md, D2,
// D3, D4, and D9) adds the headless director that `ralph org director
// start` launches in a herdr workspace of its own. Its attached carries
// launch herdr, the token of that launch (the names of its herdr agent,
// workspace, and tab hold it, D9), the model, and the permission mode, and
// is written before any herdr call, so the registration is decided first
// (D3). workspace_created and pane_created add the ids herdr gave, launched
// says the agent started while the registration was still current, and
// stopped says its pane and workspace were closed (D4). stopped is not
// detached: detach only ends the current registration, so a headless
// registration detached that way stays open, and `ralph org director stop`
// can still close its pane, until a stopped is written for it. The headless
// director is not an org seat, so none of this is a manifest line (D2). The
// writers of a headless registration, its token and names, and the verb
// StartDirector are in director_start.go; StopDirector is in
// director_stop.go.
//
// Stage 8a-2b (plan docs/plans/active/2026-10-16-org-director-launch-cmd.md,
// D3, D4, D7, and D11) adds the headless director that start opens in a
// terminal of the human's with [org.director].launch_cmd instead of herdr.
// Its attached carries launch cmd with the same token, model, and permission
// mode, and is written first too. It has no workspace or pane: its launched
// is the check-in of the launch script (`ralph org director launched`,
// MarkCmdLaunched) and carries the pid of that script, which execs the agent
// and so keeps its pid, and the pid's start time as ps reads it
// (ReadDirectorProcess), which together tell this launch's process from a
// later one that got the same pid. Both launches are headless (Headless);
// what only herdr has, the workspace and the pane, is for launch herdr only
// (HerdrLaunch). The launch script, the launch_cmd runner, and the process
// checks are in director_cmd.go.

// Director event kinds: the "event" field of a director.jsonl line. The last
// four are written only for a headless registration (launch herdr or cmd;
// workspace_created and pane_created for launch herdr only).
const (
	DirectorEventAttached         = "attached"
	DirectorEventDetached         = "detached"
	DirectorEventWorkspaceCreated = "workspace_created"
	DirectorEventPaneCreated      = "pane_created"
	DirectorEventLaunched         = "launched"
	DirectorEventStopped          = "stopped"
)

// The launches of a registration that `ralph org director start` wrote. A
// promoted session's attached (`ralph org director attach`) has no launch.
const (
	// DirectorLaunchHerdr is a headless director in a herdr workspace of its
	// own.
	DirectorLaunchHerdr = "herdr"
	// DirectorLaunchCmd is a headless director in a terminal that
	// [org.director].launch_cmd opened, which runs the launch script
	// (stage 8a-2b).
	DirectorLaunchCmd = "cmd"
)

// The states of a headless registration (DirectorRegistration.State):
// starting until its launched event, running after it, and stopped after
// its stopped event. A promoted session's registration has no state.
const (
	DirectorStateStarting = "starting"
	DirectorStateRunning  = "running"
	DirectorStateStopped  = "stopped"
)

// Drivers a director can be attached with.
const (
	DirectorDriverClaude = "claude"
	DirectorDriverCodex  = "codex"
)

// DirectorWakePull is the wake mode of a director that waits on the inbox
// itself (`ralph org wait --inbox`). It is the only mode so far; stage 8b
// adds push.
const DirectorWakePull = "pull"

// directorLockFile is the lock file every director.jsonl write takes, in the
// same directory as director.jsonl.
const directorLockFile = "director.lock"

// ErrDirectorAttached is returned (wrapped) by Attach without Replace and by
// AttachHeadless while a registration is current, a headless one in state
// starting included. The message names that registration and no way out of
// it; the caller says how to end or replace it (the CLI's attachedError).
var ErrDirectorAttached = errors.New("a director is already attached")

// ErrDirectorNoOpenHeadless is returned (wrapped) by RecordWorkspace,
// RecordPane, MarkStopped, MarkStoppedIfUnchanged, and MarkForceStopped when
// no open headless registration has the id: no registration has it, a
// promoted session attached it (no launch herdr or cmd), or it was already
// stopped, when the error wraps ErrDirectorStopped too.
var ErrDirectorNoOpenHeadless = errors.New("no open headless registration has the id")

// ErrDirectorNotHerdrLaunch is returned (wrapped) by RecordWorkspace and
// RecordPane for an open headless registration with launch cmd, which has no
// herdr workspace or pane, and for now by StopDirector for one: it closes
// herdr panes and workspaces only, and the stop of a launch cmd (plan
// 2026-10-16-org-director-launch-cmd, D7) is not wired in yet.
var ErrDirectorNotHerdrLaunch = errors.New("the headless registration was started with [org.director].launch_cmd, not in herdr")

// ErrDirectorCheckInRefused is returned (wrapped) by MarkCmdLaunched, the
// check-in of a launch script, whenever it writes nothing: the id is not a
// registration with launch cmd, the token is not that launch's, the
// registration is stopped, no longer current, or no longer starting, or the
// pid or its start time is not usable. The message says which. The launch
// script then exits without starting the agent.
var ErrDirectorCheckInRefused = errors.New("the check-in of the launch script is refused, so its agent must not start")

// ErrDirectorStopped is wrapped together with ErrDirectorNoOpenHeadless when
// the headless registration the id names already has its stopped event.
var ErrDirectorStopped = errors.New("the headless registration was already stopped")

// ErrDirectorNotAttached is returned (wrapped) by Detach when no
// registration is current.
var ErrDirectorNotAttached = errors.New("no director is attached")

// ErrDirectorIDMismatch is returned (wrapped) by Detach with an id that is
// not the current registration's: the registration it names was replaced or
// has already ended, and the current one belongs to someone else (D8).
var ErrDirectorIDMismatch = errors.New("the id is not the current registration's")

// ErrDirectorWriteNotRestored is returned (wrapped) by every DirectorStore
// writer (Attach, Detach, AttachHeadless, RecordWorkspace, RecordPane,
// MarkLaunched, MarkCmdLaunched, MarkStopped, MarkStoppedIfUnchanged, and
// MarkForceStopped) when a write to director.jsonl failed and the file could
// not be cut back to its size before the write: it may hold part of the
// write (a replace cut off after the detached line leaves no registration
// current), so the error cannot say that nothing was recorded, and the
// caller says how to read the registration again (the CLI names `ralph org
// director status`).
// Every other failed append left nothing in the file: a failed open or
// marshal writes nothing, and a failed write or close is cut back.
var ErrDirectorWriteNotRestored = errors.New("director.jsonl could not be restored to its size before the append")

// directorRawIDPattern finds an "id":"d<N>" in the raw text of
// director.jsonl, including lines that do not parse, so the next id is never
// one a damaged line already holds (D8; inboxRawIDPattern does the same for
// the inbox). It does not require the closing quote: a line cut off inside
// the digits still counts.
var directorRawIDPattern = regexp.MustCompile(`"id"\s*:\s*"d([0-9]+)`)

// DirectorEvent is one line of director.jsonl. Every line carries ts, id,
// event, and who wrote it: user (USER), cwd, and herdr_pane (whether
// HERDR_PANE_ID was set; its value is not recorded). attached also carries
// driver and wake. A promoted session's attached (`ralph org director
// attach`) carries nothing more; the fold still reads a pane_id in it, as
// stage 8a-1 did. The attached of `ralph org director start` adds launch
// (herdr), token, model, and permission_mode, and leaves pane_id empty: the
// pane is carried by pane_created, and the workspace by workspace_created
// (workspace_id). launched carries nothing more. stopped carries force when
// `ralph org director stop --force` wrote it without closing the pane and the
// workspace (plan D11: herdr's labels did not confirm them, or a close
// failed); the fold reads it as any stopped. detached carries the id of the
// registration it ends, and force when it was removed without that id being
// given. A line of stage 8a-1 has none of the new fields and reads as it
// did.
//
// Stage 8a-2b (plan 2026-10-16-org-director-launch-cmd): the attached of a
// start with [org.director].launch_cmd carries launch cmd and the same
// token, model, and permission mode; its launched, the launch script's
// check-in (MarkCmdLaunched), adds pid and start_time, the pid's start time
// as ReadDirectorProcess reads it. No other event carries them, so every
// line of stage 8a-2 reads as it did.
type DirectorEvent struct {
	TS             string   `json:"ts"`    // UTC RFC3339
	ID             string   `json:"id"`    // d<N>
	Event          string   `json:"event"` // one of the DirectorEvent* constants
	Driver         string   `json:"driver,omitempty"`
	Wake           []string `json:"wake,omitempty"`
	PaneID         string   `json:"pane_id,omitempty"`
	Launch         string   `json:"launch,omitempty"` // DirectorLaunchHerdr, DirectorLaunchCmd, or empty
	Token          string   `json:"token,omitempty"`  // 12 lowercase hex digits
	Model          string   `json:"model,omitempty"`
	PermissionMode string   `json:"permission_mode,omitempty"`
	WorkspaceID    string   `json:"workspace_id,omitempty"`
	PID            int      `json:"pid,omitempty"`        // launched of a launch cmd only
	StartTime      string   `json:"start_time,omitempty"` // launched of a launch cmd only
	User           string   `json:"user"`
	Cwd            string   `json:"cwd"`
	HerdrPane      bool     `json:"herdr_pane"`
	Force          bool     `json:"force,omitempty"`
}

// DirectorRegistration is a registration, folded from its attached event
// and, for a headless one, the events of its launch. AttachedAt is that
// attached's ts; User, Cwd, and HerdrPane are who attached it. Launch,
// Token, Model, and PermissionMode come from the attached of a headless
// registration (`ralph org director start`); WorkspaceID and PaneID from its
// workspace_created and pane_created (a promoted session's PaneID from its
// attached), for launch herdr only; PID and StartTime from the launched of a
// launch cmd, the check-in of its launch script; State is starting, running,
// or stopped (DirectorState*). A promoted session's registration has none of
// the fields after PaneID, so its JSON is the shape of stage 8a-1, and a
// launch herdr's has no PID or StartTime, so its JSON is the shape of stage
// 8a-2.
type DirectorRegistration struct {
	ID             string   `json:"id"`
	Driver         string   `json:"driver"`
	Wake           []string `json:"wake"`
	PaneID         string   `json:"pane_id,omitempty"`
	Launch         string   `json:"launch,omitempty"`
	State          string   `json:"state,omitempty"`
	Token          string   `json:"token,omitempty"`
	Model          string   `json:"model,omitempty"`
	PermissionMode string   `json:"permission_mode,omitempty"`
	WorkspaceID    string   `json:"workspace_id,omitempty"`
	PID            int      `json:"pid,omitempty"`
	StartTime      string   `json:"start_time,omitempty"`
	AttachedAt     string   `json:"attached_at"`
	User           string   `json:"user"`
	Cwd            string   `json:"cwd"`
	HerdrPane      bool     `json:"herdr_pane"`
}

// Headless reports whether `ralph org director start` wrote the
// registration (launch herdr or cmd), not a promoted session (plan
// 2026-10-16-org-director-launch-cmd, D11): stop, the note of `ralph org
// stop --all`, and status handle both launches. What only herdr has, the
// workspace and the pane, is for HerdrLaunch.
func (r DirectorRegistration) Headless() bool {
	return r.HerdrLaunch() || r.CmdLaunch()
}

// HerdrLaunch reports whether the registration is a headless director in a
// herdr workspace (launch herdr): only it has a workspace and a pane that
// start records and stop checks by their labels and closes.
func (r DirectorRegistration) HerdrLaunch() bool {
	return r.Launch == DirectorLaunchHerdr
}

// CmdLaunch reports whether the registration is a headless director in a
// terminal that [org.director].launch_cmd opened (launch cmd): only it has a
// pid and a start time, from its launch script's check-in.
func (r DirectorRegistration) CmdLaunch() bool {
	return r.Launch == DirectorLaunchCmd
}

// describe names the registration in an error: its id, driver, and attach
// time, and for a headless registration its state, so a refusal names a
// registration still starting as such, and for a launch cmd that launch and
// the pid its check-in recorded. The fold checks the id, the driver, and the
// launch, but the attach time is the event's ts as written, which a hand
// edit can fill with control characters, so it is escaped like inbox text
// before it reaches a terminal, and so is a launch this binary does not
// know.
func (r DirectorRegistration) describe() string {
	details := "driver " + r.Driver + ", attached " + PrintableInboxText(r.AttachedAt, false)
	switch {
	case r.HerdrLaunch():
		details += ", headless " + r.State
	case r.CmdLaunch():
		details += ", headless " + r.State + ", launch " + DirectorLaunchCmd
		if r.PID > 0 {
			details += ", pid " + strconv.Itoa(r.PID)
		}
	case r.Launch != "":
		details += ", launch " + PrintableInboxText(r.Launch, false)
	}
	return r.ID + " (" + details + ")"
}

// DirectorReadResult is the folded director.jsonl: the current registration
// (nil when none); every open headless registration, that is, every one with
// launch herdr or cmd and no stopped, current or detached, ordered by its
// d<N> (nil when none), which `ralph org director stop` can still close; the
// number of lines that did not parse (skipped, like the inbox's corrupt
// lines); and the number of parsed events the fold ignored: an unknown kind,
// an id not of the form d<N>, an attached with a driver other than claude
// and codex, an attached with launch herdr or cmd whose token is not 12
// lowercase hex digits, an attached whose id an earlier attached already
// used, an attached while another registration is current, a detached for an
// id that is not the current registration's, a workspace_created or
// pane_created that names no open headless registration with launch herdr,
// carries no id, or comes after one already recorded (the first one is
// kept), a launched for an id that is not the current registration's or not
// starting, a launched of a launch herdr that carries a pid or a start time
// (herdr's never does), a launched of a launch cmd without a positive pid
// and a start time, and a stopped for an id that names no open headless
// registration.
type DirectorReadResult struct {
	Current       *DirectorRegistration
	OpenHeadless  []DirectorRegistration
	CorruptLines  int
	IgnoredEvents int
}

// DirectorAttach is what Attach records. Replace ends the current
// registration first instead of refusing. User, Cwd, and HerdrPane describe
// the caller; the store does not read the environment.
type DirectorAttach struct {
	Driver    string
	Replace   bool
	User      string
	Cwd       string
	HerdrPane bool
}

// DirectorDetach is what Detach records. Exactly one of ID (the current
// registration's d<N>) and Force (end the current registration whatever its
// id) is set. User, Cwd, and HerdrPane describe the caller.
type DirectorDetach struct {
	ID        string
	Force     bool
	User      string
	Cwd       string
	HerdrPane bool
}

// DirectorStore reads and appends to director.jsonl.
type DirectorStore struct {
	path string
	// Now is the clock for event timestamps; nil means time.Now.
	Now func() time.Time
	// write and truncate are test-only seams for appendLocked's write and
	// its restore after a failure; nil means f.Write and os.Truncate.
	write    func(f *os.File, b []byte) (int, error)
	truncate func(path string, size int64) error
	// token is a test-only seam for AttachHeadless's launch token; nil means
	// newDirectorToken.
	token func() (string, error)
	// beforeMarkStopped is a test-only seam that markStopped runs before it
	// takes the lock, so a test can land a check-in between a caller's read
	// and its record of the stop; nil means none.
	beforeMarkStopped func()
}

// DirectorPathIn returns the director.jsonl path within an already-resolved
// org state directory, mirroring InboxPathIn.
func DirectorPathIn(stateDir string) string {
	return filepath.Join(stateDir, "director.jsonl")
}

// NewDirectorStore returns the DirectorStore of the resolved org state dir
// stateDir. Its lock file is <stateDir>/director.lock.
func NewDirectorStore(stateDir string) *DirectorStore {
	return &DirectorStore{path: DirectorPathIn(stateDir)}
}

// Path returns the director.jsonl path this store reads and appends to.
func (s *DirectorStore) Path() string {
	return s.path
}

// Read folds every event in director.jsonl. A missing file (or state dir)
// reads as no registration, not an error; a file that cannot be read is an
// error. Read takes no lock; the writers decide under the lock.
func (s *DirectorStore) Read() (DirectorReadResult, error) {
	raw, err := s.readRaw()
	if err != nil {
		return DirectorReadResult{}, err
	}
	return foldDirector(raw).result(), nil
}

// Attach records a new registration with driver a.Driver and wake pull, and
// returns it. Under the director lock it reads the file again; while a
// registration is current (a headless one too, in state starting
// included) it returns ErrDirectorAttached, naming that registration, and
// writes nothing, unless a.Replace is set: then it appends the current
// registration's detached and the new attached as two lines of one write
// (appendLocked), and returns the ended registration as replaced. A headless
// registration replaced that way stays open, for `ralph org director stop`
// to close its pane, as one detached by Detach does. A reader without the
// lock (Read) thus finds no moment with no registration between the two
// lines, as it could when they were two writes (self-review L5 of
// org-director-attach). A write that fails ends one of two ways. Either it
// is cut back under the same lock (appendLocked), even when it stopped
// after the detached line, so the old registration stays current and the
// error says nothing was recorded (only a Read between the failed write and
// the cut can see part of it); or the cut fails too, and the error wraps
// ErrDirectorWriteNotRestored and does not say that. A failed write of a
// plain attach ends the same two ways. The new id is one more than the
// largest d<N> anywhere in the file, damaged lines included. A driver other
// than claude and codex is refused before anything is read or written.
func (s *DirectorStore) Attach(a DirectorAttach) (reg DirectorRegistration, replaced *DirectorRegistration, err error) {
	if a.Driver != DirectorDriverClaude && a.Driver != DirectorDriverCodex {
		return DirectorRegistration{}, nil, fmt.Errorf("org: director: attach: unknown driver %q: use %s or %s", a.Driver, DirectorDriverClaude, DirectorDriverCodex)
	}
	err = s.withLock(func() error {
		raw, err := s.readRaw()
		if err != nil {
			return err
		}
		fold := foldDirector(raw)
		cur := fold.current
		if cur != nil && !a.Replace {
			return fmt.Errorf("org: director: attach: %w: %s", ErrDirectorAttached, cur.describe())
		}
		next, err := fold.nextID()
		if err != nil {
			return err
		}
		ts := s.now()
		var events []DirectorEvent
		if cur != nil {
			events = append(events, DirectorEvent{
				TS: ts, ID: cur.ID, Event: DirectorEventDetached,
				User: a.User, Cwd: a.Cwd, HerdrPane: a.HerdrPane,
			})
		}
		ev := DirectorEvent{
			TS: ts, ID: next, Event: DirectorEventAttached,
			Driver: a.Driver, Wake: []string{DirectorWakePull},
			User: a.User, Cwd: a.Cwd, HerdrPane: a.HerdrPane,
		}
		if err := s.appendLocked(raw, append(events, ev)...); err != nil {
			return directorAppendFailed("attach", cur, err)
		}
		reg = registrationOf(ev)
		replaced = cur
		return nil
	})
	if err != nil {
		return DirectorRegistration{}, nil, err
	}
	return reg, replaced, nil
}

// Detach ends the current registration and returns it. With d.ID it
// returns ErrDirectorIDMismatch and writes nothing when the id is not the
// current registration's, so a director that was replaced cannot end its
// successor's registration (D8); with d.Force it ends the current
// registration whatever its id (a human's cleanup) and records force.
// Giving neither or both, or an id not of the form d<N>, is refused before
// anything is read. With no current registration it returns
// ErrDirectorNotAttached; a missing director.jsonl returns it without
// creating the state dir or the lock. A failed write ends one of the same
// two ways as Attach's: it is cut back, and the error says nothing was
// recorded and the registration is still current; or the cut fails too, and
// the error wraps ErrDirectorWriteNotRestored and does not say that.
func (s *DirectorStore) Detach(d DirectorDetach) (DirectorRegistration, error) {
	verb := "detach " + d.ID
	switch {
	case d.ID == "" && !d.Force:
		return DirectorRegistration{}, errors.New("org: director: detach needs the registration id or force")
	case d.ID != "" && d.Force:
		return DirectorRegistration{}, errors.New("org: director: detach takes the registration id or force, not both")
	case d.Force:
		verb = "force detach"
	default:
		if _, ok := directorIDNumber(d.ID); !ok {
			return DirectorRegistration{}, fmt.Errorf("org: director: detach: %q is not a registration id (d<N>, such as d1)", d.ID)
		}
	}
	notAttached := fmt.Errorf("org: director: %s: %w", verb, ErrDirectorNotAttached)
	if _, err := os.Stat(s.path); errors.Is(err, fs.ErrNotExist) {
		return DirectorRegistration{}, notAttached
	}
	var ended DirectorRegistration
	err := s.withLock(func() error {
		raw, err := s.readRaw()
		if err != nil {
			return err
		}
		cur := foldDirector(raw).current
		if cur == nil {
			return notAttached
		}
		if !d.Force && d.ID != cur.ID {
			return fmt.Errorf("org: director: %s: %w: the current registration is %s, so the session that attached %s was replaced (or %s has already been detached) and does not need to detach", verb, ErrDirectorIDMismatch, cur.describe(), d.ID, d.ID)
		}
		ev := DirectorEvent{
			TS: s.now(), ID: cur.ID, Event: DirectorEventDetached,
			User: d.User, Cwd: d.Cwd, HerdrPane: d.HerdrPane, Force: d.Force,
		}
		if err := s.appendLocked(raw, ev); err != nil {
			return directorAppendFailed(verb, cur, err)
		}
		ended = *cur
		return nil
	})
	if err != nil {
		return DirectorRegistration{}, err
	}
	return ended, nil
}

// directorNoRegistration is the ErrDirectorNoOpenHeadless error for an id
// that no registration has.
func directorNoRegistration(verb, id string) error {
	return fmt.Errorf("org: director: %s: %w: no registration %s is recorded", verb, ErrDirectorNoOpenHeadless, id)
}

func (s *DirectorStore) withLock(fn func() error) error {
	return withFileLock(filepath.Dir(s.path), directorLockFile, "director", fn)
}

func (s *DirectorStore) now() string {
	nowFn := time.Now
	if s.Now != nil {
		nowFn = s.Now
	}
	return nowFn().UTC().Format(time.RFC3339)
}

// readRaw returns the whole director.jsonl; a missing file reads as empty.
func (s *DirectorStore) readRaw() ([]byte, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("org: director: read %s: %w", s.path, err)
	}
	return raw, nil
}

// appendLocked appends evs, one line each, in a single write, as the inbox's
// appendLocked does for its one event: Attach with replace passes the
// detached and the attached together, and MarkStopped,
// MarkStoppedIfUnchanged, and MarkForceStopped of the current registration
// the detached and the stopped. raw is the file as read under the same lock
// hold. When raw does not end with a newline (a write cut off mid-line), the
// write starts with one, so the cut line stays a separate corrupt line and
// evs stay readable. An event that does not marshal fails
// it before anything is written. When the write or the close fails, part of
// the lines may be in the file (a full disk can stop a write after the
// detached line), so it cuts the file back to len(raw) before returning
// (restoreLocked); every writer takes the lock, so that is the file's size
// before the append. A file this call created is cut to empty, not removed:
// an empty director.jsonl reads as no registration, as a missing one does.
// Only when the cut fails does the error wrap ErrDirectorWriteNotRestored;
// any other error means nothing was recorded. Its errors do not start with
// "org: director: ": the store's writers, its only callers, add that once
// (directorAppendFailed).
func (s *DirectorStore) appendLocked(raw []byte, evs ...DirectorEvent) error {
	var lines []byte
	if len(raw) > 0 && raw[len(raw)-1] != '\n' {
		lines = append(lines, '\n')
	}
	for _, ev := range evs {
		line, err := json.Marshal(ev)
		if err != nil {
			return fmt.Errorf("marshal the %s event: %w", ev.Event, err)
		}
		lines = append(append(lines, line...), '\n')
	}
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", s.path, err)
	}
	write := (*os.File).Write
	if s.write != nil {
		write = s.write
	}
	if _, err := write(f, lines); err != nil {
		_ = f.Close()
		return s.restoreLocked(len(raw), fmt.Errorf("append to %s: %w", s.path, err))
	}
	if err := f.Close(); err != nil {
		return s.restoreLocked(len(raw), fmt.Errorf("close %s: %w", s.path, err))
	}
	return nil
}

// restoreLocked cuts director.jsonl back to size, its length before the
// append that failed with failed, under the same lock hold, and returns
// failed. When the cut fails, the error it returns also wraps
// ErrDirectorWriteNotRestored and the cut's error.
func (s *DirectorStore) restoreLocked(size int, failed error) error {
	truncate := os.Truncate
	if s.truncate != nil {
		truncate = s.truncate
	}
	if err := truncate(s.path, int64(size)); err != nil {
		return fmt.Errorf("%w, and %w (%w), so it may hold part of the write", failed, ErrDirectorWriteNotRestored, err)
	}
	return failed
}

// directorAppendFailed is the error the store's writers (verb, as in their
// other errors) return when appendLocked fails while cur is the current
// registration that the write would have changed (nil when none, or when the
// write leaves the current registration as it is). It says "org: director:"
// once, since appendLocked's errors do not. When the file may hold part of
// the write, it ends at appendLocked's error, which says so, instead of
// saying nothing was recorded; the caller names the command that reads the
// registration of the ledger it was given (the CLI's notRestoredError,
// self-review C2-L1 of org-director-attach).
func directorAppendFailed(verb string, cur *DirectorRegistration, err error) error {
	switch {
	case errors.Is(err, ErrDirectorWriteNotRestored):
		return fmt.Errorf("org: director: %s: %w", verb, err)
	case cur != nil:
		return fmt.Errorf("org: director: %s: nothing was recorded, so %s is still the current registration: %w", verb, cur.describe(), err)
	default:
		return fmt.Errorf("org: director: %s: nothing was recorded: %w", verb, err)
	}
}

// registrationOf returns the registration an attached event records. A
// headless attached (launch herdr or cmd) starts in state starting; a launch
// herdr's pane and workspace come only from its pane_created and
// workspace_created, and a launch cmd has none, so a pane_id in that attached
// (start never writes one) is not read, nor is a pid or a start time (only a
// launch cmd's launched carries them). An attached with another launch keeps
// it, with its token, model, and permission mode, so a later binary's
// registration still counts as one.
func registrationOf(ev DirectorEvent) DirectorRegistration {
	reg := DirectorRegistration{
		ID: ev.ID, Driver: ev.Driver, Wake: ev.Wake, PaneID: ev.PaneID, AttachedAt: ev.TS,
		User: ev.User, Cwd: ev.Cwd, HerdrPane: ev.HerdrPane,
	}
	if ev.Launch != "" {
		reg.Launch, reg.Token, reg.Model, reg.PermissionMode = ev.Launch, ev.Token, ev.Model, ev.PermissionMode
	}
	if reg.Headless() {
		reg.State, reg.PaneID = DirectorStateStarting, ""
	}
	return reg
}

// directorFold is director.jsonl folded from its raw text.
type directorFold struct {
	// current is the current registration, one of regs.
	current *DirectorRegistration
	// regs holds every registration an applied attached event made, by id,
	// current or not: an attached that repeats an id is ignored, and the
	// events of a headless registration apply to it after it was detached.
	regs    map[string]*DirectorRegistration
	corrupt int
	ignored int
	// maxN is the largest d<N> found anywhere in the file: in the raw text
	// (directorRawIDPattern) and in the decoded id of every parsed line, so
	// an id written with JSON escapes counts too.
	maxN int64
}

func foldDirector(raw []byte) *directorFold {
	f := &directorFold{regs: map[string]*DirectorRegistration{}}
	for _, m := range directorRawIDPattern.FindAllSubmatch(raw, -1) {
		f.seeN(string(m[1]))
	}
	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var ev DirectorEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			f.corrupt++
			continue
		}
		f.apply(ev)
	}
	return f
}

// seeN raises maxN to digits' value. A value too large for int64 is
// skipped: nextID can never produce it, so it cannot be reused.
func (f *directorFold) seeN(digits string) {
	if n, err := strconv.ParseInt(digits, 10, 64); err == nil && n > f.maxN {
		f.maxN = n
	}
}

func (f *directorFold) apply(ev DirectorEvent) {
	if digits, ok := strings.CutPrefix(ev.ID, "d"); ok {
		f.seeN(digits)
	}
	if _, ok := directorIDNumber(ev.ID); !ok {
		f.ignored++
		return
	}
	// Only a hand edit (or a damaged file) writes an event that is ignored
	// below: Attach writes the current registration's detached before a new
	// attached, AttachHeadless writes a token of the right shape, and the
	// other writers decide under the lock on the same fold.
	reg := f.regs[ev.ID]
	switch ev.Event {
	case DirectorEventAttached:
		headless := ev.Launch == DirectorLaunchHerdr || ev.Launch == DirectorLaunchCmd
		if (ev.Driver != DirectorDriverClaude && ev.Driver != DirectorDriverCodex) || reg != nil || f.current != nil ||
			(headless && !directorTokenPattern.MatchString(ev.Token)) {
			f.ignored++
			return
		}
		r := registrationOf(ev)
		f.regs[ev.ID] = &r
		f.current = &r
	case DirectorEventDetached:
		if f.current == nil || ev.ID != f.current.ID {
			f.ignored++
			return
		}
		f.current = nil
	case DirectorEventWorkspaceCreated, DirectorEventPaneCreated:
		// A launch cmd has no herdr workspace or pane: start never records
		// one for it (recordHerdrID refuses), and stop must not close one.
		if !isOpenHeadless(reg) || !reg.HerdrLaunch() {
			f.ignored++
			return
		}
		field, value := &reg.WorkspaceID, ev.WorkspaceID
		if ev.Event == DirectorEventPaneCreated {
			field, value = &reg.PaneID, ev.PaneID
		}
		// The first id recorded is kept: start records each once, and the
		// stop that closes the pane and workspace by label must not be sent
		// to another id by a later line.
		if value == "" || *field != "" {
			f.ignored++
			return
		}
		*field = value
	case DirectorEventLaunched:
		// Start (MarkLaunched) and the check-in of a launch script
		// (MarkCmdLaunched) write launched under the lock only while its
		// registration is the current one and starting. Only the check-in's
		// carries a pid and its start time, and it always carries both.
		if reg == nil || reg != f.current || reg.State != DirectorStateStarting {
			f.ignored++
			return
		}
		switch {
		case reg.CmdLaunch() && (ev.PID <= 0 || ev.StartTime == ""),
			!reg.CmdLaunch() && (ev.PID != 0 || ev.StartTime != ""):
			f.ignored++
			return
		}
		reg.State, reg.PID, reg.StartTime = DirectorStateRunning, ev.PID, ev.StartTime
	case DirectorEventStopped:
		if !isOpenHeadless(reg) {
			f.ignored++
			return
		}
		reg.State = DirectorStateStopped
	default:
		f.ignored++
	}
}

// isOpenHeadless reports whether r is a headless registration with no
// stopped yet; a nil r is not.
func isOpenHeadless(r *DirectorRegistration) bool {
	return r != nil && r.Headless() && r.State != DirectorStateStopped
}

// openHeadless returns the open headless registration id of the fold, or the
// ErrDirectorNoOpenHeadless error that verb returns: no registration has
// id, a promoted session (or a later binary's launch) attached it, or it was
// already stopped (ErrDirectorStopped).
func (f *directorFold) openHeadless(verb, id string) (*DirectorRegistration, error) {
	reg := f.regs[id]
	switch {
	case reg == nil:
		return nil, directorNoRegistration(verb, id)
	case !reg.Headless():
		return nil, fmt.Errorf("org: director: %s: %w: %s was not started in herdr by ralph org director start", verb, ErrDirectorNoOpenHeadless, reg.describe())
	case reg.State == DirectorStateStopped:
		return nil, fmt.Errorf("org: director: %s: %w: %w: %s", verb, ErrDirectorNoOpenHeadless, ErrDirectorStopped, reg.describe())
	}
	return reg, nil
}

// openHerdrHeadless is openHeadless for what only a launch herdr has (its
// workspace and pane): an open headless registration with launch cmd is
// refused too, with ErrDirectorNotHerdrLaunch.
func (f *directorFold) openHerdrHeadless(verb, id string) (*DirectorRegistration, error) {
	reg, err := f.openHeadless(verb, id)
	if err != nil {
		return nil, err
	}
	if !reg.HerdrLaunch() {
		return nil, directorNotHerdrLaunchErr(verb, *reg)
	}
	return reg, nil
}

// directorNotHerdrLaunchErr is verb's ErrDirectorNotHerdrLaunch refusal of
// reg, an open headless registration with launch cmd.
func directorNotHerdrLaunchErr(verb string, reg DirectorRegistration) error {
	return fmt.Errorf("org: director: %s: %w: %s has no herdr workspace or pane", verb, ErrDirectorNotHerdrLaunch, reg.describe())
}

func (f *directorFold) nextID() (string, error) {
	if f.maxN == math.MaxInt64 {
		return "", fmt.Errorf("org: director: no registration id is left after d%d", f.maxN)
	}
	return "d" + strconv.FormatInt(f.maxN+1, 10), nil
}

// result returns the fold with copies of its registrations, so a caller
// cannot change the fold through them.
func (f *directorFold) result() DirectorReadResult {
	res := DirectorReadResult{CorruptLines: f.corrupt, IgnoredEvents: f.ignored}
	if f.current != nil {
		cur := *f.current
		res.Current = &cur
	}
	for _, reg := range f.regs {
		if isOpenHeadless(reg) {
			res.OpenHeadless = append(res.OpenHeadless, *reg)
		}
	}
	sort.Slice(res.OpenHeadless, func(i, j int) bool {
		a, _ := directorIDNumber(res.OpenHeadless[i].ID)
		b, _ := directorIDNumber(res.OpenHeadless[j].ID)
		return a < b
	})
	return res
}

// directorIDNumber returns N for a registration id d<N> written the way
// nextID writes it: N at least 1, no leading zero, within int64. Any other
// id (d01, d0, dx, D1) names no registration.
func directorIDNumber(id string) (int64, bool) {
	if !strings.HasPrefix(id, "d") {
		return 0, false
	}
	n, err := strconv.ParseInt(id[1:], 10, 64)
	if err != nil || n < 1 || "d"+strconv.FormatInt(n, 10) != id {
		return 0, false
	}
	return n, true
}
