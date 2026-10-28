package config

// TestDefaultsLockStep verifies that defaults written in more than one file
// stay in lock-step. Its "org" and "standard" sub-tests compare the [org]
// and [standard] defaults on two surfaces:
//
//  1. templates/base/ralph.toml — declarative project config
//  2. config.Default()          — Go CLI defaults
//
// For each [org] key of orgTemplateProbe, orgTemplateDrift checks that the
// template sets the key and that its value equals Default()'s, and
// standardTemplateDrift does the same for the [standard] keys of
// standardTemplateProbe. The template is not read with Load: Load starts
// from Default(), so a key the template leaves out would keep Default()'s
// value and pass.
//
// Its "shell_and_skills" sub-test reads scripts/ralph-config.sh (shell
// defaults, VAR="${VAR:-value}" idiom). org.agmsg_home, the one [org] key
// the shell file still carries (RALPH_ORG_AGMSG_HOME, #276), is compared
// there with Default(); with the "org" sub-test's template check, that
// keeps all three surfaces equal. org.review.max_cross_review_cycles is
// compared with RALPH_STANDARD_MAX_PIPELINE_CYCLES's default, the standard
// flow's cap it follows. It also checks that the /plan and /cross-review
// SKILL.md fallback tokens for RALPH_CLAUDE_REVIEWER_MODEL,
// RALPH_CODEX_REVIEWER_MODEL, and RALPH_CODEX_REASONING_EFFORT match the
// shell defaults.
//
// Any mismatch is a test failure naming the surface and key.  The test uses
// runtime.Caller to resolve paths relative to this file so it works from any
// working directory.  Files are resolved relative to the repo root
// (two directories above internal/config/).
//
// If a file is absent (e.g. the package is vendored), the checks that need
// it are skipped with an explanatory message so the test does not break
// in downstream repos that embed internal/config without the meta-repo tree.

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"
)

// repoRoot resolves the repository root from this source file's location.
// internal/config/defaults_sync_test.go → repo root is ../../
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location via runtime.Caller")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

// parseShellDefaults reads a shell file and extracts VAR="${VAR:-value}"
// defaults into a map.  Only top-level assignments of the form
//
//	VARNAME="${VARNAME:-default_value}"
//
// are captured.  This is the idiom scripts/ralph-config.sh uses for its
// defaults.
//
// Go's regexp package does not support backreferences, so we match the full
// pattern with two named groups and verify post-match that the variable name
// in the LHS matches the name inside the parameter expansion.
func parseShellDefaults(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("parseShellDefaults: cannot read %s: %v", path, err)
	}
	// Matches: VARNAME="${ANYNAME:-value}"
	// Captures: [1]=LHS name  [2]=inner name  [3]=default value
	// We then confirm [1]==[2] to enforce the self-referential idiom.
	re := regexp.MustCompile(`^([A-Z_]+)="\$\{([A-Z_]+):-([^}]*)}"`)
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		m := re.FindStringSubmatch(line)
		if len(m) == 4 && m[1] == m[2] {
			result[m[1]] = m[3]
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("parseShellDefaults: scanner error: %v", err)
	}
	return result
}

// mustShell returns the value for the given var from the shell map, failing
// loudly if missing.
func mustShell(t *testing.T, m map[string]string, varName string) string {
	t.Helper()
	v, ok := m[varName]
	if !ok {
		t.Fatalf("scripts/ralph-config.sh: %s not found by VAR=\"${VAR:-default}\" regex — update the regex or the file", varName)
	}
	return v
}

