package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/scaffold"
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
			"watch heartbeat and pending alert count.\n\n" +
			"Also displays a scaffold-ownership section (FR-12): each `.ralph/manifest.toml`-\n" +
			"tracked path's owner attribute (core/fork/seed/block) and any unresolved drift.\n" +
			"This section always reads the current working directory's project (the same\n" +
			"resolution `ralph doctor` uses) regardless of --state-dir/--org-id, which only\n" +
			"scope the org-runtime portion above. It is omitted entirely when the current\n" +
			"directory has no `.ralph/manifest.toml` (not a ralph project).",
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedStateDir, stateDirSource := org.ResolveOrgStateDir(stateDir, cmd.Flags().Changed("state-dir"))
			return runStatus(cmd, resolvedStateDir, stateDirSource, orgID, jsonOut)
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

func runStatus(cmd *cobra.Command, stateDir, stateDirSource, filterOrgID string, jsonOut bool) error {
	store := org.NewManifestStoreAtPath(org.ManifestPathIn(stateDir))
	rr, err := store.Read()
	if err != nil {
		return fmt.Errorf("status: read manifest: %w", err)
	}

	// Scaffold ownership (FR-12) always reads the current working
	// directory's project, independent of --state-dir/--org-id -- those
	// flags scope org-runtime state only (see the command's Long
	// description). "." mirrors how `ralph doctor` resolves its own
	// scaffold-integrity checks (checkScaffoldIntegrity, doctor.go).
	//
	// A scaffold-ownership computation error (e.g. a template load failure,
	// or a disk read PlanCoreReplaceDesired performs) degrades to an
	// "unavailable" scaffold section rather than aborting the whole
	// command: before FR-12, `ralph status` was a read-only org report with
	// no scaffold failure mode at all, and buildScaffoldStatus already
	// degrades gracefully for the "no manifest" case -- a single unreadable
	// tracked file taking the entire org roster down with it would be a new
	// asymmetry within this same function.
	scaffoldStat, scaffoldErr := buildScaffoldStatus(".")
	if scaffoldErr != nil {
		scaffoldStat = &scaffoldStatus{Err: scaffoldErr.Error()}
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
		return printStatusEmpty(cmd, stateDir, stateDirSource, filterOrgID, jsonOut, rr.CorruptLines, scaffoldStat)
	}

	if jsonOut {
		return printStatusJSONAllOrgs(cmd, stateDir, stateDirSource, filterOrgID, groups, realCounts, rr.CorruptLines, scaffoldStat)
	}
	printStatusTableAllOrgs(cmd, stateDir, stateDirSource, groups, realCounts, rr.CorruptLines, scaffoldStat)
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

// printStateDirLine prints the resolved org state directory alongside the
// org.ResolveOrgStateDir precedence tier that produced it (flag/env/
// git-toplevel/cwd) -- tech-debt "watchdog deferred LOW (1)": the source was
// being resolved everywhere but discarded (`_`) at every production call
// site, so an operator debugging "why did `ralph status` read from an
// unexpected directory" had no way to see which tier won without re-deriving
// ResolveOrgStateDir's precedence by hand.
func printStateDirLine(out io.Writer, stateDir, stateDirSource string) {
	_, _ = fmt.Fprintf(out, "state-dir: %s (source: %s)\n", stateDir, stateDirSource)
}

func printStatusEmpty(cmd *cobra.Command, stateDir, stateDirSource, filterOrgID string, jsonOut bool, corruptLines int, scaffoldStat *scaffoldStatus) error {
	out := cmd.OutOrStdout()
	if jsonOut {
		payload := statusJSON{StateDir: stateDir, StateDirSource: stateDirSource, OrgID: filterOrgID, Orgs: []statusOrgJSON{}, CorruptLines: corruptLines, Scaffold: buildStatusScaffoldJSON(scaffoldStat)}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}
	// Scaffold ownership renders even when org state is absent -- an
	// org-less directory can still be a valid ralph project (AC-10's "org
	// state absent but scaffold present" matrix cell).
	printScaffoldSection(out, scaffoldStat)
	// An empty roster does not mean "nothing to report": a manifest whose
	// every line failed to parse also yields zero groups, and that is a
	// data-integrity signal the operator needs, not silence (see
	// printStatusTableAllOrgs's identical warning for the non-empty path,
	// and `ralph org status`'s corrupt-count warning in org.go, which is
	// likewise unconditional on seat count).
	if filterOrgID != "" {
		_, _ = fmt.Fprintf(out, "no org runtime state found (org-id=%s, state-dir=%s (source: %s)) — run `ralph org spawn` to start one.\n", filterOrgID, stateDir, stateDirSource)
	} else {
		_, _ = fmt.Fprintf(out, "no org runtime state found (state-dir=%s (source: %s)) — run `ralph org spawn` to start one.\n", stateDir, stateDirSource)
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
	StateDir       string          `json:"state_dir"`
	StateDirSource string          `json:"state_dir_source,omitempty"`
	OrgID          string          `json:"org_id,omitempty"`
	Orgs           []statusOrgJSON `json:"orgs"`
	CorruptLines   int             `json:"corrupt_lines,omitempty"`
	// Scaffold is the FR-12 scaffold-ownership section (see
	// buildStatusScaffoldJSON). Additive to the pre-FR-12 schema above: nil
	// (omitted) when the current working directory has no
	// `.ralph/manifest.toml` at all -- every existing key's shape and
	// meaning is unchanged.
	Scaffold *statusScaffoldJSON `json:"scaffold,omitempty"`
}

// statusScaffoldFileJSON is one manifest-tracked path's ownership summary,
// mirroring scaffoldOwnershipFile's fields for the --json wire shape.
type statusScaffoldFileJSON struct {
	Path              string `json:"path"`
	Owner             string `json:"owner"`
	ForkedFromVersion string `json:"forked_from_version,omitempty"`
	Drift             bool   `json:"drift,omitempty"`
}

// statusScaffoldJSON is the --json wire shape of the FR-12 scaffold-
// ownership section. Layout is "v2" or "legacy"; Files/Drift are only
// populated for "v2" (a legacy manifest carries just {"layout":"legacy"} --
// see buildStatusScaffoldJSON). Error is additive: set only when
// buildScaffoldStatus's computation itself failed (M7), in which case
// Layout/Files/Drift are all left zero -- a machine consumer should check
// Error first before assuming a v2/legacy shape.
type statusScaffoldJSON struct {
	Error  string                   `json:"error,omitempty"`
	Layout string                   `json:"layout,omitempty"`
	Files  []statusScaffoldFileJSON `json:"files,omitempty"`
	Drift  []string                 `json:"drift,omitempty"`
}

// buildStatusScaffoldJSON renders s into its --json wire shape. Returns nil
// when s is nil (no manifest at all), which the statusJSON.Scaffold
// `omitempty` tag then drops from the encoded payload entirely -- the
// chosen representation for "not a ralph project" (see status.go's package
// doc / newStatusCmd's Long description). Distinct from s.Err != "" (a
// manifest exists but the ownership computation failed), which renders
// {"error": "..."} instead of omitting the key -- the caller still gets a
// "scaffold" key, just one it must check Error on before reading the rest.
func buildStatusScaffoldJSON(s *scaffoldStatus) *statusScaffoldJSON {
	if s == nil {
		return nil
	}
	if s.Err != "" {
		return &statusScaffoldJSON{Error: s.Err}
	}
	if s.Layout != scaffold.LayoutV2 {
		return &statusScaffoldJSON{Layout: s.Layout}
	}
	files := make([]statusScaffoldFileJSON, len(s.Files))
	for i, f := range s.Files {
		files[i] = statusScaffoldFileJSON(f)
	}
	return &statusScaffoldJSON{Layout: s.Layout, Files: files, Drift: append([]string(nil), s.Drift...)}
}

func printStatusJSONAllOrgs(cmd *cobra.Command, stateDir, stateDirSource, filterOrgID string, groups []orgGroup, realCounts map[string]realSeatCount, corruptLines int, scaffoldStat *scaffoldStatus) error {
	orgsJSON := make([]statusOrgJSON, 0, len(groups))
	for _, g := range groups {
		orgsJSON = append(orgsJSON, buildStatusOrgJSON(stateDir, g, realCounts[g.OrgID]))
	}
	payload := statusJSON{StateDir: stateDir, StateDirSource: stateDirSource, OrgID: filterOrgID, Orgs: orgsJSON, CorruptLines: corruptLines, Scaffold: buildStatusScaffoldJSON(scaffoldStat)}
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

func printStatusTableAllOrgs(cmd *cobra.Command, stateDir, stateDirSource string, groups []orgGroup, realCounts map[string]realSeatCount, corruptLines int, scaffoldStat *scaffoldStatus) {
	out := cmd.OutOrStdout()

	// Surfaced once at the top of the populated-roster table (tech-debt:
	// "watchdog deferred LOW (1)") -- the empty-roster path
	// (printStatusEmpty) folds the same "(source: ...)" annotation into its
	// existing state-dir=... message instead of a separate line, since that
	// path has no table to put a header line above.
	printStateDirLine(out, stateDir, stateDirSource)
	printScaffoldSection(out, scaffoldStat)

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

// --- FR-12 scaffold ownership (docs/specs/2026-08-17-overlay-scaffold-v2.md,
// FR-12; Phase 5 plan Scope item 4) ---

// scaffoldOwnershipFile is one manifest-tracked path's ownership summary for
// the scaffold-ownership section below.
type scaffoldOwnershipFile struct {
	Path              string
	Owner             string
	ForkedFromVersion string
	Drift             bool
}

// scaffoldStatus is the FR-12 "scaffold ownership" section: which owner
// attribute (core/fork/seed/block) each v2-manifest-tracked path carries,
// plus which paths are in unresolved drift. Layout is scaffold.LayoutV2
// ("v2") or "legacy"; Files/Drift are only populated for "v2" (see
// buildScaffoldStatus).
//
// Err is set (Layout/Files/Drift left zero) when buildScaffoldStatus's
// computation itself failed -- a template load error, or a disk read
// PlanCoreReplaceDesired performs -- as opposed to the "no manifest at
// all" case, which callers represent as a nil *scaffoldStatus instead (see
// buildScaffoldStatus's doc). Both printScaffoldSection and
// buildStatusScaffoldJSON check Err first and render an "unavailable"
// section/JSON shape rather than erroring `ralph status` out entirely
// (M7: a scaffold-ownership computation failure must not take the org
// roster section down with it).
type scaffoldStatus struct {
	Err    string
	Layout string
	Files  []scaffoldOwnershipFile
	Drift  []string
}

// buildScaffoldStatus reads the ownership manifest at targetDir and
// classifies every tracked path's ownership plus unresolved drift, reusing
// the exact same read-only classification eject/adopt/doctor share
// (resolveOwnershipPlan, adopt.go) so `ralph status`'s notion of
// "fork"/"drift" can never disagree with those commands.
//
// Returns (nil, nil) when targetDir has no `.ralph/manifest.toml` at all --
// this is deliberately not an error: `ralph status` must keep reporting
// org-runtime state in a directory that is not (or not yet) a ralph
// project. A legacy (pre-v2) manifest also short-circuits to a
// Layout-only result, mirroring checkScaffoldIntegrity's identical
// distinction (doctor.go) between "not a ralph project" and "not upgraded
// yet".
func buildScaffoldStatus(targetDir string) (*scaffoldStatus, error) {
	manifestPath := filepath.Join(targetDir, ".ralph", "manifest.toml")
	manifest, err := scaffold.ReadManifest(manifestPath)
	if err != nil {
		return nil, nil
	}
	if manifest.Meta.Layout != scaffold.LayoutV2 {
		return &scaffoldStatus{Layout: "legacy"}, nil
	}

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("resolving target dir: %w", err)
	}
	_, plan, err := resolveOwnershipPlan(absDir, manifest, io.Discard)
	if err != nil {
		return nil, fmt.Errorf("computing scaffold ownership plan: %w", err)
	}

	driftSet := driftPathSet(plan)

	paths := make([]string, 0, len(manifest.Files))
	for p := range manifest.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	files := make([]scaffoldOwnershipFile, 0, len(paths))
	for _, p := range paths {
		entry := manifest.Files[p]
		files = append(files, scaffoldOwnershipFile{
			Path:              p,
			Owner:             entry.Owner,
			ForkedFromVersion: entry.ForkedFromVersion,
			Drift:             driftSet[p],
		})
	}

	// plan.Drift can include genuinely untracked paths too (classifyUntracked,
	// internal/upgrade/replaceplan.go) -- listed here in full, matching
	// doctor's checkScaffoldCoreHashes (FR-9(a)), so status's drift list is
	// never a strict subset of what `ralph doctor --strict` would flag.
	driftPaths := make([]string, 0, len(plan.Drift))
	for _, d := range plan.Drift {
		driftPaths = append(driftPaths, d.Path)
	}
	sort.Strings(driftPaths)

	return &scaffoldStatus{Layout: scaffold.LayoutV2, Files: files, Drift: driftPaths}, nil
}

// printScaffoldSection renders the FR-12 scaffold-ownership section. s==nil
// (no manifest at all -- not a ralph project) prints nothing, so a
// non-scaffold directory's `ralph status` text output is byte-identical to
// pre-FR-12 behavior. When it does render something, it always ends with a
// blank separator line so callers never need to track whether this section
// produced output before printing whatever comes next.
//
// Layout: an owner-grouped count line first (core/fork/seed/block
// totals), then only non-core rows individually (core paths carry no
// per-file signal worth a line unless they are drifted, which the
// "Unresolved drift" list below already covers), then the drift list.
func printScaffoldSection(out io.Writer, s *scaffoldStatus) {
	if s == nil {
		return
	}
	if s.Err != "" {
		_, _ = fmt.Fprintf(out, "Scaffold ownership: unavailable (%s)\n", s.Err)
		_, _ = fmt.Fprintln(out)
		return
	}
	if s.Layout != scaffold.LayoutV2 {
		_, _ = fmt.Fprintln(out, "Scaffold ownership: legacy manifest layout -- run `ralph upgrade` to migrate before ownership/drift details are available.")
		_, _ = fmt.Fprintln(out)
		return
	}

	counts := make(map[string]int, 4)
	for _, f := range s.Files {
		counts[f.Owner]++
	}
	_, _ = fmt.Fprintf(out, "Scaffold ownership: %d path(s) tracked (core: %d, fork: %d, seed: %d, block: %d)\n",
		len(s.Files), counts[scaffold.OwnerCore], counts[scaffold.OwnerFork], counts[scaffold.OwnerSeed], counts[scaffold.OwnerBlock])
	for _, f := range s.Files {
		if f.Owner == scaffold.OwnerCore {
			continue
		}
		if f.Owner == scaffold.OwnerFork && f.ForkedFromVersion != "" {
			_, _ = fmt.Fprintf(out, "  %s: %s (forked_from_version=%s)\n", f.Path, f.Owner, f.ForkedFromVersion)
		} else {
			_, _ = fmt.Fprintf(out, "  %s: %s\n", f.Path, f.Owner)
		}
	}

	if len(s.Drift) == 0 {
		_, _ = fmt.Fprintln(out, "Unresolved drift: none")
	} else {
		_, _ = fmt.Fprintf(out, "Unresolved drift: %d path(s)\n", len(s.Drift))
		for _, p := range s.Drift {
			_, _ = fmt.Fprintf(out, "  %s\n", p)
		}
	}
	_, _ = fmt.Fprintln(out)
}
