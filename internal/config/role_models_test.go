package config

import (
	"io"
	"maps"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Tests of [org.role_models] and of the record of the keys it replaces (plan
// 2026-11-05-org-role-models, D1 to D3, D8, D9, AC1 to AC3). A test that
// writes a ralph.local.toml sets RALPH_LOCAL_CONFIG to "" first, as
// local_test.go does, and uses its localTestFiles: a directory outside git,
// where the personal file is the one beside ralph.toml.

// claudeRoleModels and codexRoleModels are the two built-in sets of spec
// FR-2, written out so that a change to DefaultRoleModels fails here.
var (
	claudeRoleModels = map[string]OrgRoleModel{
		"leader":      {Driver: "claude", Model: "opus"},
		"implementer": {Driver: "claude", Model: "opus"},
		"reviewer":    {Driver: "claude", Model: "opus"},
		"director":    {Driver: "claude", Model: ""},
	}
	codexRoleModels = map[string]OrgRoleModel{
		"leader":      {Driver: "codex", Model: "gpt-6.1-sol"},
		"implementer": {Driver: "codex", Model: "gpt-6.1-sol"},
		"reviewer":    {Driver: "codex", Model: "gpt-6.1-sol"},
		"director":    {Driver: "codex", Model: ""},
	}
)

// withRoles returns a copy of base with entries set on top.
func withRoles(base map[string]OrgRoleModel, entries map[string]OrgRoleModel) map[string]OrgRoleModel {
	out := maps.Clone(base)
	maps.Copy(out, entries)
	return out
}

// loadRoleModels loads configPath and fails the test on an error.
func loadRoleModels(t *testing.T, configPath string) Config {
	t.Helper()
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load(%q): %v", configPath, err)
	}
	return cfg
}

// checkRoleModels fails the test unless got equals want.
func checkRoleModels(t *testing.T, what string, got, want map[string]OrgRoleModel) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s: [org.role_models] = %v, want %v", what, got, want)
	}
}

// TestDefaultRoleModels_PicksTheSetByDriverPool covers FR-2: a driver_pool
// with "claude" gives the claude set whatever else it holds, and any other
// (codex only, empty, nil) gives the codex set. Each call returns a new map.
func TestDefaultRoleModels_PicksTheSetByDriverPool(t *testing.T) {
	for _, tc := range []struct {
		driverPool []string
		want       map[string]OrgRoleModel
	}{
		{[]string{"claude", "codex"}, claudeRoleModels},
		{[]string{"codex", "claude"}, claudeRoleModels},
		{[]string{"claude"}, claudeRoleModels},
		{[]string{"codex"}, codexRoleModels},
		{[]string{}, codexRoleModels},
		{nil, codexRoleModels},
	} {
		checkRoleModels(t, "DefaultRoleModels("+strings.Join(tc.driverPool, ",")+")", DefaultRoleModels(tc.driverPool), tc.want)
	}

	first := DefaultRoleModels([]string{"claude"})
	first["leader"] = OrgRoleModel{Driver: "codex", Model: "changed"}
	delete(first, "director")
	checkRoleModels(t, "a second DefaultRoleModels after the first was changed", DefaultRoleModels([]string{"claude"}), claudeRoleModels)
}

// TestDefault_RoleModelsIsTheClaudeSet covers D2: Default(), whose
// driver_pool holds claude, holds the claude set, and a caller that changes
// its map does not change the next Default().
func TestDefault_RoleModelsIsTheClaudeSet(t *testing.T) {
	def := Default()
	if !slices.Contains(def.Org.DriverPool, "claude") {
		t.Fatalf("Default() driver_pool = %v, want claude in it", def.Org.DriverPool)
	}
	checkRoleModels(t, "Default()", def.Org.RoleModels, claudeRoleModels)
	if def.Org.LegacyModelKeys != nil {
		t.Errorf("Default() LegacyModelKeys = %v, want nil", def.Org.LegacyModelKeys)
	}
	def.Org.RoleModels["reviewer"] = OrgRoleModel{Driver: "codex", Model: "changed"}
	checkRoleModels(t, "a second Default()", Default().Org.RoleModels, claudeRoleModels)
}

