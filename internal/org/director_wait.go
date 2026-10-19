package org

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// The director's inbox wait (stage 8b-2b, plan
// docs/plans/active/2026-10-19-org-director-wait.md, D1 to D3):
// WaitInboxForDirector is the org layer of `ralph org wait --inbox --director
// <id>`. It returns the open inbox items as WaitInbox does, and besides:
//
//   - it waits only while the current director registration is <id> and not
//     stopped, read before it starts and again every
//     DirectorWaitMarkInterval (D1), so the wait of a session that
//     `attach --replace` replaced ends by itself (ErrDirectorWaitNotCurrent)
//     at the next such read;
//   - it writes its own mark, director-wait.json, under director-wait.lock
//     (D2): when it started, when it last wrote, the watch it last saw, and
//     how it ended. A promoted session's registration tells nothing about
//     its session, and this mark is what `ralph status` and `ralph org
//     director status` show next to it;
//   - it probes the cross-org watch (D3) and ends with ErrDirectorWaitNoWatch
//     when a watch that accepted work no longer does, so the director can
//     tell the human. Neither the wait nor the director starts a watch: the
//     watch's mark cannot tell a watch the human stopped from one that ended
//     on its own, so the text says only what the lock and the mark show.
//
// A wait without --director (WaitInbox) does none of this. The mark is read
// by status, and by the next wait of the same registration for the
// watch_accepting it carries over (D3); nothing reads it to start or stop a
// watch or a session. It says what the wait wrote, not that the director is
// alive.

// File names of the director's wait, within the org state dir.
const (
	DirectorWaitFileName = "director-wait.json"
	// directorWaitLockFile is the lock every write of director-wait.json
	// takes (withFileLock), as director.lock is for director.jsonl. A
	// wait refused before it starts creates neither file.
	directorWaitLockFile = "director-wait.lock"
)

// How a director's wait ended (DirectorWaitMark.End), one for each end of
// D2. The CLI maps them to exit codes: items 0, timeout 1, watch 4 and
// not_current 5 (ExitCode in internal/cli/exitcode.go, by the error that
// wraps ErrDirectorWaitNoWatch or ErrDirectorWaitNotCurrent), error 1, and
// signal the signal's own (runOrgWaitInboxDirector raises the signal again).
const (
	// DirectorWaitEndItems: it returned the open inbox items.
	DirectorWaitEndItems = "items"
	// DirectorWaitEndTimeout: no open item arrived within the timeout.
	DirectorWaitEndTimeout = "timeout"
	// DirectorWaitEndWatch: a watch that accepted work no longer does
	// (ErrDirectorWaitNoWatch).
	DirectorWaitEndWatch = "watch"
	// DirectorWaitEndNotCurrent: the registration it waited for is no longer
	// current (ErrDirectorWaitNotCurrent).
	DirectorWaitEndNotCurrent = "not_current"
	// DirectorWaitEndError: the inbox or director.jsonl could not be read.
	DirectorWaitEndError = "error"
	// DirectorWaitEndSignal: its context was cancelled, which the CLI does on
	// SIGINT, SIGTERM, and SIGHUP. A SIGKILL writes nothing, so the mark of a
	// wait killed that way keeps its last seen_at and no end.
	DirectorWaitEndSignal = "signal"
)

// The intervals of a director's wait (D2 and D3), which
// DirectorWaitParams.MarkInterval and WatchProbeInterval override.
const (
	// DirectorWaitMarkInterval is how often the wait reads director.jsonl
	// again and rewrites its mark's seen_at. The readers of the mark use it
	// too: `ralph status` and `ralph org director status` say that a running
	// wait has not written its mark since when its seen_at is more than two
	// of these old (D5).
	DirectorWaitMarkInterval = 30 * time.Second
	// directorWaitWatchProbeInterval is how often the wait probes the watch.
	directorWaitWatchProbeInterval = 5 * time.Second
)

// directorWaitErrPrefix starts every error of WaitInboxForDirector, as "org:
// wait --inbox: " starts WaitInbox's.
const directorWaitErrPrefix = "org: wait --inbox --director: "

// directorWaitNoticePrefix starts the stderr lines about the mark (sayLost,
// sayWriteFailed); the /director skill tells these lines apart by it.
const directorWaitNoticePrefix = "wait --inbox --director: "

