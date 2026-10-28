package cli

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org merge` (plan
// docs/plans/active/2026-10-17-org-auto-merge.md, S4: AC2, AC3, AC6, AC10
// and AC12 at the command line). The org layer's checks, the merge, the
// read-back, the removal of an auto-merge request, and the record are covered
// by internal/org/merge_test.go and merge_run_test.go with a fake
// MergeOrderExec. These unit-test printMergeResult and run the command with
// the real git against a bare origin (newMergeOrderFixture, whose git wrapper
// answers git remote get-url origin with https://github.com/o/r.git) and a
// fake gh (mergeCLIFakeGH) that answers each call from a reply file, so no
// test runs a real gh or reaches a network. None runs in parallel: they set
// PATH and chdir.

// mergeCLIFakeGH is the fake gh of these tests: it appends one line per call
// to $MERGE_CLI_GH_LOG (GH_PROMPT_DISABLED's value, then its arguments),
// names the call (pr-list, pr-view, pr-checks, api-repo, api-graphql,
// api-rules, api-ruleset, repo-view, pr-merge, pr-read-back, pr-disable-auto,
// ...), prints $MERGE_CLI_GH_REPLIES/<name>.out on stdout and <name>.err on
// stderr when they exist, and exits with the number in <name>.rc, else 0. A
// call it cannot name fails with exit 97. It uses only shell builtins, since
// PATH holds no other tool.
const mergeCLIFakeGH = `#!/bin/sh
printf 'GH_PROMPT_DISABLED=%s %s\n' "$GH_PROMPT_DISABLED" "$*" >> "$MERGE_CLI_GH_LOG"
case "$*" in
"pr list "*) reply=pr-list ;;
"pr view "*" --json state,headRefOid,baseRefName,mergeCommit,autoMergeRequest") reply=pr-read-back ;;
"pr view "*) reply=pr-view ;;
"pr checks "*) reply=pr-checks ;;
"pr merge "*" --disable-auto "*) reply=pr-disable-auto ;;
"pr merge "*) reply=pr-merge ;;
"repo view "*) reply=repo-view ;;
"api "*" graphql") reply=api-graphql ;;
"api "*"/rules/branches/"*) reply=api-rules ;;
"api "*"/rulesets/"*) reply=api-ruleset ;;
"api "*"/protection/"*) reply=api-protection ;;
"api "*"/branches/"*) reply=api-branch ;;
"api "*) reply=api-repo ;;
*) printf 'fake gh: no reply for: %s\n' "$*" >&2; exit 97 ;;
esac
dir=$MERGE_CLI_GH_REPLIES
if [ -f "$dir/$reply.out" ]; then
  while IFS= read -r line || [ -n "$line" ]; do printf '%s\n' "$line"; done < "$dir/$reply.out"
fi
if [ -f "$dir/$reply.err" ]; then
  while IFS= read -r line || [ -n "$line" ]; do printf '%s\n' "$line" >&2; done < "$dir/$reply.err"
fi
rc=0
if [ -f "$dir/$reply.rc" ]; then
  read -r rc < "$dir/$reply.rc"
fi
exit "$rc"
`

const (
	// mergeCLIRepoArg is the -R of every gh pr call, origin's repository.
	mergeCLIRepoArg = "github.com/o/r"
	// mergeCLIMergeCommit is the merge commit the fake gh reads back.
	mergeCLIMergeCommit = "0123456789abcdef0123456789abcdef01234567"
	// mergeCLIProtection is CheckMergeEnforcement's Detail for the fake gh's
	// ruleset, which the PROTECTION: line prints.
	mergeCLIProtection = `the ruleset "protect main" (id 1) of main, the default branch of origin's repository o/r, requires branches ` +
		`to be up to date before merging with its 1 required status check, and you cannot bypass it (current_user_can_bypass: never)`
)

// mergeCLIFixture is a repository whose origin is a bare repository with
// the branch feat/a pushed one commit ahead of main, a ledger that binds
// feat/a to org a, records its gate_passed of head on main, and records org a
// disbanded, and the fake gh with replies under which ralph org merge merges
// PR #7 of feat/a.
type mergeCLIFixture struct {
	*mergeOrderFixture
	replies string // the fake gh's reply files
	head    string // feat/a's tip, the head the gate passed
	main    string // main's tip, the gate's base
}

// newMergeCLIFixture builds a mergeCLIFixture. With mainLedger, its ledger is
// the work checkout's .harness/state/org, the one org.ResolveOrgStateDir
// finds from a linked worktree of it; else a temp dir of its own.
func newMergeCLIFixture(t *testing.T, mainLedger bool) *mergeCLIFixture {
	t.Helper()
	fx := newMergeCLIFixtureOrgRunning(t, mainLedger)
	appendSeedEvent(t, fx.stateDir, org.ManifestEvent{OrgID: "a", Event: org.EventDisbanded})
	return fx
}

// newMergeCLIFixtureOrgRunning is newMergeCLIFixture with org a not
// disbanded: it still runs the feature of feat/a.
func newMergeCLIFixtureOrgRunning(t *testing.T, mainLedger bool) *mergeCLIFixture {
	t.Helper()
	fx := newMergeCLIFixtureRepo(t, mainLedger)
	fx.main = fx.tip(t, "main")
	fx.head = fx.branch(t, "feat/a")
	fx.push(t, "feat/a")
	fx.bind(t, "a", "a", "feat/a")
	fx.gateAndReply(t)
	return fx
}

// newMergeCLIFixtureRepo is the repository, its bare origin, and the fake gh
// of a mergeCLIFixture, with no feat/a and nothing in the ledger yet.
func newMergeCLIFixtureRepo(t *testing.T, mainLedger bool) *mergeCLIFixture {
	t.Helper()
	fx := &mergeCLIFixture{mergeOrderFixture: newMergeOrderFixture(t, false)}
	if mainLedger {
		fx.stateDir = filepath.Join(fx.work, ".harness", "state", "org")
	}
	fx.replies = filepath.Join(filepath.Dir(fx.bin), "gh-replies")
	if err := os.MkdirAll(fx.replies, 0o755); err != nil {
		t.Fatal(err)
	}
	writeStubScript(t, filepath.Join(fx.bin, "gh"), mergeCLIFakeGH)
	t.Setenv("MERGE_CLI_GH_REPLIES", fx.replies)
	t.Setenv("MERGE_CLI_GH_LOG", fx.ghLog)
	t.Setenv("MERGE_ORDER_ORIGIN_URL", "https://github.com/o/r.git")
	return fx
}

// gateAndReply records org a's gate_passed of fx.head on fx.main for feat/a
// and gives the fake gh the replies under which ralph org merge merges PR #7
// of feat/a at fx.head.
func (fx *mergeCLIFixture) gateAndReply(t *testing.T) {
	t.Helper()
	fx.gate(t, "a", "feat/a", "", fx.head, fx.main)

	fx.reply(t, "pr-list", `[{"number":7,"state":"OPEN","url":"https://github.com/o/r/pull/7","headRefOid":"`+fx.head+
		`","baseRefName":"main","isCrossRepository":false}]`)
	fx.reply(t, "pr-view", `{"autoMergeRequest":null,"baseRefName":"main","headRefOid":"`+fx.head+
		`","mergeCommit":null,"state":"OPEN","url":"https://github.com/o/r/pull/7"}`)
	fx.reply(t, "pr-checks", `[{"bucket":"pass","name":"verify","state":"SUCCESS","workflow":"ci"}]`)
	fx.reply(t, "api-repo", `{"default_branch":"main","full_name":"o/r"}`)
	fx.reply(t, "api-graphql", `{"data":{"repository":{"mergeQueue":null}}}`)
	fx.reply(t, "api-rules", `[{"type":"required_status_checks","ruleset_id":1,"parameters":`+
		`{"strict_required_status_checks_policy":true,"required_status_checks":[{"context":"verify"}]}}]`)
	fx.reply(t, "api-ruleset", `{"name":"protect main","current_user_can_bypass":"never"}`)
	fx.reply(t, "repo-view", `{"viewerDefaultMergeMethod":"MERGE"}`)
	fx.reply(t, "pr-read-back", `{"state":"MERGED","headRefOid":"`+fx.head+`","baseRefName":"main","mergeCommit":{"oid":"`+
		mergeCLIMergeCommit+`"},"autoMergeRequest":null}`)
}

