package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"slices"
	"strings"

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

// OrgConfig holds the `[org]` table: the envelope and the settings that the
// `ralph org` verbs (internal/org) and `ralph doctor`'s org checks read. Load
// validates the fields, and their defaults are kept in lock-step with
// templates/base/ralph.toml; AgmsgHome's is also compared with
// RALPH_ORG_AGMSG_HOME in scripts/ralph-config.sh (both checks are in
// defaults_sync_test.go). The org runtime's design is
// docs/specs/2026-08-01-org-runtime.md.
type OrgConfig struct {
	// ModelPool is the allowlist of driver/model pairs `ralph org spawn` may
	// launch. Model is a CLI-native model name or alias, passed verbatim to
	// `claude --model` / `codex --model`. Claude entries are stable CLI
	// aliases (e.g. "opus") that do not go stale like full model IDs would.
	// Codex has no aliases, so codex entries are model slugs taken from
	// codex's own models_cache.json and can go stale when codex retires a
	// model.
	ModelPool []OrgModelPoolEntry `toml:"model_pool"`
	// DriverPool is the allowlist of driver CLIs org seats may use.
	DriverPool []string `toml:"driver_pool"`
	// Roles maps a role name to the subset of model_pool model names that
	// role is allowed to run under. An absent or empty roles map means "no
	// role restrictions" — the full model_pool is allowed for every role.
	Roles map[string][]string `toml:"roles"`
	// MaxSeats caps concurrently spawned seats per org_id namespace.
	MaxSeats int `toml:"max_seats"`
	// MaxOrgs caps the number of running orgs across every org_id that shares
	// the same org state dir. `ralph org spawn` enforces it under the manifest
	// lock when a spawn would start an org that is not running yet.
	MaxOrgs int `toml:"max_orgs"`
	// MaxTotalSeats caps the number of active seats summed across every
	// org_id that shares the same org state dir. `ralph org spawn` enforces
	// it under the manifest lock, alongside the per-org MaxSeats.
	MaxTotalSeats int `toml:"max_total_seats"`
	// DeadmanMinutes is how long an ALERT of `ralph org watch` may go
	// unanswered by the leader before the deadman escalates it. The
	// escalation is an ALERT item of the org inbox, which reaches the human
	// like a leader's escalate does: left to the director and the inbox
	// deadlines when [org.director].enabled and the watch accepts work,
	// otherwise sent at once (the inbox section of the /org skill has the
	// details). 0 (or any value below 1) means the deadman never escalates
	// by timeout.
	// The liveness mark of `ralph org watch --all` counts as stale once it is
	// older than the larger of this value (the default 10 when it is below
	// 1) and two watch intervals.
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
	// watch`. See OrgWatchdogConfig's doc comment.
	Watchdog OrgWatchdogConfig `toml:"watchdog"`
	// Director holds the director switch (whether `ralph org escalate`
	// leaves an inbox item to the director and the ack deadline), the
	// auto_merge mark the /director skill reads, the inbox deadlines that
	// `ralph org watch --all` checks, the command `ralph org director start`
	// opens a terminal with instead of herdr, and how the watch wakes the
	// director (wake, wake_cmd). See OrgDirectorConfig's doc comment.
	Director OrgDirectorConfig `toml:"director"`
	// Approval holds how a feature org's plan is approved. See
	// OrgApprovalConfig's doc comment.
	Approval OrgApprovalConfig `toml:"approval"`
}

// OrgPermissionsConfig is the `[org.permissions]` envelope: a driver-
// independent permission-mode enum (autonomous|edits|guarded), applied
// per-role. Default applies to every role without an explicit entry in
// Roles. internal/org maps the resolved mode to driver-native CLI flags
// (e.g. claude: --permission-mode bypassPermissions); codex seats accept
// only guarded until the operator sets CodexVerified after verifying the
// codex flag mapping on their machine (docs/recipes/codex-seat-permissions.md).
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
	// docs/plans/archive/2026-08-02-org-runtime-watchdog.md). The mapping was
	// live-verified on 2026-09-18 against codex-cli 0.154.0
	// (docs/evidence/codex-seat-permissions-2026-09-18.md), but the flags
	// inherit the operator's own ~/.codex/config.toml (approval_policy,
	// sandbox writable_roots), so the default stays false by design: each
	// machine opts in after running docs/recipes/codex-seat-permissions.md.
	CodexVerified bool `toml:"codex_verified"`
}

