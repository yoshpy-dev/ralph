package org

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// Package note (pulse layer, PR④ Slice 3): RunWatch implements
// `ralph org watch`'s deterministic pulse layer (see plan
// docs/plans/active/2026-08-02-org-runtime-watchdog.md, AC-3/3b/3c/4/5). It
// evaluates one cycle at a time (evaluateCycle below), never invokes an LLM
// itself (the on-demand semantic-judgment watcher is a separate Slice 4
// concern reached only through the WatchHooks.OnSemanticTrigger seam), and
// persists its own heartbeat/dedupe state to watch-status-<org_id>.json (see
// WatchStatusFileName) so a restart does not re-fire already-handled
// conditions.

// watchdogIdentity is the agmsg identity `ralph org watch` joins/sends
// under -- distinct from LeadIdentity and every seat id, so ALERT traffic
// is attributable to the pulse layer itself in agmsg history (see
// .claude/rules/agent-messaging.md's "watchdog is a mechanism identity, not
// a spawned seat").
const watchdogIdentity = "watchdog"

// WatchStatusFileName returns the file name (within the caller-resolved org
// state directory, typically ResolveOrgStateDir's result) RunWatch reads and
// rewrites every cycle for orgID: heartbeat (last_cycle_ts/cycles) plus the
// AC-3c condition-dedupe and AC-5 deadman state. `ralph org status` never
// reads this file -- it is watch's own observability record, not seat
// roster.
//
// The name is namespaced per org_id (self-review H-1 fix): a single fixed,
// org-agnostic file name meant two orgs watched from the same repository
// (one state directory, since manifest.jsonl/model-receipts.jsonl are also
// shared there) silently clobbered each other's org-scoped fields --
// OrgID/OrgStartTS/Cycles/LastCycleTS/WatchdogJoined and SeatSnapshots (keyed
// by bare SeatID) all overwrote across orgs, while Conditions/PendingAlerts/
// Escalated happened to be safe only because conditionKey already namespaces
// them by org_id. orgID is guaranteed path-safe here: RunWatch's own
// strings.TrimSpace(p.OrgID) == "" gate plus every production caller's
// upstream requireOrgID/ValidateIdentifier check (identifierPattern,
// identifier.go: ^[a-z][a-z0-9-]{0,29}$) both run before this is ever
// reached.
func WatchStatusFileName(orgID string) string {
	return fmt.Sprintf("watch-status-%s.json", orgID)
}

// EscalationsRelName is the file name (within the same state directory)
// AC-5 deadman escalations are appended to, one JSON line per escalation.
const EscalationsRelName = "escalations.jsonl"

