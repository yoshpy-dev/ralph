package org

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// defaultSpawnTimeoutMS is applied when a caller passes TimeoutMS <= 0, so
// the saga always has a bounded context even if the CLI layer's own flag
// default is somehow bypassed.
const defaultSpawnTimeoutMS = 60000

// defaultAgentStartRetryInterval is the wait between AgentStart retry
// attempts when Org.AgentStartRetryInterval is unset (zero value). See
// agentStartWithRetry's doc comment for why this retry exists.
const defaultAgentStartRetryInterval = 500 * time.Millisecond

// maxAgentStartAttempts bounds agentStartWithRetry's total AgentStart call
// count (including the first, non-retry attempt) so a herdr pane that never
// becomes ready cannot retry forever independent of the saga's own ctx
// deadline -- ctx cancellation/deadline is still the primary bound; this cap
// is a hard backstop under it.
const maxAgentStartAttempts = 20

// agentPaneBusyMarker is the literal substring the herdr adapter's error
// text carries when `agent start` is rejected because the target pane's
// shell is not ready yet (herdr code "agent_pane_busy"). Matching on this
// literal (rather than a typed sentinel) mirrors how the rest of this saga
// already treats herdr/agmsg errors -- see the HerdrClient doc comment.
const agentPaneBusyMarker = "agent_pane_busy"

// leadIdentity is the single, grep-able definition of the org's coordinating
// "lead" agmsg identity name (see .claude/rules/agent-messaging.md's "star
// topology" section). Every production call site that names or targets the
// lead identity (ensureLeadJoined's Join, Spawn's HELLO Send TO field) must
// use this constant rather than a bare "lead" literal.
const leadIdentity = "lead"

// defaultLeadDriver is the driver ensureLeadJoined uses to derive the lead
// identity's agmsg type (agmsgTypeForDriver) when SpawnParams.LeadDriver is
// left unset -- matches the `ralph org spawn --lead-driver` flag's own
// default.
const defaultLeadDriver = "claude"

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
// send/stop/disband verbs need. See HerdrClient's doc comment for the
// interface-placement rationale.
type AgmsgClient interface {
	Send(ctx context.Context, team, from, to, message string) error
	// Join registers agentID (agmsg-native agmsgType, e.g. "claude-code" or
	// "codex") on team's roster at projectPath. The spawn saga calls this
	// twice per seat: once for the org's "lead" identity (idempotent,
	// best-effort -- see ensureLeadJoined in Spawn) and once for the seat
	// itself (hard failure gate).
	Join(ctx context.Context, team, agentID, agmsgType, projectPath string) error
	// Leave removes agentID from team's roster (agmsg's `leave.sh TEAM
	// AGENT_ID`). Stop/Disband call this best-effort: a Leave failure is
	// recorded in the stopped event's Details but never fails the verb
	// outright. Leave -- not Despawn -- is the correct roster-removal verb
	// for a seat that joined via Join: despawn.sh only targets processes
	// agmsg itself spawned (it tracks a placement record Join never
	// creates), so it is a silent no-op for every seat this saga ever
	// registers (live-smoke-verified: leave.sh removes the member and
	// auto-deletes an emptied team; despawn.sh exits 0 without touching the
	// roster at all -- see plan "Implementation notes (deviations)", fourth
	// bullet).
	Leave(ctx context.Context, team, agentID string) error
}