// OrgModelPoolEntry pairs a driver CLI with a CLI-native model name or
// alias. Claude entries are CLI aliases (stable across model releases,
// e.g. "opus", "sonnet", "haiku"). Codex has no aliases, so codex entries
// are model slugs as listed in codex's own models_cache.json and go stale
// when codex retires a model -- the `ralph doctor` slug check
// (checkCodexModelSlugs, internal/cli/doctor_codex_models.go) warns about
// that.
type OrgModelPoolEntry struct {
	Driver string `toml:"driver"`
	Model  string `toml:"model"`
}

// OrgWatchdogConfig holds the `[org.watchdog]` settings for `ralph org
// watch`'s two layers: a deterministic pulse-layer timer (IntervalSeconds,
// StallMinutes) and an on-demand semantic-judgment watcher layer
// (WatcherEnabled, WatcherModel). See
// docs/plans/archive/2026-08-02-org-runtime-watchdog.md for the full design.
type OrgWatchdogConfig struct {
	// IntervalSeconds is how often the pulse layer evaluates watch
	// conditions (heartbeat stall, process liveness, scope change).
	IntervalSeconds int `toml:"interval_seconds"`
	// StallMinutes is the heartbeat-stall threshold: how long a seat's last
	// manifest event time and herdr state_change_seq may both stay unchanged
	// before the pulse layer treats it as stalled.
	StallMinutes int `toml:"stall_minutes"`
	// WatcherEnabled toggles the on-demand `claude -p` semantic-judgment
	// watcher layer. When false, the pulse layer still runs (ALERT
	// notifications) but never triggers the watcher.
	WatcherEnabled bool `toml:"watcher_enabled"`
	// WatcherModel is the model alias passed to the on-demand watcher
	// invocation (e.g. "haiku"). Required (non-empty) when WatcherEnabled is
	// true.
	WatcherModel string `toml:"watcher_model"`
}

