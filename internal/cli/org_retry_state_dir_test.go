package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// The commands ralph org cleanup and ralph org merge name to run again keep
// the --state-dir they were given (plan
// docs/plans/active/2026-11-01-org-ledger-gaps.md, S7: AC1, tech-debt 395),
// and those about a whole cleanup run keep its --branch, or its lack of one:
// cleanupRetryCommand builds the CLI's, and orgInboxHintStateDir's value
// reaches the org layer's as CleanupParams.HintStateDir and
// MergeParams.HintStateDir. A ledger the personal file names gets no
// --state-dir, since the bare command reads the same file. The replay tests
// take a command out of the output, split it into words with sh as a person
// pasting it would, and run those words, to show that it acts on the ledger
// named first and not on the one ralph finds by default.

// orgRetryHint is the --state-dir of the unit tests, with a space, so that it
// is quoted as one shell word.
const orgRetryHint = "/my ledgers/org"

// cleanupRetryCommand over a branch and a hint, each empty, plain, or with
// characters a shell reads: --branch only with a branch, --state-dir only with
// a hint, each value escaped and one shell word.
func TestOrgCleanup_RetryCommandKeepsTheBranchAndTheStateDir(t *testing.T) {
	for _, tc := range []struct{ branch, hint, want string }{
		{"", "", "ralph org cleanup"},
		{"feat/a", "", "ralph org cleanup --branch feat/a"},
		{"", "/state/org", "ralph org cleanup --state-dir /state/org"},
		{"feat/a", "/state/org", "ralph org cleanup --branch feat/a --state-dir /state/org"},
		{"feat/a;b", orgRetryHint, `ralph org cleanup --branch 'feat/a;b' --state-dir '/my ledgers/org'`},
		{"feat/a", "/o'brien/org", `ralph org cleanup --branch feat/a --state-dir '/o'\''brien/org'`},
		{"feat/\x1b", "", `ralph org cleanup --branch 'feat/\x1b'`},
	} {
		if got := cleanupRetryCommand(tc.branch, tc.hint); got != tc.want {
			t.Errorf("cleanupRetryCommand(%q, %q) = %q, want %q", tc.branch, tc.hint, got, tc.want)
		}
	}
}

// The refusals and interruptions of ralph org cleanup that the CLI words name
// the command with the hint, and with --branch only when the run was given
// one: origin that names no repository, gh not on PATH, a PR not known to be
// merged (with and without --branch), and an interruption before a cleanup
// and after the steps.
func TestOrgCleanup_RefusalsAndInterruptionsNameTheStateDir(t *testing.T) {
	const quoted = "'" + orgRetryHint + "'"
	bind := &org.MergeOrderBinding{OrgID: "a", Feature: "f", Split: "s", TS: "t"}
	unread := org.MergeOrderResult{GHAvailable: true, Entries: []org.MergeOrderEntry{{Branch: "feat/a", Class: org.MergeClassNoGate,
		PRLookup: org.MergeOrderPRLookupFailed, Binding: bind}}}
	noGH := org.MergeOrderResult{Entries: unread.Entries}
	stopped, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("test stopped"))
	merged := cleanupStep{branch: "feat/a", head: "0123456789abcdef0123456789abcdef01234567", pr: 7}
	for _, run := range []struct{ name, branch, command string }{
		{"every merged branch", "", "ralph org cleanup --state-dir " + quoted},
		{"--branch", "feat/a", "ralph org cleanup --branch feat/a --state-dir " + quoted},
	} {
		branch, command := run.branch, run.command
		t.Run(run.name, func(t *testing.T) {
			got := map[string]string{}
			if err := cleanupOriginRefusal(stopped, org.MergeOriginNoRepository, nil, branch, orgRetryHint); err != nil {
				got["origin"] = err.Error()
			}
			if _, err := cleanupSteps(noGH, branch, orgRetryHint); err != nil {
				got["no gh"] = err.Error()
			}
			steps, err := cleanupSteps(unread, branch, orgRetryHint)
			switch {
			case err != nil:
				got["unread"] = err.Error()
			case len(steps) == 1:
				got["unread"] = steps[0].unknown
			}
			var out strings.Builder
			if err := runCleanupSteps(stopped, nil, "/r", branch, orgRetryHint, []cleanupStep{merged}, &cleanupPrinter{w: &out}); err != nil {
				got["before a cleanup"] = err.Error()
			}
			if err := runCleanupSteps(stopped, nil, "/r", branch, orgRetryHint, nil, &cleanupPrinter{w: &out}); err != nil {
				got["after the steps"] = err.Error()
			}
			for what, suffix := range map[string]string{
				"origin":           "and run " + command + " again",
				"no gh":            "put gh on PATH and run " + command + " again",
				"unread":           "once gh reads its PR, run " + command + " again",
				"before a cleanup": "before it; run " + command + " again",
				"after the steps":  "; run " + command + " again for what is left",
			} {
				if !strings.HasSuffix(got[what], suffix) || strings.Count(got[what], "--state-dir") != 1 {
					t.Errorf("%s: %q; want it to end with %q, with --state-dir there only", what, got[what], suffix)
				}
			}
		})
	}
}

