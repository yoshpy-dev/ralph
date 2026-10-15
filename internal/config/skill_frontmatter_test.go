package config

// TestSkillFrontmatter_PlainValuesReadAsTheySay keeps two shapes that YAML
// rejects or cuts out of every unquoted top-level value of every SKILL.md
// frontmatter (plan docs/plans/active/2026-10-15-org-director-start.md, D15
// and AC16). The `/director` skill's description once held "ways: the" in an
// unquoted value: YAML reads a colon and a space there as a mapping and
// rejects the whole frontmatter, so a loader that reads it as YAML loses the
// skill.
// scripts/check-skill-sync.sh compares the four copies of each skill but does
// not read the frontmatter as YAML and strips the quotes of its values, so it
// did not catch that. go.mod has no YAML library, so this test looks for the
// two shapes that break or cut a plain value instead of parsing YAML: a
// colon followed by whitespace (or ending the value), and whitespace
// followed by #, where YAML starts a comment and silently drops the rest.
// Other YAML errors pass it.
//
// It walks the skill roots (skillRoots) rather than asking git, so it runs
// the same way in CI. It skips only when .claude/skills itself is absent (a
// repo that vendors internal/config without the meta-repo tree, as
// TestDefaultsLockStep's SKILL.md sub-test does); past that gate a root with
// no SKILL.md, or a SKILL.md with no frontmatter, is an error.
//
// If it fails, quote the value named in the failure (or write it as a `>`
// block), or reword it, in the .claude copy of the skill (root and
// templates/base), then run scripts/sync-skills.sh for the .agents copies
// (root, then with CLAUDE_ROOT=templates/base/.claude/skills
// CODEX_ROOT=templates/base/.agents/skills).

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

// skillRoots are the directories whose SKILL.md files are skills: the Claude
// and Codex copies of the meta-repo and of the template ralph init scaffolds.
// `git ls-files '*SKILL.md'` lists no SKILL.md anywhere else (54 files on
// 2026-10-16); add a root here when one appears.
var skillRoots = []string{
	filepath.Join(".claude", "skills"),
	filepath.Join(".agents", "skills"),
	filepath.Join("templates", "base", ".claude", "skills"),
	filepath.Join("templates", "base", ".agents", "skills"),
}

// frontmatterFinding is a top-level frontmatter value that YAML does not
// read as written: the line and the column (1-based, in characters) of the
// colon or the # in the file, the key, and what is wrong.
type frontmatterFinding struct {
	Line, Column int
	Key, What    string
}

// What a frontmatterFinding says, one per shape plainFrontmatterFindings
// looks for.
const (
	whatColonSpace = `holds ": ", which YAML reads as a mapping and rejects`
	whatColonTab   = "holds a colon followed by a tab, which YAML reads as a mapping and rejects"
	whatColonEnd   = "ends with a colon, which YAML reads as a mapping and rejects"
	whatSpaceHash  = `holds " #", where YAML starts a comment and drops the rest of the value`
	whatTabHash    = "holds a tab followed by #, where YAML starts a comment and drops the rest of the value"
)

// plainValueShapes are the shapes plainFrontmatterFindings looks for inside
// a plain value; at is the offset of the character a finding points at (the
// colon, or the #) from the start of the match.
var plainValueShapes = []struct {
	needle string
	at     int
	what   string
}{
	{": ", 0, whatColonSpace},
	{":\t", 0, whatColonTab},
	{" #", 1, whatSpaceHash},
	{"\t#", 1, whatTabHash},
}

// frontmatterKeyLine is a top-level `key: value` line: a key at column 1,
// then a colon that ends the line or is followed by a space or a tab.
var frontmatterKeyLine = regexp.MustCompile(`^([A-Za-z0-9_.-]+):([ \t].*)?$`)

// plainFrontmatterFindings returns the findings of text's frontmatter, the
// lines between its first line `---` and the next `---`. It checks every
// top-level `key: value` line whose value is a plain YAML scalar, that is,
// not empty and not starting with ", ', >, or |, and returns, per line, the
// first of: a colon followed by a space or a tab, a colon that ends the value,
// and a space or a tab followed by #. Indented lines (nested values, the
// lines of a block scalar) and the body after the frontmatter are not
// checked. ok is false when text does not start with a closed frontmatter.
func plainFrontmatterFindings(text string) (findings []frontmatterFinding, ok bool) {
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = strings.TrimSuffix(lines[i], "\r")
	}
	isFence := func(line string) bool { return strings.TrimRight(line, " \t") == "---" }
	if len(lines) == 0 || !isFence(lines[0]) {
		return nil, false
	}
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if isFence(line) {
			return findings, true
		}
		m := frontmatterKeyLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		value := strings.TrimLeft(m[2], " \t")
		if value == "" || strings.ContainsAny(value[:1], `"'>|`) {
			continue
		}
		valueStart := len(line) - len(value)
		at, what := -1, ""
		for _, shape := range plainValueShapes {
			if j := strings.Index(value, shape.needle); j >= 0 && (at < 0 || j+shape.at < at) {
				at, what = j+shape.at, shape.what
			}
		}
		if trimmed := strings.TrimRight(value, " \t"); strings.HasSuffix(trimmed, ":") && (at < 0 || len(trimmed)-1 < at) {
			at, what = len(trimmed)-1, whatColonEnd
		}
		if at >= 0 {
			column := utf8.RuneCountInString(line[:valueStart+at]) + 1
			findings = append(findings, frontmatterFinding{Line: i + 1, Column: column, Key: m[1], What: what})
		}
	}
	return nil, false
}

