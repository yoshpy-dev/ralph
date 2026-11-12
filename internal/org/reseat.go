package org

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// `ralph org reseat` (spec docs/specs/2026-11-12-org-reseat.md, plan
// docs/plans/active/2026-11-12-org-reseat.md) stops the seats of an org
// started with `ralph org start --plan` and starts them again in the current
// environment, without a disband, so the org's current life keeps its plan
// approvals, its reservation and its binding. PlanReseat is its first half:
// it decides what the reseat would do and refuses, before anything is stopped
// or written, every case the later stops and starts would refuse (FR-4). It
// writes nothing to the manifest, stops nothing and starts nothing; with
// DryRun it does not start the cross-org watch or the herdr server either
// (P8).

// reseatIDTimeLayout is the time part of a reseat id (P2): the UTC time,
// YYYYMMDDThhmmss.
const reseatIDTimeLayout = "20060102T150405"

// reseatIDRand is where the 4 hex digits of a reseat id come from:
// crypto/rand in production. A test sets a fixed reader.
var reseatIDRand io.Reader = rand.Reader

// ReseatParams describes one `ralph org reseat` invocation.
type ReseatParams struct {
	OrgID string
	// Seats are the --seat values; empty means every active seat of the
	// org's current life (D2). A seat named twice is refused.
	Seats []string
	// Driver and Model are --driver and --model: accepted only together and
	// only with exactly one seat in Seats (D1, FR-1); "" is not given.
	Driver, Model string
	StateDir      string // resolved org state dir; split plans are in SplitPlansDirIn(StateDir)
	RepoRoot      string // root of the main worktree, as StartFeatureParams.RepoRoot
	// Reviewer goes to the leader's start as StartFeatureParams.Reviewer.
	Reviewer config.ReviewerConfig
	// TimeoutMS goes to each start as its TimeoutMS.
	TimeoutMS int
	DryRun    bool
	// WatchProbe and HerdrServerProbe are what a dry run reads in place of
	// Org.WatchGate and Org.HerdrServerGate, which a real reseat runs and
	// which may start the cross-org watch or `herdr server` (P8). A dry run
	// reads each only when the Org has that gate set. A nil WatchProbe is
	// ProbeWatch on the ledger at the Org's clock. HerdrServerProbe reports
	// whether the herdr server runs; its error is the refusal of a gate that
	// can neither find nor start a server, and a gate that passes without a
	// server it can check reports running. A nil HerdrServerProbe reads
	// nothing, so the dry run says nothing about the server.
	WatchProbe       func() WatchLiveness
	HerdrServerProbe func() (running bool, err error)
}

// ReseatSeat is one seat a reseat stops or starts.
type ReseatSeat struct {
	SeatID string
	Role   string
	// Driver and Model are what the seat starts with: those of its latest
	// spawn_started or spawned in the org's current life (a Model "" ran the
	// driver CLI's default and stays ""), or --driver and --model (P5).
	Driver, Model string
	// RecordedDriver and RecordedModel are those of the ledger; Overridden
	// says Driver and Model came from --driver and --model instead.
	RecordedDriver, RecordedModel string
	Overridden                    bool
	// Leader says the seat is the org's leader, which starts again through
	// StartFeature with Start; any other seat starts through Spawn with
	// Spawn.
	Leader bool
	Start  *StartFeatureParams
	Spawn  *SpawnParams
	// Active says the seat is active now (its latest state event is
	// spawn_started, spawn_step or spawned), so the reseat stops it before it
	// starts it. Latest is that latest state event of the seat in the org's
	// current life and LatestIndex its index in the manifest's events
	// PlanReseat read: the reseat reads the ledger again right before it
	// stops the seat, and goes on only while that is still the seat's latest
	// state event (P6).
	Active      bool
	Latest      ManifestEvent
	LatestIndex int
}

