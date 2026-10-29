package config

import (
	"fmt"
	"os"
	"testing"
)

// TestMain sets RALPH_LOCAL_CONFIG (LocalConfigEnv) to off, so Load reads
// no personal ralph.local.toml, for two reasons. A test that loads a file of
// this repository (TestLoad_TemplateRalphToml loads templates/base/ralph.toml)
// would otherwise read the developer's ralph.local.toml at the main
// worktree's root (LocalPath), and its result would depend on that file. And
// with off, Load does not look for the file and runs no git for it. A test
// of the personal file (local_test.go) sets it to "" (t.Setenv), which reads
// the file.
func TestMain(m *testing.M) {
	if err := os.Setenv(LocalConfigEnv, "off"); err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: set %s: %v\n", LocalConfigEnv, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
