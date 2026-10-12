package org

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// Package note (pulse layer, PR④ Slice 3): evaluateCycle below is
// `ralph org watch`'s deterministic pulse layer for one org (see plan
// docs/plans/active/2026-08-02-org-runtime-watchdog.md, AC-3/3b/3c/4/5). It
// evaluates one cycle at a time, never invokes an LLM itself (the on-demand
// semantic-judgment watcher is a separate Slice 4 concern reached only
// through the WatchHooks.OnSemanticTrigger seam), and persists its own
// heartbeat/dedupe state to watch-status-<org_id>.json (see
// WatchStatusFileName) so a restart does not re-fire already-handled
// conditions.
//
// RunWatchAll (docs/plans/active/2026-10-11-org-cross-watch.md) runs that
// per-org cycle for every running org of a ledger in one process, after the
// inbox deadlines (deadline.go), under watch.lock and the liveness mark
// (watchalive.go). It replaced RunWatch, the loop of the single-org
// `ralph org watch --org-id`, which was retired so that no org's status file
// has two writers.

// watchdogIdentity is the agmsg identity `ralph org watch` joins/sends
// under -- distinct from LeaderIdentity and every seat id, so ALERT traffic
// is attributable to the pulse layer itself in agmsg history (see
// .claude/rules/ralph/agent-messaging.md's "watchdog is a mechanism identity, not
// a spawned seat").
const watchdogIdentity = "watchdog"

// WatchStatusFileName returns the file name (within the caller-resolved org
// state directory, typically ResolveOrgStateDir's result) RunWatchAll reads
// once and rewrites every cycle for orgID: heartbeat (last_cycle_ts/cycles)
// plus the AC-3c condition-dedupe and AC-5 deadman state. `ralph org status`
// never reads this file -- it is watch's own observability record, not seat
// roster.
//
// The name is namespaced per org_id (self-review H-1 fix): a single fixed,
// org-agnostic file name meant two orgs watched from the same repository
// (one state directory, since manifest.jsonl/model-receipts.jsonl are also
// shared there) silently clobbered each other's org-scoped fields --
// OrgID/Cycles/LastCycleTS/WatchdogJoined and SeatSnapshots (keyed
// by bare SeatID) all overwrote across orgs, while Conditions/PendingAlerts/
// Escalated happened to be safe only because conditionKey already namespaces
// them by org_id. orgID is path-safe where RunWatchAll names the file: it
// checks each org_id it reads from the manifest with ValidateIdentifier
// (identifierPattern, identifier.go: ^[a-z][a-z0-9-]{0,29}$) first
// (pulseOrg).
func WatchStatusFileName(orgID string) string {
	return fmt.Sprintf("watch-status-%s.json", orgID)
}

// EscalationsRelName is the file name (within the same state directory)
// AC-5 deadman escalations are appended to, one JSON line per escalation.
const EscalationsRelName = "escalations.jsonl"

// Pulse-layer condition type tags. Used both as the third segment of a
// dedupe conditionKey and as the ALERT message's CONDITION header value.
const (
	condStall       = "stall"
	condLiveness    = "liveness"
	condScopeChange = "scope_change"
)

// watchHerdrProbe is a consumption-side extension of HerdrClient (see
// architecture.md: "prefer interfaces at consumption sites") for the pulse
// layer's liveness/stall conditions, which need herdr `agent get` --
// something send/wait/read/stop never call. Defined here rather than added
// to HerdrClient itself in spawn.go, so that interface's existing
// implementers (real driver.Herdr, and every pre-Slice-3 test fake) are
// unaffected. driver.Herdr -- the real implementation
// internal/cli/org.go's newOrgRuntime wires into every Org.Herdr -- already
// satisfies this structurally; the pulse cycle type-asserts Org.Herdr
// against it per call and treats a failed assertion as "probe unavailable"
// (best effort, never fatal to the pulse loop).
type watchHerdrProbe interface {
	AgentGet(ctx context.Context, target string) (string, error)
}

// watchAgmsgHistory is the AgmsgClient analogue of watchHerdrProbe: only the
// deadman check's "has anything new happened in agmsg" 3rd information
// source needs History, so it stays out of AgmsgClient itself.
type watchAgmsgHistory interface {
	History(ctx context.Context, team, agentID string, limit int) (string, error)
}

// GitStatusFunc returns `git status --porcelain` output for cwd (the
// scope-change condition's declared-scope signal). Injectable so tests can
// drive scope changes deterministically without a real git worktree; the
// real one is realGitStatus.
type GitStatusFunc func(cwd string) (string, error)

// maxScopeChangeLines bounds how many `git status --porcelain` lines the
// scope-change ALERT body interpolates before summarizing the rest as
// "... N more lines" (tech-debt: "watchdog deferred LOW (2)"). A busy seat's
// worktree can have hundreds of dirty files; interpolating all of them
// whole made the eventual sendAlert message exceed
// protocol.DefaultMaxBodyChars and silently degrade to the content-free
// protocol-validation-failure fallback below, so Leader saw nothing about
// what actually changed.
const maxScopeChangeLines = 20

// scopeChangeBodyBudget is a hard byte/char cap on the (possibly
// line-truncated) porcelain text truncateScopeOutput returns, applied on top
// of maxScopeChangeLines: porcelain paths have no length limit, so 20 lines
// alone is not itself a size guarantee (e.g. 20 lines of long generated-file
// paths can still exceed protocol.DefaultMaxBodyChars once the message's own
// "TYPE/ORG_ID/SEAT/CONDITION" header and "seat %s worktree scope
// changed:" prose are added). Chosen with headroom under
// protocol.DefaultMaxBodyChars (2000) for that surrounding text -- the
// scope-change message shape in evaluateSeat is small and fixed, so 1600
// leaves well over 100 chars of margin for org_id/seat_id (each capped at 30
// chars, see identifier.go) plus the fixed prose around it.
const scopeChangeBodyBudget = 1600

// truncateScopeOutput bounds git status --porcelain output (out) before it
// is interpolated into a scope-change ALERT body: first by line count
// (maxScopeChangeLines), then -- because that alone is not a byte-size
// guarantee -- by repeatedly dropping trailing lines until the result (plus
// the "... N more lines" summary line, when anything was dropped) fits
// within byteBudget. This keeps the eventual sendAlert message reliably
// under protocol.DefaultMaxBodyChars instead of relying on sendAlert's own
// oversized-body fallback (which drops the scope text entirely) to catch
// what this function should have bounded in the first place.
func truncateScopeOutput(out string, byteBudget int) string {
	trimmed := strings.TrimRight(out, "\n")
	if trimmed == "" {
		return trimmed
	}
	lines := strings.Split(trimmed, "\n")
	total := len(lines)
	kept := lines
	if len(kept) > maxScopeChangeLines {
		kept = kept[:maxScopeChangeLines]
	}
	for {
		hidden := total - len(kept)
		body := strings.Join(kept, "\n")
		if hidden > 0 {
			body = fmt.Sprintf("%s\n… %d more lines", body, hidden)
		}
		if len(body) <= byteBudget || len(kept) == 0 {
			return body
		}
		kept = kept[:len(kept)-1]
	}
}

// EscalateFunc performs the AC-5 best-effort platform notification beyond
// the escalations.jsonl record and stderr banner (both of which
// escalateAlert always writes itself). The real implementation runs
// `osascript` on darwin and no-ops elsewhere; tests inject a stub to assert
// on the call without depending on macOS.
type EscalateFunc func(ctx context.Context, message string) error

// WatchHooks lets a caller observe watch-cycle events. Every hook is
// optional (nil-safe). OnSemanticTrigger and OnCycle only observe;
// BeforeStop also sets when the run ends, because RunWatchAll keeps
// watch.lock and a fresh stopping mark until it returns.
type WatchHooks struct {
	// OnSemanticTrigger fires when a condition warrants on-demand watcher
	// judgment (the Slice 4 seam: an on-demand `claude -p` verdict call for
	// stall/scope-change findings). No-op by default -- this pulse layer
	// never itself invokes an LLM.
	OnSemanticTrigger func(orgID, seatID, conditionType, evidence string)
	// OnCycle fires once at the end of every evaluated cycle (cycle number,
	// 1-based, and the clock value used for that cycle). Mainly test
	// observability.
	OnCycle func(cycleN int, ts time.Time)
	// BeforeStop fires once as RunWatchAll ends, after its last cycle and
	// before it writes the stopped mark and releases watch.lock, so the work
	// a caller started from OnSemanticTrigger (the on-demand watcher
	// goroutines of `ralph org watch --all`) can end while this watch still
	// holds watch.lock: a second watch is refused, and `--stop` waits for the
	// lock (cross-review cycle 1 #1,
	// docs/reports/cross-review-triage-org-cross-watch.md). RunWatchAll
	// writes the mark as stopping before it calls BeforeStop and rewrites
	// the stopping mark every interval until BeforeStop returns, so
	// ProbeWatch says stopping, and the watch gate does not take a watch
	// that runs no more cycles for one that accepts new work (plan
	// 2026-10-12-org-watch-gate, F9). BeforeStop must return within a bound:
	// while it runs, the watch keeps watch.lock and a fresh stopping mark but
	// runs no cycle, so one that never returns leaves the watch "stopping"
	// with no end. The CLI's BeforeStop waits for its watcher goroutines,
	// each bounded by watcherInvokeTimeout plus the ALERT send's
	// DriverCallTimeout.
	BeforeStop func()
}

