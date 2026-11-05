package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// readInitSettingsTemplate reads the real templates/base/ralph.toml from
// disk. The package's tests replace scaffold.EmbeddedFS with a small fake
// whose ralph.toml has none of the keys init writes, so the embed is not
// read here.
func readInitSettingsTemplate(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "templates", "base", "ralph.toml"))
	if err != nil {
		t.Fatalf("reading the template ralph.toml: %v", err)
	}
	return data
}

// allInitSettings lists every answer combination: the three CLIs times the
// four yes/no answers, 48 in all.
func allInitSettings() []initSettings {
	var all []initSettings
	for _, cli := range []string{initCLIClaude, initCLICodex, initCLIBoth} {
		for bits := range 16 {
			all = append(all, initSettings{
				CLI:          cli,
				PlanReview:   bits&1 != 0,
				ChangeReview: bits&2 != 0,
				CI:           bits&4 != 0,
				AutoMerge:    bits&8 != 0,
			})
		}
	}
	return all
}

func initSettingsName(s initSettings) string {
	return fmt.Sprintf("%s/plan=%t,change=%t,ci=%t,merge=%t", s.CLI, s.PlanReview, s.ChangeReview, s.CI, s.AutoMerge)
}

// loadInitSettingsTOML writes data as ralph.toml in a new directory and
// loads it with config.Load (TestMain turns ralph.local.toml off).
func loadInitSettingsTOML(t *testing.T, data []byte) config.Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ralph.toml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("config.Load: %v\n%s", err, data)
	}
	return cfg
}

// initSettingsTemplateLine returns the index in lines of the line that sets
// key in the table [table]: the first line after the line "[table]" that
// starts with "key = ", before the next line that starts with "[".
func initSettingsTemplateLine(t *testing.T, lines []string, table, key string) int {
	t.Helper()
	header := slices.Index(lines, "["+table+"]")
	if header < 0 {
		t.Fatalf("the template has no [%s] line", table)
	}
	for i := header + 1; i < len(lines) && !strings.HasPrefix(lines[i], "["); i++ {
		if strings.HasPrefix(lines[i], key+" = ") {
			return i
		}
	}
	t.Fatalf("the template has no %s line in [%s]", key, table)
	return -1
}

// initSettingsModelPoolRange returns the index of [org]'s "model_pool = ["
// line and of the "]" line that closes the array.
func initSettingsModelPoolRange(t *testing.T, lines []string) (open, closing int) {
	t.Helper()
	open = slices.Index(lines, "model_pool = [")
	if open < 0 {
		t.Fatal(`the template has no "model_pool = [" line`)
	}
	closing = slices.Index(lines[open:], "]")
	if closing < 0 {
		t.Fatal("the template's model_pool has no closing ] line")
	}
	return open, open + closing
}

// initSettingsWithout returns lines without the lines from index from to
// index to, both included, as a file.
func initSettingsWithout(lines []string, from, to int) []byte {
	kept := slices.Concat(lines[:from], lines[to+1:])
	return []byte(strings.Join(kept, "\n"))
}

