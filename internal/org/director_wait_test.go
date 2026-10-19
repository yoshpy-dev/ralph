package org

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The tests of the director's inbox wait (plan 2026-10-19-org-director-wait,
// AC1 to AC5 at the org layer; AC9: every ledger is a t.TempDir, and no test
// runs herdr, ps, a terminal, gh, or the network, or sends a signal). A watch
// is simulated as the watch gate tests do: the test holds watch.lock
// (mustAcquireWatchLock) and writes watch-alive.json (mustWriteWatchAlive).

// dirWaitT0 is the clock the waits are given (DirectorWaitParams.Now) and the
// ts of the watch marks, so a running mark written at it is fresh.
var dirWaitT0 = time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)

// dirWaitPID is the pid a wait records unless a test gives another.
const dirWaitPID = 4100

// dirWaitLimit bounds every wait a test runs, so one that never ends fails
// instead of hanging the test binary.
const dirWaitLimit = 10 * time.Second

// dirWaitCauseWords are what the text about a watch must never say: it
// gives the facts of the lock and the mark, not why the watch stopped.
var dirWaitCauseWords = []string{"crash", "died", "killed", "you stopped", "by the human", "deliberately", "panic", "hung"}

type dirWaitFixture struct {
	o      *Org
	dir    string
	store  *DirectorStore
	stderr *lockedBuffer
}

func newDirWaitFixture(t *testing.T) *dirWaitFixture {
	t.Helper()
	ef := newEscalateFixture(t)
	ef.o.InboxPollInterval = 5 * time.Millisecond
	store := NewDirectorStore(ef.stateDir)
	store.Now = func() time.Time { return dirWaitT0.Add(-time.Hour) }
	return &dirWaitFixture{o: ef.o, dir: ef.stateDir, store: store, stderr: &lockedBuffer{}}
}

// params is a wait for id with short intervals and a settle of 50ms, at
// dirWaitT0 and with dirWaitPID; a test changes what it needs.
func (f *dirWaitFixture) params(id string) DirectorWaitParams {
	return DirectorWaitParams{
		ID: id, StateDir: f.dir, Stderr: f.stderr,
		MarkInterval: 20 * time.Millisecond, WatchProbeInterval: 10 * time.Millisecond, Settle: 50 * time.Millisecond,
		Now: func() time.Time { return dirWaitT0 }, PID: dirWaitPID,
	}
}

// attach records a promoted session's registration: d1 on an empty ledger.
func (f *dirWaitFixture) attach(t *testing.T) DirectorRegistration {
	t.Helper()
	return mustAttachDirector(t, f.store, DirectorAttach{Driver: DirectorDriverClaude, Wake: []string{DirectorWakePull}, User: "u", Cwd: "/w"})
}

func (f *dirWaitFixture) escalate(t *testing.T, orgID string) {
	t.Helper()
	if _, err := f.o.Inbox.Escalate(InboxEscalation{OrgID: orgID, Type: "QUESTION", Body: "TYPE: QUESTION\n\nwhich?"}); err != nil {
		t.Fatalf("escalate: %v", err)
	}
}

// mark reads director-wait.json, which must be there.
func (f *dirWaitFixture) mark(t *testing.T) DirectorWaitMark {
	t.Helper()
	m, err := ReadDirectorWait(f.dir)
	if err != nil || m == nil {
		t.Fatalf("ReadDirectorWait = %+v, %v; want a mark", m, err)
	}
	return *m
}

// writeMark writes m as director-wait.json, as an earlier wait left it.
func (f *dirWaitFixture) writeMark(t *testing.T, m DirectorWaitMark) {
	t.Helper()
	if err := writeJSONFileAtomic(DirectorWaitPathIn(f.dir), m); err != nil {
		t.Fatal(err)
	}
}

