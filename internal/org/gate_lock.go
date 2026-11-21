package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode"
)

// gate.lock: one gate's scripts at a time on a ledger (plan
// docs/plans/active/2026-11-12-gate-lock-accept.md, D1 and D8, issue #309).
// ralph org gate (Gate) and a ralph org regate that runs the gate scripts
// (Regate with local checks) take it in the ledger's state dir and wait for it
// with no time limit while another holds it, saying on stderr who holds it
// (acquireGateLock). Gate takes it once its first checks pass and makes them
// again after it, so a wait changes neither what it gates nor whom it
// refuses; Regate takes it before regate.lock, so a merge or a cleanup is not
// refused while a regate only waits for another gate. The holder writes its
// marker into the file (gateLockMarker) and empties it when it releases the
// lock. The lock's open file is passed to the scripts (runGateScript), so a
// script that runs on after the ralph that started it was killed keeps the
// lock until it ends.

// gateLockFile is the lock in the state dir that one gate holds at a time.
const gateLockFile = "gate.lock"

var (
	// gateLockPoll is how often a gate waiting for gate.lock tries it again
	// and reads the holder's marker. Tests shorten it.
	gateLockPoll = time.Second
	// gateLockRemind is how often a gate that still waits for the same holder
	// says so again on stderr. Tests shorten it.
	gateLockRemind = 5 * time.Minute
	// gateLockPolled, when not nil, gets what gateLockHolder said at each try
	// of a gate waiting for gate.lock, before the wait picks the holder it
	// names. Tests set it to know what a try read; it is nil otherwise.
	gateLockPolled func(seen string)
)

// gateLockNotWritten is what gateLockHolder says of an empty marker.
const gateLockNotWritten = "a gate whose marker is not written yet"

// gateLockMarker is what gate.lock holds while a gate holds it, as one JSON
// line: the verb (gateWording.verb), the org it gates (a regate writes it
// once its first check found the branch's org) or the branch a regate was
// named with, when it took the lock, and its pid. A gate waiting for the lock
// reads it to say who holds it (gateLockHolder).
type gateLockMarker struct {
	Verb      string `json:"verb"`
	OrgID     string `json:"org_id,omitempty"`
	Branch    string `json:"branch,omitempty"`
	StartedAt string `json:"started_at"`
	PID       int    `json:"pid"`
}

// gateLock is gate.lock as one gate holds it, with the marker it wrote.
type gateLock struct {
	held   *heldFileLock
	marker gateLockMarker
	prefix string // the verb's error prefix: "org: gate: "
}

// gateLockWait is what a gate's wait for gate.lock was: how long it waited,
// and the holder it named last (acquireGateLock): the one its last try read
// (gateLockHolder), or the one read with a name before an empty marker.
// waited is 0 when the lock was free.
type gateLockWait struct {
	waited time.Duration
	holder string
}

// logLines are the lines the gate log's header gets for w (gateRunOptions):
// none when the gate did not wait.
func (w gateLockWait) logLines() []string {
	if w.waited <= 0 {
		return nil
	}
	return []string{fmt.Sprintf("# waited %s for %s: %s", gateLockDuration(w.waited), gateLockFile, w.holder)}
}

// gateLockPathIn is the gate.lock of the state dir stateDir.
func gateLockPathIn(stateDir string) string {
	return filepath.Join(stateDir, gateLockFile)
}

// acquireGateLock takes gate.lock in o's state dir and writes marker into it,
// with StartedAt the time it took the lock (o.nowTime(), UTC) and PID this
// process's. While another holds the lock it waits with no time limit
// (waitHeldFileLock, every gateLockPoll) and writes a line to stderr, starting
// with prefix ("org: gate: "), that names the lock and its holder
// (gateLockHolder) and how long it has waited: when the wait starts, when the
// holder changes, and every gateLockRemind with the same holder; once it took
// the lock after a wait, one more line says so. The holder is the one its last
// try read, except that an empty marker read while the holder is one read
// with a name keeps that name: a gate empties its marker before it drops the
// lock (gateLock.release), and a rewrite empties it for a moment
// (writeGateLockContent), so a try that reads that gap does not change the
// holder: no line says the marker is not written yet, and the log
// (gateLockWait) names the gate this one waited for. Until a gate that took
// the lock writes its marker, the name stays the one before it. An empty
// marker with no named holder before it is named as not written yet
// (gateLockNotWritten), and a marker that cannot be read replaces any holder.
// When ctx is done during the wait, it returns an error that wraps ctx's
// cause, holds nothing, and writes nothing into the lock. An error writing the
// marker releases the lock.
func (o *Org) acquireGateLock(ctx context.Context, marker gateLockMarker, stderr io.Writer, prefix string) (*gateLock, gateLockWait, error) {
	stateDir := filepath.Dir(o.Manifest.Path())
	path := gateLockPathIn(stateDir)
	var w gateLockWait
	named := false // w.holder was read from a marker that names its holder
	said, saidAt := "", time.Duration(-1)
	begin := time.Now()
	held, err := waitHeldFileLock(ctx, stateDir, gateLockFile, "gate", gateLockPoll, func(waited time.Duration) {
		seen, seenNamed := gateLockHolder(path)
		if gateLockPolled != nil {
			gateLockPolled(seen)
		}
		if seen != gateLockNotWritten || !named {
			w.holder, named = seen, seenNamed
		}
		if saidAt >= 0 && w.holder == said && waited-saidAt < gateLockRemind {
			return
		}
		said, saidAt = w.holder, waited
		_, _ = fmt.Fprintf(stderr, "%swaiting for %s (waited %s): held by %s; a ledger runs one gate's scripts at a time, so this one "+
			"starts when that one ends\n", prefix, path, gateLockDuration(waited), w.holder)
	})
	if err != nil {
		return nil, gateLockWait{}, err
	}
	if saidAt >= 0 {
		w.waited = time.Since(begin)
		_, _ = fmt.Fprintf(stderr, "%stook %s after waiting %s\n", prefix, path, gateLockDuration(w.waited))
	}
	marker.StartedAt = o.nowTime().UTC().Format(time.RFC3339)
	marker.PID = os.Getpid()
	l := &gateLock{held: held, marker: marker, prefix: prefix}
	if err := l.write(); err != nil {
		_ = held.release()
		return nil, gateLockWait{}, err
	}
	return l, w, nil
}

