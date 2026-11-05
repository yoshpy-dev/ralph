package org

import "testing"

// A re-run of a seat already spawned is not refused by the driver_pool and
// model-name checks (plan 2026-11-05-org-role-models, D4 and D9; cross-review
// cycle 1): the CLI's ResolveRoleModel only decides the driver and the
// model, and ValidateSpawnEnvelope checks both after Spawn's idempotent
// return (AC-3), so the re-run returns the seat, while a new seat with the
// same request is still refused and recorded as rejected.

// TestOrgSpawn_RetryAfterDriverPoolNarrowed_ReturnsTheSeat: a codex seat
// re-run with the same --driver codex --model after driver_pool was
// narrowed to ["claude"] (and also with a driver marked as the role entry's)
// returns the existing seat with no new manifest event, no receipt, and no
// herdr or agmsg call. A new seat with the same request is refused, in the
// plain driver_pool wording for a driver from --driver and naming the
// role's [org.role_models] entry for a driver from it, real and dry-run.
func TestOrgSpawn_RetryAfterDriverPoolNarrowed_ReturnsTheSeat(t *testing.T) {
	o, h, a := codexGuardedOrg(t, implementerRole)
	if r := o.Spawn(mustCodexSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("first spawn = %+v, want spawned", r)
	}
	o.Config.DriverPool = []string{"claude"}
	events, receipts, herdrCalls, agmsgCalls := len(mustReadEvents(t, o)), receiptCount(t, o), len(h.calls), len(a.calls)

	for _, fromEntry := range []bool{false, true} {
		p := mustCodexSpawnParams("org-a", "seat-1")
		p.DriverFromRoleEntry = fromEntry
		if r := o.Spawn(p); r.Outcome != SpawnOutcomeIdempotent || r.Err != nil || r.Seat.Driver != "codex" {
			t.Fatalf("re-run of the codex seat (driver from the entry: %t) = %+v, want the seat returned idempotently", fromEntry, r)
		}
	}
	if n := len(mustReadEvents(t, o)); n != events {
		t.Errorf("events %d -> %d, want nothing written by the re-runs", events, n)
	}
	if n := receiptCount(t, o); n != receipts {
		t.Errorf("receipts %d -> %d, want nothing written by the re-runs", receipts, n)
	}
	if len(h.calls) != herdrCalls || len(a.calls) != agmsgCalls {
		t.Errorf("the re-runs called herdr %v and agmsg %v, want no call", h.calls[herdrCalls:], a.calls[agmsgCalls:])
	}
	assertSeatUnchanged(t, o, "org-a", "seat-1")

	const plainWording = `org: driver "codex" not in [org].driver_pool [claude]`
	const entryWording = `org: role "implementer" runs on driver "codex" in [org.role_models].implementer, which is not in [org].driver_pool [claude]: ` +
		"add codex to driver_pool, change the entry, or pass both --driver and --model"
	for _, tc := range []struct {
		name              string
		fromEntry, dryRun bool
		want              string
	}{
		{"a driver from --driver", false, false, plainWording},
		{"a driver from the role's entry", true, false, entryWording},
		{"a driver from --driver, dry run", false, true, plainWording},
		{"a driver from the role's entry, dry run", true, true, entryWording},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := mustCodexSpawnParams("org-a", "seat-2")
			p.DriverFromRoleEntry, p.DryRun = tc.fromEntry, tc.dryRun
			r := o.Spawn(p)
			if r.Err == nil || r.Err.Error() != tc.want {
				t.Fatalf("a new seat = %+v, want refused with %q", r, tc.want)
			}
			assertRejectedRecorded(t, o, r, "org-a", "seat-2")
		})
	}
}

// TestOrgSpawn_RetryWithAModelNotOneWord_ReturnsTheSeat: a re-run of a seat
// already spawned whose --model is not one word returns the seat as it was,
// writing nothing, since the model's characters are checked after the
// idempotent return; a new seat with that model is refused naming it.
func TestOrgSpawn_RetryWithAModelNotOneWord_ReturnsTheSeat(t *testing.T) {
	o, h, a := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("first spawn = %+v, want spawned", r)
	}
	events, receipts, herdrCalls, agmsgCalls := len(mustReadEvents(t, o)), receiptCount(t, o), len(h.calls), len(a.calls)

	p := mustSpawnParams("org-a", "seat-1")
	p.Model = "not a real model"
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeIdempotent || r.Err != nil || r.Seat.Model != "sonnet" {
		t.Fatalf("re-run with --model %q = %+v, want the sonnet seat returned idempotently", p.Model, r)
	}
	if n := len(mustReadEvents(t, o)); n != events {
		t.Errorf("events %d -> %d, want nothing written by the re-run", events, n)
	}
	if n := receiptCount(t, o); n != receipts {
		t.Errorf("receipts %d -> %d, want nothing written by the re-run", receipts, n)
	}
	if len(h.calls) != herdrCalls || len(a.calls) != agmsgCalls {
		t.Errorf("the re-run called herdr %v and agmsg %v, want no call", h.calls[herdrCalls:], a.calls[agmsgCalls:])
	}

	p = mustSpawnParams("org-a", "seat-2")
	p.Model = "not a real model"
	r := o.Spawn(p)
	assertRejectedRecorded(t, o, r, "org-a", "seat-2", `org: model "not a real model" must be ""`)
}
