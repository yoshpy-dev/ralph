package org

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Where Spawn, StartFeature and StartDirector run the herdr server gate
// (plan 2026-11-06-org-herdr-autostart, D1 and D2; AC3, AC4 and AC9 at the
// org layer). The Org of each test sets HerdrServerGate to a stub; every
// other org-layer test leaves it nil, which checks nothing.

// errHerdrServerDown is what the stub gates return for a herdr server that
// is not running and could not be started.
var errHerdrServerDown = errors.New("herdr server is not running and could not be started")

// stubHerdrServerGate is an Org.HerdrServerGate that counts its calls,
// appends "herdr_server_gate" to log (when set) on each, and returns err.
type stubHerdrServerGate struct {
	calls int
	err   error
	log   *[]string
}

func (g *stubHerdrServerGate) gate() error {
	g.calls++
	if g.log != nil {
		*g.log = append(*g.log, "herdr_server_gate")
	}
	return g.err
}

// spawnLockExists reports whether the spawn lock file of orgID/seatID is in
// o's ledger: Spawn creates it when it takes the lock and leaves it there.
func spawnLockExists(t *testing.T, o *Org, orgID, seatID string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(filepath.Dir(o.Manifest.Path()), spawnLockName(orgID, seatID)))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("stat the spawn lock of %s/%s: %v", orgID, seatID, err)
	}
	return err == nil
}

// A real spawn whose gate fails is SpawnOutcomeFailed with the gate's error
// (D2) and writes nothing: no manifest event, no receipt, no herdr or agmsg
// call, and no spawn lock taken. A gate that passes is asked once per spawn
// and lets it go on.
func TestOrgSpawn_HerdrServerGate_AFailureRecordsAndCallsNothing(t *testing.T) {
	o, h, a := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-0")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn with no gate failed: %+v", r)
	}
	gate := &stubHerdrServerGate{err: errHerdrServerDown}
	o.HerdrServerGate = gate.gate
	before := takeSpawnSnapshot(t, o, h, a)

	p := mustSpawnParams("org-a", "seat-1")
	r := o.Spawn(p)
	if r.Outcome != SpawnOutcomeFailed || !errors.Is(r.Err, errHerdrServerDown) {
		t.Fatalf("spawn with a failing herdr server gate = %+v; want failed with the gate's error", r)
	}
	if r.ModelReceipt != (Receipt{}) {
		t.Errorf("ModelReceipt = %+v; want none", r.ModelReceipt)
	}
	if after := takeSpawnSnapshot(t, o, h, a); after != before {
		t.Errorf("a spawn the herdr server gate failed recorded or called something: %+v -> %+v", before, after)
	}
	if spawnLockExists(t, o, "org-a", "seat-1") {
		t.Error("a spawn the herdr server gate failed took the seat's spawn lock")
	}
	if gate.calls != 1 {
		t.Errorf("gate called %d times; want once", gate.calls)
	}

	gate.err = nil
	if r := o.Spawn(p); r.Err != nil || r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn with a passing herdr server gate = %+v; want it to go on", r)
	}
	if gate.calls != 2 {
		t.Errorf("gate called %d times after the second spawn; want twice", gate.calls)
	}
}

// The gate runs before Spawn's idempotent early return (D1: before any
// manifest read), so a no-op respawn of an active seat asks it too. A gate
// that fails makes the respawn SpawnOutcomeFailed with the gate's error and
// changes nothing: no manifest event, no receipt, no herdr or agmsg call. A
// gate that passes is asked once, and the respawn answers idempotently with
// the seat as before, still writing and calling nothing.
func TestOrgSpawn_HerdrServerGate_ARespawnOfAnActiveSeatAsksItToo(t *testing.T) {
	o, h, a := testOrg(t)
	p := mustSpawnParams("org-a", "seat-1")
	first := o.Spawn(p)
	if first.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn with no gate failed: %+v", first)
	}
	gate := &stubHerdrServerGate{err: errHerdrServerDown}
	o.HerdrServerGate = gate.gate
	before := takeSpawnSnapshot(t, o, h, a)

	r := o.Spawn(p)
	if r.Outcome != SpawnOutcomeFailed || !errors.Is(r.Err, errHerdrServerDown) {
		t.Fatalf("respawn of an active seat with a failing herdr server gate = %+v; want failed with the gate's error", r)
	}
	if r.ModelReceipt != (Receipt{}) {
		t.Errorf("ModelReceipt = %+v; want none", r.ModelReceipt)
	}
	if after := takeSpawnSnapshot(t, o, h, a); after != before {
		t.Errorf("a respawn the herdr server gate failed recorded or called something: %+v -> %+v", before, after)
	}
	if gate.calls != 1 {
		t.Errorf("gate called %d times by the failed respawn; want once", gate.calls)
	}

	gate.err = nil
	gate.calls = 0
	r = o.Spawn(p)
	if r.Outcome != SpawnOutcomeIdempotent || r.Err != nil || r.Seat.SeatID != "seat-1" || r.Seat.PaneID != first.Seat.PaneID {
		t.Fatalf("respawn of an active seat with a passing herdr server gate = %+v; want the idempotent answer with pane %q", r, first.Seat.PaneID)
	}
	if gate.calls != 1 {
		t.Errorf("gate called %d times by the idempotent respawn; want once", gate.calls)
	}
	if after := takeSpawnSnapshot(t, o, h, a); after != before {
		t.Errorf("an idempotent respawn recorded or called something: %+v -> %+v", before, after)
	}
}