// ReseatPlan is what a reseat of an org would do, as PlanReseat checked it.
type ReseatPlan struct {
	// ID names the reseat in the records it writes (P2):
	// <UTC YYYYMMDDThhmmss>-<4 lowercase hex digits>.
	ID    string
	OrgID string
	// Binding is the org's split plan binding in its current life.
	Binding *FeatureBinding
	// Stop holds the target seats that are active now, in the order the
	// reseat stops them: every seat but the leader in the reverse of Start's
	// order, then the leader (D3). Start holds every target seat in the order
	// the reseat starts them: the leader first, then the others in the order
	// of their latest spawn_started in the org's current life.
	Stop  []ReseatSeat
	Start []ReseatSeat
	// DryRun says the plan is a dry run's. StartsWatch and StartsHerdrServer
	// are set only by a dry run: no cross-org watch runs on the ledger, or
	// the herdr server is not running, and a real reseat starts it before it
	// stops any seat (P8).
	DryRun            bool
	StartsWatch       bool
	StartsHerdrServer bool
}

// reseatRefusal is a refusal of PlanReseat: err is the cause, and seat the
// seat whose start was checked when it was found ("" for the org). Its
// message says that nothing was stopped and keeps err's own words, without
// their "org: " prefix; errors.Is and errors.As see err.
type reseatRefusal struct {
	orgID string
	seat  string
	err   error
}

func (e *reseatRefusal) Error() string {
	cause := strings.TrimPrefix(e.err.Error(), "org: ")
	if e.seat != "" {
		return fmt.Sprintf("org: reseat of org_id %q is refused before anything is stopped: seat %q would not start again: %s", e.orgID, e.seat, cause)
	}
	return fmt.Sprintf("org: reseat of org_id %q is refused before anything is stopped: %s", e.orgID, cause)
}

func (e *reseatRefusal) Unwrap() error { return e.err }

// reseatLifeSeat is what the org's current life (its real events after its
// latest real `disbanded`, all of them when it has none) holds about one
// seat: its latest state event and that event's index, whether one of its
// events is `spawned`, its latest spawn_started or spawned (ran, which P5
// reads the role, driver and model from), and the indexes of its latest
// spawn_started and spawned, which its start order is taken from (order).
type reseatLifeSeat struct {
	id        string
	latest    ManifestEvent
	latestIdx int
	spawned   bool
	ran       ManifestEvent
	hasRan    bool
	started   int // index of the latest spawn_started; -1 for none
	spawnedAt int // index of the latest spawned; -1 for none
}

// active says the seat's latest state event is an active one.
func (s *reseatLifeSeat) active() bool {
	return isActiveEvent(s.latest.Event)
}

// order is the seat's place in the start order: the index of its latest
// spawn_started, else of its latest spawned, else of its latest state event.
func (s *reseatLifeSeat) order() int {
	switch {
	case s.started >= 0:
		return s.started
	case s.spawnedAt >= 0:
		return s.spawnedAt
	}
	return s.latestIdx
}

// recorded is the role, driver and model of the seat's latest spawn_started
// or spawned, or of its latest state event when it has neither.
func (s *reseatLifeSeat) recorded() ManifestEvent {
	if s.hasRan {
		return s.ran
	}
	return s.latest
}

// reseatLifeSeats returns, in start order (reseatLifeSeat.order), every seat
// of orgID that has a state event in the org's current life.
func reseatLifeSeats(events []ManifestEvent, orgID string) []*reseatLifeSeat {
	from := 0
	if d, ok := lastDisbandedIndexes(events)[orgID]; ok {
		from = d + 1
	}
	byID := make(map[string]*reseatLifeSeat)
	var seats []*reseatLifeSeat
	for i := from; i < len(events); i++ {
		ev := events[i]
		if ev.DryRun || ev.OrgID != orgID || ev.SeatID == "" || !isStateEvent(ev.Event) {
			continue
		}
		s, ok := byID[ev.SeatID]
		if !ok {
			s = &reseatLifeSeat{id: ev.SeatID, started: -1, spawnedAt: -1}
			byID[ev.SeatID] = s
			seats = append(seats, s)
		}
		s.latest, s.latestIdx = ev, i
		switch ev.Event {
		case EventSpawnStarted:
			s.ran, s.hasRan, s.started = ev, true, i
		case EventSpawned:
			s.ran, s.hasRan, s.spawnedAt, s.spawned = ev, true, i, true
		}
	}
	slices.SortStableFunc(seats, func(a, b *reseatLifeSeat) int { return a.order() - b.order() })
	return seats
}

