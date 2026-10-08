package org

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

// defaultCodexModelObserveTimeout and defaultCodexModelObserveInterval
// bound observeCodexSpawnReceipt's poll when Org.CodexModelObserveTimeout/
// Org.CodexModelObserveInterval are unset (zero value): real-record
// measurements (plan Assumptions) put a codex session's first turn_context
// about 2.3-3.2s after spawn, so 8s at 500ms gives several polls of margin
// beyond the slowest observed case. Tests override both to tiny values so
// the not-found/timeout path stays fast.
const (
	defaultCodexModelObserveTimeout  = 8 * time.Second
	defaultCodexModelObserveInterval = 500 * time.Millisecond
)

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

// LeaderIdentity is the single, grep-able definition of the org's coordinating
// "leader" agmsg identity name (see .claude/rules/ralph/agent-messaging.md's "star
// topology" section). Every production call site that names or targets the
// leader identity (ensureLeaderJoined's Join, Spawn's HELLO Send TO field) must
// use this constant rather than a bare "leader" literal. Exported so
// internal/cli/org.go's newOrgStartCmd (`ralph org start`) can spawn the
// leader seat itself under SeatID == LeaderIdentity, Role == LeaderIdentity
// (design decision: "org start" = the leader-seat spawn sugar, see
// docs/plans/active/2026-08-02-org-runtime-lead.md) without a duplicate
// "leader" literal in that package.
const LeaderIdentity = "leader"

// defaultLeaderDriver is the driver ensureLeaderJoined uses to derive the leader
// identity's agmsg type (agmsgTypeForDriver) when SpawnParams.LeaderDriver is
// left unset -- matches the `ralph org spawn --leader-driver` flag's own
// default.
const defaultLeaderDriver = "claude"

// EventOrgWorkspaceCreated is an org-level event (SeatID empty) recorded
// whenever a herdr workspace is created for an org_id. Later spawns within
// the same org_id reuse the recorded PaneID (the workspace id) instead of
// calling Herdr.WorkspaceCreate again -- one workspace per org, many tabs
// (one per seat) -- until an EventOrgWorkspaceClosed for that id follows.
const EventOrgWorkspaceCreated = "org_workspace_created"

// EventOrgWorkspaceClosed is the org-level event (SeatID empty, PaneID = the
// workspace id) Disband records once the org's workspace is closed, herdr
// reports it not found, or it is the caller's own workspace whose close
// Disband leaves to the caller (see DisbandResult.DeferredSelfWorkspaceID).
// Details say which. With Disband's Force it is also recorded for a close
// that failed. A spawn after it creates a new workspace (resolveWorkspace).
// Like EventOrgWorkspaceCreated it is not a state event.
const EventOrgWorkspaceClosed = "org_workspace_closed"

// HerdrClient is the subset of driver.Herdr's methods the spawn saga and the
// send/wait/read/stop verbs need. Defined here (consumption side, per
// .claude/rules/ralph/architecture.md) rather than in internal/org/driver, so
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
	// PaneClose and WorkspaceClose run `herdr pane close` / `herdr
	// workspace close` on an id the manifest recorded. An id herdr no
	// longer knows (already closed) comes back as an error for which
	// driver.IsNotFound is true; for any other failure it is false.
	PaneClose(ctx context.Context, paneID string) error
	WorkspaceClose(ctx context.Context, workspaceID string) error
	// PaneGet, TabGet and WorkspaceGet run `herdr pane get` / `herdr tab
	// get` / `herdr workspace get`: the ids of the tab and workspace that
	// hold a pane, and the label of a tab or workspace. Stop and Disband
	// read them before a C-c or a close, to confirm a recorded id still
	// names what spawn created (it labels a seat's tab with the seat id and
	// the org's workspace with the org_id; see confirmSeatPane in
	// verbs.go). An unknown id comes back as an error for which
	// driver.IsNotFound is true.
	PaneGet(ctx context.Context, paneID string) (tabID, workspaceID string, err error)
	TabGet(ctx context.Context, tabID string) (label string, err error)
	WorkspaceGet(ctx context.Context, workspaceID string) (label string, err error)
}

// AgmsgClient is the subset of driver.Agmsg's methods the spawn saga and the
// send/stop/disband verbs need. See HerdrClient's doc comment for the
// interface-placement rationale.
type AgmsgClient interface {
	Send(ctx context.Context, team, from, to, message string) error
	// Join registers agentID (agmsg-native agmsgType, e.g. "claude-code" or
	// "codex") on team's roster at projectPath. The spawn saga calls this
	// twice per seat: once for the org's "leader" identity (idempotent,
	// best-effort -- see ensureLeaderJoined in Spawn) and once for the seat
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
	// SendEnterDelay overrides the wait between PaneSendText and
	// PaneSendKeys("Enter") in Send. Zero (the field's default) means "use
	// defaultSendEnterDelay" -- tests set this to a tiny value so send
	// tests run fast; SendParams.EnterDelayMS (when > 0) overrides this
	// per-call instead.
	SendEnterDelay time.Duration
	// SendSubmitConfirmTimeout overrides how long Send waits, after Enter,
	// to confirm the target seat left idle/done. Zero means "use
	// defaultSendSubmitConfirmTimeout".
	SendSubmitConfirmTimeout time.Duration
	// CodexSessionsDir overrides where observeCodexSpawnReceipt/
	// observeStopModelReceipt look for codex session records (see
	// codex_session.go's CodexSessionsDir). Empty (the field's default)
	// means "resolve from the environment at call time" -- CODEX_HOME if
	// set, else <os.UserHomeDir()>/.codex/sessions -- via this package's
	// own codexSessionsDir() method; see its doc comment for the
	// home-directory-unresolvable fallback. Tests set this to an empty
	// t.TempDir() so no test ever observes a developer's real ~/.codex.
	CodexSessionsDir string
	// CodexModelObserveTimeout and CodexModelObserveInterval override how
	// long and how often observeCodexSpawnReceipt polls
	// ObserveCodexEffectiveModel after a codex seat's spawn. Zero (the
	// fields' default) means "use defaultCodexModelObserveTimeout/
	// defaultCodexModelObserveInterval". Tests set both to tiny values so
	// the poll's not-found/timeout path runs fast.
	CodexModelObserveTimeout  time.Duration
	CodexModelObserveInterval time.Duration
	// DriverCallTimeout bounds each herdr / agmsg call Stop makes (the pane,
	// tab and workspace gets of the ownership check, the C-c, the pane
	// close, the agmsg Leave), each workspace get and close Disband makes,
	// and the gets and the close in CloseDeferredSelfPane /
	// CloseDeferredSelfWorkspace, one fresh deadline per call. Zero (the field's default) means "use
	// defaultDriverCallTimeout" -- tests set a tiny value so a call that
	// never answers times out fast.
	DriverCallTimeout time.Duration
	// Getenv overrides how Stop and Disband read the caller's herdr
	// environment (HERDR_PANE_ID and HERDR_WORKSPACE_ID, set by herdr inside
	// every pane). nil (the field's default) means os.Getenv -- tests set it
	// so the result does not depend on whether the test process itself runs
	// inside a herdr pane.
	Getenv func(string) string
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
	// Reserve lists the paths, relative to the repo root, that this spawn
	// reserves for the org (plan 2026-10-08-org-limits-reserve): a path
	// ending in `/` is a directory, any other a file, `.` the whole repo
	// (NormalizeReservePaths, reserve.go). Only the leader seat
	// (SeatID == LeaderIdentity) may pass it, and it is optional. Spawn
	// normalizes it with the input checks and, under the manifest lock,
	// records it as the org's EventScopeReserved when the org holds no
	// reservation and no other running org's reservation overlaps it. The
	// same set again passes, a different set is refused, and the reservation
	// stays until the org is disbanded, also when the spawn fails later. A
	// non-empty Reserve satisfies the AC-2b gate like Scope does.
	Reserve []string
	// LeaderDriver is the driver (claude|codex) the org's coordinating "leader"
	// identity itself runs as -- independent of Driver, which names this
	// seat's own driver. It is only consulted by ensureLeaderJoined to pick
	// the agmsg type ("claude-code"/"codex") registered for the leader
	// identity on the team roster. Empty defaults to defaultLeaderDriver
	// ("claude"), matching the CLI flag's default.
	LeaderDriver string
	// Task is substituted into the seat's role prompt template as {{TASK}}
	// (RolePromptVars.Task, prompts.go). Only prompts/leader.md references
	// {{TASK}} today -- `ralph org start` (internal/cli/org.go's
	// newOrgStartCmd) is the only production caller that sets this field,
	// passing its required positional task argument straight through. Every
	// other --role spawn leaves it empty (harmless: an unreferenced
	// substitution is simply never used, same as Envelope below).
	Task string
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
	// ModelReceipt is the Receipt this call appended, set on every path
	// that successfully appends one: the real spawn path's own observation,
	// reject()'s envelope-rejection receipt, and dryRunSpawn's dry-run
	// receipt all set it -- each only once its own Receipts.Append call has
	// actually succeeded, so this field never names a receipt that was
	// never persisted. It is the zero Receipt on every path that appends no
	// receipt at all, or whose append failed: an idempotent respawn, a
	// plain rejection that never reaches reject() (identifier validation, a
	// retired role name, a retired ralph.toml key; see Spawn's own doc
	// comment), every SpawnOutcomeFailed return, and a
	// reject()/dryRunSpawn receipts-append
	// failure. The CLI layer reads this to decide whether to print the
	// codex model-mismatch warning (AC-6), without re-reading the receipts
	// file -- its gate (Honored=="false" AND a non-empty
	// ReportedEffectiveModel) is what keeps a rejection or dry-run receipt
	// (both honored=false/unknown with no reported model) from ever being
	// printed as a mismatch.
	ModelReceipt Receipt
}

