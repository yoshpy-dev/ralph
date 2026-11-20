package org

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
	"github.com/yoshpy-dev/ralph/internal/org/protocol"
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
// (P8). Reseat is the second half: under the ledger's reseat.lock it plans,
// stops the seats, starts them again, and records `reseated` (P4, P6). A
// reseat cut off before it recorded `reseated` (killed, say) leaves the seats
// it stopped with reseat=<id> and no `reseated` of that id. Running the same
// reseat again, with the same --seat values, starts them and finishes it; a
// reseat with no --seat starts them too, besides stopping and starting every
// active seat (P9).

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
	// org's current life (D2) and every seat a reseat that was cut off
	// stopped (P9, reseatLifeSeat.cutOff). A seat named twice is refused.
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
	// nothing, so the dry run says nothing about the server. With
	// RestartHerdr, a dry run and a real reseat both read it before the gates
	// (reseatRestartHerdrErr), and a real reseat reads it again after
	// `herdr server stop` until it reports not running
	// (reseatRun.waitHerdrStopped); there it must report an error, not
	// running, for a server whose status it cannot read.
	WatchProbe       func() WatchLiveness
	HerdrServerProbe func() (running bool, err error)
	// RestartHerdr is --restart-herdr (spec
	// docs/specs/2026-11-20-org-reseat-restart-herdr.md, FR-1 to FR-4; plan
	// 2026-11-20-org-reseat-restart-herdr, D1 to D4): once the reseat has
	// stopped the seats, it stops the herdr server, waits for it to stop, and
	// starts it again with Org.HerdrServerGate before it starts the seats, so
	// they take the environment of the new server. PlanReseat refuses it,
	// with nothing stopped, from a herdr pane (HERDR_PANE_ID), when a target
	// list leaves out an active seat of the org, and when the server is not
	// running (reseatRestartHerdrErr); Reseat refuses it when the server holds
	// a pane that is not a pane of the seats it stops, before the first stop
	// and again right before `herdr server stop` (reseatOtherPanes).
	RestartHerdr bool
	// ClaudeConfigDir and ClaudeConfigDirGiven are --claude-config-dir as the
	// CLI checked it (FR-5): the CLAUDE_CONFIG_DIR of the `herdr server` the
	// gate starts ("" removes it), and whether it was given. The CLI applies
	// it to its gate; the reseat only repeats it in Retry (FR-5b).
	ClaudeConfigDir      string
	ClaudeConfigDirGiven bool
	// OnPlan is called with the reseat's plan once it is built and before
	// any seat is stopped: in a dry run right after PlanReseat returns it,
	// and in a real reseat once PlanReseat passed and the plan's id is in
	// the lock's marker, before the first stop; with RestartHerdr, in both,
	// once the first pane check passed too. Reseat calls it once at
	// most, and not at all when the reseat is refused before it has its
	// plan (a flag error, another reseat holding the lock, a refusal of
	// PlanReseat), the marker cannot be written, or the first pane check
	// refuses the restart (reseatFirstPanesErr). In a real reseat it runs
	// under the reseat lock, so it runs no ralph verb and nothing slow, and
	// waits for nothing; it reads the plan and does not change it. A nil
	// OnPlan is not called. The CLI uses it to print the codex hook trust
	// note of `ralph org reseat` for each codex seat of Plan.Start, reading a
	// few small files and writing lines to stderr (plan
	// 2026-11-16-reseat-notices).
	OnPlan func(ReseatPlan)
	// HintConfigPath is the --config the CLI was given, made absolute
	// against the CLI's working directory, so a printed command works from
	// another directory too, as --state-dir does; "" when it was given none.
	// Every ralph org command the reseat writes for its user to run repeats
	// it as --config, one shell word (reseatVerbCommand), so that command
	// reads the same ralph.toml as the reseat: the reseat, stop, spawn and
	// start commands of the retry (ReseatResult.Retry) and of its own
	// refusals, which read its [org.approval].mode, say, and the DECISION's
	// ralph org send (ReseatResult.Decision) and the ralph org read of the
	// leader's pane before it (ReseatResult.DecisionRead). Those two load
	// --config, or ./ralph.toml without it, and fail on a file they cannot
	// read or that is not valid. A command in the words of a check or step
	// the reseat shares with stop, spawn or start (stopSeatCommand, say) does
	// not carry it. Nothing else reads it.
	HintConfigPath string
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
	// of their first spawn_started in the org's current life, the org's
	// original spawn order (FR-6), which a reseat and its retry keep.
	Stop  []ReseatSeat
	Start []ReseatSeat
	// DryRun says the plan is a dry run's. StartsWatch and StartsHerdrServer
	// are set only by a dry run: no cross-org watch runs on the ledger, or
	// the herdr server is not running, and a real reseat starts it before it
	// stops any seat (P8).
	DryRun            bool
	StartsWatch       bool
	StartsHerdrServer bool
	// RestartsHerdr says the reseat stops the herdr server once the seats of
	// Stop are stopped and starts it again before the seats of Start
	// (ReseatParams.RestartHerdr, which passed reseatRestartHerdrErr).
	RestartsHerdr bool
	// LastDisbandedIndex is the index, in the manifest's events PlanReseat
	// read, of the org's latest real `disbanded` (reseatLastDisbanded); -1
	// when the org has none. Reseat reads the manifest again right before
	// each stop and each start and ends there when the org's latest
	// `disbanded` is no longer this one: the org was disbanded during the
	// reseat (ReseatResult.Disbanded).
	LastDisbandedIndex int
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
// reads the role, driver and model from), the index of its first
// spawn_started, which its start order is taken from (reseatStartOrder),
// touched, and cutOff.
type reseatLifeSeat struct {
	id           string
	latest       ManifestEvent
	latestIdx    int
	spawned      bool
	ran          ManifestEvent
	hasRan       bool
	firstStarted int // index of the first spawn_started; -1 for none
	// touched says one of the seat's events in the org's current life, a
	// state event or not, ends with a reseat=<id> note (reseatNoteID): a
	// reseat stopped the seat or started it, or tried to. --seat takes such a
	// seat, so the retry a reseat prints is not refused for a seat whose
	// first spawn in that life never reached `spawned` (P7).
	touched bool
	// cutOff is the id of the reseat that stopped the seat and was cut off
	// before it recorded `reseated` (P9): the seat's latest state event is a
	// `stopped` whose Details end with reseat=<id> (StopParams.Note), and the
	// org's current life holds no `reseated` with id=<id>. "" otherwise: the
	// seat is not stopped, a stop with no note stopped it (ralph org stop by
	// hand), or the reseat that stopped it recorded what it did.
	cutOff string
}

// active says the seat's latest state event is an active one.
func (s *reseatLifeSeat) active() bool {
	return isActiveEvent(s.latest.Event)
}

// startable says --seat takes the seat (D2, P7): it was spawned in the org's
// current life, it is active now, or a reseat stopped or started it in that
// life (touched). reseatTargets accepts a --seat by it, and the refusal of a
// reseat with no --seat names only such seats (reseatNoActiveSeatErr), so
// the reseat that refusal prints is not refused for its targets.
func (s *reseatLifeSeat) startable() bool {
	return s.spawned || s.active() || s.touched
}

// reseatStartOrder compares two seats by their place in the start order:
// the index of each one's first spawn_started in the org's current life, the
// org's original spawn order (FR-6). A later start of the seat, a failed one
// or a reseat's, does not move it. A seat with no spawn_started in the
// current life (its events there are a spawned or a stopped with none before
// them, as in a ledger written by hand) comes after every seat that has one,
// and such seats are ordered by seat id.
func reseatStartOrder(a, b *reseatLifeSeat) int {
	switch {
	case a.firstStarted >= 0 && b.firstStarted >= 0:
		return a.firstStarted - b.firstStarted
	case a.firstStarted >= 0:
		return -1
	case b.firstStarted >= 0:
		return 1
	}
	return strings.Compare(a.id, b.id)
}

// recorded is the role, driver and model of the seat's latest spawn_started
// or spawned, or of its latest state event when it has neither.
func (s *reseatLifeSeat) recorded() ManifestEvent {
	if s.hasRan {
		return s.ran
	}
	return s.latest
}

