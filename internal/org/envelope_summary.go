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
