package org

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// The progress section in the leader's task at StartFeature and at `ralph org
// reseat` (plan docs/plans/archive/2026-11-15-reseat-leader-progress.md, slice
// S2: AC1, AC2, AC4, AC7). The fixture is featureStart (feature_test.go) on
// fake drivers and fake worktrees, with a real git repository at the feature
// worktree where a test needs commits. reseat_progress_test.go covers the text
// of the section; these tests cover what StartFeature reads and when it adds
// the section.

// progressStartRepo makes dir a git repository whose default branch main holds
// one commit and whose branch fix/auth-core, checked out, holds a commit for
// each of subjects on top of it. It returns what `git log --format='%h %s'
// main..HEAD` prints, newest first (nil for no subject).
func progressStartRepo(t *testing.T, dir string, subjects ...string) []string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	isolateGitEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	runGit(t, "-c", "init.defaultBranch=main", "init", "--quiet", dir)
	gateCommit(t, dir, "base")
	runGit(t, "-C", dir, "switch", "--quiet", "--create", "fix/auth-core")
	for _, subject := range subjects {
		gateCommit(t, dir, subject)
	}
	if len(subjects) == 0 {
		return nil
	}
	return strings.Split(gitOut(t, dir, "log", "--format=%h %s", "main..HEAD"), "\n")
}

// progressStartRepoAheadOfOrigin makes dir a git repository whose origin's
// default branch (refs/remotes/origin/main, which refs/remotes/origin/HEAD
// names) holds one commit, whose local main holds one more that origin does
// not have yet (a plan committed on main and not pushed), and whose branch
// fix/auth-core, made from the local main and checked out, holds a commit for
// each of subjects. It returns the feature's own commits as `git log
// --format='%h %s' main..HEAD` prints them, newest first (nil for no
// subject).
func progressStartRepoAheadOfOrigin(t *testing.T, dir string, subjects ...string) []string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not on PATH")
	}
	isolateGitEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	runGit(t, "-c", "init.defaultBranch=main", "init", "--quiet", dir)
	gateCommit(t, dir, "base")
	runGit(t, "-C", dir, "update-ref", "refs/remotes/origin/main", "HEAD")
	runGit(t, "-C", dir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	gateCommit(t, dir, "docs: a plan on the local main only")
	runGit(t, "-C", dir, "switch", "--quiet", "--create", "fix/auth-core")
	for _, subject := range subjects {
		gateCommit(t, dir, subject)
	}
	if got := gitOut(t, dir, "rev-list", "--count", "refs/remotes/origin/main..main"); got != "1" {
		t.Fatalf("the fixture's local main is %s commit(s) ahead of origin's, want 1", got)
	}
	if len(subjects) == 0 {
		return nil
	}
	return strings.Split(gitOut(t, dir, "log", "--format=%h %s", "main..HEAD"), "\n")
}

