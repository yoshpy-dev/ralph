package org

import (
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// directorSkillQueueText is a fragment of the /director skill's merge queue
// part and the Go text that prints it.
type directorSkillQueueText struct {
	// skill is the fragment as the skill writes it, backquotes included.
	skill string
	// file is the non-test Go file that prints it, from the repository root,
	// with slashes.
	file string
	// printed is the text that file's string literals print, where each
	// <...> placeholder stands for a format verb or a value the code joins
	// with + (directorSkillPrintedPattern).
	printed string
}

// directorSkillQueueTexts are the fragments of ralph's output that the
// /director skill's merge queue part decides on, each with the Go text that
// prints it, in the order of the procedure's steps that read them.
var directorSkillQueueTexts = []directorSkillQueueText{
	// Where to look: the line that comes right before the merge queue line.
	{"`merge guarantee:`", "internal/cli/status.go", "merge guarantee: <text>"},
	// Chooses the two-stage procedure: origin's default branch has a merge queue (mergeQueueLine).
	{"`merge queue: origin/<既定のブランチ> has one`", "internal/cli/org_merge_order.go", "merge queue: origin/<branch> has one"},
	// Keeps the one-at-a-time procedure: merge-order could not read the queue (mergeQueueLine).
	{"`merge queue: unknown (<理由>)`", "internal/cli/org_merge_order.go", "merge queue: unknown (<why>)"},
	// Step 1 defers the branch: its own org still runs (runningRefusal, mergeRefusalOrgRunning).
	{"`which is running, so its leader may still push`", "internal/org/merge.go", "which is running, so its leader may still push"},
	// Step 1 stops the branch, not defers it: its org runs another feature now (runningRefusal).
	{"`now for another feature`", "internal/org/merge.go", "now for another feature"},
	// Step 1 defers the branch: gh reported no checks yet (readChecks, mergeRefusalChecksZero).
	{"`reported no checks`", "internal/org/merge.go", "reported no checks"},
	// Step 1 reads the checks named after it to defer or stop the branch (readChecks).
	{"`are not pass or skipping:`", "internal/org/merge.go", "are not pass or skipping: <checks>"},
	// Step 1: the end of a list of checks cut after mergeOrderMaxNamed names (mergeNamedBranches).
	{"`and <N> more`", "internal/org/mergeorder.go", " and <N> more"},
	// Step 1: exit 0 with this line is a merge with no queue, so the procedure is chosen again.
	{"`MERGED:`", "internal/cli/org_merge.go", "MERGED: <url> (PR #<n> of <repo>)"},
	// Step 3: the line that says merge found the PR in the queue already.
	{"`NOTE:`", "internal/cli/org_merge.go", "NOTE: <note>"},
	// Step 3: the line told to the human after exit 7 twice in a row.
	{"`POSITION:`", "internal/cli/org_merge.go", "POSITION: <n>"},
	// Step 5: the refusal of a branch the queue merged, which does not stop it; <class> is MergeClassMerged (classify).
	{"`as merged, not ready or accepted`", "internal/org/merge_queue_order.go", "as <class>, not ready or accepted"},
	// Step 5: the line reported once the cleanup of a branch the queue merged ended.
	{"`CLEANED:`", "internal/cli/org_cleanup.go", "CLEANED: <branch> org-<org_id>"},
	// Step 6: the Detail of a branch to regate one at a time (mergeRegateCommand).
	{"`ralph org regate --branch <ブランチ>`", "internal/org/mergeorder.go", "ralph org regate --branch <branch>"},
}

// directorSkillQueueValues are fragments of the merge queue part that carry
// a Go value, built from that value.
func directorSkillQueueValues() []string {
	return []string{
		// The class of a branch whose gate base is old when the queue could not be read.
		"`" + MergeClassRebase + "`",
		// The classes the two-stage procedure adds to the queue.
		"`" + MergeClassReady + "`",
		"`" + MergeClassAccepted + "`",
		// How many checks the refusal names before `and <N> more` (mergeNamedBranches).
		"列に名前が出る検査は " + strconv.Itoa(mergeOrderMaxNamed) + " 件まで",
		// The class --all gives a branch the queue merged (step 5).
		"`" + MergeClassMerged + "`",
	}
}

// directorSkillQueueBuckets are the buckets of gh pr checks whose name in a
// check's parentheses step 1 decides on: every named check pending defers
// the branch, and one fail or cancel stops it. They are gh's words, which
// ghPRCheck.label prints first in the parentheses.
var directorSkillQueueBuckets = []string{"pending", "fail", "cancel"}

const (
	// directorSkillQueueBulletStart starts the bullet of the section merge と
	// auto_merge that chooses the procedure by the merge queue line.
	directorSkillQueueBulletStart = "- `auto_merge = true`(既定)"
	// directorSkillQueueStepsStart starts the paragraph before the steps of
	// the two-stage procedure.
	directorSkillQueueStepsStart = "merge queue のある repo の 2 段の手順は次のとおり"
)

// directorSkillQueuePart returns the merge queue part of the /director
// skill's text: the bullet of the section merge と auto_merge that chooses
// the procedure, and the paragraph and the numbered steps of the two-stage
// procedure. problems names each piece it did not find.
func directorSkillQueuePart(text string) (part string, problems []string) {
	section := directorSkillSubsection(text, "### merge と auto_merge")
	if section == "" {
		return "", []string{"the skill has no subsection ### merge と auto_merge"}
	}
	bullet := directorSkillBullet(section, directorSkillQueueBulletStart)
	if bullet == "" {
		problems = append(problems, fmt.Sprintf("### merge と auto_merge has no bullet that starts %q", directorSkillQueueBulletStart))
	}
	steps := directorSkillParagraphAndList(section, directorSkillQueueStepsStart)
	if steps == "" {
		problems = append(problems, fmt.Sprintf("### merge と auto_merge has no paragraph that starts %q", directorSkillQueueStepsStart))
	}
	return bullet + "\n" + steps, problems
}

// directorSkillParagraphAndList returns the paragraph of section whose first
// line starts with start and the numbered list after it: the lines from that
// line up to the first blank line, then the lines that are blank, start with
// a space, or start "<number>. ", up to the first other line or the end of
// section, with the blank lines at the end left out. It returns "" when no
// line starts with start.
func directorSkillParagraphAndList(section, start string) string {
	lines := strings.Split(section, "\n")
	first := -1
	for i, line := range lines {
		if strings.HasPrefix(line, start) {
			first = i
			break
		}
	}
	if first < 0 {
		return ""
	}
	end := first
	for end < len(lines) && strings.TrimSpace(lines[end]) != "" {
		end++
	}
	for end < len(lines) {
		line := lines[end]
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, " ") && !directorSkillListItemRe.MatchString(line) {
			break
		}
		end++
	}
	return strings.TrimRight(strings.Join(lines[first:end], "\n"), "\n ")
}

