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
	cmd.Flags().StringVar(&pager, "pager", pagerAuto, "dry-run diff pager mode: auto, always, or never")

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
	packRulePaths := make(map[string]bool, len(m.Meta.Packs))
	for _, pack := range m.Meta.Packs {
		packRulePaths[filepath.ToSlash(packRuleRelPath(pack))] = true
	}
	for k, v := range m.Files {
		slashPath := filepath.ToSlash(k)
		if strings.HasPrefix(slashPath, packNamespacePrefix) || packRulePaths[slashPath] {
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
			preservePackState(oldManifest, pack, preservedPackEntries)
			retainedPacks = append(retainedPacks, pack)
		}
		installedPacks = nil
	}
	available := make(map[string]bool, len(availablePacks))
	for _, p := range availablePacks {
		available[p] = true
	}

	for _, pack := range installedPacks {
		if !available[pack] {
			writef(errOut, "Notice: pack %q no longer exists in templates — manifest tracking dropped (files on disk left untouched)\n", pack)
			continue
		}

		packFS, pErr := scaffold.PackFS(pack)
		if pErr != nil {
			writef(errOut, "Warning: pack %s load failed: %v (preserving manifest entries)\n", pack, pErr)
			preservePackState(oldManifest, pack, preservedPackEntries)
			retainedPacks = append(retainedPacks, pack)
			continue
		}
		packDir := filepath.Join(absDir, packRelDir(pack))
		packManifest := splitManifestForPack(oldManifest, pack)
		packDiffs, pErr := upgrade.ComputeDiffsWithManifestOptions(packManifest, packDir, packFS, upgrade.DiffOptions{
			CheckRemovals: true,
			SkipPaths:     packRenderSkipPaths,
		})
		if pErr != nil {
			writef(errOut, "Warning: pack %s diff failed: %v (preserving manifest entries)\n", pack, pErr)
			preservePackState(oldManifest, pack, preservedPackEntries)
			retainedPacks = append(retainedPacks, pack)
			continue
		}
		for i := range packDiffs {
			packDiffs[i].Path = filepath.Join(packRelDir(pack), packDiffs[i].Path)
		}

		ruleContent, ok, pErr := packRuleContent(packFS)
		if pErr != nil {
			writef(errOut, "Warning: pack %s rule diff failed: %v (preserving manifest entries)\n", pack, pErr)
			preservePackState(oldManifest, pack, preservedPackEntries)
			retainedPacks = append(retainedPacks, pack)
			continue
		}
		if ok {
			rulePath := packRuleRelPath(pack)
			packDiffs = append(packDiffs, upgrade.ComputeFileDiff(oldManifest, absDir, rulePath, ruleContent))
		}
		diffs = append(diffs, packDiffs...)
		retainedPacks = append(retainedPacks, pack)
	}

	if opts.DryRun {
		renderUpgradePreview(diffs, absDir, Version, out, errOut, colorize, opts)
		return nil
	}

	manifest := scaffold.NewManifest(Version)
	manifest.Meta.Packs = retainedPacks
	maps.Copy(manifest.Files, preservedPackEntries)

	reader := bufio.NewReader(in)
	applyPlan := &upgradeApplyPlan{}

	for _, d := range diffs {
		switch d.Action {
		case upgrade.ActionAutoUpdate:
			if err := applyPlan.addTemplateWrite(manifest, d.Path, d.NewHash, d.NewContent, fmt.Sprintf("  ✓ %s (unchanged, auto-update)\n", d.Path)); err != nil {
				return err
			}

		case upgrade.ActionConflict:
			if opts.Force {
				if err := applyPlan.addTemplateWrite(manifest, d.Path, d.NewHash, d.NewContent, fmt.Sprintf("  ✓ %s (force overwritten)\n", d.Path)); err != nil {
					return err
				}
				continue
			}
			result := resolveConflict(d, oldManifest, absDir, Version, reader, out, errOut, colorize)
			applyPlan.mergeConflictStats(result.stats)
			if result.conflictReviewed {
				applyPlan.needsConfirmation = true
			}
			switch result.kind {
			case resolutionOverwrite:
				if err := applyPlan.addTemplateWrite(manifest, d.Path, d.NewHash, d.NewContent, ""); err != nil {
					return err
				}
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
				applyPlan.addMessage(fmt.Sprintf("  ⊘ %s (kept local; future upgrades will skip silently)\n", d.Path))
				applyPlan.skipped++
			case resolutionResolved:
				if result.message != "" {
					if err := applyPlan.addResolvedWriteWithMessage(manifest, d.Path, d.NewHash, d.NewContent, result.content, result.message); err != nil {
						return err
					}
				} else {
					if err := applyPlan.addResolvedWrite(manifest, d.Path, d.NewHash, d.NewContent, result.content); err != nil {
						return err
					}
				}
			}

		case upgrade.ActionAdd:
			if err := applyPlan.addTemplateWrite(manifest, d.Path, d.NewHash, d.NewContent, fmt.Sprintf("  + %s (new file)\n", d.Path)); err != nil {
				return err
			}

		case upgrade.ActionRemove:
			applyPlan.addMessage(fmt.Sprintf("  ⚠ %s (removed from template — review and delete manually)\n", d.Path))
			applyPlan.notified++

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
				if err := applyPlan.addTemplateWrite(manifest, d.Path, d.NewHash, d.NewContent, fmt.Sprintf("  ✓ %s (force re-adopted)\n", d.Path)); err != nil {
					return err
				}
			case wasUnmanaged:
				preserveUnmanaged(manifest, d.Path, prev)
			default:
				if err := preserveManagedSkip(manifest, oldManifest, absDir, applyPlan, d); err != nil {
					return err
				}
			}
		}
	}

	if applyPlan.needsConfirmation && !confirmApplySummary(applyPlan, reader, out, errOut) {
		writef(out, "\nNo changes applied.\n")
		return nil
	}

	if err := applyPlan.apply(absDir); err != nil {
		return err
	}
	applyPlan.writeMessages(out)

	if err := manifest.Write(manifestPath); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	installManagedGitHooks(absDir, out, errOut)

	writef(out, "\n  Updated: %d files\n", applyPlan.updated)
	writef(out, "  Skipped: %d files (user-modified)\n", applyPlan.skipped)
	if applyPlan.notified > 0 {
		writef(out, "  Removed from template: %d files (review manually)\n", applyPlan.notified)
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

func preservePackState(src *scaffold.Manifest, pack string, dst map[string]scaffold.ManifestFile) {
	preservePackEntries(src, packPrefixFor(pack), dst)
	if v, ok := src.Files[packRuleRelPath(pack)]; ok {
		dst[packRuleRelPath(pack)] = v
	}
}

type plannedFileWrite struct {
	relPath string
	content []byte
}

type upgradeApplyPlan struct {
	writes            []plannedFileWrite
	baselines         []plannedFileWrite
	messages          []string
	updated           int
	skipped           int
	notified          int
	needsConfirmation bool
	conflictStats     conflictReviewStats
}

func (p *upgradeApplyPlan) addMessage(message string) {
	if message != "" {
		p.messages = append(p.messages, message)
	}
}

func (p *upgradeApplyPlan) addTemplateWrite(manifest *scaffold.Manifest, relPath, templateHash string, content []byte, message string) error {
	return p.addResolvedWriteWithMessage(manifest, relPath, templateHash, content, content, message)
}

func (p *upgradeApplyPlan) addResolvedWrite(manifest *scaffold.Manifest, relPath, templateHash string, templateContent, resolvedContent []byte) error {
	return p.addResolvedWriteWithMessage(manifest, relPath, templateHash, templateContent, resolvedContent, fmt.Sprintf("  ✓ %s (file conflict resolved)\n", relPath))
}

func (p *upgradeApplyPlan) addResolvedWriteWithMessage(manifest *scaffold.Manifest, relPath, templateHash string, templateContent, resolvedContent []byte, message string) error {
	baselinePath, err := scaffold.BaselinePath(relPath)
	if err != nil {
		return err
	}
	diskHash := scaffold.HashBytes(resolvedContent)
	state := scaffold.FileStatePartial
	if diskHash == templateHash {
		state = scaffold.FileStateManaged
	}
	manifest.SetFileResolvedWithBaseline(relPath, templateHash, diskHash, state, baselinePath)
	p.writes = append(p.writes, plannedFileWrite{relPath: relPath, content: resolvedContent})
	p.baselines = append(p.baselines, plannedFileWrite{relPath: relPath, content: templateContent})
	p.addMessage(message)
	p.updated++
	return nil
}

func (p *upgradeApplyPlan) addBaselineOnly(manifest *scaffold.Manifest, relPath, templateHash string, templateContent []byte) error {
	baselinePath, err := scaffold.BaselinePath(relPath)
	if err != nil {
		return err
	}
	manifest.SetFileResolvedWithBaseline(relPath, templateHash, templateHash, scaffold.FileStateManaged, baselinePath)
	p.baselines = append(p.baselines, plannedFileWrite{relPath: relPath, content: templateContent})
	return nil
}

func (p *upgradeApplyPlan) apply(absDir string) error {
	for _, write := range p.writes {
		target := filepath.Join(absDir, filepath.FromSlash(write.relPath))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("creating parent for %s: %w", write.relPath, err)
		}
		if err := os.WriteFile(target, write.content, scaffold.FilePerm(write.relPath)); err != nil {
			return fmt.Errorf("writing %s: %w", write.relPath, err)
		}
	}
	for _, baseline := range p.baselines {
		if _, err := scaffold.WriteBaseline(absDir, baseline.relPath, baseline.content); err != nil {
			return err
		}
	}
	return nil
}

