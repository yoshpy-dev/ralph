package org

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

// The checks of `ralph org merge` for an origin that is an ssh alias (plan
// docs/plans/active/2026-11-19-org-ssh-alias-host.md): checkMerge on the
// fixtures of merge_test.go, with ssh -G answered by mxExec (merge_fake_test.go).

// TestCheckMerge_ReadsAnSSHAliasOriginOnTheHostSSHNames covers AC2 (the
// merge side), AC5b and AC6 of plan
// docs/plans/active/2026-11-19-org-ssh-alias-host.md: with origin an ssh
// alias whose ssh -G names github.com, checkMerge runs one ssh -G
// github.com.emu in /repo through Exec, apart from the commands of mxAll, and
// passes with every gh call pinned to github.com/o/r and --hostname
// github.com, the host CheckMergeEnforcement gets; the PR's url on github.com
// is origin's.
func TestCheckMerge_ReadsAnSSHAliasOriginOnTheHostSSHNames(t *testing.T) {
	fx := newMXFixture(t, nil)
	fx.exec.origin = "git@github.com.emu:o/r.git\n"
	fx.exec.ssh = &mgReply{out: "host github.com.emu\nuser git\nhostname github.com\nport 22\n"}
	start := time.Now()
	r, refused, err := fx.check(context.Background(), t, fx.params())
	end := time.Now()
	if err != nil || refused != "" {
		t.Fatalf("checkMerge = %v, refused %q", err, refused)
	}
	if r.apiHost != "github.com" || r.repoArg != mxRepoArg || r.originHost != "github.com.emu" || r.unresolvedWhy != "" ||
		r.prURL != "https://github.com/o/r/pull/7" {
		t.Errorf("mergeReady = %+v; want github.com/o/r read from the alias github.com.emu", r)
	}
	if got := fx.exec.keys(); !slices.Equal(got, mxAll) {
		t.Errorf("commands = %q\nwant %q", got, mxAll)
	}
	mxAssertPinned(t, fx.exec, "github.com", mxRepoArg)
	ssh := fx.exec.sshSnapshot()
	if len(ssh) != 1 || !slices.Equal(ssh[0].args, []string{"-G", "github.com.emu"}) || ssh[0].dir != "/repo" || ssh[0].env != nil ||
		ssh[0].deadline.Before(start.Add(mergeSSHResolveTimeout)) || ssh[0].deadline.After(end.Add(mergeSSHResolveTimeout)) {
		t.Errorf("ssh calls = %+v; want one ssh -G github.com.emu in /repo with no env and a 5 s deadline", ssh)
	}
}

// TestCheckMerge_ReadOriginRunsSSHThroughRunDetached covers AC5b for ralph
// org merge: readOrigin runs the get-url through the runner of the checks and
// ssh -G through runDetached, which newMergeCheck sets to
// realMergeDetachedExec for the real commands
// (TestMerge_CallsAfterGHPRMergeRunDetached).
func TestCheckMerge_ReadOriginRunsSSHThroughRunDetached(t *testing.T) {
	checks := &mxExec{origin: "git@github.com.emu:o/r.git\n"}
	detached := &mxExec{ssh: &mgReply{out: "hostname github.com\n"}}
	m := newMergeCheck(MergeParams{Branch: "feat/a", Dir: "/repo", Exec: checks.run, LookPath: mgFound})
	m.runDetached = detached.run
	var r mergeReady
	if refused, err := m.readOrigin(context.Background(), &r); err != nil || refused != "" || r.repoArg != mxRepoArg {
		t.Fatalf("readOrigin = %q, %v, %+v; want github.com/o/r", refused, err, r)
	}
	if got, ssh := checks.keys(), checks.sshSnapshot(); !slices.Equal(got, []string{mxGetURL}) || len(ssh) != 0 {
		t.Errorf("through the runner of the checks: %q and ssh %+v; want the get-url only", got, ssh)
	}
	if got, ssh := detached.keys(), detached.sshSnapshot(); len(got) != 0 || len(ssh) != 1 ||
		!slices.Equal(ssh[0].args, []string{"-G", "github.com.emu"}) {
		t.Errorf("through runDetached: %q and ssh %+v; want ssh -G github.com.emu only", got, ssh)
	}
}

// TestCheckMerge_GHRefusalNamesTheSSHKeyFixForAnUnresolvedHost covers AC5 of
// the plan for ralph org merge: when ssh -G did not resolve an ssh origin's
// host and gh read that host, the refusal of a gh that fails or is not on
// PATH adds why and the fix, git config core.sshCommand, before the command
// to run again; when ssh -G resolved it, when gh read github.com, and for an
// https origin, which runs no ssh, the refusal is as before.
func TestCheckMerge_GHRefusalNamesTheSSHKeyFixForAnUnresolvedHost(t *testing.T) {
	const alias = "git@github.com.emu:o/r.git\n"
	listFailed := func(repoArg string) string {
		return "org: merge: gh pr list --head feat/a -R " + repoArg + " failed (exit status 1: HTTP 404: Not Found), so the " +
			"branch's PR is unknown and nothing was merged: "
	}
	access := func(host string) string {
		return "check that gh can reach " + host + " and is logged in to it (gh auth status --hostname " + host + ")"
	}
	const hint = "; ssh -G github.com.emu failed (exit status 255: no such identity), so gh read the host github.com.emu as " +
		"origin names it: if github.com.emu is an ssh alias (a Host in ~/.ssh/config), point origin at the real host (git " +
		"remote set-url origin <URL>) and choose the ssh key for this repository with git config core.sshCommand 'ssh -i <key> " +
		"-o IdentitiesOnly=yes'"
	sshFails := &mgReply{err: errors.New("exit status 255: no such identity")}
	for _, tc := range []struct {
		name, origin string
		ssh          *mgReply
		lookPath     func(string) (string, error)
		want         string
	}{
		{"unresolved alias, gh pr list fails", alias, sshFails, mgFound,
			listFailed("github.com.emu/o/r") + access("github.com.emu") + hint + ", and " + mxAgain},
		{"unresolved alias, gh is not on PATH", alias, sshFails, mgNotFound, `org: merge: gh is not on PATH (exec: "gh": ` +
			"executable file not found in $PATH), and ralph org merge reads the PR and merges it with gh, so nothing was merged: " +
			"install gh, log in to github.com.emu (gh auth login --hostname github.com.emu)" + hint + ", and " + mxAgain},
		{"resolved alias, gh pr list fails", alias, &mgReply{out: "hostname github.com\n"}, mgFound, listFailed(mxRepoArg) + mxAccess},
		{"github.com origin whose ssh -G fails", moOriginURL + "\n", sshFails, mgFound, listFailed(mxRepoArg) + mxAccess},
		{"https origin", "https://github.com.emu/o/r.git\n", sshFails, mgFound,
			listFailed("github.com.emu/o/r") + access("github.com.emu") + ", and " + mxAgain},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMXFixture(t, nil)
			fx.exec.origin, fx.exec.ssh = tc.origin, tc.ssh
			fx.exec.replies[mxPRList] = mgReply{err: errors.New("exit status 1: HTTP 404: Not Found")}
			p := fx.params()
			p.LookPath = tc.lookPath
			r, refused, err := fx.check(context.Background(), t, p)
			if err != nil || refused != tc.want || r != (mergeReady{}) {
				t.Errorf("checkMerge = %+v, %v, refused\n%q\nwant\n%q", r, err, refused, tc.want)
			}
		})
	}
}
