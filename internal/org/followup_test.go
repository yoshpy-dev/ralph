package org

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// followupTestNow is the fixed clock of newTestFollowupStore.
const followupTestNow = "2026-10-30T12:00:00Z"

func newTestFollowupStore(t *testing.T) (*FollowupStore, string) {
	t.Helper()
	dir := t.TempDir()
	s := NewFollowupStore(dir)
	s.Now = func() time.Time { return time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC) }
	return s, dir
}

func writeFollowupFile(t *testing.T, s *FollowupStore, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(s.Path()), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(s.Path(), []byte(content), 0o644); err != nil {
		t.Fatalf("write followups: %v", err)
	}
}

func readFollowupFile(t *testing.T, s *FollowupStore) string {
	t.Helper()
	raw, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatalf("read followups: %v", err)
	}
	return string(raw)
}

func mustReadFollowups(t *testing.T, s *FollowupStore) FollowupReadResult {
	t.Helper()
	res, err := s.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return res
}

func mustAddFollowup(t *testing.T, s *FollowupStore, issue int, branch string) string {
	t.Helper()
	id, err := s.Add(FollowupAdd{Issue: issue, AfterBranch: branch})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	return id
}

func followupByID(t *testing.T, res FollowupReadResult, id string) Followup {
	t.Helper()
	for _, item := range res.Items {
		if item.ID == id {
			return item
		}
	}
	t.Fatalf("no follow-up %s in %+v", id, res.Items)
	return Followup{}
}

func followupIDs(res FollowupReadResult) string {
	ids := make([]string, 0, len(res.Items))
	for _, item := range res.Items {
		ids = append(ids, item.ID)
	}
	return strings.Join(ids, ",")
}

// followupAddedLine is an added event line as a test fixture.
func followupAddedLine(id string, issue int, branch string) string {
	return fmt.Sprintf(`{"ts":"2026-10-30T00:00:00Z","id":%q,"event":"added","issue":%d,"after_branch":%q}`, id, issue, branch)
}

func TestFollowupStore_LivesUnderTheGivenStateDir(t *testing.T) {
	state := filepath.Join(t.TempDir(), "ledger", "org")
	s := NewFollowupStore(state)
	if want := filepath.Join(state, "followups.jsonl"); s.Path() != want || FollowupsPathIn(state) != want {
		t.Fatalf("Path() = %q, FollowupsPathIn = %q; want %q", s.Path(), FollowupsPathIn(state), want)
	}
	if _, err := s.Add(FollowupAdd{Issue: 270, AfterBranch: "feat/a"}); err != nil {
		t.Fatalf("Add into a state dir that does not exist yet: %v", err)
	}
	for _, name := range []string{"followups.jsonl", "followups.lock"} {
		if _, err := os.Stat(filepath.Join(state, name)); err != nil {
			t.Errorf("%s under the state dir: %v", name, err)
		}
	}
	// The follow-up record touches no other record and no other lock.
	for _, name := range []string{"manifest.jsonl", manifestLockFile, "inbox.jsonl", inboxLockFile, "director.jsonl", directorLockFile} {
		if _, err := os.Stat(filepath.Join(state, name)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s should not exist after an add, stat err = %v", name, err)
		}
	}
}

// AC1 and AC2 of plan 2026-10-30-org-followups: a missing record reads as
// no follow-ups and creates nothing; a done on it is an unknown ID and
// creates nothing either.
func TestFollowupRead_MissingStateDirIsEmptyAndCreatesNothing(t *testing.T) {
	state := filepath.Join(t.TempDir(), "absent", "org")
	s := NewFollowupStore(state)
	res, err := s.Read()
	if err != nil {
		t.Fatalf("Read of a missing state dir: %v", err)
	}
	if len(res.Items) != 0 || res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Fatalf("want an empty result, got %+v", res)
	}
	if changed, err := s.Done("f1", ""); changed || !errors.Is(err, ErrFollowupUnknownID) ||
		!strings.Contains(err.Error(), "org: followup: done f1: ") {
		t.Fatalf("Done on a missing record = %v, %v; want org: followup: done f1: ... ErrFollowupUnknownID", changed, err)
	}
	if _, err := os.Stat(state); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Read or Done created the state dir (stat err = %v)", err)
	}
}

