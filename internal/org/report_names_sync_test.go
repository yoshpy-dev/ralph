package org

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// reportNamesRepoRoot resolves the repository root from this source file's
// location: internal/org/report_names_sync_test.go -> ../../, as
// sendDefaultsRepoRoot does.
func reportNamesRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location via runtime.Caller")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..")
}

// readRepoFile reads rel (a slash-separated path) under root. ok is false,
// with a nil err, when the file does not exist; any other read error is
// returned.
func readRepoFile(root, rel string) (content string, ok bool, err error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return string(data), true, nil
}

// readRepoFileOrSkip reads rel under the repo root, skips the test when the
// file is absent (a project that has internal/org without this repo's
// scripts and docs), and fails it on any other read error.
func readRepoFileOrSkip(t *testing.T, rel string) string {
	t.Helper()
	text, ok, err := readRepoFile(reportNamesRepoRoot(t), rel)
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	if !ok {
		t.Skipf("%s not found -- skipping the report name check for it", rel)
	}
	return text
}

// hasLine reports whether one line of text, with its surrounding white space
// trimmed, equals want.
func hasLine(text, want string) bool {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == want {
			return true
		}
	}
	return false
}

// TestReportNames_DocsMatchConstants pins AC4 of plan
// 2026-10-20-org-report-names: the names of the two reports a feature org
// writes into its worktree, the reviewer seat's report (reviewerReportPrefix)
// and the org report of `ralph org report` (defaultReportDir and
// orgReportFilePrefix in report.go), are spelled the same in the hand-written
// places listed below, the ones a feature org's gate and the naming list rely
// on. A feature org's gate runs scripts/check-sync.sh, which reports either
// file as ROOT_ONLY unless its prefix is excluded, and nothing else catches a
// rename on one side only. Other prose that names the org report is not
// checked: the help of `ralph org report` (internal/cli/org.go), the prefix
// check in internal/cli/org_test.go, and the /org skill's `inbox resolve`
// example and closing checklist.
//
// If this test fails, make these agree with reviewerReportPrefix and with
// defaultReportDir + "/" + orgReportFilePrefix:
//   - internal/org/prompts/reviewer.md, the first item of レポート契約
//   - scripts/check-sync.sh, the ROOT_ONLY_EXCLUSIONS array
//   - docs/reports/README.md, the Suggested naming list
//     (templates/base/docs/reports/README.md is kept identical by
//     scripts/check-sync.sh)
//   - .claude/skills/org/SKILL.md, the `report` row (its mirrors are kept
//     identical by scripts/check-skill-sync.sh)
//
// A repo file that is absent skips its subtest; the reviewer template is
// read from promptFS, so it is always checked.
func TestReportNames_DocsMatchConstants(t *testing.T) {
	orgReportPrefix := defaultReportDir + "/" + orgReportFilePrefix

	t.Run("reviewer template", func(t *testing.T) {
		data, err := promptFS.ReadFile("prompts/reviewer.md")
		if err != nil {
			t.Fatalf("read the embedded reviewer template: %v", err)
		}
		contract, found := markdownSection(string(data), "## レポート契約")
		if !found {
			t.Fatalf("expected the reviewer template to contain a '## レポート契約' section, got:\n%s", data)
		}
		want := "`" + reviewerReportPrefix + "<YYYY-MM-DD>-{{ORG_ID}}-{{SEAT_ID}}.md`"
		if !strings.Contains(contract, want) {
			t.Errorf("expected the reviewer template's report contract to name the report %s, got section:\n%s", want, contract)
		}
	})

	t.Run("check-sync.sh exclusions", func(t *testing.T) {
		const rel = "scripts/check-sync.sh"
		text := readRepoFileOrSkip(t, rel)
		_, rest, found := strings.Cut(text, "\nROOT_ONLY_EXCLUSIONS=(\n")
		if !found {
			t.Fatalf("%s: expected a line ROOT_ONLY_EXCLUSIONS=(, found none", rel)
		}
		block, _, closed := strings.Cut(rest, "\n)\n")
		if !closed {
			t.Fatalf("%s: expected the ROOT_ONLY_EXCLUSIONS array to end with a line ), found none", rel)
		}
		for _, want := range []string{`"` + reviewerReportPrefix + `"`, `"` + orgReportPrefix + `"`} {
			if !hasLine(block, want) {
				t.Errorf("%s: expected the ROOT_ONLY_EXCLUSIONS array to have the line %s", rel, want)
			}
		}
	})

	t.Run("reports README naming list", func(t *testing.T) {
		const rel = "docs/reports/README.md"
		text := readRepoFileOrSkip(t, rel)
		_, rest, found := strings.Cut(text, "\nSuggested naming:\n")
		if !found {
			t.Fatalf("%s: expected a line Suggested naming:, found none", rel)
		}
		list, _, _ := strings.Cut(rest, "\n\n")
		for _, want := range []string{
			"- `" + reviewerReportFilePrefix + "YYYY-MM-DD-<org_id>-<seat_id>.md`",
			"- `" + orgReportFilePrefix + "<org_id>-YYYY-MM-DD.md`",
		} {
			if !hasLine(list, want) {
				t.Errorf("%s: expected the Suggested naming list to have the bullet %s, got list:\n%s", rel, want, list)
			}
		}
	})

	t.Run("org skill report row", func(t *testing.T) {
		const rel = ".claude/skills/org/SKILL.md"
		text := readRepoFileOrSkip(t, rel)
		want := "`" + orgReportPrefix + "<org_id>-<date>.md`"
		for _, line := range strings.Split(text, "\n") {
			if strings.HasPrefix(line, "| `report` |") {
				if !strings.Contains(line, want) {
					t.Errorf("%s: expected the report row to name %s, got row:\n%s", rel, want, line)
				}
				return
			}
		}
		t.Errorf("%s: expected a table row starting with | `report` |, found none", rel)
	})
}

