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
)

// PlanReseat at the org layer (plan docs/plans/active/2026-11-12-org-reseat.md,
// slice S1: AC-4 to AC-7, AC-9, AC-11 and AC-19, and P7's --seat). The
// fixture is featureStart (feature_test.go): the feature org auth-core of
// the split plan auth-split on fake drivers and fake worktrees.

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
func (st *featureStart) assertReseatRefused(t *testing.T, p ReseatParams, want ...string) error {
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
// --state-dir when the ledger was named with it. That reseat then passes and
// starts them all, stopping none.
func TestPlanReseat_NoActiveSeatNamesTheStoppedSeats(t *testing.T) {
	st := newReseatOrg(t)
	for _, id := range []string{"reviewer", implementerSeatID, LeaderIdentity} {
		st.stopSeat(t, id)
	}
	st.assertReseatRefused(t, st.reseatParams(),
		`org_id "auth-core" has no active seat since its last disband, so a reseat with no --seat has no seat to start again; `+
			`its stopped seats are leader, implementer, reviewer: to start them again, run `+
			`ralph org reseat --org-id auth-core --seat leader --seat implementer --seat reviewer`)
	st.o.InboxHintStateDir = "/state/org dir"
	err := st.assertReseatRefused(t, st.reseatParams())
	if !strings.HasSuffix(err.Error(), "--seat reviewer --state-dir '/state/org dir'") {
		t.Errorf("refusal %q does not end with the command's --state-dir", err)
	}

	plan := st.mustPlanReseat(t, st.reseatParams(LeaderIdentity, implementerSeatID, "reviewer"))
	if len(plan.Stop) != 0 || !slices.Equal(seatIDs(plan.Start), []string{LeaderIdentity, implementerSeatID, "reviewer"}) {
		t.Fatalf("stop %q start %q; want no stop and all three started", seatIDs(plan.Stop), seatIDs(plan.Start))
	}
}

// TestPlanReseat_NoActiveSeatWithoutStoppedOnes covers the rest of the
// refusal of a reseat with no --seat and no active seat: with no stopped
// seat, it names the seats spawned in the current life with their state;
// with none spawned at all, the start of the org's leader.
func TestPlanReseat_NoActiveSeatWithoutStoppedOnes(t *testing.T) {
	t.Run("a seat whose spawn failed", func(t *testing.T) {
		st := newFeatureStart(t)
		st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
		if err := st.o.Manifest.Append(ManifestEvent{OrgID: "auth-core", SeatID: LeaderIdentity, Event: EventSpawnFailed}); err != nil {
			t.Fatal(err)
		}
		st.assertReseatRefused(t, st.reseatParams(), "none of its seats is stopped, and the seats spawned since then are leader (spawn_failed): "+
			"name the ones to start again with --seat (ralph org reseat --org-id auth-core --seat <seat>)")
	})
	t.Run("no seat spawned", func(t *testing.T) {
		st := newFeatureStart(t)
		f := &FeatureBinding{Split: "auth-split", Feature: "auth-core", Digest: planDigestOf(t, st.plan), Branch: "fix/auth-core",
			Worktree: st.worktree("auth-core"), Approval: config.ApprovalModePerPlan, FeatureDigest: featureDigestOf(t, st.plan, "auth-core")}
		if err := st.o.Manifest.Append(boundReserveEvent("auth-core", f, "docs/auth.md", "internal/auth/")); err != nil {
			t.Fatal(err)
		}
		st.assertReseatRefused(t, st.reseatParams(), "no seat has been spawned since then: start its leader with ralph org start --plan "+
			ShellWord(filepath.Join(SplitPlansDirIn(st.stateDir), "auth-split.md"))+" --feature auth-core --org-id auth-core")
	})
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
			err := st.assertReseatRefused(t, st.reseatParams(tc.seats...), want...)
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
