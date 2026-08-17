package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
)

func newPackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pack",
		Short: "Manage language packs",
	}

	cmd.AddCommand(newPackAddCmd())
	cmd.AddCommand(newPackListCmd())

	return cmd
}

func newPackAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <language>",
		Short: "Add a language pack to the project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return addPack(".", args[0])
		},
	}
}

func newPackListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available language packs",
		RunE: func(cmd *cobra.Command, args []string) error {
			packs, err := scaffold.AvailablePacks()
			if err != nil {
				return err
			}
			fmt.Println("Available language packs:")
			for _, p := range packs {
				fmt.Printf("  - %s\n", p)
			}
			return nil
		},
	}
}

// addPack adds a language pack to an existing project rooted at targetDir.
// Pack payload files are written to packs/languages/<lang>/ and the rule.md
// control file is mapped to .claude/rules/ralph/<lang>.md (matching init.go's layout).
// The shared renderPackInto helper (language_pack.go) is used here so this
// path cannot diverge from ralph init's pack rendering.
func addPack(targetDir string, lang string) error {
	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return err
	}

	// renderPackInto handles directory layout, rule.md mapping, and baseline
	// writes — identical to what executeInit does for each pack.
	pr, err := renderPackInto(absDir, lang, true /* overwrite existing files */)
	if err != nil {
		return err
	}

	// Update manifest: read existing, merge pack entries, write back.
	manifestPath := filepath.Join(absDir, ".ralph", "manifest.toml")
	manifest, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		fmt.Printf("⚠ Could not update manifest: %v\n", err)
	} else {
		for path, hash := range pr.hashes {
			if baselinePath, ok := pr.baselinePaths[path]; ok {
				manifest.SetFileWithBaseline(path, hash, baselinePath)
			} else {
				manifest.SetFile(path, hash)
			}
		}
		// v2-layout manifests track ownership per entry; classification is
		// shared with ralph init via ownerForScaffoldPath (init.go) rather
		// than mirrored, so the two entry points cannot diverge on a future
		// pack payload path (e.g. under docs/ or .ralph/local/). Legacy
		// (pre-v2) manifests have no ownership model, so entries stay
		// ownerless as before.
		if manifest.Meta.Layout == scaffold.LayoutV2 {
			for path := range pr.hashes {
				if err := manifest.SetOwner(path, ownerForScaffoldPath(path)); err != nil {
					// SetOwner only fails for an invalid owner (impossible —
					// ownerForScaffoldPath always returns a valid constant)
					// or a manifest entry missing for path (impossible here
					// — the loop just called SetFile/SetFileWithBaseline for
					// every key in pr.hashes). Kept as a defensive guard,
					// matching the surrounding ReadManifest/Write style.
					fmt.Printf("⚠ Could not set owner for %s: %v\n", path, err)
				}
			}
		}
		// Record the pack in Meta.Packs if not already present.
		alreadyListed := false
		for _, p := range manifest.Meta.Packs {
			if p == lang {
				alreadyListed = true
				break
			}
		}
		if !alreadyListed {
			manifest.Meta.Packs = append(manifest.Meta.Packs, lang)
		}
		if err := manifest.Write(manifestPath); err != nil {
			fmt.Printf("⚠ Could not write manifest: %v\n", err)
		}
	}

	created := len(pr.result.Created)
	updated := len(pr.result.Overwritten)
	fmt.Printf("✓ Pack %s added (%d created, %d updated)\n", lang, created, updated)

	return nil
}
