package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for what the wait: line of `ralph org director status`, the
// director: line of a promoted session in `ralph status` and `ralph org
// director status`, and both --json outputs say of the mark of the
// director's wait, director-wait.json (plan
// docs/plans/active/2026-10-19-org-director-wait.md, D5 and AC7). The
// registration is recorded in a t.TempDir() ledger through org.DirectorStore
// or the director verbs, and the mark is written by hand, except in the
// end-to-end test, where `ralph org wait --inbox --director` writes it. AC9:
// no herdr, ps, osascript, terminal, gh, or network.

// directorNoWaitMarkClause is what the director: line of a promoted session
// adds when there is no mark of its wait --inbox --director.
const directorNoWaitMarkClause = "; no wait --inbox --director mark for it"

// directorNoWaitMarkLine is the wait: line of `ralph org director status`
// when there is no mark of the current registration's wait.
const directorNoWaitMarkLine = "wait: no wait --inbox --director mark for this registration"

// directorWaitClock is the now of the stale checks in these tests
// (orgDirectorWaitNow); the times of the marks are set from it.
var directorWaitClock = time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)

// directorWaitJSONKeys are the keys of director.wait, sorted.
var directorWaitJSONKeys = []string{"detail", "end", "ended_at", "pid", "seen_at", "started_at", "wait_id", "watch_accepting"}

// directorWaitStaleSuffix is what a running wait's facts add when its mark
// was not rewritten for two of its intervals.
const directorWaitStaleSuffix = ", and has not written it since (it writes every 30s)"

// setDirectorWaitNow makes orgDirectorWaitNow return now until the test ends.
func setDirectorWaitNow(t *testing.T, now time.Time) {
	t.Helper()
	orig := orgDirectorWaitNow
	orgDirectorWaitNow = func() time.Time { return now }
	t.Cleanup(func() { orgDirectorWaitNow = orig })
}

// writeDirectorWaitMark writes m as director-wait.json in stateDir.
func writeDirectorWaitMark(t *testing.T, stateDir string, m org.DirectorWaitMark) {
	t.Helper()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, org.DirectorWaitPathIn(stateDir), string(raw))
}

// directorWaitMarkOf is a running wait's mark for reg (pid 4242, started at
// 09:50 of directorWaitClock's day) last written seenAgo before
// directorWaitClock.
func directorWaitMarkOf(reg org.DirectorRegistration, seenAgo time.Duration) org.DirectorWaitMark {
	return org.DirectorWaitMark{
		WaitID: "4242-1760867400000000000", ID: reg.ID, AttachedAt: reg.AttachedAt, PID: 4242,
		StartedAt: "2026-10-19T09:50:00Z", SeenAt: directorWaitClock.Add(-seenAgo).Format(time.RFC3339), WatchAccepting: true,
	}
}

// reportWaitOf is m as director.wait carries it.
func reportWaitOf(m org.DirectorWaitMark) *orgDirectorReportWait {
	return &orgDirectorReportWait{WaitID: m.WaitID, PID: m.PID, StartedAt: m.StartedAt, SeenAt: m.SeenAt, WatchAccepting: m.WatchAccepting,
		EndedAt: m.EndedAt, End: m.End, Detail: m.Detail}
}

// directorWaitJSON is director.wait of one --json output: whether the key is
// there at all (null included), its keys sorted, and its value decoded.
type directorWaitJSON struct {
	present bool
	keys    []string
	wait    *orgDirectorReportWait
}

// directorWaitJSONOf reads director.wait from director, the director object
// of a --json output.
func directorWaitJSONOf(t *testing.T, director []byte) directorWaitJSON {
	t.Helper()
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(director, &obj); err != nil {
		t.Fatalf("decode the director object: %v\n%s", err, director)
	}
	raw, ok := obj["wait"]
	if !ok {
		return directorWaitJSON{}
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatalf("decode director.wait: %v\n%s", err, raw)
	}
	var w *orgDirectorReportWait
	if err := json.Unmarshal(raw, &w); err != nil {
		t.Fatalf("decode director.wait: %v\n%s", err, raw)
	}
	return directorWaitJSON{present: true, keys: slices.Sorted(maps.Keys(keys)), wait: w}
}

