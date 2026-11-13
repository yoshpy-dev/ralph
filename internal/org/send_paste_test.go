package org

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// The tests of Send's bracketed paste for claude seats (send_paste.go; plan
// docs/plans/active/2026-11-13-send-claude-paste.md, AC1 to AC4 and AC7).

// pastedForTest is text between the bracketed paste markers, spelled out
// here as bytes rather than with the constants, as Send types it into the
// pane of a claude seat.
func pastedForTest(text string) string {
	return "\x1b[200~" + text + "\x1b[201~"
}

// typedForTest is what Send types into the pane of a seat whose driver is
// driver: pastedForTest(text) for claude, and text for any other driver.
func typedForTest(driver, text string) string {
	if driver == "claude" {
		return pastedForTest(text)
	}
	return text
}

// sendPasteLongMsg is a DECISION of 21 lines and more than 800 bytes, the
// size whose start Claude Code 2.1.316 dropped when it was typed plainly
// (docs/evidence/org-send-paste-probe-2026-11-13.md, A1).
var sendPasteLongMsg = func() string {
	var b strings.Builder
	b.WriteString("TYPE: DECISION\n\n")
	for i := range 18 {
		fmt.Fprintf(&b, "line %02d: the plan stays as approved, and the reviewer checks it again.\n", i)
	}
	b.WriteString("reply with the word tango")
	return b.String()
}()

// sendPasteMarkerErr is the error of a send whose text holds a bracketed
// paste marker.
const sendPasteMarkerErr = "org: send: the message contains a bracketed paste marker (ESC [200~ or ESC [201~), " +
	"which a seat's terminal reads as the start or the end of a paste; nothing was sent"

// TestSendTextForDriver_WrapsOnlyClaude covers AC1 and AC2 for the text
// itself: a driver of exactly "claude" gets the text between the two
// markers, a short one with no newline and one of many lines alike; codex,
// no driver, another driver, and a "Claude" spelled with a capital get it
// unchanged.
func TestSendTextForDriver_WrapsOnlyClaude(t *testing.T) {
	short := "TYPE: STOP"
	for _, tc := range []struct{ name, driver, text, want string }{
		{"claude, a short text with no newline", "claude", short, "\x1b[200~TYPE: STOP\x1b[201~"},
		{"claude, a text of many lines", "claude", sendPasteLongMsg, "\x1b[200~" + sendPasteLongMsg + "\x1b[201~"},
		{"claude, an empty text", "claude", "", "\x1b[200~\x1b[201~"},
		{"codex, a short text", "codex", short, short},
		{"codex, a text of many lines", "codex", sendPasteLongMsg, sendPasteLongMsg},
		{"no driver recorded", "", sendPasteLongMsg, sendPasteLongMsg},
		{"another driver", "gemini", sendPasteLongMsg, sendPasteLongMsg},
		{"claude with a capital", "Claude", short, short},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := sendTextForDriver(tc.driver, tc.text); got != tc.want {
				t.Fatalf("sendTextForDriver(%q, %q) = %q, want %q", tc.driver, tc.text, got, tc.want)
			}
		})
	}
}

// TestCheckNoPasteMarkers_RefusesEitherMarkerAnywhere covers AC4 for the
// check itself: either marker alone, at the start, in the middle, or at the
// end of the text is refused with the one error; a text with neither, and
// pieces of a marker (no ESC, no tilde, ESC alone), passes.
func TestCheckNoPasteMarkers_RefusesEitherMarkerAnywhere(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		refused    bool
	}{
		{"the start marker alone", "\x1b[200~", true},
		{"the end marker alone", "\x1b[201~", true},
		{"the start marker in the middle", "TYPE: DECISION\n\ngo \x1b[200~ on", true},
		{"the end marker in the middle", "TYPE: DECISION\n\ngo \x1b[201~ on", true},
		{"the end marker at the end", "TYPE: DECISION\n\ngo on\x1b[201~", true},
		{"both markers around the text", "\x1b[200~TYPE: STOP\x1b[201~", true},
		{"no marker", sendPasteLongMsg, false},
		{"a marker with no ESC", "TYPE: DECISION\n\n[200~ and [201~", false},
		{"a marker with no tilde", "TYPE: DECISION\n\n\x1b[200 and \x1b[201", false},
		{"ESC alone", "TYPE: DECISION\n\n\x1b[31mred", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkNoPasteMarkers(tc.text)
			if !tc.refused {
				if err != nil {
					t.Fatalf("checkNoPasteMarkers(%q) = %v, want nil", tc.text, err)
				}
				return
			}
			if err == nil || err.Error() != sendPasteMarkerErr {
				t.Fatalf("checkNoPasteMarkers(%q) = %v, want %q", tc.text, err, sendPasteMarkerErr)
			}
		})
	}
}

