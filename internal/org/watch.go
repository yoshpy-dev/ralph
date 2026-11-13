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
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

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

// EscalationsRelName is the file name (within the same state directory) the
// human path appends to, one JSON line per send: each send of an inbox item
// (escalate.go), the watch's ALERT items included, and each AC-5 watch
// escalation the inbox could not take (escalateWithoutInbox).
const EscalationsRelName = "escalations.jsonl"

// Pulse-layer condition type tags. Used both as the third segment of a
// dedupe conditionKey and as the ALERT message's CONDITION header value.
// condUsageLimited is a condition of the org, not of one seat: its key's
// seat is the leader (raiseUsageLimited, watch_usage_limit.go; plan
// 2026-11-13-watch-usage-limited, D2).
const (
	condStall        = "stall"
	condLiveness     = "liveness"
	condScopeChange  = "scope_change"
	condUsageLimited = "usage_limited"
)

// watchHerdrProbe is a consumption-side extension of HerdrClient (see
// architecture.md: "prefer interfaces at consumption sites") for the herdr
// `agent get` that send/wait/read/stop never call. Two readers need it: the
// pulse layer's liveness/stall conditions, and the check of a headless
// director in herdr (CheckDirector, director_check.go; plan
// 2026-10-18-org-director-liveness, D3), which the cross-org watch runs every
// cycle. Defined here rather than added to HerdrClient itself in spawn.go,
// so that interface's existing implementers (real driver.Herdr, and every
// pre-Slice-3 test fake) are unaffected. driver.Herdr -- the real
// implementation internal/cli/org.go's newOrgRuntime wires into every
// Org.Herdr -- already satisfies this structurally; both readers type-assert
// Org.Herdr against it per call. The pulse cycle treats a failed assertion
// as "probe unavailable" (best effort, never fatal to the pulse loop), and
// the director check as unknown (DirectorCheckHowNoAgentGet).
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

