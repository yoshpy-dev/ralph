package org

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// manifestLockFile is the fixed lock file name withManifestLock creates (if
// absent) inside dir and flocks for the duration of fn. One lock file per
// manifest directory -- every Spawn call for every org_id sharing that
// directory serializes through the same lock, since [org].max_seats is
// enforced per state-dir/manifest file, not per org_id (see ActiveSeatCount,
// which is scoped by org_id but reads the same manifest file).
const manifestLockFile = "manifest.lock"

// spawnLockName is the name of the lock file a non-dry-run Spawn of seatID
// in orgID holds (acquireHeldFileLock) in the manifest directory from
// before its first manifest read until it returns, so a second spawn of the
// same seat is refused at once instead of taking the first one's in-flight
// records for a stale saga (plan 2026-10-22-seat-spawn-close, D5). Both ids
// passed ValidateIdentifier, which allows no `_` and no path separator, so
// the name is one file per seat and stays inside the directory.
func spawnLockName(orgID, seatID string) string {
	return "spawn-" + orgID + "_" + seatID + ".lock"
}

// manifestLockTimeout bounds how long withManifestLock waits to acquire the
// flock before giving up with a clear timeout error. flock(2) is released
// automatically when the holding process exits or its file descriptor
// closes, so an orphaned lock from a crashed process cannot wedge future
// spawns forever -- this timeout instead guards against a slow-but-alive
// holder (e.g. a concurrent spawn's compensateStale call) blocking a caller
// past a reasonable bound. It also bounds the waits for the inbox lock, the
// director lock, the director wait lock, and the follow-up lock, since
// withFileLock serves all five locks.
const manifestLockTimeout = 5 * time.Second

// manifestLockPollInterval is how often withManifestLock retries a
// non-blocking LOCK_EX attempt while waiting for manifestLockTimeout. The
// inbox, director, director wait, and follow-up locks are retried at the
// same interval, since withFileLock serves all five locks, and so is the
// watch's held lock (acquireHeldFileLock).
const manifestLockPollInterval = 25 * time.Millisecond

// withManifestLock serializes fn against every other withManifestLock call
// racing on the same dir -- in this process (goroutines) and in any other
// process on the same host -- via an OS advisory file lock (flock(2),
// LOCK_EX) on <dir>/manifest.lock. This closes the max_seats TOCTOU race
// described in docs/tech-debt/README.md ("max_seats is enforced across an
// unlocked read-then-append window in the spawn saga"): two concurrent
// `ralph org spawn` calls previously could both observe the same
// activeSeats snapshot and both pass ValidateSpawnCapacity, exceeding
// [org].max_seats.
//
// fn runs with the lock held; withManifestLock always releases the lock
// (LOCK_UN, then closing the fd) before returning, whether fn succeeded or
// not. A timeout acquiring the lock returns a plain error (not fn's error),
// so callers can distinguish "never got to run fn" from "fn itself failed".
// The lock scope is deliberately the caller's responsibility to keep
// minimal -- see Spawn's own doc comment for why only the read-validate-
// append section is wrapped, not the subsequent herdr/agmsg round trip.
func withManifestLock(dir string, fn func() error) error {
	return withManifestLockContext(context.Background(), dir, fn)
}

// withManifestLockContext is withManifestLock whose wait for the lock also
// ends when ctx is done, with an error that wraps ctx's cause and does not
// say the wait timed out (withFileLockContext). recordGate uses it, so an
// interrupted `ralph org gate` stops waiting at once. Taking a free lock does
// not look at ctx: fn runs even when ctx is already done, so a caller that
// must not act once ctx is done checks ctx in fn as well.
func withManifestLockContext(ctx context.Context, dir string, fn func() error) error {
	return withFileLockContext(ctx, dir, manifestLockFile, "manifest", fn)
}

// withFileLock is the flock mechanism behind withManifestLock, the inbox
// lock (inbox.go's inboxLockFile), the director lock (director.go's
// directorLockFile, director.lock), the director wait lock
// (director_wait.go's directorWaitLockFile, director-wait.lock), and the
// follow-up lock (followup.go's followupLockFile, followups.lock): it
// creates dir and <dir>/<name> as needed, takes LOCK_EX on that file within
// manifestLockTimeout, which bounds the wait for each of the five, runs fn,
// and releases the lock before returning. what names the lock in the error
// messages ("manifest" gives withManifestLock's messages). Each lock file is
// independent: holding one never blocks a caller of another.
func withFileLock(dir, name, what string, fn func() error) error {
	return withFileLockContext(context.Background(), dir, name, what, fn)
}

// withFileLockContext is withFileLock whose wait for the lock also ends when
// ctx is done. The error then wraps context.Cause(ctx) instead of saying the
// wait timed out, so a caller can tell its own interruption from a lock held
// past manifestLockTimeout.
func withFileLockContext(ctx context.Context, dir, name, what string, fn func() error) error {
	f, lockPath, err := openLockFile(dir, name, what)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	waitCtx, cancel := context.WithTimeout(ctx, manifestLockTimeout)
	defer cancel()

	if err := acquireFlock(waitCtx, f); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("org: acquire %s lock %s: the wait was stopped: %w", what, lockPath, context.Cause(ctx))
		}
		return fmt.Errorf("org: acquire %s lock %s: %w", what, lockPath, err)
	}
	defer func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }()

	return fn()
}

