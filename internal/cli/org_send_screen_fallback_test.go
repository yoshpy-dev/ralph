package cli

import (
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// TestOrgSend_HelpDescribesTheScreenFallback pins the `ralph org send --help`
// paragraph and flag of plan 2026-11-12-send-screen-fallback (D1, D5): the
// wait on herdr leaves 10 seconds (at most a third of --timeout-ms) for a
// look at the screen, send types only on a screen that shows the seat
// waiting for input on two reads, the sent event records via=screen, any
// other screen gets nothing typed and the rest of --timeout-ms on herdr, and
// --no-screen-fallback turns the look off.
func TestOrgSend_HelpDescribesTheScreenFallback(t *testing.T) {
	out, err := runOrgCmd(t, "send", "--help")
	if err != nil {
		t.Fatal(err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"herdr can keep reporting a seat working after its turn ended.",
		"the wait on herdr leaves 10 seconds of --timeout-ms (at most a third of it) for a look at the seat's screen.",
		"send reads the pane twice, 3 seconds apart.",
		"Only when both reads show the same screen waiting for input",
		"checks that the text is in the input box before it presses Enter, and confirms the submit on the screen instead of with herdr.",
		"The sent event then starts with `via=screen herdr_state=<state>`.",
		"send waits on herdr for the rest of --timeout-ms",
		"A --timeout-ms whose third is less than 6 seconds plus the pause before Enter gets no look at the screen.",
		"--no-screen-fallback turns the look off: send waits on herdr alone.",
		"--no-screen-fallback wait on herdr's idle/done state alone for all of --timeout-ms",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("expected `ralph org send --help` to contain %q, got:\n%s", want, out)
		}
	}
}

// TestOrgSend_NoScreenFallbackFlag_ReachesSend checks that the CLI passes
// --no-screen-fallback on to Send: for a claude seat and the default
// --timeout-ms of 30000, herdr's first idle/done wait gets 20000 ms (the
// rest is the screen check's reserve), and with the flag it gets all 30000.
// The herdr stub reports idle at once, so neither send reads the screen.
func TestOrgSend_NoScreenFallbackFlag_ReachesSend(t *testing.T) {
	for _, tc := range []struct {
		name  string
		extra []string
		want  string
	}{
		{"the fallback on", nil, "agent wait org-a_seat-1 --until idle --until done --timeout 20000"},
		{"--no-screen-fallback", []string{"--no-screen-fallback"}, "agent wait org-a_seat-1 --until idle --until done --timeout 30000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			herdrLog, _ := setupOrgStubPATH(t)
			stateDir := filepath.Join(t.TempDir(), "state")
			if _, err := runOrgCmd(t,
				"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
				"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
				"--scope", "test-scope", "--state-dir", stateDir,
			); err != nil {
				t.Fatalf("spawn failed: %v", err)
			}
			before := len(readLogLines(t, herdrLog))

			args := append([]string{"send", "--org-id", "org-a", "--to", "seat-1", "--text", "TYPE: QUESTION\n\nwhich plan?",
				"--enter-delay-ms", "1", "--state-dir", stateDir}, tc.extra...)
			if out, err := runOrgCmd(t, args...); err != nil {
				t.Fatalf("send failed: %v (output: %s)", err, out)
			}
			var waits []string
			for _, line := range readLogLines(t, herdrLog)[before:] {
				if strings.HasPrefix(line, "agent wait ") {
					waits = append(waits, line)
				}
				if strings.HasPrefix(line, "pane read ") || strings.HasPrefix(line, "agent get ") {
					t.Errorf("herdr call %q; want no screen check when herdr reports idle", line)
				}
			}
			if len(waits) == 0 || waits[0] != tc.want {
				t.Fatalf("send's agent waits = %q; want the first to be %q", waits, tc.want)
			}
		})
	}
}

// The tests below drive `ralph org send` through the screen check from the
// CLI (the plan's Test plan, Integration tests). herdr is the resume herdr
// of stubResumeHerdr (org_resume_test.go): `agent get` reports the seat
// working, and `agent wait` reports no state before its caller's deadline
// (keepsWorking), so herdr's waits run out and only the screen can let the
// send type. The screens are the real ones of
// internal/org/testdata/resume/. No flag of the CLI shortens the 3 s
// between the check's two reads (org.Org's SendScreenReadGap, which the CLI
// leaves at its default), so a --timeout-ms that still gets the check is at
// least 18 s, and each send runs for most of its --timeout-ms, 18 to 20 s.

const (
	// sendScreenPathMsg is the short message of evidence 2026-11-12 section
	// 10, whose input box claude-input-typed-multiline-short.txt shows.
	sendScreenPathMsg = "TYPE: QUESTION\n\nReply with the single word: alpha. This is a probe, so do not run any tool and do not read or edit any file."
	// sendScreenPathTimeoutMS is the --timeout-ms of the send that types by
	// the screen. A third of it, 8 s, is the reserve, and the screen check
	// gets half of that: the 3 s between its two reads and 1 s for its herdr
	// calls (agent get and two pane reads). herdr's first wait gets the
	// other 16 s.
	sendScreenPathTimeoutMS = 24000
	// sendScreenPathShortTimeoutMS is the --timeout-ms of the sends whose
	// waits run out: a little over the least whose third still makes a
	// reserve (two 3 s read gaps and the 1 ms pause of --enter-delay-ms 1),
	// so they end after about 18 s. herdr's first wait gets two thirds of
	// it, 12200 ms.
	sendScreenPathShortTimeoutMS = 18300
	// sendScreenPathRead is the herdr call that reads the seat's pane.
	sendScreenPathRead = "pane read pane-s --source recent --lines 60 --format text"
)

