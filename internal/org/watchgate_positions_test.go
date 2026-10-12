package org

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
)

// Where Spawn, StartFeature and Send run the watch gate (plan
// 2026-10-12-org-watch-gate, F1 to F5, AC1 and AC2 at the org layer). The
// Org of each test sets WatchGate to a stub; every other org-layer test
// leaves it nil, which checks nothing.

// stubWatchGate is an Org.WatchGate that counts its calls and returns err.
type stubWatchGate struct {
	calls int
	err   error
}

func (g *stubWatchGate) gate() error {
	g.calls++
	return g.err
}

// refusingWatchGate is a stub that refuses as the gate does with no watch
// running.
func refusingWatchGate() *stubWatchGate {
	return &stubWatchGate{err: &WatchGateError{StateDir: "/ledger", Liveness: WatchLiveness{State: WatchLivenessNotRunning}}}
}

// A refused spawn, real or dry-run, returns the gate's error as a rejection
// and writes nothing: no manifest event (not even `rejected`), no receipt,
// and no herdr or agmsg call (F1, F3). A gate that passes lets the spawn go
// on, and the gate is called once per spawn.
func TestOrgSpawn_WatchGate_ARefusalRecordsAndCallsNothing(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		name := "real"
		if dryRun {
			name = "dry-run"
		}
		t.Run(name, func(t *testing.T) {
			o, h, a := testOrg(t)
			if r := o.Spawn(mustSpawnParams("org-a", "seat-0")); r.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("spawn with no gate failed: %+v", r)
			}
			gate := refusingWatchGate()
			o.WatchGate = gate.gate
			before := takeSpawnSnapshot(t, o, h, a)

			p := mustSpawnParams("org-a", "seat-1")
			p.DryRun = dryRun
			r := o.Spawn(p)
			if r.Outcome != SpawnOutcomeRejected || !errors.Is(r.Err, ErrNoRunningWatch) {
				t.Fatalf("spawn with a refusing gate = %+v; want a rejection that is ErrNoRunningWatch", r)
			}
			if r.ModelReceipt != (Receipt{}) {
				t.Errorf("ModelReceipt = %+v; want none", r.ModelReceipt)
			}
			if after := takeSpawnSnapshot(t, o, h, a); after != before {
				t.Errorf("a refused spawn recorded or called something: %+v -> %+v", before, after)
			}
			if gate.calls != 1 {
				t.Errorf("gate called %d times; want once", gate.calls)
			}

			gate.err = nil
			if r := o.Spawn(p); r.Err != nil || r.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("spawn with a passing gate = %+v; want it to go on", r)
			}
			if gate.calls != 2 {
				t.Errorf("gate called %d times after the second spawn; want twice", gate.calls)
			}
		})
	}
}

// The gate runs after the input checks and before the manifest is read:
// an invalid org_id is refused for itself with no gate call, and with a
// manifest that cannot be read the refusal is the gate's, not the read's.
func TestOrgSpawn_WatchGate_RunsAfterTheInputChecksAndBeforeTheManifestRead(t *testing.T) {
	o, _, _ := testOrg(t)
	gate := refusingWatchGate()
	o.WatchGate = gate.gate
	r := o.Spawn(mustSpawnParams("Bad Org", "seat-1"))
	if r.Outcome != SpawnOutcomeRejected || errors.Is(r.Err, ErrNoRunningWatch) || !strings.Contains(r.Err.Error(), "invalid org_id") {
		t.Fatalf("spawn with an invalid org_id = %+v; want the identifier's rejection", r)
	}
	if gate.calls != 0 {
		t.Errorf("gate called %d times for an invalid org_id; want none", gate.calls)
	}

	// A directory where the manifest should be cannot be read.
	o.Manifest = NewManifestStoreAtPath(t.TempDir())
	o.WatchGate = nil
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeFailed {
		t.Fatalf("spawn with no gate on an unreadable manifest = %+v; the test needs a read that fails", r)
	}
	o.WatchGate = gate.gate
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeRejected || !errors.Is(r.Err, ErrNoRunningWatch) {
		t.Fatalf("spawn with a refusing gate on an unreadable manifest = %+v; want the gate's rejection first", r)
	}
}

