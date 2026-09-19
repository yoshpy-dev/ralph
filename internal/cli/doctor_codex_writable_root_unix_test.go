//go:build !windows

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestCheckCodexAgmsgWritableRoot_ConfigIsFIFO_InfoAndCompletes proves a
// FIFO config path is rejected WITHOUT ever being opened -- opening it
// would block forever, hanging the whole doctor run. This test (and
// syscall.Mkfifo, which it depends on) only exists on unix, hence this
// file's build tag: internal/cli's own test suite is already unix-only
// (internal/org/lockfile.go uses syscall.Flock unconditionally), so no
// runtime.GOOS check is needed here -- the build tag alone keeps this file
// out of a windows build entirely (self-review LOW-7).
func TestCheckCodexAgmsgWritableRoot_ConfigIsFIFO_InfoAndCompletes(t *testing.T) {
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.toml")
	if err := syscall.Mkfifo(cfgPath, 0o644); err != nil {
		t.Fatal(err)
	}

	type result struct {
		r checkResult
	}
	done := make(chan result, 1)
	go func() {
		done <- result{checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))}
	}()

	select {
	case got := <-done:
		if got.r.Status != "info" {
			t.Fatalf("expected info, got %s (%s)", got.r.Status, got.r.Detail)
		}
		if !strings.Contains(got.r.Detail, "not a regular file") {
			t.Errorf("expected detail to mention not a regular file, got: %s", got.r.Detail)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("checkCodexAgmsgWritableRoot did not return within 5s -- likely blocked opening the FIFO")
	}
}

// TestCheckCodexAgmsgWritableRoot_UnreadableConfig_InfoWithReasonOnly pins
// the *fs.PathError branch of codexConfigReadReason: a config that exists
// but cannot be opened is reported with the inner reason only (the Detail
// already names the file), and the check stays informational. Unix-only:
// mode 000 does not deny reads on windows, and root ignores it.
func TestCheckCodexAgmsgWritableRoot_UnreadableConfig_InfoWithReasonOnly(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read a mode-000 file")
	}
	agmsgHome := t.TempDir()
	writeAgmsgHome(t, agmsgHome, "")
	cfgPath := writeCodexConfig(t, t.TempDir(), "sandbox_mode = \"workspace-write\"\n")
	if err := os.Chmod(cfgPath, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(cfgPath, 0o644) })

	r := checkCodexAgmsgWritableRoot(codexOrgConfig(true, "", nil), agmsgHome, codexSandboxTestEnv(cfgPath, ""))
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	want := "could not read " + cfgPath + " (permission denied)"
	if !strings.Contains(r.Detail, want) {
		t.Errorf("expected detail to contain %q, got: %s", want, r.Detail)
	}
	if strings.Count(r.Detail, cfgPath) != 1 {
		t.Errorf("the config path must appear once (the reason must not repeat it), got: %s", r.Detail)
	}
}
