package cli

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for what the director: line of `ralph status` and `ralph org
// director status`, and both --json outputs, say of the watch's check of a
// headless director (plan docs/plans/active/2026-10-18-org-director-liveness.md,
// D8 and the display half of AC7). The registration and the watch's mark
// (director-alive.json) are written into a t.TempDir() ledger through
// org.DirectorStore or by hand; no herdr, ps, or watch runs.

// directorCheckAt is the checked_at of the marks these tests write.
const directorCheckAt = "2026-10-18T09:30:00Z"

// directorReportCheckKeys returns the keys of the check object of director,
// the director object of a --json output, sorted, or nil when it has none.
func directorReportCheckKeys(t *testing.T, director []byte) []string {
	t.Helper()
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(director, &obj); err != nil {
		t.Fatalf("decode the director object: %v\n%s", err, director)
	}
	raw, ok := obj["check"]
	if !ok {
		return nil
	}
	var check map[string]json.RawMessage
	if err := json.Unmarshal(raw, &check); err != nil {
		t.Fatalf("decode director.check: %v\n%s", err, raw)
	}
	return slices.Sorted(maps.Keys(check))
}

// directorCheckOutputs is the director: line and the director object of
// --json, from `ralph status` and `ralph org director status` on stateDir.
type directorCheckOutputs struct {
	statusLine, directorLine           string
	statusJSON, directorJSON           orgDirectorReport
	statusCheckKeys, directorCheckKeys []string
	statusOut, directorOut             string
}

