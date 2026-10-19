package org

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The tests of how the director's inbox wait ends while it waits (plan
// 2026-10-19-org-director-wait, AC2, AC4, and AC5 at the org layer): open
// items, the timeout, a watch that stops or starts accepting work (with the
// carry-over from the previous mark and the settle of a stale watch), a
// registration that changes, a done ctx, a read that fails, and the text
// about the watch. The fixture and the helpers, and the AC9 rules the tests
// follow, are in director_wait_test.go.

// AC2: an open item ends the wait with the items, and the mark holds every
// field of D2 with end items. With a watch that accepts work there is no
// note.
func TestWaitInboxForDirector_AnOpenItemEndsWithItems(t *testing.T) {
	f := newDirWaitFixture(t)
	reg := f.attach(t)
	holdRunningWatch(t, f.dir)
	f.escalate(t, "org-a")
	f.escalate(t, "org-b")
	if _, err := f.o.Inbox.Ack("e1"); err != nil {
		t.Fatal(err)
	}
	out := f.runDirectorWait(t, f.params("d1"))
	if out.err != nil || !slices.Equal(waitInboxItemIDs(out.items), []string{"e2"}) {
		t.Fatalf("wait = %v, %v; want only the open e2", waitInboxItemIDs(out.items), out.err)
	}
	want := DirectorWaitMark{
		WaitID: strconv.Itoa(dirWaitPID) + "-" + strconv.FormatInt(dirWaitT0.UnixNano(), 10),
		ID:     "d1", AttachedAt: reg.AttachedAt, PID: dirWaitPID,
		StartedAt: "2026-10-19T09:00:00Z", SeenAt: "2026-10-19T09:00:00Z", WatchAccepting: true,
		EndedAt: "2026-10-19T09:00:00Z", End: DirectorWaitEndItems, Detail: "1 open inbox item",
	}
	if got := f.mark(t); got != want {
		t.Errorf("mark = %+v; want %+v", got, want)
	}
	if s := f.stderr.String(); s != "" {
		t.Errorf("stderr = %q; want nothing", s)
	}
}

// AC2: the timeout's error holds inboxWaitTimeoutPhrase as WaitInbox's does,
// which the /director skill matches, and the mark says end timeout. With no
// watch and no previous mark the wait writes one note: line and waits on
// (AC5).
func TestWaitInboxForDirector_TimesOutWithThePhraseAfterANote(t *testing.T) {
	f := newDirWaitFixture(t)
	f.attach(t)
	p := f.params("d1")
	p.Timeout = 200 * time.Millisecond
	out := f.runDirectorWait(t, p)
	if out.err == nil || errors.Is(out.err, ErrDirectorWaitNoWatch) || !strings.Contains(out.err.Error(), inboxWaitTimeoutPhrase+" 200 ms") {
		t.Fatalf("err = %v; want the timeout with %q", out.err, inboxWaitTimeoutPhrase)
	}
	if out.elapsed < p.Timeout {
		t.Errorf("the wait timed out after %v; want at least %v", out.elapsed, p.Timeout)
	}
	wantDirectorWaitEnd(t, f.mark(t), DirectorWaitEndTimeout, false, "200 ms")
	stderr := f.stderr.String()
	if strings.Count(stderr, "note: ") != 1 || !strings.HasPrefix(stderr, "note: no ralph org watch --all accepts work on this ledger, and this wait keeps waiting: watch.lock is free and there is no readable watch-alive.json") {
		t.Errorf("stderr = %q; want one note: line about no watch", stderr)
	}
}

// The deadline is followed by one more read, as WaitInbox's: with an hour
// between rounds, the only reads are the first one and the one after the
// deadline, and an item that arrives between them is returned.
func TestWaitInboxForDirector_ReadsOnceMoreAtTheDeadline(t *testing.T) {
	f := newDirWaitFixture(t)
	f.attach(t)
	holdRunningWatch(t, f.dir)
	f.o.InboxPollInterval = time.Hour
	p := f.params("d1")
	p.MarkInterval, p.WatchProbeInterval, p.Timeout = time.Hour, time.Hour, 300*time.Millisecond
	escalateLater(t, f.o, "org-a", 50*time.Millisecond)
	out := f.runDirectorWait(t, p)
	if out.err != nil || !slices.Equal(waitInboxItemIDs(out.items), []string{"e1"}) {
		t.Fatalf("wait = %v, %v; want e1, read after the deadline", waitInboxItemIDs(out.items), out.err)
	}
	wantDirectorWaitEnd(t, f.mark(t), DirectorWaitEndItems, true, "1 open inbox item")
}