// agmsgTypeForDriver maps a ralph driver name to the agmsg-native agent type
// string expected by join.sh's TYPE positional argument. Unknown drivers are
// passed through unchanged -- envelope validation (ValidateSpawnEnvelope)
// already gates driver names against [org].driver_pool before Spawn ever
// reaches this function, so "unknown" here means "a pool member this
// function hasn't been taught about yet", not "unvalidated input".
func agmsgTypeForDriver(driver string) string {
	switch driver {
	case "claude":
		return "claude-code"
	case "codex":
		return "codex"
	default:
		return driver
	}
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
	// AgentStartRetryInterval overrides the wait between agent_pane_busy
	// retries in agentStartWithRetry. Zero (the field's default) means "use
	// defaultAgentStartRetryInterval" -- tests set this to a tiny value so
	// the retry-path tests run fast without an accompanying fake Clock.
	AgentStartRetryInterval time.Duration
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
	OrgID  string
	SeatID string
	Role   string
	Driver string
	Model  string
	Cwd    string
	Prompt string
	// Scope is a free-text description of what this seat is allowed to
	// touch (e.g. a glob or a short prose description). It is not enforced
	// deterministically in this PR (see plan Non-goals -- that lands with
	// the PR④ Watchdog pulse layer); here it is (1) substituted into the
	// seat's role prompt template as {{SCOPE}} and (2) recorded on the
	// `spawned` manifest event's Details as "scope=<value>" so it is at
	// least auditable after the fact.
	Scope     string
	TimeoutMS int
	DryRun    bool
	// AllowUnscoped bypasses the minimum control gate (AC-2b) that would
	// otherwise fail-closed an autonomous-mode spawn with an empty Scope --
	// see the gate check near the top of Spawn for the full rationale. Its
	// use is recorded on the spawned event's Details ("allow_unscoped=true")
	// so an unscoped autonomous seat stays auditable after the fact.
	AllowUnscoped bool
	// LeadDriver is the driver (claude|codex) the org's coordinating "lead"
	// identity itself runs as -- independent of Driver, which names this
	// seat's own driver. It is only consulted by ensureLeadJoined to pick
	// the agmsg type ("claude-code"/"codex") registered for the lead
	// identity on the team roster. Empty defaults to defaultLeadDriver
	// ("claude"), matching the CLI flag's default.
	LeadDriver string
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
// docs/plans/active/2026-08-01-org-runtime-mechanism.md. For a non-dry-run
// call, the ordering is, in this order:
//  1. Idempotent early return: an already-spawned seat returns the existing
//     seat with no validation attempted at all (so an at-cap org can never
//     reject a respawn-of-active-seat retry).
//  2. Stateless envelope validation (ValidateSpawnEnvelope: driver/model
//     pool membership, role restriction) -- a pure function of cfg+req, run
//     before any external side effect (including stale-seat compensation)
//     is attempted, so an envelope-invalid request is always a pure no-op.
//  3. Stale-in-flight compensation: a stale seat (prior spawn_started/
//     spawn_step never resolved) is best-effort compensated and the
//     manifest re-read, so it no longer counts toward max_seats.
//  4. Capacity validation (ValidateSpawnCapacity) against the recomputed
//     activeSeats.
//
// Only then does the saga proceed to (unless DryRun) the workspace/tab/
// agent/agmsg side effects with a spawn_started -> spawn_step* ->
// spawned|spawn_failed manifest trail and a tri-state model receipt. The
// DryRun path is unchanged: validate the full envelope (ValidateSpawn) up
// front, then simulate the trail.
func (o *Org) Spawn(p SpawnParams) SpawnResult {
	// Identifier shape validation runs first, before any manifest read or
	// write and before any path is derived from p.OrgID/p.SeatID (see
	// promptFilePath below). An invalid id is a plain rejection: no
	// `rejected` manifest event is appended for it (unlike envelope
	// validation failures further down, via reject()) because a value that
	// fails this check must never be written into the manifest as if it
	// were a real seat identifier.
	if err := ValidateIdentifier("org_id", p.OrgID); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeRejected, Err: err}
	}
	if err := ValidateIdentifier("seat_id", p.SeatID); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeRejected, Err: err}
	}
	// herdrAgentName joins org_id and seat_id with a single `_` separator
	// (len(org)+1+len(seat)); herdr's live-probed agent-name limit is 32
	// characters, so a combination that individually passes
	// identifierPattern (max 30 chars each) can still overflow herdr's
	// limit once joined. Reject that combination here, before any manifest
	// write, the same way an individually-invalid id is rejected above.
	if n := len(p.OrgID) + 1 + len(p.SeatID); n > maxHerdrAgentNameLen {
		return SpawnResult{Outcome: SpawnOutcomeRejected, Err: fmt.Errorf(
			"org: combined org_id+seat_id length %d exceeds herdr's %d-character agent-name limit (org_id=%q seat_id=%q)",
			n, maxHerdrAgentNameLen, p.OrgID, p.SeatID,
		)}
	}

	// AC-2b minimum control gate (Codex advisory 1): an autonomous-mode seat
	// runs its driver with no interactive permission dialog at all
	// (permissionArgsForDriver's bypassPermissions) -- --scope is the only
	// thing left standing between "autonomous" and "unrestricted". A
	// scope-less autonomous spawn is fail-closed right here, before any
	// manifest read or write, unless the caller explicitly opts out via
	// AllowUnscoped (recorded on the spawned event's Details below so its
	// use stays auditable). This is a plain rejection -- no `rejected`
	// manifest event, mirroring the identifier-shape checks above: the
	// request never reached a state worth recording as an attempted (and
	// envelope-valid) spawn. It runs before the `if p.DryRun` branch below
	// so dry-run spawns are gated identically to real ones.
	resolvedPermMode := ResolvePermissionMode(o.Config, p.Role)
	if resolvedPermMode == "autonomous" && p.Scope == "" && !p.AllowUnscoped {
		return SpawnResult{Outcome: SpawnOutcomeRejected, Err: fmt.Errorf(
			"org: autonomous permission mode requires --scope (or --allow-unscoped to explicitly bypass)",
		)}
	}

	if p.TimeoutMS <= 0 {
		p.TimeoutMS = defaultSpawnTimeoutMS
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: fmt.Errorf("org: read manifest: %w", err)}
	}
	events := rr.Events

	if p.DryRun {
		// Dry-run path is unchanged: validate the envelope up front, then
		// simulate the saga trail without ever consulting existing roster
		// state (dry-run events are excluded from ActiveSeatCount/roster
		// entirely, so there is no idempotency/stale-in-flight concept to
		// apply here). No manifest lock is needed here either -- dry-run
		// events never count toward [org].max_seats, so two concurrent
		// dry-runs cannot race on capacity.
		activeSeats := ActiveSeatCount(events, p.OrgID, RosterOptions{})
		req := SpawnRequest{OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver, Model: p.Model}
		if err := ValidateSpawn(o.Config, req, activeSeats); err != nil {
			return o.reject(p, err)
		}
		// Permission-mode mapping validation (AC-2, codex fail-closed) runs
		// in dry-run too -- validate-then-record contract, same as the
		// envelope check just above. dryRunSpawn never calls AgentStart, so
		// the resolved args themselves are discarded; only the possible
		// error matters here.
		if _, err := permissionArgsForDriver(p.Driver, resolvedPermMode); err != nil {
			return o.reject(p, err)
		}
		return o.dryRunSpawn(p, resolvedPermMode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.TimeoutMS)*time.Millisecond)
	defer cancel()

	// The idempotent/stale-in-flight/envelope/capacity checks and the
	// spawn_started append below run inside withManifestLock: this is the
	// exact "read manifest -> ActiveSeatCount -> ValidateSpawn ->
	// appendEvent" window docs/tech-debt/README.md flagged as an unlocked
	// TOCTOU race ("max_seats is enforced across an unlocked read-then-
	// append window"). Two concurrent Spawn calls used to be able to both
	// observe the same activeSeats snapshot and both pass
	// ValidateSpawnCapacity, exceeding [org].max_seats; the flock in
	// withManifestLock (lockfile.go) now serializes this section across
	// goroutines and processes on the same host.
	//
	// The lock is scoped to this section only (not the workspace/agent/
	// agmsg side effects that follow) -- per the plan's rollout note, a
	// long-running herdr/agmsg round trip (bounded by p.TimeoutMS, up to
	// defaultSpawnTimeoutMS) must not serialize every concurrent spawn
	// behind it.
	var existing *SeatStatus
	var early *SpawnResult
	// permArgs is set inside the locked closure below (permissionArgsForDriver's
	// success value) and consumed after the lock releases, when Spawn builds
	// AgentStart's agentArgs. It stays nil on any early-return path (idempotent
	// respawn, envelope/permission rejection) -- those paths never reach the
	// AgentStart call that would read it.
	var permArgs []string
	req := SpawnRequest{OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver, Model: p.Model}
	lockErr := withManifestLock(filepath.Dir(o.Manifest.Path()), func() error {
		// Fresh read while holding the lock: the outer `events`/`rr` read
		// above (taken before lock acquisition, and shared with the DryRun
		// branch) is not trustworthy for the capacity decision under
		// concurrency -- only a read taken while the lock is held is.
		rr, err := o.Manifest.Read()
		if err != nil {
			return fmt.Errorf("org: read manifest: %w", err)
		}
		events = rr.Events

		roster := Roster(events, RosterOptions{})
		for i := range roster {
			if roster[i].OrgID == p.OrgID && roster[i].SeatID == p.SeatID {
				e := roster[i]
				existing = &e
				break
			}
		}

		if existing != nil && existing.Event == EventSpawned {
			// AC-3: idempotent respawn of an already-spawned seat returns
			// the existing seat, exit 0, no new manifest events, no driver
			// calls -- checked and returned *before* envelope validation,
			// so an already-spawned seat can never be rejected by e.g.
			// max_seats pressure at the at-cap boundary. An idempotent
			// no-op must not be able to fail validation.
			r := SpawnResult{Outcome: SpawnOutcomeIdempotent, Seat: *existing}
			early = &r
			return nil
		}

		// Stateless envelope checks (driver/model pool membership, role
		// restriction) run before any external side effect -- including the
		// best-effort compensation below -- is attempted. Unlike the
		// capacity check, their outcome is a pure function of cfg+req and
		// cannot change as a result of compensating a stale seat, so a
		// request that fails here must be rejected with zero driver calls:
		// reject()'s "no external side effect was ever attempted" claim
		// only holds if this check runs first.
		if err := ValidateSpawnEnvelope(o.Config, req); err != nil {
			r := o.reject(p, err)
			early = &r
			return nil
		}

		// Permission-mode mapping validation (AC-2, codex fail-closed) is
		// also a pure function of cfg+req (driver + the already-resolved
		// mode), so it runs alongside the envelope check above -- before any
		// external side effect, including the stale-seat compensation below.
		// permArgs (captured in the enclosing function scope) survives past
		// this closure for the AgentStart argv construction further down in
		// Spawn.
		args, permErr := permissionArgsForDriver(p.Driver, resolvedPermMode)
		if permErr != nil {
			r := o.reject(p, permErr)
			early = &r
			return nil
		}
		permArgs = args

		if existing != nil && (existing.Event == EventSpawnStarted || existing.Event == EventSpawnStep) {
			// Stale in-flight saga from a prior crashed/interrupted spawn:
			// best-effort compensate and mark it spawn_failed, then re-read
			// the manifest so the now-terminal stale seat no longer counts
			// toward activeSeats for the capacity check below.
			o.compensateStale(ctx, p, *existing)
			rr2, err := o.Manifest.Read()
			if err != nil {
				return fmt.Errorf("org: read manifest: %w", err)
			}
			events = rr2.Events
		}

		activeSeats := ActiveSeatCount(events, p.OrgID, RosterOptions{})
		if err := ValidateSpawnCapacity(o.Config, req, activeSeats); err != nil {
			r := o.reject(p, err)
			early = &r
			return nil
		}

		if err := o.appendEvent(ManifestEvent{
			TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnStarted,
			Role: p.Role, Driver: p.Driver, Model: p.Model, Worktree: p.Cwd,
		}); err != nil {
			r := SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
			early = &r
			return nil
		}
		return nil
	})
	if lockErr != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: fmt.Errorf("org: manifest lock: %w", lockErr)}
	}
	if early != nil {
		return *early
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

	// team is computed here (rather than after AgentStart, as PR① had it)
	// because RenderRolePrompt needs it for the {{TEAM}} substitution below
	// -- agmsgTeam is a pure function of OrgID, so moving it earlier has no
	// observable effect on the agmsg steps further down.
	team := agmsgTeam(p.OrgID)

	// AC-4: a known --role expands the embedded template (reviewer.md /
	// qa.md) into the initial prompt; --prompt, if also given, is appended
	// after it. An unknown role leaves initialPrompt as plain --prompt
	// (possibly empty) -- no error, no fallback template.
	initialPrompt := p.Prompt
	rendered, ok, err := RenderRolePrompt(p.Role, RolePromptVars{
		OrgID: p.OrgID, SeatID: p.SeatID, Team: team, Role: p.Role, Scope: p.Scope,
	})
	if err != nil {
		return o.failStep(p, "agent_start", err, paneID)
	}
	if ok {
		if p.Prompt != "" {
			initialPrompt = rendered + "\n\n" + p.Prompt
		} else {
			initialPrompt = rendered
		}
	}

	// AC-4 deviation (see plan "Implementation notes (deviations)", second
	// bullet): real herdr (v0.7.5) rejects any agent argument containing a
	// newline, and long single-line arguments are also unsafe to assume safe
	// -- so a prompt that trips needsPromptFile is written to
	// <state-dir>/prompts/<org_id>_<seat_id>.md and only a short one-line
	// pointer is passed as the agent arg. The write happens here, strictly
	// before AgentStart, so a write failure never reaches the driver at all.
	//
	// AC-2: permArgs (resolved+validated above, inside the locked closure)
	// come first, then --model, then the prompt (if any) -- a deterministic
	// order the argv tests assert on exactly. A guarded-mode seat has
	// permArgs == nil, so agentArgs starts out identical to pre-permission-
	// mode behavior.
	agentArgs := append([]string{}, permArgs...)
	agentArgs = append(agentArgs, "--model", p.Model)
	agentStartedDetails := "agent_started"
	if initialPrompt != "" {
		if needsPromptFile(initialPrompt) {
			promptPath, perr := o.promptFilePath(p.OrgID, p.SeatID)
			if perr != nil {
				return o.failStep(p, "prompt_file", perr, paneID)
			}
			if err := writePromptFile(promptPath, initialPrompt); err != nil {
				return o.failStep(p, "prompt_file", err, paneID)
			}
			agentArgs = append(agentArgs, promptFilePointer(promptPath))
			agentStartedDetails = fmt.Sprintf("agent_started prompt_file=%s", promptPath)
		} else {
			agentArgs = append(agentArgs, initialPrompt)
		}
	}
	retries, err := o.agentStartWithRetry(ctx, herdrAgentName(p.OrgID, p.SeatID), p.Driver, paneID, p.TimeoutMS, agentArgs)
	if err != nil {
		// Carry the retry count into the failure's Details too (not just the
		// success path below) -- agentStartWithRetry already computed it, so
		// this is a cheap addition that answers "how many attempts were made
		// before this gave up" for a failed spawn, not only a successful one.
		return o.failStepWithNote(p, "agent_start", err, paneID, fmt.Sprintf("agent_start_retries=%d", retries))
	}
	if retries > 0 {
		agentStartedDetails = fmt.Sprintf("%s agent_start_retries=%d", agentStartedDetails, retries)
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnStep,
		PaneID: paneID, Details: agentStartedDetails,
	}); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
	}

	leadJoinNote, err := o.ensureLeadJoined(ctx, p, team, paneID)
	if err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
	}

	if err := o.Agmsg.Join(ctx, team, p.SeatID, agmsgTypeForDriver(p.Driver), p.Cwd); err != nil {
		return o.failStep(p, "agmsg_join", err, paneID)
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnStep,
		PaneID: paneID, AgmsgTeam: team, Details: "agmsg_joined",
	}); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
	}

	// AC-11: the HELLO body must itself be protocol.ValidateText-conformant
	// (see TestSpawn_HelloMessage_IsProtocolConformant) -- HELLO does not
	// require TASK_ID, so a TYPE header plus these fields alone is valid.
	msg := fmt.Sprintf("TYPE: HELLO\nSEAT: %s\nROLE: %s\nORG_ID: %s", p.SeatID, p.Role, p.OrgID)
	if err := o.Agmsg.Send(ctx, team, p.SeatID, leadIdentity, msg); err != nil {
		// tech-debt (docs/tech-debt/README.md, "spawn の agmsg_announce(HELLO
		// send)失敗パスの補償..."): the seat's own Join already succeeded by
		// this point, so a failed HELLO announce must not leave a stale
		// roster entry behind -- best-effort Leave it back out, and record
		// the outcome in spawn_failed's Details alongside the lead-join note
		// so both compensation steps stay auditable from the manifest alone.
		leaveNote := compensateLeave(o.Agmsg, team, p.SeatID)
		return o.failStepWithNote(p, "agmsg_announce", err, paneID, fmt.Sprintf("lead_join=%s leave=%s", leadJoinNote, leaveNote))
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnStep,
		PaneID: paneID, AgmsgTeam: team, Details: "agmsg_announced",
	}); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
	}

	// Scope/AllowUnscoped/permission_mode have no dedicated ManifestEvent
	// field: they are recorded as free-text fragments in Details (see
	// spawnedEventDetails) so they stay auditable without a manifest schema
	// change. permission_mode is always present (AC-2b); scope/
	// allow_unscoped are present only when the corresponding param was set.
	spawnedDetails := spawnedEventDetails(p, resolvedPermMode)
	// herdr_agent_name is persisted here (tech-debt, docs/tech-debt/
	// README.md, "The herdr agent name is derived at every call site...
	// instead of being persisted"): PaneID already gets this treatment
	// ("herdr external id, persisted as soon as known" -- manifest.go). A
	// future change to herdrAgentName's naming convention now orphans no
	// existing seat: verbs.go's Send prefers this recorded value and only
	// falls back to re-deriving it for pre-existing (legacy) events.
	agentName := herdrAgentName(p.OrgID, p.SeatID)
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawned,
		Role: p.Role, Driver: p.Driver, Model: p.Model, Worktree: p.Cwd,
		PaneID: paneID, AgmsgTeam: team, HerdrAgentName: agentName, Details: spawnedDetails,
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
		Worktree: p.Cwd, PaneID: paneID, AgmsgTeam: team, HerdrAgentName: agentName,
		Event: EventSpawned, Active: true,
	}}
}

