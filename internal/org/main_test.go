package org

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
	"github.com/yoshpy-dev/ralph/internal/toolchain"
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
//
// It also pins the version probe a gate runs before its scripts
// (gateToolsProbe) to testGateToolsProbe, which runs nothing and reports a
// fixed version for each tool it is asked for, so the gate log, the Details
// of a gate record, and GateResult.Tools and RegateResult.Tools do not change
// with the tools on the machine: for the fixtures, whose repositories have no
// JS marker and no go.mod, python3 3.12.1 at the root (testGateToolsDetails,
// testGateTools, testGateToolsLog). A test of the real probe sets it back
// with setGateToolsProbe.
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
	gateToolsProbe = testGateToolsProbe
	os.Exit(m.Run())
}

// testGateToolVersions are the versions testGateToolsProbe reports.
var testGateToolVersions = map[string]string{
	toolchain.ToolNode: "v24.22.1", toolchain.ToolPnpm: "10.10.0", toolchain.ToolYarn: "4.5.0", toolchain.ToolNpm: "10.9.2",
	toolchain.ToolPython3: "3.12.1", toolchain.ToolGo: "go1.27.3",
}

// testGateToolsProbe is the probe TestMain pins: it runs nothing and reports
// each target found at /usr/bin/<tool> with its testGateToolVersions version,
// with no mise; when ctx is done it returns ctx's cause, as toolchain.Probe
// does.
func testGateToolsProbe(ctx context.Context, opts toolchain.ProbeOptions) (toolchain.Report, error) {
	if ctx.Err() != nil {
		return toolchain.Report{}, context.Cause(ctx)
	}
	rep := toolchain.Report{Complete: true}
	for _, t := range opts.Targets {
		rep.Measurements = append(rep.Measurements, toolchain.Measurement{Dir: t.Dir, Tool: t.Tool, Status: toolchain.StatusFound,
			Path: "/usr/bin/" + t.Tool, Version: testGateToolVersions[t.Tool]})
	}
	return rep, nil
}

// testGateToolsDetails is what the Details of a gate record of scripts that
// ran in a fixture hold for the pinned probe, after the load keys (and
// max_load=) and before log=, with its leading space and no trailing one.
const testGateToolsDetails = " tool_dirs=1 tool_python3=3.12.1"

// testGateToolsLog is the log's lines about the tools for the pinned probe in
// a fixture, right after its `# load at start:` line.
const testGateToolsLog = gateToolsExplainLine + "\n# tool python3 (.): 3.12.1 (/usr/bin/python3)\n"

// testGateTools is GateResult.Tools and RegateResult.Tools for scripts that
// ran in a fixture with the pinned probe.
func testGateTools() *GateTools {
	return &GateTools{Dirs: 1, Tools: []GateTool{{Name: toolchain.ToolPython3, Value: "3.12.1"}}}
}

// setGateToolsProbe makes gateToolsProbe probe until the test ends, when
// TestMain's pin comes back.
func setGateToolsProbe(t *testing.T, probe func(context.Context, toolchain.ProbeOptions) (toolchain.Report, error)) {
	t.Helper()
	old := gateToolsProbe
	t.Cleanup(func() { gateToolsProbe = old })
	gateToolsProbe = probe
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
