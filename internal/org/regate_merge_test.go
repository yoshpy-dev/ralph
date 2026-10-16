package org

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Regate's merge path (regate_merge.go) at the org layer: the merge commit,
// its undo, and the push (AC4 to AC7 of plan
// docs/plans/active/2026-10-16-org-regate.md), with regate_test.go's
// fixture.

// TestRegate_MergesTheTipInAndPushes covers AC4: after origin's main moved on
// past the branch's gate, Regate makes D3's merge commit with the caller's
// environment (its committer and its locale, with only GIT_MERGE_AUTOEDIT=no
// over the caller's value) in a process group of git's own, runs the scripts
// on it, records a gate_passed of it via regate for the binding's org,
// pushes it to origin's feat/auth without force, with GIT_TERMINAL_PROMPT=0
// and the caller's locale, from ralph's process group, and merge-order then
// classifies the branch ready.
func TestRegate_MergesTheTipInAndPushes(t *testing.T) {
	fx := newRegateFixture(t)
	pre, oldTip := fx.tips(t)
	fx.appendEvents(t, moGate("auth", "feat/auth", pre, oldTip, "refs/remotes/origin/main", ""))
	tip := fx.pushToOrigin(t, "main")
	if class := fx.mergeClass(t); class != MergeClassRebase {
		t.Fatalf("merge-order class before the regate = %s; want rebase", class)
	}
	fx.setHooks(t, "pre-merge-commit", "env > \"$GATE_TEST_OUT/merge-env\"\n"+regateRecordPgid("merge-pgid"))
	writeGateFile(t, fx.origin, "hooks/pre-receive", "#!/bin/sh\nenv > \"$GATE_TEST_OUT/push-env\"\n"+regateRecordPgid("push-pgid"), 0o755)
	fx.env = append(fx.env, "GIT_COMMITTER_NAME=Carol", "GIT_COMMITTER_EMAIL=carol@example.com", "GIT_MERGE_AUTOEDIT=yes",
		"LC_ALL=ja_JP.UTF-8", "GIT_TERMINAL_PROMPT=1")

	res := fx.mustRegate(t)
	merge := gitOut(t, fx.worktree, "rev-parse", "HEAD")
	logPath := filepath.Join(fx.stateDir, gateLogDir, regateTestLog)
	want := RegateResult{Outcome: RegateOutcomeMerged, Passed: true, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth",
		Head: merge, Base: tip, BaseRef: "refs/remotes/origin/main", Merge: merge, PreMergeHead: pre, LogPath: logPath, Pushed: true}
	if !reflect.DeepEqual(res, want) {
		t.Fatalf("regate = %+v\nwant     %+v", res, want)
	}
	fx.assertMergeCommit(t, merge, pre, tip)
	if who := gitOut(t, fx.worktree, "log", "-1", "--format=%cn <%ce>, %an <%ae>", merge); who != "Carol <carol@example.com>, Regate Tester <regate@example.com>" {
		t.Errorf("merge commit by %q; want the caller's committer, and the repository's user as the author", who)
	}
	vars := strings.Split(fileOrAbsent(t, filepath.Join(fx.out, "merge-env")), "\n")
	for _, w := range []string{"GIT_MERGE_AUTOEDIT=no", "LC_ALL=ja_JP.UTF-8", "GIT_TERMINAL_PROMPT=1", "GATE_TEST_OUT=" + fx.out} {
		if !slices.Contains(vars, w) {
			t.Errorf("the merge's environment has no %s", w)
		}
	}
	for _, notW := range []string{"GIT_MERGE_AUTOEDIT=yes", "LC_ALL=C", "GIT_TERMINAL_PROMPT=0"} {
		if slices.Contains(vars, notW) {
			t.Errorf("the merge's environment has %s", notW)
		}
	}
	fx.assertRegateGitGroup(t, "merge-pgid", false)
	vars = strings.Split(fileOrAbsent(t, filepath.Join(fx.out, "push-env")), "\n")
	for _, w := range []string{"GIT_TERMINAL_PROMPT=0", "LC_ALL=ja_JP.UTF-8"} {
		if !slices.Contains(vars, w) {
			t.Errorf("the push's environment has no %s", w)
		}
	}
	for _, notW := range []string{"GIT_TERMINAL_PROMPT=1", "LC_ALL=C"} {
		if slices.Contains(vars, notW) {
			t.Errorf("the push's environment has %s", notW)
		}
	}
	fx.assertRegateGitGroup(t, "push-pgid", true)
	if calls := fx.calls(t); !slices.Equal(calls, []string{"static", "test"}) {
		t.Fatalf("scripts run = %v; want static then test", calls)
	}
	if log := fileOrAbsent(t, logPath); !strings.Contains(log, "# head: "+merge+"\n# base: refs/remotes/origin/main "+tip+"\n") {
		t.Errorf("the log does not name the merge commit and the tip:\n%s", log)
	}
	events := fx.gateEvents(t)
	wantDetails := "head=" + merge + " base=" + tip + " base_ref=refs/remotes/origin/main branch=feat/auth by=alice herdr_pane=unset " +
		"via=regate log=" + strconv.Quote(logPath)
	if len(events) != 2 || events[1].Event != EventGatePassed || events[1].OrgID != "auth" || events[1].SeatID != "" ||
		events[1].Details != wantDetails {
		t.Fatalf("gate events = %+v\nwant the seeded one, then a gate_passed of auth with Details %q", events, wantDetails)
	}
	if got := fx.originBranch(t); got != merge {
		t.Fatalf("origin's feat/auth = %s; want the merge commit %s", got, merge)
	}
	if got := gitOut(t, fx.worktree, "rev-parse", "refs/remotes/origin/feat/auth"); got != merge {
		t.Fatalf("refs/remotes/origin/feat/auth = %s; want the pushed %s", got, merge)
	}
	if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Fatalf("the worktree after the regate:\n%s", st)
	}
	if class := fx.mergeClass(t); class != MergeClassReady {
		t.Fatalf("merge-order class after the regate = %s; want ready", class)
	}
	fx.assertUnlocked(t)
}

// TestRegate_PointsTheRemoteTrackingRefAtThePush: with no remote.origin.fetch
// in the repository, the push leaves refs/remotes/origin/feat/auth where the
// fetch put it, and Regate moves it to the merge commit it pushed.
func TestRegate_PointsTheRemoteTrackingRefAtThePush(t *testing.T) {
	fx := newRegateFixture(t)
	pre, _ := fx.tips(t)
	fx.pushToOrigin(t, "main")
	runGit(t, "-C", fx.main, "config", "--unset-all", "remote.origin.fetch")
	res := fx.mustRegate(t)
	if !res.Pushed || res.Merge == "" || res.PreMergeHead != pre || fx.originBranch(t) != res.Merge {
		t.Fatalf("regate = %+v; want the merge commit pushed", res)
	}
	if got := gitOut(t, fx.worktree, "rev-parse", "refs/remotes/origin/feat/auth"); got != res.Merge {
		t.Fatalf("refs/remotes/origin/feat/auth = %s; want the pushed merge commit %s", got, res.Merge)
	}
}

