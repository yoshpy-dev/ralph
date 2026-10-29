package org

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"syscall"
)

// How Regate (regate.go) reads origin and runs git with a bound: the fetches
// (regateFetch, D8) and the check of HEAD against origin's tip of the branch
// (regateCheckOriginHead, D4), the env of the fetches and the push
// (regateFetchEnv, regateRemoteEnv), and the runner every git command that
// may hang goes through (regateBoundedGit), in ralph's process group for one
// that talks to origin (regateRemoteGit) or in one of its own
// (regateUncancelledGit, regate_merge.go).

// regateFetch fetches origin's default branch t.base.branch and then the
// branch t.branch in t.root, each with its own refspec into
// refs/remotes/origin/ (D8), so that origin having no t.branch is told apart
// from every other failure, and returns the two tips it then reads there.
// The refusal says why a fetch failed: origin has no t.branch, the fetch ran
// past regateGitTimeout, or it failed another way, quoting the end of git's
// stderr. The error is for ctx done, a git that could not run, or a fetch a
// signal ended, which, like ctx done, records nothing.
func regateFetch(ctx context.Context, t gateTarget, env []string, w gateWording) (defaultTip, branchTip, refused string, err error) {
	var tips []string
	for _, name := range []string{t.base.branch, t.branch} {
		spec := "+refs/heads/" + name + ":refs/remotes/origin/" + name
		command := "git fetch origin " + spec
		r, timedOut, err := regateRemoteGit(ctx, t.root, regateFetchEnv(env), "fetch", "origin", spec)
		switch {
		case errors.As(err, new(regateGitSignaled)):
			return "", "", "", fmt.Errorf("%w, so nothing was recorded: %s", err, w.again)
		case err != nil:
			return "", "", "", err
		case timedOut:
			return "", "", w.prefix + fmt.Sprintf("%s in %s ran past %s and was stopped, so nothing was recorded: check the "+
				"network and origin's credentials and %s%s", command, t.root, regateGitTimeout, w.again, regateStderrTail(r.stderr)), nil
		case r.code != 0 && name == t.branch && strings.Contains(r.stderr, regateRemoteNoRef):
			return "", "", w.prefix + fmt.Sprintf("origin has no branch %s: it was deleted after the merge, or never pushed "+
				"(%s in %s failed), so nothing was recorded: when its PR is merged there is nothing to regate; otherwise push "+
				"it (git -C %s push origin %s) and %s", t.branch, command, t.root, t.root, t.branch, w.again), nil
		case r.code != 0:
			return "", "", w.prefix + fmt.Sprintf("%s in %s failed (exit %d), so nothing was recorded: check origin (git "+
				"remote get-url origin) and its credentials and %s%s", command, t.root, r.code, w.again, regateStderrTail(r.stderr)), nil
		}
		tip, err := gateGitOutput(ctx, w.prefix, t.root, "rev-parse", "--verify", "refs/remotes/origin/"+name+"^{commit}")
		if err != nil {
			return "", "", "", err
		}
		tips = append(tips, strings.TrimSpace(tip))
	}
	return tips[0], tips[1], "", nil
}

