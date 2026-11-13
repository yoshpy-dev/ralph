package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// newOrgCmd wires the `ralph org` verb set (spawn/send/wait/read/stop/
// status/disband). All business logic (envelope validation, saga engine,
// manifest/receipt bookkeeping) lives in internal/org -- this file is thin
// flag parsing plus wiring: bind CLI flags, construct an org.Org, call a
// method, format the result.
func newOrgCmd() *cobra.Command {
	var orgID, stateDir, configPath string

	cmd := &cobra.Command{
		Use:   "org",
		Short: "Manage org-runtime seats (spawn, send, wait, read, stop, status, disband)",
		Long: "ralph org drives the org-runtime mechanism layer: spawning herdr/agmsg-backed\n" +
			"seats within an org_id namespace, sending them messages, waiting on their\n" +
			"state, reading their pane output, stopping them, showing roster status, and\n" +
			"disbanding an entire org_id. Every verb records its outcome to an\n" +
			"append-only manifest so `ralph org status` works even with herdr/agmsg\n" +
			"absent or stopped. `ralph org escalate` and `ralph org inbox` keep the\n" +
			"org inbox, which every org in the ledger shares, next to the manifest,\n" +
			"and `ralph org approve` records a human's approval of the plan a feature\n" +
			"org's leader asked for there. `ralph org gate` runs the verification\n" +
			"scripts on the head a feature org's leader pushes and records which head\n" +
			"passed on which base, and `ralph org gate accept` records a human's\n" +
			"acceptance of a branch's latest gate failure. `ralph org merge-order`\n" +
			"lists the feature branches in the order their PRs may be merged, from\n" +
			"those records and origin's branches. `ralph org regate` merges the\n" +
			"default branch into a disbanded feature org's branch, runs the gate on\n" +
			"it, and pushes it, so a branch another PR's merge moved behind is ready\n" +
			"again. `ralph org merge` merges, with gh, the PR of a branch\n" +
			"merge-order lists as ready or accepted, once it has checked the PR, its\n" +
			"checks, and GitHub's protection of the default branch again, records\n" +
			"the merge, and then cleans up the branch's feature worktree.\n" +
			"`ralph org cleanup` cleans up the feature worktree of\n" +
			"a branch whose PR was merged on GitHub. `ralph org director`\n" +
			"registers the interactive session the human talks to as the director\n" +
			"of the ledger, or launches a headless director in herdr (ralph org\n" +
			"director start), or with [org.director].launch_cmd in the terminal\n" +
			"that command opens. `ralph org followup` records the issues the\n" +
			"director asks the human about only once another branch's PR is merged\n" +
			"(add, list, done), and `ralph status` and `ralph org merge-order` show\n" +
			"which of them are ready. `ralph org resume` types a fixed sentence into\n" +
			"the seats whose CLI stopped on an API error (a connection failure or a\n" +
			"usage limit), so they go on where they stopped.",
	}

	cmd.PersistentFlags().StringVar(&orgID, "org-id", "", "org execution namespace (required by every verb except stop --all, disband --all, merge-order, merge, cleanup, regate, wait --inbox, watch, approve, gate accept, the director verbs, the inbox verbs, and the followup verbs, which refuse it, and start --plan, which defaults it to the feature's slug; optional for resume, which without it looks at the seats of every org and at the headless directors; with --plan it is at most 20 characters)")
	cmd.PersistentFlags().StringVar(&stateDir, "state-dir", "", "org manifest/receipts/inbox state directory (default: resolved by org.ResolveOrgStateDir -- [org].state_dir in the ralph.local.toml at the main worktree's root, else the main worktree's .harness/state/org (shared by its linked worktrees), else the enclosing git toplevel's .harness/state/org, else cwd's .harness/state/org)")
	cmd.PersistentFlags().StringVar(&configPath, "config", "", "path to ralph.toml (default: ./ralph.toml if present, else built-in defaults; without --config, spawn and start read the org-wide limits, [org.role_models] and driver_pool (see ralph org spawn --help), start --plan also [org.approval].mode and the four keys of [reviewer] (see ralph org start --help), escalate and director attach read [org.director].enabled (director attach also [org.director].wake and wake_cmd), director status reads it and [org.director].auto_merge, merge reads auto_merge, director start reads enabled with [org.permissions], [org.role_models], driver_pool and [org.director].launch_cmd, wake and wake_cmd, and watch --all reads [org.director].wake and wake_cmd (and runs wake_cmd in the main worktree root), from the main worktree's ralph.toml when the ledger is the main worktree's; ralph status, which has no --config, reads enabled and auto_merge by the same rule; reseat reads what start --plan reads: those limits, [org.role_models] and driver_pool, [org.approval].mode, and the four keys of [reviewer])")

	cmd.AddCommand(
		newOrgSpawnCmd(&orgID, &stateDir, &configPath),
		newOrgStartCmd(&orgID, &stateDir, &configPath),
		newOrgSendCmd(&orgID, &stateDir, &configPath),
		newOrgWaitCmd(&orgID, &stateDir, &configPath),
		newOrgReadCmd(&orgID, &stateDir, &configPath),
		newOrgResumeCmd(&orgID, &stateDir, &configPath),
		newOrgStopCmd(&orgID, &stateDir, &configPath),
		newOrgReseatCmd(&orgID, &stateDir, &configPath),
		newOrgStatusCmd(&orgID, &stateDir, &configPath),
		newOrgDisbandCmd(&orgID, &stateDir, &configPath),
		newOrgReportCmd(&orgID, &stateDir, &configPath),
		newOrgWatchCmd(&orgID, &stateDir, &configPath),
		newOrgEscalateCmd(&orgID, &stateDir, &configPath),
		newOrgInboxCmd(&stateDir, &configPath),
		newOrgFollowupCmd(&stateDir),
		newOrgApproveCmd(&stateDir, &configPath),
		newOrgDirectorCmd(&stateDir, &configPath),
		newOrgGateCmd(&orgID, &stateDir, &configPath),
		newOrgRegateCmd(&stateDir, &configPath),
		newOrgMergeOrderCmd(&stateDir, &configPath),
		newOrgMergeCmd(&stateDir, &configPath),
		newOrgCleanupCmd(&stateDir, &configPath),
	)

	return cmd
}

// requireOrgID returns an error unless orgID is non-blank and shaped like a
// safe identifier (org.ValidateIdentifier) -- the shared --org-id validation
// every verb needs (global required flag, checked manually rather than via
// cobra's MarkPersistentFlagRequired so tests and error messages stay simple
// and uniform). Shape validation runs here, before an org.Org is even
// constructed, as a second gate alongside (*org.Org).Spawn's own check --
// every CLI entry point that turns an org_id into a path (directly, via
// spawn, or indirectly, via any state-dir lookup) rejects a malformed value
// before it reaches that point.
func requireOrgID(orgID string) error {
	if strings.TrimSpace(orgID) == "" {
		return fmt.Errorf("org: --org-id is required")
	}
	return org.ValidateIdentifier("org_id", orgID)
}

// requireSeatIdentifier returns an error unless value is non-blank and
// shaped like a safe identifier (org.ValidateIdentifier) -- the shared
// validation for every CLI flag that names a target seat id (spawn's --id,
// send's --to, wait/read/stop's --seat). flag is used only in the blank-value
// error message so each call site keeps its own flag name in diagnostics.
func requireSeatIdentifier(flag, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("org: %s is required", flag)
	}
	return org.ValidateIdentifier("seat_id", value)
}

// newOrgRuntime constructs an org.Org wired to real driver adapters
// (driver.ExecRunner, which shells out to the herdr/agmsg binaries on PATH)
// and manifest/receipt stores rooted at the resolved state directory. cmd is
// used by resolveOrgLedger to detect whether --state-dir was explicitly
// passed (cmd.Flags().Changed("state-dir")) for org.ResolveOrgStateDir's
// flag > config > git-main-worktree > git-toplevel > cwd precedence --
// see that function's doc comment for the full rationale (fixes the
// leader/operator cwd-split, tech-debt "state-dir の cwd 相対解決", and
// shares one ledger across linked worktrees) -- and to print the notes of
// checkRetiredOrgStateDirEnv and guardLegacyOrgStateDir. access is the
// calling verb's orgLedgerAccess: a mutating verb gets their refusals as
// the error, before any manifest, herdr, or agmsg call, and so does a ledger
// that cannot be resolved. A caller that also needs the resolved
// config.OrgConfig for its own purposes beyond wiring (e.g. the driver and
// model org.ResolveRoleModel takes from [org.role_models]) reads it back off
// the returned *org.Org's exported Config field rather than newOrgRuntime
// returning a second value.
func newOrgRuntime(cmd *cobra.Command, stateDir, configPath string, access orgLedgerAccess) (*org.Org, error) {
	resolvedStateDir, _, err := resolveOrgLedger(cmd, stateDir, access)
	if err != nil {
		return nil, err
	}
	return newOrgRuntimeAt(resolvedStateDir, configPath)
}

// newOrgSpawnRuntimeAt is newOrgRuntime for the verbs that start seats
// (`ralph org spawn` and `ralph org start <task>`). It also replaces the
// config's MaxOrgs and MaxTotalSeats, and the RoleModels and DriverPool the
// seat's driver and model are resolved from, via withMainWorktreeOrgLimits,
// so the ledger shared by every worktree gets one set of limits and one set
// of role defaults. `ralph org start --plan` takes the same values but builds
// its runtime on another path (below); every other verb keeps reading only
// the caller's config.
// It takes a state dir that resolveOrgLedger already resolved and guarded
// (orgLedgerMutating), the same split as newOrgRuntimeAt: each caller also
// passes the directory on, to wireOrgWatchGate (spawn) or
// wireOrgStartWatchGate (start). It sets InboxHintStateDir
// (orgInboxHintStateDir), so the `ralph org stop` commands Spawn's errors
// print for a seat left with an unfinished spawn repeat --state-dir when it
// was given. `ralph org start --plan` builds its runtime with
// newOrgRuntimeFromConfig instead, from the config it reads once
// (runOrgStartPlan), and sets the same InboxHintStateDir.
func newOrgSpawnRuntimeAt(cmd *cobra.Command, resolvedStateDir, stateDirSource, configPath string) (*org.Org, error) {
	rt, err := newOrgRuntimeAt(resolvedStateDir, configPath)
	if err != nil {
		return nil, err
	}
	rt.Config, err = withMainWorktreeOrgLimits(rt.Config, configPath, resolvedStateDir, stateDirSource)
	if err != nil {
		return nil, err
	}
	// runOrgStartPlan sets the same value on the runtime it builds;
	// TestOrgStartPlan_InFlightLeaderRefusal_StopHintRepeatsStateDir fails
	// when that line is missing.
	rt.InboxHintStateDir = orgInboxHintStateDir(cmd, resolvedStateDir)
	return rt, nil
}

// withMainWorktreeOrgLimits returns cfg with MaxOrgs, MaxTotalSeats,
// Approval, RoleModels, DriverPool and LegacyModelKeys taken from <main
// worktree root>/ralph.toml (built-in defaults when that file does not
// exist, with the ralph.local.toml of that root applied on top, as
// config.Load reads them; applyMainWorktreeOrgLimits) when configPath is empty
// and the ledger is the main worktree's (org.LedgerMainWorktreeRoot):
// org.ResolveOrgStateDir placed it there (source "git-main-worktree"), or
// --state-dir or the personal file's [org].state_dir names the
// .harness/state/org of the main worktree of the repository spawn
// runs in, as a feature org's leader does with --state-dir.
// Every org_id in that ledger then gets the same org-wide limits, whichever
// subdirectory or linked worktree (with its own ./ralph.toml) spawn runs in
// (plan 2026-10-08-org-limits-reserve, Codex plan advisory 2). Approval
// ([org.approval]) is read only by `ralph org start --plan`, which records it
// in the org's feature binding, by the same rule, so a feature worktree's
// copy of ralph.toml, which its leader can change, does not choose the mode
// (plan 2026-10-12-org-plan-approval, A8); spawn carries it unused. The
// role defaults ([org.role_models] and [org].driver_pool) and the record of
// the keys ralph no longer reads ([org].model_pool and [org.roles]) come from
// the same file, so the copy of ralph.toml in a feature worktree does not
// choose the driver and model of a seat spawned without --driver and
// --model, nor whether spawn refuses an [org.roles] table (plan
// 2026-11-05-org-role-models, D6). With --config, a state dir from
// --state-dir or [org].state_dir naming another directory, or any other
// source, cfg is returned unchanged.
// No other setting is taken from the main worktree here; escalate, director
// attach and status, merge, and `ralph status` take Director.Enabled and
// Director.AutoMerge by the same rule (withMainWorktreeDirectorEnabled),
// director attach also Director.Wake and Director.WakeCmd
// (withMainWorktreeDirectorWakeConfig), `ralph org watch --all` also
// Director.Wake and Director.WakeCmd, through orgWatchDirectorWake
// (withMainWorktreeDirectorWakeConfigFrom), and director start the settings
// of the headless director, those two included
// (withMainWorktreeDirectorStartConfig). A main worktree ralph.toml that
// fails to load is an error, not a fallback. `ralph org start --plan` reads
// that file itself, once, and applies the same values with
// applyMainWorktreeOrgLimits, with its [reviewer] (resolveOrgStartConfig).
func withMainWorktreeOrgLimits(cfg config.OrgConfig, configPath, resolvedStateDir, stateDirSource string) (config.OrgConfig, error) {
	mainCfg, path, ok, err := loadMainWorktreeOrgConfig(configPath, resolvedStateDir, stateDirSource)
	if err != nil {
		return config.OrgConfig{}, fmt.Errorf("org: load max_orgs, max_total_seats, [org.approval].mode, [org.role_models] and driver_pool "+
			"from the main worktree's %s: %w", path, err)
	}
	if !ok {
		return cfg, nil
	}
	return applyMainWorktreeOrgLimits(cfg, mainCfg), nil
}

// applyMainWorktreeOrgLimits returns cfg with MaxOrgs, MaxTotalSeats,
// Approval, RoleModels, DriverPool and LegacyModelKeys taken from mainOrg,
// the [org] of the main worktree's ralph.toml: the values
// withMainWorktreeOrgLimits (spawn and start <task>) and
// resolveOrgStartConfig (start --plan) take from that file. RoleModels is
// replaced as a whole, so a role that only the caller's file sets is not
// kept, and DriverPool with it, since Spawn checks the driver
// org.ResolveRoleModel resolves against that pool (ValidateSpawnEnvelope;
// plan 2026-11-05-org-role-models, D6 and D9). LegacyModelKeys then names
// the main worktree's files, so the [org.roles] refusal and the model_pool
// note follow the file the role defaults come from.
func applyMainWorktreeOrgLimits(cfg, mainOrg config.OrgConfig) config.OrgConfig {
	cfg.MaxOrgs, cfg.MaxTotalSeats = mainOrg.MaxOrgs, mainOrg.MaxTotalSeats
	cfg.Approval = mainOrg.Approval
	cfg.RoleModels = mainOrg.RoleModels
	cfg.DriverPool = mainOrg.DriverPool
	cfg.LegacyModelKeys = mainOrg.LegacyModelKeys
	return cfg
}