// Pulse-layer condition type tags. Used both as the third segment of a
// dedupe conditionKey and as the ALERT message's CONDITION header value.
const (
	condSeatBudget  = "seat_budget"
	condTotalBudget = "total_budget"
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
// satisfies this structurally; RunWatch type-asserts Org.Herdr against it
// per cycle and treats a failed assertion as "probe unavailable" (best
// effort, never fatal to the pulse loop).
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
// scope-change condition's declared-scope signal). Injectable so watch.go
// itself never calls exec.Command directly for this and so tests can drive
// scope changes deterministically without a real git worktree.
type GitStatusFunc func(cwd string) (string, error)

// maxScopeChangeLines bounds how many `git status --porcelain` lines the
// scope-change ALERT body interpolates before summarizing the rest as
// "... N more lines" (tech-debt: "watchdog deferred LOW (2)"). A busy seat's
// worktree can have hundreds of dirty files; interpolating all of them
// whole made the eventual sendAlert message exceed
// protocol.DefaultMaxBodyChars and silently degrade to the content-free
// protocol-validation-failure fallback below, so Lead saw nothing about
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
// the escalations.jsonl record and stderr banner (both of which RunWatch
// always does itself). The real implementation runs `osascript` on darwin
// and no-ops elsewhere; tests inject a stub to assert on the call without
// depending on macOS.
type EscalateFunc func(ctx context.Context, message string) error

// WatchHooks lets a caller observe watch-cycle events without changing
// pulse-layer behavior. Every hook is optional (nil-safe).
type WatchHooks struct {
	// OnSemanticTrigger fires when a condition warrants on-demand watcher
	// judgment (the Slice 4 seam: an on-demand `claude -p` verdict call for
	// stall/scope-change findings that are below the hard budget limit).
	// No-op by default -- this pulse layer never itself invokes an LLM.
	OnSemanticTrigger func(orgID, seatID, conditionType, evidence string)
	// OnCycle fires once at the end of every evaluated cycle (cycle number,
	// 1-based, and the clock value used for that cycle). Mainly test
	// observability.
	OnCycle func(cycleN int, ts time.Time)
}

// WatchParams describes one `ralph org watch` invocation (see RunWatch).
type WatchParams struct {
	OrgID string
	// Interval is the pulse-cycle wait. <= 0 falls back to
	// Org.Config.Watchdog.IntervalSeconds, then to 30s if that is also <= 0.
	Interval time.Duration
	// Cycles, when > 0, caps RunWatch to exactly that many cycles before
	// returning nil -- the deterministic test/CLI (`--once` => Cycles: 1)
	// path that needs no real ticker/sleep. Zero (the default for a real
	// long-running `ralph org watch`) means "run until ctx is done".
	Cycles int
	// StatusDir is the directory watch-status-<org_id>.json (see
	// WatchStatusFileName) and escalations.jsonl live in (typically the
	// resolved org state-dir, the same directory the
	// manifest/receipts stores are rooted at). Required.
	StatusDir string
	// GitStatus overrides the scope-change condition's git probe; nil uses
	// the real `git status --porcelain` (os/exec).
	GitStatus GitStatusFunc
	// Escalate overrides the AC-5 platform-notification side channel; nil
	// uses the real darwin osascript best-effort (no-op elsewhere).
	Escalate EscalateFunc
	// Stderr is where the AC-5 escalation banner is written; nil uses
	// os.Stderr.
	Stderr io.Writer
}

// watchConditionRecord is the AC-3c dedupe record for one conditionKey.
// Active transitions drive the "1 alert/1 cutoff until recovery" rule: a
// condition already Active is never re-alerted; it clears (Active: false)
// the first cycle it is no longer observed true, so a later re-occurrence
// re-alerts. Cutoff is a one-way ratchet -- once true it is never cleared, so
// a budget cutoff is ALERTed at most once per key, ever, and is retried
// until one Stop pass succeeds (self-review H-2 fix; see
// evaluateTotalBudget/evaluateSeatBudget's own doc comments) -- only the
// ALERT is capped at once per key, not the Stop attempt itself (Codex
// advisory finding 1).
type watchConditionRecord struct {
	Active  bool   `json:"active"`
	Cutoff  bool   `json:"cutoff"`
	FirstTS string `json:"first_ts"`
}

// watchPendingAlert is the AC-5 deadman bookkeeping recorded when an ALERT
// is sent: a snapshot of the 3 lead-activity information sources at ALERT
// time, compared against their current value each subsequent cycle. Subject
// is the seat_id the ALERT concerned (empty for the org-level total-budget
// ALERT); Subject == LeadIdentity is the "anomaly subject is Lead itself"
// AC-5 branch that escalates without waiting for the deadman timeout.
//
// HistoryLeadLines is a COUNT of lead-authored agmsg history lines, not the
// filtered text itself (self-review cycle-3 M3-2 fix): see
// historyLeadLineCount's doc comment for why a count-based "did it grow"
// comparison is required instead of exact string equality.
type watchPendingAlert struct {
	AlertID          string `json:"alert_id"`
	TS               string `json:"ts"`
	Subject          string `json:"subject"`
	ManifestLen      int    `json:"manifest_len"`
	LeadAgentGet     string `json:"lead_agent_get"`
	HistoryLeadLines int    `json:"history_lead_lines"`
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
	OrgStartTS     string                           `json:"org_start_ts,omitempty"`
	WatchdogJoined bool                             `json:"watchdog_joined,omitempty"`
	Conditions     map[string]*watchConditionRecord `json:"conditions,omitempty"`
	PendingAlerts  map[string]*watchPendingAlert    `json:"pending_alerts,omitempty"`
	Escalated      map[string]bool                  `json:"escalated,omitempty"`
	SeatSnapshots  map[string]*watchSeatSnapshot    `json:"seat_snapshots,omitempty"`
}

// escalationRecord is one JSON line appended to EscalationsRelName (AC-5).
type escalationRecord struct {
	TS      string `json:"ts"`
	OrgID   string `json:"org_id"`
	AlertID string `json:"alert_id"`
	Subject string `json:"subject,omitempty"`
	Reason  string `json:"reason"`
}

// conditionKey identifies one (org_id, seat_id, condition_type) dedupe slot
// (Codex advisory finding 1). seatID is "" for the org-level total-budget
// condition.
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

// save rewrites path with s's current contents (one os.WriteFile call --
// small enough that partial-write risk is the same tradeoff ManifestStore's
// single-write Append already accepts for this package).
func (s *watchStatusFile) save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("org: watch: create state dir for %s: %w", path, err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("org: watch: marshal status: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("org: watch: write %s: %w", path, err)
	}
	return nil
}

// nowTime returns o.Now() (or time.Now() when Org.Now is unset), mirroring
// (*Org).now's Clock fallback but returning time.Time instead of a
// formatted string -- RunWatch needs time.Time for duration arithmetic.
func (o *Org) nowTime() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

// earliestSpawnTS returns the earliest `spawned` event TS for a real
// (non-dry-run) seat within orgID -- the AC-3b org_start definition -- or ""
// if none exists yet. Manifest TS values are UTC RFC3339 strings written by
// the same (*Org).now clock, so lexical comparison orders them correctly.
func earliestSpawnTS(events []ManifestEvent, orgID string) string {
	var earliest string
	for _, ev := range events {
		if ev.OrgID != orgID || ev.DryRun || ev.Event != EventSpawned {
			continue
		}
		if earliest == "" || ev.TS < earliest {
			earliest = ev.TS
		}
	}
	return earliest
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

// realGitStatus is GitStatusFunc's real implementation.
func realGitStatus(cwd string) (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// realEscalate is EscalateFunc's real implementation: a best-effort darwin
// notification via osascript. Any other GOOS -- or an osascript failure --
// is a silent no-op; escalations.jsonl and the stderr banner are always the
// authoritative escalation record (see RunWatch's doc comment / AC-5's own
// "tests are file-output-authoritative" assumption).
func realEscalate(ctx context.Context, message string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	script := fmt.Sprintf("display notification %q with title \"ralph org watch\"", message)
	return exec.CommandContext(ctx, "osascript", "-e", script).Run()
}

// ResolveWatchInterval returns the effective pulse-cycle interval RunWatch
// will use: requested if positive, else cfg.IntervalSeconds, else a fixed
// 30s fallback. Exported so a caller that needs to report the effective
// cadence before RunWatch itself resolves it internally (e.g. `ralph org
// watch`'s startup banner, self-review LOW fix: the banner used to print the
// raw --interval-seconds flag value, which is 0 by default, rather than the
// interval that is actually running) does not have to duplicate this
// fallback chain.
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

// RunWatch runs the pulse layer for p.OrgID: one evaluateCycle call per
// interval, until p.Cycles is reached (p.Cycles > 0) or ctx is done
// (p.Cycles == 0, the default long-running mode). Every cycle rewrites
// WatchStatusFileName(p.OrgID) inside p.StatusDir with its heartbeat and
// dedupe state; a returned error means a cycle itself failed unrecoverably
// (e.g. manifest read/status write failure) -- individual condition
// evaluation/ALERT/escalation problems are handled best-effort inside a
// cycle and never abort the loop.
func (o *Org) RunWatch(ctx context.Context, p WatchParams, hooks WatchHooks) error {
	if strings.TrimSpace(p.OrgID) == "" {
		return fmt.Errorf("org: watch: org_id is required")
	}
	if strings.TrimSpace(p.StatusDir) == "" {
		return fmt.Errorf("org: watch: state dir is required")
	}

	interval := ResolveWatchInterval(p.Interval, o.Config.Watchdog)

	gitStatus := p.GitStatus
	if gitStatus == nil {
		gitStatus = realGitStatus
	}
	escalateFn := p.Escalate
	if escalateFn == nil {
		escalateFn = realEscalate
	}
	stderr := p.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	statusPath := filepath.Join(p.StatusDir, WatchStatusFileName(p.OrgID))
	escalationsPath := filepath.Join(p.StatusDir, EscalationsRelName)

	status, err := loadWatchStatus(statusPath, p.OrgID)
	if err != nil {
		return err
	}

	run := &watchRun{
		org: o, cfg: o.Config, hooks: hooks,
		gitStatus: gitStatus, escalateFn: escalateFn, stderr: stderr,
		statusPath: statusPath, escalationsPath: escalationsPath,
	}

	n := 0
	for {
		if err := run.evaluateCycle(ctx, p.OrgID, status); err != nil {
			return err
		}
		n++
		if hooks.OnCycle != nil {
			hooks.OnCycle(n, o.nowTime())
		}
		if p.Cycles > 0 && n >= p.Cycles {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(interval):
		}
	}
}

// watchRun bundles the per-invocation dependencies evaluateCycle and its
// helpers need, so RunWatch's own signature stays small and every helper
// below is a method rather than a long parameter list.
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

// evaluateCycle runs exactly one pulse-layer cycle: total-budget check,
// per-active-seat checks (seat budget / stall / liveness / scope-change),
// then the AC-5 deadman sweep over any still-pending alerts, then persists
// status.
func (w *watchRun) evaluateCycle(ctx context.Context, orgID string, status *watchStatusFile) error {
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

	if start := earliestSpawnTS(rr.Events, orgID); start != "" {
		status.OrgStartTS = start
	}

	var activeSeats []SeatStatus
	for _, s := range Roster(rr.Events, RosterOptions{}) {
		if s.OrgID == orgID && s.Active {
			activeSeats = append(activeSeats, s)
		}
	}

	cutSeats := w.evaluateTotalBudget(ctx, status, orgID, activeSeats, now)
	for _, s := range activeSeats {
		if cutSeats[s.SeatID] {
			// Cross-review triage cycle-2 #4: a seat evaluateTotalBudget just
			// stopped this cycle must not also run through evaluateSeat's
			// stall/liveness/scope-change checks against its now-stale
			// SeatStatus snapshot -- that would ALERT (and record a deadman
			// pending-alert) against a seat that is no longer active, purely
			// because activeSeats was computed before this cycle's cutoff ran.
			continue
		}
		w.evaluateSeat(ctx, status, orgID, s, now, rr.Events)
	}

	rr2, err := w.org.Manifest.Read()
	if err != nil {
		return fmt.Errorf("org: watch: re-read manifest: %w", err)
	}
	w.checkDeadman(ctx, status, rr2, now)

	status.LastCycleTS = now.UTC().Format(time.RFC3339)
	status.Cycles++
	if err := status.save(w.statusPath); err != nil {
		return err
	}
	return nil
}

// evaluateTotalBudget implements AC-3b: once now - org_start exceeds
// [org.budget].total_wall_clock_minutes, every currently active seat is cut
// off (each Stop call carries its own Reason) and a single org-level ALERT
// is sent -- deduped by the org-level conditionKey (seatID "") exactly like
// any other cutoff condition (Codex advisory findings 1+2).
//
// The Cutoff ratchet (self-review H-2 fix) is only set once every Stop call
// in this pass returned no error: Stop's error is non-nil exactly when
// findSeat's manifest read or the `stopped` event's own appendEvent write
// failed (Stop's PaneSendKeys/Leave failures are best-effort and only ever
// recorded in the stopped event's Details, never returned as an error -- see
// verbs.go's Stop). Setting Cutoff on a failed Stop would permanently
// disable the org's only enforcement action for that key with no path to
// retry and no log line; instead, a failed Stop is logged to w.stderr and
// the condition is left un-ratcheted, so the next cycle re-evaluates (and
// retries Stop only for seats that are still active -- a seat this pass did
// successfully stop no longer appears in activeSeats on the next call,
// since Roster no longer reports it Active). The ALERT itself is still sent
// exactly once per key (guarded by the same Conditions[key].Active flag
// raiseOrClear uses for non-cutoff conditions), regardless of whether the
// Stop attempt(s) succeeded, so lead is never left uninformed of a
// still-in-progress cutoff.
//
// The returned set (seat_id -> true) names every seat this call itself
// successfully stopped, so evaluateCycle's per-seat loop can skip them for
// the remainder of THIS cycle (cross-review-triage cycle-2 #4): activeSeats
// is a snapshot taken before this call runs, so without that skip the
// caller's subsequent evaluateSeat pass would still probe a just-cut seat's
// now-stale SeatStatus and could raise a spurious stall/liveness/scope-change
// ALERT (and deadman pending-alert record) against a seat that is no longer
// active.
func (w *watchRun) evaluateTotalBudget(ctx context.Context, status *watchStatusFile, orgID string, activeSeats []SeatStatus, now time.Time) map[string]bool {
	cutSeats := map[string]bool{}
	if status.OrgStartTS == "" || w.cfg.Budget.TotalWallClockMinutes <= 0 {
		return cutSeats
	}
	if len(activeSeats) == 0 {
		// Cross-review AR-2: no active seats means nothing to cut off. Without
		// this guard, the loop below ranges over an empty activeSeats, so
		// allStopped stays vacuously true and the Cutoff ratchet gets set
		// (self-review H-2's own "one cutoff per key, ever" rule, watch.go's
		// rec.Cutoff early return above) even though no seat was ever
		// actually stopped -- permanently disabling this key's enforcement
		// for any seat spawned into the org afterward, with only a single
		// spurious ALERT to show for it (see
		// TestWatch_TotalBudgetCutoff_NoActiveSeats_NoAlertNoEscalation_ThenCutsNewSeat's
		// third phase, the concrete regression this guard prevents).
		//
		// Recovered semantics (tech-debt: "watchdog deferred LOW (4)"): if
		// this key already has an Active condition record -- i.e. an ALERT
		// for this exact org-level condition fired in a prior cycle, while
		// seats were still active -- clear its Active flag here, mirroring
		// raiseOrClear's own "recovered: clears, a future re-occurrence
		// re-alerts" rule for non-cutoff conditions. Without this, a
		// resumed org (every seat stopped, then a new seat spawned into the
		// still-over-budget org) would have its next cutoff gated silently:
		// evaluateTotalBudget's `alreadyAlerted := rec != nil && rec.Active`
		// check below would see the never-cleared true from before every
		// seat stopped and skip the ALERT, even though nothing was ever
		// actually sent to lead about *this* occurrence. This only clears
		// Active, not rec.Cutoff -- the separate "one cutoff per key, ever"
		// ratchet above is unaffected, so a prior successful cutoff still
		// permanently disables Stop for this key regardless.
		if rec := status.Conditions[conditionKey(orgID, "", condTotalBudget)]; rec != nil {
			rec.Active = false
		}
		return cutSeats
	}
	start, err := time.Parse(time.RFC3339, status.OrgStartTS)
	if err != nil {
		return cutSeats
	}
	observed := now.Sub(start)
	if observed <= time.Duration(w.cfg.Budget.TotalWallClockMinutes)*time.Minute {
		return cutSeats
	}

	key := conditionKey(orgID, "", condTotalBudget)
	rec := status.Conditions[key]
	if rec != nil && rec.Cutoff {
		return cutSeats // AC-3c: one cutoff per key, ever
	}

	details := fmt.Sprintf("watchdog_total_budget_cutoff total_wall_clock=%dm observed=%s",
		w.cfg.Budget.TotalWallClockMinutes, observed.Round(time.Minute))
	allStopped := true
	for _, s := range activeSeats {
		if stopErr := w.org.Stop(StopParams{OrgID: orgID, Seat: s.SeatID, Reason: details}).Err; stopErr != nil {
			allStopped = false
			_, _ = fmt.Fprintf(w.stderr, "watchdog: total-budget cutoff Stop failed for org %q seat %q: %v -- will retry next cycle\n",
				orgID, s.SeatID, stopErr)
		} else {
			cutSeats[s.SeatID] = true
		}
	}

	alreadyAlerted := rec != nil && rec.Active
	status.Conditions[key] = &watchConditionRecord{Active: true, Cutoff: allStopped, FirstTS: conditionFirstTS(rec, now)}

	if !alreadyAlerted {
		msg := fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nCONDITION: %s\n\n%s", orgID, condTotalBudget, details)
		w.sendAlert(ctx, status, orgID, "", condTotalBudget, msg, now)
	}
	return cutSeats
}

// conditionFirstTS preserves rec's original FirstTS across a retried cutoff
// attempt (self-review H-2 fix): a failed Stop leaves rec.Cutoff false but
// must not reset FirstTS to the retry cycle's own now, or a condition that
// takes several cycles to successfully cut off would misreport when it
// first became active. rec is nil on the first-ever observation of the key,
// in which case now is the correct FirstTS.
func conditionFirstTS(rec *watchConditionRecord, now time.Time) string {
	if rec != nil && rec.FirstTS != "" {
		return rec.FirstTS
	}
	return now.UTC().Format(time.RFC3339)
}

// evaluateSeat runs every per-seat pulse condition for s: (a) seat wall-
// clock budget cutoff, (c) heartbeat stall, (d) process liveness, (e)
// worktree scope change. events is the cycle's manifest snapshot (see
// evaluateCycle), needed by the stall condition's M-6 fix below.
//
// A successful (a) cutoff this cycle returns early before (c)/(d)/(e) run
// (cross-review-triage cycle-2 #4): s is a snapshot taken before the cutoff,
// so probing it further this cycle would raise a spurious stall/liveness/
// scope-change ALERT (and deadman pending-alert record) against a seat that
// evaluateSeatBudget, moments earlier in this same call, already stopped. A
// failed Stop attempt (evaluateSeatBudget returns false) still falls through
// to the rest of this cycle's checks, unchanged from before this fix -- the
// seat is still genuinely active, so there is nothing to skip.
func (w *watchRun) evaluateSeat(ctx context.Context, status *watchStatusFile, orgID string, s SeatStatus, now time.Time, events []ManifestEvent) {
	if w.evaluateSeatBudget(ctx, status, orgID, s, now) {
		return
	}

	snap := status.SeatSnapshots[s.SeatID]
	if snap == nil {
		snap = &watchSeatSnapshot{}
		status.SeatSnapshots[s.SeatID] = snap
	}

	var agentGetOut string
	var agentGetErr error
	if probe, ok := w.org.Herdr.(watchHerdrProbe); ok {
		agentGetOut, agentGetErr = probe.AgentGet(ctx, resolvedHerdrAgentName(s))
	}

	if agentGetErr != nil {
		// (d) process liveness: herdr agent get failed for an active seat.
		w.raiseOrClear(ctx, status, orgID, s.SeatID, condLiveness, true, now,
			fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: %s\n\nherdr agent get failed for seat %s: %v",
				orgID, s.SeatID, condLiveness, s.SeatID, agentGetErr), false)
	} else {
		w.raiseOrClear(ctx, status, orgID, s.SeatID, condLiveness, false, now, "", false)

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
		w.raiseOrClear(ctx, status, orgID, s.SeatID, condStall, stalled, now,
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
			w.raiseOrClear(ctx, status, orgID, s.SeatID, condScopeChange, changed, now,
				fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: %s\n\nseat %s worktree scope changed:\n%s",
					orgID, s.SeatID, condScopeChange, s.SeatID, truncateScopeOutput(out, scopeChangeBodyBudget)), true)
			snap.GitStatus = out
			snap.GitStatusSeen = true
		}
	}
}

// evaluateSeatBudget implements AC-3: seat wall-clock budget cutoff. s.TS is
// the seat's latest applicable manifest event time, which for an Active
// seat is the `spawned` event's own TS (no further state event has
// superseded it) -- exactly the "spawned.ts" the plan's Reason format
// references.
//
// The Cutoff ratchet (self-review H-2 fix) is only set once Stop returns no
// error -- see evaluateTotalBudget's doc comment for the full rationale
// (identical for the single-seat case here): a failed Stop is logged to
// w.stderr and the condition is left un-ratcheted so the next cycle retries,
// while the ALERT is still sent exactly once per key regardless of the Stop
// outcome.
//
// The returned bool is true exactly when this call itself successfully
// stopped s this cycle, so evaluateSeat can skip its remaining per-cycle
// checks for s (cross-review-triage cycle-2 #4). It is false both when no
// cutoff condition applied at all and when a cutoff was attempted but Stop
// failed (s is still genuinely active in the latter case).
func (w *watchRun) evaluateSeatBudget(ctx context.Context, status *watchStatusFile, orgID string, s SeatStatus, now time.Time) bool {
	if w.cfg.Budget.SeatWallClockMinutes <= 0 || s.TS == "" {
		return false
	}
	spawned, err := time.Parse(time.RFC3339, s.TS)
	if err != nil {
		return false
	}
	observed := now.Sub(spawned)
	if observed <= time.Duration(w.cfg.Budget.SeatWallClockMinutes)*time.Minute {
		return false
	}

	key := conditionKey(orgID, s.SeatID, condSeatBudget)
	rec := status.Conditions[key]
	if rec != nil && rec.Cutoff {
		return false // AC-3c: one cutoff per key, ever -- already cut off in a prior cycle
	}

	details := fmt.Sprintf("watchdog_budget_cutoff seat_wall_clock=%dm observed=%s",
		w.cfg.Budget.SeatWallClockMinutes, observed.Round(time.Minute))
	stopErr := w.org.Stop(StopParams{OrgID: orgID, Seat: s.SeatID, Reason: details}).Err
	if stopErr != nil {
		_, _ = fmt.Fprintf(w.stderr, "watchdog: seat-budget cutoff Stop failed for org %q seat %q: %v -- will retry next cycle\n",
			orgID, s.SeatID, stopErr)
	}

	alreadyAlerted := rec != nil && rec.Active
	status.Conditions[key] = &watchConditionRecord{Active: true, Cutoff: stopErr == nil, FirstTS: conditionFirstTS(rec, now)}

	if !alreadyAlerted {
		msg := fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: %s\n\n%s", orgID, s.SeatID, condSeatBudget, details)
		w.sendAlert(ctx, status, orgID, s.SeatID, condSeatBudget, msg, now)
	}
	return stopErr == nil
}

// raiseOrClear implements the AC-3c idempotent ALERT dedupe for a
// non-cutoff condition: an active==false->true transition sends exactly one
// ALERT and records the key as Active; the key clears (Active: false) the
// first cycle active is observed false again, so a later re-occurrence
// re-alerts. semantic, when true and the condition is newly active, also
// fires WatchHooks.OnSemanticTrigger (the Slice 4 on-demand-watcher seam) --
// liveness intentionally passes semantic=false: a dead/unreachable pane
// gives a watcher nothing useful to judge.
func (w *watchRun) raiseOrClear(ctx context.Context, status *watchStatusFile, orgID, seatID, condType string, active bool, now time.Time, message string, semantic bool) {
	key := conditionKey(orgID, seatID, condType)
	rec := status.Conditions[key]
	if active {
		if rec != nil && rec.Active {
			return // already alerted and still active: dedupe
		}
		status.Conditions[key] = &watchConditionRecord{Active: true, FirstTS: now.UTC().Format(time.RFC3339)}
		w.sendAlert(ctx, status, orgID, seatID, condType, message, now)
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
// identity to LeadIdentity over orgID's agmsg team, using Agmsg.Send
// directly rather than the seat-steering Send verb (verbs.go). Send resolves
// its To target as a spawned SEAT via findSeat, which fails -- silently
// dropping the message -- in the normal "session-promoted lead" org shape
// where no lead SEAT was ever spawned (only the lead identity itself,
// registered via ensureLeadJoined/ensureWatchdogJoined's Join calls). Live
// smoke (docs/plans/active/2026-08-02-org-runtime-watchdog.md) found zero
// ALERTs reaching agmsg history under exactly that shape while escalations
// still fired. Both this package's own pulse-layer sendAlert and
// internal/cli/org.go's on-demand watcher-verdict ALERT path
// (newWatchdogHooks) call this so ALERT delivery is identical regardless of
// which layer produced the finding.
func (o *Org) SendWatchdogAlert(ctx context.Context, orgID, message string) error {
	return o.Agmsg.Send(ctx, agmsgTeam(orgID), watchdogIdentity, LeadIdentity, message)
}

// ensureWatchdogJoined best-effort-joins the "watchdog" mechanism identity
// (see the const's doc comment) onto the org's agmsg team exactly once per
// RunWatch's persisted status -- mirrors ensureLeadJoined's idempotent,
// best-effort Join semantics in spawn.go, but for the watchdog identity
// instead of lead.
//
// WatchdogJoined is set only after Join actually succeeds (PR④ known gap
// #6, docs/reports/cross-review-triage-org-runtime-watchdog.md Cycle 3 #6):
// setting it unconditionally would persist a false "already joined" record
// across a transient Join failure (e.g. agmsg momentarily unreachable),
// permanently skipping every future retry for that org -- and, since
// SendWatchdogAlert has no membership dependency of its own that would
// otherwise surface the gap, ALERT delivery would keep failing silently
// forever after just one bad cycle. Leaving the flag false on error lets
// the very next cycle retry Join.
func (w *watchRun) ensureWatchdogJoined(ctx context.Context, status *watchStatusFile, orgID string) {
	if status.WatchdogJoined {
		return
	}
	cwd, _ := os.Getwd()
	if err := w.org.Agmsg.Join(ctx, agmsgTeam(orgID), watchdogIdentity, "claude-code", cwd); err == nil {
		status.WatchdogJoined = true
	}
}

// sendAlert validates and sends one ALERT to lead via the watchdog identity
// (SendWatchdogAlert, not the seat-steering Send verb -- see that method's
// doc comment for why) best-effort -- a Send failure, e.g. agmsg itself is
// unreachable, never aborts the pulse cycle but is logged to w.stderr -- then
// registers an AC-5 pending-alert deadman record regardless of whether Send
// itself succeeded: the whole point of the deadman clause is to catch the
// case where lead cannot be reached at all.
func (w *watchRun) sendAlert(ctx context.Context, status *watchStatusFile, orgID, seatID, condType, message string, now time.Time) {
	if err := protocol.ValidateText(message, protocol.DefaultMaxBodyChars); err != nil {
		// SEAT is always included here (tech-debt: "watchdog deferred LOW
		// (2)"), even when seatID is "" (an org-level condition like
		// condTotalBudget): before this fix, a busy seat whose original
		// message failed validation degraded to a fallback with no subject
		// at all, so Lead saw only "message failed protocol validation" with
		// no way to tell which seat (or that it was org-level) the finding
		// was about.
		message = fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: %s\n\nwatchdog: message failed protocol validation: %v",
			orgID, seatID, condType, err)
	}
	w.ensureWatchdogJoined(ctx, status, orgID)
	if err := w.org.SendWatchdogAlert(ctx, orgID, message); err != nil {
		_, _ = fmt.Fprintf(w.stderr, "watchdog: failed to ALERT lead for org %q condition %q: %v\n", orgID, condType, err)
	}

	rr, _ := w.org.Manifest.Read()
	alertID := fmt.Sprintf("%s@%d", conditionKey(orgID, seatID, condType), now.UnixNano())
	status.PendingAlerts[alertID] = &watchPendingAlert{
		AlertID:          alertID,
		TS:               now.UTC().Format(time.RFC3339),
		Subject:          seatID,
		ManifestLen:      leadActivityEventCount(rr.Events, orgID),
		LeadAgentGet:     w.leadProbeSnapshot(ctx, orgID),
		HistoryLeadLines: w.historyLeadLineCount(ctx, orgID),
	}
}

// leadActivityEventCount counts manifest events attributable to lead for
// orgID (self-review M-4 fix, org-scoped per cross-review AR-1; the seat-
// attribution model itself was corrected by self-review cycle-3 H3-1 --
// see below): a genuinely unresponsive lead must not have its deadman
// escalation silently cleared by an unrelated seat's own manifest traffic,
// so an event only counts here when either (a) or (b) holds:
//
//	(a) ev.Event == EventSent. A `sent` event is lead-authored BY
//	    CONSTRUCTION, regardless of ev.SeatID: ev.SeatID on a `sent` event
//	    is the *recipient* (Send writes SeatID: p.To -- see verbs.go), not
//	    the author, and `ralph org send` is the only verb that ever appends
//	    one. In the star topology (.claude/rules/agent-messaging.md), only
//	    lead/the operator drives that verb -- a seat's reply travels over
//	    the agmsg skill, which never touches this manifest at all -- so
//	    every `sent` event in orgID's manifest was written by lead sending
//	    to someone, never by a seat sending to lead. (Cycle-2's fix used
//	    `ev.SeatID == LeadIdentity` here, which is backwards: it excluded
//	    the star topology's mandated seat->lead `sent` traffic while
//	    treating lead->seat sends as nothing. See
//	    TestWatch_Deadman_SeatSentEvent_ClearsPendingAlert_WatchdogCutoffDoesNot.)
//	(b) it is a non-watchdog event from the lead-driven lifecycle set
//	    (spawned, spawn_started, stopped, disbanded, rejected) that is not
//	    the watchdog's own cutoff write. Each of these is only producible by
//	    a `ralph org` verb that lead/the operator runs (spawn/stop/disband),
//	    so it is evidence lead is alive and acting, even when the event
//	    itself names a seat, not lead (e.g. lead spawning a replacement seat
//	    in response to a stall ALERT, self-review cycle-3 M3-1). Every
//	    `stopped` event a budget cutoff produces carries
//	    "reason=watchdog_..." in its Details (see
//	    evaluateTotalBudget/evaluateSeatBudget's Reason format), which is
//	    what excludes the watchdog's own cutoffs from (b).
//
// The orgID filter excludes another org's activity in the same shared
// manifest: without it, a new event in a different, active org would clear
// a stalled org's pending deadman alert even though nothing happened in the
// stalled org itself.
func leadActivityEventCount(events []ManifestEvent, orgID string) int {
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
		case EventSpawned, EventSpawnStarted, EventStopped, EventDisbanded, EventRejected:
			if !strings.Contains(ev.Details, "reason=watchdog_") {
				n++
			}
		}
	}
	return n
}

