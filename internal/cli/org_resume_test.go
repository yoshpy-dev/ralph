package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for `ralph org resume` through the CLI (plan
// docs/plans/active/2026-11-07-org-seat-resume.md, S6, D6 and D7; spec
// docs/specs/2026-11-07-org-seat-resume.md, FR-1 and FR-7). The ledger is
// seeded directly (appendSeedEvent, seedHeadless), and herdr is the stub of
// setupOrgStubPATH behind the resume herdr of stubResumeHerdr, which answers
// the agent status and the screen of each seat. The seat logic itself is
// tested in internal/org/resume_test.go; these pin the lines, the summary,
// the notes, and the exit code.

// resumeHerdrScript is the herdr stubResumeHerdr puts in front of the stub
// herdr (@STUB@ is the stub's path, quoted). It logs the calls it answers
// itself to the stub's log, as the stub logs its own, and hands every other
// call to the stub:
//   - `agent get <name>` answers herdr 0.7.5's JSON with the agent_status
//     in status-<name>, and fails when there is no such file.
//   - `pane read <pane> ...` prints screen-<pane>, or typed-screen-<pane>
//     once a `pane send-text` to <pane> succeeded.
//   - `pane send-text <pane> ...` fails when fail-text-<pane> exists, and
//     otherwise marks <pane> typed.
const resumeHerdrScript = `#!/bin/sh
s="${ORG_RESUME_STUB:?set by stubResumeHerdr}"
case "$1 $2" in
  "agent get" | "pane read" | "pane send-text")
    if [ -n "$ORG_HERDR_LOG" ]; then
      printf '%s\n' "$*" >> "$ORG_HERDR_LOG"
    fi ;;
esac
case "$1 $2" in
  "agent get")
    if [ ! -f "$s/status-$3" ]; then
      echo "stub failure: agent get $3" >&2
      exit 1
    fi
    printf '{"id":"cli:agent:get","result":{"agent":{"agent_status":"%s","name":"%s"},"type":"agent_info"}}\n' "$(cat "$s/status-$3")" "$3"
    exit 0 ;;
  "pane read")
    if [ -f "$s/typed-$3" ]; then
      cat "$s/typed-screen-$3"
    else
      cat "$s/screen-$3"
    fi
    exit 0 ;;
  "pane send-text")
    if [ -f "$s/fail-text-$3" ]; then
      echo "stub failure: pane send-text $3" >&2
      exit 1
    fi
    : > "$s/typed-$3"
    exit 0 ;;
esac
exec @STUB@ "$@"
`

// resumeStub is the state directory of the herdr stubResumeHerdr puts on
// PATH.
type resumeStub struct{ dir string }

// stubResumeHerdr runs setupOrgStubPATH, puts the herdr of resumeHerdrScript
// in front of its stub, and returns the stub's log and the state of the
// resume herdr. Like stubHerdrAgentNotFound, it fails the test when the
// herdr PATH finds is not under this test's temp dir, so no call reaches a
// real herdr.
func stubResumeHerdr(t *testing.T) (herdrLog string, s resumeStub) {
	t.Helper()
	herdrLog, _ = setupOrgStubPATH(t)
	stub, err := exec.LookPath("herdr")
	if err != nil {
		t.Fatalf("look up the stub herdr: %v", err)
	}
	dir := t.TempDir()
	if root := filepath.Dir(dir) + string(filepath.Separator); !strings.HasPrefix(stub, root) {
		t.Fatalf("the herdr on PATH is %s, not a stub under this test's temp dir %s", stub, root)
	}
	writeStubScript(t, filepath.Join(dir, "herdr"), strings.ReplaceAll(resumeHerdrScript, "@STUB@", shellQuoteForTest(stub)))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	s = resumeStub{dir: t.TempDir()}
	t.Setenv("ORG_RESUME_STUB", s.dir)
	return herdrLog, s
}

// agent makes `herdr agent get <name>` report status.
func (s resumeStub) agent(t *testing.T, name, status string) {
	t.Helper()
	writeTestFile(t, filepath.Join(s.dir, "status-"+name), status)
}

// screen makes `herdr pane read <pane>` print screen, and typed once a
// `pane send-text` to pane succeeded.
func (s resumeStub) screen(t *testing.T, pane, screen, typed string) {
	t.Helper()
	writeTestFile(t, filepath.Join(s.dir, "screen-"+pane), screen)
	writeTestFile(t, filepath.Join(s.dir, "typed-screen-"+pane), typed)
}

// failText makes `herdr pane send-text <pane>` fail.
func (s resumeStub) failText(t *testing.T, pane string) {
	t.Helper()
	writeTestFile(t, filepath.Join(s.dir, "fail-text-"+pane), "")
}

// The screens of the tests, in the shape of the real ones in
// internal/org/testdata/resume/ (docs/evidence/org-seat-resume-probe-2026-11-07.md).
const (
	resumeTestRule         = "─────────────────────────────────────────────────────"
	resumeTestClaudeFooter = "  🤖 Haiku 5.5 | ⚡ medium\n  ⏵⏵ bypass permissions on (shift+tab to cycle)"
	// resumeTestENOTFOUND is a claude turn that ended on a connection
	// failure (evidence section 2), and resumeTestENOTFOUNDLine the line
	// Resume matches in it, wrapped lines joined.
	resumeTestENOTFOUND = "❯ say hi\n\n⏺ API Error: Can't reach the API server —\n  check your internet or DNS (ENOTFOUND)\n\n" +
		"✻ Worked for 1s · done 4:33 AM"
	resumeTestENOTFOUNDLine = "⏺ API Error: Can't reach the API server — check your internet or DNS (ENOTFOUND)"
	// resumeTestRetrying is a claude seat still retrying (evidence section 3).
	resumeTestRetrying     = "❯ say hi\n\n✻ Can't reach the… · Retrying in 4s · attempt 13/300"
	resumeTestRetryingLine = "✻ Can't reach the… · Retrying in 4s · attempt 13/300"
	// resumeTestLimit is a claude turn that ended on a usage limit, and
	// resumeTestPressEnterFooter the footer claude shows once the limit
	// reset while the computer slept.
	resumeTestLimit            = "❯ implement the parser\n\n⏺ You've hit your session limit · resets 3:45pm\n\n✻ Worked for 12s · done 1:02 PM"
	resumeTestPressEnterFooter = "  Your usage limit has reset\n  Press enter to continue\n\n  🤖 Opus 5.5 | ⚡ high\n" +
		"  ⏵⏵ bypass permissions on (shift+tab to cycle)"
	resumeTestPressEnterLine = "Your usage limit has reset Press enter to continue"
	// resumeTestCodexHooks is codex's hook review, a choice dialog (evidence
	// section 4).
	resumeTestCodexHooks = "  Hooks need review\n  7 hooks are new or changed.\n  Hooks can run outside the sandbox after you trust\n" +
		"  them.\n\n\n› 1. Review hooks\n  2. Trust all and continue\n  3. Continue without trusting (hooks won't run)\n\n" +
		"  enter confirm · esc skip\n"
)

// claudeResumeScreen is a claude pane as `herdr pane read --format text`
// prints it: history, then the input box holding input (none when input is
// ""), then footer.
func claudeResumeScreen(history, input, footer string) string {
	marker := "❯"
	if input != "" {
		marker += " " + input
	}
	return history + "\n\n" + resumeTestRule + "\n" + marker + "\n" + resumeTestRule + "\n" + footer + "\n"
}

// seedResumeSeat records seatID of orgID as spawned with role and driver in
// pane, and returns its herdr agent name (<org_id>_<seat_id>, as spawn
// names it).
func seedResumeSeat(t *testing.T, stateDir, orgID, seatID, role, driverName, pane string) string {
	t.Helper()
	appendSeedEvent(t, stateDir, org.ManifestEvent{
		OrgID: orgID, SeatID: seatID, Event: org.EventSpawned, Role: role, Driver: driverName, PaneID: pane,
	})
	return orgID + "_" + seatID
}

// seedStoppedClaudeSeat is seedResumeSeat of a claude seat that is idle on
// the ENOTFOUND screen, whose input box holds the resume sentence once it
// was typed.
func seedStoppedClaudeSeat(t *testing.T, s resumeStub, stateDir, orgID, seatID, role, pane string) {
	t.Helper()
	s.agent(t, seedResumeSeat(t, stateDir, orgID, seatID, role, "claude", pane), "idle")
	s.screen(t, pane, claudeResumeScreen(resumeTestENOTFOUND, "", resumeTestClaudeFooter),
		claudeResumeScreen(resumeTestENOTFOUND, org.ResumeText, resumeTestClaudeFooter))
}

// seedPressEnterDirector records the headless director d1 (claude) in herdr
// pane pane-d, idle on the screen that asks for Enter after a usage limit
// reset.
func seedPressEnterDirector(t *testing.T, s resumeStub, stateDir string) {
	t.Helper()
	reg := seedHeadless(t, stateDir, "claude", "opus", "ws-d", "pane-d", true)
	s.agent(t, org.DirectorAgentName(reg.Token), "idle")
	screen := claudeResumeScreen(resumeTestLimit, "", resumeTestPressEnterFooter)
	s.screen(t, "pane-d", screen, screen)
}

// seedResumeMix seeds the ledger of the main test: a stopped implementer,
// a reviewer still retrying, and the leader of org-a; a codex worker of
// org-b on its hook review; and the headless director d1 asked for Enter.
func seedResumeMix(t *testing.T, s resumeStub, stateDir string) {
	t.Helper()
	seedStoppedClaudeSeat(t, s, stateDir, "org-a", "implementer", "implementer", "pane-impl")
	s.agent(t, seedResumeSeat(t, stateDir, "org-a", "reviewer", "reviewer", "claude", "pane-rev"), "idle")
	retrying := claudeResumeScreen(resumeTestRetrying, "", resumeTestClaudeFooter)
	s.screen(t, "pane-rev", retrying, retrying)
	seedStoppedClaudeSeat(t, s, stateDir, "org-a", "leader", "leader", "pane-lead")
	s.agent(t, seedResumeSeat(t, stateDir, "org-b", "worker", "worker", "codex", "pane-cx"), "idle")
	s.screen(t, "pane-cx", resumeTestCodexHooks, resumeTestCodexHooks)
	seedPressEnterDirector(t, s, stateDir)
}

// resumeSendCalls is the `pane send-text` and `pane send-keys` lines of the
// stub's log, the text of a send-text cut to its pane: what reached a pane.
func resumeSendCalls(t *testing.T, herdrLog string) []string {
	t.Helper()
	var sends []string
	for _, l := range readLogLines(t, herdrLog) {
		switch {
		case strings.HasPrefix(l, "pane send-text "):
			if fields := strings.Fields(l); len(fields) >= 3 {
				sends = append(sends, strings.Join(fields[:3], " "))
			}
		case strings.HasPrefix(l, "pane send-keys "):
			sends = append(sends, l)
		}
	}
	return sends
}

// resumedEvents is the `resumed` events of the manifest under stateDir.
func resumedEvents(t *testing.T, stateDir string) []org.ManifestEvent {
	t.Helper()
	var out []org.ManifestEvent
	for _, ev := range readManifestEvents(t, org.ManifestPathIn(stateDir)) {
		if ev.Event == org.EventResumed {
			out = append(out, ev)
		}
	}
	return out
}

// FR-1, FR-4, and FR-7 through the CLI: one aligned line per target in the
// order Resume handled them (the seats that are not leaders, the leader,
// the director), each with the screen line it matched or the reason, then
// the summary, and exit 0 with a waiting and two skipped targets (D7). Only
// the stopped implementer gets the sentence and Enter, and a `resumed`
// event; the director gets Enter alone and no event.
func TestOrgResume_PrintsALinePerTargetAndTheSummary(t *testing.T) {
	herdrLog, s := stubResumeHerdr(t)
	stateDir := t.TempDir()
	seedResumeMix(t, s, stateDir)
	t.Setenv("HERDR_PANE_ID", "pane-lead")

	stdout, stderr, err := runOrgCmdStreams(t, "resume", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("resume: %v (stdout %s stderr %s)", err, stdout, stderr)
	}
	want := "resumed  org-a/implementer  claude  " + resumeTestENOTFOUNDLine + "\n" +
		"waiting  org-a/reviewer     claude  " + resumeTestRetryingLine + "\n" +
		"skipped  org-b/worker       codex   unknown screen: choice dialog\n" +
		"skipped  org-a/leader       claude  the caller's own pane\n" +
		"entered  director d1        claude  " + resumeTestPressEnterLine + "\n" +
		"resume: 1 resumed, 1 entered, 1 waiting, 2 skipped, 0 failed\n"
	if stdout != want {
		t.Errorf("stdout =\n%s\nwant\n%s", stdout, want)
	}
	if stderr != "" {
		t.Errorf("stderr = %q; want nothing (both submits confirmed)", stderr)
	}
	if got, want := resumeSendCalls(t, herdrLog), []string{
		"pane send-text pane-impl", "pane send-keys pane-impl Enter", "pane send-keys pane-d Enter",
	}; !slices.Equal(got, want) {
		t.Errorf("sends = %q; want %q", got, want)
	}
	events := resumedEvents(t, stateDir)
	if len(events) != 1 || events[0].OrgID != "org-a" || events[0].SeatID != "implementer" || events[0].PaneID != "pane-impl" {
		t.Errorf("resumed events = %+v; want one for org-a/implementer in pane-impl", events)
	}
}

// AC-6 through the CLI: --dry-run looks at every target the same way, says
// would_resume and would_enter where a real run types, sends nothing, and
// records nothing.
func TestOrgResume_DryRunTypesNothingAndSummarizes(t *testing.T) {
	herdrLog, s := stubResumeHerdr(t)
	stateDir := t.TempDir()
	seedResumeMix(t, s, stateDir)
	t.Setenv("HERDR_PANE_ID", "pane-lead")

	stdout, stderr, err := runOrgCmdStreams(t, "resume", "--dry-run", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("resume --dry-run: %v (stdout %s stderr %s)", err, stdout, stderr)
	}
	want := "would_resume  org-a/implementer  claude  " + resumeTestENOTFOUNDLine + "\n" +
		"waiting       org-a/reviewer     claude  " + resumeTestRetryingLine + "\n" +
		"skipped       org-b/worker       codex   unknown screen: choice dialog\n" +
		"skipped       org-a/leader       claude  the caller's own pane\n" +
		"would_enter   director d1        claude  " + resumeTestPressEnterLine + "\n" +
		"resume (dry run): 1 would resume, 1 would enter, 1 waiting, 2 skipped\n"
	if stdout != want {
		t.Errorf("stdout =\n%s\nwant\n%s", stdout, want)
	}
	if stderr != "" {
		t.Errorf("stderr = %q; want nothing", stderr)
	}
	if sends := resumeSendCalls(t, herdrLog); len(sends) != 0 {
		t.Errorf("sends = %q; want none", sends)
	}
	if events := resumedEvents(t, stateDir); len(events) != 0 {
		t.Errorf("resumed events = %+v; want none", events)
	}
}

// FR-7 through the CLI: a ledger with no running seat and no director prints
// one line and exits 0, with or without --dry-run, and asks herdr nothing.
func TestOrgResume_NoRunningSeats(t *testing.T) {
	herdrLog, _ := stubResumeHerdr(t)
	stateDir := t.TempDir()
	appendSeedEvent(t, stateDir, org.ManifestEvent{OrgID: "org-a", SeatID: "gone", Event: org.EventStopped, PaneID: "pane-gone"})

	for _, tc := range []struct {
		args []string
		want string
	}{
		{nil, "resume: no running seats\n"},
		{[]string{"--dry-run"}, "resume (dry run): no running seats\n"},
	} {
		stdout, stderr, err := runOrgCmdStreams(t, append([]string{"resume", "--state-dir", stateDir}, tc.args...)...)
		if err != nil || stdout != tc.want || stderr != "" {
			t.Errorf("resume %v = stdout %q, stderr %q, err %v; want stdout %q, no stderr, exit 0", tc.args, stdout, stderr, err, tc.want)
		}
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %q; want none", lines)
	}
}

// D7 through the CLI: a seat whose send-text fails is failed, with its
// reason on its line and its error on stderr, the next seat is still
// resumed, and the command exits 1.
func TestOrgResume_ExitsOneWhenASendFails(t *testing.T) {
	herdrLog, s := stubResumeHerdr(t)
	stateDir := t.TempDir()
	seedStoppedClaudeSeat(t, s, stateDir, "org-a", "a", "worker", "pane-a")
	seedStoppedClaudeSeat(t, s, stateDir, "org-a", "b", "worker", "pane-b")
	s.failText(t, "pane-a")

	stdout, stderr, err := runOrgCmdStreams(t, "resume", "--state-dir", stateDir)
	if err == nil || ExitCode(err) != 1 {
		t.Fatalf("resume with a failing send: err %v (exit %d); want exit 1 (stdout %s stderr %s)", err, ExitCode(err), stdout, stderr)
	}
	if want := "org: resume: 1 error(s), listed above"; err.Error() != want {
		t.Errorf("err = %q; want %q", err, want)
	}
	lines := outputLines(stdout)
	if len(lines) != 3 {
		t.Fatalf("stdout =\n%s\nwant the two seats' lines and the summary", stdout)
	}
	if !strings.HasPrefix(lines[0], "failed   org-a/a  claude  herdr pane send-text pane-a failed: ") ||
		!strings.HasSuffix(lines[0], "the text may or may not have reached the pane; Enter not pressed") {
		t.Errorf("line of the failed seat = %q; want failed with the send-text reason", lines[0])
	}
	if want := "resumed  org-a/b  claude  " + resumeTestENOTFOUNDLine; lines[1] != want {
		t.Errorf("line of the resumed seat = %q; want %q", lines[1], want)
	}
	if want := "resume: 1 resumed, 0 entered, 0 waiting, 0 skipped, 1 failed"; lines[2] != want {
		t.Errorf("summary = %q; want %q", lines[2], want)
	}
	if !strings.HasPrefix(stderr, "error: org-a/a: herdr pane send-text pane-a failed: ") || strings.Count(stderr, "\n") != 1 {
		t.Errorf("stderr = %q; want one error: line for org-a/a", stderr)
	}
	if got, want := resumeSendCalls(t, herdrLog), []string{
		"pane send-text pane-a", "pane send-text pane-b", "pane send-keys pane-b Enter",
	}; !slices.Equal(got, want) {
		t.Errorf("sends = %q; want %q (no Enter for pane-a)", got, want)
	}
	if events := resumedEvents(t, stateDir); len(events) != 1 || events[0].SeatID != "b" {
		t.Errorf("resumed events = %+v; want one for b", events)
	}
}

// D6 through the CLI: --org-id limits resume to that org's seats, with no
// director, and herdr is asked about nothing else.
func TestOrgResume_OrgIDLimitsItToThatOrg(t *testing.T) {
	herdrLog, s := stubResumeHerdr(t)
	stateDir := t.TempDir()
	seedStoppedClaudeSeat(t, s, stateDir, "org-a", "implementer", "implementer", "pane-a")
	seedStoppedClaudeSeat(t, s, stateDir, "org-b", "implementer", "implementer", "pane-b")
	seedPressEnterDirector(t, s, stateDir)

	stdout, stderr, err := runOrgCmdStreams(t, "resume", "--org-id", "org-a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("resume --org-id org-a: %v (stdout %s stderr %s)", err, stdout, stderr)
	}
	want := "resumed  org-a/implementer  claude  " + resumeTestENOTFOUNDLine + "\n" +
		"resume: 1 resumed, 0 entered, 0 waiting, 0 skipped, 0 failed\n"
	if stdout != want || stderr != "" {
		t.Errorf("stdout =\n%s\nstderr %q\nwant\n%s", stdout, stderr, want)
	}
	for _, l := range readLogLines(t, herdrLog) {
		if strings.Contains(l, "org-b") || strings.Contains(l, "pane-b") || strings.Contains(l, "director_") || strings.Contains(l, "pane-d") {
			t.Errorf("herdr call %q is about another org or the director", l)
		}
	}
}

// D6 through the CLI: an --org-id that is given is checked before anything
// is read: an empty one is refused rather than read as every org, and one
// that is not an identifier as the other verbs refuse it.
func TestOrgResume_RefusesAnInvalidOrgID(t *testing.T) {
	herdrLog, s := stubResumeHerdr(t)
	stateDir := t.TempDir()
	seedStoppedClaudeSeat(t, s, stateDir, "org-a", "implementer", "implementer", "pane-a")

	for _, tc := range []struct {
		orgID, want string
	}{
		{"", "org: resume: --org-id is empty; leave it out to resume the seats of every org"},
		{"  ", "org: resume: --org-id is empty; leave it out to resume the seats of every org"},
		{"Org A", `org: invalid org_id "Org A": must match `},
		{"../org-a", `org: invalid org_id "../org-a": must match `},
	} {
		stdout, _, err := runOrgCmdStreams(t, "resume", "--org-id", tc.orgID, "--state-dir", stateDir)
		if err == nil || !strings.HasPrefix(err.Error(), tc.want) {
			t.Errorf("resume --org-id %q: err %v; want it to start with %q", tc.orgID, err, tc.want)
		}
		if stdout != "" {
			t.Errorf("resume --org-id %q: stdout %q; want nothing", tc.orgID, stdout)
		}
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %q; want none", lines)
	}
	if events := resumedEvents(t, stateDir); len(events) != 0 {
		t.Errorf("resumed events = %+v; want none", events)
	}
}

// FR-7 through the CLI: a resumed seat and an entered director whose submit
// herdr did not confirm each get a note naming the command that reads the
// pane. For the seat it is ralph org read, with --state-dir only when it
// was given; for the director, herdr pane read. The exit code stays 0.
func TestOrgResume_UnconfirmedSubmitNamesTheReadCommand(t *testing.T) {
	directorNote := "note: director d1: the submit was not confirmed; read the pane with " +
		"herdr pane read pane-d --source recent --lines 60 --format text\n"
	for _, tc := range []struct {
		name     string
		stateDir bool
	}{
		{"with --state-dir", true},
		{"without --state-dir", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, s := stubResumeHerdr(t)
			t.Setenv("ORG_STUB_AGENT_WAIT_CONFIRM_FAIL", "1")
			args := []string{"resume"}
			var stateDir, hint string
			if tc.stateDir {
				stateDir = filepath.Join(t.TempDir(), "state dir")
				args = append(args, "--state-dir", stateDir)
				hint = " --state-dir " + org.ShellWord(stateDir)
			} else {
				t.Chdir(t.TempDir())
				cwd, err := os.Getwd()
				if err != nil {
					t.Fatal(err)
				}
				stateDir = filepath.Join(cwd, ".harness", "state", "org")
			}
			seedStoppedClaudeSeat(t, s, stateDir, "org-a", "implementer", "implementer", "pane-a")
			seedPressEnterDirector(t, s, stateDir)

			stdout, stderr, err := runOrgCmdStreams(t, args...)
			if err != nil {
				t.Fatalf("resume: %v (stdout %s stderr %s)", err, stdout, stderr)
			}
			if want := "resume: 1 resumed, 1 entered, 0 waiting, 0 skipped, 0 failed\n"; !strings.HasSuffix(stdout, want) {
				t.Errorf("stdout =\n%s\nwant it to end with %q", stdout, want)
			}
			want := "note: org-a/implementer: the submit was not confirmed; read the pane with " +
				"ralph org read --org-id org-a --seat implementer" + hint + "\n" + directorNote
			if stderr != want {
				t.Errorf("stderr =\n%s\nwant\n%s", stderr, want)
			}
			if events := resumedEvents(t, stateDir); len(events) != 1 || !strings.Contains(events[0].Details, "submit_unconfirmed=true") {
				t.Errorf("resumed events = %+v; want one with submit_unconfirmed=true", events)
			}
		})
	}
}

