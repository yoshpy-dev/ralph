package org

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// directorTestNow is the clock newTestDirectorStore starts at.
const directorTestNow = "2026-10-15T09:00:00Z"

// newTestDirectorStore returns a store under t.TempDir() whose clock reads
// *clock, so a test can move it between calls.
func newTestDirectorStore(t *testing.T) (*DirectorStore, string, *time.Time) {
	t.Helper()
	dir := t.TempDir()
	clock := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	s := NewDirectorStore(dir)
	s.Now = func() time.Time { return clock }
	return s, dir, &clock
}

func writeDirectorFile(t *testing.T, s *DirectorStore, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(s.Path()), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(s.Path(), []byte(content), 0o644); err != nil {
		t.Fatalf("write director.jsonl: %v", err)
	}
}

func readDirectorFile(t *testing.T, s *DirectorStore) string {
	t.Helper()
	raw, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatalf("read director.jsonl: %v", err)
	}
	return string(raw)
}

func directorFileLines(t *testing.T, s *DirectorStore) []string {
	t.Helper()
	return strings.Split(strings.TrimSuffix(readDirectorFile(t, s), "\n"), "\n")
}

func mustReadDirector(t *testing.T, s *DirectorStore) DirectorReadResult {
	t.Helper()
	res, err := s.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return res
}

func mustAttachDirector(t *testing.T, s *DirectorStore, a DirectorAttach) DirectorRegistration {
	t.Helper()
	reg, _, err := s.Attach(a)
	if err != nil {
		t.Fatalf("Attach(%+v): %v", a, err)
	}
	return reg
}

// currentDirectorID is the id of the current registration, or "" when none.
func currentDirectorID(res DirectorReadResult) string {
	if res.Current == nil {
		return ""
	}
	return res.Current.ID
}

// directorAttachedLine and directorDetachedLine are director.jsonl lines as
// test fixtures.
func directorAttachedLine(id, driver string) string {
	return fmt.Sprintf(`{"ts":"2026-10-15T00:00:00Z","id":%q,"event":"attached","driver":%q,"wake":["pull"],"user":"u","cwd":"/w","herdr_pane":false}`, id, driver)
}

func directorDetachedLine(id string) string {
	return fmt.Sprintf(`{"ts":"2026-10-15T00:00:01Z","id":%q,"event":"detached","user":"u","cwd":"/w","herdr_pane":false}`, id)
}

// directorPartialWrite is a write seam that writes b to the real file up to
// half of its last line, as a full disk can stop a write, and then fails with
// ENOSPC. *onDisk gets the file as that partial write left it, before
// appendLocked cuts it back.
func directorPartialWrite(t *testing.T, s *DirectorStore, onDisk *string) func(*os.File, []byte) (int, error) {
	return func(f *os.File, b []byte) (int, error) {
		last := bytes.LastIndexByte(b[:len(b)-1], '\n') + 1
		n, err := f.Write(b[:last+(len(b)-last)/2])
		if err != nil {
			t.Errorf("partial write: %v", err)
		}
		raw, err := os.ReadFile(s.Path())
		if err != nil {
			t.Errorf("read after the partial write: %v", err)
		}
		*onDisk = string(raw)
		return n, syscall.ENOSPC
	}
}

func dirEntryNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestDirectorStore_LivesUnderTheGivenStateDir(t *testing.T) {
	state := filepath.Join(t.TempDir(), "ledger", "org")
	s := NewDirectorStore(state)
	if want := filepath.Join(state, "director.jsonl"); s.Path() != want || DirectorPathIn(state) != want {
		t.Fatalf("Path() = %q, DirectorPathIn = %q; want %q", s.Path(), DirectorPathIn(state), want)
	}
	if _, _, err := s.Attach(DirectorAttach{Driver: DirectorDriverClaude}); err != nil {
		t.Fatalf("Attach into a state dir that does not exist yet: %v", err)
	}
	if got := strings.Join(dirEntryNames(t, state), ","); got != "director.jsonl,director.lock" {
		t.Fatalf("state dir holds %s, want only director.jsonl and director.lock (no manifest, no inbox, no their locks)", got)
	}
}

func TestDirectorRead_MissingStateDirIsNoRegistration(t *testing.T) {
	state := filepath.Join(t.TempDir(), "absent", "org")
	res, err := NewDirectorStore(state).Read()
	if err != nil {
		t.Fatalf("Read of a missing state dir: %v", err)
	}
	if res.Current != nil || res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Fatalf("want no registration, got %+v", res)
	}
	if _, err := os.Stat(state); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Read created the state dir (stat err = %v)", err)
	}
}

func TestDirectorAttach_RecordsOneAttachedWithAnID(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	reg, replaced, err := s.Attach(DirectorAttach{Driver: DirectorDriverClaude, User: "alice", Cwd: "/work/a", HerdrPane: true})
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	want := DirectorRegistration{
		ID: "d1", Driver: DirectorDriverClaude, Wake: []string{DirectorWakePull},
		AttachedAt: directorTestNow, User: "alice", Cwd: "/work/a", HerdrPane: true,
	}
	if !reflect.DeepEqual(reg, want) || replaced != nil {
		t.Fatalf("Attach = %+v, replaced %v; want %+v and nil", reg, replaced, want)
	}
	wantLine := `{"ts":"2026-10-15T09:00:00Z","id":"d1","event":"attached","driver":"claude","wake":["pull"],"user":"alice","cwd":"/work/a","herdr_pane":true}`
	if got := readDirectorFile(t, s); got != wantLine+"\n" {
		t.Fatalf("director.jsonl = %q, want %q", got, wantLine+"\n")
	}
	res := mustReadDirector(t, s)
	if res.Current == nil || !reflect.DeepEqual(*res.Current, want) || res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Fatalf("Read = %+v (current %+v), want %+v", res, res.Current, want)
	}
}

