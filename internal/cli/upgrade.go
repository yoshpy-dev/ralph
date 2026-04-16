package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

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

	diffs, err := upgrade.ComputeDiffs(manifestPath, absDir, baseFS)
	if err != nil {
		return fmt.Errorf("computing diffs: %w", err)
	}

	// Only diff packs that were installed (recorded in manifest metadata).
	installedPacks := oldManifest.Meta.Packs
	for _, pack := range installedPacks {
		packFS, pErr := scaffold.PackFS(pack)
		if pErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: pack %s: %v\n", pack, pErr)
			continue
		}
		packDir := filepath.Join(absDir, "packs", "languages", pack)
		packManifestPath := manifestPath // same manifest, namespaced paths
		packDiffs, pErr := upgrade.ComputeDiffsNoRemovals(packManifestPath, packDir, packFS)
		if pErr != nil {
			continue
		}
		// Namespace pack diff paths under packs/languages/<pack>/.
		packPrefix := filepath.Join("packs", "languages", pack)
		for i := range packDiffs {
			packDiffs[i].Path = filepath.Join(packPrefix, packDiffs[i].Path)
		}
		diffs = append(diffs, packDiffs...)
	}

	var updated, skipped, notified int

	manifest := scaffold.NewManifest(Version)
	manifest.Meta.Packs = installedPacks

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
			// Preserve the old manifest entry so next upgrade doesn't re-notify.
			manifest.SetFile(d.Path, d.OldHash)
			fmt.Printf("  ⚠ %s (removed from template — review and delete manually)\n", d.Path)
			notified++

		case upgrade.ActionSkip:
			// Template unchanged → keep current manifest entry.
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