// directorSkillListItemRe matches the first line of an item of a numbered
// list.
var directorSkillListItemRe = regexp.MustCompile(`^[0-9]+\. `)

// directorSkillGoPrintedGap stands, in directorSkillGoPrinted's text, for
// whatever comes between two string literals that are not joined with a lone
// +: a value the code joins in, or the code between two statements.
const directorSkillGoPrintedGap = "\x01"

// directorSkillGoPrinted returns the text of the string literals of the Go
// source src, comments left out: the literals in order, a literal joined to
// the one before it with a lone + written right after it, and
// directorSkillGoPrintedGap before every other literal and at the end. A
// fragment of a refusal is in this text whether the code writes it in one
// literal or splits it with +, and a comment that quotes the fragment does
// not count.
func directorSkillGoPrinted(src []byte) (string, error) {
	var s scanner.Scanner
	fset := token.NewFileSet()
	var scanErr error
	s.Init(fset.AddFile("", fset.Base(), len(src)), src, func(pos token.Position, msg string) {
		if scanErr == nil {
			scanErr = fmt.Errorf("%s: %s", pos, msg)
		}
	}, 0)
	var b strings.Builder
	between := -1   // tokens since the last string literal, -1 before the first
	joined := false // the tokens since the last string literal are a lone +
	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok != token.STRING {
			if between >= 0 {
				between++
				joined = between == 1 && tok == token.ADD
			}
			continue
		}
		value, err := strconv.Unquote(lit)
		if err != nil {
			return "", fmt.Errorf("unquote %s: %w", lit, err)
		}
		if between >= 0 && !joined {
			b.WriteString(directorSkillGoPrintedGap)
		}
		b.WriteString(value)
		between, joined = 0, false
	}
	b.WriteString(directorSkillGoPrintedGap)
	return b.String(), scanErr
}

