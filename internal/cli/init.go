package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/scaffold"
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
agents, rules, and pipeline settings. Supports both new and existing projects.`,
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
		fmt.Printf("\nExisting project detected. Running upgrade instead...\n\n")
		return runUpgrade(targetDir, false)
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

func executeInit(targetDir string, cfg initConfig, force bool) error {
	// Ensure target directory exists.
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// If a manifest already exists, this is a re-init on an existing project.
	// Delegate to upgrade logic to preserve user-edited files.
	manifestPath := filepath.Join(targetDir, ".ralph", "manifest.toml")
	if _, err := os.Stat(manifestPath); err == nil {
		fmt.Printf("\nExisting project detected. Running upgrade instead...\n\n")
		return runUpgrade(targetDir, false)
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

	// Step 2: Render selected language packs into packs/languages/<lang>/.
	// Pack rule.md files are control files: they render to
	// .claude/rules/<lang>.md instead of packs/languages/<lang>/rule.md.
	for _, pack := range cfg.Packs {
		packFS, err := scaffold.PackFS(pack)
		if err != nil {
			fmt.Printf("  ⚠ pack %s: %v\n", pack, err)
			continue
		}
		packDir := filepath.Join(targetDir, packRelDir(pack))
		packResult, packHashes, err := scaffold.RenderFS(packFS, scaffold.RenderOptions{
			TargetDir: packDir,
			Overwrite: force,
			SkipPaths: packRenderSkipPaths,
		})
		if err != nil {
			fmt.Printf("  ⚠ pack %s: %v\n", pack, err)
			continue
		}
		// Merge pack hashes with namespaced paths for manifest.
		packPrefix := packRelDir(pack)
		for k, v := range packHashes {
			hashes[filepath.Join(packPrefix, k)] = v
		}
		packBaselines, err := writeRenderedBaselines(targetDir, packFS, packPrefix, packResult)
		if err != nil {
			return err
		}
		for k, v := range packBaselines {
			baselinePaths[k] = v
		}

		ruleContent, ok, err := packRuleContent(packFS)
		if err != nil {
			fmt.Printf("  ⚠ pack %s rule: %v\n", pack, err)
			continue
		}
		if ok {
			rulePath := packRuleRelPath(pack)
			ruleResult, ruleHash, err := renderMappedFile(targetDir, rulePath, ruleContent, force)
			if err != nil {
				fmt.Printf("  ⚠ pack %s rule: %v\n", pack, err)
				continue
			}
			hashes[rulePath] = ruleHash
			if len(ruleResult.Created)+len(ruleResult.Overwritten) > 0 {
				baselinePath, err := scaffold.WriteBaseline(targetDir, rulePath, ruleContent)
				if err != nil {
					return err
				}
				baselinePaths[rulePath] = baselinePath
			}
			mergeRenderResult(packResult, ruleResult)
		}
		printRenderSummary("pack/"+pack, packResult)
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
