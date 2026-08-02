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
	Org      OrgConfig      `toml:"org"`
}

// LoopConfig holds Ralph Loop driver settings — which driver runs the
// per-slice pipeline, and how the Codex driver runs (sandbox + approvals).
// Phase 2 of the Codex parity work; see issue #44.
type LoopConfig struct {
	Driver              string `toml:"driver"`
	CodexSandbox        string `toml:"codex_sandbox"`
	CodexApprovalPolicy string `toml:"codex_approval_policy"`
	// ClaudeReviewerModel is the model used by `claude -p` when it plays
	// adversarial reviewer in the cross-review reviewer-inversion path
	// (driver=codex). Lives on LoopConfig so it shares the same env > TOML >
	// default priority as the other Phase 2 knobs.
	ClaudeReviewerModel string `toml:"claude_reviewer_model"`
}

// PipelineConfig holds pipeline execution settings.
type PipelineConfig struct {
	Model          string           `toml:"model"`
	Effort         string           `toml:"effort"`
	MaxIterations  int              `toml:"max_iterations"`
	MaxParallel    int              `toml:"max_parallel"`
	SliceTimeout   string           `toml:"slice_timeout"`
	PermissionMode string           `toml:"permission_mode"`
	Prompts        PromptConfig     `toml:"prompts"`
	Phases         PhaseModelConfig `toml:"phases"`
}