// EscalateFunc sends one best-effort desktop notification of message and
// returns its error, which the caller records as the notification's result
// (ok or failed: <err>). It is the type of Org.DesktopNotify (the inbox's
// human path, which records the result in the item's notified event) and of
// WatchAllParams.Escalate (a watch escalation the inbox could not take,
// whose escalations.jsonl line records it). When either is nil, the
// notification is osascript on darwin and none elsewhere (skipped); tests
// inject a stub to assert on the call without depending on macOS.
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
// cycle it is no longer observed true, so a later re-occurrence is a new
// rising edge. That edge sends a new ALERT only when the key's earlier
// ALERT has ended; while it is open, the edge is a recurrence of it
// (raiseOrClear; plan 2026-11-12-watch-alert-dedupe, D1).
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
//
// Recurrences counts the rising edges of the same (org, seat, condition)
// the watch saw while this alert was pending (raiseOrClear; plan
// 2026-11-12-watch-alert-dedupe, D1). Each was recorded here instead of a
// new ALERT; escalateToInbox carries the count to the alert's inbox item
// (AC4) in one recurred event, which is recorded at escalation time, so the
// item's last_recurred_at for that event is the escalation time, not the
// time of the last firing. It is omitted while zero, so a status file with
// no recurrence has the shape it had before that plan, and one written
// before it loads with no recurrence (AC12).
//
// UsageLimited holds, for an alert of the org condition usage_limited, the
// seats of the org stopped on a usage limit, each with its driver and the
// usage limit line of its screen (raiseUsageLimited, watch_usage_limit.go;
// plan 2026-11-13-watch-usage-limited, D3), from which escalateToInbox
// builds the item's body. Its Subject is the leader, the seat of the
// condition's key, so checkDeadman escalates it in the cycle that raised
// it. It is nil for every other condition and omitted while nil, so a
// status file written before that plan loads with none. It is a pointer to
// a struct, not a slice, so that watchPendingAlert stays comparable:
// TestWatch_Recurrence_APendingAlertTheInboxDoesNotHoldIsKept (watch_test.go)
// compares a pending alert with a copy of it (`*p != want`), which does not
// compile for a struct with a slice field.
type watchPendingAlert struct {
	AlertID            string                  `json:"alert_id"`
	TS                 string                  `json:"ts"`
	Subject            string                  `json:"subject"`
	ManifestLen        int                     `json:"manifest_len"`
	LeaderAgentGet     string                  `json:"lead_agent_get"`     // tag predates the leader rename so older state decodes; renaming it would zero the value
	HistoryLeaderLines int                     `json:"history_lead_lines"` // tag predates the leader rename so older state decodes; renaming it would lose the -1 sentinel
	Recurrences        int                     `json:"recurrences,omitempty"`
	UsageLimited       *watchUsageLimitedSeats `json:"usage_limited,omitempty"`
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
//
// GitStatus is the scope-change baseline of an org without a reservation.
// One that holds a reservation keeps ScopeHead, ScopeOutsidePaths, and
// ScopeOutsidePathsSeen instead (evaluateReservedScope, watch_scope.go; plan
// 2026-11-12-watch-alert-dedupe, D5 and D6): the worktree's HEAD the
// previous cycle read (or an earlier one, while the commits after it could
// not be listed), and the sorted paths outside the reservation it found.
// ScopeOutsidePathsSeen tells a baseline with no path from none. Its key,
// scope_outside_paths_seen, differs from an earlier scope_seen (with
// scope_outside), which held porcelain entries per path, so a status file
// that holds that earlier per-entry baseline starts a new baseline instead of
// comparing against it. All three are omitted while empty, so a status file
// written before that plan loads with no baseline, and its first cycle
// records one (AC12).
type watchSeatSnapshot struct {
	AgentGet      string `json:"agent_get,omitempty"`
	AgentGetSeen  bool   `json:"agent_get_seen,omitempty"`
	GitStatus     string `json:"git_status,omitempty"`
	GitStatusSeen bool   `json:"git_status_seen,omitempty"`
	// The scope-change baseline of an org that holds a reservation.
	ScopeHead             string   `json:"scope_head,omitempty"`
	ScopeOutsidePaths     []string `json:"scope_outside_paths,omitempty"`
	ScopeOutsidePathsSeen bool     `json:"scope_outside_paths_seen,omitempty"`
}

// watchStatusFile is the JSON shape persisted to WatchStatusFileName(org_id).
//
// UsageLimitedSeats is the set of the org's seats stopped on a usage limit
// in the last cycle while its usage_limited condition is active, sorted; a
// seat that is not in it is a new member, which raises the condition again
// (raiseUsageLimited, watch_usage_limit.go; plan
// 2026-11-13-watch-usage-limited, AC4). It is emptied when the condition
// clears, and omitted while empty, so a status file written before that
// plan loads with none.
type watchStatusFile struct {
	OrgID             string                           `json:"org_id"`
	LastCycleTS       string                           `json:"last_cycle_ts"`
	Cycles            int                              `json:"cycles"`
	WatchdogJoined    bool                             `json:"watchdog_joined,omitempty"`
	Conditions        map[string]*watchConditionRecord `json:"conditions,omitempty"`
	PendingAlerts     map[string]*watchPendingAlert    `json:"pending_alerts,omitempty"`
	Escalated         map[string]bool                  `json:"escalated,omitempty"`
	SeatSnapshots     map[string]*watchSeatSnapshot    `json:"seat_snapshots,omitempty"`
	UsageLimitedSeats []string                         `json:"usage_limited_seats,omitempty"`
}

// escalationRecord is one JSON line appended to EscalationsRelName. An
// inbox item sent to the human path (sendInboxItemToHuman, escalate.go)
// carries inbox_id; when the item is a TYPE ALERT item, which the cross-org
// watch records for one of its escalations (escalateAlert), the line also
// carries its alert_id and the ALERT's seat as subject. A watch escalation
// the inbox could not take (escalateAlert's fallback) carries alert_id and
// no inbox_id, and osascript holds the result of its desktop notification
// with the values of the inbox's notified event: ok, failed: <err>, or
// skipped (plan 2026-10-23-watch-escalation-inbox, D5 and F13). An inbox
// line keeps that result in the item's notified event instead.
type escalationRecord struct {
	TS        string `json:"ts"`
	OrgID     string `json:"org_id"`
	AlertID   string `json:"alert_id,omitempty"`
	InboxID   string `json:"inbox_id,omitempty"`
	Subject   string `json:"subject,omitempty"`
	Reason    string `json:"reason"`
	Osascript string `json:"osascript,omitempty"`
}

// conditionKey identifies one (org_id, seat_id, condition_type) dedupe slot
// (Codex advisory finding 1).
func conditionKey(orgID, seatID, condType string) string {
	return orgID + "/" + seatID + "/" + condType
}

// watchAlertID returns the alert_id of one firing of key's condition at now:
// key + "@" + now in Unix nanoseconds, "<org>/<seat>/<condition>@<n>". sendAlert
// names its pending alert with it, and recordRecurrence the recurred event of
// a firing it folds into an open inbox item. alertIDKey, alertIDConditionType,
// escalatedTimestamp, and alertIDSeat (escalate.go) read this shape, so a
// change to it changes them too.
func watchAlertID(key string, now time.Time) string {
	return fmt.Sprintf("%s@%d", key, now.UnixNano())
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
// and agmsg call of the pulse (runGitBounded, watch_scope.go), so a git that
// does not return cannot hold the cycle. An exit 0 whose pipe a child still
// held is a success, the way driver.ExecRunner reads it.
func (o *Org) realGitStatus(cwd string) (string, error) {
	return o.runGitBounded(cwd, "status", "--porcelain")
}

// realEscalate is the notification of a watch escalation the inbox could
// not take when WatchAllParams.Escalate is nil and desktopNotifyGOOS is
// darwin (notifyWithoutInbox, through notifyBounded): a macOS notification
// via osascript, titled after the watch. Its error is recorded in the
// escalation's escalations.jsonl line (osascript: failed: <err>), not
// dropped.
func realEscalate(ctx context.Context, message string) error {
	return osascriptNotify(ctx, inboxNotifyTitleWatch, message)
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
	gitRun          GitRunFunc // of an org with a reservation (evaluateReservedScope); nil uses realGitRun
	escalateFn      EscalateFunc
	stderr          io.Writer
	statusPath      string
	escalationsPath string
	// acceptsWork reports whether the watch running this cycle accepts work
	// (watchAllRun.acceptsWork), so that it will check the ack deadline of
	// an ALERT item it leaves to the director (escalateAlert; plan
	// 2026-10-23-watch-escalation-inbox, D3). nil means it does not.
	acceptsWork func() bool
}

// defaultWatchStopPollInterval is the tick of RunWatchAll's wait between
// cycles when WatchAllParams.StopPollInterval is unset: how often it looks
// at ctx and the stop request.
const defaultWatchStopPollInterval = time.Second

// WatchAllParams describes one `ralph org watch --all` run (see
// RunWatchAll).
type WatchAllParams struct {
	// StatusDir is the org state dir: watch.lock, watch-alive.json,
	// watch-stop.json, each org's watch-status-<org_id>.json,
	// escalations.jsonl, and the director's director.jsonl and
	// director-alive.json live in it (the directory of the manifest the Org
	// reads, in production). Required. The watch writes the escalations.jsonl
	// line of an escalation the inbox could not take to
	// EscalationsPathIn(StatusDir) (escalateWithoutInbox), but an
	// escalation the inbox takes is sent through the inbox's human path,
	// which writes its line to Org.EscalationsPath (sendInboxItemToHuman).
	// So a caller sets Org.EscalationsPath to EscalationsPathIn(StatusDir),
	// as the CLI does: with Org.EscalationsPath unset, each send of an ALERT
	// item to the human shows its banner and desktop notification but
	// writes no escalations.jsonl line and no notified event, so the ack
	// deadline sends it again while it is open.
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
	// GitStatus overrides the scope-change condition's git probe of an org
	// that holds no reservation; nil uses the real `git status --porcelain`
	// (realGitStatus).
	GitStatus GitStatusFunc
	// GitRun overrides the git of the scope-change condition of an org that
	// holds a reservation (evaluateReservedScope, watch_scope.go: the
	// porcelain, HEAD, the paths of the commits since the previous cycle,
	// and the merge base of the org's records); nil uses the real git
	// (realGitRun).
	GitRun GitRunFunc
	// Escalate is the desktop notification of an AC-5 watch escalation the
	// inbox could not take (escalateAlert's fallback); nil means osascript
	// on darwin (realEscalate) and none elsewhere, recorded as skipped. An
	// escalation the inbox takes is an ALERT item, notified through
	// Org.DesktopNotify like a leader's item (plan
	// 2026-10-23-watch-escalation-inbox, D9); StatusDir says where each
	// path writes its escalations.jsonl line.
	Escalate EscalateFunc
	// Stderr gets the AC-5 escalation banners, the inbox deadline banners,
	// the lines of the director's wake (WakeDirector), and one line for each
	// failure RunWatchAll goes past; nil uses os.Stderr.
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
	// DirectorWakeDir is the absolute directory [org.director].wake_cmd runs
	// in when the cycle wakes the director by push (WakeDirector; plan
	// 2026-10-18-org-director-wake, D9): the directory of the ralph.toml
	// wake_cmd was read from. The CLI sets it to the main worktree root when
	// wake_cmd was read by the main-worktree rule, to the directory of the
	// --config file, or else to the directory the watch started in, whose
	// ralph.toml it read. Empty means wake_cmd is never run: each wake by
	// wake_cmd is a failure line, while the herdr default (a headless
	// director in herdr with an empty wake_cmd) still works.
	DirectorWakeDir string
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
//     git, so no org's pulse can hold it back. The TYPE ALERT items it sends
//     get one desktop notification together, sent at the end of the sweep,
//     before step 2 (none with [org.watchdog].alert_notify = "off"; plan
//     2026-11-12-alert-notify-batch, D2), and every other item one each;
//  2. checks the current registration of director.jsonl when it is
//     headless and not stopped (CheckDirector; plan
//     2026-10-18-org-director-liveness, D4) and replaces director-alive.json
//     with the result (DirectorAlive). With nothing to check it writes
//     nothing and leaves the mark as it was. It writes nothing to
//     director.jsonl and closes, types, and signals nothing; it takes
//     director.lock only to read the registration again. It comes after
//     step 1, so a check that waits on herdr or ps does not hold back that
//     cycle's sends to the human, and, like step 3, it does not look at
//     Config.Director.Enabled;
//  3. wakes the attached director for the open inbox items it was not
//     woken for yet when its registration wakes by push (WakeDirector, with
//     step 2's result, one DirectorWakeMemory for the whole run, and
//     p.DirectorWakeDir; plan 2026-10-18-org-director-wake, D6), unless step
//     2 found that registration ended (plan
//     2026-10-18-org-director-liveness, D5). It comes after step 1, so a
//     wake that waits on wake_cmd or herdr does not hold back that cycle's
//     sends to the human, and it does not look at Config.Director.Enabled;
//  4. runs the pulse cycle (evaluateCycle) of each org in RunningOrgs of
//     the manifest, in that order, with each org's watch-status file read
//     once and then kept in memory across cycles. An org that no longer
//     runs (disbanded) is not evaluated and is dropped. The ALERT items an
//     org's deadman step sends to the human at once get one desktop
//     notification together at the end of that step (checkDeadman), so a
//     cycle sends at most one for the sweep and one for each org that
//     escalated;
//  5. rewrites the mark with the cycle's ts;
//  6. calls hooks.OnCycle.
//
// A failure in step 1, 2, 3, or 4 (an inbox or a director.jsonl that cannot
// be read or written, a director-alive.json that cannot be written, a wake
// that fails, the manifest, one org's status file) is one line on Stderr and
// skips only its own part: the other orgs, the mark, and the loop go on. A
// check that fails gives the wake no result, so the wake goes on as before
// the check existed. A status file that cannot be read is said once per org
// rather than every cycle (pulseOrg).
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
	// wakeDir and wakeMemory are WakeDirector's directory of wake_cmd
	// (WatchAllParams.DirectorWakeDir) and its memory across cycles (wake).
	wakeDir    string
	wakeMemory DirectorWakeMemory
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
	gitRun := p.GitRun
	if gitRun == nil {
		gitRun = o.realGitRun
	}
	stderr := p.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	deadman := o.Config.DeadmanMinutes
	w := &watchAllRun{
		org: o, dir: p.StatusDir, stderr: stderr,
		interval: interval, tick: tick, startedAt: startedAt,
		run: watchRun{
			org: o, cfg: o.Config, hooks: hooks,
			gitStatus: gitStatus, gitRun: gitRun, escalateFn: p.Escalate, stderr: stderr,
			escalationsPath: EscalationsPathIn(p.StatusDir),
		},
		statuses:         map[string]*watchStatusFile{},
		wakeDir:          p.DirectorWakeDir,
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
	w.run.acceptsWork = w.acceptsWork
	return w
}

// acceptsWork reports whether this run accepts work in the cycle that asks:
// it does not mark itself once (Cycles > 0), and no stop request for it is
// there (stopRequested). An ALERT item that a run which accepts no work
// left to the director could go unchecked until the next watch starts, so
// such a run sends its escalations at once (plan
// 2026-10-23-watch-escalation-inbox, D3).
func (w *watchAllRun) acceptsWork() bool {
	return !w.mark.Once && !w.stopRequested()
}

// cycle runs steps 1 to 5 of RunWatchAll's cycle: the inbox deadlines, the
// director check, the director's wake, the pulse, and the mark, in that order
// (plan 2026-10-18-org-director-wake, D6; plan
// 2026-10-18-org-director-liveness, D4 and D5).
func (w *watchAllRun) cycle() {
	sweep := w.org.SweepInboxDeadlines(w.org.nowTime(), &w.inboxMemory, w.stderr)
	if sweep.Err != nil {
		_, _ = fmt.Fprintf(w.stderr, "watchdog: inbox deadlines skipped this cycle: %v\n", sweep.Err)
	}
	for _, f := range sweep.Failed {
		_, _ = fmt.Fprintf(w.stderr, "watchdog: inbox item %s not sent for %s: %v\n", f.ID, f.Reason, f.Err)
	}
	check := w.checkDirector()
	w.wake(check)
	w.pulse()
	w.writeMark()
}

// checkDirector runs the director check of the cycle (CheckDirector on the
// ledger's director.jsonl, at the cycle's time; plan
// 2026-10-18-org-director-liveness, D4) and returns its result for the
// wake, or nil when nothing was checked. A check that fails (director.jsonl
// cannot be read, or director.lock cannot be taken for the read under it) is
// one line and returns nil. A result with Checked replaces
// director-alive.json (writeDirectorAlive, with the attached_at of the
// registration it checked); a write that fails is one line, and the result
// is still returned, since the wake goes by the check, not by the mark.
// Nothing is written when nothing was checked.
func (w *watchAllRun) checkDirector() *DirectorCheckResult {
	res, err := w.org.CheckDirector(NewDirectorStore(w.dir), w.org.nowTime())
	if err != nil {
		_, _ = fmt.Fprintf(w.stderr, "watchdog: director check skipped this cycle: %v\n", err)
		return nil
	}
	if !res.Checked {
		return nil
	}
	if err := writeDirectorAlive(w.dir, directorAliveOf(res)); err != nil {
		_, _ = fmt.Fprintf(w.stderr, "watchdog: director check mark not written: %v\n", err)
	}
	return &res
}

// wake runs the wake step of the cycle (WakeDirector) with check, the
// cycle's director check (nil for none). WakeDirector writes its own lines
// for a failed or skipped wake; an error that kept it from starting is one
// line here.
func (w *watchAllRun) wake(check *DirectorCheckResult) {
	res := w.org.WakeDirector(w.org.nowTime(), w.dir, w.wakeDir, check, &w.wakeMemory, w.stderr)
	if res.Err != nil {
		_, _ = fmt.Fprintf(w.stderr, "watchdog: director wake skipped this cycle: %v\n", res.Err)
	}
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

// evaluateCycle runs exactly one pulse-layer cycle: the escalation of any
// pending alert the inbox already holds (reconcilePendingAlertsWithInbox),
// per-active-seat checks (stall / liveness / scope-change), the org's
// usage_limited condition from the seats those checks found stopped on a
// usage limit (raiseUsageLimited; plan 2026-11-13-watch-usage-limited, D1),
// then the AC-5 deadman sweep over any still-pending alerts, which
// escalates a usage_limited ALERT raised in this cycle at once (D3), then
// persists status.
//
// Each herdr and agmsg call of the cycle, and the real git status, runs
// under its own Org.callWithTimeout deadline, and the desktop notification
// of an escalation under its own inboxDesktopNotifyTimeout (notifyBounded,
// for the inbox's human path, at the end of the deadman step or at once,
// or notifyWithoutInbox; escalateAlert), each
// cut from context.Background(), not from the watch loop's context (Codex
// plan advisory finding 1 of docs/plans/active/2026-10-11-org-cross-watch.md;
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

	// A pending alert the inbox already holds was escalated by a watch that
	// stopped before it saved this status file: it is escalated again (a
	// replay that records nothing new) and dropped before any seat is
	// evaluated, so it cannot take a new firing of its key as a recurrence
	// (reconcilePendingAlertsWithInbox).
	w.reconcilePendingAlertsWithInbox(status, now)

	// Every active seat but the leader is evaluated first, in Roster order,
	// and the leader last, so the leader's stall sees which of them were
	// busy this cycle (plan 2026-11-12-watch-alert-dedupe, D4).
	var seats, leaders []SeatStatus
	for _, s := range Roster(rr.Events, RosterOptions{}) {
		if s.OrgID != orgID || !s.Active {
			continue
		}
		if s.SeatID == LeaderIdentity {
			leaders = append(leaders, s)
		} else {
			seats = append(seats, s)
		}
	}

	cycle := &watchOrgCycle{}
	for _, s := range append(seats, leaders...) {
		w.evaluateSeat(status, orgID, s, now, rr.Events, cycle)
	}
	w.raiseUsageLimited(status, orgID, cycle.usageLimited, now)

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
		if retiredConditionNames[alertIDConditionType(alertID)] {
			delete(status.PendingAlerts, alertID)
		}
	}
	for alertID := range status.Escalated {
		if retiredConditionNames[alertIDConditionType(alertID)] {
			delete(status.Escalated, alertID)
		}
	}
}

// alertIDConditionType recovers the condition-type segment from an AlertID
// shaped "<org_id>/<seat_id>/<condition_type>@<unixnano>" (sendAlert's
// "%s@%d" format over conditionKey) -- the segment between the last "/" and
// the "@". Returns "" (never a retiredConditionNames match) if the id does
// not have that shape, which is the safe default: an unparseable id is left
// alone rather than guessed at. pruneRetiredConditions reads it to find the
// alerts of a retired condition, and watchEscalationBody for the CONDITION
// of an ALERT item's body. It checks only the separators; alertIDSeat
// (escalate.go) reads the seat of the same shape and checks all of it, and
// alertIDKey its conditionKey part, so a change to the shape
// (watchAlertID) changes all three.
func alertIDConditionType(alertID string) string {
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

// watchOrgCycle is what the seats of one org's pulse cycle share (plan
// 2026-11-12-watch-alert-dedupe, D3 and D4). evaluateCycle makes a new one
// for each org's cycle and passes it to each evaluateSeat call of that
// cycle; the watch evaluates orgs one after another, so nothing else reads
// or writes it.
type watchOrgCycle struct {
	// anotherSeatWorking is set once a seat other than the leader was busy
	// this cycle (seatBusy).
	anotherSeatWorking bool
	// leaderItemsRead is set once the inbox was read for leaderItemOpen,
	// which then holds whether the org has an item its leader raised that
	// is not resolved (orgHasOpenLeaderItem).
	leaderItemsRead bool
	leaderItemOpen  bool
	// usageLimited holds the seats whose screen showed them stopped on a
	// usage limit this cycle, in the order they were evaluated
	// (evaluateSeat adds the entry readUsageLimitScreen returns);
	// evaluateCycle raises the org's usage_limited condition from it after
	// the last seat (plan 2026-11-13-watch-usage-limited, D1).
	usageLimited []watchUsageLimitedSeat
}

// leaderIsWaiting reports whether the leader of orgID is waiting this
// cycle, so that its stall is not raised (plan
// 2026-11-12-watch-alert-dedupe, D3 and D4): a seat other than the leader
// was busy this cycle, or the org has an item its leader raised that is
// not resolved. evaluateSeat calls it only for a leader that would
// otherwise stall, so the inbox is read only then, and at most once in the
// cycle.
func (w *watchRun) leaderIsWaiting(cycle *watchOrgCycle, orgID string) bool {
	if cycle.anotherSeatWorking {
		return true
	}
	if !cycle.leaderItemsRead {
		cycle.leaderItemsRead = true
		cycle.leaderItemOpen = w.orgHasOpenLeaderItem(orgID)
	}
	return cycle.leaderItemOpen
}

// orgHasOpenLeaderItem reports whether the inbox holds an item of orgID of
// a TYPE its leader raises with `ralph org escalate` (escalateTypes:
// QUESTION, BLOCKED, RESULT) that is not resolved (open or acked): the
// leader is waiting on a person or the director (plan
// 2026-11-12-watch-alert-dedupe, D3). With no inbox store, or an inbox that
// cannot be read, it reports false, so the leader's stall is raised as
// before (the deadline sweep at the start of the cycle already says on
// stderr that the inbox cannot be read).
func (w *watchRun) orgHasOpenLeaderItem(orgID string) bool {
	if w.org.Inbox == nil {
		return false
	}
	inbox, err := w.org.Inbox.Read()
	if err != nil {
		return false
	}
	for _, item := range inbox.Items {
		if item.OrgID == orgID && item.State != InboxStateResolved && slices.Contains(escalateTypes, item.Type) {
			return true
		}
	}
	return false
}

// seatBusy reports whether out, a seat's `herdr agent get` reply this
// cycle, shows the seat working within the cycle (plan
// 2026-11-12-watch-alert-dedupe, D4): snap holds a reply of the previous
// cycle, out differs from it, and out's agent_status is working
// (herdrAgentStatus). A reply that does not parse is not busy, and neither
// is a working reply that did not change, as herdr reports for a seat
// stuck in working (issue #308).
func seatBusy(snap *watchSeatSnapshot, out string) bool {
	if !snap.AgentGetSeen || snap.AgentGet == out {
		return false
	}
	agentStatus, err := herdrAgentStatus(out)
	return err == nil && agentStatus == "working"
}

// evaluateSeat runs every per-seat pulse condition for s: (c) heartbeat
// stall, (d) process liveness, (e) worktree scope change. events is the
// cycle's manifest snapshot (see evaluateCycle), needed by the stall
// condition's M-6 fix below. cycle is the org's cycle state: a seat other
// than the leader that is busy (seatBusy) sets its anotherSeatWorking, and
// the leader's stall is not raised while the leader is waiting
// (leaderIsWaiting), so evaluateCycle evaluates the leader last. A seat,
// the leader included, whose stall holds before that exemption and whose
// screen shows it stopped on a usage limit (readUsageLimitScreen) is added
// to the cycle's usageLimited and gets no stall (plan
// 2026-11-13-watch-usage-limited, D4); when its screen could not be read,
// its stall ALERT says why (usageLimitUnreadNote).
func (w *watchRun) evaluateSeat(status *watchStatusFile, orgID string, s SeatStatus, now time.Time, events []ManifestEvent, cycle *watchOrgCycle) {
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
				orgID, s.SeatID, condLiveness, s.SeatID, agentGetErr), false, "")
	} else {
		w.raiseOrClear(status, orgID, s.SeatID, condLiveness, false, now, "", false, "")

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
		//
		// The leader's stall is not raised while the leader is waiting on a
		// busy seat or on an item it raised (leaderIsWaiting; plan
		// 2026-11-12-watch-alert-dedupe, D3 and D4): the condition clears
		// as if the leader had moved. Other seats' stall is unchanged.
		//
		// Before that exemption, a seat whose stall holds has its screen
		// read once; one stopped on a usage limit gets no stall and goes
		// into the org's usage_limited instead (plan
		// 2026-11-13-watch-usage-limited, D4 and AC11), so a leader stopped
		// on one is reported however the exemption would read it. A screen
		// that could not be read leaves the stall as it is, and the stall
		// ALERT says why (usageLimitUnreadNote).
		lastEventTS := latestSeatEventTS(events, orgID, s.SeatID)
		stalled := snap.AgentGetSeen && snap.AgentGet == agentGetOut && isStallByTime(lastEventTS, now, w.cfg.Watchdog.StallMinutes)
		var unread error
		if stalled {
			seat, limited, err := w.readUsageLimitScreen(s)
			if limited {
				cycle.usageLimited = append(cycle.usageLimited, seat)
				stalled = false
			}
			unread = err
		}
		if s.SeatID == LeaderIdentity {
			stalled = stalled && !w.leaderIsWaiting(cycle, orgID)
		} else if seatBusy(snap, agentGetOut) {
			cycle.anotherSeatWorking = true
		}
		w.raiseOrClear(status, orgID, s.SeatID, condStall, stalled, now,
			fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: %s\n\nseat %s heartbeat stalled for over %dm%s",
				orgID, s.SeatID, condStall, s.SeatID, w.cfg.Watchdog.StallMinutes, usageLimitUnreadNote(unread)), true, "")
		snap.AgentGet = agentGetOut
		snap.AgentGetSeen = true
	}

	// (e) scope change: seat worktree `git status --porcelain` differs from
	// the previous cycle's snapshot. No cutoff -- ALERT + semantic trigger
	// only (free-text scope, AC-4/plan Non-goals). In an org that holds a
	// reservation, only a change outside it counts, the org's own new plan
	// and reports aside (evaluateReservedScope; plan
	// 2026-11-12-watch-alert-dedupe, D5 and D6).
	if s.Worktree != "" {
		if reserved := ActiveReservation(events, orgID); len(reserved) > 0 {
			w.evaluateReservedScope(status, orgID, s.SeatID, s.Worktree, reserved, snap, now)
			return
		}
		out, err := w.gitStatus(s.Worktree)
		if err == nil {
			changed := snap.GitStatusSeen && snap.GitStatus != out
			w.raiseOrClear(status, orgID, s.SeatID, condScopeChange, changed, now,
				fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: %s\n\nseat %s worktree scope changed:\n%s",
					orgID, s.SeatID, condScopeChange, s.SeatID, truncateScopeOutput(out, scopeChangeBodyBudget)), true, "")
			snap.GitStatus = out
			snap.GitStatusSeen = true
		}
	}
}

