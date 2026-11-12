package org

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// gate.lock (gate_lock.go, plan docs/plans/active/2026-11-12-gate-lock-accept.md,
// D1 and D8): the wait for it, the lines a waiting gate writes, the marker,
// and the lock's file passed to the scripts, with real processes. Gate's and
// Regate's use of it are in gate_lock_gate_test.go and regate_test.go.

// shortenGateLockTimes sets gateLockPoll and gateLockRemind for the test.
func shortenGateLockTimes(t *testing.T, poll, remind time.Duration) {
	t.Helper()
	oldPoll, oldRemind := gateLockPoll, gateLockRemind
	gateLockPoll, gateLockRemind = poll, remind
	t.Cleanup(func() { gateLockPoll, gateLockRemind = oldPoll, oldRemind })
}

// holdGateLock takes gate.lock in stateDir as another gate would hold it,
// with m written into it as its marker (none when m is nil). release drops
// it; a cleanup calls it when the test does not.
func holdGateLock(t *testing.T, stateDir string, m *gateLockMarker) (release func()) {
	t.Helper()
	held, err := acquireHeldFileLock(stateDir, gateLockFile, "gate", 0)
	if err != nil {
		t.Fatal(err)
	}
	if m != nil {
		writeGateFile(t, stateDir, gateLockFile, gateLockMarkerLine(t, *m), 0o644)
	}
	released := false
	release = func() {
		if !released {
			released = true
			if err := held.release(); err != nil {
				t.Errorf("release the held gate.lock: %v", err)
			}
		}
	}
	t.Cleanup(release)
	return release
}

// gateLockMarkerLine is m as gate.lock holds it: one JSON line.
func gateLockMarkerLine(t *testing.T, m gateLockMarker) string {
	t.Helper()
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(data) + "\n"
}

// gateLockHeld is fileLockHeld of the gate.lock in stateDir, failing the test
// when the check cannot be made.
func gateLockHeld(t *testing.T, stateDir string) bool {
	t.Helper()
	held, err := fileLockHeld(gateLockPathIn(stateDir))
	if err != nil {
		t.Fatal(err)
	}
	return held
}

// gateLockTestOrg is an Org whose ledger is in a new state dir, with
// gateTestClock as its clock.
func gateLockTestOrg(t *testing.T) (*Org, string) {
	t.Helper()
	stateDir := filepath.Join(resolved(t, t.TempDir()), "state")
	return &Org{Manifest: NewManifestStoreAtPath(ManifestPathIn(stateDir)), Now: func() time.Time { return gateTestClock }}, stateDir
}

// gateLockOutcome is one acquireGateLock in a goroutine.
type gateLockOutcome struct {
	lock *gateLock
	wait gateLockWait
	err  error
}

// startAcquireGateLock runs acquireGateLock of m in a goroutine, its lines
// going to stderr.
func startAcquireGateLock(ctx context.Context, o *Org, m gateLockMarker, stderr io.Writer) <-chan gateLockOutcome {
	done := make(chan gateLockOutcome, 1)
	go func() {
		lock, wait, err := o.acquireGateLock(ctx, m, stderr, gateWords.prefix)
		done <- gateLockOutcome{lock, wait, err}
	}()
	return done
}

// waitGateLockOutcome waits up to 10s for done.
func waitGateLockOutcome(t *testing.T, done <-chan gateLockOutcome) gateLockOutcome {
	t.Helper()
	select {
	case got := <-done:
		return got
	case <-time.After(10 * time.Second):
		t.Fatal("acquireGateLock did not return within 10s")
	}
	return gateLockOutcome{}
}

// assertStillWaiting fails the test when done has an outcome.
func assertStillWaiting(t *testing.T, done <-chan gateLockOutcome) {
	t.Helper()
	select {
	case got := <-done:
		t.Fatalf("acquireGateLock = %+v while another held gate.lock; want it waiting", got)
	default:
	}
}

