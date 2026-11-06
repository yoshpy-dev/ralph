package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Herdr adapts the herdr CLI (herdr.dev/docs/cli-reference) via a Runner, so
// the org verb layer (internal/org) never shells out directly. Argv shapes
// below mirror the confirmed CLI reference 1:1, one method per subcommand --
// do not invent subcommands beyond what herdr documents. Seat termination is
// sequenced by the caller, not this adapter: see (*Org).Stop in
// internal/org/verbs.go. PaneSendKeys, PaneClose and WorkspaceClose below
// only run the herdr subcommands, and PaneGet, PaneList, TabGet and
// WorkspaceGet only report what herdr answers; the caller compares the
// labels.
//
// Real herdr (confirmed live, v0.7.5) wraps every command's stdout in a JSON
// envelope -- {"id":"cli:<verb>","result":{...}} on success or
// {"error":{"code":"...","message":"..."},"id":"cli:<verb>"} on failure --
// except `pane read`, which is plain text, and `status server --json`, which
// prints a bare JSON object (ServerStatus reads it without
// parseHerdrEnvelope). parseHerdrEnvelope below extracts the `result`
// payload (or a structured error) from that envelope; when stdout is not
// JSON at all (e.g. `pane read`, or a bare id emitted by unit test fakes),
// callers fall back to the trimmed raw string unchanged.
//
// Build a Herdr that sets PaneEnv with NewHerdr. A literal Herdr{R: r}, as
// test fakes use, runs every command exactly as before PaneEnv existed, as
// long as no TabCreate, WorkspaceCreate or PaneSplit is given extraEnv.
type Herdr struct {
	R Runner
	// PaneEnv holds KEY=VALUE entries added as `--env KEY=VALUE`, in order,
	// to the subcommands that make a pane, on every call: `herdr tab create`
	// (TabCreate), `herdr pane split` (PaneSplit), and `herdr workspace
	// create` (WorkspaceCreate, for its root pane). Each call adds its
	// own extraEnv after them, and adds them only when the herdr lists --env
	// in that subcommand's --help (paneEnvSupported). herdr 0.7.5 adds them
	// to the new pane's environment and keeps the rest of it
	// (docs/evidence/agmsg-self-name-2026-10-22.md, (a)). Empty adds no
	// --env and runs no check. Production passes the entries of
	// SeatPaneEnv, for the panes of seats and of a headless director:
	// AgmsgSelfNameOff, so an agmsg run inside such a pane cannot rename the
	// pane's herdr agent (finding F3; the same evidence, (e)), and
	// WIPCheckpointOff, so the WIP-checkpoint hooks of a Claude Code session
	// in it make no wip: commit (finding F11). An entry whose value depends
	// on the pane is not here but the call's extraEnv: spawn passes a seat's
	// PrimaryCLIEntry that way. The values appear in the herdr argv, which ps
	// shows, and in the warning line, so never put a secret in PaneEnv or
	// extraEnv.
	PaneEnv []string
	// Warn receives the warning line a call writes when it cannot add
	// PaneEnv and extraEnv; nil means os.Stderr.
	Warn io.Writer
	// envProbes holds the --env checks NewHerdr sets up, one per subcommand.
	// It is a pointer because Herdr has value receivers and is copied into
	// org.Org's Herdr field, so every copy of one NewHerdr value checks each
	// subcommand once and warns once for it.
	envProbes *paneEnvProbes
}

// NewHerdr returns a Herdr that runs herdr through r and adds paneEnv to
// the panes it makes (see PaneEnv). The value and every copy of it share
// one --env check per subcommand.
func NewHerdr(r Runner, paneEnv ...string) Herdr {
	return Herdr{R: r, PaneEnv: append([]string(nil), paneEnv...), envProbes: &paneEnvProbes{}}
}

// The herdr subcommands that take --env, as the words of the subcommand.
// Each has its own --env check (paneEnvSupported), which runs
// `herdr <words> --help`, and its own warning line, which names the words:
// a herdr can list --env for one of them and not for another (spec
// 2026-11-06-org-herdr-autostart-panes, FR-B8).
const (
	herdrSubTabCreate       = "tab create"
	herdrSubWorkspaceCreate = "workspace create"
	herdrSubPaneSplit       = "pane split"
)

// paneEnvProbes holds the --env check of each subcommand that takes --env,
// shared through Herdr.envProbes.
type paneEnvProbes struct {
	tabCreate       paneEnvProbe
	workspaceCreate paneEnvProbe
	paneSplit       paneEnvProbe
}

// envProbe returns the --env check of sub. A Herdr built as a literal has
// no envProbes, and a sub with no shared check has none either, so each of
// them gets a new check on every call.
func (h Herdr) envProbe(sub string) *paneEnvProbe {
	if h.envProbes == nil {
		return &paneEnvProbe{}
	}
	switch sub {
	case herdrSubTabCreate:
		return &h.envProbes.tabCreate
	case herdrSubWorkspaceCreate:
		return &h.envProbes.workspaceCreate
	case herdrSubPaneSplit:
		return &h.envProbes.paneSplit
	}
	return &paneEnvProbe{}
}