// AC1: add records the follow-up and returns its ID, the read lists it
// open, and done moves it to done with its note; the lines have the shape
// D1 gives them.
func TestFollowupStore_AddReadAndDone(t *testing.T) {
	s, _ := newTestFollowupStore(t)
	id, err := s.Add(FollowupAdd{Issue: 270, AfterBranch: "feat/org-followups", Note: "ask which repo the doc goes to"})
	if err != nil || id != "f1" {
		t.Fatalf("first Add = %q, %v; want f1", id, err)
	}
	if id := mustAddFollowup(t, s, 271, "feat/director-reqs"); id != "f2" {
		t.Fatalf("second Add = %s, want f2", id)
	}

	res := mustReadFollowups(t, s)
	if got := followupIDs(res); got != "f1,f2" || res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Fatalf("items = %s (corrupt %d, ignored %d), want f1,f2 (0, 0)", got, res.CorruptLines, res.IgnoredEvents)
	}
	want := Followup{
		ID: "f1", Issue: 270, AfterBranch: "feat/org-followups", Note: "ask which repo the doc goes to",
		State: FollowupStateOpen, AddedAt: followupTestNow,
	}
	if got := followupByID(t, res, "f1"); got != want {
		t.Fatalf("f1 = %+v, want %+v", got, want)
	}

	changed, err := s.Done("f1", "asked in #272")
	if err != nil || !changed {
		t.Fatalf("Done f1 = %v, %v; want changed", changed, err)
	}
	changed, err = s.Done("f2", "")
	if err != nil || !changed {
		t.Fatalf("Done f2 with no note = %v, %v; want changed", changed, err)
	}
	lines := strings.Split(strings.TrimSuffix(readFollowupFile(t, s), "\n"), "\n")
	wantLines := []string{
		`{"ts":"2026-10-30T12:00:00Z","id":"f1","event":"added","issue":270,"after_branch":"feat/org-followups","note":"ask which repo the doc goes to"}`,
		`{"ts":"2026-10-30T12:00:00Z","id":"f2","event":"added","issue":271,"after_branch":"feat/director-reqs"}`,
		`{"ts":"2026-10-30T12:00:00Z","id":"f1","event":"done","note":"asked in #272"}`,
		`{"ts":"2026-10-30T12:00:00Z","id":"f2","event":"done"}`,
	}
	if strings.Join(lines, "\n") != strings.Join(wantLines, "\n") {
		t.Fatalf("followups.jsonl =\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(wantLines, "\n"))
	}

	res = mustReadFollowups(t, s)
	want.State, want.DoneAt, want.DoneNote = FollowupStateDone, followupTestNow, "asked in #272"
	if got := followupByID(t, res, "f1"); got != want {
		t.Fatalf("f1 after done = %+v, want %+v", got, want)
	}
	if f2 := followupByID(t, res, "f2"); f2.State != FollowupStateDone || f2.DoneAt != followupTestNow || f2.DoneNote != "" || f2.Note != "" {
		t.Fatalf("f2 after done = %+v, want done with no notes", f2)
	}
}

// AC1: a second done changes nothing, writes nothing, and is no error, as
// the inbox's ack on an acked item; the first done's note stays.
func TestFollowupDone_SecondDoneChangesNothing(t *testing.T) {
	s, _ := newTestFollowupStore(t)
	mustAddFollowup(t, s, 270, "feat/a")
	if changed, err := s.Done("f1", "first"); err != nil || !changed {
		t.Fatalf("first Done = %v, %v; want changed", changed, err)
	}
	before := readFollowupFile(t, s)
	for _, note := range []string{"second", ""} {
		changed, err := s.Done("f1", note)
		if err != nil || changed {
			t.Fatalf("Done again with note %q = %v, %v; want unchanged and no error", note, changed, err)
		}
	}
	if got := readFollowupFile(t, s); got != before {
		t.Fatalf("followups.jsonl changed to %q", got)
	}
	if f1 := followupByID(t, mustReadFollowups(t, s), "f1"); f1.DoneNote != "first" {
		t.Fatalf("f1 DoneNote = %q, want the first done's note", f1.DoneNote)
	}
}

// AC1: done on an ID the record has no follow-up for is
// ErrFollowupUnknownID and writes nothing.
func TestFollowupDone_UnknownIDIsErrFollowupUnknownID(t *testing.T) {
	s, _ := newTestFollowupStore(t)
	mustAddFollowup(t, s, 270, "feat/a")
	before := readFollowupFile(t, s)
	for _, id := range []string{"f2", "f01", "F1", "e1", ""} {
		changed, err := s.Done(id, "note")
		if changed || !errors.Is(err, ErrFollowupUnknownID) {
			t.Errorf("Done(%q) = %v, %v; want ErrFollowupUnknownID", id, changed, err)
		}
	}
	if got := readFollowupFile(t, s); got != before {
		t.Fatalf("followups.jsonl changed to %q", got)
	}
}