// TestReseatProgress_CommitsLeaveOutTheLocalDefaultBranchAheadOfOrigin pins
// finding 2 of the run 5 cross-review: the feature worktree is made from the
// local default branch (scripts/ralph-worktree.sh ensure), and the gate's base
// prefers origin's, so a commit the local main has and origin does not was
// counted as the branch's work, and a first start got a section that named
// it. Only the feature's own commits count.
func TestReseatProgress_CommitsLeaveOutTheLocalDefaultBranchAheadOfOrigin(t *testing.T) {
	dir := t.TempDir()
	want := progressStartRepoAheadOfOrigin(t, dir, "feat: one", "feat: two")
	commits, total, readError := readReseatProgressCommits(dir)
	if readError != "" {
		t.Fatalf("readReseatProgressCommits failed: %s", readError)
	}
	if total != 2 || strings.Join(commits, "\n") != strings.Join(want, "\n") {
		t.Errorf("commits = %q (total %d), want only the feature's two commits %q", commits, total, want)
	}
	for _, line := range commits {
		if strings.Contains(line, "on the local main only") {
			t.Errorf("commits = %q, want no commit that the local main holds", commits)
		}
	}

	// A file named HEAD in the worktree is not read as a revision (the cycle 4
	// self-review, C4-1: without `--` git stops with "ambiguous argument").
	if err := os.WriteFile(filepath.Join(dir, "HEAD"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if commits, total, readError := readReseatProgressCommits(dir); readError != "" || total != 2 || strings.Join(commits, "\n") != strings.Join(want, "\n") {
		t.Errorf("with a file named HEAD in the worktree, readReseatProgressCommits = %q, %d, %q; want the feature's two commits %q", commits, total, readError, want)
	}

	// A branch with no commit of its own, made from that main, has none.
	fresh := t.TempDir()
	progressStartRepoAheadOfOrigin(t, fresh)
	if commits, total, readError := readReseatProgressCommits(fresh); readError != "" || total != 0 || len(commits) != 0 {
		t.Errorf("readReseatProgressCommits of a fresh branch = %q, %d, %q; want no commit", commits, total, readError)
	}
}

// leaderTaskOf returns the ## タスク section of the leader prompt that the
// latest spawn of auth-core's leader wrote.
func (st *featureStart) leaderTaskOf(t *testing.T) string {
	t.Helper()
	promptPath, err := st.o.promptFilePath("auth-core", LeaderIdentity)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("read the leader's prompt: %v", err)
	}
	_, rest, ok := strings.Cut(string(data), "\n## タスク\n\n")
	task, _, closed := strings.Cut(rest, "\n\n## スター型トポロジのルール\n")
	if !ok || !closed {
		t.Fatalf("expected the leader's prompt to have a ## タスク section followed by ## スター型トポロジのルール, got:\n%s", data)
	}
	return task
}

// freshLeaderTask is the task featureLeaderTask builds for auth-core: what a
// start with no progress gives the leader, with nothing after it. The split
// plan is read from the path StartFeature resolves, as it reads it.
func (st *featureStart) freshLeaderTask(t *testing.T) string {
	t.Helper()
	path, _, err := ResolveSplitPlanPath(st.stateDir, st.plan)
	if err != nil {
		t.Fatalf("ResolveSplitPlanPath: %v", err)
	}
	plan, err := LoadSplitPlan(path)
	if err != nil {
		t.Fatalf("LoadSplitPlan: %v", err)
	}
	feature, ok := plan.Feature("auth-core")
	if !ok {
		t.Fatal("the split plan has no feature auth-core")
	}
	return featureLeaderTask(plan, feature, st.stateDir, st.worktree("auth-core"), "fix/auth-core", config.ApprovalModePerPlan, config.ReviewerConfig{})
}

// progressSectionOf returns what a leader's task holds after fresh, the task
// featureLeaderTask builds, and a blank line: the progress section. It fails
// when the task does not start with fresh and a blank line.
func progressSectionOf(t *testing.T, task, fresh string) string {
	t.Helper()
	rest, ok := strings.CutPrefix(task, fresh+"\n\n")
	if !ok {
		t.Fatalf("the task does not hold the section after the fresh task and a blank line:\n%s", task)
	}
	return rest
}

// leaderRestarted stops auth-core's leader and starts it again with p, which
// must end spawned, and returns the result.
func (st *featureStart) leaderRestarted(t *testing.T, p StartFeatureParams) StartFeatureResult {
	t.Helper()
	st.stopSeat(t, LeaderIdentity)
	return st.mustStart(t, p, SpawnOutcomeSpawned)
}

// appendGatePassed appends a passed gate record of auth-core for head to the
// ledger.
func (st *featureStart) appendGatePassed(t *testing.T, head string) {
	t.Helper()
	ev := progressGateEvent("auth-core", "2026-11-15T02:00:00Z", progressGateRecord(head, "", "", "/state/gate.log"))
	if err := st.o.Manifest.Append(ev); err != nil {
		t.Fatalf("append the gate record: %v", err)
	}
}

// TestReseatProgress_StartFeatureTellsARestartedLeaderWhatTheOrgHolds covers
// AC1 and AC2 at the org layer for a `start --plan` that starts the leader
// again (no reseat): the leader's task ends with the progress section that
// holds the approval, the last gate, the open inbox item of the org (not the
// one of another org), and the two commits of the branch with their total,
// after the task featureLeaderTask builds, unchanged. It names no reseat.
func TestReseatProgress_StartFeatureTellsARestartedLeaderWhatTheOrgHolds(t *testing.T) {
	st := newFeatureStart(t)
	st.o.Inbox = NewInboxStore(st.stateDir)
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	fresh := st.freshLeaderTask(t)
	if got := st.leaderTaskOf(t); got != fresh {
		t.Fatalf("the first start's task is not the fresh task:\n%s", got)
	}

	ref := approveFeaturePlan(t, st.o, "auth-core", st.worktree("auth-core"))
	st.appendGatePassed(t, progressHeadA)
	for _, in := range []InboxEscalation{
		{OrgID: "auth-core", Type: "QUESTION", TaskID: "auth-core", Body: "TYPE: QUESTION\nTASK_ID: auth-core\n\nどちらの案にしますか"},
		{OrgID: "other", Type: "QUESTION", TaskID: "other", Body: "TYPE: QUESTION\nTASK_ID: other\n\nこの件は載らない"},
	} {
		if _, err := st.o.Inbox.Escalate(in); err != nil {
			t.Fatalf("escalate %+v: %v", in, err)
		}
	}
	commits := progressStartRepo(t, st.worktree("auth-core"), "feat: add the token store", "test: pin the token store")
	if len(commits) != 2 {
		t.Fatalf("the fixture has %d commits, want 2", len(commits))
	}

	st.leaderRestarted(t, st.params("auth-core"))
	section := progressSectionOf(t, st.leaderTaskOf(t), fresh)
	block := progressSectionLines(t, section)
	want := []string{
		"計画の承認: " + ref.Path + " sha256:" + ref.Digest,
		"最後の gate: pass(gate_passed)head " + progressHeadA + " base " + progressBase + " branch feat/auth-core(2026-11-15T02:00:00Z) log /state/gate.log",
		"片付いていない受信箱の件: 1 件(古い順に 1 件)",
		"- e1 QUESTION auth-core open: どちらの案にしますか",
		"branch の commit: 2 件(新しい順に 2 件)",
		"- " + commits[0],
		"- " + commits[1],
	}
	if len(block) != len(want) {
		t.Fatalf("the section has %d lines, want %d:\n%s", len(block), len(want), section)
	}
	for i, w := range want {
		if i == 0 {
			if !strings.HasPrefix(block[i], w+" (承認 ") {
				t.Errorf("line %d = %q, want it to start with %q", i, block[i], w+" (承認 ")
			}
			continue
		}
		if block[i] != w {
			t.Errorf("line %d = %q, want %q", i, block[i], w)
		}
	}
}

// TestReseatProgress_StartFeatureAddsNothingForAnOrgWithNoProgress covers AC2
// and AC7: a start with no approval, no gate record, no open inbox item, and
// no commit on the branch leaves the task exactly as featureLeaderTask builds
// it, also when git or the inbox cannot be read, since a read that failed is
// not progress.
func TestReseatProgress_StartFeatureAddsNothingForAnOrgWithNoProgress(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T, st *featureStart)
	}{
		{"a worktree that is not a git repository", func(t *testing.T, st *featureStart) {
			isolateGitEnv(t)
			t.Setenv("GIT_CEILING_DIRECTORIES", st.root)
		}},
		{"a branch with no commit past the default branch", func(t *testing.T, st *featureStart) {
			progressStartRepo(t, st.worktree("auth-core"))
		}},
		{"a branch made from a local default branch that is ahead of origin's", func(t *testing.T, st *featureStart) {
			progressStartRepoAheadOfOrigin(t, st.worktree("auth-core"))
		}},
		{"a repository with no default branch", func(t *testing.T, st *featureStart) {
			progressStartRepo(t, st.worktree("auth-core"))
			runGit(t, "-C", st.worktree("auth-core"), "branch", "--move", "main", "trunk")
		}},
		{"an inbox that cannot be read", func(t *testing.T, st *featureStart) {
			st.o.Inbox = NewInboxStore(st.stateDir)
			if err := os.Mkdir(st.o.Inbox.Path(), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"an inbox with only another org's item", func(t *testing.T, st *featureStart) {
			st.o.Inbox = NewInboxStore(st.stateDir)
			if _, err := st.o.Inbox.Escalate(InboxEscalation{OrgID: "other", Type: "QUESTION", TaskID: "other", Body: "TYPE: QUESTION\nTASK_ID: other\n\nq"}); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := newFeatureStart(t)
			if err := os.MkdirAll(st.worktree("auth-core"), 0o755); err != nil {
				t.Fatal(err)
			}
			tc.setup(t, st)
			st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
			if got, want := st.leaderTaskOf(t), st.freshLeaderTask(t); got != want {
				t.Errorf("the task =\n%s\nwant featureLeaderTask's\n%s", got, want)
			}
		})
	}
}

// TestReseatProgress_StartFeatureCarriesOnWhenGitFails covers AC4: with an
// approval in the ledger and a worktree git cannot read, the start succeeds
// and the section has the approval and one line that says the branch could not
// be read, with no commit.
func TestReseatProgress_StartFeatureCarriesOnWhenGitFails(t *testing.T) {
	st := newFeatureStart(t)
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	approveFeaturePlan(t, st.o, "auth-core", st.worktree("auth-core"))
	isolateGitEnv(t)
	t.Setenv("GIT_CEILING_DIRECTORIES", st.root)

	res := st.leaderRestarted(t, st.params("auth-core"))
	if res.Spawn.Err != nil {
		t.Fatalf("the start failed: %v", res.Spawn.Err)
	}
	block := progressSectionLines(t, progressSectionOf(t, st.leaderTaskOf(t), st.freshLeaderTask(t)))
	if len(block) != 2 || !strings.HasPrefix(block[0], "計画の承認: "+testPlanRel+" sha256:") ||
		!strings.HasPrefix(block[1], "branch の commit: 読めなかった(git ") || !strings.Contains(block[1], "not a git repository") {
		t.Errorf("the section = %q, want the approval and one line for the git that failed", block)
	}
}

// TestReseatProgress_StartFeatureStopsAGitThatDoesNotAnswer covers D4 and R5:
// the git commands that read the commits are held to reseatProgressGitTimeout
// together, a git that sleeps is stopped when it runs out, the section says so
// in one line, and the start goes on. The stub git replaces itself with sleep,
// so that stopping it closes the pipes.
func TestReseatProgress_StartFeatureStopsAGitThatDoesNotAnswer(t *testing.T) {
	st := newFeatureStart(t)
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	approveFeaturePlan(t, st.o, "auth-core", st.worktree("auth-core"))

	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte("#!/bin/sh\nexec sleep 8\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	old := reseatProgressGitTimeout
	reseatProgressGitTimeout = 300 * time.Millisecond
	t.Cleanup(func() { reseatProgressGitTimeout = old })

	began := time.Now()
	st.leaderRestarted(t, st.params("auth-core"))
	if took := time.Since(began); took > 4*time.Second {
		t.Errorf("the start took %s, want git stopped after %s", took, reseatProgressGitTimeout)
	}
	block := progressSectionLines(t, progressSectionOf(t, st.leaderTaskOf(t), st.freshLeaderTask(t)))
	if len(block) != 2 || !strings.HasPrefix(block[0], "計画の承認: ") ||
		!strings.HasPrefix(block[1], "branch の commit: 読めなかった(git ") ||
		!strings.Contains(block[1], "context deadline exceeded") || !strings.HasSuffix(block[1], "300ms で打ち切った)") {
		t.Errorf("the section = %q, want the approval and one line for the git that was stopped", block)
	}
}

// TestReseatProgress_StartFeatureSaysWhenTheInboxCannotBeRead: with an
// approval in the ledger and an inbox file that cannot be read (a directory),
// the start succeeds and the section says so in one line.
func TestReseatProgress_StartFeatureSaysWhenTheInboxCannotBeRead(t *testing.T) {
	st := newFeatureStart(t)
	st.o.Inbox = NewInboxStore(st.stateDir)
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	approveFeaturePlan(t, st.o, "auth-core", st.worktree("auth-core"))
	if err := os.Mkdir(st.o.Inbox.Path(), 0o755); err != nil {
		t.Fatal(err)
	}
	isolateGitEnv(t)
	t.Setenv("GIT_CEILING_DIRECTORIES", st.root)

	st.leaderRestarted(t, st.params("auth-core"))
	block := progressSectionLines(t, progressSectionOf(t, st.leaderTaskOf(t), st.freshLeaderTask(t)))
	if len(block) != 3 || !strings.HasPrefix(block[0], "計画の承認: ") ||
		!strings.HasPrefix(block[1], "受信箱: 読めなかった(org: inbox: read "+st.o.Inbox.Path()+": ") ||
		!strings.HasPrefix(block[2], "branch の commit: 読めなかった(") {
		t.Errorf("the section = %q, want the approval, the inbox that could not be read, and the branch that could not be read", block)
	}
}

// TestReseatProgress_ReseatNamesItselfAndTheSeatsAfterTheLeader covers AC1's
// last sentence: a reseat of the leader and the reviewer starts the leader with
// the section that holds the approval, the one commit, and `今回の立て直し`
// with this reseat's id and the reviewer as the one seat started after the
// leader; the implementer, which this reseat leaves running, is not named.
func TestReseatProgress_ReseatNamesItselfAndTheSeatsAfterTheLeader(t *testing.T) {
	st := newReseatOrg(t)
	st.h.closeRemoves = true
	commits := progressStartRepo(t, st.worktree("auth-core"), "feat: add the token store")
	fresh := st.freshLeaderTask(t)

	res := st.mustReseat(t, st.reseatParams(LeaderIdentity, "reviewer"))
	block := progressSectionLines(t, progressSectionOf(t, st.leaderTaskOf(t), fresh))
	if len(block) != 4 || !strings.HasPrefix(block[0], "計画の承認: "+testPlanRel+" sha256:") ||
		block[1] != "branch の commit: 1 件(新しい順に 1 件)" || block[2] != "- "+commits[0] ||
		block[3] != "今回の立て直し: reseat="+res.Plan.ID+"、leader のあとに起動する予定の座席: reviewer" {
		t.Errorf("the section = %q, want the approval, the commit, and this reseat %s with the reviewer after the leader", block, res.Plan.ID)
	}
}

// TestReseatProgress_ASecondReseatNamesTheFirstAsThePreviousOne: the `reseated`
// event the first reseat wrote is the previous reseat of the second, which
// names no seat after the leader as it starts the leader alone. A reseat of the
// reviewer alone does not start the leader, so the leader's prompt stays as it
// was.
func TestReseatProgress_ASecondReseatNamesTheFirstAsThePreviousOne(t *testing.T) {
	st := newReseatOrg(t)
	st.h.closeRemoves = true
	progressStartRepo(t, st.worktree("auth-core"))
	fresh := st.freshLeaderTask(t)

	first := st.mustReseat(t, st.reseatParams(LeaderIdentity, "reviewer"))
	second := st.mustReseat(t, st.reseatParams(LeaderIdentity))
	block := progressSectionLines(t, progressSectionOf(t, st.leaderTaskOf(t), fresh))
	if len(block) != 3 || !strings.HasPrefix(block[0], "計画の承認: ") ||
		block[1] != "今回の立て直し: reseat="+second.Plan.ID+"、leader のあとに起動する予定の座席: なし(leader だけ)" ||
		!strings.HasPrefix(block[2], "前の立て直し: ") || !strings.Contains(block[2], "id="+first.Plan.ID+" seats=") {
		t.Errorf("the section = %q, want the approval, this reseat %s with no seat after the leader, and the previous reseat %s", block, second.Plan.ID, first.Plan.ID)
	}

	promptPath, err := st.o.promptFilePath("auth-core", LeaderIdentity)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatal(err)
	}
	st.mustReseat(t, st.reseatParams("reviewer"))
	after, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("a reseat of the reviewer alone rewrote the leader's prompt:\n%s\nwas\n%s", after, before)
	}
}

