package org

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The commands the refusals and failures of ralph org cleanup and ralph org
// merge name to run again keep the --state-dir the command was given (plan
// docs/plans/active/2026-11-01-org-ledger-gaps.md, S7: AC1, tech-debt 395):
// CleanupParams.HintStateDir and MergeParams.HintStateDir reach every such
// command through orgRetryCommand, and Merge passes its own on to the cleanup
// it runs. The step that moves the ledger out of the worktree
// (ledgerMoveStep) is the exception: it names the new place, never the old
// one.

// cleanupRetryHint is the --state-dir the tests give, with a space, so that
// it must be quoted as one shell word.
const cleanupRetryHint = "/my ledgers/org"

// orgRetryCommand over each verb, a branch with and without characters a
// shell reads, and a hint that is empty, plain, or holds a space or a single
// quote: the words are cleanupShellWord's, and an empty hint adds nothing.
func TestCleanupRetryCommand_QuotesEachWordAndAddsTheStateDirOnlyWhenGiven(t *testing.T) {
	for _, tc := range []struct{ verb, branch, hint, want string }{
		{"cleanup", "feat/a", "", "ralph org cleanup --branch feat/a"},
		{"cleanup", "feat/a", "/state/org", "ralph org cleanup --branch feat/a --state-dir /state/org"},
		{"cleanup", "feat/a;b", cleanupRetryHint, `ralph org cleanup --branch 'feat/a;b' --state-dir '/my ledgers/org'`},
		{"cleanup", "feat/a", "/o'brien/org", `ralph org cleanup --branch feat/a --state-dir '/o'\''brien/org'`},
		{"merge", "feat/a", "", "ralph org merge --branch feat/a"},
		{"merge", "feat/$(x)", "", `ralph org merge --branch 'feat/$(x)'`},
		{"merge", "feat/a", cleanupRetryHint, `ralph org merge --branch feat/a --state-dir '/my ledgers/org'`},
	} {
		if got := orgRetryCommand(tc.verb, tc.branch, tc.hint); got != tc.want {
			t.Errorf("orgRetryCommand(%q, %q, %q) = %q, want %q", tc.verb, tc.branch, tc.hint, got, tc.want)
		}
	}
}

// cleanupRetryAgain is the step a refusal of feat/auth's cleanup names, with
// the hint when it is set.
func cleanupRetryAgain(hint string) string {
	if hint == "" {
		return "run ralph org cleanup --branch feat/auth again"
	}
	return "run ralph org cleanup --branch feat/auth --state-dir '" + hint + "' again"
}

// assertCleanupRetry checks that s names the cleanup of feat/auth with the
// hint, and names --state-dir only there.
func assertCleanupRetry(t *testing.T, s, hint string) {
	t.Helper()
	want := cleanupRetryAgain(hint)
	if !strings.Contains(s, want) {
		t.Errorf("%q does not say %q", s, want)
	}
	wantN := 0
	if hint != "" {
		wantN = 1
	}
	if n := strings.Count(s, "--state-dir"); n != wantN {
		t.Errorf("%q names --state-dir %d time(s), want %d", s, n, wantN)
	}
}

// The refusals before any git (the head, regate.lock, the ledger's check,
// the record) name the cleanup of the branch with HintStateDir when it is
// set, and with no --state-dir when it is not.
func TestCleanupFeatureWorktree_RefusalsBeforeGitRepeatTheStateDir(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, fx *worktreeCleanupFakeFixture, p *CleanupParams)
	}{
		{"a head that is no commit SHA", func(_ *testing.T, _ *worktreeCleanupFakeFixture, p *CleanupParams) { p.Head = "0123abc" }},
		{"regate.lock held", func(t *testing.T, fx *worktreeCleanupFakeFixture, _ *CleanupParams) {
			other, err := acquireHeldFileLock(fx.stateDir, regateLockFile, "regate", 0)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = other.release() })
		}},
		{"the org runs", func(t *testing.T, fx *worktreeCleanupFakeFixture, _ *CleanupParams) {
			seedSeat(t, fx.o, "auth", "worker", "worker", "pane-2")
		}},
		{"the org runs another feature", func(t *testing.T, fx *worktreeCleanupFakeFixture, _ *CleanupParams) {
			g := *fx.f
			g.Feature, g.Branch = "auth-2", "feat/auth-2"
			appendCleanupEvents(t, fx.o, boundReserveEvent("auth", &g, "internal/auth/"))
		}},
		{"a record that cannot be read", func(_ *testing.T, fx *worktreeCleanupFakeFixture, _ *CleanupParams) {
			fx.wt.lookupErr = errors.New("org: /x/org-auth.json is not a JSON record")
		}},
		{"no script to run", func(_ *testing.T, fx *worktreeCleanupFakeFixture, _ *CleanupParams) { fx.o.Worktrees = nil }},
	} {
		for _, run := range []struct{ name, hint string }{{"no --state-dir", ""}, {"--state-dir", cleanupRetryHint}} {
			t.Run(tc.name+", "+run.name, func(t *testing.T) {
				fx := newWorktreeCleanupFakeFixture(t)
				p := fx.params()
				p.HintStateDir = run.hint
				tc.setup(t, fx, &p)
				res, err := fx.o.CleanupFeatureWorktree(context.Background(), p)
				if err != nil || res.Refused == "" || res.Cleaned {
					t.Fatalf("cleanup = %+v, %v; want a refusal", res, err)
				}
				assertCleanupRetry(t, res.Refused, run.hint)
			})
		}
	}
}