// AC2: lines that do not parse are counted as corrupt, and events the fold
// cannot use are counted as ignored: an unknown kind, an ID not of the form
// f<N>, a second added for an ID, an added Add would refuse, a done for an
// ID with no added, and a second done.
func TestFollowupRead_FoldsInIDOrderAndCountsWhatItCannotUse(t *testing.T) {
	s, _ := newTestFollowupStore(t)
	writeFollowupFile(t, s, strings.Join([]string{
		`not json`,
		`{"ts":"t","id":"f3",`,
		`[1,2]`,
		``,
		`   `,
		followupAddedLine("f10", 12, "feat/b"),
		followupAddedLine("f2", 270, "feat/a"),
		// A second added for f2: ignored, the first stays.
		followupAddedLine("f2", 999, "feat/other"),
		// An unknown kind, a done for an ID with no added, and IDs not of the
		// form f<N>.
		`{"ts":"t","id":"f2","event":"reopened"}`,
		`{"ts":"t","id":"f7","event":"done"}`,
		followupAddedLine("f01", 1, "feat/x"),
		followupAddedLine("f0", 1, "feat/x"),
		followupAddedLine("fx", 1, "feat/x"),
		followupAddedLine("", 1, "feat/x"),
		followupAddedLine("F4", 1, "feat/x"),
		followupAddedLine("e5", 1, "feat/x"),
		// Adds that Add would refuse: no issue, issue 0 or negative, no
		// branch, a branch that starts with -, a branch with a space.
		`{"ts":"t","id":"f20","event":"added","after_branch":"feat/x"}`,
		followupAddedLine("f21", 0, "feat/x"),
		followupAddedLine("f22", -3, "feat/x"),
		`{"ts":"t","id":"f23","event":"added","issue":5}`,
		followupAddedLine("f24", 5, "--upload-pack=x"),
		followupAddedLine("f25", 5, "feat/a b"),
		// done applies once; the second done is ignored.
		`{"ts":"2026-10-30T01:00:00Z","id":"f10","event":"done","note":"n1"}`,
		`{"ts":"2026-10-30T02:00:00Z","id":"f10","event":"done","note":"n2"}`,
		// A done for a refused add has no follow-up to apply to.
		`{"ts":"t","id":"f21","event":"done"}`,
	}, "\n")+"\n")

	res := mustReadFollowups(t, s)
	if res.CorruptLines != 3 {
		t.Errorf("CorruptLines = %d, want 3 (blank lines are not counted)", res.CorruptLines)
	}
	if res.IgnoredEvents != 17 {
		t.Errorf("IgnoredEvents = %d, want 17", res.IgnoredEvents)
	}
	if got := followupIDs(res); got != "f2,f10" {
		t.Fatalf("items = %s, want f2,f10 (numeric ID order)", got)
	}
	if f2 := followupByID(t, res, "f2"); f2.Issue != 270 || f2.AfterBranch != "feat/a" || f2.State != FollowupStateOpen {
		t.Errorf("f2 = %+v, want the first added, open", f2)
	}
	if f10 := followupByID(t, res, "f10"); f10.State != FollowupStateDone || f10.DoneAt != "2026-10-30T01:00:00Z" || f10.DoneNote != "n1" {
		t.Errorf("f10 = %+v, want done by the first done (n1)", f10)
	}
	// Every ID in the file counts for the next one, the refused ones
	// included.
	if id := mustAddFollowup(t, s, 1, "feat/next"); id != "f26" {
		t.Fatalf("Add = %s, want f26", id)
	}
}