// orgTemplateProbe holds the [org] keys that templates/base/ralph.toml sets,
// with pointer fields: go-toml/v2 leaves a pointer nil when its key is
// absent, so a key the template leaves out is told apart from one set to its
// zero value, such as auto_merge = false (the idiom of orgPoolKeysPresent in
// config.go).
type orgTemplateProbe struct {
	Org struct {
		DriverPool     *[]string            `toml:"driver_pool"`
		ModelPool      *[]OrgModelPoolEntry `toml:"model_pool"`
		MaxSeats       *int                 `toml:"max_seats"`
		MaxOrgs        *int                 `toml:"max_orgs"`
		MaxTotalSeats  *int                 `toml:"max_total_seats"`
		DeadmanMinutes *int                 `toml:"deadman_minutes"`
		AgmsgHome      *string              `toml:"agmsg_home"`
		Permissions    struct {
			Default       *string `toml:"default"`
			CodexVerified *bool   `toml:"codex_verified"`
		} `toml:"permissions"`
		Watchdog struct {
			IntervalSeconds *int    `toml:"interval_seconds"`
			StallMinutes    *int    `toml:"stall_minutes"`
			WatcherEnabled  *bool   `toml:"watcher_enabled"`
			WatcherModel    *string `toml:"watcher_model"`
		} `toml:"watchdog"`
		Director struct {
			Enabled        *bool     `toml:"enabled"`
			AckMinutes     *int      `toml:"ack_minutes"`
			ResolveMinutes *int      `toml:"resolve_minutes"`
			AutoMerge      *bool     `toml:"auto_merge"`
			Wake           *[]string `toml:"wake"`
		} `toml:"director"`
		Approval struct {
			Mode *string `toml:"mode"`
		} `toml:"approval"`
		Review struct {
			PlanAdvisory         *bool `toml:"plan_advisory"`
			CrossReview          *bool `toml:"cross_review"`
			MaxCrossReviewCycles *int  `toml:"max_cross_review_cycles"`
		} `toml:"review"`
	} `toml:"org"`
}

// orgTemplateDrift reads templateData (a ralph.toml) into orgTemplateProbe
// once and compares each of its [org] keys with def. It returns one line
// per key that the template does not set or sets to another value, naming
// the dotted key; none means the two agree.
func orgTemplateDrift(templateData []byte, def Config) ([]string, error) {
	var probe orgTemplateProbe
	if err := toml.Unmarshal(templateData, &probe); err != nil {
		return nil, err
	}
	p, d := probe.Org, def.Org
	return slices.Concat(
		listDrift("org.driver_pool", p.DriverPool, d.DriverPool),
		listDrift("org.model_pool", p.ModelPool, d.ModelPool),
		valueDrift("org.max_seats", p.MaxSeats, d.MaxSeats),
		valueDrift("org.max_orgs", p.MaxOrgs, d.MaxOrgs),
		valueDrift("org.max_total_seats", p.MaxTotalSeats, d.MaxTotalSeats),
		valueDrift("org.deadman_minutes", p.DeadmanMinutes, d.DeadmanMinutes),
		valueDrift("org.agmsg_home", p.AgmsgHome, d.AgmsgHome),
		valueDrift("org.permissions.default", p.Permissions.Default, d.Permissions.Default),
		valueDrift("org.permissions.codex_verified", p.Permissions.CodexVerified, d.Permissions.CodexVerified),
		valueDrift("org.watchdog.interval_seconds", p.Watchdog.IntervalSeconds, d.Watchdog.IntervalSeconds),
		valueDrift("org.watchdog.stall_minutes", p.Watchdog.StallMinutes, d.Watchdog.StallMinutes),
		valueDrift("org.watchdog.watcher_enabled", p.Watchdog.WatcherEnabled, d.Watchdog.WatcherEnabled),
		valueDrift("org.watchdog.watcher_model", p.Watchdog.WatcherModel, d.Watchdog.WatcherModel),
		valueDrift("org.director.enabled", p.Director.Enabled, d.Director.Enabled),
		valueDrift("org.director.ack_minutes", p.Director.AckMinutes, d.Director.AckMinutes),
		valueDrift("org.director.resolve_minutes", p.Director.ResolveMinutes, d.Director.ResolveMinutes),
		valueDrift("org.director.auto_merge", p.Director.AutoMerge, d.Director.AutoMerge),
		listDrift("org.director.wake", p.Director.Wake, d.Director.Wake),
		valueDrift("org.approval.mode", p.Approval.Mode, d.Approval.Mode),
		valueDrift("org.review.plan_advisory", p.Review.PlanAdvisory, d.Review.PlanAdvisory),
		valueDrift("org.review.cross_review", p.Review.CrossReview, d.Review.CrossReview),
		valueDrift("org.review.max_cross_review_cycles", p.Review.MaxCrossReviewCycles, d.Review.MaxCrossReviewCycles),
	), nil
}

// standardTemplateProbe holds the [standard] keys that
// templates/base/ralph.toml sets, with pointer fields, as orgTemplateProbe
// does for [org]: both keys default to false, so only a nil pointer tells a
// key the template leaves out from one it sets to false.
type standardTemplateProbe struct {
	Standard struct {
		PlanAdvisory *bool `toml:"plan_advisory"`
		CrossReview  *bool `toml:"cross_review"`
	} `toml:"standard"`
}