// watchConditionRecord is the AC-3c dedupe record for one conditionKey.
// Active transitions drive the "1 alert until recovery" rule: a condition
// already Active is never re-alerted; it clears (Active: false) the first
// cycle it is no longer observed true, so a later re-occurrence re-alerts.
type watchConditionRecord struct {
	Active  bool   `json:"active"`
	FirstTS string `json:"first_ts"`
}

// watchPendingAlert is the AC-5 deadman bookkeeping recorded when an ALERT
// is sent: a snapshot of the 3 leader-activity information sources at ALERT
// time, compared against their current value each subsequent cycle. Subject
// is the seat_id the ALERT concerned; Subject == LeaderIdentity is the
// "anomaly subject is Leader itself" AC-5 branch that escalates without
// waiting for the deadman timeout.
//
// HistoryLeaderLines is a COUNT of leader-authored agmsg history lines, not the
// filtered text itself (self-review cycle-3 M3-2 fix): see
// historyLeaderLineCount's doc comment for why a count-based "did it grow"
// comparison is required instead of exact string equality.
type watchPendingAlert struct {
	AlertID            string `json:"alert_id"`
	TS                 string `json:"ts"`
	Subject            string `json:"subject"`
	ManifestLen        int    `json:"manifest_len"`
	LeaderAgentGet     string `json:"lead_agent_get"`     // tag predates the leader rename so older state decodes; renaming it would zero the value
	HistoryLeaderLines int    `json:"history_lead_lines"` // tag predates the leader rename so older state decodes; renaming it would lose the -1 sentinel
}

// watchSeatSnapshot holds the previous cycle's raw comparison values for a
// seat's stall (herdr agent get raw text) and scope-change (git status
// --porcelain) conditions. Comparing the whole raw string -- rather than
// parsing a JSON field out of it -- mirrors how AgentWait's "idle"/"done"
// check already treats herdr's informational raw text elsewhere in this
// package (see checkHerdrEnvelopeError's doc comment in
// internal/org/driver/herdr.go): a state-machine caller pattern-matches
// against it, it is never unmarshalled into a struct.
// AgentGetSeen/GitStatusSeen distinguish "no previous cycle recorded yet"
// from "the previous cycle recorded a legitimately empty string" (a clean
// `git status --porcelain` -- no local changes -- output is exactly "",
// which must not be confused with the string zero value meaning "no
// baseline exists yet").
type watchSeatSnapshot struct {
	AgentGet      string `json:"agent_get,omitempty"`
	AgentGetSeen  bool   `json:"agent_get_seen,omitempty"`
	GitStatus     string `json:"git_status,omitempty"`
	GitStatusSeen bool   `json:"git_status_seen,omitempty"`
}

// watchStatusFile is the JSON shape persisted to WatchStatusFileName(org_id).
type watchStatusFile struct {
	OrgID          string                           `json:"org_id"`
	LastCycleTS    string                           `json:"last_cycle_ts"`
	Cycles         int                              `json:"cycles"`
	WatchdogJoined bool                             `json:"watchdog_joined,omitempty"`
	Conditions     map[string]*watchConditionRecord `json:"conditions,omitempty"`
	PendingAlerts  map[string]*watchPendingAlert    `json:"pending_alerts,omitempty"`
	Escalated      map[string]bool                  `json:"escalated,omitempty"`
	SeatSnapshots  map[string]*watchSeatSnapshot    `json:"seat_snapshots,omitempty"`
}

// escalationRecord is one JSON line appended to EscalationsRelName: a
// watch escalation (AC-5) carries alert_id, and an inbox item sent to the
// human path (escalate.go) carries inbox_id instead.
type escalationRecord struct {
	TS      string `json:"ts"`
	OrgID   string `json:"org_id"`
	AlertID string `json:"alert_id,omitempty"`
	InboxID string `json:"inbox_id,omitempty"`
	Subject string `json:"subject,omitempty"`
	Reason  string `json:"reason"`
}

// conditionKey identifies one (org_id, seat_id, condition_type) dedupe slot
// (Codex advisory finding 1).
func conditionKey(orgID, seatID, condType string) string {
	return orgID + "/" + seatID + "/" + condType
}

// loadWatchStatus reads path, returning a fresh (zero-cycle) status for
// orgID if the file does not yet exist -- the first `ralph org watch`
// invocation for an org_id always starts clean.
func loadWatchStatus(path, orgID string) (*watchStatusFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &watchStatusFile{OrgID: orgID}, nil
		}
		return nil, fmt.Errorf("org: watch: read %s: %w", path, err)
	}
	var s watchStatusFile
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("org: watch: parse %s: %w", path, err)
	}
	return &s, nil
}

// save replaces path with s's current contents through a temporary file and
// a rename (writeJSONFileAtomic), as the liveness mark is written: a watch
// killed or out of disk space mid-write leaves the old file, not a part that
// the next watch cannot read (self-review L3).
func (s *watchStatusFile) save(path string) error {
	return writeJSONFileAtomic(path, s)
}

// nowTime returns o.Now() (or time.Now() when Org.Now is unset), mirroring
// (*Org).now's Clock fallback but returning time.Time instead of a
// formatted string -- the watch needs time.Time for duration arithmetic.
func (o *Org) nowTime() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

// latestSeatEventTS returns the latest TS across every manifest event of any
// type recorded for orgID/seatID, or "" if none exist. Unlike
// Roster-derived SeatStatus.TS -- which only advances on *state* events
// (stateEvents, seat.go) and so stays pinned at a seat's `spawned` TS for as
// long as it remains active with no further state transition -- this
// reflects genuine seat activity of any kind, which is what the stall
// condition's time term needs (self-review M-6 fix; see evaluateSeat's call
// site).
func latestSeatEventTS(events []ManifestEvent, orgID, seatID string) string {
	var latest string
	for _, ev := range events {
		if ev.OrgID != orgID || ev.SeatID != seatID {
			continue
		}
		if ev.TS > latest {
			latest = ev.TS
		}
	}
	return latest
}

// isStallByTime reports whether seatTS (a seat's latest manifest event
// time) is older than stallMinutes relative to now.
func isStallByTime(seatTS string, now time.Time, stallMinutes int) bool {
	if stallMinutes <= 0 || seatTS == "" {
		return false
	}
	ts, err := time.Parse(time.RFC3339, seatTS)
	if err != nil {
		return false
	}
	return now.Sub(ts) > time.Duration(stallMinutes)*time.Minute
}

// gitStatusWaitDelay is realGitStatus's exec.Cmd.WaitDelay: how long after
// the kill it waits for the output pipe, which a child of git can hold.
const gitStatusWaitDelay = time.Second

