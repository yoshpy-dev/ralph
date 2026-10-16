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

// TestRegate_KeepsARemoteTrackingRefSomethingMovedAfterThePush covers AC7's
// last sentence (D4): when refs/remotes/origin/feat/auth moves off where it
// was before the push, here by a fetch of a newer push of origin's feat/auth
// right after the push (which, with the default remote.origin.fetch, moved
// the ref to the merge commit) or, with no remote.origin.fetch, right before
// Regate's git update-ref of it, Regate leaves the ref at what the fetch
// brought instead of rewinding it to the merge commit (after a fetch right
// after the push, it runs no update-ref at all), and reports a pushed pass
// with no error. An update-ref that fails while the ref is still where it was
// before the push (here on a lock file another git left) is still an error.
func TestRegate_KeepsARemoteTrackingRefSomethingMovedAfterThePush(t *testing.T) {
	const ref = "refs/remotes/origin/feat/auth"
	for _, tc := range []struct {
		name          string
		hook          *func()
		noFetchConfig bool
	}{
		{"a fetch right after the push", &regateTestHooks.afterPush, false},
		{"a fetch right before the update-ref", &regateTestHooks.beforeTrackingUpdate, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			fx.pushToOrigin(t, "main")
			if tc.noFetchConfig {
				runGit(t, "-C", fx.main, "config", "--unset-all", "remote.origin.fetch")
			}
			var newer string
			setRegateHook(t, tc.hook, func() {
				newer = fx.pushToOrigin(t, "feat/auth")
				runGit(t, "-C", fx.worktree, "fetch", "--quiet", "origin", "+refs/heads/feat/auth:"+ref)
			})
			if tc.hook != &regateTestHooks.beforeTrackingUpdate {
				setRegateHook(t, &regateTestHooks.beforeTrackingUpdate, func() {
					t.Errorf("regate went on to git update-ref %s after the fetch moved it", ref)
				})
			}
			res, err := fx.regate(context.Background())
			if err != nil || res.Outcome != RegateOutcomeMerged || !res.Passed || !res.Pushed || res.Merge == "" || newer == "" {
				t.Fatalf("regate = %+v, %v; want a pushed pass of a merge commit, with no error, and the hook run", res, err)
			}
			if got := gitOut(t, fx.worktree, "rev-parse", ref); got != newer {
				t.Fatalf("%s = %s after the regate; want what the fetch brought, %s, not the pushed %s", ref, got, newer, res.Merge)
			}
		})
	}

	t.Run("an update-ref that fails with the ref where it was before the push", func(t *testing.T) {
		fx := newRegateFixture(t)
		pre, _ := fx.tips(t)
		fx.pushToOrigin(t, "main")
		runGit(t, "-C", fx.main, "config", "--unset-all", "remote.origin.fetch")
		common := gitOut(t, fx.worktree, "rev-parse", "--path-format=absolute", "--git-common-dir")
		setRegateHook(t, &regateTestHooks.beforeTrackingUpdate, func() { writeGateFile(t, common, ref+".lock", "", 0o644) })
		res, err := fx.regate(context.Background())
		if err == nil || !res.Pushed || res.Merge == "" || fx.originBranch(t) != res.Merge {
			t.Fatalf("regate = %+v, %v; want the merge commit pushed and an error for the update-ref", res, err)
		}
		if prefix := "org: regate: git update-ref " + ref + " " + res.Merge + " " + pre + " in " + fx.worktree + ": exit "; !strings.HasPrefix(err.Error(), prefix) {
			t.Errorf("error %q does not start with %q", err, prefix)
		}
		if suffix := "; " + res.Merge + " was pushed to origin's feat/auth, and git fetch origin feat/auth brings " + ref + " up to date"; !strings.HasSuffix(err.Error(), suffix) {
			t.Errorf("error %q does not end with %q", err, suffix)
		}
		if got := gitOut(t, fx.worktree, "rev-parse", ref); got != pre {
			t.Fatalf("%s = %s after the failed update-ref; want it where the fetch put it, %s", ref, got, pre)
		}
	})
}

