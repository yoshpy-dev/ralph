package org

import (
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// Spawn and Stop under [org.role_models] (plan 2026-11-05-org-role-models,
// D5 and D7; AC6 and AC8): the refusal of a config that still sets
// [org.roles], and a seat started with no model, which passes no --model,
// marks its spawn_started, and gets a codex receipt honored unknown with the
// observed model, at spawn or at stop.

// withLegacyKey records in o's config that path sets key, as config.Load
// records an [org].model_pool or [org.roles] it finds.
func withLegacyKey(o *Org, key, path string) {
	o.Config.LegacyModelKeys = append(o.Config.LegacyModelKeys, config.OrgLegacyModelKey{Key: key, Path: path})
}

// TestOrgSpawn_LegacyRolesTable_RefusedBeforeAnyWrite pins AC8's refusal at
// the org layer: a config that still sets [org.roles] refuses a new seat,
// real and dry-run alike, with an error naming the file, as a plain
// rejection: no herdr or agmsg call, no manifest event, no receipt.
func TestOrgSpawn_LegacyRolesTable_RefusedBeforeAnyWrite(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		mode := "real"
		if dryRun {
			mode = "dry-run"
		}
		t.Run(mode, func(t *testing.T) {
			o, h, a := testOrg(t)
			withLegacyKey(o, config.OrgLegacyRolesKey, "/repo/ralph.local.toml")
			p := mustSpawnParams("org-a", "seat-1")
			p.DryRun = dryRun

			result := o.Spawn(p)

			if result.Outcome != SpawnOutcomeRejected || result.Err == nil {
				t.Fatalf("Outcome = %v, Err = %v; want SpawnOutcomeRejected with an error", result.Outcome, result.Err)
			}
			for _, want := range []string{"/repo/ralph.local.toml sets [org.roles]", "no longer restricts a role's models", "[org.role_models]"} {
				if !strings.Contains(result.Err.Error(), want) {
					t.Errorf("error %q should contain %q", result.Err.Error(), want)
				}
			}
			if len(h.calls) != 0 || len(a.calls) != 0 {
				t.Errorf("expected no driver calls, got herdr=%v agmsg=%v", h.calls, a.calls)
			}
			if got := eventNames(t, o); len(got) != 0 {
				t.Errorf("expected no manifest event, got %v", got)
			}
			if receiptCount(t, o) != 0 || result.ModelReceipt != (Receipt{}) {
				t.Errorf("expected no receipt, got %d and %+v", receiptCount(t, o), result.ModelReceipt)
			}
		})
	}
}

// TestOrgSpawn_LegacyRolesTable_AfterTheSpawnKeepsTheRetryIdempotent pins
// the refusal's position on the real path (D7): it runs after the
// idempotent early return, like the retired role keys, so an [org.roles]
// added after a seat was spawned does not turn a re-run of that seat into a
// rejection, while a new seat is still refused.
func TestOrgSpawn_LegacyRolesTable_AfterTheSpawnKeepsTheRetryIdempotent(t *testing.T) {
	o, _, _ := testOrg(t)
	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("first spawn: %+v", r)
	}
	withLegacyKey(o, config.OrgLegacyRolesKey, "/repo/ralph.toml")
	events, receipts := len(mustReadEvents(t, o)), receiptCount(t, o)

	if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeIdempotent || r.Err != nil {
		t.Fatalf("re-run of the spawned seat = %v (err=%v), want SpawnOutcomeIdempotent", r.Outcome, r.Err)
	}
	r := o.Spawn(mustSpawnParams("org-a", "seat-2"))
	if r.Outcome != SpawnOutcomeRejected || r.Err == nil || !strings.Contains(r.Err.Error(), "/repo/ralph.toml sets [org.roles]") {
		t.Fatalf("a new seat = %+v, want the [org.roles] refusal", r)
	}
	if n := len(mustReadEvents(t, o)); n != events {
		t.Errorf("events %d -> %d, want nothing written", events, n)
	}
	if n := receiptCount(t, o); n != receipts {
		t.Errorf("receipts %d -> %d, want nothing written", receipts, n)
	}
}

