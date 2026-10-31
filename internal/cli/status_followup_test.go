package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// The reservations and the follow-ups of `ralph status` (issue #270; plan
// docs/plans/active/2026-10-30-org-followups.md, D2, D4, D5, AC4, AC5 and
// AC8). The tests write the manifest events, ts included, and the follow-up
// record straight into a temp state dir; the follow-up states themselves are
// covered by internal/org/followup_test.go (FollowupRecordedMerge).

// statusFollowupAddedAt is when the tests' follow-ups are recorded.
var statusFollowupAddedAt = time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)

// statusFollowupAt is statusFollowupAddedAt moved by d, as RFC3339.
func statusFollowupAt(d time.Duration) string {
	return statusFollowupAddedAt.Add(d).Format(time.RFC3339)
}

// statusFollowupHeader is the header line of the follow-up table.
const statusFollowupHeader = "ID\tISSUE\tAFTER_BRANCH\tSTATE\tSOURCE\tDETAIL"

// statusFollowupWaiting is the detail of a waiting follow-up.
const statusFollowupWaiting = "no merge recorded in the ledger since the follow-up; ralph org merge-order asks GitHub"

// statusFollowupWant is one item of `ralph status --json`'s followups.
type statusFollowupWant struct {
	ID          string `json:"id"`
	Issue       int    `json:"issue"`
	AfterBranch string `json:"after_branch"`
	AddedAt     string `json:"added_at"`
	State       string `json:"state"`
	Source      string `json:"source"`
	MergedEvent string `json:"merged_event"`
	MergedAt    string `json:"merged_at"`
	Detail      string `json:"detail"`
}

// statusFollowupsWant is `ralph status --json`'s followups object.
type statusFollowupsWant struct {
	Items         []statusFollowupWant `json:"items"`
	CorruptLines  int                  `json:"corrupt_lines"`
	IgnoredEvents int                  `json:"ignored_events"`
	Error         string               `json:"error"`
}

// statusFollowupsNone is the followups object, compacted, with no record:
// no item, no damage, no error.
const statusFollowupsNone = `{"items":[],"corrupt_lines":0,"ignored_events":0,"error":""}`

// statusFollowupUnknown is the detail of a follow-up whose added_at is
// addedAt, not an RFC3339 time, as `ralph org merge-order` words it.
func statusFollowupUnknown(addedAt string) string {
	return fmt.Sprintf("the follow-up's added_at %q is not an RFC3339 time, so no merge can be compared with it", addedAt)
}