// raiseOrClear implements the AC-3c idempotent ALERT dedupe for a
// condition: an active==false->true transition records the key as Active
// and sends exactly one ALERT; the key clears (Active: false) the first
// cycle active is observed false again, so a later re-occurrence is a new
// rising edge. Such an edge sends a new ALERT only when the key's earlier
// ALERT has ended (plan 2026-11-12-watch-alert-dedupe, D1): while a pending
// alert of the key (one the leader has not acted on since it was sent) or
// an ALERT item of the key that is not resolved is there, the edge is
// recorded as a recurrence of it (recordRecurrence, with
// recurNote as the note of an inbox item's recurred event), and no ALERT is
// sent to the leader, no pending alert is made, and OnSemanticTrigger is
// not fired. The key is still recorded as Active, so the cycles after the
// edge are deduped as before. semantic, when true and an ALERT is sent,
// also fires WatchHooks.OnSemanticTrigger (the Slice 4 on-demand-watcher
// seam) -- liveness intentionally passes semantic=false: a dead/unreachable
// pane gives a watcher nothing useful to judge.
func (w *watchRun) raiseOrClear(status *watchStatusFile, orgID, seatID, condType string, active bool, now time.Time, message string, semantic bool, recurNote string) {
	key := conditionKey(orgID, seatID, condType)
	rec := status.Conditions[key]
	if active {
		if rec != nil && rec.Active {
			return // already alerted and still active: dedupe
		}
		status.Conditions[key] = &watchConditionRecord{Active: true, FirstTS: now.UTC().Format(time.RFC3339)}
		if w.recordRecurrence(status, orgID, seatID, condType, now, recurNote) {
			return // the key's earlier ALERT is still open: a recurrence of it
		}
		w.sendAlert(status, orgID, seatID, condType, message, now)
		if semantic && w.hooks.OnSemanticTrigger != nil {
			w.hooks.OnSemanticTrigger(orgID, seatID, condType, message)
		}
		return
	}
	if rec != nil && rec.Active {
		rec.Active = false // recovered: clears, a future re-occurrence is a new rising edge
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
// cannot be reached at all. It returns that pending alert, to which
// raiseUsageLimited adds the seats of its org condition.
func (w *watchRun) sendAlert(status *watchStatusFile, orgID, seatID, condType, message string, now time.Time) *watchPendingAlert {
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
	alertID := watchAlertID(conditionKey(orgID, seatID, condType), now)
	pending := &watchPendingAlert{
		AlertID:            alertID,
		TS:                 now.UTC().Format(time.RFC3339),
		Subject:            seatID,
		ManifestLen:        leaderActivityEventCount(rr.Events, orgID),
		LeaderAgentGet:     w.leaderProbeSnapshot(orgID),
		HistoryLeaderLines: w.historyLeaderLineCount(orgID),
	}
	status.PendingAlerts[alertID] = pending
	return pending
}

// leaderActivityEventCount counts manifest events attributable to leader for
// orgID (self-review M-4 fix, org-scoped per cross-review AR-1; the seat-
// attribution model itself was corrected by self-review cycle-3 H3-1 --
// see below): a genuinely unresponsive leader must not have its deadman
// escalation silently cleared by an unrelated seat's own manifest traffic,
// so an event only counts here when (a), (b) or (c) holds:
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
//	    has. Of a spawn's own writes only spawn_started counts, not
//	    spawn_failed or the non-state spawn_close_failed, so a spawn whose
//	    step failed after spawn_started counts once, whether or not it closed
//	    its pane. A spawn refused before spawn_started counts only by the
//	    `rejected` it writes, and these refusals write none, so they do not
//	    count: a seat in flight refused by the envelope, permission or
//	    --scope check (rejectUnlessInFlight), the seat's spawn lock held by
//	    another spawn, and compensateStale's refusals (the caller's own pane,
//	    which writes nothing, and a pane neither confirmed nor closed, which
//	    writes only spawn_close_failed). A leader that only retries into
//	    those refusals is stuck, so the deadman may go on to escalate it as an
//	    ALERT item of the inbox (plan 2026-10-22-seat-spawn-close,
//	    self-review L2). The exclusion applies to any event in this
//	    lifecycle set whose Details carry
//	    "reason=watchdog_..." -- in practice only the pre-#152
//	    watchdog's cutoff `stopped` writes ever carried it.
//	    No code path has produced such an event since PR #152 removed the
//	    org budget concept (2026-09-17): StopParams no longer has a Reason
//	    field and no pulse-layer condition calls Stop. Why the guard is
//	    kept: this function's two production call sites are exactly a
//	    baseline/recount pair -- sendAlert records the ManifestLen snapshot
//	    and leaderActedSinceAlert recounts against it -- so the count is only ever
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
//	(c) ev.Event is EventGatePassed or EventGateFailed. `ralph org gate`
//	    and `ralph org regate` write them. The leader template has the
//	    reviewer seat run a feature org's gate at the leader's TASK (plan
//	    2026-10-26-reviewer-runs-gate). The gate starts from the leader's
//	    `sent` TASK, so its record counts as a sign that the work the
//	    leader asked for moved on. Regate writes one only for an org that
//	    does not run: it refuses a running org at its first check and again
//	    under the manifest lock right before it records (regateFirstCheck,
//	    recordRegate), and while it runs a spawn or start of that org is
//	    refused (regateSpawnErr). The watch pulses only running orgs, so a
//	    regate record never lands while the watch counts that org's
//	    activity; one written between two lives of the org follows the stop
//	    or disband that ended the earlier one, which counts too, so the
//	    count stays the leader's. Of the codex seats, only the leader and,
//	    in a feature org, the reviewer get --add-dir and so can write the
//	    ledger from a cwd outside it (codexWritableRootArgs, permissions.go);
//	    a claude seat has no such sandbox, so a claude seat of any role that
//	    ran the gate would be counted as leader activity too. The count is
//	    kept as it was when the leader ran the gate itself: a pending alert
//	    stores the count as its baseline (watchPendingAlert.ManifestLen) in
//	    the status file, and leaderActedSinceAlert compares the count with it across
//	    watch restarts, so changing what counts would shift that baseline
//	    across an upgrade or a rollback. A side effect is that a reviewer's
//	    gate record can clear a pending alert about a seat other than the
//	    leader. An ALERT whose subject is the leader escalates whatever the
//	    activity (checkDeadman clears an alert only when there is activity
//	    and the subject is not the leader, !subjectIsLeader), so a stalled
//	    leader is not missed. A gate records nothing while its scripts run,
//	    so it counts when it ends. The watchdog exclusion of (b) does not
//	    apply: no gate record is a watchdog write. ralph releases before
//	    `ralph org gate` never write them, so the stop_failed exception
//	    above applies the same way: an older `ralph org watch` saves a
//	    pending alert whose baseline lacks a gate record a newer gate
//	    wrote, and restarting watch on the newer binary while that alert is
//	    pending reads one higher and clears it. An older watch that keeps
//	    running does not count them at all. The CI gate records
//	    (EventGateCIPassed, EventGateCIFailed) are not counted: only
//	    ralph org regate writes them, so never while the watch counts that
//	    org's activity, and leaving them out keeps the count, and so a
//	    pending alert's baseline, the same across an upgrade or a rollback.
//	    Nor is gate_accepted (EventGateAccepted) counted, which falls to no
//	    case below: a human, or the director a human told to, writes it with
//	    `ralph org gate accept`, not the leader, so it is no sign that the
//	    leader acts, and ralph releases before it never write it, so
//	    counting it would move a pending alert's baseline across an upgrade
//	    or a rollback, as counting the CI gate records would (plan
//	    2026-11-12-gate-lock-accept, D6).
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
		case EventGatePassed, EventGateFailed:
			n++
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
// see leaderActedSinceAlert) as activity does not have that failure mode: eviction
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
// self-report while it is the thing that is anomalous). An ALERT of the org
// condition usage_limited has the leader as its subject (the seat of its
// key), so it takes that branch too and is escalated in the cycle that
// raised it (plan 2026-11-13-watch-usage-limited, D3).
//
// Whether the leader acted since the ALERT was sent is leaderActedSinceAlert,
// the test foldIntoPendingAlert (watch_recurrence.go) shares; the rule for
// the probe-based sources is in its doc.
//
// The step is one sending step of the watch (alert_notify.go; plan
// 2026-11-12-alert-notify-batch, D2): the ALERT items its escalations send
// to the human at once each get their escalations.jsonl line and banner at
// once, and at the end of the step one desktop notification together (none
// with [org.watchdog].alert_notify = "off") and each its notified event
// (flushDeadmanAlertNotifications).
func (w *watchRun) checkDeadman(status *watchStatusFile, rr ManifestReadResult, now time.Time) {
	batch := &alertNotifyBatch{}
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
		activity := w.leaderActedSinceAlert(status.OrgID, pending, rr.Events)

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
			w.escalateAlert(status, alertID, pending, now, batch)
		}
	}
	w.flushDeadmanAlertNotifications(batch)
}

