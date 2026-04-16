package scaffold

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// RenderOptions controls how files are expanded.
type RenderOptions struct {
	// TargetDir is the root directory to write files to.
	TargetDir string
	// Overwrite controls whether existing files are overwritten.
	Overwrite bool
}

// RenderResult tracks what happened during rendering.
type RenderResult struct {
	Created     []string
	Overwritten []string
	Skipped     []string
}

// RenderFS walks the given filesystem and writes files to the target directory.
// Returns a map of relative paths to their SHA256 hashes.
func RenderFS(src fs.FS, opts RenderOptions) (*RenderResult, map[string]string, error) {
	result := &RenderResult{}
	hashes := make(map[string]string)

	absTarget, err := filepath.Abs(opts.TargetDir)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving target dir: %w", err)
	}

	err = fs.WalkDir(src, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		target := filepath.Join(opts.TargetDir, path)

		// Boundary check: ensure target does not escape TargetDir.
		absFile, absErr := filepath.Abs(target)
		if absErr != nil {
			return fmt.Errorf("resolving path %s: %w", path, absErr)
		}
		if !strings.HasPrefix(absFile, absTarget+string(filepath.Separator)) && absFile != absTarget {
			return fmt.Errorf("template path %q escapes target directory", path)
		}

		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}

		// Read source content.
		content, err := fs.ReadFile(src, path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		hash := HashBytes(content)
		hashes[path] = hash

		// Check if target already exists.
		if _, statErr := os.Stat(target); statErr == nil {
			if !opts.Overwrite {
				result.Skipped = append(result.Skipped, path)
				return nil
			}
			result.Overwritten = append(result.Overwritten, path)
		} else {
			result.Created = append(result.Created, path)
		}

		// Ensure parent directory exists.
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("creating parent dir for %s: %w", path, err)
		}

		// Determine file permission. go:embed returns 0444 for all files,
		// so we cannot rely on FS metadata for execute bits. Instead, use
		// suffix and name heuristics to identify executable scripts.
		perm := filePerm(path)

		return os.WriteFile(target, content, perm)
	})

	return result, hashes, err
}

// FilePerm returns the appropriate file permission for the given path.
// Shell scripts (.sh suffix) and the extensionless "ralph" script in
// scripts/ get 0755; all others get 0644.
// Note: go:embed returns 0444 for all files, so FS metadata cannot be used.
func FilePerm(path string) fs.FileMode {
	if strings.HasSuffix(path, ".sh") {
		return 0755
	}
	if filepath.Base(path) == "ralph" && strings.Contains(path, "scripts/") {
		return 0755
	}
	return 0644
}

// filePerm is an unexported alias for internal use in this package.
func filePerm(path string) fs.FileMode { return FilePerm(path) }

// HashBytes returns the SHA256 hash of data as "sha256:<hex>".
func HashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", h)
}

// HashFile returns the SHA256 hash of a file on disk.
func HashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}
