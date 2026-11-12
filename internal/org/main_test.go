package org

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// TestMain sets RALPH_LOCAL_CONFIG (config.LocalConfigEnv) to off, so
// config.Load reads no personal ralph.local.toml, for two reasons. A test
// whose working directory or config path is inside this repository would
// otherwise read the developer's ralph.local.toml at the main worktree's
// root (config.LocalPath), and its result would depend on that file. And
// with off, Load does not look for the file and runs no git for it. A test
// that needs the personal file sets it to "" (t.Setenv), which reads the
// file.
//
// It also pins driver.AgmsgHomeOverride to an agmsg home that does not
// exist, so no test reaches the developer's real agmsg (the default
// ~/.agents/skills/agmsg): a role prompt rendered with no AgmsgHome reads
// driver.ResolveAgmsgHome(""), which returns the override first. A test of
// how ResolveAgmsgHome("") falls back to the default home clears it
// (setAgmsgHomeOverride).
//
// It also unsets RALPH_ORG_STATE_DIR (retiredOrgStateDirEnv), which
// ResolveOrgStateDir no longer reads, so no test sees a developer's value
// of it: the tests that check the variable is ignored set it with t.Setenv.
//
// It also pins the load average a gate reads (gateLoadReader) to
// testGateLoadAvg and the CPU count (gateNumCPU) to testGateNumCPU, so the
// Details of a gate record, the gate log, and GateResult.Load and
// RegateResult.Load (testGateLoad) do not change with the machine's load. A
// test of the real reader calls readGateLoad; a test that needs other values
// sets the two with setGateLoadReader.
func TestMain(m *testing.M) {
	if err := os.Setenv(config.LocalConfigEnv, "off"); err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: set %s: %v\n", config.LocalConfigEnv, err)
		os.Exit(1)
	}
	if err := os.Unsetenv(retiredOrgStateDirEnv); err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: unset %s: %v\n", retiredOrgStateDirEnv, err)
		os.Exit(1)
	}
	driver.AgmsgHomeOverride = filepath.Join(os.TempDir(), "ralph-org-test-no-such-home", "agmsg")
	gateLoadReader = func(context.Context) (gateLoadAvg, error) { return testGateLoadAvg, nil }
	gateNumCPU = func() int { return testGateNumCPU }
	os.Exit(m.Run())
}

// testGateLoadAvg and testGateNumCPU are the load average and the CPU count
// TestMain pins for every gate.
var testGateLoadAvg = gateLoadAvg{one: 1.5, five: 1.25, fifteen: 1}

const testGateNumCPU = 8

// testGateLoadDetails is what the Details of a gate record of scripts that
// ran hold for the pinned load (testGateLoadAvg), before log= and after
// herdr_pane= or via=, with its leading space and no trailing one.
const testGateLoadDetails = " load_start=1.50 load_max=1.50 load_end=1.50 ncpu=8"

// testGateLoad is GateResult.Load and RegateResult.Load for scripts that ran
// with the pinned load and no [org.gate].max_load.
func testGateLoad() *GateLoad {
	return &GateLoad{Start: "1.50", Highest: "1.50", End: "1.50", NCPU: testGateNumCPU}
}

// setGateLoadReader makes gateLoadReader return read's values and gateNumCPU
// ncpu until the test ends, when TestMain's pins come back.
func setGateLoadReader(t *testing.T, read gateLoadReadFunc, ncpu int) {
	t.Helper()
	oldRead, oldCPU := gateLoadReader, gateNumCPU
	t.Cleanup(func() { gateLoadReader, gateNumCPU = oldRead, oldCPU })
	gateLoadReader = read
	gateNumCPU = func() int { return ncpu }
}

// setAgmsgHomeOverride points driver.AgmsgHomeOverride at home until the
// test ends, when the value it had comes back. An empty home clears the
// override, so driver.ResolveAgmsgHome returns [org].agmsg_home, else the
// default home, as it does outside the tests.
func setAgmsgHomeOverride(t *testing.T, home string) {
	t.Helper()
	old := driver.AgmsgHomeOverride
	t.Cleanup(func() { driver.AgmsgHomeOverride = old })
	driver.AgmsgHomeOverride = home
}