// statusFollowupsField decodes the followups of `ralph status --json` output,
// failing the test when it is not an object whose keys are exactly items,
// corrupt_lines, ignored_events and error, with items an array.
func statusFollowupsField(t *testing.T, out string) statusFollowupsWant {
	t.Helper()
	var fields map[string]json.RawMessage
	statusJSONField(t, out, "followups", &fields)
	var keys []string
	for k := range fields {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	if want := []string{"corrupt_lines", "error", "ignored_events", "items"}; !slices.Equal(keys, want) {
		t.Fatalf("followups keys = %q, want %q:\n%s", keys, want, out)
	}
	if !strings.HasPrefix(string(fields["items"]), "[") {
		t.Fatalf("followups items = %s, want an array:\n%s", fields["items"], out)
	}
	var got statusFollowupsWant
	statusJSONField(t, out, "followups", &got)
	return got
}

// row is w as a row of the follow-up table.
func (w statusFollowupWant) row() []string {
	return []string{w.ID, fmt.Sprint(w.Issue), w.AfterBranch, w.State, w.Source, w.Detail}
}

// statusFollowupReady is the detail of a follow-up of issue made ready by
// event at at.
func statusFollowupReady(event, at string, issue int, id string) string {
	return fmt.Sprintf("the ledger records %s at %s, after the follow-up was recorded at %s: ask the human about issue #%d now, "+
		"then run ralph org followup done %s", event, at, statusFollowupAt(0), issue, id)
}

// seedStatusEvents appends events to the manifest in stateDir as they are,
// ts included.
func seedStatusEvents(t *testing.T, stateDir string, events ...org.ManifestEvent) {
	t.Helper()
	store := org.NewManifestStoreAtPath(org.ManifestPathIn(stateDir))
	for _, ev := range events {
		if err := store.Append(ev); err != nil {
			t.Fatalf("seed manifest event %+v: %v", ev, err)
		}
	}
}

// addStatusFollowup records a follow-up of issue after branch in stateDir at
// statusFollowupAddedAt and returns its ID.
func addStatusFollowup(t *testing.T, stateDir string, issue int, branch string) string {
	t.Helper()
	s := org.NewFollowupStore(stateDir)
	s.Now = func() time.Time { return statusFollowupAddedAt }
	id, err := s.Add(org.FollowupAdd{Issue: issue, AfterBranch: branch})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// runStatusSplit runs `ralph status <args...>` and returns its stdout and
// stderr, failing the test when it fails.
func runStatusSplit(t *testing.T, args ...string) (stdout, stderr string) {
	t.Helper()
	stdout, stderr, err := runRootCmdSplit(t, append([]string{"status"}, args...)...)
	if err != nil {
		t.Fatalf("status %q: %v (stdout: %s, stderr: %s)", args, err, stdout, stderr)
	}
	return stdout, stderr
}

// statusGroupsText is the text of `ralph status` from its first `org_id:`
// line to the end, failing the test when there is none.
func statusGroupsText(t *testing.T, out string) string {
	t.Helper()
	i := strings.Index(out, "org_id: ")
	if i < 0 || (i > 0 && out[i-1] != '\n') {
		t.Fatalf("no `org_id:` line in:\n%s", out)
	}
	return out[i:]
}

// statusFollowupRows cuts the follow-up section out of `ralph status`'s text:
// its rows, each split on tabs into 6 fields, nil with no section. It fails
// the test on a section out of printStatusFollowups's shape.
func statusFollowupRows(t *testing.T, out string) [][]string {
	t.Helper()
	_, section, found := strings.Cut(out, "\n\nfollow-ups:\n")
	if !found {
		if strings.Contains(out, "follow-ups:") {
			t.Fatalf("a follow-ups: line not after a blank line:\n%s", out)
		}
		return nil
	}
	lines := strings.Split(strings.TrimSuffix(section, "\n"), "\n")
	if lines[0] != statusFollowupHeader {
		t.Fatalf("follow-up header = %q, want %q:\n%s", lines[0], statusFollowupHeader, out)
	}
	var rows [][]string
	for _, line := range lines[1:] {
		if line == "" {
			break
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 6 {
			t.Fatalf("follow-up row %q has %d fields, want 6:\n%s", line, len(fields), out)
		}
		rows = append(rows, fields)
	}
	return rows
}

// statusJSONField decodes the top-level field key of `ralph status --json`
// output into v, failing the test when it is missing.
func statusJSONField(t *testing.T, out, key string, v any) {
	t.Helper()
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json.Unmarshal: %v\noutput:\n%s", err, out)
	}
	raw, ok := payload[key]
	if !ok {
		t.Fatalf("--json carries no %s:\n%s", key, out)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("decode %s: %v\n%s", key, err, raw)
	}
}

// seedStatusReservations writes org-a (a leader and a reservation), org-b (a
// seat, no reservation), org-r (a reservation, no seat), org-d (a reservation
// released by disband), and org-c (a dry-run reservation).
func seedStatusReservations(t *testing.T, stateDir string) {
	t.Helper()
	ts := statusFollowupAt(-time.Hour)
	seedStatusEvents(t, stateDir,
		org.ManifestEvent{TS: ts, OrgID: "org-a", Event: org.EventScopeReserved, Details: "paths=docs/x.md,internal/auth/"},
		org.ManifestEvent{TS: ts, OrgID: "org-a", SeatID: "leader", Event: org.EventSpawned, Role: "leader", Driver: "claude", Model: "opus"},
		org.ManifestEvent{TS: ts, OrgID: "org-b", SeatID: "qa", Event: org.EventSpawned, Role: "qa", Driver: "codex", Model: "gpt"},
		org.ManifestEvent{TS: ts, OrgID: "org-r", Event: org.EventScopeReserved, Details: "paths=internal/r/"},
		org.ManifestEvent{TS: ts, OrgID: "org-d", Event: org.EventScopeReserved, Details: "paths=docs/d/"},
		org.ManifestEvent{TS: ts, OrgID: "org-d", Event: org.EventDisbanded},
		org.ManifestEvent{TS: ts, OrgID: "org-c", Event: org.EventScopeReserved, Details: "paths=cmd/", DryRun: true},
	)
}

// D4 and AC5: a running org's reservation is a `reserved:` line in its group,
// an org with a reservation and no seat is a group with that line only, and
// a reservation released by disband or made by a dry run shows nothing.
// --org-id keeps one org's group, a reservation-only one too.
func TestStatusCmd_ReservationLinesInTheirGroups(t *testing.T) {
	stateDir := t.TempDir()
	seedStatusReservations(t, stateDir)

	groupA := "org_id: org-a (active 1/1)\n" +
		"  SEAT_ID\tROLE\tDRIVER\tMODEL\tSTATE\tPANE_ID\n" +
		"  leader\tleader\tclaude\topus\tspawned (active)\t\n" +
		"  reserved: docs/x.md, internal/auth/\n"
	groupB := "org_id: org-b (active 1/1)\n" +
		"  SEAT_ID\tROLE\tDRIVER\tMODEL\tSTATE\tPANE_ID\n" +
		"  qa\tqa\tcodex\tgpt\tspawned (active)\t\n"
	groupR := "org_id: org-r (active 0/0)\n" +
		"  reserved: internal/r/\n"
	for _, tc := range []struct {
		name   string
		args   []string
		groups string
	}{
		{"every org", nil, groupA + "\n" + groupB + "\n" + groupR},
		{"--org-id of an org with seats", []string{"--org-id", "org-a"}, groupA},
		{"--org-id of an org with a reservation only", []string{"--org-id", "org-r"}, groupR},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, stderr := runStatusSplit(t, append([]string{"--state-dir", stateDir}, tc.args...)...)
			if got := statusGroupsText(t, out); got != tc.groups {
				t.Errorf("groups:\n got %q\nwant %q", got, tc.groups)
			}
			if stderr != "" {
				t.Errorf("stderr = %q, want none", stderr)
			}
		})
	}

	for _, orgID := range []string{"org-d", "org-c"} {
		out, _ := runStatusSplit(t, "--state-dir", stateDir, "--org-id", orgID)
		if !strings.Contains(out, "no org runtime state found (org-id="+orgID) || strings.Contains(out, "reserved:") {
			t.Errorf("--org-id %s shows a group or a reservation:\n%s", orgID, out)
		}
	}
}

