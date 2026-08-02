package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// newStatusCmd wires the top-level `ralph status` command. Unlike `ralph org
// status` (which requires --org-id and shows one org's roster), this command
// summarizes every org found in the manifest at the resolved state
// directory (or a single org via --org-id), plus the latest `ralph org
// watch` heartbeat and pending-alert count for each org when available. It
// reads org.ManifestStore/org.Roster directly (read-only, no herdr/agmsg
// process required) -- the same derivation `ralph org status` uses, just
// grouped across every org_id instead of filtered to exactly one.
func newStatusCmd() *cobra.Command {
	var (
		stateDir string
		orgID    string
		jsonOut  bool
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show org runtime roster status",
		Long: "Displays the org runtime roster across every org_id found in the manifest\n" +
			"(or a single org via --org-id): seats with role/driver/model/state, active\n" +
			"seat counts, and -- when `ralph org watch` has run for that org -- the last\n" +
			"watch heartbeat and pending alert count.",
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedStateDir, _ := org.ResolveOrgStateDir(stateDir, cmd.Flags().Changed("state-dir"))
			return runStatus(cmd, resolvedStateDir, orgID, jsonOut)
		},
	}

	cmd.Flags().StringVar(&stateDir, "state-dir", "", "org manifest/receipts state directory (default: resolved by org.ResolveOrgStateDir -- env RALPH_ORG_STATE_DIR, else the enclosing git repo's toplevel .harness/state/org, else cwd's .harness/state/org)")
	cmd.Flags().StringVar(&orgID, "org-id", "", "filter roster to a single org_id (default: every org found in the manifest)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "machine-readable JSON output")

	return cmd
}

// orgGroup is one org_id's derived roster, grouped for display.
type orgGroup struct {
	OrgID string
	Seats []org.SeatStatus
}

func runStatus(cmd *cobra.Command, stateDir, filterOrgID string, jsonOut bool) error {
	store := org.NewManifestStore(stateDir)
	rr, err := store.Read()
	if err != nil {
		return fmt.Errorf("status: read manifest: %w", err)
	}

	seats := org.Roster(rr.Events, org.RosterOptions{})
	if filterOrgID != "" {
		filtered := make([]org.SeatStatus, 0, len(seats))
		for _, s := range seats {
			if s.OrgID == filterOrgID {
				filtered = append(filtered, s)
			}
		}
		seats = filtered
	}

	groups := groupSeatsByOrg(seats)

	if len(groups) == 0 {
		return printStatusEmpty(cmd, stateDir, filterOrgID, jsonOut)
	}

	if jsonOut {
		return printStatusJSONAllOrgs(cmd, stateDir, groups, rr.CorruptLines)
	}
	printStatusTableAllOrgs(cmd, stateDir, groups, rr.CorruptLines)
	return nil
}

// groupSeatsByOrg splits Roster's flat, OrgID-sorted seat slice into
// per-org groups, preserving Roster's existing sort order (org.Roster
// already sorts by OrgID then SeatID -- see manifest.go).
func groupSeatsByOrg(seats []org.SeatStatus) []orgGroup {
	var groups []orgGroup
	var current *orgGroup
	for _, s := range seats {
		if current == nil || current.OrgID != s.OrgID {
			groups = append(groups, orgGroup{OrgID: s.OrgID})
			current = &groups[len(groups)-1]
		}
		current.Seats = append(current.Seats, s)
	}
	return groups
}

// orgWatchHeartbeat mirrors the small subset of fields this command needs
// from the JSON shape `ralph org watch` persists to
// org.WatchStatusFileName(org_id) (internal/org/watch.go's unexported
// watchStatusFile). That type stays unexported by design -- internal/org
// keeps its watch-status JSON shape private -- so this command declares its
// own read-only mirror struct instead of reaching into internal/org
// internals, and reads the file directly via os.ReadFile.
type orgWatchHeartbeat struct {
	LastCycleTS   string                     `json:"last_cycle_ts"`
	Cycles        int                        `json:"cycles"`
	PendingAlerts map[string]json.RawMessage `json:"pending_alerts"`
}

// readOrgWatchHeartbeat returns the watch heartbeat for orgID, or ok=false
// when no watch-status file exists yet (i.e. `ralph org watch` has never
// run for this org) or it cannot be parsed.
func readOrgWatchHeartbeat(stateDir, orgID string) (heartbeat orgWatchHeartbeat, ok bool) {
	path := filepath.Join(stateDir, org.WatchStatusFileName(orgID))
	data, err := os.ReadFile(path)
	if err != nil {
		return orgWatchHeartbeat{}, false
	}
	if err := json.Unmarshal(data, &heartbeat); err != nil {
		return orgWatchHeartbeat{}, false
	}
	return heartbeat, true
}