func TestInitSettings_EveryAnswerLoadsFromTheTemplateWithTheTableValues(t *testing.T) {
	tmpl := readInitSettingsTemplate(t)
	base := loadInitSettingsTOML(t, tmpl)
	// The checks below lean on these template values.
	if !slices.Equal(base.Org.DriverPool, []string{initCLIClaude, initCLICodex}) {
		t.Fatalf("template driver_pool = %v, want [claude codex]", base.Org.DriverPool)
	}
	for _, driver := range []string{initCLIClaude, initCLICodex} {
		if !slices.ContainsFunc(base.Org.ModelPool, func(e config.OrgModelPoolEntry) bool { return e.Driver == driver }) {
			t.Fatalf("template model_pool has no %s entry: %v", driver, base.Org.ModelPool)
		}
	}

	for _, s := range allInitSettings() {
		t.Run(initSettingsName(s), func(t *testing.T) {
			out, err := applyInitSettings(tmpl, s)
			if err != nil {
				t.Fatalf("applyInitSettings: %v", err)
			}
			got := loadInitSettingsTOML(t, out)

			both := s.CLI == initCLIBoth
			wantDrivers, wantPool := base.Org.DriverPool, base.Org.ModelPool
			if !both {
				wantDrivers = []string{s.CLI}
				wantPool = slices.DeleteFunc(slices.Clone(base.Org.ModelPool), func(e config.OrgModelPoolEntry) bool { return e.Driver != s.CLI })
			}
			if !slices.Equal(got.Org.DriverPool, wantDrivers) {
				t.Errorf("[org].driver_pool = %v, want %v", got.Org.DriverPool, wantDrivers)
			}
			if !slices.Equal(got.Org.ModelPool, wantPool) {
				t.Errorf("[org].model_pool = %v, want %v", got.Org.ModelPool, wantPool)
			}
			for _, e := range got.Org.ModelPool {
				if !slices.Contains(wantDrivers, e.Driver) {
					t.Errorf("[org].model_pool keeps %v, whose driver was not chosen", e)
				}
			}
			if want := s.CLI != initCLICodex; got.Doctor.RequireClaudeCLI != want {
				t.Errorf("[doctor].require_claude_cli = %t, want %t", got.Doctor.RequireClaudeCLI, want)
			}
			if want := s.CLI != initCLIClaude; got.Doctor.RequireCodexCLI != want {
				t.Errorf("[doctor].require_codex_cli = %t, want %t", got.Doctor.RequireCodexCLI, want)
			}
			wantPlan, wantChange := both && s.PlanReview, both && s.ChangeReview
			if got.Standard.PlanAdvisory != wantPlan || got.Org.Review.PlanAdvisory != wantPlan {
				t.Errorf("plan_advisory: [standard] %t, [org.review] %t, want %t in both", got.Standard.PlanAdvisory, got.Org.Review.PlanAdvisory, wantPlan)
			}
			if got.Standard.CrossReview != wantChange || got.Org.Review.CrossReview != wantChange {
				t.Errorf("cross_review: [standard] %t, [org.review] %t, want %t in both", got.Standard.CrossReview, got.Org.Review.CrossReview, wantChange)
			}
			wantRegate := config.RegateChecksLocal
			if s.CI {
				wantRegate = config.RegateChecksCI
			}
			if got.Org.Gate.RegateChecks != wantRegate {
				t.Errorf("[org.gate].regate_checks = %q, want %q", got.Org.Gate.RegateChecks, wantRegate)
			}
			if got.Org.Director.AutoMerge != s.AutoMerge {
				t.Errorf("[org.director].auto_merge = %t, want %t", got.Org.Director.AutoMerge, s.AutoMerge)
			}
			wantWatcher := base.Org.Watchdog.WatcherEnabled
			if s.CLI == initCLICodex {
				wantWatcher = false
			}
			if got.Org.Watchdog.WatcherEnabled != wantWatcher {
				t.Errorf("[org.watchdog].watcher_enabled = %t, want %t", got.Org.Watchdog.WatcherEnabled, wantWatcher)
			}
			// [org.role_models] follows the chosen driver_pool (plan
			// 2026-11-05-org-role-models, D2), and each load records the
			// template's [org].model_pool with the path it read (D1). The
			// template sets model_pool until slice S5 of that plan removes
			// it; drop the model_pool record check and its copy below then.
			if want := config.DefaultRoleModels(wantDrivers); !reflect.DeepEqual(got.Org.RoleModels, want) {
				t.Errorf("[org.role_models] = %v, want %v", got.Org.RoleModels, want)
			}
			if len(got.Org.LegacyModelKeys) != 1 || got.Org.LegacyModelKeys[0].Key != config.OrgLegacyModelPoolKey {
				t.Errorf("LegacyModelKeys = %v, want one %s record", got.Org.LegacyModelKeys, config.OrgLegacyModelPoolKey)
			}

			// Every value the answers do not write loads as the template's.
			rest := got
			rest.Org.DriverPool, rest.Org.ModelPool = base.Org.DriverPool, base.Org.ModelPool
			rest.Doctor.RequireClaudeCLI, rest.Doctor.RequireCodexCLI = base.Doctor.RequireClaudeCLI, base.Doctor.RequireCodexCLI
			rest.Standard.PlanAdvisory, rest.Standard.CrossReview = base.Standard.PlanAdvisory, base.Standard.CrossReview
			rest.Org.Review.PlanAdvisory, rest.Org.Review.CrossReview = base.Org.Review.PlanAdvisory, base.Org.Review.CrossReview
			rest.Org.Gate.RegateChecks = base.Org.Gate.RegateChecks
			rest.Org.Director.AutoMerge = base.Org.Director.AutoMerge
			rest.Org.Watchdog.WatcherEnabled = base.Org.Watchdog.WatcherEnabled
			rest.Org.RoleModels, rest.Org.LegacyModelKeys = base.Org.RoleModels, base.Org.LegacyModelKeys
			if !reflect.DeepEqual(rest, base) {
				t.Errorf("a value the answers do not write changed:\ngot  %+v\nwant %+v", rest, base)
			}
		})
	}
}

