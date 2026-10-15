package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org director attach | detach | status` (plan
// docs/plans/active/2026-10-14-org-director-attach.md, AC2, AC5 and AC9).
// Every test uses a ledger under t.TempDir(), named with --state-dir or
// found in a temp repository (newLegacyLedgerRepo), never this package's
// checkout, and none runs herdr or osascript.

// directorNotAttachedLine is the director: line with no registration and
// the switch on.
const directorNotAttachedLine = "director: not attached -- inbox items reach the human at the ack deadline of ralph org watch --all, or at once while no watch accepts work"

// directorDisabledLine is the director: line with the switch off and no
// registration.
const directorDisabledLine = "director: disabled ([org.director].enabled = false) -- ralph org escalate sends every item to the human at once"

// directorAttachedLine is the director: line of a registration with id,
// driver, attach time ts, and user, and no pane.
func directorAttachedLine(id, driver, ts, user string) string {
	by := ""
	if user != "" {
		by = ", by " + user
	}
	return "director: attached " + id + " (driver=" + driver + ", attached at " + ts + by + ") -- " +
		"the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline"
}

// directorFileOrAbsent returns director.jsonl in stateDir, or "<absent>"
// when it does not exist.
func directorFileOrAbsent(t *testing.T, stateDir string) string {
	t.Helper()
	raw, err := os.ReadFile(org.DirectorPathIn(stateDir))
	if errors.Is(err, os.ErrNotExist) {
		return "<absent>"
	}
	if err != nil {
		t.Fatalf("read director.jsonl: %v", err)
	}
	return string(raw)
}

// directorEventsIn decodes every line of director.jsonl in stateDir.
func directorEventsIn(t *testing.T, stateDir string) []org.DirectorEvent {
	t.Helper()
	var events []org.DirectorEvent
	for _, line := range readLogLines(t, org.DirectorPathIn(stateDir)) {
		var ev org.DirectorEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("parse the director line %q: %v", line, err)
		}
		events = append(events, ev)
	}
	return events
}

// directorEventKinds returns "<event> <id>" for each event.
func directorEventKinds(events []org.DirectorEvent) []string {
	kinds := make([]string, 0, len(events))
	for _, ev := range events {
		kinds = append(kinds, ev.Event+" "+ev.ID)
	}
	return kinds
}

// directorReportOf decodes `ralph org director status --json` output.
func directorReportOf(t *testing.T, out string) orgDirectorReport {
	t.Helper()
	var report orgDirectorReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("director status --json: %v\n%s", err, out)
	}
	return report
}

// directorStatusLine returns the first line of `ralph org director status`,
// the director: line.
func directorStatusLine(t *testing.T, args ...string) string {
	t.Helper()
	stdout, _ := runInboxOK(t, append([]string{"director", "status"}, args...)...)
	line, _, _ := strings.Cut(stdout, "\n")
	return line
}

// AC2: attach records one attached line with its id and who ran it (USER,
// the current directory, whether HERDR_PANE_ID is set, never its value),
// and status shows it. A second attach is refused and writes nothing;
// --replace ends d1 and records d2 under one lock. Then the session that
// attached d1 runs its detach, as the /director skill says, and is refused
// (the A -> B -> A sequence): d2 stays attached until its own detach.
func TestOrgDirector_AttachReplaceAndDetachByID(t *testing.T) {
	setEscalateWatch(t, acceptingWatchLiveness())
	t.Setenv("USER", "alice")
	const pane = "w9-pane-secret"
	t.Setenv("HERDR_PANE_ID", pane)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	stateDir := t.TempDir()
	ledger := []string{"--state-dir", stateDir}
	suffix := " --state-dir '" + stateDir + "'"

	stdout, stderr := runInboxOK(t, append([]string{"director", "attach", "--driver", "claude"}, ledger...)...)
	if want := "attached d1 (driver=claude)\ndetach it with: ralph org director detach --id d1" + suffix + "\n"; stdout != want {
		t.Errorf("attach stdout = %q; want %q", stdout, want)
	}
	if stderr != "" {
		t.Errorf("attach stderr = %q; want nothing while a watch accepts work", stderr)
	}
	events := directorEventsIn(t, stateDir)
	if len(events) != 1 {
		t.Fatalf("director.jsonl = %+v; want one attached line", events)
	}
	ev := events[0]
	if ev.Event != org.DirectorEventAttached || ev.ID != "d1" || ev.Driver != "claude" || ev.User != "alice" || ev.Cwd != cwd ||
		!ev.HerdrPane || !slices.Equal(ev.Wake, []string{org.DirectorWakePull}) || ev.PaneID != "" || ev.TS == "" {
		t.Errorf("attached event = %+v; want d1, claude, alice, cwd %s, herdr_pane true, wake pull, no pane_id", ev, cwd)
	}
	if raw := directorFileOrAbsent(t, stateDir); strings.Contains(raw, pane) {
		t.Errorf("director.jsonl holds the value of HERDR_PANE_ID:\n%s", raw)
	}

	statusOut, _ := runInboxOK(t, append([]string{"director", "status"}, ledger...)...)
	wantStatus := directorAttachedLine("d1", "claude", ev.TS, "alice") + "\n" +
		"enabled: true\nid: d1\ndriver: claude\nwake: pull\nattached_at: " + ev.TS + "\nuser: alice\ncwd: " + cwd + "\nherdr_pane: yes\n"
	if statusOut != wantStatus {
		t.Errorf("director status =\n%s\nwant\n%s", statusOut, wantStatus)
	}
	jsonOut, _ := runInboxOK(t, append([]string{"director", "status", "--json"}, ledger...)...)
	report := directorReportOf(t, jsonOut)
	if !report.Enabled || report.State != "attached" || report.Registration == nil || report.Registration.ID != "d1" ||
		report.Registration.User != "alice" || !report.Registration.HerdrPane || report.Error != "" {
		t.Errorf("director status --json = %s; want enabled, attached d1 by alice", jsonOut)
	}

	before := directorFileOrAbsent(t, stateDir)
	stdout, _, err = runOrgCmdStreams(t, append([]string{"director", "attach", "--driver", "codex"}, ledger...)...)
	if !errors.Is(err, org.ErrDirectorAttached) {
		t.Fatalf("second attach error = %v; want ErrDirectorAttached", err)
	}
	for _, want := range []string{
		// The store's message ends at the registration, and the two ways
		// out follow it once (self-review L7 of org-director-attach).
		"a director is already attached: d1 (driver claude, attached " + ev.TS + "): end it with ralph org director detach --id d1" + suffix +
			" from the session that attached it",
		"--force in place of --id",
		"replace it with ralph org director attach --driver codex --replace" + suffix,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("second attach error %q does not contain %q", err, want)
		}
	}
	if n := strings.Count(err.Error(), "replace"); n != 2 {
		t.Errorf("second attach error %q says replace %d times; want 2 (the clause and its --replace command)", err, n)
	}
	if stdout != "" {
		t.Errorf("refused attach stdout = %q; want nothing", stdout)
	}
	if after := directorFileOrAbsent(t, stateDir); after != before {
		t.Errorf("refused attach changed director.jsonl:\nbefore: %s\nafter:  %s", before, after)
	}

	t.Setenv("USER", "bob")
	stdout, _ = runInboxOK(t, append([]string{"director", "attach", "--driver", "codex", "--replace"}, ledger...)...)
	if want := "attached d2 (driver=codex), replacing d1 (driver=claude, attached at " + ev.TS + ")\n" +
		"detach it with: ralph org director detach --id d2" + suffix + "\n"; stdout != want {
		t.Errorf("attach --replace stdout = %q; want %q", stdout, want)
	}
	events = directorEventsIn(t, stateDir)
	if got, want := directorEventKinds(events), []string{"attached d1", "detached d1", "attached d2"}; !slices.Equal(got, want) {
		t.Fatalf("director.jsonl after --replace = %v; want %v", got, want)
	}
	if events[1].User != "bob" || events[2].User != "bob" || events[2].Driver != "codex" {
		t.Errorf("the replace events = %+v; want bob's detached d1 and codex attached d2", events[1:])
	}

	// A, whose registration was replaced, detaches as told.
	before = directorFileOrAbsent(t, stateDir)
	stdout, _, err = runOrgCmdStreams(t, append([]string{"director", "detach", "--id", "d1"}, ledger...)...)
	if !errors.Is(err, org.ErrDirectorIDMismatch) {
		t.Fatalf("detach --id d1 after the replace: error = %v; want ErrDirectorIDMismatch", err)
	}
	for _, want := range []string{"the current registration is d2", "was replaced", "does not need to detach"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("detach --id d1 error %q does not contain %q", err, want)
		}
	}
	if stdout != "" {
		t.Errorf("refused detach stdout = %q; want nothing", stdout)
	}
	if after := directorFileOrAbsent(t, stateDir); after != before {
		t.Errorf("refused detach changed director.jsonl:\nbefore: %s\nafter:  %s", before, after)
	}
	if got, want := directorStatusLine(t, ledger...), directorAttachedLine("d2", "codex", events[2].TS, "bob"); got != want {
		t.Errorf("director line after the refused detach:\n got %q\nwant %q", got, want)
	}

	stdout, _ = runInboxOK(t, append([]string{"director", "detach", "--id", "d2"}, ledger...)...)
	if stdout != "detached d2\n" {
		t.Errorf("detach --id d2 stdout = %q; want detached d2", stdout)
	}
	events = directorEventsIn(t, stateDir)
	if last := events[len(events)-1]; last.Event != org.DirectorEventDetached || last.ID != "d2" || last.Force {
		t.Errorf("last event = %+v; want detached d2 without force", last)
	}
	if got := directorStatusLine(t, ledger...); got != directorNotAttachedLine {
		t.Errorf("director line after detach = %q; want %q", got, directorNotAttachedLine)
	}
	jsonOut, _ = runInboxOK(t, append([]string{"director", "status", "--json"}, ledger...)...)
	if report := directorReportOf(t, jsonOut); report.State != "not_attached" || report.Registration != nil || !strings.Contains(jsonOut, `"registration": null`) {
		t.Errorf("director status --json after detach = %s; want not_attached with registration null", jsonOut)
	}

	before = directorFileOrAbsent(t, stateDir)
	if _, _, err := runOrgCmdStreams(t, append([]string{"director", "detach", "--id", "d2"}, ledger...)...); !errors.Is(err, org.ErrDirectorNotAttached) {
		t.Errorf("a second detach --id d2: error = %v; want ErrDirectorNotAttached", err)
	}
	if after := directorFileOrAbsent(t, stateDir); after != before {
		t.Errorf("refused detach changed director.jsonl")
	}
}

// A write the store could not cut back is followed by the command that reads
// the registration again, with the --state-dir of the call when one was
// given, as the other commands the director verbs print are (self-review
// C2-L1 of org-director-attach). The store's write seam is unexported, so
// this tests the helper that attach and detach both return through. Any
// other error, ErrDirectorAttached included, is returned as it is.
func TestOrgDirector_NotRestoredErrorNamesTheStatusCommand(t *testing.T) {
	notRestored := fmt.Errorf("org: director: attach: append to /work/ledger/director.jsonl: no space left on device, "+
		"and %w (truncate refused), so it may hold part of the write", org.ErrDirectorWriteNotRestored)
	for _, tc := range []struct {
		name, hintStateDir, want string
	}{
		{"without --state-dir", "", notRestored.Error() + ": read the registration with ralph org director status"},
		{"with --state-dir", "/work/the ledger", notRestored.Error() + ": read the registration with ralph org director status --state-dir '/work/the ledger'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt := &orgDirectorRuntime{hintStateDir: tc.hintStateDir}
			err := rt.notRestoredError(notRestored)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("notRestoredError = %v; want %q", err, tc.want)
			}
			if !errors.Is(err, org.ErrDirectorWriteNotRestored) {
				t.Errorf("notRestoredError = %v; want it to still wrap ErrDirectorWriteNotRestored", err)
			}
			for _, other := range []error{
				fmt.Errorf("org: director: attach: nothing was recorded: append to /work/ledger/director.jsonl: %w", os.ErrPermission),
				fmt.Errorf("org: director: attach: %w: d1 (driver claude, attached 2026-10-15T09:00:00Z)", org.ErrDirectorAttached),
			} {
				if got := rt.notRestoredError(other); got != other {
					t.Errorf("notRestoredError(%q) = %v; want the error returned as it is", other, got)
				}
			}
		})
	}
}

// AC2: detach takes exactly one of --id and --force, an id of the form
// d<N>, and writes nothing when refused.
func TestOrgDirector_DetachNeedsExactlyOneOfIDAndForce(t *testing.T) {
	setEscalateWatch(t, acceptingWatchLiveness())
	stateDir := t.TempDir()
	runInboxOK(t, "director", "attach", "--driver", "claude", "--state-dir", stateDir)
	before := directorFileOrAbsent(t, stateDir)

	for _, tc := range []struct {
		args []string
		want string
	}{
		{nil, "org: director detach: give --id <id>, the id attach printed, or --force"},
		{[]string{"--id", ""}, "org: director detach: give --id <id>, the id attach printed, or --force"},
		{[]string{"--id", "d1", "--force"}, "org: director detach: --id and --force cannot be used together"},
		{[]string{"--id", "x1"}, `"x1" is not a registration id (d<N>, such as d1)`},
	} {
		stdout, _, err := runOrgCmdStreams(t, append(append([]string{"director", "detach"}, tc.args...), "--state-dir", stateDir)...)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("detach %v error = %v; want %q", tc.args, err, tc.want)
		}
		if stdout != "" {
			t.Errorf("detach %v stdout = %q; want nothing", tc.args, stdout)
		}
	}
	if after := directorFileOrAbsent(t, stateDir); after != before {
		t.Errorf("a refused detach changed director.jsonl:\nbefore: %s\nafter:  %s", before, after)
	}
}

// AC2: detach --force ends the current registration whatever its id, says
// it was forced, and records force and who ran it.
func TestOrgDirector_DetachForce(t *testing.T) {
	setEscalateWatch(t, acceptingWatchLiveness())
	stateDir := t.TempDir()
	t.Setenv("USER", "alice")
	runInboxOK(t, "director", "attach", "--driver", "claude", "--state-dir", stateDir)
	attachedAt := directorEventsIn(t, stateDir)[0].TS

	t.Setenv("USER", "bob")
	t.Setenv("HERDR_PANE_ID", "")
	stdout, _ := runInboxOK(t, "director", "detach", "--force", "--state-dir", stateDir)
	if want := "detached d1 (forced; driver=claude, attached at " + attachedAt + ")\n"; stdout != want {
		t.Errorf("detach --force stdout = %q; want %q", stdout, want)
	}
	events := directorEventsIn(t, stateDir)
	if len(events) != 2 || events[1].Event != org.DirectorEventDetached || events[1].ID != "d1" || !events[1].Force ||
		events[1].User != "bob" || events[1].HerdrPane {
		t.Errorf("director.jsonl = %+v; want attached d1, then bob's forced detached d1", events)
	}
	if _, _, err := runOrgCmdStreams(t, "director", "detach", "--force", "--state-dir", stateDir); !errors.Is(err, org.ErrDirectorNotAttached) {
		t.Errorf("detach --force with no registration: error = %v; want ErrDirectorNotAttached", err)
	}
}

// AC2: a --driver other than claude and codex, or none, is refused before
// anything is read: not even a ralph.toml that does not load is reported,
// and the state dir is not created. director start refuses it the same way
// (plan 2026-10-15-org-director-start, AC2), and runs no herdr.
func TestOrgDirector_AttachRefusesABadDriverBeforeReadingAnything(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	broken := filepath.Join(t.TempDir(), "ralph.toml")
	writeTestFile(t, broken, "[org\n")
	for _, verb := range []string{"attach", "start"} {
		for _, tc := range []struct {
			args []string
			want string
		}{
			{[]string{"--driver", "gemini"}, `org: director ` + verb + `: --driver "gemini" is not claude or codex`},
			{nil, "org: director " + verb + ": --driver is required: claude or codex"},
			{[]string{"--driver", "Claude"}, `--driver "Claude" is not claude or codex`},
		} {
			stdout, _, err := runOrgCmdStreams(t, append(append([]string{"director", verb}, tc.args...), "--state-dir", stateDir, "--config", broken)...)
			if err == nil || !strings.HasPrefix(err.Error(), "org: director "+verb+": ") || !strings.HasSuffix(err.Error(), tc.want) || strings.Contains(err.Error(), "config") {
				t.Errorf("%s %v error = %v; want %q, with nothing about the config", verb, tc.args, err, tc.want)
			}
			if stdout != "" {
				t.Errorf("%s %v stdout = %q; want nothing", verb, tc.args, stdout)
			}
		}
	}
	if _, err := os.Stat(stateDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a refused attach or start created the state dir (stat: %v)", err)
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %v; want none", lines)
	}
}

// AC2: the registration belongs to the whole ledger, so every director verb
// refuses --org-id and writes nothing.
func TestOrgDirector_RefusesOrgID(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	for _, args := range [][]string{
		{"director", "attach", "--driver", "claude"},
		{"director", "attach", "--driver", "codex", "--replace"},
		{"director", "detach", "--id", "d1"},
		{"director", "detach", "--force"},
		{"director", "start", "--driver", "claude", "--model", "opus"},
		{"director", "stop", "--id", "d1"},
		{"director", "stop", "--id", "d1", "--force"},
		{"director", "status"},
		{"director", "status", "--json"},
	} {
		stdout, _, err := runOrgCmdStreams(t, append(args, "--org-id", "org-a", "--state-dir", stateDir)...)
		verb := strings.Join(args[:2], " ")
		if want := "org: " + verb + ": --org-id does not apply: the director registration belongs to the whole ledger, not to one org"; err == nil || err.Error() != want {
			t.Errorf("%v --org-id error = %v; want %q", args, err, want)
		}
		if stdout != "" {
			t.Errorf("%v --org-id stdout = %q; want nothing", args, stdout)
		}
	}
	if _, err := os.Stat(stateDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a refused verb created the state dir (stat: %v)", err)
	}
}

// A mistyped director verb exits 1 as an unknown command, as for `ralph org
// inbox`, while the bare group prints its help and exits 0.
func TestOrgDirector_UnknownVerbIsAnError(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	if _, _, err := runOrgCmdStreams(t, "director", "atach", "--state-dir", stateDir); err == nil ||
		!strings.Contains(err.Error(), `unknown command "atach" for "ralph org director"`) {
		t.Errorf("director atach error = %v; want unknown command", err)
	}
	stdout, _ := runInboxOK(t, "director", "--state-dir", stateDir)
	if !strings.HasPrefix(stdout, "ralph org director keeps the director registration of the ledger") {
		t.Errorf("bare director stdout = %q; want the group's help", stdout)
	}
	if _, err := os.Stat(stateDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the group created the state dir (stat: %v)", err)
	}
}

