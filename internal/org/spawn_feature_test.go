package org

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// Spawn with a split plan feature binding (plan
// docs/plans/active/2026-10-09-org-feature-worktree.md, AC6 and AC7 at the
// org layer). testFeature and boundReserveEvent live in reserve_test.go.

// featureLeaderParams is leaderParams for orgID's leader bound to f.
func featureLeaderParams(orgID string, f *FeatureBinding, reserve ...string) SpawnParams {
	p := leaderParams(orgID, reserve...)
	p.Feature = f
	return p
}

// spawnFeatureLeaderIn is spawnLeaderIn for a leader bound to f.
func spawnFeatureLeaderIn(t *testing.T, o *Org, h *fakeHerdr, orgID, workspace, pane string, f *FeatureBinding, reserve ...string) {
	t.Helper()
	h.workspaceID, h.paneID = workspace, pane
	if r := o.Spawn(featureLeaderParams(orgID, f, reserve...)); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn %s/%s bound to %s failed: %+v", orgID, LeaderIdentity, f.Feature, r)
	}
}

// assertActiveFeature checks that orgID's current binding is want (nil: none).
func assertActiveFeature(t *testing.T, o *Org, orgID string, want *FeatureBinding) {
	t.Helper()
	got := ActiveFeature(mustReadEvents(t, o), orgID)
	if (got == nil) != (want == nil) || (got != nil && *got != *want) {
		t.Fatalf("ActiveFeature(%s) = %+v, want %+v", orgID, got, want)
	}
}

// spawnSnapshot counts what a spawn can write or call.
type spawnSnapshot struct {
	events, receipts, herdr, agmsg int
}

func takeSpawnSnapshot(t *testing.T, o *Org, h *fakeHerdr, a *fakeAgmsg) spawnSnapshot {
	t.Helper()
	return spawnSnapshot{len(mustReadEvents(t, o)), receiptCount(t, o), len(h.calls), len(a.calls)}
}

// TestOrgSpawn_Feature_RecordedWithReservation covers plan AC3's record at
// the org layer: a leader spawn with Feature appends one scope_reserved
// carrying the paths and the binding tokens, with the worktree (cleaned) in
// its Worktree field, right before spawn_started, and that is the org's
// reservation and binding. The caller's FeatureBinding is not changed.
func TestOrgSpawn_Feature_RecordedWithReservation(t *testing.T) {
	o, _, _ := testOrg(t)
	f := testFeature("split-1", "auth", "0123456789ab")
	given := *f
	given.Worktree = "/repo/.claude/worktrees/../worktrees/org-auth/"
	if r := o.Spawn(featureLeaderParams("org-a", &given, "internal/auth/", "docs/")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn failed: %+v", r)
	}
	if given.Worktree != "/repo/.claude/worktrees/../worktrees/org-auth/" {
		t.Fatalf("expected the caller's binding unchanged, got worktree %q", given.Worktree)
	}
	events := mustReadEvents(t, o)
	want := "paths=docs/,internal/auth/ split=split-1 feature=auth digest=0123456789ab branch=feat/auth"
	if ev := events[0]; ev.Event != EventScopeReserved || ev.OrgID != "org-a" || ev.SeatID != "" || ev.DryRun ||
		ev.Details != want || ev.Worktree != f.Worktree {
		t.Fatalf("expected the bound scope_reserved first (Details %q, Worktree %q), got %+v", want, f.Worktree, ev)
	}
	if ev := events[1]; ev.Event != EventSpawnStarted || ev.SeatID != LeaderIdentity {
		t.Fatalf("expected the leader's spawn_started right after it, got %+v", ev)
	}
	if got := ActiveReservation(events, "org-a"); !slices.Equal(got, []string{"docs/", "internal/auth/"}) {
		t.Fatalf("ActiveReservation = %q, want docs/ and internal/auth/", got)
	}
	assertActiveFeature(t, o, "org-a", f)
	assertSeatUnchanged(t, o, "org-a", LeaderIdentity)
}