func TestInitSettings_KeepsEveryCommentLineAndTheLineCount(t *testing.T) {
	tmpl := readInitSettingsTemplate(t)
	lines := strings.Split(string(tmpl), "\n")
	for _, s := range allInitSettings() {
		out, err := applyInitSettings(tmpl, s)
		if err != nil {
			t.Fatalf("%s: applyInitSettings: %v", initSettingsName(s), err)
		}
		outLines := strings.Split(string(out), "\n")
		if len(outLines) != len(lines) {
			t.Fatalf("%s: %d lines, want the template's %d", initSettingsName(s), len(outLines), len(lines))
		}
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimLeft(line, " \t"), "#") && outLines[i] != line {
				t.Errorf("%s: comment line %d changed:\ngot  %q\nwant %q", initSettingsName(s), i+1, outLines[i], line)
			}
		}
	}
}

func TestInitSettings_ChangesOnlyTheAnsweredLines(t *testing.T) {
	tmpl := readInitSettingsTemplate(t)
	lines := strings.Split(string(tmpl), "\n")
	open, closing := initSettingsModelPoolRange(t, lines)
	for _, s := range allInitSettings() {
		both := s.CLI == initCLIBoth
		regate := "local"
		if s.CI {
			regate = "ci"
		}
		// want maps the index of each line the answers write to its new text.
		want := map[int]string{}
		set := func(table, key, value string) {
			want[initSettingsTemplateLine(t, lines, table, key)] = key + " = " + value
		}
		set("doctor", "require_claude_cli", strconv.FormatBool(s.CLI != initCLICodex))
		set("doctor", "require_codex_cli", strconv.FormatBool(s.CLI != initCLIClaude))
		set("standard", "plan_advisory", strconv.FormatBool(both && s.PlanReview))
		set("standard", "cross_review", strconv.FormatBool(both && s.ChangeReview))
		set("org.review", "plan_advisory", strconv.FormatBool(both && s.PlanReview))
		set("org.review", "cross_review", strconv.FormatBool(both && s.ChangeReview))
		set("org.gate", "regate_checks", `"`+regate+`"`)
		set("org.director", "auto_merge", strconv.FormatBool(s.AutoMerge))
		if !both {
			set("org", "driver_pool", `["`+s.CLI+`"]`)
			drop := initCLICodex
			if s.CLI == initCLICodex {
				drop = initCLIClaude
			}
			for i := open + 1; i < closing; i++ {
				if strings.Contains(lines[i], `driver = "`+drop+`"`) {
					entry := strings.TrimLeft(lines[i], " ")
					want[i] = lines[i][:len(lines[i])-len(entry)] + "# " + entry
				}
			}
		}
		if s.CLI == initCLICodex {
			set("org.watchdog", "watcher_enabled", "false")
		}

		out, err := applyInitSettings(tmpl, s)
		if err != nil {
			t.Fatalf("%s: applyInitSettings: %v", initSettingsName(s), err)
		}
		outLines := strings.Split(string(out), "\n")
		if len(outLines) != len(lines) {
			t.Fatalf("%s: %d lines, want the template's %d", initSettingsName(s), len(outLines), len(lines))
		}
		for i, line := range lines {
			w, answered := want[i]
			if !answered {
				w = line
			}
			if outLines[i] != w {
				t.Errorf("%s: line %d (answered %t):\ngot  %q\nwant %q", initSettingsName(s), i+1, answered, outLines[i], w)
			}
		}
	}
}