// PlanReseat decides what `ralph org reseat` of p.OrgID would do and checks,
// before anything is stopped, that its stops and starts would not be refused
// (FR-4). It writes nothing to the manifest and calls neither herdr nor
// agmsg. It refuses, in this order:
//
//  1. --driver or --model without exactly one --seat, one of them without
//     the other, and a seat named twice (FR-1, AC-11);
//  2. a run from the herdr pane (HERDR_PANE_ID) of an active seat of the
//     org, which the reseat would close (FR-3, AC-7);
//  3. an org with no complete split plan binding in its current life, which
//     is to be stopped and spawned seat by seat instead (FR-2, AC-5);
//  4. targets (D2, P7): with no --seat, the org's active seats, and none is
//     refused with its stopped seats and the --seat to name them (AC-6); a
//     --seat is a seat spawned in the org's current life (whatever its state
//     is now: active, stopped, spawn_failed or rejected) or active now;
//  5. the leader's start as StartFeature would check it
//     (startFeatureInput: the split plan <state dir>/splits/<split>.md is
//     there, approved and has the feature; the org_id and the leader's
//     spawn input; the reviewer values);
//  6. on a copy of the manifest's events with a `stopped` for each target
//     seat that is active, as Stop records it (P1): when the leader is a
//     target, the rest of StartFeature's checks (checkLedger:
//     spawnPrecheckErr, regateSpawnErr, checkFeatureWorktreeReuse), after
//     which a `spawned` of the leader is added to the copy; otherwise the
//     binding the split plan gives now against the org's
//     (reservationDecision: the same feature, feature digest and approval
//     mode), regateSpawnErr and checkFeatureWorktreeReuse;
//  7. a feature worktree with no ralph-worktree.sh record: reseat never makes
//     a worktree (FR-4, stricter than start);
//  8. for every other target in start order, checkSpawnInput and
//     spawnPrecheckErr of its spawn on the copy, after which a `spawned` of
//     it is added to the copy, so max_seats counts as it will when the seats
//     start, and spawnFeatureSeatErr finds the leader active (and, for an
//     implementer of a per-plan org, the approval of the current life);
//  9. the watch gate and then the herdr server gate, which may start the
//     cross-org watch and `herdr server`; a dry run reads them instead
//     (reseatDryRunGates).
//
// Every refusal is a *reseatRefusal that names the org, says that nothing
// was stopped, and keeps the words of the check that refused it, which say
// what to do. A manifest that cannot be read is an error of its own.
func (o *Org) PlanReseat(p ReseatParams) (ReseatPlan, error) {
	orgID := p.OrgID
	if err := ValidateIdentifier("org_id", orgID); err != nil {
		return ReseatPlan{}, err
	}
	refuse := func(seat string, err error) (ReseatPlan, error) {
		return ReseatPlan{}, &reseatRefusal{orgID: orgID, seat: seat, err: err}
	}
	if err := checkReseatFlags(p); err != nil {
		return refuse("", err)
	}
	rr, err := o.Manifest.Read()
	if err != nil {
		return ReseatPlan{}, fmt.Errorf("org: reseat: read manifest: %w", err)
	}
	events := rr.Events
	life := reseatLifeSeats(events, orgID)
	if err := o.reseatSelfPaneErr(orgID, life); err != nil {
		return refuse("", err)
	}
	binding := ActiveFeature(events, orgID)
	if binding == nil || !binding.Complete() {
		return refuse("", o.reseatNoBindingErr(orgID, binding))
	}
	targets, err := o.reseatTargets(p, binding, life)
	if err != nil {
		return refuse("", err)
	}

	plan := ReseatPlan{OrgID: orgID, Binding: binding, DryRun: p.DryRun}
	// The leader's driver and model: its target's when it is one, else
	// those the ledger recorded. Every start names the driver as the
	// leader identity's (SpawnParams.LeaderDriver).
	var leaderDriver, leaderModel string
	for _, s := range life {
		if s.id == LeaderIdentity {
			rec := s.recorded()
			leaderDriver, leaderModel = rec.Driver, rec.Model
		}
	}
	leaderTarget := false
	for _, s := range targets {
		seat := reseatSeat(s, p)
		if seat.Leader {
			leaderTarget, leaderDriver, leaderModel = true, seat.Driver, seat.Model
		}
		plan.Start = append(plan.Start, seat)
	}
	// stops are the indexes in plan.Start of the seats to stop, in stop
	// order (D3); plan.Stop is filled from them once plan.Start is complete.
	var stops []int
	for i := len(plan.Start) - 1; i >= 0; i-- {
		if plan.Start[i].Active && !plan.Start[i].Leader {
			stops = append(stops, i)
		}
	}
	if leaderTarget && plan.Start[0].Active {
		stops = append(stops, 0)
	}

	leaderStart := StartFeatureParams{
		StateDir: p.StateDir, RepoRoot: p.RepoRoot,
		SplitPath: filepath.Join(SplitPlansDirIn(p.StateDir), binding.Split+".md"), Feature: binding.Feature, OrgID: orgID,
		Driver: leaderDriver, Model: leaderModel, LeaderDriver: leaderDriver, TimeoutMS: p.TimeoutMS, Reviewer: p.Reviewer,
	}
	leaderSeat := ""
	if leaderTarget {
		leaderSeat = LeaderIdentity
		plan.Start[0].Start = &leaderStart
	}
	checked, err := startFeatureInput(leaderStart, featureApprovalMode(o.Config))
	if err != nil {
		return refuse(leaderSeat, err)
	}

	ledgerDir := filepath.Dir(o.Manifest.Path())
	worktrees := o.worktrees()
	sim := slices.Clone(events)
	for _, i := range stops {
		sim = append(sim, reseatStoppedEvent(plan.Start[i].Latest))
	}
	if leaderTarget {
		if err := checked.checkLedger(o.Config, sim, o.InboxHintStateDir, ledgerDir, worktrees); err != nil {
			return refuse(leaderSeat, err)
		}
		sim = append(sim, reseatSpawnedEvent(checked.checked))
	} else {
		if _, err := reservationDecision(sim, orgID, checked.checked.reservation()); err != nil {
			return refuse("", err)
		}
		if err := regateSpawnErr(ledgerDir, orgID); err != nil {
			return refuse("", err)
		}
		if err := checkFeatureWorktreeReuse(worktrees, checked.root, orgID, checked.feature.Slug, checked.want); err != nil {
			return refuse("", err)
		}
	}
	if err := reseatWorktreeRecordErr(worktrees, checked, binding); err != nil {
		return refuse("", err)
	}

	scope := strings.Join(ActiveReservation(events, orgID), ", ")
	for i := range plan.Start {
		seat := &plan.Start[i]
		if seat.Leader {
			continue
		}
		sp := SpawnParams{
			OrgID: orgID, SeatID: seat.SeatID, Role: seat.Role, Driver: seat.Driver, Model: seat.Model,
			Cwd: binding.Worktree, Scope: scope, TimeoutMS: p.TimeoutMS, LeaderDriver: leaderDriver,
		}
		in, err := checkSpawnInput(sp)
		if err != nil {
			return refuse(seat.SeatID, err)
		}
		if err := spawnPrecheckErr(o.Config, in, sim, o.InboxHintStateDir); err != nil {
			return refuse(seat.SeatID, err)
		}
		sim = append(sim, reseatSpawnedEvent(in))
		seat.Spawn = &sp
	}
	for _, i := range stops {
		plan.Stop = append(plan.Stop, plan.Start[i])
	}

	if plan.ID, err = newReseatID(o.nowTime()); err != nil {
		return ReseatPlan{}, err
	}
	if p.DryRun {
		if err := o.reseatDryRunGates(p, ledgerDir, &plan); err != nil {
			return refuse("", err)
		}
		return plan, nil
	}
	if err := o.checkWatchGate(); err != nil {
		return refuse("", err)
	}
	if err := o.checkHerdrServerGate(); err != nil {
		return refuse("", err)
	}
	return plan, nil
}