// standardTemplateDrift is orgTemplateDrift for the [standard] keys of
// standardTemplateProbe.
func standardTemplateDrift(templateData []byte, def Config) ([]string, error) {
	var probe standardTemplateProbe
	if err := toml.Unmarshal(templateData, &probe); err != nil {
		return nil, err
	}
	p, d := probe.Standard, def.Standard
	return slices.Concat(
		valueDrift("standard.plan_advisory", p.PlanAdvisory, d.PlanAdvisory),
		valueDrift("standard.cross_review", p.CrossReview, d.CrossReview),
	), nil
}

// valueDrift returns the drift line of one scalar key: none when the
// template sets it to def, else that it is missing or what each side holds.
func valueDrift[T comparable](key string, tmpl *T, def T) []string {
	switch {
	case tmpl == nil:
		return []string{fmt.Sprintf("%s: missing from the template (Default() = %#v)", key, def)}
	case *tmpl != def:
		return []string{fmt.Sprintf("%s: template = %#v, Default() = %#v", key, *tmpl, def)}
	}
	return nil
}

// listDrift is valueDrift for an array key, compared element by element in
// order, so a reordered pool is a drift too.
func listDrift[E comparable](key string, tmpl *[]E, def []E) []string {
	switch {
	case tmpl == nil:
		return []string{fmt.Sprintf("%s: missing from the template (Default() = %v)", key, def)}
	case !slices.Equal(*tmpl, def):
		return []string{fmt.Sprintf("%s: template = %v, Default() = %v", key, *tmpl, def)}
	}
	return nil
}