// A dry run asks herdr nothing, so it never asks the gate either (spec
// FR-A1, AC3): a gate that would fail does not stop it, and the dry run
// records its dry-run trail with no herdr call.
func TestOrgSpawn_HerdrServerGate_ADryRunDoesNotAskIt(t *testing.T) {
	o, h, _ := testOrg(t)
	gate := &stubHerdrServerGate{err: errHerdrServerDown}
	o.HerdrServerGate = gate.gate

	p := mustSpawnParams("org-a", "seat-1")
	p.DryRun = true
	r := o.Spawn(p)
	if r.Err != nil || r.Outcome != SpawnOutcomeSpawned || !r.Seat.DryRun {
		t.Fatalf("dry-run spawn with a failing herdr server gate = %+v; want the dry run to go on", r)
	}
	if gate.calls != 0 {
		t.Errorf("gate called %d times by a dry run; want none", gate.calls)
	}
	if last := lastEvent(t, o); last.Event != EventSpawned || !last.DryRun {
		t.Errorf("last event = %+v; want the dry run's spawned", last)
	}
	if len(h.calls) != 0 {
		t.Errorf("a dry run called herdr: %q", h.calls)
	}
}

// The gate runs after the input checks and the watch gate, and before the
// manifest is read: an invalid org_id and a refusing watch gate are refused
// for themselves with no herdr server gate call, both gates are asked in
// that order when they pass, and with a manifest that cannot be read the
// failure is the gate's, not the read's.
func TestOrgSpawn_HerdrServerGate_RunsAfterTheInputChecksAndTheWatchGate(t *testing.T) {
	o, _, _ := testOrg(t)
	var order []string
	gate := &stubHerdrServerGate{err: errHerdrServerDown, log: &order}
	o.HerdrServerGate = gate.gate

	r := o.Spawn(mustSpawnParams("Bad Org", "seat-1"))
	if r.Outcome != SpawnOutcomeRejected || !strings.Contains(r.Err.Error(), "invalid org_id") {
		t.Fatalf("spawn with an invalid org_id = %+v; want the identifier's rejection", r)
	}
	if gate.calls != 0 {
		t.Errorf("gate called %d times for an invalid org_id; want none", gate.calls)
	}

	watch := refusingWatchGate()
	o.WatchGate = watch.gate
	r = o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if r.Outcome != SpawnOutcomeRejected || !errors.Is(r.Err, ErrNoRunningWatch) || errors.Is(r.Err, errHerdrServerDown) {
		t.Fatalf("spawn with a refusing watch gate = %+v; want the watch gate's rejection", r)
	}
	if gate.calls != 0 {
		t.Errorf("herdr server gate called %d times after the watch gate refused; want none", gate.calls)
	}

	o.WatchGate = func() error {
		order = append(order, "watch_gate")
		return nil
	}
	gate.err = nil
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn with both gates passing = %+v; want it to go on", r)
	}
	if want := []string{"watch_gate", "herdr_server_gate"}; !slices.Equal(order, want) {
		t.Errorf("gates asked in the order %q; want %q", order, want)
	}

	// A directory where the manifest should be cannot be read.
	o.Manifest = NewManifestStoreAtPath(t.TempDir())
	o.WatchGate = nil
	o.HerdrServerGate = nil
	if r := o.Spawn(mustSpawnParams("org-a", "seat-2")); r.Outcome != SpawnOutcomeFailed || errors.Is(r.Err, errHerdrServerDown) {
		t.Fatalf("spawn with no gate on an unreadable manifest = %+v; the test needs a read that fails", r)
	}
	gate.err = errHerdrServerDown
	o.HerdrServerGate = gate.gate
	if r := o.Spawn(mustSpawnParams("org-a", "seat-2")); r.Outcome != SpawnOutcomeFailed || !errors.Is(r.Err, errHerdrServerDown) {
		t.Fatalf("spawn with a failing herdr server gate on an unreadable manifest = %+v; want the gate's failure first", r)
	}
}