func TestDirectorRegistration_JSONShape(t *testing.T) {
	reg := DirectorRegistration{
		ID: "d3", Driver: DirectorDriverCodex, Wake: []string{DirectorWakePull},
		AttachedAt: directorTestNow, User: "alice", Cwd: "/work/a",
	}
	raw, err := json.Marshal(reg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"id":"d3","driver":"codex","wake":["pull"],"attached_at":"2026-10-15T09:00:00Z","user":"alice","cwd":"/work/a","herdr_pane":false}`
	if string(raw) != want {
		t.Fatalf("JSON = %s, want %s", raw, want)
	}
	reg.PaneID = "w1:p2"
	if raw, _ = json.Marshal(reg); !strings.Contains(string(raw), `"pane_id":"w1:p2"`) {
		t.Fatalf("JSON with a pane id = %s", raw)
	}
}

func TestDirectorAttach_RefusesASecondAttachWithoutReplace(t *testing.T) {
	s, _, clock := newTestDirectorStore(t)
	mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude, User: "alice", Cwd: "/work/a"})
	before := readDirectorFile(t, s)
	*clock = clock.Add(time.Minute)

	for _, driver := range []string{DirectorDriverClaude, DirectorDriverCodex} {
		reg, replaced, err := s.Attach(DirectorAttach{Driver: driver, User: "bob", Cwd: "/work/b"})
		if !errors.Is(err, ErrDirectorAttached) {
			t.Fatalf("second Attach (%s) err = %v, want ErrDirectorAttached", driver, err)
		}
		// The message names the registration and ends there: the way out
		// is the caller's to say (self-review L7 of org-director-attach).
		if want := "org: director: attach: a director is already attached: d1 (driver claude, attached " + directorTestNow + ")"; err.Error() != want {
			t.Errorf("err = %q, want %q", err, want)
		}
		if reg.ID != "" || replaced != nil {
			t.Errorf("refused Attach returned %+v, %v", reg, replaced)
		}
	}
	if got := readDirectorFile(t, s); got != before {
		t.Fatalf("director.jsonl changed to %q", got)
	}
	if id := currentDirectorID(mustReadDirector(t, s)); id != "d1" {
		t.Fatalf("current = %q, want d1", id)
	}
}

func TestDirectorAttach_EscapesAHandEditedAttachTimeInItsError(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	writeDirectorFile(t, s, `{"ts":"2026-10-15\u001b[2J","id":"d1","event":"attached","driver":"claude","wake":["pull"],"user":"u","cwd":"/w","herdr_pane":false}`+"\n")

	_, _, err := s.Attach(DirectorAttach{Driver: DirectorDriverCodex})
	if !errors.Is(err, ErrDirectorAttached) {
		t.Fatalf("Attach err = %v, want ErrDirectorAttached", err)
	}
	if strings.ContainsRune(err.Error(), '\x1b') {
		t.Fatalf("err %q carries a raw escape character", err)
	}
	if want := `attached 2026-10-15\x1b[2J`; !strings.Contains(err.Error(), want) {
		t.Errorf("err %q does not show the escaped attach time %q", err, want)
	}
}

func TestDirectorAttach_ReplaceWritesTheDetachedAndANewAttached(t *testing.T) {
	s, _, clock := newTestDirectorStore(t)
	first := mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude, User: "alice", Cwd: "/work/a", HerdrPane: true})
	*clock = clock.Add(time.Minute)

	reg, replaced, err := s.Attach(DirectorAttach{Driver: DirectorDriverCodex, Replace: true, User: "bob", Cwd: "/work/b"})
	if err != nil {
		t.Fatalf("Attach with replace: %v", err)
	}
	if reg.ID != "d2" || reg.Driver != DirectorDriverCodex || reg.AttachedAt != "2026-10-15T09:01:00Z" || reg.User != "bob" {
		t.Errorf("new registration = %+v, want d2 codex by bob at 09:01", reg)
	}
	if replaced == nil || !reflect.DeepEqual(*replaced, first) {
		t.Errorf("replaced = %+v, want the first registration %+v", replaced, first)
	}
	// The detached is written by the one who replaces: its user, cwd, and
	// herdr_pane are bob's.
	want := []string{
		`{"ts":"2026-10-15T09:00:00Z","id":"d1","event":"attached","driver":"claude","wake":["pull"],"user":"alice","cwd":"/work/a","herdr_pane":true}`,
		`{"ts":"2026-10-15T09:01:00Z","id":"d1","event":"detached","user":"bob","cwd":"/work/b","herdr_pane":false}`,
		`{"ts":"2026-10-15T09:01:00Z","id":"d2","event":"attached","driver":"codex","wake":["pull"],"user":"bob","cwd":"/work/b","herdr_pane":false}`,
	}
	if got := directorFileLines(t, s); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("director.jsonl =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	res := mustReadDirector(t, s)
	if currentDirectorID(res) != "d2" || res.Current.Driver != DirectorDriverCodex || res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Fatalf("Read = %+v (current %+v), want d2 codex and nothing skipped", res, res.Current)
	}
}

// The replace is one write (self-review L5 of org-director-attach): when
// it fails, neither the detached nor the new attached is in the file, the
// old registration is still current, and the error says nothing was
// recorded. A read-only director.jsonl makes the write fail after the read
// under the lock succeeds.
func TestDirectorAttach_AReplaceThatCannotWriteRecordsNothing(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes a read-only file")
	}
	s, _, clock := newTestDirectorStore(t)
	mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude, User: "alice", Cwd: "/work/a"})
	before := readDirectorFile(t, s)
	if err := os.Chmod(s.Path(), 0o444); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(s.Path(), 0o644) })
	*clock = clock.Add(time.Minute)

	reg, replaced, err := s.Attach(DirectorAttach{Driver: DirectorDriverCodex, Replace: true, User: "bob", Cwd: "/work/b"})
	want := "org: director: attach: nothing was recorded, so d1 (driver claude, attached " + directorTestNow + ") is still the current registration: org: director: open " + s.Path() + ": "
	if err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("Attach with replace on a read-only file err = %v, want it to start with %q", err, want)
	}
	if reg.ID != "" || replaced != nil {
		t.Errorf("failed Attach returned %+v, %v", reg, replaced)
	}
	if got := readDirectorFile(t, s); got != before {
		t.Fatalf("director.jsonl changed to %q", got)
	}
	if id := currentDirectorID(mustReadDirector(t, s)); id != "d1" {
		t.Fatalf("current = %q, want d1", id)
	}
}

// A replace whose write stops after the detached line and half of the new
// attached (a full disk) is cut back (cross-review finding 1 of
// org-director-attach). Left in place, the detached would end d1 and the cut
// attached would be a corrupt line, so no registration would be current
// while the error said d1 still was.
func TestDirectorAttach_AReplaceCutOffPartwayIsRestored(t *testing.T) {
	s, _, clock := newTestDirectorStore(t)
	mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude, User: "alice", Cwd: "/work/a"})
	before := readDirectorFile(t, s)
	*clock = clock.Add(time.Minute)
	var onDisk string
	s.write = directorPartialWrite(t, s, &onDisk)

	reg, replaced, err := s.Attach(DirectorAttach{Driver: DirectorDriverCodex, Replace: true, User: "bob", Cwd: "/work/b"})
	want := "org: director: attach: nothing was recorded, so d1 (driver claude, attached " + directorTestNow + ") is still the current registration: org: director: append to " + s.Path() + ": no space left on device"
	if err == nil || err.Error() != want {
		t.Fatalf("Attach with replace err = %v, want %q", err, want)
	}
	if !errors.Is(err, syscall.ENOSPC) || errors.Is(err, ErrDirectorWriteNotRestored) {
		t.Errorf("err = %v: want ENOSPC wrapped and not ErrDirectorWriteNotRestored", err)
	}
	if reg.ID != "" || replaced != nil {
		t.Errorf("failed Attach returned %+v, %v", reg, replaced)
	}
	// The write did reach the file: the whole detached and part of the
	// attached, which holds d2.
	detached := `{"ts":"2026-10-15T09:01:00Z","id":"d1","event":"detached","user":"bob","cwd":"/work/b","herdr_pane":false}` + "\n"
	cut, ok := strings.CutPrefix(onDisk, before+detached)
	if !ok || !strings.Contains(cut, `"id":"d2","event":"attached"`) || strings.HasSuffix(cut, "\n") {
		t.Fatalf("the file after the partial write = %q, want the old lines, the whole detached, and part of the attached", onDisk)
	}
	if got := readDirectorFile(t, s); got != before {
		t.Fatalf("director.jsonl = %q, want it cut back to %q", got, before)
	}
	if res := mustReadDirector(t, s); currentDirectorID(res) != "d1" || res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Fatalf("Read = %+v (current %+v), want d1 current and nothing skipped", res, res.Current)
	}
	// The cut text that held d2 is gone with the rest, so the next replace
	// gets d2.
	s.write = nil
	reg, replaced, err = s.Attach(DirectorAttach{Driver: DirectorDriverCodex, Replace: true, User: "bob", Cwd: "/work/b"})
	if err != nil || reg.ID != "d2" || replaced == nil || replaced.ID != "d1" {
		t.Fatalf("Attach with replace after the failure = %+v, %+v, %v; want d2 replacing d1", reg, replaced, err)
	}
}

// When the cut back fails too, the error does not say nothing was recorded:
// the file holds the detached and the cut attached, so no registration is
// current. It wraps ErrDirectorWriteNotRestored and names the command that
// reads the registration.
func TestDirectorAttach_AReplaceThatCannotBeCutBackSaysSo(t *testing.T) {
	s, _, clock := newTestDirectorStore(t)
	mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude, User: "alice", Cwd: "/work/a"})
	before := readDirectorFile(t, s)
	*clock = clock.Add(time.Minute)
	var onDisk string
	s.write = directorPartialWrite(t, s, &onDisk)
	errTruncate := errors.New("truncate refused")
	var cuts []string
	s.truncate = func(path string, size int64) error {
		cuts = append(cuts, fmt.Sprintf("%s %d", path, size))
		return errTruncate
	}

	_, _, err := s.Attach(DirectorAttach{Driver: DirectorDriverCodex, Replace: true, User: "bob", Cwd: "/work/b"})
	want := "org: director: attach: org: director: append to " + s.Path() + ": no space left on device, and director.jsonl could not be restored to its size before the append (truncate refused), " +
		"so it may hold part of the write: read the registration with ralph org director status"
	if err == nil || err.Error() != want {
		t.Fatalf("Attach with replace err = %v, want %q", err, want)
	}
	if !errors.Is(err, ErrDirectorWriteNotRestored) || !errors.Is(err, syscall.ENOSPC) || !errors.Is(err, errTruncate) {
		t.Errorf("err = %v: want ErrDirectorWriteNotRestored, ENOSPC, and the truncate error wrapped", err)
	}
	if want := fmt.Sprintf("%s %d", s.Path(), len(before)); len(cuts) != 1 || cuts[0] != want {
		t.Errorf("truncate calls = %q, want one to %q", cuts, want)
	}
	if got := readDirectorFile(t, s); got != onDisk || got == before {
		t.Fatalf("director.jsonl = %q, want the partial write left in place %q", got, onDisk)
	}
	if res := mustReadDirector(t, s); res.Current != nil || res.CorruptLines != 1 {
		t.Fatalf("Read = %+v (current %+v), want no registration and the cut attached corrupt", res, res.Current)
	}
}

// Plain attach and detach fail the same way: a write cut off partway is cut
// back, the registration is as it was, and the error says nothing was
// recorded. An attach that created director.jsonl leaves it empty, which
// reads as no registration.
func TestDirectorStore_PlainAttachAndDetachCutOffPartwayAreRestored(t *testing.T) {
	t.Run("attach into a missing file", func(t *testing.T) {
		s, _, _ := newTestDirectorStore(t)
		var onDisk string
		s.write = directorPartialWrite(t, s, &onDisk)

		_, _, err := s.Attach(DirectorAttach{Driver: DirectorDriverClaude})
		want := "org: director: attach: nothing was recorded: org: director: append to " + s.Path() + ": no space left on device"
		if err == nil || err.Error() != want {
			t.Fatalf("Attach err = %v, want %q", err, want)
		}
		if !strings.HasPrefix(onDisk, `{"ts":"2026-10-15T09:00:00Z","id":"d1",`) || strings.HasSuffix(onDisk, "\n") {
			t.Fatalf("the file after the partial write = %q, want part of the attached", onDisk)
		}
		if got := readDirectorFile(t, s); got != "" {
			t.Fatalf("director.jsonl = %q, want it cut back to empty", got)
		}
		if res := mustReadDirector(t, s); res.Current != nil || res.CorruptLines != 0 || res.IgnoredEvents != 0 {
			t.Fatalf("Read = %+v (current %+v), want no registration and nothing skipped", res, res.Current)
		}
		s.write = nil
		if _, err := s.Detach(DirectorDetach{Force: true}); !errors.Is(err, ErrDirectorNotAttached) {
			t.Errorf("force Detach of the empty file err = %v, want ErrDirectorNotAttached", err)
		}
		if reg := mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude}); reg.ID != "d1" {
			t.Fatalf("Attach after the failure = %s, want d1", reg.ID)
		}
	})
	t.Run("detach", func(t *testing.T) {
		s, _, clock := newTestDirectorStore(t)
		mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude})
		before := readDirectorFile(t, s)
		*clock = clock.Add(time.Minute)
		var onDisk string
		s.write = directorPartialWrite(t, s, &onDisk)

		_, err := s.Detach(DirectorDetach{ID: "d1"})
		want := "org: director: detach d1: nothing was recorded, so d1 (driver claude, attached " + directorTestNow + ") is still the current registration: org: director: append to " + s.Path() + ": no space left on device"
		if err == nil || err.Error() != want {
			t.Fatalf("Detach err = %v, want %q", err, want)
		}
		if cut, ok := strings.CutPrefix(onDisk, before); !ok || !strings.HasPrefix(cut, `{"ts":"2026-10-15T09:01:00Z","id":"d1",`) || strings.HasSuffix(cut, "\n") {
			t.Fatalf("the file after the partial write = %q, want the attached and part of the detached", onDisk)
		}
		if got := readDirectorFile(t, s); got != before {
			t.Fatalf("director.jsonl = %q, want it cut back to %q", got, before)
		}
		if id := currentDirectorID(mustReadDirector(t, s)); id != "d1" {
			t.Fatalf("current = %q, want d1", id)
		}
		s.write = nil
		if _, err := s.Detach(DirectorDetach{ID: "d1"}); err != nil {
			t.Fatalf("Detach d1 after the failure: %v", err)
		}
	})
}

// A close that fails after a full write is cut back the same way. The write
// seam writes every line and closes the file itself, so appendLocked's own
// Close fails (os.ErrClosed) with both lines on disk.
func TestDirectorAttach_ACloseFailureAfterAFullWriteIsRestored(t *testing.T) {
	s, _, clock := newTestDirectorStore(t)
	mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude, User: "alice", Cwd: "/work/a"})
	before := readDirectorFile(t, s)
	*clock = clock.Add(time.Minute)
	var onDisk string
	s.write = func(f *os.File, b []byte) (int, error) {
		n, err := f.Write(b)
		if cerr := f.Close(); cerr != nil {
			t.Errorf("close in the write seam: %v", cerr)
		}
		raw, _ := os.ReadFile(s.Path())
		onDisk = string(raw)
		return n, err
	}

	_, _, err := s.Attach(DirectorAttach{Driver: DirectorDriverCodex, Replace: true, User: "bob", Cwd: "/work/b"})
	want := "org: director: attach: nothing was recorded, so d1 (driver claude, attached " + directorTestNow + ") is still the current registration: org: director: close " + s.Path() + ": "
	if err == nil || !strings.HasPrefix(err.Error(), want) || !errors.Is(err, os.ErrClosed) || errors.Is(err, ErrDirectorWriteNotRestored) {
		t.Fatalf("Attach with replace err = %v, want it to start with %q and wrap os.ErrClosed", err, want)
	}
	if cut, ok := strings.CutPrefix(onDisk, before); !ok || strings.Count(cut, "\n") != 2 || !strings.Contains(cut, `"id":"d2","event":"attached"`) {
		t.Fatalf("the file after the write = %q, want the detached and the attached in full", onDisk)
	}
	if got := readDirectorFile(t, s); got != before {
		t.Fatalf("director.jsonl = %q, want it cut back to %q", got, before)
	}
	if id := currentDirectorID(mustReadDirector(t, s)); id != "d1" {
		t.Fatalf("current = %q, want d1", id)
	}
}

func TestDirectorAttach_ReplaceWithNoRegistrationOnlyAttaches(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	writeDirectorFile(t, s, directorAttachedLine("d1", "claude")+"\n"+directorDetachedLine("d1")+"\n")
	reg, replaced, err := s.Attach(DirectorAttach{Driver: DirectorDriverClaude, Replace: true})
	if err != nil || reg.ID != "d2" || replaced != nil {
		t.Fatalf("Attach with replace and nothing current = %+v, %v, %v; want d2, nil, nil", reg, replaced, err)
	}
	if lines := directorFileLines(t, s); len(lines) != 3 || !strings.Contains(lines[2], `"id":"d2","event":"attached"`) {
		t.Fatalf("director.jsonl lines = %q, want one attached added", lines)
	}
}

func TestDirectorDetach_WithTheCurrentIDEndsTheRegistration(t *testing.T) {
	s, _, clock := newTestDirectorStore(t)
	mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude, User: "alice", Cwd: "/work/a"})
	second := mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverCodex, Replace: true, User: "bob", Cwd: "/work/b"})
	*clock = clock.Add(2 * time.Minute)

	ended, err := s.Detach(DirectorDetach{ID: "d2", User: "bob", Cwd: "/work/b", HerdrPane: true})
	if err != nil {
		t.Fatalf("Detach d2: %v", err)
	}
	if !reflect.DeepEqual(ended, second) {
		t.Errorf("Detach returned %+v, want the ended registration %+v", ended, second)
	}
	lines := directorFileLines(t, s)
	if want := `{"ts":"2026-10-15T09:02:00Z","id":"d2","event":"detached","user":"bob","cwd":"/work/b","herdr_pane":true}`; lines[len(lines)-1] != want {
		t.Fatalf("last line = %s, want %s", lines[len(lines)-1], want)
	}
	res := mustReadDirector(t, s)
	if res.Current != nil || res.IgnoredEvents != 0 {
		t.Fatalf("Read after detach = %+v (current %+v), want no registration", res, res.Current)
	}
	// The ids of ended registrations are not handed out again.
	if reg := mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude}); reg.ID != "d3" {
		t.Fatalf("Attach after detach = %s, want d3", reg.ID)
	}
}

func TestDirectorDetach_WithNoRegistrationIsRefused(t *testing.T) {
	calls := []DirectorDetach{{ID: "d1"}, {Force: true}}

	t.Run("missing state dir", func(t *testing.T) {
		state := filepath.Join(t.TempDir(), "absent", "org")
		s := NewDirectorStore(state)
		for _, d := range calls {
			if _, err := s.Detach(d); !errors.Is(err, ErrDirectorNotAttached) {
				t.Errorf("Detach(%+v) err = %v, want ErrDirectorNotAttached", d, err)
			}
		}
		if _, err := os.Stat(state); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Detach created the state dir (stat err = %v)", err)
		}
	})
	t.Run("empty state dir", func(t *testing.T) {
		s, dir, _ := newTestDirectorStore(t)
		for _, d := range calls {
			_, err := s.Detach(d)
			if !errors.Is(err, ErrDirectorNotAttached) {
				t.Errorf("Detach(%+v) err = %v, want ErrDirectorNotAttached", d, err)
			}
			if d.ID != "" && (err == nil || !strings.Contains(err.Error(), "org: director: detach d1: no director is attached")) {
				t.Errorf("Detach(%+v) err = %v, want org: director: detach d1: no director is attached", d, err)
			}
		}
		if names := dirEntryNames(t, dir); len(names) != 0 {
			t.Fatalf("Detach left %v in the empty state dir, want neither director.jsonl nor director.lock", names)
		}
	})
	t.Run("the last registration was detached", func(t *testing.T) {
		s, _, _ := newTestDirectorStore(t)
		content := directorAttachedLine("d1", "claude") + "\n" + directorDetachedLine("d1") + "\n"
		writeDirectorFile(t, s, content)
		for _, d := range calls {
			if _, err := s.Detach(d); !errors.Is(err, ErrDirectorNotAttached) {
				t.Errorf("Detach(%+v) err = %v, want ErrDirectorNotAttached", d, err)
			}
		}
		if got := readDirectorFile(t, s); got != content {
			t.Fatalf("director.jsonl changed to %q", got)
		}
	})
}

func TestDirectorDetach_AReplacedDirectorCannotEndItsSuccessor(t *testing.T) {
	dir := t.TempDir()
	clock := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	now := func() time.Time { return clock }
	// A and B are two sessions on the same ledger.
	a, b := NewDirectorStore(dir), NewDirectorStore(dir)
	a.Now, b.Now = now, now

	if reg := mustAttachDirector(t, a, DirectorAttach{Driver: DirectorDriverClaude, User: "alice"}); reg.ID != "d1" {
		t.Fatalf("A's Attach = %s, want d1", reg.ID)
	}
	clock = clock.Add(time.Minute)
	if reg := mustAttachDirector(t, b, DirectorAttach{Driver: DirectorDriverCodex, Replace: true, User: "bob"}); reg.ID != "d2" {
		t.Fatalf("B's Attach with replace = %s, want d2", reg.ID)
	}
	before := readDirectorFile(t, a)

	// A finishes as its steps say: detach with the id it was given.
	_, err := a.Detach(DirectorDetach{ID: "d1", User: "alice"})
	if !errors.Is(err, ErrDirectorIDMismatch) {
		t.Fatalf("A's Detach d1 err = %v, want ErrDirectorIDMismatch", err)
	}
	for _, want := range []string{"detach d1", "the current registration is d2 (driver codex, attached 2026-10-15T09:01:00Z)", "replaced", "does not need to detach"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err %q does not say %q", err, want)
		}
	}
	// An id no registration ever had is refused the same way.
	if _, err := a.Detach(DirectorDetach{ID: "d9"}); !errors.Is(err, ErrDirectorIDMismatch) {
		t.Errorf("Detach d9 err = %v, want ErrDirectorIDMismatch", err)
	}
	if got := readDirectorFile(t, a); got != before {
		t.Fatalf("director.jsonl changed to %q", got)
	}
	if res := mustReadDirector(t, b); currentDirectorID(res) != "d2" || res.Current.User != "bob" {
		t.Fatalf("current = %+v, want B's d2 still attached", res.Current)
	}
}

func TestDirectorDetach_ForceEndsTheCurrentRegistrationWhateverItsID(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude})
	second := mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverCodex, Replace: true, User: "bob"})

	ended, err := s.Detach(DirectorDetach{Force: true, User: "root", Cwd: "/r"})
	if err != nil {
		t.Fatalf("force Detach: %v", err)
	}
	if ended.ID != second.ID {
		t.Errorf("force Detach ended %s, want %s", ended.ID, second.ID)
	}
	lines := directorFileLines(t, s)
	if want := `{"ts":"2026-10-15T09:00:00Z","id":"d2","event":"detached","user":"root","cwd":"/r","herdr_pane":false,"force":true}`; lines[len(lines)-1] != want {
		t.Fatalf("last line = %s, want %s", lines[len(lines)-1], want)
	}
	if res := mustReadDirector(t, s); res.Current != nil || res.IgnoredEvents != 0 {
		t.Fatalf("Read after force detach = %+v (current %+v), want no registration", res, res.Current)
	}
}

func TestDirectorDetach_NeedsExactlyOneOfAnIDAndForce(t *testing.T) {
	for _, tc := range []struct {
		name string
		d    DirectorDetach
		want string
	}{
		{"neither", DirectorDetach{User: "u"}, "needs the registration id or force"},
		{"both", DirectorDetach{ID: "d1", Force: true}, "the registration id or force, not both"},
		{"d0", DirectorDetach{ID: "d0"}, `"d0" is not a registration id`},
		{"leading zero", DirectorDetach{ID: "d01"}, `"d01" is not a registration id`},
		{"not a number", DirectorDetach{ID: "dx"}, `"dx" is not a registration id`},
		{"only the prefix", DirectorDetach{ID: "d"}, `"d" is not a registration id`},
		{"upper case", DirectorDetach{ID: "D1"}, `"D1" is not a registration id`},
		{"an inbox id", DirectorDetach{ID: "e1"}, `"e1" is not a registration id`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := newTestDirectorStore(t)
			mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude})
			before := readDirectorFile(t, s)
			_, err := s.Detach(tc.d)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Detach(%+v) err = %v, want %q", tc.d, err, tc.want)
			}
			if errors.Is(err, ErrDirectorNotAttached) || errors.Is(err, ErrDirectorIDMismatch) {
				t.Errorf("err = %v: refused for its arguments, not for the registration", err)
			}
			if got := readDirectorFile(t, s); got != before {
				t.Fatalf("director.jsonl changed to %q", got)
			}
			if id := currentDirectorID(mustReadDirector(t, s)); id != "d1" {
				t.Fatalf("current = %q, want d1", id)
			}
		})
	}
}

func TestDirectorAttach_RefusesAnUnknownDriverBeforeReadingAnything(t *testing.T) {
	for _, driver := range []string{"", "Claude", "gemini", " codex"} {
		t.Run(fmt.Sprintf("%q", driver), func(t *testing.T) {
			state := filepath.Join(t.TempDir(), "absent", "org")
			_, _, err := NewDirectorStore(state).Attach(DirectorAttach{Driver: driver})
			if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("unknown driver %q: use claude or codex", driver)) {
				t.Fatalf("Attach err = %v, want unknown driver %q", err, driver)
			}
			if _, err := os.Stat(state); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("Attach with a bad driver created the state dir (stat err = %v)", err)
			}
		})
	}
	// The driver is checked before the file is read: an unreadable
	// director.jsonl still gets the driver error.
	s, _, _ := newTestDirectorStore(t)
	if err := os.Mkdir(s.Path(), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, _, err := s.Attach(DirectorAttach{Driver: "gemini"}); err == nil || !strings.Contains(err.Error(), "unknown driver") {
		t.Fatalf("Attach err = %v, want the driver error", err)
	}
}

func TestDirectorAttach_NeverReusesTheIDOfADamagedLine(t *testing.T) {
	// "%u00647" becomes a JSON escape for "d" followed by 7, in both lines.
	escaped := strings.ReplaceAll(`{"ts":"t","id":"%u00647","event":"attached","driver":"claude","wake":["pull"],"user":"u","cwd":"/w","herdr_pane":false}`+"\n"+
		`{"ts":"t","id":"%u00647","event":"detached","user":"u","cwd":"/w","herdr_pane":false}`, "%u", "\\"+"u")
	for _, tc := range []struct {
		name        string
		content     string
		wantID      string
		wantCorrupt int
		wantIgnored int
	}{
		{
			name:        "a damaged line holds d7",
			content:     directorAttachedLine("d1", "claude") + "\n" + `{"ts":"t","id":"d7","event":"attac` + "\n" + directorDetachedLine("d1") + "\n",
			wantID:      "d8",
			wantCorrupt: 1,
		},
		{
			name:        "spaces around the colon",
			content:     `{"ts":"t", "id" : "d5", "event":` + "\n" + directorAttachedLine("d2", "codex") + "\n" + directorDetachedLine("d2") + "\n",
			wantID:      "d6",
			wantCorrupt: 1,
		},
		{
			name:        "an id too large for int64 in a damaged line is skipped",
			content:     `{"id":"d99999999999999999999999",` + "\n" + directorAttachedLine("d2", "codex") + "\n" + directorDetachedLine("d2") + "\n",
			wantID:      "d3",
			wantCorrupt: 1,
		},
		{
			// The decoded id is d7, which the raw-text scan alone would
			// not find.
			name:    "an id written with a JSON escape",
			content: escaped + "\n",
			wantID:  "d8",
		},
		{
			// An ignored attached (bad driver) still holds its id.
			name:        "an ignored attached",
			content:     directorAttachedLine("d4", "gemini") + "\n",
			wantID:      "d5",
			wantIgnored: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := newTestDirectorStore(t)
			writeDirectorFile(t, s, tc.content)
			if res := mustReadDirector(t, s); res.Current != nil || res.CorruptLines != tc.wantCorrupt || res.IgnoredEvents != tc.wantIgnored {
				t.Fatalf("Read before = %+v (current %+v), want no registration, corrupt %d, ignored %d", res, res.Current, tc.wantCorrupt, tc.wantIgnored)
			}
			if reg := mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude}); reg.ID != tc.wantID {
				t.Fatalf("Attach = %s, want %s", reg.ID, tc.wantID)
			}
			if id := currentDirectorID(mustReadDirector(t, s)); id != tc.wantID {
				t.Fatalf("current = %q, want %s", id, tc.wantID)
			}
		})
	}
}

