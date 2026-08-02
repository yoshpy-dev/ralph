package driver

import (
	"context"
	"strconv"
)

// Herdr adapts the herdr CLI (herdr.dev/docs/cli-reference) via a Runner, so
// the org verb layer (internal/org) never shells out directly. Argv shapes
// below mirror the confirmed CLI reference 1:1, one method per subcommand --
// do not invent subcommands beyond what herdr documents. Seat termination
// (send-keys based) is implemented by the caller, not this adapter: see
// (*Org).Stop in internal/org/verbs.go, which calls PaneSendKeys.
type Herdr struct {
	R Runner
}

// WorkspaceCreate runs `herdr workspace create --cwd CWD --label LABEL` and
// returns the created workspace id (trimmed stdout).
func (h Herdr) WorkspaceCreate(ctx context.Context, cwd, label string) (string, error) {
	return h.R.Run(ctx, "herdr", "workspace", "create", "--cwd", cwd, "--label", label)
}

// TabCreate runs `herdr tab create --workspace ID --cwd CWD --label LABEL`.
func (h Herdr) TabCreate(ctx context.Context, workspaceID, cwd, label string) (string, error) {
	return h.R.Run(ctx, "herdr", "tab", "create", "--workspace", workspaceID, "--cwd", cwd, "--label", label)
}

// AgentStart runs
// `herdr agent start NAME --kind KIND --pane PANEID [--timeout MS] [-- agentArgs...]`.
// timeoutMS <= 0 omits --timeout. agentArgs, when non-empty, are appended
// after a literal `--` separator so herdr passes them through unparsed to
// the underlying agent CLI; the separator is omitted entirely when
// agentArgs is empty.
func (h Herdr) AgentStart(ctx context.Context, name, kind, paneID string, timeoutMS int, agentArgs []string) (string, error) {
	args := []string{"agent", "start", name, "--kind", kind, "--pane", paneID}
	if timeoutMS > 0 {
		args = append(args, "--timeout", strconv.Itoa(timeoutMS))
	}
	if len(agentArgs) > 0 {
		args = append(args, "--")
		args = append(args, agentArgs...)
	}
	return h.R.Run(ctx, "herdr", args...)
}

// AgentGet runs `herdr agent get TARGET`.
func (h Herdr) AgentGet(ctx context.Context, target string) (string, error) {
	return h.R.Run(ctx, "herdr", "agent", "get", target)
}

// AgentWait runs
// `herdr agent wait TARGET --until U1 [--until U2...] [--timeout MS]`.
// timeoutMS <= 0 omits --timeout.
func (h Herdr) AgentWait(ctx context.Context, target string, until []string, timeoutMS int) (string, error) {
	args := []string{"agent", "wait", target}
	for _, u := range until {
		args = append(args, "--until", u)
	}
	if timeoutMS > 0 {
		args = append(args, "--timeout", strconv.Itoa(timeoutMS))
	}
	return h.R.Run(ctx, "herdr", args...)
}

// PaneRead runs `herdr pane read PANEID --source recent --lines N --format text`.
func (h Herdr) PaneRead(ctx context.Context, paneID string, lines int) (string, error) {
	return h.R.Run(ctx, "herdr", "pane", "read", paneID, "--source", "recent", "--lines", strconv.Itoa(lines), "--format", "text")
}

// PaneSendText runs `herdr pane send-text PANEID TEXT`.
func (h Herdr) PaneSendText(ctx context.Context, paneID, text string) error {
	_, err := h.R.Run(ctx, "herdr", "pane", "send-text", paneID, text)
	return err
}

// PaneSendKeys runs `herdr pane send-keys PANEID KEYS...`.
func (h Herdr) PaneSendKeys(ctx context.Context, paneID string, keys ...string) error {
	args := append([]string{"pane", "send-keys", paneID}, keys...)
	_, err := h.R.Run(ctx, "herdr", args...)
	return err
}

// PaneRun runs `herdr pane run PANEID COMMAND`.
func (h Herdr) PaneRun(ctx context.Context, paneID, command string) error {
	_, err := h.R.Run(ctx, "herdr", "pane", "run", paneID, command)
	return err
}