// newMergeCLIWorktreeFixture is newMergeCLIFixture with the ledger of the
// main worktree (work) and feat/a made the way ralph org start --plan makes
// it (plan docs/plans/active/2026-10-28-org-merge-cleanup.md, S2): work's main
// holds this repo's scripts/ralph-worktree.sh and scripts/ralph-common.sh and
// a .gitignore for .claude/worktrees/ and .harness/, pushed to origin; the
// script's ensure made the feature worktree org-a on feat/a, whose one commit
// is pushed to origin; and org a's binding names that worktree. PATH gets the
// tools the script needs after the fixture's bin, so git and gh are still the
// fixture's. It returns the fixture and the worktree.
func newMergeCLIWorktreeFixture(t *testing.T) (*mergeCLIFixture, string) {
	t.Helper()
	path := os.Getenv("PATH")
	for _, tool := range []string{"bash", "git", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Fatalf("%s is needed to run scripts/ralph-worktree.sh: %v", tool, err)
		}
	}
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file location via runtime.Caller")
	}
	source := filepath.Join(filepath.Dir(thisFile), "..", "..", "scripts")

	fx := newMergeCLIFixtureRepo(t, true)
	t.Setenv("PATH", fx.bin+string(os.PathListSeparator)+path)
	if err := os.MkdirAll(filepath.Join(fx.work, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ralph-worktree.sh", "ralph-common.sh"} {
		data, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(fx.work, "scripts", name), data, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeTestFile(t, filepath.Join(fx.work, ".gitignore"), ".claude/worktrees/\n.harness/\n")
	runLedgerTestGit(t, "-C", fx.work, "add", ".")
	gateCLICommit(t, fx.work, "the scripts")
	fx.push(t, "main")
	fx.main = fx.tip(t, "main")

	lines := outputLines(fx.worktreeScript(t, "ensure", "--id", "org-a", "--kind", "org", "--branch", "feat/a", "--path",
		".claude/worktrees/org-a", "--canonical-ref", "split:/plans/demo.md#a", "--cleanup-policy", "manual"))
	wt := filepath.Join(fx.work, ".claude", "worktrees", "org-a")
	if len(lines) == 0 || lines[len(lines)-1] != wt {
		t.Fatalf("ensure printed %q, want the worktree %s last", lines, wt)
	}
	gateCLIAddCommit(t, wt, "internal/a/a.go")
	fx.head = gateCLIGitOut(t, wt, "rev-parse", "HEAD")
	fx.push(t, "feat/a")

	appendSeedEvent(t, fx.stateDir, org.ManifestEvent{
		OrgID: "a", Event: org.EventScopeReserved, Worktree: wt,
		Details: "paths=internal/a/ split=demo feature=a digest=0123456789ab branch=feat/a approval=per-plan",
	})
	fx.gateAndReply(t)
	appendSeedEvent(t, fx.stateDir, org.ManifestEvent{OrgID: "a", Event: org.EventDisbanded})
	return fx, wt
}

// worktreeScript runs `bash scripts/ralph-worktree.sh args...` in work and
// returns its stdout, failing the test with its stderr when it fails.
func (fx *mergeCLIFixture) worktreeScript(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("bash", append([]string{"scripts/ralph-worktree.sh"}, args...)...)
	cmd.Dir = fx.work
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("ralph-worktree.sh %q: %v\n%s", args, err, stderr.String())
	}
	return string(out)
}

// worktreeRecord is the file of the ralph-worktree.sh record org-a in work,
// "<absent>" when there is none.
func (fx *mergeCLIFixture) worktreeRecord(t *testing.T) string {
	t.Helper()
	return fileBytesOrAbsent(t, strings.TrimSpace(fx.worktreeScript(t, "state-path", "org-a")))
}

// reply makes the fake gh print out for the call it names name.
func (fx *mergeCLIFixture) reply(t *testing.T, name, out string) {
	t.Helper()
	writeTestFile(t, filepath.Join(fx.replies, name+".out"), out+"\n")
}

// fail makes the fake gh print stderr and exit rc for the call it names
// name.
func (fx *mergeCLIFixture) fail(t *testing.T, name string, rc int, stderr string) {
	t.Helper()
	writeTestFile(t, filepath.Join(fx.replies, name+".rc"), strconv.Itoa(rc)+"\n")
	writeTestFile(t, filepath.Join(fx.replies, name+".err"), stderr+"\n")
}

// merge runs `ralph org merge --state-dir <the fixture's ledger> --branch
// feat/a` with extra after it.
func (fx *mergeCLIFixture) merge(t *testing.T, extra ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runOrgCmdStreams(t, append([]string{"merge", "--state-dir", fx.stateDir, "--branch", "feat/a"}, extra...)...)
}

// mergeCalls are the fake gh's gh pr merge calls, --disable-auto included.
func (fx *mergeCLIFixture) mergeCalls(t *testing.T) []string {
	t.Helper()
	var calls []string
	for _, c := range fx.ghCalls(t) {
		if strings.HasPrefix(c, "GH_PROMPT_DISABLED=1 pr merge ") {
			calls = append(calls, c)
		}
	}
	return calls
}

// prMergedEvents are the ledger's pr_merged events.
func (fx *mergeCLIFixture) prMergedEvents(t *testing.T) []org.ManifestEvent {
	t.Helper()
	var out []org.ManifestEvent
	for _, ev := range readManifestEvents(t, org.ManifestPathIn(fx.stateDir)) {
		if ev.Event == org.EventPRMerged {
			out = append(out, ev)
		}
	}
	return out
}

// mergeCLILines is the stdout of a recorded merge of the fixture's PR #7 at
// head, with autoMerge, and a NOTE: line when note is not empty.
func mergeCLILines(head string, autoMerge bool, note string) string {
	s := "MERGED: https://github.com/o/r/pull/7 (PR #7 of feat/a)\nMETHOD: merge\nHEAD: " + head + "\nMERGE COMMIT: " +
		mergeCLIMergeCommit + "\nAUTO MERGE: " + strconv.FormatBool(autoMerge) + "\nPROTECTION: " + mergeCLIProtection + "\n"
	if note != "" {
		s += "NOTE: " + note + "\n"
	}
	return s
}

// assertNoScriptCleanup checks the stdout of a recorded merge in the
// fixture: lines (mergeCLILines), then one CLEANUP: skipped line about
// scripts/ralph-worktree.sh, which the fixture's main worktree does not have,
// so the cleanup cannot read the worktree record and removes nothing. The
// line names no step of start, and no CLEANUP MANUAL: line follows, since the
// cleanup names no command by hand. internal/org's cleanup tests pin the
// reason's words.
func assertNoScriptCleanup(t *testing.T, stdout, lines string) {
	t.Helper()
	rest, ok := strings.CutPrefix(stdout, lines)
	if !ok || !strings.HasPrefix(rest, "CLEANUP: skipped (scripts/ralph-worktree.sh ") || !strings.HasSuffix(rest, ")\n") ||
		strings.Count(rest, "\n") != 1 || strings.Contains(rest, "start again") || strings.Contains(stdout, "CLEANUP MANUAL:") {
		t.Errorf("stdout:\n%s\nwant:\n%sCLEANUP: skipped (scripts/ralph-worktree.sh ...), on one line, naming no step of start, "+
			"with no CLEANUP MANUAL: line", stdout, lines)
	}
}

