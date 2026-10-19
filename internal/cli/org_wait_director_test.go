package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org wait --inbox --director <id>` (plan
// 2026-10-19-org-director-wait, AC1, AC2's signals, and AC6 on the CLI side).
// AC9: every ledger is a t.TempDir named with --state-dir; no test runs
// herdr, osascript, a terminal, gh, or the network. A signal goes only to the
// test process with a channel of the test's own registered first
// (signalSelf), or to a child copy of the test binary the test started. A
// watch is simulated by holding watch.lock and writing watch-alive.json
// (holdWatchLock).

// dirWaitCLILimit bounds every wait a test runs, so one that never ends fails
// instead of hanging the test binary.
const dirWaitCLILimit = 10 * time.Second

// setDirectorWaitIntervals makes the inbox poll, the mark rewrite, the watch
// probe, and the settle of a --director wait tiny for the rest of the test.
func setDirectorWaitIntervals(t *testing.T) {
	t.Helper()
	setInboxPollIntervalOverride(t, 5*time.Millisecond)
	mark, probe, settle := orgDirectorWaitMarkIntervalOverride, orgDirectorWaitWatchProbeIntervalOverride, orgDirectorWaitSettleOverride
	orgDirectorWaitMarkIntervalOverride = 20 * time.Millisecond
	orgDirectorWaitWatchProbeIntervalOverride = 10 * time.Millisecond
	orgDirectorWaitSettleOverride = 50 * time.Millisecond
	t.Cleanup(func() {
		orgDirectorWaitMarkIntervalOverride, orgDirectorWaitWatchProbeIntervalOverride, orgDirectorWaitSettleOverride = mark, probe, settle
	})
}

// reraiseRecorder records each call of orgDirectorWaitReraise with the mark
// on disk at that moment.
type reraiseRecorder struct {
	mu    sync.Mutex
	sigs  []os.Signal
	marks []*org.DirectorWaitMark
}

func (r *reraiseRecorder) calls() ([]os.Signal, []*org.DirectorWaitMark) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.sigs), slices.Clone(r.marks)
}

// recordDirectorWaitReraise swaps orgDirectorWaitReraise for a recorder that
// reads the mark in stateDir when it is called, and restores it at cleanup.
func recordDirectorWaitReraise(t *testing.T, stateDir string) *reraiseRecorder {
	t.Helper()
	rec := &reraiseRecorder{}
	orig := orgDirectorWaitReraise
	orgDirectorWaitReraise = func(sig os.Signal) {
		m, err := org.ReadDirectorWait(stateDir)
		if err != nil {
			t.Errorf("read the mark when the signal was raised again: %v", err)
		}
		rec.mu.Lock()
		defer rec.mu.Unlock()
		rec.sigs = append(rec.sigs, sig)
		rec.marks = append(rec.marks, m)
	}
	t.Cleanup(func() { orgDirectorWaitReraise = orig })
	return rec
}

// attachD1 records d1, a promoted session's registration with driver
// claude, in stateDir, and returns it as director.jsonl holds it.
func attachD1(t *testing.T, stateDir string) org.DirectorEvent {
	t.Helper()
	runInboxOK(t, "director", "attach", "--driver", "claude", "--state-dir", stateDir)
	events := directorEventsIn(t, stateDir)
	if len(events) != 1 || events[0].ID != "d1" {
		t.Fatalf("director.jsonl = %+v; want one attached d1", events)
	}
	return events[0]
}

// directorWaitArgs is `wait --inbox --director <id>` on stateDir, with
// extra flags after it.
func directorWaitArgs(id, stateDir string, extra ...string) []string {
	return append([]string{"wait", "--inbox", "--director", id, "--state-dir", stateDir}, extra...)
}

// readDirectorWaitMark reads director-wait.json in stateDir, which must be
// there.
func readDirectorWaitMark(t *testing.T, stateDir string) org.DirectorWaitMark {
	t.Helper()
	m, err := org.ReadDirectorWait(stateDir)
	if err != nil || m == nil {
		t.Fatalf("ReadDirectorWait(%s) = %+v, %v; want a mark", stateDir, m, err)
	}
	return *m
}

// waitForDirectorWaitMark polls director-wait.json in stateDir until ok
// holds for it, and fails the test when it does not within dirWaitCLILimit.
func waitForDirectorWaitMark(t *testing.T, stateDir, what string, ok func(org.DirectorWaitMark) bool) {
	t.Helper()
	deadline := time.Now().Add(dirWaitCLILimit)
	for {
		m, err := org.ReadDirectorWait(stateDir)
		if err == nil && m != nil && ok(*m) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("director-wait.json never showed %s: last read %+v, %v", what, m, err)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// wantNoDirectorWaitFiles checks that stateDir holds neither
// director-wait.json nor director-wait.lock.
func wantNoDirectorWaitFiles(t *testing.T, stateDir string) {
	t.Helper()
	for _, name := range []string{org.DirectorWaitFileName, "director-wait.lock"} {
		if _, err := os.Lstat(filepath.Join(stateDir, name)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s exists in %s (or cannot be read: %v); want none", name, stateDir, err)
		}
	}
}

// wantDirectorWaitEnd checks that m ended with end, watch_accepting, and a
// detail equal to detail.
func wantDirectorWaitEnd(t *testing.T, m org.DirectorWaitMark, end string, accepting bool, detail string) {
	t.Helper()
	if m.End != end || m.EndedAt == "" || m.WatchAccepting != accepting || m.Detail != detail {
		t.Errorf("mark = %+v; want end %q with ended_at, watch_accepting %v, and the detail %q", m, end, accepting, detail)
	}
}

// dirWaitCLIResult is one run of a command in a goroutine.
type dirWaitCLIResult struct {
	stdout, stderr string
	err            error
}

// startOrgCmd runs `ralph org <args...>` with ctx as its context in a
// goroutine.
func startOrgCmd(ctx context.Context, args ...string) <-chan dirWaitCLIResult {
	done := make(chan dirWaitCLIResult, 1)
	go func() {
		stdout, stderr, err := runOrgCmdStreamsContext(ctx, args...)
		done <- dirWaitCLIResult{stdout, stderr, err}
	}()
	return done
}

// awaitOrgCmd returns the result of a command startOrgCmd started, and fails
// the test when it has not returned within dirWaitCLILimit.
func awaitOrgCmd(t *testing.T, done <-chan dirWaitCLIResult) dirWaitCLIResult {
	t.Helper()
	select {
	case r := <-done:
		return r
	case <-time.After(dirWaitCLILimit):
		t.Fatalf("the command did not return within %v", dirWaitCLILimit)
		return dirWaitCLIResult{}
	}
}

// AC1: --director without --inbox is refused before anything is read or
// written, even next to a seat wait's flags; with --inbox, the refusal of
// --org-id, --seat, and --until comes first, as without --director. Each
// exits 1, prints nothing on stdout, and leaves the state dir uncreated.
func TestOrgWaitDirector_RefusesItsFlagsBeforeReadingAnything(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	for _, tc := range []struct {
		args    []string
		wantSub string
	}{
		{[]string{"wait", "--director", "d1"}, "org: wait: --director needs --inbox"},
		{[]string{"wait", "--director", ""}, "org: wait: --director needs --inbox"},
		{[]string{"wait", "--director", "d1", "--org-id", "org-a", "--seat", "seat-1"}, "org: wait: --director needs --inbox"},
		{[]string{"wait", "--inbox", "--director", "d1", "--seat", "seat-1"}, "org: wait: --inbox cannot be combined with --seat"},
		{[]string{"wait", "--inbox", "--director", "d1", "--until", "idle"}, "org: wait: --inbox cannot be combined with --until"},
		{[]string{"wait", "--inbox", "--director", "d1", "--org-id", "org-a"}, "org: wait: --inbox cannot be combined with --org-id"},
	} {
		args := append(slices.Clone(tc.args), "--timeout-ms", "0", "--state-dir", stateDir)
		stdout, _, _, err := runOrgCmdStreamsWithin(t, 5*time.Second, args...)
		if err == nil || err.Error() != tc.wantSub || ExitCode(err) != 1 {
			t.Errorf("%v: error = %v (exit %d); want %q and exit 1", tc.args, err, ExitCode(err), tc.wantSub)
		}
		if stdout != "" {
			t.Errorf("%v: stdout = %q; want nothing", tc.args, stdout)
		}
	}
	if _, err := os.Stat(stateDir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a refused wait --director created the state dir (stat err %v)", err)
	}
}

// AC1: an id that is not d<N> is refused by the org layer before it reads
// director.jsonl (a directory here, which a read would fail on), with exit
// 1, and nothing is written: no mark, no lock, and a state dir that did not
// exist is not created. A well-formed id is read for, and with no
// registration exits 5.
func TestOrgWaitDirector_RefusesAnIDThatIsNotDN(t *testing.T) {
	for _, id := range []string{"x", "d", "d0"} {
		stateDir := t.TempDir()
		if err := os.MkdirAll(org.DirectorPathIn(stateDir), 0o755); err != nil {
			t.Fatal(err)
		}
		stdout, _, _, err := runOrgCmdStreamsWithin(t, 5*time.Second, directorWaitArgs(id, stateDir, "--timeout-ms", "0")...)
		if want := "org: wait --inbox --director: " + strconv.Quote(id) + " is not a registration id (d<N>, such as d1)"; err == nil || err.Error() != want || ExitCode(err) != 1 {
			t.Errorf("id %q: error = %v (exit %d); want %q and exit 1", id, err, ExitCode(err), want)
		}
		if stdout != "" {
			t.Errorf("id %q: stdout = %q; want nothing", id, stdout)
		}
		wantNoDirectorWaitFiles(t, stateDir)

		missing := filepath.Join(t.TempDir(), "state")
		if _, _, _, err := runOrgCmdStreamsWithin(t, 5*time.Second, directorWaitArgs(id, missing, "--timeout-ms", "0")...); ExitCode(err) != 1 {
			t.Errorf("id %q on a state dir that does not exist: error = %v; want exit 1", id, err)
		}
		if _, err := os.Stat(missing); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("id %q: the refused wait created the state dir (stat err %v)", id, err)
		}
	}

	missing := filepath.Join(t.TempDir(), "state")
	_, _, _, err := runOrgCmdStreamsWithin(t, 5*time.Second, directorWaitArgs("d12", missing, "--timeout-ms", "0")...)
	if !errors.Is(err, org.ErrDirectorWaitNotCurrent) || ExitCode(err) != 5 ||
		err.Error() != "org: wait --inbox --director: d12 is not the current director registration: no registration is current" {
		t.Errorf("d12 with no registration: error = %v (exit %d); want ErrDirectorWaitNotCurrent and exit 5", err, ExitCode(err))
	}
	if _, err := os.Stat(missing); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a wait for d12 with no registration created the state dir (stat err %v)", err)
	}
}

// AC1 and AC4 on the CLI side: a registration that is not current (none,
// another id, stopped) exits 5 without waiting, names the current
// registration, prints nothing on stdout, and writes neither the mark nor its
// lock, nor anything in director.jsonl. A registration replaced while the
// wait runs ends it with exit 5 and end not_current.
func TestOrgWaitDirector_ARegistrationThatIsNotCurrentExits5(t *testing.T) {
	const ts = "2026-10-18T08:00:00Z"
	stoppedLines := func(t *testing.T, stateDir string) {
		ev := func(kind string, set func(*org.DirectorEvent)) string {
			e := org.DirectorEvent{TS: ts, ID: "d1", Event: kind, User: "u", Cwd: "/w"}
			if set != nil {
				set(&e)
			}
			return directorEventLine(t, e)
		}
		writeDirectorLines(t, stateDir,
			ev(org.DirectorEventAttached, func(e *org.DirectorEvent) {
				e.Driver, e.Wake, e.Launch, e.Token, e.Model, e.PermissionMode = "codex", []string{"pull"}, org.DirectorLaunchHerdr, "0123456789ab", "m", org.PermissionModeGuarded
			}),
			ev(org.DirectorEventWorkspaceCreated, func(e *org.DirectorEvent) { e.WorkspaceID = "w-1" }),
			ev(org.DirectorEventPaneCreated, func(e *org.DirectorEvent) { e.PaneID = "p-1" }),
			ev(org.DirectorEventLaunched, nil),
			ev(org.DirectorEventStopped, nil),
		)
	}
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, stateDir string)
		want  string
	}{
		{"no registration", func(*testing.T, string) {}, "d1 is not the current director registration: no registration is current"},
		{"another id", func(t *testing.T, stateDir string) {
			attachD1(t, stateDir)
			runInboxOK(t, "director", "attach", "--driver", "codex", "--replace", "--state-dir", stateDir)
		}, "d1 is not the current director registration: the current registration is d2 (driver codex, attached "},
		{"stopped", stoppedLines, "d1 is not the current director registration: d1 (driver codex, attached " + ts + ", headless stopped) is stopped"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := t.TempDir()
			rec := recordDirectorWaitReraise(t, stateDir)
			tc.setup(t, stateDir)
			before := directorFileOrAbsent(t, stateDir)
			stdout, _, _, err := runOrgCmdStreamsWithin(t, 5*time.Second, directorWaitArgs("d1", stateDir, "--timeout-ms", "0")...)
			if !errors.Is(err, org.ErrDirectorWaitNotCurrent) || ExitCode(err) != 5 || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v (exit %d); want ErrDirectorWaitNotCurrent saying %q and exit 5", err, ExitCode(err), tc.want)
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			wantNoDirectorWaitFiles(t, stateDir)
			if after := directorFileOrAbsent(t, stateDir); after != before {
				t.Errorf("the refused wait changed director.jsonl:\nbefore: %s\nafter:  %s", before, after)
			}
			if sigs, _ := rec.calls(); len(sigs) != 0 {
				t.Errorf("a signal was raised again (%v); want none", sigs)
			}
		})
	}

	t.Run("replaced while it waits", func(t *testing.T) {
		setDirectorWaitIntervals(t)
		stateDir := t.TempDir()
		attachD1(t, stateDir)
		done := startOrgCmd(context.Background(), directorWaitArgs("d1", stateDir, "--timeout-ms", "0")...)
		waitForDirectorWaitMark(t, stateDir, "the start", func(m org.DirectorWaitMark) bool { return m.StartedAt != "" })
		runInboxOK(t, "director", "attach", "--driver", "codex", "--replace", "--state-dir", stateDir)
		r := awaitOrgCmd(t, done)
		if !errors.Is(r.err, org.ErrDirectorWaitNotCurrent) || ExitCode(r.err) != 5 || !strings.Contains(r.err.Error(), "the current registration is d2 (driver codex") {
			t.Fatalf("error = %v (exit %d); want ErrDirectorWaitNotCurrent naming d2 and exit 5", r.err, ExitCode(r.err))
		}
		if r.stdout != "" {
			t.Errorf("stdout = %q; want nothing", r.stdout)
		}
		wantDirectorWaitEnd(t, readDirectorWaitMark(t, stateDir), org.DirectorWaitEndNotCurrent, false, "the current registration is d2")
	})
}

// D2 through the CLI: open items are printed on stdout exactly as a wait
// without --director prints them, the wait exits 0, and its mark is d1's
// with this process's pid and end items. A wait without --director writes no
// mark and no lock. With no watch on the ledger, the note: line goes to the
// command's stderr.
func TestOrgWaitDirector_PrintsOpenItemsAsThePlainWaitDoes(t *testing.T) {
	setDirectorWaitIntervals(t)
	stateDir := t.TempDir()
	rec := recordDirectorWaitReraise(t, stateDir)
	reg := attachD1(t, stateDir)
	escalateOK(t, stateDir, "org-a", inboxQuestion("which plan?\nmore detail"))

	plain, _, _, err := runOrgCmdStreamsWithin(t, 5*time.Second, "wait", "--inbox", "--timeout-ms", "0", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("wait --inbox: %v", err)
	}
	if want := "e1\torg-a\tQUESTION\twhich plan?\n"; plain != want {
		t.Fatalf("wait --inbox stdout = %q; want %q", plain, want)
	}
	wantNoDirectorWaitFiles(t, stateDir)

	stdout, stderr, _, err := runOrgCmdStreamsWithin(t, 5*time.Second, directorWaitArgs("d1", stateDir, "--timeout-ms", "0")...)
	if err != nil || ExitCode(err) != 0 {
		t.Fatalf("wait --inbox --director d1: %v", err)
	}
	if stdout != plain {
		t.Errorf("stdout = %q; want the plain wait's %q", stdout, plain)
	}
	if !strings.Contains(stderr, "note: no ralph org watch --all accepts work on this ledger, and this wait keeps waiting: watch.lock is free and there is no readable watch-alive.json") {
		t.Errorf("stderr = %q; want the note: line about no watch", stderr)
	}
	m := readDirectorWaitMark(t, stateDir)
	if pid := os.Getpid(); m.ID != "d1" || m.AttachedAt != reg.TS || m.PID != pid || !strings.HasPrefix(m.WaitID, strconv.Itoa(pid)+"-") || m.StartedAt == "" || m.SeenAt == "" {
		t.Errorf("mark = %+v; want d1 attached at %s, pid %d, and a wait_id that starts with it", m, reg.TS, pid)
	}
	wantDirectorWaitEnd(t, m, org.DirectorWaitEndItems, false, "1 open inbox item")
	if sigs, _ := rec.calls(); len(sigs) != 0 {
		t.Errorf("a signal was raised again (%v); want none", sigs)
	}
}

// D2: no open item within --timeout-ms exits 1 with the phrase the /director
// skill matches, and the mark says end timeout.
func TestOrgWaitDirector_TimesOutWithThePhrase(t *testing.T) {
	setDirectorWaitIntervals(t)
	stateDir := t.TempDir()
	attachD1(t, stateDir)
	stdout, _, elapsed, err := runOrgCmdStreamsWithin(t, 5*time.Second, directorWaitArgs("d1", stateDir, "--timeout-ms", "200")...)
	if err == nil || ExitCode(err) != 1 || !strings.Contains(err.Error(), "no open inbox item arrived within 200 ms") {
		t.Fatalf("error = %v (exit %d); want the timeout phrase after 200 ms and exit 1", err, ExitCode(err))
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if elapsed < 200*time.Millisecond {
		t.Errorf("the wait gave up after %v, before the 200ms timeout", elapsed)
	}
	wantDirectorWaitEnd(t, readDirectorWaitMark(t, stateDir), org.DirectorWaitEndTimeout, false, "no open inbox item within 200 ms")
}

// D3 and D4 through the CLI: a watch that accepted work and ends without its
// stopped mark (the lock released, the running mark left) exits 4 with
// nothing on stdout. The error says what the lock and the mark show and
// names the command that starts a watch with the --state-dir the wait was
// given; the mark says end watch with watch_accepting false.
func TestOrgWaitDirector_AWatchThatStopsAcceptingWorkExits4(t *testing.T) {
	setDirectorWaitIntervals(t)
	stateDir := t.TempDir()
	release := holdWatchLock(t, stateDir, runningMark(4242, time.Now()))
	attachD1(t, stateDir)
	done := startOrgCmd(context.Background(), directorWaitArgs("d1", stateDir, "--timeout-ms", "10000")...)
	waitForDirectorWaitMark(t, stateDir, "a start with a watch that accepts work", func(m org.DirectorWaitMark) bool { return m.WatchAccepting })
	release()
	r := awaitOrgCmd(t, done)
	if !errors.Is(r.err, org.ErrDirectorWaitNoWatch) || ExitCode(r.err) != 4 {
		t.Fatalf("error = %v (exit %d); want ErrDirectorWaitNoWatch and exit 4", r.err, ExitCode(r.err))
	}
	for _, want := range []string{
		"org: wait --inbox --director: no watch accepts work: a ralph org watch --all accepted work when this wait last probed it, and none does now: ",
		"watch.lock is free but watch-alive.json still says running (pid 4242, written at ",
		"the watch ended without writing its stopped mark",
		"the command that starts one is `ralph org watch --all --detach --state-dir '" + stateDir + "'`",
	} {
		if !strings.Contains(r.err.Error(), want) {
			t.Errorf("error = %q; want it to hold %q", r.err, want)
		}
	}
	if r.stdout != "" {
		t.Errorf("stdout = %q; want nothing", r.stdout)
	}
	if strings.Contains(r.stderr, "note: ") {
		t.Errorf("stderr = %q; want no note: line", r.stderr)
	}
	wantDirectorWaitEnd(t, readDirectorWaitMark(t, stateDir), org.DirectorWaitEndWatch, false, "no watch accepts work: watch not_running, pid 4242")
}

// AC2: when the command's context ends with no signal, the wait writes
// end=signal with the cause "context canceled" and returns its error (exit
// 1); no signal is raised again.
func TestOrgWaitDirector_AContextThatEndsWritesSignalAndRaisesNothing(t *testing.T) {
	setDirectorWaitIntervals(t)
	stateDir := t.TempDir()
	rec := recordDirectorWaitReraise(t, stateDir)
	attachD1(t, stateDir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := startOrgCmd(ctx, directorWaitArgs("d1", stateDir, "--timeout-ms", "0")...)
	waitForDirectorWaitMark(t, stateDir, "the start", func(m org.DirectorWaitMark) bool { return m.StartedAt != "" })
	cancel()
	r := awaitOrgCmd(t, done)
	if r.err == nil || ExitCode(r.err) != 1 || r.err.Error() != "org: wait --inbox --director: the wait was stopped: context canceled" {
		t.Fatalf("error = %v (exit %d); want the stop with context canceled and exit 1", r.err, ExitCode(r.err))
	}
	if r.stdout != "" {
		t.Errorf("stdout = %q; want nothing", r.stdout)
	}
	wantDirectorWaitEnd(t, readDirectorWaitMark(t, stateDir), org.DirectorWaitEndSignal, false, "context canceled")
	if sigs, _ := rec.calls(); len(sigs) != 0 {
		t.Errorf("a signal was raised again (%v); want none", sigs)
	}
}

// AC2: SIGINT, SIGTERM, or SIGHUP to the ralph process ends the wait; the
// mark says end=signal with the cause that names the signal, and only then
// is the same signal raised again (recorded here in place of the real one,
// which would end the test binary). When the process is still alive after
// that, the wait's error is returned (exit 1).
func TestOrgWaitDirector_ASignalWritesItsEndThenRaisesItAgain(t *testing.T) {
	for _, tc := range []struct {
		sig   syscall.Signal
		cause string
	}{
		{syscall.SIGINT, "interrupt signal received"},
		{syscall.SIGTERM, "terminated signal received"},
		{syscall.SIGHUP, "hangup signal received"},
	} {
		t.Run(tc.sig.String(), func(t *testing.T) {
			setDirectorWaitIntervals(t)
			stateDir := t.TempDir()
			rec := recordDirectorWaitReraise(t, stateDir)
			attachD1(t, stateDir)
			done := startOrgCmd(context.Background(), directorWaitArgs("d1", stateDir, "--timeout-ms", "0")...)
			// The mark is written after the signals were registered.
			waitForDirectorWaitMark(t, stateDir, "the start", func(m org.DirectorWaitMark) bool { return m.StartedAt != "" })
			signalSelf(tc.sig)(t, nil)
			r := awaitOrgCmd(t, done)
			if r.err == nil || ExitCode(r.err) != 1 || r.err.Error() != "org: wait --inbox --director: the wait was stopped: "+tc.cause {
				t.Fatalf("error = %v (exit %d); want the stop with %q and exit 1", r.err, ExitCode(r.err), tc.cause)
			}
			if r.stdout != "" {
				t.Errorf("stdout = %q; want nothing", r.stdout)
			}
			wantDirectorWaitEnd(t, readDirectorWaitMark(t, stateDir), org.DirectorWaitEndSignal, false, tc.cause)
			sigs, marks := rec.calls()
			if !slices.Equal(sigs, []os.Signal{tc.sig}) {
				t.Fatalf("signals raised again = %v; want only %v", sigs, tc.sig)
			}
			if m := marks[0]; m == nil || m.End != org.DirectorWaitEndSignal || m.Detail != tc.cause {
				t.Errorf("the mark when %v was raised again = %+v; want end signal with the detail %q already written", tc.sig, m, tc.cause)
			}
		})
	}
}

// AC6: a wait without --director has no signal handling of its own: a
// SIGTERM the test catches does not end it, it times out as before, raises
// nothing again, and writes no mark.
func TestOrgWaitInbox_WithoutDirectorHandlesNoSignal(t *testing.T) {
	setInboxPollIntervalOverride(t, 5*time.Millisecond)
	stateDir := t.TempDir()
	rec := recordDirectorWaitReraise(t, stateDir)
	attachD1(t, stateDir)
	done := startOrgCmd(context.Background(), "wait", "--inbox", "--timeout-ms", "400", "--state-dir", stateDir)
	time.Sleep(100 * time.Millisecond)
	signalSelf(syscall.SIGTERM)(t, nil)
	r := awaitOrgCmd(t, done)
	if r.err == nil || ExitCode(r.err) != 1 || r.err.Error() != "org: wait --inbox: no open inbox item arrived within 400 ms" {
		t.Fatalf("error = %v (exit %d); want the plain timeout and exit 1", r.err, ExitCode(r.err))
	}
	if sigs, _ := rec.calls(); len(sigs) != 0 {
		t.Errorf("a signal was raised again (%v); want none", sigs)
	}
	wantNoDirectorWaitFiles(t, stateDir)
}

// holdDirectorWaitLock takes director-wait.lock in stateDir, as a wait
// writing its mark does, until the returned unlock runs (or the cleanup).
func holdDirectorWaitLock(t *testing.T, stateDir string) (unlock func()) {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(stateDir, "director-wait.lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		t.Fatalf("lock director-wait.lock: %v", err)
	}
	var once sync.Once
	unlock = func() {
		once.Do(func() {
			_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			_ = f.Close()
		})
	}
	t.Cleanup(unlock)
	return unlock
}

// dirWaitSignalChildEnv makes TestOrgWaitDirector_ASignalEndsTheProcessByIt
// run its child side: the value is the state dir to wait on.
const dirWaitSignalChildEnv = "RALPH_TEST_ORG_WAIT_DIRECTOR_SIGNAL_CHILD"

// AC2 with the real re-raise, in a child copy of the test binary that runs
// `ralph org wait --inbox --director d1`: SIGTERM to it ends it by SIGTERM
// once the mark says end=signal with "terminated signal received". While the
// test holds director-wait.lock, so the child cannot write its end yet, a
// second SIGTERM does not end it: it ends by SIGTERM only after the lock is
// released and the end is written. A child started with SIGHUP ignored (as
// nohup starts a process) still ends the wait on SIGHUP and writes its end,
// but the signal raised again is ignored, so after the bounded grace the
// command returns the wait's error, which is exit 1.
func TestOrgWaitDirector_ASignalEndsTheProcessByIt(t *testing.T) {
	if dir := os.Getenv(dirWaitSignalChildEnv); dir != "" {
		root := NewRootCmd()
		root.SetArgs(append([]string{"org"}, directorWaitArgs("d1", dir, "--timeout-ms", "0")...))
		err := root.Execute()
		fmt.Printf("returned: exit %d: %v\n", ExitCode(err), err)
		return
	}

	for _, tc := range []struct {
		name     string
		sig      syscall.Signal
		cause    string
		ignored  bool
		holdLock bool
	}{
		{"SIGTERM", syscall.SIGTERM, "terminated signal received", false, false},
		{"a second SIGTERM while the end waits for the lock", syscall.SIGTERM, "terminated signal received", false, true},
		{"SIGHUP ignored at the start", syscall.SIGHUP, "hangup signal received", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := t.TempDir()
			attachD1(t, stateDir)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			args := []string{"-test.run=^TestOrgWaitDirector_ASignalEndsTheProcessByIt$", "-test.count=1"}
			child := exec.CommandContext(ctx, os.Args[0], args...)
			if tc.ignored {
				// sh sets SIGHUP to ignored, which exec keeps for the
				// test binary.
				child = exec.CommandContext(ctx, "sh", append([]string{"-c", `trap '' HUP; exec "$0" "$@"`, os.Args[0]}, args...)...)
			}
			child.Env = append(os.Environ(), dirWaitSignalChildEnv+"="+stateDir)
			var out bytes.Buffer
			child.Stdout, child.Stderr = &out, &out
			if err := child.Start(); err != nil {
				t.Fatalf("start the child: %v", err)
			}
			// waitErr is set before exited is closed. A child that is still
			// running at the end of the test (it failed) is killed.
			var waitErr error
			exited := make(chan struct{})
			go func() {
				waitErr = child.Wait()
				close(exited)
			}()
			t.Cleanup(func() {
				_ = child.Process.Kill()
				<-exited
			})
			waitForDirectorWaitMark(t, stateDir, "the child's start", func(m org.DirectorWaitMark) bool { return m.StartedAt != "" })
			var unlock func()
			if tc.holdLock {
				unlock = holdDirectorWaitLock(t, stateDir)
			}
			if err := child.Process.Signal(tc.sig); err != nil {
				t.Fatalf("send %v to the child: %v", tc.sig, err)
			}
			if tc.holdLock {
				time.Sleep(300 * time.Millisecond)
				if err := child.Process.Signal(tc.sig); err != nil {
					t.Fatalf("send the second %v to the child: %v", tc.sig, err)
				}
				select {
				case <-exited:
					t.Fatalf("the child ended (%v) before it could write its end\n%s", waitErr, out.String())
				case <-time.After(300 * time.Millisecond):
				}
				if m := readDirectorWaitMark(t, stateDir); m.End != "" {
					t.Fatalf("mark = %+v while the test held director-wait.lock; want no end yet", m)
				}
				unlock()
			}
			select {
			case <-exited:
			case <-time.After(dirWaitCLILimit):
				t.Fatalf("the child did not end within %v of the signal\n%s", dirWaitCLILimit, out.String())
			}
			err := waitErr

			if tc.ignored {
				if err != nil {
					t.Fatalf("the child with SIGHUP ignored = %v; want it to return and exit 0\n%s", err, out.String())
				}
				if want := "returned: exit 1: org: wait --inbox --director: the wait was stopped: " + tc.cause + "\n"; !strings.Contains(out.String(), want) {
					t.Errorf("the child printed:\n%s\nwant the line %q", out.String(), want)
				}
			} else {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("the child = %v; want it ended by %v\n%s", err, tc.sig, out.String())
				}
				ws, ok := exitErr.Sys().(syscall.WaitStatus)
				if !ok || !ws.Signaled() || ws.Signal() != tc.sig {
					t.Fatalf("the child ended with %v; want it ended by %v\n%s", exitErr, tc.sig, out.String())
				}
				if strings.Contains(out.String(), "returned:") {
					t.Errorf("the child returned from the command:\n%s\nwant it ended by %v first", out.String(), tc.sig)
				}
			}
			m := readDirectorWaitMark(t, stateDir)
			wantDirectorWaitEnd(t, m, org.DirectorWaitEndSignal, false, tc.cause)
			if m.PID != child.Process.Pid {
				t.Errorf("mark pid = %d; want the child's %d", m.PID, child.Process.Pid)
			}
		})
	}
}
