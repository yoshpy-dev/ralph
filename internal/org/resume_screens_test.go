package org

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resumeTestSentence is the resume sentence of the spec (FR-5), typed into
// the input boxes of the ResumeInputHolds fixtures.
const resumeTestSentence = "`ralph org resume` からの続行の指示です。直前の API エラー(通信断か利用上限)で作業が止まっていました。止まったところから続けてください。"

// resumeTestRule is the rule claude 2.1.312 draws above and below its input
// box (evidence 2026-11-07 section 2).
const resumeTestRule = "─────────────────────────────────────────────────────"

func readResumeScreen(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "resume", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return string(b)
}

// claudeResumeScreen is a claude screen with above as the history, then the
// input box holding input (after ❯) between two rules, then a footer.
func claudeResumeScreen(above, input string) string {
	return above + "\n\n" + resumeTestRule + "\n❯" + input + "\n" + resumeTestRule +
		"\n  🤖 Haiku 5.5 | ⚡ medium\n  ⏵⏵ bypass permissions on (shift+tab to cycle)\n"
}

// codexResumeScreen is a codex screen with above as the history, then the
// composer line holding input (after ›), then a footer.
func codexResumeScreen(above, input string) string {
	return above + "\n\n\n›" + input + "\n\n  GPT-6-Astra · xhigh · Context 0% used\n  ? for shortcuts\n"
}

func assertResumeScreen(t *testing.T, got ResumeScreen, kind ResumeScreenKind, line, reason string) {
	t.Helper()
	if got.Kind != kind || got.Line != line || got.Reason != reason {
		t.Errorf("ClassifyResumeScreen = {Kind:%q Line:%q Reason:%q}, want {Kind:%q Line:%q Reason:%q}",
			got.Kind, got.Line, got.Reason, kind, line, reason)
	}
}