// withMainWorktreeDirectorEnabled returns cfg with two values of
// [org.director], not only the Enabled its name says: Director.Enabled (the
// director switch, [org.director].enabled) and Director.AutoMerge
// ([org.director].auto_merge, the mark the /director skill reads for merging
// without asking), both taken from <main worktree root>/ralph.toml, by the
// rule withMainWorktreeOrgLimits follows: only when configPath is empty and
// the ledger is the main worktree's (org.LedgerMainWorktreeRoot), with the
// built-in defaults (both true) when that file does not exist. A
// feature org's leader escalates from its feature worktree, whose ralph.toml
// it can change, so that copy does not choose whether its items are left to
// the director (plan 2026-10-14-org-director-attach, D6), nor whether the
// director merges its PR without asking (plan 2026-10-17-org-auto-merge,
// AC2). With --config, or a ledger that is not the main worktree's, cfg is
// returned unchanged. No other setting is taken; escalate goes by Enabled
// alone, and director attach takes Director.Wake and Director.WakeCmd with
// withMainWorktreeDirectorWakeConfig. A main worktree ralph.toml that fails
// to load is an error, which `ralph org escalate` handles like a ralph.toml
// that does not load, `ralph org director attach` and `status` and `ralph
// org merge` return, and `ralph status` warns about before it shows the
// built-in Enabled with AutoMerge false (statusDirectorConfig). A verb that
// reports or acts on the switch or on auto_merge reads it through this
// function, so it sees the value escalate and the director go by.
func withMainWorktreeDirectorEnabled(cfg config.OrgConfig, configPath, resolvedStateDir, stateDirSource string) (config.OrgConfig, error) {
	mainCfg, path, ok, err := loadMainWorktreeOrgConfig(configPath, resolvedStateDir, stateDirSource)
	if err != nil {
		return config.OrgConfig{}, fmt.Errorf("org: load [org.director] from the main worktree's %s: %w", path, err)
	}
	if !ok {
		return cfg, nil
	}
	cfg.Director.Enabled = mainCfg.Director.Enabled
	cfg.Director.AutoMerge = mainCfg.Director.AutoMerge
	return cfg, nil
}

// withMainWorktreeDirectorStartConfig returns cfg with the settings `ralph
// org director start` launches the headless director by taken from <main
// worktree root>/ralph.toml, by the rule withMainWorktreeDirectorEnabled
// follows: Director.Enabled, Director.LaunchCmd ([org.director].launch_cmd),
// Director.Wake and Director.WakeCmd ([org.director].wake, which start
// records, and wake_cmd, which a launch_cmd start that pushes needs),
// Permissions ([org.permissions]: default, roles, and codex_verified),
// RoleModels ([org.role_models], replaced as a whole), DriverPool, and
// LegacyModelKeys (the files that still set [org].model_pool or [org.roles];
// plan 2026-11-05-org-role-models, D6). Only when configPath is empty and
// the ledger is the main worktree's (org.LedgerMainWorktreeRoot), with the
// built-in defaults when that file does not exist. The director acts for
// every org of the ledger, and a feature worktree's ralph.toml, which the
// leader running there can change, must not choose its permission mode or
// its driver and model (plan 2026-10-15-org-director-start, D6 and Scope),
// any more than whether a director is used at all, nor the command start
// runs to open its terminal (plan 2026-10-16-org-director-launch-cmd, D10),
// nor how it is woken (plan 2026-10-18-org-director-wake, D9). With
// --config, or a ledger that is not the main worktree's, cfg is returned
// unchanged. No other setting is taken. A main worktree ralph.toml that
// fails to load is an error, which director start returns.
func withMainWorktreeDirectorStartConfig(cfg config.OrgConfig, configPath, resolvedStateDir, stateDirSource string) (config.OrgConfig, error) {
	mainCfg, path, ok, err := loadMainWorktreeOrgConfig(configPath, resolvedStateDir, stateDirSource)
	if err != nil {
		return config.OrgConfig{}, fmt.Errorf("org: load [org.director].enabled, launch_cmd, wake and wake_cmd, [org.permissions], [org.role_models] and driver_pool "+
			"from the main worktree's %s: %w", path, err)
	}
	if !ok {
		return cfg, nil
	}
	cfg.Director.Enabled = mainCfg.Director.Enabled
	cfg.Director.LaunchCmd = mainCfg.Director.LaunchCmd
	cfg.Director.Wake = mainCfg.Director.Wake
	cfg.Director.WakeCmd = mainCfg.Director.WakeCmd
	cfg.Permissions = mainCfg.Permissions
	cfg.RoleModels = mainCfg.RoleModels
	cfg.DriverPool = mainCfg.DriverPool
	cfg.LegacyModelKeys = mainCfg.LegacyModelKeys
	return cfg, nil
}

// withMainWorktreeDirectorWakeConfig returns cfg with Director.Wake
// ([org.director].wake) and Director.WakeCmd ([org.director].wake_cmd) taken
// from <main worktree root>/ralph.toml, by the rule
// withMainWorktreeDirectorEnabled follows: only when configPath is empty and
// the ledger is the main worktree's (org.LedgerMainWorktreeRoot), with the
// built-in defaults (["pull"] and none) when that file does not exist (plan
// 2026-10-18-org-director-wake, D9 and AC4). `ralph org director attach`
// records the wake when --wake is not given and checks with the wake_cmd
// whether a wake with push can be sent; the watch runs that wake_cmd, so a
// feature worktree's ralph.toml, which the leader running there can change,
// must choose neither. With --config, or a ledger that is not the main
// worktree's, cfg is returned unchanged. No other setting is taken. A main
// worktree ralph.toml that fails to load is an error, which attach returns.
func withMainWorktreeDirectorWakeConfig(cfg config.OrgConfig, configPath, resolvedStateDir, stateDirSource string) (config.OrgConfig, error) {
	cfg, _, err := withMainWorktreeDirectorWakeConfigFrom(cfg, configPath, resolvedStateDir, stateDirSource)
	return cfg, err
}

// withMainWorktreeDirectorWakeConfigFrom is withMainWorktreeDirectorWakeConfig
// that also returns mainToml, the main worktree's ralph.toml the two values
// were taken from, "" when cfg is returned unchanged (--config, or a ledger
// that is not the main worktree's). `ralph org watch --all` runs wake_cmd in
// that file's directory (orgWatchDirectorWake; plan
// 2026-10-18-org-director-wake, D9), so it learns which file the rule read
// here instead of deriving the rule again.
func withMainWorktreeDirectorWakeConfigFrom(cfg config.OrgConfig, configPath, resolvedStateDir, stateDirSource string) (_ config.OrgConfig, mainToml string, _ error) {
	mainCfg, path, ok, err := loadMainWorktreeOrgConfig(configPath, resolvedStateDir, stateDirSource)
	if err != nil {
		return config.OrgConfig{}, "", fmt.Errorf("org: load [org.director].wake and wake_cmd from the main worktree's %s: %w", path, err)
	}
	if !ok {
		return cfg, "", nil
	}
	cfg.Director.Wake = mainCfg.Director.Wake
	cfg.Director.WakeCmd = mainCfg.Director.WakeCmd
	return cfg, path, nil
}

// loadMainWorktreeOrgConfig loads the [org] config of <main worktree
// root>/ralph.toml (loadMainWorktreeConfig) for withMainWorktreeOrgLimits,
// withMainWorktreeDirectorEnabled, withMainWorktreeDirectorStartConfig, and
// withMainWorktreeDirectorWakeConfigFrom.
func loadMainWorktreeOrgConfig(configPath, resolvedStateDir, stateDirSource string) (mainCfg config.OrgConfig, path string, ok bool, err error) {
	cfg, path, ok, err := loadMainWorktreeConfig(configPath, resolvedStateDir, stateDirSource)
	return cfg.Org, path, ok, err
}

// loadMainWorktreeConfig loads every table of <main worktree
// root>/ralph.toml (config.Load: built-in defaults when the file does not
// exist, with the ralph.local.toml of that root applied on top), for
// loadMainWorktreeOrgConfig and for resolveOrgStartConfig, which also takes
// [reviewer] from it. ok is false, with nothing loaded, when configPath is
// set or the ledger is not the main worktree's (org.LedgerMainWorktreeRoot).
// path is the file it loaded, also when err (the load error, unwrapped) is
// set.
func loadMainWorktreeConfig(configPath, resolvedStateDir, stateDirSource string) (mainCfg config.Config, path string, ok bool, err error) {
	if configPath != "" {
		return config.Config{}, "", false, nil
	}
	root, ok := org.LedgerMainWorktreeRoot(resolvedStateDir, stateDirSource)
	if !ok {
		return config.Config{}, "", false, nil
	}
	path = filepath.Join(root, "ralph.toml")
	cfg, err := loadConfigFile(path)
	if err != nil {
		return config.Config{}, path, true, err
	}
	return cfg, path, true, nil
}

// newOrgRuntimeAt is newOrgRuntime's shared implementation, taking an
// already-resolved state directory instead of resolving it itself. A caller
// that also needs the resolved directory for its own purposes beyond wiring
// (newOrgSendCmd's read hint, newOrgWatchCmd's banner +
// WatchAllParams.StatusDir) resolves and guards it once via
// resolveOrgLedger and calls this directly, instead of resolving twice
// (self-review LOW fix -- each resolution shells out to git).
func newOrgRuntimeAt(resolvedStateDir, configPath string) (*org.Org, error) {
	orgCfg, err := resolveOrgConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("org: load config: %w", err)
	}
	return newOrgRuntimeFromConfig(resolvedStateDir, orgCfg), nil
}

// newOrgRuntimeFromConfig is newOrgRuntimeAt with the [org] config already
// loaded: `ralph org start --plan` builds its runtime from the config it
// reads once for [org] and [reviewer] (resolveOrgStartConfig).
func newOrgRuntimeFromConfig(resolvedStateDir string, orgCfg config.OrgConfig) *org.Org {
	runner := driver.ExecRunner{}
	agmsgHome := driver.ResolveAgmsgHome(orgCfg.AgmsgHome)
	rt := &org.Org{
		Config:   orgCfg,
		Manifest: org.NewManifestStoreAtPath(org.ManifestPathIn(resolvedStateDir)),
		Receipts: org.NewReceiptStoreAtPath(org.ReceiptsPathIn(resolvedStateDir)),
		// The seat panes spawn creates carry driver.SeatPaneEnv:
		// AGMSG_SELF_NAME=off, so an agmsg a seat runs itself cannot rename
		// its herdr agent (finding F3), and RALPH_WIP_CHECKPOINT=off, so the
		// seat's Claude Code session makes no wip: commit (finding F11).
		// Those are the Herdr's fixed PaneEnv; after them spawn appends
		// RALPH_PRIMARY_CLI=<the seat's driver> as the extraEnv of the call
		// that makes the seat's pane, WorkspaceCreate, PaneSplit, or
		// TabCreate (driver.PrimaryCLIEntry).
		Herdr: driver.NewHerdr(runner, driver.SeatPaneEnv()...),
		Agmsg: driver.Agmsg{R: runner, Home: agmsgHome},
		// The leader's role prompt names the same home's scripts in its
		// inbox commands ({{AGMSG_SCRIPTS}}).
		AgmsgHome: agmsgHome,
		// Inbox, EscalationsPath, and DesktopNotify are set below by
		// wireOrgInbox, which newOrgEscalateFallbackRuntime shares.
		InboxPollInterval: orgInboxPollIntervalOverride,
		// The three Codex* fields below are only ever non-zero in tests
		// (TestMain pins orgCodexSessionsDirOverride to a directory that
		// does not exist, plus tiny observe timeout/interval, the same
		// seam shape as doctorShellAliasEnv/doctorCodexSandboxEnv --
		// main_test.go). Production always leaves all three at their zero
		// value here, so org.Org resolves CodexSessionsDir from the
		// environment at call time and uses its own built-in observe
		// timeout/interval, exactly as before this plan.
		CodexSessionsDir:          orgCodexSessionsDirOverride,
		CodexModelObserveTimeout:  orgCodexModelObserveTimeoutOverride,
		CodexModelObserveInterval: orgCodexModelObserveIntervalOverride,
	}
	wireOrgInbox(rt, resolvedStateDir)
	return rt
}

// wireOrgInbox sets the inbox fields of rt for the ledger at
// resolvedStateDir: Inbox, EscalationsPath, and DesktopNotify. The inbox and
// the human path of `ralph org escalate` and `ralph org inbox notify` live
// in the same ledger as the manifest. DesktopNotify is nil in production
// (osascript on darwin); a test that escalates through the CLI sets
// orgDesktopNotifyOverride so it shows no real notification. Both
// newOrgRuntimeFromConfig (which newOrgRuntimeAt calls) and
// newOrgEscalateFallbackRuntime call it, so a change to this wiring reaches
// both.
func wireOrgInbox(rt *org.Org, resolvedStateDir string) {
	rt.Inbox = org.NewInboxStore(resolvedStateDir)
	rt.EscalationsPath = org.EscalationsPathIn(resolvedStateDir)
	rt.DesktopNotify = orgDesktopNotifyOverride
}

// orgCodexSessionsDirOverride, orgCodexModelObserveTimeoutOverride, and
// orgCodexModelObserveIntervalOverride are copied into every org.Org this
// package constructs (newOrgRuntimeFromConfig), overriding org.Org's own
// environment-at-call-time default resolution for CodexSessionsDir and its
// built-in observe timeout/interval defaults. All three are the zero value
// in production. TestMain pins orgCodexSessionsDirOverride to a directory
// that does not exist and both durations to near-zero, for the same reason
// it pins doctorShellAliasEnv/doctorCodexSandboxEnv (main_test.go): without
// this seam, every runOrg*-based test would silently start reading the
// developer's real ~/.codex/sessions and waiting out the real 8s poll
// timeout. A test that specifically exercises codex model observation
// overrides orgCodexSessionsDirOverride (and, if it needs the poll to
// actually retry, the two duration overrides) to its own fixture directory
// for the duration of that one test, restoring it via t.Cleanup -- the
// same save/restore/cleanup pattern doctor_shell_alias_test.go and
// doctor_codex_writable_root_test.go already use for their own seams.
var (
	orgCodexSessionsDirOverride          string
	orgCodexModelObserveTimeoutOverride  time.Duration
	orgCodexModelObserveIntervalOverride time.Duration
)

// orgDesktopNotifyOverride is copied into every org.Org's DesktopNotify
// (wireOrgInbox). nil in production, which means osascript on darwin.
var orgDesktopNotifyOverride org.EscalateFunc

// orgInboxPollIntervalOverride is copied into every org.Org's
// InboxPollInterval (newOrgRuntimeFromConfig). Zero in production, which means
// `ralph org wait --inbox` reads the inbox again every second; a test that
// waits for an item escalated later sets a tiny value.
var orgInboxPollIntervalOverride time.Duration

// wireOrgWatchGate sets rt.WatchGate, the watch gate of plan
// 2026-10-12-org-watch-gate: spawn, both forms of start, director start, and
// send call it right after they build their runtime, and no other verb does,
// so stop, disband, escalate, the inbox verbs, wait, read, status, report,
// and watch work with no watch running. The gate (org.NewWatchGate) probes
// the ledger at resolvedStateDir and judges a liveness mark that carries no
// deadman_minutes by rt.Config's. Its refusal repeats --state-dir in the
// commands it names when --state-dir was given, by orgInboxHintStateDir's
// rule.
func wireOrgWatchGate(cmd *cobra.Command, rt *org.Org, resolvedStateDir string) {
	if orgWatchGateOverride != nil {
		rt.WatchGate = orgWatchGateOverride
		return
	}
	settle := orgWatchGateSettleOverride
	if settle == 0 {
		settle = org.DefaultWatchGateSettle
	}
	rt.WatchGate = org.NewWatchGate(resolvedStateDir, orgInboxHintStateDir(cmd, resolvedStateDir), nil, rt.Config.DeadmanMinutes, settle)
}

