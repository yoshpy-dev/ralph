package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// The doctor checks of plan 2026-11-05-org-role-models (AC9; spec FR-6,
// FR-7): "Org codex model slugs" and --probe-models read [org.role_models],
// and two new checks warn about the keys ralph no longer reads
// ([org].model_pool and [org.roles]) and about role entries whose driver is
// not in [org].driver_pool.

// roleModelsTestConfig returns a Config whose [org.role_models] holds
// entries under the roles role-1, role-2, ... in order, for checks that
// look at the entries' drivers and models, not at their roles.
func roleModelsTestConfig(entries []config.OrgRoleModel) config.Config {
	roleModels := make(map[string]config.OrgRoleModel, len(entries))
	for i, entry := range entries {
		roleModels[fmt.Sprintf("role-%d", i+1)] = entry
	}
	return config.Config{Org: config.OrgConfig{RoleModels: roleModels}}
}

// TestCheckCodexModelSlugs_RoleModelsEmptyModelAndDuplicates pins which
// slugs the check looks up: the model of each codex entry, once per slug,
// sorted; a codex entry with model "" (codex runs its own default) and a
// claude entry are not looked up. A cache missing one slug warns naming it
// only, and with only a "" codex entry there is nothing to look up.
func TestCheckCodexModelSlugs_RoleModelsEmptyModelAndDuplicates(t *testing.T) {
	mixed := []config.OrgRoleModel{
		{Driver: "codex", Model: ""},
		{Driver: "codex", Model: "gpt-9000-retired"},
		{Driver: "codex", Model: "gpt-5.5"},
		{Driver: "codex", Model: "gpt-5.5"},
		{Driver: "claude", Model: "opus"},
	}
	for _, tc := range []struct {
		name       string
		entries    []config.OrgRoleModel
		cached     []string
		wantStatus string
		// wantDetail is the start of Detail, with <cache> for the cache
		// path ("" for no cache lookup: then it is the whole Detail).
		wantDetail string
	}{
		{"a slug missing from the cache", mixed, []string{"gpt-5.5", "opus"}, "warn",
			"1 codex [org.role_models] slug(s) not found in <cache>: gpt-9000-retired (cache written "},
		{"every slug in the cache", mixed, []string{"gpt-5.5", "gpt-9000-retired"}, "pass",
			"2 codex [org.role_models] slug(s) present in <cache> (cache written "},
		{"only a codex entry with model \"\"", []config.OrgRoleModel{{Driver: "codex", Model: ""}, {Driver: "claude", Model: "opus"}}, nil, "info",
			"no codex entry with a model in [org.role_models]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("CODEX_HOME", dir)
			writeCodexModelsCache(t, dir, tc.cached...)

			r := checkCodexModelSlugs(roleModelsTestConfig(tc.entries))
			if r.Status != tc.wantStatus {
				t.Fatalf("status = %q, want %q (detail=%q)", r.Status, tc.wantStatus, r.Detail)
			}
			want := strings.ReplaceAll(tc.wantDetail, "<cache>", filepath.Join(dir, "models_cache.json"))
			if (tc.wantStatus == "info" && r.Detail != want) || !strings.HasPrefix(r.Detail, want) {
				t.Errorf("detail = %q, want it to be or start with %q", r.Detail, want)
			}
		})
	}
}

// probeCall is one Run call a probeRecordingRunner saw.
type probeCall struct {
	name string
	args []string
}

// probeRecordingRunner is a driver.Runner that records every call and
// answers each with "pong", so --probe-models can be checked without a
// real CLI.
type probeRecordingRunner struct {
	calls *[]probeCall
}

func (r probeRecordingRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	*r.calls = append(*r.calls, probeCall{name: name, args: append([]string{}, args...)})
	return "pong", nil
}