// Spawn runs the full spawn saga described in
// docs/plans/active/2026-08-01-org-runtime-mechanism.md. For a non-dry-run
// call, the ordering is, in this order:
//  0. Input-only checks (identifier shape, the joined herdr agent-name
//     length, RetiredRoleInputErr, and Reserve: leader seat only, paths
//     normalized by NormalizeReservePaths): pure functions of the request,
//     run before the manifest is read, each a plain rejection with no
//     manifest event and no receipt. They run in dry-run mode too.
//  1. Idempotent early return: an already-spawned seat returns the existing
//     seat with no config-dependent validation attempted at all (so an
//     at-cap org can never reject a respawn-of-active-seat retry, a no-op
//     retry under the default autonomous mode can never be rejected by the
//     AC-2b scope gate below either -- see that gate's doc comment for the
//     fix this encodes -- and a retired [org.roles] / [org.permissions.roles]
//     key added after the seat was spawned cannot reject the retry). With
//     Reserve, the reservation is decided first (idempotentRespawn).
//  2. ralph.toml retired-key check (retiredRoleConfigErr): a plain
//     rejection (no `rejected` event, no receipt), run before stale-seat
//     compensation and every manifest write, so only a genuinely new spawn
//     attempt is refused for an old key.
//  3. Stateless envelope validation (ValidateSpawnEnvelope: driver/model
//     pool membership, role restriction; and permissionArgsForDriver:
//     driver + resolved permission mode) -- both are pure functions of
//     cfg+req, run before any external side effect (including stale-seat
//     compensation) is attempted, so an envelope-invalid request is always
//     a pure no-op.
//  4. AC-2b minimum control gate (the autonomous-mode --scope requirement)
//     -- also stateless, but checked after the idempotent return and the
//     envelope/permission checks above so it only ever applies to a
//     genuinely new spawn attempt.
//  5. Stale-in-flight compensation: a stale seat (prior spawn_started/
//     spawn_step never resolved) is best-effort compensated and the
//     manifest re-read, so it no longer counts toward max_seats.
//  6. Capacity validation (ValidateSpawnCapacity) against the recomputed
//     activeSeats, then the org-wide limits (ValidateOrgWideCapacity:
//     max_orgs for an org that is not running yet, max_total_seats), then,
//     with Reserve, the reservation (reservationDecision), all from the same
//     locked read (spawnCapacityErr). A reservation to record is appended as
//     EventScopeReserved right before spawn_started.
//
// Only then does the saga proceed to (unless DryRun) the workspace/tab/
// agent/agmsg side effects with a spawn_started -> spawn_step* ->
// spawned|spawn_failed manifest trail and a tri-state model receipt.
//
// The DryRun path (self-review Cycle-2 M-1 fix) runs steps 0, 2, 3, 4, and
// 6 in the exact same order as the real path above: the input-only checks,
// retiredRoleConfigErr, ValidateSpawnEnvelope, then permissionArgsForDriver,
// then the AC-2b gate, then spawnCapacityErr against the real events.
// Steps 1 and 5 have no dry-run analogue -- dry-run events are excluded
// from ActiveSeatCount/roster entirely, so there is no idempotent-respawn
// case to short-circuit and no stale-in-flight saga to detect or
// compensate. Because the two paths now check the same conditions in the
// same order, a request that fails more than one check (e.g. an
// out-of-pool model *and* a scope-less autonomous spawn) is rejected for
// the same first cause in both modes, so `--dry-run`'s predicted rejection
// always matches what a real spawn of the same request would record.
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
	// A retired role name in the request (retiredRoles, prompts.go: a
	// renamed role's old name as --role or --id, or a removed role spawned
	// with no --prompt) is a plain rejection too, for the same reason as the
	// identifier checks above: it is a property of the input alone. It runs
	// before ResolvePermissionMode and the manifest read, so neither a
	// `rejected` event nor a receipt is written, in dry-run and real mode
	// alike. `ralph org start` spawns through here, so it is covered by the
	// same check. The ralph.toml half of the guard (retiredRoleConfigErr)
	// depends on the config instead and does NOT run here: it runs after the
	// idempotent early return (real path) or first in the dry-run branch, so
	// re-running an already-spawned seat under such a config stays a no-op.
	if err := RetiredRoleInputErr(p.Role, p.SeatID, p.Prompt); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeRejected, Err: err}
	}
	// Reserve is input too: the org's reservation is taken by its leader seat
	// only, and its paths must follow NormalizeReservePaths' rules. Both are
	// plain rejections like the checks above. From here on p.Reserve holds
	// the normalized paths, so every later comparison and record uses them.
	if len(p.Reserve) > 0 {
		if p.SeatID != LeaderIdentity {
			return SpawnResult{Outcome: SpawnOutcomeRejected, Err: fmt.Errorf(
				"org: only the %s seat reserves paths for its org; seat_id %q cannot (spawn it without reserve paths)",
				LeaderIdentity, p.SeatID,
			)}
		}
		reserve, err := NormalizeReservePaths(p.Reserve)
		if err != nil {
			return SpawnResult{Outcome: SpawnOutcomeRejected, Err: err}
		}
		p.Reserve = reserve
	}

	// resolvedPermMode is a pure function of cfg+role, computed once here so
	// every later consumer (the AC-2b gate, permissionArgsForDriver, the
	// spawned event's Details, dryRunSpawn) sees the same value. Computing it
	// does not itself decide anything -- the AC-2b gate check that used to
	// sit right here has moved: see autonomousScopeGateErr's doc comment for
	// why (PR① precedent: an idempotent early return must precede any
	// validation a no-op retry doesn't need).
	resolvedPermMode := ResolvePermissionMode(o.Config, p.Role)

	if p.TimeoutMS <= 0 {
		p.TimeoutMS = defaultSpawnTimeoutMS
	}

	rr, err := o.Manifest.Read()
	if err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: fmt.Errorf("org: read manifest: %w", err)}
	}
	events := rr.Events

	if p.DryRun {
		// Dry-run mirrors the real path's ordering exactly (self-review
		// Cycle-2 M-1 fix): retiredRoleConfigErr, then
		// ValidateSpawnEnvelope, then permissionArgsForDriver, then the
		// AC-2b gate, then spawnCapacityErr (max_seats, the org-wide limits,
		// the reservation) -- the same first-cause-wins order the real
		// path's locked closure uses below, minus the two steps that have no
		// dry-run analogue (the idempotent early return and stale-in-flight
		// compensation; dry-run events are excluded from ActiveSeatCount/
		// roster entirely, so neither concept applies here). No manifest lock
		// is needed either -- dry-run events never count toward
		// [org].max_seats, the org-wide limits or the reservations, so two
		// concurrent dry-runs cannot race on them.
		//
		// The ralph.toml retired-key check (retiredRoleConfigErr) is a plain
		// rejection with no manifest event or receipt, unlike the checks
		// after it (those go through reject()). The real path runs it right
		// after its idempotent early return; dry-run has no idempotent case,
		// so first is the matching position.
		if err := retiredRoleConfigErr(o.Config); err != nil {
			return SpawnResult{Outcome: SpawnOutcomeRejected, Err: err}
		}
		req := SpawnRequest{OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver, Model: p.Model}
		if err := ValidateSpawnEnvelope(o.Config, req); err != nil {
			return o.reject(p, err)
		}
		// Permission-mode mapping validation (AC-2, codex fail-closed) runs
		// in dry-run too -- validate-then-record contract, same as the
		// envelope check just above. dryRunSpawn never calls AgentStart, so
		// the resolved args themselves are discarded; only the possible
		// error matters here.
		if _, err := permissionArgsForDriver(o.Config, p.Driver, resolvedPermMode); err != nil {
			return o.reject(p, err)
		}
		if err := autonomousScopeGateErr(p, resolvedPermMode); err != nil {
			return o.reject(p, err)
		}
		reserve, err := spawnCapacityErr(o.Config, p, req, events)
		if err != nil {
			return o.reject(p, err)
		}
		return o.dryRunSpawn(p, resolvedPermMode, reserve)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.TimeoutMS)*time.Millisecond)
	defer cancel()

	// The idempotent/retired-key/envelope/permission/AC-2b-gate checks,
	// stale-in-flight detection, the capacity check, and the spawn_started
	// append all run inside withManifestLock: this is the exact "read
	// manifest -> ActiveSeatCount -> ValidateSpawn -> appendEvent" window
	// docs/tech-debt/README.md
	// flagged as an unlocked TOCTOU race ("max_seats is enforced across an
	// unlocked read-then-append window"). Two concurrent Spawn calls used to
	// be able to both observe the same activeSeats snapshot and both pass
	// ValidateSpawnCapacity, exceeding [org].max_seats; the flock in
	// withManifestLock (lockfile.go) now serializes this section across
	// goroutines and processes on the same host.
	//
	// Final locking shape (fixed self-review MEDIUM-3: the lock used to also
	// wrap compensateStale's herdr PaneSendKeys round trip, contradicting
	// this very comment and risking a concurrent spawn timing out on
	// manifestLockTimeout -- 5s -- behind a compensation call bounded by the
	// much longer p.TimeoutMS): when a stale in-flight seat is detected, the
	// lock below stops at that detection (no compensation, no capacity
	// check, no append) and releases; compensateStale's driver round trip
	// runs lock-free immediately after; a *second* withManifestLock call
	// then re-reads the manifest and runs the capacity check + spawn_started
	// append against that fresh, lock-held snapshot (checkCapacityAndStart).
	// A fresh read is required rather than reusing the first lock's snapshot
	// because compensateStale's own manifest write happened without the
	// lock held -- another concurrent spawn could have raced in during that
	// window, so the capacity decision must be made against a snapshot taken
	// while the (re-acquired) lock is held, preserving the same TOCTOU
	// guarantee as the non-stale path below. The lock is never scoped over
	// the workspace/agent/agmsg side effects that follow Spawn's own locked
	// section(s) -- per the plan's rollout note, a long-running herdr/agmsg
	// round trip (bounded by p.TimeoutMS, up to defaultSpawnTimeoutMS) must
	// not serialize every concurrent spawn behind it.
	var existing *SeatStatus
	var early *SpawnResult
	// permArgs is set inside the locked closure below (permissionArgsForDriver's
	// success value) and consumed after the lock releases, when Spawn builds
	// AgentStart's agentArgs. It stays nil on any early-return path that
	// precedes its own assignment (idempotent respawn, retired-key or
	// envelope/permission rejection) -- none of those paths ever reach the
	// AgentStart call that would read it. The one early-return path that
	// runs *after* the assignment -- the AC-2b scope-gate rejection just
	// below it -- also never reaches AgentStart, so a non-nil permArgs on
	// that path is harmless: nothing reads it once Spawn has already
	// returned.
	var permArgs []string
	// staleExisting is set inside Phase 1's locked closure when the target
	// seat has a stale in-flight saga (a prior spawn_started/spawn_step that
	// never resolved) -- compensation is deferred to after Phase 1's lock
	// releases (see the doc comment above).
	var staleExisting *SeatStatus
	// spawnStartedAt is checkCapacityAndStart's own capture of the instant
	// (via o.nowTime(), the same clock o.now() formats onto the
	// spawn_started event's TS) it appended that event -- the codex
	// model-observation step far below needs this exact value to correlate
	// a codex session record with this spawn, not a fresh time.Now() taken
	// after the workspace/agent/agmsg round trip that follows. Set on
	// whichever of Phase 1/Phase 2 below actually appends spawn_started;
	// stays the zero value on every early-return path, which never reaches
	// the observation step.
	var spawnStartedAt time.Time
	req := SpawnRequest{OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver, Model: p.Model}

	// Phase 1 (locked): fresh read, idempotent/retired-key/envelope/permission
	// checks, and stale-in-flight *detection*. No herdr/agmsg call happens
	// while this lock is held.
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
			// no-op must not be able to fail validation. A retry that
			// carries Reserve is the one exception: idempotentRespawn
			// decides the reservation before returning the seat.
			r := o.idempotentRespawn(p, *existing, events)
			early = &r
			return nil
		}

		// The ralph.toml retired-key check (retiredRoleConfigErr) runs here,
		// right after the idempotent return and before stale-in-flight
		// detection/compensation and every manifest write: an old key in the
		// config must not turn a re-run of an already-spawned seat into a
		// rejection, but it must still refuse every new spawn. It is a plain
		// rejection like the input checks at the top of Spawn (not reject()),
		// so no `rejected` event and no receipt are written.
		if err := retiredRoleConfigErr(o.Config); err != nil {
			r := SpawnResult{Outcome: SpawnOutcomeRejected, Err: err}
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
		args, permErr := permissionArgsForDriver(o.Config, p.Driver, resolvedPermMode)
		if permErr != nil {
			r := o.reject(p, permErr)
			early = &r
			return nil
		}
		permArgs = args

		// AC-2b minimum control gate (Codex advisory 1): an autonomous-mode
		// seat runs its driver with no interactive permission dialog at all
		// (permissionArgsForDriver's bypassPermissions) -- --scope is the
		// only thing left standing between "autonomous" and "unrestricted".
		// A scope-less autonomous spawn is fail-closed here, unless the
		// caller explicitly opts out via AllowUnscoped (recorded on the
		// spawned event's Details below so its use stays auditable).
		//
		// This check runs *after* the idempotent early return and the
		// envelope/permission checks above, but *before* stale-in-flight
		// detection and the capacity check below -- cross-review triage
		// fix (docs/reports/cross-review-triage-org-runtime-lead.md,
		// ACTION_REQUIRED #1): this gate used to sit before Spawn's manifest
		// read entirely, ahead of the idempotent check further up in this
		// closure, so a bare retry of `spawn` for an already-spawned seat
		// (no new autonomous seat being created at all) was rejected by this
		// gate before it ever reached the idempotent return above -- the
		// exact same idempotent-vs-validation ordering bug PR① fixed for
		// envelope validation (see this func's own doc comment, item 1).
		// A no-op respawn of an existing active seat needs no --scope, so
		// the gate must never be reachable before the idempotent return has
		// had a chance to fire.
		//
		// Routed through reject() (self-review LOW finding), same as every
		// other envelope-validation rejection: a `rejected` manifest event
		// plus an honored=false receipt are appended, so an autonomous leader
		// that retry-loops unscoped spawns for a genuinely new seat leaves a
		// visible trace in `ralph org report`'s timeline instead of none.
		// No spawn_started is ever written for this path (reject() never
		// appends one), so a rejected attempt still cannot count toward
		// [org].max_seats or leave a stale saga behind -- only the audit
		// trail changed, not the fail-closed semantics.
		if err := autonomousScopeGateErr(p, resolvedPermMode); err != nil {
			r := o.reject(p, err)
			early = &r
			return nil
		}

		if existing != nil && (existing.Event == EventSpawnStarted || existing.Event == EventSpawnStep) {
			// Stale in-flight saga from a prior crashed/interrupted spawn:
			// record it for lock-free compensation right after this closure
			// returns, then let Phase 2 (below) re-acquire the lock for a
			// fresh capacity check + append -- see the doc comment above
			// Phase 1 for why compensation itself must not run in here.
			e := *existing
			staleExisting = &e
			return nil
		}

		early, spawnStartedAt = checkCapacityAndStart(o, p, req, events)
		return nil
	})
	if lockErr != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: fmt.Errorf("org: manifest lock: %w", lockErr)}
	}
	if early != nil {
		return *early
	}

	if staleExisting != nil {
		// Lock-free: best-effort compensate the stale seat (a herdr
		// PaneSendKeys round trip) and mark it spawn_failed, exactly as
		// Phase 1 used to do while still holding the lock.
		o.compensateStale(ctx, p, *staleExisting)
		afterStaleCompensation()

		// Phase 2 (locked): re-read post-compensation and re-run BOTH checks
		// that could have gone stale during the lock-free compensate window
		// above (self-review Cycle-2 M-2 fix):
		//   - idempotency: a concurrent racer's Spawn call for this exact
		//     seat could have completed a full saga while our lock was
		//     released for compensateStale's herdr round trip. Phase 1's
		//     EventSpawned early return only guards against a racer that
		//     was already spawned *before* our own Phase 1 read -- it says
		//     nothing about a racer finishing in the window between our
		//     Phase 1 release and this re-acquire. Re-scanning the fresh
		//     roster here for that same EventSpawned terminus and returning
		//     Idempotent, instead of falling through to
		//     checkCapacityAndStart, is what stops this call from
		//     appending a second spawn_started on top of an already-
		//     spawned seat.
		//   - capacity: unchanged from before -- the now-terminal stale
		//     seat no longer counts toward activeSeats, so
		//     checkCapacityAndStart runs the same capacity check +
		//     spawn_started append Phase 1 would have run had no stale seat
		//     been in the way.
		// A racer whose own saga is still in-flight (spawn_started/
		// spawn_step) needs no extra handling here: ActiveSeatCount already
		// counts that roster entry as active, so checkCapacityAndStart's
		// capacity check already treats it correctly, exactly as it would
		// for any other in-flight seat. And if the fresh read instead shows
		// our target seat is no longer stale because a racer's own
		// compensateStale already resolved it (event == spawn_failed), no
		// duplicate spawn_failed is at risk from this closure: it never
		// appends one -- only the idempotent check above and
		// checkCapacityAndStart's reject()/spawn_started path below do --
		// so re-validating "still stale" here requires no action beyond the
		// idempotent check itself.
		lockErr = withManifestLock(filepath.Dir(o.Manifest.Path()), func() error {
			rr2, err := o.Manifest.Read()
			if err != nil {
				return fmt.Errorf("org: read manifest: %w", err)
			}
			events = rr2.Events

			roster := Roster(events, RosterOptions{})
			for i := range roster {
				if roster[i].OrgID == p.OrgID && roster[i].SeatID == p.SeatID {
					if roster[i].Event == EventSpawned {
						r := o.idempotentRespawn(p, roster[i], events)
						early = &r
						return nil
					}
					break
				}
			}

			early, spawnStartedAt = checkCapacityAndStart(o, p, req, events)
			return nil
		})
		if lockErr != nil {
			return SpawnResult{Outcome: SpawnOutcomeFailed, Err: fmt.Errorf("org: manifest lock: %w", lockErr)}
		}
		if early != nil {
			return *early
		}
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

	// AC-4: a known --role expands the embedded template (leader.md /
	// implementer.md / reviewer.md) into the initial prompt;
	// --prompt, if also given, is appended after it. An unknown role leaves
	// initialPrompt as plain --prompt (possibly empty) -- no error, no
	// fallback template. Task and Envelope are only referenced by
	// prompts/leader.md today; every other template ignores them.
	initialPrompt := p.Prompt
	rendered, ok, err := RenderRolePrompt(p.Role, RolePromptVars{
		OrgID: p.OrgID, SeatID: p.SeatID, Team: team, Role: p.Role, Scope: p.Scope,
		Task: p.Task, Envelope: EnvelopeSummary(o.Config),
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
	// mode behavior. A workspace-write codex leader seat whose cwd does not
	// contain the state dir (the manifest store's directory, as in
	// promptFilePath) also gets codexWritableRootArgs' --add-dir right after
	// permArgs, still before --model; other roles never do (plan
	// 2026-10-07-org-state-dir-common, AC8).
	agentArgs := append([]string{}, permArgs...)
	agentArgs = append(agentArgs, codexWritableRootArgs(p.Role, p.Driver, resolvedPermMode, p.Cwd, filepath.Dir(o.Manifest.Path()))...)
	agentArgs = append(agentArgs, "--model", p.Model)
	agentStartedDetails := "agent_started"
	// promptPath is hoisted to this outer scope (rather than declared
	// inside the needsPromptFile branch below, as before) because the
	// codex model-observation step far below needs it too, to correlate a
	// codex session record with this exact spawn (AC-2c: a seat with no
	// role-prompt file at all -- prompt passed inline, or none -- has
	// nothing to correlate a session record with, so it stays ""). It is
	// also embedded in agentStartedDetails below via
	// codexPromptFileDetailsPrefix, the same constant Stop's own
	// correlation (verbs.go's codexSpawnCorrelation) parses back out of
	// the persisted spawn_step Details, so the two call sites cannot drift
	// out of sync with each other.
	var promptPath string
	if initialPrompt != "" {
		if needsPromptFile(initialPrompt) {
			var perr error
			promptPath, perr = o.promptFilePath(p.OrgID, p.SeatID)
			if perr != nil {
				return o.failStep(p, "prompt_file", perr, paneID)
			}
			if err := writePromptFile(promptPath, initialPrompt); err != nil {
				return o.failStep(p, "prompt_file", err, paneID)
			}
			agentArgs = append(agentArgs, PromptFilePointer(promptPath))
			agentStartedDetails = codexPromptFileDetailsPrefix + promptPath
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
		// This must stay the LAST thing appended to agentStartedDetails:
		// promptPathFromAgentStartedDetails (verbs.go) strips this suffix
		// only when it runs to the very end of the string. A field added
		// after this point would silently become part of the "path"
		// codexSpawnCorrelation recovers at Stop time, breaking the
		// recovered pointer sentence's match against the session record
		// the same way an unstripped suffix once did. A future field
		// belongs before `prompt_file=` instead.
		agentStartedDetails = fmt.Sprintf("%s %s%d", agentStartedDetails, codexAgentStartRetriesDetailsSuffixKey, retries)
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnStep,
		PaneID: paneID, Details: agentStartedDetails,
	}); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
	}

	// leaderSelfSpawn is true exactly when this Spawn call's own SeatID is the
	// leader identity itself: `ralph org start` (internal/cli/org.go's
	// newOrgStartCmd) spawns SeatID == LeaderIdentity, Role == LeaderIdentity by
	// design ("org start" = the leader-seat spawn sugar, see
	// docs/plans/active/2026-08-02-org-runtime-lead.md, "Design decisions").
	// In that one case, the seat Join call just below IS the leader-identity
	// join -- there is no separate coordinating identity to announce to --
	// so a preceding ensureLeaderJoined call would just re-join the exact same
	// identity a moment later, and a HELLO from leader announcing itself to
	// leader would violate the star topology's single-coordinator premise
	// (.claude/rules/ralph/agent-messaging.md: every non-leader seat addresses
	// TO: leader; leader has no "TO: leader" of its own). Both steps are skipped
	// only for this case; every other --role spawn still gets both,
	// unchanged.
	leaderSelfSpawn := p.SeatID == LeaderIdentity

	var leaderJoinNote string
	if !leaderSelfSpawn {
		note, err := o.ensureLeaderJoined(ctx, p, team, paneID)
		if err != nil {
			return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
		}
		leaderJoinNote = note
	}

	if err := o.Agmsg.Join(ctx, team, p.SeatID, agmsgTypeForDriver(p.Driver), p.Cwd); err != nil {
		return o.failStep(p, "agmsg_join", err, paneID)
	}
	joinedDetails := "agmsg_joined"
	if leaderSelfSpawn {
		joinedDetails = "agmsg_joined leader_self=true"
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnStep,
		PaneID: paneID, AgmsgTeam: team, Details: joinedDetails,
	}); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
	}

	if !leaderSelfSpawn {
		// AC-11: the HELLO body must itself be protocol.ValidateText-conformant
		// (see TestSpawn_HelloMessage_IsProtocolConformant) -- HELLO does not
		// require TASK_ID, so a TYPE header plus these fields alone is valid.
		msg := fmt.Sprintf("TYPE: HELLO\nSEAT: %s\nROLE: %s\nORG_ID: %s", p.SeatID, p.Role, p.OrgID)
		if err := o.Agmsg.Send(ctx, team, p.SeatID, LeaderIdentity, msg); err != nil {
			// tech-debt (docs/tech-debt/README.md, "spawn の agmsg_announce(HELLO
			// send)失敗パスの補償..."): the seat's own Join already succeeded by
			// this point, so a failed HELLO announce must not leave a stale
			// roster entry behind -- best-effort Leave it back out, and record
			// the outcome in spawn_failed's Details alongside the leader-join note
			// so both compensation steps stay auditable from the manifest alone.
			leaveNote := compensateLeave(o.Agmsg, team, p.SeatID)
			return o.failStepWithNote(p, "agmsg_announce", err, paneID, fmt.Sprintf("leader_join=%s leave=%s", leaderJoinNote, leaveNote))
		}
		if err := o.appendEvent(ManifestEvent{
			TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnStep,
			PaneID: paneID, AgmsgTeam: team, Details: "agmsg_announced",
		}); err != nil {
			return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
		}
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

	// The model receipt is built after `spawned`, before it is appended:
	// for a codex seat with a role-prompt file, observeCodexSpawnReceipt
	// polls ObserveCodexEffectiveModel (codex_session.go) against
	// spawnStartedAt, up to o.codexModelObserveTimeout() -- see its own
	// doc comment for the full true/false/unknown decision table (plan
	// AC-2/AC-2b/AC-2c). Every other seat (claude, or a codex seat with no
	// role-prompt file to correlate) keeps today's receipt text exactly.
	receipt := Receipt{OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver, CommandedModel: p.Model}
	if p.Driver == "codex" {
		receipt = o.observeCodexSpawnReceipt(ctx, receipt, promptPath, spawnStartedAt)
	} else {
		receipt.Honored = HonoredUnknown
		receipt.Reason = "interactive session; effective model not yet observable"
	}
	receipt.TS = o.now()
	if err := o.Receipts.Append(receipt); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
	}

	return SpawnResult{Outcome: SpawnOutcomeSpawned, Seat: SeatStatus{
		OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver, Model: p.Model,
		Worktree: p.Cwd, PaneID: paneID, AgmsgTeam: team, HerdrAgentName: agentName,
		Event: EventSpawned, Active: true,
	}, ModelReceipt: receipt}
}