// ErrDirectorWaitNoWatch is wrapped by WaitInboxForDirector's error when a
// cross-org watch that accepted work no longer does and no inbox item is open
// (D3). The CLI exits 4 on it (D4).
var ErrDirectorWaitNoWatch = errors.New("no watch accepts work")

// ErrDirectorWaitNotCurrent is wrapped by WaitInboxForDirector's error when
// the registration it was given is not the current one, or not any more: no
// registration is current, another one is, the current one is stopped, or it
// was attached again since the wait started (D1). The CLI exits 5 on it (D4).
var ErrDirectorWaitNotCurrent = errors.New("not the current director registration")

// DirectorWaitMark is the mark, director-wait.json (D2). WaitID tells the
// waits apart: the pid and the start time in Unix nanoseconds, joined by "-"
// (not a random hex, which a secret scan can flag). ID and AttachedAt are
// those of the registration the wait was for, so a reader goes by
// DirectorWaitFor; PID is the wait's process. StartedAt is when it started
// and SeenAt when it last wrote the mark: at its start, every
// DirectorWaitMarkInterval, at a change of the watch it saw, and at its end.
// WatchAccepting is whether the watch accepted work when the wait last
// probed it, which the next wait of the same registration carries over (D3),
// with one exception: a change from accepting to not accepting is written
// only by the end that reports it (end watch). Until then, even when open
// items or a signal end the wait first, it stays true, so the next wait of
// the same registration ends for the change instead of losing it.
// EndedAt, End (a DirectorWaitEnd* constant), and Detail, a short reason
// escaped for a terminal (PrintableInboxText), are set once it ended. The
// times are UTC RFC3339 in whole seconds. The values are read from a file, so
// a reader escapes them where it prints them, as it does the registration's.
type DirectorWaitMark struct {
	WaitID         string `json:"wait_id"`
	ID             string `json:"id"`
	AttachedAt     string `json:"attached_at"`
	PID            int    `json:"pid"`
	StartedAt      string `json:"started_at"`
	SeenAt         string `json:"seen_at"`
	WatchAccepting bool   `json:"watch_accepting"`
	EndedAt        string `json:"ended_at,omitempty"`
	End            string `json:"end,omitempty"`
	Detail         string `json:"detail,omitempty"`
}

// DirectorWaitPathIn returns the director-wait.json path within an
// already-resolved org state dir (see DirectorAlivePathIn).
func DirectorWaitPathIn(stateDir string) string {
	return filepath.Join(stateDir, DirectorWaitFileName)
}

// ReadDirectorWait reads the mark in stateDir. A missing mark (or state dir)
// is nil with a nil error: no director's wait has written one on this ledger
// yet. A mark that cannot be read or does not parse is an error. The mark
// returned may be of any registration; pass it through DirectorWaitFor.
func ReadDirectorWait(stateDir string) (*DirectorWaitMark, error) {
	path := DirectorWaitPathIn(stateDir)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("org: director: read %s: %w", path, err)
	}
	var mark DirectorWaitMark
	if err := json.Unmarshal(data, &mark); err != nil {
		return nil, fmt.Errorf("org: director: parse %s: %w", path, err)
	}
	return &mark, nil
}

// DirectorWaitFor returns mark when it is the mark of a wait for reg: both
// are set, and the mark's id and attached_at equal reg's ID and AttachedAt,
// as DirectorAliveFor does for the check's mark. Otherwise it returns nil. It
// compares the values only; it does not check them.
func DirectorWaitFor(mark *DirectorWaitMark, reg *DirectorRegistration) *DirectorWaitMark {
	if mark == nil || reg == nil || mark.ID != reg.ID || mark.AttachedAt != reg.AttachedAt {
		return nil
	}
	return mark
}

