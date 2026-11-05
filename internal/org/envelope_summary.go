package org

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// EnvelopeSummary renders a single-line, human-readable summary of cfg's
// [org] envelope: each role's driver and model from [org.role_models]
// (`<role>=<driver>/<model>`, with CLIDefaultModelText for a model of ""),
// max_seats, and the resolved permissions default. It is substituted into
// prompts/leader.md's {{ENVELOPE}} placeholder (RenderRolePrompt, called
// from Spawn/dryRunSpawn in spawn.go) so a headless leader seat (`ralph org
// start`) starts with a compact picture of the driver and model each role
// gets by default, without needing to read ralph.toml itself. The roles are
// rendered sorted by name, since cfg.RoleModels is a map, so repeated calls
// with the same cfg are byte-identical -- callers (and tests) depend on that
// determinism.
func EnvelopeSummary(cfg config.OrgConfig) string {
	roles := slices.Sorted(maps.Keys(cfg.RoleModels))
	models := make([]string, 0, len(roles))
	for _, role := range roles {
		entry := cfg.RoleModels[role]
		models = append(models, fmt.Sprintf("%s=%s/%s", role, entry.Driver, ModelOrCLIDefault(entry.Model)))
	}
	modelText := "(none configured)"
	if len(models) > 0 {
		modelText = strings.Join(models, ", ")
	}
	permDefault := cfg.Permissions.Default
	if permDefault == "" {
		// Mirrors ResolvePermissionMode's own fallback (permissions.go):
		// config.Load() already backfills an absent [org.permissions].default
		// to "autonomous", so this only matters for a config.OrgConfig built
		// by hand (e.g. a test literal) rather than through Load().
		permDefault = defaultPermissionMode
	}
	return fmt.Sprintf("role_models: %s | max_seats: %d | permission default: %s", modelText, cfg.MaxSeats, permDefault)
}

// DefaultModelForDriver returns the Model of the first cfg.ModelPool entry
// whose Driver matches driver, in cfg.ModelPool's declared order, ignoring
// [org.roles]. It is the role-agnostic primitive behind
// DefaultModelForDriverAndRole; the CLI's --model fallback for both
// `ralph org spawn` and `ralph org start` goes through the role-aware
// variant (internal/cli/org.go's resolveModelOrWarn), so this function has
// no production caller today and is kept as the documented "pool head per
// driver" query (unit-tested in envelope_summary_test.go). An error is
// returned when no model_pool entry matches driver. Plan
// 2026-11-05-org-role-models removes it once the CLI uses ResolveRoleModel.
func DefaultModelForDriver(cfg config.OrgConfig, driver string) (string, error) {
	for _, entry := range cfg.ModelPool {
		if entry.Driver == driver {
			return entry.Model, nil
		}
	}
	return "", fmt.Errorf("org: no [org].model_pool entry for driver %q; pass --model explicitly", driver)
}

// DefaultModelForDriverAndRole returns the Model of the first cfg.ModelPool
// entry whose Driver matches driver AND is permitted for role under
// [org.roles] (ModelAllowedForRole, envelope.go), in cfg.ModelPool's
// declared order. This is `ralph org spawn`'s --model default when the
// caller omits --model (and `ralph org start`'s, with role leader): spawn
// accepts arbitrary roles, so a role-restricted pool (e.g. `implementer =
// ["sonnet"]`) can make the pool's own head entry impermissible for the
// requesting role (self-review MEDIUM-2). Skipping straight to the first
// driver-and-role-permitted entry keeps the fallback warning honest. An
// error is returned when no model_pool entry satisfies both driver and
// role, so the caller fails fast with an actionable message instead of
// falling through to an empty --model. ValidateSpawnEnvelope no longer
// checks the pool or the role (plan 2026-11-05-org-role-models), which
// removes this function once the CLI uses ResolveRoleModel.
func DefaultModelForDriverAndRole(cfg config.OrgConfig, driver, role string) (string, error) {
	for _, entry := range cfg.ModelPool {
		if entry.Driver == driver && ModelAllowedForRole(cfg, role, entry.Model) {
			return entry.Model, nil
		}
	}
	return "", fmt.Errorf("org: no [org].model_pool entry for driver %q is permitted for role %q; pass --model explicitly", driver, role)
}
