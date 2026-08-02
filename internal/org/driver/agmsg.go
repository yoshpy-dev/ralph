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
// script signatures exactly.
type Agmsg struct {
	R Runner
	// Home is the resolved agmsg home directory (see ResolveAgmsgHome).
	// Callers must resolve Home before constructing Agmsg; the adapter
	// itself does no env/config lookups so its argv-building stays a pure
	// function of its fields, keeping stub tests simple.
	Home string
}

// run shells out to `bash <home>/scripts/<script> <args...>` via the
// injected Runner. Centralizing the script-path join here means every
// method's stub test asserts the same "bash <home>/scripts/<script>"
// prefix, catching any future drift in one place.
func (a Agmsg) run(ctx context.Context, script string, args ...string) (string, error) {
	scriptPath := filepath.Join(a.Home, "scripts", script)
	full := append([]string{scriptPath}, args...)
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

// ResolveAgmsgHome resolves the agmsg home directory used to build script
// paths. Precedence: env RALPH_ORG_AGMSG_HOME > cfgValue (the loaded
// [org].agmsg_home config value) > defaultAgmsgHome. A leading "~/" is
// expanded via os.UserHomeDir; other paths are returned unchanged.
func ResolveAgmsgHome(cfgValue string) string {
	home := os.Getenv("RALPH_ORG_AGMSG_HOME")
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
