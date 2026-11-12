package cli

import (
	"path/filepath"
	"strings"
	"testing"
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
