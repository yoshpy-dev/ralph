package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/action"
	"github.com/yoshpy-dev/harness-engineering-scaffolding-template/internal/config"
)

func newRunCmd() *cobra.Command {
	var (
		planPath      string
		maxIterations int
		maxParallel   int
		preflight     bool
		resume        bool
		dryRun        bool
		unifiedPR     bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute the autonomous development pipeline",
		Long:  "Runs the Ralph Loop orchestrator for parallel slice execution.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPipeline(planPath, maxIterations, maxParallel, preflight, resume, dryRun, unifiedPR)
		},
	}

	cmd.Flags().StringVar(&planPath, "plan", "", "plan directory (auto-detected if omitted)")
	cmd.Flags().IntVar(&maxIterations, "max-iterations", 0, "total iteration cap (default from ralph.toml)")
	cmd.Flags().IntVar(&maxParallel, "max-parallel", 0, "max concurrent slices (default from ralph.toml)")
	cmd.Flags().BoolVar(&preflight, "preflight", false, "run capability probe only")
	cmd.Flags().BoolVar(&resume, "resume", false, "resume from existing checkpoint")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would run without executing")
	cmd.Flags().BoolVar(&unifiedPR, "unified-pr", false, "create a unified PR from integration branch")

	return cmd
}

func runPipeline(planPath string, maxIter, maxPar int, preflight, resume, dryRun, unifiedPR bool) error {
	// Load config for defaults.
	cfg, err := config.Load("ralph.toml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: ralph.toml parse error: %v — using defaults\n", err)
	}

	// Build environment from TOML config.
	env := os.Environ()
	env = append(env,
		"RALPH_MODEL="+cfg.Pipeline.Model,
		"RALPH_EFFORT="+cfg.Pipeline.Effort,
		"RALPH_PERMISSION_MODE="+cfg.Pipeline.PermissionMode,
	)
	if maxIter == 0 {
		maxIter = cfg.Pipeline.MaxIterations
	}
	if maxPar == 0 {
		maxPar = cfg.Pipeline.MaxParallel
	}
	env = append(env,
		fmt.Sprintf("RALPH_MAX_ITERATIONS=%d", maxIter),
		fmt.Sprintf("RALPH_MAX_PARALLEL=%d", maxPar),
	)

	// Find the orchestrator script.
	scriptPath, err := findScript("ralph-orchestrator.sh")
	if err != nil {
		return fmt.Errorf("orchestrator script not found: %w", err)
	}

	// Build args.
	var scriptArgs []string
	if planPath != "" {
		scriptArgs = append(scriptArgs, "--plan", planPath)
	}
	if preflight {
		scriptArgs = append(scriptArgs, "--preflight")
	}
	if resume {
		scriptArgs = append(scriptArgs, "--resume")
	}
	if dryRun {
		scriptArgs = append(scriptArgs, "--dry-run")
	}
	if unifiedPR {
		scriptArgs = append(scriptArgs, "--unified-pr")
	}

	execCmd := exec.Command(scriptPath, scriptArgs...)
	execCmd.Env = env
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin

	return execCmd.Run()
}

func newRetryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "retry <slice-name>",
		Short: "Retry a failed or stuck slice",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := action.ValidateSliceName(args[0]); err != nil {
				return fmt.Errorf("invalid slice name: %w", err)
			}
			scriptPath, err := findScript("ralph")
			if err != nil {
				return err
			}
			execCmd := exec.Command(scriptPath, "retry", args[0])
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			return execCmd.Run()
		},
	}
}

func newAbortCmd() *cobra.Command {
	var sliceName string

	cmd := &cobra.Command{
		Use:   "abort",
		Short: "Safely stop and clean up pipeline state",
		RunE: func(cmd *cobra.Command, args []string) error {
			scriptPath, err := findScript("ralph")
			if err != nil {
				return err
			}
			scriptArgs := []string{"abort"}
			if sliceName != "" {
				if err := action.ValidateSliceName(sliceName); err != nil {
					return fmt.Errorf("invalid slice name: %w", err)
				}
				scriptArgs = append(scriptArgs, "--slice", sliceName)
			}
			execCmd := exec.Command(scriptPath, scriptArgs...)
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			return execCmd.Run()
		},
	}

	cmd.Flags().StringVar(&sliceName, "slice", "", "abort a specific slice only")

	return cmd
}

// findScript locates a script in scripts/ relative to the current directory.
func findScript(name string) (string, error) {
	path := filepath.Join("scripts", name)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("script %q not found in scripts/", name)
}
