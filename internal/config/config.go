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
	Doctor DoctorConfig `toml:"doctor"`
	Org    OrgConfig    `toml:"org"`
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
	// AgmsgHome is the agmsg installation directory (a collection of
	// scripts, not a single binary -- see internal/org/driver/agmsg.go).
	// The runtime-effective value is resolved by
	// driver.ResolveAgmsgHome(cfg.Org.AgmsgHome), which lets env
	// RALPH_ORG_AGMSG_HOME override this config value.
	AgmsgHome string `toml:"agmsg_home"`
	// Permissions holds the role-scoped seat permission-mode envelope
	// (internal/org's ResolvePermissionMode/permissionArgsForDriver consume
	// this at spawn time to pick the driver-native permission flag). See
	// OrgPermissionsConfig's doc comment.
	Permissions OrgPermissionsConfig `toml:"permissions"`
	// Watchdog holds the pulse-layer/watcher-layer settings for `ralph org
	// watch` (PR④). See OrgWatchdogConfig's doc comment.
	Watchdog OrgWatchdogConfig `toml:"watchdog"`
}

// OrgPermissionsConfig is the `[org.permissions]` envelope: a driver-
// independent permission-mode enum (autonomous|edits|guarded), applied
// per-role. Default applies to every role without an explicit entry in
// Roles. internal/org maps the resolved mode to driver-native CLI flags
// (e.g. claude: --permission-mode bypassPermissions); codex seats only
// accept guarded until codex's own interactive permission flags are
// live-verified (fail-closed -- see docs/tech-debt/README.md).
type OrgPermissionsConfig struct {
	// Default is the permission mode every role uses unless overridden in
	// Roles. One of "autonomous" | "edits" | "guarded".
	Default string `toml:"default"`
	// Roles maps a role name to a permission-mode override. A role absent
	// from this map uses Default.
	Roles map[string]string `toml:"roles"`
	// CodexVerified gates whether internal/org's permissionArgsForDriver
	// maps codex's autonomous/edits modes to real CLI flags (`--sandbox
	// workspace-write --ask-for-approval never` / `--sandbox
	// workspace-write`) instead of fail-closed-rejecting them (PR④ AC-8,
	// docs/plans/active/2026-08-02-org-runtime-watchdog.md). codex's
	// interactive sandbox/approval flags have not been live-verified against
	// a real codex seat as of this field's introduction -- only Slice 5's
	// live smoke does that. Default false keeps codex fail-closed (guarded
	// only) until an operator who has live-verified their installed codex
	// version's flags flips this on explicitly; see docs/tech-debt/README.md
	// for the verification follow-up this flag exists to close out.
	CodexVerified bool `toml:"codex_verified"`
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

// OrgWatchdogConfig holds the `[org.watchdog]` settings for `ralph org
// watch`'s two layers: a deterministic pulse-layer timer (IntervalSeconds,
// StallMinutes) and an on-demand semantic-judgment watcher layer
// (WatcherEnabled, WatcherModel). See
// docs/plans/active/2026-08-02-org-runtime-watchdog.md for the full design.
type OrgWatchdogConfig struct {
	// IntervalSeconds is how often the pulse layer evaluates watch
	// conditions (heartbeat stall, process liveness, budget, scope change).
	IntervalSeconds int `toml:"interval_seconds"`
	// StallMinutes is the heartbeat-stall threshold: how long a seat's last
	// manifest event time and herdr state_change_seq may both stay unchanged
	// before the pulse layer treats it as stalled.
	StallMinutes int `toml:"stall_minutes"`
	// WatcherEnabled toggles the on-demand `claude -p` semantic-judgment
	// watcher layer. When false, the pulse layer still runs (budget cutoff,
	// ALERT notifications) but never triggers the watcher.
	WatcherEnabled bool `toml:"watcher_enabled"`
	// WatcherModel is the model alias passed to the on-demand watcher
	// invocation (e.g. "haiku"). Required (non-empty) when WatcherEnabled is
	// true.
	WatcherModel string `toml:"watcher_model"`
}

// Default returns a Config with sensible defaults.
func Default() Config {
	return Config{
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
			AgmsgHome:      "~/.agents/skills/agmsg",
			Permissions: OrgPermissionsConfig{
				Default:       "autonomous",
				Roles:         map[string]string{},
				CodexVerified: false,
			},
			Watchdog: OrgWatchdogConfig{
				IntervalSeconds: 30,
				StallMinutes:    15,
				WatcherEnabled:  true,
				WatcherModel:    "haiku",
			},
		},
	}
}

// orgPermissionModeAllowed is the enum [org.permissions].default and every
// [org.permissions.roles] value must belong to. The canonical, grep-able
// definition of this same three-value enum is
// internal/org.PermissionModeAutonomous/Edits/Guarded (self-review
// MEDIUM-4) -- this package cannot import internal/org to reuse it directly
// (internal/org already imports internal/config, so the reverse would be an
// import cycle), hence the literal re-spelling here. Keep these three
// string values in sync with internal/org/permissions.go's constants by
// hand if the enum ever changes.
var orgPermissionModeAllowed = map[string]bool{
	"autonomous": true,
	"edits":      true,
	"guarded":    true,
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
	// AgmsgHome is a string default, so (unlike the strict-validation fields
	// above) it follows the same explicit zero-value backfill pattern as
	// PipelineConfig: an explicit `agmsg_home = ""` in the source document
	// falls back to the default rather than being treated as a validation
	// error.
	if cfg.Org.AgmsgHome == "" {
		cfg.Org.AgmsgHome = Default().Org.AgmsgHome
	}

	// [org.permissions] validation. Default follows AgmsgHome's zero-value
	// backfill pattern (an absent or explicitly empty `default` falls back
	// rather than erroring), but once a value is present -- backfilled or
	// explicit -- it must belong to orgPermissionModeAllowed. Roles entries
	// are never backfilled (an absent role means "use Default"); each
	// present entry is validated the same way.
	if cfg.Org.Permissions.Default == "" {
		cfg.Org.Permissions.Default = Default().Org.Permissions.Default
	}
	if !orgPermissionModeAllowed[cfg.Org.Permissions.Default] {
		return cfg, fmt.Errorf("[org.permissions].default %q must be one of autonomous, edits, or guarded", cfg.Org.Permissions.Default)
	}
	for role, mode := range cfg.Org.Permissions.Roles {
		if !orgPermissionModeAllowed[mode] {
			return cfg, fmt.Errorf("[org.permissions.roles].%s %q must be one of autonomous, edits, or guarded", role, mode)
		}
	}

	// [org.watchdog] validation. Same strict pattern as [org.budget]: cfg
	// already carries Default()'s Watchdog values before Unmarshal runs, so
	// an absent [org.watchdog] section (or an absent key within a present
	// one) never reaches these checks with a zero/invalid value.
	if cfg.Org.Watchdog.IntervalSeconds < 1 {
		return cfg, fmt.Errorf("[org.watchdog].interval_seconds must be >= 1, got %d", cfg.Org.Watchdog.IntervalSeconds)
	}
	if cfg.Org.Watchdog.StallMinutes < 1 {
		return cfg, fmt.Errorf("[org.watchdog].stall_minutes must be >= 1, got %d", cfg.Org.Watchdog.StallMinutes)
	}
	if cfg.Org.Watchdog.WatcherEnabled && cfg.Org.Watchdog.WatcherModel == "" {
		return cfg, fmt.Errorf("[org.watchdog].watcher_model must be non-empty when watcher_enabled is true")
	}

	return cfg, nil
}
