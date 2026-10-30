package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
)

// The doctor check "Merge guarantee" (plan
// docs/plans/active/2026-10-14-org-merge-guarantee.md, the doctor half of
// AC1 to AC5, and AC8): checkMergeGuarantee and runDoctorFull with a fake
// org.MergeOrderExec standing in for git and gh and a fake gh lookup, and
// the cache in temp ledger dirs. No test runs gh or reaches a network.

// The gh api endpoints of acme/app with the default branch main.
const (
	doctorMGRepoEP   = "repos/acme/app"
	doctorMGRulesEP  = "repos/acme/app/rules/branches/main"
	doctorMGBranchEP = "repos/acme/app/branches/main"
)

// doctorMGExec is a fake org.MergeOrderExec for acme/app on github.com:
// `git remote get-url origin` answers its https URL, and gh answers, per
// endpoint (the last argument), the output in replies. Anything else is an
// error.
type doctorMGExec struct {
	mu      sync.Mutex
	calls   []doctorMGCall
	replies map[string]string
}

type doctorMGCall struct {
	dir, name string
	args      []string
}

// newDoctorMGExec is a doctorMGExec whose main is of kind: github_protection
// (a ruleset requires main to be up to date, with the check ci) or
// pr_checks_only (no ruleset and no branch protection).
func newDoctorMGExec(t *testing.T, kind string) *doctorMGExec {
	t.Helper()
	e := &doctorMGExec{replies: map[string]string{
		doctorMGRepoEP: `{"full_name":"acme/app","default_branch":"main"}`,
		doctorMGBranchEP: `{"name":"main","protected":false,"protection":{"enabled":false,"required_status_checks":` +
			`{"enforcement_level":"off","contexts":[],"checks":[]}}}`,
	}}
	switch kind {
	case org.MergeGuaranteeGitHubProtection:
		e.replies[doctorMGRulesEP] = `[{"type":"required_status_checks","parameters":` +
			`{"strict_required_status_checks_policy":true,"required_status_checks":[{"context":"ci"}]}}]`
	case org.MergeGuaranteePRChecksOnly:
		e.replies[doctorMGRulesEP] = `[]`
	default:
		t.Fatalf("newDoctorMGExec: no fixture for kind %q", kind)
	}
	return e
}

func (e *doctorMGExec) run(_ context.Context, dir string, _ []string, name string, args ...string) (string, error) {
	e.mu.Lock()
	e.calls = append(e.calls, doctorMGCall{dir: dir, name: name, args: slices.Clone(args)})
	e.mu.Unlock()
	switch {
	case name == "git" && slices.Equal(args, []string{"remote", "get-url", "origin"}):
		return "https://github.com/acme/app.git\n", nil
	case name == "gh" && len(args) > 0:
		if out, ok := e.replies[args[len(args)-1]]; ok {
			return out, nil
		}
	}
	return "", fmt.Errorf("doctorMGExec: unexpected %s %q", name, args)
}

// names are the commands run, in order.
func (e *doctorMGExec) names() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []string
	for _, c := range e.calls {
		out = append(out, c.name)
	}
	return out
}

// assertDir fails the test unless every command ran in dir.
func (e *doctorMGExec) assertDir(t *testing.T, dir string) {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, c := range e.calls {
		if c.dir != dir {
			t.Errorf("%s %q ran in %q, want %q", c.name, c.args, c.dir, dir)
		}
	}
}

// doctorMGEnv is a checkMergeGuarantee resolver that runs e, finds gh only
// when ghFound, and caches into stateDir.
func doctorMGEnv(e *doctorMGExec, ghFound bool, stateDir string) func(string) (org.MergeGuaranteeParams, string, error) {
	lookPath := func(string) (string, error) { return "/usr/local/bin/gh", nil }
	if !ghFound {
		lookPath = func(file string) (string, error) { return "", &exec.Error{Name: file, Err: exec.ErrNotFound} }
	}
	return func(targetDir string) (org.MergeGuaranteeParams, string, error) {
		return org.MergeGuaranteeParams{Dir: targetDir, Exec: e.run, LookPath: lookPath}, stateDir, nil
	}
}