// OrgDirectorConfig holds the `[org.director]` settings of
// docs/specs/2026-10-07-org-multi-org-director.md: the director switch
// Enabled (FR-9), AutoMerge, whether the director merges ready PRs without
// asking (FR-8), the inbox deadlines that `ralph org watch --all` checks
// (FR-6), LaunchCmd, the terminal of a headless director outside herdr
// (FR-9's launch (B)), and Wake and WakeCmd, how `ralph org watch --all`
// wakes the director when a new inbox item arrives (FR-10).
type OrgDirectorConfig struct {
	// Enabled is the director switch. When true (the default), `ralph org
	// escalate` records an item in the org inbox and leaves it to the
	// director or to the ack deadline that `ralph org watch --all` checks, as
	// long as a watch accepts work; with no such watch nobody checks the
	// deadline, so escalate sends the item to the human at once. When false,
	// escalate sends every item to the human at once, as it did before stage
	// 8a. The `ralph org` verbs read it by the same rule as MaxOrgs: from the
	// main worktree's ralph.toml when --config is not given and the ledger
	// is the main worktree's, so an edit to the ralph.toml of a feature org's
	// worktree does not switch it.
	Enabled bool `toml:"enabled"`
	// AutoMerge is the mark the /director skill reads for whether it may,
	// without asking, merge each PR `ralph org merge-order` lists as ready
	// with `ralph org merge` and re-gate the rest with `ralph org regate`
	// (spec FR-8). When false (the default), the director only reports the
	// merge order, and merges are the human's. `ralph org merge` runs the
	// same checks whichever value is set (plan 2026-10-17-org-auto-merge,
	// D1). It is read from the main worktree's ralph.toml by the same rule
	// as Enabled.
	AutoMerge bool `toml:"auto_merge"`
	// AckMinutes is the ack deadline: an open inbox item that has not been
	// sent to the human yet is sent once this long after it was escalated.
	AckMinutes int `toml:"ack_minutes"`
	// ResolveMinutes is the resolve deadline: an item that is not resolved
	// this long after it was escalated is sent to the human once more.
	ResolveMinutes int `toml:"resolve_minutes"`
	// LaunchCmd is the command `ralph org director start` runs to open a
	// terminal for the headless director instead of a herdr workspace (stage
	// 8a-2b). Empty, the default, keeps herdr. It is an argv array: start
	// runs LaunchCmd[0] with the other elements as its arguments, without a
	// shell, after replacing these placeholders in every element
	// (ExpandDirectorLaunchArg):
	//
	//   - {script} (DirectorLaunchPlaceholderScript): the absolute path of the
	//     launch script start writes under the ledger, which checks in with
	//     ralph and then execs the agent
	//   - {script_sh} (DirectorLaunchPlaceholderScriptSh): the same path
	//     single-quoted for a shell, for a terminal that reads its command as
	//     a shell string (tmux's shell-command, Orca's --command)
	//   - {name} (DirectorLaunchPlaceholderName): director_<token>, a name
	//     for the terminal
	//   - {cwd} (DirectorLaunchPlaceholderCwd): the directory the director
	//     runs in, the ledger's main worktree root
	//
	// A placeholder is a "{" and the next "}" with no other brace between
	// them. Load refuses a non-empty LaunchCmd whose first element is empty
	// or whitespace only, that has such a pair outside the four above (a
	// shell brace group such as "{ a; }" included), or that has neither
	// {script} nor {script_sh} in any element, since the launch script would
	// then never run (ValidateDirectorLaunchCmd). A "{" or "}" that is not
	// part of such a pair is passed through as written. The command must
	// return once the terminal is open.
	LaunchCmd []string `toml:"launch_cmd"`
	// Wake is how the director learns of a new open inbox item (stage 8b,
	// FR-10): one or both of these values, in any order:
	//
	//   - pull (DirectorWakePull): the director waits on the inbox itself
	//     with `ralph org wait --inbox`
	//   - push (DirectorWakePush): when a new open inbox item arrives,
	//     `ralph org watch --all` wakes the director once for each item, by
	//     sending it, once per cycle, one fixed line that names the ids of
	//     the items it was not woken for yet, never the message body,
	//     with WakeCmd, or, when WakeCmd is empty, as input to the herdr pane
	//     of a headless director in herdr
	//
	// The default is ["pull"]. The watch acts on the wake of the director's
	// registration in director.jsonl, not on this value: this value is the
	// wake `ralph org director attach` (when --wake is not given) and
	// `ralph org director start` record, so a change takes effect at the next
	// attach or start. Load refuses an empty array, a value other than pull
	// and push, and a value listed twice (ValidateDirectorWake). It accepts
	// push with an empty WakeCmd, since whether a director can be pushed
	// depends on its registration: attach and start check that. Wake is read
	// from the main worktree's ralph.toml by the same rule as Enabled and
	// LaunchCmd (when --config is not given and the ledger is the main
	// worktree's), so an edit to the ralph.toml of a feature org's worktree
	// does not change it.
	Wake []string `toml:"wake"`
	// WakeCmd is the command `ralph org watch --all` runs to wake a director
	// whose registration's wake has push (Wake). Empty, the default, leaves
	// only a headless director in herdr able to be pushed, by input to its
	// herdr pane; a non-empty WakeCmd is used for every registration, that
	// one included. It is an argv array: the watch runs WakeCmd[0] with the
	// other elements as its arguments, without a shell, after replacing these
	// placeholders in every element (ExpandDirectorWakeArg):
	//
	//   - {target} (DirectorWakePlaceholderTarget): where the registration
	//     says to send: the --wake-target of `ralph org director attach`, the
	//     herdr pane of a headless director in herdr, or director_<token>,
	//     the terminal name of a director started with LaunchCmd
	//   - {text} (DirectorWakePlaceholderText): the one fixed line, which
	//     names the new items' ids and never their message body
	//
	// The watch runs WakeCmd in the directory of the ralph.toml it was read
	// from: the main worktree root under the rule below, the directory of
	// the --config file, or else the directory the watch started in, whose
	// ralph.toml it read. A relative WakeCmd[0] that contains "/" is resolved
	// from that directory; a name without "/" is looked up in PATH. The watch
	// reads WakeCmd when it starts, so restart the watch after changing it.
	// WakeCmd is read from the main worktree's ralph.toml by the same rule as
	// Wake, so a leader, who can edit its feature worktree's ralph.toml,
	// cannot change what the watch runs.
	//
	// A placeholder is found by the same rule as in LaunchCmd: a "{" and the
	// next "}" with no other brace between them. Load refuses a non-empty
	// WakeCmd whose first element is empty or whitespace only, whose first
	// element has any placeholder (the first element is the program to run,
	// and {target} comes from director.jsonl, which a leader can write, so a
	// registration must never choose the program), or that has a placeholder
	// outside the two above in any element (ValidateDirectorWakeCmd). A "{"
	// or "}" that is not part of such a pair is passed through as written.
	//
	// {target} comes from the registration (director.jsonl, which a leader
	// can write; for a promoted session, the --wake-target of attach), and
	// attach and the watch check only its shape (not empty, no control
	// character, at most 256 characters, not starting with "-"), so it may
	// hold ";", "$", backquotes, and spaces. Never put {target} inside a
	// string a shell reads (sh -c '...', bash -c, a tmux shell-command, an
	// Orca or terminal command line): give it an element of its own
	// ("{target}", or a flag's value such as "--terminal", "{target}"), or
	// pass it to a script that reads it quoted ("$1").
	WakeCmd []string `toml:"wake_cmd"`
}

