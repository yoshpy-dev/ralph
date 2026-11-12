package org

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// sendScreenWant is what ClassifySendScreen should say about one fixture of
// testdata/resume/. line is checked when it is not empty.
type sendScreenWant struct {
	kind   SendScreenKind
	reason string
	line   string
}

// sendScreenFixtureWants holds an expectation for every file of
// testdata/resume/ (TestClassifySendScreen_EveryFixture fails on a file
// with none). The scenes of the 2026-11-12 fixtures are in the table at the
// end of docs/evidence/org-send-screen-probe-2026-11-12.md; the others are
// the resume fixtures of TestClassifyResumeScreen_Fixtures.
var sendScreenFixtureWants = map[string]sendScreenWant{
	// claude 2.1.316, evidence 2026-11-12.
	"claude-idle-after-reply.txt":       {SendScreenIdle, sendScreenReasonTurnEnded, "✻ Cooked for 3s · done 6:09 PM"},
	"claude-idle-fresh-placeholder.txt": {SendScreenIdle, sendScreenReasonFreshSession, "▐▛███▛█   Claude Code v2.1.316"},
	"claude-working-thinking-start.txt": {SendScreenBusy, sendScreenReasonBusyLine, "✳ Elucidating…"},
	"claude-working-thinking.txt":       {SendScreenBusy, sendScreenReasonBusyLine, "✳ Pollinating… (3s · thinking)"},
	// The busy line wrapped onto a second line; its start still counts.
	"claude-working-tool-hooks.txt":    {SendScreenBusy, sendScreenReasonBusyLine, "✢ Deliberating… (running PreToolUse hooks… 5/10 · 2s"},
	"claude-working-tool.txt":          {SendScreenBusy, sendScreenReasonBusyLine, "✶ Deliberating… (6s · ↓ 100 tokens)"},
	"claude-working-tool-3s-later.txt": {SendScreenBusy, sendScreenReasonBusyLine, "✶ Deliberating… (9s · ↓ 100 tokens)"},
	// The text typed into the box decides first: typing more would join it.
	"claude-working-input-typed.txt":  {SendScreenUnknown, sendScreenReasonInputHasText, "Reply with the single word: queued"},
	"claude-queued-while-working.txt": {SendScreenBusy, sendScreenReasonQueuedInput, "Press up to edit queued messages"},
	"claude-submitted-just-now.txt":   {SendScreenBusy, sendScreenReasonBusyLine, "✽ Bunning…"},
	"claude-submitted-1s.txt":         {SendScreenBusy, sendScreenReasonBusyLine, "· Bunning…"},
	// No numbers on the options; the hint line marks the dialog.
	"claude-folder-trust-unnumbered.txt": {SendScreenChoice, sendScreenReasonChoiceDialog, "Enter to confirm · Esc to cancel"},

	// codex-cli 0.166.0, evidence 2026-11-12.
	"codex-idle-fresh-update-notice.txt": {SendScreenIdle, sendScreenReasonFreshSession, ">_ OpenAI Codex (v0.166.0)"},
	"codex-idle-after-reply.txt":         {SendScreenIdle, sendScreenReasonTurnEnded, "Worked for 4s • 18:17"},
	// The turn ended (`Worked for 11s`, herdr done, evidence section 5); the
	// `sleep` codex left in a background terminal does not hold the turn,
	// and the line under the turn-end line is a trailer. Idle.
	"codex-idle-background-terminal.txt":   {SendScreenIdle, sendScreenReasonTurnEnded, "Worked for 11s • 18:19"},
	"codex-working-thinking.txt":           {SendScreenBusy, sendScreenReasonBusyLine, "• Working (2s • esc to interrupt)"},
	"codex-working-tool.txt":               {SendScreenBusy, sendScreenReasonBusyLine, "• Working (6s • esc to interrupt) · 1 background term…"},
	"codex-working-tool-3s-later.txt":      {SendScreenBusy, sendScreenReasonBusyLine, "• Working (8s • esc to interrupt) · 1 background term…"},
	"codex-working-input-typed.txt":        {SendScreenUnknown, sendScreenReasonInputHasText, ""},
	"codex-queued-steer-while-working.txt": {SendScreenBusy, sendScreenReasonBusyLine, "• Working (10s • esc to interrupt) · 1 background ter…"},
	"codex-submitted-just-now.txt":         {SendScreenBusy, sendScreenReasonBusyLine, "• Working (0s • esc to interrupt)"},
	"codex-update-available.txt":           {SendScreenChoice, sendScreenReasonChoiceDialog, "› 1. Update now (runs `pnpm add -g @openai/codex`)"},
	"codex-folder-trust.txt":               {SendScreenChoice, sendScreenReasonChoiceDialog, "› 1. Trust and continue"},

	// A message of several lines typed and not submitted, then submitted,
	// evidence 2026-11-12 section 10 (claude 2.1.316, codex-cli 0.166.0).
	// The text in the box decides first, whatever the box shows of it.
	"claude-input-typed-multiline-short.txt": {
		SendScreenUnknown, sendScreenReasonInputHasText,
		"TYPE: QUESTION Reply with the single word: alpha. This is a probe, so do not run any tool and do not read or edit any file.",
	},
	"claude-input-typed-multiline-medium.txt":         {SendScreenUnknown, sendScreenReasonInputHasText, ""},
	"claude-input-typed-multiline-long.txt":           {SendScreenUnknown, sendScreenReasonInputHasText, "[Pasted text #1 +17 lines][Pasted text #2 +11 lines]"},
	"claude-input-typed-multiline-head-lost.txt":      {SendScreenUnknown, sendScreenReasonInputHasText, ""},
	"claude-input-typed-multiline-scrolled-paste.txt": {SendScreenUnknown, sendScreenReasonInputHasText, ""},
	"codex-input-typed-multiline-short.txt":           {SendScreenUnknown, sendScreenReasonInputHasText, ""},
	"codex-input-typed-multiline-medium.txt":          {SendScreenUnknown, sendScreenReasonInputHasText, ""},
	"codex-input-typed-multiline-long.txt":            {SendScreenUnknown, sendScreenReasonInputHasText, ""},
	// The long message's prompt scrolled off the screen: no prompt line is
	// above the input box, so the whole screen above it is read.
	"claude-submitted-multiline-long.txt":      {SendScreenBusy, sendScreenReasonBusyLine, "✽ Contemplating…"},
	"claude-submitted-multiline-long-1.5s.txt": {SendScreenBusy, sendScreenReasonBusyLine, "✢ Contemplating… (1s · thinking)"},
	"claude-idle-after-multiline-reply.txt":    {SendScreenIdle, sendScreenReasonTurnEnded, "✻ Worked for 3s · done 7:46 PM"},
	"codex-submitted-multiline-long.txt":       {SendScreenBusy, sendScreenReasonBusyLine, "• Working (0s • esc to interrupt)"},
	"codex-submitted-multiline-long-1.5s.txt":  {SendScreenBusy, sendScreenReasonBusyLine, "• Working (1s • esc to interrupt)"},
	"codex-idle-after-multiline-reply.txt":     {SendScreenIdle, sendScreenReasonTurnEnded, "Worked for 2s • 19:49"},
	// Synthetic: the medium message's boxes showing the end of
	// sendNumberedStepsMsg instead, whose first line is one of its numbered
	// steps. Before anything is typed, such a marker line is a choice
	// dialog's cursor line to the rules; nothing is typed on it either way.
	"claude-input-typed-numbered-steps.txt": {SendScreenChoice, sendScreenReasonChoiceDialog, "❯ 7. Run the checks the plan names."},
	"codex-input-typed-numbered-steps.txt":  {SendScreenChoice, sendScreenReasonChoiceDialog, "› 2. Read the files the plan lists for the slice.    ↑"},

	// The resume fixtures (evidence 2026-11-07 and synthetic).
	"codex-hooks-review.txt":              {SendScreenChoice, sendScreenReasonChoiceDialog, "› 1. Review hooks"},
	"claude-folder-trust.txt":             {SendScreenChoice, sendScreenReasonChoiceDialog, "❯ 1. Yes, I trust this folder"},
	"claude-dialog-below-old-input.txt":   {SendScreenChoice, sendScreenReasonChoiceDialog, "❯ 1. Yes"},
	"claude-retrying.txt":                 {SendScreenBusy, sendScreenReasonWaiting, "✻ Can't reach the… · Retrying in 4s · attempt 13/300"},
	"claude-continuing-automatically.txt": {SendScreenBusy, sendScreenReasonWaiting, "Usage limit reached · limit resets 3:45pm Continuing automatically at 3:45pm · Esc to cancel"},
	"codex-network-waiting.txt": {
		SendScreenBusy, sendScreenReasonWaiting,
		"• Reconnecting... waiting for network (2m 56s • esc t… └ Connection failed: error sending request └ Tip: Use /mcp to list configured MCP tools.",
	},
	"claude-press-enter.txt":                  {SendScreenUnknown, sendScreenReasonPressEnter, "Your usage limit has reset Press enter to continue"},
	"claude-input-has-text.txt":               {SendScreenUnknown, sendScreenReasonInputHasText, "wait, check the proxy before you go on"},
	"claude-input-draft-after-empty-line.txt": {SendScreenUnknown, sendScreenReasonInputHasText, "wait, check the proxy before you go on"},
	"claude-input-typed-wrapped.txt":          {SendScreenUnknown, sendScreenReasonInputHasText, ""},
	"codex-input-typed.txt":                   {SendScreenUnknown, sendScreenReasonInputHasText, ""},
	"codex-input-draft-after-empty-line.txt":  {SendScreenUnknown, sendScreenReasonInputHasText, "wait, check the proxy before you go on"},
	"codex-input-draft-no-footer.txt":         {SendScreenUnknown, sendScreenReasonInputHasText, "wait, check the proxy before you go on"},
	// A turn that ended on an API error or a limit, with an empty input box
	// under the turn-end line, waits for input: herdr reports it idle too,
	// and a message sent there starts a new turn. Idle.
	"claude-enotfound.txt":                 {SendScreenIdle, sendScreenReasonTurnEnded, "✻ Brewed for 1s · done 4:34 AM"},
	"claude-session-limit.txt":             {SendScreenIdle, sendScreenReasonTurnEnded, "✻ Worked for 12s · done 1:02 PM"},
	"claude-weekly-limit-under-prompt.txt": {SendScreenIdle, sendScreenReasonTurnEnded, "✻ Churned for 0s · done 7:48 AM"},
	// codex printed no `Worked for` line under its limit error in this
	// synthetic screen, so nothing shows the turn ended. Unknown.
	"codex-usage-limit.txt":                     {SendScreenUnknown, sendScreenReasonNoTurnEnd, "■ You've hit your usage limit. Upgrade to Pro or purchase more credits, or try"},
	"claude-old-error-in-history.txt":           {SendScreenIdle, sendScreenReasonTurnEnded, "✻ Worked for 2s · done 4:41 AM"},
	"claude-prompt-quotes-error-then-reply.txt": {SendScreenIdle, sendScreenReasonTurnEnded, "✻ Worked for 6s · done 9:12 AM"},
	"claude-reply-error-paragraph.txt":          {SendScreenIdle, sendScreenReasonTurnEnded, "✻ Worked for 12s · done 2:15 PM"},
	"claude-reply-quotes-error.txt":             {SendScreenIdle, sendScreenReasonTurnEnded, "✻ Worked for 41s · done 2:15 PM"},
	"claude-tool-output-error-paragraph.txt":    {SendScreenIdle, sendScreenReasonTurnEnded, "✻ Worked for 9s · done 2:17 PM"},
	"claude-prompt-quotes-limit-last.txt":       {SendScreenUnknown, sendScreenReasonPromptLast, ""},
	// An interrupted turn (`⎿  Interrupted · What should Claude do
	// instead?`) has no turn-end line; the rules do not take the
	// interruption as one. Unknown.
	"claude-prompt-quotes-error-interrupted.txt": {SendScreenUnknown, sendScreenReasonNoTurnEnd, "⎿  Interrupted · What should Claude do instead?"},
}

