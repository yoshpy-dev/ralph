package org

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// How Gate (gate.go) and Regate (regate.go) run the gate scripts: the log
// under <state-dir>/gates/ (createGateLog), the environment the scripts get
// (gateScriptEnv), each script in a process group of its own
// (runGateScript), and the tail of the failing script's output (gateOutput).
// The log's lines and the errors name the verb that runs them (gateWording).

const (
	// gateLogDir is the directory under the org state dir that holds the
	// gate logs, <org_id>-<UTC time>.log (createGateLog).
	gateLogDir = "gates"
	// gateLogTimeLayout is the UTC time in a gate log's name.
	gateLogTimeLayout = "20060102T150405Z"
	// gateLogMaxSuffix bounds the -2, -3, ... suffixes createGateLog tries
	// when a log of the same org and second exists.
	gateLogMaxSuffix = 100
	// gateTailLines is how many of the failing script's last output lines
	// GateResult.Tail keeps.
	gateTailLines = 20
	// gateTailMaxLineBytes caps one line of the tail; a line without a line
	// break grows no further.
	gateTailMaxLineBytes = 4096
	// gateScopeEnv is the one verification variable the scripts get (B9):
	// the full language scope, never narrowed by a diff.
	gateScopeEnv = "RALPH_VERIFY_SCOPE=full"
	// gateLocalConfigEnv keeps the scripts off the developer's personal
	// ralph.local.toml (config.LocalConfigEnv), for the reason the
	// RALPH_VERIFY_* variables are dropped (B9): what the gate's `go test`
	// checks, and any value the scripts read from ralph's config, must not
	// change with that file. Gate drops an inherited RALPH_LOCAL_CONFIG
	// first, so this is the one the scripts get whatever the caller has
	// (plan 2026-10-29-local-config-org, AC4).
	gateLocalConfigEnv = config.LocalConfigEnv + "=off"
)

// gateStrippedEnvPrefixes are the prefixes of the variable names Gate drops
// from the caller's environment before it runs the scripts (B9): a
// RALPH_VERIFY_BASE=HEAD or a RALPH_VERIFY_PROJECT_ROOTS could otherwise make
// the language verifiers skip the change.
var gateStrippedEnvPrefixes = []string{"RALPH_VERIFY_", "HARNESS_VERIFY_"}

// gateKillGrace is how long the gate waits for a script's process group to
// exit after SIGTERM before it sends SIGKILL to the group: on an interrupt,
// and for what a script left running when it exited. Tests shorten it.
var gateKillGrace = 5 * time.Second

// gateScriptEnv is the environment the gate scripts get (B9): env without
// every variable whose name starts with RALPH_VERIFY_ or HARNESS_VERIFY_
// and without RALPH_LOCAL_CONFIG, then RALPH_VERIFY_SCOPE=full and
// RALPH_LOCAL_CONFIG=off.
func gateScriptEnv(env []string) []string {
	out := make([]string, 0, len(env)+2)
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		stripped := name == config.LocalConfigEnv
		for _, prefix := range gateStrippedEnvPrefixes {
			stripped = stripped || strings.HasPrefix(name, prefix)
		}
		if !stripped {
			out = append(out, kv)
		}
	}
	return append(out, gateScopeEnv, gateLocalConfigEnv)
}

// gateEnvValue returns key's value in env, the last entry winning as it does
// for a child process, and "" when env has none.
func gateEnvValue(env []string, key string) string {
	var value string
	for _, kv := range env {
		if name, v, ok := strings.Cut(kv, "="); ok && name == key {
			value = v
		}
	}
	return value
}

