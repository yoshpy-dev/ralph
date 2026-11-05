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
// for a text line that shows a seat's or a director's model (a --json field
// keeps ""). An empty Model is shown as the driver CLI's default whether or
// not the spawn_started event carries ModelCLIDefault, so a seat whose old
// manifest event had no model field shows the same (CLI default).
func ModelOrCLIDefault(model string) string {
	if model == "" {
		return CLIDefaultModelText
	}
	return model
}

// ResolvedRoleModel is what ResolveRoleModel decides for a seat or a
// headless director: its Driver and Model, and DriverFromRoleEntry, true
// when Driver came from the role's [org.role_models] entry (no --driver was
// given) and false when it came from --driver. The CLI passes all three on
// to Spawn (SpawnParams) or StartDirector (DirectorStartParams), whose
// driver_pool check names the role and its entry when DriverFromRoleEntry
// is true (driverPoolErr, envelope.go).
type ResolvedRoleModel struct {
	Driver              string
	Model               string
	DriverFromRoleEntry bool
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
// role and its [org.role_models] key.
//
// It only decides (plan 2026-11-05-org-role-models, D4 and D9): it checks
// neither the characters of --model nor whether the driver is in
// cfg.DriverPool. For a seat, Spawn's ValidateSpawnEnvelope checks both
// after Spawn's idempotent return (spawn.go's AC-3), so a re-run of a seat
// already spawned returns that seat even when driver_pool was narrowed
// since; a headless director, which is not a seat, is checked by
// checkDirectorStart. Both name the role, its entry, and driver_pool when
// the driver from the entry is not in driver_pool (DriverFromRoleEntry).
func ResolveRoleModel(cfg config.OrgConfig, role, driverFlag, modelFlag string) (ResolvedRoleModel, error) {
	driverFlag = strings.TrimSpace(driverFlag)
	modelFlag = strings.TrimSpace(modelFlag)
	entry, hasEntry := cfg.RoleModels[role]
	switch {
	case driverFlag != "" && modelFlag != "":
		return ResolvedRoleModel{Driver: driverFlag, Model: modelFlag}, nil
	case !hasEntry:
		return ResolvedRoleModel{}, fmt.Errorf("org: role %q has no [org.role_models].%s entry: pass both --driver and --model, "+
			"or add [org.role_models].%s = { driver = \"<claude|codex>\", model = \"<model>\" } to ralph.toml", role, role, role)
	case modelFlag != "":
		return ResolvedRoleModel{Driver: entry.Driver, Model: modelFlag, DriverFromRoleEntry: true}, nil
	case driverFlag == "":
		return ResolvedRoleModel{Driver: entry.Driver, Model: entry.Model, DriverFromRoleEntry: true}, nil
	case driverFlag == entry.Driver:
		return ResolvedRoleModel{Driver: driverFlag, Model: entry.Model}, nil
	case entry.Model == "":
		return ResolvedRoleModel{Driver: driverFlag, Model: ""}, nil
	default:
		return ResolvedRoleModel{}, fmt.Errorf("org: role %q runs on %s in [org.role_models]; pass --model for --driver %s", role, entry.Driver, driverFlag)
	}
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

// legacyKeyFilesSet is the subject and verb of the texts about a key still
// set: "<path> sets" for one file and "<path> and <path> set" for more
// (self-review L5 of plan 2026-11-05-org-role-models).
func legacyKeyFilesSet(paths []string) string {
	if len(paths) == 1 {
		return paths[0] + " sets"
	}
	return strings.Join(paths, " and ") + " set"
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
	return fmt.Errorf("org: %s %s, which ralph no longer reads and which no longer restricts a role's models: "+
		"remove it, or move each role's driver and model to [org.role_models] (for example reviewer = { driver = \"claude\", model = \"opus\" })",
		legacyKeyFilesSet(paths), config.OrgLegacyRolesKey)
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
	return fmt.Sprintf("org: %s %s, which ralph no longer reads: [org.role_models] sets each role's driver and model; remove model_pool to drop this note",
		legacyKeyFilesSet(paths), config.OrgLegacyModelPoolKey)
}
