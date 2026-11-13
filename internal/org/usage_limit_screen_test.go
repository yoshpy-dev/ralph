package org

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// usageLimitFixtureWant is what UsageLimitScreen should say about one
// fixture of testdata/resume/. line is the whole line it returns; "" means
// the screen is no usage limit (ok false). has lists the parts the line must
// hold: the limit sentence and the reset time the screen shows with it.
type usageLimitFixtureWant struct {
	line string
	has  []string
}

// usageLimitFixtureWants holds an expectation for every file of
// testdata/resume/ (TestUsageLimitScreen_EveryFixture fails on a file with
// none). Only the three usage limit screens are true.
var usageLimitFixtureWants = map[string]usageLimitFixtureWant{
	// The usage limits. The weekly limit is a ⎿ line right under the
	// prompt, its time zone wrapped onto the next line; the two spaces after
	// ⎿ become one. codex puts the reset time on the next line.
	"claude-session-limit.txt": {
		line: "⏺ You've hit your session limit · resets 3:45pm",
		has:  []string{"You've hit your session limit", "resets 3:45pm"},
	},
	"claude-weekly-limit-under-prompt.txt": {
		line: "⎿ You've hit your weekly limit · resets 11pm (Asia/Tokyo) /usage-credits to finish what you’re working on.",
		has:  []string{"You've hit your weekly limit", "resets 11pm", "(Asia/Tokyo)"},
	},
	"codex-usage-limit.txt": {
		line: "■ You've hit your usage limit. Upgrade to Pro or purchase more credits, or try again at 3:05 PM.",
		has:  []string{"You've hit your usage limit", "try again at 3:05 PM"},
	},

	// Stopped on a network error, which resume continues: no usage limit.
	"claude-enotfound.txt": {},

	// The CLI waits for Enter after a limit reset (an old session limit is
	// above it), or waits or retries on its own after a limit or a network
	// error.
	"claude-press-enter.txt":              {},
	"claude-continuing-automatically.txt": {},
	"claude-retrying.txt":                 {},
	"codex-network-waiting.txt":           {},

	// A limit or an error quoted in a prompt, in a reply, or in a tool
	// result, or an old one above a later reply: never a stop.
	"claude-prompt-quotes-limit-last.txt":        {},
	"claude-prompt-quotes-error-interrupted.txt": {},
	"claude-prompt-quotes-error-then-reply.txt":  {},
	"claude-reply-error-paragraph.txt":           {},
	"claude-reply-quotes-error.txt":              {},
	"claude-tool-output-error-paragraph.txt":     {},
	"claude-old-error-in-history.txt":            {},

	// Text in the input box. codex-input-typed.txt and the two codex drafts
	// are typed under a usage limit (the drafts under the history of
	// codex-usage-limit.txt), which does not count while the input box holds
	// text.
	"claude-input-draft-after-empty-line.txt":         {},
	"claude-input-has-text.txt":                       {},
	"claude-input-typed-multiline-head-lost.txt":      {},
	"claude-input-typed-multiline-long.txt":           {},
	"claude-input-typed-multiline-medium.txt":         {},
	"claude-input-typed-multiline-scrolled-paste.txt": {},
	"claude-input-typed-multiline-short.txt":          {},
	"claude-input-typed-numbered-steps.txt":           {},
	"claude-input-typed-wrapped.txt":                  {},
	"claude-working-input-typed.txt":                  {},
	"codex-input-draft-after-empty-line.txt":          {},
	"codex-input-draft-no-footer.txt":                 {},
	"codex-input-typed-multiline-long.txt":            {},
	"codex-input-typed-multiline-medium.txt":          {},
	"codex-input-typed-multiline-short.txt":           {},
	"codex-input-typed-numbered-steps.txt":            {},
	"codex-input-typed.txt":                           {},
	"codex-working-input-typed.txt":                   {},

	// Idle after a reply, or a fresh session.
	"claude-idle-after-multiline-reply.txt": {},
	"claude-idle-after-reply.txt":           {},
	"claude-idle-fresh-placeholder.txt":     {},
	"codex-idle-after-multiline-reply.txt":  {},
	"codex-idle-after-reply.txt":            {},
	"codex-idle-background-terminal.txt":    {},
	"codex-idle-fresh-update-notice.txt":    {},

	// Working, just submitted, or queued.
	"claude-queued-while-working.txt":          {},
	"claude-submitted-1s.txt":                  {},
	"claude-submitted-just-now.txt":            {},
	"claude-submitted-multiline-long-1.5s.txt": {},
	"claude-submitted-multiline-long.txt":      {},
	"claude-working-thinking-start.txt":        {},
	"claude-working-thinking.txt":              {},
	"claude-working-tool-3s-later.txt":         {},
	"claude-working-tool-hooks.txt":            {},
	"claude-working-tool.txt":                  {},
	"codex-queued-steer-while-working.txt":     {},
	"codex-submitted-just-now.txt":             {},
	"codex-submitted-multiline-long-1.5s.txt":  {},
	"codex-submitted-multiline-long.txt":       {},
	"codex-working-thinking.txt":               {},
	"codex-working-tool-3s-later.txt":          {},
	"codex-working-tool.txt":                   {},

	// Choice dialogs.
	"claude-dialog-below-old-input.txt":  {},
	"claude-folder-trust-unnumbered.txt": {},
	"claude-folder-trust.txt":            {},
	"codex-folder-trust.txt":             {},
	"codex-hooks-review.txt":             {},
	"codex-update-available.txt":         {},

	// A message with a line that starts with the marker, typed and then
	// submitted (docs/evidence/org-send-marker-probe-2026-11-13.md).
	"claude-input-typed-marker-line.txt":         {},
	"claude-input-typed-marker-line-one-nl.txt":  {},
	"claude-input-typed-marker-line-wrapped.txt": {},
	"claude-input-typed-marker-no-nl.txt":        {},
	"claude-input-typed-paste-short.txt":         {},
	"claude-submitted-marker-line.txt":           {},
	"codex-input-typed-marker-line.txt":          {},
	"codex-input-typed-marker-line-last.txt":     {},
	"codex-input-typed-marker-line-scrolled.txt": {},
	"codex-input-typed-marker-numbered.txt":      {},
	"codex-submitted-marker-line.txt":            {},
}

