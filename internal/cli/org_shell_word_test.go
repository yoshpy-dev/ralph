package cli

import (
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// The commands the CLI prints (cleanupRetryCommand for ralph org cleanup and
// ralph org merge, orgReadCommandHint for ralph org read) write a value as one
// shell word with org.ShellWord, the function internal/org writes the commands
// of its refusals with (orgRetryCommand), so a value is spelled the same way in
// both layers (plan docs/plans/active/2026-11-02-org-ledger-leftovers.md, S2:
// AC2, tech-debt 388). The words below are the ones the table of internal/org's
// TestShellWord_OrgRetryCommandWritesTheBranchAndTheStateDirWithIt pins too: a
// value with a +, :, @, or , stays bare, and a space or a single quote is
// quoted.
func TestShellWord_PrintedCommandsSpellAValueLikeTheOrgLayer(t *testing.T) {
	for _, tc := range []struct{ name, in, word string }{
		{"a plus", "feat/a+b", "feat/a+b"},
		{"a colon", "a:b", "a:b"},
		{"an at sign", "a@b", "a@b"},
		{"a comma", "a,b", "a,b"},
		{"a space", "a b", `'a b'`},
		{"a single quote", "it's", `'it'\''s'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := org.ShellWord(tc.in); got != tc.word {
				t.Fatalf("org.ShellWord(%q) = %q, want %q", tc.in, got, tc.word)
			}
			wantCleanup := "ralph org cleanup --branch " + tc.word + " --state-dir " + tc.word
			if got := cleanupRetryCommand(tc.in, tc.in); got != wantCleanup {
				t.Errorf("cleanupRetryCommand(%q, %q) = %q, want %q", tc.in, tc.in, got, wantCleanup)
			}
			wantRead := "ralph org read --org-id org-a --seat seat-1 --state-dir " + tc.word
			if got := orgReadCommandHint("org-a", "seat-1", tc.in, true); got != wantRead {
				t.Errorf("orgReadCommandHint(%q, true) = %q, want %q", tc.in, got, wantRead)
			}
			if got := orgReadCommandHint("org-a", "seat-1", tc.in, false); got != "ralph org read --org-id org-a --seat seat-1" {
				t.Errorf("orgReadCommandHint(%q, false) = %q, want no --state-dir", tc.in, got)
			}
		})
	}
}
