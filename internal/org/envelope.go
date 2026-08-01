package org

import (
	"fmt"
	"slices"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// SpawnRequest describes a proposed `ralph org spawn` invocation prior to
// any external side effect (herdr/agmsg calls are Slice 3/4 concerns; this
// package validates and records only).
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
// Each rejection reason returns a distinct, grep-able error so callers can
// assert on failure mode in tests and pass the message straight through to
// the manifest `details` field and receipt `reason` field (AC-1/AC-2).
func ValidateSpawn(cfg config.OrgConfig, req SpawnRequest, activeSeats int) error {
	if !driverInPool(cfg, req.Driver) {
		return fmt.Errorf("org: driver %q not in [org].driver_pool %v", req.Driver, cfg.DriverPool)
	}
	if !modelInPool(cfg, req.Driver, req.Model) {
		return fmt.Errorf("org: model %q not in [org].model_pool for driver %q", req.Model, req.Driver)
	}
	if !modelAllowedForRole(cfg, req.Role, req.Model) {
		return fmt.Errorf("org: model %q not permitted for role %q", req.Model, req.Role)
	}
	if activeSeats >= cfg.MaxSeats {
		return fmt.Errorf("org: max_seats %d reached for org_id %q", cfg.MaxSeats, req.OrgID)
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