// assertUsageLimitLineShape checks the shape of a line UsageLimitScreen
// returned with ok: one line of valid UTF-8 with no control character, no
// run of spaces, no space at either end, and at most 300 runes.
func assertUsageLimitLineShape(t *testing.T, line string) {
	t.Helper()
	if !utf8.ValidString(line) {
		t.Errorf("line %q is not valid UTF-8", line)
	}
	if i := strings.IndexFunc(line, func(r rune) bool { return unicode.IsControl(r) || isInboxLineBreak(r) }); i >= 0 {
		t.Errorf("line %q holds a control character or a line break at byte %d", line, i)
	}
	if strings.Contains(line, "  ") || strings.TrimSpace(line) != line {
		t.Errorf("line %q has a run of spaces or a space at an end", line)
	}
	if n := utf8.RuneCountInString(line); n > 300 {
		t.Errorf("line is %d runes, want at most 300", n)
	}
}

func TestUsageLimitScreen_EveryFixture(t *testing.T) {
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
		want, ok := usageLimitFixtureWants[name]
		if !ok {
			t.Errorf("fixture %s/%s has no expectation in usageLimitFixtureWants", dir, name)
			continue
		}
		t.Run(name, func(t *testing.T) {
			driver := strings.SplitN(name, "-", 2)[0]
			screen := readResumeScreen(t, name)
			line, ok := UsageLimitScreen(driver, screen)
			if ok != (want.line != "") || line != want.line {
				t.Fatalf("UsageLimitScreen(%q) = (%q, %v), want (%q, %v)", driver, line, ok, want.line, want.line != "")
			}
			if !ok {
				return
			}
			for _, part := range want.has {
				if !strings.Contains(line, part) {
					t.Errorf("line %q does not hold %q", line, part)
				}
			}
			assertUsageLimitLineShape(t, line)
			if got := ClassifyResumeScreen(driver, screen); got.Kind != ResumeScreenStopped {
				t.Errorf("ClassifyResumeScreen says %q, but a usage limit is a stopped screen", got.Kind)
			}
		})
	}
	for name := range usageLimitFixtureWants {
		if !seen[name] {
			t.Errorf("usageLimitFixtureWants names %s, which is not in %s", name, dir)
		}
	}
}

