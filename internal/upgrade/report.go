package upgrade

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// UpgradeReportData is the input to RenderUpgradeReport: the outcome of a
// ReplacePlan applied against targetDir, plus rendered advisory diffs and
// free-form notes. All slices may be given in any order — rendering sorts
// by path so output is deterministic regardless of caller ordering.
type UpgradeReportData struct {
	// TemplateVersion is the version being upgraded to.
	TemplateVersion string
	// GeneratedAt is a caller-supplied timestamp string. The render path
	// never calls time.Now itself, so output is reproducible in tests.
	GeneratedAt string

	// DeletedPaths, CreatedPaths, UpdatedPaths are the per-op path lists
	// from an applied ReplacePlan.Ops.
	DeletedPaths []string
	CreatedPaths []string
	UpdatedPaths []string

	// ManifestRefreshPaths lists paths whose manifest hash was advanced
	// without a file write (ReplacePlan.ManifestRefresh).
	ManifestRefreshPaths []string

	// UnresolvedDrift is ReplacePlan.Drift: paths left untouched because
	// disk content diverges from both the recorded and new template hash.
	UnresolvedDrift []DriftEntry

	// Advisories is the rendered output of RenderAdvisoryDiffs.
	Advisories []AdvisoryDiff

	// LegacySkipped is ReplacePlan.LegacySkipped: block-owned or legacy
	// (unattributed) paths left entirely alone.
	LegacySkipped []string

	// Notes are free-form lines appended in a trailing "Notes" section.
	Notes []string
}

// RenderUpgradeReport renders data as deterministic markdown. Sections with
// no content are omitted; sections with content list entries in sorted-path
// order. Rendering the same data twice produces byte-identical output.
func RenderUpgradeReport(data UpgradeReportData) []byte {
	var b strings.Builder

	fmt.Fprintf(&b, "# Upgrade Report — %s\n\n", data.TemplateVersion)
	fmt.Fprintf(&b, "Generated: %s\n\n", data.GeneratedAt)

	renderSummaryTable(&b, data)
	renderAppliedSection(&b, data)
	renderPathListSection(&b, "Manifest refresh", data.ManifestRefreshPaths)
	renderDriftSection(&b, data.UnresolvedDrift)
	renderAdvisoriesSection(&b, data.Advisories)
	renderPathListSection(&b, "Legacy skipped", data.LegacySkipped)
	renderNotesSection(&b, data.Notes)

	return []byte(b.String())
}

func renderSummaryTable(b *strings.Builder, data UpgradeReportData) {
	b.WriteString("## Summary\n\n")
	b.WriteString("| Category | Count |\n")
	b.WriteString("|---|---|\n")
	fmt.Fprintf(b, "| Deleted | %d |\n", len(data.DeletedPaths))
	fmt.Fprintf(b, "| Created | %d |\n", len(data.CreatedPaths))
	fmt.Fprintf(b, "| Updated | %d |\n", len(data.UpdatedPaths))
	fmt.Fprintf(b, "| Manifest refresh | %d |\n", len(data.ManifestRefreshPaths))
	fmt.Fprintf(b, "| Unresolved drift | %d |\n", len(data.UnresolvedDrift))
	fmt.Fprintf(b, "| Advisories | %d |\n", len(data.Advisories))
	fmt.Fprintf(b, "| Legacy skipped | %d |\n", len(data.LegacySkipped))
	b.WriteString("\n")
}

func renderAppliedSection(b *strings.Builder, data UpgradeReportData) {
	if len(data.DeletedPaths) == 0 && len(data.CreatedPaths) == 0 && len(data.UpdatedPaths) == 0 {
		return
	}
	b.WriteString("## Applied\n\n")
	renderSubPathList(b, "Deleted", data.DeletedPaths)
	renderSubPathList(b, "Created", data.CreatedPaths)
	renderSubPathList(b, "Updated", data.UpdatedPaths)
}

func renderSubPathList(b *strings.Builder, title string, paths []string) {
	if len(paths) == 0 {
		return
	}
	fmt.Fprintf(b, "### %s\n\n", title)
	for _, p := range sortedCopy(paths) {
		fmt.Fprintf(b, "- `%s`\n", p)
	}
	b.WriteString("\n")
}

func renderPathListSection(b *strings.Builder, title string, paths []string) {
	if len(paths) == 0 {
		return
	}
	fmt.Fprintf(b, "## %s\n\n", title)
	for _, p := range sortedCopy(paths) {
		fmt.Fprintf(b, "- `%s`\n", p)
	}
	b.WriteString("\n")
}