// DirectorWaitParams describes one WaitInboxForDirector call. The fields
// after Stderr are seams whose zero value is the default.
type DirectorWaitParams struct {
	// ID is the registration the wait is for, d<N>.
	ID string
	// StateDir is the resolved org state dir: director.jsonl, the watch's
	// lock and mark, and the wait's own mark are read and written there.
	StateDir string
	// HintStateDir is the --state-dir that the `ralph org watch` command in
	// the text repeats (CommandWithStateDir); the CLI sets it only when the
	// ledger was named with --state-dir, as for Org.InboxHintStateDir.
	HintStateDir string
	// Timeout bounds the wait; <= 0 waits without a bound.
	Timeout time.Duration
	// Stderr gets the note: lines and the lines about the mark; nil means
	// os.Stderr.
	Stderr io.Writer
	// MarkInterval is how often director.jsonl is read again and the mark's
	// seen_at rewritten (DirectorWaitMarkInterval when zero).
	MarkInterval time.Duration
	// WatchProbeInterval is how often the watch is probed
	// (directorWaitWatchProbeInterval when zero).
	WatchProbeInterval time.Duration
	// Settle is how long a stale watch is probed again before it counts as
	// not accepting work (DefaultWatchGateSettle when zero), as for the
	// watch gate. It reads the real clock.
	Settle time.Duration
	// Now is the clock of the mark's times and of ProbeWatch; nil means
	// time.Now.
	Now func() time.Time
	// PID is the pid the mark records (os.Getpid() when zero).
	PID int
}

// CheckDirectorWaitID returns nil for an id of the form d<N>, written the
// way a registration id is (directorIDNumber), and otherwise the error
// WaitInboxForDirector refuses that id with. It reads nothing, so the CLI
// calls it before it resolves the state dir or loads ralph.toml (AC1).
func CheckDirectorWaitID(id string) error {
	if _, ok := directorIDNumber(id); !ok {
		return fmt.Errorf("%s%q is not a registration id (d<N>, such as d1)", directorWaitErrPrefix, id)
	}
	return nil
}

// WaitInboxForDirector is `ralph org wait --inbox --director <id>` (plan
// 2026-10-19-org-director-wait): it returns the open inbox items in ID order,
// as WaitInbox does, reading the inbox every Org.InboxPollInterval
// (defaultInboxPollInterval when unset), and while it waits it keeps its mark
// and probes the watch.
//
// Before it reads anything it refuses an id that is not of the form d<N>
// with a plain error (CheckDirectorWaitID). It then reads director.jsonl:
// the current registration must be p.ID and not stopped (a promoted
// session's registration has no state; a headless one may be starting or
// running). Otherwise the error wraps ErrDirectorWaitNotCurrent and names the
// current registration or says there is none, and nothing is written: no
// mark, no lock file. A director.jsonl that cannot be read is a plain error,
// with nothing written either (D1).
//
// Then it probes the watch, and under director-wait.lock reads director.jsonl
// again before it writes the mark: the registration must still be the one
// the first read found, its attached_at included. A registration replaced,
// detached, stopped, or attached again between the two reads (for example
// while the probe settles a stale watch) ends the wait with
// ErrDirectorWaitNotCurrent, and a director.jsonl that cannot be read with a
// plain error; neither writes the mark, so the mark of the wait that is now
// current stays as it was (the lock file may exist then). Otherwise it reads
// the mark left before it and replaces it with its own, so it is the mark's
// owner (D2).
// Every later write of the mark takes the lock, reads the mark, and writes
// only while the mark still holds this wait's wait_id. Once another wait has
// replaced it, this one says so on p.Stderr once and never writes the mark
// again, but waits and returns as before; a write that fails is said once
// too, and the wait goes on.
//
// Each round reads the inbox first: open items end the wait with them (end
// items). Then, every p.WatchProbeInterval, it probes the watch (D3):
// WatchAcceptsWork, with a stale watch probed again until p.Settle has
// passed, so a watch that took watch.lock and has not written its first mark
// yet is not read as one that stopped. A watch that accepted work and no
// longer does ends the wait with ErrDirectorWaitNoWatch (end watch), but only
// after one more read of the inbox finds no open item: open items win.
// "Accepted" is what this wait saw at its last probe, or, at its start, the
// previous mark's watch_accepting when that mark is of the same registration
// (DirectorWaitFor), so a watch that stopped between two waits is caught by
// the second. When open items (or a signal) win over such a change, the mark
// keeps watch_accepting true (DirectorWaitMark), so the next wait ends for
// it. A wait that finds no watch accepting work at its start, with nothing
// to carry over, writes one note: line and waits on; a watch that starts
// accepting work later gets one note: line and an immediate rewrite of the
// mark. The text of the error and of the note says only what watch.lock and
// watch-alive.json show (directorWaitWatchFacts). Every p.MarkInterval it
// reads director.jsonl again: a registration that is no longer current
// (another id, none, stopped, or a new attached_at) ends the wait with
// ErrDirectorWaitNotCurrent (end not_current); otherwise it rewrites the
// mark's seen_at.
//
// The other ends: the timeout (end timeout) is an error that holds
// inboxWaitTimeoutPhrase, after one more read of the inbox, as WaitInbox's;
// a failed read of the inbox or of director.jsonl (end error) returns that
// error; and a done ctx (end signal, with context.Cause(ctx) as the detail)
// returns an error that wraps the cause. Each end is written to the mark
// before the wait returns, while it is the owner.
func (o *Org) WaitInboxForDirector(ctx context.Context, p DirectorWaitParams) ([]InboxItem, error) {
	if err := CheckDirectorWaitID(p.ID); err != nil {
		return nil, err
	}
	if o.Inbox == nil {
		return nil, errors.New(directorWaitErrPrefix + "no inbox store is configured")
	}
	if p.StateDir == "" {
		return nil, errors.New(directorWaitErrPrefix + "no org state dir was given")
	}
	w := newDirectorWaitRun(o, p)
	res, err := NewDirectorStore(w.p.StateDir).Read()
	if err != nil {
		return nil, fmt.Errorf("%s%w", directorWaitErrPrefix, err)
	}
	if _, err := directorWaitCheckCurrent(w.p.ID, nil, res.Current); err != nil {
		return nil, err
	}
	w.reg = *res.Current
	if err := w.begin(ctx); err != nil {
		return nil, err
	}
	return w.loop(ctx)
}

