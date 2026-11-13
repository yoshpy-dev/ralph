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
//     `  check your internet or DNS (ENOTFOUND)`) reads as one line. Two
//     kinds of line start a new block inside such a run. A prompt line, a
//     line whose trimmed text is the driver's marker (step 2) and then
//     white space (`❯ say hi`, `› say hi`), starts a prompt block: a
//     message the user sent, its wrapped lines included. And after a
//     prompt line, the first line whose trimmed text starts with ⎿ starts
//     a block of its own; the ⎿ lines after it join that block. Claude Code
//     2.1.312 prints a usage limit error as such a ⎿ line right under the
//     prompt, with no blank line between them
//     (docs/evidence/org-seat-resume-live-2026-11-07.md, sections 0 and 1).
//  2. The input box. claude: the last line whose trimmed text starts with ❯.
//     It is the input box only when the line right above it is a rule (a
//     line of nothing but ─, U+2500, at least 10 of them) and a rule follows
//     below it; the lines from the ❯ line to that lower rule are the input,
//     and the lines after the lower rule are the footer
//     (`🤖 Haiku 5.5 | ⚡ medium`, `⏵⏵ bypass permissions on ...`). codex:
//     the last line whose trimmed text starts with › (U+203A). The footer
//     is the last run of non-blank lines on the screen when a blank line
//     separates it from the › line, as in the evidence (section 4), and
//     every line of the run, trimmed, has the shape of a footer line
//     (resumeCodexFooterLine, anchored at the start of the line, and the
//     two context forms also at its end): a status line with
//     `Context <n>% used` or `Context <n>% left` after another field, at
//     its end or before more fields
//     (`GPT-6-Astra · xhigh · Context 0% used`,
//     `GPT-6-Luna · low · Context 7% used · main · No chan…`), a line that is
//     `<n>% context left`, alone or as the last field after ` · `,
//     optionally followed by ` · ? for shortcuts`, or a hint line that
//     starts with `? for shortcuts`. The lines from the › line down to that run are
//     the input, blank ones included. With no such run (a run with any
//     other line in it is not one), every line after the › line is input:
//     a draft below the › line with no footer under it is input, and so is
//     a footer line of another shape, which makes the input box hold text.
//     Both markers also start the user's past messages in the history
//     (`❯ say hi`, `› say hi`) and the cursor line of a choice dialog, so
//     only the last marker line counts: a dialog drawn below an old claude
//     input box makes the screen unknown instead of letting the input box
//     above it count. A marker line whose text is a numbered option
//     (`› 1. Review hooks`, `❯ 1. Yes, I trust this folder`) is a choice
//     dialog. No input box: unknown.
//  3. An empty input box: nothing after the marker on the marker line, and
//     every other line of the input blank. For codex the marker line may
//     instead hold exactly one of the composer's placeholder texts
//     (codexComposerPlaceholders), which a plain-text read cannot tell from
//     typed text. Any other text, a draft whose first line is empty
//     included: unknown.
//  4. The checks, once the input box is found and empty. The last block
//     above the input box is taken; for claude, when it is a status line
//     (`✻ Worked for 1s · done 4:33 AM`, `✻ Brewed for 1s · done 4:34 AM`),
//     exactly one such block is skipped and the block before it is the
//     candidate, otherwise the last block is the candidate. Then, in order:
//     (a) an enter-only sentence in the last block, the candidate, or a
//     footer block: enter_only; (b) a waiting sentence in the same places:
//     waiting; (c) a stopped sentence of the driver at the start of the
//     candidate, which counts only when the candidate's text starts with
//     one of the driver's StopGlyphs (claude ⏺ and ⎿, codex ■ and ⚠), then
//     white space, then the sentence: stopped, with the candidate as Line;
//     (d) otherwise unknown. The glyphs do not mark an error only: claude's
//     ⏺ starts every reply and its ⎿ every line printed under a prompt, so
//     a reply whose first words are a stopped sentence (`⏺ API Error: ...`
//     as the reply's own words) is still stopped. The sentences of (a) and
//     (b) count anywhere in their blocks, as they never make resume type
//     text. A stopped sentence later in the candidate (a reply or a tool
//     result that quotes `API Error:`) does not count, and neither does
//     one that starts a candidate with no glyph: a later paragraph of a
//     reply or of a tool result, which a blank line sets apart as a block
//     of its own (`⏺ Documented the expected failure:`, a blank line,
//     `API Error: Connection refused`). A prompt block is read by none of
//     (a) to (c), so a sentence the user typed or quoted never counts;
//     when the candidate is a prompt block, the reason of (d) says so.
//     Blocks further up the history never count, so an old API error
//     above a normal reply does not make the screen stopped.
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
	resumeReasonPromptLast    = "the last block above the input box is a user prompt"
	resumeReasonWaiting       = "the CLI is retrying or waiting"
)