// TestOrgSpawn_Feature_InputRejectedBeforeAnyRecord: Feature without Reserve,
// Feature on a seat other than the leader, and a binding whose fields the
// record cannot hold are plain rejections before the manifest is read (no
// event, no receipt, no driver call), in real and dry-run mode alike.
func TestOrgSpawn_Feature_InputRejectedBeforeAnyRecord(t *testing.T) {
	f := testFeature("split-1", "auth", "0123456789ab")
	withField := func(change func(*FeatureBinding)) *FeatureBinding {
		g := *f
		change(&g)
		return &g
	}
	worker := mustSpawnParams("org-a", "seat-1")
	worker.Feature = f
	workerReserving := worker
	workerReserving.Reserve = []string{"internal/"}
	for _, tc := range []struct {
		name string
		p    SpawnParams
		want string
	}{
		{"leader without reserve paths", featureLeaderParams("org-a", f), `seat_id "leader" cannot be spawned for one without reserve paths`},
		{"a worker without reserve paths", worker, `seat_id "seat-1" cannot be spawned for one without reserve paths`},
		{"a worker with reserve paths", workerReserving, `only the leader seat reserves paths for its org; seat_id "seat-1" cannot`},
		{"an empty digest", featureLeaderParams("org-a", withField(func(g *FeatureBinding) { g.Digest = "" }), "internal/"), "feature's digest is empty"},
		{"a space in the split id", featureLeaderParams("org-a", withField(func(g *FeatureBinding) { g.Split = "a b" }), "internal/"), `split "a b" has whitespace`},
		{"an = in the branch", featureLeaderParams("org-a", withField(func(g *FeatureBinding) { g.Branch = "feat/x=y" }), "internal/"), `branch "feat/x=y" has whitespace, a comma, an =`},
		{"a relative worktree", featureLeaderParams("org-a", withField(func(g *FeatureBinding) { g.Worktree = "wt" }), "internal/"), `worktree "wt" is not an absolute path`},
		{"a bad reserve path comes first", featureLeaderParams("org-a", withField(func(g *FeatureBinding) { g.Digest = "" }), "/abs/"), "is absolute"},
	} {
		for _, dryRun := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s dry-run=%v", tc.name, dryRun), func(t *testing.T) {
				o, h, a := testOrg(t)
				p := tc.p
				p.DryRun = dryRun
				r := o.Spawn(p)
				if r.Outcome != SpawnOutcomeRejected || r.Err == nil || !strings.Contains(r.Err.Error(), tc.want) {
					t.Fatalf("expected a rejection containing %q, got %+v", tc.want, r)
				}
				if events := mustReadEvents(t, o); len(events) != 0 {
					t.Fatalf("expected no manifest event, got %+v", events)
				}
				if n := receiptCount(t, o); n != 0 {
					t.Fatalf("expected no receipt, got %d", n)
				}
				if len(h.calls) != 0 || len(a.calls) != 0 {
					t.Fatalf("expected no driver call, herdr %v agmsg %v", h.calls, a.calls)
				}
			})
		}
	}
}

// TestOrgSpawn_Feature_SameBindingAgainRecordsNothing covers plan AC6's
// re-run at the org layer: the same paths (however written) with the same
// binding pass with no new record, both for the active leader (the
// idempotent return: nothing written or called at all) and for a new leader
// after the first one stopped (a new saga, but no second scope_reserved).
func TestOrgSpawn_Feature_SameBindingAgainRecordsNothing(t *testing.T) {
	o, h, a := testOrg(t)
	f := testFeature("split-1", "auth", "0123456789ab")
	spawnFeatureLeaderIn(t, o, h, "org-a", "ws-a", "pane-1", f, "internal/auth/")

	before := takeSpawnSnapshot(t, o, h, a)
	same := *f
	if r := o.Spawn(featureLeaderParams("org-a", &same, "./internal//auth/")); r.Outcome != SpawnOutcomeIdempotent || r.Err != nil || r.Seat.SeatID != LeaderIdentity {
		t.Fatalf("expected the existing leader returned, got %+v", r)
	}
	if after := takeSpawnSnapshot(t, o, h, a); after != before {
		t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
	}

	if r := o.Stop(StopParams{OrgID: "org-a", Seat: LeaderIdentity}); r.Err != nil {
		t.Fatalf("stop org-a's leader: %v", r.Err)
	}
	spawnFeatureLeaderIn(t, o, h, "org-a", "ws-a", "pane-2", &same, "internal/auth/")
	if n := countString(eventNames(t, o), EventScopeReserved); n != 1 {
		t.Fatalf("expected the same binding to add no scope_reserved, got %d", n)
	}
	assertActiveFeature(t, o, "org-a", f)
}