// realGitStatus is GitStatusFunc's real implementation: `git status
// --porcelain` in cwd, killed when driverCallTimeout passes like each herdr
// and agmsg call of the pulse (callWithTimeout), so a git that does not
// return cannot hold the cycle. An exit 0 whose pipe a child still held is
// a success, the way driver.ExecRunner reads it.
func (o *Org) realGitStatus(cwd string) (string, error) {
	var out []byte
	err := o.callWithTimeout(func(ctx context.Context) error {
		cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
		cmd.Dir = cwd
		cmd.WaitDelay = gitStatusWaitDelay
		var runErr error
		out, runErr = cmd.Output()
		if errors.Is(runErr, exec.ErrWaitDelay) && ctx.Err() == nil {
			runErr = nil
		}
		return runErr
	})
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// realEscalate is EscalateFunc's real implementation: a best-effort darwin
// notification via osascript. Any other GOOS -- or an osascript failure --
// is a silent no-op; escalations.jsonl and the stderr banner are always the
// authoritative escalation record (see escalateAlert's doc comment / AC-5's
// own "tests are file-output-authoritative" assumption).
func realEscalate(ctx context.Context, message string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	return osascriptNotify(ctx, "ralph org watch", message)
}

// osascriptNotify shows a macOS notification via osascript. realEscalate
// and the inbox's human path (escalate.go) share it; the error carries
// osascript's own output, which the inbox records in its notified event.
func osascriptNotify(ctx context.Context, title, message string) error {
	script := fmt.Sprintf("display notification %q with title %q", message, title)
	out, err := exec.CommandContext(ctx, "osascript", "-e", script).CombinedOutput()
	if err != nil {
		if detail := strings.TrimSpace(string(out)); detail != "" {
			return fmt.Errorf("%w: %s", err, detail)
		}
		return err
	}
	return nil
}

// ResolveWatchInterval returns the effective pulse-cycle interval
// RunWatchAll will use: requested if positive, else cfg.IntervalSeconds,
// else a fixed 30s fallback. Exported so a caller that needs to report the
// effective cadence before RunWatchAll itself resolves it internally (e.g.
// `ralph org watch`'s startup banner, self-review LOW fix: the banner used
// to print the raw --interval-seconds flag value, which is 0 by default,
// rather than the interval that is actually running) does not have to
// duplicate this fallback chain.
func ResolveWatchInterval(requested time.Duration, cfg config.OrgWatchdogConfig) time.Duration {
	interval := requested
	if interval <= 0 {
		interval = time.Duration(cfg.IntervalSeconds) * time.Second
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return interval
}

// watchRun bundles the dependencies one org's evaluateCycle and its helpers
// need, so every helper below is a method rather than a long parameter
// list. RunWatchAll builds one per run and copies it for each org with that
// org's statusPath (pulseOrg).
type watchRun struct {
	org             *Org
	cfg             config.OrgConfig
	hooks           WatchHooks
	gitStatus       GitStatusFunc
	escalateFn      EscalateFunc
	stderr          io.Writer
	statusPath      string
	escalationsPath string
}

// defaultWatchStopPollInterval is the tick of RunWatchAll's wait between
// cycles when WatchAllParams.StopPollInterval is unset: how often it looks
// at ctx and the stop request.
const defaultWatchStopPollInterval = time.Second

// WatchAllParams describes one `ralph org watch --all` run (see
// RunWatchAll).
type WatchAllParams struct {
	// StatusDir is the org state dir: watch.lock, watch-alive.json,
	// watch-stop.json, each org's watch-status-<org_id>.json, and
	// escalations.jsonl live in it (the directory of the manifest the Org
	// reads, in production). Required.
	StatusDir string
	// Interval is the pulse-cycle wait (ResolveWatchInterval): <= 0 falls
	// back to Org.Config.Watchdog.IntervalSeconds, then to 30s if that is
	// also <= 0.
	Interval time.Duration
	// Cycles, when > 0, ends the run after exactly that many cycles -- the
	// `--once` (Cycles: 1) and test path. Such a run marks itself once
	// (WatchAlive.Once), so it accepts no new work. Zero (a real
	// long-running `ralph org watch --all`) means run until ctx is done or a
	// stop request for this run is there.
	Cycles int
	// GitStatus overrides the scope-change condition's git probe; nil uses
	// the real `git status --porcelain` (realGitStatus).
	GitStatus GitStatusFunc
	// Escalate overrides the AC-5 deadman's platform notification; nil uses
	// the real darwin osascript best-effort (realEscalate, no-op elsewhere).
	Escalate EscalateFunc
	// Stderr gets the AC-5 escalation banners, the inbox deadline banners,
	// and one line for each failure RunWatchAll goes past; nil uses
	// os.Stderr.
	Stderr io.Writer
	// LockWait is how long to wait for watch.lock (AcquireWatchLock); zero
	// means DefaultWatchLockWait.
	LockWait time.Duration
	// StopPollInterval is the tick of the wait between cycles; zero means
	// defaultWatchStopPollInterval. Tests shorten it.
	StopPollInterval time.Duration
	// OnLocked, when not nil, is called once RunWatchAll holds watch.lock,
	// before it writes the first mark. A watch refused the lock does not
	// call it, so `ralph org watch --all` prints its banner there (tech-debt
	// 204 (e)).
	OnLocked func()
}

// RunWatchAll is `ralph org watch --all`: one process that watches every
// org of the ledger. It takes watch.lock and holds it until it returns
// (when another watch holds it, the error wraps ErrWatchRunning and nothing
// is written or called), calls p.OnLocked, removes a stop request that is
// not later than its start (one that is later is followed after the first
// cycle; takeStopRequestAtStart), and writes the liveness mark (running,
// its pid, its start, its deadman_minutes, and once when Cycles > 0) before
// the first cycle. Each cycle then, in this order:
//
//  1. sends the inbox items past a deadline (SweepInboxDeadlines, with one
//     InboxDeadlineMemory for the whole run). It needs no herdr, agmsg, or
//     git, so no org's pulse can hold it back;
//  2. runs the pulse cycle (evaluateCycle) of each org in RunningOrgs of
//     the manifest, in that order, with each org's watch-status file read
//     once and then kept in memory across cycles. An org that no longer
//     runs (disbanded) is not evaluated and is dropped;
//  3. rewrites the mark with the cycle's ts;
//  4. calls hooks.OnCycle.
//
// A failure in step 1 or 2 (an inbox that cannot be read or written, the
// manifest, one org's status file) is one line on Stderr and skips only its
// own part: the other orgs, the mark, and the loop go on. A status file that
// cannot be read is said once per org rather than every cycle (pulseOrg).
// A cycle does not look at ctx (evaluateCycle). Between cycles RunWatchAll
// waits Interval in ticks of StopPollInterval and ends after the cycle when
// ctx is done or a stop request later than its start is there; Cycles > 0
// ends it after that many cycles. However it ends once it holds the lock (a
// panic included), it writes the mark as stopping and calls
// hooks.BeforeStop while it still holds the lock (with no BeforeStop it
// writes no stopping mark), then writes the mark as stopped with
// stopped_at, removes a stop request meant for it, and releases the lock
// (finish). It returns an error only when it does not start (no StatusDir,
// or watch.lock not taken), and then calls no hook; every end after the
// start returns nil.
func (o *Org) RunWatchAll(ctx context.Context, p WatchAllParams, hooks WatchHooks) error {
	if strings.TrimSpace(p.StatusDir) == "" {
		return fmt.Errorf("org: watch: state dir is required")
	}
	// The start is read before the lock wait, so a stop request written
	// while this watch waits for the lock or right after it took it is
	// later than the start and is not removed below.
	startedAt := o.nowTime()
	lock, err := AcquireWatchLock(p.StatusDir, p.LockWait)
	if err != nil {
		return err
	}
	w := o.newWatchAllRun(p, hooks, startedAt)
	defer w.finish(lock)
	if p.OnLocked != nil {
		p.OnLocked()
	}
	w.takeStopRequestAtStart()
	w.writeMark()

	for n := 1; ; n++ {
		w.cycle()
		if hooks.OnCycle != nil {
			hooks.OnCycle(n, o.nowTime())
		}
		if p.Cycles > 0 && n >= p.Cycles {
			return nil
		}
		if w.waitForNextCycle(ctx) {
			return nil
		}
	}
}

// watchAllRun is one RunWatchAll: what every org's cycle shares, and what
// the run keeps between cycles.
type watchAllRun struct {
	org            *Org
	dir            string
	stderr         io.Writer
	interval, tick time.Duration
	startedAt      time.Time
	// run holds the dependencies of one org's evaluateCycle; pulseOrg
	// copies it and sets the org's statusPath.
	run         watchRun
	statuses    map[string]*watchStatusFile
	inboxMemory InboxDeadlineMemory
	// unreadableWarned holds the orgs whose status file could not be read
	// and has been said so on stderr, so the line is written once per org,
	// not every cycle (pulseOrg).
	unreadableWarned map[string]bool
	mark             WatchAlive
	// beforeStop is WatchHooks.BeforeStop (finish).
	beforeStop func()
	// stopTaken is set when a stop request later than the start was there
	// at the start (takeStopRequestAtStart), so the run follows it even if
	// it could not be put back.
	stopTaken bool
}

func (o *Org) newWatchAllRun(p WatchAllParams, hooks WatchHooks, startedAt time.Time) *watchAllRun {
	interval := ResolveWatchInterval(p.Interval, o.Config.Watchdog)
	tick := p.StopPollInterval
	if tick <= 0 {
		tick = defaultWatchStopPollInterval
	}
	gitStatus := p.GitStatus
	if gitStatus == nil {
		gitStatus = o.realGitStatus
	}
	escalateFn := p.Escalate
	if escalateFn == nil {
		escalateFn = realEscalate
	}
	stderr := p.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	deadman := o.Config.DeadmanMinutes
	return &watchAllRun{
		org: o, dir: p.StatusDir, stderr: stderr,
		interval: interval, tick: tick, startedAt: startedAt,
		run: watchRun{
			org: o, cfg: o.Config, hooks: hooks,
			gitStatus: gitStatus, escalateFn: escalateFn, stderr: stderr,
			escalationsPath: EscalationsPathIn(p.StatusDir),
		},
		statuses:         map[string]*watchStatusFile{},
		unreadableWarned: map[string]bool{},
		mark: WatchAlive{
			State: WatchAliveRunning, PID: os.Getpid(),
			StartedAt:       startedAt.UTC().Format(time.RFC3339Nano),
			IntervalSeconds: int(interval / time.Second),
			DeadmanMinutes:  &deadman,
			Once:            p.Cycles > 0,
		},
		beforeStop: hooks.BeforeStop,
	}
}

// cycle runs steps 1 to 3 of RunWatchAll's cycle.
func (w *watchAllRun) cycle() {
	sweep := w.org.SweepInboxDeadlines(w.org.nowTime(), &w.inboxMemory, w.stderr)
	if sweep.Err != nil {
		_, _ = fmt.Fprintf(w.stderr, "watchdog: inbox deadlines skipped this cycle: %v\n", sweep.Err)
	}
	for _, f := range sweep.Failed {
		_, _ = fmt.Fprintf(w.stderr, "watchdog: inbox item %s not sent for %s: %v\n", f.ID, f.Reason, f.Err)
	}
	w.pulse()
	w.writeMark()
}

// pulse runs the pulse cycle of every running org.
func (w *watchAllRun) pulse() {
	rr, err := w.org.Manifest.Read()
	if err != nil {
		_, _ = fmt.Fprintf(w.stderr, "watchdog: pulse skipped this cycle: read manifest: %v\n", err)
		return
	}
	running := RunningOrgs(rr.Events)
	for orgID := range w.statuses {
		if !slices.Contains(running, orgID) {
			delete(w.statuses, orgID)
		}
	}
	for _, orgID := range running {
		if err := w.pulseOrg(orgID); err != nil {
			_, _ = fmt.Fprintf(w.stderr, "watchdog: org %q skipped this cycle: %v\n", orgID, err)
		}
	}
}

// pulseOrg runs orgID's pulse cycle, loading its status file the first time.
// orgID comes from the manifest, so it is checked before it names a file: a
// hand-edited line could hold a path. A status file that cannot be read
// skips the org this cycle, and the next cycle tries to load it again; that
// is said on stderr once per org (unreadableWarned), not every cycle, and is
// not returned as the error pulse writes every cycle.
func (w *watchAllRun) pulseOrg(orgID string) error {
	if err := ValidateIdentifier("org_id", orgID); err != nil {
		return err
	}
	run := w.run
	run.statusPath = filepath.Join(w.dir, WatchStatusFileName(orgID))
	status := w.statuses[orgID]
	if status == nil {
		loaded, err := loadWatchStatus(run.statusPath, orgID)
		if err != nil {
			if !w.unreadableWarned[orgID] {
				w.unreadableWarned[orgID] = true
				_, _ = fmt.Fprintf(w.stderr, "watchdog: org %q is skipped every cycle until its watch-status file can be read (said once): %v\n", orgID, err)
			}
			return nil
		}
		status = loaded
		w.statuses[orgID] = status
	}
	return run.evaluateCycle(orgID, status)
}

// writeMark rewrites the mark with ts now.
func (w *watchAllRun) writeMark() {
	w.mark.TS = w.org.nowTime().UTC().Format(time.RFC3339)
	if err := writeWatchAlive(w.dir, w.mark); err != nil {
		_, _ = fmt.Fprintf(w.stderr, "watchdog: liveness mark not written: %v\n", err)
	}
}

// stopRequested reports whether a stop request for this run is there, or
// was there at the start (stopTaken).
func (w *watchAllRun) stopRequested() bool {
	if w.stopTaken {
		return true
	}
	req, err := ReadWatchStopRequest(w.dir)
	return err == nil && req.AppliesTo(w.startedAt)
}

// afterStopRequestTaken is a no-op by default; tests reassign it (the
// package-level seam of afterStaleCompensation in spawn.go) to write a new
// stop request between takeStopRequestAtStart's take and what it does with
// the request it took, which is the window a concurrent `--stop` can write
// in.
var afterStopRequestTaken = func() {}

// takeStopRequestAtStart deals with a stop request that is there at the
// start. It takes the request with one rename (takeWatchStopRequest)
// rather than reading and then removing watch-stop.json, which deleted a
// request a concurrent `--stop` wrote between the two (cross-review cycle 1
// #2, docs/reports/cross-review-triage-org-cross-watch.md): such a request
// is a new file this leaves alone. A request later than the start is put
// back and remembered (stopTaken), so the run follows it after the first
// cycle even when the put-back fails; a put-back over a request written
// meanwhile replaces one request for this run with another. A request that
// is not later than the start (left for an earlier watch), or that cannot
// be read, is removed.
func (w *watchAllRun) takeStopRequestAtStart() {
	req, taken, err := takeWatchStopRequest(w.dir)
	if taken == "" {
		if !errors.Is(err, fs.ErrNotExist) {
			_, _ = fmt.Fprintf(w.stderr, "watchdog: %v\n", err)
		}
		return
	}
	afterStopRequestTaken()
	if err == nil && req.AppliesTo(w.startedAt) {
		w.stopTaken = true
		putBackErr := os.Rename(taken, WatchStopPathIn(w.dir))
		if putBackErr == nil {
			return
		}
		_, _ = fmt.Fprintf(w.stderr, "watchdog: put the stop request back: %v (this watch still stops after its first cycle)\n", putBackErr)
	}
	if err := os.Remove(taken); err != nil {
		_, _ = fmt.Fprintf(w.stderr, "watchdog: remove the stop request taken at the start: %v\n", err)
	}
}

// waitForNextCycle waits the interval in ticks and reports whether the run
// is to end: ctx is done or a stop request for this run is there. It looks
// before the first tick too, so a request written during a cycle ends the
// run right after that cycle. The wait reads the real clock, not Org.Now,
// which tests freeze.
func (w *watchAllRun) waitForNextCycle(ctx context.Context) bool {
	deadline := time.Now().Add(w.interval)
	for {
		if ctx.Err() != nil || w.stopRequested() {
			return true
		}
		left := time.Until(deadline)
		if left <= 0 {
			return false
		}
		select {
		case <-ctx.Done():
			return true
		case <-time.After(min(w.tick, left)):
		}
	}
}

// finish ends the run once it holds the lock: drainBeforeStop, then
// stopAndRelease, which is deferred so that it runs when beforeStop panics
// too.
func (w *watchAllRun) finish(lock *WatchLock) {
	defer w.stopAndRelease(lock)
	w.drainBeforeStop()
}

// drainBeforeStop calls beforeStop, if set, while the lock is held. It
// first writes the mark as stopping, and until beforeStop returns, a
// goroutine rewrites the stopping mark every interval (writeMark), so a
// drain longer than the window of ProbeWatch reads as stopping, not stale;
// drainBeforeStop waits for that goroutine to end before it returns, so its
// last rewrite comes before the stopped mark.
func (w *watchAllRun) drainBeforeStop() {
	if w.beforeStop == nil {
		return
	}
	w.mark.State = WatchAliveStopping
	w.writeMark()
	done := make(chan struct{})
	refresherDone := make(chan struct{})
	go func() {
		defer close(refresherDone)
		for {
			select {
			case <-done:
				return
			case <-time.After(w.interval):
				w.writeMark()
			}
		}
	}()
	defer func() {
		close(done)
		<-refresherDone
	}()
	w.beforeStop()
}

// stopAndRelease writes the stopped mark, removes a stop request meant for
// this run (it is being followed now, whatever ended the run), and releases
// the lock, in that order: a `--stop` that sees the lock free also sees the
// stopped mark. A request a `--stop` writes between stopRequested's read
// and the remove is one for this run (the lock is still held), and that
// `--stop` waits for the lock, which is released next, so removing it loses
// nothing.
func (w *watchAllRun) stopAndRelease(lock *WatchLock) {
	now := w.org.nowTime().UTC().Format(time.RFC3339)
	w.mark.State, w.mark.StoppedAt = WatchAliveStopped, now
	if err := writeWatchAlive(w.dir, w.mark); err != nil {
		_, _ = fmt.Fprintf(w.stderr, "watchdog: stopped mark not written: %v\n", err)
	}
	if w.stopRequested() {
		if err := RemoveWatchStopRequest(w.dir); err != nil {
			_, _ = fmt.Fprintf(w.stderr, "watchdog: %v\n", err)
		}
	}
	if err := lock.Release(); err != nil {
		_, _ = fmt.Fprintf(w.stderr, "watchdog: release %s: %v\n", WatchLockFileName, err)
	}
}

// evaluateCycle runs exactly one pulse-layer cycle: per-active-seat checks
// (stall / liveness / scope-change), then the AC-5 deadman sweep over any
// still-pending alerts, then persists status.
//
// Each herdr and agmsg call of the cycle, and the real git status, runs
// under its own Org.callWithTimeout deadline, and the deadman's EscalateFunc
// under its own inboxDesktopNotifyTimeout (escalateAlert), each cut from
// context.Background(), not from the watch loop's context (Codex plan
// advisory finding 1 of docs/plans/active/2026-10-11-org-cross-watch.md;
// self-review L2): a call that does not answer fails like any other failure
// of that call, so one org cannot hold the cycle of the others, and a watch
// told to stop still finishes its cycle with real answers rather than with
// cancelled calls that would read as dead seats, or with an escalation
// recorded whose desktop notification never showed. So the cycle takes no
// context.
func (w *watchRun) evaluateCycle(orgID string, status *watchStatusFile) error {
	now := w.org.nowTime()

	rr, err := w.org.Manifest.Read()
	if err != nil {
		return fmt.Errorf("org: watch: read manifest: %w", err)
	}

	if status.Conditions == nil {
		status.Conditions = map[string]*watchConditionRecord{}
	}
	if status.PendingAlerts == nil {
		status.PendingAlerts = map[string]*watchPendingAlert{}
	}
	if status.Escalated == nil {
		status.Escalated = map[string]bool{}
	}
	if status.SeatSnapshots == nil {
		status.SeatSnapshots = map[string]*watchSeatSnapshot{}
	}
	status.OrgID = orgID

	// Prune any retired-condition entries (retiredConditionNames) a status
	// file written before this package's now-removed per-seat/org
	// wall-clock cutoff feature was removed may still carry -- run before
	// anything else touches Conditions/PendingAlerts/Escalated (in
	// particular, before checkDeadman below) so a stale pending alert for a
	// condition type that no longer exists can never reach the deadman
	// sweep and escalate.
	pruneRetiredConditions(status)

	var activeSeats []SeatStatus
	for _, s := range Roster(rr.Events, RosterOptions{}) {
		if s.OrgID == orgID && s.Active {
			activeSeats = append(activeSeats, s)
		}
	}

	for _, s := range activeSeats {
		w.evaluateSeat(status, orgID, s, now, rr.Events)
	}

	rr2, err := w.org.Manifest.Read()
	if err != nil {
		return fmt.Errorf("org: watch: re-read manifest: %w", err)
	}
	w.checkDeadman(status, rr2, now)

	status.LastCycleTS = now.UTC().Format(time.RFC3339)
	status.Cycles++
	if err := status.save(w.statusPath); err != nil {
		return err
	}
	return nil
}

// retiredConditionNames holds the condition-type segments (their exact
// string values are the map keys below) that a watch-status file written
// before this package's now-removed per-seat/org wall-clock cutoff feature
// was removed may still carry. Kept here, unexported, purely so
// pruneRetiredConditions can grep-ably drop them -- these are not live
// condition types (see the condStall/condLiveness/condScopeChange enum
// above) and nothing in this package ever writes them again.
var retiredConditionNames = map[string]bool{"seat_budget": true, "total_budget": true}

// pruneRetiredConditions drops any Conditions/PendingAlerts/Escalated entry
// belonging to a retiredConditionNames condition type from a persisted
// watch-status file, so a status file written before this package's
// now-removed per-seat/org wall-clock cutoff feature was removed can never
// carry a stale pending alert into checkDeadman's deadman sweep (which
// would otherwise, given enough elapsed time, escalate a condition that no
// longer exists). conditionKey's shape is
// "org_id/seat_id/condition_type" (see that function), so the condition
// type is the key's last "/"-delimited segment; a pending alert's condition
// type is recovered the same way from its AlertID, whose shape is
// "<conditionKey>@<unixnano>" (sendAlert's format) -- everything between the
// last "/" and the "@" is the condition type.
func pruneRetiredConditions(status *watchStatusFile) {
	for key := range status.Conditions {
		if idx := strings.LastIndex(key, "/"); idx >= 0 && retiredConditionNames[key[idx+1:]] {
			delete(status.Conditions, key)
		}
	}
	for alertID := range status.PendingAlerts {
		if retiredConditionNames[retiredConditionTypeFromAlertID(alertID)] {
			delete(status.PendingAlerts, alertID)
		}
	}
	for alertID := range status.Escalated {
		if retiredConditionNames[retiredConditionTypeFromAlertID(alertID)] {
			delete(status.Escalated, alertID)
		}
	}
}

// retiredConditionTypeFromAlertID recovers the condition-type segment from
// an AlertID shaped "<org_id>/<seat_id>/<condition_type>@<unixnano>"
// (sendAlert's "%s@%d" format over conditionKey) -- the segment between the
// last "/" and the "@". Returns "" (never a retiredConditionNames match) if
// the id does not have that shape, which is the safe default: an
// unparseable id is left alone rather than guessed at.
func retiredConditionTypeFromAlertID(alertID string) string {
	at := strings.LastIndex(alertID, "@")
	if at < 0 {
		return ""
	}
	head := alertID[:at]
	slash := strings.LastIndex(head, "/")
	if slash < 0 {
		return ""
	}
	return head[slash+1:]
}

// evaluateSeat runs every per-seat pulse condition for s: (c) heartbeat
// stall, (d) process liveness, (e) worktree scope change. events is the
// cycle's manifest snapshot (see evaluateCycle), needed by the stall
// condition's M-6 fix below.
func (w *watchRun) evaluateSeat(status *watchStatusFile, orgID string, s SeatStatus, now time.Time, events []ManifestEvent) {
	snap := status.SeatSnapshots[s.SeatID]
	if snap == nil {
		snap = &watchSeatSnapshot{}
		status.SeatSnapshots[s.SeatID] = snap
	}

	var agentGetOut string
	var agentGetErr error
	if probe, ok := w.org.Herdr.(watchHerdrProbe); ok {
		agentGetErr = w.org.callWithTimeout(func(ctx context.Context) error {
			var err error
			agentGetOut, err = probe.AgentGet(ctx, resolvedHerdrAgentName(s))
			return err
		})
	}

	if agentGetErr != nil {
		// (d) process liveness: herdr agent get failed for an active seat
		// (a call cut off by the deadline included).
		w.raiseOrClear(status, orgID, s.SeatID, condLiveness, true, now,
			fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: %s\n\nherdr agent get failed for seat %s: %v",
				orgID, s.SeatID, condLiveness, s.SeatID, agentGetErr), false)
	} else {
		w.raiseOrClear(status, orgID, s.SeatID, condLiveness, false, now, "", false)

		// (c) heartbeat stall: last manifest event time AND herdr raw probe
		// text both unchanged since the previous cycle. lastEventTS (self-
		// review M-6 fix) is the seat's latest event of ANY type, not s.TS --
		// Roster's SeatStatus.TS only advances on *state* events
		// (stateEvents, seat.go, deliberately excludes e.g. `sent`), so for a
		// healthy active seat s.TS stays frozen at its `spawned` TS and
		// isStallByTime(s.TS, ...) would be permanently true for any seat
		// older than stall_minutes -- the only real discriminator left would
		// be the single-interval herdr raw-text comparison above, contrary to
		// [org.watchdog].stall_minutes' documented "how long ... may both
		// stay unchanged" semantics (config.go).
		lastEventTS := latestSeatEventTS(events, orgID, s.SeatID)
		stalled := snap.AgentGetSeen && snap.AgentGet == agentGetOut && isStallByTime(lastEventTS, now, w.cfg.Watchdog.StallMinutes)
		w.raiseOrClear(status, orgID, s.SeatID, condStall, stalled, now,
			fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: %s\n\nseat %s heartbeat stalled for over %dm",
				orgID, s.SeatID, condStall, s.SeatID, w.cfg.Watchdog.StallMinutes), true)
		snap.AgentGet = agentGetOut
		snap.AgentGetSeen = true
	}

	// (e) scope change: seat worktree `git status --porcelain` differs from
	// the previous cycle's snapshot. No cutoff -- ALERT + semantic trigger
	// only (free-text scope, AC-4/plan Non-goals).
	if s.Worktree != "" {
		out, err := w.gitStatus(s.Worktree)
		if err == nil {
			changed := snap.GitStatusSeen && snap.GitStatus != out
			w.raiseOrClear(status, orgID, s.SeatID, condScopeChange, changed, now,
				fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: %s\n\nseat %s worktree scope changed:\n%s",
					orgID, s.SeatID, condScopeChange, s.SeatID, truncateScopeOutput(out, scopeChangeBodyBudget)), true)
			snap.GitStatus = out
			snap.GitStatusSeen = true
		}
	}
}

