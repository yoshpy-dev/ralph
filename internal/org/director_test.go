package org

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
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
	want := "org: director: attach: nothing was recorded, so d1 (driver claude, attached " + directorTestNow + ") is still the current registration: open " + s.Path() + ": "
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
// attached (a full disk) is cut back. Left in place, the detached would end
// d1 and the cut attached would be a corrupt line, so no registration would
// be current while the error said d1 still was.
func TestDirectorAttach_AReplaceCutOffPartwayIsRestored(t *testing.T) {
	s, _, clock := newTestDirectorStore(t)
	mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude, User: "alice", Cwd: "/work/a"})
	before := readDirectorFile(t, s)
	*clock = clock.Add(time.Minute)
	var onDisk string
	s.write = directorPartialWrite(t, s, &onDisk)

	reg, replaced, err := s.Attach(DirectorAttach{Driver: DirectorDriverCodex, Replace: true, User: "bob", Cwd: "/work/b"})
	want := "org: director: attach: nothing was recorded, so d1 (driver claude, attached " + directorTestNow + ") is still the current registration: append to " + s.Path() + ": no space left on device"
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
// current. It wraps ErrDirectorWriteNotRestored, says "org: director:" once,
// and names no command: the store does not know the --state-dir of the call,
// so the CLI adds the command that reads the registration (notRestoredError,
// self-review C2-L1 and C2-L2 of org-director-attach).
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
	want := "org: director: attach: append to " + s.Path() + ": no space left on device, and director.jsonl could not be restored to its size before the append (truncate refused), " +
		"so it may hold part of the write"
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
		want := "org: director: attach: nothing was recorded: append to " + s.Path() + ": no space left on device"
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
		want := "org: director: detach d1: nothing was recorded, so d1 (driver claude, attached " + directorTestNow + ") is still the current registration: append to " + s.Path() + ": no space left on device"
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
	want := "org: director: attach: nothing was recorded, so d1 (driver claude, attached " + directorTestNow + ") is still the current registration: close " + s.Path() + ": "
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
// detached d1 instead, and the two cancel out there: that test stays green
// when the fold's id check on a detached is dropped. So this file holds only
// the two lines.
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

// The headless registration of `ralph org director start` (plan
// docs/plans/active/2026-10-15-org-director-start.md, D2, D3, D4, and D9).

// directorTestToken is the launch token a newTestHeadlessStore hands out
// when the test names none.
const directorTestToken = "0123456789ab"

// newTestHeadlessStore is newTestDirectorStore whose AttachHeadless gets the
// given launch tokens in order (directorTestToken when none is given), so a
// test can name the lines it writes. It is for one goroutine only.
func newTestHeadlessStore(t *testing.T, tokens ...string) (*DirectorStore, string, *time.Time) {
	t.Helper()
	s, dir, clock := newTestDirectorStore(t)
	if len(tokens) == 0 {
		tokens = []string{directorTestToken}
	}
	s.token = func() (string, error) {
		if len(tokens) == 0 {
			t.Fatal("AttachHeadless asked for more launch tokens than the test gave")
		}
		token := tokens[0]
		tokens = tokens[1:]
		return token, nil
	}
	return s, dir, clock
}

func testHeadlessAttach() DirectorHeadlessAttach {
	return DirectorHeadlessAttach{
		Driver: DirectorDriverCodex, Model: "gpt-x", PermissionMode: PermissionModeAutonomous, Wake: []string{DirectorWakePull},
		User: "alice", Cwd: "/work/a",
	}
}

func mustAttachHeadless(t *testing.T, s *DirectorStore) DirectorRegistration {
	t.Helper()
	reg, err := s.AttachHeadless(testHeadlessAttach())
	if err != nil {
		t.Fatalf("AttachHeadless: %v", err)
	}
	return reg
}

// testDirectorWho is who writes the later events of a headless registration
// in these tests.
var testDirectorWho = DirectorWho{User: "bob", Cwd: "/work/b"}

// directorHeadlessAttachedLine is the attached line of a headless
// registration as a test fixture.
func directorHeadlessAttachedLine(id, token string) string {
	return fmt.Sprintf(`{"ts":"2026-10-15T00:00:00Z","id":%q,"event":"attached","driver":"codex","wake":["pull"],"launch":"herdr","token":%q,"model":"m","permission_mode":"guarded","user":"u","cwd":"/w","herdr_pane":false}`, id, token)
}

// directorEventLine is a director.jsonl line of kind event for id as a test
// fixture, with fields (`"key":"value",` pairs, each with its comma) after
// the event.
func directorEventLine(id, event, fields string) string {
	return fmt.Sprintf(`{"ts":"2026-10-15T00:00:02Z","id":%q,"event":%q,%s"user":"u","cwd":"/w","herdr_pane":false}`, id, event, fields)
}

func directorWorkspaceLine(id, workspaceID string) string {
	return directorEventLine(id, DirectorEventWorkspaceCreated, fmt.Sprintf(`"workspace_id":%q,`, workspaceID))
}

func directorPaneLine(id, paneID string) string {
	return directorEventLine(id, DirectorEventPaneCreated, fmt.Sprintf(`"pane_id":%q,`, paneID))
}

// directorRegSummary names r by its id, state, workspace, and pane.
func directorRegSummary(r DirectorRegistration) string {
	return fmt.Sprintf("%s %s ws=%s pane=%s", r.ID, r.State, r.WorkspaceID, r.PaneID)
}

func openHeadlessSummary(res DirectorReadResult) []string {
	var out []string
	for _, r := range res.OpenHeadless {
		out = append(out, directorRegSummary(r))
	}
	return out
}

func TestDirectorAttachHeadless_RecordsTheLaunchAndItsEvents(t *testing.T) {
	s, _, clock := newTestHeadlessStore(t)
	reg, err := s.AttachHeadless(testHeadlessAttach())
	if err != nil {
		t.Fatalf("AttachHeadless: %v", err)
	}
	want := DirectorRegistration{
		ID: "d1", Driver: DirectorDriverCodex, Wake: []string{DirectorWakePull},
		Launch: DirectorLaunchHerdr, State: DirectorStateStarting, Token: directorTestToken,
		Model: "gpt-x", PermissionMode: PermissionModeAutonomous,
		AttachedAt: directorTestNow, User: "alice", Cwd: "/work/a",
	}
	if !reflect.DeepEqual(reg, want) {
		t.Fatalf("AttachHeadless = %+v, want %+v", reg, want)
	}
	*clock = clock.Add(time.Second)
	if err := s.RecordWorkspace("d1", "w7", testDirectorWho); err != nil {
		t.Fatalf("RecordWorkspace: %v", err)
	}
	if err := s.RecordPane("d1", "w7:p1", DirectorWho{User: "bob", Cwd: "/work/b", HerdrPane: true}); err != nil {
		t.Fatalf("RecordPane: %v", err)
	}
	if still, err := s.MarkLaunched("d1", testDirectorWho); err != nil || !still {
		t.Fatalf("MarkLaunched = %v, %v; want true, nil", still, err)
	}

	wantLines := []string{
		`{"ts":"2026-10-15T09:00:00Z","id":"d1","event":"attached","driver":"codex","wake":["pull"],"launch":"herdr","token":"0123456789ab","model":"gpt-x","permission_mode":"autonomous","user":"alice","cwd":"/work/a","herdr_pane":false}`,
		`{"ts":"2026-10-15T09:00:01Z","id":"d1","event":"workspace_created","workspace_id":"w7","user":"bob","cwd":"/work/b","herdr_pane":false}`,
		`{"ts":"2026-10-15T09:00:01Z","id":"d1","event":"pane_created","pane_id":"w7:p1","user":"bob","cwd":"/work/b","herdr_pane":true}`,
		`{"ts":"2026-10-15T09:00:01Z","id":"d1","event":"launched","user":"bob","cwd":"/work/b","herdr_pane":false}`,
	}
	if got := directorFileLines(t, s); strings.Join(got, "\n") != strings.Join(wantLines, "\n") {
		t.Fatalf("director.jsonl =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(wantLines, "\n"))
	}
	res := mustReadDirector(t, s)
	want.State, want.WorkspaceID, want.PaneID = DirectorStateRunning, "w7", "w7:p1"
	if res.Current == nil || !reflect.DeepEqual(*res.Current, want) || res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Fatalf("Read = %+v (current %+v), want %+v", res, res.Current, want)
	}
	if len(res.OpenHeadless) != 1 || !reflect.DeepEqual(res.OpenHeadless[0], want) {
		t.Fatalf("OpenHeadless = %+v, want only %+v", res.OpenHeadless, want)
	}
	raw, err := json.Marshal(*res.Current)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	wantJSON := `{"id":"d1","driver":"codex","wake":["pull"],"pane_id":"w7:p1","launch":"herdr","state":"running","token":"0123456789ab","model":"gpt-x",` +
		`"permission_mode":"autonomous","workspace_id":"w7","attached_at":"2026-10-15T09:00:00Z","user":"alice","cwd":"/work/a","herdr_pane":false}`
	if string(raw) != wantJSON {
		t.Fatalf("JSON = %s, want %s", raw, wantJSON)
	}
}

// A director.jsonl of stage 8a-1 (no new field, no new event) folds as it
// did: the same current registration, nothing ignored, and no open headless
// registration.
func TestDirectorRead_AStage8a1FileReadsAsBefore(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	writeDirectorFile(t, s, strings.Join([]string{
		directorAttachedLine("d1", "claude"),
		directorDetachedLine("d1"),
		`{"ts":"2026-10-15T08:00:00Z","id":"d2","event":"attached","driver":"codex","wake":["pull"],"pane_id":"w1:p2","user":"carol","cwd":"/work/c","herdr_pane":true}`,
	}, "\n")+"\n")
	res := mustReadDirector(t, s)
	want := DirectorRegistration{
		ID: "d2", Driver: DirectorDriverCodex, Wake: []string{DirectorWakePull}, PaneID: "w1:p2",
		AttachedAt: "2026-10-15T08:00:00Z", User: "carol", Cwd: "/work/c", HerdrPane: true,
	}
	if res.Current == nil || !reflect.DeepEqual(*res.Current, want) || res.OpenHeadless != nil || res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Fatalf("Read = %+v (current %+v), want %+v and no open headless registration", res, res.Current, want)
	}
	if res.Current.Headless() {
		t.Fatalf("a promoted registration reads as headless: %+v", res.Current)
	}
}

