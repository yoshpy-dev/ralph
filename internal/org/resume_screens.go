package org

import (
	"regexp"
	"strings"
	"unicode"
)

// The screens of `ralph org resume` (plan 2026-11-07-org-seat-resume, S4;
// spec docs/specs/2026-11-07-org-seat-resume.md, FR-2 step 4 and FR-3).
// ClassifyResumeScreen reads the text of a seat's pane, as
// `herdr pane read --source recent --lines 60 --format text` prints it, and
// says whether the CLI in it ended its turn on an API error (type the resume
// sentence), waits for Enter after a usage limit reset (press Enter only),
// is still retrying or waiting (do nothing), or shows anything else (do
// nothing). It is deterministic and does no I/O: the sentence tables below
// and the screen model are all it reads. When a CLI release changes one of
// these sentences, this file is the one to change.
//
// The screen model, taken from the real screens in
// docs/evidence/org-seat-resume-probe-2026-11-07.md (sections 2 to 4):
//
//  1. Lines and blocks. The screen is split into lines, each without its
//     trailing white space (a "\r" included). A block is a run of non-blank
//     lines; its text is its lines, each trimmed, joined with single
//     spaces, so a sentence the pane width wrapped
//     (`⏺ API Error: Can't reach the API server —` and
//     `  check your internet or DNS (ENOTFOUND)`) reads as one line.
//  2. The input box. claude: the last line whose trimmed text starts with ❯.
//     It is the input box only when the line right above it is a rule (a
//     line of nothing but ─, U+2500, at least 10 of them) and a rule follows
//     below it; the lines from the ❯ line to that lower rule are the input,
//     and the lines after the lower rule are the footer
//     (`🤖 Haiku 5.5 | ⚡ medium`, `⏵⏵ bypass permissions on ...`). codex:
//     the last line whose trimmed text starts with › (U+203A); that line
//     and the non-blank lines right below it are the input, and the lines
//     after the › line are the footer
//     (`GPT-6-Astra · xhigh · Context 0% used`, `? for shortcuts`).
//     Both markers also start the user's past messages in the history
//     (`❯ say hi`, `› say hi`) and the cursor line of a choice dialog, so
//     only the last marker line counts: a dialog drawn below an old claude
//     input box makes the screen unknown instead of letting the input box
//     above it count. A marker line whose text is a numbered option
//     (`› 1. Review hooks`, `❯ 1. Yes, I trust this folder`) is a choice
//     dialog. No input box: unknown.
//  3. An empty input box. claude: nothing after ❯ and no other text before
//     the lower rule. codex: nothing after ›, or exactly one of the
//     composer's placeholder texts (codexComposerPlaceholders), which a
//     plain-text read cannot tell from typed text. Any other text: unknown.
//  4. The checks, once the input box is found and empty. The last block
//     above the input box is taken; for claude, when it is a status line
//     (`✻ Worked for 1s · done 4:33 AM`, `✻ Brewed for 1s · done 4:34 AM`),
//     exactly one such block is skipped and the block before it is the
//     candidate, otherwise the last block is the candidate. Then, in order:
//     (a) an enter-only sentence in the last block, the candidate, or a
//     footer block: enter_only; (b) a waiting sentence in the same places:
//     waiting; (c) a stopped sentence of the driver in the candidate:
//     stopped, with the candidate as Line; (d) otherwise unknown. Blocks
//     further up the history never count, so an old API error above a
//     normal reply does not make the screen stopped.
//
// A choice or approval dialog whose cursor line carries the marker (codex's
// hook review in the evidence, section 4, does) never classifies as stopped
// or enter_only: its cursor line is the last marker line and has text after
// the marker.

// ResumeScreenKind is what ClassifyResumeScreen found on a seat's screen.
type ResumeScreenKind string

const (
	// ResumeScreenStopped: the CLI ended its turn on an API error and waits
	// at an empty input box. Type the resume sentence.
	ResumeScreenStopped ResumeScreenKind = "stopped"
	// ResumeScreenEnterOnly: the CLI asks for Enter after a usage limit
	// reset. Press Enter once and type nothing.
	ResumeScreenEnterOnly ResumeScreenKind = "enter_only"
	// ResumeScreenWaiting: the CLI is retrying or waiting on its own. Do
	// nothing.
	ResumeScreenWaiting ResumeScreenKind = "waiting"
	// ResumeScreenUnknown: anything else, a dialog or a normal reply
	// included. Do nothing.
	ResumeScreenUnknown ResumeScreenKind = "unknown"
)

