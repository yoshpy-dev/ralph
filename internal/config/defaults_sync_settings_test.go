package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"
)

// The lock-step checks of the settings the standard flow's scripts and skills
// read: the keys templates/base/ralph.toml writes only as a comment
// ([standard].max_pipeline_cycles and base_branch, [reviewer], [plan_visual],
// and [verify]), the table of defaults of scripts/ralph-config.sh, and the
// SKILL.md bodies that read it, with the tests that pin them.
// TestDefaultsLockStep in defaults_sync_test.go runs the checks; the helpers
// they share with its [org] checks (templateTableHeader, templateKeyRow,
// checkEachKeyIsComparedOnBothSides, tomlKeys) stay there.

// shellSettingDefault matches one line of the table of defaults in
// scripts/ralph-config.sh, `<key>) printf '%s\n' '<value>' ;;` in
// ralph_setting_default, and captures the dotted key and the value. The
// helper's doc comment names this test as a reader of that form.
var shellSettingDefault = regexp.MustCompile(`^\s*([a-z_]+(?:\.[a-z_]+)+)\) printf '%s\\n' '([^']*)' ;;$`)

// parseShellSettingDefaults reads the table of defaults of the shell file at
// path into a map by key. A key on two lines is a test failure, so the map
// cannot hide one of them.
func parseShellSettingDefaults(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("parseShellSettingDefaults: cannot read %s: %v", path, err)
	}
	result := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		m := shellSettingDefault.FindStringSubmatch(scanner.Text())
		if m == nil {
			continue
		}
		if _, dup := result[m[1]]; dup {
			t.Errorf("%s: the table of defaults holds %s twice", path, m[1])
		}
		result[m[1]] = m[2]
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("parseShellSettingDefaults: scanner error: %v", err)
	}
	return result
}