func TestDirectorRead_FoldsHeadlessRegistrations(t *testing.T) {
	att := directorHeadlessAttachedLine
	ev := func(id, event string) string { return directorEventLine(id, event, "") }
	tok := directorTestToken
	for _, tc := range []struct {
		name        string
		lines       []string
		wantCurrent string // directorRegSummary of the current registration, "" for none
		wantOpen    []string
		wantIgnored int
		wantCorrupt int
	}{
		{
			name:        "starting",
			lines:       []string{att("d1", tok)},
			wantCurrent: "d1 starting ws= pane=",
			wantOpen:    []string{"d1 starting ws= pane="},
		},
		{
			name:        "running with its workspace and pane",
			lines:       []string{att("d1", tok), directorWorkspaceLine("d1", "w1"), directorPaneLine("d1", "p1"), ev("d1", DirectorEventLaunched)},
			wantCurrent: "d1 running ws=w1 pane=p1",
			wantOpen:    []string{"d1 running ws=w1 pane=p1"},
		},
		{
			name:     "detached but not stopped stays open",
			lines:    []string{att("d1", tok), directorWorkspaceLine("d1", "w1"), directorDetachedLine("d1")},
			wantOpen: []string{"d1 starting ws=w1 pane="},
		},
		{
			name:  "stopped while current, as MarkStopped writes it",
			lines: []string{att("d1", tok), directorPaneLine("d1", "p1"), ev("d1", DirectorEventLaunched), directorDetachedLine("d1"), ev("d1", DirectorEventStopped)},
		},
		{
			name:  "stopped after a detach",
			lines: []string{att("d1", tok), directorDetachedLine("d1"), ev("d1", DirectorEventStopped)},
		},
		{
			// A stopped with no detached is a hand edit: the registration
			// stays current, so attach and start refuse, but it is not open.
			name:        "stopped while current without a detached",
			lines:       []string{att("d1", tok), ev("d1", DirectorEventStopped)},
			wantCurrent: "d1 stopped ws= pane=",
		},
		{
			name: "the ids of a replaced registration are still recorded",
			lines: []string{
				att("d1", tok), directorDetachedLine("d1"), directorAttachedLine("d2", "claude"),
				directorWorkspaceLine("d1", "w1"), directorPaneLine("d1", "p1"),
			},
			wantCurrent: "d2  ws= pane=",
			wantOpen:    []string{"d1 starting ws=w1 pane=p1"},
		},
		{
			name:        "a launched for a registration that is not current is ignored",
			lines:       []string{att("d1", tok), directorDetachedLine("d1"), directorAttachedLine("d2", "claude"), ev("d1", DirectorEventLaunched)},
			wantCurrent: "d2  ws= pane=",
			wantOpen:    []string{"d1 starting ws= pane="},
			wantIgnored: 1,
		},
		{
			name:        "a second launched is ignored",
			lines:       []string{att("d1", tok), ev("d1", DirectorEventLaunched), ev("d1", DirectorEventLaunched)},
			wantCurrent: "d1 running ws= pane=",
			wantOpen:    []string{"d1 running ws= pane="},
			wantIgnored: 1,
		},
		{
			name:        "a second stopped is ignored",
			lines:       []string{att("d1", tok), directorDetachedLine("d1"), ev("d1", DirectorEventStopped), ev("d1", DirectorEventStopped)},
			wantIgnored: 1,
		},
		{
			name: "events after the stopped are ignored",
			lines: []string{
				att("d1", tok), directorDetachedLine("d1"), ev("d1", DirectorEventStopped),
				directorWorkspaceLine("d1", "w1"), directorPaneLine("d1", "p1"), ev("d1", DirectorEventLaunched),
			},
			wantIgnored: 3,
		},
		{
			name: "a headless attached with a malformed token is ignored",
			lines: []string{
				att("d1", "0123456789AB"), att("d2", ""), att("d3", "0123456789abc"), att("d4", "0123456789a"), att("d5", "0123456789ag"),
				att("d6", tok),
			},
			wantCurrent: "d6 starting ws= pane=",
			wantOpen:    []string{"d6 starting ws= pane="},
			wantIgnored: 5,
		},
		{
			name: "events for an id no registration has are ignored",
			lines: []string{
				att("d1", tok), directorWorkspaceLine("d9", "w9"), directorPaneLine("d9", "p9"),
				ev("d9", DirectorEventLaunched), ev("d9", DirectorEventStopped),
			},
			wantCurrent: "d1 starting ws= pane=",
			wantOpen:    []string{"d1 starting ws= pane="},
			wantIgnored: 4,
		},
		{
			name: "events for a promoted registration are ignored",
			lines: []string{
				directorAttachedLine("d1", "claude"), directorWorkspaceLine("d1", "w1"), directorPaneLine("d1", "p1"),
				ev("d1", DirectorEventLaunched), ev("d1", DirectorEventStopped),
			},
			wantCurrent: "d1  ws= pane=",
			wantIgnored: 4,
		},
		{
			name: "a second workspace or pane keeps the first",
			lines: []string{
				att("d1", tok), directorWorkspaceLine("d1", "w1"), directorWorkspaceLine("d1", "w2"),
				directorPaneLine("d1", "p1"), directorPaneLine("d1", "p2"),
			},
			wantCurrent: "d1 starting ws=w1 pane=p1",
			wantOpen:    []string{"d1 starting ws=w1 pane=p1"},
			wantIgnored: 2,
		},
		{
			name:        "a workspace_created or pane_created with no id is ignored",
			lines:       []string{att("d1", tok), ev("d1", DirectorEventWorkspaceCreated), ev("d1", DirectorEventPaneCreated)},
			wantCurrent: "d1 starting ws= pane=",
			wantOpen:    []string{"d1 starting ws= pane="},
			wantIgnored: 2,
		},
		{
			name: "the pane_id of a headless attached is not read",
			lines: []string{
				`{"ts":"2026-10-15T00:00:00Z","id":"d1","event":"attached","driver":"codex","wake":["pull"],"pane_id":"p0","launch":"herdr","token":"0123456789ab","user":"u","cwd":"/w","herdr_pane":false}`,
				directorPaneLine("d1", "p1"),
			},
			wantCurrent: "d1 starting ws= pane=p1",
			wantOpen:    []string{"d1 starting ws= pane=p1"},
		},
		{
			// A launch this binary does not know is still a registration, so
			// attach and start refuse on it; it is not headless.
			name: "an attached with an unknown launch is a registration but not headless",
			lines: []string{
				`{"ts":"2026-10-15T00:00:00Z","id":"d1","event":"attached","driver":"codex","wake":["pull"],"launch":"terminal","token":"zz","user":"u","cwd":"/w","herdr_pane":false}`,
				ev("d1", DirectorEventStopped),
			},
			wantCurrent: "d1  ws= pane=",
			wantIgnored: 1,
		},
		{
			name: "open headless registrations are ordered by their number",
			lines: []string{
				att("d10", tok), directorDetachedLine("d10"), att("d2", tok), directorDetachedLine("d2"), att("d3", tok),
			},
			wantCurrent: "d3 starting ws= pane=",
			wantOpen:    []string{"d2 starting ws= pane=", "d3 starting ws= pane=", "d10 starting ws= pane="},
		},
		{
			name:        "a damaged line between the events is counted and skipped",
			lines:       []string{att("d1", tok), `{"ts":"t","id":"d1","event":"pane_cre`, directorPaneLine("d1", "p1")},
			wantCurrent: "d1 starting ws= pane=p1",
			wantOpen:    []string{"d1 starting ws= pane=p1"},
			wantCorrupt: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := newTestDirectorStore(t)
			writeDirectorFile(t, s, strings.Join(tc.lines, "\n")+"\n")
			res := mustReadDirector(t, s)
			current := ""
			if res.Current != nil {
				current = directorRegSummary(*res.Current)
			}
			if current != tc.wantCurrent {
				t.Errorf("current = %q, want %q", current, tc.wantCurrent)
			}
			if got := openHeadlessSummary(res); !reflect.DeepEqual(got, tc.wantOpen) {
				t.Errorf("OpenHeadless = %q, want %q", got, tc.wantOpen)
			}
			if res.IgnoredEvents != tc.wantIgnored || res.CorruptLines != tc.wantCorrupt {
				t.Errorf("ignored %d, corrupt %d; want %d and %d", res.IgnoredEvents, res.CorruptLines, tc.wantIgnored, tc.wantCorrupt)
			}
		})
	}
}

// Attach and AttachHeadless refuse on each other: any current registration,
// a headless one starting included, refuses both without a replace. The
// message names the registration, and a headless one by its state.
func TestDirectorAttachHeadless_RefusesWhileARegistrationIsCurrent(t *testing.T) {
	const unknownLaunch = `{"ts":"2026-10-15T00:00:00Z","id":"d1","event":"attached","driver":"codex","wake":["pull"],"launch":"terminal","user":"u","cwd":"/w","herdr_pane":false}`
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, s *DirectorStore)
		named string
	}{
		{"a promoted session", func(t *testing.T, s *DirectorStore) {
			mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude})
		}, "d1 (driver claude, attached " + directorTestNow + ")"},
		{"a headless director starting", func(t *testing.T, s *DirectorStore) {
			mustAttachHeadless(t, s)
		}, "d1 (driver codex, attached " + directorTestNow + ", headless starting)"},
		{"a headless director running", func(t *testing.T, s *DirectorStore) {
			mustAttachHeadless(t, s)
			if still, err := s.MarkLaunched("d1", testDirectorWho); err != nil || !still {
				t.Fatalf("MarkLaunched = %v, %v", still, err)
			}
		}, "d1 (driver codex, attached " + directorTestNow + ", headless running)"},
		{"a registration with a launch this binary does not know", func(t *testing.T, s *DirectorStore) {
			writeDirectorFile(t, s, unknownLaunch+"\n")
		}, "d1 (driver codex, attached 2026-10-15T00:00:00Z, launch terminal)"},
		{"a launch cmd starting", func(t *testing.T, s *DirectorStore) {
			mustAttachCmdHeadless(t, s)
		}, "d1 (driver codex, attached " + directorTestNow + ", headless starting, launch cmd)"},
		{"a launch cmd checked in", func(t *testing.T, s *DirectorStore) {
			mustAttachCmdHeadless(t, s)
			if err := s.MarkCmdLaunched(testCheckIn(4242)); err != nil {
				t.Fatalf("MarkCmdLaunched: %v", err)
			}
		}, "d1 (driver codex, attached " + directorTestNow + ", headless running, launch cmd, pid 4242)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := newTestHeadlessStore(t, directorTestToken, "aaaaaaaaaaaa")
			tc.setup(t, s)
			before := readDirectorFile(t, s)

			reg, err := s.AttachHeadless(testHeadlessAttach())
			if !errors.Is(err, ErrDirectorAttached) || err.Error() != "org: director: start: a director is already attached: "+tc.named {
				t.Errorf("AttachHeadless err = %v, want ErrDirectorAttached naming %s", err, tc.named)
			}
			if reg.ID != "" {
				t.Errorf("refused AttachHeadless returned %+v", reg)
			}
			_, _, err = s.Attach(DirectorAttach{Driver: DirectorDriverClaude})
			if !errors.Is(err, ErrDirectorAttached) || err.Error() != "org: director: attach: a director is already attached: "+tc.named {
				t.Errorf("Attach err = %v, want ErrDirectorAttached naming %s", err, tc.named)
			}
			if got := readDirectorFile(t, s); got != before {
				t.Fatalf("director.jsonl changed to %q", got)
			}
		})
	}
}

