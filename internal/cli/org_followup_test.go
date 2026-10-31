package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for the `ralph org followup` verbs (issue #270; plan
// docs/plans/active/2026-10-30-org-followups.md, S2: AC1, D3, and the help
// of D5). None of them runs in parallel: the legacy-ledger test chdirs and
// swaps PATH.

// followupHeader is the first line of `ralph org followup list`'s table, and
// followupAllHeader that of `ralph org followup list --all`'s, which adds the
// done's time and note (self-review L7).
const (
	followupHeader    = "ID\tSTATE\tISSUE\tAFTER_BRANCH\tADDED_AT\tNOTE"
	followupAllHeader = followupHeader + "\tDONE_AT\tDONE_NOTE"
)

func readCLIFollowups(t *testing.T, stateDir string) org.FollowupReadResult {
	t.Helper()
	res, err := org.NewFollowupStore(stateDir).Read()
	if err != nil {
		t.Fatalf("read the follow-ups in %s: %v", stateDir, err)
	}
	return res
}

func cliFollowup(t *testing.T, stateDir, id string) org.Followup {
	t.Helper()
	for _, item := range readCLIFollowups(t, stateDir).Items {
		if item.ID == id {
			return item
		}
	}
	t.Fatalf("no follow-up %s in %s", id, stateDir)
	return org.Followup{}
}

func cliFollowupIDs(items []org.Followup) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

// followupFileOrAbsent returns followups.jsonl's content, or "<absent>".
func followupFileOrAbsent(t *testing.T, stateDir string) string {
	t.Helper()
	raw, err := os.ReadFile(org.FollowupsPathIn(stateDir))
	if errors.Is(err, fs.ErrNotExist) {
		return "<absent>"
	}
	if err != nil {
		t.Fatalf("read the follow-ups: %v", err)
	}
	return string(raw)
}

// followupTableRows returns `ralph org followup list`'s rows (the header
// checked and dropped), each split on tabs, with ADDED_AT checked to be
// RFC3339 and blanked so the rows compare as fixed values. With all, the
// header is followupAllHeader, and DONE_AT is checked and blanked the same
// way on a done row; an open row keeps its "-".
func followupTableRows(t *testing.T, stdout string, all bool) [][]string {
	t.Helper()
	header := followupHeader
	if all {
		header = followupAllHeader
	}
	columns := len(strings.Split(header, "\t"))
	lines := outputLines(stdout)
	if len(lines) == 0 || lines[0] != header {
		t.Fatalf("follow-up table does not start with the header %q:\n%s", header, stdout)
	}
	rows := make([][]string, 0, len(lines)-1)
	for _, line := range lines[1:] {
		row := strings.Split(line, "\t")
		if len(row) != columns {
			t.Fatalf("row %q has %d columns; want %d", line, len(row), columns)
		}
		if _, err := time.Parse(time.RFC3339, row[4]); err != nil {
			t.Errorf("row %q ADDED_AT is not RFC3339: %v", line, err)
		}
		row[4] = ""
		if all && row[1] == org.FollowupStateDone {
			if _, err := time.Parse(time.RFC3339, row[6]); err != nil {
				t.Errorf("row %q DONE_AT is not RFC3339: %v", line, err)
			}
			row[6] = ""
		}
		rows = append(rows, row)
	}
	return rows
}

// seedFollowups adds f1 (issue 270 after feat/a, no note, then done with a
// note) and f2 (issue 271 after feat/b, with a note, open) into stateDir
// through the CLI.
func seedFollowups(t *testing.T, stateDir string) {
	t.Helper()
	runInboxOK(t, "followup", "add", "--issue", "270", "--after-branch", "feat/a", "--state-dir", stateDir)
	runInboxOK(t, "followup", "add", "--issue", "271", "--after-branch", "feat/b", "--note", "ask about the split", "--state-dir", stateDir)
	runInboxOK(t, "followup", "done", "f1", "--note", "asked in #270", "--state-dir", stateDir)
}

// AC1: add prints the new ID, list shows the open follow-ups (done ones too
// with --all), and done moves one to done with its note, all through the
// record in the ledger.
func TestOrgFollowup_AddListDoneRoundTrip(t *testing.T) {
	stateDir := t.TempDir()

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"--issue", "270", "--after-branch", "feat/a"}, "added f1: issue #270 after feat/a\n"},
		{[]string{"--issue", "271", "--after-branch", "feat/b", "--note", "ask about the split"}, "added f2: issue #271 after feat/b\n"},
	} {
		stdout, stderr := runInboxOK(t, append(append([]string{"followup", "add"}, tc.args...), "--state-dir", stateDir)...)
		if stdout != tc.want || stderr != "" {
			t.Errorf("add %v printed stdout %q, stderr %q; want %q and nothing", tc.args, stdout, stderr, tc.want)
		}
	}

	stdout, stderr := runInboxOK(t, "followup", "list", "--state-dir", stateDir)
	if stderr != "" {
		t.Errorf("list stderr = %q; want nothing for an undamaged record", stderr)
	}
	if rows, want := followupTableRows(t, stdout, false), [][]string{
		{"f1", "open", "270", "feat/a", "", "-"},
		{"f2", "open", "271", "feat/b", "", "ask about the split"},
	}; !slices.EqualFunc(rows, want, slices.Equal) {
		t.Errorf("list rows = %q; want %q", rows, want)
	}

	if stdout, _ := runInboxOK(t, "followup", "done", "f1", "--note", "asked in #270", "--state-dir", stateDir); stdout != "done f1\n" {
		t.Errorf("done f1 stdout = %q; want %q", stdout, "done f1\n")
	}
	stdout, _ = runInboxOK(t, "followup", "list", "--state-dir", stateDir)
	if rows, want := followupTableRows(t, stdout, false), [][]string{{"f2", "open", "271", "feat/b", "", "ask about the split"}}; !slices.EqualFunc(rows, want, slices.Equal) {
		t.Errorf("list rows after done f1 = %q; want only f2", rows)
	}
	// With --all, DONE_AT and DONE_NOTE follow NOTE: the done's time and
	// note for f1, "-" for the open f2 (self-review L7).
	stdout, _ = runInboxOK(t, "followup", "list", "--all", "--state-dir", stateDir)
	if rows, want := followupTableRows(t, stdout, true), [][]string{
		{"f1", "done", "270", "feat/a", "", "-", "", "asked in #270"},
		{"f2", "open", "271", "feat/b", "", "ask about the split", "-", "-"},
	}; !slices.EqualFunc(rows, want, slices.Equal) {
		t.Errorf("list --all rows = %q; want f1 done with its done note and f2 open", rows)
	}

	f1, f2 := cliFollowup(t, stateDir, "f1"), cliFollowup(t, stateDir, "f2")
	if f1.State != org.FollowupStateDone || f1.DoneNote != "asked in #270" || f1.Issue != 270 || f1.AfterBranch != "feat/a" {
		t.Errorf("recorded f1 = %+v; want issue 270 after feat/a, done with its note", f1)
	}
	if f2.State != org.FollowupStateOpen || f2.Note != "ask about the split" || f2.Issue != 271 || f2.AfterBranch != "feat/b" {
		t.Errorf("recorded f2 = %+v; want issue 271 after feat/b, open with its note", f2)
	}
}

