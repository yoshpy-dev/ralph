package org

import (
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// `ralph org start --plan` with a split plan's shared paths (plan
// 2026-11-22-split-shared-paths, AC3, AC5 and AC7 at the
// org layer). The featureStart fixture lives in feature_test.go.

// sharedPlanFixture is splitPlanFixture with a `- Shared: <shared>` line in
// its header.
func sharedPlanFixture(t *testing.T, shared string) string {
	t.Helper()
	return editedSplitPlan(t, "- Owner: someone\n", "- Owner: someone\n- Shared: "+shared+"\n")
}

// otherSplitPlan is an approved split plan with one feature, lock, that
// reserves reserve, and a `- Shared: <shared>` line in its header when shared
// is not empty.
func otherSplitPlan(t *testing.T, reserve, shared string) string {
	t.Helper()
	header := "# 別の分割\n\n- Status: Approved\n- Approved: 2026-11-22 sha256:000000000000\n"
	if shared != "" {
		header += "- Shared: " + shared + "\n"
	}
	return approveSplitPlan(t, header+"\n## Features\n\n### lock\n\n- Reserve: "+reserve+"\n\nObjective: update the lockfile.\n")
}

// startSharedAuthCore writes the split plan auth-split with the shared paths
// shared and starts its feature auth-core, returning the binding it records.
func (st *featureStart) startSharedAuthCore(t *testing.T, shared string) *FeatureBinding {
	t.Helper()
	st.plan = st.writePlan(t, "auth-split", approveSplitPlan(t, sharedPlanFixture(t, shared)))
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	return ActiveFeature(mustReadEvents(t, st.o), "auth-core")
}

// TestStartFeature_RecordsSharedPaths covers AC3 at the org layer: start
// records the split plan's shared paths, normalized and sorted, as the last
// token of the feature's scope_reserved, right after its feature digest, and
// that is the binding the org reads back; the reservation's paths stay the
// feature's own.
func TestStartFeature_RecordsSharedPaths(t *testing.T) {
	st := newFeatureStart(t)
	got := st.startSharedAuthCore(t, "pnpm-lock.yaml, ./packages//shared/")
	wt, digest, featureDigest := st.worktree("auth-core"), planDigestOf(t, st.plan), featureDigestOf(t, st.plan, "auth-core")
	want := "paths=docs/auth.md,internal/auth/ split=auth-split feature=auth-core digest=" + digest + " branch=fix/auth-core approval=per-plan" +
		" feature_digest=" + featureDigest + " shared=packages/shared/,pnpm-lock.yaml"
	if ev := eventOf(t, st.o, "auth-core", "", EventScopeReserved); ev.Details != want || ev.Worktree != wt {
		t.Fatalf("scope_reserved = %+v, want Details %q and Worktree %q", ev, want, wt)
	}
	assertActiveFeature(t, st.o, "auth-core", &FeatureBinding{
		Split: "auth-split", Feature: "auth-core", Digest: digest, Branch: "fix/auth-core", Worktree: wt, Approval: config.ApprovalModePerPlan,
		FeatureDigest: featureDigest, Shared: "packages/shared/,pnpm-lock.yaml",
	})
	if shared := got.SharedPaths(); !slices.Equal(shared, []string{"packages/shared/", "pnpm-lock.yaml"}) {
		t.Fatalf("SharedPaths = %q, want packages/shared/ and pnpm-lock.yaml", shared)
	}
	if paths := ActiveReservation(mustReadEvents(t, st.o), "auth-core"); !slices.Equal(paths, []string{"docs/auth.md", "internal/auth/"}) {
		t.Fatalf("ActiveReservation = %q, want the feature's own paths only", paths)
	}
}

// TestStartFeature_SharedRestartsWithTheSamePlan covers the restart of AC3:
// with the same split plan, its shared paths included, the same start again
// records and calls nothing while the leader runs, and once the leader
// stopped it spawns a new leader with no second scope_reserved, the binding
// and its shared paths as the first start recorded them.
func TestStartFeature_SharedRestartsWithTheSamePlan(t *testing.T) {
	st := newFeatureStart(t)
	first := st.startSharedAuthCore(t, "pnpm-lock.yaml")
	if first == nil || first.Shared != "pnpm-lock.yaml" {
		t.Fatalf("expected the first start to record the shared path, got %+v", first)
	}
	before := takeSpawnSnapshot(t, st.o, st.h, st.a)
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeIdempotent)
	if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
		t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
	}

	st.stopLeader(t, "auth-core")
	st.h.nextPaneID = "pane-2"
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	if n := countString(eventNames(t, st.o), EventScopeReserved); n != 1 {
		t.Fatalf("expected one scope_reserved, got %d", n)
	}
	if ev := eventOf(t, st.o, "auth-core", LeaderIdentity, EventSpawned); ev.PaneID != "pane-2" {
		t.Fatalf("expected a new leader, got %+v", ev)
	}
	assertActiveFeature(t, st.o, "auth-core", first)
}