// sendScreenNeverIdleName and sendScreenIdleName tie a fixture's name to its
// kind, so that a new fixture named after its scene is checked even before
// its exact expectation is right.
var (
	sendScreenNeverIdleName = regexp.MustCompile(`-working-|-queued-|-submitted-|retrying|network-waiting|continuing-automatically|-input-|folder-trust|hooks-review|update-available|dialog-`)
	sendScreenIdleName      = regexp.MustCompile(`-idle-`)
)

func TestClassifySendScreen_EveryFixture(t *testing.T) {
	dir := filepath.Join("testdata", "resume")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	seen := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		seen[name] = true
		want, ok := sendScreenFixtureWants[name]
		if !ok {
			t.Errorf("fixture %s/%s has no expectation in sendScreenFixtureWants", dir, name)
			continue
		}
		t.Run(name, func(t *testing.T) {
			driver := strings.SplitN(name, "-", 2)[0]
			got := ClassifySendScreen(driver, readResumeScreen(t, name))
			if got.Kind != want.kind || got.Reason != want.reason || (want.line != "" && got.Line != want.line) {
				t.Errorf("ClassifySendScreen(%q) = {Kind:%q Reason:%q Line:%q}, want {Kind:%q Reason:%q Line:%q}",
					driver, got.Kind, got.Reason, got.Line, want.kind, want.reason, want.line)
			}
			if sendScreenNeverIdleName.MatchString(name) && got.Kind == SendScreenIdle {
				t.Errorf("a %s screen must never be idle", name)
			}
			if sendScreenIdleName.MatchString(name) && got.Kind != SendScreenIdle {
				t.Errorf("an -idle- screen must be idle, got %q (%s)", got.Kind, got.Reason)
			}
		})
	}
	for name := range sendScreenFixtureWants {
		if !seen[name] {
			t.Errorf("sendScreenFixtureWants names %s, which is not in %s", name, dir)
		}
	}
}

