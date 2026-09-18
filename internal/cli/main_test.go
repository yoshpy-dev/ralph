package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain pins the shell-alias check's environment resolver
// (doctorShellAliasEnv) to a home directory that does not exist, so no
// runDoctor*-based test in this package reads the developer's real shell rc
// files (self-review L7, docs/reports/self-review-2026-09-18-doctor-shell-alias-check.md):
// without this seam, every one of the 13 pre-existing runDoctor* call sites
// would silently start opening up to 11 files under the real $HOME.
func TestMain(m *testing.M) {
	doctorShellAliasEnv = func() (shellAliasEnv, error) {
		return shellAliasEnv{Home: filepath.Join(os.TempDir(), "ralph-cli-test-no-such-home")}, nil
	}
	os.Exit(m.Run())
}
