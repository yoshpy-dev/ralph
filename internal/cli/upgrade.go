package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
	"github.com/yoshpy-dev/ralph/internal/upgrade"
)

func newUpgradeCmd() *cobra.Command {
	var (
		force       bool
		dryRun      bool
		diffPreview bool
		pager       string
	)

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Update scaffold files to the latest template version",
		Long: `Compares the current project files against the embedded templates,
auto-updates unchanged files, and prompts for conflict resolution on edited files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if diffPreview {
				dryRun = true
			}
			return runUpgradeWithOptions(".", upgradeOptions{
				Force:       force,
				DryRun:      dryRun,
				DiffPreview: diffPreview,
				Pager:       pager,
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "overwrite all files without prompting")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview upgrade actions without writing files")
	cmd.Flags().BoolVar(&diffPreview, "diff", false, "show conflict diffs without writing files (implies --dry-run)")
	cmd.Flags().StringVar(&pager, "pager", pagerAuto, "diff pager mode: auto, always, or never")

	return cmd
}

const (
	pagerAuto   = "auto"
	pagerAlways = "always"
	pagerNever  = "never"
)

type upgradeOptions struct {
	Force       bool
	DryRun      bool
	DiffPreview bool
	Pager       string
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
	return runUpgradeWithOptions(targetDir, upgradeOptions{
		Force: force,
		Pager: pagerAuto,
	})
}

func runUpgradeWithOptions(targetDir string, opts upgradeOptions) error {
	return runUpgradeIOWithOptions(targetDir, opts, os.Stdin, os.Stdout, os.Stderr, shouldColorize(os.Stdout))
}

// shouldColorize reports whether ANSI color escapes should be emitted for the
// given output. Honors the de-facto NO_COLOR standard (https://no-color.org)
// and only enables color when writing directly to a terminal — pipes,
// redirects, and CI capture all stay plain text.
func shouldColorize(out *os.File) bool {
	if v := os.Getenv("NO_COLOR"); v != "" {
		return false
	}
	if out == nil {
		return false
	}
	return term.IsTerminal(out.Fd())
}

// runUpgradeIO is the testable core of the upgrade command. I/O is injected so
// integration tests can drive interactive conflict resolution without touching
// the real stdin/stdout. `colorize` controls whether the unified-diff render
// is wrapped in ANSI escapes — callers must decide based on the destination.
func runUpgradeIO(targetDir string, force bool, in io.Reader, out, errOut io.Writer, colorize bool) error {
	return runUpgradeIOWithOptions(targetDir, upgradeOptions{
		Force: force,
		Pager: pagerNever,
	}, in, out, errOut, colorize)
}

func runUpgradeIOWithOptions(targetDir string, opts upgradeOptions, in io.Reader, out, errOut io.Writer, colorize bool) error {
	if opts.Pager == "" {
		opts.Pager = pagerAuto
	}
	if opts.DiffPreview {
		opts.DryRun = true
	}
	if err := validatePagerMode(opts.Pager); err != nil {
		return err
	}

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

	if opts.DryRun {
		renderUpgradePreview(diffs, absDir, Version, out, errOut, colorize, opts)
		return nil
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
			if err := setManagedWithBaseline(manifest, absDir, d.Path, d.NewHash, d.NewContent); err != nil {
				return err
			}
			writef(out, "  ✓ %s (unchanged, auto-update)\n", d.Path)
			updated++

		case upgrade.ActionConflict:
			if opts.Force {
				targetPath := filepath.Join(absDir, d.Path)
				if err := os.WriteFile(targetPath, d.NewContent, scaffold.FilePerm(d.Path)); err != nil {
					return fmt.Errorf("writing %s: %w", d.Path, err)
				}
				if err := setManagedWithBaseline(manifest, absDir, d.Path, d.NewHash, d.NewContent); err != nil {
					return err
				}
				writef(out, "  ✓ %s (force overwritten)\n", d.Path)
				updated++
				continue
			}
			switch resolveConflict(d, oldManifest, absDir, Version, reader, out, errOut, colorize, opts.Pager) {
			case resolutionOverwrite:
				targetPath := filepath.Join(absDir, d.Path)
				if err := os.WriteFile(targetPath, d.NewContent, scaffold.FilePerm(d.Path)); err != nil {
					return fmt.Errorf("writing %s: %w", d.Path, err)
				}
				if err := setManagedWithBaseline(manifest, absDir, d.Path, d.NewHash, d.NewContent); err != nil {
					return err
				}
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
			if err := setManagedWithBaseline(manifest, absDir, d.Path, d.NewHash, d.NewContent); err != nil {
				return err
			}
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
			case opts.Force && wasUnmanaged && d.NewContent != nil:
				targetPath := filepath.Join(absDir, d.Path)
				if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
					return fmt.Errorf("creating parent dir for %s: %w", d.Path, err)
				}
				if err := os.WriteFile(targetPath, d.NewContent, scaffold.FilePerm(d.Path)); err != nil {
					return fmt.Errorf("writing %s: %w", d.Path, err)
				}
				if err := setManagedWithBaseline(manifest, absDir, d.Path, d.NewHash, d.NewContent); err != nil {
					return err
				}
				writef(out, "  ✓ %s (force re-adopted)\n", d.Path)
				updated++
			case wasUnmanaged:
				preserveUnmanaged(manifest, d.Path, prev)
			default:
				if err := preserveManagedSkip(manifest, oldManifest, absDir, d); err != nil {
					return err
				}
			}
		}
	}

	if err := manifest.Write(manifestPath); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	installManagedGitHooks(absDir, out, errOut)

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

func setManagedWithBaseline(manifest *scaffold.Manifest, absDir, relPath, hash string, content []byte) error {
	baselinePath, err := scaffold.WriteBaseline(absDir, relPath, content)
	if err != nil {
		return err
	}
	manifest.SetFileWithBaseline(relPath, hash, baselinePath)
	return nil
}

func preserveUnmanaged(manifest *scaffold.Manifest, relPath string, prev scaffold.ManifestFile) {
	if prev.State == "" {
		prev.State = scaffold.FileStateUnmanaged
	}
	if prev.BaselineStatus == "" {
		prev.BaselineStatus = scaffold.BaselineStatusMissing
	}
	manifest.Files[relPath] = prev
}

func preserveManagedSkip(manifest, oldManifest *scaffold.Manifest, absDir string, d upgrade.FileDiff) error {
	if prev, ok := oldManifest.Files[d.Path]; ok {
		prev = prev.WithTemplateHash(d.NewHash)
		if _, err := scaffold.ReadBaseline(absDir, prev); err == nil {
			manifest.Files[d.Path] = prev
			return nil
		}
	}
	if d.NewContent != nil && d.DiskHash == d.NewHash {
		return setManagedWithBaseline(manifest, absDir, d.Path, d.NewHash, d.NewContent)
	}
	manifest.SetFile(d.Path, d.NewHash)
	return nil
}

func validatePagerMode(mode string) error {
	switch mode {
	case pagerAuto, pagerAlways, pagerNever:
		return nil
	default:
		return fmt.Errorf("invalid --pager %q (want auto, always, or never)", mode)
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
func resolveConflict(d upgrade.FileDiff, oldManifest *scaffold.Manifest, absDir, version string, in *bufio.Reader, out, errOut io.Writer, colorize bool, pagerMode string) resolution {
	writef(out, "  ⚠ %s (modified locally)\n", d.Path)

	if hasReadableBaseline(oldManifest, absDir, d) {
		return resolveConflictWithBaseline(d, absDir, version, in, out, errOut, colorize, pagerMode)
	}

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
			showDiff(d, absDir, version, out, errOut, colorize, pagerMode)
			// Loop back to the prompt so the user still picks overwrite or skip.
		default:
			// Unrecognized input — reprompt.
		}
	}
}

// showDiff renders the local-vs-template unified diff for a conflict entry.
// When colorize is true the diff is wrapped in ANSI escapes for terminal
// display. Disk read failures degrade gracefully to a hash summary so the
// user can still make an informed choice when, e.g., the file was moved
// between diff computation and the prompt.
func showDiff(d upgrade.FileDiff, absDir, version string, out, errOut io.Writer, colorize bool, pagerMode string) {
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
		if colorize {
			diff = upgrade.Colorize(diff)
		}
		writeDiffOutput(diff, out, errOut, pagerMode)
	}
	writef(out, "    template hash: %s  local hash: %s\n", d.NewHash, d.DiskHash)
}

func hasReadableBaseline(oldManifest *scaffold.Manifest, absDir string, d upgrade.FileDiff) bool {
	prev, ok := oldManifest.Files[d.Path]
	if !ok || !prev.IsBaselineAvailable() {
		return false
	}
	if _, err := scaffold.ReadBaseline(absDir, prev); err != nil {
		return false
	}
	return true
}

func resolveConflictWithBaseline(d upgrade.FileDiff, absDir, version string, in *bufio.Reader, out, errOut io.Writer, colorize bool, pagerMode string) resolution {
	showDiff(d, absDir, version, out, errOut, colorize, pagerMode)
	for {
		writef(out, "    [a]pply template file / [k]eep local file / [e]dit / [s]kip file ? ")
		line, err := in.ReadString('\n')
		if err != nil && line == "" {
			writef(errOut, "\n  (non-interactive input detected, skipping)\n")
			return resolutionSkip
		}
		switch strings.TrimSpace(line) {
		case "a", "apply":
			return resolutionOverwrite
		case "k", "keep", "s", "skip":
			return resolutionSkip
		case "e", "edit":
			writef(errOut, "    (manual edit is not available yet; choose apply, keep, or skip file)\n")
		default:
			// Unrecognized input — reprompt.
		}
	}
}

func renderUpgradePreview(diffs []upgrade.FileDiff, absDir, version string, out, errOut io.Writer, colorize bool, opts upgradeOptions) {
	var autoUpdate, conflict, add, remove, skip int
	for _, d := range diffs {
		switch d.Action {
		case upgrade.ActionAutoUpdate:
			autoUpdate++
		case upgrade.ActionConflict:
			conflict++
		case upgrade.ActionAdd:
			add++
		case upgrade.ActionRemove:
			remove++
		case upgrade.ActionSkip:
			skip++
		}
	}

	writef(out, "Upgrade preview (dry run)\n")
	writef(out, "  auto-update: %d files\n", autoUpdate)
	writef(out, "  conflict:    %d files\n", conflict)
	writef(out, "  add:         %d files\n", add)
	writef(out, "  remove:      %d files\n", remove)
	writef(out, "  skip:        %d files\n", skip)

	if len(diffs) == 0 {
		writef(out, "\nNo changes.\n")
		return
	}

	writef(out, "\nFiles:\n")
	for _, d := range diffs {
		writef(out, "  %-11s %s\n", actionLabel(d.Action), d.Path)
	}

	if !opts.DiffPreview {
		return
	}

	for _, d := range diffs {
		if d.Action != upgrade.ActionConflict && d.Action != upgrade.ActionAutoUpdate {
			continue
		}
		writef(out, "\n--- %s ---\n", d.Path)
		showDiff(d, absDir, version, out, errOut, colorize, opts.Pager)
	}
}

func actionLabel(action upgrade.FileAction) string {
	switch action {
	case upgrade.ActionAutoUpdate:
		return "auto-update"
	case upgrade.ActionConflict:
		return "conflict"
	case upgrade.ActionAdd:
		return "add"
	case upgrade.ActionRemove:
		return "remove"
	case upgrade.ActionSkip:
		return "skip"
	default:
		return "unknown"
	}
}

func writeDiffOutput(diff string, out, errOut io.Writer, pagerMode string) {
	if !shouldUsePager(pagerMode, out) {
		_, _ = io.WriteString(out, diff)
		return
	}
	if err := writeThroughPager(diff, out, errOut); err != nil {
		writef(errOut, "    (pager failed: %v — writing diff directly)\n", err)
		_, _ = io.WriteString(out, diff)
	}
}

func shouldUsePager(pagerMode string, out io.Writer) bool {
	switch pagerMode {
	case pagerNever:
		return false
	case pagerAlways:
		return true
	case pagerAuto:
		f, ok := out.(*os.File)
		return ok && term.IsTerminal(f.Fd())
	default:
		return false
	}
}

func writeThroughPager(diff string, out, errOut io.Writer) error {
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less -R"
	}
	parts := strings.Fields(pager)
	if len(parts) == 0 {
		return fmt.Errorf("empty pager")
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = strings.NewReader(diff)
	cmd.Stdout = out
	cmd.Stderr = errOut
	return cmd.Run()
}