// With --state-dir and --branch, the refusal of a cleanup run where no main
// worktree is found names the command with both.
func TestOrgCleanup_NoMainWorktreeNamesTheBranchAndTheStateDir(t *testing.T) {
	fx := newMergeCLIFixture(t, false)
	outside := filepath.Join(filepath.Dir(fx.work), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(outside))
	t.Chdir(outside)

	stdout, _, err := runOrgCmdStreams(t, "cleanup", "--state-dir", fx.stateDir, "--branch", "feat/a")
	want := "so nothing was read or removed: run ralph org cleanup --branch feat/a --state-dir " + shellQuoteIfNeeded(fx.stateDir) +
		" from the main worktree"
	if err == nil || !strings.HasSuffix(err.Error(), want) || stdout != "" {
		t.Errorf("cleanup outside a repository: err %v, stdout %q; want an error ending with %q", err, stdout, want)
	}
}

// mergeCleanupRoot's reason names the cleanup of the branch with the hint
// merge was given, and with none without it.
func TestOrgMerge_CleanupRootNamesTheStateDir(t *testing.T) {
	outside := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(outside))
	t.Chdir(outside)
	for _, tc := range []struct{ hint, want string }{
		{"", "run ralph org cleanup --branch 'feat/a;b' from the main worktree"},
		{orgRetryHint, "run ralph org cleanup --branch 'feat/a;b' --state-dir '/my ledgers/org' from the main worktree"},
	} {
		root, skipped := mergeCleanupRoot(filepath.Join(outside, "state"), "flag", "feat/a;b", tc.hint)
		if root != "" || !strings.HasSuffix(skipped, tc.want) {
			t.Errorf("mergeCleanupRoot with the hint %q = %q, %q; want an empty root and a reason ending with %q", tc.hint, root,
				skipped, tc.want)
		}
	}
}

// retryCommandIn is the ralph org command s names to run again: from its
// last "run ralph org " to the last " again" after it.
func retryCommandIn(t *testing.T, s string) string {
	t.Helper()
	start, end := strings.LastIndex(s, "run ralph org "), strings.LastIndex(s, " again")
	if start < 0 || end < start {
		t.Fatalf("%q names no ralph org command to run again", s)
	}
	return s[start+len("run ") : end]
}