// directorWaitCheckCurrent checks that cur, the current registration (nil
// for none), is the one a wait for id may wait for (D1): cur is id and not
// stopped, and, once the wait has started for started, has its attached_at
// (started is nil before the start). It returns a nil error when it is.
// Otherwise the error wraps ErrDirectorWaitNotCurrent and names cur
// (describe, which escapes its attach time), and detail is the short reason
// for the mark.
func directorWaitCheckCurrent(id string, started, cur *DirectorRegistration) (detail string, err error) {
	var why string
	switch {
	case cur == nil:
		detail = "no registration is current"
		why = detail
	case cur.ID != id:
		detail = "the current registration is " + cur.ID
		why = "the current registration is " + cur.describe()
	case cur.State == DirectorStateStopped:
		detail = cur.ID + " is stopped"
		why = cur.describe() + " is stopped"
	case started != nil && cur.AttachedAt != started.AttachedAt:
		detail = cur.ID + " was attached again"
		why = fmt.Sprintf("the current registration is %s, not the %s attached %s that this wait started for",
			cur.describe(), id, PrintableInboxText(started.AttachedAt, false))
	default:
		return "", nil
	}
	return detail, fmt.Errorf("%s%s is %w: %s", directorWaitErrPrefix, id, ErrDirectorWaitNotCurrent, why)
}

// directorWaitRun is one WaitInboxForDirector call: its parameters with the
// defaults filled in, the registration it waits for, its mark as it last
// wrote it (or tried to), and what it saw and said. One goroutine runs it.
type directorWaitRun struct {
	o      *Org
	p      DirectorWaitParams
	stderr io.Writer
	reg    DirectorRegistration
	mark   DirectorWaitMark
	// prevWaitID is the wait_id of the mark the start read under the lock
	// ("" for none or an unreadable one). Until a write of this wait lands
	// (written), a later write may still replace that mark.
	prevWaitID string
	written    bool
	// lost is set once another wait's mark replaced this one's; this wait
	// writes the mark no more.
	lost bool
	// writeFailedSaid is set once a failed write was said on stderr.
	writeFailedSaid bool
	// accepting is what the mark's watch_accepting says: whether the watch
	// accepted work at the last probe that settled, or the value carried
	// over from the previous mark. A change to not accepting leaves it true
	// until endNoWatch reports the change.
	accepting bool
	// noWatch is set when a watch that accepted work no longer does; the
	// next round ends the wait unless the inbox has open items. noWatchSince
	// says when it was seen accepting work, for the error.
	noWatch      bool
	noWatchSince string
	// watch is what the last probe found.
	watch WatchLiveness
}

func newDirectorWaitRun(o *Org, p DirectorWaitParams) *directorWaitRun {
	if p.MarkInterval <= 0 {
		p.MarkInterval = DirectorWaitMarkInterval
	}
	if p.WatchProbeInterval <= 0 {
		p.WatchProbeInterval = directorWaitWatchProbeInterval
	}
	if p.Settle <= 0 {
		p.Settle = DefaultWatchGateSettle
	}
	if p.Now == nil {
		p.Now = time.Now
	}
	if p.PID <= 0 {
		p.PID = os.Getpid()
	}
	return &directorWaitRun{o: o, p: p, stderr: bannerOrStderr(p.Stderr)}
}

