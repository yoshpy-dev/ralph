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

// The follow-up record (issue #270; plan
// docs/plans/active/2026-10-30-org-followups.md, D1, D2, and D5):
// followups.jsonl in the shared org state dir, next to the manifest, the
// inbox, and the director registration. A follow-up is an issue the director
// asks the human about only once the PR of another branch, its after-branch,
// is merged. It belongs to the whole ledger, not to one org, so it is not a
// manifest line. It is append-only, as the inbox is; a follow-up's state is
// folded from its events (open -> done). Writes take followups.lock, a lock
// of its own: no follow-up write touches another record, so no path holds it
// together with the manifest's, the inbox's, or the director's lock.

// Follow-up event kinds: the "event" field of a followups.jsonl line.
const (
	FollowupEventAdded = "added"
	FollowupEventDone  = "done"
)

// Follow-up states, folded from the events.
const (
	FollowupStateOpen = "open"
	FollowupStateDone = "done"
)

// followupLockFile is the lock file every follow-up write takes, in the same
// directory as followups.jsonl.
const followupLockFile = "followups.lock"

// ErrFollowupUnknownID is returned (wrapped) by Done when the record has no
// follow-up with the given ID.
var ErrFollowupUnknownID = errors.New("no such follow-up")

// followupRawIDPattern finds an "id":"f<N>" in the raw text of
// followups.jsonl, including lines that do not parse, so the next ID is never
// one a damaged line already used, as inboxRawIDPattern does for the inbox.
// It does not require the closing quote: a line cut off inside the digits
// still counts.
var followupRawIDPattern = regexp.MustCompile(`"id"\s*:\s*"f([0-9]+)`)

// FollowupEvent is one line of followups.jsonl. Every line carries ts, id,
// and event. added carries issue, after_branch, and note (none when the note
// is empty); done carries note (none when empty).
type FollowupEvent struct {
	TS          string `json:"ts"`    // UTC RFC3339
	ID          string `json:"id"`    // f<N>
	Event       string `json:"event"` // one of the FollowupEvent* constants
	Issue       int    `json:"issue,omitempty"`
	AfterBranch string `json:"after_branch,omitempty"`
	Note        string `json:"note,omitempty"`
}

// FollowupAdd is what Add records: the number of the issue the director
// waits to ask the human about, the branch whose PR it waits for, and an
// optional note. ValidateFollowupAdd has the rules.
type FollowupAdd struct {
	Issue       int
	AfterBranch string
	Note        string
}

// Followup is one follow-up folded from its events. Note is its added
// event's note; AddedAt is that event's ts. DoneAt and DoneNote are its done
// event's ts and note, empty while it is open.
type Followup struct {
	ID          string `json:"id"`
	Issue       int    `json:"issue"`
	AfterBranch string `json:"after_branch"`
	Note        string `json:"note,omitempty"`
	State       string `json:"state"`
	AddedAt     string `json:"added_at"`
	DoneAt      string `json:"done_at,omitempty"`
	DoneNote    string `json:"done_note,omitempty"`
}

// FollowupReadResult is the folded record: the follow-ups in ID order (f2
// before f10), the number of lines that did not parse (skipped, like the
// inbox's corrupt lines), and the number of parsed events the fold ignored:
// an unknown kind, an ID not of the form f<N>, a second added for an ID, an
// added whose issue or after-branch Add would refuse (ValidateFollowupAdd), a
// done for an ID with no added, and a done for a follow-up that is already
// done.
type FollowupReadResult struct {
	Items         []Followup
	CorruptLines  int
	IgnoredEvents int
}

// FollowupStore reads and appends to followups.jsonl.
type FollowupStore struct {
	path string
	// Now is the clock for event timestamps; nil means time.Now.
	Now func() time.Time
}

// FollowupsPathIn returns the followups.jsonl path within an
// already-resolved org state directory, mirroring InboxPathIn.
func FollowupsPathIn(stateDir string) string {
	return filepath.Join(stateDir, "followups.jsonl")
}