// retryCommandArgv is the words sh splits command into, as a person who
// pastes it runs it: a stub ralph first on PATH prints its arguments, each
// ended by a NUL. The first word is org.
func retryCommandArgv(t *testing.T, sh, command string) []string {
	t.Helper()
	stub := t.TempDir()
	writeStubScript(t, filepath.Join(stub, "ralph"), "#!/bin/sh\nfor a in \"$@\"; do printf '%s\\000' \"$a\"; done\n")
	run := exec.Command(sh, "-c", command)
	run.Env = append(os.Environ(), "PATH="+stub+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := run.Output()
	if err != nil {
		t.Fatalf("sh -c %q: %v", command, err)
	}
	return strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
}

// moveLedgerAway moves the fixture's ledger, the default one of the main
// worktree, to "<temp>/first ledger" and leaves another ledger at the default
// place: one line of another org, with no binding of feat/a, so a command that
// read it would refuse feat/a as not in the ledger. It returns the default
// ledger and what its manifest holds.
func moveLedgerAway(t *testing.T, fx *mergeCLIFixture) (defaultLedger, manifest string) {
	t.Helper()
	defaultLedger = fx.stateDir
	first := filepath.Join(mergeOrderTempDir(t), "first ledger")
	if err := os.Rename(defaultLedger, first); err != nil {
		t.Fatal(err)
	}
	fx.stateDir = first
	appendSeedEvent(t, defaultLedger, org.ManifestEvent{OrgID: "z", Event: org.EventDisbanded})
	return defaultLedger, fileBytesOrAbsent(t, org.ManifestPathIn(defaultLedger))
}

// AC1's replay for ralph org cleanup: with the ledger moved out of the
// repository and named with --state-dir, a refused cleanup (an untracked file
// in the worktree) prints a command with --state-dir, which sh splits into the
// words org cleanup --branch feat/a --state-dir <ledger>; run as printed, it
// is refused the same way, reading the ledger named first, where the default
// ledger has no feat/a. With the personal file naming the ledger and no
// --state-dir, the command carries none, and run as printed it reads the same
// ledger through the file. With the file gone and the untracked file removed,
// the first command cleans feat/a up and records worktree_cleaned in the
// ledger named first, while the default ledger stays as it was.
func TestOrgCleanup_RetryCommandReplaysOnTheLedgerNamedFirst(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Fatalf("sh is needed to split the printed command: %v", err)
	}
	fx := newCleanupCLIFixture(t)
	fx.squashMergeOnGitHub(t)
	defaultLedger, defaultManifest := moveLedgerAway(t, fx.mergeCLIFixture)
	writeTestFile(t, filepath.Join(fx.wt, "stray.txt"), "x\n")

	stdout, _, err := fx.cleanup(t)
	if err == nil || err.Error() != cleanupLeftErr(1) || !strings.HasPrefix(stdout, "SKIPPED: feat/a ") {
		t.Fatalf("cleanup --state-dir: err %v, stdout %q; want feat/a skipped", err, stdout)
	}
	command := retryCommandIn(t, stdout)
	argv := retryCommandArgv(t, sh, command)
	if want := []string{"org", "cleanup", "--branch", "feat/a", "--state-dir", fx.stateDir}; !slices.Equal(argv, want) {
		t.Fatalf("the command %q runs ralph with %q, want %q", command, argv, want)
	}
	replayed, _, replayErr := runOrgCmdStreams(t, argv[1:]...)
	if replayed != stdout || fmt.Sprint(replayErr) != fmt.Sprint(err) {
		t.Errorf("the command run as printed: err %v, stdout %q; want the same refusal from the ledger named first:\n%s", replayErr,
			replayed, stdout)
	}

	t.Run("the personal file names the ledger", func(t *testing.T) {
		t.Setenv(config.LocalConfigEnv, "")
		path := filepath.Join(fx.work, config.LocalConfigFileName)
		writeTestFile(t, path, fmt.Sprintf("[org]\nstate_dir = %q\n", fx.stateDir))
		defer func() { _ = os.Remove(path) }()
		stdout, _, err := runOrgCmdStreams(t, "cleanup", "--branch", "feat/a")
		bare := retryCommandIn(t, stdout)
		if err == nil || bare != "ralph org cleanup --branch feat/a" {
			t.Fatalf("cleanup with the personal file: err %v, stdout %q; want a command with no --state-dir", err, stdout)
		}
		argv := retryCommandArgv(t, sh, bare)
		if replayed, _, replayErr := runOrgCmdStreams(t, argv[1:]...); replayed != stdout || fmt.Sprint(replayErr) != fmt.Sprint(err) {
			t.Errorf("the bare command run as printed: err %v, stdout %q; want the same refusal:\n%s", replayErr, replayed, stdout)
		}
	})

	if err := os.Remove(filepath.Join(fx.wt, "stray.txt")); err != nil {
		t.Fatal(err)
	}
	stdout, _, err = runOrgCmdStreams(t, argv[1:]...)
	if err != nil || stdout != "CLEANED: feat/a org-a\n" {
		t.Fatalf("the command run as printed once the file is gone: err %v, stdout %q; want feat/a cleaned", err, stdout)
	}
	if got := fx.cleanedEvents(t); len(got) != 1 {
		t.Errorf("worktree_cleaned in the ledger named first = %+v; want one", got)
	}
	if got := fileBytesOrAbsent(t, org.ManifestPathIn(defaultLedger)); got != defaultManifest {
		t.Errorf("the default ledger changed:\nbefore %s\nafter  %s", defaultManifest, got)
	}
}

