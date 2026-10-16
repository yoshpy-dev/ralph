package org

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// `ralph org regate` at the org layer (plan
// docs/plans/active/2026-10-16-org-regate.md, AC1 to AC9, and the regate
// side of D7 and D9): Regate on real git repositories whose origin is a bare
// repository under the test's temp dir, so no test reaches a network, with
// the stub gate scripts of gate_test.go.

// regateTestClock is the fixtures' fixed clock, so a gate log of the org
// auth is regateTestLog.
var regateTestClock = time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

const regateTestLog = "auth-20261016T120000Z.log"

// regateFixture is the branch feat/auth of the feature org auth after its PR
// was opened and the org disbanded. origin is a bare repository whose main
// holds one commit (a .gitignore, a README.md and the stub gate scripts,
// writeGateStubs) and whose feat/auth holds the feature's one commit on top
// of it. main is the main checkout, a repository on main whose origin remote
// is that path, with refs/remotes/origin/HEAD pointing to
// refs/remotes/origin/main as a clone leaves it; the feature worktree is a
// linked worktree of it on feat/auth, made from main as
// scripts/ralph-worktree.sh makes one. Git's automatic maintenance is off in
// both repositories, so no background git outlives a test in its temp dir.
// The main checkout's config names a user (Regate Tester), whom a merge
// commit Regate makes is by unless the caller's environment says otherwise.
// The ledger holds auth's complete binding to feat/auth with that worktree,
// its leader, and its disband. The embedded gateFixture gives the stubs'
// helpers (setExit, hook, calls, gateEvents, stopRebaseOnAConflict) and the
// caller environment env, as gate_test.go describes them.
type regateFixture struct {
	*gateFixture
	origin string
}

func newRegateFixture(t *testing.T) *regateFixture {
	t.Helper()
	isolateGitEnv(t)
	tmp := resolved(t, t.TempDir())
	fx := &regateFixture{
		gateFixture: &gateFixture{stateDir: filepath.Join(tmp, "state"), main: filepath.Join(tmp, "main"),
			worktree: filepath.Join(tmp, "wt-auth"), out: filepath.Join(tmp, "out")},
		origin: filepath.Join(tmp, "origin.git"),
	}
	runGit(t, "init", "--quiet", "--bare", "-b", "main", fx.origin)
	runGit(t, "-C", fx.origin, "config", "gc.auto", "0")
	runGit(t, "-C", fx.origin, "config", "receive.autogc", "false")
	runGit(t, "init", "--quiet", "-b", "main", fx.main)
	runGit(t, "-C", fx.main, "config", "maintenance.auto", "false")
	runGit(t, "-C", fx.main, "config", "gc.auto", "0")
	runGit(t, "-C", fx.main, "config", "user.name", "Regate Tester")
	runGit(t, "-C", fx.main, "config", "user.email", "regate@example.com")
	writeGateFile(t, fx.main, ".gitignore", ".harness/\ndocs/evidence/*.log\nignored/\n", 0o644)
	writeGateFile(t, fx.main, "README.md", "regate fixture\n", 0o644)
	writeGateStubs(t, fx.main)
	runGit(t, "-C", fx.main, "add", ".")
	gateCommit(t, fx.main, "init")
	runGit(t, "-C", fx.main, "remote", "add", "origin", fx.origin)
	runGit(t, "-C", fx.main, "push", "--quiet", "origin", "main")
	runGit(t, "-C", fx.main, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	runGit(t, "-C", fx.main, "worktree", "add", "--quiet", "-b", "feat/auth", fx.worktree)
	writeGateFile(t, fx.worktree, "internal/auth/auth.go", "package auth\n", 0o644)
	runGit(t, "-C", fx.worktree, "add", ".")
	gateCommit(t, fx.worktree, "the feature")
	runGit(t, "-C", fx.worktree, "push", "--quiet", "origin", "feat/auth")
	mkdirAll(t, fx.out)

	fx.f = testFeature("split-1", "auth", "0123456789ab")
	fx.f.Worktree = fx.worktree
	fx.o = &Org{Manifest: NewManifestStoreAtPath(ManifestPathIn(fx.stateDir)), Now: func() time.Time { return regateTestClock }}
	fx.appendEvents(t, boundReserveEvent("auth", fx.f, "internal/auth/"))
	seedSeat(t, fx.o, "auth", LeaderIdentity, LeaderIdentity, "pane-1")
	fx.appendEvents(t, ManifestEvent{TS: fx.o.now(), OrgID: "auth", Event: EventDisbanded})

	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); name != "USER" && name != herdrPaneIDEnv && name != "GATE_TEST_OUT" {
			fx.env = append(fx.env, kv)
		}
	}
	fx.env = append(fx.env, "USER=alice", "GATE_TEST_OUT="+fx.out)
	return fx
}

// appendEvents appends events to the fixture's manifest.
func (fx *regateFixture) appendEvents(t *testing.T, events ...ManifestEvent) {
	t.Helper()
	for _, ev := range events {
		if err := fx.o.Manifest.Append(ev); err != nil {
			t.Fatal(err)
		}
	}
}