// TestSendScreenBoxCheck_ChoiceDialogWithNoMarkerLine: a choice dialog
// reason with no line starting with the driver's marker (which
// findResumeInputBox never returns today) does not index lines[-1]; the
// screen is unknown, with no input box, so nothing is typed. The same
// screen through ClassifySendScreen is unknown too.
func TestSendScreenBoxCheck_ChoiceDialogWithNoMarkerLine(t *testing.T) {
	screen := "⏺ Hi.\n\n✻ Cooked for 3s · done 6:09 PM\n"
	lines := resumeScreenLines(screen)
	for _, driver := range []string{"claude", "codex"} {
		got, decided := sendScreenBoxCheck(driver, lines, resumeInputBox{}, resumeReasonChoiceDialog)
		if !decided || got.Kind != SendScreenUnknown || got.Reason != sendScreenReasonNoInputBox || got.Line != "" {
			t.Errorf("%s: sendScreenBoxCheck = %+v, %v; want unknown (%s), decided", driver, got, decided, sendScreenReasonNoInputBox)
		}
		if got := ClassifySendScreen(driver, screen); got.Kind != SendScreenUnknown || got.Reason != sendScreenReasonNoInputBox {
			t.Errorf("%s: ClassifySendScreen = %+v; want unknown (%s)", driver, got, sendScreenReasonNoInputBox)
		}
	}
}

func TestClassifySendScreen_Edges(t *testing.T) {
	cases := []struct {
		name   string
		driver string
		screen string
		want   sendScreenWant
	}{
		{name: "unknown driver", driver: "gemini", screen: claudeResumeScreen("✻ Cooked for 3s · done 6:09 PM", ""), want: sendScreenWant{SendScreenUnknown, sendScreenReasonUnknownDriver, ""}},
		{name: "empty screen", driver: "claude", screen: "", want: sendScreenWant{SendScreenUnknown, sendScreenReasonNoInputBox, ""}},
		{
			name: "claude busy ✻ word… under a prompt is busy", driver: "claude",
			screen: claudeResumeScreen("❯ say hi\n\n✻ Cooking… (2s · thinking)", ""),
			want:   sendScreenWant{SendScreenBusy, sendScreenReasonBusyLine, "✻ Cooking… (2s · thinking)"},
		},
		{
			name: "claude ✻ word for 3s · done under a reply is idle", driver: "claude",
			screen: claudeResumeScreen("❯ say hi\n\n⏺ Hi.\n\n✻ Cooked for 3s · done 6:09 PM", ""),
			want:   sendScreenWant{SendScreenIdle, sendScreenReasonTurnEnded, "✻ Cooked for 3s · done 6:09 PM"},
		},
		{
			name: "claude turn-end line with the time taken only is not a turn end", driver: "claude",
			screen: claudeResumeScreen("❯ say hi\n\n⏺ Hi.\n\n✻ Cooked for 3s", ""),
			want:   sendScreenWant{SendScreenUnknown, sendScreenReasonNoTurnEnd, "✻ Cooked for 3s"},
		},
		{
			name: "claude busy line wrapped onto a second line is busy", driver: "claude",
			screen: claudeResumeScreen("❯ go\n\n✢ Deliberating… (running PreToolUse hooks… 5/10 · 2s\n · ↓ 14 tokens)\n  ⎿  Tip: Use ctrl+v to paste images", ""),
			want:   sendScreenWant{SendScreenBusy, sendScreenReasonBusyLine, "✢ Deliberating… (running PreToolUse hooks… 5/10 · 2s"},
		},
		{
			name: "claude busy line under an old turn-end line is busy", driver: "claude",
			screen: claudeResumeScreen("⏺ Hi.\n\n✻ Cooked for 3s · done 6:09 PM\n\n✽ Bunning…", ""),
			want:   sendScreenWant{SendScreenBusy, sendScreenReasonBusyLine, "✽ Bunning…"},
		},
		{
			name: "claude prompt last with no busy line yet is unknown", driver: "claude",
			screen: claudeResumeScreen("⏺ Hi.\n\n✻ Cooked for 3s · done 6:09 PM\n\n❯ second", ""),
			want:   sendScreenWant{SendScreenUnknown, sendScreenReasonPromptLast, ""},
		},
		{
			name: "codex prompt last with no busy line yet is unknown", driver: "codex",
			screen: codexResumeScreen("• Hi!\n\n  Worked for 4s • 18:17\n\n\n› say hi", " Ask Codex to do anything"),
			want:   sendScreenWant{SendScreenUnknown, sendScreenReasonPromptLast, ""},
		},
		{
			name: "claude reply after the turn-end line is not a turn end", driver: "claude",
			screen: claudeResumeScreen("❯ go\n\n✻ Cooked for 3s · done 6:09 PM\n\n⏺ More text", ""),
			want:   sendScreenWant{SendScreenUnknown, sendScreenReasonNoTurnEnd, "⏺ More text"},
		},
		{
			name: "claude tip block under the turn-end line is idle", driver: "claude",
			screen: claudeResumeScreen("❯ go\n\n✻ Cooked for 3s · done 6:09 PM\n\n  ⎿  Tip: Use ctrl+v to paste images", ""),
			want:   sendScreenWant{SendScreenIdle, sendScreenReasonTurnEnded, "✻ Cooked for 3s · done 6:09 PM"},
		},
		{
			name: "codex turn end inside a rule line is idle", driver: "codex",
			screen: codexResumeScreen("› go\n\n• Done.\n\n─ Worked for 1m 05s ──────────────", " Ask Codex to do anything"),
			want:   sendScreenWant{SendScreenIdle, sendScreenReasonTurnEnded, "─ Worked for 1m 05s ──────────────"},
		},
		{
			name: "codex working header other than Working is busy", driver: "codex",
			screen: codexResumeScreen("› go\n\n• Exploring the parser (12s • esc to…", " Ask Codex to do anything"),
			want:   sendScreenWant{SendScreenBusy, sendScreenReasonBusyLine, "• Exploring the parser (12s • esc to…"},
		},
		{
			name: "codex working line cut right after the paren is busy", driver: "codex",
			screen: codexResumeScreen("› go\n\n• Working (…", " Ask Codex to do anything"),
			want:   sendScreenWant{SendScreenBusy, sendScreenReasonBusyLine, "• Working (…"},
		},
		{
			name: "codex queued follow-up inputs is busy", driver: "codex",
			screen: codexResumeScreen("› go\n\n  Worked for 4s • 18:17\n\n  Queued follow-up inputs\n  ↳ check the proxy", " Ask Codex to do anything"),
			want:   sendScreenWant{SendScreenBusy, sendScreenReasonQueuedInput, "Queued follow-up inputs"},
		},
		{
			name: "claude esc to interrupt in the footer is busy", driver: "claude",
			screen: "❯ go\n\n✻ Cooked for 3s · done 6:09 PM\n\n" + resumeTestRule + "\n❯\n" + resumeTestRule + "\n  ⏵⏵ bypass permissions on · esc to interrupt\n",
			want:   sendScreenWant{SendScreenBusy, sendScreenReasonBusyLine, "⏵⏵ bypass permissions on · esc to interrupt"},
		},
		{
			name: "claude Try placeholder with a draft under it has text", driver: "claude",
			screen: "Claude Code v2.1.316\n\n" + resumeTestRule + "\n❯ Try \"how do I log an error?\"\n  draft\n" + resumeTestRule + "\n",
			want:   sendScreenWant{SendScreenUnknown, sendScreenReasonInputHasText, ""},
		},
		{
			name: "claude input box alone with no banner and no turn-end line is unknown", driver: "claude",
			screen: resumeTestRule + "\n❯\n" + resumeTestRule + "\n",
			want:   sendScreenWant{SendScreenUnknown, sendScreenReasonNoTurnEnd, ""},
		},
		{
			name: "claude banner above a prompt is not a fresh session", driver: "claude",
			screen: claudeResumeScreen(" ▐▛███▛█   Claude Code v2.1.316\n\n❯ say hi", ""),
			want:   sendScreenWant{SendScreenUnknown, sendScreenReasonPromptLast, ""},
		},
		{
			name: "claude banner above a prompt and a reply with no turn-end line is unknown", driver: "claude",
			screen: claudeResumeScreen(" ▐▛███▛█   Claude Code v2.1.316\n\n❯ say hi\n\n⏺ Hi.", ""),
			want:   sendScreenWant{SendScreenUnknown, sendScreenReasonNoTurnEnd, "⏺ Hi."},
		},
		{
			name: "dialog hint with no marker line is a choice", driver: "codex",
			screen: "  Allow this?\n\n  Yes\n  No\n\n  enter confirm · esc skip\n",
			want:   sendScreenWant{SendScreenChoice, sendScreenReasonChoiceDialog, "enter confirm · esc skip"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifySendScreen(tc.driver, tc.screen)
			w := tc.want
			if got.Kind != w.kind || got.Reason != w.reason || (w.line != "" && got.Line != w.line) {
				t.Errorf("ClassifySendScreen = {Kind:%q Reason:%q Line:%q}, want {Kind:%q Reason:%q Line:%q}",
					got.Kind, got.Reason, got.Line, w.kind, w.reason, w.line)
			}
		})
	}
}

