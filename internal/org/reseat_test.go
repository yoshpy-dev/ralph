package org

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// PlanReseat at the org layer (plan docs/plans/active/2026-11-12-org-reseat.md,
// slice S1: AC-4 to AC-7, AC-9, AC-11 and AC-19, and P7's --seat; slice S6:
// P9 and P10, AC-20 and AC-21). The fixture is featureStart
// (feature_test.go): the feature org auth-core of the split plan auth-split
// on fake drivers and fake worktrees.

// reseatScope is the --scope a reseat gives auth-core's seats: the org's
// reservation, joined with ", ".
const reseatScope = "docs/auth.md, internal/auth/"

// newReseatOrg is newFeatureStart with auth-core started (its leader on
// claude sonnet), its implementation plan approved (per-plan, the fixture's
// mode), and its implementer (claude opus) and reviewer (claude, the CLI's
// default model) spawned as its leader spawns them.
func newReseatOrg(t *testing.T) *featureStart {
	t.Helper()
	st := newFeatureStart(t)
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	approveFeaturePlan(t, st.o, "auth-core", st.worktree("auth-core"))
	st.spawnSeat(t, implementerSeatID, implementerRole, "opus")
	st.spawnSeat(t, "reviewer", "reviewer", "")
	return st
}

// spawnSeat spawns seat id of auth-core with role and model on claude, in
// the feature worktree with the reservation as its scope.
func (st *featureStart) spawnSeat(t *testing.T, id, role, model string) {
	t.Helper()
	r := st.o.Spawn(SpawnParams{
		OrgID: "auth-core", SeatID: id, Role: role, Driver: "claude", Model: model,
		Cwd: st.worktree("auth-core"), Scope: reseatScope, TimeoutMS: 5000,
	})
	if r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn %s: %+v", id, r)
	}
}

// stopSeat stops seat id of auth-core.
func (st *featureStart) stopSeat(t *testing.T, id string) {
	t.Helper()
	if r := st.o.Stop(StopParams{OrgID: "auth-core", Seat: id}); r.Err != nil {
		t.Fatalf("stop %s: %v", id, r.Err)
	}
}

// reseatParams is a reseat of auth-core naming seats.
func (st *featureStart) reseatParams(seats ...string) ReseatParams {
	return ReseatParams{OrgID: "auth-core", Seats: seats, StateDir: st.stateDir, RepoRoot: st.root, TimeoutMS: 5000}
}

// reseatLedger is what PlanReseat must leave as it is: the counts of
// spawnSnapshot, the manifest's bytes, the names in the state dir, and the
// fake worktrees' records.
type reseatLedger struct {
	snap     spawnSnapshot
	manifest string
	names    string
	records  int
}

func takeReseatLedger(t *testing.T, st *featureStart) reseatLedger {
	t.Helper()
	entries, err := os.ReadDir(st.stateDir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	return reseatLedger{takeSpawnSnapshot(t, st.o, st.h, st.a), fileOrAbsent(t, st.o.Manifest.Path()), strings.Join(names, ","), len(st.wt.records)}
}

// planReseat runs PlanReseat with p and fails when it wrote, called or made
// anything: a manifest event, a receipt, a herdr or agmsg call, a file in the
// state dir, or a worktree (ensure or cleanup).
func (st *featureStart) planReseat(t *testing.T, p ReseatParams) (ReseatPlan, error) {
	t.Helper()
	before, calls := takeReseatLedger(t, st), len(st.wt.calls)
	plan, err := st.o.PlanReseat(p)
	if after := takeReseatLedger(t, st); after != before {
		t.Errorf("PlanReseat wrote or called something: %+v -> %+v", before, after)
	}
	for _, c := range st.wt.calls[calls:] {
		if !strings.HasPrefix(c, "lookup:") && !strings.HasPrefix(c, "branch:") {
			t.Errorf("PlanReseat made the worktree call %q", c)
		}
	}
	return plan, err
}

// mustPlanReseat is planReseat that fails on a refusal.
func (st *featureStart) mustPlanReseat(t *testing.T, p ReseatParams) ReseatPlan {
	t.Helper()
	plan, err := st.planReseat(t, p)
	if err != nil {
		t.Fatalf("PlanReseat(%+v): %v", p, err)
	}
	return plan
}

// assertReseatRefused runs planReseat and checks that it refused with a
// *reseatRefusal of auth-core whose message holds every string of want.
func (st *featureStart) assertReseatRefused(t *testing.T, p ReseatParams, want ...string) {
	t.Helper()
	_ = st.checkReseatRefused(t, p, want...)
}

// checkReseatRefused is assertReseatRefused for a test that reads the
// refusal further: it returns the refusal.
func (st *featureStart) checkReseatRefused(t *testing.T, p ReseatParams, want ...string) error {
	t.Helper()
	plan, err := st.planReseat(t, p)
	var refusal *reseatRefusal
	if err == nil || !errors.As(err, &refusal) {
		t.Fatalf("PlanReseat(%+v) = %+v, %v; want a refusal", p, plan, err)
	}
	want = append([]string{`org: reseat of org_id "auth-core" is refused before anything is stopped: `}, want...)
	for _, w := range want {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("refusal %q does not contain %q", err, w)
		}
	}
	return err
}

// seatIDs are the seat ids of seats, in order.
func seatIDs(seats []ReseatSeat) []string {
	ids := make([]string, len(seats))
	for i, s := range seats {
		ids[i] = s.SeatID
	}
	return ids
}

// TestPlanReseat_DefaultTargetsAndOrder covers D2 and D3 (AC-9's plan): with
// no --seat, every active seat is a target; the leader stops last and starts
// first, the other seats stop in the reverse of their spawn order and start in
// it. Each seat keeps the role, driver and model of its spawn (the reviewer's
// "" stays the CLI's default), the leader starts through StartFeature from
// the split plan in the ledger, and the others through Spawn in the feature
// worktree with the reservation as their scope. The id is the clock's UTC
// time and 4 hex digits, and nothing is written or called.
func TestPlanReseat_DefaultTargetsAndOrder(t *testing.T) {
	st := newReseatOrg(t)
	st.o.Now = func() time.Time { return time.Date(2026, 11, 12, 12, 4, 5, 0, time.FixedZone("JST", 9*3600)) }
	saved := reseatIDRand
	reseatIDRand = bytes.NewReader([]byte{0xab, 0x0d})
	t.Cleanup(func() { reseatIDRand = saved })
	reviewer := config.ReviewerConfig{ClaudeModel: "opus", ClaudeEffort: "high", CodexModel: "gpt-x", CodexEffort: "low"}
	p := st.reseatParams()
	p.Reviewer = reviewer

	plan := st.mustPlanReseat(t, p)
	if plan.ID != "20261112T030405-ab0d" || plan.OrgID != "auth-core" || plan.DryRun || plan.StartsWatch || plan.StartsHerdrServer {
		t.Errorf("plan id %q org %q dry-run %v notes %v %v; want 20261112T030405-ab0d of auth-core, no dry run and no notes",
			plan.ID, plan.OrgID, plan.DryRun, plan.StartsWatch, plan.StartsHerdrServer)
	}
	events := mustReadEvents(t, st.o)
	if want := ActiveFeature(events, "auth-core"); plan.Binding == nil || *plan.Binding != *want {
		t.Errorf("Binding = %+v; want %+v", plan.Binding, want)
	}
	if got := seatIDs(plan.Start); !slices.Equal(got, []string{LeaderIdentity, implementerSeatID, "reviewer"}) {
		t.Fatalf("start order = %q; want leader, implementer, reviewer", got)
	}
	if got := seatIDs(plan.Stop); !slices.Equal(got, []string{"reviewer", implementerSeatID, LeaderIdentity}) {
		t.Fatalf("stop order = %q; want reviewer, implementer, leader", got)
	}

	wt := st.worktree("auth-core")
	leader := plan.Start[0]
	wantStart := StartFeatureParams{
		StateDir: st.stateDir, RepoRoot: st.root, SplitPath: filepath.Join(SplitPlansDirIn(st.stateDir), "auth-split.md"),
		Feature: "auth-core", OrgID: "auth-core", Driver: "claude", Model: "sonnet", LeaderDriver: "claude", TimeoutMS: 5000,
		Reviewer: reviewer,
	}
	if !leader.Leader || leader.Role != LeaderIdentity || leader.Spawn != nil || leader.Start == nil || *leader.Start != wantStart {
		t.Errorf("leader = %+v (start %+v); want the leader's start %+v", leader, leader.Start, wantStart)
	}
	for i, want := range []struct{ id, role, model string }{{implementerSeatID, implementerRole, "opus"}, {"reviewer", "reviewer", ""}} {
		seat := plan.Start[i+1]
		wantSpawn := SpawnParams{
			OrgID: "auth-core", SeatID: want.id, Role: want.role, Driver: "claude", Model: want.model,
			Cwd: wt, Scope: reseatScope, TimeoutMS: 5000, LeaderDriver: "claude",
		}
		if seat.Leader || seat.Start != nil || seat.Spawn == nil || seat.Spawn.Task != "" || seat.Spawn.Prompt != "" ||
			seat.Spawn.OrgID != wantSpawn.OrgID || seat.Spawn.SeatID != wantSpawn.SeatID || seat.Spawn.Role != wantSpawn.Role ||
			seat.Spawn.Driver != wantSpawn.Driver || seat.Spawn.Model != wantSpawn.Model || seat.Spawn.Cwd != wantSpawn.Cwd ||
			seat.Spawn.Scope != wantSpawn.Scope || seat.Spawn.TimeoutMS != wantSpawn.TimeoutMS || seat.Spawn.LeaderDriver != wantSpawn.LeaderDriver {
			t.Errorf("seat %s = %+v (spawn %+v); want the spawn %+v", want.id, seat, seat.Spawn, wantSpawn)
		}
	}
	for _, seat := range plan.Start {
		if !seat.Active || seat.Overridden || seat.Driver != seat.RecordedDriver || seat.Model != seat.RecordedModel ||
			seat.Latest.Event != EventSpawned || events[seat.LatestIndex] != seat.Latest {
			t.Errorf("seat %s: active %v overridden %v driver %s/%s model %q/%q latest %s at %d; "+
				"want an active seat with its recorded driver and model, and its spawned at its index",
				seat.SeatID, seat.Active, seat.Overridden, seat.Driver, seat.RecordedDriver, seat.Model, seat.RecordedModel,
				seat.Latest.Event, seat.LatestIndex)
		}
	}
	for i, seat := range plan.Stop {
		if j := slices.IndexFunc(plan.Start, func(s ReseatSeat) bool { return s.SeatID == seat.SeatID }); plan.Start[j].Start != seat.Start ||
			plan.Start[j].Spawn != seat.Spawn {
			t.Errorf("stop %d (%s) does not carry the start of the same seat", i, seat.SeatID)
		}
	}
}

