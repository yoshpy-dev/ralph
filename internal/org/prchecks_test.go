package org

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

// The gh reading of a branch's PR, its head, and its checks that ralph org
// merge and the ci regate share (prchecks.go, S3 of plan
// docs/plans/active/2026-10-29-regate-ci-checks.md): findOriginPR,
// readPRHead, and readPRChecks, each called without a mergeCheck, with
// mxExec (merge_fake_test.go) standing in for gh, so no test runs gh or
// reaches a network. merge_test.go has what checkMerge words from them.

// prOrigin is the fixture's origin, github.com/o/r (mxRepoArg).
var prOrigin = MergeOrigin{apiHost: "github.com", repo: mergeRepo{owner: "o", name: "r"}}

// prExec is an mxExec whose gh answers r under key and fails for any other
// call.
func prExec(key string, r mgReply) *mxExec {
	return &mxExec{replies: map[string]mgReply{key: r}}
}

// prAssertOneCall fails the test unless e ran exactly one command: gh with
// args in /repo, with GH_PROMPT_DISABLED=1 and a deadline.
func prAssertOneCall(t *testing.T, e *mxExec, args ...string) {
	t.Helper()
	calls := e.snapshot()
	if len(calls) != 1 {
		t.Fatalf("%d calls, want 1: %q", len(calls), e.keys())
	}
	c := calls[0]
	if c.name != "gh" || !slices.Equal(c.args, args) || c.dir != "/repo" || !slices.Equal(c.env, []string{"GH_PROMPT_DISABLED=1"}) ||
		c.deadline.IsZero() {
		t.Errorf("call = %s %q in %q with %q, deadline %v; want gh %q in /repo with GH_PROMPT_DISABLED=1 and a deadline", c.name,
			c.args, c.dir, c.env, c.deadline, args)
	}
}

// prBlockedExec is prExec whose call under key waits for its ctx, so it runs
// past a timeout.
func prBlockedExec(key string) *mxExec {
	e := prExec(key, mgReply{})
	e.block = key
	return e
}

// prCanceled is a ctx that is done before any call.
func prCanceled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// TestPRFind_ReadsTheBranchsPRPinnedToOrigin: findOriginPR runs gh pr list
// --head feat/a pinned to origin and picks, of the PRs not from a fork, the
// highest-numbered one; it is usable only when its url names origin's
// repository, it is OPEN, and it merges into the default branch, and each
// other answer says which of these it is not, checked in that order, with the
// PR kept for a foreign, a closed or merged, and a wrong-base PR. No PR, only
// fork PRs, and a limit full of fork PRs keep none.
func TestPRFind_ReadsTheBranchsPRPinnedToOrigin(t *testing.T) {
	fork := func(n int) ghPullRequest {
		pr := moPR(n, "OPEN", "main")
		pr.URL, pr.IsCrossRepository = "https://github.com/fork/r/pull/1", true
		return pr
	}
	foreign := func(state string) ghPullRequest {
		pr := moPR(7, state, "main")
		pr.URL = "https://github.com/x/r/pull/7"
		return pr
	}
	forks := make([]ghPullRequest, mergeOrderGHLimit)
	for i := range forks {
		forks[i] = fork(i + 100)
	}
	for _, tc := range []struct {
		name    string
		out     string
		problem originPRProblem
		number  int // the PR kept, 0 for none
	}{
		{"open into the default branch", moPRs(moPR(7, "OPEN", "main")), originPRUsable, 7},
		{"the highest of origin's PRs, forks left out", moPRs(moPR(5, "CLOSED", "main"), moPR(7, "OPEN", "main"), fork(9)),
			originPRUsable, 7},
		{"closed", moPRs(moPR(7, "CLOSED", "main")), originPRNotOpen, 7},
		{"merged", moPRs(moPR(7, "MERGED", "main")), originPRNotOpen, 7},
		{"closed into another base is not open first", moPRs(moPR(7, "CLOSED", "develop")), originPRNotOpen, 7},
		{"open into another base", moPRs(moPR(7, "OPEN", "develop")), originPRWrongBase, 7},
		{"a PR of another repository", moPRs(foreign("OPEN")), originPRForeign, 7},
		{"a closed PR of another repository is foreign first", moPRs(foreign("CLOSED")), originPRForeign, 7},
		{"no PR", "[]", originPRNone, 0},
		{"only fork PRs", moPRs(fork(9)), originPRNone, 0},
		{"a limit full of fork PRs", moPRs(forks...), originPRLimitForks, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := prExec(mxPRList, mgReply{out: tc.out})
			f := findOriginPR(context.Background(), "/repo", e.run, time.Minute, prOrigin, "feat/a", "main")
			number := 0
			if f.pr != nil {
				number = f.pr.Number
			}
			if f.problem != tc.problem || number != tc.number || f.err != nil || f.cmd != "gh pr list --head feat/a -R "+mxRepoArg {
				t.Errorf("findOriginPR = %q, problem %d, PR #%d, err %v; want problem %d, PR #%d, no error", f.cmd, f.problem, number,
					f.err, tc.problem, tc.number)
			}
			if tc.problem == originPRUsable && (f.pr.HeadRefOid != moHeadA || f.pr.URL != "https://github.com/o/r/pull/7" ||
				f.pr.State != "OPEN" || f.pr.BaseRefName != "main") {
				t.Errorf("PR = %+v, want #7 open at moHeadA into main", *f.pr)
			}
			prAssertOneCall(t, e, "pr", "list", "--head", "feat/a", "--state", "all", "--limit", "1000", "--json", mergeOrderGHFields,
				"-R", mxRepoArg)
		})
	}
}

