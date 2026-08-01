// Package org implements the core library for the `ralph org` verb set
// (org-runtime-mechanism PR①): envelope validation against [org] config,
// seat/saga state derivation from an org_id-namespaced manifest, and
// tri-state model receipts. This package is pure library code — no
// exec.Command, no cobra wiring. Those land in later slices (driver
// adapters and the `ralph org` verbs, respectively).
package org

// Saga/event name constants recorded in ManifestEvent.Event.
//
// The spawn saga transitions: spawn_started -> spawned | spawn_failed.
// spawn_started is written before any external side effect is attempted, so
// a crash mid-spawn leaves a visible, auditable in-flight record rather than
// silence. Rejection (envelope validation failure, e.g. AC-1/AC-2) is
// recorded as `rejected` without a preceding spawn_started, since no
// external side effects were attempted. `stopped` is a single-seat terminal
// event; `disbanded` is an org-level terminal event (SeatID empty) that
// affects every seat in that org_id.
//
// Reading tolerates unknown/future event strings without failing
// derivation -- only the constants below participate in "is this seat
// active" logic (see isActiveEvent).
const (
	EventSpawnStarted = "spawn_started"
	EventSpawned      = "spawned"
	EventSpawnFailed  = "spawn_failed"
	EventRejected     = "rejected"
	EventStopped      = "stopped"
	EventDisbanded    = "disbanded"
)

// activeEvents are the saga states considered "active" for roster/status
// purposes. spawn_started counts as active on purpose: a stale, never-
// resolved in-flight spawn must stay visible in `status` rather than
// disappearing, so operators can spot orphaned sagas (AC-10).
var activeEvents = map[string]bool{
	EventSpawnStarted: true,
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
	Event     string // latest event name that produced this status
	Active    bool
	DryRun    bool
	Details   string
	TS        string
}