// appendLines appends each line to the manifest: a ManifestEvent as a JSON
// line, a string as raw bytes (a corrupt line, which ends with "\n").
func (fx *regateFixture) appendLines(t *testing.T, lines ...any) {
	t.Helper()
	for _, line := range lines {
		switch l := line.(type) {
		case ManifestEvent:
			fx.appendEvents(t, l)
		case string:
			f, err := os.OpenFile(fx.o.Manifest.Path(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.WriteString(l); err != nil {
				t.Fatal(err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
		default:
			t.Fatalf("appendLines: a %T is neither a ManifestEvent nor a raw line", line)
		}
	}
}

// rebind binds feat/auth to the org auth-2, disbanded, with the feature
// approved again (another digest) in worktree: the branch's latest binding.
func (fx *regateFixture) rebind(t *testing.T, worktree string) {
	t.Helper()
	f := *fx.f
	f.Digest, f.Worktree = "fedcba987654", worktree
	fx.appendEvents(t, boundReserveEvent("auth-2", &f, "internal/auth/"), ManifestEvent{TS: fx.o.now(), OrgID: "auth-2", Event: EventDisbanded})
}

// regate is Regate of feat/auth with the fixture's environment.
func (fx *regateFixture) regate(ctx context.Context) (RegateResult, error) {
	return fx.o.Regate(ctx, RegateParams{Branch: "feat/auth", Env: fx.env})
}

// mustRegate is regate, failing the test on an error.
func (fx *regateFixture) mustRegate(t *testing.T) RegateResult {
	t.Helper()
	res, err := fx.regate(context.Background())
	if err != nil {
		t.Fatalf("regate: %v", err)
	}
	return res
}

// tips returns the worktree's HEAD and origin's main.
func (fx *regateFixture) tips(t *testing.T) (head, tip string) {
	t.Helper()
	return gitOut(t, fx.worktree, "rev-parse", "HEAD"), gitOut(t, fx.origin, "rev-parse", "refs/heads/main")
}

// pushToOrigin moves origin's branch to a new empty commit on top of its tip,
// made in origin itself: a push from elsewhere, which the main checkout's
// remote-tracking refs do not know of until a fetch. It returns the commit.
func (fx *regateFixture) pushToOrigin(t *testing.T, branch string) string {
	t.Helper()
	ref := "refs/heads/" + branch
	sha := gitOut(t, fx.origin, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit-tree", "-p", ref, "-m", "pushed", ref+"^{tree}")
	runGit(t, "-C", fx.origin, "update-ref", ref, sha)
	return sha
}

// commitAndPush commits content at each rel (path, content, path, content,
// ...) in dir, the main checkout or the feature worktree, and pushes the
// branch checked out there to origin: a change that a merge of the other
// branch then conflicts with. It returns the commit.
func (fx *regateFixture) commitAndPush(t *testing.T, dir string, files ...string) string {
	t.Helper()
	for i := 0; i+1 < len(files); i += 2 {
		writeGateFile(t, dir, files[i], files[i+1], 0o644)
		runGit(t, "-C", dir, "add", files[i])
	}
	gateCommit(t, dir, "change "+files[0])
	runGit(t, "-C", dir, "push", "--quiet", "origin", "HEAD")
	return gitOut(t, dir, "rev-parse", "HEAD")
}

// originBranch returns origin's feat/auth.
func (fx *regateFixture) originBranch(t *testing.T) string {
	t.Helper()
	return gitOut(t, fx.origin, "rev-parse", "refs/heads/feat/auth")
}

// rejectPushes makes origin refuse every push, with a pre-receive hook that
// says so on stderr, until allow removes the hook.
func (fx *regateFixture) rejectPushes(t *testing.T) (allow func()) {
	t.Helper()
	writeGateFile(t, fx.origin, "hooks/pre-receive", "#!/bin/sh\necho 'pushes are closed' >&2\nexit 1\n", 0o755)
	return func() {
		if err := os.Remove(filepath.Join(fx.origin, "hooks", "pre-receive")); err != nil {
			t.Fatal(err)
		}
	}
}

// setHooks makes the main checkout and the feature worktree run the git
// hooks in fx.out/hooks (core.hooksPath in the repository's own config), and
// writes each name, body pair there.
func (fx *regateFixture) setHooks(t *testing.T, hooks ...string) {
	t.Helper()
	for i := 0; i+1 < len(hooks); i += 2 {
		writeGateFile(t, fx.out, "hooks/"+hooks[i], "#!/bin/sh\n"+hooks[i+1], 0o755)
	}
	runGit(t, "-C", fx.main, "config", "core.hooksPath", filepath.Join(fx.out, "hooks"))
}

// undoneMerge returns the merge commit Regate made and undid: feat/auth's
// previous value in its reflog.
func (fx *regateFixture) undoneMerge(t *testing.T) string {
	t.Helper()
	return gitOut(t, fx.worktree, "rev-parse", "feat/auth@{1}")
}

// mergeInProgress reports whether the feature worktree has a MERGE_HEAD.
func (fx *regateFixture) mergeInProgress(t *testing.T) bool {
	t.Helper()
	_, err := os.Stat(gitOut(t, fx.worktree, "rev-parse", "--path-format=absolute", "--git-path", "MERGE_HEAD"))
	return err == nil
}

// assertMergeCommit checks that commit is D3's merge commit of origin's main
// at tip into feat/auth at pre: its parents are pre and tip, in this order,
// and its message is "Merge origin/main <tip's 12 hex> into feat/auth".
func (fx *regateFixture) assertMergeCommit(t *testing.T, commit, pre, tip string) {
	t.Helper()
	if parents := strings.Fields(gitOut(t, fx.worktree, "rev-list", "--parents", "-n", "1", commit))[1:]; !slices.Equal(parents, []string{pre, tip}) {
		t.Errorf("merge commit %s has the parents %v; want the branch's %s, then origin's main %s", commit, parents, pre, tip)
	}
	if msg, want := gitOut(t, fx.worktree, "log", "-1", "--format=%B", commit), "Merge origin/main "+tip[:12]+" into feat/auth"; msg != want {
		t.Errorf("merge commit message = %q; want %q", msg, want)
	}
}

// regateMergeKeptText is RegateResult.NotUndone for the merge commit merge on
// pre in worktree, kept because of why.
func regateMergeKeptText(worktree, merge, pre, why string) string {
	return "the merge commit " + merge + " was not undone, since " + why + ": check the worktree (git -C " + worktree + " status, " +
		"git -C " + worktree + " log), and to undo the merge run git -C " + worktree + " reset --keep " + pre + ", which also drops " +
		"every commit made after it"
}

// regateUndoneTextFor is what an error or refusal of Regate says after it
// undid the merge commit merge on pre.
func regateUndoneTextFor(merge, pre string) string {
	return "the merge commit " + merge + " was undone (git reset --keep " + pre + "), so only the reflog has it now"
}

// mergeClass is feat/auth's merge-order class in the ledger now, against
// origin's refs (what the ls-remote of ralph org merge-order lists):
// collectMergeTargets and classifyMergeTarget, with no PR.
func (fx *regateFixture) mergeClass(t *testing.T) string {
	t.Helper()
	l, err := readMergeLedger(fx.o.Manifest.Path())
	if err != nil {
		t.Fatal(err)
	}
	main := gitOut(t, fx.origin, "rev-parse", "refs/heads/main")
	remote := mergeRemote{defaultBranch: "main", defaultHead: main, heads: map[string]string{"main": main}}
	if out, err := exec.Command("git", "-C", fx.origin, "rev-parse", "--verify", "--quiet", "refs/heads/feat/auth").Output(); err == nil {
		remote.heads["feat/auth"] = strings.TrimSpace(string(out))
	}
	targets, _ := collectMergeTargets(l)
	for _, target := range targets {
		if target.branch == "feat/auth" {
			class, _ := classifyMergeTarget(target, remote)
			return class
		}
	}
	t.Fatal("merge-order has no target feat/auth")
	return ""
}

// regateSnapshot is what a refused Regate must leave as it was: the
// manifest, the worktree's HEAD and status, and origin's refs. A part that
// cannot be read holds the error, the same each time.
type regateSnapshot struct {
	manifest, head, status, originRefs string
}

func (fx *regateFixture) snapshot(t *testing.T) regateSnapshot {
	t.Helper()
	return regateSnapshot{
		manifest:   fileOrAbsent(t, fx.o.Manifest.Path()),
		head:       regateGitState(fx.worktree, "rev-parse", "HEAD"),
		status:     regateGitState(fx.worktree, "status", "--porcelain", "--untracked-files=all"),
		originRefs: regateGitState(fx.origin, "for-each-ref"),
	}
}

// regateGitState is git -C dir args's output, or its error and output when
// it fails.
func regateGitState(dir string, args ...string) string {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		return "<" + err.Error() + "> " + string(out)
	}
	return string(out)
}

// assertRefused checks a refusal: no error, Refused starting "org: regate: "
// with every want and no notWant, every other field empty, the manifest, the
// worktree, and origin as before, no gates dir, no stub run, and regate.lock
// released.
func (fx *regateFixture) assertRefused(t *testing.T, res RegateResult, err error, before regateSnapshot, want, notWant []string) {
	t.Helper()
	fx.assertRefusal(t, res, err, before, want, notWant)
	if _, err := os.Stat(filepath.Join(fx.stateDir, gateLogDir)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a refused regate made %s (stat: %v); want no log", gateLogDir, err)
	}
	if calls := fx.calls(t); calls != nil {
		t.Errorf("a refused regate ran %v; want no script run", calls)
	}
}

// assertRefusedAgain is assertRefused for a refusal after an earlier Regate
// of the fixture ran the scripts: the stubs that ran are still callsBefore.
func (fx *regateFixture) assertRefusedAgain(t *testing.T, res RegateResult, err error, before regateSnapshot, callsBefore, want,
	notWant []string) {
	t.Helper()
	fx.assertRefusal(t, res, err, before, want, notWant)
	if calls := fx.calls(t); !slices.Equal(calls, callsBefore) {
		t.Errorf("a refused regate ran scripts: %v, before it %v", calls, callsBefore)
	}
}

// assertRefusal is what assertRefused and assertRefusedAgain both check:
// the refusal's text, every other field empty, the manifest, the worktree,
// and origin as before, and regate.lock released.
func (fx *regateFixture) assertRefusal(t *testing.T, res RegateResult, err error, before regateSnapshot, want, notWant []string) {
	t.Helper()
	if err != nil {
		t.Fatalf("regate = %v; want a refusal, not an error", err)
	}
	if !strings.HasPrefix(res.Refused, "org: regate: ") {
		t.Fatalf("regate = %+v; want a refusal starting \"org: regate: \"", res)
	}
	for _, w := range want {
		if !strings.Contains(res.Refused, w) {
			t.Errorf("refusal %q does not say %q", res.Refused, w)
		}
	}
	for _, w := range notWant {
		if strings.Contains(res.Refused, w) {
			t.Errorf("refusal %q says %q", res.Refused, w)
		}
	}
	if !reflect.DeepEqual(res, RegateResult{Refused: res.Refused}) {
		t.Errorf("refused regate = %+v; want only Refused set", res)
	}
	if after := fx.snapshot(t); after != before {
		t.Errorf("a refused regate changed the ledger, the worktree, or origin:\nbefore %+v\nafter  %+v", before, after)
	}
	fx.assertUnlocked(t)
}

// assertUnlocked checks that regate.lock is not held.
func (fx *regateFixture) assertUnlocked(t *testing.T) {
	t.Helper()
	if held, err := fileLockHeld(regateLockPathIn(fx.stateDir)); err != nil || held {
		t.Errorf("regate.lock held = %v (%v) after Regate returned; want it released", held, err)
	}
}

// setRegateHook sets one of regateTestHooks to fn for the test.
func setRegateHook(t *testing.T, hook *func(), fn func()) {
	t.Helper()
	*hook = fn
	t.Cleanup(func() { *hook = nil })
}

// regateCorruptLine is a manifest line that is not JSON: a gate record of
// auth cut off while it was written.
const regateCorruptLine = `{"ts":"2026-10-16T11:30:00Z","org_id":"auth","event":"gate_pa` + "\n"

// TestRegate_RefusedBeforeTheFetch covers AC1: each check before the fetch
// refuses with its reason and one next step, records nothing, changes
// neither the worktree nor origin, runs no script, and releases regate.lock.
func TestRegate_RefusedBeforeTheFetch(t *testing.T) {
	t.Run("no branch named", func(t *testing.T) {
		fx := newRegateFixture(t)
		before := fx.snapshot(t)
		res, err := fx.o.Regate(context.Background(), RegateParams{Env: fx.env})
		fx.assertRefused(t, res, err, before, []string{"no branch named", "ralph org merge-order lists the ledger's branches"}, nil)
	})
	noBinding := []string{"no complete feature binding in the ledger names the branch feat/other", "ralph org merge-order lists"}
	cases := []struct {
		name    string
		branch  string // "" is feat/auth
		setup   func(t *testing.T, fx *regateFixture)
		want    func(fx *regateFixture) []string
		notWant []string
	}{
		{name: "a branch no binding names", branch: "feat/other",
			want: func(*regateFixture) []string { return noBinding }},
		{name: "a branch only an incomplete binding names", branch: "feat/other",
			setup: func(t *testing.T, fx *regateFixture) {
				f := *fx.f
				f.Branch, f.Digest = "feat/other", ""
				fx.appendEvents(t, boundReserveEvent("other", &f, "internal/other/"), ManifestEvent{OrgID: "other", Event: EventDisbanded})
			},
			want: func(*regateFixture) []string { return noBinding }},
		{name: "the org runs",
			setup: func(t *testing.T, fx *regateFixture) {
				fx.appendEvents(t, boundReserveEvent("auth", fx.f, "internal/auth/"))
				seedSeat(t, fx.o, "auth", LeaderIdentity, LeaderIdentity, "pane-2")
			},
			want: func(*regateFixture) []string {
				return []string{`the branch feat/auth is bound to org_id "auth", which is running`, "ralph org gate --org-id auth",
					"wait for the org to be disbanded and run ralph org regate --branch feat/auth then"}
			}},
		// The org_id was started again for another feature: its leader may
		// use the worktree, and a record would count as that org's.
		{name: "the org_id runs again for another feature",
			setup: func(t *testing.T, fx *regateFixture) {
				fx.appendEvents(t, boundReserveEvent("auth", testFeature("split-2", "billing", "0123456789ab"), "internal/billing/"))
			},
			want: func(*regateFixture) []string { return []string{`bound to org_id "auth", which is running`} }},
		{name: "the latest of two bindings is of an org that runs",
			setup: func(t *testing.T, fx *regateFixture) {
				f := *fx.f
				f.Digest = "fedcba987654"
				fx.appendEvents(t, boundReserveEvent("auth-2", &f, "internal/auth/"))
			},
			want:    func(*regateFixture) []string { return []string{`bound to org_id "auth-2", which is running`} },
			notWant: []string{`org_id "auth",`}},
		{name: "the worktree is gone",
			setup: func(t *testing.T, fx *regateFixture) {
				if err := os.RemoveAll(fx.worktree); err != nil {
					t.Fatal(err)
				}
			},
			want: func(fx *regateFixture) []string {
				return []string{`org_id "auth"'s feature worktree ` + fx.worktree + " of the branch feat/auth does not exist",
					"clean up what is left with ./scripts/ralph-worktree.sh cleanup --id org-auth in the main checkout",
					"git worktree add " + fx.worktree + " feat/auth", "run ralph org regate --branch feat/auth again"}
			}},
		{name: "the worktree is a file",
			setup: func(t *testing.T, fx *regateFixture) {
				writeGateFile(t, fx.out, "a-file", "x\n", 0o644)
				fx.rebind(t, filepath.Join(fx.out, "a-file"))
			},
			want: func(fx *regateFixture) []string {
				return []string{`org_id "auth-2"'s feature worktree ` + filepath.Join(fx.out, "a-file") + " is not a directory"}
			}},
		{name: "the worktree is outside git",
			setup: func(t *testing.T, fx *regateFixture) { fx.rebind(t, fx.out) },
			want: func(fx *regateFixture) []string {
				return []string{`org_id "auth-2"'s feature worktree ` + fx.out + " is not a git worktree (git rev-parse --show-toplevel:"}
			}},
		{name: "the worktree is a directory in the main checkout",
			setup: func(t *testing.T, fx *regateFixture) {
				mkdirAll(t, filepath.Join(fx.main, "ignored", "wt"))
				fx.rebind(t, filepath.Join(fx.main, "ignored", "wt"))
			},
			want: func(fx *regateFixture) []string {
				return []string{filepath.Join(fx.main, "ignored", "wt") + " is not the root of a git worktree but a directory in " + fx.main}
			}},
		{name: "another branch checked out",
			setup: func(t *testing.T, fx *regateFixture) {
				runGit(t, "-C", fx.worktree, "switch", "--quiet", "-c", "other")
			},
			want: func(fx *regateFixture) []string {
				return []string{fx.worktree + " has the branch other checked out, not the feature branch feat/auth: run git switch " +
					"feat/auth and run ralph org regate --branch feat/auth again"}
			}},
		{name: "a detached HEAD",
			setup: func(t *testing.T, fx *regateFixture) { runGit(t, "-C", fx.worktree, "switch", "--quiet", "--detach") },
			want: func(fx *regateFixture) []string {
				return []string{"HEAD is detached in " + fx.worktree + ", not on the feature branch feat/auth: run git switch " +
					"feat/auth and run ralph org regate --branch feat/auth again"}
			},
			notWant: []string{"rebase"}},
		{name: "a rebase stopped on a conflict",
			setup: func(t *testing.T, fx *regateFixture) { fx.stopRebaseOnAConflict(t, false) },
			want: func(fx *regateFixture) []string {
				return []string{"a rebase is in progress in " + fx.worktree + ", so HEAD is detached",
					"git rebase --abort, and run ralph org regate --branch feat/auth again"}
			}},
		// gateCheckBranch looks for a rebase only with HEAD detached;
		// regateCheckInProgress also looks with the branch checked out.
		{name: "a rebase directory with the branch checked out",
			setup: func(t *testing.T, fx *regateFixture) {
				mkdirAll(t, gitOut(t, fx.worktree, "rev-parse", "--path-format=absolute", "--git-path", "rebase-merge"))
			},
			want: func(fx *regateFixture) []string {
				return []string{"a rebase is in progress in " + fx.worktree + ": finish it with git rebase --continue, or undo it " +
					"with git rebase --abort, and run ralph org regate --branch feat/auth again"}
			},
			notWant: []string{"detached"}},
		// The merge check comes before the clean tree check, so a merge with
		// staged changes is told as a merge.
		{name: "a merge stopped before its commit",
			setup: func(t *testing.T, fx *regateFixture) {
				runGit(t, "-C", fx.worktree, "switch", "--quiet", "-c", "side")
				writeGateFile(t, fx.worktree, "README.md", "the side's readme\n", 0o644)
				runGit(t, "-C", fx.worktree, "add", "README.md")
				gateCommit(t, fx.worktree, "the side's readme")
				runGit(t, "-C", fx.worktree, "switch", "--quiet", "feat/auth")
				runGit(t, "-C", fx.worktree, "-c", "user.name=t", "-c", "user.email=t@example.com", "merge", "--quiet", "--no-commit", "--no-ff", "side")
			},
			want: func(fx *regateFixture) []string {
				return []string{"a merge is in progress in " + fx.worktree + ": finish it with git commit, or undo it with git merge " +
					"--abort, and run ralph org regate --branch feat/auth again"}
			},
			notWant: []string{"uncommitted"}},
		{name: "an untracked file",
			setup: func(t *testing.T, fx *regateFixture) { writeGateFile(t, fx.worktree, "new.txt", "x\n", 0o644) },
			want: func(fx *regateFixture) []string {
				return []string{"the feature worktree " + fx.worktree + " has 1 uncommitted change(s) or untracked file(s)",
					"(a file the scripts write belongs in .gitignore), push the commit to origin's feat/auth and run ralph org regate " +
						"--branch feat/auth again:\n  ?? new.txt"}
			}},
		{name: "no default branch",
			setup: func(t *testing.T, fx *regateFixture) {
				runGit(t, "-C", fx.main, "symbolic-ref", "--delete", "refs/remotes/origin/HEAD")
				runGit(t, "-C", fx.main, "update-ref", "-d", "refs/remotes/origin/main")
				runGit(t, "-C", fx.main, "update-ref", "-d", "refs/heads/main")
			},
			want: func(fx *regateFixture) []string {
				return []string{"no default branch found in " + fx.worktree, "run git remote set-head origin --auto there and run " +
					"ralph org regate --branch feat/auth again"}
			}},
		{name: "the static script is missing",
			setup: func(t *testing.T, fx *regateFixture) {
				runGit(t, "-C", fx.worktree, "rm", "--quiet", GateStaticScript)
				gateCommit(t, fx.worktree, "drop the static script")
			},
			want: func(fx *regateFixture) []string {
				return []string{filepath.Join(fx.worktree, "scripts", "run-static-verify.sh") + " does not exist",
					"add it, commit it, push the commit to origin's feat/auth, and run ralph org regate --branch feat/auth again"}
			}},
		{name: "the test script is not executable",
			setup: func(t *testing.T, fx *regateFixture) {
				if err := os.Chmod(filepath.Join(fx.worktree, filepath.FromSlash(GateTestScript)), 0o644); err != nil {
					t.Fatal(err)
				}
				runGit(t, "-C", fx.worktree, "add", GateTestScript)
				gateCommit(t, fx.worktree, "the test script loses its x bit")
			},
			want: func(fx *regateFixture) []string {
				return []string{filepath.Join(fx.worktree, "scripts", "run-test.sh") + " is not executable",
					"commit the change, push the commit to origin's feat/auth, and run ralph org regate --branch feat/auth again"}
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			if tc.setup != nil {
				tc.setup(t, fx)
			}
			branch := tc.branch
			if branch == "" {
				branch = "feat/auth"
			}
			before := fx.snapshot(t)
			res, err := fx.o.Regate(context.Background(), RegateParams{Branch: branch, Env: fx.env})
			fx.assertRefused(t, res, err, before, tc.want(fx), tc.notWant)
		})
	}
}

// TestRegate_RefusalOfARunningOrgNamesTheBranchItRuns covers AC1's refusal
// of a branch whose org runs, word for word: while the org runs bound to
// the branch, the refusal names the leader's ralph org gate; once the org_id
// was started again for another feature, its leader's gate checks that
// feature's branch, so the refusal names that branch instead and only says
// to wait for the disband.
func TestRegate_RefusalOfARunningOrgNamesTheBranchItRuns(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, fx *regateFixture)
		want  string
	}{
		{"the org runs bound to the branch", func(t *testing.T, fx *regateFixture) {
			fx.appendEvents(t, boundReserveEvent("auth", fx.f, "internal/auth/"))
			seedSeat(t, fx.o, "auth", LeaderIdentity, LeaderIdentity, "pane-2")
		}, `org: regate: the branch feat/auth is bound to org_id "auth", which is running: while it runs, its leader runs ralph ` +
			"org gate --org-id auth in the feature worktree, so wait for the org to be disbanded and run ralph org regate --branch " +
			"feat/auth then"},
		{"the org_id runs again for another feature", func(t *testing.T, fx *regateFixture) {
			fx.appendEvents(t, boundReserveEvent("auth", testFeature("split-2", "billing", "0123456789ab"), "internal/billing/"))
		}, `org: regate: the branch feat/auth is bound to org_id "auth", which is running, but now for another feature, on the ` +
			"branch feat/billing: wait for the org to be disbanded and run ralph org regate --branch feat/auth then"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			tc.setup(t, fx)
			before := fx.snapshot(t)
			res, err := fx.o.Regate(context.Background(), RegateParams{Branch: "feat/auth", Env: fx.env})
			fx.assertRefused(t, res, err, before, nil, nil)
			if res.Refused != tc.want {
				t.Fatalf("refusal =\n%q\nwant\n%q", res.Refused, tc.want)
			}
		})
	}
}

// TestRegate_RefusedWhileAnotherRegateHoldsTheLock covers D7's half of
// AC1: while another regate holds regate.lock, Regate retries it for 1s and
// then refuses, writing nothing into the lock; once the lock is free it runs.
func TestRegate_RefusedWhileAnotherRegateHoldsTheLock(t *testing.T) {
	fx := newRegateFixture(t)
	other, err := acquireHeldFileLock(fx.stateDir, regateLockFile, "regate", 0)
	if err != nil {
		t.Fatal(err)
	}
	before := fx.snapshot(t)
	begin := time.Now()
	res, err := fx.regate(context.Background())
	if took := time.Since(begin); took < time.Second {
		t.Errorf("regate refused after %s; want it to retry the lock for 1s (D7)", took)
	}
	if err != nil || !strings.Contains(res.Refused, "org: regate: another ralph org regate runs on this ledger ("+
		regateLockPathIn(fx.stateDir)+" is held): one regate runs at a time, so wait for it to end and run ralph org regate "+
		"--branch feat/auth again") || !reflect.DeepEqual(res, RegateResult{Refused: res.Refused}) {
		t.Fatalf("regate = %+v, %v; want the held-lock refusal", res, err)
	}
	if after := fx.snapshot(t); after != before || fx.calls(t) != nil {
		t.Fatalf("a refused regate changed %+v to %+v or ran %v", before, after, fx.calls(t))
	}
	if marker := fileOrAbsent(t, regateLockPathIn(fx.stateDir)); marker != "" {
		t.Fatalf("a refused regate wrote %q into the lock another regate holds", marker)
	}
	if err := other.release(); err != nil {
		t.Fatal(err)
	}
	if res := fx.mustRegate(t); res.Outcome != RegateOutcomeGated || !res.Passed {
		t.Fatalf("regate after the lock was released = %+v; want a recorded pass", res)
	}
}

// TestGateWording_RefusalsThatAskForACommit pins, word for word, the shared
// refusals that ask for a commit (gateCheckScript's four, gateDirtyRefusal)
// in both wordings: Gate's read as they did before ralph org regate shared
// them, and regate's ask for a push of the commit too, since regate gates
// only what origin has (gateWording.afterCommit).
func TestGateWording_RefusalsThatAskForACommit(t *testing.T) {
	root := t.TempDir()
	writeGateFile(t, root, "scripts/plain.sh", "#!/bin/sh\n", 0o644)
	mkdirAll(t, filepath.Join(root, "scripts", "dir.sh"))
	writeGateFile(t, root, "file", "x\n", 0o644)
	missing, unreadable := filepath.Join(root, "scripts", "missing.sh"), filepath.Join(root, "file", "x.sh")
	dir, plain := filepath.Join(root, "scripts", "dir.sh"), filepath.Join(root, "scripts", "plain.sh")
	for _, tc := range []struct {
		name string
		w    gateWording
		want []string
	}{
		{"gate", gateWords, []string{
			"org: gate: " + missing + " does not exist: the gate runs ./scripts/missing.sh in the feature worktree, so add it, " +
				"commit it, and run the gate again",
			"org: gate: " + unreadable + " cannot be read (not a directory): fix it, commit it, and run the gate again",
			"org: gate: " + dir + " is not a regular file: the gate runs ./scripts/dir.sh, so make it the script, commit it, and " +
				"run the gate again",
			"org: gate: " + plain + " is not executable: run chmod +x scripts/plain.sh, commit the change, and run the gate again",
			"org: gate: the feature worktree " + root + " has 1 uncommitted change(s) or untracked file(s): commit or remove them " +
				"(a file the scripts write belongs in .gitignore) and run the gate again:\n  ?? new.txt",
		}},
		{"regate", regateWords("feat/auth"), []string{
			"org: regate: " + missing + " does not exist: the gate runs ./scripts/missing.sh in the feature worktree, so add it, " +
				"commit it, push the commit to origin's feat/auth, and run ralph org regate --branch feat/auth again",
			"org: regate: " + unreadable + " cannot be read (not a directory): fix it, commit it, push the commit to origin's " +
				"feat/auth, and run ralph org regate --branch feat/auth again",
			"org: regate: " + dir + " is not a regular file: the gate runs ./scripts/dir.sh, so make it the script, commit it, push " +
				"the commit to origin's feat/auth, and run ralph org regate --branch feat/auth again",
			"org: regate: " + plain + " is not executable: run chmod +x scripts/plain.sh, commit the change, push the commit to " +
				"origin's feat/auth, and run ralph org regate --branch feat/auth again",
			"org: regate: the feature worktree " + root + " has 1 uncommitted change(s) or untracked file(s): commit or remove " +
				"them (a file the scripts write belongs in .gitignore), push the commit to origin's feat/auth and run ralph org " +
				"regate --branch feat/auth again:\n  ?? new.txt",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := []string{
				gateCheckScript(root, "scripts/missing.sh", tc.w),
				gateCheckScript(root, "file/x.sh", tc.w),
				gateCheckScript(root, "scripts/dir.sh", tc.w),
				gateCheckScript(root, "scripts/plain.sh", tc.w),
				gateDirtyRefusal(root, []string{"?? new.txt"}, tc.w),
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("refusal %d =\n%q\nwant\n%q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestRegate_FetchFailures covers AC2's fetch half: a fetch that fails
// refuses with git's stderr; origin with no feature branch is told apart
// (deleted after the merge, or never pushed), but only for the feature
// branch; a fetch past the bound is stopped and refused. Each records
// nothing and changes neither the worktree nor origin.
func TestRegate_FetchFailures(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(t *testing.T, fx *regateFixture)
		want    func(fx *regateFixture) []string
		notWant []string
	}{
		{name: "origin is gone",
			setup: func(t *testing.T, fx *regateFixture) {
				if err := os.RemoveAll(fx.origin); err != nil {
					t.Fatal(err)
				}
			},
			want: func(fx *regateFixture) []string {
				return []string{"git fetch origin +refs/heads/main:refs/remotes/origin/main in " + fx.worktree + " failed (exit 128), " +
					"so nothing was recorded", "run ralph org regate --branch feat/auth again; git said:\n  ", "does not appear to be a git repository"}
			},
			notWant: []string{"origin has no branch"}},
		{name: "origin has no feature branch",
			setup: func(t *testing.T, fx *regateFixture) {
				runGit(t, "-C", fx.origin, "update-ref", "-d", "refs/heads/feat/auth")
			},
			want: func(fx *regateFixture) []string {
				return []string{"origin has no branch feat/auth: it was deleted after the merge, or never pushed",
					"when its PR is merged there is nothing to regate", "git -C " + fx.worktree + " push origin feat/auth"}
			},
			notWant: []string{"git said"}},
		// Only the feature branch's missing ref is told apart: a default
		// branch origin does not have is a failure like any other.
		{name: "origin has no default branch",
			setup: func(t *testing.T, fx *regateFixture) {
				runGit(t, "-C", fx.origin, "update-ref", "-d", "refs/heads/main")
			},
			want: func(fx *regateFixture) []string {
				return []string{"git fetch origin +refs/heads/main:refs/remotes/origin/main in " + fx.worktree + " failed",
					"couldn't find remote ref refs/heads/main"}
			},
			notWant: []string{"origin has no branch"}},
		{name: "the fetch runs past the bound",
			setup: func(t *testing.T, fx *regateFixture) {
				timeout := regateGitTimeout
				regateGitTimeout = 300 * time.Millisecond
				t.Cleanup(func() { regateGitTimeout = timeout })
				// An upload-pack that never answers, with its output away from
				// git's, so nothing holds the fetch's pipes once git is stopped.
				writeGateFile(t, fx.out, "slow-upload-pack", "#!/bin/sh\nexec >/dev/null 2>&1 </dev/null\nsleep 3\n", 0o755)
				runGit(t, "-C", fx.main, "config", "remote.origin.uploadpack", filepath.Join(fx.out, "slow-upload-pack"))
			},
			want: func(fx *regateFixture) []string {
				return []string{"git fetch origin +refs/heads/main:refs/remotes/origin/main in " + fx.worktree + " ran past 300ms and " +
					"was stopped, so nothing was recorded", "check the network and origin's credentials"}
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			tc.setup(t, fx)
			before := fx.snapshot(t)
			res, err := fx.regate(context.Background())
			fx.assertRefused(t, res, err, before, tc.want(fx), tc.notWant)
		})
	}
}

// TestRegate_AFetchEndedByASignalRecordsNothing: a fetch a signal ends (here
// a reference-transaction hook kills git as the fetch moves
// refs/remotes/origin/main) is told as what it is, as an interrupt of the
// fetch is: an error that says nothing was recorded, with the worktree and
// origin as they were and no script run.
func TestRegate_AFetchEndedByASignalRecordsNothing(t *testing.T) {
	fx := newRegateFixture(t)
	fx.pushToOrigin(t, "main")
	fx.setHooks(t, "reference-transaction", "kill -KILL $PPID\n")
	before := fx.snapshot(t)
	res, err := fx.regate(context.Background())
	if want := "org: regate: git fetch origin +refs/heads/main:refs/remotes/origin/main in " + fx.worktree + " was ended by a " +
		"signal (killed), so nothing was recorded: run ralph org regate --branch feat/auth again"; err == nil || err.Error() != want {
		t.Fatalf("regate = %+v, %v\nwant the error %s", res, err, want)
	}
	if !reflect.DeepEqual(res, RegateResult{}) {
		t.Errorf("regate = %+v with the error; want an empty result", res)
	}
	if after := fx.snapshot(t); after != before {
		t.Errorf("the regate changed the ledger, the worktree, or origin:\nbefore %+v\nafter  %+v", before, after)
	}
	if calls := fx.calls(t); calls != nil {
		t.Errorf("scripts run = %v; want none", calls)
	}
	fx.assertUnlocked(t)
}

// TestRegate_RefusesAHeadThatIsNotOriginsTip covers AC2's D4 half: after the
// fetch, a HEAD that is not origin's tip of the branch is refused, saying
// which side has commits the other does not: a commit that was not pushed,
// a push from elsewhere (which only the fetch shows), or both.
func TestRegate_RefusesAHeadThatIsNotOriginsTip(t *testing.T) {
	commitLocally := func(t *testing.T, fx *regateFixture) {
		t.Helper()
		writeGateFile(t, fx.worktree, "internal/auth/more.go", "package auth\n", 0o644)
		runGit(t, "-C", fx.worktree, "add", ".")
		gateCommit(t, fx.worktree, "not pushed")
	}
	cases := []struct {
		name  string
		setup func(t *testing.T, fx *regateFixture) (pushed string)
		want  func(fx *regateFixture, head, pushed string) []string
	}{
		{"a commit that was not pushed",
			func(t *testing.T, fx *regateFixture) string { commitLocally(t, fx); return "" },
			func(fx *regateFixture, head, _ string) []string {
				return []string{"HEAD " + head + " of feat/auth in " + fx.worktree + " has commits that origin's tip",
					"so push them (git -C " + fx.worktree + " push origin feat/auth) and run ralph org regate --branch feat/auth again"}
			}},
		{"a push from elsewhere",
			func(t *testing.T, fx *regateFixture) string { return fx.pushToOrigin(t, "feat/auth") },
			func(fx *regateFixture, head, pushed string) []string {
				return []string{"origin's tip " + pushed + " of feat/auth has commits that HEAD " + head + " in " + fx.worktree + " does not",
					"git -C " + fx.worktree + " merge --ff-only refs/remotes/origin/feat/auth"}
			}},
		{"both",
			func(t *testing.T, fx *regateFixture) string {
				commitLocally(t, fx)
				return fx.pushToOrigin(t, "feat/auth")
			},
			func(fx *regateFixture, head, pushed string) []string {
				return []string{"HEAD " + head + " of feat/auth in " + fx.worktree + " and origin's tip " + pushed + " of the branch have " +
					"diverged", "push the result, and run ralph org regate --branch feat/auth again"}
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			pushed := tc.setup(t, fx)
			head, _ := fx.tips(t)
			before := fx.snapshot(t)
			res, err := fx.regate(context.Background())
			fx.assertRefused(t, res, err, before, tc.want(fx, head, pushed), nil)
			if pushed != "" {
				if got := gitOut(t, fx.worktree, "rev-parse", "refs/remotes/origin/feat/auth"); got != pushed {
					t.Fatalf("refs/remotes/origin/feat/auth = %s after the regate; want the fetched %s", got, pushed)
				}
			}
		})
	}
}

// TestRegateRemoteEnv pins GIT_TERMINAL_PROMPT=0 (D8) on the git commands
// Regate runs against origin, and LC_ALL=C on the fetches only, the one
// whose messages Regate reads: each comes after the caller's environment, so
// it wins over a value the caller sets, and the caller's slice is not
// written to.
func TestRegateRemoteEnv(t *testing.T) {
	env := make([]string, 2, 8)
	copy(env, []string{"GIT_TERMINAL_PROMPT=1", "LC_ALL=ja_JP.UTF-8"})
	if got, want := regateRemoteEnv(env), []string{"GIT_TERMINAL_PROMPT=1", "LC_ALL=ja_JP.UTF-8", "GIT_TERMINAL_PROMPT=0"}; !slices.Equal(got, want) {
		t.Fatalf("regateRemoteEnv = %q; want %q", got, want)
	}
	got := regateFetchEnv(env)
	if want := []string{"GIT_TERMINAL_PROMPT=1", "LC_ALL=ja_JP.UTF-8", "GIT_TERMINAL_PROMPT=0", "LC_ALL=C"}; !slices.Equal(got, want) {
		t.Fatalf("regateFetchEnv = %q; want %q", got, want)
	}
	if gateEnvValue(got, "GIT_TERMINAL_PROMPT") != "0" || gateEnvValue(got, "LC_ALL") != "C" {
		t.Fatalf("regateFetchEnv = %q; want GIT_TERMINAL_PROMPT=0 and LC_ALL=C to win", got)
	}
	if spare := env[:4]; spare[2] != "" || spare[3] != "" {
		t.Fatalf("regateRemoteEnv or regateFetchEnv wrote %q into the caller's spare capacity", spare[2:])
	}
	if got := regateFetchEnv(nil); !slices.Equal(got, []string{"GIT_TERMINAL_PROMPT=0", "LC_ALL=C"}) {
		t.Fatalf("regateFetchEnv(nil) = %q", got)
	}
}

// regateRecordPgid is a hook line that appends the process group of the
// hook, which is git's, to the file name in $GATE_TEST_OUT.
func regateRecordPgid(name string) string {
	return "ps -o pgid= -p $$ >> \"$GATE_TEST_OUT/" + name + "\"\n"
}

// assertRegateGitGroup checks the process groups regateRecordPgid wrote to
// name: at least one, and each the test's own (ralph's) when inTestGroup,
// else none of them (a group of git's own, regateBoundedGit's ownGroup).
func (fx *regateFixture) assertRegateGitGroup(t *testing.T, name string, inTestGroup bool) {
	t.Helper()
	pgids := strings.Fields(fileOrAbsent(t, filepath.Join(fx.out, name)))
	if len(pgids) == 0 {
		t.Fatalf("no process group in %s", name)
	}
	test := strconv.Itoa(syscall.Getpgrp())
	for _, pgid := range pgids {
		if inTestGroup && pgid != test {
			t.Errorf("%s: git ran in the process group %s; want ralph's, %s", name, pgid, test)
		}
		if !inTestGroup && pgid == test {
			t.Errorf("%s: git ran in ralph's process group %s; want one of its own", name, pgid)
		}
	}
}

// TestRegate_FetchRunsWithNoTerminalPrompt covers AC2's environment: both
// fetches run with GIT_TERMINAL_PROMPT=0 and LC_ALL=C even when the caller
// sets other values, as an upload-pack that saves its environment shows. They
// run in ralph's process group, where ssh can ask for a key on the terminal.
func TestRegate_FetchRunsWithNoTerminalPrompt(t *testing.T) {
	fx := newRegateFixture(t)
	writeGateFile(t, fx.out, "upload-pack", "#!/bin/sh\nenv >> \"$GATE_TEST_OUT/fetch-env\"\necho fetch >> \"$GATE_TEST_OUT/fetches\"\n"+
		regateRecordPgid("fetch-pgid")+"exec git upload-pack \"$@\"\n", 0o755)
	runGit(t, "-C", fx.main, "config", "remote.origin.uploadpack", filepath.Join(fx.out, "upload-pack"))
	fx.env = append(fx.env, "GIT_TERMINAL_PROMPT=1", "LC_ALL=ja_JP.UTF-8")
	if res := fx.mustRegate(t); res.Outcome != RegateOutcomeGated || !res.Passed {
		t.Fatalf("regate = %+v; want a recorded pass", res)
	}
	if fetches := strings.Fields(fileOrAbsent(t, filepath.Join(fx.out, "fetches"))); len(fetches) != 2 {
		t.Fatalf("upload-pack ran %d times; want once per fetch, 2", len(fetches))
	}
	vars := strings.Split(fileOrAbsent(t, filepath.Join(fx.out, "fetch-env")), "\n")
	for _, w := range []string{"GIT_TERMINAL_PROMPT=0", "LC_ALL=C"} {
		if !slices.Contains(vars, w) {
			t.Errorf("the fetch's environment has no %s", w)
		}
	}
	for _, notW := range []string{"GIT_TERMINAL_PROMPT=1", "LC_ALL=ja_JP.UTF-8"} {
		if slices.Contains(vars, notW) {
			t.Errorf("the fetch's environment has the caller's %s", notW)
		}
	}
	fx.assertRegateGitGroup(t, "fetch-pgid", true)
}

// TestRegate_UpToDateDoesNothing covers AC3's first half: with HEAD
// containing origin's main and a gate_passed of HEAD on that tip, against
// origin/main or the local main, Regate runs nothing, records nothing,
// writes no log, and changes neither the worktree nor origin; the result
// names the log of that gate_passed. The fetch gives the tip it compares
// with, also when the local origin/main is gone.
func TestRegate_UpToDateDoesNothing(t *testing.T) {
	cases := []struct {
		name    string
		baseRef string
		setup   func(t *testing.T, fx *regateFixture)
	}{
		{"a gate_passed against origin/main", "refs/remotes/origin/main", nil},
		{"a gate_passed against the local main", "refs/heads/main", nil},
		{"the local origin/main is gone", "refs/remotes/origin/main", func(t *testing.T, fx *regateFixture) {
			runGit(t, "-C", fx.main, "update-ref", "-d", "refs/remotes/origin/main")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			head, tip := fx.tips(t)
			fx.appendEvents(t, moGate("auth", "feat/auth", head, tip, tc.baseRef, ""))
			if tc.setup != nil {
				tc.setup(t, fx)
			}
			before := fx.snapshot(t)
			res := fx.mustRegate(t)
			want := RegateResult{Outcome: RegateOutcomeUpToDate, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth", Head: head,
				Base: tip, BaseRef: "refs/remotes/origin/main", LogPath: "/state/gates/auth.log"}
			if !reflect.DeepEqual(res, want) {
				t.Fatalf("regate = %+v\nwant     %+v", res, want)
			}
			if after := fx.snapshot(t); after != before {
				t.Fatalf("an up-to-date regate changed %+v\nto %+v", before, after)
			}
			if _, err := os.Stat(filepath.Join(fx.stateDir, gateLogDir)); !errors.Is(err, os.ErrNotExist) || fx.calls(t) != nil {
				t.Fatalf("an up-to-date regate wrote a log (stat: %v) or ran %v", err, fx.calls(t))
			}
			if got := gitOut(t, fx.worktree, "rev-parse", "refs/remotes/origin/main"); got != tip {
				t.Fatalf("refs/remotes/origin/main = %s; want the fetched %s", got, tip)
			}
			fx.assertUnlocked(t)
		})
	}
}

// TestRegate_GatesWhenNoRecordLetsItStop covers AC9 and AC3's second half:
// with HEAD containing origin's main but no readable gate_passed of HEAD on
// that tip as the branch's latest record (none, a gate_failed, another head,
// another base, another default branch), or a matching one that a record
// that cannot be read or a corrupt line follows, Regate runs the scripts
// without merging, records a gate_passed, pushes nothing, and merge-order
// then classifies the branch ready.
func TestRegate_GatesWhenNoRecordLetsItStop(t *testing.T) {
	cases := []struct {
		name   string
		lines  func(fx *regateFixture, head, tip string) []any
		before string // feat/auth's merge-order class before the regate
	}{
		{"no gate record", nil, MergeClassNoGate},
		{"a gate_failed on HEAD", func(_ *regateFixture, head, tip string) []any {
			return []any{moGate("auth", "feat/auth", head, tip, "refs/remotes/origin/main", GateReasonTestFailed)}
		}, MergeClassFailed},
		{"a gate_passed on another head", func(_ *regateFixture, _, tip string) []any {
			return []any{moGate("auth", "feat/auth", tip, tip, "refs/remotes/origin/main", "")}
		}, MergeClassRegate},
		{"a gate_passed on another base", func(_ *regateFixture, head, _ string) []any {
			return []any{moGate("auth", "feat/auth", head, strings.Repeat("1", 40), "refs/remotes/origin/main", "")}
		}, MergeClassRebase},
		{"a gate_passed against another default branch", func(_ *regateFixture, head, tip string) []any {
			return []any{moGate("auth", "feat/auth", head, tip, "refs/remotes/origin/trunk", "")}
		}, MergeClassRebase},
		{"a gate record that cannot be read after a matching gate_passed", func(_ *regateFixture, head, tip string) []any {
			return []any{moGate("auth", "feat/auth", head, tip, "refs/remotes/origin/main", ""), moGarbled("auth", EventGatePassed)}
		}, MergeClassUnreadable},
		{"a corrupt line after a matching gate_passed", func(_ *regateFixture, head, tip string) []any {
			return []any{moGate("auth", "feat/auth", head, tip, "refs/remotes/origin/main", ""), regateCorruptLine}
		}, MergeClassUnreadable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			head, tip := fx.tips(t)
			if tc.lines != nil {
				fx.appendLines(t, tc.lines(fx, head, tip)...)
			}
			if class := fx.mergeClass(t); class != tc.before {
				t.Fatalf("merge-order class before the regate = %s; want %s", class, tc.before)
			}
			before := fx.snapshot(t)
			res := fx.mustRegate(t)
			if res.Outcome != RegateOutcomeGated || !res.Passed || res.Head != head || res.Base != tip || res.Refused != "" {
				t.Fatalf("regate = %+v; want a recorded pass on %s with base %s", res, head, tip)
			}
			if calls := fx.calls(t); !slices.Equal(calls, []string{"static", "test"}) {
				t.Fatalf("scripts run = %v; want static then test", calls)
			}
			last := lastEvent(t, fx.o)
			if r, ok := gateRecordFromDetails(last.Details); last.Event != EventGatePassed || !ok || r.Via != gateViaRegate || r.Head != head {
				t.Fatalf("last event = %+v; want a gate_passed of %s via regate", last, head)
			}
			if after := fx.snapshot(t); after.head != before.head || after.status != before.status || after.originRefs != before.originRefs {
				t.Fatalf("a gate-only regate changed the worktree or origin:\nbefore %+v\nafter  %+v", before, after)
			}
			if class := fx.mergeClass(t); class != MergeClassReady {
				t.Fatalf("merge-order class after the regate = %s; want ready", class)
			}
			fx.assertUnlocked(t)
		})
	}
}

// TestRegate_GateOnlyRecordsAGatePassedViaRegate covers AC9's record: one
// org-level gate_passed of the binding's org, the Details of Gate's record
// with via=regate before log=, the log under <state-dir>/gates/ named after
// the org, and both scripts run in the worktree's root.
func TestRegate_GateOnlyRecordsAGatePassedViaRegate(t *testing.T) {
	fx := newRegateFixture(t)
	head, tip := fx.tips(t)
	res := fx.mustRegate(t)
	logPath := filepath.Join(fx.stateDir, gateLogDir, regateTestLog)
	want := RegateResult{Outcome: RegateOutcomeGated, Passed: true, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth",
		Head: head, Base: tip, BaseRef: "refs/remotes/origin/main", LogPath: logPath}
	if !reflect.DeepEqual(res, want) {
		t.Fatalf("regate = %+v\nwant     %+v", res, want)
	}
	for _, name := range []string{"static", "test"} {
		if pwd := strings.TrimSpace(fileOrAbsent(t, filepath.Join(fx.out, "pwd-"+name))); pwd != fx.worktree {
			t.Errorf("%s ran in %s; want the worktree root %s", name, pwd, fx.worktree)
		}
	}
	events := fx.gateEvents(t)
	wantDetails := "head=" + head + " base=" + tip + " base_ref=refs/remotes/origin/main branch=feat/auth by=alice herdr_pane=unset " +
		"via=regate log=" + strconv.Quote(logPath)
	if len(events) != 1 || events[0].Event != EventGatePassed || events[0].OrgID != "auth" || events[0].SeatID != "" ||
		events[0].TS != "2026-10-16T12:00:00Z" || events[0].Details != wantDetails || events[0].DryRun {
		t.Fatalf("gate events = %+v\nwant one gate_passed of auth with Details %q", events, wantDetails)
	}
	log := fileOrAbsent(t, logPath)
	for _, w := range []string{
		"# ralph org regate\n# worktree: " + fx.worktree + "\n# branch: feat/auth\n# head: " + head + "\n# base: refs/remotes/origin/main " + tip + "\n",
		"==> ralph org regate: running ./scripts/run-static-verify.sh\n", "==> ralph org regate: ./scripts/run-static-verify.sh exited 0\n",
		"==> ralph org regate: running ./scripts/run-test.sh\n", "==> ralph org regate: ./scripts/run-test.sh exited 0\n",
	} {
		if !strings.Contains(log, w) {
			t.Errorf("log does not have %q:\n%s", w, log)
		}
	}
	if strings.Contains(log, "ralph org gate") {
		t.Errorf("regate's log names ralph org gate:\n%s", log)
	}
	if st := gitOut(t, fx.worktree, "status", "--porcelain", "--untracked-files=all", "--ignored"); st != "" {
		t.Fatalf("the regate wrote in the worktree:\n%s", st)
	}
}

// TestRegate_RecordsWithTheOrgOfTheLatestBinding: of two bindings of the
// branch, both of disbanded orgs, the later one names the org the record and
// the log are of.
func TestRegate_RecordsWithTheOrgOfTheLatestBinding(t *testing.T) {
	fx := newRegateFixture(t)
	fx.rebind(t, fx.worktree)
	res := fx.mustRegate(t)
	if res.OrgID != "auth-2" || res.LogPath != filepath.Join(fx.stateDir, gateLogDir, "auth-2-20261016T120000Z.log") || !res.Passed {
		t.Fatalf("regate = %+v; want a pass recorded for auth-2", res)
	}
	if events := fx.gateEvents(t); len(events) != 1 || events[0].OrgID != "auth-2" {
		t.Fatalf("gate events = %+v; want one of auth-2", events)
	}
}

// TestRegate_GateOnlyFailuresRecordGateFailed covers AC9's failures: a
// failing script, a commit during the scripts, and a file a script leaves
// each record one gate_failed via regate with Gate's reason, Tail, and Dirty,
// and refuse nothing.
func TestRegate_GateOnlyFailuresRecordGateFailed(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(t *testing.T, fx *regateFixture)
		reason string
		calls  []string
		tail   []string
		dirty  []string
	}{
		{"the static script fails", func(t *testing.T, fx *regateFixture) { fx.setExit(t, "static", 3) },
			GateReasonStaticFailed, []string{"static"}, gateStubTail("static"), nil},
		{"the test script fails", func(t *testing.T, fx *regateFixture) { fx.setExit(t, "test", 1) },
			GateReasonTestFailed, []string{"static", "test"}, gateStubTail("test"), nil},
		{"HEAD moves during the scripts", func(t *testing.T, fx *regateFixture) {
			fx.hook(t, "test", "git -c user.name=t -c user.email=t@example.com -c commit.gpgsign=false commit --quiet --allow-empty -m moved\n")
		}, GateReasonHeadMoved, []string{"static", "test"}, nil, nil},
		{"a script leaves a file in the worktree", func(t *testing.T, fx *regateFixture) { fx.hook(t, "static", "echo x > untracked.txt\n") },
			GateReasonTreeChanged, []string{"static", "test"}, nil, []string{"?? untracked.txt"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			tc.setup(t, fx)
			head, tip := fx.tips(t)
			res := fx.mustRegate(t)
			logPath := filepath.Join(fx.stateDir, gateLogDir, regateTestLog)
			want := RegateResult{Outcome: RegateOutcomeGated, Reason: tc.reason, OrgID: "auth", Worktree: fx.worktree, Branch: "feat/auth",
				Head: head, Base: tip, BaseRef: "refs/remotes/origin/main", LogPath: logPath, Tail: tc.tail, Dirty: tc.dirty}
			if !reflect.DeepEqual(res, want) {
				t.Fatalf("regate = %#v\nwant     %#v", res, want)
			}
			if calls := fx.calls(t); !slices.Equal(calls, tc.calls) {
				t.Fatalf("scripts run = %v; want %v", calls, tc.calls)
			}
			wantDetails := "reason=" + tc.reason + " head=" + head + " base=" + tip + " base_ref=refs/remotes/origin/main branch=feat/auth " +
				"by=alice herdr_pane=unset via=regate log=" + strconv.Quote(logPath)
			if events := fx.gateEvents(t); len(events) != 1 || events[0].Event != EventGateFailed || events[0].OrgID != "auth" ||
				events[0].Details != wantDetails {
				t.Fatalf("gate events = %+v\nwant one gate_failed of auth with Details %q", events, wantDetails)
			}
			if class := fx.mergeClass(t); class != MergeClassFailed {
				t.Fatalf("merge-order class after the regate = %s; want failed", class)
			}
		})
	}
}

// TestRegate_NothingRecordedWhenTheBranchChangedWhileTheScriptsRan covers
// AC8 (D6): when, while the scripts run, the org starts again, another org
// binds the branch, or the org binds it again, Regate records nothing and
// refuses, keeping the log of the run.
func TestRegate_NothingRecordedWhenTheBranchChangedWhileTheScriptsRan(t *testing.T) {
	cases := []struct {
		name   string
		events func(fx *regateFixture) []ManifestEvent
		want   string
	}{
		{"the org starts again", func(*regateFixture) []ManifestEvent {
			return []ManifestEvent{reserveEvent("auth", "internal/other/")}
		}, `(org_id "auth" is running again)`},
		{"another org binds the branch", func(fx *regateFixture) []ManifestEvent {
			f := *fx.f
			f.Digest = "fedcba987654"
			return []ManifestEvent{boundReserveEvent("auth-2", &f, "internal/auth/"), {OrgID: "auth-2", Event: EventDisbanded}}
		}, `(it is now bound to org_id "auth-2"'s feature "auth" of split plan "split-1" (digest "fedcba987654"`},
		{"the org binds the branch again", func(fx *regateFixture) []ManifestEvent {
			return []ManifestEvent{boundReserveEvent("auth", fx.f, "internal/auth/"), {OrgID: "auth", Event: EventDisbanded}}
		}, `(org_id "auth" bound it again)`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			setRegateHook(t, &regateTestHooks.beforeRecord, func() { fx.appendEvents(t, tc.events(fx)...) })
			res, err := fx.regate(context.Background())
			if err != nil || res.Outcome != "" || res.Passed ||
				!strings.HasPrefix(res.Refused, "org: regate: nothing was recorded, since the branch feat/auth changed while the scripts ran "+tc.want) {
				t.Fatalf("regate = %+v, %v; want the record refused with %q", res, err, tc.want)
			}
			if !strings.Contains(res.Refused, "check it with ralph org merge-order and ralph org status, and run ralph org regate --branch feat/auth again") {
				t.Errorf("refusal %q has no next step", res.Refused)
			}
			if events := fx.gateEvents(t); len(events) != 0 {
				t.Fatalf("gate events = %+v; want none", events)
			}
			if _, err := os.Stat(res.LogPath); res.LogPath == "" || err != nil {
				t.Fatalf("log %q: %v; want the log of the run kept", res.LogPath, err)
			}
			fx.assertUnlocked(t)
		})
	}
}

// TestRegate_MarksTheLockWithTheOrgItWorksOn covers the regate side of D9:
// from the first check on, regate.lock is held and holds the marker of the
// org, branch, and worktree Regate works on and its pid, which
// readRegateMarker reads; it is released when Regate returns, its marker
// left as it was, and the record is appended with the lock still held.
func TestRegate_MarksTheLockWithTheOrgItWorksOn(t *testing.T) {
	fx := newRegateFixture(t)
	path := regateLockPathIn(fx.stateDir)
	type seen struct {
		held  bool
		orgID string
		ok    bool
		raw   string
	}
	look := func() seen {
		held, err := fileLockHeld(path)
		if err != nil {
			t.Fatal(err)
		}
		orgID, ok := readRegateMarker(path)
		return seen{held, orgID, ok, fileOrAbsent(t, path)}
	}
	var first, recorded seen
	setRegateHook(t, &regateTestHooks.afterFirstCheck, func() { first = look() })
	setRegateHook(t, &regateTestHooks.afterRecord, func() { recorded = look() })
	if res := fx.mustRegate(t); !res.Passed {
		t.Fatalf("regate = %+v; want a pass", res)
	}
	if !first.held || first.orgID != "auth" || !first.ok {
		t.Fatalf("after the first check: %+v; want regate.lock held with auth's marker", first)
	}
	var m regateMarker
	if err := json.Unmarshal([]byte(first.raw), &m); err != nil ||
		m != (regateMarker{OrgID: "auth", Branch: "feat/auth", Worktree: fx.worktree, PID: os.Getpid()}) {
		t.Fatalf("marker %q read as %+v (%v); want auth, feat/auth, the worktree, and this pid", first.raw, m, err)
	}
	if recorded != first {
		t.Fatalf("right after the record: %+v; want the lock still held with the same marker %+v", recorded, first)
	}
	last := look()
	if last.held || last.raw != first.raw {
		t.Fatalf("after Regate returned: %+v; want the lock released and the marker left as it was", last)
	}
}

// TestRegate_EmptiesAnEarlierMarkerBeforeTheManifestLock covers D9's
// emptied marker: right after it takes regate.lock, before it waits for the
// manifest lock, Regate empties the marker an earlier regate left, so no
// spawn of that org reads it during the wait; once the manifest lock is
// free it goes on and writes its own.
func TestRegate_EmptiesAnEarlierMarkerBeforeTheManifestLock(t *testing.T) {
	fx := newRegateFixture(t)
	path := regateLockPathIn(fx.stateDir)
	writeGateFile(t, fx.stateDir, regateLockFile, `{"org_id":"old","branch":"feat/old","worktree":"/old","pid":1}`+"\n", 0o644)
	release := holdManifestLock(t, fx.o)
	type outcome struct {
		res RegateResult
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		res, err := fx.regate(context.Background())
		done <- outcome{res, err}
	}()
	// The wait for the manifest lock lasts up to manifestLockTimeout (5s);
	// the marker must be empty well before it ends.
	deadline := time.Now().Add(3 * time.Second)
	for fileOrAbsent(t, path) != "" {
		if time.Now().After(deadline) {
			t.Fatalf("regate.lock still holds %q 3s into the wait for the manifest lock; want it emptied at once", fileOrAbsent(t, path))
		}
		time.Sleep(10 * time.Millisecond)
	}
	if held, err := fileLockHeld(path); err != nil || !held {
		t.Fatalf("regate.lock held = %v (%v) while Regate waits for the manifest lock; want it held", held, err)
	}
	if orgID, ok := readRegateMarker(path); ok {
		t.Fatalf("readRegateMarker = %q during the wait; want no marker", orgID)
	}
	release()
	got := <-done
	if got.err != nil || !got.res.Passed {
		t.Fatalf("regate = %+v, %v; want a pass once the manifest lock is free", got.res, got.err)
	}
	if orgID, ok := readRegateMarker(path); !ok || orgID != "auth" {
		t.Fatalf("readRegateMarker = %q, %v after the regate; want its own marker, auth", orgID, ok)
	}
}

// TestRegate_ARefusedFirstCheckLeavesNoMarker: a first check that refuses
// leaves regate.lock empty, the marker an earlier regate left gone, so no
// spawn reads an org from it while the lock is still held: for a branch no
// binding names and for an org that runs.
func TestRegate_ARefusedFirstCheckLeavesNoMarker(t *testing.T) {
	for _, tc := range []struct {
		name   string
		branch string
		setup  func(t *testing.T, fx *regateFixture)
	}{
		{"a branch no binding names", "feat/other", nil},
		{"an org that runs", "feat/auth", func(t *testing.T, fx *regateFixture) {
			fx.appendEvents(t, boundReserveEvent("auth", fx.f, "internal/auth/"))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			if tc.setup != nil {
				tc.setup(t, fx)
			}
			path := regateLockPathIn(fx.stateDir)
			writeGateFile(t, fx.stateDir, regateLockFile, `{"org_id":"old","branch":"feat/old","worktree":"/old","pid":1}`+"\n", 0o644)
			res, err := fx.o.Regate(context.Background(), RegateParams{Branch: tc.branch, Env: fx.env})
			if err != nil || res.Refused == "" {
				t.Fatalf("regate = %+v, %v; want a refusal", res, err)
			}
			if raw := fileOrAbsent(t, path); raw != "" {
				t.Fatalf("regate.lock holds %q after a refused first check; want it empty", raw)
			}
			if orgID, ok := readRegateMarker(path); ok {
				t.Fatalf("readRegateMarker = %q; want no marker", orgID)
			}
		})
	}
}

// TestReadRegateMarker: a marker is read only from a JSON line with an
// org_id; a file that is missing, empty, or holds anything else gives none.
func TestReadRegateMarker(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		name, content, orgID string
		ok                   bool
	}{
		{"a marker", `{"org_id":"auth","branch":"feat/auth","worktree":"/w","pid":7}` + "\n", "auth", true},
		{"empty", "", "", false},
		{"not JSON", "auth\n", "", false},
		{"no org_id", `{"branch":"feat/auth"}` + "\n", "", false},
		{"cut off", `{"org_id":"au`, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, strings.ReplaceAll(tc.name, " ", "-"))
			writeGateFile(t, dir, filepath.Base(path), tc.content, 0o644)
			if orgID, ok := readRegateMarker(path); orgID != tc.orgID || ok != tc.ok {
				t.Fatalf("readRegateMarker(%q) = %q, %v; want %q, %v", tc.content, orgID, ok, tc.orgID, tc.ok)
			}
		})
	}
	if orgID, ok := readRegateMarker(filepath.Join(dir, "missing")); ok || orgID != "" {
		t.Fatalf("readRegateMarker of a missing file = %q, %v; want none", orgID, ok)
	}
}

