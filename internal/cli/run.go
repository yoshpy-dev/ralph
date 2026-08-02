package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/action"
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
	cmd.Flags().IntVar(&maxIterations, "max-iterations", 0, "total iteration cap (default from RALPH_MAX_ITERATIONS env or ralph.toml)")
	cmd.Flags().IntVar(&maxParallel, "max-parallel", 0, "max concurrent slices (default from RALPH_MAX_PARALLEL env or ralph.toml)")
	cmd.Flags().BoolVar(&preflight, "preflight", false, "run capability probe only")
	cmd.Flags().BoolVar(&resume, "resume", false, "resume from existing checkpoint")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would run without executing")
	cmd.Flags().BoolVar(&unifiedPR, "unified-pr", false, "create a unified PR from integration branch")

	return cmd
}

// runDefaults holds the literal fallback values `ralph run` used to read
// from ralph.toml's now-removed [pipeline]/[loop] sections. The Ralph Loop
// execution system these fed (scripts/ralph-orchestrator.sh,
// scripts/ralph-pipeline.sh, and friends) was removed along with those config
// sections; `ralph run`/`retry`/`abort` themselves are scheduled for full
// removal in the Go-side deletion slice of the same plan. Until then, these
// literals keep `go build`/`go vet` green by matching the config defaults
// that used to live in config.Default().Pipeline / config.Default().Loop.
var runDefaults = struct {
	model, effort, permissionMode                                 string
	maxIterations, maxParallel                                    int
	codexSandbox, codexApprovalPolicy, claudeReviewerModel        string
	implement, selfReview, verify, test, syncDocs, pr, probe, esc string
}{
	model: "opus", effort: "high", permissionMode: "bypassPermissions",
	maxIterations: 20, maxParallel: 4,
	codexSandbox: "workspace-write", codexApprovalPolicy: "on-failure", claudeReviewerModel: "opus",
	implement: "sonnet", selfReview: "opus", verify: "sonnet", test: "sonnet",
	syncDocs: "sonnet", pr: "sonnet", probe: "haiku", esc: "opus",
}

func runPipeline(planPath string, maxIter, maxPar int, preflight, resume, dryRun, unifiedPR bool, maxIterChanged, maxParChanged bool) error {
	var err error

	// Build environment with env > default priority.
	// RALPH_MODEL, RALPH_EFFORT, RALPH_PERMISSION_MODE have no CLI flags;
	// use appendEnvIfMissing so a pre-set env var wins over the literal default.
	env := os.Environ()
	env = appendEnvIfMissing(env, "RALPH_MODEL", runDefaults.model)
	env = appendEnvIfMissing(env, "RALPH_EFFORT", runDefaults.effort)
	env = appendEnvIfMissing(env, "RALPH_PERMISSION_MODE", runDefaults.permissionMode)

	// RALPH_MAX_ITERATIONS and RALPH_MAX_PARALLEL honour CLI > env > default.
	// Use Flags().Changed() (Cobra flag-presence) — NOT the != 0 heuristic —
	// to detect whether the flag was explicitly set. When the flag was set,
	// export the CLI value unconditionally (it wins over any env var). When
	// absent, fall back to the env-or-default value via appendEnvIfMissing.
	if maxIterChanged {
		env = append(env, fmt.Sprintf("RALPH_MAX_ITERATIONS=%d", maxIter))
	} else {
		env = appendEnvIfMissing(env, "RALPH_MAX_ITERATIONS", fmt.Sprintf("%d", runDefaults.maxIterations))
	}
	if maxParChanged {
		env = append(env, fmt.Sprintf("RALPH_MAX_PARALLEL=%d", maxPar))
	} else {
		env = appendEnvIfMissing(env, "RALPH_MAX_PARALLEL", fmt.Sprintf("%d", runDefaults.maxParallel))
	}

	env = appendEnvIfMissing(env, "RALPH_CODEX_SANDBOX", runDefaults.codexSandbox)
	env = appendEnvIfMissing(env, "RALPH_CODEX_APPROVAL_POLICY", runDefaults.codexApprovalPolicy)
	env = appendEnvIfMissing(env, "RALPH_CLAUDE_REVIEWER_MODEL", runDefaults.claudeReviewerModel)

	// Per-phase model routing — propagate literal defaults only when the user
	// has not already set the corresponding env var (env > default).
	env = appendEnvIfMissing(env, "RALPH_IMPLEMENT_MODEL", runDefaults.implement)
	env = appendEnvIfMissing(env, "RALPH_SELF_REVIEW_MODEL", runDefaults.selfReview)
	env = appendEnvIfMissing(env, "RALPH_VERIFY_MODEL", runDefaults.verify)
	env = appendEnvIfMissing(env, "RALPH_TEST_MODEL", runDefaults.test)
	env = appendEnvIfMissing(env, "RALPH_SYNC_DOCS_MODEL", runDefaults.syncDocs)
	env = appendEnvIfMissing(env, "RALPH_PR_MODEL", runDefaults.pr)
	env = appendEnvIfMissing(env, "RALPH_PROBE_MODEL", runDefaults.probe)
	env = appendEnvIfMissing(env, "RALPH_ESCALATION_MODEL", runDefaults.esc)

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
			return "", fmt.Errorf("no directory-based plan found. Specify --plan <directory>")
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
		return "", fmt.Errorf("no directory-based plan found. Specify --plan <directory>")
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