func renderDriftSection(b *strings.Builder, entries []DriftEntry) {
	if len(entries) == 0 {
		return
	}
	sorted := make([]DriftEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })

	b.WriteString("## Unresolved drift\n\n")
	b.WriteString("| Path | Recorded | Disk | New |\n")
	b.WriteString("|---|---|---|---|\n")
	for _, e := range sorted {
		fmt.Fprintf(b, "| `%s` | %s | %s | %s |\n", e.Path, emptyDash(e.RecordedHash), emptyDash(e.DiskHash), emptyDash(e.NewHash))
	}
	b.WriteString("\n")
}

func renderAdvisoriesSection(b *strings.Builder, advisories []AdvisoryDiff) {
	if len(advisories) == 0 {
		return
	}
	sorted := make([]AdvisoryDiff, len(advisories))
	copy(sorted, advisories)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })

	b.WriteString("## Advisories\n\n")
	for _, a := range sorted {
		fmt.Fprintf(b, "### `%s` (owner: %s)\n\n", a.Path, a.Owner)
		if a.Skipped {
			fmt.Fprintf(b, "_Skipped: %s._\n\n", a.Note)
			continue
		}
		if a.Diff == "" {
			b.WriteString("_No differences._\n\n")
			continue
		}
		b.WriteString("```diff\n")
		b.WriteString(a.Diff)
		if !strings.HasSuffix(a.Diff, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("```\n\n")
	}
}

func renderNotesSection(b *strings.Builder, notes []string) {
	if len(notes) == 0 {
		return
	}
	b.WriteString("## Notes\n\n")
	for _, n := range notes {
		fmt.Fprintf(b, "- %s\n", n)
	}
	b.WriteString("\n")
}

func emptyDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func sortedCopy(paths []string) []string {
	sorted := make([]string, len(paths))
	copy(sorted, paths)
	sort.Strings(sorted)
	return sorted
}

// reportNameSanitizeRe matches characters not allowed in an upgrade report
// filename's version or date component.
var reportNameSanitizeRe = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// upgradeReportDir is the single source of truth for the upgrade report
// directory: UpgradeReportRelPath builds its output path under this
// directory, and WriteUpgradeReport requires relPath to resolve under it.
const upgradeReportDir = "docs/reports"

// UpgradeReportRelPath returns the manifest-relative path an upgrade report
// for the given template version and date should be written to:
// docs/reports/upgrade-<version>-<date>.md. Both version and date are
// sanitized by stripping any character outside [A-Za-z0-9._-] so path
// separators or spaces in either value cannot escape the reports directory.
func UpgradeReportRelPath(version, date string) string {
	safeVersion := reportNameSanitizeRe.ReplaceAllString(version, "")
	safeDate := reportNameSanitizeRe.ReplaceAllString(date, "")
	return filepath.ToSlash(filepath.Join(upgradeReportDir, fmt.Sprintf("upgrade-%s-%s.md", safeVersion, safeDate)))
}

// WriteUpgradeReport validates relPath (via scaffold.CleanLocalRelPath) and
// writes content to targetDir/relPath, creating parent directories as
// needed. relPath must resolve under docs/reports/ — WriteUpgradeReport
// rejects any cleaned path outside that prefix (including the directory
// path itself, with no filename), even if it is otherwise a valid
// local-relative path.
//
// This write also bypasses ApplyOps (it runs after the core replace plan
// has already been applied), so it applies the same containment checks
// ApplyOps and the v2 exception-face writes use: ValidateRealParentChain
// against every existing parent path component (guards against e.g. a
// symlinked docs/ or docs/reports/ directory), plus an Lstat of the leaf
// that rejects anything other than a regular file or an absent entry.
func WriteUpgradeReport(targetDir string, relPath string, content []byte) error {
	clean, err := cleanPathKey(relPath)
	if err != nil {
		return fmt.Errorf("upgrade report path %q: %w", relPath, err)
	}
	if !strings.HasPrefix(clean, upgradeReportDir+"/") {
		return fmt.Errorf("upgrade report path %q: must be under %s/", relPath, upgradeReportDir)
	}

	if err := ValidateRealParentChain(targetDir, clean); err != nil {
		return fmt.Errorf("upgrade report path %q: %w", relPath, err)
	}

	full := filepath.Join(targetDir, filepath.FromSlash(clean))
	if fi, err := os.Lstat(full); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("upgrade report path %q: lstat: %w", relPath, err)
		}
	} else if !fi.Mode().IsRegular() {
		return fmt.Errorf("upgrade report path %q: refusing to operate on non-regular file (mode %s)", relPath, fi.Mode())
	}

	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return fmt.Errorf("creating upgrade report dir for %q: %w", clean, err)
	}
	if err := os.WriteFile(full, content, 0644); err != nil {
		return fmt.Errorf("writing upgrade report %q: %w", clean, err)
	}
	return nil
}
