package org

import (
	"fmt"
	"os"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// TestMain sets RALPH_LOCAL_CONFIG (config.LocalConfigEnv) to off, so
// config.Load reads no personal ralph.local.toml, for two reasons. A test
// whose working directory or config path is inside this repository would
// otherwise read the developer's ralph.local.toml at the main worktree's
// root (config.LocalPath), and its result would depend on that file. And
// with off, Load does not look for the file and runs no git for it. A test
// that needs the personal file sets it to "" (t.Setenv), which reads the
// file.
func TestMain(m *testing.M) {
	if err := os.Setenv(config.LocalConfigEnv, "off"); err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: set %s: %v\n", config.LocalConfigEnv, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