// TestOrgSpawn_Feature_DifferentBindingRefused covers plan AC6's refusal at
// the org layer: another feature, or the same feature approved again (another
// digest), for an org bound to a feature is refused, as a plain rejection
// with nothing written while its leader is active, and through reject() (a
// `rejected` record) for a new leader after it stopped. The binding stays.
// After a disband the org takes the binding approved again.
func TestOrgSpawn_Feature_DifferentBindingRefused(t *testing.T) {
	f := testFeature("split-1", "auth", "0123456789ab")
	reapproved := testFeature("split-1", "auth", "ba9876543210")
	other := testFeature("split-1", "api", "0123456789ab")
	const boundTo = `org_id "org-a" is bound to feature "auth" of split plan "split-1" (digest "0123456789ab"`
	for _, tc := range []struct {
		name string
		f    *FeatureBinding
		want string
	}{
		{"another digest", reapproved, `for feature "auth" of split plan "split-1" (digest "ba9876543210"`},
		{"another feature", other, `for feature "api" of split plan "split-1"`},
	} {
		t.Run(tc.name+": active leader, nothing written", func(t *testing.T) {
			o, h, a := testOrg(t)
			spawnFeatureLeaderIn(t, o, h, "org-a", "ws-a", "pane-1", f, "internal/auth/")
			before := takeSpawnSnapshot(t, o, h, a)
			r := o.Spawn(featureLeaderParams("org-a", tc.f, "internal/auth/"))
			if r.Outcome != SpawnOutcomeRejected || r.Err == nil || !strings.Contains(r.Err.Error(), boundTo) ||
				!strings.Contains(r.Err.Error(), tc.want) || !strings.Contains(r.Err.Error(), "ralph org disband --org-id org-a first") {
				t.Fatalf("expected the refusal naming both bindings and the disband, got %+v", r)
			}
			if after := takeSpawnSnapshot(t, o, h, a); after != before {
				t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
			}
			assertSeatUnchanged(t, o, "org-a", LeaderIdentity)
			assertActiveFeature(t, o, "org-a", f)
		})
		t.Run(tc.name+": new leader, rejected recorded", func(t *testing.T) {
			o, h, _ := testOrg(t)
			spawnFeatureLeaderIn(t, o, h, "org-a", "ws-a", "pane-1", f, "internal/auth/")
			if r := o.Stop(StopParams{OrgID: "org-a", Seat: LeaderIdentity}); r.Err != nil {
				t.Fatalf("stop org-a's leader: %v", r.Err)
			}
			assertRejectedRecorded(t, o, o.Spawn(featureLeaderParams("org-a", tc.f, "internal/auth/")), "org-a", LeaderIdentity, boundTo, tc.want)
			assertActiveFeature(t, o, "org-a", f)
		})
	}
	t.Run("after a disband the binding approved again is taken", func(t *testing.T) {
		o, h, _ := testOrg(t)
		spawnFeatureLeaderIn(t, o, h, "org-a", "ws-a", "pane-1", f, "internal/auth/")
		if d := o.Disband(DisbandParams{OrgID: "org-a"}); len(d.Errs) != 0 || !d.Disbanded {
			t.Fatalf("disband org-a: %+v", d)
		}
		assertActiveFeature(t, o, "org-a", nil)
		spawnFeatureLeaderIn(t, o, h, "org-a", "ws-a2", "pane-2", reapproved, "internal/auth/")
		assertActiveFeature(t, o, "org-a", reapproved)
	})
}

