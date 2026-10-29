package org

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// The reading of a branch's PR, its head, and its checks with gh, which
// `ralph org merge` makes (findPR, readChecks: merge.go) and the ci regate of
// plan docs/plans/active/2026-10-29-regate-ci-checks.md (D2, D4) makes too:
// findOriginPR, readPRHead, and readPRChecks. They take what they read with
// as arguments, the gh runner, the directory, the timeout, and origin's
// repository, so a caller needs no mergeCheck. They refuse nothing: each
// returns what it read and why it read nothing usable, and the caller words
// its own refusal from that. Every gh call goes through runGH, with
// GH_PROMPT_DISABLED=1, bounded by the timeout, and pinned to origin's
// repository with -R.

// runGH runs gh with args in dir through run with GH_PROMPT_DISABLED=1,
// bounded by timeout, and returns its stdout. timedOut reports that the
// timeout stopped it, and err then says so.
func runGH(ctx context.Context, dir string, run MergeOrderExec, timeout time.Duration, args ...string) (out string, timedOut bool, err error) {
	ghCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out, err = run(ghCtx, dir, []string{"GH_PROMPT_DISABLED=1"}, "gh", args...)
	if err != nil && ghCtx.Err() != nil && ctx.Err() == nil {
		return out, true, fmt.Errorf("timed out after %s: %w", timeout, err)
	}
	return out, false, err
}

// originPRProblem is why findOriginPR found no PR of the branch to use.
type originPRProblem int

const (
	// originPRUsable: the PR is origin's, OPEN, and merges into the default
	// branch.
	originPRUsable originPRProblem = iota
	// originPRInterrupted: ctx was done when gh pr list returned.
	originPRInterrupted
	// originPRGHFailed: gh pr list failed, timed out, was not found, or
	// printed no JSON list of PRs (lookupMergePR), as err says.
	originPRGHFailed
	// originPRLimitForks: gh returned mergeOrderGHLimit PRs, all from forks,
	// so the branch's own PR may be past them.
	originPRLimitForks
	// originPRNone: gh returned no PR of the branch that is not from a fork.
	originPRNone
	// originPRForeign: the PR's url names another host or owner/repo than
	// origin's (MergeOrigin.hasPR).
	originPRForeign
	// originPRNotOpen: the PR is origin's and not OPEN (MERGED or CLOSED).
	originPRNotOpen
	// originPRWrongBase: the PR is origin's and OPEN, and merges into another
	// branch than the default branch.
	originPRWrongBase
)

// originPR is the branch's PR as findOriginPR found it.
type originPR struct {
	// cmd names the lookup in messages: gh pr list --head <branch> -R
	// <origin>.
	cmd string
	// pr is the PR gh returned for the branch (pickMergePR); nil with
	// originPRGHFailed, originPRLimitForks, and originPRNone.
	pr *MergeOrderPR
	// problem is why pr is not used; originPRUsable when it is.
	problem originPRProblem
	// err is lookupMergePR's error with originPRInterrupted and
	// originPRGHFailed, nil otherwise; with originPRGHFailed it is
	// errors.Is exec.ErrNotFound for a gh not found and says "timed out
	// after" for a timeout.
	err error
}

// findOriginPR finds branch's PR as ralph org merge does (findPR, D10 of
// docs/plans/active/2026-10-17-org-auto-merge.md): `gh pr list --head
// <branch>` in dir pinned to origin with `-R <origin.Arg()>` (lookupMergePR,
// bounded by timeout), the highest-numbered PR that is not from a fork. Its
// problem is the first of these that applies: ctx done, gh failed, a limit
// full of fork PRs, no PR, a PR whose url does not name origin's repository,
// a PR that is not OPEN, and an open PR whose base is not defaultBranch.
func findOriginPR(ctx context.Context, dir string, run MergeOrderExec, timeout time.Duration, origin MergeOrigin, branch, defaultBranch string) originPR {
	pr, limitFullOfForks, err := lookupMergePR(ctx, dir, run, timeout, origin.Arg(), branch)
	f := originPR{cmd: "gh pr list --head " + branch + " -R " + origin.Arg(), pr: pr}
	switch {
	case ctx.Err() != nil:
		f.problem, f.err = originPRInterrupted, err
	case err != nil:
		f.problem, f.err = originPRGHFailed, err
	case limitFullOfForks:
		f.problem = originPRLimitForks
	case pr == nil:
		f.problem = originPRNone
	case !origin.hasPR(pr.URL):
		f.problem = originPRForeign
	case pr.State != "OPEN":
		f.problem = originPRNotOpen
	case pr.BaseRefName != defaultBranch:
		f.problem = originPRWrongBase
	}
	return f
}