// doctorMGCheck is checkMergeGuarantee for /repo with doctorMGEnv, which
// resolves the ledger, so the check gives its one line.
func doctorMGCheck(t *testing.T, e *doctorMGExec, ghFound bool, stateDir string) checkResult {
	t.Helper()
	rs := checkMergeGuarantee(context.Background(), "/repo", doctorMGEnv(e, ghFound, stateDir))
	if len(rs) != 1 {
		t.Fatalf("checkMergeGuarantee gave %d lines, want 1: %+v", len(rs), rs)
	}
	return rs[0]
}

// readDoctorMGCache reads the cache in stateDir.
func readDoctorMGCache(t *testing.T, stateDir string) org.MergeGuaranteeCheck {
	t.Helper()
	data, err := os.ReadFile(org.MergeGuaranteePathIn(stateDir))
	if err != nil {
		t.Fatalf("read the cache: %v", err)
	}
	var c org.MergeGuaranteeCheck
	if err := json.Unmarshal(data, &c); err != nil {
		t.Fatalf("the cache does not parse: %v\n%s", err, data)
	}
	return c
}

// useDoctorMGEnv sets doctorMergeGuaranteeEnv to env for the test, and puts
// TestMain's pin back afterwards.
func useDoctorMGEnv(t *testing.T, env func(targetDir, stateDir string, stateDirSet bool) (org.MergeGuaranteeParams, string, error)) {
	t.Helper()
	orig := doctorMergeGuaranteeEnv
	doctorMergeGuaranteeEnv = env
	t.Cleanup(func() { doctorMergeGuaranteeEnv = orig })
}

// useDoctorMGEnvResolvingTheLedger is useDoctorMGEnv with e for git and gh
// (gh found) and the ledger the production resolver,
// mergeGuaranteeEnvFromOS, names for doctor's --state-dir, so doctor's real
// ledger resolution runs while no real git remote or gh call does.
func useDoctorMGEnvResolvingTheLedger(t *testing.T, e *doctorMGExec) {
	t.Helper()
	fake := doctorMGEnv(e, true, "")
	useDoctorMGEnv(t, func(targetDir, stateDir string, stateDirSet bool) (org.MergeGuaranteeParams, string, error) {
		params, _, _ := fake(targetDir)
		_, ledger, err := mergeGuaranteeEnvFromOS(targetDir, stateDir, stateDirSet)
		return params, ledger, err
	})
}

// TestCheckMergeGuarantee_StatusFollowsTheKindAndTheResultIsCached:
// github_protection is pass (AC1), pr_checks_only is warn and says what to
// set (AC3), and unknown is info with the reason, here gh not on PATH, which
// runs no gh (AC4). Each kind, unknown included, is cached in a ledger dir
// that exists, and the Detail is the kind, the cached Detail, and where it
// was cached (AC5).
func TestCheckMergeGuarantee_StatusFollowsTheKindAndTheResultIsCached(t *testing.T) {
	for _, tc := range []struct {
		name       string
		fixture    string
		ghFound    bool
		wantKind   string
		wantStatus string
		wantNames  []string
		wantDetail string
	}{
		{
			name: "github_protection", fixture: org.MergeGuaranteeGitHubProtection, ghFound: true,
			wantKind: org.MergeGuaranteeGitHubProtection, wantStatus: "pass", wantNames: []string{"git", "gh", "gh"},
			wantDetail: "a ruleset requires branches to be up to date before merging",
		},
		{
			name: "pr_checks_only", fixture: org.MergeGuaranteePRChecksOnly, ghFound: true,
			wantKind: org.MergeGuaranteePRChecksOnly, wantStatus: "warn", wantNames: []string{"git", "gh", "gh", "gh"},
			wantDetail: "set Require branches to be up to date before merging with at least one required status check, " +
				"in a ruleset or the branch protection of the branch, or add a merge queue with at least one required " +
				"status check to a ruleset",
		},
		{
			name: "unknown without gh", fixture: org.MergeGuaranteeGitHubProtection, ghFound: false,
			wantKind: org.MergeGuaranteeUnknown, wantStatus: "info", wantNames: []string{"git"},
			wantDetail: "gh is not on PATH",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newDoctorMGExec(t, tc.fixture)
			stateDir := t.TempDir()
			r := doctorMGCheck(t, e, tc.ghFound, stateDir)
			if r.Name != "Merge guarantee" || r.Status != tc.wantStatus {
				t.Errorf("result = %q %q, want Merge guarantee %q (Detail: %s)", r.Name, r.Status, tc.wantStatus, r.Detail)
			}
			if got := e.names(); !slices.Equal(got, tc.wantNames) {
				t.Errorf("commands run = %q, want %q", got, tc.wantNames)
			}
			e.assertDir(t, "/repo")

			cache := readDoctorMGCache(t, stateDir)
			if cache.Kind != tc.wantKind || cache.Host != "github.com" || cache.Repo != "acme/app" {
				t.Errorf("cache = %+v, want kind %s for github.com acme/app", cache, tc.wantKind)
			}
			want := tc.wantKind + ": " + cache.Detail + "; cached in " + org.MergeGuaranteePathIn(stateDir)
			if r.Detail != want {
				t.Errorf("Detail = %q\nwant      %q", r.Detail, want)
			}
			if !strings.Contains(r.Detail, tc.wantDetail) {
				t.Errorf("Detail = %q, want it to say %q", r.Detail, tc.wantDetail)
			}
		})
	}
}