// TestLoad_RoleModelsFollowTheDriverPoolInEffect covers AC1 and D2: with no
// file, Load returns Default(), the claude set; a driver_pool narrowed to
// claude keeps it; a driver_pool narrowed to codex, in ralph.toml or only in
// ralph.local.toml, gives the codex set, so none of Default()'s claude
// entries survive; a ralph.local.toml that adds claude back gives the claude
// set. A driver_pool Load refuses (empty, or naming another driver) never
// reaches the choice of a set: TestLoad_DriverPoolMustNameKnownDrivers.
func TestLoad_RoleModelsFollowTheDriverPoolInEffect(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	codexOnly := "[org]\ndriver_pool = [\"codex\"]\n"

	t.Run("no file", func(t *testing.T) {
		configPath, _ := localTestFiles(t, "", "")
		cfg := loadRoleModels(t, configPath)
		if !reflect.DeepEqual(cfg, Default()) {
			t.Errorf("Load = %+v, want Default()", cfg)
		}
		checkRoleModels(t, "no file", cfg.Org.RoleModels, claudeRoleModels)
	})
	t.Run("a file that sets no org key", func(t *testing.T) {
		configPath, _ := localTestFiles(t, "[standard]\ncross_review = true\n", "")
		checkRoleModels(t, "no org key", loadRoleModels(t, configPath).Org.RoleModels, claudeRoleModels)
	})
	t.Run("claude only in ralph.toml", func(t *testing.T) {
		configPath, _ := localTestFiles(t, "[org]\ndriver_pool = [\"claude\"]\n", "")
		checkRoleModels(t, "ralph.toml", loadRoleModels(t, configPath).Org.RoleModels, claudeRoleModels)
	})
	t.Run("codex only in ralph.toml", func(t *testing.T) {
		configPath, _ := localTestFiles(t, codexOnly, "")
		checkRoleModels(t, "ralph.toml", loadRoleModels(t, configPath).Org.RoleModels, codexRoleModels)
	})
	t.Run("codex only in ralph.local.toml", func(t *testing.T) {
		for _, ralphToml := range []string{"", "[org]\nmax_seats = 2\n"} {
			configPath, _ := localTestFiles(t, ralphToml, codexOnly)
			checkRoleModels(t, "ralph.toml "+strings.TrimSpace(ralphToml), loadRoleModels(t, configPath).Org.RoleModels, codexRoleModels)
		}
	})
	t.Run("claude added back in ralph.local.toml", func(t *testing.T) {
		configPath, _ := localTestFiles(t, codexOnly, "[org]\ndriver_pool = [\"codex\", \"claude\"]\n")
		checkRoleModels(t, "ralph.local.toml", loadRoleModels(t, configPath).Org.RoleModels, claudeRoleModels)
	})
}