// mergeCLIDetails are the Details of the fixture's pr_merged at head with
// autoMerge.
func mergeCLIDetails(head string, autoMerge bool) string {
	return "branch=feat/a pr=7 head=" + head + " method=merge merge_commit=" + mergeCLIMergeCommit + " auto_merge=" +
		strconv.FormatBool(autoMerge)
}

// printMergeResult over every outcome Merge returns: a refusal and a
// failure (with gh's multi-line tail) are the error with nothing on stdout;
// a recorded merge prints the lines in order, NOTE: only with a Note,
// MERGE COMMIT: unknown without one, and exits 0; a merge with an error (the
// record failed, or an interrupt came after the merge started) prints the
// lines and returns the error; an error without a merge prints nothing; and
// a result with no outcome is an error, not exit 0. The cleanup after a
// recorded merge adds CLEANUP: done, with what it resumed from and an error
// of its record, or CLEANUP: skipped with its refusal, failure, or error,
// and CLEANUP MANUAL: with a command by hand, escaped, and never turns exit 0
// into an error.
func TestPrintMergeResult_EachOutcome(t *testing.T) {
	merged := org.MergeResult{
		Merged: true, Recorded: true, Branch: "feat/a", OrgID: "a", PRNumber: 7, PRURL: "https://github.com/o/r/pull/7",
		Method: "squash", Head: "aaaa", MergeCommit: "bbbb", AutoMerge: true, Protection: "a ruleset keeps it up to date",
	}
	lines := "MERGED: https://github.com/o/r/pull/7 (PR #7 of feat/a)\nMETHOD: squash\nHEAD: aaaa\nMERGE COMMIT: bbbb\n" +
		"AUTO MERGE: true\nPROTECTION: a ruleset keeps it up to date\n"
	const tail = "; gh said:\n  exit status 1: X\n  Y"
	withNote := merged
	withNote.Note = "gh pr merge 7 -R github.com/o/r failed, but gh pr view 7 -R github.com/o/r read PR #7 back as MERGED" + tail
	noCommit := merged
	noCommit.MergeCommit, noCommit.AutoMerge = "", false
	notRecorded := merged
	notRecorded.Recorded = false
	failed := org.MergeResult{
		Failed: "org: merge: gh pr merge 7 -R github.com/o/r failed and PR #7 is OPEN, not merged, so nothing was merged or recorded: " +
			"run ralph org merge-order again and follow the branch's Detail" + tail,
		Branch: "feat/a", OrgID: "a", PRNumber: 7, PRURL: "https://github.com/o/r/pull/7", Method: "merge", Head: "aaaa",
	}
	escaped := merged
	escaped.Protection = "a ruleset \x1b[31mnamed red"
	// The cleanup after the merge (plan
	// docs/plans/active/2026-10-28-org-merge-cleanup.md, S2): its outcome adds
	// the CLEANUP: lines and never the error. A command by hand comes only
	// with a refusal ralph cannot go on from, such as a missing record, and
	// with a failure.
	const manual = "cd /r && ./scripts/ralph-worktree.sh cleanup --id org-a --force-branch"
	const branchDelete = "git -C /r branch -D feat/a"
	withCleanup := func(c org.CleanupResult, cleanupErr string) org.MergeResult {
		res := merged
		res.Cleanup, res.CleanupErr = &c, cleanupErr
		return res
	}
	const cleanedNotRecorded = "org: cleanup: the feature worktree /w of the branch feat/a and its record org-a were removed, but " +
		"worktree_cleaned could not be recorded (denied)"
	const gitFailed = "org: cleanup: git worktree list --porcelain in /r: exit 128: fatal: not a git repository"
	cleanupInterrupted := withCleanup(org.CleanupResult{Refused: "ralph org merge was interrupted (SIGINT) once pr_merged was " +
		"recorded, so the feature worktree of the branch feat/a was not cleaned up: run ralph org cleanup --branch feat/a"}, "")

	for _, tc := range []struct {
		name       string
		res        org.MergeResult
		err        error
		wantStdout string
		wantErr    string
	}{
		{"refused", org.MergeResult{Refused: "org: merge: PR #7 is CLOSED, not OPEN, so nothing was merged: run ralph org merge-order"}, nil,
			"", "org: merge: PR #7 is CLOSED, not OPEN, so nothing was merged: run ralph org merge-order"},
		{"failed with gh's tail", failed, nil, "", failed.Failed},
		{"merged and recorded", merged, nil, lines, ""},
		{"merged with a note", withNote, nil, lines + "NOTE: " + withNote.Note + "\n", ""},
		{"no merge commit", noCommit, nil, strings.Replace(strings.Replace(lines, "bbbb", "unknown", 1), "AUTO MERGE: true", "AUTO MERGE: false", 1), ""},
		{"merged, the record failed", notRecorded, errors.New("org: merge: PR #7 was merged but pr_merged could not be recorded"),
			lines, "org: merge: PR #7 was merged but pr_merged could not be recorded"},
		{"interrupted after a recorded merge", merged, errors.New("org: merge: interrupted (SIGTERM) once gh pr merge had started, and then: merged"),
			lines, "org: merge: interrupted (SIGTERM) once gh pr merge had started, and then: merged"},
		{"interrupted without a merge", failed, errors.New("org: merge: interrupted (SIGINT) once gh pr merge had started, and then: not merged"),
			"", "org: merge: interrupted (SIGINT) once gh pr merge had started, and then: not merged"},
		{"an error before the merge", org.MergeResult{}, errors.New("org: merge: read manifest: denied"), "", "org: merge: read manifest: denied"},
		{"control characters are escaped", escaped, nil, strings.Replace(lines, "a ruleset keeps it up to date", `a ruleset \x1b[31mnamed red`, 1), ""},
		{"no outcome", org.MergeResult{Branch: "feat/a"}, nil, "",
			"org: merge: ralph org merge ended with no outcome for the branch feat/a (merged=false, recorded=false), so whether its PR " +
				"was merged is unknown: run ralph org merge-order to see the branch"},
		{"cleaned up", withCleanup(org.CleanupResult{Cleaned: true, Recorded: true}, ""), nil, lines + "CLEANUP: done\n", ""},
		{"cleaned up from what an earlier cleanup left", withCleanup(org.CleanupResult{Cleaned: true, Recorded: true,
			Resumed: org.CleanupResumedWorktreeGone}, ""), nil, lines + "CLEANUP: done (resumed: worktree_gone)\n", ""},
		{"cleaned up, worktree_cleaned not recorded", withCleanup(org.CleanupResult{Cleaned: true,
			Resumed: org.CleanupResumedRecordOnly}, cleanedNotRecorded), nil, lines + "CLEANUP: done (resumed: record_only; " +
			cleanedNotRecorded + ")\n", ""},
		{"cleanup refused, with the command by hand", withCleanup(org.CleanupResult{Refused: "the worktree record org-a of the " +
			"branch feat/a does not exist, so ralph removes nothing: the branch feat/a at \x1b left", Manual: branchDelete}, ""), nil,
			lines + `CLEANUP: skipped (the worktree record org-a of the branch feat/a does not exist, so ralph removes nothing: ` +
				`the branch feat/a at \x1b left)` + "\nCLEANUP MANUAL: " + branchDelete + "\n", ""},
		{"cleanup refused, no command by hand", withCleanup(org.CleanupResult{Refused: "the org runs"}, ""), nil,
			lines + "CLEANUP: skipped (the org runs)\n", ""},
		{"cleanup failed", withCleanup(org.CleanupResult{Failed: "scripts/ralph-worktree.sh cleanup --id org-a --force-branch failed",
			Manual: manual}, ""), nil, lines + "CLEANUP: skipped (scripts/ralph-worktree.sh cleanup --id org-a --force-branch failed)\n" +
			"CLEANUP MANUAL: " + manual + "\n", ""},
		{"cleanup error", withCleanup(org.CleanupResult{}, gitFailed), nil, lines + "CLEANUP: skipped (" + gitFailed + ")\n", ""},
		{"cleanup with no outcome", withCleanup(org.CleanupResult{}, ""), nil,
			lines + "CLEANUP: skipped (the cleanup ended with no outcome)\n", ""},
		{"interrupted once the merge was recorded", cleanupInterrupted, errors.New("org: merge: interrupted (SIGINT) once gh pr " +
			"merge had started, and then: merged"), lines + "CLEANUP: skipped (" + cleanupInterrupted.Cleanup.Refused + ")\n",
			"org: merge: interrupted (SIGINT) once gh pr merge had started, and then: merged"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out strings.Builder
			err := printMergeResult(&out, tc.res, tc.err)
			if out.String() != tc.wantStdout {
				t.Errorf("stdout:\n%q\nwant:\n%q", out.String(), tc.wantStdout)
			}
			switch {
			case tc.wantErr == "" && err != nil:
				t.Errorf("err = %v; want nil (exit 0)", err)
			case tc.wantErr != "" && (err == nil || err.Error() != tc.wantErr):
				t.Errorf("err = %v; want %q", err, tc.wantErr)
			case tc.err != nil && !errors.Is(err, tc.err):
				t.Errorf("err = %v; want Merge's error itself", err)
			}
		})
	}
}

