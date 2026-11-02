package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for ralph org cleanup's steps, printer, interruption, and help; the fixture and the other tests are in org_cleanup_test.go.

// The command's context reaches the reads before the removal: with it done
// before the command runs, cleanup exits 1 with the interruption of the first
// of them, the read of origin (S-X1 of the plan), nothing on stdout, and
// nothing removed or recorded. merge-order's own interruption is
// TestMergeOrder_InterruptedWhileGHRunsIsAnError's.
func TestOrgCleanup_InterruptedBeforeItRunsRemovesNothing(t *testing.T) {
	fx := newCleanupCLIFixture(t)
	fx.squashMergeOnGitHub(t)
	before := fx.snapshot(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stdout, _, err := runOrgCmdStreamsContext(ctx, "cleanup", "--state-dir", fx.stateDir)
	const want = "org: cleanup: interrupted (context canceled) while git remote get-url origin ran: nothing was removed"
	if err == nil || err.Error() != want || stdout != "" {
		t.Errorf("cleanup with its context done: err %v, stdout %q; want the error %q and nothing on stdout", err, stdout, want)
	}
	if got := fx.ghCalls(t); len(got) != 0 {
		t.Errorf("gh ran: %q", got)
	}
	if after := fx.snapshot(t); after != before {
		t.Errorf("an interrupted cleanup changed something:\n%s\n->\n%s", before, after)
	}
}

// cleanupInterruptGit is the git of the bin dir for
// TestOrgCleanup_InterruptedDuringTheLastCleanupExitsOne (fmt verb: the git
// it wraps, shell-quoted): for the script's git branch -D, it creates
// $CLEANUP_TEST_AT and waits up to 20 s for $CLEANUP_TEST_GO, then runs the
// git it wraps, as every other call does.
const cleanupInterruptGit = `#!/bin/sh
case " $* " in
*" branch -D "*)
  : > "$CLEANUP_TEST_AT"
  i=0
  while [ ! -f "$CLEANUP_TEST_GO" ] && [ "$i" -lt 2000 ]; do sleep 0.01; i=$((i + 1)); done ;;
esac
exec %s "$@"
`

// Finding 2 (cross-review cycle 2,
// docs/reports/cross-review-triage-org-merge-cleanup.md; S-X3 of the plan):
// an interrupt that comes while the only branch's cleanup runs the script
// (at its git branch -D, the worktree removed already) does not stop that
// cleanup, which removes the branch, records worktree_cleaned, and prints its
// CLEANED: line, but cleanup still exits 1 with the interruption, as its help
// says, not 0, with and without --branch. The bin dir's git waits at git
// branch -D until the test has canceled the command's context.
func TestOrgCleanup_InterruptedDuringTheLastCleanupExitsOne(t *testing.T) {
	for _, tc := range []struct {
		name  string
		extra []string
	}{
		{"every merged branch", nil},
		{"--branch", []string{"--branch", "feat/a"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newCleanupCLIFixture(t)
			fx.squashMergeOnGitHub(t)
			wrapped := filepath.Join(fx.bin, "git-wrapped")
			if err := os.Rename(filepath.Join(fx.bin, "git"), wrapped); err != nil {
				t.Fatal(err)
			}
			writeStubScript(t, filepath.Join(fx.bin, "git"), fmt.Sprintf(cleanupInterruptGit, shellQuoteIfNeeded(wrapped)))
			signals := t.TempDir()
			at, proceed := filepath.Join(signals, "at"), filepath.Join(signals, "go")
			t.Setenv("CLEANUP_TEST_AT", at)
			t.Setenv("CLEANUP_TEST_GO", proceed)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			stop, fired := make(chan struct{}), make(chan bool, 1)
			go func() {
				for {
					if _, err := os.Stat(at); err == nil {
						cancel(errors.New("test stopped"))
						fired <- os.WriteFile(proceed, nil, 0o644) == nil
						return
					}
					select {
					case <-stop:
						fired <- false
						return
					case <-time.After(10 * time.Millisecond):
					}
				}
			}()

			stdout, _, err := runOrgCmdStreamsContext(ctx, append([]string{"cleanup", "--state-dir", fx.stateDir}, tc.extra...)...)
			close(stop)
			if !<-fired {
				t.Fatalf("the script's git branch -D did not run, so nothing was interrupted: err %v, stdout %q", err, stdout)
			}
			// The command to run again keeps --branch, or its lack, and
			// --state-dir (plan 2026-11-01-org-ledger-gaps, AC1).
			words := slices.Concat([]string{"ralph org cleanup"}, tc.extra, []string{"--state-dir", shellQuoteIfNeeded(fx.stateDir)})
			want := "org: cleanup: interrupted (test stopped) before the run ended: the lines above say what was cleaned up " +
				"or skipped; run " + strings.Join(words, " ") + " again for what is left"
			if err == nil || err.Error() != want || ExitCode(err) != 1 {
				t.Errorf("cleanup interrupted at git branch -D: err %v (exit %d); want %q, exit 1", err, ExitCode(err), want)
			}
			if stdout != "CLEANED: feat/a org-a\n" {
				t.Errorf("stdout = %q; want the CLEANED: line of feat/a", stdout)
			}
			if _, err := os.Stat(fx.wt); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("the worktree %s is still there (stat: %v)", fx.wt, err)
			}
			if got := gateCLIGitOut(t, fx.work, "branch", "--list", "feat/a"); got != "" {
				t.Errorf("the branch feat/a is still there: %q", got)
			}
			if got := fx.cleanedEvents(t); len(got) != 1 {
				t.Errorf("worktree_cleaned events = %+v; want one, the cleanup having run to its end", got)
			}
		})
	}
}