// The placeholders of [org.director].launch_cmd (OrgDirectorConfig.LaunchCmd),
// with their braces.
const (
	// DirectorLaunchPlaceholderScript is replaced by the absolute path of
	// the launch script `ralph org director start` writes.
	DirectorLaunchPlaceholderScript = "{script}"
	// DirectorLaunchPlaceholderScriptSh is replaced by the same path
	// single-quoted for a shell.
	DirectorLaunchPlaceholderScriptSh = "{script_sh}"
	// DirectorLaunchPlaceholderName is replaced by director_<token>.
	DirectorLaunchPlaceholderName = "{name}"
	// DirectorLaunchPlaceholderCwd is replaced by the directory the director
	// runs in, the ledger's main worktree root.
	DirectorLaunchPlaceholderCwd = "{cwd}"
)

// DirectorLaunchPlaceholders returns the placeholders of
// [org.director].launch_cmd in a fixed order: {script}, {script_sh}, {name},
// {cwd}. Each call returns a new slice.
func DirectorLaunchPlaceholders() []string {
	return []string{
		DirectorLaunchPlaceholderScript,
		DirectorLaunchPlaceholderScriptSh,
		DirectorLaunchPlaceholderName,
		DirectorLaunchPlaceholderCwd,
	}
}

// The [org.director] keys that the errors of the validation and replacement
// functions below start with.
const (
	directorLaunchCmdKey = "[org.director].launch_cmd"
	directorWakeKey      = "[org.director].wake"
	directorWakeCmdKey   = "[org.director].wake_cmd"
)

// directorCmdPlaceholderPattern finds the placeholders of an element of
// the two argv settings of [org.director], launch_cmd and wake_cmd: a "{"
// and the next "}" with no other brace between them. The validation and the
// replacement of both settings use it, so what Load accepts is what start
// and the watch replace.
var directorCmdPlaceholderPattern = regexp.MustCompile(`\{[^{}]*\}`)

// checkDirectorCmdProgram returns an error naming key, an argv setting of
// [org.director], when first, its first element, is empty or whitespace
// only.
func checkDirectorCmdProgram(key, first string) error {
	if strings.TrimSpace(first) == "" {
		return fmt.Errorf("%s: the first element is the command to run and must not be empty or whitespace only, got %q", key, first)
	}
	return nil
}

// checkDirectorCmdPlaceholders returns an error naming key, an argv setting
// of [org.director], for the first placeholder of cmd, element by element,
// that is not one of placeholders.
func checkDirectorCmdPlaceholders(key string, cmd, placeholders []string) error {
	for i, arg := range cmd {
		for _, placeholder := range directorCmdPlaceholderPattern.FindAllString(arg, -1) {
			if !slices.Contains(placeholders, placeholder) {
				return fmt.Errorf("%s: element %d %q has the unknown placeholder %s; the placeholders are %s",
					key, i, arg, placeholder, strings.Join(placeholders, ", "))
			}
		}
	}
	return nil
}

