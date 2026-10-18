package org

import (
	"context"
	"os"
	"slices"
	"sync"
	"testing"
	"time"
)

// The test of CheckDirector's read under director.lock (plan
// 2026-10-18-org-director-liveness, D3 and AC3) against a writer that holds
// the lock, the real flock, while the check reads. It uses dirCheckFixture:
// ps and herdr are fakes, and the ledger is a t.TempDir.

// AC3, the read under the lock: d1 is a starting launch cmd. Right after the
// check's read, its launch script's check-in (MarkCmdLaunched) takes
// director.lock and stops in the middle of its write (the store's write
// seam), and the check's probe then finds d1's start process gone. The
// check's read under the lock must wait for the check-in: once the launched
// is written and the lock released, it finds d1 changed, throws start_ended
// away, and checks d1 again as running (alive process_found). A read that
// did not wait would find the version unchanged and keep start_ended. The
// test confirms for a bounded time that the check does not return while the
// lock is held; the order of the reads is set by the lock alone.
func TestCheckDirector_TheReadUnderTheLockWaitsForAWriterThatHoldsIt(t *testing.T) {
	f := newDirCheckFixture(t)
	writeDirectorEvents(t, f.store, checkAttached(DirectorLaunchCmd, true))
	// d1's start process has no entry, so ps finds it gone; the pid the
	// check-in records is alive with the start time it records.
	f.alive(checkAgentPID, checkAgentStart)
	writing, release := make(chan struct{}), make(chan struct{})
	f.store.write = func(file *os.File, b []byte) (int, error) {
		close(writing)
		<-release
		return file.Write(b)
	}
	// let lets the check-in finish its write; a test that fails before it
	// does lets it in its cleanup.
	var releaseOnce sync.Once
	let := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(let)
	checkInDone := make(chan error, 1)
	f.o.directorCheckAfterRead = func(attempt int) {
		f.attempts = append(f.attempts, attempt)
		if attempt != 1 {
			return
		}
		// The check-in reads d1's start process as still running, so it is
		// not refused for a start that has ended.
		go func() {
			checkInDone <- f.store.MarkCmdLaunched(DirectorCmdCheckIn{ID: "d1", Token: directorTestToken, PID: checkAgentPID, StartTime: checkAgentStart,
				Who: testDirectorWho, ReadStarter: func(context.Context, int) (DirectorProcess, error) {
					return DirectorProcess{State: DirectorProcessAlive, StartTime: checkStarterStart}, nil
				}})
		}()
		<-writing
	}
	probed := make(chan struct{})
	var probedOnce sync.Once
	read := f.o.DirectorProcessReader
	f.o.DirectorProcessReader = func(ctx context.Context, pid int) (DirectorProcess, error) {
		proc, err := read(ctx, pid)
		if pid == checkStarterPID {
			probedOnce.Do(func() { close(probed) })
		}
		return proc, err
	}
	type result struct {
		res DirectorCheckResult
		err error
	}
	done := make(chan result, 1)
	go func() {
		res, err := f.o.CheckDirector(f.store, checkNow)
		done <- result{res, err}
	}()
	select {
	case <-probed:
	case r := <-done:
		t.Fatalf("CheckDirector returned %+v, %v before its probe of the start process", r.res.Check, r.err)
	case <-time.After(10 * time.Second):
		t.Fatal("CheckDirector did not probe the start process within 10s")
	}
	select {
	case r := <-done:
		t.Fatalf("CheckDirector returned %+v, %v while the check-in held director.lock: its read under the lock did not wait", r.res.Check, r.err)
	case <-time.After(300 * time.Millisecond):
	}
	let()
	if err := <-checkInDone; err != nil {
		t.Fatalf("MarkCmdLaunched: %v", err)
	}
	r := <-done
	if r.err != nil {
		t.Fatalf("CheckDirector: %v", r.err)
	}
	wantBarrierResult(t, f, r.res, DirectorCheckHowProcessFound)
	if r.res.Registration.PID != checkAgentPID || len(f.h.calls) != 0 || !slices.Equal(f.psReads, []int{checkStarterPID, checkAgentPID}) {
		t.Fatalf("registration pid %d, herdr calls %q, ps reads %v; want the starter read by the first try and the checked-in pid by the second",
			r.res.Registration.PID, f.h.calls, f.psReads)
	}
	wantTrail(t, f.store, "attached d1", "launched d1")
}
