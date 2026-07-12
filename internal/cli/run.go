package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/action"
	"github.com/yoshpy-dev/ralph/internal/config"
)

const activePlansDir = "docs/plans/active"

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
			maxIterChanged := cmd.Flags().Changed("max-iterations")
			maxParChanged := cmd.Flags().Changed("max-parallel")
			return runPipeline(planPath, maxIterations, maxParallel, preflight, resume, dryRun, unifiedPR, maxIterChanged, maxParChanged)
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

func runPipeline(planPath string, maxIter, maxPar int, preflight, resume, dryRun, unifiedPR bool, maxIterChanged, maxParChanged bool) error {
	// Load config for defaults.
	cfg, err := config.Load("ralph.toml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: ralph.toml parse error: %v — using defaults\n", err)
	}

	// Build environment with env > TOML > default priority.
	// RALPH_MODEL, RALPH_EFFORT, RALPH_PERMISSION_MODE have no CLI flags;
	// use appendEnvIfMissing so a pre-set env var wins over the TOML value.
	env := os.Environ()
	env = appendEnvIfMissing(env, "RALPH_MODEL", cfg.Pipeline.Model)
	env = appendEnvIfMissing(env, "RALPH_EFFORT", cfg.Pipeline.Effort)
	env = appendEnvIfMissing(env, "RALPH_PERMISSION_MODE", cfg.Pipeline.PermissionMode)

	// RALPH_MAX_ITERATIONS and RALPH_MAX_PARALLEL honour CLI > env > TOML.
	// Use Flags().Changed() (Cobra flag-presence) — NOT the != 0 heuristic —
	// to detect whether the flag was explicitly set. When the flag was set,
	// export the CLI value unconditionally (it wins over any env var). When
	// absent, fall back to the env-or-TOML value via appendEnvIfMissing.
	if maxIterChanged {
		env = append(env, fmt.Sprintf("RALPH_MAX_ITERATIONS=%d", maxIter))
	} else {
		env = appendEnvIfMissing(env, "RALPH_MAX_ITERATIONS", fmt.Sprintf("%d", cfg.Pipeline.MaxIterations))
	}
	if maxParChanged {
		env = append(env, fmt.Sprintf("RALPH_MAX_PARALLEL=%d", maxPar))
	} else {
		env = appendEnvIfMissing(env, "RALPH_MAX_PARALLEL", fmt.Sprintf("%d", cfg.Pipeline.MaxParallel))
	}

	// Phase 2 (issue #44) — propagate [loop] settings only when the user has
	// not already set the env var. This makes `[loop] driver = "codex"` in
	// ralph.toml runtime-effective for `ralph run`, while preserving the
	// documented priority "env > TOML > default".
	env = appendEnvIfMissing(env, "RALPH_LOOP_DRIVER", cfg.Loop.Driver)
	env = appendEnvIfMissing(env, "RALPH_CODEX_SANDBOX", cfg.Loop.CodexSandbox)
	env = appendEnvIfMissing(env, "RALPH_CODEX_APPROVAL_POLICY", cfg.Loop.CodexApprovalPolicy)
	env = appendEnvIfMissing(env, "RALPH_CLAUDE_REVIEWER_MODEL", cfg.Loop.ClaudeReviewerModel)

	// Per-phase model routing — propagate [pipeline.phases] values only when
	// the user has not already set the corresponding env var (env > TOML > default).
	// RALPH_FORCE_MODEL is only exported when the toml value is non-empty.
	// appendEnvIfMissing skips adding when the key is already present in env
	// (i.e. when the user exported it), so an empty cfg.Pipeline.Phases.Force
	// would append "RALPH_FORCE_MODEL=" which would then mask any user-set env
	// var on subsequent reads (the loop finds "RALPH_FORCE_MODEL=" and returns
	// early). To avoid that, we guard with an explicit non-empty check so that
	// an absent/empty [pipeline.phases] force never writes a blank override.
	env = appendEnvIfMissing(env, "RALPH_IMPLEMENT_MODEL", cfg.Pipeline.Phases.Implement)
	env = appendEnvIfMissing(env, "RALPH_SELF_REVIEW_MODEL", cfg.Pipeline.Phases.SelfReview)
	env = appendEnvIfMissing(env, "RALPH_VERIFY_MODEL", cfg.Pipeline.Phases.Verify)
	env = appendEnvIfMissing(env, "RALPH_TEST_MODEL", cfg.Pipeline.Phases.Test)
	env = appendEnvIfMissing(env, "RALPH_SYNC_DOCS_MODEL", cfg.Pipeline.Phases.SyncDocs)
	env = appendEnvIfMissing(env, "RALPH_PR_MODEL", cfg.Pipeline.Phases.PR)
	env = appendEnvIfMissing(env, "RALPH_PROBE_MODEL", cfg.Pipeline.Phases.Probe)
	env = appendEnvIfMissing(env, "RALPH_ESCALATION_MODEL", cfg.Pipeline.Phases.Escalation)
	if cfg.Pipeline.Phases.Force != "" {
		env = appendEnvIfMissing(env, "RALPH_FORCE_MODEL", cfg.Pipeline.Phases.Force)
	}

	if planPath == "" {
		planPath, err = detectLatestPlanDir(activePlansDir)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Auto-detected plan: %s\n", planPath)
	}

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

func detectLatestPlanDir(activeDir string) (string, error) {
	entries, err := os.ReadDir(activeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no directory-based plan found. Create one with './scripts/new-ralph-plan.sh --type <type> <slug>' or specify --plan <directory>")
		}
		return "", fmt.Errorf("read active plans: %w", err)
	}

	var plans []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		planDir := filepath.Join(activeDir, entry.Name())
		manifest := filepath.Join(planDir, "_manifest.md")
		if _, err := os.Stat(manifest); err == nil {
			plans = append(plans, planDir)
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect plan manifest %q: %w", manifest, err)
		}
	}

	if len(plans) == 0 {
		return "", fmt.Errorf("no directory-based plan found. Create one with './scripts/new-ralph-plan.sh --type <type> <slug>' or specify --plan <directory>")
	}

	sort.Sort(sort.Reverse(sort.StringSlice(plans)))
	return plans[0], nil
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

// appendEnvIfMissing appends "KEY=value" to env only when KEY is not already
// present. Used to propagate ralph.toml [loop] settings to the orchestrator
// while letting an explicit env var win — the documented priority is
// env > TOML > built-in default.
func appendEnvIfMissing(env []string, key, value string) []string {
	prefix := key + "="
	for _, e := range env {
		if len(e) >= len(prefix) && e[:len(prefix)] == prefix {
			return env
		}
	}
	return append(env, prefix+value)
}
