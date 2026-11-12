package org

import (
	"regexp"
	"strings"
)

// The screens of `ralph org send`'s screen fallback (plan
// docs/plans/active/2026-11-12-send-screen-fallback.md, D4 and D8; issue
// #308). When herdr keeps reporting a seat as working, send reads the seat's
// pane and types only when ClassifySendScreen says the CLI waits for input.
// Text typed while a turn runs changes that turn's work
// (docs/evidence/org-send-screen-probe-2026-11-12.md, section 5), so the
// rules ask for positive evidence that the turn ended and fail closed: a
// screen they cannot judge is unknown, and send types only on idle. Like
// ClassifyResumeScreen, it is deterministic and does no I/O.
//
// The screen model is the one of ClassifyResumeScreen (resume_screens.go):
// its lines and blocks, and its input box (findResumeInputBox, with codex's
// footer by resumeCodexFooterLine). The checks, in order:
//
//  1. Choice. The last marker line is a numbered option (resume's choice
//     dialog), or a dialog hint line (`Enter to confirm · Esc to cancel`,
//     `enter continue · esc quit`) is on or below the last marker line
//     (anywhere on the screen when there is no marker line): choice.
//     claude 2.1.316's folder trust has no numbers (evidence
//     section 1, claude-folder-trust-unnumbered.txt), so its hint line is
//     what marks it. With neither and no input box: unknown.
//  2. The input box. claude's `Press up to edit queued messages` in it is a
//     message queued for the running turn (section 5,
//     claude-queued-while-working.txt): busy. Any other text: unknown
//     (input box has text). Placeholders count as empty: codex's composer
//     placeholders (codexComposerPlaceholders, shown while working too) and
//     claude's `Try "…"` of a fresh session (section 1).
//  3. The region: the lines above the input box after the last prompt block
//     (a prompt line and the lines under it, up to a blank line or a ⎿
//     line, as in resume's blocks), or every line above the input box when
//     there is no prompt line. In the region and the footer, resume's
//     enter-only sentences: unknown (press-enter screen); resume's waiting
//     sentences: busy; a busy line or a queued-input sentence: busy.
//  4. Positive evidence that the turn ended. (a) The last block of the
//     region, once trailing tip blocks and codex's `N background terminal
//     running` line are dropped, starts with a turn-end line (claude
//     `✻ Cooked for 3s · done 6:09 PM`, codex `Worked for 4s • 18:17`;
//     section 2): idle. (b) No prompt line above the input box and the
//     CLI's banner there: a fresh session, idle (section 1; herdr reported
//     a fresh codex seat working, section 9). Otherwise unknown: the last
//     block is a prompt, sent with no busy line under it yet (section 6),
//     or no turn-end line follows the last prompt.

// SendScreenKind is what ClassifySendScreen found on a seat's screen.
type SendScreenKind string

const (
	// SendScreenIdle: the CLI ended its turn, or has not started one, and
	// waits at an empty input box. The only kind send types on.
	SendScreenIdle SendScreenKind = "idle"
	// SendScreenBusy: a busy line, a waiting sentence, or queued input shows
	// that a turn runs.
	SendScreenBusy SendScreenKind = "busy"
	// SendScreenChoice: a choice dialog, where Enter picks an option.
	SendScreenChoice SendScreenKind = "choice"
	// SendScreenUnknown: anything the rules cannot judge as idle, busy, or a
	// choice, text in the input box included.
	SendScreenUnknown SendScreenKind = "unknown"
)

// SendScreen is the result of ClassifySendScreen.
type SendScreen struct {
	Kind SendScreenKind
	// Reason is a short English reason, set for every kind.
	Reason string
	// Line is the trimmed screen line (for a sentence, the block, its
	// wrapped lines joined) that decided the kind, when there is one: the
	// turn-end line or the banner for idle, the busy line for busy. For an
	// input box with text or queued input, it is the input's text without
	// the marker.
	Line string
}