// TestStartFeature_SharedChangedRefused covers the other half of the
// restart: after its leader stopped, a feature org is refused a start once
// the split plan's shared paths changed (added to, or the line removed) and
// the plan was approved again, since its feature digest changed, with
// nothing recorded or called and no worktree looked up; the refusal names
// the shared paths before and now.
func TestStartFeature_SharedChangedRefused(t *testing.T) {
	for _, tc := range []struct {
		name, from, to, clause string
	}{
		{"a path added", "- Shared: pnpm-lock.yaml\n", "- Shared: pnpm-lock.yaml, go.sum\n", `("pnpm-lock.yaml", now "go.sum,pnpm-lock.yaml"; `},
		{"the line removed", "- Shared: pnpm-lock.yaml\n", "", `("pnpm-lock.yaml", now none; `},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newFeatureStart(t)
			st.startSharedAuthCore(t, "pnpm-lock.yaml")
			oldDigest := featureDigestOf(t, st.plan, "auth-core")
			st.stopLeader(t, "auth-core")

			st.writePlan(t, "auth-split", approveSplitPlan(t, strings.Replace(sharedPlanFixture(t, "pnpm-lock.yaml"), tc.from, tc.to, 1)))
			newDigest := featureDigestOf(t, st.plan, "auth-core")
			if newDigest == oldDigest {
				t.Fatalf("expected the edit to change the feature digest %s", oldDigest)
			}
			before := takeSpawnSnapshot(t, st.o, st.h, st.a)
			calls := slices.Clone(st.wt.calls)
			assertStartRefused(t, st.o.StartFeature(st.params("auth-core")),
				`org_id "auth-core" is bound to feature "auth-core" of split plan "auth-split"`,
				`is refused, since the shared paths of split plan "auth-split" (its - Shared: line) changed after org_id "auth-core" was started `+
					tc.clause+`feature digest "`+oldDigest+`", now "`+newDigest+`"): ralph org disband --org-id auth-core first, or use another org_id`)
			if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
				t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
			}
			st.wantCalls(t, calls...)
		})
	}
}