// TestLoad_DriverPoolMustNameKnownDrivers covers the check of [org].driver_pool
// that [org].model_pool's removal left to Load itself: an empty driver_pool and
// an entry other than "claude" or "codex" (letter case included) fail, from
// ralph.toml and from ralph.local.toml, while a pool of either driver or both,
// with a repeated entry too, loads. The merged value is checked, so a
// ralph.local.toml may fix a pool ralph.toml gets wrong, and may break a pool
// ralph.toml gets right.
func TestLoad_DriverPoolMustNameKnownDrivers(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	const (
		emptyErr  = `[org].driver_pool must name at least one of "claude" and "codex"`
		geminiErr = `[org].driver_pool entry "gemini" must be "claude" or "codex"`
	)
	pool := func(array string) string { return "[org]\ndriver_pool = " + array + "\n" }

	for _, tc := range []struct {
		name, array, want string // want "" means the pool loads
	}{
		{"empty", `[]`, emptyErr},
		{"gemini only", `["gemini"]`, geminiErr},
		{"codex and gemini", `["codex", "gemini"]`, geminiErr},
		{"gemini after claude and codex", `["claude", "codex", "gemini"]`, geminiErr},
		{"claude in capitals", `["Claude"]`, `[org].driver_pool entry "Claude" must be "claude" or "codex"`},
		{"an empty entry", `["claude", ""]`, `[org].driver_pool entry "" must be "claude" or "codex"`},
		{"claude and codex", `["claude", "codex"]`, ""},
		{"codex only", `["codex"]`, ""},
		{"claude only", `["claude"]`, ""},
		{"a repeated entry", `["codex", "codex"]`, ""},
	} {
		for _, inLocal := range []bool{false, true} {
			name := tc.name + " in ralph.toml"
			if inLocal {
				name = tc.name + " in ralph.local.toml"
			}
			t.Run(name, func(t *testing.T) {
				ralphToml, local := pool(tc.array), ""
				if inLocal {
					ralphToml, local = local, ralphToml
				}
				configPath, localPath := localTestFiles(t, ralphToml, local)
				_, err := Load(configPath)
				if tc.want == "" {
					if err != nil {
						t.Errorf("Load: %v, want it to load", err)
					}
					return
				}
				want := tc.want
				if inLocal {
					want += " (with " + localPath + " applied)"
				}
				if err == nil || err.Error() != want {
					t.Errorf("Load: err = %v, want %q", err, want)
				}
			})
		}
	}

	t.Run("ralph.local.toml fixes the pool of ralph.toml", func(t *testing.T) {
		configPath, _ := localTestFiles(t, pool(`["gemini"]`), pool(`["codex"]`))
		cfg := loadRoleModels(t, configPath)
		if want := []string{"codex"}; !slices.Equal(cfg.Org.DriverPool, want) {
			t.Errorf("driver_pool = %v, want %v", cfg.Org.DriverPool, want)
		}
		checkRoleModels(t, "fixed in ralph.local.toml", cfg.Org.RoleModels, codexRoleModels)
	})
	t.Run("ralph.local.toml breaks the pool of ralph.toml", func(t *testing.T) {
		configPath, localPath := localTestFiles(t, pool(`["claude"]`), pool(`[]`))
		_, err := Load(configPath)
		if want := emptyErr + " (with " + localPath + " applied)"; err == nil || err.Error() != want {
			t.Errorf("Load: err = %v, want %q", err, want)
		}
	})
}

// TestLoad_RoleModelsKeepWhatTheFilesSet covers AC1 and AC2: a role ralph.toml
// sets replaces that role only, the others keep the default of the
// driver_pool in effect; a ralph.local.toml entry, inline or under its own
// table header, replaces only its role whole; an entry with model = "" and a
// role of the user's own load as written.
func TestLoad_RoleModelsKeepWhatTheFilesSet(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	ralphToml := "[org.role_models]\n" +
		"reviewer = { driver = \"codex\", model = \"gpt-6.1-sol\" }\n" +
		"leader = { driver = \"codex\", model = \"gpt-6-astra\" }\n"

	t.Run("one role in ralph.toml", func(t *testing.T) {
		configPath, _ := localTestFiles(t, "[org.role_models]\nreviewer = { driver = \"codex\", model = \"gpt-6.1-sol\" }\n", "")
		checkRoleModels(t, "one role", loadRoleModels(t, configPath).Org.RoleModels,
			withRoles(claudeRoleModels, map[string]OrgRoleModel{"reviewer": {Driver: "codex", Model: "gpt-6.1-sol"}}))
	})
	t.Run("one table-header role with driver_pool codex", func(t *testing.T) {
		configPath, _ := localTestFiles(t, "[org]\ndriver_pool = [\"codex\"]\n\n[org.role_models.implementer]\ndriver = \"codex\"\nmodel = \"gpt-6-astra\"\n", "")
		checkRoleModels(t, "codex pool", loadRoleModels(t, configPath).Org.RoleModels,
			withRoles(codexRoleModels, map[string]OrgRoleModel{"implementer": {Driver: "codex", Model: "gpt-6-astra"}}))
	})
	for _, local := range []struct{ name, text string }{
		{"inline", "[org.role_models]\nreviewer = { driver = \"claude\", model = \"sonnet\" }\n"},
		{"table header", "[org.role_models.reviewer]\ndriver = \"claude\"\nmodel = \"sonnet\"\n"},
	} {
		t.Run("ralph.local.toml overrides one role, "+local.name, func(t *testing.T) {
			configPath, _ := localTestFiles(t, ralphToml, local.text)
			checkRoleModels(t, local.name, loadRoleModels(t, configPath).Org.RoleModels, withRoles(claudeRoleModels, map[string]OrgRoleModel{
				"reviewer": {Driver: "claude", Model: "sonnet"},
				"leader":   {Driver: "codex", Model: "gpt-6-astra"},
			}))
		})
	}
	t.Run("model empty", func(t *testing.T) {
		configPath, _ := localTestFiles(t, "[org.role_models]\nreviewer = { driver = \"codex\", model = \"\" }\n", "")
		checkRoleModels(t, "model empty", loadRoleModels(t, configPath).Org.RoleModels,
			withRoles(claudeRoleModels, map[string]OrgRoleModel{"reviewer": {Driver: "codex", Model: ""}}))
	})
	t.Run("a role of the user's own", func(t *testing.T) {
		configPath, _ := localTestFiles(t, "[org.role_models]\nqa_bot = { driver = \"claude\", model = \"haiku\" }\n", "")
		checkRoleModels(t, "qa_bot", loadRoleModels(t, configPath).Org.RoleModels,
			withRoles(claudeRoleModels, map[string]OrgRoleModel{"qa_bot": {Driver: "claude", Model: "haiku"}}))
	})
	t.Run("Default() is not changed by a load", func(t *testing.T) {
		configPath, _ := localTestFiles(t, ralphToml, "")
		cfg := loadRoleModels(t, configPath)
		cfg.Org.RoleModels["director"] = OrgRoleModel{Driver: "codex", Model: "changed"}
		checkRoleModels(t, "Default() after a load", Default().Org.RoleModels, claudeRoleModels)
	})
}