// checkReseatFlags checks p's --seat, --driver and --model (FR-1, D1): each
// seat is a valid seat_id named once, and --driver and --model come together
// and only with exactly one --seat.
func checkReseatFlags(p ReseatParams) error {
	for i, seat := range p.Seats {
		if err := ValidateIdentifier("seat_id", seat); err != nil {
			return err
		}
		if slices.Contains(p.Seats[:i], seat) {
			return fmt.Errorf("--seat %s is given twice: name each seat once", seat)
		}
	}
	driver, model := strings.TrimSpace(p.Driver), strings.TrimSpace(p.Model)
	switch {
	case driver == "" && model == "":
		return nil
	case len(p.Seats) != 1:
		return fmt.Errorf("--driver and --model start one seat on another driver or model, so they need exactly one --seat "+
			"(%d given): run ralph org reseat once for each seat to change, and without them for seats that keep "+
			"the driver and model the ledger recorded", len(p.Seats))
	case driver == "" || model == "":
		return fmt.Errorf("--driver and --model go together: pass both with --seat %s", p.Seats[0])
	}
	return nil
}

// reseatSelfPaneErr refuses a reseat run from the herdr pane of an active
// seat of the org (FR-3): stopping that seat would close the pane the reseat
// runs in, before it starts anything.
func (o *Org) reseatSelfPaneErr(orgID string, life []*reseatLifeSeat) error {
	pane := o.getenv(herdrPaneIDEnv)
	if pane == "" {
		return nil
	}
	for _, s := range life {
		if s.active() && s.latest.PaneID == pane {
			return fmt.Errorf("this command runs in the herdr pane %s of seat %q of org_id %q, which the reseat would close "+
				"before it starts the seats again: run ralph org reseat from outside the org's seats, in the director's "+
				"session or a shell of your own", pane, s.id, orgID)
		}
	}
	return nil
}