// expandDirectorCmdArg returns arg, an element of key, an argv setting of
// [org.director], with each placeholder replaced by its entry in values in
// one pass. A placeholder with no entry in values is an error naming key,
// and arg is then returned unchanged.
func expandDirectorCmdArg(key, arg string, values map[string]string) (string, error) {
	missing := ""
	expanded := directorCmdPlaceholderPattern.ReplaceAllStringFunc(arg, func(placeholder string) string {
		value, ok := values[placeholder]
		if !ok {
			if missing == "" {
				missing = placeholder
			}
			return placeholder
		}
		return value
	})
	if missing != "" {
		return arg, fmt.Errorf("%s: %q has the placeholder %s, which has no value", key, arg, missing)
	}
	return expanded, nil
}

// ValidateDirectorLaunchCmd returns an error naming [org.director].launch_cmd
// and the reason when cmd is not empty and its first element is empty or
// whitespace only, an element has a placeholder outside
// DirectorLaunchPlaceholders, or no element has {script} or {script_sh}.
// An empty cmd (start uses herdr) is valid. Load calls it; see
// OrgDirectorConfig.LaunchCmd for the rules.
func ValidateDirectorLaunchCmd(cmd []string) error {
	if len(cmd) == 0 {
		return nil
	}
	if err := checkDirectorCmdProgram(directorLaunchCmdKey, cmd[0]); err != nil {
		return err
	}
	if err := checkDirectorCmdPlaceholders(directorLaunchCmdKey, cmd, DirectorLaunchPlaceholders()); err != nil {
		return err
	}
	for _, arg := range cmd {
		for _, placeholder := range directorCmdPlaceholderPattern.FindAllString(arg, -1) {
			if placeholder == DirectorLaunchPlaceholderScript || placeholder == DirectorLaunchPlaceholderScriptSh {
				return nil
			}
		}
	}
	return fmt.Errorf("%s: no element has %s or %s, so the terminal would never run the launch script; got %q",
		directorLaunchCmdKey, DirectorLaunchPlaceholderScript, DirectorLaunchPlaceholderScriptSh, cmd)
}

// ExpandDirectorLaunchArg returns arg, an element of
// [org.director].launch_cmd, with each placeholder replaced by its entry in
// values, which is keyed by the placeholder with its braces (such as
// DirectorLaunchPlaceholderScript). It finds placeholders by the rule Load
// validates with, and replaces them all in one pass, so a value that itself
// contains a placeholder is not replaced again. A placeholder with no entry
// in values is an error, and arg is then returned unchanged.
func ExpandDirectorLaunchArg(arg string, values map[string]string) (string, error) {
	return expandDirectorCmdArg(directorLaunchCmdKey, arg, values)
}

// The two values of [org.director].wake (OrgDirectorConfig.Wake). The org
// package names the same values for a director's registration.
const (
	// DirectorWakePull is the wake of a director that waits on the inbox
	// itself (`ralph org wait --inbox`).
	DirectorWakePull = "pull"
	// DirectorWakePush is the wake of a director that `ralph org watch
	// --all` wakes when a new open inbox item arrives.
	DirectorWakePush = "push"
)

// ValidateDirectorWake returns an error naming [org.director].wake and the
// reason when wake is empty, has a value other than DirectorWakePull and
// DirectorWakePush, or lists a value twice. The order does not matter. Load
// calls it; see OrgDirectorConfig.Wake for the rules. It is
// ValidateDirectorWakeNamed with the key as the name.
func ValidateDirectorWake(wake []string) error {
	return ValidateDirectorWakeNamed(directorWakeKey, wake)
}

