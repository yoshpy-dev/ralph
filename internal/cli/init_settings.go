package cli

import (
	"cmp"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"charm.land/huh/v2"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

// The answers to ralph init's CLI question (initSettings.CLI). claude and
// codex are also the driver names of [org].driver_pool.
const (
	initCLIClaude = "claude"
	initCLICodex  = "codex"
	initCLIBoth   = "both"
)

// initSettings holds the answers to ralph init's settings questions, which
// applyInitSettings writes into the template ralph.toml.
type initSettings struct {
	// CLI is the AI coding CLI the project uses: initCLIClaude,
	// initCLICodex, or initCLIBoth. It writes [doctor].require_claude_cli
	// and require_codex_cli (true for each CLI it names). For one CLI it
	// also writes [org].driver_pool with that CLI alone, which picks the
	// built-in [org.role_models] set of that CLI (config.DefaultRoleModels),
	// and for codex alone [org.watchdog].watcher_enabled = false (the watcher
	// runs claude).
	CLI string
	// PlanReview writes [standard].plan_advisory and
	// [org.review].plan_advisory. Both are false with one CLI.
	PlanReview bool
	// ChangeReview writes [standard].cross_review and
	// [org.review].cross_review. Both are false with one CLI.
	ChangeReview bool
	// CI writes [org.gate].regate_checks: "ci" when true, "local" when false.
	CI bool
	// AutoMerge writes [org.director].auto_merge.
	AutoMerge bool
}

// initSettingsFile is the scaffolded file executeInit writes the answers
// into, a path of the base template FS.
const initSettingsFile = "ralph.toml"

// answeredInitSettingsTOML reads ralph.toml from baseFS and returns it with
// the answers in s written (applyInitSettings), and the hash of the
// template's own bytes, which the manifest records as ralph.toml's
// TemplateHash so that upgrade compares the next template with this one, not
// with the answers.
func answeredInitSettingsTOML(baseFS fs.FS, s initSettings) (answered []byte, templateHash string, err error) {
	template, err := fs.ReadFile(baseFS, initSettingsFile)
	if err != nil {
		return nil, "", err
	}
	answered, err = applyInitSettings(template, s)
	if err != nil {
		return nil, "", err
	}
	return answered, scaffold.HashBytes(template), nil
}

// initAnsweredFS is the base template FS executeInit renders when it has
// answers: ReadFile returns answered for ralph.toml, so scaffold.RenderFS,
// which reads each file with fs.ReadFile, writes the answers the first time
// it writes ralph.toml, and reads every other file from the embedded FS.
// Open passes through to the embedded FS, so fs.WalkDir walks the
// template's tree, and an Open of ralph.toml reads the template's bytes.
type initAnsweredFS struct {
	fs.FS
	answered []byte
}

// ReadFile returns a copy of answered for ralph.toml and the embedded FS's
// file for any other name.
func (f initAnsweredFS) ReadFile(name string) ([]byte, error) {
	if name == initSettingsFile {
		return slices.Clone(f.answered), nil
	}
	return fs.ReadFile(f.FS, name)
}

var (
	// initTOMLHeader matches a table header ([org.gate]) or an array-of-tables
	// header ([[name]]) at column 0. The closing ] of a multi-line array and
	// an indented line never match.
	initTOMLHeader = regexp.MustCompile(`^(\[[A-Za-z0-9_.]+\]|\[\[[A-Za-z0-9_.]+\]\])\s*$`)
	// initTOMLKeyLine matches an uncommented key = value line: the indent,
	// the key, the = with its spaces, and the rest of the line.
	initTOMLKeyLine = regexp.MustCompile(`^(\s*)([A-Za-z0-9_-]+)(\s*=\s*)(.*)$`)
	// initTOMLOneLineValue matches the values applyInitSettings replaces: a
	// boolean, a string with no escape, or an array closed on the same line,
	// with the spaces and the comment after it.
	initTOMLOneLineValue = regexp.MustCompile(`^(true|false|"[^"\\]*"|\[[^\[\]]*\])(\s*(#.*)?)$`)
)

// initTOMLSetting is one key applyInitSettings writes: the header of its
// table, the key, and its new value as TOML text.
type initTOMLSetting struct {
	table, key, value string
}

// applyInitSettings returns template, the bytes of templates/base/ralph.toml,
// with the keys the answers in s write (see initSettings) set to them. It
// replaces only the value of each key's uncommented key = value line in the
// key's own table, from the table's header to the next header, so
// plan_advisory and cross_review are set in [standard] and [org.review]
// each. Every other line, the line endings, and the final newline stay as
// they are, so the template's comments still describe the file. With one
// CLI it writes [org].driver_pool with that CLI alone and writes nothing in
// [org.role_models]: the roles take the built-in set that driver_pool picks.
//
// A key that is not on exactly one line of its table, a value that is not on
// one line, and an unknown CLI are errors naming what is wrong, and then
// nothing is returned. It does not touch the filesystem.
func applyInitSettings(template []byte, s initSettings) ([]byte, error) {
	settings, err := initSettingsToTOML(s)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(template), "\n")
	tables := initTOMLTables(lines)
	for _, set := range settings {
		if err := initTOMLSet(lines, tables, set); err != nil {
			return nil, err
		}
	}
	return []byte(strings.Join(lines, "\n")), nil
}

