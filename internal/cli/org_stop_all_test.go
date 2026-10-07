package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org stop --all`, `ralph org disband --all`, --force, and
// the deferred close of the caller's own pane / workspace (plan
// docs/plans/active/2026-10-07-org-stop-all.md, AC4-AC7, AC10, AC12, S5).
// The ledger is seeded directly so every seat gets its own pane id: the herdr
// stub answers every `tab create` with the same pane id, which would make
// "fail one seat's pane close" impossible to express.

// appendSeedEvent appends ev, stamped with the current time, to the manifest
// under stateDir.
func appendSeedEvent(t *testing.T, stateDir string, ev org.ManifestEvent) {
	t.Helper()
	ev.TS = time.Now().UTC().Format(time.RFC3339)
	if err := org.NewManifestStoreAtPath(org.ManifestPathIn(stateDir)).Append(ev); err != nil {
		t.Fatalf("seed manifest event %+v: %v", ev, err)
	}
}

// seedWorkspace records workspaceID as orgID's herdr workspace, the way
// spawn's org_workspace_created does.
func seedWorkspace(t *testing.T, stateDir, orgID, workspaceID string) {
	t.Helper()
	appendSeedEvent(t, stateDir, org.ManifestEvent{OrgID: orgID, Event: org.EventOrgWorkspaceCreated, PaneID: workspaceID})
}

// seedSeat records seatID of orgID as spawned in paneID, joined to the org's
// agmsg team.
func seedSeat(t *testing.T, stateDir, orgID, seatID, paneID string) {
	t.Helper()
	appendSeedEvent(t, stateDir, org.ManifestEvent{
		OrgID: orgID, SeatID: seatID, Event: org.EventSpawned,
		Role: "worker", Driver: "claude", Model: "sonnet", PaneID: paneID, AgmsgTeam: "ralph-" + orgID,
	})
}

// runOrgCmdStreams runs `ralph org <args...>` in-process with stdout and
// stderr captured separately.
func runOrgCmdStreams(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := NewRootCmd()
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetArgs(append([]string{"org"}, args...))
	err = root.Execute()
	return out.String(), errBuf.String(), err
}

// outputLines splits command output into its non-empty lines.
func outputLines(s string) []string {
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

// activeSeats returns `<org_id>/<seat_id>` of every real seat the manifest
// under stateDir still has active.
func activeSeats(t *testing.T, stateDir string) []string {
	t.Helper()
	var active []string
	for _, s := range org.Roster(readManifestEvents(t, org.ManifestPathIn(stateDir)), org.RosterOptions{}) {
		if s.Active {
			active = append(active, s.OrgID+"/"+s.SeatID)
		}
	}
	return active
}

// lastSeatEvent returns the last manifest event of orgID/seatID.
func lastSeatEvent(t *testing.T, stateDir, orgID, seatID string) org.ManifestEvent {
	t.Helper()
	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].OrgID == orgID && events[i].SeatID == seatID {
			return events[i]
		}
	}
	t.Fatalf("no manifest event for %s/%s", orgID, seatID)
	return org.ManifestEvent{}
}

// realDisbandedOrgs returns the org_ids with a real (non-dry-run)
// `disbanded` event, in manifest order.
func realDisbandedOrgs(t *testing.T, stateDir string) []string {
	t.Helper()
	var orgs []string
	for _, ev := range readManifestEvents(t, org.ManifestPathIn(stateDir)) {
		if ev.Event == org.EventDisbanded && ev.SeatID == "" && !ev.DryRun {
			orgs = append(orgs, ev.OrgID)
		}
	}
	return orgs
}

func TestOrgStopAll_StopsEveryActiveSeatAcrossOrgs(t *testing.T) {
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	seedWorkspace(t, stateDir, "org-a", "ws-a")
	seedSeat(t, stateDir, "org-a", "seat-1", "p-a1")
	seedSeat(t, stateDir, "org-a", "seat-2", "p-a2")
	seedWorkspace(t, stateDir, "org-b", "ws-b")
	seedSeat(t, stateDir, "org-b", "seat-1", "p-b1")
	// org-c's pane is already gone: herdr's pane_not_found counts as closed.
	seedSeat(t, stateDir, "org-c", "seat-x", "p-c1")
	t.Setenv("ORG_STUB_CLOSE_NOT_FOUND_IDS", "p-c1")

	stdout, stderr, err := runOrgCmdStreams(t, "stop", "--all", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("stop --all failed: %v (stdout: %s, stderr: %s)", err, stdout, stderr)
	}
	want := []string{"stopped seat org-a/seat-1", "stopped seat org-a/seat-2", "stopped seat org-b/seat-1", "stopped seat org-c/seat-x"}
	if got := outputLines(stdout); !slices.Equal(got, want) {
		t.Errorf("stdout lines = %q, want %q", got, want)
	}
	if stderr != "" {
		t.Errorf("expected nothing on stderr, got: %s", stderr)
	}

	herdrLines := readLogLines(t, herdrLog)
	for _, pane := range []string{"p-a1", "p-a2", "p-b1", "p-c1"} {
		if !containsLine(herdrLines, "pane close "+pane) {
			t.Errorf("expected a 'pane close %s' call, got herdr log: %v", pane, herdrLines)
		}
	}
	// stop never closes workspaces; that is disband's job.
	if n := countLinesWithPrefix(herdrLines, "workspace close"); n != 0 {
		t.Errorf("expected no workspace close from stop --all, got %d (log: %v)", n, herdrLines)
	}
	if agmsgLines := readLogLines(t, agmsgLog); !containsLine(agmsgLines, "ralph-org-b seat-1") {
		t.Errorf("expected org-b/seat-1 to leave agmsg, got agmsg log: %v", agmsgLines)
	}
	if active := activeSeats(t, stateDir); len(active) != 0 {
		t.Errorf("expected no active seat after stop --all, got %v", active)
	}
	if ev := lastSeatEvent(t, stateDir, "org-c", "seat-x"); ev.Event != org.EventStopped || !strings.Contains(ev.Details, "pane=already closed") {
		t.Errorf("expected org-c/seat-x stopped with pane=already closed, got %s %q", ev.Event, ev.Details)
	}
}

func TestOrgStopAll_OneCloseFails_ExitsOneThenRerunStopsIt(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	seedSeat(t, stateDir, "org-a", "seat-1", "p-a1")
	seedSeat(t, stateDir, "org-a", "seat-2", "p-a2")
	seedSeat(t, stateDir, "org-b", "seat-1", "p-b1")
	t.Setenv("ORG_STUB_CLOSE_FAIL_IDS", "p-a2")

	stdout, stderr, err := runOrgCmdStreams(t, "stop", "--all", "--state-dir", stateDir)
	if err == nil {
		t.Fatalf("expected a non-zero exit when one pane cannot be closed (stdout: %s, stderr: %s)", stdout, stderr)
	}
	want := []string{"stopped seat org-a/seat-1", "stopped seat org-b/seat-1"}
	if got := outputLines(stdout); !slices.Equal(got, want) {
		t.Errorf("stdout lines = %q, want %q", got, want)
	}
	if !strings.Contains(stderr, "seat org-a/seat-2 not stopped: ") || !strings.Contains(stderr, "stub failure: pane close p-a2") {
		t.Errorf("expected stderr to name org-a/seat-2 and the close failure, got: %s", stderr)
	}
	if strings.Contains(stderr, "warning:") {
		t.Errorf("a failure without --force must not print as a warning, got: %s", stderr)
	}
	if active := activeSeats(t, stateDir); !slices.Equal(active, []string{"org-a/seat-2"}) {
		t.Errorf("expected only org-a/seat-2 left active, got %v", active)
	}
	if ev := lastSeatEvent(t, stateDir, "org-a", "seat-2"); ev.Event != org.EventStopFailed {
		t.Errorf("expected org-a/seat-2's last event to be %s, got %s", org.EventStopFailed, ev.Event)
	}
	statusOut, err := runOrgCmd(t, "status", "--org-id", "org-a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if !strings.Contains(statusOut, "seat-2\t") || !strings.Contains(statusOut, "(active)") {
		t.Errorf("expected status to show seat-2 still active, got: %s", statusOut)
	}

	// herdr answers again: the same command picks the seat up.
	t.Setenv("ORG_STUB_CLOSE_FAIL_IDS", "")
	stdout, stderr, err = runOrgCmdStreams(t, "stop", "--all", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("rerun of stop --all failed: %v (stdout: %s, stderr: %s)", err, stdout, stderr)
	}
	if got := outputLines(stdout); !slices.Equal(got, []string{"stopped seat org-a/seat-2"}) {
		t.Errorf("rerun stdout lines = %q, want only org-a/seat-2", got)
	}
	if active := activeSeats(t, stateDir); len(active) != 0 {
		t.Errorf("expected no active seat after the rerun, got %v", active)
	}
}

func TestOrgDisbandAll_DisbandsEveryOrgThatNeedsIt(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	seedWorkspace(t, stateDir, "org-a", "ws-a")
	seedSeat(t, stateDir, "org-a", "seat-1", "p-a1")
	seedWorkspace(t, stateDir, "org-b", "ws-b")
	seedSeat(t, stateDir, "org-b", "seat-1", "p-b1")
	seedSeat(t, stateDir, "org-b", "seat-2", "p-b2")
	// org-c was disbanded already (its seat stopped, no workspace left open).
	seedSeat(t, stateDir, "org-c", "seat-1", "p-c1")
	appendSeedEvent(t, stateDir, org.ManifestEvent{OrgID: "org-c", SeatID: "seat-1", Event: org.EventStopped, PaneID: "p-c1"})
	appendSeedEvent(t, stateDir, org.ManifestEvent{OrgID: "org-c", Event: org.EventDisbanded})

	stdout, stderr, err := runOrgCmdStreams(t, "disband", "--all", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("disband --all failed: %v (stdout: %s, stderr: %s)", err, stdout, stderr)
	}
	want := []string{
		"stopped seat org-a/seat-1", "stopped seat org-b/seat-1", "stopped seat org-b/seat-2",
		"disbanded org org-a", "disbanded org org-b",
	}
	if got := outputLines(stdout); !slices.Equal(got, want) {
		t.Errorf("stdout lines = %q, want %q", got, want)
	}
	if stderr != "" {
		t.Errorf("expected nothing on stderr, got: %s", stderr)
	}
	herdrLines := readLogLines(t, herdrLog)
	for _, call := range []string{"workspace close ws-a", "workspace close ws-b"} {
		if !containsLine(herdrLines, call) {
			t.Errorf("expected %q, got herdr log: %v", call, herdrLines)
		}
	}
	for _, l := range herdrLines {
		if strings.Contains(l, "p-c1") {
			t.Errorf("expected no herdr call for the already disbanded org-c, got %q", l)
		}
	}
	if got := realDisbandedOrgs(t, stateDir); !slices.Equal(got, []string{"org-c", "org-a", "org-b"}) {
		t.Errorf("disbanded events = %v, want org-c (seeded) then org-a and org-b", got)
	}
}

func TestOrgDisbandAll_OneOrgFails_ExitsOneAndNextRunRetriesIt(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	seedWorkspace(t, stateDir, "org-a", "ws-a")
	seedSeat(t, stateDir, "org-a", "seat-1", "p-a1")
	seedWorkspace(t, stateDir, "org-b", "ws-b")
	seedSeat(t, stateDir, "org-b", "seat-1", "p-b1")
	t.Setenv("ORG_STUB_CLOSE_FAIL_IDS", "ws-b")

	stdout, stderr, err := runOrgCmdStreams(t, "disband", "--all", "--state-dir", stateDir)
	if err == nil {
		t.Fatalf("expected a non-zero exit when org-b's workspace cannot be closed (stdout: %s, stderr: %s)", stdout, stderr)
	}
	if !strings.Contains(err.Error(), "org-b") || strings.Contains(err.Error(), "org-a") {
		t.Errorf("expected the error to name org-b as not disbanded and not org-a, got: %v", err)
	}
	stdoutLines := outputLines(stdout)
	if !containsLine(stdoutLines, "disbanded org org-a") || containsLine(stdoutLines, "disbanded org org-b") {
		t.Errorf("expected org-a disbanded and org-b not, got stdout: %s", stdout)
	}
	if !strings.Contains(stderr, "workspace org-b/ws-b not closed: ") || !strings.Contains(stderr, "stub failure: workspace close ws-b") {
		t.Errorf("expected stderr to name org-b/ws-b and the close failure, got: %s", stderr)
	}
	if got := realDisbandedOrgs(t, stateDir); !slices.Equal(got, []string{"org-a"}) {
		t.Errorf("disbanded events = %v, want only org-a", got)
	}

	t.Setenv("ORG_STUB_CLOSE_FAIL_IDS", "")
	stdout, stderr, err = runOrgCmdStreams(t, "disband", "--all", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("rerun of disband --all failed: %v (stdout: %s, stderr: %s)", err, stdout, stderr)
	}
	// org-b's seat was stopped by the first run; only its workspace was left.
	if got := outputLines(stdout); !slices.Equal(got, []string{"disbanded org org-b"}) {
		t.Errorf("rerun stdout lines = %q, want only org-b disbanded", got)
	}
	if got := realDisbandedOrgs(t, stateDir); !slices.Equal(got, []string{"org-a", "org-b"}) {
		t.Errorf("disbanded events = %v, want org-a then org-b", got)
	}
}

// TestOrgDisband_OneOrgSeatFails_NoDisbandedLine is the single-org disband:
// a seat that cannot be stopped keeps the org from being disbanded, the
// command no longer prints `disbanded org` regardless, and that seat is not
// listed as stopped.
func TestOrgDisband_OneOrgSeatFails_NoDisbandedLine(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	seedWorkspace(t, stateDir, "org-a", "ws-a")
	seedSeat(t, stateDir, "org-a", "seat-1", "p-a1")
	seedSeat(t, stateDir, "org-a", "seat-2", "p-a2")
	t.Setenv("ORG_STUB_CLOSE_FAIL_IDS", "p-a1")

	stdout, stderr, err := runOrgCmdStreams(t, "disband", "--org-id", "org-a", "--state-dir", stateDir)
	if err == nil {
		t.Fatalf("expected a non-zero exit (stdout: %s, stderr: %s)", stdout, stderr)
	}
	if !strings.Contains(err.Error(), `org_id "org-a" not disbanded`) {
		t.Errorf("expected the error to say org-a was not disbanded, got: %v", err)
	}
	if got := outputLines(stdout); !slices.Equal(got, []string{`stopped seat "seat-2"`}) {
		t.Errorf("stdout lines = %q, want only seat-2 stopped and no disbanded line", got)
	}
	if !strings.Contains(stderr, `seat "seat-1" not stopped: `) || !strings.Contains(stderr, `workspace "ws-a" not closed: `) {
		t.Errorf("expected stderr to list seat-1 and the workspace left open, got: %s", stderr)
	}
	if n := countLinesWithPrefix(readLogLines(t, herdrLog), "workspace close"); n != 0 {
		t.Errorf("expected the workspace left open while a seat runs, got %d workspace close call(s)", n)
	}
	if got := realDisbandedOrgs(t, stateDir); len(got) != 0 {
		t.Errorf("expected no disbanded event, got %v", got)
	}
}

func TestOrgStopAndDisband_Force_FailuresAreWarningsExitZero(t *testing.T) {
	t.Run("stop --all", func(t *testing.T) {
		setupOrgStubPATH(t)
		stateDir := filepath.Join(t.TempDir(), "state")
		seedSeat(t, stateDir, "org-a", "seat-1", "p-a1")
		seedSeat(t, stateDir, "org-a", "seat-2", "p-a2")
		t.Setenv("ORG_STUB_CLOSE_FAIL_IDS", "p-a2")

		stdout, stderr, err := runOrgCmdStreams(t, "stop", "--all", "--force", "--state-dir", stateDir)
		if err != nil {
			t.Fatalf("stop --all --force must exit 0, got: %v (stderr: %s)", err, stderr)
		}
		if got := outputLines(stdout); !slices.Equal(got, []string{"stopped seat org-a/seat-1"}) {
			t.Errorf("stdout lines = %q, want only the cleanly stopped seat", got)
		}
		if !strings.Contains(stderr, "warning: seat org-a/seat-2 recorded stopped (--force), but its pane was not closed: ") {
			t.Errorf("expected a forced warning for org-a/seat-2, got: %s", stderr)
		}
		if active := activeSeats(t, stateDir); len(active) != 0 {
			t.Errorf("expected --force to record every seat stopped, got active %v", active)
		}
		if ev := lastSeatEvent(t, stateDir, "org-a", "seat-2"); ev.Event != org.EventStopped || !strings.Contains(ev.Details, "pane=close failed (forced)") {
			t.Errorf("expected org-a/seat-2 stopped with a forced note, got %s %q", ev.Event, ev.Details)
		}
	})

	t.Run("stop --seat", func(t *testing.T) {
		setupOrgStubPATH(t)
		stateDir := filepath.Join(t.TempDir(), "state")
		seedSeat(t, stateDir, "org-a", "seat-1", "p-a1")
		t.Setenv("ORG_STUB_CLOSE_FAIL_IDS", "p-a1")

		stdout, stderr, err := runOrgCmdStreams(t, "stop", "--org-id", "org-a", "--seat", "seat-1", "--force", "--state-dir", stateDir)
		if err != nil {
			t.Fatalf("stop --force must exit 0, got: %v (stderr: %s)", err, stderr)
		}
		if stdout != "" {
			t.Errorf("expected no stopped line for a seat whose pane was not closed, got: %s", stdout)
		}
		if !strings.Contains(stderr, `warning: seat "seat-1" recorded stopped (--force), but its pane was not closed: `) {
			t.Errorf("expected a forced warning for seat-1, got: %s", stderr)
		}
	})

	for _, tc := range []struct {
		name string
		args []string
		// wantDisbanded is the stdout line for the disbanded org.
		wantDisbanded, seatLabel, workspaceLabel string
	}{
		{"disband --all", []string{"disband", "--all"}, "disbanded org org-a", "org-a/seat-1", "org-a/ws-a"},
		{"disband --org-id", []string{"disband", "--org-id", "org-a"}, `disbanded org "org-a"`, `"seat-1"`, `"ws-a"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupOrgStubPATH(t)
			stateDir := filepath.Join(t.TempDir(), "state")
			seedWorkspace(t, stateDir, "org-a", "ws-a")
			seedSeat(t, stateDir, "org-a", "seat-1", "p-a1")
			t.Setenv("ORG_STUB_CLOSE_FAIL_IDS", "p-a1 ws-a")

			stdout, stderr, err := runOrgCmdStreams(t, append(tc.args, "--force", "--state-dir", stateDir)...)
			if err != nil {
				t.Fatalf("%s --force must exit 0, got: %v (stderr: %s)", tc.name, err, stderr)
			}
			if got := outputLines(stdout); !slices.Equal(got, []string{tc.wantDisbanded}) {
				t.Errorf("stdout lines = %q, want only %q", got, tc.wantDisbanded)
			}
			for _, want := range []string{
				"warning: seat " + tc.seatLabel + " recorded stopped (--force)",
				"warning: workspace " + tc.workspaceLabel + " recorded closed (--force)",
			} {
				if !strings.Contains(stderr, want) {
					t.Errorf("expected %q on stderr, got: %s", want, stderr)
				}
			}
			if got := realDisbandedOrgs(t, stateDir); !slices.Equal(got, []string{"org-a"}) {
				t.Errorf("expected --force to record org-a disbanded, got %v", got)
			}
		})
	}
}

