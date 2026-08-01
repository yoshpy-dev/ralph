package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	cmd.PersistentFlags().StringVar(&stateDir, "state-dir", ".harness/state/org", "org manifest/receipts state directory")
	cmd.PersistentFlags().StringVar(&configPath, "config", "", "path to ralph.toml (default: ./ralph.toml if present, else built-in defaults)")

	cmd.AddCommand(
		newOrgSpawnCmd(&orgID, &stateDir, &configPath),
		newOrgSendCmd(&orgID, &stateDir, &configPath),
		newOrgWaitCmd(&orgID, &stateDir, &configPath),
		newOrgReadCmd(&orgID, &stateDir, &configPath),
		newOrgStopCmd(&orgID, &stateDir, &configPath),
		newOrgStatusCmd(&orgID, &stateDir, &configPath),
		newOrgDisbandCmd(&orgID, &stateDir, &configPath),
	)

	return cmd
}

// requireOrgID returns an error unless orgID is non-blank -- the shared
// --org-id validation every verb needs (global required flag, checked
// manually rather than via cobra's MarkPersistentFlagRequired so tests and
// error messages stay simple and uniform).
func requireOrgID(orgID string) error {
	if strings.TrimSpace(orgID) == "" {
		return fmt.Errorf("org: --org-id is required")
	}
	return nil
}

// newOrgRuntime constructs an org.Org wired to real driver adapters
// (driver.ExecRunner, which shells out to the herdr/agmsg binaries on PATH)
// and manifest/receipt stores rooted at stateDir.
func newOrgRuntime(stateDir, configPath string) (*org.Org, error) {
	orgCfg, err := resolveOrgConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("org: load config: %w", err)
	}
	runner := driver.ExecRunner{}
	return &org.Org{
		Config:   orgCfg,
		Manifest: org.NewManifestStoreAtPath(filepath.Join(stateDir, "manifest.jsonl")),
		Receipts: org.NewReceiptStoreAtPath(filepath.Join(stateDir, "model-receipts.jsonl")),
		Herdr:    driver.Herdr{R: runner},
		Agmsg:    driver.Agmsg{R: runner},
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
		seatID, role, driverName, model, cwd, prompt string
		timeoutMS                                    int
		dryRun                                       bool
	)

	cmd := &cobra.Command{
		Use:   "spawn",
		Short: "Spawn a new org seat",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			for flag, val := range map[string]string{"--id": seatID, "--role": role, "--driver": driverName, "--model": model, "--cwd": cwd} {
				if strings.TrimSpace(val) == "" {
					return fmt.Errorf("org: %s is required", flag)
				}
			}

			rt, err := newOrgRuntime(*stateDir, *configPath)
			if err != nil {
				return err
			}
			result := rt.Spawn(org.SpawnParams{
				OrgID: *orgID, SeatID: seatID, Role: role, Driver: driverName, Model: model,
				Cwd: cwd, Prompt: prompt, TimeoutMS: timeoutMS, DryRun: dryRun,
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
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 60000, "per-step herdr timeout in milliseconds")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate and record without starting a real seat")

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

func newOrgSendCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var (
		to, text  string
		timeoutMS int
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a message to a seat",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			if strings.TrimSpace(to) == "" {
				return fmt.Errorf("org: --to is required")
			}
			rt, err := newOrgRuntime(*stateDir, *configPath)
			if err != nil {
				return err
			}
			result := rt.Send(org.SendParams{OrgID: *orgID, To: to, Text: text, TimeoutMS: timeoutMS, DryRun: dryRun})
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireOrgID(*orgID); err != nil {
				return err
			}
			if strings.TrimSpace(seat) == "" {
				return fmt.Errorf("org: --seat is required")
			}
			rt, err := newOrgRuntime(*stateDir, *configPath)
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
	cmd.Flags().StringVar(&until, "until", "idle", "comma-separated states to wait for (idle,done,blocked)")
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 0, "wait timeout in milliseconds (0 = no timeout)")

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
			if strings.TrimSpace(seat) == "" {
				return fmt.Errorf("org: --seat is required")
			}
			rt, err := newOrgRuntime(*stateDir, *configPath)
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
			if strings.TrimSpace(seat) == "" {
				return fmt.Errorf("org: --seat is required")
			}
			rt, err := newOrgRuntime(*stateDir, *configPath)
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
			rt, err := newOrgRuntime(*stateDir, *configPath)
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
			rt, err := newOrgRuntime(*stateDir, *configPath)
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