// initSettingsToTOML lists the keys and values s writes. With one CLI, the
// list includes [org].driver_pool with that CLI alone.
func initSettingsToTOML(s initSettings) ([]initTOMLSetting, error) {
	switch s.CLI {
	case initCLIClaude, initCLICodex, initCLIBoth:
	default:
		return nil, fmt.Errorf("unknown CLI %q (want %s, %s, or %s)", s.CLI, initCLIClaude, initCLICodex, initCLIBoth)
	}
	both := s.CLI == initCLIBoth
	regate := config.RegateChecksLocal
	if s.CI {
		regate = config.RegateChecksCI
	}
	settings := []initTOMLSetting{
		{"[doctor]", "require_claude_cli", strconv.FormatBool(s.CLI != initCLICodex)},
		{"[doctor]", "require_codex_cli", strconv.FormatBool(s.CLI != initCLIClaude)},
		{"[standard]", "plan_advisory", strconv.FormatBool(both && s.PlanReview)},
		{"[standard]", "cross_review", strconv.FormatBool(both && s.ChangeReview)},
		{"[org.review]", "plan_advisory", strconv.FormatBool(both && s.PlanReview)},
		{"[org.review]", "cross_review", strconv.FormatBool(both && s.ChangeReview)},
		{"[org.gate]", "regate_checks", strconv.Quote(regate)},
		{"[org.director]", "auto_merge", strconv.FormatBool(s.AutoMerge)},
	}
	if !both {
		settings = append(settings, initTOMLSetting{"[org]", "driver_pool", "[" + strconv.Quote(s.CLI) + "]"})
	}
	if s.CLI == initCLICodex {
		settings = append(settings, initTOMLSetting{"[org.watchdog]", "watcher_enabled", "false"})
	}
	return settings, nil
}

// initTOMLTables returns, for each line, the header of the table the line is
// in ("" before the first header). A header line is in its own table.
func initTOMLTables(lines []string) []string {
	tables := make([]string, len(lines))
	current := ""
	for i, line := range lines {
		if m := initTOMLHeader.FindStringSubmatch(strings.TrimSuffix(line, "\r")); m != nil {
			current = m[1]
		}
		tables[i] = current
	}
	return tables
}

// initTOMLSet replaces the value of set.key's line in the table set.table,
// keeping the indent, the key, the = with its spaces, the comment after the
// value, and the line ending.
func initTOMLSet(lines, tables []string, set initTOMLSetting) error {
	at := -1
	for i, line := range lines {
		if tables[i] != set.table {
			continue
		}
		m := initTOMLKeyLine.FindStringSubmatch(strings.TrimSuffix(line, "\r"))
		if m == nil || m[2] != set.key {
			continue
		}
		if at >= 0 {
			return fmt.Errorf("template ralph.toml sets %s.%s on more than one line", set.table, set.key)
		}
		at = i
	}
	if at < 0 {
		return fmt.Errorf("template ralph.toml has no %s.%s line", set.table, set.key)
	}
	body := strings.TrimSuffix(lines[at], "\r")
	eol := lines[at][len(body):]
	m := initTOMLKeyLine.FindStringSubmatch(body)
	v := initTOMLOneLineValue.FindStringSubmatch(m[4])
	if v == nil {
		return fmt.Errorf("template ralph.toml: the value of %s.%s is not on one line: %s", set.table, set.key, body)
	}
	lines[at] = m[1] + m[2] + m[3] + set.value + v[2] + eol
	return nil
}

// askInitSettings reports whether ralph init asks its settings questions for
// targetDir. Without force it does not when targetDir already has something
// named ralph.toml (a file, a symlink, or a directory): executeInit keeps that
// ralph.toml, so the answers would not be written. It then writes one line
// saying so to w.
func askInitSettings(targetDir string, force bool, w io.Writer) bool {
	if force {
		return true
	}
	if _, err := os.Lstat(filepath.Join(targetDir, initSettingsFile)); err != nil {
		return true
	}
	writef(w, "%s already exists, so ralph init does not ask about its settings (edit %s to change them; --force asks, and overwrites every file init writes).\n", initSettingsFile, initSettingsFile)
	return false
}

// The CLI question of the init form (newCLIQuestion). Each description line
// is short enough not to wrap in an 80-column terminal.
const (
	initCLIQuestionTitle       = "Which AI coding CLI will you use?"
	initCLIQuestionDescription = "Both: Claude Code and Codex can check each other's plans and changes."
	// initCLIRecommendedNote explains the "(Recommended)" label, so
	// cliQuestionDescription adds it only when an option carries the label.
	initCLIRecommendedNote = "(Recommended): found on this machine's PATH."
)

// initCLIOptions are the options of the CLI question, in the order it lists
// them.
var initCLIOptions = []struct{ label, value string }{
	{label: "Claude Code", value: initCLIClaude},
	{label: "Codex", value: initCLICodex},
	{label: "Both", value: initCLIBoth},
}