// TestCheckMergeGuarantee_NoLedgerDirIsNotCreated: with no org state dir,
// nothing is written or created, the status still follows the kind, and the
// Detail says the result was not cached because there is no ledger yet
// (AC5).
func TestCheckMergeGuarantee_NoLedgerDirIsNotCreated(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "org")
	r := doctorMGCheck(t, newDoctorMGExec(t, org.MergeGuaranteeGitHubProtection), true, stateDir)
	if r.Status != "pass" {
		t.Errorf("Status = %q, want pass (Detail: %s)", r.Status, r.Detail)
	}
	if want := "; not cached: there is no org ledger yet at " + stateDir; !strings.HasSuffix(r.Detail, want) {
		t.Errorf("Detail = %q, want it to end with %q", r.Detail, want)
	}
	if _, err := os.Stat(stateDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the state dir exists after the check (stat: %v), want it not created", err)
	}
}

// TestCheckMergeGuarantee_FailedWriteRemovesTheOldCache: when the cache
// cannot be written (an empty directory holds its path), the old one is
// removed, the status still follows the kind, and the Detail says both
// (AC5).
func TestCheckMergeGuarantee_FailedWriteRemovesTheOldCache(t *testing.T) {
	stateDir := t.TempDir()
	cache := org.MergeGuaranteePathIn(stateDir)
	if err := os.Mkdir(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	r := doctorMGCheck(t, newDoctorMGExec(t, org.MergeGuaranteeGitHubProtection), true, stateDir)
	if r.Status != "pass" {
		t.Errorf("Status = %q, want pass (Detail: %s)", r.Status, r.Detail)
	}
	if !strings.Contains(r.Detail, "; not cached (org: write "+cache) || !strings.HasSuffix(r.Detail, "): the old cache was removed") {
		t.Errorf("Detail = %q, want the write error and the old cache removed", r.Detail)
	}
	if _, err := os.Stat(cache); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the old cache remains (stat: %v)", err)
	}
}

// TestCheckMergeGuarantee_StaleCacheThatCannotBeRemovedWarns: when the cache
// can be neither written nor removed (a non-empty directory holds its path),
// the check is warn whatever the kind, and the Detail says a stale cache
// remains (AC5).
func TestCheckMergeGuarantee_StaleCacheThatCannotBeRemovedWarns(t *testing.T) {
	for _, tc := range []struct {
		fixture string
		ghFound bool
		kind    string
	}{
		{org.MergeGuaranteeGitHubProtection, true, org.MergeGuaranteeGitHubProtection},
		{org.MergeGuaranteePRChecksOnly, true, org.MergeGuaranteePRChecksOnly},
		{org.MergeGuaranteeGitHubProtection, false, org.MergeGuaranteeUnknown},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			stateDir := t.TempDir()
			cache := org.MergeGuaranteePathIn(stateDir)
			if err := os.MkdirAll(filepath.Join(cache, "held"), 0o755); err != nil {
				t.Fatal(err)
			}
			r := doctorMGCheck(t, newDoctorMGExec(t, tc.fixture), tc.ghFound, stateDir)
			if r.Status != "warn" {
				t.Errorf("Status = %q, want warn (Detail: %s)", r.Status, r.Detail)
			}
			if !strings.HasPrefix(r.Detail, tc.kind+": ") {
				t.Errorf("Detail = %q, want it to start with the kind %s", r.Detail, tc.kind)
			}
			for _, want := range []string{"the old cache could not be removed (org: remove " + cache, "a stale cache remains",
				"may show an older kind"} {
				if !strings.Contains(r.Detail, want) {
					t.Errorf("Detail = %q, want it to say %q", r.Detail, want)
				}
			}
			if _, err := os.Stat(filepath.Join(cache, "held")); err != nil {
				t.Errorf("the directory at the cache's path is gone: %v", err)
			}
		})
	}
}

