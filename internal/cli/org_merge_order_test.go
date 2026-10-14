package cli

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org merge-order` (plan
// docs/plans/active/2026-10-14-org-merge-order.md, the CLI half of AC1, AC2,
// AC5 to AC9 and AC11, and the Integration line of its test plan). The org
// layer's classes, order, PR choice, repository check of gh's PRs (C5) and
// grouping of gh failures are covered by internal/org/mergeorder_test.go
// with a fake seam; these run the command with the real git against a bare
// origin in a temp dir, and with a fake gh or none, and check what it prints
// (the notes for a gh that fails and for a PR from another repository than
// origin's included), its exit (the error Execute returns), the ledger it
// picks, and that it writes nothing.
//
// Every test sets PATH itself to a temp bin dir: a git wrapper that execs the
// real git by its absolute path, except that it answers `git remote get-url
// origin` with $MERGE_ORDER_ORIGIN_URL when that is set
// (mergeOrderGitWrapper), and the fake gh (mergeOrderFakeGH) when the test
// wants one. No directory of the developer's PATH is on it, so no test can
// reach a real gh (Homebrew puts gh next to git). None runs in parallel: they
// set PATH and chdir.
//
// Origin is a local path unless a test sets $MERGE_ORDER_ORIGIN_URL. A local
// path names no owner/repo, so the repository check is skipped and gh's PRs
// are used as they are.

// mergeOrderFakeGH is the fake gh: it appends one line per call to
// $MERGE_ORDER_GH_LOG (GH_PROMPT_DISABLED's value, then its arguments), and
// prints $MERGE_ORDER_GH_PRS/<the --head value>.json, or [] when there is no
// such file. It uses only shell builtins, since PATH holds no other tool.
const mergeOrderFakeGH = `#!/bin/sh
printf 'GH_PROMPT_DISABLED=%s %s\n' "$GH_PROMPT_DISABLED" "$*" >> "$MERGE_ORDER_GH_LOG"
head=
while [ "$#" -gt 0 ]; do
  if [ "$1" = --head ] && [ "$#" -ge 2 ]; then
    head=$2
  fi
  shift
done
file="$MERGE_ORDER_GH_PRS/$head.json"
if [ -f "$file" ]; then
  while IFS= read -r line || [ -n "$line" ]; do
    printf '%s\n' "$line"
  done < "$file"
else
  printf '[]\n'
fi
`

// mergeOrderCallLogger is a fake command (fmt verb: its name) that appends
// `<name> <args>` to $MERGE_ORDER_CALLS and fails: for the tests that check
// that no git or gh runs at all.
const mergeOrderCallLogger = `#!/bin/sh
printf '%%s %%s\n' %s "$*" >> "$MERGE_ORDER_CALLS"
exit 1
`

// mergeOrderTableHeader is the header line of merge-order's table.
const mergeOrderTableHeader = "RANK\tCLASS\tBRANCH\tPR\tGATE_HEAD\tGATE_BASE\tORG\tDETAIL"

// mergeOrderGHCall is the fake gh's log line for merge-order's PR lookup of
// branch.
func mergeOrderGHCall(branch string) string {
	return "GH_PROMPT_DISABLED=1 pr list --head " + branch + " --state all --limit 1000 --json number,state,url,headRefOid,baseRefName,isCrossRepository"
}

// mergeOrderFixture is a repository whose origin is a bare repository in the
// same temp dir, and an empty ledger. The current directory is work, and
// PATH holds only bin.
type mergeOrderFixture struct {
	stateDir string // the ledger, passed as --state-dir
	bare     string // origin: a bare repository whose HEAD names main
	work     string // the current directory, one commit on main, pushed
	bin      string // PATH: the git wrapper, and the fake gh with withGH
	ghPRs    string // the fake gh's canned output: <ghPRs>/<branch>.json
	ghLog    string // the fake gh's calls, one line each
}

func newMergeOrderFixture(t *testing.T, withGH bool) *mergeOrderFixture {
	t.Helper()
	isolateFeatureGitEnv(t)
	tmp := mergeOrderTempDir(t)
	fx := &mergeOrderFixture{
		stateDir: filepath.Join(tmp, "state"), bare: filepath.Join(tmp, "origin.git"), work: filepath.Join(tmp, "work"),
		bin: filepath.Join(tmp, "bin"), ghPRs: filepath.Join(tmp, "gh-prs"), ghLog: filepath.Join(tmp, "gh.log"),
	}
	setMergeOrderPATH(t, fx.bin, withGH)
	t.Setenv("MERGE_ORDER_GH_PRS", fx.ghPRs)
	t.Setenv("MERGE_ORDER_GH_LOG", fx.ghLog)
	runLedgerTestGit(t, "init", "--quiet", "--bare", "-b", "main", fx.bare)
	runLedgerTestGit(t, "init", "--quiet", "-b", "main", fx.work)
	gateCLICommit(t, fx.work, "init")
	runLedgerTestGit(t, "-C", fx.work, "remote", "add", "origin", fx.bare)
	fx.push(t, "main")
	t.Chdir(fx.work)
	return fx
}

// mergeOrderTempDir is t.TempDir() with symlinks resolved.
func mergeOrderTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// mergeOrderGitWrapper is the git of the tests' bin dir (fmt verb: the real
// git's path, shell-quoted): it execs the real git, except that, when
// $MERGE_ORDER_ORIGIN_URL is set, it prints that for `git remote get-url
// origin`, so a test can give origin a GitHub URL while the ls-remote still
// reads the bare origin. It uses only shell builtins.
const mergeOrderGitWrapper = `#!/bin/sh
if [ -n "$MERGE_ORDER_ORIGIN_URL" ] && [ "$#" -eq 3 ] && [ "$1" = remote ] && [ "$2" = get-url ] && [ "$3" = origin ]; then
  printf '%%s\n' "$MERGE_ORDER_ORIGIN_URL"
  exit 0
fi
exec %s "$@"
`

// setMergeOrderPATH writes, in bin, a git that execs the real git (looked up
// on the PATH before this call; mergeOrderGitWrapper) and, with withGH, the
// fake gh, sets PATH to bin alone, and clears MERGE_ORDER_ORIGIN_URL.
func setMergeOrderPATH(t *testing.T, bin string, withGH bool) {
	t.Helper()
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("git is needed for the merge-order tests: %v", err)
	}
	if realGit, err = filepath.Abs(realGit); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	writeStubScript(t, filepath.Join(bin, "git"), fmt.Sprintf(mergeOrderGitWrapper, shellQuoteIfNeeded(realGit)))
	if withGH {
		writeStubScript(t, filepath.Join(bin, "gh"), mergeOrderFakeGH)
	}
	t.Setenv("MERGE_ORDER_ORIGIN_URL", "")
	t.Setenv("PATH", bin)
}

// commitOn points refs/heads/<branch> at a new empty commit on top of from
// (a commit or a ref), with msg, without touching the checkout, and returns
// it. Each msg makes a distinct commit, so two branches made in the same
// second never share a tip.
func (fx *mergeOrderFixture) commitOn(t *testing.T, branch, from, msg string) string {
	t.Helper()
	sha := gateCLIGitOut(t, fx.work, "-c", "user.name=t", "-c", "user.email=t@example.com",
		"commit-tree", "-p", from, "-m", msg, from+"^{tree}")
	runLedgerTestGit(t, "-C", fx.work, "update-ref", "refs/heads/"+branch, sha)
	return sha
}

// branch makes branch one commit ahead of main and returns its tip.
func (fx *mergeOrderFixture) branch(t *testing.T, branch string) string {
	t.Helper()
	return fx.commitOn(t, branch, "refs/heads/main", "work on "+branch)
}

// tip returns the tip of refs/heads/<branch> in work.
func (fx *mergeOrderFixture) tip(t *testing.T, branch string) string {
	t.Helper()
	return gateCLIGitOut(t, fx.work, "rev-parse", "refs/heads/"+branch)
}

