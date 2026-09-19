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
// without this seam, every pre-existing runDoctor* call site would silently
// start reading every candidate rc file under the real $HOME.
//
// It also pins the codex-sandbox check's environment resolver
// (doctorCodexSandboxEnv) to a config path that does not exist, for the same
// reason: without this seam, every runDoctor*-based test would silently
// start reading the developer's real ~/.codex/config.toml.
func TestMain(m *testing.M) {
	doctorShellAliasEnv = func() (shellAliasEnv, error) {
		return shellAliasEnv{Home: filepath.Join(os.TempDir(), "ralph-cli-test-no-such-home")}, nil
	}
	doctorCodexSandboxEnv = func() (codexSandboxEnv, error) {
		return codexSandboxEnv{ConfigPath: filepath.Join(os.TempDir(), "ralph-cli-test-no-such-home", ".codex", "config.toml")}, nil
	}
	os.Exit(m.Run())
}
