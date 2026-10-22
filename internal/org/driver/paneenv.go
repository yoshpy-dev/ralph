package driver

// WIPCheckpointEnv is the variable the Claude Code WIP-checkpoint hooks
// (.claude/hooks/session_end_summary.sh and precompact_checkpoint.sh) read:
// when it is exactly "off" they write their summary but make no wip: commit
// (see WIPCheckpointOff).
const WIPCheckpointEnv = "RALPH_WIP_CHECKPOINT"

// WIPCheckpointOff is an entry of SeatPaneEnv, so a Claude Code session in a
// pane ralph creates does not commit a feature branch's uncommitted changes
// as wip: when it ends or compacts its context, in a feature worktree the
// seats of an org share (finding F11; plan 2026-10-22-seat-wip-checkpoint).
const WIPCheckpointOff = WIPCheckpointEnv + "=off"

// SeatPaneEnv returns the KEY=VALUE entries ralph adds to the environment of
// every pane it creates, in this order: AgmsgSelfNameOff, WIPCheckpointOff.
// The CLI passes them to NewHerdr, so they are the Herdr.PaneEnv of the
// panes of seats and of a headless director, and the launch script of a
// headless director started with launch_cmd (internal/org/director_cmd.go)
// exports each of them. Every call returns a new slice, so a caller that
// changes it changes no other caller's entries.
func SeatPaneEnv() []string {
	return []string{AgmsgSelfNameOff, WIPCheckpointOff}
}

// paneEnvConsequence is the clause warnNoPaneEnv writes for entry, saying
// what a pane created without it may do, or "" for an entry this driver does
// not know.
func paneEnvConsequence(entry string) string {
	switch entry {
	case AgmsgSelfNameOff:
		return "an agmsg run inside such a pane (a seat's or a headless director's) may rename the pane's herdr agent, " +
			"and ralph may then fail to reach it (agent_not_found)"
	case WIPCheckpointOff:
		return "a Claude Code session in such a pane may commit a feature branch's uncommitted changes as 'wip:' " +
			"at its session end or context compaction (its WIP-checkpoint hooks)"
	}
	return ""
}
