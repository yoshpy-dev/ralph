// Package driver implements exec.Command-based adapters for the external
// herdr and agmsg CLIs behind an injectable Runner interface, so callers
// (the `ralph org` verbs in internal/org, spawn.go and verbs.go) and tests
// never need the real binaries on PATH. CI has neither herdr nor agmsg
// installed; every test in this package uses a fake Runner that records
// argv and returns canned output -- no real herdr/agmsg process is ever
// spawned by `go test`.
package driver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner executes name with args, returning combined stdout (trimmed).
// Implementations must honor ctx cancellation/timeout. Production code uses
// ExecRunner; tests inject fakes that record argv without touching the
// filesystem or spawning processes.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

// ExecRunner is the real Runner, backed by exec.CommandContext.
type ExecRunner struct{}

// Run executes name with args via exec.CommandContext, honoring ctx
// cancellation/timeout. On failure, captured stderr (if any) is folded into
// the returned error so callers get CLI diagnostics without re-running the
// command.
func (ExecRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return out, fmt.Errorf("%s: timed out: %w", name, ctx.Err())
		}
		if stderrText := strings.TrimSpace(stderr.String()); stderrText != "" {
			return out, fmt.Errorf("%s: %w: %s", name, err, stderrText)
		}
		return out, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

// ErrNotInstalled is wrapped with the binary name by the availability
// checks below (HerdrAvailable, AgmsgAvailable).
var ErrNotInstalled = errors.New("binary not installed")

// lookPath is exec.LookPath by default; tests reassign it to simulate a
// missing/present binary without needing a real fake executable on PATH.
var lookPath = exec.LookPath

// HerdrAvailable reports whether the herdr binary is discoverable on PATH.
// A non-nil error wraps ErrNotInstalled (errors.Is-compatible).
func HerdrAvailable() error {
	return checkAvailable("herdr")
}

// agmsgSendScript is the file AgmsgAvailable stats to decide whether home
// looks like a usable agmsg installation. send.sh is the most fundamental
// script in the collection (every seat sends at least one HELLO), so its
// presence is a reasonable proxy for "agmsg is installed here".
const agmsgSendScript = "send.sh"

// AgmsgAvailable reports whether home is a usable agmsg installation by
// checking for <home>/scripts/send.sh. Unlike HerdrAvailable, this does NOT
// consult PATH: an `agmsg` executable on PATH (e.g. the npm bootstrapper
// that installs/updates the real script collection) must not be mistaken
// for the real interface -- only a populated script home counts. A non-nil
// error wraps ErrNotInstalled (errors.Is-compatible).
func AgmsgAvailable(home string) error {
	scriptPath := filepath.Join(home, "scripts", agmsgSendScript)
	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("agmsg: %w (expected %s)", ErrNotInstalled, scriptPath)
	}
	return nil
}

func checkAvailable(bin string) error {
	if _, err := lookPath(bin); err != nil {
		return fmt.Errorf("%s: %w", bin, ErrNotInstalled)
	}
	return nil
}
