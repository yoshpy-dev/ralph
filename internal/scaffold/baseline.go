package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const baselineRoot = ".ralph/baseline"

// BaselinePath returns the manifest-relative baseline cache path for a
// template-managed relative path.
func BaselinePath(relPath string) (string, error) {
	clean, err := CleanLocalRelPath(relPath)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join(baselineRoot, clean)), nil
}

// WriteBaseline stores template content in the baseline cache and returns the
// manifest-relative path. The cache mirrors template-relative paths under
// .ralph/baseline so humans can inspect it when needed.
func WriteBaseline(targetDir, relPath string, content []byte) (string, error) {
	baselinePath, err := BaselinePath(relPath)
	if err != nil {
		return "", err
	}
	absPath := filepath.Join(targetDir, filepath.FromSlash(baselinePath))
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return "", fmt.Errorf("creating baseline parent for %s: %w", relPath, err)
	}
	if err := os.WriteFile(absPath, content, FilePerm(relPath)); err != nil {
		return "", fmt.Errorf("writing baseline for %s: %w", relPath, err)
	}
	return baselinePath, nil
}

// ReadBaseline reads the baseline content referenced by a manifest entry.
func ReadBaseline(targetDir string, entry ManifestFile) ([]byte, error) {
	if !entry.IsBaselineAvailable() {
		return nil, fmt.Errorf("baseline unavailable")
	}
	clean, err := CleanLocalRelPath(entry.BaselinePath)
	if err != nil {
		return nil, err
	}
	cleanSlash := filepath.ToSlash(clean)
	if cleanSlash == baselineRoot || !strings.HasPrefix(cleanSlash, baselineRoot+"/") {
		return nil, fmt.Errorf("baseline path %q is outside %s", entry.BaselinePath, baselineRoot)
	}
	path := filepath.Join(targetDir, filepath.FromSlash(entry.BaselinePath))
	return os.ReadFile(path)
}