func TestDirectorAttach_TornLastLineStaysSeparateAndTheNewEventIsReadable(t *testing.T) {
	for _, tc := range []struct {
		name   string
		torn   string
		wantID string
	}{
		{name: "cut after the id", torn: `{"ts":"2026-10-15T00:00:02Z","id":"d5","event":"attached","dri`, wantID: "d6"},
		{name: "cut inside the id", torn: `{"ts":"2026-10-15T00:00:02Z","id":"d12`, wantID: "d13"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := newTestDirectorStore(t)
			writeDirectorFile(t, s, directorAttachedLine("d4", "claude")+"\n"+directorDetachedLine("d4")+"\n"+tc.torn)
			if reg := mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverCodex}); reg.ID != tc.wantID {
				t.Fatalf("Attach = %s, want %s", reg.ID, tc.wantID)
			}
			if _, err := s.Detach(DirectorDetach{ID: tc.wantID}); err != nil {
				t.Fatalf("Detach of the new registration: %v", err)
			}
			lines := strings.Split(readDirectorFile(t, s), "\n")
			if len(lines) != 6 || lines[2] != tc.torn || lines[5] != "" ||
				!strings.Contains(lines[3], `"id":"`+tc.wantID+`","event":"attached"`) ||
				!strings.Contains(lines[4], `"id":"`+tc.wantID+`","event":"detached"`) {
				t.Fatalf("director.jsonl lines = %q, want d4's two lines, the torn line on its own, the new attached and detached, and a final newline", lines)
			}
			if res := mustReadDirector(t, s); res.Current != nil || res.CorruptLines != 1 || res.IgnoredEvents != 0 {
				t.Fatalf("Read = %+v (current %+v), want no registration, corrupt 1, ignored 0", res, res.Current)
			}
		})
	}
}

