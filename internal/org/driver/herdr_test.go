package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// realWorkspaceCreateEnvelope and realTabCreateEnvelope are captured live
// from herdr v0.7.5 (see docs/plans/active/2026-08-02-org-runtime-seats.md,
// "Implementation notes (deviations)"). Real herdr wraps every command's
// stdout in a JSON envelope; the PR① adapter wrongly assumed trimmed stdout
// was a bare id.
const realWorkspaceCreateEnvelope = `{"id":"cli:workspace:create","result":{"root_pane":{"pane_id":"w3:p1","tab_id":"w3:t1","workspace_id":"w3"},"tab":{"tab_id":"w3:t1"},"type":"workspace_created","workspace":{"active_tab_id":"w3:t1","workspace_id":"w3"}}}`

const realTabCreateEnvelope = `{"id":"cli:tab:create","result":{"root_pane":{"pane_id":"w3:p2","tab_id":"w3:t2","workspace_id":"w3"},"tab":{"tab_id":"w3:t2"},"type":"tab_created"}}`

const realAgentListEnvelope = `{"id":"cli:agent:list","result":{"agents":[],"type":"agent_list"}}`

const realErrorEnvelope = `{"error":{"code":"workspace_not_found","message":"workspace not found"},"id":"cli:tab:create"}`

// The close envelopes below were captured live from herdr v0.7.5 on an
// isolated server (2026-10-07; docs/evidence/herdr-pane-close-2026-10-07.md).
// The not-found ones come with exit status 1.
const realPaneCloseOKEnvelope = `{"id":"cli:pane:close","result":{"type":"ok"}}`

const realPaneNotFoundEnvelope = `{"error":{"code":"pane_not_found","message":"pane w99:p99 not found"},"id":"cli:pane:close"}`

const realWorkspaceCloseOKEnvelope = `{"id":"cli:workspace:close","result":{"type":"ok"}}`

const realWorkspaceNotFoundEnvelope = `{"error":{"code":"workspace_not_found","message":"workspace w99 not found"},"id":"cli:workspace:close"}`

// The get replies below follow the result payloads herdr v0.7.5 returned on
// an isolated server (2026-10-07; docs/evidence/herdr-pane-close-2026-10-07.md,
// Run 4), with the fields ralph does not read left out and an `id` like every
// other command's envelope. The not-found ones come with exit status 1; their
// codes are the live ones, the messages are made up.
const realPaneGetEnvelope = `{"id":"cli:pane:get","result":{"pane":{"pane_id":"w1:p2","tab_id":"w1:t2","workspace_id":"w1"},"type":"pane_info"}}`

const realTabGetEnvelope = `{"id":"cli:tab:get","result":{"tab":{"label":"seatZ","tab_id":"w1:t2","workspace_id":"w1"},"type":"tab_info"}}`

const realWorkspaceGetEnvelope = `{"id":"cli:workspace:get","result":{"type":"workspace_info","workspace":{"label":"orgA","workspace_id":"w1"}}}`

const realPaneGetNotFoundEnvelope = `{"error":{"code":"pane_not_found","message":"pane w9:p9 not found"},"id":"cli:pane:get"}`

const realTabGetNotFoundEnvelope = `{"error":{"code":"tab_not_found","message":"tab w9:t9 not found"},"id":"cli:tab:get"}`

const realWorkspaceGetNotFoundEnvelope = `{"error":{"code":"workspace_not_found","message":"workspace w9 not found"},"id":"cli:workspace:get"}`

// realAgentNotFoundEnvelope is herdr's `agent get` reply, with exit status 1,
// for an agent it does not know. Its code is the live one (herdr v0.7.5 on an
// isolated server, 2026-10-18; plan 2026-10-18-org-director-liveness,
// Assumptions); the message is made up.
const realAgentNotFoundEnvelope = `{"error":{"code":"agent_not_found","message":"agent director_0123456789ab not found"},"id":"cli:agent:get"}`

// agentNotFoundEnvelopeText is how the error text renders that envelope.
const agentNotFoundEnvelopeText = "herdr: agent_not_found: agent director_0123456789ab not found"

