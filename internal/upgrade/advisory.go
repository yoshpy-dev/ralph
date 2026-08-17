package upgrade

import (
	"errors"
	"fmt"
	"io/fs"
)

// AdvisoryDiff is the rendered outcome of comparing one AdvisoryEntry's disk
// content against the new template content.
type AdvisoryDiff struct {
	Path  string
	Owner string
	// Diff is the unified diff (disk vs template), labeled "local" and
	// "template (<version>)". Empty when contents are byte-identical, or
	// when Skipped is true.
	Diff string
	// Skipped is true when the template no longer has this path — nothing
	// to compare against, so no diff is rendered.
	Skipped bool
	// Note explains why Skipped is true. Empty otherwise.
	Note string
}

// RenderAdvisoryDiffs loads and renders unified diffs for each AdvisoryEntry
// in entries, comparing disk content in targetDir against the new template
// content in templateFS, labeled with templateVersion.
//
// Advisory selection is not re-derived here: entries is expected to be
// ReplacePlan.Advisories from PlanCoreReplace, which is the source of truth
// for which paths are advisory-worthy. This function only loads both sides
// and renders the diff. Every path is validated with
// scaffold.CleanLocalRelPath (via the shared cleanPathKey helper) before
// reading, per spec AC-9.
//
// A missing disk file renders as a full-addition diff against an empty
// local side. A missing template file (removed from the template since the
// advisory was recorded) is reported as Skipped with an explanatory Note
// rather than an error, since the template legitimately dropping a fork/seed
// path is not a failure condition.
func RenderAdvisoryDiffs(targetDir string, templateFS fs.FS, templateVersion string, entries []AdvisoryEntry) ([]AdvisoryDiff, error) {
	diffs := make([]AdvisoryDiff, 0, len(entries))
	for _, entry := range entries {
		clean, err := cleanPathKey(entry.Path)
		if err != nil {
			return nil, fmt.Errorf("advisory path %q: %w", entry.Path, err)
		}

		localContent, _, err := readDiskFile(targetDir, clean)
		if err != nil {
			return nil, fmt.Errorf("reading disk file %q: %w", clean, err)
		}

		tmplContent, hasTemplate, err := readTemplateFile(templateFS, clean)
		if err != nil {
			return nil, fmt.Errorf("reading template file %q: %w", clean, err)
		}

		if !hasTemplate {
			diffs = append(diffs, AdvisoryDiff{
				Path:    clean,
				Owner:   entry.Owner,
				Skipped: true,
				Note:    "template no longer has this path",
			})
			continue
		}

		newLabel := fmt.Sprintf("template (%s)", templateVersion)
		diffs = append(diffs, AdvisoryDiff{
			Path:  clean,
			Owner: entry.Owner,
			Diff:  UnifiedDiff(localContent, tmplContent, "local", newLabel),
		})
	}
	return diffs, nil
}

// readTemplateFile reads path from templateFS, reporting hasTemplate=false
// (with no error) when the path does not exist.
func readTemplateFile(templateFS fs.FS, path string) (content []byte, hasTemplate bool, err error) {
	data, err := fs.ReadFile(templateFS, path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}