// regateNoIdentityEnv is env with no name or email git could make a commit
// with: no GIT_AUTHOR_*, GIT_COMMITTER_* or EMAIL, HOME an empty directory,
// and no global or system git config (GIT_CONFIG_GLOBAL=/dev/null,
// GIT_CONFIG_NOSYSTEM=1). With user.useConfigOnly in the repository, git
// does not guess one from the host either.
func regateNoIdentityEnv(t *testing.T, env []string) []string {
	t.Helper()
	var out []string
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		switch {
		case strings.HasPrefix(name, "GIT_AUTHOR_"), strings.HasPrefix(name, "GIT_COMMITTER_"), name == "EMAIL", name == "HOME",
			name == "GIT_CONFIG_GLOBAL", name == "GIT_CONFIG_NOSYSTEM":
			continue
		}
		out = append(out, kv)
	}
	return append(out, "HOME="+t.TempDir(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
}

// TestRegate_RefusedWhenTheMergeCommitCannotBeMade covers AC5: when git makes
// no merge commit (a conflict, a pre-merge-commit hook that fails, a merge a
// signal ends before it changed the worktree) or cannot make one (no
// user.name or user.email to make it with, which Regate checks first),
// Regate undoes a merge left in progress, records nothing, runs no script,
// leaves HEAD, the tree, and origin as they were with no MERGE_HEAD, and
// refuses with the reason (the conflicting files for a conflict) and a next
// step.
func TestRegate_RefusedWhenTheMergeCommitCannotBeMade(t *testing.T) {
	mainTip := func(t *testing.T, fx *regateFixture) string {
		return gitOut(t, fx.origin, "rev-parse", "refs/heads/main")
	}
	cases := []struct {
		name    string
		setup   func(t *testing.T, fx *regateFixture) (env []string)
		want    func(t *testing.T, fx *regateFixture) []string
		notWant []string
	}{
		{name: "a conflict",
			setup: func(t *testing.T, fx *regateFixture) []string {
				fx.commitAndPush(t, fx.main, "README.md", "main's readme\n", "internal/auth/auth.go", "package auth // main\n")
				fx.commitAndPush(t, fx.worktree, "README.md", "the feature's readme\n", "internal/auth/auth.go", "package auth // feature\n")
				return fx.env
			},
			want: func(t *testing.T, fx *regateFixture) []string {
				return []string{"merging origin's tip " + mainTip(t, fx) + " of main into feat/auth in " + fx.worktree + " conflicts in " +
					"README.md, internal/auth/auth.go, and regate does not resolve conflicts (git merge --abort undid the merge it " +
					"left in progress), so nothing was recorded: merge it yourself (git -C " + fx.worktree + " merge " +
					"refs/remotes/origin/main), resolve the conflicts, commit, push, and run ralph org regate --branch feat/auth again"}
			}},
		// git diff --name-only quotes a non-ASCII name ("\303\244.txt") and
		// does not mark where a name with a space ends, unless -z.
		{name: "a conflict in files named with a space and a non-ASCII letter",
			setup: func(t *testing.T, fx *regateFixture) []string {
				fx.commitAndPush(t, fx.main, "a b.txt", "main's\n", "ä.txt", "main's\n")
				fx.commitAndPush(t, fx.worktree, "a b.txt", "the feature's\n", "ä.txt", "the feature's\n")
				return fx.env
			},
			want: func(t *testing.T, fx *regateFixture) []string {
				return []string{" into feat/auth in " + fx.worktree + " conflicts in a b.txt, ä.txt, and regate does not resolve conflicts"}
			},
			notWant: []string{`\303`, `"`}},
		{name: "a pre-merge-commit hook stops it",
			setup: func(t *testing.T, fx *regateFixture) []string {
				fx.pushToOrigin(t, "main")
				fx.setHooks(t, "pre-merge-commit", "echo 'the hook says no' >&2\nexit 1\n")
				return fx.env
			},
			want: func(t *testing.T, fx *regateFixture) []string {
				tip := mainTip(t, fx)
				return []string{"git merge --no-ff --no-edit -m 'Merge origin/main " + tip[:12] + " into feat/auth' " + tip + " in " +
					fx.worktree + " made no merge commit (exit 1) (git merge --abort undid the merge it left in progress), so " +
					"nothing was recorded: a hook such as pre-merge-commit may have stopped it; fix what git said and run ralph org " +
					"regate --branch feat/auth again; git said:\n  the hook says no"}
			}},
		// origin's main moved by an empty commit, so the merge changes no
		// file, and the hook kills git before it makes the merge commit.
		{name: "a signal ends it before it changed the worktree",
			setup: func(t *testing.T, fx *regateFixture) []string {
				fx.pushToOrigin(t, "main")
				fx.setHooks(t, "pre-merge-commit", "kill -KILL $PPID\n")
				return fx.env
			},
			want: func(t *testing.T, fx *regateFixture) []string {
				tip := mainTip(t, fx)
				return []string{"git merge --no-ff --no-edit -m 'Merge origin/main " + tip[:12] + " into feat/auth' " + tip + " in " +
					fx.worktree + " was ended by a signal (killed) before it made a merge commit, so nothing was recorded: run ralph " +
					"org regate --branch feat/auth again"}
			},
			notWant: []string{"git merge --abort undid"}},
		{name: "no user.name or user.email",
			setup: func(t *testing.T, fx *regateFixture) []string {
				fx.pushToOrigin(t, "main")
				runGit(t, "-C", fx.main, "config", "user.useConfigOnly", "true")
				runGit(t, "-C", fx.main, "config", "--unset", "user.name")
				runGit(t, "-C", fx.main, "config", "--unset", "user.email")
				return regateNoIdentityEnv(t, fx.env)
			},
			// git var says so in the caller's locale: Regate reads only its
			// exit code, so the merge runs with no LC_ALL=C.
			want: func(t *testing.T, fx *regateFixture) []string {
				return []string{"org: regate: git in " + fx.worktree + " has no user.name and user.email to make the merge commit of " +
					"origin's tip " + mainTip(t, fx) + " of main with (git var GIT_AUTHOR_IDENT failed), so regate merged nothing and " +
					"recorded nothing: set them (git -C " + fx.worktree + " config user.name <name> and git -C " + fx.worktree +
					" config user.email <email>, or in your global git config) and run ralph org regate --branch feat/auth again; " +
					"git said:\n", "auto-detection is disabled"}
			},
			notWant: []string{"git merge --abort undid", "git merge --no-ff"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			env := tc.setup(t, fx)
			before := fx.snapshot(t)
			res, err := fx.o.Regate(context.Background(), RegateParams{Branch: "feat/auth", Env: env})
			fx.assertRefused(t, res, err, before, tc.want(t, fx), tc.notWant)
			if fx.mergeInProgress(t) {
				t.Fatal("a merge is still in progress in the worktree after the refusal")
			}
		})
	}
}

// TestRegate_AMergeEndedByASignalSaysHowToPutTheWorktreeBack: a signal that
// ends git merge in its pre-merge-commit hook (here the hook kills it) leaves
// the merge's result staged with no merge in progress, which git merge
// --abort refuses and git reset --keep leaves in the files. Regate records
// nothing, runs no script, and returns an error that says git was ended by
// the signal, how the worktree is, and the git reset --merge that puts it
// back, which it then does. The merge ran in a process group of its own.
func TestRegate_AMergeEndedByASignalSaysHowToPutTheWorktreeBack(t *testing.T) {
	fx := newRegateFixture(t)
	pre, _ := fx.tips(t)
	fx.commitAndPush(t, fx.main, "README.md", "main's readme\n")
	tip := gitOut(t, fx.origin, "rev-parse", "refs/heads/main")
	fx.setHooks(t, "pre-merge-commit", regateRecordPgid("merge-pgid")+"kill -KILL $PPID\n")
	manifest := fileOrAbsent(t, fx.o.Manifest.Path())

	res, err := fx.regate(context.Background())
	want := "org: regate: git merge --no-ff --no-edit -m 'Merge origin/main " + tip[:12] + " into feat/auth' " + tip + " in " +
		fx.worktree + " was ended by a signal (killed), and the worktree is not as it was before: HEAD is " + pre + " (it was " +
		pre + "), and the worktree has 1 uncommitted change(s) or untracked file(s); nothing was recorded: check it with git -C " +
		fx.worktree + " status, put it back with git -C " + fx.worktree + " reset --merge " + pre + ", which also undoes a merge " +
		"in progress, and run ralph org regate --branch feat/auth again"
	if err == nil || err.Error() != want {
		t.Fatalf("regate = %+v, %v\nwant the error\n%s", res, err, want)
	}
	if !reflect.DeepEqual(res, RegateResult{}) {
		t.Errorf("regate = %+v with the error; want an empty result", res)
	}
	if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "M  README.md" || fx.mergeInProgress(t) {
		t.Fatalf("the worktree after the regate (merge in progress: %v):\n%s\nwant README.md staged and no merge in progress",
			fx.mergeInProgress(t), st)
	}
	if got := fileOrAbsent(t, fx.o.Manifest.Path()); got != manifest {
		t.Errorf("the regate changed the manifest:\n%s", got)
	}
	if calls := fx.calls(t); calls != nil {
		t.Errorf("scripts run = %v; want none", calls)
	}
	fx.assertRegateGitGroup(t, "merge-pgid", false)
	fx.assertUnlocked(t)

	runGit(t, "-C", fx.worktree, "reset", "--merge", pre)
	if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Fatalf("the worktree after the git reset --merge the error names:\n%s", st)
	}
	if head := gitOut(t, fx.worktree, "rev-parse", "HEAD"); head != pre {
		t.Fatalf("HEAD = %s after the git reset --merge; want %s", head, pre)
	}
}

