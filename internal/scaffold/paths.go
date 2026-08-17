package scaffold

import (
	"fmt"
	"path/filepath"
)

// CleanLocalRelPath validates and normalizes a manifest-relative path shared
// across scaffold and upgrade primitives. It rejects empty paths, paths that
// escape the local tree (e.g. "..", absolute paths), and paths that resolve
// to the root itself (".").
func CleanLocalRelPath(relPath string) (string, error) {
	if relPath == "" {
		return "", fmt.Errorf("empty relative path")
	}
	fromSlash := filepath.FromSlash(relPath)
	if !filepath.IsLocal(fromSlash) {
		return "", fmt.Errorf("path %q is not local", relPath)
	}
	clean := filepath.Clean(fromSlash)
	if clean == "." {
		return "", fmt.Errorf("path %q is not a file path", relPath)
	}
	return clean, nil
}
