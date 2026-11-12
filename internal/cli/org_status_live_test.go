package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org status --live` through the CLI (plan
// docs/plans/active/2026-11-12-send-screen-fallback.md, D6 and AC6; issue
// #308). herdr is the resume herdr of stubResumeHerdr (org_resume_test.go),
// which answers `herdr agent get` from a status file and `herdr pane read`
// from a screen file per pane; the screens are the real ones of
// internal/org/testdata/resume/. The reads themselves are tested in
// internal/org/status_live_test.go; these pin the columns, the mismatch
// line, the JSON, and that without --live nothing changes.

// liveStatusScreen is the text of the fixture name of
// internal/org/testdata/resume/.
func liveStatusScreen(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "org", "testdata", "resume", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}

// seedLiveStatusOrg seeds org-a with a fixed time stamp on each event: the
// implementer (claude) that herdr reports working while its screen waits for
// input, the leader (claude) working on a busy screen, the reviewer (codex)
// done on an idle screen, and a stopped seat. s answers herdr for all four,
// the stopped seat included (working on an idle screen), so a read of it
// would show in the output.
func seedLiveStatusOrg(t *testing.T, s resumeStub, stateDir string) {
	t.Helper()
	store := org.NewManifestStoreAtPath(org.ManifestPathIn(stateDir))
	for _, ev := range []org.ManifestEvent{
		{OrgID: "org-a", SeatID: "implementer", Event: org.EventSpawned, Role: "implementer", Driver: "claude", Model: "opus", PaneID: "pane-i", HerdrAgentName: "org-a_implementer"},
		{OrgID: "org-a", SeatID: "leader", Event: org.EventSpawned, Role: "leader", Driver: "claude", Model: "opus", PaneID: "pane-l", HerdrAgentName: "org-a_leader"},
		{OrgID: "org-a", SeatID: "reviewer", Event: org.EventSpawned, Role: "reviewer", Driver: "codex", Model: "gpt-6-astra", PaneID: "pane-r", HerdrAgentName: "org-a_reviewer"},
		{OrgID: "org-a", SeatID: "old", Event: org.EventSpawned, Role: "implementer", Driver: "claude", Model: "opus", PaneID: "pane-o", HerdrAgentName: "org-a_old"},
		{OrgID: "org-a", SeatID: "old", Event: org.EventStopped, Role: "implementer", Driver: "claude", Model: "opus", PaneID: "pane-o"},
	} {
		ev.TS = "2026-11-12T00:00:00Z"
		if err := store.Append(ev); err != nil {
			t.Fatalf("seed %+v: %v", ev, err)
		}
	}
	for _, seat := range []struct{ name, status, pane, screen string }{
		{"org-a_implementer", "working", "pane-i", "claude-idle-after-reply.txt"},
		{"org-a_leader", "working", "pane-l", "claude-working-tool.txt"},
		{"org-a_reviewer", "done", "pane-r", "codex-idle-after-reply.txt"},
		{"org-a_old", "working", "pane-o", "claude-idle-after-reply.txt"},
	} {
		s.agent(t, seat.name, seat.status)
		screen := liveStatusScreen(t, seat.screen)
		s.screen(t, seat.pane, screen, screen)
	}
}

// liveStatusMismatchLine is the line status --live prints after the table
// for the implementer of seedLiveStatusOrg.
const liveStatusMismatchLine = "mismatch: seat implementer: herdr says working but the screen waits for input; " +
	"see the rule behind herdr's state with `herdr agent explain org-a_implementer` " +
	"(ralph org send types into such a seat after its screen check)\n"

// Without --live, status reads the ledger only: the table and the JSON are
// the bytes they were before --live existed, and herdr is not called, even
// with a herdr on PATH that would answer.
func TestOrgStatus_WithoutLiveTheOutputIsUnchangedAndHerdrIsNotCalled(t *testing.T) {
	herdrLog, s := stubResumeHerdr(t)
	stateDir := t.TempDir()
	seedLiveStatusOrg(t, s, stateDir)

	stdout, stderr, err := runOrgCmdStreams(t, "status", "--org-id", "org-a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status: %v (stderr %s)", err, stderr)
	}
	wantTable := "SEAT_ID\tROLE\tDRIVER\tMODEL\tSTATE\tPANE_ID\n" +
		"implementer\timplementer\tclaude\topus\tspawned (active)\tpane-i\n" +
		"leader\tleader\tclaude\topus\tspawned (active)\tpane-l\n" +
		"old\timplementer\tclaude\topus\tstopped\tpane-o\n" +
		"reviewer\treviewer\tcodex\tgpt-6-astra\tspawned (active)\tpane-r\n"
	if stdout != wantTable || stderr != "" {
		t.Errorf("stdout =\n%q\nwant\n%q\nstderr %q", stdout, wantTable, stderr)
	}

	stdout, stderr, err = runOrgCmdStreams(t, "status", "--org-id", "org-a", "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (stderr %s)", err, stderr)
	}
	seat := func(id, role, driver, model, pane, event string, active bool) string {
		return "    {\n" +
			`      "org_id": "org-a",` + "\n" +
			`      "seat_id": "` + id + `",` + "\n" +
			`      "role": "` + role + `",` + "\n" +
			`      "driver": "` + driver + `",` + "\n" +
			`      "model": "` + model + `",` + "\n" +
			`      "pane_id": "` + pane + `",` + "\n" +
			`      "event": "` + event + `",` + "\n" +
			`      "active": ` + map[bool]string{true: "true", false: "false"}[active] + ",\n" +
			`      "ts": "2026-11-12T00:00:00Z"` + "\n" +
			"    }"
	}
	wantJSON := "{\n  \"seats\": [\n" +
		seat("implementer", "implementer", "claude", "opus", "pane-i", "spawned", true) + ",\n" +
		seat("leader", "leader", "claude", "opus", "pane-l", "spawned", true) + ",\n" +
		seat("old", "implementer", "claude", "opus", "pane-o", "stopped", false) + ",\n" +
		seat("reviewer", "reviewer", "codex", "gpt-6-astra", "pane-r", "spawned", true) + "\n" +
		"  ],\n  \"corrupt_lines\": 0\n}\n"
	if stdout != wantJSON || stderr != "" {
		t.Errorf("stdout =\n%s\nwant\n%s\nstderr %q", stdout, wantJSON, stderr)
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %v, want none without --live", lines)
	}
}

// With --live the table gains a HERDR and a SCREEN column: herdr's state and
// the screen's kind, reason, and line, `-` for a seat that is not active.
// After the table, one line per seat herdr reports working while the screen
// waits for input points at `herdr agent explain`. herdr is asked only
// `agent get` and `pane read` of the active seats, nothing that could start
// a server, and the command exits 0.
func TestOrgStatusLive_TableShowsHerdrAndTheScreenAndTheMismatch(t *testing.T) {
	herdrLog, s := stubResumeHerdr(t)
	stateDir := t.TempDir()
	seedLiveStatusOrg(t, s, stateDir)

	stdout, stderr, err := runOrgCmdStreams(t, "status", "--org-id", "org-a", "--state-dir", stateDir, "--live")
	if err != nil {
		t.Fatalf("status --live: %v (stderr %s)", err, stderr)
	}
	want := "SEAT_ID\tROLE\tDRIVER\tMODEL\tSTATE\tPANE_ID\tHERDR\tSCREEN\n" +
		"implementer\timplementer\tclaude\topus\tspawned (active)\tpane-i\tworking\t" +
		`idle (the turn ended: "✻ Cooked for 3s · done 6:09 PM")` + "\n" +
		"leader\tleader\tclaude\topus\tspawned (active)\tpane-l\tworking\t" +
		`busy (a busy line is on the screen: "✶ Deliberating… (6s · ↓ 100 tokens)")` + "\n" +
		"old\timplementer\tclaude\topus\tstopped\tpane-o\t-\t-\n" +
		"reviewer\treviewer\tcodex\tgpt-6-astra\tspawned (active)\tpane-r\tdone\t" +
		`idle (the turn ended: "Worked for 4s • 18:17")` + "\n" +
		liveStatusMismatchLine
	if stdout != want || stderr != "" {
		t.Errorf("stdout =\n%s\nwant\n%s\nstderr %q", stdout, want, stderr)
	}
	wantCalls := []string{"agent get org-a_implementer", "pane read pane-i", "agent get org-a_leader", "pane read pane-l", "agent get org-a_reviewer", "pane read pane-r"}
	got := readLogLines(t, herdrLog)
	if len(got) != len(wantCalls) {
		t.Fatalf("herdr calls = %v, want %v (in that order, with pane read's flags)", got, wantCalls)
	}
	for i, w := range wantCalls {
		if !strings.HasPrefix(got[i], w+" ") && got[i] != w {
			t.Errorf("herdr call %d = %q, want %q", i, got[i], w)
		}
	}
}

// With --live --json each active seat gets a `live` object, and a seat that
// is not active none.
func TestOrgStatusLive_JSONAddsALiveObjectPerActiveSeat(t *testing.T) {
	_, s := stubResumeHerdr(t)
	stateDir := t.TempDir()
	seedLiveStatusOrg(t, s, stateDir)

	stdout, stderr, err := runOrgCmdStreams(t, "status", "--org-id", "org-a", "--state-dir", stateDir, "--live", "--json")
	if err != nil {
		t.Fatalf("status --live --json: %v (stderr %s)", err, stderr)
	}
	var payload struct {
		Seats []struct {
			SeatID string                     `json:"seat_id"`
			Live   map[string]json.RawMessage `json:"live"`
		} `json:"seats"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode %s: %v", stdout, err)
	}
	want := map[string]string{
		"implementer": `{"agent":"org-a_implementer","herdr_state":"working","screen":"idle","screen_reason":"the turn ended","screen_line":"✻ Cooked for 3s · done 6:09 PM","mismatch":true}`,
		"leader":      `{"agent":"org-a_leader","herdr_state":"working","screen":"busy","screen_reason":"a busy line is on the screen","screen_line":"✶ Deliberating… (6s · ↓ 100 tokens)","mismatch":false}`,
		"old":         "",
		"reviewer":    `{"agent":"org-a_reviewer","herdr_state":"done","screen":"idle","screen_reason":"the turn ended","screen_line":"Worked for 4s • 18:17","mismatch":false}`,
	}
	if len(payload.Seats) != len(want) {
		t.Fatalf("seats = %d, want %d: %s", len(payload.Seats), len(want), stdout)
	}
	for _, seat := range payload.Seats {
		got := ""
		if seat.Live != nil {
			got = liveStatusCompactJSON(t, seat.Live)
		}
		if got != want[seat.SeatID] {
			t.Errorf("seat %s: live =\n%s\nwant\n%s", seat.SeatID, got, want[seat.SeatID])
		}
	}
}