// AC5: a watch that accepted work and no longer does, in each way D3 names,
// ends the wait with ErrDirectorWaitNoWatch and the facts of that way, end
// watch and watch_accepting false.
func TestWaitInboxForDirector_AWatchThatStopsAcceptingWorkEndsTheWait(t *testing.T) {
	stopped := runningMark(4242, dirWaitT0)
	stopped.State, stopped.StoppedAt = WatchAliveStopped, "2026-10-19T09:00:05Z"
	rows := []struct {
		name   string
		change func(t *testing.T, dir string, lock *WatchLock)
		want   string
	}{
		{"the lock is released with the running mark left", func(t *testing.T, _ string, lock *WatchLock) { _ = lock.Release() },
			"watch.lock is free but watch-alive.json still says running (pid 4242, written at 2026-10-19T09:00:00Z): the watch ended without writing its stopped mark"},
		{"the stopped mark and the lock released", func(t *testing.T, dir string, lock *WatchLock) {
			mustWriteWatchAlive(t, dir, stopped)
			_ = lock.Release()
		}, "watch.lock is free and watch-alive.json says stopped at 2026-10-19T09:00:05Z (pid 4242)"},
		{"stopping", func(t *testing.T, dir string, _ *WatchLock) {
			mustWriteWatchAlive(t, dir, stoppingMark(4242, dirWaitT0))
		},
			"the watch (pid 4242) is stopping"},
		{"a --once run", func(t *testing.T, dir string, _ *WatchLock) { mustWriteWatchAlive(t, dir, onceMark(4242, dirWaitT0)) },
			"is a run that ends after its cycles (--once)"},
		{"stale after the settle", func(t *testing.T, dir string, _ *WatchLock) {
			mustWriteWatchAlive(t, dir, runningMark(4242, dirWaitT0.Add(-time.Hour)))
		}, "watch.lock is held, but watch-alive.json has not been rewritten as running since 2026-10-19T08:00:00Z (pid 4242, it says running)"},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			f := newDirWaitFixture(t)
			f.attach(t)
			lock := holdRunningWatch(t, f.dir)
			p := f.params("d1")
			p.Timeout = dirWaitLimit
			done := f.startDirectorWait(context.Background(), p)
			f.waitForMark(t, "the start with a watch that accepts work", func(m DirectorWaitMark) bool { return m.WatchAccepting })
			r.change(t, f.dir, lock)
			out := awaitDirectorWait(t, done)
			if !errors.Is(out.err, ErrDirectorWaitNoWatch) {
				t.Fatalf("err = %v; want ErrDirectorWaitNoWatch", out.err)
			}
			msg := out.err.Error()
			for _, want := range []string{
				directorWaitErrPrefix + "no watch accepts work: a ralph org watch --all accepted work when this wait last probed it, and none does now: ",
				r.want, "`ralph org watch --all --detach`", "an item escalated reaches the human at once",
			} {
				if !strings.Contains(msg, want) {
					t.Errorf("err = %q; want it to hold %q", msg, want)
				}
			}
			wantDirectorWaitEnd(t, f.mark(t), DirectorWaitEndWatch, false, "no watch accepts work")
			if s := f.stderr.String(); strings.Contains(s, "note: ") {
				t.Errorf("stderr = %q; want no note: line", s)
			}
		})
	}
}