// A headless registration ended by a detach (or replaced) no longer blocks a
// new start, and stays open next to the new one until it is stopped.
func TestDirectorAttachHeadless_AfterADetachTheOldOneStaysOpen(t *testing.T) {
	s, _, _ := newTestHeadlessStore(t, directorTestToken, "bbbbbbbbbbbb")
	mustAttachHeadless(t, s)
	if _, err := s.Detach(DirectorDetach{ID: "d1"}); err != nil {
		t.Fatalf("Detach d1: %v", err)
	}
	reg := mustAttachHeadless(t, s)
	if reg.ID != "d2" || reg.Token != "bbbbbbbbbbbb" {
		t.Fatalf("second AttachHeadless = %+v, want d2 with the second token", reg)
	}
	res := mustReadDirector(t, s)
	if currentDirectorID(res) != "d2" || !reflect.DeepEqual(openHeadlessSummary(res), []string{"d1 starting ws= pane=", "d2 starting ws= pane="}) {
		t.Fatalf("Read = current %+v, open %q; want d2 current and d1, d2 open", res.Current, openHeadlessSummary(res))
	}
}

// Every launch gets a token of its own from crypto/rand, and the names built
// from it fit herdr's agent-name rule and cannot be an org's or a seat's.
func TestDirectorAttachHeadless_TokensAndTheNamesBuiltFromThem(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	first := mustAttachHeadless(t, s)
	if _, err := s.Detach(DirectorDetach{ID: first.ID}); err != nil {
		t.Fatalf("Detach: %v", err)
	}
	second := mustAttachHeadless(t, s)
	if first.Token == second.Token {
		t.Fatalf("two launches share the token %q", first.Token)
	}
	herdrAgentNameRule := regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)
	for _, token := range []string{first.Token, second.Token} {
		if !regexp.MustCompile(`^[0-9a-f]{12}$`).MatchString(token) {
			t.Errorf("token %q is not 12 lowercase hex digits", token)
		}
		agent, tab, workspace := DirectorAgentName(token), DirectorTabLabel(token), DirectorWorkspaceLabel(token)
		if agent != "director_"+token || tab != agent || workspace != "ralph_director_"+token {
			t.Errorf("names of %q = agent %q, tab %q, workspace %q", token, agent, tab, workspace)
		}
		if len(agent) != 21 || len(agent) > maxHerdrAgentNameLen || !herdrAgentNameRule.MatchString(agent) {
			t.Errorf("agent name %q (%d characters) does not fit herdr's rule", agent, len(agent))
		}
		// An org's workspace label is its org_id and a seat's tab label its
		// seat_id, and neither may hold "_".
		if identifierPattern.MatchString(tab) || identifierPattern.MatchString(workspace) {
			t.Errorf("label %q or %q could be an org's or a seat's", tab, workspace)
		}
	}
}

func TestDirectorAttachHeadless_RefusesBadInputBeforeReadingAnything(t *testing.T) {
	errToken := errors.New("no randomness")
	for _, tc := range []struct {
		name   string
		mutate func(h *DirectorHeadlessAttach)
		token  func() (string, error)
		want   string
	}{
		{name: "an unknown launch", mutate: func(h *DirectorHeadlessAttach) { h.Launch = "terminal" }, want: `start: unknown launch "terminal": use herdr or cmd`},
		{name: "no driver", mutate: func(h *DirectorHeadlessAttach) { h.Driver = "" }, want: `start: unknown driver "": use claude or codex`},
		{name: "an unknown driver", mutate: func(h *DirectorHeadlessAttach) { h.Driver = "gemini" }, want: `start: unknown driver "gemini"`},
		{name: "no model", mutate: func(h *DirectorHeadlessAttach) { h.Model = "" }, want: "start: the model is empty"},
		{name: "no permission mode", mutate: func(h *DirectorHeadlessAttach) { h.PermissionMode = "" }, want: `start: unknown permission mode "": use autonomous, edits, or guarded`},
		{name: "an unknown permission mode", mutate: func(h *DirectorHeadlessAttach) { h.PermissionMode = "bypass" }, want: `unknown permission mode "bypass"`},
		// Plan 2026-10-18-org-director-wake, AC3: start always passes the
		// wake it read, so an empty one is a caller's bug.
		{name: "no wake", mutate: func(h *DirectorHeadlessAttach) { h.Wake = nil }, want: "start: [org.director].wake: the array must not be empty"},
		{name: "an unknown wake", mutate: func(h *DirectorHeadlessAttach) { h.Wake = []string{"poll"} }, want: `start: [org.director].wake: element 0 "poll" is not a wake value`},
		{name: "a wake listed twice", mutate: func(h *DirectorHeadlessAttach) { h.Wake = []string{"push", "push"} }, want: `element 1 "push" repeats element 0`},
		{name: "no token", token: func() (string, error) { return "", errToken }, want: "start: no randomness"},
		{name: "a malformed token", token: func() (string, error) { return "0123456789AB", nil }, want: `launch token "0123456789AB" is not 12 lowercase hex digits`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := filepath.Join(t.TempDir(), "absent", "org")
			s := NewDirectorStore(state)
			s.token = tc.token
			h := testHeadlessAttach()
			if tc.mutate != nil {
				tc.mutate(&h)
			}
			reg, err := s.AttachHeadless(h)
			if err == nil || !strings.HasPrefix(err.Error(), "org: director: start: ") || !strings.Contains(err.Error(), tc.want) || reg.ID != "" {
				t.Fatalf("AttachHeadless(%+v) = %+v, %v; want an error with %q", h, reg, err, tc.want)
			}
			if _, err := os.Stat(state); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("AttachHeadless with bad input created the state dir (stat err = %v)", err)
			}
		})
	}
}

// MarkLaunched writes launched only while the registration is still the
// current one and starting (AC4): after a human's attach --replace or
// detach --force it writes nothing and returns false, so start closes its
// own pane.
func TestDirectorMarkLaunched_OnlyWhileTheRegistrationIsStillCurrent(t *testing.T) {
	for _, tc := range []struct {
		name      string
		setup     func(t *testing.T, s *DirectorStore)
		id        string
		wantStill bool
	}{
		{"after an attach --replace", func(t *testing.T, s *DirectorStore) {
			mustAttachHeadless(t, s)
			mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude, Replace: true})
		}, "d1", false},
		{"after a detach --force", func(t *testing.T, s *DirectorStore) {
			mustAttachHeadless(t, s)
			if _, err := s.Detach(DirectorDetach{Force: true}); err != nil {
				t.Fatalf("force Detach: %v", err)
			}
		}, "d1", false},
		{"a promoted registration", func(t *testing.T, s *DirectorStore) {
			mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude})
		}, "d1", false},
		{"a current registration with a stopped and no detached", func(t *testing.T, s *DirectorStore) {
			writeDirectorFile(t, s, directorHeadlessAttachedLine("d1", directorTestToken)+"\n"+directorEventLine("d1", DirectorEventStopped, "")+"\n")
		}, "d1", false},
		{"an id no registration has", func(t *testing.T, s *DirectorStore) {
			mustAttachHeadless(t, s)
		}, "d2", false},
		{"already running", func(t *testing.T, s *DirectorStore) {
			mustAttachHeadless(t, s)
			if still, err := s.MarkLaunched("d1", testDirectorWho); err != nil || !still {
				t.Fatalf("first MarkLaunched = %v, %v", still, err)
			}
		}, "d1", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := newTestHeadlessStore(t)
			tc.setup(t, s)
			before := readDirectorFile(t, s)
			still, err := s.MarkLaunched(tc.id, testDirectorWho)
			if err != nil || still != tc.wantStill {
				t.Fatalf("MarkLaunched(%s) = %v, %v; want %v, nil", tc.id, still, err, tc.wantStill)
			}
			if got := readDirectorFile(t, s); got != before {
				t.Fatalf("MarkLaunched wrote %q", strings.TrimPrefix(got, before))
			}
		})
	}
	t.Run("a missing state dir", func(t *testing.T) {
		state := filepath.Join(t.TempDir(), "absent", "org")
		if still, err := NewDirectorStore(state).MarkLaunched("d1", testDirectorWho); err != nil || still {
			t.Fatalf("MarkLaunched = %v, %v; want false, nil", still, err)
		}
		if _, err := os.Stat(state); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("MarkLaunched created the state dir (stat err = %v)", err)
		}
	})
	t.Run("an id not of the form d<N>", func(t *testing.T) {
		s, _, _ := newTestHeadlessStore(t)
		if still, err := s.MarkLaunched("d0", testDirectorWho); err == nil || still || !strings.Contains(err.Error(), `"d0" is not a registration id`) {
			t.Fatalf("MarkLaunched(d0) = %v, %v; want the id error", still, err)
		}
	})
}