// AC1: --json carries the listed follow-ups with their recorded fields and
// the damage counts; --all adds the done ones.
func TestOrgFollowup_ListJSON(t *testing.T) {
	stateDir := t.TempDir()
	seedFollowups(t, stateDir)

	for _, tc := range []struct {
		args []string
		want []string
	}{
		{[]string{"followup", "list", "--json"}, []string{"f2"}},
		{[]string{"followup", "list", "--all", "--json"}, []string{"f1", "f2"}},
	} {
		stdout, stderr := runInboxOK(t, append(tc.args, "--state-dir", stateDir)...)
		if stderr != "" {
			t.Errorf("%v stderr = %q; want nothing", tc.args, stderr)
		}
		var payload orgFollowupJSON
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("%v stdout is not JSON: %v\n%s", tc.args, err, stdout)
		}
		if ids := cliFollowupIDs(payload.Items); !slices.Equal(ids, tc.want) {
			t.Errorf("%v items = %v; want %v", tc.args, ids, tc.want)
		}
		if payload.CorruptLines != 0 || payload.IgnoredEvents != 0 {
			t.Errorf("%v payload = %+v; want no damage", tc.args, payload)
		}
	}

	stdout, _ := runInboxOK(t, "followup", "list", "--all", "--json", "--state-dir", stateDir)
	var payload struct {
		Items         []map[string]any `json:"items"`
		CorruptLines  *int             `json:"corrupt_lines"`
		IgnoredEvents *int             `json:"ignored_events"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("list --all --json stdout is not JSON: %v\n%s", err, stdout)
	}
	if payload.CorruptLines == nil || payload.IgnoredEvents == nil || len(payload.Items) != 2 {
		t.Fatalf("list --all --json = %s; want items, corrupt_lines, and ignored_events", stdout)
	}
	f1, f2 := payload.Items[0], payload.Items[1]
	if f1["id"] != "f1" || f1["state"] != "done" || f1["issue"] != float64(270) || f1["after_branch"] != "feat/a" ||
		f1["done_note"] != "asked in #270" || f1["done_at"] == nil || f1["added_at"] == nil {
		t.Errorf("list --all --json f1 = %v; want issue 270 after feat/a, done with done_at and done_note", f1)
	}
	if _, hasNote := f1["note"]; hasNote {
		t.Errorf("list --all --json f1 = %v; want no note for a follow-up added without one", f1)
	}
	if _, hasDoneAt := f2["done_at"]; f2["id"] != "f2" || f2["state"] != "open" || f2["note"] != "ask about the split" || hasDoneAt {
		t.Errorf("list --all --json f2 = %v; want open with its note and no done_at", f2)
	}
}

// An empty record lists as one line saying so, --json as an empty items
// array, and reading it creates nothing.
func TestOrgFollowup_EmptyRecord(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"followup", "list"}, "no open follow-ups\n"},
		{[]string{"followup", "list", "--all"}, "no follow-ups\n"},
	} {
		if stdout, _ := runInboxOK(t, append(tc.args, "--state-dir", stateDir)...); stdout != tc.want {
			t.Errorf("%v stdout = %q; want %q", tc.args, stdout, tc.want)
		}
	}
	stdout, _ := runInboxOK(t, "followup", "list", "--json", "--state-dir", stateDir)
	for _, sub := range []string{`"items": []`, `"corrupt_lines": 0`, `"ignored_events": 0`} {
		if !strings.Contains(stdout, sub) {
			t.Errorf("list --json on an empty record = %s; want %s", stdout, sub)
		}
	}

	// Every follow-up of a record with only done ones is done.
	other := t.TempDir()
	runInboxOK(t, "followup", "add", "--issue", "1", "--after-branch", "feat/a", "--state-dir", other)
	runInboxOK(t, "followup", "done", "f1", "--state-dir", other)
	if stdout, _ := runInboxOK(t, "followup", "list", "--state-dir", other); stdout != "no open follow-ups\n" {
		t.Errorf("list of a record with only a done follow-up = %q; want no open follow-ups", stdout)
	}

	if _, err := os.Stat(stateDir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("listing an empty record created the state dir (stat err %v)", err)
	}
}

// AC1 and D1: a second done changes nothing, writes nothing, and exits 0,
// keeping the first done's note; an unknown ID exits 1 and writes nothing,
// also on a ledger with no record; a refused note leaves the follow-up open.
func TestOrgFollowupDone_SecondDoneAndUnknownID(t *testing.T) {
	stateDir := t.TempDir()
	runInboxOK(t, "followup", "add", "--issue", "270", "--after-branch", "feat/a", "--state-dir", stateDir)
	runInboxOK(t, "followup", "add", "--issue", "271", "--after-branch", "feat/b", "--state-dir", stateDir)
	runInboxOK(t, "followup", "done", "f1", "--note", "first", "--state-dir", stateDir)

	before := followupFileOrAbsent(t, stateDir)
	if stdout, _ := runInboxOK(t, "followup", "done", "f1", "--note", "second", "--state-dir", stateDir); stdout != "f1 is already done; nothing changed\n" {
		t.Errorf("second done f1 stdout = %q", stdout)
	}
	if after := followupFileOrAbsent(t, stateDir); after != before {
		t.Errorf("a second done wrote to the record:\nbefore: %s\nafter:  %s", before, after)
	}
	if got := cliFollowup(t, stateDir, "f1").DoneNote; got != "first" {
		t.Errorf("f1 done note = %q; want the first done's note", got)
	}

	for _, args := range [][]string{
		{"followup", "done", "f9"},
		{"followup", "done", "F2"},
		{"followup", "done", "f02"},
	} {
		stdout, _, err := runOrgCmdStreams(t, append(args, "--state-dir", stateDir)...)
		if !errors.Is(err, org.ErrFollowupUnknownID) {
			t.Errorf("%v error = %v; want ErrFollowupUnknownID", args, err)
		}
		if stdout != "" {
			t.Errorf("%v stdout = %q; want nothing", args, stdout)
		}
	}
	for _, tc := range []struct {
		args    []string
		wantSub string
	}{
		{[]string{"followup", "done"}, "accepts 1 arg(s)"},
		{[]string{"followup", "done", "f2", "--note", "one\ntwo"}, "the note must be a single line"},
		{[]string{"followup", "done", "f2", "--note", " "}, "the note is blank"},
	} {
		stdout, _, err := runOrgCmdStreams(t, append(tc.args, "--state-dir", stateDir)...)
		if err == nil || !strings.Contains(err.Error(), tc.wantSub) {
			t.Errorf("%v error = %v; want it to contain %q", tc.args, err, tc.wantSub)
		}
		if stdout != "" {
			t.Errorf("%v stdout = %q; want nothing", tc.args, stdout)
		}
	}
	if after := followupFileOrAbsent(t, stateDir); after != before {
		t.Errorf("a refused done wrote to the record:\nbefore: %s\nafter:  %s", before, after)
	}
	if got := cliFollowup(t, stateDir, "f2").State; got != org.FollowupStateOpen {
		t.Errorf("f2 state = %s; want open after the refused dones", got)
	}

	empty := filepath.Join(t.TempDir(), "state")
	if _, _, err := runOrgCmdStreams(t, "followup", "done", "f1", "--state-dir", empty); !errors.Is(err, org.ErrFollowupUnknownID) {
		t.Errorf("done f1 on a ledger with no record: error = %v; want ErrFollowupUnknownID", err)
	}
	if _, err := os.Stat(empty); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a done on a ledger with no record created the state dir (stat err %v)", err)
	}
}

// D3: add refuses an issue that is not a positive number, an after-branch
// that is empty, holds whitespace or a control character, or starts with -,
// and a note that is blank, not one line, too long, or holds a control
// character, and then writes nothing, not even the state dir.
func TestOrgFollowupAdd_ValidationErrorsWriteNothing(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	branch := []string{"--after-branch", "feat/a"}
	issue := []string{"--issue", "270"}
	for _, tc := range []struct {
		name    string
		args    []string
		wantSub string
	}{
		{"no issue", branch, "org: followup add: --issue is required"},
		{"issue 0", append([]string{"--issue", "0"}, branch...), "the issue must be a positive number, not 0"},
		{"a negative issue", append([]string{"--issue=-3"}, branch...), "the issue must be a positive number, not -3"},
		{"an issue that is not a number", append([]string{"--issue", "#270"}, branch...), "invalid argument"},
		{"no after-branch", issue, "org: followup add: --after-branch is required"},
		{"an empty after-branch", append(slices.Clone(issue), "--after-branch", ""), "the after-branch is required"},
		{"an after-branch that starts with -", append(slices.Clone(issue), "--after-branch=-x"), `the after-branch "-x" starts with -`},
		{"an after-branch with a space", append(slices.Clone(issue), "--after-branch", "feat/a b"), "whitespace or a control character (U+0020)"},
		{"an after-branch with a newline", append(slices.Clone(issue), "--after-branch", "feat/a\nfeat/b"), "(U+000A)"},
		{"a multi-line note", append(append(slices.Clone(issue), branch...), "--note", "one\ntwo"), "the note must be a single line"},
		{"a blank note", append(append(slices.Clone(issue), branch...), "--note", "  "), "the note is blank"},
		{"a note of 501 characters", append(append(slices.Clone(issue), branch...), "--note", strings.Repeat("n", org.InboxNoteMaxRunes+1)),
			"the note is 501 characters; at most 500"},
		{"a note with a tab", append(append(slices.Clone(issue), branch...), "--note", "a\tb"), "the note contains a control character (U+0009)"},
		{"an argument", append(append(slices.Clone(issue), branch...), "extra"), `unknown command "extra"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, _, err := runOrgCmdStreams(t, append(append([]string{"followup", "add"}, tc.args...), "--state-dir", stateDir)...)
			if err == nil || !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error = %v; want it to contain %q", err, tc.wantSub)
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			if _, err := os.Stat(stateDir); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("a refused add created the state dir (stat err %v)", err)
			}
		})
	}

	// The longest note and an empty --note (no note) are accepted.
	longest := strings.Repeat("あ", org.InboxNoteMaxRunes)
	runInboxOK(t, "followup", "add", "--issue", "1", "--after-branch", "feat/a", "--note", longest, "--state-dir", stateDir)
	runInboxOK(t, "followup", "add", "--issue", "2", "--after-branch", "feat/b", "--note", "", "--state-dir", stateDir)
	if f1, f2 := cliFollowup(t, stateDir, "f1"), cliFollowup(t, stateDir, "f2"); f1.Note != longest || f2.Note != "" {
		t.Errorf("f1 note has %d characters, f2 note = %q; want %d and none", len([]rune(f1.Note)), f2.Note, org.InboxNoteMaxRunes)
	}
}

