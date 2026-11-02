package cli

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestOrgRetiredStateDirEnv_KeepsARelativeValueAsGivenWhenAbsFails covers
// AC9 of plan 2026-11-01-org-ledger-gaps for the branch of
// checkRetiredOrgStateDirEnv that uses the variable's value as given when
// making it absolute fails, which filepath.Abs does only when os.Getwd fails.
// Removing the working directory does not make os.Getwd fail on macOS, so the
// test reassigns retiredOrgStateDirEnvAbs to a failing one. A relative value
// then appears as it was set in the note of a verb that goes on (the flag
// names the ledger, or a read-only verb) and in the refusal of a mutating
// verb, which also quotes it in the lines that keep the variable's ledger;
// while filepath.Abs works, the same note names an absolute path.
func TestOrgRetiredStateDirEnv_KeepsARelativeValueAsGivenWhenAbsFails(t *testing.T) {
	const rel = "old/ledger"
	t.Setenv(retiredOrgStateDirEnv, rel)
	defaultLedger := filepath.Join(t.TempDir(), ".harness", "state", "org")
	flagLedger := filepath.Join(t.TempDir(), "flag-ledger")

	check := func(source, ledger string, access orgLedgerAccess) (stderr string, err error) {
		var buf bytes.Buffer
		cmd := &cobra.Command{Use: "spawn"}
		cmd.SetErr(&buf)
		err = checkRetiredOrgStateDirEnv(cmd, ledger, source, access)
		return buf.String(), err
	}

	note, err := check("flag", flagLedger, orgLedgerMutating)
	if err != nil || !strings.HasPrefix(note, "note: "+retiredOrgStateDirEnv+" is set (/") {
		t.Fatalf("with filepath.Abs working: err = %v, stderr = %q; want the note to name an absolute path", err, note)
	}

	orig := retiredOrgStateDirEnvAbs
	retiredOrgStateDirEnvAbs = func(string) (string, error) { return "", errors.New("stub: getwd failed") }
	t.Cleanup(func() { retiredOrgStateDirEnvAbs = orig })

	wantNote := "note: " + retiredOrgStateDirEnv + " is set (" + rel + "), but ralph no longer reads it; "
	for _, tc := range []struct {
		name, source, ledger string
		access               orgLedgerAccess
	}{
		{"--state-dir names the ledger", "flag", flagLedger, orgLedgerMutating},
		{"a read-only verb on the default ledger", "git-main-worktree", defaultLedger, orgLedgerReadOnly},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stderr, err := check(tc.source, tc.ledger, tc.access)
			if err != nil {
				t.Errorf("err = %v; want only a note", err)
			}
			if !strings.HasPrefix(stderr, wantNote) || strings.Count(stderr, "\n") != 1 {
				t.Errorf("stderr = %q; want one note line starting %q", stderr, wantNote)
			}
		})
	}

	t.Run("a mutating verb on the default ledger", func(t *testing.T) {
		stderr, err := check("git-main-worktree", defaultLedger, orgLedgerMutating)
		if err == nil {
			t.Fatalf("err = nil, stderr = %q; want a refusal", stderr)
		}
		if stderr != "" {
			t.Errorf("stderr = %q; want nothing with the refusal", stderr)
		}
		for _, want := range []string{
			retiredOrgStateDirEnv + " names " + rel + ", which ralph no longer reads",
			"nothing was changed",
			"to keep using " + rel + ", write [org].state_dir = \"" + rel + "\"",
			"or pass --state-dir " + rel + ", and unset " + retiredOrgStateDirEnv,
		} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("err = %q; want it to contain %q", err, want)
			}
		}
	})
}