// createGateLog creates the log of a gate of orgID, empty, under
// <state-dir>/gates/ (state-dir is the manifest's directory, made absolute):
// <org_id>-<UTC time, gateLogTimeLayout>.log, or the same name with -2, -3,
// ... before .log when one exists, created with O_EXCL so two gates never
// share a log. It returns the log's absolute path and the open file. An
// error starts with w.prefix.
func (o *Org) createGateLog(orgID string, w gateWording) (string, *os.File, error) {
	stateDir, err := absPath(filepath.Dir(o.Manifest.Path()))
	if err != nil {
		return "", nil, fmt.Errorf("%sresolve the state dir for the gate log: %w", w.prefix, err)
	}
	dir := filepath.Join(stateDir, gateLogDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", nil, fmt.Errorf("%screate the gate log dir %s: %w", w.prefix, dir, err)
	}
	name := orgID + "-" + o.nowTime().UTC().Format(gateLogTimeLayout)
	for n := 1; n <= gateLogMaxSuffix; n++ {
		path := filepath.Join(dir, name+".log")
		if n > 1 {
			path = filepath.Join(dir, name+"-"+strconv.Itoa(n)+".log")
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			return path, f, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", nil, fmt.Errorf("%screate the gate log %s: %w", w.prefix, path, err)
		}
	}
	return "", nil, fmt.Errorf("%screate the gate log: %d logs named %s already exist in %s", w.prefix, gateLogMaxSuffix, name, dir)
}

// gateRun is what runGateScripts found.
type gateRun struct {
	failed string   // GateReasonStaticFailed or GateReasonTestFailed; "" when both passed
	tail   []string // the failing script's last output lines
}

// runGateScripts runs the static script and, when it exits 0, the test
// script in t.root (runGateScript), with env, writing a header line before
// each and its exit status after it to log, and their output in between.
// When a script left processes running that runGateScript stopped, a line
// after its exit status says so; its exit status stands. The log's lines
// name w.verb ("ralph org gate").
// The error is ctx's when it is done, or a log write that failed, which
// starts with w.prefix.
func runGateScripts(ctx context.Context, t gateTarget, env []string, log io.Writer, w gateWording) (gateRun, error) {
	header := func(format string, args ...any) error {
		if _, err := fmt.Fprintf(log, format, args...); err != nil {
			return fmt.Errorf("%swrite the log: %w", w.prefix, err)
		}
		return nil
	}
	if err := header("# %s\n# worktree: %s\n# branch: %s\n# head: %s\n# base: %s %s\n",
		w.verb, t.root, t.branch, t.head, t.base.ref, t.base.sha); err != nil {
		return gateRun{}, err
	}
	for _, s := range []struct{ rel, reason string }{
		{GateStaticScript, GateReasonStaticFailed},
		{GateTestScript, GateReasonTestFailed},
	} {
		if err := header("\n==> %s: running ./%s\n", w.verb, s.rel); err != nil {
			return gateRun{}, err
		}
		out := &gateOutput{log: log}
		code, strays, err := runGateScript(ctx, t.root, s.rel, env, out, w.verb)
		if err != nil {
			_ = header("\n==> %s: ./%s interrupted\n", w.verb, s.rel)
			return gateRun{}, err
		}
		if out.err != nil {
			return gateRun{}, fmt.Errorf("%swrite the log: %w", w.prefix, out.err)
		}
		if err := header("\n==> %s: ./%s exited %d\n", w.verb, s.rel, code); err != nil {
			return gateRun{}, err
		}
		if strays {
			if err := header("==> %s: stopped the processes ./%s left running after it exited\n", w.verb, s.rel); err != nil {
				return gateRun{}, err
			}
		}
		if code != 0 {
			return gateRun{failed: s.reason, tail: out.tail()}, nil
		}
	}
	return gateRun{}, nil
}