// paneEnvProbe is the cached result of one subcommand's --env check. mu is
// held across the check, so concurrent calls of that subcommand run its
// help once and wait at most herdrPaneEnvProbeTimeout for it.
type paneEnvProbe struct {
	mu        sync.Mutex
	done      bool
	supported bool
}

// herdrPaneEnvProbeTimeout bounds the `herdr <subcommand> --help` call of
// the --env check, within the caller's ctx. The check runs inside the
// budget of the call it precedes (the 10 s driver call timeout of director
// start, the spawn timeout of a seat), so it is short; herdr 0.7.5 prints
// the help in under 0.1 s. A var so a test can shorten it.
var herdrPaneEnvProbeTimeout = 3 * time.Second

// herdrHelpEnvFlag matches --env as a whole flag in help text
// (`      --env <KEY=VALUE>` in herdr 0.7.5, `[--env KEY=VALUE]`,
// `--env=...`), not a longer flag such as --envelope or --environment.
var herdrHelpEnvFlag = regexp.MustCompile(`(?m)(^|[\s,\[(])--env([\s=<\]),]|$)`)

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

// Herdr error codes for a pane, tab or workspace id herdr does not know
// (confirmed live, v0.7.5, as the `pane close` / `workspace close` / `pane
// get` / `tab get` / `workspace get` replies for an unknown id). IsNotFound
// matches exactly these three codes.
const (
	HerdrCodePaneNotFound      = "pane_not_found"
	HerdrCodeTabNotFound       = "tab_not_found"
	HerdrCodeWorkspaceNotFound = "workspace_not_found"
)

// HerdrCodeAgentNotFound is herdr's code for an agent name it does not know
// (confirmed live, v0.7.5, as the `agent get` reply, with exit status 1, for
// an agent that ended with /quit or was killed, while its pane stays open;
// plan 2026-10-18-org-director-liveness, Assumptions). IsNotFound does not
// match it: an agent herdr no longer has is not a pane, tab or workspace
// that is already closed. IsAgentNotFound does, for the errors AgentGet
// returns.
const HerdrCodeAgentNotFound = "agent_not_found"

// NewHerdrError returns the error this adapter builds from a herdr error
// envelope with the given code and message. Production code never needs it
// (the adapter parses envelopes itself); it exists so HerdrClient fakes in
// other packages (internal/org's fakeHerdr) can return an error that
// IsNotFound recognises the same way it recognises a real herdr reply.
func NewHerdrError(code, message string) error {
	return &herdrError{Code: code, Message: message}
}

// NotFound reports whether e's code is pane_not_found, tab_not_found or
// workspace_not_found, the codes IsNotFound matches. internal/org reads it
// through its own one-method interface and errors.As, so it can tell an
// already-closed pane from a failed close without importing this package.
func (e *herdrError) NotFound() bool {
	return e.Code == HerdrCodePaneNotFound || e.Code == HerdrCodeTabNotFound || e.Code == HerdrCodeWorkspaceNotFound
}

// IsNotFound reports whether err carries a herdr error envelope whose code is
// pane_not_found, tab_not_found or workspace_not_found, i.e. the pane, tab or
// workspace the command targeted does not exist (for PaneClose /
// WorkspaceClose: it is already closed). It sees through fmt.Errorf("%w")
// wrapping. Every other
// failure -- other codes, a Runner error with no envelope (connection
// refused, timeout), malformed output -- returns false, as does nil. Only a
// parsed envelope counts: an error whose text merely contains
// "pane_not_found" does not.
func IsNotFound(err error) bool {
	var he *herdrError
	if !errors.As(err, &he) {
		return false
	}
	return he.NotFound()
}

// AgentNotFound reports whether e's code is agent_not_found, the code
// IsAgentNotFound matches. internal/org reads it through its own one-method
// interface and errors.As, as it reads NotFound, so the liveness check of a
// headless director can tell an agent that has ended from an `agent get`
// that failed (plan 2026-10-18-org-director-liveness, D3).
func (e *herdrError) AgentNotFound() bool {
	return e.Code == HerdrCodeAgentNotFound
}