// TestRegate_Interrupted: a ctx done before the checks finish, while a
// script runs, right before the record's lock is taken, or under it right
// before the append records nothing, returns an error that says so (naming
// the log once the scripts started), and releases regate.lock.
func TestRegate_Interrupted(t *testing.T) {
	logPath := func(fx *regateFixture) string { return filepath.Join(fx.stateDir, gateLogDir, regateTestLog) }
	afterScripts := func(fx *regateFixture) string {
		return "org: regate: interrupted (context canceled): the scripts were stopped and nothing was recorded; the output so far is in " +
			logPath(fx)
	}
	check := func(t *testing.T, fx *regateFixture, before string, err error, want string) {
		t.Helper()
		if err == nil || err.Error() != want {
			t.Fatalf("regate = %v\nwant     %s", err, want)
		}
		if after := fileOrAbsent(t, fx.o.Manifest.Path()); after != before {
			t.Fatalf("an interrupted regate changed the manifest:\nbefore %s\nafter  %s", before, after)
		}
		fx.assertUnlocked(t)
	}

	t.Run("before the checks", func(t *testing.T) {
		fx := newRegateFixture(t)
		before := fileOrAbsent(t, fx.o.Manifest.Path())
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		res, err := fx.regate(ctx)
		check(t, fx, before, err, "org: regate: interrupted (context canceled) before the scripts ran: nothing was recorded")
		if !reflect.DeepEqual(res, RegateResult{}) || fx.calls(t) != nil {
			t.Fatalf("regate = %+v and ran %v; want nothing", res, fx.calls(t))
		}
	})

	t.Run("while a script runs", func(t *testing.T) {
		grace := gateKillGrace
		gateKillGrace = 300 * time.Millisecond
		t.Cleanup(func() { gateKillGrace = grace })
		fx := newRegateFixture(t)
		started, release := filepath.Join(fx.out, "started"), filepath.Join(fx.out, "release")
		fx.hook(t, "static", "touch '"+started+"'\nwhile [ ! -f '"+release+"' ]; do sleep 0.05; done\n")
		t.Cleanup(func() { _ = os.WriteFile(release, nil, 0o644) })
		before := fileOrAbsent(t, fx.o.Manifest.Path())
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() {
			_, err := fx.regate(ctx)
			done <- err
		}()
		waitForFile(t, started)
		cancel()
		var err error
		select {
		case err = <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("regate did not return within 10s of the interrupt")
		}
		check(t, fx, before, err, afterScripts(fx))
		if calls := fx.calls(t); !slices.Equal(calls, []string{"static"}) {
			t.Fatalf("scripts run = %v; want only static", calls)
		}
	})

	t.Run("right before the record's lock", func(t *testing.T) {
		fx := newRegateFixture(t)
		before := fileOrAbsent(t, fx.o.Manifest.Path())
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		setRegateHook(t, &regateTestHooks.beforeRecord, cancel)
		_, err := fx.regate(ctx)
		check(t, fx, before, err, afterScripts(fx))
	})

	t.Run("under the record's lock, right before the append", func(t *testing.T) {
		fx := newRegateFixture(t)
		before := fileOrAbsent(t, fx.o.Manifest.Path())
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// The clock is read twice: for the log's name (createGateLog) and for
		// the event's TS under the lock (recordRegate), where the interrupt
		// lands.
		reads := 0
		fx.o.Now = func() time.Time {
			if reads++; reads >= 2 {
				cancel()
			}
			return regateTestClock
		}
		_, err := fx.regate(ctx)
		check(t, fx, before, err, afterScripts(fx))
		if reads != 2 {
			t.Fatalf("the clock was read %d times; want 2 (the log's name, then the event's TS)", reads)
		}
	})
}