// cleanupSteps over merge-order's results: without --branch, only a branch
// with a binding that is its org's latest is dealt with, cleaned up at its PR's
// head when the PR is merged and reported when gh did not tell (failed,
// limit_forks, skipped), with merge-order's PRLookupWhy (S-X1 of the plan)
// when it has one; with --branch, that branch alone, its BoundTo ignored, or
// the refusal of a branch not in the ledger, with no binding, no PR, a PR gh
// did not read (its PRLookupWhy escaped), or a PR that is not merged. No entry
// is nothing to do without --branch, and gh not found is refused either way.
func TestCleanupSteps_PicksTheMergedBranches(t *testing.T) {
	const head = "0123456789abcdef0123456789abcdef01234567"
	bind := func(boundTo string) *org.MergeOrderBinding {
		return &org.MergeOrderBinding{OrgID: "a", Feature: "f", Split: "s", TS: "t", BoundTo: boundTo}
	}
	pr := func(state string) *org.MergeOrderPR {
		return &org.MergeOrderPR{Number: 7, State: state, URL: "https://github.com/o/r/pull/7", HeadRefOid: head, BaseRefName: "main"}
	}
	entry := func(branch, class string, p *org.MergeOrderPR, lookup string, b *org.MergeOrderBinding) org.MergeOrderEntry {
		return org.MergeOrderEntry{Branch: branch, Class: class, PR: p, PRLookup: lookup, Binding: b}
	}
	withWhy := func(e org.MergeOrderEntry, why string) org.MergeOrderEntry {
		e.PRLookupWhy = why
		return e
	}
	const (
		failedWhy  = "exit status 1: HTTP 502\x1b[31m"
		skippedWhy = `gh pr list --head feat/x -R github.com/o/r returned PR #9 with the url "https://github.com/x/r/pull/9", ` +
			"which is not a PR of origin's repository github.com/o/r; check origin (git remote get-url origin)"
	)
	found := org.MergeOrderPRLookupFound
	res := org.MergeOrderResult{GHAvailable: true, Entries: []org.MergeOrderEntry{
		entry("feat/open", org.MergeClassReady, pr("OPEN"), found, bind("")),
		entry("feat/merged", org.MergeClassMerged, pr("MERGED"), found, bind("")),
		entry("feat/rebound", org.MergeClassMerged, pr("MERGED"), found, bind("feat/merged")),
		entry("feat/gate-only", org.MergeClassMerged, pr("MERGED"), found, nil),
		entry("feat/closed", org.MergeClassClosed, pr("CLOSED"), found, bind("")),
		entry("feat/none", org.MergeClassNoGate, nil, org.MergeOrderPRLookupNone, bind("")),
		withWhy(entry("feat/failed", org.MergeClassNoGate, nil, org.MergeOrderPRLookupFailed, bind("")), failedWhy),
		entry("feat/forks", org.MergeClassNoGate, nil, org.MergeOrderPRLookupLimitForks, bind("")),
		withWhy(entry("feat/skipped", org.MergeClassNoGate, nil, org.MergeOrderPRLookupSkipped, bind("")), skippedWhy),
		entry("feat/failed-rebound", org.MergeClassNoGate, nil, org.MergeOrderPRLookupFailed, bind("feat/merged")),
		entry("feat/failed-gate-only", org.MergeClassNoGate, nil, org.MergeOrderPRLookupFailed, nil),
	}}
	unknown := func(why string) string {
		return "whether its PR was merged is not known (" + why + "), so nothing was removed: once gh reads its PR, run ralph " +
			"org cleanup again"
	}
	steps, err := cleanupSteps(res, "", "")
	want := []cleanupStep{
		{branch: "feat/merged", head: head, pr: 7},
		{branch: "feat/failed", unknown: unknown("gh pr list failed for the branch: " + failedWhy)},
		{branch: "feat/forks", unknown: unknown("gh returned its limit of PRs for the branch, all from forks")},
		{branch: "feat/skipped", unknown: unknown("no answer of gh was used: " + skippedWhy)},
	}
	if err != nil || !reflect.DeepEqual(steps, want) {
		t.Errorf("cleanupSteps without --branch = %+v, %v; want %+v", steps, err, want)
	}

	const gone = "org: cleanup: no feature binding or gate record in the ledger names the branch feat/nope, so nothing was " +
		"removed: ralph org merge-order --all lists the ledger's branches"
	const noGH = "org: cleanup: gh is not on PATH, so whether the PRs of the ledger's feature branches were merged cannot be " +
		"told, and nothing was removed: put gh on PATH and run ralph org cleanup again"
	// The command to run again keeps --branch (plan
	// 2026-11-01-org-ledger-gaps, AC1).
	noGHBranch := strings.Replace(noGH, "run ralph org cleanup again", "run ralph org cleanup --branch feat/merged again", 1)
	noGHRes := res
	noGHRes.GHAvailable = false
	for _, tc := range []struct {
		name    string
		res     org.MergeOrderResult
		branch  string
		want    []cleanupStep
		wantErr string
	}{
		{"a merged branch", res, "feat/merged", []cleanupStep{{branch: "feat/merged", head: head, pr: 7}}, ""},
		{"an old binding named", res, "feat/rebound", []cleanupStep{{branch: "feat/rebound", head: head, pr: 7}}, ""},
		{"a branch not in the ledger", res, "feat/nope", nil, gone},
		{"gate records only", res, "feat/gate-only", nil, "org: cleanup: no complete feature binding in the ledger names the " +
			"branch feat/gate-only, only a gate record, so it has no feature worktree to clean up and nothing was removed: " +
			"cleanup removes only the worktree ralph org start --plan made for a feature org"},
		{"no PR", res, "feat/none", nil, "org: cleanup: gh found no PR for the branch feat/none, so nothing was removed: cleanup " +
			"removes the feature worktree of a branch whose PR was merged"},
		{"a PR gh did not read", res, "feat/failed", nil, "org: cleanup: whether the PR of the branch feat/failed was merged is " +
			`not known (gh pr list failed for the branch: exit status 1: HTTP 502\x1b[31m), so nothing was removed: once gh reads ` +
			"its PR, run ralph org cleanup --branch feat/failed again"},
		{"a PR gh did not use", res, "feat/skipped", nil, "org: cleanup: whether the PR of the branch feat/skipped was merged is " +
			"not known (no answer of gh was used: " + skippedWhy + "), so nothing was removed: once gh reads its PR, run ralph org " +
			"cleanup --branch feat/skipped again"},
		{"an open PR", res, "feat/open", nil, "org: cleanup: PR #7 of the branch feat/open is OPEN, not MERGED, so nothing was " +
			"removed: cleanup removes the feature worktree of a branch whose PR was merged"},
		{"a closed PR", res, "feat/closed", nil, "org: cleanup: PR #7 of the branch feat/closed is CLOSED, not MERGED, so " +
			"nothing was removed: cleanup removes the feature worktree of a branch whose PR was merged"},
		{"no entry", org.MergeOrderResult{}, "", nil, ""},
		{"no entry, a branch named", org.MergeOrderResult{}, "feat/nope", nil, gone},
		{"gh not found", noGHRes, "", nil, noGH},
		{"gh not found, a branch named", noGHRes, "feat/merged", nil, noGHBranch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			steps, err := cleanupSteps(tc.res, tc.branch, "")
			switch {
			case tc.wantErr == "" && err != nil:
				t.Errorf("err = %v; want none", err)
			case tc.wantErr != "" && (err == nil || err.Error() != tc.wantErr):
				t.Errorf("err = %v; want %q", err, tc.wantErr)
			}
			if !reflect.DeepEqual(steps, tc.want) {
				t.Errorf("steps = %+v; want %+v", steps, tc.want)
			}
		})
	}
}