// MarkStopped of the current registration writes its detached and its
// stopped as two lines of one write (AC8), detached first.
func TestDirectorMarkStopped_TheCurrentRegistrationGetsDetachedAndStoppedInOneWrite(t *testing.T) {
	s, _, clock := newTestHeadlessStore(t, directorTestToken, "cccccccccccc")
	mustAttachHeadless(t, s)
	if err := s.RecordPane("d1", "p1", testDirectorWho); err != nil {
		t.Fatalf("RecordPane: %v", err)
	}
	if still, err := s.MarkLaunched("d1", testDirectorWho); err != nil || !still {
		t.Fatalf("MarkLaunched = %v, %v", still, err)
	}
	before := readDirectorFile(t, s)
	*clock = clock.Add(time.Minute)
	var writes []string
	s.write = func(f *os.File, b []byte) (int, error) {
		writes = append(writes, string(b))
		return f.Write(b)
	}

	wasCurrent, err := s.MarkStopped("d1", testDirectorWho)
	if err != nil || !wasCurrent {
		t.Fatalf("MarkStopped = %v, %v; want true, nil", wasCurrent, err)
	}
	want := `{"ts":"2026-10-15T09:01:00Z","id":"d1","event":"detached","user":"bob","cwd":"/work/b","herdr_pane":false}` + "\n" +
		`{"ts":"2026-10-15T09:01:00Z","id":"d1","event":"stopped","user":"bob","cwd":"/work/b","herdr_pane":false}` + "\n"
	if len(writes) != 1 || writes[0] != want {
		t.Fatalf("writes = %q, want one write of %q", writes, want)
	}
	if got := readDirectorFile(t, s); got != before+want {
		t.Fatalf("director.jsonl = %q, want %q", got, before+want)
	}
	res := mustReadDirector(t, s)
	if res.Current != nil || res.OpenHeadless != nil || res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Fatalf("Read = %+v (current %+v), want no registration, nothing open, nothing skipped", res, res.Current)
	}
	// The stopped id is not handed out again.
	s.write = nil
	if reg := mustAttachHeadless(t, s); reg.ID != "d2" {
		t.Fatalf("AttachHeadless after the stop = %s, want d2", reg.ID)
	}
}

// A headless registration that a replace ended still takes its pane and its
// stop; the stop writes stopped only and leaves the new registration current.
func TestDirectorMarkStopped_ARegistrationNoLongerCurrentGetsStoppedOnly(t *testing.T) {
	s, _, clock := newTestHeadlessStore(t)
	mustAttachHeadless(t, s)
	if err := s.RecordWorkspace("d1", "w1", testDirectorWho); err != nil {
		t.Fatalf("RecordWorkspace: %v", err)
	}
	mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude, Replace: true, User: "carol"})
	if err := s.RecordPane("d1", "p1", testDirectorWho); err != nil {
		t.Fatalf("RecordPane of the replaced registration: %v", err)
	}
	if res := mustReadDirector(t, s); currentDirectorID(res) != "d2" || !reflect.DeepEqual(openHeadlessSummary(res), []string{"d1 starting ws=w1 pane=p1"}) {
		t.Fatalf("Read = current %+v, open %q; want d2 current and d1 open with w1 and p1", res.Current, openHeadlessSummary(res))
	}
	before := readDirectorFile(t, s)
	*clock = clock.Add(time.Minute)

	wasCurrent, err := s.MarkStopped("d1", testDirectorWho)
	if err != nil || wasCurrent {
		t.Fatalf("MarkStopped = %v, %v; want false, nil", wasCurrent, err)
	}
	want := `{"ts":"2026-10-15T09:01:00Z","id":"d1","event":"stopped","user":"bob","cwd":"/work/b","herdr_pane":false}` + "\n"
	if got := readDirectorFile(t, s); got != before+want {
		t.Fatalf("director.jsonl = %q, want %q", got, before+want)
	}
	res := mustReadDirector(t, s)
	if currentDirectorID(res) != "d2" || res.Current.User != "carol" || res.OpenHeadless != nil || res.IgnoredEvents != 0 {
		t.Fatalf("Read = %+v (current %+v), want carol's d2 current and nothing open", res, res.Current)
	}
}

// MarkForceStopped (plan D11) writes what MarkStopped writes with force on
// the stopped line only: the detached written first for the current
// registration names its id, so it carries no force. The fold reads the
// forced stopped as any stopped, and a second stop is refused as already
// stopped.
func TestDirectorMarkForceStopped_OnlyTheStoppedLineCarriesForce(t *testing.T) {
	const stoppedForce = `{"ts":"2026-10-15T09:01:00Z","id":"d1","event":"stopped","user":"bob","cwd":"/work/b","herdr_pane":false,"force":true}` + "\n"
	t.Run("current", func(t *testing.T) {
		s, _, clock := newTestHeadlessStore(t)
		mustAttachHeadless(t, s)
		if err := s.RecordPane("d1", "p1", testDirectorWho); err != nil {
			t.Fatalf("RecordPane: %v", err)
		}
		before := readDirectorFile(t, s)
		*clock = clock.Add(time.Minute)
		var writes []string
		s.write = func(f *os.File, b []byte) (int, error) {
			writes = append(writes, string(b))
			return f.Write(b)
		}

		wasCurrent, err := s.MarkForceStopped("d1", testDirectorWho)
		if err != nil || !wasCurrent {
			t.Fatalf("MarkForceStopped = %v, %v; want true, nil", wasCurrent, err)
		}
		want := `{"ts":"2026-10-15T09:01:00Z","id":"d1","event":"detached","user":"bob","cwd":"/work/b","herdr_pane":false}` + "\n" + stoppedForce
		if len(writes) != 1 || writes[0] != want {
			t.Fatalf("writes = %q, want one write of %q", writes, want)
		}
		if got := readDirectorFile(t, s); got != before+want {
			t.Fatalf("director.jsonl = %q, want %q", got, before+want)
		}
		res := mustReadDirector(t, s)
		if res.Current != nil || res.OpenHeadless != nil || res.CorruptLines != 0 || res.IgnoredEvents != 0 {
			t.Fatalf("Read = %+v (current %+v), want no registration, nothing open, nothing skipped", res, res.Current)
		}
		if _, err := s.MarkForceStopped("d1", testDirectorWho); !errors.Is(err, ErrDirectorNoOpenHeadless) || !errors.Is(err, ErrDirectorStopped) {
			t.Fatalf("second MarkForceStopped err = %v, want ErrDirectorNoOpenHeadless with ErrDirectorStopped", err)
		}
	})
	t.Run("no longer current", func(t *testing.T) {
		s, _, clock := newTestHeadlessStore(t)
		mustAttachHeadless(t, s)
		mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude, Replace: true, User: "carol"})
		before := readDirectorFile(t, s)
		*clock = clock.Add(time.Minute)

		wasCurrent, err := s.MarkForceStopped("d1", testDirectorWho)
		if err != nil || wasCurrent {
			t.Fatalf("MarkForceStopped = %v, %v; want false, nil", wasCurrent, err)
		}
		if got := readDirectorFile(t, s); got != before+stoppedForce {
			t.Fatalf("director.jsonl = %q, want %q", got, before+stoppedForce)
		}
		res := mustReadDirector(t, s)
		if currentDirectorID(res) != "d2" || res.OpenHeadless != nil || res.IgnoredEvents != 0 {
			t.Fatalf("Read = %+v (current %+v), want d2 current and nothing open", res, res.Current)
		}
	})
	t.Run("not an id", func(t *testing.T) {
		s, _, _ := newTestHeadlessStore(t)
		if _, err := s.MarkForceStopped("e1", testDirectorWho); err == nil || !strings.Contains(err.Error(), "not a registration id") {
			t.Fatalf("MarkForceStopped(e1) err = %v, want the id error", err)
		}
		if _, err := os.Stat(s.Path()); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("MarkForceStopped(e1) wrote director.jsonl (stat err %v)", err)
		}
	})
}

// recordForTest records, for d1, each step in order as start records it:
// "workspace" (w1), "pane" (p1), or "launched".
func recordForTest(t *testing.T, s *DirectorStore, steps ...string) {
	t.Helper()
	for _, step := range steps {
		var err error
		switch step {
		case "workspace":
			err = s.RecordWorkspace("d1", "w1", testDirectorWho)
		case "pane":
			err = s.RecordPane("d1", "p1", testDirectorWho)
		case "launched":
			var still bool
			if still, err = s.MarkLaunched("d1", testDirectorWho); err == nil && !still {
				err = errors.New("d1 is no longer the current registration")
			}
		default:
			t.Fatalf("unknown step %q", step)
		}
		if err != nil {
			t.Fatalf("record %s: %v", step, err)
		}
	}
}