// wireOrgStartWatchGate is wireOrgWatchGate for both forms of `ralph org
// start`, which starts the cross-org watch when none runs (plan
// 2026-10-12-org-watch-gate, FR-6): its gate first runs ensureOrgWatch,
// which launches the watch --detach would (orgWatchChild) and prints one
// line on stderr when it did, and then the gate wireOrgWatchGate sets. The
// org layer calls the gate after the start's own input checks and before it
// writes anything (StartFeature before the worktree, then Spawn again, which
// finds the watch running and launches nothing), so a start refused for its
// input launches no watch. A watch that holds watch.lock and accepts no work
// is refused as the gate refuses it (*org.WatchGateError). The launched
// watch keeps running after start returns. With orgWatchGateOverride set
// (TestMain), only that gate runs.
//
// `ralph org director start` wires the same gate (plan
// 2026-10-15-org-director-start, D5), which StartDirector calls after its
// own input checks and its first look at a current registration, and
// before it writes anything, so a start refused for its input or while a
// director is attached launches no watch. verb is the verb after
// "ralph org" ("start", "director start") and starts what it starts
// ("leader", "director"): they name the verb in the stderr line of a
// launched watch and in a failed launch's error, which says that no leader
// (no director) was started.
func wireOrgStartWatchGate(cmd *cobra.Command, rt *org.Org, resolvedStateDir, stateDirSource, configPath, verb, starts string) {
	wireOrgWatchGate(cmd, rt, resolvedStateDir)
	if orgWatchGateOverride != nil {
		return
	}
	check := rt.WatchGate
	hintStateDir := orgInboxHintStateDir(cmd, resolvedStateDir)
	ensure := orgWatchEnsure{
		stateDir: resolvedStateDir, deadman: rt.Config.DeadmanMinutes,
		child: func() (orgWatchLaunchRequest, error) {
			return orgWatchChild(cmd, resolvedStateDir, stateDirSource, configPath, rt.Config.AgmsgHome, 0)
		},
		errPrefix:    "org: " + verb + ": the cross-org watch did not start, so no " + starts + " was started",
		hintStateDir: hintStateDir,
		refuse: func(probe org.WatchLiveness) error {
			return &org.WatchGateError{StateDir: resolvedStateDir, HintStateDir: hintStateDir, Liveness: probe}
		},
	}
	rt.WatchGate = func() error {
		got, err := ensureOrgWatch(ensure)
		if err != nil {
			return err
		}
		if got.launched {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "ralph org %s: started the cross-org watch (pid %d); log: %s\n",
				verb, got.pid, filepath.Join(resolvedStateDir, org.WatchLogFileName))
		}
		return check()
	}
}

// orgWatchGateOverride, when not nil, is the Org.WatchGate that
// wireOrgWatchGate sets in place of the one org.NewWatchGate builds, and
// makes wireOrgStartWatchGate set only that gate. nil in production.
// TestMain sets one that always passes, so the tests of spawn, start, and
// send need no watch and start launches none; a test of the gate itself sets
// it back to nil.
var orgWatchGateOverride func() error

// orgWatchGateSettleOverride is the settle of the gate wireOrgWatchGate
// builds: how long it waits for a stale watch to accept work. Zero in
// production, which means org.DefaultWatchGateSettle (2 s); TestMain
// shortens it so a test of a stale refusal does not wait 2 s.
var orgWatchGateSettleOverride time.Duration

// wireOrgEscalateWatch sets rt.EscalateWatch for `ralph org escalate` to
// orgEscalateWatchProbe's probe of the ledger at resolvedStateDir, judging a
// liveness mark that carries no deadman_minutes by rt.Config's. Unlike the
// watch gate it does not wait out a stale watch: escalate then sends the
// item to the human at once.
func wireOrgEscalateWatch(rt *org.Org, resolvedStateDir string) {
	rt.EscalateWatch = orgEscalateWatchProbe(resolvedStateDir, rt.Config.DeadmanMinutes)
}

// orgEscalateWatchProbe returns a probe (org.ProbeWatch at the time of the
// call) of the cross-org watch on the ledger at resolvedStateDir, judging a
// liveness mark that carries no deadman_minutes by deadmanMinutes: the probe
// `ralph org escalate` decides by (wireOrgEscalateWatch) and `ralph org
// director attach` warns by, so the warning says what escalate would do.
// orgEscalateWatchOverride, when set, is returned instead.
func orgEscalateWatchProbe(resolvedStateDir string, deadmanMinutes int) func() org.WatchLiveness {
	if orgEscalateWatchOverride != nil {
		return orgEscalateWatchOverride
	}
	return func() org.WatchLiveness {
		return org.ProbeWatch(resolvedStateDir, time.Now(), deadmanMinutes)
	}
}

// orgEscalateWatchOverride, when not nil, is the probe orgEscalateWatchProbe
// returns, for escalate's Org.EscalateWatch and director attach's warning.
// nil in production. TestMain leaves it nil, so a test's escalate probes its
// own ledger, where no watch runs, and sends the item at once
// (inbox_no_watch) as before the director switch, and a test's attach warns;
// a test of the recorded-only path, or of an attach with no warning, sets
// it, or holds watch.lock with a running mark.
var orgEscalateWatchOverride func() org.WatchLiveness

// orgReadCommandHint builds the `ralph org read` recovery command printed
// in send's post-failure notes and its unconfirmed-submit warning (AR-2,
// docs/reports/cross-review-triage-org-send-enter-timing.md). It appends
// --state-dir resolvedStateDir only when stateDirFlagSet is true (--state-dir
// was explicitly passed to this invocation of `send`): a bare `ralph org
// read --org-id ... --seat ...` would otherwise resolve the DEFAULT state
// dir (config/git-main-worktree/git-toplevel/cwd, see
// org.ResolveOrgStateDir), which either fails with "seat not found" or --
// worse -- silently reads a different seat that happens to share the same
// org_id/seat_id in that default manifest. A state dir from the personal
// file or a default tier is deliberately NOT appended: the same shell, run
// from the same directory, resolves it the same way when the operator runs
// the printed command themselves, so repeating it would be redundant as
// long as neither changes between the two commands (the personal file's
// [org].state_dir can be edited in between; it and the git-main-worktree,
// git-toplevel, and cwd fallbacks depend on cwd, since
// org.ResolveOrgStateDir runs git in the current directory).
// resolvedStateDir must be the value org.ResolveOrgStateDir already
// returned (always absolute), not the raw flag text, so the hint survives a
// cwd change before the operator acts on it.
//
// --config is deliberately not included: newOrgReadCmd's RunE loads
// *configPath into org.Org.Config, but (*org.Org).Read never reads that
// field (it only calls findSeat, which reads the manifest, and
// Herdr.PaneRead) -- so --config cannot affect read's seat lookup.
func orgReadCommandHint(orgID, seat, resolvedStateDir string, stateDirFlagSet bool) string {
	hint := fmt.Sprintf("ralph org read --org-id %s --seat %s", orgID, seat)
	if stateDirFlagSet {
		hint += " --state-dir " + org.ShellWord(resolvedStateDir)
	}
	return hint
}

// resolveOrgConfig loads the [org] envelope from configPath, falling back
// to ./ralph.toml when configPath is empty, and to config.Default().Org
// when the file it reads does not exist -- matching the plan's documented
// --config default -- with ralph.local.toml (config.LocalPath) applied on
// top either way. config.Load decides what "does not exist" means: only
// fs.ErrNotExist. A ./ralph.toml that exists but cannot be read (no
// permission, or a symlink into a directory it may not enter) is an error,
// not the defaults: the defaults would turn a file that says
// codex_verified = false or auto_merge = false into true (cross-review
// cycle 1 of codex-verified-auto-merge-on).
func resolveOrgConfig(configPath string) (config.OrgConfig, error) {
	cfg, err := resolveConfig(configPath)
	if err != nil {
		return config.OrgConfig{}, err
	}
	return cfg.Org, nil
}

