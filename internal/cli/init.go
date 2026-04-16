package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/scaffold"
)

func newInitCmd() *cobra.Command {
	var nonInteractive bool

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
				return runInitNonInteractive(absDir)
			}
			return runInitInteractive(absDir)
		},
	}

	cmd.Flags().BoolVar(&nonInteractive, "yes", false, "skip interactive prompts, use defaults")

	return cmd
}

type initConfig struct {
	ProjectName string
	Packs       []string
}

func runInitInteractive(targetDir string) error {
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

	return executeInit(targetDir, cfg)
}

func runInitNonInteractive(targetDir string) error {
	availPacks, err := scaffold.AvailablePacks()
	if err != nil {
		return fmt.Errorf("listing packs: %w", err)
	}

	cfg := initConfig{
		ProjectName: filepath.Base(targetDir),
		Packs:       availPacks,
	}

	return executeInit(targetDir, cfg)
}

func executeInit(targetDir string, cfg initConfig) error {
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

	// Step 1: Render base templates (fresh init — no manifest, safe to overwrite).
	baseFS, err := scaffold.BaseFS()
	if err != nil {
		return fmt.Errorf("loading base templates: %w", err)
	}

	result, hashes, err := scaffold.RenderFS(baseFS, scaffold.RenderOptions{
		TargetDir: targetDir,
		Overwrite: true,
	})
	if err != nil {
		return fmt.Errorf("rendering base templates: %w", err)
	}
	printRenderSummary("base", result)

	// Step 2: Render selected language packs.
	for _, pack := range cfg.Packs {
		packFS, err := scaffold.PackFS(pack)
		if err != nil {
			fmt.Printf("  ⚠ pack %s: %v\n", pack, err)
			continue
		}
		packResult, packHashes, err := scaffold.RenderFS(packFS, scaffold.RenderOptions{
			TargetDir: targetDir,
			Overwrite: true,
		})
		if err != nil {
			fmt.Printf("  ⚠ pack %s: %v\n", pack, err)
			continue
		}
		// Merge pack hashes (prefix with pack path for manifest).
		for k, v := range packHashes {
			hashes[k] = v
		}
		printRenderSummary("pack/"+pack, packResult)
	}

	// Step 3: Create manifest.
	manifest := scaffold.NewManifest(Version)
	for path, hash := range hashes {
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
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
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

	fmt.Printf("\nDone. Next steps:\n")
	if targetDir != "." {
		fmt.Printf("  cd %s\n", targetDir)
	}
	fmt.Printf("  Edit AGENTS.md to describe your project\n")
	fmt.Printf("  ralph doctor to verify setup\n")

	return nil
}

func printRenderSummary(label string, result *scaffold.RenderResult) {
	created := len(result.Created)
	overwritten := len(result.Overwritten)
	skipped := len(result.Skipped)
	total := created + overwritten + skipped
	fmt.Printf("  ✓ %s (%d files: %d created, %d updated, %d skipped)\n",
		label, total, created, overwritten, skipped)
}