func TestSendScreenSubmitted(t *testing.T) {
	// The text typed in evidence 2026-11-12 sections 5 and 6.
	const queued = "Reply with the single word: queued"
	const second = "Reply with the single word: second"
	cases := []struct {
		name   string
		driver string
		file   string
		screen string // used when file is empty
		typed  string
		want   bool
	}{
		{name: "claude typed text still in the box", driver: "claude", file: "claude-working-input-typed.txt", typed: queued, want: false},
		{name: "codex typed text still in the box", driver: "codex", file: "codex-working-input-typed.txt", typed: queued, want: false},
		{name: "claude box empty after Enter with a busy line", driver: "claude", file: "claude-submitted-just-now.txt", typed: second, want: true},
		{name: "codex placeholder after Enter with a busy line", driver: "codex", file: "codex-submitted-just-now.txt", typed: second, want: true},
		{name: "claude message queued for the running turn", driver: "claude", file: "claude-queued-while-working.txt", typed: queued, want: true},
		{name: "codex message steered into the running turn", driver: "codex", file: "codex-queued-steer-while-working.txt", typed: queued, want: true},
		{name: "codex box empty after the turn already ended", driver: "codex", file: "codex-idle-background-terminal.txt", typed: queued, want: true},
		{name: "no input box", driver: "claude", file: "claude-folder-trust-unnumbered.txt", typed: queued, want: false},
		{name: "choice dialog", driver: "codex", file: "codex-update-available.txt", typed: queued, want: false},
		{name: "empty screen", driver: "claude", screen: "", typed: queued, want: false},
		{name: "unknown driver", driver: "gemini", file: "claude-submitted-just-now.txt", typed: second, want: false},
		{name: "all-space typed", driver: "claude", file: "claude-submitted-just-now.txt", typed: "  ", want: false},
		{
			// The queued-input text counts by itself, with no busy line on
			// the screen.
			name: "claude queued-input text with no busy line visible", driver: "claude",
			screen: claudeResumeScreen("❯ "+queued+"\n  ctrl+enter to send now", " Press up to edit queued messages"), typed: queued, want: true,
		},
		{
			name: "other text in the box and a busy line", driver: "claude",
			screen: claudeResumeScreen("❯ "+second+"\n\n✽ Bunning…", " a draft"), typed: second, want: true,
		},
		{
			name: "other text in the box and nothing busy", driver: "claude",
			screen: claudeResumeScreen("❯ "+second+"\n\n⏺ second\n\n✻ Cooked for 1s · done 6:20 PM", " a draft"), typed: second, want: false,
		},

		// A message of several lines (evidence 2026-11-12 section 10).
		{name: "claude end of the message still in the box", driver: "claude", file: "claude-input-typed-multiline-medium.txt", typed: sendProbeMediumMsg, want: false},
		{name: "claude paste placeholders still in the box", driver: "claude", file: "claude-input-typed-multiline-long.txt", typed: sendProbeLongMsg, want: false},
		{name: "codex end of the message still in the box", driver: "codex", file: "codex-input-typed-multiline-long.txt", typed: sendProbeLongMsg, want: false},
		{name: "codex short message still in the box", driver: "codex", file: "codex-input-typed-multiline-short.txt", typed: sendProbeShortMsg, want: false},
		{name: "claude long message submitted, prompt scrolled off", driver: "claude", file: "claude-submitted-multiline-long.txt", typed: sendProbeLongMsg, want: true},
		{name: "claude long message submitted, 1.5 s later", driver: "claude", file: "claude-submitted-multiline-long-1.5s.txt", typed: sendProbeLongMsg, want: true},
		{name: "codex long message submitted, prompt scrolled off", driver: "codex", file: "codex-submitted-multiline-long.txt", typed: sendProbeLongMsg, want: true},
		{name: "codex long message submitted, 1.5 s later", driver: "codex", file: "codex-submitted-multiline-long-1.5s.txt", typed: sendProbeLongMsg, want: true},
		{name: "claude long message answered already", driver: "claude", file: "claude-idle-after-multiline-reply.txt", typed: sendProbeLongMsg, want: true},
		{
			// The end of the message, still in the box, is the message even
			// with a busy line on the screen.
			name: "claude end of the message in the box and a busy line", driver: "claude",
			screen: claudeResumeScreen("❯ go\n\n✽ Bunning…", " the lazy dog again.\n  Reply with the single word: bravo."), typed: sendProbeMediumMsg, want: false,
		},
		{
			// A paste placeholder in the box is never a submitted box, even
			// one that does not fit the message, with a busy line on the
			// screen.
			name: "claude paste placeholder in the box and a busy line", driver: "claude",
			screen: claudeResumeScreen("❯ go\n\n✽ Bunning…", " [Pasted text #1 +40 lines]"), typed: sendProbeShortMsg, want: false,
		},

		// The box's first line is a numbered step of the message: a choice
		// dialog to findResumeInputBox, so never submitted, a busy line on
		// the screen or not.
		{name: "claude numbered step of the message still in the box", driver: "claude", file: "claude-input-typed-numbered-steps.txt", typed: sendNumberedStepsMsg, want: false},
		{name: "codex numbered step of the message still in the box", driver: "codex", file: "codex-input-typed-numbered-steps.txt", typed: sendNumberedStepsMsg, want: false},
		{
			name: "claude numbered step of the message in the box and a busy line", driver: "claude",
			screen: claudeResumeScreen("❯ go\n\n✽ Bunning…", " 7. Run the checks the plan names.\n  8. Fix what fails and run them again."), typed: sendNumberedStepsMsg, want: false,
		},
		{
			name: "codex numbered step of the message in the box and a busy line", driver: "codex",
			screen: codexResumeScreen("› go\n\n• Working (3s • esc to interrupt)", " 7. Run the checks the plan names.\n  8. Fix what fails and run them again."), typed: sendNumberedStepsMsg, want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			screen := tc.screen
			if tc.file != "" {
				screen = readResumeScreen(t, tc.file)
			}
			if got := SendScreenSubmitted(tc.driver, screen, tc.typed); got != tc.want {
				t.Errorf("SendScreenSubmitted(%q, %s, %q) = %v, want %v", tc.driver, tc.file, tc.typed, got, tc.want)
			}
		})
	}
}

