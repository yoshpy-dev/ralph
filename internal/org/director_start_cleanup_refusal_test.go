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
)

// Two refusals of start's check and cleanup of a current headless
// registration (plan 2026-10-18-org-director-liveness, D6 and AC6) that
// director_start_cleanup_test.go does not reach: a check that cannot read
// the ledger (checkCurrentDirector's CheckDirector error), and a launch_cmd
// cleanup whose ps cannot confirm the end of the process the check found
// gone. They use cleanupFixture: herdr is a fake, ps is a fake, and every
// ledger is a t.TempDir.

// wantNothingStarted fails t unless start made no herdr call, wrote no
// prompt file, asked no watch gate, and wrote no line.
func wantNothingStarted(t *testing.T, f *cleanupFixture) {
	t.Helper()
	if len(f.h.calls) != 0 {
		t.Fatalf("herdr calls = %q, want none: d1 is a launch cmd, and no d2 may be launched", f.h.calls)
	}
	if _, err := os.Stat(filepath.Join(f.dir, "prompts")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stat prompts = %v, want no prompt file written", err)
	}
	if f.notice.Len() != 0 || f.gateCalls != 0 {
		t.Fatalf("notice %q and %d watch gate calls, want none", f.notice.String(), f.gateCalls)
	}
}

// AC6, a check that cannot read the ledger: start's first look reads d1, a
// launch cmd whose process ps finds gone, but the check's read under
// director.lock after the probe fails, because director.jsonl is a directory
// by then or director.lock cannot be opened. start returns that error as it
// is, not a refusal naming d1 and not a cleanup: ps is read by the probe
// only, nothing is written, and nothing is launched.
func TestStartDirector_ACheckThatCannotReadTheLedgerRefusesWithItsError(t *testing.T) {
	events := checkCmdRunning()
	for _, tc := range []struct {
		name string
		// setup makes the check's read under the lock fail and returns the
		// prefix of its error and what puts director.jsonl back (nil: none).
		setup func(t *testing.T, f *cleanupFixture) (cause string, restore func())
	}{
		{"director.jsonl is a directory under the lock", func(t *testing.T, f *cleanupFixture) (string, func()) {
			path := f.store.Path()
			aside := path + ".aside"
			f.o.directorCheckAfterRead = func(attempt int) {
				if attempt != 1 {
					t.Fatalf("the check tried again (try %d) after a read that failed", attempt)
				}
				if err := os.Rename(path, aside); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			return "org: director: read " + path + ": ", func() {
				if err := os.Remove(path); err != nil {
					t.Fatalf("remove the directory at director.jsonl (it must be empty): %v", err)
				}
				if err := os.Rename(aside, path); err != nil {
					t.Fatal(err)
				}
			}
		}},
		{"director.lock cannot be opened", func(t *testing.T, f *cleanupFixture) (string, func()) {
			lock := filepath.Join(f.dir, directorLockFile)
			if err := os.Mkdir(lock, 0o755); err != nil {
				t.Fatal(err)
			}
			return "org: open director lock file " + lock + ": ", nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCleanupFixture(t, events...)
			before := directorFileState(t, f.store)
			cause, restore := tc.setup(t, f)
			_, err := f.start()
			if restore != nil {
				restore()
			}
			if err == nil || !strings.HasPrefix(err.Error(), cause) {
				t.Fatalf("StartDirector err = %v, want the check's error, starting %q", err, cause)
			}
			var ae *DirectorAttachedError
			var ce *DirectorCleanupError
			if errors.As(err, &ae) || errors.As(err, &ce) || errors.Is(err, ErrDirectorAttached) {
				t.Fatalf("err = %#v, want the check's error alone, not a refusal of d1 or a cleanup", err)
			}
			if want := []int{checkAgentPID}; !slices.Equal(f.psReads, want) {
				t.Fatalf("ps reads = %v, want the check's probe only: %v", f.psReads, want)
			}
			if after := directorFileState(t, f.store); after != before {
				t.Fatalf("director.jsonl changed from %q to %q", before, after)
			}
			wantTrail(t, f.store, d1Trail(events)...)
			wantNothingStarted(t, f)
		})
	}
}

// AC6 on the launch_cmd path, a cleanup that cannot confirm the end: the
// check's ps read finds d1's process gone (process_ended), and the cleanup's
// own read fails, so stopDirectorProcess cannot tell whether the process
// ended. start refuses with start's *DirectorCleanupError, whose error wraps
// ErrDirectorProcessNotEnded and ErrDirectorProcessUnreadable and names the
// stop to run once ps can read the pid; nothing is recorded (no stopped, no
// d2), and nothing is launched. d1's pid is a process the test started,
// which is still running afterwards: nothing was sent to it.
func TestStartDirector_ALaunchCmdCleanupThatCannotConfirmTheEndRefusesStart(t *testing.T) {
	agent := startTestProcess(t, sleepPath(t), "30")
	proc := mustReadProcess(t, agent.pid())
	events := []DirectorEvent{checkAttached(DirectorLaunchCmd, true), checkEvent("d1", DirectorEventLaunched, func(ev *DirectorEvent) {
		ev.PID, ev.StartTime = agent.pid(), proc.StartTime
	})}
	f := newCleanupFixture(t, events...)
	read := f.o.DirectorProcessReader
	f.o.DirectorProcessReader = func(ctx context.Context, pid int) (DirectorProcess, error) {
		if len(f.psReads) == 0 {
			return read(ctx, pid) // gone: d1's pid has no entry in f.procs
		}
		f.psReads = append(f.psReads, pid)
		return brokenProcessReader(ctx, pid)
	}
	_, err := f.start()
	ce := wantCleanupRefused(t, f, err, DirectorLaunchCmd, ErrDirectorProcessNotEnded, "ralph org director stop --id d1", d1Trail(events))
	if !errors.Is(err, ErrDirectorProcessNotEnded) || !errors.Is(ce.CleanupErr, ErrDirectorProcessUnreadable) || ce.PID != agent.pid() {
		t.Fatalf("err = %v (pid %d), want it to wrap ErrDirectorProcessNotEnded and ErrDirectorProcessUnreadable for pid %d", err, ce.PID, agent.pid())
	}
	pid := strconv.Itoa(agent.pid())
	if !strings.Contains(err.Error(), "ps is broken") || !strings.Contains(err.Error(), "run ralph org director stop --id d1 again once ps can read pid "+pid) {
		t.Fatalf("err = %q, want ps's error and the stop to run once ps can read pid %s", err, pid)
	}
	if want := []int{agent.pid(), agent.pid()}; !slices.Equal(f.psReads, want) {
		t.Fatalf("ps reads = %v, want the check's and the cleanup's of d1's pid, and no starter read: %v", f.psReads, want)
	}
	wantNothingStarted(t, f)
	wantAliveWith(t, agent.pid(), proc.StartTime, "a cleanup that cannot confirm the end must send the process nothing")
}