// begin starts the wait (D2 and D3): it probes the watch, settling a stale
// one, then under director-wait.lock reads director.jsonl again and checks
// the registration as refresh does (directorWaitCheckCurrent with w.reg), so
// a registration replaced since WaitInboxForDirector's first read, while the
// probe ran, is caught before the write. Then it reads the previous mark and
// writes its own with what it found, which makes it the owner. A registration
// that is no longer current returns the error that wraps
// ErrDirectorWaitNotCurrent, and a director.jsonl that cannot be read a plain
// error; either way begin writes no mark, no end, and no note: line, and the
// mark of the wait that is now current stays as it was. The probe runs before
// the lock, so the lock is held for the reads and the write only, as for
// every later write. A probe that ctx cut short in its settle decides
// nothing: the wait carries over the previous mark's watch_accepting, and its
// loop ends at once for ctx. A watch the previous mark saw accepting work
// that accepts none now is a change for the first round to report, so the
// mark keeps watch_accepting true until then (DirectorWaitMark). When the
// lock cannot be taken, the previous mark is read without it, the failed
// write is said once, and the wait goes on: the first refresh reads the
// registration again.
func (w *directorWaitRun) begin(ctx context.Context) error {
	l, settled := w.probe(ctx, true)
	w.watch = l
	observed := WatchAcceptsWork(l)
	start := w.p.Now()
	w.mark = DirectorWaitMark{
		WaitID: strconv.Itoa(w.p.PID) + "-" + strconv.FormatInt(start.UnixNano(), 10),
		ID:     w.reg.ID, AttachedAt: w.reg.AttachedAt, PID: w.p.PID,
		StartedAt: start.UTC().Format(time.RFC3339),
	}
	var wasAccepting bool
	decide := func() {
		// An unreadable previous mark carries nothing over, and the start
		// replaces it.
		prev, _ := ReadDirectorWait(w.p.StateDir)
		if prev != nil {
			w.prevWaitID = prev.WaitID
		}
		wasAccepting = DirectorWaitFor(prev, &w.reg) != nil && prev.WatchAccepting
		// A probe ctx cut short found the watch stale, so observed is false
		// then, and the carried value stands.
		w.accepting = observed || wasAccepting
		w.mark.WatchAccepting = w.accepting
		w.mark.SeenAt = w.p.Now().UTC().Format(time.RFC3339)
	}
	if directorWaitBeforeBeginLock != nil {
		directorWaitBeforeBeginLock()
	}
	// readErr and notCurrent carry out of the lock the two outcomes of the
	// read under it that end the wait with nothing written.
	var readErr, notCurrent error
	decided := false
	err := withFileLock(w.p.StateDir, directorWaitLockFile, "director wait", func() error {
		res, err := NewDirectorStore(w.p.StateDir).Read()
		if err != nil {
			readErr = err
			return nil
		}
		if _, err := directorWaitCheckCurrent(w.p.ID, &w.reg, res.Current); err != nil {
			notCurrent = err
			return nil
		}
		decide()
		decided = true
		return writeJSONFileAtomic(DirectorWaitPathIn(w.p.StateDir), w.mark)
	})
	switch {
	case readErr != nil:
		return fmt.Errorf("%s%w", directorWaitErrPrefix, readErr)
	case notCurrent != nil:
		return notCurrent
	}
	if !decided {
		decide()
	}
	if err != nil {
		w.sayWriteFailed(err)
	} else {
		w.written = true
	}
	switch {
	case !settled, observed:
	case wasAccepting:
		w.noWatch = true
		w.noWatchSince = "the last wait of " + w.reg.ID + " wrote its mark"
	default:
		_, _ = fmt.Fprintf(w.stderr, "note: no ralph org watch --all accepts work on this ledger, and this wait keeps waiting: %s\n", directorWaitWatchFacts(w.p, l))
	}
	return nil
}

// directorWaitBeforeBeginLock is a test-only seam that begin runs after its
// probe of the watch and before it takes director-wait.lock, so a test can
// replace the registration (or director.jsonl) in between; nil means none.
var directorWaitBeforeBeginLock func()

