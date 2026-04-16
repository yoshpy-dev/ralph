package upgrade

import (
	"io/fs"
	"path/filepath"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/scaffold"
)

// FileAction describes what should happen to a file during upgrade.
type FileAction int

const (
	ActionAutoUpdate FileAction = iota // Template unchanged by user → auto overwrite
	ActionConflict                     // Template changed by user → needs resolution
	ActionAdd                          // New file in template → add
	ActionRemove                       // File removed from template → notify user
	ActionSkip                         // User-owned file → skip
)

// FileDiff describes a single file's upgrade status.
type FileDiff struct {
	Path       string
	Action     FileAction
	OldHash    string // hash from manifest
	DiskHash   string // current hash on disk
	NewHash    string // hash from new template
	NewContent []byte // content from new template (for display/write)
}

// ComputeDiffs compares the manifest, disk state, and new template to determine actions.
// Use CheckRemovals=true only for the primary (base) FS, not for supplementary pack FSes.
func ComputeDiffs(manifestPath string, targetDir string, newFS fs.FS) ([]FileDiff, error) {
	return computeDiffsOpts(manifestPath, targetDir, newFS, true)
}

// ComputeDiffsNoRemovals is like ComputeDiffs but skips removal detection.
// Use for supplementary FS layers (language packs) where missing base files
// should not trigger removal notifications.
func ComputeDiffsNoRemovals(manifestPath string, targetDir string, newFS fs.FS) ([]FileDiff, error) {
	return computeDiffsOpts(manifestPath, targetDir, newFS, false)
}

func computeDiffsOpts(manifestPath string, targetDir string, newFS fs.FS, checkRemovals bool) ([]FileDiff, error) {
	manifest, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	var diffs []FileDiff

	// Walk new template to find adds and updates.
	newFiles := make(map[string]bool)
	err = fs.WalkDir(newFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		newFiles[path] = true

		content, err := fs.ReadFile(newFS, path)
		if err != nil {
			return err
		}
		newHash := scaffold.HashBytes(content)

		mf, inManifest := manifest.Files[path]

		if !inManifest {
			// New file not in manifest → add.
			diffs = append(diffs, FileDiff{
				Path:       path,
				Action:     ActionAdd,
				NewHash:    newHash,
				NewContent: content,
			})
			return nil
		}

		// File exists in manifest. Check disk state.
		diskPath := filepath.Join(targetDir, path)
		diskHash, diskErr := scaffold.HashFile(diskPath)

		if diskErr != nil {
			// File in manifest but missing on disk → treat as add.
			diffs = append(diffs, FileDiff{
				Path:       path,
				Action:     ActionAdd,
				NewHash:    newHash,
				NewContent: content,
			})
			return nil
		}

		// Template hasn't changed → skip regardless of user edits.
		if newHash == mf.Hash {
			diffs = append(diffs, FileDiff{
				Path:   path,
				Action: ActionSkip,
			})
			return nil
		}

		// Template changed. Did user also edit?
		if diskHash == mf.Hash {
			// User didn't edit → auto update.
			diffs = append(diffs, FileDiff{
				Path:       path,
				Action:     ActionAutoUpdate,
				OldHash:    mf.Hash,
				DiskHash:   diskHash,
				NewHash:    newHash,
				NewContent: content,
			})
		} else {
			// User edited → conflict.
			diffs = append(diffs, FileDiff{
				Path:       path,
				Action:     ActionConflict,
				OldHash:    mf.Hash,
				DiskHash:   diskHash,
				NewHash:    newHash,
				NewContent: content,
			})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Check for files in manifest that are no longer in the template → remove notification.
	// Only performed for the primary FS (base), not for supplementary pack FSes.
	if checkRemovals {
		for path := range manifest.Files {
			if !newFiles[path] {
				diffs = append(diffs, FileDiff{
					Path:    path,
					Action:  ActionRemove,
					OldHash: manifest.Files[path].Hash,
				})
			}
		}
	}

	return diffs, nil
}
