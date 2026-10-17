package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
			"passed on which base. `ralph org merge-order` lists the feature branches\n" +
			"in the order their PRs may be merged, from those records and origin's\n" +
			"branches. `ralph org regate` merges the default branch into a disbanded\n" +
			"feature org's branch, runs the gate on it, and pushes it, so a branch\n" +
			"another PR's merge moved behind is ready again. `ralph org merge`\n" +
			"merges, with gh, the PR of a branch merge-order lists as ready, once it\n" +
			"has checked the PR, its checks, and GitHub's protection of the default\n" +
			"branch again, and records the merge. `ralph org director`\n" +
			"registers the interactive session the human talks to as the director\n" +
			"of the ledger, or launches a headless director in herdr (ralph org\n" +
			"director start), or with [org.director].launch_cmd in the terminal\n" +
			"that command opens.",
	}

	cmd.PersistentFlags().StringVar(&orgID, "org-id", "", "org execution namespace (required by every verb except stop --all, disband --all, merge-order, merge, regate, wait --inbox, watch, approve, the director verbs, and the inbox verbs, which refuse it, and start --plan, which defaults it to the feature's slug; with --plan it is at most 20 characters)")
	cmd.PersistentFlags().StringVar(&stateDir, "state-dir", "", "org manifest/receipts/inbox state directory (default: resolved by org.ResolveOrgStateDir -- env RALPH_ORG_STATE_DIR, else the main worktree's .harness/state/org (shared by its linked worktrees), else the enclosing git toplevel's .harness/state/org, else cwd's .harness/state/org)")
	cmd.PersistentFlags().StringVar(&configPath, "config", "", "path to ralph.toml (default: ./ralph.toml if present, else built-in defaults; without --config, spawn and start read the org-wide limits (see ralph org spawn --help), escalate and director attach read [org.director].enabled, director status reads it and [org.director].auto_merge, merge reads auto_merge, and director start reads enabled with [org.permissions], [org.roles], model_pool and [org.director].launch_cmd, from the main worktree's ralph.toml when the ledger is the main worktree's; ralph status, which has no --config, reads enabled and auto_merge by the same rule)")

	cmd.AddCommand(
		newOrgSpawnCmd(&orgID, &stateDir, &configPath),
		newOrgStartCmd(&orgID, &stateDir, &configPath),
		newOrgSendCmd(&orgID, &stateDir, &configPath),
		newOrgWaitCmd(&orgID, &stateDir, &configPath),
		newOrgReadCmd(&orgID, &stateDir, &configPath),
		newOrgStopCmd(&orgID, &stateDir, &configPath),
		newOrgStatusCmd(&orgID, &stateDir, &configPath),
		newOrgDisbandCmd(&orgID, &stateDir, &configPath),
		newOrgReportCmd(&orgID, &stateDir, &configPath),
		newOrgWatchCmd(&orgID, &stateDir, &configPath),
		newOrgEscalateCmd(&orgID, &stateDir, &configPath),
		newOrgInboxCmd(&stateDir, &configPath),
		newOrgApproveCmd(&stateDir, &configPath),
		newOrgDirectorCmd(&stateDir, &configPath),
		newOrgGateCmd(&orgID, &stateDir, &configPath),
		newOrgRegateCmd(&stateDir, &configPath),
		newOrgMergeOrderCmd(&stateDir, &configPath),
		newOrgMergeCmd(&stateDir, &configPath),
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
// used to detect whether --state-dir was explicitly passed
// (cmd.Flags().Changed("state-dir")) for org.ResolveOrgStateDir's flag >
// env > git-main-worktree > git-toplevel > cwd precedence -- see that
// function's doc comment for the full rationale (fixes the leader/operator
// cwd-split, tech-debt "state-dir の cwd 相対解決", and shares one ledger
// across linked worktrees) -- and to print guardLegacyOrgStateDir's note.
// access is the calling verb's orgLedgerAccess: a mutating verb gets
// guardLegacyOrgStateDir's refusal as the error, before any manifest, herdr,
// or agmsg call. A caller that also needs the resolved config.OrgConfig for
// its own purposes beyond wiring (e.g. resolveModelOrWarn's --model default
// resolution via org.DefaultModelForDriverAndRole) reads it back off the
// returned *org.Org's exported Config field rather than newOrgRuntime
// returning a second value.
func newOrgRuntime(cmd *cobra.Command, stateDir, configPath string, access orgLedgerAccess) (*org.Org, error) {
	resolvedStateDir, stateDirSource := org.ResolveOrgStateDir(stateDir, cmd.Flags().Changed("state-dir"))
	if err := guardLegacyOrgStateDir(cmd, resolvedStateDir, stateDirSource, access); err != nil {
		return nil, err
	}
	return newOrgRuntimeAt(resolvedStateDir, configPath)
}

// newOrgSpawnRuntimeAt is newOrgRuntime for the verbs that start seats
// (`ralph org spawn` and `ralph org start`), the only ones that check the
// org-wide limits: it also replaces the config's MaxOrgs and MaxTotalSeats
// via withMainWorktreeOrgLimits, so the ledger shared by every worktree gets
// one set of limits. Every other verb keeps reading only the caller's config.
// It takes a state dir that org.ResolveOrgStateDir already resolved, the same
// split as newOrgRuntimeAt: each caller also passes the directory on, to
// wireOrgWatchGate (spawn) or wireOrgStartWatchGate (start), and `ralph org
// start --plan` to StartFeatureParams.StateDir and org.FeatureRepoRoot with
// its source.
func newOrgSpawnRuntimeAt(cmd *cobra.Command, resolvedStateDir, stateDirSource, configPath string) (*org.Org, error) {
	if err := guardLegacyOrgStateDir(cmd, resolvedStateDir, stateDirSource, orgLedgerMutating); err != nil {
		return nil, err
	}
	rt, err := newOrgRuntimeAt(resolvedStateDir, configPath)
	if err != nil {
		return nil, err
	}
	rt.Config, err = withMainWorktreeOrgLimits(rt.Config, configPath, resolvedStateDir, stateDirSource)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

// withMainWorktreeOrgLimits returns cfg with MaxOrgs, MaxTotalSeats and
// Approval taken from <main worktree root>/ralph.toml (built-in defaults when
// that file does not exist, as config.Load reads it) when configPath is empty
// and the ledger is the main worktree's (org.LedgerMainWorktreeRoot):
// org.ResolveOrgStateDir placed it there (source "git-main-worktree"), or
// --state-dir or RALPH_ORG_STATE_DIR names the .harness/state/org of the main
// worktree of the repository spawn runs in, as a feature org's leader does.
// Every org_id in that ledger then gets the same org-wide limits, whichever
// subdirectory or linked worktree (with its own ./ralph.toml) spawn runs in
// (plan 2026-10-08-org-limits-reserve, Codex plan advisory 2). Approval
// ([org.approval]) is read only by `ralph org start --plan`, which records it
// in the org's feature binding, by the same rule, so a feature worktree's
// copy of ralph.toml, which its leader can change, does not choose the mode
// (plan 2026-10-12-org-plan-approval, A8); spawn carries it unused. With
// --config, a state dir from --state-dir or RALPH_ORG_STATE_DIR naming
// another directory, or any other source, cfg is returned unchanged. No
// other setting is taken from the main worktree here; escalate, director
// attach and status, merge, and `ralph status` take Director.Enabled and
// Director.AutoMerge by the same rule (withMainWorktreeDirectorEnabled),
// and director start the settings of the headless director
// (withMainWorktreeDirectorStartConfig). A main worktree ralph.toml that
// fails to load is an error, not a fallback.
func withMainWorktreeOrgLimits(cfg config.OrgConfig, configPath, resolvedStateDir, stateDirSource string) (config.OrgConfig, error) {
	mainCfg, path, ok, err := loadMainWorktreeOrgConfig(configPath, resolvedStateDir, stateDirSource)
	if err != nil {
		return config.OrgConfig{}, fmt.Errorf("org: load max_orgs, max_total_seats and [org.approval].mode from the main worktree's %s: %w", path, err)
	}
	if !ok {
		return cfg, nil
	}
	cfg.MaxOrgs, cfg.MaxTotalSeats = mainCfg.MaxOrgs, mainCfg.MaxTotalSeats
	cfg.Approval = mainCfg.Approval
	return cfg, nil
}

// withMainWorktreeDirectorEnabled returns cfg with Director.Enabled (the
// director switch, [org.director].enabled) and Director.AutoMerge
// ([org.director].auto_merge, the mark the /director skill reads for merging
// without asking) taken from <main worktree root>/ralph.toml, by the rule
// withMainWorktreeOrgLimits follows: only when configPath is empty and the
// ledger is the main worktree's (org.LedgerMainWorktreeRoot), with the
// built-in defaults (true and false) when that file does not exist. A
// feature org's leader escalates from its feature worktree, whose ralph.toml
// it can change, so that copy does not choose whether its items are left to
// the director (plan 2026-10-14-org-director-attach, D6), nor whether the
// director merges its PR without asking (plan 2026-10-17-org-auto-merge,
// AC2). With --config, or a ledger that is not the main worktree's, cfg is
// returned unchanged. No other setting is taken; escalate goes by Enabled
// alone. A main worktree ralph.toml that fails to load is an error, which
// `ralph org escalate` handles like a ralph.toml that does not load, `ralph
// org director attach` and `status` and `ralph org merge` return, and `ralph
// status` warns about before it shows the built-in defaults
// (statusDirectorConfig). A verb that reports or acts on the switch or on
// auto_merge reads it through this function, so it sees the value escalate
// and the director go by.
func withMainWorktreeDirectorEnabled(cfg config.OrgConfig, configPath, resolvedStateDir, stateDirSource string) (config.OrgConfig, error) {
	mainCfg, path, ok, err := loadMainWorktreeOrgConfig(configPath, resolvedStateDir, stateDirSource)
	if err != nil {
		return config.OrgConfig{}, fmt.Errorf("org: load [org.director].enabled and auto_merge from the main worktree's %s: %w", path, err)
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
// Permissions ([org.permissions]: default, roles, and codex_verified), Roles
// ([org.roles]), and ModelPool. Only when configPath is empty and the ledger
// is the main worktree's (org.LedgerMainWorktreeRoot), with the built-in
// defaults when that file does not exist. The director acts for every org of
// the ledger, and a feature worktree's ralph.toml, which the leader running
// there can change, must not choose its permission mode or its model (plan
// 2026-10-15-org-director-start, D6 and Scope), any more than whether a
// director is used at all, nor the command start runs to open its terminal
// (plan 2026-10-16-org-director-launch-cmd, D10). With --config, or a ledger
// that is not the main worktree's, cfg is returned unchanged. No other
// setting is taken. A main worktree ralph.toml that fails to load is an
// error, which director start returns.
func withMainWorktreeDirectorStartConfig(cfg config.OrgConfig, configPath, resolvedStateDir, stateDirSource string) (config.OrgConfig, error) {
	mainCfg, path, ok, err := loadMainWorktreeOrgConfig(configPath, resolvedStateDir, stateDirSource)
	if err != nil {
		return config.OrgConfig{}, fmt.Errorf("org: load [org.director].enabled and launch_cmd, [org.permissions], [org.roles] and model_pool from the main worktree's %s: %w", path, err)
	}
	if !ok {
		return cfg, nil
	}
	cfg.Director.Enabled = mainCfg.Director.Enabled
	cfg.Director.LaunchCmd = mainCfg.Director.LaunchCmd
	cfg.Permissions = mainCfg.Permissions
	cfg.Roles = mainCfg.Roles
	cfg.ModelPool = mainCfg.ModelPool
	return cfg, nil
}

// loadMainWorktreeOrgConfig loads the [org] config of <main worktree
// root>/ralph.toml (config.Load: built-in defaults when the file does not
// exist) for withMainWorktreeOrgLimits, withMainWorktreeDirectorEnabled, and
// withMainWorktreeDirectorStartConfig.
// ok is false, with nothing loaded, when configPath is set or the ledger is
// not the main worktree's (org.LedgerMainWorktreeRoot). path is the file it
// loaded, also when err (the load error, unwrapped) is set.
func loadMainWorktreeOrgConfig(configPath, resolvedStateDir, stateDirSource string) (mainCfg config.OrgConfig, path string, ok bool, err error) {
	if configPath != "" {
		return config.OrgConfig{}, "", false, nil
	}
	root, ok := org.LedgerMainWorktreeRoot(resolvedStateDir, stateDirSource)
	if !ok {
		return config.OrgConfig{}, "", false, nil
	}
	path = filepath.Join(root, "ralph.toml")
	cfg, err := config.Load(path)
	if err != nil {
		return config.OrgConfig{}, path, true, err
	}
	return cfg.Org, path, true, nil
}

// newOrgRuntimeAt is newOrgRuntime's shared implementation, taking an
// already-resolved state directory instead of resolving it itself. A caller
// that also needs the resolved directory for its own purposes beyond wiring
// (newOrgSendCmd's read hint, newOrgWatchCmd's banner +
// WatchAllParams.StatusDir) resolves it once via org.ResolveOrgStateDir, calls
// guardLegacyOrgStateDir itself, and calls this directly, instead of
// resolving twice (self-review LOW fix -- each resolution shells out to
// git).
func newOrgRuntimeAt(resolvedStateDir, configPath string) (*org.Org, error) {
	orgCfg, err := resolveOrgConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("org: load config: %w", err)
	}
	runner := driver.ExecRunner{}
	rt := &org.Org{
		Config:   orgCfg,
		Manifest: org.NewManifestStoreAtPath(org.ManifestPathIn(resolvedStateDir)),
		Receipts: org.NewReceiptStoreAtPath(org.ReceiptsPathIn(resolvedStateDir)),
		Herdr:    driver.Herdr{R: runner},
		Agmsg:    driver.Agmsg{R: runner, Home: driver.ResolveAgmsgHome(orgCfg.AgmsgHome)},
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
	return rt, nil
}

// wireOrgInbox sets the inbox fields of rt for the ledger at
// resolvedStateDir: Inbox, EscalationsPath, and DesktopNotify. The inbox and
// the human path of `ralph org escalate` and `ralph org inbox notify` live
// in the same ledger as the manifest. DesktopNotify is nil in production
// (osascript on darwin); a test that escalates through the CLI sets
// orgDesktopNotifyOverride so it shows no real notification. Both
// newOrgRuntimeAt and newOrgEscalateFallbackRuntime call it, so a change to
// this wiring reaches both.
func wireOrgInbox(rt *org.Org, resolvedStateDir string) {
	rt.Inbox = org.NewInboxStore(resolvedStateDir)
	rt.EscalationsPath = org.EscalationsPathIn(resolvedStateDir)
	rt.DesktopNotify = orgDesktopNotifyOverride
}

// orgCodexSessionsDirOverride, orgCodexModelObserveTimeoutOverride, and
// orgCodexModelObserveIntervalOverride are copied into every org.Org this
// package constructs (newOrgRuntimeAt), overriding org.Org's own
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
// InboxPollInterval (newOrgRuntimeAt). Zero in production, which means
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
// dir (env/git-main-worktree/git-toplevel/cwd, see org.ResolveOrgStateDir),
// which either fails with "seat not found" or -- worse -- silently reads a
// different seat that happens to share the same org_id/seat_id in that
// default manifest. An env-resolved or default-resolved state dir is
// deliberately NOT appended: the same shell, run from the same directory
// with the same environment, resolves it the same way when the operator
// runs the printed command themselves, so repeating it would be redundant
// as long as neither changes between the two commands (RALPH_ORG_STATE_DIR
// can go stale if the env changes; the git-main-worktree, git-toplevel,
// and cwd fallbacks depend on cwd the same way, since org.ResolveOrgStateDir
// runs git in the current directory). resolvedStateDir must be the value
// org.ResolveOrgStateDir already returned (always absolute), not the raw
// flag text, so the hint survives a cwd change before the operator acts on
// it.
//
// --config is deliberately not included: newOrgReadCmd's RunE loads
// *configPath into org.Org.Config, but (*org.Org).Read never reads that
// field (it only calls findSeat, which reads the manifest, and
// Herdr.PaneRead) -- so --config cannot affect read's seat lookup.
func orgReadCommandHint(orgID, seat, resolvedStateDir string, stateDirFlagSet bool) string {
	hint := fmt.Sprintf("ralph org read --org-id %s --seat %s", orgID, seat)
	if stateDirFlagSet {
		hint += " --state-dir " + shellQuoteIfNeeded(resolvedStateDir)
	}
	return hint
}

// shellSafeUnquoted matches the characters that need no quoting in a POSIX
// shell word -- shellQuoteIfNeeded's sole caller only ever passes a
// filesystem path, so this set (alphanumerics plus the handful of
// characters a path commonly contains) is deliberately narrow rather than
// attempting to cover every shell-safe character in general.
var shellSafeUnquoted = regexp.MustCompile(`^[A-Za-z0-9_./-]+$`)

// shellQuoteIfNeeded returns s unchanged when it contains only
// shellSafeUnquoted characters; otherwise it wraps s in single quotes,
// escaping any embedded single quote the POSIX way (close the quote, emit
// an escaped literal quote, reopen), so a path containing a space or other
// shell metacharacter stays copy-paste-safe in a printed recovery command.
func shellQuoteIfNeeded(s string) string {
	if shellSafeUnquoted.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// resolveOrgConfig loads the [org] envelope from configPath, falling back
// to ./ralph.toml when configPath is empty and that file exists, and to
// config.Default().Org when neither is present -- matching the plan's
// documented --config default.
func resolveOrgConfig(configPath string) (config.OrgConfig, error) {
	path := configPath
	if path == "" {
		if _, err := os.Stat("ralph.toml"); err == nil {
			path = "ralph.toml"
		} else {
			return config.Default().Org, nil
		}
	}
	cfg, err := config.Load(path)
	if err != nil {
		return config.OrgConfig{}, err
	}
	return cfg.Org, nil
}

// resolveModelOrWarn resolves the effective --model value for driver/role:
// when model is non-blank it is returned unchanged, otherwise it falls back
// to org.DefaultModelForDriverAndRole(cfg, driver, role) (the first
// [org].model_pool entry for driver that is also permitted for role under
// [org.roles]) and prints exactly one warning line to stderr, so both
// `ralph org spawn` and `ralph org start` share the same fallback behavior
// and the same warning wording instead of each hand-rolling it. Threading
// role through the fallback (rather than plain org.DefaultModelForDriver)
// keeps the warning honest: if it prints a model, that model also passes
// ValidateSpawnEnvelope's own ModelAllowedForRole check, instead of
// warn-then-reject when a role-restricted pool's head entry is
// impermissible for the requesting role (self-review MEDIUM-2). stderr is
// cmd.ErrOrStderr() at call sites so tests can capture the warning without
// touching the real process stderr.
func resolveModelOrWarn(cfg config.OrgConfig, driverName, role, model string, stderr io.Writer) (string, error) {
	if strings.TrimSpace(model) != "" {
		return model, nil
	}
	resolved, err := org.DefaultModelForDriverAndRole(cfg, driverName, role)
	if err != nil {
		return "", err
	}
	_, _ = fmt.Fprintf(stderr,
		"org: --model omitted; falling back to first [org].model_pool entry permitted for role %s on %s: %s (pass --model explicitly)\n",
		role, driverName, resolved)
	return resolved, nil
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
	"found by default or named with --state-dir or RALPH_ORG_STATE_DIR (as a\n" +
	"feature org's leader does), so every subdirectory and linked worktree\n" +
	"gets the same limits. In every other case (--config, a --state-dir or\n" +
	"RALPH_ORG_STATE_DIR naming another ledger, a ledger found from the git\n" +
	"toplevel, no git repository) the --config file, else ./ralph.toml, is\n" +
	"used. A refusal says how to free a slot with `ralph org disband`."

// orgReserveFlagUsage is the --reserve usage `ralph org spawn` and `ralph org
// start` share. No backticks: pflag would take the first backticked word as
// the flag's value name.
const orgReserveFlagUsage = "reserve a path for this org, relative to the repo root; repeat the flag for more paths. " +
	"A path ending in / reserves that directory prefix, any other path one file, and . the whole repo. " +
	"Refused when it overlaps the reservation of another running org; released by ralph org disband. " +
	"Satisfies the autonomous-mode --scope requirement"

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
		Long: "ralph org spawn starts one seat of --org-id in a herdr tab and joins it\n" +
			"to the org's agmsg team, recording each step in the manifest.\n" +
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
			"In an org started with `ralph org start --plan`, a spawn of any seat\n" +
			"but the leader is refused while the org's leader seat is not running.\n" +
			"When the org's approval mode (the [org.approval].mode recorded when\n" +
			"the org was started) is per-plan, a spawn of a seat whose role is\n" +
			"implementer is also refused until an approval recorded with\n" +
			"`ralph org approve` since the org's last disband matches the plan\n" +
			"file. Either refusal, --dry-run included, is recorded as `rejected`.\n" +
			"\n" +
			"A spawn (not a --dry-run) into an org is refused, with nothing\n" +
			"recorded, while `ralph org regate` works on that org's branch; run\n" +
			"the spawn again once the regate ends.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			if err := requireSeatIdentifier("--id", seatID); err != nil {
				return err
			}
			for flag, val := range map[string]string{"--role": role, "--driver": driverName, "--cwd": cwd} {
				if strings.TrimSpace(val) == "" {
					return fmt.Errorf("org: %s is required", flag)
				}
			}
			// A retired --role / --id is refused here, before
			// resolveModelOrWarn, so the operator sees the successor
			// guidance instead of a --model fallback warning or a
			// model_pool error for a role that no longer exists. Spawn runs
			// the same check again. The ralph.toml retired-key check stays
			// in Spawn only: it must run after Spawn's idempotent return.
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

			resolvedStateDir, stateDirSource := org.ResolveOrgStateDir(*stateDir, cmd.Flags().Changed("state-dir"))
			rt, err := newOrgSpawnRuntimeAt(cmd, resolvedStateDir, stateDirSource, *configPath)
			if err != nil {
				return err
			}
			wireOrgWatchGate(cmd, rt, resolvedStateDir)
			resolvedModel, err := resolveModelOrWarn(rt.Config, driverName, role, model, cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			result := rt.Spawn(org.SpawnParams{
				OrgID: *orgID, SeatID: seatID, Role: role, Driver: driverName, Model: resolvedModel,
				Cwd: cwd, Prompt: prompt, Scope: scope, Reserve: reserve, TimeoutMS: timeoutMS, DryRun: dryRun,
				LeaderDriver: effectiveLeaderDriver, AllowUnscoped: allowUnscoped,
			})
			printSpawnResult(cmd, result)
			return result.Err
		},
	}

	cmd.Flags().StringVar(&seatID, "id", "", "seat id (required)")
	cmd.Flags().StringVar(&role, "role", "", "seat role (required)")
	cmd.Flags().StringVar(&driverName, "driver", "", "driver CLI: claude|codex (required)")
	cmd.Flags().StringVar(&model, "model", "", "model name or alias (default: first [org].model_pool entry permitted for the role on --driver, with a warning)")
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory for the new seat (required); a relative path is made absolute from the directory this command runs in")
	cmd.Flags().StringVar(&prompt, "prompt", "", "optional initial prompt passed to the agent")
	cmd.Flags().StringVar(&scope, "scope", "", "optional scope description (recorded on the spawned event; substituted into --role templates as {{SCOPE}})")
	cmd.Flags().StringArrayVar(&reserve, "reserve", nil, orgReserveFlagUsage+"; only the leader seat (--id leader) takes it")
	cmd.Flags().StringVar(&leaderDriver, "leader-driver", "claude", "driver (claude|codex) the org's coordinating leader identity runs as, for the agmsg type registered on ensureLeaderJoined")
	cmd.Flags().StringVar(&deprecatedDriver, deprecatedLeaderDriverFlag, "", "deprecated alias of --leader-driver")
	if err := cmd.Flags().MarkDeprecated(deprecatedLeaderDriverFlag, "use --leader-driver"); err != nil {
		panic(err) // the flag was registered on the line above
	}
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 60000, "per-step herdr timeout in milliseconds")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate and record without starting a real seat")
	cmd.Flags().BoolVar(&allowUnscoped, "allow-unscoped", false, "explicitly bypass the autonomous-mode --scope requirement (recorded on the spawned event)")

	return cmd
}

func printSpawnResult(cmd *cobra.Command, r org.SpawnResult) {
	out := cmd.OutOrStdout()
	switch r.Outcome {
	case org.SpawnOutcomeRejected:
		_, _ = fmt.Fprintf(out, "rejected: %v\n", r.Err)
	case org.SpawnOutcomeIdempotent:
		_, _ = fmt.Fprintf(out, "seat %q already spawned (org_id=%s driver=%s model=%s pane_id=%s)\n",
			r.Seat.SeatID, r.Seat.OrgID, r.Seat.Driver, r.Seat.Model, r.Seat.PaneID)
	case org.SpawnOutcomeSpawned:
		_, _ = fmt.Fprintf(out, "spawned seat %q (org_id=%s driver=%s model=%s pane_id=%s dry_run=%t)\n",
			r.Seat.SeatID, r.Seat.OrgID, r.Seat.Driver, r.Seat.Model, r.Seat.PaneID, r.Seat.DryRun)
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
			"feature's body, so --cwd, --scope, --reserve, --allow-unscoped and a\n" +
			"task argument are refused. Running the same start again reuses the\n" +
			"worktree; a start into a running org that was not started for that\n" +
			"feature is refused. `ralph org disband` leaves the worktree and the\n" +
			"branch; `./scripts/ralph-worktree.sh cleanup --id org-<org_id>` removes\n" +
			"them. The org's approval mode, [org.approval].mode (per-plan by\n" +
			"default, or split), is read from the same ralph.toml as max_orgs and\n" +
			"max_total_seats below, recorded with the org's binding to the feature\n" +
			"and named in the leader's task, so a change to the setting applies to\n" +
			"orgs started after it; starting a running feature org again with\n" +
			"another mode is refused.\n" +
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
			"lock but is stale, stopping, or a --once run.\n" +
			"\n" +
			"Both forms are refused, with nothing recorded, while\n" +
			"`ralph org regate` works on the org's branch; run start again once\n" +
			"the regate ends.\n" +
			"\n" +
			orgWideLimitsHelp,
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

			resolvedStateDir, stateDirSource := org.ResolveOrgStateDir(*stateDir, cmd.Flags().Changed("state-dir"))
			rt, err := newOrgSpawnRuntimeAt(cmd, resolvedStateDir, stateDirSource, *configPath)
			if err != nil {
				return err
			}
			wireOrgStartWatchGate(cmd, rt, resolvedStateDir, stateDirSource, *configPath, "start", "leader")
			resolvedModel, err := resolveModelOrWarn(rt.Config, driverName, org.LeaderIdentity, model, cmd.ErrOrStderr())
			if err != nil {
				return err
			}

			result := rt.Spawn(org.SpawnParams{
				OrgID: *orgID, SeatID: org.LeaderIdentity, Role: org.LeaderIdentity,
				Driver: driverName, Model: resolvedModel, Cwd: cwd, Task: task,
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

	cmd.Flags().StringVar(&driverName, "driver", "claude", "driver CLI the leader seat runs as: claude|codex")
	cmd.Flags().StringVar(&model, "model", "", "model name or alias (default: first [org].model_pool entry permitted for the role on --driver, with a warning)")
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory for the leader seat (required without --plan); a relative path is made absolute from the directory this command runs in")
	cmd.Flags().StringVar(&scope, "scope", "", "optional scope description, as for ralph org spawn --scope")
	cmd.Flags().StringArrayVar(&reserve, "reserve", nil, orgReserveFlagUsage)
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 60000, "per-step herdr timeout in milliseconds")
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
		return fmt.Errorf("org: start: --plan takes no task argument (got %q): %s, and the feature's body is its task", args[0], supplies)
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
// it resolves the state dir once, builds the spawn runtime on it (the
// legacy-ledger guard and the main worktree's org-wide limits, as for every
// start) with start's watch gate (wireOrgStartWatchGate, which launches the
// watch when none runs; StartFeature runs it before it makes the worktree),
// and runs (*org.Org).StartFeature with that dir and the main worktree root
// (org.FeatureRepoRoot). The result is printed as a spawn's, followed on
// success by the worktree, the branch and the status hint; a refusal or
// failure is returned as StartFeature's error, which already says whether
// the worktree stays and how to remove it.
func runOrgStartPlan(cmd *cobra.Command, p orgStartPlanParams) error {
	resolvedStateDir, stateDirSource := org.ResolveOrgStateDir(p.stateDir, cmd.Flags().Changed("state-dir"))
	rt, err := newOrgSpawnRuntimeAt(cmd, resolvedStateDir, stateDirSource, p.configPath)
	if err != nil {
		return err
	}
	wireOrgStartWatchGate(cmd, rt, resolvedStateDir, stateDirSource, p.configPath, "start", "leader")
	repoRoot, err := org.FeatureRepoRoot(resolvedStateDir, stateDirSource)
	if err != nil {
		return err
	}
	resolvedModel, err := resolveModelOrWarn(rt.Config, p.driverName, org.LeaderIdentity, p.model, cmd.ErrOrStderr())
	if err != nil {
		return err
	}

	res := rt.StartFeature(org.StartFeatureParams{
		StateDir: resolvedStateDir, RepoRoot: repoRoot, SplitPath: p.plan, Feature: p.feature, OrgID: p.orgID,
		Driver: p.driverName, Model: resolvedModel, TimeoutMS: p.timeoutMS,
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
		to, text     string
		timeoutMS    int
		dryRun       bool
		raw          bool
		enterDelayMS int
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a message to a seat",
		Long: "ralph org send validates --text against the typed message protocol\n" +
			"(internal/org/protocol, see .claude/rules/ralph/agent-messaging.md) before\n" +
			"sending: TYPE must be a known value, TASK_ID is required for\n" +
			"TASK/RESULT/REVIEW/BLOCKED/CONTRACT, and the body must not exceed the\n" +
			"size cap. Pass --raw to bypass validation entirely for free-form text.\n" +
			"After typing the text, send waits briefly and presses Enter once, then\n" +
			"tries to confirm the seat left idle/done. It never presses Enter a\n" +
			"second time: if the submit cannot be confirmed, it prints a warning\n" +
			"instead of guessing -- see --enter-delay-ms below.\n" +
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
			resolvedStateDir, stateDirSource := org.ResolveOrgStateDir(*stateDir, stateDirFlagSet)
			if err := guardLegacyOrgStateDir(cmd, resolvedStateDir, stateDirSource, orgLedgerMutating); err != nil {
				return err
			}
			rt, err := newOrgRuntimeAt(resolvedStateDir, *configPath)
			if err != nil {
				return err
			}
			wireOrgWatchGate(cmd, rt, resolvedStateDir)
			result := rt.Send(org.SendParams{
				OrgID: *orgID, To: to, Text: text, TimeoutMS: timeoutMS, DryRun: dryRun, Raw: raw,
				EnterDelayMS: enterDelayMS,
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
					// return success before ctx expired.
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
						"note: the message text was typed into pane %s of seat %q but not submitted. "+
							"Check it with '%s' and clear or submit it there before sending again: a "+
							"second send would be typed after it.\n",
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
		"overall herdr timeout in milliseconds for one send (idle wait + pre-Enter wait + submit confirmation)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "record without sending a real message")
	cmd.Flags().BoolVar(&raw, "raw", false, "bypass typed message protocol validation")
	cmd.Flags().IntVar(&enterDelayMS, "enter-delay-ms", 0,
		fmt.Sprintf("wait this many milliseconds between typing the message and pressing Enter (0 = built-in default of %d)", org.DefaultSendEnterDelayMS))

	return cmd
}

func newOrgWaitCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var (
		seat      string
		until     string
		timeoutMS int
		inbox     bool
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
			"--until.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if inbox {
				return runOrgWaitInbox(cmd, *stateDir, *configPath, timeoutMS)
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

	return cmd
}

// runOrgWaitInbox is `ralph org wait --inbox`: (*org.Org).WaitInbox, then
// one line per open item on stdout. The inbox is shared by every org in the
// ledger, so --org-id is refused rather than ignored, as are --seat and
// --until, which name a seat wait.
func runOrgWaitInbox(cmd *cobra.Command, stateDir, configPath string, timeoutMS int) error {
	for _, flag := range []string{"org-id", "seat", "until"} {
		if cmd.Flags().Changed(flag) {
			return fmt.Errorf("org: wait: --inbox cannot be combined with --%s", flag)
		}
	}
	rt, err := newOrgRuntime(cmd, stateDir, configPath, orgLedgerReadOnly)
	if err != nil {
		return err
	}
	items, err := rt.WaitInbox(time.Duration(timeoutMS) * time.Millisecond)
	for _, item := range items {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n",
			inboxField(item.ID), inboxField(item.OrgID), inboxField(item.Type), inboxSummary(item.Body))
	}
	return err
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
			"Before sending C-c or closing, it checks that herdr has the pane in a tab\n" +
			"labelled with the seat id, inside a workspace labelled with the org_id (the\n" +
			"labels spawn gave them). A pane that fails the check (a label differs, or\n" +
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
	var all, jsonOut bool

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
			"worktree). A binding read from a damaged record ends the line with\n" +
			"`(incomplete record)` and adds `\"incomplete\": true` to the object.",
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
			if jsonOut {
				return printStatusJSON(cmd, result, reservation, feature)
			}
			printStatusTable(cmd, result, reservation, feature)
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "include dry-run seats")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "machine-readable JSON output")

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
// plan feature (org.FeatureBinding, which carries no json tags). Incomplete
// is true for a binding read from a damaged record (some fields empty), and
// omitted for a complete one, so a complete binding prints the five fields
// only.
type orgFeatureJSON struct {
	Split      string `json:"split"`
	Feature    string `json:"feature"`
	Digest     string `json:"digest"`
	Branch     string `json:"branch"`
	Worktree   string `json:"worktree"`
	Incomplete bool   `json:"incomplete,omitempty"`
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

func printStatusJSON(cmd *cobra.Command, result org.StatusResult, reservation []string, feature *org.FeatureBinding) error {
	seats := make([]orgSeatJSON, len(result.Seats))
	for i, s := range result.Seats {
		seats[i] = orgSeatJSON{
			OrgID: s.OrgID, SeatID: s.SeatID, Role: s.Role, Driver: s.Driver, Model: s.Model,
			Worktree: s.Worktree, PaneID: s.PaneID, AgmsgTeam: s.AgmsgTeam, Event: s.Event,
			Active: s.Active, DryRun: s.DryRun, Details: s.Details, TS: s.TS,
		}
	}
	payload := orgStatusJSON{Seats: seats, CorruptLines: result.CorruptLines, Reservation: reservation}
	if feature != nil {
		payload.Feature = &orgFeatureJSON{
			Split: feature.Split, Feature: feature.Feature, Digest: feature.Digest, Branch: feature.Branch, Worktree: feature.Worktree,
			Incomplete: !feature.Complete(),
		}
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// printStatusTable prints the roster, then a `reserved: <path>, <path>` line
// when the org holds a reservation (also when it has no seat left, e.g. a
// spawn that failed after reserving), then a `feature: <split>/<slug> branch
// <branch> worktree <path>` line when that reservation is bound to a split
// plan feature (ending with incompleteFeatureNote when the binding was read
// from a damaged record), then the corrupt-line warning.
func printStatusTable(cmd *cobra.Command, result org.StatusResult, reservation []string, feature *org.FeatureBinding) {
	out := cmd.OutOrStdout()
	if len(result.Seats) == 0 {
		_, _ = fmt.Fprintln(out, "no seats")
	} else {
		_, _ = fmt.Fprintln(out, "SEAT_ID\tROLE\tDRIVER\tMODEL\tSTATE\tPANE_ID")
		for _, s := range result.Seats {
			state := s.Event
			if s.Active {
				state += " (active)"
			}
			if s.DryRun {
				state += " [dry-run]"
			}
			_, _ = fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\t%s\n", s.SeatID, s.Role, s.Driver, s.Model, state, s.PaneID)
		}
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

func newOrgDisbandCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var dryRun, all, force bool

	cmd := &cobra.Command{
		Use:   "disband",
		Short: "Stop every active seat, close the org's herdr workspace, and disband the org (every org with --all)",
		Long: "ralph org disband stops every active seat of --org-id the way `ralph org\n" +
			"stop` does (closing each seat's pane), then closes the org's herdr\n" +
			"workspace and records `disbanded`. Before closing a pane or the workspace\n" +
			"it checks that herdr labels the pane's tab with the seat id and the\n" +
			"workspace with the org_id; one that fails the check is not closed and\n" +
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
			"ends every pane in it, including a seat pane that failed the tab check.\n" +
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