// AC2: the next ID is one more than the largest f<N> anywhere in the file,
// damaged lines included.
func TestFollowupAdd_NeverReusesTheIDOfADamagedLine(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		wantID  string
		wantIDs string
	}{
		{
			name:    "damaged middle line f2 between f1 and f3",
			content: followupAddedLine("f1", 1, "a") + "\n" + `{"ts":"t","id":"f2","event":"add` + "\n" + followupAddedLine("f3", 3, "c") + "\n",
			wantID:  "f4", wantIDs: "f1,f3,f4",
		},
		{
			name:    "the damaged line holds the largest ID, with spaces around the colon",
			content: followupAddedLine("f1", 1, "a") + "\n" + `{"ts":"t", "id" : "f8", "event":` + "\n" + followupAddedLine("f3", 3, "c") + "\n",
			wantID:  "f9", wantIDs: "f1,f3,f9",
		},
		{
			name:    "an ID too large for int64 in a damaged line is skipped",
			content: `{"id":"f99999999999999999999999",` + "\n" + followupAddedLine("f2", 2, "b") + "\n",
			wantID:  "f3", wantIDs: "f2,f3",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newTestFollowupStore(t)
			writeFollowupFile(t, s, tc.content)
			if id := mustAddFollowup(t, s, 270, "feat/a"); id != tc.wantID {
				t.Fatalf("Add = %s, want %s", id, tc.wantID)
			}
			res := mustReadFollowups(t, s)
			if got := followupIDs(res); got != tc.wantIDs || res.CorruptLines != 1 {
				t.Fatalf("items = %s (corrupt %d), want %s (corrupt 1)", got, res.CorruptLines, tc.wantIDs)
			}
		})
	}
}

// AC2: a torn last line gets a newline before the next append, so it stays
// one corrupt line and the new events read back.
func TestFollowupAdd_TornLastLineStaysSeparateAndTheNewEventIsReadable(t *testing.T) {
	for _, tc := range []struct {
		name   string
		torn   string
		wantID string
	}{
		{name: "cut after the id", torn: `{"ts":"2026-10-30T00:00:00Z","id":"f5","event":"added","issue":2`, wantID: "f6"},
		{name: "cut inside the id", torn: `{"ts":"2026-10-30T00:00:00Z","id":"f12`, wantID: "f13"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newTestFollowupStore(t)
			writeFollowupFile(t, s, followupAddedLine("f4", 4, "feat/d")+"\n"+tc.torn)
			if id := mustAddFollowup(t, s, 270, "feat/a"); id != tc.wantID {
				t.Fatalf("Add = %s, want %s", id, tc.wantID)
			}
			if changed, err := s.Done(tc.wantID, ""); err != nil || !changed {
				t.Fatalf("Done of the new follow-up = %v, %v", changed, err)
			}
			lines := strings.Split(readFollowupFile(t, s), "\n")
			if len(lines) != 5 || lines[1] != tc.torn || lines[4] != "" {
				t.Fatalf("followups.jsonl lines = %q, want f4, the torn line on its own, the new added, done, and a final newline", lines)
			}
			res := mustReadFollowups(t, s)
			if got := followupIDs(res); got != "f4,"+tc.wantID || res.CorruptLines != 1 || res.IgnoredEvents != 0 {
				t.Fatalf("items = %s (corrupt %d, ignored %d), want f4,%s (corrupt 1, ignored 0)", got, res.CorruptLines, res.IgnoredEvents, tc.wantID)
			}
			if item := followupByID(t, res, tc.wantID); item.State != FollowupStateDone {
				t.Fatalf("new follow-up state = %s, want done", item.State)
			}
		})
	}
}

func TestFollowupAdd_CountsAnIDWrittenWithJSONEscapes(t *testing.T) {
	s, _ := newTestFollowupStore(t)
	// "%u00669" becomes a JSON escape for "f" followed by 9: the decoded ID
	// is f9, which the raw-text scan alone would not find.
	line := `{"ts":"2026-10-30T00:00:00Z","id":"%u00669","event":"added","issue":1,"after_branch":"a"}`
	writeFollowupFile(t, s, strings.ReplaceAll(line, "%u", "\\"+"u")+"\n")
	if res := mustReadFollowups(t, s); followupIDs(res) != "f9" {
		t.Fatalf("items = %s, want f9", followupIDs(res))
	}
	if id := mustAddFollowup(t, s, 270, "feat/a"); id != "f10" {
		t.Fatalf("Add = %s, want f10", id)
	}
}

func TestFollowupAdd_FailsWhenNoIDIsLeft(t *testing.T) {
	s, _ := newTestFollowupStore(t)
	content := followupAddedLine("f9223372036854775807", 1, "a") + "\n"
	writeFollowupFile(t, s, content)
	_, err := s.Add(FollowupAdd{Issue: 270, AfterBranch: "feat/a"})
	if err == nil || !strings.Contains(err.Error(), "no follow-up ID is left") {
		t.Fatalf("Add err = %v, want no follow-up ID is left", err)
	}
	if got := readFollowupFile(t, s); got != content {
		t.Fatalf("followups.jsonl changed to %q", got)
	}
}