// TestOrgStopDisbandAll_ConflictingFlags_RejectedWithoutChanges covers AC6:
// --all next to a flag naming one org or seat is refused before the manifest
// or any driver is touched, and without --all, --org-id stays required.
func TestOrgStopDisbandAll_ConflictingFlags_RejectedWithoutChanges(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"stop --all --org-id", []string{"stop", "--all", "--org-id", "org-a"}, "--all cannot be combined with --org-id"},
		{"stop --all --seat", []string{"stop", "--all", "--seat", "seat-1"}, "--all cannot be combined with --seat"},
		{"disband --all --org-id", []string{"disband", "--all", "--org-id", "org-a"}, "--all cannot be combined with --org-id"},
		{"stop without --all or --org-id", []string{"stop", "--seat", "seat-1"}, "--org-id is required"},
		{"disband without --all or --org-id", []string{"disband"}, "--org-id is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, agmsgLog := setupOrgStubPATH(t)
			stateDir := filepath.Join(t.TempDir(), "state")
			seedWorkspace(t, stateDir, "org-a", "ws-a")
			seedSeat(t, stateDir, "org-a", "seat-1", "p-a1")
			before, err := os.ReadFile(org.ManifestPathIn(stateDir))
			if err != nil {
				t.Fatal(err)
			}

			stdout, stderr, err := runOrgCmdStreams(t, append(tc.args, "--state-dir", stateDir)...)
			if err == nil {
				t.Fatalf("expected an error (stdout: %s, stderr: %s)", stdout, stderr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q should contain %q", err.Error(), tc.wantErr)
			}
			after, err := os.ReadFile(org.ManifestPathIn(stateDir))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Errorf("expected the manifest unchanged, got:\n%s", after)
			}
			if lines := append(readLogLines(t, herdrLog), readLogLines(t, agmsgLog)...); len(lines) != 0 {
				t.Errorf("expected no herdr or agmsg call, got %v", lines)
			}
		})
	}
}

