package org

// The herdr server gate (plan 2026-11-06-org-herdr-autostart, D1 and D2;
// spec docs/specs/2026-11-06-org-herdr-autostart-panes.md, FR-A1): a
// `ralph org start --plan` whose herdr server was not running made the
// feature worktree and its branch and then failed at workspace_create with
// `Connection refused`, leaving both behind. Org.HerdrServerGate is the
// check that runs before anything herdr is asked to make: the CLI's gate
// starts `herdr server` when it is not running and returns an error when it
// cannot. Spawn calls it right after the watch gate, in real mode only (a
// dry run asks herdr nothing); StartFeature right after the watch gate,
// before it reads the manifest or makes the worktree; and StartDirector, for
// a start in herdr only, after its input checks and before its first look at
// the current registration, since that look asks herdr whether a headless
// herdr director is still there (checkCurrentDirector) and, with herdr
// stopped, gets unknown and refuses. A start with [org.director].launch_cmd
// makes nothing in herdr and never calls it. A refusal is the gate's error
// alone, with no manifest event, no receipt, no director.jsonl line, and no
// herdr or agmsg call; Spawn and StartFeature return it as
// SpawnOutcomeFailed, not as a rejection of their input (D2). Spawn calls it
// before its idempotent early return too, on purpose (D1: before any
// manifest read): a respawn of a seat already spawned, which makes nothing,
// also asks it, so the CLI's gate reads the status (once, with a server
// running) and starts a stopped server, and a gate that fails makes that
// respawn SpawnOutcomeFailed with nothing changed.

// checkHerdrServerGate runs o.HerdrServerGate; a nil HerdrServerGate checks
// nothing.
func (o *Org) checkHerdrServerGate() error {
	if o.HerdrServerGate == nil {
		return nil
	}
	return o.HerdrServerGate()
}
