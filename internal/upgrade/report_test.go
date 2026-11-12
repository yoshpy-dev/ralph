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

// TestRenderUpgradeReport_FromVersion_VersionLineAfterGenerated pins AC7 of
// plan 2026-11-12-scaffold-version-notice (D6): with FromVersion set, the
// "Version: <from> → <to>" line sits right after the Generated line, before
// the Summary section.
func TestRenderUpgradeReport_FromVersion_VersionLineAfterGenerated(t *testing.T) {
	data := UpgradeReportData{
		TemplateVersion: "v2.0.0",
		FromVersion:     "v1.9.0",
		GeneratedAt:     "2026-08-17T00:00:00Z",
	}
	got := string(RenderUpgradeReport(data))

	wantHeader := "# Upgrade Report — v2.0.0\n\nGenerated: 2026-08-17T00:00:00Z\n\nVersion: v1.9.0 → v2.0.0\n\n## Summary\n"
	if !strings.HasPrefix(got, wantHeader) {
		t.Errorf("report header = %q, want prefix %q", got, wantHeader)
	}
	if n := strings.Count(got, "Version: "); n != 1 {
		t.Errorf("want exactly one Version line, got %d:\n%s", n, got)
	}
}

// TestRenderUpgradeReport_EmptyFromVersion_NoVersionLine is the other half of
// D6: with FromVersion empty (unknown), the report has no Version line and
// its header is byte-identical to the one rendered before FromVersion
// existed.
func TestRenderUpgradeReport_EmptyFromVersion_NoVersionLine(t *testing.T) {
	data := UpgradeReportData{
		TemplateVersion: "v2.0.0",
		GeneratedAt:     "2026-08-17T00:00:00Z",
	}
	got := string(RenderUpgradeReport(data))

	wantHeader := "# Upgrade Report — v2.0.0\n\nGenerated: 2026-08-17T00:00:00Z\n\n## Summary\n"
	if !strings.HasPrefix(got, wantHeader) {
		t.Errorf("report header = %q, want prefix %q", got, wantHeader)
	}
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "Version:") {
			t.Errorf("want no Version line with an empty FromVersion, got %q in:\n%s", line, got)
		}
	}
}

// twoBehaviorNotes returns two notes, oldest first, as BehaviorNotesBetween
// would return them for an upgrade from v1.9.0 to v2.0.0.
func twoBehaviorNotes() []BehaviorNote {
	return []BehaviorNote{
		{Version: "1.9.5", Body: "- the director now reads the inbox first\n- /pr asks before it pushes"},
		{Version: "2.0.0", Body: "- the reviewer seat runs the gate"},
	}
}

// TestRenderUpgradeReport_BehaviorNotes_SectionAfterVersionLine pins AC1 and
// AC3 of plan 2026-11-13-upgrade-behavior-notes (issue #317): the "Behavior
// changes" section comes right after the Version line and before the
// Summary, with one "### <version>" heading per note, in the order given,
// each followed by its body.
func TestRenderUpgradeReport_BehaviorNotes_SectionAfterVersionLine(t *testing.T) {
	data := UpgradeReportData{
		TemplateVersion: "2.0.0",
		FromVersion:     "1.9.0",
		GeneratedAt:     "2026-08-17T00:00:00Z",
		BehaviorNotes:   twoBehaviorNotes(),
	}
	got := string(RenderUpgradeReport(data))

	wantHeader := "# Upgrade Report — 2.0.0\n\nGenerated: 2026-08-17T00:00:00Z\n\nVersion: 1.9.0 → 2.0.0\n\n## Behavior changes\n\n"
	if !strings.HasPrefix(got, wantHeader) {
		t.Fatalf("report header = %q, want prefix %q", got, wantHeader)
	}
	wantNotes := "### 1.9.5\n\n- the director now reads the inbox first\n- /pr asks before it pushes\n\n" +
		"### 2.0.0\n\n- the reviewer seat runs the gate\n\n## Summary\n"
	if !strings.Contains(got, wantNotes) {
		t.Errorf("want the notes in the given order, each body after its heading, then the Summary:\n%s", got)
	}
	// One sentence sits between the section heading and the first note.
	intro := got[len(wantHeader):strings.Index(got, "### 1.9.5")]
	if strings.Count(intro, "\n") != 2 || !strings.HasSuffix(intro, ".\n\n") {
		t.Errorf("want one paragraph between the section heading and the first note, got %q", intro)
	}
	if strings.Contains(intro, "#") {
		t.Errorf("the sentence before the notes must not hold a heading, got %q", intro)
	}
	if n := strings.Count(got, "## Behavior changes"); n != 1 {
		t.Errorf("want exactly one Behavior changes section, got %d:\n%s", n, got)
	}
}

