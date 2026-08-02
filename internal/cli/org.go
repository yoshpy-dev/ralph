package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	cmd.PersistentFlags().StringVar(&orgID, "org-id", "", "org execution namespace (required)")
	cmd.PersistentFlags().StringVar(&stateDir, "state-dir", "", "org manifest/receipts state directory (default: resolved by org.ResolveOrgStateDir -- env RALPH_ORG_STATE_DIR, else the enclosing git repo's toplevel .harness/state/org, else cwd's .harness/state/org)")
	cmd.PersistentFlags().StringVar(&configPath, "config", "", "path to ralph.toml (default: ./ralph.toml if present, else built-in defaults)")

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
// used only to detect whether --state-dir was explicitly passed
// (cmd.Flags().Changed("state-dir")) for org.ResolveOrgStateDir's flag >
// env > git-toplevel > cwd precedence -- see that function's doc comment
// for the full rationale (fixes the lead/operator cwd-split, tech-debt
// "state-dir の cwd 相対解決"). A caller that also needs the resolved
// config.OrgConfig for its own purposes beyond wiring (e.g.
// newOrgStartCmd's --model default resolution via
// org.DefaultModelForDriver) reads it back off the returned *org.Org's
// exported Config field rather than newOrgRuntime returning a second
// value.
func newOrgRuntime(cmd *cobra.Command, stateDir, configPath string) (*org.Org, error) {
	resolvedStateDir, _ := org.ResolveOrgStateDir(stateDir, cmd.Flags().Changed("state-dir"))
	return newOrgRuntimeAt(resolvedStateDir, configPath)
}

// newOrgRuntimeAt is newOrgRuntime's shared implementation, taking an
// already-resolved state directory instead of resolving it itself. A caller
// that also needs the resolved directory for its own purposes beyond wiring
// (today, only newOrgWatchCmd's banner + WatchParams.StatusDir) resolves it
// once via org.ResolveOrgStateDir and calls this directly, instead of
// resolving twice (self-review LOW fix -- each resolution shells out to `git
// rev-parse --show-toplevel`).
func newOrgRuntimeAt(resolvedStateDir, configPath string) (*org.Org, error) {
	orgCfg, err := resolveOrgConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("org: load config: %w", err)
	}
	runner := driver.ExecRunner{}
	return &org.Org{
		Config:   orgCfg,
		Manifest: org.NewManifestStoreAtPath(filepath.Join(resolvedStateDir, "manifest.jsonl")),
		Receipts: org.NewReceiptStoreAtPath(filepath.Join(resolvedStateDir, "model-receipts.jsonl")),
		Herdr:    driver.Herdr{R: runner},
		Agmsg:    driver.Agmsg{R: runner, Home: driver.ResolveAgmsgHome(orgCfg.AgmsgHome)},
	}, nil
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

func newOrgSpawnCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var (
		seatID, role, driverName, model, cwd, prompt, scope, leadDriver string
		timeoutMS                                                       int
		dryRun, allowUnscoped                                           bool
	)

	cmd := &cobra.Command{
		Use:   "spawn",
		Short: "Spawn a new org seat",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			if err := requireSeatIdentifier("--id", seatID); err != nil {
				return err
			}
			for flag, val := range map[string]string{"--role": role, "--driver": driverName, "--model": model, "--cwd": cwd} {
				if strings.TrimSpace(val) == "" {
					return fmt.Errorf("org: %s is required", flag)
				}
			}

			rt, err := newOrgRuntime(cmd, *stateDir, *configPath)
			if err != nil {
				return err
			}
			result := rt.Spawn(org.SpawnParams{
				OrgID: *orgID, SeatID: seatID, Role: role, Driver: driverName, Model: model,
				Cwd: cwd, Prompt: prompt, Scope: scope, TimeoutMS: timeoutMS, DryRun: dryRun,
				LeadDriver: leadDriver, AllowUnscoped: allowUnscoped,
			})
			printSpawnResult(cmd, result)
			return result.Err
		},
	}

	cmd.Flags().StringVar(&seatID, "id", "", "seat id (required)")
	cmd.Flags().StringVar(&role, "role", "", "seat role (required)")
	cmd.Flags().StringVar(&driverName, "driver", "", "driver CLI: claude|codex (required)")
	cmd.Flags().StringVar(&model, "model", "", "model name or alias (required)")
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory for the new seat (required)")
	cmd.Flags().StringVar(&prompt, "prompt", "", "optional initial prompt passed to the agent")
	cmd.Flags().StringVar(&scope, "scope", "", "optional scope description (recorded on the spawned event; substituted into --role templates as {{SCOPE}})")
	cmd.Flags().StringVar(&leadDriver, "lead-driver", "claude", "driver (claude|codex) the org's coordinating lead identity runs as, for the agmsg type registered on ensureLeadJoined")
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
}