// AC5: --json gives every org a reservation array, empty for an org that
// holds none, and lists a reservation-only org with no seats.
func TestStatusCmd_ReservationJSON(t *testing.T) {
	stateDir := t.TempDir()
	seedStatusReservations(t, stateDir)

	type orgJSON struct {
		OrgID       string            `json:"org_id"`
		Seats       []json.RawMessage `json:"seats"`
		ActiveCount int               `json:"active_count"`
		Reservation *[]string         `json:"reservation"`
	}
	for _, tc := range []struct {
		args []string
		want map[string][]string
	}{
		{nil, map[string][]string{"org-a": {"docs/x.md", "internal/auth/"}, "org-b": {}, "org-r": {"internal/r/"}}},
		{[]string{"--org-id", "org-r"}, map[string][]string{"org-r": {"internal/r/"}}},
		{[]string{"--org-id", "org-b"}, map[string][]string{"org-b": {}}},
	} {
		out, _ := runStatusSplit(t, append([]string{"--state-dir", stateDir, "--json"}, tc.args...)...)
		var orgs []orgJSON
		statusJSONField(t, out, "orgs", &orgs)
		var ids []string
		for _, o := range orgs {
			ids = append(ids, o.OrgID)
			want, ok := tc.want[o.OrgID]
			switch {
			case !ok:
				t.Errorf("%q: unexpected org %s:\n%s", tc.args, o.OrgID, out)
			case o.Reservation == nil:
				t.Errorf("%q: %s has no reservation array:\n%s", tc.args, o.OrgID, out)
			case !slices.Equal(*o.Reservation, want):
				t.Errorf("%q: %s reservation = %q, want %q", tc.args, o.OrgID, *o.Reservation, want)
			}
			if o.OrgID == "org-r" && (o.Seats == nil || len(o.Seats) != 0 || o.ActiveCount != 0) {
				t.Errorf("%q: org-r seats = %v, active_count %d; want an empty array and 0", tc.args, o.Seats, o.ActiveCount)
			}
		}
		if !slices.IsSorted(ids) || len(ids) != len(tc.want) {
			t.Errorf("%q: orgs = %q, want the %d orgs of %v in order", tc.args, ids, len(tc.want), tc.want)
		}
	}
}

