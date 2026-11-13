package org

import (
	"strings"
	"testing"
)

// A reseat's start that comes back idempotent (plan
// docs/plans/active/2026-11-12-org-reseat.md, P6 and FR-6; cross-review
// cycle 2, finding 1): another command spawned the seat after the reseat
// stopped it, so the start returns the seat that command started. On the
// fixture of reseat_test.go (newReseatOrg).

// intrudeBeforeStart sets reseatTestHooks.beforeStart, for the rest of t, to
// run start once, right before the reseat starts seat.
func intrudeBeforeStart(t *testing.T, seat string, start func()) {
	t.Helper()
	done := false
	reseatTestHooks.beforeStart = func(id string) {
		if id == seat && !done {
			done = true
			start()
		}
	}
	t.Cleanup(func() { reseatTestHooks.beforeStart = nil })
}

// TestReseat_ASeatAnotherCommandStartedOnOtherSettings covers a seat another
// command spawned on its old driver and model after the reseat stopped it to
// start it on codex: the reseat's start comes back idempotent on that seat,
// which runs on claude opus. The reseat reports the seat start_failed with an
// error naming both settings, records it so, gives no DECISION, and leaves
// the other command's seat running. Having been given --driver and --model
// for the seat, it does not set Changed: the retry it prints carries --driver
// and --model and, run as it is, stops that seat and starts it on codex.
func TestReseat_ASeatAnotherCommandStartedOnOtherSettings(t *testing.T) {
	st := newReseatOrg(t)
	st.o.Config.Permissions.CodexVerified = true
	intrudeBeforeStart(t, implementerSeatID, func() { st.spawnSeat(t, implementerSeatID, implementerRole, "opus") })
	from := len(mustReadEvents(t, st.o))
	p := st.reseatParams(implementerSeatID)
	p.Driver, p.Model = "codex", "gpt-x"

	res := st.o.Reseat(p)
	if got := reseatOutcomes(res); got != "implementer:start_failed" {
		t.Fatalf("seats %s (error %v); want the implementer's start failed", got, res.Err)
	}
	want := `org: reseat ` + res.Plan.ID + ` of org_id "auth-core": seat "implementer" was started by another command ` +
		`during the reseat, on role implementer, driver claude and model opus, not the role implementer, driver codex ` +
		`and model gpt-x the reseat meant to start it on, so the reseat left it running as it is and started no seat after it`
	if res.Err == nil || res.Err.Error() != want {
		t.Errorf("Reseat error %v; want %q", res.Err, want)
	}
	if res.Changed != "" {
		t.Errorf("changed %q; want none, the reseat having been given --driver and --model for the implementer", res.Changed)
	}
	if want := "ralph org reseat --org-id auth-core --seat implementer --driver codex --model gpt-x"; res.Retry != want {
		t.Errorf("retry %q; want %q", res.Retry, want)
	}
	if res.Decision != "" {
		t.Errorf("Decision = %q; want none, no seat having started", res.Decision)
	}
	assertReseated(t, mustReadEvents(t, st.o)[from:], "id="+res.Plan.ID+" seats=implementer:start_failed changed=implementer:codex/gpt-x")
	impl := eventOf(t, st.o, "auth-core", implementerSeatID, EventSpawned)
	if impl.Driver != "claude" || impl.Model != "opus" || strings.Contains(impl.Details, "reseat=") {
		t.Errorf("implementer spawned %+v; want the other command's seat on claude opus, with no reseat note", impl)
	}

	again := st.mustReseat(t, st.retryParams(t, res.Retry))
	if len(again.Plan.Stop) != 1 || again.Plan.Stop[0].SeatID != implementerSeatID {
		t.Errorf("the retry stopped %q; want the other command's implementer", seatIDs(again.Plan.Stop))
	}
	impl = eventOf(t, st.o, "auth-core", implementerSeatID, EventSpawned)
	if impl.Driver != "codex" || impl.Model != "gpt-x" || !strings.HasSuffix(impl.Details, "reseat="+again.Plan.ID) {
		t.Errorf("implementer spawned %+v after the retry; want the retry's start on codex gpt-x", impl)
	}
	if want := ReseatDecisionCommand("auth-core", ReseatDecisionText(again.Plan.ID, []string{implementerSeatID}), ""); again.Decision != want {
		t.Errorf("the retry's Decision = %q; want %q", again.Decision, want)
	}
}