// AC1's replay for ralph org merge, and for the cleanup it runs: with the
// ledger moved out of the repository and named with --state-dir, a merge
// refused for a pending check names ralph org merge with --state-dir, which,
// run as printed once the check passed, merges and records pr_merged in the
// ledger named first; the cleanup it then runs is refused for an untracked
// file in the worktree, and its CLEANUP: skipped line names ralph org cleanup
// with --state-dir, which, run as printed once the file is gone, cleans feat/a
// up in that ledger too. The default ledger stays as it was throughout.
func TestOrgMerge_RetryCommandsReplayOnTheLedgerNamedFirst(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Fatalf("sh is needed to split the printed command: %v", err)
	}
	fx, wt := newMergeCLIWorktreeFixture(t)
	defaultLedger, defaultManifest := moveLedgerAway(t, fx)
	writeTestFile(t, filepath.Join(wt, "stray.txt"), "x\n")
	fx.reply(t, "pr-checks", `[{"bucket":"pending","name":"verify","state":"IN_PROGRESS","workflow":"ci"}]`)
	fx.fail(t, "pr-checks", 8, "")

	_, _, err = fx.merge(t)
	if err == nil || !strings.Contains(err.Error(), "are not pass or skipping") {
		t.Fatalf("merge with a check pending: %v; want the refusal of the checks", err)
	}
	argv := retryCommandArgv(t, sh, retryCommandIn(t, err.Error()))
	if want := []string{"org", "merge", "--branch", "feat/a", "--state-dir", fx.stateDir}; !slices.Equal(argv, want) {
		t.Fatalf("the merge's command runs ralph with %q, want %q", argv, want)
	}

	fx.reply(t, "pr-checks", `[{"bucket":"pass","name":"verify","state":"SUCCESS","workflow":"ci"}]`)
	for _, name := range []string{"pr-checks.rc", "pr-checks.err"} {
		if err := os.Remove(filepath.Join(fx.replies, name)); err != nil {
			t.Fatal(err)
		}
	}
	stdout, _, err := runOrgCmdStreams(t, argv[1:]...)
	lines := mergeCLILines(fx.head, true, "")
	if err != nil || !strings.HasPrefix(stdout, lines+"CLEANUP: skipped (") || len(fx.prMergedEvents(t)) != 1 {
		t.Fatalf("the merge's command run as printed: err %v, stdout %q; want the merge recorded in the ledger named first and "+
			"its cleanup skipped", err, stdout)
	}
	argv = retryCommandArgv(t, sh, retryCommandIn(t, strings.TrimPrefix(stdout, lines)))
	if want := []string{"org", "cleanup", "--branch", "feat/a", "--state-dir", fx.stateDir}; !slices.Equal(argv, want) {
		t.Fatalf("the cleanup's command runs ralph with %q, want %q", argv, want)
	}

	// ralph org cleanup reads the PR with gh pr list, which now finds it
	// merged at the gate's head.
	fx.reply(t, "pr-list", `[{"number":7,"state":"MERGED","url":"https://github.com/o/r/pull/7","headRefOid":"`+fx.head+
		`","baseRefName":"main","isCrossRepository":false}]`)
	if err := os.Remove(filepath.Join(wt, "stray.txt")); err != nil {
		t.Fatal(err)
	}
	stdout, _, err = runOrgCmdStreams(t, argv[1:]...)
	if err != nil || stdout != "CLEANED: feat/a org-a\n" {
		t.Fatalf("the cleanup's command run as printed: err %v, stdout %q; want feat/a cleaned", err, stdout)
	}
	cleaned := 0
	for _, ev := range readManifestEvents(t, org.ManifestPathIn(fx.stateDir)) {
		if ev.Event == org.EventWorktreeCleaned {
			cleaned++
		}
	}
	if cleaned != 1 {
		t.Errorf("the ledger named first has %d worktree_cleaned; want 1", cleaned)
	}
	if got := fileBytesOrAbsent(t, org.ManifestPathIn(defaultLedger)); got != defaultManifest {
		t.Errorf("the default ledger changed:\nbefore %s\nafter  %s", defaultManifest, got)
	}
}