// TestOrgStopDisbandAll_DryRun_RecordsWithoutDriverCalls covers AC7.
func TestOrgStopDisbandAll_DryRun_RecordsWithoutDriverCalls(t *testing.T) {
	for _, tc := range []struct {
		verb      string
		wantLines []string
		wantEvent string
	}{
		{"stop", []string{"stopped seat org-a/seat-1", "stopped seat org-b/seat-1"}, org.EventStopped},
		{"disband", []string{"disbanded org org-a", "disbanded org org-b"}, org.EventDisbanded},
	} {
		t.Run(tc.verb, func(t *testing.T) {
			herdrLog, agmsgLog := setupOrgStubPATH(t)
			stateDir := filepath.Join(t.TempDir(), "state")
			seedWorkspace(t, stateDir, "org-a", "ws-a")
			seedSeat(t, stateDir, "org-a", "seat-1", "p-a1")
			seedSeat(t, stateDir, "org-b", "seat-1", "p-b1")
			seeded := len(readManifestEvents(t, org.ManifestPathIn(stateDir)))

			stdout, stderr, err := runOrgCmdStreams(t, tc.verb, "--all", "--dry-run", "--state-dir", stateDir)
			if err != nil {
				t.Fatalf("%s --all --dry-run failed: %v (stderr: %s)", tc.verb, err, stderr)
			}
			if got := outputLines(stdout); !slices.Equal(got, tc.wantLines) {
				t.Errorf("stdout lines = %q, want %q", got, tc.wantLines)
			}
			if lines := append(readLogLines(t, herdrLog), readLogLines(t, agmsgLog)...); len(lines) != 0 {
				t.Errorf("expected no herdr or agmsg call in a dry run, got %v", lines)
			}
			added := readManifestEvents(t, org.ManifestPathIn(stateDir))[seeded:]
			if len(added) != 2 {
				t.Fatalf("expected 2 dry-run events, got %+v", added)
			}
			for _, ev := range added {
				if !ev.DryRun || ev.Event != tc.wantEvent {
					t.Errorf("expected a dry-run %s event, got %+v", tc.wantEvent, ev)
				}
			}
			if active := activeSeats(t, stateDir); !slices.Equal(active, []string{"org-a/seat-1", "org-b/seat-1"}) {
				t.Errorf("a dry run must leave the real seats active, got %v", active)
			}
		})
	}
}