// checkCapacityAndStart runs the capacity check + spawn_started append
// against the given events snapshot for org o, the in-flight spawn params p,
// and the already-built SpawnRequest req. It takes its dependencies as
// explicit parameters (rather than as a closure capturing enclosing-scope
// variables) so it reads and tests the same whether it is called from Phase 1
// (no-stale-seat path) or Phase 2 (post-compensation re-check) in Spawn --
// both call sites must be inside a locked closure, since neither the capacity
// check nor the spawn_started append is safe unlocked. The returned
// *SpawnResult is non-nil only on a capacity rejection or an append failure;
// callers should assign it to their enclosing `early` var and `return nil`
// right after, exactly as the closure this replaced did. The second return
// value is the instant (o.nowTime()) the spawn_started event's TS was
// formatted from -- zero on either failure path, meaningful only when the
// first return is nil -- so a codex spawn's later model observation
// correlates against the exact time this saga's own spawn_started was
// recorded, not a fresh read taken after the workspace/agent/agmsg round
// trip that follows in Spawn.
//
// The capacity check is spawnCapacityErr (max_seats, then the org-wide
// limits, then the reservation). When it says the reservation is to be
// recorded, the EventScopeReserved is appended here, before spawn_started,
// so it is in the manifest before the lock is released and a racing spawn of
// another org sees it; it stays when the saga fails later.
func checkCapacityAndStart(o *Org, p SpawnParams, req SpawnRequest, events []ManifestEvent) (*SpawnResult, time.Time) {
	reserve, err := spawnCapacityErr(o.Config, p, req, events)
	if err != nil {
		r := o.reject(p, err)
		return &r, time.Time{}
	}
	if reserve {
		if err := o.appendEvent(scopeReservedEvent(o.now(), p.OrgID, p.Reserve, "", false)); err != nil {
			r := SpawnResult{Outcome: SpawnOutcomeFailed, Err: fmt.Errorf("org: record %s: %w", EventScopeReserved, err)}
			return &r, time.Time{}
		}
	}
	startedAt := o.nowTime()
	if err := o.appendEvent(ManifestEvent{
		TS: startedAt.UTC().Format(time.RFC3339), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnStarted,
		Role: p.Role, Driver: p.Driver, Model: p.Model, Worktree: p.Cwd,
	}); err != nil {
		r := SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
		return &r, time.Time{}
	}
	return nil, startedAt
}