// D4 and AC5: a ledger whose only org holds a reservation and has no seat is
// not empty. With no --org-id the text shows that org's group with its
// reserved line instead of the empty-roster note, and --json lists the org
// with no seat and its reservation.
func TestStatusCmd_ReservationOnlyLedgerShowsItsGroup(t *testing.T) {
	stateDir := t.TempDir()
	seedStatusEvents(t, stateDir, org.ManifestEvent{
		TS: statusFollowupAt(-time.Hour), OrgID: "org-r", Event: org.EventScopeReserved, Details: "paths=internal/r/",
	})

	out, stderr := runStatusSplit(t, "--state-dir", stateDir)
	if strings.Contains(out, "no org runtime state found") {
		t.Errorf("a ledger with a reservation reads as empty:\n%s", out)
	}
	if got, want := statusGroupsText(t, out), "org_id: org-r (active 0/0)\n  reserved: internal/r/\n"; got != want {
		t.Errorf("groups:\n got %q\nwant %q", got, want)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want none", stderr)
	}

	jsonOut, _ := runStatusSplit(t, "--state-dir", stateDir, "--json")
	var orgs []struct {
		OrgID       string            `json:"org_id"`
		Seats       []json.RawMessage `json:"seats"`
		ActiveCount int               `json:"active_count"`
		TotalCount  int               `json:"total_count"`
		Reservation []string          `json:"reservation"`
	}
	statusJSONField(t, jsonOut, "orgs", &orgs)
	if len(orgs) != 1 || orgs[0].OrgID != "org-r" || orgs[0].Seats == nil || len(orgs[0].Seats) != 0 ||
		orgs[0].ActiveCount != 0 || orgs[0].TotalCount != 0 || !slices.Equal(orgs[0].Reservation, []string{"internal/r/"}) {
		t.Errorf("orgs = %+v, want only org-r with no seat, counts 0, and its reservation:\n%s", orgs, jsonOut)
	}
}

// seedStatusFollowups writes, next to a seat of org-a, follow-ups whose
// after-branch the ledger records merged after them by pr_merged (f1) and,
// at the same second, by worktree_cleaned (f2), merged only before (f3,
// AC8), merged under another name only (f4), and one marked done with a
// merge after it (f5). It returns the follow-ups status shows, in order.
func seedStatusFollowups(t *testing.T, stateDir string) []statusFollowupWant {
	t.Helper()
	seedStatusEvents(t, stateDir, org.ManifestEvent{
		TS: statusFollowupAt(-2 * time.Hour), OrgID: "org-a", SeatID: "leader", Event: org.EventSpawned, Role: "leader",
	})
	f1 := addStatusFollowup(t, stateDir, 11, "feat/merged")
	f2 := addStatusFollowup(t, stateDir, 12, "feat/cleaned")
	f3 := addStatusFollowup(t, stateDir, 13, "feat/old")
	f4 := addStatusFollowup(t, stateDir, 14, "feat/none")
	f5 := addStatusFollowup(t, stateDir, 15, "feat/done")
	if _, err := org.NewFollowupStore(stateDir).Done(f5, ""); err != nil {
		t.Fatal(err)
	}
	seedStatusEvents(t, stateDir,
		org.ManifestEvent{TS: statusFollowupAt(-time.Hour), OrgID: "org-o", Event: org.EventPRMerged, Details: "branch=feat/old pr=3 head=abc"},
		org.ManifestEvent{TS: statusFollowupAt(time.Hour), OrgID: "org-m", Event: org.EventPRMerged, Details: "branch=feat/merged pr=1 head=abc"},
		org.ManifestEvent{TS: statusFollowupAt(0), OrgID: "org-c", Event: org.EventWorktreeCleaned, Details: `branch=feat/cleaned head=abc via=gh worktree="/w t"`},
		org.ManifestEvent{TS: statusFollowupAt(time.Hour), OrgID: "org-n", Event: org.EventPRMerged, Details: "branch=feat/none-2 pr=4 head=abc"},
		org.ManifestEvent{TS: statusFollowupAt(time.Hour), OrgID: "org-e", Event: org.EventPRMerged, Details: "branch=feat/done pr=5 head=abc"},
	)
	added := statusFollowupAt(0)
	return []statusFollowupWant{
		{ID: f1, Issue: 11, AfterBranch: "feat/merged", AddedAt: added, State: "ready", Source: "recorded", MergedEvent: "pr_merged",
			MergedAt: statusFollowupAt(time.Hour), Detail: statusFollowupReady("pr_merged", statusFollowupAt(time.Hour), 11, f1)},
		{ID: f2, Issue: 12, AfterBranch: "feat/cleaned", AddedAt: added, State: "ready", Source: "recorded", MergedEvent: "worktree_cleaned",
			MergedAt: added, Detail: statusFollowupReady("worktree_cleaned", added, 12, f2)},
		{ID: f3, Issue: 13, AfterBranch: "feat/old", AddedAt: added, State: "waiting", Source: "recorded", Detail: statusFollowupWaiting},
		{ID: f4, Issue: 14, AfterBranch: "feat/none", AddedAt: added, State: "waiting", Source: "recorded", Detail: statusFollowupWaiting},
	}
}