// TestOrgSend_PasteMarker_RefusedWithNothingSent covers AC4: a send whose
// text holds a bracketed paste marker is refused with the marker's error,
// whatever the seat's driver (claude, codex) and whether the seat exists,
// for Raw and DryRun too, before the watch gate of a TASK: nothing is
// typed, no manifest event is appended, herdr is not called, and Progress
// stays SendProgressNothingSent. An invalid message with a marker gets
// protocol validation's error, which comes first.
func TestOrgSend_PasteMarker_RefusedWithNothingSent(t *testing.T) {
	withStart := "TYPE: DECISION\n\nkeep \x1b[200~ going"
	withEnd := "TYPE: DECISION\n\nkeep \x1b[201~ going"
	task := "TYPE: TASK\nTASK_ID: t-1\n\ndo \x1b[201~ this"
	for _, tc := range []struct {
		name string
		p    SendParams
		want string
	}{
		{"the start marker, to a claude seat", SendParams{To: "seat-c", Text: withStart}, sendPasteMarkerErr},
		{"the end marker, to a claude seat", SendParams{To: "seat-c", Text: withEnd}, sendPasteMarkerErr},
		{"the end marker, to a codex seat", SendParams{To: "seat-x", Text: withEnd}, sendPasteMarkerErr},
		{"raw, the start marker alone", SendParams{To: "seat-c", Text: "\x1b[200~", Raw: true}, sendPasteMarkerErr},
		{"raw, the end marker alone", SendParams{To: "seat-x", Text: "\x1b[201~", Raw: true}, sendPasteMarkerErr},
		{"raw, a marker in the middle", SendParams{To: "seat-c", Text: "free text \x1b[201~ more", Raw: true}, sendPasteMarkerErr},
		{"dry run", SendParams{To: "seat-c", Text: withEnd, DryRun: true}, sendPasteMarkerErr},
		{"raw dry run", SendParams{To: "seat-x", Text: "\x1b[200~free", Raw: true, DryRun: true}, sendPasteMarkerErr},
		{"a seat the manifest does not have", SendParams{To: "no-such-seat", Text: withStart}, sendPasteMarkerErr},
		{"a TASK, before the watch gate", SendParams{To: "seat-c", Text: task}, sendPasteMarkerErr},
		{"an invalid message with a marker", SendParams{To: "seat-c", Text: "no type \x1b[201~"},
			"org: send: message rejected by protocol validation (use --raw to bypass): "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h := resumeTestOrg(t)
			seedResumeSeat(t, o, "org-a", "seat-c", "worker", "claude", "pane-1")
			seedResumeSeat(t, o, "org-a", "seat-x", "worker", "codex", "pane-2")
			gateCalled := false
			o.WatchGate = func() error {
				gateCalled = true
				return errors.New("org: stub: no watch")
			}
			before := len(mustReadEvents(t, o))

			p := tc.p
			p.OrgID = "org-a"
			p.TimeoutMS = 10000
			res := o.Send(p)
			if res.Err == nil || !strings.HasPrefix(res.Err.Error(), tc.want) {
				t.Fatalf("Send = %+v; want an error starting %q", res, tc.want)
			}
			if tc.want == sendPasteMarkerErr && res.Err.Error() != sendPasteMarkerErr {
				t.Fatalf("Send error = %q, want %q", res.Err, sendPasteMarkerErr)
			}
			if res.Progress != SendProgressNothingSent || res.PaneID != "" || res.SubmitConfirmed {
				t.Fatalf("Send = %+v; want nothing sent", res)
			}
			if got := mustReadEvents(t, o); len(got) != before {
				t.Fatalf("manifest events = %+v; want the %d seeded ones only", got, before)
			}
			if got := h.calls(); len(got) != 0 {
				t.Fatalf("herdr calls = %q; want none", got)
			}
			if len(h.fakeHerdr.calls) != 0 {
				t.Fatalf("herdr calls = %q; want none", h.fakeHerdr.calls)
			}
			if gateCalled {
				t.Fatal("the watch gate ran; want the refusal before it")
			}
		})
	}
}