// The reasons ClassifySendScreen gives.
const (
	sendScreenReasonUnknownDriver = "unknown driver"
	sendScreenReasonNoInputBox    = "no input box"
	sendScreenReasonChoiceDialog  = "choice dialog"
	sendScreenReasonInputHasText  = "input box has text"
	sendScreenReasonQueuedInput   = "input is queued for the running turn"
	sendScreenReasonBusyLine      = "a busy line is on the screen"
	sendScreenReasonWaiting       = "the CLI is retrying or waiting"
	sendScreenReasonPressEnter    = "the CLI asks for Enter"
	sendScreenReasonPromptLast    = "the last block above the input box is a user prompt"
	sendScreenReasonNoTurnEnd     = "no turn-end line after the last prompt"
	sendScreenReasonTurnEnded     = "the turn ended"
	sendScreenReasonFreshSession  = "a fresh session with no prompt yet"
)

// sendClaudeQueuedInput is what claude 2.1.316 shows in its input box while
// a message waits for the running turn (evidence 2026-11-12 section 5).
const sendClaudeQueuedInput = "Press up to edit queued messages"

// sendCodexQueuedSentences are the texts codex shows above its composer for
// input queued for the running turn. They come from the screen rule
// `screen_working_fallback` of herdr 0.7.5's codex manifest (evidence
// 2026-11-12 section 7) and were not seen on a screen.
var sendCodexQueuedSentences = []string{
	"Queued follow-up inputs",
	"Messages to be submitted after next tool call",
	"Messages to be submitted at end of turn",
}

var (
	// sendClaudeBusyLine matches claude's spinner line at the start of a
	// trimmed line: one of the symbols, a space, one word ending in …, then
	// the end or white space (`✶ Deliberating… (6s · ↓ 100 tokens)`,
	// `✳ Elucidating…`, `· Bunning…`, evidence 2026-11-12 sections 3, 4,
	// and 6). Only the start is matched, so a line the pane wrapped
	// (`✢ Deliberating… (running PreToolUse hooks… 5/10 · 2s`) still counts.
	// The symbols are the ones seen there and those of herdr's
	// `live_turn_working` rule (section 7).
	sendClaudeBusyLine = regexp.MustCompile(`^[*·✢✳✶✻✽] \S+…(?:\s|$)`)
	// sendClaudeTurnEnd matches the line claude prints when a turn ends
	// (`✻ Cooked for 3s · done 6:09 PM`, `✻ Churned for 2m 57s · done 7:48 AM`,
	// section 2). It starts with ✻ like a busy line, but a word and
	// ` for <duration> · done <time>` follow the symbol instead of a word
	// ending in ….
	sendClaudeTurnEnd = regexp.MustCompile(`^[*·✢✳✶✻✽] \S+ for \d[^·]*· done \d`)
	// sendCodexBusyLine matches codex's working line
	// (`• Working (6s • esc to interrupt)`, sections 3 and 6), also cut at
	// the pane's width, and any other header in front of `(<elapsed> •`.
	sendCodexBusyLine = regexp.MustCompile(`^• (?:Working \(|.*\(\d+[hms](?: \d+[hms])* •)`)
	// sendCodexTurnEnd matches the line codex prints when a turn ends
	// (`Worked for 4s • 18:17`, section 2), also inside a rule line
	// (`─ Worked for 4s ─────`).
	sendCodexTurnEnd = regexp.MustCompile(`^(?:─+ ?)?Worked for \d`)
	// sendScreenTrailer matches the first line of a block that may follow a
	// turn-end line: a tip (`⎿  Tip: …`, `└ Tip: …`) or codex's line for a
	// terminal still running in the background
	// (`1 background terminal running · /ps to view · /stop`, section 5).
	sendScreenTrailer = regexp.MustCompile(`^(?:[⎿└] +Tip:|\d+ background terminals? running)`)
	// sendDialogHint matches the hint line of a choice dialog:
	// `Enter to confirm · Esc to cancel` (claude, section 1),
	// `enter continue · esc quit` and `enter continue · esc skip` (codex,
	// section 1), `enter confirm · esc skip` (codex, evidence 2026-11-07).
	sendDialogHint = regexp.MustCompile(`(?i)^enter (?:to )?(?:confirm|continue|select)\b.* · esc\b`)
	// sendClaudePlaceholder matches the placeholder of claude's empty input
	// box in a fresh session (`Try "how do I log an error?"`, section 1).
	sendClaudePlaceholder = regexp.MustCompile(`^Try "[^"]+"$`)
	// sendClaudeBanner and sendCodexBanner match the banner a CLI prints
	// when a session starts (`Claude Code v2.1.316`,
	// `>_ OpenAI Codex (v0.166.0)`, section 1).
	sendClaudeBanner = regexp.MustCompile(`Claude Code v\d`)
	sendCodexBanner  = regexp.MustCompile(`OpenAI Codex \(v\d`)
)

