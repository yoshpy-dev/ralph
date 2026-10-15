package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for the `merge guarantee:` line and the merge_guarantee object of
// `ralph status` (plan docs/plans/active/2026-10-14-org-merge-guarantee.md,
// AC6, AC8 and AC10). The org layer's reading of the cache is covered by
// internal/org/mergeguarantee_test.go with a fake seam; these run the command
// with the real git in a temp repository whose origin each test sets
// (statusMGRepo), never this package's checkout, and write the cache only
// into temp state dirs, through org.SaveMergeGuarantee as doctor does.

// mergeGuaranteeTestDetail is the Detail of mergeGuaranteeCacheFor's caches.
const mergeGuaranteeTestDetail = "a ruleset requires branches to be up to date with 1 required check"

// mergeGuaranteeCacheFor is a github_protection cache of github.com/acme/app's
// default branch branch, checked at checkedAt.
func mergeGuaranteeCacheFor(branch string, checkedAt time.Time) org.MergeGuaranteeCheck {
	return org.MergeGuaranteeCheck{
		Kind: org.MergeGuaranteeGitHubProtection, Detail: mergeGuaranteeTestDetail,
		Host: "github.com", Repo: "acme/app", APIRepo: "acme/app", DefaultBranch: branch,
		CheckedAt: checkedAt.UTC().Format(time.RFC3339),
	}
}

// freshMergeGuaranteeCache is mergeGuaranteeCacheFor checked an hour ago.
func freshMergeGuaranteeCache(branch string) org.MergeGuaranteeCheck {
	return mergeGuaranteeCacheFor(branch, time.Now().Add(-time.Hour))
}

// cachedMergeGuarantee is c as ReadMergeGuarantee returns its kind.
func cachedMergeGuarantee(c org.MergeGuaranteeCheck) org.MergeGuarantee {
	return org.MergeGuarantee{
		Kind: c.Kind, Detail: c.Detail, Host: c.Host, Repo: c.Repo, APIRepo: c.APIRepo,
		DefaultBranch: c.DefaultBranch, CheckedAt: c.CheckedAt,
	}
}

// unknownFromCache is c made unknown for reason with detail, as
// ReadMergeGuarantee and ForDefaultBranch make it.
func unknownFromCache(c org.MergeGuaranteeCheck, reason, detail string) org.MergeGuarantee {
	g := cachedMergeGuarantee(c)
	g.LastKind, g.Kind, g.Reason, g.Detail = g.Kind, org.MergeGuaranteeUnknown, reason, detail
	return g
}

// cachedMergeGuaranteeText is the text of the line for c's kind, as
// mergeGuaranteeCacheFor's caches print it.
func cachedMergeGuaranteeText(c org.MergeGuaranteeCheck) string {
	return c.Kind + " (" + c.DefaultBranch + " of github.com/acme/app, checked " + c.CheckedAt + "): " + c.Detail
}

// writeMergeGuaranteeCache creates stateDir and writes c in it as the cache,
// with org.SaveMergeGuarantee, failing the test unless it was written.
func writeMergeGuaranteeCache(t *testing.T, stateDir string, c org.MergeGuaranteeCheck) {
	t.Helper()
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if s := org.SaveMergeGuarantee(stateDir, c); !s.Written || s.WriteErr != nil {
		t.Fatalf("save the merge-guarantee cache in %s: %+v", stateDir, s)
	}
}

// statusMGRepo makes a git repository in a temp dir whose origin is
// originURL, makes it the current directory, and returns it, so the
// guarantee's `git remote get-url origin` reads that origin.
func statusMGRepo(t *testing.T, originURL string) string {
	t.Helper()
	isolateFeatureGitEnv(t)
	dir := mergeOrderTempDir(t)
	runLedgerTestGit(t, "init", "--quiet", dir)
	runLedgerTestGit(t, "-C", dir, "remote", "add", "origin", originURL)
	t.Chdir(dir)
	return dir
}

// statusMGStateDir is a state dir path under a fresh temp dir, not created.
func statusMGStateDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(mergeOrderTempDir(t), "state")
}