// newOrgStartCmd wires `ralph org start` -- headless-lead spawn sugar over
// (*org.Org).Spawn, per the plan's design decision ("`org start` = lead 座席
// の spawn 糖衣", docs/plans/active/2026-08-02-org-runtime-lead.md). It
// always spawns SeatID == Role == org.LeadIdentity ("lead"): the org's
// coordinating agmsg identity and the lead seat are, by design, the same
// seat -- see the leadSelfSpawn branch in internal/org/spawn.go's Spawn.
// Every other concern (envelope validation, the permission-mode gate,
// manifest/receipt bookkeeping) flows through the same saga every other
// `ralph org spawn` call uses; this command does not special-case the lead
// runtime object in any way beyond picking its SeatID/Role and required
// positional task argument.
func newOrgStartCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var (
		driverName, model, cwd, scope string
		timeoutMS                     int
		allowUnscoped                 bool
	)

	cmd := &cobra.Command{
		Use:   "start <task>",
		Short: "Spawn a headless lead seat (sugar over `ralph org spawn --role lead`)",
		Long: "ralph org start is a thin wrapper over the same Spawn saga every other\n" +
			"`ralph org spawn` call uses: it always spawns the org's coordinating\n" +
			"\"lead\" identity itself (seat id \"lead\", role \"lead\"), expands\n" +
			"internal/org/prompts/lead.md with the task argument substituted for\n" +
			"{{TASK}} and a one-line [org] envelope summary substituted for\n" +
			"{{ENVELOPE}}. Envelope validation, the permission-mode gate, and\n" +
			"manifest/receipt bookkeeping all flow through Spawn exactly as they\n" +
			"would for any other seat. See .claude/skills/org/SKILL.md for the\n" +
			"lead's full operating manual.",
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

			rt, err := newOrgRuntime(cmd, *stateDir, *configPath)
			if err != nil {
				return err
			}
			resolvedModel := model
			if strings.TrimSpace(resolvedModel) == "" {
				resolvedModel, err = org.DefaultModelForDriver(rt.Config, driverName)
				if err != nil {
					return err
				}
			}

			result := rt.Spawn(org.SpawnParams{
				OrgID: *orgID, SeatID: org.LeadIdentity, Role: org.LeadIdentity,
				Driver: driverName, Model: resolvedModel, Cwd: cwd, Task: task,
				Scope: scope, TimeoutMS: timeoutMS, AllowUnscoped: allowUnscoped,
			})
			printSpawnResult(cmd, result)
			if result.Err == nil {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(),
					"hint: ralph org status --org-id %s ; attach with herdr to observe the lead pane\n", *orgID)
			}
			return result.Err
		},
	}

	cmd.Flags().StringVar(&driverName, "driver", "claude", "driver CLI the lead seat runs as: claude|codex")
	cmd.Flags().StringVar(&model, "model", "", "model name or alias (default: first [org].model_pool entry for --driver)")
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory for the lead seat (required)")
	cmd.Flags().StringVar(&scope, "scope", "", "optional scope description (see `ralph org spawn --scope`)")
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 60000, "per-step herdr timeout in milliseconds")
	cmd.Flags().BoolVar(&allowUnscoped, "allow-unscoped", false, "explicitly bypass the autonomous-mode --scope requirement")

	return cmd
}

func newOrgSendCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var (
		to, text  string
		timeoutMS int
		dryRun    bool
		raw       bool
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a message to a seat",
		Long: "ralph org send validates --text against the typed message protocol\n" +
			"(internal/org/protocol, see .claude/rules/agent-messaging.md) before\n" +
			"sending: TYPE must be a known value, TASK_ID is required for\n" +
			"TASK/RESULT/REVIEW/BLOCKED/CONTRACT, and the body must not exceed the\n" +
			"size cap. Pass --raw to bypass validation entirely for free-form text.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			if err := requireSeatIdentifier("--to", to); err != nil {
				return err
			}
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath)
			if err != nil {
				return err
			}
			result := rt.Send(org.SendParams{OrgID: *orgID, To: to, Text: text, TimeoutMS: timeoutMS, DryRun: dryRun, Raw: raw})
			if result.Err != nil {
				return fmt.Errorf("org: send: %w", result.Err)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "sent message to seat %q\n", to)
			return nil
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "target seat id (required)")
	cmd.Flags().StringVar(&text, "text", "", "message text")
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 30000, "idle-wait timeout in milliseconds before sending")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "record without sending a real message")
	cmd.Flags().BoolVar(&raw, "raw", false, "bypass typed message protocol validation")

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
			"defaults to a bounded 60000ms so a headless lead following this\n" +
			"command's own default cannot block forever; pass --timeout-ms 0 to\n" +
			"explicitly opt into an unbounded wait.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			if err := requireSeatIdentifier("--seat", seat); err != nil {
				return err
			}
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath)
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
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath)
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
		seat   string
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a seat",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			if err := requireSeatIdentifier("--seat", seat); err != nil {
				return err
			}
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath)
			if err != nil {
				return err
			}
			result := rt.Stop(org.StopParams{OrgID: *orgID, Seat: seat, DryRun: dryRun})
			if result.Err != nil {
				return fmt.Errorf("org: stop: %w", result.Err)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "stopped seat %q\n", seat)
			return nil
		},
	}

	cmd.Flags().StringVar(&seat, "seat", "", "seat id to stop (required)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "record without sending a real stop signal")

	return cmd
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
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath)
			if err != nil {
				return err
			}
			result, err := rt.Status(*orgID, all)
			if err != nil {
				return fmt.Errorf("org: status: %w", err)
			}
			if jsonOut {
				return printStatusJSON(cmd, result)
			}
			printStatusTable(cmd, result)
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
type orgStatusJSON struct {
	Seats        []orgSeatJSON `json:"seats"`
	CorruptLines int           `json:"corrupt_lines"`
}

