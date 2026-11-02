package org

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// The commands of the verbs other than merge and cleanup that ralph org merge
// and ralph org cleanup print (ralph org merge-order, with --all or not,
// ralph org inbox, ralph org status --org-id, and ralph org disband --org-id)
// carry --state-dir only when the command was given it (plan
// docs/plans/active/2026-11-02-org-ledger-leftovers.md, S3: AC3, tech-debt
// 395): MergeParams.HintStateDir and CleanupParams.HintStateDir reach each
// through VerbCommand, after the verb's own arguments. The hint of these tests
// is cleanupRetryHint, whose space makes ShellWord quote it.

// hintFlag is what a command of the tests' hint carries after its own
// arguments: --state-dir and the hint as ShellWord writes it, or nothing
// without a hint.
func hintFlag(hint string) string {
	if hint == "" {
		return ""
	}
	return " --state-dir '/my ledgers/org'"
}

// assertHintStateDir checks that got says each of wants, with every %S% in
// it replaced by hintFlag(hint), and names --state-dir nowhere else: as many
// times as the wants hold %S% with a hint, and never without one.
func assertHintStateDir(t *testing.T, got, hint string, wants ...string) {
	t.Helper()
	n := 0
	for _, w := range wants {
		if want := strings.ReplaceAll(w, "%S%", hintFlag(hint)); !strings.Contains(got, want) {
			t.Errorf("with the hint %q:\n%q\ndoes not say\n%q", hint, got, want)
		}
		n += strings.Count(w, "%S%")
	}
	if hint == "" {
		n = 0
	}
	if c := strings.Count(got, "--state-dir"); c != n {
		t.Errorf("with the hint %q, %q names --state-dir %d time(s), want %d", hint, got, c, n)
	}
}

// hintStateDirRuns are the two runs of each case: without a hint, as the CLI
// leaves it without --state-dir (also when [org].state_dir names the
// ledger), and with one.
var hintStateDirRuns = []string{"", cleanupRetryHint}

// hintRunName names the run of hint in a subtest.
func hintRunName(hint string) string {
	if hint == "" {
		return "without --state-dir"
	}
	return "with --state-dir"
}

// VerbCommand writes the verb and its arguments as they are and adds
// --state-dir and the hint, as one shell word, only when the hint is set.
func TestVerbCommand_AddsTheHintStateDirOnlyWhenGiven(t *testing.T) {
	for _, tc := range []struct{ verbAndArgs, hint, want string }{
		{"merge-order", "", "ralph org merge-order"},
		{"merge-order --all", "", "ralph org merge-order --all"},
		{"merge-order", "/state/org", "ralph org merge-order --state-dir /state/org"},
		{"merge-order --all", cleanupRetryHint, "ralph org merge-order --all --state-dir '/my ledgers/org'"},
		{"inbox", "/o'brien/org", `ralph org inbox --state-dir '/o'\''brien/org'`},
		{"status --org-id org-a", cleanupRetryHint, "ralph org status --org-id org-a --state-dir '/my ledgers/org'"},
		{"disband --org-id org-a", "", "ralph org disband --org-id org-a"},
	} {
		if got := VerbCommand(tc.verbAndArgs, tc.hint); got != tc.want {
			t.Errorf("VerbCommand(%q, %q) = %q, want %q", tc.verbAndArgs, tc.hint, got, tc.want)
		}
	}
}

// The cleanup's refusal of no branch named names ralph org merge-order --all
// with the hint.
func TestCleanupFeatureWorktree_HintStateDirReachesMergeOrderWithNoBranch(t *testing.T) {
	for _, hint := range hintStateDirRuns {
		p := CleanupParams{RepoRoot: "/repo", Via: CleanupViaCleanup, HintStateDir: hint}
		res, err := (&Org{}).CleanupFeatureWorktree(context.Background(), p)
		if err != nil || res.Cleaned {
			t.Fatalf("cleanup with no branch = %+v, %v; want a refusal", res, err)
		}
		assertHintStateDir(t, res.Refused, hint, "no branch named: name the merged feature branch to clean up "+
			"(ralph org merge-order --all%S% lists the merged ones)")
	}
}