// TestPlanReseat_IDShape covers P2 with the real random source: the UTC time
// of the clock, a hyphen and 4 lowercase hex digits.
func TestPlanReseat_IDShape(t *testing.T) {
	id, err := newReseatID(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	if err != nil || !regexp.MustCompile(`^20260102T030405-[0-9a-f]{4}$`).MatchString(id) {
		t.Fatalf("newReseatID = %q, %v; want 20260102T030405-<4 hex>", id, err)
	}
}

// TestPlanReseat_SeatAcceptsASeatSpawnedInTheCurrentLife covers P7 (AC-18's
// half at this layer): --seat names a seat whose latest state is stopped,
// spawn_failed or rejected, once it was spawned in the org's current life.
// Such a seat is started and not stopped, with the driver and model of its
// latest spawn_started or spawned. The leader, which keeps running, is in
// neither list.
func TestPlanReseat_SeatAcceptsASeatSpawnedInTheCurrentLife(t *testing.T) {
	for _, tc := range []struct {
		name, event, model string
		after              []ManifestEvent // appended after the reviewer's stop
	}{
		{"stopped", EventStopped, "", nil},
		{"spawn_failed after a later spawn_started", EventSpawnFailed, "haiku", []ManifestEvent{
			{OrgID: "auth-core", SeatID: "reviewer", Event: EventSpawnStarted, Role: "reviewer", Driver: "claude", Model: "haiku"},
			{OrgID: "auth-core", SeatID: "reviewer", Event: EventSpawnFailed, Role: "reviewer", Driver: "claude", Model: "haiku"},
		}},
		{"rejected", EventRejected, "", []ManifestEvent{
			{OrgID: "auth-core", SeatID: "reviewer", Event: EventRejected, Role: "reviewer", Driver: "claude", Model: "nope nope"},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newReseatOrg(t)
			st.stopSeat(t, "reviewer")
			for _, ev := range tc.after {
				if err := st.o.Manifest.Append(ev); err != nil {
					t.Fatal(err)
				}
			}
			plan := st.mustPlanReseat(t, st.reseatParams("reviewer"))
			if len(plan.Stop) != 0 || !slices.Equal(seatIDs(plan.Start), []string{"reviewer"}) {
				t.Fatalf("stop %q start %q; want no stop and the reviewer started", seatIDs(plan.Stop), seatIDs(plan.Start))
			}
			seat := plan.Start[0]
			if seat.Active || seat.Latest.Event != tc.event || seat.Model != tc.model || seat.Driver != "claude" || seat.Spawn == nil ||
				seat.Spawn.Model != tc.model {
				t.Fatalf("reviewer = %+v; want an inactive seat whose latest event is %s, started on claude %q", seat, tc.event, tc.model)
			}
		})
	}
}

// TestPlanReseat_NoActiveSeatNamesTheStoppedSeats covers AC-6's refusal: with
// every seat stopped and no --seat, the reseat is refused, naming the stopped
// seats in start order and the reseat that names them with --seat, with
// --config when the reseat was given one (ReseatParams.HintConfigPath) and
// --state-dir when the ledger was named with it. That reseat then passes and
// starts them all, stopping none.
func TestPlanReseat_NoActiveSeatNamesTheStoppedSeats(t *testing.T) {
	st := newReseatOrg(t)
	for _, id := range []string{"reviewer", implementerSeatID, LeaderIdentity} {
		st.stopSeat(t, id)
	}
	st.assertReseatRefused(t, st.reseatParams(),
		`org_id "auth-core" has no active seat since its last disband and no seat stopped by a reseat that was cut off, `+
			`so a reseat with no --seat has no seat to start again; `+
			`its stopped seats are leader, implementer, reviewer: to start them again, run `+
			`ralph org reseat --org-id auth-core --seat leader --seat implementer --seat reviewer`)
	st.o.InboxHintStateDir = "/state/org dir"
	err := st.checkReseatRefused(t, st.reseatParams())
	if !strings.HasSuffix(err.Error(), "--seat reviewer --state-dir '/state/org dir'") {
		t.Errorf("refusal %q does not end with the command's --state-dir", err)
	}
	withConfig := st.reseatParams()
	withConfig.HintConfigPath = "/cfg dir/ralph.toml"
	err = st.checkReseatRefused(t, withConfig)
	if !strings.HasSuffix(err.Error(), "--seat reviewer --config '/cfg dir/ralph.toml' --state-dir '/state/org dir'") {
		t.Errorf("refusal %q does not end with the command's --config and --state-dir", err)
	}

	plan := st.mustPlanReseat(t, st.reseatParams(LeaderIdentity, implementerSeatID, "reviewer"))
	if len(plan.Stop) != 0 || !slices.Equal(seatIDs(plan.Start), []string{LeaderIdentity, implementerSeatID, "reviewer"}) {
		t.Fatalf("stop %q start %q; want no stop and all three started", seatIDs(plan.Stop), seatIDs(plan.Start))
	}
}

// TestPlanReseat_NoActiveSeatWithoutStoppedOnes covers the rest of the
// refusal of a reseat with no --seat and no active seat: with no stopped
// seat, it names the seats spawned in the current life with their state;
// with none spawned at all, the start of the org's leader. Each command it
// names carries the --config the reseat was given.
func TestPlanReseat_NoActiveSeatWithoutStoppedOnes(t *testing.T) {
	withConfig := func(st *featureStart) ReseatParams {
		p := st.reseatParams()
		p.HintConfigPath = "/cfg/ralph.toml"
		return p
	}
	t.Run("a seat whose spawn failed", func(t *testing.T) {
		st := newFeatureStart(t)
		st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
		if err := st.o.Manifest.Append(ManifestEvent{OrgID: "auth-core", SeatID: LeaderIdentity, Event: EventSpawnFailed}); err != nil {
			t.Fatal(err)
		}
		st.assertReseatRefused(t, st.reseatParams(), "none of its seats is stopped, and the seats spawned since then are leader (spawn_failed): "+
			"name the ones to start again with --seat (ralph org reseat --org-id auth-core --seat <seat>)")
		st.assertReseatRefused(t, withConfig(st), "(ralph org reseat --org-id auth-core --seat <seat> --config /cfg/ralph.toml)")
	})
	t.Run("no seat spawned", func(t *testing.T) {
		st := newFeatureStart(t)
		f := &FeatureBinding{Split: "auth-split", Feature: "auth-core", Digest: planDigestOf(t, st.plan), Branch: "fix/auth-core",
			Worktree: st.worktree("auth-core"), Approval: config.ApprovalModePerPlan, FeatureDigest: featureDigestOf(t, st.plan, "auth-core")}
		if err := st.o.Manifest.Append(boundReserveEvent("auth-core", f, "docs/auth.md", "internal/auth/")); err != nil {
			t.Fatal(err)
		}
		start := "no seat has been spawned since then: start its leader with ralph org start --plan " +
			ShellWord(filepath.Join(SplitPlansDirIn(st.stateDir), "auth-split.md")) + " --feature auth-core --org-id auth-core"
		if err := st.checkReseatRefused(t, st.reseatParams(), start); !strings.HasSuffix(err.Error(), start) {
			t.Errorf("refusal %q does not end with the start; want no --config", err)
		}
		st.assertReseatRefused(t, withConfig(st), start+" --config /cfg/ralph.toml")
	})
}

// cutOffReseatID is the id of a reseat the tests of P9 leave cut off: it
// stopped seats and recorded no `reseated`.
const cutOffReseatID = "20261112T000000-0c0f"

// stopSeatByReseat stops seat id of auth-core as the reseat reseatID stops
// it, with the note reseat=<reseatID> at the end of the `stopped`'s Details
// (StopParams.Note), and records nothing else: a reseat cut off after that
// stop leaves the ledger so.
func (st *featureStart) stopSeatByReseat(t *testing.T, id, reseatID string) {
	t.Helper()
	if r := st.o.Stop(StopParams{OrgID: "auth-core", Seat: id, Note: "reseat=" + reseatID}); r.Err != nil {
		t.Fatalf("stop %s by reseat %s: %v", id, reseatID, r.Err)
	}
}

// appendReseated records an org-level `reseated` of auth-core with id and
// seats, as a reseat that got to its record writes it.
func (st *featureStart) appendReseated(t *testing.T, id, seats string) {
	t.Helper()
	if err := st.o.Manifest.Append(ManifestEvent{OrgID: "auth-core", Event: EventReseated, Details: "id=" + id + " seats=" + seats}); err != nil {
		t.Fatal(err)
	}
}

// TestPlanReseat_FinishesAReseatThatWasCutOff covers P9 (AC-20 at this
// layer): a seat whose latest state event is a `stopped` with reseat=<id>,
// where the org's current life holds no `reseated` of that id, is a target
// of a reseat with no --seat. It is not active, so the reseat starts it and
// does not stop it, in the normal order, and running the reseat again
// finishes the one that was cut off. When the cut-off reseat named its seats
// with --seat, the same --seat values start them and leave the leader
// running, which a reseat with no --seat would stop and start too. A
// `reseated` of that id, which a reseat that got to its record wrote, takes
// the seat out (the refusal of an org with no active seat applies then), and
// so does a later stop by hand, which carries no note.
func TestPlanReseat_FinishesAReseatThatWasCutOff(t *testing.T) {
	// newLeaderAndReviewer is auth-core with its leader and its reviewer,
	// both stopped by the cut-off reseat, the reviewer first, as Reseat stops
	// them. herdr closes the org's workspace with its last pane, as the real
	// one does.
	newLeaderAndReviewer := func(t *testing.T) *featureStart {
		st := newFeatureStart(t)
		st.h.closeRemoves = true
		st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
		st.spawnSeat(t, "reviewer", "reviewer", "")
		st.stopSeatByReseat(t, "reviewer", cutOffReseatID)
		st.stopSeatByReseat(t, LeaderIdentity, cutOffReseatID)
		st.h.workspaceID = "ws-2"
		return st
	}

	t.Run("the leader and the reviewer it stopped", func(t *testing.T) {
		st := newLeaderAndReviewer(t)
		plan := st.mustPlanReseat(t, st.reseatParams())
		if len(plan.Stop) != 0 || !slices.Equal(seatIDs(plan.Start), []string{LeaderIdentity, "reviewer"}) {
			t.Fatalf("stop %q start %q; want no stop, and the leader started and then the reviewer",
				seatIDs(plan.Stop), seatIDs(plan.Start))
		}
		for _, seat := range plan.Start {
			if seat.Active || seat.Latest.Event != EventStopped || !strings.HasSuffix(seat.Latest.Details, " reseat="+cutOffReseatID) {
				t.Errorf("seat %s: active %v, latest %s %q; want the stopped of the cut-off reseat", seat.SeatID, seat.Active,
					seat.Latest.Event, seat.Latest.Details)
			}
		}
		if plan.Start[0].Start == nil || plan.Start[1].Spawn == nil || plan.Start[1].Spawn.Role != "reviewer" {
			t.Errorf("leader start %+v, reviewer spawn %+v; want the leader through StartFeature and the reviewer through Spawn",
				plan.Start[0].Start, plan.Start[1].Spawn)
		}

		from := len(mustReadEvents(t, st.o))
		res := st.mustReseat(t, st.reseatParams())
		added := mustReadEvents(t, st.o)[from:]
		if got := eventSeats(added, EventSpawned); !slices.Equal(got, []string{LeaderIdentity, "reviewer"}) || len(eventsNamed(added, EventStopped)) != 0 {
			t.Errorf("spawned %q stopped %q; want the leader and then the reviewer started, and nothing stopped",
				got, eventSeats(added, EventStopped))
		}
		assertReseated(t, added, "id="+res.Plan.ID+" seats=leader:started,reviewer:started")
		if res.Retry != "" || res.Decision != "" {
			t.Errorf("retry %q decision %q; want none", res.Retry, res.Decision)
		}
	})

	t.Run("a reseated of that id is recorded", func(t *testing.T) {
		st := newLeaderAndReviewer(t)
		st.appendReseated(t, cutOffReseatID, "leader:stopped,reviewer:stopped")
		st.assertReseatRefused(t, st.reseatParams(),
			`org_id "auth-core" has no active seat since its last disband and no seat stopped by a reseat that was cut off`,
			"its stopped seats are leader, reviewer: to start them again, run ralph org reseat --org-id auth-core --seat leader --seat reviewer")
	})

	t.Run("seats it stopped while the leader runs", func(t *testing.T) {
		// The reviewer was stopped by the cut-off reseat, the implementer by
		// another reseat that recorded `reseated`: only the reviewer is added
		// to the active leader.
		st := newReseatOrg(t)
		st.stopSeatByReseat(t, "reviewer", cutOffReseatID)
		st.stopSeatByReseat(t, implementerSeatID, "20261112T000000-0001")
		st.appendReseated(t, "20261112T000000-0001", "implementer:stopped")
		plan := st.mustPlanReseat(t, st.reseatParams())
		if !slices.Equal(seatIDs(plan.Stop), []string{LeaderIdentity}) || !slices.Equal(seatIDs(plan.Start), []string{LeaderIdentity, "reviewer"}) {
			t.Fatalf("stop %q start %q; want the leader stopped, and the leader and then the reviewer started",
				seatIDs(plan.Stop), seatIDs(plan.Start))
		}
		if plan.Start[1].Active {
			t.Errorf("the reviewer is active in the plan; want it started only")
		}
		// The cut-off reseat was `--seat reviewer`: the same command again
		// starts the reviewer and stops nothing.
		plan = st.mustPlanReseat(t, st.reseatParams("reviewer"))
		if len(plan.Stop) != 0 || !slices.Equal(seatIDs(plan.Start), []string{"reviewer"}) {
			t.Errorf("--seat reviewer: stop %q start %q; want no stop, and the reviewer started", seatIDs(plan.Stop), seatIDs(plan.Start))
		}
	})

	t.Run("a finished reseat's seat stopped later by hand", func(t *testing.T) {
		st := newReseatOrg(t)
		st.mustReseat(t, st.reseatParams("reviewer"))
		st.stopSeat(t, "reviewer")
		plan := st.mustPlanReseat(t, st.reseatParams())
		if !slices.Equal(seatIDs(plan.Start), []string{LeaderIdentity, implementerSeatID}) {
			t.Fatalf("start %q; want the leader and the implementer, not the reviewer stopped by hand", seatIDs(plan.Start))
		}
	})
}

// TestReseatNoteAndReseatedID pins how a reseat id is read back (P9): from
// the last token of a stopped's Details, reseat=<id>, and from the first
// token of a reseated's Details, id=<id>; a token elsewhere does not count.
func TestReseatNoteAndReseatedID(t *testing.T) {
	for _, tc := range []struct{ details, want string }{
		{"ctrl_c=ok pane=closed leave=ok reseat=" + cutOffReseatID, cutOffReseatID},
		{"reseat=" + cutOffReseatID, cutOffReseatID},
		{"reseat=" + cutOffReseatID + " ctrl_c=ok", ""},
		{"ctrl_c=ok pane=closed leave=ok", ""},
		{"", ""},
	} {
		if got := reseatNoteID(tc.details); got != tc.want {
			t.Errorf("reseatNoteID(%q) = %q; want %q", tc.details, got, tc.want)
		}
	}
	for _, tc := range []struct{ details, want string }{
		{"id=" + cutOffReseatID + " seats=leader:started", cutOffReseatID},
		{"seats=leader:started id=" + cutOffReseatID, ""},
		{"", ""},
	} {
		if got := reseatedID(tc.details); got != tc.want {
			t.Errorf("reseatedID(%q) = %q; want %q", tc.details, got, tc.want)
		}
	}
}

// TestPlanReseat_RefusesACustomRoleSeat covers P10 (AC-21): a seat whose
// role has no role prompt template (spawned with --role scribe --prompt) is
// refused as a target, with no --seat, with --seat and in a dry run, before
// anything is stopped, with the stop and the spawn that start it again by
// hand and the advice to name the other seats. The spawn names the driver
// and model the ledger recorded and the --scope reseat gives the seats, so
// the default autonomous mode does not refuse it, and the stop and the spawn
// carry the --config the reseat was given, if any. Once it is stopped, a
// reseat with no --seat leaves it out, and --seat names it with the spawn
// alone. A seat that ran the CLI's own default model (model "") gets a spawn
// with no --model.
func TestPlanReseat_RefusesACustomRoleSeat(t *testing.T) {
	st := newReseatOrg(t)
	st.o.Config.MaxSeats = 4
	if r := st.o.Spawn(SpawnParams{
		OrgID: "auth-core", SeatID: "scribe", Role: "scribe", Driver: "claude", Model: "haiku",
		Cwd: st.worktree("auth-core"), Scope: reseatScope, Prompt: "Keep the notes.", TimeoutMS: 5000,
	}); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn scribe: %+v", r)
	}
	head := `seat "scribe" would not start again: its role "scribe" has no role prompt template, and the ledger does not hold ` +
		`the --prompt it was spawned with, so reseat would start it with no instructions: `
	spawn := "spawn it again with ralph org spawn --org-id auth-core --id scribe --role scribe --driver claude --model haiku " +
		"--cwd " + ShellWord(st.worktree("auth-core")) + " --scope " + ShellWord(reseatScope) + " --prompt <its prompt> by hand, " +
		"and to reseat the org's other seats, name them with --seat and leave scribe out"
	want := head + "stop it with ralph org stop --org-id auth-core --seat scribe and " + spawn
	st.assertReseatRefused(t, st.reseatParams(), want)
	st.assertReseatRefused(t, st.reseatParams("reviewer", "scribe"), want)
	dry := st.reseatParams()
	dry.DryRun = true
	st.assertReseatRefused(t, dry, want)
	withConfig := st.reseatParams()
	withConfig.HintConfigPath = "/cfg/ralph.toml"
	st.assertReseatRefused(t, withConfig, "stop it with ralph org stop --org-id auth-core --seat scribe --config /cfg/ralph.toml and "+
		"spawn it again with ralph org spawn --org-id auth-core --id scribe --role scribe --driver claude --model haiku --cwd "+
		ShellWord(st.worktree("auth-core"))+" --scope "+ShellWord(reseatScope)+" --prompt <its prompt> --config /cfg/ralph.toml by hand")

	before := takeReseatLedger(t, st)
	if res := st.o.Reseat(st.reseatParams()); res.Err == nil || !strings.Contains(res.Err.Error(), want) || res.Seats != nil {
		t.Fatalf("Reseat = %v with seats %s; want the custom role's refusal", res.Err, reseatOutcomes(res))
	}
	if after := takeReseatLedger(t, st); after.snap != before.snap || after.manifest != before.manifest {
		t.Errorf("the refused reseat wrote or called something: %+v -> %+v", before, after)
	}
	if plan := st.mustPlanReseat(t, st.reseatParams(LeaderIdentity, implementerSeatID, "reviewer")); len(plan.Start) != 3 {
		t.Errorf("start %q; want the three seats named", seatIDs(plan.Start))
	}

	st.stopSeat(t, "scribe")
	if plan := st.mustPlanReseat(t, st.reseatParams()); !slices.Equal(seatIDs(plan.Start), []string{LeaderIdentity, implementerSeatID, "reviewer"}) {
		t.Errorf("start %q; want the active seats, not the stopped scribe", seatIDs(plan.Start))
	}
	err := st.checkReseatRefused(t, st.reseatParams("scribe"), head+spawn)
	if strings.Contains(err.Error(), "stop it with") {
		t.Errorf("refusal %q names a stop for a seat that is stopped", err)
	}

	st.o.Config.Permissions.CodexVerified = true
	if r := st.o.Spawn(SpawnParams{
		OrgID: "auth-core", SeatID: "tally", Role: "tally", Driver: "codex", Model: "",
		Cwd: st.worktree("auth-core"), Scope: reseatScope, Prompt: "Count the votes.", TimeoutMS: 5000,
	}); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn tally: %+v", r)
	}
	err = st.checkReseatRefused(t, st.reseatParams("tally"), "spawn it again with ralph org spawn --org-id auth-core --id tally "+
		"--role tally --driver codex --cwd "+ShellWord(st.worktree("auth-core"))+" --scope "+ShellWord(reseatScope)+
		" --prompt <its prompt> by hand")
	if strings.Contains(err.Error(), "--model") {
		t.Errorf("refusal %q names a --model for a seat that ran the CLI's own default model", err)
	}
}

