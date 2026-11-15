package toolchain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Running the probe script in a process group of its own within one budget
// (plan 2026-11-12-gate-tool-versions, D3), the way the gate runs its
// scripts (runGateScript in internal/org). Release builds target darwin and
// linux only, where Setpgid and kill(-pgid) exist.

// KillGrace is how long Run waits for the probe's process group to exit after
// SIGTERM before it sends SIGKILL to the group, and how long it waits for
// the output to close after the shell exits (exec.Cmd.WaitDelay), so a
// process that holds the output cannot keep Run waiting. Tests shorten it.
var KillGrace = 2 * time.Second

// MaxOutputBytes caps the stdout Run keeps; what comes after is read and
// dropped.
var MaxOutputBytes = 1 << 20

// maxStderrBytes caps the stderr Run keeps.
const maxStderrBytes = 64 << 10

// groupPollInterval is how often Run checks whether a process group it
// signalled is gone.
const groupPollInterval = 20 * time.Millisecond

// RunResult is what Run saw of the script.
type RunResult struct {
	// Stdout is what the script printed, up to MaxOutputBytes; after a
	// timeout, what it printed until then.
	Stdout []byte
	// Stderr is the first 64 KiB of what the script wrote to stderr.
	Stderr []byte
	// TimedOut is whether the budget ran out while the shell still ran.
	TimedOut bool
	// ExitCode is the shell's exit status; -1 when a signal ended it or it
	// did not start.
	ExitCode int
}

// Run runs script with argv before it (argv[0] argv[1:]... script, such as
// /bin/sh -c script or $SHELL -lc script) in dir with env, its stdin empty,
// in a process group of its own, for at most timeout. When the timeout
// passes or ctx is done, the group gets SIGTERM, and whatever of it is
// still there KillGrace later gets SIGKILL; the output printed until then
// is kept. When the shell exits on its own, what it left in its group (a
// background process holding the output, say) is stopped the same way, and
// Run waits at most KillGrace for the output to close. The error is non-nil
// when the shell could not start or ctx is done (context.Cause(ctx)); a
// timeout sets TimedOut, with no error.
func Run(ctx context.Context, argv []string, script, dir string, env []string, timeout time.Duration) (RunResult, error) {
	res := RunResult{ExitCode: -1}
	if len(argv) == 0 || argv[0] == "" {
		return res, errors.New("no shell to run the probe with")
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, argv[0], append(slices.Clone(argv[1:]), script)...)
	cmd.Dir, cmd.Env = dir, env
	stdout := &cappedBuffer{limit: MaxOutputBytes}
	stderr := &cappedBuffer{limit: maxStderrBytes}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// cancelledAt is when Cancel sent SIGTERM (UnixNano), set from the
	// goroutine os/exec watches runCtx in; os/exec calls Cancel only while
	// the shell still runs.
	var cancelledAt atomic.Int64
	cmd.Cancel = func() error {
		cancelledAt.Store(time.Now().UnixNano())
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		return nil
	}
	cmd.WaitDelay = KillGrace
	runErr := cmd.Run()
	if cmd.Process == nil {
		return res, fmt.Errorf("could not start %s: %w", argv[0], runErr)
	}
	if at := cancelledAt.Load(); at != 0 {
		stopGroup(cmd.Process.Pid, time.Unix(0, at).Add(KillGrace))
	} else {
		stopStrays(cmd.Process.Pid)
	}
	res.Stdout, res.Stderr = stdout.bytes(), stderr.bytes()
	if cmd.ProcessState != nil {
		res.ExitCode = cmd.ProcessState.ExitCode()
	}
	if ctx.Err() != nil {
		return res, context.Cause(ctx)
	}
	res.TimedOut = cancelledAt.Load() != 0
	return res, nil
}

// stopStrays stops what is left of process group pgid after its leader
// exited: when the group still has a process (kill(-pgid, 0) succeeds), it
// gets SIGTERM, and whatever is still there KillGrace later gets SIGKILL.
func stopStrays(pgid int) {
	if syscall.Kill(-pgid, 0) != nil {
		return
	}
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	stopGroup(pgid, time.Now().Add(KillGrace))
}

// stopGroup waits until deadline for process group pgid to have no process
// left (kill(-pgid, 0) fails), then sends SIGKILL to the group. A group
// whose processes are gone answers ESRCH and is left alone.
func stopGroup(pgid int, deadline time.Time) {
	for time.Now().Before(deadline) {
		if syscall.Kill(-pgid, 0) != nil {
			return
		}
		time.Sleep(groupPollInterval)
	}
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
}

// cappedBuffer keeps the first limit bytes written to it and drops the
// rest, reporting every write as whole so the writer is never stopped.
// os/exec writes to it from its own goroutines.
type cappedBuffer struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	limit int
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if room := c.limit - c.buf.Len(); room > 0 {
		c.buf.Write(p[:min(room, len(p))])
	}
	return len(p), nil
}

func (c *cappedBuffer) bytes() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.buf.Bytes())
}
