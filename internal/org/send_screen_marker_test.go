package org

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The tests of the check before Enter on a message with a line that starts
// with the input box's marker (sendTypedMarkerLineBox in send_screen.go;
// plan 2026-11-13-send-input-box-marker, issue #322).

// The messages typed into the seats of
// docs/evidence/org-send-marker-probe-2026-11-13.md, byte for byte. Each has
// a line that starts with the input box's marker of the seat's CLI: › for
// codex, ❯ for claude. TestSendMarkerProbeMessages_AsTyped checks their
// sizes against the evidence.
const (
	// codex short: the marker line in the middle.
	sendMarkerCodexShortMsg = `TYPE: QUESTION

› explain this
Reply with the single word: alpha.`

	// codex last: the marker line last.
	sendMarkerCodexLastMsg = `TYPE: QUESTION

Reply with the single word: bravo.
› explain this`

	// codex medium: lines 05 and 09 start with the marker; the box shows the
	// message from line 06 on.
	sendMarkerCodexMediumMsg = `TYPE: TASK
TASK_ID: t-322-med

line 01: the quick brown fox jumps over the lazy dog
line 02: the quick brown fox jumps over the lazy dog
line 03: the quick brown fox jumps over the lazy dog
line 04: the quick brown fox jumps over the lazy dog
› quoted line 05: the quick brown fox jumps over the lazy dog
line 06: the quick brown fox jumps over the lazy dog
line 07: the quick brown fox jumps over the lazy dog
line 08: the quick brown fox jumps over the lazy dog
› quoted line 09: the quick brown fox jumps over the lazy dog
line 10: the quick brown fox jumps over the lazy dog
Reply with the single word: charlie.`

	// codex numbered: the marker line is a numbered step.
	sendMarkerCodexNumberedMsg = `TYPE: TASK
TASK_ID: t-322-num

1. read the plan
› 2. explain this step
Reply with the single word: delta.`

	// codex, sent with Enter.
	sendMarkerCodexSubmitMsg = `TYPE: QUESTION

› explain this
This is a probe: do not run any tool or read any file. Reply with the single word: foxtrot.`

	// claude two-nl, one-nl, and no-nl: claude shows a message of two
	// newlines or fewer as text, not as a paste placeholder.
	sendMarkerClaudeTwoNLMsg = "TYPE: QUESTION\n\n❯ explain this"
	sendMarkerClaudeOneNLMsg = "TYPE: QUESTION\n❯ explain this"
	sendMarkerClaudeNoNLMsg  = "❯ explain this"

	// claude short: three newlines, which claude shows as
	// `[Pasted text #1 +3 lines]`.
	sendMarkerClaudeShortMsg = `TYPE: QUESTION

❯ explain this
Reply with the single word: alpha.`

	// claude, sent with Enter: the marker line wraps in the box.
	sendMarkerClaudeSubmitMsg = "TYPE: QUESTION\n\n❯ explain this. This is a probe: do not run any tool or read any file, and reply with the single word: echo."
)

// TestSendMarkerProbeMessages_AsTyped: the constants above have the sizes the
// evidence gives for the messages typed (its two tables of messages).
func TestSendMarkerProbeMessages_AsTyped(t *testing.T) {
	for _, tc := range []struct {
		name            string
		msg             string
		bytes, newlines int
	}{
		{"codex short", sendMarkerCodexShortMsg, 67, 3},
		{"codex last", sendMarkerCodexLastMsg, 67, 3},
		{"codex medium", sendMarkerCodexMediumMsg, 619, 13},
		{"codex numbered", sendMarkerCodexNumberedMsg, 107, 5},
		{"codex submit", sendMarkerCodexSubmitMsg, 124, 3},
		{"claude two-nl", sendMarkerClaudeTwoNLMsg, 32, 2},
		{"claude one-nl", sendMarkerClaudeOneNLMsg, 31, 1},
		{"claude no-nl", sendMarkerClaudeNoNLMsg, 16, 0},
		{"claude short", sendMarkerClaudeShortMsg, 67, 3},
		{"claude submit", sendMarkerClaudeSubmitMsg, 126, 2},
	} {
		if got, nl := len(tc.msg), strings.Count(tc.msg, "\n"); got != tc.bytes || nl != tc.newlines {
			t.Errorf("%s message: %d bytes, %d newlines; want %d and %d", tc.name, got, nl, tc.bytes, tc.newlines)
		}
	}
}