// AC5 and the plan's edge case: an open item that arrives in the round the
// watch stops is returned first. Here the item arrives while the wait
// settles a stale watch it saw accepting work, so only the read after the
// settle finds it. The mark keeps watch_accepting true, as the change was not
// reported, and the next wait, with the item acked and the watch still
// stale, ends for it at its start.
func TestWaitInboxForDirector_AnOpenItemWinsOverAWatchThatStopped(t *testing.T) {
	f := newDirWaitFixture(t)
	f.attach(t)
	holdRunningWatch(t, f.dir)
	p := f.params("d1")
	p.Timeout = dirWaitLimit
	p.Settle = 400 * time.Millisecond
	done := f.startDirectorWait(context.Background(), p)
	f.waitForMark(t, "the start with a watch that accepts work", func(m DirectorWaitMark) bool { return m.WatchAccepting })
	if err := os.Remove(WatchAlivePathIn(f.dir)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	f.escalate(t, "org-a")
	out := awaitDirectorWait(t, done)
	if out.err != nil || !slices.Equal(waitInboxItemIDs(out.items), []string{"e1"}) {
		t.Fatalf("wait = %v, %v; want e1 before the watch", waitInboxItemIDs(out.items), out.err)
	}
	wantDirectorWaitEnd(t, f.mark(t), DirectorWaitEndItems, true, "1 open inbox item")

	if _, err := f.o.Inbox.Ack("e1"); err != nil {
		t.Fatal(err)
	}
	next := f.runDirectorWait(t, f.params("d1"))
	if !errors.Is(next.err, ErrDirectorWaitNoWatch) || !strings.Contains(next.err.Error(), "accepted work when the last wait of d1 wrote its mark") ||
		!strings.Contains(next.err.Error(), "watch.lock is held, but there is no readable watch-alive.json") {
		t.Fatalf("the next wait = %v; want ErrDirectorWaitNoWatch carried over from the mark, with the facts of a held lock and no mark", next.err)
	}
	wantDirectorWaitEnd(t, f.mark(t), DirectorWaitEndWatch, false, "no watch accepts work")
}

// AC5, the carry-over (D3): a previous mark of this registration that saw a
// watch accepting work ends a wait that finds none at its start. One of
// another registration, or one that saw none, carries nothing over: the wait
// writes a note: line and waits on.
func TestWaitInboxForDirector_CarriesOverOnlyThisRegistrationsPreviousMark(t *testing.T) {
	t.Run("this registration, accepting", func(t *testing.T) {
		f := newDirWaitFixture(t)
		reg := f.attach(t)
		f.writeMark(t, DirectorWaitMark{WaitID: "1-1", ID: reg.ID, AttachedAt: reg.AttachedAt, PID: 1, WatchAccepting: true, End: DirectorWaitEndTimeout})
		out := f.runDirectorWait(t, f.params("d1"))
		if !errors.Is(out.err, ErrDirectorWaitNoWatch) || !strings.Contains(out.err.Error(), "watch.lock is free and there is no readable watch-alive.json") {
			t.Fatalf("err = %v; want ErrDirectorWaitNoWatch with the facts of no watch", out.err)
		}
		if s := f.stderr.String(); strings.Contains(s, "note: ") {
			t.Errorf("stderr = %q; want no note: line", s)
		}
		wantDirectorWaitEnd(t, f.mark(t), DirectorWaitEndWatch, false, "watch not_running")
	})
	t.Run("this registration, accepting, with an open item at the start", func(t *testing.T) {
		f := newDirWaitFixture(t)
		reg := f.attach(t)
		f.writeMark(t, DirectorWaitMark{WaitID: "1-1", ID: reg.ID, AttachedAt: reg.AttachedAt, PID: 1, WatchAccepting: true})
		f.escalate(t, "org-a")
		out := f.runDirectorWait(t, f.params("d1"))
		if out.err != nil || !slices.Equal(waitInboxItemIDs(out.items), []string{"e1"}) {
			t.Fatalf("wait = %v, %v; want e1 before the watch", waitInboxItemIDs(out.items), out.err)
		}
		// The change was not reported, so the mark still says accepting.
		wantDirectorWaitEnd(t, f.mark(t), DirectorWaitEndItems, true, "1 open inbox item")
		if _, err := f.o.Inbox.Ack("e1"); err != nil {
			t.Fatal(err)
		}
		if next := f.runDirectorWait(t, f.params("d1")); !errors.Is(next.err, ErrDirectorWaitNoWatch) {
			t.Fatalf("the next wait = %v; want ErrDirectorWaitNoWatch", next.err)
		}
	})
	rows := map[string]func(reg DirectorRegistration) DirectorWaitMark{
		"this registration, not accepting": func(reg DirectorRegistration) DirectorWaitMark {
			return DirectorWaitMark{WaitID: "1-1", ID: reg.ID, AttachedAt: reg.AttachedAt}
		},
		"another id": func(reg DirectorRegistration) DirectorWaitMark {
			return DirectorWaitMark{WaitID: "1-1", ID: "d7", AttachedAt: reg.AttachedAt, WatchAccepting: true}
		},
		"another attached_at": func(reg DirectorRegistration) DirectorWaitMark {
			return DirectorWaitMark{WaitID: "1-1", ID: reg.ID, AttachedAt: "2026-10-01T00:00:00Z", WatchAccepting: true}
		},
	}
	for name, prev := range rows {
		t.Run(name, func(t *testing.T) {
			f := newDirWaitFixture(t)
			reg := f.attach(t)
			f.writeMark(t, prev(reg))
			p := f.params("d1")
			p.Timeout = 150 * time.Millisecond
			out := f.runDirectorWait(t, p)
			if out.err == nil || !strings.Contains(out.err.Error(), inboxWaitTimeoutPhrase) {
				t.Fatalf("err = %v; want the timeout", out.err)
			}
			if s := f.stderr.String(); strings.Count(s, "note: no ralph org watch --all accepts work") != 1 {
				t.Errorf("stderr = %q; want one note: line", s)
			}
			if m := f.mark(t); m.WaitID == "1-1" || m.ID != "d1" || m.AttachedAt != reg.AttachedAt {
				t.Errorf("mark = %+v; want this wait's, bound to d1", m)
			}
		})
	}
}

// AC5: a watch that holds the lock and writes its running mark within the
// settle is not read as one that stopped, at the start (with a previous mark
// that would carry the change over) and in a round.
func TestWaitInboxForDirector_AStaleWatchThatWritesItsMarkWithinTheSettle(t *testing.T) {
	writeLater := func(t *testing.T, dir string, delay time.Duration) {
		done := make(chan error, 1)
		go func() {
			time.Sleep(delay)
			done <- writeWatchAlive(dir, runningMark(4242, dirWaitT0))
		}()
		t.Cleanup(func() {
			if err := <-done; err != nil {
				t.Errorf("write the running mark: %v", err)
			}
		})
	}
	t.Run("at the start", func(t *testing.T) {
		f := newDirWaitFixture(t)
		reg := f.attach(t)
		f.writeMark(t, DirectorWaitMark{WaitID: "1-1", ID: reg.ID, AttachedAt: reg.AttachedAt, WatchAccepting: true})
		mustAcquireWatchLock(t, f.dir)
		writeLater(t, f.dir, 150*time.Millisecond)
		p := f.params("d1")
		p.Settle, p.Timeout = 5*time.Second, 200*time.Millisecond
		out := f.runDirectorWait(t, p)
		if out.err == nil || errors.Is(out.err, ErrDirectorWaitNoWatch) || !strings.Contains(out.err.Error(), inboxWaitTimeoutPhrase) {
			t.Fatalf("err = %v; want the timeout", out.err)
		}
		if out.elapsed >= p.Settle {
			t.Errorf("the wait took %v; want the settle to end once the mark said running", out.elapsed)
		}
		wantDirectorWaitEnd(t, f.mark(t), DirectorWaitEndTimeout, true, "")
		if s := f.stderr.String(); s != "" {
			t.Errorf("stderr = %q; want nothing", s)
		}
	})
	t.Run("in a round", func(t *testing.T) {
		f := newDirWaitFixture(t)
		f.attach(t)
		holdRunningWatch(t, f.dir)
		p := f.params("d1")
		p.Settle, p.Timeout = 5*time.Second, dirWaitLimit
		done := f.startDirectorWait(context.Background(), p)
		f.waitForMark(t, "the start with a watch that accepts work", func(m DirectorWaitMark) bool { return m.WatchAccepting })
		if err := os.Remove(WatchAlivePathIn(f.dir)); err != nil {
			t.Fatal(err)
		}
		writeLater(t, f.dir, 150*time.Millisecond)
		time.Sleep(400 * time.Millisecond)
		f.escalate(t, "org-a")
		out := awaitDirectorWait(t, done)
		if out.err != nil || !slices.Equal(waitInboxItemIDs(out.items), []string{"e1"}) {
			t.Fatalf("wait = %v, %v; want e1, with the watch still accepting work", waitInboxItemIDs(out.items), out.err)
		}
		wantDirectorWaitEnd(t, f.mark(t), DirectorWaitEndItems, true, "")
		if s := f.stderr.String(); s != "" {
			t.Errorf("stderr = %q; want nothing", s)
		}
	})
}

// AC5: a watch that starts accepting work while the wait waits gets one
// note: line and an immediate rewrite of the mark; a later stop of it is then
// a change the wait ends for.
func TestWaitInboxForDirector_AWatchThatStartsAcceptingWorkIsNoted(t *testing.T) {
	f := newDirWaitFixture(t)
	f.attach(t)
	p := f.params("d1")
	p.MarkInterval, p.Timeout = time.Hour, dirWaitLimit
	done := f.startDirectorWait(context.Background(), p)
	f.waitForMark(t, "the start without a watch", func(m DirectorWaitMark) bool { return !m.WatchAccepting })
	lock := holdRunningWatch(t, f.dir)
	f.waitForMark(t, "watch_accepting true", func(m DirectorWaitMark) bool { return m.WatchAccepting && m.End == "" })
	_ = lock.Release()
	out := awaitDirectorWait(t, done)
	if !errors.Is(out.err, ErrDirectorWaitNoWatch) {
		t.Fatalf("err = %v; want ErrDirectorWaitNoWatch once the watch that accepted work let go of the lock", out.err)
	}
	stderr := f.stderr.String()
	if strings.Count(stderr, "note: ") != 2 || !strings.Contains(stderr,
		"note: a ralph org watch --all accepts work again on this ledger: watch.lock is held and watch-alive.json says running (pid 4242, written at 2026-10-19T09:00:00Z)\n") {
		t.Errorf("stderr = %q; want the note of no watch and then the note of a watch accepting work again", stderr)
	}
}

// AC4: a registration that changes while the wait waits (replaced, detached,
// stopped, stopped by a hand edit that left it current, attached again under
// the same id) ends the wait at the next rewrite with
// ErrDirectorWaitNotCurrent and end not_current.
func TestWaitInboxForDirector_ARegistrationThatChangesEndsTheWait(t *testing.T) {
	again := checkAttached(DirectorLaunchHerdr, true)
	again.TS = "2026-10-19T08:59:00Z"
	rows := []struct {
		name   string
		setup  func(t *testing.T, f *dirWaitFixture)
		change func(t *testing.T, f *dirWaitFixture)
		detail string
		why    string
	}{
		{"replaced", func(t *testing.T, f *dirWaitFixture) { f.attach(t) }, func(t *testing.T, f *dirWaitFixture) {
			if _, _, err := f.store.Attach(DirectorAttach{Driver: DirectorDriverCodex, Replace: true, Wake: []string{DirectorWakePull}}); err != nil {
				t.Fatal(err)
			}
		}, "the current registration is d2", "the current registration is d2 (driver codex"},
		{"detached", func(t *testing.T, f *dirWaitFixture) { f.attach(t) }, func(t *testing.T, f *dirWaitFixture) {
			if _, err := f.store.Detach(DirectorDetach{ID: "d1"}); err != nil {
				t.Fatal(err)
			}
		}, "no registration is current", "no registration is current"},
		{"stopped", func(t *testing.T, f *dirWaitFixture) { writeDirectorEvents(t, f.store, checkHerdrRunning()...) }, func(t *testing.T, f *dirWaitFixture) {
			if _, err := f.store.MarkStopped("d1", DirectorWho{User: "u", Cwd: "/w"}); err != nil {
				t.Fatal(err)
			}
		}, "no registration is current", "no registration is current"},
		{"stopped by a hand edit that left it current", func(t *testing.T, f *dirWaitFixture) { writeDirectorEvents(t, f.store, checkHerdrRunning()...) },
			func(t *testing.T, f *dirWaitFixture) {
				replaceDirectorEvents(t, f.store, append(checkHerdrRunning(), checkEvent("d1", DirectorEventStopped, nil))...)
			}, "d1 is stopped", "headless stopped) is stopped"},
		{"attached again under the same id", func(t *testing.T, f *dirWaitFixture) {
			writeDirectorEvents(t, f.store, checkAttached(DirectorLaunchHerdr, true))
		},
			func(t *testing.T, f *dirWaitFixture) { replaceDirectorEvents(t, f.store, again) },
			"d1 was attached again", "not the d1 attached 2026-10-18T08:00:00Z that this wait started for"},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			f := newDirWaitFixture(t)
			r.setup(t, f)
			p := f.params("d1")
			p.Timeout = dirWaitLimit
			done := f.startDirectorWait(context.Background(), p)
			f.waitForMark(t, "the start", func(DirectorWaitMark) bool { return true })
			r.change(t, f)
			out := awaitDirectorWait(t, done)
			if !errors.Is(out.err, ErrDirectorWaitNotCurrent) || !strings.Contains(out.err.Error(), r.why) {
				t.Fatalf("err = %v; want ErrDirectorWaitNotCurrent saying %q", out.err, r.why)
			}
			m := f.mark(t)
			wantDirectorWaitEnd(t, m, DirectorWaitEndNotCurrent, false, r.detail)
			if m.Detail != r.detail {
				t.Errorf("detail = %q; want %q", m.Detail, r.detail)
			}
		})
	}
}