func TestOrgStopDisbandAll_EmptyLedger_SaysSoAndExitsZero(t *testing.T) {
	for _, tc := range []struct{ verb, want string }{
		{"stop", "no active seats"},
		{"disband", "no orgs to disband"},
	} {
		t.Run(tc.verb, func(t *testing.T) {
			herdrLog, _ := setupOrgStubPATH(t)
			stateDir := filepath.Join(t.TempDir(), "state")

			stdout, stderr, err := runOrgCmdStreams(t, tc.verb, "--all", "--state-dir", stateDir)
			if err != nil {
				t.Fatalf("%s --all on an empty ledger failed: %v (stderr: %s)", tc.verb, err, stderr)
			}
			if got := outputLines(stdout); !slices.Equal(got, []string{tc.want}) {
				t.Errorf("stdout lines = %q, want %q", got, tc.want)
			}
			if lines := readLogLines(t, herdrLog); len(lines) != 0 {
				t.Errorf("expected no herdr call, got %v", lines)
			}
		})
	}
}

// TestOrgStop_ErrorPrefixAppearsOnce pins the fix for the doubled
// "org: stop: org: stop: ..." a single-seat stop error used to print.
func TestOrgStop_ErrorPrefixAppearsOnce(t *testing.T) {
	for _, tc := range []struct {
		name, seat, failIDs string
	}{
		{"unknown seat", "never-spawned", ""},
		{"pane close failure", "seat-1", "p-a1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupOrgStubPATH(t)
			stateDir := filepath.Join(t.TempDir(), "state")
			seedSeat(t, stateDir, "org-a", "seat-1", "p-a1")
			t.Setenv("ORG_STUB_CLOSE_FAIL_IDS", tc.failIDs)

			_, _, err := runOrgCmdStreams(t, "stop", "--org-id", "org-a", "--seat", tc.seat, "--state-dir", stateDir)
			if err == nil {
				t.Fatal("expected an error")
			}
			if msg := err.Error(); !strings.HasPrefix(msg, "org: stop: ") || strings.Count(msg, "org: stop: ") != 1 {
				t.Errorf("expected exactly one leading %q, got %q", "org: stop: ", msg)
			}
		})
	}
}