// statusMergeGuaranteeLine returns the one `merge guarantee:` line of `ralph
// status` text output, failing the test when there is not exactly one.
func statusMergeGuaranteeLine(t *testing.T, out string) string {
	t.Helper()
	var found []string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "merge guarantee:") {
			found = append(found, line)
		}
	}
	if len(found) != 1 {
		t.Fatalf("want exactly one `merge guarantee:` line, got %d:\n%s", len(found), out)
	}
	return found[0]
}

// statusMergeGuaranteeJSONOf decodes the merge_guarantee object of `ralph
// status --json` output, failing the test when there is none.
func statusMergeGuaranteeJSONOf(t *testing.T, out string) org.MergeGuarantee {
	t.Helper()
	var payload struct {
		MergeGuarantee *org.MergeGuarantee `json:"merge_guarantee"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v\noutput:\n%s", err, out)
	}
	if payload.MergeGuarantee == nil {
		t.Fatalf("--json carries no merge_guarantee object:\n%s", out)
	}
	return *payload.MergeGuarantee
}

// checkStatusMergeGuarantee runs `ralph status --state-dir stateDir` and its
// --json, and checks the line and the object.
func checkStatusMergeGuarantee(t *testing.T, stateDir, wantLine string, wantJSON org.MergeGuarantee) {
	t.Helper()
	out, err := runStatusCmd(t, "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	if got := statusMergeGuaranteeLine(t, out); got != wantLine {
		t.Errorf("merge guarantee line:\n got %q\nwant %q", got, wantLine)
	}
	jsonOut, err := runStatusCmd(t, "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
	}
	if got := statusMergeGuaranteeJSONOf(t, jsonOut); got != wantJSON {
		t.Errorf("merge_guarantee:\n got %+v\nwant %+v", got, wantJSON)
	}
}

// AC6: with no usable cache the guarantee is unknown with the reason: the
// state dir does not exist (and status does not create it), it has no cache,
// the cache does not parse, or it was checked 25 hours ago (stale, with the
// kind and time it last had).
func TestStatusCmd_MergeGuaranteeWithoutAUsableCacheIsUnknown(t *testing.T) {
	stale := mergeGuaranteeCacheFor("main", time.Now().Add(-25*time.Hour))
	staleDetail := "stale: last github_protection at " + stale.CheckedAt + "; run ralph doctor"
	for _, tc := range []struct {
		name     string
		setup    func(t *testing.T, stateDir string)
		wantLine string
		wantJSON org.MergeGuarantee
	}{
		{"no state dir", func(*testing.T, string) {}, "merge guarantee: unknown (no org ledger yet)",
			org.MergeGuarantee{Kind: org.MergeGuaranteeUnknown, Reason: org.MergeGuaranteeReasonNoLedger, Detail: "no org ledger yet"}},
		{"no cache", func(t *testing.T, stateDir string) {
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatal(err)
			}
		}, "merge guarantee: unknown (not checked: run ralph doctor)",
			org.MergeGuarantee{Kind: org.MergeGuaranteeUnknown, Reason: org.MergeGuaranteeReasonNotChecked, Detail: "not checked: run ralph doctor"}},
		{"corrupt cache", func(t *testing.T, stateDir string) {
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatal(err)
			}
			writeTestFile(t, org.MergeGuaranteePathIn(stateDir), `{"version":1,"kind":"github_prot`)
		}, "merge guarantee: unknown (cache unreadable: run ralph doctor)",
			org.MergeGuarantee{Kind: org.MergeGuaranteeUnknown, Reason: org.MergeGuaranteeReasonUnreadable, Detail: "cache unreadable: run ralph doctor"}},
		{"cache checked 25 hours ago", func(t *testing.T, stateDir string) { writeMergeGuaranteeCache(t, stateDir, stale) },
			"merge guarantee: unknown (" + staleDetail + ")", unknownFromCache(stale, org.MergeGuaranteeReasonStale, staleDetail)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			statusMGRepo(t, "https://github.com/acme/app.git")
			stateDir := statusMGStateDir(t)
			tc.setup(t, stateDir)
			checkStatusMergeGuarantee(t, stateDir, tc.wantLine, tc.wantJSON)
			if tc.name == "no state dir" {
				if _, err := os.Stat(stateDir); !os.IsNotExist(err) {
					t.Errorf("status created the state dir %s (stat: %v)", stateDir, err)
				}
			}
		})
	}
}

