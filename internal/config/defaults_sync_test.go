package config

// TestDefaultsLockStep verifies that defaults written in more than one file
// stay in lock-step. Its "org", "standard", and "commented" sub-tests compare
// the [org], [standard], and [reviewer] defaults on two surfaces:
//
//  1. templates/base/ralph.toml — declarative project config
//  2. config.Default()          — Go CLI defaults
//
// For each [org] key of orgTemplateProbe, orgTemplateDrift checks that the
// template sets the key and that its value equals Default()'s, and
// standardTemplateDrift does the same for the [standard] keys of
// standardTemplateProbe. commentedTemplateDrift checks the keys of
// commentedTemplateProbe, which the template writes only as a comment
// (`# key = value`): the template must not set them, and the comment's value
// must equal Default()'s. The template is not read with Load: Load starts
// from Default(), so a key the template leaves out would keep Default()'s
// value and pass.
//
// Its "org_review_cycles" sub-test checks that Default()'s
// org.review.max_cross_review_cycles equals its standard.max_pipeline_cycles,
// the standard flow's cap it follows.
//
// Its "shell_and_skills" sub-test reads scripts/ralph-config.sh (shell
// defaults, VAR="${VAR:-value}" idiom). The shell file carries no [org]
// key; org.review.max_cross_review_cycles is compared with
// RALPH_STANDARD_MAX_PIPELINE_CYCLES's default, the standard flow's cap it
// follows. It also checks that the /plan and /cross-review
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
		Gate struct {
			RegateChecks *string `toml:"regate_checks"`
		} `toml:"gate"`
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
		valueDrift("org.gate.regate_checks", p.Gate.RegateChecks, d.Gate.RegateChecks),
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

// commentedTemplateProbe holds the keys that templates/base/ralph.toml writes
// only as a comment, `# key = value` in their table, with pointer fields, as
// orgTemplateProbe does: a key the template sets is a drift, so only a nil
// pointer passes. ralph.toml is a seed that ralph upgrade does not overwrite,
// so a key the template set would keep a scaffolded project on the old value
// after a release changed Default(); a comment leaves the key to the built-in
// default (plan 2026-10-30-standard-flow-config, D3).
type commentedTemplateProbe struct {
	Standard struct {
		MaxPipelineCycles *int `toml:"max_pipeline_cycles"`
	} `toml:"standard"`
	Reviewer struct {
		ClaudeModel  *string `toml:"claude_model"`
		ClaudeEffort *string `toml:"claude_effort"`
		CodexModel   *string `toml:"codex_model"`
		CodexEffort  *string `toml:"codex_effort"`
	} `toml:"reviewer"`
}

// commentedTemplateDrift is orgTemplateDrift for the keys of
// commentedTemplateProbe: it returns one line per key that the template sets,
// does not comment exactly once in its table, or comments with a value other
// than def's.
func commentedTemplateDrift(templateData []byte, def Config) ([]string, error) {
	var probe commentedTemplateProbe
	if err := toml.Unmarshal(templateData, &probe); err != nil {
		return nil, err
	}
	comments := templateComments(string(templateData))
	s, r := probe.Standard, probe.Reviewer
	return slices.Concat(
		commentedDrift("standard.max_pipeline_cycles", s.MaxPipelineCycles, comments, def.Standard.MaxPipelineCycles),
		commentedDrift("reviewer.claude_model", r.ClaudeModel, comments, def.Reviewer.ClaudeModel),
		commentedDrift("reviewer.claude_effort", r.ClaudeEffort, comments, def.Reviewer.ClaudeEffort),
		commentedDrift("reviewer.codex_model", r.CodexModel, comments, def.Reviewer.CodexModel),
		commentedDrift("reviewer.codex_effort", r.CodexEffort, comments, def.Reviewer.CodexEffort),
	), nil
}