// ResumeScreen is the result of ClassifyResumeScreen.
type ResumeScreen struct {
	Kind ResumeScreenKind
	// Line is the block that held the matched sentence, its wrapped lines
	// joined into one line, for the result line and the manifest event. It
	// is set for stopped, enter_only, and waiting, and empty for unknown.
	Line string
	// Reason is a short English reason, set for waiting and unknown.
	Reason string
}

// The reasons ClassifyResumeScreen gives.
const (
	resumeReasonUnknownDriver = "unknown driver"
	resumeReasonNoInputBox    = "no input box"
	resumeReasonChoiceDialog  = "choice dialog"
	resumeReasonInputHasText  = "input box has text"
	resumeReasonNoStop        = "no known stop sentence above the input box"
	resumeReasonWaiting       = "the CLI is retrying or waiting"
)

// resumeSentence is one known screen sentence. Text is matched as a
// substring of a block's joined text, so a leading `⏺ `, `⎿ `, `■ `, or no
// prefix all match. Source says where it was found: "seen: <CLI version>,
// <evidence>" for a real screen, "docs: <URL>" for the official docs, or
// "binary strings: <CLI version>" for a sentence found only in the CLI's
// binary (TestResumeScreenSentences_EveryEntryHasASource checks the form).
type resumeSentence struct {
	Text   string
	Source string
}

// resumeSentenceTable is one driver's sentences, by kind.
type resumeSentenceTable struct {
	Stopped   []resumeSentence
	EnterOnly []resumeSentence
	Waiting   []resumeSentence
}

// resumeScreenSentences holds the sentences of each driver ralph spawns
// seats with. A driver missing here classifies as unknown.
var resumeScreenSentences = map[string]resumeSentenceTable{
	"claude": {
		Stopped: []resumeSentence{
			// Covers the network errors (`Can't reach the API server`,
			// `Unable to connect to API`, `Connection refused`,
			// `No internet route`, `Connection dropped`, `Request timed out`)
			// and `Repeated 529 Overloaded errors`, `No response from API`.
			{"API Error:", "seen: Claude Code 2.1.312 (ENOTFOUND), evidence 2026-11-07 section 2; the other messages docs: https://code.claude.com/docs/en/errors"},
			{"Connection lost before a response was produced", "docs: https://code.claude.com/docs/en/errors#automatic-retries"},
			{"You've hit your session limit", "docs: https://code.claude.com/docs/en/errors#youve-hit-your-session-limit"},
			{"You've hit your weekly limit", "docs: https://code.claude.com/docs/en/errors#youve-hit-your-session-limit"},
			{"You've hit your Opus limit", "docs: https://code.claude.com/docs/en/errors#youve-hit-your-session-limit"},
			{"You've hit your Sonnet limit", "docs: https://code.claude.com/docs/en/errors#youve-hit-your-session-limit"},
		},
		EnterOnly: []resumeSentence{
			// Shown with `Your usage limit has reset` when the limit reset
			// while the computer slept for more than about 30 minutes.
			{"Press enter to continue", "docs: https://code.claude.com/docs/en/interactive-mode#wait-for-a-usage-limit-to-reset"},
		},
		Waiting: []resumeSentence{
			{"Retrying in", "seen: Claude Code 2.1.312, evidence 2026-11-07 section 3"},
			// Case-sensitive on purpose: the line the conversation records
			// when a wait starts reads `continuing automatically at` in
			// lower case and stays in the history after the wait ends.
			{"Continuing automatically at", "docs: https://code.claude.com/docs/en/interactive-mode#wait-for-a-usage-limit-to-reset"},
			{"Continuing shortly", "docs: https://code.claude.com/docs/en/interactive-mode#wait-for-a-usage-limit-to-reset"},
			{"Waiting for API response", "docs: https://code.claude.com/docs/en/errors#waiting-for-api-response"},
		},
	},
	"codex": {
		Stopped: []resumeSentence{
			{"You've hit your usage limit", "binary strings: codex-cli 0.165.0"},
			{"Usage limit reached", "binary strings: codex-cli 0.165.0"},
		},
		Waiting: []resumeSentence{
			{"Reconnecting... waiting for network", "seen: codex-cli 0.165.0, evidence 2026-11-07 section 4"},
			{"waiting to retry", "binary strings: codex-cli 0.165.0"},
		},
	},
}