// TestLoad_RoleModelsDriverOutsideDriverPoolLoads covers D9 and AC3: Load does
// not check an entry's driver against driver_pool, in ralph.toml or in
// ralph.local.toml; the verbs check the driver they resolve.
func TestLoad_RoleModelsDriverOutsideDriverPoolLoads(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	codexReviewer := "[org.role_models]\nreviewer = { driver = \"codex\", model = \"gpt-6.1-sol\" }\n"
	want := withRoles(claudeRoleModels, map[string]OrgRoleModel{"reviewer": {Driver: "codex", Model: "gpt-6.1-sol"}})

	configPath, _ := localTestFiles(t, "[org]\ndriver_pool = [\"claude\"]\n\n"+codexReviewer, "")
	checkRoleModels(t, "ralph.toml", loadRoleModels(t, configPath).Org.RoleModels, want)

	configPath, _ = localTestFiles(t, "[org]\ndriver_pool = [\"claude\"]\n", codexReviewer)
	checkRoleModels(t, "ralph.local.toml", loadRoleModels(t, configPath).Org.RoleModels, want)
}

// TestLoad_RoleModelsEntryMustSetBothKeys covers AC2 and D3: an entry without
// driver or model, inline or under its own table header, in ralph.toml or in
// ralph.local.toml, fails naming the file and the role. ralph.toml sets a
// complete reviewer under each ralph.local.toml case, so a table-header
// override that sets one key would merge into a complete entry: it fails all
// the same, since each file is checked alone. A table header with no key
// fails too.
func TestLoad_RoleModelsEntryMustSetBothKeys(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	complete := "[org.role_models]\nreviewer = { driver = \"codex\", model = \"gpt-6.1-sol\" }\n"
	for _, entry := range []struct{ name, text string }{
		{"inline without driver", "[org.role_models]\nreviewer = { model = \"opus\" }\n"},
		{"inline without model", "[org.role_models]\nreviewer = { driver = \"claude\" }\n"},
		{"inline without either", "[org.role_models]\nreviewer = {}\n"},
		{"table header without driver", "[org.role_models.reviewer]\nmodel = \"opus\"\n"},
		{"table header without model", "[org.role_models.reviewer]\ndriver = \"claude\"\n"},
		{"table header without either", "[org.role_models.reviewer]\n"},
	} {
		t.Run("ralph.toml, "+entry.name, func(t *testing.T) {
			configPath, _ := localTestFiles(t, entry.text, "")
			_, err := Load(configPath)
			want := configPath + ": [org.role_models].reviewer must set both driver and model"
			if err == nil || !strings.HasPrefix(err.Error(), want) {
				t.Errorf("Load: err = %v, want it to start with %q", err, want)
			}
		})
		t.Run("ralph.local.toml, "+entry.name, func(t *testing.T) {
			configPath, localPath := localTestFiles(t, complete, entry.text)
			_, err := Load(configPath)
			want := localPath + ": [org.role_models].reviewer must set both driver and model"
			if err == nil || !strings.HasPrefix(err.Error(), want) {
				t.Errorf("Load: err = %v, want it to start with %q", err, want)
			}
		})
	}
}

