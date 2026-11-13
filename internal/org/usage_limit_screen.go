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
// tables (resume_screens.go): the screen must be one ClassifyResumeScreen
// reads as stopped, and the stopped sentence that decided it must be one the
// driver's table marks in UsageLimit.

// usageLimitLineMaxRunes is the most runes a line of UsageLimitScreen holds,
// the "…" that marks a cut included. The watch puts one such line for each
// seat into one inbox item, whose body is capped at
// protocol.DefaultMaxBodyChars (2,000 characters) for all the seats
// together.
const usageLimitLineMaxRunes = 300

// UsageLimitScreen reports whether screen, the text of a pane of a seat
// whose driver is driver ("claude" or "codex"; any other value is false),
// shows the seat stopped on a usage limit: ClassifyResumeScreen reads it as
// stopped, and the stopped sentence that decided it is one the driver's
// table marks as a usage limit (resumeSentenceTable.UsageLimit). A stop on
// an API or network error (`API Error:`, `Connection lost before a response
// was produced`) is false, and so is every screen ClassifyResumeScreen does
// not read as stopped: one that waits for Enter after the limit reset
// (enter_only), one still waiting or retrying, and every unknown screen, a
// limit quoted in a prompt or in the middle of a reply included.
//
// line is the block that holds the sentence (ResumeScreen.Line): the lines
// the pane's width wrapped and the reset time printed after the sentence in
// the same block (`· resets 11pm (Asia/Tokyo)`, `try again at 3:05 PM.`)
// included, as one line (usageLimitLine). It is empty when ok is false.
func UsageLimitScreen(driver, screen string) (line string, ok bool) {
	table, known := resumeScreenSentences[driver]
	if !known {
		return "", false
	}
	res := ClassifyResumeScreen(driver, screen)
	if res.Kind != ResumeScreenStopped {
		return "", false
	}
	s, found := matchedStopSentence(res.Line, table)
	if !found || !slices.Contains(table.UsageLimit, s.Text) {
		return "", false
	}
	return usageLimitLine(res.Line), true
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
