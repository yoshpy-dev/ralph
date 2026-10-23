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