// AC2, AC5: attach is refused, and writes nothing, when [org.director].enabled
// is false, read as escalate reads it: from the main worktree's ralph.toml
// when no --config is given and the ledger is the main worktree's, so a
// feature worktree's ralph.toml does not switch it. The error names the file
// it read and how to turn the director on. director status and ralph status
// say disabled by the same rule.
func TestOrgDirector_EnabledFromTheMainWorktreeRalphToml(t *testing.T) {
	const disabled = "[org.director]\nenabled = false\n"
	cases := []struct {
		name     string
		mainToml string // the main checkout's ralph.toml ("" for none)
		// setup writes the worktree's files and returns the flags of every
		// verb, the ledger they use ("" for the shared one), and the file
		// the refusal names ("" when attach succeeds).
		setup func(t *testing.T, r legacyLedgerRepo) (extra []string, ledger, refusedBy string)
	}{
		{"main says false, from the linked worktree", disabled, func(t *testing.T, r legacyLedgerRepo) ([]string, string, string) {
			return nil, "", "the main worktree's " + filepath.Join(r.mainRoot, "ralph.toml")
		}},
		{"main says false, the worktree says true", disabled, func(t *testing.T, r legacyLedgerRepo) ([]string, string, string) {
			writeTestFile(t, filepath.Join(r.worktree, "ralph.toml"), "[org.director]\nenabled = true\n")
			return nil, "", "the main worktree's " + filepath.Join(r.mainRoot, "ralph.toml")
		}},
		{"main says false, --state-dir names the shared ledger", disabled, func(t *testing.T, r legacyLedgerRepo) ([]string, string, string) {
			return []string{"--state-dir", r.sharedDir}, "", "the main worktree's " + filepath.Join(r.mainRoot, "ralph.toml")
		}},
		{"no main ralph.toml, the worktree says false", "", func(t *testing.T, r legacyLedgerRepo) ([]string, string, string) {
			writeTestFile(t, filepath.Join(r.worktree, "ralph.toml"), disabled)
			return nil, "", ""
		}},
		{"--config says false", "", func(t *testing.T, r legacyLedgerRepo) ([]string, string, string) {
			path := filepath.Join(t.TempDir(), "ralph.toml")
			writeTestFile(t, path, disabled)
			return []string{"--config", path}, "", "the --config file " + path
		}},
		{"the worktree says false, --state-dir names another ledger", "", func(t *testing.T, r legacyLedgerRepo) ([]string, string, string) {
			writeTestFile(t, filepath.Join(r.worktree, "ralph.toml"), disabled)
			other := filepath.Join(t.TempDir(), "org")
			return []string{"--state-dir", other}, other, filepath.Join(r.worktree, "ralph.toml")
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setEscalateWatch(t, acceptingWatchLiveness())
			r := newLegacyLedgerRepo(t)
			if tc.mainToml != "" {
				writeTestFile(t, filepath.Join(r.mainRoot, "ralph.toml"), tc.mainToml)
			}
			extra, ledger, refusedBy := tc.setup(t, r)
			if ledger == "" {
				ledger = r.sharedDir
			}

			stdout, _, err := runOrgCmdStreams(t, append([]string{"director", "attach", "--driver", "claude"}, extra...)...)
			if refusedBy == "" {
				if err != nil || !strings.HasPrefix(stdout, "attached d1 (driver=claude)\n") {
					t.Fatalf("attach = %q, %v; want attached d1", stdout, err)
				}
				if got := directorEventKinds(directorEventsIn(t, ledger)); !slices.Equal(got, []string{"attached d1"}) {
					t.Errorf("director.jsonl in %s = %v; want attached d1", ledger, got)
				}
				if got := directorStatusLine(t, extra...); !strings.HasPrefix(got, "director: attached d1 (driver=claude, ") {
					t.Errorf("director line = %q; want attached d1", got)
				}
				return
			}
			want := "org: director attach: [org.director].enabled is false in " + refusedBy + ", so ralph org escalate already sends every item to the human at once " +
				"and leaves nothing to a director; to use one, set [org.director].enabled = true there"
			if err == nil || err.Error() != want {
				t.Errorf("attach error = %v; want %q", err, want)
			}
			if stdout != "" {
				t.Errorf("refused attach stdout = %q; want nothing", stdout)
			}
			if got := directorFileOrAbsent(t, ledger); got != "<absent>" {
				t.Errorf("refused attach wrote director.jsonl in %s: %s", ledger, got)
			}
			if got := directorStatusLine(t, extra...); got != directorDisabledLine {
				t.Errorf("director line = %q; want %q", got, directorDisabledLine)
			}
			if !slices.Contains(extra, "--config") {
				statusOut, statusErr, err := runRootCmdSplit(t, append([]string{"status"}, extra...)...)
				if err != nil {
					t.Fatalf("ralph status: %v (stderr %s)", err, statusErr)
				}
				if got := statusDirectorLine(t, statusOut); got != directorDisabledLine {
					t.Errorf("ralph status director line = %q; want %q", got, directorDisabledLine)
				}
			}
		})
	}
}