func printStatusEmpty(cmd *cobra.Command, stateDir, filterOrgID string, jsonOut bool) error {
	out := cmd.OutOrStdout()
	if jsonOut {
		payload := struct {
			StateDir string `json:"state_dir"`
			OrgID    string `json:"org_id,omitempty"`
			Orgs     []any  `json:"orgs"`
		}{StateDir: stateDir, OrgID: filterOrgID, Orgs: []any{}}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}
	if filterOrgID != "" {
		_, _ = fmt.Fprintf(out, "org runtime state が見つかりません(org-id=%s, state-dir=%s)。ralph org spawn で開始してください。\n", filterOrgID, stateDir)
	} else {
		_, _ = fmt.Fprintf(out, "org runtime state が見つかりません(state-dir=%s)。ralph org spawn で開始してください。\n", stateDir)
	}
	_, _ = fmt.Fprintln(out, "Run `ralph doctor` to check environment readiness.")
	return nil
}

// statusSeatJSON is the --json wire shape for one seat, mirroring
// orgSeatJSON in org.go (internal/org's SeatStatus carries no json tags on
// purpose -- output-format concerns stay in the cli package).
type statusSeatJSON struct {
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

type statusWatchJSON struct {
	LastCycleTS   string `json:"last_cycle_ts"`
	Cycles        int    `json:"cycles"`
	PendingAlerts int    `json:"pending_alerts"`
}

type statusOrgJSON struct {
	OrgID       string           `json:"org_id"`
	Seats       []statusSeatJSON `json:"seats"`
	ActiveCount int              `json:"active_count"`
	TotalCount  int              `json:"total_count"`
	Watch       *statusWatchJSON `json:"watch,omitempty"`
}

func printStatusJSONAllOrgs(cmd *cobra.Command, stateDir string, groups []orgGroup, corruptLines int) error {
	orgsJSON := make([]statusOrgJSON, 0, len(groups))
	for _, g := range groups {
		orgsJSON = append(orgsJSON, buildStatusOrgJSON(stateDir, g))
	}
	payload := struct {
		StateDir     string          `json:"state_dir"`
		Orgs         []statusOrgJSON `json:"orgs"`
		CorruptLines int             `json:"corrupt_lines,omitempty"`
	}{StateDir: stateDir, Orgs: orgsJSON, CorruptLines: corruptLines}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func buildStatusOrgJSON(stateDir string, g orgGroup) statusOrgJSON {
	seatsJSON := make([]statusSeatJSON, len(g.Seats))
	active := 0
	for i, s := range g.Seats {
		if s.Active {
			active++
		}
		seatsJSON[i] = statusSeatJSON{
			SeatID: s.SeatID, Role: s.Role, Driver: s.Driver, Model: s.Model,
			Worktree: s.Worktree, PaneID: s.PaneID, AgmsgTeam: s.AgmsgTeam, Event: s.Event,
			Active: s.Active, DryRun: s.DryRun, Details: s.Details, TS: s.TS,
		}
	}
	result := statusOrgJSON{OrgID: g.OrgID, Seats: seatsJSON, ActiveCount: active, TotalCount: len(g.Seats)}
	if hb, ok := readOrgWatchHeartbeat(stateDir, g.OrgID); ok {
		result.Watch = &statusWatchJSON{
			LastCycleTS:   hb.LastCycleTS,
			Cycles:        hb.Cycles,
			PendingAlerts: len(hb.PendingAlerts),
		}
	}
	return result
}

func printStatusTableAllOrgs(cmd *cobra.Command, stateDir string, groups []orgGroup, corruptLines int) {
	out := cmd.OutOrStdout()

	// Deterministic org order for readability, matching Roster's own
	// OrgID-then-SeatID sort (groupSeatsByOrg preserves it already, but
	// sort explicitly here so this stays correct even if Roster's own
	// ordering guarantee ever changes upstream).
	sort.Slice(groups, func(i, j int) bool { return groups[i].OrgID < groups[j].OrgID })

	for i, g := range groups {
		if i > 0 {
			_, _ = fmt.Fprintln(out)
		}
		active := 0
		for _, s := range g.Seats {
			if s.Active {
				active++
			}
		}
		_, _ = fmt.Fprintf(out, "org_id: %s (active %d/%d)\n", g.OrgID, active, len(g.Seats))
		_, _ = fmt.Fprintln(out, "  SEAT_ID\tROLE\tDRIVER\tMODEL\tSTATE\tPANE_ID")
		for _, s := range g.Seats {
			state := s.Event
			if s.Active {
				state += " (active)"
			}
			if s.DryRun {
				state += " [dry-run]"
			}
			_, _ = fmt.Fprintf(out, "  %s\t%s\t%s\t%s\t%s\t%s\n", s.SeatID, s.Role, s.Driver, s.Model, state, s.PaneID)
		}
		if hb, ok := readOrgWatchHeartbeat(stateDir, g.OrgID); ok {
			_, _ = fmt.Fprintf(out, "  watch: last heartbeat %s (cycle %d), pending alerts: %d\n",
				hb.LastCycleTS, hb.Cycles, len(hb.PendingAlerts))
		}
	}

	if corruptLines > 0 {
		_, _ = fmt.Fprintf(out, "\nwarning: %d corrupt manifest line(s) skipped\n", corruptLines)
	}
}