// openLockFile creates dir and opens <dir>/<name> (creating it if absent)
// for withFileLock and acquireHeldFileLock. what names the lock in the
// error messages.
func openLockFile(dir, name, what string) (*os.File, string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, "", fmt.Errorf("org: create %s lock dir %s: %w", what, dir, err)
	}
	lockPath := filepath.Join(dir, name)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, "", fmt.Errorf("org: open %s lock file %s: %w", what, lockPath, err)
	}
	return f, lockPath, nil
}

// errFileLockHeld is wrapped by acquireHeldFileLock's error when the lock
// was still held by another open file when its wait ran out.
var errFileLockHeld = errors.New("the lock is held")

// heldFileLock is a flock that outlives the call that took it, unlike
// withFileLock's: the lock file stays open with LOCK_EX on it until
// release. The cross-org watch holds one for its lifetime (watchalive.go's
// watch.lock), and ralph org regate one from its first check to its end
// (regate.go's regate.lock), for minutes rather than for the watch's life,
// with its marker written into the file for a spawn to read once its
// fileLockHeld check finds the lock held (regateSpawnErr); a regate under
// [org.gate].regate_checks = "ci" with --wait-checks releases it while its
// wait for the PR's checks only reads them and takes it again once, when the
// wait stops, to check again what its record relies on and to record
// (regate_ci.go's regateChecksWithWait), so that a merge, a cleanup, or
// another regate can run meanwhile. ralph org merge
// holds regate.lock too, from its checks through gh pr merge and the record
// (merge_run.go's mergeLocked), with the marker emptied and none written, so
// one regate, merge, or cleanup holds it on a ledger at a time and a spawn
// goes on while a merge runs. Once the merge is recorded, merge cleans up the
// branch's feature worktree with the lock still held, and ralph org cleanup
// takes the lock for its cleanup (cleanup.go); either writes the marker of
// the org whose worktree it cleans up, so a spawn of that org is refused
// while it runs. A merge with --wait-checks releases it for every wait in
// merge_wait.go, so that a regate or another merge can run meanwhile, and
// takes it again for the checks after the wait. A spawn holds one on its
// seat's spawnLockName for the whole call, with no marker. The lock is tied
// to the open file, not the process, so two heldFileLocks on the same file
// in one process exclude each other too, and the OS releases it when the
// process exits however it exits.
type heldFileLock struct {
	f *os.File
}

// acquireHeldFileLock takes LOCK_EX on <dir>/<name> (both created as
// needed) and keeps it. It retries a non-blocking attempt at
// manifestLockPollInterval for up to wait (at least one attempt), so a
// short shared lock that fileLockHeld takes (a spawn's check of
// regate.lock, ProbeWatch's of watch.lock) does not make it fail. When the
// lock is still held after wait, the error wraps errFileLockHeld; any other
// flock error returns at once.
func acquireHeldFileLock(dir, name, what string, wait time.Duration) (*heldFileLock, error) {
	f, lockPath, err := openLockFile(dir, name, what)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(wait)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return &heldFileLock{f: f}, nil
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EINTR) {
			_ = f.Close()
			return nil, fmt.Errorf("org: acquire %s lock %s: %w", what, lockPath, err)
		}
		left := time.Until(deadline)
		if left <= 0 {
			_ = f.Close()
			return nil, fmt.Errorf("org: acquire %s lock %s: %w after %s", what, lockPath, errFileLockHeld, wait)
		}
		time.Sleep(min(manifestLockPollInterval, left))
	}
}

// release drops the lock and closes the file.
func (l *heldFileLock) release() error {
	_ = syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	return l.f.Close()
}

// fileLockHeld reports whether another open file holds a lock on path: it
// takes LOCK_SH|LOCK_NB on a fresh read-only open of path and drops it at
// once, so a holder of LOCK_EX shows as held (EWOULDBLOCK) and nothing else
// does. A missing file is not held. err is set when the check could not be
// made (path cannot be opened, or flock fails another way).
func fileLockHeld(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	defer func() { _ = f.Close() }()
	fd := int(f.Fd())
	err = syscall.Flock(fd, syscall.LOCK_SH|syscall.LOCK_NB)
	switch {
	case err == nil:
		_ = syscall.Flock(fd, syscall.LOCK_UN)
		return false, nil
	case errors.Is(err, syscall.EWOULDBLOCK):
		return true, nil
	default:
		return false, err
	}
}

// acquireFlock polls a non-blocking LOCK_EX attempt (LOCK_EX|LOCK_NB) at
// manifestLockPollInterval until it succeeds or ctx is done. Polling (rather
// than a single blocking flock call) keeps the wait bounded strictly by
// ctx's own timeout.
func acquireFlock(ctx context.Context, f *os.File) error {
	fd := int(f.Fd())
	for {
		err := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out after %s: %w", manifestLockTimeout, ctx.Err())
		case <-time.After(manifestLockPollInterval):
		}
	}
}