// resolveConfig loads every table of the config that resolveOrgConfig takes
// its [org] from, for a verb that also needs one outside [org]: `ralph org
// start --plan` takes its [org] and its [reviewer] from it, the latter when
// the main worktree's ralph.toml does not apply (resolveOrgStartConfig).
// An error is prefixed with the path it was asked to read, as given (the
// --config value, or the relative ralph.toml), so every `org: load config: `
// sentence that wraps it starts with that file (plan
// 2026-11-02-config-path-and-ledger-docs, D1). The prefix is not always the
// file at fault, since some errors of config.Load name a file themselves. An
// error of the personal file ralph.local.toml names it after the prefix
// (`ralph.toml: /repo/ralph.local.toml: toml: ...`): the file that failed is
// the second one named. An error of opening the file, or of a key only
// ralph.local.toml may set, names the same path twice
// (`ralph.toml: open ralph.toml: permission denied`), as `ralph config get`
// prints it. TestResolveConfig_PrefixesTheFileItWasAskedToRead pins the
// shapes of the first two.
func resolveConfig(configPath string) (config.Config, error) {
	path := configPath
	if path == "" {
		path = "ralph.toml"
	}
	cfg, err := loadConfigFile(path)
	if err != nil {
		return cfg, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// loadConfigFile is config.Load by default. resolveConfig and
// loadMainWorktreeConfig read a config file through it, and tests reassign
// it to count the reads of each file.
var loadConfigFile = config.Load

// resolveOrgStartConfig returns the [org] config `ralph org start --plan`
// builds its runtime from and the [reviewer] table it writes in the feature
// org's leader's task, reading each file once: the --config file or
// ./ralph.toml (resolveConfig), then, when configPath is empty and the ledger
// is the main worktree's, <main worktree root>/ralph.toml
// (loadMainWorktreeConfig), ralph.local.toml on top either way. The second
// file, when read, gives MaxOrgs, MaxTotalSeats, Approval, RoleModels,
// DriverPool and LegacyModelKeys, as for spawn (applyMainWorktreeOrgLimits),
// and the whole [reviewer]. A subdirectory or a linked worktree with a
// ralph.toml of its own then chooses neither the org's approval mode, nor
// the leader's driver and model when --driver and --model are omitted (plan
// 2026-11-05-org-role-models, D6), nor the models and efforts of the
// leader's plan advisory (plan 2026-10-30-standard-flow-config, D5), and
// all come from one version of the file (plan
// 2026-11-02-standard-config-gaps, D1). When the working directory is the
// main worktree's root, the two files are the same ralph.toml and it is
// read twice, once as the cwd's config and once as the main worktree's
// override (plan 2026-11-02-standard-config-gaps, AC2).
func resolveOrgStartConfig(configPath, resolvedStateDir, stateDirSource string) (config.OrgConfig, config.ReviewerConfig, error) {
	cfg, err := resolveConfig(configPath)
	if err != nil {
		return config.OrgConfig{}, config.ReviewerConfig{}, fmt.Errorf("org: load config: %w", err)
	}
	mainCfg, path, ok, err := loadMainWorktreeConfig(configPath, resolvedStateDir, stateDirSource)
	if err != nil {
		return config.OrgConfig{}, config.ReviewerConfig{}, fmt.Errorf(
			"org: load max_orgs, max_total_seats, [org.approval].mode, [org.role_models], driver_pool and [reviewer] from the main worktree's %s: %w", path, err)
	}
	if !ok {
		return cfg.Org, cfg.Reviewer, nil
	}
	return applyMainWorktreeOrgLimits(cfg.Org, mainCfg.Org), mainCfg.Reviewer, nil
}

// printLegacyModelPoolNotice writes org.LegacyModelPoolNotice(cfg), the one
// line that says cfg still sets [org].model_pool, which ralph no longer
// reads, to stderr when there is one (plan 2026-11-05-org-role-models,
// spec FR-6). spawn, both forms of start, and director start call it once,
// with the config they resolve the seat's or the director's driver and
// model from (after the main worktree's values are applied), and go on.
// stderr is cmd.ErrOrStderr() at call sites so tests can capture the line.
func printLegacyModelPoolNotice(cfg config.OrgConfig, stderr io.Writer) {
	if notice := org.LegacyModelPoolNotice(cfg); notice != "" {
		_, _ = fmt.Fprintln(stderr, notice)
	}
}

// splitCommaList splits a comma-separated flag value into a trimmed,
// non-empty slice. An all-blank input yields nil.
func splitCommaList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// deprecatedLeaderDriverFlag is the flag's old spelling, kept as a hidden
// alias of --leader-driver (cobra MarkDeprecated prints the notice on use).
const deprecatedLeaderDriverFlag = "lead-driver"

// resolveLeaderDriver picks the effective leader driver from --leader-driver
// and its deprecated alias. Which flag was given is decided by
// Flags().Changed, not by comparing against a default: an explicit
// --leader-driver claude (the default value) next to --lead-driver codex is
// still a conflict. Both spellings with the same value is accepted.
func resolveLeaderDriver(cmd *cobra.Command, leaderDriver, deprecatedDriver string) (string, error) {
	if !cmd.Flags().Changed(deprecatedLeaderDriverFlag) {
		return leaderDriver, nil
	}
	if !cmd.Flags().Changed("leader-driver") {
		return deprecatedDriver, nil
	}
	if deprecatedDriver != leaderDriver {
		return "", fmt.Errorf("org: --%s %q conflicts with --leader-driver %q; --%s is a deprecated alias of --leader-driver, pass only --leader-driver",
			deprecatedLeaderDriverFlag, deprecatedDriver, leaderDriver, deprecatedLeaderDriverFlag)
	}
	return leaderDriver, nil
}

// orgWideLimitsHelp is the paragraph `ralph org spawn --help` and `ralph org
// start --help` print about the limits a new seat is checked against.
const orgWideLimitsHelp = "Before a seat starts, `ralph org spawn` and `ralph org start` check\n" +
	"[org].max_seats for the org and two limits every org_id in the ledger\n" +
	"shares: [org].max_orgs (running orgs) and [org].max_total_seats (active\n" +
	"seats across all orgs). Those two are read from the main worktree's\n" +
	"ralph.toml (built-in defaults when it has none) when --config is not\n" +
	"given and the ledger is the main worktree's .harness/state/org, whether\n" +
	"found by default or named with --state-dir (as a feature org's leader\n" +
	"does) or with [org].state_dir in ralph.local.toml, so every subdirectory\n" +
	"and linked worktree gets the same limits. In every other case (--config,\n" +
	"a --state-dir or [org].state_dir naming another ledger, a ledger found\n" +
	"from the git toplevel, no git repository) the --config file, else\n" +
	"./ralph.toml, is used. A refusal says how to free a slot with\n" +
	"`ralph org disband`."

// orgRoleModelsHelp is the paragraph `ralph org spawn --help` and `ralph org
// start --help` print about where a seat's driver and model come from when
// --driver and --model are omitted (org.ResolveRoleModel; plan
// 2026-11-05-org-role-models, spec FR-3, FR-4 and FR-6). It comes before
// orgWideLimitsHelp, which its last sentence points at.
const orgRoleModelsHelp = "Without --driver and --model, the seat runs with the driver and model\n" +
	"of its role's [org.role_models] entry; a role that ralph.toml does not\n" +
	"set gets the built-in entry, from a claude set when [org].driver_pool\n" +
	"has claude and from a codex set otherwise. --model alone keeps the\n" +
	"entry's driver. --driver alone, for a driver other than the entry's,\n" +
	"also needs --model, unless the entry's model is \"\": a model of \"\"\n" +
	"launches the CLI without --model, on its own default model. A role\n" +
	"with no entry needs both flags. A ralph.toml that still sets\n" +
	"[org.roles] is refused; one that still sets [org].model_pool, which\n" +
	"ralph no longer reads, gets one line on stderr. Without --config,\n" +
	"[org.role_models] and [org].driver_pool are read from the same\n" +
	"ralph.toml as max_orgs and max_total_seats below."

// orgDriverFlagUsage is the --driver usage of the verbs that resolve a
// driver with org.ResolveRoleModel: what the flag picks, and entry, the
// [org.role_models] entry it defaults to. No backticks, as for
// orgReserveFlagUsage.
func orgDriverFlagUsage(what, entry string) string {
	return what + ": claude|codex (default: the driver of " + entry + ")"
}

// orgModelFlagUsage is the --model usage of the verbs that resolve a model
// with org.ResolveRoleModel, for entry, the [org.role_models] entry it
// defaults to. No backticks, as for orgReserveFlagUsage.
func orgModelFlagUsage(entry string) string {
	return "model name or alias (default: the model of " + entry + "; \"\" there launches the CLI without --model, on its own default model)"
}

// orgReserveFlagUsage is the --reserve usage `ralph org spawn` and `ralph org
// start` share. No backticks: pflag would take the first backticked word as
// the flag's value name.
const orgReserveFlagUsage = "reserve a path for this org, relative to the repo root; repeat the flag for more paths. " +
	"A path ending in / reserves that directory prefix, any other path one file, and . the whole repo. " +
	"Refused when it overlaps the reservation of another running org; released by ralph org disband. " +
	"Satisfies the autonomous-mode --scope requirement"

// orgSpawnTimeoutFlagUsage is the --timeout-ms usage `ralph org spawn` and
// `ralph org start` share, given the herdr and agmsg steps of the seat's
// spawn: org.Spawn runs them all under one deadline of --timeout-ms, and
// each compensating call after a failed step under a 10s bound of its own
// (internal/org's defaultDriverCallTimeout; plan
// 2026-10-22-seat-spawn-close, D2 and D4). No backticks, as for
// orgReserveFlagUsage.
func orgSpawnTimeoutFlagUsage(steps string) string {
	return "deadline in milliseconds for the whole spawn, shared by every herdr and agmsg step (" + steps + "); " +
		"the cleanup after a failed step is bounded separately, 10s per call"
}

func newOrgSpawnCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var (
		seatID, role, driverName, model, cwd, prompt, scope, leaderDriver string
		deprecatedDriver                                                  string
		reserve                                                           []string
		timeoutMS                                                         int
		dryRun, allowUnscoped                                             bool
	)

	cmd := &cobra.Command{
		Use:   "spawn",
		Short: "Spawn a new org seat",
		Long: "ralph org spawn starts one seat of --org-id in a pane of the org's one\n" +
			"herdr tab and joins it to the org's agmsg team, recording each step in\n" +
			"the manifest. The org's first seat gets the root pane of a new herdr\n" +
			"workspace, and the workspace and its tab are labelled with the org_id.\n" +
			"A later seat splits a live seat's pane: with one seat live, its pane to\n" +
			"the right; with more, the pane of the last seat spawned other than the\n" +
			"leader, downward. So seats spawned in the order leader, implementer,\n" +
			"reviewer sit left, right top, and right bottom. Each pane is labelled\n" +
			"with its seat id. An org an earlier version started keeps its tab per\n" +
			"seat: a new seat splits a live seat's pane in one of them, and gets a\n" +
			"tab of its own only when no seat's pane is left in the workspace.\n" +
			"\n" +
			orgRoleModelsHelp + "\n" +
			"\n" +
			orgWideLimitsHelp + "\n" +
			"\n" +
			"--reserve claims paths of the repo for the org. Only the leader seat\n" +
			"(--id leader) takes it, and an org keeps one reservation until it is\n" +
			"disbanded: the same paths again pass, different ones are refused.\n" +
			"\n" +
			"A spawn, --dry-run included, is refused while no long-running\n" +
			"`ralph org watch --all` (not stopping, not a --once run) runs on the\n" +
			"ledger; the refusal says how to start one or stop a stale one.\n" +
			"\n" +
			"A spawn that is not a --dry-run then makes sure the herdr server runs.\n" +
			orgHerdrServerHelp + "\n" +
			"\n" +
			"In an org started with `ralph org start --plan`, a spawn of any seat\n" +
			"but the leader is refused while the org's leader seat is not running.\n" +
			"When the org's approval mode (the [org.approval].mode recorded when\n" +
			"the org was started) is per-plan, a spawn of a seat whose role is\n" +
			"implementer is also refused until an approval recorded with\n" +
			"`ralph org approve` since the org's last disband matches the plan\n" +
			"file. Either refusal, --dry-run included, is recorded as `rejected`.\n" +
			"\n" +
			"A spawn (not a --dry-run) into an org is refused, with nothing\n" +
			"recorded, while `ralph org regate` works on that org's branch, or\n" +
			"while `ralph org merge` or `ralph org cleanup` removes that org's\n" +
			"merged feature worktree; run the spawn again once it ends. A ci regate\n" +
			"that waits for the PR's checks has released regate.lock and does not\n" +
			"refuse it meanwhile (see ralph org regate --help).\n" +
			"\n" +
			orgCodexHookTrustNoticeHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			if err := requireSeatIdentifier("--id", seatID); err != nil {
				return err
			}
			for flag, val := range map[string]string{"--role": role, "--cwd": cwd} {
				if strings.TrimSpace(val) == "" {
					return fmt.Errorf("org: %s is required", flag)
				}
			}
			// A retired --role / --id is refused here, before
			// org.ResolveRoleModel, so the operator sees the successor
			// guidance instead of an [org.role_models] error for a role
			// that no longer exists. Spawn runs the same check again. The
			// ralph.toml retired-key check ([org.roles] included) stays in
			// Spawn only: it must run after Spawn's idempotent return.
			// The refusal is printed the same way as a Spawn rejection.
			// --role director is refused here too, for the same reason: the
			// director is not a seat (plan 2026-10-15-org-director-start, D7).
			inputErr := org.RetiredRoleInputErr(role, seatID, prompt)
			if inputErr == nil {
				inputErr = org.DirectorRoleInputErr(role)
			}
			if inputErr != nil {
				printSpawnResult(cmd, org.SpawnResult{Outcome: org.SpawnOutcomeRejected, Err: inputErr})
				return inputErr
			}
			effectiveLeaderDriver, err := resolveLeaderDriver(cmd, leaderDriver, deprecatedDriver)
			if err != nil {
				return err
			}

			resolvedStateDir, stateDirSource, err := resolveOrgLedger(cmd, *stateDir, orgLedgerMutating)
			if err != nil {
				return err
			}
			rt, err := newOrgSpawnRuntimeAt(cmd, resolvedStateDir, stateDirSource, *configPath)
			if err != nil {
				return err
			}
			wireOrgWatchGate(cmd, rt, resolvedStateDir)
			wireOrgHerdrServerGate(cmd, rt, resolvedStateDir, "spawn")
			printLegacyModelPoolNotice(rt.Config, cmd.ErrOrStderr())
			resolved, err := org.ResolveRoleModel(rt.Config, role, driverName, model)
			if err != nil {
				return err
			}
			printCodexHookTrustNotice(cmd.ErrOrStderr(), resolved.Driver, "org spawn", seatID, cwd)
			result := rt.Spawn(org.SpawnParams{
				OrgID: *orgID, SeatID: seatID, Role: role,
				Driver: resolved.Driver, Model: resolved.Model, DriverFromRoleEntry: resolved.DriverFromRoleEntry,
				Cwd: cwd, Prompt: prompt, Scope: scope, Reserve: reserve, TimeoutMS: timeoutMS, DryRun: dryRun,
				LeaderDriver: effectiveLeaderDriver, AllowUnscoped: allowUnscoped,
			})
			printSpawnResult(cmd, result)
			return result.Err
		},
	}

	cmd.Flags().StringVar(&seatID, "id", "", "seat id (required)")
	cmd.Flags().StringVar(&role, "role", "", "seat role (required)")
	cmd.Flags().StringVar(&driverName, "driver", "", orgDriverFlagUsage("driver CLI the seat runs as", "the role's [org.role_models] entry"))
	cmd.Flags().StringVar(&model, "model", "", orgModelFlagUsage("the role's [org.role_models] entry"))
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory for the new seat (required); a relative path is made absolute from the directory this command runs in")
	cmd.Flags().StringVar(&prompt, "prompt", "", "optional initial prompt passed to the agent")
	cmd.Flags().StringVar(&scope, "scope", "", "optional scope description (recorded on the spawned event; substituted into --role templates as {{SCOPE}})")
	cmd.Flags().StringArrayVar(&reserve, "reserve", nil, orgReserveFlagUsage+"; only the leader seat (--id leader) takes it")
	cmd.Flags().StringVar(&leaderDriver, "leader-driver", "claude", "driver (claude|codex) the org's coordinating leader identity runs as, for the agmsg type registered on ensureLeaderJoined")
	cmd.Flags().StringVar(&deprecatedDriver, deprecatedLeaderDriverFlag, "", "deprecated alias of --leader-driver")
	if err := cmd.Flags().MarkDeprecated(deprecatedLeaderDriverFlag, "use --leader-driver"); err != nil {
		panic(err) // the flag was registered on the line above
	}
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 60000, orgSpawnTimeoutFlagUsage("workspace create or pane split, labels, agent start, agmsg joins, HELLO"))
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate and record without starting a real seat")
	cmd.Flags().BoolVar(&allowUnscoped, "allow-unscoped", false, "explicitly bypass the autonomous-mode --scope requirement (recorded on the spawned event)")

	return cmd
}

// printSpawnResult prints the outcome of a spawn on stdout, for spawn and
// both forms of start. A seat's model of "" (launched without --model, on
// the driver CLI's own default) is shown as org.CLIDefaultModelText.
func printSpawnResult(cmd *cobra.Command, r org.SpawnResult) {
	out := cmd.OutOrStdout()
	switch r.Outcome {
	case org.SpawnOutcomeRejected:
		_, _ = fmt.Fprintf(out, "rejected: %v\n", r.Err)
	case org.SpawnOutcomeIdempotent:
		_, _ = fmt.Fprintf(out, "seat %q already spawned (org_id=%s driver=%s model=%s pane_id=%s)\n",
			r.Seat.SeatID, r.Seat.OrgID, r.Seat.Driver, org.ModelOrCLIDefault(r.Seat.Model), r.Seat.PaneID)
	case org.SpawnOutcomeSpawned:
		_, _ = fmt.Fprintf(out, "spawned seat %q (org_id=%s driver=%s model=%s pane_id=%s dry_run=%t)\n",
			r.Seat.SeatID, r.Seat.OrgID, r.Seat.Driver, org.ModelOrCLIDefault(r.Seat.Model), r.Seat.PaneID, r.Seat.DryRun)
	case org.SpawnOutcomeFailed:
		_, _ = fmt.Fprintf(out, "spawn failed: %v\n", r.Err)
	}
	printCodexModelMismatchWarning(cmd, r.Seat.SeatID, r.ModelReceipt)
}

// printCodexModelMismatchWarning writes AC-6's stderr warning when receipt
// is a genuine codex model mismatch: Honored == "false" AND a non-empty
// ReportedEffectiveModel. That second condition is what keeps an envelope
// rejection (also honored=false, but with no reported model at all --
// reject(), spawn.go) from ever being mistaken for a model mismatch here.
// Exit code is never touched by this function -- the caller's own Err
// still decides that, unchanged.
func printCodexModelMismatchWarning(cmd *cobra.Command, seatID string, r org.Receipt) {
	if r.Honored != org.HonoredFalse || r.ReportedEffectiveModel == "" {
		return
	}
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
		"warning: seat %q was started with --model %s, but codex reports it is running %s. codex switches retired models on its own, and a codex config override has the same effect. Check 'ralph doctor' (codex model slugs) and the seat's pane.\n",
		seatID, r.CommandedModel, r.ReportedEffectiveModel)
}