// codexComposerPlaceholders are the grey texts codex's empty composer shows
// after ›. Plain text cannot tell them from typed text, so a › line that
// holds exactly one of them is an empty input box. They are PLACEHOLDER
// and SIDE_PLACEHOLDER of codex-rs/tui/src/chatwidget.rs on codex's main
// branch (2026-11-07). The example prompts older codex releases rotated
// through (`Explain this codebase`, `Summarize recent commits`, ...) are
// not in the codex-cli 0.165.0 binary.
var codexComposerPlaceholders = []resumeSentence{
	{"Ask Codex to do anything", "seen: codex-cli 0.165.0, evidence 2026-11-07 section 4"},
	{"Ask a follow-up question", "binary strings: codex-cli 0.165.0"},
}

// resumeInputHoldsRunes is how many runes of the typed text ResumeInputHolds
// looks for in the input box.
const resumeInputHoldsRunes = 12

// resumeRuleMinRunes is the shortest line of ─ that counts as the rule
// above or below claude's input box.
const resumeRuleMinRunes = 10

var (
	// resumeNumberedOption matches the text of a choice dialog's cursor
	// line after its marker (`1. Review hooks`).
	resumeNumberedOption = regexp.MustCompile(`^\d+\.\s`)
	// resumeClaudeWorkedFor matches a claude status line that names how
	// long the turn took (`✻ Worked for 1s`, `✻ Brewed for 2m 5s`).
	resumeClaudeWorkedFor = regexp.MustCompile(`^✻ \S+ for \d`)
)

// ClassifyResumeScreen classifies screen, the text of a pane of a seat whose
// driver is driver ("claude" or "codex"; any other value is unknown), with
// the screen model and the sentence tables above.
func ClassifyResumeScreen(driver, screen string) ResumeScreen {
	table, ok := resumeScreenSentences[driver]
	if !ok {
		return resumeUnknown(resumeReasonUnknownDriver)
	}
	lines := resumeScreenLines(screen)
	box, reason := findResumeInputBox(driver, lines)
	if reason != "" {
		return resumeUnknown(reason)
	}
	if !box.empty(driver, lines) {
		return resumeUnknown(resumeReasonInputHasText)
	}

	above := resumeScreenBlocks(lines[:box.aboveEnd])
	var last, candidate string
	if n := len(above); n > 0 {
		last, candidate = above[n-1], above[n-1]
		if driver == "claude" && isClaudeStatusBlock(last) {
			candidate = ""
			if n > 1 {
				candidate = above[n-2]
			}
		}
	}
	look := []string{last, candidate}
	look = append(look, resumeScreenBlocks(lines[box.footerStart:])...)

	if line, ok := findResumeSentence(look, table.EnterOnly); ok {
		return ResumeScreen{Kind: ResumeScreenEnterOnly, Line: line}
	}
	if line, ok := findResumeSentence(look, table.Waiting); ok {
		return ResumeScreen{Kind: ResumeScreenWaiting, Line: line, Reason: resumeReasonWaiting}
	}
	if line, ok := findResumeSentence([]string{candidate}, table.Stopped); ok {
		return ResumeScreen{Kind: ResumeScreenStopped, Line: line}
	}
	return resumeUnknown(resumeReasonNoStop)
}

// ResumeInputHolds reports whether the input box of screen (found by the
// same rules as ClassifyResumeScreen, step 2) holds the first 12 runes of
// typed. The input's lines are joined and white space is dropped from both
// sides before the comparison, so a wrap at a space and a wrap in the
// middle of a word (Japanese text) both match. It is false for an unknown
// driver, a screen with no input box or a choice dialog, and an empty or
// all-space typed.
func ResumeInputHolds(driver, screen, typed string) bool {
	if _, ok := resumeScreenSentences[driver]; !ok {
		return false
	}
	head := []rune(typed)
	if len(head) > resumeInputHoldsRunes {
		head = head[:resumeInputHoldsRunes]
	}
	want := resumeDropSpace(string(head))
	if want == "" {
		return false
	}
	lines := resumeScreenLines(screen)
	box, reason := findResumeInputBox(driver, lines)
	if reason != "" {
		return false
	}
	return strings.Contains(resumeDropSpace(box.text(lines)), want)
}

// resumeInputBox is where findResumeInputBox found the input box in a
// screen's lines.
type resumeInputBox struct {
	marker      int    // the line that starts with ❯ or ›
	inputEnd    int    // the line after the input's last line
	aboveEnd    int    // the line after the last line above the input box
	footerStart int    // the footer's first line
	first       string // the marker line's text after the marker, trimmed
}