// TestStartFeature_SharedAdmission covers AC7 and AC5 at the org layer, in
// both orders of start. With the feature org that shares packages/shared/
// and pnpm-lock.yaml running, a feature org of another split plan that
// reserves a file under the shared directory, and a plain --reserve leader
// that reserves the shared file, are refused, naming the running org and
// the pair; with an org that reserves packages/ (a directory holding the
// shared one) or pnpm-lock.yaml running, the feature org that shares them is
// refused, naming that org and the pair. A refused start records nothing,
// and the refused plain spawn records only its `rejected`. An org of another
// split plan that shares the same paths starts alongside.
func TestStartFeature_SharedAdmission(t *testing.T) {
	const shared = "pnpm-lock.yaml, packages/shared/"
	const reservedFirst = `the shared paths of split plan "auth-split" overlap the reservation of running org_id "lock" ` +
		`(each pair names the shared path first, then the reserved one): `
	sharedFirst := func(reserved string) string {
		return `reservation overlaps the shared paths of running org_id "auth-core" (each pair names the reserved path first, ` +
			`then the path split plan "auth-split" shares, which every org of it may change): ` + reserved
	}
	startLock := func(t *testing.T, st *featureStart, reserve, lockShared string) StartFeatureResult {
		t.Helper()
		p := st.params("lock")
		p.SplitPath = st.writePlan(t, "lock-split", otherSplitPlan(t, reserve, lockShared))
		st.h.workspaceID, st.h.nextPaneID = "ws-2", "pane-2"
		return st.o.StartFeature(p)
	}
	assertNothingRecorded := func(t *testing.T, st *featureStart, before spawnSnapshot) {
		t.Helper()
		if after := takeSpawnSnapshot(t, st.o, st.h, st.a); after != before {
			t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
		}
	}

	t.Run("the shared org first, a file under the shared directory", func(t *testing.T) {
		st := newFeatureStart(t)
		st.startSharedAuthCore(t, shared)
		before := takeSpawnSnapshot(t, st.o, st.h, st.a)
		assertStartRefused(t, startLock(t, st, "packages/shared/index.ts", ""), sharedFirst("packages/shared/index.ts overlaps packages/shared/"))
		assertNothingRecorded(t, st, before)
	})
	t.Run("the shared org first, the shared file by a plain reservation", func(t *testing.T) {
		st := newFeatureStart(t)
		st.startSharedAuthCore(t, shared)
		herdrCalls := len(st.h.calls)
		st.h.workspaceID, st.h.nextPaneID = "ws-2", "pane-2"
		r := st.o.Spawn(leaderParams("lock", "pnpm-lock.yaml"))
		if r.Outcome != SpawnOutcomeRejected || r.Err == nil || !strings.Contains(r.Err.Error(), sharedFirst("pnpm-lock.yaml overlaps pnpm-lock.yaml")) {
			t.Fatalf("expected the plain reservation refused, got %+v", r)
		}
		// A refused new seat records its `rejected`, as for an overlap of two
		// reservations, but no reservation and no pane.
		if ev := lastEvent(t, st.o); ev.Event != EventRejected || ev.OrgID != "lock" || ev.SeatID != LeaderIdentity {
			t.Fatalf("expected the leader's rejected last, got %+v", ev)
		}
		if n := countString(eventNames(t, st.o), EventScopeReserved); n != 1 || len(st.h.calls) != herdrCalls {
			t.Fatalf("expected only auth-core's scope_reserved and no herdr call, got %d and %d calls (was %d)", n, len(st.h.calls), herdrCalls)
		}
	})
	t.Run("the reserving org first, a directory holding the shared one", func(t *testing.T) {
		st := newFeatureStart(t)
		if res := startLock(t, st, "packages/", ""); res.Spawn.Outcome != SpawnOutcomeSpawned || res.Spawn.Err != nil {
			t.Fatalf("expected lock to start, got %+v", res.Spawn)
		}
		st.plan = st.writePlan(t, "auth-split", approveSplitPlan(t, sharedPlanFixture(t, shared)))
		before := takeSpawnSnapshot(t, st.o, st.h, st.a)
		assertStartRefused(t, st.o.StartFeature(st.params("auth-core")), reservedFirst+"packages/shared/ overlaps packages/")
		assertNothingRecorded(t, st, before)
	})
	t.Run("the reserving org first, the shared file by a plain reservation", func(t *testing.T) {
		st := newFeatureStart(t)
		spawnLeaderIn(t, st.o, st.h, "lock", "ws-2", "pane-2", "pnpm-lock.yaml")
		st.plan = st.writePlan(t, "auth-split", approveSplitPlan(t, sharedPlanFixture(t, shared)))
		before := takeSpawnSnapshot(t, st.o, st.h, st.a)
		assertStartRefused(t, st.o.StartFeature(st.params("auth-core")), reservedFirst+"pnpm-lock.yaml overlaps pnpm-lock.yaml")
		assertNothingRecorded(t, st, before)
	})
	t.Run("another split plan that shares the same paths starts alongside", func(t *testing.T) {
		st := newFeatureStart(t)
		st.startSharedAuthCore(t, shared)
		if res := startLock(t, st, "web/", "pnpm-lock.yaml, packages/"); res.Spawn.Outcome != SpawnOutcomeSpawned || res.Spawn.Err != nil {
			t.Fatalf("expected lock to start alongside, got %+v", res.Spawn)
		}
		if f := ActiveFeature(mustReadEvents(t, st.o), "lock"); f == nil || f.Shared != "packages/,pnpm-lock.yaml" {
			t.Fatalf("expected lock bound with its shared paths, got %+v", f)
		}
	})
}
