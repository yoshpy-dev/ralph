package driver

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestProbeModel_ClaudeArgv(t *testing.T) {
	f := &fakeRunner{outputs: []string{"pong"}}

	if err := ProbeModel(context.Background(), f, "claude", "sonnet"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"--model", "sonnet", "-p", "ping", "--output-format", "text"}
	if c := f.lastCall(); c.name != "claude" || !reflect.DeepEqual(c.args, want) {
		t.Fatalf("argv mismatch: got name=%q args=%v, want name=claude args=%v", c.name, c.args, want)
	}
}

func TestProbeModel_CodexArgv(t *testing.T) {
	f := &fakeRunner{outputs: []string{"pong"}}

	if err := ProbeModel(context.Background(), f, "codex", "gpt-5-codex"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"exec", "--model", "gpt-5-codex", "--skip-git-repo-check", "ping"}
	if c := f.lastCall(); c.name != "codex" || !reflect.DeepEqual(c.args, want) {
		t.Fatalf("argv mismatch: got name=%q args=%v, want name=codex args=%v", c.name, c.args, want)
	}
}

func TestProbeModel_RunnerErrorSurfacesDetail(t *testing.T) {
	f := &fakeRunner{errs: []error{errTest}}

	err := ProbeModel(context.Background(), f, "claude", "not-a-model")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not-a-model") {
		t.Fatalf("expected error to mention the model id, got: %v", err)
	}
	if !strings.Contains(err.Error(), errTest.Error()) {
		t.Fatalf("expected error to wrap runner error detail, got: %v", err)
	}
}

func TestProbeModel_UnknownDriverRejected(t *testing.T) {
	f := &fakeRunner{}

	err := ProbeModel(context.Background(), f, "gemini", "some-model")
	if err == nil {
		t.Fatal("expected unknown driver to be rejected, got nil")
	}
	if len(f.calls) != 0 {
		t.Fatalf("expected no Run call for unknown driver, got %d calls: %v", len(f.calls), f.calls)
	}
}