// TestReseatProgress_CommitsIgnoreLogShowSignature covers the commit reader
// under `log.showSignature=true` (self-review L1): for a signed commit git log
// would print the signature check's lines before the commit's own, and the
// reader would take them for commits. The fake gpg program signs and verifies
// anything, and answers a verify with a "gpg: Good signature" line; the test
// first checks that a plain git log prints that line here, so that it is not
// passing for want of a signature, then that the section lists one line per
// commit and counts three.
func TestReseatProgress_CommitsIgnoreLogShowSignature(t *testing.T) {
	dir := t.TempDir()
	progressStartRepo(t, dir)
	gpg := filepath.Join(t.TempDir(), "fake-gpg")
	script := `#!/bin/sh
for a in "$@"; do
  if [ "$a" = "--verify" ]; then
    echo '[GNUPG:] NEWSIG'
    echo '[GNUPG:] GOODSIG 0000000000000000 fake <fake@example.com>'
    echo 'gpg: Good signature from "fake"' >&2
    echo '[GNUPG:] VALIDSIG 0000000000000000000000000000000000000000 2026-11-15 1 0 4 0 1 8 00 0000000000000000000000000000000000000000'
    exit 0
  fi
done
cat >/dev/null
echo '[GNUPG:] SIG_CREATED D 1 8 00 1 0000000000000000000000000000000000000000' >&2
printf '%s\n' '-----BEGIN PGP SIGNATURE-----' '' 'ZmFrZQ==' '-----END PGP SIGNATURE-----'
`
	if err := os.WriteFile(gpg, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, "-C", dir, "config", "gpg.program", gpg)
	signed := func(subject string) {
		t.Helper()
		runGit(t, "-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "--quiet", "--allow-empty", "-S", "-m", subject)
	}
	signed("feat: signed one")
	gateCommit(t, dir, "feat: plain two")
	signed("feat: signed three")
	runGit(t, "-C", dir, "config", "log.showSignature", "true")

	if plain := gitOut(t, dir, "log", "--format=%h %s", "main..HEAD"); !strings.Contains(plain, "gpg: Good signature") {
		t.Fatalf("the fixture's git log prints no signature line, so this test would pass with or without the flag that hides it:\n%s", plain)
	}
	want := strings.Split(gitOut(t, dir, "log", "--no-show-signature", "--format=%h %s", "main..HEAD"), "\n")
	commits, total, readError := readReseatProgressCommits(dir)
	if readError != "" {
		t.Fatalf("readReseatProgressCommits failed: %s", readError)
	}
	if total != 3 || strings.Join(commits, "\n") != strings.Join(want, "\n") {
		t.Fatalf("commits = %q (total %d), want the three commits %q", commits, total, want)
	}
	section := reseatProgressSection(reseatProgress{}, reseatProgressSupplied{Commits: commits, CommitTotal: total})
	block := progressSectionLines(t, section)
	if len(block) != 4 || block[0] != "branch の commit: 3 件(新しい順に 3 件)" || strings.Contains(section, "gpg:") {
		t.Errorf("the section = %q, want the heading and one line per commit, with no signature line", block)
	}
}