// MarkStoppedIfUnchanged (plan D14) refuses, writing nothing, when the
// workspace or the pane recorded now is not the one in the registration as
// stop read it: a start still running recorded it after the read, so stop did
// not close it. Either id alone refuses, and the registration stays current
// and open. With the ids stop read it writes what MarkStopped writes, also
// when launched came after the read (launched changes nothing stop closes);
// after a refusal, a stop that reads again is recorded.
func TestDirectorMarkStoppedIfUnchanged_RefusesAnIDRecordedAfterTheRead(t *testing.T) {
	const refusal = "org: director: stop d1: a start in progress recorded a workspace or pane after stop read the registration: "
	cases := []struct {
		name              string
		atRead, afterRead []string
		wantErr           string // "" when the stop is recorded
		wantOpen          string // the open registration after a refusal
	}{
		{"nothing at the read, then the workspace, the pane, and launched", nil, []string{"workspace", "pane", "launched"},
			refusal + `d1 had no workspace and no pane when stop read it, and has workspace "w1" and pane "p1" now`, "d1 running ws=w1 pane=p1"},
		{"the workspace at the read, then the pane and launched", []string{"workspace"}, []string{"pane", "launched"},
			refusal + `d1 had workspace "w1" and no pane when stop read it, and has workspace "w1" and pane "p1" now`, "d1 running ws=w1 pane=p1"},
		{"the pane at the read, then the workspace", []string{"pane"}, []string{"workspace"},
			refusal + `d1 had no workspace and pane "p1" when stop read it, and has workspace "w1" and pane "p1" now`, "d1 starting ws=w1 pane=p1"},
		{"both at the read, then launched", []string{"workspace", "pane"}, []string{"launched"}, "", ""},
		{"nothing at the read and nothing after", nil, nil, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := newTestHeadlessStore(t)
			mustAttachHeadless(t, s)
			recordForTest(t, s, tc.atRead...)
			read := *mustReadDirector(t, s).Current
			recordForTest(t, s, tc.afterRead...)
			trail := directorTrail(t, s)
			before := readDirectorFile(t, s)

			wasCurrent, err := s.MarkStoppedIfUnchanged(read, testDirectorWho)
			if tc.wantErr == "" {
				if err != nil || !wasCurrent {
					t.Fatalf("MarkStoppedIfUnchanged = %v, %v; want true, nil", wasCurrent, err)
				}
				wantTrail(t, s, append(trail, "detached d1", "stopped d1")...)
				if res := mustReadDirector(t, s); res.Current != nil || res.OpenHeadless != nil {
					t.Fatalf("Read = current %+v, open %q; want nothing current or open", res.Current, openHeadlessSummary(res))
				}
				return
			}
			if !errors.Is(err, ErrDirectorChangedDuringStop) || errors.Is(err, ErrDirectorNoOpenHeadless) || err.Error() != tc.wantErr || wasCurrent {
				t.Fatalf("MarkStoppedIfUnchanged = %v, %v;\nwant false and %q", wasCurrent, err, tc.wantErr)
			}
			if got := readDirectorFile(t, s); got != before {
				t.Fatalf("director.jsonl changed to %q", got)
			}
			res := mustReadDirector(t, s)
			if currentDirectorID(res) != "d1" || !reflect.DeepEqual(openHeadlessSummary(res), []string{tc.wantOpen}) {
				t.Fatalf("Read = current %+v, open %q; want d1 current and %q open", res.Current, openHeadlessSummary(res), tc.wantOpen)
			}

			// A stop that reads again finds the ids recorded now and is recorded.
			if wasCurrent, err := s.MarkStoppedIfUnchanged(*res.Current, testDirectorWho); err != nil || !wasCurrent {
				t.Fatalf("MarkStoppedIfUnchanged after a new read = %v, %v; want true, nil", wasCurrent, err)
			}
			wantTrail(t, s, append(trail, "detached d1", "stopped d1")...)
		})
	}
}

func TestDirectorMarkStopped_RefusesWithNoOpenHeadlessRegistration(t *testing.T) {
	for _, tc := range []struct {
		name        string
		setup       func(t *testing.T, s *DirectorStore)
		id          string
		want        string
		wantStopped bool
	}{
		{"an id no registration has", func(t *testing.T, s *DirectorStore) { mustAttachHeadless(t, s) },
			"d5", "org: director: stop d5: no open headless registration has the id: no registration d5 is recorded", false},
		{"a promoted registration", func(t *testing.T, s *DirectorStore) {
			mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude})
		}, "d1", "org: director: stop d1: no open headless registration has the id: d1 (driver claude, attached " + directorTestNow +
			") was not started in herdr by ralph org director start", false},
		{"a registration with a launch this binary does not know", func(t *testing.T, s *DirectorStore) {
			writeDirectorFile(t, s, `{"ts":"2026-10-15T00:00:00Z","id":"d1","event":"attached","driver":"codex","wake":["pull"],"launch":"terminal","user":"u","cwd":"/w","herdr_pane":false}`+"\n")
		}, "d1", "was not started in herdr by ralph org director start", false},
		{"an already stopped registration", func(t *testing.T, s *DirectorStore) {
			mustAttachHeadless(t, s)
			if _, err := s.MarkStopped("d1", testDirectorWho); err != nil {
				t.Fatalf("first MarkStopped: %v", err)
			}
		}, "d1", "org: director: stop d1: no open headless registration has the id: the headless registration was already stopped: d1 (driver codex, attached " +
			directorTestNow + ", headless stopped)", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := newTestHeadlessStore(t)
			tc.setup(t, s)
			before := readDirectorFile(t, s)
			wasCurrent, err := s.MarkStopped(tc.id, testDirectorWho)
			if !errors.Is(err, ErrDirectorNoOpenHeadless) || errors.Is(err, ErrDirectorStopped) != tc.wantStopped || !strings.Contains(err.Error(), tc.want) || wasCurrent {
				t.Fatalf("MarkStopped(%s) = %v, %v; want ErrDirectorNoOpenHeadless (stopped %v) with %q", tc.id, wasCurrent, err, tc.wantStopped, tc.want)
			}
			if got := readDirectorFile(t, s); got != before {
				t.Fatalf("director.jsonl changed to %q", got)
			}
		})
	}
	t.Run("a missing state dir", func(t *testing.T) {
		state := filepath.Join(t.TempDir(), "absent", "org")
		if _, err := NewDirectorStore(state).MarkStopped("d1", testDirectorWho); !errors.Is(err, ErrDirectorNoOpenHeadless) {
			t.Fatalf("MarkStopped err = %v, want ErrDirectorNoOpenHeadless", err)
		}
		if _, err := os.Stat(state); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("MarkStopped created the state dir (stat err = %v)", err)
		}
	})
	t.Run("an id not of the form d<N>", func(t *testing.T) {
		s, _, _ := newTestHeadlessStore(t)
		_, err := s.MarkStopped("e1", testDirectorWho)
		if err == nil || errors.Is(err, ErrDirectorNoOpenHeadless) || !strings.Contains(err.Error(), `"e1" is not a registration id`) {
			t.Fatalf("MarkStopped(e1) err = %v, want the id error", err)
		}
	})
}

// RecordWorkspace and RecordPane refuse, writing nothing, when no open
// headless registration has the id or when the id is already recorded.
func TestDirectorRecordHerdrIDs_Refusals(t *testing.T) {
	record := map[string]func(s *DirectorStore, id, value string) error{
		"workspace": func(s *DirectorStore, id, value string) error { return s.RecordWorkspace(id, value, testDirectorWho) },
		"pane":      func(s *DirectorStore, id, value string) error { return s.RecordPane(id, value, testDirectorWho) },
	}
	for _, what := range []string{"workspace", "pane"} {
		for _, tc := range []struct {
			name        string
			setup       func(t *testing.T, s *DirectorStore)
			id, value   string
			want        string
			wantOpen    bool // the error wraps ErrDirectorNoOpenHeadless
			wantStopped bool
		}{
			{"an id no registration has", func(t *testing.T, s *DirectorStore) { mustAttachHeadless(t, s) },
				"d4", "x1", "no registration d4 is recorded", true, false},
			{"a promoted registration", func(t *testing.T, s *DirectorStore) {
				mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude})
			}, "d1", "x1", "was not started in herdr", true, false},
			{"a stopped registration", func(t *testing.T, s *DirectorStore) {
				mustAttachHeadless(t, s)
				if _, err := s.MarkStopped("d1", testDirectorWho); err != nil {
					t.Fatalf("MarkStopped: %v", err)
				}
			}, "d1", "x1", "already stopped", true, true},
			{"an id already recorded", func(t *testing.T, s *DirectorStore) {
				mustAttachHeadless(t, s)
				if err := record[what](s, "d1", "x1"); err != nil {
					t.Fatalf("first record: %v", err)
				}
			}, "d1", "x2", "record the " + what + " of d1: d1 already has " + what + " x1 recorded", false, false},
			{"an empty value", func(t *testing.T, s *DirectorStore) { mustAttachHeadless(t, s) },
				"d1", "", "record the " + what + " of d1: the " + what + " id is empty", false, false},
			{"an id not of the form d<N>", func(t *testing.T, s *DirectorStore) { mustAttachHeadless(t, s) },
				"d1x", "x1", `record the ` + what + `: "d1x" is not a registration id`, false, false},
		} {
			t.Run(what+"/"+tc.name, func(t *testing.T) {
				s, _, _ := newTestHeadlessStore(t)
				tc.setup(t, s)
				before := readDirectorFile(t, s)
				err := record[what](s, tc.id, tc.value)
				if err == nil || !strings.HasPrefix(err.Error(), "org: director: ") || !strings.Contains(err.Error(), tc.want) ||
					errors.Is(err, ErrDirectorNoOpenHeadless) != tc.wantOpen || errors.Is(err, ErrDirectorStopped) != tc.wantStopped {
					t.Fatalf("record %s(%s, %q) err = %v; want %q (no open headless %v, stopped %v)", what, tc.id, tc.value, err, tc.want, tc.wantOpen, tc.wantStopped)
				}
				if got := readDirectorFile(t, s); got != before {
					t.Fatalf("director.jsonl changed to %q", got)
				}
			})
		}
		t.Run(what+"/a missing state dir", func(t *testing.T) {
			state := filepath.Join(t.TempDir(), "absent", "org")
			if err := record[what](NewDirectorStore(state), "d1", "x1"); !errors.Is(err, ErrDirectorNoOpenHeadless) {
				t.Fatalf("record err = %v, want ErrDirectorNoOpenHeadless", err)
			}
			if _, err := os.Stat(state); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("record created the state dir (stat err = %v)", err)
			}
		})
	}
}

