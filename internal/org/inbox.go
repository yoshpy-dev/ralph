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
	"unicode"
	"unicode/utf8"
)

// The inbox is where a leader raises an item for the director (FR-5 of
// docs/specs/2026-10-07-org-multi-org-director.md): inbox.jsonl in the
// shared org state dir, next to the manifest. It is append-only; an item's
// state is folded from its events (open -> acked -> resolved). Writes take
// inbox.lock, a lock separate from the manifest's: no inbox write touches
// the manifest, so no path holds both locks.

// Inbox event kinds: the "event" field of an inbox.jsonl line.
const (
	InboxEventEscalated = "escalated"
	InboxEventNotified  = "notified"
	InboxEventAcked     = "acked"
	InboxEventResolved  = "resolved"
)

// Inbox item states, folded from the events. notified records that the
// item was sent to the human path and changes no state.
const (
	InboxStateOpen     = "open"
	InboxStateAcked    = "acked"
	InboxStateResolved = "resolved"
)

// InboxNoteMaxRunes is the longest resolve note ValidateInboxNote accepts,
// in characters (runes).
const InboxNoteMaxRunes = 500

// inboxLockFile is the lock file every inbox write takes, in the same
// directory as inbox.jsonl.
const inboxLockFile = "inbox.lock"

// ErrInboxUnknownID is returned (wrapped) by Ack, Resolve, and
// AppendNotified when the inbox has no item with the given ID.
var ErrInboxUnknownID = errors.New("no such inbox item")

// ErrInboxResolved is returned (wrapped) by Ack and Resolve on an item that
// is already resolved.
var ErrInboxResolved = errors.New("the item is already resolved")

// inboxRawIDPattern finds an "id":"e<N>" in the raw text of inbox.jsonl,
// including lines that do not parse, so the next ID is never one a damaged
// line already used (Codex plan advisory finding 1). It does not require
// the closing quote: a line cut off inside the digits still counts.
var inboxRawIDPattern = regexp.MustCompile(`"id"\s*:\s*"e([0-9]+)`)

// InboxEvent is one line of inbox.jsonl. Every line carries ts, id, and
// event; the other fields belong to one kind each. escalated carries
// org_id, type, task_id (when the message has one), and body; notified
// carries reason and osascript (the result of the macOS notification, e.g.
// "ok", "failed: ...", "skipped"); resolved carries note; acked carries
// nothing more.
type InboxEvent struct {
	TS        string `json:"ts"`    // UTC RFC3339
	ID        string `json:"id"`    // e<N>
	Event     string `json:"event"` // one of the InboxEvent* constants
	OrgID     string `json:"org_id,omitempty"`
	Type      string `json:"type,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
	Body      string `json:"body,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Osascript string `json:"osascript,omitempty"`
	Note      string `json:"note,omitempty"`
}

// InboxEscalation is what Escalate records: the org that raised the item,
// the message's TYPE and TASK_ID (empty when it has none), and the whole
// message text as Body. Escalate does not check the message against
// internal/org/protocol; its caller does.
type InboxEscalation struct {
	OrgID  string
	Type   string
	TaskID string
	Body   string
}

// InboxItem is one item folded from its events. Events holds the events
// that were applied to it, in file order, for `ralph org inbox show`.
// NotifiedAt is the time of the latest notified event.
type InboxItem struct {
	ID          string       `json:"id"`
	OrgID       string       `json:"org_id"`
	Type        string       `json:"type"`
	TaskID      string       `json:"task_id,omitempty"`
	Body        string       `json:"body"`
	State       string       `json:"state"`
	EscalatedAt string       `json:"escalated_at"`
	AckedAt     string       `json:"acked_at,omitempty"`
	ResolvedAt  string       `json:"resolved_at,omitempty"`
	Note        string       `json:"note,omitempty"`
	Notified    bool         `json:"notified"`
	NotifiedAt  string       `json:"notified_at,omitempty"`
	Events      []InboxEvent `json:"events"`
}