// flushDeadmanAlertNotifications ends the deadman step's sending step:
// it flushes batch (flushAlertNotifyBatch, which sends the one desktop
// notification and appends the notified events) and reports the failed
// appends: for each ALERT item of batch whose notified event could not be
// appended, the stderr line sendAlertItemToHuman writes for a send that was
// not completed.
func (w *watchRun) flushDeadmanAlertNotifications(batch *alertNotifyBatch) {
	for _, r := range w.org.flushAlertNotifyBatch(batch) {
		if r.err != nil {
			w.alertSendNotCompleted(r.orgID, r.id, r.err)
		}
	}
}

// leaderActedSinceAlert reports whether the leader of orgID shows activity
// since pending was sent, by the 3 information sources: a count of the
// leader's manifest events in events (leaderActivityEventCount) beyond
// pending.ManifestLen, the leader's herdr `agent get` text
// (leaderProbeSnapshot) differing from a valid baseline, or the count of the
// leader's agmsg history lines (historyLeaderLineCount) beyond a valid
// baseline. It asks a probe only when the sources before it show nothing.
// checkDeadman clears a pending alert whose subject is not the leader when
// it holds, and foldIntoPendingAlert does not fold a rising edge into such
// an alert when it holds, but ends that alert's episode (plan
// 2026-11-12-watch-alert-dedupe, D1), so the two cannot disagree about when
// the leader has answered an ALERT.
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
func (w *watchRun) leaderActedSinceAlert(orgID string, pending *watchPendingAlert, events []ManifestEvent) bool {
	if leaderActivityEventCount(events, orgID) > pending.ManifestLen {
		return true
	}
	if pending.LeaderAgentGet != "" {
		if cur := w.leaderProbeSnapshot(orgID); cur != "" && cur != pending.LeaderAgentGet {
			return true
		}
	}
	if pending.HistoryLeaderLines >= 0 {
		if cur := w.historyLeaderLineCount(orgID); cur >= 0 && cur > pending.HistoryLeaderLines {
			return true
		}
	}
	return false
}