// push pushes refspecs from work to origin. Every push in these tests
// creates a branch or moves one forward.
func (fx *mergeOrderFixture) push(t *testing.T, refspecs ...string) {
	t.Helper()
	runLedgerTestGit(t, append([]string{"-C", fx.work, "push", "--quiet", "origin"}, refspecs...)...)
}

// bind records orgID bound to feature of split plan demo on branch, as
// `ralph org start --plan` records it (seedGateOrg's scope_reserved).
func (fx *mergeOrderFixture) bind(t *testing.T, orgID, feature, branch string) {
	t.Helper()
	appendSeedEvent(t, fx.stateDir, org.ManifestEvent{
		OrgID: orgID, Event: org.EventScopeReserved, Worktree: fx.work,
		Details: "paths=internal/" + orgID + "/ split=demo feature=" + feature + " digest=0123456789ab branch=" + branch + " approval=per-plan",
	})
}

// gate records orgID's gate on branch, as `ralph org gate` records it:
// gate_passed when reason is empty, else gate_failed with reason, on head
// and base against refs/remotes/origin/main.
func (fx *mergeOrderFixture) gate(t *testing.T, orgID, branch, reason, head, base string) {
	t.Helper()
	event := org.EventGatePassed
	if reason != "" {
		event = org.EventGateFailed
	}
	appendSeedEvent(t, fx.stateDir, org.ManifestEvent{
		OrgID: orgID, Event: event, Details: mergeOrderGateDetails(reason, head, base, "refs/remotes/origin/main", branch),
	})
}

// mergeOrderGateDetails is the Details of a gate record (gateCLIRecord with
// the branch and base_ref given), by tester with no herdr pane.
func mergeOrderGateDetails(reason, head, base, baseRef, branch string) string {
	d := "head=" + head + " base=" + base + " base_ref=" + baseRef + " branch=" + branch +
		" by=tester herdr_pane=unset log=" + strconv.Quote("/state/gates/x.log")
	if reason != "" {
		d = "reason=" + reason + " " + d
	}
	return d
}

// mergeOrderGHPR is one PR of the fake gh's output, in gh's field names.
type mergeOrderGHPR struct {
	Number            int    `json:"number"`
	State             string `json:"state"`
	URL               string `json:"url"`
	HeadRefOid        string `json:"headRefOid"`
	BaseRefName       string `json:"baseRefName"`
	IsCrossRepository bool   `json:"isCrossRepository"`
}

// prs makes the fake gh print prs for `gh pr list --head <branch>`.
func (fx *mergeOrderFixture) prs(t *testing.T, branch string, prs ...mergeOrderGHPR) {
	t.Helper()
	data, err := json.Marshal(prs)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(fx.ghPRs, filepath.FromSlash(branch)+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, path, string(data)+"\n")
}

// run runs `ralph org merge-order --state-dir <the fixture's ledger>` with
// extra after it.
func (fx *mergeOrderFixture) run(t *testing.T, extra ...string) (stdout, stderr string, err error) {
	t.Helper()
	return runOrgCmdStreams(t, append([]string{"merge-order", "--state-dir", fx.stateDir}, extra...)...)
}