func TestWithPrefixOnce(t *testing.T) {
	prefixed := errors.New("org: stop: seat \"s\" not found")
	bare := errors.New("org: append to /x/manifest.jsonl: disk full")
	for _, tc := range []struct {
		err  error
		want string
	}{
		{prefixed, prefixed.Error()},
		{bare, "org: stop: " + bare.Error()},
	} {
		got := withPrefixOnce("org: stop: ", tc.err)
		if got.Error() != tc.want {
			t.Errorf("withPrefixOnce(%q) = %q, want %q", tc.err, got, tc.want)
		}
		if !errors.Is(got, tc.err) {
			t.Errorf("withPrefixOnce(%q) no longer wraps the original error", tc.err)
		}
	}
}

// appendFileWriter appends every Write to path. Pointed at the file the herdr
// stub logs its argv to, it makes one file record the command's output and
// its herdr calls in the order they happened.
type appendFileWriter struct{ path string }

func (w appendFileWriter) Write(p []byte) (n int, err error) {
	f, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	return f.Write(p)
}

// runOrgCmdTo runs `ralph org <args...>` in-process with stdout and stderr
// both written to w.
func runOrgCmdTo(t *testing.T, w io.Writer, args ...string) error {
	t.Helper()
	root := NewRootCmd()
	root.SetOut(w)
	root.SetErr(w)
	root.SetArgs(append([]string{"org"}, args...))
	return root.Execute()
}

