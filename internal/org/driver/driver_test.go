package driver

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// errTest is a shared sentinel used by tests that only need to assert a
// runner-supplied error propagates unchanged (no real process, no argv
// assertion needed).
var errTest = errors.New("fake runner error")

// call records one Run() invocation's argv, without executing anything.
type call struct {
	name string
	args []string
}

// fakeRunner is the Runner used by every adapter test in this package --
// it never spawns a process, so herdr/agmsg do not need to be installed.
type fakeRunner struct {
	calls   []call
	outputs []string
	errs    []error
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	i := len(f.calls)
	f.calls = append(f.calls, call{name: name, args: append([]string{}, args...)})
	var out string
	var err error
	if i < len(f.outputs) {
		out = f.outputs[i]
	}
	if i < len(f.errs) {
		err = f.errs[i]
	}
	return out, err
}

func (f *fakeRunner) lastCall() call {
	if len(f.calls) == 0 {
		return call{}
	}
	return f.calls[len(f.calls)-1]
}

func TestExecRunner_Run_Success(t *testing.T) {
	r := ExecRunner{}
	out, err := r.Run(context.Background(), "echo", "hello", "world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "hello world" {
		t.Fatalf("want %q, got %q", "hello world", out)
	}
}

func TestExecRunner_Run_TimeoutHonored(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	r := ExecRunner{}
	start := time.Now()
	_, err := r.Run(ctx, "sleep", "5")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("Run did not honor ctx timeout: took %v", elapsed)
	}
}

func TestExecRunner_Run_StderrCaptured(t *testing.T) {
	r := ExecRunner{}
	_, err := r.Run(context.Background(), "sh", "-c", "echo err >&2; exit 3")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "err") {
		t.Fatalf("expected stderr detail in error, got: %v", err)
	}
}

func TestHerdrAvailable_NotInstalled(t *testing.T) {
	orig := lookPath
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	defer func() { lookPath = orig }()

	err := HerdrAvailable()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("expected errors.Is(err, ErrNotInstalled), got: %v", err)
	}
	if !strings.Contains(err.Error(), "herdr") {
		t.Fatalf("expected error to mention herdr, got: %v", err)
	}
}

func TestAgmsgAvailable_NotInstalled(t *testing.T) {
	orig := lookPath
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	defer func() { lookPath = orig }()

	err := AgmsgAvailable()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("expected errors.Is(err, ErrNotInstalled), got: %v", err)
	}
	if !strings.Contains(err.Error(), "agmsg") {
		t.Fatalf("expected error to mention agmsg, got: %v", err)
	}
}

func TestHerdrAvailable_Installed(t *testing.T) {
	orig := lookPath
	lookPath = func(string) (string, error) { return "/usr/bin/herdr", nil }
	defer func() { lookPath = orig }()

	if err := HerdrAvailable(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAgmsgAvailable_Installed(t *testing.T) {
	orig := lookPath
	lookPath = func(string) (string, error) { return "/usr/bin/agmsg", nil }
	defer func() { lookPath = orig }()

	if err := AgmsgAvailable(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
