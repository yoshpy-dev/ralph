package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/scaffold"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/upgrade"
)

func newUpgradeCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Update scaffold files to the latest template version",
		Long: `Compares the current project files against the embedded templates,
auto-updates unchanged files, and prompts for conflict resolution on edited files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpgrade(".", force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "overwrite all files without prompting")

	return cmd
}

// packNamespacePrefix is the root namespace for all language pack entries in
// the manifest. Keys under this prefix are pack-scoped and must not
// participate in base-level removal detection.
const packNamespacePrefix = "packs/languages/"

// packPrefixFor returns the namespace prefix used for a specific pack's files
// in the project manifest (e.g. "packs/languages/golang/").
func packPrefixFor(pack string) string {
	return packNamespacePrefix + pack + "/"
}

// splitManifestForBase returns a manifest containing only base entries, i.e.
// those not namespaced under any language pack. This lets the base diff sweep
// detect removals without flagging every pack file as removed.
func splitManifestForBase(m *scaffold.Manifest) *scaffold.Manifest {
	out := scaffold.NewManifest(m.Meta.Version)
	out.Meta = m.Meta
	out.Files = make(map[string]scaffold.ManifestFile, len(m.Files))
	for k, v := range m.Files {
		if strings.HasPrefix(filepath.ToSlash(k), packNamespacePrefix) {
			continue
		}
		out.Files[k] = v
	}
	return out
}

// splitManifestForPack returns a manifest whose keys are stripped of the
// pack's namespace prefix, so they match the pack FS walk's relative paths.
func splitManifestForPack(m *scaffold.Manifest, pack string) *scaffold.Manifest {
	prefix := packPrefixFor(pack)
	out := scaffold.NewManifest(m.Meta.Version)
	out.Meta = m.Meta
	out.Files = make(map[string]scaffold.ManifestFile)
	for k, v := range m.Files {
		if rel, ok := strings.CutPrefix(filepath.ToSlash(k), prefix); ok {
			out.Files[rel] = v
		}
	}
	return out
}

func runUpgrade(targetDir string, force bool) error {
	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	manifestPath := filepath.Join(absDir, ".ralph", "manifest.toml")
	if _, err := os.Stat(manifestPath); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("no .ralph/manifest.toml found — run 'ralph init' first")
	}

	oldManifest, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	fmt.Printf("Checking for updates...\n")
	fmt.Printf("  Current: %s → Available: %s\n\n", oldManifest.Meta.Version, Version)

	baseFS, err := scaffold.BaseFS()
	if err != nil {
		return fmt.Errorf("loading templates: %w", err)
	}

	baseManifest := splitManifestForBase(oldManifest)
	diffs, err := upgrade.ComputeDiffsWithManifest(baseManifest, absDir, baseFS, true)
	if err != nil {
		return fmt.Errorf("computing diffs: %w", err)
	}

	installedPacks := oldManifest.Meta.Packs
	availablePacks, err := scaffold.AvailablePacks()
	if err != nil {
		return fmt.Errorf("listing available packs: %w", err)
	}
	available := make(map[string]bool, len(availablePacks))
	for _, p := range availablePacks {
		available[p] = true
	}

	// Track pack entries whose diff could not be computed so a transient
	// error does not permanently drop their tracking. Packs that have been
	// removed from the template release are explicitly NOT preserved.
	preservedPackEntries := make(map[string]scaffold.ManifestFile)
	retainedPacks := make([]string, 0, len(installedPacks))

	for _, pack := range installedPacks {
		prefix := packPrefixFor(pack)

		// Pack was removed or renamed in this release: drop manifest tracking
		// and notify the user that the on-disk files are now unmanaged.
		if !available[pack] {
			fmt.Fprintf(os.Stderr, "Notice: pack %q no longer exists in templates — manifest tracking dropped (files on disk left untouched)\n", pack)
			continue
		}

		packFS, pErr := scaffold.PackFS(pack)
		if pErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: pack %s load failed: %v (preserving manifest entries)\n", pack, pErr)
			preservePackEntries(oldManifest, prefix, preservedPackEntries)
			retainedPacks = append(retainedPacks, pack)
			continue
		}
		packDir := filepath.Join(absDir, "packs", "languages", pack)
		packManifest := splitManifestForPack(oldManifest, pack)
		// checkRemovals=true: a file dropped from the pack template but still
		// tracked in the manifest surfaces as ActionRemove (with the pack
		// prefix re-applied below) so the user still sees the "removed from
		// template" warning for genuine pack-file deletions.
		packDiffs, pErr := upgrade.ComputeDiffsWithManifest(packManifest, packDir, packFS, true)
		if pErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: pack %s diff failed: %v (preserving manifest entries)\n", pack, pErr)
			preservePackEntries(oldManifest, prefix, preservedPackEntries)
			retainedPacks = append(retainedPacks, pack)
			continue
		}
		for i := range packDiffs {
			packDiffs[i].Path = filepath.Join("packs", "languages", pack, packDiffs[i].Path)
		}
		diffs = append(diffs, packDiffs...)
		retainedPacks = append(retainedPacks, pack)
	}

	var updated, skipped, notified int

	manifest := scaffold.NewManifest(Version)
	manifest.Meta.Packs = retainedPacks

	// Carry over entries for packs we could not diff so a transient pack
	// error does not permanently drop their tracking.
	maps.Copy(manifest.Files, preservedPackEntries)

	for _, d := range diffs {
		switch d.Action {
		case upgrade.ActionAutoUpdate:
			targetPath := filepath.Join(absDir, d.Path)
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("creating parent dir for %s: %w", d.Path, err)
			}
			if err := os.WriteFile(targetPath, d.NewContent, scaffold.FilePerm(d.Path)); err != nil {
				return fmt.Errorf("writing %s: %w", d.Path, err)
			}
			manifest.SetFile(d.Path, d.NewHash)
			fmt.Printf("  ✓ %s (unchanged, auto-update)\n", d.Path)
			updated++

		case upgrade.ActionConflict:
			if force {
				targetPath := filepath.Join(absDir, d.Path)
				if err := os.WriteFile(targetPath, d.NewContent, scaffold.FilePerm(d.Path)); err != nil {
					return fmt.Errorf("writing %s: %w", d.Path, err)
				}
				manifest.SetFile(d.Path, d.NewHash)
				fmt.Printf("  ✓ %s (force overwritten)\n", d.Path)
				updated++
			} else {
				// Interactive conflict resolution.
				resolution := resolveConflict(d)
				switch resolution {
				case "overwrite":
					targetPath := filepath.Join(absDir, d.Path)
					if err := os.WriteFile(targetPath, d.NewContent, scaffold.FilePerm(d.Path)); err != nil {
						return fmt.Errorf("writing %s: %w", d.Path, err)
					}
					manifest.SetFile(d.Path, d.NewHash)
					updated++
				case "skip":
					// Preserve the OLD template hash so next upgrade still
					// detects this file as user-modified (not auto-overwritable).
					manifest.SetFile(d.Path, d.OldHash)
					fmt.Printf("  ⊘ %s (skipped)\n", d.Path)
					skipped++
				}
			}

		case upgrade.ActionAdd:
			targetPath := filepath.Join(absDir, d.Path)
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("creating parent dir for %s: %w", d.Path, err)
			}
			if err := os.WriteFile(targetPath, d.NewContent, scaffold.FilePerm(d.Path)); err != nil {
				return fmt.Errorf("writing %s: %w", d.Path, err)
			}
			manifest.SetFile(d.Path, d.NewHash)
			fmt.Printf("  + %s (new file)\n", d.Path)
			updated++

		case upgrade.ActionRemove:
			// Drop the entry from the new manifest. Preserving it caused the
			// same removal to be re-notified on every subsequent upgrade,
			// which breaks idempotency. The user was told to "review and
			// delete manually", so untracking the file after one warning is
			// the intended contract.
			fmt.Printf("  ⚠ %s (removed from template — review and delete manually)\n", d.Path)
			notified++

		case upgrade.ActionSkip:
			// Template unchanged → write the real template hash into the
			// manifest so future upgrades can keep comparing correctly.
			manifest.SetFile(d.Path, d.NewHash)
		}
	}

	if err := manifest.Write(manifestPath); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	fmt.Printf("\n  Updated: %d files\n", updated)
	fmt.Printf("  Skipped: %d files (user-modified)\n", skipped)
	if notified > 0 {
		fmt.Printf("  Removed from template: %d files (review manually)\n", notified)
	}
	fmt.Printf("  Manifest updated: .ralph/manifest.toml\n")

	return nil
}

// preservePackEntries copies all manifest entries under prefix from src into
// dst unchanged. Called when a pack's FS or diff computation fails so the
// manifest does not lose tracking of that pack's files.
func preservePackEntries(src *scaffold.Manifest, prefix string, dst map[string]scaffold.ManifestFile) {
	for k, v := range src.Files {
		if strings.HasPrefix(filepath.ToSlash(k), prefix) {
			dst[k] = v
		}
	}
}

func resolveConflict(d upgrade.FileDiff) string {
	fmt.Printf("  ⚠ %s (modified locally)\n", d.Path)
	fmt.Printf("    [o]verwrite / [s]kip / [d]iff ? ")

	var choice string
	for {
		if _, err := fmt.Scanln(&choice); err != nil {
			fmt.Fprintf(os.Stderr, "\n  (non-interactive input detected, skipping)\n")
			return "skip"
		}
		switch choice {
		case "o", "overwrite":
			return "overwrite"
		case "s", "skip":
			return "skip"
		case "d", "diff":
			// Show a simple diff indication.
			fmt.Printf("    --- ralph template (%s)\n", Version)
			fmt.Printf("    +++ local\n")
			fmt.Printf("    (template hash: %s)\n", d.NewHash)
			fmt.Printf("    (local hash:    %s)\n", d.DiskHash)
			fmt.Printf("    [o]verwrite / [s]kip ? ")
			continue
		default:
			fmt.Printf("    [o]verwrite / [s]kip / [d]iff ? ")
			continue
		}
	}
}
