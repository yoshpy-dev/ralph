package config

import (
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

// Config represents the ralph.toml project configuration.
type Config struct {
	Pipeline PipelineConfig `toml:"pipeline"`
	Doctor   DoctorConfig   `toml:"doctor"`
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
	RequireGo        bool `toml:"require_go"`
}

// Default returns a Config with sensible defaults.
func Default() Config {
	return Config{
		Pipeline: PipelineConfig{
			Model:          "claude-sonnet-4-20250514",
			Effort:         "high",
			MaxIterations:  20,
			MaxParallel:    4,
			SliceTimeout:   "30m",
			PermissionMode: "auto",
			Prompts: PromptConfig{
				Dir: ".ralph/prompts",
			},
		},
		Doctor: DoctorConfig{
			RequireClaudeCLI: true,
			RequireGo:        false,
		},
	}
}

// Load reads ralph.toml from the given path, falling back to defaults.
func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
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

	return cfg, nil
}