// AC6 through the CLI: a ready branch whose PR passes every check is merged
// with one gh pr merge at the gate's head with the method's flag and
// origin's -R, and no --auto, --admin, or --delete-branch; stdout gets the
// lines, then the CLEANUP: line of a main worktree with no
// ralph-worktree.sh, stderr nothing, exit 0, and the ledger exactly one
// pr_merged with D4's Details. Every gh pr call names origin's repository
// with -R, every gh api call --hostname, and regate.lock is free afterwards.
// When gh pr merge fails but the read-back finds the PR MERGED, the merge is
// recorded the same way and NOTE: quotes gh's last lines.
func TestOrgMerge_MergesAReadyBranchAndRecordsPRMerged(t *testing.T) {
	for _, tc := range []struct {
		name     string
		ghFails  bool
		wantNote string
	}{
		{"gh pr merge exits 0", false, ""},
		{"gh pr merge fails, the PR merged anyway", true, "gh pr merge 7 -R github.com/o/r failed, but gh pr view 7 -R " +
			"github.com/o/r read PR #7 back as MERGED; gh said:\n  exit status 1: a network error\n  after the merge"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMergeCLIFixture(t, false)
			if tc.ghFails {
				fx.fail(t, "pr-merge", 1, "a network error\nafter the merge")
			}
			before := len(readManifestEvents(t, org.ManifestPathIn(fx.stateDir)))

			stdout, stderr, err := fx.merge(t)
			if err != nil || stderr != "" {
				t.Fatalf("merge: err %v, stderr %q (stdout %q)", err, stderr, stdout)
			}
			assertNoScriptCleanup(t, stdout, mergeCLILines(fx.head, false, tc.wantNote))
			events := readManifestEvents(t, org.ManifestPathIn(fx.stateDir))
			if len(events) != before+1 {
				t.Fatalf("the ledger got %d events; want exactly one: %+v", len(events)-before, events[before:])
			}
			if ev := events[before]; ev.Event != org.EventPRMerged || ev.OrgID != "a" || ev.SeatID != "" ||
				ev.Details != mergeCLIDetails(fx.head, false) {
				t.Errorf("recorded %+v; want pr_merged of org a with %q", ev, mergeCLIDetails(fx.head, false))
			}

			want := []string{"GH_PROMPT_DISABLED=1 pr merge 7 --merge --match-head-commit " + fx.head + " -R " + mergeCLIRepoArg}
			if got := fx.mergeCalls(t); !slices.Equal(got, want) {
				t.Errorf("gh pr merge calls = %q; want %q", got, want)
			}
			calls := fx.ghCalls(t)
			for _, c := range calls {
				args := strings.TrimPrefix(c, "GH_PROMPT_DISABLED=1 ")
				switch {
				case args == c:
					t.Errorf("gh call without GH_PROMPT_DISABLED=1: %q", c)
				case strings.Contains(args, "--auto") || strings.Contains(args, "--admin") || strings.Contains(args, "--delete-branch"):
					t.Errorf("gh call with --auto, --admin, or --delete-branch: %q", c)
				case strings.HasPrefix(args, "pr ") && !strings.Contains(args, " -R "+mergeCLIRepoArg):
					t.Errorf("gh pr call without -R %s: %q", mergeCLIRepoArg, c)
				case strings.HasPrefix(args, "api ") && !strings.HasPrefix(args, "api --hostname github.com "):
					t.Errorf("gh api call without --hostname github.com: %q", c)
				case strings.HasPrefix(args, "repo view ") && args != "repo view "+mergeCLIRepoArg+" --json viewerDefaultMergeMethod":
					t.Errorf("gh repo view call = %q; want origin's repository as its argument", c)
				}
			}
			if !slices.ContainsFunc(calls, func(c string) bool {
				return strings.HasSuffix(c, "pr view 7 -R "+mergeCLIRepoArg+" --json state,headRefOid,baseRefName,mergeCommit,autoMergeRequest")
			}) {
				t.Errorf("no read-back of the PR after gh pr merge: %q", calls)
			}
			holdMergeOrderLock(t, filepath.Join(fx.stateDir, "regate.lock"))
		})
	}
}

// AC7 and AC8 through the CLI: when the read-back finds the PR not MERGED,
// MERGED at another head than the gate's, or MERGED into another base than
// main, the command exits 1 with Merge's Failed as the error and nothing on
// stdout (no MERGED: line), and nothing is recorded: after a gh pr merge that
// failed (GitHub refused a base that moved), with gh's last lines in the
// error; after an auto-merge request GitHub held instead, which is removed
// with --disable-auto; after a gh pr merge that --match-head-commit failed
// while another head was merged; and after a merge into a retargeted base.
func TestOrgMerge_NotMergedExitsOneWithNothingOnStdout(t *testing.T) {
	for _, tc := range []struct {
		name      string
		setup     func(t *testing.T, fx *mergeCLIFixture)
		wantErr   []string
		wantCalls int // gh pr merge calls, --disable-auto included
	}{
		{"gh pr merge failed", func(t *testing.T, fx *mergeCLIFixture) {
			fx.fail(t, "pr-merge", 1, "GraphQL: Base branch was modified\nReview and try the merge again")
			fx.reply(t, "pr-read-back", `{"state":"OPEN","mergeCommit":null,"autoMergeRequest":null}`)
		}, []string{"org: merge: gh pr merge 7 -R github.com/o/r failed and PR #7 is OPEN, not merged, so nothing was merged or " +
			"recorded: run ralph org merge-order again and follow the branch's Detail; gh said:\n  exit status 1: GraphQL: Base branch " +
			"was modified\n  Review and try the merge again"}, 1},
		{"GitHub held an auto-merge request", func(t *testing.T, fx *mergeCLIFixture) {
			fx.reply(t, "pr-read-back", `{"state":"OPEN","mergeCommit":null,"autoMergeRequest":{"enabledAt":"2026-10-17T00:00:00Z"}}`)
		}, []string{"org: merge: gh pr merge 7 -R github.com/o/r exited 0 and did not merge PR #7, which is OPEN: GitHub held an " +
			"auto-merge request for it instead, which ralph org merge removed (gh pr merge 7 --disable-auto -R github.com/o/r)"}, 2},
		{"another head was merged", func(t *testing.T, fx *mergeCLIFixture) {
			fx.fail(t, "pr-merge", 1, "GraphQL: Head branch was modified. Review and try the merge again.")
			fx.reply(t, "pr-read-back", `{"state":"MERGED","headRefOid":"`+strings.Repeat("b", 40)+`","mergeCommit":{"oid":"`+
				mergeCLIMergeCommit+`"},"autoMergeRequest":null}`)
		}, []string{"org: merge: gh pr view 7 -R github.com/o/r read PR #7 back as MERGED after gh pr merge 7 -R github.com/o/r " +
			"failed, but at the head bbbbbbbbbbbb, not ", ", the head the gate passed, so a head the gate did not pass was merged, " +
			"and nothing was recorded: check that merge yourself (gh pr view 7 -R github.com/o/r --json state,headRefOid," +
			"baseRefName,mergeCommit), run ralph org merge-order again, and go on with the other branches; gh said:\n  exit status 1: " +
			"GraphQL: Head branch was modified. Review and try the merge again."}, 1},
		{"the PR merged into another base", func(t *testing.T, fx *mergeCLIFixture) {
			fx.reply(t, "pr-read-back", `{"state":"MERGED","headRefOid":"`+fx.head+`","baseRefName":"release","mergeCommit":{"oid":"`+
				mergeCLIMergeCommit+`"},"autoMergeRequest":null}`)
		}, []string{"org: merge: gh pr view 7 -R github.com/o/r read PR #7 back as MERGED after gh pr merge 7 -R github.com/o/r " +
			"exited 0, but into the base release, not main, the default branch the checks read, so main got nothing from it as far " +
			"as ralph knows, and nothing was recorded: check that merge yourself (gh pr view 7 -R github.com/o/r --json state," +
			"headRefOid,baseRefName,mergeCommit), run ralph org merge-order again (it lists the branch as merged, though main got " +
			"nothing from this merge), and go on with the other branches"}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMergeCLIFixture(t, false)
			tc.setup(t, fx)
			before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))

			stdout, _, err := fx.merge(t)
			if err == nil {
				t.Fatalf("merge exited 0 (stdout %q); want exit 1", stdout)
			}
			for _, want := range tc.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("err = %v; want it to say %q", err, want)
				}
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			if got := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); got != before {
				t.Errorf("the ledger changed:\nbefore %s\nafter  %s", before, got)
			}
			if got := fx.mergeCalls(t); len(got) != tc.wantCalls {
				t.Errorf("gh pr merge calls = %q; want %d", got, tc.wantCalls)
			}
		})
	}
}

