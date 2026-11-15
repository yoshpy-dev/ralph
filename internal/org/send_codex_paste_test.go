package org

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"
)

// The tests of Send's bracketed paste for codex seats (send_paste.go and
// send_screen.go; plan 2026-11-15-send-codex-paste, AC1 to AC4; issue #326).
// The screens are the real ones of
// docs/evidence/org-send-codex-paste-probe-2026-11-15.md (codex-cli 0.166.0,
// herdr 0.7.5, a pane 53 columns wide), and the messages are the ones typed
// there, byte for byte (TestSendCodexPasteProbeMessages_AsTyped).

// sendCodexPasteShortMsg is p1 of the evidence. Pasted, it showed as its text
// in the box (codex-input-pasted-short.txt).
const sendCodexPasteShortMsg = `TYPE: QUESTION

This is a probe: do not run any tool or read any file. Reply with the single word: alpha.`

// sendCodexPasteMediumMsg is p2 of the evidence. Pasted, the box showed its
// last lines with the scroll arrow on the first
// (codex-input-pasted-medium.txt).
const sendCodexPasteMediumMsg = `TYPE: TASK
TASK_ID: t-326-med

line 01: the quick brown fox jumps over the lazy dog again
line 02: the quick brown fox jumps over the lazy dog again
line 03: the quick brown fox jumps over the lazy dog again
line 04: the quick brown fox jumps over the lazy dog again
line 05: the quick brown fox jumps over the lazy dog again
line 06: the quick brown fox jumps over the lazy dog again
line 07: the quick brown fox jumps over the lazy dog again
line 08: the quick brown fox jumps over the lazy dog again
line 09: the quick brown fox jumps over the lazy dog again
This is a probe: do not run any tool or read any file. Reply with the single word: bravo.`

// sendCodexPasteLongMsg is p3 of the evidence. Pasted, the box showed
// `[Pasted Content 1698 chars]` (codex-input-pasted-placeholder-ascii.txt).
const sendCodexPasteLongMsg = `TYPE: TASK
TASK_ID: t-326-long

line 01: the quick brown fox jumps over the lazy dog once more
line 02: the quick brown fox jumps over the lazy dog once more
line 03: the quick brown fox jumps over the lazy dog once more
line 04: the quick brown fox jumps over the lazy dog once more
line 05: the quick brown fox jumps over the lazy dog once more
line 06: the quick brown fox jumps over the lazy dog once more
line 07: the quick brown fox jumps over the lazy dog once more
line 08: the quick brown fox jumps over the lazy dog once more
line 09: the quick brown fox jumps over the lazy dog once more
line 10: the quick brown fox jumps over the lazy dog once more
line 11: the quick brown fox jumps over the lazy dog once more
line 12: the quick brown fox jumps over the lazy dog once more
line 13: the quick brown fox jumps over the lazy dog once more
line 14: the quick brown fox jumps over the lazy dog once more
line 15: the quick brown fox jumps over the lazy dog once more
line 16: the quick brown fox jumps over the lazy dog once more
line 17: the quick brown fox jumps over the lazy dog once more
line 18: the quick brown fox jumps over the lazy dog once more
line 19: the quick brown fox jumps over the lazy dog once more
line 20: the quick brown fox jumps over the lazy dog once more
line 21: the quick brown fox jumps over the lazy dog once more
line 22: the quick brown fox jumps over the lazy dog once more
line 23: the quick brown fox jumps over the lazy dog once more
line 24: the quick brown fox jumps over the lazy dog once more
line 25: the quick brown fox jumps over the lazy dog once more
This is a probe: do not run any tool or read any file. Reply with the single word: charlie.`

// sendCodexPasteJapaneseMsg is p4 of the evidence: a DECISION whose body is
// 2000 runes of Japanese with one emoji, so its runes (2016) and its UTF-16
// units (2017) differ. Pasted, the box showed `[Pasted Content 2016 chars]`
// (codex-input-pasted-placeholder.txt).
var sendCodexPasteJapaneseMsg = func() string {
	const line = "これは codex の座席に貼り付けで送る確かめの文で、頭が欠けずに届くかと、Enter で送られるかを見ます。"
	var b strings.Builder
	b.WriteString("TYPE: DECISION\n\n")
	b.WriteString("第01行 😀: " + line + strings.Repeat("あ", 62) + "\n")
	for i := 2; i <= 30; i++ {
		fmt.Fprintf(&b, "第%02d行: %s\n", i, line)
	}
	b.WriteString("最後の行です。道具を使わず、ファイルも読まずに、返事には「delta」とだけ書いてください。")
	return b.String()
}()