// readDirectorCheckOutputs runs both commands, text and --json, with
// --state-dir stateDir. Each must exit 0, print exactly one director: line,
// and print no raw control character.
func readDirectorCheckOutputs(t *testing.T, stateDir string) directorCheckOutputs {
	t.Helper()
	var o directorCheckOutputs
	out, err := runStatusCmd(t, "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	o.statusOut, o.statusLine = out, statusDirectorLine(t, out)
	o.directorOut, _ = runInboxOK(t, "director", "status", "--state-dir", stateDir)
	o.directorLine = statusDirectorLine(t, o.directorOut)
	for name, text := range map[string]string{"status": o.statusOut, "director status": o.directorOut} {
		if strings.ContainsAny(text, "\x1b\r") {
			t.Errorf("%s output holds a raw control character: %q", name, text)
		}
	}

	jsonOut, err := runStatusCmd(t, "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
	}
	o.statusJSON = statusDirectorJSONOf(t, jsonOut)
	var payload struct {
		Director json.RawMessage `json:"director"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("status --json: %v\n%s", err, jsonOut)
	}
	o.statusCheckKeys = directorReportCheckKeys(t, payload.Director)
	jsonOut, _ = runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
	o.directorJSON = directorReportOf(t, jsonOut)
	o.directorCheckKeys = directorReportCheckKeys(t, []byte(jsonOut))
	return o
}

// wantCheckIn fails t unless both --json outputs carry check exactly as want
// (nil: no check key at all), with exactly the keys state, how, checked_at,
// and detail when they carry one.
func (o directorCheckOutputs) wantCheckIn(t *testing.T, want *orgDirectorReportCheck) {
	t.Helper()
	wantKeys := []string{"checked_at", "detail", "how", "state"}
	if want == nil {
		wantKeys = nil
	}
	for name, got := range map[string]orgDirectorReport{"status --json director": o.statusJSON, "director status --json": o.directorJSON} {
		if (got.Check == nil) != (want == nil) || (want != nil && *got.Check != *want) {
			t.Errorf("%s: check = %+v; want %+v", name, got.Check, want)
		}
	}
	for name, keys := range map[string][]string{"status --json director": o.statusCheckKeys, "director status --json": o.directorCheckKeys} {
		if !slices.Equal(keys, wantKeys) {
			t.Errorf("%s: check keys = %q; want %q", name, keys, wantKeys)
		}
	}
}

// D8 for both commands: a headless registration, running in herdr or
// starting with launch cmd, shows on the director: line what the watch's mark
// says of it when the mark is its own (the same id and attached_at), in place
// of the sentence that the registration does not tell whether its session is
// still running: alive, ended with the two ways out (start, which cleans it up
// first, and stop, both with --state-dir), and unknown with its detail; --json
// carries the same as check. With no mark, a mark of another id or of another
// attached_at, or a mark that cannot be read (it does not parse, or it is a
// directory), the line says not checked yet (with why, for the mark that
// cannot be read, and the status still exits 0) and --json has no check.
// Values from the mark are escaped on the line and kept as read in the JSON.
func TestDirectorLine_ShowsTheWatchsCheckOfAHeadlessDirector(t *testing.T) {
	markOf := func(state, how, detail string, edit func(*org.DirectorAlive)) func(*testing.T, string, org.DirectorRegistration) {
		return func(t *testing.T, stateDir string, reg org.DirectorRegistration) {
			t.Helper()
			m := org.DirectorAlive{ID: reg.ID, AttachedAt: reg.AttachedAt, State: state, How: how, CheckedAt: directorCheckAt, Detail: detail}
			if edit != nil {
				edit(&m)
			}
			writeDirectorAliveMark(t, stateDir, m)
		}
	}
	notChecked := func(string) string { return directorNotCheckedClause }
	checked := func(text string) func(string) string {
		return func(string) string { return "checked by the watch at " + directorCheckAt + ": " + text }
	}
	endedWithWaysOut := func(how string) func(string) string {
		return func(stateDir string) string {
			suffix := " --state-dir '" + stateDir + "'"
			return "checked by the watch at " + directorCheckAt + ": ended (" + how + "); as it has ended, ralph org director start --driver <claude|codex>" + suffix +
				" cleans it up and then starts a new headless director, or ralph org director stop --id d1" + suffix + " stops it"
		}
	}
	// unreadable is the clause of a mark that org.ReadDirectorAlive fails to
	// read or parse (what), with its error; the error holds only the mark's
	// path and the reason, neither with a control character.
	unreadable := func(what string) func(string) string {
		return func(stateDir string) string {
			_, err := org.ReadDirectorAlive(stateDir)
			if err == nil || !strings.HasPrefix(err.Error(), "org: director: "+what+" "+org.DirectorAlivePathIn(stateDir)+": ") {
				return "<the mark was meant to fail to " + what + ", and ReadDirectorAlive returned " + fmt.Sprint(err) + ">"
			}
			return directorNotCheckedClause + ", but its mark cannot be read (" + err.Error() + ")"
		}
	}
	for _, tc := range []struct {
		name   string
		cmd    bool // a launch cmd still starting, else a launch herdr running
		write  func(t *testing.T, stateDir string, reg org.DirectorRegistration)
		clause func(stateDir string) string
		check  *orgDirectorReportCheck
	}{
		{"no mark", false, func(*testing.T, string, org.DirectorRegistration) {}, notChecked, nil},
		{"a mark of another id", false, markOf(org.DirectorCheckEnded, org.DirectorCheckHowAgentEnded, "", func(m *org.DirectorAlive) { m.ID = "d2" }), notChecked, nil},
		{"a mark of the same id and another attached_at", false,
			markOf(org.DirectorCheckEnded, org.DirectorCheckHowAgentEnded, "", func(m *org.DirectorAlive) { m.AttachedAt = "2026-10-01T00:00:00Z" }), notChecked, nil},
		{"alive", false, markOf(org.DirectorCheckAlive, org.DirectorCheckHowAgentFound, "", nil), checked("alive (agent_found)"),
			&orgDirectorReportCheck{State: "alive", How: "agent_found", CheckedAt: directorCheckAt}},
		{"ended", false, markOf(org.DirectorCheckEnded, org.DirectorCheckHowAgentEnded, "", nil), endedWithWaysOut("agent_ended"),
			&orgDirectorReportCheck{State: "ended", How: "agent_ended", CheckedAt: directorCheckAt}},
		{"unknown with a detail", false, markOf(org.DirectorCheckUnknown, org.DirectorCheckHowAgentGetFailed, "herdr agent get director_x failed: no server", nil),
			checked("unknown (agent_get_failed: herdr agent get director_x failed: no server)"),
			&orgDirectorReportCheck{State: "unknown", How: "agent_get_failed", CheckedAt: directorCheckAt, Detail: "herdr agent get director_x failed: no server"}},
		{"a hand-edited mark with control characters", false,
			markOf(org.DirectorCheckUnknown, "agent_get_failed\x1b[31m", "bad\x1b[2J\ndirector: forged", func(m *org.DirectorAlive) { m.CheckedAt += "\r" }),
			func(string) string {
				return `checked by the watch at ` + directorCheckAt + `\r: unknown (agent_get_failed\x1b[31m: bad\x1b[2J\ndirector: forged)`
			},
			&orgDirectorReportCheck{State: "unknown", How: "agent_get_failed\x1b[31m", CheckedAt: directorCheckAt + "\r", Detail: "bad\x1b[2J\ndirector: forged"}},
		{"a mark that does not parse", false, func(t *testing.T, stateDir string, _ org.DirectorRegistration) {
			writeTestFile(t, org.DirectorAlivePathIn(stateDir), `{"id": "d1", "state": `)
		}, unreadable("parse"), nil},
		{"a mark that is a directory", false, func(t *testing.T, stateDir string, _ org.DirectorRegistration) {
			if err := os.Mkdir(org.DirectorAlivePathIn(stateDir), 0o755); err != nil {
				t.Fatal(err)
			}
		}, unreadable("read"), nil},
		{"a launch cmd still starting, no mark", true, func(*testing.T, string, org.DirectorRegistration) {}, notChecked, nil},
		{"a launch cmd whose start ended", true, markOf(org.DirectorCheckEnded, org.DirectorCheckHowStartEnded, "", nil), endedWithWaysOut("start_ended"),
			&orgDirectorReportCheck{State: "ended", How: "start_ended", CheckedAt: directorCheckAt}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			stateDir := t.TempDir()
			var reg org.DirectorRegistration
			var head string
			if tc.cmd {
				reg = seedCmdHeadless(t, stateDir)
				head = "director: attached d1 (driver=claude, headless starting, model=opus, launch cmd, name director_" + reg.Token +
					", attached at " + reg.AttachedAt + ", by alice) -- "
			} else {
				reg = seedHeadless(t, stateDir, "codex", "gpt-6-astra", "w-1", "p-1", true)
				head = "director: attached d1 (driver=codex, headless running, model=gpt-6-astra, pane p-1, workspace w-1, attached at " + reg.AttachedAt + ", by alice) -- "
			}
			tc.write(t, stateDir, reg)

			o := readDirectorCheckOutputs(t, stateDir)
			want := head + tc.clause(stateDir) + "; inbox items it does not ack reach the human at the ack deadline" + directorAutoMergeOnAttached
			if o.statusLine != want {
				t.Errorf("status director line:\n got %q\nwant %q", o.statusLine, want)
			}
			if o.directorLine != want {
				t.Errorf("director status line:\n got %q\nwant %q", o.directorLine, want)
			}
			if strings.Contains(o.statusLine, "does not tell whether its session is still running") {
				t.Errorf("the line of a headless director still says the registration does not tell: %q", o.statusLine)
			}
			o.wantCheckIn(t, tc.check)
		})
	}
}

// D8: a promoted session's registration (attach) is never checked. With a
// mark that has its id and attached_at, ended, both lines are the ones of
// stage 8a-1 with only the clause stage 8b-2b added for a registration with
// no wait --inbox --director mark (directorNoWaitMarkClause), and neither
// --json has a check key.
func TestDirectorLine_APromotedSessionIsNotChecked(t *testing.T) {
	t.Chdir(t.TempDir())
	stateDir := t.TempDir()
	store := org.NewDirectorStore(stateDir)
	store.Now = func() time.Time { return time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC) }
	reg, _, err := store.Attach(org.DirectorAttach{Driver: org.DirectorDriverClaude, User: "alice", Cwd: "/work"})
	if err != nil {
		t.Fatal(err)
	}
	writeDirectorAliveMark(t, stateDir, org.DirectorAlive{ID: reg.ID, AttachedAt: reg.AttachedAt, State: org.DirectorCheckEnded,
		How: org.DirectorCheckHowAgentEnded, CheckedAt: directorCheckAt})

	o := readDirectorCheckOutputs(t, stateDir)
	want := directorAttachedLine("d1", "claude", reg.AttachedAt, "alice")
	if o.statusLine != want || o.directorLine != want {
		t.Errorf("director lines:\n status          %q\n director status %q\nwant %q", o.statusLine, o.directorLine, want)
	}
	o.wantCheckIn(t, nil)
}

// D8, edge cases of the registration: one the watch does not check, a
// headless registration recorded stopped while it is still current (a hand
// edit, as stop writes detached and stopped in one write), keeps the
// sentence that the registration does not tell, with no check, even with a
// mark of its own. With [org.director].enabled false a headless registration
// that is still recorded shows the check after the stop command, and an
// ended one names no start, which is refused while the switch is off.
func TestDirectorLine_EdgeCasesOfTheCheck(t *testing.T) {
	t.Run("a headless registration recorded stopped while current", func(t *testing.T) {
		t.Chdir(t.TempDir())
		stateDir := t.TempDir()
		const ts = "2026-10-18T08:00:00Z"
		ev := func(kind string, set func(*org.DirectorEvent)) string {
			e := org.DirectorEvent{TS: ts, ID: "d1", Event: kind, User: "u", Cwd: "/w"}
			if set != nil {
				set(&e)
			}
			return directorEventLine(t, e)
		}
		writeDirectorLines(t, stateDir,
			ev(org.DirectorEventAttached, func(e *org.DirectorEvent) {
				e.Driver, e.Wake, e.Launch, e.Token, e.Model, e.PermissionMode = "codex", []string{"pull"}, org.DirectorLaunchHerdr, "0123456789ab", "m", org.PermissionModeGuarded
			}),
			ev(org.DirectorEventWorkspaceCreated, func(e *org.DirectorEvent) { e.WorkspaceID = "w-1" }),
			ev(org.DirectorEventPaneCreated, func(e *org.DirectorEvent) { e.PaneID = "p-1" }),
			ev(org.DirectorEventLaunched, nil),
			ev(org.DirectorEventStopped, nil),
		)
		writeDirectorAliveMark(t, stateDir, org.DirectorAlive{ID: "d1", AttachedAt: ts, State: org.DirectorCheckEnded,
			How: org.DirectorCheckHowAgentEnded, CheckedAt: directorCheckAt})

		o := readDirectorCheckOutputs(t, stateDir)
		want := "director: attached d1 (driver=codex, headless stopped, model=m, pane p-1, workspace w-1, attached at " + ts + ", by u) -- " +
			"the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline" + directorAutoMergeOnAttached
		if o.statusLine != want || o.directorLine != want {
			t.Errorf("director lines:\n status          %q\n director status %q\nwant %q", o.statusLine, o.directorLine, want)
		}
		o.wantCheckIn(t, nil)
	})

	t.Run("the switch off", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeTestFile(t, "ralph.toml", "[org.director]\nenabled = false\n")
		stateDir := t.TempDir()
		reg := seedHeadless(t, stateDir, "codex", "gpt-6-astra", "w-1", "p-1", true)
		suffix := " --state-dir '" + stateDir + "'"
		head := "director: disabled ([org.director].enabled = false, so [org.director].auto_merge = true has no effect; " +
			"registration d1 is still recorded, headless running, pane p-1, workspace w-1: " +
			"stop it with ralph org director stop --id d1" + suffix + "; "
		const end = ") -- ralph org escalate sends every item to the human at once"

		o := readDirectorCheckOutputs(t, stateDir)
		if want := head + directorNotCheckedClause + end; o.statusLine != want || o.directorLine != want {
			t.Errorf("director lines before a check:\n status          %q\n director status %q\nwant %q", o.statusLine, o.directorLine, want)
		}
		o.wantCheckIn(t, nil)

		writeDirectorAliveMark(t, stateDir, org.DirectorAlive{ID: reg.ID, AttachedAt: reg.AttachedAt, State: org.DirectorCheckEnded,
			How: org.DirectorCheckHowPaneClosed, CheckedAt: directorCheckAt})
		o = readDirectorCheckOutputs(t, stateDir)
		if want := head + "checked by the watch at " + directorCheckAt + ": ended (pane_closed)" + end; o.statusLine != want || o.directorLine != want {
			t.Errorf("director lines of an ended d1:\n status          %q\n director status %q\nwant %q", o.statusLine, o.directorLine, want)
		}
		if o.statusJSON.State != "disabled" || o.directorJSON.State != "disabled" {
			t.Errorf("states = %q, %q; want disabled", o.statusJSON.State, o.directorJSON.State)
		}
		o.wantCheckIn(t, &orgDirectorReportCheck{State: "ended", How: "pane_closed", CheckedAt: directorCheckAt})
	})
}