// AC8's last case through the CLI: when the PR merged but pr_merged cannot
// be appended (the manifest is read-only), stdout still gets the lines, with
// no CLEANUP: line since no cleanup runs, and the command exits 1 saying the
// merge stands and was not recorded.
func TestOrgMerge_RecordThatFailsPrintsTheLinesAndExitsOne(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores read-only permission bits")
	}
	fx := newMergeCLIFixture(t, false)
	manifest := org.ManifestPathIn(fx.stateDir)
	if err := os.Chmod(manifest, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(manifest, 0o644) })

	stdout, _, err := fx.merge(t)
	if err == nil || !strings.Contains(err.Error(), "org: merge: PR #7 (https://github.com/o/r/pull/7) was merged (merge commit "+
		mergeCLIMergeCommit+") but pr_merged could not be recorded") {
		t.Fatalf("merge err = %v; want that the merge stands and pr_merged was not recorded", err)
	}
	if want := mergeCLILines(fx.head, false, ""); stdout != want {
		t.Errorf("stdout:\n%s\nwant the lines of the merge:\n%s", stdout, want)
	}
	if got := fx.prMergedEvents(t); len(got) != 0 {
		t.Errorf("pr_merged recorded: %+v", got)
	}
}

// AC1 and AC3 through the CLI and the real scripts/ralph-worktree.sh (plan
// docs/plans/active/2026-10-28-org-merge-cleanup.md, S2): once the merge of
// feat/a is recorded, merge removes the feature worktree org-a, its
// registration, the local branch, and the record with the script's cleanup
// --force-branch, though main does not contain feat/a, which git branch -d
// refuses; the ledger gets worktree_cleaned with via=merge right after
// pr_merged, stdout ends with CLEANUP: done, and the command exits 0. With an
// untracked file in the worktree, merge prints CLEANUP: skipped with the
// reason and no CLEANUP MANUAL: line (ralph org cleanup goes on once the file
// is gone), removes nothing, records no worktree_cleaned, and still exits 0.
func TestOrgMerge_CleansUpTheFeatureWorktree(t *testing.T) {
	t.Run("cleaned up", func(t *testing.T) {
		fx, wt := newMergeCLIWorktreeFixture(t)
		if err := exec.Command("git", "-C", fx.work, "merge-base", "--is-ancestor", "feat/a", "main").Run(); err == nil {
			t.Fatal("main contains feat/a; want a branch git branch -d refuses")
		}
		if fx.worktreeRecord(t) == "<absent>" {
			t.Fatal("no record org-a before the merge")
		}

		stdout, stderr, err := fx.merge(t)
		if err != nil || stderr != "" {
			t.Fatalf("merge: err %v, stderr %q (stdout %q)", err, stderr, stdout)
		}
		if want := mergeCLILines(fx.head, false, "") + "CLEANUP: done\n"; stdout != want {
			t.Errorf("stdout:\n%s\nwant:\n%s", stdout, want)
		}
		if _, err := os.Stat(wt); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("the worktree %s is still there (stat: %v)", wt, err)
		}
		if list := gateCLIGitOut(t, fx.work, "worktree", "list", "--porcelain"); strings.Contains(list, wt) {
			t.Errorf("git still lists the worktree:\n%s", list)
		}
		if got := gateCLIGitOut(t, fx.work, "branch", "--list", "feat/a"); got != "" {
			t.Errorf("the branch feat/a is still there: %q", got)
		}
		if got := fx.worktreeRecord(t); got != "<absent>" {
			t.Errorf("the record org-a is still there:\n%s", got)
		}
		events := readManifestEvents(t, org.ManifestPathIn(fx.stateDir))
		want := "branch=feat/a head=" + fx.head + " via=merge worktree=" + strconv.Quote(wt)
		if n := len(events); n < 2 || events[n-2].Event != org.EventPRMerged || events[n-1].Event != org.EventWorktreeCleaned ||
			events[n-1].OrgID != "a" || events[n-1].SeatID != "" || events[n-1].Details != want {
			t.Errorf("the ledger ends with %+v; want pr_merged, then org a's worktree_cleaned with %q", events[max(0, n-2):], want)
		}
		holdMergeOrderLock(t, filepath.Join(fx.stateDir, "regate.lock"))
	})
	t.Run("an untracked file", func(t *testing.T) {
		fx, wt := newMergeCLIWorktreeFixture(t)
		writeTestFile(t, filepath.Join(wt, "stray.txt"), "x\n")
		record := fx.worktreeRecord(t)

		stdout, stderr, err := fx.merge(t)
		if err != nil || stderr != "" {
			t.Fatalf("merge: err %v, stderr %q (stdout %q); want exit 0", err, stderr, stdout)
		}
		want := mergeCLILines(fx.head, false, "") + "CLEANUP: skipped (the feature worktree " + wt + " has 1 uncommitted " +
			"change(s) or untracked file(s) (?? stray.txt), which removing it would lose: move what you need out of the worktree, " +
			"remove the rest, and run ralph org cleanup --branch feat/a again)\n"
		if stdout != want {
			t.Errorf("stdout:\n%s\nwant:\n%s", stdout, want)
		}
		if info, err := os.Stat(filepath.Join(wt, "stray.txt")); err != nil || !info.Mode().IsRegular() {
			t.Errorf("the untracked file is gone (stat: %v)", err)
		}
		if got := gateCLIGitOut(t, fx.work, "rev-parse", "refs/heads/feat/a"); got != fx.head {
			t.Errorf("feat/a is at %s, want %s", got, fx.head)
		}
		if got := fx.worktreeRecord(t); got != record {
			t.Errorf("the record changed:\n%s\n->\n%s", record, got)
		}
		events := readManifestEvents(t, org.ManifestPathIn(fx.stateDir))
		if last := events[len(events)-1]; last.Event != org.EventPRMerged || slices.ContainsFunc(events, func(ev org.ManifestEvent) bool {
			return ev.Event == org.EventWorktreeCleaned
		}) {
			t.Errorf("the ledger ends with %+v; want pr_merged and no worktree_cleaned", last)
		}
	})
}