// TestPRFind_GHThatFailsOrIsInterrupted: a gh pr list that fails, is not
// found, prints no JSON list of PRs, or times out is originPRGHFailed with
// its error, and ctx done is originPRInterrupted; neither keeps a PR.
func TestPRFind_GHThatFailsOrIsInterrupted(t *testing.T) {
	ghErr := errors.New("exit status 1: HTTP 502: Bad Gateway")
	for _, tc := range []struct {
		name  string
		reply mgReply
		want  string // the error's text
	}{
		{"fails", mgReply{err: ghErr}, ghErr.Error()},
		{"is not found", mxNotFound, mxNotFound.err.Error()},
		{"prints no JSON list", mgReply{out: "<html>"}, "its output is not a JSON list of PRs: invalid character '<' looking for " +
			"beginning of value"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := prExec(mxPRList, tc.reply)
			f := findOriginPR(context.Background(), "/repo", e.run, time.Minute, prOrigin, "feat/a", "main")
			if f.problem != originPRGHFailed || f.pr != nil || f.err == nil || f.err.Error() != tc.want ||
				tc.reply.err != nil && !errors.Is(f.err, tc.reply.err) {
				t.Errorf("findOriginPR = problem %d, PR %+v, err %v; want originPRGHFailed, no PR, %q", f.problem, f.pr, f.err, tc.want)
			}
		})
	}
	t.Run("times out", func(t *testing.T) {
		e := prBlockedExec(mxPRList)
		f := findOriginPR(context.Background(), "/repo", e.run, 30*time.Millisecond, prOrigin, "feat/a", "main")
		if f.problem != originPRGHFailed || f.pr != nil || f.err == nil ||
			f.err.Error() != "timed out after 30ms: context deadline exceeded" {
			t.Errorf("findOriginPR = problem %d, PR %+v, err %v; want originPRGHFailed with the timeout", f.problem, f.pr, f.err)
		}
	})
	t.Run("interrupted", func(t *testing.T) {
		e := prBlockedExec(mxPRList)
		f := findOriginPR(prCanceled(), "/repo", e.run, time.Minute, prOrigin, "feat/a", "main")
		if f.problem != originPRInterrupted || f.pr != nil || !errors.Is(f.err, context.Canceled) {
			t.Errorf("findOriginPR = problem %d, PR %+v, err %v; want originPRInterrupted, no PR", f.problem, f.pr, f.err)
		}
	})
}

