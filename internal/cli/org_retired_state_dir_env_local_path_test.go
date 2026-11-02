package cli

import (
	"bytes"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// TestOrgRetiredStateDirEnv_RefusalOfALedgerElsewhereNamesTheWorkingDirectorysLocalFile
// covers the branch of checkRetiredOrgStateDirEnv where
// org.LedgerLocalConfigPath returns ok == false (420(d) of
// docs/tech-debt/README.md, plan 2026-11-02-org-ledger-leftovers, AC7). The
// default tiers resolve a ledger at <dir>/.harness/state/org, so no run of a
// verb reaches it, and the test calls the function directly: for a ledger
// that does not end with .harness/state/org, on each default tier, a
// mutating verb is refused while RALPH_ORG_STATE_DIR names another directory,
// and the refusal names config.LocalPath("ralph.toml"), the personal file of
// the working directory (a temp dir outside git), as the place to write
// [org].state_dir.
func TestOrgRetiredStateDirEnv_RefusalOfALedgerElsewhereNamesTheWorkingDirectorysLocalFile(t *testing.T) {
	isolateFeatureGitEnv(t)
	cwd := t.TempDir()
	t.Chdir(cwd)
	local := config.LocalPath("ralph.toml")
	if want := filepath.Join(cwd, config.LocalConfigFileName); local != want {
		t.Fatalf("config.LocalPath(%q) = %q in %s, outside git; want %q", "ralph.toml", local, cwd, want)
	}
	envDir := filepath.Join(t.TempDir(), "old ledger")
	t.Setenv(retiredOrgStateDirEnv, envDir)
	ledger := filepath.Join(t.TempDir(), "ledger")

	for _, source := range []string{"git-main-worktree", "git-toplevel", "cwd"} {
		t.Run(source, func(t *testing.T) {
			if path, ok := org.LedgerLocalConfigPath(ledger, source); ok {
				t.Fatalf("org.LedgerLocalConfigPath(%q, %q) = %q, true; want ok false for a ledger elsewhere", ledger, source, path)
			}
			var stderr bytes.Buffer
			cmd := &cobra.Command{Use: "spawn"}
			cmd.SetErr(&stderr)
			err := checkRetiredOrgStateDirEnv(cmd, ledger, source, orgLedgerMutating)
			if err == nil {
				t.Fatalf("err = nil, stderr = %q; want a refusal", stderr.String())
			}
			if stderr.Len() != 0 {
				t.Errorf("stderr = %q; want nothing with the refusal", stderr.String())
			}
			for _, want := range []string{
				retiredOrgStateDirEnv + " names " + envDir + ", which ralph no longer reads, and the ledger ralph finds by default is " +
					ledger + ", so nothing was changed",
				"write [org].state_dir = " + strconv.Quote(envDir) + " in " + local + " or pass --state-dir " +
					org.ShellWord(envDir) + ", and unset " + retiredOrgStateDirEnv,
			} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("err = %q; want it to contain %q", err, want)
				}
			}
		})
	}
}