// ValidateDirectorWakeNamed is ValidateDirectorWake for a wake given
// somewhere other than [org.director].wake: its errors start with name in
// place of the key, such as "--wake" for the flag of `ralph org director
// attach`, whose value is split on "," into wake (plan
// 2026-10-18-org-director-wake, D3). The rules and the rest of each error's
// text are ValidateDirectorWake's.
func ValidateDirectorWakeNamed(name string, wake []string) error {
	if len(wake) == 0 {
		return fmt.Errorf("%s: the array must not be empty; the values are %q and %q, one or both", name, DirectorWakePull, DirectorWakePush)
	}
	for i, value := range wake {
		if value != DirectorWakePull && value != DirectorWakePush {
			return fmt.Errorf("%s: element %d %q is not a wake value; the values are %q and %q", name, i, value, DirectorWakePull, DirectorWakePush)
		}
		if first := slices.Index(wake, value); first < i {
			return fmt.Errorf("%s: element %d %q repeats element %d; list each value once", name, i, value, first)
		}
	}
	return nil
}

// The placeholders of [org.director].wake_cmd (OrgDirectorConfig.WakeCmd),
// with their braces.
const (
	// DirectorWakePlaceholderTarget is replaced by where the director's
	// registration says to send the wake line.
	DirectorWakePlaceholderTarget = "{target}"
	// DirectorWakePlaceholderText is replaced by the one fixed wake line.
	DirectorWakePlaceholderText = "{text}"
)

// DirectorWakePlaceholders returns the placeholders of
// [org.director].wake_cmd in a fixed order: {target}, {text}. Each call
// returns a new slice.
func DirectorWakePlaceholders() []string {
	return []string{
		DirectorWakePlaceholderTarget,
		DirectorWakePlaceholderText,
	}
}

// ValidateDirectorWakeCmd returns an error naming [org.director].wake_cmd
// and the reason when cmd is not empty and its first element is empty or
// whitespace only, its first element has any placeholder, or an element has
// a placeholder outside DirectorWakePlaceholders. An empty cmd is valid.
// Load calls it; see OrgDirectorConfig.WakeCmd for the rules.
func ValidateDirectorWakeCmd(cmd []string) error {
	if len(cmd) == 0 {
		return nil
	}
	if err := checkDirectorCmdProgram(directorWakeCmdKey, cmd[0]); err != nil {
		return err
	}
	if directorCmdPlaceholderPattern.MatchString(cmd[0]) {
		return fmt.Errorf("%s: the first element is the command to run and must not contain a placeholder (a director's registration must never choose the program), got %q", directorWakeCmdKey, cmd[0])
	}
	return checkDirectorCmdPlaceholders(directorWakeCmdKey, cmd, DirectorWakePlaceholders())
}

// ExpandDirectorWakeArg returns arg, an element of [org.director].wake_cmd,
// with each placeholder replaced by its entry in values, which is keyed by
// the placeholder with its braces (such as DirectorWakePlaceholderTarget).
// It finds placeholders by the rule Load validates with, and replaces them
// all in one pass, so a value that itself contains a placeholder is not
// replaced again. A placeholder with no entry in values is an error, and arg
// is then returned unchanged.
func ExpandDirectorWakeArg(arg string, values map[string]string) (string, error) {
	return expandDirectorCmdArg(directorWakeCmdKey, arg, values)
}

// The two values of [org.approval].mode.
const (
	// ApprovalModePerPlan makes the implementer of a feature org wait for a
	// human to approve the feature's own plan (`ralph org approve`).
	ApprovalModePerPlan = "per-plan"
	// ApprovalModeSplit lets a feature org proceed on the approval of the
	// split plan alone, as before [org.approval] existed.
	ApprovalModeSplit = "split"
)

// OrgApprovalConfig holds the `[org.approval]` setting of how the plan of a
// feature org is approved (FR-7 of
// docs/specs/2026-10-07-org-multi-org-director.md). Only `ralph org start
// --plan` reads it, and it records the mode in the org's feature binding
// (internal/org's FeatureBinding.Approval), so a change takes effect from
// the next org started. spawn, send and approve read the recorded mode, never
// this setting, so the ralph.toml of the directory a seat runs in cannot turn
// the approval off for a running org.
type OrgApprovalConfig struct {
	// Mode is ApprovalModePerPlan (a human approves each feature's plan
	// before its implementer starts) or ApprovalModeSplit (the approval of
	// the split plan is enough).
	Mode string `toml:"mode"`
}