// ClassifySendScreen classifies screen, the text of a pane of a seat whose
// driver is driver ("claude" or "codex"; any other value is unknown), by the
// checks above. Send types only on SendScreenIdle.
func ClassifySendScreen(driver, screen string) SendScreen {
	table, ok := resumeScreenSentences[driver]
	if !ok {
		return sendScreenUnknown(sendScreenReasonUnknownDriver, "")
	}
	lines := resumeScreenLines(screen)
	box, reason := findResumeInputBox(driver, lines)
	if s, decided := sendScreenBoxCheck(driver, lines, box, reason); decided {
		return s
	}
	above := lines[:box.aboveEnd]
	region, prompt := sendScreenRegion(resumeMarker(driver), above)
	if s, decided := sendScreenBusyCheck(driver, table, region, lines[box.footerStart:]); decided {
		return s
	}
	return sendScreenTurnCheck(driver, above, region, prompt)
}

// SendScreenSubmitted reports whether screen, read after send pressed Enter
// on the screen path, shows that the CLI took typed (plan D8): the input box
// is found and does not hold typed (ResumeInputHolds), and it is empty
// (placeholders and claude's queued-input text included) or a busy line, a
// waiting sentence, or queued input is under the last prompt block. It is
// false when typed is still in the input box, when there is no input box or
// a choice dialog, for an unknown driver or an all-space typed, and when the
// input box holds other text and nothing shows a turn.
func SendScreenSubmitted(driver, screen, typed string) bool {
	table, ok := resumeScreenSentences[driver]
	if !ok || resumeDropSpace(typed) == "" {
		return false
	}
	lines := resumeScreenLines(screen)
	box, reason := findResumeInputBox(driver, lines)
	if reason != "" || ResumeInputHolds(driver, screen, typed) {
		return false
	}
	if sendInputBoxEmpty(driver, lines, box) || sendClaudeInputQueued(driver, lines, box) {
		return true
	}
	region, _ := sendScreenRegion(resumeMarker(driver), lines[:box.aboveEnd])
	s, _ := sendScreenBusyCheck(driver, table, region, lines[box.footerStart:])
	return s.Kind == SendScreenBusy
}

// sendScreenBoxCheck runs checks 1 and 2: a choice dialog, no input box, or
// an input box that is not empty decides the screen.
func sendScreenBoxCheck(driver string, lines []string, box resumeInputBox, reason string) (SendScreen, bool) {
	last := sendLastMarkerLine(lines, resumeMarker(driver))
	if reason == resumeReasonChoiceDialog {
		return SendScreen{Kind: SendScreenChoice, Reason: sendScreenReasonChoiceDialog, Line: strings.TrimSpace(lines[last])}, true
	}
	if line, ok := sendDialogHintLine(lines[max(last, 0):]); ok {
		return SendScreen{Kind: SendScreenChoice, Reason: sendScreenReasonChoiceDialog, Line: line}, true
	}
	if reason != "" {
		return sendScreenUnknown(sendScreenReasonNoInputBox, ""), true
	}
	if sendClaudeInputQueued(driver, lines, box) {
		return SendScreen{Kind: SendScreenBusy, Reason: sendScreenReasonQueuedInput, Line: box.first}, true
	}
	if !sendInputBoxEmpty(driver, lines, box) {
		return sendScreenUnknown(sendScreenReasonInputHasText, strings.TrimSpace(box.text(lines))), true
	}
	return SendScreen{}, false
}

