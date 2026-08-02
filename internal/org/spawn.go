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
	// Despawn removes name from team's roster, sent as from. Stop/Disband
	// call this best-effort: a Despawn failure is recorded in the stopped
	// event's Details but never fails the verb outright.
	Despawn(ctx context.Context, team, from, name string) error
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
		// apply here).
		activeSeats := ActiveSeatCount(events, p.OrgID, RosterOptions{})
		req := SpawnRequest{OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver, Model: p.Model}
		if err := ValidateSpawn(o.Config, req, activeSeats); err != nil {
			return o.reject(p, err)
		}
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

	if existing != nil && existing.Event == EventSpawned {
		// AC-3: idempotent respawn of an already-spawned seat returns the
		// existing seat, exit 0, no new manifest events, no driver calls --
		// checked and returned *before* envelope validation, so an
		// already-spawned seat can never be rejected by e.g. max_seats
		// pressure at the at-cap boundary. An idempotent no-op must not be
		// able to fail validation.
		return SpawnResult{Outcome: SpawnOutcomeIdempotent, Seat: *existing}
	}

	// Stateless envelope checks (driver/model pool membership, role
	// restriction) run before any external side effect -- including the
	// best-effort compensation below -- is attempted. Unlike the capacity
	// check, their outcome is a pure function of cfg+req and cannot change
	// as a result of compensating a stale seat, so a request that fails
	// here must be rejected with zero driver calls: reject()'s "no external
	// side effect was ever attempted" claim only holds if this check runs
	// first.
	req := SpawnRequest{OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver, Model: p.Model}
	if err := ValidateSpawnEnvelope(o.Config, req); err != nil {
		return o.reject(p, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.TimeoutMS)*time.Millisecond)
	defer cancel()

	if existing != nil && (existing.Event == EventSpawnStarted || existing.Event == EventSpawnStep) {
		// Stale in-flight saga from a prior crashed/interrupted spawn:
		// best-effort compensate and mark it spawn_failed, then re-read the
		// manifest so the now-terminal stale seat no longer counts toward
		// activeSeats for the capacity check below.
		o.compensateStale(ctx, p, *existing)
		rr, err = o.Manifest.Read()
		if err != nil {
			return SpawnResult{Outcome: SpawnOutcomeFailed, Err: fmt.Errorf("org: read manifest: %w", err)}
		}
		events = rr.Events
	}

	activeSeats := ActiveSeatCount(events, p.OrgID, RosterOptions{})
	if err := ValidateSpawnCapacity(o.Config, req, activeSeats); err != nil {
		return o.reject(p, err)
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
	// <state-dir>/prompts/<org_id>-<seat_id>.md and only a short one-line
	// pointer is passed as the agent arg. The write happens here, strictly
	// before AgentStart, so a write failure never reaches the driver at all.
	agentArgs := []string{"--model", p.Model}
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
		return o.failStep(p, "agent_start", err, paneID)
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

	// ensureLeadJoined: best-effort join.sh <team> lead claude-code <cwd>. A
	// clean agmsg team has no "lead" identity registered yet, and agmsg's
	// roster-based send validation rejects HELLO messages whose from/to
	// identity was never join.sh'd (agmsg #355) -- so the saga must attempt
	// to register "lead" before the seat's own Join+Send below. join.sh is
	// treated as idempotent (re-joining an existing member is a documented
	// no-op/soft-fail in agmsg), so a lead-join error here does *not* fail
	// the saga on its own: the definitive, single-authoritative-failure-point
	// gate is the seat Join immediately below and, ultimately, the HELLO
	// Send -- if the roster is genuinely missing "lead", Send fails and the
	// lead-join error recorded here is carried into that failure's Details
	// for diagnosis (see the failStepWithNote call below).
	leadJoinErr := o.Agmsg.Join(ctx, team, "lead", agmsgTypeForDriver("claude"), p.Cwd)
	leadJoinNote := "ok"
	if leadJoinErr != nil {
		leadJoinNote = fmt.Sprintf("error=%v", leadJoinErr)
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnStep,
		PaneID: paneID, AgmsgTeam: team, Details: fmt.Sprintf("agmsg_lead_joined %s", leadJoinNote),
	}); err != nil {
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
	if err := o.Agmsg.Send(ctx, team, p.SeatID, "lead", msg); err != nil {
		return o.failStepWithNote(p, "agmsg_announce", err, paneID, fmt.Sprintf("lead_join=%s", leadJoinNote))
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnStep,
		PaneID: paneID, AgmsgTeam: team, Details: "agmsg_announced",
	}); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
	}

	// Scope has no dedicated ManifestEvent field (see the SpawnParams.Scope
	// doc comment): it is recorded as "scope=<value>" in Details so it
	// stays auditable without a manifest schema change. Empty Scope leaves
	// Details empty, matching pre-Scope behavior exactly.
	spawnedDetails := ""
	if p.Scope != "" {
		spawnedDetails = fmt.Sprintf("scope=%s", p.Scope)
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawned,
		Role: p.Role, Driver: p.Driver, Model: p.Model, Worktree: p.Cwd,
		PaneID: paneID, AgmsgTeam: team, Details: spawnedDetails,
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
// agent_started step's Details for audit purposes.
func (o *Org) agentStartWithRetry(ctx context.Context, name, kind, paneID string, timeoutMS int, agentArgs []string) (int, error) {
	interval := o.agentStartRetryInterval()
	var lastErr error
	for attempt := range maxAgentStartAttempts {
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
	return maxAgentStartAttempts, lastErr
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
// <state-dir>/prompts/<org_id>-<seat_id>.md. state-dir is derived from the
// manifest store's own directory (filepath.Dir(o.Manifest.Path())) rather
// than a separate config field, since the manifest store is already the
// single source of truth for where this Org's on-disk state lives. The path
// is namespaced by both org_id and seat_id so a respawn of the same seat
// overwrites its own prompt file (intentional -- see writePromptFile) while
// two different seats never collide.
func (o *Org) promptFilePath(orgID, seatID string) (string, error) {
	stateDir, err := filepath.Abs(filepath.Dir(o.Manifest.Path()))
	if err != nil {
		return "", fmt.Errorf("resolve state dir for prompt file: %w", err)
	}
	return filepath.Join(stateDir, "prompts", fmt.Sprintf("%s-%s.md", orgID, seatID)), nil
}

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
// they carry no real side effects and no [org].max_seats pressure.
func (o *Org) dryRunSpawn(p SpawnParams) SpawnResult {
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
	agentStartedDetails := "agent_started"
	initialPrompt := p.Prompt
	if rendered, ok, err := RenderRolePrompt(p.Role, RolePromptVars{
		OrgID: p.OrgID, SeatID: p.SeatID, Team: team, Role: p.Role, Scope: p.Scope,
	}); err == nil && ok {
		if p.Prompt != "" {
			initialPrompt = rendered + "\n\n" + p.Prompt
		} else {
			initialPrompt = rendered
		}
	}
	if initialPrompt != "" && needsPromptFile(initialPrompt) {
		if promptPath, perr := o.promptFilePath(p.OrgID, p.SeatID); perr == nil {
			agentStartedDetails = fmt.Sprintf("agent_started prompt_file=%s", promptPath)
		}
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
// Details. Used by the agmsg_announce failure path to carry the
// ensureLeadJoined outcome forward: when HELLO Send fails, whether the
// preceding best-effort lead-join succeeded or errored is essential
// diagnostic context (a missing "lead" roster entry is the most likely root
// cause of a Send rejection), so it must not be lost.
func (o *Org) failStepWithNote(p SpawnParams, step string, cause error, paneID, note string) SpawnResult {
	compensation := compensatePane(o.Herdr, paneID)
	details := fmt.Sprintf("step=%s error=%v compensation=%s", step, cause, compensation)
	if note != "" {
		details += " " + note
	}
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
