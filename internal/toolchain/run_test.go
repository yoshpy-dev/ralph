package toolchain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// setKillGrace shortens KillGrace for one test.
func setKillGrace(t *testing.T, d time.Duration) {
	t.Helper()
	old := KillGrace
	KillGrace = d
	t.Cleanup(func() { KillGrace = old })
}

// readPID reads the process id a test script wrote to p, waiting briefly
// for the file to be written.
func readPID(t *testing.T, p string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		data, err := os.ReadFile(p)
		if err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && pid > 0 {
				return pid
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("no process id in %s: %v", p, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// waitProcessGone fails the test unless process pid is gone within a few
// seconds (an orphan is reaped by init once it dies).
func waitProcessGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for syscall.Kill(pid, 0) == nil {
		if time.Now().After(deadline) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
			t.Fatalf("process %d is still running after Run returned", pid)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestRun_TimeoutStopsTheWholeGroupAndKeepsTheOutput(t *testing.T) {
	setKillGrace(t, 300*time.Millisecond)
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	// The child ignores SIGTERM, so only the SIGKILL after the grace stops it.
	script := "printf 'before\\n'; trap '' TERM; /bin/sleep 30 & echo $! > '" + pidFile + "'; wait; printf 'after\\n'"
	timeout := 1 * time.Second
	start := time.Now()
	res, err := Run(context.Background(), []string{"/bin/sh", "-c"}, script, t.TempDir(), nil, timeout)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !res.TimedOut {
		t.Error("TimedOut = false, want true")
	}
	if got := string(res.Stdout); got != "before\n" {
		t.Errorf("Stdout = %q, want what came before the timeout", got)
	}
	if res.ExitCode != -1 {
		t.Errorf("ExitCode = %d, want -1 (killed)", res.ExitCode)
	}
	// The budget and two graces (WaitDelay, then the group's SIGKILL); the
	// slack is for a loaded machine, far below the child's 30 s sleep.
	if limit := timeout + 2*KillGrace + 5*time.Second; elapsed > limit {
		t.Errorf("Run took %s, over %s", elapsed, limit)
	}
	waitProcessGone(t, readPID(t, pidFile))
}

func TestRun_ChildHoldingStdoutAfterTheShellExitsDoesNotHang(t *testing.T) {
	setKillGrace(t, 300*time.Millisecond)
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	script := "/bin/sleep 30 & echo $! > '" + pidFile + "'; printf 'done\\n'; exit 0"
	start := time.Now()
	res, err := Run(context.Background(), []string{"/bin/sh", "-c"}, script, t.TempDir(), nil, 20*time.Second)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.TimedOut || res.ExitCode != 0 || string(res.Stdout) != "done\n" {
		t.Errorf("Run = %+v, want exit 0 with done and no timeout", res)
	}
	if elapsed > 10*time.Second {
		t.Errorf("Run took %s: it waited for the child holding stdout", elapsed)
	}
	waitProcessGone(t, readPID(t, pidFile))
}

func TestRun_ContextCancelStopsTheGroup(t *testing.T) {
	setKillGrace(t, 300*time.Millisecond)
	pidFile := filepath.Join(t.TempDir(), "child.pid")
	script := "/bin/sleep 30 & echo $! > '" + pidFile + "'; wait"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Cancel once the child has started (or after 5 s at most); the
	// goroutine does not call t, which only the test's goroutine may.
	go func() {
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
			if data, err := os.ReadFile(pidFile); err == nil && strings.TrimSpace(string(data)) != "" {
				break
			}
		}
		cancel()
	}()
	start := time.Now()
	res, err := Run(ctx, []string{"/bin/sh", "-c"}, script, t.TempDir(), nil, 20*time.Second)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Run error = %v, want context.Canceled", err)
	}
	if res.TimedOut {
		t.Error("TimedOut = true after a cancel, want false")
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("Run took %s after the cancel", elapsed)
	}
	waitProcessGone(t, readPID(t, pidFile))
}

func TestRun_StartFailureIsAnError(t *testing.T) {
	for _, argv := range [][]string{nil, {""}, {filepath.Join(t.TempDir(), "no-such-shell"), "-c"}} {
		res, err := Run(context.Background(), argv, "true", t.TempDir(), nil, 5*time.Second)
		if err == nil {
			t.Errorf("Run(%q): no error", argv)
		}
		if res.ExitCode != -1 || res.TimedOut {
			t.Errorf("Run(%q) = %+v, want exit code -1 and no timeout", argv, res)
		}
	}
}

func TestRun_KeepsStderrApartAndCapsStdout(t *testing.T) {
	old := MaxOutputBytes
	MaxOutputBytes = 10
	t.Cleanup(func() { MaxOutputBytes = old })
	res, err := Run(context.Background(), []string{"/bin/sh", "-c"}, "printf '0123456789abcdef'; printf 'oops\\n' >&2; exit 4", t.TempDir(), nil, 5*time.Second)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if string(res.Stdout) != "0123456789" || string(res.Stderr) != "oops\n" || res.ExitCode != 4 {
		t.Errorf("Run = stdout %q, stderr %q, exit %d; want the first 10 bytes, oops, 4", res.Stdout, res.Stderr, res.ExitCode)
	}
}

func TestRun_GivesTheScriptAfterArgvAndRunsInDirWithEnv(t *testing.T) {
	dir := t.TempDir()
	res, err := Run(context.Background(), []string{"/bin/sh", "-c"}, `printf '%s|%s|%s\n' "$PWD" "$PROBE_X" "$0"`, dir, []string{"PROBE_X=y"}, 5*time.Second)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	parts := strings.Split(strings.TrimSpace(string(res.Stdout)), "|")
	if len(parts) != 3 {
		t.Fatalf("Stdout = %q", res.Stdout)
	}
	wantDir, _ := filepath.EvalSymlinks(dir)
	gotDir, _ := filepath.EvalSymlinks(parts[0])
	if gotDir != wantDir || parts[1] != "y" || parts[2] != "/bin/sh" {
		t.Errorf("PWD, PROBE_X, $0 = %q; want %s, y, /bin/sh", parts, dir)
	}
}