// Every new writer goes through appendLocked: a write that a full disk cut
// off is cut back, the file and its fold are as they were, and the error
// says nothing was recorded (naming the current registration when the write
// would have ended it).
func TestDirectorHeadlessWriters_ACutOffWriteIsRestored(t *testing.T) {
	headless := func(t *testing.T, s *DirectorStore) { mustAttachHeadless(t, s) }
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, s *DirectorStore)
		call  func(s *DirectorStore) error
		want  string // the error up to appendLocked's
	}{
		{"AttachHeadless into a missing file", func(*testing.T, *DirectorStore) {},
			func(s *DirectorStore) error { _, err := s.AttachHeadless(testHeadlessAttach()); return err },
			"org: director: start: nothing was recorded: "},
		{"AttachHeadless after an ended registration", func(t *testing.T, s *DirectorStore) {
			writeDirectorFile(t, s, directorAttachedLine("d1", "claude")+"\n"+directorDetachedLine("d1")+"\n")
		}, func(s *DirectorStore) error { _, err := s.AttachHeadless(testHeadlessAttach()); return err },
			"org: director: start: nothing was recorded: "},
		{"RecordWorkspace", headless,
			func(s *DirectorStore) error { return s.RecordWorkspace("d1", "w1", testDirectorWho) },
			"org: director: record the workspace of d1: nothing was recorded: "},
		{"RecordPane", headless,
			func(s *DirectorStore) error { return s.RecordPane("d1", "p1", testDirectorWho) },
			"org: director: record the pane of d1: nothing was recorded: "},
		{"MarkLaunched", headless,
			func(s *DirectorStore) error {
				still, err := s.MarkLaunched("d1", testDirectorWho)
				if still {
					return errors.New("MarkLaunched returned true for a write that failed")
				}
				return err
			},
			"org: director: mark d1 launched: nothing was recorded: "},
		{"MarkStopped of the current registration", headless,
			func(s *DirectorStore) error { _, err := s.MarkStopped("d1", testDirectorWho); return err },
			"org: director: stop d1: nothing was recorded, so d1 (driver codex, attached " + directorTestNow + ", headless starting) is still the current registration: "},
		{"MarkStopped of a detached registration", func(t *testing.T, s *DirectorStore) {
			mustAttachHeadless(t, s)
			if _, err := s.Detach(DirectorDetach{ID: "d1"}); err != nil {
				t.Fatalf("Detach: %v", err)
			}
		}, func(s *DirectorStore) error { _, err := s.MarkStopped("d1", testDirectorWho); return err },
			"org: director: stop d1: nothing was recorded: "},
		{"MarkStoppedIfUnchanged of the current registration", headless,
			func(s *DirectorStore) error {
				read, err := s.openHeadlessRegistration("stop d1", "d1")
				if err != nil {
					return err
				}
				_, err = s.MarkStoppedIfUnchanged(read, testDirectorWho)
				return err
			},
			"org: director: stop d1: nothing was recorded, so d1 (driver codex, attached " + directorTestNow + ", headless starting) is still the current registration: "},
		{"MarkForceStopped of the current registration", headless,
			func(s *DirectorStore) error { _, err := s.MarkForceStopped("d1", testDirectorWho); return err },
			"org: director: stop d1: nothing was recorded, so d1 (driver codex, attached " + directorTestNow + ", headless starting) is still the current registration: "},
		{"MarkCmdLaunched", func(t *testing.T, s *DirectorStore) { mustAttachCmdHeadless(t, s) },
			func(s *DirectorStore) error { return s.MarkCmdLaunched(testCheckIn(4242)) },
			"org: director: check in d1: nothing was recorded: "},
		{"MarkStopped of a launch cmd", func(t *testing.T, s *DirectorStore) { mustAttachCmdHeadless(t, s) },
			func(s *DirectorStore) error { _, err := s.MarkStopped("d1", testDirectorWho); return err },
			"org: director: stop d1: nothing was recorded, so d1 (driver codex, attached " + directorTestNow + ", headless starting, launch cmd) is still the current registration: "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, clock := newTestHeadlessStore(t, directorTestToken, "dddddddddddd")
			tc.setup(t, s)
			var before string
			if raw, err := os.ReadFile(s.Path()); err == nil {
				before = string(raw)
			}
			beforeRead := mustReadDirector(t, s)
			*clock = clock.Add(time.Minute)
			var onDisk string
			s.write = directorPartialWrite(t, s, &onDisk)

			err := tc.call(s)
			want := tc.want + "append to " + s.Path() + ": no space left on device"
			if err == nil || err.Error() != want || !errors.Is(err, syscall.ENOSPC) || errors.Is(err, ErrDirectorWriteNotRestored) {
				t.Fatalf("err = %v, want %q", err, want)
			}
			if cut, ok := strings.CutPrefix(onDisk, before); !ok || cut == "" || strings.HasSuffix(cut, "\n") {
				t.Fatalf("the file after the partial write = %q, want the old text and part of a line", onDisk)
			}
			if got := readDirectorFile(t, s); got != before {
				t.Fatalf("director.jsonl = %q, want it cut back to %q", got, before)
			}
			if after := mustReadDirector(t, s); !reflect.DeepEqual(after, beforeRead) {
				t.Fatalf("Read after the failure = %+v, want %+v", after, beforeRead)
			}
		})
	}
}

// The order of MarkStopped's two lines: when the write stops after the
// detached and its cut back fails too, the file holds the detached alone.
// That leaves nothing current and the registration open, so the next stop
// finds it and records its stopped; with the stopped first, the registration
// would stay current and refuse every later stop as already stopped.
func TestDirectorMarkStopped_DetachedFirstSoAWriteThatCannotBeCutBackStaysRecoverable(t *testing.T) {
	s, _, clock := newTestHeadlessStore(t)
	mustAttachHeadless(t, s)
	if err := s.RecordPane("d1", "p1", testDirectorWho); err != nil {
		t.Fatalf("RecordPane: %v", err)
	}
	*clock = clock.Add(time.Minute)
	var onDisk string
	s.write = directorPartialWrite(t, s, &onDisk)
	s.truncate = func(string, int64) error { return errors.New("truncate refused") }

	if _, err := s.MarkStopped("d1", testDirectorWho); !errors.Is(err, ErrDirectorWriteNotRestored) {
		t.Fatalf("MarkStopped err = %v, want ErrDirectorWriteNotRestored", err)
	}
	if !strings.Contains(onDisk, `"id":"d1","event":"detached"`) || strings.Contains(onDisk, `"event":"stopped","user"`) {
		t.Fatalf("the file after the partial write = %q, want the whole detached and part of the stopped", onDisk)
	}
	res := mustReadDirector(t, s)
	if res.Current != nil || !reflect.DeepEqual(openHeadlessSummary(res), []string{"d1 starting ws= pane=p1"}) || res.CorruptLines != 1 {
		t.Fatalf("Read = %+v (current %+v), want nothing current, d1 open, the cut stopped corrupt", res, res.Current)
	}

	s.write, s.truncate = nil, nil
	wasCurrent, err := s.MarkStopped("d1", testDirectorWho)
	if err != nil || wasCurrent {
		t.Fatalf("MarkStopped again = %v, %v; want false, nil", wasCurrent, err)
	}
	if res := mustReadDirector(t, s); res.Current != nil || res.OpenHeadless != nil || res.CorruptLines != 1 || res.IgnoredEvents != 0 {
		t.Fatalf("Read = %+v (current %+v), want d1 stopped and only the cut line skipped", res, res.Current)
	}
}

// Attach and AttachHeadless decide under the same lock: of many at once,
// one registers and the others are refused.
func TestDirectorAttachAndAttachHeadless_ConcurrentlyOnlyOneRegisters(t *testing.T) {
	s, _, _ := newTestDirectorStore(t)
	const n = 10
	errs := make([]error, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if i%2 == 0 {
				_, errs[i] = s.AttachHeadless(testHeadlessAttach())
			} else {
				_, _, errs[i] = s.Attach(DirectorAttach{Driver: DirectorDriverClaude})
			}
		}()
	}
	close(start)
	wg.Wait()
	winners := 0
	for i, err := range errs {
		switch {
		case err == nil:
			winners++
		case !errors.Is(err, ErrDirectorAttached):
			t.Fatalf("call %d err = %v, want ErrDirectorAttached", i, err)
		}
	}
	if winners != 1 {
		t.Fatalf("%d calls registered, want 1", winners)
	}
	if lines := directorFileLines(t, s); len(lines) != 1 {
		t.Fatalf("director.jsonl has %d lines, want 1: %q", len(lines), lines)
	}
}

// The launch cmd registration of stage 8a-2b (plan
// 2026-10-16-org-director-launch-cmd, AC2 and AC3): its attached, its
// check-in (MarkCmdLaunched), and how the fold and the herdr writers read
// it.

// directorTestStartTime is the start time the check-ins of these tests
// record, in ReadDirectorProcess's form.
const directorTestStartTime = "Thu Oct 15 21:51:09 2026"

func testCmdHeadlessAttach() DirectorHeadlessAttach {
	h := testHeadlessAttach()
	h.Launch = DirectorLaunchCmd
	return h
}

func mustAttachCmdHeadless(t *testing.T, s *DirectorStore) DirectorRegistration {
	t.Helper()
	reg, err := s.AttachHeadless(testCmdHeadlessAttach())
	if err != nil {
		t.Fatalf("AttachHeadless(launch cmd): %v", err)
	}
	return reg
}

// testCheckIn is the check-in of d1's launch script with pid and
// directorTestStartTime.
func testCheckIn(pid int) DirectorCmdCheckIn {
	return DirectorCmdCheckIn{ID: "d1", Token: directorTestToken, PID: pid, StartTime: directorTestStartTime, Who: testDirectorWho}
}

// directorCmdAttachedLine is the attached line of a launch cmd registration
// as a test fixture.
func directorCmdAttachedLine(id, token string) string {
	return fmt.Sprintf(`{"ts":"2026-10-15T00:00:00Z","id":%q,"event":"attached","driver":"claude","wake":["pull"],"launch":"cmd","token":%q,"model":"m","permission_mode":"guarded","user":"u","cwd":"/w","herdr_pane":false}`, id, token)
}

// directorLaunchedLine is a launched line for id with pid and start_time,
// each only when set, as a test fixture.
func directorLaunchedLine(id string, pid int, startTime string) string {
	fields := ""
	if pid != 0 {
		fields += fmt.Sprintf(`"pid":%d,`, pid)
	}
	if startTime != "" {
		fields += fmt.Sprintf(`"start_time":%q,`, startTime)
	}
	return directorEventLine(id, DirectorEventLaunched, fields)
}

// directorLaunchSummary names r by its id, launch, state, pid, start time,
// workspace, and pane.
func directorLaunchSummary(r DirectorRegistration) string {
	return fmt.Sprintf("%s %s %s pid=%d start=%q ws=%s pane=%s", r.ID, r.Launch, r.State, r.PID, r.StartTime, r.WorkspaceID, r.PaneID)
}