// sendTypedMarkerFixtures are the fixtures of a message with a marker line
// typed into an input box and not submitted
// (docs/evidence/org-send-marker-probe-2026-11-13.md), each with the message
// typed, the line of the box's head (its marker at column 0), and the
// head's text after the marker. sendTypedMarkerName matches their names.
var sendTypedMarkerFixtures = []struct {
	file, driver, msg, head, first string
}{
	// The marker line in the middle of the message, last, in a box scrolled
	// to line 06 (the scroll arrow on the head), and a numbered step, which
	// findResumeInputBox takes for a choice dialog's cursor line.
	{"codex-input-typed-marker-line.txt", "codex", sendMarkerCodexShortMsg, "› TYPE: QUESTION", "TYPE: QUESTION"},
	{"codex-input-typed-marker-line-last.txt", "codex", sendMarkerCodexLastMsg, "› TYPE: QUESTION", "TYPE: QUESTION"},
	{
		"codex-input-typed-marker-line-scrolled.txt", "codex", sendMarkerCodexMediumMsg,
		"› line 06: the quick brown fox jumps over the lazy   ↑", "line 06: the quick brown fox jumps over the lazy   ↑",
	},
	{"codex-input-typed-marker-numbered.txt", "codex", sendMarkerCodexNumberedMsg, "› TYPE: TASK", "TYPE: TASK"},
	// claude puts U+00A0 after the ❯ of the head. The marker line after a
	// blank line, right under the head, and wrapped onto three lines.
	{"claude-input-typed-marker-line.txt", "claude", sendMarkerClaudeTwoNLMsg, "❯\xc2\xa0TYPE: QUESTION", "TYPE: QUESTION"},
	{"claude-input-typed-marker-line-one-nl.txt", "claude", sendMarkerClaudeOneNLMsg, "❯\xc2\xa0TYPE: QUESTION", "TYPE: QUESTION"},
	{"claude-input-typed-marker-line-wrapped.txt", "claude", sendMarkerClaudeSubmitMsg, "❯\xc2\xa0TYPE: QUESTION", "TYPE: QUESTION"},
	// These two held before #322's fix too: the message's marker is on the
	// head's line, and claude shows the message of three newlines as a paste
	// placeholder (rule 3 of sendInputHolds).
	{"claude-input-typed-marker-no-nl.txt", "claude", sendMarkerClaudeNoNLMsg, "❯\xc2\xa0❯ explain this", "❯ explain this"},
	{"claude-input-typed-paste-short.txt", "claude", sendMarkerClaudeShortMsg, "❯\xc2\xa0[Pasted text #1 +3 lines]", "[Pasted text #1 +3 lines]"},
}

// sendTypedMarkerName matches the names of the fixtures that must be in
// sendTypedMarkerFixtures.
var sendTypedMarkerName = regexp.MustCompile(`-input-typed-(?:marker|paste)-`)

// TestSendInputHolds_TypedMarkerFixtures covers AC1, AC2, and AC4 of plan
// 2026-11-13-send-input-box-marker (issue #322): on every fixture of a
// message with a marker line typed and not submitted, the check before Enter
// finds the box at the head line, above the message's indented marker line
// (sendTypedInputBox), and holds; the check after Enter does not call the
// screen submitted. The head's text leaves out the U+00A0 claude puts after
// its marker. Every such fixture in testdata/resume/ must be in the table.
func TestSendInputHolds_TypedMarkerFixtures(t *testing.T) {
	dir := filepath.Join("testdata", "resume")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	listed := map[string]bool{}
	for _, f := range sendTypedMarkerFixtures {
		listed[f.file] = true
	}
	for _, e := range entries {
		if sendTypedMarkerName.MatchString(e.Name()) && !listed[e.Name()] {
			t.Errorf("fixture %s/%s is not in sendTypedMarkerFixtures", dir, e.Name())
		}
	}
	for _, f := range sendTypedMarkerFixtures {
		t.Run(f.file, func(t *testing.T) {
			screen := readResumeScreen(t, f.file)
			lines := resumeScreenLines(screen)
			box, ok := sendTypedInputBox(f.driver, lines, resumeDropSpace(f.msg))
			if !ok || lines[box.marker] != f.head || box.first != f.first {
				t.Fatalf("sendTypedInputBox = box at %q with text %q, %v; want the box at %q with text %q, true",
					lines[box.marker], box.first, ok, f.head, f.first)
			}
			if !sendInputHolds(f.driver, screen, f.msg) {
				t.Errorf("sendInputHolds = false, want true")
			}
			if SendScreenSubmitted(f.driver, screen, f.msg) {
				t.Errorf("SendScreenSubmitted = true, want false")
			}
		})
	}
}