// TestOrgSpawn_Feature_RunningUnboundOrgRefused covers plan AC7 at the org
// layer, for the three shapes of a running org that was not started for the
// feature: (a) a promoted session's leader's org (another seat active, no
// leader seat, no reservation) refuses a new bound leader through reject();
// (b) the org of `ralph org start <task>` (a leader spawned without a
// reservation) refuses the bound leader in idempotentRespawn, writing
// nothing; (c) an org with a plain --reserve refuses a binding for the same
// paths, from its active leader and from a new leader after it stopped. In
// every case the org holds no binding afterwards.
func TestOrgSpawn_Feature_RunningUnboundOrgRefused(t *testing.T) {
	f := testFeature("split-1", "auth", "0123456789ab")
	t.Run("(a) promoted leader's org", func(t *testing.T) {
		o, h, _ := testOrg(t)
		spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-1")
		assertRejectedRecorded(t, o, o.Spawn(featureLeaderParams("org-a", f, "internal/auth/")), "org-a", LeaderIdentity,
			`org_id "org-a" is running without a split plan`, `for feature "auth" of split plan "split-1"`,
			"use another org_id, or ralph org disband --org-id org-a first")
		events := mustReadEvents(t, o)
		if n := countString(eventNames(t, o), EventScopeReserved); n != 0 || ActiveReservation(events, "org-a") != nil {
			t.Fatalf("expected no reservation for org-a, got %d scope_reserved and %q", n, ActiveReservation(events, "org-a"))
		}
		assertActiveFeature(t, o, "org-a", nil)
	})
	t.Run("(b) ralph org start <task>'s org", func(t *testing.T) {
		o, h, a := testOrg(t)
		spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-1")
		before := takeSpawnSnapshot(t, o, h, a)
		r := o.Spawn(featureLeaderParams("org-a", f, "internal/auth/"))
		if r.Outcome != SpawnOutcomeRejected || r.Err == nil || !strings.Contains(r.Err.Error(), `org_id "org-a" is running without a split plan`) {
			t.Fatalf("expected the refusal for a running unbound org, got %+v", r)
		}
		if after := takeSpawnSnapshot(t, o, h, a); after != before {
			t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
		}
		assertSeatUnchanged(t, o, "org-a", LeaderIdentity)
		assertActiveFeature(t, o, "org-a", nil)
	})
	const plain = `org_id "org-a" was started without a split plan (it reserves internal/auth/ with a plain --reserve)`
	t.Run("(c) plain --reserve org, active leader", func(t *testing.T) {
		o, h, a := testOrg(t)
		spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-1", "internal/auth/")
		before := takeSpawnSnapshot(t, o, h, a)
		r := o.Spawn(featureLeaderParams("org-a", f, "internal/auth/"))
		if r.Outcome != SpawnOutcomeRejected || r.Err == nil || !strings.Contains(r.Err.Error(), plain) {
			t.Fatalf("expected the refusal for a plain reservation, got %+v", r)
		}
		if after := takeSpawnSnapshot(t, o, h, a); after != before {
			t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
		}
		assertActiveFeature(t, o, "org-a", nil)
	})
	t.Run("(c) plain --reserve org, new leader", func(t *testing.T) {
		o, h, _ := testOrg(t)
		spawnLeaderIn(t, o, h, "org-a", "ws-a", "pane-1", "internal/auth/")
		if r := o.Stop(StopParams{OrgID: "org-a", Seat: LeaderIdentity}); r.Err != nil {
			t.Fatalf("stop org-a's leader: %v", r.Err)
		}
		assertRejectedRecorded(t, o, o.Spawn(featureLeaderParams("org-a", f, "internal/auth/")), "org-a", LeaderIdentity, plain)
		assertActiveFeature(t, o, "org-a", nil)
	})
}