// TestOrgSpawn_LegacyModelPoolRecord_DoesNotRefuse: a config that still sets
// [org].model_pool only is not refused (spec FR-6: the CLI writes one line
// and goes on), real or dry-run.
func TestOrgSpawn_LegacyModelPoolRecord_DoesNotRefuse(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		o, _, _ := testOrg(t)
		withLegacyKey(o, config.OrgLegacyModelPoolKey, "/repo/ralph.toml")
		p := mustSpawnParams("org-a", "seat-1")
		p.DryRun = dryRun
		if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned || r.Err != nil {
			t.Fatalf("dry_run=%t: Spawn = %+v, want spawned", dryRun, r)
		}
	}
}

// TestOrgSpawn_LegacyRolesTable_ComesBeforeRetiredRoleKeys: with both an
// [org.roles] table and a renamed role's old name as a key, the [org.roles]
// refusal is the one returned (retiredRoleConfigErr checks it first).
func TestOrgSpawn_LegacyRolesTable_ComesBeforeRetiredRoleKeys(t *testing.T) {
	o, _, _ := testOrg(t)
	withLegacyKey(o, config.OrgLegacyRolesKey, "/repo/ralph.toml")
	o.Config.RoleModels[oldLeaderName] = config.OrgRoleModel{Driver: "claude", Model: "sonnet"}
	r := o.Spawn(mustSpawnParams("org-a", "seat-1"))
	if r.Outcome != SpawnOutcomeRejected || r.Err == nil || !strings.Contains(r.Err.Error(), "sets [org.roles]") || strings.Contains(r.Err.Error(), "retired role key") {
		t.Fatalf("Spawn = %+v, want the [org.roles] refusal alone", r)
	}
}

// TestOrgSpawn_EmptyModel_PassesNoModelFlagAndMarksSpawnStarted pins D5 for
// a claude seat: a model of "" spawns with no --model in the agent's
// arguments, and its spawn_started (alone of its events) carries
// model_cli_default, on disk as JSON, while a seat with a model passes
// --model and leaves the key out. The receipt records the commanded model as
// "", honored unknown.
func TestOrgSpawn_EmptyModel_PassesNoModelFlagAndMarksSpawnStarted(t *testing.T) {
	for _, tc := range []struct {
		name      string
		model     string
		wantFlag  bool
		wantMarks int
	}{
		{"no model", "", false, 1},
		{"a model", "sonnet", true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h, _ := testOrg(t)
			p := mustSpawnParams("org-a", "seat-1")
			p.Model = tc.model

			r := o.Spawn(p)
			if r.Outcome != SpawnOutcomeSpawned || r.Err != nil {
				t.Fatalf("Spawn = %+v, want spawned", r)
			}
			if len(h.agentStartArgs) != 1 {
				t.Fatalf("expected one agent start, got %d", len(h.agentStartArgs))
			}
			args := h.agentStartArgs[0]
			if got := slices.Contains(args, "--model"); got != tc.wantFlag {
				t.Fatalf("agent args %q: --model present = %t, want %t", args, got, tc.wantFlag)
			}
			if tc.wantFlag {
				if i := slices.Index(args, "--model"); i+1 >= len(args) || args[i+1] != tc.model {
					t.Fatalf("agent args %q: want --model %s", args, tc.model)
				}
			}
			if slices.Contains(args, "") {
				t.Fatalf("agent args %q hold an empty argument", args)
			}

			marks := 0
			for _, ev := range mustReadEvents(t, o) {
				if ev.ModelCLIDefault {
					marks++
					if ev.Event != EventSpawnStarted || ev.Model != "" {
						t.Errorf("model_cli_default on %+v, want it on the spawn_started only", ev)
					}
				}
			}
			if marks != tc.wantMarks {
				t.Fatalf("%d events carry model_cli_default, want %d", marks, tc.wantMarks)
			}
			raw, err := os.ReadFile(o.Manifest.Path())
			if err != nil {
				t.Fatalf("read the manifest: %v", err)
			}
			if got := strings.Count(string(raw), `"model_cli_default":true`); got != tc.wantMarks {
				t.Fatalf("the manifest holds %d model_cli_default keys, want %d:\n%s", got, tc.wantMarks, raw)
			}

			if r.ModelReceipt.CommandedModel != tc.model || r.ModelReceipt.Honored != HonoredUnknown {
				t.Fatalf("receipt = %+v, want commanded %q, honored unknown", r.ModelReceipt, tc.model)
			}
		})
	}
}

