package driver

import (
	"math"
	"strconv"
	"strings"
)

// WIPCheckpointEnv is the variable the Claude Code WIP-checkpoint hooks
// (.claude/hooks/session_end_summary.sh and precompact_checkpoint.sh) read:
// when it is exactly "off" they write their summary but make no wip: commit
// (see WIPCheckpointOff).
const WIPCheckpointEnv = "RALPH_WIP_CHECKPOINT"

// WIPCheckpointOff is an entry of SeatPaneEnv. With it, a Claude Code
// session in a pane ralph creates makes no wip: commit of its worktree's
// uncommitted changes when the session ends or compacts its context. In a
// feature org, a seat's worktree is the feature worktree that all the seats
// of the org share, so such a commit could take in another seat's work
// (finding F11; plan 2026-10-22-seat-wip-checkpoint).
const WIPCheckpointOff = WIPCheckpointEnv + "=off"

// SeatPaneEnv returns the KEY=VALUE entries ralph adds to the environment of
// every pane it creates, in this order: AgmsgSelfNameOff, WIPCheckpointOff.
// Despite its name it is not only for seats: the CLI passes the entries to
// NewHerdr, so they are the Herdr.PaneEnv of the panes of seats (leaders
// included) and of a headless director in herdr, and the launch script of a
// headless director started with launch_cmd (internal/org/director_cmd.go)
// exports each of them. Every call returns a new slice, so a caller that
// changes it changes no other caller's entries. A seat's pane also gets
// entries of its own after these, which spawn passes as extraEnv to the call
// that makes the pane (WorkspaceCreate, PaneSplit, or TabCreate):
// PrimaryCLIEntry of its driver, and then, for a claude seat only, the
// entries of ClaudeRetryEnv for [org].api_wait_minutes. A headless
// director's pane and launch script get no PrimaryCLIEntry; a claude one
// gets the ClaudeRetryEnv entries after these, and a codex one nothing more.
func SeatPaneEnv() []string {
	return []string{AgmsgSelfNameOff, WIPCheckpointOff}
}

// PrimaryCLIEnv is the variable /plan's plan advisory (its step 11.b) and
// /cross-review (its step 2) read as the CLI a session runs on, its driver
// (claude or codex, case-insensitive): it overrides the CLI running the
// skill, and the skill passes the driver to scripts/xreview-helpers.sh's
// pick_reviewer to call the other one as the reviewer. pick_reviewer called
// without a driver reads it too, and treats the driver as claude and picks
// codex when it is unset. A feature org leader's plan advisory (the 計画助言
// subsection of the 機能ごとの org section of internal/org/prompts/leader.md)
// reads it the same way, and uses the leader's own CLI only when the
// variable is unset, empty, or neither claude nor codex, so a claude or codex
// value the pane takes from the herdr server's environment wins over it. A
// reviewer seat's cross-review does not read it: it passes pick_reviewer the
// implementer's driver that its TASK names (plan 2026-11-01-org-cross-review,
// D6).
const PrimaryCLIEnv = "RALPH_PRIMARY_CLI"

// PrimaryCLIEntry returns the entry PrimaryCLIEnv=driverName, which spawn
// (internal/org/spawn.go) adds to the pane of every seat, leaders included,
// as the extraEnv of the call that makes the pane (WorkspaceCreate,
// PaneSplit, or TabCreate), after the Herdr.PaneEnv entries and before a
// claude seat's ClaudeRetryEnv entries: unlike the Herdr.PaneEnv entries, its
// value depends on the seat. driverName is used as it is: spawn validates the
// seat's driver (permissionArgsForDriver) before it creates the pane, and
// accepts only claude and codex, the values pick_reviewer knows (plan
// 2026-10-29-review-switches-config, D5).
func PrimaryCLIEntry(driverName string) string {
	return PrimaryCLIEnv + "=" + driverName
}

// The variables of ClaudeRetryEnv (spec docs/specs/2026-11-07-org-seat-resume.md,
// FR-8). ClaudeRetryWatchdogEnv=1 makes Claude Code keep retrying a rate or
// usage limit (429/529) and a connection failure instead of ending the turn
// after its default retries; ClaudeRetryWatchdogMaxWaitMSEnv bounds its wait
// on a rate or usage limit, in milliseconds; and ClaudeMaxRetriesEnv bounds
// its retries of a connection failure, by count.
const (
	ClaudeRetryWatchdogEnv          = "CLAUDE_CODE_RETRY_WATCHDOG"
	ClaudeRetryWatchdogMaxWaitMSEnv = "CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS"
	ClaudeMaxRetriesEnv             = "CLAUDE_CODE_MAX_RETRIES"
)

