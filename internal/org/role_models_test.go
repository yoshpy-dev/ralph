package org

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// roleModelsTestConfig is an [org] envelope with both drivers in driver_pool
// and three [org.role_models] entries: leader on claude with opus, reviewer
// on codex with a slug, and director on claude with the CLI's default model.
// No entry exists for "worker".
func roleModelsTestConfig() config.OrgConfig {
	return config.OrgConfig{
		DriverPool: []string{"claude", "codex"},
		RoleModels: map[string]config.OrgRoleModel{
			"leader":   {Driver: "claude", Model: "opus"},
			"reviewer": {Driver: "codex", Model: "gpt-6.1-sol"},
			"director": {Driver: "claude", Model: ""},
		},
	}
}

// TestResolveRoleModel pins the rules of spec FR-3 (plan
// 2026-11-05-org-role-models, D4 and D9, AC4): both flags win; --model alone
// takes the entry's driver; --driver alone takes the entry's model on the
// entry's driver, the CLI's default on another driver when the entry's model
// is "", and is refused naming --model otherwise; neither takes the entry; a
// role with no entry needs both flags; a --model that is not one plain word
// is refused; and the resolved driver must be in driver_pool, named by the
// entry when it came from the entry and in ValidateSpawnEnvelope's words
// when it came from --driver. Flags are trimmed, so a blank flag is an
// omitted one.
func TestResolveRoleModel(t *testing.T) {
	cases := []struct {
		name               string
		mutate             func(*config.OrgConfig)
		role               string
		driverFlag         string
		modelFlag          string
		wantDriver         string
		wantModel          string
		wantErr            []string
		wantErrDoesNotHave []string
	}{
		{name: "both flags override the entry", role: "leader", driverFlag: "codex", modelFlag: "gpt-6-astra", wantDriver: "codex", wantModel: "gpt-6-astra"},
		{name: "both flags for a role with no entry", role: "worker", driverFlag: "claude", modelFlag: "sonnet", wantDriver: "claude", wantModel: "sonnet"},
		{name: "only --model takes the entry's driver", role: "reviewer", modelFlag: "gpt-6-astra", wantDriver: "codex", wantModel: "gpt-6-astra"},
		{name: "only --driver, the entry's driver, takes the entry's model", role: "leader", driverFlag: "claude", wantDriver: "claude", wantModel: "opus"},
		{name: "only --driver, another driver, with a model in the entry is refused naming --model", role: "leader", driverFlag: "codex",
			wantErr: []string{`role "leader" runs on claude in [org.role_models]`, "pass --model for --driver codex"}},
		{name: "only --driver, another driver, with the entry's model empty runs the CLI default", role: "director", driverFlag: "codex", wantDriver: "codex", wantModel: ""},
		{name: "neither flag takes the entry", role: "reviewer", wantDriver: "codex", wantModel: "gpt-6.1-sol"},
		{name: "neither flag takes an entry whose model is empty", role: "director", wantDriver: "claude", wantModel: ""},
		{name: "blank flags count as omitted", role: "leader", driverFlag: "  ", modelFlag: "\t", wantDriver: "claude", wantModel: "opus"},
		{name: "flags are trimmed", role: "worker", driverFlag: " claude ", modelFlag: " sonnet ", wantDriver: "claude", wantModel: "sonnet"},
		{name: "no entry and no flags is refused", role: "worker",
			wantErr: []string{`role "worker" has no [org.role_models].worker entry`, "pass both --driver and --model"}},
		{name: "no entry and only --driver is refused", role: "worker", driverFlag: "claude",
			wantErr: []string{`role "worker" has no [org.role_models].worker entry`, "pass both --driver and --model"}},
		{name: "no entry and only --model is refused", role: "worker", modelFlag: "sonnet",
			wantErr: []string{`role "worker" has no [org.role_models].worker entry`, "pass both --driver and --model"}},
		{name: "a --model with a space is refused", role: "leader", modelFlag: "a b", wantErr: []string{"--model", `model "a b" must be ""`}},
		{name: "a --model starting with - is refused", role: "leader", driverFlag: "claude", modelFlag: "-x", wantErr: []string{"--model", `model "-x" must be ""`}},
		{name: "a --model with $ is refused even with no entry", role: "worker", driverFlag: "claude", modelFlag: "$HOME", wantErr: []string{`model "$HOME" must be ""`}},
		{name: "an entry driver not in driver_pool is refused naming the role and driver_pool",
			mutate: func(c *config.OrgConfig) { c.DriverPool = []string{"claude"} }, role: "reviewer",
			wantErr: []string{`role "reviewer" runs on driver "codex" in [org.role_models].reviewer`, "not in [org].driver_pool [claude]"}},
		{name: "--model alone keeps the entry's driver, so its driver_pool refusal names the entry",
			mutate: func(c *config.OrgConfig) { c.DriverPool = []string{"claude"} }, role: "reviewer", modelFlag: "gpt-6-astra",
			wantErr: []string{"[org.role_models].reviewer", "not in [org].driver_pool [claude]"}},
		{name: "a --driver not in driver_pool keeps the old wording",
			mutate: func(c *config.OrgConfig) { c.DriverPool = []string{"claude"} }, role: "leader", driverFlag: "codex", modelFlag: "gpt-6-astra",
			wantErr: []string{`org: driver "codex" not in [org].driver_pool [claude]`}, wantErrDoesNotHave: []string{"[org.role_models]"}},
		{name: "a --driver matching the entry but not in driver_pool keeps the old wording",
			mutate: func(c *config.OrgConfig) { c.DriverPool = []string{"claude"} }, role: "reviewer", driverFlag: "codex",
			wantErr: []string{`org: driver "codex" not in [org].driver_pool [claude]`}, wantErrDoesNotHave: []string{"[org.role_models]"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := roleModelsTestConfig()
			if tc.mutate != nil {
				tc.mutate(&cfg)
			}
			driver, model, err := ResolveRoleModel(cfg, tc.role, tc.driverFlag, tc.modelFlag)
			if len(tc.wantErr) > 0 {
				if err == nil {
					t.Fatalf("ResolveRoleModel = (%q, %q, nil), want an error containing %q", driver, model, tc.wantErr)
				}
				for _, want := range tc.wantErr {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("error %q should contain %q", err.Error(), want)
					}
				}
				for _, not := range tc.wantErrDoesNotHave {
					if strings.Contains(err.Error(), not) {
						t.Errorf("error %q should not contain %q", err.Error(), not)
					}
				}
				if driver != "" || model != "" {
					t.Errorf("a refusal returned driver %q and model %q, want both empty", driver, model)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveRoleModel: unexpected error: %v", err)
			}
			if driver != tc.wantDriver || model != tc.wantModel {
				t.Fatalf("ResolveRoleModel = (%q, %q), want (%q, %q)", driver, model, tc.wantDriver, tc.wantModel)
			}
		})
	}
}