// sendScreenBusyCheck runs check 3 on region and footer: a press-enter
// screen, a waiting sentence, a busy line, or queued input decides the
// screen.
func sendScreenBusyCheck(driver string, table resumeSentenceTable, region, footer []string) (SendScreen, bool) {
	marker := resumeMarker(driver)
	blocks := append(resumeScreenBlocks(region, marker), resumeScreenBlocks(footer, marker)...)
	if line, ok := findResumeSentence(blocks, table.EnterOnly); ok {
		return SendScreen{Kind: SendScreenUnknown, Reason: sendScreenReasonPressEnter, Line: line}, true
	}
	if line, ok := findResumeSentence(blocks, table.Waiting); ok {
		return SendScreen{Kind: SendScreenBusy, Reason: sendScreenReasonWaiting, Line: line}, true
	}
	for _, part := range [][]string{region, footer} {
		for _, l := range part {
			t := strings.TrimSpace(l)
			if isSendBusyLine(driver, t) {
				return SendScreen{Kind: SendScreenBusy, Reason: sendScreenReasonBusyLine, Line: t}, true
			}
			if isSendQueuedLine(driver, t) {
				return SendScreen{Kind: SendScreenBusy, Reason: sendScreenReasonQueuedInput, Line: t}, true
			}
		}
	}
	return SendScreen{}, false
}

// sendScreenTurnCheck runs check 4 once nothing busy is in region: a
// turn-end line as the region's last block, or a fresh session, is idle;
// anything else is unknown.
func sendScreenTurnCheck(driver string, above, region []string, prompt bool) SendScreen {
	firsts := sendScreenBlockFirstLines(region)
	for len(firsts) > 0 && sendScreenTrailer.MatchString(firsts[len(firsts)-1]) {
		firsts = firsts[:len(firsts)-1]
	}
	lastBlock := ""
	if n := len(firsts); n > 0 {
		lastBlock = firsts[n-1]
	}
	if isSendTurnEndLine(driver, lastBlock) {
		return SendScreen{Kind: SendScreenIdle, Reason: sendScreenReasonTurnEnded, Line: lastBlock}
	}
	if prompt && lastBlock == "" {
		return sendScreenUnknown(sendScreenReasonPromptLast, "")
	}
	if !prompt {
		if line, ok := sendBannerLine(driver, above); ok {
			return SendScreen{Kind: SendScreenIdle, Reason: sendScreenReasonFreshSession, Line: line}
		}
	}
	return sendScreenUnknown(sendScreenReasonNoTurnEnd, lastBlock)
}

// sendScreenRegion returns the lines of above after the last prompt block,
// and whether above has a prompt line. The prompt block is the last prompt
// line of marker and the lines under it up to a blank line or a line that
// starts with ⎿, which resume's blocks also set apart (step 1 of its screen
// model). With no prompt line, the region is all of above.
func sendScreenRegion(marker string, above []string) ([]string, bool) {
	for i := len(above) - 1; i >= 0; i-- {
		if !isResumePromptLine(strings.TrimSpace(above[i]), marker) {
			continue
		}
		j := i + 1
		for j < len(above) {
			t := strings.TrimSpace(above[j])
			if t == "" || strings.HasPrefix(t, "⎿") {
				break
			}
			j++
		}
		return above[j:], true
	}
	return above, false
}