// directorWaitOutputs is what `ralph status` and `ralph org director status`
// print on one ledger, text and --json.
type directorWaitOutputs struct {
	statusOut, directorOut     string
	statusLine, directorLine   string
	waitLines                  []string
	statusJSON, directorJSON   directorWaitJSON
	statusState, directorState string
}

// readDirectorWaitOutputs runs both commands, text and --json, with
// --state-dir stateDir. Each must exit 0 and print exactly one director:
// line and no raw control character.
func readDirectorWaitOutputs(t *testing.T, stateDir string) directorWaitOutputs {
	t.Helper()
	var o directorWaitOutputs
	out, err := runStatusCmd(t, "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status: %v (output: %s)", err, out)
	}
	o.statusOut, o.statusLine = out, statusDirectorLine(t, out)
	o.directorOut, _ = runInboxOK(t, "director", "status", "--state-dir", stateDir)
	o.directorLine = statusDirectorLine(t, o.directorOut)
	for _, line := range strings.Split(o.directorOut, "\n") {
		if strings.HasPrefix(line, "wait:") {
			o.waitLines = append(o.waitLines, line)
		}
	}
	for name, text := range map[string]string{"status": o.statusOut, "director status": o.directorOut} {
		if strings.ContainsAny(text, "\x1b\r\a") {
			t.Errorf("%s output holds a raw control character: %q", name, text)
		}
	}
	if strings.Contains(o.statusOut, "\nwait:") {
		t.Errorf("status prints a wait: line; only director status does:\n%s", o.statusOut)
	}

	jsonOut, err := runStatusCmd(t, "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status --json: %v (output: %s)", err, jsonOut)
	}
	var payload struct {
		Director json.RawMessage `json:"director"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &payload); err != nil {
		t.Fatalf("status --json: %v\n%s", err, jsonOut)
	}
	o.statusJSON, o.statusState = directorWaitJSONOf(t, payload.Director), statusDirectorJSONOf(t, jsonOut).State
	jsonOut, _ = runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
	o.directorJSON, o.directorState = directorWaitJSONOf(t, []byte(jsonOut)), directorReportOf(t, jsonOut).State
	return o
}

// wantWaitLine fails t unless director status printed exactly one wait:
// line, want ("" for none).
func (o directorWaitOutputs) wantWaitLine(t *testing.T, want string) {
	t.Helper()
	var wantLines []string
	if want != "" {
		wantLines = []string{want}
	}
	if !slices.Equal(o.waitLines, wantLines) {
		t.Errorf("director status wait: lines:\n got %q\nwant %q\noutput:\n%s", o.waitLines, wantLines, o.directorOut)
	}
}

// wantWaitIn fails t unless both --json outputs carry wait exactly as want
// (nil: no wait key at all), with exactly directorWaitJSONKeys when they
// carry one.
func (o directorWaitOutputs) wantWaitIn(t *testing.T, want *orgDirectorReportWait) {
	t.Helper()
	for name, got := range map[string]directorWaitJSON{"status --json director": o.statusJSON, "director status --json": o.directorJSON} {
		if want == nil {
			if got.present {
				t.Errorf("%s: has a wait key (%+v); want none", name, got.wait)
			}
			continue
		}
		if !got.present || got.wait == nil || *got.wait != *want {
			t.Errorf("%s: wait = %+v (present %v); want %+v", name, got.wait, got.present, want)
		}
		if !slices.Equal(got.keys, directorWaitJSONKeys) {
			t.Errorf("%s: wait keys = %q; want %q", name, got.keys, directorWaitJSONKeys)
		}
	}
}

// directorWaitReadError is the error org.ReadDirectorWait returns for the
// mark in stateDir, which must start with "org: director: <what> <path>: "
// (what is read or parse); otherwise the returned text names the mismatch,
// so the comparison that uses it fails.
func directorWaitReadError(stateDir, what string) string {
	_, err := org.ReadDirectorWait(stateDir)
	if err == nil || !strings.HasPrefix(err.Error(), "org: director: "+what+" "+org.DirectorWaitPathIn(stateDir)+": ") {
		return "<the mark was meant to fail to " + what + ", and ReadDirectorWait returned " + fmt.Sprint(err) + ">"
	}
	return err.Error()
}

// D5 and AC7 for a promoted session: the wait: line of director status and
// the clause the director: line of both commands adds after "the
// registration does not tell whether its session is still running" say, for
// a mark of the current registration (the same id and attached_at), what it
// wrote: a running wait's pid, start, and last write, with the clause that it
// has not written since when that was more than 60s (two intervals) before
// now and not at 60s, in the future, or for a seen_at that does not parse;
// an ended wait's end, for each end, without its detail. No mark, a mark of
// another id or attached_at, and a mark that cannot be read (not JSON, a
// directory, no permission) say so, and the status exits 0. --json carries
// wait with its eight keys only with a matching mark. Values from the mark
// are escaped on both lines and kept as read in the JSON.
func TestDirectorWaitLine_ShowsTheMarkOfAPromotedSessionsWait(t *testing.T) {
	const (
		head    = "director: attached d1 (driver=claude, attached at 2026-10-19T09:00:00Z, by alice) -- the registration does not tell whether its session is still running"
		tail    = "; inbox items it does not ack reach the human at the ack deadline" + directorAutoMergeOnAttached
		started = "(pid 4242) started at 2026-10-19T09:50:00Z and last wrote its mark at "
	)
	running := func(seenAgo time.Duration) func(*org.DirectorWaitMark) {
		return func(m *org.DirectorWaitMark) {
			*m = directorWaitMarkOf(org.DirectorRegistration{ID: m.ID, AttachedAt: m.AttachedAt}, seenAgo)
		}
	}
	ended := func(end string) func(*org.DirectorWaitMark) {
		return func(m *org.DirectorWaitMark) {
			// A seen_at far behind now: an ended wait is never said to have
			// stopped writing.
			m.SeenAt, m.EndedAt, m.End, m.Detail, m.WatchAccepting = "2026-10-19T08:00:00Z", "2026-10-19T09:59:40Z", end, "the detail of "+end, false
		}
	}
	// facts is a case whose mark matches: the wait: line and the clause
	// both say text.
	facts := func(text string) (string, string) {
		return "wait: the wait --inbox --director of d1 " + text, "; its wait --inbox --director " + text
	}
	type tc struct {
		name string
		// edit changes the running mark directorWaitMarkOf(reg, 30s) before
		// it is written; write writes something else in its place.
		edit       func(*org.DirectorWaitMark)
		write      func(t *testing.T, stateDir string)
		waitLine   func(stateDir string) string
		clause     func(stateDir string) string
		reportWait bool // --json carries the mark as written
	}
	matching := func(name string, edit func(*org.DirectorWaitMark), text string) tc {
		line, clause := facts(text)
		return tc{name: name, edit: edit, waitLine: func(string) string { return line }, clause: func(string) string { return clause }, reportWait: true}
	}
	none := func(name string, edit func(*org.DirectorWaitMark), write func(*testing.T, string)) tc {
		return tc{name: name, edit: edit, write: write,
			waitLine: func(string) string { return directorNoWaitMarkLine }, clause: func(string) string { return directorNoWaitMarkClause }}
	}
	unreadable := func(name, what string, write func(*testing.T, string)) tc {
		cannot := func(stateDir string) string {
			return "director-wait.json cannot be read: " + directorWaitReadError(stateDir, what)
		}
		return tc{name: name, write: write,
			waitLine: func(stateDir string) string { return "wait: " + cannot(stateDir) },
			clause:   func(stateDir string) string { return "; " + cannot(stateDir) }}
	}
	cases := []tc{
		matching("running, written 30s ago", running(30*time.Second), started+"2026-10-19T09:59:30Z"),
		matching("running, written 60s ago: not more than two intervals", running(60*time.Second), started+"2026-10-19T09:59:00Z"),
		matching("running, written 61s ago", running(61*time.Second), started+"2026-10-19T09:58:59Z"+directorWaitStaleSuffix),
		matching("running, written an hour ago", running(time.Hour), started+"2026-10-19T09:00:00Z"+directorWaitStaleSuffix),
		matching("running, a seen_at after now", running(-5*time.Minute), started+"2026-10-19T10:05:00Z"),
		matching("running, a seen_at that does not parse", func(m *org.DirectorWaitMark) { m.SeenAt = "an hour ago" }, started+"an hour ago"),
		matching("running, an empty seen_at", func(m *org.DirectorWaitMark) { m.SeenAt = "" }, started+"-"),
	}
	for _, end := range []string{org.DirectorWaitEndItems, org.DirectorWaitEndTimeout, org.DirectorWaitEndWatch, org.DirectorWaitEndNotCurrent,
		org.DirectorWaitEndError, org.DirectorWaitEndSignal} {
		cases = append(cases, matching("ended "+end, ended(end), "(pid 4242) ended at 2026-10-19T09:59:40Z ("+end+")"))
	}
	cases = append(cases,
		matching("a running mark with control characters",
			func(m *org.DirectorWaitMark) {
				m.StartedAt += "\x1b[2J"
				m.SeenAt = "2026-10-19T09:00:00Z\r\nwait: forged"
			},
			`(pid 4242) started at 2026-10-19T09:50:00Z\x1b[2J and last wrote its mark at 2026-10-19T09:00:00Z\r\nwait: forged`),
		matching("an ended mark with control characters",
			func(m *org.DirectorWaitMark) {
				ended(org.DirectorWaitEndSignal)(m)
				m.EndedAt += "\a"
				m.End = "signal\x1b[31m\ndirector: forged"
			},
			`(pid 4242) ended at 2026-10-19T09:59:40Z\a (signal\x1b[31m\ndirector: forged)`),
		none("no mark", nil, func(*testing.T, string) {}),
		none("a mark of another id", func(m *org.DirectorWaitMark) { m.ID = "d2" }, nil),
		none("a mark of the same id and another attached_at", func(m *org.DirectorWaitMark) { m.AttachedAt = "2026-10-01T00:00:00Z" }, nil),
		unreadable("a mark that is not JSON", "parse", func(t *testing.T, stateDir string) {
			writeTestFile(t, org.DirectorWaitPathIn(stateDir), `{"wait_id": "4242-1", "id": `)
		}),
		unreadable("a mark that is a directory", "read", func(t *testing.T, stateDir string) {
			if err := os.Mkdir(org.DirectorWaitPathIn(stateDir), 0o755); err != nil {
				t.Fatal(err)
			}
		}),
		unreadable("a mark with no read permission", "read", func(t *testing.T, stateDir string) {
			if os.Geteuid() == 0 {
				t.Skip("root reads a file with no read permission")
			}
			path := org.DirectorWaitPathIn(stateDir)
			writeTestFile(t, path, `{}`)
			if err := os.Chmod(path, 0); err != nil {
				t.Fatal(err)
			}
		}),
	)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			setDirectorWaitNow(t, directorWaitClock)
			stateDir := t.TempDir()
			store := org.NewDirectorStore(stateDir)
			store.Now = func() time.Time { return time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC) }
			reg, _, err := store.Attach(org.DirectorAttach{Driver: org.DirectorDriverClaude, User: "alice", Cwd: "/work"})
			if err != nil {
				t.Fatal(err)
			}
			mark := directorWaitMarkOf(reg, 30*time.Second)
			if tc.edit != nil {
				tc.edit(&mark)
			}
			if tc.write != nil {
				tc.write(t, stateDir)
			} else {
				writeDirectorWaitMark(t, stateDir, mark)
			}

			o := readDirectorWaitOutputs(t, stateDir)
			want := head + tc.clause(stateDir) + tail
			if o.statusLine != want {
				t.Errorf("status director line:\n got %q\nwant %q", o.statusLine, want)
			}
			if o.directorLine != want {
				t.Errorf("director status line:\n got %q\nwant %q", o.directorLine, want)
			}
			o.wantWaitLine(t, tc.waitLine(stateDir))
			if strings.Contains(o.statusOut+o.directorOut, "the detail of") {
				t.Errorf("the text form shows the detail, which stays in the JSON:\n%s\n%s", o.statusOut, o.directorOut)
			}
			for _, text := range []string{o.statusLine, o.directorLine} {
				if strings.Contains(text, "alive") || strings.Contains(text, "is waiting") {
					t.Errorf("the line says the director is alive or waiting: %q", text)
				}
			}
			var wantWait *orgDirectorReportWait
			if tc.reportWait {
				wantWait = reportWaitOf(mark)
			}
			o.wantWaitIn(t, wantWait)
		})
	}
}

// D5: a headless registration's director: line says nothing of the wait's
// mark: with a mark of its own, running and stale or ended, the line is the
// one it was with no mark, byte for byte, in both commands, while director
// status prints the wait: line and both --json outputs carry wait. That holds
// for a headless director the watch checks (running in herdr) and for one
// recorded stopped while still current, whose line keeps the sentence that
// the registration does not tell whether its session is still running.
func TestDirectorWaitLine_AHeadlessRegistrationsLineIsUnchanged(t *testing.T) {
	seedStoppedWhileCurrent := func(t *testing.T, stateDir string) org.DirectorRegistration {
		t.Helper()
		const ts = "2026-10-19T08:00:00Z"
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
		return org.DirectorRegistration{ID: "d1", AttachedAt: ts}
	}
	for _, tc := range []struct {
		name string
		seed func(t *testing.T, stateDir string) org.DirectorRegistration
	}{
		{"running in herdr", func(t *testing.T, stateDir string) org.DirectorRegistration {
			return seedHeadless(t, stateDir, "codex", "gpt-6-astra", "w-1", "p-1", true)
		}},
		{"a launch cmd still starting", func(t *testing.T, stateDir string) org.DirectorRegistration { return seedCmdHeadless(t, stateDir) }},
		{"recorded stopped while current", seedStoppedWhileCurrent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			setDirectorWaitNow(t, directorWaitClock)
			stateDir := t.TempDir()
			reg := tc.seed(t, stateDir)
			before := readDirectorWaitOutputs(t, stateDir)
			if before.statusLine != before.directorLine {
				t.Fatalf("the two director: lines differ with no mark:\n status          %q\n director status %q", before.statusLine, before.directorLine)
			}
			before.wantWaitLine(t, directorNoWaitMarkLine)
			before.wantWaitIn(t, nil)
			if strings.Contains(before.statusLine, "wait --inbox --director") {
				t.Errorf("a headless registration's line names the wait with no mark: %q", before.statusLine)
			}

			stale := directorWaitMarkOf(reg, 2*time.Minute)
			endedMark := directorWaitMarkOf(reg, 2*time.Minute)
			endedMark.EndedAt, endedMark.End, endedMark.Detail = "2026-10-19T09:59:40Z", org.DirectorWaitEndTimeout, "no open inbox item within 1000 ms"
			for _, m := range []struct {
				mark     org.DirectorWaitMark
				waitLine string
			}{
				{stale, "wait: the wait --inbox --director of d1 (pid 4242) started at 2026-10-19T09:50:00Z and last wrote its mark at 2026-10-19T09:58:00Z" +
					directorWaitStaleSuffix},
				{endedMark, "wait: the wait --inbox --director of d1 (pid 4242) ended at 2026-10-19T09:59:40Z (timeout)"},
			} {
				writeDirectorWaitMark(t, stateDir, m.mark)
				o := readDirectorWaitOutputs(t, stateDir)
				if o.statusLine != before.statusLine || o.directorLine != before.directorLine {
					t.Errorf("with the mark %+v the director: lines changed:\n status          %q\n director status %q\nwant %q",
						m.mark, o.statusLine, o.directorLine, before.statusLine)
				}
				o.wantWaitLine(t, m.waitLine)
				o.wantWaitIn(t, reportWaitOf(m.mark))
			}
		})
	}
}

// D5, the registration around the mark: with [org.director].enabled false a
// promoted registration that is still recorded keeps the disabled director:
// line as it was, and director status prints its wait: line, with wait in
// both --json outputs; once the registration is detached, no registration is
// current, so a mark it left gets no wait: line and no wait in the JSON. The
// wait: line follows the registration's fields and comes before the
// open_headless lines.
func TestDirectorWaitLine_DisabledNotAttachedAndOpenHeadless(t *testing.T) {
	t.Run("before the open headless lines", func(t *testing.T) {
		t.Chdir(t.TempDir())
		setDirectorWaitNow(t, directorWaitClock)
		stateDir := t.TempDir()
		seedHeadless(t, stateDir, "codex", "gpt-6-astra", "w-1", "p-1", true)
		d2, _, err := org.NewDirectorStore(stateDir).Attach(org.DirectorAttach{Driver: org.DirectorDriverClaude, Replace: true, User: "bob", Cwd: "/w2"})
		if err != nil {
			t.Fatal(err)
		}
		mark := directorWaitMarkOf(d2, 30*time.Second)
		mark.EndedAt, mark.End = "2026-10-19T09:59:40Z", org.DirectorWaitEndItems
		writeDirectorWaitMark(t, stateDir, mark)

		stdout, _ := runInboxOK(t, "director", "status", "--state-dir", stateDir)
		lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
		want := []string{"herdr_pane: no", "wait: the wait --inbox --director of d2 (pid 4242) ended at 2026-10-19T09:59:40Z (items)",
			"open_headless: d1 (running, pane p-1, workspace w-1)"}
		if len(lines) < len(want) || !slices.Equal(lines[len(lines)-len(want):], want) {
			t.Errorf("director status ends with %q; want %q:\n%s", lines[max(0, len(lines)-len(want)):], want, stdout)
		}
	})

	t.Run("the switch off", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeTestFile(t, "ralph.toml", "[org.director]\nenabled = false\n")
		setDirectorWaitNow(t, directorWaitClock)
		stateDir := t.TempDir()
		// attach refuses while the switch is off; the registration was made
		// before it was turned off.
		reg, _, err := org.NewDirectorStore(stateDir).Attach(org.DirectorAttach{Driver: org.DirectorDriverClaude, User: "alice", Cwd: "/work"})
		if err != nil {
			t.Fatal(err)
		}
		const want = "director: disabled ([org.director].enabled = false, so [org.director].auto_merge = true has no effect; registration d1 is still recorded) " +
			"-- ralph org escalate sends every item to the human at once"
		mark := directorWaitMarkOf(reg, 30*time.Second)
		writeDirectorWaitMark(t, stateDir, mark)

		o := readDirectorWaitOutputs(t, stateDir)
		if o.statusLine != want || o.directorLine != want {
			t.Errorf("director lines:\n status          %q\n director status %q\nwant %q", o.statusLine, o.directorLine, want)
		}
		if o.statusState != orgDirectorStateDisabled || o.directorState != orgDirectorStateDisabled {
			t.Errorf("states = %q, %q; want disabled", o.statusState, o.directorState)
		}
		o.wantWaitLine(t, "wait: the wait --inbox --director of d1 (pid 4242) started at 2026-10-19T09:50:00Z and last wrote its mark at 2026-10-19T09:59:30Z")
		o.wantWaitIn(t, reportWaitOf(mark))
	})

	t.Run("not attached", func(t *testing.T) {
		t.Chdir(t.TempDir())
		stateDir := t.TempDir()
		ev := attachD1(t, stateDir)
		writeDirectorWaitMark(t, stateDir, directorWaitMarkOf(org.DirectorRegistration{ID: ev.ID, AttachedAt: ev.TS}, 30*time.Second))
		runInboxOK(t, "director", "detach", "--id", "d1", "--state-dir", stateDir)

		o := readDirectorWaitOutputs(t, stateDir)
		if o.statusLine != directorNotAttachedLine || o.directorLine != directorNotAttachedLine {
			t.Errorf("director lines:\n status          %q\n director status %q\nwant %q", o.statusLine, o.directorLine, directorNotAttachedLine)
		}
		o.wantWaitLine(t, "")
		o.wantWaitIn(t, nil)
	})
}

// D5: the reason a mark cannot be read is escaped on both lines too. The
// error holds the mark's path, and a state dir can be named with any byte,
// so the lines are built here from a report whose error holds control
// characters.
func TestDirectorWaitLine_EscapesTheReadError(t *testing.T) {
	r := orgDirectorReport{
		Registration: &orgDirectorReportRegistration{DirectorRegistration: org.DirectorRegistration{ID: "d1"}},
		waitErr:      "org: director: read /tmp/x\x1b[2J\nwait: forged/director-wait.json: is a directory",
	}
	const escaped = `org: director: read /tmp/x\x1b[2J\nwait: forged/director-wait.json: is a directory`
	if got, want := orgDirectorWaitLine(r), "wait: director-wait.json cannot be read: "+escaped; got != want {
		t.Errorf("wait: line:\n got %q\nwant %q", got, want)
	}
	if got, want := orgDirectorWaitClause(r), "; director-wait.json cannot be read: "+escaped; got != want {
		t.Errorf("director: line clause:\n got %q\nwant %q", got, want)
	}
}

// D5 end to end: a real `ralph org wait --inbox --director d1` writes the
// mark. While it runs, director status says that the wait (this process's
// pid) started at the mark's started_at and last wrote it, with no stale
// clause, and the promoted session's director: line says the same; once the
// context is cancelled, both say it ended (signal), and --json carries the
// mark as written. Then a wait that times out leaves end timeout.
func TestDirectorWaitLine_EndToEndWithARealWait(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("USER", "alice")
	setDirectorWaitIntervals(t)
	stateDir := t.TempDir()
	ev := attachD1(t, stateDir)
	pid := strconv.Itoa(os.Getpid())
	head := "director: attached d1 (driver=claude, attached at " + ev.TS + ", by alice) -- the registration does not tell whether its session is still running"
	const tail = "; inbox items it does not ack reach the human at the ack deadline" + directorAutoMergeOnAttached

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := startOrgCmd(ctx, directorWaitArgs("d1", stateDir, "--timeout-ms", "10000")...)
	waitForDirectorWaitMark(t, stateDir, "the start of the wait", func(m org.DirectorWaitMark) bool { return m.StartedAt != "" && m.End == "" })
	started := readDirectorWaitMark(t, stateDir).StartedAt
	o := readDirectorWaitOutputs(t, stateDir)
	runningFacts := "(pid " + pid + ") started at " + started + " and last wrote its mark at "
	if len(o.waitLines) != 1 || !strings.HasPrefix(o.waitLines[0], "wait: the wait --inbox --director of d1 "+runningFacts) ||
		strings.HasSuffix(o.waitLines[0], directorWaitStaleSuffix) {
		t.Errorf("wait: lines of a running wait = %q; want one starting %q with no stale clause", o.waitLines, "wait: the wait --inbox --director of d1 "+runningFacts)
	}
	if !strings.HasPrefix(o.statusLine, head+"; its wait --inbox --director "+runningFacts) || !strings.HasSuffix(o.statusLine, tail) ||
		strings.Contains(o.statusLine, directorWaitStaleSuffix) {
		t.Errorf("status director line of a running wait:\n got %q\nwant %q ... %q", o.statusLine, head+"; its wait --inbox --director "+runningFacts, tail)
	}
	if !o.statusJSON.present || o.statusJSON.wait.PID != os.Getpid() || o.statusJSON.wait.End != "" || o.statusJSON.wait.StartedAt != started {
		t.Errorf("status --json director.wait of a running wait = %+v; want pid %s, started_at %s, no end", o.statusJSON.wait, pid, started)
	}

	cancel()
	if r := awaitOrgCmd(t, done); r.err == nil {
		t.Fatalf("the cancelled wait returned no error (stdout %q)", r.stdout)
	}
	mark := readDirectorWaitMark(t, stateDir)
	if mark.End != org.DirectorWaitEndSignal {
		t.Fatalf("mark after the cancel = %+v; want end signal", mark)
	}
	o = readDirectorWaitOutputs(t, stateDir)
	endedFacts := "(pid " + pid + ") ended at " + mark.EndedAt + " (signal)"
	if want := head + "; its wait --inbox --director " + endedFacts + tail; o.statusLine != want || o.directorLine != want {
		t.Errorf("director lines after the cancel:\n status          %q\n director status %q\nwant %q", o.statusLine, o.directorLine, want)
	}
	o.wantWaitLine(t, "wait: the wait --inbox --director of d1 "+endedFacts)
	o.wantWaitIn(t, reportWaitOf(mark))

	if _, _, _, err := runOrgCmdStreamsWithin(t, 5*time.Second, directorWaitArgs("d1", stateDir, "--timeout-ms", "100")...); ExitCode(err) != 1 {
		t.Fatalf("the wait with --timeout-ms 100 = %v (exit %d); want exit 1", err, ExitCode(err))
	}
	mark = readDirectorWaitMark(t, stateDir)
	o = readDirectorWaitOutputs(t, stateDir)
	o.wantWaitLine(t, "wait: the wait --inbox --director of d1 (pid "+pid+") ended at "+mark.EndedAt+" (timeout)")
	o.wantWaitIn(t, reportWaitOf(mark))
}

// D5: the help of both commands says what the wait: line and the clause of
// the director: line say, that they come from director-wait.json, which
// `ralph org wait --inbox --director <id>` writes, that they say what the
// wait wrote and not that the director is alive, the 60s clause, and the
// JSON key with its fields, next to the sentence that a promoted session's
// registration does not tell whether its session is still running.
func TestDirectorWaitLine_HelpDescribesTheWait(t *testing.T) {
	directorHelp, _ := runInboxOK(t, "director", "status", "--help")
	statusHelp, err := runStatusCmd(t, "--help")
	if err != nil {
		t.Fatalf("status --help: %v", err)
	}
	for _, c := range []struct {
		name, out string
		want      []string
	}{
		{"director status --help", directorHelp, []string{
			"The registration does not tell whether its session is still running for a promoted session (the /director skill, `ralph org director attach`).",
			"A wait: line follows the fields while a registration is current, headless or not: what director-wait.json, the mark " +
				"`ralph org wait --inbox --director <id>` writes, says of the last such wait for this registration (the same id and attach time): " +
				"that the wait (pid <pid>) started at <time> and last wrote its mark at <time>, adding that it has not written it since " +
				"when that is more than 60s before now (the wait rewrites it every 30s); that it ended at <time> " +
				"(<end>: items, timeout, watch, not_current, error, or signal); that there is no such mark; or why the mark cannot be read.",
			"For a promoted session the director: line adds the same after the sentence that the registration does not tell whether its session is still running.",
			"Both say what the wait wrote, not that the director is waiting or alive.",
			"--json carries the mark as wait (wait_id, pid, started_at, seen_at, watch_accepting, ended_at, end, detail; director.wait in " +
				"`ralph status --json`), left out when there is no mark of this registration or it cannot be read.",
		}},
		{"status --help", statusHelp, []string{
			"A registration does not tell whether its session is still running for a promoted session (the /director skill, `ralph org director attach`).",
			"For a promoted session the line adds, after the sentence that the registration does not tell whether its session is still running, " +
				"what director-wait.json, the mark `ralph org wait --inbox --director <id>` writes, says of the last such wait for the registration " +
				"(the same id and attach time): that it (pid <pid>) started at <time> and last wrote its mark at <time>, adding that it has not written it " +
				"since when that is more than 60s before now (the wait rewrites it every 30s); that it ended at <time> (<end>); " +
				"that there is no such mark; or why the mark cannot be read.",
			"It says what the wait wrote, not that the director is waiting or alive.",
			"The JSON adds director.wait (wait_id, pid, started_at, seen_at, watch_accepting, ended_at, end, detail) " +
				"when there is a mark of the current registration, a headless one's included.",
		}},
	} {
		help := strings.Join(strings.Fields(c.out), " ")
		for _, want := range c.want {
			if !strings.Contains(help, want) {
				t.Errorf("%s does not say %q:\n%s", c.name, want, c.out)
			}
		}
	}
	// The 30s of the help is the interval the wait rewrites its mark at, and
	// the 60s two of them.
	if org.DirectorWaitMarkInterval != 30*time.Second {
		t.Errorf("org.DirectorWaitMarkInterval = %v; the help says 30s and 60s", org.DirectorWaitMarkInterval)
	}
}

// D2 and D3: the help of `ralph org wait` says the intervals the org layer
// waits with, org.DirectorWaitMarkInterval for the read of the registration
// and the rewrite of the mark and org.DirectorWaitWatchProbeInterval for the
// probe of the watch, so a change of either constant fails here.
func TestDirectorWaitLine_TheWaitHelpSaysTheIntervalsOfTheOrgLayer(t *testing.T) {
	stdout, _ := runInboxOK(t, "wait", "--help")
	help := strings.Join(strings.Fields(stdout), " ")
	secs := func(d time.Duration) string { return strconv.FormatFloat(d.Seconds(), 'f', -1, 64) + " s" }
	mark, probe := secs(org.DirectorWaitMarkInterval), secs(org.DirectorWaitWatchProbeInterval)
	for _, want := range []string{
		"While it waits, it reads the registration again every " + mark + " and exits 5",
		"in the state dir, when it starts, every " + mark + ", when the watch it sees changes",
		"Every " + probe + " it probes the cross-org watch (ralph org watch --all)",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("ralph org wait --help does not say %q:\n%s", want, stdout)
		}
	}
}
