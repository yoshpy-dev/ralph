package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
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
			"absent or stopped.",
	}

	cmd.PersistentFlags().StringVar(&orgID, "org-id", "", "org execution namespace (required, except for stop --all and disband --all)")
	cmd.PersistentFlags().StringVar(&stateDir, "state-dir", "", "org manifest/receipts state directory (default: resolved by org.ResolveOrgStateDir -- env RALPH_ORG_STATE_DIR, else the main worktree's .harness/state/org (shared by its linked worktrees), else the enclosing git toplevel's .harness/state/org, else cwd's .harness/state/org)")
	cmd.PersistentFlags().StringVar(&configPath, "config", "", "path to ralph.toml (default: ./ralph.toml if present, else built-in defaults; without --config, spawn and start read max_orgs and max_total_seats from the main worktree's ralph.toml when the ledger is the main worktree's)")

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

// newOrgSpawnRuntime is newOrgRuntime for the verbs that start seats
// (`ralph org spawn` and `ralph org start`), the only ones that check the
// org-wide limits: it also replaces the config's MaxOrgs and MaxTotalSeats
// via withMainWorktreeOrgLimits, so the ledger shared by every worktree gets
// one set of limits. Every other verb keeps reading only the caller's config.
func newOrgSpawnRuntime(cmd *cobra.Command, stateDir, configPath string) (*org.Org, error) {
	resolvedStateDir, stateDirSource := org.ResolveOrgStateDir(stateDir, cmd.Flags().Changed("state-dir"))
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

// withMainWorktreeOrgLimits returns cfg with MaxOrgs and MaxTotalSeats taken
// from <main worktree root>/ralph.toml (built-in defaults when that file does
// not exist, as config.Load reads it) when configPath is empty and
// org.ResolveOrgStateDir placed the ledger under the main worktree (source
// "git-main-worktree", org.MainWorktreeRoot). Every org_id in that ledger then
// gets the same org-wide limits, whichever subdirectory or linked worktree
// (with its own ./ralph.toml) spawn runs in (plan
// 2026-10-08-org-limits-reserve, Codex plan advisory 2). With --config, a
// state dir from --state-dir or RALPH_ORG_STATE_DIR, or any other source, cfg
// is returned unchanged. No other setting is read from the main worktree. A
// main worktree ralph.toml that fails to load is an error, not a fallback.
func withMainWorktreeOrgLimits(cfg config.OrgConfig, configPath, resolvedStateDir, stateDirSource string) (config.OrgConfig, error) {
	if configPath != "" {
		return cfg, nil
	}
	root, ok := org.MainWorktreeRoot(resolvedStateDir, stateDirSource)
	if !ok {
		return cfg, nil
	}
	path := filepath.Join(root, "ralph.toml")
	mainCfg, err := config.Load(path)
	if err != nil {
		return config.OrgConfig{}, fmt.Errorf("org: load max_orgs and max_total_seats from the main worktree's %s: %w", path, err)
	}
	cfg.MaxOrgs, cfg.MaxTotalSeats = mainCfg.Org.MaxOrgs, mainCfg.Org.MaxTotalSeats
	return cfg, nil
}

// newOrgRuntimeAt is newOrgRuntime's shared implementation, taking an
// already-resolved state directory instead of resolving it itself. A caller
// that also needs the resolved directory for its own purposes beyond wiring
// (newOrgSendCmd's read hint, newOrgWatchCmd's banner +
// WatchParams.StatusDir) resolves it once via org.ResolveOrgStateDir, calls
// guardLegacyOrgStateDir itself, and calls this directly, instead of
// resolving twice (self-review LOW fix -- each resolution shells out to
// git).
func newOrgRuntimeAt(resolvedStateDir, configPath string) (*org.Org, error) {
	orgCfg, err := resolveOrgConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("org: load config: %w", err)
	}
	runner := driver.ExecRunner{}
	return &org.Org{
		Config:   orgCfg,
		Manifest: org.NewManifestStoreAtPath(org.ManifestPathIn(resolvedStateDir)),
		Receipts: org.NewReceiptStoreAtPath(org.ReceiptsPathIn(resolvedStateDir)),
		Herdr:    driver.Herdr{R: runner},
		Agmsg:    driver.Agmsg{R: runner, Home: driver.ResolveAgmsgHome(orgCfg.AgmsgHome)},
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
	}, nil
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
// ValidateSpawnEnvelope's own modelAllowedForRole check, instead of
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
const orgWideLimitsHelp = "Before starting a seat, spawn checks [org].max_seats for the org and two\n" +
	"limits every org_id in the ledger shares: [org].max_orgs (running orgs)\n" +
	"and [org].max_total_seats (active seats across all orgs). When --config\n" +
	"is not given and the ledger is the main worktree's (no --state-dir or\n" +
	"RALPH_ORG_STATE_DIR), those two are read from the main worktree's\n" +
	"ralph.toml (built-in defaults when it has none), so every subdirectory\n" +
	"and linked worktree gets the same limits. A refusal says how to free a\n" +
	"slot with `ralph org disband`."

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
			"disbanded: the same paths again pass, different ones are refused.",
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
			if err := org.RetiredRoleInputErr(role, seatID, prompt); err != nil {
				printSpawnResult(cmd, org.SpawnResult{Outcome: org.SpawnOutcomeRejected, Err: err})
				return err
			}
			effectiveLeaderDriver, err := resolveLeaderDriver(cmd, leaderDriver, deprecatedDriver)
			if err != nil {
				return err
			}

			rt, err := newOrgSpawnRuntime(cmd, *stateDir, *configPath)
			if err != nil {
				return err
			}
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
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory for the new seat (required)")
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
func newOrgStartCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var (
		driverName, model, cwd, scope string
		reserve                       []string
		timeoutMS                     int
		allowUnscoped                 bool
	)

	cmd := &cobra.Command{
		Use:   "start <task>",
		Short: "Spawn a headless leader seat (sugar over `ralph org spawn --role leader`)",
		Long: "ralph org start is a thin wrapper over the same Spawn saga every other\n" +
			"`ralph org spawn` call uses: it always spawns the org's coordinating\n" +
			"\"leader\" identity itself (seat id \"leader\", role \"leader\"), expands\n" +
			"internal/org/prompts/leader.md with the task argument substituted for\n" +
			"{{TASK}} and a one-line [org] envelope summary substituted for\n" +
			"{{ENVELOPE}}. Envelope validation, the permission-mode gate, and\n" +
			"manifest/receipt bookkeeping all flow through Spawn exactly as they\n" +
			"would for any other seat. See .claude/skills/org/SKILL.md for the\n" +
			"leader's full operating manual.\n" +
			"\n" +
			orgWideLimitsHelp + "\n" +
			"\n" +
			"--reserve claims paths of the repo for the org until it is disbanded:\n" +
			"starting the org again with the same paths passes, with different ones\n" +
			"is refused.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			rt, err := newOrgSpawnRuntime(cmd, *stateDir, *configPath)
			if err != nil {
				return err
			}
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
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory for the leader seat (required)")
	cmd.Flags().StringVar(&scope, "scope", "", "optional scope description (see `ralph org spawn --scope`)")
	cmd.Flags().StringArrayVar(&reserve, "reserve", nil, orgReserveFlagUsage)
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 60000, "per-step herdr timeout in milliseconds")
	cmd.Flags().BoolVar(&allowUnscoped, "allow-unscoped", false, "explicitly bypass the autonomous-mode --scope requirement")

	return cmd
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
			"instead of guessing -- see --enter-delay-ms below.",
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
				return fmt.Errorf("org: send: %w", result.Err)
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
	)

	cmd := &cobra.Command{
		Use:   "wait",
		Short: "Wait for a seat to reach one of the given states",
		Long: "ralph org wait defaults --until to \"idle,done\": live-probed herdr\n" +
			"(v0.7.5) reports an interactive agent resting at its input prompt as\n" +
			"\"done\" (turn finished), not \"idle\" -- waiting on \"idle\" alone times\n" +
			"out against a perfectly receptive seat (same finding that fixed `ralph\n" +
			"org send`'s own wait, internal/org/verbs.go's Send). --timeout-ms\n" +
			"defaults to a bounded 60000ms so a headless leader following this\n" +
			"command's own default cannot block forever; pass --timeout-ms 0 to\n" +
			"explicitly opt into an unbounded wait.",
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

	cmd.Flags().StringVar(&seat, "seat", "", "seat id to wait on (required)")
	cmd.Flags().StringVar(&until, "until", "idle,done", "comma-separated states to wait for (idle,done,blocked)")
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 60000, "wait timeout in milliseconds, bounded by default (pass 0 to explicitly wait unbounded)")

	return cmd
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
			"the seat stays recorded stopped and the failure is only a warning.",
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
			reservation, err := orgReservation(rt, *orgID)
			if err != nil {
				return fmt.Errorf("org: status: %w", err)
			}
			if jsonOut {
				return printStatusJSON(cmd, result, reservation)
			}
			printStatusTable(cmd, result, reservation)
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
// Reservation is the org's reservation (orgReservation), omitted when it holds
// none, so an org without one prints the same JSON as before reservations.
type orgStatusJSON struct {
	Seats        []orgSeatJSON `json:"seats"`
	CorruptLines int           `json:"corrupt_lines"`
	Reservation  []string      `json:"reservation,omitempty"`
}

