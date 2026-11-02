package org

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// realAbsPath is absPath as spawn.go sets it, read when the package's
// variables are initialized, before any test runs, so a test that reassigns
// absPath to a failing stub (TestMustAbs_KeepsThePathAsGivenWhenAbsFails and
// the dry-run spawn tests) cannot leave that stub in what
// TestAbsPath_GetwdFailsOnLinuxOnceTheWorkingDirectoryIsRemoved calls.
var realAbsPath = absPath

// TestAbsPath_GetwdFailsOnLinuxOnceTheWorkingDirectoryIsRemoved checks the
// premise of D7 of plan 2026-11-01-org-ledger-gaps, which the tests of
// mustAbs's fallback stand in for by reassigning absPath: with the working
// directory removed, os.Getwd fails, and so does the real absPath of a
// relative path. darwin's os.Getwd still returns the old path then, so the
// test runs on Linux only, in CI on ubuntu-latest (D7 of plan
// 2026-11-02-org-ledger-leftovers, 420(d) of docs/tech-debt/README.md).
func TestAbsPath_GetwdFailsOnLinuxOnceTheWorkingDirectoryIsRemoved(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("os.Getwd does not fail for a deleted working directory on " + runtime.GOOS)
	}
	dir := filepath.Join(t.TempDir(), "gone")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	const rel = "rel/x"
	if got, err := realAbsPath(rel); err == nil {
		t.Errorf("absPath(%q) = %q, nil with the working directory %s removed; want the error of os.Getwd", rel, got, dir)
	}
}