// TestRenderUpgradeReport_BehaviorNotes_OnlyAddTheSection checks that the
// notes change nothing else in a full report: the report with notes is the
// report without them plus the section, inserted right before "## Summary"
// (AC3 of plan 2026-11-13-upgrade-behavior-notes).
func TestRenderUpgradeReport_BehaviorNotes_OnlyAddTheSection(t *testing.T) {
	without := fullReportData()
	without.FromVersion = "v1.9.0"
	with := without
	with.BehaviorNotes = twoBehaviorNotes()

	gotWithout := string(RenderUpgradeReport(without))
	gotWith := string(RenderUpgradeReport(with))

	start := strings.Index(gotWith, "## Behavior changes\n")
	end := strings.Index(gotWith, "## Summary\n")
	if start < 0 || end < start {
		t.Fatalf("want a Behavior changes section before the Summary:\n%s", gotWith)
	}
	if stripped := gotWith[:start] + gotWith[end:]; stripped != gotWithout {
		t.Errorf("the report with notes, less its Behavior changes section, must equal the report without notes:\n--- with, section removed ---\n%s\n--- without ---\n%s", stripped, gotWithout)
	}
}

// TestRenderUpgradeReport_NoBehaviorNotes_ByteIdentical is the other half of
// AC3 of plan 2026-11-13-upgrade-behavior-notes: with no notes (nil or an
// empty slice) the report is byte for byte the one rendered before
// BehaviorNotes existed.
func TestRenderUpgradeReport_NoBehaviorNotes_ByteIdentical(t *testing.T) {
	want := "# Upgrade Report — 2.0.0\n\n" +
		"Generated: 2026-08-17T00:00:00Z\n\n" +
		"Version: 1.9.0 → 2.0.0\n\n" +
		"## Summary\n\n" +
		"| Category | Count |\n" +
		"|---|---|\n" +
		"| Deleted | 0 |\n" +
		"| Created | 0 |\n" +
		"| Updated | 0 |\n" +
		"| Manifest refresh | 0 |\n" +
		"| Unresolved drift | 0 |\n" +
		"| Advisories | 0 |\n" +
		"| Legacy skipped | 0 |\n" +
		"\n"
	for _, tc := range []struct {
		name  string
		notes []BehaviorNote
	}{
		{"nil", nil},
		{"empty", []BehaviorNote{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := UpgradeReportData{
				TemplateVersion: "2.0.0",
				FromVersion:     "1.9.0",
				GeneratedAt:     "2026-08-17T00:00:00Z",
				BehaviorNotes:   tc.notes,
			}
			if got := string(RenderUpgradeReport(data)); got != want {
				t.Errorf("report = %q, want %q", got, want)
			}
		})
	}
}