// TestPlanReseat_RefusesASeatNotSpawnedInTheCurrentLife covers P7's other
// half: a --seat the org never spawned, one only rejected in its current
// life, and one spawned only before its last disband are refused, naming the
// seats it can start.
func TestPlanReseat_RefusesASeatNotSpawnedInTheCurrentLife(t *testing.T) {
	t.Run("never spawned", func(t *testing.T) {
		st := newReseatOrg(t)
		st.assertReseatRefused(t, st.reseatParams("nope"), `seat "nope" of org_id "auth-core" has not been spawned since the org's last disband`,
			"(the seats it can start again are: leader, implementer, reviewer): a seat the org never ran is spawned with ralph org spawn")
	})
	t.Run("only rejected", func(t *testing.T) {
		st := newReseatOrg(t)
		if err := st.o.Manifest.Append(ManifestEvent{OrgID: "auth-core", SeatID: "extra", Event: EventRejected, Role: "worker"}); err != nil {
			t.Fatal(err)
		}
		st.assertReseatRefused(t, st.reseatParams(implementerSeatID, "extra"), `seat "extra" of org_id "auth-core" has not been spawned`)
	})
	t.Run("spawned before the last disband", func(t *testing.T) {
		st := newReseatOrg(t)
		if d := st.o.Disband(DisbandParams{OrgID: "auth-core"}); len(d.Errs) != 0 || !d.Disbanded {
			t.Fatalf("disband auth-core: %+v", d)
		}
		st.h.workspaceID, st.h.nextPaneID = "ws-2", "pane-2"
		st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
		st.assertReseatRefused(t, st.reseatParams("reviewer"), `seat "reviewer" of org_id "auth-core" has not been spawned`,
			"(the seats it can start again are: leader)")
		if plan := st.mustPlanReseat(t, st.reseatParams()); !slices.Equal(seatIDs(plan.Start), []string{LeaderIdentity}) {
			t.Errorf("start %q; want only the leader of the current life", seatIDs(plan.Start))
		}
	})
}