var (
	// directorSkillPlaceholderRe matches a placeholder of a fragment, <...>.
	directorSkillPlaceholderRe = regexp.MustCompile(`<[^<>\n]+>`)
	// directorSkillPrintedValue matches, in directorSkillGoPrinted's text,
	// what a placeholder stands for: a format verb, or a value joined in.
	directorSkillPrintedValue = `(?:%[-+# 0-9.*\[\]]*[a-zA-Z]|` + directorSkillGoPrintedGap + `)`
)

// directorSkillPrintedPattern is the regular expression that finds printed,
// a text with <...> placeholders, in directorSkillGoPrinted's text: its
// fixed parts as they are, and directorSkillPrintedValue for each
// placeholder.
func directorSkillPrintedPattern(printed string) *regexp.Regexp {
	fixed := directorSkillPlaceholderRe.Split(printed, -1)
	for i, f := range fixed {
		fixed[i] = regexp.QuoteMeta(f)
	}
	return regexp.MustCompile(strings.Join(fixed, directorSkillPrintedValue))
}

// directorSkillQueueSkillProblems returns one line per fragment of
// directorSkillQueueTexts and directorSkillQueueValues that skill, the text
// of a copy of the /director skill, does not have in its merge queue part
// (directorSkillQueuePart), compared with spaces and line breaks removed
// (withoutSpaces), and one per piece of that part it did not find. It
// returns nil when every fragment is there.
func directorSkillQueueSkillProblems(skill string) []string {
	part, problems := directorSkillQueuePart(skill)
	inPart := withoutSpaces(part)
	for _, e := range directorSkillQueueTexts {
		if !strings.Contains(inPart, withoutSpaces(e.skill)) {
			problems = append(problems, fmt.Sprintf("the merge queue part of ### merge と auto_merge does not contain %s, which %s prints", e.skill, e.file))
		}
	}
	for _, v := range directorSkillQueueValues() {
		if !strings.Contains(inPart, withoutSpaces(v)) {
			problems = append(problems, fmt.Sprintf("the merge queue part of ### merge と auto_merge does not contain %q, built from a Go value", v))
		}
	}
	return problems
}

// directorSkillQueueGoProblems returns one line per entry of
// directorSkillQueueTexts whose Go file has no string literal that prints its
// text (directorSkillGoPrinted, directorSkillPrintedPattern), and one per Go
// file it could not read. readGo reads a Go file by its path from the
// repository root. It returns nil when every file prints its texts.
func directorSkillQueueGoProblems(readGo func(rel string) ([]byte, error)) []string {
	var problems []string
	printed := map[string]string{}
	for _, e := range directorSkillQueueTexts {
		text, ok := printed[e.file]
		if !ok {
			src, err := readGo(e.file)
			if err == nil {
				text, err = directorSkillGoPrinted(src)
			}
			if err != nil {
				problems = append(problems, fmt.Sprintf("read %s for %s: %v", e.file, e.skill, err))
				continue
			}
			printed[e.file] = text
		}
		if !directorSkillPrintedPattern(e.printed).MatchString(text) {
			problems = append(problems, fmt.Sprintf("%s, which the skill names as %s, has no string literal that prints %q", e.file, e.skill, e.printed))
		}
	}
	return problems
}