// escalateAlert escalates one pending ALERT to a person (AC-5), deduped by
// alertID ("one escalation per alert, ever") via the
// delete(status.PendingAlerts, alertID) below, which removes alertID from
// the set checkDeadman ranges over whatever became of the escalation, so
// this same alert can never be escalated a second time by this watch.
// status.Escalated is written here too, but only as a bounded historical
// audit trail (see checkDeadman's own comment) -- it has no remaining
// reader that dedupe depends on.
//
// The reason is watchEscalationReason's: usage_limit for the org condition
// usage_limited, leader_is_anomaly_subject for another ALERT about the
// leader, and deadman_timeout otherwise. The escalation becomes a TYPE
// ALERT item of the org inbox (escalateToInbox; plan
// 2026-10-23-watch-escalation-inbox, D1), which reaches a person the way a
// leader's escalate does: inboxDelivery, with Config.Director.Enabled and
// whether this watch accepts work (watchRun.acceptsWork), either leaves it
// to the director and the inbox deadlines or sends it to the human path at
// once. An alert the inbox already holds as an item or as a recurred event
// of one (a watch that ended after recording it and before saving its
// status file) gets no second record (D4); evaluateCycle escalates such an
// alert before it evaluates the seats (reconcilePendingAlertsWithInbox).
// An alert whose (org, seat, condition) has an ALERT item that is not
// resolved gets no new item either: the escalation is a recurrence of that
// item (plan 2026-11-12-watch-alert-dedupe, D2). When the inbox cannot take
// the item (no inbox store, an inbox that cannot be read or written, or
// values of the ALERT in the watch-status file that the watch would not
// have written), the escalation takes the path it took before the inbox
// held ALERT items (escalateWithoutInbox, D5).
//
// batch is the sending step the escalation belongs to: checkDeadman's, so
// an item sent to the human at once is notified with the others of that
// step at its end, or nil (reconcilePendingAlertsWithInbox), so it is
// notified on its own (plan 2026-11-12-alert-notify-batch, D2). The path
// without the inbox never takes a batch.
func (w *watchRun) escalateAlert(status *watchStatusFile, alertID string, pending *watchPendingAlert, now time.Time, batch *alertNotifyBatch) {
	reason := watchEscalationReason(alertID, pending.Subject)
	if err := w.escalateToInbox(status.OrgID, alertID, pending, reason, batch); err != nil {
		w.escalateWithoutInbox(status.OrgID, alertID, pending, reason, now, err)
	}

	status.Escalated[alertID] = true
	pruneEscalated(status.Escalated)
	delete(status.PendingAlerts, alertID)
}

