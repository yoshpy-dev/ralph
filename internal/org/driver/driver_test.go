package driver

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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

func TestHerdrAvailable_Installed(t *testing.T) {
	orig := lookPath
	lookPath = func(string) (string, error) { return "/usr/bin/herdr", nil }
	defer func() { lookPath = orig }()

	if err := HerdrAvailable(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestAgmsgAvailable_HomeWithSendScript_NoError pins AC-1: a home directory
// with scripts/send.sh present is reported as available.
func TestAgmsgAvailable_HomeWithSendScript_NoError(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "scripts", "send.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := AgmsgAvailable(home); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestAgmsgAvailable_HomeWithoutSendScript_ErrNotInstalled pins AC-1: a home
// directory that does not contain scripts/send.sh is unavailable.
func TestAgmsgAvailable_HomeWithoutSendScript_ErrNotInstalled(t *testing.T) {
	home := t.TempDir() // empty -- no scripts/ subdir at all.

	err := AgmsgAvailable(home)
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

// TestAgmsgAvailable_PathBootstrapperIgnored pins AC-1's explicit
// requirement: an `agmsg` executable on PATH (e.g. the npm bootstrapper that
// installs/updates the real script collection) must NOT count as available
// when the home directory itself has no scripts/send.sh. AgmsgAvailable
// never consults PATH, so this is really just confirming that fact end to
// end via a PATH environment that *would* fool a LookPath-based check.
func TestAgmsgAvailable_PathBootstrapperIgnored(t *testing.T) {
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "agmsg"), []byte("#!/bin/sh\necho bootstrapper\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)

	home := t.TempDir() // no scripts/send.sh here.
	err := AgmsgAvailable(home)
	if err == nil {
		t.Fatal("expected error despite agmsg bootstrapper on PATH, got nil")
	}
	if !errors.Is(err, ErrNotInstalled) {
		t.Fatalf("expected errors.Is(err, ErrNotInstalled), got: %v", err)
	}
}