// TestPlanReseat_RefusesBadFlags covers FR-1 (AC-11): --driver and --model
// only together and only with exactly one --seat, and a seat named once,
// each refused before the manifest is read.
func TestPlanReseat_RefusesBadFlags(t *testing.T) {
	for _, tc := range []struct {
		name          string
		seats         []string
		driver, model string
		want          string
	}{
		{"both with no --seat", nil, "claude", "opus", "so they need exactly one --seat (0 given)"},
		{"both with two seats", []string{implementerSeatID, "reviewer"}, "claude", "opus", "so they need exactly one --seat (2 given)"},
		{"--model with two seats", []string{implementerSeatID, "reviewer"}, "", "opus", "(2 given)"},
		{"--driver alone", []string{implementerSeatID}, "codex", "", "--driver and --model go together: pass both with --seat implementer"},
		{"--model alone", []string{implementerSeatID}, "", "opus", "--driver and --model go together"},
		{"a seat named twice", []string{"reviewer", implementerSeatID, "reviewer"}, "", "", "--seat reviewer is given twice: name each seat once"},
		{"a seat that is no seat_id", []string{"Bad Seat"}, "", "", `invalid seat_id "Bad Seat"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newReseatOrg(t)
			// A manifest that cannot be read: the flags are refused first.
			st.o.Manifest = NewManifestStoreAtPath(t.TempDir())
			p := st.reseatParams(tc.seats...)
			p.Driver, p.Model = tc.driver, tc.model
			_, err := st.o.PlanReseat(p)
			var refusal *reseatRefusal
			if !errors.As(err, &refusal) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("PlanReseat = %v; want a refusal with %q", err, tc.want)
			}
		})
	}
}

// TestPlanReseat_DriverAndModelForOneSeat covers D1 and P5: --driver and
// --model with one --seat start only that seat with them, noted as
// overridden, while the leader keeps running; a driver the spawn would refuse
// (codex, unverified in the fixture) is refused before anything is stopped.
func TestPlanReseat_DriverAndModelForOneSeat(t *testing.T) {
	st := newReseatOrg(t)
	p := st.reseatParams(implementerSeatID)
	p.Driver, p.Model = "claude", "sonnet"
	plan := st.mustPlanReseat(t, p)
	if !slices.Equal(seatIDs(plan.Stop), []string{implementerSeatID}) || !slices.Equal(seatIDs(plan.Start), []string{implementerSeatID}) {
		t.Fatalf("stop %q start %q; want the implementer only", seatIDs(plan.Stop), seatIDs(plan.Start))
	}
	seat := plan.Start[0]
	if !seat.Overridden || seat.Driver != "claude" || seat.Model != "sonnet" || seat.RecordedModel != "opus" || seat.Spawn == nil ||
		seat.Spawn.Model != "sonnet" || seat.Spawn.Role != implementerRole {
		t.Fatalf("implementer = %+v; want it overridden to claude sonnet from opus", seat)
	}

	p.Driver, p.Model = "codex", "gpt-x"
	st.assertReseatRefused(t, p, `seat "implementer" would not start again: `, "requires [org.permissions].codex_verified=true")
}

// TestPlanReseat_RefusedFromASeatPane covers FR-3 (AC-7): a reseat run from
// the herdr pane of an active seat of the org is refused; a pane of another
// org's seat, or of a stopped seat of this org, is not.
func TestPlanReseat_RefusedFromASeatPane(t *testing.T) {
	st := newReseatOrg(t)
	pane := eventOf(t, st.o, "auth-core", implementerSeatID, EventSpawned).PaneID
	if pane == "" {
		t.Fatal("the implementer has no pane on record")
	}
	st.o.Getenv = func(key string) string {
		if key == herdrPaneIDEnv {
			return pane
		}
		return ""
	}
	st.assertReseatRefused(t, st.reseatParams("reviewer"), "this command runs in the herdr pane "+pane+` of seat "implementer" of org_id "auth-core"`,
		"run ralph org reseat from outside the org's seats")

	st.stopSeat(t, implementerSeatID)
	st.mustPlanReseat(t, st.reseatParams("reviewer"))
	spawnSeatIn(t, st.o, st.h, "other", "seat-1", "ws-9", pane+"-other")
	st.o.Getenv = func(key string) string {
		if key == herdrPaneIDEnv {
			return pane + "-other"
		}
		return ""
	}
	st.mustPlanReseat(t, st.reseatParams())
}

// TestPlanReseat_RefusesAnOrgWithoutABinding covers FR-2 (AC-5): an org with
// no split plan binding in its current life, or an incomplete one, is
// refused, with the stop and spawn to use instead.
func TestPlanReseat_RefusesAnOrgWithoutABinding(t *testing.T) {
	const hint = "so ralph org reseat does not start its seats again: stop a seat with ralph org stop --org-id auth-core --seat <seat> " +
		"and start it again with ralph org spawn --org-id auth-core --id <seat> --role <role> (its leader with ralph org start)"
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, st *featureStart)
		want  string
	}{
		{"ralph org start <task>'s org", func(t *testing.T, st *featureStart) {
			spawnLeaderIn(t, st.o, st.h, "auth-core", "ws-1", "pane-1")
		}, "has no split plan binding since its last disband (it was not started with ralph org start --plan)"},
		{"a plain --reserve org", func(t *testing.T, st *featureStart) {
			spawnLeaderIn(t, st.o, st.h, "auth-core", "ws-1", "pane-1", "internal/auth/")
		}, "has no split plan binding since its last disband"},
		{"an org with no record", func(*testing.T, *featureStart) {}, "has no split plan binding since its last disband"},
		{"an incomplete binding", func(t *testing.T, st *featureStart) {
			if err := st.o.Manifest.Append(boundReserveEvent("auth-core", &FeatureBinding{Split: "auth-split", Feature: "auth-core", Worktree: "/w"}, "docs/")); err != nil {
				t.Fatal(err)
			}
		}, "has an incomplete split plan binding since its last disband (feature \"auth-core\" of split plan \"auth-split\""},
		{"a feature org disbanded", func(t *testing.T, st *featureStart) {
			st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
			if d := st.o.Disband(DisbandParams{OrgID: "auth-core"}); len(d.Errs) != 0 || !d.Disbanded {
				t.Fatalf("disband auth-core: %+v", d)
			}
		}, "has no split plan binding since its last disband"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newFeatureStart(t)
			tc.setup(t, st)
			st.assertReseatRefused(t, st.reseatParams(), `org_id "auth-core" `+tc.want, hint)
		})
	}

	// Given --config, the stop and the spawn it names repeat it.
	st := newFeatureStart(t)
	p := st.reseatParams()
	p.HintConfigPath = "/cfg dir/ralph.toml"
	st.assertReseatRefused(t, p, "stop a seat with ralph org stop --org-id auth-core --seat <seat> --config '/cfg dir/ralph.toml' "+
		"and start it again with ralph org spawn --org-id auth-core --id <seat> --role <role> --config '/cfg dir/ralph.toml' (its leader")
}

// TestPlanReseat_RefusesWhatTheStartsWouldRefuse covers FR-4 (AC-4): a
// changed split plan, its own section or preamble changed and approved
// again, the split plan gone, another approval mode, no watch, no herdr
// server, the worktree record missing or not the feature's, a regate on the
// org, and an implementer whose approved plan changed since are each refused
// before anything is stopped, with nothing written or called, and the gates
// asked only when every other check passed. The binding's checks refuse a
// reseat of one seat other than the leader too.
func TestPlanReseat_RefusesWhatTheStartsWouldRefuse(t *testing.T) {
	approved := approveSplitPlan(t, splitPlanFixture)
	changedDocs := strings.Replace(approved, fixtureDocsBody, "Objective: document it in full.", 1)
	for _, tc := range []struct {
		name      string
		seats     []string
		setup     func(t *testing.T, st *featureStart)
		want      []string // %WT% stands for the feature worktree
		watchErr  error    // the watch gate's refusal
		serverErr error    // the herdr server gate's
		is        error    // what errors.Is finds in the refusal, when set
	}{
		{name: "the split plan changed after its approval", setup: func(t *testing.T, st *featureStart) {
			st.writePlan(t, "auth-split", changedDocs)
		}, want: []string{`seat "leader" would not start again: `, "changed after its approval"}},
		{name: "the split plan changed, one other seat", seats: []string{"reviewer"}, setup: func(t *testing.T, st *featureStart) {
			st.writePlan(t, "auth-split", changedDocs)
		}, want: []string{"changed after its approval"}},
		{name: "its own section changed and approved again", setup: func(t *testing.T, st *featureStart) {
			st.writePlan(t, "auth-split", approveSplitPlan(t, editedSplitPlan(t, fixtureCoreBody, "Objective: add the token store v2.")))
		}, want: []string{`since the feature's section (### auth-core) or the preamble of split plan "auth-split" changed after org_id "auth-core" was started`}},
		{name: "the preamble changed and approved again, one other seat", seats: []string{implementerSeatID}, setup: func(t *testing.T, st *featureStart) {
			st.writePlan(t, "auth-split", approveSplitPlan(t, editedSplitPlan(t, fixturePreambleText, "認証まわりを 3 つの機能に分ける。")))
		}, want: []string{`or the preamble of split plan "auth-split" changed`}},
		{name: "the split plan is gone", setup: func(t *testing.T, st *featureStart) {
			if err := os.Remove(st.plan); err != nil {
				t.Fatal(err)
			}
		}, want: []string{"a split plan is a file <id>.md directly in"}},
		{name: "another approval mode in the config", setup: func(_ *testing.T, st *featureStart) {
			st.o.Config.Approval.Mode = config.ApprovalModeSplit
		}, want: []string{`org_id "auth-core" is bound to feature "auth-core"`}},
		{name: "the worktree record is missing", setup: func(_ *testing.T, st *featureStart) {
			delete(st.wt.records, "org-auth-core")
		}, want: []string{`the ralph-worktree.sh record org-auth-core of org_id "auth-core"'s feature worktree %WT% is missing, and reseat makes no worktree`}},
		{name: "the worktree record is missing, one other seat", seats: []string{"reviewer"}, setup: func(_ *testing.T, st *featureStart) {
			delete(st.wt.records, "org-auth-core")
		}, want: []string{"is missing, and reseat makes no worktree"}},
		{name: "the worktree has another branch checked out", setup: func(_ *testing.T, st *featureStart) {
			st.wt.checkouts[st.worktree("auth-core")] = "main"
		}, want: []string{`names %WT%, which has branch "main" checked out, not "fix/auth-core"`}},
		{name: "a regate holds the org's marker", seats: []string{implementerSeatID}, setup: func(t *testing.T, st *featureStart) {
			lock, err := acquireHeldFileLock(st.stateDir, regateLockFile, "regate", 0)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = lock.release() })
			if err := writeRegateMarker(lock.f, &regateMarker{OrgID: "auth-core", Branch: "fix/auth-core", Worktree: "/w", PID: 1}); err != nil {
				t.Fatal(err)
			}
		}, want: []string{`org_id "auth-core" is not spawned while ralph org regate runs on its branch`}},
		{name: "the implementer's approved plan changed since", setup: func(t *testing.T, st *featureStart) {
			writePlan(t, st.worktree("auth-core"), testPlanRel, testPlanDraft)
		}, want: []string{`seat "implementer" would not start again: `}, is: ErrPlanNotApproved},
		{name: "no watch accepts work", setup: func(*testing.T, *featureStart) {}, want: []string{"no ralph org watch --all is running"},
			watchErr: refusingWatchGate().err, is: ErrNoRunningWatch},
		{name: "the herdr server cannot be started", setup: func(*testing.T, *featureStart) {}, want: []string{errHerdrServerDown.Error()},
			serverErr: errHerdrServerDown, is: errHerdrServerDown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newReseatOrg(t)
			watch, server := &stubWatchGate{err: tc.watchErr}, &stubHerdrServerGate{err: tc.serverErr}
			st.o.WatchGate, st.o.HerdrServerGate = watch.gate, server.gate
			tc.setup(t, st)
			want := slices.Clone(tc.want)
			for i := range want {
				want[i] = strings.ReplaceAll(want[i], "%WT%", st.worktree("auth-core"))
			}
			err := st.checkReseatRefused(t, st.reseatParams(tc.seats...), want...)
			if tc.is != nil && !errors.Is(err, tc.is) {
				t.Errorf("refusal %v is not %v", err, tc.is)
			}
			// The gates come last: neither is asked when another check
			// refused, and the herdr server gate only after the watch gate
			// passed.
			wantWatch, wantServer := 0, 0
			switch {
			case tc.watchErr != nil:
				wantWatch = 1
			case tc.serverErr != nil:
				wantWatch, wantServer = 1, 1
			}
			if watch.calls != wantWatch || server.calls != wantServer {
				t.Errorf("gate calls: watch %d, herdr server %d; want %d and %d", watch.calls, server.calls, wantWatch, wantServer)
			}
		})
	}
}

// TestPlanReseat_GatesRunOnceAfterTheChecks covers P1's gates: a reseat that
// passes every check asks the watch gate and then the herdr server gate once
// each, after the other checks.
func TestPlanReseat_GatesRunOnceAfterTheChecks(t *testing.T) {
	st := newReseatOrg(t)
	var log []string
	st.o.WatchGate = func() error { log = append(log, "watch"); return nil }
	st.o.HerdrServerGate = (&stubHerdrServerGate{log: &log}).gate
	st.mustPlanReseat(t, st.reseatParams())
	if !slices.Equal(log, []string{"watch", "herdr_server_gate"}) {
		t.Fatalf("gate calls %q; want the watch gate, then the herdr server gate, once each", log)
	}
}

// TestPlanReseat_LeaderNotATarget covers D3's last condition: a reseat of a
// seat other than the leader goes on only while the leader runs, and stops
// and starts neither the leader nor the seats not named.
func TestPlanReseat_LeaderNotATarget(t *testing.T) {
	st := newReseatOrg(t)
	plan := st.mustPlanReseat(t, st.reseatParams("reviewer"))
	if !slices.Equal(seatIDs(plan.Stop), []string{"reviewer"}) || !slices.Equal(seatIDs(plan.Start), []string{"reviewer"}) {
		t.Fatalf("stop %q start %q; want the reviewer only", seatIDs(plan.Stop), seatIDs(plan.Start))
	}

	st.stopSeat(t, LeaderIdentity)
	st.assertReseatRefused(t, st.reseatParams("reviewer"), `seat "reviewer" would not start again: `,
		`org_id "auth-core" was started with ralph org start --plan and its leader seat is not running`)
	plan = st.mustPlanReseat(t, st.reseatParams("reviewer", LeaderIdentity))
	if !slices.Equal(seatIDs(plan.Stop), []string{"reviewer"}) || !slices.Equal(seatIDs(plan.Start), []string{LeaderIdentity, "reviewer"}) {
		t.Fatalf("stop %q start %q; want the reviewer stopped and the leader started first", seatIDs(plan.Stop), seatIDs(plan.Start))
	}
}

// TestPlanReseat_LeaderOnly covers an org whose only seat is its leader: it
// is stopped and started again, and max_seats counts it once.
func TestPlanReseat_LeaderOnly(t *testing.T) {
	st := newFeatureStart(t)
	st.o.Config.MaxSeats = 1
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	plan := st.mustPlanReseat(t, st.reseatParams())
	if !slices.Equal(seatIDs(plan.Stop), []string{LeaderIdentity}) || !slices.Equal(seatIDs(plan.Start), []string{LeaderIdentity}) {
		t.Fatalf("stop %q start %q; want the leader only", seatIDs(plan.Stop), seatIDs(plan.Start))
	}
}