// regateStderrTail is the end of git's stderr for a refusal: "" when it
// printed nothing, else "; git said:" and its last regateStderrTailLines
// lines, each on a line of its own.
func regateStderrTail(stderr string) string {
	var lines []string
	for line := range strings.SplitSeq(stderr, "\n") {
		if line = strings.TrimRight(line, "\r \t"); line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "; git said:\n  " + strings.Join(lines[max(0, len(lines)-regateStderrTailLines):], "\n  ")
}

// regateCheckOriginHead refuses t when HEAD is not originHead, origin's tip
// of the branch after the fetch (D4): Regate gates only a head origin has, so
// it neither takes in commits that were not pushed nor goes past what
// someone pushed. The refusal says which side has commits the other does
// not, and how to make them the same.
func regateCheckOriginHead(ctx context.Context, t gateTarget, originHead string, w gateWording) (string, error) {
	if t.head == originHead {
		return "", nil
	}
	originRef := "refs/remotes/origin/" + t.branch
	headHasOrigin, err := regateIsAncestor(ctx, t.root, originHead, t.head)
	if err != nil {
		return "", err
	}
	originHasHead, err := regateIsAncestor(ctx, t.root, t.head, originHead)
	if err != nil {
		return "", err
	}
	switch {
	case headHasOrigin:
		return w.prefix + fmt.Sprintf("HEAD %s of %s in %s has commits that origin's tip %s of the branch does not: regate "+
			"gates only what origin has, so push them (git -C %s push origin %s) and %s",
			t.head, t.branch, t.root, originHead, t.root, t.branch, w.again), nil
	case originHasHead:
		return w.prefix + fmt.Sprintf("origin's tip %s of %s has commits that HEAD %s in %s does not (they were pushed from "+
			"elsewhere): bring the worktree up to date (git -C %s merge --ff-only %s) and %s",
			originHead, t.branch, t.head, t.root, t.root, originRef, w.again), nil
	}
	return w.prefix + fmt.Sprintf("HEAD %s of %s in %s and origin's tip %s of the branch have diverged: reconcile them in the "+
		"worktree, push the result, and %s", t.head, t.branch, t.root, originHead, w.again), nil
}

// regateIsAncestor reports whether commit a is an ancestor of commit b, or
// b itself, in root's repository (`git merge-base --is-ancestor`, exit 1
// for no).
func regateIsAncestor(ctx context.Context, root, a, b string) (bool, error) {
	r, err := gateGit(ctx, regatePrefix, root, "merge-base", "--is-ancestor", a, b)
	switch {
	case err != nil:
		return false, err
	case r.code == 0:
		return true, nil
	case r.code == 1:
		return false, nil
	}
	return false, fmt.Errorf("%sgit merge-base --is-ancestor %s %s in %s: exit %d: %s", regatePrefix, a, b, root, r.code, r.stderr)
}

// regateRemoteEnv is env, copied, with GIT_TERMINAL_PROMPT=0 after it, so it
// wins (D8): a fetch or the push asks for no credentials on the terminal.
// ssh's own prompt for a key is left as it is (regateRemoteGit).
func regateRemoteEnv(env []string) []string {
	return append(slices.Clip(env), "GIT_TERMINAL_PROMPT=0")
}

// regateFetchEnv is regateRemoteEnv's env with LC_ALL=C after it, so that
// git prints its messages untranslated, which regateFetch reads
// (regateRemoteNoRef). The push and the merge read none of git's text, so
// their hooks run in the caller's locale.
func regateFetchEnv(env []string) []string {
	return append(regateRemoteEnv(env), "LC_ALL=C")
}

// regateRemoteGit runs a git command that talks to origin, a fetch or the
// push (D8), with env (regateBoundedGit) in ralph's process group: ssh can
// then ask for a key on the terminal, and a Ctrl-C or a hangup of the
// terminal reaches git as it reaches ralph, which stops the fetch or the
// push the way an interrupt does (ctx done).
func regateRemoteGit(ctx context.Context, dir string, env []string, args ...string) (gitResult, bool, error) {
	return regateBoundedGit(ctx, dir, env, false, args...)
}

// regateGitSignaled is in the error regateBoundedGit returns for a git that
// a signal ended while ctx was live and before the bound: a kill from
// outside ralph, or, for a git in ralph's process group (regateRemoteGit),
// the terminal's Ctrl-C or hangup before ctx saw it. Its text says what
// became of git: "was ended by a signal (interrupt)".
type regateGitSignaled struct{ signal syscall.Signal }

func (e regateGitSignaled) Error() string {
	return fmt.Sprintf("was ended by a signal (%v)", e.signal)
}

// regateBoundedGit runs `git --no-optional-locks -C dir args...` with env as
// it is, its stdin empty, stopped after regateGitTimeout; timedOut reports
// that the bound stopped it. With ownGroup, git runs in a process group of
// its own, as the gate scripts do (runGateScript), so a signal the terminal
// sends its foreground group (Ctrl-C, a hangup) reaches ralph and not git,
// and the bound kills the whole group, git's children (a hook, ssh) with it.
// A hook that reads the terminal then stops (SIGTTIN) until the bound.
// WaitDelay closes the output pipes a child still holds after the kill or
// after git exited, so a child cannot hold the bound up; an exit 0 whose pipe
// a child held is a success, the way realMergeOrderExec reads it. The error
// is for a git that could not run, ctx done, and a git a signal ended
// (regateGitSignaled); a non-zero exit is not one. Release builds target
// darwin and linux only, where Setpgid and kill(-pgid) exist.
func regateBoundedGit(ctx context.Context, dir string, env []string, ownGroup bool, args ...string) (r gitResult, timedOut bool,
	err error) {
	return boundedGit(ctx, regatePrefix, dir, env, ownGroup, args...)
}

// boundedGit is regateBoundedGit with its errors starting with prefix, the
// verb that ran git: regatePrefix for Regate, cleanupPrefix for the checks of
// the cleanup of a merged feature worktree (cleanup.go).
func boundedGit(ctx context.Context, prefix, dir string, env []string, ownGroup bool, args ...string) (r gitResult, timedOut bool,
	err error) {
	gitCtx, cancel := context.WithTimeout(ctx, regateGitTimeout)
	defer cancel()
	cmd := exec.CommandContext(gitCtx, "git", append([]string{"--no-optional-locks", "-C", dir}, args...)...)
	cmd.Env = env
	if ownGroup {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Cancel = func() error {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			return nil
		}
	}
	cmd.WaitDelay = regateGitWaitDelay
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	r = gitResult{stdout: stdout.String(), stderr: strings.TrimSpace(stderr.String())}
	var exitErr *exec.ExitError
	var status syscall.WaitStatus
	if errors.As(runErr, &exitErr) {
		status, _ = exitErr.Sys().(syscall.WaitStatus)
	}
	switch {
	case ctx.Err() != nil:
		return r, false, fmt.Errorf("%sgit %s in %s: %w", prefix, strings.Join(args, " "), dir, context.Cause(ctx))
	case gitCtx.Err() != nil:
		return r, true, nil
	case runErr == nil, errors.Is(runErr, exec.ErrWaitDelay):
	case exitErr != nil && exitErr.ExitCode() > 0:
		r.code = exitErr.ExitCode()
	case exitErr != nil && status.Signaled():
		return r, false, fmt.Errorf("%sgit %s in %s %w", prefix, strings.Join(args, " "), dir, regateGitSignaled{status.Signal()})
	default:
		return r, false, fmt.Errorf("%sgit %s in %s: %w", prefix, strings.Join(args, " "), dir, runErr)
	}
	return r, false, nil
}