// TestRegate_MergesADefaultBranchTipThatAlreadyContainsTheBranch covers the
// test plan's edge case of origin's main a descendant of HEAD, a branch with
// no commit main does not have: main took feat/auth's commit by a
// fast-forward and moved on. HEAD still does not contain the tip, so Regate
// makes D3's merge commit (--no-ff) on HEAD, whose tree is the tip's, runs
// the scripts on it, records a gate_passed of it via regate, pushes it to
// origin's feat/auth, and merge-order then classifies the branch ready.
func TestRegate_MergesADefaultBranchTipThatAlreadyContainsTheBranch(t *testing.T) {
	fx := newRegateFixture(t)
	pre, oldTip := fx.tips(t)
	fx.appendEvents(t, moGate("auth", "feat/auth", pre, oldTip, "refs/remotes/origin/main", ""))
	runGit(t, "-C", fx.main, "merge", "--quiet", "--ff-only", "feat/auth")
	tip := fx.commitAndPush(t, fx.main, "MAIN.md", "main moved on\n")
	if parent := gitOut(t, fx.origin, "rev-parse", tip+"^"); parent != pre {
		t.Fatalf("origin's main %s has the parent %s; want feat/auth's HEAD %s", tip, parent, pre)
	}
	if class := fx.mergeClass(t); class != MergeClassRebase {
		t.Fatalf("merge-order class before the regate = %s; want rebase", class)
	}

	res := fx.mustRegate(t)
	merge := gitOut(t, fx.worktree, "rev-parse", "HEAD")
	logPath := filepath.Join(fx.stateDir, gateLogDir, regateTestLog)
	want := RegateResult{Outcome: RegateOutcomeMerged, Passed: true, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth",
		Head: merge, Base: tip, BaseRef: "refs/remotes/origin/main", Merge: merge, PreMergeHead: pre, LogPath: logPath, Pushed: true}
	if !reflect.DeepEqual(res, want) {
		t.Fatalf("regate = %+v\nwant     %+v", res, want)
	}
	fx.assertMergeCommit(t, merge, pre, tip)
	if diff := gitOut(t, fx.worktree, "diff", tip, merge); diff != "" {
		t.Errorf("git diff %s %s:\n%s\nwant none: the merge commit's tree is origin's main", tip, merge, diff)
	}
	if files := gitOut(t, fx.worktree, "diff", "--name-only", pre, merge); files != "MAIN.md" {
		t.Errorf("git diff --name-only %s %s = %q; want MAIN.md, main's commit after the branch", pre, merge, files)
	}
	if calls := fx.calls(t); !slices.Equal(calls, []string{"static", "test"}) {
		t.Fatalf("scripts run = %v; want static then test", calls)
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
	if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Fatalf("the worktree after the regate:\n%s", st)
	}
	if class := fx.mergeClass(t); class != MergeClassReady {
		t.Fatalf("merge-order class after the regate = %s; want ready", class)
	}
	fx.assertUnlocked(t)
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

// TestRegate_AnotherCommitAfterTheMergeNamesResetKeep: when git merge made
// its merge commit (exit 0) but HEAD is then another commit (here a
// post-merge hook makes a commit on top of it and moves the branch there, so
// HEAD's only parent is the merge commit), Regate does not take HEAD for its
// merge commit: it runs no script, records nothing, pushes nothing, and
// leaves both commits, and its error names HEAD, its parents, the HEAD before
// the merge and origin's tip, and the git reset --keep to the HEAD before the
// merge, which drops both.
func TestRegate_AnotherCommitAfterTheMergeNamesResetKeep(t *testing.T) {
	fx := newRegateFixture(t)
	pre, _ := fx.tips(t)
	fx.commitAndPush(t, fx.main, "README.md", "main's readme\n")
	tip := gitOut(t, fx.origin, "rev-parse", "refs/heads/main")
	// git commit in the hook would read the MERGE_HEAD the merge still has
	// and make a second merge commit; git commit-tree reads none.
	fx.setHooks(t, "post-merge", `c=$(git -c user.name=t -c user.email=t@example.com commit-tree -p HEAD -m extra 'HEAD^{tree}') && `+
		`git update-ref HEAD "$c"`+"\n")
	manifest := fileOrAbsent(t, fx.o.Manifest.Path())
	originRefs := regateGitState(fx.origin, "for-each-ref")

	res, err := fx.regate(context.Background())
	head := gitOut(t, fx.worktree, "rev-parse", "HEAD")
	merge := gitOut(t, fx.worktree, "rev-parse", "HEAD^")
	want := "org: regate: git merge --no-ff --no-edit -m 'Merge origin/main " + tip[:12] + " into feat/auth' " + tip + " in " +
		fx.worktree + " left HEAD at " + head + `, whose parents are ["` + merge + `"], not HEAD ` + pre + " and origin's tip " + tip +
		" of main (a hook may have made another commit), so regate ran nothing and recorded nothing: check git -C " + fx.worktree +
		" log, and to undo the merge run git -C " + fx.worktree + " reset --keep " + pre
	if err == nil || err.Error() != want {
		t.Fatalf("regate = %+v, %v\nwant the error\n%s", res, err, want)
	}
	if !reflect.DeepEqual(res, RegateResult{}) {
		t.Errorf("regate = %+v with the error; want an empty result", res)
	}
	if parents := gitOut(t, fx.worktree, "rev-list", "--parents", "-n", "1", head); parents != head+" "+merge {
		t.Fatalf("HEAD %s has the parents %q; want only the merge commit %s", head, parents, merge)
	}
	if msg := gitOut(t, fx.worktree, "log", "-1", "--format=%B", head); msg != "extra" {
		t.Errorf("HEAD's message = %q; want the hook's commit, extra", msg)
	}
	fx.assertMergeCommit(t, merge, pre, tip)
	if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" || fx.mergeInProgress(t) {
		t.Fatalf("the worktree after the regate (merge in progress: %v):\n%s\nwant it clean", fx.mergeInProgress(t), st)
	}
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

	runGit(t, "-C", fx.worktree, "reset", "--keep", pre)
	if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
		t.Fatalf("the worktree after the git reset --keep the error names:\n%s", st)
	}
	if head := gitOut(t, fx.worktree, "rev-parse", "HEAD"); head != pre {
		t.Fatalf("HEAD = %s after the git reset --keep; want %s", head, pre)
	}
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
// feat/auth moved (someone pushed between the fetch and the push, or it was
// put back under the merge commit's first parent, a force push), pushing
// nothing and naming no reset hint, since HEAD's first parent is no longer
// origin's tip of the branch; and when origin's main moved, with the git
// reset --keep that drops the merge commit nobody pushed, after which Regate
// merges anew.
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

	t.Run("origin's branch is put back under the merge commit's first parent", func(t *testing.T) {
		fx := newRegateFixture(t)
		pre, _ := fx.tips(t)
		fx.pushToOrigin(t, "main")
		allow := fx.rejectPushes(t)
		first := fx.mustRegate(t)
		if first.PushFailed == "" || first.Merge == "" {
			t.Fatalf("regate = %+v; want a merge commit whose push failed", first)
		}
		allow()
		under := gitOut(t, fx.worktree, "rev-parse", pre+"^")
		runGit(t, "-C", fx.origin, "update-ref", "refs/heads/feat/auth", under)
		callsBefore, before := fx.calls(t), fx.snapshot(t)
		res, err := fx.regate(context.Background())
		fx.assertRefusedAgain(t, res, err, before, callsBefore, []string{"HEAD " + first.Merge + " of feat/auth in " + fx.worktree +
			" has commits that origin's tip " + under + " of the branch does not"}, []string{"reset --keep"})
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

// TestRegate_AGateOnlyHeadIsNotPushedAgainOnceOriginLeftIt covers D4 for the
// record of a gate-only Regate (AC9), which is via=regate too: it gated
// origin's tip of feat/auth, and then origin's feat/auth was put back to that
// head's first parent (a force push). Whether that gate passed or failed, and
// also when the head is a merge commit on that parent but not of origin's
// main the record names, HEAD is not the merge commit a Regate makes, so
// running Regate again refuses, pushes nothing (origin's feat/auth stays at
// the parent), records nothing, and names no git reset --keep, which would
// drop the commit from the branch.
func TestRegate_AGateOnlyHeadIsNotPushedAgainOnceOriginLeftIt(t *testing.T) {
	for _, tc := range []struct {
		name   string
		setup  func(t *testing.T, fx *regateFixture) // before the gate-only regate
		passed bool
	}{
		{"a pass on the branch's own commit", nil, true},
		{"a failure on the branch's own commit", func(t *testing.T, fx *regateFixture) { fx.setExit(t, "test", 1) }, false},
		{"a pass on a merge of another branch", func(t *testing.T, fx *regateFixture) {
			side := gitOut(t, fx.worktree, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit-tree", "-p", "HEAD~1", "-m", "side",
				"HEAD~1^{tree}")
			runGit(t, "-C", fx.worktree, "merge", "--quiet", "--no-ff", "--no-edit", side)
			runGit(t, "-C", fx.worktree, "push", "--quiet", "origin", "HEAD")
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			if tc.setup != nil {
				tc.setup(t, fx)
			}
			head, _ := fx.tips(t)
			if first := fx.mustRegate(t); first.Outcome != RegateOutcomeGated || first.Passed != tc.passed || first.Head != head {
				t.Fatalf("gate-only regate = %+v; want a record of %s that passed: %v", first, head, tc.passed)
			}
			parent := gitOut(t, fx.worktree, "rev-parse", head+"^")
			runGit(t, "-C", fx.origin, "update-ref", "refs/heads/feat/auth", parent)
			callsBefore, before := fx.calls(t), fx.snapshot(t)
			res, err := fx.regate(context.Background())
			fx.assertRefusedAgain(t, res, err, before, callsBefore, []string{"HEAD " + head + " of feat/auth in " + fx.worktree +
				" has commits that origin's tip " + parent + " of the branch does not"}, []string{"reset --keep"})
		})
	}
}

// TestRegate_APersonsMergeOfTheDefaultBranchIsNotTakenForRegates covers D4
// for a merge commit of origin's main into feat/auth that a person made, as
// the conflict refusal asks (git merge origin/main, commit, push) or as
// GitHub's Update branch makes one: its parents are those of the merge commit
// a Regate makes, origin's tip of feat/auth then and origin's main, but its
// subject is git's or GitHub's. Once a gate-only Regate passed or failed on it
// and origin's feat/auth was put back to its first parent (a force push),
// running Regate again refuses, pushes nothing, records nothing, and names no
// git reset --keep, which would drop the person's merge: with origin's main
// as the merge took it in (no push-only), and once origin's main moved on.
func TestRegate_APersonsMergeOfTheDefaultBranchIsNotTakenForRegates(t *testing.T) {
	for _, m := range []struct {
		name    string
		subject string
		merge   func(t *testing.T, fx *regateFixture) // after origin's main moved on
	}{
		{"git merge origin/main", "Merge remote-tracking branch 'origin/main' into feat/auth", func(t *testing.T, fx *regateFixture) {
			runGit(t, "-C", fx.worktree, "fetch", "--quiet", "origin")
			runGit(t, "-C", fx.worktree, "merge", "--quiet", "--no-ff", "--no-edit", "origin/main")
			runGit(t, "-C", fx.worktree, "push", "--quiet", "origin", "HEAD")
		}},
		{"GitHub's Update branch", "Merge branch 'main' into feat/auth", func(t *testing.T, fx *regateFixture) {
			// origin's new main commit (pushToOrigin) changes no file, so the
			// merge's tree is feat/auth's.
			merge := gitOut(t, fx.origin, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit-tree", "-p", "refs/heads/feat/auth",
				"-p", "refs/heads/main", "-m", "Merge branch 'main' into feat/auth", "refs/heads/feat/auth^{tree}")
			runGit(t, "-C", fx.origin, "update-ref", "refs/heads/feat/auth", merge)
			runGit(t, "-C", fx.worktree, "fetch", "--quiet", "origin")
			runGit(t, "-C", fx.worktree, "merge", "--quiet", "--ff-only", "origin/feat/auth")
		}},
	} {
		for _, passed := range []bool{true, false} {
			name := m.name + ", a pass"
			if !passed {
				name = m.name + ", a failure"
			}
			t.Run(name, func(t *testing.T) {
				fx := newRegateFixture(t)
				pre, _ := fx.tips(t)
				fx.pushToOrigin(t, "main")
				m.merge(t, fx)
				head, tip := fx.tips(t)
				if parents := strings.Fields(gitOut(t, fx.worktree, "rev-list", "--parents", "-n", "1", head))[1:]; !slices.Equal(parents,
					[]string{pre, tip}) {
					t.Fatalf("the person's merge %s has the parents %v; want %s, then origin's main %s", head, parents, pre, tip)
				}
				if subject := gitOut(t, fx.worktree, "log", "-1", "--format=%s", head); subject != m.subject {
					t.Fatalf("the person's merge %s has the subject %q; want %q", head, subject, m.subject)
				}
				if !passed {
					fx.setExit(t, "test", 1)
				}
				if first := fx.mustRegate(t); first.Outcome != RegateOutcomeGated || first.Passed != passed || first.Head != head ||
					first.Base != tip {
					t.Fatalf("gate-only regate = %+v; want a record of %s on %s that passed: %v", first, head, tip, passed)
				}
				runGit(t, "-C", fx.origin, "update-ref", "refs/heads/feat/auth", pre)
				refusal := "HEAD " + head + " of feat/auth in " + fx.worktree + " has commits that origin's tip " + pre + " of the branch does not"
				callsBefore, before := fx.calls(t), fx.snapshot(t)
				res, err := fx.regate(context.Background())
				fx.assertRefusedAgain(t, res, err, before, callsBefore, []string{refusal}, []string{"reset --keep"})

				fx.pushToOrigin(t, "main")
				callsBefore, before = fx.calls(t), fx.snapshot(t)
				res, err = fx.regate(context.Background())
				fx.assertRefusedAgain(t, res, err, before, callsBefore, []string{refusal}, []string{"reset --keep"})
			})
		}
	}
}

// TestRegate_AMergeCommitWhoseSubjectAHookChangedIsNotPushedOnly covers D4
// for a merge commit Regate made whose subject a prepare-commit-msg hook
// changed: its parents are D3's, but its subject is not the one Regate
// writes, so Regate cannot tell it from a person's merge. After its push
// failed, running Regate again refuses, pushes nothing, and names no git
// reset --keep (the safe side) instead of pushing it only.
func TestRegate_AMergeCommitWhoseSubjectAHookChangedIsNotPushedOnly(t *testing.T) {
	fx := newRegateFixture(t)
	pre, _ := fx.tips(t)
	tip := fx.pushToOrigin(t, "main")
	fx.setHooks(t, "prepare-commit-msg", `printf '[checked] %s\n' "$(cat "$1")" > "$1.new" && mv "$1.new" "$1"`+"\n")
	allow := fx.rejectPushes(t)
	first := fx.mustRegate(t)
	if first.Outcome != RegateOutcomeMerged || !first.Passed || first.PushFailed == "" || first.Merge == "" {
		t.Fatalf("regate = %+v; want a merge commit that passed and whose push failed", first)
	}
	if parents := strings.Fields(gitOut(t, fx.worktree, "rev-list", "--parents", "-n", "1", first.Merge))[1:]; !slices.Equal(parents,
		[]string{pre, tip}) {
		t.Fatalf("merge commit %s has the parents %v; want the branch's %s, then origin's main %s", first.Merge, parents, pre, tip)
	}
	if subject, want := gitOut(t, fx.worktree, "log", "-1", "--format=%s", first.Merge), "[checked] Merge origin/main "+tip[:12]+
		" into feat/auth"; subject != want {
		t.Fatalf("merge commit subject = %q; want the hook's %q", subject, want)
	}
	allow()
	callsBefore, before := fx.calls(t), fx.snapshot(t)
	res, err := fx.regate(context.Background())
	fx.assertRefusedAgain(t, res, err, before, callsBefore, []string{"HEAD " + first.Merge + " of feat/auth in " + fx.worktree +
		" has commits that origin's tip " + pre + " of the branch does not"}, []string{"reset --keep"})
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

// excludeLocally makes the main checkout and its worktrees ignore each
// pattern, in the repository's own .git/info/exclude: a local ignore that no
// commit carries.
func (fx *regateFixture) excludeLocally(t *testing.T, patterns ...string) {
	t.Helper()
	writeGateFile(t, fx.main, ".git/info/exclude", strings.Join(patterns, "\n")+"\n", 0o644)
}

// commitIgnoredAndPush is commitAndPush for paths dir ignores (git add -f):
// the branch checked out there starts tracking a path that is ignored.
func (fx *regateFixture) commitIgnoredAndPush(t *testing.T, dir string, files ...string) string {
	t.Helper()
	for i := 0; i+1 < len(files); i += 2 {
		writeGateFile(t, dir, files[i], files[i+1], 0o644)
		runGit(t, "-C", dir, "add", "-f", files[i])
	}
	gateCommit(t, dir, "track "+files[0])
	runGit(t, "-C", dir, "push", "--quiet", "origin", "HEAD")
	return gitOut(t, dir, "rev-parse", "HEAD")
}

// shareFromMain commits content at each rel (path, content, ...) on main and
// has feat/auth take it in with a merge, both pushed: the paths are then in
// the merge base of a later regate.
func (fx *regateFixture) shareFromMain(t *testing.T, files ...string) {
	t.Helper()
	fx.commitAndPush(t, fx.main, files...)
	runGit(t, "-C", fx.worktree, "merge", "--quiet", "--no-edit", "main")
	runGit(t, "-C", fx.worktree, "push", "--quiet", "origin", "HEAD")
}

// moveAndPush renames from to to with git mv in dir, the main checkout or
// the feature worktree, commits, and pushes the branch checked out there.
func (fx *regateFixture) moveAndPush(t *testing.T, dir, from, to string) {
	t.Helper()
	runGit(t, "-C", dir, "mv", from, to)
	gateCommit(t, dir, "rename "+from+" to "+to)
	runGit(t, "-C", dir, "push", "--quiet", "origin", "HEAD")
}

// shareMoveFromMain is shareFromMain for a rename: main renames from to to
// (moveAndPush) and feat/auth takes it in with a merge, both pushed.
func (fx *regateFixture) shareMoveFromMain(t *testing.T, from, to string) {
	t.Helper()
	fx.moveAndPush(t, fx.main, from, to)
	runGit(t, "-C", fx.worktree, "merge", "--quiet", "--no-edit", "main")
	runGit(t, "-C", fx.worktree, "push", "--quiet", "origin", "HEAD")
}

// regateSkipOnACaseSensitiveFS skips t unless the file system of t's temp
// directories, the regate fixture's, does not tell case apart (macOS's
// default): lstat of A there finds a file a.
func regateSkipOnACaseSensitiveFS(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	writeGateFile(t, dir, "a", "a\n", 0o644)
	_, err := os.Lstat(filepath.Join(dir, "A"))
	switch {
	case errors.Is(err, os.ErrNotExist):
		t.Skipf("the file system of %s tells case apart; the case needs one that does not", dir)
	case err != nil:
		t.Fatal(err)
	}
}

// untrackRunLog has main track docs/evidence/run.log (an ignored path) and
// feat/auth take that in with a merge and then stop tracking it, pushed: in
// the worktree, run.log is an ignored file HEAD deleted, with the content
// "the local log\n".
func (fx *regateFixture) untrackRunLog(t *testing.T) {
	t.Helper()
	fx.commitIgnoredAndPush(t, fx.main, "docs/evidence/run.log", "main's log\n")
	runGit(t, "-C", fx.worktree, "merge", "--quiet", "--no-edit", "main")
	runGit(t, "-C", fx.worktree, "rm", "--quiet", "--cached", "docs/evidence/run.log")
	gateCommit(t, fx.worktree, "stop tracking the run log")
	runGit(t, "-C", fx.worktree, "push", "--quiet", "origin", "HEAD")
	writeGateFile(t, fx.worktree, "docs/evidence/run.log", "the local log\n", 0o644)
}

// regateInTheWayText is Regate's refusal of a merge of origin's main at tip
// into feat/auth in worktree that would write over listed, what is on disk
// there.
func regateInTheWayText(worktree, tip, listed string) string {
	return "org: regate: merging origin's tip " + tip + " of main into feat/auth in " + worktree + " would write over what is on " +
		"disk at " + listed + ", which HEAD does not track: untracked files, or directories with untracked files in them, which " +
		"the clean-tree check passes over as ignored and git merge overwrites without a word (a file where the merge needs a " +
		"directory too), so regate merged nothing and recorded nothing: move them out of the worktree and run ralph org regate " +
		"--branch feat/auth again"
}

// TestRegate_RefusedWhenSomethingIgnoredIsInTheMergesWay covers D3's check
// before the merge (AC1): git merge overwrites an ignored file or directory at
// a path it writes, and the clean-tree check does not see one, so when origin's
// main adds or changes a path that HEAD does not track and something is on
// disk there (or a file where the merge needs a directory), Regate merges
// nothing, records nothing, runs no script, leaves HEAD, the tree, and origin
// as they were with no MERGE_HEAD and the ignored files as they were, and
// refuses naming what is in the way. The path the merge writes is also one a
// directory rename on either side moves a file to, or the name git merge
// gives HEAD's file in a file/directory conflict, which neither side's diff
// names. On a file system that does not tell case apart, what is in the way is
// also spelled on disk in another case than the path the merge writes, and
// the refusal names it as it is spelled on disk.
func TestRegate_RefusedWhenSomethingIgnoredIsInTheMergesWay(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(t *testing.T, fx *regateFixture)
		listed string
		// kept returns what is on disk in the way, to compare before and after.
		kept func(t *testing.T, fx *regateFixture) string
	}{
		{name: "an ignored file at a path the tip adds",
			setup: func(t *testing.T, fx *regateFixture) {
				fx.commitIgnoredAndPush(t, fx.main, "docs/evidence/run.log", "main's log\n")
				writeGateFile(t, fx.worktree, "docs/evidence/run.log", "the local log\n", 0o644)
			},
			listed: "docs/evidence/run.log",
			kept: func(t *testing.T, fx *regateFixture) string {
				return fileOrAbsent(t, filepath.Join(fx.worktree, "docs", "evidence", "run.log"))
			}},
		{name: "an ignored file HEAD deleted at a path the tip changed",
			setup: func(t *testing.T, fx *regateFixture) {
				fx.untrackRunLog(t)
				fx.commitIgnoredAndPush(t, fx.main, "docs/evidence/run.log", "main's log, changed\n")
			},
			listed: "docs/evidence/run.log",
			kept: func(t *testing.T, fx *regateFixture) string {
				return fileOrAbsent(t, filepath.Join(fx.worktree, "docs", "evidence", "run.log"))
			}},
		{name: "an ignored file where the tip adds a directory",
			setup: func(t *testing.T, fx *regateFixture) {
				fx.excludeLocally(t, "d")
				fx.commitIgnoredAndPush(t, fx.main, "d/x", "main's x\n", "d/y", "main's y\n")
				writeGateFile(t, fx.worktree, "d", "the local d\n", 0o644)
			},
			listed: "d",
			kept: func(t *testing.T, fx *regateFixture) string {
				return fileOrAbsent(t, filepath.Join(fx.worktree, "d"))
			}},
		{name: "an ignored directory at a path the tip adds a file at",
			setup: func(t *testing.T, fx *regateFixture) {
				fx.excludeLocally(t, "cache")
				fx.commitIgnoredAndPush(t, fx.main, "cache", "main's cache\n")
				writeGateFile(t, fx.worktree, "cache/inner", "the local cache\n", 0o644)
			},
			listed: "cache",
			kept: func(t *testing.T, fx *regateFixture) string {
				return fileOrAbsent(t, filepath.Join(fx.worktree, "cache", "inner"))
			}},
		{name: "an ignored symlink at a path the tip adds",
			setup: func(t *testing.T, fx *regateFixture) {
				fx.excludeLocally(t, "link")
				fx.commitIgnoredAndPush(t, fx.main, "link", "main's link\n")
				if err := os.Symlink("README.md", filepath.Join(fx.worktree, "link")); err != nil {
					t.Fatal(err)
				}
			},
			listed: "link",
			kept: func(t *testing.T, fx *regateFixture) string {
				target, err := os.Readlink(filepath.Join(fx.worktree, "link"))
				if err != nil {
					return "<" + err.Error() + ">"
				}
				return target
			}},
		{name: "as many paths in the way as the refusal lists",
			setup:  regateManyIgnoredInTheWay(regateInTheWayListed),
			listed: regateManyIgnoredListed,
			kept:   regateManyIgnoredKept(regateInTheWayListed)},
		{name: "one path in the way more than the refusal lists",
			setup:  regateManyIgnoredInTheWay(regateInTheWayListed + 1),
			listed: regateManyIgnoredListed + ", and 1 more",
			kept:   regateManyIgnoredKept(regateInTheWayListed + 1)},
		{name: "more paths in the way than the refusal lists",
			setup:  regateManyIgnoredInTheWay(regateInTheWayListed + 2),
			listed: regateManyIgnoredListed + ", and 2 more",
			kept:   regateManyIgnoredKept(regateInTheWayListed + 2)},
		{name: "an ignored file where the tip adds into a directory HEAD renamed",
			setup: func(t *testing.T, fx *regateFixture) {
				fx.excludeLocally(t, "lib2/new.txt")
				fx.shareFromMain(t, "lib/a.txt", "a\n")
				fx.moveAndPush(t, fx.worktree, "lib", "lib2")
				fx.commitAndPush(t, fx.main, "lib/new.txt", "main's new\n")
				writeGateFile(t, fx.worktree, "lib2/new.txt", "the local new\n", 0o644)
			},
			listed: "lib2/new.txt",
			kept: func(t *testing.T, fx *regateFixture) string {
				return fileOrAbsent(t, filepath.Join(fx.worktree, "lib2", "new.txt"))
			}},
		{name: "an ignored file where HEAD adds into a directory the tip renamed",
			setup: func(t *testing.T, fx *regateFixture) {
				fx.excludeLocally(t, "lib2/new.txt")
				fx.shareFromMain(t, "lib/a.txt", "a\n")
				fx.commitAndPush(t, fx.worktree, "lib/new.txt", "the feature's new\n")
				fx.moveAndPush(t, fx.main, "lib", "lib2")
				writeGateFile(t, fx.worktree, "lib2/new.txt", "the local new\n", 0o644)
			},
			listed: "lib2/new.txt",
			kept: func(t *testing.T, fx *regateFixture) string {
				return fileOrAbsent(t, filepath.Join(fx.worktree, "lib2", "new.txt"))
			}},
		{name: "an ignored file at the name a file/directory conflict gives HEAD's file",
			setup: func(t *testing.T, fx *regateFixture) {
				fx.excludeLocally(t, "d~HEAD")
				fx.commitAndPush(t, fx.worktree, "d", "the feature's d\n")
				fx.commitAndPush(t, fx.main, "d/x", "main's x\n")
				writeGateFile(t, fx.worktree, "d~HEAD", "the local d~HEAD\n", 0o644)
			},
			listed: "d~HEAD",
			kept: func(t *testing.T, fx *regateFixture) string {
				return fileOrAbsent(t, filepath.Join(fx.worktree, "d~HEAD"))
			}},
		{name: "an ignored file at a path the tip adds spelled in another case",
			setup: func(t *testing.T, fx *regateFixture) {
				regateSkipOnACaseSensitiveFS(t)
				fx.excludeLocally(t, "notes.txt")
				fx.commitIgnoredAndPush(t, fx.main, "NOTES.txt", "main's notes\n")
				writeGateFile(t, fx.worktree, "notes.txt", "the local notes\n", 0o644)
			},
			listed: "notes.txt",
			kept: func(t *testing.T, fx *regateFixture) string {
				return fileOrAbsent(t, filepath.Join(fx.worktree, "notes.txt"))
			}},
		{name: "an ignored file where the tip adds a directory spelled in another case",
			setup: func(t *testing.T, fx *regateFixture) {
				regateSkipOnACaseSensitiveFS(t)
				fx.excludeLocally(t, "d")
				fx.commitIgnoredAndPush(t, fx.main, "D/x", "main's x\n")
				writeGateFile(t, fx.worktree, "d", "the local d\n", 0o644)
			},
			listed: "d",
			kept: func(t *testing.T, fx *regateFixture) string {
				return fileOrAbsent(t, filepath.Join(fx.worktree, "d"))
			}},
		{name: "an ignored directory at a path the tip adds a file at spelled in another case",
			setup: func(t *testing.T, fx *regateFixture) {
				regateSkipOnACaseSensitiveFS(t)
				fx.excludeLocally(t, "cache")
				fx.commitIgnoredAndPush(t, fx.main, "CACHE", "main's cache\n")
				writeGateFile(t, fx.worktree, "cache/inner", "the local cache\n", 0o644)
			},
			listed: "cache",
			kept: func(t *testing.T, fx *regateFixture) string {
				return fileOrAbsent(t, filepath.Join(fx.worktree, "cache", "inner"))
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			tc.setup(t, fx)
			_, tip := fx.tips(t)
			if exec.Command("git", "-C", fx.worktree, "merge-base", "--is-ancestor", tip, "HEAD").Run() == nil {
				t.Fatalf("HEAD contains origin's main %s; the case needs a merge", tip)
			}
			kept, before := tc.kept(t, fx), fx.snapshot(t)
			if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
				t.Fatalf("the worktree is not clean before the regate:\n%s", st)
			}
			res, err := fx.regate(context.Background())
			fx.assertRefused(t, res, err, before, nil, nil)
			if want := regateInTheWayText(fx.worktree, tip, tc.listed); res.Refused != want {
				t.Errorf("refusal:\n%s\nwant\n%s", res.Refused, want)
			}
			if got := tc.kept(t, fx); got != kept {
				t.Errorf("what was in the way is now %q; want it as it was, %q", got, kept)
			}
			if fx.mergeInProgress(t) {
				t.Fatal("a merge is in progress in the worktree after the refusal")
			}
		})
	}
}

// regateManyIgnoredListed is what the refusal of regateManyIgnoredInTheWay's
// cases lists before its ", and N more": the first regateInTheWayListed paths.
const regateManyIgnoredListed = "ignored/f101, ignored/f102, ignored/f103, ignored/f104, ignored/f105, ignored/f106, " +
	"ignored/f107, ignored/f108, ignored/f109, ignored/f110"

// regateManyIgnoredInTheWay sets up n ignored files in the merge's way,
// ignored/f101 to ignored/f<100+n>, each one origin's main adds and the
// worktree holds a file of its own at.
func regateManyIgnoredInTheWay(n int) func(t *testing.T, fx *regateFixture) {
	return func(t *testing.T, fx *regateFixture) {
		var files []string
		for i := 1; i <= n; i++ {
			files = append(files, "ignored/f"+strconv.Itoa(100+i), "main's\n")
		}
		fx.commitIgnoredAndPush(t, fx.main, files...)
		for i := 0; i < len(files); i += 2 {
			writeGateFile(t, fx.worktree, files[i], "the local file\n", 0o644)
		}
	}
}

// regateManyIgnoredKept returns what is on disk at the n paths
// regateManyIgnoredInTheWay(n) put in the way.
func regateManyIgnoredKept(n int) func(t *testing.T, fx *regateFixture) string {
	return func(t *testing.T, fx *regateFixture) string {
		var all []string
		for i := 1; i <= n; i++ {
			all = append(all, fileOrAbsent(t, filepath.Join(fx.worktree, "ignored", "f"+strconv.Itoa(100+i))))
		}
		return strings.Join(all, "|")
	}
}

// TestRegate_MergesOverAPathHEADDeletedThatTheTipDidNotTouch covers the other
// side of D3's check: an ignored file sits at a path HEAD deleted and origin's
// main has as the merge base had it, so the merge leaves the path deleted and
// does not write it. Regate merges, passes, and pushes as usual, and the file
// is as it was.
func TestRegate_MergesOverAPathHEADDeletedThatTheTipDidNotTouch(t *testing.T) {
	fx := newRegateFixture(t)
	fx.untrackRunLog(t)
	pre, _ := fx.tips(t)
	tip := fx.pushToOrigin(t, "main")
	res := fx.mustRegate(t)
	if res.Outcome != RegateOutcomeMerged || !res.Passed || !res.Pushed || res.Refused != "" || res.PreMergeHead != pre {
		t.Fatalf("regate = %+v; want the merge commit of %s on %s gated and pushed", res, tip, pre)
	}
	fx.assertMergeCommit(t, res.Merge, pre, tip)
	if got := fileOrAbsent(t, filepath.Join(fx.worktree, "docs", "evidence", "run.log")); got != "the local log\n" {
		t.Errorf("docs/evidence/run.log after the regate = %q; want the local log as it was", got)
	}
	if tracked := gitOut(t, fx.worktree, "ls-tree", "--name-only", res.Merge, "docs/evidence/"); tracked != "" {
		t.Errorf("the merge commit tracks %q; want docs/evidence/run.log still deleted", tracked)
	}
}

// TestRegate_MergesWhenNothingUntrackedIsInTheMergesWay covers the paths D3's
// check lets the merge write although lstat finds something on disk there,
// since git merge loses nothing there: an empty directory (empty directories
// in it included) where origin's main adds a file, a directory HEAD tracks
// that main turns into a file, and a file HEAD tracks that main turns into a
// directory. On a file system that does not tell case apart, also a file or a
// directory HEAD tracks that main renames only in case, and a file HEAD tracks
// that main turns into a directory spelled in another case: lstat finds HEAD's
// file there under its spelling on disk. Regate merges, passes, and pushes as
// usual, and the path is then as main has it, in a clean tree.
func TestRegate_MergesWhenNothingUntrackedIsInTheMergesWay(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, fx *regateFixture)
		// path and content are a file the merge writes, as main has it.
		path, content string
	}{
		{name: "an empty directory where the tip adds a file",
			setup: func(t *testing.T, fx *regateFixture) {
				fx.commitAndPush(t, fx.main, "notes", "main's notes\n")
				mkdirAll(t, filepath.Join(fx.worktree, "notes", "empty"))
			},
			path: "notes", content: "main's notes\n"},
		{name: "a directory HEAD tracks that the tip turns into a file",
			setup: func(t *testing.T, fx *regateFixture) {
				fx.shareFromMain(t, "lib/a.txt", "a\n")
				runGit(t, "-C", fx.main, "rm", "--quiet", "lib/a.txt")
				fx.commitAndPush(t, fx.main, "lib", "main's lib\n")
			},
			path: "lib", content: "main's lib\n"},
		{name: "a file HEAD tracks that the tip turns into a directory",
			setup: func(t *testing.T, fx *regateFixture) {
				fx.shareFromMain(t, "conf", "main's conf\n")
				runGit(t, "-C", fx.main, "rm", "--quiet", "conf")
				fx.commitAndPush(t, fx.main, "conf/x", "main's x\n")
			},
			path: "conf/x", content: "main's x\n"},
		{name: "a file HEAD tracks that the tip renames only in case",
			setup: func(t *testing.T, fx *regateFixture) {
				regateSkipOnACaseSensitiveFS(t)
				fx.shareMoveFromMain(t, "README.md", "readme.md")
				fx.moveAndPush(t, fx.main, "readme.md", "README.md")
			},
			path: "README.md", content: "regate fixture\n"},
		{name: "a directory HEAD tracks that the tip renames only in case",
			setup: func(t *testing.T, fx *regateFixture) {
				regateSkipOnACaseSensitiveFS(t)
				fx.shareFromMain(t, "Lib/a.txt", "a\n")
				// git mv cannot rename a directory only in case there.
				fx.moveAndPush(t, fx.main, "Lib", "lib.tmp")
				fx.moveAndPush(t, fx.main, "lib.tmp", "lib")
			},
			path: "lib/a.txt", content: "a\n"},
		{name: "a file HEAD tracks that the tip turns into a directory spelled in another case",
			setup: func(t *testing.T, fx *regateFixture) {
				regateSkipOnACaseSensitiveFS(t)
				fx.shareFromMain(t, "conf", "main's conf\n")
				runGit(t, "-C", fx.main, "rm", "--quiet", "conf")
				fx.commitAndPush(t, fx.main, "CONF/x", "main's x\n")
			},
			path: "CONF/x", content: "main's x\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			tc.setup(t, fx)
			pre, tip := fx.tips(t)
			if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
				t.Fatalf("the worktree is not clean before the regate:\n%s", st)
			}
			res := fx.mustRegate(t)
			if res.Outcome != RegateOutcomeMerged || !res.Passed || !res.Pushed || res.Refused != "" || res.PreMergeHead != pre {
				t.Fatalf("regate = %+v; want the merge commit of %s on %s gated and pushed", res, tip, pre)
			}
			fx.assertMergeCommit(t, res.Merge, pre, tip)
			if got := fileOrAbsent(t, filepath.Join(fx.worktree, filepath.FromSlash(tc.path))); got != tc.content {
				t.Errorf("%s after the regate = %q; want main's %q", tc.path, got, tc.content)
			}
			if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all"); st != "" {
				t.Errorf("the worktree after the regate:\n%s", st)
			}
		})
	}
}