// TestPlanReseat_DryRunStartsNoWatchOrServer covers P8 (AC-19 at this
// layer): a dry run reads whether a watch accepts work and whether the herdr
// server runs instead of running the two gates, so with neither running it
// starts neither and writes nothing, and its plan says a real reseat would
// start both. A watch that accepts work and a running server need no note; a
// stale watch, and a server whose status the probe cannot give, are refused
// as the gates would refuse them. With no gate set, nothing is read.
func TestPlanReseat_DryRunStartsNoWatchOrServer(t *testing.T) {
	running := WatchLiveness{State: WatchLivenessRunning, Mark: &WatchAlive{PID: 42}}
	for _, tc := range []struct {
		name                 string
		watch                func() WatchLiveness // nil: ProbeWatch on the fixture's ledger, where no watch runs
		server               func() (bool, error)
		noGates              bool
		wantWatch, wantServe bool
		want                 string // a refusal's words; "" for none
	}{
		{"neither runs", nil, func() (bool, error) { return false, nil }, false, true, true, ""},
		{"both run", func() WatchLiveness { return running }, func() (bool, error) { return true, nil }, false, false, false, ""},
		{"a stale watch", func() WatchLiveness { return WatchLiveness{State: WatchLivenessStale} }, nil, false, false, false,
			"has not refreshed its liveness mark"},
		{"a server status that cannot be read", func() WatchLiveness { return running }, func() (bool, error) { return false, errHerdrServerDown },
			false, false, false, errHerdrServerDown.Error()},
		{"no gates", nil, func() (bool, error) { return false, nil }, true, false, false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newReseatOrg(t)
			watchGate, serverGate := &stubWatchGate{}, &stubHerdrServerGate{}
			if !tc.noGates {
				st.o.WatchGate, st.o.HerdrServerGate = watchGate.gate, serverGate.gate
			}
			p := st.reseatParams()
			p.DryRun, p.WatchProbe, p.HerdrServerProbe = true, tc.watch, tc.server
			if tc.want != "" {
				st.assertReseatRefused(t, p, tc.want)
			} else {
				plan := st.mustPlanReseat(t, p)
				if !plan.DryRun || plan.StartsWatch != tc.wantWatch || plan.StartsHerdrServer != tc.wantServe || len(plan.Start) != 3 {
					t.Errorf("dry run plan: dry-run %v, starts watch %v, starts server %v, %d starts; want true, %v, %v, 3",
						plan.DryRun, plan.StartsWatch, plan.StartsHerdrServer, len(plan.Start), tc.wantWatch, tc.wantServe)
				}
			}
			if watchGate.calls != 0 || serverGate.calls != 0 {
				t.Errorf("a dry run ran the gates: watch %d, herdr server %d; want none", watchGate.calls, serverGate.calls)
			}
		})
	}
}

// TestStartFeature_InputRefusalsComeBeforeTheGates pins the order
// StartFeature kept when its checks were split into startFeatureInput and
// checkLedger: a split plan, a leader spawn input (checkSpawnInput) or a
// reviewer value that is refused is refused before either gate runs, as a
// rejection, with the org_id in the result once the split plan named it, and
// a herdr server gate that fails after a passing watch gate is a failure.
// The envelope's checks (a model that is not one word) are spawnPrecheckErr's
// and come after the gates (TestStartFeature_WatchGate_ARefusalMakesNoWorktree).
func TestStartFeature_InputRefusalsComeBeforeTheGates(t *testing.T) {
	for _, tc := range []struct {
		name      string
		change    func(st *featureStart, p *StartFeatureParams)
		want      string
		wantOrgID string
	}{
		{"a split plan not approved", func(st *featureStart, _ *StartFeatureParams) {
			st.writePlan(t, "auth-split", approveSplitPlan(t, strings.Replace(splitPlanFixture, "- Status: Approved", "- Status: Draft", 1)))
		}, "not approved", ""},
		{"an approval mode the binding cannot hold", func(st *featureStart, _ *StartFeatureParams) { st.o.Config.Approval.Mode = "bogus" },
			`the split plan feature's approval mode "bogus" is not`, "auth-core"},
		{"a reviewer value that splits the shell", func(_ *featureStart, p *StartFeatureParams) {
			p.Reviewer = config.ReviewerConfig{CodexModel: "gpt x"}
		}, "the leader's task copies the [reviewer] values into its shell commands", "auth-core"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newFeatureStart(t)
			watch, server := refusingWatchGate(), &stubHerdrServerGate{err: errHerdrServerDown}
			st.o.WatchGate, st.o.HerdrServerGate = watch.gate, server.gate
			p := st.params("auth-core")
			tc.change(st, &p)
			res := st.o.StartFeature(p)
			assertStartRefused(t, res, tc.want)
			if res.OrgID != tc.wantOrgID || watch.calls != 0 || server.calls != 0 {
				t.Errorf("org_id %q, gate calls %d and %d; want %q and no gate call", res.OrgID, watch.calls, server.calls, tc.wantOrgID)
			}
		})
	}

	st := newFeatureStart(t)
	server := &stubHerdrServerGate{err: errHerdrServerDown}
	st.o.WatchGate, st.o.HerdrServerGate = (&stubWatchGate{}).gate, server.gate
	res := st.o.StartFeature(st.params("auth-core"))
	if res.Spawn.Outcome != SpawnOutcomeFailed || !errors.Is(res.Spawn.Err, errHerdrServerDown) || res.OrgID != "auth-core" {
		t.Fatalf("start with a failing herdr server gate = %+v (org_id %q); want failed with the gate's error", res.Spawn, res.OrgID)
	}
	st.wantCalls(t)
}

// Reseat at the org layer (slice S2: AC-1 to AC-3, AC-8, AC-10, AC-15,
// AC-17 and AC-18), on the same fixture: the stops and starts run on the
// fake herdr and agmsg, and the leader's start on the fake worktrees.

// mustReseat runs Reseat with p and fails unless it ended with no error and
// every target seat started again.
func (st *featureStart) mustReseat(t *testing.T, p ReseatParams) ReseatResult {
	t.Helper()
	res := st.o.Reseat(p)
	if res.Err != nil {
		t.Fatalf("Reseat(%+v): %v", p, res.Err)
	}
	for _, s := range res.Seats {
		if s.Outcome != ReseatSeatStarted {
			t.Fatalf("Reseat(%+v): seats %s; want every seat started", p, reseatOutcomes(res))
		}
	}
	return res
}

// retryParams is the reseat of auth-core that the printed command retry
// runs: its --seat, --driver, --model and --config words.
func (st *featureStart) retryParams(t *testing.T, retry string) ReseatParams {
	t.Helper()
	rest, ok := strings.CutPrefix(retry, "ralph org reseat --org-id auth-core")
	fields := strings.Fields(rest)
	if !ok || len(fields)%2 != 0 {
		t.Fatalf("retry %q is not a reseat of auth-core with flags and values", retry)
	}
	p := st.reseatParams()
	for i := 0; i < len(fields); i += 2 {
		switch fields[i] {
		case "--seat":
			p.Seats = append(p.Seats, fields[i+1])
		case "--driver":
			p.Driver = fields[i+1]
		case "--model":
			p.Model = fields[i+1]
		case "--config":
			p.HintConfigPath = fields[i+1]
		default:
			t.Fatalf("retry %q has the flag %q", retry, fields[i])
		}
	}
	return p
}

// reseatOutcomes is res's seats as <seat>:<outcome>, comma-separated, in
// order.
func reseatOutcomes(res ReseatResult) string {
	parts := make([]string, len(res.Seats))
	for i, s := range res.Seats {
		parts[i] = s.SeatID + ":" + string(s.Outcome)
	}
	return strings.Join(parts, ",")
}

// eventSeats is the seat id of each event of events named name, in order
// (eventsNamed, verbs_test.go).
func eventSeats(events []ManifestEvent, name string) []string {
	var ids []string
	for _, ev := range eventsNamed(events, name) {
		ids = append(ids, ev.SeatID)
	}
	return ids
}

// assertReseated checks that events hold exactly one `reseated`, of
// auth-core and no seat, with details.
func assertReseated(t *testing.T, events []ManifestEvent, details string) {
	t.Helper()
	got := eventsNamed(events, EventReseated)
	if len(got) != 1 || got[0].OrgID != "auth-core" || got[0].SeatID != "" || got[0].Details != details {
		t.Fatalf("reseated events %+v; want one of auth-core with Details %q", got, details)
	}
}

// TestReseat_StopsAndStartsEverySeatInOrder covers AC-1 (issue #312's
// acceptance criterion 2): the leader, implementer and reviewer of a feature
// org with a per-plan approval are stopped, the others before the leader, and
// started again leader first, with the role, driver and model each had and
// the feature worktree as cwd. No scope_reserved is added, the implementer's
// spawn and a TASK naming the approved plan still pass the approval check,
// and reseat.lock is released, empty.
func TestReseat_StopsAndStartsEverySeatInOrder(t *testing.T) {
	st := newReseatOrg(t)
	st.h.closeRemoves = true
	from := len(mustReadEvents(t, st.o))
	reserved := countString(eventNames(t, st.o), EventScopeReserved)

	res := st.mustReseat(t, st.reseatParams())
	if got := reseatOutcomes(res); got != "leader:started,implementer:started,reviewer:started" {
		t.Fatalf("seats %s; want leader, implementer and reviewer started", got)
	}
	if res.Retry != "" || res.Decision != "" || res.Changed != "" {
		t.Errorf("retry %q decision %q changed %q; want none, the leader being restarted too", res.Retry, res.Decision, res.Changed)
	}
	events := mustReadEvents(t, st.o)
	added := events[from:]
	if got := eventSeats(added, EventStopped); !slices.Equal(got, []string{"reviewer", implementerSeatID, LeaderIdentity}) {
		t.Errorf("stopped %q; want reviewer, implementer, leader", got)
	}
	lastStop := slices.IndexFunc(added, func(ev ManifestEvent) bool { return ev.Event == EventStopped && ev.SeatID == LeaderIdentity })
	if first := slices.IndexFunc(added, func(ev ManifestEvent) bool { return ev.Event == EventSpawnStarted }); first < lastStop {
		t.Errorf("a seat started (event %d) before the last stop (event %d)", first, lastStop)
	}
	for _, name := range []string{EventSpawnStarted, EventSpawned} {
		if got := eventSeats(added, name); !slices.Equal(got, []string{LeaderIdentity, implementerSeatID, "reviewer"}) {
			t.Errorf("%s %q; want leader, implementer, reviewer", name, got)
		}
	}
	for _, want := range []struct{ id, role, model string }{
		{LeaderIdentity, LeaderIdentity, "sonnet"}, {implementerSeatID, implementerRole, "opus"}, {"reviewer", "reviewer", ""},
	} {
		ev := eventsNamed(added, EventSpawned)[slices.Index(eventSeats(added, EventSpawned), want.id)]
		if ev.Role != want.role || ev.Driver != "claude" || ev.Model != want.model || ev.Worktree != st.worktree("auth-core") {
			t.Errorf("spawned %s = %+v; want role %s on claude %q in the feature worktree", want.id, ev, want.role, want.model)
		}
	}
	if n := countString(eventNames(t, st.o), EventScopeReserved); n != reserved {
		t.Errorf("scope_reserved: %d after the reseat, %d before; want no new one", n, reserved)
	}

	implementer := SpawnParams{OrgID: "auth-core", SeatID: implementerSeatID, Role: implementerRole}
	if err := spawnFeatureSeatErr(implementer, events); err != nil {
		t.Errorf("the implementer's spawn after the reseat: %v; want the approval of the org's life to pass", err)
	}
	ref := PlanRef{Path: testPlanRel, Digest: PlanDigest([]byte(testPlanDraft))}
	if err := checkTaskPlanApproved(events, "auth-core", ActiveFeature(events, "auth-core"), planTask(ref)); err != nil {
		t.Errorf("a TASK naming the approved plan after the reseat: %v; want it to pass", err)
	}

	if data := fileOrAbsent(t, filepath.Join(st.stateDir, reseatLockFile)); data != "" {
		t.Errorf("reseat.lock holds %q after the reseat; want it empty", data)
	}
	lock, err := acquireHeldFileLock(st.stateDir, reseatLockFile, "reseat", 0)
	if err != nil {
		t.Fatalf("reseat.lock is still held after the reseat: %v", err)
	}
	_ = lock.release()
}

