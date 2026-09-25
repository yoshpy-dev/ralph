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

// unblockFIFOOpen opens path for writing, non-blocking, and immediately
// closes it -- releasing a goroutine that is still blocked in os.Open(path)
// on the reading side. Test cleanups call this unconditionally after a FIFO
// test finishes (whether it passed or the 5s timeout fired), so a failing
// run never leaves the blocked goroutine running past the test. When the
// check under test never opened the FIFO at all (the fix working as
// intended), there is no blocked reader to release and the non-blocking
// write-side open simply fails; that failure is ignored. os.OpenFile has no
// portable O_NONBLOCK flag, hence syscall.Open here rather than os.
func unblockFIFOOpen(path string) {
	fd, err := syscall.Open(path, syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return
	}
	_ = syscall.Close(fd)
}

// checkShellAliasesWithin runs checkShellAliases in a goroutine and returns
// its result, or fails the test if it does not return within 5s -- the
// shared guard every FIFO test in this file needs, since opening a FIFO for
// read can block forever and a stuck goroutine must not hang the test
// binary itself (the goroutine leaks harmlessly past the failing test; see
// unblockFIFOOpen).
func checkShellAliasesWithin(t *testing.T, resolveEnv func() (shellAliasEnv, error), herdrPresent bool) checkResult {
	t.Helper()
	done := make(chan checkResult, 1)
	go func() {
		done <- checkShellAliases(resolveEnv, herdrPresent)
	}()
	select {
	case r := <-done:
		return r
	case <-time.After(5 * time.Second):
		t.Fatal("checkShellAliases did not return within 5s -- likely blocked opening the FIFO")
		return checkResult{} // unreachable: t.Fatal stops this goroutine
	}
}

// TestCheckShellAliases_ZshrcIsFIFO_InfoAndCompletes is AC-1: a FIFO
// candidate rc must never reach os.Open -- opening it would block until a
// writer attaches, hanging the whole doctor run. This test (and
// syscall.Mkfifo, which it depends on) only exists on unix, hence this
// file's build tag: internal/cli's own test suite is already unix-only
// (internal/org/lockfile.go uses syscall.Flock unconditionally), so no
// runtime.GOOS check is needed here.
func TestCheckShellAliases_ZshrcIsFIFO_InfoAndCompletes(t *testing.T) {
	home := t.TempDir()
	fifoPath := filepath.Join(home, ".zshrc")
	if err := syscall.Mkfifo(fifoPath, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { unblockFIFOOpen(fifoPath) })

	r := checkShellAliasesWithin(t, shellAliasTestEnv(home), true)
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	want := "could not read: ~" + string(filepath.Separator) + ".zshrc (not a regular file)"
	if !strings.Contains(r.Detail, want) {
		t.Errorf("expected detail to contain %q, got: %s", want, r.Detail)
	}
}

// TestCheckShellAliases_ZshrcSymlinksToFIFO_InfoNamedBySymlink is AC-2: a
// candidate that is a symlink to a FIFO elsewhere must be rejected the same
// way, reported under the symlink candidate's own name (not the FIFO's
// real path) -- checkShellAliases always names the candidate as scanned,
// never the resolved target.
func TestCheckShellAliases_ZshrcSymlinksToFIFO_InfoNamedBySymlink(t *testing.T) {
	home := t.TempDir()
	elsewhere := t.TempDir()
	fifoPath := filepath.Join(elsewhere, "fifo-target")
	if err := syscall.Mkfifo(fifoPath, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { unblockFIFOOpen(fifoPath) })
	if err := os.Symlink(fifoPath, filepath.Join(home, ".zshrc")); err != nil {
		t.Fatal(err)
	}

	r := checkShellAliasesWithin(t, shellAliasTestEnv(home), true)
	if r.Status != "info" {
		t.Fatalf("expected info, got %s (%s)", r.Status, r.Detail)
	}
	want := "could not read: ~" + string(filepath.Separator) + ".zshrc (not a regular file)"
	if !strings.Contains(r.Detail, want) {
		t.Errorf("expected detail to contain %q, got: %s", want, r.Detail)
	}
}

// TestCheckShellAliases_FIFOAndOtherRcAlias_WarnsAndNamesBoth is AC-3: a
// FIFO candidate must not stop the scan of the remaining candidates -- a
// codex alias found in another rc (with herdr present) still warns, and the
// Detail names both the FIFO's unreadable clause and the alias finding.
func TestCheckShellAliases_FIFOAndOtherRcAlias_WarnsAndNamesBoth(t *testing.T) {
	home := t.TempDir()
	fifoPath := filepath.Join(home, ".zshrc")
	if err := syscall.Mkfifo(fifoPath, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { unblockFIFOOpen(fifoPath) })
	writeAliasRc(t, filepath.Join(home, ".bashrc"), `alias codex="codex -m x"`+"\n")

	r := checkShellAliasesWithin(t, shellAliasTestEnv(home), true)
	if r.Status != "warn" {
		t.Fatalf("expected warn, got %s (%s)", r.Status, r.Detail)
	}
	for _, want := range []string{
		"could not read: ~" + string(filepath.Separator) + ".zshrc (not a regular file)",
		"--model (-m)",
	} {
		if !strings.Contains(r.Detail, want) {
			t.Errorf("expected detail to contain %q, got: %s", want, r.Detail)
		}
	}
}
