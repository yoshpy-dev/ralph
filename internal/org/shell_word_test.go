package org

import "testing"

// ShellWord is the one function that writes a value as one shell word in a
// printed command, here and in internal/cli (plan
// docs/plans/active/2026-11-02-org-ledger-leftovers.md, S2: AC2, tech-debt
// 388). A value of characters no shell reads specially stays bare, and that
// includes the `+ : @ ,` of a branch name such as feat/a+b; anything else, or
// an empty string, is single-quoted, with a single quote written the POSIX way.
func TestShellWord_QuotesOnlyWhatAShellReads(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"a plus", "feat/a+b", "feat/a+b"},
		{"a colon", "a:b", "a:b"},
		{"an at sign", "a@b", "a@b"},
		{"a comma", "a,b", "a,b"},
		{"a plain path", "/repo/.claude/worktrees/org-auth", "/repo/.claude/worktrees/org-auth"},
		{"a space", "a b", `'a b'`},
		{"a single quote", "it's", `'it'\''s'`},
		{"a dollar sign", "$x", `'$x'`},
		{"a backtick", "a`b", "'a`b'"},
		{"an empty string", "", `''`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ShellWord(tc.in); got != tc.want {
				t.Errorf("ShellWord(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// The command a refusal of the cleanup or the merge names to run again writes
// the branch and the state dir the way ShellWord writes a value, whatever the
// value holds. The words below are the ones the table of internal/cli's
// TestShellWord_PrintedCommandsSpellAValueLikeTheOrgLayer pins too, so that
// the two layers spell a value the same way.
func TestShellWord_OrgRetryCommandWritesTheBranchAndTheStateDirWithIt(t *testing.T) {
	for _, tc := range []struct{ name, in, word string }{
		{"a plus", "feat/a+b", "feat/a+b"},
		{"a colon", "a:b", "a:b"},
		{"an at sign", "a@b", "a@b"},
		{"a comma", "a,b", "a,b"},
		{"a space", "a b", `'a b'`},
		{"a single quote", "it's", `'it'\''s'`},
	} {
		for _, verb := range []string{"cleanup", "merge"} {
			t.Run(verb+", "+tc.name, func(t *testing.T) {
				if got := ShellWord(tc.in); got != tc.word {
					t.Fatalf("ShellWord(%q) = %q, want %q", tc.in, got, tc.word)
				}
				// The branch and the dir are each ShellWord's word, which the
				// check above pinned to tc.word.
				want := "ralph org " + verb + " --branch " + tc.word + " --state-dir " + tc.word
				if got := orgRetryCommand(verb, tc.in, tc.in); got != want {
					t.Errorf("orgRetryCommand(%q, %q, %q) = %q, want %q", verb, tc.in, tc.in, got, want)
				}
			})
		}
	}
}
