package upgrade

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fullReportData() UpgradeReportData {
	return UpgradeReportData{
		TemplateVersion:      "v2.0.0",
		GeneratedAt:          "2026-08-17T00:00:00Z",
		DeletedPaths:         []string{"old/gone.md", "another/gone.md"},
		CreatedPaths:         []string{"new/added.md"},
		UpdatedPaths:         []string{"core/updated.md"},
		ManifestRefreshPaths: []string{"settled/already-applied.md"},
		UnresolvedDrift: []DriftEntry{
			{Path: "drift/file.md", RecordedHash: "sha256:aaa", DiskHash: "sha256:bbb", NewHash: "sha256:ccc"},
		},
		Advisories: []AdvisoryDiff{
			{Path: "fork/changed.md", Owner: "fork", Diff: "--- local\n+++ template (v2.0.0)\n@@ 旧 L1  →  新 L1 @@\n 1  1 │  x\n"},
			{Path: "seed/removed.md", Owner: "seed", Skipped: true, Note: "template no longer has this path"},
		},
		LegacySkipped: []string{"legacy/entry.md"},
		Notes:         []string{"free-form note one", "free-form note two"},
	}
}

func TestRenderUpgradeReport_FullData_SectionsInOrder(t *testing.T) {
	got := string(RenderUpgradeReport(fullReportData()))

	sections := []string{
		"# Upgrade Report — v2.0.0",
		"## Summary",
		"## Applied",
		"### Deleted",
		"### Created",
		"### Updated",
		"## Manifest refresh",
		"## Unresolved drift",
		"## Advisories",
		"## Legacy skipped",
		"## Notes",
	}
	lastIdx := -1
	for _, s := range sections {
		idx := strings.Index(got, s)
		if idx < 0 {
			t.Fatalf("expected section %q in report, got:\n%s", s, got)
		}
		if idx <= lastIdx {
			t.Errorf("section %q out of order (idx %d <= previous %d)", s, idx, lastIdx)
		}
		lastIdx = idx
	}

	if !strings.Contains(got, "| Deleted | 2 |") {
		t.Errorf("expected deleted count 2, got:\n%s", got)
	}
	if !strings.Contains(got, "| Created | 1 |") {
		t.Errorf("expected created count 1, got:\n%s", got)
	}
	if !strings.Contains(got, "| Advisories | 2 |") {
		t.Errorf("expected advisories count 2, got:\n%s", got)
	}
	if !strings.Contains(got, "`another/gone.md`") || !strings.Contains(got, "`old/gone.md`") {
		t.Errorf("expected both deleted paths listed, got:\n%s", got)
	}
	// Sorted order: "another/gone.md" < "old/gone.md".
	if strings.Index(got, "another/gone.md") > strings.Index(got, "old/gone.md") {
		t.Errorf("expected deleted paths in sorted order, got:\n%s", got)
	}
	if !strings.Contains(got, "```diff") {
		t.Errorf("expected fenced diff block for advisory, got:\n%s", got)
	}
	if !strings.Contains(got, "_Skipped: template no longer has this path._") {
		t.Errorf("expected skipped advisory note, got:\n%s", got)
	}
	if !strings.Contains(got, "sha256:aaa") || !strings.Contains(got, "sha256:bbb") || !strings.Contains(got, "sha256:ccc") {
		t.Errorf("expected all three drift hashes, got:\n%s", got)
	}
	if !strings.Contains(got, "free-form note one") || !strings.Contains(got, "free-form note two") {
		t.Errorf("expected both notes, got:\n%s", got)
	}
}

func TestRenderUpgradeReport_EmptySectionsOmitted(t *testing.T) {
	data := UpgradeReportData{
		TemplateVersion: "v2.0.0",
		GeneratedAt:     "2026-08-17T00:00:00Z",
	}
	got := string(RenderUpgradeReport(data))

	for _, s := range []string{"## Applied", "## Manifest refresh", "## Unresolved drift", "## Advisories", "## Legacy skipped", "## Notes"} {
		if strings.Contains(got, s) {
			t.Errorf("expected section %q to be omitted for empty data, got:\n%s", s, got)
		}
	}
	if !strings.Contains(got, "## Summary") {
		t.Errorf("expected Summary section to always render, got:\n%s", got)
	}
}