// TestSendInputHolds_MarkerLineUnderANumberedHead: a box's head that is a
// numbered step of the message stays the head above the message's indented
// marker line (condition 5 of sendTypedMarkerLineBox). The screens are the
// numbered fixtures, whose head shows step 7 (claude) or step 2 with codex's
// scroll arrow, with step 8 starting with the marker in the box and in the
// message. The arrow is left out of the head's text, as on the numbered path
// of sendTypedInputBox.
func TestSendInputHolds_MarkerLineUnderANumberedHead(t *testing.T) {
	const step = "8. Fix what fails and run them again."
	if n := strings.Count(sendNumberedStepsMsg, "\n"+step); n != 1 {
		t.Fatalf("sendNumberedStepsMsg holds step 8 %d times; want once", n)
	}
	for _, tc := range []struct{ file, driver string }{
		{"claude-input-typed-numbered-steps.txt", "claude"},
		{"codex-input-typed-numbered-steps.txt", "codex"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			marker := resumeMarker(tc.driver)
			msg := strings.Replace(sendNumberedStepsMsg, "\n"+step, "\n"+marker+" "+step, 1)
			screen := sendMarkerEdit(t, tc.file, "\n  "+step, "\n  "+marker+" "+step)
			if !sendInputHolds(tc.driver, screen, msg) {
				t.Errorf("sendInputHolds = false, want true")
			}
			if SendScreenSubmitted(tc.driver, screen, msg) {
				t.Errorf("SendScreenSubmitted = true, want false")
			}
		})
	}
}

// sendMarkerEdit is fixture file with each pair of old and new text in edits
// replaced in turn; each old text must be in the screen exactly once.
func sendMarkerEdit(t *testing.T, file string, edits ...string) string {
	t.Helper()
	if len(edits)%2 != 0 {
		t.Fatalf("sendMarkerEdit(%s): %d edits; want pairs of old and new", file, len(edits))
	}
	screen := readResumeScreen(t, file)
	for i := 0; i < len(edits); i += 2 {
		if n := strings.Count(screen, edits[i]); n != 1 {
			t.Fatalf("%s holds %q %d times; want once", file, edits[i], n)
		}
		screen = strings.Replace(screen, edits[i], edits[i+1], 1)
	}
	return screen
}