// seedSendScreenPathSeat seeds seat-1 of org-a, a claude worker in pane-s
// that herdr keeps reporting working, whose pane shows screen until a send
// types into it and then the input box holding sendScreenPathMsg.
func seedSendScreenPathSeat(t *testing.T, s resumeStub, stateDir, screen string) {
	t.Helper()
	name := seedResumeSeat(t, stateDir, "org-a", "seat-1", "worker", "claude", "pane-s")
	s.agent(t, name, "working")
	s.keepsWorking(t, name)
	s.screen(t, "pane-s", screen, liveStatusScreen(t, "claude-input-typed-multiline-short.txt"))
}

// runSendScreenPath runs `ralph org send` of sendScreenPathMsg to seat-1
// with --timeout-ms timeoutMS, --enter-delay-ms 1, and extra, and returns
// its streams, how long it ran, and its error.
func runSendScreenPath(t *testing.T, stateDir string, timeoutMS int, extra ...string) (stdout, stderr string, elapsed time.Duration, err error) {
	t.Helper()
	args := append([]string{"send", "--org-id", "org-a", "--to", "seat-1", "--text", sendScreenPathMsg,
		"--timeout-ms", strconv.Itoa(timeoutMS), "--enter-delay-ms", "1", "--state-dir", stateDir}, extra...)
	start := time.Now()
	stdout, stderr, err = runOrgCmdStreams(t, args...)
	return stdout, stderr, time.Since(start), err
}

// sendScreenPathCalls is the herdr calls of herdrLog, one entry per call: a
// `pane send-text` is cut to its pane, and the lines of the log that the
// rest of sendScreenPathMsg's lines take are dropped.
func sendScreenPathCalls(t *testing.T, herdrLog string) []string {
	t.Helper()
	lines := readLogLines(t, herdrLog)
	var calls []string
	for i := 0; i < len(lines); i++ {
		call := lines[i]
		if strings.HasPrefix(call, "pane send-text ") {
			call = strings.Join(strings.Fields(call)[:3], " ")
			i += strings.Count(sendScreenPathMsg, "\n")
		}
		calls = append(calls, call)
	}
	return calls
}

// herdr keeps reporting the seat working, so its first wait runs out after
// --timeout-ms less the reserve; the screen waits for input on both reads of
// the check, so send takes the pane's lock, reads the screen again, types,
// finds the text in the input box, presses Enter once, and confirms the
// submit on the screen, which shows the empty input box under a running
// turn. It exits 0 within --timeout-ms, warns of nothing, and the sent event
// records via=screen and herdr's state.
func TestOrgSendScreenPath_ScreenWaitsForInput_TypesAndRecordsViaScreen(t *testing.T) {
	herdrLog, s := stubResumeHerdr(t)
	stateDir := t.TempDir()
	seedSendScreenPathSeat(t, s, stateDir, liveStatusScreen(t, "claude-idle-after-reply.txt"))
	s.entered(t, "pane-s", liveStatusScreen(t, "claude-submitted-just-now.txt"))

	stdout, stderr, elapsed, err := runSendScreenPath(t, stateDir, sendScreenPathTimeoutMS)
	if code := ExitCode(err); code != 0 || stdout != "sent message to seat \"seat-1\"\n" || stderr != "" {
		t.Fatalf("send: exit %d, err %v, stdout %q, stderr %q; want exit 0, the sent line, and no warning", code, err, stdout, stderr)
	}
	want := []string{
		"agent wait org-a_seat-1 --until idle --until done --timeout 16000", // herdr's first wait, which runs out
		"agent get org-a_seat-1", sendScreenPathRead, sendScreenPathRead, // the screen check
		sendScreenPathRead,                          // again, holding the pane lock
		"pane send-text pane-s", sendScreenPathRead, // the typing and the check before Enter
		"pane send-keys pane-s Enter", sendScreenPathRead, // Enter, and the check after it on the screen
	}
	if got := sendScreenPathCalls(t, herdrLog); !slices.Equal(got, want) {
		t.Fatalf("herdr calls =\n%q\nwant\n%q", got, want)
	}
	ev := lastSeatEvent(t, stateDir, "org-a", "seat-1")
	if want := "via=screen herdr_state=working " + sendScreenPathMsg; ev.Event != org.EventSent || ev.PaneID != "pane-s" || ev.Details != want {
		t.Fatalf("last event = %+v; want a sent event for pane-s with Details %q", ev, want)
	}
	if limit := time.Duration(sendScreenPathTimeoutMS) * time.Millisecond; elapsed > limit {
		t.Fatalf("send took %v; want it within --timeout-ms (%v)", elapsed, limit)
	}
}