// TestDefaultsLockStep is the main tripwire.
func TestDefaultsLockStep(t *testing.T) {
	root := repoRoot(t)
	tomlPath := filepath.Join(root, "templates", "base", "ralph.toml")

	// ── [org] and [standard] defaults: templates/base/ralph.toml vs
	// config.Default() ──────────────────────────────────────────────────────
	for _, table := range []struct {
		name  string // the sub-test and the toml table
		drift func([]byte, Config) ([]string, error)
	}{
		{"org", orgTemplateDrift},
		{"standard", standardTemplateDrift},
	} {
		t.Run(table.name, func(t *testing.T) {
			data, err := os.ReadFile(tomlPath)
			if errors.Is(err, fs.ErrNotExist) {
				t.Skipf("templates/base/ralph.toml not found (%v) — skipping the [%s] lock-step check", err, table.name)
			}
			if err != nil {
				t.Fatalf("cannot read templates/base/ralph.toml: %v", err)
			}
			drift, err := table.drift(data, Default())
			if err != nil {
				t.Fatalf("templates/base/ralph.toml: %v", err)
			}
			if len(drift) != 0 {
				t.Errorf("lock-step mismatch between templates/base/ralph.toml and config.Default():\n  %s",
					strings.Join(drift, "\n  "))
			}
		})
	}

	// ── scripts/ralph-config.sh: org.agmsg_home, org.review's cycle cap, and
	// SKILL.md fallbacks ────────────────────────────────────────────────────
	t.Run("shell_and_skills", func(t *testing.T) {
		shellPath := filepath.Join(root, "scripts", "ralph-config.sh")
		if _, err := os.Stat(shellPath); err != nil {
			t.Skipf("scripts/ralph-config.sh not found (%v) — skipping lock-step check (vendored repo?)", err)
		}
		shell := parseShellDefaults(t, shellPath)

		// org.agmsg_home: the shell default against Default(). The template
		// against Default() is the "org" sub-test's job, so when both pass
		// the three surfaces agree, and this sub-test needs no template.
		agmsgShell := mustShell(t, shell, "RALPH_ORG_AGMSG_HOME")
		agmsgDefault := Default().Org.AgmsgHome
		if agmsgShell != agmsgDefault {
			t.Errorf("lock-step mismatch for org.agmsg_home:\n  scripts/ralph-config.sh RALPH_ORG_AGMSG_HOME default = %q\n  config.Default() = %q",
				agmsgShell, agmsgDefault)
		}

		// org.review.max_cross_review_cycles: an org's cap on cross-review
		// runs (the first one included) counts as the standard flow's
		// pipeline cap does, so Default()'s value is
		// RALPH_STANDARD_MAX_PIPELINE_CYCLES's default. The template against Default() is again the "org"
		// sub-test's job.
		cyclesShell := mustShell(t, shell, "RALPH_STANDARD_MAX_PIPELINE_CYCLES")
		cyclesDefault := strconv.Itoa(Default().Org.Review.MaxCrossReviewCycles)
		if cyclesShell != cyclesDefault {
			t.Errorf("lock-step mismatch for org.review.max_cross_review_cycles:\n  scripts/ralph-config.sh RALPH_STANDARD_MAX_PIPELINE_CYCLES default = %q\n  config.Default() = %q",
				cyclesShell, cyclesDefault)
		}

		// ── SKILL.md fallback tokens match shell defaults ─────────────────────
		// Both /plan and /cross-review document `${VAR:-fallback}` tokens for
		// the model/effort env vars they source from scripts/ralph-config.sh.
		// This is table-driven so every (skill, var) pair is checked the same
		// way and every occurrence in the file (not only the first) is
		// verified: for cross-review that means both the codex-exec invocation
		// line and the "CLI execution modes" table row (plan has no such
		// table; its only occurrence is the step 11.c invocation line) must
		// carry the current default (issue #184, AC-4).
		//
		// Skip scope: a single gate below, before the loop, skips the rest of
		// this sub-test when .claude/skills itself is absent (a
		// vendored/downstream repo without the meta-repo's .claude tree).
		// Once that gate has passed, a missing individual SKILL.md is a
		// defect, not an environment difference, so the loop reports it with
		// t.Errorf and continues instead of skipping.
		skillsRoot := filepath.Join(root, ".claude", "skills")
		if _, err := os.Stat(skillsRoot); err != nil {
			t.Skipf(".claude/skills not found (%v) — skipping /plan and /cross-review SKILL.md fallback checks (vendored repo?)", err)
		}

		type skillVarCheck struct {
			skillDir string // under .claude/skills/
			envVar   string
		}
		skillVarChecks := []skillVarCheck{
			{"cross-review", "RALPH_CLAUDE_REVIEWER_MODEL"},
			{"cross-review", "RALPH_CODEX_REVIEWER_MODEL"},
			{"cross-review", "RALPH_CODEX_REASONING_EFFORT"},
			{"plan", "RALPH_CODEX_REVIEWER_MODEL"},
			{"plan", "RALPH_CODEX_REASONING_EFFORT"},
		}

		for _, c := range skillVarChecks {
			skillPath := filepath.Join(root, ".claude", "skills", c.skillDir, "SKILL.md")
			if _, err := os.Stat(skillPath); err != nil {
				t.Errorf(".claude/skills/%s/SKILL.md not found (%v) — skipping %s fallback check", c.skillDir, err, c.envVar)
				continue
			}

			data, err := os.ReadFile(skillPath)
			if err != nil {
				t.Fatalf("cannot read %s/SKILL.md: %v", c.skillDir, err)
			}

			// Pattern: ${VAR:-<fallback>}
			re := regexp.MustCompile(`\$\{` + regexp.QuoteMeta(c.envVar) + `:-([^}]+)\}`)
			matches := re.FindAllSubmatch(data, -1)
			if len(matches) == 0 {
				t.Errorf(".claude/skills/%s/SKILL.md: no ${%s:-<fallback>} pattern found — update the regex if the wording changed", c.skillDir, c.envVar)
				continue
			}

			shellVal := mustShell(t, shell, c.envVar)
			for _, m := range matches {
				skillFallback := string(m[1])
				if skillFallback != shellVal {
					t.Errorf("%s/SKILL.md %s fallback = %q, want %q (scripts/ralph-config.sh %s default)",
						c.skillDir, c.envVar, skillFallback, shellVal, c.envVar)
				}
			}
		}
	})
}