// D3: the issue is positive; the after-branch is not empty, holds no
// whitespace or control character, does not start with -, and is valid
// UTF-8; the note is optional and, when given, follows the inbox note's
// rules.
func TestValidateFollowupAdd(t *testing.T) {
	const branch = "feat/org-followups"
	for _, tc := range []struct {
		name string
		in   FollowupAdd
		want string // "" means valid
	}{
		{"a feature branch and no note", FollowupAdd{Issue: 270, AfterBranch: branch}, ""},
		{"a branch with non-ASCII letters", FollowupAdd{Issue: 1, AfterBranch: "feat/日本語"}, ""},
		{"a branch with a - inside", FollowupAdd{Issue: 1, AfterBranch: "a-b"}, ""},
		{"a note of 500 multibyte characters", FollowupAdd{Issue: 1, AfterBranch: branch, Note: strings.Repeat("あ", 500)}, ""},
		{"issue 0", FollowupAdd{Issue: 0, AfterBranch: branch}, "the issue must be a positive number, not 0"},
		{"a negative issue", FollowupAdd{Issue: -1, AfterBranch: branch}, "the issue must be a positive number, not -1"},
		{"no branch", FollowupAdd{Issue: 1}, "the after-branch is required"},
		{"a branch that starts with -", FollowupAdd{Issue: 1, AfterBranch: "-x"}, `the after-branch "-x" starts with -`},
		{"a branch that is a long option", FollowupAdd{Issue: 1, AfterBranch: "--upload-pack=x"}, "starts with -"},
		{"a space", FollowupAdd{Issue: 1, AfterBranch: "feat/a b"}, "whitespace or a control character (U+0020)"},
		{"a leading space", FollowupAdd{Issue: 1, AfterBranch: " feat/a"}, "(U+0020)"},
		{"a tab", FollowupAdd{Issue: 1, AfterBranch: "feat/a\tb"}, "(U+0009)"},
		{"a newline", FollowupAdd{Issue: 1, AfterBranch: "feat/a\n"}, "(U+000A)"},
		{"an escape", FollowupAdd{Issue: 1, AfterBranch: "feat/a" + string(rune(0x1b)) + "[31m"}, "(U+001B)"},
		{"DEL", FollowupAdd{Issue: 1, AfterBranch: "feat/a" + string(rune(0x7f))}, "(U+007F)"},
		{"a no-break space", FollowupAdd{Issue: 1, AfterBranch: "feat/a" + string(rune(0xa0)) + "b"}, "(U+00A0)"},
		{"a line separator", FollowupAdd{Issue: 1, AfterBranch: "feat/a" + string(rune(0x2028))}, "(U+2028)"},
		{"invalid UTF-8 in the branch", FollowupAdd{Issue: 1, AfterBranch: "feat/a\xffb"}, "the after-branch is not valid UTF-8"},
		{"a blank note", FollowupAdd{Issue: 1, AfterBranch: branch, Note: "  "}, "the note is blank"},
		{"a two-line note", FollowupAdd{Issue: 1, AfterBranch: branch, Note: "one\ntwo"}, "the note must be a single line"},
		{"a note with a CR", FollowupAdd{Issue: 1, AfterBranch: branch, Note: "one\rtwo"}, "the note must be a single line"},
		{"a note of 501 characters", FollowupAdd{Issue: 1, AfterBranch: branch, Note: strings.Repeat("a", 501)}, "the note is 501 characters; at most 500"},
		{"a note with a tab", FollowupAdd{Issue: 1, AfterBranch: branch, Note: "a\tb"}, "the note contains a control character (U+0009)"},
		{"a note with invalid UTF-8", FollowupAdd{Issue: 1, AfterBranch: branch, Note: "a\xffb"}, "the note is not valid UTF-8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFollowupAdd(tc.in)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("ValidateFollowupAdd: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.HasPrefix(err.Error(), "org: followup: ") {
				t.Fatalf("err = %v, want org: followup: ... %q", err, tc.want)
			}
		})
	}
}

