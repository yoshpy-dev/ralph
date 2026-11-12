package org

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

// Gate and gate.lock (plan docs/plans/active/2026-11-12-gate-lock-accept.md,
// AC1 to AC3): Gate waits for another gate's lock before its scripts, makes
// its checks again once it holds it, and stops waiting with nothing recorded
// when it is interrupted, on the fixtures of gate_test.go.

// gateOutcome is one Gate in a goroutine.
type gateOutcome struct {
	res GateResult
	err error
}

// startGate runs Gate of org-a from the fixture's worktree in a goroutine,
// with env and its lines going to stderr.
func (fx *gateFixture) startGate(ctx context.Context, env []string, stderr *lockedBuffer) <-chan gateOutcome {
	done := make(chan gateOutcome, 1)
	go func() {
		res, err := fx.o.Gate(ctx, GateParams{OrgID: "org-a", Dir: fx.worktree, Env: env, Stderr: stderr})
		done <- gateOutcome{res, err}
	}()
	return done
}

// waitGate waits up to limit for done.
func waitGate(t *testing.T, done <-chan gateOutcome, limit time.Duration) gateOutcome {
	t.Helper()
	select {
	case got := <-done:
		return got
	case <-time.After(limit):
		t.Fatalf("gate did not return within %s", limit)
	}
	return gateOutcome{}
}

// gateLockOrgB is the marker of a gate of org-b that holds gate.lock in the
// tests, alive (this process's pid).
func gateLockOrgB() *gateLockMarker {
	return &gateLockMarker{Verb: gateWords.verb, OrgID: "org-b", StartedAt: "2026-10-13T11:00:00Z", PID: os.Getpid()}
}

// gateLockOrgBHolder is how a waiting gate names gateLockOrgB.
func gateLockOrgBHolder() string {
	return fmt.Sprintf(`ralph org gate of org_id "org-b", started 2026-10-13T11:00:00Z by pid %d`, os.Getpid())
}

// TestGate_WaitsForGateLockAndGatesTheHeadItFindsAfterTheWait covers AC1's
// wait and AC3's HEAD: while another gate holds gate.lock, Gate runs no
// script and says on its Stderr whose gate it waits for; HEAD moves on
// meanwhile, and once the lock is free Gate checks again and gates the new
// HEAD (a pass, not head_moved), with the wait in its log's header after the
// base line, and releases the lock with its marker emptied when it returns.
func TestGate_WaitsForGateLockAndGatesTheHeadItFindsAfterTheWait(t *testing.T) {
	shortenGateLockTimes(t, 20*time.Millisecond, time.Hour)
	fx := newGateFixture(t)
	release := holdGateLock(t, fx.stateDir, gateLockOrgB())
	var stderr lockedBuffer
	done := fx.startGate(context.Background(), fx.env, &stderr)
	path := gateLockPathIn(fx.stateDir)
	waitForStderr(t, &stderr, gateLockWaitLine(path, "0s", gateLockOrgBHolder()))
	if calls := fx.calls(t); calls != nil {
		t.Fatalf("scripts run while gate.lock was held = %v; want none", calls)
	}

	gateCommit(t, fx.worktree, "moved while the gate waited")
	head, base := fx.head(t)
	release()
	got := waitGate(t, done, 60*time.Second)
	want := GateResult{Passed: true, Head: head, Base: base, BaseRef: "refs/remotes/origin/main", Branch: "feat/auth",
		LogPath: filepath.Join(fx.stateDir, gateLogDir, gateTestLog)}
	if got.err != nil || !reflect.DeepEqual(got.res, want) {
		t.Fatalf("gate = %+v, %v\nwant   %+v (the HEAD after the wait)", got.res, got.err, want)
	}
	if events := fx.gateEvents(t); len(events) != 1 || events[0].Event != EventGatePassed || !strings.HasPrefix(events[0].Details, "head="+head+" ") {
		t.Fatalf("gate events = %+v; want one gate_passed of %s", events, head)
	}
	log := fileOrAbsent(t, want.LogPath)
	if !strings.Contains(log, "# base: refs/remotes/origin/main "+base+"\n# waited ") ||
		!strings.Contains(log, " for gate.lock: "+gateLockOrgBHolder()+"\n\n==> ralph org gate: running") {
		t.Fatalf("log header has no wait for gate.lock after the base line:\n%s", log)
	}
	if !strings.Contains(stderr.String(), "org: gate: took "+path+" after waiting ") {
		t.Fatalf("stderr = %q; want the line that the gate took gate.lock", stderr.String())
	}
	if gateLockHeld(t, fx.stateDir) || fileOrAbsent(t, path) != "" {
		t.Fatalf("after the gate gate.lock is held %v and holds %q; want it free and empty", gateLockHeld(t, fx.stateDir), fileOrAbsent(t, path))
	}
}