// TestReseat_RecordsItsIDOnEveryStopAndStart covers AC-15 (FR-8, P3, P4):
// every stopped, spawn_started and spawned of the reseat ends with
// reseat=<id>, the stopped after Stop's own notes and the spawned after
// permission_mode, and one org-level `reseated` with each seat's outcome is
// the reseat's last event. No event before the reseat carries a note.
func TestReseat_RecordsItsIDOnEveryStopAndStart(t *testing.T) {
	st := newReseatOrg(t)
	from := len(mustReadEvents(t, st.o))
	res := st.mustReseat(t, st.reseatParams())
	note := "reseat=" + res.Plan.ID
	events := mustReadEvents(t, st.o)
	added := events[from:]

	spawned := regexp.MustCompile(` permission_mode=\S+ ` + regexp.QuoteMeta(note) + `$`)
	counts := make(map[string]int)
	for _, ev := range added {
		counts[ev.Event]++
		switch ev.Event {
		case EventStopped:
			if ev.Details != "ctrl_c=ok pane=closed leave=ok "+note {
				t.Errorf("stopped %s: Details %q; want Stop's notes and then %s", ev.SeatID, ev.Details, note)
			}
		case EventSpawnStarted:
			if ev.Details != note {
				t.Errorf("spawn_started %s: Details %q; want %s alone", ev.SeatID, ev.Details, note)
			}
		case EventSpawned:
			if !spawned.MatchString(ev.Details) {
				t.Errorf("spawned %s: Details %q; want permission_mode and then %s at the end", ev.SeatID, ev.Details, note)
			}
		}
	}
	if counts[EventStopped] != 3 || counts[EventSpawnStarted] != 3 || counts[EventSpawned] != 3 || counts[EventReseated] != 1 {
		t.Errorf("event counts %v; want 3 stopped, 3 spawn_started, 3 spawned and 1 reseated", counts)
	}
	assertReseated(t, added, "id="+res.Plan.ID+" seats=leader:started,implementer:started,reviewer:started")
	if last := added[len(added)-1]; last.Event != EventReseated {
		t.Errorf("the reseat's last event is %s; want reseated", last.Event)
	}
	for _, ev := range events[:from] {
		if strings.Contains(ev.Details, "reseat=") {
			t.Errorf("%s of %s before the reseat has a note: %q", ev.Event, ev.SeatID, ev.Details)
		}
	}
}

// TestReseat_LeaderIsTheFirstPaneOfANewWorkspace covers AC-2 (D3): once
// every seat is stopped, herdr has closed the org's workspace with its last
// pane, so the leader's start records that workspace closed and makes a new
// one whose root pane is the leader's; the implementer splits it to the
// right and the reviewer splits the implementer's pane down. No tab is made.
func TestReseat_LeaderIsTheFirstPaneOfANewWorkspace(t *testing.T) {
	st := newReseatOrg(t)
	st.h.closeRemoves = true
	st.h.workspaceID = "ws-2"
	workspaces, tabs, splits := len(st.h.workspaceCreateCwds), len(st.h.tabCreateCwds), len(st.h.paneSplitCalls)
	st.mustReseat(t, st.reseatParams())

	if got := st.h.workspaceCreateCwds[workspaces:]; !slices.Equal(got, []string{st.worktree("auth-core")}) || len(st.h.tabCreateCwds) != tabs {
		t.Fatalf("workspace creates in %q and %d tab creates; want one workspace in the feature worktree and no tab",
			got, len(st.h.tabCreateCwds)-tabs)
	}
	root := fakeRootPaneID("ws-2")
	leader := eventOf(t, st.o, "auth-core", LeaderIdentity, EventSpawned)
	implementer := eventOf(t, st.o, "auth-core", implementerSeatID, EventSpawned)
	if leader.PaneID != root || st.h.paneWorkspaces[root] != "ws-2" {
		t.Errorf("the leader runs in pane %q; want %s, the root pane of the new workspace ws-2", leader.PaneID, root)
	}
	got := st.h.paneSplitCalls[splits:]
	if len(got) != 2 || got[0].paneID != root || got[0].direction != driver.HerdrSplitRight ||
		got[1].paneID != implementer.PaneID || got[1].direction != driver.HerdrSplitDown {
		t.Errorf("splits %+v; want the leader's pane split right, then the implementer's %s split down", got, implementer.PaneID)
	}
	closed := eventOf(t, st.o, "auth-core", "", EventOrgWorkspaceClosed)
	if closed.PaneID != "ws-1" || closed.Details != "not found at spawn" {
		t.Errorf("org_workspace_closed %+v; want ws-1, not found at spawn", closed)
	}
}

// TestReseat_OneSeatOnAnotherDriver covers AC-3 (D1, D5, P4): --seat
// implementer with --driver codex and --model starts only the implementer,
// on codex with that model; the leader and the reviewer get no event. The
// reseated records the change, and the result holds the DECISION for the
// leader with --state-dir as the ledger was named, a message ralph org send
// accepts. Nothing is left to retry.
func TestReseat_OneSeatOnAnotherDriver(t *testing.T) {
	st := newReseatOrg(t)
	st.o.Config.Permissions.CodexVerified = true
	st.o.InboxHintStateDir = "/state/org dir"
	from := len(mustReadEvents(t, st.o))
	p := st.reseatParams(implementerSeatID)
	p.Driver, p.Model = "codex", "gpt-x"

	res := st.mustReseat(t, p)
	if got := reseatOutcomes(res); got != "implementer:started" || res.Retry != "" {
		t.Fatalf("seats %s retry %q; want the implementer started and nothing to retry", got, res.Retry)
	}
	added := mustReadEvents(t, st.o)[from:]
	for _, ev := range added {
		if ev.SeatID == LeaderIdentity || ev.SeatID == "reviewer" {
			t.Errorf("the reseat of the implementer recorded %s of %s", ev.Event, ev.SeatID)
		}
	}
	impl := eventOf(t, st.o, "auth-core", implementerSeatID, EventSpawned)
	if impl.Driver != "codex" || impl.Model != "gpt-x" || impl.Role != implementerRole {
		t.Errorf("implementer spawned %+v; want the implementer role on codex gpt-x", impl)
	}
	assertReseated(t, added, "id="+res.Plan.ID+" seats=implementer:started changed=implementer:codex/gpt-x")

	text := "TYPE: DECISION\n\nralph org reseat が座席 implementer を止めて立て直しました(reseat=" + res.Plan.ID + ")。" +
		"立て直した座席は TASK を持たずに立っています。それぞれに今の TASK を送り直してください。"
	if got := ReseatDecisionText(res.Plan.ID, []string{implementerSeatID}); got != text {
		t.Errorf("ReseatDecisionText = %q; want %q", got, text)
	}
	if err := protocol.ValidateText(text, protocol.DefaultMaxBodyChars); err != nil {
		t.Errorf("the DECISION is not a valid message: %v", err)
	}
	if want := "ralph org send --org-id auth-core --to leader --state-dir '/state/org dir' --text '" + text + "'"; res.Decision != want {
		t.Errorf("Decision = %q; want %q", res.Decision, want)
	}
}

// TestReseat_StartsAStoppedSeatNamedWithSeat covers P7 and P4 for a reseat
// that stops nothing: --seat names a stopped reviewer, which is started
// again with the note; `reseated` is recorded all the same, and the
// DECISION is for that seat.
func TestReseat_StartsAStoppedSeatNamedWithSeat(t *testing.T) {
	st := newReseatOrg(t)
	st.stopSeat(t, "reviewer")
	from := len(mustReadEvents(t, st.o))
	res := st.mustReseat(t, st.reseatParams("reviewer"))
	added := mustReadEvents(t, st.o)[from:]
	if len(res.Plan.Stop) != 0 || len(eventsNamed(added, EventStopped)) != 0 || !slices.Equal(eventSeats(added, EventSpawned), []string{"reviewer"}) {
		t.Fatalf("stops %d, stopped %q, spawned %q; want the reviewer started and nothing stopped",
			len(res.Plan.Stop), eventSeats(added, EventStopped), eventSeats(added, EventSpawned))
	}
	assertReseated(t, added, "id="+res.Plan.ID+" seats=reviewer:started")
	if want := ReseatDecisionCommand("auth-core", ReseatDecisionText(res.Plan.ID, []string{"reviewer"}), ""); res.Decision != want {
		t.Errorf("Decision = %q; want %q", res.Decision, want)
	}
}

// TestReseat_SeatsWhosePanesAreGone covers AC-8: after the herdr server was
// started again, herdr knows none of the org's panes, so each stop records
// `pane=already closed` with no close call, and every seat starts again in a
// new workspace.
func TestReseat_SeatsWhosePanesAreGone(t *testing.T) {
	st := newReseatOrg(t)
	st.h.paneTabs, st.h.paneWorkspaces, st.h.paneLabels, st.h.tabLabels, st.h.workspaceLabels = nil, nil, nil, nil, nil
	st.h.workspaceID = "ws-2"
	closes := len(st.h.paneCloseCalls)
	from := len(mustReadEvents(t, st.o))

	res := st.mustReseat(t, st.reseatParams())
	added := mustReadEvents(t, st.o)[from:]
	stopped := eventsNamed(added, EventStopped)
	if len(stopped) != 3 {
		t.Fatalf("stopped %q; want the three seats", eventSeats(added, EventStopped))
	}
	for _, ev := range stopped {
		if ev.Details != "ctrl_c=skipped pane=already closed leave=ok reseat="+res.Plan.ID {
			t.Errorf("stopped %s: Details %q; want the pane already closed", ev.SeatID, ev.Details)
		}
	}
	if n := len(st.h.paneCloseCalls) - closes; n != 0 {
		t.Errorf("%d pane closes; want none for panes herdr does not know", n)
	}
	if leader := eventOf(t, st.o, "auth-core", LeaderIdentity, EventSpawned); leader.PaneID != fakeRootPaneID("ws-2") {
		t.Errorf("the leader runs in pane %q; want the root pane of the new workspace", leader.PaneID)
	}
}

// TestReseat_AStopThatFailsStartsNothing covers AC-10 (FR-5): the
// implementer's pane cannot be closed, so after the reviewer stopped, the
// reseat ends at the implementer's stop_failed, starts no seat, and leaves
// the leader running. The result and `reseated` say what became of each
// seat, and the printed reseat starts them all again once herdr closes the
// pane.
func TestReseat_AStopThatFailsStartsNothing(t *testing.T) {
	st := newReseatOrg(t)
	sentinel := errors.New("herdr: close refused")
	pane := eventOf(t, st.o, "auth-core", implementerSeatID, EventSpawned).PaneID
	st.h.paneCloseErrs = map[string]error{pane: sentinel}
	from := len(mustReadEvents(t, st.o))

	res := st.o.Reseat(st.reseatParams())
	if res.Err == nil || !errors.Is(res.Err, sentinel) ||
		!strings.Contains(res.Err.Error(), `the stop of seat "implementer" failed, so the reseat stopped there and started no seat`) {
		t.Fatalf("Reseat error %v; want the implementer's stop failure", res.Err)
	}
	if got := reseatOutcomes(res); got != "leader:not_stopped,implementer:stop_failed,reviewer:stopped" {
		t.Errorf("seats %s; want the leader not stopped, the implementer's stop failed and the reviewer stopped", got)
	}
	if want := "ralph org reseat --org-id auth-core --seat leader --seat implementer --seat reviewer"; res.Retry != want || res.Decision != "" {
		t.Errorf("retry %q decision %q; want %q and no DECISION", res.Retry, res.Decision, want)
	}
	added := mustReadEvents(t, st.o)[from:]
	var names []string
	for _, ev := range added {
		names = append(names, ev.Event+":"+ev.SeatID)
	}
	if want := []string{"stopped:reviewer", "stop_failed:implementer", "reseated:"}; !slices.Equal(names, want) {
		t.Errorf("events %q; want %q", names, want)
	}
	if failed := eventsNamed(added, EventStopFailed); len(failed) != 1 || !strings.HasSuffix(failed[0].Details, " reseat="+res.Plan.ID) {
		t.Errorf("stop_failed %+v; want one ending with the reseat's note", failed)
	}
	assertReseated(t, added, "id="+res.Plan.ID+" seats=leader:not_stopped,implementer:stop_failed,reviewer:stopped")

	delete(st.h.paneCloseErrs, pane)
	again := st.mustReseat(t, st.retryParams(t, res.Retry))
	if !slices.Equal(seatIDs(again.Plan.Stop), []string{implementerSeatID, LeaderIdentity}) || len(again.Seats) != 3 {
		t.Errorf("the retry stopped %q and handled %s; want the implementer and the leader stopped and all three started",
			seatIDs(again.Plan.Stop), reseatOutcomes(again))
	}
}