// newOrgStartCmd wires `ralph org start` -- headless-leader spawn sugar over
// (*org.Org).Spawn, per the plan's design decision ("`org start` = lead 座席
// の spawn 糖衣", docs/plans/active/2026-08-02-org-runtime-lead.md). It
// always spawns SeatID == Role == org.LeaderIdentity ("leader"): the org's
// coordinating agmsg identity and the leader seat are, by design, the same
// seat -- see the leaderSelfSpawn branch in internal/org/spawn.go's Spawn.
// Every other concern (envelope validation, the permission-mode gate,
// manifest/receipt bookkeeping) flows through the same saga every other
// `ralph org spawn` call uses; this command does not special-case the leader
// runtime object in any way beyond picking its SeatID/Role and required
// positional task argument.
//
// With --plan and --feature it runs runOrgStartPlan instead, which takes the
// leader's cwd, scope, reservation and task from a split plan feature
// ((*org.Org).StartFeature, plan
// docs/plans/active/2026-10-09-org-feature-worktree.md). Without them the
// command behaves as before.
func newOrgStartCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var (
		driverName, model, cwd, scope string
		plan, feature                 string
		reserve                       []string
		timeoutMS                     int
		allowUnscoped                 bool
	)

	cmd := &cobra.Command{
		Use:   "start (<task> | --plan <split plan> --feature <slug>)",
		Short: "Spawn a headless leader seat (with --plan, for one split plan feature in its own worktree)",
		Long: "ralph org start spawns the org's headless leader seat, in one of two\n" +
			"forms. Both are a thin wrapper over the same Spawn saga every other\n" +
			"`ralph org spawn` call uses: each spawns the org's coordinating\n" +
			"\"leader\" identity itself (seat id \"leader\", role \"leader\"), expands\n" +
			"internal/org/prompts/leader.md with the task substituted for\n" +
			"{{TASK}} and a one-line [org] envelope summary substituted for\n" +
			"{{ENVELOPE}}. Envelope validation, the permission-mode gate, and\n" +
			"manifest/receipt bookkeeping all flow through Spawn exactly as they\n" +
			"would for any other seat. See .claude/skills/org/SKILL.md for the\n" +
			"leader's full operating manual.\n" +
			"\n" +
			"`ralph org start --plan <split plan> --feature <slug>` starts the org\n" +
			"for one feature of a split plan. The split plan is a file <id>.md\n" +
			"directly in <state dir>/splits/ (see --state-dir), approved by writing\n" +
			"the digest `scripts/plan-visual.sh digest <file>` prints on its\n" +
			"`- Approved: <date> sha256:<digest>` line; a plan that changed after\n" +
			"its approval is refused. The org_id is the feature's slug unless\n" +
			"--org-id is given; it is at most 20 characters, so that the leader can\n" +
			"spawn its \"implementer\" seat within herdr's 32-character agent name\n" +
			"<org_id>_<seat_id>. From the main worktree, which must be a clean\n" +
			"checkout of the default branch, scripts/ralph-worktree.sh makes the\n" +
			"worktree .claude/worktrees/org-<org_id> on the branch <type>/<slug>,\n" +
			"and the leader runs there. Its reservation (the feature's `- Reserve:`\n" +
			"paths) and the org's binding to the feature (shown by\n" +
			"`ralph org status`) come from the plan, and its task from the\n" +
			"feature's body and the plan's preamble (its ## sections before\n" +
			"## Features), so --cwd, --scope, --reserve, --allow-unscoped and a\n" +
			"task argument are refused. Running the same start again reuses the\n" +
			"worktree; a start into a running org that was not started for that\n" +
			"feature is refused. An org whose leader was stopped but that was not\n" +
			"disbanded can be started again with the same --plan and --feature\n" +
			"after the split plan was changed and approved again, as long as the\n" +
			"plan's preamble and the feature's own section did not change and the\n" +
			"approval mode is the one the org was started with: the binding\n" +
			"records a digest of the preamble and that section (the feature\n" +
			"digest), so a change to the sections of other features, the features\n" +
			"this one depends on included, does not refuse the start. Nothing is\n" +
			"added to the ledger then, so the binding keeps the whole plan's\n" +
			"digest of the first start. An org started by an older ralph has no\n" +
			"feature digest in its record, so any change to the split plan\n" +
			"refuses its start, as before. `ralph org disband` leaves the worktree\n" +
			"and the branch, which `ralph org regate` works in until the PR is\n" +
			"merged. Once the PR is merged, `ralph org merge` removes the worktree, the\n" +
			"branch, and the record of scripts/ralph-worktree.sh right after its\n" +
			"merge, and `ralph org cleanup` does the same for a PR merged on\n" +
			"GitHub (see their --help for what they check first). For a PR closed\n" +
			"without a merge, run `./scripts/ralph-worktree.sh cleanup --id\n" +
			"org-<org_id> --force-branch` in the main worktree once the branch is\n" +
			"not needed. The org's approval mode, [org.approval].mode (per-plan by\n" +
			"default, or split), is read from the same ralph.toml as max_orgs and\n" +
			"max_total_seats below, recorded with the org's binding to the feature\n" +
			"and named in the leader's task, so a change to the setting applies to\n" +
			"orgs started after it; starting a running feature org again with\n" +
			"another mode is refused. The four keys of [reviewer], the model and\n" +
			"effort of the reviewers the leader's plan advisory calls, are read\n" +
			"from the same ralph.toml as max_orgs and max_total_seats below and\n" +
			"written in the leader's task as two lines, one for codex and one for\n" +
			"claude: the leader uses them as they were when the org started and\n" +
			"does not read them itself.\n" +
			"\n" +
			"`ralph org start <task>` spawns the leader in --cwd with <task> as its\n" +
			"task, as before --plan existed. It is not tied to a split plan: it\n" +
			"makes no worktree, and --scope and --reserve give the scope and the\n" +
			"reservation. --reserve claims paths of the repo for the org until it\n" +
			"is disbanded: starting the org again with the same paths passes, with\n" +
			"different ones is refused.\n" +
			"\n" +
			"Both forms make sure a long-running `ralph org watch --all` watches\n" +
			"the ledger before the leader is spawned: when none is running, start\n" +
			"launches one as `ralph org watch --all --detach` does (in the ledger's\n" +
			"main worktree root when it has one, logging to <state dir>/watch.log)\n" +
			"and waits up to 10s for it, and it is refused when a watch holds the\n" +
			"lock but is stale, stopping, or a --once run. When it would launch\n" +
			"one, a --config whose [org].agmsg_home is a relative path is refused\n" +
			"before the launch, wherever start runs, and no leader is started: the\n" +
			"watch may resolve it from that root, not from here.\n" +
			"\n" +
			"Both forms then make sure the herdr server runs, the --plan form before\n" +
			"it makes the worktree.\n" +
			orgHerdrServerHelp + "\n" +
			"\n" +
			"Both forms are refused, with nothing recorded, while\n" +
			"`ralph org regate` works on the org's branch, or while `ralph org\n" +
			"merge` or `ralph org cleanup` removes the org's merged feature\n" +
			"worktree; run start again once it ends. A ci regate that waits for the\n" +
			"PR's checks has released regate.lock and does not refuse them\n" +
			"meanwhile (see ralph org regate --help).\n" +
			"\n" +
			orgRoleModelsHelp + " For start, the role is leader.\n" +
			"\n" +
			orgWideLimitsHelp + "\n" +
			"\n" +
			orgScaffoldVersionNoticeHelp + "\n" +
			"\n" +
			orgCodexHookTrustNoticeHelp,
		Args: func(cmd *cobra.Command, args []string) error {
			if orgStartUsesPlan(cmd) {
				return nil // checkOrgStartPlanInput refuses a task argument with its own message
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if orgStartUsesPlan(cmd) {
				if err := checkOrgStartPlanInput(cmd, args, *orgID, plan, feature); err != nil {
					return err
				}
				return runOrgStartPlan(cmd, orgStartPlanParams{
					orgID: *orgID, stateDir: *stateDir, configPath: *configPath, plan: plan, feature: feature,
					driverName: driverName, model: model, timeoutMS: timeoutMS,
				})
			}
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			task := strings.TrimSpace(args[0])
			if task == "" {
				return fmt.Errorf("org: start: task must not be blank")
			}
			if strings.TrimSpace(cwd) == "" {
				return fmt.Errorf("org: --cwd is required")
			}

			resolvedStateDir, stateDirSource, err := resolveOrgLedger(cmd, *stateDir, orgLedgerMutating)
			if err != nil {
				return err
			}
			rt, err := newOrgSpawnRuntimeAt(cmd, resolvedStateDir, stateDirSource, *configPath)
			if err != nil {
				return err
			}
			wireOrgStartWatchGate(cmd, rt, resolvedStateDir, stateDirSource, *configPath, "start", "leader")
			wireOrgHerdrServerGate(cmd, rt, resolvedStateDir, "start")
			printLegacyModelPoolNotice(rt.Config, cmd.ErrOrStderr())
			printScaffoldVersionNotice(cmd.ErrOrStderr(), "org start", orgScaffoldRoot(resolvedStateDir, stateDirSource))
			resolved, err := org.ResolveRoleModel(rt.Config, org.LeaderIdentity, driverName, model)
			if err != nil {
				return err
			}
			printCodexHookTrustNotice(cmd.ErrOrStderr(), resolved.Driver, "org start", org.LeaderIdentity, cwd)

			result := rt.Spawn(org.SpawnParams{
				OrgID: *orgID, SeatID: org.LeaderIdentity, Role: org.LeaderIdentity,
				Driver: resolved.Driver, Model: resolved.Model, DriverFromRoleEntry: resolved.DriverFromRoleEntry, Cwd: cwd, Task: task,
				Scope: scope, Reserve: reserve, TimeoutMS: timeoutMS, AllowUnscoped: allowUnscoped,
			})
			printSpawnResult(cmd, result)
			if result.Err == nil {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(),
					"hint: ralph org status --org-id %s ; attach with herdr to observe the leader pane\n", *orgID)
			}
			return result.Err
		},
	}

	cmd.Flags().StringVar(&driverName, "driver", "", orgDriverFlagUsage("driver CLI the leader seat runs as", "[org.role_models].leader"))
	cmd.Flags().StringVar(&model, "model", "", orgModelFlagUsage("[org.role_models].leader"))
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory for the leader seat (required without --plan); a relative path is made absolute from the directory this command runs in")
	cmd.Flags().StringVar(&scope, "scope", "", "optional scope description, as for ralph org spawn --scope")
	cmd.Flags().StringArrayVar(&reserve, "reserve", nil, orgReserveFlagUsage)
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 60000, orgSpawnTimeoutFlagUsage("workspace create or pane split, labels, agent start, agmsg join"))
	cmd.Flags().BoolVar(&allowUnscoped, "allow-unscoped", false, "explicitly bypass the autonomous-mode --scope requirement")
	cmd.Flags().StringVar(&plan, "plan", "", "approved split plan, a file <id>.md in <state dir>/splits/, to start one feature of; "+
		"needs --feature and replaces the task argument, --cwd, --scope and --reserve; --allow-unscoped is refused with it")
	cmd.Flags().StringVar(&feature, "feature", "", "slug of the split plan feature (its ### <slug> heading) the org is started for; needs --plan")

	return cmd
}

// orgStartPlanConflicts are the `ralph org start` flags the --plan form
// refuses, besides a task argument (checkOrgStartPlanInput).
var orgStartPlanConflicts = []string{"cwd", "scope", "reserve", "allow-unscoped"}

// orgStartUsesPlan reports whether `ralph org start` was given --plan or
// --feature, which selects the split plan form (runOrgStartPlan).
func orgStartUsesPlan(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("plan") || cmd.Flags().Changed("feature")
}

// checkOrgStartPlanInput refuses what the --plan form cannot take, before a
// state dir is resolved: --plan without --feature or the reverse, a blank
// value for either, a task argument, the flags in orgStartPlanConflicts, and
// a malformed --org-id. --org-id may be left out (StartFeature then uses the
// feature's slug).
func checkOrgStartPlanInput(cmd *cobra.Command, args []string, orgID, plan, feature string) error {
	if cmd.Flags().Changed("plan") != cmd.Flags().Changed("feature") {
		return errors.New("org: start: --plan and --feature go together: pass both to start the org for one feature of a split plan, " +
			"or neither for ralph org start <task>")
	}
	if strings.TrimSpace(plan) == "" || strings.TrimSpace(feature) == "" {
		return errors.New("org: start: --plan and --feature must not be blank")
	}
	const supplies = "with --plan the split plan supplies the leader's cwd (the feature worktree), its scope and its reservation"
	if len(args) > 0 {
		return fmt.Errorf("org: start: --plan takes no task argument (got %q): %s, and the feature's body and the plan's preamble are its task", args[0], supplies)
	}
	for _, flag := range orgStartPlanConflicts {
		if cmd.Flags().Changed(flag) {
			return fmt.Errorf("org: start: --plan cannot be combined with --%s: %s", flag, supplies)
		}
	}
	if orgID != "" {
		return org.ValidateIdentifier("org_id", orgID)
	}
	return nil
}

// orgStartPlanParams is what runOrgStartPlan takes from the flags of `ralph
// org start`.
type orgStartPlanParams struct {
	orgID, stateDir, configPath string
	plan, feature               string
	driverName, model           string
	timeoutMS                   int
}

// runOrgStartPlan is `ralph org start --plan <split plan> --feature <slug>`:
// it resolves the state dir once with the legacy-ledger guard
// (resolveOrgLedger), reads the config once (resolveOrgStartConfig: the
// main worktree's org-wide limits, approval mode, and role defaults, as for
// every start, and the [reviewer] table from the same read, which the
// runtime does not keep since it holds only [org]), prints the model_pool
// note of that config (printLegacyModelPoolNotice) and the note that the
// main worktree's scaffold is older than this CLI
// (printScaffoldVersionNotice with orgScaffoldRoot; plan
// 2026-11-12-scaffold-version-notice), builds the runtime on
// that dir from it (newOrgRuntimeFromConfig, with newOrgSpawnRuntimeAt's
// InboxHintStateDir) with start's watch gate (wireOrgStartWatchGate, which
// launches the watch when none runs; StartFeature runs it before it makes
// the worktree) and the herdr server gate (wireOrgHerdrServerGate, which
// starts `herdr server` when none runs; StartFeature runs it right after the
// watch gate), resolves the leader's driver and model
// (org.ResolveRoleModel) before anything is made, prints the note of the
// hooks a codex leader would run without (printCodexHookTrustNotice, for the
// main worktree root; plan 2026-11-12-codex-hook-trust-check), and runs
// (*org.Org).StartFeature with that dir, the main worktree root
// (org.FeatureRepoRoot), and the [reviewer] table. The result is
// printed as a spawn's, followed on success by the worktree, the branch and
// the status hint; a refusal or failure is returned as StartFeature's error,
// which already says whether the worktree stays and how to remove it.
func runOrgStartPlan(cmd *cobra.Command, p orgStartPlanParams) error {
	resolvedStateDir, stateDirSource, err := resolveOrgLedger(cmd, p.stateDir, orgLedgerMutating)
	if err != nil {
		return err
	}
	orgCfg, reviewer, err := resolveOrgStartConfig(p.configPath, resolvedStateDir, stateDirSource)
	if err != nil {
		return err
	}
	printLegacyModelPoolNotice(orgCfg, cmd.ErrOrStderr())
	printScaffoldVersionNotice(cmd.ErrOrStderr(), "org start", orgScaffoldRoot(resolvedStateDir, stateDirSource))
	rt := newOrgRuntimeFromConfig(resolvedStateDir, orgCfg)
	// The same value newOrgSpawnRuntimeAt sets;
	// TestOrgStartPlan_InFlightLeaderRefusal_StopHintRepeatsStateDir fails
	// when this line is missing.
	rt.InboxHintStateDir = orgInboxHintStateDir(cmd, resolvedStateDir)
	wireOrgStartWatchGate(cmd, rt, resolvedStateDir, stateDirSource, p.configPath, "start", "leader")
	wireOrgHerdrServerGate(cmd, rt, resolvedStateDir, "start")
	repoRoot, err := org.FeatureRepoRoot(resolvedStateDir, stateDirSource)
	if err != nil {
		return err
	}
	resolved, err := org.ResolveRoleModel(rt.Config, org.LeaderIdentity, p.driverName, p.model)
	if err != nil {
		return err
	}
	// The feature worktree does not exist yet, and a codex leader in it
	// reads the main checkout's hooks (codexHookCheckoutOf maps a linked
	// worktree to its main checkout), so repoRoot is the directory to check.
	printCodexHookTrustNotice(cmd.ErrOrStderr(), resolved.Driver, "org start", org.LeaderIdentity, repoRoot)

	res := rt.StartFeature(org.StartFeatureParams{
		StateDir: resolvedStateDir, RepoRoot: repoRoot, SplitPath: p.plan, Feature: p.feature, OrgID: p.orgID,
		Driver: resolved.Driver, Model: resolved.Model, DriverFromRoleEntry: resolved.DriverFromRoleEntry, TimeoutMS: p.timeoutMS, Reviewer: reviewer,
	})
	printSpawnResult(cmd, res.Spawn)
	if res.Spawn.Err != nil {
		return res.Spawn.Err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "worktree: %s\nbranch: %s\n"+
		"hint: ralph org status --org-id %s ; attach with herdr to observe the leader pane\n", res.Worktree, res.Branch, res.OrgID)
	return nil
}

func newOrgSendCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var (
		to, text         string
		timeoutMS        int
		dryRun           bool
		raw              bool
		enterDelayMS     int
		noScreenFallback bool
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a message to a seat",
		Long: "ralph org send validates --text against the typed message protocol\n" +
			"(internal/org/protocol, see .claude/rules/ralph/agent-messaging.md) before\n" +
			"sending: TYPE must be a known value, TASK_ID is required for\n" +
			"TASK/RESULT/REVIEW/BLOCKED/CONTRACT, and the body must not exceed the\n" +
			"size cap. Pass --raw to bypass this validation for free-form text. A\n" +
			"text that holds a bracketed paste marker (ESC [200~ or ESC [201~) is\n" +
			"refused with or without --raw.\n" +
			"After typing the text, send waits briefly and presses Enter once, then\n" +
			"tries to confirm the seat left idle/done. It never presses Enter a\n" +
			"second time: if the submit cannot be confirmed, it prints a warning\n" +
			"instead of guessing -- see --enter-delay-ms below.\n" +
			"\n" +
			"Before typing, send waits up to --timeout-ms for the seat to be idle or\n" +
			"done. A seat in the middle of a turn (waiting in the foreground on a\n" +
			"loop, `ralph org wait`, or a gate) does not get the message until the\n" +
			"turn ends. When the wait runs out, nothing is typed and the error says\n" +
			"the seat's turn did not end and how to send again. The submit check only\n" +
			"sees the seat's turn go working or blocked, so a turn the seat started on\n" +
			"its own (a Claude Code leader woken by its Monitor, say) also reads as\n" +
			"confirmed; `ralph org read` shows whether the message arrived.\n" +
			"\n" +
			"herdr can keep reporting a seat working after its turn ended. So, for a\n" +
			"claude or codex seat, the wait on herdr leaves 10 seconds of --timeout-ms\n" +
			"(at most a third of it) for a look at the seat's screen. When herdr has\n" +
			"not reported idle or done by then, send reads the pane twice, 3 seconds\n" +
			"apart. Only when both reads show the same screen waiting for input (an\n" +
			"empty input box under the line that ends a turn, or a fresh session) does\n" +
			"send type: it takes the pane's lock, reads the screen once more, checks\n" +
			"that the text is in the input box before it presses Enter, and confirms\n" +
			"the submit on the screen instead of with herdr. The sent event then\n" +
			"starts with `via=screen herdr_state=<state>`. Any other screen (a turn\n" +
			"at work, a dialog, one it cannot judge) gets nothing typed, and send\n" +
			"waits on herdr for the rest of --timeout-ms; when that runs out too, the\n" +
			"error also gives herdr's state, what the screen showed, and the commands\n" +
			"to look into it. A --timeout-ms whose third is less than 6 seconds plus\n" +
			"the pause before Enter gets no look at the screen. Neither does a send\n" +
			"whose pause before Enter is over 4 seconds. --no-screen-fallback\n" +
			"turns the look off: send waits on herdr alone.\n" +
			"\n" +
			"Send also takes the lock of the seat's pane, the one `ralph org resume`\n" +
			"takes, and holds it until Enter is pressed and the submit is checked. It\n" +
			"waits for the lock no longer than what is left of --timeout-ms. When\n" +
			"another send or resume still holds it then, nothing is typed, nothing is\n" +
			"recorded, and send exits 1.\n" +
			"\n" +
			"A TASK (TYPE: TASK, also on the first line of --raw text), --dry-run\n" +
			"included, is refused while no long-running `ralph org watch --all`\n" +
			"(not stopping, not a --once run) runs on the ledger.\n" +
			"\n" +
			"In an org started with `ralph org start --plan` whose approval mode is\n" +
			"per-plan, a TASK to a seat whose role is implementer, --raw and\n" +
			"--dry-run included, is refused with nothing recorded unless its header\n" +
			"has exactly one `PLAN: <path in the worktree> sha256:<digest>` line\n" +
			"and that plan matches an approval recorded with `ralph org approve`.\n" +
			"The refusal says what the check found, then\n" +
			"`refused by the approval check` and the next step.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if enterDelayMS < 0 {
				return fmt.Errorf("org: --enter-delay-ms must be >= 0")
			}
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			if err := requireSeatIdentifier("--to", to); err != nil {
				return err
			}
			// Resolved once (mirrors newOrgWatchCmd's self-review LOW fix)
			// so the recovery-command hint below (AR-2,
			// docs/reports/cross-review-triage-org-send-enter-timing.md)
			// can use the SAME resolved, absolute state dir newOrgRuntimeAt
			// wires the runtime to, instead of re-deriving it (or, worse,
			// printing the raw --state-dir flag text, which breaks if the
			// operator's cwd differs when they run the printed command).
			stateDirFlagSet := cmd.Flags().Changed("state-dir")
			resolvedStateDir, _, err := resolveOrgLedger(cmd, *stateDir, orgLedgerMutating)
			if err != nil {
				return err
			}
			rt, err := newOrgRuntimeAt(resolvedStateDir, *configPath)
			if err != nil {
				return err
			}
			wireOrgWatchGate(cmd, rt, resolvedStateDir)
			result := rt.Send(org.SendParams{
				OrgID: *orgID, To: to, Text: text, TimeoutMS: timeoutMS, DryRun: dryRun, Raw: raw,
				EnterDelayMS: enterDelayMS, NoScreenFallback: noScreenFallback,
				HintStateDir: orgInboxHintStateDir(cmd, resolvedStateDir),
			})
			readHint := orgReadCommandHint(*orgID, to, resolvedStateDir, stateDirFlagSet)
			if result.Err != nil {
				// Four distinct outcomes after a driver call failed or its
				// result could not be recorded, and each needs its own
				// operator instruction -- see SendResult.Progress's and
				// SendProgress's doc comments in internal/org/verbs.go for
				// the full state-by-state contract this switch mirrors.
				// SendProgressNothingSent (no pane call was ever attempted)
				// prints no note.
				switch result.Progress {
				case org.SendProgressTextUnacknowledged:
					// Outcome unknown, not "failed" -- see SendProgress's
					// doc comment for why.
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
						"note: the send-text call to seat %q (pane %s) failed, so the message may or "+
							"may not have been typed into the input box. Check the pane before sending "+
							"again: a second send would be typed after whatever is there. Check the "+
							"seat with '%s'.\n",
						to, result.PaneID, readHint)
				case org.SendProgressTextTyped:
					// The one state that IS certain: PaneSendText really did
					// return success, and Enter was never pressed -- ctx
					// expired before Enter, or the screen path's check before
					// Enter did not find the text in the input box. In the
					// second case the pane may show something other than the
					// input box, so the note says not to press Enter then.
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
						"note: the message text was typed into pane %s of seat %q but not submitted. "+
							"Check it with '%s' and clear or submit it there before sending again: a "+
							"second send would be typed after it. If the pane shows no input box "+
							"holding the text (a dialog, say), do not press Enter.\n",
						result.PaneID, to, readHint)
				case org.SendProgressEnterUnacknowledged:
					// Outcome unknown -- the note is conditional on purpose:
					// never tell the operator to press Enter unconditionally.
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
						"note: the text was typed and Enter was sent to seat %q (pane %s), but herdr "+
							"did not acknowledge it, so the message may or may not have been "+
							"submitted. Read the pane first with '%s'. Only if the text is still "+
							"sitting in the input box, submit or clear it there. If the seat shows "+
							"anything else (it is working, or it shows a dialog), do not press Enter "+
							"and do not send the message again.\n",
						to, result.PaneID, readHint)
				case org.SendProgressEnterPressed:
					// The message was very likely delivered -- must NOT
					// suggest retyping, clearing, or pressing Enter.
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
						"note: Enter was already pressed for seat %q (pane %s); only the sent "+
							"history event could not be recorded. Do not send the message again. "+
							"Check the seat with '%s'.\n",
						to, result.PaneID, readHint)
				}
				// Every error of Send already starts with "org: send: ".
				return result.Err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "sent message to seat %q\n", to)
			// A submit that Send could not confirm is not a failure (a very
			// short agent turn can go working -> done before the confirm wait
			// even starts) -- so this stays a stderr warning with exit 0, not
			// an error. Send never resends Enter itself (see confirmSubmitted's
			// doc comment in internal/org/verbs.go for why a blind second
			// keystroke is unsafe), so the operator is the one who decides
			// whether to submit it by hand.
			if !dryRun && !result.SubmitConfirmed {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
					"warning: could not confirm that seat %q started working after Enter. "+
						"Check its pane with '%s'. "+
						"If the message is still sitting in the input box, submit it with "+
						"'herdr pane send-keys %s Enter'. If the pane shows anything else "+
						"(the seat is working, or it shows a dialog), do not press Enter. "+
						"ralph does not resend Enter on its own: a blind keystroke could "+
						"confirm an approval dialog.\n",
					to, readHint, result.PaneID)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "target seat id (required)")
	cmd.Flags().StringVar(&text, "text", "", "message text")
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 30000,
		"overall herdr timeout in milliseconds for one send (idle wait and screen check + pre-Enter wait + submit confirmation)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "record without sending a real message")
	cmd.Flags().BoolVar(&raw, "raw", false, "bypass typed message protocol validation")
	cmd.Flags().IntVar(&enterDelayMS, "enter-delay-ms", 0,
		fmt.Sprintf("wait this many milliseconds between typing the message and pressing Enter (0 = built-in default of %d)", org.DefaultSendEnterDelayMS))
	cmd.Flags().BoolVar(&noScreenFallback, "no-screen-fallback", false,
		"wait on herdr's idle/done state alone for all of --timeout-ms, and never read the seat's screen to type into a seat herdr still reports working")

	return cmd
}

func newOrgWaitCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var (
		seat      string
		until     string
		timeoutMS int
		inbox     bool
		director  string
		types     string
	)

	cmd := &cobra.Command{
		Use:   "wait",
		Short: "Wait for a seat to reach one of the given states, or for an open inbox item (--inbox)",
		Long: "ralph org wait defaults --until to \"idle,done\": live-probed herdr\n" +
			"(v0.7.5) reports an interactive agent resting at its input prompt as\n" +
			"\"done\" (turn finished), not \"idle\" -- waiting on \"idle\" alone times\n" +
			"out against a perfectly receptive seat (same finding that fixed `ralph\n" +
			"org send`'s own wait, internal/org/verbs.go's Send). --timeout-ms\n" +
			"defaults to a bounded 60000ms so a headless leader following this\n" +
			"command's own default cannot block forever; pass --timeout-ms 0 to\n" +
			"explicitly opt into an unbounded wait.\n" +
			"\n" +
			"--inbox waits on the org inbox instead of a seat, without --org-id or\n" +
			"--seat: it returns at once when an item is open, else reads the inbox\n" +
			"again every second until one is, and prints each open item (ID, org,\n" +
			"TYPE, first line of the body). Acked items do not count. --timeout-ms\n" +
			"bounds it the same way (exit 1 when no open item arrives in time, 0\n" +
			"waits unbounded). --inbox cannot be combined with --org-id, --seat, or\n" +
			"--until.\n" +
			"\n" +
			"--types <list> (only with --inbox) limits the wait to the open items of\n" +
			"the listed TYPEs: a comma-separated list of QUESTION, BLOCKED, RESULT,\n" +
			"and ALERT, upper case, with no spaces (QUESTION,BLOCKED,RESULT waits\n" +
			"for the items a leader raised and leaves out the watch's ALERT items).\n" +
			"The wait returns only for an open item of a listed TYPE and prints only\n" +
			"those; an open item of another TYPE neither ends it nor is printed.\n" +
			"Without --types every TYPE counts, as before. An empty, lower-case, or\n" +
			"unknown entry is refused with exit 1 before anything waits or writes a\n" +
			"mark. With --director, the exit 4 check (no inbox item is open) and the\n" +
			"item count of the mark count only the listed TYPEs too. An open ALERT\n" +
			"item left out of the wait is still sent to the human when its ack\n" +
			"deadline passes, by ralph org watch --all.\n" +
			"\n" +
			"--director <id> (only with --inbox) is the wait of the director\n" +
			"registered as <id>, of the form d<N> (such as d1). It waits only while\n" +
			"<id> is the current registration and not stopped: otherwise it exits 5\n" +
			"without waiting. While it waits, it reads the registration again\n" +
			"every 30 s and exits 5 at the first such read after <id> is replaced,\n" +
			"detached, or stopped. It writes its own mark, director-wait.json in\n" +
			"the state dir, when it starts, every 30 s, when the watch it sees\n" +
			"changes, and when it ends, with how it ended; `ralph org director\n" +
			"status` and `ralph status` show it. Every 5 s it probes the cross-org\n" +
			"watch (ralph org watch --all), and when a watch that accepted work\n" +
			"(at its last probe, or at the previous --director wait of <id>) no\n" +
			"longer does and no inbox item is open, it exits 4, naming what\n" +
			"watch.lock and watch-alive.json show and the command that starts a\n" +
			"watch, ralph org watch --all --detach. It starts no watch itself.\n" +
			"When no watch accepts work at its start, it prints a note: line on\n" +
			"stderr and keeps waiting. SIGINT, SIGTERM, and SIGHUP end it: it\n" +
			"writes end=signal to its mark and then ends by the same signal. A\n" +
			"SIGHUP or SIGINT that was ignored when it started (nohup ignores\n" +
			"SIGHUP) stays ignored.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// --director is checked first, so it is never taken for a
			// seat wait's flag, and nothing is read before the refusal.
			if cmd.Flags().Changed("director") && !inbox {
				return errors.New("org: wait: --director needs --inbox")
			}
			if cmd.Flags().Changed("types") && !inbox {
				return errors.New("org: wait: --types needs --inbox")
			}
			if inbox {
				return runOrgWaitInbox(cmd, *stateDir, *configPath, timeoutMS, director, types)
			}
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			if err := requireSeatIdentifier("--seat", seat); err != nil {
				return err
			}
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath, orgLedgerReadOnly)
			if err != nil {
				return err
			}
			result := rt.Wait(org.WaitParams{OrgID: *orgID, Seat: seat, Until: splitCommaList(until), TimeoutMS: timeoutMS})
			if result.Output != "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), result.Output)
			}
			if result.Err != nil {
				return fmt.Errorf("org: wait: %w", result.Err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&seat, "seat", "", "seat id to wait on (required without --inbox)")
	cmd.Flags().StringVar(&until, "until", "idle,done", "comma-separated states to wait for (idle,done,blocked; not with --inbox)")
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 60000, "wait timeout in milliseconds, bounded by default (pass 0 to explicitly wait unbounded)")
	cmd.Flags().BoolVar(&inbox, "inbox", false, "wait for an open item in the org inbox instead of a seat (no --org-id, --seat, or --until)")
	cmd.Flags().StringVar(&director, "director", "",
		"wait as the director registration `id` (d<N>, such as d1; only with --inbox): exit 5 unless it is the current registration, exit 4 when the watch stops accepting work, and keep the mark director-wait.json")
	cmd.Flags().StringVar(&types, "types", "",
		"wait only for open items of these inbox TYPEs, a comma-separated `list` of QUESTION, BLOCKED, RESULT, ALERT (only with --inbox; default every TYPE)")

	return cmd
}

// runOrgWaitInbox is `ralph org wait --inbox`: (*org.Org).WaitInbox, then
// one line per open item on stdout (printInboxWaitItems). The inbox is
// shared by every org in the ledger, so --org-id is refused rather than
// ignored, as are --seat and --until, which name a seat wait. A --types
// value that org.ParseInboxWaitTypes refuses is refused here too, before the
// state dir is resolved and before a --director wait checks its
// registration, so nothing waits and no mark is written. With --director it
// is runOrgWaitInboxDirector instead, after the same refusals.
func runOrgWaitInbox(cmd *cobra.Command, stateDir, configPath string, timeoutMS int, director, typesFlag string) error {
	for _, flag := range []string{"org-id", "seat", "until"} {
		if cmd.Flags().Changed(flag) {
			return fmt.Errorf("org: wait: --inbox cannot be combined with --%s", flag)
		}
	}
	var types []string
	if cmd.Flags().Changed("types") {
		parsed, err := org.ParseInboxWaitTypes(typesFlag)
		if err != nil {
			return err
		}
		types = parsed
	}
	if cmd.Flags().Changed("director") {
		return runOrgWaitInboxDirector(cmd, stateDir, configPath, timeoutMS, director, types)
	}
	rt, err := newOrgRuntime(cmd, stateDir, configPath, orgLedgerReadOnly)
	if err != nil {
		return err
	}
	items, err := rt.WaitInbox(time.Duration(timeoutMS)*time.Millisecond, types...)
	printInboxWaitItems(cmd.OutOrStdout(), items)
	return err
}