// recommendedCLI returns the answer to the CLI question that matches the
// commands lookPath (exec.LookPath outside tests) finds, which are named like
// the drivers: initCLIClaude or initCLICodex when it finds that one alone,
// initCLIBoth when it finds both, and "" when it finds neither.
func recommendedCLI(lookPath func(file string) (string, error)) string {
	_, claudeErr := lookPath(initCLIClaude)
	_, codexErr := lookPath(initCLICodex)
	switch {
	case claudeErr == nil && codexErr == nil:
		return initCLIBoth
	case claudeErr == nil:
		return initCLIClaude
	case codexErr == nil:
		return initCLICodex
	}
	return ""
}

// cliQuestionDescription returns the description of the CLI question for the
// answer recommendedCLI returned: the line on Both, and the line on
// "(Recommended)" only when recommended names an option, so that it never
// explains a label no option carries.
func cliQuestionDescription(recommended string) string {
	if recommended == "" {
		return initCLIQuestionDescription
	}
	return initCLIQuestionDescription + "\n" + initCLIRecommendedNote
}

// cliOptions returns the options of the CLI question, the one whose value is
// recommended labeled "<label> (Recommended)".
func cliOptions(recommended string) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(initCLIOptions))
	for _, o := range initCLIOptions {
		label := o.label
		if o.value == recommended {
			label += " (Recommended)"
		}
		options = append(options, huh.NewOption(label, o.value))
	}
	return options
}

// newCLIQuestion returns the CLI question of the init form. It sets *cli to
// the answer the question starts on, the CLI lookPath finds alone, or
// initCLIBoth when it finds both or neither, and writes the answer back to
// it.
func newCLIQuestion(lookPath func(file string) (string, error), cli *string) *huh.Select[string] {
	recommended := recommendedCLI(lookPath)
	*cli = cmp.Or(recommended, initCLIBoth)
	return huh.NewSelect[string]().
		Title(initCLIQuestionTitle).
		Description(cliQuestionDescription(recommended)).
		Options(cliOptions(recommended)...).
		Value(cli)
}

// initSettingsQuestion is a yes/no question of the settings form
// (newSettingsForm): its title and description, whether it is asked only
// when the project uses both CLIs, and the answer in initSettings it sets.
type initSettingsQuestion struct {
	title       string
	description string
	bothOnly    bool
	answer      func(s *initSettings) *bool
}

// initSettingsQuestions are the questions of the settings form, in the order
// it asks them. Each description line is short enough not to wrap in an
// 80-column terminal.
var initSettingsQuestions = []initSettingsQuestion{
	{
		title: "Have the other AI check each plan before work starts?",
		description: "Claude Code reads the plans Codex writes, Codex reads Claude Code's,\n" +
			"and each points out the risks it sees.",
		bothOnly: true,
		answer:   func(s *initSettings) *bool { return &s.PlanReview },
	},
	{
		title: "Have the other AI review each change before its pull request?",
		description: "Claude Code reviews the changes Codex makes, Codex reviews Claude Code's,\n" +
			"and each points out the problems it finds.",
		bothOnly: true,
		answer:   func(s *initSettings) *bool { return &s.ChangeReview },
	},
	{
		title: "Does CI run your tests on every pull request?",
		description: "Yes: when ralph brings a branch up to date with the main branch,\n" +
			"it lets CI run the tests.\n" +
			"No: ralph runs the tests on this machine.",
		answer: func(s *initSettings) *bool { return &s.CI },
	},
	{
		title: "Merge pull requests whose checks pass without asking you?",
		description: "Yes: ralph merges each pull request once its checks pass.\n" +
			"No: ralph tells you which pull requests are ready, and you merge them.",
		answer: func(s *initSettings) *bool { return &s.AutoMerge },
	},
}

// newInitSettings returns the answers the settings form starts on for cli,
// the template ralph.toml's own values: no plan check, no change review, the
// tests run on this machine, and merging without asking.
func newInitSettings(cli string) initSettings {
	return initSettings{CLI: cli, AutoMerge: true}
}

// newSettingsFields returns the questions of the settings form for s.CLI,
// the answer to the CLI question: every question of initSettingsQuestions
// with both CLIs, and those that are not bothOnly with one. Each starts on
// its answer in s and writes the answer back to s.
func newSettingsFields(s *initSettings) []huh.Field {
	var fields []huh.Field
	for _, q := range initSettingsQuestions {
		if q.bothOnly && s.CLI != initCLIBoth {
			continue
		}
		fields = append(fields, huh.NewConfirm().
			Title(q.title).
			Description(q.description).
			Value(q.answer(s)))
	}
	return fields
}

// newSettingsForm returns the init form's second part, the questions of
// newSettingsFields. It is a form of its own, run after the first one, not a
// group of the first form hidden by the CLI answer: in accessible mode
// (TERM=dumb) huh asks every field of every group, hidden or not.
func newSettingsForm(s *initSettings) *huh.Form {
	return huh.NewForm(huh.NewGroup(newSettingsFields(s)...))
}
