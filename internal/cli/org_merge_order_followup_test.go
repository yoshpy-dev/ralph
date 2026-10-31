package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// The follow-up section of `ralph org merge-order` through the CLI (issue
// #270; plan docs/plans/active/2026-10-30-org-followups.md, AC3 and AC7), on
// the fixture of org_merge_order_test.go: the real git against a bare origin
// and the fake gh, whose answer for a --head branch serves both the entry's
// lookup and the follow-up's. The states themselves are covered by
// internal/org/mergeorder_followup_test.go.

// mergeOrderFollowupAddedAt is when the tests' follow-ups are recorded.
var mergeOrderFollowupAddedAt = time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)

// mergeOrderFollowupHeader is the header line of the follow-up table.
const mergeOrderFollowupHeader = "ID\tISSUE\tAFTER_BRANCH\tSTATE\tPR\tDETAIL"

// mergeOrderFollowupGHCall is the fake gh's log line for the follow-up
// lookup of branch: the entry's lookup with mergedAt added.
func mergeOrderFollowupGHCall(branch string) string {
	return mergeOrderGHCall(branch) + ",mergedAt"
}

// followup records a follow-up of issue after branch in fx's ledger at
// mergeOrderFollowupAddedAt and returns its ID.
func (fx *mergeOrderFixture) followup(t *testing.T, issue int, branch string) string {
	t.Helper()
	s := org.NewFollowupStore(fx.stateDir)
	s.Now = func() time.Time { return mergeOrderFollowupAddedAt }
	id, err := s.Add(org.FollowupAdd{Issue: issue, AfterBranch: branch})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// mergeOrderFollowupGHPR is one PR of the fake gh's output with its merge
// time.
type mergeOrderFollowupGHPR struct {
	mergeOrderGHPR
	MergedAt string `json:"mergedAt,omitempty"`
}

// followupPRs makes the fake gh print prs for `gh pr list --head <branch>`.
func (fx *mergeOrderFixture) followupPRs(t *testing.T, branch string, prs ...mergeOrderFollowupGHPR) {
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

// mergeOrderFollowupAt is mergeOrderFollowupAddedAt moved by d, as RFC3339.
func mergeOrderFollowupAt(d time.Duration) string {
	return mergeOrderFollowupAddedAt.Add(d).Format(time.RFC3339)
}

// splitMergeOrderFollowups cuts merge-order's text out at the follow-up
// section: the text before it, which parseMergeOrderText reads, and the
// section's rows, each split on tabs into 6 fields, nil with no section. It
// fails the test on a section out of printMergeOrderFollowups's shape.
func splitMergeOrderFollowups(t *testing.T, out string) (before string, rows [][]string) {
	t.Helper()
	head, section, found := strings.Cut(out, "\n\nfollow-ups:\n")
	if !found {
		if strings.Contains(out, "follow-ups:") || strings.Contains(out, "\n\n") {
			t.Fatalf("a follow-up section out of its shape:\n%s", out)
		}
		return out, nil
	}
	body, ok := strings.CutSuffix(section, "\n")
	if !ok {
		t.Fatalf("the follow-up section does not end with a newline: %q", section)
	}
	lines := strings.Split(body, "\n")
	if lines[0] != mergeOrderFollowupHeader {
		t.Fatalf("follow-up header = %q; want %q\n%s", lines[0], mergeOrderFollowupHeader, out)
	}
	for _, line := range lines[1:] {
		fields := strings.Split(line, "\t")
		if len(fields) != 6 {
			t.Fatalf("follow-up row %q has %d tab-separated fields; want 6\n%s", line, len(fields), out)
		}
		rows = append(rows, fields)
	}
	return head + "\n", rows
}

// seedMergeOrderFollowups gives fx's ledger, besides seedReadyMergeOrderBranch's
// feat/a, four open follow-ups and a done one: f1 after feat/done, whose PR
// #4 was merged after the record; f2 after feat/open, whose PR #5 is open and
// has a url with a tab; f3 after feat/none, which has no PR; f4 after
// feat/old, whose PR #6 was merged before the record; and f5, done, after
// feat/done again.
func seedMergeOrderFollowups(t *testing.T, fx *mergeOrderFixture) {
	t.Helper()
	seedReadyMergeOrderBranch(t, fx)
	for i, branch := range []string{"feat/done", "feat/open", "feat/none", "feat/old", "feat/done"} {
		fx.followup(t, 270+i, branch)
	}
	if _, err := org.NewFollowupStore(fx.stateDir).Done("f5", ""); err != nil {
		t.Fatal(err)
	}
	fx.followupPRs(t, "feat/done", mergeOrderFollowupGHPR{mergeOrderGHPR: mergeOrderGHPR{Number: 4, State: "MERGED",
		URL: "https://example.test/pull/4", BaseRefName: "main"}, MergedAt: mergeOrderFollowupAt(time.Minute)})
	fx.followupPRs(t, "feat/open", mergeOrderFollowupGHPR{mergeOrderGHPR: mergeOrderGHPR{Number: 5, State: "OPEN",
		URL: "https://example.test/pull/5\tINJECT", BaseRefName: "main"}})
	fx.followupPRs(t, "feat/old", mergeOrderFollowupGHPR{mergeOrderGHPR: mergeOrderGHPR{Number: 6, State: "MERGED",
		URL: "https://example.test/pull/6", BaseRefName: "main"}, MergedAt: mergeOrderFollowupAt(-time.Hour)})
}

// AC3 and AC7 through the CLI: after the table and its notes, a blank line,
// `follow-ups:`, and the open follow-ups in ID order (the done one left out),
// each with its issue, after-branch, state, PR and detail, values escaped as
// in the entries' table; gh runs once for the entry and once per distinct
// after-branch, asking for mergedAt too. It exits 0 with nothing on stderr.
func TestOrgMergeOrder_FollowupSectionFollowsTheNotes(t *testing.T) {
	fx := newMergeOrderFixture(t, true)
	seedMergeOrderFollowups(t, fx)

	stdout, stderr, err := fx.run(t)
	if err != nil || stderr != "" {
		t.Fatalf("merge-order: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	before, rows := splitMergeOrderFollowups(t, stdout)
	out := parseMergeOrderText(t, before)
	if len(out.rows) != 1 || out.rows[0][2] != "feat/a" || len(out.notes) == 0 {
		t.Errorf("entries %q, notes %q; want feat/a's row and the notes before the follow-ups", out.rows, out.notes)
	}
	added := mergeOrderFollowupAt(0)
	want := []struct{ id, issue, branch, state, pr, detail string }{
		{"f1", "270", "feat/done", "ready", "#4 https://example.test/pull/4", "PR #4 was merged at " + mergeOrderFollowupAt(time.Minute) +
			", after the follow-up was recorded at " + added + ": ask the human about issue #270 now, then run ralph org followup done f1"},
		{"f2", "271", "feat/open", "waiting", `#5 https://example.test/pull/5\tINJECT`, "PR #5 is open: the follow-up waits for its merge"},
		{"f3", "272", "feat/none", "waiting", "-", "feat/none has no PR yet"},
		{"f4", "273", "feat/old", "waiting", "#6 https://example.test/pull/6", "PR #6 was merged before the follow-up was recorded"},
	}
	if len(rows) != len(want) {
		t.Fatalf("follow-up rows = %q; want %d", rows, len(want))
	}
	for i, w := range want {
		if got := rows[i]; !slices.Equal(got[:5], []string{w.id, w.issue, w.branch, w.state, w.pr}) || !strings.Contains(got[5], w.detail) {
			t.Errorf("follow-up row %d = %q\nwant %q with a detail containing %q", i+1, got, []string{w.id, w.issue, w.branch, w.state, w.pr}, w.detail)
		}
	}
	wantCalls := []string{mergeOrderGHCall("feat/a"), mergeOrderFollowupGHCall("feat/done"), mergeOrderFollowupGHCall("feat/open"),
		mergeOrderFollowupGHCall("feat/none"), mergeOrderFollowupGHCall("feat/old")}
	if calls := fx.ghCalls(t); !slices.Equal(calls, wantCalls) {
		t.Errorf("gh calls = %q\nwant %q", calls, wantCalls)
	}
}

// AC3's JSON through the CLI: --json has followups, the object `ralph status
// --json` gives, with items holding each open follow-up's id, issue,
// after_branch, added_at, state, source gh, pr (number and url, or null),
// merged_at and detail, in the order of the text, and the record's
// corrupt_lines, ignored_events and error.
func TestOrgMergeOrder_FollowupJSONHasEveryOpenFollowup(t *testing.T) {
	fx := newMergeOrderFixture(t, true)
	seedMergeOrderFollowups(t, fx)

	res, raw := fx.runJSON(t)
	if len(res.Followups.Items) != 4 {
		t.Fatalf("followups = %+v; want four", res.Followups)
	}
	ready := org.MergeOrderFollowup{
		ID: "f1", Issue: 270, AfterBranch: "feat/done", AddedAt: mergeOrderFollowupAt(0), State: org.FollowupMergeReady,
		Source: org.FollowupMergeSourceGH, PR: &org.MergeOrderFollowupPR{Number: 4, URL: "https://example.test/pull/4"},
		MergedAt: mergeOrderFollowupAt(time.Minute), Detail: res.Followups.Items[0].Detail,
	}
	if got := res.Followups.Items[0]; got.PR == nil || *got.PR != *ready.PR || got.ID != ready.ID || got.Issue != ready.Issue ||
		got.AfterBranch != ready.AfterBranch || got.AddedAt != ready.AddedAt || got.State != ready.State || got.Source != ready.Source ||
		got.MergedAt != ready.MergedAt || !strings.Contains(got.Detail, "after the follow-up was recorded") {
		t.Errorf("f1 = %+v; want %+v", got, ready)
	}
	var ids, states []string
	for _, f := range res.Followups.Items {
		ids, states = append(ids, f.ID), append(states, f.State)
	}
	if !slices.Equal(ids, []string{"f1", "f2", "f3", "f4"}) || !slices.Equal(states, []string{"ready", "waiting", "waiting", "waiting"}) {
		t.Errorf("followups %q with states %q; want f1 to f4, ready then waiting", ids, states)
	}
	var keys struct {
		Followups struct {
			Items         []map[string]json.RawMessage `json:"items"`
			CorruptLines  json.RawMessage              `json:"corrupt_lines"`
			IgnoredEvents json.RawMessage              `json:"ignored_events"`
			Error         json.RawMessage              `json:"error"`
		} `json:"followups"`
	}
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		t.Fatal(err)
	}
	if pr := string(keys.Followups.Items[2]["pr"]); pr != "null" {
		t.Errorf("f3's pr = %s; want null", pr)
	}
	if mergedAt := string(keys.Followups.Items[3]["merged_at"]); mergedAt != strconv.Quote(mergeOrderFollowupAt(-time.Hour)) {
		t.Errorf("f4's merged_at = %s; want the old merge's time", mergedAt)
	}
	if got := []string{string(keys.Followups.CorruptLines), string(keys.Followups.IgnoredEvents), string(keys.Followups.Error)}; !slices.Equal(got, []string{"0", "0", `""`}) {
		t.Errorf("followups corrupt_lines, ignored_events, error = %q; want 0, 0 and \"\"", got)
	}
}

// With no follow-up, the text has no follow-up section (no blank line, no
// `follow-ups:`), --json has followups with an empty items list, and gh runs
// only for the entries.
func TestOrgMergeOrder_NoFollowupPrintsNoSectionAndAnEmptyJSONList(t *testing.T) {
	fx := newMergeOrderFixture(t, true)
	seedReadyMergeOrderBranch(t, fx)

	stdout, stderr, err := fx.run(t)
	if err != nil || stderr != "" {
		t.Fatalf("merge-order: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	if before, rows := splitMergeOrderFollowups(t, stdout); rows != nil || before != stdout {
		t.Errorf("a follow-up section with no follow-up:\n%s", stdout)
	}
	_, raw := fx.runJSON(t)
	var keys map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		t.Fatal(err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, keys["followups"]); err != nil {
		t.Fatal(err)
	}
	if got, want := compact.String(), `{"items":[],"corrupt_lines":0,"ignored_events":0,"error":""}`; got != want {
		t.Errorf("followups = %q; want %q", got, want)
	}
	if calls := fx.ghCalls(t); !slices.Equal(calls, []string{mergeOrderGHCall("feat/a"), mergeOrderGHCall("feat/a")}) {
		t.Errorf("gh calls = %q; want the entry's lookup alone in each run", calls)
	}
}

// With no gh and no feature branch, merge-order still lists the follow-ups,
// after the no-branches note, each unknown with the reason, and exits 0.
func TestOrgMergeOrder_FollowupWithoutGHIsUnknown(t *testing.T) {
	fx := newMergeOrderFixture(t, false)
	fx.followup(t, 270, "feat/done")

	stdout, stderr, err := fx.run(t)
	if err != nil || stderr != "" {
		t.Fatalf("merge-order: err %v, stderr %q (stdout: %s)", err, stderr, stdout)
	}
	before, rows := splitMergeOrderFollowups(t, stdout)
	if out := parseMergeOrderText(t, before); len(out.notes) == 0 || !strings.HasPrefix(out.notes[0], "no feature branches in the ledger") {
		t.Errorf("notes = %q; want the no-branches note", out.notes)
	}
	if len(rows) != 1 || !slices.Equal(rows[0][:5], []string{"f1", "270", "feat/done", "unknown", "-"}) ||
		!strings.Contains(rows[0][5], "gh was not found, so no PR was read") {
		t.Errorf("follow-up rows = %q; want f1 unknown because gh was not found", rows)
	}
}

// Damage in the follow-up record is warned about on stderr, as ralph org
// followup list warns about it, in the text and with --json, whose stdout
// stays one JSON value; the exit stays 0.
func TestOrgMergeOrder_FollowupDamageIsWarnedOnStderr(t *testing.T) {
	fx := newMergeOrderFixture(t, true)
	seedReadyMergeOrderBranch(t, fx)
	fx.followup(t, 270, "feat/done")
	path := org.FollowupsPathIn(fx.stateDir)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"ts":"2026-10-30T09:10:00Z","id":"f2","eve` + "\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	warning := "warning: 1 unreadable follow-up line(s) skipped in " + path + "\n"

	stdout, stderr, err := fx.run(t)
	if err != nil || stderr != warning {
		t.Fatalf("merge-order: err %v, stderr %q; want exit 0 and %q (stdout: %s)", err, stderr, warning, stdout)
	}
	if _, rows := splitMergeOrderFollowups(t, stdout); len(rows) != 1 || rows[0][0] != "f1" {
		t.Errorf("follow-up rows = %q; want f1", rows)
	}
	stdout, stderr, err = fx.run(t, "--json")
	if err != nil || stderr != warning {
		t.Fatalf("merge-order --json: err %v, stderr %q; want exit 0 and %q", err, stderr, warning)
	}
	var res org.MergeOrderResult
	if err := json.Unmarshal([]byte(stdout), &res); err != nil || len(res.Followups.Items) != 1 || res.Followups.CorruptLines != 1 {
		t.Errorf("--json stdout = %s (%v); want one JSON value with f1 and the corrupt line counted", stdout, err)
	}
}

// AC3's help: merge-order --help says it lists the open follow-ups, that
// --json's followups has the keys of ralph status's (self-review L3), how
// their state is decided (gh, a merge after the record), every reason a
// follow-up is unknown the CLI can meet (L6), and that ralph status decides
// from the ledger's records without gh, so the two can differ both ways and
// merge-order's answer is the one to use when one says ready and the other
// waiting, and status's, with merge-order's reason, when merge-order says
// unknown (M1 and R2-1).
func TestOrgMergeOrder_HelpDescribesTheFollowups(t *testing.T) {
	out, err := runOrgCmd(t, "merge-order", "--help")
	if err != nil {
		t.Fatal(err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"After the notes, merge-order lists the open follow-ups `ralph org followup add` recorded, under follow-ups: (ID, ISSUE, " +
			"AFTER_BRANCH, STATE, PR, DETAIL), and nothing when there is none.",
		"--json has them in followups, an object always there with the keys of the followups of `ralph status --json`: items, the " +
			"follow-ups, each with source gh and its pr; corrupt_lines and ignored_events, the record's damage; and error, why the " +
			"record could not be read, empty when it was read or does not exist.",
		"For each after-branch it runs one more gh pr list --head <branch>, asking for mergedAt too",
		"A follow-up is ready when one of them was merged at or after the follow-up was recorded (a PR merged before does not count, " +
			"since a branch name can be used again)",
		"and unknown, with the reason in DETAIL, when gh is missing, gh pr list fails or times out, gh's PRs are not of origin's " +
			"repository or cannot be checked against it, a merged PR's mergedAt cannot be read, gh returns its limit of 1000 PRs with " +
			"none merged after the follow-up, a PR's state is none of OPEN, MERGED and CLOSED, or the follow-up's added_at cannot be read.",
		"`ralph status` decides it from the ledger's records of merges (pr_merged, worktree_cleaned), without gh, so the two can " +
			"differ: a PR merged on GitHub and not cleaned up yet is waiting there, and one merged on GitHub before the follow-up was " +
			"recorded and cleaned up after it is ready there; when one says ready and the other waiting, use merge-order's " +
			"answer, and when merge-order says unknown, go by status's answer and give merge-order's reason.",
		"the follow-ups never change the exit.",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("merge-order --help does not say %q:\n%s", want, out)
		}
	}
}