// herdr keeps reporting the seat working and the screen shows a turn at
// work, so send types nothing: it waits on herdr for the rest of
// --timeout-ms and exits 1 with nothing recorded. The error, which ralph
// prints on stderr, keeps the deadline's hint and adds herdr's state, what
// the screen showed, and the commands to look into it.
func TestOrgSendScreenPath_ScreenBusy_TypesNothingAndSaysWhy(t *testing.T) {
	herdrLog, s := stubResumeHerdr(t)
	stateDir := t.TempDir()
	seedSendScreenPathSeat(t, s, stateDir, liveStatusScreen(t, "claude-working-tool.txt"))

	stdout, stderr, _, err := runSendScreenPath(t, stateDir, sendScreenPathShortTimeoutMS)
	if code := ExitCode(err); code != 1 || stdout != "" || stderr != "" {
		t.Fatalf("send: exit %d, err %v, stdout %q, stderr %q; want exit 1 and no note", code, err, stdout, stderr)
	}
	msg := err.Error()
	stateDirWord := org.ShellWord(stateDir)
	for _, want := range []string{
		"org: send: wait for seat \"seat-1\" idle/done: ",
		"(the seat's turn did not end within the --timeout-ms deadline of 18300 ms; nothing was typed;",
		"; screen check: herdr state working, the screen was busy (a busy line is on the screen: \"✶ Deliberating… (6s · ↓ 100 tokens)\"); ",
		"read the pane with `ralph org read --org-id org-a --seat seat-1 --state-dir " + stateDirWord + "`",
		"compare herdr's state with the screen with `ralph org status --org-id org-a --live --state-dir " + stateDirWord + "`",
		"see the rule behind herdr's state with `herdr agent explain org-a_seat-1`",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("error lacks %q:\n%s", want, msg)
		}
	}

	calls := sendScreenPathCalls(t, herdrLog)
	want := []string{
		"agent wait org-a_seat-1 --until idle --until done --timeout 12200", // herdr's first wait, which runs out
		"agent get org-a_seat-1", sendScreenPathRead, // the screen check, which stops at the busy screen
	}
	const againPrefix = "agent wait org-a_seat-1 --until idle --until done --timeout "
	if len(calls) != len(want)+1 || !slices.Equal(calls[:len(want)], want) || !strings.HasPrefix(calls[len(want)], againPrefix) {
		t.Fatalf("herdr calls =\n%q\nwant\n%q\nand then herdr's wait for the rest of --timeout-ms", calls, want)
	}
	if ms, err := strconv.Atoi(strings.TrimPrefix(calls[len(want)], againPrefix)); err != nil || ms < 1 || ms > sendScreenPathShortTimeoutMS/3 {
		t.Errorf("herdr's second wait %q; want a --timeout of at most the reserve, %d ms", calls[len(want)], sendScreenPathShortTimeoutMS/3)
	}
	if ev := lastSeatEvent(t, stateDir, "org-a", "seat-1"); ev.Event != org.EventSpawned {
		t.Errorf("last event = %+v; want the seeded spawn, nothing recorded", ev)
	}
}

// --no-screen-fallback turns the screen check off: on a seat herdr keeps
// reporting working whose screen waits for input, herdr's wait gets all of
// --timeout-ms (the --timeout-ms that, without the flag, reads the screen in
// the test above), and when it runs out send exits 1 with the deadline's
// hint alone. herdr is asked for neither the seat's state nor its screen,
// and nothing is typed or recorded.
func TestOrgSendScreenPath_NoScreenFallback_ReadsNoScreen(t *testing.T) {
	herdrLog, s := stubResumeHerdr(t)
	stateDir := t.TempDir()
	seedSendScreenPathSeat(t, s, stateDir, liveStatusScreen(t, "claude-idle-after-reply.txt"))

	stdout, stderr, _, err := runSendScreenPath(t, stateDir, sendScreenPathShortTimeoutMS, "--no-screen-fallback")
	if code := ExitCode(err); code != 1 || stdout != "" || stderr != "" {
		t.Fatalf("send: exit %d, err %v, stdout %q, stderr %q; want exit 1 and no note", code, err, stdout, stderr)
	}
	if msg := err.Error(); !strings.Contains(msg, "(the seat's turn did not end within the --timeout-ms deadline of 18300 ms;") ||
		strings.Contains(msg, "screen check:") {
		t.Errorf("error = %s\nwant the deadline's hint and no screen check", msg)
	}
	want := []string{"agent wait org-a_seat-1 --until idle --until done --timeout 18300"}
	if got := sendScreenPathCalls(t, herdrLog); !slices.Equal(got, want) {
		t.Fatalf("herdr calls = %q; want only herdr's wait, for all of --timeout-ms: %q", got, want)
	}
	if ev := lastSeatEvent(t, stateDir, "org-a", "seat-1"); ev.Event != org.EventSpawned {
		t.Errorf("last event = %+v; want the seeded spawn, nothing recorded", ev)
	}
}