// TestMergeGuaranteeResult_EverySaveOutcome pins the status and the Detail
// for each kind and each save outcome, the one with no old cache to remove
// included: only a cache that could not be removed changes the kind's
// status, to warn (C3, C5).
func TestMergeGuaranteeResult_EverySaveOutcome(t *testing.T) {
	const path = "/p/.harness/state/org/merge-guarantee.json"
	writeErr := errors.New("org: write " + path + ": no space left on device")
	removeErr := errors.New("org: remove " + path + ": permission denied")
	for _, tc := range []struct {
		name       string
		save       org.MergeGuaranteeSave
		forceWarn  bool
		wantSuffix string
	}{
		{"written", org.MergeGuaranteeSave{Path: path, Written: true}, false, "; cached in " + path},
		{"no ledger", org.MergeGuaranteeSave{Path: path, NoLedger: true}, false,
			"; not cached: there is no org ledger yet at /p/.harness/state/org"},
		{"old cache removed", org.MergeGuaranteeSave{Path: path, WriteErr: writeErr, Removed: true}, false,
			"; not cached (" + writeErr.Error() + "): the old cache was removed"},
		{"no old cache", org.MergeGuaranteeSave{Path: path, WriteErr: writeErr}, false,
			"; not cached (" + writeErr.Error() + "): there was no old cache"},
		{"old cache remains", org.MergeGuaranteeSave{Path: path, WriteErr: writeErr, RemoveErr: removeErr}, true,
			"; not cached (" + writeErr.Error() + "), and the old cache could not be removed (" + removeErr.Error() +
				"): a stale cache remains, and ralph status and ralph org merge-order may show an older kind from it"},
	} {
		for kind, status := range map[string]string{
			org.MergeGuaranteeGitHubProtection: "pass",
			org.MergeGuaranteePRChecksOnly:     "warn",
			org.MergeGuaranteeUnknown:          "info",
		} {
			t.Run(tc.name+"/"+kind, func(t *testing.T) {
				wantStatus := status
				if tc.forceWarn {
					wantStatus = "warn"
				}
				r := mergeGuaranteeResult(org.MergeGuaranteeCheck{Kind: kind, Detail: "the reason"}, tc.save)
				want := kind + ": the reason" + tc.wantSuffix
				if r.Name != "Merge guarantee" || r.Status != wantStatus || r.Detail != want {
					t.Errorf("result = %+v\nwant %q %q %q", r, "Merge guarantee", wantStatus, want)
				}
				if strings.Contains(r.Detail, "\n") {
					t.Errorf("Detail spans lines: %q", r.Detail)
				}
			})
		}
	}
}

// TestMergeGuaranteeResult_EscapesControlCharacters: a ledger path with ESC
// and a line break reaches Detail as its Go escapes, never as raw bytes.
func TestMergeGuaranteeResult_EscapesControlCharacters(t *testing.T) {
	path := "/p/\x1b[31mred\nline/merge-guarantee.json"
	r := mergeGuaranteeResult(org.MergeGuaranteeCheck{Kind: org.MergeGuaranteeUnknown, Detail: "the reason"},
		org.MergeGuaranteeSave{Path: path, NoLedger: true})
	if strings.ContainsAny(r.Detail, "\x1b\n") {
		t.Errorf("Detail has a raw control character: %q", r.Detail)
	}
	if !strings.Contains(r.Detail, `\x1b[31mred\nline`) {
		t.Errorf("Detail lacks the escaped path: %q", r.Detail)
	}
}

