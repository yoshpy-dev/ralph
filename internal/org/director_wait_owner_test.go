package org

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// The tests of who writes the director's wait mark (plan
// 2026-10-19-org-director-wait, D1 and D2 at the org layer): the start reads
// director.jsonl again under director-wait.lock, so a wait whose registration
// was replaced or became unreadable since its first read writes nothing and
// leaves the mark on disk as it was; a wait whose mark file was removed
// after its first write does not write it again; and a start that cannot
// take the lock still carries the previous mark's watch over. The fixture and
// the helpers, and the AC9 rules the tests follow, are in
// director_wait_test.go; the tests of the owner once both waits have started
// are there too.

// setDirectorWaitBeforeBeginLock sets the directorWaitBeforeBeginLock seam
// for the rest of the test.
func setDirectorWaitBeforeBeginLock(t *testing.T, fn func()) {
	t.Helper()
	orig := directorWaitBeforeBeginLock
	directorWaitBeforeBeginLock = fn
	t.Cleanup(func() { directorWaitBeforeBeginLock = orig })
}

// rawDirectorWait returns the bytes of director-wait.json in dir.
func rawDirectorWait(dir string) (string, error) {
	data, err := os.ReadFile(DirectorWaitPathIn(dir))
	return string(data), err
}

// D1 and D2, the race of the cross-review: d1's wait passes its first check,
// and before its start takes director-wait.lock, `attach --replace` makes d2
// current and d2's wait writes its mark. d1's start reads director.jsonl
// again under the lock and ends with ErrDirectorWaitNotCurrent: the mark is
// still d2's byte for byte, with no end, and d1 says nothing on stderr. d2
// keeps the mark: a later rewrite and its end land, and it never says that
// another wait took the mark over. There is no watch, so a start that wrote
// would also have printed a note: line.
func TestWaitInboxForDirector_ARegistrationReplacedBeforeTheStartTakesTheLock(t *testing.T) {
	f := newDirWaitFixture(t)
	f.attach(t)
	d2Ctx, d2Cancel := context.WithCancelCause(context.Background())
	defer d2Cancel(nil)
	var d2Err lockedBuffer
	// d2's clock stays at dirWaitT0, so its rewrites leave the bytes as they
	// are, until the test moves it a minute on.
	var d2Offset atomic.Int64
	p2 := f.params("d2")
	p2.PID, p2.Stderr, p2.Timeout = 4102, &d2Err, dirWaitLimit
	p2.Now = func() time.Time { return dirWaitT0.Add(time.Duration(d2Offset.Load())) }
	d2WaitID := "4102-" + strconv.FormatInt(dirWaitT0.UnixNano(), 10)

	// The seam runs on d1's goroutine, so it fails the test with t.Errorf
	// only; what it sets is read once d1's wait has returned.
	var fired atomic.Bool
	var d2Done <-chan dirWaitOutcome
	var d2Raw string
	setDirectorWaitBeforeBeginLock(t, func() {
		if !fired.CompareAndSwap(false, true) {
			return // d2's own start
		}
		if _, _, err := f.store.Attach(DirectorAttach{Driver: DirectorDriverCodex, Replace: true, Wake: []string{DirectorWakePull}}); err != nil {
			t.Errorf("attach --replace: %v", err)
			return
		}
		d2Done = f.startDirectorWait(d2Ctx, p2)
		deadline := time.Now().Add(dirWaitLimit)
		for {
			raw, err := rawDirectorWait(f.dir)
			var m DirectorWaitMark
			if err == nil && json.Unmarshal([]byte(raw), &m) == nil && m.WaitID == d2WaitID {
				d2Raw = raw
				return
			}
			if time.Now().After(deadline) {
				t.Errorf("d2's wait never wrote its mark: last read %q, %v", raw, err)
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	})

	var d1Err lockedBuffer
	p1 := f.params("d1")
	p1.Stderr, p1.Timeout = &d1Err, dirWaitLimit
	out := f.runDirectorWait(t, p1)
	if d2Done == nil || d2Raw == "" {
		t.Fatal("the seam did not start d2's wait and see its mark")
	}
	if !errors.Is(out.err, ErrDirectorWaitNotCurrent) || !strings.Contains(out.err.Error(), "d1 is not the current director registration: the current registration is d2 (driver codex") {
		t.Errorf("d1's wait = %v; want ErrDirectorWaitNotCurrent naming d2", out.err)
	}
	if got, err := rawDirectorWait(f.dir); err != nil || got != d2Raw {
		t.Errorf("after d1's wait director-wait.json is %q (%v); want d2's as it was:\n%s", got, err, d2Raw)
	}
	if m := f.mark(t); m.WaitID != d2WaitID || m.End != "" || m.EndedAt != "" || m.Detail != "" {
		t.Errorf("after d1's wait the mark is %+v; want d2's with no end (no not_current)", m)
	}
	if s := d1Err.String(); s != "" {
		t.Errorf("d1's stderr = %q; want nothing (no note: line, no line about the mark)", s)
	}

	d2Offset.Store(int64(time.Minute))
	f.waitForMark(t, "d2's rewrite a minute later", func(m DirectorWaitMark) bool {
		return m.WaitID == d2WaitID && m.SeenAt == "2026-10-19T09:01:00Z" && m.End == ""
	})
	d2Cancel(errors.New("signal: terminated"))
	if d2Out := awaitDirectorWait(t, d2Done); d2Out.err == nil || !strings.Contains(d2Out.err.Error(), "signal: terminated") {
		t.Fatalf("d2's wait = %v; want its signal", d2Out.err)
	}
	m := f.mark(t)
	if m.WaitID != d2WaitID {
		t.Errorf("after d2's end the mark is %+v; want d2's", m)
	}
	wantDirectorWaitEnd(t, m, DirectorWaitEndSignal, false, "signal: terminated")
	if s := d2Err.String(); strings.Contains(s, directorWaitNoticePrefix) {
		t.Errorf("d2's stderr = %q; want no line about the mark (no wait took it over)", s)
	}
}

// D1, the start's read under the lock: a director.jsonl that cannot be read
// there (a directory put in its place after the first read) ends the wait
// with a plain error, not ErrDirectorWaitNotCurrent, and writes nothing: no
// mark where there was none, the previous wait's mark as it was where there
// was one, and nothing on stderr.
func TestWaitInboxForDirector_ARegistrationThatCannotBeReadUnderTheStartsLockWritesNothing(t *testing.T) {
	for name, prev := range map[string]bool{"no mark before it": false, "the previous wait's mark": true} {
		t.Run(name, func(t *testing.T) {
			f := newDirWaitFixture(t)
			reg := f.attach(t)
			before := ""
			if prev {
				f.writeMark(t, DirectorWaitMark{WaitID: "1-1", ID: reg.ID, AttachedAt: reg.AttachedAt, PID: 1, WatchAccepting: true, End: DirectorWaitEndTimeout})
				raw, err := rawDirectorWait(f.dir)
				if err != nil {
					t.Fatal(err)
				}
				before = raw
			}
			setDirectorWaitBeforeBeginLock(t, func() {
				if err := os.Remove(f.store.Path()); err != nil {
					t.Errorf("remove director.jsonl: %v", err)
					return
				}
				if err := os.Mkdir(f.store.Path(), 0o755); err != nil {
					t.Errorf("put a directory in place of director.jsonl: %v", err)
				}
			})
			p := f.params("d1")
			p.Timeout = dirWaitLimit
			out := f.runDirectorWait(t, p)
			if out.err == nil || errors.Is(out.err, ErrDirectorWaitNotCurrent) || !strings.HasPrefix(out.err.Error(), directorWaitErrPrefix+"org: director: read ") {
				t.Fatalf("err = %v; want a plain read error with the prefix", out.err)
			}
			got, err := rawDirectorWait(f.dir)
			switch {
			case !prev && !errors.Is(err, os.ErrNotExist):
				t.Errorf("director-wait.json = %q (%v); want none", got, err)
			case prev && (err != nil || got != before):
				t.Errorf("director-wait.json = %q (%v); want the previous wait's as it was:\n%s", got, err, before)
			}
			if s := f.stderr.String(); s != "" {
				t.Errorf("stderr = %q; want nothing", s)
			}
		})
	}
}

// D2: a mark removed while the wait runs, after its first write, is said
// once and not written again; the wait goes on and returns the item. The
// test removes the file under director-wait.lock, so the removal cannot land
// between a save's read and its rename.
func TestWaitInboxForDirector_AMarkRemovedWhileTheWaitRunsIsNotWrittenAgain(t *testing.T) {
	f := newDirWaitFixture(t)
	f.attach(t)
	holdRunningWatch(t, f.dir)
	p := f.params("d1")
	p.Timeout = dirWaitLimit
	done := f.startDirectorWait(context.Background(), p)
	f.waitForMark(t, "the start", func(DirectorWaitMark) bool { return true })
	if err := withFileLock(f.dir, directorWaitLockFile, "test", func() error { return os.Remove(DirectorWaitPathIn(f.dir)) }); err != nil {
		t.Fatal(err)
	}
	waitForStderr(t, f.stderr, "no longer holds the mark of this wait")
	time.Sleep(150 * time.Millisecond) // several more mark intervals of 20ms
	f.escalate(t, "org-a")
	out := awaitDirectorWait(t, done)
	if out.err != nil || len(out.items) != 1 {
		t.Fatalf("wait = %v, %v; want e1", waitInboxItemIDs(out.items), out.err)
	}
	if _, err := os.Stat(DirectorWaitPathIn(f.dir)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the removed mark was written again (stat err %v)", err)
	}
	if s := f.stderr.String(); strings.Count(s, "\n") != 1 || !strings.HasPrefix(s, directorWaitNoticePrefix) {
		t.Errorf("stderr = %q; want one line about the mark", s)
	}
}

// D2 and D3: a start that cannot take director-wait.lock (a directory here)
// does not get to the read of director.jsonl under it, reads the previous
// mark of this registration without the lock, and carries its
// watch_accepting over, so with no watch now it ends for the watch. The
// failed write is said once, no note: line is written, and the previous mark
// stays, since the end cannot take the lock either.
func TestWaitInboxForDirector_AStartThatCannotTakeTheLockStillCarriesOverTheWatch(t *testing.T) {
	f := newDirWaitFixture(t)
	reg := f.attach(t)
	f.writeMark(t, DirectorWaitMark{WaitID: "1-1", ID: reg.ID, AttachedAt: reg.AttachedAt, PID: 1, WatchAccepting: true, End: DirectorWaitEndTimeout})
	if err := os.Mkdir(filepath.Join(f.dir, directorWaitLockFile), 0o755); err != nil {
		t.Fatal(err)
	}
	p := f.params("d1")
	p.Timeout = 300 * time.Millisecond
	out := f.runDirectorWait(t, p)
	if !errors.Is(out.err, ErrDirectorWaitNoWatch) || !strings.Contains(out.err.Error(), "accepted work when the last wait of d1 wrote its mark") {
		t.Fatalf("err = %v; want ErrDirectorWaitNoWatch carried over without the lock", out.err)
	}
	if s := f.stderr.String(); strings.Count(s, "the wait mark is not written") != 1 || strings.Contains(s, "note: ") {
		t.Errorf("stderr = %q; want one line about the mark and no note:", s)
	}
	if m := f.mark(t); m.WaitID != "1-1" {
		t.Errorf("mark = %+v; want the previous wait's", m)
	}
}