// TestRegate_AMergeEndedAfterItsCommitNamesResetKeep: a signal that ends git
// merge in its post-merge hook, once the merge commit is made, leaves HEAD on
// that commit. Regate records nothing, runs no script, and returns an error
// that names git reset --keep, which stops rather than drop a change of the
// user's, not the git reset --merge it names while HEAD is where it was; that
// reset puts the worktree back.
func TestRegate_AMergeEndedAfterItsCommitNamesResetKeep(t *testing.T) {
	fx := newRegateFixture(t)
	pre, _ := fx.tips(t)
	fx.commitAndPush(t, fx.main, "README.md", "main's readme\n")
	tip := gitOut(t, fx.origin, "rev-parse", "refs/heads/main")
	fx.setHooks(t, "post-merge", "kill -KILL $PPID\n")
	manifest := fileOrAbsent(t, fx.o.Manifest.Path())

	res, err := fx.regate(context.Background())
	merge := gitOut(t, fx.worktree, "rev-parse", "HEAD")
	want := "org: regate: git merge --no-ff --no-edit -m 'Merge origin/main " + tip[:12] + " into feat/auth' " + tip + " in " +
		fx.worktree + " was ended by a signal (killed), and the worktree is not as it was before: HEAD is " + merge + " (it was " +
		pre + "), and the worktree has 0 uncommitted change(s) or untracked file(s); nothing was recorded: check it with git -C " +
		fx.worktree + " status, put it back with git -C " + fx.worktree + " reset --keep " + pre + ", which drops the commits made " +
		"on it and stops rather than drop a change of yours, and run ralph org regate --branch feat/auth again"
	if err == nil || err.Error() != want {
		t.Fatalf("regate = %+v, %v\nwant the error\n%s", res, err, want)
	}
	if parents := gitOut(t, fx.worktree, "rev-list", "--parents", "-n", "1", merge); parents != merge+" "+pre+" "+tip {
		t.Fatalf("HEAD %s has the parents %q; want the merge commit of %s and %s", merge, parents, pre, tip)
	}
	if !reflect.DeepEqual(res, RegateResult{}) {
		t.Errorf("regate = %+v with the error; want an empty result", res)
	}
	if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" || fx.mergeInProgress(t) {
		t.Fatalf("the worktree after the regate (merge in progress: %v):\n%s\nwant it clean", fx.mergeInProgress(t), st)
	}
	if got := fileOrAbsent(t, fx.o.Manifest.Path()); got != manifest {
		t.Errorf("the regate changed the manifest:\n%s", got)
	}
	if calls := fx.calls(t); calls != nil {
		t.Errorf("scripts run = %v; want none", calls)
	}
	fx.assertUnlocked(t)

	runGit(t, "-C", fx.worktree, "reset", "--keep", pre)
	if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Fatalf("the worktree after the git reset --keep the error names:\n%s", st)
	}
	if head := gitOut(t, fx.worktree, "rev-parse", "HEAD"); head != pre {
		t.Fatalf("HEAD = %s after the git reset --keep; want %s", head, pre)
	}
}

// TestRegate_AMergeCommitItCannotReadNamesResetKeep: when git merge made its
// merge commit (exit 0) but HEAD or the parents of HEAD cannot be read after
// it (here a post-merge hook points HEAD at a branch that does not exist, or
// moves the branch on to a commit whose parent is missing), Regate runs no
// script, records nothing, pushes nothing, and leaves the merge commit, and
// its error says that git made a merge commit, what cannot be read and why,
// and the git reset --keep to the HEAD before the merge that undoes it.
func TestRegate_AMergeCommitItCannotReadNamesResetKeep(t *testing.T) {
	cases := []struct {
		name     string
		hook     string
		mergeRef string // where the merge commit is after the hook
		unread   func(t *testing.T, fx *regateFixture) string
	}{
		{name: "HEAD", hook: "git symbolic-ref HEAD refs/heads/missing\n", mergeRef: "refs/heads/feat/auth",
			unread: func(t *testing.T, fx *regateFixture) string {
				return "HEAD cannot be read after it (org: regate: git rev-parse --verify HEAD^{commit} in " + fx.worktree + ": " +
					gitFailure(t, fx.worktree, "rev-parse", "--verify", "HEAD^{commit}") + ")"
			}},
		{name: "the parents of HEAD",
			hook: `c=$(printf 'tree %s\nparent ` + strings.Repeat("0123456789", 4) + `\nauthor t <t@example.com> 0 +0000\n` +
				`committer t <t@example.com> 0 +0000\n\nno parent\n' "$(git rev-parse 'HEAD^{tree}')" | ` +
				`git hash-object -t commit -w --stdin) && git update-ref HEAD "$c"` + "\n",
			mergeRef: "feat/auth@{1}",
			unread: func(t *testing.T, fx *regateFixture) string {
				head := gitOut(t, fx.worktree, "rev-parse", "HEAD")
				return "the parents of HEAD " + head + " cannot be read (org: regate: git rev-list --parents -n 1 " + head + " in " +
					fx.worktree + ": " + gitFailure(t, fx.worktree, "rev-list", "--parents", "-n", "1", head) + ")"
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			pre, _ := fx.tips(t)
			fx.commitAndPush(t, fx.main, "README.md", "main's readme\n")
			tip := gitOut(t, fx.origin, "rev-parse", "refs/heads/main")
			fx.setHooks(t, "post-merge", tc.hook)
			manifest := fileOrAbsent(t, fx.o.Manifest.Path())
			originRefs := regateGitState(fx.origin, "for-each-ref")

			res, err := fx.regate(context.Background())
			want := "org: regate: git merge --no-ff --no-edit -m 'Merge origin/main " + tip[:12] + " into feat/auth' " + tip + " in " +
				fx.worktree + " made a merge commit, but " + tc.unread(t, fx) + ", so regate ran nothing and recorded nothing: check " +
				"git -C " + fx.worktree + " log, and to undo the merge run git -C " + fx.worktree + " reset --keep " + pre
			if err == nil || err.Error() != want {
				t.Fatalf("regate = %+v, %v\nwant the error\n%s", res, err, want)
			}
			if !reflect.DeepEqual(res, RegateResult{}) {
				t.Errorf("regate = %+v with the error; want an empty result", res)
			}
			fx.assertMergeCommit(t, gitOut(t, fx.worktree, "rev-parse", tc.mergeRef), pre, tip)
			if got := fileOrAbsent(t, fx.o.Manifest.Path()); got != manifest {
				t.Errorf("the regate changed the manifest:\n%s", got)
			}
			if calls := fx.calls(t); calls != nil {
				t.Errorf("scripts run = %v; want none", calls)
			}
			if got := regateGitState(fx.origin, "for-each-ref"); got != originRefs {
				t.Errorf("origin's refs after the regate:\n%s\nwant them as before:\n%s", got, originRefs)
			}
			fx.assertUnlocked(t)
		})
	}
}