// TestSendCodexPasteProbeMessages_AsTyped: the messages above are the files
// typed in the evidence (their SHA-256), with the sizes its table gives;
// the runes of each are the number codex showed in its placeholder.
func TestSendCodexPasteProbeMessages_AsTyped(t *testing.T) {
	for _, tc := range []struct {
		name                          string
		msg                           string
		bytes, runes, units, newlines int
		sum                           string
	}{
		{"p1 short", sendCodexPasteShortMsg, 105, 105, 105, 2, "9f70feffefd6f3af477fa958c5ddface4bf7816b805d2db0c947aa9c067d3cdc"},
		{"p2 medium", sendCodexPasteMediumMsg, 651, 651, 651, 12, "6a248d3fc08dd7e5771c987aad5d60292ded229643f69163634bb4edc2bdd8ae"},
		{"p3 long", sendCodexPasteLongMsg, 1698, 1698, 1698, 28, "5b2f70059179ef31b562945f22fa1a906d60a941357127fa38d70af3d56fd9b3"},
		{"p4 japanese", sendCodexPasteJapaneseMsg, 4925, 2016, 2017, 32, "944850c89e79fea53626a170f1a56099e9fbce4ff62f17918f12019903a10cc0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runes, units := utf8.RuneCountInString(tc.msg), len(utf16.Encode([]rune(tc.msg)))
			if len(tc.msg) != tc.bytes || runes != tc.runes || units != tc.units || strings.Count(tc.msg, "\n") != tc.newlines {
				t.Errorf("%d bytes, %d runes, %d UTF-16 units, %d newlines; want %d, %d, %d, %d",
					len(tc.msg), runes, units, strings.Count(tc.msg, "\n"), tc.bytes, tc.runes, tc.units, tc.newlines)
			}
			if sum := sha256.Sum256([]byte(tc.msg)); hex.EncodeToString(sum[:]) != tc.sum {
				t.Errorf("SHA-256 %x, want %s (the file typed in the evidence)", sum, tc.sum)
			}
		})
	}
}

// sendCodexPastedFixtures are the screens of the evidence 0.75 s after a
// message was typed as a bracketed paste, before Enter, each with that
// message. placeholder is whether codex showed the paste as one placeholder;
// the other two showed its text, which rules 1 and 2 of sendInputHolds hold
// for.
var sendCodexPastedFixtures = []struct {
	file, msg   string
	placeholder bool
}{
	{"codex-input-pasted-placeholder.txt", sendCodexPasteJapaneseMsg, true},
	{"codex-input-pasted-placeholder-ascii.txt", sendCodexPasteLongMsg, true},
	{"codex-input-pasted-short.txt", sendCodexPasteShortMsg, false},
	{"codex-input-pasted-medium.txt", sendCodexPasteMediumMsg, false},
}