// directorSkillReadGo reads a Go file by its path from root, the repository
// root, with slashes.
func directorSkillReadGo(root string) func(rel string) ([]byte, error) {
	return func(rel string) ([]byte, error) { return os.ReadFile(filepath.Join(root, filepath.FromSlash(rel))) }
}

// TestDirectorSkill_PinsTheMergeQueueTexts ties the /director skill's merge
// queue procedure to the texts ralph prints (plan
// docs/plans/active/2026-11-20-org-merge-queue-order.md, AC6; the root of
// self-review M1, item D10 of docs/reports/verify-2026-11-20-org-merge-queue-order.md).
// The procedure tells one refusal or output of ralph org merge and ralph org
// merge-order from another by English fragments of their stderr and stdout,
// and defers, stops, or reports a branch on what it finds, so a reworded
// refusal makes the director treat a branch the wrong way with nothing else
// failing. scripts/check-skill-sync.sh keeps the four copies of the skill
// identical but does not read Go. ghPRCheck.label must print a check's bucket
// first in its parentheses (directorSkillQueueBuckets); each copy's merge
// queue part must contain every fragment of directorSkillQueueTexts and
// directorSkillQueueValues; and each file of directorSkillQueueTexts must
// print its text.
//
// If this test fails after a deliberate change to a Go text, write the new
// words, in backquotes, in the merge queue part of the section merge と
// auto_merge of .claude/skills/director/SKILL.md and
// templates/base/.claude/skills/director/SKILL.md, run scripts/sync-skills.sh
// for the two .agents copies (root, then with
// CLAUDE_ROOT=templates/base/.claude/skills
// CODEX_ROOT=templates/base/.agents/skills), and update the entry in
// directorSkillQueueTexts or directorSkillQueueValues; otherwise restore the
// text the failure names.
func TestDirectorSkill_PinsTheMergeQueueTexts(t *testing.T) {
	for _, bucket := range directorSkillQueueBuckets {
		check := ghPRCheck{Workflow: "ci", Name: "build", Bucket: bucket, State: "STATE"}
		if want := "ci / build (" + bucket + ", "; !strings.HasPrefix(check.label(), want) {
			t.Errorf("ghPRCheck.label() = %q, want it to start %q: the /director skill reads a check's bucket first in its parentheses -- see this test's doc comment", check.label(), want)
		}
	}
	root := directorSkillRepoRoot(t)
	for _, rel := range directorSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's merge queue text check (vendored repo?)", rel, err)
		}
		for _, p := range directorSkillQueueSkillProblems(string(data)) {
			t.Errorf("%s: %s -- see this test's doc comment for what to update", rel, p)
		}
	}
	for _, p := range directorSkillQueueGoProblems(directorSkillReadGo(root)) {
		t.Errorf("%s -- see this test's doc comment for what to update", p)
	}
}

