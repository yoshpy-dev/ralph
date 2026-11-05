package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// [org.role_models]: the driver and the model of each org role (plan
// 2026-11-05-org-role-models; spec docs/specs/2026-11-05-org-role-models.md,
// FR-1, FR-2, FR-5, FR-6). Load reads the table from ralph.toml and
// ralph.local.toml, checks each file's entries on their own
// (checkRoleModelEntries, D3), fills every role neither file sets from
// DefaultRoleModels of the driver_pool in effect (D2), and records the keys
// ralph no longer reads, [org].model_pool and [org.roles], with the file that
// wrote them (legacyModelKeys, D1).

// OrgRoleModel is one entry of [org.role_models]: the driver CLI a role runs
// under and the model passed to it. Load requires both keys in every entry of
// each file, so an inline entry and a table-header entry of ralph.local.toml
// override a role the same way: whole.
type OrgRoleModel struct {
	// Driver is "claude" or "codex" (orgRoleModelDrivers). Load does not
	// check that it is in [org].driver_pool (D9): the verbs check the driver
	// they resolve once they know which config's values apply, and `ralph
	// doctor` warns about it.
	Driver string `toml:"driver"`
	// Model is the CLI-native model name passed to the driver CLI's --model,
	// a claude alias or a codex slug. "" means the driver CLI is launched
	// without --model, so it runs its own default model. CheckOrgModelName
	// says which names Load accepts.
	Model string `toml:"model"`
}

// OrgLegacyModelKey records that a file Load read sets a key of the [org]
// envelope that ralph no longer reads: Key is OrgLegacyModelPoolKey or
// OrgLegacyRolesKey, and Path is the file that sets it (D1). Load neither
// fails nor warns on such a key; the verbs and `ralph doctor` read the
// record.
type OrgLegacyModelKey struct {
	Key  string
	Path string
}

// The keys OrgLegacyModelKey.Key names, as ralph's messages spell them.
const (
	// OrgLegacyModelPoolKey is [org].model_pool, the pool of driver and
	// model pairs that [org.role_models] replaces.
	OrgLegacyModelPoolKey = "[org].model_pool"
	// OrgLegacyRolesKey is [org.roles], the per-role subsets of model_pool
	// that [org.role_models] replaces.
	OrgLegacyRolesKey = "[org.roles]"
)

// The models of DefaultRoleModels' two sets.
const (
	// defaultClaudeRoleModel is the claude alias of the claude set.
	defaultClaudeRoleModel = "opus"
	// defaultCodexRoleModel is the codex slug of the codex set. Codex has no
	// aliases, so it goes stale when codex retires the model.
	defaultCodexRoleModel = "gpt-6.1-sol"
)

// orgRoleModelDrivers lists the values an [org.role_models] entry's driver may
// take: the driver CLIs internal/org launches seats with.
var orgRoleModelDrivers = []string{"claude", "codex"}

// DefaultRoleModels returns the built-in [org.role_models] for driverPool, the
// [org].driver_pool in effect: when it includes "claude", the claude set
// (leader, implementer, and reviewer on claude with opus, the director on
// claude with the CLI's default model), and otherwise the codex set (the same
// roles on codex with gpt-6.1-sol, the director with the CLI's default
// model). Each call returns a new map. templates/base/ralph.toml shows both
// sets as comments (defaults_sync_role_models_test.go).
//
// internal/config cannot import internal/org, which imports it, so the role
// names are spelled here: "leader" is org.LeaderIdentity, "director" is
// org.DirectorRole, and "implementer" and "reviewer" are the roles of org's
// role prompts (implementerRole and reviewerRole in internal/org/approval.go).
func DefaultRoleModels(driverPool []string) map[string]OrgRoleModel {
	driver, model := "codex", defaultCodexRoleModel
	if slices.Contains(driverPool, "claude") {
		driver, model = "claude", defaultClaudeRoleModel
	}
	return map[string]OrgRoleModel{
		"leader":      {Driver: driver, Model: model},
		"implementer": {Driver: driver, Model: model},
		"reviewer":    {Driver: driver, Model: model},
		"director":    {Driver: driver, Model: ""},
	}
}

