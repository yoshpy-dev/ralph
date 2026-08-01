package org

import (
	"context"
	"fmt"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// defaultSpawnTimeoutMS is applied when a caller passes TimeoutMS <= 0, so
// the saga always has a bounded context even if the CLI layer's own flag
// default is somehow bypassed.
const defaultSpawnTimeoutMS = 60000

// EventOrgWorkspaceCreated is an org-level event (SeatID empty) recorded the
// first time a herdr workspace is created for an org_id. Later spawns within
// the same org_id reuse the recorded PaneID (the workspace id) instead of
// calling Herdr.WorkspaceCreate again -- one workspace per org, many tabs
// (one per seat).
const EventOrgWorkspaceCreated = "org_workspace_created"

// HerdrClient is the subset of driver.Herdr's methods the spawn saga and the
// send/wait/read/stop verbs need. Defined here (consumption side, per
// .claude/rules/architecture.md) rather than in internal/org/driver, so
// internal/org stays free of any exec.Command dependency -- driver.Herdr
// satisfies this interface structurally. Wiring lives in internal/cli/org.go:
// driver.Herdr{R: driver.ExecRunner{}} is assigned directly to Org.Herdr.
type HerdrClient interface {
	WorkspaceCreate(ctx context.Context, cwd, label string) (string, error)
	TabCreate(ctx context.Context, workspaceID, cwd, label string) (string, error)
	AgentStart(ctx context.Context, name, kind, paneID string, timeoutMS int, agentArgs []string) (string, error)
	AgentWait(ctx context.Context, target string, until []string, timeoutMS int) (string, error)
	PaneRead(ctx context.Context, paneID string, lines int) (string, error)
	PaneSendText(ctx context.Context, paneID, text string) error
	PaneSendKeys(ctx context.Context, paneID string, keys ...string) error
}

// AgmsgClient is the subset of driver.Agmsg's methods the spawn saga and the
// send verb need. See HerdrClient's doc comment for the interface-placement
// rationale.
type AgmsgClient interface {
	Send(ctx context.Context, team, from, to, message string) error
}

// Clock abstracts time.Now so spawn/verb tests can inject deterministic
// timestamps. A nil Clock on Org falls back to time.Now.
type Clock func() time.Time

// Org bundles the manifest/receipt stores and driver clients needed by every
// `ralph org` verb: spawn saga (this file) and send/wait/read/stop/status/
// disband (verbs.go). internal/cli/org.go constructs one Org per command
// invocation and calls its methods -- it never touches ManifestStore,
// ReceiptStore, or the driver clients directly (thin flag parsing + wiring
// only).
type Org struct {
	Config   config.OrgConfig
	Manifest *ManifestStore
	Receipts *ReceiptStore
	Herdr    HerdrClient
	Agmsg    AgmsgClient
	Now      Clock
}

func (o *Org) now() string {
	nowFn := time.Now
	if o.Now != nil {
		nowFn = o.Now
	}
	return nowFn().UTC().Format(time.RFC3339)
}

func (o *Org) appendEvent(ev ManifestEvent) error {
	return o.Manifest.Append(ev)
}

// SpawnParams describes one `ralph org spawn` invocation.
type SpawnParams struct {
	OrgID     string
	SeatID    string
	Role      string
	Driver    string
	Model     string
	Cwd       string
	Prompt    string
	TimeoutMS int
	DryRun    bool
}

// SpawnOutcome classifies how a Spawn call concluded, so the CLI layer can
// choose exit code and message without re-deriving saga state.
type SpawnOutcome string

const (
	SpawnOutcomeRejected   SpawnOutcome = "rejected"
	SpawnOutcomeIdempotent SpawnOutcome = "idempotent"
	SpawnOutcomeSpawned    SpawnOutcome = "spawned"
	SpawnOutcomeFailed     SpawnOutcome = "failed"
)

// SpawnResult is Spawn's return value. Err is non-nil for Rejected and
// Failed outcomes (the CLI layer returns it so main() exits non-zero); it is
// nil for Idempotent and Spawned (exit 0).
type SpawnResult struct {
	Outcome SpawnOutcome
	Seat    SeatStatus
	Err     error
}

// Spawn runs the full spawn saga described in
// docs/plans/active/2026-08-01-org-runtime-mechanism.md: envelope
// validation, idempotency/stale-in-flight handling, then (unless DryRun) the
// workspace/tab/agent/agmsg side effects with a spawn_started -> spawn_step*
// -> spawned|spawn_failed manifest trail and a tri-state model receipt.
func (o *Org) Spawn(p SpawnParams) SpawnResult {
	if p.TimeoutMS <= 0 {
		p.TimeoutMS = defaultSpawnTimeoutMS
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: fmt.Errorf("org: read manifest: %w", err)}
	}
	events := rr.Events

	activeSeats := ActiveSeatCount(events, p.OrgID, RosterOptions{})
	req := SpawnRequest{OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver, Model: p.Model}
	if err := ValidateSpawn(o.Config, req, activeSeats); err != nil {
		return o.reject(p, err)
	}

	if p.DryRun {
		return o.dryRunSpawn(p)
	}

	roster := Roster(events, RosterOptions{})
	var existing *SeatStatus
	for i := range roster {
		if roster[i].OrgID == p.OrgID && roster[i].SeatID == p.SeatID {
			existing = &roster[i]
			break
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.TimeoutMS)*time.Millisecond)
	defer cancel()

	if existing != nil {
		switch existing.Event {
		case EventSpawned:
			// AC-3: idempotent respawn of an already-spawned seat returns
			// the existing seat, exit 0, no new driver calls.
			return SpawnResult{Outcome: SpawnOutcomeIdempotent, Seat: *existing}
		case EventSpawnStarted, EventSpawnStep:
			// Stale in-flight saga from a prior crashed/interrupted spawn:
			// best-effort compensate, mark it spawn_failed, then fall
			// through to a fresh spawn below.
			o.compensateStale(ctx, p, *existing)
		}
	}

	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnStarted,
		Role: p.Role, Driver: p.Driver, Model: p.Model, Worktree: p.Cwd,
	}); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
	}

	workspaceID, err := o.resolveWorkspace(ctx, p, events)
	if err != nil {
		return o.failStep(p, "workspace_create", err, "")
	}

	paneID, err := o.Herdr.TabCreate(ctx, workspaceID, p.Cwd, p.SeatID)
	if err != nil {
		return o.failStep(p, "tab_create", err, "")
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnStep,
		PaneID: paneID, Details: "tab_created",
	}); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
	}

	agentArgs := []string{"--model", p.Model}
	if p.Prompt != "" {
		agentArgs = append(agentArgs, p.Prompt)
	}
	if _, err := o.Herdr.AgentStart(ctx, herdrAgentName(p.OrgID, p.SeatID), p.Driver, paneID, p.TimeoutMS, agentArgs); err != nil {
		return o.failStep(p, "agent_start", err, paneID)
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnStep,
		PaneID: paneID, Details: "agent_started",
	}); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
	}

	team := agmsgTeam(p.OrgID)
	msg := fmt.Sprintf("TYPE: HELLO\nSEAT: %s\nROLE: %s", p.SeatID, p.Role)
	if err := o.Agmsg.Send(ctx, team, p.SeatID, "lead", msg); err != nil {
		return o.failStep(p, "agmsg_announce", err, paneID)
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnStep,
		PaneID: paneID, AgmsgTeam: team, Details: "agmsg_announced",
	}); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
	}

	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawned,
		Role: p.Role, Driver: p.Driver, Model: p.Model, Worktree: p.Cwd,
		PaneID: paneID, AgmsgTeam: team,
	}); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
	}
	if err := o.Receipts.Append(Receipt{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver,
		CommandedModel: p.Model, Honored: HonoredUnknown,
		Reason: "interactive session; effective model not yet observable",
	}); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
	}

	return SpawnResult{Outcome: SpawnOutcomeSpawned, Seat: SeatStatus{
		OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver, Model: p.Model,
		Worktree: p.Cwd, PaneID: paneID, AgmsgTeam: team, Event: EventSpawned, Active: true,
	}}
}