// InboxReadResult is the folded inbox: the items in ID order (e2 before
// e10), the number of lines that did not parse (skipped, like the
// manifest's corrupt lines), and the number of parsed events the fold
// ignored: an unknown kind, an ID not of the form e<N>, an event for an ID
// with no escalated event, a second escalated for an ID, and a transition
// the item's state does not allow (acked on an item that is not open,
// resolved on a resolved item).
type InboxReadResult struct {
	Items         []InboxItem
	CorruptLines  int
	IgnoredEvents int
}

// InboxStore reads and appends to inbox.jsonl.
type InboxStore struct {
	path string
	// Now is the clock for event timestamps; nil means time.Now.
	Now func() time.Time
}

// InboxPathIn returns the inbox.jsonl path within an already-resolved org
// state directory, mirroring ManifestPathIn.
func InboxPathIn(stateDir string) string {
	return filepath.Join(stateDir, "inbox.jsonl")
}

// NewInboxStore returns the InboxStore of the resolved org state dir
// stateDir. Its lock file is <stateDir>/inbox.lock.
func NewInboxStore(stateDir string) *InboxStore {
	return &InboxStore{path: InboxPathIn(stateDir)}
}

// Path returns the inbox.jsonl path this store reads and appends to.
func (s *InboxStore) Path() string {
	return s.path
}

// Read folds every event in the inbox. A missing file (or state dir) reads
// as an empty inbox, not an error. Read takes no lock; a decision that
// depends on an item's state is made under the lock by Ack and Resolve.
func (s *InboxStore) Read() (InboxReadResult, error) {
	raw, err := s.readRaw()
	if err != nil {
		return InboxReadResult{}, err
	}
	return foldInbox(raw).result(), nil
}