// TestLoad_RoleModelsValues covers AC3: a driver other than "claude" or
// "codex" (letter case included) and a model that is not one word of
// reviewerWordPattern's characters, or that starts with "-", fail naming the
// file, the role, and the value, in either file. A model of a full ID's
// characters loads.
func TestLoad_RoleModelsValues(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	for _, tc := range []struct {
		name, entry, want string // want "" means the entry loads
	}{
		{"driver gemini", `{ driver = "gemini", model = "opus" }`, `[org.role_models].reviewer driver "gemini" must be "claude" or "codex"`},
		{"driver in capitals", `{ driver = "Claude", model = "opus" }`, `[org.role_models].reviewer driver "Claude" must be "claude" or "codex"`},
		{"driver empty", `{ driver = "", model = "opus" }`, `[org.role_models].reviewer driver "" must be "claude" or "codex"`},
		{"model with a space", `{ driver = "claude", model = "a b" }`, `[org.role_models].reviewer model "a b" must be ""`},
		{"model with a quote", `{ driver = "claude", model = 'a"b' }`, `[org.role_models].reviewer model "a\"b" must be ""`},
		{"model with a semicolon", `{ driver = "claude", model = "a;b" }`, `[org.role_models].reviewer model "a;b" must be ""`},
		{"model with a dollar", `{ driver = "claude", model = "a$b" }`, `[org.role_models].reviewer model "a$b" must be ""`},
		{"model with a leading dash", `{ driver = "codex", model = "-c" }`, `[org.role_models].reviewer model "-c" must be ""`},
		{"model of a full ID", `{ driver = "claude", model = "arn:aws:bedrock:us-east-1:1:inference-profile/us.anthropic.claude-opus-4@v1+x" }`, ""},
	} {
		for _, inLocal := range []bool{false, true} {
			name := tc.name + " in ralph.toml"
			if inLocal {
				name = tc.name + " in ralph.local.toml"
			}
			t.Run(name, func(t *testing.T) {
				ralphToml, local := "[org.role_models]\nreviewer = "+tc.entry+"\n", ""
				if inLocal {
					ralphToml, local = local, ralphToml
				}
				configPath, localPath := localTestFiles(t, ralphToml, local)
				path := configPath
				if inLocal {
					path = localPath
				}
				_, err := Load(configPath)
				if tc.want == "" {
					if err != nil {
						t.Errorf("Load: %v, want it to load", err)
					}
					return
				}
				if want := path + ": " + tc.want; err == nil || !strings.HasPrefix(err.Error(), want) {
					t.Errorf("Load: err = %v, want it to start with %q", err, want)
				}
			})
		}
	}
}

// TestCheckOrgModelName covers D8: "" and one word of reviewerWordPattern's
// characters that does not start with "-" pass; a character a shell or the
// CLI's argument parser would read, and a leading "-", fail with an error
// that quotes the name.
func TestCheckOrgModelName(t *testing.T) {
	for _, name := range []string{"", "opus", "gpt-6.1-sol", "claude-opus-4-1", "us.anthropic.claude:0", "a/b+c@d_e"} {
		if err := CheckOrgModelName(name); err != nil {
			t.Errorf("CheckOrgModelName(%q) = %v, want nil", name, err)
		}
	}
	for _, name := range []string{"-x", "--model", "a b", "a\tb", "a\nb", `a"b`, "a'b", "a;b", "a$b", "a`b", `a\b`, "a|b", "é"} {
		err := CheckOrgModelName(name)
		if err == nil {
			t.Errorf("CheckOrgModelName(%q) = nil, want an error", name)
			continue
		}
		if want := "model " + strconv.Quote(name) + ` must be ""`; !strings.HasPrefix(err.Error(), want) {
			t.Errorf("CheckOrgModelName(%q) = %v, want it to start with %q", name, err, want)
		}
	}
}