// TestSendInputHolds_CodexPastedFixtures covers AC3 on the real screens:
// before Enter, the check holds for each message pasted, by the placeholder
// rule (rule 4) for the two placeholders and by the existing rules for the
// two texts; ClassifySendScreen reads each box as text, and
// SendScreenSubmitted never calls one submitted. 0.3 s after Enter, the box
// is empty under a working line, and SendScreenSubmitted holds.
func TestSendInputHolds_CodexPastedFixtures(t *testing.T) {
	for _, f := range sendCodexPastedFixtures {
		t.Run(f.file, func(t *testing.T) {
			screen := readResumeScreen(t, f.file)
			if !sendInputHolds("codex", screen, f.msg) {
				t.Errorf("sendInputHolds = false, want true")
			}
			lines := resumeScreenLines(screen)
			box, ok := sendTypedInputBox("codex", lines, resumeDropSpace(f.msg))
			if !ok {
				t.Fatalf("sendTypedInputBox found no box")
			}
			got, _ := sendCodexTypedInputText(lines, box)
			if rule4 := sendCodexPasteHolds(got, f.msg); rule4 != f.placeholder {
				t.Errorf("sendCodexPasteHolds(%q) = %v, want %v", got, rule4, f.placeholder)
			}
			if f.placeholder && ResumeInputHolds("codex", screen, f.msg) {
				t.Errorf("ResumeInputHolds = true, want false (the box holds the placeholder, not the message's start)")
			}
			if s := ClassifySendScreen("codex", screen); s.Kind != SendScreenUnknown || s.Reason != sendScreenReasonInputHasText {
				t.Errorf("ClassifySendScreen = %+v, want unknown (%s)", s, sendScreenReasonInputHasText)
			}
			if SendScreenSubmitted("codex", screen, f.msg) {
				t.Errorf("SendScreenSubmitted = true, want false")
			}
		})
	}
	t.Run("codex-submitted-pasted.txt", func(t *testing.T) {
		screen := readResumeScreen(t, "codex-submitted-pasted.txt")
		if !SendScreenSubmitted("codex", screen, sendCodexPasteJapaneseMsg) {
			t.Errorf("SendScreenSubmitted = false, want true")
		}
		if sendInputHolds("codex", screen, sendCodexPasteJapaneseMsg) {
			t.Errorf("sendInputHolds = true, want false (the box is empty)")
		}
	})
}

// sendCodexPlaceholderLine is the line of codex-input-pasted-placeholder.txt
// that holds its input box's head, and sendCodexPlaceholderASCIILine that of
// codex-input-pasted-placeholder-ascii.txt.
const (
	sendCodexPlaceholderLine      = "\n› [Pasted Content 2016 chars]\n"
	sendCodexPlaceholderASCIILine = "\n› [Pasted Content 1698 chars]\n"
)

// TestSendInputHolds_CodexPlaceholderRefuses covers AC3's refusals: the
// real placeholder screens with the box's line edited fail the check before
// Enter when the number is one more or one less than the message's runes,
// equals the message's UTF-16 units (2017 for p4) or bytes, or counts the
// markers; when the box holds two placeholders, other text after or before
// the placeholder, or claude's form of placeholder; and when the message
// is another one. The unedited screens are the controls
// (TestSendInputHolds_CodexPastedFixtures).
func TestSendInputHolds_CodexPlaceholderRefuses(t *testing.T) {
	const (
		jp    = "codex-input-pasted-placeholder.txt"
		ascii = "codex-input-pasted-placeholder-ascii.txt"
	)
	box := func(input string) string { return "\n› " + input + "\n" }
	cases := []struct{ name, screen, typed string }{
		{"p4, one less than its runes", sendMarkerEdit(t, jp, sendCodexPlaceholderLine, box("[Pasted Content 2015 chars]")), sendCodexPasteJapaneseMsg},
		{"p4, its UTF-16 units, one more than its runes", sendMarkerEdit(t, jp, sendCodexPlaceholderLine, box("[Pasted Content 2017 chars]")), sendCodexPasteJapaneseMsg},
		{"p4, its bytes", sendMarkerEdit(t, jp, sendCodexPlaceholderLine, box("[Pasted Content 4925 chars]")), sendCodexPasteJapaneseMsg},
		{"p3, one less than its runes", sendMarkerEdit(t, ascii, sendCodexPlaceholderASCIILine, box("[Pasted Content 1697 chars]")), sendCodexPasteLongMsg},
		{"p3, one more than its runes", sendMarkerEdit(t, ascii, sendCodexPlaceholderASCIILine, box("[Pasted Content 1699 chars]")), sendCodexPasteLongMsg},
		{"p4 typed with the markers counted", readResumeScreen(t, jp), pastedForTest(sendCodexPasteJapaneseMsg)},
		{"p4, two placeholders on the line",
			sendMarkerEdit(t, jp, sendCodexPlaceholderLine, box("[Pasted Content 2016 chars][Pasted Content 2016 chars]")), sendCodexPasteJapaneseMsg},
		{"p4, two placeholders with a space between",
			sendMarkerEdit(t, jp, sendCodexPlaceholderLine, box("[Pasted Content 2016 chars] [Pasted Content 2016 chars]")), sendCodexPasteJapaneseMsg},
		{"p4, the placeholder followed by other text",
			sendMarkerEdit(t, jp, sendCodexPlaceholderLine, box("[Pasted Content 2016 chars] and more")), sendCodexPasteJapaneseMsg},
		{"p4, the placeholder followed by text on the next line",
			sendMarkerEdit(t, jp, sendCodexPlaceholderLine, box("[Pasted Content 2016 chars]\n  and more")), sendCodexPasteJapaneseMsg},
		{"p4, the placeholder preceded by text",
			sendMarkerEdit(t, jp, sendCodexPlaceholderLine, box("go [Pasted Content 2016 chars]")), sendCodexPasteJapaneseMsg},
		{"p4, claude's form of placeholder",
			sendMarkerEdit(t, jp, sendCodexPlaceholderLine, box("[Pasted text #1 +32 lines]")), sendCodexPasteJapaneseMsg},
		{"p3's screen with p4", readResumeScreen(t, ascii), sendCodexPasteJapaneseMsg},
		{"p4's screen with p3", readResumeScreen(t, jp), sendCodexPasteLongMsg},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if sendInputHolds("codex", tc.screen, tc.typed) {
				t.Errorf("sendInputHolds = true, want false")
			}
		})
	}
}