// D2, D5, AC4 and AC8: the open follow-ups come after the org groups, ready
// with the recorded event and its time when the ledger records a pr_merged
// or a worktree_cleaned of the after-branch at or after the follow-up, and
// waiting for a merge only before it, for another branch's merge, or for
// none. A done follow-up is not shown. --org-id does not filter them.
func TestStatusCmd_FollowupsReadyFromTheLedgersMerges(t *testing.T) {
	stateDir := t.TempDir()
	want := seedStatusFollowups(t, stateDir)
	var wantRows [][]string
	section := "\n\nfollow-ups:\n" + statusFollowupHeader + "\n"
	for _, w := range want {
		wantRows = append(wantRows, w.row())
		section += strings.Join(w.row(), "\t") + "\n"
	}

	for _, extra := range [][]string{nil, {"--org-id", "org-a"}, {"--org-id", "org-x"}} {
		out, stderr := runStatusSplit(t, append([]string{"--state-dir", stateDir}, extra...)...)
		if stderr != "" {
			t.Errorf("%q: stderr = %q, want none", extra, stderr)
		}
		if got := statusFollowupRows(t, out); !slices.EqualFunc(got, wantRows, slices.Equal[[]string]) {
			t.Errorf("%q: follow-up rows:\n got %q\nwant %q\n%s", extra, got, wantRows, out)
		}
		if !strings.HasSuffix(out, section) {
			t.Errorf("%q: the text does not end with the follow-up section %q:\n%s", extra, section, out)
		}

		jsonOut, _ := runStatusSplit(t, append([]string{"--state-dir", stateDir, "--json"}, extra...)...)
		got := statusFollowupsField(t, jsonOut)
		if !slices.Equal(got.Items, want) || got.CorruptLines != 0 || got.IgnoredEvents != 0 || got.Error != "" {
			t.Errorf("%q: followups:\n got %+v\nwant items %+v, no damage, no error", extra, got, want)
		}
	}
}