func TestRenderUpgradeReport_DeterministicDoubleRender(t *testing.T) {
	data := fullReportData()
	got1 := RenderUpgradeReport(data)
	got2 := RenderUpgradeReport(data)
	if string(got1) != string(got2) {
		t.Errorf("expected identical output across renders:\n%s\n---\n%s", got1, got2)
	}
}

func TestUpgradeReportRelPath_Basic(t *testing.T) {
	got := UpgradeReportRelPath("v2.0.0", "2026-08-17")
	want := "docs/reports/upgrade-v2.0.0-2026-08-17.md"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestUpgradeReportRelPath_SanitizesSlashesAndSpaces(t *testing.T) {
	got := UpgradeReportRelPath("v2/0 0", "2026-08-17")
	if strings.Contains(got, "/0 0") || strings.Contains(got, " ") {
		t.Errorf("expected slashes and spaces stripped from version, got %q", got)
	}
	if !strings.HasPrefix(got, "docs/reports/upgrade-") || !strings.HasSuffix(got, "-2026-08-17.md") {
		t.Errorf("expected sanitized version to still fit the upgrade-<version>-<date>.md shape, got %q", got)
	}
}

func TestUpgradeReportRelPath_SanitizesDateParentEscape(t *testing.T) {
	got := UpgradeReportRelPath("v2.0.0", "x/../../../AGENTS")
	if !strings.HasPrefix(got, "docs/reports/upgrade-v2.0.0-") {
		t.Fatalf("expected sanitized date to still fit the upgrade-<version>-<date>.md shape, got %q", got)
	}
	dateComponent := strings.TrimSuffix(strings.TrimPrefix(got, "docs/reports/upgrade-v2.0.0-"), ".md")
	if strings.ContainsAny(dateComponent, "/\\") {
		t.Errorf("expected date component to contain no path separators, got %q (full path %q)", dateComponent, got)
	}
	if !strings.HasPrefix(got, "docs/reports/") {
		t.Errorf("expected path to stay inside docs/reports/, got %q", got)
	}
}

func TestWriteUpgradeReport_HappyPath(t *testing.T) {
	dir := t.TempDir()
	relPath := UpgradeReportRelPath("v2.0.0", "2026-08-17")
	content := []byte("# report\n")

	if err := WriteUpgradeReport(dir, relPath, content); err != nil {
		t.Fatalf("WriteUpgradeReport: %v", err)
	}

	full := filepath.Join(dir, filepath.FromSlash(relPath))
	got, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("reading written report: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestWriteUpgradeReport_RejectsParentEscape(t *testing.T) {
	dir := t.TempDir()
	err := WriteUpgradeReport(dir, "../escape.md", []byte("x"))
	if err == nil {
		t.Fatalf("expected an error for a path escaping the local tree")
	}
}

// TestWriteUpgradeReport_RejectsPathOutsideReportsDir proves the stricter
// prefix check: a path that is a perfectly valid local-relative path (no
// "..", not absolute) but does not resolve under docs/reports/ must still
// be rejected, since WriteUpgradeReport's whole contract is writing upgrade
// reports to that directory.
func TestWriteUpgradeReport_RejectsPathOutsideReportsDir(t *testing.T) {
	dir := t.TempDir()
	err := WriteUpgradeReport(dir, "AGENTS.md", []byte("x"))
	if err == nil {
		t.Fatalf("expected an error for a path outside docs/reports/")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "AGENTS.md")); !os.IsNotExist(statErr) {
		t.Errorf("AGENTS.md should not have been written: stat err = %v", statErr)
	}
}

// TestWriteUpgradeReport_RejectsReportsDirItself proves the guard has no
// exception for the reports directory path with no filename: relPath ==
// "docs/reports" must be rejected, not accepted as if it named the
// directory. Accepting it would let os.WriteFile create a *file* named
// docs/reports, permanently blocking every later report write in that tree.
func TestWriteUpgradeReport_RejectsReportsDirItself(t *testing.T) {
	dir := t.TempDir()
	err := WriteUpgradeReport(dir, "docs/reports", []byte("x"))
	if err == nil {
		t.Fatalf("expected an error for relPath == \"docs/reports\" (no filename)")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "docs", "reports")); !os.IsNotExist(statErr) {
		t.Errorf("docs/reports should not have been written as a file: stat err = %v", statErr)
	}
}