// readPRHead is the head of PR number of the repository repoArg
// ([HOST/]OWNER/REPO): the headRefOid of `gh pr view <number> -R <repoArg>
// --json headRefOid` in dir (runGH, bounded by timeout). The error is gh's,
// which says "timed out after" for a timeout, or says that its output is not
// the PR's JSON or has no headRefOid.
func readPRHead(ctx context.Context, dir string, run MergeOrderExec, timeout time.Duration, repoArg string, number int) (head string, err error) {
	out, _, err := runGH(ctx, dir, run, timeout, "pr", "view", strconv.Itoa(number), "-R", repoArg, "--json", "headRefOid")
	if err != nil {
		return "", err
	}
	var pr struct {
		HeadRefOid string `json:"headRefOid"`
	}
	switch err := json.Unmarshal([]byte(out), &pr); {
	case err != nil:
		return "", fmt.Errorf("its output is not the PR's JSON: %w", err)
	case pr.HeadRefOid == "":
		return "", errors.New("its output has no headRefOid")
	}
	return pr.HeadRefOid, nil
}

// prChecksRead says whether readPRChecks read a PR's checks, and why not.
type prChecksRead int

const (
	// prChecksListed: gh printed a JSON list of checks on stdout, whatever
	// its exit, or reported no checks (mergeNoChecksSentence).
	prChecksListed prChecksRead = iota
	// prChecksInterrupted: ctx was done when gh pr checks returned.
	prChecksInterrupted
	// prChecksGHFailed: gh pr checks timed out or was not found, or failed
	// with no JSON list on stdout and without mergeNoChecksSentence, as err
	// says.
	prChecksGHFailed
	// prChecksNotList: gh pr checks exited 0 and its stdout is not a JSON
	// list.
	prChecksNotList
)

// prChecks is a PR's checks as readPRChecks read them. The counts and the
// labels are set only with prChecksListed.
type prChecks struct {
	// cmd names the call in messages: gh pr checks <number> -R <repoArg>.
	cmd string
	// read says whether the checks were read, and why not.
	read prChecksRead
	// err is gh's error with prChecksInterrupted and prChecksGHFailed, nil
	// otherwise; with prChecksGHFailed it is errors.Is exec.ErrNotFound for
	// a gh not found and says "timed out after" for a timeout.
	err error
	// stdout is what gh printed on stdout, whatever its exit: with
	// prChecksListed the JSON list, or nothing when gh reported no checks
	// with mergeNoChecksSentence (gh 2.102.0 prints no JSON then).
	stdout string
	// total is how many checks gh listed, and passed how many of them are
	// pass.
	total, passed int
	// pending are the labels (ghPRCheck.label) of the checks that are
	// pending, failed those of the checks that ended and are not pass or
	// skipping (fail, cancel, or a bucket ralph does not know), and notPassed
	// both, in gh's order.
	pending, failed, notPassed []string
}

// readPRChecks reads the checks of PR number of the repository repoArg
// ([HOST/]OWNER/REPO) as ralph org merge does (readChecks, D7 of
// docs/plans/active/2026-10-17-org-auto-merge.md): `gh pr checks <number> -R
// <repoArg> --json bucket,name,state,workflow` in dir (runGH, bounded by
// timeout). gh pr checks exits 8 while a check is pending and may exit
// non-zero in other cases with the JSON on stdout, so stdout is read whatever
// the exit; only ctx done, a timeout, a gh not found, and a stdout that is not
// a JSON list leave the checks unread, except that a gh whose error has
// mergeNoChecksSentence for branch, the PR's head branch, reported no checks.
func readPRChecks(ctx context.Context, dir string, run MergeOrderExec, timeout time.Duration, repoArg string, number int, branch string) prChecks {
	n := strconv.Itoa(number)
	c := prChecks{cmd: "gh pr checks " + n + " -R " + repoArg}
	out, timedOut, err := runGH(ctx, dir, run, timeout, "pr", "checks", n, "-R", repoArg, "--json", mergePRChecksFields)
	c.stdout = out
	var checks []ghPRCheck
	listed := strings.HasPrefix(strings.TrimSpace(out), "[") && json.Unmarshal([]byte(out), &checks) == nil
	switch {
	case ctx.Err() != nil:
		c.read, c.err = prChecksInterrupted, err
		return c
	case errors.Is(err, exec.ErrNotFound) || timedOut:
		c.read, c.err = prChecksGHFailed, err
		return c
	case err != nil && !listed && strings.Contains(err.Error(), fmt.Sprintf(mergeNoChecksSentence, branch)):
		checks = nil
	case err != nil && !listed:
		c.read, c.err = prChecksGHFailed, err
		return c
	case !listed:
		c.read = prChecksNotList
		return c
	}
	c.total = len(checks)
	for _, check := range checks {
		switch check.Bucket {
		case "pass":
			c.passed++
		case "skipping":
		case "pending":
			c.pending = append(c.pending, check.label())
			c.notPassed = append(c.notPassed, check.label())
		default:
			c.failed = append(c.failed, check.label())
			c.notPassed = append(c.notPassed, check.label())
		}
	}
	return c
}
