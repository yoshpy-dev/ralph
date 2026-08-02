// Package org implements the core library for the `ralph org` verb set
// (spawn/send/wait/read/stop/status/disband): envelope validation against
// [org] config, seat/saga state derivation from an org_id-namespaced
// manifest, and tri-state model receipts. This package is pure library
// code — no exec.Command, no cobra wiring. Those live in
// internal/org/driver (herdr/agmsg adapters) and internal/cli/org.go
// (`ralph org` cobra wiring) respectively.
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