func TestDirectorRead_FoldsTheRegistrationFromItsAttachedEvent(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	writeDirectorFile(t, s, strings.Join([]string{
		directorAttachedLine("d1", "claude"),
		directorDetachedLine("d1"),
		`{"ts":"2026-10-15T08:00:00Z","id":"d2","event":"attached","driver":"codex","wake":["pull","push"],"pane_id":"w1:p2","user":"carol","cwd":"/work/c","herdr_pane":true}`,
	}, "\n")+"\n")
	res := mustReadDirector(t, s)
	want := DirectorRegistration{
		ID: "d2", Driver: DirectorDriverCodex, Wake: []string{"pull", "push"}, PaneID: "w1:p2",
		AttachedAt: "2026-10-15T08:00:00Z", User: "carol", Cwd: "/work/c", HerdrPane: true,
	}
	if res.Current == nil || !reflect.DeepEqual(*res.Current, want) || res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Fatalf("Read = %+v (current %+v), want %+v", res, res.Current, want)
	}
}

func TestDirectorRead_SkipsAndCountsLinesItCannotUse(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	writeDirectorFile(t, s, strings.Join([]string{
		`not json`,
		`{"ts":"t","id":"d3",`,
		`[1,2]`,
		``,
		`   `,
		directorAttachedLine("d1", "claude"),
		// While d1 is current: a second attached, a detached for another
		// id, and an unknown kind are ignored.
		directorAttachedLine("d2", "codex"),
		directorDetachedLine("d2"),
		`{"ts":"t","id":"d1","event":"expired"}`,
		directorDetachedLine("d1"),
		// Nothing is current now, so each of these would become current
		// but for its flaw: an id not of the form d<N>, a driver other
		// than claude and codex, an id an earlier attached used, and a
		// detached with nothing current.
		directorAttachedLine("d01", "claude"),
		directorAttachedLine("d0", "claude"),
		directorAttachedLine("dx", "claude"),
		directorAttachedLine("", "claude"),
		directorAttachedLine("D4", "claude"),
		directorAttachedLine("e5", "claude"),
		directorAttachedLine("d5", "gemini"),
		`{"ts":"t","id":"d6","event":"attached","wake":["pull"],"user":"u","cwd":"/w","herdr_pane":false}`,
		directorAttachedLine("d1", "codex"),
		directorDetachedLine("d1"),
		// d4 then applies.
		directorAttachedLine("d4", "codex"),
	}, "\n")+"\n")

	res := mustReadDirector(t, s)
	if res.CorruptLines != 3 {
		t.Errorf("CorruptLines = %d, want 3 (blank lines are not counted)", res.CorruptLines)
	}
	if res.IgnoredEvents != 13 {
		t.Errorf("IgnoredEvents = %d, want 13", res.IgnoredEvents)
	}
	if res.Current == nil || res.Current.ID != "d4" || res.Current.Driver != DirectorDriverCodex {
		t.Fatalf("current = %+v, want d4 codex", res.Current)
	}
	// The next id is past every d<N> in the file: the corrupt d3 and the
	// ignored d5 and d6 included.
	reg, replaced, err := s.Attach(DirectorAttach{Driver: DirectorDriverClaude, Replace: true})
	if err != nil || reg.ID != "d7" || replaced == nil || replaced.ID != "d4" {
		t.Fatalf("Attach with replace = %+v, %+v, %v; want d7 replacing d4", reg, replaced, err)
	}
}