// resumeSentence is one known screen sentence. A stopped sentence counts
// only at the start of the candidate block, right after one of the
// driver's StopGlyphs and white space (step 4 (c) of the screen model):
// `⏺ API Error: ...`, `⎿  You've hit your ...`, and
// `■ You've hit your usage limit ...` match; a block that quotes the
// sentence after other text does not, and neither does a block that starts
// with the sentence but with no glyph. An enter-only or waiting sentence is
// matched as a substring of a block's joined text, anywhere in it. Source
// says where it was found: "seen: <CLI version>, <evidence>" for a real
// screen, "docs: <URL>" for the official docs, or
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
	// StopGlyphs are the glyphs a stopped sentence was seen right after on
	// a real screen (claude) or is expected after (codex). A stopped
	// sentence counts only in a block that starts with one of them, then
	// white space, then the sentence; a block that starts with none of them
	// never holds a stop (step 4 (c) of the screen model). They do not mark
	// an error only: claude's ⏺ starts every reply and its ⎿ every line
	// printed under a prompt, so a reply whose first words are a stopped
	// sentence (`⏺ API Error: ...` as the reply's own words) still counts as
	// stopped.
	StopGlyphs []string
	// UsageLimit marks the stopped sentences that say the CLI hit a usage
	// limit, not an API or network error: each is the Text of one entry of
	// Stopped (TestUsageLimitSentences_AreStoppedSentences checks it).
	// ClassifyResumeScreen does not read it; UsageLimitScreen, which the
	// cross-org watch calls, does.
	UsageLimit []string
	// UsageLimitWaiting marks the waiting sentences that say the CLI waits
	// for a usage limit to reset and then goes on by itself, not for an API
	// retry: each is the Text of one entry of Waiting
	// (TestUsageLimitWaitingSentences_AreWaitingSentences checks it).
	// ClassifyResumeScreen does not read it. UsageLimitScreen counts one only
	// in a block of the footer below the input box, which the CLI draws and
	// no reply or tool output reaches; the same sentence in a block above the
	// input box (a reply or a tool output that quotes it) does not count.
	UsageLimitWaiting []string
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
			// "evidence 2026-11-07 live" is
			// docs/evidence/org-seat-resume-live-2026-11-07.md (sections 0
			// and 1), where it was a ⎿ line under the prompt line.
			{"You've hit your weekly limit", "seen: Claude Code 2.1.312, evidence 2026-11-07 live"},
			// "evidence 2026-11-13 claude-5h-limit" is
			// docs/evidence/claude-5h-limit-screen-2026-11-13.md, where it was
			// a ⎿ line under the prompt line, above the reply
			// `⏺ Usage limit reached · continuing automatically at ...`.
			{"You've hit your 5-hour limit", "seen: Claude Code 2.1.316, evidence 2026-11-13 claude-5h-limit"},
			{"You've hit your Opus limit", "docs: https://code.claude.com/docs/en/errors#youve-hit-your-session-limit"},
			{"You've hit your Sonnet limit", "docs: https://code.claude.com/docs/en/errors#youve-hit-your-session-limit"},
		},
		UsageLimit: []string{
			"You've hit your session limit",
			"You've hit your weekly limit",
			"You've hit your 5-hour limit",
			"You've hit your Opus limit",
			"You've hit your Sonnet limit",
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
		// The two waits for a usage limit to reset (docs
		// interactive-mode#wait-for-a-usage-limit-to-reset). Claude Code
		// 2.1.316 printed `Continuing automatically at 3:50pm` in the footer
		// below the input box (evidence 2026-11-13 claude-5h-limit).
		// `Retrying in` and `Waiting for API response` wait for an API retry
		// and are not marked.
		UsageLimitWaiting: []string{"Continuing automatically at", "Continuing shortly"},
		// ⏺ starts an assistant message and ⎿ a line printed under a
		// prompt or a tool call. Both were seen before a stop on a real
		// screen: `⏺ API Error: ...` (evidence 2026-11-07 section 2) and
		// `⎿  You've hit your weekly limit ...` (evidence 2026-11-07 live).
		StopGlyphs: []string{"⏺", "⎿"},
	},
	"codex": {
		Stopped: []resumeSentence{
			{"You've hit your usage limit", "binary strings: codex-cli 0.165.0"},
			{"Usage limit reached", "binary strings: codex-cli 0.165.0"},
		},
		UsageLimit: []string{"You've hit your usage limit", "Usage limit reached"},
		Waiting: []resumeSentence{
			{"Reconnecting... waiting for network", "seen: codex-cli 0.165.0, evidence 2026-11-07 section 4"},
			{"waiting to retry", "binary strings: codex-cli 0.165.0"},
		},
		// ■ starts an error (`■ Conversation interrupted` in the
		// codex-cli 0.165.0 binary strings) and ⚠ a warning (also in those
		// binary strings). • is not one of them: codex starts ordinary
		// items with it, such as events, replies, and tool and agent lines
		// (`• Reconnecting... waiting for network`, evidence 2026-11-07
		// section 4). The glyph before codex's usage limit sentences was
		// not seen on a real screen (both entries come from binary strings
		// only); if codex prints them with no ■ or ⚠, resume classifies
		// the screen unknown and types nothing, the safe side.
		StopGlyphs: []string{"■", "⚠"},
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
	// resumeCodexFooterLine matches a trimmed line of codex's footer under
	// its composer (step 2 of the screen model). The status line
	// `GPT-6-Astra · xhigh · Context 0% used` and the hint line
	// `? for shortcuts` were seen (codex-cli 0.165.0, evidence 2026-11-07
	// section 4). codex-cli 0.166.0 goes on after the context figure with
	// the branch and a summary of the changes, cut at the pane's width
	// (`GPT-6-Luna · low · Context 7% used · main · No chan…`,
	// docs/evidence/org-send-screen-probe-2026-11-12.md section 6).
	// `Context <n>% left` and `<n>% context left` are the other forms of the
	// context figure in the codex-cli 0.165.0 binary strings.
	//
	// Each alternative is anchored at the start of the line, and the two
	// context alternatives (2 and 3) also at its end, so a draft that
	// merely contains a context figure is not a footer line:
	//  1. a hint line starts with `? for shortcuts` (what follows it on the
	//     line is not checked);
	//  2. a status line has `Context <n>% used` or `Context <n>% left` with
	//     at least one other field before it, set off by ` · `, and either
	//     ends there or goes on with more fields after ` · ` (codex-cli
	//     0.166.0);
	//  3. a context line is `<n>% context left`, alone or as the last field
	//     after ` · `, optionally followed by ` · ? for shortcuts`.
	resumeCodexFooterLine = regexp.MustCompile(`^\? for shortcuts|^.+ · Context \d+% (?:used|left)(?: · .+)?$|^(?:.+ · )?\d+% context left(?: · \? for shortcuts)?$`)
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

	marker := resumeMarker(driver)
	above := resumeScreenBlocks(lines[:box.aboveEnd], marker)
	var last, candidate resumeBlock
	if n := len(above); n > 0 {
		last, candidate = above[n-1], above[n-1]
		if driver == "claude" && isClaudeStatusBlock(last.text) {
			candidate = resumeBlock{}
			if n > 1 {
				candidate = above[n-2]
			}
		}
	}
	look := []resumeBlock{last, candidate}
	look = append(look, resumeScreenBlocks(lines[box.footerStart:], marker)...)

	if line, ok := findResumeSentence(look, table.EnterOnly); ok {
		return ResumeScreen{Kind: ResumeScreenEnterOnly, Line: line}
	}
	if line, ok := findResumeSentence(look, table.Waiting); ok {
		return ResumeScreen{Kind: ResumeScreenWaiting, Line: line, Reason: resumeReasonWaiting}
	}
	if line, ok := findResumeStop(candidate, table); ok {
		return ResumeScreen{Kind: ResumeScreenStopped, Line: line}
	}
	if candidate.prompt {
		return resumeUnknown(resumeReasonPromptLast)
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
	lines := resumeScreenLines(screen)
	box, reason := findResumeInputBox(driver, lines)
	if reason != "" {
		return false
	}
	return box.holdsHead(lines, typed)
}

// holdsHead reports whether b holds the first resumeInputHoldsRunes runes of
// typed, the input's lines joined and white space dropped from both sides
// (ResumeInputHolds). It is false for an empty or all-space typed.
func (b resumeInputBox) holdsHead(lines []string, typed string) bool {
	head := []rune(typed)
	if len(head) > resumeInputHoldsRunes {
		head = head[:resumeInputHoldsRunes]
	}
	want := resumeDropSpace(string(head))
	if want == "" {
		return false
	}
	return strings.Contains(resumeDropSpace(b.text(lines)), want)
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
// screen model). A non-empty reason means there is none. The last marker
// line is a choice dialog when its text is a numbered option, before the
// lines around it are read (resumeInputBoxShape).
func findResumeInputBox(driver string, lines []string) (resumeInputBox, string) {
	box, ok := resumeMarkerLineBox(driver, lines)
	if !ok {
		return resumeInputBox{}, resumeReasonNoInputBox
	}
	if resumeNumberedOption.MatchString(box.first) {
		return resumeInputBox{}, resumeReasonChoiceDialog
	}
	return resumeInputBoxShape(driver, lines, box)
}

// resumeMarkerLineBox returns a box with only marker and first set, for the
// last line of lines whose trimmed text starts with driver's marker, and
// false when there is no such line.
func resumeMarkerLineBox(driver string, lines []string) (resumeInputBox, bool) {
	marker := resumeMarker(driver)
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), marker) {
			return resumeInputBox{
				marker: i,
				first:  strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i]), marker)),
			}, true
		}
	}
	return resumeInputBox{}, false
}