func TestReadRepoFile_MissingFileIsNotAnError(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "a.md"), []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, rel   string
		wantContent string
		wantOK      bool
		wantErr     bool
	}{
		{"a present file is read", "docs/a.md", "text", true, false},
		{"a missing file is not ok and not an error", "scripts/check-sync.sh", "", false, false},
		{"a directory is an error", "docs", "", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content, ok, err := readRepoFile(root, tc.rel)
			if content != tc.wantContent || ok != tc.wantOK || (err != nil) != tc.wantErr {
				t.Errorf("readRepoFile(root, %q) = (%q, %v, %v), want (%q, %v, error=%v)", tc.rel, content, ok, err, tc.wantContent, tc.wantOK, tc.wantErr)
			}
		})
	}
}

// codeBlockAfterLine returns the contents of the first fenced code block (from
// a line starting with "```" to the next line that is exactly "```", both
// fences left out) after the first line of text that starts with prefix.
// found is false when no line starts with prefix, or when no code block opens
// and closes after that line. Indented fences are not recognized.
func codeBlockAfterLine(text, prefix string) (block string, found bool) {
	lines := strings.Split(text, "\n")
	lead := slices.IndexFunc(lines, func(line string) bool { return strings.HasPrefix(line, prefix) })
	if lead < 0 {
		return "", false
	}
	open := -1
	for i := lead + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "```") {
			open = i
			break
		}
	}
	if open < 0 {
		return "", false
	}
	for i := open + 1; i < len(lines); i++ {
		if lines[i] == "```" {
			return strings.Join(lines[open+1:i], "\n"), true
		}
	}
	return "", false
}

func TestCodeBlockAfterLine_TakesTheFirstBlockAfterTheLead(t *testing.T) {
	cases := []struct {
		name, text, wantBlock string
		wantFound             bool
	}{
		{"the block after the lead line", "intro\nLEAD:\n\n```\na\nb\n```\ntail", "a\nb", true},
		{"a block before the lead line is not taken", "```\nbefore\n```\nLEAD:\n```text\nafter\n```\n```\nlater\n```", "after", true},
		{"an absent lead", "intro\n```\na\n```", "", false},
		{"a lead with no block after it", "```\na\n```\nLEAD:\ntail", "", false},
		{"a block that never closes", "LEAD:\n```\na\n", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			block, found := codeBlockAfterLine(tc.text, "LEAD")
			if block != tc.wantBlock || found != tc.wantFound {
				t.Errorf("codeBlockAfterLine(%q, \"LEAD\") = (%q, %v), want (%q, %v)", tc.text, block, found, tc.wantBlock, tc.wantFound)
			}
		})
	}
}

