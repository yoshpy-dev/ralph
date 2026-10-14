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
	"strconv"
	"strings"
	"time"
)

// The director registration (FR-9 of
// docs/specs/2026-10-07-org-multi-org-director.md; plan
// docs/plans/active/2026-10-14-org-director-attach.md, D3 and D8):
// director.jsonl in the shared org state dir, next to the manifest and the
// inbox. It belongs to the whole ledger, not to one org, so it is not a
// manifest line (every manifest line needs an org_id). It is append-only;
// the current registration is folded from its attached and detached events.
// At most one registration is current: the human talks to one director.
// Writes take director.lock, a lock of its own, so no path holds it together
// with the manifest's or the inbox's lock. A registration says only that a
// director was attached; it is no sign that the director is alive (D4).

// Director event kinds: the "event" field of a director.jsonl line.
const (
	DirectorEventAttached = "attached"
	DirectorEventDetached = "detached"
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

// ErrDirectorAttached is returned (wrapped) by Attach without Replace while
// a registration is current.
var ErrDirectorAttached = errors.New("a director is already attached")

// ErrDirectorNotAttached is returned (wrapped) by Detach when no
// registration is current.
var ErrDirectorNotAttached = errors.New("no director is attached")

// ErrDirectorIDMismatch is returned (wrapped) by Detach with an id that is
// not the current registration's: the registration it names was replaced or
// has already ended, and the current one belongs to someone else (D8).
var ErrDirectorIDMismatch = errors.New("the id is not the current registration's")

// directorRawIDPattern finds an "id":"d<N>" in the raw text of
// director.jsonl, including lines that do not parse, so the next id is never
// one a damaged line already holds (D8; inboxRawIDPattern does the same for
// the inbox). It does not require the closing quote: a line cut off inside
// the digits still counts.
var directorRawIDPattern = regexp.MustCompile(`"id"\s*:\s*"d([0-9]+)`)

// DirectorEvent is one line of director.jsonl. Every line carries ts, id,
// event, and who wrote it: user (USER), cwd, and herdr_pane (whether
// HERDR_PANE_ID was set; its value is not recorded). attached also carries
// driver, wake, and pane_id (empty until `ralph org director start` of
// stage 8a-2 writes it). detached carries the id of the registration it
// ends, and force when it was removed without that id being given.
type DirectorEvent struct {
	TS        string   `json:"ts"`    // UTC RFC3339
	ID        string   `json:"id"`    // d<N>
	Event     string   `json:"event"` // one of the DirectorEvent* constants
	Driver    string   `json:"driver,omitempty"`
	Wake      []string `json:"wake,omitempty"`
	PaneID    string   `json:"pane_id,omitempty"`
	User      string   `json:"user"`
	Cwd       string   `json:"cwd"`
	HerdrPane bool     `json:"herdr_pane"`
	Force     bool     `json:"force,omitempty"`
}

// DirectorRegistration is the current registration, folded from its
// attached event. AttachedAt is that event's ts; User, Cwd, and HerdrPane
// are who attached it.
type DirectorRegistration struct {
	ID         string   `json:"id"`
	Driver     string   `json:"driver"`
	Wake       []string `json:"wake"`
	PaneID     string   `json:"pane_id,omitempty"`
	AttachedAt string   `json:"attached_at"`
	User       string   `json:"user"`
	Cwd        string   `json:"cwd"`
	HerdrPane  bool     `json:"herdr_pane"`
}

// describe names the registration in an error: its id, driver, and attach
// time. The fold checks the id and the driver, but the attach time is the
// event's ts as written, which a hand edit can fill with control characters,
// so it is escaped like inbox text before it reaches a terminal.
func (r DirectorRegistration) describe() string {
	return fmt.Sprintf("%s (driver %s, attached %s)", r.ID, r.Driver, PrintableInboxText(r.AttachedAt, false))
}

// DirectorReadResult is the folded director.jsonl: the current registration
// (nil when none), the number of lines that did not parse (skipped, like the
// inbox's corrupt lines), and the number of parsed events the fold ignored:
// an unknown kind, an id not of the form d<N>, an attached with a driver
// other than claude and codex, an attached whose id an earlier attached
// already used, an attached while another registration is current, and a
// detached for an id that is not the current registration's.
type DirectorReadResult struct {
	Current       *DirectorRegistration
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
// error. Read takes no lock; Attach and Detach decide under the lock.
func (s *DirectorStore) Read() (DirectorReadResult, error) {
	raw, err := s.readRaw()
	if err != nil {
		return DirectorReadResult{}, err
	}
	return foldDirector(raw).result(), nil
}

// Attach records a new registration with driver a.Driver and wake pull, and
// returns it. Under the director lock it reads the file again; while a
// registration is current it returns ErrDirectorAttached and writes nothing,
// unless a.Replace is set: then it appends the current registration's
// detached and then the new attached in the same lock hold, and returns the
// ended registration as replaced. The new id is one more than the largest
// d<N> anywhere in the file, damaged lines included. A driver other than
// claude and codex is refused before anything is read or written.
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
			return fmt.Errorf("org: director: attach: %w: %s; detach it first, or attach with replace", ErrDirectorAttached, cur.describe())
		}
		next, err := fold.nextID()
		if err != nil {
			return err
		}
		ts := s.now()
		if cur != nil {
			ended := DirectorEvent{
				TS: ts, ID: cur.ID, Event: DirectorEventDetached,
				User: a.User, Cwd: a.Cwd, HerdrPane: a.HerdrPane,
			}
			if err := s.appendLocked(raw, ended); err != nil {
				return err
			}
			// The detached line ended with a newline, so the attached line
			// needs none in front.
			raw = nil
		}
		ev := DirectorEvent{
			TS: ts, ID: next, Event: DirectorEventAttached,
			Driver: a.Driver, Wake: []string{DirectorWakePull},
			User: a.User, Cwd: a.Cwd, HerdrPane: a.HerdrPane,
		}
		if err := s.appendLocked(raw, ev); err != nil {
			if cur != nil {
				return fmt.Errorf("org: director: attach: %s was detached, but the new registration was not recorded: %w", cur.describe(), err)
			}
			return err
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
// creating the state dir or the lock.
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
			return err
		}
		ended = *cur
		return nil
	})
	if err != nil {
		return DirectorRegistration{}, err
	}
	return ended, nil
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

