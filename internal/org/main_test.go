package org

import (
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
func TestMain(m *testing.M) {
	if err := os.Setenv(config.LocalConfigEnv, "off"); err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: set %s: %v\n", config.LocalConfigEnv, err)
		os.Exit(1)
	}
	driver.AgmsgHomeOverride = filepath.Join(os.TempDir(), "ralph-org-test-no-such-home", "agmsg")
	os.Exit(m.Run())
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
