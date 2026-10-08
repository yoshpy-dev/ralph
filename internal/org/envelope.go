package org

import (
	"fmt"
	"slices"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// SpawnRequest describes a proposed `ralph org spawn` invocation prior to
// any external side effect. The herdr/agmsg calls themselves live in
// spawn.go (via HerdrClient/AgmsgClient) and internal/org/driver; this file
// only validates the request against the [org] envelope.
type SpawnRequest struct {
	OrgID  string
	SeatID string
	Role   string
	Driver string
	Model  string
}

// ValidateSpawn checks req against cfg's [org] envelope and activeSeats,
// the number of seats already active within req.OrgID (computed by the
// caller, typically via (*ManifestStore).ActiveSeatCount -- this function
// performs no I/O so it stays trivially unit-testable).
//
// ValidateSpawn is the composition of ValidateSpawnEnvelope (stateless: pure
// functions of cfg+req) and ValidateSpawnCapacity (depends on activeSeats,
// which changes as a result of stale-seat compensation). Callers that need
// to run stale compensation *between* the two -- so a stateless-envelope
// rejection never triggers a destructive compensation side effect first --
// should call the two halves directly instead of this composition; see
// (*Org).Spawn in spawn.go.
//
// Each rejection reason returns a distinct, grep-able error so callers can
// assert on failure mode in tests and pass the message straight through to
// the manifest `details` field and receipt `reason` field (AC-1/AC-2).
func ValidateSpawn(cfg config.OrgConfig, req SpawnRequest, activeSeats int) error {
	if err := ValidateSpawnEnvelope(cfg, req); err != nil {
		return err
	}
	return ValidateSpawnCapacity(cfg, req, activeSeats)
}

// ValidateSpawnEnvelope checks the stateless part of req against cfg's [org]
// envelope: driver/model pool membership and role-based model restriction.
// These are pure functions of cfg+req -- unlike the max_seats capacity
// check, nothing about stale-seat compensation or manifest state can change
// their outcome, so they are safe to run before any external side effect is
// attempted.
func ValidateSpawnEnvelope(cfg config.OrgConfig, req SpawnRequest) error {
	if !driverInPool(cfg, req.Driver) {
		return fmt.Errorf("org: driver %q not in [org].driver_pool %v", req.Driver, cfg.DriverPool)
	}
	if !modelInPool(cfg, req.Driver, req.Model) {
		return fmt.Errorf("org: model %q not in [org].model_pool for driver %q", req.Model, req.Driver)
	}
	if !modelAllowedForRole(cfg, req.Role, req.Model) {
		return fmt.Errorf("org: model %q not permitted for role %q", req.Model, req.Role)
	}
	return nil
}

// ValidateSpawnCapacity checks activeSeats -- the number of seats already
// active within req.OrgID -- against cfg.MaxSeats. Unlike
// ValidateSpawnEnvelope, this check's outcome depends on manifest state
// (activeSeats), so callers that compensate a stale in-flight seat before
// re-deriving activeSeats must re-run this check with the recomputed count.
func ValidateSpawnCapacity(cfg config.OrgConfig, req SpawnRequest, activeSeats int) error {
	if activeSeats >= cfg.MaxSeats {
		return fmt.Errorf("org: max_seats %d reached for org_id %q", cfg.MaxSeats, req.OrgID)
	}
	return nil
}

// disbandFreesSlotHint ends both ValidateOrgWideCapacity errors: the
// operator's way out is to disband an org that has finished.
const disbandFreesSlotHint = "a finished org frees its slot and seats with ralph org disband --org-id <id>, or ralph org disband --all for every org"

// ValidateOrgWideCapacity checks req against the limits shared by every
// org_id in one org state dir. runningOrgs is RunningOrgs and
// totalActiveSeats TotalActiveSeats of the manifest (reserve.go), both
// derived by the caller from events read under the manifest lock, the same
// way ValidateSpawnCapacity gets activeSeats. A spawn into an org_id that is
// not in runningOrgs is refused once len(runningOrgs) reaches cfg.MaxOrgs; a
// spawn into a running org is not limited by MaxOrgs. Any new seat is refused
// once totalActiveSeats reaches cfg.MaxTotalSeats. Like max_seats, a limit
// of 0 or less is not "no limit": it refuses every spawn the limit covers
// (config.Load rejects such a value, so only a hand-built config.OrgConfig
// can carry one).
func ValidateOrgWideCapacity(cfg config.OrgConfig, req SpawnRequest, runningOrgs []string, totalActiveSeats int) error {
	if len(runningOrgs) >= cfg.MaxOrgs && !slices.Contains(runningOrgs, req.OrgID) {
		running := strings.Join(runningOrgs, ", ")
		if running == "" {
			running = "none"
		}
		return fmt.Errorf("org: max_orgs %d reached: org_id %q is not running and %d orgs are (%s); %s",
			cfg.MaxOrgs, req.OrgID, len(runningOrgs), running, disbandFreesSlotHint)
	}
	if totalActiveSeats >= cfg.MaxTotalSeats {
		return fmt.Errorf("org: max_total_seats %d reached: %d seats are active across all orgs; %s",
			cfg.MaxTotalSeats, totalActiveSeats, disbandFreesSlotHint)
	}
	return nil
}

func driverInPool(cfg config.OrgConfig, driver string) bool {
	return slices.Contains(cfg.DriverPool, driver)
}

func modelInPool(cfg config.OrgConfig, driver, model string) bool {
	for _, entry := range cfg.ModelPool {
		if entry.Driver == driver && entry.Model == model {
			return true
		}
	}
	return false
}

// modelAllowedForRole reports whether model is permitted for role. A role
// absent from cfg.Roles, or mapped to an empty list, means "no restriction"
// -- the full model_pool is allowed for that role. This mirrors the
// [org].roles Load() semantics in internal/config: only an explicit,
// non-empty allowlist narrows things.
func modelAllowedForRole(cfg config.OrgConfig, role, model string) bool {
	if len(cfg.Roles) == 0 {
		return true
	}
	allowed, ok := cfg.Roles[role]
	if !ok || len(allowed) == 0 {
		return true
	}
	return slices.Contains(allowed, model)
}