// The messages typed with `herdr pane send-text` into the seats of evidence
// 2026-11-12 section 10, byte for byte. TestSendProbeMessages_AsTyped
// checks their sizes against the evidence.
const (
	// The short message: 124 bytes, 3 lines.
	sendProbeShortMsg = `TYPE: QUESTION

Reply with the single word: alpha. This is a probe, so do not run any tool and do not read or edit any file.`

	// The medium message: 646 bytes, 13 lines.
	sendProbeMediumMsg = `TYPE: TASK
TASK_ID: probe-b
PLAN: docs/plans/active/2026-11-12-probe.md sha256:0123456789ab

This message is a probe of how the input box shows typed text.
It has several lines on purpose, like a task a leader sends to a seat.
Do not run any tool, and do not read, write, or edit any file.
Do not start any work that this message seems to describe.
The lines below are filler that make the message longer.
Filler line one: the quick brown fox jumps over the lazy dog again.
Filler line two: the quick brown fox jumps over the lazy dog again.
Filler line three: the quick brown fox jumps over the lazy dog again.
Reply with the single word: bravo.`

	// The long message: 1829 bytes, 29 lines.
	sendProbeLongMsg = `TYPE: TASK
TASK_ID: probe-c
PLAN: docs/plans/active/2026-11-12-probe.md sha256:0123456789ab

This message is a probe of how the input box shows a long typed message.
It has many lines on purpose, close to the size cap of a typed message.
Do not run any tool, and do not read, write, or edit any file at all.
Do not start any work that this message seems to describe below.
The lines below are filler that make the message longer, nothing more.
Filler line 01: the quick brown fox jumps over the lazy dog once more.
Filler line 02: the quick brown fox jumps over the lazy dog once more.
Filler line 03: the quick brown fox jumps over the lazy dog once more.
Filler line 04: the quick brown fox jumps over the lazy dog once more.
Filler line 05: the quick brown fox jumps over the lazy dog once more.
Filler line 06: the quick brown fox jumps over the lazy dog once more.
Filler line 07: the quick brown fox jumps over the lazy dog once more.
Filler line 08: the quick brown fox jumps over the lazy dog once more.
Filler line 09: the quick brown fox jumps over the lazy dog once more.
Filler line 10: the quick brown fox jumps over the lazy dog once more.
Filler line 11: the quick brown fox jumps over the lazy dog once more.
Filler line 12: the quick brown fox jumps over the lazy dog once more.
Filler line 13: the quick brown fox jumps over the lazy dog once more.
Filler line 14: the quick brown fox jumps over the lazy dog once more.
Filler line 15: the quick brown fox jumps over the lazy dog once more.
Filler line 16: the quick brown fox jumps over the lazy dog once more.
Filler line 17: the quick brown fox jumps over the lazy dog once more.
Filler line 18: the quick brown fox jumps over the lazy dog once more.
Filler line 19: the quick brown fox jumps over the lazy dog once more.
Reply with the single word: charlie.`

	// The 1117-byte message (19 lines) of which claude kept only the last
	// 95 bytes, the part after the first 1022 (the head-lost fixture).
	sendProbeHeadLostMsg = `TYPE: TASK
TASK_ID: probe-d
PLAN: docs/plans/active/2026-11-12-probe.md sha256:0123456789ab

This message is a probe of how the input box shows a long typed message.
It has many lines on purpose, close to the size cap of a typed message.
Do not run any tool, and do not read, write, or edit any file at all.
Do not start any work that this message seems to describe below.
The lines below are filler that make the message longer, nothing more.
Filler line 01: the quick brown fox jumps over the lazy dog once more.
Filler line 02: the quick brown fox jumps over the lazy dog once more.
Filler line 03: the quick brown fox jumps over the lazy dog once more.
Filler line 04: the quick brown fox jumps over the lazy dog once more.
Filler line 05: the quick brown fox jumps over the lazy dog once more.
Filler line 06: the quick brown fox jumps over the lazy dog once more.
Filler line 07: the quick brown fox jumps over the lazy dog once more.
Filler line 08: the quick brown fox jumps over the lazy dog once more.
Filler line 09: the quick brown fox jumps over the lazy dog once more.
Reply with the single word: delta.`

	// The 1329-byte message (22 lines) claude got whole: a paste
	// placeholder for its first 1022 bytes, scrolled out of the box, and the
	// rest as text (the scrolled-paste fixture).
	sendProbeScrolledPasteMsg = `TYPE: TASK
TASK_ID: probe-e
PLAN: docs/plans/active/2026-11-12-probe.md sha256:0123456789ab

This message is a probe of how the input box shows a long typed message.
It has many lines on purpose, close to the size cap of a typed message.
Do not run any tool, and do not read, write, or edit any file at all.
Do not start any work that this message seems to describe below.
The lines below are filler that make the message longer, nothing more.
Filler line 01: the quick brown fox jumps over the lazy dog once more.
Filler line 02: the quick brown fox jumps over the lazy dog once more.
Filler line 03: the quick brown fox jumps over the lazy dog once more.
Filler line 04: the quick brown fox jumps over the lazy dog once more.
Filler line 05: the quick brown fox jumps over the lazy dog once more.
Filler line 06: the quick brown fox jumps over the lazy dog once more.
Filler line 07: the quick brown fox jumps over the lazy dog once more.
Filler line 08: the quick brown fox jumps over the lazy dog once more.
Filler line 09: the quick brown fox jumps over the lazy dog once more.
Filler line 10: the quick brown fox jumps over the lazy dog once more.
Filler line 11: the quick brown fox jumps over the lazy dog once more.
Filler line 12: the quick brown fox jumps over the lazy dog once more.
Reply with the single word: echo.`
)