// TestSendScreenSubmitted_CodexPlaceholderStillInTheBox covers AC3 after
// Enter: a codex box that still shows a paste placeholder is never
// submitted, whatever number it shows and with a working line above it,
// which by itself would count as submitted (the control: the same screen
// with the box empty).
func TestSendScreenSubmitted_CodexPlaceholderStillInTheBox(t *testing.T) {
	const jp = "codex-input-pasted-placeholder.txt"
	const workedLine, workingLine = "  Worked for 2s • 20:40\n", "• Working (3s • esc to interrupt)\n"
	working := func(input string) string {
		return sendMarkerEdit(t, jp, workedLine, workingLine, sendCodexPlaceholderLine, "\n› "+input+"\n")
	}
	if !SendScreenSubmitted("codex", working("Ask Codex to do anything"), sendCodexPasteJapaneseMsg) {
		t.Fatalf("SendScreenSubmitted of the working screen with an empty box = false; want true (the control)")
	}
	for _, input := range []string{
		"[Pasted Content 2016 chars]",
		"[Pasted Content 2015 chars]",
		"[Pasted Content 2017 chars]",
		"[Pasted Content 1 chars]",
		"[Pasted Content 99999 chars]",
		"[Pasted Content 2016 chars] and more",
		"go [Pasted Content 7 chars]",
	} {
		t.Run(input, func(t *testing.T) {
			if SendScreenSubmitted("codex", working(input), sendCodexPasteJapaneseMsg) {
				t.Errorf("SendScreenSubmitted with %q in the box under a working line = true, want false", input)
			}
		})
	}
}

// TestOrgSendScreen_CodexPastePlaceholder_EnterPressedAndConfirmed covers AC4
// and AC2: on the screen path, a codex seat gets p4 (a DECISION of 2000
// Japanese runes) as a bracketed paste. The pane shows the real screens of
// the evidence: before Enter, `[Pasted Content 2016 chars]` in the box;
// 0.3 s after Enter, an empty box under a working line. The check before
// Enter reads the message without the markers, so Send presses Enter once,
// confirms the submit on the screen, and records via=screen with the
// message without the markers and no submit_unconfirmed.
func TestOrgSendScreen_CodexPastePlaceholder_EnterPressedAndConfirmed(t *testing.T) {
	o, h, name := sendFallbackOrg(t, "codex")
	idle := readResumeScreen(t, "codex-idle-after-reply.txt")
	h.setScreens("pane-1", idle, idle, idle,
		readResumeScreen(t, "codex-input-pasted-placeholder.txt"), readResumeScreen(t, "codex-submitted-pasted.txt"))
	before := len(mustReadEvents(t, o))

	msg := sendCodexPasteJapaneseMsg
	res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg, TimeoutMS: sendFallbackTimeoutMS})
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
	if got := h.typed(); !slices.Equal(got, []paneSendTextCall{{paneID: "pane-1", text: pastedForTest(msg)}}) {
		t.Fatalf("typed = %+v; want the message once into pane-1 as a bracketed paste", got)
	}
	details := lastSentDetails(t, o, before)
	if !strings.Contains(details, "via=screen") || strings.Contains(details, "submit_unconfirmed") || strings.Contains(details, "\x1b") {
		t.Fatalf("sent Details = %q; want via=screen, no submit_unconfirmed, and no marker", details)
	}
	if want := "via=screen herdr_state=working " + truncateForDetails(msg); details != want {
		t.Fatalf("sent Details = %q, want %q", details, want)
	}
	mustPaneLockFree(t, o, "pane-1")
}