// A main worktree ralph.toml that does not load fails attach and director
// status with an error that names the file, and writes nothing.
func TestOrgDirector_MainWorktreeRalphTomlLoadError(t *testing.T) {
	r := newLegacyLedgerRepo(t)
	mainToml := filepath.Join(r.mainRoot, "ralph.toml")
	writeTestFile(t, mainToml, "[org]\nmax_orgs = 0\n")
	for _, args := range [][]string{{"director", "attach", "--driver", "claude"}, {"director", "status"}} {
		stdout, _, err := runOrgCmdStreams(t, args...)
		if err == nil || !strings.Contains(err.Error(), "org: load [org.director].enabled from the main worktree's "+mainToml) ||
			!strings.Contains(err.Error(), "[org].max_orgs must be >= 1, got 0") {
			t.Errorf("%v error = %v; want the main worktree load error", args, err)
		}
		if stdout != "" {
			t.Errorf("%v stdout = %q; want nothing", args, stdout)
		}
	}
	if got := directorFileOrAbsent(t, r.sharedDir); got != "<absent>" {
		t.Errorf("director.jsonl = %q; want absent", got)
	}
}

// detach reads no ralph.toml (self-review L4 of org-director-attach): a
// --config file, a ./ralph.toml, or a main worktree ralph.toml that does not
// parse blocks neither detach --id nor detach --force, so a broken config
// cannot keep a registration from being cleaned up, while attach is still
// refused on it.
func TestOrgDirector_DetachReadsNoConfig(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T) (args []string, brokenPath string)
	}{
		{"the --config file", func(t *testing.T) ([]string, string) {
			path := filepath.Join(t.TempDir(), "ralph.toml")
			return []string{"--state-dir", t.TempDir(), "--config", path}, path
		}},
		{"./ralph.toml", func(t *testing.T) ([]string, string) {
			t.Chdir(t.TempDir())
			return []string{"--state-dir", t.TempDir()}, "ralph.toml"
		}},
		{"the main worktree's ralph.toml", func(t *testing.T) ([]string, string) {
			r := newLegacyLedgerRepo(t)
			return nil, filepath.Join(r.mainRoot, "ralph.toml")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setEscalateWatch(t, acceptingWatchLiveness())
			args, brokenPath := tc.setup(t)
			run := func(verb ...string) (string, error) {
				stdout, _, err := runOrgCmdStreams(t, append(append([]string{"director"}, verb...), args...)...)
				return stdout, err
			}
			for _, detach := range []struct {
				args []string
				want string
			}{
				{[]string{"detach", "--id", "d1"}, "detached d1\n"},
				{[]string{"detach", "--force"}, "detached d2 (forced; driver=claude, attached at "},
			} {
				if err := os.Remove(brokenPath); err != nil && !errors.Is(err, os.ErrNotExist) {
					t.Fatal(err)
				}
				if _, err := run("attach", "--driver", "claude"); err != nil {
					t.Fatalf("attach with no ralph.toml: %v", err)
				}
				writeTestFile(t, brokenPath, "[org\n")
				if _, err := run("attach", "--driver", "codex", "--replace"); err == nil || !strings.Contains(err.Error(), "load") {
					t.Errorf("attach --replace with a broken %s: error = %v; want the load error", tc.name, err)
				}
				stdout, err := run(detach.args...)
				if err != nil {
					t.Fatalf("%v with a broken %s: %v", detach.args, tc.name, err)
				}
				if !strings.HasPrefix(stdout, detach.want) {
					t.Errorf("%v stdout = %q; want it to start with %q", detach.args, stdout, detach.want)
				}
			}
		})
	}
}

// AC2: attach probes the watch of its ledger as escalate does. While no
// watch accepts work it still records the registration, exits 0, and warns
// on stderr that escalate sends items to the human at once, naming why and
// the command that starts a watch; a long-running watch, through the real
// probe or the test seam, gets no warning.
func TestOrgDirector_AttachWarnsWhileNoWatchAcceptsWork(t *testing.T) {
	once := runningMark(4242, time.Now())
	once.Once = true
	for _, tc := range []struct {
		name    string
		setup   func(t *testing.T, stateDir string)
		wantGap string // "" means no warning
	}{
		{"no watch", func(*testing.T, string) {}, "none runs"},
		{"a running watch", func(t *testing.T, stateDir string) { holdWatchLock(t, stateDir, runningMark(4242, time.Now())) }, ""},
		{"a --once watch", func(t *testing.T, stateDir string) { holdWatchLock(t, stateDir, once) }, "it is a --once run"},
		{"a stale watch", func(t *testing.T, stateDir string) {
			holdWatchLock(t, stateDir, runningMark(4242, time.Now().Add(-time.Hour)))
		}, "its liveness mark is stale"},
		{"a stopping watch", func(t *testing.T, stateDir string) {
			mark := runningMark(4242, time.Now())
			mark.State = org.WatchAliveStopping
			holdWatchLock(t, stateDir, mark)
		}, "it is stopping"},
		{"no watch, the seam says running", func(t *testing.T, _ string) { setEscalateWatch(t, acceptingWatchLiveness()) }, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := t.TempDir()
			tc.setup(t, stateDir)
			suffix := " --state-dir '" + stateDir + "'"

			stdout, stderr := runInboxOK(t, "director", "attach", "--driver", "codex", "--state-dir", stateDir)
			if !strings.HasPrefix(stdout, "attached d1 (driver=codex)\n") {
				t.Errorf("stdout = %q; want attached d1", stdout)
			}
			if got := directorEventKinds(directorEventsIn(t, stateDir)); !slices.Equal(got, []string{"attached d1"}) {
				t.Errorf("director.jsonl = %v; want attached d1", got)
			}
			want := ""
			if tc.wantGap != "" {
				want = "warning: director attach: d1 is recorded, but no ralph org watch --all accepts work on the ledger " + stateDir + " (" + tc.wantGap + "), " +
					"so ralph org escalate sends items to the human at once instead of leaving them to the director; start one with ralph org watch --all --detach" + suffix + "\n"
			}
			if stderr != want {
				t.Errorf("stderr = %q; want %q", stderr, want)
			}
		})
	}
}

// attach, detach, start, and stop change the ledger, so a legacy worktree
// ledger with active seats refuses them and they write nothing (start and
// stop run no herdr either); director status only reads it and prints the
// note once.
func TestOrgDirector_LegacyLedgerGuard(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	r := newLegacyLedgerRepo(t)
	if _, _, err := org.NewDirectorStore(r.sharedDir).Attach(org.DirectorAttach{Driver: "claude", User: "alice"}); err != nil {
		t.Fatalf("seed the registration: %v", err)
	}
	spawnLedgerSeat(t, r.legacyDir, "seat-1")
	before := directorFileOrAbsent(t, r.sharedDir)
	herdrBefore := len(readLogLines(t, herdrLog))

	for _, args := range [][]string{
		{"director", "attach", "--driver", "codex", "--replace"},
		{"director", "detach", "--id", "d1"},
		{"director", "detach", "--force"},
		{"director", "start", "--driver", "claude", "--model", "opus"},
		{"director", "stop", "--id", "d1"},
		{"director", "stop", "--id", "d1", "--force"},
	} {
		stdout, _, err := runOrgCmdStreams(t, args...)
		if err == nil || !strings.Contains(err.Error(), "refusing to change the org ledger") {
			t.Errorf("%v error = %v; want the legacy-ledger refusal", args, err)
		}
		if stdout != "" {
			t.Errorf("%v stdout = %q; want nothing", args, stdout)
		}
	}
	if after := directorFileOrAbsent(t, r.sharedDir); after != before {
		t.Errorf("a refused verb wrote director.jsonl:\nbefore: %s\nafter:  %s", before, after)
	}
	if got := directorFileOrAbsent(t, r.legacyDir); got != "<absent>" {
		t.Errorf("a verb wrote the legacy ledger's director.jsonl: %s", got)
	}
	if got := len(readLogLines(t, herdrLog)); got != herdrBefore {
		t.Errorf("herdr was called %d time(s) after the refusals, want 0", got-herdrBefore)
	}

	stdout, stderr, err := runOrgCmdStreams(t, "director", "status")
	if err != nil {
		t.Fatalf("director status: %v (stderr: %s)", err, stderr)
	}
	if n := strings.Count(stderr, legacyNoteMarker); n != 1 {
		t.Errorf("director status printed the legacy-ledger note %d time(s): %s", n, stderr)
	}
	if !strings.HasPrefix(stdout, "director: attached d1 (driver=claude, ") {
		t.Errorf("director status stdout = %q; want the shared ledger's d1", stdout)
	}
}

// writeDirectorLines writes lines as director.jsonl in stateDir, one per
// line.
func writeDirectorLines(t *testing.T, stateDir string, lines ...string) {
	t.Helper()
	writeTestFile(t, org.DirectorPathIn(stateDir), strings.Join(lines, "\n")+"\n")
}

// directorEventLine is ev as one director.jsonl line.
func directorEventLine(t *testing.T, ev org.DirectorEvent) string {
	t.Helper()
	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// director status warns on stderr about lines that do not parse and events
// the fold ignores, as the inbox verbs do, and keeps --json stdout one JSON
// document carrying the counts. Values a human wrote into director.jsonl
// print with their control characters escaped, so the director: line stays
// one line and the fields cannot drive the terminal.
func TestOrgDirector_StatusWarnsAboutDamageAndEscapesValues(t *testing.T) {
	stateDir := t.TempDir()
	writeDirectorLines(t, stateDir,
		`{"ts":"2026-10-15T09:00:00Z","id":"d1","event":"atta`,
		directorEventLine(t, org.DirectorEvent{
			TS: "2026-10-15T09:00:00Z", ID: "d2", Event: org.DirectorEventAttached, Driver: "claude", Wake: []string{"pull"},
			PaneID: "%7\r", User: "eve\x1b[2J\ndirector: forged", Cwd: "/w\tx",
		}),
		directorEventLine(t, org.DirectorEvent{TS: "2026-10-15T09:01:00Z", ID: "d9", Event: "paused"}),
	)
	path := org.DirectorPathIn(stateDir)

	stdout, stderr, err := runOrgCmdStreams(t, "director", "status", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("director status: %v (stderr: %s)", err, stderr)
	}
	for _, want := range []string{
		"warning: 1 unreadable director line(s) skipped in " + path + "\n",
		"warning: 1 director event(s) ignored in " + path + " (",
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr does not contain %q:\n%s", want, stderr)
		}
	}
	wantLine := `director: attached d2 (driver=claude, attached at 2026-10-15T09:00:00Z, by eve\x1b[2J\ndirector: forged, pane %7\r) -- ` +
		"the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline"
	lines := strings.Split(stdout, "\n")
	if lines[0] != wantLine {
		t.Errorf("director line:\n got %q\nwant %q", lines[0], wantLine)
	}
	for _, want := range []string{`user: eve\x1b[2J\ndirector: forged` + "\n", `cwd: /w\tx` + "\n", `pane_id: %7\r` + "\n"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("director status does not contain %q:\n%s", want, stdout)
		}
	}
	for _, line := range lines[1:] {
		if strings.HasPrefix(line, "director:") {
			t.Errorf("the user forged a second director: line %q:\n%s", line, stdout)
		}
	}
	if strings.ContainsAny(stdout, "\x1b\r\t") {
		t.Errorf("stdout holds a raw control character: %q", stdout)
	}

	stdout, stderr, err = runOrgCmdStreams(t, "director", "status", "--json", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("director status --json: %v (stderr: %s)", err, stderr)
	}
	report := directorReportOf(t, stdout)
	if report.CorruptLines != 1 || report.IgnoredEvents != 1 || report.State != "attached" || report.Registration.User != "eve\x1b[2J\ndirector: forged" {
		t.Errorf("director status --json = %s; want attached d2 with 1 corrupt line and 1 ignored event", stdout)
	}
	if !strings.Contains(stderr, "warning: 1 unreadable director line(s)") {
		t.Errorf("director status --json stderr = %q; want the warning", stderr)
	}
}