// Default returns a Config with sensible defaults.
func Default() Config {
	return Config{
		Doctor: DoctorConfig{
			RequireClaudeCLI: false,
			RequireCodexCLI:  false,
			RequireGo:        false,
		},
		Org: OrgConfig{
			DriverPool: []string{"claude", "codex"},
			ModelPool: []OrgModelPoolEntry{
				{Driver: "claude", Model: "fable"},
				{Driver: "claude", Model: "opus"},
				{Driver: "claude", Model: "sonnet"},
				{Driver: "claude", Model: "haiku"},
				{Driver: "codex", Model: "gpt-6-astra"},
				{Driver: "codex", Model: "gpt-5.6-sol"},
				{Driver: "codex", Model: "gpt-5.6-terra"},
				{Driver: "codex", Model: "gpt-5.6-luna"},
			},
			Roles:          map[string][]string{},
			MaxSeats:       5,
			MaxOrgs:        10,
			MaxTotalSeats:  30,
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
			Director: OrgDirectorConfig{
				Enabled:        true,
				AutoMerge:      false,
				AckMinutes:     10,
				ResolveMinutes: 60,
				Wake:           []string{DirectorWakePull},
			},
			Approval: OrgApprovalConfig{
				Mode: ApprovalModePerPlan,
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

// orgPoolKeysPresent reports whether the source document explicitly sets
// [org].driver_pool and/or [org].model_pool. toml.Unmarshal into a
// Default()-populated Config cannot answer this on its own (an absent key
// and an explicit key that happens to match the default look identical
// afterwards), so this does a second, minimal unmarshal into a probe struct
// with pointer fields — go-toml/v2 leaves an unset pointer field nil, which
// is how "key absent" becomes observable.
func orgPoolKeysPresent(data []byte) (bool, bool, error) {
	var probe struct {
		Org struct {
			DriverPool *[]string            `toml:"driver_pool"`
			ModelPool  *[]OrgModelPoolEntry `toml:"model_pool"`
		} `toml:"org"`
	}
	if err := toml.Unmarshal(data, &probe); err != nil {
		return false, false, err
	}
	return probe.Org.DriverPool != nil, probe.Org.ModelPool != nil, nil
}

// filterModelPoolByDrivers returns the subset of pool whose Driver is present
// in drivers, preserving pool's declared order.
func filterModelPoolByDrivers(pool []OrgModelPoolEntry, drivers []string) []OrgModelPoolEntry {
	driverSet := make(map[string]bool, len(drivers))
	for _, d := range drivers {
		driverSet[d] = true
	}
	filtered := make([]OrgModelPoolEntry, 0, len(pool))
	for _, entry := range pool {
		if driverSet[entry.Driver] {
			filtered = append(filtered, entry)
		}
	}
	return filtered
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

	driverPoolSet, modelPoolSet, err := orgPoolKeysPresent(data)
	if err != nil {
		return cfg, err
	}
	if driverPoolSet && !modelPoolSet {
		// The document narrows [org].driver_pool but leaves [org].model_pool
		// unset, so cfg.Org.ModelPool is still Default()'s pool (including
		// any driver Default()'s driver_pool covers but this document's
		// narrower driver_pool doesn't). Inherit the default pool filtered to
		// the drivers this document actually declared, so e.g.
		// `driver_pool = ["claude"]` alone keeps loading even as Default()
		// grows more non-claude default model_pool entries over time.
		cfg.Org.ModelPool = filterModelPoolByDrivers(Default().Org.ModelPool, cfg.Org.DriverPool)
		if len(cfg.Org.ModelPool) == 0 {
			// Name the key the document actually wrote: the empty pool is a
			// consequence of this driver_pool, not of a model_pool the
			// document never set.
			return cfg, fmt.Errorf("[org].driver_pool %v has no default [org].model_pool entries; set [org].model_pool explicitly", cfg.Org.DriverPool)
		}
	}

	// [org] validation.
	//
	// Unlike PipelineConfig's ints (which silently backfill on zero), Org's
	// fields are validated strictly: toml.Unmarshal only overwrites fields
	// present in the source document (cfg already carries Default()'s Org
	// values before Unmarshal runs), so an absent [org] section — or an
	// absent individual key within a present [org] section — never reaches
	// these checks with a zero/invalid value, with one deliberate exception:
	// when driver_pool is set but model_pool is not, the inherited default
	// model_pool is filtered to the declared driver_pool above (so a
	// claude-only driver_pool override doesn't inherit codex's default
	// entries and fail the driver-membership check below). A document that
	// sets model_pool explicitly skips that filtering and is validated
	// strictly as before. Only an *explicit* invalid value (e.g. `max_seats
	// = 0`, or an explicit model_pool entry naming a driver outside
	// driver_pool) can trigger a validation error here.
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
	if cfg.Org.MaxOrgs < 1 {
		return cfg, fmt.Errorf("[org].max_orgs must be >= 1, got %d", cfg.Org.MaxOrgs)
	}
	if cfg.Org.MaxTotalSeats < 1 {
		return cfg, fmt.Errorf("[org].max_total_seats must be >= 1, got %d", cfg.Org.MaxTotalSeats)
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

	// [org.watchdog] validation. Same strict pattern as [org].max_seats
	// above: cfg already carries Default()'s Watchdog values before
	// Unmarshal runs, so an absent [org.watchdog] section (or an absent key
	// within a present one) never reaches these checks with a zero/invalid
	// value.
	if cfg.Org.Watchdog.IntervalSeconds < 1 {
		return cfg, fmt.Errorf("[org.watchdog].interval_seconds must be >= 1, got %d", cfg.Org.Watchdog.IntervalSeconds)
	}
	if cfg.Org.Watchdog.StallMinutes < 1 {
		return cfg, fmt.Errorf("[org.watchdog].stall_minutes must be >= 1, got %d", cfg.Org.Watchdog.StallMinutes)
	}
	if cfg.Org.Watchdog.WatcherEnabled && cfg.Org.Watchdog.WatcherModel == "" {
		return cfg, fmt.Errorf("[org.watchdog].watcher_model must be non-empty when watcher_enabled is true")
	}

	// [org.director] validation. Same strict pattern as [org.watchdog]
	// above: an absent [org.director] section (or an absent key within a
	// present one) keeps Default()'s value. resolve_minutes may be shorter
	// than ack_minutes; the two deadlines are checked independently. An
	// absent or empty launch_cmd (start uses herdr) is valid; a non-empty one
	// is checked by ValidateDirectorLaunchCmd. An absent wake keeps
	// Default()'s ["pull"], and an explicit `wake = []` reaches
	// ValidateDirectorWake as empty and fails it. wake_cmd is checked like
	// launch_cmd, by ValidateDirectorWakeCmd; a wake with push and an empty
	// wake_cmd is valid here (attach and start check whether a director can
	// be pushed). enabled and auto_merge need no check here: toml.Unmarshal
	// already refuses a value that is not a bool.
	if cfg.Org.Director.AckMinutes < 1 {
		return cfg, fmt.Errorf("[org.director].ack_minutes must be >= 1, got %d", cfg.Org.Director.AckMinutes)
	}
	if cfg.Org.Director.ResolveMinutes < 1 {
		return cfg, fmt.Errorf("[org.director].resolve_minutes must be >= 1, got %d", cfg.Org.Director.ResolveMinutes)
	}
	if err := ValidateDirectorLaunchCmd(cfg.Org.Director.LaunchCmd); err != nil {
		return cfg, err
	}
	if err := ValidateDirectorWake(cfg.Org.Director.Wake); err != nil {
		return cfg, err
	}
	if err := ValidateDirectorWakeCmd(cfg.Org.Director.WakeCmd); err != nil {
		return cfg, err
	}

	// [org.approval] validation. Same strict pattern as [org.director]
	// above: an absent [org.approval] section (or an absent mode) keeps
	// Default()'s per-plan. Unlike AgmsgHome, an explicit `mode = ""` is not
	// backfilled: it is an error, like any value other than the two modes.
	if mode := cfg.Org.Approval.Mode; mode != ApprovalModePerPlan && mode != ApprovalModeSplit {
		return cfg, fmt.Errorf("[org.approval].mode %q must be %q or %q", mode, ApprovalModePerPlan, ApprovalModeSplit)
	}

	return cfg, nil
}