// reseatNoBindingErr is the refusal of an org whose current life has no
// complete split plan binding (binding nil or incomplete; FR-2, D4): its
// leader's task cannot be made again from the ledger, so its seats are
// stopped and started one by one.
func (o *Org) reseatNoBindingErr(orgID string, binding *FeatureBinding) error {
	why := "has no split plan binding since its last disband (it was not started with ralph org start --plan)"
	if binding != nil {
		why = "has an incomplete split plan binding since its last disband (" + binding.String() + ")"
	}
	id := ShellWord(orgID)
	return fmt.Errorf("org_id %q %s, so ralph org reseat does not start its seats again: stop a seat with %s and start it "+
		"again with %s (its leader with ralph org start)", orgID, why,
		VerbCommand("stop --org-id "+id+" --seat <seat>", o.InboxHintStateDir),
		VerbCommand("spawn --org-id "+id+" --id <seat> --role <role>", o.InboxHintStateDir))
}

// reseatTargets returns the seats of the org's current life a reseat with p
// starts again, in start order: with no --seat, the active ones, and
// otherwise the named ones, each spawned in the current life or active now
// (D2, P7). The leader is first.
func (o *Org) reseatTargets(p ReseatParams, binding *FeatureBinding, life []*reseatLifeSeat) ([]*reseatLifeSeat, error) {
	life = reseatLeaderFirst(life)
	if len(p.Seats) == 0 {
		var targets []*reseatLifeSeat
		for _, s := range life {
			if s.active() {
				targets = append(targets, s)
			}
		}
		if len(targets) == 0 {
			return nil, o.reseatNoActiveSeatErr(p.OrgID, p.StateDir, binding, life)
		}
		return targets, nil
	}
	accepted := func(s *reseatLifeSeat) bool { return s.spawned || s.active() }
	for _, name := range p.Seats {
		if slices.ContainsFunc(life, func(s *reseatLifeSeat) bool { return s.id == name && accepted(s) }) {
			continue
		}
		var can []string
		for _, s := range life {
			if accepted(s) {
				can = append(can, s.id)
			}
		}
		named := "none"
		if len(can) > 0 {
			named = strings.Join(can, ", ")
		}
		return nil, fmt.Errorf("seat %q of org_id %q has not been spawned since the org's last disband, so reseat does not "+
			"start it (the seats it can start again are: %s): a seat the org never ran is spawned with ralph org spawn",
			name, p.OrgID, named)
	}
	var targets []*reseatLifeSeat
	for _, s := range life {
		if slices.Contains(p.Seats, s.id) {
			targets = append(targets, s)
		}
	}
	return targets, nil
}

