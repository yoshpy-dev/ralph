package org

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The ci regate for an origin that is an ssh alias (plan
// docs/plans/active/2026-11-19-org-ssh-alias-host.md): the fixtures of
// regate_ci_test.go, with ssh -G answered by mxExec (merge_fake_test.go).

// TestRegateCI_ResolvesAnSSHAliasOriginThroughExec covers AC5b of plan
// docs/plans/active/2026-11-19-org-ssh-alias-host.md for the ci regate: with
// no Exec, ssh -G runs through realMergeDetachedExec and git and gh through
// realMergeOrderExec (newRegateCI); with Exec set, ssh -G runs through it, once,
// in the feature worktree, so an alias origin whose ssh -G names github.com
// is read there: the regate records its gate_ci_passed with every gh call
// pinned to github.com/o/r and the PR's url on github.com taken as origin's.
func TestRegateCI_ResolvesAnSSHAliasOriginThroughExec(t *testing.T) {
	ci := newRegateCI(RegateParams{})
	if reflect.ValueOf(ci.run).Pointer() != reflect.ValueOf(realMergeOrderExec).Pointer() ||
		reflect.ValueOf(ci.sshRun).Pointer() != reflect.ValueOf(realMergeDetachedExec).Pointer() ||
		ci.timeout != defaultMergeOrderGHTimeout {
		t.Error("newRegateCI with no Exec: git and gh do not run through realMergeOrderExec, ssh -G not through " +
			"realMergeDetachedExec, or the timeout is not 15 s")
	}

	fx := newRCFixture(t)
	fx.exec.origin = "git@github.com.emu:o/r.git\n"
	fx.exec.ssh = &mgReply{out: "host github.com.emu\nhostname github.com\n"}
	start := time.Now()
	res, err := fx.regateCI(context.Background())
	end := time.Now()
	if err != nil || res.Outcome != RegateOutcomeGated || !res.Passed || res.PR != 7 || res.Refused != "" {
		t.Fatalf("regate = %#v, %v; want a gated pass of PR #7", res, err)
	}
	fx.assertKeys(t, rcReadKeys)
	ssh := fx.exec.sshSnapshot()
	if len(ssh) != 1 || !slices.Equal(ssh[0].args, []string{"-G", "github.com.emu"}) || ssh[0].dir != fx.worktree ||
		ssh[0].deadline.Before(start.Add(mergeSSHResolveTimeout)) || ssh[0].deadline.After(end.Add(mergeSSHResolveTimeout)) {
		t.Errorf("ssh calls = %+v; want one ssh -G github.com.emu in %s with a 5 s deadline", ssh, fx.worktree)
	}
	if last := lastEvent(t, fx.o); last.Event != EventGateCIPassed {
		t.Errorf("last event = %+v; want a gate_ci_passed", last)
	}
}

// TestRegateCI_RunsSSHInAProcessGroupOfItsOwnWithNoExec covers AC5b for the
// ci regate's real commands: with no Exec, the pre-check runs the real `git
// remote get-url origin` in the worktree and ssh -G through
// realMergeDetachedExec, here a fake ssh at the front of PATH that writes its
// pid and process group and names github.com for the alias; gh is not on
// PATH, so the refusal names the host the pre-check read.
func TestRegateCI_RunsSSHInAProcessGroupOfItsOwnWithNoExec(t *testing.T) {
	isolateGitEnv(t)
	root := resolved(t, t.TempDir())
	runGit(t, "init", "--quiet", root)
	runGit(t, "-C", root, "remote", "add", "origin", "git@github.com.emu:o/r.git")
	group := filepath.Join(t.TempDir(), "group")
	soFakeSSH(t, "echo \"$$ $(ps -o pgid= -p $$)\" > "+ShellWord(group)+"\necho 'hostname github.com'\n")
	tgt := gateTarget{root: root, branch: "feat/auth", base: gateBase{branch: "main"}}
	ci, refused, err := regateCIPreCheck(context.Background(), RegateParams{LookPath: mgNotFound}, tgt, regateWords("feat/auth"))
	if err != nil || ci != nil || !strings.Contains(refused, rcMissing+rcDone+rcInstall) {
		t.Fatalf("regateCIPreCheck = %v, %q, %v; want gh's refusal for github.com", ci, refused, err)
	}
	b, err := os.ReadFile(group)
	var pid, pgid int
	if _, scanErr := fmt.Sscan(string(b), &pid, &pgid); err != nil || scanErr != nil {
		t.Fatalf("the fake ssh wrote %q, %v (%v)", b, err, scanErr)
	}
	if test := syscall.Getpgrp(); pgid != pid || pgid == test {
		t.Errorf("ssh (pid %d) ran in the process group %d; want its own, %d (the test's is %d)", pid, pgid, pid, test)
	}
}