// A refused `start --plan` makes no worktree and no branch, calls no
// ralph-worktree.sh, and records and calls nothing else. The gate runs
// before spawnPrecheckErr, in Spawn's order: with max_orgs reached as well,
// the refusal is the gate's. A gate that passes is called twice, by
// StartFeature and again by the leader's Spawn (F2).
func TestStartFeature_WatchGate_ARefusalMakesNoWorktree(t *testing.T) {
	for _, maxOrgsReached := range []bool{false, true} {
		name := "watch gate only"
		if maxOrgsReached {
			name = "max_orgs reached too"
		}
		t.Run(name, func(t *testing.T) {
			st := newFeatureStart(t)
			if maxOrgsReached {
				st.o.Config.MaxOrgs = 1
				spawnLeaderIn(t, st.o, st.h, "other", "ws-1", "pane-1", "web/")
			}
			gate := refusingWatchGate()
			st.o.WatchGate = gate.gate
			before := takeSpawnSnapshot(t, st.o, st.h, st.a)

			res := st.o.StartFeature(st.params("auth-core"))
			assertStartRefused(t, res, "no ralph org watch --all is running")
			if !errors.Is(res.Spawn.Err, ErrNoRunningWatch) {
				t.Errorf("error %v is not ErrNoRunningWatch", res.Spawn.Err)
			}
			st.wantCalls(t)
			if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
				t.Errorf("a refused start recorded or called something: %+v -> %+v", before, after)
			}
			if _, err := os.Stat(st.worktree("auth-core")); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("worktree after a refused start: %v; want none", err)
			}
			if gate.calls != 1 {
				t.Errorf("gate called %d times; want once", gate.calls)
			}
		})
	}

	st := newFeatureStart(t)
	gate := &stubWatchGate{}
	st.o.WatchGate = gate.gate
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	if gate.calls != 2 {
		t.Errorf("gate called %d times by a start that went through; want twice (StartFeature and the leader's Spawn)", gate.calls)
	}
}

// Send runs the gate only for a TASK: a typed TASK, a raw message whose
// first line Parse reads as TYPE: TASK (spaces around key and value
// trimmed), and either one in dry-run (F3 to F5). A refused send records
// nothing and calls no driver. Any other TYPE, a raw first line that is no
// TYPE header (a leading blank line, lowercase `type:`), and a raw TYPE of
// another case are sent with no gate call, and an invalid typed message is
// refused by validation before the gate.
func TestOrgSend_WatchGate_RefusesATaskOnly(t *testing.T) {
	const task = "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing"
	cases := []struct {
		name        string
		text        string
		raw, dryRun bool
		refused     bool
	}{
		{name: "typed TASK", text: task, refused: true},
		{name: "typed TASK, dry-run", text: task, dryRun: true, refused: true},
		{name: "raw TYPE: TASK", text: "TYPE: TASK\n\nanything", raw: true, refused: true},
		{name: "raw TYPE: TASK with spaces around key and value", text: "  TYPE:   TASK  \nanything", raw: true, refused: true},
		{name: "raw TYPE: TASK, dry-run", text: "TYPE: TASK", raw: true, dryRun: true, refused: true},
		{name: "typed QUESTION", text: "TYPE: QUESTION\n\nwhich file?"},
		{name: "typed RESULT, dry-run", text: "TYPE: RESULT\nTASK_ID: t-1\n\ndone", dryRun: true},
		{name: "raw text with no TYPE line", text: "please look at TYPE: TASK", raw: true},
		{name: "raw TYPE: TASK after a blank line", text: "\nTYPE: TASK", raw: true},
		{name: "raw lowercase type: TASK", text: "type: TASK", raw: true},
		{name: "raw TYPE: task", text: "TYPE: task", raw: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o, h, a := testOrg(t)
			if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("spawn failed: %+v", r)
			}
			gate := refusingWatchGate()
			o.WatchGate = gate.gate
			before := takeSpawnSnapshot(t, o, h, a)

			result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: tc.text, Raw: tc.raw, DryRun: tc.dryRun})
			if !tc.refused {
				if result.Err != nil {
					t.Fatalf("send = %v; want it sent with no gate", result.Err)
				}
				if gate.calls != 0 {
					t.Errorf("gate called %d times for a message that is no TASK; want none", gate.calls)
				}
				if last := lastEvent(t, o); last.Event != EventSent || last.DryRun != tc.dryRun {
					t.Errorf("last event = %+v; want sent, dry_run %t", last, tc.dryRun)
				}
				return
			}
			if !errors.Is(result.Err, ErrNoRunningWatch) || !strings.Contains(result.Err.Error(), "org: send: TASK not sent: ") {
				t.Fatalf("send = %v; want the gate's refusal of the TASK", result.Err)
			}
			if result.Progress != SendProgressNothingSent || result.PaneID != "" {
				t.Errorf("send = %+v; want nothing sent", result)
			}
			if after := takeSpawnSnapshot(t, o, h, a); after != before {
				t.Errorf("a refused send recorded or called something: %+v -> %+v", before, after)
			}
			if gate.calls != 1 {
				t.Errorf("gate called %d times; want once", gate.calls)
			}
		})
	}

	o, _, _ := testOrg(t)
	gate := refusingWatchGate()
	o.WatchGate = gate.gate
	result := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: "TYPE: TASK\n\nno task id"})
	if result.Err == nil || errors.Is(result.Err, ErrNoRunningWatch) || !strings.Contains(result.Err.Error(), "protocol validation") {
		t.Fatalf("send of a TASK with no TASK_ID = %v; want the protocol validation's rejection", result.Err)
	}
	if gate.calls != 0 {
		t.Errorf("gate called %d times for an invalid message; want none", gate.calls)
	}
}