func TestInitSettings_MissingKeyIsAnErrorNamingIt(t *testing.T) {
	tmpl := readInitSettingsTemplate(t)
	lines := strings.Split(string(tmpl), "\n")
	allCLIs := []string{initCLIClaude, initCLICodex, initCLIBoth}
	cases := []struct {
		table, key string
		clis       []string
	}{
		{"doctor", "require_claude_cli", allCLIs},
		{"doctor", "require_codex_cli", allCLIs},
		// [org.review] keeps plan_advisory and cross_review, and
		// [standard]'s keep them for the two after: a key of the same name
		// in another table does not stand in for the missing one.
		{"standard", "plan_advisory", allCLIs},
		{"standard", "cross_review", allCLIs},
		{"org.review", "plan_advisory", allCLIs},
		{"org.review", "cross_review", allCLIs},
		{"org.gate", "regate_checks", allCLIs},
		{"org.director", "auto_merge", allCLIs},
		{"org", "driver_pool", []string{initCLIClaude, initCLICodex}},
		{"org.watchdog", "watcher_enabled", []string{initCLICodex}},
	}
	for _, tc := range cases {
		i := initSettingsTemplateLine(t, lines, tc.table, tc.key)
		without := initSettingsWithout(lines, i, i)
		for _, cli := range tc.clis {
			out, err := applyInitSettings(without, initSettings{CLI: cli, AutoMerge: true})
			name := "[" + tc.table + "]." + tc.key
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Errorf("%s: template without %s: err = %v, want one naming %s", cli, name, err, name)
			}
			if out != nil {
				t.Errorf("%s: template without %s: returned %d bytes, want nil", cli, name, len(out))
			}
		}
	}
}

func TestInitSettings_KeyTheAnswerDoesNotWriteMayBeMissing(t *testing.T) {
	tmpl := readInitSettingsTemplate(t)
	lines := strings.Split(string(tmpl), "\n")
	open, closing := initSettingsModelPoolRange(t, lines)
	driverPool := initSettingsTemplateLine(t, lines, "org", "driver_pool")
	watcher := initSettingsTemplateLine(t, lines, "org.watchdog", "watcher_enabled")

	if _, err := applyInitSettings(initSettingsWithout(lines, open, closing), initSettings{CLI: initCLIBoth}); err != nil {
		t.Errorf("both, template without model_pool: %v", err)
	}
	if _, err := applyInitSettings(initSettingsWithout(lines, driverPool, driverPool), initSettings{CLI: initCLIBoth}); err != nil {
		t.Errorf("both, template without driver_pool: %v", err)
	}
	for _, cli := range []string{initCLIClaude, initCLIBoth} {
		if _, err := applyInitSettings(initSettingsWithout(lines, watcher, watcher), initSettings{CLI: cli}); err != nil {
			t.Errorf("%s, template without watcher_enabled: %v", cli, err)
		}
	}
}