// gateLockWaitLine is the line acquireGateLock writes, with Gate's prefix,
// while holder holds the gate.lock at path and it has waited waited.
func gateLockWaitLine(path, waited, holder string) string {
	return "org: gate: waiting for " + path + " (waited " + waited + "): held by " + holder + "; a ledger runs one gate's scripts " +
		"at a time, so this one starts when that one ends\n"
}

// TestAcquireGateLock_AFreeLockIsTakenAtOnceWithItsMarker: with gate.lock
// free, acquireGateLock takes it at once, says nothing, reports no wait (no
// log line either), and writes its marker with the time it took the lock
// from the Org's clock and this pid; release empties the marker and drops the
// lock.
func TestAcquireGateLock_AFreeLockIsTakenAtOnceWithItsMarker(t *testing.T) {
	o, stateDir := gateLockTestOrg(t)
	var stderr strings.Builder
	lock, wait, err := o.acquireGateLock(context.Background(), gateLockMarker{Verb: gateWords.verb, OrgID: "org-a"}, &stderr, gateWords.prefix)
	if err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 || wait != (gateLockWait{}) || wait.logLines() != nil {
		t.Fatalf("a free gate.lock: stderr %q, wait %+v, log lines %q; want nothing said and no wait", stderr.String(), wait, wait.logLines())
	}
	path := gateLockPathIn(stateDir)
	want := `{"verb":"ralph org gate","org_id":"org-a","started_at":"2026-10-13T12:00:00Z","pid":` + strconv.Itoa(os.Getpid()) + "}\n"
	if got := fileOrAbsent(t, path); got != want {
		t.Fatalf("gate.lock holds %q\nwant           %q", got, want)
	}
	if !gateLockHeld(t, stateDir) {
		t.Fatal("gate.lock is not held after acquireGateLock")
	}
	if err := lock.release(); err != nil {
		t.Fatal(err)
	}
	if got := fileOrAbsent(t, path); got != "" || gateLockHeld(t, stateDir) {
		t.Fatalf("after release gate.lock holds %q, held %v; want it empty and free", got, gateLockHeld(t, stateDir))
	}
}

// TestAcquireGateLock_WaitsForTheHolderAndSaysWhoItIs covers AC1's lines:
// while another gate holds gate.lock, acquireGateLock does not return and
// says once, when the wait starts, which lock it waits for and the holder's
// verb, org_id, start, and pid; once the holder lets go it takes the lock,
// says after how long, and reports the wait and the holder for the log's
// header.
func TestAcquireGateLock_WaitsForTheHolderAndSaysWhoItIs(t *testing.T) {
	shortenGateLockTimes(t, 10*time.Millisecond, time.Hour)
	o, stateDir := gateLockTestOrg(t)
	pid := os.Getpid()
	release := holdGateLock(t, stateDir, &gateLockMarker{Verb: gateWords.verb, OrgID: "org-b", StartedAt: "2026-10-13T11:00:00Z", PID: pid})
	var stderr lockedBuffer
	done := startAcquireGateLock(context.Background(), o, gateLockMarker{Verb: gateWords.verb, OrgID: "org-a"}, &stderr)

	path := gateLockPathIn(stateDir)
	holder := fmt.Sprintf(`ralph org gate of org_id "org-b", started 2026-10-13T11:00:00Z by pid %d`, pid)
	waitForStderr(t, &stderr, gateLockWaitLine(path, "0s", holder))
	time.Sleep(100 * time.Millisecond)
	assertStillWaiting(t, done)
	if n := strings.Count(stderr.String(), "waiting for"); n != 1 {
		t.Fatalf("stderr has %d waiting lines for one holder within the reminder interval; want 1:\n%s", n, stderr.String())
	}
	release()
	got := waitGateLockOutcome(t, done)
	if got.err != nil {
		t.Fatal(got.err)
	}
	defer func() { _ = got.lock.release() }()
	if got.wait.waited <= 0 || got.wait.holder != holder {
		t.Fatalf("wait = %+v; want a wait for %q", got.wait, holder)
	}
	if lines := got.wait.logLines(); len(lines) != 1 || lines[0] != "# waited "+gateLockDuration(got.wait.waited)+" for gate.lock: "+holder {
		t.Fatalf("log lines = %q; want the wait for %q", lines, holder)
	}
	if took := "org: gate: took " + path + " after waiting " + gateLockDuration(got.wait.waited) + "\n"; !strings.HasSuffix(stderr.String(), took) {
		t.Fatalf("stderr = %q; want it to end with %q", stderr.String(), took)
	}
	if m := fileOrAbsent(t, path); !strings.Contains(m, `"org_id":"org-a"`) {
		t.Fatalf("gate.lock holds %q after the wait; want the marker of org-a", m)
	}
}

