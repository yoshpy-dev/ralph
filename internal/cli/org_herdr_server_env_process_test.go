package cli

import (
	"context"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// The read of the herdr server's CLAUDE_CONFIG_DIR against a real process
// (plan 2026-11-20-org-reseat-restart-herdr, gap 1 of
// docs/reports/test-2026-11-20-org-reseat-restart-herdr.md). TestMain sets
// orgHerdrServerClaudeConfigDirReader to nil, and the other tests of the
// read give it a fake probe, so none of them runs the probe production
// builds (orgHerdrServerEnvProbeNow). The test below does, with only the
// status read faked: the real lsof finds the pid that has the socket open,
// and the real ps (macOS) or /proc (Linux) reads that process's environment.
// The process is a child the test starts, the test binary itself, which
// macOS ps prints the environment of (it does not for an Apple platform
// binary such as /bin/sleep). No herdr runs, and nothing reaches the
// machine's herdr server or its socket.

// orgHerdrServerEnvListenerSocketEnv names the variable that makes the test
// binary the child: its value is the socket the child listens on.
const orgHerdrServerEnvListenerSocketEnv = "RALPH_TEST_HERDR_SERVER_ENV_LISTENER_SOCKET"

// TestOrgHerdrServerReadClaudeConfigDir_ListenerProcess is the child of
// TestOrgHerdrServerReadClaudeConfigDir_ReadsARealProcess, and does nothing
// in any other run. It listens on the unix socket
// orgHerdrServerEnvListenerSocketEnv names until the parent kills it, or
// until its stdin ends (the parent's end of the pipe closes, as it does when
// the parent exits), or for a minute at most.
func TestOrgHerdrServerReadClaudeConfigDir_ListenerProcess(t *testing.T) {
	sock := os.Getenv(orgHerdrServerEnvListenerSocketEnv)
	if sock == "" {
		return
	}
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen on %s: %v", sock, err)
	}
	defer func() { _ = l.Close() }()
	stdinEnded := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, os.Stdin)
		close(stdinEnded)
	}()
	select {
	case <-stdinEnded:
	case <-time.After(time.Minute):
	}
}

// orgHerdrServerEnvSocketDir makes a directory for the child's socket, under
// os.TempDir(), or under /tmp when that path would be too long for a unix
// socket (104 bytes on macOS with the name), and removes it at cleanup.
func orgHerdrServerEnvSocketDir(t *testing.T, name string) string {
	t.Helper()
	for _, parent := range []string{"", "/tmp"} {
		dir, err := os.MkdirTemp(parent, "rrh")
		if err != nil {
			t.Fatalf("make the socket's directory: %v", err)
		}
		if len(filepath.Join(dir, name)) < 100 {
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			return dir
		}
		_ = os.RemoveAll(dir)
	}
	t.Fatalf("no directory for the socket %s is short enough, under %s or /tmp", name, os.TempDir())
	return ""
}

// AC6 against a real process: orgHerdrServerEnvProbeNow, the probe of
// production, with only the status read faked to a running server whose
// socket a child process listens on, reads the child's CLAUDE_CONFIG_DIR as
// it is. The value has a space and non-ASCII characters (ps eww runs under
// en_US.UTF-8 on macOS; Linux reads /proc/<pid>/environ), and the entry after
// it has a name that is not an identifier, X-Y, so the value has to end where
// that entry starts. The child is killed by its pid at cleanup.
func TestOrgHerdrServerReadClaudeConfigDir_ReadsARealProcess(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("the read of a real process is checked on darwin (lsof and ps) and linux (lsof and /proc) only, not %s", runtime.GOOS)
	}
	if testing.Short() {
		t.Skip("starts a child process and runs lsof on its socket; skipped under -short")
	}
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skipf("lsof, which the read runs to find the server's pid, is not on PATH: %v", err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	value := filepath.Join(t.TempDir(), "日本語 dir")
	if err := os.Mkdir(value, 0o700); err != nil {
		t.Fatal(err)
	}
	dir := orgHerdrServerEnvSocketDir(t, "herdr.sock")
	sock := filepath.Join(dir, "herdr.sock")
	logPath := filepath.Join(dir, "listener.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = logFile.Close() }()

	cmd := exec.Command(exe, "-test.run", "^TestOrgHerdrServerReadClaudeConfigDir_ListenerProcess$")
	cmd.Env = []string{
		orgHerdrServerEnvListenerSocketEnv + "=" + sock,
		"CLAUDE_CONFIG_DIR=" + value,
		"X-Y=next",
	}
	// A pipe, not /dev/null, so the child's stdin ends only when this
	// process closes it (Wait) or exits.
	if _, err := cmd.StdinPipe(); err != nil {
		t.Fatal(err)
	}
	cmd.Stdout, cmd.Stderr = logFile, logFile
	if err := cmd.Start(); err != nil {
		t.Fatalf("start the listener: %v", err)
	}
	// exited closes once Wait returns, after which waitErr holds its error.
	exited := make(chan struct{})
	var waitErr error
	go func() {
		waitErr = cmd.Wait()
		close(exited)
	}()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		<-exited
	})
	childLog := func() string {
		data, _ := os.ReadFile(logPath)
		return string(data)
	}
	for deadline := time.Now().Add(20 * time.Second); ; {
		if info, err := os.Stat(sock); err == nil && info.Mode().Type() == fs.ModeSocket {
			break
		}
		select {
		case <-exited:
			t.Fatalf("the listener (pid %d) exited before its socket appeared: %v; its output:\n%s", cmd.Process.Pid, waitErr, childLog())
		case <-time.After(10 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			t.Fatalf("the listener's socket %s did not appear in 20s; its output:\n%s", sock, childLog())
		}
	}

	origStatus, origTimeout := orgHerdrServerStatus, orgHerdrServerEnvCallTimeout
	t.Cleanup(func() { orgHerdrServerStatus, orgHerdrServerEnvCallTimeout = origStatus, origTimeout })
	orgHerdrServerStatus = func(context.Context) (driver.HerdrServerStatus, error) {
		return driver.HerdrServerStatus{Running: true, Socket: sock}, nil
	}
	// lsof walks every process; under load it can take longer than the
	// read's 5 s bound, which this test does not check.
	orgHerdrServerEnvCallTimeout = 30 * time.Second

	got := readOrgHerdrServerClaudeConfigDir(context.Background(), orgHerdrServerEnvProbeNow())
	if want := (orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvSet, value: value}); got != want {
		t.Errorf("read of the listener (pid %d) = %+v; want %+v", cmd.Process.Pid, got, want)
	}
}