// D3: the followup verbs refuse --org-id instead of ignoring it, before they
// read or write the record.
func TestOrgFollowup_RefusesOrgID(t *testing.T) {
	stateDir := t.TempDir()
	runInboxOK(t, "followup", "add", "--issue", "270", "--after-branch", "feat/a", "--state-dir", stateDir)
	before := followupFileOrAbsent(t, stateDir)

	for _, args := range [][]string{
		{"followup", "add", "--issue", "271", "--after-branch", "feat/b"},
		{"followup", "list"},
		{"followup", "list", "--json"},
		{"followup", "done", "f1"},
	} {
		stdout, _, err := runOrgCmdStreams(t, append(args, "--org-id", "org-a", "--state-dir", stateDir)...)
		if err == nil || !strings.Contains(err.Error(), "--org-id does not apply: a follow-up belongs to the whole ledger") {
			t.Errorf("%v --org-id error = %v; want a refusal", args, err)
		}
		if stdout != "" {
			t.Errorf("%v --org-id stdout = %q; want nothing", args, stdout)
		}
	}
	if after := followupFileOrAbsent(t, stateDir); after != before {
		t.Errorf("a refused followup verb wrote to the record:\nbefore: %s\nafter:  %s", before, after)
	}
}

// AC2 on the CLI side: lines that do not parse and events the fold ignores
// are counted in --json and warned about on stderr, never on stdout; the
// warning names the kinds of events ignored (self-review L5).
func TestOrgFollowup_WarnsAboutDamageOnStderr(t *testing.T) {
	stateDir := t.TempDir()
	content := `{"ts":"2026-10-30T12:00:00Z","id":"f1","event":"added","issue":270,"after_branch":"feat/a"}` + "\n" +
		"not json\n" +
		`{"ts":"2026-10-30T12:00:01Z","id":"f9","event":"done"}` + "\n" +
		`{"ts":"2026-10-30T12:00:02Z","id":"f2","event":"added","issue":271,"after_branch":"-x"}` + "\n"
	if err := os.WriteFile(org.FollowupsPathIn(stateDir), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{{"followup", "list"}, {"followup", "list", "--json"}} {
		stdout, stderr := runInboxOK(t, append(args, "--state-dir", stateDir)...)
		for _, sub := range []string{"1 unreadable follow-up line(s) skipped", "2 follow-up event(s) ignored", org.FollowupsPathIn(stateDir),
			"(unknown kind or ID, a second added, an added event that add would refuse, or a done with no open follow-up)"} {
			if !strings.Contains(stderr, sub) {
				t.Errorf("%v stderr %q does not contain %q", args, stderr, sub)
			}
		}
		if strings.Contains(stdout, "warning") {
			t.Errorf("%v printed the warning on stdout: %s", args, stdout)
		}
	}
	stdout, _ := runInboxOK(t, "followup", "list", "--json", "--state-dir", stateDir)
	var payload orgFollowupJSON
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
	}
	if payload.CorruptLines != 1 || payload.IgnoredEvents != 2 || !slices.Equal(cliFollowupIDs(payload.Items), []string{"f1"}) {
		t.Errorf("payload = %+v; want 1 corrupt line, 2 ignored events, and f1", payload)
	}
	// The next ID passes the damaged f9 and the ignored f2.
	if stdout, _ := runInboxOK(t, "followup", "add", "--issue", "272", "--after-branch", "feat/c", "--state-dir", stateDir); stdout != "added f10: issue #272 after feat/c\n" {
		t.Errorf("add after the damage stdout = %q; want f10", stdout)
	}
}