func TestInitSettings_KeyInAnotherTableIsAnError(t *testing.T) {
	tmpl := readInitSettingsTemplate(t)
	lines := strings.Split(string(tmpl), "\n")
	merge := initSettingsTemplateLine(t, lines, "org.director", "auto_merge")
	regate := initSettingsTemplateLine(t, lines, "org.gate", "regate_checks")
	if merge > regate {
		t.Fatalf("the template has [org.director].auto_merge (line %d) after [org.gate].regate_checks (line %d)", merge+1, regate+1)
	}
	// Move auto_merge from [org.director] to [org.gate].
	moved := slices.Concat(lines[:merge], lines[merge+1:regate+1], []string{lines[merge]}, lines[regate+1:])
	out, err := applyInitSettings([]byte(strings.Join(moved, "\n")), initSettings{CLI: initCLIBoth})
	if err == nil || !strings.Contains(err.Error(), "[org.director].auto_merge") {
		t.Errorf("auto_merge in [org.gate] only: err = %v, want one naming [org.director].auto_merge", err)
	}
	if out != nil {
		t.Errorf("auto_merge in [org.gate] only: returned %d bytes, want nil", len(out))
	}
}

func TestInitSettings_DuplicateOrCommentedKeyIsAnError(t *testing.T) {
	tmpl := readInitSettingsTemplate(t)
	lines := strings.Split(string(tmpl), "\n")
	merge := initSettingsTemplateLine(t, lines, "org.director", "auto_merge")
	for name, edited := range map[string][]string{
		"set twice": slices.Concat(lines[:merge+1], []string{"auto_merge = false"}, lines[merge+1:]),
		"commented": slices.Concat(lines[:merge], []string{"# " + lines[merge]}, lines[merge+1:]),
	} {
		out, err := applyInitSettings([]byte(strings.Join(edited, "\n")), initSettings{CLI: initCLIBoth})
		if err == nil || !strings.Contains(err.Error(), "[org.director].auto_merge") {
			t.Errorf("auto_merge %s: err = %v, want one naming [org.director].auto_merge", name, err)
		}
		if out != nil {
			t.Errorf("auto_merge %s: returned %d bytes, want nil", name, len(out))
		}
	}
}

func TestInitSettings_ModelPoolProblemsAreErrorsForOneCLI(t *testing.T) {
	tmpl := readInitSettingsTemplate(t)
	lines := strings.Split(string(tmpl), "\n")
	open, closing := initSettingsModelPoolRange(t, lines)
	for _, cli := range []string{initCLIClaude, initCLICodex} {
		var noEntry []string
		for i, line := range lines {
			if i <= open || i >= closing || !strings.Contains(line, `driver = "`+cli+`"`) {
				noEntry = append(noEntry, line)
			}
		}
		// A "]" after the next table's header does not close the array.
		closedLate := append(slices.Concat(lines[:closing], lines[closing+1:]), "]")
		for name, template := range map[string][]byte{
			"no model_pool":                 initSettingsWithout(lines, open, closing),
			"no closing ]":                  initSettingsWithout(lines, closing, closing),
			"] only at the end of the file": []byte(strings.Join(closedLate, "\n")),
			"no entry of the chosen CLI":    []byte(strings.Join(noEntry, "\n")),
		} {
			out, err := applyInitSettings(template, initSettings{CLI: cli})
			if err == nil || !strings.Contains(err.Error(), "[org].model_pool") {
				t.Errorf("%s, %s: err = %v, want one naming [org].model_pool", cli, name, err)
			}
			if out != nil {
				t.Errorf("%s, %s: returned %d bytes, want nil", cli, name, len(out))
			}
		}
	}
}

func TestInitSettings_ValueNotOnOneLineIsAnError(t *testing.T) {
	tmpl := readInitSettingsTemplate(t)
	lines := strings.Split(string(tmpl), "\n")
	i := initSettingsTemplateLine(t, lines, "org", "driver_pool")
	multi := slices.Concat(lines[:i], []string{"driver_pool = [", `  "claude",`, `  "codex",`, "]"}, lines[i+1:])
	for _, cli := range []string{initCLIClaude, initCLICodex} {
		out, err := applyInitSettings([]byte(strings.Join(multi, "\n")), initSettings{CLI: cli})
		if err == nil || !strings.Contains(err.Error(), "[org].driver_pool") {
			t.Errorf("%s, multi-line driver_pool: err = %v, want one naming [org].driver_pool", cli, err)
		}
		if out != nil {
			t.Errorf("%s, multi-line driver_pool: returned %d bytes, want nil", cli, len(out))
		}
	}
}