// The refusals of checkMerge that name another verb: no branch, a branch the
// ledger does not name, a branch merge-order does not list as ready, an org
// that still runs (ralph org inbox, with the merge to run again), a PR that is
// not OPEN, a PR whose head is not the gate's, and origin that moved.
func TestCheckMerge_HintStateDirReachesTheOtherVerbs(t *testing.T) {
	view := func(state, head string) func(fx *mxFixture, _ *MergeParams) {
		return func(fx *mxFixture, _ *MergeParams) {
			fx.exec.replies[mxPRView] = mgReply{out: mxPRViewJSON(state, head, "main", "https://github.com/o/r/pull/7", "null")}
		}
	}
	for _, tc := range []struct {
		name   string
		ledger []any // nil: mxReadyLedger
		setup  func(fx *mxFixture, p *MergeParams)
		wants  []string
	}{
		{"no branch named", nil, func(_ *mxFixture, p *MergeParams) { p.Branch = "" },
			[]string{"name the feature branch to merge (ralph org merge-order%S% lists the ledger's branches)"}},
		{"a branch the ledger does not name", nil, func(_ *mxFixture, p *MergeParams) { p.Branch = "feat/zzz" },
			[]string{"ralph org merge merges only a branch ralph org merge-order%S% lists as ready, so run it to see the ledger's " +
				"branches"}},
		{"not ready", nil, func(fx *mxFixture, _ *MergeParams) {
			fx.exec.replies[mxPRList] = mgReply{out: moPRs(moPR(7, "MERGED", "main"))}
		}, []string{"org: merge: ralph org merge-order%S% lists the branch feat/a as merged, not ready"}},
		{"the org still runs", []any{moBind("org-a", "a"), moPassed("org-a", "feat/a", moHeadA, moMain)}, func(*mxFixture, *MergeParams) {},
			[]string{"handle the org's RESULT in the inbox (ralph org inbox%S%), and once the org is disbanded, run ralph org merge " +
				"--branch feat/a%S% again"}},
		{"the PR is not OPEN", nil, view("MERGED", moHeadA),
			[]string{"so nothing was merged: run ralph org merge-order%S% to see the branch's state"}},
		{"the PR's head is not the gate's", nil, view("OPEN", moHeadB),
			[]string{"so nothing was merged: run ralph org merge-order%S% again and follow the branch's Detail"}},
		{"origin moved", nil, func(fx *mxFixture, _ *MergeParams) {
			fx.exec.ls = []mgReply{{out: mxListing()}, {out: moListing("main", "main", moMain2, "feat/a", moHeadA)}}
		}, []string{"so nothing was merged: run ralph org merge-order%S% again and follow the branch's Detail"}},
	} {
		for _, hint := range hintStateDirRuns {
			t.Run(tc.name+", "+hintRunName(hint), func(t *testing.T) {
				fx := newMXFixture(t, tc.ledger)
				p := fx.params()
				p.HintStateDir = hint
				tc.setup(fx, &p)
				_, refused, _, err := fx.checkKind(context.Background(), t, p)
				if err != nil || refused == "" {
					t.Fatalf("checkMerge = %q, %v; want a refusal", refused, err)
				}
				assertHintStateDir(t, refused, hint, tc.wants...)
			})
		}
	}
}

// CheckMergeEnforcement's Detail of a default branch gh api names otherwise
// than origin's HEAD names ralph org merge-order with the hint
// checkMerge passes it (MergeEnforcementParams.HintStateDir).
func TestCheckMergeEnforcement_HintStateDirReachesMergeOrder(t *testing.T) {
	for _, hint := range hintStateDirRuns {
		fx := newMXFixture(t, nil)
		fx.exec.replies[mxRepoAPI] = mgReply{out: meRepoJSON("trunk", "o/r", nil, false)}
		p := fx.params()
		p.HintStateDir = hint
		_, refused, err := fx.check(context.Background(), t, p)
		if err != nil || !strings.HasPrefix(refused, mxProtect) {
			t.Fatalf("checkMerge = %q, %v; want the refusal of the protection", refused, err)
		}
		assertHintStateDir(t, refused, hint, `returned the default branch "trunk", but origin's HEAD names main, so which `+
			"branch's protection to read is unknown: run ralph org merge-order%S% again")
	}
}