// TestGate_DisbandedWhileWaitingForGateLockRunsNothing covers AC3's disband:
// an org disbanded while its gate waits for gate.lock is refused once the
// lock is free, before any script or log, with nothing recorded.
func TestGate_DisbandedWhileWaitingForGateLockRunsNothing(t *testing.T) {
	shortenGateLockTimes(t, 20*time.Millisecond, time.Hour)
	fx := newGateFixture(t)
	release := holdGateLock(t, fx.stateDir, gateLockOrgB())
	var stderr lockedBuffer
	done := fx.startGate(context.Background(), fx.env, &stderr)
	waitForStderr(t, &stderr, "org: gate: waiting for ")
	if err := fx.o.Manifest.Append(ManifestEvent{TS: fx.o.now(), OrgID: "org-a", Event: EventDisbanded}); err != nil {
		t.Fatal(err)
	}
	before := fileOrAbsent(t, fx.o.Manifest.Path())
	release()
	got := waitGate(t, done, 60*time.Second)
	fx.assertGateRefused(t, got.res, got.err, before, []string{`org_id "org-a" was disbanded and has not started again`}, nil)
	if gateLockHeld(t, fx.stateDir) {
		t.Fatal("gate.lock is held after a refused gate returned")
	}
}

// TestGate_InterruptedWhileWaitingForGateLockRecordsNothing covers AC2: ctx
// done while Gate waits for gate.lock ends the wait at once with the error
// that nothing was recorded, before any script or log, and leaves the
// holder's lock and marker as they were.
func TestGate_InterruptedWhileWaitingForGateLockRecordsNothing(t *testing.T) {
	shortenGateLockTimes(t, 20*time.Millisecond, time.Hour)
	fx := newGateFixture(t)
	holder := gateLockOrgB()
	holdGateLock(t, fx.stateDir, holder)
	before := fileOrAbsent(t, fx.o.Manifest.Path())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stderr lockedBuffer
	done := fx.startGate(ctx, fx.env, &stderr)
	waitForStderr(t, &stderr, "org: gate: waiting for ")
	begin := time.Now()
	cancel()
	got := waitGate(t, done, 10*time.Second)
	if took := time.Since(begin); took > 3*time.Second {
		t.Errorf("gate took %s to return after the interrupt; want the wait to end at once", took)
	}
	if got.err == nil || got.err.Error() != "org: gate: interrupted (context canceled) before the scripts ran: nothing was recorded" ||
		!reflect.DeepEqual(got.res, GateResult{}) {
		t.Fatalf("gate = %+v, %v; want the interrupt error and nothing else", got.res, got.err)
	}
	if after := fileOrAbsent(t, fx.o.Manifest.Path()); after != before {
		t.Fatalf("an interrupted gate changed the manifest:\nbefore %s\nafter  %s", before, after)
	}
	if _, err := os.Stat(filepath.Join(fx.stateDir, gateLogDir)); !os.IsNotExist(err) || fx.calls(t) != nil {
		t.Fatalf("an interrupted gate wrote a log (stat: %v) or ran %v", err, fx.calls(t))
	}
	if !gateLockHeld(t, fx.stateDir) || fileOrAbsent(t, gateLockPathIn(fx.stateDir)) != gateLockMarkerLine(t, *holder) {
		t.Fatalf("gate.lock after the interrupt: held %v, %q; want the holder's lock and marker as they were",
			gateLockHeld(t, fx.stateDir), fileOrAbsent(t, gateLockPathIn(fx.stateDir)))
	}
}