// A director.jsonl that cannot be read is reported, not an error: status
// says unreadable with the read error (in JSON too) and exits 0; with the
// switch off it says disabled and names the read error.
func TestOrgDirector_StatusOfAnUnreadableFile(t *testing.T) {
	stateDir := t.TempDir()
	if err := os.Mkdir(org.DirectorPathIn(stateDir), 0o755); err != nil {
		t.Fatal(err)
	}
	readErr := "org: director: read " + org.DirectorPathIn(stateDir) + ": "

	got := directorStatusLine(t, "--state-dir", stateDir)
	if !strings.HasPrefix(got, "director: unreadable ("+readErr) || !strings.HasSuffix(got, ")") {
		t.Errorf("director line = %q; want unreadable with the read error", got)
	}
	jsonOut, _ := runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
	if report := directorReportOf(t, jsonOut); report.State != "unreadable" || !strings.HasPrefix(report.Error, readErr) || report.Registration != nil || !report.Enabled {
		t.Errorf("director status --json = %s; want unreadable with the read error", jsonOut)
	}

	path := filepath.Join(t.TempDir(), "ralph.toml")
	writeTestFile(t, path, "[org.director]\nenabled = false\n")
	got = directorStatusLine(t, "--state-dir", stateDir, "--config", path)
	if !strings.HasPrefix(got, "director: disabled ([org.director].enabled = false; director.jsonl cannot be read: "+readErr) ||
		!strings.HasSuffix(got, ") -- ralph org escalate sends every item to the human at once") {
		t.Errorf("director line with the switch off = %q; want disabled naming the read error", got)
	}
	jsonOut, _ = runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir, "--config", path)
	if report := directorReportOf(t, jsonOut); report.State != "disabled" || report.Enabled || !strings.HasPrefix(report.Error, readErr) {
		t.Errorf("director status --json with the switch off = %s; want disabled with the read error", jsonOut)
	}
}

// With the switch off, a registration left in director.jsonl is named on
// the disabled line and still shown in the fields.
func TestOrgDirector_StatusDisabledNamesALeftRegistration(t *testing.T) {
	setEscalateWatch(t, acceptingWatchLiveness())
	stateDir := t.TempDir()
	runInboxOK(t, "director", "attach", "--driver", "claude", "--state-dir", stateDir)
	path := filepath.Join(t.TempDir(), "ralph.toml")
	writeTestFile(t, path, "[org.director]\nenabled = false\n")

	stdout, _ := runInboxOK(t, "director", "status", "--state-dir", stateDir, "--config", path)
	want := "director: disabled ([org.director].enabled = false; registration d1 is still recorded) -- ralph org escalate sends every item to the human at once\n" +
		"enabled: false\nid: d1\n"
	if !strings.HasPrefix(stdout, want) {
		t.Errorf("director status =\n%s\nwant it to start with\n%s", stdout, want)
	}
}

// AC2 for the Go help: the director group and its verbs say what the
// registration is and is not, and `ralph org --help` names the group and
// the --config rule for enabled. Stage 8a-2 (plan
// 2026-10-15-org-director-start, AC12) replaced the group's "(ralph org
// director start) is not available yet" with the two ways to have a
// director, and added start and stop: a wording update, the pins of
// stage 8a-1 that still apply are kept.
func TestOrgDirectorHelp_DescribesTheRegistration(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		wants []string
	}{
		{[]string{"director", "--help"}, []string{
			"<state-dir>/director.jsonl", "promoted with the /director skill", "One director is registered at a time for every org in the ledger",
			"unless --replace is given", "so a session that was replaced is refused and need not detach", "--force ends the current registration whatever its id",
			"The registration does not change what `ralph org escalate` does", "Nor does it mean the session is still running",
			"There are two ways to have one.",
			"`ralph org director start` launches a headless director in a herdr workspace of its own and registers it, and `ralph org director stop --id <id>` closes it again",
			"status shows the registration, the headless directors that are still open, and [org.director].enabled",
			"attach", "detach", "start", "stop", "status",
		}},
		{[]string{"director", "attach", "--help"}, []string{
			"attached d1 (driver=claude)", "Keep the id: `ralph org director detach --id <id>` needs it", "claude or codex",
			"whether HERDR_PANE_ID was set (not its value)", "attach is refused unless --replace is given",
			"The replaced session's detach is then refused, and it need not detach",
			"A headless director that `ralph org director start` launched counts too, starting or running: the refusal then names `ralph org director stop --id`",
			"attach is refused, and writes nothing, when [org.director].enabled is false",
			"Without --config, enabled is read from the main worktree's ralph.toml", "Start one with `ralph org watch --all --detach`",
		}},
		{[]string{"director", "start", "--help"}, []string{
			"labelled ralph_director_<token>, in one tab labelled director_<token>, where <token> is 12 hex digits new for every start",
			"started d1 (driver=codex, model=..., permission=..., pane=..., workspace=...)",
			"Open herdr and talk to the director in that pane",
			"and nothing to the manifest: the director is not an org seat, so max_orgs and max_total_seats do not count it and `ralph org stop --all` does not stop it",
			"is written before any herdr call, so the registration is decided first; workspace_created and pane_created follow as herdr hands out the ids, and launched once the agent has started",
			"[org.permissions.roles].director, else [org.permissions].default",
			"with --model, which [org.roles].director must allow",
			"Without --config, those settings and [org.director].enabled are read from the main worktree's ralph.toml when the ledger is the main worktree's",
			"inside a herdr pane (HERDR_PANE_ID is set): run start from a terminal outside herdr. start has no --replace.",
			"it launches one when none runs, and is refused when a watch holds the lock but is stale, stopping, or a --once run",
		}},
		{[]string{"director", "stop", "--help"}, []string{
			"a stopped event in <state-dir>/director.jsonl and, when it is the current registration, a detached in the same write",
			"Nothing is written to the manifest.",
			"running, still starting (it closes what was recorded), or detached",
			"stop checks that herdr has the pane in a tab labelled director_<token> inside a workspace labelled ralph_director_<token>",
			"so a director whose pane a human closed in herdr is cleaned up with stop too",
			"(a promoted session's registration is ended with `ralph org director detach --id`)",
			"run it from a terminal outside the director's workspace",
			"--force, a human's cleanup, then records the director stopped anyway, closing nothing more",
			"(herdr workspace list, the workspace labelled ralph_director_<token>)",
			"stop reads no ralph.toml (nor the main worktree's), so one that does not load does not block it",
		}},
		{[]string{"director", "detach", "--help"}, []string{
			"Give --id, the id attach printed", "it need not detach", "--force ends the current registration whatever its id",
			"it cannot be combined with --id",
			"detach reads no ralph.toml (nor the main worktree's), so one that does not load does not block it",
		}},
		{[]string{"director", "status", "--help"}, []string{
			"attached, not attached, disabled, or unreadable", "--json",
			"For a headless director (ralph org director start) the fields add its state (starting or running), model, permission mode, token, workspace, and pane",
			"still open are listed (open_headless), to be closed with `ralph org director stop --id <id>`",
			"With enabled false the state is disabled whether or not a registration is recorded; one that is still recorded is named and its fields are shown",
			"The registration does not tell whether its session is still running",
		}},
		{[]string{"--help"}, []string{
			"`ralph org director` registers the interactive session the human talks to as the director of the ledger, or launches a headless director in herdr (ralph org director start)",
			"approve, the director verbs, and the inbox verbs, which refuse it",
			"escalate, director attach, and director status read [org.director].enabled, and director start reads it with [org.permissions], [org.roles] and model_pool, " +
				"from the main worktree's ralph.toml when the ledger is the main worktree's",
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
	out, err := runOrgCmd(t, "director", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "not available yet") {
		t.Errorf("director --help still says start is not available yet:\n%s", out)
	}
}

// statusDirectorLine returns the one director: line of `ralph status` text
// output, failing the test when there is not exactly one.
func statusDirectorLine(t *testing.T, out string) string {
	t.Helper()
	var found []string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "director:") {
			found = append(found, line)
		}
	}
	if len(found) != 1 {
		t.Fatalf("want exactly one director: line, got %d:\n%s", len(found), out)
	}
	return found[0]
}

// Tests for `ralph org director start` and `stop`, the headless director in
// director status, and the spawn refusal of role director (plan
// docs/plans/active/2026-10-15-org-director-start.md, AC1 to AC3, AC5 to
// AC11, AC13, and AC14 through the CLI). Every test runs the stub herdr
// (setupOrgStubPATH, which also pins HERDR_PANE_ID and HERDR_WORKSPACE_ID
// unset), never herdr itself, on a ledger under t.TempDir().

// directorStartLedger is a ledger for director start under t.TempDir(), with
// the stub herdr on PATH and a working directory of its own with no
// ralph.toml, which the director runs in (the ledger is not a main
// worktree's). It returns the state dir, the herdr stub's log, and that
// directory as os.Getwd spells it.
func directorStartLedger(t *testing.T) (stateDir, herdrLog, cwd string) {
	t.Helper()
	herdrLog, _ = setupOrgStubPATH(t)
	t.Chdir(t.TempDir())
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return t.TempDir(), herdrLog, cwd
}

// writeDirectorConfig writes content as a ralph.toml under t.TempDir() and
// returns its path, for --config.
func writeDirectorConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ralph.toml")
	writeTestFile(t, path, content)
	return path
}

// startDirectorOK runs director start with args on the ledger stateDir,
// fails the test when it fails, and returns the token of the last attached
// event, the registration start made.
func startDirectorOK(t *testing.T, stateDir string, args ...string) (token string) {
	t.Helper()
	runInboxOK(t, append([]string{"director", "start", "--state-dir", stateDir}, args...)...)
	for _, ev := range directorEventsIn(t, stateDir) {
		if ev.Event == org.DirectorEventAttached {
			token = ev.Token
		}
	}
	if token == "" {
		t.Fatalf("no attached event with a token in %s", stateDir)
	}
	return token
}

// seedHeadless records a headless registration in the ledger stateDir
// through the store, as start does around its herdr calls: attached (by
// alice in /work, permission autonomous), then the workspace and the pane
// when given, then launched when launched is set. It runs no herdr and
// returns the registration as attached (with its token).
func seedHeadless(t *testing.T, stateDir, driverName, model, workspaceID, paneID string, launched bool) org.DirectorRegistration {
	t.Helper()
	store := org.NewDirectorStore(stateDir)
	reg, err := store.AttachHeadless(org.DirectorHeadlessAttach{
		Driver: driverName, Model: model, PermissionMode: org.PermissionModeAutonomous, User: "alice", Cwd: "/work",
	})
	if err != nil {
		t.Fatalf("seed the headless registration: %v", err)
	}
	who := org.DirectorWho{User: "alice", Cwd: "/work"}
	if workspaceID != "" {
		if err := store.RecordWorkspace(reg.ID, workspaceID, who); err != nil {
			t.Fatalf("seed the workspace: %v", err)
		}
	}
	if paneID != "" {
		if err := store.RecordPane(reg.ID, paneID, who); err != nil {
			t.Fatalf("seed the pane: %v", err)
		}
	}
	if launched {
		if ok, err := store.MarkLaunched(reg.ID, who); err != nil || !ok {
			t.Fatalf("seed launched: %v, %v", ok, err)
		}
	}
	return reg
}

// herdrActions returns the lines of the herdr stub's log that send a key or
// close something: what a stop that refuses must not run.
func herdrActions(t *testing.T, herdrLog string) []string {
	t.Helper()
	var actions []string
	for _, line := range readLogLines(t, herdrLog) {
		if strings.HasPrefix(line, "pane send-keys ") || strings.HasPrefix(line, "pane close ") || strings.HasPrefix(line, "workspace close ") {
			actions = append(actions, line)
		}
	}
	return actions
}