// The failures of Merge once gh pr merge ran that name ralph org merge-order:
// a read-back that fails, a PR still open after gh pr merge failed or exited
// 0, and a PR merged at another head; and its error when pr_merged cannot be
// recorded.
func TestMerge_HintStateDirReachesMergeOrderOnceGHPRMergeRan(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, fx *mrFixture)
		want  string
	}{
		{"the read-back fails", func(_ *testing.T, fx *mrFixture) {
			fx.ex.reread = mgReply{err: errors.New("exit status 4: HTTP 401")}
		}, "--json state,headRefOid,baseRefName,mergeCommit (ralph org merge-order%S% lists the branch as merged once it is)"},
		{"gh pr merge failed and the PR is open", func(_ *testing.T, fx *mrFixture) {
			fx.ex.merge = mgReply{err: errors.New("exit status 1")}
			fx.ex.reread = mgReply{out: mrReadBack("OPEN", "", "null")}
		}, "so nothing was merged or recorded: run ralph org merge-order%S% again and follow the branch's Detail"},
		{"gh pr merge exited 0 and the PR is open", func(_ *testing.T, fx *mrFixture) {
			fx.ex.reread = mgReply{out: mrReadBack("OPEN", "", "null")}
		}, "check the PR with " + mrViewCmd + " and run ralph org merge-order%S% again"},
		{"another head was merged", func(_ *testing.T, fx *mrFixture) {
			fx.ex.reread = mgReply{out: mrReadBackAt("MERGED", moHeadB, mrOID, "null")}
		}, "run ralph org merge-order%S% again, and go on with the other branches"},
		{"pr_merged cannot be recorded", func(t *testing.T, fx *mrFixture) {
			if os.Geteuid() == 0 {
				t.Skip("root can append to a read-only manifest")
			}
			if err := os.Chmod(fx.mo.path, 0o444); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(fx.mo.path, 0o644) })
		}, "the merge stands, and ralph org merge-order%S% lists the branch as merged, so only the ledger misses it"},
	} {
		for _, hint := range hintStateDirRuns {
			t.Run(tc.name+", "+hintRunName(hint), func(t *testing.T) {
				fx := newMRFixture(t, nil)
				tc.setup(t, fx)
				p := fx.params()
				p.HintStateDir = hint
				res, err := fx.merge(context.Background(), t, p)
				got := res.Failed
				if err != nil {
					got = err.Error()
				}
				if got == "" || res.Recorded {
					t.Fatalf("Merge = %+v, %v; want a failure or an error", res, err)
				}
				assertHintStateDir(t, got, hint, tc.want)
			})
		}
	}
}

// Once --wait-checks is used up with the branch's org still running, the
// refusal names ralph org inbox, the merge to run again, and ralph org status
// and ralph org disband of the org, each with the hint after the org_id.
func TestMerge_HintStateDirReachesStatusAndDisbandOnceTheWaitIsUsedUp(t *testing.T) {
	for _, hint := range hintStateDirRuns {
		t.Run(hintRunName(hint), func(t *testing.T) {
			fx := newMWFixtureOn(t, mwOrgRunning(), mwPass, 70*time.Second)
			fx.p.HintStateDir = hint
			res, err := fx.run(t)
			if err != nil || res.Refused == "" {
				t.Fatalf("Merge = %+v, %v; want the refusal once the wait is used up", res, err)
			}
			assertHintStateDir(t, res.Refused, hint,
				"in the inbox (ralph org inbox%S%), and once the org is disbanded, run ralph org merge --branch feat/a%S% again",
				"see what it does with ralph org status --org-id org-a%S%, and a human decides whether to disband it (ralph org "+
					"disband --org-id org-a%S%)")
		})
	}
}