// A seat whose `resumed` event cannot be recorded after Enter is still
// resumed on its line, gets a warning, and its error makes the command exit
// 1 (the error is in ResumeResult.Errs).
func TestOrgResume_WarnsWhenTheResumedEventIsNotRecorded(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes a read-only manifest")
	}
	herdrLog, s := stubResumeHerdr(t)
	stateDir := t.TempDir()
	seedStoppedClaudeSeat(t, s, stateDir, "org-a", "implementer", "implementer", "pane-a")
	manifest := org.ManifestPathIn(stateDir)
	if err := os.Chmod(manifest, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(manifest, 0o644) })

	stdout, stderr, err := runOrgCmdStreams(t, "resume", "--state-dir", stateDir)
	if err == nil || ExitCode(err) != 1 {
		t.Fatalf("resume with a read-only manifest: err %v; want exit 1 (stdout %s stderr %s)", err, stdout, stderr)
	}
	want := "resumed  org-a/implementer  claude  " + resumeTestENOTFOUNDLine + "\n" +
		"resume: 1 resumed, 0 entered, 0 waiting, 0 skipped, 0 failed\n"
	if stdout != want {
		t.Errorf("stdout =\n%s\nwant\n%s", stdout, want)
	}
	lines := outputLines(stderr)
	if len(lines) != 2 ||
		lines[0] != "warning: org-a/implementer: Enter was pressed, but the resumed event could not be recorded in the manifest" ||
		!strings.HasPrefix(lines[1], "error: org-a/implementer: the resumed event could not be recorded after Enter was pressed: ") {
		t.Errorf("stderr =\n%s\nwant the warning and the error of org-a/implementer", stderr)
	}
	if got, want := resumeSendCalls(t, herdrLog), []string{"pane send-text pane-a", "pane send-keys pane-a Enter"}; !slices.Equal(got, want) {
		t.Errorf("sends = %q; want %q", got, want)
	}
}