// TestAcquireGateLock_SaysSoWhenTheHolderChanges: a holder whose marker
// changes (another gate took the lock, or a regate wrote its org) gets a
// line of its own, with the regate's branch and org.
func TestAcquireGateLock_SaysSoWhenTheHolderChanges(t *testing.T) {
	shortenGateLockTimes(t, 10*time.Millisecond, time.Hour)
	o, stateDir := gateLockTestOrg(t)
	pid := os.Getpid()
	release := holdGateLock(t, stateDir, &gateLockMarker{Verb: gateWords.verb, OrgID: "org-b", StartedAt: "2026-10-13T11:00:00Z", PID: pid})
	var stderr lockedBuffer
	done := startAcquireGateLock(context.Background(), o, gateLockMarker{Verb: gateWords.verb, OrgID: "org-a"}, &stderr)
	waitForStderr(t, &stderr, `held by ralph org gate of org_id "org-b"`)

	writeGateFile(t, stateDir, gateLockFile, gateLockMarkerLine(t, gateLockMarker{Verb: "ralph org regate", Branch: "feat/c", OrgID: "c",
		StartedAt: "2026-10-13T11:30:00Z", PID: pid}), 0o644)
	waitForStderr(t, &stderr, fmt.Sprintf(`): held by ralph org regate of the branch "feat/c" (org_id "c"), started 2026-10-13T11:30:00Z by pid %d; `, pid))
	if n := strings.Count(stderr.String(), "waiting for"); n != 2 {
		t.Fatalf("stderr has %d waiting lines for two holders; want 2:\n%s", n, stderr.String())
	}
	release()
	got := waitGateLockOutcome(t, done)
	if got.err != nil || !strings.Contains(got.wait.holder, `regate of the branch "feat/c"`) {
		t.Fatalf("acquireGateLock = %+v; want the lock, with the last holder seen", got)
	}
	_ = got.lock.release()
}