// D3: a refused add creates neither the record nor its lock, and a done with
// a refused note writes nothing and leaves the follow-up open.
func TestFollowupAddAndDone_RefusedInputWritesNothing(t *testing.T) {
	s, dir := newTestFollowupStore(t)
	for _, in := range []FollowupAdd{{AfterBranch: "feat/a"}, {Issue: 1}, {Issue: 1, AfterBranch: "-x"}, {Issue: 1, AfterBranch: "feat/a", Note: "a\nb"}} {
		if _, err := s.Add(in); err == nil {
			t.Errorf("Add(%+v) succeeded", in)
		}
	}
	for _, name := range []string{"followups.jsonl", followupLockFile} {
		if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("a refused add created %s (stat err = %v)", name, err)
		}
	}

	mustAddFollowup(t, s, 270, "feat/a")
	before := readFollowupFile(t, s)
	for _, note := range []string{" ", "a\nb", strings.Repeat("x", 501), "a" + string(rune(0x07)) + "b"} {
		if changed, err := s.Done("f1", note); err == nil || changed {
			t.Errorf("Done(f1, %q) = %v, %v; want a refusal", note, changed, err)
		}
	}
	if got := readFollowupFile(t, s); got != before {
		t.Fatalf("followups.jsonl changed to %q", got)
	}
	if f1 := followupByID(t, mustReadFollowups(t, s), "f1"); f1.State != FollowupStateOpen {
		t.Fatalf("f1 state = %s, want open", f1.State)
	}
}

// AC2: adds racing on the same record each read it again under the lock,
// so each gets its own ID and every line reads back.
func TestFollowupAdd_ConcurrentCallsGetDistinctIDs(t *testing.T) {
	s, _ := newTestFollowupStore(t)
	const n = 10
	ids := make([]string, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids[i], errs[i] = s.Add(FollowupAdd{Issue: 100 + i, AfterBranch: fmt.Sprintf("feat/b%d", i)})
		}()
	}
	wg.Wait()

	seen := map[string]bool{}
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("Add %d: %v", i, errs[i])
		}
		if seen[ids[i]] {
			t.Fatalf("ID %s was handed out twice: %v", ids[i], ids)
		}
		seen[ids[i]] = true
	}
	for i := 1; i <= n; i++ {
		if !seen[fmt.Sprintf("f%d", i)] {
			t.Fatalf("IDs %v, want f1..f%d", ids, n)
		}
	}
	res := mustReadFollowups(t, s)
	if len(res.Items) != n || res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Fatalf("Read: %d items, corrupt %d, ignored %d; want %d, 0, 0", len(res.Items), res.CorruptLines, res.IgnoredEvents, n)
	}
}

// AC2: dones racing on the same follow-up: exactly one changes it.
func TestFollowupDone_ConcurrentCallsChangeTheFollowupOnce(t *testing.T) {
	s, _ := newTestFollowupStore(t)
	mustAddFollowup(t, s, 270, "feat/a")
	const n = 10
	changed := make([]bool, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() { defer wg.Done(); changed[i], errs[i] = s.Done("f1", fmt.Sprintf("note %d", i)) }()
	}
	wg.Wait()
	count := 0
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("Done %d: %v", i, errs[i])
		}
		if changed[i] {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("%d Dones reported changed, want exactly 1", count)
	}
	if lines := strings.Count(readFollowupFile(t, s), "\n"); lines != 2 {
		t.Fatalf("followups.jsonl has %d lines, want the added and one done", lines)
	}
	if res := mustReadFollowups(t, s); res.IgnoredEvents != 0 {
		t.Fatalf("IgnoredEvents = %d: a second done was written", res.IgnoredEvents)
	}
}

