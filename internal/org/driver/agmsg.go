package driver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Agmsg adapts the real agmsg interface via a Runner. agmsg v1.1.13
// (locally verified) is NOT a single binary with flags -- it is a
// collection of shell scripts under an "agmsg home" directory (default
// ~/.agents/skills/agmsg). Every method below shells out to
// `bash <home>/scripts/<script> <positional args...>`, matching the real
// script signatures exactly, with AGMSG_SELF_NAME=off added to the script's
// environment when R implements EnvRunner (AgmsgSelfNameOff).
type Agmsg struct {
	R Runner
	// Home is the resolved agmsg home directory (see ResolveAgmsgHome).
	// Callers must resolve Home before constructing Agmsg; the adapter
	// itself does no env/config lookups so its argv-building stays a pure
	// function of its fields, keeping stub tests simple.
	Home string
}

// AgmsgSelfNameEnv is the variable that turns agmsg's naming of the herdr
// agent and pane it runs in on or off (see AgmsgSelfNameOff).
const AgmsgSelfNameEnv = "AGMSG_SELF_NAME"

// AgmsgSelfNameOff is added to the environment of every agmsg script ralph
// runs (Agmsg.run), winning over a value ralph's own environment has.
// agmsg 1.5.1 otherwise names the herdr agent and pane it runs in, and that
// rename loses the <org_id>_<seat_id> name ralph addresses a seat's agent
// by (finding F3; docs/evidence/agmsg-self-name-2026-10-22.md). It is also
// the first entry of SeatPaneEnv, which the CLI's wiring passes as
// Herdr.PaneEnv, so the panes ralph creates carry it for the agmsg a seat
// runs itself, and which the launch script of a headless director
// (internal/org/director_cmd.go) exports.
const AgmsgSelfNameOff = AgmsgSelfNameEnv + "=off"

// run shells out to `bash <home>/scripts/<script> <args...>` via the
// injected Runner, through RunEnv with AgmsgSelfNameOff when the Runner
// implements EnvRunner and through Run otherwise. Centralizing the
// script-path join here means every method's stub test asserts the same
// "bash <home>/scripts/<script>" prefix, catching any future drift in one
// place.
func (a Agmsg) run(ctx context.Context, script string, args ...string) (string, error) {
	scriptPath := filepath.Join(a.Home, "scripts", script)
	full := append([]string{scriptPath}, args...)
	if er, ok := a.R.(EnvRunner); ok {
		return er.RunEnv(ctx, []string{AgmsgSelfNameOff}, "bash", full...)
	}
	return a.R.Run(ctx, "bash", full...)
}

// Send runs `bash <home>/scripts/send.sh TEAM FROM TO MESSAGE`.
func (a Agmsg) Send(ctx context.Context, team, from, to, message string) error {
	_, err := a.run(ctx, "send.sh", team, from, to, message)
	return err
}

// Join runs `bash <home>/scripts/join.sh TEAM AGENT_ID TYPE PROJECT_PATH`.
// agmsgType is the agmsg-native agent kind (e.g. "claude-code", "codex",
// "gemini"), distinct from ralph's own driver names -- callers map
// ralph driver -> agmsg type before calling Join.
func (a Agmsg) Join(ctx context.Context, team, agentID, agmsgType, projectPath string) error {
	_, err := a.run(ctx, "join.sh", team, agentID, agmsgType, projectPath)
	return err
}

// TeamMembers runs `bash <home>/scripts/team.sh TEAM`.
func (a Agmsg) TeamMembers(ctx context.Context, team string) (string, error) {
	return a.run(ctx, "team.sh", team)
}

// History runs `bash <home>/scripts/history.sh TEAM [AGENT_ID [LIMIT]]`.
// agentID is omitted (along with limit) when empty; limit is only appended
// when agentID is present and limit > 0, matching the script's positional
// argument order -- there is no way to pass limit without agentID.
func (a Agmsg) History(ctx context.Context, team, agentID string, limit int) (string, error) {
	args := []string{team}
	if agentID != "" {
		args = append(args, agentID)
		if limit > 0 {
			args = append(args, strconv.Itoa(limit))
		}
	}
	return a.run(ctx, "history.sh", args...)
}