// TestAcquireGateLock_RemindsWhileTheSameHolderHoldsIt: with the same holder,
// a line comes again every gateLockRemind, not at every try of the lock.
func TestAcquireGateLock_RemindsWhileTheSameHolderHoldsIt(t *testing.T) {
	const remind = 300 * time.Millisecond
	shortenGateLockTimes(t, 10*time.Millisecond, remind)
	o, stateDir := gateLockTestOrg(t)
	release := holdGateLock(t, stateDir, &gateLockMarker{Verb: gateWords.verb, OrgID: "org-b", StartedAt: "2026-10-13T11:00:00Z", PID: os.Getpid()})
	var stderr lockedBuffer
	done := startAcquireGateLock(context.Background(), o, gateLockMarker{Verb: gateWords.verb, OrgID: "org-a"}, &stderr)
	waitForStderr(t, &stderr, "waiting for")
	first := time.Now()
	deadline := first.Add(10 * time.Second)
	for strings.Count(stderr.String(), "waiting for") < 3 {
		if time.Now().After(deadline) {
			t.Fatalf("stderr has no third waiting line within 10s:\n%s", stderr.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
	if took := time.Since(first); took < 2*remind-50*time.Millisecond {
		t.Fatalf("three waiting lines within %s of the first; want one every %s:\n%s", took, remind, stderr.String())
	}
	release()
	got := waitGateLockOutcome(t, done)
	if got.err != nil {
		t.Fatal(got.err)
	}
	_ = got.lock.release()
	lines := strings.Split(strings.TrimSuffix(stderr.String(), "\n"), "\n")
	if !strings.Contains(lines[0], "(waited 0s)") {
		t.Errorf("first line %q; want it written when the wait starts", lines[0])
	}
	for _, line := range lines[:3] {
		if !strings.Contains(line, `held by ralph org gate of org_id "org-b"`) {
			t.Errorf("line %q; want the same holder", line)
		}
	}
}

// TestAcquireGateLock_ACancelledWaitHoldsAndWritesNothing covers AC2 at the
// lock: ctx done during the wait returns an error that wraps ctx's cause and
// says the wait was stopped, leaves the holder's marker as it was, and holds
// nothing, so the lock is free once the holder lets go.
func TestAcquireGateLock_ACancelledWaitHoldsAndWritesNothing(t *testing.T) {
	shortenGateLockTimes(t, 10*time.Millisecond, time.Hour)
	o, stateDir := gateLockTestOrg(t)
	marker := gateLockMarker{Verb: gateWords.verb, OrgID: "org-b", StartedAt: "2026-10-13T11:00:00Z", PID: os.Getpid()}
	release := holdGateLock(t, stateDir, &marker)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stderr lockedBuffer
	done := startAcquireGateLock(ctx, o, gateLockMarker{Verb: gateWords.verb, OrgID: "org-a"}, &stderr)
	waitForStderr(t, &stderr, "waiting for")
	cancel()
	got := waitGateLockOutcome(t, done)
	if !errors.Is(got.err, context.Canceled) || !strings.Contains(got.err.Error(), "org: acquire gate lock "+gateLockPathIn(stateDir)+": the wait was stopped") ||
		got.lock != nil {
		t.Fatalf("acquireGateLock = %+v; want the stopped wait and no lock", got)
	}
	if m := fileOrAbsent(t, gateLockPathIn(stateDir)); m != gateLockMarkerLine(t, marker) {
		t.Fatalf("gate.lock holds %q after a cancelled wait; want the holder's marker as it was", m)
	}
	release()
	if gateLockHeld(t, stateDir) {
		t.Fatal("gate.lock is held after the holder let go; want the cancelled wait to hold nothing")
	}
}

// TestAcquireGateLock_SaysAMarkerIsNotWrittenYet: a holder that has not
// written its marker yet is named so on the waiting line.
func TestAcquireGateLock_SaysAMarkerIsNotWrittenYet(t *testing.T) {
	shortenGateLockTimes(t, 10*time.Millisecond, time.Hour)
	o, stateDir := gateLockTestOrg(t)
	release := holdGateLock(t, stateDir, nil)
	var stderr lockedBuffer
	done := startAcquireGateLock(context.Background(), o, gateLockMarker{Verb: gateWords.verb, OrgID: "org-a"}, &stderr)
	waitForStderr(t, &stderr, gateLockWaitLine(gateLockPathIn(stateDir), "0s", "a gate whose marker is not written yet"))
	release()
	got := waitGateLockOutcome(t, done)
	if got.err != nil {
		t.Fatal(got.err)
	}
	_ = got.lock.release()
}

// TestGateLockHolder: how a waiting gate names the holder from gate.lock's
// marker: Gate's with its org_id, a regate's with its branch and, once found,
// its org_id, a marker not written yet, one that cannot be read, and a holder
// whose pid has exited, whose script still holds the lock (D8).
func TestGateLockHolder(t *testing.T) {
	dir := t.TempDir()
	pid := os.Getpid()
	gone := exec.Command("true")
	if err := gone.Run(); err != nil {
		t.Fatal(err)
	}
	gonePID := gone.Process.Pid
	line := func(m gateLockMarker) string { return gateLockMarkerLine(t, m) }
	for _, tc := range []struct {
		name, content, want string
	}{
		{"gate", line(gateLockMarker{Verb: "ralph org gate", OrgID: "org-a", StartedAt: "2026-10-13T11:00:00Z", PID: pid}),
			fmt.Sprintf(`ralph org gate of org_id "org-a", started 2026-10-13T11:00:00Z by pid %d`, pid)},
		{"regate before its first check", line(gateLockMarker{Verb: "ralph org regate", Branch: "feat/a", StartedAt: "2026-10-13T11:00:00Z", PID: pid}),
			fmt.Sprintf(`ralph org regate of the branch "feat/a", started 2026-10-13T11:00:00Z by pid %d`, pid)},
		{"regate with its org", line(gateLockMarker{Verb: "ralph org regate", Branch: "feat/a", OrgID: "a", StartedAt: "2026-10-13T11:00:00Z", PID: pid}),
			fmt.Sprintf(`ralph org regate of the branch "feat/a" (org_id "a"), started 2026-10-13T11:00:00Z by pid %d`, pid)},
		{"a holder whose pid has exited", line(gateLockMarker{Verb: "ralph org gate", OrgID: "org-a", StartedAt: "2026-10-13T11:00:00Z", PID: gonePID}),
			fmt.Sprintf(`ralph org gate of org_id "org-a", started 2026-10-13T11:00:00Z by pid %d (pid %d has exited, so a script it started still holds the lock)`,
				gonePID, gonePID)},
		{"empty", "", "a gate whose marker is not written yet"},
		{"a line break only", "\n", "a gate whose marker is not written yet"},
		{"not JSON", "gate\n", "a gate whose marker cannot be read (invalid character 'g' looking for beginning of value)"},
		{"no verb", `{"org_id":"org-a","pid":7}` + "\n", "a gate whose marker cannot be read (it names no verb or pid, or holds a control character)"},
		{"no pid", `{"verb":"ralph org gate"}` + "\n", "a gate whose marker cannot be read (it names no verb or pid, or holds a control character)"},
		{"a control character", line(gateLockMarker{Verb: "ralph org gate\x1b[2J", PID: pid}),
			"a gate whose marker cannot be read (it names no verb or pid, or holds a control character)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, strings.ReplaceAll(tc.name, " ", "-"))
			if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}
			if got := gateLockHolder(path); got != tc.want {
				t.Fatalf("gateLockHolder(%q) = %q\nwant %q", tc.content, got, tc.want)
			}
		})
	}
	if got := gateLockHolder(filepath.Join(dir, "missing")); !strings.HasPrefix(got, "a gate whose marker cannot be read (") {
		t.Fatalf("gateLockHolder of a missing file = %q; want a marker that cannot be read", got)
	}
}

// The child process of TestGateLock_AScriptKeepsTheLockAfterItsRalphIsKilled.
const (
	gateLockHelperStateEnv = "RALPH_TEST_GATE_LOCK_HELPER_STATE"
	gateLockHelperRootEnv  = "RALPH_TEST_GATE_LOCK_HELPER_ROOT"
)

// TestGateLockHelperProcess is not a test of its own: run by
// TestGateLock_AScriptKeepsTheLockAfterItsRalphIsKilled as a child process
// with gateLockHelperStateEnv set, it takes gate.lock in that state dir, as a
// gate does, and runs ./run.sh in gateLockHelperRootEnv with the lock's file
// (runGateScript), then exits. Without the variable it does nothing.
func TestGateLockHelperProcess(t *testing.T) {
	stateDir := os.Getenv(gateLockHelperStateEnv)
	if stateDir == "" {
		return
	}
	o := &Org{Manifest: NewManifestStoreAtPath(ManifestPathIn(stateDir))}
	lock, _, err := o.acquireGateLock(context.Background(), gateLockMarker{Verb: gateWords.verb, OrgID: "helper"}, os.Stderr, gateWords.prefix)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	code, _, err := runGateScript(context.Background(), os.Getenv(gateLockHelperRootEnv), "run.sh", os.Environ(), os.Stderr, gateWords.verb, lock.file())
	_ = lock.release()
	if err != nil || code != 0 {
		fmt.Fprintf(os.Stderr, "run.sh: exit %d, %v\n", code, err)
		os.Exit(3)
	}
	os.Exit(0)
}

// TestGateLock_AScriptKeepsTheLockAfterItsRalphIsKilled covers AC14 (D8) with
// real processes: a child process takes gate.lock as a gate does and runs a
// script with the lock's file; killed with SIGKILL while the script runs, it
// releases nothing, so gate.lock stays held, and a waiting gate's line says
// the holder's pid has exited, until the script ends; then the lock is free.
func TestGateLock_AScriptKeepsTheLockAfterItsRalphIsKilled(t *testing.T) {
	tmp := resolved(t, t.TempDir())
	stateDir, root, out := filepath.Join(tmp, "state"), filepath.Join(tmp, "root"), filepath.Join(tmp, "out")
	mkdirAll(t, out)
	writeGateFile(t, root, "run.sh", "#!/bin/sh\necho $$ > \"$GATE_LOCK_TEST_OUT/pid\"\ntouch \"$GATE_LOCK_TEST_OUT/started\"\n"+
		"sleep 2\ntouch \"$GATE_LOCK_TEST_OUT/done\"\n", 0o755)
	log, err := os.Create(filepath.Join(tmp, "helper.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = log.Close() }()
	helper := exec.Command(os.Args[0], "-test.run=^TestGateLockHelperProcess$")
	helper.Env = append(os.Environ(), gateLockHelperStateEnv+"="+stateDir, gateLockHelperRootEnv+"="+root, "GATE_LOCK_TEST_OUT="+out)
	helper.Stdout, helper.Stderr = log, log
	if err := helper.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = helper.Process.Kill()
		_ = helper.Wait()
		if pid, err := strconv.Atoi(strings.TrimSpace(fileOrAbsent(t, filepath.Join(out, "pid")))); err == nil {
			_ = syscall.Kill(-pid, syscall.SIGKILL)
		}
	})
	waitForFile(t, filepath.Join(out, "started"))
	if !gateLockHeld(t, stateDir) {
		t.Fatalf("gate.lock is not held while the helper's script runs; helper log:\n%s", fileOrAbsent(t, log.Name()))
	}

	if err := helper.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = helper.Wait()
	path := gateLockPathIn(stateDir)
	heldChecks := 0
	for {
		held := gateLockHeld(t, stateDir)
		if _, err := os.Stat(filepath.Join(out, "done")); err == nil {
			break
		}
		if !held {
			t.Fatal("gate.lock was free while the script still ran after its ralph was killed; want the script to keep it")
		}
		if heldChecks++; heldChecks == 1 {
			want := fmt.Sprintf("(pid %d has exited, so a script it started still holds the lock)", helper.Process.Pid)
			if holder := gateLockHolder(path); !strings.Contains(holder, want) {
				t.Errorf("gateLockHolder = %q; want it to say %q", holder, want)
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if heldChecks == 0 {
		t.Fatal("the script ended before gate.lock was checked after the kill; want the check while it ran")
	}
	deadline := time.Now().Add(10 * time.Second)
	for gateLockHeld(t, stateDir) {
		if time.Now().After(deadline) {
			t.Fatal("gate.lock is still held 10s after the script ended")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestGateLock_ReleaseFreesTheLockAChildStillHasOpen covers AC14's normal
// end: a process the script left running with gate.lock's file open does not
// keep the lock once the gate releases it (LOCK_UN frees the lock for every
// descriptor of the open file).
func TestGateLock_ReleaseFreesTheLockAChildStillHasOpen(t *testing.T) {
	o, stateDir := gateLockTestOrg(t)
	lock, _, err := o.acquireGateLock(context.Background(), gateLockMarker{Verb: gateWords.verb, OrgID: "org-a"}, io.Discard, gateWords.prefix)
	if err != nil {
		t.Fatal(err)
	}
	child := exec.Command("sleep", "30")
	child.ExtraFiles = []*os.File{lock.file()}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = child.Process.Kill()
		_ = child.Wait()
	})
	if err := lock.release(); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Kill(child.Process.Pid, 0); err != nil {
		t.Fatalf("the child that has the lock's file open is gone (kill 0: %v); want it running", err)
	}
	if gateLockHeld(t, stateDir) {
		t.Fatal("gate.lock is held after the gate released it while a child still has its file open; want it free")
	}
	if m := fileOrAbsent(t, gateLockPathIn(stateDir)); m != "" {
		t.Fatalf("gate.lock holds %q after the release; want it empty", m)
	}
}