// TestOrgSend_HerdrPath_TypesABracketedPasteOnlyForClaude covers AC1 to AC3
// on herdr's path: a claude seat gets the message between the markers in
// one PaneSendText call, a short one and one of more than 800 bytes alike;
// codex, no driver, and another driver get it as it is. Enter is pressed
// once either way, and the `sent` event's Details hold the message without
// the markers.
func TestOrgSend_HerdrPath_TypesABracketedPasteOnlyForClaude(t *testing.T) {
	short := "TYPE: QUESTION\n\nwhich plan?"
	for _, tc := range []struct{ name, driver, msg, typed string }{
		{"claude, short", "claude", short, "\x1b[200~" + short + "\x1b[201~"},
		{"claude, more than 800 bytes", "claude", sendPasteLongMsg, "\x1b[200~" + sendPasteLongMsg + "\x1b[201~"},
		{"codex", "codex", sendPasteLongMsg, sendPasteLongMsg},
		{"no driver recorded", "", sendPasteLongMsg, sendPasteLongMsg},
		{"another driver", "gemini", short, short},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h := resumeTestOrg(t)
			seedResumeSeat(t, o, "org-a", "seat-1", "worker", tc.driver, "pane-1")
			before := len(mustReadEvents(t, o))

			res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: tc.msg, TimeoutMS: 10000})
			if res.Err != nil || res.Progress != SendProgressEnterPressed || res.PaneID != "pane-1" {
				t.Fatalf("Send = %+v; want the message sent", res)
			}
			if got := resumeSends(h.calls()); !slices.Equal(got, []string{"text pane-1", "keys pane-1 Enter"}) {
				t.Fatalf("sends = %q; want the text and Enter once", got)
			}
			if got := h.typed(); !slices.Equal(got, []paneSendTextCall{{paneID: "pane-1", text: tc.typed}}) {
				t.Fatalf("typed = %+v; want %q once into pane-1", got, tc.typed)
			}
			details := lastSentDetails(t, o, before)
			if want := truncateForDetails(tc.msg); details != want {
				t.Fatalf("sent Details = %q, want %q", details, want)
			}
			if strings.Contains(details, "\x1b") {
				t.Fatalf("sent Details = %q; want no marker", details)
			}
			mustPaneLockFree(t, o, "pane-1")
		})
	}
}

// TestOrgSend_DryRun_ClaudeSeat_DetailsWithoutMarkers covers AC3 for a dry
// run to a claude seat: the dry-run event's Details hold the message as it
// was given, and herdr is not called.
func TestOrgSend_DryRun_ClaudeSeat_DetailsWithoutMarkers(t *testing.T) {
	o, h := resumeTestOrg(t)
	seedResumeSeat(t, o, "org-a", "seat-1", "worker", "claude", "pane-1")
	msg := "TYPE: QUESTION\n\nwhich plan?"
	before := len(mustReadEvents(t, o))

	if res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg, DryRun: true}); res.Err != nil {
		t.Fatalf("Send = %+v; want the dry run recorded", res)
	}
	events := mustReadEvents(t, o)[before:]
	if len(events) != 1 || events[0].Event != EventSent || !events[0].DryRun || events[0].Details != msg {
		t.Fatalf("new events = %+v; want one dry-run sent event whose Details are %q", events, msg)
	}
	if got := h.calls(); len(got) != 0 {
		t.Fatalf("herdr calls = %q; want none", got)
	}
}

