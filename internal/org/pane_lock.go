package org

import (
	"errors"
	"path/filepath"
	"time"
)

// The pane lock (plan 2026-11-07-org-seat-resume, D8; spec FR-5a). Send and
// Resume each take the lock of the herdr pane they type into, in the
// manifest's directory, after their check of the seat (Send: the idle/done
// wait and the second approval check; Resume: the agent get, the pane read,
// and the screen's classification) and before they type, and hold it until
// Enter is pressed and the submit confirmed, so two of them never type into
// one pane at the same time. Resume reads the screen again once it holds the
// lock, so a second Resume, or one that comes after a Send, types only when
// the screen still shows the stop it classified. Send waits for the previous
// holder to release the lock (within its --timeout-ms) and then waits for the
// seat to go idle or done again once it holds the lock (cross-review cycle 1
// #1 of org-seat-resume), so it types nothing into a turn that a Resume or
// another Send started, or an approval dialog it brought up, while the turn
// is still on; it may type after that turn has ended. If the previous
// holder's submit was unconfirmed when it released the lock, herdr may still
// report idle or done, and the second wait passes.
//
// The push wake of a headless director in herdr (typeDirectorWakeLine,
// director_wake.go) takes the same lock of the director's pane around its
// send-text, pause, and Enter (cross-review cycle 1 #3), waiting for it no
// longer than one herdr call's driverCallTimeout; a lock still held then is
// a failed wake, with nothing typed.
//
// The lock is a heldFileLock (acquireHeldFileLock), tied to the open file
// rather than the process, so two callers in one process exclude each other
// too, and the OS releases it when the process exits however it exits. A
// caller that cannot take it within its wait types nothing.

// paneLockName is the name of the lock file of the herdr pane paneID in the
// manifest's directory: "pane-", paneID with every byte outside
// [A-Za-z0-9._-] replaced by '_', and ".lock". herdr's pane ids
// (`pane-3`, `w1:p1`) hold no path separator after the replacement, so the
// name stays inside the directory. Two ids that differ only in the replaced
// bytes (`w1:p1` and `w1_p1`) share a lock file; that is harmless, since a
// shared lock only makes the two panes' sends and resumes wait for each other.
func paneLockName(paneID string) string {
	b := []byte(paneID)
	for i, c := range b {
		if !isPaneLockNameByte(c) {
			b[i] = '_'
		}
	}
	return "pane-" + string(b) + ".lock"
}

// isPaneLockNameByte reports whether c is kept as it is in paneLockName.
func isPaneLockNameByte(c byte) bool {
	switch {
	case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9':
		return true
	}
	return c == '.' || c == '_' || c == '-'
}

// acquirePaneLock takes the lock of paneID (paneLockName) in the manifest's
// directory, waiting for it up to wait (at least one attempt), and keeps it
// until release. When another holder kept it for the whole wait, the error
// is one for which isPaneLockHeld is true; any other error (the directory or
// the file cannot be made or opened) is not.
func (o *Org) acquirePaneLock(paneID string, wait time.Duration) (*heldFileLock, error) {
	return acquireHeldFileLock(filepath.Dir(o.Manifest.Path()), paneLockName(paneID), "pane", wait)
}

// isPaneLockHeld reports whether err is acquirePaneLock's error for a lock
// another holder kept for the whole wait.
func isPaneLockHeld(err error) bool {
	return errors.Is(err, errFileLockHeld)
}
