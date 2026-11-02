package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// The commands of the verbs other than merge and cleanup that ralph org merge
// and ralph org cleanup print at the command line (ralph org merge-order, with
// --all or not) carry --state-dir only when the command was given it (plan
// docs/plans/active/2026-11-02-org-ledger-leftovers.md, S3: AC3, tech-debt
// 395), after the verb's own arguments, as org.ShellWord writes the dir
// (orgVerbHint). internal/org/merge_hint_state_dir_test.go has the org
// layer's. The replay test takes the merge-order command out of a refusal and
// runs it as printed, to show that it reads the ledger named first and not the
// one ralph finds by default.

// With no --branch, ralph org merge refuses before the ledger is resolved and
// names ralph org merge-order with the --state-dir it was given, made
// absolute, and with none without it.
func TestOrgMerge_HintStateDirInTheRefusalOfAMissingBranch(t *testing.T) {
	isolateFeatureGitEnv(t)
	tmp := mergeOrderTempDir(t)
	t.Chdir(tmp)
	abs := filepath.Join(tmp, "my ledgers", "org")
	for _, tc := range []struct {
		args    []string
		command string
	}{
		{[]string{"merge"}, "ralph org merge-order"},
		{[]string{"merge", "--state-dir", abs}, "ralph org merge-order --state-dir '" + abs + "'"},
		{[]string{"merge", "--branch", " ", "--state-dir", filepath.Join("my ledgers", "org")},
			"ralph org merge-order --state-dir '" + abs + "'"},
	} {
		stdout, _, err := runOrgCmdStreams(t, tc.args...)
		want := "org: merge: --branch is required: name the feature branch whose PR to merge (" + tc.command +
			" lists the ledger's branches)"
		if err == nil || err.Error() != want || stdout != "" {
			t.Errorf("%q: err %v, stdout %q; want the error %q", tc.args, err, stdout, want)
		}
	}
	if _, err := os.Stat(abs); err == nil {
		t.Errorf("the refusal made %s", abs)
	}
}

// printMergeResult's error for a result with no outcome names ralph org
// merge-order with the hint, and with none without it.
func TestPrintMergeResult_HintStateDirInTheNoOutcomeError(t *testing.T) {
	for _, tc := range []struct{ hint, command string }{
		{"", "ralph org merge-order"},
		{orgRetryHint, "ralph org merge-order --state-dir '/my ledgers/org'"},
	} {
		var out strings.Builder
		err := printMergeResult(&out, org.MergeResult{Branch: "feat/a"}, nil, tc.hint)
		want := "so whether its PR was merged is unknown: run " + tc.command + " to see the branch"
		if err == nil || !strings.HasSuffix(err.Error(), want) || out.String() != "" {
			t.Errorf("printMergeResult with the hint %q: err %v, stdout %q; want an error ending with %q", tc.hint, err,
				out.String(), want)
		}
	}
}

// cleanupSteps' refusal of a --branch the ledger does not name, whether it
// names other branches or none, names ralph org merge-order --all with the
// hint, escaped as cleanupRetryCommand escapes it, and with none without it.
func TestOrgCleanup_HintStateDirInTheRefusalOfABranchNotInTheLedger(t *testing.T) {
	bind := &org.MergeOrderBinding{OrgID: "a", Feature: "f", Split: "s", TS: "t"}
	other := org.MergeOrderResult{GHAvailable: true, Entries: []org.MergeOrderEntry{{Branch: "feat/a", Class: org.MergeClassNoGate,
		PRLookup: org.MergeOrderPRLookupNone, Binding: bind}}}
	for _, tc := range []struct{ hint, command string }{
		{"", "ralph org merge-order --all"},
		{orgRetryHint, "ralph org merge-order --all --state-dir '/my ledgers/org'"},
		{"/st\x1bte", `ralph org merge-order --all --state-dir '/st\x1bte'`},
	} {
		for _, res := range []org.MergeOrderResult{other, {GHAvailable: true}} {
			_, err := cleanupSteps(res, "feat/nope", tc.hint)
			want := "org: cleanup: no feature binding or gate record in the ledger names the branch feat/nope, so nothing was " +
				"removed: " + tc.command + " lists the ledger's branches"
			if err == nil || err.Error() != want {
				t.Errorf("cleanupSteps with %d entries and the hint %q: %v; want %q", len(res.Entries), tc.hint, err, want)
			}
		}
	}
}