// A `start --plan` whose gate fails is SpawnOutcomeFailed with the gate's
// error and makes no worktree and no branch: no ralph-worktree.sh call, no
// manifest event, no receipt, and no herdr or agmsg call (spec FR-A1 and
// FR-A5, AC4). A refusing watch gate comes first, with no herdr server gate
// call. A gate that passes is asked twice, by StartFeature and again by the
// leader's Spawn (D1).
func TestStartFeature_HerdrServerGate_AFailureMakesNoWorktree(t *testing.T) {
	t.Run("the herdr server gate fails", func(t *testing.T) {
		st := newFeatureStart(t)
		gate := &stubHerdrServerGate{err: errHerdrServerDown}
		st.o.HerdrServerGate = gate.gate
		before := takeSpawnSnapshot(t, st.o, st.h, st.a)

		res := st.o.StartFeature(st.params("auth-core"))
		if res.Spawn.Outcome != SpawnOutcomeFailed || !errors.Is(res.Spawn.Err, errHerdrServerDown) {
			t.Fatalf("start with a failing herdr server gate = %+v; want failed with the gate's error", res.Spawn)
		}
		if res.Worktree != "" || res.Branch != "" {
			t.Errorf("result names the worktree %q on %q; want none", res.Worktree, res.Branch)
		}
		st.wantCalls(t)
		if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
			t.Errorf("a start the herdr server gate failed recorded or called something: %+v -> %+v", before, after)
		}
		if _, err := os.Stat(st.worktree("auth-core")); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("worktree after a failed start: %v; want none", err)
		}
		if gate.calls != 1 {
			t.Errorf("gate called %d times; want once", gate.calls)
		}
	})

	t.Run("the watch gate refuses first", func(t *testing.T) {
		st := newFeatureStart(t)
		gate := &stubHerdrServerGate{err: errHerdrServerDown}
		st.o.HerdrServerGate = gate.gate
		st.o.WatchGate = refusingWatchGate().gate

		res := st.o.StartFeature(st.params("auth-core"))
		assertStartRefused(t, res, "no ralph org watch --all is running")
		st.wantCalls(t)
		if gate.calls != 0 {
			t.Errorf("herdr server gate called %d times after the watch gate refused; want none", gate.calls)
		}
	})

	t.Run("the herdr server gate passes", func(t *testing.T) {
		st := newFeatureStart(t)
		gate := &stubHerdrServerGate{}
		st.o.HerdrServerGate = gate.gate
		st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
		if gate.calls != 2 {
			t.Errorf("gate called %d times by a start that went through; want twice (StartFeature and the leader's Spawn)", gate.calls)
		}
	})
}

// A director start in herdr whose gate fails returns the gate's error and
// writes and calls nothing: no director.jsonl line, no prompt file, no ps
// read, no watch gate call, and no herdr call, so CheckDirector did not
// run even with a current headless herdr director whose check would call
// herdr (AC4 and AC9).
func TestStartDirector_HerdrServerGate_AFailureWritesAndCallsNothing(t *testing.T) {
	for name, events := range map[string][]DirectorEvent{
		"no registration":                      nil,
		"a current headless director in herdr": checkHerdrRunning(),
	} {
		t.Run(name, func(t *testing.T) {
			f := newCleanupFixture(t, events...)
			f.ch.agentGetErr = checkAgentNotFound()
			f.d1HerdrHasItsOwn()
			gate := &stubHerdrServerGate{err: errHerdrServerDown}
			f.o.HerdrServerGate = gate.gate
			before := directorFileState(t, f.store)

			if _, err := f.start(); !errors.Is(err, errHerdrServerDown) {
				t.Fatalf("StartDirector err = %v, want the herdr server gate's error", err)
			}
			if gate.calls != 1 {
				t.Errorf("gate called %d times; want once", gate.calls)
			}
			if len(f.h.calls) != 0 || len(f.psReads) != 0 {
				t.Fatalf("herdr calls %q and ps reads %v, want none", f.h.calls, f.psReads)
			}
			if f.gateCalls != 0 {
				t.Errorf("watch gate called %d times; want none", f.gateCalls)
			}
			if after := directorFileState(t, f.store); after != before || f.notice.Len() != 0 {
				t.Fatalf("director.jsonl changed from %q to %q, notice %q", before, after, f.notice.String())
			}
			if _, err := os.Stat(filepath.Join(f.dir, "prompts")); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("a failed start made the prompts dir (stat err %v)", err)
			}
		})
	}
}