// D3 through the CLI: when the main worktree cannot be found (the ledger is
// chosen with --state-dir and the current directory is in no repository, its
// origin a path to the bare origin), merge merges and records as before,
// runs no cleanup, prints CLEANUP: skipped with the reason, and exits 0.
func TestOrgMerge_NoMainWorktreeSkipsTheCleanup(t *testing.T) {
	fx := newMergeCLIFixture(t, false)
	outside := filepath.Join(filepath.Dir(fx.work), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(fx.bare, filepath.Join(outside, "origin")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(outside))
	t.Chdir(outside)

	stdout, stderr, err := fx.merge(t)
	if err != nil || stderr != "" {
		t.Fatalf("merge: err %v, stderr %q (stdout %q); want exit 0", err, stderr, stdout)
	}
	want := mergeCLILines(fx.head, false, "") + "CLEANUP: skipped (the main worktree, where scripts/ralph-worktree.sh removes " +
		"the feature worktree, was not found (the current directory is in no git repository, or in one with no main worktree, " +
		"such as a linked worktree of a bare repository), so the feature worktree of the branch feat/a was not cleaned up: run " +
		"ralph org cleanup --branch feat/a from the main worktree)\n"
	if stdout != want {
		t.Errorf("stdout:\n%s\nwant:\n%s", stdout, want)
	}
	if got := fx.prMergedEvents(t); len(got) != 1 || got[0].Details != mergeCLIDetails(fx.head, false) {
		t.Errorf("pr_merged = %+v; want one with %q", got, mergeCLIDetails(fx.head, false))
	}
	if got := fileBytesOrAbsent(t, filepath.Join(fx.stateDir, "regate.lock")); got != "" {
		t.Errorf("regate.lock holds %q; want no marker, since no cleanup ran", got)
	}
}

// AC3 and AC10 through the CLI: a refusal of the runtime is the error, with
// nothing on stdout, no gh pr merge, and the ledger as it was: a branch the
// ledger does not name, gh missing from PATH, a check still pending, and
// regate.lock held by another regate, merge, or cleanup.
func TestOrgMerge_RefusalsGoToStderrWithNothingMerged(t *testing.T) {
	for _, tc := range []struct {
		name    string
		setup   func(t *testing.T, fx *mergeCLIFixture)
		branch  string
		wantErr string
	}{
		{"a branch the ledger does not name", func(*testing.T, *mergeCLIFixture) {}, "feat/none",
			"org: merge: no feature binding or gate record in the ledger names the branch feat/none, so nothing was merged"},
		{"gh missing from PATH", func(t *testing.T, fx *mergeCLIFixture) {
			if err := os.Remove(filepath.Join(fx.bin, "gh")); err != nil {
				t.Fatal(err)
			}
		}, "feat/a", "org: merge: gh is not on PATH"},
		{"a check pending", func(t *testing.T, fx *mergeCLIFixture) {
			fx.reply(t, "pr-checks", `[{"bucket":"pass","name":"verify","state":"SUCCESS","workflow":"ci"},`+
				`{"bucket":"pending","name":"e2e","state":"IN_PROGRESS","workflow":"ci"}]`)
			fx.fail(t, "pr-checks", 8, "")
		}, "feat/a", "org: merge: 1 of the 2 checks of PR #7 are not pass or skipping: ci / e2e (pending, IN_PROGRESS)"},
		{"regate.lock held", func(t *testing.T, fx *mergeCLIFixture) {
			holdMergeOrderLock(t, filepath.Join(fx.stateDir, "regate.lock"))
		}, "feat/a", "org: merge: another ralph org regate, merge, or cleanup runs on this ledger (" +
			"%STATE%/regate.lock is held): one runs at a time, so wait for it to end and run ralph org merge --branch feat/a again"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMergeCLIFixture(t, false)
			tc.setup(t, fx)
			before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))

			stdout, _, err := runOrgCmdStreams(t, "merge", "--state-dir", fx.stateDir, "--branch", tc.branch)
			want := strings.ReplaceAll(tc.wantErr, "%STATE%", fx.stateDir)
			if err == nil || !strings.HasPrefix(err.Error(), want) {
				t.Errorf("err = %v; want it to start with %q", err, want)
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			if got := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); got != before {
				t.Errorf("the ledger changed:\nbefore %s\nafter  %s", before, got)
			}
			if got := fx.mergeCalls(t); len(got) != 0 {
				t.Errorf("gh pr merge ran: %q", got)
			}
		})
	}
}

// AC3's flag refusals, and AC13's range of --wait-checks: --org-id, no
// --branch, a blank --branch, and a --wait-checks below 0 or above 6h are
// refused before anything is read: exit 1 with the reason, nothing on
// stdout, no git or gh run (not even to resolve the state dir), and no state
// dir made.
func TestOrgMerge_FlagRefusalsReadNothing(t *testing.T) {
	isolateFeatureGitEnv(t)
	calls := writeMergeOrderCallLoggers(t)
	tmp := mergeOrderTempDir(t)
	t.Chdir(tmp)
	stateDir := filepath.Join(tmp, "state")
	const (
		orgIDRefusal  = "org: merge: --org-id does not apply: merge reads the org from the binding or the gate record of the branch --branch names"
		branchRefusal = "org: merge: --branch is required: name the feature branch whose PR to merge (ralph org merge-order lists the ledger's branches)"
	)
	waitRefusal := func(d string) string {
		return "org: merge: --wait-checks " + d + " is out of range, so nothing was merged: give a duration from 0 (no wait, the " +
			"default) to 6h0m0s, such as 60m"
	}
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"merge", "--org-id", "a", "--branch", "feat/a", "--state-dir", stateDir}, orgIDRefusal},
		{[]string{"merge", "--org-id", "a", "--branch", "feat/a"}, orgIDRefusal},
		{[]string{"merge", "--state-dir", stateDir}, branchRefusal},
		{[]string{"merge", "--branch", "  ", "--state-dir", stateDir}, branchRefusal},
		{[]string{"merge"}, branchRefusal},
		{[]string{"merge", "--branch", "feat/a", "--wait-checks", "-1s", "--state-dir", stateDir}, waitRefusal("-1s")},
		{[]string{"merge", "--branch", "feat/a", "--wait-checks", "6h1m"}, waitRefusal("6h1m0s")},
	} {
		stdout, _, err := runOrgCmdStreams(t, tc.args...)
		if err == nil || err.Error() != tc.want || stdout != "" {
			t.Errorf("%q: err %v, stdout %q; want the error %q and nothing on stdout", tc.args, err, stdout, tc.want)
		}
	}
	if got := readLogLines(t, calls); len(got) != 0 {
		t.Errorf("git or gh ran before the refusal: %q", got)
	}
	for _, dir := range []string{stateDir, filepath.Join(tmp, ".harness")} {
		if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("a refused merge made %s (stat: %v)", dir, err)
		}
	}
}

