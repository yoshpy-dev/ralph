// Package org implements the core library for the `ralph org` verb set
// (spawn/send/wait/read/stop/status/disband): envelope validation against
// [org] config, seat/saga state derivation from an org_id-namespaced
// manifest, and tri-state model receipts. It has no cobra wiring, which
// lives in internal/cli/org.go, and reaches herdr and agmsg only through
// the adapters in internal/org/driver. It runs a few other commands itself
// with exec.Command: git (watch.go, feature.go, gate.go,
// regate_git.go for Regate's fetches, merge, and push and for the checks of
// the cleanup of a merged feature worktree (cleanup.go), `git ls-remote`, plus
// `git remote get-url origin` once gh returned a PR, in mergeorder.go, `git
// remote get-url origin` in mergeguarantee.go, and both in merge.go; the
// git calls with which statedir.go finds the main worktree and the toplevel
// are in internal/gitroot),
// scripts/ralph-worktree.sh (feature.go), scripts/run-static-verify.sh and
// scripts/run-test.sh of a feature worktree (gate_run.go), osascript
// (watch.go), gh (mergeorder.go, mergeguarantee.go, mergeenforce.go,
// merge.go, and merge_run.go), ps and [org.director].launch_cmd
// (director_cmd.go), and `claude -p` (watcher.go).
package org

// Saga/event name constants recorded in ManifestEvent.Event.
//
// The spawn saga transitions: spawn_started -> spawn_step (zero or more,
// in-flight progress markers) -> spawned | spawn_failed. spawn_started is
// written before any external side effect is attempted, so a crash mid-spawn
// leaves a visible, auditable in-flight record rather than silence.
// Rejection (envelope validation failure, e.g. AC-1/AC-2) is recorded as
// `rejected` without a preceding spawn_started, since no external side
// effects were attempted. `stopped` is a single-seat terminal event;
// `disbanded` is an org-level terminal event (SeatID empty) that affects
// every seat in that org_id.
//
// Reading tolerates unknown/future event strings without failing
// derivation. Two independent classifications apply to every event:
//   - isStateEvent (see stateEvents below): whether the event participates
//     in "what is this seat's current status" derivation at all (Roster's
//     latest-pointer). Non-state events (e.g. the `sent` event recorded by
//     `ralph org send`) are fully readable via ManifestStore.Read for
//     history, but never move or clear a seat's derived status.
//   - isActiveEvent (see activeEvents below): among state events, which ones
//     count as "seat currently active" for roster/status/max_seats purposes.
const (
	EventSpawnStarted = "spawn_started"
	EventSpawnStep    = "spawn_step"
	EventSpawned      = "spawned"
	EventSpawnFailed  = "spawn_failed"
	EventRejected     = "rejected"
	EventStopped      = "stopped"
	EventDisbanded    = "disbanded"
	// EventStopFailed is a non-state event (deliberately absent from
	// stateEvents below): Stop records it instead of `stopped` when the
	// seat's herdr pane could not be closed, so the seat stays active in the
	// roster and a later stop picks it up again. Details say what failed.
	EventStopFailed = "stop_failed"
	// EventSpawnCloseFailed is a non-state event (deliberately absent from
	// stateEvents and activeEvents below): Spawn records it instead of
	// `spawn_failed` when a step failed after tab_create and the pane that
	// spawn made could not be closed (failStepWithNote), so the seat's latest
	// state event stays the spawn_step that recorded the pane_id, the seat
	// stays active in the roster, and a later stop closes that pane. It
	// mirrors stop_failed (spec FR-2). Details say which step failed, why,
	// and what the compensation did.
	EventSpawnCloseFailed = "spawn_close_failed"
	// EventScopeReserved is an org-level, non-state event (SeatID empty,
	// absent from stateEvents below): a leader spawn with
	// SpawnParams.Reserve records the org's reservation with it, Details
	// `paths=<normalized paths, comma-separated>`. For an org started for a
	// split plan feature (SpawnParams.Feature) the binding follows as
	// `split=<id> feature=<slug> digest=<hex> branch=<branch>`, and the
	// feature worktree goes into the event's Worktree field. A note may
	// come last (see scopeReservedPathsKey in reserve.go). The latest one
	// after the org's latest `disbanded` is the org's reservation and
	// binding, and it makes the org count as running for max_orgs.
	EventScopeReserved = "scope_reserved"
	// EventPlanApproved is an org-level, non-state event (SeatID empty,
	// absent from stateEvents below) that only `ralph org approve` writes
	// (Approve, approve.go): a human approved the plan a leader's inbox
	// QUESTION named, one event per item, Details `plan=<path in the feature
	// worktree> digest=<12 hex> content=<64 hex> inbox=<item ID> by=<USER or
	// unknown> herdr_pane=<set|unset> cwd=<quoted directory>`. The approval
	// check of an org started for a split plan feature in per-plan mode
	// (checkPlanApproved) reads the ones after the org's latest `disbanded`,
	// so an approval from an earlier life of the same org_id never counts.
	// Roster skips it like every org-level event, and it does not make an
	// org count as running.
	EventPlanApproved = "plan_approved"
	// EventGatePassed and EventGateFailed are org-level, non-state events
	// (SeatID empty, absent from stateEvents below) that `ralph org gate`
	// (Gate, gate.go) and `ralph org regate` (Regate, regate.go) write: the
	// gate scripts passed, or failed, on the feature worktree's HEAD. Details
	// are `head=<sha> base=<sha> base_ref=<ref> branch=<branch> by=<USER or
	// unknown> herdr_pane=<set|unset> log=<quoted path>`, with
	// `reason=<static_failed, test_failed, head_moved or tree_changed>` first
	// on gate_failed and, on a record ralph org regate wrote, ` via=regate`
	// before `log=` (gateRecord in gate_record.go, which has the record's
	// shape). A refusal before the scripts run records neither. The
	// watch counts both as leader activity (leaderActivityEventCount); Roster
	// skips them like every org-level event, and they do not make an org
	// count as running.
	EventGatePassed = "gate_passed"
	EventGateFailed = "gate_failed"
	// EventGateCIPassed and EventGateCIFailed are the gate records that
	// `ralph org regate` writes under [org.gate].regate_checks = "ci" (plan
	// docs/plans/active/2026-10-29-regate-ci-checks.md, D4): the checks of
	// the PR of the branch's head on origin, which it pushed or found there,
	// all passed, or one failed (reason ci_failed). They are org-level,
	// non-state events like gate_passed and gate_failed, with the same
	// Details (gateRecord.details) and ` via=regate_ci` before `log=`, whose
	// log holds the checks it read. Their names differ so that a binary
	// older than them, which picks the gate records by event name, skips
	// them: after a revert, an old regate must not read a CI pass as one of
	// its own and push it again (regateHeadNotOnOrigin). ralph org
	// merge-order, merge, and regate read them as a pass and a failure
	// (collectMergeTargets). The watch does not count them as leader
	// activity (leaderActivityEventCount), and Roster skips them like every
	// org-level event.
	EventGateCIPassed = "gate_ci_passed"
	EventGateCIFailed = "gate_ci_failed"
	// EventPRMerged is an org-level, non-state event (SeatID empty, absent
	// from stateEvents below) that only `ralph org merge` writes (Merge,
	// merge_run.go): gh pr merge merged the PR of a branch ralph org
	// merge-order listed as ready, and gh pr view read the PR back as MERGED
	// at the gate's head into the default branch. It is written once per
	// merge, under the org of the branch's latest binding (else of its latest
	// gate record), Details `branch=<branch> pr=<number> head=<the head the gate
	// passed> method=<merge, squash or rebase> merge_commit=<sha, or unknown
	// when gh gave none> auto_merge=<true|false>`. auto_merge= is the value of
	// [org.director].auto_merge when the merge ran; it does not say whether
	// the director merged without asking, since a merge a human asked for
	// records the same value. A refusal and a merge that failed record
	// none. Roster skips it like every org-level event, so stop --all finds
	// no seat in it; it makes no org count as running (RunningOrgs, which
	// max_orgs counts) nor a target of disband --all (orgsToDisband), and the
	// watch does not count it as leader activity (leaderActivityEventCount).
	// ralph org merge-order does not read it.
	EventPRMerged = "pr_merged"
	// EventWorktreeCleaned is an org-level, non-state event (SeatID empty,
	// absent from stateEvents below) that only the cleanup of a merged
	// feature worktree writes (CleanupFeatureWorktree, cleanup.go), for ralph
	// org merge after its merge (via=merge) and for ralph org cleanup
	// (via=cleanup): scripts/ralph-worktree.sh cleanup --force-branch removed
	// the feature worktree of a branch, its local branch, and its
	// ralph-worktree.sh record, once the checks found nothing there that
	// would be lost. It is written once per cleanup, under the org of the
	// branch's latest binding, Details `branch=<branch> head=<the head the PR
	// merged> via=<merge or cleanup> worktree=<quoted path>`, the worktree
	// last since its value is strconv.Quote'd and may hold spaces, as a gate
	// record's log is. A refusal and a removal that failed record none.
	// Roster skips it like every org-level event; it makes no org count as
	// running (RunningOrgs) nor a target of disband --all (orgsToDisband),
	// and the watch does not count it as leader activity
	// (leaderActivityEventCount).
	EventWorktreeCleaned = "worktree_cleaned"
)