// raiseOrClear implements the AC-3c idempotent ALERT dedupe for a
// condition: an active==false->true transition sends exactly one
// ALERT and records the key as Active; the key clears (Active: false) the
// first cycle active is observed false again, so a later re-occurrence
// re-alerts. semantic, when true and the condition is newly active, also
// fires WatchHooks.OnSemanticTrigger (the Slice 4 on-demand-watcher seam) --
// liveness intentionally passes semantic=false: a dead/unreachable pane
// gives a watcher nothing useful to judge.
func (w *watchRun) raiseOrClear(status *watchStatusFile, orgID, seatID, condType string, active bool, now time.Time, message string, semantic bool) {
	key := conditionKey(orgID, seatID, condType)
	rec := status.Conditions[key]
	if active {
		if rec != nil && rec.Active {
			return // already alerted and still active: dedupe
		}
		status.Conditions[key] = &watchConditionRecord{Active: true, FirstTS: now.UTC().Format(time.RFC3339)}
		w.sendAlert(status, orgID, seatID, condType, message, now)
		if semantic && w.hooks.OnSemanticTrigger != nil {
			w.hooks.OnSemanticTrigger(orgID, seatID, condType, message)
		}
		return
	}
	if rec != nil && rec.Active {
		rec.Active = false // recovered: clears, a future re-occurrence re-alerts
	}
}