// AC6: a fresh cache whose host and owner/repo are origin's, compared
// without case, gives its kind with the default branch, the repository and
// checked_at, then the detail.
func TestStatusCmd_MergeGuaranteeShowsTheCachedKindForTheSameOrigin(t *testing.T) {
	for _, origin := range []string{"https://github.com/acme/app.git", "https://github.com/ACME/App.git", "git@github.com:acme/app.git"} {
		t.Run(origin, func(t *testing.T) {
			statusMGRepo(t, origin)
			stateDir := statusMGStateDir(t)
			c := freshMergeGuaranteeCache("main")
			writeMergeGuaranteeCache(t, stateDir, c)
			want := "merge guarantee: github_protection (main of github.com/acme/app, checked " + c.CheckedAt + "): " +
				mergeGuaranteeTestDetail
			checkStatusMergeGuarantee(t, stateDir, want, cachedMergeGuarantee(c))
		})
	}
}

// AC10 on `ralph status`: the cache doctor wrote for github.com/acme/app
// gives its kind until origin is pointed at another repository with the same
// default branch name; then it is unknown, naming both repositories.
func TestStatusCmd_MergeGuaranteeIsUnknownOnceOriginPointsToAnotherRepository(t *testing.T) {
	repo := statusMGRepo(t, "https://github.com/acme/app.git")
	stateDir := statusMGStateDir(t)
	c := freshMergeGuaranteeCache("main")
	writeMergeGuaranteeCache(t, stateDir, c)
	checkStatusMergeGuarantee(t, stateDir, "merge guarantee: "+cachedMergeGuaranteeText(c), cachedMergeGuarantee(c))

	runLedgerTestGit(t, "-C", repo, "remote", "set-url", "origin", "https://github.com/acme/other.git")
	detail := "doctor checked github.com/acme/app; origin is now github.com/acme/other"
	checkStatusMergeGuarantee(t, stateDir, "merge guarantee: unknown ("+detail+")",
		unknownFromCache(c, org.MergeGuaranteeReasonOriginChanged, detail))
}

// AC6: run outside any repository (git cannot climb past
// GIT_CEILING_DIRECTORIES) with --state-dir naming a ledger with a fresh
// cache, origin cannot be read, so the guarantee is unknown with git's
// failure and the repository doctor checked.
func TestStatusCmd_MergeGuaranteeOutsideARepositoryIsOriginUnreadable(t *testing.T) {
	isolateFeatureGitEnv(t)
	tmp := mergeOrderTempDir(t)
	work := filepath.Join(tmp, "not-a-repo")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CEILING_DIRECTORIES", tmp)
	t.Chdir(work)
	stateDir := filepath.Join(tmp, "state")
	writeMergeGuaranteeCache(t, stateDir, freshMergeGuaranteeCache("main"))

	out, err := runStatusCmd(t, "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	line := statusMergeGuaranteeLine(t, out)
	const prefix = "merge guarantee: unknown (origin cannot be read here (git remote get-url origin failed: "
	const suffix = "); doctor checked github.com/acme/app)"
	if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, suffix) || !strings.Contains(line, "not a git repository") {
		t.Errorf("merge guarantee line = %q\nwant %q<git's not a git repository error>%q", line, prefix, suffix)
	}
	jsonOut, err := runStatusCmd(t, "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
	}
	if g := statusMergeGuaranteeJSONOf(t, jsonOut); g.Kind != org.MergeGuaranteeUnknown ||
		g.Reason != org.MergeGuaranteeReasonOriginUnreadable || g.LastKind != org.MergeGuaranteeGitHubProtection {
		t.Errorf("merge_guarantee = %+v; want unknown (origin_unreadable) with the last kind github_protection", g)
	}
}

