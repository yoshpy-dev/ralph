package driver

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
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
	if got != "ws-123" {
		t.Fatalf("want ws-123, got %q", got)
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
// producing workspace_not_found).
func TestHerdr_WorkspaceCreate_RealEnvelope(t *testing.T) {
	f := &fakeRunner{outputs: []string{realWorkspaceCreateEnvelope}}
	h := Herdr{R: f}

	got, err := h.WorkspaceCreate(context.Background(), "/tmp/cwd", "leader")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "w3" {
		t.Fatalf("want w3 (result.workspace.workspace_id), got %q", got)
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
	if got != "tab-9" {
		t.Fatalf("want tab-9, got %q", got)
	}
	want := []string{"tab", "create", "--workspace", "ws-123", "--cwd", "/tmp/cwd", "--label", "worker-1"}
	if c := f.lastCall(); !reflect.DeepEqual(c.args, want) {
		t.Fatalf("argv mismatch: got %v, want %v", c.args, want)
	}
}

// TestHerdr_TabCreate_RealEnvelope pins the fix: TabCreate must extract
// result.root_pane.pane_id from the real JSON envelope.
func TestHerdr_TabCreate_RealEnvelope(t *testing.T) {
	f := &fakeRunner{outputs: []string{realTabCreateEnvelope}}
	h := Herdr{R: f}

	got, err := h.TabCreate(context.Background(), "w3", "/tmp/cwd", "worker-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "w3:p2" {
		t.Fatalf("want w3:p2 (result.root_pane.pane_id), got %q", got)
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
				tabID, workspaceID, err := h.PaneGet(context.Background(), "w1:p2")
				return tabID + " " + workspaceID, err
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