// spawnedEventDetails builds the `spawned` manifest event's free-text
// Details field for both the real Spawn path and dryRunSpawn's simulated
// trail: "scope=<v>" when Scope is set, "allow_unscoped=true" when
// AllowUnscoped was passed, and always "permission_mode=<mode>" (AC-2b --
// the resolved mode is recorded for every seat, not only autonomous ones,
// so `ralph org report` can show every seat's effective mode uniformly).
func spawnedEventDetails(p SpawnParams, mode string) string {
	parts := make([]string, 0, 3)
	if p.Scope != "" {
		parts = append(parts, fmt.Sprintf("scope=%s", p.Scope))
	}
	if p.AllowUnscoped {
		parts = append(parts, "allow_unscoped=true")
	}
	parts = append(parts, fmt.Sprintf("permission_mode=%s", mode))
	return strings.Join(parts, " ")
}

// ensureLeadJoined best-effort join.sh's <team> lead <type> <cwd>, where
// <type> is agmsgTypeForDriver(p.LeadDriver) (defaultLeadDriver ("claude")
// when p.LeadDriver is unset) -- the lead identity's own driver is
// independent of this seat's Driver, so a Codex-coordinated org must not
// register "lead" under a hardcoded claude-code type (tech-debt,
// docs/tech-debt/README.md, "lead identity is a bare 'lead' string literal
// ... and its agmsg type is hardcoded agmsgTypeForDriver('claude')"). A
// clean agmsg team has no "lead" identity registered yet, and agmsg's
// roster-based send validation rejects HELLO messages whose from/to
// identity was never join.sh'd (agmsg #355) -- so the saga must attempt to
// register leadIdentity before the seat's own Join+Send that follows it in
// Spawn. join.sh is treated as idempotent (re-joining an existing member is
// a documented no-op/soft-fail in agmsg), so a lead-join error here does
// *not* fail the saga on its own: the definitive, single-authoritative-
// failure-point gate is the seat Join immediately after this call and,
// ultimately, the HELLO Send -- if the roster is genuinely missing
// leadIdentity, Send fails and the lead-join error recorded here is carried
// into that failure's Details for diagnosis (see failStepWithNote's doc
// comment).
//
// The returned string is the "agmsg_lead_joined <note>" note recorded on the
// step's manifest event -- "ok" on success, "error=<err>" otherwise -- so
// Spawn can also fold it into a later failure's Details. The returned error
// is only non-nil when appending that manifest event itself fails (a
// manifest-write failure, not a Join failure); a Join failure is captured in
// the returned note instead of being treated as fatal, per the doc comment
// above.
func (o *Org) ensureLeadJoined(ctx context.Context, p SpawnParams, team, paneID string) (string, error) {
	leadDriver := p.LeadDriver
	if leadDriver == "" {
		leadDriver = defaultLeadDriver
	}
	leadJoinErr := o.Agmsg.Join(ctx, team, leadIdentity, agmsgTypeForDriver(leadDriver), p.Cwd)
	leadJoinNote := "ok"
	if leadJoinErr != nil {
		leadJoinNote = fmt.Sprintf("error=%v", leadJoinErr)
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnStep,
		PaneID: paneID, AgmsgTeam: team, Details: fmt.Sprintf("agmsg_lead_joined %s", leadJoinNote),
	}); err != nil {
		return leadJoinNote, err
	}
	return leadJoinNote, nil
}