// leadProbeSnapshot returns the lead seat's current herdr `agent get` raw
// text, or "" if the probe is unavailable/errors (best-effort deadman
// information source #2).
func (w *watchRun) leadProbeSnapshot(ctx context.Context, orgID string) string {
	probe, ok := w.org.Herdr.(watchHerdrProbe)
	if !ok {
		return ""
	}
	out, err := probe.AgentGet(ctx, herdrAgentName(orgID, LeadIdentity))
	if err != nil {
		return ""
	}
	return out
}

// historyLeadLineCount returns the COUNT of the org's agmsg team history
// lines that are LEAD-authored (leadHistoryLines), or -1 if the probe is
// unavailable/errors (best-effort deadman information source #3; -1, not 0,
// so an unavailable probe is distinguishable from "lead has genuinely never
// sent anything yet").
//
// Counting rather than comparing the filtered text (self-review cycle-3
// M3-2 fix) matters because the window this reads is NOT scoped to lead's
// own traffic: agentID is passed as "" below, which -- per the driver's own
// doc comment on History -- makes it drop its LIMIT argument entirely and
// fall through to the agmsg skill's history.sh script's own default
// (LIMIT=20, see ~/.agents/skills/agmsg/scripts/history.sh -- confirmed by
// reading that script directly; it is a user-global skill install, not
// vendored into this repo, so `.agents/skills/` here has no `agmsg/`
// subdirectory to find it in), applied to the WHOLE team's traffic, not
// just lead's. As other seats chat, older lead lines get evicted from that
// last-20-of-everyone window and the filtered text can shrink even though
// lead did nothing new -- comparing exact strings would read a pure
// eviction as "activity" and wrongly clear a pending alert (the bug this
// fix closes). A count comparison that only treats growth (cur > baseline,
// see checkDeadman) as activity does not have that failure mode: eviction
// can only ever decrease the count, never manufacture an increase. Passing
// LeadIdentity as agentID to scope the query itself would be a more
// complete fix (and would make the `20` argument here non-dead) but is
// deferred -- the count comparison alone is sufficient to close the false-
// activity bug the cycle-3 finding described.
func (w *watchRun) historyLeadLineCount(ctx context.Context, orgID string) int {
	probe, ok := w.org.Agmsg.(watchAgmsgHistory)
	if !ok {
		return -1
	}
	out, err := probe.History(ctx, agmsgTeam(orgID), "", 20)
	if err != nil {
		return -1
	}
	return len(leadHistoryLines(out))
}