// TestOrgSpawn_DryRun_EmptyModel_MarksSpawnStarted: the dry-run trail marks
// its spawn_started the way the real spawn does.
func TestOrgSpawn_DryRun_EmptyModel_MarksSpawnStarted(t *testing.T) {
	o, h, _ := testOrg(t)
	p := mustSpawnParams("org-a", "seat-1")
	p.Model = ""
	p.DryRun = true
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned || !r.Seat.DryRun {
		t.Fatalf("dry-run Spawn = %+v, want a dry-run spawned", r)
	}
	if len(h.calls) != 0 {
		t.Fatalf("a dry run called herdr: %v", h.calls)
	}
	started := eventsNamed(mustReadEvents(t, o), EventSpawnStarted)
	if len(started) != 1 || !started[0].DryRun || !started[0].ModelCLIDefault {
		t.Fatalf("dry-run spawn_started = %+v, want one marked model_cli_default", started)
	}
}

// TestOrgSpawn_Codex_EmptyModel_ReceiptUnknownWithObservedModel pins D5's
// codex receipt at spawn: with no model commanded there is nothing to honor,
// so a found session record gives honored unknown with the observed model
// recorded and named in the reason, not honored true or false; and the seat
// is started with no --model.
func TestOrgSpawn_Codex_EmptyModel_ReceiptUnknownWithObservedModel(t *testing.T) {
	o, h, _ := codexGuardedOrg(t, "implementer")
	o.CodexModelObserveTimeout = testCodexObserveGenerousBudget
	p := mustCodexSpawnParams("org-a", "seat-1")
	p.Model = ""
	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, time.Now().Add(time.Second), "gpt-6.1-sol")

	r := o.Spawn(p)
	if r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("Spawn = %+v, want spawned", r)
	}
	rc := r.ModelReceipt
	if rc.Honored != HonoredUnknown || rc.CommandedModel != "" || rc.ReportedEffectiveModel != "gpt-6.1-sol" {
		t.Fatalf("receipt = %+v, want honored unknown, commanded \"\", reported gpt-6.1-sol", rc)
	}
	for _, want := range []string{"no model was commanded", "gpt-6.1-sol"} {
		if !strings.Contains(rc.Reason, want) {
			t.Errorf("reason %q should contain %q", rc.Reason, want)
		}
	}
	if slices.Contains(h.agentStartArgs[0], "--model") {
		t.Fatalf("agent args %q pass --model for a seat with no model", h.agentStartArgs[0])
	}
}

// TestOrgStop_Codex_EmptyModel_ObservedAtStop pins D5 at stop for a codex
// seat started with no model: its spawn_started is marked model_cli_default,
// so when the spawn's own observation found nothing and the session record
// appears afterwards, Stop observes it and appends a receipt honored unknown
// with the observed model, and records model_observed=unknown.
func TestOrgStop_Codex_EmptyModel_ObservedAtStop(t *testing.T) {
	o, _, _ := codexGuardedOrg(t, "implementer")
	p := mustCodexSpawnParams("org-a", "seat-1")
	p.Model = ""
	r := o.Spawn(p)
	if r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("Spawn = %+v, want spawned", r)
	}
	if r.ModelReceipt.Honored != HonoredUnknown || r.ModelReceipt.ReportedEffectiveModel != "" {
		t.Fatalf("spawn receipt = %+v, want unknown with no reported model (no session record yet)", r.ModelReceipt)
	}
	raiseCodexObserveBudgetForStop(o)
	promptPath, err := o.promptFilePath(p.OrgID, p.SeatID)
	if err != nil {
		t.Fatalf("promptFilePath: %v", err)
	}
	writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, time.Now().Add(time.Second), "gpt-6.1-sol")

	res := o.Stop(StopParams{OrgID: "org-a", Seat: "seat-1"})
	if res.Err != nil {
		t.Fatalf("Stop: %v", res.Err)
	}
	rc := res.ModelReceipt
	if rc.Honored != HonoredUnknown || rc.CommandedModel != "" || rc.ReportedEffectiveModel != "gpt-6.1-sol" {
		t.Fatalf("stop receipt = %+v, want honored unknown, commanded \"\", reported gpt-6.1-sol", rc)
	}
	for _, want := range []string{"no model was commanded", "(observed at stop)"} {
		if !strings.Contains(rc.Reason, want) {
			t.Errorf("reason %q should contain %q", rc.Reason, want)
		}
	}
	if n := receiptCount(t, o); n != 2 {
		t.Fatalf("receipts = %d, want 2 (spawn's unknown and stop's observed)", n)
	}
	assertDetailsContains(t, lastEvent(t, o).Details, "model_observed=unknown")
}

