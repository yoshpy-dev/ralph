package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests that every text table and line that shows a seat's or a headless
// director's model shows a model of "" as (CLI default), as the spawn line,
// `ralph org report`, and the director: line already do (self-review L1 of
// plan docs/plans/active/2026-11-05-org-role-models.md, D5): `ralph org
// status`'s table, `ralph status`'s seat table, and `ralph org director
// status`'s model: line. The --json forms keep the model "".

// seedModelDisplaySeats records, in org-a, a seat whose spawn passed no
// --model (model "", marked ModelCLIDefault), a seat of an old manifest
// event with no model field (model "", no mark), and a seat with a model.
func seedModelDisplaySeats(t *testing.T, stateDir string) {
	t.Helper()
	ts := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	seedStatusEvents(t, stateDir,
		org.ManifestEvent{TS: ts, OrgID: "org-a", SeatID: "seat-a", Event: org.EventSpawnStarted, Role: "reviewer", Driver: "claude", ModelCLIDefault: true},
		org.ManifestEvent{TS: ts, OrgID: "org-a", SeatID: "seat-a", Event: org.EventSpawned, Role: "reviewer", Driver: "claude", PaneID: "p-a"},
		org.ManifestEvent{TS: ts, OrgID: "org-a", SeatID: "seat-b", Event: org.EventSpawned, Role: "worker", Driver: "codex", PaneID: "p-b"},
		org.ManifestEvent{TS: ts, OrgID: "org-a", SeatID: "seat-c", Event: org.EventSpawned, Role: "implementer", Driver: "claude", Model: "opus", PaneID: "p-c"},
	)
}

// modelDisplayRows are the three seats' rows of seedModelDisplaySeats, in
// SeatID order: both seats with a model of "" show (CLI default).
var modelDisplayRows = []string{
	"seat-a\treviewer\tclaude\t(CLI default)\tspawned (active)\tp-a",
	"seat-b\tworker\tcodex\t(CLI default)\tspawned (active)\tp-b",
	"seat-c\timplementer\tclaude\topus\tspawned (active)\tp-c",
}

// TestOrgStatus_EmptyModelShowsCLIDefault: `ralph org status`'s table shows
// a model of "" as (CLI default), for a seat spawned without --model and for
// a seat of an old event with no model alike, and --json keeps the model "".
func TestOrgStatus_EmptyModelShowsCLIDefault(t *testing.T) {
	stateDir := t.TempDir()
	seedModelDisplaySeats(t, stateDir)

	stdout, _, err := runOrgCmdStreams(t, "status", "--org-id", "org-a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("org status: %v", err)
	}
	want := "SEAT_ID\tROLE\tDRIVER\tMODEL\tSTATE\tPANE_ID\n" + strings.Join(modelDisplayRows, "\n") + "\n"
	if stdout != want {
		t.Errorf("org status =\n%q\nwant\n%q", stdout, want)
	}

	jsonOut, _, err := runOrgCmdStreams(t, "status", "--org-id", "org-a", "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("org status --json: %v", err)
	}
	var payload struct {
		Seats []orgSeatJSON `json:"seats"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("org status --json: %v\n%s", err, jsonOut)
	}
	assertModelDisplayJSON(t, jsonOut, len(payload.Seats), func(i int) (string, string) { return payload.Seats[i].SeatID, payload.Seats[i].Model })
}

// TestStatus_SeatTableShowsCLIDefaultForEmptyModel: `ralph status`'s seat
// table shows a model of "" as (CLI default), and --json keeps it "".
func TestStatus_SeatTableShowsCLIDefaultForEmptyModel(t *testing.T) {
	stateDir := t.TempDir()
	seedModelDisplaySeats(t, stateDir)

	out, _ := runStatusSplit(t, "--state-dir", stateDir)
	want := "org_id: org-a (active 3/3)\n  SEAT_ID\tROLE\tDRIVER\tMODEL\tSTATE\tPANE_ID\n  " + strings.Join(modelDisplayRows, "\n  ") + "\n"
	if got := statusGroupsText(t, out); got != want {
		t.Errorf("status groups =\n%q\nwant\n%q", got, want)
	}

	jsonOut, _ := runStatusSplit(t, "--state-dir", stateDir, "--json")
	var payload struct {
		Orgs []statusOrgJSON `json:"orgs"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("status --json: %v\n%s", err, jsonOut)
	}
	if len(payload.Orgs) != 1 {
		t.Fatalf("status --json orgs = %+v, want org-a only", payload.Orgs)
	}
	seats := payload.Orgs[0].Seats
	assertModelDisplayJSON(t, jsonOut, len(seats), func(i int) (string, string) { return seats[i].SeatID, seats[i].Model })
}