// waitForMark polls director-wait.json until ok holds for it, and fails the
// test when it does not within dirWaitLimit.
func (f *dirWaitFixture) waitForMark(t *testing.T, what string, ok func(DirectorWaitMark) bool) DirectorWaitMark {
	t.Helper()
	deadline := time.Now().Add(dirWaitLimit)
	for {
		m, err := ReadDirectorWait(f.dir)
		if err == nil && m != nil && ok(*m) {
			return *m
		}
		if time.Now().After(deadline) {
			t.Fatalf("director-wait.json never showed %s: last read %+v, %v", what, m, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// dirWaitOutcome is what one WaitInboxForDirector call returned.
type dirWaitOutcome struct {
	items   []InboxItem
	err     error
	elapsed time.Duration
}

// startDirectorWait runs f.o.WaitInboxForDirector(ctx, p) in a goroutine.
func (f *dirWaitFixture) startDirectorWait(ctx context.Context, p DirectorWaitParams) <-chan dirWaitOutcome {
	done := make(chan dirWaitOutcome, 1)
	start := time.Now()
	go func() {
		items, err := f.o.WaitInboxForDirector(ctx, p)
		done <- dirWaitOutcome{items: items, err: err, elapsed: time.Since(start)}
	}()
	return done
}

func awaitDirectorWait(t *testing.T, done <-chan dirWaitOutcome) dirWaitOutcome {
	t.Helper()
	select {
	case out := <-done:
		return out
	case <-time.After(dirWaitLimit):
		t.Fatalf("WaitInboxForDirector did not return within %v", dirWaitLimit)
		return dirWaitOutcome{}
	}
}

// runDirectorWait runs one wait to its end.
func (f *dirWaitFixture) runDirectorWait(t *testing.T, p DirectorWaitParams) dirWaitOutcome {
	t.Helper()
	return awaitDirectorWait(t, f.startDirectorWait(context.Background(), p))
}

// replaceDirectorEvents is writeDirectorEvents through a rename, so a wait
// that reads director.jsonl meanwhile reads the old file or the new one.
func replaceDirectorEvents(t *testing.T, s *DirectorStore, evs ...DirectorEvent) {
	t.Helper()
	var b strings.Builder
	for _, ev := range evs {
		raw, err := json.Marshal(ev)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(raw)
		b.WriteByte('\n')
	}
	tmp := s.Path() + ".new"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, s.Path()); err != nil {
		t.Fatal(err)
	}
}

// holdRunningWatch takes watch.lock for the rest of the test and writes a
// running mark of pid 4242 at dirWaitT0: a watch that accepts work.
func holdRunningWatch(t *testing.T, dir string) *WatchLock {
	t.Helper()
	lock := mustAcquireWatchLock(t, dir)
	mustWriteWatchAlive(t, dir, runningMark(4242, dirWaitT0))
	return lock
}

// wantDirectorWaitEnd checks the mark's end and watch_accepting, and that
// its detail holds detail.
func wantDirectorWaitEnd(t *testing.T, m DirectorWaitMark, end string, accepting bool, detail string) {
	t.Helper()
	if m.End != end || m.EndedAt == "" || m.WatchAccepting != accepting || !strings.Contains(m.Detail, detail) {
		t.Errorf("mark = %+v; want end %q with ended_at, watch_accepting %v, and a detail holding %q", m, end, accepting, detail)
	}
}

// wantNoDirectorWaitFiles checks that dir holds neither director-wait.json
// nor director-wait.lock.
func wantNoDirectorWaitFiles(t *testing.T, dir string) {
	t.Helper()
	for _, name := range []string{DirectorWaitFileName, directorWaitLockFile} {
		if _, err := os.Lstat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s exists (or cannot be read: %v); want none", name, err)
		}
	}
}

// The mark's JSON keys (D2): the ten of the plan, the last three only once
// the wait ended.
func TestDirectorWaitMark_JSONKeys(t *testing.T) {
	keysOf := func(m DirectorWaitMark) []string {
		raw, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			t.Fatal(err)
		}
		var keys []string
		for k := range obj {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		return keys
	}
	running := DirectorWaitMark{WaitID: "4100-1", ID: "d1", AttachedAt: "a", PID: 4100, StartedAt: "s", SeenAt: "t", WatchAccepting: true}
	if got, want := keysOf(running), []string{"attached_at", "id", "pid", "seen_at", "started_at", "wait_id", "watch_accepting"}; !slices.Equal(got, want) {
		t.Errorf("keys of a running wait's mark = %v; want %v", got, want)
	}
	ended := running
	ended.EndedAt, ended.End, ended.Detail = "u", DirectorWaitEndTimeout, "d"
	if got, want := keysOf(ended), []string{"attached_at", "detail", "end", "ended_at", "id", "pid", "seen_at", "started_at", "wait_id", "watch_accepting"}; !slices.Equal(got, want) {
		t.Errorf("keys of an ended wait's mark = %v; want %v", got, want)
	}
}

// ReadDirectorWait: a missing mark or state dir is nil with no error; a mark
// that does not parse or cannot be read is an error; a written mark reads
// back as written, and the write leaves no temporary file.
func TestReadDirectorWait_MissingCorruptAndRoundTrip(t *testing.T) {
	dir := t.TempDir()
	for name, d := range map[string]string{"no mark": dir, "no state dir": filepath.Join(dir, "missing")} {
		if m, err := ReadDirectorWait(d); m != nil || err != nil {
			t.Errorf("%s: ReadDirectorWait = %+v, %v; want nil, nil", name, m, err)
		}
	}

	want := DirectorWaitMark{WaitID: "4100-1760864400000000000", ID: "d3", AttachedAt: "2026-10-19T08:00:00Z", PID: 4100,
		StartedAt: "2026-10-19T09:00:00Z", SeenAt: "2026-10-19T09:00:30Z", EndedAt: "2026-10-19T09:00:31Z", End: DirectorWaitEndItems, Detail: "1 open inbox item"}
	if err := writeJSONFileAtomic(DirectorWaitPathIn(dir), want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadDirectorWait(dir)
	if err != nil || got == nil || *got != want {
		t.Fatalf("ReadDirectorWait = %+v, %v; want %+v", got, err, want)
	}
	if names := dirEntries(t, dir); !slices.Equal(names, []string{DirectorWaitFileName}) {
		t.Errorf("state dir holds %v; want only %s (no temporary file)", names, DirectorWaitFileName)
	}

	if err := os.WriteFile(DirectorWaitPathIn(dir), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if m, err := ReadDirectorWait(dir); m != nil || err == nil || !strings.Contains(err.Error(), "parse") {
		t.Errorf("a corrupt mark: ReadDirectorWait = %+v, %v; want a parse error", m, err)
	}

	asDir := t.TempDir()
	if err := os.Mkdir(DirectorWaitPathIn(asDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if m, err := ReadDirectorWait(asDir); m != nil || err == nil {
		t.Errorf("a mark that cannot be read: ReadDirectorWait = %+v, %v; want an error", m, err)
	}
}

// DirectorWaitFor: a mark counts only for the registration whose id and
// attached_at it holds, as DirectorAliveFor's.
func TestDirectorWaitFor(t *testing.T) {
	reg := &DirectorRegistration{ID: "d2", AttachedAt: "2026-10-19T08:00:00Z"}
	mark := &DirectorWaitMark{WaitID: "1-2", ID: "d2", AttachedAt: "2026-10-19T08:00:00Z", WatchAccepting: true}
	if got := DirectorWaitFor(mark, reg); got != mark {
		t.Fatalf("DirectorWaitFor of its own registration = %+v; want the mark", got)
	}
	for name, tc := range map[string]struct {
		mark *DirectorWaitMark
		reg  *DirectorRegistration
	}{
		"another id":               {&DirectorWaitMark{ID: "d1", AttachedAt: mark.AttachedAt}, reg},
		"another attached_at":      {&DirectorWaitMark{ID: "d2", AttachedAt: "2026-10-18T08:00:00Z"}, reg},
		"no attached_at in a mark": {&DirectorWaitMark{ID: "d2"}, reg},
		"no mark":                  {nil, reg},
		"no registration":          {mark, nil},
	} {
		if got := DirectorWaitFor(tc.mark, tc.reg); got != nil {
			t.Errorf("%s: DirectorWaitFor = %+v; want nil", name, got)
		}
	}
}

// D1: the registrations a wait for d1 may wait for, and the refusals, which
// wrap ErrDirectorWaitNotCurrent and name the current registration with its
// attach time escaped.
func TestDirectorWaitCheckCurrent(t *testing.T) {
	at := "2026-10-19T08:00:00Z"
	promoted := DirectorRegistration{ID: "d1", Driver: DirectorDriverClaude, AttachedAt: at}
	headless := func(state string) *DirectorRegistration {
		return &DirectorRegistration{ID: "d1", Driver: DirectorDriverCodex, Launch: DirectorLaunchHerdr, State: state, AttachedAt: at}
	}
	for name, cur := range map[string]*DirectorRegistration{
		"promoted session": &promoted, "headless starting": headless(DirectorStateStarting), "headless running": headless(DirectorStateRunning),
	} {
		if detail, err := directorWaitCheckCurrent("d1", nil, cur); err != nil || detail != "" {
			t.Errorf("%s at the start: %q, %v; want no refusal", name, detail, err)
		}
		if detail, err := directorWaitCheckCurrent("d1", &promoted, cur); err != nil || detail != "" {
			t.Errorf("%s once started: %q, %v; want no refusal", name, detail, err)
		}
	}

	again := promoted
	again.AttachedAt = "2026-10-19T08:30:00Z"
	if _, err := directorWaitCheckCurrent("d1", nil, &again); err != nil {
		t.Errorf("another attached_at at the start: %v; want no refusal (the start binds the wait to it)", err)
	}
	other := promoted
	other.ID = "d2"
	escaped := promoted
	escaped.AttachedAt = "2026-10-19\x1b[2J"
	rows := []struct {
		name    string
		started *DirectorRegistration
		cur     *DirectorRegistration
		detail  string
		why     string
	}{
		{"none", nil, nil, "no registration is current", "d1 is not the current director registration: no registration is current"},
		{"another id", nil, &other, "the current registration is d2", "the current registration is d2 (driver claude, attached " + at + ")"},
		{"stopped", nil, headless(DirectorStateStopped), "d1 is stopped", "d1 (driver codex, attached " + at + ", headless stopped) is stopped"},
		{"attached again", &promoted, &again, "d1 was attached again", "the current registration is d1 (driver claude, attached 2026-10-19T08:30:00Z), not the d1 attached " + at + " that this wait started for"},
		{"another id with a hand-edited attach time", nil, func() *DirectorRegistration { r := escaped; r.ID = "d2"; return &r }(), "the current registration is d2", `attached 2026-10-19\x1b[2J`},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			detail, err := directorWaitCheckCurrent("d1", r.started, r.cur)
			if !errors.Is(err, ErrDirectorWaitNotCurrent) {
				t.Fatalf("err = %v; want ErrDirectorWaitNotCurrent", err)
			}
			if !strings.HasPrefix(err.Error(), directorWaitErrPrefix+"d1 is not the current director registration: ") || !strings.Contains(err.Error(), r.why) {
				t.Errorf("err = %q; want the prefix and %q", err, r.why)
			}
			if strings.Contains(err.Error(), "\x1b") {
				t.Errorf("err = %q holds a raw control character", err)
			}
			if detail != r.detail {
				t.Errorf("detail = %q; want %q", detail, r.detail)
			}
		})
	}
}

// AC1 at the org layer: an id that is not d<N> is refused before anything is
// read (director.jsonl is a directory here, which a read would fail on), and
// a registration that is not current is refused with
// ErrDirectorWaitNotCurrent. Neither writes a mark or a lock file. A
// director.jsonl that cannot be read is a plain error, with nothing written.
func TestWaitInboxForDirector_RefusesBeforeItWaits(t *testing.T) {
	// A wait that is not refused ends at a short timeout, so the check fails
	// instead of waiting.
	refusal := func(t *testing.T, f *dirWaitFixture, id string) error {
		t.Helper()
		p := f.params(id)
		p.Timeout = 100 * time.Millisecond
		return f.runDirectorWait(t, p).err
	}
	for _, id := range []string{"", "d0", "d01", "D1", "x", "d1 "} {
		f := newDirWaitFixture(t)
		if err := os.MkdirAll(DirectorPathIn(f.dir), 0o755); err != nil {
			t.Fatal(err)
		}
		err := refusal(t, f, id)
		if err == nil || errors.Is(err, ErrDirectorWaitNotCurrent) || !strings.Contains(err.Error(), "is not a registration id (d<N>") {
			t.Errorf("id %q: err = %v; want the id refused, not a read", id, err)
		}
		wantNoDirectorWaitFiles(t, f.dir)
	}

	notCurrent := []struct {
		name  string
		setup func(t *testing.T, f *dirWaitFixture)
		want  string
	}{
		{"no registration", func(*testing.T, *dirWaitFixture) {}, "no registration is current"},
		{"another id", func(t *testing.T, f *dirWaitFixture) {
			f.attach(t)
			if _, _, err := f.store.Attach(DirectorAttach{Driver: DirectorDriverCodex, Replace: true, Wake: []string{DirectorWakePull}}); err != nil {
				t.Fatal(err)
			}
		}, "the current registration is d2 (driver codex"},
		{"detached", func(t *testing.T, f *dirWaitFixture) {
			f.attach(t)
			if _, err := f.store.Detach(DirectorDetach{ID: "d1"}); err != nil {
				t.Fatal(err)
			}
		}, "no registration is current"},
		{"stopped but still current (a hand edit)", func(t *testing.T, f *dirWaitFixture) {
			writeDirectorEvents(t, f.store, append(checkHerdrRunning(), checkEvent("d1", DirectorEventStopped, nil))...)
		}, "headless stopped) is stopped"},
	}
	for _, tc := range notCurrent {
		t.Run(tc.name, func(t *testing.T) {
			f := newDirWaitFixture(t)
			tc.setup(t, f)
			err := refusal(t, f, "d1")
			if !errors.Is(err, ErrDirectorWaitNotCurrent) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v; want ErrDirectorWaitNotCurrent saying %q", err, tc.want)
			}
			wantNoDirectorWaitFiles(t, f.dir)
		})
	}

	t.Run("director.jsonl cannot be read", func(t *testing.T) {
		f := newDirWaitFixture(t)
		if err := os.MkdirAll(DirectorPathIn(f.dir), 0o755); err != nil {
			t.Fatal(err)
		}
		err := refusal(t, f, "d1")
		if err == nil || errors.Is(err, ErrDirectorWaitNotCurrent) || !strings.HasPrefix(err.Error(), directorWaitErrPrefix+"org: director: read ") {
			t.Fatalf("err = %v; want a plain read error with the prefix", err)
		}
		wantNoDirectorWaitFiles(t, f.dir)
	})
}

// AC1: CheckDirectorWaitID, which the CLI calls before it reads anything,
// accepts the ids a registration is written with and refuses every other id
// with the text WaitInboxForDirector refused it with before the check was
// split out of it (spelled out here, not built from directorWaitErrPrefix).
func TestCheckDirectorWaitID_AcceptsOnlyDN(t *testing.T) {
	for _, id := range []string{"d1", "d12", "d9223372036854775807"} {
		if err := CheckDirectorWaitID(id); err != nil {
			t.Errorf("CheckDirectorWaitID(%q) = %v; want nil", id, err)
		}
	}
	for _, id := range []string{"", "d", "d0", "d01", "d-1", "d+1", "x1", "D1", "d1x", "d1 ", "e1", "d9223372036854775808"} {
		want := "org: wait --inbox --director: " + strconv.Quote(id) + " is not a registration id (d<N>, such as d1)"
		if err := CheckDirectorWaitID(id); err == nil || err.Error() != want {
			t.Errorf("CheckDirectorWaitID(%q) = %v; want %q", id, err, want)
		}
	}
}

// D1: a promoted session's registration and a headless one that is starting
// or running are waited for, and the mark binds the wait to that
// registration.
func TestWaitInboxForDirector_WaitsForEachRegistrationThatIsCurrent(t *testing.T) {
	rows := map[string]func(t *testing.T, f *dirWaitFixture){
		"promoted session": func(t *testing.T, f *dirWaitFixture) { f.attach(t) },
		"headless starting": func(t *testing.T, f *dirWaitFixture) {
			writeDirectorEvents(t, f.store, checkAttached(DirectorLaunchHerdr, true))
		},
		"headless running": func(t *testing.T, f *dirWaitFixture) { writeDirectorEvents(t, f.store, checkHerdrRunning()...) },
	}
	for name, setup := range rows {
		t.Run(name, func(t *testing.T) {
			f := newDirWaitFixture(t)
			setup(t, f)
			f.escalate(t, "org-a")
			out := f.runDirectorWait(t, f.params("d1"))
			if out.err != nil || !slices.Equal(waitInboxItemIDs(out.items), []string{"e1"}) {
				t.Fatalf("wait = %v, %v; want e1", waitInboxItemIDs(out.items), out.err)
			}
			cur := mustReadDirector(t, f.store).Current
			if m := f.mark(t); DirectorWaitFor(&m, cur) == nil {
				t.Errorf("mark %+v is not bound to the current registration %+v", m, cur)
			}
		})
	}
}

// AC2: a mark that cannot be written is said once on stderr, however many
// writes fail, and the wait goes on to its timeout.
func TestWaitInboxForDirector_AMarkThatCannotBeWrittenIsSaidOnce(t *testing.T) {
	f := newDirWaitFixture(t)
	f.attach(t)
	holdRunningWatch(t, f.dir)
	if err := os.Mkdir(DirectorWaitPathIn(f.dir), 0o755); err != nil {
		t.Fatal(err)
	}
	p := f.params("d1")
	p.MarkInterval, p.Timeout = 5*time.Millisecond, 300*time.Millisecond
	out := f.runDirectorWait(t, p)
	if out.err == nil || !strings.Contains(out.err.Error(), inboxWaitTimeoutPhrase) {
		t.Fatalf("err = %v; want the timeout", out.err)
	}
	if s := f.stderr.String(); strings.Count(s, "\n") != 1 || !strings.HasPrefix(s, "wait --inbox --director: the wait mark is not written (this wait keeps waiting, and says this once): ") {
		t.Errorf("stderr = %q; want one line about the mark", s)
	}
}

// D2: a wait whose first write failed writes once it can, in place of the
// mark it read at its start (none, or the previous wait's): that is not
// another wait taking the mark over.
func TestWaitInboxForDirector_AWaitWhoseFirstWriteFailedWritesOnceItCan(t *testing.T) {
	finish := func(t *testing.T, f *dirWaitFixture, done <-chan dirWaitOutcome) {
		t.Helper()
		f.waitForMark(t, "this wait's mark", func(m DirectorWaitMark) bool { return m.PID == dirWaitPID })
		f.escalate(t, "org-a")
		out := awaitDirectorWait(t, done)
		if out.err != nil || len(out.items) != 1 {
			t.Fatalf("wait = %v, %v; want e1", waitInboxItemIDs(out.items), out.err)
		}
		wantDirectorWaitEnd(t, f.mark(t), DirectorWaitEndItems, true, "1 open inbox item")
		s := f.stderr.String()
		if strings.Count(s, "\n") != 1 || !strings.Contains(s, "the wait mark is not written") {
			t.Errorf("stderr = %q; want only the line about the failed write", s)
		}
	}
	t.Run("no mark before it", func(t *testing.T) {
		f := newDirWaitFixture(t)
		f.attach(t)
		holdRunningWatch(t, f.dir)
		if err := os.Mkdir(DirectorWaitPathIn(f.dir), 0o755); err != nil {
			t.Fatal(err)
		}
		p := f.params("d1")
		p.Timeout = dirWaitLimit
		done := f.startDirectorWait(context.Background(), p)
		waitForStderr(t, f.stderr, "the wait mark is not written")
		if err := os.Remove(DirectorWaitPathIn(f.dir)); err != nil {
			t.Fatal(err)
		}
		finish(t, f, done)
	})
	t.Run("the previous wait's mark", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root writes into a directory without the write bit")
		}
		f := newDirWaitFixture(t)
		reg := f.attach(t)
		holdRunningWatch(t, f.dir)
		f.writeMark(t, DirectorWaitMark{WaitID: "1-1", ID: reg.ID, AttachedAt: reg.AttachedAt, PID: 1, End: DirectorWaitEndTimeout})
		// The lock file is there, so the start takes the lock and reads the
		// previous mark, and only its write into the read-only dir fails.
		if err := os.WriteFile(filepath.Join(f.dir, directorWaitLockFile), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(f.dir, 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(f.dir, 0o755) })
		p := f.params("d1")
		p.Timeout = dirWaitLimit
		done := f.startDirectorWait(context.Background(), p)
		waitForStderr(t, f.stderr, "the wait mark is not written")
		if m := f.mark(t); m.WaitID != "1-1" {
			t.Fatalf("mark = %+v; want the previous wait's while the dir is read-only", m)
		}
		if err := os.Chmod(f.dir, 0o755); err != nil {
			t.Fatal(err)
		}
		finish(t, f, done)
	})
}

// waitForStderr polls b until it holds want, and fails the test when it
// does not within dirWaitLimit.
func waitForStderr(t *testing.T, b *lockedBuffer, want string) {
	t.Helper()
	deadline := time.Now().Add(dirWaitLimit)
	for !strings.Contains(b.String(), want) {
		if time.Now().After(deadline) {
			t.Fatalf("stderr never said %q: %q", want, b.String())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// AC3 with the write steps driven one by one on a clock the test moves: A
// starts, B starts, A rewrites, B ends. The mark is B's with B's end; A
// writes nothing after B took over, and says so once.
func TestDirectorWait_AMarkTakenOverIsNotWrittenAgain(t *testing.T) {
	f := newDirWaitFixture(t)
	reg := f.attach(t)
	clock := dirWaitT0
	newRun := func(pid int, stderr *lockedBuffer) *directorWaitRun {
		p := f.params("d1")
		p.PID, p.Stderr = pid, stderr
		p.Now = func() time.Time { return clock }
		w := newDirectorWaitRun(f.o, p)
		w.reg = reg
		return w
	}
	raw := func() string {
		data, err := os.ReadFile(DirectorWaitPathIn(f.dir))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	var aErr, bErr lockedBuffer
	a := newRun(4101, &aErr)
	a.begin(context.Background())
	if m := f.mark(t); m.WaitID != "4101-"+strconv.FormatInt(dirWaitT0.UnixNano(), 10) {
		t.Fatalf("after A's start the mark is %+v; want A's", m)
	}

	clock = clock.Add(time.Second)
	b := newRun(4102, &bErr)
	b.begin(context.Background())
	bWaitID := "4102-" + strconv.FormatInt(clock.UnixNano(), 10)
	if m := f.mark(t); m.WaitID != bWaitID {
		t.Fatalf("after B's start the mark is %+v; want B's", m)
	}
	afterB := raw()

	clock = clock.Add(time.Second)
	a.save()
	if got := raw(); got != afterB {
		t.Errorf("A's rewrite after B's start changed the mark:\n%s\nwant\n%s", got, afterB)
	}

	clock = clock.Add(time.Second)
	b.end(DirectorWaitEndTimeout, "no open inbox item within 300000 ms")
	m := f.mark(t)
	if m.WaitID != bWaitID || m.PID != 4102 || m.End != DirectorWaitEndTimeout || m.SeenAt != "2026-10-19T09:00:03Z" || m.EndedAt != "2026-10-19T09:00:03Z" {
		t.Fatalf("after B's end the mark is %+v; want B's with its end", m)
	}
	afterBEnd := raw()

	clock = clock.Add(time.Second)
	a.save()
	a.end(DirectorWaitEndSignal, "signal: terminated")
	if got := raw(); got != afterBEnd {
		t.Errorf("A wrote after B took over:\n%s\nwant\n%s", got, afterBEnd)
	}
	took := "wait --inbox --director: another ralph org wait --inbox --director (wait_id " + bWaitID + ", pid 4102) took over "
	if s := aErr.String(); strings.Count(s, "took over") != 1 || !strings.Contains(s, took) {
		t.Errorf("A's stderr = %q; want one line saying %q", s, took)
	}
	if s := bErr.String(); strings.Contains(s, "took over") || strings.Contains(s, "not written") {
		t.Errorf("B's stderr = %q; want no line about the mark", s)
	}
}

// AC3 with two waits running: B starts while A waits, and times out; A's
// later rewrites and its end leave B's ended mark as it is.
func TestWaitInboxForDirector_TwoWaitsLeaveTheLaterOnesMark(t *testing.T) {
	f := newDirWaitFixture(t)
	f.attach(t)
	holdRunningWatch(t, f.dir)
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	var aErr, bErr lockedBuffer
	pa := f.params("d1")
	pa.PID, pa.Stderr = 4101, &aErr
	aDone := f.startDirectorWait(ctx, pa)
	f.waitForMark(t, "A's start", func(m DirectorWaitMark) bool { return m.PID == 4101 })

	pb := f.params("d1")
	pb.PID, pb.Stderr, pb.Timeout = 4102, &bErr, 300*time.Millisecond
	bOut := awaitDirectorWait(t, f.startDirectorWait(context.Background(), pb))
	if bOut.err == nil || !strings.Contains(bOut.err.Error(), inboxWaitTimeoutPhrase) {
		t.Fatalf("B = %v; want its timeout", bOut.err)
	}
	afterB := f.mark(t)
	if afterB.PID != 4102 || afterB.End != DirectorWaitEndTimeout {
		t.Fatalf("after B the mark is %+v; want B's timeout", afterB)
	}
	time.Sleep(100 * time.Millisecond)
	cancel(errors.New("signal: terminated"))
	if aOut := awaitDirectorWait(t, aDone); !strings.Contains(aOut.err.Error(), "signal: terminated") {
		t.Fatalf("A = %v; want its signal", aOut.err)
	}
	if got := f.mark(t); got != afterB {
		t.Errorf("after A ended the mark is %+v; want B's %+v", got, afterB)
	}
	if s := aErr.String(); strings.Count(s, "took over") != 1 {
		t.Errorf("A's stderr = %q; want one line saying B took over", s)
	}
}

// AC6 at the org layer: the wait without --director writes no mark and no
// lock file.
func TestWaitInbox_WritesNoDirectorWaitMark(t *testing.T) {
	f := newDirWaitFixture(t)
	f.attach(t)
	f.escalate(t, "org-a")
	if items, err := f.o.WaitInbox(0); err != nil || len(items) != 1 {
		t.Fatalf("WaitInbox = %v, %v; want e1", waitInboxItemIDs(items), err)
	}
	wantNoDirectorWaitFiles(t, f.dir)
}