// fillDefaultRoleModels sets every role of DefaultRoleModels(org.DriverPool)
// that org.RoleModels lacks, and keeps the roles the files set as written
// (D2). Load runs it once both files are applied, so the default set follows
// the driver_pool in effect, whichever file narrowed it.
func fillDefaultRoleModels(org *OrgConfig) {
	if org.RoleModels == nil {
		org.RoleModels = map[string]OrgRoleModel{}
	}
	for role, entry := range DefaultRoleModels(org.DriverPool) {
		if _, ok := org.RoleModels[role]; !ok {
			org.RoleModels[role] = entry
		}
	}
}

// CheckOrgModelName checks name, the model of an [org.role_models] entry or
// of a --model flag: "" passes (the driver CLI runs its own default model),
// and any other name must match reviewerWordPattern, the shape of
// [reviewer].claude_model, and not start with "-". The name is passed to the
// driver CLI as one argument in a herdr pane or a launch script, so a space,
// a quote, ';', or '$' must not split or expand it, and a leading '-' must
// not read as an option (D8; spec "Security considerations").
func CheckOrgModelName(name string) error {
	if name == "" || (reviewerWordPattern.MatchString(name) && !strings.HasPrefix(name, "-")) {
		return nil
	}
	return fmt.Errorf(`model %q must be "" (the CLI's default model) or one word of A-Z, a-z, 0-9, '.', '_', ':', '@', '/', '+', and '-' that does not start with '-'`, name)
}

// checkRoleModelEntries checks the [org.role_models] entries of data, one
// file Load reads, on their own, before the next file is applied (D3): each
// entry must set both driver and model (model may be ""), driver must be one
// of orgRoleModelDrivers, and model must pass CheckOrgModelName. The keys are
// checked in this file alone because the merged config cannot tell a
// table-header override that sets only model from a complete entry: go-toml
// merges a table-header entry key by key into the entry before it, while an
// inline entry replaces it whole. Roles are checked in sorted order, and an
// error names path and the role.
func checkRoleModelEntries(path string, data []byte) error {
	var probe struct {
		Org struct {
			RoleModels map[string]struct {
				Driver *string `toml:"driver"`
				Model  *string `toml:"model"`
			} `toml:"role_models"`
		} `toml:"org"`
	}
	if err := toml.Unmarshal(data, &probe); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	for _, role := range slices.Sorted(maps.Keys(probe.Org.RoleModels)) {
		entry := probe.Org.RoleModels[role]
		if entry.Driver == nil || entry.Model == nil {
			return fmt.Errorf(`%s: [org.role_models].%s must set both driver and model (model = "" runs the CLI's default model)`, path, role)
		}
		if !slices.Contains(orgRoleModelDrivers, *entry.Driver) {
			return fmt.Errorf(`%s: [org.role_models].%s driver %q must be "claude" or "codex"`, path, role, *entry.Driver)
		}
		if err := CheckOrgModelName(*entry.Model); err != nil {
			return fmt.Errorf("%s: [org.role_models].%s %w", path, role, err)
		}
	}
	return nil
}

// legacyModelKeys returns a record for each key ralph no longer reads that
// data, one file Load reads from path, sets: OrgLegacyModelPoolKey, then
// OrgLegacyRolesKey (D1). The document is read into a map and searched with
// tomlDocHasKey, so a key of any shape counts, an empty [org.roles] table
// included, in any letter case the decoder reads into the key. None gives
// nil.
func legacyModelKeys(path string, data []byte) ([]OrgLegacyModelKey, error) {
	var doc map[string]any
	if err := toml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var keys []OrgLegacyModelKey
	for _, k := range []struct{ dotted, name string }{
		{"org.model_pool", OrgLegacyModelPoolKey},
		{"org.roles", OrgLegacyRolesKey},
	} {
		if tomlDocHasKey(doc, k.dotted) {
			keys = append(keys, OrgLegacyModelKey{Key: k.name, Path: path})
		}
	}
	return keys, nil
}