// A detached for an id that is not the current registration's is ignored and
// leaves that registration current. TestDirectorRead_SkipsAndCountsLinesItCannotUse
// has such a line, but a fold that let it end d1 would ignore the later
// detached d1 instead, and the two cancel out there (mutant D5 of the
// org-director-attach test report), so this file holds only the two lines.
func TestDirectorRead_ADetachedForAnotherIDLeavesTheCurrentRegistration(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	writeDirectorFile(t, s, directorAttachedLine("d1", "claude")+"\n"+directorDetachedLine("d2")+"\n")
	res := mustReadDirector(t, s)
	if currentDirectorID(res) != "d1" || res.CorruptLines != 0 || res.IgnoredEvents != 1 {
		t.Fatalf("Read = %+v (current %+v), want d1 current and the detached d2 ignored", res, res.Current)
	}
}

func TestDirectorAttach_FailsWhenNoIDIsLeft(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	content := directorAttachedLine("d9223372036854775807", "claude") + "\n" + directorDetachedLine("d9223372036854775807") + "\n"
	writeDirectorFile(t, s, content)
	if _, _, err := s.Attach(DirectorAttach{Driver: DirectorDriverClaude}); err == nil || !strings.Contains(err.Error(), "no registration id is left") {
		t.Fatalf("Attach err = %v, want no registration id is left", err)
	}
	if got := readDirectorFile(t, s); got != content {
		t.Fatalf("director.jsonl changed to %q", got)
	}
}