// printInboxWaitItems prints the open items a wait --inbox returned, one
// line each: ID, org, TYPE, and the first line of the body, tab-separated.
func printInboxWaitItems(w io.Writer, items []org.InboxItem) {
	for _, item := range items {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			inboxField(item.ID), inboxField(item.OrgID), inboxField(item.Type), inboxSummary(item.Body))
	}
}

func newOrgReadCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var (
		seat  string
		lines int
	)

	cmd := &cobra.Command{
		Use:   "read",
		Short: "Read recent pane output from a seat",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			if err := requireSeatIdentifier("--seat", seat); err != nil {
				return err
			}
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath, orgLedgerReadOnly)
			if err != nil {
				return err
			}
			result := rt.Read(org.ReadParams{OrgID: *orgID, Seat: seat, Lines: lines})
			if result.Output != "" {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), result.Output)
			}
			if result.Err != nil {
				return fmt.Errorf("org: read: %w", result.Err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&seat, "seat", "", "seat id to read from (required)")
	cmd.Flags().IntVar(&lines, "lines", 50, "number of recent pane lines to read")

	return cmd
}

func newOrgStopCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var (
		seat               string
		dryRun, all, force bool
	)

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a seat, or every active seat with --all, and close its herdr pane",
		Long: "ralph org stop sends C-c to the seat's herdr pane, closes the pane (which\n" +
			"ends the seat's process and its screen output; read it first with\n" +
			"`ralph org read` if you need it), leaves agmsg, and records `stopped`.\n" +
			"Before sending C-c or closing, it checks that herdr labels the pane with\n" +
			"the seat id (or, for a seat an earlier version started, which has no pane\n" +
			"label, its tab, unless that tab label is also the org_id), inside a\n" +
			"workspace labelled with the org_id (the labels spawn gave them). A pane\n" +
			"that fails the check (a label differs, or\n" +
			"herdr cannot be asked) is not sent C-c and is not closed, because its\n" +
			"recorded id may now name another pane.\n" +
			"When the pane cannot be closed, the seat stays active (stop_failed) and the\n" +
			"command exits 1; run it again once herdr answers.\n" +
			"\n" +
			"--all stops every active seat of every org_id, without --org-id or --seat.\n" +
			"It keeps going past a seat it cannot stop, lists each one on stderr, and\n" +
			"exits 1 if any is left. --force records `stopped` even when the pane could\n" +
			"not be closed, printing the failure as a warning and exiting 0; a pane that\n" +
			"failed the check stays open, only the record is written. When the\n" +
			"command runs inside a pane it stops (HERDR_PANE_ID), that pane is closed\n" +
			"last, after all output, which ends the command. If that last close fails,\n" +
			"the seat is recorded active again and the command exits 1, so running it\n" +
			"again (or stop --all from another pane) retries the close; with --force\n" +
			"the seat stays recorded stopped and the failure is only a warning.\n" +
			"A headless director (`ralph org director start`) is not an org seat, so\n" +
			"--all leaves it open: when the ledger has one, a note: line on stderr names\n" +
			"`ralph org director stop --id`, which stops it, and the exit code does not\n" +
			"change.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				if err := rejectFlagsWithAll(cmd, "stop", "org-id", "seat"); err != nil {
					return err
				}
				rt, err := newOrgRuntime(cmd, *stateDir, *configPath, orgLedgerMutating)
				if err != nil {
					return err
				}
				result := rt.StopAll(org.StopAllParams{DryRun: dryRun, Force: force})
				runErr := printStopAllResult(cmd, result)
				printStopAllDirectorNote(cmd.ErrOrStderr(), result, orgInboxHintStateDir(cmd, filepath.Dir(rt.Manifest.Path())))
				return closeDeferredSelf(cmd, rt, "", result.DeferredSelfPaneID, force, runErr)
			}
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			if err := requireSeatIdentifier("--seat", seat); err != nil {
				return err
			}
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath, orgLedgerMutating)
			if err != nil {
				return err
			}
			result := rt.Stop(org.StopParams{OrgID: *orgID, Seat: seat, DryRun: dryRun, Force: force})
			printCodexModelMismatchWarning(cmd, seat, result.ModelReceipt)
			if result.Err != nil {
				return withPrefixOnce("org: stop: ", result.Err)
			}
			if result.CloseErr != nil {
				printSeatFailure(cmd.ErrOrStderr(), fmt.Sprintf("%q", seat), org.SeatFailure{SeatID: seat, Err: result.CloseErr, Forced: true})
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "stopped seat %q\n", seat)
			}
			return closeDeferredSelf(cmd, rt, "", result.DeferredSelfPaneID, force, nil)
		},
	}

	cmd.Flags().StringVar(&seat, "seat", "", "seat id to stop (required without --all)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "record without any herdr or agmsg call")
	cmd.Flags().BoolVar(&all, "all", false, "stop every active seat of every org_id (cannot be combined with --org-id or --seat)")
	cmd.Flags().BoolVar(&force, "force", false, "record the seat stopped even when its pane could not be closed (failures become warnings, exit 0)")

	return cmd
}

// rejectFlagsWithAll returns an error when any of flags was passed next to
// --all, before the verb resolves a state dir or touches the manifest: --all
// targets every org_id, so a flag naming one org or seat contradicts it.
func rejectFlagsWithAll(cmd *cobra.Command, verb string, flags ...string) error {
	for _, flag := range flags {
		if cmd.Flags().Changed(flag) {
			return fmt.Errorf("org: %s: --all cannot be combined with --%s", verb, flag)
		}
	}
	return nil
}

// withPrefixOnce returns err with prefix in front unless its message already
// starts with it. (*org.Org).Stop prefixes most of its errors with
// "org: stop: " itself but returns a manifest append failure bare, so
// wrapping every error unconditionally printed the prefix twice.
func withPrefixOnce(prefix string, err error) error {
	if strings.HasPrefix(err.Error(), prefix) {
		return err
	}
	return fmt.Errorf("%s%w", prefix, err)
}

// printSeatFailure writes one seat that stop or disband did not stop cleanly
// to w (stderr). label names the seat: `<org_id>/<seat_id>` for --all,
// `"<seat_id>"` for one org. A Forced failure was recorded `stopped` anyway
// and prints as a warning; any other is still active.
func printSeatFailure(w io.Writer, label string, f org.SeatFailure) {
	if f.Forced {
		_, _ = fmt.Fprintf(w, "warning: seat %s recorded stopped (--force), but its pane was not closed: %v\n", label, f.Err)
		return
	}
	_, _ = fmt.Fprintf(w, "seat %s not stopped: %v\n", label, f.Err)
}

// printWorkspaceFailure is printSeatFailure for a recorded herdr workspace
// that disband did not close.
func printWorkspaceFailure(w io.Writer, label string, f org.WorkspaceFailure) {
	if f.Forced {
		_, _ = fmt.Fprintf(w, "warning: workspace %s recorded closed (--force), but herdr did not close it: %v\n", label, f.Err)
		return
	}
	_, _ = fmt.Fprintf(w, "workspace %s not closed: %v\n", label, f.Err)
}

// printOtherErrs writes to w (stderr) each entry of errs that is not one of
// the already printed failures (shown, matched with errors.Is, which sees
// through the org_id / seat prefixes StopAll and DisbandAll wrap around a
// failure's Err): a manifest read or append failure, for example.
func printOtherErrs(w io.Writer, errs, shown []error) {
	for _, err := range errs {
		if !slices.ContainsFunc(shown, func(s error) bool { return errors.Is(err, s) }) {
			_, _ = fmt.Fprintf(w, "error: %v\n", err)
		}
	}
}

// printStopAllResult prints `ralph org stop --all`'s outcome: one
// `stopped seat <org_id>/<seat_id>` line on stdout per seat stopped with its
// pane closed, each failure on stderr (printSeatFailure), any other error,
// and the codex model-mismatch warning per receipt. It returns the error
// that makes the command exit 1, nil when every active seat was recorded
// `stopped` (forced ones included).
func printStopAllResult(cmd *cobra.Command, r org.StopAllResult) error {
	out, stderr := cmd.OutOrStdout(), cmd.ErrOrStderr()
	if len(r.StoppedSeats) == 0 && len(r.FailedSeats) == 0 && len(r.Errs) == 0 {
		_, _ = fmt.Fprintln(out, "no active seats")
	}
	for _, s := range r.StoppedSeats {
		_, _ = fmt.Fprintf(out, "stopped seat %s/%s\n", s.OrgID, s.SeatID)
	}
	var shown []error
	for _, f := range r.FailedSeats {
		printSeatFailure(stderr, f.OrgID+"/"+f.SeatID, f.SeatFailure)
		shown = append(shown, f.Err)
	}
	printOtherErrs(stderr, r.Errs, shown)
	for _, receipt := range r.ModelReceipts {
		printCodexModelMismatchWarning(cmd, receipt.OrgID+"/"+receipt.SeatID, receipt)
	}
	if len(r.Errs) == 0 {
		return nil
	}
	return fmt.Errorf("org: stop --all: %d error(s), listed above", len(r.Errs))
}

// printStopAllDirectorNote writes to w (stderr) the one line `ralph org stop
// --all` adds about the open headless directors of the ledger
// (StopAllResult.OpenHeadlessDirectors): stop --all leaves them alone,
// since the director is not an org seat, and the line names them and how to
// stop them, repeating hintStateDir as --state-dir (plan
// 2026-10-15-org-director-start, D10). With none it writes nothing; when
// director.jsonl could not be read it writes a warning instead. Neither
// changes the exit code.
func printStopAllDirectorNote(w io.Writer, r org.StopAllResult, hintStateDir string) {
	if r.DirectorReadErr != nil {
		_, _ = fmt.Fprintf(w, "warning: stop --all: the headless directors of the ledger could not be listed: %s\n", inboxField(r.DirectorReadErr.Error()))
		return
	}
	open := r.OpenHeadlessDirectors
	if len(open) == 0 {
		return
	}
	names := make([]string, 0, len(open))
	for _, reg := range open {
		names = append(names, inboxField(reg.ID)+" ("+inboxField(reg.State)+")")
	}
	what, id := "directors "+strings.Join(names, ", ")+" are still open: stop each", "<id>"
	if len(open) == 1 {
		what, id = "director "+names[0]+" is still open: stop it", inboxField(open[0].ID)
	}
	_, _ = fmt.Fprintf(w, "note: stop --all stops org seats only, so the headless %s with %s\n",
		what, org.CommandWithStateDir("ralph org director stop --id "+id, hintStateDir))
}

// closeDeferredSelf is stop's and disband's very last action: when the
// result names the herdr workspace or pane this command runs in (recorded
// closed or stopped in the manifest but left open, the DeferredSelf* ids),
// it closes it now, the workspace when both are set since it holds the
// pane. Closing it ends this process, so the caller prints everything
// first; os.Stdout and os.Stderr are unbuffered, so nothing written so far
// is left behind. runErr is the command's own outcome and is returned as is
// when there is nothing to close or the close succeeds (if the process
// survives it). A failed close has already recorded the seat active and the
// workspace open again (CloseDeferredSelfPane / CloseDeferredSelfWorkspace),
// and is added to runErr, so the command exits 1 and running it again
// retries the close. With force the records stay as written and the failure
// is printed as a warning instead, so --force still exits 0 when nothing
// else failed.
func closeDeferredSelf(cmd *cobra.Command, rt *org.Org, workspaceID, paneID string, force bool, runErr error) error {
	var closeErr error
	switch {
	case workspaceID != "":
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "note: closing this command's own herdr workspace %s last; this ends the command\n", workspaceID)
		closeErr = rt.CloseDeferredSelfWorkspace(workspaceID, force)
	case paneID != "":
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "note: closing this command's own herdr pane %s last; this ends the command\n", paneID)
		closeErr = rt.CloseDeferredSelfPane(paneID, force)
	}
	switch {
	case closeErr == nil:
		return runErr
	case force:
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v\n", closeErr)
		return runErr
	}
	return errors.Join(runErr, closeErr)
}

func newOrgStatusCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var all, jsonOut, live bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show org seat roster",
		Long: "ralph org status shows the seat roster of --org-id (--all adds dry-run\n" +
			"seats). While the org holds a reservation made with --reserve or\n" +
			"`ralph org start --plan` (released by `ralph org disband`), a\n" +
			"`reserved: <path>, ...` line follows the table and --json adds a\n" +
			"`reservation` array. For an org started with --plan, a\n" +
			"`feature: <split>/<slug> branch <branch> worktree <path>` line follows\n" +
			"it and --json adds a `feature` object (split, feature, digest, branch,\n" +
			"worktree, and feature_digest, which is left out for an org whose\n" +
			"record an older ralph wrote). A binding read from a damaged record\n" +
			"ends the line with `(incomplete record)` and adds\n" +
			"`\"incomplete\": true` to the object.\n" +
			"\n" +
			"--live also asks herdr for each active seat's agent_status and reads its\n" +
			"pane: each row gains a HERDR and a SCREEN column (--json, a `live`\n" +
			"object), and a seat herdr reports working while its screen waits for\n" +
			"input gets a `mismatch:` line that names `herdr agent explain`. It\n" +
			"starts no herdr server. Without --live the command reads the ledger only.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath, orgLedgerReadOnly)
			if err != nil {
				return err
			}
			result, err := rt.Status(*orgID, all)
			if err != nil {
				return fmt.Errorf("org: status: %w", err)
			}
			reservation, feature, err := orgReservation(rt, *orgID)
			if err != nil {
				return fmt.Errorf("org: status: %w", err)
			}
			var states []org.LiveSeatState
			if live {
				states = rt.LiveSeatStates(result.Seats)
			}
			if jsonOut {
				return printStatusJSON(cmd, result, states, reservation, feature)
			}
			printStatusTable(cmd, result, states, reservation, feature)
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "include dry-run seats")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "machine-readable JSON output")
	cmd.Flags().BoolVar(&live, "live", false, "also read herdr's state and each active seat's screen (the default reads the ledger only)")

	return cmd
}

// orgSeatJSON is the --json wire shape for one seat. Defined here (not on
// org.SeatStatus, which carries no json tags) so internal/org stays free of
// CLI output-format concerns -- machine-readable field naming is this
// package's responsibility.
type orgSeatJSON struct {
	OrgID     string `json:"org_id"`
	SeatID    string `json:"seat_id"`
	Role      string `json:"role,omitempty"`
	Driver    string `json:"driver,omitempty"`
	Model     string `json:"model,omitempty"`
	Worktree  string `json:"worktree,omitempty"`
	PaneID    string `json:"pane_id,omitempty"`
	AgmsgTeam string `json:"agmsg_team,omitempty"`
	Event     string `json:"event"`
	Active    bool   `json:"active"`
	DryRun    bool   `json:"dry_run,omitempty"`
	Details   string `json:"details,omitempty"`
	TS        string `json:"ts,omitempty"`
	// Live is what --live read for the seat (orgSeatLiveJSON), omitted
	// without --live and for a seat it did not read (not active, a dry run,
	// or no pane), so the JSON without --live is the same as before it.
	Live *orgSeatLiveJSON `json:"live,omitempty"`
}