// resetStubHerdrState empties the herdr stub's state, as a herdr that lost
// its state does: its next workspace, tab, and pane get the ids it handed
// out first again (ws-stub-1, pane-stub-1).
func resetStubHerdrState(t *testing.T) {
	t.Helper()
	dir := os.Getenv("ORG_STUB_HERDR_STATE")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			t.Fatal(err)
		}
	}
}

// AC1 and AC8 through the CLI: director start, against the stub herdr,
// records attached (launch herdr, a token, the model, the permission mode,
// and who ran it), workspace_created, pane_created, and launched, runs
// herdr's workspace create, tab create, and agent start with the token's
// names, the permission flags (and codex's --add-dir for a ledger outside
// the cwd), --model, and the prompt pointer in that order, writes the
// prompt file, and prints the registration, the pane, and the stop command
// with --state-dir. Nothing goes to the manifest or the model receipts.
// director status then shows it running with its pane, workspace, and
// model, and stop closes the pane and the workspace and ends the
// registration.
func TestOrgDirectorStart_LaunchesRegistersAndStops(t *testing.T) {
	for _, tc := range []struct {
		driver, model string
		config        string // the ralph.toml --config names ("" for none)
		permArgs      func(stateDir string) string
	}{
		{"claude", "opus", "", func(string) string { return "--permission-mode bypassPermissions" }},
		{"codex", "gpt-6-astra", "[org.permissions]\ncodex_verified = true\n", func(stateDir string) string {
			return "--sandbox workspace-write --ask-for-approval never --add-dir " + stateDir
		}},
	} {
		t.Run(tc.driver, func(t *testing.T) {
			stateDir, herdrLog, cwd := directorStartLedger(t)
			t.Setenv("USER", "alice")
			args := []string{"director", "start", "--driver", tc.driver, "--model", tc.model, "--state-dir", stateDir}
			if tc.config != "" {
				args = append(args, "--config", writeDirectorConfig(t, tc.config))
			}
			stdout, stderr, err := runOrgCmdStreams(t, args...)
			if err != nil {
				t.Fatalf("director start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			events := directorEventsIn(t, stateDir)
			if got, want := directorEventKinds(events), []string{"attached d1", "workspace_created d1", "pane_created d1", "launched d1"}; !slices.Equal(got, want) {
				t.Fatalf("director.jsonl = %v; want %v", got, want)
			}
			att := events[0]
			if att.Launch != org.DirectorLaunchHerdr || len(att.Token) != 12 || att.Model != tc.model || att.PermissionMode != org.PermissionModeAutonomous ||
				att.Driver != tc.driver || att.User != "alice" || att.Cwd != cwd || att.HerdrPane || att.PaneID != "" {
				t.Errorf("attached = %+v; want launch herdr, a token, %s, autonomous, by alice in %s, no pane", att, tc.model, cwd)
			}
			if events[1].WorkspaceID != "ws-stub-1" || events[2].PaneID != "pane-stub-1" {
				t.Errorf("workspace_created and pane_created = %+v, %+v; want ws-stub-1 and pane-stub-1", events[1], events[2])
			}
			tok := att.Token
			suffix := " --state-dir '" + stateDir + "'"
			wantOut := "started d1 (driver=" + tc.driver + ", model=" + tc.model + ", permission=autonomous, pane=pane-stub-1, workspace=ws-stub-1)\n" +
				"talk to the director in herdr: pane pane-stub-1, in the tab director_" + tok + " of the workspace labelled ralph_director_" + tok + "\n" +
				"stop it with: ralph org director stop --id d1" + suffix + "\n"
			if stdout != wantOut {
				t.Errorf("stdout =\n%s\nwant\n%s", stdout, wantOut)
			}
			if stderr != "" {
				t.Errorf("stderr = %q; want nothing with --model given", stderr)
			}
			prompt := filepath.Join(stateDir, "prompts", "director_"+tok+".md")
			if data, err := os.ReadFile(prompt); err != nil || !strings.Contains(string(data), "/director") {
				t.Errorf("prompt file %s = %v; want the director prompt", prompt, err)
			}
			wantHerdr := []string{
				"workspace create --cwd " + cwd + " --label ralph_director_" + tok,
				"tab create --workspace ws-stub-1 --cwd " + cwd + " --label director_" + tok,
				"agent start director_" + tok + " --kind " + tc.driver + " --pane pane-stub-1 --timeout 60000 -- " + tc.permArgs(stateDir) +
					" --model " + tc.model + " " + org.PromptFilePointer(prompt),
			}
			if got := readLogLines(t, herdrLog); !slices.Equal(got, wantHerdr) {
				t.Errorf("herdr calls =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(wantHerdr, "\n"))
			}
			for _, path := range []string{org.ManifestPathIn(stateDir), org.ReceiptsPathIn(stateDir)} {
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("stat %s = %v; want nothing written to it", path, err)
				}
			}

			jsonOut, _ := runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
			report := directorReportOf(t, jsonOut)
			if reg := report.Registration; report.State != "attached" || reg == nil || reg.ID != "d1" || reg.Launch != org.DirectorLaunchHerdr ||
				reg.State != org.DirectorStateRunning || reg.PaneID != "pane-stub-1" || reg.WorkspaceID != "ws-stub-1" || reg.Model != tc.model ||
				reg.Token != tok || reg.PermissionMode != org.PermissionModeAutonomous || len(report.OpenHeadless) != 0 {
				t.Errorf("director status --json = %s; want d1 running in pane-stub-1 and ws-stub-1 with %s", jsonOut, tc.model)
			}

			stdout, _ = runInboxOK(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
			if want := "stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)\nthe director is no longer attached\n"; stdout != want {
				t.Errorf("director stop stdout = %q; want %q", stdout, want)
			}
			if got, want := herdrActions(t, herdrLog), []string{"pane send-keys pane-stub-1 C-c", "pane close pane-stub-1", "workspace close ws-stub-1"}; !slices.Equal(got, want) {
				t.Errorf("herdr actions of stop = %v; want %v", got, want)
			}
			if got := directorEventKinds(directorEventsIn(t, stateDir)); !slices.Equal(got[4:], []string{"detached d1", "stopped d1"}) {
				t.Errorf("director.jsonl after stop = %v; want detached d1 and stopped d1 appended", got)
			}
			if got := directorStatusLine(t, "--state-dir", stateDir); got != directorNotAttachedLine {
				t.Errorf("director line after stop = %q; want %q", got, directorNotAttachedLine)
			}
		})
	}
}

// AC2 and AC6 through the CLI: director start refuses, with exit 1, nothing
// on stdout, director.jsonl as it was, and no herdr call, when
// [org.director].enabled is false (naming the file), for a --model
// [org.roles].director does not allow, for codex autonomous before
// codex_verified, while a promoted session is attached (naming detach), while
// a headless registration is still starting (naming stop), and inside a
// herdr pane.
func TestOrgDirectorStart_RefusedWithNothingWritten(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, stateDir string) (args []string)
		is    error // a sentinel the error wraps (nil: none checked)
		wants func(stateDir string) []string
	}{
		{"enabled false", func(t *testing.T, _ string) []string {
			return []string{"--driver", "claude", "--model", "opus", "--config", writeDirectorConfig(t, "[org.director]\nenabled = false\n")}
		}, nil, func(string) []string {
			return []string{"org: director start: [org.director].enabled is false in the --config file ",
				", so ralph org escalate already sends every item to the human at once and leaves nothing to a director, and no director is started; " +
					"to use one, set [org.director].enabled = true there"}
		}},
		{"a model [org.roles].director does not allow", func(t *testing.T, _ string) []string {
			return []string{"--driver", "claude", "--model", "opus", "--config", writeDirectorConfig(t, "[org.roles]\ndirector = [\"sonnet\", \"haiku\"]\n")}
		}, nil, func(string) []string {
			return []string{`org: director start: model "opus" is not permitted for role director by [org.roles].director in the --config file `, "(it allows sonnet, haiku)"}
		}},
		{"codex autonomous before codex_verified", func(*testing.T, string) []string {
			return []string{"--driver", "codex", "--model", "gpt-6-astra"}
		}, nil, func(string) []string {
			return []string{`org: director: start: permission mode "autonomous" of role director ([org.permissions.roles].director, else [org.permissions].default): ` +
				`org: codex seat permission mode "autonomous" requires [org.permissions].codex_verified=true`}
		}},
		{"a promoted session is attached", func(t *testing.T, stateDir string) []string {
			setEscalateWatch(t, acceptingWatchLiveness())
			runInboxOK(t, "director", "attach", "--driver", "claude", "--state-dir", stateDir)
			return []string{"--driver", "claude", "--model", "opus"}
		}, org.ErrDirectorAttached, func(stateDir string) []string {
			return []string{"a director is already attached: d1 (driver claude, attached ",
				"): end it with ralph org director detach --id d1 --state-dir '" + stateDir + "' from the session that attached it " +
					"(or with --force in place of --id, when that session is gone), then run start again (start has no --replace)"}
		}},
		{"a headless director is still starting", func(t *testing.T, stateDir string) []string {
			seedHeadless(t, stateDir, "codex", "gpt-6-astra", "", "", false)
			return []string{"--driver", "claude", "--model", "opus"}
		}, org.ErrDirectorAttached, func(stateDir string) []string {
			return []string{"a director is already attached: d1 (driver codex, attached ",
				", headless starting): stop it with ralph org director stop --id d1 --state-dir '" + stateDir + "' from a terminal outside herdr, " +
					"then run start again (start has no --replace)"}
		}},
		{"inside a herdr pane", func(t *testing.T, _ string) []string {
			t.Setenv("HERDR_PANE_ID", "p-9")
			return []string{"--driver", "claude", "--model", "opus"}
		}, org.ErrDirectorStartInsideHerdr, func(string) []string {
			return []string{"HERDR_PANE_ID is set, so this runs inside a herdr pane: run ralph org director start from a terminal outside herdr"}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stateDir, herdrLog, _ := directorStartLedger(t)
			args := tc.setup(t, stateDir)
			before := directorFileOrAbsent(t, stateDir)

			stdout, _, err := runOrgCmdStreams(t, append([]string{"director", "start", "--state-dir", stateDir}, args...)...)
			if err == nil {
				t.Fatalf("director start succeeded; want refused (stdout %s)", stdout)
			}
			if tc.is != nil && !errors.Is(err, tc.is) {
				t.Errorf("error = %v; want it to wrap %v", err, tc.is)
			}
			for _, want := range tc.wants(stateDir) {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			if after := directorFileOrAbsent(t, stateDir); after != before {
				t.Errorf("a refused start changed director.jsonl:\nbefore: %s\nafter:  %s", before, after)
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("herdr calls = %v; want none", lines)
			}
			if _, err := os.Stat(filepath.Join(stateDir, "prompts")); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("stat prompts/ = %v; want no prompt file written", err)
			}
		})
	}
}

// AC6 through the CLI (D6): without --config, on the main worktree's ledger
// found from a linked worktree, start takes [org.director].enabled,
// [org.permissions], [org.roles], and model_pool from the main worktree's
// ralph.toml, not the linked worktree's, and the director runs in the main
// worktree root; --config names the file to read instead. A main worktree
// ralph.toml that does not load is an error.
func TestOrgDirectorStart_ReadsTheMainWorktreeRalphToml(t *testing.T) {
	const mainToml = "[org]\nmodel_pool = [{ driver = \"claude\", model = \"haiku\" }, { driver = \"claude\", model = \"opus\" }]\n\n" +
		"[org.permissions]\ndefault = \"guarded\"\n\n[org.roles]\ndirector = [\"haiku\"]\n"
	const worktreeToml = "[org.permissions]\ndefault = \"autonomous\"\n\n[org.roles]\ndirector = [\"opus\"]\n"
	cases := []struct {
		name               string
		mainToml, wtToml   string
		args               []string
		wantErr            func(r legacyLedgerRepo) []string // nil: start succeeds
		wantStderr         string
		wantPerm, wantArgs string // the started line's permission, and agent start's flags before the pointer
		wantModel          string
	}{
		{name: "no --config: the main worktree's settings", mainToml: mainToml, wtToml: worktreeToml,
			args:       []string{"--driver", "claude"},
			wantStderr: "org: --model omitted; falling back to first [org].model_pool entry permitted for role director on claude: haiku (pass --model explicitly)\n",
			wantPerm:   "guarded", wantArgs: "--model haiku", wantModel: "haiku"},
		{name: "no --config: a model only the worktree allows", mainToml: mainToml, wtToml: worktreeToml,
			args: []string{"--driver", "claude", "--model", "opus"},
			wantErr: func(r legacyLedgerRepo) []string {
				return []string{`org: director start: model "opus" is not permitted for role director by [org.roles].director in the main worktree's ` +
					filepath.Join(r.mainRoot, "ralph.toml") + " (it allows haiku)"}
			}},
		{name: "--config names the worktree's ralph.toml", mainToml: mainToml, wtToml: worktreeToml,
			args:     []string{"--driver", "claude", "--model", "opus", "--config", "ralph.toml"},
			wantPerm: "autonomous", wantArgs: "--permission-mode bypassPermissions --model opus", wantModel: "opus"},
		{name: "the main worktree says enabled false, the worktree true", mainToml: "[org.director]\nenabled = false\n", wtToml: "[org.director]\nenabled = true\n",
			args: []string{"--driver", "claude", "--model", "opus"},
			wantErr: func(r legacyLedgerRepo) []string {
				return []string{"org: director start: [org.director].enabled is false in the main worktree's " + filepath.Join(r.mainRoot, "ralph.toml") + ", "}
			}},
		{name: "a main worktree ralph.toml that does not load", mainToml: "[org]\nmax_orgs = 0\n",
			args: []string{"--driver", "claude", "--model", "opus"},
			wantErr: func(r legacyLedgerRepo) []string {
				return []string{"org: load [org.director].enabled, [org.permissions], [org.roles] and model_pool from the main worktree's " +
					filepath.Join(r.mainRoot, "ralph.toml"), "[org].max_orgs must be >= 1, got 0"}
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, _ := setupOrgStubPATH(t)
			r := newLegacyLedgerRepo(t)
			writeTestFile(t, filepath.Join(r.mainRoot, "ralph.toml"), tc.mainToml)
			if tc.wtToml != "" {
				writeTestFile(t, filepath.Join(r.worktree, "ralph.toml"), tc.wtToml)
			}

			stdout, stderr, err := runOrgCmdStreams(t, append([]string{"director", "start"}, tc.args...)...)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("director start succeeded; want refused (stdout %s)", stdout)
				}
				for _, want := range tc.wantErr(r) {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("error %q does not contain %q", err, want)
					}
				}
				if got := directorFileOrAbsent(t, r.sharedDir); got != "<absent>" {
					t.Errorf("a refused start wrote director.jsonl: %s", got)
				}
				if lines := readLogLines(t, herdrLog); len(lines) != 0 {
					t.Errorf("herdr calls = %v; want none", lines)
				}
				return
			}
			if err != nil {
				t.Fatalf("director start: %v (stdout: %s stderr: %s)", err, stdout, stderr)
			}
			if stderr != tc.wantStderr {
				t.Errorf("stderr = %q; want %q", stderr, tc.wantStderr)
			}
			events := directorEventsIn(t, r.sharedDir)
			if len(events) != 4 || events[0].PermissionMode != tc.wantPerm || events[0].Model != tc.wantModel {
				t.Fatalf("director.jsonl in the main worktree's ledger = %+v; want d1 with %s and %s", events, tc.wantPerm, tc.wantModel)
			}
			tok := events[0].Token
			if !strings.HasPrefix(stdout, "started d1 (driver=claude, model="+tc.wantModel+", permission="+tc.wantPerm+", ") {
				t.Errorf("stdout = %q; want d1 started with %s and %s", stdout, tc.wantModel, tc.wantPerm)
			}
			herdrLines := readLogLines(t, herdrLog)
			if len(herdrLines) != 3 || herdrLines[0] != "workspace create --cwd "+r.mainRoot+" --label ralph_director_"+tok {
				t.Fatalf("herdr calls = %v; want the workspace made in the main worktree root %s", herdrLines, r.mainRoot)
			}
			prompt := filepath.Join(r.sharedDir, "prompts", "director_"+tok+".md")
			if want := " -- " + tc.wantArgs + " " + org.PromptFilePointer(prompt); !strings.HasSuffix(herdrLines[2], want) {
				t.Errorf("agent start = %q; want it to end with %q", herdrLines[2], want)
			}
		})
	}
}

