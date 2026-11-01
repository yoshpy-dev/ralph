package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// retiredEnvGitLogger is the git of
// TestOrgRetiredStateDirEnv_RefusalNamesThePersonalFileWithoutGit (fmt verb:
// the real git's path, shell-quoted): it logs its arguments to
// $RETIRED_ENV_GIT_LOG and execs the real git.
const retiredEnvGitLogger = `#!/bin/sh
printf '%%s\n' "$*" >> "$RETIRED_ENV_GIT_LOG"
exec %s "$@"
`

// TestOrgRetiredStateDirEnv_RefusalNamesThePersonalFileWithoutGit covers
// AC7 of plan 2026-11-01-org-ledger-gaps for the refusal of
// checkRetiredOrgStateDirEnv: it names the ralph.local.toml that
// config.LocalPath("ralph.toml") gives from the working directory, as it did
// when it called that function, but reads the path back from the ledger
// org.ResolveOrgStateDir returned and starts no git. The git on PATH logs
// each start and runs the real git.
func TestOrgRetiredStateDirEnv_RefusalNamesThePersonalFileWithoutGit(t *testing.T) {
	r := newLegacyLedgerRepo(t)
	outside, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("git is needed: %v", err)
	}
	if realGit, err = filepath.Abs(realGit); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	writeStubScript(t, filepath.Join(bin, "git"), fmt.Sprintf(retiredEnvGitLogger, shellQuoteIfNeeded(realGit)))
	gitLog := filepath.Join(t.TempDir(), "git.log")
	t.Setenv("RETIRED_ENV_GIT_LOG", gitLog)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	for _, tc := range []struct{ name, cwd, wantSource string }{
		{"a linked worktree", filepath.Join(r.worktree, "pkg"), "git-main-worktree"},
		{"outside git", outside, "cwd"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.MkdirAll(tc.cwd, 0o755); err != nil {
				t.Fatal(err)
			}
			t.Chdir(tc.cwd)
			dir, source, err := org.ResolveOrgStateDir("", false)
			if err != nil || source != tc.wantSource {
				t.Fatalf("ResolveOrgStateDir = %q, %q, %v; want source %q", dir, source, err, tc.wantSource)
			}
			wantPath := config.LocalPath("ralph.toml")
			if len(readLogLines(t, gitLog)) == 0 {
				t.Fatal("the git wrapper logged no call for ResolveOrgStateDir and config.LocalPath; it is not the git on PATH")
			}
			if err := os.Remove(gitLog); err != nil {
				t.Fatal(err)
			}
			t.Setenv(retiredOrgStateDirEnv, filepath.Join(t.TempDir(), "old ledger"))

			err = checkRetiredOrgStateDirEnv(&cobra.Command{Use: "spawn"}, dir, source, orgLedgerMutating)
			if want := " in " + wantPath + " or pass --state-dir "; err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("checkRetiredOrgStateDirEnv = %v; want a refusal containing %q", err, want)
			}
			if calls := readLogLines(t, gitLog); len(calls) != 0 {
				t.Errorf("checkRetiredOrgStateDirEnv started git %d times: %q; want none", len(calls), calls)
			}
		})
	}
}