// agmsgTeam is the team name convention used to announce a newly spawned
// seat to the org's lead (see plan Open questions -- provisional pending
// PR②'s seat prompt design).
func agmsgTeam(orgID string) string {
	return fmt.Sprintf("ralph-%s", orgID)
}

// agentStartRetryInterval returns o.AgentStartRetryInterval, falling back to
// defaultAgentStartRetryInterval when unset (the Org zero value).
func (o *Org) agentStartRetryInterval() time.Duration {
	if o.AgentStartRetryInterval > 0 {
		return o.AgentStartRetryInterval
	}
	return defaultAgentStartRetryInterval
}

// agentStartWithRetry calls Herdr.AgentStart, retrying with a bounded
// interval when the herdr adapter reports agent_pane_busy: a freshly created
// tab's pane is still initializing its shell for ~1-3s and rejects
// `agent start` with agent_pane_busy until it is ready (real-herdr smoke
// probe: immediate call -> busy, ~3s later -> accepted -- see plan
// docs/plans/active/2026-08-02-org-runtime-seats.md, "Implementation notes
// (deviations)", third bullet). Any other error is returned immediately,
// exactly as a bare AgentStart call would today -- this function changes
// AgentStart's retry behavior, not its error semantics.
//
// The retry loop is bounded two ways: ctx (the saga's own spawn deadline,
// honored via ctx.Done() during the inter-attempt wait) and
// maxAgentStartAttempts (a hard backstop independent of ctx, so a caller
// that passes a very long or no-deadline ctx still cannot retry forever).
// The returned int is the number of retry attempts made before the call
// that ultimately returned (0 when the first attempt succeeds or fails with
// a non-agent_pane_busy error) -- callers use it to annotate the
// agent_started/agent_start-failure step's Details for audit purposes. On
// the maxAgentStartAttempts-exhaustion path this is maxAgentStartAttempts-1
// (the first attempt is attempt 0, not itself a retry, so exhausting all
// maxAgentStartAttempts calls means exactly maxAgentStartAttempts-1 retries
// followed it) -- returning maxAgentStartAttempts here would overcount by
// one retry that was never actually made.
func (o *Org) agentStartWithRetry(ctx context.Context, name, kind, paneID string, timeoutMS int, agentArgs []string) (int, error) {
	interval := o.agentStartRetryInterval()
	var lastErr error
	var lastAttempt int
	for attempt := range maxAgentStartAttempts {
		lastAttempt = attempt
		_, err := o.Herdr.AgentStart(ctx, name, kind, paneID, timeoutMS, agentArgs)
		if err == nil {
			return attempt, nil
		}
		if !strings.Contains(err.Error(), agentPaneBusyMarker) {
			return attempt, err
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return attempt, lastErr
		case <-time.After(interval):
		}
	}
	return lastAttempt, lastErr
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
//
// The join uses `_`, not `-`: identifierPattern (identifier.go) forbids `_`
// in either orgID or seatID, so `_` is guaranteed to be a byte that appears
// nowhere else in either half. That makes the join unambiguous -- unlike a
// `-` join, where org_id="a-b"/seat_id="c" and org_id="a"/seat_id="b-c"
// would both produce "a-b-c" and collide in herdr's global agent namespace
// (the exact bug this fixes; see the cross-review cycle-2 fix note in
// docs/plans/active/2026-08-02-org-runtime-seats.md, "Implementation notes
// (deviations)"). `_` is also herdr-legal on its own, per the same live
// probe referenced in identifier.go.
func herdrAgentName(orgID, seatID string) string {
	return fmt.Sprintf("%s_%s", orgID, seatID)
}

// maxHerdrAgentNameLen is herdr's live-probed agent-name length limit
// (`^[a-z][a-z0-9_-]{0,31}$`, v0.7.5 -- see identifierPattern's doc comment
// in identifier.go). Spawn checks the joined `<org>_<seat>` length against
// this before any manifest write, since identifierPattern alone (max 30
// chars per half) does not prevent the sum from exceeding it.
const maxHerdrAgentNameLen = 32

// maxInlinePromptRunes is the longest initial prompt herdr's real
// `agent start` argv encoding is trusted to accept inline. Real herdr
// (v0.7.5) outright rejects any agent argument containing a newline
// (invalid_agent_argument: "agent arguments cannot be encoded safely for
// the target shell") -- see plan
// docs/plans/active/2026-08-02-org-runtime-seats.md, "Implementation notes
// (deviations)". A long single-line prompt is treated the same way out of
// caution, even though only the newline case has been observed to fail
// against the real CLI.
const maxInlinePromptRunes = 200

// needsPromptFile reports whether prompt is too unsafe to pass directly as
// a herdr agent argument and must instead be written to a prompt file with
// only a one-line pointer passed inline (see promptFilePath/writePromptFile/
// promptFilePointer below).
func needsPromptFile(prompt string) bool {
	return strings.Contains(prompt, "\n") || utf8.RuneCountInString(prompt) > maxInlinePromptRunes
}

// promptFilePath returns the absolute path a spawn's initial prompt is
// written to when needsPromptFile is true:
// <state-dir>/prompts/<org_id>_<seat_id>.md. state-dir is derived from the
// manifest store's own directory (filepath.Dir(o.Manifest.Path())) rather
// than a separate config field, since the manifest store is already the
// single source of truth for where this Org's on-disk state lives. The path
// is namespaced by both org_id and seat_id so a respawn of the same seat
// overwrites its own prompt file (intentional -- see writePromptFile) while
// two different seats never collide.
//
// The join uses `_`, the same reserved separator as herdrAgentName (see its
// doc comment) and for the same reason: identifierPattern forbids `_` in
// either half, so the join is unambiguous. Before this fix both this
// function and herdrAgentName joined with `-`, which meant org_id="a-b"/
// seat_id="c" and org_id="a"/seat_id="b-c" both wrote to the same prompt
// file path -- a later spawn's role prompt silently overwriting an earlier
// seat's.
func (o *Org) promptFilePath(orgID, seatID string) (string, error) {
	stateDir, err := absPath(filepath.Dir(o.Manifest.Path()))
	if err != nil {
		return "", fmt.Errorf("resolve state dir for prompt file: %w", err)
	}
	return filepath.Join(stateDir, "prompts", fmt.Sprintf("%s_%s.md", orgID, seatID)), nil
}

// absPath is filepath.Abs by default; tests reassign it to inject a
// resolution failure (mirroring driver.go's lookPath seam) so
// promptFilePath's error path -- otherwise only reachable via a broken
// process cwd -- is deterministically testable (AC-10b: dryRunSpawn must
// propagate this failure instead of silently swallowing it).
var absPath = filepath.Abs

// writePromptFile writes content to path with 0644 permissions, creating
// any missing parent directories first. It always overwrites an existing
// file at path (os.WriteFile truncates) -- a respawn of the same org_id/
// seat_id must replace the previous prompt file's content, not append to it
// or fail because it already exists.
func writePromptFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create prompt file directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write prompt file: %w", err)
	}
	return nil
}

