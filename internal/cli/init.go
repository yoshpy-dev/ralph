package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
	"github.com/yoshpy-dev/ralph/internal/upgrade"
)

func newInitCmd() *cobra.Command {
	var (
		nonInteractive bool
		force          bool
	)

	cmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize a new project with harness engineering scaffold",
		Long: `Scaffolds a project with Claude Code configurations, hooks, skills,
agents, rules, and org-runtime config. Supports both new and existing projects.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetDir := "."
			if len(args) > 0 {
				targetDir = args[0]
			}

			absDir, err := filepath.Abs(targetDir)
			if err != nil {
				return fmt.Errorf("resolving directory: %w", err)
			}

			if nonInteractive {
				return runInitNonInteractive(absDir, force)
			}
			return runInitInteractive(absDir, force)
		},
	}

	cmd.Flags().BoolVar(&nonInteractive, "yes", false, "skip interactive prompts, use defaults")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing files without prompting")

	return cmd
}

type initConfig struct {
	ProjectName string
	Packs       []string
}

func runInitInteractive(targetDir string, force bool) error {
	// Detect an existing project up front so the user is not asked for a
	// project name and language packs that will be ignored by the upgrade
	// path. executeInit retains the same check as a safety net for
	// non-interactive callers.
	manifestPath := filepath.Join(targetDir, ".ralph", "manifest.toml")
	if _, err := os.Stat(manifestPath); err == nil {
		return handleExistingProjectInit(targetDir, manifestPath)
	}

	defaultName := filepath.Base(targetDir)

	availPacks, err := scaffold.AvailablePacks()
	if err != nil {
		return fmt.Errorf("listing packs: %w", err)
	}

	cfg := initConfig{
		ProjectName: defaultName,
		Packs:       availPacks, // Default: all packs selected.
	}

	// Build multi-select options with all packs pre-selected.
	packOptions := make([]huh.Option[string], len(availPacks))
	for i, p := range availPacks {
		packOptions[i] = huh.NewOption(p, p).Selected(true)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project name").
				Value(&cfg.ProjectName),
			huh.NewMultiSelect[string]().
				Title("Language packs").
				Options(packOptions...).
				Value(&cfg.Packs),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("interactive form: %w", err)
	}

	return executeInit(targetDir, cfg, force)
}

func runInitNonInteractive(targetDir string, force bool) error {
	availPacks, err := scaffold.AvailablePacks()
	if err != nil {
		return fmt.Errorf("listing packs: %w", err)
	}

	cfg := initConfig{
		ProjectName: filepath.Base(targetDir),
		Packs:       availPacks,
	}

	return executeInit(targetDir, cfg, force)
}

// handleExistingProjectInit is invoked whenever `ralph init` targets a
// directory that already has a .ralph/manifest.toml. Legacy (pre-v2)
// projects keep the prior behavior of delegating to the legacy upgrade
// engine. Projects already on the overlay (v2) layout cannot use that
// engine — upgrade.go's fail-closed guard (AC-10, docs/specs
// 2026-08-17-overlay-scaffold-v2.md) refuses to run against a
// meta.layout = "v2" manifest — so re-running init against them is a no-op
// today; the non-interactive v2 upgrade path lands in a later ralph
// release (Phase 3).
func handleExistingProjectInit(targetDir, manifestPath string) error {
	m, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}
	if m.Meta.Layout == scaffold.LayoutV2 {
		fmt.Printf("\nExisting v2-layout project detected. Nothing to do — the non-interactive v2 upgrade path lands in a later ralph release (Phase 3).\n\n")
		return nil
	}
	fmt.Printf("\nExisting project detected. Running upgrade instead...\n\n")
	return runUpgrade(targetDir, false)
}

func executeInit(targetDir string, cfg initConfig, force bool) error {
	// Ensure target directory exists.
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// If a manifest already exists, this is a re-init on an existing project.
	// Delegate to upgrade logic to preserve user-edited files.
	manifestPath := filepath.Join(targetDir, ".ralph", "manifest.toml")
	if _, err := os.Stat(manifestPath); err == nil {
		return handleExistingProjectInit(targetDir, manifestPath)
	}

	fmt.Printf("\nScaffolding %q into %s ...\n\n", cfg.ProjectName, targetDir)

	// Step 1: Render base templates.
	baseFS, err := scaffold.BaseFS()
	if err != nil {
		return fmt.Errorf("loading base templates: %w", err)
	}

	result, hashes, err := scaffold.RenderFS(baseFS, scaffold.RenderOptions{
		TargetDir: targetDir,
		Overwrite: force,
	})
	if err != nil {
		return fmt.Errorf("rendering base templates: %w", err)
	}
	baselinePaths, err := writeRenderedBaselines(targetDir, baseFS, "", result)
	if err != nil {
		return err
	}
	printRenderSummary("base", result)

	// Step 1b: reconcile the ralph managed block into any block-owned
	// surfaces (AGENTS.md, .gitignore) that RenderFS skipped because they
	// already existed. Files RenderFS actually wrote (fresh init, or
	// --force) already carry the block as part of the template content, so
	// this only ever touches pre-existing user files.
	blockDiskHashes, err := reconcileBlockSurfaces(targetDir, baseFS, result.Skipped, os.Stdout)
	if err != nil {
		return err
	}

	// Step 2: Render selected language packs into packs/languages/<lang>/.
	// Pack rule.md files are control files: they render to
	// .claude/rules/ralph/<lang>.md instead of packs/languages/<lang>/rule.md.
	// renderPackInto (language_pack.go) is the shared helper used here and by
	// addPack (pack.go) so the two code paths cannot diverge.
	for _, pack := range cfg.Packs {
		pr, err := renderPackInto(targetDir, pack, force)
		if err != nil {
			fmt.Printf("  ⚠ pack %s: %v\n", pack, err)
			continue
		}
		for k, v := range pr.hashes {
			hashes[k] = v
		}
		for k, v := range pr.baselinePaths {
			baselinePaths[k] = v
		}
		printRenderSummary("pack/"+pack, pr.result)
	}

	// Step 3: Create manifest.
	manifest := scaffold.NewManifest(Version)
	manifest.Meta.Packs = cfg.Packs
	for path, hash := range hashes {
		if baselinePath, ok := baselinePaths[path]; ok {
			manifest.SetFileWithBaseline(path, hash, baselinePath)
			continue
		}
		manifest.SetFile(path, hash)
	}

	// Step 3b: mark the manifest as v2 layout and assign every entry an
	// ownership attribute (core/seed/block). SetOwner mutates only the
	// Owner field, leaving baseline metadata set above untouched. Block
	// surfaces that were actually rewritten in step 1b additionally get
	// their DiskHash recorded via SetFileOwned.
	manifest.SetLayoutV2()
	for path, templateHash := range hashes {
		owner := ownerForScaffoldPath(path)
		if diskHash, ok := blockDiskHashes[path]; ok && owner == scaffold.OwnerBlock {
			if err := manifest.SetFileOwned(path, owner, templateHash, diskHash); err != nil {
				return fmt.Errorf("recording block owner for %s: %w", path, err)
			}
			continue
		}
		if err := manifest.SetOwner(path, owner); err != nil {
			return fmt.Errorf("setting owner for %s: %w", path, err)
		}
	}

	manifestDir := filepath.Join(targetDir, ".ralph")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		return fmt.Errorf("creating .ralph dir: %w", err)
	}
	mPath := filepath.Join(manifestDir, "manifest.toml")
	if err := manifest.Write(mPath); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}
	fmt.Printf("  ✓ .ralph/manifest.toml\n")

	// Step 4: Git init if needed.
	gitDir := filepath.Join(targetDir, ".git")
	if _, err := os.Stat(gitDir); errors.Is(err, fs.ErrNotExist) {
		if gitBin, err := exec.LookPath("git"); err == nil {
			cmd := exec.Command(gitBin, "init")
			cmd.Dir = targetDir
			if out, err := cmd.CombinedOutput(); err != nil {
				fmt.Printf("  ⚠ git init failed: %s\n", out)
			} else {
				fmt.Printf("  ✓ git init\n")
			}
		}
	} else {
		fmt.Printf("  ✓ .git exists (skipped)\n")
	}

	// Step 5: Install local git hooks when possible. This is a runtime side
	// effect rather than a manifest-managed template file because .git/ is
	// local to each clone/worktree.
	installManagedGitHooks(targetDir, os.Stdout, os.Stdout)

	fmt.Printf("\nDone. Next steps:\n")
	if targetDir != "." {
		fmt.Printf("  cd %s\n", targetDir)
	}
	fmt.Printf("  Edit AGENTS.md to describe your project\n")
	fmt.Printf("  ralph doctor to verify setup\n")

	return nil
}

// ownerForScaffoldPath returns the manifest v3 ownership attribute for a
// scaffolded file, keyed by its manifest-relative path (the same path used
// as the key of the hashes map built during rendering). See docs/specs
// 2026-08-17-overlay-scaffold-v2.md, section "層モデル".
//
// .ralph/local/** is classified as seed, not core, even though the
// catch-all below would otherwise mark it core: per the spec's layer model
// it is the L3 overlay, a user drop-in area that is 不可侵 (create-once,
// then advisory-only) once it exists. Filing it under core would let the
// Phase 3 replace planner treat it as a full-replace target and overwrite
// user content living there.
//
// Manifest keys are always fs.FS slash paths (from fs.WalkDir in
// render.go), regardless of host OS, so relPath is normalized with
// filepath.ToSlash before comparison to keep classification slash-stable
// on Windows.
func ownerForScaffoldPath(relPath string) string {
	relPath = filepath.ToSlash(relPath)
	switch relPath {
	case "AGENTS.md", ".gitignore":
		return scaffold.OwnerBlock
	case "CLAUDE.md", "ralph.toml", ".github/workflows/verify.yml":
		return scaffold.OwnerSeed
	}
	if strings.HasPrefix(relPath, "docs/") {
		return scaffold.OwnerSeed
	}
	if strings.HasPrefix(relPath, ".ralph/local/") {
		return scaffold.OwnerSeed
	}
	return scaffold.OwnerCore
}

// blockSurface pairs a block-owned manifest path with the ralph surface
// token and marker style used to locate its managed block.
type blockSurface struct {
	path    string
	surface string
	style   upgrade.BlockMarkerStyle
}

// blockSurfaces lists every file whose ownership attribute is "block":
// files that are user-owned outside a single ralph-managed marker pair.
var blockSurfaces = []blockSurface{
	{path: "AGENTS.md", surface: "agents-md", style: upgrade.BlockMarkerHTML},
	{path: ".gitignore", surface: "gitignore", style: upgrade.BlockMarkerHash},
}

// reconcileBlockSurfaces appends the ralph managed block into pre-existing
// AGENTS.md / .gitignore files that RenderFS skipped because they already
// existed (init never overwrites arbitrary existing files outside
// --force). Bytes outside the block are preserved exactly; a malformed
// existing block, or one that already contains a well-formed block, is left
// untouched (with a warning for the malformed case), matching the block
// engine's non-destructive stance — updating an already-present block's
// content is upgrade's job (Phase 3), not init's.
//
// Returns the on-disk SHA256 hash (scaffold.HashBytes format) for every path
// actually rewritten, keyed by manifest-relative path.
func reconcileBlockSurfaces(targetDir string, baseFS fs.FS, skipped []string, w io.Writer) (map[string]string, error) {
	skippedSet := make(map[string]bool, len(skipped))
	for _, p := range skipped {
		skippedSet[p] = true
	}

	diskHashes := make(map[string]string)
	for _, bs := range blockSurfaces {
		if !skippedSet[bs.path] {
			continue
		}

		templateContent, err := fs.ReadFile(baseFS, bs.path)
		if err != nil {
			// Not every embedded scaffold ships every block surface (e.g.
			// tests with a minimal mock FS); nothing to reconcile.
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("reading template %s: %w", bs.path, err)
		}
		interior, err := extractBlockInterior(templateContent, bs.surface, bs.style)
		if err != nil {
			return nil, fmt.Errorf("extracting managed block from template %s: %w", bs.path, err)
		}

		diskPath := filepath.Join(targetDir, bs.path)
		current, err := os.ReadFile(diskPath)
		if err != nil {
			return nil, fmt.Errorf("reading existing %s: %w", bs.path, err)
		}

		result := upgrade.UpdateManagedBlockStyled(current, bs.surface, interior, bs.style)
		switch result.Outcome {
		case upgrade.BlockAppended:
			if err := os.WriteFile(diskPath, result.Content, scaffold.FilePerm(bs.path)); err != nil {
				return nil, fmt.Errorf("writing %s: %w", bs.path, err)
			}
			diskHashes[bs.path] = scaffold.HashBytes(result.Content)
			writef(w, "  ✓ %s (appended ralph managed block)\n", bs.path)
		case upgrade.BlockMalformed:
			writef(w, "  ⚠ %s: existing ralph managed block is malformed (%s); left untouched\n", bs.path, result.Reason)
		default:
			// BlockUpdated / BlockUnchanged: a well-formed block already
			// exists on disk. Init only appends when a block is absent.
		}
	}
	return diskHashes, nil
}

// extractBlockInterior returns the bytes strictly between the BEGIN and END
// marker lines of a rendered template's managed block, suitable for passing
// as the "managed" argument to UpdateManagedBlockStyled when seeding a block
// into a file that does not have one yet.
func extractBlockInterior(templateContent []byte, surface string, style upgrade.BlockMarkerStyle) ([]byte, error) {
	begin := upgrade.BeginMarkerStyled(surface, style)
	end := upgrade.EndMarkerStyled(style)

	lines := strings.Split(string(templateContent), "\n")
	beginIdx, endIdx := -1, -1
	for i, l := range lines {
		trimmed := strings.TrimSuffix(l, "\r")
		if trimmed == begin && beginIdx == -1 {
			beginIdx = i
		}
		if trimmed == end && beginIdx != -1 && endIdx == -1 {
			endIdx = i
		}
	}
	if beginIdx == -1 || endIdx == -1 {
		return nil, fmt.Errorf("template block markers not found for surface %q", surface)
	}

	interior := lines[beginIdx+1 : endIdx]
	if len(interior) == 0 {
		return nil, nil
	}
	return []byte(strings.Join(interior, "\n") + "\n"), nil
}

func writeRenderedBaselines(targetDir string, src fs.FS, prefix string, result *scaffold.RenderResult) (map[string]string, error) {
	out := make(map[string]string)
	written := make(map[string]bool, len(result.Created)+len(result.Overwritten))
	for _, path := range result.Created {
		written[path] = true
	}
	for _, path := range result.Overwritten {
		written[path] = true
	}
	for path := range written {
		content, err := fs.ReadFile(src, path)
		if err != nil {
			return nil, fmt.Errorf("reading baseline source %s: %w", path, err)
		}
		manifestPath := filepath.Join(prefix, path)
		baselinePath, err := scaffold.WriteBaseline(targetDir, manifestPath, content)
		if err != nil {
			return nil, err
		}
		out[manifestPath] = baselinePath
	}
	return out, nil
}

func printRenderSummary(label string, result *scaffold.RenderResult) {
	created := len(result.Created)
	overwritten := len(result.Overwritten)
	skipped := len(result.Skipped)
	total := created + overwritten + skipped
	fmt.Printf("  ✓ %s (%d files: %d created, %d updated, %d skipped)\n",
		label, total, created, overwritten, skipped)
}