// orgSkillResultExampleLead starts the line of the /org skill's typed
// protocol section that introduces its RESULT example.
const orgSkillResultExampleLead = "RESULT の例"

// TestOrgSkill_ResultExampleFollowsTheReviewerTemplate pins AC3 of plan
// 2026-10-23-trial-docs-wrapup: the RESULT example of the /org skill's typed
// protocol section is a reviewer seat's report written the way the reviewer
// template (prompts/reviewer.md, its RESULT example and レポート契約) writes
// one. The example passes protocol.Parse and protocol.Validate as a RESULT,
// carries the header lines GATE: pass, HEAD: and BASE:, and has the EVIDENCE
// line "EVIDENCE: " + reviewerReportPrefix + "<YYYY-MM-DD>-<org_id>-<seat_id>.md"
// exactly, in that placeholder form; an example with real names needs this
// test changed. The test does not read prompts/reviewer.md, and the header
// names are spelled here: it ties the example to the template through
// reviewerReportPrefix, which TestReportNames_DocsMatchConstants pins in the
// template's レポート契約.
// The example used to name docs/reports/self-review-foo.md, while the
// reviewer template says a self-review- name is read as the /self-review
// report and must not be used.
//
// If this test fails, make the first code block after the line starting with
// orgSkillResultExampleLead in the ## typed protocol section of
// .claude/skills/org/SKILL.md and templates/base/.claude/skills/org/SKILL.md
// agree with internal/org/prompts/reviewer.md again, and run
// scripts/sync-skills.sh for the two .agents copies (root, then with
// CLAUDE_ROOT=templates/base/.claude/skills
// CODEX_ROOT=templates/base/.agents/skills). scripts/check-skill-sync.sh keeps
// the four copies identical, so this test reads the root copy only.
func TestOrgSkill_ResultExampleFollowsTheReviewerTemplate(t *testing.T) {
	const rel = ".claude/skills/org/SKILL.md"
	text := readRepoFileOrSkip(t, rel)
	section, found := markdownSection(text, "## typed protocol")
	if !found {
		t.Fatalf("%s: expected a '## typed protocol' section, found none", rel)
	}
	msg, found := codeBlockAfterLine(section, orgSkillResultExampleLead)
	if !found {
		t.Fatalf("%s: expected the typed protocol section to show a code block after a line starting with %q, got section:\n%s", rel, orgSkillResultExampleLead, section)
	}

	m, err := protocol.Parse(msg)
	if err != nil {
		t.Fatalf("%s: expected the example RESULT to parse, got %v; message:\n%s", rel, err, msg)
	}
	if err := protocol.Validate(m, protocol.DefaultMaxBodyChars); err != nil || m.Type != protocol.TypeResult {
		t.Errorf("%s: expected the example to be a valid RESULT, got TYPE %q, err %v; message:\n%s", rel, m.Type, err, msg)
	}
	if got := protocol.HeaderValues(msg, "GATE"); !slices.Equal(got, []string{"pass"}) {
		t.Errorf("%s: expected the example RESULT to carry one header GATE: pass, got %q; message:\n%s", rel, got, msg)
	}
	for _, key := range []string{"HEAD", "BASE"} {
		if got := protocol.HeaderValues(msg, key); len(got) != 1 || got[0] == "" {
			t.Errorf("%s: expected the example RESULT to carry one non-empty %s: header, got %q; message:\n%s", rel, key, got, msg)
		}
	}
	want := "EVIDENCE: " + reviewerReportPrefix + "<YYYY-MM-DD>-<org_id>-<seat_id>.md"
	if !hasLine(m.Body, want) {
		t.Errorf("%s: expected the example RESULT's body to have the line %s, got body:\n%s", rel, want, m.Body)
	}
}