// orgReservation returns orgID's reservation (org.ActiveReservation) for
// `ralph org status`, nil when it holds none. (*org.Org).Status returns the
// roster only, so this reads rt's manifest once more.
func orgReservation(rt *org.Org, orgID string) ([]string, error) {
	rr, err := rt.Manifest.Read()
	if err != nil {
		return nil, err
	}
	return org.ActiveReservation(rr.Events, orgID), nil
}

func printStatusJSON(cmd *cobra.Command, result org.StatusResult, reservation []string) error {
	seats := make([]orgSeatJSON, len(result.Seats))
	for i, s := range result.Seats {
		seats[i] = orgSeatJSON{
			OrgID: s.OrgID, SeatID: s.SeatID, Role: s.Role, Driver: s.Driver, Model: s.Model,
			Worktree: s.Worktree, PaneID: s.PaneID, AgmsgTeam: s.AgmsgTeam, Event: s.Event,
			Active: s.Active, DryRun: s.DryRun, Details: s.Details, TS: s.TS,
		}
	}
	payload := orgStatusJSON{Seats: seats, CorruptLines: result.CorruptLines, Reservation: reservation}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// printStatusTable prints the roster, then a `reserved: <path>, <path>` line
// when the org holds a reservation (also when it has no seat left, e.g. a
// spawn that failed after reserving), then the corrupt-line warning.
func printStatusTable(cmd *cobra.Command, result org.StatusResult, reservation []string) {
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

// newOrgWatchCmd wires `ralph org watch` (PR④ pulse layer, AC-3/3b/3c/4/5):
// a deterministic, interval-driven condition loop over (*org.Org).RunWatch.
// All condition evaluation, ALERT dedupe, and deadman escalation logic lives
// in internal/org/watch.go -- this command resolves the state directory (the
// same one manifest/receipts already live in, via org.ResolveOrgStateDir)
// and wires flags through to org.WatchParams.
func newOrgWatchCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var (
		intervalSeconds int
		once            bool
	)

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Run the deterministic pulse-layer watchdog for an org",
		Long: "ralph org watch evaluates watch conditions every --interval-seconds\n" +
			"(default: [org.watchdog].interval_seconds) for --org-id:\n" +
			"heartbeat-stall / process-liveness / worktree-scope-change\n" +
			"ALERTs sent to the leader seat, and a deadman escalation\n" +
			"(<state-dir>/escalations.jsonl + stderr banner + best-effort darwin\n" +
			"notification) when the leader does not respond within\n" +
			"[org].deadman_minutes. Pass --once to run exactly one cycle and exit\n" +
			"(useful for cron/smoke); the default loops until the command's\n" +
			"context is done (e.g. SIGINT).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			// Resolved exactly once (self-review LOW fix) and passed through
			// to newOrgRuntimeAt instead of also re-resolving inside a second
			// newOrgRuntime call. stateDirSource (tech-debt: "watchdog
			// deferred LOW (1)") is surfaced in the startup banner below so
			// an operator can tell which precedence tier (flag/env/
			// git-main-worktree/git-toplevel/cwd) produced resolvedStateDir
			// without re-deriving ResolveOrgStateDir's logic by hand.
			resolvedStateDir, stateDirSource := org.ResolveOrgStateDir(*stateDir, cmd.Flags().Changed("state-dir"))
			if err := guardLegacyOrgStateDir(cmd, resolvedStateDir, stateDirSource, orgLedgerMutating); err != nil {
				return err
			}
			rt, err := newOrgRuntimeAt(resolvedStateDir, *configPath)
			if err != nil {
				return err
			}
			cycles := 0
			if once {
				cycles = 1
			}
			// Effective interval (self-review LOW fix): the raw
			// --interval-seconds flag value is 0 by default, so the banner
			// used to print a cadence that was never actually running --
			// ResolveWatchInterval mirrors RunWatch's own fallback chain so
			// the banner reports what will really execute.
			effectiveInterval := org.ResolveWatchInterval(time.Duration(intervalSeconds)*time.Second, rt.Config.Watchdog)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "watching org %q (interval=%s once=%t state-dir=%s (source: %s))\n",
				*orgID, effectiveInterval, once, resolvedStateDir, stateDirSource)
			hooks, watcherWG := newWatchdogHooks(cmd.Context(), rt, cmd.ErrOrStderr())
			err = rt.RunWatch(cmd.Context(), org.WatchParams{
				OrgID:     *orgID,
				Interval:  effectiveInterval,
				Cycles:    cycles,
				StatusDir: resolvedStateDir,
			}, hooks)
			// Wait for any still-in-flight on-demand watcher goroutine
			// before returning (self-review M-5 fix): without this, `--once`
			// returned as soon as cycle 1's synchronous pulse evaluation
			// finished, killing the process before an OnSemanticTrigger
			// goroutine it had just started could ever produce a watcher
			// receipt or ALERT. Bounded by RunWatcher's own
			// watcherInvokeTimeout, and -- for the long-running (non-`--once`)
			// mode -- by cmd.Context() itself: newWatchdogHooks threads
			// cmd.Context() into the tracked goroutine's RunWatcher call
			// (self-review cycle-2 M2-2 fix), so a SIGINT-cancelled context
			// unwinds an in-flight judgment call immediately instead of
			// running out the full 60s bound.
			watcherWG.Wait()
			return err
		},
	}

	cmd.Flags().IntVar(&intervalSeconds, "interval-seconds", 0, "pulse cycle interval in seconds (default: [org.watchdog].interval_seconds)")
	cmd.Flags().BoolVar(&once, "once", false, "run exactly one cycle and exit")

	return cmd
}