// commentedTemplateProbe holds the keys that templates/base/ralph.toml writes
// only as a comment, `# key = value` in their table, with pointer fields, as
// orgTemplateProbe does: a key the template sets is a drift, so only a nil
// pointer passes. ralph.toml is a seed that ralph upgrade does not overwrite,
// so a key the template set would keep a scaffolded project on the old value
// after a release changed Default(); a comment leaves the key to the built-in
// default (plan 2026-10-30-standard-flow-config, D3). The local-only keys the
// scripts read (standard.base_branch, [plan_visual], and [verify]) are here
// too: Load refuses a ralph.toml that sets one (LocalOnlyKeys), so the
// template may only show it as a comment, the line to copy into
// ralph.local.toml (plan 2026-10-30-script-local-config, D2).
type commentedTemplateProbe struct {
	Standard struct {
		MaxPipelineCycles *int    `toml:"max_pipeline_cycles"`
		BaseBranch        *string `toml:"base_branch"`
	} `toml:"standard"`
	Reviewer struct {
		ClaudeModel  *string `toml:"claude_model"`
		ClaudeEffort *string `toml:"claude_effort"`
		CodexModel   *string `toml:"codex_model"`
		CodexEffort  *string `toml:"codex_effort"`
	} `toml:"reviewer"`
	PlanVisual struct {
		Browser *string `toml:"browser"`
		Opener  *string `toml:"opener"`
	} `toml:"plan_visual"`
	Verify struct {
		Base                 *string `toml:"base"`
		SkipBranchSecretScan *bool   `toml:"skip_branch_secret_scan"`
	} `toml:"verify"`
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
	s, r, p, v := probe.Standard, probe.Reviewer, probe.PlanVisual, probe.Verify
	return slices.Concat(
		commentedDrift("standard.max_pipeline_cycles", s.MaxPipelineCycles, comments, def.Standard.MaxPipelineCycles),
		commentedDrift("standard.base_branch", s.BaseBranch, comments, def.Standard.BaseBranch),
		commentedDrift("reviewer.claude_model", r.ClaudeModel, comments, def.Reviewer.ClaudeModel),
		commentedDrift("reviewer.claude_effort", r.ClaudeEffort, comments, def.Reviewer.ClaudeEffort),
		commentedDrift("reviewer.codex_model", r.CodexModel, comments, def.Reviewer.CodexModel),
		commentedDrift("reviewer.codex_effort", r.CodexEffort, comments, def.Reviewer.CodexEffort),
		commentedDrift("plan_visual.browser", p.Browser, comments, def.PlanVisual.Browser),
		commentedDrift("plan_visual.opener", p.Opener, comments, def.PlanVisual.Opener),
		commentedDrift("verify.base", v.Base, comments, def.Verify.Base),
		commentedDrift("verify.skip_branch_secret_scan", v.SkipBranchSecretScan, comments, def.Verify.SkipBranchSecretScan),
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

// retiredSettingEnvs are the environment variables the standard flow read
// before [standard].max_pipeline_cycles and [reviewer]. ralph no longer
// reads them, so no skill may expand one.
var retiredSettingEnvs = []string{
	"RALPH_STANDARD_MAX_PIPELINE_CYCLES",
	"RALPH_CLAUDE_REVIEWER_MODEL",
	"RALPH_CODEX_REVIEWER_MODEL",
	"RALPH_CODEX_REASONING_EFFORT",
}

// skillSettingKey matches a key a skill body passes to ralph_setting, such
// as `ralph_setting reviewer.codex_model`. A key has at least one dot, so
// prose such as "ralph_setting prints" does not match.
var skillSettingKey = regexp.MustCompile(`ralph_setting ([a-z_]+(?:\.[a-z_]+)+)`)

// skillSettingUses holds, per skill, the keys its body must pass to
// ralph_setting, the number of claude reviewer launches it holds (lines
// with `claude -p --model`: /plan's step 11.c, and /cross-review's step 4
// and its CLI execution modes row), and the sentences that state the cap
// rule of plan 2026-10-30-standard-flow-config's D8: the cap is the larger
// of [standard].max_pipeline_cycles and a positive integer `cap` in
// cycle-count.json, which /cross-review reads with a jq that prints nothing
// for any other `cap`, the cap-reached Option 1 writes `cap` without
// incrementing `cycle` and stops when the printed file shows no new `cap`
// (self-review L4), and /implement keeps `cap` when it keeps the file.
var skillSettingUses = map[string]struct {
	keys           []string
	claudeLaunches int
	capRule        []string
}{
	"plan": {
		keys:           []string{"reviewer.codex_model", "reviewer.codex_effort", "reviewer.claude_model", "reviewer.claude_effort"},
		claudeLaunches: 1,
	},
	"cross-review": {
		keys:           []string{"standard.max_pipeline_cycles", "reviewer.codex_model", "reviewer.codex_effort", "reviewer.claude_model", "reviewer.claude_effort"},
		claudeLaunches: 2,
		capRule: []string{
			"the cap is the larger of the configured value and that `cap`",
			"Otherwise (no `cap`, or a `cap` that is not a positive integer) the cap is the configured value.",
			`jq -r '.cap | select(type == "number" and . >= 1 and . == floor)' "$f"`,
			"Do **NOT** increment `cycle`.",
			`jq -c '.cap = .cycle + 1' "$f" > "$f.tmp" && mv "$f.tmp" "$f"`,
			"If the printed line does not show the new `cap`",
			"the cap was not raised: do not re-run; tell the user what the command printed and stop.",
		},
	},
	"implement": {
		capRule: []string{"Keep the file as it is, its `cap` included when it has one"},
	},
}

// checkSkillSettings checks the body text of one copy of skill, named rel
// in failures: it expands none of retiredSettingEnvs; it passes each key of
// skillSettingUses[skill] to ralph_setting, and every key it passes is in
// table, the helper's table of defaults; each claude reviewer launch
// passes --model "$model" --effort "$effort", with both variables set by
// ralph_setting earlier on its line or on the line above; and it holds
// each sentence of the cap rule.
func checkSkillSettings(t *testing.T, rel, skill, text string, table map[string]string) {
	t.Helper()
	use := skillSettingUses[skill]
	for _, env := range retiredSettingEnvs {
		if strings.Contains(text, "${"+env) {
			t.Errorf("%s expands ${%s, which ralph no longer reads; read the setting with ralph_setting", rel, env)
		}
	}
	for _, key := range use.keys {
		if !strings.Contains(text, "ralph_setting "+key) {
			t.Errorf("%s does not read %s with `ralph_setting %s`", rel, key, key)
		}
	}
	for _, m := range skillSettingKey.FindAllStringSubmatch(text, -1) {
		if _, ok := table[m[1]]; !ok {
			t.Errorf("%s passes %s to ralph_setting, a key that is not in the table of defaults of scripts/ralph-config.sh", rel, m[1])
		}
	}
	lines := strings.Split(text, "\n")
	launches := 0
	for i, line := range lines {
		at := strings.Index(line, "claude -p --model")
		if at < 0 {
			continue
		}
		launches++
		if !strings.Contains(line[at:], `claude -p --model "$model" --effort "$effort" `) {
			t.Errorf("%s:%d: the claude reviewer launch does not pass --model \"$model\" --effort \"$effort\": %s", rel, i+1, line)
		}
		before := line[:at]
		if i > 0 {
			before = lines[i-1] + "\n" + before
		}
		for _, set := range []string{"model=$(ralph_setting reviewer.claude_model)", "effort=$(ralph_setting reviewer.claude_effort)"} {
			if !strings.Contains(before, set) {
				t.Errorf("%s:%d: the claude reviewer launch is not preceded by %s on its line or the line above", rel, i+1, set)
			}
		}
	}
	if launches != use.claudeLaunches {
		t.Errorf("%s holds %d claude reviewer launches (`claude -p --model`), want %d", rel, launches, use.claudeLaunches)
	}
	for _, sentence := range use.capRule {
		if !strings.Contains(text, sentence) {
			t.Errorf("%s does not state the cap rule %q", rel, sentence)
		}
	}
}

// TestCommentedTemplateDrift_EachKeyIsComparedOnBothSides runs
// checkEachKeyIsComparedOnBothSides on commentedTemplateDrift, editing the
// `# key = value` comments: claude_effort and codex_effort, both "xhigh" by
// default, swapped on either side fail here, and so do any two of
// base_branch, browser, opener, and base, all "" by default. Each key the
// template sets instead of commenting it, here by uncommenting its line, is a
// drift line naming the key, so a template that sets a key fails even when
// the value is Default()'s.
func TestCommentedTemplateDrift_EachKeyIsComparedOnBothSides(t *testing.T) {
	defaults := Default()
	rows := []templateKeyRow{
		{"standard.max_pipeline_cycles", "987", func(c *Config) { c.Standard.MaxPipelineCycles++ }},
		{"standard.base_branch", `"changed"`, func(c *Config) { c.Standard.BaseBranch += "-changed" }},
		{"reviewer.claude_model", `"changed"`, func(c *Config) { c.Reviewer.ClaudeModel += "-changed" }},
		{"reviewer.claude_effort", `"changed"`, func(c *Config) { c.Reviewer.ClaudeEffort += "-changed" }},
		{"reviewer.codex_model", `"changed"`, func(c *Config) { c.Reviewer.CodexModel += "-changed" }},
		{"reviewer.codex_effort", `"changed"`, func(c *Config) { c.Reviewer.CodexEffort += "-changed" }},
		{"plan_visual.browser", `"changed"`, func(c *Config) { c.PlanVisual.Browser += "-changed" }},
		{"plan_visual.opener", `"changed"`, func(c *Config) { c.PlanVisual.Opener += "-changed" }},
		{"verify.base", `"changed"`, func(c *Config) { c.Verify.Base += "-changed" }},
		{"verify.skip_branch_secret_scan", strconv.FormatBool(!defaults.Verify.SkipBranchSecretScan), func(c *Config) {
			c.Verify.SkipBranchSecretScan = !c.Verify.SkipBranchSecretScan
		}},
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

// TestTemplateDrift_EveryStandardReviewerPlanVisualAndVerifyKeyIsChecked pins
// that each key of [standard], [reviewer], [plan_visual], and [verify] is
// checked against the template, either set (standardTemplateProbe) or written
// as a comment (commentedTemplateProbe), so a key added to StandardConfig,
// ReviewerConfig, PlanVisualConfig, or VerifyConfig needs a probe field.
func TestTemplateDrift_EveryStandardReviewerPlanVisualAndVerifyKeyIsChecked(t *testing.T) {
	probed := slices.Concat(
		tomlKeys(reflect.TypeFor[standardTemplateProbe](), ""),
		tomlKeys(reflect.TypeFor[commentedTemplateProbe](), ""),
	)
	want := slices.Concat(
		tomlKeys(reflect.TypeFor[StandardConfig](), "standard"),
		tomlKeys(reflect.TypeFor[ReviewerConfig](), "reviewer"),
		tomlKeys(reflect.TypeFor[PlanVisualConfig](), "plan_visual"),
		tomlKeys(reflect.TypeFor[VerifyConfig](), "verify"),
	)
	if got, want := slices.Sorted(slices.Values(probed)), slices.Sorted(slices.Values(want)); !slices.Equal(got, want) {
		t.Errorf("the template drift probes check %q,\nwant the keys of [standard], [reviewer], [plan_visual], and [verify] %q", got, want)
	}
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