// sendScreenBlockFirstLines returns the first line, trimmed, of each run of
// non-blank lines in lines.
func sendScreenBlockFirstLines(lines []string) []string {
	var firsts []string
	inBlock := false
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t != "" && !inBlock {
			firsts = append(firsts, t)
		}
		inBlock = t != ""
	}
	return firsts
}

// sendInputBoxEmpty reports whether box holds no text: resume's rule (step 3
// of its screen model, which takes codex's placeholders as empty), or, for
// claude, a box whose marker line holds the `Try "…"` placeholder and whose
// other lines are blank.
func sendInputBoxEmpty(driver string, lines []string, box resumeInputBox) bool {
	if box.empty(driver, lines) {
		return true
	}
	return driver == "claude" && sendInputBoxHoldsOnly(driver, lines, box, sendClaudePlaceholder.MatchString)
}

// sendClaudeInputQueued reports whether box is claude's input box holding
// only sendClaudeQueuedInput.
func sendClaudeInputQueued(driver string, lines []string, box resumeInputBox) bool {
	return driver == "claude" && sendInputBoxHoldsOnly(driver, lines, box, func(t string) bool { return t == sendClaudeQueuedInput })
}

// sendInputBoxHoldsOnly reports whether match accepts the text on box's
// marker line and every other line of the input is blank.
func sendInputBoxHoldsOnly(driver string, lines []string, box resumeInputBox, match func(string) bool) bool {
	if !match(box.first) {
		return false
	}
	rest := box
	rest.first = ""
	return rest.empty(driver, lines)
}

// sendLastMarkerLine returns the index of the last line whose trimmed text
// starts with marker (the line findResumeInputBox starts from), or -1.
func sendLastMarkerLine(lines []string, marker string) int {
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), marker) {
			return i
		}
	}
	return -1
}

// sendDialogHintLine returns the first line of lines, trimmed, that is a
// choice dialog's hint line (sendDialogHint).
func sendDialogHintLine(lines []string) (string, bool) {
	for _, l := range lines {
		if t := strings.TrimSpace(l); sendDialogHint.MatchString(t) {
			return t, true
		}
	}
	return "", false
}

// isSendBusyLine reports whether t, a trimmed line, shows that a turn runs:
// driver's busy line, or `esc to interrupt` anywhere in it (codex's working
// line has it, and herdr's claude rules count a line with it as working,
// evidence 2026-11-12 section 7).
func isSendBusyLine(driver, t string) bool {
	if strings.Contains(t, "esc to interrupt") {
		return true
	}
	if driver == "claude" {
		return sendClaudeBusyLine.MatchString(t)
	}
	return sendCodexBusyLine.MatchString(t)
}

// isSendQueuedLine reports whether t, a trimmed line of a codex screen,
// holds one of sendCodexQueuedSentences.
func isSendQueuedLine(driver, t string) bool {
	if driver != "codex" {
		return false
	}
	for _, s := range sendCodexQueuedSentences {
		if strings.Contains(t, s) {
			return true
		}
	}
	return false
}

// isSendTurnEndLine reports whether t, the first line of a block, is
// driver's turn-end line.
func isSendTurnEndLine(driver, t string) bool {
	if driver == "claude" {
		return sendClaudeTurnEnd.MatchString(t)
	}
	return sendCodexTurnEnd.MatchString(t)
}

// sendBannerLine returns the first line of lines, trimmed, that holds
// driver's banner.
func sendBannerLine(driver string, lines []string) (string, bool) {
	banner := sendClaudeBanner
	if driver == "codex" {
		banner = sendCodexBanner
	}
	for _, l := range lines {
		if t := strings.TrimSpace(l); banner.MatchString(t) {
			return t, true
		}
	}
	return "", false
}

// sendScreenUnknown is an unknown screen with reason and line.
func sendScreenUnknown(reason, line string) SendScreen {
	return SendScreen{Kind: SendScreenUnknown, Reason: reason, Line: line}
}