// spawnCapacityErr runs, against events, every check of a new seat that
// depends on the manifest, in this order: max_seats of the org
// (ValidateSpawnCapacity), the org-wide max_orgs and max_total_seats
// (ValidateOrgWideCapacity, from RunningOrgs and TotalActiveSeats), and,
// when p.Reserve (already normalized) is set, the reservation
// (reservationDecision). reserve is true when the reservation is new and the
// caller must record it. The real path calls this under the manifest lock
// (checkCapacityAndStart); the dry-run path calls it on its unlocked read of
// the real events and only predicts.
func spawnCapacityErr(cfg config.OrgConfig, p SpawnParams, req SpawnRequest, events []ManifestEvent) (reserve bool, err error) {
	if err := ValidateSpawnCapacity(cfg, req, ActiveSeatCount(events, p.OrgID, RosterOptions{})); err != nil {
		return false, err
	}
	if err := ValidateOrgWideCapacity(cfg, req, RunningOrgs(events), TotalActiveSeats(events)); err != nil {
		return false, err
	}
	if len(p.Reserve) == 0 {
		return false, nil
	}
	return reservationDecision(events, p.OrgID, p.Reserve)
}

// idempotentRespawn is the idempotent return for a spawn of a seat that is
// already spawned: it returns that seat with no new record, as before,
// unless p.Reserve is set (only possible for the leader seat). Then the
// reservation is decided first against the same locked events
// (reservationDecision): an org without one records it, the same set passes,
// and a different set or an overlap with another running org is refused. A
// refusal is a plain rejection (no `rejected` event, no receipt): a
// `rejected` for the seat would become its latest state event and show the
// running leader inactive, and the seat must stay as it is.
func (o *Org) idempotentRespawn(p SpawnParams, seat SeatStatus, events []ManifestEvent) SpawnResult {
	if len(p.Reserve) > 0 {
		record, err := reservationDecision(events, p.OrgID, p.Reserve)
		if err != nil {
			return SpawnResult{Outcome: SpawnOutcomeRejected, Err: err}
		}
		if record {
			if err := o.appendEvent(scopeReservedEvent(o.now(), p.OrgID, p.Reserve, "", false)); err != nil {
				return SpawnResult{Outcome: SpawnOutcomeFailed, Err: fmt.Errorf("org: record %s: %w", EventScopeReserved, err)}
			}
		}
	}
	return SpawnResult{Outcome: SpawnOutcomeIdempotent, Seat: seat}
}