// TestOrgTemplateDrift_DetectsChangedAndMissingKeys pins that
// orgTemplateDrift catches both kinds of drift in a copy of the real
// template: a changed value, and a removed key (auto_merge, codex_verified),
// which reading through Load would miss because Load backfills Default()'s
// value. Each case replaces text that must occur exactly once, so a
// rewording of the template fails the case instead of making it vacuous.
func TestOrgTemplateDrift_DetectsChangedAndMissingKeys(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "templates", "base", "ralph.toml"))
	if errors.Is(err, fs.ErrNotExist) {
		t.Skipf("templates/base/ralph.toml not found (%v)", err)
	}
	if err != nil {
		t.Fatalf("cannot read templates/base/ralph.toml: %v", err)
	}
	template := string(data)

	cases := []struct {
		name       string
		old, new   string // old == "" leaves the template as it is
		wantPrefix string // "" means no drift
	}{
		{
			name: "the real template has no drift",
		},
		{
			name:       "a changed max_seats",
			old:        "\nmax_seats = 5\n",
			new:        "\nmax_seats = 6\n",
			wantPrefix: "org.max_seats: template = 6, Default() = 5",
		},
		{
			name:       "a removed auto_merge, whose default is true",
			old:        "\nauto_merge = true\n",
			new:        "\n",
			wantPrefix: "org.director.auto_merge: missing from the template (Default() = true)",
		},
		{
			name:       "a removed codex_verified, whose default is true",
			old:        "\ncodex_verified = true\n",
			new:        "\n",
			wantPrefix: "org.permissions.codex_verified: missing from the template (Default() = true)",
		},
		{
			name:       "a removed wake",
			old:        "\nwake = [\"pull\"]\n",
			new:        "\n",
			wantPrefix: "org.director.wake: missing from the template",
		},
		{
			name: "two model_pool entries swapped",
			old: "  { driver = \"claude\", model = \"fable\" },\n" +
				"  { driver = \"claude\", model = \"opus\" },\n",
			new: "  { driver = \"claude\", model = \"opus\" },\n" +
				"  { driver = \"claude\", model = \"fable\" },\n",
			wantPrefix: "org.model_pool: template = ",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mutated := template
			if c.old != "" {
				if n := strings.Count(template, c.old); n != 1 {
					t.Fatalf("templates/base/ralph.toml contains %q %d times, want exactly 1 — update this case", c.old, n)
				}
				mutated = strings.Replace(template, c.old, c.new, 1)
			}
			drift, err := orgTemplateDrift([]byte(mutated), Default())
			if err != nil {
				t.Fatalf("orgTemplateDrift: %v", err)
			}
			if c.wantPrefix == "" {
				if len(drift) != 0 {
					t.Errorf("drift = %q, want none", drift)
				}
				return
			}
			if len(drift) != 1 || !strings.HasPrefix(drift[0], c.wantPrefix) {
				t.Errorf("drift = %q, want one line starting with %q", drift, c.wantPrefix)
			}
		})
	}
}

// templateKeyRow is one key that a template drift function compares, as a
// row of checkEachKeyIsComparedOnBothSides.
type templateKeyRow struct {
	key        string        // dotted, as the drift function's lines name it
	tmplValue  string        // a TOML value for the template, unlike its own
	setDefault func(*Config) // changes only this key's field
}

// checkEachKeyIsComparedOnBothSides pins, for every row, that each side
// reaches drift's comparison of the row's key and no other: changing the
// key's field in a Default() copy, changing its value in the template, and
// removing it from the template each give exactly one drift line, naming the
// key. The rows must cover every toml key of probe, the struct drift reads
// the template into, so a key added to the probe needs a row.
func checkEachKeyIsComparedOnBothSides(t *testing.T, drift func([]byte, Config) ([]string, error), probe reflect.Type, rows []templateKeyRow) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "templates", "base", "ralph.toml"))
	if errors.Is(err, fs.ErrNotExist) {
		t.Skipf("templates/base/ralph.toml not found (%v)", err)
	}
	if err != nil {
		t.Fatalf("cannot read templates/base/ralph.toml: %v", err)
	}
	template := string(data)
	if got, err := drift(data, Default()); err != nil || len(got) != 0 {
		t.Fatalf("the real template must match Default() first (TestDefaultsLockStep): drift = %q, err = %v", got, err)
	}

	var rowKeys []string
	for _, r := range rows {
		rowKeys = append(rowKeys, r.key)
	}
	probeKeys := tomlKeys(probe, "")
	if got, want := slices.Sorted(slices.Values(rowKeys)), slices.Sorted(slices.Values(probeKeys)); !slices.Equal(got, want) {
		t.Fatalf("row keys = %q,\nwant the toml keys of %s = %q", got, probe.Name(), want)
	}

	for _, r := range rows {
		t.Run(r.key, func(t *testing.T) {
			dot := strings.LastIndex(r.key, ".")
			table, key := r.key[:dot], r.key[dot+1:]
			wantOneLine := func(mutation, templateText string, def Config, wantPrefix string) {
				t.Helper()
				got, err := drift([]byte(templateText), def)
				if err != nil {
					t.Fatalf("%s: %s: %v", mutation, probe.Name(), err)
				}
				if len(got) != 1 || !strings.HasPrefix(got[0], wantPrefix) {
					t.Errorf("%s: drift = %q, want one line starting with %q", mutation, got, wantPrefix)
				}
			}

			changedDef := Default()
			r.setDefault(&changedDef)
			wantOneLine("Default() changed", template, changedDef, r.key+": template = ")

			changed, err := editTemplateKey(template, table, key, r.tmplValue)
			if err != nil {
				t.Fatalf("templates/base/ralph.toml: %v — update this row", err)
			}
			wantOneLine("template value changed", changed, Default(), r.key+": template = ")

			removed, err := editTemplateKey(template, table, key, "")
			if err != nil {
				t.Fatalf("templates/base/ralph.toml: %v — update this row", err)
			}
			wantOneLine("template key removed", removed, Default(), r.key+": missing from the template")
		})
	}
}