// TestMergeGuaranteeEnvFromOS_RealCommandsAndTheResolvedStateDir: the
// production resolver leaves Exec and LookPath nil (real git and gh), runs
// them in targetDir, and caches into the state dir org.ResolveOrgStateDir
// names (here the personal file's [org].state_dir), not the one
// RALPH_ORG_STATE_DIR names, which ralph no longer reads (plan
// 2026-10-30-org-state-dir-local, D5).
func TestMergeGuaranteeEnvFromOS_RealCommandsAndTheResolvedStateDir(t *testing.T) {
	stateDir := t.TempDir()
	usePersonalStateDir(t, stateDir)
	t.Setenv(retiredOrgStateDirEnv, t.TempDir())
	params, got, err := mergeGuaranteeEnvFromOS("some/project", "", false)
	if err != nil {
		t.Fatalf("mergeGuaranteeEnvFromOS: %v", err)
	}
	if params.Dir != "some/project" || params.Exec != nil || params.LookPath != nil || params.Now != nil || params.Timeout != 0 {
		t.Errorf("params = %+v, want Dir some/project and every other field zero", params)
	}
	if got != stateDir {
		t.Errorf("state dir = %q, want %q", got, stateDir)
	}
}

// TestMergeGuaranteeEnvFromOS_StateDirFlagNamesTheLedgerWithoutGit: with
// doctor's --state-dir, the production resolver returns that dir made
// absolute, runs no git, and does not read the personal file, so one whose
// [org].state_dir is relative, which fails the same call without the flag,
// is no error (plan 2026-10-30-org-state-dir-local, AC7). The git on PATH
// is a stub that logs every run.
func TestMergeGuaranteeEnvFromOS_StateDirFlagNamesTheLedgerWithoutGit(t *testing.T) {
	r := newLegacyLedgerRepo(t)
	t.Setenv(config.LocalConfigEnv, "")
	path := filepath.Join(r.mainRoot, config.LocalConfigFileName)
	writeTestFile(t, path, "[org]\nstate_dir = \"ledger\"\n")
	if _, _, err := mergeGuaranteeEnvFromOS("some/project", "", false); err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("without --state-dir: err = %v; want the personal file's error naming %s", err, path)
	}

	binDir := t.TempDir()
	gitLog := filepath.Join(binDir, "git.log")
	writeStubBin(t, binDir, "git", gitLog)
	t.Setenv("PATH", binDir)
	absLedger := filepath.Join(t.TempDir(), "ledger")
	for _, tc := range []struct{ flag, want string }{
		{absLedger, absLedger},
		{"rel/ledger", filepath.Join(r.worktree, "rel", "ledger")},
	} {
		params, got, err := mergeGuaranteeEnvFromOS("some/project", tc.flag, true)
		if err != nil || got != tc.want {
			t.Errorf("--state-dir %s: state dir = %q, %v; want %q and no error", tc.flag, got, err, tc.want)
		}
		if params.Dir != "some/project" || params.Exec != nil || params.LookPath != nil {
			t.Errorf("--state-dir %s: params = %+v, want Dir some/project and real git and gh", tc.flag, params)
		}
	}
	if runs, err := os.ReadFile(gitLog); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("git ran with --state-dir (log read: %v):\n%s", err, runs)
	}
}

// doctorMGProject is a ralph init'ed temp project with stub claude, codex
// and go first on PATH (so every other check passes), and agmsg's home
// pinned to a dir that does not exist.
func doctorMGProject(t *testing.T) (dir, binDir string) {
	t.Helper()
	setupTestEmbedFS(t)
	Version = "0.1.0-test"
	dir = t.TempDir()
	if err := executeInit(dir, initConfig{ProjectName: "test", Packs: []string{"golang"}}, false); err != nil {
		t.Fatalf("init: %v", err)
	}
	binDir = filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, bin := range []string{"claude", "codex", "go"} {
		writeStubBin(t, binDir, bin, "")
	}
	t.Setenv("PATH", binDir)
	setAgmsgHomeOverride(t, filepath.Join(dir, "no-such-agmsg-home"))
	return dir, binDir
}