// TestPRHead_ReadsTheHeadOrSaysWhyNot: readPRHead reads headRefOid from gh pr
// view <number> -R <repo> --json headRefOid; a gh that fails, is not found,
// or times out returns its error, and an answer that is not JSON or has no
// headRefOid an error that says so.
func TestPRHead_ReadsTheHeadOrSaysWhyNot(t *testing.T) {
	t.Run("reads it", func(t *testing.T) {
		e := prExec(mxPRView, mgReply{out: `{"headRefOid":"` + moHeadA + `"}` + "\n"})
		head, err := readPRHead(context.Background(), "/repo", e.run, time.Minute, mxRepoArg, 7)
		if head != moHeadA || err != nil {
			t.Errorf("readPRHead = %q, %v; want %q", head, err, moHeadA)
		}
		prAssertOneCall(t, e, "pr", "view", "7", "-R", mxRepoArg, "--json", "headRefOid")
	})
	ghErr := errors.New("exit status 1: GraphQL: Could not resolve to a PullRequest with the number of 7.")
	for _, tc := range []struct {
		name  string
		reply mgReply
		want  string // the error's text
	}{
		{"fails", mgReply{err: ghErr}, ghErr.Error()},
		{"is not found", mxNotFound, mxNotFound.err.Error()},
		{"not JSON", mgReply{out: "<html>"}, "its output is not the PR's JSON: invalid character '<' looking for beginning of value"},
		{"no headRefOid", mgReply{out: `{"state":"OPEN"}`}, "its output has no headRefOid"},
		{"an empty headRefOid", mgReply{out: `{"headRefOid":""}`}, "its output has no headRefOid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := prExec(mxPRView, tc.reply)
			head, err := readPRHead(context.Background(), "/repo", e.run, time.Minute, mxRepoArg, 7)
			if head != "" || err == nil || err.Error() != tc.want || tc.reply.err != nil && !errors.Is(err, tc.reply.err) {
				t.Errorf("readPRHead = %q, %v; want no head and %q", head, err, tc.want)
			}
		})
	}
	t.Run("times out", func(t *testing.T) {
		e := prBlockedExec(mxPRView)
		head, err := readPRHead(context.Background(), "/repo", e.run, 30*time.Millisecond, mxRepoArg, 7)
		if head != "" || err == nil || err.Error() != "timed out after 30ms: context deadline exceeded" {
			t.Errorf("readPRHead = %q, %v; want no head and the timeout", head, err)
		}
	})
}

// TestPRChecks_CountsTheChecksGHListed: readPRChecks reads gh pr checks'
// stdout whatever its exit (8 while a check is pending) and counts the
// checks, the passed ones, and keeps apart the labels of the pending ones and
// of those that ended and are not pass or skipping (fail, cancel, a bucket
// not known), with both in gh's order in notPassed; skipping counts as
// neither. [] and gh's sentence of no checks for the branch are zero checks
// read.
func TestPRChecks_CountsTheChecksGHListed(t *testing.T) {
	check := func(bucket, name, state string) ghPRCheck {
		return ghPRCheck{Bucket: bucket, Name: name, State: state, Workflow: "ci"}
	}
	const (
		lint = "ci / lint (pending, QUEUED)"
		test = "ci / test (fail, FAILURE)"
		e2e  = "ci / e2e (cancel, CANCELLED)"
	)
	for _, tc := range []struct {
		name                       string
		reply                      mgReply
		total, passed              int
		pending, failed, notPassed []string
	}{
		{"pass", mgReply{out: mxChecks(mxPass("verify"), mxPass("test"))}, 2, 2, nil, nil, nil},
		{"pass and skipping", mgReply{out: mxChecks(mxPass("verify"), check("skipping", "deploy", "SKIPPED"))}, 2, 1, nil, nil, nil},
		{"skipping only", mgReply{out: mxChecks(check("skipping", "deploy", "SKIPPED"), check("skipping", "docs", "SKIPPED"))}, 2, 0,
			nil, nil, nil},
		{"pending, exit 8", mgReply{out: mxChecks(mxPass("verify"), check("pending", "lint", "QUEUED")),
			err: errors.New("exit status 8")}, 2, 1, []string{lint}, nil, []string{lint}},
		{"fail and pending, in gh's order", mgReply{out: mxChecks(check("fail", "test", "FAILURE"), mxPass("verify"),
			check("pending", "lint", "QUEUED")), err: errors.New("exit status 1")}, 3, 1, []string{lint}, []string{test},
			[]string{test, lint}},
		{"fail only", mgReply{out: mxChecks(check("fail", "test", "FAILURE"), check("cancel", "e2e", "CANCELLED")),
			err: errors.New("exit status 1")}, 2, 0, nil, []string{test, e2e}, []string{test, e2e}},
		{"a bucket not known ended", mgReply{out: mxChecks(mxPass("verify"), ghPRCheck{Name: "x"})}, 2, 1, nil,
			[]string{"x (no bucket, no state)"}, []string{"x (no bucket, no state)"}},
		{"zero as []", mgReply{out: "[]"}, 0, 0, nil, nil, nil},
		{"zero as gh's sentence", mgReply{err: errors.New("exit status 1: no checks reported on the 'feat/a' branch")}, 0, 0, nil, nil,
			nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := prExec(mxPRChecks, tc.reply)
			c := readPRChecks(context.Background(), "/repo", e.run, time.Minute, mxRepoArg, 7, "feat/a")
			if c.read != prChecksListed || c.err != nil || c.cmd != "gh pr checks 7 -R "+mxRepoArg || c.stdout != tc.reply.out {
				t.Fatalf("readPRChecks = %q, read %d, err %v, stdout %q; want listed, no error, stdout %q", c.cmd, c.read, c.err,
					c.stdout, tc.reply.out)
			}
			if c.total != tc.total || c.passed != tc.passed || !slices.Equal(c.pending, tc.pending) || !slices.Equal(c.failed, tc.failed) ||
				!slices.Equal(c.notPassed, tc.notPassed) {
				t.Errorf("readPRChecks = total %d, passed %d, pending %q, failed %q, notPassed %q\nwant  total %d, passed %d, "+
					"pending %q, failed %q, notPassed %q", c.total, c.passed, c.pending, c.failed, c.notPassed, tc.total, tc.passed,
					tc.pending, tc.failed, tc.notPassed)
			}
			prAssertOneCall(t, e, "pr", "checks", "7", "-R", mxRepoArg, "--json", mergePRChecksFields)
		})
	}
}

