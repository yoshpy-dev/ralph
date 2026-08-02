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

	got, err := h.WorkspaceCreate(context.Background(), "/tmp/cwd", "lead")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "ws-123" {
		t.Fatalf("want ws-123, got %q", got)
	}
	want := []string{"workspace", "create", "--cwd", "/tmp/cwd", "--label", "lead"}
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

	got, err := h.WorkspaceCreate(context.Background(), "/tmp/cwd", "lead")
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

	_, err := h.WorkspaceCreate(context.Background(), "/tmp/cwd", "lead")
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