func (p *upgradeApplyPlan) writeMessages(out io.Writer) {
	for _, message := range p.messages {
		writef(out, "%s", message)
	}
}

func (p *upgradeApplyPlan) mergeConflictStats(stats conflictReviewStats) {
	p.conflictStats.merge(stats)
}

type conflictReviewStats struct {
	applyFiles map[string]bool
	keepFiles  map[string]bool
	editFiles  map[string]bool
}

func (s *conflictReviewStats) addApply(path string) {
	if s.applyFiles == nil {
		s.applyFiles = make(map[string]bool)
	}
	s.applyFiles[path] = true
}

func (s *conflictReviewStats) addEdit(path string) {
	if s.editFiles == nil {
		s.editFiles = make(map[string]bool)
	}
	s.editFiles[path] = true
}

func (s *conflictReviewStats) addKeep(path string) {
	if s.keepFiles == nil {
		s.keepFiles = make(map[string]bool)
	}
	s.keepFiles[path] = true
}

func (s *conflictReviewStats) merge(other conflictReviewStats) {
	mergeBoolMap(&s.applyFiles, other.applyFiles)
	mergeBoolMap(&s.keepFiles, other.keepFiles)
	mergeBoolMap(&s.editFiles, other.editFiles)
}

func mergeBoolMap(dst *map[string]bool, src map[string]bool) {
	if len(src) == 0 {
		return
	}
	if *dst == nil {
		*dst = make(map[string]bool, len(src))
	}
	for k, v := range src {
		(*dst)[k] = v
	}
}