// TestOrgTemplateDrift_EachKeyIsComparedOnBothSides runs
// checkEachKeyIsComparedOnBothSides on orgTemplateDrift. Two keys with the
// same default (for example max_orgs, deadman_minutes, and ack_minutes, all
// 10; max_total_seats and interval_seconds, both 30; watcher_enabled,
// director.enabled, codex_verified, and auto_merge, all true;
// review.plan_advisory and review.cross_review, both false) swapped on
// either side fail here, which a single real-template comparison would
// miss.
func TestOrgTemplateDrift_EachKeyIsComparedOnBothSides(t *testing.T) {
	defaults := Default()
	// Bool values are negated, so a row keeps differing from the template
	// whichever default it has. Other values are ones no default uses.
	rows := []templateKeyRow{
		{"org.driver_pool", `["changed"]`, func(c *Config) { c.Org.DriverPool = append(c.Org.DriverPool, "changed") }},
		{"org.model_pool", `[{ driver = "claude", model = "changed" }]`, func(c *Config) {
			c.Org.ModelPool = append(c.Org.ModelPool, OrgModelPoolEntry{Driver: "claude", Model: "changed"})
		}},
		{"org.max_seats", "987", func(c *Config) { c.Org.MaxSeats++ }},
		{"org.max_orgs", "987", func(c *Config) { c.Org.MaxOrgs++ }},
		{"org.max_total_seats", "987", func(c *Config) { c.Org.MaxTotalSeats++ }},
		{"org.deadman_minutes", "987", func(c *Config) { c.Org.DeadmanMinutes++ }},
		{"org.agmsg_home", `"changed"`, func(c *Config) { c.Org.AgmsgHome += "-changed" }},
		{"org.permissions.default", `"changed"`, func(c *Config) { c.Org.Permissions.Default += "-changed" }},
		{"org.permissions.codex_verified", strconv.FormatBool(!defaults.Org.Permissions.CodexVerified), func(c *Config) {
			c.Org.Permissions.CodexVerified = !c.Org.Permissions.CodexVerified
		}},
		{"org.watchdog.interval_seconds", "987", func(c *Config) { c.Org.Watchdog.IntervalSeconds++ }},
		{"org.watchdog.stall_minutes", "987", func(c *Config) { c.Org.Watchdog.StallMinutes++ }},
		{"org.watchdog.watcher_enabled", strconv.FormatBool(!defaults.Org.Watchdog.WatcherEnabled), func(c *Config) {
			c.Org.Watchdog.WatcherEnabled = !c.Org.Watchdog.WatcherEnabled
		}},
		{"org.watchdog.watcher_model", `"changed"`, func(c *Config) { c.Org.Watchdog.WatcherModel += "-changed" }},
		{"org.director.enabled", strconv.FormatBool(!defaults.Org.Director.Enabled), func(c *Config) {
			c.Org.Director.Enabled = !c.Org.Director.Enabled
		}},
		{"org.director.ack_minutes", "987", func(c *Config) { c.Org.Director.AckMinutes++ }},
		{"org.director.resolve_minutes", "987", func(c *Config) { c.Org.Director.ResolveMinutes++ }},
		{"org.director.auto_merge", strconv.FormatBool(!defaults.Org.Director.AutoMerge), func(c *Config) {
			c.Org.Director.AutoMerge = !c.Org.Director.AutoMerge
		}},
		{"org.director.wake", `["changed"]`, func(c *Config) { c.Org.Director.Wake = append(c.Org.Director.Wake, "changed") }},
		{"org.approval.mode", `"changed"`, func(c *Config) { c.Org.Approval.Mode += "-changed" }},
		{"org.review.plan_advisory", strconv.FormatBool(!defaults.Org.Review.PlanAdvisory), func(c *Config) {
			c.Org.Review.PlanAdvisory = !c.Org.Review.PlanAdvisory
		}},
		{"org.review.cross_review", strconv.FormatBool(!defaults.Org.Review.CrossReview), func(c *Config) {
			c.Org.Review.CrossReview = !c.Org.Review.CrossReview
		}},
		{"org.review.max_cross_review_cycles", "987", func(c *Config) { c.Org.Review.MaxCrossReviewCycles++ }},
	}
	checkEachKeyIsComparedOnBothSides(t, orgTemplateDrift, reflect.TypeFor[orgTemplateProbe](), rows)
}