// gitFailure is how gateGitOutput's error ends for git args run in dir as
// gateGit runs them, which fail: "exit <code>: <git's stderr>".
func gitFailure(t *testing.T, dir string, args ...string) string {
	t.Helper()
	var stderr bytes.Buffer
	cmd := exec.Command("git", append([]string{"--no-optional-locks", "-C", dir}, args...)...)
	cmd.Stderr = &stderr
	var exitErr *exec.ExitError
	if err := cmd.Run(); !errors.As(err, &exitErr) {
		t.Fatalf("git -C %s %v: %v; want it to fail", dir, args, err)
	}
	return "exit " + strconv.Itoa(exitErr.ExitCode()) + ": " + strings.TrimSpace(stderr.String())
}

// TestRegate_AScriptFailureAfterTheMergeUndoesIt covers AC6's static_failed
// and test_failed: Regate records a gate_failed of its merge commit, undoes
// the merge commit with git reset --keep (HEAD back where it was, the merge
// commit only in the reflog), pushes nothing, and returns the log.
func TestRegate_AScriptFailureAfterTheMergeUndoesIt(t *testing.T) {
	for _, tc := range []struct {
		name, script, reason string
		calls                []string
	}{
		{"the static script fails", "static", GateReasonStaticFailed, []string{"static"}},
		{"the test script fails", "test", GateReasonTestFailed, []string{"static", "test"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			pre, _ := fx.tips(t)
			tip := fx.pushToOrigin(t, "main")
			fx.setExit(t, tc.script, 1)
			res := fx.mustRegate(t)
			merge := fx.undoneMerge(t)
			logPath := filepath.Join(fx.stateDir, gateLogDir, regateTestLog)
			want := RegateResult{Outcome: RegateOutcomeMerged, Reason: tc.reason, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth",
				Head: merge, Base: tip, BaseRef: "refs/remotes/origin/main", Merge: merge, PreMergeHead: pre, Undone: true, LogPath: logPath,
				Tail: gateStubTail(tc.script)}
			if !reflect.DeepEqual(res, want) {
				t.Fatalf("regate = %#v\nwant     %#v", res, want)
			}
			fx.assertMergeCommit(t, merge, pre, tip)
			if head := gitOut(t, fx.worktree, "rev-parse", "HEAD"); head != pre {
				t.Fatalf("HEAD = %s after the undo; want %s, where it was before the merge", head, pre)
			}
			if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
				t.Fatalf("the worktree after the undo:\n%s", st)
			}
			if calls := fx.calls(t); !slices.Equal(calls, tc.calls) {
				t.Fatalf("scripts run = %v; want %v", calls, tc.calls)
			}
			prefix := "reason=" + tc.reason + " head=" + merge + " base=" + tip + " "
			if events := fx.gateEvents(t); len(events) != 1 || events[0].Event != EventGateFailed || !strings.HasPrefix(events[0].Details, prefix) {
				t.Fatalf("gate events = %+v; want one gate_failed starting %q", events, prefix)
			}
			if got := fx.originBranch(t); got != pre {
				t.Fatalf("origin's feat/auth = %s; want it not pushed, %s", got, pre)
			}
			if class := fx.mergeClass(t); class != MergeClassFailed {
				t.Fatalf("merge-order class after the regate = %s; want failed", class)
			}
			fx.assertUnlocked(t)
		})
	}
}