// SendWatchdogAlert sends message from the watchdogIdentity mechanism
// identity to LeaderIdentity over orgID's agmsg team, using Agmsg.Send
// directly rather than the seat-steering Send verb (verbs.go). Send resolves
// its To target as a spawned SEAT via findSeat, which fails -- silently
// dropping the message -- in the normal "session-promoted leader" org shape
// where no leader SEAT was ever spawned (only the leader identity itself,
// registered via ensureLeaderJoined/ensureWatchdogJoined's Join calls). Live
// smoke (docs/plans/active/2026-08-02-org-runtime-watchdog.md) found zero
// ALERTs reaching agmsg history under exactly that shape while escalations
// still fired. Both this package's own pulse-layer sendAlert and
// internal/cli/org_watch.go's on-demand watcher-verdict ALERT path
// (newWatchdogHooks) call this so ALERT delivery is identical regardless of
// which layer produced the finding.
func (o *Org) SendWatchdogAlert(ctx context.Context, orgID, message string) error {
	return o.Agmsg.Send(ctx, agmsgTeam(orgID), watchdogIdentity, LeaderIdentity, message)
}

// SendWatchdogAlertBounded is SendWatchdogAlert cut off at the deadline
// each driver call of the pulse gets (Org.DriverCallTimeout, default 10s),
// or when ctx is done, whichever comes first. newWatchdogHooks sends the
// on-demand watcher's ALERT through it, so an agmsg that does not answer
// cannot keep a stopped watch's process alive (self-review L1).
func (o *Org) SendWatchdogAlertBounded(ctx context.Context, orgID, message string) error {
	ctx, cancel := context.WithTimeout(ctx, o.driverCallTimeout())
	defer cancel()
	return o.SendWatchdogAlert(ctx, orgID, message)
}