func TestClassifyResumeScreen_Fixtures(t *testing.T) {
	cases := []struct {
		name   string
		file   string
		origin string
		driver string
		kind   ResumeScreenKind
		line   string
		reason string
	}{
		{
			name: "claude ENOTFOUND error with a status line is stopped", file: "claude-enotfound.txt",
			origin: "evidence 2026-11-07 section 2, verbatim", driver: "claude", kind: ResumeScreenStopped,
			line: "⏺ API Error: Can't reach the API server — check your internet or DNS (ENOTFOUND)",
		},
		{
			name: "claude retrying is waiting", file: "claude-retrying.txt",
			origin: "evidence 2026-11-07 section 3, verbatim", driver: "claude", kind: ResumeScreenWaiting,
			line: "✻ Can't reach the… · Retrying in 4s · attempt 13/300", reason: resumeReasonWaiting,
		},
		{
			name: "codex reconnecting is waiting", file: "codex-network-waiting.txt",
			origin: "evidence 2026-11-07 section 4, second screen, verbatim", driver: "codex", kind: ResumeScreenWaiting,
			line:   "• Reconnecting... waiting for network (2m 56s • esc t… └ Connection failed: error sending request └ Tip: Use /mcp to list configured MCP tools.",
			reason: resumeReasonWaiting,
		},
		{
			name: "codex hooks review dialog is unknown", file: "codex-hooks-review.txt",
			origin: "evidence 2026-11-07 section 4, first screen, verbatim", driver: "codex", kind: ResumeScreenUnknown,
			reason: resumeReasonChoiceDialog,
		},
		{
			name: "claude session limit before a status line is stopped", file: "claude-session-limit.txt",
			origin: "synthetic, docs errors#youve-hit-your-session-limit", driver: "claude", kind: ResumeScreenStopped,
			line: "⏺ You've hit your session limit · resets 3:45pm",
		},
		{
			name: "claude press enter in the footer is enter_only", file: "claude-press-enter.txt",
			origin: "synthetic, docs interactive-mode#wait-for-a-usage-limit-to-reset", driver: "claude", kind: ResumeScreenEnterOnly,
			line: "Your usage limit has reset Press enter to continue",
		},
		{
			name: "claude continuing automatically is waiting", file: "claude-continuing-automatically.txt",
			origin: "synthetic, docs interactive-mode#wait-for-a-usage-limit-to-reset", driver: "claude", kind: ResumeScreenWaiting,
			line:   "Usage limit reached · limit resets 3:45pm Continuing automatically at 3:45pm · Esc to cancel",
			reason: resumeReasonWaiting,
		},
		{
			name: "claude input box with text after the error is unknown", file: "claude-input-has-text.txt",
			origin: "synthetic", driver: "claude", kind: ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		{
			name: "claude folder trust dialog is unknown", file: "claude-folder-trust.txt",
			origin: "synthetic", driver: "claude", kind: ResumeScreenUnknown, reason: resumeReasonChoiceDialog,
		},
		{
			name: "codex usage limit above an empty composer is stopped", file: "codex-usage-limit.txt",
			origin: "synthetic, binary strings codex-cli 0.165.0", driver: "codex", kind: ResumeScreenStopped,
			line: "■ You've hit your usage limit. Upgrade to Pro or purchase more credits, or try again at 3:05 PM.",
		},
		{
			name: "claude old error in the history above a normal reply is unknown", file: "claude-old-error-in-history.txt",
			origin: "synthetic", driver: "claude", kind: ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "claude dialog below an old input box is unknown", file: "claude-dialog-below-old-input.txt",
			origin: "synthetic", driver: "claude", kind: ResumeScreenUnknown, reason: resumeReasonChoiceDialog,
		},
		{
			name: "claude input box holding the wrapped resume sentence is unknown", file: "claude-input-typed-wrapped.txt",
			origin: "synthetic", driver: "claude", kind: ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		{
			name: "codex composer holding the wrapped resume sentence is unknown", file: "codex-input-typed.txt",
			origin: "synthetic", driver: "codex", kind: ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		{
			// The ⎿ line starts a block of its own, so Line holds the limit
			// error and not the resume sentence above it. Each line is
			// trimmed and the lines are joined with single spaces; the two
			// spaces after ⎿ are inside one line and stay.
			name: "claude weekly limit right under the prompt is stopped without the prompt", file: "claude-weekly-limit-under-prompt.txt",
			origin: "evidence 2026-11-07 live, section 1 steps 3 and 6, verbatim; input box and footer of claude-enotfound.txt",
			driver: "claude", kind: ResumeScreenStopped,
			line: "⎿  You've hit your weekly limit · resets 11pm (Asia/Tokyo) /usage-credits to finish what you’re working on.",
		},
		{
			name: "claude prompt quoting an error, its tool result, and a normal reply is unknown", file: "claude-prompt-quotes-error-then-reply.txt",
			origin: "synthetic", driver: "claude", kind: ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "claude prompt quoting a limit as the last block is unknown", file: "claude-prompt-quotes-limit-last.txt",
			origin: "synthetic", driver: "claude", kind: ResumeScreenUnknown, reason: resumeReasonPromptLast,
		},
		{
			name: "claude prompt quoting an error right above an interruption is unknown", file: "claude-prompt-quotes-error-interrupted.txt",
			origin: "synthetic", driver: "claude", kind: ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			// A stopped sentence counts only at the start of the block,
			// after the ⏺; this reply quotes one later in its text.
			name: "claude normal reply quoting an API error is unknown", file: "claude-reply-quotes-error.txt",
			origin: "synthetic, cross-review cycle 1 finding 4", driver: "claude", kind: ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			// The blank line sets the quoted error apart as a block of its
			// own, which starts with the sentence but with no glyph.
			name: "claude reply whose last paragraph is an API error is unknown", file: "claude-reply-error-paragraph.txt",
			origin: "synthetic, cross-review cycle 2 finding 2", driver: "claude", kind: ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "claude tool output whose last paragraph is an API error is unknown", file: "claude-tool-output-error-paragraph.txt",
			origin: "synthetic, cross-review cycle 2 finding 2", driver: "claude", kind: ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "claude draft on the line under an empty ❯ line has text", file: "claude-input-draft-after-empty-line.txt",
			origin: "synthetic, input box and footer of claude-enotfound.txt", driver: "claude", kind: ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		{
			// The blank line under the draft ends the input, as the blank
			// line above the footer does in evidence section 4.
			name: "codex draft on the line under an empty › line has text", file: "codex-input-draft-after-empty-line.txt",
			origin: "synthetic, cross-review cycle 1 finding 2; history and footer of codex-usage-limit.txt", driver: "codex", kind: ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		{
			// With no footer on the screen, the draft is the last run of
			// non-blank lines after a blank line; it has no footer shape, so
			// it is input and not a footer.
			name: "codex draft after a blank line under an empty › line with no footer has text", file: "codex-input-draft-no-footer.txt",
			origin: "synthetic, cycle 2 self-review C2-4; history of codex-usage-limit.txt", driver: "codex", kind: ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		// The codex-cli 0.166.0 screens of evidence 2026-11-12. Their footer
		// goes on after `Context <n>% used` (section 6), and until
		// resumeCodexFooterLine took that form every one of them was
		// `input box has text`. Now the composer is found and its
		// placeholder reads empty, and none of them is stopped.
		{
			name: "codex 0.166 idle after a reply has no stop", file: "codex-idle-after-reply.txt",
			origin: "evidence 2026-11-12 section 2, verbatim", driver: "codex", kind: ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "codex 0.166 fresh session has no stop", file: "codex-idle-fresh-update-notice.txt",
			origin: "evidence 2026-11-12 section 1, verbatim", driver: "codex", kind: ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "codex 0.166 idle with a background terminal has no stop", file: "codex-idle-background-terminal.txt",
			origin: "evidence 2026-11-12 section 5, verbatim", driver: "codex", kind: ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "codex 0.166 working has no stop", file: "codex-working-tool.txt",
			origin: "evidence 2026-11-12 section 3, verbatim", driver: "codex", kind: ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "codex 0.166 just submitted has no stop", file: "codex-submitted-just-now.txt",
			origin: "evidence 2026-11-12 section 6, verbatim", driver: "codex", kind: ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			// `tab to queue message` is no footer line, so the footer is not
			// found and its lines count as input; the typed text is input
			// either way.
			name: "codex 0.166 composer holding typed text has text", file: "codex-working-input-typed.txt",
			origin: "evidence 2026-11-12 section 5, verbatim", driver: "codex", kind: ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		{
			name: "codex 0.166 update notice dialog is unknown", file: "codex-update-available.txt",
			origin: "evidence 2026-11-12 section 1, verbatim", driver: "codex", kind: ResumeScreenUnknown, reason: resumeReasonChoiceDialog,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			screen := readResumeScreen(t, tc.file)
			assertResumeScreen(t, ClassifyResumeScreen(tc.driver, screen), tc.kind, tc.line, tc.reason)
			if t.Failed() {
				t.Logf("fixture testdata/resume/%s: %s", tc.file, tc.origin)
			}
		})
	}
}

func TestClassifyResumeScreen_Edges(t *testing.T) {
	enotfound := readResumeScreen(t, "claude-enotfound.txt")
	enotfoundLine := "⏺ API Error: Can't reach the API server — check your internet or DNS (ENOTFOUND)"
	cases := []struct {
		name   string
		driver string
		screen string
		kind   ResumeScreenKind
		line   string
		reason string
	}{
		{name: "empty screen, claude", driver: "claude", screen: "", kind: ResumeScreenUnknown, reason: resumeReasonNoInputBox},
		{name: "empty screen, codex", driver: "codex", screen: "", kind: ResumeScreenUnknown, reason: resumeReasonNoInputBox},
		{name: "blank lines only", driver: "claude", screen: "\n\n   \n", kind: ResumeScreenUnknown, reason: resumeReasonNoInputBox},
		{name: "unknown driver", driver: "gemini", screen: enotfound, kind: ResumeScreenUnknown, reason: resumeReasonUnknownDriver},
		{name: "empty driver", driver: "", screen: enotfound, kind: ResumeScreenUnknown, reason: resumeReasonUnknownDriver},
		{name: "driver names are case-sensitive", driver: "Claude", screen: enotfound, kind: ResumeScreenUnknown, reason: resumeReasonUnknownDriver},
		{name: "claude screen read as codex has no input box", driver: "codex", screen: enotfound, kind: ResumeScreenUnknown, reason: resumeReasonNoInputBox},
		{
			name: "codex screen read as claude has no input box", driver: "claude",
			screen: readResumeScreen(t, "codex-network-waiting.txt"), kind: ResumeScreenUnknown, reason: resumeReasonNoInputBox,
		},
		{
			name: "CRLF line ends read like LF", driver: "claude",
			screen: strings.ReplaceAll(enotfound, "\n", "\r\n"), kind: ResumeScreenStopped, line: enotfoundLine,
		},
		{
			name: "claude error right above the input box with no status line is stopped", driver: "claude",
			screen: claudeResumeScreen("❯ say hi\n\n⏺ API Error: Repeated 529 Overloaded errors", ""),
			kind:   ResumeScreenStopped, line: "⏺ API Error: Repeated 529 Overloaded errors",
		},
		{
			// The block starts with the tool call, not with the sentence.
			name: "an API error in a tool result under a tool call is not a stop", driver: "claude",
			screen: claudeResumeScreen("⏺ Bash(go test ./...)\n  ⎿  API Error: No response from API\n\n✻ Worked for 3m 2s · done 2:10 PM", ""),
			kind:   ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "a claude reply quoting a limit after other text is not a stop", driver: "claude",
			screen: claudeResumeScreen("⏺ The banner now reads You've hit your session limit · resets 3:45pm.", ""),
			kind:   ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "a codex reply quoting a usage limit after other text is not a stop", driver: "codex",
			screen: codexResumeScreen("• Fixed the banner for You've hit your usage limit.", ""),
			kind:   ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			// Cross-review cycle 2 finding 2: a stop sentence counts only
			// after one of the driver's error glyphs. Until then this case
			// was stopped.
			name: "a stop sentence with no glyph before it is not a stop", driver: "codex",
			screen: codexResumeScreen("Usage limit reached. You've reached your usage limit.", ""),
			kind:   ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "a stop glyph with no white space after it is not a stop", driver: "claude",
			screen: claudeResumeScreen("⏺API Error: Connection refused", ""),
			kind:   ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "a codex warning glyph before a stop sentence is dropped", driver: "codex",
			screen: codexResumeScreen("⚠ You've hit your usage limit. Try again at 3:05 PM.", ""),
			kind:   ResumeScreenStopped, line: "⚠ You've hit your usage limit. Try again at 3:05 PM.",
		},
		{
			// codex starts ordinary items (events, replies, tool lines) with
			// •, so • is not one of its error glyphs.
			name: "a codex item glyph before a usage limit is not a stop", driver: "codex",
			screen: codexResumeScreen("• You've hit your usage limit. Upgrade to Pro or purchase more credits, or try again at 3:05 PM.", ""),
			kind:   ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "a codex error glyph before a usage limit is stopped", driver: "codex",
			screen: codexResumeScreen("■ You've hit your usage limit. Upgrade to Pro or purchase more credits, or try again at 3:05 PM.", ""),
			kind:   ResumeScreenStopped, line: "■ You've hit your usage limit. Upgrade to Pro or purchase more credits, or try again at 3:05 PM.",
		},
		{
			name: "only one leading glyph is dropped", driver: "claude",
			screen: claudeResumeScreen("⏺ ⎿  API Error: Connection refused", ""),
			kind:   ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "a glyph of the other driver is not dropped", driver: "codex",
			screen: codexResumeScreen("⏺ You've hit your usage limit. Try again at 3:05 PM.", ""),
			kind:   ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "claude connection lost before a response is stopped", driver: "claude",
			screen: claudeResumeScreen("⏺ Connection lost before a response was produced\n\n✻ Cogitated for 40s · done 9:01 AM", ""),
			kind:   ResumeScreenStopped, line: "⏺ Connection lost before a response was produced",
		},
		{
			name: "claude status line that only names the time taken is skipped", driver: "claude",
			screen: claudeResumeScreen("⏺ You've hit your weekly limit · resets Mon 12:00am\n\n✻ Worked for 5s", ""),
			kind:   ResumeScreenStopped, line: "⏺ You've hit your weekly limit · resets Mon 12:00am",
		},
		{
			name: "only one status line is skipped", driver: "claude",
			screen: claudeResumeScreen("⏺ API Error: Connection dropped (ECONNRESET)\n\n✻ Worked for 1s · done 4:33 AM\n\n✻ Brewed for 1s · done 4:34 AM", ""),
			kind:   ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "a status line alone above the input box is unknown", driver: "claude",
			screen: claudeResumeScreen("✻ Worked for 1s · done 4:33 AM", ""), kind: ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "nothing above the input box is unknown", driver: "claude",
			screen: resumeTestRule + "\n❯\n" + resumeTestRule + "\n", kind: ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "a stop sentence on a status line is not a stop", driver: "claude",
			screen: claudeResumeScreen("❯ say hi\n\n✻ API Error: shown on a status line · done 4:33 AM", ""),
			kind:   ResumeScreenUnknown, reason: resumeReasonPromptLast,
		},
		{
			name: "a status line skipped down to a prompt quoting a stop sentence is unknown", driver: "claude",
			screen: claudeResumeScreen("❯ You've hit your weekly limit? Check it.\n\n✻ Worked for 0s · done 7:48 AM", ""),
			kind:   ResumeScreenUnknown, reason: resumeReasonPromptLast,
		},
		{
			name: "a prompt line with U+00A0 after the marker starts a prompt block", driver: "claude",
			screen: claudeResumeScreen("❯ what does API Error: Connection refused mean?", ""),
			kind:   ResumeScreenUnknown, reason: resumeReasonPromptLast,
		},
		{
			name: "a waiting sentence in the user's prompt is not waiting", driver: "claude",
			screen: claudeResumeScreen("⏺ Done.\n\n❯ It said Retrying in 4s earlier. Why?", ""),
			kind:   ResumeScreenUnknown, reason: resumeReasonPromptLast,
		},
		{
			name: "an enter-only sentence in the user's prompt is not enter_only", driver: "claude",
			screen: claudeResumeScreen("⏺ Done.\n\n❯ Should I Press enter to continue?", ""),
			kind:   ResumeScreenUnknown, reason: resumeReasonPromptLast,
		},
		{
			// Were the second ⎿ line a block of its own, it would start with
			// the limit and be stopped; joined, the block starts with the
			// read and the limit is not at its start.
			name: "only the first ⎿ line under a prompt starts a block, the next ones join it", driver: "claude",
			screen: claudeResumeScreen("❯ read @notes.md\n  ⎿  Read notes.md (3 lines)\n  ⎿  You've hit your session limit · resets 3:45pm\n\n✻ Worked for 0s · done 3:01 PM", ""),
			kind:   ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "a prompt line right under a reply starts a prompt block", driver: "claude",
			screen: claudeResumeScreen("⏺ API Error: Connection refused\n❯ say hi", ""),
			kind:   ResumeScreenUnknown, reason: resumeReasonPromptLast,
		},
		{
			name: "waiting sentence in the block before the status line wins over the stop sentence", driver: "claude",
			screen: claudeResumeScreen("⏺ You've hit your Opus limit · resets 3:45pm\n  Continuing shortly · Esc to cancel\n\n✻ Worked for 9s · done 3:44 PM", ""),
			kind:   ResumeScreenWaiting, line: "⏺ You've hit your Opus limit · resets 3:45pm Continuing shortly · Esc to cancel", reason: resumeReasonWaiting,
		},
		{
			name: "lower-case continuing automatically in the history is not waiting", driver: "claude",
			screen: claudeResumeScreen("⏺ You've hit your Sonnet limit · resets 3:45pm\n  Usage limit reached · continuing automatically at 3:45pm · esc to cancel", ""),
			kind:   ResumeScreenStopped, line: "⏺ You've hit your Sonnet limit · resets 3:45pm Usage limit reached · continuing automatically at 3:45pm · esc to cancel",
		},
		{
			name: "claude waiting for API response banner is waiting", driver: "claude",
			screen: claudeResumeScreen("❯ say hi\n\n✢ Waiting for API response · will retry in 40s · check your network", ""),
			kind:   ResumeScreenWaiting, line: "✢ Waiting for API response · will retry in 40s · check your network", reason: resumeReasonWaiting,
		},
		{
			name: "claude press enter above the input box is enter_only", driver: "claude",
			screen: claudeResumeScreen("⏺ You've hit your session limit · resets 3:45pm\n\nYour usage limit has reset\nPress enter to continue", ""),
			kind:   ResumeScreenEnterOnly, line: "Your usage limit has reset Press enter to continue",
		},
		{
			name: "claude input box with no lower rule is no input box", driver: "claude",
			screen: "⏺ API Error: Connection refused\n\n" + resumeTestRule + "\n❯\n  🤖 Haiku 5.5 | ⚡ medium\n",
			kind:   ResumeScreenUnknown, reason: resumeReasonNoInputBox,
		},
		{
			name: "claude marker line not right below the rule is no input box", driver: "claude",
			screen: "⏺ API Error: Connection refused\n\n" + resumeTestRule + "\n\n❯\n" + resumeTestRule + "\n",
			kind:   ResumeScreenUnknown, reason: resumeReasonNoInputBox,
		},
		{
			name: "a rule shorter than 10 runes is no rule", driver: "claude",
			screen: "⏺ API Error: Connection refused\n\n─────────\n❯\n─────────\n",
			kind:   ResumeScreenUnknown, reason: resumeReasonNoInputBox,
		},
		{
			name: "a rule line with other text is no rule", driver: "claude",
			screen: "⏺ API Error: Connection refused\n\n──────── plan ────────\n❯\n" + resumeTestRule + "\n",
			kind:   ResumeScreenUnknown, reason: resumeReasonNoInputBox,
		},
		{
			name: "an unnumbered selection between rules has text", driver: "claude",
			screen: claudeResumeScreen("⏺ API Error: Connection refused", " No, exit"),
			kind:   ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		{
			name: "claude input with text on a second input line has text", driver: "claude",
			screen: "⏺ API Error: Connection refused\n\n" + resumeTestRule + "\n❯\n  second line\n" + resumeTestRule + "\n",
			kind:   ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		{
			name: "claude empty input box with trailing spaces and NBSP is empty", driver: "claude",
			screen: "⏺ API Error: Connection refused\n\n" + resumeTestRule + "\n❯    \n" + resumeTestRule + "\n",
			kind:   ResumeScreenStopped, line: "⏺ API Error: Connection refused",
		},
		{
			name: "codex bare composer marker is empty", driver: "codex",
			screen: codexResumeScreen("■ Usage limit reached. You've reached your usage limit.", ""),
			kind:   ResumeScreenStopped, line: "■ Usage limit reached. You've reached your usage limit.",
		},
		{
			name: "codex side placeholder is empty", driver: "codex",
			screen: codexResumeScreen("■ You've hit your usage limit. Try again at 3:05 PM.", " Ask a follow-up question"),
			kind:   ResumeScreenStopped, line: "■ You've hit your usage limit. Try again at 3:05 PM.",
		},
		{
			name: "codex placeholder with more text is text", driver: "codex",
			screen: codexResumeScreen("■ You've hit your usage limit. Try again at 3:05 PM.", " Ask Codex to do anything now"),
			kind:   ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		{
			name: "codex draft after a blank line under an empty › line has text", driver: "codex",
			screen: codexResumeScreen("■ You've hit your usage limit. Try again at 3:05 PM.", "\n\n  wait, check the proxy first"),
			kind:   ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		{
			name: "codex placeholder on the line under an empty › line is text", driver: "codex",
			screen: codexResumeScreen("■ You've hit your usage limit. Try again at 3:05 PM.", "\n  Ask Codex to do anything"),
			kind:   ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		{
			// No blank line separates the footer from the › line, so no
			// footer is found and its lines count as input.
			name: "codex footer right under the › line counts as input", driver: "codex",
			screen: "■ You've hit your usage limit. Try again at 3:05 PM.\n\n›\n  GPT-6-Astra · xhigh · Context 0% used\n  ? for shortcuts\n",
			kind:   ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		{
			name: "codex empty › line with nothing below it is empty", driver: "codex",
			screen: "■ You've hit your usage limit. Try again at 3:05 PM.\n\n›\n\n",
			kind:   ResumeScreenStopped, line: "■ You've hit your usage limit. Try again at 3:05 PM.",
		},
		{
			name: "codex footer of the status line alone is a footer", driver: "codex",
			screen: "■ You've hit your usage limit. Try again at 3:05 PM.\n\n›\n\n  GPT-6-Astra · xhigh · Context 0% used\n",
			kind:   ResumeScreenStopped, line: "■ You've hit your usage limit. Try again at 3:05 PM.",
		},
		{
			name: "codex footer of the hint line alone is a footer", driver: "codex",
			screen: "■ You've hit your usage limit. Try again at 3:05 PM.\n\n›\n\n  ? for shortcuts                           ⚠ 11 · f2\n",
			kind:   ResumeScreenStopped, line: "■ You've hit your usage limit. Try again at 3:05 PM.",
		},
		{
			name: "codex footer with the context left forms is a footer", driver: "codex",
			screen: "■ You've hit your usage limit. Try again at 3:05 PM.\n\n›\n\n  gpt-6.1-sol · Context 88% left\n  100% context left · ? for shortcuts\n",
			kind:   ResumeScreenStopped, line: "■ You've hit your usage limit. Try again at 3:05 PM.",
		},
		{
			// One line of the last run has no footer shape, so the whole run
			// is input.
			name: "codex draft run together with footer lines is input", driver: "codex",
			screen: "■ You've hit your usage limit. Try again at 3:05 PM.\n\n›\n\n  wait, check the proxy first\n  GPT-6-Astra · xhigh · Context 0% used\n  ? for shortcuts\n",
			kind:   ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		{
			name: "codex draft that mentions the shortcuts hint after other text is input", driver: "codex",
			screen: "■ You've hit your usage limit. Try again at 3:05 PM.\n\n›\n\n  press ? for shortcuts\n",
			kind:   ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		{
			// The status line pattern matches a whole line: a draft that only
			// contains the context figure is not a footer line.
			name: "codex draft that contains the context used figure is input", driver: "codex",
			screen: "■ You've hit your usage limit. Try again at 3:05 PM.\n\n›\n\n  please keep Context 5% used in mind\n",
			kind:   ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		{
			name: "codex draft that ends in the context left figure is input", driver: "codex",
			screen: "■ You've hit your usage limit. Try again at 3:05 PM.\n\n›\n\n  see 3% context left\n",
			kind:   ResumeScreenUnknown, reason: resumeReasonInputHasText,
		},
		{
			name: "codex waiting to retry is waiting", driver: "codex",
			screen: codexResumeScreen("• stream connection failed; waiting to retry", " Ask Codex to do anything"),
			kind:   ResumeScreenWaiting, line: "• stream connection failed; waiting to retry", reason: resumeReasonWaiting,
		},
		{
			name: "codex normal reply is unknown", driver: "codex",
			screen: codexResumeScreen("› say hi\n\n• Hi! What should I work on?", " Ask Codex to do anything"),
			kind:   ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "codex user prompt quoting a stop sentence right above the composer is unknown", driver: "codex",
			screen: codexResumeScreen("• Hi! What should I work on?\n\n› What does \"You've hit your usage limit\" mean?", " Ask Codex to do anything"),
			kind:   ResumeScreenUnknown, reason: resumeReasonPromptLast,
		},
		{
			name: "a claude stop sentence is not a codex stop sentence", driver: "codex",
			screen: codexResumeScreen("■ API Error: Can't reach the API server", " Ask Codex to do anything"),
			kind:   ResumeScreenUnknown, reason: resumeReasonNoStop,
		},
		{
			name: "codex numbered option cursor is a choice dialog", driver: "codex",
			screen: "  Allow command?\n\n› 2. No, and tell Codex what to do differently\n",
			kind:   ResumeScreenUnknown, reason: resumeReasonChoiceDialog,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertResumeScreen(t, ClassifyResumeScreen(tc.driver, tc.screen), tc.kind, tc.line, tc.reason)
		})
	}
}

func TestResumeInputHolds(t *testing.T) {
	cases := []struct {
		name   string
		driver string
		file   string
		screen string // used when file is empty
		typed  string
		want   bool
	}{
		{name: "claude input box holds the sentence wrapped across two lines", driver: "claude", file: "claude-input-typed-wrapped.txt", typed: resumeTestSentence, want: true},
		{name: "claude match spans a wrap in the middle of a word", driver: "claude", file: "claude-input-typed-wrapped.txt", typed: "通信断か利用上限)で作業が止まって", want: true},
		{name: "typed shorter than 12 runes", driver: "claude", file: "claude-input-typed-wrapped.txt", typed: "`ralph", want: true},
		{name: "claude empty input box", driver: "claude", file: "claude-enotfound.txt", typed: resumeTestSentence, want: false},
		{name: "claude input box with other text", driver: "claude", file: "claude-input-has-text.txt", typed: resumeTestSentence, want: false},
		{name: "first 12 runes differ although later text is in the box", driver: "claude", file: "claude-input-typed-wrapped.txt", typed: "Please: 止まったところから続けてください。", want: false},
		{name: "codex composer holds the sentence", driver: "codex", file: "codex-input-typed.txt", typed: resumeTestSentence, want: true},
		{name: "codex placeholder only", driver: "codex", file: "codex-network-waiting.txt", typed: resumeTestSentence, want: false},
		{name: "codex draft below the › line with no footer under it", driver: "codex", file: "codex-input-draft-no-footer.txt", typed: "wait, check the proxy before you go on", want: true},
		{name: "codex choice dialog holds no input", driver: "codex", file: "codex-hooks-review.txt", typed: "1. Review hooks", want: false},
		{name: "claude dialog below an old input box holds no input", driver: "claude", file: "claude-dialog-below-old-input.txt", typed: "1. Yes", want: false},
		{name: "claude screen read as codex", driver: "codex", file: "claude-input-typed-wrapped.txt", typed: resumeTestSentence, want: false},
		{name: "unknown driver", driver: "gemini", file: "claude-input-typed-wrapped.txt", typed: resumeTestSentence, want: false},
		{name: "empty typed", driver: "claude", file: "claude-input-typed-wrapped.txt", typed: "", want: false},
		{name: "all-space typed", driver: "claude", file: "claude-input-typed-wrapped.txt", typed: "   ", want: false},
		{name: "empty screen", driver: "claude", screen: "", typed: resumeTestSentence, want: false},
		{name: "claude input box holds the sentence on one line", driver: "claude", screen: claudeResumeScreen("⏺ API Error: Connection refused", " Continue from where you left off."), typed: "Continue from where you left off.", want: true},
		// AC7 of plan 2026-11-12-send-screen-fallback: the text typed into a
		// working seat's input box (evidence 2026-11-12 section 5).
		{name: "codex 0.166 composer holds the typed text", driver: "codex", file: "codex-working-input-typed.txt", typed: "Reply with the single word: queued", want: true},
		{name: "codex 0.166 placeholder only", driver: "codex", file: "codex-idle-after-reply.txt", typed: "Reply with the single word: queued", want: false},
		{name: "claude 2.1.316 input box holds the typed text", driver: "claude", file: "claude-working-input-typed.txt", typed: "Reply with the single word: queued", want: true},
		// Only the first 12 runes are compared (resumeInputHoldsRunes): a box
		// that shows only the head of the sentence holds it (tech-debt row
		// "Seven mutants of `ralph org resume`", item (e)).
		{name: "claude input box holds only the head of the sentence", driver: "claude", screen: claudeResumeScreen("⏺ API Error: Connection refused", " `ralph org resume` からの"), typed: resumeTestSentence, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			screen := tc.screen
			if tc.file != "" {
				screen = readResumeScreen(t, tc.file)
			}
			if got := ResumeInputHolds(tc.driver, screen, tc.typed); got != tc.want {
				t.Errorf("ResumeInputHolds(%q, %s, %q) = %v, want %v", tc.driver, tc.file, tc.typed, got, tc.want)
			}
		})
	}
}

// TestFindResumeInputBox_Codex0166Footer: on the codex-cli 0.166.0 screens
// of evidence 2026-11-12 (section 6), the footer whose first line goes on
// after `Context <n>% used` is found under the composer, and the composer,
// which holds the placeholder, reads empty (AC7 of plan
// 2026-11-12-send-screen-fallback).
func TestFindResumeInputBox_Codex0166Footer(t *testing.T) {
	cases := []struct {
		file   string
		footer string
	}{
		{"codex-idle-after-reply.txt", "GPT-6-Luna · low · Context 7% used · main · No chan…"},
		{"codex-idle-background-terminal.txt", "GPT-6-Luna · low · Context 7% used · main · No chan…"},
		{"codex-idle-fresh-update-notice.txt", "GPT-6-Luna · low · Context 0% used · main · No chan…"},
		{"codex-working-thinking.txt", "GPT-6-Luna · low · Context 0% used · main · No chan…"},
		{"codex-working-tool.txt", "GPT-6-Luna · low · Context 7% used · main · No chan…"},
		{"codex-working-tool-3s-later.txt", "GPT-6-Luna · low · Context 7% used · main · No chan…"},
		{"codex-queued-steer-while-working.txt", "GPT-6-Luna · low · Context 7% used · main · No chan…"},
		{"codex-submitted-just-now.txt", "GPT-6-Luna · low · Context 7% used · main · No chan…"},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			lines := resumeScreenLines(readResumeScreen(t, tc.file))
			box, reason := findResumeInputBox("codex", lines)
			if reason != "" {
				t.Fatalf("findResumeInputBox: reason %q, want an input box", reason)
			}
			if box.footerStart >= len(lines) {
				t.Fatalf("no footer found under the composer at line %d", box.marker+1)
			}
			if got := strings.TrimSpace(lines[box.footerStart]); got != tc.footer {
				t.Errorf("footer starts with %q, want %q", got, tc.footer)
			}
			if !box.empty("codex", lines) {
				t.Errorf("composer %q reads as holding text, want empty", box.text(lines))
			}
		})
	}
}

// TestResumeCodexFooterLine: the footer line shapes of codex-cli 0.165.0 and
// 0.166.0 match, and a draft that only contains a context figure does not.
func TestResumeCodexFooterLine(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{"GPT-6-Astra · xhigh · Context 0% used", true},
		{"GPT-6-Luna · low · Context 7% used · main · No chan…", true},
		{"GPT-6-Luna · low · Context 7% used · main", true},
		{"gpt-6.1-sol · Context 88% left · main", true},
		{"? for shortcuts                           ⚠ 11 · f2", true},
		{"100% context left · ? for shortcuts", true},
		{"Context 7% used · main · No chan…", false},
		{"please keep Context 5% used in mind", false},
		{"GPT-6-Luna · low · Context 7% used ·", false},
		{"tab to queue message", false},
	}
	for _, tc := range cases {
		if got := resumeCodexFooterLine.MatchString(tc.line); got != tc.want {
			t.Errorf("resumeCodexFooterLine.MatchString(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}

func TestResumeScreenSentences_EveryEntryHasASource(t *testing.T) {
	sources := []string{"seen: ", "docs: https://", "binary strings: "}
	check := func(where string, list []resumeSentence) {
		for _, s := range list {
			if strings.TrimSpace(s.Text) == "" {
				t.Errorf("%s: an entry has no text (source %q)", where, s.Source)
			}
			ok := false
			for _, p := range sources {
				if strings.HasPrefix(s.Source, p) && len(s.Source) > len(p) {
					ok = true
				}
			}
			if !ok {
				t.Errorf("%s %q: source %q does not start with one of %q", where, s.Text, s.Source, sources)
			}
		}
	}
	for _, driver := range []string{"claude", "codex"} {
		table, ok := resumeScreenSentences[driver]
		if !ok {
			t.Fatalf("no sentence table for driver %q", driver)
		}
		if len(table.Stopped) == 0 || len(table.Waiting) == 0 {
			t.Errorf("driver %q: want stopped and waiting sentences, got %d and %d", driver, len(table.Stopped), len(table.Waiting))
		}
		check(driver+" stopped", table.Stopped)
		check(driver+" enter_only", table.EnterOnly)
		check(driver+" waiting", table.Waiting)
	}
	if len(resumeScreenSentences) != 2 {
		t.Errorf("sentence tables for %d drivers, want claude and codex only", len(resumeScreenSentences))
	}
	check("codex placeholder", codexComposerPlaceholders)
}
