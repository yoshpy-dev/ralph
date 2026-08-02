package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Herdr adapts the herdr CLI (herdr.dev/docs/cli-reference) via a Runner, so
// the org verb layer (internal/org) never shells out directly. Argv shapes
// below mirror the confirmed CLI reference 1:1, one method per subcommand --
// do not invent subcommands beyond what herdr documents. Seat termination
// (send-keys based) is implemented by the caller, not this adapter: see
// (*Org).Stop in internal/org/verbs.go, which calls PaneSendKeys.
//
// Real herdr (confirmed live, v0.7.5) wraps every command's stdout in a JSON
// envelope -- {"id":"cli:<verb>","result":{...}} on success or
// {"error":{"code":"...","message":"..."},"id":"cli:<verb>"} on failure --
// except `pane read`, which is plain text. parseHerdrEnvelope below extracts
// the `result` payload (or a structured error) from that envelope; when
// stdout is not JSON at all (e.g. `pane read`, or a bare id emitted by unit
// test fakes), callers fall back to the trimmed raw string unchanged.
type Herdr struct {
	R Runner
}

// herdrEnvelope mirrors the JSON wrapper every herdr CLI command emits on
// stdout. Exactly one of Result/Error is populated.
type herdrEnvelope struct {
	Result json.RawMessage `json:"result"`
	Error  *herdrError     `json:"error"`
}

// herdrError is the structured error payload inside a herdr envelope.
type herdrError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *herdrError) Error() string {
	return fmt.Sprintf("herdr: %s: %s", e.Code, e.Message)
}

// parseHerdrEnvelope parses trimmed herdr CLI stdout as a JSON envelope.
//
//   - Success envelope ({"result":...}): returns (result, nil, true).
//   - Error envelope ({"error":...}): returns (nil, *herdrError, true).
//   - Not JSON at all (out does not start with `{`): returns
//     (nil, nil, false) -- "not an envelope", callers fall back to the
//     trimmed raw string. This keeps `pane read` (plain text) and
//     unit-test fakes (bare ids like "ws-123") working unchanged.
//   - `{`-prefixed but malformed JSON: returns (nil, err, true) rather than
//     falling back, so a truncated/corrupt envelope never gets mistaken for
//     a bare ID and fed downstream as e.g. a workspace_id.
func parseHerdrEnvelope(out string) (result json.RawMessage, envErr error, isEnvelope bool) {
	trimmed := strings.TrimSpace(out)
	if !strings.HasPrefix(trimmed, "{") {
		return nil, nil, false
	}
	var env herdrEnvelope
	if err := json.Unmarshal([]byte(trimmed), &env); err != nil {
		return nil, fmt.Errorf("herdr: malformed JSON envelope: %w", err), true
	}
	if env.Error != nil {
		return nil, env.Error, true
	}
	return env.Result, nil, true
}