// watchEscalationReason returns the reason escalateAlert gives the
// escalation of alertID, whose ALERT is about subject, by the alert's
// condition first (alertIDConditionType; plan
// 2026-11-13-watch-usage-limited, D3 and AC10): usage_limit for the org
// condition usage_limited, whose subject is the leader only as the seat of
// its key; then by the subject: leader_is_anomaly_subject for an ALERT
// about the leader, and deadman_timeout for any other.
func watchEscalationReason(alertID, subject string) string {
	switch {
	case isUsageLimitedAlert(alertID):
		return usageLimitReason
	case subject == LeaderIdentity:
		return "leader_is_anomaly_subject"
	}
	return "deadman_timeout"
}

// escalateToInbox records the escalation of alertID as a TYPE ALERT item of
// orgID, with the delivery inboxDelivery decides, and writes on stderr what
// it did. With delivery InboxDeliveryDeadline the item is only recorded: no
// escalations.jsonl line and no desktop notification, and one stderr line
// says it was left to the director (plan 2026-10-23-watch-escalation-inbox,
// AC2). With InboxDeliveryNow it is sent to the human path at once
// (sendAlertItemToHuman; plan 2026-10-23-watch-escalation-inbox, AC3), whose
// escalations.jsonl line goes to Org.EscalationsPath, the file the CLI also
// gives the watch. An alert that is already an item is handled by
// escalateRecordedItem. An alert whose alert_id is already on a recurred
// event of an item (alertItemWithRecurredID), in any state, resolved
// included, was already recorded as a recurrence of that item by a watch
// that ended before it saved its status file: as for an item of the same
// alert_id, nothing new is recorded or sent, and one stderr line names the
// item (plan 2026-11-12-watch-alert-dedupe, D2).
//
// When the inbox holds no item with alertID but an ALERT item of orgID
// whose alert_id has the same (org, seat, condition) (alertIDKey) and that
// is not resolved (openAlertItemForKey), the escalation is recorded as a
// recurrence of that item instead of a new item (plan
// 2026-11-12-watch-alert-dedupe, D2: a watch that lost the episode from its
// status file, or one of an earlier ralph that made the alert, does not add
// an item): one recurred event with alertID and a count of 1 plus the
// alert's Recurrences, one stderr line naming the item, and no send to the
// human; the item keeps its own deadlines. When that append fails because
// the director resolved the item meanwhile (ErrInboxResolved), or for any
// other reason (one stderr line), a new item is recorded as below. An alert
// of the org condition usage_limited is never recorded that way: the watch
// made it only after raiseUsageLimited found that its edge was not a
// recurrence of such an item (an acked item, say, which must not take seats
// that stopped since it was acked; plan 2026-11-13-watch-usage-limited,
// AC4), so it is always a new item.
//
// A new item of an alert that recurred while it was pending (Recurrences >
// 0) gets one recurred event with alertID and that count right after it is
// recorded (AC4 of that plan); a failure there is one stderr line, and the
// item stays.
//
// It returns an error, having written nothing to the inbox (and nothing to
// stderr but the line of a recurrence that was not recorded), only when the
// inbox cannot take the item: the ALERT's values from the
// watch-status file are not ones the watch writes
// (checkWatchEscalationValues), Org.Inbox is nil, the inbox cannot be read,
// the body fails the protocol check (which the body watchEscalationBody
// builds passes), or the item cannot be recorded. Once the item is recorded,
// or the escalation is recorded as a recurrence, it returns nil. A send to
// the human goes into batch (escalateAlert).
func (w *watchRun) escalateToInbox(orgID, alertID string, pending *watchPendingAlert, reason string, batch *alertNotifyBatch) error {
	if err := checkWatchEscalationValues(alertID, pending); err != nil {
		return err
	}
	if w.org.Inbox == nil {
		return errors.New("no inbox store is configured")
	}
	inbox, err := w.org.Inbox.Read()
	if err != nil {
		return err
	}
	delivery, sendReason := inboxDelivery(w.cfg.Director.Enabled, w.acceptsWork)
	head := escalationHead(orgID, alertID, pending.Subject, reason)
	if i := slices.IndexFunc(inbox.Items, func(item InboxItem) bool { return item.AlertID == alertID }); i >= 0 {
		w.escalateRecordedItem(orgID, alertID, inbox.Items[i], head, delivery, sendReason, batch)
		return nil
	}
	if item, ok := alertItemWithRecurredID(inbox.Items, alertID); ok {
		_, _ = fmt.Fprintf(w.stderr, "%s -- already in the inbox as a recurrence on %s (state=%s); nothing new is recorded or sent; read it with: %s\n",
			head, item.ID, item.State, InboxCommand("show", item.ID, w.org.InboxHintStateDir))
		return nil
	}
	if item, ok := openAlertItemForKey(inbox.Items, orgID, alertIDKey(alertID)); ok && !isUsageLimitedAlert(alertID) && w.escalateAsRecurrence(alertID, pending, item, head) {
		return nil
	}
	body := watchEscalationBody(orgID, alertID, pending, reason, w.cfg.DeadmanMinutes)
	if err := protocol.ValidateText(body, protocol.DefaultMaxBodyChars); err != nil {
		return fmt.Errorf("the body of the ALERT item fails protocol validation: %w", err)
	}
	id, err := w.org.Inbox.Escalate(InboxEscalation{OrgID: orgID, Type: protocol.TypeAlert, Body: body, Delivery: delivery, AlertID: alertID})
	if err != nil {
		return err
	}
	if pending.Recurrences > 0 {
		if err := w.org.Inbox.AppendRecurred(id, alertID, "", pending.Recurrences); err != nil {
			_, _ = fmt.Fprintf(w.stderr, "%s item=%s -- recorded, but not the %d recurrences the alert had while it was pending: %v\n",
				head, id, pending.Recurrences, err)
		}
	}
	if delivery == InboxDeliveryDeadline {
		_, _ = fmt.Fprintf(w.stderr, "%s item=%s -- left in %s for the director; "+
			"if it is still open after [org.director].ack_minutes, or not resolved after [org.director].resolve_minutes, ralph org watch --all sends it to the human; read it with: %s\n",
			head, id, w.org.Inbox.Path(), InboxCommand("show", id, w.org.InboxHintStateDir))
		return nil
	}
	w.sendAlertItemToHuman(orgID, id, alertID, sendReason, batch)
	return nil
}

