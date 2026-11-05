package org

import (
	"fmt"
	"slices"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// The driver and the model of a seat or a headless director, from
// [org.role_models] and the --driver and --model flags (plan
// 2026-11-05-org-role-models, D4, D5, D7, D9; spec
// docs/specs/2026-11-05-org-role-models.md, FR-3, FR-5, FR-6).

// CLIDefaultModelText is how ralph shows a model of "": no --model is passed,
// so the driver CLI runs its own default model.
const CLIDefaultModelText = "(CLI default)"

// ModelOrCLIDefault returns model, or CLIDefaultModelText when model is "",
// for a line that shows a seat's or a director's model.
func ModelOrCLIDefault(model string) string {
	if model == "" {
		return CLIDefaultModelText
	}
	return model
}

// ResolveRoleModel returns the driver and the model a seat or a headless
// director of role runs with, from cfg.RoleModels and the --driver and
// --model flags (driverFlag and modelFlag; each is trimmed, and "" means the
// flag was omitted), by the rules of spec FR-3:
//
//   - both flags: those values.
//   - --model only: the driver of role's entry, with that model.
//   - --driver only: the entry's model when the driver is the entry's; for
//     another driver, model "" (the CLI's default) when the entry's model is
//     "", and otherwise a refusal that asks for --model, since the entry's
//     model belongs to the entry's driver.
//   - neither: role's entry as it is.
//
// A role with no entry needs both flags; otherwise it is refused, naming the
// role and its [org.role_models] key. A non-empty --model must pass
// config.CheckOrgModelName. The driver resolved must be in cfg.DriverPool
// (D9: config.Load does not check an entry's driver against driver_pool,
// since which config's values apply is decided after loading); a driver
// from the entry is refused naming the role, the entry, and driver_pool,
// and a driver from --driver with the wording ValidateSpawnEnvelope uses.
func ResolveRoleModel(cfg config.OrgConfig, role, driverFlag, modelFlag string) (driver, model string, err error) {
	driverFlag = strings.TrimSpace(driverFlag)
	modelFlag = strings.TrimSpace(modelFlag)
	if modelFlag != "" {
		if err := config.CheckOrgModelName(modelFlag); err != nil {
			return "", "", fmt.Errorf("org: --model: %w", err)
		}
	}
	entry, hasEntry := cfg.RoleModels[role]
	fromEntry := false
	switch {
	case driverFlag != "" && modelFlag != "":
		driver, model = driverFlag, modelFlag
	case !hasEntry:
		return "", "", fmt.Errorf("org: role %q has no [org.role_models].%s entry: pass both --driver and --model, "+
			"or add [org.role_models].%s = { driver = \"<claude|codex>\", model = \"<model>\" } to ralph.toml", role, role, role)
	case modelFlag != "":
		driver, model, fromEntry = entry.Driver, modelFlag, true
	case driverFlag == "":
		driver, model, fromEntry = entry.Driver, entry.Model, true
	case driverFlag == entry.Driver:
		driver, model = driverFlag, entry.Model
	case entry.Model == "":
		driver, model = driverFlag, ""
	default:
		return "", "", fmt.Errorf("org: role %q runs on %s in [org.role_models]; pass --model for --driver %s", role, entry.Driver, driverFlag)
	}
	if !driverInPool(cfg, driver) {
		if fromEntry {
			return "", "", fmt.Errorf("org: role %q runs on driver %q in [org.role_models].%s, which is not in [org].driver_pool %v: "+
				"add %s to driver_pool, change the entry, or pass both --driver and --model", role, driver, role, cfg.DriverPool, driver)
		}
		return "", "", fmt.Errorf("org: driver %q not in [org].driver_pool %v", driver, cfg.DriverPool)
	}
	return driver, model, nil
}

// legacyKeyPaths returns the files cfg.LegacyModelKeys records as setting
// key, in the order config.Load read them.
func legacyKeyPaths(cfg config.OrgConfig, key string) []string {
	var paths []string
	for _, k := range cfg.LegacyModelKeys {
		if k.Key == key {
			paths = append(paths, k.Path)
		}
	}
	return slices.Compact(paths)
}

// LegacyRolesErr returns the refusal of a config that still sets
// [org.roles] (config.OrgLegacyRolesKey in cfg.LegacyModelKeys), naming each
// file that sets it, and nil otherwise (D7; spec FR-6). [org.roles] narrowed
// a role to some models of [org].model_pool; ralph no longer reads it, so a
// repo that relied on it would lose the restriction without a word. Spawn,
// and so `ralph org start`, refuses with it (retiredRoleConfigErr), and so
// does `ralph org director start`; config.Load does not, so status, stop,
// and disband keep working.
func LegacyRolesErr(cfg config.OrgConfig) error {
	paths := legacyKeyPaths(cfg, config.OrgLegacyRolesKey)
	if len(paths) == 0 {
		return nil
	}
	return fmt.Errorf("org: %s sets %s, which ralph no longer reads and which no longer restricts a role's models: "+
		"remove it, or move each role's driver and model to [org.role_models] (for example reviewer = { driver = \"claude\", model = \"opus\" })",
		strings.Join(paths, " and "), config.OrgLegacyRolesKey)
}

// LegacyModelPoolNotice returns the one line spawn, start, and director
// start write on stderr when cfg still sets [org].model_pool
// (config.OrgLegacyModelPoolKey in cfg.LegacyModelKeys), naming each file
// that sets it, and "" otherwise (spec FR-6). The key no longer changes
// anything, so the verbs go on.
func LegacyModelPoolNotice(cfg config.OrgConfig) string {
	paths := legacyKeyPaths(cfg, config.OrgLegacyModelPoolKey)
	if len(paths) == 0 {
		return ""
	}
	return fmt.Sprintf("org: %s sets %s, which ralph no longer reads: [org.role_models] sets each role's driver and model; remove model_pool to drop this note",
		strings.Join(paths, " and "), config.OrgLegacyModelPoolKey)
}