// liveStatusCompactJSON is m's JSON object, compact, keys in the order of
// the wire shape.
func liveStatusCompactJSON(t *testing.T, m map[string]json.RawMessage) string {
	t.Helper()
	var parts []string
	for _, key := range []string{"agent", "herdr_state", "herdr_error", "screen", "screen_reason", "screen_line", "mismatch"} {
		if v, ok := m[key]; ok {
			parts = append(parts, `"`+key+`":`+string(v))
			delete(m, key)
		}
	}
	if len(m) != 0 {
		t.Errorf("live has keys outside the wire shape: %v", m)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// With no herdr reachable (none on PATH), every active seat says why its
// state and screen are unknown, no mismatch is printed, and the command
// still exits 0.
func TestOrgStatusLive_NoHerdrReachableStillExitsZero(t *testing.T) {
	stateDir := t.TempDir()
	seedLiveStatusOrg(t, resumeStub{dir: t.TempDir()}, stateDir)
	t.Setenv("PATH", "")

	stdout, stderr, err := runOrgCmdStreams(t, "status", "--org-id", "org-a", "--state-dir", stateDir, "--live")
	if err != nil {
		t.Fatalf("status --live without herdr: %v (stdout %s stderr %s)", err, stdout, stderr)
	}
	lines := outputLines(stdout)
	if len(lines) != 5 {
		t.Fatalf("stdout has %d lines, want the header and 4 seats:\n%s", len(lines), stdout)
	}
	for _, l := range lines[1:] {
		cols := strings.Split(l, "\t")
		if len(cols) != 8 {
			t.Fatalf("row %q has %d columns, want 8", l, len(cols))
		}
		if cols[0] == "old" {
			continue
		}
		if !strings.HasPrefix(cols[6], "unknown (herdr agent get org-a_"+cols[0]+" failed: ") ||
			!strings.HasPrefix(cols[7], "unreadable (herdr pane read "+cols[5]+" failed: ") {
			t.Errorf("row %q: want herdr unknown and the screen unreadable, each with its failure", l)
		}
	}
	if strings.Contains(stdout, "mismatch:") {
		t.Errorf("stdout has a mismatch line with no herdr reachable:\n%s", stdout)
	}
}

// The help of `ralph org status` says what --live reads and that the
// default reads the ledger only.
func TestOrgStatus_HelpDescribesLive(t *testing.T) {
	out, err := runOrgCmd(t, "status", "--help")
	if err != nil {
		t.Fatalf("status --help: %v", err)
	}
	got := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"--live also asks herdr for each active seat's agent_status and reads its pane",
		"Without --live the command reads the ledger only.",
		"--live also read herdr's state and each active seat's screen",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("status --help lacks %q:\n%s", want, out)
		}
	}
}