// stateEvents are the event types that drive seat status derivation
// (Roster's "latest applicable event" pointer, see manifest.go). Any other
// event type -- present or future -- is ignored for that purpose: it does
// not overwrite or clear a seat's derived status, only a state event can.
// This keeps non-state manifest traffic (e.g. `sent`) from ever masking a
// seat's true saga/lifecycle state.
var stateEvents = map[string]bool{
	EventSpawnStarted: true,
	EventSpawnStep:    true,
	EventSpawned:      true,
	EventSpawnFailed:  true,
	EventRejected:     true,
	EventStopped:      true,
	EventDisbanded:    true,
}

func isStateEvent(event string) bool {
	return stateEvents[event]
}

// activeEvents are the saga states considered "active" for roster/status
// purposes. spawn_started and spawn_step count as active on purpose: a
// stale, never-resolved in-flight spawn must stay visible in `status` rather
// than disappearing, so operators can spot orphaned sagas (AC-10).
var activeEvents = map[string]bool{
	EventSpawnStarted: true,
	EventSpawnStep:    true,
	EventSpawned:      true,
}

func isActiveEvent(event string) bool {
	return activeEvents[event]
}

// SeatStatus is the derived state of a single seat within an org_id
// namespace, computed purely from the latest applicable manifest event(s)
// for that seat (see Roster).
type SeatStatus struct {
	OrgID     string
	SeatID    string
	Role      string
	Driver    string
	Model     string
	Worktree  string
	PaneID    string
	AgmsgTeam string
	// HerdrAgentName is the persisted herdr agent name from the `spawned`
	// event's ManifestEvent.HerdrAgentName (see its doc comment). Empty for
	// seats spawned before this field existed -- callers must fall back to
	// re-deriving the name via herdrAgentName(OrgID, SeatID) in that case
	// (see verbs.go's resolvedHerdrAgentName).
	HerdrAgentName string
	Event          string // latest event name that produced this status
	Active         bool
	DryRun         bool
	Details        string
	TS             string
}
