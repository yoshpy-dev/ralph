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
	manifest, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	return ComputeDiffsWithManifest(manifest, targetDir, newFS, true)
}

// ComputeDiffsNoRemovals is like ComputeDiffs but skips removal detection.
// Use for supplementary FS layers (language packs) where missing base files
// should not trigger removal notifications.
func ComputeDiffsNoRemovals(manifestPath string, targetDir string, newFS fs.FS) ([]FileDiff, error) {
	manifest, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	return ComputeDiffsWithManifest(manifest, targetDir, newFS, false)
}

// ComputeDiffsWithManifest compares a pre-parsed manifest, disk state, and new
// template to determine actions. The caller can provide a scoped subset of the
// full manifest (e.g. base entries only, or pack-scoped entries with the pack
// prefix stripped) so that removal detection and key lookups operate over the
// correct namespace.
func ComputeDiffsWithManifest(manifest *scaffold.Manifest, targetDir string, newFS fs.FS, checkRemovals bool) ([]FileDiff, error) {
	var diffs []FileDiff

	// Walk new template to find adds and updates.
	newFiles := make(map[string]bool)
	err := fs.WalkDir(newFS, ".", func(path string, d fs.DirEntry, err error) error {
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

		// Peek at disk state up front so the ActionAdd path can distinguish
		// a safe add (disk missing or already matches template) from a
		// potentially overwriting add (disk has different content — e.g. a
		// file that was previously removed from the template, kept locally
		// by the user, and now reintroduced in a later release).
		diskPath := filepath.Join(targetDir, path)
		diskHash, diskErr := scaffold.HashFile(diskPath)

		// User-accepted local variant: a prior `ralph upgrade` run recorded
		// Managed=false for this path (the user chose skip on a conflict).
		// Respect that ownership and silent-skip regardless of disk/template
		// drift until an explicit resync (or `--force`) brings the file back
		// under template management. NewContent is carried so the caller can
		// implement re-adoption without a second FS walk.
		if inManifest && !mf.Managed {
			diffs = append(diffs, FileDiff{
				Path:       path,
				Action:     ActionSkip,
				OldHash:    mf.Hash,
				DiskHash:   diskHash,
				NewHash:    newHash,
				NewContent: content,
			})
			return nil
		}

		if !inManifest {
			// New file not in manifest. If disk has something different,
			// surface as a conflict so the user is asked rather than
			// silently overwritten. OldHash=newHash mirrors the empty-hash
			// heal contract: skip writes the template hash into the
			// manifest so the next upgrade resolves cleanly.
			if diskErr == nil && diskHash != newHash {
				diffs = append(diffs, FileDiff{
					Path:       path,
					Action:     ActionConflict,
					OldHash:    newHash,
					DiskHash:   diskHash,
					NewHash:    newHash,
					NewContent: content,
				})
				return nil
			}
			diffs = append(diffs, FileDiff{
				Path:       path,
				Action:     ActionAdd,
				NewHash:    newHash,
				NewContent: content,
			})
			return nil
		}

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

		// Heal corrupted manifest entries where hash is empty (caused by a
		// prior bug that wrote ActionSkip entries without a hash). Treat
		// "disk matches new template" as equivalent to an unchanged file and
		// silently repair the manifest hash. If disk differs, fall through to
		// the conflict path so the user is still asked.
		if mf.Hash == "" {
			if diskHash == newHash {
				diffs = append(diffs, FileDiff{
					Path:     path,
					Action:   ActionSkip,
					OldHash:  mf.Hash,
					DiskHash: diskHash,
					NewHash:  newHash,
				})
				return nil
			}
			// Empty-hash heal + user edit: use newHash as OldHash so the
			// "skip" resolution path rewrites the manifest with a real
			// hash, ending the perpetual-conflict loop on non-interactive
			// re-runs. If the user overwrites, they accept the template;
			// if they skip, we mark the template as the new baseline and
			// let future upgrades detect edits against it.
			diffs = append(diffs, FileDiff{
				Path:       path,
				Action:     ActionConflict,
				OldHash:    newHash,
				DiskHash:   diskHash,
				NewHash:    newHash,
				NewContent: content,
			})
			return nil
		}

		// Template hasn't changed. If disk matches the recorded hash, the file
		// is untouched → skip. If disk drifted from the recorded hash, the
		// user edited a managed file locally: surface it as a conflict so the
		// user can choose overwrite / skip / diff. Skip resolution writes
		// Managed=false, converging subsequent upgrades to silent skip.
		if newHash == mf.Hash {
			if diskHash == mf.Hash {
				diffs = append(diffs, FileDiff{
					Path:     path,
					Action:   ActionSkip,
					OldHash:  mf.Hash,
					DiskHash: diskHash,
					NewHash:  newHash,
				})
				return nil
			}
			diffs = append(diffs, FileDiff{
				Path:       path,
				Action:     ActionConflict,
				OldHash:    mf.Hash,
				DiskHash:   diskHash,
				NewHash:    newHash,
				NewContent: content,
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

	// Check for files in manifest that are no longer in the template.
	// Only performed for the primary FS (base), not for supplementary pack FSes.
	// Managed=false entries are preserved as ActionSkip across template
	// removals so the "user-owned forever until resync" contract survives
	// arbitrary template changes (including path deletions).
	if checkRemovals {
		for path, mf := range manifest.Files {
			if newFiles[path] {
				continue
			}
			if !mf.Managed {
				diffs = append(diffs, FileDiff{
					Path:    path,
					Action:  ActionSkip,
					OldHash: mf.Hash,
				})
				continue
			}
			diffs = append(diffs, FileDiff{
				Path:    path,
				Action:  ActionRemove,
				OldHash: mf.Hash,
			})
		}
	}

	return diffs, nil
}