// findResumeInputBox finds the input box of driver in lines (step 2 of the
// screen model). A non-empty reason means there is none.
func findResumeInputBox(driver string, lines []string) (resumeInputBox, string) {
	marker := "❯"
	if driver == "codex" {
		marker = "›"
	}
	idx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), marker) {
			idx = i
			break
		}
	}
	if idx < 0 {
		return resumeInputBox{}, resumeReasonNoInputBox
	}
	box := resumeInputBox{
		marker: idx,
		first:  strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[idx]), marker)),
	}
	if resumeNumberedOption.MatchString(box.first) {
		return resumeInputBox{}, resumeReasonChoiceDialog
	}
	if driver == "codex" {
		box.inputEnd = idx + 1
		for box.inputEnd < len(lines) && strings.TrimSpace(lines[box.inputEnd]) != "" {
			box.inputEnd++
		}
		box.aboveEnd, box.footerStart = idx, idx+1
		return box, ""
	}
	if idx == 0 || !isResumeRule(lines[idx-1]) {
		return resumeInputBox{}, resumeReasonNoInputBox
	}
	lower := -1
	for i := idx + 1; i < len(lines); i++ {
		if isResumeRule(lines[i]) {
			lower = i
			break
		}
	}
	if lower < 0 {
		return resumeInputBox{}, resumeReasonNoInputBox
	}
	box.inputEnd, box.aboveEnd, box.footerStart = lower, idx-1, lower+1
	return box, ""
}

// empty reports whether the input box holds no text (step 3 of the screen
// model).
func (b resumeInputBox) empty(driver string, lines []string) bool {
	if driver == "codex" {
		if b.first == "" {
			return true
		}
		for _, p := range codexComposerPlaceholders {
			if b.first == p.Text {
				return true
			}
		}
		return false
	}
	if b.first != "" {
		return false
	}
	for _, l := range lines[b.marker+1 : b.inputEnd] {
		if strings.TrimSpace(l) != "" {
			return false
		}
	}
	return true
}

// text is the input's lines, the marker dropped, each trimmed and joined
// with single spaces.
func (b resumeInputBox) text(lines []string) string {
	parts := []string{b.first}
	for _, l := range lines[b.marker+1 : b.inputEnd] {
		if t := strings.TrimSpace(l); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ")
}

// resumeUnknown is an unknown screen with reason.
func resumeUnknown(reason string) ResumeScreen {
	return ResumeScreen{Kind: ResumeScreenUnknown, Reason: reason}
}

// resumeScreenLines splits screen into lines without trailing white space.
func resumeScreenLines(screen string) []string {
	lines := strings.Split(screen, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRightFunc(l, unicode.IsSpace)
	}
	return lines
}

// resumeScreenBlocks joins each run of non-blank lines into one line: each
// line trimmed, joined with single spaces.
func resumeScreenBlocks(lines []string) []string {
	var blocks, cur []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" {
			if len(cur) > 0 {
				blocks = append(blocks, strings.Join(cur, " "))
				cur = nil
			}
			continue
		}
		cur = append(cur, t)
	}
	if len(cur) > 0 {
		blocks = append(blocks, strings.Join(cur, " "))
	}
	return blocks
}

// isResumeRule reports whether line is a rule of claude's input box: only
// ─ (U+2500) once trimmed, at least resumeRuleMinRunes of them.
func isResumeRule(line string) bool {
	t := strings.TrimSpace(line)
	n := 0
	for _, r := range t {
		if r != '─' {
			return false
		}
		n++
	}
	return n >= resumeRuleMinRunes
}

// isClaudeStatusBlock reports whether block is the status line claude
// prints when a turn ends (`✻ Worked for 1s · done 4:33 AM`).
func isClaudeStatusBlock(block string) bool {
	if !strings.HasPrefix(block, "✻ ") {
		return false
	}
	return strings.Contains(block, " · done ") || resumeClaudeWorkedFor.MatchString(block)
}

// findResumeSentence returns the first of blocks that holds one of
// sentences. Empty blocks are skipped.
func findResumeSentence(blocks []string, sentences []resumeSentence) (string, bool) {
	for _, b := range blocks {
		if b == "" {
			continue
		}
		for _, s := range sentences {
			if strings.Contains(b, s.Text) {
				return b, true
			}
		}
	}
	return "", false
}

// resumeDropSpace drops every white-space rune from s.
func resumeDropSpace(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}