// TestCheckOrgModelProbes_RoleModelsPairsDedupedAndEmptyModelWithoutFlag
// pins --probe-models over [org.role_models]: each distinct driver and
// model is probed once, sorted by driver and then by model, a model of ""
// is probed without --model and named "(CLI default)", and a missing CLI
// still gives one skip line counting its distinct pairs.
func TestCheckOrgModelProbes_RoleModelsPairsDedupedAndEmptyModelWithoutFlag(t *testing.T) {
	cfg := config.Config{Org: config.OrgConfig{
		DriverPool: []string{"claude", "codex"},
		RoleModels: map[string]config.OrgRoleModel{
			"leader":      {Driver: "claude", Model: "opus"},
			"implementer": {Driver: "claude", Model: "opus"},
			"reviewer":    {Driver: "codex", Model: "gpt-x"},
			"director":    {Driver: "claude", Model: ""},
			"watcher":     {Driver: "codex", Model: ""},
			"auditor":     {Driver: "codex", Model: ""},
		},
	}}

	t.Run("both CLIs on PATH", func(t *testing.T) {
		dir := t.TempDir()
		writeStubBin(t, dir, "claude", "")
		writeStubBin(t, dir, "codex", "")
		t.Setenv("PATH", dir)

		var calls []probeCall
		results := checkOrgModelProbes(cfg, probeRecordingRunner{calls: &calls})

		wantCalls := []probeCall{
			{"claude", []string{"-p", "ping", "--output-format", "text"}},
			{"claude", []string{"--model", "opus", "-p", "ping", "--output-format", "text"}},
			{"codex", []string{"exec", "--skip-git-repo-check", "ping"}},
			{"codex", []string{"exec", "--model", "gpt-x", "--skip-git-repo-check", "ping"}},
		}
		if !reflect.DeepEqual(calls, wantCalls) {
			t.Errorf("probe calls = %+v, want %+v", calls, wantCalls)
		}
		wantNames := []string{
			"Org model probe (claude/(CLI default))",
			"Org model probe (claude/opus)",
			"Org model probe (codex/(CLI default))",
			"Org model probe (codex/gpt-x)",
		}
		var gotNames []string
		for _, r := range results {
			gotNames = append(gotNames, r.Name)
			if r.Status != "pass" {
				t.Errorf("%s: status = %q, want pass (detail=%q)", r.Name, r.Status, r.Detail)
			}
		}
		if !reflect.DeepEqual(gotNames, wantNames) {
			t.Errorf("probe result names = %q, want %q", gotNames, wantNames)
		}
	})

	t.Run("codex not on PATH", func(t *testing.T) {
		dir := t.TempDir()
		writeStubBin(t, dir, "claude", "")
		t.Setenv("PATH", dir)

		var calls []probeCall
		results := checkOrgModelProbes(cfg, probeRecordingRunner{calls: &calls})
		if len(calls) != 2 || calls[0].name != "claude" || calls[1].name != "claude" {
			t.Errorf("probe calls = %+v, want the two claude probes only", calls)
		}
		if len(results) != 3 {
			t.Fatalf("results = %+v, want two claude probes and one codex skip line", results)
		}
		skip := results[2]
		if skip.Name != "Org model probe (codex)" || skip.Status != "info" || skip.Detail != "codex not installed — skipping 2 model probe(s)" {
			t.Errorf("codex skip line = %+v, want info %q", skip, "codex not installed — skipping 2 model probe(s)")
		}
	})
}

// legacyKeysWant is the Detail checkOrgLegacyModelKeys gives for key set in
// path, spelled out here so the test pins the wording.
func legacyKeysWant(path, key string) string {
	if key == config.OrgLegacyRolesKey {
		return path + " sets [org.roles], which ralph no longer reads and which no longer restricts a role's models; " +
			"`ralph org spawn`, `ralph org start`, and `ralph org director start` refuse while it is set: " +
			"remove it, or move each role's driver and model to [org.role_models]"
	}
	return path + " sets [org].model_pool, which ralph no longer reads: [org.role_models] sets each role's driver and model; remove it"
}