// AC6: the line comes right after the `org watch:` line and the `director:`
// line that follows it (plan 2026-10-14-org-director-attach) on both text
// paths: with no org (after the empty-roster note, before the doctor hint)
// and with orgs (after the state-dir line, before the scaffold section and
// the rosters).
func TestStatusCmd_MergeGuaranteeLineFollowsTheOrgWatchAndDirectorLines(t *testing.T) {
	statusMGRepo(t, "https://github.com/acme/app.git")
	c := freshMergeGuaranteeCache("main")
	wantLine := "merge guarantee: " + cachedMergeGuaranteeText(c)
	for _, tc := range []struct {
		name   string
		seed   bool
		before string // the line before the org watch line
		after  string // the start of the line after the guarantee line
	}{
		{"no org", false, "no org runtime state found", "Run `ralph doctor`"},
		{"two orgs", true, "state-dir: ", "org_id: org-a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := statusMGStateDir(t)
			writeMergeGuaranteeCache(t, stateDir, c)
			if tc.seed {
				seedTwoOrgManifest(t, stateDir)
			}
			out, err := runStatusCmd(t, "--state-dir", stateDir)
			if err != nil {
				t.Fatalf("status: %v (output: %s)", err, out)
			}
			lines := strings.Split(out, "\n")
			if len(lines) < 5 || !strings.HasPrefix(lines[0], tc.before) || !strings.HasPrefix(lines[1], "org watch: ") ||
				!strings.HasPrefix(lines[2], "director: ") || lines[3] != wantLine || !strings.HasPrefix(lines[4], tc.after) {
				t.Errorf("want %q, the org watch line, the director line, %q, then %q, got:\n%s", tc.before, wantLine, tc.after, out)
			}
		})
	}
}

// AC6: --json carries merge_guarantee on the empty-roster path and the
// populated one alike, with the same value.
func TestStatusCmd_MergeGuaranteeJSONOnBothPaths(t *testing.T) {
	statusMGRepo(t, "https://github.com/acme/app.git")
	c := freshMergeGuaranteeCache("main")
	for _, seed := range []bool{false, true} {
		stateDir := statusMGStateDir(t)
		writeMergeGuaranteeCache(t, stateDir, c)
		if seed {
			seedTwoOrgManifest(t, stateDir)
		}
		jsonOut, err := runStatusCmd(t, "--state-dir", stateDir, "--json")
		if err != nil {
			t.Fatalf("status --json (orgs %v): %v (output: %s)", seed, err, jsonOut)
		}
		if got := statusMergeGuaranteeJSONOf(t, jsonOut); got != cachedMergeGuarantee(c) {
			t.Errorf("merge_guarantee with orgs %v:\n got %+v\nwant %+v", seed, got, cachedMergeGuarantee(c))
		}
	}
}

// statusMGGitLogger is the git of TestStatusCmd_MergeGuaranteeRunsNoGH's bin
// dir (fmt verb: the real git's path, shell-quoted): it logs its arguments to
// $STATUS_MG_GIT_LOG and execs the real git.
const statusMGGitLogger = `#!/bin/sh
printf '%%s\n' "$*" >> "$STATUS_MG_GIT_LOG"
exec %s "$@"
`

// statusMGFakeGH is the gh of the same bin dir: it logs its arguments to
// $STATUS_MG_GH_LOG and fails.
const statusMGFakeGH = `#!/bin/sh
printf '%s\n' "$*" >> "$STATUS_MG_GH_LOG"
exit 1
`

// statusMGMaxRevParsePerRun is the most `git rev-parse` calls one `ralph
// status --state-dir` run of TestStatusCmd_MergeGuaranteeRunsNoGH may make.
// It is the number one run makes today: the two of gitMainWorktree in a
// main checkout (rev-parse --git-dir --git-common-dir, then rev-parse
// --show-toplevel), run once, when withMainWorktreeDirectorEnabled tells
// whether --state-dir names the main worktree's ledger. A second read of
// the switch, or a loop around it, goes over it (the bound the self-review
// of org-director-attach asked for, Test intent changes (1)).
const statusMGMaxRevParsePerRun = 2