// TestSendProbeMessages_AsTyped: the constants above have the sizes the
// evidence gives for the messages typed (section 10); the newlines matter to
// claude's paste placeholders (`+17 lines` and `+11 lines` are the 28
// newlines of the long message).
func TestSendProbeMessages_AsTyped(t *testing.T) {
	for _, tc := range []struct {
		name            string
		msg             string
		bytes, newlines int
	}{
		{"short", sendProbeShortMsg, 124, 2},
		{"medium", sendProbeMediumMsg, 646, 12},
		{"long", sendProbeLongMsg, 1829, 28},
		{"head lost", sendProbeHeadLostMsg, 1117, 18},
		{"scrolled paste", sendProbeScrolledPasteMsg, 1329, 21},
	} {
		if got, nl := len(tc.msg), strings.Count(tc.msg, "\n"); got != tc.bytes || nl != tc.newlines {
			t.Errorf("%s message: %d bytes, %d newlines; want %d and %d", tc.name, got, nl, tc.bytes, tc.newlines)
		}
	}
}

// sendTypedMultiLineFixtures are the fixtures of a message typed into an
// input box and not submitted (evidence 2026-11-12 section 10), each with
// the message typed and whether the box shows its start (ResumeInputHolds
// alone holds).
var sendTypedMultiLineFixtures = []struct {
	file, driver, msg string
	head              bool
}{
	{"claude-input-typed-multiline-short.txt", "claude", sendProbeShortMsg, true},
	// The box shows its last six lines.
	{"claude-input-typed-multiline-medium.txt", "claude", sendProbeMediumMsg, false},
	// Two paste placeholders, wrapped onto two lines.
	{"claude-input-typed-multiline-long.txt", "claude", sendProbeLongMsg, false},
	// claude kept only the end of the message, and the box shows all it
	// kept. Nothing on the screen tells this from a box that shows the end
	// of a whole message, so the check holds: the loss is the typing's, not
	// something this check can see.
	{"claude-input-typed-multiline-head-lost.txt", "claude", sendProbeHeadLostMsg, false},
	// A placeholder for the first piece is above the box's last six lines.
	{"claude-input-typed-multiline-scrolled-paste.txt", "claude", sendProbeScrolledPasteMsg, false},
	{"codex-input-typed-multiline-short.txt", "codex", sendProbeShortMsg, true},
	// The last lines, the scroll arrow on the first, and the footer's
	// `⚠ 11 · f2` line, which findResumeInputBox counts as input.
	{"codex-input-typed-multiline-medium.txt", "codex", sendProbeMediumMsg, false},
	{"codex-input-typed-multiline-long.txt", "codex", sendProbeLongMsg, false},
}

// TestSendInputHolds_TypedMultiLineFixtures: on every fixture of a message
// typed and not submitted, the check before Enter that Send runs
// (sendInputHolds) holds with the message typed, where the start of the
// message alone (ResumeInputHolds) is in the box only for the short one;
// ClassifySendScreen never calls such a screen idle, and SendScreenSubmitted
// never calls it submitted. Every such fixture in testdata/resume/ must be
// in the table.
func TestSendInputHolds_TypedMultiLineFixtures(t *testing.T) {
	dir := filepath.Join("testdata", "resume")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	listed := map[string]bool{}
	for _, f := range sendTypedMultiLineFixtures {
		listed[f.file] = true
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), "-input-typed-multiline-") && !listed[e.Name()] {
			t.Errorf("fixture %s/%s is not in sendTypedMultiLineFixtures", dir, e.Name())
		}
	}
	for _, f := range sendTypedMultiLineFixtures {
		t.Run(f.file, func(t *testing.T) {
			screen := readResumeScreen(t, f.file)
			if !sendInputHolds(f.driver, screen, f.msg) {
				t.Errorf("sendInputHolds = false, want true")
			}
			if got := ResumeInputHolds(f.driver, screen, f.msg); got != f.head {
				t.Errorf("ResumeInputHolds = %v, want %v", got, f.head)
			}
			if got := ClassifySendScreen(f.driver, screen); got.Kind != SendScreenUnknown || got.Reason != sendScreenReasonInputHasText {
				t.Errorf("ClassifySendScreen = {Kind:%q Reason:%q}, want unknown (%s)", got.Kind, got.Reason, sendScreenReasonInputHasText)
			}
			if SendScreenSubmitted(f.driver, screen, f.msg) {
				t.Errorf("SendScreenSubmitted = true, want false")
			}
		})
	}
}

