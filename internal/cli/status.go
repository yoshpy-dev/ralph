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
	store := org.NewManifestStoreAtPath(org.ManifestPathIn(stateDir))
	rr, err := store.Read()
	if err != nil {
		return fmt.Errorf("status: read manifest: %w", err)
	}

	// IncludeDryRun: true -- this top-level summary command has no --all
	// flag (unlike `ralph org status`, which gates dry-run visibility
	// behind one, see newOrgStatusCmd) and is documented as showing "every
	// org found in the manifest", so it always includes dry-run seats as
	// rows. As a side benefit, this also makes the "[dry-run]" marker
	// printStatusTableAllOrgs/buildStatusOrgJSON already render per seat
	// reachable. This flip only widens the per-row listing -- the
	// aggregate active/total counts below are computed separately from a
	// real-seats-only roster (see buildRealSeatCounts) so a dry-run seat
	// showing up as a row never inflates the numbers that gate spawning.
	seats := org.Roster(rr.Events, org.RosterOptions{IncludeDryRun: true})
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
	realCounts := buildRealSeatCounts(rr.Events, groups)

	if len(groups) == 0 {
		return printStatusEmpty(cmd, stateDir, filterOrgID, jsonOut, rr.CorruptLines)
	}

	if jsonOut {
		return printStatusJSONAllOrgs(cmd, stateDir, filterOrgID, groups, realCounts, rr.CorruptLines)
	}
	printStatusTableAllOrgs(cmd, stateDir, groups, realCounts, rr.CorruptLines)
	return nil
}

// realSeatCount holds one org_id's active/total seat counts derived from
// the real (non-dry-run) roster only. This is the number that actually
// gates `ralph org spawn`'s max_seats check (org.ActiveSeatCount /
// RosterOptions{}, the same convention internal/org/report.go:201 and
// internal/org/spawn.go:333,788 use) -- distinct from the dry-run-inclusive
// rows this command renders per seat.
type realSeatCount struct {
	Active int
	Total  int
}