// TestReseat_ASeatAnotherCommandStartedOnTheSameSettings covers the same
// intrusion on the role, driver and model the reseat planned: the start that
// comes back idempotent counts as started, and the DECISION names the seat.
func TestReseat_ASeatAnotherCommandStartedOnTheSameSettings(t *testing.T) {
	t.Run("the reviewer on the driver CLI's default model", func(t *testing.T) {
		st := newReseatOrg(t)
		intrudeBeforeStart(t, "reviewer", func() { st.spawnSeat(t, "reviewer", "reviewer", "") })
		from := len(mustReadEvents(t, st.o))

		res := st.mustReseat(t, st.reseatParams("reviewer"))
		if res.Retry != "" {
			t.Errorf("retry %q; want none", res.Retry)
		}
		if want := ReseatDecisionCommand("auth-core", ReseatDecisionText(res.Plan.ID, []string{"reviewer"}), ""); res.Decision != want {
			t.Errorf("Decision = %q; want %q", res.Decision, want)
		}
		added := mustReadEvents(t, st.o)[from:]
		assertReseated(t, added, "id="+res.Plan.ID+" seats=reviewer:started")
		if spawned := eventsNamed(added, EventSpawned); len(spawned) != 1 || spawned[0].Model != "" || strings.Contains(spawned[0].Details, "reseat=") {
			t.Errorf("spawned %+v; want only the other command's reviewer, on the CLI's default model", spawned)
		}
	})

	t.Run("the implementer on the --driver and --model given", func(t *testing.T) {
		st := newReseatOrg(t)
		st.o.Config.Permissions.CodexVerified = true
		intrudeBeforeStart(t, implementerSeatID, func() {
			r := st.o.Spawn(SpawnParams{
				OrgID: "auth-core", SeatID: implementerSeatID, Role: implementerRole, Driver: "codex", Model: "gpt-x",
				Cwd: st.worktree("auth-core"), Scope: reseatScope, TimeoutMS: 5000,
			})
			if r.Outcome != SpawnOutcomeSpawned {
				t.Errorf("the other command's spawn of the implementer: %+v", r)
			}
		})
		p := st.reseatParams(implementerSeatID)
		p.Driver, p.Model = "codex", "gpt-x"

		res := st.mustReseat(t, p)
		assertReseated(t, mustReadEvents(t, st.o), "id="+res.Plan.ID+" seats=implementer:started changed=implementer:codex/gpt-x")
		if want := ReseatDecisionCommand("auth-core", ReseatDecisionText(res.Plan.ID, []string{implementerSeatID}), ""); res.Decision != want {
			t.Errorf("Decision = %q; want %q", res.Decision, want)
		}
		if impl := eventOf(t, st.o, "auth-core", implementerSeatID, EventSpawned); strings.Contains(impl.Details, "reseat=") {
			t.Errorf("implementer spawned %+v; want the other command's seat, with no reseat note", impl)
		}
	})
}

// TestReseat_TheLeaderAnotherCommandStartedOnAnotherModel covers the leader,
// which the reseat starts with StartFeature: another start of the feature
// org spawns the leader on opus after the reseat stopped it to start it
// again on sonnet, so the reseat's start comes back idempotent on opus and
// the leader is start_failed, with no DECISION. Not given --driver and
// --model, the reseat sets Changed to the leader and leaves it out of the
// retry, which is none: the leader was its only target.
func TestReseat_TheLeaderAnotherCommandStartedOnAnotherModel(t *testing.T) {
	st := newReseatOrg(t)
	intrudeBeforeStart(t, LeaderIdentity, func() {
		p := st.params("auth-core")
		p.Model = "opus"
		st.mustStart(t, p, SpawnOutcomeSpawned)
	})
	from := len(mustReadEvents(t, st.o))

	res := st.o.Reseat(st.reseatParams(LeaderIdentity))
	if got := reseatOutcomes(res); got != "leader:start_failed" {
		t.Fatalf("seats %s (error %v); want the leader's start failed", got, res.Err)
	}
	want := `seat "leader" was started by another command during the reseat, on role leader, driver claude and model opus, ` +
		`not the role leader, driver claude and model sonnet the reseat meant to start it on`
	if res.Err == nil || !strings.Contains(res.Err.Error(), want) {
		t.Errorf("Reseat error %v; want it to contain %q", res.Err, want)
	}
	if res.Changed != LeaderIdentity || res.Retry != "" || res.Decision != "" {
		t.Errorf("changed %q retry %q decision %q; want the leader changed, and no retry or DECISION", res.Changed, res.Retry, res.Decision)
	}
	assertReseated(t, mustReadEvents(t, st.o)[from:], "id="+res.Plan.ID+" seats=leader:start_failed")
	if leader := eventOf(t, st.o, "auth-core", LeaderIdentity, EventSpawned); leader.Model != "opus" {
		t.Errorf("leader spawned %+v; want the other start's leader on opus", leader)
	}
}