// RetiredRoleInputErr is the input half of Spawn's guard for the role names
// in retiredRoles (prompts.go): it returns an error naming the successor
// when role or seatID is a retired name in a position the table rejects;
// nil otherwise. It reads only the three arguments, so it is a pure check
// on the request, like identifier validation: Spawn runs it before the
// manifest read, and `ralph org spawn` also runs it before the --model
// fallback so a retired name is refused before any fallback warning or
// model_pool error. Matching is exact and case-sensitive, like the rest of
// the role handling.
//
//   - role <renamed>: rejected whatever prompt says, because the old name
//     used to select the coordinator's template and permission mode.
//   - role <removed> with an empty prompt: rejected, because the template is
//     gone and the seat would start with nothing to do. With a prompt the
//     name is an ordinary custom role and the seat starts with only that
//     text (RenderRolePrompt finds no template for it).
//   - seatID <renamed>: rejected whatever the role or prompt, because the old
//     name was the coordinator's agmsg identity -- a seat registered under it
//     would receive the messages that a procedure written for the old
//     binary addresses to the coordinator. A removed name was never an
//     identity anything addresses, so a removed name as seatID is accepted.
//
// The ralph.toml key half is retiredRoleConfigErr, which Spawn runs later
// (after the idempotent early return); see its doc comment for why.
func RetiredRoleInputErr(role, seatID, prompt string) error {
	if r, ok := retiredRoles[role]; ok && r.Kind == retiredRoleRenamed {
		return fmt.Errorf("org: role %q was renamed to %q: use --role %s", role, r.Successor, r.Successor)
	}
	if r, ok := retiredRoles[role]; ok && r.Kind == retiredRoleRemoved && prompt == "" {
		return fmt.Errorf("org: role %q was removed: its deterministic-gate re-run moved to the %q role; "+
			"spawn with --role %s, or pass --prompt to run a custom %q seat",
			role, r.Successor, r.Successor, role)
	}
	if r, ok := retiredRoles[seatID]; ok && r.Kind == retiredRoleRenamed {
		return fmt.Errorf("org: seat id %q is retired: it was the coordinator's agmsg identity and is now %q, "+
			"so a seat by that name would receive messages addressed by the old procedure "+
			"(pick another --id; the coordinator itself is %q)", seatID, r.Successor, r.Successor)
	}
	return nil
}