// commentedDrift is valueDrift for a key the template writes only as a
// comment: none when the template does not set it (set is nil) and comments
// holds one value for it, equal to def; else what is wrong.
func commentedDrift[T comparable](key string, set *T, comments map[string][]string, def T) []string {
	values := comments[key]
	switch {
	case set != nil:
		return []string{fmt.Sprintf("%s: set in the template, want it only as a comment (Default() = %#v)", key, def)}
	case len(values) == 0:
		return []string{fmt.Sprintf("%s: missing from the template's comments (Default() = %#v)", key, def)}
	case len(values) > 1:
		return []string{fmt.Sprintf("%s: commented %d times in the template, want once (Default() = %#v)", key, len(values), def)}
	}
	var value struct {
		V T `toml:"v"`
	}
	if err := toml.Unmarshal([]byte("v = "+values[0]), &value); err != nil {
		return []string{fmt.Sprintf("%s: template = %s, which does not read as Default()'s type: %v", key, values[0], err)}
	}
	if value.V != def {
		return []string{fmt.Sprintf("%s: template = %#v, Default() = %#v", key, value.V, def)}
	}
	return nil
}

// templateTableHeader matches a table header line of the template, such as
// [org.director], and captures the table's dotted name.
var templateTableHeader = regexp.MustCompile(`^\[([A-Za-z0-9_.]+)\]\s*(#.*)?$`)

// templateCommentStatement matches a commented-out statement, `# key = value`,
// and captures the key and the value's text.
var templateCommentStatement = regexp.MustCompile(`^#\s*([A-Za-z0-9_-]+)\s*=\s*(.*\S)\s*$`)