// A record edited by hand reaches the terminal escaped: control characters
// in a note are printed as escapes, the table keeps its six columns (eight
// with --all, whose DONE_NOTE is escaped the same way), and an empty value
// is "-".
func TestOrgFollowup_EscapesControlCharactersInTheRecord(t *testing.T) {
	stateDir := t.TempDir()
	var content []byte
	for _, ev := range []org.FollowupEvent{
		{TS: "2026-10-30T12:00:00Z", ID: "f1", Event: org.FollowupEventAdded, Issue: 270, AfterBranch: "feat/a",
			Note: "a" + string(rune(0x1b)) + "[2J\tb\nc"},
		{ID: "f2", Event: org.FollowupEventAdded, Issue: 271, AfterBranch: "feat/b"},
		{TS: "2026-10-30T12:00:01Z", ID: "f3", Event: org.FollowupEventAdded, Issue: 272, AfterBranch: "feat/c"},
		{TS: "2026-10-30T12:00:02Z", ID: "f3", Event: org.FollowupEventDone, Note: "d" + string(rune(0x1b)) + "[2J\te"},
	} {
		line, err := json.Marshal(ev)
		if err != nil {
			t.Fatal(err)
		}
		content = append(append(content, line...), '\n')
	}
	if err := os.WriteFile(org.FollowupsPathIn(stateDir), content, 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _ := runInboxOK(t, "followup", "list", "--state-dir", stateDir)
	if strings.ContainsRune(stdout, 0x1b) {
		t.Errorf("list printed a raw ESC: %q", stdout)
	}
	lines := outputLines(stdout)
	if len(lines) != 3 || lines[0] != followupHeader {
		t.Fatalf("list = %q; want the header and two rows", lines)
	}
	if row := strings.Split(lines[1], "\t"); len(row) != 6 || row[5] != `a\x1b[2J\tb\nc` {
		t.Errorf("f1 row = %q; want 6 columns and the note escaped", row)
	}
	if row := strings.Split(lines[2], "\t"); !slices.Equal(row, []string{"f2", "open", "271", "feat/b", "-", "-"}) {
		t.Errorf("f2 row = %q; want - for its empty ADDED_AT and NOTE", row)
	}

	stdout, _ = runInboxOK(t, "followup", "list", "--all", "--state-dir", stateDir)
	if strings.ContainsRune(stdout, 0x1b) {
		t.Errorf("list --all printed a raw ESC: %q", stdout)
	}
	lines = outputLines(stdout)
	if len(lines) != 4 || lines[0] != followupAllHeader {
		t.Fatalf("list --all = %q; want the header and three rows", lines)
	}
	if row := strings.Split(lines[3], "\t"); !slices.Equal(row,
		[]string{"f3", "done", "272", "feat/c", "2026-10-30T12:00:01Z", "-", "2026-10-30T12:00:02Z", `d\x1b[2J\te`}) {
		t.Errorf("f3 row = %q; want 8 columns and the done note escaped", row)
	}
}

// The verbs share the ledger every worktree resolves to, so from a linked
// worktree add records into the main checkout's ledger. add and done change
// the ledger, so a legacy worktree ledger with active seats refuses them,
// though input they refuse (add's, done's note) is reported first, before
// the ledger is resolved; list only reads it and prints the note once.
func TestOrgFollowup_LegacyLedgerGuard(t *testing.T) {
	setupOrgStubPATH(t)
	r := newLegacyLedgerRepo(t)

	if stdout, _ := runInboxOK(t, "followup", "add", "--issue", "270", "--after-branch", "feat/a"); stdout != "added f1: issue #270 after feat/a\n" {
		t.Fatalf("add from the worktree stdout = %q", stdout)
	}
	if got := followupFileOrAbsent(t, r.sharedDir); got == "<absent>" {
		t.Fatalf("no follow-up record in the shared ledger %s", r.sharedDir)
	}
	if got := followupFileOrAbsent(t, r.legacyDir); got != "<absent>" {
		t.Errorf("the worktree got its own follow-up record in %s: %s", r.legacyDir, got)
	}

	spawnLedgerSeat(t, r.legacyDir, "seat-1")
	before := readFileBytes(t, org.FollowupsPathIn(r.sharedDir))
	for _, args := range [][]string{
		{"followup", "add", "--issue", "271", "--after-branch", "feat/b"},
		{"followup", "done", "f1"},
	} {
		stdout, _, err := runOrgCmdStreams(t, args...)
		if err == nil || !strings.Contains(err.Error(), "refusing to change the org ledger") {
			t.Errorf("%v error = %v; want the legacy-ledger refusal", args, err)
		}
		if stdout != "" {
			t.Errorf("%v stdout = %q; want nothing", args, stdout)
		}
	}
	// Refused input is checked before the ledger is resolved, so it is
	// reported as such, with no legacy-ledger note or refusal: add's input,
	// and done's note (self-review L5).
	for _, tc := range []struct {
		args    []string
		wantSub string
	}{
		{[]string{"followup", "add", "--issue", "0", "--after-branch", "feat/b"}, "the issue must be a positive number, not 0"},
		{[]string{"followup", "done", "f1", "--note", "one\ntwo"}, "the note must be a single line"},
		{[]string{"followup", "done", "f1", "--note", " "}, "the note is blank"},
	} {
		stdout, stderr, err := runOrgCmdStreams(t, tc.args...)
		if err == nil || !strings.Contains(err.Error(), tc.wantSub) || strings.Contains(err.Error(), "refusing to change the org ledger") {
			t.Errorf("%v error = %v; want only %q", tc.args, err, tc.wantSub)
		}
		if stdout != "" || strings.Contains(stderr, legacyNoteMarker) {
			t.Errorf("%v stdout = %q, stderr = %q; want nothing on stdout and no legacy-ledger note", tc.args, stdout, stderr)
		}
	}
	if after := readFileBytes(t, org.FollowupsPathIn(r.sharedDir)); !bytes.Equal(before, after) {
		t.Errorf("a refused verb wrote to the record:\nbefore: %s\nafter:  %s", before, after)
	}

	for _, args := range [][]string{{"followup", "list"}, {"followup", "list", "--json"}} {
		stdout, stderr, err := runOrgCmdStreams(t, args...)
		if err != nil {
			t.Errorf("%v: %v (stderr: %s)", args, err, stderr)
			continue
		}
		if n := strings.Count(stderr, legacyNoteMarker); n != 1 {
			t.Errorf("%v printed the legacy-ledger note %d time(s): %s", args, n, stderr)
		}
		if !strings.Contains(stdout, "f1") {
			t.Errorf("%v stdout = %q; want f1 from the shared ledger", args, stdout)
		}
	}
}

// AC1, AC2, and D5: the help says what add records, that a branch already
// merged needs no follow-up, where readiness is shown, and that when status
// and merge-order differ merge-order's answer is the one to use (self-review
// M1); list's help points at ralph status and ralph org merge-order for
// readiness and names --all's DONE_AT and DONE_NOTE (L7); and
// `ralph org --help` names the verb group and lists it among the verbs that
// refuse --org-id. A mistyped verb is an unknown command.
func TestOrgFollowup_Help(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		wants []string
	}{
		{[]string{"followup", "add", "--help"}, []string{
			"ralph org followup add records a follow-up in the ledger and prints its ID (f1, f2, ...): " +
				"issue --issue, which the director asks the human about only once the PR of --after-branch is merged, with an optional --note.",
			"--issue is a positive number. --after-branch is not empty, holds no whitespace or control character, and does not start with -",
			"--note is one line, at most 500 characters, with no control characters. Input that fails these checks is refused and nothing is written.",
			"`ralph status` (from the ledger's pr_merged and worktree_cleaned records) and `ralph org merge-order` (from the PR, with gh) " +
				"show the follow-up as ready once the after-branch's PR is merged after the follow-up was recorded.",
			"A merge from before it does not count, since a branch name can be used again.",
			"So a branch whose PR is already merged needs no follow-up: ask the human about the issue now instead.",
			"One case still differs: a PR merged on GitHub before the follow-up was recorded and cleaned up with `ralph org cleanup` " +
				"after it shows as ready in `ralph status` (from its worktree_cleaned) and as waiting in `ralph org merge-order`; " +
				"when the two differ, use merge-order's answer.",
		}},
		{[]string{"followup", "list", "--help"}, []string{
			"open ones by default, done ones too with --all, as JSON with --json.",
			"With --all, DONE_AT (when done recorded it) and DONE_NOTE (done's note) follow, both - for an open follow-up; " +
				"--json has them as done_at and done_note on a done one.",
			"list shows only the record, not whether a follow-up is ready: `ralph status` says that from the ledger's records of merges, " +
				"and `ralph org merge-order` from the after-branch's PR, with gh.",
		}},
		{[]string{"followup", "done", "--help"}, []string{
			"--note is optional",
			"On a follow-up that is already done it changes nothing and exits 0, and the first done's note stays; an unknown ID exits 1.",
		}},
		{[]string{"followup", "--help"}, []string{
			"<state-dir>/followups.jsonl",
			"A follow-up belongs to the whole ledger, not to one org, so these verbs take no --org-id.",
			"show which open follow-ups are ready; when the two differ, use merge-order's answer.",
			// The verb list carries each Short.
			"Record a follow-up: an issue to ask about once a branch's PR is merged", "List the follow-ups of the ledger", "Mark an open follow-up done",
		}},
		{[]string{"followup"}, []string{"<state-dir>/followups.jsonl"}},
		{[]string{"--help"}, []string{
			"`ralph org followup` records the issues the director asks the human about only once another branch's PR is merged (add, list, done), " +
				"and `ralph status` and `ralph org merge-order` show which of them are ready.",
			"approve, the director verbs, the inbox verbs, and the followup verbs, which refuse it",
			"Record issues to ask the human about once another branch's PR is merged",
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

	if _, err := runOrgCmd(t, "followup", "ad"); err == nil || !strings.Contains(err.Error(), `unknown command "ad"`) {
		t.Errorf("followup ad error = %v; want an unknown command", err)
	}
}