// TestOrgSend_ScreenPath_ClaudeSeat_TypesABracketedPaste covers AC1 and AC3
// on the screen path: for a claude seat herdr keeps reporting working, Send
// types the message between the markers, the check before Enter finds the
// message without them in the input box (a short one as typed, one of many
// lines as claude's `[Pasted text #1 +N lines]` placeholder, N its
// newlines), Enter is pressed once, the submit is confirmed on the screen,
// and the `sent` Details hold the message without the markers.
func TestOrgSend_ScreenPath_ClaudeSeat_TypesABracketedPaste(t *testing.T) {
	short := "TYPE: QUESTION\n\nwhich plan?"
	for _, tc := range []struct{ name, msg, box string }{
		{"short, the message in the box", short, short},
		{"many lines, the paste placeholder in the box", sendPasteLongMsg,
			fmt.Sprintf("[Pasted text #1 +%d lines]", strings.Count(sendPasteLongMsg, "\n"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h, _ := sendFallbackOrg(t, "claude")
			idle := readResumeScreen(t, "claude-idle-after-reply.txt")
			h.setScreens("pane-1", idle, idle, idle, sendTypedScreen(t, "claude", idle, tc.box), readResumeScreen(t, "claude-submitted-just-now.txt"))
			before := len(mustReadEvents(t, o))

			res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: tc.msg, TimeoutMS: sendFallbackTimeoutMS})
			if res.Err != nil || res.Progress != SendProgressEnterPressed || !res.SubmitConfirmed || res.PaneID != "pane-1" {
				t.Fatalf("Send = %+v; want the message sent and the submit confirmed", res)
			}
			if got := resumeSends(h.calls()); !slices.Equal(got, []string{"text pane-1", "keys pane-1 Enter"}) {
				t.Fatalf("sends = %q; want the text and Enter once", got)
			}
			if got := h.typed(); !slices.Equal(got, []paneSendTextCall{{paneID: "pane-1", text: pastedForTest(tc.msg)}}) {
				t.Fatalf("typed = %+v; want the message once into pane-1 as a bracketed paste", got)
			}
			if got, want := lastSentDetails(t, o, before), "via=screen herdr_state=working "+truncateForDetails(tc.msg); got != want {
				t.Fatalf("sent Details = %q, want %q", got, want)
			}
			mustPaneLockFree(t, o, "pane-1")
		})
	}
}

// TestOrgSend_ClaudeSeat_SendTextFails_NoEnter covers AC7: when the
// PaneSendText of a claude seat's bracketed paste fails, on herdr's path and
// on the screen path, Send presses no Enter, returns
// SendProgressTextUnacknowledged with the pane, records no `sent` event, and
// releases the pane lock; the text it tried to type was the bracketed paste.
func TestOrgSend_ClaudeSeat_SendTextFails_NoEnter(t *testing.T) {
	msg := sendPasteLongMsg
	for _, tc := range []struct {
		name  string
		setup func(t *testing.T) (*Org, *fakeResumeHerdr, int)
	}{
		{"herdr's path", func(t *testing.T) (*Org, *fakeResumeHerdr, int) {
			o, h := resumeTestOrg(t)
			seedResumeSeat(t, o, "org-a", "seat-1", "worker", "claude", "pane-1")
			return o, h, 10000
		}},
		{"the screen path", func(t *testing.T) (*Org, *fakeResumeHerdr, int) {
			o, h, _ := sendFallbackOrg(t, "claude")
			idle := readResumeScreen(t, "claude-idle-after-reply.txt")
			h.setScreens("pane-1", idle)
			return o, h.fakeResumeHerdr, sendFallbackTimeoutMS
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, h, timeoutMS := tc.setup(t)
			h.sendTextErr["pane-1"] = errors.New("stub: send-text failed")
			before := len(mustReadEvents(t, o))

			res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg, TimeoutMS: timeoutMS})
			want := `org: send: send text to seat "seat-1": stub: send-text failed (the text may or may not have reached the pane)`
			if res.Err == nil || res.Err.Error() != want || res.Progress != SendProgressTextUnacknowledged || res.PaneID != "pane-1" || res.SubmitConfirmed {
				t.Fatalf("Send = %+v; want %q, SendProgressTextUnacknowledged, and pane-1", res, want)
			}
			if got := resumeSends(h.calls()); !slices.Equal(got, []string{"text pane-1"}) {
				t.Fatalf("sends = %q; want the text only, no Enter", got)
			}
			if got := h.typed(); !slices.Equal(got, []paneSendTextCall{{paneID: "pane-1", text: pastedForTest(msg)}}) {
				t.Fatalf("typed = %+v; want the bracketed paste tried once", got)
			}
			if got := mustReadEvents(t, o); len(got) != before {
				t.Fatalf("manifest events = %+v; want no new one", got[before:])
			}
			mustPaneLockFree(t, o, "pane-1")
		})
	}
}
