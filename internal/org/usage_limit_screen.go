package org

import (
	"slices"
	"strings"
	"unicode"
)

// The usage limit screen of the cross-org watch (plan
// 2026-11-13-watch-usage-limited, D5 and D6). A seat that hits its
// account's usage limit ends its turn: herdr reports it idle or done and
// nothing reaches the manifest, so the watch's stall check holds for it.
// Before it raises that stall, the watch reads the seat's pane and asks
// UsageLimitScreen whether the screen shows the seat stopped on a usage
// limit. The answer rests on `ralph org resume`'s screen model and sentence
// tables (resume_screens.go), and a screen counts in one of two ways. It is
// one ClassifyResumeScreen reads as stopped, and the stopped sentence that
// decided it is one the driver's table marks in UsageLimit. Or it is one
// ClassifyResumeScreen reads as waiting, and a block of the footer below the
// input box holds a waiting sentence the table marks in UsageLimitWaiting:
// the CLI waits for the limit to reset and goes on by itself then (plan
// 2026-11-13-claude-5h-usage-limit, D1 and D2; Claude Code 2.1.316 at its
// 5-hour limit, docs/evidence/claude-5h-limit-screen-2026-11-13.md). Such a
// sentence in a block above the input box, a reply or a tool output that
// quotes it, does not count: the footer is drawn by the CLI and no reply
// reaches it.

// usageLimitLineMaxRunes is the most runes a line of UsageLimitScreen holds,
// the "…" that marks a cut included. The watch puts one such line for each
// seat into one inbox item, whose body is capped at
// protocol.DefaultMaxBodyChars (2,000 characters) for all the seats
// together.
const usageLimitLineMaxRunes = 300

// UsageLimitScreen reports whether screen, the text of a pane of a seat
// whose driver is driver ("claude" or "codex"; any other value is false),
// shows the seat stopped on a usage limit, in one of two ways:
//
//   - ClassifyResumeScreen reads it as stopped, and the stopped sentence
//     that decided it is one the driver's table marks as a usage limit
//     (resumeSentenceTable.UsageLimit);
//   - ClassifyResumeScreen reads it as waiting, and a block of the footer
//     below the input box holds a waiting sentence the driver's table marks
//     as a wait for a usage limit to reset
//     (resumeSentenceTable.UsageLimitWaiting; claude's
//     `Continuing automatically at 3:50pm` and `Continuing shortly`).
//
// A stop on an API or network error (`API Error:`, `Connection lost before
// a response was produced`) is false. A wait for an API retry
// (`Retrying in`, `Waiting for API response`) is false too, and so is a
// screen ClassifyResumeScreen reads as waiting only because a block above
// the input box holds a marked waiting sentence: a reply or a tool output
// that quotes it. Every other screen is false as well: one that waits for
// Enter after the limit reset (enter_only), and every unknown screen, a
// limit quoted in a prompt or in the middle of a reply included.
//
// line is, for a stop, the block that holds the sentence
// (ResumeScreen.Line): the lines the pane's width wrapped and the reset time
// printed after the sentence in the same block
// (`· resets 11pm (Asia/Tokyo)`, `try again at 3:05 PM.`) included. For a
// wait it is the footer block that holds the waiting sentence, whole: the
// limit and its reset time (`Usage limit reached · limit resets 3:50pm`)
// and the status lines under them (`🤖 Haiku 5.5 | ⚡ low`) included. Either
// is one line (usageLimitLine). It is empty when ok is false.
func UsageLimitScreen(driver, screen string) (line string, ok bool) {
	table, known := resumeScreenSentences[driver]
	if !known {
		return "", false
	}
	res := ClassifyResumeScreen(driver, screen)
	switch res.Kind {
	case ResumeScreenStopped:
		s, found := matchedStopSentence(res.Line, table)
		if !found || !slices.Contains(table.UsageLimit, s.Text) {
			return "", false
		}
		return usageLimitLine(res.Line), true
	case ResumeScreenWaiting:
		block, found := usageLimitWaitingFooter(driver, screen, table)
		if !found {
			return "", false
		}
		return usageLimitLine(block), true
	}
	return "", false
}

// usageLimitWaitingFooter returns the text of the first block of the footer
// below driver's input box in screen that holds one of
// table.UsageLimitWaiting anywhere in it. It finds the input box and the
// footer's blocks as ClassifyResumeScreen does (findResumeInputBox and
// resumeInputBox.footerBlocks), and reads no block above the input box. It
// is false when there is no input box or no footer block holds such a
// sentence.
func usageLimitWaitingFooter(driver, screen string, table resumeSentenceTable) (string, bool) {
	lines := resumeScreenLines(screen)
	box, reason := findResumeInputBox(driver, lines)
	if reason != "" {
		// UsageLimitScreen never gets here: ClassifyResumeScreen reads a
		// screen as waiting only after findResumeInputBox found an input box
		// in the same lines. Were it reached, false keeps the seat's stall.
		return "", false
	}
	for _, b := range box.footerBlocks(driver, lines) {
		for _, s := range table.UsageLimitWaiting {
			if strings.Contains(b.text, s) {
				return b.text, true
			}
		}
	}
	return "", false
}

// matchedStopSentence returns the stopped sentence of table that
// findResumeStop finds at the start of block, the text of a candidate
// block. It tries the sentences one at a time in table order, as
// findResumeStop does, so the one it returns is the one that made the
// screen stopped.
func matchedStopSentence(block string, table resumeSentenceTable) (resumeSentence, bool) {
	for _, s := range table.Stopped {
		one := resumeSentenceTable{Stopped: []resumeSentence{s}, StopGlyphs: table.StopGlyphs}
		if _, ok := findResumeStop(resumeBlock{text: block}, one); ok {
			return s, true
		}
	}
	return resumeSentence{}, false
}

// usageLimitLine returns text as one line for an inbox item's body. Each
// white-space rune (a tab, a carriage return, U+2028, U+0085, and the like)
// becomes a space and every other control character (unicode.IsControl: an
// escape, a bell, a NUL, a DEL) is dropped; each run of spaces becomes one
// space, with none at either end; and a line longer than
// usageLimitLineMaxRunes runes is cut to that many, the last of them "…". A
// byte that is not valid UTF-8 becomes the replacement character
// (strings.Map), so the line is valid UTF-8.
func usageLimitLine(text string) string {
	clean := strings.Map(func(r rune) rune {
		switch {
		case unicode.IsSpace(r):
			return ' '
		case unicode.IsControl(r):
			return -1
		}
		return r
	}, text)
	runes := []rune(strings.Join(strings.Fields(clean), " "))
	if len(runes) <= usageLimitLineMaxRunes {
		return string(runes)
	}
	return strings.TrimRight(string(runes[:usageLimitLineMaxRunes-1]), " ") + "…"
}