// AC2: followups.lock is a lock of its own: holding the manifest's, the
// inbox's, or the director's lock does not block an add, and holding
// followups.lock does until it is released.
func TestFollowupLock_IsSeparateFromTheOtherLocksAndSerializesWrites(t *testing.T) {
	s, dir := newTestFollowupStore(t)

	for _, name := range []string{manifestLockFile, inboxLockFile, directorLockFile} {
		start := time.Now()
		if err := withFileLock(dir, name, "other", func() error {
			_, err := s.Add(FollowupAdd{Issue: 270, AfterBranch: "feat/a"})
			return err
		}); err != nil {
			t.Fatalf("Add under %s: %v", name, err)
		}
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Fatalf("Add under %s took %s: it waited for that lock", name, elapsed)
		}
	}

	held, release := make(chan struct{}), make(chan struct{})
	holderDone := make(chan error, 1)
	go func() {
		holderDone <- withFileLock(dir, followupLockFile, "followup", func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	type result struct {
		id  string
		err error
	}
	done := make(chan result, 1)
	go func() {
		id, err := s.Add(FollowupAdd{Issue: 271, AfterBranch: "feat/b"})
		done <- result{id, err}
	}()
	select {
	case r := <-done:
		t.Fatalf("Add returned %+v while followups.lock was held", r)
	case <-time.After(150 * time.Millisecond):
	}
	close(release)
	if err := <-holderDone; err != nil {
		t.Fatalf("lock holder: %v", err)
	}
	if r := <-done; r.err != nil || r.id != "f4" {
		t.Fatalf("Add after release = %+v, want f4", r)
	}
}

func TestFollowupStore_ReportsAnUnreadableRecordAndAnUnavailableLock(t *testing.T) {
	t.Run("followups.jsonl is a directory", func(t *testing.T) {
		s, _ := newTestFollowupStore(t)
		if err := os.Mkdir(s.Path(), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if _, err := s.Read(); err == nil || !strings.Contains(err.Error(), "org: followup: read ") {
			t.Errorf("Read err = %v", err)
		}
		if _, err := s.Add(FollowupAdd{Issue: 270, AfterBranch: "feat/a"}); err == nil {
			t.Errorf("Add succeeded on an unreadable record")
		}
		if _, err := s.Done("f1", ""); err == nil || errors.Is(err, ErrFollowupUnknownID) {
			t.Errorf("Done err = %v, want the read error, not ErrFollowupUnknownID", err)
		}
	})
	t.Run("followups.lock is a directory", func(t *testing.T) {
		s, dir := newTestFollowupStore(t)
		if err := os.Mkdir(filepath.Join(dir, followupLockFile), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if _, err := s.Add(FollowupAdd{Issue: 270, AfterBranch: "feat/a"}); err == nil ||
			!strings.Contains(err.Error(), "org: open followup lock file ") {
			t.Errorf("Add err = %v, want org: open followup lock file ...", err)
		}
		if _, err := os.Stat(s.Path()); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Add wrote the record without the lock (stat err = %v)", err)
		}
	})
}

// prMergedEvent is a pr_merged as ralph org merge writes it (merge_run.go's
// recordMerge).
func prMergedEvent(ts, branch string) ManifestEvent {
	return ManifestEvent{TS: ts, OrgID: "org-a", Event: EventPRMerged, Details: "branch=" + branch +
		" pr=12 head=0123456789abcdef0123456789abcdef01234567 method=merge merge_commit=unknown auto_merge=false"}
}

// worktreeCleanedEvent is a worktree_cleaned as the cleanup writes it
// (cleanupDetails), with worktree as its quoted path.
func worktreeCleanedEvent(ts, branch, worktree string) ManifestEvent {
	return ManifestEvent{TS: ts, OrgID: "org-a", Event: EventWorktreeCleaned,
		Details: cleanupDetails(branch, "0123456789abcdef0123456789abcdef01234567", "cleanup", worktree)}
}

// D2, D5, AC4, and AC8: the ledger records the after-branch as merged only
// by a pr_merged or a worktree_cleaned that names the same branch exactly at
// or after the follow-up was added.
func TestFollowupRecordedMerge(t *testing.T) {
	const added = "2026-10-30T12:00:00Z"
	followup := Followup{ID: "f1", Issue: 270, AfterBranch: "feat/a", State: FollowupStateOpen, AddedAt: added}
	for _, tc := range []struct {
		name      string
		f         Followup
		events    []ManifestEvent
		wantAt    string
		wantEvent string // "" means not merged
	}{
		{name: "no events", f: followup},
		{
			name: "a pr_merged after the add", f: followup,
			events: []ManifestEvent{prMergedEvent("2026-10-30T13:00:00Z", "feat/a")},
			wantAt: "2026-10-30T13:00:00Z", wantEvent: EventPRMerged,
		},
		{
			name: "a pr_merged in the second of the add", f: followup,
			events: []ManifestEvent{prMergedEvent(added, "feat/a")},
			wantAt: added, wantEvent: EventPRMerged,
		},
		{
			name: "a pr_merged before the add (a branch name used again)", f: followup,
			events: []ManifestEvent{prMergedEvent("2026-10-30T11:59:59Z", "feat/a")},
		},
		{
			name: "a worktree_cleaned after the add", f: followup,
			events: []ManifestEvent{worktreeCleanedEvent("2026-10-30T14:00:00Z", "feat/a", "/repo/.claude/worktrees/a")},
			wantAt: "2026-10-30T14:00:00Z", wantEvent: EventWorktreeCleaned,
		},
		{
			name: "a pr_merged before the add and a worktree_cleaned after it", f: followup,
			events: []ManifestEvent{
				prMergedEvent("2026-10-29T09:00:00Z", "feat/a"),
				worktreeCleanedEvent("2026-10-30T12:30:00Z", "feat/a", "/repo/wt"),
			},
			wantAt: "2026-10-30T12:30:00Z", wantEvent: EventWorktreeCleaned,
		},
		{
			name: "the first of two in file order", f: followup,
			events: []ManifestEvent{
				prMergedEvent("2026-10-30T13:00:00Z", "feat/a"),
				worktreeCleanedEvent("2026-10-30T13:00:05Z", "feat/a", "/repo/wt"),
			},
			wantAt: "2026-10-30T13:00:00Z", wantEvent: EventPRMerged,
		},
		{
			name: "another branch", f: followup,
			events: []ManifestEvent{
				prMergedEvent("2026-10-30T13:00:00Z", "feat/b"),
				worktreeCleanedEvent("2026-10-30T13:00:00Z", "feat/b", "/repo/wt"),
			},
		},
		{
			name: "a branch the after-branch is a prefix of", f: followup,
			events: []ManifestEvent{
				prMergedEvent("2026-10-30T13:00:00Z", "feat/ab"),
				worktreeCleanedEvent("2026-10-30T13:00:00Z", "feat/a/b", "/repo/wt"),
			},
		},
		{
			name: "a branch that is a prefix of the after-branch",
			f:    Followup{ID: "f1", Issue: 270, AfterBranch: "feat/ab", State: FollowupStateOpen, AddedAt: added},
			events: []ManifestEvent{
				prMergedEvent("2026-10-30T13:00:00Z", "feat/a"),
				worktreeCleanedEvent("2026-10-30T13:00:00Z", "feat/a", "/repo/wt"),
			},
		},
		{
			name: "another branch's worktree_cleaned whose quoted worktree holds the branch", f: followup,
			events: []ManifestEvent{worktreeCleanedEvent("2026-10-30T13:00:00Z", "feat/b", "/repo/x branch=feat/a")},
		},
		{
			name: "a gate record and a scope reservation that name the branch", f: followup,
			events: []ManifestEvent{
				{TS: "2026-10-30T13:00:00Z", OrgID: "org-a", Event: EventGatePassed, Details: "branch=feat/a"},
				{TS: "2026-10-30T13:00:00Z", OrgID: "org-a", Event: EventScopeReserved, Details: "branch=feat/a"},
			},
		},
		{
			name: "times compare as instants, not as text", f: followup,
			events: []ManifestEvent{prMergedEvent("2026-10-30T20:30:00+09:00", "feat/a")},
		},
		{
			name: "an event ts that does not parse", f: followup,
			events: []ManifestEvent{prMergedEvent("yesterday", "feat/a")},
		},
		{
			name:   "an added_at that does not parse",
			f:      Followup{ID: "f1", Issue: 270, AfterBranch: "feat/a", State: FollowupStateOpen, AddedAt: "t"},
			events: []ManifestEvent{prMergedEvent("2026-10-30T13:00:00Z", "feat/a")},
		},
		{
			name:   "a Details with an empty branch, for a follow-up with none",
			f:      Followup{ID: "f1", Issue: 270, State: FollowupStateOpen, AddedAt: added},
			events: []ManifestEvent{{TS: "2026-10-30T13:00:00Z", OrgID: "org-a", Event: EventPRMerged, Details: "branch= pr=12"}},
		},
		{
			name: "a Details whose first token is not branch=", f: followup,
			events: []ManifestEvent{{TS: "2026-10-30T13:00:00Z", OrgID: "org-a", Event: EventPRMerged, Details: "pr=12 branch=feat/a"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			merged, at, event := FollowupRecordedMerge(tc.events, tc.f)
			if merged != (tc.wantEvent != "") || at != tc.wantAt || event != tc.wantEvent {
				t.Fatalf("FollowupRecordedMerge = %v, %q, %q; want %v, %q, %q", merged, at, event, tc.wantEvent != "", tc.wantAt, tc.wantEvent)
			}
		})
	}
}
