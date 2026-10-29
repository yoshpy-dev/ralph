package driver

import "strings"

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
// changes it changes no other caller's entries. A seat's pane also gets one
// entry of its own after these, PrimaryCLIEntry of its driver, which spawn
// passes to TabCreate as extraEnv; a headless director's pane and launch
// script do not.
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
// value the pane takes from the herdr server's environment wins over it.
const PrimaryCLIEnv = "RALPH_PRIMARY_CLI"

// PrimaryCLIEntry returns the entry PrimaryCLIEnv=driverName, which spawn
// (internal/org/spawn.go) adds to the pane of every seat, leaders included,
// as TabCreate's extraEnv, after the Herdr.PaneEnv entries: unlike those, its
// value depends on the seat. driverName is used as it is: spawn validates the
// seat's driver (permissionArgsForDriver) before it creates the pane, and
// accepts only claude and codex, the values pick_reviewer knows (plan
// 2026-10-29-review-switches-config, D5).
func PrimaryCLIEntry(driverName string) string {
	return PrimaryCLIEnv + "=" + driverName
}

// paneEnvConsequence is the clause warnNoPaneEnv writes for entry, saying
// what a pane created without it may do, or "" for an entry this driver does
// not know. Every PrimaryCLIEnv entry, whatever its value, has one clause.
func paneEnvConsequence(entry string) string {
	switch {
	case entry == AgmsgSelfNameOff:
		return "an agmsg run inside such a pane (a seat's or a headless director's) may rename the pane's herdr agent, " +
			"and ralph may then fail to reach it (agent_not_found)"
	case entry == WIPCheckpointOff:
		return "a Claude Code session in such a pane may commit a feature branch's uncommitted changes as 'wip:' " +
			"at its session end or context compaction (its WIP-checkpoint hooks)"
	case strings.HasPrefix(entry, PrimaryCLIEnv+"="):
		return "a seat's cross-review may pick the wrong CLI as its other model " +
			"(scripts/xreview-helpers.sh pick_reviewer treats the seat's driver as claude), " +
			"and so may a feature org leader's plan advisory"
	}
	return ""
}