// orgSeatLiveJSON is the --json wire shape of one seat's org.LiveSeatState
// under `ralph org status --live`: the herdr agent name, herdr's state and,
// when herdr could not tell it, why (herdr_error), the screen's kind, reason,
// and deciding line (screen_line, omitted when there is none), and whether
// herdr reports working while the screen waits for input.
type orgSeatLiveJSON struct {
	Agent        string `json:"agent"`
	HerdrState   string `json:"herdr_state"`
	HerdrError   string `json:"herdr_error,omitempty"`
	Screen       string `json:"screen"`
	ScreenReason string `json:"screen_reason"`
	ScreenLine   string `json:"screen_line,omitempty"`
	Mismatch     bool   `json:"mismatch"`
}

// orgStatusJSON is the --json wire shape for `ralph org status`.
// Reservation is the org's reservation and Feature the split plan feature it
// is bound to (orgReservation), each omitted when the org has none, so an org
// without them prints the same JSON as before reservations and bindings.
type orgStatusJSON struct {
	Seats        []orgSeatJSON   `json:"seats"`
	CorruptLines int             `json:"corrupt_lines"`
	Reservation  []string        `json:"reservation,omitempty"`
	Feature      *orgFeatureJSON `json:"feature,omitempty"`
}

// orgFeatureJSON is the --json wire shape of an org's binding to a split
// plan feature (org.FeatureBinding, which carries no json tags).
// FeatureDigest is the feature digest the binding records (the split plan's
// preamble and the feature's section, org.FeatureBinding.FeatureDigest); a
// binding an older ralph recorded has none, nor does one whose
// `feature_digest=` token is given twice or empty, so the key is left out
// for them. Incomplete is true for a binding read from a damaged record
// (some of the five fields that name the feature empty), and omitted for a
// complete one, so a complete binding an older ralph recorded prints the
// five fields only.
type orgFeatureJSON struct {
	Split         string `json:"split"`
	Feature       string `json:"feature"`
	Digest        string `json:"digest"`
	FeatureDigest string `json:"feature_digest,omitempty"`
	Branch        string `json:"branch"`
	Worktree      string `json:"worktree"`
	Incomplete    bool   `json:"incomplete,omitempty"`
}

// incompleteFeatureNote ends the status feature line of a binding read from
// a damaged record.
const incompleteFeatureNote = " (incomplete record)"

// orgReservation returns orgID's reservation (org.ActiveReservation) and the
// split plan feature that reservation is bound to (org.ActiveFeature) for
// `ralph org status`, each nil when the org has none. (*org.Org).Status
// returns the roster only, so this reads rt's manifest once more, once for
// both.
func orgReservation(rt *org.Org, orgID string) ([]string, *org.FeatureBinding, error) {
	rr, err := rt.Manifest.Read()
	if err != nil {
		return nil, nil, err
	}
	return org.ActiveReservation(rr.Events, orgID), org.ActiveFeature(rr.Events, orgID), nil
}

// printStatusJSON prints result as orgStatusJSON. live is nil without
// --live, and otherwise holds one org.LiveSeatState per seat of result, in
// the same order; a seat it read gets a `live` object.
func printStatusJSON(cmd *cobra.Command, result org.StatusResult, live []org.LiveSeatState, reservation []string, feature *org.FeatureBinding) error {
	seats := make([]orgSeatJSON, len(result.Seats))
	for i, s := range result.Seats {
		seats[i] = orgSeatJSON{
			OrgID: s.OrgID, SeatID: s.SeatID, Role: s.Role, Driver: s.Driver, Model: s.Model,
			Worktree: s.Worktree, PaneID: s.PaneID, AgmsgTeam: s.AgmsgTeam, Event: s.Event,
			Active: s.Active, DryRun: s.DryRun, Details: s.Details, TS: s.TS,
		}
		if live != nil && live[i].Read {
			st := live[i]
			seats[i].Live = &orgSeatLiveJSON{
				Agent: st.Agent, HerdrState: st.HerdrState, HerdrError: st.HerdrErr, Screen: st.Screen,
				ScreenReason: st.ScreenReason, ScreenLine: st.ScreenLine, Mismatch: st.Mismatch,
			}
		}
	}
	payload := orgStatusJSON{Seats: seats, CorruptLines: result.CorruptLines, Reservation: reservation}
	if feature != nil {
		payload.Feature = &orgFeatureJSON{
			Split: feature.Split, Feature: feature.Feature, Digest: feature.Digest, FeatureDigest: feature.FeatureDigest,
			Branch: feature.Branch, Worktree: feature.Worktree, Incomplete: !feature.Complete(),
		}
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// printStatusTable prints the roster, with a model of "" shown as
// org.CLIDefaultModelText (org.ModelOrCLIDefault; --json keeps it ""), then a
// `reserved: <path>, <path>` line when the org holds a reservation (also
// when it has no seat left, e.g. a
// spawn that failed after reserving), then a `feature: <split>/<slug> branch
// <branch> worktree <path>` line when that reservation is bound to a split
// plan feature (ending with incompleteFeatureNote when the binding was read
// from a damaged record), then the corrupt-line warning. live is nil without
// --live; with it (one org.LiveSeatState per seat, in the same order), each
// row gains a HERDR and a SCREEN column, `-` for a seat --live did not read,
// and a `mismatch:` line (printLiveMismatches) follows the rows.
func printStatusTable(cmd *cobra.Command, result org.StatusResult, live []org.LiveSeatState, reservation []string, feature *org.FeatureBinding) {
	out := cmd.OutOrStdout()
	if len(result.Seats) == 0 {
		_, _ = fmt.Fprintln(out, "no seats")
	} else {
		header := "SEAT_ID\tROLE\tDRIVER\tMODEL\tSTATE\tPANE_ID"
		if live != nil {
			header += "\tHERDR\tSCREEN"
		}
		_, _ = fmt.Fprintln(out, header)
		for i, s := range result.Seats {
			state := s.Event
			if s.Active {
				state += " (active)"
			}
			if s.DryRun {
				state += " [dry-run]"
			}
			row := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s", s.SeatID, s.Role, s.Driver, org.ModelOrCLIDefault(s.Model), state, s.PaneID)
			if live != nil {
				row += "\t" + liveStatusColumns(live[i])
			}
			_, _ = fmt.Fprintln(out, row)
		}
		printLiveMismatches(out, result.Seats, live)
	}
	if len(reservation) > 0 {
		_, _ = fmt.Fprintf(out, "reserved: %s\n", strings.Join(reservation, ", "))
	}
	if feature != nil {
		note := ""
		if !feature.Complete() {
			note = incompleteFeatureNote
		}
		_, _ = fmt.Fprintf(out, "feature: %s/%s branch %s worktree %s%s\n", feature.Split, feature.Feature, feature.Branch, feature.Worktree, note)
	}
	if result.CorruptLines > 0 {
		_, _ = fmt.Fprintf(out, "warning: %d corrupt manifest line(s) skipped\n", result.CorruptLines)
	}
}

// liveStatusColumns is the HERDR and SCREEN columns of one row of `ralph org
// status --live`, tab-separated: st's HerdrText and ScreenText, or `-` and
// `-` for a seat --live did not read.
func liveStatusColumns(st org.LiveSeatState) string {
	if !st.Read {
		return "-\t-"
	}
	return st.HerdrText() + "\t" + st.ScreenText()
}

// printLiveMismatches prints, for each seat of seats whose live state is a
// mismatch (herdr reports working while the screen waits for input), one
// line that names `herdr agent explain <agent>`, the command
// `ralph org send`'s screen check error names too. live is nil without
// --live, and then it prints nothing.
func printLiveMismatches(out io.Writer, seats []org.SeatStatus, live []org.LiveSeatState) {
	for i, st := range live {
		if !st.Mismatch {
			continue
		}
		_, _ = fmt.Fprintf(out, "mismatch: seat %s: herdr says working but the screen waits for input; "+
			"see the rule behind herdr's state with `herdr agent explain %s` "+
			"(ralph org send types into such a seat after its screen check)\n",
			seats[i].SeatID, org.ShellWord(st.Agent))
	}
}

func newOrgDisbandCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var dryRun, all, force bool

	cmd := &cobra.Command{
		Use:   "disband",
		Short: "Stop every active seat, close the org's herdr workspace, and disband the org (every org with --all)",
		Long: "ralph org disband stops every active seat of --org-id the way `ralph org\n" +
			"stop` does (closing each seat's pane), then closes the org's herdr\n" +
			"workspace and records `disbanded`. Before closing a pane or the workspace\n" +
			"it checks that herdr labels the pane with the seat id (or, for a seat an\n" +
			"earlier version started, which has no pane label, its tab, unless that\n" +
			"tab label is also the org_id) and the workspace with the org_id; one\n" +
			"that fails the check is not closed and\n" +
			"counts as a failure. When a seat or the workspace cannot be closed, the\n" +
			"org is not disbanded: each failure is listed on stderr and the command\n" +
			"exits 1. Run it again to retry what is left.\n" +
			"\n" +
			"--all disbands every org_id that still needs it, without --org-id, which\n" +
			"includes an org that an older ralph disbanded without closing its\n" +
			"workspace, and keeps going past an org that fails. --force records past\n" +
			"close failures (`stopped`, the workspace closed, `disbanded`), printing\n" +
			"them as warnings and exiting 0; a pane or workspace that failed the check\n" +
			"is not closed itself, only the record is written. Even with --force,\n" +
			"disband closes the org's workspace once herdr confirms its label, which\n" +
			"ends every pane in it, including a seat pane that failed the label check.\n" +
			"When the command runs inside a pane or workspace it closes\n" +
			"(HERDR_PANE_ID / HERDR_WORKSPACE_ID), that one is closed last, after all\n" +
			"output, which ends the command. If that last close fails, the command\n" +
			"records the pane's seat active again and, when it was closing the\n" +
			"workspace, the workspace open again, and exits 1, so running it again\n" +
			"(or disband --all from another pane) retries the close; with --force the\n" +
			"records stay closed and the failure is only a warning.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				if err := rejectFlagsWithAll(cmd, "disband", "org-id"); err != nil {
					return err
				}
				rt, err := newOrgRuntime(cmd, *stateDir, *configPath, orgLedgerMutating)
				if err != nil {
					return err
				}
				result := rt.DisbandAll(org.DisbandAllParams{DryRun: dryRun, Force: force})
				runErr := printDisbandAllResult(cmd, result)
				return closeDeferredSelf(cmd, rt, result.DeferredSelfWorkspaceID, result.DeferredSelfPaneID, force, runErr)
			}
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath, orgLedgerMutating)
			if err != nil {
				return err
			}
			result := rt.Disband(org.DisbandParams{OrgID: *orgID, DryRun: dryRun, Force: force})
			runErr := printDisbandResult(cmd, *orgID, result)
			return closeDeferredSelf(cmd, rt, result.DeferredSelfWorkspaceID, result.DeferredSelfPaneID, force, runErr)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "record without any herdr or agmsg call")
	cmd.Flags().BoolVar(&all, "all", false, "disband every org_id that still needs it (cannot be combined with --org-id)")
	cmd.Flags().BoolVar(&force, "force", false, "record past pane and workspace close failures and disband anyway (failures become warnings, exit 0)")

	return cmd
}

// printDisbandResult prints one org's `ralph org disband` outcome: a
// `stopped seat "<seat_id>"` line on stdout per seat stopped with its pane
// closed, each seat and workspace failure on stderr, any other error, and
// `disbanded org "<org_id>"` only when the org got `disbanded`. It returns
// the error that makes the command exit 1, nil when the org was disbanded.
func printDisbandResult(cmd *cobra.Command, orgID string, r org.DisbandResult) error {
	out, stderr := cmd.OutOrStdout(), cmd.ErrOrStderr()
	for _, seat := range r.StoppedSeats {
		_, _ = fmt.Fprintf(out, "stopped seat %q\n", seat)
	}
	var shown []error
	for _, f := range r.FailedSeats {
		printSeatFailure(stderr, fmt.Sprintf("%q", f.SeatID), f)
		shown = append(shown, f.Err)
	}
	for _, f := range r.FailedWorkspaces {
		printWorkspaceFailure(stderr, fmt.Sprintf("%q", f.WorkspaceID), f)
		shown = append(shown, f.Err)
	}
	printOtherErrs(stderr, r.Errs, shown)
	if r.Disbanded {
		_, _ = fmt.Fprintf(out, "disbanded org %q\n", orgID)
	}
	if len(r.Errs) == 0 {
		return nil
	}
	return fmt.Errorf("org: disband: org_id %q not disbanded: %d error(s), listed above", orgID, len(r.Errs))
}

// printDisbandAllResult is printDisbandResult for `ralph org disband --all`:
// seats as `<org_id>/<seat_id>`, workspaces as `<org_id>/<workspace_id>`, and
// one `disbanded org <org_id>` line per org that got `disbanded`. The error
// names the org_ids left without `disbanded`, which the next run targets
// again.
func printDisbandAllResult(cmd *cobra.Command, r org.DisbandAllResult) error {
	out, stderr := cmd.OutOrStdout(), cmd.ErrOrStderr()
	if len(r.Orgs) == 0 && len(r.Errs) == 0 {
		_, _ = fmt.Fprintln(out, "no orgs to disband")
	}
	for _, s := range r.StoppedSeats {
		_, _ = fmt.Fprintf(out, "stopped seat %s/%s\n", s.OrgID, s.SeatID)
	}
	var shown []error
	for _, f := range r.FailedSeats {
		printSeatFailure(stderr, f.OrgID+"/"+f.SeatID, f.SeatFailure)
		shown = append(shown, f.Err)
	}
	for _, f := range r.FailedWorkspaces {
		printWorkspaceFailure(stderr, f.OrgID+"/"+f.WorkspaceID, f.WorkspaceFailure)
		shown = append(shown, f.Err)
	}
	printOtherErrs(stderr, r.Errs, shown)
	for _, orgID := range r.DisbandedOrgs {
		_, _ = fmt.Fprintf(out, "disbanded org %s\n", orgID)
	}
	if len(r.Errs) == 0 {
		return nil
	}
	var left []string
	for _, orgID := range r.Orgs {
		if !slices.Contains(r.DisbandedOrgs, orgID) {
			left = append(left, orgID)
		}
	}
	if len(left) == 0 {
		return fmt.Errorf("org: disband --all: %d error(s), listed above", len(r.Errs))
	}
	return fmt.Errorf("org: disband --all: %d of %d org(s) not disbanded (%s), errors listed above", len(left), len(r.Orgs), strings.Join(left, ", "))
}

// newOrgReportCmd wires `ralph org report` (AC-4, FR-9 後半): reads the
// manifest + model receipts for --org-id and writes an org-manifest report
// to docs/reports/ via (*org.Org).Report -- see internal/org/report.go's
// BuildOrgReport for the report's sections (roster, event timeline, model
// receipts, known residuals).
func newOrgReportCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var outDir string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Write an org-manifest report (roster, event timeline, model receipts) to docs/reports/",
		Long: "ralph org report reads the manifest and model receipts for --org-id and\n" +
			"writes docs/reports/org-manifest-<org_id>-<date>.md: a roster summary,\n" +
			"the full event timeline, the model-receipts table, and known residuals\n" +
			"(active seat count, corrupt manifest line count). An org with no\n" +
			"recorded events still produces a report, noting that explicitly.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath, orgLedgerReadOnly)
			if err != nil {
				return err
			}
			result := rt.Report(org.ReportParams{OrgID: *orgID, OutDir: outDir})
			if result.Err != nil {
				return fmt.Errorf("org: report: %w", result.Err)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", result.Path)
			return nil
		},
	}

	cmd.Flags().StringVar(&outDir, "out", "", "output directory for the report (default: docs/reports)")

	return cmd
}