// TestDirectorSkillQueueProblems runs directorSkillQueueSkillProblems on the
// root copy of the skill and directorSkillQueueGoProblems on the Go files, as
// they are and with one change to a fragment of each side: a fragment of
// step 1 reworded in the skill, and the refusal of readChecks reworded in
// merge.go.
func TestDirectorSkillQueueProblems(t *testing.T) {
	root := directorSkillRepoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, directorSkillCopies[0]))
	if err != nil {
		t.Skipf("%s not found (%v) -- skipping (vendored repo?)", directorSkillCopies[0], err)
	}
	skill, readGo := string(data), directorSkillReadGo(root)
	if p := directorSkillQueueSkillProblems(skill); p != nil {
		t.Errorf("directorSkillQueueSkillProblems on the skill as it is = %q, want none", p)
	}
	if p := directorSkillQueueGoProblems(readGo); p != nil {
		t.Errorf("directorSkillQueueGoProblems on the Go files as they are = %q, want none", p)
	}
	replaced := func(s, from, to string) string {
		t.Helper()
		if n := strings.Count(s, from); n != 1 {
			t.Fatalf("%q is in the text %d times, want once", from, n)
		}
		return strings.Replace(s, from, to, 1)
	}

	want := "the merge queue part of ### merge と auto_merge does not contain `now for another feature`, which internal/org/merge.go prints"
	if p := directorSkillQueueSkillProblems(replaced(skill, "`now for another feature`", "`now for a different feature`")); len(p) != 1 || p[0] != want {
		t.Errorf("directorSkillQueueSkillProblems with a fragment reworded = %q, want [%q]", p, want)
	}

	goChanged := func(rel string) ([]byte, error) {
		src, err := readGo(rel)
		if err != nil || rel != "internal/org/merge.go" {
			return src, err
		}
		return []byte(replaced(string(src), "reported no checks, so", "reported zero checks, so")), nil
	}
	want = "internal/org/merge.go, which the skill names as `reported no checks`, has no string literal that prints \"reported no checks\""
	if p := directorSkillQueueGoProblems(goChanged); len(p) != 1 || p[0] != want {
		t.Errorf("directorSkillQueueGoProblems with a Go text reworded = %q, want [%q]", p, want)
	}
}

// TestDirectorSkillGoPrinted runs directorSkillGoPrinted on a source with a
// comment that quotes a literal, literals joined with + on one line and
// across lines, a value joined in, and a raw string.
func TestDirectorSkillGoPrinted(t *testing.T) {
	src := "package p\n\n// says \"quoted\"\nvar a = \"one \" + \"two\"\nvar b = \"x\" +\n\t\"y\"\nvar c = \"(\" + f(d, e) + \")\"\nvar r = `raw`\n"
	want := "one two" + directorSkillGoPrintedGap + "xy" + directorSkillGoPrintedGap + "(" + directorSkillGoPrintedGap + ")" +
		directorSkillGoPrintedGap + "raw" + directorSkillGoPrintedGap
	if got, err := directorSkillGoPrinted([]byte(src)); err != nil || got != want {
		t.Errorf("directorSkillGoPrinted = %q, %v; want %q, nil", got, err, want)
	}
	for _, tc := range []struct {
		printed, text string
		want          bool
	}{
		{"merge queue: unknown (<why>)", "merge queue: unknown (" + directorSkillGoPrintedGap + ")", true},
		{"merge queue: origin/<b> has one", "merge queue: origin/%s has one%s", true},
		{" and <N> more", " and %d more", true},
		{" and <N> more", " and N more", false},
		{"reported no checks", "reported zero checks", false},
	} {
		if got := directorSkillPrintedPattern(tc.printed).MatchString(tc.text); got != tc.want {
			t.Errorf("directorSkillPrintedPattern(%q).MatchString(%q) = %v, want %v", tc.printed, tc.text, got, tc.want)
		}
	}
}

// TestDirectorSkillParagraphAndList runs directorSkillParagraphAndList on a
// paragraph of two lines followed by a list with an indented line, a nested
// bullet and a blank line, then a paragraph; on a paragraph with no list; and
// on a start no line has.
func TestDirectorSkillParagraphAndList(t *testing.T) {
	section := "### x\n\nlead\nmore\n\n1. one\n   - sub\n\n2. two\n   2b\n\nafter\n\nlone\nend\n"
	for _, tc := range []struct{ start, want string }{
		{"lead", "lead\nmore\n\n1. one\n   - sub\n\n2. two\n   2b"},
		{"lone", "lone\nend"},
		{"missing", ""},
	} {
		if got := directorSkillParagraphAndList(section, tc.start); got != tc.want {
			t.Errorf("directorSkillParagraphAndList(section, %q) = %q, want %q", tc.start, got, tc.want)
		}
	}
}