// AC9 (D1): with a current headless director in herdr and the herdr server
// stopped, the gate runs before CheckDirector asks herdr about that
// director. Here herdr answers agent get with an error until the gate
// "starts" it, and with agent_not_found after, so a start that checked
// first would get unknown and be refused. The gate comes first, the check
// then finds d1 ended, start cleans it up, asks the watch gate, and starts
// d2.
func TestStartDirector_HerdrServerGate_RunsBeforeTheCheckOfAHeadlessHerdrDirector(t *testing.T) {
	events := checkHerdrRunning()
	f := newCleanupFixture(t, events...)
	f.d1HerdrHasItsOwn()
	f.ch.agentGetErr = errors.New("herdr: no server is running")
	logCall := func(name string) {
		f.h.mu.Lock()
		defer f.h.mu.Unlock()
		f.h.calls = append(f.h.calls, name)
	}
	gateCalls := 0
	f.o.HerdrServerGate = func() error {
		gateCalls++
		logCall("herdr_server_gate")
		f.ch.agentGetErr = checkAgentNotFound()
		return nil
	}
	f.o.WatchGate = func() error {
		f.gateCalls++
		logCall("watch_gate")
		return nil
	}

	res, err := f.start()
	if err != nil {
		t.Fatalf("StartDirector: %v", err)
	}
	wantStartedD2(t, res)
	want := append([]string{"herdr_server_gate", "agent_get", "pane_get", "tab_get", "workspace_get"}, directorCloseCalls...)
	want = append(want, "watch_gate", "workspace_create", "tab_create", "agent_start")
	if !slices.Equal(f.h.calls, want) {
		t.Fatalf("gate and herdr calls = %q, want %q", f.h.calls, want)
	}
	if gateCalls != 1 || f.gateCalls != 1 {
		t.Errorf("herdr server gate called %d times and watch gate %d; want once each", gateCalls, f.gateCalls)
	}
	if !strings.HasPrefix(f.notice.String(), "cleaned up d1, which had ended (agent_ended): ") {
		t.Errorf("notice = %q, want the line of d1's cleanup", f.notice.String())
	}
	wantTrail(t, f.store, append(d1Trail(events), d2Trail...)...)
}

// The input checks come before the gate: a start refused for one of them
// does not ask it (here [org.director].enabled is false, and HERDR_PANE_ID
// is set).
func TestStartDirector_HerdrServerGate_RunsAfterTheInputChecks(t *testing.T) {
	for name, setup := range map[string]func(f *dirStartFixture){
		"director disabled":    func(f *dirStartFixture) { f.o.Config.Director.Enabled = false },
		"HERDR_PANE_ID is set": func(f *dirStartFixture) { f.env[herdrPaneIDEnv] = "pane-9" },
	} {
		t.Run(name, func(t *testing.T) {
			f := newDirStartFixture(t, directorTestToken)
			setup(f)
			gate := &stubHerdrServerGate{}
			f.o.HerdrServerGate = gate.gate
			if _, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus")); err == nil {
				t.Fatal("StartDirector succeeded, want a refusal")
			}
			if gate.calls != 0 {
				t.Errorf("gate called %d times by a start refused for its input; want none", gate.calls)
			}
		})
	}
}

// A start with [org.director].launch_cmd makes nothing in herdr, so it
// never asks the gate (spec FR-A1, AC3): a gate that would fail does not
// stop it, and the start opens its terminal and checks in as before.
func TestStartDirector_HerdrServerGate_ALaunchCmdStartDoesNotAskIt(t *testing.T) {
	f := newCmdStartFixture(t, launchCmdInBackground, cmdStartOptions{})
	gate := &stubHerdrServerGate{err: errHerdrServerDown}
	f.o.HerdrServerGate = gate.gate
	p := f.params(DirectorDriverClaude, "opus")
	p.TimeoutMS = 30000
	res, err := f.o.StartDirector(p)
	if err != nil {
		t.Fatalf("StartDirector with launch_cmd and a failing herdr server gate: %v", err)
	}
	if !res.Registration.CmdLaunch() || res.Registration.State != DirectorStateRunning {
		t.Fatalf("Registration = %+v, want a running launch cmd registration", res.Registration)
	}
	if gate.calls != 0 {
		t.Errorf("gate called %d times by a launch_cmd start; want none", gate.calls)
	}
	if len(f.h.calls) != 0 {
		t.Errorf("herdr calls = %q, want none on the launch_cmd path", f.h.calls)
	}
	wantTrail(t, f.store, "attached d1", "launched d1")
}