// AC2's record shape: AttachHeadless with launch cmd writes the attached of
// a launch herdr with launch cmd, and the check-in adds launched with pid
// and start_time; the fold reads them into the registration and its JSON.
// The launch herdr lines of TestDirectorAttachHeadless_RecordsTheLaunchAndItsEvents
// stay as they were.
func TestDirectorAttachHeadless_RecordsALaunchCmdAndItsCheckIn(t *testing.T) {
	s, _, clock := newTestHeadlessStore(t)
	reg := mustAttachCmdHeadless(t, s)
	want := DirectorRegistration{
		ID: "d1", Driver: DirectorDriverCodex, Wake: []string{DirectorWakePull},
		Launch: DirectorLaunchCmd, State: DirectorStateStarting, Token: directorTestToken,
		Model: "gpt-x", PermissionMode: PermissionModeAutonomous,
		AttachedAt: directorTestNow, User: "alice", Cwd: "/work/a",
	}
	if !reflect.DeepEqual(reg, want) {
		t.Fatalf("AttachHeadless = %+v, want %+v", reg, want)
	}
	*clock = clock.Add(time.Second)
	if err := s.MarkCmdLaunched(DirectorCmdCheckIn{ID: "d1", Token: directorTestToken, PID: 4242, StartTime: "  " + directorTestStartTime + "\n", Who: testDirectorWho}); err != nil {
		t.Fatalf("MarkCmdLaunched: %v", err)
	}
	wantLines := []string{
		`{"ts":"2026-10-15T09:00:00Z","id":"d1","event":"attached","driver":"codex","wake":["pull"],"launch":"cmd","token":"0123456789ab","model":"gpt-x","permission_mode":"autonomous","user":"alice","cwd":"/work/a","herdr_pane":false}`,
		`{"ts":"2026-10-15T09:00:01Z","id":"d1","event":"launched","pid":4242,"start_time":"Thu Oct 15 21:51:09 2026","user":"bob","cwd":"/work/b","herdr_pane":false}`,
	}
	if got := directorFileLines(t, s); strings.Join(got, "\n") != strings.Join(wantLines, "\n") {
		t.Fatalf("director.jsonl =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(wantLines, "\n"))
	}
	res := mustReadDirector(t, s)
	want.State, want.PID, want.StartTime = DirectorStateRunning, 4242, directorTestStartTime
	if res.Current == nil || !reflect.DeepEqual(*res.Current, want) || res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Fatalf("Read = %+v (current %+v), want %+v", res, res.Current, want)
	}
	if len(res.OpenHeadless) != 1 || !reflect.DeepEqual(res.OpenHeadless[0], want) {
		t.Fatalf("OpenHeadless = %+v, want only %+v", res.OpenHeadless, want)
	}
	raw, err := json.Marshal(*res.Current)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON := `{"id":"d1","driver":"codex","wake":["pull"],"launch":"cmd","state":"running","token":"0123456789ab","model":"gpt-x","permission_mode":"autonomous",` +
		`"pid":4242,"start_time":"Thu Oct 15 21:51:09 2026","attached_at":"2026-10-15T09:00:00Z","user":"alice","cwd":"/work/a","herdr_pane":false}`
	if string(raw) != wantJSON {
		t.Fatalf("JSON = %s, want %s", raw, wantJSON)
	}
}

// D11: a registration start wrote is headless whether it runs in herdr or in
// a terminal of launch_cmd; only launch herdr has herdr's workspace and pane.
func TestDirectorRegistration_HeadlessHerdrAndCmdLaunches(t *testing.T) {
	for _, tc := range []struct {
		launch                 string
		headless, herdr, isCmd bool
	}{
		{"", false, false, false},
		{DirectorLaunchHerdr, true, true, false},
		{DirectorLaunchCmd, true, false, true},
		{"terminal", false, false, false},
		{"CMD", false, false, false},
	} {
		r := DirectorRegistration{Launch: tc.launch}
		if r.Headless() != tc.headless || r.HerdrLaunch() != tc.herdr || r.CmdLaunch() != tc.isCmd {
			t.Errorf("launch %q: Headless %v, HerdrLaunch %v, CmdLaunch %v; want %v, %v, %v",
				tc.launch, r.Headless(), r.HerdrLaunch(), r.CmdLaunch(), tc.headless, tc.herdr, tc.isCmd)
		}
	}
}

// The fold of a launch cmd: launched applies only with a positive pid and a
// start time, while the registration is current and starting; a launch
// herdr's launched never carries them, so one that does is ignored; herdr's
// workspace_created and pane_created do not apply to a launch cmd.
func TestDirectorRead_FoldsLaunchCmdRegistrations(t *testing.T) {
	att := directorCmdAttachedLine
	tok := directorTestToken
	stopped := directorEventLine("d1", DirectorEventStopped, "")
	const st = directorTestStartTime
	for _, tc := range []struct {
		name        string
		lines       []string
		wantCurrent string // directorLaunchSummary of the current registration, "" for none
		wantOpen    []string
		wantIgnored int
	}{
		{
			name:        "starting",
			lines:       []string{att("d1", tok)},
			wantCurrent: `d1 cmd starting pid=0 start="" ws= pane=`,
			wantOpen:    []string{`d1 cmd starting pid=0 start="" ws= pane=`},
		},
		{
			name:        "checked in",
			lines:       []string{att("d1", tok), directorLaunchedLine("d1", 4242, st)},
			wantCurrent: `d1 cmd running pid=4242 start="` + st + `" ws= pane=`,
			wantOpen:    []string{`d1 cmd running pid=4242 start="` + st + `" ws= pane=`},
		},
		{
			name:        "a launched with no pid is ignored",
			lines:       []string{att("d1", tok), directorLaunchedLine("d1", 0, st)},
			wantCurrent: `d1 cmd starting pid=0 start="" ws= pane=`,
			wantOpen:    []string{`d1 cmd starting pid=0 start="" ws= pane=`},
			wantIgnored: 1,
		},
		{
			name:        "a launched with no start time is ignored",
			lines:       []string{att("d1", tok), directorLaunchedLine("d1", 4242, "")},
			wantCurrent: `d1 cmd starting pid=0 start="" ws= pane=`,
			wantOpen:    []string{`d1 cmd starting pid=0 start="" ws= pane=`},
			wantIgnored: 1,
		},
		{
			name:        "a launched with a negative pid is ignored",
			lines:       []string{att("d1", tok), directorLaunchedLine("d1", -7, st)},
			wantCurrent: `d1 cmd starting pid=0 start="" ws= pane=`,
			wantOpen:    []string{`d1 cmd starting pid=0 start="" ws= pane=`},
			wantIgnored: 1,
		},
		{
			name:        "a second launched keeps the first pid",
			lines:       []string{att("d1", tok), directorLaunchedLine("d1", 4242, st), directorLaunchedLine("d1", 99, "Fri Oct 16 00:00:00 2026")},
			wantCurrent: `d1 cmd running pid=4242 start="` + st + `" ws= pane=`,
			wantOpen:    []string{`d1 cmd running pid=4242 start="` + st + `" ws= pane=`},
			wantIgnored: 1,
		},
		{
			name:        "a launched after a detach is ignored",
			lines:       []string{att("d1", tok), directorDetachedLine("d1"), directorLaunchedLine("d1", 4242, st)},
			wantOpen:    []string{`d1 cmd starting pid=0 start="" ws= pane=`},
			wantIgnored: 1,
		},
		{
			name:        "a launched after the stopped is ignored",
			lines:       []string{att("d1", tok), directorDetachedLine("d1"), stopped, directorLaunchedLine("d1", 4242, st)},
			wantIgnored: 1,
		},
		{
			name:        "herdr's workspace and pane do not apply",
			lines:       []string{att("d1", tok), directorWorkspaceLine("d1", "w1"), directorPaneLine("d1", "p1")},
			wantCurrent: `d1 cmd starting pid=0 start="" ws= pane=`,
			wantOpen:    []string{`d1 cmd starting pid=0 start="" ws= pane=`},
			wantIgnored: 2,
		},
		{
			name:        "a malformed token is ignored",
			lines:       []string{att("d1", "0123456789AB"), att("d2", "")},
			wantIgnored: 2,
		},
		{
			name:  "stopped after its check-in, as MarkStopped writes it",
			lines: []string{att("d1", tok), directorLaunchedLine("d1", 4242, st), directorDetachedLine("d1"), stopped},
		},
		{
			name:        "a launch herdr's launched with a pid is ignored",
			lines:       []string{directorHeadlessAttachedLine("d1", tok), directorLaunchedLine("d1", 4242, st)},
			wantCurrent: `d1 herdr starting pid=0 start="" ws= pane=`,
			wantOpen:    []string{`d1 herdr starting pid=0 start="" ws= pane=`},
			wantIgnored: 1,
		},
		{
			name:        "a launch herdr's launched with a start time only is ignored",
			lines:       []string{directorHeadlessAttachedLine("d1", tok), directorLaunchedLine("d1", 0, st)},
			wantCurrent: `d1 herdr starting pid=0 start="" ws= pane=`,
			wantOpen:    []string{`d1 herdr starting pid=0 start="" ws= pane=`},
			wantIgnored: 1,
		},
		{
			name:        "a launch herdr's launched without them still applies",
			lines:       []string{directorHeadlessAttachedLine("d1", tok), directorLaunchedLine("d1", 0, "")},
			wantCurrent: `d1 herdr running pid=0 start="" ws= pane=`,
			wantOpen:    []string{`d1 herdr running pid=0 start="" ws= pane=`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := newTestDirectorStore(t)
			writeDirectorFile(t, s, strings.Join(tc.lines, "\n")+"\n")
			res := mustReadDirector(t, s)
			current := ""
			if res.Current != nil {
				current = directorLaunchSummary(*res.Current)
			}
			if current != tc.wantCurrent {
				t.Errorf("current = %q, want %q", current, tc.wantCurrent)
			}
			var open []string
			for _, r := range res.OpenHeadless {
				open = append(open, directorLaunchSummary(r))
			}
			if !reflect.DeepEqual(open, tc.wantOpen) {
				t.Errorf("OpenHeadless = %q, want %q", open, tc.wantOpen)
			}
			if res.IgnoredEvents != tc.wantIgnored || res.CorruptLines != 0 {
				t.Errorf("ignored %d, corrupt %d; want %d and 0", res.IgnoredEvents, res.CorruptLines, tc.wantIgnored)
			}
		})
	}
}