// The refusals of the checks of a worktree that is there, and the failure of
// the script, name the cleanup of the branch with HintStateDir: the refusal of
// a caller in the worktree says to run that command from outside it, and the
// failure says that command goes on from what the script left.
func TestCleanupFeatureWorktree_WorktreeRefusalsRepeatTheStateDir(t *testing.T) {
	const hint = cleanupRetryHint
	quoted := "'" + hint + "'"
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, fx *worktreeCleanupFixture, p *CleanupParams)
		want  string // what the refusal or the failure says, %HINT% the quoted hint
	}{
		{"run in the worktree", func(_ *testing.T, fx *worktreeCleanupFixture, p *CleanupParams) { p.Cwd = fx.worktree },
			"run ralph org cleanup --branch feat/auth --state-dir %HINT% from outside it"},
		{"an untracked file", func(t *testing.T, fx *worktreeCleanupFixture, _ *CleanupParams) {
			writeGateFile(t, fx.worktree, "stray.txt", "x\n", 0o644)
		}, "remove the rest, and run ralph org cleanup --branch feat/auth --state-dir %HINT% again"},
		{"a detached HEAD", func(t *testing.T, fx *worktreeCleanupFixture, _ *CleanupParams) {
			runGit(t, "-C", fx.worktree, "checkout", "--quiet", "--detach")
		}, "and run ralph org cleanup --branch feat/auth --state-dir %HINT% again"},
		{"the branch moved past the merged head", func(t *testing.T, fx *worktreeCleanupFixture, _ *CleanupParams) {
			gateCommit(t, fx.worktree, "not merged")
		}, "reset --hard %HEAD%), and run ralph org cleanup --branch feat/auth --state-dir %HINT% again"},
		{"the script fails", func(_ *testing.T, fx *worktreeCleanupFixture, _ *CleanupParams) {
			fake := newFakeWorktrees()
			fake.records[cleanupTestID] = WorktreeRecord{WorktreePath: fx.worktree, Branch: cleanupTestBranch, Kind: featureWorktreeKind}
			fake.cleanupErr = errors.New("ralph-worktree: refusing cleanup")
			fx.o.Worktrees = fake
		}, "once its cause is fixed, ralph org cleanup --branch feat/auth --state-dir %HINT% goes on from what is left"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newWorktreeCleanupFixture(t)
			p := fx.params()
			p.HintStateDir = hint
			tc.setup(t, fx, &p)
			res, err := fx.cleanup(p)
			if err != nil || res.Cleaned || res.Refused+res.Failed == "" {
				t.Fatalf("cleanup = %+v, %v; want a refusal or a failure", res, err)
			}
			want := strings.NewReplacer("%HINT%", quoted, "%HEAD%", fx.head).Replace(tc.want)
			if got := res.Refused + res.Failed; !strings.Contains(got, want) || strings.Count(got, "--state-dir") != 1 {
				t.Errorf("cleanup says %q; want it to say %q, with --state-dir there only", got, want)
			}
		})
	}
}