// TestRunDoctorFull_PrintsTheMergeGuarantee: runDoctorFull runs the check in
// its targetDir and prints its line, and no kind changes doctor's exit code
// (AC1, AC3, AC4).
func TestRunDoctorFull_PrintsTheMergeGuarantee(t *testing.T) {
	dir, _ := doctorMGProject(t)
	for _, tc := range []struct {
		name     string
		fixture  string
		ghFound  bool
		wantLine string
	}{
		{"pass", org.MergeGuaranteeGitHubProtection, true, "  ✓ Merge guarantee: pass — github_protection: main, the default branch of"},
		{"warn", org.MergeGuaranteePRChecksOnly, true, "  ⚠ Merge guarantee: warn — pr_checks_only: main, the default branch of"},
		{"info", org.MergeGuaranteeGitHubProtection, false, "  ℹ Merge guarantee: info — unknown: gh is not on PATH"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newDoctorMGExec(t, tc.fixture)
			stateDir := t.TempDir()
			env := doctorMGEnv(e, tc.ghFound, stateDir)
			var gotDir string
			useDoctorMGEnv(t, func(targetDir, _ string, _ bool) (org.MergeGuaranteeParams, string, error) {
				gotDir = targetDir
				return env(targetDir)
			})
			var out string
			if err := captureStdoutText(t, &out, func() error { return runDoctorFull(dir, false, false) }); err != nil {
				t.Fatalf("runDoctorFull: %v\noutput:\n%s", err, out)
			}
			if gotDir != dir {
				t.Errorf("the check ran for %q, want the targetDir %q", gotDir, dir)
			}
			e.assertDir(t, dir)
			if !strings.Contains(out, tc.wantLine) {
				t.Errorf("output has no line starting %q:\n%s", tc.wantLine, out)
			}
			if !strings.Contains(out, "All checks passed.") {
				t.Errorf("output does not say all checks passed:\n%s", out)
			}
			if _, err := os.Stat(org.MergeGuaranteePathIn(stateDir)); err != nil {
				t.Errorf("no cache in the ledger dir: %v", err)
			}
		})
	}
}

// TestRunDoctor_TestMainPinRunsNoGitOrGh: with TestMain's pin in place, a
// default runDoctor runs neither gh nor `git remote get-url origin` for this
// check and writes no cache (AC8). The stub git answers get-url with a
// GitHub URL and the stub gh is on PATH, so without the pin the real
// commands would reach the stub gh and log it.
func TestRunDoctor_TestMainPinRunsNoGitOrGh(t *testing.T) {
	dir, binDir := doctorMGProject(t)
	ghLog := filepath.Join(dir, "gh-invocations.log")
	writeStubBin(t, binDir, "gh", ghLog)
	gitLog := filepath.Join(dir, "git-invocations.log")
	gitStub := "#!/bin/sh\necho \"$*\" >> '" + gitLog + "'\n" +
		"if [ \"$*\" = 'remote get-url origin' ]; then echo https://github.com/acme/app.git; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(filepath.Join(binDir, "git"), []byte(gitStub), 0o755); err != nil {
		t.Fatal(err)
	}

	params, stateDir, err := doctorMergeGuaranteeEnv(dir, "", false)
	if err != nil {
		t.Fatalf("TestMain's pin returned an error: %v", err)
	}
	if params.Exec == nil || params.LookPath == nil {
		t.Errorf("TestMain's pin leaves Exec or LookPath nil (real commands): %+v", params)
	}
	var out string
	if err := captureStdoutText(t, &out, func() error { return runDoctor(dir) }); err != nil {
		t.Fatalf("runDoctor: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "  ℹ Merge guarantee: info — unknown: git remote get-url origin failed (git remote get-url origin "+
		"not run: commands are disabled in tests)") {
		t.Errorf("output has no pinned merge guarantee line:\n%s", out)
	}
	ghRuns, err := os.ReadFile(ghLog)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read the gh log: %v", err)
	}
	if len(ghRuns) > 0 {
		t.Errorf("gh ran under TestMain's pin:\n%s", ghRuns)
	}
	gitRuns, err := os.ReadFile(gitLog)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read the git log: %v", err)
	}
	if strings.Contains(string(gitRuns), "remote get-url") {
		t.Errorf("git remote get-url ran under TestMain's pin:\n%s", gitRuns)
	}
	if _, err := os.Stat(stateDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("TestMain's state dir %s exists (stat: %v), want nothing written", stateDir, err)
	}
}