// WorkspaceCreate runs `herdr workspace create --cwd CWD --label LABEL` and
// returns the created workspace id, extracted from the envelope's
// result.workspace.workspace_id. Non-JSON output (unit-test fakes) falls
// back to the trimmed raw stdout.
func (h Herdr) WorkspaceCreate(ctx context.Context, cwd, label string) (string, error) {
	out, err := h.R.Run(ctx, "herdr", "workspace", "create", "--cwd", cwd, "--label", label)
	if err != nil {
		return out, err
	}
	result, envErr, isEnvelope := parseHerdrEnvelope(out)
	if !isEnvelope {
		return strings.TrimSpace(out), nil
	}
	if envErr != nil {
		return "", fmt.Errorf("herdr workspace create: %w", envErr)
	}
	var payload struct {
		Workspace struct {
			WorkspaceID string `json:"workspace_id"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return "", fmt.Errorf("herdr workspace create: parse result: %w", err)
	}
	if payload.Workspace.WorkspaceID == "" {
		return "", fmt.Errorf("herdr workspace create: envelope missing result.workspace.workspace_id")
	}
	return payload.Workspace.WorkspaceID, nil
}

// TabCreate runs `herdr tab create --workspace ID --cwd CWD --label LABEL`
// and returns the created pane id, extracted from the envelope's
// result.root_pane.pane_id. Non-JSON output (unit-test fakes) falls back to
// the trimmed raw stdout.
func (h Herdr) TabCreate(ctx context.Context, workspaceID, cwd, label string) (string, error) {
	out, err := h.R.Run(ctx, "herdr", "tab", "create", "--workspace", workspaceID, "--cwd", cwd, "--label", label)
	if err != nil {
		return out, err
	}
	result, envErr, isEnvelope := parseHerdrEnvelope(out)
	if !isEnvelope {
		return strings.TrimSpace(out), nil
	}
	if envErr != nil {
		return "", fmt.Errorf("herdr tab create: %w", envErr)
	}
	var payload struct {
		RootPane struct {
			PaneID string `json:"pane_id"`
		} `json:"root_pane"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return "", fmt.Errorf("herdr tab create: parse result: %w", err)
	}
	if payload.RootPane.PaneID == "" {
		return "", fmt.Errorf("herdr tab create: envelope missing result.root_pane.pane_id")
	}
	return payload.RootPane.PaneID, nil
}

// AgentStart runs
// `herdr agent start NAME --kind KIND --pane PANEID [--timeout MS] [-- agentArgs...]`.
// timeoutMS <= 0 omits --timeout. agentArgs, when non-empty, are appended
// after a literal `--` separator so herdr passes them through unparsed to
// the underlying agent CLI; the separator is omitted entirely when
// agentArgs is empty. The return value is informational raw text (see
// checkHerdrEnvelopeError); it is not parsed into a struct.
func (h Herdr) AgentStart(ctx context.Context, name, kind, paneID string, timeoutMS int, agentArgs []string) (string, error) {
	args := []string{"agent", "start", name, "--kind", kind, "--pane", paneID}
	if timeoutMS > 0 {
		args = append(args, "--timeout", strconv.Itoa(timeoutMS))
	}
	if len(agentArgs) > 0 {
		args = append(args, "--")
		args = append(args, agentArgs...)
	}
	out, err := h.R.Run(ctx, "herdr", args...)
	return checkHerdrEnvelopeError(out, err)
}

// AgentGet runs `herdr agent get TARGET`. The return value is informational
// raw text (see checkHerdrEnvelopeError); it is not parsed into a struct.
func (h Herdr) AgentGet(ctx context.Context, target string) (string, error) {
	out, err := h.R.Run(ctx, "herdr", "agent", "get", target)
	return checkHerdrEnvelopeError(out, err)
}

// AgentWait runs
// `herdr agent wait TARGET --until U1 [--until U2...] [--timeout MS]`.
// timeoutMS <= 0 omits --timeout. The return value is informational raw
// text (see checkHerdrEnvelopeError); it is not parsed into a struct.
func (h Herdr) AgentWait(ctx context.Context, target string, until []string, timeoutMS int) (string, error) {
	args := []string{"agent", "wait", target}
	for _, u := range until {
		args = append(args, "--until", u)
	}
	if timeoutMS > 0 {
		args = append(args, "--timeout", strconv.Itoa(timeoutMS))
	}
	out, err := h.R.Run(ctx, "herdr", args...)
	return checkHerdrEnvelopeError(out, err)
}

// checkHerdrEnvelopeError implements the shared contract for AgentStart,
// AgentGet and AgentWait: their return values are informational raw text
// (state-machine callers, e.g. AgentWait's "idle" check in internal/org,
// pattern-match on it), so a success envelope's `result` payload is never
// extracted -- only checked for a defensive error envelope, since a
// well-behaved herdr should never return {"error":...} with exit 0. On
// failure (err != nil), the existing Runner error -- which already folds
// captured stderr into its message -- is enriched with the envelope's
// code/message when one is found in stdout or embedded in the error text
// itself, for readability; the original error is preserved via %w either
// way.
func checkHerdrEnvelopeError(out string, err error) (string, error) {
	trimmed := strings.TrimSpace(out)
	if err != nil {
		if _, envErr, isEnvelope := parseHerdrEnvelope(trimmed); isEnvelope && envErr != nil {
			return trimmed, fmt.Errorf("%w (%s)", err, envErr)
		}
		if detail, ok := extractHerdrErrorEnvelope(err.Error()); ok {
			return trimmed, fmt.Errorf("%w (%s)", err, detail)
		}
		return trimmed, err
	}
	if _, envErr, isEnvelope := parseHerdrEnvelope(trimmed); isEnvelope && envErr != nil {
		return "", fmt.Errorf("herdr: unexpected error envelope with exit 0: %w", envErr)
	}
	return trimmed, nil
}

// extractHerdrErrorEnvelope scans s -- typically an already stderr-wrapped
// Runner error's message -- for an embedded `{"error":{...}}` JSON object
// and returns its formatted "code: message" text. This lets
// checkHerdrEnvelopeError attach herdr's structured error detail even when
// the envelope landed on stderr (folded into err.Error()) rather than on
// stdout.
func extractHerdrErrorEnvelope(s string) (string, bool) {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end < start {
		return "", false
	}
	var env herdrEnvelope
	if jsonErr := json.Unmarshal([]byte(s[start:end+1]), &env); jsonErr != nil || env.Error == nil {
		return "", false
	}
	return env.Error.Error(), true
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