// reseatNoActiveSeatErr is the refusal of a reseat with no --seat of an org
// that has no active seat (D2, AC-6): it names the org's stopped seats and
// the reseat that names them with --seat; with none stopped, the seats that
// --seat accepts; with none spawned at all, the start of its leader from the
// split plan in stateDir.
func (o *Org) reseatNoActiveSeatErr(orgID, stateDir string, binding *FeatureBinding, life []*reseatLifeSeat) error {
	var stopped, spawned []string
	for _, s := range life {
		switch {
		case s.latest.Event == EventStopped:
			stopped = append(stopped, s.id)
		case s.spawned:
			spawned = append(spawned, s.id+" ("+s.latest.Event+")")
		}
	}
	id := ShellWord(orgID)
	lead := fmt.Sprintf("org_id %q has no active seat since its last disband, so a reseat with no --seat has no seat to start again", orgID)
	switch {
	case len(stopped) > 0:
		cmd := "reseat --org-id " + id
		for _, s := range stopped {
			cmd += " --seat " + ShellWord(s)
		}
		return fmt.Errorf("%s; its stopped seats are %s: to start them again, run %s", lead, strings.Join(stopped, ", "),
			VerbCommand(cmd, o.InboxHintStateDir))
	case len(spawned) > 0:
		return fmt.Errorf("%s; none of its seats is stopped, and the seats spawned since then are %s: name the ones to start "+
			"again with --seat (%s)", lead, strings.Join(spawned, ", "),
			VerbCommand("reseat --org-id "+id+" --seat <seat>", o.InboxHintStateDir))
	}
	return fmt.Errorf("%s; no seat has been spawned since then: start its leader with %s", lead,
		VerbCommand("start --plan "+ShellWord(filepath.Join(SplitPlansDirIn(stateDir), binding.Split+".md"))+
			" --feature "+ShellWord(binding.Feature)+" --org-id "+id, o.InboxHintStateDir))
}

// reseatLeaderFirst is life with the leader moved to the front, the order
// the reseat starts seats in.
func reseatLeaderFirst(life []*reseatLifeSeat) []*reseatLifeSeat {
	out := make([]*reseatLifeSeat, 0, len(life))
	for _, s := range life {
		if s.id == LeaderIdentity {
			out = append(out, s)
		}
	}
	for _, s := range life {
		if s.id != LeaderIdentity {
			out = append(out, s)
		}
	}
	return out
}

// reseatSeat is the ReseatSeat of target s for p: the role, driver and model
// the ledger recorded (reseatLifeSeat.recorded), or p's --driver and --model
// when s is the one seat p names (checkReseatFlags passed them only with
// exactly one --seat, and both).
func reseatSeat(s *reseatLifeSeat, p ReseatParams) ReseatSeat {
	rec := s.recorded()
	seat := ReseatSeat{
		SeatID: s.id, Role: rec.Role, Driver: rec.Driver, Model: rec.Model,
		RecordedDriver: rec.Driver, RecordedModel: rec.Model, Leader: s.id == LeaderIdentity,
		Active: s.active(), Latest: s.latest, LatestIndex: s.latestIdx,
	}
	if driver := strings.TrimSpace(p.Driver); driver != "" && len(p.Seats) == 1 && p.Seats[0] == s.id {
		seat.Driver, seat.Model, seat.Overridden = driver, strings.TrimSpace(p.Model), true
	}
	return seat
}