// TestResolveRoleModel_DefaultsOfConfigDefault pins the two built-in sets as
// ResolveRoleModel sees them with no flags: config.Default's claude set, and
// the codex set config.DefaultRoleModels gives a codex-only driver_pool.
func TestResolveRoleModel_DefaultsOfConfigDefault(t *testing.T) {
	claudeCfg := config.Default().Org
	codexCfg := config.Default().Org
	codexCfg.DriverPool = []string{"codex"}
	codexCfg.RoleModels = config.DefaultRoleModels(codexCfg.DriverPool)
	for _, tc := range []struct {
		name                  string
		cfg                   config.OrgConfig
		role                  string
		wantDriver, wantModel string
	}{
		{"claude set leader", claudeCfg, LeaderIdentity, "claude", "opus"},
		{"claude set director", claudeCfg, DirectorRole, "claude", ""},
		{"codex set reviewer", codexCfg, reviewerRole, "codex", "gpt-6.1-sol"},
		{"codex set director", codexCfg, DirectorRole, "codex", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			driver, model, err := ResolveRoleModel(tc.cfg, tc.role, "", "")
			if err != nil || driver != tc.wantDriver || model != tc.wantModel {
				t.Fatalf("ResolveRoleModel = (%q, %q, %v), want (%q, %q, nil)", driver, model, err, tc.wantDriver, tc.wantModel)
			}
		})
	}
}

// TestDefaultRoleModels_KeysAreTheOrgRoles ties the role names
// config.DefaultRoleModels spells out (internal/config cannot import this
// package) to this package's constants (self-review L3 of plan
// 2026-11-05-org-role-models): the keys of config.Default's [org.role_models]
// and of both built-in sets are exactly LeaderIdentity, implementerRole,
// reviewerRole, and DirectorRole, so renaming one of those constants fails
// here instead of leaving that role with no default entry.
func TestDefaultRoleModels_KeysAreTheOrgRoles(t *testing.T) {
	want := []string{LeaderIdentity, implementerRole, reviewerRole, DirectorRole}
	slices.Sort(want)
	for name, roleModels := range map[string]map[string]config.OrgRoleModel{
		"config.Default":                  config.Default().Org.RoleModels,
		"DefaultRoleModels(codex)":        config.DefaultRoleModels([]string{"codex"}),
		"DefaultRoleModels(claude)":       config.DefaultRoleModels([]string{"claude"}),
		"DefaultRoleModels(codex,claude)": config.DefaultRoleModels([]string{"codex", "claude"}),
	} {
		if got := slices.Sorted(maps.Keys(roleModels)); !slices.Equal(got, want) {
			t.Errorf("%s keys = %q, want %q", name, got, want)
		}
	}
}