func TestSkillFrontmatter_PlainValuesReadAsTheySay(t *testing.T) {
	root := repoRoot(t)
	if _, err := os.Stat(filepath.Join(root, ".claude", "skills")); err != nil {
		t.Skipf(".claude/skills not found (%v) — skipping the SKILL.md frontmatter check (vendored repo?)", err)
	}
	checked := 0
	for _, rel := range skillRoots {
		var files []string
		err := filepath.WalkDir(filepath.Join(root, rel), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && d.Name() == "SKILL.md" {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			t.Errorf("walk %s: %v", filepath.ToSlash(rel), err)
			continue
		}
		if len(files) == 0 {
			t.Errorf("%s holds no SKILL.md: a skills root that moved would leave this check silent, so update skillRoots", filepath.ToSlash(rel))
			continue
		}
		for _, path := range files {
			name, err := filepath.Rel(root, path)
			if err != nil {
				name = path
			}
			name = filepath.ToSlash(name)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("%s: %v", name, err)
				continue
			}
			checked++
			findings, ok := plainFrontmatterFindings(string(data))
			if !ok {
				t.Errorf("%s: no frontmatter: the file does not start with a --- line closed by another", name)
				continue
			}
			for _, f := range findings {
				t.Errorf("%s:%d:%d: the value of %s is a plain YAML scalar that %s; quote it (or write it as a > block) or reword it -- see this test's doc comment",
					name, f.Line, f.Column, f.Key, f.What)
			}
		}
	}
	t.Logf("checked the frontmatter of %d SKILL.md files under %s", checked, strings.Join(skillRoots, ", "))
}

// TestPlainFrontmatterFindings pins what TestSkillFrontmatter_PlainValuesReadAsTheySay
// counts as a finding, starting with the description the /director skill
// had before the fix (plan D15), and what it leaves alone. Each case was
// checked against Ruby's YAML.safe_load (Psych): every finding's line is
// rejected (at the column the finding names), or for #, read with the rest
// of the value dropped, and every line left alone reads as written, except
// in the nested case, which YAML rejects too: it shows that indented lines
// are a limit of this check.
func TestPlainFrontmatterFindings(t *testing.T) {
	cases := []struct {
		name   string
		text   string
		want   []frontmatterFinding
		wantOK bool
	}{
		{"the /director description before the fix",
			"---\nname: director\ndescription: Director's operating manual for the org runtime. The director is the one session the human talks to " +
				"for every feature org, in one of two ways: the current interactive session, promoted with this skill.\n---\nbody\n",
			[]frontmatterFinding{{3, 155, "description", whatColonSpace}}, true},
		{"the /director description after the fix",
			"---\nname: director\ndescription: Director's operating manual for the org runtime. The director is the one session the human talks to " +
				"for every feature org, in one of two ways, either the current interactive session, promoted with this skill.\n---\n",
			nil, true},
		{"a colon followed by a tab", "---\na: x:\ty\n---\n", []frontmatterFinding{{2, 5, "a", whatColonTab}}, true},
		{"a colon that ends the value", "---\na: x:\n---\n", []frontmatterFinding{{2, 5, "a", whatColonEnd}}, true},
		{"a space and a #", "---\na: use C #2\n---\n", []frontmatterFinding{{2, 10, "a", whatSpaceHash}}, true},
		{"a tab and a #", "---\na: x\t#y\n---\n", []frontmatterFinding{{2, 6, "a", whatTabHash}}, true},
		{"the first of two shapes on a line", "---\na: x #y: z\n---\n", []frontmatterFinding{{2, 6, "a", whatSpaceHash}}, true},
		{"one finding per line, every line", "---\na: x: y\nb: ok\nc: z #w\n---\n",
			[]frontmatterFinding{{2, 5, "a", whatColonSpace}, {4, 6, "c", whatSpaceHash}}, true},
		{"a column counted in characters", "---\na: 日本語: x\n---\n", []frontmatterFinding{{2, 7, "a", whatColonSpace}}, true},
		{"CRLF line ends", "---\r\na: x: y\r\n---\r\n", []frontmatterFinding{{2, 5, "a", whatColonSpace}}, true},
		{"quoted values", "---\na: \"x: y #z\"\nb: 'x: y #z'\n---\n", nil, true},
		{"block scalars and their lines", "---\na: >\n  x: y\nb: |\n  z #w\n---\n", nil, true},
		{"a colon or a # with no whitespace", "---\na: http://x and C# and x:y\n---\n", nil, true},
		{"an empty value, and a nested value it does not check", "---\nmetadata:\n  type: a: b\n---\n", nil, true},
		{"a comment line", "---\n# a: b: c\na: b\n---\n", nil, true},
		{"the body", "---\na: b\n---\nx: y: z\n", nil, true},
		{"no frontmatter", "# Title\n\nx: y: z\n", nil, false},
		{"a frontmatter never closed", "---\na: x: y\n", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := plainFrontmatterFindings(tc.text)
			if ok != tc.wantOK || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("plainFrontmatterFindings = %+v, %v; want %+v, %v", got, ok, tc.want, tc.wantOK)
			}
		})
	}
}