// loop runs the rounds of the wait (WaitInboxForDirector's doc) until one
// ends it.
func (w *directorWaitRun) loop(ctx context.Context) ([]InboxItem, error) {
	interval := w.o.InboxPollInterval
	if interval <= 0 {
		interval = defaultInboxPollInterval
	}
	var deadline <-chan time.Time
	if w.p.Timeout > 0 {
		timer := time.NewTimer(w.p.Timeout)
		defer timer.Stop()
		deadline = timer.C
	}
	inboxTick := time.NewTicker(interval)
	defer inboxTick.Stop()
	probeTick := time.NewTicker(w.p.WatchProbeInterval)
	defer probeTick.Stop()
	markTick := time.NewTicker(w.p.MarkInterval)
	defer markTick.Stop()
	// expired is set when the deadline fires; the loop then reads once more,
	// so an item that arrived just before the deadline is still returned.
	expired, probeDue, markDue := false, false, false
	for {
		if ctx.Err() != nil {
			return nil, w.endSignal(ctx)
		}
		open, err := w.o.openInboxItems()
		if err != nil {
			return nil, w.endError(fmt.Errorf("%s%w", directorWaitErrPrefix, err))
		}
		if len(open) > 0 {
			w.end(DirectorWaitEndItems, directorWaitItemsDetail(len(open)))
			return open, nil
		}
		if w.noWatch {
			return nil, w.endNoWatch()
		}
		if expired {
			w.end(DirectorWaitEndTimeout, fmt.Sprintf("no open inbox item within %d ms", w.p.Timeout.Milliseconds()))
			return nil, fmt.Errorf("%s%s %d ms", directorWaitErrPrefix, inboxWaitTimeoutPhrase, w.p.Timeout.Milliseconds())
		}
		if probeDue {
			probeDue = false
			// A watch that no longer accepts work: read the inbox once more
			// before the wait ends for it. A done ctx: end for it.
			if w.probeAgain(ctx) || ctx.Err() != nil {
				continue
			}
		}
		if markDue {
			markDue = false
			if err := w.refresh(); err != nil {
				return nil, err
			}
		}
		select {
		case <-ctx.Done():
		case <-deadline:
			expired = true
		case <-inboxTick.C:
		case <-probeTick.C:
			probeDue = true
		case <-markTick.C:
			markDue = true
		}
	}
}

// probe probes the watch at p.Now() (ProbeWatch, with Config.DeadmanMinutes
// for a mark that carries none). With settle set, a stale watch is probed
// again every watchGateProbeInterval until p.Settle has passed, as
// NewWatchGate does, and the last probe is returned. ok is false when ctx
// ended the settle first; the probe then decides nothing.
func (w *directorWaitRun) probe(ctx context.Context, settle bool) (l WatchLiveness, ok bool) {
	deadline := time.Now().Add(w.p.Settle)
	for {
		l = ProbeWatch(w.p.StateDir, w.p.Now(), w.o.Config.DeadmanMinutes)
		left := time.Until(deadline)
		if !settle || l.State != WatchLivenessStale || left <= 0 {
			return l, true
		}
		select {
		case <-ctx.Done():
			return l, false
		case <-time.After(min(watchGateProbeInterval, left)):
		}
	}
}

// probeAgain is the probe of a round (D3). Only a watch that accepted work is
// settled: one that did not, and is stale, still does not. It reports true
// when a watch that accepted work no longer does; the caller reads the inbox
// once more and then ends the wait. A watch that starts accepting work gets a
// note: line and a rewrite of the mark at once.
func (w *directorWaitRun) probeAgain(ctx context.Context) bool {
	l, ok := w.probe(ctx, w.accepting)
	if !ok || ctx.Err() != nil {
		return false
	}
	w.watch = l
	accepting := WatchAcceptsWork(l)
	switch {
	case w.accepting && !accepting:
		w.noWatch = true
		w.noWatchSince = "this wait last probed it"
		return true
	case !w.accepting && accepting:
		w.accepting = true
		w.mark.WatchAccepting = true
		_, _ = fmt.Fprintf(w.stderr, "note: a ralph org watch --all accepts work again on this ledger: %s\n", directorWaitWatchFacts(w.p, l))
		w.save()
	}
	return false
}