// promptFilePointer is the single-line agent argument passed in place of
// the full prompt once it has been written to path: a short instruction
// telling the agent to read and follow that file instead.
func promptFilePointer(path string) string {
	return "役割指示を読み込んで従ってください: " + path
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
// they carry no real side effects and no [org].max_seats pressure. mode is
// the permission mode already resolved (and validated via
// permissionArgsForDriver) by the caller, recorded on the trail's final
// EventSpawned step the same way the real Spawn path records it.
func (o *Org) dryRunSpawn(p SpawnParams, mode string) SpawnResult {
	team := agmsgTeam(p.OrgID)
	base := ManifestEvent{OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver, Model: p.Model, Worktree: p.Cwd, DryRun: true}

	steps := make([]ManifestEvent, 0, 7)
	step := base
	step.Event = EventSpawnStarted
	steps = append(steps, step)

	step = base
	step.Event = EventSpawnStep
	step.Details = "tab_created"
	steps = append(steps, step)

	// AC-4 deviation (see the matching comment in Spawn): dry-run must
	// simulate the same prompt-file-vs-inline decision, without ever writing
	// the file or calling Herdr, so the trail a dry-run produces matches what
	// a real spawn's agent_started step Details would say.
	//
	// AC-10b (tech-debt: docs/tech-debt/README.md, "dryRunSpawn silently
	// swallows RenderRolePrompt's and promptFilePath's errors"): both errors
	// used to be discarded via `err == nil && ok` / `perr == nil` guards, so
	// a dry run could report SpawnOutcomeSpawned for a spawn the real path
	// would fail at "agent_start"/"prompt_file". Both are now propagated as
	// a failed result the same way failStep would in the real path -- paneID
	// is "" since a dry run never creates a real pane (compensatePane then
	// records "no pane to compensate", matching dry-run's zero-side-effect
	// contract), and DryRun: p.DryRun on the resulting spawn_failed event
	// (see failStepWithNote) keeps it excluded from real-seat accounting.
	agentStartedDetails := "agent_started"
	initialPrompt := p.Prompt
	rendered, ok, err := RenderRolePrompt(p.Role, RolePromptVars{
		OrgID: p.OrgID, SeatID: p.SeatID, Team: team, Role: p.Role, Scope: p.Scope,
	})
	if err != nil {
		return o.failStep(p, "agent_start", err, "")
	}
	if ok {
		if p.Prompt != "" {
			initialPrompt = rendered + "\n\n" + p.Prompt
		} else {
			initialPrompt = rendered
		}
	}
	if initialPrompt != "" && needsPromptFile(initialPrompt) {
		promptPath, perr := o.promptFilePath(p.OrgID, p.SeatID)
		if perr != nil {
			return o.failStep(p, "prompt_file", perr, "")
		}
		agentStartedDetails = fmt.Sprintf("agent_started prompt_file=%s", promptPath)
	}

	step = base
	step.Event = EventSpawnStep
	step.Details = agentStartedDetails
	steps = append(steps, step)

	step = base
	step.Event = EventSpawnStep
	step.AgmsgTeam = team
	step.Details = "agmsg_lead_joined ok"
	steps = append(steps, step)

	step = base
	step.Event = EventSpawnStep
	step.AgmsgTeam = team
	step.Details = "agmsg_joined"
	steps = append(steps, step)

	step = base
	step.Event = EventSpawnStep
	step.AgmsgTeam = team
	step.Details = "agmsg_announced"
	steps = append(steps, step)

	step = base
	step.Event = EventSpawned
	step.AgmsgTeam = team
	step.Details = spawnedEventDetails(p, mode)
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
	return o.failStepWithNote(p, step, cause, paneID, "")
}

// failStepWithNote is failStep plus an extra free-text note appended to
// Details. Two note-carrying callers: the agmsg_announce failure path
// (carrying the ensureLeadJoined outcome and the agmsg_announce Leave
// compensation forward — a missing "lead" roster entry is the most likely
// root cause of a Send rejection) and the agent_start failure path
// (carrying `agent_start_retries=N` so exhausted pane-busy retries stay
// auditable). failStep (no note) additionally covers dryRunSpawn's
// RenderRolePrompt/promptFilePath error paths (AC-10b) -- paneID is always
// "" there since a dry run never creates a real pane, so compensatePane
// records "no pane to compensate" for those calls, matching dry-run's
// zero-side-effect contract.
//
// DryRun: p.DryRun on the appended event mirrors p.DryRun exactly: it is
// always false for every real-Spawn caller (failStep/failStepWithNote are
// only reachable from the non-dry-run branch of Spawn itself, after the
// `if p.DryRun { ... }` early return) and true for dryRunSpawn's callers --
// so a dry-run failure is correctly excluded from ActiveSeatCount/roster
// like every other dry-run event, instead of fabricating a real seat's
// spawn_failed state.
func (o *Org) failStepWithNote(p SpawnParams, step string, cause error, paneID, note string) SpawnResult {
	compensation := compensatePane(o.Herdr, paneID)
	details := fmt.Sprintf("step=%s error=%v compensation=%s", step, cause, compensation)
	if note != "" {
		details += " " + note
	}
	_ = o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnFailed,
		Role: p.Role, Driver: p.Driver, Model: p.Model, Worktree: p.Cwd,
		PaneID: paneID, DryRun: p.DryRun, Details: details,
	})
	return SpawnResult{Outcome: SpawnOutcomeFailed, Err: fmt.Errorf("org: spawn step %s failed: %w", step, cause)}
}

// compensateLeave sends a best-effort agmsg Leave for agentID from team,
// used by the agmsg_announce failure path (AC-6/tech-debt: "spawn の
// agmsg_announce(HELLO send)失敗パスの補償が...Leave しない"): by the time
// HELLO Send fails, the seat's own Join has already succeeded, so without
// this call a failed spawn leaves a stale roster entry behind. Errors are
// recorded in the returned string, not propagated -- like compensatePane,
// this is inherently best-effort and must never itself fail the saga.
func compensateLeave(a AgmsgClient, team, agentID string) string {
	if team == "" || agentID == "" {
		return "skipped: no team/agent to leave"
	}
	if err := a.Leave(context.Background(), team, agentID); err != nil {
		return fmt.Sprintf("failed: %v", err)
	}
	return "ok"
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