// ClaudeMaxRetriesForMinutes is the CLAUDE_CODE_MAX_RETRIES that makes a
// Claude Code session retry a connection failure for about minutes m:
// max(1, 10 + round((m*60 - 189) / 36)), rounded half away from zero
// (math.Round), so 1 gives 6, 2 gives 8, 3 gives 10, 30 gives 55, and 60
// gives 105. The constants come from
// docs/evidence/org-seat-resume-retries-2026-11-07.md, measured on Claude
// Code 2.1.312 against a name-resolution failure (ENOTFOUND): the first 10
// retries took about 189 seconds and each later one about 36. Below 3
// minutes the count ends inside the first retries, whose waits grow, so the
// time is rougher there. A connection that never answers can take longer:
// each retry may wait close to API_TIMEOUT_MS (10 minutes by default) for
// the response, so the total can be well past m minutes. A rate or usage
// limit (429/529) is not bounded by this count but by the
// ClaudeRetryWatchdogMaxWaitMSEnv entry of ClaudeRetryEnv. Measure again
// after a large Claude Code update: the retry intervals may change between
// versions.
func ClaudeMaxRetriesForMinutes(m int) int {
	return max(1, 10+int(math.Round((float64(m)*60-189)/36)))
}

// ClaudeRetryEnv returns the KEY=VALUE entries ralph adds to the environment
// of a claude seat's pane (leaders included), of a claude headless
// director's pane, and to the launch script of a claude headless director
// started with launch_cmd, for [org].api_wait_minutes minutes, in this
// order: ClaudeRetryWatchdogEnv=1, ClaudeRetryWatchdogMaxWaitMSEnv=minutes
// times 60000, and ClaudeMaxRetriesEnv=ClaudeMaxRetriesForMinutes(minutes).
// minutes below 1 returns nil: no entry, as before the setting existed.
// Codex seats and a codex director get none of them (FR-9), so a `claude -p`
// run inside codex does not wait long. Every call returns a new slice. The
// values are no secret: they appear in the herdr argv, which ps shows.
func ClaudeRetryEnv(minutes int) []string {
	if minutes <= 0 {
		return nil
	}
	return []string{
		ClaudeRetryWatchdogEnv + "=1",
		ClaudeRetryWatchdogMaxWaitMSEnv + "=" + strconv.Itoa(minutes*60000),
		ClaudeMaxRetriesEnv + "=" + strconv.Itoa(ClaudeMaxRetriesForMinutes(minutes)),
	}
}

// paneEnvConsequence is the clause warnNoPaneEnv writes for entry, saying
// what a pane created without it may do, or "" for an entry this driver does
// not know. Every PrimaryCLIEnv entry, whatever its value, has one clause, and
// so does every entry of each variable of ClaudeRetryEnv: the watchdog's says
// what the session does with none of the three, the other two what it does
// when it takes the watchdog from the herdr server's environment.
func paneEnvConsequence(entry string) string {
	switch {
	case entry == AgmsgSelfNameOff:
		return "an agmsg run inside such a pane (a seat's or a headless director's) may rename the pane's herdr agent and the pane, " +
			"and ralph may then fail to reach it (agent_not_found) and, no longer finding the label it gave the pane, " +
			"leave the pane open at a stop for you to close by hand"
	case entry == WIPCheckpointOff:
		return "a Claude Code session in such a pane may commit a feature branch's uncommitted changes as 'wip:' " +
			"at its session end or context compaction (its WIP-checkpoint hooks)"
	case strings.HasPrefix(entry, PrimaryCLIEnv+"="):
		return "a feature org leader's plan advisory in such a pane may pick the wrong CLI as its other model " +
			"if the pane takes a claude or codex RALPH_PRIMARY_CLI from the herdr server's environment " +
			"(scripts/xreview-helpers.sh pick_reviewer then sees that value instead of the leader's driver; " +
			"with none, the leader names its own CLI)"
	case strings.HasPrefix(entry, ClaudeRetryWatchdogEnv+"="):
		return "a Claude Code session in such a pane (a claude seat's or a claude headless director's) ends its turn " +
			"on a rate or usage limit (429/529) or a connection failure after its default retries " +
			"(about 3 minutes for a connection failure) instead of waiting out up to [org].api_wait_minutes"
	case strings.HasPrefix(entry, ClaudeRetryWatchdogMaxWaitMSEnv+"="):
		return "a Claude Code session in such a pane that takes " + ClaudeRetryWatchdogEnv + "=1 from the herdr server's environment " +
			"does not stop waiting on a rate or usage limit (429/529) after [org].api_wait_minutes"
	case strings.HasPrefix(entry, ClaudeMaxRetriesEnv+"="):
		return "a Claude Code session in such a pane that takes " + ClaudeRetryWatchdogEnv + "=1 from the herdr server's environment " +
			"does not stop retrying a connection failure after about [org].api_wait_minutes"
	}
	return ""
}