// checkWatchEscalationValues returns an error when a value of the pending
// ALERT that watchEscalationBody puts into the item's headers is not one
// sendAlert writes: a subject that is not a seat_id (ValidateIdentifier),
// an alertID that alertIDSeat does not read, or a control character in
// alertID or in the ALERT's ts. The values come from the watch-status file,
// which a hand edit can change, and the body is read by the director, so a
// line break in one of them must not add a header (self-review F4 of plan
// 2026-10-23-watch-escalation-inbox). escalateAlert then takes the path
// without the inbox, whose banner escapes them (escalationHead) and names
// the error as its cause, so the error holds no value unescaped: only
// ValidateIdentifier's names the subject, quoted with %q. An alert of the
// org condition usage_limited is also checked for the seats it lists
// (checkUsageLimitedSeats), which go into the SEATS header.
func checkWatchEscalationValues(alertID string, pending *watchPendingAlert) error {
	if err := ValidateIdentifier("seat_id", pending.Subject); err != nil {
		return fmt.Errorf("the ALERT's subject in the watch-status file is not a seat: %w", err)
	}
	if alertIDSeat(alertID) == "" || strings.ContainsFunc(alertID, unicode.IsControl) {
		return errors.New("the alert_id in the watch-status file does not have the shape <org>/<seat>/<condition>@<n> the watch writes")
	}
	if strings.ContainsFunc(pending.TS, unicode.IsControl) {
		return errors.New("the ALERT's ts in the watch-status file holds a control character")
	}
	if isUsageLimitedAlert(alertID) {
		return checkUsageLimitedSeats(pending.limitedSeatEntries())
	}
	return nil
}