// What only a launch herdr has stays herdr's: RecordWorkspace and RecordPane
// refuse a launch cmd with ErrDirectorNotHerdrLaunch, and MarkLaunched
// (start's record of herdr's agent start) returns false and writes nothing
// for it, as for a registration that is not headless.
func TestDirectorHerdrWriters_LeaveALaunchCmdAlone(t *testing.T) {
	for _, checkedIn := range []bool{false, true} {
		t.Run(fmt.Sprintf("checked in %v", checkedIn), func(t *testing.T) {
			s, _, _ := newTestHeadlessStore(t)
			mustAttachCmdHeadless(t, s)
			if checkedIn {
				if err := s.MarkCmdLaunched(testCheckIn(4242)); err != nil {
					t.Fatal(err)
				}
			}
			before := readDirectorFile(t, s)
			for what, record := range map[string]func() error{
				"workspace": func() error { return s.RecordWorkspace("d1", "w1", testDirectorWho) },
				"pane":      func() error { return s.RecordPane("d1", "p1", testDirectorWho) },
			} {
				err := record()
				if !errors.Is(err, ErrDirectorNotHerdrLaunch) || errors.Is(err, ErrDirectorNoOpenHeadless) ||
					!strings.HasPrefix(err.Error(), "org: director: record the "+what+" of d1: ") || !strings.HasSuffix(err.Error(), " has no herdr workspace or pane") {
					t.Errorf("record the %s err = %v, want ErrDirectorNotHerdrLaunch", what, err)
				}
			}
			if still, err := s.MarkLaunched("d1", testDirectorWho); err != nil || still {
				t.Errorf("MarkLaunched = %v, %v; want false, nil", still, err)
			}
			if got := readDirectorFile(t, s); got != before {
				t.Fatalf("director.jsonl changed to %q", got)
			}
		})
	}
}

// AC3: the check-in records launched with the pid and its start time only
// for the current launch cmd registration with the right token, still
// starting; every other check-in is refused with ErrDirectorCheckInRefused,
// says why, and writes nothing.
func TestDirectorMarkCmdLaunched_RefusesEveryOtherCheckIn(t *testing.T) {
	const prefix = "org: director: check in d1: the check-in of the launch script is refused, so its agent must not start: "
	cmdStarting := func(t *testing.T, s *DirectorStore) { mustAttachCmdHeadless(t, s) }
	for _, tc := range []struct {
		name   string
		setup  func(t *testing.T, s *DirectorStore)
		mutate func(c *DirectorCmdCheckIn)
		want   string // the whole error
	}{
		{"an id not of the form d<N>", cmdStarting, func(c *DirectorCmdCheckIn) { c.ID = "d01" },
			`org: director: check in: the check-in of the launch script is refused, so its agent must not start: "d01" is not a registration id (d<N>, such as d1)`},
		{"a malformed token", cmdStarting, func(c *DirectorCmdCheckIn) { c.Token = "0123456789AB" },
			prefix + `the token "0123456789AB" is not 12 lowercase hex digits`},
		{"pid 0", cmdStarting, func(c *DirectorCmdCheckIn) { c.PID = 0 }, prefix + "the pid 0 is not a positive number"},
		{"a negative pid", cmdStarting, func(c *DirectorCmdCheckIn) { c.PID = -1 }, prefix + "the pid -1 is not a positive number"},
		{"no start time", cmdStarting, func(c *DirectorCmdCheckIn) { c.StartTime = " \t" }, prefix + "the start time of pid 4242 is empty"},
		{"an id no registration has", cmdStarting, func(c *DirectorCmdCheckIn) { c.ID = "d2" },
			"org: director: check in d2: the check-in of the launch script is refused, so its agent must not start: no registration d2 is recorded"},
		{"a launch herdr", func(t *testing.T, s *DirectorStore) { mustAttachHeadless(t, s) }, nil,
			prefix + "d1 (driver codex, attached " + directorTestNow + ", headless starting) was not started with [org.director].launch_cmd"},
		// The token comes first, so a caller without it learns nothing of
		// the registration's launch: a promoted session has no token.
		{"another token on a launch herdr", func(t *testing.T, s *DirectorStore) { mustAttachHeadless(t, s) },
			func(c *DirectorCmdCheckIn) { c.Token = "aaaaaaaaaaaa" }, prefix + "the token is not the one d1 was started with"},
		{"a promoted session", func(t *testing.T, s *DirectorStore) {
			mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude})
		}, nil, prefix + "the token is not the one d1 was started with"},
		{"another token", cmdStarting, func(c *DirectorCmdCheckIn) { c.Token = "aaaaaaaaaaaa" },
			prefix + "the token is not the one d1 was started with"},
		{"another token on a stopped registration", func(t *testing.T, s *DirectorStore) {
			mustAttachCmdHeadless(t, s)
			if _, err := s.MarkStopped("d1", testDirectorWho); err != nil {
				t.Fatal(err)
			}
		}, func(c *DirectorCmdCheckIn) { c.Token = "aaaaaaaaaaaa" }, prefix + "the token is not the one d1 was started with"},
		{"stopped before the check-in", func(t *testing.T, s *DirectorStore) {
			mustAttachCmdHeadless(t, s)
			if _, err := s.MarkStopped("d1", testDirectorWho); err != nil {
				t.Fatal(err)
			}
		}, nil, prefix + "d1 (driver codex, attached " + directorTestNow + ", headless stopped, launch cmd) is already recorded stopped (start's cleanup or a stop came first)"},
		{"detached before the check-in", func(t *testing.T, s *DirectorStore) {
			mustAttachCmdHeadless(t, s)
			if _, err := s.Detach(DirectorDetach{ID: "d1"}); err != nil {
				t.Fatal(err)
			}
		}, nil, prefix + "d1 (driver codex, attached " + directorTestNow + ", headless starting, launch cmd) is no longer the current registration (an attach --replace or a detach came in)"},
		{"replaced before the check-in", func(t *testing.T, s *DirectorStore) {
			mustAttachCmdHeadless(t, s)
			mustAttachDirector(t, s, DirectorAttach{Driver: DirectorDriverClaude, Replace: true})
		}, nil, prefix + "d1 (driver codex, attached " + directorTestNow + ", headless starting, launch cmd) is no longer the current registration (an attach --replace or a detach came in)"},
		{"a second check-in", func(t *testing.T, s *DirectorStore) {
			mustAttachCmdHeadless(t, s)
			if err := s.MarkCmdLaunched(testCheckIn(4242)); err != nil {
				t.Fatal(err)
			}
		}, func(c *DirectorCmdCheckIn) { c.PID = 5555 }, prefix + "d1 (driver codex, attached " + directorTestNow + ", headless running, launch cmd, pid 4242) has already checked in"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := newTestHeadlessStore(t)
			tc.setup(t, s)
			before := readDirectorFile(t, s)
			c := testCheckIn(4242)
			if tc.mutate != nil {
				tc.mutate(&c)
			}
			err := s.MarkCmdLaunched(c)
			if !errors.Is(err, ErrDirectorCheckInRefused) || err.Error() != tc.want {
				t.Fatalf("MarkCmdLaunched(%+v) err =\n%v\nwant\n%s", c, err, tc.want)
			}
			if got := readDirectorFile(t, s); got != before {
				t.Fatalf("a refused check-in wrote %q", strings.TrimPrefix(got, before))
			}
		})
	}
	t.Run("a missing state dir", func(t *testing.T) {
		state := filepath.Join(t.TempDir(), "absent", "org")
		err := NewDirectorStore(state).MarkCmdLaunched(testCheckIn(4242))
		if !errors.Is(err, ErrDirectorCheckInRefused) || !strings.HasSuffix(err.Error(), "no registration d1 is recorded") {
			t.Fatalf("MarkCmdLaunched err = %v, want the refusal", err)
		}
		if _, err := os.Stat(state); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("MarkCmdLaunched created the state dir (stat err = %v)", err)
		}
	})
}

// D7 on D14's comparison: a check-in that lands after stop read the
// registration makes MarkStoppedIfUnchanged refuse with
// ErrDirectorCheckedInDuringStop, writing nothing, since stop did not stop
// that process; a stop that reads again is recorded. MarkStopped and
// MarkForceStopped, which compare nothing, stop a launch cmd as they stop a
// launch herdr, and the late check-in after them is refused.
func TestDirectorMarkStoppedIfUnchanged_RefusesACheckInAfterTheRead(t *testing.T) {
	s, _, _ := newTestHeadlessStore(t)
	mustAttachCmdHeadless(t, s)
	read := *mustReadDirector(t, s).Current
	if err := s.MarkCmdLaunched(testCheckIn(4242)); err != nil {
		t.Fatal(err)
	}
	before := readDirectorFile(t, s)
	wasCurrent, err := s.MarkStoppedIfUnchanged(read, testDirectorWho)
	want := `org: director: stop d1: the launch script checked in after stop read the registration: d1 had no pid when stop read it, and has pid 4242 started "` +
		directorTestStartTime + `" now`
	if !errors.Is(err, ErrDirectorCheckedInDuringStop) || errors.Is(err, ErrDirectorChangedDuringStop) || err.Error() != want || wasCurrent {
		t.Fatalf("MarkStoppedIfUnchanged = %v, %v;\nwant false and %q", wasCurrent, err, want)
	}
	if got := readDirectorFile(t, s); got != before {
		t.Fatalf("director.jsonl changed to %q", got)
	}
	reread := *mustReadDirector(t, s).Current
	reread.StartTime = "Fri Oct 16 00:00:00 2026"
	if _, err := s.MarkStoppedIfUnchanged(reread, testDirectorWho); !errors.Is(err, ErrDirectorCheckedInDuringStop) {
		t.Fatalf("MarkStoppedIfUnchanged with another start time = %v, want ErrDirectorCheckedInDuringStop", err)
	}
	if wasCurrent, err := s.MarkStoppedIfUnchanged(*mustReadDirector(t, s).Current, testDirectorWho); err != nil || !wasCurrent {
		t.Fatalf("MarkStoppedIfUnchanged after a new read = %v, %v; want true, nil", wasCurrent, err)
	}
	wantTrail(t, s, "attached d1", "launched d1", "detached d1", "stopped d1")

	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprintf("force %v", force), func(t *testing.T) {
			s, _, _ := newTestHeadlessStore(t)
			mustAttachCmdHeadless(t, s)
			mark := s.MarkStopped
			stopped := "stopped d1"
			if force {
				mark, stopped = s.MarkForceStopped, "stopped d1+force"
			}
			if wasCurrent, err := mark("d1", testDirectorWho); err != nil || !wasCurrent {
				t.Fatalf("stop = %v, %v; want true, nil", wasCurrent, err)
			}
			wantTrail(t, s, "attached d1", "detached d1", stopped)
			if err := s.MarkCmdLaunched(testCheckIn(4242)); !errors.Is(err, ErrDirectorCheckInRefused) {
				t.Fatalf("a check-in after the stop = %v, want ErrDirectorCheckInRefused", err)
			}
			wantNoDirectorOpen(t, s)
		})
	}
}