// TestReseat_RefusedWhileAnotherReseatHoldsTheLock covers the first half of
// AC-17 (P6, FR-5a): while reseat.lock is held, a reseat is refused at once,
// naming the holder from its marker as far as the marker says, with nothing
// written, called or stopped, and the holder's marker left as it is. A dry
// run takes no lock and goes on. Once the lock is released, the reseat runs.
func TestReseat_RefusedWhileAnotherReseatHoldsTheLock(t *testing.T) {
	st := newReseatOrg(t)
	path := filepath.Join(st.stateDir, reseatLockFile)
	for _, tc := range []struct {
		name   string
		marker *reseatMarker
		want   string
	}{
		{"a marker with the holder's id", &reseatMarker{OrgID: "other", ID: "20261112T000000-00ff", PID: 4242},
			`another ralph org reseat (reseat=20261112T000000-00ff of org_id "other", pid 4242) holds the ledger's reseat lock ` + path},
		{"a marker written before the holder's plan", &reseatMarker{OrgID: "auth-core", PID: 7},
			`another ralph org reseat (of org_id "auth-core", pid 7) holds the ledger's reseat lock ` + path},
		{"no marker yet", nil, `another ralph org reseat holds the ledger's reseat lock ` + path},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lock, err := acquireHeldFileLock(st.stateDir, reseatLockFile, "reseat", 0)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = lock.release() }()
			if err := writeReseatMarker(lock.f, tc.marker); err != nil {
				t.Fatal(err)
			}
			before, marker, calls := takeReseatLedger(t, st), fileOrAbsent(t, path), len(st.wt.calls)

			res := st.o.Reseat(st.reseatParams())
			var refusal *reseatRefusal
			if !errors.As(res.Err, &refusal) || res.Seats != nil {
				t.Fatalf("Reseat = %v with seats %s; want a refusal", res.Err, reseatOutcomes(res))
			}
			for _, w := range []string{`org: reseat of org_id "auth-core" is refused before anything is stopped: `, tc.want,
				"the reseats of one ledger run one at a time, so run this one again once that one has ended"} {
				if !strings.Contains(res.Err.Error(), w) {
					t.Errorf("refusal %q does not contain %q", res.Err, w)
				}
			}
			if after := takeReseatLedger(t, st); after != before || fileOrAbsent(t, path) != marker || len(st.wt.calls) != calls {
				t.Errorf("the refused reseat wrote or called something: %+v -> %+v, marker %q -> %q, %d worktree calls",
					before, after, marker, fileOrAbsent(t, path), len(st.wt.calls)-calls)
			}

			p := st.reseatParams()
			p.DryRun = true
			if dry := st.o.Reseat(p); dry.Err != nil || !dry.Plan.DryRun || dry.Seats != nil {
				t.Errorf("dry run while the lock is held = %+v; want its plan", dry)
			}
		})
	}
	st.mustReseat(t, st.reseatParams())
}

// TestReseat_AFlagErrorTakesNoLock: a reseat whose --seat, --driver and
// --model are refused (FR-1, AC-11) is refused before it takes reseat.lock,
// as Spawn and Regate check their input before their lock. It leaves no
// reseat.lock in the ledger, and while another reseat holds the lock, the
// flag error is what it reports, not the lock.
func TestReseat_AFlagErrorTakesNoLock(t *testing.T) {
	st := newReseatOrg(t)
	path := filepath.Join(st.stateDir, reseatLockFile)
	p := st.reseatParams(implementerSeatID, "reviewer")
	p.Driver, p.Model = "codex", "gpt-x"
	const want = `org: reseat of org_id "auth-core" is refused before anything is stopped: --driver and --model start one ` +
		`seat on another driver or model, so they need exactly one --seat (2 given)`
	assertFlagRefusal := func(t *testing.T, res ReseatResult) {
		t.Helper()
		var refusal *reseatRefusal
		if !errors.As(res.Err, &refusal) || res.Seats != nil || !strings.HasPrefix(res.Err.Error(), want) ||
			strings.Contains(res.Err.Error(), "holds the ledger's reseat lock") {
			t.Fatalf("Reseat = %v with seats %s; want the flag refusal %q", res.Err, reseatOutcomes(res), want)
		}
	}

	before := takeReseatLedger(t, st)
	assertFlagRefusal(t, st.o.Reseat(p))
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("a flag error left %s (stat: %v); want no reseat.lock", path, err)
	}
	if after := takeReseatLedger(t, st); after != before {
		t.Errorf("a flag error wrote or called something: %+v -> %+v", before, after)
	}

	lock, err := acquireHeldFileLock(st.stateDir, reseatLockFile, "reseat", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.release() }()
	if err := writeReseatMarker(lock.f, &reseatMarker{OrgID: "other", ID: "20261112T000000-00ff", PID: 4242}); err != nil {
		t.Fatal(err)
	}
	assertFlagRefusal(t, st.o.Reseat(p))
}

// TestReseat_ARefusalWritesNothing covers AC-4 at Reseat: a reseat that
// PlanReseat refuses writes nothing to the ledger, calls nothing, and leaves
// only an empty reseat.lock; a dry run makes no reseat.lock at all.
func TestReseat_ARefusalWritesNothing(t *testing.T) {
	st := newReseatOrg(t)
	delete(st.wt.records, "org-auth-core")
	before := takeReseatLedger(t, st)
	res := st.o.Reseat(st.reseatParams())
	var refusal *reseatRefusal
	if !errors.As(res.Err, &refusal) || !strings.Contains(res.Err.Error(), "is missing, and reseat makes no worktree") || res.Seats != nil {
		t.Fatalf("Reseat = %v with seats %s; want the missing record's refusal", res.Err, reseatOutcomes(res))
	}
	after := takeReseatLedger(t, st)
	names := strings.Split(after.names, ",")
	names = slices.DeleteFunc(names, func(n string) bool { return n == reseatLockFile })
	if after.snap != before.snap || after.manifest != before.manifest || after.records != before.records || strings.Join(names, ",") != before.names {
		t.Errorf("the refused reseat wrote or called something: %+v -> %+v", before, after)
	}
	if data := fileOrAbsent(t, filepath.Join(st.stateDir, reseatLockFile)); data != "" {
		t.Errorf("reseat.lock holds %q; want it empty", data)
	}

	dry := newReseatOrg(t)
	p := dry.reseatParams()
	p.DryRun = true
	if res := dry.o.Reseat(p); res.Err != nil || !res.Plan.DryRun || len(res.Plan.Start) != 3 || res.Seats != nil {
		t.Fatalf("dry run = %+v; want the plan of three seats", res)
	}
	if data := fileOrAbsent(t, filepath.Join(dry.stateDir, reseatLockFile)); data != "<absent>" {
		t.Errorf("a dry run left reseat.lock (%q); want none", data)
	}
}

// TestReseat_ASeatChangedBeforeItsStop covers the second half of AC-17
// (P6, FR-5a): when another command stopped or started a seat after the
// reseat's check, the reseat reads that right before the seat's stop, does
// not stop it, and starts nothing. The result names the seat, says what
// became of each seat, and leaves the changed seat out of the retry.
func TestReseat_ASeatChangedBeforeItsStop(t *testing.T) {
	hook := func(t *testing.T, seat string, change func()) {
		done := false
		reseatTestHooks.beforeStopCheck = func(id string) {
			if id == seat && !done {
				done = true
				change()
			}
		}
		t.Cleanup(func() { reseatTestHooks.beforeStopCheck = nil })
	}

	t.Run("the first seat to stop was stopped meanwhile", func(t *testing.T) {
		st := newReseatOrg(t)
		hook(t, "reviewer", func() { st.stopSeat(t, "reviewer") })
		from := len(mustReadEvents(t, st.o))
		res := st.o.Reseat(st.reseatParams())
		if res.Changed != "reviewer" || res.Err == nil ||
			!strings.Contains(res.Err.Error(), `ended before the stop of seat "reviewer", and started no seat: the seat's latest state event is now stopped at `) ||
			!strings.Contains(res.Err.Error(), ", not the spawned at ") {
			t.Fatalf("changed %q error %v; want the reviewer's change", res.Changed, res.Err)
		}
		if got := reseatOutcomes(res); got != "leader:not_stopped,implementer:not_stopped,reviewer:not_stopped" {
			t.Errorf("seats %s; want none stopped", got)
		}
		if want := "ralph org reseat --org-id auth-core --seat leader --seat implementer"; res.Retry != want {
			t.Errorf("retry %q; want %q", res.Retry, want)
		}
		added := mustReadEvents(t, st.o)[from:]
		if len(added) != 1 || added[0].Event != EventStopped || strings.Contains(added[0].Details, "reseat=") {
			t.Errorf("events %+v; want only the other command's stop of the reviewer, and no reseated", added)
		}
	})

	t.Run("a later seat was started again meanwhile", func(t *testing.T) {
		st := newReseatOrg(t)
		// The same spawned again, as a leader that spawned the implementer
		// once more would leave it: equal to the one the check read, at a
		// later index.
		again := eventOf(t, st.o, "auth-core", implementerSeatID, EventSpawned)
		hook(t, implementerSeatID, func() {
			if err := st.o.Manifest.Append(again); err != nil {
				t.Error(err)
			}
		})
		from := len(mustReadEvents(t, st.o))
		res := st.o.Reseat(st.reseatParams())
		if res.Changed != implementerSeatID || res.Err == nil || !strings.Contains(res.Err.Error(), `ended before the stop of seat "implementer"`) {
			t.Fatalf("changed %q error %v; want the implementer's change", res.Changed, res.Err)
		}
		if got := reseatOutcomes(res); got != "leader:not_stopped,implementer:not_stopped,reviewer:stopped" {
			t.Errorf("seats %s; want the reviewer stopped and the others not", got)
		}
		if want := "ralph org reseat --org-id auth-core --seat leader --seat reviewer"; res.Retry != want {
			t.Errorf("retry %q; want %q", res.Retry, want)
		}
		added := mustReadEvents(t, st.o)[from:]
		if len(eventsNamed(added, EventSpawnStarted)) != 0 || !slices.Equal(eventSeats(added, EventStopped), []string{"reviewer"}) {
			t.Errorf("stopped %q spawn_started %q; want the reviewer stopped and nothing started",
				eventSeats(added, EventStopped), eventSeats(added, EventSpawnStarted))
		}
		assertReseated(t, added, "id="+res.Plan.ID+" seats=leader:not_stopped,implementer:not_stopped,reviewer:stopped")
	})
}

// TestReseat_RetryAfterAFailedStart covers AC-18 (FR-6, P7): a start that
// fails ends the reseat there, the result says which seats started, which
// failed and which were left stopped, and the printed reseat, run as it is,
// starts the rest, whose latest state is now spawn_failed or stopped.
func TestReseat_RetryAfterAFailedStart(t *testing.T) {
	t.Run("the leader's start failed", func(t *testing.T) {
		st := newReseatOrg(t)
		st.h.closeRemoves = true
		st.h.workspaceID = "ws-2"
		sentinel := errors.New("herdr: workspace create refused")
		st.h.workspaceCreateErr = sentinel
		from := len(mustReadEvents(t, st.o))

		res := st.o.Reseat(st.reseatParams())
		if !errors.Is(res.Err, sentinel) || !strings.Contains(res.Err.Error(), `seat "leader" did not start again, so the reseat started no seat after it`) {
			t.Fatalf("Reseat error %v; want the leader's failed start", res.Err)
		}
		if got := reseatOutcomes(res); got != "leader:start_failed,implementer:stopped,reviewer:stopped" {
			t.Errorf("seats %s; want the leader's start failed and the others left stopped", got)
		}
		if want := "ralph org reseat --org-id auth-core --seat leader --seat implementer --seat reviewer"; res.Retry != want || res.Decision != "" {
			t.Errorf("retry %q decision %q; want %q and no DECISION", res.Retry, res.Decision, want)
		}
		added := mustReadEvents(t, st.o)[from:]
		assertReseated(t, added, "id="+res.Plan.ID+" seats=leader:start_failed,implementer:stopped,reviewer:stopped")
		if failed := eventsNamed(added, EventSpawnFailed); len(failed) != 1 || failed[0].SeatID != LeaderIdentity {
			t.Errorf("spawn_failed %+v; want the leader's", failed)
		}

		st.h.workspaceCreateErr = nil
		again := st.mustReseat(t, st.retryParams(t, res.Retry))
		if len(again.Plan.Stop) != 0 || reseatOutcomes(again) != "leader:started,implementer:started,reviewer:started" || again.Decision != "" {
			t.Errorf("the retry stopped %q and left %s, decision %q; want all three started, none stopped, no DECISION",
				seatIDs(again.Plan.Stop), reseatOutcomes(again), again.Decision)
		}
	})

	t.Run("a seat after the leader failed", func(t *testing.T) {
		st := newReseatOrg(t)
		sentinel := errors.New("herdr: split refused")
		st.h.paneSplitErr = sentinel

		res := st.o.Reseat(st.reseatParams())
		if !errors.Is(res.Err, sentinel) || !strings.Contains(res.Err.Error(), `seat "implementer" did not start again, so the reseat started no seat after it`) {
			t.Fatalf("Reseat error %v; want the implementer's failed start", res.Err)
		}
		if got := reseatOutcomes(res); got != "leader:started,implementer:start_failed,reviewer:stopped" {
			t.Errorf("seats %s; want the leader started, the implementer's start failed and the reviewer left stopped", got)
		}
		if want := "ralph org reseat --org-id auth-core --seat implementer --seat reviewer"; res.Retry != want || res.Decision != "" {
			t.Errorf("retry %q decision %q; want %q and no DECISION", res.Retry, res.Decision, want)
		}

		// The retry starts the implementer first, as the org first spawned
		// it: the start order is that of each seat's first spawn_started in
		// the org's current life, which the implementer's failed start did
		// not move.
		st.h.paneSplitErr = nil
		again := st.mustReseat(t, st.retryParams(t, res.Retry))
		want := ReseatDecisionCommand("auth-core", ReseatDecisionText(again.Plan.ID, []string{implementerSeatID, "reviewer"}), "")
		if len(again.Plan.Stop) != 0 || reseatOutcomes(again) != "implementer:started,reviewer:started" || again.Decision != want {
			t.Errorf("the retry stopped %q and left %s, decision %q; want both started, none stopped, and %q",
				seatIDs(again.Plan.Stop), reseatOutcomes(again), again.Decision, want)
		}
	})
}