// NewFollowupStore returns the FollowupStore of the resolved org state dir
// stateDir. Its lock file is <stateDir>/followups.lock.
func NewFollowupStore(stateDir string) *FollowupStore {
	return &FollowupStore{path: FollowupsPathIn(stateDir)}
}

// Path returns the followups.jsonl path this store reads and appends to.
func (s *FollowupStore) Path() string {
	return s.path
}

// Read folds every event in the record. A missing file (or state dir) reads
// as no follow-ups, not an error, and creates nothing. Read takes no lock; a
// decision that depends on a follow-up's state is made under the lock by
// Done.
func (s *FollowupStore) Read() (FollowupReadResult, error) {
	raw, err := s.readRaw()
	if err != nil {
		return FollowupReadResult{}, err
	}
	return foldFollowups(raw).result(), nil
}

// Add records a new follow-up: it runs ValidateFollowupAdd on in before
// anything is read, then under the follow-up lock allocates the next ID (one
// more than the largest f<N> anywhere in the file, damaged lines included)
// and appends the added event. It returns the new ID.
func (s *FollowupStore) Add(in FollowupAdd) (string, error) {
	if err := ValidateFollowupAdd(in); err != nil {
		return "", err
	}
	var id string
	err := s.withLock(func() error {
		raw, err := s.readRaw()
		if err != nil {
			return err
		}
		next, err := foldFollowups(raw).nextID()
		if err != nil {
			return err
		}
		ev := FollowupEvent{TS: s.now(), ID: next, Event: FollowupEventAdded, Issue: in.Issue, AfterBranch: in.AfterBranch, Note: in.Note}
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

// Done moves follow-up id from open to done, recording note (optional;
// checked by ValidateFollowupNote before anything is read), and reports
// changed=true. On a follow-up that is already done it writes nothing and
// reports changed=false with no error, as the inbox's Ack does on an acked
// item; the first done's note stays. The state is read under the follow-up
// lock. A missing file has no follow-ups, so it returns ErrFollowupUnknownID
// without creating the state dir or the lock.
func (s *FollowupStore) Done(id, note string) (changed bool, err error) {
	if err := ValidateFollowupNote(note); err != nil {
		return false, err
	}
	unknown := fmt.Errorf("org: followup: done %s: %w", id, ErrFollowupUnknownID)
	if _, err := os.Stat(s.path); errors.Is(err, fs.ErrNotExist) {
		return false, unknown
	}
	err = s.withLock(func() error {
		raw, err := s.readRaw()
		if err != nil {
			return err
		}
		item := foldFollowups(raw).items[id]
		if item == nil {
			return unknown
		}
		if item.State == FollowupStateDone {
			return nil
		}
		changed = true
		return s.appendLocked(raw, FollowupEvent{TS: s.now(), ID: id, Event: FollowupEventDone, Note: note})
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

// ValidateFollowupAdd checks what Add records (plan D3): the issue is a
// positive number; the after-branch is not empty, is valid UTF-8, holds no
// whitespace or control character, and does not start with "-" (it is
// matched as a string against the ledger's branch= and passed to gh pr list
// --head, so git's ref-format rules are not checked); the note is optional,
// and when given follows the rules of an inbox resolve note
// (ValidateInboxNote): not blank, a single line, valid UTF-8, at most
// InboxNoteMaxRunes characters, and no control characters. The error names
// the rule that failed.
func ValidateFollowupAdd(in FollowupAdd) error {
	if err := validateFollowupIssue(in.Issue); err != nil {
		return err
	}
	if err := validateFollowupBranch(in.AfterBranch); err != nil {
		return err
	}
	if in.Note == "" {
		return nil
	}
	return validateFollowupNote(in.Note)
}

func validateFollowupIssue(issue int) error {
	if issue < 1 {
		return fmt.Errorf("org: followup: the issue must be a positive number, not %d", issue)
	}
	return nil
}

func validateFollowupBranch(branch string) error {
	switch {
	case branch == "":
		return errors.New("org: followup: the after-branch is required")
	case !utf8.ValidString(branch):
		return errors.New("org: followup: the after-branch is not valid UTF-8")
	case strings.HasPrefix(branch, "-"):
		return fmt.Errorf("org: followup: the after-branch %q starts with -", branch)
	}
	if i := strings.IndexFunc(branch, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }); i >= 0 {
		r, _ := utf8.DecodeRuneInString(branch[i:])
		return fmt.Errorf("org: followup: the after-branch %q contains whitespace or a control character (%U)", branch, r)
	}
	return nil
}

// ValidateFollowupNote checks the optional note of a done as Done checks it:
// "" is no note and passes; any other note follows the rules of an add's
// note (validateFollowupNote). ralph org followup done runs it before it
// resolves the ledger, as add runs ValidateFollowupAdd, so a refused note is
// reported before any note or refusal about the ledger.
func ValidateFollowupNote(note string) error {
	if note == "" {
		return nil
	}
	return validateFollowupNote(note)
}

// validateFollowupNote checks a given note with the rules of ValidateInboxNote,
// in messages that name the follow-up note.
func validateFollowupNote(note string) error {
	switch {
	case strings.TrimSpace(note) == "":
		return errors.New("org: followup: the note is blank: give some text or leave it out")
	case strings.IndexFunc(note, isInboxLineBreak) >= 0:
		return errors.New("org: followup: the note must be a single line")
	case !utf8.ValidString(note):
		return errors.New("org: followup: the note is not valid UTF-8")
	}
	if n := utf8.RuneCountInString(note); n > InboxNoteMaxRunes {
		return fmt.Errorf("org: followup: the note is %d characters; at most %d", n, InboxNoteMaxRunes)
	}
	if i := strings.IndexFunc(note, unicode.IsControl); i >= 0 {
		r, _ := utf8.DecodeRuneInString(note[i:])
		return fmt.Errorf("org: followup: the note contains a control character (%U)", r)
	}
	return nil
}

func (s *FollowupStore) withLock(fn func() error) error {
	return withFileLock(filepath.Dir(s.path), followupLockFile, "followup", fn)
}

func (s *FollowupStore) now() string {
	nowFn := time.Now
	if s.Now != nil {
		nowFn = s.Now
	}
	return nowFn().UTC().Format(time.RFC3339)
}

// readRaw returns the whole record file; a missing file reads as empty.
func (s *FollowupStore) readRaw() ([]byte, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("org: followup: read %s: %w", s.path, err)
	}
	return raw, nil
}

// appendLocked appends ev as one line in a single write. raw is the file as
// read under the same lock hold. When raw does not end with a newline (a
// write cut off mid-line), the write starts with one, so the cut line stays a
// separate corrupt line and the new event stays readable, as the inbox's
// appendLocked does.
func (s *FollowupStore) appendLocked(raw []byte, ev FollowupEvent) error {
	var lines []byte
	if len(raw) > 0 && raw[len(raw)-1] != '\n' {
		lines = append(lines, '\n')
	}
	line, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("org: followup: marshal the %s event: %w", ev.Event, err)
	}
	lines = append(append(lines, line...), '\n')
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("org: followup: open %s: %w", s.path, err)
	}
	if _, err := f.Write(lines); err != nil {
		_ = f.Close()
		return fmt.Errorf("org: followup: append to %s: %w", s.path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("org: followup: close %s: %w", s.path, err)
	}
	return nil
}

// followupFold is the record folded from its raw text.
type followupFold struct {
	items   map[string]*Followup
	corrupt int
	ignored int
	// maxN is the largest f<N> found anywhere in the file: in the raw text
	// (followupRawIDPattern) and in the decoded id of every parsed line, so
	// an id written with JSON escapes counts too.
	maxN int64
}

func foldFollowups(raw []byte) *followupFold {
	f := &followupFold{items: map[string]*Followup{}}
	for _, m := range followupRawIDPattern.FindAllSubmatch(raw, -1) {
		f.seeN(string(m[1]))
	}
	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var ev FollowupEvent
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
func (f *followupFold) seeN(digits string) {
	if n, err := strconv.ParseInt(digits, 10, 64); err == nil && n > f.maxN {
		f.maxN = n
	}
}

func (f *followupFold) apply(ev FollowupEvent) {
	if digits, ok := strings.CutPrefix(ev.ID, "f"); ok {
		f.seeN(digits)
	}
	if _, ok := followupIDNumber(ev.ID); !ok {
		f.ignored++
		return
	}
	item := f.items[ev.ID]
	switch ev.Event {
	case FollowupEventAdded:
		// Only an added that Add could have written makes a follow-up, so
		// every folded follow-up names a positive issue and an after-branch
		// that is safe to match and to pass to gh (a hand edit could write
		// one that starts with "-"). Its note is not checked: callers print
		// it with PrintableInboxText.
		if item != nil || validateFollowupIssue(ev.Issue) != nil || validateFollowupBranch(ev.AfterBranch) != nil {
			f.ignored++
			return
		}
		f.items[ev.ID] = &Followup{
			ID: ev.ID, Issue: ev.Issue, AfterBranch: ev.AfterBranch, Note: ev.Note, State: FollowupStateOpen, AddedAt: ev.TS,
		}
	case FollowupEventDone:
		if item == nil || item.State == FollowupStateDone {
			f.ignored++
			return
		}
		item.State = FollowupStateDone
		item.DoneAt = ev.TS
		item.DoneNote = ev.Note
	default:
		f.ignored++
	}
}

func (f *followupFold) nextID() (string, error) {
	if f.maxN == math.MaxInt64 {
		return "", fmt.Errorf("org: followup: no follow-up ID is left after f%d", f.maxN)
	}
	return "f" + strconv.FormatInt(f.maxN+1, 10), nil
}

func (f *followupFold) result() FollowupReadResult {
	items := make([]Followup, 0, len(f.items))
	for _, item := range f.items {
		items = append(items, *item)
	}
	sort.Slice(items, func(i, j int) bool {
		ni, _ := followupIDNumber(items[i].ID)
		nj, _ := followupIDNumber(items[j].ID)
		return ni < nj
	})
	return FollowupReadResult{Items: items, CorruptLines: f.corrupt, IgnoredEvents: f.ignored}
}

// followupIDNumber returns N for a follow-up ID f<N> written the way nextID
// writes it: N at least 1, no leading zero, within int64. Any other ID (f01,
// f0, fx, F1) names no follow-up.
func followupIDNumber(id string) (int64, bool) {
	if !strings.HasPrefix(id, "f") {
		return 0, false
	}
	n, err := strconv.ParseInt(id[1:], 10, 64)
	if err != nil || n < 1 || "f"+strconv.FormatInt(n, 10) != id {
		return 0, false
	}
	return n, true
}

// FollowupRecordedMerge reports whether the manifest events record a merge
// of f's after-branch since f was added (plan D2 and D5): a pr_merged
// (EventPRMerged, which ralph org merge writes) or a worktree_cleaned
// (EventWorktreeCleaned, which the cleanup of a merged feature worktree
// writes) whose branch is f.AfterBranch exactly and whose ts is at or after
// f.AddedAt. ralph writes both only once gh read the PR as merged. A branch
// name can be used again, so an event from before f was added does not
// count. A worktree_cleaned's ts is when the cleanup ran, not when the PR
// merged, so one whose head is the head of a pr_merged of the branch from
// before f was added does not count either: that merge came before f, and
// only its cleanup after it (ralph org merge's own cleanup was refused, and
// ralph org cleanup ran once f was added). A worktree_cleaned with another
// head, the branch used again, counts, and so does one of a PR merged on
// GitHub before f was added, which has no pr_merged to tell. The times are
// compared as instants; an event whose ts does not parse as RFC3339 is
// skipped, and an f whose AddedAt does not parse (FollowupAddedTime) is
// never merged, so the record never reads as ready on a time it cannot read.
// at and event are the ts and the event name of the first such event in file
// order; all three are zero when there is none. It reads only events, so
// ralph status can call it without gh.
func FollowupRecordedMerge(events []ManifestEvent, f Followup) (merged bool, at, event string) {
	added, unreadable := FollowupAddedTime(f)
	if unreadable != "" {
		return false, "", ""
	}
	// mergedBefore holds the heads of the branch's pr_merged from before f
	// was added; since, the branch's events at or after it, in file order.
	mergedBefore := make(map[string]bool)
	var since []ManifestEvent
	for _, ev := range events {
		if ev.Event != EventPRMerged && ev.Event != EventWorktreeCleaned {
			continue
		}
		if branch, ok := recordedMergeBranch(ev.Details); !ok || branch != f.AfterBranch {
			continue
		}
		ts, err := time.Parse(time.RFC3339, ev.TS)
		switch {
		case err != nil:
		case !ts.Before(added):
			since = append(since, ev)
		case ev.Event == EventPRMerged:
			if head, ok := recordedMergeHead(ev.Details); ok {
				mergedBefore[head] = true
			}
		}
	}
	for _, ev := range since {
		if head, ok := recordedMergeHead(ev.Details); ok && ev.Event == EventWorktreeCleaned && mergedBefore[head] {
			continue
		}
		return true, ev.TS, ev.Event
	}
	return false, "", ""
}

// FollowupAddedTime is f.AddedAt, the time f was recorded, read as RFC3339.
// When it does not parse (a hand edit), unreadable is the detail that ralph
// status and ralph org merge-order both give f's unknown state, since no
// merge can be compared with it; "" otherwise.
func FollowupAddedTime(f Followup) (added time.Time, unreadable string) {
	added, err := time.Parse(time.RFC3339, f.AddedAt)
	if err != nil {
		return time.Time{}, fmt.Sprintf("the follow-up's added_at %q is not an RFC3339 time, so no merge can be compared with it",
			f.AddedAt)
	}
	return added, ""
}

// FollowupReadyDetail is the detail of a ready follow-up f in the words
// ralph status and ralph org merge-order share: merge, the clause that names
// the merge ("PR #7 was merged", "the ledger records pr_merged"), at its
// time at, then that the human can be asked about f's issue now.
func FollowupReadyDetail(f Followup, merge, at string) string {
	return fmt.Sprintf("%s at %s, after the follow-up was recorded at %s: ask the human about issue #%d now, then run ralph org "+
		"followup done %s", merge, at, f.AddedAt, f.Issue, f.ID)
}

// recordedMergeBranch reads the branch from the Details of a pr_merged or a
// worktree_cleaned, whose first token is branch=<branch> (merge_run.go's
// recordMerge, cleanupDetails in cleanup.go). Only the first token is read: a
// later one may be part of the quoted worktree of a worktree_cleaned, which
// can hold spaces. ok is false when the first token is not a non-empty
// branch=.
func recordedMergeBranch(details string) (branch string, ok bool) {
	first, _, _ := strings.Cut(details, " ")
	branch, found := strings.CutPrefix(first, "branch=")
	return branch, found && branch != ""
}

// recordedMergeHead reads the head from the Details of a pr_merged or a
// worktree_cleaned: the value of the first head= token (the third token of
// recordMerge's Details, the second of cleanupDetails'). As
// gateRecordFromDetails reads a gate record, only the tokens before the
// quoted worktree= of a worktree_cleaned are read, since that value can hold
// spaces. ok is false when there is no head= among them or its value is not
// a commit SHA (gitObjectPattern).
func recordedMergeHead(details string) (head string, ok bool) {
	tokens, _, _ := strings.Cut(details, " worktree=")
	for _, token := range strings.Fields(tokens) {
		if head, found := strings.CutPrefix(token, "head="); found {
			return head, gitObjectPattern.MatchString(head)
		}
	}
	return "", false
}