// buildRealSeatCounts derives the real (non-dry-run) active/total seat
// counts for every org_id present in groups. It re-derives the roster from
// events with RosterOptions{} (IncludeDryRun defaults to false), so a
// dry-run seat contributes a display row (via the IncludeDryRun: true
// roster built in runStatus) but never moves either count here. groups is
// only consulted to enumerate which org_ids need a count entry.
func buildRealSeatCounts(events []org.ManifestEvent, groups []orgGroup) map[string]realSeatCount {
	real := org.Roster(events, org.RosterOptions{})
	totals := make(map[string]int, len(groups))
	actives := make(map[string]int, len(groups))
	for _, s := range real {
		totals[s.OrgID]++
		if s.Active {
			actives[s.OrgID]++
		}
	}
	counts := make(map[string]realSeatCount, len(groups))
	for _, g := range groups {
		counts[g.OrgID] = realSeatCount{Active: actives[g.OrgID], Total: totals[g.OrgID]}
	}
	return counts
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

func printStatusEmpty(cmd *cobra.Command, stateDir, filterOrgID string, jsonOut bool, corruptLines int) error {
	out := cmd.OutOrStdout()
	if jsonOut {
		payload := statusJSON{StateDir: stateDir, OrgID: filterOrgID, Orgs: []statusOrgJSON{}, CorruptLines: corruptLines}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}
	// An empty roster does not mean "nothing to report": a manifest whose
	// every line failed to parse also yields zero groups, and that is a
	// data-integrity signal the operator needs, not silence (see
	// printStatusTableAllOrgs's identical warning for the non-empty path,
	// and `ralph org status`'s corrupt-count warning in org.go, which is
	// likewise unconditional on seat count).
	if filterOrgID != "" {
		_, _ = fmt.Fprintf(out, "no org runtime state found (org-id=%s, state-dir=%s) — run `ralph org spawn` to start one.\n", filterOrgID, stateDir)
	} else {
		_, _ = fmt.Fprintf(out, "no org runtime state found (state-dir=%s) — run `ralph org spawn` to start one.\n", stateDir)
	}
	if corruptLines > 0 {
		_, _ = fmt.Fprintf(out, "warning: %d corrupt manifest line(s) skipped\n", corruptLines)
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

// statusOrgJSON is one org's rendered roster plus its aggregate counts.
// Seats can include dry-run rows (see runStatus's IncludeDryRun: true
// roster); ActiveCount and TotalCount deliberately do not derive from
// len(Seats) or count Seats' Active field -- they are real-seat-only
// (RosterOptions{}), matching org.ActiveSeatCount's own convention
// (internal/org/report.go:201, internal/org/spawn.go:333,788) so a
// dry-run seat appearing in Seats never changes what these two numbers
// report.
type statusOrgJSON struct {
	OrgID       string           `json:"org_id"`
	Seats       []statusSeatJSON `json:"seats"`
	ActiveCount int              `json:"active_count"`
	TotalCount  int              `json:"total_count"`
	Watch       *statusWatchJSON `json:"watch,omitempty"`
}

// statusJSON is the single `ralph status --json` wire shape, used by both
// the empty-roster path (printStatusEmpty) and the populated path
// (printStatusJSONAllOrgs). Keeping one struct for both means a machine
// consumer never has to branch on which shape it received: `org_id` is only
// present when --org-id filtered the roster, `corrupt_lines` only when the
// manifest actually had corrupt lines, and `orgs` is always an array (empty
// or populated).
type statusJSON struct {
	StateDir     string          `json:"state_dir"`
	OrgID        string          `json:"org_id,omitempty"`
	Orgs         []statusOrgJSON `json:"orgs"`
	CorruptLines int             `json:"corrupt_lines,omitempty"`
}

func printStatusJSONAllOrgs(cmd *cobra.Command, stateDir, filterOrgID string, groups []orgGroup, realCounts map[string]realSeatCount, corruptLines int) error {
	orgsJSON := make([]statusOrgJSON, 0, len(groups))
	for _, g := range groups {
		orgsJSON = append(orgsJSON, buildStatusOrgJSON(stateDir, g, realCounts[g.OrgID]))
	}
	payload := statusJSON{StateDir: stateDir, OrgID: filterOrgID, Orgs: orgsJSON, CorruptLines: corruptLines}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// buildStatusOrgJSON renders one org's seats (dry-run-inclusive, per g.Seats)
// alongside the org's real-seat-only active/total counts (realCount --
// see buildRealSeatCounts). ActiveCount/TotalCount deliberately do not
// count g.Seats directly: that slice can include dry-run rows, and the
// aggregate must match the number that gates `ralph org spawn`.
func buildStatusOrgJSON(stateDir string, g orgGroup, realCount realSeatCount) statusOrgJSON {
	seatsJSON := make([]statusSeatJSON, len(g.Seats))
	for i, s := range g.Seats {
		seatsJSON[i] = statusSeatJSON{
			SeatID: s.SeatID, Role: s.Role, Driver: s.Driver, Model: s.Model,
			Worktree: s.Worktree, PaneID: s.PaneID, AgmsgTeam: s.AgmsgTeam, Event: s.Event,
			Active: s.Active, DryRun: s.DryRun, Details: s.Details, TS: s.TS,
		}
	}
	result := statusOrgJSON{OrgID: g.OrgID, Seats: seatsJSON, ActiveCount: realCount.Active, TotalCount: realCount.Total}
	if hb, ok := readOrgWatchHeartbeat(stateDir, g.OrgID); ok {
		result.Watch = &statusWatchJSON{
			LastCycleTS:   hb.LastCycleTS,
			Cycles:        hb.Cycles,
			PendingAlerts: len(hb.PendingAlerts),
		}
	}
	return result
}

func printStatusTableAllOrgs(cmd *cobra.Command, stateDir string, groups []orgGroup, realCounts map[string]realSeatCount, corruptLines int) {
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
		// active/total here are real-seat-only (realCounts, from
		// buildRealSeatCounts) -- not a count over g.Seats, which can
		// include dry-run rows. See statusOrgJSON's doc comment for the
		// same convention on the --json path.
		rc := realCounts[g.OrgID]
		_, _ = fmt.Fprintf(out, "org_id: %s (active %d/%d)\n", g.OrgID, rc.Active, rc.Total)
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