// resumeInputBoxShape is the rest of step 2 of the screen model for box, a
// box from resumeMarkerLineBox: codex's input and footer below the › line,
// or the rules claude draws right above the ❯ line and below it. A
// non-empty reason means the lines around the marker line are no input box.
// It does not look at the marker line's text: findResumeInputBox takes a
// numbered option there for a choice dialog first, and send's check before
// Enter calls it for a numbered line that is part of the message it typed
// (sendTypedInputBox).
func resumeInputBoxShape(driver string, lines []string, box resumeInputBox) (resumeInputBox, string) {
	idx := box.marker
	if driver == "codex" {
		box.inputEnd = codexFooterStart(lines, idx)
		box.aboveEnd, box.footerStart = idx, box.inputEnd
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

// codexFooterStart returns the first line of codex's footer below the ›
// line at idx (step 2 of the screen model): the last run of non-blank lines
// on the screen, when a blank line separates it from the › line and
// resumeCodexFooterLine matches every line of the run. Otherwise it returns
// len(lines), and every line after the › line is input.
func codexFooterStart(lines []string, idx int) int {
	end := len(lines)
	for end > idx+1 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	start := end
	for start > idx+1 && strings.TrimSpace(lines[start-1]) != "" {
		start--
	}
	if start == idx+1 {
		return len(lines)
	}
	for _, l := range lines[start:end] {
		if !resumeCodexFooterLine.MatchString(strings.TrimSpace(l)) {
			return len(lines)
		}
	}
	return start
}

// empty reports whether the input box holds no text (step 3 of the screen
// model): the marker line holds nothing after the marker (for codex, or
// exactly one placeholder text), and every other line of the input is
// blank.
func (b resumeInputBox) empty(driver string, lines []string) bool {
	if b.first != "" && (driver != "codex" || !isCodexComposerPlaceholder(b.first)) {
		return false
	}
	for _, l := range lines[b.marker+1 : b.inputEnd] {
		if strings.TrimSpace(l) != "" {
			return false
		}
	}
	return true
}

// isCodexComposerPlaceholder reports whether text, the trimmed text after
// the › of codex's composer line, is one of codexComposerPlaceholders.
func isCodexComposerPlaceholder(text string) bool {
	for _, p := range codexComposerPlaceholders {
		if text == p.Text {
			return true
		}
	}
	return false
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

// resumeMarker is the marker of driver's input box, which also starts the
// user's past messages in the history: ❯ for claude, › (U+203A) for codex.
func resumeMarker(driver string) string {
	if driver == "codex" {
		return "›"
	}
	return "❯"
}

// resumeBlock is one block of a screen (step 1 of the screen model).
type resumeBlock struct {
	// text is the block's lines, each trimmed, joined with single spaces.
	text string
	// prompt is true for a prompt block: one that starts with a prompt line.
	prompt bool
}

// resumeScreenBlocks splits lines into blocks (step 1 of the screen model):
// a blank line ends a block, a prompt line of marker starts one, and so does
// the first line after a prompt line whose trimmed text starts with ⎿.
func resumeScreenBlocks(lines []string, marker string) []resumeBlock {
	var blocks []resumeBlock
	var cur []string
	prompt, afterPrompt := false, false
	flush := func() {
		if len(cur) > 0 {
			blocks = append(blocks, resumeBlock{text: strings.Join(cur, " "), prompt: prompt})
		}
		cur, prompt, afterPrompt = nil, false, false
	}
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" {
			flush()
			continue
		}
		if isResumePromptLine(t, marker) {
			flush()
			prompt, afterPrompt = true, true
		} else if afterPrompt && strings.HasPrefix(t, "⎿") {
			flush()
		}
		cur = append(cur, t)
	}
	flush()
	return blocks
}

// isResumePromptLine reports whether t, a trimmed line, is a prompt line:
// marker and then white space (`❯ say hi`; the space may be U+00A0).
func isResumePromptLine(t, marker string) bool {
	rest, ok := strings.CutPrefix(t, marker)
	return ok && strings.TrimLeftFunc(rest, unicode.IsSpace) != rest
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

// findResumeStop returns the text of block when it starts with one of
// table.StopGlyphs, then white space, then one of table's stopped sentences
// (step 4 (c) of the screen model). An empty block, a prompt block, and a
// block that starts with no glyph of table.StopGlyphs never match.
func findResumeStop(block resumeBlock, table resumeSentenceTable) (string, bool) {
	if block.text == "" || block.prompt {
		return "", false
	}
	head, glyph := "", false
	for _, g := range table.StopGlyphs {
		if rest, ok := strings.CutPrefix(block.text, g); ok {
			head = strings.TrimLeftFunc(rest, unicode.IsSpace)
			glyph = head != rest
			break
		}
	}
	if !glyph {
		return "", false
	}
	for _, s := range table.Stopped {
		if strings.HasPrefix(head, s.Text) {
			return block.text, true
		}
	}
	return "", false
}

// findResumeSentence returns the text of the first of blocks that holds one
// of sentences anywhere in it (the enter-only and waiting sentences). Empty
// blocks and prompt blocks are skipped.
func findResumeSentence(blocks []resumeBlock, sentences []resumeSentence) (string, bool) {
	for _, b := range blocks {
		if b.text == "" || b.prompt {
			continue
		}
		for _, s := range sentences {
			if strings.Contains(b.text, s.Text) {
				return b.text, true
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