// TestReseat_ASeatAnotherCommandStartedIsLeftOutOfTheRetry covers the same
// intrusion on a seat other than the leader, in a reseat not given --driver
// and --model: another command spawns the implementer on claude sonnet after
// the reseat stopped it and the reviewer to start them again on what the
// ledger had, so the implementer's start comes back idempotent on sonnet, not
// opus. The implementer is start_failed and Changed, the reviewer is left
// stopped, and the retry names the reviewer only. Run as it is, the retry
// starts the reviewer and stops nothing, so the other command's implementer
// keeps running.
func TestReseat_ASeatAnotherCommandStartedIsLeftOutOfTheRetry(t *testing.T) {
	st := newReseatOrg(t)
	intrudeBeforeStart(t, implementerSeatID, func() { st.spawnSeat(t, implementerSeatID, implementerRole, "sonnet") })
	from := len(mustReadEvents(t, st.o))

	res := st.o.Reseat(st.reseatParams(implementerSeatID, "reviewer"))
	if got := reseatOutcomes(res); got != "implementer:start_failed,reviewer:stopped" {
		t.Fatalf("seats %s (error %v); want the implementer's start failed and the reviewer left stopped", got, res.Err)
	}
	want := `seat "implementer" was started by another command during the reseat, on role implementer, driver claude and ` +
		`model sonnet, not the role implementer, driver claude and model opus the reseat meant to start it on`
	if res.Err == nil || !strings.Contains(res.Err.Error(), want) {
		t.Errorf("Reseat error %v; want it to contain %q", res.Err, want)
	}
	if res.Changed != implementerSeatID {
		t.Errorf("changed %q; want the implementer", res.Changed)
	}
	if want := "ralph org reseat --org-id auth-core --seat reviewer"; res.Retry != want || res.Decision != "" {
		t.Errorf("retry %q decision %q; want %q, without the implementer, and no DECISION", res.Retry, res.Decision, want)
	}
	assertReseated(t, mustReadEvents(t, st.o)[from:], "id="+res.Plan.ID+" seats=implementer:start_failed,reviewer:stopped")

	again := st.mustReseat(t, st.retryParams(t, res.Retry))
	if len(again.Plan.Stop) != 0 || reseatOutcomes(again) != "reviewer:started" {
		t.Errorf("the retry stopped %q and did %s; want no stop and the reviewer started", seatIDs(again.Plan.Stop), reseatOutcomes(again))
	}
	if impl := eventOf(t, st.o, "auth-core", implementerSeatID, EventSpawned); impl.Model != "sonnet" || strings.Contains(impl.Details, "reseat=") {
		t.Errorf("implementer spawned %+v after the retry; want the other command's seat on sonnet, with no reseat note", impl)
	}
}

// TestReseatIdempotentMismatchErr pins the comparison of each of the role,
// the driver and the model, the model "" equal to "" and shown as the CLI's
// default.
func TestReseatIdempotentMismatchErr(t *testing.T) {
	plan := ReseatPlan{ID: "20261113T010203-abcd", OrgID: "auth-core"}
	seat := ReseatSeat{SeatID: "reviewer", Role: "reviewer", Driver: "claude"}
	const prefix = `org: reseat 20261113T010203-abcd of org_id "auth-core": seat "reviewer" was started by another command during the reseat, on `
	const planned = ", not the role reviewer, driver claude and model (CLI default) the reseat meant to start it on, " +
		"so the reseat left it running as it is and started no seat after it"
	for _, tc := range []struct {
		name string
		got  SeatStatus
		want string // "" for no error
	}{
		{"the same, the model \"\" on both sides", SeatStatus{Role: "reviewer", Driver: "claude"}, ""},
		{"another role", SeatStatus{Role: "implementer", Driver: "claude"}, "role implementer, driver claude and model (CLI default)"},
		{"another driver", SeatStatus{Role: "reviewer", Driver: "codex"}, "role reviewer, driver codex and model (CLI default)"},
		{"another model", SeatStatus{Role: "reviewer", Driver: "claude", Model: "opus"}, "role reviewer, driver claude and model opus"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := reseatIdempotentMismatchErr(plan, seat, tc.got)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("reseatIdempotentMismatchErr = %v; want nil", err)
				}
				return
			}
			if want := prefix + tc.want + planned; err == nil || err.Error() != want {
				t.Fatalf("reseatIdempotentMismatchErr = %v; want %q", err, want)
			}
		})
	}
}