// templateComments returns the value text of each commented-out statement of
// template by its dotted key: the key under the table whose header last came
// before the line. A key commented more than once has a value per line.
func templateComments(template string) map[string][]string {
	comments := map[string][]string{}
	table := ""
	for line := range strings.SplitSeq(template, "\n") {
		if m := templateTableHeader.FindStringSubmatch(line); m != nil {
			table = m[1]
			continue
		}
		if m := templateCommentStatement.FindStringSubmatch(line); m != nil && table != "" {
			key := table + "." + m[1]
			comments[key] = append(comments[key], m[2])
		}
	}
	return comments
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

	// ── [org], [standard], and [reviewer] defaults: templates/base/ralph.toml
	// vs config.Default() ────────────────────────────────────────────────────
	for _, table := range []struct {
		name  string // the sub-test, and the toml table its keys are in
		drift func([]byte, Config) ([]string, error)
	}{
		{"org", orgTemplateDrift},
		{"standard", standardTemplateDrift},
		{"commented", commentedTemplateDrift},
	} {
		t.Run(table.name, func(t *testing.T) {
			data, err := os.ReadFile(tomlPath)
			if errors.Is(err, fs.ErrNotExist) {
				t.Skipf("templates/base/ralph.toml not found (%v) — skipping the %s lock-step check", err, table.name)
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

	// ── org.review's cycle cap follows the standard flow's ─────────────────
	t.Run("org_review_cycles", func(t *testing.T) {
		def := Default()
		if org, standard := def.Org.Review.MaxCrossReviewCycles, def.Standard.MaxPipelineCycles; org != standard {
			t.Errorf("lock-step mismatch in config.Default(): org.review.max_cross_review_cycles = %d, standard.max_pipeline_cycles = %d; an org's cap on cross-review runs counts as the standard flow's pipeline cap does",
				org, standard)
		}
	})

	// ── scripts/ralph-config.sh: org.review's cycle cap and SKILL.md
	// fallbacks ─────────────────────────────────────────────────────────────
	t.Run("shell_and_skills", func(t *testing.T) {
		shellPath := filepath.Join(root, "scripts", "ralph-config.sh")
		if _, err := os.Stat(shellPath); err != nil {
			t.Skipf("scripts/ralph-config.sh not found (%v) — skipping lock-step check (vendored repo?)", err)
		}
		shell := parseShellDefaults(t, shellPath)

		// org.review.max_cross_review_cycles: an org's cap on cross-review
		// runs (the first one included) counts as the standard flow's
		// pipeline cap does, so Default()'s value is
		// RALPH_STANDARD_MAX_PIPELINE_CYCLES's default. The template against
		// Default() is again the "org" sub-test's job.
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
		// line and the "CLI execution modes" table row (plan's table names no
		// fallback; its occurrences are the step 11.c codex and claude
		// invocation lines) must carry the current default (issue #184, AC-4;
		// the plan skill's claude fallback since issue #266).
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
			{"plan", "RALPH_CLAUDE_REVIEWER_MODEL"},
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
			name:       "the template set to ci, the value of this repository's own ralph.toml",
			old:        "\nregate_checks = \"local\"\n",
			new:        "\nregate_checks = \"ci\"\n",
			wantPrefix: `org.gate.regate_checks: template = "ci", Default() = "local"`,
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
// key. edit changes or removes the key in the template (editTemplateKey, or
// editTemplateComment for a key written as a comment). The rows must cover
// every toml key of probe, the struct drift reads the template into, so a key
// added to the probe needs a row.
func checkEachKeyIsComparedOnBothSides(t *testing.T, drift func([]byte, Config) ([]string, error), probe reflect.Type, edit func(template, table, key, value string) (string, error), rows []templateKeyRow) {
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

			changed, err := edit(template, table, key, r.tmplValue)
			if err != nil {
				t.Fatalf("templates/base/ralph.toml: %v — update this row", err)
			}
			wantOneLine("template value changed", changed, Default(), r.key+": template = ")

			removed, err := edit(template, table, key, "")
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
		{"org.gate.regate_checks", `"changed"`, func(c *Config) { c.Org.Gate.RegateChecks += "-changed" }},
	}
	checkEachKeyIsComparedOnBothSides(t, orgTemplateDrift, reflect.TypeFor[orgTemplateProbe](), editTemplateKey, rows)
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
	checkEachKeyIsComparedOnBothSides(t, standardTemplateDrift, reflect.TypeFor[standardTemplateProbe](), editTemplateKey, rows)
}

// TestCommentedTemplateDrift_EachKeyIsComparedOnBothSides runs
// checkEachKeyIsComparedOnBothSides on commentedTemplateDrift, editing the
// `# key = value` comments: claude_effort and codex_effort, both "xhigh" by
// default, swapped on either side fail here. Each key the template sets
// instead of commenting it, here by uncommenting its line, is a drift line
// naming the key, so a template that sets a key fails even when the value is
// Default()'s.
func TestCommentedTemplateDrift_EachKeyIsComparedOnBothSides(t *testing.T) {
	defaults := Default()
	rows := []templateKeyRow{
		{"standard.max_pipeline_cycles", "987", func(c *Config) { c.Standard.MaxPipelineCycles++ }},
		{"reviewer.claude_model", `"changed"`, func(c *Config) { c.Reviewer.ClaudeModel += "-changed" }},
		{"reviewer.claude_effort", `"changed"`, func(c *Config) { c.Reviewer.ClaudeEffort += "-changed" }},
		{"reviewer.codex_model", `"changed"`, func(c *Config) { c.Reviewer.CodexModel += "-changed" }},
		{"reviewer.codex_effort", `"changed"`, func(c *Config) { c.Reviewer.CodexEffort += "-changed" }},
	}
	checkEachKeyIsComparedOnBothSides(t, commentedTemplateDrift, reflect.TypeFor[commentedTemplateProbe](), editTemplateComment, rows)

	data, err := os.ReadFile(filepath.Join(repoRoot(t), "templates", "base", "ralph.toml"))
	if err != nil {
		t.Fatalf("cannot read templates/base/ralph.toml: %v", err)
	}
	values := templateComments(string(data))
	for _, r := range rows {
		dot := strings.LastIndex(r.key, ".")
		table, key := r.key[:dot], r.key[dot+1:]
		if len(values[r.key]) != 1 {
			t.Fatalf("templates/base/ralph.toml comments %s %d times, want 1 — update this row", r.key, len(values[r.key]))
		}
		set, err := replaceTemplateComment(string(data), table, key, key+" = "+values[r.key][0])
		if err != nil {
			t.Fatalf("templates/base/ralph.toml: %v — update this row", err)
		}
		got, err := commentedTemplateDrift([]byte(set), defaults)
		if err != nil {
			t.Fatalf("%s uncommented: %v", r.key, err)
		}
		if want := r.key + ": set in the template"; len(got) != 1 || !strings.HasPrefix(got[0], want) {
			t.Errorf("%s uncommented: drift = %q, want one line starting with %q", r.key, got, want)
		}
	}
}

// TestTemplateDrift_EveryStandardAndReviewerKeyIsChecked pins that each key of
// [standard] and [reviewer] is checked against the template, either set
// (standardTemplateProbe) or written as a comment (commentedTemplateProbe), so
// a key added to StandardConfig or ReviewerConfig needs a probe field.
func TestTemplateDrift_EveryStandardAndReviewerKeyIsChecked(t *testing.T) {
	probed := slices.Concat(
		tomlKeys(reflect.TypeFor[standardTemplateProbe](), ""),
		tomlKeys(reflect.TypeFor[commentedTemplateProbe](), ""),
	)
	want := slices.Concat(
		tomlKeys(reflect.TypeFor[StandardConfig](), "standard"),
		tomlKeys(reflect.TypeFor[ReviewerConfig](), "reviewer"),
	)
	if got, want := slices.Sorted(slices.Values(probed)), slices.Sorted(slices.Values(want)); !slices.Equal(got, want) {
		t.Errorf("the template drift probes check %q,\nwant the keys of [standard] and [reviewer] %q", got, want)
	}
}

// TestTemplateDriftProbesHoldNoLocalOnlyKey pins that neither drift check
// requires templates/base/ralph.toml to set a key of LocalOnlyKeys, such as
// org.state_dir: Load refuses a ralph.toml that sets one (plan
// 2026-10-30-org-state-dir-local, D3), so the template only mentions it in a
// comment, and the probes leave it out.
func TestTemplateDriftProbesHoldNoLocalOnlyKey(t *testing.T) {
	probeKeys := slices.Concat(
		tomlKeys(reflect.TypeFor[orgTemplateProbe](), ""),
		tomlKeys(reflect.TypeFor[standardTemplateProbe](), ""),
	)
	for _, key := range LocalOnlyKeys() {
		if slices.Contains(probeKeys, key) {
			t.Errorf("%s is in a template drift probe, but it is local-only (LocalOnlyKeys): the template cannot set it", key)
		}
	}
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
	keyLine := regexp.MustCompile(`^` + regexp.QuoteMeta(key) + `\s*=`)
	lines := strings.SplitAfter(template, "\n")
	var out []string
	section, found := "", 0
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSuffix(lines[i], "\n")
		if m := templateTableHeader.FindStringSubmatch(line); m != nil {
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

// editTemplateComment is editTemplateKey for a key the template writes only
// as a comment: it replaces the `# key = ...` line inside [table] with
// `# key = value`, or removes it when value is "".
func editTemplateComment(template, table, key, value string) (string, error) {
	line := ""
	if value != "" {
		line = "# " + key + " = " + value
	}
	return replaceTemplateComment(template, table, key, line)
}

// replaceTemplateComment returns template with the `# key = ...` line inside
// [table] replaced by line, or removed when line is "". It is an error unless
// [table] comments key exactly once, as templateComments reads the comments.
func replaceTemplateComment(template, table, key, line string) (string, error) {
	lines := strings.SplitAfter(template, "\n")
	var out []string
	section, found := "", 0
	for _, l := range lines {
		text := strings.TrimSuffix(l, "\n")
		if m := templateTableHeader.FindStringSubmatch(text); m != nil {
			section = m[1]
		}
		if m := templateCommentStatement.FindStringSubmatch(text); m == nil || section != table || m[1] != key {
			out = append(out, l)
			continue
		}
		found++
		if line != "" {
			out = append(out, line+"\n")
		}
	}
	if found != 1 {
		return "", fmt.Errorf("[%s] comments %s %d times, want exactly 1", table, key, found)
	}
	return strings.Join(out, ""), nil
}