func printStatusJSON(cmd *cobra.Command, result org.StatusResult) error {
	seats := make([]orgSeatJSON, len(result.Seats))
	for i, s := range result.Seats {
		seats[i] = orgSeatJSON{
			OrgID: s.OrgID, SeatID: s.SeatID, Role: s.Role, Driver: s.Driver, Model: s.Model,
			Worktree: s.Worktree, PaneID: s.PaneID, AgmsgTeam: s.AgmsgTeam, Event: s.Event,
			Active: s.Active, DryRun: s.DryRun, Details: s.Details, TS: s.TS,
		}
	}
	payload := orgStatusJSON{Seats: seats, CorruptLines: result.CorruptLines}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func printStatusTable(cmd *cobra.Command, result org.StatusResult) {
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
	if result.CorruptLines > 0 {
		_, _ = fmt.Fprintf(out, "warning: %d corrupt manifest line(s) skipped\n", result.CorruptLines)
	}
}

func newOrgDisbandCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "disband",
		Short: "Stop every active seat and disband the org",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath)
			if err != nil {
				return err
			}
			result := rt.Disband(org.DisbandParams{OrgID: *orgID, DryRun: dryRun})
			for _, seat := range result.StoppedSeats {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "stopped seat %q\n", seat)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "disbanded org %q\n", *orgID)
			if len(result.Errs) > 0 {
				return fmt.Errorf("org: disband encountered %d error(s), first: %w", len(result.Errs), result.Errs[0])
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "record without stopping real seats")

	return cmd
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
			rt, err := newOrgRuntime(cmd, *stateDir, *configPath)
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
// All condition evaluation, budget-cutoff, ALERT dedupe, and deadman
// escalation logic lives in internal/org/watch.go -- this command resolves
// the state directory (the same one manifest/receipts already live in, via
// org.ResolveOrgStateDir) and wires flags through to org.WatchParams.
func newOrgWatchCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var (
		intervalSeconds int
		once            bool
	)

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Run the deterministic pulse-layer watchdog for an org",
		Long: "ralph org watch evaluates watch conditions every --interval-seconds\n" +
			"(default: [org.watchdog].interval_seconds) for --org-id: seat/org\n" +
			"wall-clock budget cutoff (auto Stop, the same verb `ralph org stop`\n" +
			"uses -- StopParams.Reason records the condition/threshold/observed\n" +
			"value), heartbeat-stall / process-liveness / worktree-scope-change\n" +
			"ALERTs sent to the lead seat, and a deadman escalation\n" +
			"(<state-dir>/escalations.jsonl + stderr banner + best-effort darwin\n" +
			"notification) when the lead does not respond within\n" +
			"[org].deadman_minutes. Pass --once to run exactly one cycle and exit\n" +
			"(useful for cron/smoke); the default loops until the command's\n" +
			"context is done (e.g. SIGINT).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			// Resolved exactly once (self-review LOW fix) and passed through
			// to newOrgRuntimeAt instead of also re-resolving inside a second
			// newOrgRuntime call.
			resolvedStateDir, _ := org.ResolveOrgStateDir(*stateDir, cmd.Flags().Changed("state-dir"))
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
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "watching org %q (interval=%s once=%t state-dir=%s)\n",
				*orgID, effectiveInterval, once, resolvedStateDir)
			hooks, watcherWG := newWatchdogHooks(rt, cmd.ErrOrStderr())
			err = rt.RunWatch(cmd.Context(), org.WatchParams{
				OrgID:     *orgID,
				Interval:  effectiveInterval,
				Cycles:    cycles,
				StatusDir: resolvedStateDir,
			}, hooks)
			// Wait for any still-in-flight on-demand watcher goroutine
			// before returning (self-review M-7 fix): without this, `--once`
			// returned as soon as cycle 1's synchronous pulse evaluation
			// finished, killing the process before an OnSemanticTrigger
			// goroutine it had just started could ever produce a watcher
			// receipt or ALERT. Bounded by RunWatcher's own
			// watcherInvokeTimeout, so this can never hang the command.
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
// lead as an ALERT via rt.SendWatchdogAlert (identity-level Agmsg.Send, not
// the seat-steering Send verb -- see that method's doc comment for why:
// Send's findSeat lookup fails, silently dropping the message, in the
// normal "session-promoted lead" org shape where no lead SEAT was ever
// spawned), in the same message shape watch.go's own (unexported) sendAlert
// already uses for pulse-layer ALERTs, so ALERT traffic stays uniform
// regardless of which layer produced it.
//
// The returned *sync.WaitGroup (self-review M-7 fix) tracks every
// OnSemanticTrigger goroutine this closure starts; newOrgWatchCmd's RunE
// waits on it after RunWatch returns, so `--once` (Cycles: 1, RunWatch
// returns as soon as cycle 1 finishes) cannot exit the process out from
// under an in-flight on-demand judgment call before it produces a watcher
// receipt or ALERT. When WatcherEnabled is false the returned WaitGroup has
// nothing ever added to it, so Wait() returns immediately.
func newWatchdogHooks(rt *org.Org, stderr io.Writer) (org.WatchHooks, *sync.WaitGroup) {
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
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer atomic.StoreInt32(&busy, 0)
				verdict, err := rt.RunWatcher(context.Background(), rt.Config.Watchdog, org.WatcherParams{
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
				if err := rt.SendWatchdogAlert(context.Background(), orgID, msg); err != nil {
					_, _ = fmt.Fprintf(stderr, "watchdog: failed to ALERT lead for org %q seat %q verdict %q: %v\n",
						orgID, seatID, verdict.Verdict, err)
				}
			}()
		},
	}, &wg
}