// retiredRoleConfigErr is the ralph.toml half of Spawn's guard for the
// names in retiredRoles: it returns an error naming every key in cfg that
// still uses a renamed role's old name (RetiredRoleConfigKeys) and the key
// to rename it to; nil otherwise. config.Load does not validate role names,
// so a key under the old name loads fine and is never read again. A
// `lead = "guarded"` ignored that way would run the leader with the full
// model_pool and with [org.permissions].default instead of the mode the key
// asked for, so a new spawn is refused with the exact key to rename. Only
// spawn (and so `org start`) refuses; the other verbs and stop / disband
// keep working so an old org can still be cleaned up.
//
// Unlike RetiredRoleInputErr this depends on the config rather than on the
// request, and the config can gain such a key after a seat was spawned (an
// org started by an older binary, or ralph.toml edited mid-org). On the
// real path Spawn therefore runs it inside the locked closure right after
// the idempotent early return, so re-running an already-spawned seat stays
// a no-op; the dry-run path, which has no idempotent case, runs it first.
func retiredRoleConfigErr(cfg config.OrgConfig) error {
	keys := RetiredRoleConfigKeys(cfg)
	if len(keys) == 0 {
		return nil
	}
	old := make([]string, len(keys))
	renamed := make([]string, len(keys))
	for i, k := range keys {
		old[i], renamed[i] = k.Key, k.RenameTo
	}
	return fmt.Errorf("org: ralph.toml has retired role key(s) %s: rename to %s "+
		"(the old key is no longer read, so a permission mode or model list set under it "+
		"would be silently ignored and the role would fall back to the full model_pool "+
		"and to [org.permissions].default)",
		strings.Join(old, ", "), strings.Join(renamed, ", "))
}

// autonomousScopeGateErr reports the AC-2b minimum control gate's error when
// mode (the already-resolved ResolvePermissionMode(o.Config, p.Role) value)
// is autonomous, p.Scope and p.Reserve are both empty, and the caller has not
// explicitly opted out via p.AllowUnscoped -- nil otherwise. A reservation
// names the paths the org works on, so it satisfies the gate the same way a
// --scope does. Extracted to a pure function so both call sites
// (dryRunSpawn's gate-before-record check inside Spawn's
// `if p.DryRun` branch, and the real-spawn path's post-idempotent,
// post-envelope-validation check inside Spawn's locked closure) share the
// exact same rejection condition and error text; see those two call sites'
// doc comments for why each is ordered where it is.
func autonomousScopeGateErr(p SpawnParams, mode string) error {
	if mode == PermissionModeAutonomous && p.Scope == "" && len(p.Reserve) == 0 && !p.AllowUnscoped {
		return fmt.Errorf(
			"org: autonomous permission mode requires --scope or --reserve (--reserve on the leader seat only; or --allow-unscoped to explicitly bypass)",
		)
	}
	return nil
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

// ensureLeaderJoined best-effort join.sh's <team> leader <type> <cwd>, where
// <type> is agmsgTypeForDriver(p.LeaderDriver) (defaultLeaderDriver ("claude")
// when p.LeaderDriver is unset) -- the leader identity's own driver is
// independent of this seat's Driver, so a Codex-coordinated org must not
// register "leader" under a hardcoded claude-code type (tech-debt,
// docs/tech-debt/README.md, "lead identity is a bare 'lead' string literal
// ... and its agmsg type is hardcoded agmsgTypeForDriver('claude')"). A
// clean agmsg team has no "leader" identity registered yet, and agmsg's
// roster-based send validation rejects HELLO messages whose from/to
// identity was never join.sh'd (agmsg #355) -- so the saga must attempt to
// register LeaderIdentity before the seat's own Join+Send that follows it in
// Spawn. join.sh is treated as idempotent (re-joining an existing member is
// a documented no-op/soft-fail in agmsg), so a leader-join error here does
// *not* fail the saga on its own: the definitive, single-authoritative-
// failure-point gate is the seat Join immediately after this call and,
// ultimately, the HELLO Send -- if the roster is genuinely missing
// LeaderIdentity, Send fails and the leader-join error recorded here is carried
// into that failure's Details for diagnosis (see failStepWithNote's doc
// comment).
//
// The returned string is the "agmsg_leader_joined <note>" note recorded on the
// step's manifest event -- "ok" on success, "error=<err>" otherwise -- so
// Spawn can also fold it into a later failure's Details. The returned error
// is only non-nil when appending that manifest event itself fails (a
// manifest-write failure, not a Join failure); a Join failure is captured in
// the returned note instead of being treated as fatal, per the doc comment
// above.
func (o *Org) ensureLeaderJoined(ctx context.Context, p SpawnParams, team, paneID string) (string, error) {
	leaderDriver := p.LeaderDriver
	if leaderDriver == "" {
		leaderDriver = defaultLeaderDriver
	}
	leaderJoinErr := o.Agmsg.Join(ctx, team, LeaderIdentity, agmsgTypeForDriver(leaderDriver), p.Cwd)
	leaderJoinNote := "ok"
	if leaderJoinErr != nil {
		leaderJoinNote = fmt.Sprintf("error=%v", leaderJoinErr)
	}
	if err := o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventSpawnStep,
		PaneID: paneID, AgmsgTeam: team, Details: fmt.Sprintf("agmsg_leader_joined %s", leaderJoinNote),
	}); err != nil {
		return leaderJoinNote, err
	}
	return leaderJoinNote, nil
}

// agmsgTeam is the team name convention used to announce a newly spawned
// seat to the org's leader (see plan Open questions -- provisional pending
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

// codexModelObserveTimeout returns o.CodexModelObserveTimeout, falling back
// to defaultCodexModelObserveTimeout when unset.
func (o *Org) codexModelObserveTimeout() time.Duration {
	if o.CodexModelObserveTimeout > 0 {
		return o.CodexModelObserveTimeout
	}
	return defaultCodexModelObserveTimeout
}

// codexModelObserveInterval returns o.CodexModelObserveInterval, falling
// back to defaultCodexModelObserveInterval when unset.
func (o *Org) codexModelObserveInterval() time.Duration {
	if o.CodexModelObserveInterval > 0 {
		return o.CodexModelObserveInterval
	}
	return defaultCodexModelObserveInterval
}

// codexSessionsDir returns o.CodexSessionsDir when set, otherwise resolves
// the same way codex itself does at call time: CODEX_HOME if non-empty,
// else <os.UserHomeDir()>/.codex/sessions (via CodexSessionsDir,
// codex_session.go). If CODEX_HOME is empty and the home directory cannot
// be resolved, this returns "" -- ObserveCodexEffectiveModel already treats
// an empty sessionsDir as not-found with a nil error, so that single
// fallback is enough; no separate error path is needed here.
func (o *Org) codexSessionsDir() string {
	if o.CodexSessionsDir != "" {
		return o.CodexSessionsDir
	}
	codexHome := os.Getenv("CODEX_HOME")
	home, err := os.UserHomeDir()
	if err != nil {
		if codexHome == "" {
			return ""
		}
		home = ""
	}
	return CodexSessionsDir(codexHome, home)
}