// TestRegate_ErrorsOfTheSharedHelpersSayRegate covers the errors of the
// helpers Regate shares with Gate: from Regate they start with "org:
// regate: ". A corrupt index in the feature worktree passes the branch checks
// and then makes git status fail (gateDirtyLines); a file where the gates
// directory goes makes createGateLog fail. Nothing is run or recorded.
func TestRegate_ErrorsOfTheSharedHelpersSayRegate(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, fx *regateFixture)
		want  func(fx *regateFixture) string // what the error starts with
	}{
		{"git status", func(t *testing.T, fx *regateFixture) {
			index := gitOut(t, fx.worktree, "rev-parse", "--path-format=absolute", "--git-path", "index")
			if err := os.WriteFile(index, []byte("garbage"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, func(fx *regateFixture) string {
			return "org: regate: git status --porcelain --untracked-files=all in " + fx.worktree + ": exit 128: "
		}},
		{"the gate log", func(t *testing.T, fx *regateFixture) {
			writeGateFile(t, fx.stateDir, gateLogDir, "not a directory\n", 0o644)
		}, func(fx *regateFixture) string {
			return "org: regate: create the gate log dir " + filepath.Join(fx.stateDir, gateLogDir) + ": "
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fx := newRegateFixture(t)
			tc.setup(t, fx)
			before := fileOrAbsent(t, fx.o.Manifest.Path())
			res, err := fx.regate(context.Background())
			if want := tc.want(fx); err == nil || !strings.HasPrefix(err.Error(), want) {
				t.Fatalf("regate = %v; want an error starting with %q", err, want)
			}
			if res.Outcome != "" || res.Passed || res.Refused != "" || res.LogPath != "" || fx.calls(t) != nil {
				t.Fatalf("regate = %+v and ran %v; want nothing run", res, fx.calls(t))
			}
			if after := fileOrAbsent(t, fx.o.Manifest.Path()); after != before {
				t.Fatalf("regate changed the manifest:\nbefore %s\nafter  %s", before, after)
			}
			fx.assertUnlocked(t)
		})
	}
}