// TestRegate_AMergeCommitIsKeptWhenUndoingItCouldLoseWork covers AC6's
// head_moved and tree_changed, and the undo's own checks after a script
// failure: a gate_failed of the merge commit is recorded and nothing is
// pushed, but the merge commit is kept, and NotUndone says how to look at
// the worktree and which git reset --keep undoes it, when a commit was made
// after it, the tree is dirty, or git reset --keep fails. After tree_changed
// it also says how to set the scripts' changes aside when the reset stops on
// them.
func TestRegate_AMergeCommitIsKeptWhenUndoingItCouldLoseWork(t *testing.T) {
	commit := "git -c user.name=t -c user.email=t@example.com -c commit.gpgsign=false commit --quiet --allow-empty -m moved\n"
	cases := []struct {
		name         string
		script, hook string // the stub whose hook is set, and the hook
		fails        bool   // the stub exits 1
		commits      bool   // the hook makes a commit on top of the merge commit
		reason       string
		tail, dirty  []string
		why          func(head, pre string) string
		aside        bool   // NotUndone ends with how to set the changes aside
		said         string // in git's stderr after NotUndone's text
	}{
		{name: "a commit while the scripts run", script: "test", hook: commit, commits: true, reason: GateReasonHeadMoved,
			why: func(head, _ string) string {
				return "HEAD moved off it to " + head + " while the scripts ran (head_moved), and git reset --keep would drop what came after it"
			}},
		{name: "a file the scripts leave", script: "static", hook: "echo x > untracked.txt\n", reason: GateReasonTreeChanged,
			dirty: []string{"?? untracked.txt"}, why: func(string, string) string {
				return "the scripts left 1 uncommitted change(s) or untracked file(s) in the worktree (tree_changed), which git reset " +
					"--keep stops on or carries along"
			}, aside: true},
		{name: "a failing script that commits", script: "static", hook: commit, fails: true, commits: true,
			reason: GateReasonStaticFailed, tail: gateStubTail("static"), why: func(head, _ string) string {
				return "HEAD is now " + head + ", not the merge commit"
			}},
		{name: "a failing script that leaves a file", script: "static", hook: "echo x > untracked.txt\n", fails: true,
			reason: GateReasonStaticFailed, tail: gateStubTail("static"), why: func(string, string) string {
				return "the worktree has 1 uncommitted change(s) or untracked file(s)"
			}},
		// The index lock a failing script leaves stops git reset --keep, but
		// neither the HEAD nor the status the undo reads first. git's last
		// lines are its advice on a lock another git holds.
		{name: "git reset --keep fails", script: "static", hook: "touch \"$(git rev-parse --git-path index.lock)\"\n", fails: true,
			reason: GateReasonStaticFailed, tail: gateStubTail("static"), why: func(_, pre string) string {
				return "git reset --keep " + pre + " failed (exit 128)"
			}, said: "Another git process seems to be running in this repository"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			pre, _ := fx.tips(t)
			tip := fx.pushToOrigin(t, "main")
			fx.hook(t, tc.script, tc.hook)
			if tc.fails {
				fx.setExit(t, tc.script, 1)
			}
			res := fx.mustRegate(t)
			events := fx.gateEvents(t)
			if len(events) != 1 || events[0].Event != EventGateFailed {
				t.Fatalf("gate events = %+v; want one gate_failed", events)
			}
			r, _ := gateRecordFromDetails(events[0].Details)
			merge, head := r.Head, gitOut(t, fx.worktree, "rev-parse", "HEAD")
			fx.assertMergeCommit(t, merge, pre, tip)
			notUndone := regateMergeKeptText(fx.worktree, merge, pre, tc.why(head, pre))
			if tc.aside {
				notUndone += "; if git reset --keep stops on those changes, set them aside first (git -C " + fx.worktree + " stash -u, " +
					"or move the files out of the worktree)"
			}
			if tc.said == "" && res.NotUndone != notUndone ||
				tc.said != "" && (!strings.HasPrefix(res.NotUndone, notUndone+"; git said:\n  ") || !strings.Contains(res.NotUndone, tc.said)) {
				t.Fatalf("NotUndone = %q\nwant       %q (git saying %q)", res.NotUndone, notUndone, tc.said)
			}
			want := RegateResult{Outcome: RegateOutcomeMerged, Reason: tc.reason, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth",
				Head: merge, Base: tip, BaseRef: "refs/remotes/origin/main", Merge: merge, PreMergeHead: pre, NotUndone: res.NotUndone,
				LogPath: filepath.Join(fx.stateDir, gateLogDir, regateTestLog), Tail: tc.tail, Dirty: tc.dirty}
			if !reflect.DeepEqual(res, want) {
				t.Fatalf("regate = %#v\nwant     %#v", res, want)
			}
			if r.Reason != tc.reason {
				t.Fatalf("gate_failed reason = %q; want %q", r.Reason, tc.reason)
			}
			if tc.commits {
				if parent := gitOut(t, fx.worktree, "rev-parse", head+"^"); parent != merge {
					t.Fatalf("HEAD %s has the parent %s; want the hook's commit on top of the merge commit %s kept", head, parent, merge)
				}
			} else if head != merge {
				t.Fatalf("HEAD = %s; want the merge commit %s kept", head, merge)
			}
			if got := fx.originBranch(t); got != pre {
				t.Fatalf("origin's feat/auth = %s; want it not pushed, %s", got, pre)
			}
			fx.assertUnlocked(t)
		})
	}
}

// TestRegate_AnInterruptBeforeTheRecordUndoesTheMerge covers AC6's interrupt:
// a ctx done while the merge runs (which is not stopped: it finishes first),
// while a script runs, right before the record's lock, or under it right
// before the append records nothing, undoes the merge commit, pushes nothing,
// and returns an error that says so, naming the log once the scripts
// started. Stderr gets one line, that Regate waits for the merge, only when
// the interrupt comes while the merge runs: the undo that starts after an
// interrupt adds none.
func TestRegate_AnInterruptBeforeTheRecordUndoesTheMerge(t *testing.T) {
	type outcome struct {
		res RegateResult
		err error
	}
	setup := func(t *testing.T) (*regateFixture, string) {
		t.Helper()
		fx := newRegateFixture(t)
		pre, _ := fx.tips(t)
		fx.pushToOrigin(t, "main")
		return fx, pre
	}
	afterScripts := func(fx *regateFixture) string {
		return "org: regate: interrupted (context canceled): the scripts were stopped and nothing was recorded; the output so far is in " +
			filepath.Join(fx.stateDir, gateLogDir, regateTestLog) + "; "
	}
	check := func(t *testing.T, fx *regateFixture, pre, manifest string, got outcome, wantPrefix string) {
		t.Helper()
		merge := fx.undoneMerge(t)
		if want := wantPrefix + regateUndoneTextFor(merge, pre); got.err == nil || got.err.Error() != want {
			t.Fatalf("regate = %v\nwant     %s", got.err, want)
		}
		if !got.res.Undone || got.res.Merge != merge || got.res.PreMergeHead != pre || got.res.Pushed {
			t.Fatalf("regate = %+v; want the merge commit %s undone and nothing pushed", got.res, merge)
		}
		if after := fileOrAbsent(t, fx.o.Manifest.Path()); after != manifest {
			t.Fatalf("an interrupted regate changed the manifest:\nbefore %s\nafter  %s", manifest, after)
		}
		if head := gitOut(t, fx.worktree, "rev-parse", "HEAD"); head != pre {
			t.Fatalf("HEAD = %s; want %s, where it was before the merge", head, pre)
		}
		if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
			t.Fatalf("the worktree after the undo:\n%s", st)
		}
		if got := fx.originBranch(t); got != pre {
			t.Fatalf("origin's feat/auth = %s; want it not pushed, %s", got, pre)
		}
		fx.assertUnlocked(t)
	}
	regate := func(ctx context.Context, fx *regateFixture, stderr *bytes.Buffer) (RegateResult, error) {
		return fx.o.Regate(ctx, RegateParams{Branch: "feat/auth", Env: fx.env, Stderr: stderr})
	}
	start := func(ctx context.Context, fx *regateFixture, stderr *bytes.Buffer) <-chan outcome {
		done := make(chan outcome, 1)
		go func() {
			res, err := regate(ctx, fx, stderr)
			done <- outcome{res, err}
		}()
		return done
	}
	noLine := func(t *testing.T, stderr *bytes.Buffer) {
		t.Helper()
		if stderr.Len() != 0 {
			t.Errorf("stderr = %q; want nothing, since no git that is not stopped ran when the interrupt came", stderr)
		}
	}
	wait := func(t *testing.T, done <-chan outcome) outcome {
		t.Helper()
		select {
		case got := <-done:
			return got
		case <-time.After(10 * time.Second):
			t.Fatal("regate did not return within 10s of the interrupt")
		}
		return outcome{}
	}

	t.Run("while the merge runs", func(t *testing.T) {
		fx, pre := setup(t)
		started, release := filepath.Join(fx.out, "started"), filepath.Join(fx.out, "release")
		fx.setHooks(t, "pre-merge-commit", "touch '"+started+"'\nwhile [ ! -f '"+release+"' ]; do sleep 0.05; done\n")
		t.Cleanup(func() { _ = os.WriteFile(release, nil, 0o644) })
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var stderr bytes.Buffer
		done := start(ctx, fx, &stderr)
		waitForFile(t, started)
		cancel()
		select {
		case got := <-done:
			t.Fatalf("regate returned %+v, %v while its merge waited for the hook; want the merge left to finish", got.res, got.err)
		case <-time.After(300 * time.Millisecond):
		}
		if err := os.WriteFile(release, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		check(t, fx, pre, manifest, wait(t, done), "org: regate: interrupted (context canceled) before the scripts ran: nothing was recorded; ")
		if calls := fx.calls(t); calls != nil {
			t.Fatalf("scripts run = %v; want none", calls)
		}
		want := "org: regate: interrupted (context canceled): waiting for git merge --no-ff in " + fx.worktree + " to finish (at most " +
			regateGitTimeout.String() + ") before it stops, since a git stopped half-way would leave the worktree in neither state\n"
		if got := stderr.String(); got != want {
			t.Errorf("stderr = %q\nwant      %q", got, want)
		}
	})

	t.Run("while a script runs", func(t *testing.T) {
		grace := gateKillGrace
		gateKillGrace = 300 * time.Millisecond
		t.Cleanup(func() { gateKillGrace = grace })
		fx, pre := setup(t)
		started, release := filepath.Join(fx.out, "started"), filepath.Join(fx.out, "release")
		fx.hook(t, "static", "touch '"+started+"'\nwhile [ ! -f '"+release+"' ]; do sleep 0.05; done\n")
		t.Cleanup(func() { _ = os.WriteFile(release, nil, 0o644) })
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var stderr bytes.Buffer
		done := start(ctx, fx, &stderr)
		waitForFile(t, started)
		cancel()
		check(t, fx, pre, manifest, wait(t, done), afterScripts(fx))
		noLine(t, &stderr)
	})

	t.Run("right before the record's lock", func(t *testing.T) {
		fx, pre := setup(t)
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		setRegateHook(t, &regateTestHooks.beforeRecord, cancel)
		var stderr bytes.Buffer
		res, err := regate(ctx, fx, &stderr)
		check(t, fx, pre, manifest, outcome{res, err}, afterScripts(fx))
		noLine(t, &stderr)
	})

	t.Run("under the record's lock, right before the append", func(t *testing.T) {
		fx, pre := setup(t)
		manifest := fileOrAbsent(t, fx.o.Manifest.Path())
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		reads := 0
		fx.o.Now = func() time.Time {
			if reads++; reads >= 2 {
				cancel()
			}
			return regateTestClock
		}
		var stderr bytes.Buffer
		res, err := regate(ctx, fx, &stderr)
		check(t, fx, pre, manifest, outcome{res, err}, afterScripts(fx))
		noLine(t, &stderr)
	})
}