// runGateScript runs ./rel in root with env and both output streams going
// to out, its stdin empty, in a process group of its own, and returns its
// exit code (-1 when a signal ended it or it could not start; the reason then
// goes to out). When ctx is done the group gets SIGTERM, and whatever of it
// is still there gateKillGrace later gets SIGKILL (stopGateGroup); the error
// is then ctx's. When the script exits on its own but leaves processes in its
// group (a background job it did not wait for), they are stopped the same way
// (stopGateStrays) and strays is set; the exit code stands. The lines it
// writes to out name verb ("ralph org gate"). Release builds target darwin
// and linux only, where Setpgid and kill(-pgid) exist.
func runGateScript(ctx context.Context, root, rel string, env []string, out io.Writer, verb string) (code int, strays bool, err error) {
	cmd := exec.CommandContext(ctx, filepath.Join(root, filepath.FromSlash(rel)))
	cmd.Args[0] = "./" + rel
	cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr = root, env, out, out
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// cancelledAt is when Cancel sent SIGTERM (UnixNano), set from the
	// goroutine os/exec watches ctx in.
	var cancelledAt atomic.Int64
	cmd.Cancel = func() error {
		cancelledAt.Store(time.Now().UnixNano())
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		return nil
	}
	cmd.WaitDelay = gateKillGrace
	runErr := cmd.Run()
	if ctx.Err() != nil {
		if cmd.Process != nil {
			since := time.Now()
			if at := cancelledAt.Load(); at != 0 {
				since = time.Unix(0, at)
			}
			stopGateGroup(cmd.Process.Pid, since.Add(gateKillGrace))
		}
		return 0, false, context.Cause(ctx)
	}
	if cmd.ProcessState == nil {
		_, _ = fmt.Fprintf(out, "%s: ./%s could not run: %v\n", verb, rel, runErr)
		return -1, false, nil
	}
	if errors.Is(runErr, exec.ErrWaitDelay) {
		_, _ = fmt.Fprintf(out, "%s: a process ./%s started still held its output %s after it exited\n", verb, rel, gateKillGrace)
	}
	return cmd.ProcessState.ExitCode(), stopGateStrays(cmd.Process.Pid), nil
}

// stopGateStrays stops what is left of process group pgid after its leader
// exited: when the group still has a process (kill(-pgid, 0) succeeds), the
// group gets SIGTERM, and whatever of it is still there gateKillGrace later
// gets SIGKILL (stopGateGroup). It reports whether any process was left.
func stopGateStrays(pgid int) bool {
	if syscall.Kill(-pgid, 0) != nil {
		return false
	}
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	stopGateGroup(pgid, time.Now().Add(gateKillGrace))
	return true
}

// stopGateGroup waits until deadline for process group pgid to have no
// process left (kill(-pgid, 0) fails), then sends SIGKILL to the group. A
// group whose processes are gone answers ESRCH and is left alone.
func stopGateGroup(pgid int, deadline time.Time) {
	for time.Now().Before(deadline) {
		if syscall.Kill(-pgid, 0) != nil {
			return
		}
		time.Sleep(manifestLockPollInterval)
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
}

// gateOutput is the writer a gate script's stdout and stderr both go to
// (os/exec calls Write from one goroutine at a time for the two when they
// are the same writer): every byte goes to log, and the last gateTailLines
// lines are kept for the tail. The first log write error is kept in err and
// later writes skip the log, so the script is not stopped by it.
type gateOutput struct {
	log     io.Writer
	err     error
	lines   []string
	partial []byte
}

func (w *gateOutput) Write(b []byte) (int, error) {
	if w.err == nil {
		if _, err := w.log.Write(b); err != nil {
			w.err = err
		}
	}
	rest := b
	for len(rest) > 0 {
		i := bytes.IndexByte(rest, '\n')
		if i < 0 {
			w.partial = append(w.partial, rest[:min(len(rest), gateTailMaxLineBytes-len(w.partial))]...)
			break
		}
		w.partial = append(w.partial, rest[:min(i, gateTailMaxLineBytes-len(w.partial))]...)
		w.push(string(w.partial))
		w.partial = w.partial[:0]
		rest = rest[i+1:]
	}
	return len(b), nil
}

// push keeps line as the latest complete line, a trailing carriage return
// dropped, and the oldest beyond gateTailLines dropped.
func (w *gateOutput) push(line string) {
	w.lines = append(w.lines, strings.TrimSuffix(line, "\r"))
	if len(w.lines) > gateTailLines {
		w.lines = w.lines[len(w.lines)-gateTailLines:]
	}
}

// tail returns the last gateTailLines lines written, a last line with no
// line break included.
func (w *gateOutput) tail() []string {
	lines := append([]string(nil), w.lines...)
	if len(w.partial) > 0 {
		lines = append(lines, strings.TrimSuffix(string(w.partial), "\r"))
	}
	if len(lines) > gateTailLines {
		lines = lines[len(lines)-gateTailLines:]
	}
	return lines
}