// AC13 through the CLI: --wait-checks 6h, the longest, is taken, and with
// checks that pass merge merges with nothing on stderr. With checks pending,
// --wait-checks 1ms waits once: stderr gets the line of its start and the
// line of its end, and once the time is up the checks run again and refuse
// as without the flag, adding how long merge waited, with no gh pr merge and
// the ledger as it was. With the branch's org still running (D5), it waits
// the same way for the org's disband, running no gh meanwhile, and the
// refusal also names ralph org status for the org.
func TestOrgMerge_WaitChecks(t *testing.T) {
	t.Run("6h is taken", func(t *testing.T) {
		fx := newMergeCLIFixture(t, false)
		stdout, stderr, err := fx.merge(t, "--wait-checks", "6h")
		if err != nil || stderr != "" {
			t.Fatalf("merge --wait-checks 6h: err %v, stderr %q, stdout %q; want the merge", err, stderr, stdout)
		}
		assertNoScriptCleanup(t, stdout, mergeCLILines(fx.head, false, ""))
	})
	t.Run("the time is up with checks pending", func(t *testing.T) {
		fx := newMergeCLIFixture(t, false)
		fx.reply(t, "pr-checks", `[{"bucket":"pass","name":"verify","state":"SUCCESS","workflow":"ci"},`+
			`{"bucket":"pending","name":"e2e","state":"IN_PROGRESS","workflow":"ci"}]`)
		fx.fail(t, "pr-checks", 8, "")
		before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))

		stdout, stderr, err := fx.merge(t, "--wait-checks", "1ms")
		const refusal = "org: merge: 1 of the 2 checks of PR #7 are not pass or skipping: ci / e2e (pending, IN_PROGRESS), so " +
			"nothing was merged: wait for the pending ones and fix the others, and once all passed, run ralph org merge --branch " +
			"feat/a again; ralph org merge waited "
		if err == nil || !strings.HasPrefix(err.Error(), refusal) || !strings.HasSuffix(err.Error(), " for the checks (--wait-checks 1ms)") ||
			stdout != "" {
			t.Fatalf("merge --wait-checks 1ms: err %v, stdout %q; want the refusal with the time waited", err, stdout)
		}
		lines := outputLines(stderr)
		start := "org: merge: PR #7 has checks pending, so ralph org merge waits for them for up to 1ms (--wait-checks 1ms), with " +
			"regate.lock released, reading gh pr checks 7 -R " + mergeCLIRepoArg + " every 30s"
		if len(lines) != 2 || lines[0] != start || !strings.HasPrefix(lines[1], "org: merge: stopped waiting for the checks of PR #7 "+
			"after ") || !strings.HasSuffix(lines[1], ": --wait-checks 1ms is used up; running every check again") {
			t.Errorf("stderr:\n%s\nwant the line of the wait's start and the line of its end", stderr)
		}
		if got := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); got != before {
			t.Errorf("the ledger changed:\nbefore %s\nafter  %s", before, got)
		}
		if got := fx.mergeCalls(t); len(got) != 0 {
			t.Errorf("gh pr merge ran: %q", got)
		}
	})
	t.Run("the time is up with the org still running", func(t *testing.T) {
		fx := newMergeCLIFixtureOrgRunning(t, false)
		before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))

		stdout, stderr, err := fx.merge(t, "--wait-checks", "1ms")
		const refusal = `org: merge: the branch feat/a is bound to org_id "a", which is running, so its leader may still push to ` +
			"the branch and open or change the PR, and nothing was merged: handle the org's RESULT in the inbox (ralph org inbox), " +
			"and once the org is disbanded, run ralph org merge --branch feat/a again; ralph org merge waited "
		const hint = ` for org_id "a" to be disbanded (--wait-checks 1ms); the org still runs: see what it does with ralph org ` +
			"status --org-id a, and a human decides whether to disband it (ralph org disband --org-id a)"
		if err == nil || !strings.HasPrefix(err.Error(), refusal) || !strings.HasSuffix(err.Error(), hint) || stdout != "" {
			t.Fatalf("merge --wait-checks 1ms: err %v, stdout %q; want the refusal with the time waited and ralph org status", err,
				stdout)
		}
		lines := outputLines(stderr)
		start := `org: merge: org_id "a" of the branch is still running (its leader disbands it after its RESULT), so ralph org ` +
			"merge waits for it to be disbanded for up to 1ms (--wait-checks 1ms), with regate.lock released, reading the ledger " +
			"every 30s"
		if len(lines) != 2 || lines[0] != start || !strings.HasPrefix(lines[1], `org: merge: stopped waiting for org_id "a" to be `+
			"disbanded after ") || !strings.HasSuffix(lines[1], ": --wait-checks 1ms is used up; running every check again") {
			t.Errorf("stderr:\n%s\nwant the line of the wait's start and the line of its end", stderr)
		}
		if got := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); got != before {
			t.Errorf("the ledger changed:\nbefore %s\nafter  %s", before, got)
		}
		// Only the gh pr list of each round of the checks: none while merge
		// waited, and no gh pr view, checks, or merge.
		calls := fx.ghCalls(t)
		if len(calls) != 2 || !strings.HasPrefix(calls[0], "GH_PROMPT_DISABLED=1 pr list ") ||
			!strings.HasPrefix(calls[1], "GH_PROMPT_DISABLED=1 pr list ") {
			t.Errorf("gh calls = %q; want the gh pr list of the two rounds of the checks only", calls)
		}
	})
}

// AC2 for merge: with no --config and the main worktree's ledger, run from a
// linked worktree, [org.director].auto_merge comes from the main worktree's
// ralph.toml, not the linked worktree's, into AUTO MERGE: and pr_merged's
// auto_merge=, both ways.
func TestOrgMerge_AutoMergeComesFromTheMainWorktreeRalphToml(t *testing.T) {
	const (
		on  = "[org.director]\nauto_merge = true\n"
		off = "[org.director]\nauto_merge = false\n"
	)
	for _, tc := range []struct {
		name             string
		mainToml, wtToml string
		want             bool
	}{
		{"main says true, the worktree says false", on, off, true},
		{"main says false, the worktree says true", off, on, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMergeCLIFixture(t, true)
			wt := filepath.Join(filepath.Dir(fx.work), "wt")
			runLedgerTestGit(t, "-C", fx.work, "worktree", "add", "--quiet", wt, "-b", "wt")
			writeTestFile(t, filepath.Join(fx.work, "ralph.toml"), tc.mainToml)
			writeTestFile(t, filepath.Join(wt, "ralph.toml"), tc.wtToml)
			t.Chdir(wt)

			stdout, stderr, err := runOrgCmdStreams(t, "merge", "--branch", "feat/a")
			if err != nil || stderr != "" {
				t.Fatalf("merge from the linked worktree: err %v, stderr %q (stdout %q)", err, stderr, stdout)
			}
			assertNoScriptCleanup(t, stdout, mergeCLILines(fx.head, tc.want, ""))
			got := fx.prMergedEvents(t)
			if len(got) != 1 || got[0].Details != mergeCLIDetails(fx.head, tc.want) {
				t.Errorf("pr_merged in the main worktree's ledger = %+v; want one with %q", got, mergeCLIDetails(fx.head, tc.want))
			}
		})
	}
}

// merge changes the ledger, so guardLegacyOrgStateDir refuses it while the
// linked worktree's older ledger still has an active seat, before origin, gh,
// or the shared ledger is touched.
func TestOrgMerge_RefusedWhileALegacyLedgerHasActiveSeats(t *testing.T) {
	setupOrgStubPATH(t)
	r := newLegacyLedgerRepo(t)
	spawnLedgerSeat(t, r.legacyDir, "seat-1")
	before := readFileBytes(t, org.ManifestPathIn(r.legacyDir))

	stdout, _, err := runOrgCmdStreams(t, "merge", "--branch", "feat/a")
	if err == nil {
		t.Fatalf("merge ran while the legacy ledger has an active seat (stdout %q)", stdout)
	}
	for _, want := range []string{"org: refusing to change the org ledger", r.legacyDir, "1 active seat(s)"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err = %v; want it to say %q", err, want)
		}
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if after := readFileBytes(t, org.ManifestPathIn(r.legacyDir)); string(after) != string(before) {
		t.Errorf("the legacy ledger changed:\nbefore %s\nafter  %s", before, after)
	}
	if _, err := os.Stat(r.sharedDir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a refused merge made the shared ledger %s (stat: %v)", r.sharedDir, err)
	}
}