// TestGate_TwoGatesRunTheirScriptsOneAtATime covers AC1 with two gates of the
// same ledger started at once: their scripts never overlap (each gate's
// static and test scripts run back to back), the one that waited starts its
// scripts only after the other's record, and its stderr and log say so.
func TestGate_TwoGatesRunTheirScriptsOneAtATime(t *testing.T) {
	shortenGateLockTimes(t, 20*time.Millisecond, time.Hour)
	fx := newGateFixture(t)
	spans := filepath.Join(fx.out, "spans")
	for _, name := range []string{"static", "test"} {
		hook := `echo "begin $GATE_ID $name" >> '` + spans + "'\n"
		if name == "static" {
			hook += `grep -c '"event":"gate_passed"' '` + fx.o.Manifest.Path() + `' > "$GATE_TEST_OUT/passed-before-$GATE_ID"` + "\n"
		}
		fx.hook(t, name, hook+"sleep 0.3\necho \"end $GATE_ID $name\" >> '"+spans+"'\n")
	}
	ids := []string{"a", "b"}
	stderr := map[string]*lockedBuffer{}
	done := map[string]<-chan gateOutcome{}
	for _, id := range ids {
		stderr[id] = &lockedBuffer{}
		done[id] = fx.startGate(context.Background(), append(slices.Clone(fx.env), "GATE_ID="+id), stderr[id])
	}
	got := map[string]gateOutcome{}
	for _, id := range ids {
		got[id] = waitGate(t, done[id], 120*time.Second)
		if got[id].err != nil || !got[id].res.Passed {
			t.Fatalf("gate %s = %+v, %v; want a pass", id, got[id].res, got[id].err)
		}
	}

	lines := strings.Split(strings.TrimSpace(fileOrAbsent(t, spans)), "\n")
	if len(lines) != 8 {
		t.Fatalf("spans = %q; want the begin and end of four scripts", lines)
	}
	first := strings.Fields(lines[0])[1]
	for i, line := range lines {
		id := first
		if i >= 4 {
			id = map[string]string{"a": "b", "b": "a"}[first]
		}
		name := []string{"static", "test"}[i/2%2]
		if want := []string{"begin", "end"}[i%2] + " " + id + " " + name; line != want {
			t.Fatalf("span %d = %q, want %q: the two gates' scripts overlapped:\n%s", i, line, want, strings.Join(lines, "\n"))
		}
	}
	waited := map[string]string{"a": "b", "b": "a"}[first]
	if n := fileOrAbsent(t, filepath.Join(fx.out, "passed-before-"+waited)); strings.TrimSpace(n) != "1" {
		t.Errorf("gate_passed records when the second gate's scripts started = %q; want the first gate's one", n)
	}
	if !strings.Contains(stderr[waited].String(), "org: gate: waiting for ") || strings.Contains(stderr[first].String(), "waiting for") {
		t.Errorf("stderr of the gate that ran first %q, of the one that waited %q; want only the second to wait", stderr[first].String(),
			stderr[waited].String())
	}
	if log := fileOrAbsent(t, got[waited].res.LogPath); !strings.Contains(log, " for gate.lock: ralph org gate of org_id \"org-a\", started ") {
		t.Errorf("log of the gate that waited has no wait for the other gate:\n%s", log)
	}
	if events := fx.gateEvents(t); len(events) != 2 {
		t.Errorf("gate events = %+v; want two gate_passed", events)
	}
}