// TestOrgStop_Codex_EmptyModel_MarkDecidesTheObservation contrasts the two
// spawn_started events with no model on the same seeded trail: one marked
// model_cli_default (a seat started with no model) is observed at stop, and
// one without the mark (recorded before the model field existed) is not,
// as TestOrgStop_Codex_EmptyModelOnCorrelatedSpawn_NothingObserved pins.
func TestOrgStop_Codex_EmptyModel_MarkDecidesTheObservation(t *testing.T) {
	for _, tc := range []struct {
		name        string
		mark        bool
		wantHonored string
		wantToken   string
	}{
		{"marked", true, HonoredUnknown, "model_observed=unknown"},
		{"not marked", false, "", "model_observed=none"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, _, _ := codexGuardedOrg(t, "implementer")
			raiseCodexObserveBudgetForStop(o)
			const orgID, seatID = "org-a", "seat-1"
			promptPath, err := o.promptFilePath(orgID, seatID)
			if err != nil {
				t.Fatalf("promptFilePath: %v", err)
			}
			spawnStartedAt := time.Now()
			appendInterruptedAgentStartedTrail(t, o, ManifestEvent{
				OrgID: orgID, SeatID: seatID, Role: "implementer", Driver: "codex", ModelCLIDefault: tc.mark,
			}, promptPath, spawnStartedAt)
			writeQualifyingCodexFixture(t, o.CodexSessionsDir, promptPath, spawnStartedAt.Add(time.Second), "gpt-6.1-sol")

			res := o.Stop(StopParams{OrgID: orgID, Seat: seatID})
			if res.Err != nil {
				t.Fatalf("Stop: %v", res.Err)
			}
			if res.ModelReceipt.Honored != tc.wantHonored {
				t.Fatalf("stop receipt = %+v, want honored %q", res.ModelReceipt, tc.wantHonored)
			}
			if tc.mark && res.ModelReceipt.ReportedEffectiveModel != "gpt-6.1-sol" {
				t.Fatalf("stop receipt = %+v, want the observed gpt-6.1-sol", res.ModelReceipt)
			}
			if !tc.mark && res.ModelReceipt != (Receipt{}) {
				t.Fatalf("stop receipt = %+v, want none for an old event", res.ModelReceipt)
			}
			assertDetailsContains(t, lastEvent(t, o).Details, tc.wantToken)
		})
	}
}

// TestCodexFoundReceipt_NoCommandedModel pins codexFoundReceipt's third
// outcome next to its two others: a commanded model of "" gives honored
// unknown with the observed model, while a commanded model gives true or
// false as before.
func TestCodexFoundReceipt_NoCommandedModel(t *testing.T) {
	base := Receipt{OrgID: "org-a", SeatID: "seat-1", Driver: "codex"}
	for _, tc := range []struct {
		commanded, effective, wantHonored string
	}{
		{"", "gpt-6.1-sol", HonoredUnknown},
		{"gpt-6.1-sol", "gpt-6.1-sol", HonoredTrue},
		{"gpt-6-astra", "gpt-6.1-sol", HonoredFalse},
	} {
		got := codexFoundReceipt(base, tc.commanded, tc.effective)
		if got.Honored != tc.wantHonored || got.ReportedEffectiveModel != tc.effective {
			t.Errorf("codexFoundReceipt(%q, %q) = %+v, want honored %s with reported %s", tc.commanded, tc.effective, got, tc.wantHonored, tc.effective)
		}
	}
}