// reseatLifeSeats returns, in start order (reseatStartOrder), every seat of
// orgID that has a state event in the org's current life, each with its
// touched from the notes of that life's seat events and its cutOff from the
// `reseated` events of that life.
func reseatLifeSeats(events []ManifestEvent, orgID string) []*reseatLifeSeat {
	from := reseatLastDisbanded(events, orgID) + 1
	byID := make(map[string]*reseatLifeSeat)
	var seats []*reseatLifeSeat
	recorded := make(map[string]bool) // the ids of the life's `reseated`
	touched := make(map[string]bool)  // the seats with a reseat=<id> note
	for i := from; i < len(events); i++ {
		ev := events[i]
		if ev.DryRun || ev.OrgID != orgID {
			continue
		}
		if ev.SeatID == "" {
			if ev.Event == EventReseated {
				if id := reseatedID(ev.Details); id != "" {
					recorded[id] = true
				}
			}
			continue
		}
		if reseatNoteID(ev.Details) != "" {
			touched[ev.SeatID] = true
		}
		if !isStateEvent(ev.Event) {
			continue
		}
		s, ok := byID[ev.SeatID]
		if !ok {
			s = &reseatLifeSeat{id: ev.SeatID, firstStarted: -1}
			byID[ev.SeatID] = s
			seats = append(seats, s)
		}
		s.latest, s.latestIdx = ev, i
		switch ev.Event {
		case EventSpawnStarted:
			s.ran, s.hasRan = ev, true
			if s.firstStarted < 0 {
				s.firstStarted = i
			}
		case EventSpawned:
			s.ran, s.hasRan, s.spawned = ev, true, true
		}
	}
	for _, s := range seats {
		s.touched = touched[s.id]
		if id := reseatNoteID(s.latest.Details); s.latest.Event == EventStopped && id != "" && !recorded[id] {
			s.cutOff = id
		}
	}
	slices.SortStableFunc(seats, reseatStartOrder)
	return seats
}

// reseatLastDisbanded is the index in events of orgID's latest real
// `disbanded` (lastDisbandedIndexes), where the org's current life starts,
// or -1 when the org has none.
func reseatLastDisbanded(events []ManifestEvent, orgID string) int {
	if d, ok := lastDisbandedIndexes(events)[orgID]; ok {
		return d
	}
	return -1
}

// reseatNoteID is the reseat id of the note reseat=<id> that a reseat's stop
// or start puts at the end of an event's Details (noteDetails), or "" when
// details does not end with one.
func reseatNoteID(details string) string {
	fields := strings.Fields(details)
	if len(fields) == 0 {
		return ""
	}
	id, ok := strings.CutPrefix(fields[len(fields)-1], "reseat=")
	if !ok {
		return ""
	}
	return id
}