// TestRegate_ARefusedRecordAfterTheMergeUndoesIt covers AC8 after a merge:
// when, while the scripts run, the org's leader spawns again or another org
// binds the branch, Regate records nothing, undoes the merge commit, pushes
// nothing, and refuses, saying the merge commit was undone.
func TestRegate_ARefusedRecordAfterTheMergeUndoesIt(t *testing.T) {
	for _, tc := range []struct {
		name string
		act  func(t *testing.T, fx *regateFixture)
		want string
	}{
		{"the org's leader spawns again", func(t *testing.T, fx *regateFixture) {
			fx.appendEvents(t, ManifestEvent{TS: fx.o.now(), OrgID: "auth", SeatID: LeaderIdentity, Event: EventSpawnStarted,
				Role: LeaderIdentity, Driver: "claude", Model: "sonnet"})
		}, `(org_id "auth" is running again)`},
		{"another org binds the branch", func(t *testing.T, fx *regateFixture) { fx.rebind(t, fx.worktree) },
			`(it is now bound to org_id "auth-2"'s feature "auth"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			pre, _ := fx.tips(t)
			fx.pushToOrigin(t, "main")
			setRegateHook(t, &regateTestHooks.beforeRecord, func() { tc.act(t, fx) })
			res, err := fx.regate(context.Background())
			merge := fx.undoneMerge(t)
			if err != nil || res.Outcome != "" || res.Passed || res.Pushed ||
				!strings.HasPrefix(res.Refused, "org: regate: nothing was recorded, since the branch feat/auth changed while the scripts ran "+tc.want) ||
				!strings.HasSuffix(res.Refused, "run ralph org regate --branch feat/auth again once its org is disbanded; "+regateUndoneTextFor(merge, pre)) {
				t.Fatalf("regate = %+v, %v; want the record refused with %q and the merge commit %s undone", res, err, tc.want, merge)
			}
			if !res.Undone || res.Merge != merge || res.PreMergeHead != pre || res.NotUndone != "" {
				t.Fatalf("regate = %+v; want the merge commit %s undone", res, merge)
			}
			if events := fx.gateEvents(t); len(events) != 0 {
				t.Fatalf("gate events = %+v; want none", events)
			}
			if head := gitOut(t, fx.worktree, "rev-parse", "HEAD"); head != pre {
				t.Fatalf("HEAD = %s; want %s, where it was before the merge", head, pre)
			}
			if got := fx.originBranch(t); got != pre {
				t.Fatalf("origin's feat/auth = %s; want it not pushed, %s", got, pre)
			}
			fx.assertUnlocked(t)
		})
	}
}

// TestRegate_APushThatFailsKeepsTheRecordAndTheMerge covers AC7's first half
// and D4's retry: a push origin refuses keeps the gate_passed record and the
// merge commit and says to run Regate again; run again once origin takes
// pushes, Regate pushes the same merge commit and does nothing else, and a
// third run finds the branch up to date.
func TestRegate_APushThatFailsKeepsTheRecordAndTheMerge(t *testing.T) {
	fx := newRegateFixture(t)
	pre, _ := fx.tips(t)
	tip := fx.pushToOrigin(t, "main")
	allow := fx.rejectPushes(t)
	res := fx.mustRegate(t)
	merge := gitOut(t, fx.worktree, "rev-parse", "HEAD")
	logPath := filepath.Join(fx.stateDir, gateLogDir, regateTestLog)
	want := RegateResult{Outcome: RegateOutcomeMerged, Passed: true, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth",
		Head: merge, Base: tip, BaseRef: "refs/remotes/origin/main", Merge: merge, PreMergeHead: pre, LogPath: logPath,
		PushFailed: res.PushFailed}
	if !reflect.DeepEqual(res, want) {
		t.Fatalf("regate = %+v\nwant     %+v", res, want)
	}
	for _, w := range []string{"org: regate: git push origin " + merge + ":refs/heads/feat/auth in " + fx.worktree + " failed (exit 1), " +
		"so origin's feat/auth is not " + merge + ": the gate_passed record of " + merge + " and HEAD are kept",
		"run ralph org regate --branch feat/auth again, which pushes only, or says what changed; git said:\n", "pushes are closed"} {
		if !strings.Contains(res.PushFailed, w) {
			t.Errorf("PushFailed %q does not say %q", res.PushFailed, w)
		}
	}
	fx.assertMergeCommit(t, merge, pre, tip)
	if got := fx.originBranch(t); got != pre {
		t.Fatalf("origin's feat/auth = %s after the refused push; want %s", got, pre)
	}
	if events := fx.gateEvents(t); len(events) != 1 || events[0].Event != EventGatePassed {
		t.Fatalf("gate events = %+v; want the gate_passed kept", events)
	}

	allow()
	res = fx.mustRegate(t)
	want = RegateResult{Outcome: RegateOutcomePushOnly, Passed: true, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth",
		Head: merge, Base: tip, BaseRef: "refs/remotes/origin/main", LogPath: logPath, Pushed: true}
	if !reflect.DeepEqual(res, want) {
		t.Fatalf("regate after origin takes pushes = %+v\nwant     %+v", res, want)
	}
	if calls, events := fx.calls(t), fx.gateEvents(t); !slices.Equal(calls, []string{"static", "test"}) || len(events) != 1 {
		t.Fatalf("the retry ran %v and left the gate events %+v; want nothing run and nothing recorded again", calls, events)
	}
	if head, got := gitOut(t, fx.worktree, "rev-parse", "HEAD"), fx.originBranch(t); head != merge || got != merge {
		t.Fatalf("HEAD = %s, origin's feat/auth = %s after the retry; want both the merge commit %s", head, got, merge)
	}
	if class := fx.mergeClass(t); class != MergeClassReady {
		t.Fatalf("merge-order class after the retry = %s; want ready", class)
	}
	if res = fx.mustRegate(t); res.Outcome != RegateOutcomeUpToDate || res.Pushed {
		t.Fatalf("a third regate = %+v; want it up to date", res)
	}
	fx.assertUnlocked(t)
}

// TestRegate_APushThatLandedWhoseAnswerWasLost covers AC7's lost answer: when
// origin took the push but Regate never heard (here the test pushes the merge
// commit itself after a push that failed), running Regate again fetches it
// and finds the branch up to date (AC3).
func TestRegate_APushThatLandedWhoseAnswerWasLost(t *testing.T) {
	fx := newRegateFixture(t)
	fx.pushToOrigin(t, "main")
	allow := fx.rejectPushes(t)
	first := fx.mustRegate(t)
	if first.PushFailed == "" || first.Merge == "" {
		t.Fatalf("regate = %+v; want a merge commit whose push failed", first)
	}
	allow()
	runGit(t, "-C", fx.worktree, "push", "--quiet", "origin", "HEAD:refs/heads/feat/auth")
	res := fx.mustRegate(t)
	if res.Outcome != RegateOutcomeUpToDate || res.Head != first.Merge || res.Pushed || res.Merge != "" {
		t.Fatalf("regate after the push landed = %+v; want it up to date on %s", res, first.Merge)
	}
	if calls, events := fx.calls(t), fx.gateEvents(t); !slices.Equal(calls, []string{"static", "test"}) || len(events) != 1 {
		t.Fatalf("the second regate ran %v and left the gate events %+v; want nothing run or recorded", calls, events)
	}
}

// TestRegate_AFailedPushIsNotRetriedPastWhatChanged covers AC7's second half
// (D4): after a push that failed, running Regate again refuses when origin's
// feat/auth moved (someone pushed between the fetch and the push), with no
// reset hint, since HEAD's first parent is no longer origin's tip of the
// branch; and when origin's main moved, with the git reset --keep that drops
// the merge commit nobody pushed, after which Regate merges anew.
func TestRegate_AFailedPushIsNotRetriedPastWhatChanged(t *testing.T) {
	t.Run("someone pushes the branch between the fetch and the push", func(t *testing.T) {
		fx := newRegateFixture(t)
		fx.pushToOrigin(t, "main")
		var pushed string
		setRegateHook(t, &regateTestHooks.beforeRecord, func() { pushed = fx.pushToOrigin(t, "feat/auth") })
		first := fx.mustRegate(t)
		if first.Pushed || !first.Passed || !strings.Contains(first.PushFailed, "failed (exit 1), so origin's feat/auth is not "+first.Merge) ||
			!strings.Contains(first.PushFailed, "rejected") {
			t.Fatalf("regate = %+v; want a recorded pass whose push origin rejected", first)
		}
		if events := fx.gateEvents(t); len(events) != 1 || events[0].Event != EventGatePassed {
			t.Fatalf("gate events = %+v; want the gate_passed kept", events)
		}
		if got := fx.originBranch(t); got != pushed {
			t.Fatalf("origin's feat/auth = %s; want what the other push left, %s", got, pushed)
		}
		regateTestHooks.beforeRecord = nil
		callsBefore, before := fx.calls(t), fx.snapshot(t)
		res, err := fx.regate(context.Background())
		fx.assertRefusedAgain(t, res, err, before, callsBefore, []string{"HEAD " + first.Merge + " of feat/auth in " + fx.worktree +
			" and origin's tip " + pushed + " of the branch have diverged"}, []string{"reset --keep"})
	})

	t.Run("origin's main moves on", func(t *testing.T) {
		fx := newRegateFixture(t)
		pre, _ := fx.tips(t)
		fx.pushToOrigin(t, "main")
		allow := fx.rejectPushes(t)
		first := fx.mustRegate(t)
		if first.PushFailed == "" || first.Merge == "" {
			t.Fatalf("regate = %+v; want a merge commit whose push failed", first)
		}
		allow()
		tip := fx.pushToOrigin(t, "main")
		callsBefore, before := fx.calls(t), fx.snapshot(t)
		res, err := fx.regate(context.Background())
		fx.assertRefusedAgain(t, res, err, before, callsBefore, []string{"HEAD " + first.Merge + " of feat/auth in " + fx.worktree +
			" has commits that origin's tip " + pre + " of the branch does not", "and run ralph org regate --branch feat/auth again; or, " +
			"since HEAD is the merge commit an earlier regate made on origin's tip of the branch and did not push, drop it with git -C " +
			fx.worktree + " reset --keep refs/remotes/origin/feat/auth and run ralph org regate --branch feat/auth again, which merges " +
			"origin's main anew"}, nil)

		runGit(t, "-C", fx.worktree, "reset", "--quiet", "--keep", "refs/remotes/origin/feat/auth")
		res = fx.mustRegate(t)
		if res.Outcome != RegateOutcomeMerged || !res.Passed || !res.Pushed || res.PreMergeHead != pre || res.Base != tip {
			t.Fatalf("regate after the reset = %+v; want a merge of %s on %s, pushed", res, tip, pre)
		}
		fx.assertMergeCommit(t, res.Merge, pre, tip)
		if got := fx.originBranch(t); got != res.Merge {
			t.Fatalf("origin's feat/auth = %s; want the new merge commit %s", got, res.Merge)
		}
		if events := fx.gateEvents(t); len(events) != 2 || events[1].Event != EventGatePassed {
			t.Fatalf("gate events = %+v; want two gate_passed", events)
		}
		if class := fx.mergeClass(t); class != MergeClassReady {
			t.Fatalf("merge-order class = %s; want ready", class)
		}
	})
}

// TestRegate_AnInterruptAfterTheRecordKeepsItAndTheMerge covers AC7's
// interrupt: a ctx done after the gate_passed was appended, before the push,
// keeps the record and the merge commit, pushes nothing, and returns an
// error that says to run Regate again, which then pushes only.
func TestRegate_AnInterruptAfterTheRecordKeepsItAndTheMerge(t *testing.T) {
	fx := newRegateFixture(t)
	pre, _ := fx.tips(t)
	tip := fx.pushToOrigin(t, "main")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setRegateHook(t, &regateTestHooks.afterRecord, cancel)
	res, err := fx.regate(ctx)
	merge := gitOut(t, fx.worktree, "rev-parse", "HEAD")
	if want := "org: regate: interrupted (context canceled) before " + merge + " was pushed to origin's feat/auth, so nothing was " +
		"pushed: the gate_passed record of it and HEAD are kept; run ralph org regate --branch feat/auth again, which pushes it, or " +
		"finds it pushed"; err == nil || err.Error() != want {
		t.Fatalf("regate = %v\nwant     %s", err, want)
	}
	if res.Merge != merge || !res.Passed || res.Pushed || res.Undone || res.NotUndone != "" || res.PushFailed != "" {
		t.Fatalf("regate = %+v; want the recorded pass of the merge commit %s kept, not pushed", res, merge)
	}
	fx.assertMergeCommit(t, merge, pre, tip)
	if events := fx.gateEvents(t); len(events) != 1 || events[0].Event != EventGatePassed {
		t.Fatalf("gate events = %+v; want the gate_passed kept", events)
	}
	if got := fx.originBranch(t); got != pre {
		t.Fatalf("origin's feat/auth = %s; want it not pushed, %s", got, pre)
	}
	fx.assertUnlocked(t)

	regateTestHooks.afterRecord = nil
	if res = fx.mustRegate(t); res.Outcome != RegateOutcomePushOnly || !res.Pushed || res.Head != merge || fx.originBranch(t) != merge {
		t.Fatalf("regate run again = %+v; want the merge commit %s pushed only", res, merge)
	}
}

// TestRegate_APushEndedByASignalKeepsTheRecordAndTheMerge covers AC7 for a
// push a signal ends (here its pre-push hook kills git): as for an interrupt
// while the push runs, the record and the merge commit are kept, nothing is
// pushed, and the error says that git was ended by the signal, that origin
// may or may not have the merge commit, and to run Regate again, which then
// pushes only.
func TestRegate_APushEndedByASignalKeepsTheRecordAndTheMerge(t *testing.T) {
	fx := newRegateFixture(t)
	pre, _ := fx.tips(t)
	tip := fx.pushToOrigin(t, "main")
	fx.setHooks(t, "pre-push", "kill -KILL $PPID\n")
	res, err := fx.regate(context.Background())
	merge := gitOut(t, fx.worktree, "rev-parse", "HEAD")
	if want := "org: regate: git push origin " + merge + ":refs/heads/feat/auth in " + fx.worktree + " was ended by a signal " +
		"(killed) while " + merge + " was pushed to origin's feat/auth, so origin may or may not have it: the gate_passed record " +
		"of it and HEAD are kept; run ralph org regate --branch feat/auth again, which pushes it, or finds it pushed"; err == nil ||
		err.Error() != want {
		t.Fatalf("regate = %v\nwant     %s", err, want)
	}
	if res.Outcome != RegateOutcomeMerged || res.Merge != merge || !res.Passed || res.Pushed || res.Undone || res.NotUndone != "" ||
		res.PushFailed != "" {
		t.Fatalf("regate = %+v; want the recorded pass of the merge commit %s kept, not pushed", res, merge)
	}
	fx.assertMergeCommit(t, merge, pre, tip)
	if events := fx.gateEvents(t); len(events) != 1 || events[0].Event != EventGatePassed {
		t.Fatalf("gate events = %+v; want the gate_passed kept", events)
	}
	if got := fx.originBranch(t); got != pre {
		t.Fatalf("origin's feat/auth = %s; want it not pushed, %s", got, pre)
	}
	fx.assertUnlocked(t)

	if err := os.Remove(filepath.Join(fx.out, "hooks", "pre-push")); err != nil {
		t.Fatal(err)
	}
	if res = fx.mustRegate(t); res.Outcome != RegateOutcomePushOnly || !res.Pushed || res.Head != merge || fx.originBranch(t) != merge {
		t.Fatalf("regate run again = %+v; want the merge commit %s pushed only", res, merge)
	}
}

// TestRegate_PushesALeadersGatedHeadButNamesNoResetForIt covers D4 for a head
// that a gate of the running org passed and its leader did not push: its
// gate_passed on origin's main lets Regate push it (the retry asks for no
// via=regate), but once origin's main moved, the refusal names no git reset
// --keep, which would drop the leader's commit from the branch.
func TestRegate_PushesALeadersGatedHeadButNamesNoResetForIt(t *testing.T) {
	setup := func(t *testing.T) (fx *regateFixture, head, tip string) {
		t.Helper()
		fx = newRegateFixture(t)
		writeGateFile(t, fx.worktree, "internal/auth/more.go", "package auth\n", 0o644)
		runGit(t, "-C", fx.worktree, "add", ".")
		gateCommit(t, fx.worktree, "the leader's last commit")
		head, tip = fx.tips(t)
		fx.appendEvents(t, moGate("auth", "feat/auth", head, tip, "refs/remotes/origin/main", ""))
		return fx, head, tip
	}
	t.Run("origin's main as the gate found it", func(t *testing.T) {
		fx, head, tip := setup(t)
		res := fx.mustRegate(t)
		want := RegateResult{Outcome: RegateOutcomePushOnly, Passed: true, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth",
			Head: head, Base: tip, BaseRef: "refs/remotes/origin/main", LogPath: "/state/gates/auth.log", Pushed: true}
		if !reflect.DeepEqual(res, want) {
			t.Fatalf("regate = %+v\nwant     %+v", res, want)
		}
		if got := fx.originBranch(t); got != head || fx.calls(t) != nil || len(fx.gateEvents(t)) != 1 {
			t.Fatalf("origin's feat/auth = %s, scripts run %v; want %s pushed and nothing run", got, fx.calls(t), head)
		}
	})
	t.Run("after origin's main moved", func(t *testing.T) {
		fx, head, _ := setup(t)
		fx.pushToOrigin(t, "main")
		before := fx.snapshot(t)
		res, err := fx.regate(context.Background())
		fx.assertRefused(t, res, err, before, []string{"HEAD " + head + " of feat/auth in " + fx.worktree + " has commits that origin's tip"},
			[]string{"reset --keep"})
	})
}

// TestRegate_AnUnreadableLatestRecordGivesNoPushOnlyAndNoReset covers D2 for
// a HEAD that is not origin's tip: after a push that failed, a gate record
// that cannot be read or a corrupt line after the gate_passed may hide a
// newer record, so running Regate again neither pushes nor names the git
// reset --keep, and refuses.
func TestRegate_AnUnreadableLatestRecordGivesNoPushOnlyAndNoReset(t *testing.T) {
	for _, tc := range []struct {
		name string
		line any
	}{
		{"a gate record that cannot be read", moGarbled("auth", EventGatePassed)},
		{"a corrupt line", regateCorruptLine},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			pre, _ := fx.tips(t)
			fx.pushToOrigin(t, "main")
			allow := fx.rejectPushes(t)
			first := fx.mustRegate(t)
			if first.PushFailed == "" || first.Merge == "" {
				t.Fatalf("regate = %+v; want a merge commit whose push failed", first)
			}
			allow()
			fx.appendLines(t, tc.line)
			callsBefore, before := fx.calls(t), fx.snapshot(t)
			res, err := fx.regate(context.Background())
			fx.assertRefusedAgain(t, res, err, before, callsBefore, []string{"HEAD " + first.Merge + " of feat/auth in " + fx.worktree +
				" has commits that origin's tip " + pre + " of the branch does not"}, []string{"reset --keep"})
		})
	}
}
