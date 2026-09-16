package org

import (
	"fmt"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// EnvelopeSummary renders a single-line, human-readable summary of cfg's
// [org] envelope: model_pool entries, max_seats, and the resolved
// permissions default. It is substituted into prompts/lead.md's
// {{ENVELOPE}} placeholder (RenderRolePrompt, called from Spawn/dryRunSpawn
// in spawn.go) so a headless lead seat (`ralph org start`) starts with a
// compact picture of what it is allowed to spawn, without needing to read
// ralph.toml itself. Model pool entries are rendered in cfg.ModelPool's own
// declared order (no re-sorting) so repeated calls with the same cfg are
// byte-identical -- callers (and tests) depend on that determinism.
func EnvelopeSummary(cfg config.OrgConfig) string {
	models := make([]string, 0, len(cfg.ModelPool))
	for _, entry := range cfg.ModelPool {
		models = append(models, fmt.Sprintf("%s/%s", entry.Driver, entry.Model))
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
	return fmt.Sprintf("model_pool: %s | max_seats: %d | permission default: %s", modelText, cfg.MaxSeats, permDefault)
}

// DefaultModelForDriver returns the Model of the first cfg.ModelPool entry
// whose Driver matches driver, in cfg.ModelPool's declared order, ignoring
// [org.roles]. It is the role-agnostic primitive behind
// DefaultModelForDriverAndRole; the CLI's --model fallback for both
// `ralph org spawn` and `ralph org start` goes through the role-aware
// variant (internal/cli/org.go's resolveModelOrWarn), so this function has
// no production caller today and is kept as the documented "pool head per
// driver" query (unit-tested in envelope_summary_test.go). The org's own
// model_pool -- already the allowlist ValidateSpawnEnvelope checks Spawn's
// request against -- is the single source of truth for "what's available",
// so a value returned here can never itself be out-of-pool. An error is
// returned when no model_pool entry matches driver.
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
// [org.roles] (modelAllowedForRole, envelope.go), in cfg.ModelPool's
// declared order. This is `ralph org spawn`'s --model default when the
// caller omits --model (and `ralph org start`'s, with role lead): spawn
// accepts arbitrary roles, so a role-restricted pool (e.g. `implementer =
// ["sonnet"]`) can make the pool's own head entry impermissible for the
// requesting role -- picking that head anyway would warn-then-reject via
// ValidateSpawnEnvelope's own modelAllowedForRole check
// (self-review MEDIUM-2). Skipping straight to the first
// driver-and-role-permitted entry keeps the fallback warning honest: if it
// prints a model, that model will also pass validation. An error is
// returned when no model_pool entry satisfies both driver and role, so the
// caller fails fast with an actionable message instead of falling through
// to an empty --model.
func DefaultModelForDriverAndRole(cfg config.OrgConfig, driver, role string) (string, error) {
	for _, entry := range cfg.ModelPool {
		if entry.Driver == driver && modelAllowedForRole(cfg, role, entry.Model) {
			return entry.Model, nil
		}
	}
	return "", fmt.Errorf("org: no [org].model_pool entry for driver %q is permitted for role %q; pass --model explicitly", driver, role)
}