// TestRenderUpgradeReport_BehaviorNotes_EmptyFromVersionStillRendered pins
// that the renderer does not choose the notes: BehaviorNotesBetween does
// (it returns none when from is not a semver, an empty one included), and
// RenderUpgradeReport writes whatever it is given, right after the
// Generated line when there is no Version line (plan
// 2026-11-13-upgrade-behavior-notes).
func TestRenderUpgradeReport_BehaviorNotes_EmptyFromVersionStillRendered(t *testing.T) {
	data := UpgradeReportData{
		TemplateVersion: "2.0.0",
		GeneratedAt:     "2026-08-17T00:00:00Z",
		BehaviorNotes:   twoBehaviorNotes()[1:],
	}
	got := string(RenderUpgradeReport(data))

	wantHeader := "# Upgrade Report — 2.0.0\n\nGenerated: 2026-08-17T00:00:00Z\n\n## Behavior changes\n\n"
	if !strings.HasPrefix(got, wantHeader) {
		t.Errorf("report header = %q, want prefix %q", got, wantHeader)
	}
	if !strings.Contains(got, "### 2.0.0\n\n- the reviewer seat runs the gate\n\n## Summary\n") {
		t.Errorf("want the given note before the Summary:\n%s", got)
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

// mustWriteReportFile creates targetDir/relPath as a regular file holding
// content, with its parent directories.
func mustWriteReportFile(t *testing.T, targetDir, relPath, content string) {
	t.Helper()
	full := filepath.Join(targetDir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", relPath, err)
	}
}

// TestFreeUpgradeReportRelPath_TakesTheFirstFreeName pins D6 and AC9 of plan
// 2026-11-13-upgrade-behavior-notes (issue #317): the first name is
// UpgradeReportRelPath's; once something is at it, the report goes to -2,
// then -3, and so on.
func TestFreeUpgradeReportRelPath_TakesTheFirstFreeName(t *testing.T) {
	dir := t.TempDir()
	const base = "docs/reports/upgrade-8.3.0-2026-11-13"

	steps := []string{base + ".md", base + "-2.md", base + "-3.md"}
	for i, want := range steps {
		got, err := FreeUpgradeReportRelPath(dir, "8.3.0", "2026-11-13")
		if err != nil {
			t.Fatalf("step %d: FreeUpgradeReportRelPath: %v", i+1, err)
		}
		if got != want {
			t.Fatalf("step %d: got %q, want %q", i+1, got, want)
		}
		mustWriteReportFile(t, dir, got, "report\n")
	}
	if got := UpgradeReportRelPath("8.3.0", "2026-11-13"); got != steps[0] {
		t.Errorf("UpgradeReportRelPath = %q, want it unchanged at %q", got, steps[0])
	}
}

// TestFreeUpgradeReportRelPath_AnyEntryCountsAsTaken checks that a directory
// and a symlink at a name, dangling or not, take that name like a file does:
// the name is never handed to WriteUpgradeReport, which would refuse it.
func TestFreeUpgradeReportRelPath_AnyEntryCountsAsTaken(t *testing.T) {
	dir := t.TempDir()
	const base = "docs/reports/upgrade-8.3.0-2026-11-13"

	if err := os.MkdirAll(filepath.Join(dir, filepath.FromSlash(base+".md")), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.Symlink(filepath.Join(dir, "missing-target.md"), filepath.Join(dir, filepath.FromSlash(base+"-2.md"))); err != nil {
		t.Fatalf("Symlink (dangling): %v", err)
	}
	mustWriteReportFile(t, dir, "elsewhere.md", "x\n")
	if err := os.Symlink(filepath.Join(dir, "elsewhere.md"), filepath.Join(dir, filepath.FromSlash(base+"-3.md"))); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	got, err := FreeUpgradeReportRelPath(dir, "8.3.0", "2026-11-13")
	if err != nil {
		t.Fatalf("FreeUpgradeReportRelPath: %v", err)
	}
	if want := base + "-4.md"; got != want {
		t.Errorf("got %q, want %q (a directory, a dangling symlink, and a symlink to a file all take their names)", got, want)
	}
}

// TestFreeUpgradeReportRelPath_GivesUpAfterTheLimit checks the bound: with
// the first name and -2 to -upgradeReportNameLimit all taken it returns an
// error naming the first name, and writes nothing.
func TestFreeUpgradeReportRelPath_GivesUpAfterTheLimit(t *testing.T) {
	if upgradeReportNameLimit != 100 {
		t.Errorf("upgradeReportNameLimit = %d, want 100", upgradeReportNameLimit)
	}
	dir := t.TempDir()
	for n := 1; n <= upgradeReportNameLimit; n++ {
		mustWriteReportFile(t, dir, upgradeReportRelPathN("8.3.0", "2026-11-13", n), "report\n")
	}

	got, err := FreeUpgradeReportRelPath(dir, "8.3.0", "2026-11-13")
	if err == nil {
		t.Fatalf("want an error once all %d names are taken, got %q", upgradeReportNameLimit, got)
	}
	if !strings.Contains(err.Error(), "docs/reports/upgrade-8.3.0-2026-11-13.md") || !strings.Contains(err.Error(), "-100") {
		t.Errorf("error must name the first name and the last suffix tried: %v", err)
	}
	entries, rerr := os.ReadDir(filepath.Join(dir, "docs", "reports"))
	if rerr != nil {
		t.Fatalf("ReadDir: %v", rerr)
	}
	if len(entries) != upgradeReportNameLimit {
		t.Errorf("docs/reports holds %d entries, want the %d made by the test", len(entries), upgradeReportNameLimit)
	}

	// One name freed in the middle is found again.
	freed := upgradeReportRelPathN("8.3.0", "2026-11-13", 57)
	if err := os.Remove(filepath.Join(dir, filepath.FromSlash(freed))); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	got, err = FreeUpgradeReportRelPath(dir, "8.3.0", "2026-11-13")
	if err != nil || got != freed {
		t.Errorf("after freeing %q: got %q, %v; want %q, nil", freed, got, err, freed)
	}
}

// TestFreeUpgradeReportRelPath_SymlinkedReportsDir_Refused checks that the
// name is not looked up through a symlinked docs/reports: the parent chain is
// checked first, and the error comes before any name is chosen.
func TestFreeUpgradeReportRelPath_SymlinkedReportsDir_Refused(t *testing.T) {
	dir := t.TempDir()
	external := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.Symlink(external, filepath.Join(dir, "docs", "reports")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	got, err := FreeUpgradeReportRelPath(dir, "8.3.0", "2026-11-13")
	if err == nil {
		t.Fatalf("want an error for a symlinked docs/reports, got %q", got)
	}
}

// TestFreeUpgradeReportRelPath_SanitizesLikeUpgradeReportRelPath checks that
// the -<n> names are sanitized the same way as the first one, so a version or
// date holding a separator cannot leave docs/reports/.
func TestFreeUpgradeReportRelPath_SanitizesLikeUpgradeReportRelPath(t *testing.T) {
	dir := t.TempDir()
	first := UpgradeReportRelPath("v2/0 0", "x/../../AGENTS")
	mustWriteReportFile(t, dir, first, "report\n")

	got, err := FreeUpgradeReportRelPath(dir, "v2/0 0", "x/../../AGENTS")
	if err != nil {
		t.Fatalf("FreeUpgradeReportRelPath: %v", err)
	}
	if want := strings.TrimSuffix(first, ".md") + "-2.md"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if rest := strings.TrimPrefix(got, "docs/reports/"); rest == got || strings.ContainsAny(rest, "/\\ ") {
		t.Errorf("want a file name directly under docs/reports/ with no separator or space, got %q", got)
	}
}