// reseatWorktreeRecordErr refuses an org whose feature worktree has no
// ralph-worktree.sh record (FR-4): reseat never makes a worktree, and the
// leader's start after the stops would run ensure on a worktree and a branch
// with no record. A record that exists has passed checkFeatureWorktreeReuse
// by now, so it is the one the binding's worktree and branch name.
func reseatWorktreeRecordErr(worktrees FeatureWorktrees, s startFeatureChecked, binding *FeatureBinding) error {
	id := featureWorktreeID(s.orgID)
	_, ok, err := worktrees.Lookup(s.root, id)
	switch {
	case err != nil:
		return err
	case ok:
		return nil
	}
	return fmt.Errorf("the ralph-worktree.sh record %s of org_id %q's feature worktree %s is missing, and reseat makes no "+
		"worktree: find out with git worktree list in the main worktree what became of that worktree and its branch %s",
		id, s.orgID, binding.Worktree, binding.Branch)
}

// reseatStoppedEvent is the `stopped` Stop records for the seat whose latest
// state event is latest (the roster's fields of the seat), with no TS and no
// Details: PlanReseat adds it to its copy of the events only.
func reseatStoppedEvent(latest ManifestEvent) ManifestEvent {
	return ManifestEvent{
		OrgID: latest.OrgID, SeatID: latest.SeatID, Event: EventStopped,
		Role: latest.Role, Driver: latest.Driver, Model: latest.Model, Worktree: latest.Worktree,
		PaneID: latest.PaneID, AgmsgTeam: latest.AgmsgTeam,
	}
}

// reseatSpawnedEvent is the `spawned` of a seat started with p, with no TS:
// PlanReseat adds it to its copy of the events only.
func reseatSpawnedEvent(p SpawnParams) ManifestEvent {
	return ManifestEvent{
		OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawned,
		Role: p.Role, Driver: p.Driver, Model: p.Model, Worktree: p.Cwd,
	}
}

// reseatDryRunGates is what a dry run does in place of the two gates (P8):
// it reads, and starts nothing. With Org.WatchGate set, it probes the watch
// (p.WatchProbe, or ProbeWatch on the ledger in ledgerDir): a watch that
// accepts work passes, none running sets plan.StartsWatch (the gate of a
// real reseat starts one), and any other state (stale, stopping, a --once
// run) is the gate's refusal. With Org.HerdrServerGate and
// p.HerdrServerProbe set, a server that is not running sets
// plan.StartsHerdrServer, and the probe's error is the refusal.
func (o *Org) reseatDryRunGates(p ReseatParams, ledgerDir string, plan *ReseatPlan) error {
	if o.WatchGate != nil {
		probe := p.WatchProbe
		if probe == nil {
			probe = func() WatchLiveness { return ProbeWatch(ledgerDir, o.nowTime(), o.Config.DeadmanMinutes) }
		}
		switch l := probe(); {
		case WatchAcceptsWork(l):
		case l.State == WatchLivenessNotRunning:
			plan.StartsWatch = true
		default:
			return &WatchGateError{StateDir: ledgerDir, HintStateDir: o.InboxHintStateDir, Liveness: l}
		}
	}
	if o.HerdrServerGate != nil && p.HerdrServerProbe != nil {
		running, err := p.HerdrServerProbe()
		if err != nil {
			return err
		}
		plan.StartsHerdrServer = !running
	}
	return nil
}

// newReseatID returns a reseat id for now (P2): the UTC time as
// YYYYMMDDThhmmss, a hyphen, and 2 bytes of reseatIDRand in lowercase hex.
func newReseatID(now time.Time) (string, error) {
	b := make([]byte, 2)
	if _, err := io.ReadFull(reseatIDRand, b); err != nil {
		return "", fmt.Errorf("org: reseat: make an id: %w", err)
	}
	return now.UTC().Format(reseatIDTimeLayout) + "-" + hex.EncodeToString(b), nil
}
