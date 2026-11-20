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
// tables (resume_screens.go), and a screen counts in one of three ways. It is
// one ClassifyResumeScreen reads as stopped, and the stopped sentence that
// decided it is one the driver's table marks in UsageLimit. Or it is one
// ClassifyResumeScreen reads as waiting, and a block of the footer below the
// input box holds a waiting sentence the table marks in UsageLimitWaiting:
// the CLI waits for the limit to reset and goes on by itself then (plan
// 2026-11-13-claude-5h-usage-limit, D1 and D2; Claude Code 2.1.316 at its
// 5-hour limit, docs/evidence/claude-5h-limit-screen-2026-11-13.md). Such a
// sentence in a block above the input box, a reply or a tool output that
// quotes it, does not count: the footer is drawn by the CLI and no reply
// reaches it. Or, for claude only, it is one ClassifyResumeScreen reads as
// waiting because the CLI retries inside its turn until the limit resets:
// a block of the current turn holds a marked UsageLimit sentence right after
// one of claude's StopGlyphs and white space, with `resets <time>` after it,
// and the last block above the input box is claude's retry status line with
// the same time in parentheses (plan
// docs/plans/archive/2026-11-20-watch-usage-limit-retry.md; issue #353,
// Claude Code 2.1.316 with CLAUDE_CODE_RETRY_WATCHDOG=1 at its weekly
// limit). The time ties the retry to the limit: a retry after a network
// error shows no reset time, so a limit sentence a tool printed earlier in
// the turn does not count then.

// usageLimitLineMaxRunes is the most runes a line of UsageLimitScreen holds,
// the "…" that marks a cut included. The watch puts one such line for each
// seat into one inbox item, whose body is capped at
// protocol.DefaultMaxBodyChars (2,000 characters) for all the seats
// together.
const usageLimitLineMaxRunes = 300

// UsageLimitScreen reports whether screen, the text of a pane of a seat
// whose driver is driver ("claude" or "codex"; any other value is false),
// shows the seat stopped on a usage limit, in one of three ways, tried in
// this order:
//
//   - ClassifyResumeScreen reads it as stopped, and the stopped sentence
//     that decided it is one the driver's table marks as a usage limit
//     (resumeSentenceTable.UsageLimit);
//   - ClassifyResumeScreen reads it as waiting, and a block of the footer
//     below the input box holds a waiting sentence the driver's table marks
//     as a wait for a usage limit to reset
//     (resumeSentenceTable.UsageLimitWaiting; claude's
//     `Continuing automatically at 3:50pm` and `Continuing shortly`);
//   - for claude only, ClassifyResumeScreen reads it as waiting, the last
//     block above the input box is claude's retry status line, and a block
//     of the current turn holds a usage limit that resets at the time the
//     status line shows (usageLimitRetryingTurn; plan
//     docs/plans/archive/2026-11-20-watch-usage-limit-retry.md). On the
//     screen of issue #353 the turn holds
//     `⎿  You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)`
//     and the status line, wrapped in two columns, reads
//     `✻ Weekly    · Retrying in 29m (Nov 21 at 3pm) · attempt li…      1/55`.
//
// A stop on an API or network error (`API Error:`, `Connection lost before
// a response was produced`) is false. A wait for an API retry
// (`Retrying in`, `Waiting for API response`) is false too, unless the third
// way holds, and so is a screen ClassifyResumeScreen reads as waiting only
// because a block above the input box holds a marked waiting sentence: a
// reply or a tool output that quotes it. Every other screen is false as
// well: one that waits for Enter after the limit reset (enter_only), and
// every unknown screen, a limit quoted in a prompt or in the middle of a
// reply included.
//
// line is, for a stop, the block that holds the sentence
// (ResumeScreen.Line): the lines the pane's width wrapped and the reset time
// printed after the sentence in the same block
// (`· resets 11pm (Asia/Tokyo)`, `try again at 3:05 PM.`) included. For a
// wait in the footer it is the footer block that holds the waiting
// sentence, whole: the limit and its reset time
// (`Usage limit reached · limit resets 3:50pm`) and the status lines under
// them (`🤖 Haiku 5.5 | ⚡ low`) included. For a retry it is the block of the
// turn that holds the limit, from the glyph before the limit sentence to the
// block's end: the limit and its reset time, the time zone the pane's width
// wrapped onto the next line included. Each is one line (usageLimitLine). It
// is empty when ok is false.
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
		if block, found := usageLimitWaitingFooter(driver, screen, table); found {
			return usageLimitLine(block), true
		}
		if driver != "claude" {
			return "", false
		}
		text, found := usageLimitRetryingTurn(driver, screen, table)
		if !found {
			return "", false
		}
		return usageLimitLine(text), true
	}
	return "", false
}

// usageLimitRetryWords are the words claude's retry status line holds
// (`✻ Can't reach the… · Retrying in 4s · attempt 13/300`, evidence
// 2026-11-07 section 3). usageLimitRetryingTurn asks for each of them
// anywhere in the line and not for their order: a narrow pane wraps the
// line in two columns and puts the words of the two columns in turn
// (`✻`, `Weekly    · Retrying in 29m (Nov 21 at 3pm) · attempt`,
// `li…      1/55` on the screen of issue #353).
var usageLimitRetryWords = []string{"Retrying in", "attempt"}