// AC6 and AC8: status never runs gh, and with a fresh cache its git calls are
// local: the `git remote get-url origin` of the guarantee, once a run, and
// the `git rev-parse` calls by which the director: line tells whether
// --state-dir names the main worktree's ledger, whose ralph.toml then holds
// [org.director].enabled (withMainWorktreeDirectorEnabled, plan
// 2026-10-14-org-director-attach), at most statusMGMaxRevParsePerRun a run,
// on both text and --json, with orgs and without. PATH holds only a bin dir
// whose gh and git log their calls (the git then runs the real one).
func TestStatusCmd_MergeGuaranteeRunsNoGH(t *testing.T) {
	statusMGRepo(t, "https://github.com/acme/app.git")
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("git is needed: %v", err)
	}
	if realGit, err = filepath.Abs(realGit); err != nil {
		t.Fatal(err)
	}
	tmp := mergeOrderTempDir(t)
	bin := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	writeStubScript(t, filepath.Join(bin, "git"), fmt.Sprintf(statusMGGitLogger, shellQuoteIfNeeded(realGit)))
	writeStubScript(t, filepath.Join(bin, "gh"), statusMGFakeGH)
	gitLog, ghLog := filepath.Join(tmp, "git.log"), filepath.Join(tmp, "gh.log")
	t.Setenv("STATUS_MG_GIT_LOG", gitLog)
	t.Setenv("STATUS_MG_GH_LOG", ghLog)
	t.Setenv("PATH", bin)

	emptyDir, populatedDir := statusMGStateDir(t), statusMGStateDir(t)
	c := freshMergeGuaranteeCache("main")
	for _, dir := range []string{emptyDir, populatedDir} {
		writeMergeGuaranteeCache(t, dir, c)
	}
	seedTwoOrgManifest(t, populatedDir)
	runs := 0
	for _, dir := range []string{emptyDir, populatedDir} {
		for _, extra := range [][]string{nil, {"--json"}} {
			before := len(readLogLines(t, gitLog))
			out, err := runStatusCmd(t, append([]string{"--state-dir", dir}, extra...)...)
			if err != nil {
				t.Fatalf("status %v: %v (output: %s)", extra, err, out)
			}
			runs++
			revParse := 0
			for _, call := range readLogLines(t, gitLog)[before:] {
				if strings.HasPrefix(call, "rev-parse ") {
					revParse++
				}
			}
			if revParse > statusMGMaxRevParsePerRun {
				t.Errorf("status %v ran git rev-parse %d times in one run; want at most %d", extra, revParse, statusMGMaxRevParsePerRun)
			}
			if extra == nil && statusMergeGuaranteeLine(t, out) != "merge guarantee: "+cachedMergeGuaranteeText(c) {
				t.Errorf("status read no cached kind with the logging git:\n%s", out)
			}
		}
	}
	if _, err := os.Stat(ghLog); !os.IsNotExist(err) {
		t.Errorf("status ran gh: %q (stat: %v)", readLogLines(t, ghLog), err)
	}
	gitCalls := readLogLines(t, gitLog)
	origin := 0
	for _, call := range gitCalls {
		switch {
		case call == "remote get-url origin":
			origin++
		case strings.HasPrefix(call, "rev-parse "):
		default:
			t.Errorf("status ran git %s; want only the local remote get-url origin and rev-parse", call)
		}
	}
	if origin != runs {
		t.Errorf("git remote get-url origin ran %d times over %d status runs: %q; want once a run", origin, runs, gitCalls)
	}
}

// A cache whose default branch holds a tab and a newline, and whose detail a
// newline, cannot break the line or forge a second one: they print escaped.
func TestStatusCmd_MergeGuaranteeFromACacheWithControlCharactersStaysOneLine(t *testing.T) {
	statusMGRepo(t, "https://github.com/acme/app.git")
	stateDir := statusMGStateDir(t)
	c := freshMergeGuaranteeCache("main\tx\nmerge guarantee: forged")
	c.Detail = "line one\nline two"
	writeMergeGuaranteeCache(t, stateDir, c)

	out, err := runStatusCmd(t, "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	want := `merge guarantee: github_protection (main\tx\nmerge guarantee: forged of github.com/acme/app, checked ` +
		c.CheckedAt + `): line one\nline two`
	if got := statusMergeGuaranteeLine(t, out); got != want {
		t.Errorf("merge guarantee line:\n got %q\nwant %q", got, want)
	}
}