// escalateRecordedItem is escalateToInbox for an alert the inbox already
// holds as item, one with its alert_id: a watch that recorded the item and
// ended before it saved its status file left the alert pending, so the next
// watch escalates it again (plan 2026-10-23-watch-escalation-inbox, D4). It
// writes no second item. As for a new item, only this watch's decision
// counts, whatever delivery the item was recorded with: it sends item to
// the human path, under its own ID and with sendReason, only when the item
// is open with no notified event and this watch cannot leave it to the
// director (watchDelivery, which inboxDelivery decided for this watch, is
// InboxDeliveryNow, so sendReason is set). Otherwise it sends nothing: a
// notified or resolved item needs no send, an acked one is left to the
// resolve deadline, and an open one with no notified event to the director
// and the ack deadline. Either way one stderr line names the item, with the
// item's own state, notified, and delivery (item.Delivery, the delivery it
// was recorded with, not watchDelivery). The send goes into batch
// (escalateAlert).
func (w *watchRun) escalateRecordedItem(orgID, alertID string, item InboxItem, head, watchDelivery, sendReason string, batch *alertNotifyBatch) {
	state := fmt.Sprintf("state=%s notified=%t delivery=%s", item.State, item.Notified, item.Delivery)
	if item.State != InboxStateOpen || item.Notified || watchDelivery != InboxDeliveryNow {
		_, _ = fmt.Fprintf(w.stderr, "%s -- already in the inbox as %s (%s); nothing new is recorded or sent; read it with: %s\n",
			head, item.ID, state, InboxCommand("show", item.ID, w.org.InboxHintStateDir))
		return
	}
	_, _ = fmt.Fprintf(w.stderr, "%s -- already in the inbox as %s (%s); sending that item to the human\n", head, item.ID, state)
	w.sendAlertItemToHuman(orgID, item.ID, alertID, sendReason, batch)
}

// sendAlertItemToHuman sends ALERT item id to the human path with reason
// (sendInboxItemToHuman, which writes its banner on stderr and records the
// notification's result in the item's notified event). With a batch, the
// desktop notification and the notified event wait for the end of the
// step (flushDeadmanAlertNotifications); with none, they come at once. A
// send that is not completed, here or at the end of the step, is one stderr
// line (alertSendNotCompleted).
func (w *watchRun) sendAlertItemToHuman(orgID, id, alertID, reason string, batch *alertNotifyBatch) {
	if err := w.org.sendInboxItemToHuman(orgID, id, protocol.TypeAlert, alertID, reason, w.stderr, batch); err != nil {
		w.alertSendNotCompleted(orgID, id, err)
	}
}

// alertSendNotCompleted writes the stderr line of a send of ALERT item id to
// the human path that was not completed (err): it names `ralph org inbox
// notify <id>`, and the item has no notified event, so the ack deadline
// sends it too while it is open (plan 2026-10-23-watch-escalation-inbox,
// AC3).
func (w *watchRun) alertSendNotCompleted(orgID, id string, err error) {
	_, _ = fmt.Fprintf(w.stderr, "WATCHDOG ESCALATION: org=%s item=%s -- recorded, but the send to the human was not completed (%v): run `%s` to send it again; "+
		"until then it has no notified event, so ralph org watch --all also sends it after [org.director].ack_minutes while it is open\n",
		PrintableInboxText(orgID, false), id, err, InboxCommand("notify", id, w.org.InboxHintStateDir))
}

// escalationHead is the start of the stderr lines of escalateAlert:
// WATCHDOG ESCALATION, the org, the alert, its subject, and the reason. The
// alert and the subject come from the watch-status file, so they are
// escaped like the inbox banners (PrintableInboxText).
func escalationHead(orgID, alertID, subject, reason string) string {
	return fmt.Sprintf("WATCHDOG ESCALATION: org=%s alert=%s subject=%s reason=%s",
		PrintableInboxText(orgID, false), PrintableInboxText(alertID, false), PrintableInboxText(subject, false), reason)
}

// watchEscalationBody is the body of the ALERT item escalateAlert records
// (plan 2026-10-23-watch-escalation-inbox, AC1): a typed message with the
// headers of the watch's ALERT (sendAlert) -- ORG_ID, SEAT (the ALERT's
// subject), and CONDITION (read from alertID) -- then REASON, ALERT_ID, and
// ALERT_TS (the time of the ALERT), and one line saying why the watch
// escalated it. escalateToInbox checks the values from the watch-status
// file first (checkWatchEscalationValues), so none of them adds a header.
// The body of an alert of the org condition usage_limited is chosen by that
// condition first: usageLimitedEscalationBody's, with its SEATS header and
// a line for each seat (plan 2026-11-13-watch-usage-limited, D2, D3, and
// AC10).
func watchEscalationBody(orgID, alertID string, pending *watchPendingAlert, reason string, deadmanMinutes int) string {
	if isUsageLimitedAlert(alertID) {
		return usageLimitedEscalationBody(orgID, alertID, pending)
	}
	why := fmt.Sprintf("the leader showed no activity within %d minutes (deadman_minutes) of the ALERT", deadmanMinutes)
	if reason == "leader_is_anomaly_subject" {
		why = "the ALERT is about the leader itself, so the watch did not wait for the leader to act"
	}
	return fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: %s\nREASON: %s\nALERT_ID: %s\nALERT_TS: %s\n\nwatchdog: escalated to a person because %s",
		orgID, pending.Subject, alertIDConditionType(alertID), reason, alertID, pending.TS, why)
}

// escalateWithoutInbox is escalateAlert's path when the inbox cannot take
// the escalation (cause), the path every escalation took before the inbox
// held ALERT items (plan 2026-10-23-watch-escalation-inbox, D5), in this
// order: the stderr banner, which names the cause and, for an alert of the
// org condition usage_limited, the seats that stopped
// (usageLimitedBannerSeats); the desktop notification (notifyWithoutInbox);
// and one escalations.jsonl line with alert_id, subject (pending's), reason,
// and the notification's result as osascript (F13). A line that cannot be
// written is one more stderr line, which carries that result as
// notification=<result>, so the result is not lost with the line
// (self-review F1 of plan 2026-10-23-watch-escalation-inbox).
func (w *watchRun) escalateWithoutInbox(orgID, alertID string, pending *watchPendingAlert, reason string, now time.Time, cause error) {
	_, _ = fmt.Fprintf(w.stderr, "%s%s -- see %s (not recorded in the inbox: %v)\n",
		escalationHead(orgID, alertID, pending.Subject, reason), usageLimitedBannerSeats(alertID, pending), w.escalationsPath, cause)
	osascript := w.notifyWithoutInbox(fmt.Sprintf("ralph org watch: escalation for org %s (%s)", orgID, reason))
	rec := escalationRecord{
		TS: now.UTC().Format(time.RFC3339), OrgID: orgID, AlertID: alertID,
		Subject: pending.Subject, Reason: reason, Osascript: osascript,
	}
	if err := appendJSONLine(w.escalationsPath, rec); err != nil {
		_, _ = fmt.Fprintf(w.stderr, "watchdog: escalation of alert %s not recorded: %v; notification=%s\n",
			PrintableInboxText(alertID, false), err, PrintableInboxText(osascript, false))
	}
}

// notifyWithoutInbox sends the desktop notification of an escalation the
// inbox could not take with w.escalateFn (WatchAllParams.Escalate) through
// notifyBounded, as the inbox's human path sends its own
// (inboxDesktopNotify), and returns the result with the values of the
// inbox's notified event: ok, failed: <err>, or skipped. A nil escalateFn
// is realEscalate on darwin and skipped elsewhere (desktopNotifyGOOS). The
// call gets its own bound from context.Background(): an osascript that does
// not return cannot hold the cycle, and a SIGINT or SIGTERM in the middle of
// the cycle does not cut off the notification of the escalation, which is
// never sent again (self-review L2).
func (w *watchRun) notifyWithoutInbox(message string) string {
	return notifyBounded(w.escalateFn, realEscalate, message)
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