// ledgerMoveStep names the two ways to point at the new place of the ledger,
// --state-dir <new place> and [org].state_dir, and never the --state-dir the
// command was given, which names the old place: with HintStateDir set to the
// ledger (as ralph org cleanup --state-dir <ledger> sets it) and without it,
// the refusal of a ledger inside the worktree is the same text, whose only
// --state-dir is the one of <new place>. The repo's path holds a space, so a
// --state-dir of the old place would be quoted.
func TestCleanupFeatureWorktree_LedgerMoveStepNamesNoOldStateDir(t *testing.T) {
	for _, withHint := range []bool{false, true} {
		fx := newWorktreeCleanupFakeFixtureAt(t, filepath.Join(resolvedOrClean(t.TempDir()), "my repo"))
		ledger := filepath.Join(fx.f.Worktree, ".harness", "state", "org")
		if err := os.MkdirAll(filepath.Dir(ledger), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(fx.stateDir, ledger); err != nil {
			t.Fatal(err)
		}
		fx.stateDir = ledger
		fx.o.Manifest = NewManifestStoreAtPath(ManifestPathIn(ledger))
		p := fx.params()
		if withHint {
			p.HintStateDir = ledger
		}
		res, err := fx.o.CleanupFeatureWorktree(context.Background(), p)
		want := "the ledger's directory " + ledger + " is inside the feature worktree " + fx.f.Worktree + ", which the cleanup " +
			"removes with the files .gitignore names, so the ledger would be removed with it, and nothing is removed: " +
			cleanupLedgerMoveStep
		if err != nil || res.Refused != want || res.Manual != "" {
			t.Fatalf("cleanup with HintStateDir %q = %+v, %v\nwant the refusal %q", p.HintStateDir, res, err, want)
		}
		if strings.Contains(res.Refused, cleanupShellWord(ledger)) || strings.Count(res.Refused, "--state-dir") != 1 {
			t.Errorf("refusal %q names a --state-dir other than <new place>", res.Refused)
		}
	}
}

// The refusals and failures of ralph org merge name the merge of the branch,
// as one shell word, with HintStateDir: a refusal of checkMerge (gh not on
// PATH, before the ledger is read, for a branch that needs quoting),
// regate.lock held, and an auto-merge request removed after gh pr merge.
func TestMerge_RetryCommandsRepeatTheStateDir(t *testing.T) {
	const quoted = "'" + cleanupRetryHint + "'"
	t.Run("a refusal of the checks", func(t *testing.T) {
		for _, hint := range []string{"", cleanupRetryHint} {
			fx := newMXFixture(t, nil)
			p := fx.params()
			p.Branch, p.LookPath, p.HintStateDir = "feat/a;b", mgNotFound, hint
			_, refused, err := fx.check(context.Background(), t, p)
			want := "and run ralph org merge --branch 'feat/a;b' again"
			if hint != "" {
				want = "and run ralph org merge --branch 'feat/a;b' --state-dir " + quoted + " again"
			}
			if err != nil || !strings.HasPrefix(refused, "org: merge: gh is not on PATH") || !strings.HasSuffix(refused, want) {
				t.Errorf("checkMerge with HintStateDir %q = %q, %v; want the refusal of gh ending with %q", hint, refused, err, want)
			}
		}
	})
	t.Run("regate.lock held", func(t *testing.T) {
		fx := newMRFixture(t, nil)
		other, err := acquireHeldFileLock(fx.mo.stateDir, regateLockFile, "regate", 0)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = other.release() }()
		p := fx.params()
		p.HintStateDir = cleanupRetryHint
		res, err := fx.merge(context.Background(), t, p)
		want := "org: merge: another ralph org regate, merge, or cleanup runs on this ledger (" + regateLockPathIn(fx.mo.stateDir) +
			" is held): one runs at a time, so wait for it to end and run ralph org merge --branch feat/a --state-dir " + quoted +
			" again"
		if err != nil || res.Refused != want {
			t.Errorf("Merge = %+v, %v\nwant the refusal %q", res, err, want)
		}
	})
	t.Run("an auto-merge request removed", func(t *testing.T) {
		fx := newMRFixture(t, nil)
		fx.ex.reread = mgReply{out: mrReadBack("OPEN", "", mrRequest)}
		p := fx.params()
		p.HintStateDir = cleanupRetryHint
		res, err := fx.merge(context.Background(), t, p)
		want := "so nothing was merged or recorded: run ralph org merge --branch feat/a --state-dir " + quoted + " again, which " +
			"checks the PR and the protection again"
		if err != nil || !strings.HasSuffix(res.Failed, want) {
			t.Errorf("Merge = %+v, %v\nwant Failed ending with %q", res, err, want)
		}
	})
}

// Merge passes its HintStateDir on to the cleanup it runs once pr_merged is
// recorded: a refusal of that cleanup (an untracked file in the worktree),
// and the refusal Merge gives the cleanup it skips once interrupted, name the
// cleanup of the branch with it.
func TestMerge_CleanupRepeatsTheStateDir(t *testing.T) {
	const quoted = "'" + cleanupRetryHint + "'"
	t.Run("a refusal of the cleanup", func(t *testing.T) {
		fx := newMCFixture(t)
		writeGateFile(t, fx.worktree, "stray.txt", "x\n", 0o644)
		p := fx.mergeParams()
		p.HintStateDir = cleanupRetryHint
		res, err := fx.o.Merge(context.Background(), p)
		if err != nil || !res.Recorded || res.Cleanup == nil || res.Cleanup.Cleaned {
			t.Fatalf("Merge = %+v, %v (cleanup %+v); want a recorded merge whose cleanup refused", res, err, res.Cleanup)
		}
		assertCleanupRetry(t, res.Cleanup.Refused, cleanupRetryHint)
	})
	t.Run("the cleanup skipped once interrupted", func(t *testing.T) {
		fx, _ := mcFakeFixture(t)
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		fx.ex.inMerge = func(context.Context) { cancel(errors.New("test stopped")) }
		p := fx.params()
		p.RepoRoot, p.HintStateDir = "/repo", cleanupRetryHint
		res, err := fx.merge(ctx, t, p)
		want := ": run ralph org cleanup --branch feat/a --state-dir " + quoted
		if err == nil || res.Cleanup == nil || !strings.HasSuffix(res.Cleanup.Refused, want) {
			t.Errorf("Merge = %+v, %v (cleanup %+v); want the skipped cleanup's refusal ending with %q", res, err, res.Cleanup, want)
		}
	})
}