// ensureWatchdogJoined best-effort-joins the "watchdog" mechanism identity
// (see the const's doc comment) onto the org's agmsg team exactly once per
// org's persisted watch status -- mirrors ensureLeaderJoined's idempotent,
// best-effort Join semantics in spawn.go, but for the watchdog identity
// instead of leader.
//
// WatchdogJoined is set only after Join actually succeeds (PR④ known gap
// #6, docs/reports/cross-review-triage-org-runtime-watchdog.md Cycle 3 #6):
// setting it unconditionally would persist a false "already joined" record
// across a transient Join failure (e.g. agmsg momentarily unreachable),
// permanently skipping every future retry for that org -- and, since
// SendWatchdogAlert has no membership dependency of its own that would
// otherwise surface the gap, ALERT delivery would keep failing silently
// forever after just one bad cycle. Leaving the flag false on error lets
// the very next cycle retry Join. A Join cut off by its callWithTimeout
// deadline is such a failure.
func (w *watchRun) ensureWatchdogJoined(status *watchStatusFile, orgID string) {
	if status.WatchdogJoined {
		return
	}
	cwd, _ := os.Getwd()
	if err := w.org.callWithTimeout(func(ctx context.Context) error {
		return w.org.Agmsg.Join(ctx, agmsgTeam(orgID), watchdogIdentity, "claude-code", cwd)
	}); err == nil {
		status.WatchdogJoined = true
	}
}

// sendAlert validates and sends one ALERT to leader via the watchdog identity
// (SendWatchdogAlert, not the seat-steering Send verb -- see that method's
// doc comment for why) best-effort -- a Send failure, e.g. agmsg itself is
// unreachable or the Send is cut off by its callWithTimeout deadline, never
// aborts the pulse cycle but is logged to w.stderr -- then registers an AC-5
// pending-alert deadman record regardless of whether Send itself succeeded:
// the whole point of the deadman clause is to catch the case where leader
// cannot be reached at all.
func (w *watchRun) sendAlert(status *watchStatusFile, orgID, seatID, condType, message string, now time.Time) {
	if err := protocol.ValidateText(message, protocol.DefaultMaxBodyChars); err != nil {
		// SEAT is always included here (tech-debt: "watchdog deferred LOW
		// (2)"): before this fix, a busy seat whose original message failed
		// validation degraded to a fallback with no subject at all, so Leader
		// saw only "message failed protocol validation" with no way to tell
		// which seat the finding was about.
		message = fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: %s\n\nwatchdog: message failed protocol validation: %v",
			orgID, seatID, condType, err)
	}
	w.ensureWatchdogJoined(status, orgID)
	if err := w.org.callWithTimeout(func(ctx context.Context) error {
		return w.org.SendWatchdogAlert(ctx, orgID, message)
	}); err != nil {
		_, _ = fmt.Fprintf(w.stderr, "watchdog: failed to ALERT leader for org %q condition %q: %v\n", orgID, condType, err)
	}

	rr, _ := w.org.Manifest.Read()
	alertID := fmt.Sprintf("%s@%d", conditionKey(orgID, seatID, condType), now.UnixNano())
	status.PendingAlerts[alertID] = &watchPendingAlert{
		AlertID:            alertID,
		TS:                 now.UTC().Format(time.RFC3339),
		Subject:            seatID,
		ManifestLen:        leaderActivityEventCount(rr.Events, orgID),
		LeaderAgentGet:     w.leaderProbeSnapshot(orgID),
		HistoryLeaderLines: w.historyLeaderLineCount(orgID),
	}
}

