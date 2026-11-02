package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

// The answers to ralph init's CLI question (initSettings.CLI). claude and
// codex are also the driver names of [org].driver_pool and model_pool.
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
	// also writes [org].driver_pool with that CLI alone and comments out the
	// other driver's entries of [org].model_pool, and for codex alone
	// [org.watchdog].watcher_enabled = false (the watcher runs claude).
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
	// initTOMLModelPoolOpen matches the first line of a multi-line
	// model_pool array.
	initTOMLModelPoolOpen = regexp.MustCompile(`^\s*model_pool\s*=\s*\[\s*(#.*)?$`)
	// initTOMLPoolEntryDriver matches an entry line of the model_pool array
	// (its indent already trimmed) and captures its driver.
	initTOMLPoolEntryDriver = regexp.MustCompile(`^\{.*\bdriver\s*=\s*"([^"]*)"`)
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
// CLI it comments out the other driver's lines of [org].model_pool instead
// of removing them.
//
// A key that is not on exactly one line of its table, a value that is not on
// one line, with one CLI a missing model_pool array (or one with no closing
// ] or no entry of that CLI), and an unknown CLI are errors naming what is
// wrong, and then nothing is returned. It does not touch the filesystem.
func applyInitSettings(template []byte, s initSettings) ([]byte, error) {
	settings, keep, drop, err := initTOMLSettings(s)
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
	if keep != "" {
		if err := initTOMLCommentModelPool(lines, tables, keep, drop); err != nil {
			return nil, err
		}
	}
	return []byte(strings.Join(lines, "\n")), nil
}

// initTOMLSettings lists the keys and values s writes. With one CLI, keep is
// that CLI's driver and drop the other one, whose model_pool entries are
// commented out; with both, keep and drop are "".
func initTOMLSettings(s initSettings) (settings []initTOMLSetting, keep, drop string, err error) {
	switch s.CLI {
	case initCLIClaude:
		keep, drop = initCLIClaude, initCLICodex
	case initCLICodex:
		keep, drop = initCLICodex, initCLIClaude
	case initCLIBoth:
	default:
		return nil, "", "", fmt.Errorf("unknown CLI %q (want %s, %s, or %s)", s.CLI, initCLIClaude, initCLICodex, initCLIBoth)
	}
	both := s.CLI == initCLIBoth
	regate := config.RegateChecksLocal
	if s.CI {
		regate = config.RegateChecksCI
	}
	settings = []initTOMLSetting{
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
		settings = append(settings, initTOMLSetting{"[org]", "driver_pool", "[" + strconv.Quote(keep) + "]"})
	}
	if s.CLI == initCLICodex {
		settings = append(settings, initTOMLSetting{"[org.watchdog]", "watcher_enabled", "false"})
	}
	return settings, keep, drop, nil
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

// initTOMLCommentModelPool comments out each entry of the driver drop in
// [org]'s multi-line model_pool array by inserting "# " after the line's
// indent, so the line count stays the same and a person who adds that CLI
// later can uncomment it. The array must keep an entry of the driver keep,
// since an empty model_pool fails the load.
func initTOMLCommentModelPool(lines, tables []string, keep, drop string) error {
	open := -1
	for i, line := range lines {
		if tables[i] != "[org]" || !initTOMLModelPoolOpen.MatchString(strings.TrimSuffix(line, "\r")) {
			continue
		}
		if open >= 0 {
			return errors.New("template ralph.toml sets [org].model_pool on more than one line")
		}
		open = i
	}
	if open < 0 {
		return errors.New("template ralph.toml has no [org].model_pool = [ line")
	}
	kept := 0
	for i := open + 1; i < len(lines) && tables[i] == "[org]"; i++ {
		body := strings.TrimSuffix(lines[i], "\r")
		entry := strings.TrimLeft(body, " \t")
		if strings.HasPrefix(entry, "]") {
			if kept == 0 {
				return fmt.Errorf("template ralph.toml: [org].model_pool has no %s entry", keep)
			}
			return nil
		}
		m := initTOMLPoolEntryDriver.FindStringSubmatch(entry)
		switch {
		case m == nil:
		case m[1] == drop:
			indent := len(body) - len(entry)
			lines[i] = lines[i][:indent] + "# " + lines[i][indent:]
		case m[1] == keep:
			kept++
		}
	}
	return errors.New("template ralph.toml: [org].model_pool has no closing ]")
}
