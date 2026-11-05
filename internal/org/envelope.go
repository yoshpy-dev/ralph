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
	// DriverFromRoleEntry is SpawnParams.DriverFromRoleEntry: Driver came
	// from Role's [org.role_models] entry, so ValidateSpawnEnvelope refuses
	// a Driver outside [org].driver_pool naming the role and its entry
	// (driverPoolErr).
	DriverFromRoleEntry bool
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
// envelope: the driver must be in [org].driver_pool (driverPoolErr, which
// names the role and its [org.role_models] entry when
// req.DriverFromRoleEntry is set), and the model must pass
// config.CheckOrgModelName ("" passes: the seat runs the driver CLI's own
// default model). The model is not checked against a list: [org.role_models]
// sets each role's default, and --model may name any model. The CLI's
// ResolveRoleModel checks neither, so both run here, after Spawn's
// idempotent return (spawn.go's AC-3): a re-run of a seat already spawned
// returns that seat even when driver_pool was narrowed since, or when its
// --model would be refused now (plan 2026-11-05-org-role-models, D4 and D9).
// These are pure functions of cfg+req -- unlike the max_seats capacity
// check, nothing about stale-seat compensation or manifest state can change
// their outcome, so they are safe to run before any external side effect is
// attempted.
func ValidateSpawnEnvelope(cfg config.OrgConfig, req SpawnRequest) error {
	if err := driverPoolErr(cfg, req.Role, req.Driver, req.DriverFromRoleEntry); err != nil {
		return err
	}
	if err := config.CheckOrgModelName(req.Model); err != nil {
		return fmt.Errorf("org: %w", err)
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
	if err := validateMaxOrgs(cfg, req.OrgID, runningOrgs); err != nil {
		return err
	}
	if totalActiveSeats >= cfg.MaxTotalSeats {
		return fmt.Errorf("org: max_total_seats %d reached: %d seats are active across all orgs; %s",
			cfg.MaxTotalSeats, totalActiveSeats, disbandFreesSlotHint)
	}
	return nil
}

// validateMaxOrgs is the max_orgs half of ValidateOrgWideCapacity: orgID is
// refused when it is not in runningOrgs and len(runningOrgs) has reached
// cfg.MaxOrgs. idempotentRespawnDecision (spawn.go), which idempotentRespawn
// and spawnPrecheckErr run, calls it alone, for a reservation that would make
// an org run without adding a seat.
func validateMaxOrgs(cfg config.OrgConfig, orgID string, runningOrgs []string) error {
	if len(runningOrgs) >= cfg.MaxOrgs && !slices.Contains(runningOrgs, orgID) {
		running := strings.Join(runningOrgs, ", ")
		if running == "" {
			running = "none"
		}
		return fmt.Errorf("org: max_orgs %d reached: org_id %q is not running and %d orgs are (%s); %s",
			cfg.MaxOrgs, orgID, len(runningOrgs), running, disbandFreesSlotHint)
	}
	return nil
}

func driverInPool(cfg config.OrgConfig, driver string) bool {
	return slices.Contains(cfg.DriverPool, driver)
}

// driverPoolErr refuses a driver that is not in cfg.DriverPool, for a seat
// of role (ValidateSpawnEnvelope) and for a headless director, role
// DirectorRole (checkDirectorStart), and returns nil for one that is in it.
// fromEntry says the driver came from role's [org.role_models] entry
// (ResolvedRoleModel.DriverFromRoleEntry): the refusal then names the role,
// the entry, and driver_pool, and the three ways out, since no --driver was
// given (plan 2026-11-05-org-role-models, D9). A driver from --driver gets
// the plain wording.
func driverPoolErr(cfg config.OrgConfig, role, driver string, fromEntry bool) error {
	if driverInPool(cfg, driver) {
		return nil
	}
	if fromEntry {
		return fmt.Errorf("org: role %q runs on driver %q in [org.role_models].%s, which is not in [org].driver_pool %v: "+
			"add %s to driver_pool, change the entry, or pass both --driver and --model", role, driver, role, cfg.DriverPool, driver)
	}
	return fmt.Errorf("org: driver %q not in [org].driver_pool %v", driver, cfg.DriverPool)
}
