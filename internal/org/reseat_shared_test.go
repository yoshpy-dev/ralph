package org

import (
	"slices"
	"strings"
	"testing"
)

// The --scope `ralph org reseat` gives the seats other than the leader of a
// feature org whose split plan has shared paths (plan
// docs/plans/active/2026-11-22-split-shared-paths.md, AC8's reseat): the
// reservation followed by the shared paths, the --scope the leader's task
// names for the seats (featureLeaderTask). The featureStart fixture lives in
// feature_test.go and the shared plan in feature_shared_test.go.

// reseatSharedScope is the --scope of auth-core's seats when its split plan
// shares pnpm-lock.yaml and packages/shared/: the reservation, then the
// shared paths as the binding records them, normalized and sorted.
const reseatSharedScope = "docs/auth.md, internal/auth/, packages/shared/, pnpm-lock.yaml"

// reseatScopeCases are auth-core with a split plan that shares nothing
// (splitPlanFixture as it is) and with one that shares two paths, written in
// another order than the record's, each with the --scope its seats get.
var reseatScopeCases = []struct{ name, shared, scope string }{
	{"no shared paths", "", reseatScope},
	{"shared paths", "pnpm-lock.yaml, packages/shared/", reseatSharedScope},
}

// newSharedReseatOrg is newReseatOrg for a split plan that shares shared (""
// for splitPlanFixture, which shares nothing): auth-core started, its
// implementation plan approved, and its implementer (claude opus) and
// reviewer (claude, the CLI's default model) spawned with scope, as its
// leader spawns them with the --scope of its task.
func newSharedReseatOrg(t *testing.T, shared, scope string) *featureStart {
	t.Helper()
	st := newFeatureStart(t)
	if shared == "" {
		st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	} else {
		st.startSharedAuthCore(t, shared)
	}
	approveFeaturePlan(t, st.o, "auth-core", st.worktree("auth-core"))
	for _, s := range []struct{ id, role, model string }{{implementerSeatID, implementerRole, "opus"}, {"reviewer", "reviewer", ""}} {
		r := st.o.Spawn(SpawnParams{
			OrgID: "auth-core", SeatID: s.id, Role: s.role, Driver: "claude", Model: s.model,
			Cwd: st.worktree("auth-core"), Scope: scope, TimeoutMS: 5000,
		})
		if r.Outcome != SpawnOutcomeSpawned {
			t.Fatalf("spawn %s: %+v", s.id, r)
		}
	}
	return st
}

// assertTaskScope checks that the leader's task names scope as the --scope
// of every seat's spawn, so that a test which expects scope of a reseat
// expects the --scope the leader's task names.
func (st *featureStart) assertTaskScope(t *testing.T, scope string) {
	t.Helper()
	if task, want := st.freshLeaderTask(t), "(座席の spawn には --scope "+shellQuote(scope)+" を付ける)"; !strings.Contains(task, want) {
		t.Fatalf("the leader's task does not hold %q:\n%s", want, task)
	}
}

// TestPlanReseat_SeatsGetTheLeadersTaskScope covers AC8 for the seats other
// than the leader at the org layer: a reseat with no --seat, its dry run, and
// one of the implementer and the reviewer alone give each of them the
// --scope the leader's task names, the reservation followed by the shared
// paths for a split plan that has some, and the reservation alone, as
// before, for one that has none.
func TestPlanReseat_SeatsGetTheLeadersTaskScope(t *testing.T) {
	for _, tc := range reseatScopeCases {
		t.Run(tc.name, func(t *testing.T) {
			st := newSharedReseatOrg(t, tc.shared, tc.scope)
			st.assertTaskScope(t, tc.scope)
			dry := st.reseatParams()
			dry.DryRun = true
			for _, p := range []ReseatParams{st.reseatParams(), dry, st.reseatParams(implementerSeatID, "reviewer")} {
				plan := st.mustPlanReseat(t, p)
				var seats []string
				for _, seat := range plan.Start {
					if seat.Leader {
						continue
					}
					seats = append(seats, seat.SeatID)
					if seat.Spawn == nil || seat.Spawn.Scope != tc.scope {
						t.Errorf("reseat %q (dry run %v): seat %s spawn %+v; want --scope %q", p.Seats, p.DryRun, seat.SeatID, seat.Spawn, tc.scope)
					}
				}
				if !slices.Equal(seats, []string{implementerSeatID, "reviewer"}) {
					t.Errorf("reseat %q (dry run %v) starts %q besides the leader; want implementer, reviewer", p.Seats, p.DryRun, seats)
				}
			}
		})
	}
}

// TestReseat_StartsTheSeatsWithTheLeadersTaskScope covers AC8's reseat end
// to end: the implementer and the reviewer a reseat stops and starts again
// are spawned with that --scope, which their `spawned` records.
func TestReseat_StartsTheSeatsWithTheLeadersTaskScope(t *testing.T) {
	for _, tc := range reseatScopeCases {
		t.Run(tc.name, func(t *testing.T) {
			st := newSharedReseatOrg(t, tc.shared, tc.scope)
			st.h.closeRemoves = true
			from := len(mustReadEvents(t, st.o))

			res := st.mustReseat(t, st.reseatParams(implementerSeatID, "reviewer"))
			added := mustReadEvents(t, st.o)[from:]
			spawned := eventsNamed(added, EventSpawned)
			if got := eventSeats(added, EventSpawned); !slices.Equal(got, []string{implementerSeatID, "reviewer"}) {
				t.Fatalf("spawned %q; want implementer, reviewer", got)
			}
			for _, ev := range spawned {
				if want := "scope=" + tc.scope + " permission_mode="; !strings.HasPrefix(ev.Details, want) ||
					!strings.HasSuffix(ev.Details, "reseat="+res.Plan.ID) {
					t.Errorf("spawned %s details %q; want %q first and reseat=%s last", ev.SeatID, ev.Details, want, res.Plan.ID)
				}
			}
		})
	}
}

// TestPlanReseat_CustomRoleSpawnNamesTheLeadersTaskScope covers the spawn a
// reseat names for a seat of a custom role (reseatCustomRoleErr) in an org
// whose split plan has shared paths: its --scope is the one reseat gives the
// other seats, the reservation followed by the shared paths, as one shell
// word.
func TestPlanReseat_CustomRoleSpawnNamesTheLeadersTaskScope(t *testing.T) {
	st := newSharedReseatOrg(t, "pnpm-lock.yaml, packages/shared/", reseatSharedScope)
	st.o.Config.MaxSeats = 4
	if r := st.o.Spawn(SpawnParams{
		OrgID: "auth-core", SeatID: "scribe", Role: "scribe", Driver: "claude", Model: "haiku",
		Cwd: st.worktree("auth-core"), Scope: reseatSharedScope, Prompt: "Keep the notes.", TimeoutMS: 5000,
	}); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn scribe: %+v", r)
	}
	st.assertReseatRefused(t, st.reseatParams(), "stop it with ralph org stop --org-id auth-core --seat scribe and "+
		"spawn it again with ralph org spawn --org-id auth-core --id scribe --role scribe --driver claude --model haiku "+
		"--cwd "+ShellWord(st.worktree("auth-core"))+" --scope "+ShellWord(reseatSharedScope)+" --prompt <its prompt> by hand")
}