func TestInitSettings_UnknownCLIIsAnError(t *testing.T) {
	tmpl := readInitSettingsTemplate(t)
	for _, cli := range []string{"", "gemini", "Claude", "claude,codex"} {
		out, err := applyInitSettings(tmpl, initSettings{CLI: cli})
		if err == nil || !strings.Contains(err.Error(), strconv.Quote(cli)) {
			t.Errorf("CLI %q: err = %v, want one naming it", cli, err)
		}
		if out != nil {
			t.Errorf("CLI %q: returned %d bytes, want nil", cli, len(out))
		}
	}
}

// initSettingsCraftedTemplate puts driver_pool after the closing ] of
// model_pool, an array-of-tables header after [org.director] with an
// auto_merge of its own, a comment after a value, an indented key, and
// spaces around an =.
const initSettingsCraftedTemplate = `[doctor]
require_claude_cli = false
require_codex_cli  =  false
[standard]
plan_advisory = false
cross_review = false
[org]
model_pool = [
  { driver = "claude", model = "opus" },
  { driver = "codex", model = "gpt-6-astra" },
]
driver_pool = ["claude", "codex"]
[org.watchdog]
watcher_enabled = true
[org.director]
auto_merge = true # a note
[[org.director.extra]]
auto_merge = true
[org.review]
plan_advisory = false
cross_review = false
[org.gate]
  regate_checks = "local"
`

const initSettingsCraftedCodex = `[doctor]
require_claude_cli = false
require_codex_cli  =  true
[standard]
plan_advisory = false
cross_review = false
[org]
model_pool = [
  # { driver = "claude", model = "opus" },
  { driver = "codex", model = "gpt-6-astra" },
]
driver_pool = ["codex"]
[org.watchdog]
watcher_enabled = false
[org.director]
auto_merge = false # a note
[[org.director.extra]]
auto_merge = true
[org.review]
plan_advisory = false
cross_review = false
[org.gate]
  regate_checks = "ci"
`

func TestInitSettings_CraftedTemplateKeepsTablesLineEndingsAndFinalNewline(t *testing.T) {
	s := initSettings{CLI: initCLICodex, PlanReview: true, ChangeReview: true, CI: true, AutoMerge: false}
	for name, tc := range map[string]struct{ template, want string }{
		"LF":                 {initSettingsCraftedTemplate, initSettingsCraftedCodex},
		"CRLF":               {strings.ReplaceAll(initSettingsCraftedTemplate, "\n", "\r\n"), strings.ReplaceAll(initSettingsCraftedCodex, "\n", "\r\n")},
		"no final newline":   {strings.TrimSuffix(initSettingsCraftedTemplate, "\n"), strings.TrimSuffix(initSettingsCraftedCodex, "\n")},
		"CRLF, no final one": {strings.TrimSuffix(strings.ReplaceAll(initSettingsCraftedTemplate, "\n", "\r\n"), "\r\n"), strings.TrimSuffix(strings.ReplaceAll(initSettingsCraftedCodex, "\n", "\r\n"), "\r\n")},
	} {
		out, err := applyInitSettings([]byte(tc.template), s)
		if err != nil {
			t.Errorf("%s: applyInitSettings: %v", name, err)
			continue
		}
		if string(out) != tc.want {
			t.Errorf("%s:\ngot\n%q\nwant\n%q", name, out, tc.want)
		}
	}
}

func TestInitSettings_DoesNotChangeTheTemplateBytes(t *testing.T) {
	tmpl := []byte(initSettingsCraftedTemplate)
	if _, err := applyInitSettings(tmpl, initSettings{CLI: initCLIClaude, CI: true}); err != nil {
		t.Fatalf("applyInitSettings: %v", err)
	}
	if string(tmpl) != initSettingsCraftedTemplate {
		t.Errorf("the template bytes changed:\n%s", tmpl)
	}
}