// agmsgTeam is the team name convention used to announce a newly spawned
// seat to the org's lead (see plan Open questions -- provisional pending
// PR②'s seat prompt design).
func agmsgTeam(orgID string) string {
	return fmt.Sprintf("ralph-%s", orgID)
}

// herdrAgentName is the single, grep-able definition of the herdr agent-name
// convention: every call site that names or targets a herdr agent (spawn's
// AgentStart, and send/wait's AgentWait) must derive the name through this
// function rather than passing the bare seat id. herdr's agent namespace is
// global across all orgs, so two org_ids that both spawn a seat named
// (for example) "reviewer" would otherwise register (and later target)
// exactly the same herdr agent -- silently colliding across org boundaries.
// Namespacing by org_id here mirrors the agmsgTeam convention above and
// keeps the external-resource boundary isolated the same way manifest
// accounting already is.
func herdrAgentName(orgID, seatID string) string {
	return fmt.Sprintf("%s-%s", orgID, seatID)
}

// reject records an envelope-validation rejection: a `rejected` manifest
// event plus an honored=false receipt, per AC-1/AC-2. No spawn_started is
// written since no external side effect was ever attempted.
func (o *Org) reject(p SpawnParams, cause error) SpawnResult {
	_ = o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventRejected,
		Role: p.Role, Driver: p.Driver, Model: p.Model, Worktree: p.Cwd,
		DryRun: p.DryRun, Details: cause.Error(),
	})
	_ = o.Receipts.Append(Receipt{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver,
		CommandedModel: p.Model, Honored: HonoredFalse, Reason: cause.Error(),
	})
	return SpawnResult{Outcome: SpawnOutcomeRejected, Err: cause}
}