// The command's context reaches Merge through signal.NotifyContext: with it
// done before the merge, the command exits 1 saying it was interrupted,
// with nothing on stdout, no gh call, and nothing recorded.
func TestOrgMerge_InterruptedBeforeTheMergeMergesNothing(t *testing.T) {
	fx := newMergeCLIFixture(t, false)
	before := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stdout, _, err := runOrgCmdStreamsContext(ctx, "merge", "--state-dir", fx.stateDir, "--branch", "feat/a")
	if err == nil || !strings.HasPrefix(err.Error(), "org: merge: interrupted (") || !strings.HasSuffix(err.Error(), "nothing was merged") {
		t.Errorf("err = %v; want the interruption before the merge", err)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if got := fx.ghCalls(t); len(got) != 0 {
		t.Errorf("gh ran: %q", got)
	}
	if got := fileBytesOrAbsent(t, org.ManifestPathIn(fx.stateDir)); got != before {
		t.Errorf("the ledger changed:\nbefore %s\nafter  %s", before, got)
	}
}

// AC12 for the CLI: merge's help says what it checks, how it merges, what it
// prints, and how auto_merge is read; org --help names the verb, lists it
// among the verbs that refuse --org-id, and says that merge reads auto_merge
// without --config.
func TestOrgMerge_Help(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		wants []string
	}{
		{[]string{"merge", "--help"}, []string{
			"ralph org merge merges, with gh, the PR of one feature branch that ralph org merge-order lists as ready",
			"so --org-id is refused",
			"gh pr with -R <host>/<owner>/<repo>, gh repo view with that as its argument, and gh api with --hostname <host>",
			"that no other ralph org regate, merge, or cleanup runs on the ledger (one runs at a time; it waits up to 1 s for regate.lock)",
			"that the branch's org is not running", "and no auto-merge request",
			"that every check gh pr checks lists is pass or skipping, and at least one is pass",
			"shows no merge queue", "whose current_user_can_bypass is never",
			"strict with required checks and has enforce_admins enabled, whether or not you are an admin of the repository, since " +
				"without it an admin or a custom role that may bypass branch protections can merge a head that is not up to date",
			"(viewerDefaultMergeMethod: MERGE, SQUASH, or REBASE)",
			"a second git ls-remote finds origin's default branch and the branch where the gate left them",
			"gh pr merge <n> --merge|--squash|--rebase --match-head-commit <the gate's head> -R <repo> once, for up to 60 s, " +
				"never with --auto, --admin, or --delete-branch",
			"Whatever that returned, merge reads the PR back once with gh pr view",
			"prints MERGED: <PR url> (PR #<n> of <branch>), METHOD:, HEAD:, MERGE COMMIT: (unknown when gh gave none), AUTO MERGE:, PROTECTION:",
			"is removed with gh pr merge <n> --disable-auto -R <repo>",
			// The merged head (S8b, cross-review #3) and base (S10, cross-review cycle 2 #1).
			"When the PR is MERGED at the gate's head into origin's default branch, merge records pr_merged",
			"When the PR is MERGED at another head, or with no head read back (someone may have pushed a new head and merged it " +
				"after the checks), merge records nothing, prints no MERGED: line, names the head merged on stderr, and exits 1.",
			"When the PR is MERGED at the gate's head into another base, or with no base read back (someone may have retargeted " +
				"the PR after the checks; --match-head-commit binds the head, not the base), merge records nothing, prints no " +
				"MERGED: line, names the base on stderr, and exits 1.",
			"is of another head than the gate's, or went into another base than the default branch",
			"When the PR merged but pr_merged cannot be recorded, merge prints the lines above and exits 1.",
			"An interrupt (SIGINT, SIGTERM, or SIGHUP)",
			"The read-back and the removal run in a process group of their own, so a Ctrl-C or hangup the terminal sends does " +
				"not stop them",
			"it needs no running watch and no approved plan",
			"ralph org spawn and start of the branch's org are refused only while merge cleans up its feature worktree, since the " +
				"merge itself changes no worktree.",
			// The cleanup after a recorded merge (plan
			// docs/plans/active/2026-10-28-org-merge-cleanup.md, S2).
			"Once pr_merged is recorded, merge cleans up the branch's feature worktree, the one ralph org start --plan made, with " +
				"regate.lock still held: it runs ./scripts/ralph-worktree.sh cleanup --id org-<org_id> --force-branch in the main " +
				"worktree (found as start --plan finds it), which removes the worktree, the local branch, and the script's record, " +
				"and records worktree_cleaned in the ledger.",
			"that the worktree has no uncommitted change or untracked file and no rebase or merge in progress (files .gitignore " +
				"names do not count and go with the worktree); that the local branch is at the head the gate passed; and that the " +
				"worktree has that branch checked out at that head. When a check fails, it removes nothing.",
			"After the lines above, merge prints CLEANUP: done, with (resumed: <what was left>) when it went on from such a " +
				"cleanup, or CLEANUP: skipped (<why>) followed, when there is one, by CLEANUP MANUAL: <the command that cleans up " +
				"by hand once the reason is checked>. Only a cleanup ralph cannot go on with by itself has such a command (the " +
				"record is gone or is not the binding's, the worktree is left with no local branch, or the removal itself " +
				"failed); after any other skip, fix what the reason says and run ralph org cleanup --branch <branch> again. The " +
				"cleanup never changes the exit code.",
			"an interrupt once pr_merged is recorded skips the cleanup or stops it, and the CLEANUP: line says how it ended.",
			"Exit codes: 0 for a merge that was recorded, whatever the cleanup did;",
			"[org.director].auto_merge is only reported, in AUTO MERGE: and in pr_merged: merge merges whatever it is.",
			"Without --config, it is read from the main worktree's ralph.toml when the ledger is the main worktree's",
			"--branch string",
			// --wait-checks (S7, AC13), and its wait for the org (S8c, D5).
			"--wait-checks duration how long to wait, with regate.lock released, for checks that are pending or not reported yet " +
				"and for the branch's org to be disbanded before refusing, such as 60m (0 waits not at all; at most 6h)",
			"With --wait-checks <duration> (Go's form, such as 60m; 0, the default, waits not at all; at most 6h, and a negative " +
				"value or one over 6h is refused before anything is read), merge waits instead of refusing when the only refusal is " +
				"that the branch's org is still running (its leader disbands it after its RESULT), that a check is pending, or that " +
				"gh pr checks reports no checks yet.",
			"It releases regate.lock, so a ralph org regate or another merge can run meanwhile.",
			"For the org, it reads only the ledger every 30 s, with no git and no gh, until the org is disbanded, runs another " +
				"feature (told apart only on a branch with a feature binding; on a branch with only a gate record, the wait goes " +
				"on until the org is disbanded or the time is up), or the ledger cannot be read; for the checks, it reads only " +
				"gh pr checks <n> -R <repo> every 30 s, until no check is pending any more or gh pr checks fails. Either wait also " +
				"ends when the time is up. Then merge takes regate.lock again and runs every check above from the start",
			"An org that runs another feature now (on a branch with a feature binding), a check that failed or was cancelled, " +
				"checks that are all skipping, and every other refusal are not waited for.",
			"When the time is up and the org still runs or the checks are still pending or none, merge refuses as without the flag " +
				"and adds how long it waited; for the org, it also names ralph org status --org-id <org>, since whether to disband " +
				"the org is a human's call.",
			"stderr gets one line when a wait starts and one when it ends.",
			"An interrupt while merge waits merges nothing and exits 1.",
			"In a repository with no CI, the checks stay at none, and merge waits until the time is up.",
			"While it runs, except while it waits, ralph org regate and another ralph org merge are refused",
		}},
		{[]string{"--help"}, []string{
			"merge Merge, with gh, the PR of a feature branch ralph org merge-order lists as ready",
			"`ralph org merge` merges, with gh, the PR of a branch merge-order lists as ready",
			"stop --all, disband --all, merge-order, merge, cleanup, regate,",
			"merge reads auto_merge",
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
	}
}