// TestSendInputHolds_MarkerLineRefuses covers AC3 of plan
// 2026-11-13-send-input-box-marker: the check before Enter takes the marker
// line at column 0 above the message's indented marker line L for the box's
// head only when every condition of D2 holds (sendTypedMarkerLineBox). Each
// case is a fixture with lines edited or a made-up screen on which one
// condition fails; sendTypedMarkerLineBox takes no box there, and the check
// fails: L at column 0 (it is the head; here the box under the message sent
// before holds a part of it), L's text not part of the message, or nothing
// after L's marker, no marker line at column 0 above L or a line at column
// 0 between them, L outside the box (under claude's bottom rule, also one
// drawn with two spaces before it, or under codex's footer), no box around
// the head, a numbered head that is not part of the message, and a driver
// that is neither claude nor codex. The refusals of
// TestSendInputHolds_Refuses and TestSendInputHolds_NumberedMarkerLineRefuses
// stay as they were.
func TestSendInputHolds_MarkerLineRefuses(t *testing.T) {
	const (
		codexShort   = "codex-input-typed-marker-line.txt"
		claudeTwoNL  = "claude-input-typed-marker-line.txt"
		codexWarning = "\n                                            ⚠ 11 · f2\n"
		codexFooter  = "\n  GPT-6-Luna · low · Context 0% used · main · No chan…" + codexWarning
		claudeBoxEnd = "\n\n  ❯ explain this\n" + resumeTestRule + "\n"
	)
	cases := []struct{ name, driver, screen, typed string }{
		{
			"codex, a head at column 0 that holds a part of the message, under the message sent", "codex",
			sendMarkerEdit(t, "codex-submitted-marker-line.txt", "\n› Ask Codex to do anything\n", "\n› Reply with the single word: foxtrot.\n"),
			sendMarkerCodexSubmitMsg,
		},
		{
			"codex, the indented marker line not of the message", "codex",
			sendMarkerEdit(t, codexShort, "\n  › explain this\n", "\n  › explain that\n"), sendMarkerCodexShortMsg,
		},
		{
			"claude, the indented marker line not of the message", "claude",
			sendMarkerEdit(t, claudeTwoNL, "\n  ❯ explain this\n", "\n  ❯ explain that\n"), sendMarkerClaudeTwoNLMsg,
		},
		// L with nothing after the marker, in the message too. The check falls
		// back to findResumeInputBox, which takes L for the box's head: claude
		// draws no rule right above it, so there is no box, and codex's box
		// from L does not hold the start of the message.
		{
			"codex, nothing after the indented marker", "codex",
			sendMarkerEdit(t, codexShort, "\n  › explain this\n", "\n  ›\n"),
			strings.Replace(sendMarkerCodexShortMsg, "› explain this", "›", 1),
		},
		{
			"claude, nothing after the indented marker", "claude",
			sendMarkerEdit(t, claudeTwoNL, "\n  ❯ explain this\n", "\n  ❯\n"),
			strings.Replace(sendMarkerClaudeTwoNLMsg, "❯ explain this", "❯", 1),
		},
		{
			"codex, no marker line at column 0 above it", "codex",
			sendMarkerEdit(t, codexShort, "\n› TYPE: QUESTION\n", "\n  › TYPE: QUESTION\n"), sendMarkerCodexShortMsg,
		},
		{
			"claude, no marker line at column 0 above it", "claude",
			sendMarkerEdit(t, claudeTwoNL, "\n❯\xc2\xa0TYPE: QUESTION\n", "\n  ❯ TYPE: QUESTION\n"), sendMarkerClaudeTwoNLMsg,
		},
		{
			"codex, a line at column 0 between the head and it", "codex",
			sendMarkerEdit(t, codexShort, "\n› TYPE: QUESTION\n\n  › explain this\n", "\n› TYPE: QUESTION\n• Hi!\n  › explain this\n"),
			sendMarkerCodexShortMsg,
		},
		{
			"claude, the indented marker line under the bottom rule", "claude",
			sendMarkerEdit(t, claudeTwoNL, claudeBoxEnd, "\n"+resumeTestRule+"\n  ❯ explain this\n"), sendMarkerClaudeTwoNLMsg,
		},
		{
			"claude, the indented marker line under a bottom rule with two spaces before it", "claude",
			sendMarkerEdit(t, claudeTwoNL, claudeBoxEnd, "\n  "+resumeTestRule+"\n  ❯ explain this\n"), sendMarkerClaudeTwoNLMsg,
		},
		{
			"codex, the indented marker line under the footer", "codex",
			sendMarkerEdit(t, codexShort, "\n\n  › explain this\n", "\n\n", codexWarning, "\n  › explain this\n"), sendMarkerCodexShortMsg,
		},
		{
			"claude, no rule right above the head", "claude",
			sendMarkerEdit(t, claudeTwoNL, "\n"+resumeTestRule+"\n❯\xc2\xa0", "\n❯\xc2\xa0"), sendMarkerClaudeTwoNLMsg,
		},
		{
			"codex, no footer under the box", "codex",
			sendMarkerEdit(t, codexShort, codexFooter, "\n"), sendMarkerCodexShortMsg,
		},
		{
			"claude, a numbered head not of the message", "claude",
			sendClaudeBoxScreen(" 1. Yes, I trust this folder\n  TYPE: QUESTION\n\n  ❯ explain this"), sendMarkerClaudeTwoNLMsg,
		},
		{
			"codex, a numbered head not of the message", "codex",
			codexResumeScreen("› go\n\n  Worked for 4s • 18:17", " 1. Review hooks\n  TYPE: QUESTION\n\n  › explain this\n  Reply with the single word: alpha."),
			sendMarkerCodexShortMsg,
		},
		{
			"a driver that is neither claude nor codex, on claude's screen", "gemini",
			readResumeScreen(t, claudeTwoNL), sendMarkerClaudeTwoNLMsg,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines := resumeScreenLines(tc.screen)
			if box, ok := sendTypedMarkerLineBox(tc.driver, lines, resumeDropSpace(tc.typed)); ok {
				t.Errorf("sendTypedMarkerLineBox = box at %q, true; want false", lines[box.marker])
			}
			if sendInputHolds(tc.driver, tc.screen, tc.typed) {
				t.Errorf("sendInputHolds = true, want false")
			}
		})
	}
}
