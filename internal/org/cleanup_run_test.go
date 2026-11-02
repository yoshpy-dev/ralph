package org

import (
	"os"
	"slices"
	"strings"
	"testing"
)

// Two branches of the cleanup (cleanup.go) that the tests of cleanup_test.go
// do not run, 420(d) of docs/tech-debt/README.md (plan
// 2026-11-02-org-ledger-leftovers, AC7): the refusal of cleanupRun.run when
// something that is not a directory is at the feature worktree's path
// (st.notDir), and the failure of git worktree remove in cleanupRun.remove
// for a registered worktree whose directory is gone. Both use
// worktreeCleanupFixture, which runs the real scripts/ralph-worktree.sh and
// git.

// TestCleanupFeatureWorktree_NotADirectoryAtTheWorktreesPathRemovesNothing:
// with a regular file where the feature worktree was, whether git still
// registers the worktree or not, the cleanup refuses with the reason that
// the path is not a directory and the step to remove it by hand and run the
// cleanup again, with no Manual. The file, the registration, the branch, and
// the record stay, the script's cleanup does not run, and nothing is
// recorded. Once the file is removed, the next cleanup goes on from what is
// left.
func TestCleanupFeatureWorktree_NotADirectoryAtTheWorktreesPathRemovesNothing(t *testing.T) {
	for _, tc := range []struct {
		name string
		gone func(t *testing.T, fx *worktreeCleanupFixture) // takes the worktree's directory away
	}{
		{"git still registers the worktree", func(t *testing.T, fx *worktreeCleanupFixture) {
			if err := os.RemoveAll(fx.worktree); err != nil {
				t.Fatal(err)
			}
		}},
		{"its registration removed too", func(t *testing.T, fx *worktreeCleanupFixture) {
			runGit(t, "-C", fx.root, "worktree", "remove", fx.worktree)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newWorktreeCleanupFixture(t)
			tc.gone(t, fx)
			const content = "not a worktree\n"
			if err := os.WriteFile(fx.worktree, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			before := fx.snapshot(t)

			res, err := fx.cleanup(fx.params())
			want := fx.worktree + ", the path of the feature worktree, is not a directory: check what it is, remove it by hand, " +
				"and run ralph org cleanup --branch feat/auth again"
			fx.assertRefused(t, res, err, before, want, "")
			if res.Refused != want || res.Resumed != "" || res.NothingLeft {
				t.Errorf("cleanup = %+v; want exactly the refusal %q", res, want)
			}
			if got, err := os.ReadFile(fx.worktree); err != nil || string(got) != content {
				t.Errorf("the file at %s holds %q (%v) after the refusal; want %q", fx.worktree, got, err, content)
			}
			if events := cleanedEvents(mustReadEvents(t, fx.o)); len(events) != 0 {
				t.Errorf("the refusal recorded %+v", events)
			}

			if err := os.Remove(fx.worktree); err != nil {
				t.Fatal(err)
			}
			res, err = fx.cleanup(fx.params())
			fx.assertCleaned(t, res, err, CleanupResumedWorktreeGone, CleanupViaCleanup)
		})
	}
}

// TestCleanupFeatureWorktree_FailedRemovalOfAMissingWorktreesRegistrationRemovesNothing:
// the feature worktree is locked (git worktree lock) and its directory is
// gone, so every check passes and git worktree remove of the registration,
// which the cleanup runs before the script, fails. The result is Failed, with
// git's exit and stderr and the step to fix that and run the cleanup again,
// and Manual is git worktree remove of the registration, then the script's
// cleanup. Nothing is removed (the registration, the branch, and the record
// stay), the script's cleanup does not run, and no worktree_cleaned is
// recorded. Once the worktree is unlocked, the next cleanup goes on from what
// is left.
func TestCleanupFeatureWorktree_FailedRemovalOfAMissingWorktreesRegistrationRemovesNothing(t *testing.T) {
	fx := newWorktreeCleanupFixture(t)
	runGit(t, "-C", fx.root, "worktree", "lock", "--reason", "on a removable disk", fx.worktree)
	if err := os.RemoveAll(fx.worktree); err != nil {
		t.Fatal(err)
	}
	before := fx.snapshot(t)

	res, err := fx.cleanup(fx.params())
	if err != nil || res.Failed == "" || res.Refused != "" || res.Cleaned || res.Recorded || res.Resumed != "" {
		t.Fatalf("cleanup = %+v, %v; want a failure of git worktree remove", res, err)
	}
	start := "git worktree remove " + fx.worktree + " in " + fx.root + ", which takes git's registration of the missing feature " +
		"worktree away, failed (exit 128: "
	end := "), so nothing was removed: fix what it says and run ralph org cleanup --branch feat/auth again"
	if !strings.HasPrefix(res.Failed, start) || !strings.HasSuffix(res.Failed, end) ||
		!strings.Contains(res.Failed, "cannot remove a locked working tree") || strings.Contains(res.Failed, "\n") {
		t.Errorf("Failed = %q; want one line from %q to %q with git's reason, the lock", res.Failed, start, end)
	}
	// The paths are written with ShellWord, as the code writes them, so the
	// expectation holds in a temp dir whose path has a space.
	root := ShellWord(fx.root)
	manual := "git -C " + root + " worktree remove " + ShellWord(fx.worktree) + " && cd " + root +
		" && ./scripts/ralph-worktree.sh cleanup --id org-auth --force-branch"
	if res.Manual != manual {
		t.Errorf("Manual = %q, want %q", res.Manual, manual)
	}
	if after := fx.snapshot(t); after != before {
		t.Errorf("a failed removal changed something:\n%s\n->\n%s", before, after)
	}
	if slices.Contains(fx.spy.calls, "cleanup:"+cleanupTestID) {
		t.Errorf("a failed removal ran the script's cleanup: calls %q", fx.spy.calls)
	}
	if events := cleanedEvents(mustReadEvents(t, fx.o)); len(events) != 0 {
		t.Errorf("a failed removal recorded %+v", events)
	}
	assertCleanupUnlocked(t, fx.stateDir)

	runGit(t, "-C", fx.root, "worktree", "unlock", fx.worktree)
	res, err = fx.cleanup(fx.params())
	fx.assertCleaned(t, res, err, CleanupResumedWorktreeGone, CleanupViaCleanup)
}