// legacyKeysConfig returns a config whose LegacyModelKeys records key for
// each of paths, in order.
func legacyKeysConfig(key string, paths ...string) config.OrgConfig {
	cfg := roleModelsTestConfig()
	for _, p := range paths {
		cfg.LegacyModelKeys = append(cfg.LegacyModelKeys, config.OrgLegacyModelKey{Key: key, Path: p})
	}
	return cfg
}

// TestLegacyRolesErr pins the refusal of a config that still sets
// [org.roles] (plan 2026-11-05-org-role-models, D7, AC8): none with no
// record (a model_pool record alone included), and with one or two files,
// an error naming each file, saying [org.roles] is no longer read and no
// longer restricts a role's models, and pointing at removing it or moving
// the role to [org.role_models]. One file "sets" the key and two files
// "set" it (self-review L5).
func TestLegacyRolesErr(t *testing.T) {
	if err := LegacyRolesErr(roleModelsTestConfig()); err != nil {
		t.Fatalf("LegacyRolesErr with no record = %v, want nil", err)
	}
	if err := LegacyRolesErr(legacyKeysConfig(config.OrgLegacyModelPoolKey, "/r/ralph.toml")); err != nil {
		t.Fatalf("LegacyRolesErr with a model_pool record only = %v, want nil", err)
	}
	for _, tc := range []struct {
		name       string
		paths      []string
		wantPrefix string
	}{
		{"one file", []string{"/r/ralph.toml"}, "org: /r/ralph.toml sets [org.roles], which "},
		{"two files", []string{"/r/ralph.toml", "/r/ralph.local.toml"}, "org: /r/ralph.toml and /r/ralph.local.toml set [org.roles], which "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := LegacyRolesErr(legacyKeysConfig(config.OrgLegacyRolesKey, tc.paths...))
			if err == nil {
				t.Fatal("LegacyRolesErr = nil, want the refusal")
			}
			want := append(append([]string{}, tc.paths...),
				"[org.roles]", "no longer reads", "no longer restricts a role's models", "remove it", "[org.role_models]")
			for _, w := range want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q should contain %q", err.Error(), w)
				}
			}
			if !strings.HasPrefix(err.Error(), tc.wantPrefix) {
				t.Errorf("error %q should start with %q", err.Error(), tc.wantPrefix)
			}
		})
	}
}

// TestLegacyModelPoolNotice pins the one stderr line of a config that still
// sets [org].model_pool (spec FR-6): "" with no record (an [org.roles]
// record alone included), and with one or two files, one line naming each
// file, saying model_pool is no longer read and [org.role_models] sets each
// role's model, and that removing it drops the note. One file "sets" the
// key and two files "set" it (self-review L5).
func TestLegacyModelPoolNotice(t *testing.T) {
	if got := LegacyModelPoolNotice(roleModelsTestConfig()); got != "" {
		t.Fatalf("LegacyModelPoolNotice with no record = %q, want \"\"", got)
	}
	if got := LegacyModelPoolNotice(legacyKeysConfig(config.OrgLegacyRolesKey, "/r/ralph.toml")); got != "" {
		t.Fatalf("LegacyModelPoolNotice with an [org.roles] record only = %q, want \"\"", got)
	}
	for _, tc := range []struct {
		name       string
		paths      []string
		wantPrefix string
	}{
		{"one file", []string{"/r/ralph.toml"}, "org: /r/ralph.toml sets [org].model_pool, which "},
		{"two files", []string{"/r/ralph.toml", "/r/ralph.local.toml"}, "org: /r/ralph.toml and /r/ralph.local.toml set [org].model_pool, which "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := LegacyModelPoolNotice(legacyKeysConfig(config.OrgLegacyModelPoolKey, tc.paths...))
			want := append(append([]string{}, tc.paths...),
				"[org].model_pool", "no longer reads", "[org.role_models] sets each role's driver and model", "remove model_pool to drop this note")
			for _, w := range want {
				if !strings.Contains(got, w) {
					t.Errorf("notice %q should contain %q", got, w)
				}
			}
			if !strings.HasPrefix(got, tc.wantPrefix) {
				t.Errorf("notice %q should start with %q", got, tc.wantPrefix)
			}
			if strings.Contains(got, "\n") {
				t.Errorf("notice %q should be one line", got)
			}
		})
	}
}

// TestModelOrCLIDefault pins how a model is shown: "" as CLIDefaultModelText,
// any other name as it is.
func TestModelOrCLIDefault(t *testing.T) {
	if CLIDefaultModelText != "(CLI default)" {
		t.Fatalf("CLIDefaultModelText = %q, want %q", CLIDefaultModelText, "(CLI default)")
	}
	if got := ModelOrCLIDefault(""); got != CLIDefaultModelText {
		t.Fatalf("ModelOrCLIDefault(\"\") = %q, want %q", got, CLIDefaultModelText)
	}
	if got := ModelOrCLIDefault("opus"); got != "opus" {
		t.Fatalf("ModelOrCLIDefault(opus) = %q, want opus", got)
	}
}