// usageLimitResetsWord comes before the reset time after claude's usage
// limit sentences (`You've hit your weekly limit · resets Nov 21 at 3pm`).
const usageLimitResetsWord = "resets "

// usageLimitRetryingTurn is the third way of UsageLimitScreen (plan
// docs/plans/archive/2026-11-20-watch-usage-limit-retry.md), which
// UsageLimitScreen calls for claude only. It returns the text of a block of
// the current turn from the glyph before a usage limit sentence to the
// block's end, when all of these hold:
//
//   - the last block above driver's input box in screen holds every word of
//     usageLimitRetryWords. ClassifyResumeScreen reads that block first for
//     a waiting sentence, and `Retrying in` is one, so a waiting screen with
//     such a block is waiting because of it;
//   - a block of the current turn (usageLimitTurnBlocks) holds the limit
//     anywhere in it (usageLimitResetsIn): one of table.StopGlyphs, then
//     white space, then one of table.UsageLimit, and later
//     `resets <time>`;
//   - the last block holds `(<time>)` with that time.
//
// When several limits of the turn hold, it returns the last one. A last
// block that is a prompt block leaves no block in the turn, so it is false
// then. It is false when screen has no input box (UsageLimitScreen never
// gets here then: ClassifyResumeScreen read the screen as waiting) or when
// any of the above fails.
func usageLimitRetryingTurn(driver, screen string, table resumeSentenceTable) (string, bool) {
	lines := resumeScreenLines(screen)
	box, reason := findResumeInputBox(driver, lines)
	if reason != "" {
		return "", false
	}
	above := resumeScreenBlocks(lines[:box.aboveEnd], resumeMarker(driver))
	if len(above) == 0 {
		return "", false
	}
	retry := above[len(above)-1]
	for _, w := range usageLimitRetryWords {
		if !strings.Contains(retry.text, w) {
			return "", false
		}
	}
	found := ""
	for _, b := range usageLimitTurnBlocks(above) {
		for _, limit := range usageLimitResetsIn(b.text, table) {
			if strings.Contains(retry.text, "("+limit.reset+")") {
				found = limit.text
			}
		}
	}
	return found, found != ""
}

// usageLimitTurnBlocks returns the blocks of the current turn among above,
// the blocks above the input box: those after the last prompt block, or all
// of them when none is a prompt block (the prompt scrolled off the screen).
func usageLimitTurnBlocks(above []resumeBlock) []resumeBlock {
	for i := len(above) - 1; i >= 0; i-- {
		if above[i].prompt {
			return above[i+1:]
		}
	}
	return above
}

// usageLimitReset is one usage limit usageLimitResetsIn found in a block.
type usageLimitReset struct {
	// text is the block's text from the glyph before the limit sentence to
	// the block's end.
	text string
	// reset is the time after `resets `, up to the first ` (` (the time
	// zone's parenthesis) or the block's end, trimmed: `Nov 21 at 3pm`.
	reset string
}

// usageLimitResetsIn returns, in the order they appear in text (the text of
// one block), the places where one of table.StopGlyphs is followed by white
// space (unicode.IsSpace, so U+00A0 counts, as in findResumeStop), then by
// one of table.UsageLimit, and later in text by `resets ` and a time that is
// not empty. The glyph may be anywhere in text, not only at its start: the
// limit of issue #353 is a ⎿ line inside the block of a tool call
// (`⏺ Monitor "..." stream ended`, then
// `⎿  You've hit your weekly limit · resets Nov 21 at 3pm (Asia/Tokyo)`).
func usageLimitResetsIn(text string, table resumeSentenceTable) []usageLimitReset {
	var out []usageLimitReset
	for i := range text {
		for _, g := range table.StopGlyphs {
			rest, ok := strings.CutPrefix(text[i:], g)
			if !ok {
				continue
			}
			head := strings.TrimLeftFunc(rest, unicode.IsSpace)
			if head == rest {
				continue
			}
			for _, s := range table.UsageLimit {
				after, ok := strings.CutPrefix(head, s)
				if !ok {
					continue
				}
				if reset, ok := usageLimitResetTime(after); ok {
					out = append(out, usageLimitReset{text: text[i:], reset: reset})
				}
				break
			}
		}
	}
	return out
}

// usageLimitResetTime returns the time after the first `resets ` in after,
// the text that follows a usage limit sentence: up to the first ` (` or the
// end of after, trimmed. It is false when after holds no `resets ` or the
// time is empty.
func usageLimitResetTime(after string) (string, bool) {
	_, reset, ok := strings.Cut(after, usageLimitResetsWord)
	if !ok {
		return "", false
	}
	if j := strings.Index(reset, " ("); j >= 0 {
		reset = reset[:j]
	}
	reset = strings.TrimSpace(reset)
	return reset, reset != ""
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