// TestSendInputHolds_Refuses: the check before Enter fails when the text
// went nowhere (an empty box) or into a dialog, when the box holds another
// message or text that is not part of the one typed, when claude's paste
// placeholders stand for more lines than the message has or are of another
// form, when codex's box shows the end of the message with no scroll arrow
// above it, when the end shown is shorter than sendInputHoldsTailRunes, and
// for an unknown driver or an all-space message.
func TestSendInputHolds_Refuses(t *testing.T) {
	codexMedium := readResumeScreen(t, "codex-input-typed-multiline-medium.txt")
	const arrow = "seems to   ↑"
	if !strings.Contains(codexMedium, arrow) {
		t.Fatalf("fixture codex-input-typed-multiline-medium.txt has no %q", arrow)
	}
	cases := []struct {
		name, driver, file, screen, typed string
	}{
		{name: "claude empty box", driver: "claude", file: "claude-idle-after-reply.txt", typed: sendProbeLongMsg},
		{name: "codex empty box", driver: "codex", file: "codex-idle-after-reply.txt", typed: sendProbeLongMsg},
		{name: "claude dialog", driver: "claude", file: "claude-folder-trust-unnumbered.txt", typed: sendProbeShortMsg},
		{name: "codex dialog", driver: "codex", file: "codex-update-available.txt", typed: sendProbeShortMsg},
		{name: "claude another message's end in the box", driver: "claude", file: "claude-input-typed-multiline-medium.txt", typed: sendProbeShortMsg},
		{name: "codex another message's end in the box", driver: "codex", file: "codex-input-typed-multiline-long.txt", typed: sendProbeMediumMsg},
		// 17 + 11 lines are more than the medium message's 12 newlines.
		{name: "claude placeholders for more lines than the message", driver: "claude", file: "claude-input-typed-multiline-long.txt", typed: sendProbeMediumMsg},
		{
			name: "claude other text before the placeholders", driver: "claude",
			screen: sendClaudeBoxScreen(" zzz " + sendProbeLongPlaceholders), typed: sendProbeLongMsg,
		},
		{
			name: "claude text after the placeholders that is not the message's end", driver: "claude",
			screen: sendClaudeBoxScreen(" " + sendProbeLongPlaceholders + " once more."), typed: sendProbeLongMsg,
		},
		{
			name: "claude placeholder of another form", driver: "claude",
			screen: sendClaudeBoxScreen(" [Pasted text #1]"), typed: sendProbeLongMsg,
		},
		{
			name: "codex end of the message with no scroll arrow", driver: "codex",
			screen: strings.Replace(codexMedium, "   ↑", "", 1), typed: sendProbeMediumMsg,
		},
		{
			name: "claude end of the message shorter than the tail runes", driver: "claude",
			screen: sendClaudeBoxScreen(" bravo."), typed: sendProbeMediumMsg,
		},
		{name: "unknown driver", driver: "gemini", file: "claude-input-typed-multiline-medium.txt", typed: sendProbeMediumMsg},
		{name: "all-space message", driver: "claude", file: "claude-input-typed-multiline-medium.txt", typed: " \n "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			screen := tc.screen
			if tc.file != "" {
				screen = readResumeScreen(t, tc.file)
			}
			if sendInputHolds(tc.driver, screen, tc.typed) {
				t.Errorf("sendInputHolds(%q, %s) = true, want false", tc.driver, tc.file)
			}
		})
	}
}

// sendProbeLongPlaceholders is what claude's input box showed for the long
// message (evidence 2026-11-12 section 10), on one line.
const sendProbeLongPlaceholders = "[Pasted text #1 +17 lines][Pasted text #2 +11 lines]"

// sendClaudeBoxScreen is a claude screen whose last turn ended, with input
// (after ❯) in its input box.
func sendClaudeBoxScreen(input string) string {
	return claudeResumeScreen("❯ say hi\n\n⏺ Hi.\n\n✻ Cooked for 3s · done 6:09 PM", input)
}

// TestSendInputHolds_ClaudePasteBoxes: the shapes of claude's box that rule
// 3 of sendInputHolds accepts, the controls of the placeholder cases of
// TestSendInputHolds_Refuses: the placeholders alone, also wrapped, and a
// placeholder for the first piece with the rest of the message as text
// after it (the shape of the scrolled-paste fixture's box before it
// scrolled).
func TestSendInputHolds_ClaudePasteBoxes(t *testing.T) {
	for _, tc := range []struct{ name, input, typed string }{
		{"placeholders", " " + sendProbeLongPlaceholders, sendProbeLongMsg},
		{"placeholders wrapped", " [Pasted text #1 +17 lines][Pasted text #2 +11\n  lines]", sendProbeLongMsg},
		{"a placeholder, then the end as text", " [Pasted text #1 +17 lines]e 09: the quick brown fox jumps over the lazy dog\n  once more.\n  Reply with the single word: delta.", sendProbeHeadLostMsg},
		{"a placeholder for all the newlines", " [Pasted text #1 +12 lines]", sendProbeMediumMsg},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !sendInputHolds("claude", sendClaudeBoxScreen(tc.input), tc.typed) {
				t.Errorf("sendInputHolds of box %q = false, want true", tc.input)
			}
		})
	}
}

// sendNumberedStepsMsg is a TASK whose body is numbered steps, as a leader's
// TASK often is (synthetic; not typed in the evidence). The boxes of
// claude-input-typed-numbered-steps.txt and
// codex-input-typed-numbered-steps.txt show its end the way the medium
// message's boxes show that message's (evidence 2026-11-12 section 10):
// claude its last 6 lines, from step 7, and codex its last 11, from step 2,
// with the scroll arrow on the first.
const sendNumberedStepsMsg = `TYPE: TASK
TASK_ID: probe-f
PLAN: docs/plans/active/2026-11-12-probe.md sha256:0123456789ab

Do the steps below in order, one at a time.
1. Read the plan named in the PLAN header.
2. Read the files the plan lists for the slice.
3. Write the tests the plan asks for first.
4. Run the tests and see them fail.
5. Change the code until the tests pass.
6. Run gofmt on every file you changed.
7. Run the checks the plan names.
8. Fix what fails and run them again.
9. Commit the slice as the plan says.

Do not push the branch.
Reply with the single word: foxtrot.`