// AC2: a done ctx ends the wait with end signal and the cause as the detail,
// and the error wraps the cause.
func TestWaitInboxForDirector_ACancelledContextEndsWithSignal(t *testing.T) {
	f := newDirWaitFixture(t)
	f.attach(t)
	holdRunningWatch(t, f.dir)
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("signal: terminated")
	done := f.startDirectorWait(ctx, f.params("d1"))
	f.waitForMark(t, "the start", func(DirectorWaitMark) bool { return true })
	cancel(cause)
	out := awaitDirectorWait(t, done)
	if !errors.Is(out.err, cause) || !strings.HasPrefix(out.err.Error(), directorWaitErrPrefix+"the wait was stopped: ") {
		t.Fatalf("err = %v; want one that wraps the cause", out.err)
	}
	m := f.mark(t)
	wantDirectorWaitEnd(t, m, DirectorWaitEndSignal, true, "")
	if m.Detail != "signal: terminated" {
		t.Errorf("detail = %q; want the cause", m.Detail)
	}
}

// A ctx done during the start's settle decides nothing about the watch: the
// mark keeps the previous mark's watch_accepting, so the next wait still
// ends for the change.
func TestWaitInboxForDirector_ACancelDuringTheStartSettleKeepsTheCarriedWatch(t *testing.T) {
	f := newDirWaitFixture(t)
	reg := f.attach(t)
	f.writeMark(t, DirectorWaitMark{WaitID: "1-1", ID: reg.ID, AttachedAt: reg.AttachedAt, WatchAccepting: true})
	mustAcquireWatchLock(t, f.dir)
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	p := f.params("d1")
	p.Settle = 5 * time.Second
	done := f.startDirectorWait(ctx, p)
	time.Sleep(100 * time.Millisecond)
	cancel(errors.New("signal: interrupt"))
	out := awaitDirectorWait(t, done)
	if out.elapsed >= p.Settle || errors.Is(out.err, ErrDirectorWaitNoWatch) || !strings.Contains(out.err.Error(), "signal: interrupt") {
		t.Fatalf("wait = %v after %v; want the signal before the settle of %v ended", out.err, out.elapsed, p.Settle)
	}
	wantDirectorWaitEnd(t, f.mark(t), DirectorWaitEndSignal, true, "signal: interrupt")
}

