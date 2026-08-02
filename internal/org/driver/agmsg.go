package driver

import (
	"context"
	"fmt"
	"strconv"
)

// Agmsg adapts the agmsg CLI via a Runner.
//
// ASSUMPTION (not fully confirmed against the real CLI -- see plan
// docs/plans/active/2026-08-01-org-runtime-mechanism.md Open questions):
// the agmsg CLI's exact flag shape for team/identity selection. This
// package centralizes that assumption in a single place, agmsgArgs below,
// so if the real CLI's shape differs, only that one function needs to
// change. Validate agmsgArgs against the real agmsg CLI before first live
// use.
type Agmsg struct {
	R Runner
}

// agmsgArgs builds `--team TEAM --as FROM <verb...>`. See the package-level
// ASSUMPTION note above -- this is the single point of centralization for
// agmsg's team/identity flag shape.
func agmsgArgs(team, from string, verb ...string) []string {
	args := []string{"--team", team, "--as", from}
	return append(args, verb...)
}

// Send runs `agmsg --team TEAM --as FROM send TO MESSAGE`.
func (a Agmsg) Send(ctx context.Context, team, from, to, message string) error {
	_, err := a.R.Run(ctx, "agmsg", agmsgArgs(team, from, "send", to, message)...)
	return err
}

// History runs `agmsg --team TEAM --as FROM history [--limit N]`. limit <= 0
// omits --limit.
func (a Agmsg) History(ctx context.Context, team, from string, limit int) (string, error) {
	verb := []string{"history"}
	if limit > 0 {
		verb = append(verb, "--limit", strconv.Itoa(limit))
	}
	return a.R.Run(ctx, "agmsg", agmsgArgs(team, from, verb...)...)
}

// TeamMembers runs `agmsg --team TEAM --as FROM team`.
func (a Agmsg) TeamMembers(ctx context.Context, team, from string) (string, error) {
	return a.R.Run(ctx, "agmsg", agmsgArgs(team, from, "team")...)
}

// validModes are the only agmsg mode values accepted by Mode.
var validModes = map[string]bool{
	"monitor": true,
	"turn":    true,
	"both":    true,
	"off":     true,
}

// Mode runs `agmsg --team TEAM --as FROM mode MODE`. mode must be one of
// monitor|turn|both|off; invalid modes are rejected before any Run call so
// a typo never reaches the external process.
func (a Agmsg) Mode(ctx context.Context, team, from, mode string) error {
	if !validModes[mode] {
		return fmt.Errorf("agmsg: invalid mode %q (want monitor|turn|both|off)", mode)
	}
	_, err := a.R.Run(ctx, "agmsg", agmsgArgs(team, from, "mode", mode)...)
	return err
}