// TestReseat_RetryStartsASeatWhoseFirstSpawnNeverEnded covers the retry of a
// seat whose first spawn in the org's current life never reached `spawned`
// (cross-review cycle 1, finding 1): the reviewer's spawn_started is its
// latest state event, so it is active and --seat takes it. The reseat stops
// it, its start fails, and the printed reseat, run as it is, starts it,
// because --seat also takes a seat with a reseat=<id> note on one of its
// events in that life. The same seat stopped by hand instead, with no note
// and no `spawned`, is still refused.
func TestReseat_RetryStartsASeatWhoseFirstSpawnNeverEnded(t *testing.T) {
	newStuckReviewer := func(t *testing.T) *featureStart {
		st := newFeatureStart(t)
		st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
		approveFeaturePlan(t, st.o, "auth-core", st.worktree("auth-core"))
		st.spawnSeat(t, implementerSeatID, implementerRole, "opus")
		if err := st.o.Manifest.Append(ManifestEvent{
			OrgID: "auth-core", SeatID: "reviewer", Event: EventSpawnStarted, Role: "reviewer", Driver: "claude",
			ModelCLIDefault: true, Worktree: st.worktree("auth-core"),
		}); err != nil {
			t.Fatal(err)
		}
		return st
	}

	t.Run("the reseat stopped it and its start failed", func(t *testing.T) {
		st := newStuckReviewer(t)
		sentinel := errors.New("herdr: split refused")
		st.h.paneSplitErr = sentinel
		res := st.o.Reseat(st.reseatParams("reviewer"))
		if !errors.Is(res.Err, sentinel) || reseatOutcomes(res) != "reviewer:start_failed" {
			t.Fatalf("Reseat = %v with seats %s; want the reviewer's failed start", res.Err, reseatOutcomes(res))
		}
		if want := "ralph org reseat --org-id auth-core --seat reviewer"; res.Retry != want || res.Decision != "" {
			t.Errorf("retry %q decision %q; want %q and no DECISION, no seat having started", res.Retry, res.Decision, want)
		}
		for _, s := range reseatLifeSeats(mustReadEvents(t, st.o), "auth-core") {
			if s.id == "reviewer" && (s.spawned || s.active() || s.latest.Event != EventSpawnFailed) {
				t.Fatalf("reviewer: spawned %v, active %v, latest %s; want a spawn_failed seat never spawned", s.spawned, s.active(), s.latest.Event)
			}
		}

		st.h.paneSplitErr = nil
		again := st.mustReseat(t, st.retryParams(t, res.Retry))
		want := ReseatDecisionCommand("auth-core", ReseatDecisionText(again.Plan.ID, []string{"reviewer"}), "")
		if len(again.Plan.Stop) != 0 || reseatOutcomes(again) != "reviewer:started" || again.Decision != want {
			t.Errorf("the retry stopped %q and left %s, decision %q; want the reviewer started, nothing stopped, and %q",
				seatIDs(again.Plan.Stop), reseatOutcomes(again), again.Decision, want)
		}
	})

	t.Run("stopped by hand", func(t *testing.T) {
		st := newStuckReviewer(t)
		st.stopSeat(t, "reviewer")
		st.assertReseatRefused(t, st.reseatParams("reviewer"),
			`seat "reviewer" of org_id "auth-core" has not been spawned since the org's last disband`,
			"(the seats it can start again are: leader, implementer)")
	})
}

// TestReseat_DecisionForTheSeatsStartedBeforeAFailure covers a reseat of
// seats other than the leader that fails partway (cross-review cycle 1,
// finding 2): the implementer starts, and the reviewer's start is refused,
// another spawn of the reviewer holding its spawn lock. The leader keeps
// running, and the implementer runs with no TASK, so the result holds the
// DECISION for the implementer besides the retry of the reviewer, which
// repeats the --config the reseat was given (cross-review cycle 1, finding
// 3). The retry, run as it is once the lock is free, starts the reviewer and
// holds the DECISION for the reviewer alone.
func TestReseat_DecisionForTheSeatsStartedBeforeAFailure(t *testing.T) {
	st := newReseatOrg(t)
	lock, err := acquireHeldFileLock(st.stateDir, spawnLockName("auth-core", "reviewer"), "spawn", 0)
	if err != nil {
		t.Fatal(err)
	}
	released := false
	release := func() {
		if !released {
			released = true
			_ = lock.release()
		}
	}
	t.Cleanup(release)

	p := st.reseatParams(implementerSeatID, "reviewer")
	p.HintConfigPath = "conf/ralph.toml"
	res := st.o.Reseat(p)
	if res.Err == nil || !strings.Contains(res.Err.Error(), `seat "reviewer" did not start again`) {
		t.Fatalf("Reseat error %v; want the reviewer's refused start", res.Err)
	}
	if got := reseatOutcomes(res); got != "implementer:started,reviewer:start_failed" {
		t.Errorf("seats %s; want the implementer started and the reviewer's start failed", got)
	}
	if want := "ralph org reseat --org-id auth-core --seat reviewer --config conf/ralph.toml"; res.Retry != want {
		t.Errorf("retry %q; want %q", res.Retry, want)
	}
	if want := ReseatDecisionCommand("auth-core", ReseatDecisionText(res.Plan.ID, []string{implementerSeatID}), ""); res.Decision != want {
		t.Errorf("Decision = %q; want %q", res.Decision, want)
	}

	release()
	again := st.mustReseat(t, st.retryParams(t, res.Retry))
	want := ReseatDecisionCommand("auth-core", ReseatDecisionText(again.Plan.ID, []string{"reviewer"}), "")
	if reseatOutcomes(again) != "reviewer:started" || again.Decision != want {
		t.Errorf("the retry left %s, decision %q; want the reviewer started and %q", reseatOutcomes(again), again.Decision, want)
	}
}

// TestReseatLifeSeats_StartOrderIsTheFirstSpawnOfTheCurrentLife pins the
// start order's key (FR-6): each seat's first spawn_started in the org's
// current life. A later start of the seat (the reviewer's here) does not
// move it, a spawn_started before the org's last disband does not count (the
// implementer's), and the seats with none come last, by seat id.
func TestReseatLifeSeats_StartOrderIsTheFirstSpawnOfTheCurrentLife(t *testing.T) {
	ev := func(seat, event string) ManifestEvent {
		return ManifestEvent{OrgID: "auth-core", SeatID: seat, Event: event}
	}
	events := []ManifestEvent{
		ev(implementerSeatID, EventSpawnStarted),
		ev(implementerSeatID, EventSpawned),
		{OrgID: "auth-core", Event: EventDisbanded},
		ev("reviewer", EventSpawnStarted),
		ev("reviewer", EventSpawned),
		ev(implementerSeatID, EventSpawnStarted),
		ev(implementerSeatID, EventSpawned),
		ev("reviewer", EventStopped),
		ev("reviewer", EventSpawnStarted),
		ev("reviewer", EventSpawned),
		ev("zeta", EventSpawned),
		ev("alpha", EventSpawned),
	}
	var got []string
	for _, s := range reseatLifeSeats(events, "auth-core") {
		got = append(got, s.id)
	}
	if want := []string{"reviewer", implementerSeatID, "alpha", "zeta"}; !slices.Equal(got, want) {
		t.Fatalf("start order %q; want %q", got, want)
	}
}

// TestReseatRetryCommand pins the retry's words: the target seats not
// started, in start order, but the changed one; --driver and --model when it
// names the one seat that was given them; --config when the reseat was given
// one, and none otherwise; --state-dir when the ledger was named with it;
// nothing when every seat started.
func TestReseatRetryCommand(t *testing.T) {
	plan := ReseatPlan{OrgID: "auth-core", Start: []ReseatSeat{
		{SeatID: LeaderIdentity, Leader: true}, {SeatID: implementerSeatID}, {SeatID: "reviewer"},
	}}
	seats := func(outcomes ...ReseatSeatOutcome) []ReseatSeatResult {
		out := make([]ReseatSeatResult, len(outcomes))
		for i, o := range outcomes {
			out[i] = ReseatSeatResult{SeatID: plan.Start[i].SeatID, Outcome: o}
		}
		return out
	}
	one := ReseatPlan{OrgID: "auth-core", Start: []ReseatSeat{{SeatID: implementerSeatID, Driver: "codex", Model: "gpt-x", Overridden: true}}}
	for _, tc := range []struct {
		name   string
		plan   ReseatPlan
		seats  []ReseatSeatResult
		skip   string
		config string
		hint   string
		want   string
	}{
		{"all started", plan, seats(ReseatSeatStarted, ReseatSeatStarted, ReseatSeatStarted), "", "", "", ""},
		{"all started, given --config", plan, seats(ReseatSeatStarted, ReseatSeatStarted, ReseatSeatStarted), "", "/cfg/ralph.toml", "", ""},
		{"the rest after a failed start", plan, seats(ReseatSeatStarted, ReseatSeatStartFailed, ReseatSeatStopped), "", "", "",
			"ralph org reseat --org-id auth-core --seat implementer --seat reviewer"},
		{"the rest after a failed start, given --config", plan, seats(ReseatSeatStarted, ReseatSeatStartFailed, ReseatSeatStopped), "",
			"/cfg dir/ralph.toml", "", "ralph org reseat --org-id auth-core --seat implementer --seat reviewer --config '/cfg dir/ralph.toml'"},
		{"the changed seat left out", plan, seats(ReseatSeatNotStopped, ReseatSeatNotStopped, ReseatSeatStopped), implementerSeatID, "",
			"/state/org dir", "ralph org reseat --org-id auth-core --seat leader --seat reviewer --state-dir '/state/org dir'"},
		{"only the changed seat", one, []ReseatSeatResult{{SeatID: implementerSeatID, Outcome: ReseatSeatNotStopped}}, implementerSeatID,
			"ralph.toml", "", ""},
		{"one seat given a driver and a model", one, []ReseatSeatResult{{SeatID: implementerSeatID, Outcome: ReseatSeatStartFailed}}, "", "",
			"/s", "ralph org reseat --org-id auth-core --seat implementer --driver codex --model gpt-x --state-dir /s"},
		{"one seat given a driver, a model and --config", one, []ReseatSeatResult{{SeatID: implementerSeatID, Outcome: ReseatSeatStartFailed}},
			"", "ralph.toml", "/s", "ralph org reseat --org-id auth-core --seat implementer --driver codex --model gpt-x --config ralph.toml --state-dir /s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := reseatRetryCommand(tc.plan, tc.seats, tc.skip, tc.config, tc.hint); got != tc.want {
				t.Errorf("reseatRetryCommand = %q; want %q", got, tc.want)
			}
		})
	}
}