// TestLoad_LegacyModelKeysRecordsEachFile covers D1: Load records each of
// [org].model_pool and [org.roles] with the file that sets it, ralph.toml
// before ralph.local.toml and model_pool before roles in each, in any shape
// the files may set them today (an inline array, an array of tables, a roles
// table, an empty roles table, keys in capitals), and stays quiet: no error,
// nothing on stderr. A file that sets neither adds nothing, so a config with
// neither keeps nil.
func TestLoad_LegacyModelKeysRecordsEachFile(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	poolAndRoles := "[org]\nmodel_pool = [{ driver = \"claude\", model = \"opus\" }]\n\n[org.roles]\nimplementer = [\"opus\"]\n"
	for _, tc := range []struct {
		name, ralphToml, local string
		want                   func(configPath, localPath string) []OrgLegacyModelKey
	}{
		{
			name: "neither key", ralphToml: "[org]\nmax_seats = 2\n", local: "[org]\nmax_orgs = 3\n",
			want: func(string, string) []OrgLegacyModelKey { return nil },
		},
		{
			name: "both keys in both files", ralphToml: poolAndRoles, local: poolAndRoles,
			want: func(configPath, localPath string) []OrgLegacyModelKey {
				return []OrgLegacyModelKey{
					{Key: OrgLegacyModelPoolKey, Path: configPath},
					{Key: OrgLegacyRolesKey, Path: configPath},
					{Key: OrgLegacyModelPoolKey, Path: localPath},
					{Key: OrgLegacyRolesKey, Path: localPath},
				}
			},
		},
		{
			name: "an array of tables in ralph.toml only", ralphToml: "[[org.model_pool]]\ndriver = \"claude\"\nmodel = \"opus\"\n",
			want: func(configPath, _ string) []OrgLegacyModelKey {
				return []OrgLegacyModelKey{{Key: OrgLegacyModelPoolKey, Path: configPath}}
			},
		},
		{
			name: "roles in ralph.local.toml only", ralphToml: "[org]\nmax_seats = 2\n", local: "[org.roles]\nreviewer = [\"opus\"]\n",
			want: func(_, localPath string) []OrgLegacyModelKey {
				return []OrgLegacyModelKey{{Key: OrgLegacyRolesKey, Path: localPath}}
			},
		},
		{
			name: "an empty roles table, no ralph.toml", local: "[org.roles]\n",
			want: func(_, localPath string) []OrgLegacyModelKey {
				return []OrgLegacyModelKey{{Key: OrgLegacyRolesKey, Path: localPath}}
			},
		},
		{
			name: "keys in capitals", ralphToml: "[ORG]\nMODEL_POOL = [{ driver = \"claude\", model = \"opus\" }]\n\n[ORG.ROLES]\nreviewer = [\"opus\"]\n",
			want: func(configPath, _ string) []OrgLegacyModelKey {
				return []OrgLegacyModelKey{{Key: OrgLegacyModelPoolKey, Path: configPath}, {Key: OrgLegacyRolesKey, Path: configPath}}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configPath, localPath := localTestFiles(t, tc.ralphToml, tc.local)
			var cfg Config
			var err error
			stderr := captureRoleModelsStderr(t, func() { cfg, err = Load(configPath) })
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if stderr != "" {
				t.Errorf("Load wrote %q to stderr, want nothing", stderr)
			}
			if want := tc.want(configPath, localPath); !reflect.DeepEqual(cfg.Org.LegacyModelKeys, want) {
				t.Errorf("LegacyModelKeys = %v, want %v", cfg.Org.LegacyModelKeys, want)
			}
			checkRoleModels(t, tc.name, cfg.Org.RoleModels, claudeRoleModels)
		})
	}
}