// refresh is the rewrite of every p.MarkInterval: it reads director.jsonl
// again and ends the wait when the registration is no longer current (D1),
// or rewrites the mark's seen_at. It returns the error that ends the wait.
func (w *directorWaitRun) refresh() error {
	res, err := NewDirectorStore(w.p.StateDir).Read()
	if err != nil {
		return w.endError(fmt.Errorf("%s%w", directorWaitErrPrefix, err))
	}
	if detail, err := directorWaitCheckCurrent(w.p.ID, &w.reg, res.Current); err != nil {
		w.end(DirectorWaitEndNotCurrent, detail)
		return err
	}
	w.save()
	return nil
}

// endNoWatch ends the wait for the watch: the mark says end watch and
// watch_accepting false, so the next wait does not end for the same change.
func (w *directorWaitRun) endNoWatch() error {
	w.accepting = false
	w.end(DirectorWaitEndWatch, fmt.Sprintf("no watch accepts work: watch %s, %s", w.watch.State, WatchPIDText(w.watch.Mark)))
	return fmt.Errorf("%s%w: a ralph org watch --all accepted work when %s, and none does now: %s",
		directorWaitErrPrefix, ErrDirectorWaitNoWatch, w.noWatchSince, directorWaitWatchFacts(w.p, w.watch))
}

// endError ends the wait for err, a failed read.
func (w *directorWaitRun) endError(err error) error {
	w.end(DirectorWaitEndError, err.Error())
	return err
}

// endSignal ends the wait for ctx, which is done.
func (w *directorWaitRun) endSignal(ctx context.Context) error {
	cause := context.Cause(ctx)
	w.end(DirectorWaitEndSignal, cause.Error())
	return fmt.Errorf("%sthe wait was stopped: %w", directorWaitErrPrefix, cause)
}

// end writes how the wait ended to its mark (D2), with the watch it last saw.
func (w *directorWaitRun) end(end, detail string) {
	w.mark.WatchAccepting = w.accepting
	w.mark.EndedAt = w.p.Now().UTC().Format(time.RFC3339)
	w.mark.End, w.mark.Detail = end, PrintableInboxText(detail, false)
	w.save()
}

// save rewrites the mark with seen_at now, under director-wait.lock, while
// this wait owns it (owns); once another wait's mark is there it says so once
// (sayLost) and writes no more. A read or write that fails is said once, and
// the next save tries again.
func (w *directorWaitRun) save() {
	if w.lost {
		return
	}
	w.mark.SeenAt = w.p.Now().UTC().Format(time.RFC3339)
	var other *DirectorWaitMark
	lost := false
	err := withFileLock(w.p.StateDir, directorWaitLockFile, "director wait", func() error {
		disk, err := ReadDirectorWait(w.p.StateDir)
		if err != nil {
			return err
		}
		if !w.owns(disk) {
			other, lost = disk, true
			return nil
		}
		return writeJSONFileAtomic(DirectorWaitPathIn(w.p.StateDir), w.mark)
	})
	switch {
	case err != nil:
		w.sayWriteFailed(err)
	case lost:
		w.sayLost(other)
	default:
		w.written = true
	}
}

// owns reports whether this wait may replace disk, the mark as it is under
// the lock (nil for none): it holds this wait's wait_id, or no write of this
// wait has landed yet and it is still the mark the start read.
func (w *directorWaitRun) owns(disk *DirectorWaitMark) bool {
	if disk != nil && disk.WaitID == w.mark.WaitID {
		return true
	}
	if w.written {
		return false
	}
	if disk == nil {
		return w.prevWaitID == ""
	}
	return w.prevWaitID != "" && disk.WaitID == w.prevWaitID
}

// sayLost says once that another wait's mark replaced this wait's, or that
// the mark is gone, and stops this wait's writes (D2).
func (w *directorWaitRun) sayLost(other *DirectorWaitMark) {
	w.lost = true
	path := DirectorWaitPathIn(w.p.StateDir)
	if other == nil {
		_, _ = fmt.Fprintf(w.stderr, directorWaitNoticePrefix+"%s no longer holds the mark of this wait (wait_id %s): the file is missing, so this wait does not write it again and keeps waiting\n",
			path, w.mark.WaitID)
		return
	}
	_, _ = fmt.Fprintf(w.stderr, directorWaitNoticePrefix+"another ralph org wait --inbox --director (wait_id %s, pid %d) took over %s, so this wait (wait_id %s) does not write it again and keeps waiting\n",
		PrintableInboxText(other.WaitID, false), other.PID, path, w.mark.WaitID)
}

