package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

// Config represents the ralph.toml project configuration.
type Config struct {
	Pipeline PipelineConfig `toml:"pipeline"`
	Loop     LoopConfig     `toml:"loop"`
	Doctor   DoctorConfig   `toml:"doctor"`
}

// LoopConfig holds Ralph Loop driver settings — which CLI drives the
// per-slice pipeline, and how the Codex driver runs (sandbox + approvals).
// Phase 2 of the Codex CLI parity work; see issue #44.
type LoopConfig struct {
	Driver              string `toml:"driver"`
	CodexSandbox        string `toml:"codex_sandbox"`
	CodexApprovalPolicy string `toml:"codex_approval_policy"`
}

// PipelineConfig holds pipeline execution settings.
type PipelineConfig struct {
	Model          string       `toml:"model"`
	Effort         string       `toml:"effort"`
	MaxIterations  int          `toml:"max_iterations"`
	MaxParallel    int          `toml:"max_parallel"`
	SliceTimeout   string       `toml:"slice_timeout"`
	PermissionMode string       `toml:"permission_mode"`
	Prompts        PromptConfig `toml:"prompts"`
}

// PromptConfig holds prompt template settings.
type PromptConfig struct {
	Dir string `toml:"dir"`
}

// DoctorConfig holds doctor check settings.
type DoctorConfig struct {
	RequireClaudeCLI bool `toml:"require_claude_cli"`
	RequireCodexCLI  bool `toml:"require_codex_cli"`
	RequireGo        bool `toml:"require_go"`
}

// Default returns a Config with sensible defaults.
func Default() Config {
	return Config{
		Pipeline: PipelineConfig{
			Model:          "claude-opus-4-7",
			Effort:         "xhigh",
			MaxIterations:  20,
			MaxParallel:    4,
			SliceTimeout:   "30m",
			PermissionMode: "auto",
			Prompts: PromptConfig{
				Dir: ".ralph/prompts",
			},
		},
		Loop: LoopConfig{
			Driver:              "claude",
			CodexSandbox:        "workspace-write",
			CodexApprovalPolicy: "on-failure",
		},
		Doctor: DoctorConfig{
			RequireClaudeCLI: true,
			RequireCodexCLI:  false,
			RequireGo:        false,
		},
	}
}

// Allowed driver values for [loop].driver.
var loopDriverAllowed = map[string]bool{
	"claude": true,
	"codex":  true,
}

// Allowed sandbox values, mirroring `codex exec -s` choices.
var codexSandboxAllowed = map[string]bool{
	"read-only":          true,
	"workspace-write":    true,
	"danger-full-access": true,
}

// Allowed approval policies recognised by Codex CLI.
var codexApprovalAllowed = map[string]bool{
	"untrusted":  true,
	"on-failure": true,
	"on-request": true,
	"never":      true,
}

// Load reads ralph.toml from the given path, falling back to defaults.
func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	// Apply defaults for zero values.
	if cfg.Pipeline.Model == "" {
		cfg.Pipeline.Model = Default().Pipeline.Model
	}
	if cfg.Pipeline.Effort == "" {
		cfg.Pipeline.Effort = Default().Pipeline.Effort
	}
	if cfg.Pipeline.MaxIterations == 0 {
		cfg.Pipeline.MaxIterations = Default().Pipeline.MaxIterations
	}
	if cfg.Pipeline.MaxParallel == 0 {
		cfg.Pipeline.MaxParallel = Default().Pipeline.MaxParallel
	}
	if cfg.Pipeline.SliceTimeout == "" {
		cfg.Pipeline.SliceTimeout = Default().Pipeline.SliceTimeout
	}
	if cfg.Pipeline.PermissionMode == "" {
		cfg.Pipeline.PermissionMode = Default().Pipeline.PermissionMode
	}
	if cfg.Pipeline.Prompts.Dir == "" {
		cfg.Pipeline.Prompts.Dir = Default().Pipeline.Prompts.Dir
	}

	if cfg.Loop.Driver == "" {
		cfg.Loop.Driver = Default().Loop.Driver
	}
	if cfg.Loop.CodexSandbox == "" {
		cfg.Loop.CodexSandbox = Default().Loop.CodexSandbox
	}
	if cfg.Loop.CodexApprovalPolicy == "" {
		cfg.Loop.CodexApprovalPolicy = Default().Loop.CodexApprovalPolicy
	}

	if !loopDriverAllowed[cfg.Loop.Driver] {
		return cfg, fmt.Errorf("invalid [loop].driver %q (must be claude or codex)", cfg.Loop.Driver)
	}
	if !codexSandboxAllowed[cfg.Loop.CodexSandbox] {
		return cfg, fmt.Errorf("invalid [loop].codex_sandbox %q (must be read-only, workspace-write, or danger-full-access)", cfg.Loop.CodexSandbox)
	}
	if !codexApprovalAllowed[cfg.Loop.CodexApprovalPolicy] {
		return cfg, fmt.Errorf("invalid [loop].codex_approval_policy %q (must be untrusted, on-failure, on-request, or never)", cfg.Loop.CodexApprovalPolicy)
	}

	return cfg, nil
}
