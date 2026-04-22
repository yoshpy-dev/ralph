package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
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
	return runUpgradeIO(targetDir, force, os.Stdin, os.Stdout, os.Stderr)
}

// runUpgradeIO is the testable core of the upgrade command. I/O is injected so
// integration tests can drive interactive conflict resolution without touching
// the real stdin/stdout.
func runUpgradeIO(targetDir string, force bool, in io.Reader, out, errOut io.Writer) error {
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

	writef(out, "Checking for updates...\n")
	writef(out, "  Current: %s → Available: %s\n\n", oldManifest.Meta.Version, Version)

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
	availablePacks, apErr := scaffold.AvailablePacks()

	preservedPackEntries := make(map[string]scaffold.ManifestFile)
	retainedPacks := make([]string, 0, len(installedPacks))

	if apErr != nil {
		writef(errOut, "Warning: unable to list available packs: %v (preserving installed pack entries)\n", apErr)
		for _, pack := range installedPacks {
			preservePackEntries(oldManifest, packPrefixFor(pack), preservedPackEntries)
			retainedPacks = append(retainedPacks, pack)
		}
		installedPacks = nil
	}
	available := make(map[string]bool, len(availablePacks))
	for _, p := range availablePacks {
		available[p] = true
	}

	for _, pack := range installedPacks {
		prefix := packPrefixFor(pack)

		if !available[pack] {
			writef(errOut, "Notice: pack %q no longer exists in templates — manifest tracking dropped (files on disk left untouched)\n", pack)
			continue
		}

		packFS, pErr := scaffold.PackFS(pack)
		if pErr != nil {
			writef(errOut, "Warning: pack %s load failed: %v (preserving manifest entries)\n", pack, pErr)
			preservePackEntries(oldManifest, prefix, preservedPackEntries)
			retainedPacks = append(retainedPacks, pack)
			continue
		}
		packDir := filepath.Join(absDir, "packs", "languages", pack)
		packManifest := splitManifestForPack(oldManifest, pack)
		packDiffs, pErr := upgrade.ComputeDiffsWithManifest(packManifest, packDir, packFS, true)
		if pErr != nil {
			writef(errOut, "Warning: pack %s diff failed: %v (preserving manifest entries)\n", pack, pErr)
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
	maps.Copy(manifest.Files, preservedPackEntries)

	reader := bufio.NewReader(in)

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
			writef(out, "  ✓ %s (unchanged, auto-update)\n", d.Path)
			updated++

		case upgrade.ActionConflict:
			if force {
				targetPath := filepath.Join(absDir, d.Path)
				if err := os.WriteFile(targetPath, d.NewContent, scaffold.FilePerm(d.Path)); err != nil {
					return fmt.Errorf("writing %s: %w", d.Path, err)
				}
				manifest.SetFile(d.Path, d.NewHash)
				writef(out, "  ✓ %s (force overwritten)\n", d.Path)
				updated++
				continue
			}
			switch resolveConflict(d, absDir, Version, reader, out, errOut) {
			case resolutionOverwrite:
				targetPath := filepath.Join(absDir, d.Path)
				if err := os.WriteFile(targetPath, d.NewContent, scaffold.FilePerm(d.Path)); err != nil {
					return fmt.Errorf("writing %s: %w", d.Path, err)
				}
				manifest.SetFile(d.Path, d.NewHash)
				updated++
			case resolutionSkip:
				// Mark the entry as user-owned so subsequent upgrades converge
				// to silent skip. Prefer the on-disk hash (what the user
				// actually wants kept); fall back to the recorded or new hash
				// if the disk hash is unknown.
				hash := d.DiskHash
				if hash == "" {
					if d.OldHash != "" {
						hash = d.OldHash
					} else {
						hash = d.NewHash
					}
				}
				manifest.SetFileUnmanaged(d.Path, hash)
				writef(out, "  ⊘ %s (kept local; future upgrades will skip silently)\n", d.Path)
				skipped++
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
			writef(out, "  + %s (new file)\n", d.Path)
			updated++

		case upgrade.ActionRemove:
			writef(out, "  ⚠ %s (removed from template — review and delete manually)\n", d.Path)
			notified++

		case upgrade.ActionSkip:
			// Preserve the manifest state for the path.
			// - Unmanaged + --force + template still has the file → re-adopt:
			//   overwrite the disk with template content and flip Managed=true
			//   so a single `ralph upgrade --force` restores full template
			//   coverage (matches the flag's "overwrite all files without
			//   prompting" contract).
			// - Unmanaged + no template content (e.g. the template deleted
			//   this path) → keep the entry unmanaged; force cannot re-adopt
			//   a file that no longer exists upstream.
			// - Otherwise (managed skip, heal path) → record the current
			//   template hash so future comparisons stay coherent.
			prev, hadEntry := oldManifest.Files[d.Path]
			wasUnmanaged := hadEntry && !prev.Managed
			switch {
			case force && wasUnmanaged && d.NewContent != nil:
				targetPath := filepath.Join(absDir, d.Path)
				if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
					return fmt.Errorf("creating parent dir for %s: %w", d.Path, err)
				}
				if err := os.WriteFile(targetPath, d.NewContent, scaffold.FilePerm(d.Path)); err != nil {
					return fmt.Errorf("writing %s: %w", d.Path, err)
				}
				manifest.SetFile(d.Path, d.NewHash)
				writef(out, "  ✓ %s (force re-adopted)\n", d.Path)
				updated++
			case wasUnmanaged:
				manifest.SetFileUnmanaged(d.Path, prev.Hash)
			default:
				manifest.SetFile(d.Path, d.NewHash)
			}
		}
	}

	if err := manifest.Write(manifestPath); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	writef(out, "\n  Updated: %d files\n", updated)
	writef(out, "  Skipped: %d files (user-modified)\n", skipped)
	if notified > 0 {
		writef(out, "  Removed from template: %d files (review manually)\n", notified)
	}
	writef(out, "  Manifest updated: .ralph/manifest.toml\n")

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

// writef is a best-effort write for progress text. The write destination is an
// io.Writer (for testability) so the static-analyzer cannot rule out a failing
// write the way it can for os.Stdout — silence the error explicitly here
// rather than sprinkling `_, _ =` across every call site.
func writef(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

type resolution int

const (
	resolutionSkip resolution = iota
	resolutionOverwrite
)

// resolveConflict prompts the user to pick between overwriting with the
// template content, keeping the local variant, or viewing a unified diff. EOF
// or any read error collapses to a safe skip so non-interactive runs do not
// silently overwrite edits.
func resolveConflict(d upgrade.FileDiff, absDir, version string, in *bufio.Reader, out, errOut io.Writer) resolution {
	writef(out, "  ⚠ %s (modified locally)\n", d.Path)

	for {
		writef(out, "    [o]verwrite / [s]kip / [d]iff ? ")
		line, err := in.ReadString('\n')
		if err != nil && line == "" {
			writef(errOut, "\n  (non-interactive input detected, skipping)\n")
			return resolutionSkip
		}
		switch strings.TrimSpace(line) {
		case "o", "overwrite":
			return resolutionOverwrite
		case "s", "skip":
			return resolutionSkip
		case "d", "diff":
			showDiff(d, absDir, version, out, errOut)
			// Loop back to the prompt so the user still picks overwrite or skip.
		default:
			// Unrecognized input — reprompt.
		}
	}
}

// showDiff renders the local-vs-template unified diff for a conflict entry.
// Disk read failures degrade gracefully to a hash summary so the user can
// still make an informed choice when, e.g., the file was moved between diff
// computation and the prompt.
func showDiff(d upgrade.FileDiff, absDir, version string, out, errOut io.Writer) {
	localPath := filepath.Join(absDir, d.Path)
	localBytes, err := os.ReadFile(localPath)
	if err != nil {
		writef(errOut, "    (could not read %s: %v — falling back to hash summary)\n", d.Path, err)
		writef(out, "    template hash: %s\n", d.NewHash)
		writef(out, "    local hash:    %s\n", d.DiskHash)
		return
	}
	diff := upgrade.UnifiedDiff(
		localBytes,
		d.NewContent,
		"local",
		fmt.Sprintf("template (%s)", version),
	)
	if diff == "" {
		writef(out, "    (no textual difference — manifest hash drift only)\n")
	} else {
		_, _ = io.WriteString(out, diff)
	}
	writef(out, "    template hash: %s  local hash: %s\n", d.NewHash, d.DiskHash)
}