// TestStandardTemplateDrift_EachKeyIsComparedOnBothSides runs
// checkEachKeyIsComparedOnBothSides on standardTemplateDrift: plan_advisory
// and cross_review, both false by default, swapped on either side fail here.
// The bool values are negated, as in the [org] rows.
func TestStandardTemplateDrift_EachKeyIsComparedOnBothSides(t *testing.T) {
	defaults := Default()
	rows := []templateKeyRow{
		{"standard.plan_advisory", strconv.FormatBool(!defaults.Standard.PlanAdvisory), func(c *Config) {
			c.Standard.PlanAdvisory = !c.Standard.PlanAdvisory
		}},
		{"standard.cross_review", strconv.FormatBool(!defaults.Standard.CrossReview), func(c *Config) {
			c.Standard.CrossReview = !c.Standard.CrossReview
		}},
	}
	checkEachKeyIsComparedOnBothSides(t, standardTemplateDrift, reflect.TypeFor[standardTemplateProbe](), rows)
}

// tomlKeys returns the dotted toml key of each field of t that is not a
// struct, under prefix, reading a struct field as the table of its keys.
func tomlKeys(t reflect.Type, prefix string) []string {
	var keys []string
	for i := range t.NumField() {
		f := t.Field(i)
		name, _, _ := strings.Cut(f.Tag.Get("toml"), ",")
		if prefix != "" {
			name = prefix + "." + name
		}
		if f.Type.Kind() == reflect.Struct {
			keys = append(keys, tomlKeys(f.Type, name)...)
			continue
		}
		keys = append(keys, name)
	}
	return keys
}

// editTemplateKey returns template with the statement that sets key inside
// [table] replaced by `key = value`, or removed when value is "". The
// statement is the key's line and, for an array written over several lines
// (model_pool), the lines up to the one that closes its brackets. The same
// key can be set in other tables (enabled, default), so only [table] is
// searched, from its header to the next one. It is an error unless [table]
// sets key exactly once, so a key renamed in the template fails the caller
// instead of leaving nothing to compare.
func editTemplateKey(template, table, key, value string) (string, error) {
	header := regexp.MustCompile(`^\[([A-Za-z0-9_.]+)\]\s*(#.*)?$`)
	keyLine := regexp.MustCompile(`^` + regexp.QuoteMeta(key) + `\s*=`)
	lines := strings.SplitAfter(template, "\n")
	var out []string
	section, found := "", 0
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSuffix(lines[i], "\n")
		if m := header.FindStringSubmatch(line); m != nil {
			section = m[1]
		}
		if section != table || !keyLine.MatchString(line) {
			out = append(out, lines[i])
			continue
		}
		found++
		// A bracket inside a string would miscount; the template's values
		// have none, and a miscount fails the caller's drift check.
		for depth := strings.Count(line, "[") - strings.Count(line, "]"); depth > 0; {
			i++
			if i == len(lines) {
				return "", fmt.Errorf("[%s] %s: the array is not closed", table, key)
			}
			depth += strings.Count(lines[i], "[") - strings.Count(lines[i], "]")
		}
		if value != "" {
			out = append(out, key+" = "+value+"\n")
		}
	}
	if found != 1 {
		return "", fmt.Errorf("[%s] sets %s %d times, want exactly 1", table, key, found)
	}
	return strings.Join(out, ""), nil
}