// writeDoctorRoleModelsConfig writes tomlText as dir/ralph.toml and, when
// localText is not "", localText as the ralph.local.toml config.Load reads
// for it (and turns the personal file on, which TestMain turns off), then
// loads it. It returns the Config and the two paths.
func writeDoctorRoleModelsConfig(t *testing.T, tomlText, localText string) (cfg config.Config, tomlPath, localPath string) {
	t.Helper()
	dir := t.TempDir()
	tomlPath = filepath.Join(dir, "ralph.toml")
	if err := os.WriteFile(tomlPath, []byte(tomlText), 0o644); err != nil {
		t.Fatal(err)
	}
	localPath = config.LocalPath(tomlPath)
	if localText != "" {
		if err := os.WriteFile(localPath, []byte(localText), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv(config.LocalConfigEnv, "")
	}
	cfg, err := config.Load(tomlPath)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	return cfg, tomlPath, localPath
}

// TestDoctorCheckOrgLegacyModelKeys covers D1 of the plan through
// config.Load: no record gives one pass line, and each file that sets
// [org].model_pool or [org.roles] gives one warn naming the file and the
// key, ralph.toml's before ralph.local.toml's and model_pool before roles
// within a file.
func TestDoctorCheckOrgLegacyModelKeys(t *testing.T) {
	const (
		modelPool = "[org]\nmodel_pool = [{ driver = \"claude\", model = \"opus\" }]\n"
		roles     = "[org.roles]\nreviewer = [\"opus\"]\n"
	)
	type want struct {
		local bool // the file is ralph.local.toml, not ralph.toml
		key   string
	}
	for _, tc := range []struct {
		name      string
		tomlText  string
		localText string
		want      []want
	}{
		{"none", "[org]\nmax_seats = 3\n", "", nil},
		{"model_pool in ralph.toml", modelPool, "", []want{{false, config.OrgLegacyModelPoolKey}}},
		{"[org.roles] in ralph.toml", roles, "", []want{{false, config.OrgLegacyRolesKey}}},
		{"both in ralph.toml", modelPool + roles, "", []want{{false, config.OrgLegacyModelPoolKey}, {false, config.OrgLegacyRolesKey}}},
		{"both in ralph.local.toml", "", modelPool + roles, []want{{true, config.OrgLegacyModelPoolKey}, {true, config.OrgLegacyRolesKey}}},
		{"model_pool in ralph.toml, [org.roles] in ralph.local.toml", modelPool, roles,
			[]want{{false, config.OrgLegacyModelPoolKey}, {true, config.OrgLegacyRolesKey}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, tomlPath, localPath := writeDoctorRoleModelsConfig(t, tc.tomlText, tc.localText)

			results := checkOrgLegacyModelKeys(cfg)
			wantResults := []checkResult{{Name: "Org legacy model keys", Status: "pass"}}
			if len(tc.want) > 0 {
				wantResults = nil
				for _, w := range tc.want {
					path := tomlPath
					if w.local {
						path = localPath
					}
					wantResults = append(wantResults, checkResult{Name: "Org legacy model keys", Status: "warn", Detail: legacyKeysWant(path, w.key)})
				}
			}
			if !reflect.DeepEqual(results, wantResults) {
				t.Errorf("checkOrgLegacyModelKeys = %+v\nwant %+v", results, wantResults)
			}
		})
	}
}

// TestDoctorCheckOrgRoleModelDrivers covers D9 of the plan: a role entry
// whose driver is not in [org].driver_pool gives one warn per role, sorted
// by role, naming the role, its driver, and driver_pool; none gives one
// pass line. config.Load accepts such an entry, so the last case loads it.
func TestDoctorCheckOrgRoleModelDrivers(t *testing.T) {
	warn := func(role, drv, pool string) checkResult {
		return checkResult{Name: "Org role model drivers", Status: "warn", Detail: fmt.Sprintf(
			"[org.role_models].%s runs on driver %q, which is not in [org].driver_pool %s, "+
				"so launching role %q without --driver and --model is refused: "+
				"add %s to driver_pool, change the entry, or pass both --driver and --model", role, drv, pool, role, drv)}
	}
	pass := []checkResult{{Name: "Org role model drivers", Status: "pass"}}
	codex := config.OrgRoleModel{Driver: "codex", Model: "gpt-6-astra"}
	claude := config.OrgRoleModel{Driver: "claude", Model: "opus"}

	for _, tc := range []struct {
		name string
		org  config.OrgConfig
		want []checkResult
	}{
		{"the defaults", config.Default().Org, pass},
		{"every driver in driver_pool", config.OrgConfig{DriverPool: []string{"codex", "claude"},
			RoleModels: map[string]config.OrgRoleModel{"reviewer": codex, "leader": claude}}, pass},
		{"one role off driver_pool", config.OrgConfig{DriverPool: []string{"claude"},
			RoleModels: map[string]config.OrgRoleModel{"reviewer": codex, "leader": claude}},
			[]checkResult{warn("reviewer", "codex", "[claude]")}},
		{"two roles off driver_pool, sorted", config.OrgConfig{DriverPool: []string{"claude"},
			RoleModels: map[string]config.OrgRoleModel{"reviewer": codex, "implementer": codex, "leader": claude}},
			[]checkResult{warn("implementer", "codex", "[claude]"), warn("reviewer", "codex", "[claude]")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkOrgRoleModelDrivers(config.Config{Org: tc.org}); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("checkOrgRoleModelDrivers = %+v\nwant %+v", got, tc.want)
			}
		})
	}

	t.Run("loaded from ralph.local.toml", func(t *testing.T) {
		cfg, _, _ := writeDoctorRoleModelsConfig(t, "[org]\ndriver_pool = [\"claude\"]\n",
			"[org.role_models]\nreviewer = { driver = \"codex\", model = \"\" }\n")
		want := []checkResult{warn("reviewer", "codex", "[claude]")}
		if got := checkOrgRoleModelDrivers(cfg); !reflect.DeepEqual(got, want) {
			t.Errorf("checkOrgRoleModelDrivers = %+v\nwant %+v", got, want)
		}
	})
}