// sendTypedNumberedFixtures are the fixtures of sendNumberedStepsMsg typed
// into an input box and not submitted, whose box's first line is one of its
// numbered steps.
var sendTypedNumberedFixtures = []struct{ file, driver string }{
	{"claude-input-typed-numbered-steps.txt", "claude"},
	{"codex-input-typed-numbered-steps.txt", "codex"},
}

// TestSendInputHolds_NumberedStepOnTheMarkerLine covers the cross-review
// finding of cycle 1 (AC4): when the box's first line, which carries the
// marker, is a numbered step of the message, the check before Enter finds
// the box and holds, though findResumeInputBox takes that line for a choice
// dialog's cursor line. Resume keeps that reading (ResumeInputHolds is
// false, ClassifyResumeScreen says choice dialog), and so does the check
// after Enter (SendScreenSubmitted is false). Every such fixture in
// testdata/resume/ must be in the table.
func TestSendInputHolds_NumberedStepOnTheMarkerLine(t *testing.T) {
	dir := filepath.Join("testdata", "resume")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	listed := map[string]bool{}
	for _, f := range sendTypedNumberedFixtures {
		listed[f.file] = true
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), "-input-typed-numbered-") && !listed[e.Name()] {
			t.Errorf("fixture %s/%s is not in sendTypedNumberedFixtures", dir, e.Name())
		}
	}
	for _, f := range sendTypedNumberedFixtures {
		t.Run(f.file, func(t *testing.T) {
			screen := readResumeScreen(t, f.file)
			if _, reason := findResumeInputBox(f.driver, resumeScreenLines(screen)); reason != resumeReasonChoiceDialog {
				t.Fatalf("findResumeInputBox reason = %q, want %q (the fixture's first box line must be numbered)", reason, resumeReasonChoiceDialog)
			}
			if !sendInputHolds(f.driver, screen, sendNumberedStepsMsg) {
				t.Errorf("sendInputHolds = false, want true")
			}
			if ResumeInputHolds(f.driver, screen, sendNumberedStepsMsg) {
				t.Errorf("ResumeInputHolds = true, want false (resume's reading is unchanged)")
			}
			if got := ClassifyResumeScreen(f.driver, screen); got.Kind != ResumeScreenUnknown || got.Reason != resumeReasonChoiceDialog {
				t.Errorf("ClassifyResumeScreen = %+v, want unknown (%s)", got, resumeReasonChoiceDialog)
			}
			if SendScreenSubmitted(f.driver, screen, sendNumberedStepsMsg) {
				t.Errorf("SendScreenSubmitted = true, want false")
			}
		})
	}
}

// sendQuotedDialog is a message made of the choice dialog of screen: its
// lines from the last marker line on, trimmed, the marker dropped. It
// starts with the dialog's cursor line, so the dialog's text holds the
// message's start, and it ends with the dialog's last line.
func sendQuotedDialog(t *testing.T, driver, screen string) string {
	t.Helper()
	lines := resumeScreenLines(screen)
	box, ok := resumeMarkerLineBox(driver, lines)
	if !ok {
		t.Fatalf("no %s marker line on the screen", driver)
	}
	parts := []string{box.first}
	for _, l := range lines[box.marker+1:] {
		parts = append(parts, strings.TrimSpace(l))
	}
	return strings.Join(parts, "\n")
}

// TestSendInputHolds_NumberedMarkerLineRefuses: a numbered marker line is
// the input box for the check before Enter only when it is part of the
// message and the lines around it are an input box. A real choice dialog
// fails whether the message holds its text or not: claude's cursor line has
// no rule right above it, and codex's dialog has a hint line, not a footer,
// under it. A box with a numbered line that is not part of the message
// fails although the message's start is below it, and so do the numbered
// fixtures with the rule above claude's box or codex's footer taken away.
func TestSendInputHolds_NumberedMarkerLineRefuses(t *testing.T) {
	dialogs := []struct{ file, driver string }{
		{"claude-folder-trust.txt", "claude"},
		{"claude-dialog-below-old-input.txt", "claude"},
		{"codex-hooks-review.txt", "codex"},
		{"codex-update-available.txt", "codex"},
	}
	type refuseCase struct{ name, driver, screen, typed string }
	var cases []refuseCase
	for _, d := range dialogs {
		screen := readResumeScreen(t, d.file)
		cases = append(cases,
			refuseCase{d.file + ", a message without its options", d.driver, screen, sendNumberedStepsMsg},
			refuseCase{d.file + ", a message made of it", d.driver, screen, sendQuotedDialog(t, d.driver, screen)},
		)
	}
	claudeNumbered := readResumeScreen(t, "claude-input-typed-numbered-steps.txt")
	codexNumbered := readResumeScreen(t, "codex-input-typed-numbered-steps.txt")
	const ruleAbove = resumeTestRule + "\n❯ 7."
	const footer = "\n\n  GPT-6-Luna · low · Context 7% used · main · No chan…\n                                            ⚠ 11 · f2\n"
	if !strings.Contains(claudeNumbered, ruleAbove) || !strings.HasSuffix(codexNumbered, footer) {
		t.Fatalf("the numbered fixtures lost the rule above claude's box or codex's footer")
	}
	short := strings.ReplaceAll(sendProbeShortMsg, "\n", "\n  ")
	cases = append(cases,
		refuseCase{"claude, the numbered fixture with another message", "claude", claudeNumbered, sendProbeMediumMsg},
		refuseCase{"codex, the numbered fixture with another message", "codex", codexNumbered, sendProbeLongMsg},
		refuseCase{"claude, the numbered fixture with no rule above the box", "claude",
			strings.Replace(claudeNumbered, ruleAbove, "\n❯ 7.", 1), sendNumberedStepsMsg},
		refuseCase{"codex, the numbered fixture with no footer", "codex",
			strings.TrimSuffix(codexNumbered, footer) + "\n", sendNumberedStepsMsg},
		refuseCase{"claude, a numbered line not of the message above its start", "claude",
			sendClaudeBoxScreen(" 1. Yes, I trust this folder\n  " + short), sendProbeShortMsg},
		refuseCase{"codex, a numbered line not of the message above its start", "codex",
			codexResumeScreen("› go\n\n  Worked for 4s • 18:17", " 1. Review hooks\n  "+short), sendProbeShortMsg},
	)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if sendInputHolds(tc.driver, tc.screen, tc.typed) {
				t.Errorf("sendInputHolds = true, want false")
			}
		})
	}
}