// file is gate.lock's open file, which the scripts get (gateRunOptions.lock).
func (l *gateLock) file() *os.File {
	return l.held.f
}

// setOrgID writes the marker again with orgID: a regate's, once its first
// check found the branch's org.
func (l *gateLock) setOrgID(orgID string) error {
	l.marker.OrgID = orgID
	return l.write()
}

// write replaces the content of gate.lock with l's marker as one JSON line.
func (l *gateLock) write() error {
	line, err := json.Marshal(l.marker)
	if err != nil {
		return fmt.Errorf("%sencode the marker of %s: %w", l.prefix, l.held.f.Name(), err)
	}
	return writeGateLockContent(l.prefix, l.held.f, append(line, '\n'))
}

// release empties the marker, then drops the lock and closes the file
// (heldFileLock.release), so a gate that takes the lock next never reads this
// one's marker as its holder's. The error is the first that failed; the lock
// is dropped either way.
func (l *gateLock) release() error {
	emptyErr := writeGateLockContent(l.prefix, l.held.f, nil)
	if err := l.held.release(); err != nil && emptyErr == nil {
		return err
	}
	return emptyErr
}

// writeGateLockContent replaces the content of f, the open gate.lock, with
// data. An error starts with prefix.
func writeGateLockContent(prefix string, f *os.File, data []byte) error {
	if err := f.Truncate(0); err != nil {
		return fmt.Errorf("%swrite the marker of %s: %w", prefix, f.Name(), err)
	}
	if _, err := f.WriteAt(data, 0); err != nil {
		return fmt.Errorf("%swrite the marker of %s: %w", prefix, f.Name(), err)
	}
	return nil
}

// gateLockHolder says who holds the gate.lock at path, from its marker: the
// verb, the org_id or the branch (both for a regate that has found its org),
// when it took the lock, and its pid, with the note that a script the holder
// started still holds the lock when that pid is gone (kill(pid, 0) answers
// ESRCH: the lock's file was passed to the scripts, D8). An empty marker is
// one not written yet (gateLockNotWritten: the holder took the lock a moment
// ago, is rewriting its marker, or is releasing the lock); one that cannot be
// read says why. named is true only for a marker that names its holder,
// neither empty nor unreadable: a waiting gate keeps such a holder over an
// empty marker it reads next (acquireGateLock).
func gateLockHolder(path string) (holder string, named bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("a gate whose marker cannot be read (%s)", pathErrorCause(err)), false
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return gateLockNotWritten, false
	}
	var m gateLockMarker
	switch err := json.Unmarshal(data, &m); {
	case err != nil:
		return fmt.Sprintf("a gate whose marker cannot be read (%v)", err), false
	case m.Verb == "" || m.PID <= 0 || strings.ContainsFunc(m.Verb+m.StartedAt, unicode.IsControl):
		return "a gate whose marker cannot be read (it names no verb or pid, or holds a control character)", false
	}
	what := m.Verb
	switch {
	case m.Branch != "" && m.OrgID != "":
		what += fmt.Sprintf(" of the branch %q (org_id %q)", m.Branch, m.OrgID)
	case m.Branch != "":
		what += fmt.Sprintf(" of the branch %q", m.Branch)
	case m.OrgID != "":
		what += fmt.Sprintf(" of org_id %q", m.OrgID)
	}
	what += fmt.Sprintf(", started %s by pid %d", m.StartedAt, m.PID)
	if errors.Is(syscall.Kill(m.PID, 0), syscall.ESRCH) {
		what += fmt.Sprintf(" (pid %d has exited, so a script it started still holds the lock)", m.PID)
	}
	return what, true
}

// gateLockDuration is d as the lines about gate.lock print it, to the second.
func gateLockDuration(d time.Duration) string {
	return d.Round(time.Second).String()
}