// TestOrgSendScreen_CodexPastePlaceholderOfAnotherSize_EnterNotPressed covers
// D2's refusal through Send: when codex's box shows a placeholder whose
// number is not the message's runes (here its UTF-16 units), Enter is not
// pressed, Send returns SendProgressTextTyped with the pane and the error
// of a text not found in the box, nothing is recorded, and the lock is
// released.
func TestOrgSendScreen_CodexPastePlaceholderOfAnotherSize_EnterNotPressed(t *testing.T) {
	o, h, name := sendFallbackOrg(t, "codex")
	idle := readResumeScreen(t, "codex-idle-after-reply.txt")
	typed := sendMarkerEdit(t, "codex-input-pasted-placeholder.txt", sendCodexPlaceholderLine, "\n› [Pasted Content 2017 chars]\n")
	h.setScreens("pane-1", idle, idle, idle, typed)
	before := len(mustReadEvents(t, o))

	res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: sendCodexPasteJapaneseMsg, TimeoutMS: sendFallbackTimeoutMS})
	wantErr := `org: send: check before Enter for seat "seat-1": the typed text was not found in the input box, so Enter was not pressed (text typed but not submitted)`
	if res.Err == nil || res.Err.Error() != wantErr || !errors.Is(res.Err, errSendTextNotInInput) {
		t.Fatalf("Send error = %v\nwant %s", res.Err, wantErr)
	}
	if res.Progress != SendProgressTextTyped || res.PaneID != "pane-1" || res.SubmitConfirmed {
		t.Fatalf("Send = %+v; want SendProgressTextTyped with pane-1", res)
	}
	want := []string{
		"wait " + name + " idle,done", "get " + name, "read pane-1 60", "read pane-1 60",
		"read pane-1 60", "text pane-1", "read pane-1 60",
	}
	if got := h.calls(); !slices.Equal(got, want) {
		t.Fatalf("herdr calls = %q\nwant %q (no Enter)", got, want)
	}
	if got := len(mustReadEvents(t, o)); got != before {
		t.Fatalf("events %d -> %d; want none recorded", before, got)
	}
	mustPaneLockFree(t, o, "pane-1")
}

// TestOrgSend_DryRun_CodexSeat_DetailsWithoutMarkers covers AC2 for a dry
// run to a codex seat: the dry-run event's Details hold the message as it
// was given, and herdr is not called.
func TestOrgSend_DryRun_CodexSeat_DetailsWithoutMarkers(t *testing.T) {
	o, h := resumeTestOrg(t)
	seedResumeSeat(t, o, "org-a", "seat-1", "worker", "codex", "pane-1")
	msg := sendCodexPasteShortMsg
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

// TestOrgSend_CodexSeat_SendTextFails_NoEnter: when the PaneSendText of a
// codex seat's bracketed paste fails on herdr's path, Send presses no
// Enter, returns SendProgressTextUnacknowledged with the pane, records no
// `sent` event, and releases the pane lock; the text it tried to type was
// the bracketed paste.
func TestOrgSend_CodexSeat_SendTextFails_NoEnter(t *testing.T) {
	o, h := resumeTestOrg(t)
	seedResumeSeat(t, o, "org-a", "seat-1", "worker", "codex", "pane-1")
	h.sendTextErr["pane-1"] = errors.New("stub: send-text failed")
	msg := sendCodexPasteJapaneseMsg
	before := len(mustReadEvents(t, o))

	res := o.Send(SendParams{OrgID: "org-a", To: "seat-1", Text: msg, TimeoutMS: 10000})
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
}