// FR-1 through the CLI: when director.jsonl cannot be read, the seats are
// resumed all the same, a warning says the directors were not looked at,
// and the exit code stays 0.
func TestOrgResume_WarnsWhenTheDirectorsCannotBeRead(t *testing.T) {
	_, s := stubResumeHerdr(t)
	stateDir := t.TempDir()
	seedStoppedClaudeSeat(t, s, stateDir, "org-a", "implementer", "implementer", "pane-a")
	if err := os.Mkdir(filepath.Join(stateDir, "director.jsonl"), 0o755); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runOrgCmdStreams(t, "resume", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("resume with an unreadable director.jsonl: %v (stdout %s stderr %s)", err, stdout, stderr)
	}
	if want := "resumed  org-a/implementer  claude  " + resumeTestENOTFOUNDLine + "\n"; !strings.HasPrefix(stdout, want) {
		t.Errorf("stdout =\n%s\nwant it to start with %q", stdout, want)
	}
	if want := "warning: resume: the headless directors of the ledger could not be read, so none was looked at: "; !strings.HasPrefix(stderr, want) ||
		strings.Count(stderr, "\n") != 1 {
		t.Errorf("stderr = %q; want one line starting with %q", stderr, want)
	}
}

// A manifest that cannot be read is an error (FR-7: exit 1) and prints no
// line on stdout, not "no running seats".
func TestOrgResume_ExitsOneWhenTheManifestCannotBeRead(t *testing.T) {
	herdrLog, _ := stubResumeHerdr(t)
	stateDir := t.TempDir()
	if err := os.Mkdir(org.ManifestPathIn(stateDir), 0o755); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runOrgCmdStreams(t, "resume", "--state-dir", stateDir)
	if err == nil || ExitCode(err) != 1 {
		t.Fatalf("resume with an unreadable manifest: err %v; want exit 1 (stdout %s stderr %s)", err, stdout, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing", stdout)
	}
	if !strings.HasPrefix(stderr, "error: org: resume: read manifest: ") || strings.Count(stderr, "\n") != 1 {
		t.Errorf("stderr = %q; want one error: line about the manifest", stderr)
	}
	if lines := readLogLines(t, herdrLog); len(lines) != 0 {
		t.Errorf("herdr calls = %q; want none", lines)
	}
}

// A pane's text reaches the terminal escaped: a control character in the
// matched line is printed as its Go escape, not as itself.
func TestOrgResume_EscapesControlCharactersOfTheScreen(t *testing.T) {
	_, s := stubResumeHerdr(t)
	stateDir := t.TempDir()
	s.agent(t, seedResumeSeat(t, stateDir, "org-a", "implementer", "implementer", "claude", "pane-a"), "idle")
	history := "❯ say hi\n\n⏺ API Error: Can't reach the API server \x1b[2J(ENOTFOUND)\n\n✻ Worked for 1s · done 4:33 AM"
	s.screen(t, "pane-a", claudeResumeScreen(history, "", resumeTestClaudeFooter), claudeResumeScreen(history, "", resumeTestClaudeFooter))

	stdout, _, err := runOrgCmdStreams(t, "resume", "--dry-run", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("resume --dry-run: %v (stdout %s)", err, stdout)
	}
	if strings.Contains(stdout, "\x1b") {
		t.Errorf("stdout %q holds an escape character", stdout)
	}
	if want := `would_resume  org-a/implementer  claude  ⏺ API Error: Can't reach the API server \x1b[2J(ENOTFOUND)`; !strings.HasPrefix(stdout, want+"\n") {
		t.Errorf("stdout =\n%s\nwant it to start with\n%s", stdout, want)
	}
}

// The help of resume says what it types, when, what it never types into,
// the order, the caller's own pane, --dry-run, the exit codes, and who runs
// it; the help of ralph org names resume and says --org-id is optional for
// it, next to the verbs that refuse it.
func TestOrgResume_HelpNamesTheSentenceAndDryRun(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		wants []string
	}{
		{[]string{"resume", "--help"}, []string{
			org.ResumeText,
			"Only when herdr reports the agent idle or done and the screen ends with a known API error sentence above an empty input box does it type this sentence and press Enter once",
			"(Press enter to continue), it presses Enter once and types nothing",
			"It never types into a seat that is working, retrying or waiting on its own, or shows an approval, a choice, or any screen it does not know",
			"It goes through the seats that are not leaders first, then the leaders, then the headless directors, and skips the pane it runs in (HERDR_PANE_ID)",
			"--dry-run looks at every seat the same way but types and records nothing (would_resume, would_enter)",
			"It exits 1 when typing into a seat failed or the manifest could not be read, and 0 otherwise, also when every seat was waiting or skipped",
			"The director runs it when the human asks to continue after a connection failure or a usage limit; a human can run it too.",
			"--dry-run look at every seat as a real run does, but type and record nothing",
		}},
		{[]string{"--help"}, []string{
			"resume Continue the seats whose CLI stopped on an API error (a connection failure or a usage limit)",
			"`ralph org resume` types a fixed sentence into the seats whose CLI stopped on an API error",
			"stop --all, disband --all, merge-order, merge, cleanup, regate, wait --inbox, watch, approve, the director verbs, the inbox verbs, and the followup verbs, which refuse it",
			"optional for resume, which without it looks at the seats of every org and at the headless directors",
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
}