// TestRegate_AnUnrelatedDefaultBranchTipIsLeftToGitMergeToRefuse covers D3's
// check with no merge base: origin's main is a root commit HEAD shares no
// history with, so the check looks at nothing, git merge refuses the
// unrelated histories itself, and Regate passes its refusal on, with nothing
// merged or recorded and HEAD, the tree, and origin as they were.
func TestRegate_AnUnrelatedDefaultBranchTipIsLeftToGitMergeToRefuse(t *testing.T) {
	fx := newRegateFixture(t)
	tip := gitOut(t, fx.origin, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit-tree", "-m", "unrelated",
		"refs/heads/main^{tree}")
	runGit(t, "-C", fx.origin, "update-ref", "refs/heads/main", tip)
	before := fx.snapshot(t)
	res, err := fx.regate(context.Background())
	fx.assertRefused(t, res, err, before, []string{"git merge --no-ff --no-edit -m 'Merge origin/main " + tip[:12] + " into feat/auth' " +
		tip + " in " + fx.worktree + " made no merge commit (exit 128)", "refusing to merge unrelated histories"}, nil)
	if fx.mergeInProgress(t) {
		t.Fatal("a merge is in progress in the worktree after the refusal")
	}
}

// regateWrapGit puts a git first on PATH for the test that runs the real git
// for every command but one whose arguments name merge-tree: it appends that
// command's arguments to the file it returns and runs mergeTree, a shell
// snippet, before it runs the real git.
func regateWrapGit(t *testing.T, mergeTree string) (calls string) {
	t.Helper()
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	calls = filepath.Join(dir, "merge-tree-calls")
	writeGateFile(t, dir, "bin/git", "#!/bin/sh\nfor a; do\n  if [ \"$a\" = merge-tree ]; then\n    echo \"$*\" >> '"+calls+"'\n"+
		mergeTree+"\n    break\n  fi\ndone\nexec '"+gitPath+"' \"$@\"\n", 0o755)
	t.Setenv("PATH", filepath.Join(dir, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	return calls
}

// TestRegate_ChecksTheMergesWayWithAGitMergeTreeCannotWriteATree covers D3's
// check with a git merge-tree that does not run: git before 2.38, whose
// merge-tree has no --write-tree (its -h does not name it), falls back to the
// paths the tip's side changed that HEAD does not have, and refuses an
// ignored file at a path the tip adds the same way; a merge-tree whose -h
// names --write-tree and that fails is an error, with nothing merged or
// recorded and HEAD, the tree, and origin as they were.
func TestRegate_ChecksTheMergesWayWithAGitMergeTreeCannotWriteATree(t *testing.T) {
	setup := func(t *testing.T, fx *regateFixture) {
		fx.commitIgnoredAndPush(t, fx.main, "docs/evidence/run.log", "main's log\n")
		writeGateFile(t, fx.worktree, "docs/evidence/run.log", "the local log\n", 0o644)
	}
	t.Run("a git before --write-tree", func(t *testing.T) {
		fx := newRegateFixture(t)
		setup(t, fx)
		_, tip := fx.tips(t)
		calls := regateWrapGit(t, "    echo 'usage: git merge-tree <base-tree> <branch1> <branch2>' >&2\n    exit 129")
		before := fx.snapshot(t)
		res, err := fx.regate(context.Background())
		fx.assertRefused(t, res, err, before, nil, nil)
		if want := regateInTheWayText(fx.worktree, tip, "docs/evidence/run.log"); res.Refused != want {
			t.Errorf("refusal:\n%s\nwant\n%s", res.Refused, want)
		}
		if got, want := fileOrAbsent(t, calls), "--no-optional-locks -C "+fx.worktree+" merge-tree --write-tree --no-messages HEAD "+
			tip+"\n--no-optional-locks -C "+fx.worktree+" merge-tree -h\n"; got != want {
			t.Errorf("merge-tree calls:\n%s\nwant\n%s", got, want)
		}
		if got := fileOrAbsent(t, filepath.Join(fx.worktree, "docs", "evidence", "run.log")); got != "the local log\n" {
			t.Errorf("docs/evidence/run.log after the refusal = %q; want the local log as it was", got)
		}
	})
	t.Run("a merge-tree that fails", func(t *testing.T) {
		fx := newRegateFixture(t)
		setup(t, fx)
		_, tip := fx.tips(t)
		regateWrapGit(t, "    case \" $* \" in *' -h '*) ;; *) echo 'fatal: merge-tree broke' >&2; exit 128 ;; esac")
		before := fx.snapshot(t)
		res, err := fx.regate(context.Background())
		want := "org: regate: git merge-tree --write-tree --no-messages HEAD " + tip + " in " + fx.worktree + ": exit 128: fatal: " +
			"merge-tree broke"
		if err == nil || err.Error() != want {
			t.Fatalf("regate = %+v, %v; want the error %q", res, err, want)
		}
		if !reflect.DeepEqual(res, RegateResult{}) {
			t.Errorf("regate = %+v; want no result with the error", res)
		}
		if after := fx.snapshot(t); after != before {
			t.Errorf("the regate changed the ledger, the worktree, or origin:\nbefore %+v\nafter  %+v", before, after)
		}
		if calls := fx.calls(t); calls != nil {
			t.Errorf("the regate ran %v; want no script run", calls)
		}
		fx.assertUnlocked(t)
	})
}
