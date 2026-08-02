package org

import (
	"context"
	"fmt"
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

// manifestLockTimeout bounds how long withManifestLock waits to acquire the
// flock before giving up with a clear timeout error. flock(2) is released
// automatically when the holding process exits or its file descriptor
// closes, so an orphaned lock from a crashed process cannot wedge future
// spawns forever -- this timeout instead guards against a slow-but-alive
// holder (e.g. a concurrent spawn's compensateStale call) blocking a caller
// past a reasonable bound.
const manifestLockTimeout = 5 * time.Second

// manifestLockPollInterval is how often withManifestLock retries a
// non-blocking LOCK_EX attempt while waiting for manifestLockTimeout.
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
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("org: create manifest lock dir %s: %w", dir, err)
	}
	lockPath := filepath.Join(dir, manifestLockFile)
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("org: open manifest lock file %s: %w", lockPath, err)
	}
	defer func() { _ = f.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), manifestLockTimeout)
	defer cancel()

	if err := acquireFlock(ctx, f); err != nil {
		return fmt.Errorf("org: acquire manifest lock %s: %w", lockPath, err)
	}
	defer func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }()

	return fn()
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