// Leave runs `bash <home>/scripts/leave.sh TEAM AGENT_ID`. This is the
// correct roster-removal verb for a member that joined via join.sh (see the
// doc comment on AgmsgClient.Leave in spawn.go for why despawn.sh does not
// work here: it only targets agmsg-spawned processes with placement
// records, which join.sh'd members never have).
func (a Agmsg) Leave(ctx context.Context, team, agentID string) error {
	_, err := a.run(ctx, "leave.sh", team, agentID)
	return err
}

// Whoami runs `bash <home>/scripts/whoami.sh PROJECT_PATH [TYPE]`. agmsgType
// is omitted when empty.
func (a Agmsg) Whoami(ctx context.Context, projectPath, agmsgType string) (string, error) {
	args := []string{projectPath}
	if agmsgType != "" {
		args = append(args, agmsgType)
	}
	return a.run(ctx, "whoami.sh", args...)
}

// validDeliveryModes are the only agmsg delivery modes accepted by
// DeliverySet.
var validDeliveryModes = map[string]bool{
	"monitor": true,
	"turn":    true,
	"both":    true,
	"off":     true,
}

// DeliverySet runs
// `bash <home>/scripts/delivery.sh set MODE TYPE PROJECT_PATH`. mode must be
// one of monitor|turn|both|off; invalid modes are rejected before any Run
// call so a typo never reaches the external process.
func (a Agmsg) DeliverySet(ctx context.Context, mode, agmsgType, projectPath string) error {
	if !validDeliveryModes[mode] {
		return fmt.Errorf("agmsg: invalid delivery mode %q (want monitor|turn|both|off)", mode)
	}
	_, err := a.run(ctx, "delivery.sh", "set", mode, agmsgType, projectPath)
	return err
}

// defaultAgmsgHome is used when neither RALPH_ORG_AGMSG_HOME nor
// [org].agmsg_home supply a value. It must stay in lock-step with
// config.Default().Org.AgmsgHome, templates/base/ralph.toml, and
// scripts/ralph-config.sh (see internal/config/defaults_sync_test.go).
const defaultAgmsgHome = "~/.agents/skills/agmsg"

// AgmsgHomeEnv is the variable ResolveAgmsgHome reads before the
// [org].agmsg_home config value. The commands that pass an agmsg home on to
// another process (the CLI's detached watch, the launch script of a
// headless director) set it under this name.
const AgmsgHomeEnv = "RALPH_ORG_AGMSG_HOME"

// AgmsgHomeOverride is the agmsg home ResolveAgmsgHome returns before
// anything else when it is not empty; empty means unset. Production code
// never sets it. Only tests do: the TestMains of internal/cli and
// internal/org pin it to a path that does not exist, and their helpers
// point it at a stub agmsg, so that no test reaches the developer's real
// agmsg whatever the environment or a ralph.toml says. It lives here, not
// in the CLI, because internal/org calls ResolveAgmsgHome("") itself
// (prompts.go). Like a variable a test sets with t.Setenv, a test that sets
// it must not run in parallel.
var AgmsgHomeOverride string

// ResolveAgmsgHome resolves the agmsg home directory used to build script
// paths. Precedence: AgmsgHomeOverride (tests only) > env
// RALPH_ORG_AGMSG_HOME (AgmsgHomeEnv) > cfgValue (the loaded
// [org].agmsg_home config value) > defaultAgmsgHome. A leading "~/" is
// expanded via os.UserHomeDir; other paths are returned unchanged.
func ResolveAgmsgHome(cfgValue string) string {
	home := AgmsgHomeOverride
	if home == "" {
		home = os.Getenv(AgmsgHomeEnv)
	}
	if home == "" {
		home = cfgValue
	}
	if home == "" {
		home = defaultAgmsgHome
	}
	return expandTilde(home)
}

// expandTilde expands a leading "~/" to the current user's home directory
// (via os.UserHomeDir). Paths that don't start with "~/" -- including a bare
// "~" -- are returned unchanged.
func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(homeDir, strings.TrimPrefix(path, "~/"))
}

// AgmsgVersion reads and trims <home>/VERSION, the plain-text version marker
// agmsg ships alongside its scripts/ directory.
func AgmsgVersion(home string) (string, error) {
	data, err := os.ReadFile(filepath.Join(home, "VERSION"))
	if err != nil {
		return "", fmt.Errorf("agmsg: reading VERSION: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}