// AC7 through the CLI (D5), with TestMain's gate that always passes set back
// to nil: start refuses a watch that holds the lock and is stopping or a
// --once run at once, launching nothing, writing nothing, and running no
// herdr; with no watch it launches one as `ralph org start` does, says so on
// stderr with its own verb, and starts the director; a launch that fails is
// refused with start's prefix, which says no director was started.
func TestOrgDirectorStart_WatchGate(t *testing.T) {
	for _, w := range []struct {
		name string
		mark func() *org.WatchAlive
		want string
	}{
		{"stopping", func() *org.WatchAlive { return stoppingWatchMark(4601, time.Now()) }, "(pid 4601) is stopping"},
		{"running --once", func() *org.WatchAlive { return onceWatchMark(4602, time.Now()) }, "(pid 4602) is a --once run"},
	} {
		t.Run(w.name, func(t *testing.T) {
			useRealOrgWatchGate(t)
			refuseOrgWatchLaunch(t)
			stateDir, herdrLog, _ := directorStartLedger(t)
			holdWatchLock(t, stateDir, w.mark())

			stdout, _, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
			checkGateRefusal(t, err, stateDir, w.want, "run this again once it has ended")
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			if got := directorFileOrAbsent(t, stateDir); got != "<absent>" {
				t.Errorf("director.jsonl = %s; want nothing written", got)
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("herdr calls = %v; want none", lines)
			}
		})
	}

	t.Run("no watch: start launches one", func(t *testing.T) {
		useRealOrgWatchGate(t)
		stateDir, _, _ := directorStartLedger(t)
		var got []orgWatchLaunchRequest
		setOrgWatchLaunch(t, lateMarkLauncher(t, stateDir, 4603, 50*time.Millisecond, &got))

		stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
		if err != nil {
			t.Fatalf("director start with no watch: %v (stdout: %s stderr: %s)", err, stdout, stderr)
		}
		if len(got) != 1 || !slices.Equal(got[0].args, []string{"org", "watch", "--all", "--state-dir", stateDir}) {
			t.Errorf("launches = %+v; want one foreground watch of the ledger", got)
		}
		if want := fmt.Sprintf("ralph org director start: started the cross-org watch (pid 4603); log: %s\n", filepath.Join(stateDir, org.WatchLogFileName)); stderr != want {
			t.Errorf("stderr = %q; want %q", stderr, want)
		}
		if !strings.HasPrefix(stdout, "started d1 (driver=claude, ") {
			t.Errorf("stdout = %q; want d1 started", stdout)
		}
	})

	t.Run("a launch that fails", func(t *testing.T) {
		useRealOrgWatchGate(t)
		failOrgWatchLaunch(t)
		stateDir, herdrLog, _ := directorStartLedger(t)

		_, _, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
		if want := "org: director start: the cross-org watch did not start, so no director was started: start /x/ralph: boom"; err == nil || err.Error() != want {
			t.Fatalf("director start with a launch that fails = %v; want %q", err, want)
		}
		if got := directorFileOrAbsent(t, stateDir); got != "<absent>" {
			t.Errorf("director.jsonl = %s; want nothing written", got)
		}
		if lines := readLogLines(t, herdrLog); len(lines) != 0 {
			t.Errorf("herdr calls = %v; want none", lines)
		}
	})
}

// AC3 through the CLI: when herdr's agent start fails, start closes the pane
// and the workspace it made, records d1 stopped (with detached, as it was
// current), and exits 1 saying so. When the workspace cannot be closed it
// records detached only, so d1 stays open, and names the herdr command and
// the stop command with this invocation's --state-dir; status then lists d1
// as open.
func TestOrgDirectorStart_AFailedAgentStartIsCleanedUp(t *testing.T) {
	t.Run("closed", func(t *testing.T) {
		stateDir, herdrLog, _ := directorStartLedger(t)
		t.Setenv("ORG_STUB_FAIL", "agent:start")

		stdout, _, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
		if err == nil || !strings.Contains(err.Error(), "herdr agent start director_") ||
			!strings.HasSuffix(err.Error(), "; start closed what it had made in herdr, and d1 is recorded stopped") {
			t.Fatalf("director start with a failing agent start = %v; want the failure and the cleanup", err)
		}
		if stdout != "" {
			t.Errorf("stdout = %q; want nothing", stdout)
		}
		if got, want := directorEventKinds(directorEventsIn(t, stateDir)),
			[]string{"attached d1", "workspace_created d1", "pane_created d1", "detached d1", "stopped d1"}; !slices.Equal(got, want) {
			t.Errorf("director.jsonl = %v; want %v", got, want)
		}
		if got, want := herdrActions(t, herdrLog), []string{"pane send-keys pane-stub-1 C-c", "pane close pane-stub-1", "workspace close ws-stub-1"}; !slices.Equal(got, want) {
			t.Errorf("herdr actions = %v; want %v", got, want)
		}
		if got := directorStatusLine(t, "--state-dir", stateDir); got != directorNotAttachedLine {
			t.Errorf("director line = %q; want %q", got, directorNotAttachedLine)
		}
	})

	t.Run("not closed", func(t *testing.T) {
		stateDir, _, _ := directorStartLedger(t)
		t.Setenv("ORG_STUB_FAIL", "agent:start")
		t.Setenv("ORG_STUB_CLOSE_FAIL_IDS", "ws-stub-1")
		suffix := " --state-dir '" + stateDir + "'"

		_, _, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
		var cleanup *org.DirectorCleanupError
		if !errors.As(err, &cleanup) || !cleanup.Detached {
			t.Fatalf("director start = %v; want a *DirectorCleanupError with d1 detached", err)
		}
		for _, want := range []string{
			"herdr agent start director_", `its cleanup did not finish: close workspace "ws-stub-1"`,
			"d1 was recorded detached only, so it is no longer the current registration but stays open",
			"close it by hand: herdr workspace close ws-stub-1; afterwards record the stop with ralph org director stop --id d1" + suffix,
		} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not contain %q", err, want)
			}
		}
		if got, want := directorEventKinds(directorEventsIn(t, stateDir)),
			[]string{"attached d1", "workspace_created d1", "pane_created d1", "detached d1"}; !slices.Equal(got, want) {
			t.Errorf("director.jsonl = %v; want %v", got, want)
		}
		want := directorNotAttachedLine + "; the headless director d1 (starting, pane pane-stub-1, workspace ws-stub-1) is still open but not attached: " +
			"stop it with ralph org director stop --id d1"
		if got := directorStatusLine(t, "--state-dir", stateDir); got != want {
			t.Errorf("director line:\n got %q\nwant %q", got, want)
		}
	})
}