// observeCodexSpawnReceipt builds (but does not append) the model receipt
// for a real, non-dry-run codex spawn, layering the observed outcome onto
// base (already carrying OrgID/SeatID/Role/Driver/CommandedModel; TS is
// left for the caller to set right before appending, same as every other
// receipt in this file). promptPath is the role-prompt file path this
// spawn's initial prompt was written to -- "" when the prompt was short
// enough to pass inline, or absent entirely. AC-2c: with no prompt file
// there is nothing to correlate a codex session record with, so this
// returns immediately with the "no role-prompt file" unknown receipt,
// never calling the observer (no waiting).
//
// Otherwise it polls: call ObserveCodexEffectiveModel immediately, then
// again every o.codexModelObserveInterval() until it reports found or
// ambiguous, or o.codexModelObserveTimeout() elapses -- an ambiguous
// result ends the poll at once, since waiting cannot resolve two matching
// session records into one. obsCtx (derived from ctx, bounded by
// o.codexModelObserveTimeout()) is threaded into every
// ObserveCodexEffectiveModel call, not just the waits between them: a
// single pass that runs long -- many non-matching candidates, or a large
// file -- is itself bounded by the same budget, not only checked for
// between polls. Each wait goes through waitOrCtxDone (verbs.go), which
// select{}s obsCtx against a timer, so a still-running poll never sleeps
// past whichever of Spawn's own --timeout-ms or this function's own
// timeout arrives first; obsCtx's own Done() carries that bound, so no
// separate deadline/remaining-budget arithmetic is needed here. An observer
// error is never surfaced here -- not on the receipt, not as a returned
// error, not logged, never its own text or a path -- but the RECEIPT'S
// REASON does distinguish, in bare category terms, which of three things
// actually happened, per the plan's "理由の文言は観測した事実だけを書く"
// design decision. The check order below is the code's own priority order
// (waitOrCtxDone's failure is followed by a parent-ctx check, then a
// lastErr check):
//   - the parent ctx (Spawn's own --timeout-ms) is done at all -- whether
//     it was cancelled before this function's own budget, or the two
//     happened to run out around the same time: codexCutShortReason. This
//     is checked first, so it wins over the two reasons below whenever
//     both are true.
//   - otherwise, this function's own observation budget ran out, and the
//     LAST COMPLETED pass (never a pass that was itself cut short by that
//     same budget -- see the lastErr comment below) reported a genuine
//     error (e.g. a candidate record could not be opened):
//     codexReadErrorReason.
//   - otherwise (the observation budget ran out and no completed pass
//     reported an error): codexNotFoundReason.
func (o *Org) observeCodexSpawnReceipt(ctx context.Context, base Receipt, promptPath string, spawnStartedAt time.Time) Receipt {
	if promptPath == "" {
		return codexUnknownReceipt(base, "no role-prompt file to match a codex session record with (inline or empty initial prompt)")
	}

	sessionsDir := o.codexSessionsDir()
	interval := o.codexModelObserveInterval()

	obsCtx, cancel := context.WithTimeout(ctx, o.codexModelObserveTimeout())
	defer cancel()

	// lastErr tracks the LAST COMPLETED pass's own error (nil or genuine),
	// never a pass that was itself cut short by obsCtx: a later poll racing
	// the same expiring budget can return a ctx-done pseudo-error, and that
	// must never overwrite (shadow) an earlier pass's real read error, nor
	// can it clear one -- only another COMPLETED pass can, clean or not --
	// see this function's own doc comment, second bullet.
	var lastErr error
	for {
		// until is spawnStartedAt itself, not a later instant: the poll runs
		// moments after the spawn (AC-3's own timeout is at most a handful of
		// seconds), so there is nothing later to reach -- this keeps Spawn's
		// own window at the original three directories (see
		// ObserveCodexEffectiveModel's doc comment).
		obs, err := ObserveCodexEffectiveModel(obsCtx, sessionsDir, promptPath, spawnStartedAt, spawnStartedAt)
		switch obs.Status {
		case CodexObservationFound:
			return codexFoundReceipt(base, base.CommandedModel, obs.Model)
		case CodexObservationAmbiguous:
			return codexUnknownReceipt(base, "more than one codex session record matches this spawn; not guessing")
		}
		// A completed pass sets lastErr even when it is nil: a clean later
		// pass clears an earlier read error.
		if !isCtxDoneErr(err) {
			lastErr = err
		}

		if waitErr := waitOrCtxDone(obsCtx, interval); waitErr != nil {
			// obsCtx is done -- either the parent ctx (Spawn's own
			// --timeout-ms) is done, or this function's own observation
			// budget simply ran out. ctx's own Err() (the PARENT, not
			// obsCtx) is what tells the two apart: it is non-nil only in
			// the former case, since obsCtx propagates the parent's error
			// verbatim when the parent is the actual cause.
			if ctx.Err() != nil {
				return codexUnknownReceipt(base, codexCutShortReason)
			}
			if lastErr != nil {
				return codexUnknownReceipt(base, codexReadErrorReason)
			}
			return codexUnknownReceipt(base, codexNotFoundReason)
		}
	}
}

// codexNotFoundReason, codexReadErrorReason, and codexCutShortReason are
// observeCodexSpawnReceipt's three distinct "nothing observed" causes (see
// its own doc comment). codexNotFoundReason is also used by
// observeStopModelReceipt (Stop, verbs.go) for the identical "nothing
// found yet, no error" outcome, so the two call sites' receipts read the
// same way in `ralph org report` for that one shared case -- Stop's single
// call never distinguishes the other two causes, since it never surfaces
// an unknown reason at all (an unsuccessful Stop-time observation appends
// no receipt).
const (
	codexNotFoundReason  = "no codex session record for this spawn yet (no turn started, or CODEX_HOME differs from the seat's)"
	codexReadErrorReason = "a codex session record could not be read; effective model not observed"
	codexCutShortReason  = "observation was cut short by the spawn timeout; effective model not observed"
)

// codexFoundReceipt fills base's Honored/ReportedEffectiveModel/Reason for
// a found observation: true when effective matches commanded, false with
// both model names in Reason otherwise (AC-2). Shared by Spawn's poll and
// Stop's single observation (verbs.go appends " (observed at stop)" to
// this same Reason text for the latter).
func codexFoundReceipt(base Receipt, commanded, effective string) Receipt {
	base.ReportedEffectiveModel = effective
	if effective == commanded {
		base.Honored = HonoredTrue
		base.Reason = "codex session record reports the commanded model"
		return base
	}
	base.Honored = HonoredFalse
	base.Reason = fmt.Sprintf(
		"codex session record reports %s, not the commanded %s (a retired model that codex migrated, or a codex config override)",
		effective, commanded,
	)
	return base
}