// The follow-up section follows the groups' last line, after a blank line,
// and the manifest's corrupt-line warning still comes last.
func TestStatusCmd_FollowupSectionComesAfterTheGroups(t *testing.T) {
	stateDir := t.TempDir()
	seedStatusEvents(t, stateDir, org.ManifestEvent{TS: statusFollowupAt(-time.Hour), OrgID: "org-a", SeatID: "leader", Event: org.EventSpawned, Role: "leader"})
	f1 := addStatusFollowup(t, stateDir, 7, "feat/x")
	f, err := os.OpenFile(org.ManifestPathIn(stateDir), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{not json\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	out, _ := runStatusSplit(t, "--state-dir", stateDir)
	want := "org_id: org-a (active 1/1)\n" +
		"  SEAT_ID\tROLE\tDRIVER\tMODEL\tSTATE\tPANE_ID\n" +
		"  leader\tleader\t\t\tspawned (active)\t\n" +
		"\nfollow-ups:\n" + statusFollowupHeader + "\n" +
		f1 + "\t7\tfeat/x\twaiting\trecorded\t" + statusFollowupWaiting + "\n" +
		"\nwarning: 1 corrupt manifest line(s) skipped\n"
	if got := statusGroupsText(t, out); got != want {
		t.Errorf("text after the groups:\n got %q\nwant %q", got, want)
	}
}

// D4: followups is always the same object in --json, with no item, no
// damage and no error when there is no record, on the empty-roster path and
// the populated one, and the text then has no section. With open follow-ups
// both paths list them, the text of the empty path after its last line.
func TestStatusCmd_FollowupJSONOnBothPaths(t *testing.T) {
	emptyDir, populatedDir := t.TempDir(), t.TempDir()
	seedTwoOrgManifest(t, populatedDir)
	for name, dir := range map[string]string{"empty": emptyDir, "populated": populatedDir} {
		out, _ := runStatusSplit(t, "--state-dir", dir)
		if strings.Contains(out, "follow-ups:") {
			t.Errorf("%s: a follow-up section with no follow-up:\n%s", name, out)
		}
		jsonOut, _ := runStatusSplit(t, "--state-dir", dir, "--json")
		var raw json.RawMessage
		statusJSONField(t, jsonOut, "followups", &raw)
		var compact bytes.Buffer
		if err := json.Compact(&compact, raw); err != nil {
			t.Fatal(err)
		}
		if compact.String() != statusFollowupsNone {
			t.Errorf("%s: followups = %s, want %s", name, compact.String(), statusFollowupsNone)
		}
		if _, err := os.Stat(org.FollowupsPathIn(dir)); !os.IsNotExist(err) {
			t.Errorf("%s: status created %s (stat: %v)", name, org.FollowupsPathIn(dir), err)
		}
	}

	for name, dir := range map[string]string{"empty": emptyDir, "populated": populatedDir} {
		id := addStatusFollowup(t, dir, 21, "feat/"+name)
		wantRow := []string{id, "21", "feat/" + name, "waiting", "recorded", statusFollowupWaiting}
		out, _ := runStatusSplit(t, "--state-dir", dir)
		if got := statusFollowupRows(t, out); len(got) != 1 || !slices.Equal(got[0], wantRow) {
			t.Errorf("%s: follow-up rows = %q, want [%q]:\n%s", name, got, wantRow, out)
		}
		if name == "empty" && !strings.HasSuffix(out, "Run `ralph doctor` to check environment readiness.\n\nfollow-ups:\n"+
			statusFollowupHeader+"\n"+strings.Join(wantRow, "\t")+"\n") {
			t.Errorf("empty: the follow-up section does not end the text after the doctor hint:\n%s", out)
		}
		jsonOut, _ := runStatusSplit(t, "--state-dir", dir, "--json")
		got := statusFollowupsField(t, jsonOut).Items
		want := []statusFollowupWant{{ID: id, Issue: 21, AfterBranch: "feat/" + name, AddedAt: statusFollowupAt(0), State: "waiting",
			Source: "recorded", Detail: statusFollowupWaiting}}
		if !slices.Equal(got, want) {
			t.Errorf("%s: followups items = %+v, want %+v", name, got, want)
		}
	}
}

// Damage in the follow-up record is warned about on stderr as the followup
// verbs warn about it, and the readable follow-ups are still shown. A record
// that cannot be read at all is warned about and shows no follow-up; status
// still exits 0. A follow-up whose added_at is not a time is unknown, saying
// so.
func TestStatusCmd_FollowupDamageIsWarnedOnStderr(t *testing.T) {
	stateDir := t.TempDir()
	seedStatusEvents(t, stateDir, org.ManifestEvent{TS: statusFollowupAt(time.Hour), OrgID: "org-m", Event: org.EventPRMerged, Details: "branch=feat/x pr=1"})
	path := org.FollowupsPathIn(stateDir)
	writeTestFile(t, path, `{"ts":"yesterday","id":"f1","event":"added","issue":3,"after_branch":"feat/x"}`+"\n"+
		"{not json\n"+
		`{"ts":"2026-10-30T09:00:00Z","id":"f2","event":"renamed"}`+"\n")

	out, stderr := runStatusSplit(t, "--state-dir", stateDir)
	for _, want := range []string{
		"warning: 1 unreadable follow-up line(s) skipped in " + path + "\n",
		"warning: 1 follow-up event(s) ignored in " + path + " (",
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr does not say %q:\n%s", want, stderr)
		}
	}
	wantRow := []string{"f1", "3", "feat/x", "unknown", "recorded", statusFollowupUnknown("yesterday")}
	if got := statusFollowupRows(t, out); len(got) != 1 || !slices.Equal(got[0], wantRow) {
		t.Errorf("follow-up rows = %q, want [%q]:\n%s", got, wantRow, out)
	}
	jsonOut, jsonErr := runStatusSplit(t, "--state-dir", stateDir, "--json")
	if jsonErr != stderr {
		t.Errorf("--json stderr = %q, want the text run's %q", jsonErr, stderr)
	}
	followups := statusFollowupsField(t, jsonOut).Items
	if len(followups) != 1 || followups[0].State != "unknown" || followups[0].Detail != wantRow[5] {
		t.Errorf("followups items = %+v, want f1 unknown with %q", followups, wantRow[5])
	}

	unreadable := t.TempDir()
	if err := os.Mkdir(org.FollowupsPathIn(unreadable), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, extra := range [][]string{nil, {"--json"}} {
		out, stderr := runStatusSplit(t, append([]string{"--state-dir", unreadable}, extra...)...)
		if !strings.HasPrefix(stderr, "warning: status: org: followup: read "+org.FollowupsPathIn(unreadable)+": ") ||
			!strings.HasSuffix(stderr, "; no follow-up is shown\n") {
			t.Errorf("%q: stderr = %q, want the warning that the record cannot be read", extra, stderr)
		}
		if extra == nil && strings.Contains(out, "follow-ups:") {
			t.Errorf("a follow-up section from a record that cannot be read:\n%s", out)
		}
		if extra != nil {
			if got := statusFollowupsField(t, out); len(got.Items) != 0 || got.Error == "" {
				t.Errorf("followups = %+v, want no item and the read error", got)
			}
		}
	}
}

// The followups object of --json carries, on the empty-roster path and the
// populated one, the record's damage counts next to its readable follow-ups,
// and for a record that cannot be read no item, no count, and the error
// stderr warns about: a reader tells it from a record with no open follow-up
// by error alone.
func TestStatusCmd_FollowupsJSONCarriesTheRecordsDamageAndReadError(t *testing.T) {
	for _, path := range []string{"empty", "populated"} {
		t.Run(path, func(t *testing.T) {
			damaged, unreadable := t.TempDir(), t.TempDir()
			if path == "populated" {
				seedTwoOrgManifest(t, damaged)
				seedTwoOrgManifest(t, unreadable)
			}
			writeTestFile(t, org.FollowupsPathIn(damaged),
				`{"ts":"`+statusFollowupAt(0)+`","id":"f1","event":"added","issue":3,"after_branch":"feat/x"}`+"\n"+
					"{not json\n"+
					`{"ts":"`+statusFollowupAt(0)+`","id":"f2","event":"renamed"}`+"\n"+
					`{"ts":"`+statusFollowupAt(0)+`","id":"f9","event":"done"}`+"\n")
			out, _ := runStatusSplit(t, "--state-dir", damaged, "--json")
			want := statusFollowupsWant{
				Items: []statusFollowupWant{{ID: "f1", Issue: 3, AfterBranch: "feat/x", AddedAt: statusFollowupAt(0), State: "waiting",
					Source: "recorded", Detail: statusFollowupWaiting}},
				CorruptLines: 1, IgnoredEvents: 2,
			}
			if got := statusFollowupsField(t, out); !slices.Equal(got.Items, want.Items) || got.CorruptLines != want.CorruptLines ||
				got.IgnoredEvents != want.IgnoredEvents || got.Error != "" {
				t.Errorf("damaged record: followups = %+v, want %+v", got, want)
			}

			if err := os.Mkdir(org.FollowupsPathIn(unreadable), 0o755); err != nil {
				t.Fatal(err)
			}
			out, stderr := runStatusSplit(t, "--state-dir", unreadable, "--json")
			got := statusFollowupsField(t, out)
			if len(got.Items) != 0 || got.CorruptLines != 0 || got.IgnoredEvents != 0 ||
				!strings.HasPrefix(got.Error, "org: followup: read "+org.FollowupsPathIn(unreadable)+": ") {
				t.Errorf("unreadable record: followups = %+v, want no item, no count, and the read error", got)
			}
			if want := "warning: status: " + got.Error + "; no follow-up is shown\n"; stderr != want {
				t.Errorf("unreadable record: stderr = %q, want %q", stderr, want)
			}
		})
	}
}

// As `ralph org merge-order` has it (D5), a follow-up whose added_at is not
// an RFC3339 time is unknown, not waiting, with the reason in DETAIL, even
// when the ledger records a merge of its after-branch; one with no merge
// recorded still waits, saying that ralph org merge-order asks GitHub. Both
// paths, the text and --json.
func TestStatusCmd_FollowupWithAnUnreadableAddedAtIsUnknown(t *testing.T) {
	const handEdited = "2026-10-30 09:00:00"
	want := []statusFollowupWant{
		{ID: "f1", Issue: 3, AfterBranch: "feat/x", AddedAt: handEdited, State: "unknown", Source: "recorded",
			Detail: statusFollowupUnknown(handEdited)},
		{ID: "f2", Issue: 4, AfterBranch: "feat/y", AddedAt: statusFollowupAt(0), State: "waiting", Source: "recorded",
			Detail: statusFollowupWaiting},
	}
	var wantRows [][]string
	for _, w := range want {
		wantRows = append(wantRows, w.row())
	}
	for _, path := range []string{"empty", "populated"} {
		t.Run(path, func(t *testing.T) {
			stateDir := t.TempDir()
			if path == "populated" {
				seedTwoOrgManifest(t, stateDir)
			}
			seedStatusEvents(t, stateDir, org.ManifestEvent{
				TS: statusFollowupAt(time.Hour), OrgID: "org-m", Event: org.EventPRMerged, Details: "branch=feat/x pr=1 head=abc",
			})
			writeTestFile(t, org.FollowupsPathIn(stateDir),
				`{"ts":"`+handEdited+`","id":"f1","event":"added","issue":3,"after_branch":"feat/x"}`+"\n"+
					`{"ts":"`+statusFollowupAt(0)+`","id":"f2","event":"added","issue":4,"after_branch":"feat/y"}`+"\n")

			out, _ := runStatusSplit(t, "--state-dir", stateDir)
			if got := statusFollowupRows(t, out); !slices.EqualFunc(got, wantRows, slices.Equal[[]string]) {
				t.Errorf("follow-up rows:\n got %q\nwant %q\n%s", got, wantRows, out)
			}
			jsonOut, _ := runStatusSplit(t, "--state-dir", stateDir, "--json")
			if got := statusFollowupsField(t, jsonOut); !slices.Equal(got.Items, want) || got.Error != "" {
				t.Errorf("followups:\n got %+v\nwant items %+v and no error", got, want)
			}
		})
	}
}

// AC4: status decides the follow-ups from the ledger and never runs gh, on
// both paths, text and --json, with a ready and a waiting follow-up. PATH
// holds only a bin dir whose gh logs its calls and fails, and whose git logs
// its calls and runs the real one.
func TestStatusCmd_FollowupsRunNoGH(t *testing.T) {
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
	ghLog := filepath.Join(tmp, "gh.log")
	t.Setenv("STATUS_MG_GIT_LOG", filepath.Join(tmp, "git.log"))
	t.Setenv("STATUS_MG_GH_LOG", ghLog)
	t.Setenv("PATH", bin)

	emptyDir, populatedDir := statusMGStateDir(t), statusMGStateDir(t)
	seedTwoOrgManifest(t, populatedDir)
	for _, dir := range []string{emptyDir, populatedDir} {
		seedStatusEvents(t, dir, org.ManifestEvent{TS: statusFollowupAt(time.Hour), OrgID: "org-m", Event: org.EventPRMerged, Details: "branch=feat/merged pr=1"})
		addStatusFollowup(t, dir, 1, "feat/merged")
		addStatusFollowup(t, dir, 2, "feat/open")
		for _, extra := range [][]string{nil, {"--json"}} {
			out, _ := runStatusSplit(t, append([]string{"--state-dir", dir}, extra...)...)
			if extra == nil {
				if rows := statusFollowupRows(t, out); len(rows) != 2 || rows[0][3] != "ready" || rows[1][3] != "waiting" {
					t.Errorf("follow-up rows = %q, want f1 ready and f2 waiting:\n%s", rows, out)
				}
			}
		}
	}
	if _, err := os.Stat(ghLog); !os.IsNotExist(err) {
		t.Errorf("status ran gh: %q (stat: %v)", readLogLines(t, ghLog), err)
	}
}

// AC4 and AC5 of the help: the reserved line, the follow-up section, that
// ready comes from the ledger's records of merges after the follow-up with
// no gh, the unknown state, the followups object of --json, and that `ralph
// org merge-order` asks GitHub.
func TestStatusCmd_HelpDescribesReservationsAndFollowups(t *testing.T) {
	out, err := runStatusCmd(t, "--help")
	if err != nil {
		t.Fatalf("status --help: %v", err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"While an org runs and holds a reservation made with --reserve or `ralph org start --plan` (released by `ralph org disband`), " +
			"a `reserved: <path>, ...` line follows its seats, as in `ralph org status`, and --json adds the org's reservation array " +
			"(empty when it holds none). An org that holds a reservation but has no seat is shown too: its org_id line and that line, " +
			"with no seat table.",
		"After the orgs, a `follow-ups:` section lists the open follow-ups `ralph org followup add` recorded in the state directory " +
			"(followups.jsonl), and nothing when there is none: ID, ISSUE, AFTER_BRANCH, STATE, SOURCE, DETAIL.",
		"They belong to the whole ledger, so --org-id does not filter them.",
		"A follow-up is ready when the ledger records a merge of its after-branch (the pr_merged of `ralph org merge`, or the " +
			"worktree_cleaned of the cleanup of a merged PR) at or after the follow-up was recorded, unknown when the time it was " +
			"recorded cannot be read (a hand edit of followups.jsonl), with the reason in DETAIL, and waiting otherwise; a merge " +
			"from before does not count, since a branch name can be used again.",
		"SOURCE is recorded: status reads only the ledger's records of merges and runs no gh, so a PR merged on GitHub with no such " +
			"record yet still shows as waiting; `ralph org merge-order` asks GitHub about the after-branch's PR, with gh.",
		"Damage in the follow-up record is warned about on stderr, and so is a record that cannot be read, which shows no follow-up.",
		"--json has them in followups, an object always there: items, the follow-ups (empty when there is none), each with state, " +
			"source, merged_event, merged_at, and detail; corrupt_lines and ignored_events, the record's damage; and error, why the " +
			"record could not be read, empty when it was read or does not exist.",
		"status runs no gh and does not use the network.",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("status --help does not say %q:\n%s", want, out)
		}
	}
}