func TestDirectorAttach_ConcurrentAttachesWithoutReplace_OnlyOneSucceeds(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	const n = 10
	regs := make([]DirectorRegistration, n)
	errs := make([]error, n)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			regs[i], _, errs[i] = s.Attach(DirectorAttach{Driver: DirectorDriverClaude, User: fmt.Sprintf("user-%d", i)})
		}()
	}
	close(start)
	wg.Wait()

	winner := -1
	for i := range n {
		switch {
		case errs[i] == nil:
			if winner >= 0 {
				t.Fatalf("Attach %d and %d both succeeded", winner, i)
			}
			winner = i
		case !errors.Is(errs[i], ErrDirectorAttached):
			t.Fatalf("Attach %d err = %v, want ErrDirectorAttached", i, errs[i])
		}
	}
	if winner < 0 || regs[winner].ID != "d1" {
		t.Fatalf("winner %d with %+v, want one Attach to get d1", winner, regs)
	}
	if lines := directorFileLines(t, s); len(lines) != 1 {
		t.Fatalf("director.jsonl has %d lines, want 1: %q", len(lines), lines)
	}
	if res := mustReadDirector(t, s); currentDirectorID(res) != "d1" || res.Current.User != fmt.Sprintf("user-%d", winner) {
		t.Fatalf("current = %+v, want the winner's d1", res.Current)
	}
}