// IsAgentNotFound reports whether err carries a herdr error envelope whose
// code is agent_not_found: herdr has no agent of the name the command gave.
// It sees through fmt.Errorf("%w") wrapping. Only AgentGet wraps the envelope
// of a failed call that way (checkHerdrAgentGetResult), so for an error of
// AgentStart or AgentWait, whose envelope is only folded into the text, it is
// false. Every other failure, the pane, tab and workspace not-found codes
// included, returns false, as does nil; an error whose text merely contains
// "agent_not_found" does not count.
func IsAgentNotFound(err error) bool {
	var he *herdrError
	return errors.As(err, &he) && he.AgentNotFound()
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

// HerdrWorkspace is what WorkspaceCreate reads from herdr's reply: the id of
// the new workspace (result.workspace.workspace_id), and of the tab
// (result.tab.tab_id, labelled "1" by herdr 0.7.5) and its root pane
// (result.root_pane.pane_id) that herdr makes with it. TabID and RootPaneID
// are empty when the reply has no such field, and all but WorkspaceID are
// empty for the non-JSON reply of a unit-test fake, so a caller that uses
// them checks them first.
type HerdrWorkspace struct {
	WorkspaceID string
	TabID       string
	RootPaneID  string
}

// HerdrPaneRef names a pane a TabCreate or PaneSplit made and the tab that
// holds it.
type HerdrPaneRef struct {
	PaneID string
	TabID  string
}

// HerdrPane is a pane as `herdr pane get` and `herdr pane list` report it:
// its id, the ids of the tab and the workspace that hold it, and its label
// (result.pane.label; "" for a pane herdr has no label for, which it
// reports as null or leaves out).
type HerdrPane struct {
	PaneID      string
	TabID       string
	WorkspaceID string
	Label       string
}

// WorkspaceCreate runs `herdr workspace create --cwd CWD --label LABEL` and
// returns the ids of the workspace, its tab and its root pane (see
// HerdrWorkspace). Non-JSON output (unit-test fakes) falls back to the
// trimmed raw stdout as WorkspaceID, the other ids empty. A Runner error
// comes back as it is, with the raw stdout as WorkspaceID.
//
// It adds `--env KEY=VALUE` for each PaneEnv entry and then for each
// extraEnv entry, after `--label LABEL`, when the herdr lists --env in
// `herdr workspace create --help` (withPaneEnv), so the workspace's root
// pane, where the org runtime puts a seat or a headless director (plan
// 2026-11-06-org-herdr-single-tab, D1 and D7), starts with them, as
// TabCreate's pane does. A Herdr with no PaneEnv, given no extraEnv, adds no
// --env and runs no check. On a herdr without --env it creates the
// workspace without them and warns (warnNoPaneEnv) rather than fail.
func (h Herdr) WorkspaceCreate(ctx context.Context, cwd, label string, extraEnv ...string) (HerdrWorkspace, error) {
	args := []string{"workspace", "create", "--cwd", cwd, "--label", label}
	args = h.withPaneEnv(ctx, herdrSubWorkspaceCreate, args, extraEnv)
	out, err := h.R.Run(ctx, "herdr", args...)
	if err != nil {
		return HerdrWorkspace{WorkspaceID: out}, err
	}
	result, envErr, isEnvelope := parseHerdrEnvelope(out)
	if !isEnvelope {
		return HerdrWorkspace{WorkspaceID: strings.TrimSpace(out)}, nil
	}
	if envErr != nil {
		return HerdrWorkspace{}, fmt.Errorf("herdr workspace create: %w", envErr)
	}
	var payload struct {
		Workspace struct {
			WorkspaceID string `json:"workspace_id"`
		} `json:"workspace"`
		Tab struct {
			TabID string `json:"tab_id"`
		} `json:"tab"`
		RootPane struct {
			PaneID string `json:"pane_id"`
		} `json:"root_pane"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return HerdrWorkspace{}, fmt.Errorf("herdr workspace create: parse result: %w", err)
	}
	if payload.Workspace.WorkspaceID == "" {
		return HerdrWorkspace{}, fmt.Errorf("herdr workspace create: envelope missing result.workspace.workspace_id")
	}
	return HerdrWorkspace{WorkspaceID: payload.Workspace.WorkspaceID, TabID: payload.Tab.TabID, RootPaneID: payload.RootPane.PaneID}, nil
}

// TabCreate runs `herdr tab create --workspace ID --cwd CWD --label LABEL`,
// followed by `--env KEY=VALUE` for each PaneEnv entry and then for each
// extraEnv entry, the entries of this one pane (a seat's PrimaryCLIEntry),
// when the herdr has --env (withPaneEnv), and returns the created pane's id,
// from the envelope's result.root_pane.pane_id, and its tab's id, from
// result.root_pane.tab_id, else result.tab.tab_id (empty when the reply has
// neither). Non-JSON output (unit-test fakes) falls back to the trimmed raw
// stdout as PaneID, TabID empty. A Runner error comes back as it is, with
// the raw stdout as PaneID. On a herdr without --env it creates the pane
// without PaneEnv and extraEnv and warns (warnNoPaneEnv) rather than fail.
func (h Herdr) TabCreate(ctx context.Context, workspaceID, cwd, label string, extraEnv ...string) (HerdrPaneRef, error) {
	args := []string{"tab", "create", "--workspace", workspaceID, "--cwd", cwd, "--label", label}
	args = h.withPaneEnv(ctx, herdrSubTabCreate, args, extraEnv)
	out, err := h.R.Run(ctx, "herdr", args...)
	if err != nil {
		return HerdrPaneRef{PaneID: out}, err
	}
	result, envErr, isEnvelope := parseHerdrEnvelope(out)
	if !isEnvelope {
		return HerdrPaneRef{PaneID: strings.TrimSpace(out)}, nil
	}
	if envErr != nil {
		return HerdrPaneRef{}, fmt.Errorf("herdr tab create: %w", envErr)
	}
	var payload struct {
		RootPane struct {
			PaneID string `json:"pane_id"`
			TabID  string `json:"tab_id"`
		} `json:"root_pane"`
		Tab struct {
			TabID string `json:"tab_id"`
		} `json:"tab"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return HerdrPaneRef{}, fmt.Errorf("herdr tab create: parse result: %w", err)
	}
	if payload.RootPane.PaneID == "" {
		return HerdrPaneRef{}, fmt.Errorf("herdr tab create: envelope missing result.root_pane.pane_id")
	}
	tabID := payload.RootPane.TabID
	if tabID == "" {
		tabID = payload.Tab.TabID
	}
	return HerdrPaneRef{PaneID: payload.RootPane.PaneID, TabID: tabID}, nil
}

// Directions PaneSplit accepts, the two herdr 0.7.5 lists for `pane split
// --direction`: the new pane goes to the right of the pane it splits, or
// below it.
const (
	HerdrSplitRight = "right"
	HerdrSplitDown  = "down"
)

// herdrSplitRatio is the --ratio of every PaneSplit: the pane it splits and
// the new pane each get half of its place.
const herdrSplitRatio = "0.5"

// PaneSplit runs `herdr pane split PANEID --direction DIRECTION --ratio 0.5
// --cwd CWD`, followed by `--env KEY=VALUE` for each PaneEnv entry and then
// for each extraEnv entry when the herdr lists --env in `herdr pane split
// --help` (withPaneEnv), and `--no-focus`, so the split does not move the
// focus of a person watching the tab. herdr 0.7.5 puts the new pane in the
// tab of paneID and gives it the --env entries alone: it does not take the
// environment of the pane it splits (spec
// 2026-11-06-org-herdr-autostart-panes, Research findings). direction must
// be HerdrSplitRight or HerdrSplitDown; any other is an error before herdr
// runs.
//
// It returns the new pane's id and its tab's id, from the envelope's
// result.pane.pane_id and result.pane.tab_id. It follows herdrGetResult's
// contract: a paneID herdr does not know is an error for which IsNotFound
// is true, and, unlike TabCreate, a reply it cannot read is an error, never
// a bare id (non-JSON stdout, a malformed envelope). A reply that misses
// either id is an error too.
func (h Herdr) PaneSplit(ctx context.Context, paneID, direction, cwd string, extraEnv ...string) (HerdrPaneRef, error) {
	if direction != HerdrSplitRight && direction != HerdrSplitDown {
		return HerdrPaneRef{}, fmt.Errorf("herdr pane split: direction %q is not %q or %q", direction, HerdrSplitRight, HerdrSplitDown)
	}
	args := []string{"pane", "split", paneID, "--direction", direction, "--ratio", herdrSplitRatio, "--cwd", cwd}
	args = h.withPaneEnv(ctx, herdrSubPaneSplit, args, extraEnv)
	args = append(args, "--no-focus")
	out, runErr := h.R.Run(ctx, "herdr", args...)
	result, err := herdrGetResult("pane split", out, runErr)
	if err != nil {
		return HerdrPaneRef{}, err
	}
	var payload struct {
		Pane *struct {
			PaneID string `json:"pane_id"`
			TabID  string `json:"tab_id"`
		} `json:"pane"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return HerdrPaneRef{}, fmt.Errorf("herdr pane split: parse result: %w", err)
	}
	if payload.Pane == nil || payload.Pane.PaneID == "" || payload.Pane.TabID == "" {
		return HerdrPaneRef{}, fmt.Errorf("herdr pane split: envelope missing result.pane.pane_id or result.pane.tab_id")
	}
	return HerdrPaneRef{PaneID: payload.Pane.PaneID, TabID: payload.Pane.TabID}, nil
}

// PaneRename runs `herdr pane rename PANEID LABEL`, with LABEL as one argv
// word, and TabRename runs `herdr tab rename TABID LABEL` the same way.
// herdr shows the label in `pane get` (result.pane.label) and keeps it
// across an agent start and a server restart (spec
// 2026-11-06-org-herdr-autostart-panes, Research findings). Both follow
// checkHerdrCloseResult's contract: an id herdr does not know is an error
// for which IsNotFound is true, and the success reply is not read.
func (h Herdr) PaneRename(ctx context.Context, paneID, label string) error {
	out, err := h.R.Run(ctx, "herdr", "pane", "rename", paneID, label)
	return checkHerdrCloseResult("pane rename", out, err)
}

// TabRename: see PaneRename.
func (h Herdr) TabRename(ctx context.Context, tabID, label string) error {
	out, err := h.R.Run(ctx, "herdr", "tab", "rename", tabID, label)
	return checkHerdrCloseResult("tab rename", out, err)
}

// PaneList runs `herdr pane list --workspace WORKSPACEID` and returns the
// panes herdr reports, from the envelope's result.panes, in its order. It
// follows herdrGetResult's contract (a workspace herdr does not know is an
// error for which IsNotFound is true; non-JSON stdout is an error), and a
// reply with no result.panes, or with a pane that misses pane_id, tab_id or
// workspace_id (fields herdr 0.7.5's PaneInfo always has), is an error too,
// so a pane it cannot place never passes as one. An empty list is not an
// error.
func (h Herdr) PaneList(ctx context.Context, workspaceID string) ([]HerdrPane, error) {
	out, runErr := h.R.Run(ctx, "herdr", "pane", "list", "--workspace", workspaceID)
	result, err := herdrGetResult("pane list", out, runErr)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Panes *[]herdrPaneJSON `json:"panes"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return nil, fmt.Errorf("herdr pane list: parse result: %w", err)
	}
	if payload.Panes == nil {
		return nil, fmt.Errorf("herdr pane list: envelope missing result.panes")
	}
	panes := make([]HerdrPane, 0, len(*payload.Panes))
	for i, p := range *payload.Panes {
		if p.PaneID == "" || p.TabID == "" || p.WorkspaceID == "" {
			return nil, fmt.Errorf("herdr pane list: result.panes[%d] misses pane_id, tab_id or workspace_id", i)
		}
		panes = append(panes, HerdrPane(p))
	}
	return panes, nil
}

// herdrPaneJSON is the part of herdr's PaneInfo that PaneGet and PaneList
// read. A null label decodes as "".
type herdrPaneJSON struct {
	PaneID      string `json:"pane_id"`
	TabID       string `json:"tab_id"`
	WorkspaceID string `json:"workspace_id"`
	Label       string `json:"label"`
}

// withPaneEnv returns args followed by `--env KEY=VALUE` for each PaneEnv
// entry and then for each extraEnv entry, when there is any entry and the
// herdr lists --env for sub (paneEnvSupported); otherwise args as they are.
// extraEnv is for this one call: PaneEnv itself is left as it is.
func (h Herdr) withPaneEnv(ctx context.Context, sub string, args, extraEnv []string) []string {
	env := append(append([]string(nil), h.PaneEnv...), extraEnv...)
	if len(env) == 0 || !h.paneEnvSupported(ctx, sub, env) {
		return args
	}
	for _, kv := range env {
		args = append(args, "--env", kv)
	}
	return args
}

// paneEnvSupported reports whether a call of sub (herdrSubTabCreate,
// herdrSubWorkspaceCreate or herdrSubPaneSplit) may add --env: whether
// `herdr <sub> --help`, run through R within herdrPaneEnvProbeTimeout,
// succeeds and lists --env (herdrHelpEnvFlag). A herdr without --env would
// refuse the call (`unknown option: --env`), so a failed or timed-out help
// counts as no --env too (plan 2026-10-22-seat-agmsg-self-name, D3). The
// result, a failure included, is cached in sub's envProbe and warned about
// once there, naming env, the entries of the call that checked. A failure
// while the caller's ctx has ended is not cached: it says nothing about the
// herdr, and the call fails on that ctx anyway. A Herdr built as a literal
// has no envProbes, so it checks, and warns, on every call that has PaneEnv
// or extraEnv.
func (h Herdr) paneEnvSupported(ctx context.Context, sub string, env []string) bool {
	p := h.envProbe(sub)
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done {
		return p.supported
	}
	probeCtx, cancel := context.WithTimeout(ctx, herdrPaneEnvProbeTimeout)
	out, err := h.R.Run(probeCtx, "herdr", append(strings.Fields(sub), "--help")...)
	cancel()
	if err != nil && ctx.Err() != nil {
		return false
	}
	p.done = true
	p.supported = err == nil && herdrHelpEnvFlag.MatchString(out)
	if !p.supported {
		h.warnNoPaneEnv(sub, err, env)
	}
	return p.supported
}

// warnNoPaneEnv writes to Warn (os.Stderr when nil) the one warning line for
// a herdr whose sub cannot add env, the PaneEnv and extraEnv entries of the
// call that checked. probeErr is the help call's error, nil when the help
// just does not list --env. The line starts with "warning: " as the CLI's
// other warnings do, names sub, and lists every entry of env. For each entry
// this driver knows (paneEnvConsequence: AgmsgSelfNameOff and
// WIPCheckpointOff, the entries of SeatPaneEnv that production passes for
// the panes of seats and of a headless director, and a seat's
// PrimaryCLIEntry) it names, in env's order, what a pane created without it
// may do; an entry it does not know adds no clause. The clauses are joined
// with "; ", and the advice to use a herdr with --env follows as a sentence
// of its own, after ". ".
func (h Herdr) warnNoPaneEnv(sub string, probeErr error, env []string) {
	w := h.Warn
	if w == nil {
		w = os.Stderr
	}
	reason := "herdr " + sub + " has no --env flag"
	if probeErr != nil {
		reason = "could not check whether herdr " + sub + " has --env ('herdr " + sub + " --help': " +
			strings.Join(strings.Fields(probeErr.Error()), " ") + ")"
	}
	var consequences []string
	for _, kv := range env {
		if c := paneEnvConsequence(kv); c != "" {
			consequences = append(consequences, c)
		}
	}
	what := ""
	if len(consequences) > 0 {
		what = ": " + strings.Join(consequences, "; ")
	}
	_, _ = fmt.Fprintf(w, "warning: %s, so ralph creates herdr panes without --env %s%s. "+
		"Use a herdr whose 'herdr %s --help' lists --env (0.7.5 has it)\n",
		reason, strings.Join(env, " --env "), what, sub)
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
// raw text (see checkHerdrEnvelopeError); it is not parsed into a struct. A
// failed call's error has the text checkHerdrEnvelopeError gives it, and also
// wraps the envelope herdr printed (checkHerdrAgentGetResult), so an agent
// herdr does not know comes back as an error for which IsAgentNotFound is
// true (herdr exits 1 with an agent_not_found envelope).
func (h Herdr) AgentGet(ctx context.Context, target string) (string, error) {
	out, err := h.R.Run(ctx, "herdr", "agent", "get", target)
	return checkHerdrAgentGetResult(out, err)
}

// checkHerdrAgentGetResult is checkHerdrEnvelopeError for AgentGet (plan
// 2026-10-18-org-director-liveness, D3): the same output and the same error
// text, but on a failed call the parsed envelope (from stdout, or from the
// Runner error's text, where ExecRunner folds stderr) is wrapped with %w next
// to the Runner error, as wrapHerdrRunError does, so its code stays readable
// by IsAgentNotFound, and errors.Is on the Runner error still works. Only
// AgentGet uses it: AgentStart and AgentWait keep checkHerdrEnvelopeError,
// whose callers never read a code.
func checkHerdrAgentGetResult(out string, err error) (string, error) {
	if err == nil {
		return checkHerdrEnvelopeError(out, nil)
	}
	trimmed := strings.TrimSpace(out)
	if _, envErr, isEnvelope := parseHerdrEnvelope(trimmed); isEnvelope && envErr != nil {
		return trimmed, fmt.Errorf("%w (%w)", err, envErr)
	}
	if envErr, ok := findHerdrErrorEnvelope(err.Error()); ok {
		return trimmed, fmt.Errorf("%w (%w)", err, envErr)
	}
	return trimmed, err
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
// way. AgentGet's failures go through checkHerdrAgentGetResult instead, which
// gives the same text and also wraps the envelope.
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
	envErr, ok := findHerdrErrorEnvelope(s)
	if !ok {
		return "", false
	}
	return envErr.Error(), true
}

// findHerdrErrorEnvelope is extractHerdrErrorEnvelope's parser. It returns
// the embedded envelope's *herdrError itself, so checkHerdrCloseResult can
// wrap it and keep its code readable by IsNotFound, and
// checkHerdrAgentGetResult readable by IsAgentNotFound.
func findHerdrErrorEnvelope(s string) (*herdrError, bool) {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end < start {
		return nil, false
	}
	var env herdrEnvelope
	if jsonErr := json.Unmarshal([]byte(s[start:end+1]), &env); jsonErr != nil || env.Error == nil {
		return nil, false
	}
	return env.Error, true
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

// PaneClose runs `herdr pane close PANEID`. herdr ends the pane's process and
// removes its tab when the pane was the tab's only one. An unknown id comes
// back as an error for which IsNotFound is true (herdr exits 1 with a
// pane_not_found envelope); see checkHerdrCloseResult.
func (h Herdr) PaneClose(ctx context.Context, paneID string) error {
	out, err := h.R.Run(ctx, "herdr", "pane", "close", paneID)
	return checkHerdrCloseResult("pane close", out, err)
}

// WorkspaceClose runs `herdr workspace close WORKSPACEID`. An unknown id
// comes back as an error for which IsNotFound is true (herdr exits 1 with a
// workspace_not_found envelope); see checkHerdrCloseResult.
func (h Herdr) WorkspaceClose(ctx context.Context, workspaceID string) error {
	out, err := h.R.Run(ctx, "herdr", "workspace", "close", workspaceID)
	return checkHerdrCloseResult("workspace close", out, err)
}

// checkHerdrCloseResult is the error contract PaneClose and WorkspaceClose
// share, and PaneRename and TabRename with them. herdr reports a failed
// close with a non-zero exit and an error envelope on stdout, so the Runner
// returns both an error and that stdout.
// Unlike checkHerdrEnvelopeError, which only folds the envelope's text into
// the message, this wraps the parsed *herdrError with %w next to the Runner
// error (wrapHerdrRunError), so the code survives the non-zero exit and
// IsNotFound can read it (errors.Is on the Runner error, e.g.
// context.DeadlineExceeded, still works). With exit 0, a success envelope or
// non-JSON stdout (unit-test fakes) is success; the result payload
// ({"type":"ok"}) is not read.
func checkHerdrCloseResult(verb, out string, err error) error {
	if err != nil {
		return wrapHerdrRunError(verb, out, err)
	}
	if _, envErr, isEnvelope := parseHerdrEnvelope(out); isEnvelope && envErr != nil {
		return fmt.Errorf("herdr %s: unexpected error envelope with exit 0: %w", verb, envErr)
	}
	return nil
}

// wrapHerdrRunError wraps a failed herdr call's Runner error, naming verb,
// together with the error envelope herdr printed, so its code stays readable
// by IsNotFound. The envelope is looked for on stdout first, then in the
// Runner error's text, where ExecRunner folds stderr.
func wrapHerdrRunError(verb, out string, err error) error {
	if _, envErr, isEnvelope := parseHerdrEnvelope(out); isEnvelope && envErr != nil {
		return fmt.Errorf("herdr %s: %w (%w)", verb, err, envErr)
	}
	if envErr, ok := findHerdrErrorEnvelope(err.Error()); ok {
		return fmt.Errorf("herdr %s: %w (%w)", verb, err, envErr)
	}
	return fmt.Errorf("herdr %s: %w", verb, err)
}

// herdrGetResult is the contract PaneGet, TabGet, WorkspaceGet and PaneList
// share, and PaneSplit with them: a failure follows checkHerdrCloseResult
// (an unknown id is an error for which IsNotFound is true), and a success
// must be a JSON envelope, whose result payload is returned for the caller
// to parse. Unlike WorkspaceCreate and TabCreate, non-JSON stdout is an
// error, not a bare-id fallback: the caller compares labels from the
// payload before it closes anything, or splits a pane it lists, so an
// answer it cannot read must never pass as one.
func herdrGetResult(verb, out string, err error) (json.RawMessage, error) {
	if err != nil {
		return nil, wrapHerdrRunError(verb, out, err)
	}
	result, envErr, isEnvelope := parseHerdrEnvelope(out)
	var he *herdrError
	switch {
	case !isEnvelope:
		return nil, fmt.Errorf("herdr %s: output is not a JSON envelope: %q", verb, strings.TrimSpace(out))
	case errors.As(envErr, &he):
		return nil, fmt.Errorf("herdr %s: unexpected error envelope with exit 0: %w", verb, envErr)
	case envErr != nil: // malformed JSON
		return nil, fmt.Errorf("herdr %s: %w", verb, envErr)
	}
	return result, nil
}

// PaneGet runs `herdr pane get PANEID` and returns the pane (see HerdrPane):
// the ids of the tab and the workspace that hold it, from the envelope's
// result.pane.tab_id and result.pane.workspace_id, and its label, from
// result.pane.label. PaneID is result.pane.pane_id, or paneID when the reply
// has none. An unknown id comes back as an error for which IsNotFound is
// true (herdr exits 1 with a pane_not_found envelope), and a reply that
// misses the tab or the workspace id is an error; one with no label is not.
func (h Herdr) PaneGet(ctx context.Context, paneID string) (HerdrPane, error) {
	out, runErr := h.R.Run(ctx, "herdr", "pane", "get", paneID)
	result, err := herdrGetResult("pane get", out, runErr)
	if err != nil {
		return HerdrPane{}, err
	}
	var payload struct {
		Pane *herdrPaneJSON `json:"pane"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return HerdrPane{}, fmt.Errorf("herdr pane get: parse result: %w", err)
	}
	if payload.Pane == nil || payload.Pane.TabID == "" || payload.Pane.WorkspaceID == "" {
		return HerdrPane{}, fmt.Errorf("herdr pane get: envelope missing result.pane.tab_id or result.pane.workspace_id")
	}
	pane := HerdrPane(*payload.Pane)
	if pane.PaneID == "" {
		pane.PaneID = paneID
	}
	return pane, nil
}

// TabGet runs `herdr tab get TABID` and returns the tab's label, from the
// envelope's result.tab.label (empty for a tab with no label). An unknown id
// comes back as an error for which IsNotFound is true (tab_not_found).
func (h Herdr) TabGet(ctx context.Context, tabID string) (label string, err error) {
	out, runErr := h.R.Run(ctx, "herdr", "tab", "get", tabID)
	result, err := herdrGetResult("tab get", out, runErr)
	if err != nil {
		return "", err
	}
	var payload struct {
		Tab *struct {
			Label string `json:"label"`
		} `json:"tab"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return "", fmt.Errorf("herdr tab get: parse result: %w", err)
	}
	if payload.Tab == nil {
		return "", fmt.Errorf("herdr tab get: envelope missing result.tab")
	}
	return payload.Tab.Label, nil
}

// WorkspaceGet runs `herdr workspace get WORKSPACEID` and returns the
// workspace's label, from the envelope's result.workspace.label (empty for a
// workspace with no label). An unknown id comes back as an error for which
// IsNotFound is true (workspace_not_found).
func (h Herdr) WorkspaceGet(ctx context.Context, workspaceID string) (label string, err error) {
	out, runErr := h.R.Run(ctx, "herdr", "workspace", "get", workspaceID)
	result, err := herdrGetResult("workspace get", out, runErr)
	if err != nil {
		return "", err
	}
	var payload struct {
		Workspace *struct {
			Label string `json:"label"`
		} `json:"workspace"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return "", fmt.Errorf("herdr workspace get: parse result: %w", err)
	}
	if payload.Workspace == nil {
		return "", fmt.Errorf("herdr workspace get: envelope missing result.workspace")
	}
	return payload.Workspace.Label, nil
}

// HerdrServerStatus is what ServerStatus reads from `herdr status server
// --json`: whether the herdr server answers on its socket (Running), whether
// herdr says that server speaks this CLI's protocol (Compatible, nil when
// herdr prints null, as it does for a server that is not running), and the
// socket path the CLI looked at (Socket). The org verbs that make herdr
// create a workspace or a pane start `herdr server` when Running is false
// (spec 2026-11-06-org-herdr-autostart-panes, FR-A1); they read nothing else
// here, so the other keys herdr prints (status, version, protocol,
// capabilities, session, restart_needed) are not kept.
type HerdrServerStatus struct {
	Running    bool
	Compatible *bool
	Socket     string
}

// ErrHerdrStatusJSONUnsupported is the error, wrapped, that ServerStatus
// returns when herdr does not know `status server --json`: it answered
// `unknown option: --json`, which herdr 0.7.5 prints for an argument it does
// not know (seen live on 2026-11-06 for `pane read` and `pane process-info`
// misuse; plan 2026-11-06-org-herdr-autostart, D3). It is the one failure a
// caller may tell apart from the others and go on past, as ralph did before
// it read the status (it then cannot start a server for that herdr); every
// other failure (the Runner's, output that is not the JSON object
// ServerStatus reads) means the status cannot be trusted.
var ErrHerdrStatusJSONUnsupported = errors.New("this herdr has no 'herdr status server --json'")

// herdrUnknownJSONOption matches herdr's line for the --json ServerStatus
// passes, `unknown option: --json`, and not the same line for a longer flag
// such as --jsonl, so only an answer about --json itself reads as
// ErrHerdrStatusJSONUnsupported.
var herdrUnknownJSONOption = regexp.MustCompile(`unknown option: --json([^\w-]|$)`)

// ServerStatus runs `herdr status server --json` and reports the herdr
// server's status. herdr 0.7.5 prints, with exit 0, a bare JSON object, not
// the envelope of the other subcommands (confirmed live, 2026-11-06; spec
// 2026-11-06-org-herdr-autostart-panes, Research findings):
// {"status":"running","running":true,"compatible":true,"socket":"<path>",...}
// for a running server, and
// {"status":"not_running","running":false,"compatible":null,"socket":"<path>",...}
// when no server answers, whether the socket file is stale or absent.
//
// The status decides whether ralph starts a server, so an answer it cannot
// read is an error, never a guess (plan 2026-11-06-org-herdr-autostart, D3):
//
//   - `unknown option: --json` on stdout or in the Runner error's text (where
//     ExecRunner folds stderr) is an error for which
//     errors.Is(err, ErrHerdrStatusJSONUnsupported) holds; the Runner error,
//     when there is one, is wrapped too.
//   - Any other Runner error (herdr exited non-zero, a timeout) is returned
//     wrapped with "herdr status server: ", so errors.Is on it still works.
//   - With exit 0, the trimmed stdout must be one JSON object with a
//     `running` key that holds true or false: malformed JSON, another JSON
//     value, or an object whose `running` is absent or null is an error.
func (h Herdr) ServerStatus(ctx context.Context) (HerdrServerStatus, error) {
	out, err := h.R.Run(ctx, "herdr", "status", "server", "--json")
	if herdrUnknownJSONOption.MatchString(out) || (err != nil && herdrUnknownJSONOption.MatchString(err.Error())) {
		if err != nil {
			return HerdrServerStatus{}, fmt.Errorf("herdr status server: %w (%w)", ErrHerdrStatusJSONUnsupported, err)
		}
		return HerdrServerStatus{}, fmt.Errorf("herdr status server: %w", ErrHerdrStatusJSONUnsupported)
	}
	if err != nil {
		return HerdrServerStatus{}, fmt.Errorf("herdr status server: %w", err)
	}
	trimmed := strings.TrimSpace(out)
	var payload struct {
		Running    *bool  `json:"running"`
		Compatible *bool  `json:"compatible"`
		Socket     string `json:"socket"`
	}
	if jsonErr := json.Unmarshal([]byte(trimmed), &payload); jsonErr != nil {
		return HerdrServerStatus{}, fmt.Errorf("herdr status server: cannot read the output as its JSON object: %w: %q", jsonErr, trimmed)
	}
	if payload.Running == nil {
		return HerdrServerStatus{}, fmt.Errorf("herdr status server: \"running\" is missing or null in the output: %q", trimmed)
	}
	return HerdrServerStatus{Running: *payload.Running, Compatible: payload.Compatible, Socket: payload.Socket}, nil
}
