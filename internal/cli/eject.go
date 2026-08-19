package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
	"github.com/yoshpy-dev/ralph/internal/upgrade"
)

// newEjectCmd wires `ralph eject <path>` (spec FR-2): converts a core-owned
// (or unresolved-drift core) manifest path into a user-owned fork. Zero disk
// writes — only the manifest gains a fork record (owner=fork,
// forked_from_version, disk-content hash). Once ejected, `ralph upgrade`
// leaves the path alone and surfaces it as an advisory diff instead of
// replacing or drift-flagging it (FR-4's "eject" resolution route).
func newEjectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "eject <path>",
		Short: "Convert a core-owned scaffold path into a user-owned fork",
		Long: `Records <path> as owner=fork in .ralph/manifest.toml, capturing its
current disk content hash and the manifest's recorded template version. This
is a manifest-only write: the file on disk is never touched.

Once a path is ejected, ` + "`ralph upgrade`" + ` never replaces it again — it reports
an advisory diff (see the upgrade report) instead. Eject also works on a
path that is already drifted (disk content diverges from both the recorded
and current template hash, with no fork record): this is one of the two
ways to resolve unresolved drift (the other is ` + "`ralph adopt`" + `, which discards
the drift instead of keeping it).

<path> must be a manifest-tracked, owner=core path outside the v2 exception
faces (.claude/settings.json, its merge snapshot, and the two managed-block
surfaces AGENTS.md / .gitignore) — those are rewritten by dedicated
mechanisms that a fork record cannot protect against.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEjectIO(".", args[0], cmd.OutOrStdout())
		},
	}
}

// runEjectIO is the testable core of `ralph eject`. targetDir is resolved to
// an absolute path so relative CLI invocations behave the same as tests that
// pass a temp-dir path directly.
func runEjectIO(targetDir, rawPath string, out io.Writer) error {
	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return err
	}

	manifest, manifestPath, err := requireV2ManifestForOwnership(absDir)
	if err != nil {
		return err
	}

	path, err := cleanMigrationPath(rawPath)
	if err != nil {
		return fmt.Errorf("%s: %w", rawPath, err)
	}

	if v2SkipPaths()[path] {
		return v2ExceptionFaceError(path)
	}

	entry, tracked := manifest.Files[path]
	if !tracked {
		return fmt.Errorf("%s: not tracked in the manifest; nothing to eject", path)
	}

	switch entry.Owner {
	case scaffold.OwnerFork:
		return fmt.Errorf("%s: already owner=fork; nothing to eject", path)
	case scaffold.OwnerSeed:
		return fmt.Errorf("%s: owner=seed paths are already user-owned once created; eject only applies to owner=core paths", path)
	case scaffold.OwnerBlock:
		return fmt.Errorf("%s: owner=block is managed by the block engine, not eject; customize content outside the ralph-managed block markers, or use .ralph/local/ for downstream extensions", path)
	case scaffold.OwnerCore:
		// Continue below — this is eject's only legal target (clean or
		// drifted core).
	default:
		return fmt.Errorf("%s: no recorded ownership (legacy manifest entry); eject only applies to owner=core paths — run `ralph upgrade` to migrate a legacy manifest first", path)
	}

	full := filepath.Join(absDir, filepath.FromSlash(path))
	diskContent, err := os.ReadFile(full)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s: missing on disk; nothing to eject", path)
		}
		return fmt.Errorf("%s: reading disk file: %w", path, err)
	}

	diskHash := scaffold.HashBytes(diskContent)
	forkedFrom := manifest.Meta.Version
	manifest.SetFileFork(path, diskHash, forkedFrom)
	if err := manifest.Write(manifestPath); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	writef(out, "Ejected %s: owner=core -> owner=fork (forked_from_version=%s).\n", path, forkedFrom)
	writef(out, "  `ralph upgrade` will now report this path as an advisory diff instead of replacing it.\n")
	return nil
}

// requireV2ManifestForOwnership reads .ralph/manifest.toml under absDir and
// enforces the v2-layout barrier eject/adopt share with `ralph pack add`
// (addPack, pack.go): a legacy (pre-v2) manifest is refused fail-closed
// (zero writes), pointing the operator at `ralph upgrade`'s migration path,
// rather than eject/adopt attempting to guess at legacy ownership semantics
// the core replace planner was never designed to classify.
func requireV2ManifestForOwnership(absDir string) (*scaffold.Manifest, string, error) {
	manifestPath := filepath.Join(absDir, ".ralph", "manifest.toml")
	manifest, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		return nil, "", fmt.Errorf("reading manifest: %w", err)
	}
	if manifest.Meta.Layout != scaffold.LayoutV2 {
		return nil, "", errLegacyLayoutFailClosed
	}
	return manifest, manifestPath, nil
}

// v2ExceptionFaceError builds a rejection error for a v2SkipPaths() member:
// these paths are rewritten by a dedicated mechanism outside the core
// replace planner (settings 3-way merge, or the managed-block engine), so a
// fork record recorded by eject could never actually protect them — eject
// and adopt both refuse them outright rather than promise a protection they
// cannot deliver (plan's Codex finding 1). Each branch points at that face's
// real customization channel.
func v2ExceptionFaceError(path string) error {
	switch path {
	case v2SettingsPath:
		return fmt.Errorf("%s: v2 exception face handled by the settings 3-way merge, not the core replace planner; it is user-editable directly — edit it in place instead of ejecting/adopting it", path)
	case upgrade.SettingsSnapshotRelPath:
		return fmt.Errorf("%s: v2 exception face — an internal merge-baseline snapshot maintained automatically by `ralph upgrade`; customize %s directly instead, or use .ralph/local/ for downstream extensions", path, v2SettingsPath)
	default:
		for _, bs := range blockSurfaces {
			if bs.path == path {
				return fmt.Errorf("%s: v2 exception face managed by the block engine, not the core replace planner; customize content outside the ralph-managed block markers, or use .ralph/local/ for downstream extensions", path)
			}
		}
		return fmt.Errorf("%s: v2 exception face handled by a dedicated mechanism outside the core replace planner; eject/adopt do not apply", path)
	}
}