func TestDirectorAttach_ConcurrentReplacesEachEndTheOneBefore(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	const n = 10
	regs := make([]DirectorRegistration, n)
	replaced := make([]*DirectorRegistration, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			regs[i], replaced[i], errs[i] = s.Attach(DirectorAttach{Driver: DirectorDriverCodex, Replace: true})
		}()
	}
	wg.Wait()

	ids, ended := map[string]bool{}, map[string]bool{}
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("Attach %d: %v", i, errs[i])
		}
		ids[regs[i].ID] = true
		if replaced[i] != nil {
			ended[replaced[i].ID] = true
		}
	}
	if len(ids) != n || len(ended) != n-1 {
		t.Fatalf("%d distinct ids and %d ended registrations, want %d and %d", len(ids), len(ended), n, n-1)
	}
	res := mustReadDirector(t, s)
	if res.CorruptLines != 0 || res.IgnoredEvents != 0 || currentDirectorID(res) != fmt.Sprintf("d%d", n) || ended[res.Current.ID] {
		t.Fatalf("Read = %+v (current %+v), want d%d current and nothing skipped", res, res.Current, n)
	}
	if lines := directorFileLines(t, s); len(lines) != 2*n-1 {
		t.Fatalf("director.jsonl has %d lines, want %d", len(lines), 2*n-1)
	}
}