// reseatedID is the reseat id of a `reseated` event's Details, the value of
// its first token id=<id> (reseatedDetails), or "" when it has none.
func reseatedID(details string) string {
	fields := strings.Fields(details)
	if len(fields) == 0 {
		return ""
	}
	id, ok := strings.CutPrefix(fields[0], "id=")
	if !ok {
		return ""
	}
	return id
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
//  4. targets (D2, P7, P9): with no --seat, the org's active seats and the
//     seats a reseat that was cut off stopped (reseatLifeSeat.cutOff), and
//     none is refused with its stopped seats that --seat takes and the --seat
//     to name them (AC-6); a --seat is a seat spawned in the org's current
//     life (whatever its state is now: active, stopped, spawn_failed or
//     rejected), active now, or one a reseat stopped or started in that life
//     (reseatLifeSeat.touched; reseatLifeSeat.startable), so the retry a
//     reseat prints is not refused for a seat whose first spawn never
//     reached `spawned`. A target other than the leader whose role has no
//     role prompt template is refused too (P10, AC-21; reseatCustomRoleErr);
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
//  9. with RestartHerdr, the checks of the restart (reseatRestartHerdrErr):
//     a run from any herdr pane, a target list that leaves out an active
//     seat, a server that is not running or whose status cannot be read.
//     They come before the gates, which would start the watch and a stopped
//     server, so a restart refused here starts neither. Reseat's first pane
//     check (reseatFirstPanesErr) comes after this function, so after the
//     gates in a real run: a restart refused there may have started the
//     watch, but not the herdr server, which this check found running, and
//     it stops nothing;
//  10. the watch gate and then the herdr server gate, which may start the
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
		return refuse("", o.reseatNoBindingErr(orgID, binding, p.HintConfigPath))
	}
	targets, err := o.reseatTargets(p, binding, life)
	if err != nil {
		return refuse("", err)
	}
	// scope is the --scope every start other than the leader's gets: the
	// org's reservation.
	scope := strings.Join(ActiveReservation(events, orgID), ", ")
	if seat, err := o.reseatCustomRoleErr(orgID, binding, scope, p.HintConfigPath, targets); err != nil {
		return refuse(seat, err)
	}

	plan := ReseatPlan{OrgID: orgID, Binding: binding, DryRun: p.DryRun, LastDisbandedIndex: reseatLastDisbanded(events, orgID)}
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
	if p.RestartHerdr {
		if err := o.reseatRestartHerdrErr(p, life, targets); err != nil {
			return refuse("", err)
		}
		plan.RestartsHerdr = true
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

// reseatRestartHerdrErr is check 9 of PlanReseat, for a reseat with
// RestartHerdr (spec FR-2, plan D4): `herdr server stop` closes every pane
// of the server, so it refuses a run from any herdr pane (HERDR_PANE_ID, read
// with o.getenv), which would close with it, and a target list that leaves
// out an active seat of the org (life), which would die with the server and
// not start again. It then needs p.HerdrServerProbe, and, in a real reseat,
// Org.HerdrServerGate, which starts the server again, and refuses a server
// the probe cannot read or does not find running: a stopped server needs no
// restart, since the gate of a reseat without RestartHerdr starts it in this
// command's environment.
func (o *Org) reseatRestartHerdrErr(p ReseatParams, life, targets []*reseatLifeSeat) error {
	if pane := o.getenv(herdrPaneIDEnv); pane != "" {
		return fmt.Errorf("--restart-herdr stops the herdr server, which closes every pane it holds, and this command runs in "+
			"the herdr pane %s: run ralph org reseat --restart-herdr from a shell outside herdr", pane)
	}
	var left []string
	for _, s := range life {
		if s.active() && !slices.Contains(targets, s) {
			left = append(left, s.id)
		}
	}
	if len(left) > 0 {
		return fmt.Errorf("--restart-herdr stops the herdr server, which closes the panes of every seat of org_id %q, so it "+
			"takes every active seat of the org, and the active seats %s are not named with --seat: run it with no --seat, "+
			"or name them too", p.OrgID, strings.Join(left, ", "))
	}
	if p.HerdrServerProbe == nil {
		return errors.New("--restart-herdr reads whether the herdr server runs, and this reseat has nothing to read it with " +
			"(ReseatParams.HerdrServerProbe is nil)")
	}
	if !p.DryRun && o.HerdrServerGate == nil {
		return errors.New("--restart-herdr starts the herdr server again with the herdr server gate, and this reseat has none " +
			"(Org.HerdrServerGate is nil)")
	}
	running, err := p.HerdrServerProbe()
	if err != nil {
		return err
	}
	if !running {
		return errors.New("--restart-herdr stops the herdr server and starts it again, and no herdr server is running: run " +
			"the same ralph org reseat without --restart-herdr, which starts the server in this command's environment (with " +
			"--claude-config-dir when it is given) before it starts the seats")
	}
	return nil
}

// reseatOwnPanes is the pane ids of the seats plan stops: the pane each
// one's latest state event recorded. With RestartHerdr, Stop holds every
// active seat of the org (reseatRestartHerdrErr). A seat whose latest event
// has no pane (a spawn_started) adds none, so its pane, if herdr made one,
// counts as another's and the restart is refused.
func reseatOwnPanes(plan ReseatPlan) map[string]bool {
	own := make(map[string]bool, len(plan.Stop))
	for _, seat := range plan.Stop {
		if seat.Latest.PaneID != "" {
			own[seat.Latest.PaneID] = true
		}
	}
	return own
}

// reseatOtherPanes lists every pane of the herdr server (PaneListAll, bounded
// by driverCallTimeout) and returns, in herdr's order, those that are not
// panes of the seats plan stops (reseatOwnPanes): another org's or another
// repository's seats, a headless director in herdr, a shell, or the record
// of a pane that herdr restored after an earlier `herdr server stop`.
func (o *Org) reseatOtherPanes(plan ReseatPlan) ([]driver.HerdrPane, error) {
	var panes []driver.HerdrPane
	err := o.callWithTimeout(func(ctx context.Context) error {
		var err error
		panes, err = o.Herdr.PaneListAll(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	own := reseatOwnPanes(plan)
	var others []driver.HerdrPane
	for _, pane := range panes {
		if !own[pane.PaneID] {
			others = append(others, pane)
		}
	}
	return others, nil
}

// reseatFirstPanesErr is the first pane check of a reseat with RestartsHerdr
// (spec FR-2, plan D1), which Reseat makes before it stops any seat, in a dry
// run as well: an error when the pane list cannot be read or names a pane
// that is not a pane of the seats plan stops (reseatOtherPanes), nil
// otherwise. Reseat returns it as a refusal, with nothing stopped.
func (o *Org) reseatFirstPanesErr(plan ReseatPlan) error {
	others, err := o.reseatOtherPanes(plan)
	if err != nil {
		return fmt.Errorf("--restart-herdr lists the herdr server's panes before it stops anything, and the list could not "+
			"be read (%w): the server is stopped only once it holds no pane but the org's active seats", err)
	}
	if len(others) == 0 {
		return nil
	}
	return fmt.Errorf("--restart-herdr stops the herdr server, which closes every pane it holds, and %s; %s",
		reseatOtherPanesText(fmt.Sprintf("org_id %q's active seats", plan.OrgID), others), reseatOtherPanesHint)
}

// reseatOtherPanesText says that the herdr server holds others, which are
// not panes of whose (the seats a reseat stops, said by the caller), and
// names each with its id, label, and cwd.
func reseatOtherPanesText(whose string, others []driver.HerdrPane) string {
	descs := make([]string, len(others))
	for i, pane := range others {
		label, cwd := "no label", "no cwd reported"
		if pane.Label != "" {
			label = fmt.Sprintf("label %q", pane.Label)
		}
		if pane.Cwd != "" {
			cwd = fmt.Sprintf("cwd %q", pane.Cwd)
		}
		descs[i] = fmt.Sprintf("%s (%s, %s)", pane.PaneID, label, cwd)
	}
	what := "a pane that is not a pane"
	if len(others) > 1 {
		what = fmt.Sprintf("%d panes that are not panes", len(others))
	}
	return fmt.Sprintf("the server holds %s of %s: %s", what, whose, strings.Join(descs, "; "))
}

// reseatOtherPanesHint is how a refusal for other panes says to tell a pane
// that is only a record herdr restored (no terminal and no process; the
// evidence of docs/evidence/org-reseat-live-2026-11-12.md, AC-8) from a live
// one, and what to do about each.
const reseatOtherPanesHint = "a pane for which herdr pane process-info <pane_id> finds no process is only the record of a " +
	"pane that herdr restored after an earlier herdr server stop, and herdr pane close <pane_id> closes it; the others " +
	"(another org's seats, a headless director in herdr, a shell) end with the server, so run the reseat with " +
	"--restart-herdr again once they are gone, or without it to start the seats on the server as it is"

// reseatVerbCommand is a ralph org command a reseat prints for its user to
// run, its retry, one its refusals name, or the DECISION's send and the read
// of the leader's pane before it: VerbCommand of verbAndArgs followed by
// --config hintConfigPath when that is set, one shell word (ShellWord), so
// the command reads the ralph.toml the reseat read
// (ReseatParams.HintConfigPath), and by --state-dir hintStateDir when that
// is set.
func reseatVerbCommand(verbAndArgs, hintConfigPath, hintStateDir string) string {
	if hintConfigPath != "" {
		verbAndArgs += " --config " + ShellWord(hintConfigPath)
	}
	return VerbCommand(verbAndArgs, hintStateDir)
}

// reseatNoBindingErr is the refusal of an org whose current life has no
// complete split plan binding (binding nil or incomplete; FR-2, D4): its
// leader's task cannot be made again from the ledger, so its seats are
// stopped and started one by one. The stop and the spawn it names carry
// --config hintConfigPath (reseatVerbCommand).
func (o *Org) reseatNoBindingErr(orgID string, binding *FeatureBinding, hintConfigPath string) error {
	why := "has no split plan binding since its last disband (it was not started with ralph org start --plan)"
	if binding != nil {
		why = "has an incomplete split plan binding since its last disband (" + binding.String() + ")"
	}
	id := ShellWord(orgID)
	return fmt.Errorf("org_id %q %s, so ralph org reseat does not start its seats again: stop a seat with %s and start it "+
		"again with %s (its leader with ralph org start)", orgID, why,
		reseatVerbCommand("stop --org-id "+id+" --seat <seat>", hintConfigPath, o.InboxHintStateDir),
		reseatVerbCommand("spawn --org-id "+id+" --id <seat> --role <role>", hintConfigPath, o.InboxHintStateDir))
}

// reseatTargets returns the seats of the org's current life a reseat with p
// starts again, in start order: with no --seat, the active ones and those a
// reseat that was cut off stopped (cutOff, P9), which are not active, so the
// reseat starts them and does not stop them; otherwise the named ones, each
// spawned in the current life, active now, or touched by a reseat in that
// life (reseatLifeSeat.startable; D2, P7). The leader is first.
func (o *Org) reseatTargets(p ReseatParams, binding *FeatureBinding, life []*reseatLifeSeat) ([]*reseatLifeSeat, error) {
	life = reseatLeaderFirst(life)
	if len(p.Seats) == 0 {
		var targets []*reseatLifeSeat
		for _, s := range life {
			if s.active() || s.cutOff != "" {
				targets = append(targets, s)
			}
		}
		if len(targets) == 0 {
			return nil, o.reseatNoActiveSeatErr(p.OrgID, p.StateDir, p.HintConfigPath, binding, life)
		}
		return targets, nil
	}
	for _, name := range p.Seats {
		if slices.ContainsFunc(life, func(s *reseatLifeSeat) bool { return s.id == name && s.startable() }) {
			continue
		}
		var can []string
		for _, s := range life {
			if s.startable() {
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
// that has no active seat and no seat a reseat that was cut off stopped (D2,
// AC-6, P9). It names only seats --seat takes (reseatLifeSeat.startable), so
// the reseat it prints is not refused for its targets: the stopped ones and
// the reseat that names them with --seat; with none of them stopped, all of
// them, each with its latest state event; with none at all (no seat spawned
// or touched by a reseat since the last disband), the start of its leader
// from the split plan in stateDir. A seat stopped by hand before it was ever
// spawned is not named. Each command it names carries --config
// hintConfigPath (reseatVerbCommand).
func (o *Org) reseatNoActiveSeatErr(orgID, stateDir, hintConfigPath string, binding *FeatureBinding, life []*reseatLifeSeat) error {
	var stopped, others []string
	for _, s := range life {
		if !s.startable() {
			continue
		}
		if s.latest.Event == EventStopped {
			stopped = append(stopped, s.id)
		} else {
			others = append(others, s.id+" ("+s.latest.Event+")")
		}
	}
	id := ShellWord(orgID)
	lead := fmt.Sprintf("org_id %q has no active seat since its last disband and no seat stopped by a reseat that was cut off, "+
		"so a reseat with no --seat has no seat to start again", orgID)
	switch {
	case len(stopped) > 0:
		cmd := "reseat --org-id " + id
		for _, s := range stopped {
			cmd += " --seat " + ShellWord(s)
		}
		return fmt.Errorf("%s; the stopped seats reseat can start again are %s: to start them again, run %s", lead,
			strings.Join(stopped, ", "), reseatVerbCommand(cmd, hintConfigPath, o.InboxHintStateDir))
	case len(others) > 0:
		return fmt.Errorf("%s; the seats reseat can start again are %s, none of them stopped: name the ones to start "+
			"again with --seat (%s)", lead, strings.Join(others, ", "),
			reseatVerbCommand("reseat --org-id "+id+" --seat <seat>", hintConfigPath, o.InboxHintStateDir))
	}
	return fmt.Errorf("%s; no seat has been spawned since then: start its leader with %s", lead,
		reseatVerbCommand("start --plan "+ShellWord(filepath.Join(SplitPlansDirIn(stateDir), binding.Split+".md"))+
			" --feature "+ShellWord(binding.Feature)+" --org-id "+id, hintConfigPath, o.InboxHintStateDir))
}

// reseatCustomRoleErr refuses the first target other than the leader, in
// start order, whose role has no role prompt template (P10, AC-21): a custom
// role, which `ralph org spawn --role <role> --prompt <text>` starts with
// that --prompt alone (Spawn), and the ledger does not hold the --prompt, so
// the seat would start again with no instructions. "Has a template" is what
// Spawn reads: RenderRolePrompt's ok. The leader starts through StartFeature
// from prompts/leader.md. It returns the seat and the refusal, which says to
// stop that seat and spawn it again with its --prompt by hand and to name
// the other seats with --seat; "" and nil when no target is such a seat.
//
// The spawn it names starts the seat as reseat would: with the driver and
// model the ledger recorded for it (no --model when that model is "", the
// driver CLI's own default, which spawn takes only from the role's
// [org.role_models] entry), in the feature worktree, and with scope, the
// --scope reseat gives every seat other than the leader, without which the
// default autonomous permission mode refuses the spawn
// (autonomousScopeGateErr). The stop and the spawn carry --config
// hintConfigPath (reseatVerbCommand).
func (o *Org) reseatCustomRoleErr(orgID string, binding *FeatureBinding, scope, hintConfigPath string, targets []*reseatLifeSeat) (string, error) {
	for _, s := range targets {
		if s.id == LeaderIdentity {
			continue
		}
		rec := s.recorded()
		role := rec.Role
		_, ok, err := RenderRolePrompt(role, RolePromptVars{})
		if err != nil {
			return s.id, err
		}
		if ok {
			continue
		}
		id := ShellWord(orgID)
		args := "spawn --org-id " + id + " --id " + ShellWord(s.id) + " --role " + ShellWord(role) +
			" --driver " + ShellWord(rec.Driver)
		if rec.Model != "" {
			args += " --model " + ShellWord(rec.Model)
		}
		args += " --cwd " + ShellWord(binding.Worktree) + " --scope " + ShellWord(scope) + " --prompt <its prompt>"
		spawn := "spawn it again with " + reseatVerbCommand(args, hintConfigPath, o.InboxHintStateDir)
		if s.active() {
			stop := reseatVerbCommand("stop --org-id "+id+" --seat "+ShellWord(s.id), hintConfigPath, o.InboxHintStateDir)
			spawn = "stop it with " + stop + " and " + spawn
		}
		return s.id, fmt.Errorf("its role %q has no role prompt template, and the ledger does not hold the --prompt it was "+
			"spawned with, so reseat would start it with no instructions: %s by hand, and to reseat the org's other seats, "+
			"name them with --seat and leave %s out", role, spawn, s.id)
	}
	return "", nil
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

// reseatLockFile is the ledger's reseat lock (P6), in the manifest's
// directory: a reseat holds it from before its checks until it has recorded
// `reseated`, so two reseats of one ledger never run at once. Its content is
// the holder's reseatMarker while it is held, and empty once released.
const reseatLockFile = "reseat.lock"

// How a reseat with RestartHerdr waits for the herdr server to stop (spec
// FR-4): after `herdr server stop`, it reads ReseatParams.HerdrServerProbe
// every reseatHerdrStopPollInterval until it reports the server not running,
// for up to reseatHerdrStopWait. Package vars so tests can shorten them.
var (
	reseatHerdrStopWait         = 10 * time.Second
	reseatHerdrStopPollInterval = 100 * time.Millisecond
)

// reseatTestHooks are the points where a test steps into Reseat. Each is nil
// outside tests.
var reseatTestHooks struct {
	// beforeStopCheck runs right before Reseat reads the manifest again to
	// check the seat it is about to stop (P6), with that seat's id.
	beforeStopCheck func(seatID string)
	// beforeStart runs right before Reseat starts a seat of Plan.Start, with
	// that seat's id.
	beforeStart func(seatID string)
}

// ReseatSeatOutcome is what a reseat left one of its target seats as. The
// value is the token the `reseated` event's Details record for the seat
// (EventReseated).
type ReseatSeatOutcome string

const (
	// ReseatSeatStarted: the seat was started again (stopped first when it
	// was active).
	ReseatSeatStarted ReseatSeatOutcome = "started"
	// ReseatSeatStopped: the reseat stopped the seat and ended before its
	// start.
	ReseatSeatStopped ReseatSeatOutcome = "stopped"
	// ReseatSeatStopFailed: the seat's stop failed (Stop recorded
	// stop_failed, or could not record), so the seat may still run.
	ReseatSeatStopFailed ReseatSeatOutcome = "stop_failed"
	// ReseatSeatNotStopped: the seat was active when the reseat checked it,
	// and the reseat did not stop it: its record changed after the check
	// (ReseatResult.Changed names it), or the reseat ended before its stop
	// (at a disband of the org too, ReseatResult.Disbanded).
	ReseatSeatNotStopped ReseatSeatOutcome = "not_stopped"
	// ReseatSeatNotStarted: the seat was not active (a --seat that was
	// stopped or whose start failed, or a seat a reseat that was cut off
	// stopped) and the reseat ended before its start.
	ReseatSeatNotStarted ReseatSeatOutcome = "not_started"
	// ReseatSeatStartFailed: the seat's start was refused or failed, or it
	// came back idempotent on a seat another command started during the
	// reseat on another role, driver or model (reseatIdempotentMismatchErr).
	ReseatSeatStartFailed ReseatSeatOutcome = "start_failed"
)

// ReseatSeatResult is what a reseat did with one target seat.
type ReseatSeatResult struct {
	SeatID  string
	Outcome ReseatSeatOutcome
	// Stop is the result of the seat's Stop; nil when the reseat did not
	// stop it.
	Stop *StopResult
	// Spawn is the result of the seat's start, the Spawn of StartFeature's
	// result for the leader; nil when the reseat did not start it.
	Spawn *SpawnResult
}

// ReseatResult is Reseat's outcome.
type ReseatResult struct {
	// Plan is what PlanReseat decided; the zero ReseatPlan when the reseat
	// was refused or failed before its plan.
	Plan ReseatPlan
	// Seats holds every target seat (Plan.Start) in start order, once the
	// reseat got past its checks; nil for a dry run and for a refusal.
	Seats []ReseatSeatResult
	// Err is the first error: a refusal before anything is stopped (a
	// *reseatRefusal: PlanReseat's, another reseat holding the lock, or, with
	// RestartHerdr, the first pane check's), the stop or start that failed, a
	// changed seat (Changed), a disband of the org during the reseat
	// (Disbanded), with RestartHerdr a step of the server's restart that
	// failed (reseatRun.restartHerdr), and, joined to it, a `reseated` that
	// could not be recorded.
	Err error
	// HerdrStopped says a reseat with RestartHerdr saw the herdr server not
	// running after its `herdr server stop`, and HerdrRestarted that the herdr
	// server gate passed after that, so the server runs again with the
	// environment the gate gives it. Both are false without RestartHerdr.
	HerdrStopped, HerdrRestarted bool
	// Changed is a target seat another command stopped or started during the
	// reseat, which the reseat leaves as it is and out of Retry, in either of
	// two cases. One: when the reseat came to stop it, the seat's latest state
	// event in the org's current life was no longer the one PlanReseat read
	// (FR-5a); the reseat did not stop it and started nothing. Two: its start
	// came back idempotent on a seat another command started on another role,
	// driver or model than the plan's, and the reseat was not given --driver
	// and --model for it (reseatIdempotentMismatchErr); the seat keeps running
	// as that command started it, and the reseat started no seat after it.
	// Given --driver and --model, that seat is not Changed and stays in
	// Retry, which starts it on them. "" when there is no such seat.
	Changed string
	// Disbanded says the org was disbanded during the reseat: right before a
	// stop or a start, the org's latest `disbanded` in the manifest was no
	// longer the one PlanReseat read (ReseatPlan.LastDisbandedIndex), so the
	// reseat ended there, without that stop or start or any after it, and
	// Changed is "". Retry, Decision and DecisionRead are then "": a reseat
	// of the disbanded org is refused (it has no binding, reseatNoBindingErr),
	// and the leader a DECISION would go to is gone. The feature org starts again with ralph org start --plan.
	Disbanded bool
	// Retry is the reseat that starts again the target seats not started
	// (all but Changed), with --driver and --model when the one seat it
	// names was given them, --restart-herdr and --claude-config-dir as
	// reseatRetryHerdrFlags has them, --config as ReseatParams.HintConfigPath
	// has it, and --state-dir as Org.InboxHintStateDir has it; "" when there
	// is none.
	Retry string
	// Decision is the `ralph org send` of the DECISION that asks the leader
	// to send the target seats that started their TASKs again, when the
	// leader was not one of the targets and at least one target started
	// (D5, FR-7): those seats run with no TASK. That holds also when a later
	// start failed, and the retry then gives a DECISION of its own for the
	// seats it starts. The leader keeps running through such a reseat:
	// PlanReseat refuses the start of any other seat while it is not active
	// (spawnFeatureSeatErr). "" otherwise. Reseat never sends it. It carries
	// --config as ReseatParams.HintConfigPath has it and --state-dir as
	// Org.InboxHintStateDir has it (ReseatDecisionCommand).
	Decision string
	// DecisionRead is the `ralph org read` of the leader's pane the CLI prints
	// before Decision, so its user checks the leader before running it, with
	// the same --config and --state-dir as Decision (reseatLeaderReadCommand);
	// "" when Decision is "".
	DecisionRead string
}

// reseatMarker is what reseat.lock holds while a reseat holds it (P6): the
// org it reseats, its id once PlanReseat gave one, and its pid, as one JSON
// line. A reseat that finds the lock held reads it for its refusal.
type reseatMarker struct {
	OrgID string `json:"org_id"`
	ID    string `json:"reseat_id,omitempty"`
	PID   int    `json:"pid"`
}

// writeReseatMarker replaces the content of f, the open reseat.lock, with m
// as a JSON line, or with nothing when m is nil.
func writeReseatMarker(f *os.File, m *reseatMarker) error {
	var data []byte
	if m != nil {
		line, err := json.Marshal(m)
		if err != nil {
			return fmt.Errorf("org: reseat: encode the marker of %s: %w", f.Name(), err)
		}
		data = append(line, '\n')
	}
	if err := f.Truncate(0); err != nil {
		return fmt.Errorf("org: reseat: write the marker of %s: %w", f.Name(), err)
	}
	if _, err := f.WriteAt(data, 0); err != nil {
		return fmt.Errorf("org: reseat: write the marker of %s: %w", f.Name(), err)
	}
	return nil
}

// readReseatMarker returns the marker in the reseat.lock at path. ok is
// false when the file cannot be read, is empty, or holds no JSON marker with
// an org_id.
func readReseatMarker(path string) (m reseatMarker, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return reseatMarker{}, false
	}
	if err := json.Unmarshal(bytes.TrimSpace(data), &m); err != nil || m.OrgID == "" {
		return reseatMarker{}, false
	}
	return m, true
}

// takeReseatLock takes reseat.lock in the ledger's directory without waiting
// for it (P6), and writes the marker of orgID into it. Another reseat
// holding it is a *reseatRefusal of orgID that names that reseat from its
// marker; any other error is returned as it is. Nothing is stopped either
// way.
func (o *Org) takeReseatLock(orgID string) (*heldFileLock, error) {
	dir := filepath.Dir(o.Manifest.Path())
	lock, err := acquireHeldFileLock(dir, reseatLockFile, "reseat", 0)
	if errors.Is(err, errFileLockHeld) {
		path := filepath.Join(dir, reseatLockFile)
		holder := "another ralph org reseat"
		if m, ok := readReseatMarker(path); ok {
			holder = fmt.Sprintf("another ralph org reseat (of org_id %q, pid %d)", m.OrgID, m.PID)
			if m.ID != "" {
				holder = fmt.Sprintf("another ralph org reseat (reseat=%s of org_id %q, pid %d)", m.ID, m.OrgID, m.PID)
			}
		}
		return nil, &reseatRefusal{orgID: orgID, err: fmt.Errorf("%s holds the ledger's reseat lock %s: the reseats of one "+
			"ledger run one at a time, so run this one again once that one has ended", holder, path)}
	}
	if err != nil {
		return nil, err
	}
	if err := writeReseatMarker(lock.f, &reseatMarker{OrgID: orgID, PID: os.Getpid()}); err != nil {
		_ = lock.release()
		return nil, err
	}
	return lock, nil
}

// releaseReseatLock empties the marker of lock and releases it.
func releaseReseatLock(lock *heldFileLock) {
	_ = writeReseatMarker(lock.f, nil)
	_ = lock.release()
}

// Reseat runs `ralph org reseat` with p: with p.DryRun, it is PlanReseat
// alone, with no lock taken (P8), then, when its plan RestartsHerdr, the
// first pane check (reseatFirstPanesErr), which lists the panes and stops
// nothing, and then p.OnPlan with the plan when neither refused. Otherwise,
// in this order:
//
//  0. it checks p's org_id and its --seat, --driver and --model
//     (checkReseatFlags), as Spawn and Regate check their input before their
//     lock: a refusal there writes nothing, not even reseat.lock, and comes
//     before the refusal of another reseat holding the lock;
//  1. it takes the ledger's reseat.lock without waiting, and holds it until
//     it returns: another reseat holding it is refused at once, naming it
//     (P6, FR-5a);
//  2. it plans with PlanReseat, whose refusal returns before anything is
//     stopped or written (FR-4), writes the reseat id into the lock's
//     marker, makes the first pane check when the plan RestartsHerdr
//     (reseatFirstPanesErr: a refusal, with nothing stopped), and calls
//     p.OnPlan with the plan, still under the lock;
//  3. it stops the seats of Plan.Stop in that order (D3), each with Stop and
//     the note reseat=<id>. Right before each stop it reads the manifest
//     again, and when the org was disbanded since PlanReseat read it
//     (ReseatPlan.LastDisbandedIndex), or the seat's latest state event in
//     the org's current life is no longer the one PlanReseat read (Latest at
//     LatestIndex), it does not stop that seat and ends there, starting
//     nothing (ReseatResult.Disbanded, FR-5a). A stop that fails (its pane
//     was not closed, or a record failed) ends it too, starting nothing
//     (FR-5). When the plan RestartsHerdr and every seat stopped, it then
//     restarts the herdr server (reseatRun.restartHerdr: the second pane
//     check, `herdr server stop`, the wait for the server to stop, and the
//     herdr server gate), and ends there, starting no seat, when a step of
//     that fails;
//  4. it starts the seats of Plan.Start in that order: the leader with
//     StartFeature and the others with Spawn, each with the note
//     reseat=<id>, and ends at the first start that is not spawned or
//     idempotent (FR-6), and at an idempotent one on a seat another command
//     started on another role, driver or model than the plan's
//     (reseatIdempotentMismatchErr). Right before each start it reads the
//     manifest again, and when the org was disbanded since PlanReseat read
//     it (ReseatPlan.LastDisbandedIndex), it ends there without that start
//     (ReseatResult.Disbanded): Spawn would start a seat of the disbanded
//     org with no binding and no approval check, and StartFeature a new life
//     of it without the old one's approvals;
//  5. when it stopped or started a seat (or tried to), it records
//     `reseated` (EventReseated) with each target's outcome, also when it
//     ended early (P4).
//
// The result says what became of each target, the reseat that starts the
// ones not started (Retry), and, when the leader was not one of the targets,
// the DECISION for the leader about the targets that started, also when a
// later start failed (Decision). Reseat never sends it.
//
// Reseat takes no context and its stops and starts cannot be cancelled: the
// CLI catches SIGINT, SIGTERM, SIGHUP and SIGPIPE while it runs, so it gets
// to step 5 (P9). A reseat cut off anyway between its first stop and step 5
// (killed, say) leaves the seats it stopped with reseat=<id> and no
// `reseated` of that id. The same reseat run again, with the same Seats,
// starts them, and so does one with no Seats (reseatLifeSeat.cutOff), which
// also stops and starts every active seat.
func (o *Org) Reseat(p ReseatParams) ReseatResult {
	if p.DryRun {
		plan, err := o.PlanReseat(p)
		if err == nil && plan.RestartsHerdr {
			if perr := o.reseatFirstPanesErr(plan); perr != nil {
				err = &reseatRefusal{orgID: p.OrgID, err: perr}
			}
		}
		if err == nil && p.OnPlan != nil {
			p.OnPlan(plan)
		}
		return ReseatResult{Plan: plan, Err: err}
	}
	if err := ValidateIdentifier("org_id", p.OrgID); err != nil {
		return ReseatResult{Err: err}
	}
	// PlanReseat checks the flags again under the lock, as it does for a dry
	// run; the check reads nothing, so the second run gives the same answer.
	if err := checkReseatFlags(p); err != nil {
		return ReseatResult{Err: &reseatRefusal{orgID: p.OrgID, err: err}}
	}
	lock, err := o.takeReseatLock(p.OrgID)
	if err != nil {
		return ReseatResult{Err: err}
	}
	defer releaseReseatLock(lock)

	plan, err := o.PlanReseat(p)
	if err != nil {
		return ReseatResult{Plan: plan, Err: err}
	}
	if err := writeReseatMarker(lock.f, &reseatMarker{OrgID: p.OrgID, ID: plan.ID, PID: os.Getpid()}); err != nil {
		return ReseatResult{Plan: plan, Err: &reseatRefusal{orgID: p.OrgID, err: err}}
	}
	if plan.RestartsHerdr {
		if err := o.reseatFirstPanesErr(plan); err != nil {
			return ReseatResult{Plan: plan, Err: &reseatRefusal{orgID: p.OrgID, err: err}}
		}
	}
	if p.OnPlan != nil {
		p.OnPlan(plan)
	}

	r := &reseatRun{
		o: o, plan: plan, note: "reseat=" + plan.ID, hintConfigPath: p.HintConfigPath,
		index:      make(map[string]int, len(plan.Start)),
		herdrProbe: p.HerdrServerProbe,
		retryHerdr: reseatRetryHerdr{claudeConfigDir: p.ClaudeConfigDir, claudeConfigDirGiven: p.ClaudeConfigDirGiven},
	}
	r.res = ReseatResult{Plan: plan, Seats: make([]ReseatSeatResult, len(plan.Start))}
	for i, seat := range plan.Start {
		r.index[seat.SeatID] = i
		r.res.Seats[i] = ReseatSeatResult{SeatID: seat.SeatID, Outcome: ReseatSeatNotStarted}
		if seat.Active {
			r.res.Seats[i].Outcome = ReseatSeatNotStopped
		}
	}
	if r.stopAll() && r.restartHerdr() {
		r.startAll()
	}
	return r.finish()
}

// reseatRun is one Reseat past its checks: the plan, the note its stops and
// starts carry, the --config its retry, its DECISION and the read before it
// repeat (ReseatParams.HintConfigPath), the index in plan.Start of each
// target seat, the herdr server probe (ReseatParams.HerdrServerProbe), the
// --claude-config-dir its retry repeats, the result so far, and whether a
// stop, a start, or `herdr server stop` was attempted.
type reseatRun struct {
	o              *Org
	plan           ReseatPlan
	note           string
	hintConfigPath string
	index          map[string]int
	herdrProbe     func() (bool, error)
	retryHerdr     reseatRetryHerdr
	res            ReseatResult
	attempted      bool
}

// stopAll stops the seats of plan.Stop in order (step 3 of Reseat) and
// reports whether every one stopped. It ends right before a stop that
// checkBeforeStop does not let through (a disband of the org since the plan,
// a seat whose record changed since the check, a manifest it cannot read
// again), and at a stop that failed, setting res.Err (and res.Disbanded or
// res.Changed).
func (r *reseatRun) stopAll() bool {
	id, orgID := r.plan.ID, r.plan.OrgID
	for _, seat := range r.plan.Stop {
		out := &r.res.Seats[r.index[seat.SeatID]]
		if hook := reseatTestHooks.beforeStopCheck; hook != nil {
			hook(seat.SeatID)
		}
		if !r.checkBeforeStop(seat) {
			return false
		}
		r.attempted = true
		stop := r.o.Stop(StopParams{OrgID: orgID, Seat: seat.SeatID, Note: r.note})
		out.Stop = &stop
		if stop.Err != nil {
			out.Outcome = ReseatSeatStopFailed
			r.res.Err = fmt.Errorf("org: reseat %s of org_id %q: the stop of seat %q failed, so the reseat stopped there "+
				"and started no seat: %w", id, orgID, seat.SeatID, stop.Err)
			return false
		}
		out.Outcome = ReseatSeatStopped
	}
	return true
}

// checkBeforeStop reads the manifest again right before the stop of seat
// (step 3 of Reseat) and reports whether the reseat goes on to stop it. It
// ends the reseat instead, setting res.Err, at a manifest it cannot read;
// when the org was disbanded since the plan (reseatDisbandedIn), as
// endIfDisbanded does before a start, setting res.Disbanded; and when the
// seat's latest state event in the org's current life is no longer the one
// PlanReseat read (FR-5a), setting res.Changed. One read serves both checks,
// so a disband is never taken for a changed seat: a disband starts a new
// life of the org, in which the seat has no state event.
func (r *reseatRun) checkBeforeStop(seat ReseatSeat) bool {
	rr, err := r.o.Manifest.Read()
	if err != nil {
		r.res.Err = fmt.Errorf("org: reseat %s of org_id %q ended before the stop of seat %q, and started no seat: "+
			"read the manifest again: %w", r.plan.ID, r.plan.OrgID, seat.SeatID, err)
		return false
	}
	if disbanded, changed := reseatDisbandedIn(rr.Events, r.plan); changed {
		r.res.Disbanded = true
		r.res.Err = reseatDisbandedErr(r.plan, "stop", seat.SeatID, disbanded, nil)
		return false
	}
	now, found := reseatSeatNow(rr.Events, r.plan.OrgID, seat.SeatID)
	if !found || now.latestIdx != seat.LatestIndex || now.latest != seat.Latest {
		r.res.Changed = seat.SeatID
		r.res.Err = reseatChangedErr(r.plan, seat, now, found)
		return false
	}
	return true
}

// restartHerdr restarts the herdr server between the stops and the starts
// of a plan that RestartsHerdr (step 3 of Reseat; spec FR-3 and FR-4, plan
// D1 and D2), and reports whether the reseat goes on to start the seats. A
// plan that does not restart the server goes on at once. Otherwise, in this
// order, with no other call between the pane list and the stop:
//
//  1. the second pane check: the panes of the server listed again
//     (reseatOtherPanes). A list that cannot be read, or a pane that is not a
//     pane of the seats the reseat stopped (one made while it stopped them),
//     ends the reseat with the server not stopped and the seats left stopped;
//  2. `herdr server stop` (HerdrClient.ServerStop, bounded by
//     driverCallTimeout). A stop that fails ends the reseat, after one read
//     of the probe, whose answer the error gives;
//  3. the wait for the probe to report the server not running
//     (waitHerdrStopped), which sets res.HerdrStopped;
//  4. the herdr server gate (checkHerdrServerGate), which starts the server
//     in the caller's environment, with the CLI's --claude-config-dir, and
//     sets res.HerdrRestarted. A gate that fails ends the reseat.
//
// Whatever ends it, the seats it stopped stay stopped, no `reseated` is
// missed (finish records it), and Retry repeats --restart-herdr only while
// res.HerdrStopped is false (reseatRetryHerdrFlags).
func (r *reseatRun) restartHerdr() bool {
	if !r.plan.RestartsHerdr {
		return true
	}
	id, orgID := r.plan.ID, r.plan.OrgID
	others, err := r.o.reseatOtherPanes(r.plan)
	if err != nil {
		r.res.Err = fmt.Errorf("org: reseat %s of org_id %q could not list the herdr server's panes again before it stops the "+
			"server (%w), so it did not stop the server and started no seat, and the seats it stopped are left stopped",
			id, orgID, err)
		return false
	}
	if len(others) > 0 {
		r.res.Err = fmt.Errorf("org: reseat %s of org_id %q did not stop the herdr server and started no seat, and the seats it "+
			"stopped are left stopped: listed again before the stop of the server, %s; %s", id, orgID,
			reseatOtherPanesText("the seats this reseat stopped", others), reseatOtherPanesHint)
		return false
	}
	r.attempted = true
	if err := r.o.callWithTimeout(r.o.Herdr.ServerStop); err != nil {
		running, perr := r.herdrProbe()
		if perr == nil && !running {
			r.res.HerdrStopped = true
		}
		status := "it reports the server not running"
		switch {
		case perr != nil:
			status = "it could not be read: " + perr.Error()
		case running:
			status = "it reports the server running"
		}
		r.res.Err = fmt.Errorf("org: reseat %s of org_id %q: herdr server stop failed (%w), so the reseat started no seat; "+
			"the herdr server's status read after it: %s", id, orgID, err, status)
		return false
	}
	if !r.waitHerdrStopped() {
		return false
	}
	r.res.HerdrStopped = true
	if err := r.o.checkHerdrServerGate(); err != nil {
		r.res.Err = fmt.Errorf("org: reseat %s of org_id %q: the herdr server stopped, and its start failed, so the reseat "+
			"started no seat: %w", id, orgID, err)
		return false
	}
	r.res.HerdrRestarted = true
	return true
}

// waitHerdrStopped reads the herdr server probe every
// reseatHerdrStopPollInterval, for up to reseatHerdrStopWait, until it
// reports the server not running, and reports whether it did (spec FR-4). A
// read that fails is not fatal by itself. At the deadline it sets res.Err,
// with what the last read found: the server may still stop.
func (r *reseatRun) waitHerdrStopped() bool {
	deadline := time.Now().Add(reseatHerdrStopWait)
	for {
		running, err := r.herdrProbe()
		if err == nil && !running {
			return true
		}
		if !time.Now().Before(deadline) {
			last := "it reports the server running"
			if err != nil {
				last = "it could not be read: " + err.Error()
			}
			r.res.Err = fmt.Errorf("org: reseat %s of org_id %q: the herdr server was not reported stopped within %s of herdr "+
				"server stop (the last read of its status: %s), so the reseat did not start it again and started no seat; "+
				"the server may still stop: herdr status server --json says whether it runs", r.plan.ID, r.plan.OrgID,
				reseatHerdrStopWait, last)
			return false
		}
		time.Sleep(min(reseatHerdrStopPollInterval, time.Until(deadline)))
	}
}

// startAll starts the seats of plan.Start in order (step 4 of Reseat), the
// leader with StartFeature and the others with Spawn, and ends at the first
// one whose start is neither spawned nor idempotent, or is idempotent on a
// seat running on another role, driver or model than the plan's
// (reseatIdempotentMismatchErr), setting res.Err, and res.Changed for the
// latter when the reseat was not given --driver and --model for the seat. It
// also ends right before a start when the org was disbanded since the plan
// (endIfDisbanded).
func (r *reseatRun) startAll() {
	for i, seat := range r.plan.Start {
		if hook := reseatTestHooks.beforeStart; hook != nil {
			hook(seat.SeatID)
		}
		if r.endIfDisbanded(i) {
			return
		}
		out := &r.res.Seats[i]
		r.attempted = true
		var spawn SpawnResult
		if seat.Leader {
			start := *seat.Start
			start.Note = r.note
			start.Reseat = &StartFeatureReseat{ID: r.plan.ID, Seats: reseatSeatsAfterLeader(r.plan)}
			spawn = r.o.StartFeature(start).Spawn
		} else {
			sp := *seat.Spawn
			sp.Note = r.note
			spawn = r.o.Spawn(sp)
		}
		out.Spawn = &spawn
		if spawn.Err == nil && spawn.Outcome == SpawnOutcomeIdempotent {
			if err := reseatIdempotentMismatchErr(r.plan, seat, spawn.Seat); err != nil {
				out.Outcome = ReseatSeatStartFailed
				r.res.Err = err
				if !seat.Overridden {
					r.res.Changed = seat.SeatID
				}
				return
			}
		}
		if spawn.Err == nil && (spawn.Outcome == SpawnOutcomeSpawned || spawn.Outcome == SpawnOutcomeIdempotent) {
			out.Outcome = ReseatSeatStarted
			continue
		}
		out.Outcome = ReseatSeatStartFailed
		cause := spawn.Err
		if cause == nil {
			cause = fmt.Errorf("its start ended %s", spawn.Outcome)
		}
		r.res.Err = fmt.Errorf("org: reseat %s of org_id %q: seat %q did not start again, so the reseat started no seat "+
			"after it: %w", r.plan.ID, r.plan.OrgID, seat.SeatID, cause)
		return
	}
}

// reseatSeatsAfterLeader is the ids of the target seats of plan that the
// reseat starts after the leader, in the order it starts them: plan.Start
// without the leader. The progress section of the leader's task names them
// (StartFeatureReseat), since none is running yet when the leader's task is
// written.
func reseatSeatsAfterLeader(plan ReseatPlan) []string {
	var ids []string
	for _, seat := range plan.Start {
		if !seat.Leader {
			ids = append(ids, seat.SeatID)
		}
	}
	return ids
}

// reseatIdempotentMismatchErr is the error of a start of seat that came back
// idempotent with got, a seat running on another role, driver or model than
// the ones the plan starts seat on (a model "" on both sides is the same, the
// driver CLI's default); nil when got runs on those. An idempotent start
// returns a seat whose latest state event is `spawned`, which a seat the
// reseat stopped or found not active does not have, so another command
// spawned the seat after the reseat stopped it or checked it: reseat.lock
// keeps out only other reseats. The reseat leaves that seat running, starts
// no seat after it, and, unless it was given --driver and --model for the
// seat, leaves the seat out of Retry (ReseatResult.Changed): with no
// --driver and --model, a retry would start it on the settings of the latest
// spawn in the ledger, those it already runs on.
func reseatIdempotentMismatchErr(plan ReseatPlan, seat ReseatSeat, got SeatStatus) error {
	if got.Role == seat.Role && got.Driver == seat.Driver && got.Model == seat.Model {
		return nil
	}
	settings := func(role, driver, model string) string {
		return fmt.Sprintf("role %s, driver %s and model %s", role, driver, ModelOrCLIDefault(model))
	}
	return fmt.Errorf("org: reseat %s of org_id %q: seat %q was started by another command during the reseat, on %s, "+
		"not the %s the reseat meant to start it on, so the reseat left it running as it is and started no seat after it",
		plan.ID, plan.OrgID, seat.SeatID, settings(got.Role, got.Driver, got.Model), settings(seat.Role, seat.Driver, seat.Model))
}

// finish records `reseated` when a stop or a start was attempted (step 5 of
// Reseat), and fills in res.Retry, res.Decision and res.DecisionRead: the
// DECISION names the targets that started, when the leader is not one of the
// targets, whether or not a later start failed. It leaves the three "" when
// the org was disbanded during the reseat (res.Disbanded).
func (r *reseatRun) finish() ReseatResult {
	if r.attempted {
		if err := r.o.appendEvent(ManifestEvent{
			TS: r.o.now(), OrgID: r.plan.OrgID, Event: EventReseated, Details: reseatedDetails(r.plan, r.res.Seats),
		}); err != nil {
			r.res.Err = errors.Join(r.res.Err, fmt.Errorf("org: reseat %s of org_id %q: record %s: %w",
				r.plan.ID, r.plan.OrgID, EventReseated, err))
		}
	}
	if r.res.Disbanded {
		return r.res
	}
	herdr := r.retryHerdr
	herdr.restartHerdr = r.plan.RestartsHerdr && !r.res.HerdrStopped
	r.res.Retry = reseatRetryCommandWithHerdr(r.plan, r.res.Seats, r.res.Changed, herdr, r.hintConfigPath, r.o.InboxHintStateDir)
	var started []string
	for i, seat := range r.plan.Start {
		if seat.Leader {
			return r.res
		}
		if r.res.Seats[i].Outcome == ReseatSeatStarted {
			started = append(started, seat.SeatID)
		}
	}
	if len(started) > 0 {
		text := ReseatDecisionText(r.plan.ID, started)
		r.res.Decision = ReseatDecisionCommand(r.plan.OrgID, text, r.hintConfigPath, r.o.InboxHintStateDir)
		r.res.DecisionRead = reseatLeaderReadCommand(r.plan.OrgID, r.hintConfigPath, r.o.InboxHintStateDir)
	}
	return r.res
}

// reseatSeatNow returns what the org's current life holds about seatID in
// events, the manifest read again right before the seat's stop; found is
// false when the seat has no state event in it.
func reseatSeatNow(events []ManifestEvent, orgID, seatID string) (now *reseatLifeSeat, found bool) {
	for _, s := range reseatLifeSeats(events, orgID) {
		if s.id == seatID {
			return s, true
		}
	}
	return nil, false
}

// reseatDisbandedNow reads the manifest again and reports whether the org of
// plan was disbanded since PlanReseat read it (reseatDisbandedIn).
func reseatDisbandedNow(o *Org, plan ReseatPlan) (disbanded ManifestEvent, changed bool, err error) {
	rr, err := o.Manifest.Read()
	if err != nil {
		return ManifestEvent{}, false, err
	}
	disbanded, changed = reseatDisbandedIn(rr.Events, plan)
	return disbanded, changed, nil
}

// reseatDisbandedIn reports whether events, the manifest read again, show
// the org of plan disbanded since PlanReseat read it: its latest real
// `disbanded` is no longer the one at plan.LastDisbandedIndex. disbanded is
// that latest `disbanded` when there is one.
func reseatDisbandedIn(events []ManifestEvent, plan ReseatPlan) (disbanded ManifestEvent, changed bool) {
	i := reseatLastDisbanded(events, plan.OrgID)
	if i == plan.LastDisbandedIndex {
		return ManifestEvent{}, false
	}
	if i >= 0 {
		disbanded = events[i]
	}
	return disbanded, true
}

// endIfDisbanded checks, right before the start of plan.Start[i], that the
// org was not disbanded since the plan (reseatDisbandedNow), and reports
// whether the reseat ends there instead, setting res.Err, and res.Disbanded
// when the org was disbanded. The seats from plan.Start[i] on keep the
// outcome they have, stopped or not started.
func (r *reseatRun) endIfDisbanded(i int) bool {
	id, orgID, seatID := r.plan.ID, r.plan.OrgID, r.plan.Start[i].SeatID
	disbanded, changed, err := reseatDisbandedNow(r.o, r.plan)
	if err != nil {
		r.res.Err = fmt.Errorf("org: reseat %s of org_id %q ended before the start of seat %q, and started no seat after "+
			"it: read the manifest again: %w", id, orgID, seatID, err)
		return true
	}
	if !changed {
		return false
	}
	var started []string
	for _, s := range r.res.Seats[:i] {
		if s.Outcome == ReseatSeatStarted {
			started = append(started, s.SeatID)
		}
	}
	r.res.Disbanded = true
	r.res.Err = reseatDisbandedErr(r.plan, "start", seatID, disbanded, started)
	return true
}

// reseatDisbandedErr is the error of a reseat that found its org disbanded
// right before step, "stop" or "start", of seat: disbanded is the org's
// latest `disbanded`, and started the seats the reseat started before it,
// which started in the life that disband ended and may or may not run now.
// Before a stop, started is nil: a reseat starts no seat before its last
// stop.
func reseatDisbandedErr(plan ReseatPlan, step, seat string, disbanded ManifestEvent, started []string) error {
	when := ""
	if disbanded.TS != "" {
		when = " (disbanded at " + disbanded.TS + ")"
	}
	before := ""
	switch len(started) {
	case 0:
	case 1:
		before = fmt.Sprintf("; seat %q started before the disband, in the life it ended, so ralph org status shows "+
			"whether it still runs", started[0])
	default:
		quoted := make([]string, len(started))
		for i, s := range started {
			quoted[i] = fmt.Sprintf("%q", s)
		}
		before = fmt.Sprintf("; seats %s started before the disband, in the life it ended, so ralph org status shows "+
			"whether they still run", strings.Join(quoted, ", "))
	}
	none := "started no seat after it"
	if step == "stop" {
		none = "started no seat"
	}
	return fmt.Errorf("org: reseat %s of org_id %q ended before the %s of seat %q, and %s: the "+
		"org was disbanded during the reseat%s%s; the feature org starts again with ralph org start --plan",
		plan.ID, plan.OrgID, step, seat, none, when, before)
}

// reseatChangedErr is the error of a reseat that found seat's record changed
// right before its stop (FR-5a): now is what the org's current life holds
// about the seat, found false when it holds nothing. ralph only appends to
// the manifest, so the seat it checked can lose its state events only to a
// disband of the org, which checkBeforeStop reads first (res.Disbanded):
// found is false here only for a manifest rewritten by hand, and the
// sentence for it stays for that case.
func reseatChangedErr(plan ReseatPlan, seat ReseatSeat, now *reseatLifeSeat, found bool) error {
	is := "no state event since the org's last disband"
	if found {
		is = fmt.Sprintf("%s at %s", now.latest.Event, now.latest.TS)
	}
	return fmt.Errorf("org: reseat %s of org_id %q ended before the stop of seat %q, and started no seat: the seat's "+
		"latest state event is now %s, not the %s at %s the reseat checked, so another command stopped or started it "+
		"meanwhile and the reseat left it as it is", plan.ID, plan.OrgID, seat.SeatID, is, seat.Latest.Event, seat.Latest.TS)
}

// reseatedDetails is the Details of a reseat's `reseated` (EventReseated):
// `id=<id> seats=<seat>:<outcome>,...` in start order, then
// ` changed=<seat>:<driver>/<model>,...` for the seats --driver and --model
// start on another driver or model than the ledger's, the model "" written
// as nothing after the slash. No token holds a space: seat ids and drivers
// are words, and a model is one word with no comma
// (ValidateSpawnEnvelope).
func reseatedDetails(plan ReseatPlan, seats []ReseatSeatResult) string {
	parts := make([]string, len(seats))
	for i, s := range seats {
		parts[i] = s.SeatID + ":" + string(s.Outcome)
	}
	details := "id=" + plan.ID + " seats=" + strings.Join(parts, ",")
	var changed []string
	for _, s := range plan.Start {
		if s.Overridden && (s.Driver != s.RecordedDriver || s.Model != s.RecordedModel) {
			changed = append(changed, s.SeatID+":"+s.Driver+"/"+s.Model)
		}
	}
	if len(changed) > 0 {
		details += " changed=" + strings.Join(changed, ",")
	}
	return details
}

// reseatRetryHerdr is what the retry of a reseat repeats of its herdr server
// flags (spec FR-5b, plan D4): --restart-herdr when restartHerdr is set, and
// --claude-config-dir claudeConfigDir when claudeConfigDirGiven is.
type reseatRetryHerdr struct {
	restartHerdr         bool
	claudeConfigDir      string
	claudeConfigDirGiven bool
}

// reseatRetryHerdrFlags is the words h adds to a retry: " --restart-herdr"
// when h.restartHerdr is set, then " --claude-config-dir <dir>" (one shell
// word, ShellWord, which quotes "" as two single quotes) when
// h.claudeConfigDirGiven is; "" for the zero h.
//
// Reseat sets restartHerdr for a plan that RestartsHerdr only while it has
// not seen the server stop (ReseatResult.HerdrStopped is false): the server
// may still run, so the retry restarts it, or, when it stopped meanwhile, is
// refused with the advice to run the reseat without --restart-herdr. Once it
// has seen the server stop, the retry leaves --restart-herdr out, whether the
// gate then started the server or failed to: after a start, a second restart
// would stop the seats this reseat started (the leader, say) again, and with
// the server down and no pane on it, the gate of the retry's PlanReseat starts
// the server, with --claude-config-dir, as a restart would. The retry repeats
// --claude-config-dir whenever it was given, so a server the retry starts gets
// the same CLAUDE_CONFIG_DIR, not the caller's.
func reseatRetryHerdrFlags(h reseatRetryHerdr) string {
	flags := ""
	if h.restartHerdr {
		flags += " --restart-herdr"
	}
	if h.claudeConfigDirGiven {
		flags += " --claude-config-dir " + ShellWord(h.claudeConfigDir)
	}
	return flags
}

// reseatRetryCommandWithHerdr is the reseat that starts again the target
// seats of plan that seats does not have as started, but skip, in start
// order, with --driver and --model when it names one seat that was given
// them, then the herdr server flags of herdr (reseatRetryHerdrFlags; none
// for the zero herdr), and --config hintConfigPath and --state-dir
// hintStateDir when each is set (reseatVerbCommand); "" when no seat is left.
func reseatRetryCommandWithHerdr(plan ReseatPlan, seats []ReseatSeatResult, skip string, herdr reseatRetryHerdr, hintConfigPath, hintStateDir string) string {
	var left []ReseatSeat
	for i, s := range seats {
		if s.Outcome != ReseatSeatStarted && s.SeatID != skip {
			left = append(left, plan.Start[i])
		}
	}
	if len(left) == 0 {
		return ""
	}
	cmd := "reseat --org-id " + ShellWord(plan.OrgID)
	for _, s := range left {
		cmd += " --seat " + ShellWord(s.SeatID)
	}
	if len(left) == 1 && left[0].Overridden {
		cmd += " --driver " + ShellWord(left[0].Driver) + " --model " + ShellWord(left[0].Model)
	}
	cmd += reseatRetryHerdrFlags(herdr)
	return reseatVerbCommand(cmd, hintConfigPath, hintStateDir)
}

// ReseatDecisionText is the DECISION for the leader of an org after reseat
// id started seats, none of them the leader, again (D5): they started with
// no TASK, so the leader is asked to send each its current TASK again.
func ReseatDecisionText(id string, seats []string) string {
	return "TYPE: " + protocol.TypeDecision + "\n\n" +
		"ralph org reseat が座席 " + strings.Join(seats, ", ") + " を止めて立て直しました(reseat=" + id + ")。" +
		"立て直した座席は TASK を持たずに立っています。それぞれに今の TASK を送り直してください。"
}

// ReseatDecisionCommand is the one POSIX shell command, `ralph org send`,
// that sends text to the leader seat (LeaderIdentity) of orgID, with
// --config hintConfigPath and --state-dir hintStateDir when each is set
// (reseatVerbCommand), as the retry has them: send loads --config, or
// ./ralph.toml without it, before it sends. The --text value comes last,
// single-quoted with its line breaks kept (shellQuote), as
// ApprovalDecisionCommand has it.
func ReseatDecisionCommand(orgID, text, hintConfigPath, hintStateDir string) string {
	return reseatVerbCommand("send --org-id "+ShellWord(orgID)+" --to "+LeaderIdentity, hintConfigPath, hintStateDir) +
		" --text " + shellQuote(text)
}

// reseatLeaderReadCommand is the `ralph org read` of the leader's pane of
// orgID that the CLI prints before the DECISION (ReseatResult.DecisionRead),
// with --config hintConfigPath and --state-dir hintStateDir when each is set
// (reseatVerbCommand): read loads --config, or ./ralph.toml without it,
// before it reads the pane.
func reseatLeaderReadCommand(orgID, hintConfigPath, hintStateDir string) string {
	return reseatVerbCommand("read --org-id "+ShellWord(orgID)+" --seat "+LeaderIdentity, hintConfigPath, hintStateDir)
}