// AC8 through the CLI: a headless director detached by detach stays open:
// status names it with the stop command, and stop closes its pane and
// workspace and records stopped alone (no detached: it is not current), after
// which status names it no more.
func TestOrgDirectorStop_ADetachedHeadlessDirector(t *testing.T) {
	stateDir, herdrLog, _ := directorStartLedger(t)
	startDirectorOK(t, stateDir, "--driver", "claude", "--model", "opus")
	runInboxOK(t, "director", "detach", "--id", "d1", "--state-dir", stateDir)

	want := directorNotAttachedLine + "; the headless director d1 (running, pane pane-stub-1, workspace ws-stub-1) is still open but not attached: " +
		"stop it with ralph org director stop --id d1"
	if got := directorStatusLine(t, "--state-dir", stateDir); got != want {
		t.Errorf("director line:\n got %q\nwant %q", got, want)
	}
	jsonOut, _ := runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
	if report := directorReportOf(t, jsonOut); report.Registration != nil || len(report.OpenHeadless) != 1 ||
		report.OpenHeadless[0].ID != "d1" || report.OpenHeadless[0].State != org.DirectorStateRunning {
		t.Errorf("director status --json = %s; want no registration and d1 in open_headless", jsonOut)
	}

	stdout, _ := runInboxOK(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
	if want := "stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)\n"; stdout != want {
		t.Errorf("stop stdout = %q; want %q", stdout, want)
	}
	if got := directorEventKinds(directorEventsIn(t, stateDir)); !slices.Equal(got[len(got)-2:], []string{"detached d1", "stopped d1"}) || len(got) != 6 {
		t.Errorf("director.jsonl = %v; want the detach's detached d1, then stopped d1 alone", got)
	}
	if got := herdrActions(t, herdrLog); len(got) != 3 {
		t.Errorf("herdr actions = %v; want the C-c and the two closes", got)
	}
	if got := directorStatusLine(t, "--state-dir", stateDir); got != directorNotAttachedLine {
		t.Errorf("director line after stop = %q; want %q", got, directorNotAttachedLine)
	}
	jsonOut, _ = runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
	if strings.Contains(jsonOut, "open_headless") {
		t.Errorf("director status --json = %s; want no open_headless once d1 is stopped", jsonOut)
	}
}

// AC5, AC9, and AC14 through the CLI: two ledgers share one stub herdr, and
// herdr loses its state between their starts, so B's director gets the
// pane and workspace ids A recorded. A's stop finds the labels of B's token
// on them: it sends no C-c and closes nothing, writes nothing, exits 1, and
// names the label search and stop --force. A's stop --force then records d1
// stopped with force (and detached, as it is current), still closing
// nothing, and warns how to find A's workspace by its label. B's director is
// untouched, and its own stop closes it.
func TestOrgDirectorStop_RenumberedHerdrIDsAreRefusedThenForced(t *testing.T) {
	stateA, herdrLog, _ := directorStartLedger(t)
	stateB := t.TempDir()
	tokA := startDirectorOK(t, stateA, "--driver", "claude", "--model", "opus")
	resetStubHerdrState(t)
	tokB := startDirectorOK(t, stateB, "--driver", "codex", "--model", "gpt-6-astra", "--config", writeDirectorConfig(t, "[org.permissions]\ncodex_verified = true\n"))
	if tokA == tokB {
		t.Fatalf("both starts have token %s; want one per start", tokA)
	}
	for _, s := range []string{stateA, stateB} {
		ev := directorEventsIn(t, s)
		if ev[1].WorkspaceID != "ws-stub-1" || ev[2].PaneID != "pane-stub-1" {
			t.Fatalf("%s recorded %s and %s; want ws-stub-1 and pane-stub-1 for both ledgers", s, ev[1].WorkspaceID, ev[2].PaneID)
		}
	}
	suffix := " --state-dir '" + stateA + "'"
	labelSearch := "herdr workspace list, then herdr workspace close <the id of the workspace labelled ralph_director_" + tokA + ">"
	const unconfirmed = "ctrl_c=skipped: not confirmed, pane=not closed: not confirmed, workspace=not closed: not confirmed"
	before := directorFileOrAbsent(t, stateA)

	stdout, stderr, err := runOrgCmdStreams(t, "director", "stop", "--id", "d1", "--state-dir", stateA)
	var cleanup *org.DirectorCleanupError
	if !errors.As(err, &cleanup) || !cleanup.Unconfirmed {
		t.Fatalf("A's stop = %v; want an unconfirmed *DirectorCleanupError", err)
	}
	for _, want := range []string{
		`pane "pane-stub-1" is not confirmed as d1's (a tab labelled director_` + tokA + " in a workspace labelled ralph_director_" + tokA + ")",
		"; nothing was recorded; close it by hand: " + labelSearch + "; afterwards record the stop with ralph org director stop --id d1 --force" + suffix,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("A's stop error %q does not contain %q", err, want)
		}
	}
	if stdout != "" || stderr != "stop d1: "+unconfirmed+"\n" {
		t.Errorf("A's stop stdout %q, stderr %q; want nothing, and the notes on stderr", stdout, stderr)
	}
	if after := directorFileOrAbsent(t, stateA); after != before {
		t.Errorf("A's refused stop changed director.jsonl:\nbefore: %s\nafter:  %s", before, after)
	}
	if got := herdrActions(t, herdrLog); len(got) != 0 {
		t.Errorf("herdr actions = %v; want no C-c and no close", got)
	}

	stdout, stderr = runInboxOK(t, "director", "stop", "--id", "d1", "--force", "--state-dir", stateA)
	if want := "stopped d1 (forced; " + unconfirmed + ")\nthe director is no longer attached\n"; stdout != want {
		t.Errorf("A's stop --force stdout = %q; want %q", stdout, want)
	}
	if !strings.HasPrefix(stderr, "warning: director stop --force: d1 is recorded stopped, but its herdr pane or workspace may still be open (") ||
		!strings.HasSuffix(stderr, "); find and close them by hand: "+labelSearch+"\n") {
		t.Errorf("A's stop --force stderr = %q; want the warning with the label search", stderr)
	}
	events := directorEventsIn(t, stateA)
	if got := directorEventKinds(events); !slices.Equal(got[4:], []string{"detached d1", "stopped d1"}) || !events[5].Force || events[4].Force {
		t.Errorf("A's director.jsonl = %+v; want detached d1, then stopped d1 with force", events)
	}
	if got := herdrActions(t, herdrLog); len(got) != 0 {
		t.Errorf("herdr actions after --force = %v; want still no C-c and no close", got)
	}

	if line := directorStatusLine(t, "--state-dir", stateB); !strings.HasPrefix(line, "director: attached d1 (driver=codex, headless running, ") {
		t.Errorf("B's director line = %q; want d1 still running", line)
	}
	stdout, _ = runInboxOK(t, "director", "stop", "--id", "d1", "--state-dir", stateB)
	if want := "stopped d1 (ctrl_c=ok, pane=closed, workspace=closed)\nthe director is no longer attached\n"; stdout != want {
		t.Errorf("B's stop stdout = %q; want %q", stdout, want)
	}
}

// AC9 and AC14 through the CLI: stop refuses, with or without --force,
// writing nothing and sending no C-c or close, when it runs inside the
// director's pane (HERDR_PANE_ID) or inside its workspace (HERDR_WORKSPACE_ID
// on the workspace herdr's label confirmed).
func TestOrgDirectorStop_RefusedInsideTheDirectorsPaneOrWorkspace(t *testing.T) {
	for _, tc := range []struct {
		env, value string
		is         error
	}{
		{"HERDR_PANE_ID", "pane-stub-1", org.ErrDirectorStopSelfPane},
		{"HERDR_WORKSPACE_ID", "ws-stub-1", org.ErrDirectorStopSelfWorkspace},
	} {
		t.Run(tc.env, func(t *testing.T) {
			stateDir, herdrLog, _ := directorStartLedger(t)
			startDirectorOK(t, stateDir, "--driver", "claude", "--model", "opus")
			t.Setenv(tc.env, tc.value)
			before := directorFileOrAbsent(t, stateDir)
			for _, extra := range [][]string{nil, {"--force"}} {
				stdout, _, err := runOrgCmdStreams(t, append([]string{"director", "stop", "--id", "d1", "--state-dir", stateDir}, extra...)...)
				if !errors.Is(err, tc.is) || !strings.Contains(err.Error(), tc.is.Error()) {
					t.Errorf("stop %v inside the director's %s = %v; want %v", extra, tc.env, err, tc.is)
				}
				if stdout != "" {
					t.Errorf("stop %v stdout = %q; want nothing", extra, stdout)
				}
			}
			if after := directorFileOrAbsent(t, stateDir); after != before {
				t.Errorf("a refused stop changed director.jsonl:\nbefore: %s\nafter:  %s", before, after)
			}
			if got := herdrActions(t, herdrLog); len(got) != 0 {
				t.Errorf("herdr actions = %v; want no C-c and no close", got)
			}
		})
	}
}

