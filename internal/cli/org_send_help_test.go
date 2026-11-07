package cli

import (
	"strings"
	"testing"
)

// TestOrgSend_HelpSaysItWaitsForTheTurnToEnd pins the `ralph org send --help`
// paragraph of plan 2026-10-23-busy-leader (AC3 and D12): send waits up to
// --timeout-ms for the seat to be idle or done, a seat busy in the
// foreground does not get the message until its turn ends, nothing is typed
// when the wait runs out, and the submit check reads a turn the seat started
// on its own as confirmed, so `ralph org read` is what shows whether the
// message arrived.
func TestOrgSend_HelpSaysItWaitsForTheTurnToEnd(t *testing.T) {
	out, err := runOrgCmd(t, "send", "--help")
	if err != nil {
		t.Fatal(err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"Before typing, send waits up to --timeout-ms for the seat to be idle or done.",
		"A seat in the middle of a turn (waiting in the foreground on a loop, `ralph org wait`, or a gate) does not get the message until the turn ends.",
		"When the wait runs out, nothing is typed and the error says the seat's turn did not end and how to send again.",
		"The submit check only sees the seat's turn go working or blocked, so a turn the seat started on its own " +
			"(a Claude Code leader woken by its Monitor, say) also reads as confirmed; `ralph org read` shows whether the message arrived.",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("expected `ralph org send --help` to contain %q, got:\n%s", want, out)
		}
	}
}

// TestOrgSend_HelpSaysItTakesThePaneLock pins the `ralph org send --help`
// paragraph of plan 2026-11-07-org-seat-resume (D8 and FR-5a): send takes the
// lock of the seat's pane that `ralph org resume` takes, waits for it no
// longer than what is left of --timeout-ms, and when another send or resume
// still holds it then, types and records nothing and exits 1.
func TestOrgSend_HelpSaysItTakesThePaneLock(t *testing.T) {
	out, err := runOrgCmd(t, "send", "--help")
	if err != nil {
		t.Fatal(err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"Send also takes the lock of the seat's pane, the one `ralph org resume` takes, and holds it until Enter is pressed and the submit is checked.",
		"It waits for the lock no longer than what is left of --timeout-ms.",
		"When another send or resume still holds it then, nothing is typed, nothing is recorded, and send exits 1.",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("expected `ralph org send --help` to contain %q, got:\n%s", want, out)
		}
	}
}
