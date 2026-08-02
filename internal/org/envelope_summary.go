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
// whose Driver matches driver, in cfg.ModelPool's declared order. This is
// `ralph org start`'s --model default (internal/cli/org.go's
// newOrgStartCmd) when the caller omits --model: rather than picking an
// opinionated hardcoded alias, the org's own model_pool -- already the
// allowlist ValidateSpawnEnvelope checks Spawn's request against -- is the
// single source of truth for "what's available", so the default can never
// itself be out-of-pool. An error is returned when no model_pool entry
// matches driver, so `org start` fails fast with an actionable message
// instead of falling through to an empty --model that Spawn would reject
// with a less specific "model \"\" not in [org].model_pool" error.
func DefaultModelForDriver(cfg config.OrgConfig, driver string) (string, error) {
	for _, entry := range cfg.ModelPool {
		if entry.Driver == driver {
			return entry.Model, nil
		}
	}
	return "", fmt.Errorf("org: no [org].model_pool entry for driver %q; pass --model explicitly", driver)
}