// AC2: an inbox or a director.jsonl that cannot be read ends the wait with
// end error and that error.
func TestWaitInboxForDirector_AReadThatFailsEndsWithError(t *testing.T) {
	t.Run("inbox", func(t *testing.T) {
		f := newDirWaitFixture(t)
		f.attach(t)
		if err := os.MkdirAll(f.o.Inbox.Path(), 0o755); err != nil {
			t.Fatal(err)
		}
		out := f.runDirectorWait(t, f.params("d1"))
		if out.err == nil || !strings.HasPrefix(out.err.Error(), directorWaitErrPrefix+"org: inbox: read ") {
			t.Fatalf("err = %v; want the inbox read with the prefix", out.err)
		}
		wantDirectorWaitEnd(t, f.mark(t), DirectorWaitEndError, false, "org: inbox: read ")
	})
	t.Run("director.jsonl", func(t *testing.T) {
		f := newDirWaitFixture(t)
		f.attach(t)
		p := f.params("d1")
		p.Timeout = dirWaitLimit
		done := f.startDirectorWait(context.Background(), p)
		f.waitForMark(t, "the start", func(DirectorWaitMark) bool { return true })
		// A symlink to a directory renamed over director.jsonl: a read
		// meanwhile finds the file or the directory, never no file.
		target := filepath.Join(t.TempDir(), "a-directory")
		if err := os.Mkdir(target, 0o755); err != nil {
			t.Fatal(err)
		}
		link := f.store.Path() + ".link"
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(link, f.store.Path()); err != nil {
			t.Fatal(err)
		}
		out := awaitDirectorWait(t, done)
		if out.err == nil || errors.Is(out.err, ErrDirectorWaitNotCurrent) || !strings.HasPrefix(out.err.Error(), directorWaitErrPrefix+"org: director: read ") {
			t.Fatalf("err = %v; want the director.jsonl read with the prefix", out.err)
		}
		wantDirectorWaitEnd(t, f.mark(t), DirectorWaitEndError, false, "org: director: read ")
	})
}