// runJSON runs merge-order --json with extra, checks that it exits 0 with
// nothing on stderr, and returns stdout parsed whole.
func (fx *mergeOrderFixture) runJSON(t *testing.T, extra ...string) (org.MergeOrderResult, string) {
	t.Helper()
	stdout, stderr, err := fx.run(t, append([]string{"--json"}, extra...)...)
	if err != nil || stderr != "" {
		t.Fatalf("merge-order --json: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	var res org.MergeOrderResult
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		t.Fatalf("merge-order --json printed something that is not one JSON value: %v\n%s", err, stdout)
	}
	return res, stdout
}

// ghCalls returns the fake gh's log lines, nil when it never ran.
func (fx *mergeOrderFixture) ghCalls(t *testing.T) []string {
	t.Helper()
	return readLogLines(t, fx.ghLog)
}

// mergeOrderText is merge-order's text output, read back.
type mergeOrderText struct {
	header    string     // the origin/<branch> line, "" when there is none
	guarantee string     // the merge guarantee line, "merge guarantee: " cut
	empty     bool       // the line "no branch to list" was printed
	rows      [][]string // the table's rows, each split on tabs into 8 fields
	notes     []string   // the note lines, "note: " cut
}

// parseMergeOrderText reads out, failing the test on any line out of the
// shape printMergeOrder prints: the header, the merge guarantee line (first
// when there is no header), "no branch to list" or the table, then notes,
// each line ended by a newline.
func parseMergeOrderText(t *testing.T, out string) mergeOrderText {
	t.Helper()
	body, ok := strings.CutSuffix(out, "\n")
	if !ok {
		t.Fatalf("output does not end with a newline: %q", out)
	}
	lines := strings.Split(body, "\n")
	var r mergeOrderText
	i := 0
	if strings.HasPrefix(lines[i], "origin/") {
		r.header = lines[i]
		i++
	}
	if i >= len(lines) || !strings.HasPrefix(lines[i], "merge guarantee: ") {
		t.Fatalf("no merge guarantee line where it belongs (line %d, after the header or first without one)\n%s", i+1, out)
	}
	r.guarantee = strings.TrimPrefix(lines[i], "merge guarantee: ")
	i++
	if i < len(lines) && lines[i] == "no branch to list" {
		r.empty = true
		i++
	}
	if i < len(lines) && lines[i] == mergeOrderTableHeader {
		for i++; i < len(lines) && !strings.HasPrefix(lines[i], "note: "); i++ {
			fields := strings.Split(lines[i], "\t")
			if len(fields) != 8 {
				t.Fatalf("table row %q has %d tab-separated fields; want 8\n%s", lines[i], len(fields), out)
			}
			r.rows = append(r.rows, fields)
		}
	}
	for ; i < len(lines); i++ {
		note, isNote := strings.CutPrefix(lines[i], "note: ")
		if !isNote {
			t.Fatalf("line %q is neither the header, a table line, nor a note\n%s", lines[i], out)
		}
		r.notes = append(r.notes, note)
	}
	return r
}

// mergeOrderHeaderPattern is the shape of the first line: origin's default
// branch, its tip's first 12 characters, and the time of the ls-remote.
var mergeOrderHeaderPattern = regexp.MustCompile(`^origin/(\S+) ([0-9a-f]{12}) \(listed (\S+)\)$`)

// checkMergeOrderHeader checks that header names origin/<branch> at tip,
// listed at an RFC3339 time no earlier than before, and returns that time.
func checkMergeOrderHeader(t *testing.T, header, branch, tip string, before time.Time) string {
	t.Helper()
	m := mergeOrderHeaderPattern.FindStringSubmatch(header)
	if m == nil || m[1] != branch || m[2] != tip[:12] {
		t.Fatalf("header = %q; want origin/%s %s (listed <time>)", header, branch, tip[:12])
	}
	listed, err := time.Parse(time.RFC3339, m[3])
	if err != nil || listed.Before(before.Truncate(time.Second)) || listed.After(time.Now()) {
		t.Fatalf("header's listed time %q is not an RFC3339 time of this run (from %s): %v", m[3], before.UTC().Format(time.RFC3339), err)
	}
	return m[3]
}

// mergeOrderRow is a table row as expected: every field but the detail
// exactly, the detail by a phrase it must contain.
type mergeOrderRow struct {
	class, branch, pr, gateHead, gateBase, org, detail string
}

// checkMergeOrderRows checks rows against want, in order, ranked from 1.
func checkMergeOrderRows(t *testing.T, rows [][]string, want []mergeOrderRow) {
	t.Helper()
	if len(rows) != len(want) {
		t.Fatalf("table has %d rows; want %d:\n%q", len(rows), len(want), rows)
	}
	for i, w := range want {
		got := rows[i]
		exact := []string{strconv.Itoa(i + 1), w.class, w.branch, w.pr, w.gateHead, w.gateBase, w.org}
		if !slices.Equal(got[:7], exact) || !strings.Contains(got[7], w.detail) {
			t.Errorf("row %d = %q\nwant %q with a detail containing %q", i+1, got, exact, w.detail)
		}
	}
}

// short is sha's first 12 characters, as merge-order shows it.
func short(sha string) string { return sha[:12] }

// mixedMergeOrder holds the commits of seedMixedMergeOrder.
type mixedMergeOrder struct {
	oldMain, main             string // origin/main before and after it moved on
	ready1, ready2            string // gated on main, origin has them
	rebase                    string // gated on oldMain, origin has it
	regateGated, regatePushed string // the gate's head, and what was pushed after
	failed, noRemote, noGate  string
}

// mixedMergeOrderBranches are the branches of seedMixedMergeOrder in the
// order merge-order lists them, with their classes.
var mixedMergeOrderBranches = []struct{ branch, class string }{
	{"feat/ready-1", org.MergeClassReady}, {"feat/ready-2", org.MergeClassReady}, {"feat/rebase", org.MergeClassRebase},
	{"feat/regate", org.MergeClassRegate}, {"feat/failed", org.MergeClassFailed}, {"feat/no-remote", org.MergeClassNoRemote},
	{"feat/no-gate", org.MergeClassNoGate},
}

// seedMixedMergeOrder gives fx one branch of each class gh is not needed
// for, each bound to its own org, recorded in an order other than the one
// merge-order lists them in: feat/rebase gated on main, which then moves on;
// feat/no-gate bound and pushed with no gate; feat/failed with a failed
// gate; feat/regate gated and then pushed again; feat/no-remote gated and
// never pushed; and feat/ready-1 and feat/ready-2 gated, in this order, on
// the new main.
func seedMixedMergeOrder(t *testing.T, fx *mergeOrderFixture) mixedMergeOrder {
	t.Helper()
	var m mixedMergeOrder
	m.oldMain = fx.tip(t, "main")
	m.rebase = fx.branch(t, "feat/rebase")
	fx.push(t, "feat/rebase")
	fx.bind(t, "r", "rebase", "feat/rebase")
	fx.gate(t, "r", "feat/rebase", "", m.rebase, m.oldMain)
	m.main = fx.commitOn(t, "main", "refs/heads/main", "main moves on")
	fx.push(t, "main")

	m.noGate = fx.branch(t, "feat/no-gate")
	fx.push(t, "feat/no-gate")
	fx.bind(t, "z", "no-gate", "feat/no-gate")

	m.failed = fx.branch(t, "feat/failed")
	fx.push(t, "feat/failed")
	fx.bind(t, "f", "failed", "feat/failed")
	fx.gate(t, "f", "feat/failed", org.GateReasonTestFailed, m.failed, m.main)

	m.regateGated = fx.branch(t, "feat/regate")
	fx.bind(t, "g", "regate", "feat/regate")
	fx.gate(t, "g", "feat/regate", "", m.regateGated, m.main)
	m.regatePushed = fx.commitOn(t, "feat/regate", "refs/heads/feat/regate", "pushed after the gate")
	fx.push(t, "feat/regate")

	m.noRemote = fx.branch(t, "feat/no-remote")
	fx.bind(t, "n", "no-remote", "feat/no-remote")
	fx.gate(t, "n", "feat/no-remote", "", m.noRemote, m.main)

	for i, ref := range []*string{&m.ready1, &m.ready2} {
		name := fmt.Sprintf("ready-%d", i+1)
		*ref = fx.branch(t, "feat/"+name)
		fx.push(t, "feat/"+name)
		fx.bind(t, "a"+strconv.Itoa(i+1), name, "feat/"+name)
		fx.gate(t, "a"+strconv.Itoa(i+1), "feat/"+name, "", *ref, m.main)
	}
	return m
}

// AC1, AC2 and AC6 through the CLI with no gh: a branch gated on origin's
// tips is ready, one gated before main moved on is rebase, one pushed after
// its gate is regate, a failed gate is failed with its reason, a branch
// origin lacks is no_remote, and a bound branch with no gate is no_gate. The
// text starts with origin/main and its tip, lists them in merge order (ready
// oldest gate first) with "-" for the PR gh could not read, and ends with
// the notes that gh was not found, that merging one of the two ready
// branches makes the other rebase, and that the order is as of the
// ls-remote. It exits 0 with nothing on stderr.
func TestOrgMergeOrder_TextListsEachClassInMergeOrderWithTheNotes(t *testing.T) {
	fx := newMergeOrderFixture(t, false)
	m := seedMixedMergeOrder(t, fx)

	before := time.Now()
	stdout, stderr, err := fx.run(t)
	if err != nil || stderr != "" {
		t.Fatalf("merge-order: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	out := parseMergeOrderText(t, stdout)
	listedAt := checkMergeOrderHeader(t, out.header, "main", m.main, before)
	if want := "unknown (not checked: run ralph doctor)"; out.guarantee != want {
		t.Errorf("merge guarantee = %q with no cache in the ledger; want %q", out.guarantee, want)
	}
	if out.empty {
		t.Errorf("printed no branch to list beside a table:\n%s", stdout)
	}
	checkMergeOrderRows(t, out.rows, []mergeOrderRow{
		{"ready", "feat/ready-1", "-", short(m.ready1), short(m.main), "a1/ready-1", "the gate passed on " + short(m.ready1)},
		{"ready", "feat/ready-2", "-", short(m.ready2), short(m.main), "a2/ready-2", "the gate passed on " + short(m.ready2)},
		{"rebase", "feat/rebase", "-", short(m.rebase), short(m.oldMain), "r/rebase",
			"origin/main moved: gate base " + short(m.oldMain) + ", now " + short(m.main)},
		{"regate", "feat/regate", "-", short(m.regateGated), short(m.main), "g/regate",
			"pushed after the gate: origin has " + short(m.regatePushed) + ", the gate checked " + short(m.regateGated)},
		{"failed", "feat/failed", "-", short(m.failed), short(m.main), "f/failed", "the gate failed (test_failed)"},
		{"no_remote", "feat/no-remote", "-", short(m.noRemote), short(m.main), "n/no-remote", "origin has no branch feat/no-remote"},
		{"no_gate", "feat/no-gate", "-", "-", "-", "z/no-gate", "no gate record"},
	})
	wantNotes := []string{
		"gh was not found, so no PR was read",
		"2 branches are ready, but merging one moves origin/main: the other ready ones then become rebase",
		"the order is as of the ls-remote at " + listedAt + " and is not a guarantee",
	}
	if len(out.notes) != len(wantNotes) {
		t.Fatalf("notes = %q; want %d, containing %q", out.notes, len(wantNotes), wantNotes)
	}
	for i, want := range wantNotes {
		if !strings.Contains(out.notes[i], want) {
			t.Errorf("note %d = %q; want it to contain %q", i+1, out.notes[i], want)
		}
	}
}

// AC8: --json prints the result alone, on stdout, with the keys of
// org.MergeOrderResult, and lists the branches in the same order as the
// text, with each one's class, gate record, origin's tip and binding.
func TestOrgMergeOrder_JSONListsTheSameBranchesInTheSameOrderAsTheText(t *testing.T) {
	fx := newMergeOrderFixture(t, false)
	m := seedMixedMergeOrder(t, fx)

	res, raw := fx.runJSON(t)
	var keys map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"default_branch", "default_head", "listed_at", "gh_available", "merge_guarantee", "entries", "notes"} {
		if _, ok := keys[key]; !ok {
			t.Errorf("--json output has no %q key:\n%s", key, raw)
		}
	}
	if res.DefaultBranch != "main" || res.DefaultHead != m.main || res.GHAvailable {
		t.Errorf("default branch %q at %q, gh_available %v; want main at %s with no gh", res.DefaultBranch, res.DefaultHead, res.GHAvailable, m.main)
	}
	if _, err := time.Parse(time.RFC3339, res.ListedAt); err != nil {
		t.Errorf("listed_at %q is not an RFC3339 time: %v", res.ListedAt, err)
	}
	if len(res.Entries) != len(mixedMergeOrderBranches) {
		t.Fatalf("entries = %+v; want %d", res.Entries, len(mixedMergeOrderBranches))
	}
	for i, want := range mixedMergeOrderBranches {
		if e := res.Entries[i]; e.Rank != i+1 || e.Branch != want.branch || e.Class != want.class || e.PR != nil {
			t.Errorf("entry %d = rank %d %s %s PR %v; want rank %d %s %s and no PR", i, e.Rank, e.Branch, e.Class, e.PR, i+1, want.branch, want.class)
		}
	}
	byBranch := make(map[string]org.MergeOrderEntry)
	for _, e := range res.Entries {
		byBranch[e.Branch] = e
	}
	if g := byBranch["feat/failed"].Gate; g == nil || *g != (org.MergeOrderGate{Event: org.EventGateFailed, Readable: true, Head: m.failed,
		Base: m.main, BaseRef: "refs/remotes/origin/main", Reason: org.GateReasonTestFailed, TS: g.TS}) || g.TS == "" {
		t.Errorf("feat/failed's gate = %+v; want the readable gate_failed test_failed on %s with base %s", g, m.failed, m.main)
	}
	if b := byBranch["feat/ready-1"].Binding; b == nil || b.OrgID != "a1" || b.Feature != "ready-1" || b.Split != "demo" || b.TS == "" {
		t.Errorf("feat/ready-1's binding = %+v; want org a1, feature ready-1 of split demo", b)
	}
	if e := byBranch["feat/no-gate"]; e.Gate != nil || e.RemoteHead != m.noGate {
		t.Errorf("feat/no-gate: gate %+v, remote head %q; want no gate and origin's tip %s", e.Gate, e.RemoteHead, m.noGate)
	}
	if e := byBranch["feat/regate"]; e.RemoteHead != m.regatePushed || e.Gate == nil || e.Gate.Head != m.regateGated {
		t.Errorf("feat/regate: remote head %q, gate %+v; want origin's tip %s and the gate's head %s", e.RemoteHead, e.Gate, m.regatePushed, m.regateGated)
	}
	if e := byBranch["feat/no-remote"]; e.RemoteHead != "" {
		t.Errorf("feat/no-remote's remote head = %q; want none", e.RemoteHead)
	}
	if len(res.Notes) != 3 || !strings.Contains(res.Notes[0], "gh was not found") {
		t.Errorf("notes = %q; want the gh, ready and as-of notes", res.Notes)
	}

	stdout, _, err := fx.run(t)
	if err != nil {
		t.Fatalf("merge-order: %v", err)
	}
	var textBranches, jsonBranches []string
	for _, row := range parseMergeOrderText(t, stdout).rows {
		textBranches = append(textBranches, row[1]+" "+row[2])
	}
	for _, e := range res.Entries {
		jsonBranches = append(jsonBranches, e.Class+" "+e.Branch)
	}
	if !slices.Equal(textBranches, jsonBranches) {
		t.Errorf("text lists %q; --json lists %q", textBranches, jsonBranches)
	}
}

// AC5 through the CLI with the fake gh: a branch whose PR is merged (and
// whose origin branch was deleted after the merge) and one whose PR was
// closed are left out by default, with a note that counts them, and listed
// after the rest as merged and closed with --all. Of a branch's PRs, the
// highest-numbered one from this repository is read: a fork's PR with a
// higher number is not. gh runs once per branch, with GH_PROMPT_DISABLED=1.
func TestOrgMergeOrder_AllListsMergedAndClosedPRsTheDefaultLeavesOut(t *testing.T) {
	fx := newMergeOrderFixture(t, true)
	main := fx.tip(t, "main")
	open := fx.branch(t, "feat/open")
	merged := fx.branch(t, "feat/merged")
	closed := fx.branch(t, "feat/closed")
	fx.push(t, "feat/open", "feat/closed")
	for _, b := range []struct{ org, branch, head string }{{"o", "feat/open", open}, {"m", "feat/merged", merged}, {"c", "feat/closed", closed}} {
		fx.bind(t, b.org, strings.TrimPrefix(b.branch, "feat/"), b.branch)
		fx.gate(t, b.org, b.branch, "", b.head, main)
	}
	fx.prs(t, "feat/open",
		mergeOrderGHPR{Number: 1, State: "CLOSED", URL: "https://example.test/pull/1", HeadRefOid: open, BaseRefName: "main"},
		mergeOrderGHPR{Number: 3, State: "OPEN", URL: "https://example.test/pull/3", HeadRefOid: open, BaseRefName: "main"},
		mergeOrderGHPR{Number: 9, State: "MERGED", URL: "https://example.test/fork/pull/9", HeadRefOid: open, BaseRefName: "main", IsCrossRepository: true})
	fx.prs(t, "feat/merged", mergeOrderGHPR{Number: 4, State: "MERGED", URL: "https://example.test/pull/4", HeadRefOid: merged, BaseRefName: "main"})
	fx.prs(t, "feat/closed", mergeOrderGHPR{Number: 5, State: "CLOSED", URL: "https://example.test/pull/5", HeadRefOid: closed, BaseRefName: "main"})

	before := time.Now()
	stdout, stderr, err := fx.run(t)
	if err != nil || stderr != "" {
		t.Fatalf("merge-order: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	out := parseMergeOrderText(t, stdout)
	checkMergeOrderHeader(t, out.header, "main", main, before)
	checkMergeOrderRows(t, out.rows, []mergeOrderRow{
		{"ready", "feat/open", "#3 https://example.test/pull/3", short(open), short(main), "o/open", "the gate passed"},
	})
	if len(out.notes) != 2 || out.notes[0] != "2 branch(es) whose PR is merged or closed are not shown (--all shows them)" ||
		!strings.HasPrefix(out.notes[1], "the order is as of the ls-remote") {
		t.Errorf("notes = %q; want the count of the 2 left out, then the as-of note", out.notes)
	}
	wantCalls := []string{mergeOrderGHCall("feat/open"), mergeOrderGHCall("feat/merged"), mergeOrderGHCall("feat/closed")}
	if calls := fx.ghCalls(t); !slices.Equal(calls, wantCalls) {
		t.Errorf("gh calls = %q\nwant %q", calls, wantCalls)
	}

	stdout, stderr, err = fx.run(t, "--all")
	if err != nil || stderr != "" {
		t.Fatalf("merge-order --all: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	out = parseMergeOrderText(t, stdout)
	checkMergeOrderRows(t, out.rows, []mergeOrderRow{
		{"ready", "feat/open", "#3 https://example.test/pull/3", short(open), short(main), "o/open", "the gate passed"},
		{"merged", "feat/merged", "#4 https://example.test/pull/4", short(merged), short(main), "m/merged", "PR #4 is merged"},
		{"closed", "feat/closed", "#5 https://example.test/pull/5", short(closed), short(main), "c/closed", "PR #5 was closed without a merge"},
	})
	if len(out.notes) != 1 || !strings.HasPrefix(out.notes[0], "the order is as of the ls-remote") {
		t.Errorf("--all notes = %q; want only the as-of note", out.notes)
	}

	res, _ := fx.runJSON(t, "--all")
	if !res.GHAvailable || len(res.Entries) != 3 || res.Entries[0].PR == nil ||
		*res.Entries[0].PR != (org.MergeOrderPR{Number: 3, State: "OPEN", URL: "https://example.test/pull/3", HeadRefOid: open, BaseRefName: "main"}) {
		t.Errorf("--all --json: gh_available %v, entries %+v; want gh available and feat/open's PR #3 first", res.GHAvailable, res.Entries)
	}
}

// With gh reporting only merged PRs, nothing is left to list: the text says
// so under origin's line, with the note that counts what was left out.
func TestOrgMergeOrder_OnlyMergedPRsPrintsNoBranchToList(t *testing.T) {
	fx := newMergeOrderFixture(t, true)
	main := fx.tip(t, "main")
	head := fx.branch(t, "feat/done")
	fx.bind(t, "d", "done", "feat/done")
	fx.gate(t, "d", "feat/done", "", head, main)
	fx.prs(t, "feat/done", mergeOrderGHPR{Number: 2, State: "MERGED", URL: "https://example.test/pull/2", HeadRefOid: head, BaseRefName: "main"})

	before := time.Now()
	stdout, stderr, err := fx.run(t)
	if err != nil || stderr != "" {
		t.Fatalf("merge-order: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	out := parseMergeOrderText(t, stdout)
	checkMergeOrderHeader(t, out.header, "main", main, before)
	if !out.empty || len(out.rows) != 0 {
		t.Errorf("want the line no branch to list and no table:\n%s", stdout)
	}
	if len(out.notes) != 2 || out.notes[0] != "1 branch(es) whose PR is merged or closed are not shown (--all shows them)" {
		t.Errorf("notes = %q; want the count of the one left out first", out.notes)
	}
}

// AC11, second half: a branch whose gate passed on origin's tips but whose PR
// gh reports as merging into develop, not origin's default branch main, is
// wrong_base, with the PR in its row.
func TestOrgMergeOrder_PRIntoAnotherBranchThanTheDefaultIsWrongBase(t *testing.T) {
	fx := newMergeOrderFixture(t, true)
	main := fx.tip(t, "main")
	head := fx.branch(t, "feat/wb")
	fx.push(t, "feat/wb")
	fx.bind(t, "w", "wb", "feat/wb")
	fx.gate(t, "w", "feat/wb", "", head, main)
	fx.prs(t, "feat/wb", mergeOrderGHPR{Number: 7, State: "OPEN", URL: "https://example.test/pull/7", HeadRefOid: head, BaseRefName: "develop"})

	stdout, stderr, err := fx.run(t)
	if err != nil || stderr != "" {
		t.Fatalf("merge-order: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	checkMergeOrderRows(t, parseMergeOrderText(t, stdout).rows, []mergeOrderRow{
		{"wrong_base", "feat/wb", "#7 https://example.test/pull/7", short(head), short(main), "w/wb",
			"PR #7 merges into develop, not origin's default branch main"},
	})
}

// AC11: once origin's HEAD names trunk instead of main (trunk pushed at
// main's tip, main left in place), a branch gated against
// refs/remotes/origin/main is rebase, although main's tip, and trunk's, are
// still the base it was gated on. Before the switch, the same branch is
// ready.
func TestOrgMergeOrder_GateOnTheOldDefaultBranchIsRebaseAfterOriginHEADMoves(t *testing.T) {
	fx := newMergeOrderFixture(t, false)
	main := fx.tip(t, "main")
	head := fx.branch(t, "feat/a")
	fx.push(t, "feat/a")
	fx.bind(t, "a", "a", "feat/a")
	fx.gate(t, "a", "feat/a", "", head, main)

	stdout, _, err := fx.run(t)
	if err != nil {
		t.Fatalf("merge-order before the switch: %v", err)
	}
	if rows := parseMergeOrderText(t, stdout).rows; len(rows) != 1 || rows[0][1] != org.MergeClassReady {
		t.Fatalf("before origin's HEAD moves, feat/a should be ready:\n%s", stdout)
	}

	fx.push(t, "main:trunk")
	runLedgerTestGit(t, "-C", fx.bare, "symbolic-ref", "HEAD", "refs/heads/trunk")
	before := time.Now()
	stdout, stderr, err := fx.run(t)
	if err != nil || stderr != "" {
		t.Fatalf("merge-order: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	out := parseMergeOrderText(t, stdout)
	checkMergeOrderHeader(t, out.header, "trunk", main, before)
	checkMergeOrderRows(t, out.rows, []mergeOrderRow{
		{"rebase", "feat/a", "-", short(head), short(main), "a/a",
			"the gate checked against refs/remotes/origin/main, but origin's default branch is now trunk"},
	})
}

// A branch name with two slashes is read whole from the ls-remote and passed
// whole to gh: feat/x/y is ready, with its PR.
func TestOrgMergeOrder_BranchNameWithTwoSlashesIsListedWhole(t *testing.T) {
	fx := newMergeOrderFixture(t, true)
	main := fx.tip(t, "main")
	head := fx.branch(t, "feat/x/y")
	fx.push(t, "feat/x/y")
	fx.bind(t, "x", "x-y", "feat/x/y")
	fx.gate(t, "x", "feat/x/y", "", head, main)
	fx.prs(t, "feat/x/y", mergeOrderGHPR{Number: 2, State: "OPEN", URL: "https://example.test/pull/2", HeadRefOid: head, BaseRefName: "main"})

	stdout, stderr, err := fx.run(t)
	if err != nil || stderr != "" {
		t.Fatalf("merge-order: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	checkMergeOrderRows(t, parseMergeOrderText(t, stdout).rows, []mergeOrderRow{
		{"ready", "feat/x/y", "#2 https://example.test/pull/2", short(head), short(main), "x/x-y", "the gate passed"},
	})
	if calls := fx.ghCalls(t); !slices.Equal(calls, []string{mergeOrderGHCall("feat/x/y")}) {
		t.Errorf("gh calls = %q; want one for feat/x/y", calls)
	}
	if res, _ := fx.runJSON(t); len(res.Entries) != 1 || res.Entries[0].Branch != "feat/x/y" || res.Entries[0].RemoteHead != head {
		t.Errorf("--json entries = %+v; want feat/x/y at %s", res.Entries, head)
	}
}

// A value from the ledger or gh cannot break the table: a control character
// in a bound branch's name and a tab and a newline in a PR's URL are printed
// escaped, so every row keeps its 8 fields and no forged note line appears.
func TestOrgMergeOrder_ControlCharactersInLedgerAndGHValuesAreEscaped(t *testing.T) {
	fx := newMergeOrderFixture(t, true)
	main := fx.tip(t, "main")
	head := fx.branch(t, "feat/url")
	fx.push(t, "feat/url")
	fx.bind(t, "u", "url", "feat/url")
	fx.gate(t, "u", "feat/url", "", head, main)
	fx.bind(t, "e", "esc", "feat/esc\x1b[31m")
	fx.prs(t, "feat/url", mergeOrderGHPR{Number: 1, State: "OPEN", URL: "https://example.test/pull/1\tINJECT\nnote: forged",
		HeadRefOid: head, BaseRefName: "main"})

	stdout, stderr, err := fx.run(t)
	if err != nil || stderr != "" {
		t.Fatalf("merge-order: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	out := parseMergeOrderText(t, stdout)
	checkMergeOrderRows(t, out.rows, []mergeOrderRow{
		{"ready", "feat/url", `#1 https://example.test/pull/1\tINJECT\nnote: forged`, short(head), short(main), "u/url", "the gate passed"},
		{"no_gate", `feat/esc\x1b[31m`, "-", "-", "-", "e/esc", "no gate record"},
	})
	for _, note := range out.notes {
		if strings.HasPrefix(note, "forged") {
			t.Errorf("a PR's URL forged a note line:\n%s", stdout)
		}
	}
	if strings.Contains(stdout, "\x1b") {
		t.Errorf("stdout holds a raw ESC:\n%q", stdout)
	}
}

// A gh whose stderr has two lines cannot forge a note line: the failure note
// is printed on one line, its newline escaped, before the as-of note.
func TestOrgMergeOrder_MultiLineGHStderrStaysOneNoteLine(t *testing.T) {
	fx := newMergeOrderFixture(t, false)
	writeStubScript(t, filepath.Join(fx.bin, "gh"), "#!/bin/sh\nprintf 'HTTP 401: Bad credentials\\nnote: forged\\n' >&2\nexit 1\n")
	main := fx.tip(t, "main")
	head := fx.branch(t, "feat/a")
	fx.push(t, "feat/a")
	fx.bind(t, "a", "a", "feat/a")
	fx.gate(t, "a", "feat/a", "", head, main)

	stdout, stderr, err := fx.run(t)
	if err != nil || stderr != "" {
		t.Fatalf("merge-order: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	out := parseMergeOrderText(t, stdout)
	if len(out.notes) != 2 || !strings.HasPrefix(out.notes[0], "gh pr list --head feat/a failed (") ||
		!strings.Contains(out.notes[0], `Bad credentials\nnote: forged`) || !strings.HasPrefix(out.notes[1], "the order is as of the ls-remote") {
		t.Fatalf("notes = %q; want the gh failure on one escaped line, then the as-of note", out.notes)
	}
}

// The cross-review finding through the CLI, with the real git and the fake
// gh: origin's URL names acme/app (the git wrapper answers git remote
// get-url origin), and gh returns a merged PR of upstream-org/app for
// feat/a, as gh does in a fork's checkout whose upstream remote it picks.
// The PR is not used: feat/a stays ready with no PR instead of being hidden
// as merged, gh is not asked about feat/b, and the first note names both
// fixes: git remote set-url origin when acme/app was renamed, else gh repo
// set-default acme/app. With the PR's url naming acme/app, the same run hides
// feat/a and shows feat/b's PR.
func TestOrgMergeOrder_PRFromAnotherRepositoryThanOriginsIsLeftOut(t *testing.T) {
	fx := newMergeOrderFixture(t, true)
	main := fx.tip(t, "main")
	a, b := fx.branch(t, "feat/a"), fx.branch(t, "feat/b")
	fx.push(t, "feat/a", "feat/b")
	fx.bind(t, "a", "a", "feat/a")
	fx.gate(t, "a", "feat/a", "", a, main)
	fx.bind(t, "b", "b", "feat/b")
	fx.gate(t, "b", "feat/b", "", b, main)
	t.Setenv("MERGE_ORDER_ORIGIN_URL", "git@github.com:acme/app.git")
	fx.prs(t, "feat/a", mergeOrderGHPR{Number: 7, State: "MERGED", URL: "https://github.com/upstream-org/app/pull/7", HeadRefOid: a, BaseRefName: "main"})
	fx.prs(t, "feat/b", mergeOrderGHPR{Number: 8, State: "OPEN", URL: "https://github.com/acme/app/pull/8", HeadRefOid: b, BaseRefName: "main"})

	stdout, stderr, err := fx.run(t)
	if err != nil || stderr != "" {
		t.Fatalf("merge-order: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	out := parseMergeOrderText(t, stdout)
	checkMergeOrderRows(t, out.rows, []mergeOrderRow{
		{"ready", "feat/a", "-", short(a), short(main), "a/a", "the gate passed"},
		{"ready", "feat/b", "-", short(b), short(main), "b/b", "the gate passed"},
	})
	if len(out.notes) != 3 || !strings.HasPrefix(out.notes[0], "gh reads upstream-org/app, not origin's acme/app: no PR is used") ||
		!strings.Contains(out.notes[0], "if acme/app was renamed to upstream-org/app, point origin at it (git remote set-url origin") ||
		!strings.Contains(out.notes[0], "otherwise run gh repo set-default acme/app in this repository") {
		t.Errorf("notes = %q; want the repository note first", out.notes)
	}
	if calls := fx.ghCalls(t); !slices.Equal(calls, []string{mergeOrderGHCall("feat/a")}) {
		t.Errorf("gh calls = %q; want feat/a only", calls)
	}

	fx.prs(t, "feat/a", mergeOrderGHPR{Number: 7, State: "MERGED", URL: "https://github.com/Acme/App/pull/7", HeadRefOid: a, BaseRefName: "main"})
	stdout, stderr, err = fx.run(t)
	if err != nil || stderr != "" {
		t.Fatalf("merge-order with origin's PRs: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	out = parseMergeOrderText(t, stdout)
	checkMergeOrderRows(t, out.rows, []mergeOrderRow{
		{"ready", "feat/b", "#8 https://github.com/acme/app/pull/8", short(b), short(main), "b/b", "the gate passed"},
	})
	if len(out.notes) != 2 || !strings.HasPrefix(out.notes[0], "1 branch(es) whose PR is merged or closed are not shown") {
		t.Errorf("notes with origin's PRs = %q; want feat/a counted as left out, then the as-of note", out.notes)
	}
}

// AC7: when the ls-remote cannot tell origin's state, merge-order classifies
// nothing, prints nothing on stdout, runs no gh, and exits 1 with the reason:
// with no origin remote, git's own message; with origin's HEAD naming a
// branch origin does not have, no HEAD symref.
func TestOrgMergeOrder_OriginStateUnknownExitsOneWithTheReason(t *testing.T) {
	for _, tc := range []struct {
		name        string
		breakOrigin func(t *testing.T, fx *mergeOrderFixture)
		want        []string
	}{
		{"no origin remote", func(t *testing.T, fx *mergeOrderFixture) {
			runLedgerTestGit(t, "-C", fx.work, "remote", "remove", "origin")
		}, []string{"org: merge-order: git ls-remote --symref origin HEAD 'refs/heads/*' in . failed (",
			"'origin' does not appear to be a git repository", "origin's state is unknown, so nothing was classified"}},
		{"origin HEAD names no listed branch", func(t *testing.T, fx *mergeOrderFixture) {
			runLedgerTestGit(t, "-C", fx.bare, "symbolic-ref", "HEAD", "refs/heads/nowhere")
		}, []string{"printed no HEAD symref, so origin's default branch is unknown and nothing was classified"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMergeOrderFixture(t, true)
			head := fx.branch(t, "feat/a")
			fx.push(t, "feat/a")
			fx.bind(t, "a", "a", "feat/a")
			fx.gate(t, "a", "feat/a", "", head, fx.tip(t, "main"))
			tc.breakOrigin(t, fx)

			for _, extra := range [][]string{nil, {"--json"}} {
				stdout, _, err := fx.run(t, extra...)
				if err == nil {
					t.Fatalf("merge-order %v exited 0; want exit 1 (stdout: %s)", extra, stdout)
				}
				if stdout != "" {
					t.Errorf("merge-order %v printed on stdout: %q; want nothing", extra, stdout)
				}
				for _, want := range tc.want {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("merge-order %v error = %q; want it to contain %q", extra, err, want)
					}
				}
			}
			if calls := fx.ghCalls(t); len(calls) != 0 {
				t.Errorf("gh ran: %q; want no gh once origin's state is unknown", calls)
			}
		})
	}
}

// seedReadyMergeOrderBranch gives fx one branch, feat/a, gated on origin's
// main and pushed, so merge-order runs the ls-remote.
func seedReadyMergeOrderBranch(t *testing.T, fx *mergeOrderFixture) {
	t.Helper()
	main := fx.tip(t, "main")
	head := fx.branch(t, "feat/a")
	fx.push(t, "feat/a")
	fx.bind(t, "a", "a", "feat/a")
	fx.gate(t, "a", "feat/a", "", head, main)
}

// AC7 and AC10 through the CLI: with a fresh cache in the ledger, the merge
// guarantee line follows origin/main's line. A cache of origin's repository
// (as the git wrapper answers git remote get-url origin; compared without
// case) and of its default branch gives the kind with the branch, the
// repository and checked_at; one of another default branch than the
// ls-remote names is unknown naming both branches; and once origin points to
// another repository with the same default branch it is unknown naming both
// repositories. --json carries the same as merge_guarantee.
func TestOrgMergeOrder_MergeGuaranteeLineFollowsTheDefaultBranchLine(t *testing.T) {
	fresh := freshMergeGuaranteeCache("main")
	trunk := freshMergeGuaranteeCache("trunk")
	const changedOrigin = "doctor checked github.com/acme/app; origin is now github.com/acme/other"
	const changedBranch = "doctor checked trunk; origin's default branch is now main"
	for _, tc := range []struct {
		name     string
		cache    org.MergeGuaranteeCheck
		origin   string
		wantText string
		wantJSON org.MergeGuarantee
	}{
		{"origin's repository and default branch", fresh, "https://github.com/acme/app.git",
			cachedMergeGuaranteeText(fresh), cachedMergeGuarantee(fresh)},
		{"origin's repository in another case", fresh, "git@github.com:ACME/App.git",
			cachedMergeGuaranteeText(fresh), cachedMergeGuarantee(fresh)},
		{"another default branch", trunk, "https://github.com/acme/app.git",
			"unknown (" + changedBranch + ")", unknownFromCache(trunk, org.MergeGuaranteeReasonDefaultBranchChanged, changedBranch)},
		{"origin pointed to another repository", fresh, "https://github.com/acme/other.git",
			"unknown (" + changedOrigin + ")", unknownFromCache(fresh, org.MergeGuaranteeReasonOriginChanged, changedOrigin)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fx := newMergeOrderFixture(t, false)
			seedReadyMergeOrderBranch(t, fx)
			writeMergeGuaranteeCache(t, fx.stateDir, tc.cache)
			t.Setenv("MERGE_ORDER_ORIGIN_URL", tc.origin)

			before := time.Now()
			stdout, stderr, err := fx.run(t)
			if err != nil || stderr != "" {
				t.Fatalf("merge-order: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
			}
			out := parseMergeOrderText(t, stdout)
			checkMergeOrderHeader(t, out.header, "main", fx.tip(t, "main"), before)
			if lines := strings.Split(stdout, "\n"); lines[1] != "merge guarantee: "+tc.wantText {
				t.Errorf("line 2 = %q; want %q right after origin/main's line", lines[1], "merge guarantee: "+tc.wantText)
			}
			if len(out.rows) != 1 || out.rows[0][1] != org.MergeClassReady {
				t.Errorf("rows = %q; want feat/a ready", out.rows)
			}
			res, _ := fx.runJSON(t)
			if res.MergeGuarantee != tc.wantJSON {
				t.Errorf("--json merge_guarantee:\n got %+v\nwant %+v", res.MergeGuarantee, tc.wantJSON)
			}
		})
	}
}

// AC7: with no branch in the ledger no ls-remote runs, so there is no
// origin/<branch> line and the guarantee line comes first, with the cache's
// kind (the local get-url still runs for a fresh cache).
func TestOrgMergeOrder_NoBranchPrintsTheGuaranteeLineFirst(t *testing.T) {
	fx := newMergeOrderFixture(t, false)
	c := freshMergeGuaranteeCache("main")
	writeMergeGuaranteeCache(t, fx.stateDir, c)
	t.Setenv("MERGE_ORDER_ORIGIN_URL", "https://github.com/acme/app.git")

	stdout, stderr, err := fx.run(t)
	want := "merge guarantee: " + cachedMergeGuaranteeText(c) + "\n" +
		"note: no feature branches in the ledger: a branch is listed once ralph org start --plan binds an org to it " +
		"or ralph org gate records a gate on it\n"
	if err != nil || stderr != "" || stdout != want {
		t.Fatalf("merge-order with no branch: err %v, stderr %q\nstdout %q\nwant   %q", err, stderr, stdout, want)
	}
	if res, _ := fx.runJSON(t); res.MergeGuarantee != cachedMergeGuarantee(c) || res.DefaultBranch != "" {
		t.Errorf("--json with no branch: merge_guarantee %+v, default branch %q; want the cache's kind and none", res.MergeGuarantee, res.DefaultBranch)
	}
}

// A cache whose default branch holds a tab and a newline cannot break the
// guarantee line or forge a note: the default_branch_changed detail that
// names it prints escaped, on one line.
func TestOrgMergeOrder_MergeGuaranteeWithControlCharactersStaysOneLine(t *testing.T) {
	fx := newMergeOrderFixture(t, false)
	seedReadyMergeOrderBranch(t, fx)
	writeMergeGuaranteeCache(t, fx.stateDir, freshMergeGuaranteeCache("main\tx\nnote: forged"))
	t.Setenv("MERGE_ORDER_ORIGIN_URL", "https://github.com/acme/app.git")

	stdout, stderr, err := fx.run(t)
	if err != nil || stderr != "" {
		t.Fatalf("merge-order: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	out := parseMergeOrderText(t, stdout)
	if want := `unknown (doctor checked main\tx\nnote: forged; origin's default branch is now main)`; out.guarantee != want {
		t.Errorf("merge guarantee = %q\nwant %q", out.guarantee, want)
	}
	for _, note := range out.notes {
		if strings.HasPrefix(note, "forged") {
			t.Errorf("the cache's branch forged a note line:\n%s", stdout)
		}
	}
}

// writeMergeOrderCallLoggers sets PATH to a fresh bin dir whose git and gh
// only log their call to the returned file and fail, so a test can check
// that neither runs.
func writeMergeOrderCallLoggers(t *testing.T) (calls string) {
	t.Helper()
	tmp := mergeOrderTempDir(t)
	bin := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"git", "gh"} {
		writeStubScript(t, filepath.Join(bin, name), fmt.Sprintf(mergeOrderCallLogger, name))
	}
	calls = filepath.Join(tmp, "calls")
	t.Setenv("MERGE_ORDER_CALLS", calls)
	t.Setenv("PATH", bin)
	return calls
}

// With no ledger at all, or an empty manifest, there is no branch to look at:
// merge-order prints only the merge guarantee line, first (no ledger yet, or
// not checked), and the note that the ledger has no feature branch, exits 0,
// runs neither git nor gh (the current directory is not a repository, and
// with no cache the guarantee's read runs no git), and creates no state dir.
// --json prints the same, with an empty entries list and the guarantee.
func TestOrgMergeOrder_NoLedgerPrintsTheNoBranchesNoteWithoutRunningGit(t *testing.T) {
	isolateFeatureGitEnv(t)
	calls := writeMergeOrderCallLoggers(t)
	tmp := mergeOrderTempDir(t)
	t.Chdir(tmp)
	const note = "note: no feature branches in the ledger: a branch is listed once ralph org start --plan binds an org to it " +
		"or ralph org gate records a gate on it\n"

	stateDir := filepath.Join(tmp, "state")
	stdout, stderr, err := runOrgCmdStreams(t, "merge-order", "--state-dir", stateDir)
	if want := "merge guarantee: unknown (no org ledger yet)\n" + note; err != nil || stderr != "" || stdout != want {
		t.Fatalf("merge-order with no ledger: err %v, stderr %q, stdout %q; want exit 0 and only %q", err, stderr, stdout, want)
	}
	stdout, _, err = runOrgCmdStreams(t, "merge-order", "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("merge-order --json with no ledger: %v", err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &keys); err != nil || string(keys["entries"]) != "[]" || string(keys["default_branch"]) != `""` {
		t.Errorf("--json with no ledger = %s; want empty entries and no default branch (%v)", stdout, err)
	}
	var g org.MergeGuarantee
	if err := json.Unmarshal(keys["merge_guarantee"], &g); err != nil ||
		g != (org.MergeGuarantee{Kind: org.MergeGuaranteeUnknown, Reason: org.MergeGuaranteeReasonNoLedger, Detail: "no org ledger yet"}) {
		t.Errorf("--json merge_guarantee with no ledger = %s; want unknown (no_ledger) (%v)", keys["merge_guarantee"], err)
	}
	if _, err := os.Stat(stateDir); !os.IsNotExist(err) {
		t.Errorf("merge-order created the state dir %s (stat: %v)", stateDir, err)
	}

	emptyDir := filepath.Join(tmp, "empty")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, org.ManifestPathIn(emptyDir), "")
	stdout, _, err = runOrgCmdStreams(t, "merge-order", "--state-dir", emptyDir)
	if want := "merge guarantee: unknown (not checked: run ralph doctor)\n" + note; err != nil || stdout != want {
		t.Errorf("merge-order with an empty manifest: err %v, stdout %q; want only %q", err, stdout, want)
	}
	if got := readLogLines(t, calls); len(got) != 0 {
		t.Errorf("git or gh ran: %q", got)
	}
}

// --org-id is refused before anything is read: with --state-dir naming a
// ledger whose manifest cannot be read, the error is the refusal, not the
// read; without --state-dir, no git runs to resolve the state dir.
func TestOrgMergeOrder_OrgIDIsRefusedBeforeAnythingIsRead(t *testing.T) {
	isolateFeatureGitEnv(t)
	calls := writeMergeOrderCallLoggers(t)
	tmp := mergeOrderTempDir(t)
	t.Chdir(tmp)
	stateDir := filepath.Join(tmp, "state")
	if err := os.MkdirAll(org.ManifestPathIn(stateDir), 0o755); err != nil {
		t.Fatal(err)
	}
	const want = "org: merge-order: --org-id does not apply: merge-order lists the feature branches of every org in the ledger"

	for _, args := range [][]string{{"merge-order", "--org-id", "a", "--state-dir", stateDir}, {"merge-order", "--org-id", "a"}} {
		stdout, _, err := runOrgCmdStreams(t, args...)
		if err == nil || err.Error() != want || stdout != "" {
			t.Errorf("%v: err %v, stdout %q; want the error %q and nothing on stdout", args, err, stdout, want)
		}
	}
	if got := readLogLines(t, calls); len(got) != 0 {
		t.Errorf("git or gh ran before --org-id was refused: %q", got)
	}
	if _, _, err := runOrgCmdStreams(t, "merge-order", "--state-dir", stateDir); err == nil ||
		!strings.Contains(err.Error(), "org: merge-order: read manifest") {
		t.Errorf("control: without --org-id, merge-order should fail to read that manifest; got %v", err)
	}
}

// --state-dir picks the ledger over RALPH_ORG_STATE_DIR, which picks it when
// --state-dir is not given.
func TestOrgMergeOrder_StateDirFlagPicksTheLedger(t *testing.T) {
	fx := newMergeOrderFixture(t, false)
	fx.bind(t, "f", "flag", "feat/flag")
	envDir := filepath.Join(filepath.Dir(fx.stateDir), "env-state")
	appendSeedEvent(t, envDir, org.ManifestEvent{
		OrgID: "e", Event: org.EventScopeReserved, Worktree: fx.work,
		Details: "paths=internal/e/ split=demo feature=env digest=0123456789ab branch=feat/env approval=per-plan",
	})
	t.Setenv(org.EnvOrgStateDir, envDir)

	branches := func(args ...string) []string {
		t.Helper()
		stdout, _, err := runOrgCmdStreams(t, append([]string{"merge-order"}, args...)...)
		if err != nil {
			t.Fatalf("merge-order %v: %v", args, err)
		}
		var got []string
		for _, row := range parseMergeOrderText(t, stdout).rows {
			got = append(got, row[2])
		}
		return got
	}
	if got := branches("--state-dir", fx.stateDir); !slices.Equal(got, []string{"feat/flag"}) {
		t.Errorf("with --state-dir, branches = %q; want only feat/flag from that ledger", got)
	}
	if got := branches(); !slices.Equal(got, []string{"feat/env"}) {
		t.Errorf("without --state-dir, branches = %q; want only feat/env from RALPH_ORG_STATE_DIR's ledger", got)
	}
}

// mergeOrderDirListing lists every path under dir with its mode, size and
// modification time.
func mergeOrderDirListing(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		out = append(out, fmt.Sprintf("%s %s %d %s", rel, info.Mode(), info.Size(), info.ModTime().UTC().Format(time.RFC3339Nano)))
		return nil
	})
	if err != nil {
		t.Fatalf("list %s: %v", dir, err)
	}
	return out
}

// holdMergeOrderLock takes an exclusive flock on path (creating the file)
// until the test ends, as another ralph process holding that lock would.
func holdMergeOrderLock(t *testing.T, path string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		t.Fatalf("flock %s: %v", path, err)
	}
	t.Cleanup(func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	})
}

// AC9: merge-order writes nothing and takes no lock. With the manifest lock
// and the inbox lock (lockfile.go's manifest.lock, inbox.go's inbox.lock)
// held by someone else, and the manifest and the state dir read-only, text
// and --json both succeed; afterwards the manifest's bytes, the state dir's
// listing, and the work repository's refs (no fetch) are as before. No watch
// runs on the ledger. The ledger's merge-guarantee cache is read (its kind is
// shown) and left as it was too.
func TestOrgMergeOrder_WritesNothingAndTakesNoLock(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores read-only permission bits")
	}
	fx := newMergeOrderFixture(t, true)
	m := seedMixedMergeOrder(t, fx)
	cache := freshMergeGuaranteeCache("main")
	writeMergeGuaranteeCache(t, fx.stateDir, cache)
	t.Setenv("MERGE_ORDER_ORIGIN_URL", "https://github.com/acme/app.git")
	holdMergeOrderLock(t, filepath.Join(fx.stateDir, "manifest.lock"))
	holdMergeOrderLock(t, filepath.Join(fx.stateDir, "inbox.lock"))

	manifestPath := org.ManifestPathIn(fx.stateDir)
	manifestBefore := fileBytesOrAbsent(t, manifestPath)
	listingBefore := mergeOrderDirListing(t, fx.stateDir)
	refsBefore := gateCLIGitOut(t, fx.work, "for-each-ref")
	dirInfo, err := os.Stat(fx.stateDir)
	if err != nil {
		t.Fatal(err)
	}
	fileInfo, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	restore := func() {
		_ = os.Chmod(fx.stateDir, dirInfo.Mode().Perm())
		_ = os.Chmod(manifestPath, fileInfo.Mode().Perm())
	}
	t.Cleanup(restore)
	if err := os.Chmod(manifestPath, 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(fx.stateDir, 0o555); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := fx.run(t)
	if err != nil || stderr != "" {
		t.Fatalf("merge-order on a read-only, locked ledger: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	out := parseMergeOrderText(t, stdout)
	if rows := out.rows; len(rows) != len(mixedMergeOrderBranches) || rows[0][2] != "feat/ready-1" {
		t.Errorf("merge-order on a read-only, locked ledger listed %q; want the seeded branches", rows)
	}
	if want := cachedMergeGuaranteeText(cache); out.guarantee != want {
		t.Errorf("merge guarantee on a read-only, locked ledger = %q; want the cache's %q", out.guarantee, want)
	}
	if res, _ := fx.runJSON(t, "--all"); res.DefaultHead != m.main {
		t.Errorf("--json default head = %q; want %s", res.DefaultHead, m.main)
	}
	restore()

	if got := fileBytesOrAbsent(t, manifestPath); got != manifestBefore {
		t.Errorf("the manifest changed:\nbefore %q\nafter  %q", manifestBefore, got)
	}
	if got := mergeOrderDirListing(t, fx.stateDir); !slices.Equal(got, listingBefore) {
		t.Errorf("the state dir changed:\nbefore %q\nafter  %q", listingBefore, got)
	}
	if got := gateCLIGitOut(t, fx.work, "for-each-ref"); got != refsBefore {
		t.Errorf("the work repository's refs changed (a fetch?):\nbefore %s\nafter  %s", refsBefore, got)
	}
}