// TestOrgSpawn_Feature_PlainReserveIntoBoundOrgRefused covers the last
// sentence of plan AC7: a leader spawn with a plain --reserve into an org
// bound to a feature is refused even for the same paths, from the active
// leader with nothing written and from a new leader through reject(). A seat
// spawned without Reserve is not affected, and the binding stays.
func TestOrgSpawn_Feature_PlainReserveIntoBoundOrgRefused(t *testing.T) {
	f := testFeature("split-1", "auth", "0123456789ab")
	const want = "so a reservation without a split plan (internal/auth/) is refused"
	o, h, a := testOrg(t)
	spawnFeatureLeaderIn(t, o, h, "org-a", "ws-a", "pane-1", f, "internal/auth/")

	before := takeSpawnSnapshot(t, o, h, a)
	if r := o.Spawn(leaderParams("org-a", "internal/auth/")); r.Outcome != SpawnOutcomeRejected || r.Err == nil || !strings.Contains(r.Err.Error(), want) {
		t.Fatalf("expected a plain reservation into the bound org refused, got %+v", r)
	}
	if after := takeSpawnSnapshot(t, o, h, a); after != before {
		t.Fatalf("expected nothing recorded or called, %+v -> %+v", before, after)
	}

	if r := o.Stop(StopParams{OrgID: "org-a", Seat: LeaderIdentity}); r.Err != nil {
		t.Fatalf("stop org-a's leader: %v", r.Err)
	}
	assertRejectedRecorded(t, o, o.Spawn(leaderParams("org-a", "internal/auth/")), "org-a", LeaderIdentity,
		`org_id "org-a" is bound to feature "auth" of split plan "split-1"`, want)
	spawnSeatIn(t, o, h, "org-a", "seat-1", "ws-a", "pane-2")
	assertActiveFeature(t, o, "org-a", f)
}

// TestOrgSpawn_Feature_DryRunPredictsBinding: a dry run decides the binding
// against the real records and predicts the record the real spawn would
// write, as a dry-run scope_reserved carrying the binding and the worktree,
// which holds nothing. A dry run with another binding into a bound org is a
// dry-run `rejected`.
func TestOrgSpawn_Feature_DryRunPredictsBinding(t *testing.T) {
	o, h, _ := testOrg(t)
	f := testFeature("split-1", "auth", "0123456789ab")
	dryRun := func(orgID string, f *FeatureBinding, reserve ...string) SpawnResult {
		p := featureLeaderParams(orgID, f, reserve...)
		p.DryRun = true
		return o.Spawn(p)
	}

	if r := dryRun("org-a", f, "internal/auth/"); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("expected the dry run to pass, got %+v", r)
	}
	events := mustReadEvents(t, o)
	want := "paths=internal/auth/ split=split-1 feature=auth digest=0123456789ab branch=feat/auth"
	if ev := events[0]; ev.Event != EventScopeReserved || !ev.DryRun || ev.OrgID != "org-a" || ev.Details != want || ev.Worktree != f.Worktree {
		t.Fatalf("expected the dry-run trail to start with the bound dry-run scope_reserved, got %+v", ev)
	}
	if ev := events[1]; ev.Event != EventSpawnStarted || !ev.DryRun {
		t.Fatalf("expected the dry-run spawn_started next, got %+v", ev)
	}
	if got := RunningOrgs(events); len(got) != 0 {
		t.Fatalf("RunningOrgs = %v, want none", got)
	}
	assertActiveFeature(t, o, "org-a", nil)

	spawnFeatureLeaderIn(t, o, h, "org-b", "ws-b", "pane-b", f, "docs/")
	reapproved := testFeature("split-1", "auth", "ba9876543210")
	if r := dryRun("org-b", reapproved, "docs/"); r.Outcome != SpawnOutcomeRejected || r.Err == nil || !strings.Contains(r.Err.Error(), `org_id "org-b" is bound to`) {
		t.Fatalf("expected the dry run rejected for another digest, got %+v", r)
	}
	if ev := lastEvent(t, o); ev.Event != EventRejected || !ev.DryRun || ev.OrgID != "org-b" {
		t.Fatalf("expected a dry-run rejected event for org-b, got %+v", ev)
	}
	assertActiveFeature(t, o, "org-b", f)
}