// appendLocked appends ev as one line in a single write, as the inbox's
// appendLocked does. raw is the file as read under the same lock hold. When
// raw does not end with a newline (a write cut off mid-line), the write
// starts with one, so the cut line stays a separate corrupt line and ev
// stays readable.
func (s *DirectorStore) appendLocked(raw []byte, ev DirectorEvent) error {
	line, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("org: director: marshal the %s event: %w", ev.Event, err)
	}
	if len(raw) > 0 && raw[len(raw)-1] != '\n' {
		line = append([]byte{'\n'}, line...)
	}
	line = append(line, '\n')
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("org: director: open %s: %w", s.path, err)
	}
	if _, err := f.Write(line); err != nil {
		_ = f.Close()
		return fmt.Errorf("org: director: append to %s: %w", s.path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("org: director: close %s: %w", s.path, err)
	}
	return nil
}

// registrationOf returns the registration an attached event records.
func registrationOf(ev DirectorEvent) DirectorRegistration {
	return DirectorRegistration{
		ID: ev.ID, Driver: ev.Driver, Wake: ev.Wake, PaneID: ev.PaneID, AttachedAt: ev.TS,
		User: ev.User, Cwd: ev.Cwd, HerdrPane: ev.HerdrPane,
	}
}

// directorFold is director.jsonl folded from its raw text.
type directorFold struct {
	current *DirectorRegistration
	// used holds every id an applied attached event has had, so an
	// attached that repeats one is ignored.
	used    map[string]bool
	corrupt int
	ignored int
	// maxN is the largest d<N> found anywhere in the file: in the raw text
	// (directorRawIDPattern) and in the decoded id of every parsed line, so
	// an id written with JSON escapes counts too.
	maxN int64
}

func foldDirector(raw []byte) *directorFold {
	f := &directorFold{used: map[string]bool{}}
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
	switch ev.Event {
	case DirectorEventAttached:
		// Attach writes the current registration's detached before a new
		// attached, so only a hand edit writes any of these.
		if (ev.Driver != DirectorDriverClaude && ev.Driver != DirectorDriverCodex) || f.used[ev.ID] || f.current != nil {
			f.ignored++
			return
		}
		f.used[ev.ID] = true
		reg := registrationOf(ev)
		f.current = &reg
	case DirectorEventDetached:
		if f.current == nil || ev.ID != f.current.ID {
			f.ignored++
			return
		}
		f.current = nil
	default:
		f.ignored++
	}
}

func (f *directorFold) nextID() (string, error) {
	if f.maxN == math.MaxInt64 {
		return "", fmt.Errorf("org: director: no registration id is left after d%d", f.maxN)
	}
	return "d" + strconv.FormatInt(f.maxN+1, 10), nil
}

func (f *directorFold) result() DirectorReadResult {
	return DirectorReadResult{Current: f.current, CorruptLines: f.corrupt, IgnoredEvents: f.ignored}
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