// assertModelDisplayJSON checks that a --json form lists the three seats of
// seedModelDisplaySeats with their models as recorded ("" for seat-a and
// seat-b) and never the text form's (CLI default).
func assertModelDisplayJSON(t *testing.T, jsonOut string, n int, seat func(i int) (id, model string)) {
	t.Helper()
	want := [][2]string{{"seat-a", ""}, {"seat-b", ""}, {"seat-c", "opus"}}
	if n != len(want) {
		t.Fatalf("--json lists %d seats, want %d:\n%s", n, len(want), jsonOut)
	}
	for i, w := range want {
		if id, model := seat(i); id != w[0] || model != w[1] {
			t.Errorf("--json seat %d = %s with model %q, want %s with model %q", i, id, model, w[0], w[1])
		}
	}
	if strings.Contains(jsonOut, org.CLIDefaultModelText) {
		t.Errorf("--json carries %q; want the model as recorded:\n%s", org.CLIDefaultModelText, jsonOut)
	}
}

// TestOrgDirectorStatus_HeadlessEmptyModelLine: `ralph org director
// status`'s model: line of a headless registration (launch herdr or cmd)
// whose model is "" reads model: (CLI default) instead of being left out; a
// headless one with a model shows it; and a promoted session, which has no
// model, still has no model: line.
func TestOrgDirectorStatus_HeadlessEmptyModelLine(t *testing.T) {
	for _, tc := range []struct {
		name      string
		launch    string
		model     string
		wantLine  string // "" means no model: line
		wantAfter string // the line right before the model: line
	}{
		{"herdr, model empty", org.DirectorLaunchHerdr, "", "model: (CLI default)", "state: running"},
		{"cmd, model empty", org.DirectorLaunchCmd, "", "model: (CLI default)", "state: running"},
		{"herdr, a model", org.DirectorLaunchHerdr, "opus", "model: opus", "state: running"},
		{"a promoted session", "", "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reg := org.DirectorRegistration{ID: "d1", Driver: "claude", Wake: []string{org.DirectorWakePull}, Model: tc.model}
			if tc.launch != "" {
				reg.Launch, reg.State, reg.Token, reg.PermissionMode = tc.launch, org.DirectorStateRunning, "0123456789ab", org.PermissionModeAutonomous
			}
			cur := newOrgDirectorReportRegistration(reg)
			report := orgDirectorReport{Enabled: true, State: orgDirectorStateAttached, Registration: &cur}
			var buf bytes.Buffer
			printOrgDirectorReport(&buf, report, "")
			lines := strings.Split(buf.String(), "\n")
			var modelLines []int
			for i, l := range lines {
				if strings.HasPrefix(l, "model:") {
					modelLines = append(modelLines, i)
				}
			}
			if tc.wantLine == "" {
				if len(modelLines) != 0 {
					t.Fatalf("director status has a model: line for a promoted session:\n%s", buf.String())
				}
				return
			}
			if len(modelLines) != 1 || lines[modelLines[0]] != tc.wantLine || modelLines[0] == 0 || lines[modelLines[0]-1] != tc.wantAfter {
				t.Fatalf("director status =\n%s\nwant one %q line right after %q", buf.String(), tc.wantLine, tc.wantAfter)
			}
		})
	}
}