// newWatchdogHooks builds the org.WatchHooks `ralph org watch` wires into
// RunWatch (PR④ Slice 4, AC-6): when rt.Config.Watchdog.WatcherEnabled is
// false, OnSemanticTrigger is left nil (WatchHooks' documented no-op
// default) -- the pulse layer never invokes an LLM on its own. When true,
// OnSemanticTrigger runs (*org.Org).RunWatcher in its own goroutine so a
// hang or slow judgment call can never delay the pulse loop that triggered
// it (Codex advisory 3) -- RunWatch's own cycle already returned by the
// time the goroutine even starts running.
//
// A single-flight guard (busy) keeps at most one on-demand judgment in
// flight at a time: the atomic compare-and-swap happens synchronously in
// OnSemanticTrigger itself (before the goroutine is even spawned), so two
// triggers arriving from the same synchronous evaluateCycle pass (e.g. two
// seats both flagged in one cycle) are ordered deterministically -- the
// first wins the flag and starts its goroutine, the second sees busy
// already set and is skipped (recorded to stderr, never queued or run
// concurrently).
//
// An abnormal verdict (anything but org.WatcherVerdictNormal) is sent to
// leader as an ALERT via rt.SendWatchdogAlert (identity-level Agmsg.Send, not
// the seat-steering Send verb -- see that method's doc comment for why:
// Send's findSeat lookup fails, silently dropping the message, in the
// normal "session-promoted leader" org shape where no leader SEAT was ever
// spawned), in the same message shape watch.go's own (unexported) sendAlert
// already uses for pulse-layer ALERTs, so ALERT traffic stays uniform
// regardless of which layer produced it.
//
// The returned *sync.WaitGroup (self-review M-5 fix) tracks every
// OnSemanticTrigger goroutine this closure starts; newOrgWatchCmd's RunE
// waits on it after RunWatch returns, so `--once` (Cycles: 1, RunWatch
// returns as soon as cycle 1 finishes) cannot exit the process out from
// under an in-flight on-demand judgment call before it produces a watcher
// receipt or ALERT. When WatcherEnabled is false the returned WaitGroup has
// nothing ever added to it, so Wait() returns immediately.
//
// ctx is the command's own context (cmd.Context() at the newOrgWatchCmd call
// site), not context.Background() (self-review cycle-2 M2-2 fix): the
// tracked goroutine's RunWatcher/SendWatchdogAlert calls are threaded through
// it so a SIGINT-cancelled ctx unwinds an in-flight judgment call right away
// instead of running out RunWatcher's own watcherInvokeTimeout (a fixed 60s
// bound) first -- that gap regressed the long-running (non-`--once`) `ralph
// org watch` mode's Ctrl-C responsiveness when the M-5 fix above first added
// this WaitGroup.
func newWatchdogHooks(ctx context.Context, rt *org.Org, stderr io.Writer) (org.WatchHooks, *sync.WaitGroup) {
	var wg sync.WaitGroup
	if !rt.Config.Watchdog.WatcherEnabled {
		return org.WatchHooks{}, &wg
	}

	var busy int32
	return org.WatchHooks{
		OnSemanticTrigger: func(orgID, seatID, conditionType, evidence string) {
			if !atomic.CompareAndSwapInt32(&busy, 0, 1) {
				_, _ = fmt.Fprintf(stderr, "watchdog: watcher already in flight, skipping semantic trigger for org %q seat %q condition %q\n",
					orgID, seatID, conditionType)
				return
			}
			wg.Go(func() {
				defer atomic.StoreInt32(&busy, 0)
				verdict, err := rt.RunWatcher(ctx, rt.Config.Watchdog, org.WatcherParams{
					OrgID: orgID, SeatID: seatID, ConditionType: conditionType, Evidence: evidence,
				})
				if err != nil {
					_, _ = fmt.Fprintf(stderr, "watchdog: watcher error for org %q seat %q condition %q: %v\n",
						orgID, seatID, conditionType, err)
					return
				}
				if verdict.Verdict == org.WatcherVerdictNormal {
					return
				}
				msg := fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: watcher_%s\n\nwatcher verdict=%s reason=%s",
					orgID, seatID, conditionType, verdict.Verdict, verdict.Reason)
				if err := rt.SendWatchdogAlert(ctx, orgID, msg); err != nil {
					_, _ = fmt.Fprintf(stderr, "watchdog: failed to ALERT leader for org %q seat %q verdict %q: %v\n",
						orgID, seatID, verdict.Verdict, err)
				}
			})
		},
	}, &wg
}