// Escalate records a new item: under the inbox lock it allocates the next
// ID (one more than the largest e<N> anywhere in the file, damaged lines
// included) and appends the escalated event. It returns the new ID.
func (s *InboxStore) Escalate(in InboxEscalation) (string, error) {
	for _, field := range []struct{ name, value string }{{"org_id", in.OrgID}, {"type", in.Type}, {"body", in.Body}} {
		if strings.TrimSpace(field.value) == "" {
			return "", fmt.Errorf("org: inbox: escalate needs a non-empty %s", field.name)
		}
	}
	var id string
	err := s.withLock(func() error {
		raw, err := s.readRaw()
		if err != nil {
			return err
		}
		next, err := foldInbox(raw).nextID()
		if err != nil {
			return err
		}
		ev := InboxEvent{
			TS: s.now(), ID: next, Event: InboxEventEscalated,
			OrgID: in.OrgID, Type: in.Type, TaskID: in.TaskID, Body: in.Body,
		}
		if err := s.appendLocked(raw, ev); err != nil {
			return err
		}
		id = next
		return nil
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

// AppendNotified records that item id was sent to the human path, with the
// reason it was sent and the osascript result. It changes no state, and it
// is recorded for an item in any state.
func (s *InboxStore) AppendNotified(id, reason, osascript string) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("org: inbox: notify %s: the notified event needs a reason", id)
	}
	return s.update("notify", id, func(*InboxItem) (*InboxEvent, error) {
		return &InboxEvent{Event: InboxEventNotified, Reason: reason, Osascript: osascript}, nil
	})
}

// Ack moves item id from open to acked and reports changed=true. On an
// acked item it writes nothing and reports changed=false with no error; on
// a resolved item it returns ErrInboxResolved. The state is read under the
// inbox lock.
func (s *InboxStore) Ack(id string) (changed bool, err error) {
	err = s.update("ack", id, func(item *InboxItem) (*InboxEvent, error) {
		switch item.State {
		case InboxStateOpen:
			changed = true
			return &InboxEvent{Event: InboxEventAcked}, nil
		case InboxStateAcked:
			return nil, nil
		default:
			return nil, fmt.Errorf("org: inbox: ack %s: %w", id, ErrInboxResolved)
		}
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

// Resolve moves item id from open or acked to resolved, recording note
// (checked by ValidateInboxNote before anything is read). On a resolved
// item it returns ErrInboxResolved. The state is read under the inbox lock.
func (s *InboxStore) Resolve(id, note string) error {
	if err := ValidateInboxNote(note); err != nil {
		return err
	}
	return s.update("resolve", id, func(item *InboxItem) (*InboxEvent, error) {
		if item.State == InboxStateResolved {
			return nil, fmt.Errorf("org: inbox: resolve %s: %w", id, ErrInboxResolved)
		}
		return &InboxEvent{Event: InboxEventResolved, Note: note}, nil
	})
}

// ValidateInboxNote checks a resolve note: required (not blank), a single
// line, valid UTF-8, at most InboxNoteMaxRunes characters, and no control
// characters. The error names the rule that failed.
func ValidateInboxNote(note string) error {
	switch {
	case strings.TrimSpace(note) == "":
		return errors.New("org: inbox: the resolve note is required: give a pointer such as a report path or a PR URL")
	case strings.IndexFunc(note, isInboxLineBreak) >= 0:
		return errors.New("org: inbox: the resolve note must be a single line")
	case !utf8.ValidString(note):
		return errors.New("org: inbox: the resolve note is not valid UTF-8")
	}
	if n := utf8.RuneCountInString(note); n > InboxNoteMaxRunes {
		return fmt.Errorf("org: inbox: the resolve note is %d characters; at most %d", n, InboxNoteMaxRunes)
	}
	if i := strings.IndexFunc(note, unicode.IsControl); i >= 0 {
		r, _ := utf8.DecodeRuneInString(note[i:])
		return fmt.Errorf("org: inbox: the resolve note contains a control character (%U)", r)
	}
	return nil
}

// isInboxLineBreak reports the runes that end a line: LF, CR, and the
// Unicode line and paragraph separators (U+2028, U+2029).
func isInboxLineBreak(r rune) bool {
	return r == '\n' || r == '\r' || r == 0x2028 || r == 0x2029
}

// update is the read-decide-append step of AppendNotified, Ack, and
// Resolve. Under the inbox lock it reads the inbox again, looks up id
// (ErrInboxUnknownID when absent), and appends the event decide returns;
// decide returning nil writes nothing. A missing inbox file has no items,
// so it returns ErrInboxUnknownID without creating the state dir or the
// lock.
func (s *InboxStore) update(verb, id string, decide func(item *InboxItem) (*InboxEvent, error)) error {
	unknown := fmt.Errorf("org: inbox: %s %s: %w", verb, id, ErrInboxUnknownID)
	if _, err := os.Stat(s.path); errors.Is(err, fs.ErrNotExist) {
		return unknown
	}
	return s.withLock(func() error {
		raw, err := s.readRaw()
		if err != nil {
			return err
		}
		item := foldInbox(raw).items[id]
		if item == nil {
			return unknown
		}
		ev, err := decide(item)
		if err != nil || ev == nil {
			return err
		}
		ev.TS, ev.ID = s.now(), id
		return s.appendLocked(raw, *ev)
	})
}

func (s *InboxStore) withLock(fn func() error) error {
	return withFileLock(filepath.Dir(s.path), inboxLockFile, "inbox", fn)
}

func (s *InboxStore) now() string {
	nowFn := time.Now
	if s.Now != nil {
		nowFn = s.Now
	}
	return nowFn().UTC().Format(time.RFC3339)
}

// readRaw returns the whole inbox file; a missing file reads as empty.
func (s *InboxStore) readRaw() ([]byte, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("org: inbox: read %s: %w", s.path, err)
	}
	return raw, nil
}

// appendLocked appends ev as one line in a single write. raw is the file
// as read under the same lock hold. When raw does not end with a newline (a
// write cut off mid-line), the write starts with one, so the cut line stays
// a separate corrupt line and ev stays readable (Codex plan advisory
// finding 1).
func (s *InboxStore) appendLocked(raw []byte, ev InboxEvent) error {
	line, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("org: inbox: marshal the %s event: %w", ev.Event, err)
	}
	if len(raw) > 0 && raw[len(raw)-1] != '\n' {
		line = append([]byte{'\n'}, line...)
	}
	line = append(line, '\n')
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("org: inbox: open %s: %w", s.path, err)
	}
	if _, err := f.Write(line); err != nil {
		_ = f.Close()
		return fmt.Errorf("org: inbox: append to %s: %w", s.path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("org: inbox: close %s: %w", s.path, err)
	}
	return nil
}

// inboxFold is the inbox folded from its raw text.
type inboxFold struct {
	items   map[string]*InboxItem
	corrupt int
	ignored int
	// maxN is the largest e<N> found anywhere in the file: in the raw text
	// (inboxRawIDPattern) and in the decoded id of every parsed line, so an
	// id written with JSON escapes counts too.
	maxN int64
}

func foldInbox(raw []byte) *inboxFold {
	f := &inboxFold{items: map[string]*InboxItem{}}
	for _, m := range inboxRawIDPattern.FindAllSubmatch(raw, -1) {
		f.seeN(string(m[1]))
	}
	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var ev InboxEvent
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
func (f *inboxFold) seeN(digits string) {
	if n, err := strconv.ParseInt(digits, 10, 64); err == nil && n > f.maxN {
		f.maxN = n
	}
}

func (f *inboxFold) apply(ev InboxEvent) {
	if digits, ok := strings.CutPrefix(ev.ID, "e"); ok {
		f.seeN(digits)
	}
	if _, ok := inboxIDNumber(ev.ID); !ok {
		f.ignored++
		return
	}
	item := f.items[ev.ID]
	switch ev.Event {
	case InboxEventEscalated:
		if item != nil {
			f.ignored++
			return
		}
		f.items[ev.ID] = &InboxItem{
			ID: ev.ID, OrgID: ev.OrgID, Type: ev.Type, TaskID: ev.TaskID, Body: ev.Body,
			State: InboxStateOpen, EscalatedAt: ev.TS, Events: []InboxEvent{ev},
		}
		return
	case InboxEventNotified, InboxEventAcked, InboxEventResolved:
	default:
		f.ignored++
		return
	}
	if item == nil {
		f.ignored++
		return
	}
	switch ev.Event {
	case InboxEventNotified:
		item.Notified = true
		item.NotifiedAt = ev.TS
	case InboxEventAcked:
		if item.State != InboxStateOpen {
			f.ignored++
			return
		}
		item.State = InboxStateAcked
		item.AckedAt = ev.TS
	case InboxEventResolved:
		if item.State == InboxStateResolved {
			f.ignored++
			return
		}
		item.State = InboxStateResolved
		item.ResolvedAt = ev.TS
		item.Note = ev.Note
	}
	item.Events = append(item.Events, ev)
}

func (f *inboxFold) nextID() (string, error) {
	if f.maxN == math.MaxInt64 {
		return "", fmt.Errorf("org: inbox: no item ID is left after e%d", f.maxN)
	}
	return "e" + strconv.FormatInt(f.maxN+1, 10), nil
}

func (f *inboxFold) result() InboxReadResult {
	items := make([]InboxItem, 0, len(f.items))
	for _, item := range f.items {
		items = append(items, *item)
	}
	sort.Slice(items, func(i, j int) bool {
		ni, _ := inboxIDNumber(items[i].ID)
		nj, _ := inboxIDNumber(items[j].ID)
		return ni < nj
	})
	return InboxReadResult{Items: items, CorruptLines: f.corrupt, IgnoredEvents: f.ignored}
}

// inboxIDNumber returns N for an item ID e<N> written the way nextID writes
// it: N at least 1, no leading zero, within int64. Any other ID (e01, e0,
// ex, E1) names no item.
func inboxIDNumber(id string) (int64, bool) {
	if !strings.HasPrefix(id, "e") {
		return 0, false
	}
	n, err := strconv.ParseInt(id[1:], 10, 64)
	if err != nil || n < 1 || "e"+strconv.FormatInt(n, 10) != id {
		return 0, false
	}
	return n, true
}