// leadHistoryFromField parses one agmsg history line -- the real shape is
// "  <status> [<ts>] <from> → <to>: <body>" (see the agmsg skill's
// `scripts/history.sh`, a user-global install under
// `~/.agents/skills/agmsg/`, not vendored in this repo -- its `echo "
// $status [$ts] $from → $to: $body"` line) -- and returns its from field.
// ok is false whenever the line does not contain both the "] " and " → "
// markers this parse depends on; a caller must then exclude the line
// entirely rather than guess, since an unparseable line could just as
// easily be lead- as non-lead-authored (defensive: exclude on parse
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
func leadHistoryFromField(line string) (string, bool) {
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

// leadHistoryLines returns only raw's lines whose parsed from field
// (leadHistoryFromField) is exactly LeadIdentity. Lines that fail to parse
// are excluded, not conservatively kept -- see leadHistoryFromField's doc
// comment. Shared by filterLeadHistoryLines (text, used by tests to pin the
// parsing contract directly) and historyLeadLineCount (count, used by the
// production deadman check) so the two never drift on what counts as a
// lead line.
func leadHistoryLines(raw string) []string {
	var kept []string
	for line := range strings.SplitSeq(raw, "\n") {
		if from, ok := leadHistoryFromField(line); ok && from == LeadIdentity {
			kept = append(kept, line)
		}
	}
	return kept
}

// filterLeadHistoryLines returns leadHistoryLines(raw) joined back with
// "\n" -- kept as a thin wrapper so TestFilterLeadHistoryLines can keep
// pinning the parsing contract as a single string comparison.
func filterLeadHistoryLines(raw string) string {
	return strings.Join(leadHistoryLines(raw), "\n")
}

// checkDeadman implements AC-5: for every still-pending ALERT, look for
// lead activity (any of the 3 information sources changed since the ALERT
// was sent) and either clear the pending record (activity found, and the
// anomaly subject is not lead itself) or escalate (deadman_minutes elapsed
// with no activity, OR the anomaly subject is lead itself -- escalates
// without waiting for the timeout, since lead cannot be expected to
// self-report while it is the thing that is anomalous).
//
// The probe-based sources (#2 leadProbeSnapshot, #3 historyLeadLineCount)
// only count as activity when the ALERT-time baseline itself was a valid,
// comparable snapshot -- not the "probe was unavailable" sentinel
// (LeadAgentGet == "" / HistoryLeadLines == -1, per leadProbeSnapshot's and
// historyLeadLineCount's own doc comments, the producers of these values).
// Without that guard, an alert recorded while a probe was down
// (baseline collapses to the sentinel) would false-clear the moment the
// probe merely recovers on a later cycle: cur != "" is trivially true
// against a "" baseline even though nothing about lead's behavior actually
// changed, only the probe's own availability did (PR④ known gap #5,
// docs/reports/cross-review-triage-org-runtime-watchdog.md Cycle 3 #5). A
// pending alert whose probe baseline was unavailable can still clear via
// the other, unaffected sources (manifest events, or a probe/history source
// that had a valid baseline).
func (w *watchRun) checkDeadman(ctx context.Context, status *watchStatusFile, rr ManifestReadResult, now time.Time) {
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
		activity := leadActivityEventCount(rr.Events, status.OrgID) > pending.ManifestLen
		if !activity && pending.LeadAgentGet != "" {
			if cur := w.leadProbeSnapshot(ctx, status.OrgID); cur != "" && cur != pending.LeadAgentGet {
				activity = true
			}
		}
		if !activity && pending.HistoryLeadLines >= 0 {
			if cur := w.historyLeadLineCount(ctx, status.OrgID); cur >= 0 && cur > pending.HistoryLeadLines {
				activity = true
			}
		}

		subjectIsLead := pending.Subject == LeadIdentity
		deadmanExceeded := false
		if w.cfg.DeadmanMinutes > 0 {
			if ts, err := time.Parse(time.RFC3339, pending.TS); err == nil {
				deadmanExceeded = now.Sub(ts) > time.Duration(w.cfg.DeadmanMinutes)*time.Minute
			}
		}

		if activity && !subjectIsLead {
			delete(status.PendingAlerts, alertID) // lead activity clears it
			continue
		}
		if subjectIsLead || deadmanExceeded {
			w.escalateAlert(ctx, status, alertID, pending, now)
		}
	}
}

// escalateAlert appends one line to escalations.jsonl, writes the stderr
// banner, and best-effort-fires EscalateFunc -- deduped by alertID (AC-5:
// "one escalation per alert, ever") via status.Escalated.
func (w *watchRun) escalateAlert(ctx context.Context, status *watchStatusFile, alertID string, pending *watchPendingAlert, now time.Time) {
	reason := "deadman_timeout"
	if pending.Subject == LeadIdentity {
		reason = "lead_is_anomaly_subject"
	}

	rec := escalationRecord{
		TS: now.UTC().Format(time.RFC3339), OrgID: status.OrgID, AlertID: alertID,
		Subject: pending.Subject, Reason: reason,
	}
	_ = appendJSONLine(w.escalationsPath, rec)

	_, _ = fmt.Fprintf(w.stderr, "WATCHDOG ESCALATION: org=%s alert=%s subject=%s reason=%s -- see %s\n",
		status.OrgID, alertID, pending.Subject, reason, w.escalationsPath)

	_ = w.escalateFn(ctx, fmt.Sprintf("ralph org watch: escalation for org %s (%s)", status.OrgID, reason))

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