// TestRunDoctorOpts_RoleModelsChecksWired is the wiring pin through
// runDoctorOpts on an initialized project whose ralph.toml still sets
// [org].model_pool and [org.roles] and puts role reviewer on codex while
// driver_pool has claude only: doctor prints both legacy-key warns, the
// driver warn, the envelope line with each role's driver and model, and a
// slug check that looked up the codex entry's model, and the warns leave
// the exit code alone.
func TestRunDoctorOpts_RoleModelsChecksWired(t *testing.T) {
	setupTestEmbedFS(t)
	Version = "0.1.0-test"

	dir := t.TempDir()
	if err := executeInit(dir, initConfig{ProjectName: "test", Packs: []string{"golang"}}, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	tomlPath := filepath.Join(dir, "ralph.toml")
	data, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "[org") {
		t.Fatalf("the test scaffold's ralph.toml has an [org] table already:\n%s", data)
	}
	orgTables := "\n[org]\ndriver_pool = [\"claude\"]\nmodel_pool = [{ driver = \"claude\", model = \"opus\" }]\n" +
		"\n[org.roles]\nreviewer = [\"opus\"]\n" +
		"\n[org.role_models]\nreviewer = { driver = \"codex\", model = \"gpt-6-astra\" }\n"
	if err := os.WriteFile(tomlPath, append(data, []byte(orgTables)...), 0o644); err != nil {
		t.Fatal(err)
	}

	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, bin := range []string{"claude", "codex", "go"} {
		writeStubBin(t, binDir, bin, "")
	}
	t.Setenv("PATH", binDir)
	codexHome := t.TempDir() // no models_cache.json
	t.Setenv("CODEX_HOME", codexHome)
	setAgmsgHomeOverride(t, filepath.Join(dir, "no-such-agmsg-home"))

	var out string
	if err := captureStdoutText(t, &out, func() error { return runDoctorOpts(dir, false) }); err != nil {
		t.Fatalf("the role_models warnings must not fail doctor: %v\noutput:\n%s", err, out)
	}
	for _, want := range []string{
		"  ⚠ Org legacy model keys: warn — " + legacyKeysWant(tomlPath, config.OrgLegacyModelPoolKey) + "\n",
		"  ⚠ Org legacy model keys: warn — " + legacyKeysWant(tomlPath, config.OrgLegacyRolesKey) + "\n",
		"  ⚠ Org role model drivers: warn — [org.role_models].reviewer runs on driver \"codex\", which is not in [org].driver_pool [claude], ",
		"  ℹ Org envelope: info — role_models: director=claude/(CLI default), implementer=claude/opus, leader=claude/opus, reviewer=codex/gpt-6-astra | max_seats: 5 | ",
		"  ℹ Org codex model slugs: info — codex models cache not found at " + filepath.Join(codexHome, "models_cache.json") + " — skipping\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("doctor output does not contain %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "model_pool: ") {
		t.Errorf("doctor output still has the model_pool envelope line:\n%s", out)
	}
}