// TestOrgStopDisband_OwnPaneOrWorkspace_ClosedLastAfterOutput covers the CLI
// side of AC12: when HERDR_PANE_ID / HERDR_WORKSPACE_ID name a pane or
// workspace the command closes, that close is the command's last herdr call
// and comes after everything it printed. The herdr stub and the command's
// output share one log file, so the order of its lines is the order things
// happened.
func TestOrgStopDisband_OwnPaneOrWorkspace_ClosedLastAfterOutput(t *testing.T) {
	cases := []struct {
		name string
		args []string
		// ownPane / ownWorkspace are the HERDR_PANE_ID / HERDR_WORKSPACE_ID.
		ownPane, ownWorkspace string
		failIDs               string
		wantLast              string
		// wantBefore must all appear before wantLast.
		wantBefore []string
		// wantAbsent must not appear at all.
		wantAbsent []string
		wantErr    string
	}{
		{
			name: "stop --all", args: []string{"stop", "--all"}, ownPane: "p-a1",
			wantLast:   "pane close p-a1",
			wantBefore: []string{"pane close p-b1", "stopped seat org-a/seat-1", "stopped seat org-b/seat-1"},
			wantAbsent: []string{"pane send-keys p-a1 C-c"},
		},
		{
			name: "stop --seat", args: []string{"stop", "--org-id", "org-a", "--seat", "seat-1"}, ownPane: "p-a1",
			wantLast:   "pane close p-a1",
			wantBefore: []string{`stopped seat "seat-1"`},
			wantAbsent: []string{"pane send-keys p-a1 C-c"},
		},
		{
			name: "disband --all", args: []string{"disband", "--all"}, ownPane: "p-a1", ownWorkspace: "ws-a",
			wantLast:   "workspace close ws-a",
			wantBefore: []string{"workspace close ws-b", "stopped seat org-a/seat-1", "disbanded org org-a", "disbanded org org-b"},
			wantAbsent: []string{"pane close p-a1"},
		},
		{
			name: "disband --org-id", args: []string{"disband", "--org-id", "org-a"}, ownPane: "p-a1", ownWorkspace: "ws-a",
			wantLast:   "workspace close ws-a",
			wantBefore: []string{"stopped seat \"seat-1\"", `disbanded org "org-a"`},
			wantAbsent: []string{"pane close p-a1"},
		},
		{
			// The deferred close itself fails: still last, and exit 1.
			name: "stop --seat, own pane close fails", args: []string{"stop", "--org-id", "org-a", "--seat", "seat-1"}, ownPane: "p-a1",
			failIDs:    "p-a1",
			wantLast:   "pane close p-a1",
			wantBefore: []string{`stopped seat "seat-1"`},
			wantErr:    `org: close own pane "p-a1"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, _ := setupOrgStubPATH(t)
			stateDir := filepath.Join(t.TempDir(), "state")
			seedWorkspace(t, stateDir, "org-a", "ws-a")
			seedSeat(t, stateDir, "org-a", "seat-1", "p-a1")
			seedWorkspace(t, stateDir, "org-b", "ws-b")
			seedSeat(t, stateDir, "org-b", "seat-1", "p-b1")
			t.Setenv("HERDR_PANE_ID", tc.ownPane)
			t.Setenv("HERDR_WORKSPACE_ID", tc.ownWorkspace)
			t.Setenv("ORG_STUB_CLOSE_FAIL_IDS", tc.failIDs)

			err := runOrgCmdTo(t, appendFileWriter{path: herdrLog}, append(tc.args, "--state-dir", stateDir)...)
			lines := readLogLines(t, herdrLog)
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("expected success, got: %v (log: %v)", err, lines)
			case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
				t.Fatalf("expected an error containing %q, got: %v (log: %v)", tc.wantErr, err, lines)
			}
			if len(lines) == 0 || lines[len(lines)-1] != tc.wantLast {
				t.Fatalf("expected the last line to be %q, got log:\n%s", tc.wantLast, strings.Join(lines, "\n"))
			}
			if n := slices.Index(lines, tc.wantLast); n != len(lines)-1 {
				t.Errorf("expected %q exactly once, at the end; first seen at line %d of %d", tc.wantLast, n+1, len(lines))
			}
			for _, want := range tc.wantBefore {
				if !slices.Contains(lines[:len(lines)-1], want) {
					t.Errorf("expected %q before %q, got log:\n%s", want, tc.wantLast, strings.Join(lines, "\n"))
				}
			}
			for _, absent := range tc.wantAbsent {
				if slices.Contains(lines, absent) {
					t.Errorf("expected no %q, got log:\n%s", absent, strings.Join(lines, "\n"))
				}
			}
		})
	}
}

// TestOrgStopAll_CLI_CodexModelMismatch_PrintsWarning is the stop --all
// counterpart of TestOrgStop_CLI_CodexModelMismatch_PrintsWarning: the
// warning names the seat as <org_id>/<seat_id>.
func TestOrgStopAll_CLI_CodexModelMismatch_PrintsWarning(t *testing.T) {
	setupOrgStubPATH(t)
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	configPath := writeCodexOrgConfig(t, dir, 3, "implementer")
	sessionsDir := filepath.Join(dir, "codex-sessions")
	setCodexSessionsDirOverride(t, sessionsDir)

	if out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "implementer",
		"--driver", "codex", "--model", codexOrgTestCommandedModel, "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--config", configPath,
	); err != nil {
		t.Fatalf("spawn failed: %v (output: %s)", err, out)
	}
	setCodexModelObserveTimeoutOverride(t, testOrgCodexObserveGenerousBudget)
	writeCliCodexFixture(t, sessionsDir, codexPromptPathFor(stateDir, "org-a", "seat-1"), time.Now().Add(time.Second), codexOrgTestReportedModel)

	stdout, stderr, err := runOrgCmdStreams(t, "stop", "--all", "--state-dir", stateDir, "--config", configPath)
	if err != nil {
		t.Fatalf("stop --all failed: %v (stderr: %s)", err, stderr)
	}
	if got := outputLines(stdout); !slices.Equal(got, []string{"stopped seat org-a/seat-1"}) {
		t.Errorf("stdout lines = %q, want org-a/seat-1 stopped", got)
	}
	want := fmt.Sprintf(`warning: seat "org-a/seat-1" was started with --model %s, but codex reports it is running %s.`,
		codexOrgTestCommandedModel, codexOrgTestReportedModel)
	if !strings.Contains(stderr, want) {
		t.Errorf("expected %q on stderr, got: %s", want, stderr)
	}
}
