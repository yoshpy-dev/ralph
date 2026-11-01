package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/gitroot"
)

// TestRetiredOrgStateDirRefusalCondition covers AC4 of plan
// 2026-11-01-org-ledger-gaps: the condition under which the verbs that change
// the ledger are refused while RALPH_ORG_STATE_DIR is set is one constant,
// retiredOrgStateDirRefusalCondition. Four cases (--state-dir names the
// ledger, the personal file names it, the default ledger is the directory the
// variable names, the default ledger is another directory) run through
// checkRetiredOrgStateDirEnv with both accesses, and each case's outcome is
// compared with the condition the constant states and with the table's own
// expectation. In every case the doctor's warn for the variable and the help
// of `ralph org escalate` (white space collapsed) contain the constant.
func TestRetiredOrgStateDirRefusalCondition(t *testing.T) {
	// The constant names what the judgment reads: the flag, the personal
	// file, the variable, and the directory it names.
	for _, clause := range []string{"--state-dir", "[org].state_dir", retiredOrgStateDirEnv, "another directory than the ledger"} {
		if !strings.Contains(retiredOrgStateDirRefusalCondition, clause) {
			t.Errorf("retiredOrgStateDirRefusalCondition = %q; want it to contain %q", retiredOrgStateDirRefusalCondition, clause)
		}
	}

	var doctorWarn string
	for _, r := range checkRetiredEnv(func(name string) string {
		if name == retiredOrgStateDirEnv {
			return "/old ledger"
		}
		return ""
	}) {
		if r.Name == retiredOrgStateDirEnv {
			doctorWarn = r.Detail
		}
	}
	if doctorWarn == "" {
		t.Fatalf("checkRetiredEnv has no warn for %s", retiredOrgStateDirEnv)
	}
	helpOut, err := runOrgCmd(t, "escalate", "--help")
	if err != nil {
		t.Fatalf("escalate --help: %v", err)
	}
	escalateHelp := strings.Join(strings.Fields(helpOut), " ")

	// The default ledger ends in .harness/state/org, so the refusal reads the
	// personal file's path back from it with no git call.
	defaultLedger := filepath.Join(t.TempDir(), ".harness", "state", "org")
	otherDir := t.TempDir()
	flagLedger := filepath.Join(t.TempDir(), "flag-ledger")
	configLedger := filepath.Join(t.TempDir(), "config-ledger")

	for _, tc := range []struct {
		name        string
		source      string
		ledger      string
		env         string
		wantRefused bool
	}{
		{"--state-dir names the ledger", "flag", flagLedger, otherDir, false},
		{"the personal file names the ledger", "config", configLedger, otherDir, false},
		{"the default ledger is the directory the variable names", "git-main-worktree", defaultLedger, defaultLedger, false},
		{"the default ledger is another directory", "git-main-worktree", defaultLedger, otherDir, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(retiredOrgStateDirEnv, tc.env)

			// The case's facts, read as the constant states them: refused
			// exactly when neither the flag nor the personal file names the
			// ledger and the variable names another directory.
			namesLedger := tc.source == "flag" || tc.source == "config"
			conditionHolds := !namesLedger && !gitroot.SamePath(tc.env, tc.ledger)
			if conditionHolds != tc.wantRefused {
				t.Fatalf("the condition %q holds = %v for source %q, ledger %q, variable %q; the case expects a refusal = %v",
					retiredOrgStateDirRefusalCondition, conditionHolds, tc.source, tc.ledger, tc.env, tc.wantRefused)
			}

			for _, access := range []struct {
				name   string
				access orgLedgerAccess
			}{{"mutating", orgLedgerMutating}, {"read-only", orgLedgerReadOnly}} {
				var stderr bytes.Buffer
				cmd := &cobra.Command{Use: "spawn"}
				cmd.SetErr(&stderr)
				err := checkRetiredOrgStateDirEnv(cmd, tc.ledger, tc.source, access.access)
				refuses := conditionHolds && access.access == orgLedgerMutating
				if refuses {
					if err == nil || !strings.Contains(err.Error(), tc.ledger) || !strings.Contains(err.Error(), "nothing was changed") {
						t.Errorf("%s: err = %v; want a refusal that names %s", access.name, err, tc.ledger)
					}
					if stderr.Len() != 0 {
						t.Errorf("%s: stderr = %q; want nothing with the refusal", access.name, stderr.String())
					}
					continue
				}
				if err != nil {
					t.Errorf("%s: err = %v; want only a note", access.name, err)
				}
				note := stderr.String()
				if !strings.HasPrefix(note, "note: "+retiredOrgStateDirEnv+" is set (") || strings.Count(note, "\n") != 1 {
					t.Errorf("%s: stderr = %q; want one note line", access.name, note)
				}
				// Only a read-only verb under the condition says the verbs
				// that change the ledger are refused.
				if says := strings.Contains(note, "are refused"); says != conditionHolds {
					t.Errorf("%s: note says the verbs are refused = %v; want %v (condition holds = %v): %q",
						access.name, says, conditionHolds, conditionHolds, note)
				}
			}

			if !strings.Contains(doctorWarn, retiredOrgStateDirRefusalCondition) {
				t.Errorf("the doctor's warn does not contain the condition %q:\n%s", retiredOrgStateDirRefusalCondition, doctorWarn)
			}
			if !strings.Contains(escalateHelp, retiredOrgStateDirRefusalCondition) {
				t.Errorf("ralph org escalate --help does not contain the condition %q:\n%s", retiredOrgStateDirRefusalCondition, escalateHelp)
			}
		})
	}
}