// AC5, the text (D3): one for each row, the facts of the lock and the mark
// only, with the command and what happens to the inbox for a watch that
// accepts no work. Two rows copy what a watch leaves: one that ended in a
// panic (finish defers stopAndRelease, which writes stopped and frees the
// lock) and one whose stopped mark could not be written (stopAndRelease
// frees the lock anyway, so the running mark stays). No text names a cause.
func TestDirectorWaitWatchFacts_SaysOnlyWhatTheLockAndTheMarkShow(t *testing.T) {
	stopped := runningMark(4242, dirWaitT0)
	stopped.State, stopped.StoppedAt = WatchAliveStopped, "2026-10-19T09:00:05Z"
	escaped := runningMark(4242, dirWaitT0)
	escaped.TS = "2026-10-19\x1b[2J"
	mark := func(m WatchAlive) func(t *testing.T, dir string) {
		return func(t *testing.T, dir string) { mustWriteWatchAlive(t, dir, m) }
	}
	held := func(m *WatchAlive) func(t *testing.T, dir string) {
		return func(t *testing.T, dir string) {
			mustAcquireWatchLock(t, dir)
			if m != nil {
				mustWriteWatchAlive(t, dir, *m)
			}
		}
	}
	corrupt := func(t *testing.T, dir string) {
		if err := os.WriteFile(WatchAlivePathIn(dir), []byte("{"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old := runningMark(4242, dirWaitT0.Add(-time.Hour))
	stopping := stoppingMark(4242, dirWaitT0)
	once := onceMark(4242, dirWaitT0)
	running := runningMark(4242, dirWaitT0)
	rows := []struct {
		name      string
		setup     func(t *testing.T, dir string)
		state     string
		accepting bool
		want      []string
	}{
		{"a stop whose stopped mark could not be written: lock free, mark running", mark(running), WatchLivenessNotRunning, false,
			[]string{"watch.lock is free but watch-alive.json still says running (pid 4242, written at 2026-10-19T09:00:00Z): the watch ended without writing its stopped mark"}},
		{"lock free, mark stopping", mark(stopping), WatchLivenessNotRunning, false,
			[]string{"watch.lock is free but watch-alive.json still says stopping (pid 4242, written at 2026-10-19T09:00:00Z)"}},
		{"what a panicking watch leaves: lock free, mark stopped", mark(stopped), WatchLivenessNotRunning, false,
			[]string{"watch.lock is free and watch-alive.json says stopped at 2026-10-19T09:00:05Z (pid 4242)"}},
		{"lock free, no mark", func(*testing.T, string) {}, WatchLivenessNotRunning, false,
			[]string{"watch.lock is free and there is no readable watch-alive.json"}},
		{"lock free, a mark that does not parse", corrupt, WatchLivenessNotRunning, false,
			[]string{"watch.lock is free and there is no readable watch-alive.json"}},
		{"held, a running mark older than the window", held(&old), WatchLivenessStale, false,
			[]string{"watch.lock is held, but watch-alive.json has not been rewritten as running since 2026-10-19T08:00:00Z (pid 4242, it says running)", "kill 4242 (the pid in "}},
		{"held, no mark", held(nil), WatchLivenessStale, false,
			[]string{"watch.lock is held, but there is no readable watch-alive.json", "find the process that holds "}},
		{"stopping", held(&stopping), WatchLivenessStopping, false,
			[]string{"the watch (pid 4242) is stopping: watch.lock is held and watch-alive.json says stopping (written at 2026-10-19T09:00:00Z)"}},
		{"a --once run", held(&once), WatchLivenessRunning, false,
			[]string{"the running watch (pid 4242) is a run that ends after its cycles (--once), which accepts no new work"}},
		{"running", held(&running), WatchLivenessRunning, true,
			[]string{"watch.lock is held and watch-alive.json says running (pid 4242, written at 2026-10-19T09:00:00Z)"}},
		{"a hand-edited ts is escaped", mark(escaped), WatchLivenessNotRunning, false,
			[]string{`written at 2026-10-19\x1b[2J)`}},
	}
	hint := "/tmp/ledger dir"
	command := "`" + CommandWithStateDir("ralph org watch --all --detach", hint) + "`"
	inbox := []string{"an item escalated reaches the human at once", "reaches the human at its ack deadline only once a watch runs again"}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			dir := t.TempDir()
			r.setup(t, dir)
			l := ProbeWatch(dir, dirWaitT0, 10)
			if l.State != r.state || WatchAcceptsWork(l) != r.accepting {
				t.Fatalf("ProbeWatch = %s (accepting %v); want %s (accepting %v)", l.State, WatchAcceptsWork(l), r.state, r.accepting)
			}
			text := directorWaitWatchFacts(DirectorWaitParams{StateDir: dir, HintStateDir: hint}, l)
			want := r.want
			if !r.accepting {
				want = append(slices.Clone(want), append([]string{command}, inbox...)...)
			} else if strings.Contains(text, "ralph org watch --all --detach") {
				t.Errorf("text %q names the command for a watch that accepts work", text)
			}
			for _, w := range want {
				if !strings.Contains(text, w) {
					t.Errorf("text %q does not hold %q", text, w)
				}
			}
			lower := strings.ToLower(text)
			for _, w := range dirWaitCauseWords {
				if strings.Contains(lower, w) {
					t.Errorf("text %q names a cause (%q)", text, w)
				}
			}
			if strings.Contains(text, "\x1b") {
				t.Errorf("text %q holds a raw control character", text)
			}
		})
	}

	if text := directorWaitWatchFacts(DirectorWaitParams{StateDir: t.TempDir()}, WatchLiveness{State: WatchLivenessNotRunning}); !strings.Contains(text, "`ralph org watch --all --detach`;") {
		t.Errorf("text without a hint state dir = %q; want the command without --state-dir", text)
	}
}