// mergeOrderCommandIn is the ralph org merge-order command s names: from its
// first "ralph org merge-order" to the " lists" after it.
func mergeOrderCommandIn(t *testing.T, s string) string {
	t.Helper()
	start := strings.Index(s, "ralph org merge-order")
	end := strings.Index(s[max(start, 0):], " lists")
	if start < 0 || end < 0 {
		t.Fatalf("%q names no ralph org merge-order command", s)
	}
	return s[start : start+end]
}

// mergeOrderListedAt matches the times in merge-order's output (when origin
// was listed, on the first line and in a note), which differ between two
// runs.
var mergeOrderListedAt = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})`)

// AC3's replay: with the ledger moved out of the repository and named with
// --state-dir, ralph org cleanup --branch of a branch the ledger does not name
// refuses and names ralph org merge-order --all with --state-dir, which sh
// splits into the words org merge-order --all --state-dir <ledger>; run as
// printed, it lists feat/a from the ledger named first, as merge-order run on
// that ledger does, while the default ledger, which has no feat/a, lists
// none. With the personal file naming the ledger and no --state-dir, the
// command carries none, and run as printed it reads the same ledger through
// the file. The default ledger stays as it was throughout.
func TestOrgCleanup_HintStateDirMergeOrderReplaysOnTheLedgerNamedFirst(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Fatalf("sh is needed to split the printed command: %v", err)
	}
	fx := newCleanupCLIFixture(t)
	fx.squashMergeOnGitHub(t)
	defaultLedger, defaultManifest := moveLedgerAway(t, fx.mergeCLIFixture)
	listed := func(t *testing.T, args ...string) string {
		t.Helper()
		stdout, stderr, err := runOrgCmdStreams(t, args...)
		if err != nil {
			t.Fatalf("%q: err %v, stderr %q, stdout %q", args, err, stderr, stdout)
		}
		return mergeOrderListedAt.ReplaceAllString(stdout, "<time>")
	}
	first := listed(t, "merge-order", "--all", "--state-dir", fx.stateDir)
	if !strings.Contains(first, "\tfeat/a\t") {
		t.Fatalf("merge-order --all on the ledger named first lists no feat/a:\n%s", first)
	}
	if got := listed(t, "merge-order", "--all"); strings.Contains(got, "feat/a") {
		t.Fatalf("merge-order --all on the default ledger lists feat/a:\n%s", got)
	}

	stdout, _, err := fx.cleanup(t, "--branch", "feat/nope")
	if err == nil || stdout != "" {
		t.Fatalf("cleanup --branch feat/nope: err %v, stdout %q; want the refusal", err, stdout)
	}
	command := mergeOrderCommandIn(t, err.Error())
	argv := retryCommandArgv(t, sh, command)
	if want := []string{"org", "merge-order", "--all", "--state-dir", fx.stateDir}; !slices.Equal(argv, want) {
		t.Fatalf("the command %q runs ralph with %q, want %q", command, argv, want)
	}
	if got := listed(t, argv[1:]...); got != first {
		t.Errorf("the command run as printed lists\n%s\nwant what the ledger named first lists\n%s", got, first)
	}

	t.Run("the personal file names the ledger", func(t *testing.T) {
		t.Setenv(config.LocalConfigEnv, "")
		path := filepath.Join(fx.work, config.LocalConfigFileName)
		writeTestFile(t, path, fmt.Sprintf("[org]\nstate_dir = %q\n", fx.stateDir))
		defer func() { _ = os.Remove(path) }()
		_, _, err := runOrgCmdStreams(t, "cleanup", "--branch", "feat/nope")
		if err == nil {
			t.Fatal("cleanup --branch feat/nope with the personal file exited 0; want the refusal")
		}
		bare := mergeOrderCommandIn(t, err.Error())
		if bare != "ralph org merge-order --all" {
			t.Fatalf("cleanup with the personal file names %q; want the command with no --state-dir", bare)
		}
		argv := retryCommandArgv(t, sh, bare)
		if got := listed(t, argv[1:]...); got != first {
			t.Errorf("the bare command run as printed lists\n%s\nwant what the ledger named first lists\n%s", got, first)
		}
	})

	if got := fileBytesOrAbsent(t, org.ManifestPathIn(defaultLedger)); got != defaultManifest {
		t.Errorf("the default ledger changed:\nbefore %s\nafter  %s", defaultManifest, got)
	}
}