// TestPRChecks_SaysWhyTheChecksAreUnread: a gh that fails with no JSON list
// on stdout (gh's sentence of no checks for another branch included), is not
// found, or times out is prChecksGHFailed with its error, even with a list on
// stdout for a timeout; an exit 0 with a stdout that is not a JSON list is
// prChecksNotList; ctx done is prChecksInterrupted. None counts a check, and
// stdout is what gh printed.
func TestPRChecks_SaysWhyTheChecksAreUnread(t *testing.T) {
	for _, tc := range []struct {
		name  string
		reply mgReply
		read  prChecksRead
	}{
		{"fails", mgReply{err: errors.New("exit status 1: HTTP 502: Bad Gateway")}, prChecksGHFailed},
		{"gh's sentence for another branch", mgReply{err: errors.New("exit status 1: no checks reported on the 'feat/a2' branch")},
			prChecksGHFailed},
		{"fails with output that is not a list", mgReply{out: "<html>", err: errors.New("exit status 1")}, prChecksGHFailed},
		{"is not found", mxNotFound, prChecksGHFailed},
		{"exit 0 with gh's sentence on stdout", mgReply{out: "no checks reported on the 'feat/a' branch\n"}, prChecksNotList},
		{"exit 0 with output that is not JSON", mgReply{out: "<html>"}, prChecksNotList},
		{"null is not a list", mgReply{out: "null"}, prChecksNotList},
		{"an object is not a list", mgReply{out: `{"bucket":"pass"}`}, prChecksNotList},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := prExec(mxPRChecks, tc.reply)
			c := readPRChecks(context.Background(), "/repo", e.run, time.Minute, mxRepoArg, 7, "feat/a")
			if c.read != tc.read || !errors.Is(c.err, tc.reply.err) || c.stdout != tc.reply.out || c.total != 0 || c.passed != 0 ||
				c.notPassed != nil {
				t.Errorf("readPRChecks = read %d, err %v, stdout %q, total %d; want read %d, err %v, stdout %q, nothing counted",
					c.read, c.err, c.stdout, c.total, tc.read, tc.reply.err, tc.reply.out)
			}
		})
	}
	t.Run("times out", func(t *testing.T) {
		e := prBlockedExec(mxPRChecks)
		c := readPRChecks(context.Background(), "/repo", e.run, 30*time.Millisecond, mxRepoArg, 7, "feat/a")
		if c.read != prChecksGHFailed || c.err == nil || c.err.Error() != "timed out after 30ms: context deadline exceeded" ||
			c.total != 0 {
			t.Errorf("readPRChecks = read %d, err %v, total %d; want prChecksGHFailed with the timeout", c.read, c.err, c.total)
		}
	})
	t.Run("times out with a list on stdout", func(t *testing.T) {
		run := func(ctx context.Context, _ string, _ []string, _ string, _ ...string) (string, error) {
			<-ctx.Done()
			return mxChecks(mxPass("verify")), ctx.Err()
		}
		c := readPRChecks(context.Background(), "/repo", run, 30*time.Millisecond, mxRepoArg, 7, "feat/a")
		if c.read != prChecksGHFailed || !errors.Is(c.err, context.DeadlineExceeded) || c.total != 0 ||
			!strings.HasPrefix(c.stdout, "[") {
			t.Errorf("readPRChecks = read %d, err %v, total %d, stdout %q; want prChecksGHFailed with the timeout and the list kept "+
				"as stdout", c.read, c.err, c.total, c.stdout)
		}
	})
	t.Run("interrupted", func(t *testing.T) {
		e := prBlockedExec(mxPRChecks)
		c := readPRChecks(prCanceled(), "/repo", e.run, time.Minute, mxRepoArg, 7, "feat/a")
		if c.read != prChecksInterrupted || !errors.Is(c.err, context.Canceled) || c.total != 0 {
			t.Errorf("readPRChecks = read %d, err %v, total %d; want prChecksInterrupted", c.read, c.err, c.total)
		}
	})
}