// AC9 and AC14 through the CLI: a workspace close that fails leaves stop
// refused with nothing written: it says what was closed (on stderr), the
// herdr command for what is left, the stop to run afterwards, and stop
// --force, since a close failure's own stop command does not carry it.
// stop --force then records the stop with the workspace left open.
func TestOrgDirectorStop_ACloseThatFailsNamesForce(t *testing.T) {
	stateDir, _, _ := directorStartLedger(t)
	startDirectorOK(t, stateDir, "--driver", "claude", "--model", "opus")
	t.Setenv("ORG_STUB_CLOSE_FAIL_IDS", "ws-stub-1")
	suffix := " --state-dir '" + stateDir + "'"
	before := directorFileOrAbsent(t, stateDir)

	stdout, stderr, err := runOrgCmdStreams(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
	var cleanup *org.DirectorCleanupError
	if !errors.As(err, &cleanup) || cleanup.Unconfirmed {
		t.Fatalf("stop with a failing workspace close = %v; want a confirmed *DirectorCleanupError", err)
	}
	if want := "; nothing was recorded; close it by hand: herdr workspace close ws-stub-1; afterwards record the stop with ralph org director stop --id d1" + suffix +
		"; or record the stop and leave them open with ralph org director stop --id d1 --force" + suffix; !strings.HasSuffix(err.Error(), want) {
		t.Errorf("stop error %q does not end with %q", err, want)
	}
	if stdout != "" || !strings.HasPrefix(stderr, "stop d1: ctrl_c=ok, pane=closed, workspace=close failed: ") {
		t.Errorf("stop stdout %q, stderr %q; want nothing, and the notes on stderr", stdout, stderr)
	}
	if after := directorFileOrAbsent(t, stateDir); after != before {
		t.Errorf("a refused stop changed director.jsonl:\nbefore: %s\nafter:  %s", before, after)
	}

	stdout, _ = runInboxOK(t, "director", "stop", "--id", "d1", "--force", "--state-dir", stateDir)
	if !strings.HasPrefix(stdout, "stopped d1 (forced; ctrl_c=ok, pane=closed, workspace=close failed: ") {
		t.Errorf("stop --force stdout = %q; want d1 stopped, forced, with the workspace left open", stdout)
	}
	if ev := directorEventsIn(t, stateDir); ev[len(ev)-1].Event != org.DirectorEventStopped || !ev[len(ev)-1].Force {
		t.Errorf("last event = %+v; want stopped d1 with force", ev[len(ev)-1])
	}
}

// AC9 through the CLI: stop refuses, writing nothing and running no herdr,
// without --id, for an id no registration has (naming status, which lists
// the open headless directors), for a registration a promoted session
// attached (naming detach while it is current, and saying it has ended once
// it is not), and for one already stopped.
func TestOrgDirectorStop_Refusals(t *testing.T) {
	t.Run("no --id", func(t *testing.T) {
		herdrLog, _ := setupOrgStubPATH(t)
		stateDir := filepath.Join(t.TempDir(), "state")
		_, _, err := runOrgCmdStreams(t, "director", "stop", "--state-dir", stateDir)
		if want := "org: director stop: give --id <id>, the id start printed (ralph org director status lists the open headless directors)"; err == nil || err.Error() != want {
			t.Errorf("stop without --id = %v; want %q", err, want)
		}
		if _, err := os.Stat(stateDir); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("stat state dir = %v; want it not created", err)
		}
		if lines := readLogLines(t, herdrLog); len(lines) != 0 {
			t.Errorf("herdr calls = %v; want none", lines)
		}
	})

	t.Run("an unknown id", func(t *testing.T) {
		herdrLog, _ := setupOrgStubPATH(t)
		stateDir := t.TempDir()
		_, _, err := runOrgCmdStreams(t, "director", "stop", "--id", "d9", "--state-dir", stateDir)
		if !errors.Is(err, org.ErrDirectorNoOpenHeadless) ||
			!strings.HasSuffix(err.Error(), "no registration d9 is recorded; ralph org director status --state-dir '"+stateDir+"' lists the open headless directors") {
			t.Errorf("stop --id d9 = %v; want no registration d9, naming status", err)
		}
		if got := directorFileOrAbsent(t, stateDir); got != "<absent>" {
			t.Errorf("director.jsonl = %s; want nothing written", got)
		}
		if lines := readLogLines(t, herdrLog); len(lines) != 0 {
			t.Errorf("herdr calls = %v; want none", lines)
		}
	})

	t.Run("a promoted session's registration", func(t *testing.T) {
		herdrLog, _ := setupOrgStubPATH(t)
		setEscalateWatch(t, acceptingWatchLiveness())
		stateDir := t.TempDir()
		runInboxOK(t, "director", "attach", "--driver", "claude", "--state-dir", stateDir)
		before := directorFileOrAbsent(t, stateDir)
		_, _, err := runOrgCmdStreams(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
		if !errors.Is(err, org.ErrDirectorNotHeadless) ||
			!strings.HasSuffix(err.Error(), ": end it with ralph org director detach --id d1 --state-dir '"+stateDir+"' from the session that attached it "+
				"(or with --force in place of --id, when that session is gone)") {
			t.Errorf("stop of a promoted registration = %v; want it to name detach", err)
		}
		if after := directorFileOrAbsent(t, stateDir); after != before {
			t.Errorf("a refused stop changed director.jsonl")
		}
		runInboxOK(t, "director", "detach", "--id", "d1", "--state-dir", stateDir)
		_, _, err = runOrgCmdStreams(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
		if !errors.Is(err, org.ErrDirectorNotHeadless) || !strings.HasSuffix(err.Error(), ": it is not the current registration, so it has already ended and there is nothing to stop") {
			t.Errorf("stop of a detached promoted registration = %v; want it to say it has ended", err)
		}
		if lines := readLogLines(t, herdrLog); len(lines) != 0 {
			t.Errorf("herdr calls = %v; want none", lines)
		}
	})

	t.Run("already stopped", func(t *testing.T) {
		stateDir, herdrLog, _ := directorStartLedger(t)
		startDirectorOK(t, stateDir, "--driver", "claude", "--model", "opus")
		runInboxOK(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
		before, calls := directorFileOrAbsent(t, stateDir), len(readLogLines(t, herdrLog))
		_, _, err := runOrgCmdStreams(t, "director", "stop", "--id", "d1", "--state-dir", stateDir)
		if !errors.Is(err, org.ErrDirectorStopped) || strings.Contains(err.Error(), "lists the open headless directors") {
			t.Errorf("a second stop = %v; want ErrDirectorStopped alone", err)
		}
		if after := directorFileOrAbsent(t, stateDir); after != before {
			t.Errorf("a refused stop changed director.jsonl")
		}
		if got := len(readLogLines(t, herdrLog)); got != calls {
			t.Errorf("herdr was called %d time(s) by the refused stop; want 0", got-calls)
		}
	})
}

// AC8 through the CLI: stop reads no ralph.toml, so a ./ralph.toml and a
// --config that do not load block it no more than they block detach. It
// also stops a registration still starting: with nothing recorded it runs
// no herdr and names the workspace label to look for; with only the
// workspace recorded it closes that workspace.
func TestOrgDirectorStop_AStartingRegistrationAndABrokenRalphToml(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	dir := t.TempDir()
	broken := filepath.Join(dir, "ralph.toml")
	writeTestFile(t, broken, "[org\n")
	t.Chdir(dir)
	stateDir := t.TempDir()

	reg := seedHeadless(t, stateDir, "claude", "opus", "", "", false)
	stdout, _ := runInboxOK(t, "director", "stop", "--id", reg.ID, "--state-dir", stateDir, "--config", broken)
	want := "stopped d1 (ctrl_c=skipped, pane=none recorded, workspace=none recorded)\nthe director is no longer attached\n" +
		"no workspace id was recorded for d1: herdr may still have a workspace labelled ralph_director_" + reg.Token +
		" that start made before it could record it; look for it with herdr workspace list\n"
	if stdout != want {
		t.Errorf("stop of d1 with nothing recorded =\n%s\nwant\n%s", stdout, want)
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %v; want none with nothing recorded", lines)
	}

	reg = seedHeadless(t, stateDir, "claude", "opus", "ws-9", "", false)
	recordStubHerdr(t, "workspace-ws-9", "ralph_director_"+reg.Token)
	stdout, _ = runInboxOK(t, "director", "stop", "--id", reg.ID, "--state-dir", stateDir)
	if want := "stopped d2 (ctrl_c=skipped, pane=none recorded, workspace=closed)\nthe director is no longer attached\n"; stdout != want {
		t.Errorf("stop of d2 with only its workspace = %q; want %q", stdout, want)
	}
	if got := herdrActions(t, herdrLog); !slices.Equal(got, []string{"workspace close ws-9"}) {
		t.Errorf("herdr actions = %v; want only the workspace close", got)
	}
}

// AC2 for attach: a headless registration, even one still starting, is ended
// with director stop, not detach, so attach's refusal names stop (and
// --replace, which leaves its pane open) and writes nothing.
func TestOrgDirectorAttach_RefusedByAHeadlessDirectorNamesStop(t *testing.T) {
	stateDir := t.TempDir()
	seedHeadless(t, stateDir, "codex", "gpt-6-astra", "", "", false)
	before := directorFileOrAbsent(t, stateDir)
	suffix := " --state-dir '" + stateDir + "'"

	stdout, _, err := runOrgCmdStreams(t, "director", "attach", "--driver", "claude", "--state-dir", stateDir)
	if !errors.Is(err, org.ErrDirectorAttached) {
		t.Fatalf("attach over a starting headless director = %v; want ErrDirectorAttached", err)
	}
	if want := ", headless starting): stop it with ralph org director stop --id d1" + suffix + " from a terminal outside herdr, which closes its herdr pane, " +
		"or replace it with ralph org director attach --driver claude --replace" + suffix + ", which leaves that pane open until such a stop"; !strings.HasSuffix(err.Error(), want) {
		t.Errorf("attach error %q does not end with %q", err, want)
	}
	if strings.Contains(err.Error(), "detach") {
		t.Errorf("attach error %q names detach; want stop for a headless director", err)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if after := directorFileOrAbsent(t, stateDir); after != before {
		t.Errorf("a refused attach changed director.jsonl")
	}
}

// AC10 for director status: a headless registration shows its state, model,
// pane, and workspace on the director: line, and its fields in the text
// form; the open headless registrations that are not current are named at
// the end of the line, one or several, and listed as open_headless lines and
// in JSON. A promoted session's lines keep their 8a-1 form (also
// TestOrgDirector_AttachReplaceAndDetachByID).
func TestOrgDirectorStatus_ShowsHeadlessDirectors(t *testing.T) {
	const tail = " -- the registration does not tell whether its session is still running; inbox items it does not ack reach the human at the ack deadline"
	stateDir := t.TempDir()
	store := org.NewDirectorStore(stateDir)
	who := org.DirectorWho{User: "alice", Cwd: "/work"}
	d1 := seedHeadless(t, stateDir, "codex", "gpt-6-astra", "w-1", "", false)
	ts := d1.AttachedAt

	stdout, _ := runInboxOK(t, "director", "status", "--state-dir", stateDir)
	want := "director: attached d1 (driver=codex, headless starting, model=gpt-6-astra, workspace w-1, attached at " + ts + ", by alice)" + tail + "\n" +
		"enabled: true\nid: d1\ndriver: codex\nwake: pull\nattached_at: " + ts + "\nuser: alice\ncwd: /work\nherdr_pane: no\n" +
		"launch: herdr\nstate: starting\nmodel: gpt-6-astra\npermission_mode: autonomous\ntoken: " + d1.Token + "\nworkspace_id: w-1\n"
	if stdout != want {
		t.Errorf("status of a starting d1 =\n%s\nwant\n%s", stdout, want)
	}

	if err := store.RecordPane("d1", "p-1", who); err != nil {
		t.Fatal(err)
	}
	if ok, err := store.MarkLaunched("d1", who); err != nil || !ok {
		t.Fatalf("mark d1 launched: %v, %v", ok, err)
	}
	running := "d1 (running, pane p-1, workspace w-1)"
	if got, want := directorStatusLine(t, "--state-dir", stateDir),
		"director: attached d1 (driver=codex, headless running, model=gpt-6-astra, pane p-1, workspace w-1, attached at "+ts+", by alice)"+tail; got != want {
		t.Errorf("director line of a running d1:\n got %q\nwant %q", got, want)
	}

	d2, _, err := store.Attach(org.DirectorAttach{Driver: org.DirectorDriverClaude, Replace: true, User: "bob"})
	if err != nil {
		t.Fatal(err)
	}
	stdout, _ = runInboxOK(t, "director", "status", "--state-dir", stateDir)
	wantLine := directorAttachedLine("d2", "claude", d2.AttachedAt, "bob") + "; the headless director " + running +
		" is still open but not attached: stop it with ralph org director stop --id d1"
	if lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n"); lines[0] != wantLine || lines[len(lines)-1] != "open_headless: "+running ||
		strings.Contains(stdout, "launch:") {
		t.Errorf("status with d2 promoted over d1 =\n%s\nwant the line %q, d2's 8a-1 fields, and open_headless: %s", stdout, wantLine, running)
	}
	jsonOut, _ := runInboxOK(t, "director", "status", "--json", "--state-dir", stateDir)
	report := directorReportOf(t, jsonOut)
	if report.Registration == nil || report.Registration.ID != "d2" || report.Registration.Launch != "" || len(report.OpenHeadless) != 1 ||
		report.OpenHeadless[0].ID != "d1" || report.OpenHeadless[0].PaneID != "p-1" || report.OpenHeadless[0].WorkspaceID != "w-1" {
		t.Errorf("director status --json = %s; want d2 current and d1 in open_headless", jsonOut)
	}

	if _, err := store.Detach(org.DirectorDetach{ID: "d2", User: "bob"}); err != nil {
		t.Fatal(err)
	}
	seedHeadless(t, stateDir, "claude", "opus", "", "", false)
	if _, err := store.Detach(org.DirectorDetach{ID: "d3", User: "bob"}); err != nil {
		t.Fatal(err)
	}
	wantLine = directorNotAttachedLine + "; the headless directors " + running + ", d3 (starting) are still open but not attached: " +
		"stop each with ralph org director stop --id <id>"
	if got := directorStatusLine(t, "--state-dir", stateDir); got != wantLine {
		t.Errorf("director line with two open headless directors:\n got %q\nwant %q", got, wantLine)
	}
}

// D7 through the CLI: `ralph org spawn --role director` is refused before
// the --model fallback, so with no --model it prints no fallback warning,
// prints the refusal as a rejected spawn, exits 1, and writes nothing and
// runs neither herdr nor agmsg.
func TestOrgSpawn_RoleDirectorIsRefusedBeforeTheModelFallback(t *testing.T) {
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	stateDir := t.TempDir()
	stdout, stderr, err := runOrgCmdStreams(t, "spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "director",
		"--driver", "claude", "--cwd", t.TempDir(), "--state-dir", stateDir)
	want := `org: role "director" is not a seat role: the director is not an org seat; launch a headless director with ralph org director start, ` +
		"or promote an interactive session with ralph org director attach"
	if err == nil || err.Error() != want {
		t.Fatalf("spawn --role director = %v; want %q", err, want)
	}
	if stdout != "rejected: "+want+"\n" {
		t.Errorf("stdout = %q; want the rejected line", stdout)
	}
	if stderr != "" {
		t.Errorf("stderr = %q; want no --model fallback warning", stderr)
	}
	checkNothingStarted(t, stateDir, herdrLog, agmsgLog)
}
