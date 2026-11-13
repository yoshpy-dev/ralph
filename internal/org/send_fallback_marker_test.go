package org

import (
	"slices"
	"strings"
	"testing"
)

// The test of Send's screen path on a message with a line that starts with
// the input box's marker (plan 2026-11-13-send-input-box-marker, issue #322).

// TestOrgSendScreen_MessageWithAMarkerLine_EnterPressedAndConfirmed covers
// AC4b of plan 2026-11-13-send-input-box-marker (issue #322): on the screen
// path, a codex seat gets a message with a line that starts with codex's
// marker (`› explain this`). The pane shows the real screens of
// docs/evidence/org-send-marker-probe-2026-11-13.md: before Enter, the
// message in the box with that line indented under the box's head; 0.3 s
// after Enter, the message in the history over an empty box. Send presses
// Enter once, confirms the submit on the screen, and records via=screen with
// no submit_unconfirmed.
func TestOrgSendScreen_MessageWithAMarkerLine_EnterPressedAndConfirmed(t *testing.T) {
	o, h, name := sendFallbackOrg(t, "codex")
	idle := readResumeScreen(t, "codex-idle-after-reply.txt")
	h.setScreens("pane-1", idle, idle, idle,
		readResumeScreen(t, "codex-input-typed-marker-line.txt"), readResumeScreen(t, "codex-submitted-marker-line.txt"))
	before := len(mustReadEvents(t, o))

	res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: sendMarkerCodexShortMsg, TimeoutMS: sendFallbackTimeoutMS})
	if res.Err != nil || res.Progress != SendProgressEnterPressed || !res.SubmitConfirmed || res.PaneID != "pane-1" {
		t.Fatalf("Send = %+v; want the message sent and the submit confirmed", res)
	}
	want := []string{
		"wait " + name + " idle,done",                     // herdr's first wait, which runs out
		"get " + name, "read pane-1 60", "read pane-1 60", // the screen check
		"read pane-1 60",                // again, holding the pane lock
		"text pane-1", "read pane-1 60", // the typing and the check before Enter
		"keys pane-1 Enter", "read pane-1 60", // Enter once, and the check after it on the screen
	}
	if got := h.calls(); !slices.Equal(got, want) {
		t.Fatalf("herdr calls = %q\nwant %q", got, want)
	}
	if got := h.typed(); !slices.Equal(got, []paneSendTextCall{{paneID: "pane-1", text: sendMarkerCodexShortMsg}}) {
		t.Fatalf("typed = %+v; want the message once into pane-1", got)
	}
	details := lastSentDetails(t, o, before)
	if !strings.Contains(details, "via=screen") || strings.Contains(details, "submit_unconfirmed") {
		t.Fatalf("sent Details = %q; want via=screen and no submit_unconfirmed", details)
	}
	if want := "via=screen herdr_state=working " + sendMarkerCodexShortMsg; details != want {
		t.Fatalf("sent Details = %q, want %q", details, want)
	}
	mustPaneLockFree(t, o, "pane-1")
}