func TestUsageLimitScreen_Edges(t *testing.T) {
	session := readResumeScreen(t, "claude-session-limit.txt")
	long := "⏺ You've hit your session limit · resets 3:45pm " + strings.Repeat("x", 400)
	cases := []struct {
		name   string
		driver string
		screen string
		line   string // "" means ok is false
	}{
		{
			name: "claude Opus limit above a status line", driver: "claude",
			screen: claudeResumeScreen("⏺ You've hit your Opus limit · resets 3:45pm\n\n✻ Worked for 5s · done 3:40 PM", ""),
			line:   "⏺ You've hit your Opus limit · resets 3:45pm",
		},
		{
			name: "claude Sonnet limit as a ⎿ line under the prompt", driver: "claude",
			screen: claudeResumeScreen("❯ go on\n  ⎿  You've hit your Sonnet limit · resets 3:45pm\n\n✻ Worked for 0s · done 3:01 PM", ""),
			line:   "⎿ You've hit your Sonnet limit · resets 3:45pm",
		},
		{
			name: "claude weekly limit whose reset time wrapped onto the next line", driver: "claude",
			screen: claudeResumeScreen("⏺ You've hit your weekly limit · resets Mon\n  12:00am (Asia/Tokyo)", ""),
			line:   "⏺ You've hit your weekly limit · resets Mon 12:00am (Asia/Tokyo)",
		},
		{
			name: "codex Usage limit reached after ■", driver: "codex",
			screen: codexResumeScreen("■ Usage limit reached. Try again at 3:05 PM.", ""),
			line:   "■ Usage limit reached. Try again at 3:05 PM.",
		},
		{
			name: "codex usage limit after ⚠", driver: "codex",
			screen: codexResumeScreen("⚠ You've hit your usage limit. Try again at 3:05 PM.", ""),
			line:   "⚠ You've hit your usage limit. Try again at 3:05 PM.",
		},
		{
			name: "claude API error is a stop but no usage limit", driver: "claude",
			screen: claudeResumeScreen("❯ say hi\n\n⏺ API Error: Repeated 529 Overloaded errors", ""),
		},
		{
			name: "claude lost connection is a stop but no usage limit", driver: "claude",
			screen: claudeResumeScreen("⏺ Connection lost before a response was produced", ""),
		},
		{
			name: "an API error whose words name a limit is no usage limit", driver: "claude",
			screen: claudeResumeScreen("⏺ API Error: You've hit your session limit · resets 3:45pm", ""),
		},
		{
			name: "claude has no stopped sentence Usage limit reached", driver: "claude",
			screen: claudeResumeScreen("⏺ Usage limit reached · resets 3:45pm", ""),
		},
		{
			name: "codex has no stopped sentence of claude's session limit", driver: "codex",
			screen: codexResumeScreen("■ You've hit your session limit · resets 3:45pm", ""),
		},
		{
			name: "a limit the user quoted in the last prompt", driver: "claude",
			screen: claudeResumeScreen("⏺ Done.\n\n❯ You've hit your weekly limit? Check it.", ""),
		},
		{
			name: "a limit quoted after other words of a reply", driver: "claude",
			screen: claudeResumeScreen("⏺ The banner now reads You've hit your session limit · resets 3:45pm.", ""),
		},
		{
			name: "a limit with no glyph before it", driver: "claude",
			screen: claudeResumeScreen("You've hit your session limit · resets 3:45pm", ""),
		},
		{
			name: "a codex limit after the item glyph", driver: "codex",
			screen: codexResumeScreen("• You've hit your usage limit. Try again at 3:05 PM.", ""),
		},
		{
			name: "a limit above an input box that holds text", driver: "claude",
			screen: claudeResumeScreen("⏺ You've hit your session limit · resets 3:45pm", " keep going"),
		},
		{
			name: "a limit the CLI waits out on its own", driver: "claude",
			screen: claudeResumeScreen("⏺ You've hit your Opus limit · resets 3:45pm\n  Continuing shortly · Esc to cancel\n\n✻ Worked for 9s · done 3:44 PM", ""),
		},
		{
			name: "a limit that reset and waits for Enter", driver: "claude",
			screen: claudeResumeScreen("⏺ You've hit your session limit · resets 3:45pm\n\nYour usage limit has reset\nPress enter to continue", ""),
		},
		{
			name: "an old limit above a later reply", driver: "claude",
			screen: claudeResumeScreen("⏺ You've hit your session limit · resets 3:45pm\n\n❯ go on\n\n⏺ Done.", ""),
		},
		{name: "unknown driver", driver: "gemini", screen: session},
		{name: "empty driver", driver: "", screen: session},
		{name: "driver names are case-sensitive", driver: "Claude", screen: session},
		{name: "empty screen", driver: "claude", screen: ""},
		{
			// A tab, a carriage return, U+2028, and U+0085 become spaces; an
			// escape, a bell, a NUL, and a DEL are dropped.
			name: "control characters in the limit's line", driver: "claude",
			screen: claudeResumeScreen("⏺ You've hit your session limit\t·\x1b[0m resets\x07 3:45pm\r(Asia/Tokyo)\x00\xe2\x80\xa8\xc2\x85now\x7f", ""),
			line:   "⏺ You've hit your session limit ·[0m resets 3:45pm (Asia/Tokyo) now",
		},
		{
			name: "a byte that is not UTF-8 in the limit's line", driver: "claude",
			screen: claudeResumeScreen("⏺ You've hit your session limit · resets \xff3:45pm", ""),
			line:   "⏺ You've hit your session limit · resets " + string(utf8.RuneError) + "3:45pm",
		},
		{
			name: "a long limit line is cut to 300 runes", driver: "claude",
			screen: claudeResumeScreen(long, ""),
			line:   string([]rune(long)[:299]) + "…",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			line, ok := UsageLimitScreen(tc.driver, tc.screen)
			if ok != (tc.line != "") || line != tc.line {
				t.Fatalf("UsageLimitScreen(%q) = (%q, %v), want (%q, %v)", tc.driver, line, ok, tc.line, tc.line != "")
			}
			if ok {
				assertUsageLimitLineShape(t, line)
			}
		})
	}
}