func TestParseHerdrEnvelope(t *testing.T) {
	tests := []struct {
		name        string
		out         string
		wantEnv     bool
		wantErr     bool
		wantErrText string
	}{
		{
			name:    "real workspace create envelope",
			out:     realWorkspaceCreateEnvelope,
			wantEnv: true,
		},
		{
			name:    "real tab create envelope",
			out:     realTabCreateEnvelope,
			wantEnv: true,
		},
		{
			name:    "real agent list envelope",
			out:     realAgentListEnvelope,
			wantEnv: true,
		},
		{
			name:        "real error envelope",
			out:         realErrorEnvelope,
			wantEnv:     true,
			wantErr:     true,
			wantErrText: "workspace_not_found",
		},
		{
			name:    "plain text (pane read) falls back, not an envelope",
			out:     "pane output\nline two",
			wantEnv: false,
		},
		{
			name:    "bare id (unit-test fake) falls back, not an envelope",
			out:     "ws-123",
			wantEnv: false,
		},
		{
			name:        "malformed JSON starting with { is an error, not a fallback",
			out:         `{"id":"cli:tab:create","result":{`,
			wantEnv:     true,
			wantErr:     true,
			wantErrText: "malformed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err, isEnvelope := parseHerdrEnvelope(tt.out)
			if isEnvelope != tt.wantEnv {
				t.Fatalf("isEnvelope = %v, want %v (err=%v, result=%s)", isEnvelope, tt.wantEnv, err, result)
			}
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.wantErrText != "" && !strings.Contains(err.Error(), tt.wantErrText) {
					t.Fatalf("expected error to contain %q, got: %v", tt.wantErrText, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestHerdr_WorkspaceCreate(t *testing.T) {
	f := &fakeRunner{outputs: []string{"ws-123"}}
	h := Herdr{R: f}

	got, err := h.WorkspaceCreate(context.Background(), "/tmp/cwd", "leader")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != (HerdrWorkspace{WorkspaceID: "ws-123"}) {
		t.Fatalf("want ws-123 and no tab or root pane, got %+v", got)
	}
	want := []string{"workspace", "create", "--cwd", "/tmp/cwd", "--label", "leader"}
	if c := f.lastCall(); c.name != "herdr" || !reflect.DeepEqual(c.args, want) {
		t.Fatalf("argv mismatch: got name=%q args=%v, want name=herdr args=%v", c.name, c.args, want)
	}
}

// TestHerdr_WorkspaceCreate_RealEnvelope pins the fix: real herdr wraps
// stdout in a JSON envelope, so WorkspaceCreate must extract
// result.workspace.workspace_id rather than returning the JSON blob itself
// (the bug: the blob was passed straight to `tab create --workspace`,
// producing workspace_not_found). It also reads the tab and the root pane
// herdr makes with the workspace (plan 2026-11-06-org-herdr-single-tab, D8).
func TestHerdr_WorkspaceCreate_RealEnvelope(t *testing.T) {
	f := &fakeRunner{outputs: []string{realWorkspaceCreateEnvelope}}
	h := Herdr{R: f}

	got, err := h.WorkspaceCreate(context.Background(), "/tmp/cwd", "leader")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := (HerdrWorkspace{WorkspaceID: "w3", TabID: "w3:t1", RootPaneID: "w3:p1"}); got != want {
		t.Fatalf("got %+v, want %+v (result.workspace.workspace_id, result.tab.tab_id, result.root_pane.pane_id)", got, want)
	}
}

// TestHerdr_WorkspaceCreate_ErrorEnvelope pins that an {"error":...}
// envelope surfaces as a structured Go error, including the code, rather
// than being treated as a bare id.
func TestHerdr_WorkspaceCreate_ErrorEnvelope(t *testing.T) {
	f := &fakeRunner{outputs: []string{realErrorEnvelope}}
	h := Herdr{R: f}

	_, err := h.WorkspaceCreate(context.Background(), "/tmp/cwd", "leader")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "workspace_not_found") {
		t.Fatalf("expected error to contain the envelope code, got: %v", err)
	}
}

func TestHerdr_TabCreate(t *testing.T) {
	f := &fakeRunner{outputs: []string{"tab-9"}}
	h := Herdr{R: f}

	got, err := h.TabCreate(context.Background(), "ws-123", "/tmp/cwd", "worker-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != (HerdrPaneRef{PaneID: "tab-9"}) {
		t.Fatalf("want pane tab-9 and no tab id, got %+v", got)
	}
	want := []string{"tab", "create", "--workspace", "ws-123", "--cwd", "/tmp/cwd", "--label", "worker-1"}
	if c := f.lastCall(); !reflect.DeepEqual(c.args, want) {
		t.Fatalf("argv mismatch: got %v, want %v", c.args, want)
	}
}

// TestHerdr_TabCreate_RealEnvelope pins the fix: TabCreate must extract
// result.root_pane.pane_id from the real JSON envelope, and with it the
// tab's id, result.root_pane.tab_id.
func TestHerdr_TabCreate_RealEnvelope(t *testing.T) {
	f := &fakeRunner{outputs: []string{realTabCreateEnvelope}}
	h := Herdr{R: f}

	got, err := h.TabCreate(context.Background(), "w3", "/tmp/cwd", "worker-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := (HerdrPaneRef{PaneID: "w3:p2", TabID: "w3:t2"}); got != want {
		t.Fatalf("got %+v, want %+v (result.root_pane.pane_id and tab_id)", got, want)
	}
}

// TestHerdr_TabCreate_TabIDFallsBackToResultTab: TabCreate reads the tab id
// from result.root_pane.tab_id, then from result.tab.tab_id, and leaves it
// empty, with no error, when the reply has neither; a missing pane id is
// still an error.
func TestHerdr_TabCreate_TabIDFallsBackToResultTab(t *testing.T) {
	for name, tt := range map[string]struct {
		out     string
		want    HerdrPaneRef
		wantErr string
	}{
		"root_pane.tab_id wins over tab.tab_id": {
			out:  `{"id":"cli:tab:create","result":{"root_pane":{"pane_id":"w3:p2","tab_id":"w3:t2"},"tab":{"tab_id":"w3:t9"},"type":"tab_created"}}`,
			want: HerdrPaneRef{PaneID: "w3:p2", TabID: "w3:t2"},
		},
		"tab.tab_id when root_pane has none": {
			out:  `{"id":"cli:tab:create","result":{"root_pane":{"pane_id":"w3:p2"},"tab":{"tab_id":"w3:t9"},"type":"tab_created"}}`,
			want: HerdrPaneRef{PaneID: "w3:p2", TabID: "w3:t9"},
		},
		"no tab id at all": {
			out:  `{"id":"cli:tab:create","result":{"root_pane":{"pane_id":"w3:p2"},"type":"tab_created"}}`,
			want: HerdrPaneRef{PaneID: "w3:p2"},
		},
		"no pane id": {
			out:     `{"id":"cli:tab:create","result":{"root_pane":{"tab_id":"w3:t2"},"tab":{"tab_id":"w3:t2"},"type":"tab_created"}}`,
			wantErr: "envelope missing result.root_pane.pane_id",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := (Herdr{R: &fakeRunner{outputs: []string{tt.out}}}).TabCreate(context.Background(), "w3", "/tmp/cwd", "worker-1")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) || got != (HerdrPaneRef{}) {
					t.Fatalf("TabCreate = %+v, %v; want no ids and an error containing %q", got, err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("TabCreate = %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
}

// TestHerdr_WorkspaceCreate_PartialEnvelopeKeepsTheWorkspace: a reply with
// the workspace id but no tab or root pane is no error (it was none before
// WorkspaceCreate read them); those ids stay empty. A reply without the
// workspace id is still an error.
func TestHerdr_WorkspaceCreate_PartialEnvelopeKeepsTheWorkspace(t *testing.T) {
	got, err := (Herdr{R: &fakeRunner{outputs: []string{`{"id":"cli:workspace:create","result":{"type":"workspace_created","workspace":{"workspace_id":"w3"}}}`}}}).
		WorkspaceCreate(context.Background(), "/tmp/cwd", "org-x")
	if err != nil || got != (HerdrWorkspace{WorkspaceID: "w3"}) {
		t.Fatalf("WorkspaceCreate = %+v, %v; want w3 alone", got, err)
	}
	got, err = (Herdr{R: &fakeRunner{outputs: []string{`{"id":"cli:workspace:create","result":{"root_pane":{"pane_id":"w3:p1"},"tab":{"tab_id":"w3:t1"},"type":"workspace_created","workspace":{}}}`}}}).
		WorkspaceCreate(context.Background(), "/tmp/cwd", "org-x")
	if err == nil || !strings.Contains(err.Error(), "envelope missing result.workspace.workspace_id") || got != (HerdrWorkspace{}) {
		t.Fatalf("WorkspaceCreate = %+v, %v; want no ids and the missing-workspace error", got, err)
	}
}

// TestHerdr_Create_RunnerErrorKeepsStdout pins the error behavior
// WorkspaceCreate and TabCreate keep: a Runner error comes back unwrapped,
// with the raw stdout in the id field, as they returned it as the id
// before they returned structs.
func TestHerdr_Create_RunnerErrorKeepsStdout(t *testing.T) {
	ws, err := (Herdr{R: &fakeRunner{outputs: []string{"Error: boom"}, errs: []error{errExit1}}}).WorkspaceCreate(context.Background(), "/tmp/cwd", "org-x")
	if err != errExit1 || ws != (HerdrWorkspace{WorkspaceID: "Error: boom"}) {
		t.Fatalf("WorkspaceCreate = %+v, %v; want the stdout as WorkspaceID and the Runner error itself", ws, err)
	}
	pane, err := (Herdr{R: &fakeRunner{outputs: []string{"Error: boom"}, errs: []error{errExit1}}}).TabCreate(context.Background(), "w3", "/tmp/cwd", "worker-1")
	if err != errExit1 || pane != (HerdrPaneRef{PaneID: "Error: boom"}) {
		t.Fatalf("TabCreate = %+v, %v; want the stdout as PaneID and the Runner error itself", pane, err)
	}
}

// TestHerdr_TabCreate_ErrorEnvelope mirrors the real failure this slice
// fixes: workspace_not_found returned as a structured error, not a bare
// string passed further downstream.
func TestHerdr_TabCreate_ErrorEnvelope(t *testing.T) {
	f := &fakeRunner{outputs: []string{realErrorEnvelope}}
	h := Herdr{R: f}

	_, err := h.TabCreate(context.Background(), "bogus-blob", "/tmp/cwd", "worker-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "workspace_not_found") {
		t.Fatalf("expected error to contain the envelope code, got: %v", err)
	}
}

// herdrTabCreateHelp is `herdr tab create --help` of herdr 0.7.5 (run
// 2026-10-22), with its blank description lines left out.
const herdrTabCreateHelp = `Create a tab

Usage: herdr tab create [OPTIONS]

Options:
      --workspace <WORKSPACE_ID>
      --cwd <PATH>
      --label <TEXT>
      --env <KEY=VALUE>
          Set an environment variable for the launched process
      --focus
      --no-focus`

// herdrTabCreateHelpNoEnv is a made-up help of a herdr without --env, whose
// other flags start with "--env".
const herdrTabCreateHelpNoEnv = `Usage: herdr tab create [OPTIONS]

Options:
      --workspace <WORKSPACE_ID>
      --cwd <PATH>
      --label <TEXT>
      --envelope <FILE>
      --environment <NAME>`

// tabCreateArgv is the argv TabCreate runs for the calls below, before any
// --env.
var tabCreateArgv = []string{"herdr", "tab", "create", "--workspace", "w3", "--cwd", "/tmp/cwd", "--label", "worker-1"}

// paneEnvFakeRunner answers `herdr tab create --help` with help and helpErr,
// or with helpFn when it is set, and every other call with
// realTabCreateEnvelope. It records each call's argv; mu makes it safe for
// concurrent TabCreate calls.
type paneEnvFakeRunner struct {
	mu        sync.Mutex
	help      string
	helpErr   error
	helpFn    func(ctx context.Context) (string, error)
	helpCalls int
	tabCalls  [][]string
}

func (f *paneEnvFakeRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	argv := append([]string{name}, args...)
	isHelp := reflect.DeepEqual(argv, []string{"herdr", "tab", "create", "--help"})
	f.mu.Lock()
	if isHelp {
		f.helpCalls++
	} else {
		f.tabCalls = append(f.tabCalls, argv)
	}
	f.mu.Unlock()
	if !isHelp {
		return realTabCreateEnvelope, nil
	}
	if f.helpFn != nil {
		return f.helpFn(ctx)
	}
	return f.help, f.helpErr
}

// tabCreateTwice runs TabCreate with extraEnv through h and through a copy of
// h held as a HerdrClient-shaped interface, the way org.Org holds it, and
// checks that both return the pane and tab ids of realTabCreateEnvelope.
func tabCreateTwice(t *testing.T, h Herdr, extraEnv ...string) {
	t.Helper()
	var copied interface {
		TabCreate(ctx context.Context, workspaceID, cwd, label string, extraEnv ...string) (HerdrPaneRef, error)
	} = h
	for i, create := range []func(ctx context.Context, workspaceID, cwd, label string, extraEnv ...string) (HerdrPaneRef, error){h.TabCreate, copied.TabCreate} {
		got, err := create(context.Background(), "w3", "/tmp/cwd", "worker-1", extraEnv...)
		if err != nil {
			t.Fatalf("TabCreate #%d: unexpected error: %v", i+1, err)
		}
		if want := (HerdrPaneRef{PaneID: "w3:p2", TabID: "w3:t2"}); got != want {
			t.Fatalf("TabCreate #%d = %+v; want %+v", i+1, got, want)
		}
	}
}

// TestHerdr_TabCreate_PaneEnvAddsEnvWhenHelpListsIt pins AC3 of plan
// 2026-10-22-seat-agmsg-self-name and AC2 of plan
// 2026-10-22-seat-wip-checkpoint: with a herdr whose `tab create --help`
// lists --env, every tab create of a Herdr with SeatPaneEnv gets
// `--env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off` after --label,
// and the help runs once for a NewHerdr value and its copies.
func TestHerdr_TabCreate_PaneEnvAddsEnvWhenHelpListsIt(t *testing.T) {
	f := &paneEnvFakeRunner{help: herdrTabCreateHelp}
	var warn bytes.Buffer
	h := NewHerdr(f, SeatPaneEnv()...)
	h.Warn = &warn

	tabCreateTwice(t, h)

	want := append(append([]string{}, tabCreateArgv...), "--env", "AGMSG_SELF_NAME=off", "--env", "RALPH_WIP_CHECKPOINT=off")
	if !reflect.DeepEqual(f.tabCalls, [][]string{want, want}) {
		t.Fatalf("tab create argv = %v; want %v twice", f.tabCalls, want)
	}
	if f.helpCalls != 1 {
		t.Fatalf("help ran %d times; want 1 for a value and its copy", f.helpCalls)
	}
	if warn.Len() != 0 {
		t.Fatalf("warning = %q; want none", warn.String())
	}
}

// TestHerdr_TabCreate_PaneEnvWithoutEnvFlagWarnsOnce pins AC4 (of both
// plans): with a herdr whose help does not list --env, or whose help fails
// or times out, the tab of a Herdr with SeatPaneEnv is created without
// --env and the warning, which names both entries and their consequences,
// is written once, as one line.
func TestHerdr_TabCreate_PaneEnvWithoutEnvFlagWarnsOnce(t *testing.T) {
	tests := []struct {
		name       string
		runner     func() *paneEnvFakeRunner
		probeLimit time.Duration
		wantReason string
	}{
		{
			name:       "help lists only --envelope and --environment",
			runner:     func() *paneEnvFakeRunner { return &paneEnvFakeRunner{help: herdrTabCreateHelpNoEnv} },
			wantReason: "herdr tab create has no --env flag",
		},
		{
			name: "help fails",
			runner: func() *paneEnvFakeRunner {
				return &paneEnvFakeRunner{help: "unknown option: --help", helpErr: errors.New("herdr: exit status 2:\nunknown option: --help")}
			},
			wantReason: "could not check whether herdr tab create has --env ('herdr tab create --help': herdr: exit status 2: unknown option: --help)",
		},
		{
			name: "help lists --env but exits non-zero",
			runner: func() *paneEnvFakeRunner {
				return &paneEnvFakeRunner{help: herdrTabCreateHelp, helpErr: errors.New("herdr: exit status 1")}
			},
			wantReason: "could not check whether herdr tab create has --env ('herdr tab create --help': herdr: exit status 1)",
		},
		{
			name: "help times out",
			runner: func() *paneEnvFakeRunner {
				return &paneEnvFakeRunner{helpFn: func(ctx context.Context) (string, error) {
					select {
					case <-ctx.Done():
						return "", fmt.Errorf("herdr: timed out: %w", ctx.Err())
					case <-time.After(2 * time.Second):
						return "", errors.New("fake help: the probe timeout did not end the call")
					}
				}}
			},
			probeLimit: 50 * time.Millisecond,
			wantReason: "could not check whether herdr tab create has --env ('herdr tab create --help': herdr: timed out: context deadline exceeded)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.probeLimit > 0 {
				old := herdrPaneEnvProbeTimeout
				herdrPaneEnvProbeTimeout = tt.probeLimit
				t.Cleanup(func() { herdrPaneEnvProbeTimeout = old })
			}
			f := tt.runner()
			var warn bytes.Buffer
			h := NewHerdr(f, SeatPaneEnv()...)
			h.Warn = &warn

			start := time.Now()
			tabCreateTwice(t, h)
			if elapsed := time.Since(start); elapsed > time.Second {
				t.Fatalf("two TabCreate calls took %v; want well under the fake help's 2 s", elapsed)
			}

			if !reflect.DeepEqual(f.tabCalls, [][]string{tabCreateArgv, tabCreateArgv}) {
				t.Fatalf("tab create argv = %v; want %v twice, with no --env", f.tabCalls, tabCreateArgv)
			}
			if f.helpCalls != 1 {
				t.Fatalf("help ran %d times; want 1, the failure cached", f.helpCalls)
			}
			if want := noPaneEnvWarning(tt.wantReason); warn.String() != want {
				t.Fatalf("warning =\n%q\nwant exactly one line\n%q", warn.String(), want)
			}
		})
	}
}

// The clauses warnNoPaneEnv writes for AgmsgSelfNameOff, for
// WIPCheckpointOff, and for a PrimaryCLIEnv entry, and the advice that ends
// every warning line, written out here so that a change to
// paneEnvConsequence's text fails the tests.
const (
	agmsgSelfNameClause = "an agmsg run inside such a pane (a seat's or a headless director's) may rename the pane's herdr agent, " +
		"and ralph may then fail to reach it (agent_not_found)"
	wipCheckpointClause = "a Claude Code session in such a pane may commit a feature branch's uncommitted changes as 'wip:' " +
		"at its session end or context compaction (its WIP-checkpoint hooks)"
	primaryCLIClause = "a feature org leader's plan advisory in such a pane may pick the wrong CLI as its other model " +
		"if the pane takes a claude or codex RALPH_PRIMARY_CLI from the herdr server's environment " +
		"(scripts/xreview-helpers.sh pick_reviewer then sees that value instead of the leader's driver; " +
		"with none, the leader names its own CLI)"
	noPaneEnvAdvice = ". Use a herdr whose 'herdr tab create --help' lists --env (0.7.5 has it)\n"
)

// noPaneEnvWarning is the line warnNoPaneEnv writes for PaneEnv
// SeatPaneEnv() and reason.
func noPaneEnvWarning(reason string) string {
	return "warning: " + reason + ", so ralph creates herdr panes without --env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off: " +
		agmsgSelfNameClause + "; " + wipCheckpointClause + noPaneEnvAdvice
}

// TestHerdr_TabCreate_PaneEnvWarningNamesEachEntrysConsequence pins AC4 of
// plan 2026-10-22-seat-wip-checkpoint: on a herdr without --env the one
// warning line lists every PaneEnv entry and adds, in PaneEnv's order, the
// clause of each entry the driver knows; an entry it does not know adds no
// clause. Every RALPH_PRIMARY_CLI entry, whatever its value, has the clause
// of PrimaryCLIEntry (AC5 of plan 2026-10-29-review-switches-config); a
// longer name that starts with it has none. The clauses are joined with "; "
// and the advice follows as a sentence of its own, so it does not read as
// one more clause. With AgmsgSelfNameOff alone the line is the one of plan
// 2026-10-22-seat-agmsg-self-name except that the advice starts ". Use"
// where that line had "; use". The PrimaryCLIEntry clause is about a feature
// org leader's plan advisory only: no line says a seat's cross-review may
// pick the wrong CLI, since a reviewer seat's cross-review passes
// pick_reviewer the implementer's driver from its TASK (plan
// 2026-11-01-org-cross-review, D6).
func TestHerdr_TabCreate_PaneEnvWarningNamesEachEntrysConsequence(t *testing.T) {
	const head = "warning: herdr tab create has no --env flag, so ralph creates herdr panes without --env "
	for name, tt := range map[string]struct {
		paneEnv []string
		want    string
	}{
		"SeatPaneEnv": {SeatPaneEnv(),
			head + "AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off: " + agmsgSelfNameClause + "; " + wipCheckpointClause + noPaneEnvAdvice},
		"AgmsgSelfNameOff alone": {[]string{AgmsgSelfNameOff},
			"warning: herdr tab create has no --env flag, so ralph creates herdr panes without --env AGMSG_SELF_NAME=off: " +
				"an agmsg run inside such a pane (a seat's or a headless director's) may rename the pane's herdr agent, " +
				"and ralph may then fail to reach it (agent_not_found). " +
				"Use a herdr whose 'herdr tab create --help' lists --env (0.7.5 has it)\n"},
		"WIPCheckpointOff alone": {[]string{WIPCheckpointOff},
			head + "RALPH_WIP_CHECKPOINT=off: " + wipCheckpointClause + noPaneEnvAdvice},
		"the two the other way round": {[]string{WIPCheckpointOff, AgmsgSelfNameOff},
			head + "RALPH_WIP_CHECKPOINT=off --env AGMSG_SELF_NAME=off: " + wipCheckpointClause + "; " + agmsgSelfNameClause + noPaneEnvAdvice},
		"an unknown entry": {[]string{"X=1"},
			head + "X=1" + noPaneEnvAdvice},
		"an unknown entry and WIPCheckpointOff": {[]string{"X=1", WIPCheckpointOff},
			head + "X=1 --env RALPH_WIP_CHECKPOINT=off: " + wipCheckpointClause + noPaneEnvAdvice},
		"PrimaryCLIEntry of codex alone": {[]string{PrimaryCLIEntry("codex")},
			head + "RALPH_PRIMARY_CLI=codex: " + primaryCLIClause + noPaneEnvAdvice},
		"RALPH_PRIMARY_CLI with any value": {[]string{"RALPH_PRIMARY_CLI="},
			head + "RALPH_PRIMARY_CLI=: " + primaryCLIClause + noPaneEnvAdvice},
		"a longer name that starts with RALPH_PRIMARY_CLI": {[]string{"RALPH_PRIMARY_CLI_X=claude"},
			head + "RALPH_PRIMARY_CLI_X=claude" + noPaneEnvAdvice},
	} {
		t.Run(name, func(t *testing.T) {
			f := &paneEnvFakeRunner{help: herdrTabCreateHelpNoEnv}
			var warn bytes.Buffer
			h := NewHerdr(f, tt.paneEnv...)
			h.Warn = &warn

			tabCreateTwice(t, h)

			if !reflect.DeepEqual(f.tabCalls, [][]string{tabCreateArgv, tabCreateArgv}) {
				t.Fatalf("tab create argv = %v; want %v twice, with no --env", f.tabCalls, tabCreateArgv)
			}
			if warn.String() != tt.want {
				t.Fatalf("warning =\n%q\nwant exactly one line\n%q", warn.String(), tt.want)
			}
			if strings.Contains(warn.String(), "a seat's cross-review") {
				t.Fatalf("warning = %q; want no clause about a seat's cross-review, which takes the implementer's driver from its TASK", warn.String())
			}
		})
	}
}

// TestHerdr_TabCreate_PaneEnvNilWarnWritesToStderr pins AC4 for the Herdr
// the CLI builds, a NewHerdr value with Warn nil: with a herdr whose help
// does not list --env, the warning line goes to os.Stderr, once. It swaps
// os.Stderr, so it does not run in parallel.
func TestHerdr_TabCreate_PaneEnvNilWarnWritesToStderr(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	t.Cleanup(func() {
		os.Stderr = orig
		_ = w.Close()
		_ = r.Close()
	})
	f := &paneEnvFakeRunner{help: herdrTabCreateHelpNoEnv}

	tabCreateTwice(t, NewHerdr(f, SeatPaneEnv()...))

	os.Stderr = orig
	_ = w.Close()
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read the os.Stderr pipe: %v", err)
	}
	if !reflect.DeepEqual(f.tabCalls, [][]string{tabCreateArgv, tabCreateArgv}) {
		t.Fatalf("tab create argv = %v; want %v twice, with no --env", f.tabCalls, tabCreateArgv)
	}
	if want := noPaneEnvWarning("herdr tab create has no --env flag"); string(got) != want {
		t.Fatalf("os.Stderr =\n%q\nwant exactly one line\n%q", got, want)
	}
}

// TestHerdr_TabCreate_NoPaneEnvRunsNoProbe: without PaneEnv, a literal Herdr
// and a NewHerdr value run today's argv and never run the help.
func TestHerdr_TabCreate_NoPaneEnvRunsNoProbe(t *testing.T) {
	for name, build := range map[string]func(r Runner) Herdr{
		"literal":   func(r Runner) Herdr { return Herdr{R: r} },
		"NewHerdr":  func(r Runner) Herdr { return NewHerdr(r) },
		"empty env": func(r Runner) Herdr { return NewHerdr(r, []string{}...) },
	} {
		t.Run(name, func(t *testing.T) {
			f := &paneEnvFakeRunner{help: herdrTabCreateHelp}
			var warn bytes.Buffer
			h := build(f)
			h.Warn = &warn

			tabCreateTwice(t, h)

			if f.helpCalls != 0 {
				t.Fatalf("help ran %d times; want 0 with no PaneEnv", f.helpCalls)
			}
			if !reflect.DeepEqual(f.tabCalls, [][]string{tabCreateArgv, tabCreateArgv}) {
				t.Fatalf("tab create argv = %v; want %v twice", f.tabCalls, tabCreateArgv)
			}
			if warn.Len() != 0 {
				t.Fatalf("warning = %q; want none", warn.String())
			}
		})
	}
}

// TestHerdr_TabCreate_PaneEnvKeepsOrder: each PaneEnv entry becomes one
// --env, in PaneEnv's order, a repeated key included.
func TestHerdr_TabCreate_PaneEnvKeepsOrder(t *testing.T) {
	f := &paneEnvFakeRunner{help: herdrTabCreateHelp}
	h := NewHerdr(f, "A=1", "B=2", "A=3")

	if _, err := h.TabCreate(context.Background(), "w3", "/tmp/cwd", "worker-1"); err != nil {
		t.Fatalf("TabCreate: %v", err)
	}
	want := append(append([]string{}, tabCreateArgv...), "--env", "A=1", "--env", "B=2", "--env", "A=3")
	if !reflect.DeepEqual(f.tabCalls, [][]string{want}) {
		t.Fatalf("tab create argv = %v; want %v", f.tabCalls, want)
	}
}

// TestPrimaryCLIEntry pins the variable spawn passes to a seat's pane, the
// one scripts/xreview-helpers.sh's pick_reviewer reads, and its entry.
func TestPrimaryCLIEntry(t *testing.T) {
	if PrimaryCLIEnv != "RALPH_PRIMARY_CLI" {
		t.Fatalf("PrimaryCLIEnv = %q; want RALPH_PRIMARY_CLI", PrimaryCLIEnv)
	}
	for driverName, want := range map[string]string{"claude": "RALPH_PRIMARY_CLI=claude", "codex": "RALPH_PRIMARY_CLI=codex"} {
		if got := PrimaryCLIEntry(driverName); got != want {
			t.Fatalf("PrimaryCLIEntry(%q) = %q; want %q", driverName, got, want)
		}
	}
}

// TestHerdr_TabCreate_ExtraEnvFollowsPaneEnv pins AC5 of plan
// 2026-10-29-review-switches-config at the driver: with a herdr whose help
// lists --env, TabCreate adds an --env for each PaneEnv entry and then for
// each extraEnv entry, and a Herdr with no PaneEnv adds the extraEnv
// entries alone. extraEnv is for one call: a later call without it gets
// PaneEnv alone, and PaneEnv itself is left as it was.
func TestHerdr_TabCreate_ExtraEnvFollowsPaneEnv(t *testing.T) {
	for name, tt := range map[string]struct {
		build     func(r Runner) Herdr
		extraEnv  []string
		wantEnv   []string // the --env values of the calls with extraEnv
		wantLater []string // the --env values of the call without it
		wantHelp  int
	}{
		"SeatPaneEnv and a codex seat": {
			build:     func(r Runner) Herdr { return NewHerdr(r, SeatPaneEnv()...) },
			extraEnv:  []string{PrimaryCLIEntry("codex")},
			wantEnv:   []string{"AGMSG_SELF_NAME=off", "RALPH_WIP_CHECKPOINT=off", "RALPH_PRIMARY_CLI=codex"},
			wantLater: []string{"AGMSG_SELF_NAME=off", "RALPH_WIP_CHECKPOINT=off"},
			wantHelp:  1,
		},
		"two extra entries keep their order": {
			build:     func(r Runner) Herdr { return NewHerdr(r, "A=1") },
			extraEnv:  []string{"B=2", "A=3"},
			wantEnv:   []string{"A=1", "B=2", "A=3"},
			wantLater: []string{"A=1"},
			wantHelp:  1,
		},
		"a literal with no PaneEnv": {
			build:    func(r Runner) Herdr { return Herdr{R: r} },
			extraEnv: []string{PrimaryCLIEntry("claude")},
			wantEnv:  []string{"RALPH_PRIMARY_CLI=claude"},
			wantHelp: 2, // a literal checks on every call that has env; the later call has none
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := &paneEnvFakeRunner{help: herdrTabCreateHelp}
			var warn bytes.Buffer
			h := tt.build(f)
			h.Warn = &warn
			paneEnv := slices.Clone(h.PaneEnv)

			tabCreateTwice(t, h, tt.extraEnv...)
			if _, err := h.TabCreate(context.Background(), "w3", "/tmp/cwd", "worker-1"); err != nil {
				t.Fatalf("TabCreate without extraEnv: %v", err)
			}

			withEnv := func(env []string) []string {
				argv := slices.Clone(tabCreateArgv)
				for _, kv := range env {
					argv = append(argv, "--env", kv)
				}
				return argv
			}
			want := [][]string{withEnv(tt.wantEnv), withEnv(tt.wantEnv), withEnv(tt.wantLater)}
			if !reflect.DeepEqual(f.tabCalls, want) {
				t.Fatalf("tab create argv = %v; want %v", f.tabCalls, want)
			}
			if f.helpCalls != tt.wantHelp {
				t.Fatalf("help ran %d times; want %d", f.helpCalls, tt.wantHelp)
			}
			if !slices.Equal(h.PaneEnv, paneEnv) {
				t.Fatalf("PaneEnv = %q after the calls; want it unchanged, %q", h.PaneEnv, paneEnv)
			}
			if warn.Len() != 0 {
				t.Fatalf("warning = %q; want none", warn.String())
			}
		})
	}
}

// TestHerdr_TabCreate_ExtraEnvWithoutEnvFlagWarnsWithItsClause pins AC5 of
// plan 2026-10-29-review-switches-config for a herdr without --env: the tab
// of a seat is created with no --env at all, and the one warning line lists
// the PaneEnv entries, then the seat's RALPH_PRIMARY_CLI, each with its
// clause, in that order.
func TestHerdr_TabCreate_ExtraEnvWithoutEnvFlagWarnsWithItsClause(t *testing.T) {
	f := &paneEnvFakeRunner{help: herdrTabCreateHelpNoEnv}
	var warn bytes.Buffer
	h := NewHerdr(f, SeatPaneEnv()...)
	h.Warn = &warn

	tabCreateTwice(t, h, PrimaryCLIEntry("claude"))

	if !reflect.DeepEqual(f.tabCalls, [][]string{tabCreateArgv, tabCreateArgv}) {
		t.Fatalf("tab create argv = %v; want %v twice, with no --env", f.tabCalls, tabCreateArgv)
	}
	if f.helpCalls != 1 {
		t.Fatalf("help ran %d times; want 1, the result cached", f.helpCalls)
	}
	want := "warning: herdr tab create has no --env flag, so ralph creates herdr panes without " +
		"--env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off --env RALPH_PRIMARY_CLI=claude: " +
		agmsgSelfNameClause + "; " + wipCheckpointClause + "; " + primaryCLIClause + noPaneEnvAdvice
	if warn.String() != want {
		t.Fatalf("warning =\n%q\nwant exactly one line\n%q", warn.String(), want)
	}
}

// TestNewHerdr_CopiesPaneEnv: NewHerdr keeps its own copy of the entries, so
// a caller that changes its slice afterwards does not change the panes.
func TestNewHerdr_CopiesPaneEnv(t *testing.T) {
	env := []string{"A=1"}
	h := NewHerdr(&paneEnvFakeRunner{}, env...)
	env[0] = "A=changed"
	if !reflect.DeepEqual(h.PaneEnv, []string{"A=1"}) {
		t.Fatalf("PaneEnv = %v; want [A=1]", h.PaneEnv)
	}
}

// TestHerdr_TabCreate_PaneEnvLiteralChecksEveryCall: a Herdr built as a
// literal with PaneEnv has no shared check, so it runs the help, and warns
// when it cannot add --env, on every call.
func TestHerdr_TabCreate_PaneEnvLiteralChecksEveryCall(t *testing.T) {
	for name, tt := range map[string]struct {
		help      string
		wantArgv  []string
		wantLines int
	}{
		"help lists --env": {help: herdrTabCreateHelp, wantArgv: append(append([]string{}, tabCreateArgv...), "--env", "X=1")},
		"no --env":         {help: herdrTabCreateHelpNoEnv, wantArgv: tabCreateArgv, wantLines: 2},
	} {
		t.Run(name, func(t *testing.T) {
			f := &paneEnvFakeRunner{help: tt.help}
			var warn bytes.Buffer
			h := Herdr{R: f, PaneEnv: []string{"X=1"}, Warn: &warn}

			tabCreateTwice(t, h)

			if f.helpCalls != 2 {
				t.Fatalf("help ran %d times; want 2, one per call", f.helpCalls)
			}
			if !reflect.DeepEqual(f.tabCalls, [][]string{tt.wantArgv, tt.wantArgv}) {
				t.Fatalf("tab create argv = %v; want %v twice", f.tabCalls, tt.wantArgv)
			}
			if got := strings.Count(warn.String(), "\n"); got != tt.wantLines {
				t.Fatalf("warning lines = %d (%q); want %d", got, warn.String(), tt.wantLines)
			}
		})
	}
}

// TestHerdr_TabCreate_PaneEnvCanceledCallerIsNotCached: a help that fails
// because the caller's ctx has ended is neither cached nor warned about, so
// the next call checks again.
func TestHerdr_TabCreate_PaneEnvCanceledCallerIsNotCached(t *testing.T) {
	f := &paneEnvFakeRunner{helpFn: func(ctx context.Context) (string, error) {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		return herdrTabCreateHelp, nil
	}}
	var warn bytes.Buffer
	h := NewHerdr(f, SeatPaneEnv()...)
	h.Warn = &warn

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := h.TabCreate(canceled, "w3", "/tmp/cwd", "worker-1"); err != nil {
		t.Fatalf("TabCreate with the canceled ctx: %v (the fake tab create ignores ctx)", err)
	}
	if _, err := h.TabCreate(context.Background(), "w3", "/tmp/cwd", "worker-1"); err != nil {
		t.Fatalf("TabCreate: %v", err)
	}

	withEnv := append(append([]string{}, tabCreateArgv...), "--env", "AGMSG_SELF_NAME=off", "--env", "RALPH_WIP_CHECKPOINT=off")
	if !reflect.DeepEqual(f.tabCalls, [][]string{tabCreateArgv, withEnv}) {
		t.Fatalf("tab create argv = %v; want %v, then %v", f.tabCalls, tabCreateArgv, withEnv)
	}
	if f.helpCalls != 2 {
		t.Fatalf("help ran %d times; want 2, the canceled one not cached", f.helpCalls)
	}
	if warn.Len() != 0 {
		t.Fatalf("warning = %q; want none", warn.String())
	}
}

// TestHerdr_TabCreate_PaneEnvConcurrentCallsCheckOnce: TabCreate calls that
// run at the same time on copies of one NewHerdr value run the help once and
// all get --env.
func TestHerdr_TabCreate_PaneEnvConcurrentCallsCheckOnce(t *testing.T) {
	f := &paneEnvFakeRunner{helpFn: func(context.Context) (string, error) {
		time.Sleep(20 * time.Millisecond)
		return herdrTabCreateHelp, nil
	}}
	h := NewHerdr(f, SeatPaneEnv()...)

	const n = 8
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for range n {
		wg.Add(1)
		go func(c Herdr) {
			defer wg.Done()
			_, err := c.TabCreate(context.Background(), "w3", "/tmp/cwd", "worker-1")
			errs <- err
		}(h)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("TabCreate: %v", err)
		}
	}

	if f.helpCalls != 1 {
		t.Fatalf("help ran %d times; want 1", f.helpCalls)
	}
	withEnv := append(append([]string{}, tabCreateArgv...), "--env", "AGMSG_SELF_NAME=off", "--env", "RALPH_WIP_CHECKPOINT=off")
	for _, argv := range f.tabCalls {
		if !reflect.DeepEqual(argv, withEnv) {
			t.Fatalf("tab create argv = %v; want %v", argv, withEnv)
		}
	}
	if len(f.tabCalls) != n {
		t.Fatalf("tab create ran %d times; want %d", len(f.tabCalls), n)
	}
}

// TestHerdrHelpEnvFlag: --env counts only as a whole flag, at the end of the
// help too (ExecRunner trims the output, so a help whose last line is --env
// ends in it).
func TestHerdrHelpEnvFlag(t *testing.T) {
	for help, want := range map[string]bool{
		"      --env <KEY=VALUE>":             true,
		"  -e, --env <KEY=VALUE>":             true,
		"Usage: herdr tab create [--env KEY]": true,
		"--env=KEY=VALUE":                     true,
		"--label <TEXT>\n--env\n--focus":      true,
		"Options:\n      --env":               true,
		"      --envelope <FILE>":             false,
		"      --environment <NAME>":          false,
		"      --env-file <PATH>":             false,
		"      --no-env":                      false,
		"      --keep--env":                   false,
		"":                                    false,
	} {
		if got := herdrHelpEnvFlag.MatchString(help); got != want {
			t.Errorf("herdrHelpEnvFlag.MatchString(%q) = %v; want %v", help, got, want)
		}
	}
}

func TestHerdr_AgentStart(t *testing.T) {
	tests := []struct {
		name      string
		timeoutMS int
		agentArgs []string
		want      []string
	}{
		{
			name:      "no timeout, no agent args",
			timeoutMS: 0,
			agentArgs: nil,
			want:      []string{"agent", "start", "worker-1", "--kind", "claude", "--pane", "pane-1"},
		},
		{
			name:      "timeout only",
			timeoutMS: 5000,
			agentArgs: nil,
			want:      []string{"agent", "start", "worker-1", "--kind", "claude", "--pane", "pane-1", "--timeout", "5000"},
		},
		{
			name:      "agent args only",
			timeoutMS: 0,
			agentArgs: []string{"--model", "sonnet"},
			want:      []string{"agent", "start", "worker-1", "--kind", "claude", "--pane", "pane-1", "--", "--model", "sonnet"},
		},
		{
			name:      "timeout and agent args",
			timeoutMS: 5000,
			agentArgs: []string{"--model", "sonnet"},
			want:      []string{"agent", "start", "worker-1", "--kind", "claude", "--pane", "pane-1", "--timeout", "5000", "--", "--model", "sonnet"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeRunner{outputs: []string{"agent-1"}}
			h := Herdr{R: f}
			if _, err := h.AgentStart(context.Background(), "worker-1", "claude", "pane-1", tt.timeoutMS, tt.agentArgs); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c := f.lastCall(); !reflect.DeepEqual(c.args, tt.want) {
				t.Fatalf("argv mismatch: got %v, want %v", c.args, tt.want)
			}
		})
	}
}

func TestHerdr_AgentGet(t *testing.T) {
	f := &fakeRunner{outputs: []string{"status: running"}}
	h := Herdr{R: f}

	got, err := h.AgentGet(context.Background(), "agent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "status: running" {
		t.Fatalf("want %q, got %q", "status: running", got)
	}
	want := []string{"agent", "get", "agent-1"}
	if c := f.lastCall(); !reflect.DeepEqual(c.args, want) {
		t.Fatalf("argv mismatch: got %v, want %v", c.args, want)
	}
}

func TestHerdr_AgentWait(t *testing.T) {
	tests := []struct {
		name      string
		until     []string
		timeoutMS int
		want      []string
	}{
		{
			name:      "single until, no timeout",
			until:     []string{"idle"},
			timeoutMS: 0,
			want:      []string{"agent", "wait", "agent-1", "--until", "idle"},
		},
		{
			name:      "multiple until, with timeout",
			until:     []string{"idle", "error"},
			timeoutMS: 30000,
			want:      []string{"agent", "wait", "agent-1", "--until", "idle", "--until", "error", "--timeout", "30000"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeRunner{outputs: []string{"idle"}}
			h := Herdr{R: f}
			if _, err := h.AgentWait(context.Background(), "agent-1", tt.until, tt.timeoutMS); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c := f.lastCall(); !reflect.DeepEqual(c.args, tt.want) {
				t.Fatalf("argv mismatch: got %v, want %v", c.args, tt.want)
			}
		})
	}
}

// TestHerdr_AgentWait_DefensiveErrorEnvelope_ExitZero pins the defensive
// leg of checkHerdrEnvelopeError: if herdr ever returns an {"error":...}
// envelope with exit 0 (well-behaved herdr shouldn't, but the adapter must
// not trust that), AgentWait surfaces it as a Go error instead of returning
// it as if it were a normal "idle"-style status string.
func TestHerdr_AgentWait_DefensiveErrorEnvelope_ExitZero(t *testing.T) {
	f := &fakeRunner{outputs: []string{realErrorEnvelope}}
	h := Herdr{R: f}

	_, err := h.AgentWait(context.Background(), "agent-1", []string{"idle"}, 0)
	if err == nil {
		t.Fatal("expected error for an error envelope returned with exit 0, got nil")
	}
	if !strings.Contains(err.Error(), "workspace_not_found") {
		t.Fatalf("expected error to contain the envelope code, got: %v", err)
	}
}

// TestHerdr_AgentGet_PlainTextFallback pins that AgentGet's success path
// still returns non-JSON output verbatim (the fallback path, exercised end
// to end by the AgentGet/AgentWait/PaneRead stub outputs staying plain in
// internal/cli/org_test.go).
func TestHerdr_AgentGet_PlainTextFallback(t *testing.T) {
	f := &fakeRunner{outputs: []string{"status: idle"}}
	h := Herdr{R: f}

	got, err := h.AgentGet(context.Background(), "agent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "status: idle" {
		t.Fatalf("want %q, got %q", "status: idle", got)
	}
}

// TestHerdr_AgentStart_ErrorEnvelopeOnStdout_EnrichesWrappedError pins that
// when exit != 0 and stdout carries an error envelope, the returned error
// includes the envelope's code/message alongside the original
// stderr-wrapped error (rather than replacing it).
func TestHerdr_AgentStart_ErrorEnvelopeOnStdout_EnrichesWrappedError(t *testing.T) {
	f := &fakeRunner{
		outputs: []string{realErrorEnvelope},
		errs:    []error{errTest},
	}
	h := Herdr{R: f}

	_, err := h.AgentStart(context.Background(), "worker-1", "claude", "pane-1", 0, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errTest) {
		t.Fatalf("expected wrapped error to still satisfy errors.Is(err, errTest), got: %v", err)
	}
	if !strings.Contains(err.Error(), "workspace_not_found") {
		t.Fatalf("expected error to include the envelope code for readability, got: %v", err)
	}
}

// TestHerdr_AgentStart_ErrorEnvelopeOnStderrText_EnrichesWrappedError pins
// the stderr leg: ExecRunner folds captured stderr into err.Error() (see
// driver.go's ExecRunner.Run), so a herdr error envelope printed to stderr
// still needs to be extracted from there when stdout itself is empty.
func TestHerdr_AgentStart_ErrorEnvelopeOnStderrText_EnrichesWrappedError(t *testing.T) {
	wrapped := fmt.Errorf("herdr: exit status 1: %s", realErrorEnvelope)
	f := &fakeRunner{
		outputs: []string{""},
		errs:    []error{wrapped},
	}
	h := Herdr{R: f}

	_, err := h.AgentStart(context.Background(), "worker-1", "claude", "pane-1", 0, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "workspace_not_found") {
		t.Fatalf("expected error to include the envelope code extracted from stderr text, got: %v", err)
	}
}

func TestHerdr_PaneRead(t *testing.T) {
	f := &fakeRunner{outputs: []string{"pane output"}}
	h := Herdr{R: f}

	got, err := h.PaneRead(context.Background(), "pane-1", 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "pane output" {
		t.Fatalf("want %q, got %q", "pane output", got)
	}
	want := []string{"pane", "read", "pane-1", "--source", "recent", "--lines", "200", "--format", "text"}
	if c := f.lastCall(); !reflect.DeepEqual(c.args, want) {
		t.Fatalf("argv mismatch: got %v, want %v", c.args, want)
	}
}

func TestHerdr_PaneSendText(t *testing.T) {
	f := &fakeRunner{}
	h := Herdr{R: f}

	if err := h.PaneSendText(context.Background(), "pane-1", "hello there"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"pane", "send-text", "pane-1", "hello there"}
	if c := f.lastCall(); !reflect.DeepEqual(c.args, want) {
		t.Fatalf("argv mismatch: got %v, want %v", c.args, want)
	}
}

func TestHerdr_PaneSendKeys(t *testing.T) {
	f := &fakeRunner{}
	h := Herdr{R: f}

	if err := h.PaneSendKeys(context.Background(), "pane-1", "C-c", "Enter"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"pane", "send-keys", "pane-1", "C-c", "Enter"}
	if c := f.lastCall(); !reflect.DeepEqual(c.args, want) {
		t.Fatalf("argv mismatch: got %v, want %v", c.args, want)
	}
}

func TestHerdr_PaneRun(t *testing.T) {
	f := &fakeRunner{}
	h := Herdr{R: f}

	if err := h.PaneRun(context.Background(), "pane-1", "go test ./..."); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"pane", "run", "pane-1", "go test ./..."}
	if c := f.lastCall(); !reflect.DeepEqual(c.args, want) {
		t.Fatalf("argv mismatch: got %v, want %v", c.args, want)
	}
}

func TestHerdr_PaneSendText_RunnerErrorPropagates(t *testing.T) {
	f := &fakeRunner{errs: []error{errTest}}
	h := Herdr{R: f}

	if err := h.PaneSendText(context.Background(), "pane-1", "hi"); err == nil {
		t.Fatal("expected runner error to propagate, got nil")
	}
}

// errExit1 mimics ExecRunner's error for a herdr exit status 1 with empty
// stderr (the not-found replies print their envelope on stdout).
var errExit1 = errors.New("herdr: exit status 1")

// TestHerdr_PaneClose_WorkspaceClose pins the argv of both close
// subcommands and the error contract callers use to treat an already-closed
// target as closed: an ok envelope is success, a not-found envelope that
// arrives with a non-zero exit is an error IsNotFound recognises (the Runner
// error stays reachable via errors.Is), and every other failure is an error
// IsNotFound rejects.
func TestHerdr_PaneClose_WorkspaceClose(t *testing.T) {
	closers := []struct {
		name       string
		call       func(Herdr) error
		wantArgs   []string
		okEnvelope string
		notFound   string
	}{
		{
			name:       "pane close",
			call:       func(h Herdr) error { return h.PaneClose(context.Background(), "w3:p2") },
			wantArgs:   []string{"pane", "close", "w3:p2"},
			okEnvelope: realPaneCloseOKEnvelope,
			notFound:   realPaneNotFoundEnvelope,
		},
		{
			name:       "workspace close",
			call:       func(h Herdr) error { return h.WorkspaceClose(context.Background(), "w3") },
			wantArgs:   []string{"workspace", "close", "w3"},
			okEnvelope: realWorkspaceCloseOKEnvelope,
			notFound:   realWorkspaceNotFoundEnvelope,
		},
	}
	timeoutErr := fmt.Errorf("herdr: timed out: %w", context.DeadlineExceeded)
	connRefused := errors.New("herdr: exit status 1: connect: connection refused")
	for _, c := range closers {
		cases := []struct {
			name         string
			out          string
			runErr       error
			wantErr      bool
			wantNotFound bool
			wantIs       error // when non-nil, errors.Is(err, wantIs) must hold
		}{
			{name: "ok envelope", out: c.okEnvelope},
			{name: "empty stdout from a fake runner", out: ""},
			{name: "not-found envelope on stdout with exit 1", out: c.notFound, runErr: errExit1, wantErr: true, wantNotFound: true, wantIs: errExit1},
			{name: "not-found envelope folded into the stderr text", out: "", runErr: fmt.Errorf("herdr: exit status 1: %s", c.notFound), wantErr: true, wantNotFound: true},
			{name: "not-found envelope with exit 0", out: c.notFound, wantErr: true, wantNotFound: true},
			{name: "other error code", out: `{"error":{"code":"internal","message":"boom"},"id":"cli:x"}`, runErr: errExit1, wantErr: true, wantIs: errExit1},
			{name: "connection refused, no envelope", out: "", runErr: connRefused, wantErr: true, wantIs: connRefused},
			{name: "runner timeout", out: "", runErr: timeoutErr, wantErr: true, wantIs: context.DeadlineExceeded},
			{name: "malformed envelope with exit 1", out: `{"error":{"code":"pane_not_fou`, runErr: errExit1, wantErr: true, wantIs: errExit1},
		}
		for _, tt := range cases {
			t.Run(c.name+"/"+tt.name, func(t *testing.T) {
				f := &fakeRunner{outputs: []string{tt.out}, errs: []error{tt.runErr}}
				err := c.call(Herdr{R: f})
				if got := f.lastCall(); got.name != "herdr" || !reflect.DeepEqual(got.args, c.wantArgs) {
					t.Fatalf("argv mismatch: got name=%q args=%v, want name=herdr args=%v", got.name, got.args, c.wantArgs)
				}
				if len(f.calls) != 1 {
					t.Fatalf("want exactly one herdr call, got %d: %v", len(f.calls), f.calls)
				}
				if !tt.wantErr {
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					return
				}
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if got := IsNotFound(err); got != tt.wantNotFound {
					t.Fatalf("IsNotFound = %v, want %v (err: %v)", got, tt.wantNotFound, err)
				}
				if got := notFoundViaInterface(err); got != tt.wantNotFound {
					t.Fatalf("NotFound via errors.As = %v, want %v (err: %v)", got, tt.wantNotFound, err)
				}
				if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
					t.Fatalf("errors.Is(err, %v) = false, err: %v", tt.wantIs, err)
				}
				if !strings.Contains(err.Error(), "herdr "+c.name) {
					t.Fatalf("expected error to name the subcommand %q, got: %v", c.name, err)
				}
			})
		}
	}
}

// TestHerdr_PaneGet_TabGet_WorkspaceGet pins the argv of the three get
// subcommands, the fields each reads from a real success envelope, and the
// error contract the ownership check relies on: an unknown id is an error
// IsNotFound recognises, and every other failure -- another code, a Runner
// error, a timeout, stdout that is not an envelope, an envelope without the
// field -- is an error IsNotFound rejects, so it can never pass as a label.
func TestHerdr_PaneGet_TabGet_WorkspaceGet(t *testing.T) {
	getters := []struct {
		name     string
		call     func(Herdr) (string, error)
		wantArgs []string
		ok       string
		want     string
		notFound string
		missing  string // a success envelope without the field the getter reads
	}{
		{
			name: "pane get",
			call: func(h Herdr) (string, error) {
				pane, err := h.PaneGet(context.Background(), "w1:p2")
				return pane.TabID + " " + pane.WorkspaceID, err
			},
			wantArgs: []string{"pane", "get", "w1:p2"},
			ok:       realPaneGetEnvelope, want: "w1:t2 w1",
			notFound: realPaneGetNotFoundEnvelope,
			missing:  `{"id":"cli:pane:get","result":{"pane":{"pane_id":"w1:p2"},"type":"pane_info"}}`,
		},
		{
			name:     "tab get",
			call:     func(h Herdr) (string, error) { return h.TabGet(context.Background(), "w1:t2") },
			wantArgs: []string{"tab", "get", "w1:t2"},
			ok:       realTabGetEnvelope, want: "seatZ",
			notFound: realTabGetNotFoundEnvelope,
			missing:  `{"id":"cli:tab:get","result":{"type":"tab_info"}}`,
		},
		{
			name:     "workspace get",
			call:     func(h Herdr) (string, error) { return h.WorkspaceGet(context.Background(), "w1") },
			wantArgs: []string{"workspace", "get", "w1"},
			ok:       realWorkspaceGetEnvelope, want: "orgA",
			notFound: realWorkspaceGetNotFoundEnvelope,
			missing:  `{"id":"cli:workspace:get","result":{"type":"workspace_info"}}`,
		},
	}
	timeoutErr := fmt.Errorf("herdr: timed out: %w", context.DeadlineExceeded)
	connRefused := errors.New("herdr: exit status 1: connect: connection refused")
	for _, g := range getters {
		cases := []struct {
			name         string
			out          string
			runErr       error
			wantErr      bool
			wantNotFound bool
			wantIs       error // when non-nil, errors.Is(err, wantIs) must hold
		}{
			{name: "ok envelope", out: g.ok},
			{name: "not-found envelope on stdout with exit 1", out: g.notFound, runErr: errExit1, wantErr: true, wantNotFound: true, wantIs: errExit1},
			{name: "not-found envelope folded into the stderr text", out: "", runErr: fmt.Errorf("herdr: exit status 1: %s", g.notFound), wantErr: true, wantNotFound: true},
			{name: "not-found envelope with exit 0", out: g.notFound, wantErr: true, wantNotFound: true},
			{name: "other error code", out: `{"error":{"code":"internal","message":"boom"},"id":"cli:x"}`, runErr: errExit1, wantErr: true, wantIs: errExit1},
			{name: "connection refused, no envelope", out: "", runErr: connRefused, wantErr: true, wantIs: connRefused},
			{name: "runner timeout", out: "", runErr: timeoutErr, wantErr: true, wantIs: context.DeadlineExceeded},
			{name: "empty stdout with exit 0", out: "", wantErr: true},
			{name: "bare text with exit 0", out: "seatZ", wantErr: true},
			{name: "malformed envelope with exit 0", out: `{"result":{"tab":{"label":"seat`, wantErr: true},
			{name: "envelope without the field", out: g.missing, wantErr: true},
		}
		for _, tt := range cases {
			t.Run(g.name+"/"+tt.name, func(t *testing.T) {
				f := &fakeRunner{outputs: []string{tt.out}, errs: []error{tt.runErr}}
				got, err := g.call(Herdr{R: f})
				if c := f.lastCall(); c.name != "herdr" || !reflect.DeepEqual(c.args, g.wantArgs) {
					t.Fatalf("argv mismatch: got name=%q args=%v, want name=herdr args=%v", c.name, c.args, g.wantArgs)
				}
				if len(f.calls) != 1 {
					t.Fatalf("want exactly one herdr call, got %d: %v", len(f.calls), f.calls)
				}
				if !tt.wantErr {
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if got != g.want {
						t.Fatalf("got %q, want %q", got, g.want)
					}
					return
				}
				if err == nil {
					t.Fatalf("expected error, got nil (result %q)", got)
				}
				if got := IsNotFound(err); got != tt.wantNotFound {
					t.Fatalf("IsNotFound = %v, want %v (err: %v)", got, tt.wantNotFound, err)
				}
				if got := notFoundViaInterface(err); got != tt.wantNotFound {
					t.Fatalf("NotFound via errors.As = %v, want %v (err: %v)", got, tt.wantNotFound, err)
				}
				if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
					t.Fatalf("errors.Is(err, %v) = false, err: %v", tt.wantIs, err)
				}
				if !strings.Contains(err.Error(), "herdr "+g.name) {
					t.Fatalf("expected error to name the subcommand %q, got: %v", g.name, err)
				}
			})
		}
	}
}

// TestHerdr_TabGet_WorkspaceGet_UnlabelledIsEmptyLabel: a tab or workspace
// whose envelope has no label reads as the empty label, not as an error, so
// the caller's label comparison (never against an empty id) fails closed.
func TestHerdr_TabGet_WorkspaceGet_UnlabelledIsEmptyLabel(t *testing.T) {
	tab := &fakeRunner{outputs: []string{`{"id":"cli:tab:get","result":{"tab":{"tab_id":"w1:t2"},"type":"tab_info"}}`}}
	if label, err := (Herdr{R: tab}).TabGet(context.Background(), "w1:t2"); err != nil || label != "" {
		t.Fatalf("TabGet = %q, %v; want an empty label and no error", label, err)
	}
	ws := &fakeRunner{outputs: []string{`{"id":"cli:workspace:get","result":{"type":"workspace_info","workspace":{"workspace_id":"w1"}}}`}}
	if label, err := (Herdr{R: ws}).WorkspaceGet(context.Background(), "w1"); err != nil || label != "" {
		t.Fatalf("WorkspaceGet = %q, %v; want an empty label and no error", label, err)
	}
}

// notFoundViaInterface is how internal/org reads the not-found distinction
// without importing this package: a one-method interface of its own and
// errors.As. It must agree with IsNotFound for every error.
func notFoundViaInterface(err error) bool {
	var nf interface{ NotFound() bool }
	return errors.As(err, &nf) && nf.NotFound()
}

// TestIsNotFound pins IsNotFound on its own: the three not-found codes match
// directly and through %w wrapping, while nil, other codes, and a plain error
// whose text merely contains a not-found code do not. The NewHerdrError
// cases are the contract internal/org's fakeHerdr builds its "already
// closed" replies on. Each case also checks the error type's NotFound
// method read through notFoundViaInterface, which must give the same answer.
func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "pane_not_found", err: NewHerdrError(HerdrCodePaneNotFound, "pane w1:p9 not found"), want: true},
		{name: "tab_not_found", err: NewHerdrError(HerdrCodeTabNotFound, "tab w1:t9 not found"), want: true},
		{name: "workspace_not_found", err: NewHerdrError(HerdrCodeWorkspaceNotFound, "workspace w9 not found"), want: true},
		{name: "wrapped pane_not_found", err: fmt.Errorf("stop seat reviewer: %w", NewHerdrError(HerdrCodePaneNotFound, "gone")), want: true},
		{name: "doubly wrapped workspace_not_found", err: fmt.Errorf("disband: %w", fmt.Errorf("close: %w", NewHerdrError(HerdrCodeWorkspaceNotFound, "gone"))), want: true},
		{name: "other code", err: NewHerdrError("agent_pane_busy", "busy"), want: false},
		{name: "agent_not_found is no closed pane, tab or workspace", err: NewHerdrError(HerdrCodeAgentNotFound, "agent a1 not found"), want: false},
		{name: "plain error text containing the code", err: errors.New("herdr: pane_not_found: pane w1:p9 not found"), want: false},
		{name: "unrelated sentinel", err: errTest, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Fatalf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
			if got := notFoundViaInterface(tt.err); got != tt.want {
				t.Fatalf("NotFound via errors.As (%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// agentNotFoundViaInterface is how internal/org reads agent_not_found
// (isAgentNotFound): a one-method interface of its own and errors.As. It must
// agree with IsAgentNotFound for every error.
func agentNotFoundViaInterface(err error) bool {
	var nf interface{ AgentNotFound() bool }
	return errors.As(err, &nf) && nf.AgentNotFound()
}

// TestIsAgentNotFound pins IsAgentNotFound on its own (plan
// 2026-10-18-org-director-liveness, D3): agent_not_found matches directly and
// through %w wrapping, while nil, the pane, tab and workspace not-found codes,
// another code, and a plain error whose text merely contains the code do not.
// Each case also checks the AgentNotFound method read through
// agentNotFoundViaInterface, which must give the same answer.
func TestIsAgentNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "agent_not_found", err: NewHerdrError(HerdrCodeAgentNotFound, "agent a1 not found"), want: true},
		{name: "wrapped agent_not_found", err: fmt.Errorf("check d1: %w", NewHerdrError(HerdrCodeAgentNotFound, "gone")), want: true},
		{name: "pane_not_found", err: NewHerdrError(HerdrCodePaneNotFound, "pane w1:p9 not found"), want: false},
		{name: "tab_not_found", err: NewHerdrError(HerdrCodeTabNotFound, "tab w1:t9 not found"), want: false},
		{name: "workspace_not_found", err: NewHerdrError(HerdrCodeWorkspaceNotFound, "workspace w9 not found"), want: false},
		{name: "other code", err: NewHerdrError("agent_pane_busy", "busy"), want: false},
		{name: "plain error text containing the code", err: errors.New(agentNotFoundEnvelopeText), want: false},
		{name: "unrelated sentinel", err: errTest, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAgentNotFound(tt.err); got != tt.want {
				t.Fatalf("IsAgentNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
			if got := agentNotFoundViaInterface(tt.err); got != tt.want {
				t.Fatalf("AgentNotFound via errors.As (%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// TestHerdr_AgentGet_AgentNotFoundIsReadable pins AgentGet's failure contract
// (plan 2026-10-18-org-director-liveness, D3): herdr's agent_not_found
// envelope, printed on stdout or on stderr (which ExecRunner folds into the
// Runner error), comes back as an error for which IsAgentNotFound is true and
// IsNotFound is false, that still wraps the Runner error, and whose text is
// the one checkHerdrEnvelopeError gave AgentGet before it wrapped the
// envelope. A failure with no envelope is the Runner error as it was.
func TestHerdr_AgentGet_AgentNotFoundIsReadable(t *testing.T) {
	stderrErr := fmt.Errorf("herdr: exit status 1: %s", realAgentNotFoundEnvelope)
	tests := []struct {
		name         string
		out          string
		runErr       error
		wantText     string
		wantNotFound bool
	}{
		{name: "envelope on stdout", out: realAgentNotFoundEnvelope, runErr: errTest,
			wantText: errTest.Error() + " (" + agentNotFoundEnvelopeText + ")", wantNotFound: true},
		{name: "envelope on stderr, folded into the Runner error", out: "", runErr: stderrErr,
			wantText: stderrErr.Error() + " (" + agentNotFoundEnvelopeText + ")", wantNotFound: true},
		{name: "no envelope", out: "", runErr: errTest, wantText: errTest.Error(), wantNotFound: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeRunner{outputs: []string{tt.out}, errs: []error{tt.runErr}}
			_, err := Herdr{R: f}.AgentGet(context.Background(), "director_0123456789ab")
			if err == nil {
				t.Fatal("AgentGet: want an error, got nil")
			}
			if err.Error() != tt.wantText {
				t.Fatalf("AgentGet error = %q, want %q", err.Error(), tt.wantText)
			}
			if _, before := checkHerdrEnvelopeError(tt.out, tt.runErr); before.Error() != err.Error() {
				t.Fatalf("AgentGet error = %q, but checkHerdrEnvelopeError gives %q: the text must not change", err.Error(), before.Error())
			}
			if got := IsAgentNotFound(err); got != tt.wantNotFound {
				t.Fatalf("IsAgentNotFound(%v) = %v, want %v", err, got, tt.wantNotFound)
			}
			if got := agentNotFoundViaInterface(err); got != tt.wantNotFound {
				t.Fatalf("AgentNotFound via errors.As (%v) = %v, want %v", err, got, tt.wantNotFound)
			}
			if IsNotFound(err) || notFoundViaInterface(err) {
				t.Fatalf("IsNotFound(%v) = true: agent_not_found must not read as a closed pane, tab or workspace", err)
			}
			if !errors.Is(err, tt.runErr) {
				t.Fatalf("AgentGet error %v no longer wraps the Runner error %v", err, tt.runErr)
			}
		})
	}
}

// TestHerdr_AgentStartAndWait_KeepTheEnvelopeAsText pins that only AgentGet
// wraps the envelope of a failed call: AgentStart and AgentWait fold a
// pane_not_found or agent_not_found envelope into the text as before, so
// neither IsNotFound nor IsAgentNotFound reads it, and the Runner error is
// still wrapped.
func TestHerdr_AgentStartAndWait_KeepTheEnvelopeAsText(t *testing.T) {
	envelopes := map[string]string{
		realPaneNotFoundEnvelope:  "herdr: pane_not_found: pane w99:p99 not found",
		realAgentNotFoundEnvelope: agentNotFoundEnvelopeText,
	}
	for env, envText := range envelopes {
		calls := map[string]func(h Herdr) error{
			"AgentStart": func(h Herdr) error {
				_, err := h.AgentStart(context.Background(), "worker-1", "claude", "pane-1", 0, nil)
				return err
			},
			"AgentWait": func(h Herdr) error {
				_, err := h.AgentWait(context.Background(), "worker-1", []string{"idle"}, 0)
				return err
			},
		}
		for name, call := range calls {
			err := call(Herdr{R: &fakeRunner{outputs: []string{env}, errs: []error{errTest}}})
			want := errTest.Error() + " (" + envText + ")"
			switch {
			case err == nil:
				t.Fatalf("%s with %s: want an error, got nil", name, env)
			case err.Error() != want:
				t.Fatalf("%s error = %q, want %q", name, err.Error(), want)
			case !errors.Is(err, errTest):
				t.Fatalf("%s error %v no longer wraps the Runner error", name, err)
			case IsNotFound(err), IsAgentNotFound(err):
				t.Fatalf("%s error %v reads as a not-found code; only AgentGet wraps the envelope", name, err)
			}
		}
	}
}

// realServerStatusRunning and realServerStatusNotRunning are the stdout of
// `herdr status server --json` on herdr v0.7.5, captured live on 2026-11-06
// (spec 2026-11-06-org-herdr-autostart-panes, Research findings): a bare JSON
// object with exit 0, for a running server and for no server.
const realServerStatusRunning = `{"status":"running","running":true,"version":"0.7.5","protocol":17,"capabilities":{"live_handoff":true,"detached_server_daemon":false},"compatible":true,"socket":"/Users/x/.config/herdr/herdr.sock","session":null,"restart_needed":false}`

const realServerStatusNotRunning = `{"status":"not_running","running":false,"version":null,"protocol":null,"capabilities":null,"compatible":null,"socket":"/tmp/hz-none.sock","session":null,"restart_needed":false}`

// wantServerStatusArgv is the one herdr call ServerStatus makes.
var wantServerStatusArgv = []string{"status", "server", "--json"}

// checkServerStatusCall fails t unless f saw exactly one call, the argv of
// `herdr status server --json`.
func checkServerStatusCall(t *testing.T, f *fakeRunner) {
	t.Helper()
	if len(f.calls) != 1 {
		t.Fatalf("want exactly one herdr call, got %d: %v", len(f.calls), f.calls)
	}
	if c := f.calls[0]; c.name != "herdr" || !reflect.DeepEqual(c.args, wantServerStatusArgv) {
		t.Fatalf("argv mismatch: got name=%q args=%v, want name=herdr args=%v", c.name, c.args, wantServerStatusArgv)
	}
}

// TestHerdr_ServerStatus_ReadsTheBareJSONObject pins that ServerStatus runs
// exactly `herdr status server --json` and reads running, compatible and
// socket from the bare object herdr prints: a running server, no server
// (compatible null reads as nil, not as false), a server herdr calls
// incompatible (false stays false), and the object with the whitespace
// around it that a Runner other than ExecRunner might leave.
func TestHerdr_ServerStatus_ReadsTheBareJSONObject(t *testing.T) {
	yes, no := true, false
	tests := []struct {
		name string
		out  string
		want HerdrServerStatus
	}{
		{name: "running server", out: realServerStatusRunning,
			want: HerdrServerStatus{Running: true, Compatible: &yes, Socket: "/Users/x/.config/herdr/herdr.sock"}},
		{name: "no server, compatible null", out: realServerStatusNotRunning,
			want: HerdrServerStatus{Running: false, Compatible: nil, Socket: "/tmp/hz-none.sock"}},
		{name: "running server herdr calls incompatible", out: `{"running":true,"compatible":false,"socket":"/s.sock"}`,
			want: HerdrServerStatus{Running: true, Compatible: &no, Socket: "/s.sock"}},
		{name: "object inside whitespace", out: "\n  " + realServerStatusNotRunning + "\n",
			want: HerdrServerStatus{Running: false, Compatible: nil, Socket: "/tmp/hz-none.sock"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeRunner{outputs: []string{tt.out}}
			got, err := Herdr{R: f}.ServerStatus(context.Background())
			checkServerStatusCall(t, f)
			if err != nil {
				t.Fatalf("ServerStatus: unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ServerStatus = %s, want %s", formatServerStatus(got), formatServerStatus(tt.want))
			}
		})
	}
}

// formatServerStatus prints s with Compatible's value rather than its
// pointer, so a mismatch in a failure message is readable.
func formatServerStatus(s HerdrServerStatus) string {
	compatible := "nil"
	if s.Compatible != nil {
		compatible = strconv.FormatBool(*s.Compatible)
	}
	return fmt.Sprintf("{Running:%v Compatible:%s Socket:%q}", s.Running, compatible, s.Socket)
}

// TestHerdr_ServerStatus_UnreadableOutputIsAnError pins that an exit-0 answer
// ServerStatus cannot read is an error, never a status: the caller decides
// from Running whether to start a server, so a guess (an empty struct reads
// as "not running") must not come back. None of these is
// ErrHerdrStatusJSONUnsupported. The envelope case is the shape of herdr's
// other subcommands, whose `running` would sit under `result`. Output the
// JSON decoder rejects (wantDecodeErr) comes back with the decoder's error
// wrapped, so the message says the JSON is broken rather than that a key is
// missing.
func TestHerdr_ServerStatus_UnreadableOutputIsAnError(t *testing.T) {
	tests := []struct {
		name          string
		out           string
		wantDecodeErr bool
	}{
		{name: "truncated JSON", out: `{"status":"running","running":tr`, wantDecodeErr: true},
		{name: "empty stdout", out: "", wantDecodeErr: true},
		{name: "plain text", out: "server running", wantDecodeErr: true},
		{name: "JSON without running", out: `{"status":"running","compatible":true,"socket":"/s.sock"}`},
		{name: "running is null", out: `{"running":null,"socket":"/s.sock"}`},
		{name: "running is not a bool", out: `{"running":"yes","socket":"/s.sock"}`, wantDecodeErr: true},
		{name: "a JSON array", out: `[{"running":true}]`, wantDecodeErr: true},
		{name: "JSON null", out: `null`},
		{name: "two objects", out: realServerStatusRunning + realServerStatusNotRunning, wantDecodeErr: true},
		{name: "envelope with running under result", out: `{"id":"cli:status:server","result":{"running":true}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeRunner{outputs: []string{tt.out}}
			got, err := Herdr{R: f}.ServerStatus(context.Background())
			checkServerStatusCall(t, f)
			if err == nil {
				t.Fatalf("ServerStatus: want an error for %q, got %s", tt.out, formatServerStatus(got))
			}
			if errors.Is(err, ErrHerdrStatusJSONUnsupported) {
				t.Fatalf("ServerStatus error %v reads as ErrHerdrStatusJSONUnsupported; only `unknown option: --json` may", err)
			}
			if !strings.HasPrefix(err.Error(), "herdr status server: ") {
				t.Fatalf("ServerStatus error %q does not name the subcommand", err.Error())
			}
			var syntaxErr *json.SyntaxError
			var typeErr *json.UnmarshalTypeError
			if decodeErr := errors.As(err, &syntaxErr) || errors.As(err, &typeErr); decodeErr != tt.wantDecodeErr {
				t.Fatalf("ServerStatus error %v wraps a JSON decode error = %v, want %v", err, decodeErr, tt.wantDecodeErr)
			}
		})
	}
}

// TestHerdr_ServerStatus_RunnerErrorIsWrapped pins that a failed run
// (herdr exited non-zero, the call timed out) is an error that still wraps
// the Runner error, even when herdr printed a status object before it failed,
// and is not ErrHerdrStatusJSONUnsupported.
func TestHerdr_ServerStatus_RunnerErrorIsWrapped(t *testing.T) {
	connRefused := errors.New("herdr: exit status 1: connect: connection refused")
	timeoutErr := fmt.Errorf("herdr: timed out: %w", context.DeadlineExceeded)
	tests := []struct {
		name   string
		out    string
		runErr error
		wantIs error
	}{
		{name: "exit 1 with no output", out: "", runErr: connRefused, wantIs: connRefused},
		{name: "exit 1 after a status object", out: realServerStatusRunning, runErr: errExit1, wantIs: errExit1},
		{name: "timeout", out: "", runErr: timeoutErr, wantIs: context.DeadlineExceeded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeRunner{outputs: []string{tt.out}, errs: []error{tt.runErr}}
			got, err := Herdr{R: f}.ServerStatus(context.Background())
			checkServerStatusCall(t, f)
			if err == nil {
				t.Fatalf("ServerStatus: want an error, got %s", formatServerStatus(got))
			}
			if !errors.Is(err, tt.wantIs) {
				t.Fatalf("errors.Is(%v, %v) = false: the Runner error must stay reachable", err, tt.wantIs)
			}
			if errors.Is(err, ErrHerdrStatusJSONUnsupported) {
				t.Fatalf("ServerStatus error %v reads as ErrHerdrStatusJSONUnsupported", err)
			}
			if want := "herdr status server: " + tt.runErr.Error(); err.Error() != want {
				t.Fatalf("ServerStatus error = %q, want %q", err.Error(), want)
			}
		})
	}
}

// TestHerdr_ServerStatus_UnknownJSONOptionIsUnsupported pins how an old herdr
// that does not know `status server --json` is told apart (plan
// 2026-11-06-org-herdr-autostart, D3): `unknown option: --json` on stdout, or
// only in the Runner error's text where ExecRunner folds stderr, is an error
// for which errors.Is(err, ErrHerdrStatusJSONUnsupported) holds and which
// still wraps the Runner error.
func TestHerdr_ServerStatus_UnknownJSONOptionIsUnsupported(t *testing.T) {
	stderrErr := errors.New("herdr: exit status 2: unknown option: --json")
	tests := []struct {
		name   string
		out    string
		runErr error
	}{
		{name: "on stdout with a Runner error", out: "unknown option: --json", runErr: errExit1},
		{name: "only in the Runner error's text", out: "", runErr: stderrErr},
		{name: "followed by usage text on stdout", out: "unknown option: --json\nusage: herdr status server", runErr: errExit1},
		{name: "on stdout with exit 0", out: "unknown option: --json", runErr: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeRunner{outputs: []string{tt.out}, errs: []error{tt.runErr}}
			got, err := Herdr{R: f}.ServerStatus(context.Background())
			checkServerStatusCall(t, f)
			if err == nil {
				t.Fatalf("ServerStatus: want an error, got %s", formatServerStatus(got))
			}
			if !errors.Is(err, ErrHerdrStatusJSONUnsupported) {
				t.Fatalf("errors.Is(%v, ErrHerdrStatusJSONUnsupported) = false", err)
			}
			if tt.runErr != nil && !errors.Is(err, tt.runErr) {
				t.Fatalf("ServerStatus error %v no longer wraps the Runner error %v", err, tt.runErr)
			}
		})
	}
}

// TestHerdr_ServerStatus_OtherUnknownOptionIsNotUnsupported pins that only
// herdr's line about --json itself is ErrHerdrStatusJSONUnsupported: another
// unknown option, or a longer flag that starts with --json, is a plain
// failure that wraps the Runner error, so the caller does not pass a status
// it could not read.
func TestHerdr_ServerStatus_OtherUnknownOptionIsNotUnsupported(t *testing.T) {
	tests := []struct {
		name   string
		out    string
		runErr error
	}{
		{name: "--foo on stdout", out: "unknown option: --foo", runErr: errExit1},
		{name: "--foo in the Runner error's text", out: "", runErr: errors.New("herdr: exit status 2: unknown option: --foo")},
		{name: "--jsonl on stdout", out: "unknown option: --jsonl", runErr: errExit1},
		{name: "--json-lines in the Runner error's text", out: "", runErr: errors.New("herdr: exit status 2: unknown option: --json-lines")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeRunner{outputs: []string{tt.out}, errs: []error{tt.runErr}}
			got, err := Herdr{R: f}.ServerStatus(context.Background())
			checkServerStatusCall(t, f)
			if err == nil {
				t.Fatalf("ServerStatus: want an error, got %s", formatServerStatus(got))
			}
			if errors.Is(err, ErrHerdrStatusJSONUnsupported) {
				t.Fatalf("ServerStatus error %v reads as ErrHerdrStatusJSONUnsupported", err)
			}
			if !errors.Is(err, tt.runErr) {
				t.Fatalf("ServerStatus error %v no longer wraps the Runner error %v", err, tt.runErr)
			}
		})
	}
}

// herdrWorkspaceCreateHelp and herdrPaneSplitHelp are `herdr workspace
// create --help` and `herdr pane split --help` of herdr 0.7.5 (run
// 2026-11-06), with their blank description lines left out, as
// herdrTabCreateHelp is.
const herdrWorkspaceCreateHelp = `Create a workspace

Usage: herdr workspace create [OPTIONS]

Options:
      --cwd <PATH>
      --label <TEXT>
      --env <KEY=VALUE>
          Set an environment variable for the launched process
      --focus
      --no-focus`

const herdrPaneSplitHelp = `Split a pane

Usage: herdr pane split [OPTIONS] [PANE_ID]

Arguments:
  [PANE_ID]

Options:
      --pane <ID>
      --current
      --direction <DIRECTION>
          [possible values: right, down]
      --ratio <FLOAT>
      --cwd <PATH>
      --env <KEY=VALUE>
          Set an environment variable for the launched process
      --focus
      --no-focus`

// realPaneSplitEnvelope is herdr 0.7.5's `pane split` reply on an isolated
// server (2026-11-06; plan 2026-11-06-org-herdr-single-tab), with the
// fields ralph does not read left out.
const realPaneSplitEnvelope = `{"id":"cli:pane:split","result":{"pane":{"pane_id":"w1:p2","tab_id":"w1:t1","workspace_id":"w1"},"type":"pane_info"}}`

// paneListEnvelope is a `pane list --workspace w1` reply in the shape of
// herdr 0.7.5's response schema, which the binary embeds (result
// {"panes":[PaneInfo...],"type":"pane_list"}; a PaneInfo always has
// pane_id, tab_id and workspace_id, and a label that is a string or null).
// It was not captured from a server. The three panes have a label, a null
// label, and no label key.
const paneListEnvelope = `{"id":"cli:pane:list","result":{"panes":[` +
	`{"label":"leader","pane_id":"w1:p1","tab_id":"w1:t1","workspace_id":"w1"},` +
	`{"label":null,"pane_id":"w1:p2","tab_id":"w1:t1","workspace_id":"w1"},` +
	`{"pane_id":"w1:p3","tab_id":"w1:t2","workspace_id":"w1"}],"type":"pane_list"}}`

// subcmdFakeRunner answers `herdr <sub> --help` with help[sub] and
// helpErr[sub], sub being the words before --help ("pane split"), and
// every other call with replies["<first> <second>"] (the subcommand's two
// words). It counts the helps by sub and records every other call's argv,
// in order.
type subcmdFakeRunner struct {
	mu        sync.Mutex
	help      map[string]string
	helpErr   map[string]error
	replies   map[string]string
	helpCalls map[string]int
	calls     [][]string
}

func (f *subcmdFakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if n := len(args); n > 0 && args[n-1] == "--help" {
		sub := strings.Join(args[:n-1], " ")
		if f.helpCalls == nil {
			f.helpCalls = map[string]int{}
		}
		f.helpCalls[sub]++
		return f.help[sub], f.helpErr[sub]
	}
	f.calls = append(f.calls, append([]string{name}, args...))
	if len(args) < 2 {
		return "", nil
	}
	return f.replies[args[0]+" "+args[1]], nil
}

// subcmdReplies are the replies subcmdFakeRunner gives the three
// subcommands that make a pane.
var subcmdReplies = map[string]string{
	"workspace create": realWorkspaceCreateEnvelope,
	"tab create":       realTabCreateEnvelope,
	"pane split":       realPaneSplitEnvelope,
}

// The argv the calls below run, before any --env, and the --env entries of
// SeatPaneEnv followed by a claude seat's PrimaryCLIEntry.
var (
	workspaceCreateArgv = []string{"herdr", "workspace", "create", "--cwd", "/tmp/cwd", "--label", "org-x"}
	paneSplitArgv       = []string{"herdr", "pane", "split", "w1:p1", "--direction", "right", "--ratio", "0.5", "--cwd", "/tmp/cwd"}
	seatEnvArgs         = []string{"--env", "AGMSG_SELF_NAME=off", "--env", "RALPH_WIP_CHECKPOINT=off", "--env", "RALPH_PRIMARY_CLI=claude"}
)

// argvWith returns argv followed by each of tails.
func argvWith(argv []string, tails ...[]string) []string {
	out := slices.Clone(argv)
	for _, tail := range tails {
		out = append(out, tail...)
	}
	return out
}

// TestHerdr_WorkspaceCreate_NoExtraEnvAddsPaneEnvAlone pins the --env rule
// of `workspace create` once the headless director runs in the root pane
// (plan 2026-11-06-org-herdr-single-tab, D7): a WorkspaceCreate given no
// extraEnv, as director start calls it, adds an --env for each PaneEnv entry
// alone, after --label, on a herdr whose help lists --env, as TabCreate does.
// A NewHerdr value runs `workspace create --help` once and a literal on
// every call; neither warns. A literal with no PaneEnv runs the bare argv
// and no help.
func TestHerdr_WorkspaceCreate_NoExtraEnvAddsPaneEnvAlone(t *testing.T) {
	paneEnvArgv := argvWith(workspaceCreateArgv, seatEnvArgs[:4])
	for name, tc := range map[string]struct {
		build     func(r Runner) Herdr
		wantArgv  []string
		wantHelps map[string]int
	}{
		"NewHerdr with SeatPaneEnv": {func(r Runner) Herdr { return NewHerdr(r, SeatPaneEnv()...) }, paneEnvArgv, map[string]int{"workspace create": 1}},
		"a literal with PaneEnv":    {func(r Runner) Herdr { return Herdr{R: r, PaneEnv: SeatPaneEnv()} }, paneEnvArgv, map[string]int{"workspace create": 2}},
		"a literal with no PaneEnv": {func(r Runner) Herdr { return Herdr{R: r} }, workspaceCreateArgv, nil},
	} {
		t.Run(name, func(t *testing.T) {
			f := &subcmdFakeRunner{help: map[string]string{"workspace create": herdrWorkspaceCreateHelp}, replies: subcmdReplies}
			var warn bytes.Buffer
			h := tc.build(f)
			h.Warn = &warn

			for range 2 {
				ws, err := h.WorkspaceCreate(context.Background(), "/tmp/cwd", "org-x")
				if want := (HerdrWorkspace{WorkspaceID: "w3", TabID: "w3:t1", RootPaneID: "w3:p1"}); err != nil || ws != want {
					t.Fatalf("WorkspaceCreate = %+v, %v; want %+v", ws, err, want)
				}
			}
			if want := [][]string{tc.wantArgv, tc.wantArgv}; !reflect.DeepEqual(f.calls, want) {
				t.Fatalf("herdr calls = %q; want %q", f.calls, want)
			}
			if !reflect.DeepEqual(f.helpCalls, tc.wantHelps) || warn.Len() != 0 {
				t.Fatalf("helps %v, warning %q; want helps %v and no warning", f.helpCalls, warn.String(), tc.wantHelps)
			}
		})
	}
}

// TestHerdr_WorkspaceCreate_ExtraEnvAddsPaneEnvWhenHelpListsIt pins AC10 of
// plan 2026-11-06-org-herdr-single-tab for `workspace create`: given
// extraEnv, on a herdr whose `workspace create --help` lists --env, the call
// adds an --env for each PaneEnv entry and then each extraEnv entry, after
// --label, and the help runs once for a NewHerdr value and its copy.
func TestHerdr_WorkspaceCreate_ExtraEnvAddsPaneEnvWhenHelpListsIt(t *testing.T) {
	f := &subcmdFakeRunner{help: map[string]string{"workspace create": herdrWorkspaceCreateHelp}, replies: subcmdReplies}
	var warn bytes.Buffer
	h := NewHerdr(f, SeatPaneEnv()...)
	h.Warn = &warn
	var copied interface {
		WorkspaceCreate(ctx context.Context, cwd, label string, extraEnv ...string) (HerdrWorkspace, error)
	} = h

	for i, create := range []func(ctx context.Context, cwd, label string, extraEnv ...string) (HerdrWorkspace, error){h.WorkspaceCreate, copied.WorkspaceCreate} {
		ws, err := create(context.Background(), "/tmp/cwd", "org-x", PrimaryCLIEntry("claude"))
		if want := (HerdrWorkspace{WorkspaceID: "w3", TabID: "w3:t1", RootPaneID: "w3:p1"}); err != nil || ws != want {
			t.Fatalf("WorkspaceCreate #%d = %+v, %v; want %+v", i+1, ws, err, want)
		}
	}
	want := argvWith(workspaceCreateArgv, seatEnvArgs)
	if !reflect.DeepEqual(f.calls, [][]string{want, want}) {
		t.Fatalf("herdr calls = %q; want %q twice", f.calls, want)
	}
	if !reflect.DeepEqual(f.helpCalls, map[string]int{"workspace create": 1}) {
		t.Fatalf("helps = %v; want `workspace create --help` once", f.helpCalls)
	}
	if !slices.Equal(h.PaneEnv, SeatPaneEnv()) || warn.Len() != 0 {
		t.Fatalf("PaneEnv %q, warning %q; want SeatPaneEnv unchanged and no warning", h.PaneEnv, warn.String())
	}
}

// TestHerdr_PaneSplit_ArgvAndReply pins the argv of `pane split` for both
// directions, a Herdr with no PaneEnv running no help, and the ids it reads
// from herdr 0.7.5's reply.
func TestHerdr_PaneSplit_ArgvAndReply(t *testing.T) {
	for _, direction := range []string{HerdrSplitRight, HerdrSplitDown} {
		t.Run(direction, func(t *testing.T) {
			f := &subcmdFakeRunner{replies: subcmdReplies}
			got, err := (Herdr{R: f}).PaneSplit(context.Background(), "w1:p1", direction, "/tmp/cwd")
			if want := (HerdrPaneRef{PaneID: "w1:p2", TabID: "w1:t1"}); err != nil || got != want {
				t.Fatalf("PaneSplit = %+v, %v; want %+v", got, err, want)
			}
			want := []string{"herdr", "pane", "split", "w1:p1", "--direction", direction, "--ratio", "0.5", "--cwd", "/tmp/cwd", "--no-focus"}
			if !reflect.DeepEqual(f.calls, [][]string{want}) || len(f.helpCalls) != 0 {
				t.Fatalf("herdr calls = %q, helps %v; want %q alone", f.calls, f.helpCalls, want)
			}
		})
	}
	if HerdrSplitRight != "right" || HerdrSplitDown != "down" {
		t.Fatalf("directions = %q, %q; want herdr's right and down", HerdrSplitRight, HerdrSplitDown)
	}
}

// TestHerdr_PaneSplit_PaneEnvGoesBeforeNoFocus pins AC10 of plan
// 2026-11-06-org-herdr-single-tab for `pane split`: on a herdr whose
// `pane split --help` lists --env, the PaneEnv entries and then the call's
// extraEnv come after --cwd and before --no-focus; a later call without
// extraEnv gets PaneEnv alone, and the help runs once.
func TestHerdr_PaneSplit_PaneEnvGoesBeforeNoFocus(t *testing.T) {
	f := &subcmdFakeRunner{help: map[string]string{"pane split": herdrPaneSplitHelp}, replies: subcmdReplies}
	var warn bytes.Buffer
	h := NewHerdr(f, SeatPaneEnv()...)
	h.Warn = &warn

	if _, err := h.PaneSplit(context.Background(), "w1:p1", HerdrSplitRight, "/tmp/cwd", PrimaryCLIEntry("claude")); err != nil {
		t.Fatalf("PaneSplit: %v", err)
	}
	if _, err := h.PaneSplit(context.Background(), "w1:p1", HerdrSplitRight, "/tmp/cwd"); err != nil {
		t.Fatalf("PaneSplit without extraEnv: %v", err)
	}
	noFocus := []string{"--no-focus"}
	want := [][]string{
		argvWith(paneSplitArgv, seatEnvArgs, noFocus),
		argvWith(paneSplitArgv, seatEnvArgs[:4], noFocus),
	}
	if !reflect.DeepEqual(f.calls, want) {
		t.Fatalf("herdr calls = %q; want %q", f.calls, want)
	}
	if !reflect.DeepEqual(f.helpCalls, map[string]int{"pane split": 1}) || warn.Len() != 0 {
		t.Fatalf("helps %v, warning %q; want `pane split --help` once and no warning", f.helpCalls, warn.String())
	}
}

// TestHerdr_EnvCheck_WithoutEnvFlagWarnsOncePerSubcommand pins AC10 of plan
// 2026-11-06-org-herdr-single-tab for a herdr without --env: a
// `workspace create` given extraEnv and a `pane split` whose help does not
// list --env, or fails, run without --env, and the one warning line has
// today's text with the subcommand's words, in its reason and its advice.
func TestHerdr_EnvCheck_WithoutEnvFlagWarnsOncePerSubcommand(t *testing.T) {
	calls := map[string]struct {
		run  func(h Herdr) error
		argv []string
	}{
		"workspace create": {
			run: func(h Herdr) error {
				_, err := h.WorkspaceCreate(context.Background(), "/tmp/cwd", "org-x", PrimaryCLIEntry("claude"))
				return err
			},
			argv: workspaceCreateArgv,
		},
		"pane split": {
			run: func(h Herdr) error {
				_, err := h.PaneSplit(context.Background(), "w1:p1", HerdrSplitRight, "/tmp/cwd", PrimaryCLIEntry("claude"))
				return err
			},
			argv: argvWith(paneSplitArgv, []string{"--no-focus"}),
		},
	}
	for sub, c := range calls {
		for _, tt := range []struct {
			name       string
			help       string
			helpErr    error
			wantReason string
		}{
			{name: "help lists no --env", help: herdrTabCreateHelpNoEnv, wantReason: "herdr " + sub + " has no --env flag"},
			{name: "help fails", help: "unknown option: --help", helpErr: errors.New("herdr: exit status 2:\nunknown option: --help"),
				wantReason: "could not check whether herdr " + sub + " has --env ('herdr " + sub + " --help': herdr: exit status 2: unknown option: --help)"},
		} {
			t.Run(sub+"/"+tt.name, func(t *testing.T) {
				f := &subcmdFakeRunner{
					help:    map[string]string{sub: tt.help},
					helpErr: map[string]error{sub: tt.helpErr},
					replies: subcmdReplies,
				}
				var warn bytes.Buffer
				h := NewHerdr(f, SeatPaneEnv()...)
				h.Warn = &warn

				for range 2 {
					if err := c.run(h); err != nil {
						t.Fatalf("%s: %v", sub, err)
					}
				}
				if !reflect.DeepEqual(f.calls, [][]string{c.argv, c.argv}) {
					t.Fatalf("herdr calls = %q; want %q twice, with no --env", f.calls, c.argv)
				}
				if !reflect.DeepEqual(f.helpCalls, map[string]int{sub: 1}) {
					t.Fatalf("helps = %v; want `%s --help` once, the result cached", f.helpCalls, sub)
				}
				want := "warning: " + tt.wantReason + ", so ralph creates herdr panes without " +
					"--env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off --env RALPH_PRIMARY_CLI=claude: " +
					agmsgSelfNameClause + "; " + wipCheckpointClause + "; " + primaryCLIClause +
					". Use a herdr whose 'herdr " + sub + " --help' lists --env (0.7.5 has it)\n"
				if warn.String() != want {
					t.Fatalf("warning =\n%q\nwant exactly one line\n%q", warn.String(), want)
				}
			})
		}
	}
}

// TestHerdr_EnvCheck_EachSubcommandChecksItsOwnHelp pins FR-B8 of spec
// 2026-11-06-org-herdr-autostart-panes: one NewHerdr value checks `tab
// create`, `workspace create` and `pane split` each with its own help, once,
// so a herdr with --env on some of them gets --env on those calls only and
// one warning for each of the others.
func TestHerdr_EnvCheck_EachSubcommandChecksItsOwnHelp(t *testing.T) {
	for name, tt := range map[string]struct {
		help        map[string]string
		wantEnv     map[string]bool // whether that subcommand's calls carry --env
		wantWarning []string        // the subcommands warned about, in call order
	}{
		"pane split without --env": {
			help:        map[string]string{"tab create": herdrTabCreateHelp, "workspace create": herdrWorkspaceCreateHelp, "pane split": herdrTabCreateHelpNoEnv},
			wantEnv:     map[string]bool{"tab create": true, "workspace create": true},
			wantWarning: []string{"pane split"},
		},
		"only pane split with --env": {
			help:        map[string]string{"tab create": herdrTabCreateHelpNoEnv, "workspace create": herdrTabCreateHelpNoEnv, "pane split": herdrPaneSplitHelp},
			wantEnv:     map[string]bool{"pane split": true},
			wantWarning: []string{"tab create", "workspace create"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := &subcmdFakeRunner{help: tt.help, replies: subcmdReplies}
			var warn bytes.Buffer
			h := NewHerdr(f, SeatPaneEnv()...)
			h.Warn = &warn

			for range 2 {
				if _, err := h.TabCreate(context.Background(), "w3", "/tmp/cwd", "worker-1", PrimaryCLIEntry("claude")); err != nil {
					t.Fatalf("TabCreate: %v", err)
				}
				if _, err := h.WorkspaceCreate(context.Background(), "/tmp/cwd", "org-x", PrimaryCLIEntry("claude")); err != nil {
					t.Fatalf("WorkspaceCreate: %v", err)
				}
				if _, err := h.PaneSplit(context.Background(), "w1:p1", HerdrSplitRight, "/tmp/cwd", PrimaryCLIEntry("claude")); err != nil {
					t.Fatalf("PaneSplit: %v", err)
				}
			}
			if want := map[string]int{"tab create": 1, "workspace create": 1, "pane split": 1}; !reflect.DeepEqual(f.helpCalls, want) {
				t.Fatalf("helps = %v; want each subcommand's help once", f.helpCalls)
			}
			for _, argv := range f.calls {
				sub := argv[1] + " " + argv[2]
				if got := slices.Contains(argv, "--env"); got != tt.wantEnv[sub] {
					t.Fatalf("%s argv %q has --env = %v; want %v", sub, argv, got, tt.wantEnv[sub])
				}
			}
			lines := strings.Split(strings.TrimSuffix(warn.String(), "\n"), "\n")
			if len(lines) != len(tt.wantWarning) {
				t.Fatalf("warning lines = %q; want one for each of %q", lines, tt.wantWarning)
			}
			for i, sub := range tt.wantWarning {
				if !strings.HasPrefix(lines[i], "warning: herdr "+sub+" has no --env flag, ") ||
					!strings.HasSuffix(lines[i], "'herdr "+sub+" --help' lists --env (0.7.5 has it)") {
					t.Fatalf("warning line %d = %q; want the line for %s", i+1, lines[i], sub)
				}
			}
		})
	}
}

// TestHerdr_EnvCheck_LiteralChecksEveryCallOfEachSubcommand: a Herdr built
// as a literal with PaneEnv has no shared checks, so `workspace create`
// (given extraEnv) and `pane split` run their help on every call.
func TestHerdr_EnvCheck_LiteralChecksEveryCallOfEachSubcommand(t *testing.T) {
	f := &subcmdFakeRunner{help: map[string]string{"workspace create": herdrWorkspaceCreateHelp, "pane split": herdrPaneSplitHelp}, replies: subcmdReplies}
	h := Herdr{R: f, PaneEnv: []string{"X=1"}}
	for range 2 {
		if _, err := h.WorkspaceCreate(context.Background(), "/tmp/cwd", "org-x", "Y=2"); err != nil {
			t.Fatalf("WorkspaceCreate: %v", err)
		}
		if _, err := h.PaneSplit(context.Background(), "w1:p1", HerdrSplitDown, "/tmp/cwd"); err != nil {
			t.Fatalf("PaneSplit: %v", err)
		}
	}
	if want := map[string]int{"workspace create": 2, "pane split": 2}; !reflect.DeepEqual(f.helpCalls, want) {
		t.Fatalf("helps = %v; want %v", f.helpCalls, want)
	}
	ws := argvWith(workspaceCreateArgv, []string{"--env", "X=1", "--env", "Y=2"})
	split := []string{"herdr", "pane", "split", "w1:p1", "--direction", "down", "--ratio", "0.5", "--cwd", "/tmp/cwd", "--env", "X=1", "--no-focus"}
	if want := [][]string{ws, split, ws, split}; !reflect.DeepEqual(f.calls, want) {
		t.Fatalf("herdr calls = %q; want %q", f.calls, want)
	}
}

// TestHerdr_PaneSplit_RejectsOtherDirections: a direction other than right
// or down is an error before any herdr call, the --env check's help
// included.
func TestHerdr_PaneSplit_RejectsOtherDirections(t *testing.T) {
	for _, direction := range []string{"left", "up", "", "Right", "right ", "down,right"} {
		t.Run(fmt.Sprintf("%q", direction), func(t *testing.T) {
			f := &subcmdFakeRunner{help: map[string]string{"pane split": herdrPaneSplitHelp}, replies: subcmdReplies}
			got, err := NewHerdr(f, SeatPaneEnv()...).PaneSplit(context.Background(), "w1:p1", direction, "/tmp/cwd", PrimaryCLIEntry("claude"))
			if err == nil || !strings.Contains(err.Error(), fmt.Sprintf("herdr pane split: direction %q", direction)) || got != (HerdrPaneRef{}) {
				t.Fatalf("PaneSplit = %+v, %v; want no ids and an error naming the direction", got, err)
			}
			if len(f.calls) != 0 || len(f.helpCalls) != 0 {
				t.Fatalf("herdr calls %q, helps %v; want none", f.calls, f.helpCalls)
			}
		})
	}
}

// TestHerdr_PaneSplit_UnreadableReplyIsAnError pins PaneSplit's error
// contract: a reply it cannot read (not JSON, malformed, missing the pane
// or either id) is an error, never a bare id, and a failed call keeps
// herdr's not-found code readable by IsNotFound (the pane to split was
// closed) and the Runner error readable by errors.Is.
func TestHerdr_PaneSplit_UnreadableReplyIsAnError(t *testing.T) {
	notFound := `{"error":{"code":"pane_not_found","message":"pane w1:p1 not found"},"id":"cli:pane:split"}`
	timeoutErr := fmt.Errorf("herdr: timed out: %w", context.DeadlineExceeded)
	for _, tt := range []struct {
		name         string
		out          string
		runErr       error
		wantNotFound bool
		wantIs       error
	}{
		{name: "bare id with exit 0", out: "w1:p2"},
		{name: "empty stdout with exit 0", out: ""},
		{name: "malformed envelope", out: `{"id":"cli:pane:split","result":{"pane":{"pane_id":"w1:p2"`},
		{name: "no result.pane", out: `{"id":"cli:pane:split","result":{"type":"pane_info"}}`},
		{name: "no pane_id", out: `{"id":"cli:pane:split","result":{"pane":{"tab_id":"w1:t1","workspace_id":"w1"},"type":"pane_info"}}`},
		{name: "no tab_id", out: `{"id":"cli:pane:split","result":{"pane":{"pane_id":"w1:p2","workspace_id":"w1"},"type":"pane_info"}}`},
		{name: "not-found envelope with exit 1", out: notFound, runErr: errExit1, wantNotFound: true, wantIs: errExit1},
		{name: "not-found envelope with exit 0", out: notFound, wantNotFound: true},
		{name: "not-found envelope folded into the stderr text", runErr: fmt.Errorf("herdr: exit status 1: %s", notFound), wantNotFound: true},
		{name: "runner timeout", runErr: timeoutErr, wantIs: context.DeadlineExceeded},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeRunner{outputs: []string{tt.out}, errs: []error{tt.runErr}}
			got, err := (Herdr{R: f}).PaneSplit(context.Background(), "w1:p1", HerdrSplitRight, "/tmp/cwd")
			if err == nil || got != (HerdrPaneRef{}) {
				t.Fatalf("PaneSplit = %+v, %v; want no ids and an error", got, err)
			}
			if !strings.Contains(err.Error(), "herdr pane split") {
				t.Fatalf("error %v does not name herdr pane split", err)
			}
			if IsNotFound(err) != tt.wantNotFound {
				t.Fatalf("IsNotFound(%v) = %v; want %v", err, IsNotFound(err), tt.wantNotFound)
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Fatalf("errors.Is(%v, %v) = false", err, tt.wantIs)
			}
			if len(f.calls) != 1 {
				t.Fatalf("herdr calls = %v; want the split alone", f.calls)
			}
		})
	}
}

// TestHerdr_PaneRename_TabRename pins the argv of both rename subcommands,
// the label as one argv word, and their error contract, the close calls'
// (checkHerdrCloseResult): an id herdr does not know is an error IsNotFound
// recognises, and every other failure is an error it rejects. The success
// replies below are made up; neither call reads them.
func TestHerdr_PaneRename_TabRename(t *testing.T) {
	renames := []struct {
		name     string
		call     func(Herdr) error
		wantArgs []string
		ok       string
		notFound string
	}{
		{
			name:     "pane rename",
			call:     func(h Herdr) error { return h.PaneRename(context.Background(), "w1:p2", "seat a") },
			wantArgs: []string{"pane", "rename", "w1:p2", "seat a"},
			ok:       `{"id":"cli:pane:rename","result":{"pane":{"label":"seat a","pane_id":"w1:p2","tab_id":"w1:t1","workspace_id":"w1"},"type":"pane_info"}}`,
			notFound: `{"error":{"code":"pane_not_found","message":"pane w1:p2 not found"},"id":"cli:pane:rename"}`,
		},
		{
			name:     "tab rename",
			call:     func(h Herdr) error { return h.TabRename(context.Background(), "w1:t1", "org x") },
			wantArgs: []string{"tab", "rename", "w1:t1", "org x"},
			ok:       `{"id":"cli:tab:rename","result":{"tab":{"label":"org x","tab_id":"w1:t1","workspace_id":"w1"},"type":"tab_info"}}`,
			notFound: `{"error":{"code":"tab_not_found","message":"tab w1:t1 not found"},"id":"cli:tab:rename"}`,
		},
	}
	timeoutErr := fmt.Errorf("herdr: timed out: %w", context.DeadlineExceeded)
	for _, r := range renames {
		for _, tt := range []struct {
			name         string
			out          string
			runErr       error
			wantErr      bool
			wantNotFound bool
			wantIs       error
		}{
			{name: "ok envelope", out: r.ok},
			{name: "empty stdout from a fake runner", out: ""},
			{name: "not-found envelope on stdout with exit 1", out: r.notFound, runErr: errExit1, wantErr: true, wantNotFound: true, wantIs: errExit1},
			{name: "not-found envelope folded into the stderr text", runErr: fmt.Errorf("herdr: exit status 1: %s", r.notFound), wantErr: true, wantNotFound: true},
			{name: "not-found envelope with exit 0", out: r.notFound, wantErr: true, wantNotFound: true},
			{name: "other error code", out: `{"error":{"code":"internal","message":"boom"},"id":"cli:x"}`, runErr: errExit1, wantErr: true, wantIs: errExit1},
			{name: "runner timeout", runErr: timeoutErr, wantErr: true, wantIs: context.DeadlineExceeded},
		} {
			t.Run(r.name+"/"+tt.name, func(t *testing.T) {
				f := &fakeRunner{outputs: []string{tt.out}, errs: []error{tt.runErr}}
				err := r.call(Herdr{R: f})
				if c := f.lastCall(); c.name != "herdr" || !reflect.DeepEqual(c.args, r.wantArgs) || len(f.calls) != 1 {
					t.Fatalf("herdr calls = %v; want one, herdr %q", f.calls, r.wantArgs)
				}
				if !tt.wantErr {
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					return
				}
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if IsNotFound(err) != tt.wantNotFound || notFoundViaInterface(err) != tt.wantNotFound {
					t.Fatalf("IsNotFound(%v) = %v; want %v", err, IsNotFound(err), tt.wantNotFound)
				}
				if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
					t.Fatalf("errors.Is(%v, %v) = false", err, tt.wantIs)
				}
				if !strings.Contains(err.Error(), "herdr "+r.name) {
					t.Fatalf("error %v does not name herdr %s", err, r.name)
				}
			})
		}
	}
}

// TestHerdr_PaneList pins `pane list --workspace`: its argv, the panes it
// reads in herdr's order with their tab, workspace and label (a null or
// missing label is ""), an empty list, and the replies it must refuse, so a
// pane it cannot place never passes as one.
func TestHerdr_PaneList(t *testing.T) {
	wantArgs := []string{"pane", "list", "--workspace", "w1"}
	t.Run("panes", func(t *testing.T) {
		f := &fakeRunner{outputs: []string{paneListEnvelope}}
		got, err := (Herdr{R: f}).PaneList(context.Background(), "w1")
		want := []HerdrPane{
			{PaneID: "w1:p1", TabID: "w1:t1", WorkspaceID: "w1", Label: "leader"},
			{PaneID: "w1:p2", TabID: "w1:t1", WorkspaceID: "w1"},
			{PaneID: "w1:p3", TabID: "w1:t2", WorkspaceID: "w1"},
		}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("PaneList = %+v, %v; want %+v", got, err, want)
		}
		if c := f.lastCall(); c.name != "herdr" || !reflect.DeepEqual(c.args, wantArgs) || len(f.calls) != 1 {
			t.Fatalf("herdr calls = %v; want one, herdr %q", f.calls, wantArgs)
		}
	})
	t.Run("an empty list", func(t *testing.T) {
		got, err := (Herdr{R: &fakeRunner{outputs: []string{`{"id":"cli:pane:list","result":{"panes":[],"type":"pane_list"}}`}}}).PaneList(context.Background(), "w1")
		if err != nil || got == nil || len(got) != 0 {
			t.Fatalf("PaneList = %#v, %v; want an empty list and no error", got, err)
		}
	})
	notFound := `{"error":{"code":"workspace_not_found","message":"workspace w1 not found"},"id":"cli:pane:list"}`
	for _, tt := range []struct {
		name         string
		out          string
		runErr       error
		wantNotFound bool
	}{
		{name: "bare text with exit 0", out: "w1:p1"},
		{name: "empty stdout with exit 0", out: ""},
		{name: "malformed envelope", out: `{"id":"cli:pane:list","result":{"panes":[`},
		{name: "no result.panes", out: `{"id":"cli:pane:list","result":{"type":"pane_list"}}`},
		{name: "null result.panes", out: `{"id":"cli:pane:list","result":{"panes":null,"type":"pane_list"}}`},
		{name: "a pane with no pane_id", out: `{"id":"cli:pane:list","result":{"panes":[{"tab_id":"w1:t1","workspace_id":"w1"}],"type":"pane_list"}}`},
		{name: "a pane with no tab_id", out: `{"id":"cli:pane:list","result":{"panes":[{"pane_id":"w1:p1","workspace_id":"w1"}],"type":"pane_list"}}`},
		{name: "a pane with no workspace_id", out: `{"id":"cli:pane:list","result":{"panes":[{"pane_id":"w1:p1","tab_id":"w1:t1"}],"type":"pane_list"}}`},
		{name: "not-found envelope with exit 1", out: notFound, runErr: errExit1, wantNotFound: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (Herdr{R: &fakeRunner{outputs: []string{tt.out}, errs: []error{tt.runErr}}}).PaneList(context.Background(), "w1")
			if err == nil || got != nil {
				t.Fatalf("PaneList = %+v, %v; want no panes and an error", got, err)
			}
			if !strings.Contains(err.Error(), "herdr pane list") || IsNotFound(err) != tt.wantNotFound {
				t.Fatalf("error %v: want it to name herdr pane list, IsNotFound %v", err, tt.wantNotFound)
			}
		})
	}
}

// TestHerdr_PaneGet_ReadsTheLabel: PaneGet returns the pane's label from
// result.pane.label, "" for a null or missing label (no error), and the
// pane id herdr reports, or the one asked about when the reply has none.
func TestHerdr_PaneGet_ReadsTheLabel(t *testing.T) {
	for name, tt := range map[string]struct {
		out  string
		want HerdrPane
	}{
		"a label": {
			out:  `{"id":"cli:pane:get","result":{"pane":{"label":"leader","pane_id":"w1:p2","tab_id":"w1:t1","workspace_id":"w1"},"type":"pane_info"}}`,
			want: HerdrPane{PaneID: "w1:p2", TabID: "w1:t1", WorkspaceID: "w1", Label: "leader"},
		},
		"a null label": {
			out:  `{"id":"cli:pane:get","result":{"pane":{"label":null,"pane_id":"w1:p2","tab_id":"w1:t1","workspace_id":"w1"},"type":"pane_info"}}`,
			want: HerdrPane{PaneID: "w1:p2", TabID: "w1:t1", WorkspaceID: "w1"},
		},
		"no label (the 2026-10-07 reply)": {
			out:  realPaneGetEnvelope,
			want: HerdrPane{PaneID: "w1:p2", TabID: "w1:t2", WorkspaceID: "w1"},
		},
		"no pane_id": {
			out:  `{"id":"cli:pane:get","result":{"pane":{"label":"leader","tab_id":"w1:t1","workspace_id":"w1"},"type":"pane_info"}}`,
			want: HerdrPane{PaneID: "w1:p2", TabID: "w1:t1", WorkspaceID: "w1", Label: "leader"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := (Herdr{R: &fakeRunner{outputs: []string{tt.out}}}).PaneGet(context.Background(), "w1:p2")
			if err != nil || got != tt.want {
				t.Fatalf("PaneGet = %+v, %v; want %+v", got, err, tt.want)
			}
		})
	}
}