// The refusal for a branch whose PR gh did not read ends with the command to
// run again, and a person pastes it: a branch name may hold ;, &, |, $, ', and
// (, so the command has the branch as one shell word, while the sentence
// before it names the branch as it is. A plain branch is as it was.
func TestOrgCleanup_NamedStepQuotesTheBranchInTheCommand(t *testing.T) {
	for _, tc := range []struct{ name, branch, word string }{
		{"a plain branch", "feat/a", "feat/a"},
		{"a semicolon", "feat/a;b", "'feat/a;b'"},
		{"a command substitution", "feat/$(x)", "'feat/$(x)'"},
		{"a single quote", "feat/a'b", `'feat/a'\''b'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := org.MergeOrderResult{GHAvailable: true, Entries: []org.MergeOrderEntry{{
				Branch: tc.branch, Class: org.MergeClassNoGate, PRLookup: org.MergeOrderPRLookupFailed,
				Binding: &org.MergeOrderBinding{OrgID: "a", Feature: "f", Split: "s", TS: "t"},
			}}}
			_, err := cleanupNamedStep(res, tc.branch, "")
			want := "org: cleanup: whether the PR of the branch " + tc.branch + " was merged is not known (gh pr list failed " +
				"for the branch), so nothing was removed: once gh reads its PR, run ralph org cleanup --branch " + tc.word + " again"
			if err == nil || err.Error() != want {
				t.Errorf("cleanupNamedStep error = %v; want %q", err, want)
			}
		})
	}
}

// cleanupOriginRefusal over each problem org.ReadMergeOrigin returns (S-X1 of
// the plan): none for a repository read; the interruption with ctx's cause; and
// for a get-url that timed out or failed, or an origin with no GitHub
// repository, a refusal worded as ralph org merge's, git's error escaped,
// that says the cleanup removes only on the word of origin's PR, with the
// command to run again, --branch quoted as one shell word. A problem it does
// not know is refused too.
func TestOrgCleanup_OriginRefusalForEachProblem(t *testing.T) {
	const only = "ralph org cleanup removes a feature worktree only once the PR of its branch in origin's repository is merged"
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("test stopped"))
	for _, tc := range []struct {
		name    string
		problem org.MergeOriginProblem
		err     error
		branch  string
		want    string // "" for no refusal
	}{
		{"read", org.MergeOriginRead, nil, "", ""},
		{"interrupted", org.MergeOriginInterrupted, nil, "",
			"org: cleanup: interrupted (test stopped) while git remote get-url origin ran: nothing was removed"},
		{"timed out", org.MergeOriginTimedOut, errors.New("timed out after 15s: signal: killed"), "",
			"org: cleanup: git remote get-url origin failed (timed out after 15s: signal: killed), so origin's repository is " +
				"unknown and nothing was removed: check the repository's git config and run ralph org cleanup again"},
		{"get-url failed", org.MergeOriginGetURLFailed, errors.New("exit status 2: error: No such remote 'origin'\n\x1b[31m"),
			"feat/a", `org: cleanup: git remote get-url origin failed (exit status 2: error: No such remote 'origin'\n\x1b[31m), so ` +
				"origin's repository is unknown and nothing was removed: " + only + ", so add origin (git remote add origin <URL>) " +
				"and run ralph org cleanup --branch feat/a again"},
		{"no repository", org.MergeOriginNoRepository, nil, "feat/a;b",
			"org: cleanup: origin's URL names no host and owner/repo (a local path, file://, or a scheme other than https, http, " +
				"ssh and git), so there is no GitHub repository to read the PRs in and nothing was removed: " + only + ", so point " +
				"origin at the GitHub repository (git remote set-url origin <URL>) and run ralph org cleanup --branch 'feat/a;b' again"},
		{"a problem it does not know", org.MergeOriginProblem(99), nil, "",
			"org: cleanup: origin's repository could not be read (problem 99), so nothing was removed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := cleanupOriginRefusal(ctx, tc.problem, tc.err, tc.branch, "")
			switch {
			case tc.want == "" && err != nil:
				t.Errorf("refusal = %v; want none", err)
			case tc.want != "" && (err == nil || err.Error() != tc.want):
				t.Errorf("refusal = %v;\nwant %q", err, tc.want)
			}
		})
	}
}

// cleanupPrinter over the outcomes of a run, fed as runCleanupSteps feeds
// them: CLEANED: with what it resumed from and an error of its record (which
// stops the run with that error), SKIPPED: with the refusal, the failure, or
// why the PR is unknown, MANUAL: with a command by hand, no line for a branch
// with nothing left or an error without a cleanup, nothing to clean up when
// no line was printed, values escaped, and exit 1 counting the branches left.
func TestCleanupPrinter_EachOutcome(t *testing.T) {
	type outcome struct {
		branch  string
		unknown string
		res     org.CleanupResult
		err     error
	}
	// A command by hand comes only with a refusal ralph cannot go on from,
	// such as a missing record, and with a failure.
	const manual = "cd /r && ./scripts/ralph-worktree.sh cleanup --id org-a --force-branch"
	const branchDelete = "git -C /r branch -D feat/a"
	cleaned := org.CleanupResult{Cleaned: true, Recorded: true, OrgID: "a"}
	resumed := cleaned
	resumed.Resumed = org.CleanupResumedWorktreeGone
	notRecorded := org.CleanupResult{Cleaned: true, Resumed: org.CleanupResumedRecordOnly, OrgID: "a"}
	recordErr := errors.New("org: cleanup: the feature worktree /w of the branch feat/a and its record org-a were removed, but " +
		"worktree_cleaned could not be recorded (denied)")
	gitErr := errors.New("org: cleanup: git worktree list --porcelain in /r: exit 128: fatal: not a git repository")
	nothingLeft := org.CleanupResult{Refused: "the worktree record org-a of the branch feat/a does not exist, and neither its " +
		"feature worktree /w nor the branch is left: nothing is left to clean up", NothingLeft: true, OrgID: "a"}
	const unknown = "whether its PR was merged is not known (gh pr list failed for the branch)"

	for _, tc := range []struct {
		name       string
		outcomes   []outcome
		wantStdout string
		wantErr    string
	}{
		{"no branch", nil, "nothing to clean up\n", ""},
		{"cleaned up", []outcome{{branch: "feat/a", res: cleaned}}, "CLEANED: feat/a org-a\n", ""},
		{"cleaned up from what an earlier cleanup left", []outcome{{branch: "feat/a", res: resumed}},
			"CLEANED: feat/a org-a (resumed: worktree_gone)\n", ""},
		{"cleaned up, worktree_cleaned not recorded", []outcome{{branch: "feat/a", res: notRecorded, err: recordErr},
			{branch: "feat/b", res: cleaned}}, "CLEANED: feat/a org-a (resumed: record_only; " + recordErr.Error() + ")\n",
			recordErr.Error()},
		{"refused, with the command by hand", []outcome{{branch: "feat/a", res: org.CleanupResult{Refused: "the worktree record " +
			"org-a of the branch feat/a does not exist, so ralph removes nothing: the branch feat/a at \x1b left",
			Manual: branchDelete}}}, `SKIPPED: feat/a the worktree record org-a of the branch feat/a does not exist, so ralph ` +
			`removes nothing: the branch feat/a at \x1b left` + "\nMANUAL: " + branchDelete + "\n", cleanupLeftErr(1)},
		{"refused, no command by hand", []outcome{{branch: "feat/a", res: org.CleanupResult{Refused: "the org runs"}}},
			"SKIPPED: feat/a the org runs\n", cleanupLeftErr(1)},
		{"failed", []outcome{{branch: "feat/a", res: org.CleanupResult{Failed: "scripts/ralph-worktree.sh cleanup --id org-a " +
			"--force-branch failed", Manual: manual}}}, "SKIPPED: feat/a scripts/ralph-worktree.sh cleanup --id org-a " +
			"--force-branch failed\nMANUAL: " + manual + "\n", cleanupLeftErr(1)},
		{"no outcome", []outcome{{branch: "feat/a"}}, "SKIPPED: feat/a the cleanup ended with no outcome\n", cleanupLeftErr(1)},
		{"nothing left", []outcome{{branch: "feat/a", res: nothingLeft}}, "nothing to clean up\n", ""},
		{"an error before the removal", []outcome{{branch: "feat/a", err: gitErr}, {branch: "feat/b", res: cleaned}}, "",
			gitErr.Error()},
		{"a PR not known to be merged", []outcome{{branch: "feat/a", unknown: unknown}}, "SKIPPED: feat/a " + unknown + "\n",
			cleanupLeftErr(1)},
		{"several branches", []outcome{{branch: "feat/a", res: cleaned}, {branch: "feat/b", res: nothingLeft},
			{branch: "feat/c", res: org.CleanupResult{Refused: "the org runs"}}, {branch: "feat/d", unknown: unknown}},
			"CLEANED: feat/a org-a\nSKIPPED: feat/c the org runs\nSKIPPED: feat/d " + unknown + "\n", cleanupLeftErr(2)},
		{"control characters are escaped", []outcome{{branch: "feat/\x1b[31m", res: org.CleanupResult{Cleaned: true,
			Recorded: true, OrgID: "a\x1b"}}}, `CLEANED: feat/\x1b[31m org-a\x1b` + "\n", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			p := &cleanupPrinter{w: &out}
			err := func() error {
				for _, o := range tc.outcomes {
					if o.unknown != "" {
						p.unknown(o.branch, o.unknown)
						continue
					}
					if err := p.result(o.branch, o.res, o.err); err != nil {
						return err
					}
				}
				return p.finish()
			}()
			if out.String() != tc.wantStdout {
				t.Errorf("stdout:\n%q\nwant:\n%q", out.String(), tc.wantStdout)
			}
			switch {
			case tc.wantErr == "" && err != nil:
				t.Errorf("err = %v; want nil (exit 0)", err)
			case tc.wantErr != "" && (err == nil || err.Error() != tc.wantErr):
				t.Errorf("err = %v; want %q", err, tc.wantErr)
			}
		})
	}
}

// C3-L3 of docs/reports/self-review-2026-10-28-org-merge-cleanup.md: with the
// command's context done, runCleanupSteps exits 1 with an interruption that
// says what the run did so far, after the steps or before a cleanup. With no
// line printed, as when no branch is left to deal with, it says that no
// branch was cleaned up, not that the lines above say what was, and prints
// no nothing to clean up line; with only SKIPPED: lines, it says what was
// cleaned up or skipped. No cleanup runs, so the runtime is nil.
func TestRunCleanupSteps_InterruptionSaysWhatTheRunDid(t *testing.T) {
	const why = "whether its PR was merged is not known (gh pr list failed for the branch)"
	unknown := cleanupStep{branch: "feat/a", unknown: why}
	merged := cleanupStep{branch: "feat/b", head: "0123456789abcdef0123456789abcdef01234567", pr: 2}
	for _, tc := range []struct {
		name       string
		steps      []cleanupStep
		wantStdout string
		wantErr    string
	}{
		{"no branch", nil, "", "org: cleanup: interrupted (test stopped) before the run ended: no branch was cleaned up; run " +
			"ralph org cleanup again for what is left"},
		{"only a PR not known to be merged", []cleanupStep{unknown}, "SKIPPED: feat/a " + why + "\n", "org: cleanup: " +
			"interrupted (test stopped) before the run ended: the lines above say what was cleaned up or skipped; run ralph org " +
			"cleanup again for what is left"},
		{"before the first cleanup", []cleanupStep{merged}, "", "org: cleanup: interrupted (test stopped) before the cleanup " +
			"of the branch feat/b: it and the branches after it were not cleaned up, and no branch was cleaned up before it; run " +
			"ralph org cleanup again"},
		{"before a cleanup after a SKIPPED: line", []cleanupStep{unknown, merged}, "SKIPPED: feat/a " + why + "\n",
			"org: cleanup: interrupted (test stopped) before the cleanup of the branch feat/b: it and the branches after it were " +
				"not cleaned up, and the lines above say what was cleaned up or skipped before it; run ralph org cleanup again"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			cancel(errors.New("test stopped"))
			var out strings.Builder
			err := runCleanupSteps(ctx, nil, "/r", "", "", tc.steps, &cleanupPrinter{w: &out})
			if err == nil || err.Error() != tc.wantErr || ExitCode(err) != 1 {
				t.Errorf("err = %v (exit %d); want %q, exit 1", err, ExitCode(err), tc.wantErr)
			}
			if out.String() != tc.wantStdout {
				t.Errorf("stdout:\n%q\nwant:\n%q", out.String(), tc.wantStdout)
			}
		})
	}
}

// cleanup's help says what it removes, when it removes it, what it refuses,
// where to run it, how it relates to merge, its lines, its exit codes, and
// that it refuses --org-id; org --help names the verb, says that merge cleans
// up after its merge, and lists cleanup among the verbs that refuse --org-id.
func TestOrgCleanup_Help(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		wants []string
	}{
		{[]string{"cleanup", "--help"}, []string{
			"the feature worktree, the local branch, and the record of scripts/ralph-worktree.sh",
			"./scripts/ralph-worktree.sh cleanup --id org-<org_id> --force-branch in the main worktree",
			"records worktree_cleaned in the ledger with via=cleanup",
			"ralph org merge does the same cleanup right after its own merge",
			"Run it from the main worktree, not from inside the feature worktree it removes.",
			"cleans up each branch with a feature binding whose PR gh reads as MERGED, at the PR's head (headRefOid)",
			"A branch whose PR gh could not read (pr_lookup failed, limit_forks, or skipped in merge-order's --json) gets a SKIPPED: line",
			"The old branch of an org_id that was bound again to another branch later is handled only when --branch names it.",
			"refuses, removing nothing, when the ledger does not name it, it has no feature binding, gh does not read its PR, or the PR is not MERGED",
			"As ralph org merge does, cleanup reads the PRs in origin's repository only: gh pr list gets -R <host>/<owner>/<repo>, " +
				"read from git remote get-url origin, and when gh returns a PR whose url names another host or owner/repo, no PR is " +
				"used for any branch",
			"When origin's URL names no GitHub repository (a local path, file://, or another scheme) or git remote get-url origin " +
				"fails, cleanup refuses before it reads the ledger, and removes nothing.",
			"that the branch's org is not running; that the ledger's directory (the org state dir) is not the feature worktree or " +
				"inside it, since the ledger would go with the worktree; that the record is the binding's (its branch, path, and " +
				"kind org)",
			"that the worktree has no uncommitted change or untracked file and no rebase or merge in progress (files .gitignore " +
				"names do not count and go with the worktree, but for the ledger's directory); that the local branch is at the " +
				"head the PR merged; and that the worktree has that branch checked out at that head",
			"The ledger's directory and the working directory are compared with the worktree by file identity, so another " +
				"spelling of the same directory counts too.",
			"ralph org regate, ralph org merge, and another cleanup are refused on the ledger (one runs at a time; it waits up to " +
				"1 s for regate.lock), and so are ralph org spawn and start of the org whose feature worktree it removes",
			"CLEANED: <branch> org-<org_id>, with (resumed: <what was left>)",
			"SKIPPED: <branch> <why>, followed, when there is one, by MANUAL: <the command that cleans up by hand once the reason is checked>",
			"With no such line, it prints nothing to clean up.",
			"For a worktree whose directory is gone but which git still registers, that last check is made on the registration: " +
				"it must be on that branch, and the branch must still exist.",
			"Only a branch ralph cannot go on with by itself gets a MANUAL: line (the record is gone or is not the binding's, " +
				"the worktree is left with no local branch, the worktree's directory is gone while git's registration of it " +
				"has a HEAD that removing it would drop, or the removal itself failed); after any other SKIPPED: line, fix " +
				"what the reason says and run ralph org cleanup --branch <branch> again; the reason names that command with the " +
				"--state-dir cleanup was given, if any.",
			// A ledger inside the worktree (cross-review cycle 2, finding 1) and a record whose worktree holds it.
			"A ledger whose directory is inside the feature worktree gets no MANUAL: line either, since cleaning up by hand " +
				"would remove the ledger with it: the reason says to move the ledger's directory out of the worktree, to a place " +
				"where no other ledger is, once no org on the ledger runs and no ralph org command runs on it, and to run ralph " +
				"org cleanup --branch <branch> again with --state-dir or [org].state_dir in ralph.local.toml naming the new " +
				"place. So does a " +
				"record that is not the binding's (its path or kind) when the worktree at the record's path holds the ledger; " +
				"after the ledger is moved, that cleanup is refused with the MANUAL: line above.",
			"Exit codes: 0 when every branch was cleaned up or there was nothing to clean up; 1 when a branch was skipped, gh is " +
				"not on PATH (whether a PR was merged cannot be told, so nothing is removed), the main worktree is not found, " +
				"--branch is refused,",
			"--branch is refused, origin names no GitHub repository or cannot be read, for an interrupt",
			// The interrupt of the last branch's cleanup (cross-review cycle 2, finding 2).
			"for an interrupt (SIGINT, SIGTERM, or SIGHUP; what was cleaned up before it stays cleaned up, and an interrupt " +
				"while the last branch is cleaned up still exits 1, after the lines printed), and for any other error.",
			"so --org-id is refused",
			"--branch string",
		}},
		{[]string{"--help"}, []string{
			"cleanup Clean up the feature worktree of a feature branch whose PR was merged on GitHub",
			"records the merge, and then cleans up the branch's feature worktree.",
			"`ralph org cleanup` cleans up the feature worktree of a branch whose PR was merged on GitHub.",
			"stop --all, disband --all, merge-order, merge, cleanup, regate, wait --inbox,",
		}},
	} {
		out, err := runOrgCmd(t, tc.args...)
		if err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		help := strings.Join(strings.Fields(out), " ")
		for _, want := range tc.wants {
			if !strings.Contains(help, want) {
				t.Errorf("%v does not say %q:\n%s", tc.args, want, out)
			}
		}
		// ralph no longer reads RALPH_ORG_STATE_DIR (plan
		// 2026-10-30-org-state-dir-local, AC10), so the move of the ledger
		// names the new place with --state-dir or [org].state_dir alone.
		if strings.Contains(help, "RALPH_ORG_STATE_DIR") {
			t.Errorf("%v still names RALPH_ORG_STATE_DIR:\n%s", tc.args, out)
		}
	}
}

// The help of start, spawn, and merge-order follows the cleanup after a
// merge (plan docs/plans/active/2026-10-28-org-merge-cleanup.md, S4: AC5):
// start says that merge and cleanup remove a merged feature org's worktree
// and that the script removes a closed PR's by hand, no longer that the
// script removes it after a merge; start and spawn say they are refused while
// merge or cleanup removes the org's worktree, as regateSpawnErr refuses
// them; and merge-order says what bound_to in --json is.
func TestOrgStartSpawnAndMergeOrder_HelpFollowTheCleanupAfterAMerge(t *testing.T) {
	for _, tc := range []struct {
		args            []string
		wants, notWants []string
	}{
		{[]string{"start", "--help"}, []string{
			"`ralph org disband` leaves the worktree and the branch, which `ralph org regate` works in until the PR is merged.",
			"Once the PR is merged, `ralph org merge` removes the worktree, the branch, and the record of " +
				"scripts/ralph-worktree.sh right after its merge, and `ralph org cleanup` does the same for a PR merged on GitHub",
			"For a PR closed without a merge, run `./scripts/ralph-worktree.sh cleanup --id org-<org_id> --force-branch` in the " +
				"main worktree once the branch is not needed.",
			"Both forms are refused, with nothing recorded, while `ralph org regate` works on the org's branch, or while " +
				"`ralph org merge` or `ralph org cleanup` removes the org's merged feature worktree; run start again once it ends.",
			// Plan 2026-10-29-regate-ci-checks, sync-docs: a ci regate's wait releases regate.lock.
			"A ci regate that waits for the PR's checks has released regate.lock and does not refuse them meanwhile (see ralph org regate --help).",
		}, []string{"`./scripts/ralph-worktree.sh cleanup --id org-<org_id>` removes them", "run start again once the regate ends"}},
		{[]string{"spawn", "--help"}, []string{
			"A spawn (not a --dry-run) into an org is refused, with nothing recorded, while `ralph org regate` works on that " +
				"org's branch, or while `ralph org merge` or `ralph org cleanup` removes that org's merged feature worktree; run " +
				"the spawn again once it ends.",
			// Plan 2026-10-29-regate-ci-checks, sync-docs: a ci regate's wait releases regate.lock.
			"A ci regate that waits for the PR's checks has released regate.lock and does not refuse it meanwhile (see ralph org regate --help).",
		}, []string{"run the spawn again once the regate ends"}},
		{[]string{"merge-order", "--help"}, []string{
			"Each --json entry with a binding also has bound_to in it, which the text does not show either: the branch of the " +
				"org's latest complete binding when that is another branch",
			"ralph org cleanup leaves such a branch out unless --branch names it.",
		}, nil},
	} {
		out, err := runOrgCmd(t, tc.args...)
		if err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		help := strings.Join(strings.Fields(out), " ")
		for _, want := range tc.wants {
			if !strings.Contains(help, want) {
				t.Errorf("%v does not say %q:\n%s", tc.args, want, out)
			}
		}
		for _, old := range tc.notWants {
			if strings.Contains(help, old) {
				t.Errorf("%v still says %q:\n%s", tc.args, old, out)
			}
		}
	}
}