// TestLoad_LegacyModelKeysLoadInAnyShape covers spec FR-6 now that Config has
// no field for [org].model_pool or [org.roles]: a file that sets either key
// in a shape the old fields could not hold (a string, an entry without a
// model, a roles value that is not a list, a nested roles table), or in one
// the old checks refused (an empty pool, a duplicate entry, an entry whose
// driver is not in driver_pool, a role naming a model outside the pool),
// loads without an error and is recorded with its own path, whether it is
// ralph.toml or ralph.local.toml. The roles keep the built-in claude set.
func TestLoad_LegacyModelKeysLoadInAnyShape(t *testing.T) {
	t.Setenv(LocalConfigEnv, "")
	pool := []string{OrgLegacyModelPoolKey}
	roles := []string{OrgLegacyRolesKey}
	both := []string{OrgLegacyModelPoolKey, OrgLegacyRolesKey}
	for _, tc := range []struct {
		name, body string
		keys       []string
	}{
		{"model_pool a string", "[org]\nmodel_pool = \"odd shape\"\n", pool},
		{"model_pool entry without a model", "[org]\nmodel_pool = [ { driver = \"x\" } ]\n", pool},
		{"model_pool empty", "[org]\nmodel_pool = []\n", pool},
		{"model_pool duplicate entries", "[org]\nmodel_pool = [\n  { driver = \"claude\", model = \"opus\" },\n  { driver = \"claude\", model = \"opus\" },\n]\n", pool},
		{"model_pool driver outside driver_pool", "[org]\ndriver_pool = [\"claude\"]\nmodel_pool = [{ driver = \"codex\", model = \"gpt-5\" }]\n", pool},
		{"model_pool array of tables with other keys", "[[org.model_pool]]\nname = 1\n", pool},
		{"roles with a list", "[org.roles]\nreviewer = [\"gpt-5.5\"]\n", roles},
		{"roles with a value that is not a list", "[org.roles]\nreviewer = \"opus\"\n", roles},
		{"roles as a key of [org]", "[org]\nroles = 3\n", roles},
		{"roles with a nested table", "[org.roles.reviewer]\nmodels = [\"opus\"]\n", roles},
		{"an empty roles header", "[org.roles]\n", roles},
		{"roles naming a model outside model_pool", "[org]\nmodel_pool = [{ driver = \"claude\", model = \"opus\" }]\n\n[org.roles]\nreviewer = [\"sonnet\"]\n", both},
	} {
		for _, file := range []string{"ralph.toml", "ralph.local.toml"} {
			t.Run(tc.name+" in "+file, func(t *testing.T) {
				ralphToml, local := tc.body, ""
				if file == "ralph.local.toml" {
					ralphToml, local = "[org]\nmax_seats = 2\n", tc.body
				}
				configPath, localPath := localTestFiles(t, ralphToml, local)
				path := configPath
				if file == "ralph.local.toml" {
					path = localPath
				}
				var cfg Config
				var err error
				stderr := captureRoleModelsStderr(t, func() { cfg, err = Load(configPath) })
				if err != nil {
					t.Fatalf("Load: %v", err)
				}
				if stderr != "" {
					t.Errorf("Load wrote %q to stderr, want nothing", stderr)
				}
				var want []OrgLegacyModelKey
				for _, key := range tc.keys {
					want = append(want, OrgLegacyModelKey{Key: key, Path: path})
				}
				if !reflect.DeepEqual(cfg.Org.LegacyModelKeys, want) {
					t.Errorf("LegacyModelKeys = %v, want %v", cfg.Org.LegacyModelKeys, want)
				}
				checkRoleModels(t, tc.name, cfg.Org.RoleModels, claudeRoleModels)
			})
		}
	}
}

// captureRoleModelsStderr returns what fn writes to os.Stderr.
func captureRoleModelsStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	saved := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = saved }()
	fn()
	if err := w.Close(); err != nil {
		t.Fatalf("close the pipe: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read the pipe: %v", err)
	}
	return string(out)
}