// TestUsageLimitSentences_AreStoppedSentences ties the usage limit marks to
// the stopped sentences (plan D5): each mark is the text of exactly one
// stopped sentence of its driver, the marks are the limit sentences of the
// plan and no others (the network errors are not marked), and a stopped
// sentence that names a limit is marked.
func TestUsageLimitSentences_AreStoppedSentences(t *testing.T) {
	want := map[string][]string{
		"claude": {"You've hit your session limit", "You've hit your weekly limit", "You've hit your Opus limit", "You've hit your Sonnet limit"},
		"codex":  {"You've hit your usage limit", "Usage limit reached"},
	}
	if len(resumeScreenSentences) != len(want) {
		t.Errorf("sentence tables for %d drivers, want marks for %d", len(resumeScreenSentences), len(want))
	}
	for driver, table := range resumeScreenSentences {
		stopped := make([]string, 0, len(table.Stopped))
		for _, s := range table.Stopped {
			stopped = append(stopped, s.Text)
		}
		for i, mark := range table.UsageLimit {
			if n := countString(stopped, mark); n != 1 {
				t.Errorf("driver %q: usage limit mark %q is the text of %d stopped sentences, want 1", driver, mark, n)
			}
			if slices.Contains(table.UsageLimit[:i], mark) {
				t.Errorf("driver %q: usage limit mark %q is listed twice", driver, mark)
			}
		}
		if !slices.Equal(table.UsageLimit, want[driver]) {
			t.Errorf("driver %q: usage limit marks = %q, want %q", driver, table.UsageLimit, want[driver])
		}
		for _, s := range stopped {
			if strings.Contains(strings.ToLower(s), "limit") && !slices.Contains(table.UsageLimit, s) {
				t.Errorf("driver %q: stopped sentence %q names a limit but is not marked in UsageLimit", driver, s)
			}
		}
	}
}

// TestUsageLimitScreen_EachStoppedSentence: every stopped sentence of each
// driver, after each of the driver's glyphs, makes a stopped screen, and
// UsageLimitScreen is true for it exactly when the sentence is marked.
func TestUsageLimitScreen_EachStoppedSentence(t *testing.T) {
	screens := map[string]func(above string) string{
		"claude": func(above string) string { return claudeResumeScreen(above, "") },
		"codex":  func(above string) string { return codexResumeScreen(above, "") },
	}
	for driver, table := range resumeScreenSentences {
		build, ok := screens[driver]
		if !ok {
			t.Errorf("no screen builder for driver %q", driver)
			continue
		}
		for _, s := range table.Stopped {
			for _, glyph := range table.StopGlyphs {
				block := glyph + " " + s.Text + " · resets 3:45pm"
				screen := build(block)
				if got := ClassifyResumeScreen(driver, screen); got.Kind != ResumeScreenStopped {
					t.Fatalf("driver %q: ClassifyResumeScreen(%q) = %q, want stopped", driver, block, got.Kind)
				}
				marked := slices.Contains(table.UsageLimit, s.Text)
				line, ok := UsageLimitScreen(driver, screen)
				if ok != marked {
					t.Errorf("driver %q: UsageLimitScreen(%q) = %v, want %v (marked)", driver, block, ok, marked)
				}
				if ok && line != block {
					t.Errorf("driver %q: line = %q, want %q", driver, line, block)
				}
			}
		}
	}
}

func TestUsageLimitLine(t *testing.T) {
	cases := []struct {
		name string
		text string
		want string
	}{
		{name: "empty", text: "", want: ""},
		{name: "white space only", text: " \t \r ", want: ""},
		{name: "runs of white space become one space", text: "  a \t\t b  ", want: "a b"},
		{name: "300 runes stay whole", text: strings.Repeat("a", 300), want: strings.Repeat("a", 300)},
		{name: "301 runes are cut to 299 and …", text: strings.Repeat("a", 301), want: strings.Repeat("a", 299) + "…"},
		{name: "the cut counts runes, not bytes", text: strings.Repeat("あ", 301), want: strings.Repeat("あ", 299) + "…"},
		{name: "no space before …", text: strings.Repeat("a", 298) + " bbbb", want: strings.Repeat("a", 298) + "…"},
		{name: "control characters are dropped", text: "a\x1bb\x00c\x7fd\xc2\x9be", want: "abcde"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := usageLimitLine(tc.text); got != tc.want {
				t.Errorf("usageLimitLine(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}
