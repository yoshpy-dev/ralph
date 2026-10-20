package org

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// reviewerReportPrefix starts the name of the reviewer seat's report
// (prompts/reviewer.md, レポート契約). scripts/check-sync.sh excludes it as a
// prefix, so a feature org's gate does not report the file as ROOT_ONLY.
const reviewerReportPrefix = "docs/reports/review-"

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
// docs/plans/active/2026-10-20-org-report-names.md: the names of the two
// reports a feature org writes into its worktree, the reviewer seat's report
// (reviewerReportPrefix) and the org report of `ralph org report`
// (defaultReportDir and orgReportFilePrefix in report.go), are spelled the
// same in every hand-written place. A feature org's gate runs
// scripts/check-sync.sh, which reports either file as ROOT_ONLY unless its
// prefix is excluded, and nothing else catches a rename on one side only.
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
		reviewerName := strings.TrimPrefix(reviewerReportPrefix, "docs/reports/")
		for _, want := range []string{
			"- `" + reviewerName + "YYYY-MM-DD-<org_id>-<seat_id>.md`",
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
