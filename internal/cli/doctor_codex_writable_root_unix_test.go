//go:build !windows

package cli

import (
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