// sayWriteFailed says the first failed write of the mark; the wait goes on.
func (w *directorWaitRun) sayWriteFailed(err error) {
	if w.writeFailedSaid {
		return
	}
	w.writeFailedSaid = true
	_, _ = fmt.Fprintf(w.stderr, directorWaitNoticePrefix+"the wait mark is not written (this wait keeps waiting, and says this once): %v\n", err)
}

// directorWaitItemsDetail is the detail of end items.
func directorWaitItemsDetail(n int) string {
	if n == 1 {
		return "1 open inbox item"
	}
	return strconv.Itoa(n) + " open inbox items"
}

// directorWaitWatchFacts is the text about the watch l that the wait's error
// and note: lines give (D3): what watch.lock and watch-alive.json show
// (directorWaitWatchState), and, for a watch that accepts no work, the
// command that starts one, with p.HintStateDir repeated as --state-dir, and
// what happens to the inbox meanwhile. It never says why a watch stopped:
// the mark reads the same whether the human stopped the watch or it ended on
// its own (plan Assumptions).
func directorWaitWatchFacts(p DirectorWaitParams, l WatchLiveness) string {
	state := directorWaitWatchState(p.StateDir, l)
	if WatchAcceptsWork(l) {
		return state
	}
	return state + "; the command that starts one is `" + CommandWithStateDir("ralph org watch --all --detach", p.HintStateDir) + "`" +
		"; while no watch accepts work, an item escalated reaches the human at once, but an item left for the director before then" +
		" reaches the human at its ack deadline only once a watch runs again"
}

// directorWaitWatchState says what watch.lock and watch-alive.json of
// stateDir show for l, one text for each row of D3, with every value read
// from the mark escaped (PrintableInboxText).
func directorWaitWatchState(stateDir string, l WatchLiveness) string {
	m := l.Mark
	pid := WatchPIDText(m)
	switch l.State {
	case WatchLivenessNotRunning:
		switch {
		case m == nil:
			return "watch.lock is free and there is no readable watch-alive.json"
		case m.State == WatchAliveRunning || m.State == WatchAliveStopping:
			return fmt.Sprintf("watch.lock is free but watch-alive.json still says %s (%s, written at %s): the watch ended without writing its stopped mark",
				PrintableInboxText(m.State, false), pid, directorWaitMarkTS(m.TS))
		case m.State == WatchAliveStopped:
			return fmt.Sprintf("watch.lock is free and watch-alive.json says stopped at %s (%s)", directorWaitMarkTS(m.StoppedAt), pid)
		default:
			return fmt.Sprintf("watch.lock is free and watch-alive.json says %s (%s)", PrintableInboxText(m.State, false), pid)
		}
	case WatchLivenessStale:
		if m == nil {
			return "watch.lock is held, but there is no readable watch-alive.json; " + WatchKillHint(stateDir, m)
		}
		return fmt.Sprintf("watch.lock is held, but watch-alive.json has not been rewritten as running since %s (%s, it says %s); %s",
			directorWaitMarkTS(m.TS), pid, PrintableInboxText(m.State, false), WatchKillHint(stateDir, m))
	case WatchLivenessStopping:
		return fmt.Sprintf("the watch (%s) is stopping: watch.lock is held and watch-alive.json says stopping (written at %s)", pid, directorWaitMarkTS(directorWaitTSOf(m)))
	case WatchLivenessRunning:
		if m != nil && m.Once {
			return fmt.Sprintf("the running watch (%s) is a run that ends after its cycles (--once), which accepts no new work", pid)
		}
		return fmt.Sprintf("watch.lock is held and watch-alive.json says running (%s, written at %s)", pid, directorWaitMarkTS(directorWaitTSOf(m)))
	}
	return "the watch is " + PrintableInboxText(l.State, false)
}

// directorWaitTSOf is the ts of m, "" for no mark.
func directorWaitTSOf(m *WatchAlive) string {
	if m == nil {
		return ""
	}
	return m.TS
}

// directorWaitMarkTS is a time read from the watch's mark, escaped, or "an
// unknown time" when it is empty.
func directorWaitMarkTS(ts string) string {
	if ts == "" {
		return "an unknown time"
	}
	return PrintableInboxText(ts, false)
}