// leaderActivityEventCount counts manifest events attributable to leader for
// orgID (self-review M-4 fix, org-scoped per cross-review AR-1; the seat-
// attribution model itself was corrected by self-review cycle-3 H3-1 --
// see below): a genuinely unresponsive leader must not have its deadman
// escalation silently cleared by an unrelated seat's own manifest traffic,
// so an event only counts here when either (a) or (b) holds:
//
//	(a) ev.Event == EventSent. A `sent` event is leader-authored BY
//	    CONSTRUCTION, regardless of ev.SeatID: ev.SeatID on a `sent` event
//	    is the *recipient* (Send writes SeatID: p.To -- see verbs.go), not
//	    the author, and `ralph org send` is the only verb that ever appends
//	    one. In the star topology (.claude/rules/ralph/agent-messaging.md), only
//	    leader/the operator drives that verb -- a seat's reply travels over
//	    the agmsg skill, which never touches this manifest at all -- so
//	    every `sent` event in orgID's manifest was written by leader sending
//	    to someone, never by a seat sending to leader. (Cycle-2's fix used
//	    `ev.SeatID == LeaderIdentity` here, which is backwards: it excluded
//	    the star topology's mandated seat->leader `sent` traffic while
//	    treating leader->seat sends as nothing. See
//	    TestWatch_Deadman_SeatSentEvent_ClearsPendingAlert_LegacyWatchdogStopDoesNot.)
//	(b) it is a non-watchdog event from the leader-driven lifecycle set
//	    (spawned, spawn_started, stopped, stop_failed, disbanded, rejected)
//	    that is not the watchdog's own enforcement write. Each of these is
//	    only producible by a `ralph org` verb that leader/the operator runs
//	    (spawn/stop/disband), so it is evidence leader is alive and acting,
//	    even when the event itself names a seat, not leader (e.g. leader
//	    spawning a replacement seat in response to a stall ALERT, self-review
//	    cycle-3 M3-1). `stop_failed` is what `ralph org stop` and `ralph org
//	    disband` (and their `--all` forms) write instead of `stopped` when
//	    herdr did not close the seat's pane (PR #208). No watchdog code
//	    writes it, so a leader that keeps retrying a stop while herdr does
//	    not answer still counts as acting, the same as a stop that closed
//	    the pane; a stop the operator runs counts too, as `stopped` always
//	    has. The exclusion applies to any event in this lifecycle set whose
//	    Details carry "reason=watchdog_..." -- in practice only the pre-#152
//	    watchdog's cutoff `stopped` writes ever carried it.
//	    No code path has produced such an event since PR #152 removed the
//	    org budget concept (2026-09-17): StopParams no longer has a Reason
//	    field and no pulse-layer condition calls Stop. Why the guard is
//	    kept: this function's two production call sites are exactly a
//	    baseline/recount pair -- sendAlert records the ManifestLen snapshot
//	    and checkDeadman recounts against it -- so the count is only ever
//	    used as a *difference*. A legacy cutoff that predates the alert
//	    lands in both the baseline and every recount and cancels out; the
//	    guard is load-bearing only when a cutoff is missing from the
//	    baseline but present in the recount: (a) a pending alert persisted
//	    by a pre-#152 build (its ManifestLen excluded the cutoff, and a full
//	    recount without the guard would read one higher for an unchanged
//	    manifest and silently clear the alert -- see
//	    TestWatch_Deadman_PersistedAlertBaseline_SurvivesLegacyWatchdogStop),
//	    or (b) a mixed-version window where an old `ralph org watch`
//	    appends a cutoff after a new binary recorded the baseline. Keeping
//	    the exclusion means such a cutoff never counts, so neither case can
//	    misfire. The exclusion also covers `stop_failed`, only because it
//	    shares the case below; no `stop_failed` has ever carried
//	    "reason=watchdog_". Nor does `stop_failed` need a compatibility
//	    guard of its own: ralph up to v5.1.0 never writes it, so a baseline
//	    persisted by such a watch has no `stop_failed` to miss, with one
//	    exception. If a v5.1.0 `ralph org watch` is still running when a
//	    newer `ralph org stop` writes a `stop_failed`, the alert it saves
//	    has a baseline without that event. Restarting watch on the newer
//	    binary while that alert is pending then reads one higher and clears
//	    it.
//
// The orgID filter excludes another org's activity in the same shared
// manifest: without it, a new event in a different, active org would clear
// a stalled org's pending deadman alert even though nothing happened in the
// stalled org itself.
func leaderActivityEventCount(events []ManifestEvent, orgID string) int {
	n := 0
	for _, ev := range events {
		if ev.OrgID != orgID {
			continue
		}
		if ev.Event == EventSent {
			n++
			continue
		}
		switch ev.Event {
		case EventSpawned, EventSpawnStarted, EventStopped, EventStopFailed, EventDisbanded, EventRejected:
			if !strings.Contains(ev.Details, "reason=watchdog_") {
				n++
			}
		}
	}
	return n
}

// leaderProbeSnapshot returns the leader seat's current herdr `agent get` raw
// text, or "" if the probe is unavailable/errors, a call cut off by its
// callWithTimeout deadline included (best-effort deadman information
// source #2).
func (w *watchRun) leaderProbeSnapshot(orgID string) string {
	probe, ok := w.org.Herdr.(watchHerdrProbe)
	if !ok {
		return ""
	}
	var out string
	err := w.org.callWithTimeout(func(ctx context.Context) error {
		var err error
		out, err = probe.AgentGet(ctx, herdrAgentName(orgID, LeaderIdentity))
		return err
	})
	if err != nil {
		return ""
	}
	return out
}

// historyLeaderLineCount returns the COUNT of the org's agmsg team history
// lines that are LEADER-authored (leaderHistoryLines), or -1 if the probe is
// unavailable/errors, a call cut off by its callWithTimeout deadline
// included (best-effort deadman information source #3; -1, not 0, so an
// unavailable probe is distinguishable from "leader has genuinely never
// sent anything yet").
//
// Counting rather than comparing the filtered text (self-review cycle-3
// M3-2 fix) matters because the window this reads is NOT scoped to leader's
// own traffic: agentID is passed as "" below, which -- per the driver's own
// doc comment on History -- makes it drop its LIMIT argument entirely and
// fall through to the agmsg skill's history.sh script's own default
// (LIMIT=20, see ~/.agents/skills/agmsg/scripts/history.sh -- confirmed by
// reading that script directly; it is a user-global skill install, not
// vendored into this repo, so `.agents/skills/` here has no `agmsg/`
// subdirectory to find it in), applied to the WHOLE team's traffic, not
// just leader's. As other seats chat, older leader lines get evicted from that
// last-20-of-everyone window and the filtered text can shrink even though
// leader did nothing new -- comparing exact strings would read a pure
// eviction as "activity" and wrongly clear a pending alert (the bug this
// fix closes). A count comparison that only treats growth (cur > baseline,
// see checkDeadman) as activity does not have that failure mode: eviction
// can only ever decrease the count, never manufacture an increase. Passing
// LeaderIdentity as agentID to scope the query itself would be a more
// complete fix (and would make the `20` argument here non-dead) but is
// deferred -- the count comparison alone is sufficient to close the false-
// activity bug the cycle-3 finding described.
func (w *watchRun) historyLeaderLineCount(orgID string) int {
	probe, ok := w.org.Agmsg.(watchAgmsgHistory)
	if !ok {
		return -1
	}
	var out string
	err := w.org.callWithTimeout(func(ctx context.Context) error {
		var err error
		out, err = probe.History(ctx, agmsgTeam(orgID), "", 20)
		return err
	})
	if err != nil {
		return -1
	}
	return len(leaderHistoryLines(out))
}

// leaderHistoryFromField parses one agmsg history line -- the real shape is
// "  <status> [<ts>] <from> → <to>: <body>" (see the agmsg skill's
// `scripts/history.sh`, a user-global install under
// `~/.agents/skills/agmsg/`, not vendored in this repo -- its `echo "
// $status [$ts] $from → $to: $body"` line) -- and returns its from field.
// ok is false whenever the line does not contain both the "] " and " → "
// markers this parse depends on; a caller must then exclude the line
// entirely rather than guess, since an unparseable line could just as
// easily be leader- as non-leader-authored (defensive: exclude on parse
// failure, per cross-review-triage cycle-2 #3's "parse defensively"
// instruction).
//
// Anchoring on the FIRST "] " in the line is safe because history.sh's own
// SQL formats the body with
// `replace(replace(body, char(10), '\n'), char(9), '\t')` before emitting
// it -- every literal newline/tab inside a message body is escaped to a
// two-character sequence before the line is printed, so each history
// record is always exactly one physical line and status+timestamp always
// come first; a body containing a literal "] " cannot introduce a second
// line break for this parse to trip over.
func leaderHistoryFromField(line string) (string, bool) {
	_, rest, found := strings.Cut(line, "] ")
	if !found {
		return "", false
	}
	fromPart, _, found := strings.Cut(rest, " → ")
	if !found {
		return "", false
	}
	from := strings.TrimSpace(fromPart)
	if from == "" {
		return "", false
	}
	return from, true
}