// codexUnknownReceipt fills base's Honored=unknown and Reason for every
// non-found outcome: no prompt file, not-found, ambiguous, or ctx
// cancellation.
func codexUnknownReceipt(base Receipt, reason string) Receipt {
	base.Honored = HonoredUnknown
	base.Reason = reason
	return base
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

// codexPromptFileDetailsPrefix is the fixed prefix Spawn writes onto a
// spawn_step event's Details when the initial prompt was written to a
// role-prompt file (agentStartedDetails below): "agent_started
// prompt_file=<path>". Shared with verbs.go's codexSpawnCorrelation, which
// parses this exact prefix back out of the manifest at Stop time -- a
// single named constant instead of two independent literals keeps the
// write and read sides from silently drifting apart.
const codexPromptFileDetailsPrefix = "agent_started prompt_file="

// codexAgentStartRetriesDetailsSuffixKey is the fixed key Spawn appends,
// space-separated, onto the agent_started spawn_step's Details when
// AgentStart needed a retry (agentStartedDetails below): "agent_started
// prompt_file=<path> agent_start_retries=<N>". Shared with verbs.go's
// promptPathFromAgentStartedDetails, which strips this exact trailing
// suffix back off to recover the path at Stop time -- a single named
// constant instead of two independent literals keeps the write and strip
// sides from silently drifting apart, the same reason
// codexPromptFileDetailsPrefix exists: a retried spawn used to leave this
// suffix in place unstripped, so the recovered "path" never matched the
// pointer sentence that spawn actually wrote.
//
// This must stay the LAST field ever appended to agentStartedDetails:
// promptPathFromAgentStartedDetails strips it only when it runs to the
// very end of the string, so a field appended after it would silently
// become part of the "path" instead of being stripped, breaking the
// recovered path the same way again in a different shape. A future field
// belongs before codexPromptFileDetailsPrefix's own prompt_file= instead.
const codexAgentStartRetriesDetailsSuffixKey = "agent_start_retries="

// needsPromptFile reports whether prompt is too unsafe to pass directly as
// a herdr agent argument and must instead be written to a prompt file with
// only a one-line pointer passed inline (see promptFilePath/writePromptFile/
// PromptFilePointer below).
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

// PromptFilePointer is the single-line agent argument passed in place of
// the full prompt once it has been written to path: a short instruction
// telling the agent to read and follow that file instead.
func PromptFilePointer(path string) string {
	return "役割指示を読み込んで従ってください: " + path
}

// reject records an envelope-validation rejection: a `rejected` manifest
// event plus an honored=false receipt, per AC-1/AC-2. No spawn_started is
// written since no external side effect was ever attempted. The rejection
// receipt never carries a reported model (it is not a codex model
// observation, just a fail-closed envelope decision) -- the CLI's
// model-mismatch warning gate (Honored=="false" AND a non-empty
// ReportedEffectiveModel) is what keeps a rejection from ever being printed
// as one.
//
// The manifest-event append error is still ignored (a rejection stays a
// rejection either way -- the caller's Outcome and Err are unaffected),
// but ModelReceipt on the returned SpawnResult is set only when
// Receipts.Append actually succeeded, mirroring how Stop guards its own
// ModelReceipt (verbs.go): "the receipt this call appended" (see
// SpawnResult.ModelReceipt's doc comment) must never name a receipt that
// was never persisted.
func (o *Org) reject(p SpawnParams, cause error) SpawnResult {
	_ = o.appendEvent(ManifestEvent{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Event: EventRejected,
		Role: p.Role, Driver: p.Driver, Model: p.Model, Worktree: p.Cwd,
		DryRun: p.DryRun, Details: cause.Error(),
	})
	receipt := Receipt{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver,
		CommandedModel: p.Model, Honored: HonoredFalse, Reason: cause.Error(),
	}
	result := SpawnResult{Outcome: SpawnOutcomeRejected, Err: cause}
	if err := o.Receipts.Append(receipt); err == nil {
		result.ModelReceipt = receipt
	}
	return result
}

// dryRunSpawn simulates the full saga's manifest trail with DryRun: true on
// every event, without ever calling Herdr or Agmsg (AC-8). Dry-run events are
// excluded from the default roster/status view and from ActiveSeatCount, so
// they carry no real side effects and no [org].max_seats pressure. mode is
// the permission mode already resolved (and validated via
// permissionArgsForDriver) by the caller, recorded on the trail's final
// EventSpawned step the same way the real Spawn path records it. reserve is
// spawnCapacityErr's verdict that the real spawn would record p.Reserve: the
// trail then starts with a dry-run EventScopeReserved, as the real one does
// before spawn_started. Dry-run events never count as a reservation.
func (o *Org) dryRunSpawn(p SpawnParams, mode string, reserve bool) SpawnResult {
	team := agmsgTeam(p.OrgID)
	base := ManifestEvent{OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver, Model: p.Model, Worktree: p.Cwd, DryRun: true}

	steps := make([]ManifestEvent, 0, 8)
	if reserve {
		steps = append(steps, scopeReservedEvent("", p.OrgID, p.Reserve, "", true))
	}
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
		Task: p.Task, Envelope: EnvelopeSummary(o.Config),
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
		agentStartedDetails = codexPromptFileDetailsPrefix + promptPath
	}

	step = base
	step.Event = EventSpawnStep
	step.Details = agentStartedDetails
	steps = append(steps, step)

	// leaderSelfSpawn mirrors the same branch in Spawn (see its doc comment):
	// a dry run of `ralph org start` must simulate the same skipped
	// agmsg_leader_joined/agmsg_announced steps a real spawn would skip, so the
	// dry-run trail stays a faithful preview of what a real spawn records.
	leaderSelfSpawn := p.SeatID == LeaderIdentity

	if !leaderSelfSpawn {
		step = base
		step.Event = EventSpawnStep
		step.AgmsgTeam = team
		step.Details = "agmsg_leader_joined ok"
		steps = append(steps, step)
	}

	step = base
	step.Event = EventSpawnStep
	step.AgmsgTeam = team
	if leaderSelfSpawn {
		step.Details = "agmsg_joined leader_self=true"
	} else {
		step.Details = "agmsg_joined"
	}
	steps = append(steps, step)

	if !leaderSelfSpawn {
		step = base
		step.Event = EventSpawnStep
		step.AgmsgTeam = team
		step.Details = "agmsg_announced"
		steps = append(steps, step)
	}

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

	receipt := Receipt{
		TS: o.now(), OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver,
		CommandedModel: p.Model, Honored: HonoredUnknown, Reason: "dry-run",
	}
	if err := o.Receipts.Append(receipt); err != nil {
		return SpawnResult{Outcome: SpawnOutcomeFailed, Err: err}
	}

	return SpawnResult{Outcome: SpawnOutcomeSpawned, Seat: SeatStatus{
		OrgID: p.OrgID, SeatID: p.SeatID, Role: p.Role, Driver: p.Driver, Model: p.Model,
		Worktree: p.Cwd, AgmsgTeam: team, Event: EventSpawned, Active: false, DryRun: true,
	}, ModelReceipt: receipt}
}

// resolveWorkspace reuses the org's existing herdr workspace (recorded via
// an EventOrgWorkspaceCreated org-level event, SeatID empty, PaneID =
// workspace id) if one is still open for orgID within events -- the first
// one openOrgWorkspaces returns -- otherwise creates one and records it. A
// workspace with a later EventOrgWorkspaceClosed (Disband closed it) is
// never reused: herdr no longer has it, so a tab in it would fail. events
// is the manifest snapshot read at the top of Spawn, before this seat's own
// spawn_started was appended -- irrelevant to this lookup since org-level
// workspace events are seat-independent.
func (o *Org) resolveWorkspace(ctx context.Context, p SpawnParams, events []ManifestEvent) (string, error) {
	if open := openOrgWorkspaces(events, p.OrgID); len(open) > 0 {
		return open[0], nil
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

// openOrgWorkspaces returns the herdr workspace ids the manifest still
// records as open for orgID: the PaneID of each real (non-dry-run)
// org-level EventOrgWorkspaceCreated with no later EventOrgWorkspaceClosed
// for the same org_id and id, oldest creation first. An org normally has at
// most one; two concurrent first spawns can each create one. resolveWorkspace
// reuses the first, Disband closes them all. Events with an empty PaneID
// name no workspace and are skipped.
func openOrgWorkspaces(events []ManifestEvent, orgID string) []string {
	var open []string
	for _, ev := range events {
		if ev.OrgID != orgID || ev.SeatID != "" || ev.DryRun || ev.PaneID == "" {
			continue
		}
		if ev.Event != EventOrgWorkspaceCreated && ev.Event != EventOrgWorkspaceClosed {
			continue
		}
		open = slices.DeleteFunc(open, func(id string) bool { return id == ev.PaneID })
		if ev.Event == EventOrgWorkspaceCreated {
			open = append(open, ev.PaneID)
		}
	}
	return open
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
// (carrying the ensureLeaderJoined outcome and the agmsg_announce Leave
// compensation forward — a missing "leader" roster entry is the most likely
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

// afterStaleCompensation is a no-op by default; tests reassign it (mirroring
// absPath's "package-level var, test reassigns" seam above) to inject a
// manifest mutation in the lock-free window between compensateStale's own
// write (the line just above this call site in Spawn) and Phase 2's
// re-acquire -- simulating a concurrent racer's Spawn call for the same
// seat landing in that exact window, so Phase 2's idempotent re-check
// (self-review Cycle-2 M-2 fix) is deterministically testable without a
// real goroutine race.
var afterStaleCompensation = func() {}

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