// dryRunSpawn simulates the full saga's manifest trail with DryRun: true on
// every event, without ever calling Herdr or Agmsg (AC-8). Dry-run events are
// excluded from the default roster/status view and from ActiveSeatCount, so
// they carry no real side effects and no [org].max_seats pressure.
func (o *Org) dryRunSpawn(p SpawnParams) SpawnResult {
	team := agmsgTeam(p.OrgID)
	base := ManifestEvent{OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver, Model: p.Model, Worktree: p.Cwd, DryRun: true}

	steps := make([]ManifestEvent, 0, 5)
	step := base
	step.Event = EventSpawnStarted
	steps = append(steps, step)

	step = base
	step.Event = EventSpawnStep
	step.Details = "tab_created"
	steps = append(steps, step)

	step = base
	step.Event = EventSpawnStep
	step.Details = "agent_started"
	steps = append(steps, step)

	step = base
	step.Event = EventSpawnStep
	step.AgmsgTeam = team
	step.Details = "agmsg_announced"
	steps = append(steps, step)

	step = base
	step.Event = EventSpawned
	step.AgmsgTeam = team
	steps = append(steps, step)

	for i := range steps {
		steps[i].TS = o.now()
		if err := o.appendEvent(steps[i]); err != nil {
			return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
		}
	}

	if err := o.Receipts.Append(Receipt{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver,
		CommandedModel: p.Model, Honored: HonoredUnknown, Reason: "dry-run",
	}); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
	}

	return SpawnResult{Outcome: SpawnOutcomeSpawned, Seat: SeatStatus{
		OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver, Model: p.Model,
		Worktree: p.Cwd, AgmsgTeam: team, Event: EventSpawned, Active: false, DryRun: true,
	}}
}

// resolveWorkspace reuses the org's existing herdr workspace (recorded via
// an EventOrgWorkspaceCreated org-level event, SeatID empty, PaneID =
// workspace id) if one exists for orgID within events, otherwise creates one
// and records it. events is the manifest snapshot read at the top of Spawn,
// before this seat's own spawn_started was appended -- irrelevant to this
// lookup since org-level workspace events are seat-independent.
func (o *Org) resolveWorkspace(ctx context.Context, p SpawnParams, events []ManifestEvent) (string, error) {
	for _, ev := range events {
		if ev.OrgID == p.OrgID && ev.SeatID == "" && ev.Event == EventOrgWorkspaceCreated {
			return ev.PaneID, nil
		}
	}
	workspaceID, err := o.Herdr.WorkspaceCreate(ctx, p.Cwd, p.OrgID)
	if err != nil {
		return "", err
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: "", Event: EventOrgWorkspaceCreated,
		PaneID: workspaceID, Worktree: p.Cwd,
	}); err != nil {
		return "", err
	}
	return workspaceID, nil
}

// failStep records a spawn_failed event for a saga step that returned an
// error: best-effort compensation (send C-c to the pane, if one exists yet)
// followed by a manifest event whose Details captures the failing step, the
// underlying error, and the compensation outcome. paneID (if non-empty) is
// preserved on the event so an orphaned external resource stays traceable
// from the manifest alone (AC-10).
func (o *Org) failStep(p SpawnParams, step string, cause error, paneID string) SpawnResult {
	compensation := compensatePane(o.Herdr, paneID)
	details := fmt.Sprintf("step=%s error=%v compensation=%s", step, cause, compensation)
	_ = o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnFailed,
		Role: p.Role, Driver: p.Driver, Model: p.Model, Worktree: p.Cwd,
		PaneID: paneID, Details: details,
	})
	return SpawnResult{Outcome: SpawnOutcomeFailed, Err: fmt.Errorf("org: spawn step %s failed: %w", step, cause)}
}

// compensateStale best-effort-compensates a stale in-flight saga (a prior
// spawn_started/spawn_step for the same seat, never resolved) and records
// its spawn_failed terminus before Spawn proceeds to a fresh attempt.
// existing's external ids (PaneID, AgmsgTeam) are carried forward onto the
// spawn_failed event so they remain traceable.
func (o *Org) compensateStale(ctx context.Context, p SpawnParams, existing SeatStatus) {
	compensation := compensatePaneCtx(ctx, o.Herdr, existing.PaneID)
	details := fmt.Sprintf("superseded by respawn (previous_event=%s compensation=%s)", existing.Event, compensation)
	_ = o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnFailed,
		Role: existing.Role, Driver: existing.Driver, Model: existing.Model, Worktree: existing.Worktree,
		PaneID: existing.PaneID, AgmsgTeam: existing.AgmsgTeam, Details: details,
	})
}

// compensatePane is the failStep-path compensation helper: it always uses a
// fresh background context so a best-effort cleanup call is never itself cut
// short by the saga's own (possibly already-expired) timeout context.
func compensatePane(h HerdrClient, paneID string) string {
	return compensatePaneCtx(context.Background(), h, paneID)
}

// compensatePaneCtx sends a best-effort C-c to paneID (if non-empty) and
// describes the outcome for a manifest Details string. Errors are recorded,
// not propagated -- compensation is inherently best-effort.
func compensatePaneCtx(ctx context.Context, h HerdrClient, paneID string) string {
	if paneID == "" {
		return "no pane to compensate"
	}
	if err := h.PaneSendKeys(ctx, paneID, "C-c"); err != nil {
		return fmt.Sprintf("C-c failed: %v", err)
	}
	return "C-c sent"
}