func confirmApplySummary(plan *upgradeApplyPlan, in *bufio.Reader, out, errOut io.Writer) bool {
	stats := plan.conflictStats
	writef(out, "\nApply summary\n")
	if len(stats.applyFiles) > 0 {
		writef(out, "  Apply template: %d files\n", len(stats.applyFiles))
	}
	if len(stats.keepFiles) > 0 {
		writef(out, "  Keep local:     %d files\n", len(stats.keepFiles))
	}
	if len(stats.editFiles) > 0 {
		writef(out, "  Edited:         %d files\n", len(stats.editFiles))
	}
	writef(out, "\nApply these changes? [y/N] ")
	line, err := in.ReadString('\n')
	if err != nil && line == "" {
		writef(errOut, "\n  (non-interactive input detected, no changes applied)\n")
		return false
	}
	choice := strings.ToLower(strings.TrimSpace(line))
	return choice == "y" || choice == "yes"
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

func preserveManagedSkip(manifest, oldManifest *scaffold.Manifest, absDir string, plan *upgradeApplyPlan, d upgrade.FileDiff) error {
	if prev, ok := oldManifest.Files[d.Path]; ok {
		prev = prev.WithTemplateHash(d.NewHash)
		if _, err := scaffold.ReadBaseline(absDir, prev); err == nil {
			manifest.Files[d.Path] = prev
			return nil
		}
	}
	if d.NewContent != nil && d.DiskHash == d.NewHash {
		return plan.addBaselineOnly(manifest, d.Path, d.NewHash, d.NewContent)
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
	resolutionResolved
)

type conflictResult struct {
	kind             resolution
	content          []byte
	message          string
	conflictReviewed bool
	stats            conflictReviewStats
}

// resolveConflict prompts the user to pick a file-level resolution. The
// conflict diff is rendered inline before the prompt so the decision and
// evidence are visible together. EOF or any read error collapses to a safe skip
// so non-interactive runs do not silently overwrite edits.
func resolveConflict(d upgrade.FileDiff, oldManifest *scaffold.Manifest, absDir, version string, in *bufio.Reader, out, errOut io.Writer, colorize bool) conflictResult {
	writef(out, "  ⚠ %s (modified locally)\n", d.Path)

	if hasReadableBaseline(oldManifest, absDir, d) {
		return resolveConflictWithBaseline(d, oldManifest, absDir, version, in, out, errOut, colorize)
	}

	showDiff(d, absDir, version, out, errOut, colorize, pagerNever)
	for {
		writef(out, "    [o]verwrite / [s]kip / [d]iff ? ")
		line, err := in.ReadString('\n')
		if err != nil && line == "" {
			writef(errOut, "\n  (non-interactive input detected, skipping)\n")
			return conflictResult{kind: resolutionSkip}
		}
		switch strings.TrimSpace(line) {
		case "o", "overwrite":
			return conflictResult{kind: resolutionOverwrite}
		case "s", "skip":
			return conflictResult{kind: resolutionSkip}
		case "d", "diff":
			showDiff(d, absDir, version, out, errOut, colorize, pagerNever)
			// Loop back to the prompt so the user still picks overwrite or skip.
		default:
			// Unrecognized input — reprompt.
		}
	}
}

// showDiff renders the local-vs-template unified diff for a conflict entry.
// When colorize is true the diff is wrapped in ANSI escapes for terminal
// display. Disk read failures degrade gracefully to a warning so the user can
// still make a file-level decision when, e.g., the file was moved between diff
// computation and the prompt.
func showDiff(d upgrade.FileDiff, absDir, version string, out, errOut io.Writer, colorize bool, pagerMode string) {
	localPath := filepath.Join(absDir, d.Path)
	localBytes, err := os.ReadFile(localPath)
	if err != nil {
		writef(errOut, "    (could not read %s: %v)\n", d.Path, err)
		return
	}
	diff := upgrade.UnifiedDiff(
		localBytes,
		d.NewContent,
		"local",
		fmt.Sprintf("template (%s)", version),
	)
	diff = omitRangeHeaders(diff)
	if diff == "" {
		writef(out, "    (no textual difference — manifest hash drift only)\n")
	} else {
		if colorize {
			diff = upgrade.Colorize(diff)
		}
		writeDiffOutput(diff, out, errOut, pagerMode)
	}
}

func omitRangeHeaders(diff string) string {
	if diff == "" {
		return ""
	}
	var b strings.Builder
	for _, line := range strings.SplitAfter(diff, "\n") {
		if strings.HasPrefix(strings.TrimSuffix(line, "\n"), "@@ ") {
			continue
		}
		b.WriteString(line)
	}
	return b.String()
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

func resolveConflictWithBaseline(d upgrade.FileDiff, oldManifest *scaffold.Manifest, absDir, version string, in *bufio.Reader, out, errOut io.Writer, colorize bool) conflictResult {
	prev := oldManifest.Files[d.Path]
	baselineBytes, err := scaffold.ReadBaseline(absDir, prev)
	if err != nil {
		writef(errOut, "    (could not read baseline for %s: %v)\n", d.Path, err)
		return conflictResult{kind: resolutionSkip}
	}
	localBytes, err := os.ReadFile(filepath.Join(absDir, d.Path))
	if err != nil {
		writef(errOut, "    (could not read %s: %v)\n", d.Path, err)
		return conflictResult{kind: resolutionSkip}
	}

	mergePlan := upgrade.PlanMerge(baselineBytes, localBytes, d.NewContent)
	if mergePlan.ConflictCount() == 0 {
		return conflictResult{kind: resolutionResolved, content: d.NewContent}
	}

	stats := conflictReviewStats{}
	showDiff(d, absDir, version, out, errOut, colorize, pagerNever)
	for {
		writef(out, "    [a]pply template file / [k]eep local file / [e]dit file ? ")
		line, err := in.ReadString('\n')
		if err != nil && line == "" {
			writef(errOut, "\n  (non-interactive input detected, keeping local file)\n")
			stats.addKeep(d.Path)
			return conflictResult{kind: resolutionSkip, conflictReviewed: true, stats: stats}
		}
		switch strings.TrimSpace(line) {
		case "a", "apply":
			stats.addApply(d.Path)
			return conflictResult{kind: resolutionOverwrite, conflictReviewed: true, stats: stats}
		case "k", "keep":
			stats.addKeep(d.Path)
			return conflictResult{
				kind:             resolutionResolved,
				content:          localBytes,
				message:          fmt.Sprintf("  ✓ %s (kept local file)\n", d.Path),
				conflictReviewed: true,
				stats:            stats,
			}
		case "e", "edit":
			editedContent, editErr := editConflictFile(mergePlan, version, out, errOut)
			if editErr != nil {
				writef(errOut, "    (manual edit failed: %v)\n", editErr)
				continue
			}
			stats.addEdit(d.Path)
			return conflictResult{
				kind:             resolutionResolved,
				content:          editedContent,
				conflictReviewed: true,
				stats:            stats,
			}
		default:
			// Unrecognized input — reprompt.
		}
	}
}

func editConflictFile(plan upgrade.MergePlan, version string, out, errOut io.Writer) ([]byte, error) {
	editedContent, err := editContentBytes(upgrade.JoinLines(conflictMarkerFileLines(plan, version), true), out, errOut)
	if err != nil {
		return nil, err
	}
	if hasConflictMarkers(editedContent) {
		return nil, fmt.Errorf("unresolved conflict markers remain")
	}
	return editedContent, nil
}

func conflictMarkerFileLines(plan upgrade.MergePlan, version string) []string {
	lines := make([]string, 0, len(plan.LocalLines)+len(plan.TemplateLines)+len(plan.Regions)*3)
	cursor := 0
	for _, region := range plan.Regions {
		if region.BaseStart > cursor {
			lines = append(lines, plan.BaseLines[cursor:region.BaseStart]...)
		}
		if region.NeedsResolution() {
			lines = append(lines, "<<<<<<< local")
			lines = append(lines, region.LocalLines...)
			lines = append(lines, "=======")
			lines = append(lines, region.TemplateLines...)
			lines = append(lines, fmt.Sprintf(">>>>>>> template (%s)", version))
		} else {
			lines = append(lines, region.TemplateLines...)
		}
		cursor = region.BaseEnd
	}
	if cursor < len(plan.BaseLines) {
		lines = append(lines, plan.BaseLines[cursor:]...)
	}
	return lines
}

func hasConflictMarkers(content []byte) bool {
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if strings.HasPrefix(line, "<<<<<<< ") ||
			line == "=======" ||
			strings.HasPrefix(line, ">>>>>>> ") ||
			strings.HasPrefix(line, "||||||| ") {
			return true
		}
	}
	return false
}

func editContentBytes(initial []byte, out, errOut io.Writer) ([]byte, error) {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if strings.TrimSpace(editor) == "" {
		return nil, fmt.Errorf("VISUAL/EDITOR is not set")
	}
	tmp, err := os.CreateTemp("", "ralph-upgrade-conflict-*")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(initial); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return nil, err
	}
	defer func() { _ = os.Remove(tmpPath) }()

	parts := strings.Fields(editor)
	cmd := exec.Command(parts[0], append(parts[1:], tmpPath)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = out
	cmd.Stderr = errOut
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	edited, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, err
	}
	return edited, nil
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