func TestDirectorLock_IsItsOwnLockAndSerializesWrites(t *testing.T) {
	s, dir, _ := newTestDirectorStore(t)

	// Holding the manifest and inbox locks does not block a director write.
	start := time.Now()
	if err := withManifestLock(dir, func() error {
		return withFileLock(dir, inboxLockFile, "inbox", func() error {
			_, _, err := s.Attach(DirectorAttach{Driver: DirectorDriverClaude})
			return err
		})
	}); err != nil {
		t.Fatalf("Attach under the manifest and inbox locks: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("Attach under the manifest and inbox locks took %s: it waited for one of them", elapsed)
	}

	// Holding the director lock does block it until released.
	held, release := make(chan struct{}), make(chan struct{})
	holderDone := make(chan error, 1)
	go func() {
		holderDone <- withFileLock(dir, directorLockFile, "director", func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	type result struct {
		reg DirectorRegistration
		err error
	}
	done := make(chan result, 1)
	go func() {
		reg, _, err := s.Attach(DirectorAttach{Driver: DirectorDriverCodex, Replace: true})
		done <- result{reg, err}
	}()
	select {
	case r := <-done:
		t.Fatalf("Attach returned %+v while the director lock was held", r)
	case <-time.After(150 * time.Millisecond):
	}
	close(release)
	if err := <-holderDone; err != nil {
		t.Fatalf("lock holder: %v", err)
	}
	if r := <-done; r.err != nil || r.reg.ID != "d2" {
		t.Fatalf("Attach after release = %+v, want d2", r)
	}
}

func TestDirectorStore_ReportsAnUnreadableFileAndAnUnavailableLock(t *testing.T) {
	t.Run("director.jsonl is a directory", func(t *testing.T) {
		s, _, _ := newTestDirectorStore(t)
		if err := os.Mkdir(s.Path(), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if _, err := s.Read(); err == nil || !strings.Contains(err.Error(), "org: director: read ") {
			t.Errorf("Read err = %v, want org: director: read ...", err)
		}
		if _, _, err := s.Attach(DirectorAttach{Driver: DirectorDriverClaude}); err == nil || !strings.Contains(err.Error(), "org: director: read ") {
			t.Errorf("Attach err = %v, want the read error", err)
		}
		if _, err := s.Detach(DirectorDetach{ID: "d1"}); err == nil || errors.Is(err, ErrDirectorNotAttached) {
			t.Errorf("Detach err = %v, want the read error, not ErrDirectorNotAttached", err)
		}
	})
	t.Run("director.lock is a directory", func(t *testing.T) {
		s, dir, _ := newTestDirectorStore(t)
		if err := os.Mkdir(filepath.Join(dir, directorLockFile), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if _, _, err := s.Attach(DirectorAttach{Driver: DirectorDriverClaude}); err == nil ||
			!strings.Contains(err.Error(), "org: open director lock file ") {
			t.Errorf("Attach err = %v, want org: open director lock file ...", err)
		}
		if _, err := os.Stat(s.Path()); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Attach wrote director.jsonl without the lock (stat err = %v)", err)
		}
	})
}