// leaderHistoryLines returns only raw's lines whose parsed from field
// (leaderHistoryFromField) is exactly LeaderIdentity. Lines that fail to parse
// are excluded, not conservatively kept -- see leaderHistoryFromField's doc
// comment. Shared by filterLeaderHistoryLines (text, used by tests to pin the
// parsing contract directly) and historyLeaderLineCount (count, used by the
// production deadman check) so the two never drift on what counts as a
// leader line.
func leaderHistoryLines(raw string) []string {
	var kept []string
	for line := range strings.SplitSeq(raw, "\n") {
		if from, ok := leaderHistoryFromField(line); ok && from == LeaderIdentity {
			kept = append(kept, line)
		}
	}
	return kept
}

// filterLeaderHistoryLines returns leaderHistoryLines(raw) joined back with
// "\n" -- kept as a thin wrapper so TestFilterLeaderHistoryLines can keep
// pinning the parsing contract as a single string comparison.
func filterLeaderHistoryLines(raw string) string {
	return strings.Join(leaderHistoryLines(raw), "\n")
}

// checkDeadman implements AC-5: for every still-pending ALERT, look for
// leader activity (any of the 3 information sources changed since the ALERT
// was sent) and either clear the pending record (activity found, and the
// anomaly subject is not leader itself) or escalate (deadman_minutes elapsed
// with no activity, OR the anomaly subject is leader itself -- escalates
// without waiting for the timeout, since leader cannot be expected to
// self-report while it is the thing that is anomalous).
//
// The probe-based sources (#2 leaderProbeSnapshot, #3 historyLeaderLineCount)
// only count as activity when the ALERT-time baseline itself was a valid,
// comparable snapshot -- not the "probe was unavailable" sentinel
// (LeaderAgentGet == "" / HistoryLeaderLines == -1, per leaderProbeSnapshot's and
// historyLeaderLineCount's own doc comments, the producers of these values).
// Without that guard, an alert recorded while a probe was down
// (baseline collapses to the sentinel) would false-clear the moment the
// probe merely recovers on a later cycle: cur != "" is trivially true
// against a "" baseline even though nothing about leader's behavior actually
// changed, only the probe's own availability did (PR④ known gap #5,
// docs/reports/cross-review-triage-org-runtime-watchdog.md Cycle 3 #5). A
// pending alert whose probe baseline was unavailable can still clear via
// the other, unaffected sources (manifest events, or a probe/history source
// that had a valid baseline).
func (w *watchRun) checkDeadman(status *watchStatusFile, rr ManifestReadResult, now time.Time) {
	// No `if status.Escalated[alertID] { ... }` guard here (removed,
	// tech-debt: "watchdog deferred LOW (3)"): it was unreachable dead code.
	// escalateAlert both sets status.Escalated[alertID] = true AND deletes
	// alertID from status.PendingAlerts in the same call (below), and every
	// alertID is generated fresh per sendAlert call embedding a nanosecond
	// timestamp (`"%s@%d"`, sendAlert), so no alertID this loop's `range
	// status.PendingAlerts` ever visits can already be a key in
	// status.Escalated -- by the time it were, it would already be gone from
	// PendingAlerts too. status.Escalated's only remaining purpose is the
	// historical audit trail pruneEscalated bounds (see escalateAlert).
	for alertID, pending := range status.PendingAlerts {
		activity := leaderActivityEventCount(rr.Events, status.OrgID) > pending.ManifestLen
		if !activity && pending.LeaderAgentGet != "" {
			if cur := w.leaderProbeSnapshot(status.OrgID); cur != "" && cur != pending.LeaderAgentGet {
				activity = true
			}
		}
		if !activity && pending.HistoryLeaderLines >= 0 {
			if cur := w.historyLeaderLineCount(status.OrgID); cur >= 0 && cur > pending.HistoryLeaderLines {
				activity = true
			}
		}

		subjectIsLeader := pending.Subject == LeaderIdentity
		deadmanExceeded := false
		if w.cfg.DeadmanMinutes > 0 {
			if ts, err := time.Parse(time.RFC3339, pending.TS); err == nil {
				deadmanExceeded = now.Sub(ts) > time.Duration(w.cfg.DeadmanMinutes)*time.Minute
			}
		}

		if activity && !subjectIsLeader {
			delete(status.PendingAlerts, alertID) // leader activity clears it
			continue
		}
		if subjectIsLeader || deadmanExceeded {
			w.escalateAlert(status, alertID, pending, now)
		}
	}
}

// escalateAlert appends one line to escalations.jsonl, writes the stderr
// banner, and best-effort-fires EscalateFunc -- deduped by alertID (AC-5:
// "one escalation per alert, ever") via the delete(status.PendingAlerts,
// alertID) below, which removes alertID from the set checkDeadman ranges
// over so this same alert can never be escalated a second time.
// status.Escalated is written here too, but only as a bounded historical
// audit trail (see checkDeadman's own comment) -- it has no remaining
// reader that dedupe depends on.
func (w *watchRun) escalateAlert(status *watchStatusFile, alertID string, pending *watchPendingAlert, now time.Time) {
	reason := "deadman_timeout"
	if pending.Subject == LeaderIdentity {
		reason = "leader_is_anomaly_subject"
	}

	rec := escalationRecord{
		TS: now.UTC().Format(time.RFC3339), OrgID: status.OrgID, AlertID: alertID,
		Subject: pending.Subject, Reason: reason,
	}
	_ = appendJSONLine(w.escalationsPath, rec)

	_, _ = fmt.Fprintf(w.stderr, "WATCHDOG ESCALATION: org=%s alert=%s subject=%s reason=%s -- see %s\n",
		status.OrgID, alertID, pending.Subject, reason, w.escalationsPath)

	// The notification gets its own bound from context.Background(), as the
	// inbox's does (inboxDesktopNotify): an osascript that does not return
	// cannot hold the cycle, and a SIGINT or SIGTERM in the middle of the
	// cycle does not cut off the notification of the escalation recorded
	// above, which is never sent again (self-review L2).
	notifyCtx, cancel := context.WithTimeout(context.Background(), inboxDesktopNotifyTimeout)
	_ = w.escalateFn(notifyCtx, fmt.Sprintf("ralph org watch: escalation for org %s (%s)", status.OrgID, reason))
	cancel()

	status.Escalated[alertID] = true
	pruneEscalated(status.Escalated)
	delete(status.PendingAlerts, alertID)
}

// maxEscalatedEntries caps status.Escalated (tech-debt: "watchdog deferred
// LOW (3)") so a long-running `ralph org watch` process does not grow
// watch-status-<org_id>.json without bound: every alertID this org ever
// escalated was persisted here forever, with the now-removed unreachable
// guard in checkDeadman the only place that ever consulted it (see that
// function's own comment for why status.Escalated no longer needs to gate
// anything -- it survives purely as a bounded audit trail).
const maxEscalatedEntries = 100

// pruneEscalated drops the oldest entries from escalated once it exceeds
// maxEscalatedEntries, keeping status.Escalated's growth bounded. A Go map
// has no insertion order to prune by, so "oldest" is derived deterministically
// from the nanosecond Unix timestamp each alertID already embeds as its
// "<conditionKey>@<unixnano>" suffix (sendAlert's `"%s@%d"` format) -- a
// sort, not a second insertion-order data structure (e.g. a ring buffer)
// kept in parallel with the map.
func pruneEscalated(escalated map[string]bool) {
	if len(escalated) <= maxEscalatedEntries {
		return
	}
	ids := make([]string, 0, len(escalated))
	for id := range escalated {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return escalatedTimestamp(ids[i]) < escalatedTimestamp(ids[j])
	})
	drop := len(ids) - maxEscalatedEntries
	for _, id := range ids[:drop] {
		delete(escalated, id)
	}
}

// escalatedTimestamp extracts the nanosecond Unix timestamp embedded in
// alertID's "<conditionKey>@<unixnano>" suffix (sendAlert's format), or 0 if
// the suffix is missing or unparseable -- an unparseable id then sorts first
// (oldest), the safe direction for a best-effort prune: worst case it drops
// a malformed id before a genuinely old one, never the reverse.
func escalatedTimestamp(alertID string) int64 {
	idx := strings.LastIndex(alertID, "@")
	if idx < 0 {
		return 0
	}
	n, err := strconv.ParseInt(alertID[idx+1:], 10, 64)
	if err != nil {
		return 0
	}
	return n
}