// PhaseModelConfig holds per-phase model routing for the Ralph Loop pipeline.
//
// Values here mirror the RALPH_<PHASE>_MODEL shell defaults in
// scripts/ralph-config.sh and must change in lock-step (see
// .claude/rules/model-routing.md). Precedence at runtime is:
//
//	env RALPH_<PHASE>_MODEL > [pipeline.phases] value > built-in default
//
// Force overrides everything: when RALPH_FORCE_MODEL is set (or
// [pipeline.phases] force is non-empty), every phase resolves to that model.
// Force is intentionally not backfilled in Load() — an empty force is
// semantically meaningful ("no override").
type PhaseModelConfig struct {
	// Implement is the model for the Inner Loop implement/fix phase (default sonnet).
	Implement string `toml:"implement"`
	// SelfReview is the model for the self-review judgment seat (default opus).
	SelfReview string `toml:"self_review"`
	// Verify is the model for spec-compliance / static-analysis (default sonnet).
	Verify string `toml:"verify"`
	// Test is the model for behavioral test execution (default sonnet).
	Test string `toml:"test"`
	// SyncDocs is the model for the documentation-sync phase (default sonnet).
	SyncDocs string `toml:"sync_docs"`
	// PR is the model for the PR-creation agent turn (default sonnet).
	PR string `toml:"pr"`
	// Probe is the model for CLI capability probes (default haiku — cheapest seat).
	Probe string `toml:"probe"`
	// Escalation is the model used for the implement seat when the Outer Loop
	// enters a fix-and-revalidate cycle (outer cycle >= 2) (default opus).
	Escalation string `toml:"escalation"`
	// Force, when non-empty, overrides every phase model for the current run.
	// Use RALPH_FORCE_MODEL=opus to restore pre-routing behaviour in one knob.
	// Not backfilled — empty string means "no override".
	Force string `toml:"force"`
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

// OrgConfig holds the `[org]` envelope settings consumed by the `ralph org`
// verb set (spawn/send/wait/read/stop/status/disband, internal/org) and by
// `ralph doctor`. The fields are validated and stored here so later PRs
// (seat-ification, Lead autonomy, Watchdog) build on a stable, lock-stepped
// foundation.
//
// See docs/plans/active/2026-08-01-org-runtime-mechanism.md (or its archived
// counterpart once the plan lands) for the full design.
type OrgConfig struct {
	// ModelPool is the allowlist of driver/model pairs `ralph org spawn` may
	// launch. Model is a CLI-native model name or alias (e.g. "opus"),
	// passed verbatim to `claude --model` / `codex --model` — aliases are
	// valid CLI values and do not go stale like full model IDs would.
	ModelPool []OrgModelPoolEntry `toml:"model_pool"`
	// DriverPool is the allowlist of driver CLIs org seats may use.
	DriverPool []string `toml:"driver_pool"`
	// Roles maps a role name to the subset of model_pool model names that
	// role is allowed to run under. An absent or empty roles map means "no
	// role restrictions" — the full model_pool is allowed for every role.
	Roles map[string][]string `toml:"roles"`
	// MaxSeats caps concurrently spawned seats per org_id namespace.
	MaxSeats int `toml:"max_seats"`
	// Budget holds wall-clock and fix-round ceilings for org seats. PR①
	// records these values only; enforcement (Watchdog) lands in PR④.
	Budget OrgBudgetConfig `toml:"budget"`
	// DeadmanMinutes is a reserved field for the PR④ Watchdog deadman timer.
	// PR① only stores and round-trips this value; nothing consumes it yet.
	DeadmanMinutes int `toml:"deadman_minutes"`
}

// OrgModelPoolEntry pairs a driver CLI with a CLI-native model name or alias.
type OrgModelPoolEntry struct {
	Driver string `toml:"driver"`
	Model  string `toml:"model"`
}

// OrgBudgetConfig holds wall-clock and fix-round ceilings for org seats.
type OrgBudgetConfig struct {
	SeatWallClockMinutes  int `toml:"seat_wall_clock_minutes"`
	TotalWallClockMinutes int `toml:"total_wall_clock_minutes"`
	MaxFixRounds          int `toml:"max_fix_rounds"`
}

// Default returns a Config with sensible defaults.
func Default() Config {
	return Config{
		Pipeline: PipelineConfig{
			Model:          "opus",
			Effort:         "high",
			MaxIterations:  20,
			MaxParallel:    4,
			SliceTimeout:   "30m",
			PermissionMode: "bypassPermissions",
			Prompts: PromptConfig{
				Dir: ".ralph/prompts",
			},
			Phases: PhaseModelConfig{
				Implement:  "sonnet",
				SelfReview: "opus",
				Verify:     "sonnet",
				Test:       "sonnet",
				SyncDocs:   "sonnet",
				PR:         "sonnet",
				Probe:      "haiku",
				Escalation: "opus",
				Force:      "", // empty = no override; not backfilled in Load()
			},
		},
		Loop: LoopConfig{
			Driver:              "claude",
			CodexSandbox:        "workspace-write",
			CodexApprovalPolicy: "on-failure",
			ClaudeReviewerModel: "opus",
		},
		Doctor: DoctorConfig{
			RequireClaudeCLI: true,
			RequireCodexCLI:  false,
			RequireGo:        false,
		},
		Org: OrgConfig{
			DriverPool: []string{"claude", "codex"},
			ModelPool: []OrgModelPoolEntry{
				{Driver: "claude", Model: "opus"},
				{Driver: "claude", Model: "sonnet"},
				{Driver: "claude", Model: "haiku"},
			},
			Roles:    map[string][]string{},
			MaxSeats: 5,
			Budget: OrgBudgetConfig{
				SeatWallClockMinutes:  30,
				TotalWallClockMinutes: 120,
				MaxFixRounds:          2,
			},
			DeadmanMinutes: 10,
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

// Allowed approval policies recognised by Codex.
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

	// Backfill [pipeline.phases] zero values.
	// Force is intentionally excluded: empty string means "no override".
	if cfg.Pipeline.Phases.Implement == "" {
		cfg.Pipeline.Phases.Implement = Default().Pipeline.Phases.Implement
	}
	if cfg.Pipeline.Phases.SelfReview == "" {
		cfg.Pipeline.Phases.SelfReview = Default().Pipeline.Phases.SelfReview
	}
	if cfg.Pipeline.Phases.Verify == "" {
		cfg.Pipeline.Phases.Verify = Default().Pipeline.Phases.Verify
	}
	if cfg.Pipeline.Phases.Test == "" {
		cfg.Pipeline.Phases.Test = Default().Pipeline.Phases.Test
	}
	if cfg.Pipeline.Phases.SyncDocs == "" {
		cfg.Pipeline.Phases.SyncDocs = Default().Pipeline.Phases.SyncDocs
	}
	if cfg.Pipeline.Phases.PR == "" {
		cfg.Pipeline.Phases.PR = Default().Pipeline.Phases.PR
	}
	if cfg.Pipeline.Phases.Probe == "" {
		cfg.Pipeline.Phases.Probe = Default().Pipeline.Phases.Probe
	}
	if cfg.Pipeline.Phases.Escalation == "" {
		cfg.Pipeline.Phases.Escalation = Default().Pipeline.Phases.Escalation
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
	if cfg.Loop.ClaudeReviewerModel == "" {
		cfg.Loop.ClaudeReviewerModel = Default().Loop.ClaudeReviewerModel
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

	// [org] validation.
	//
	// Unlike PipelineConfig's ints (which silently backfill on zero), Org's
	// fields are validated strictly: toml.Unmarshal only overwrites fields
	// present in the source document (cfg already carries Default()'s Org
	// values before Unmarshal runs), so an absent [org] section — or an
	// absent individual key within a present [org] section — never reaches
	// these checks with a zero/invalid value. Only an *explicit* invalid
	// value (e.g. `max_seats = 0`) can trigger a validation error here.
	if len(cfg.Org.ModelPool) == 0 {
		return cfg, fmt.Errorf("[org].model_pool must not be empty")
	}
	orgDriverSet := make(map[string]bool, len(cfg.Org.DriverPool))
	for _, d := range cfg.Org.DriverPool {
		orgDriverSet[d] = true
	}
	seenModelPoolEntries := make(map[string]bool, len(cfg.Org.ModelPool))
	validOrgModels := make(map[string]bool, len(cfg.Org.ModelPool))
	for _, entry := range cfg.Org.ModelPool {
		if !orgDriverSet[entry.Driver] {
			return cfg, fmt.Errorf("[org].model_pool entry driver %q not present in [org].driver_pool %v", entry.Driver, cfg.Org.DriverPool)
		}
		key := entry.Driver + ":" + entry.Model
		if seenModelPoolEntries[key] {
			return cfg, fmt.Errorf("[org].model_pool duplicate entry %q", key)
		}
		seenModelPoolEntries[key] = true
		validOrgModels[entry.Model] = true
	}
	for role, models := range cfg.Org.Roles {
		for _, m := range models {
			if !validOrgModels[m] {
				return cfg, fmt.Errorf("[org.roles].%s references model %q not present in [org].model_pool", role, m)
			}
		}
	}
	if cfg.Org.MaxSeats < 1 {
		return cfg, fmt.Errorf("[org].max_seats must be >= 1, got %d", cfg.Org.MaxSeats)
	}
	if cfg.Org.Budget.SeatWallClockMinutes < 1 {
		return cfg, fmt.Errorf("[org.budget].seat_wall_clock_minutes must be >= 1, got %d", cfg.Org.Budget.SeatWallClockMinutes)
	}
	if cfg.Org.Budget.TotalWallClockMinutes < 1 {
		return cfg, fmt.Errorf("[org.budget].total_wall_clock_minutes must be >= 1, got %d", cfg.Org.Budget.TotalWallClockMinutes)
	}
	if cfg.Org.Budget.MaxFixRounds < 1 {
		return cfg, fmt.Errorf("[org.budget].max_fix_rounds must be >= 1, got %d", cfg.Org.Budget.MaxFixRounds)
	}

	return cfg, nil
}
