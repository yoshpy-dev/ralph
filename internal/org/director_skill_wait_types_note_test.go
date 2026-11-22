package org

// The pins of the /director skill on the inbox waits of issue #374 (plan
// 2026-11-22-director-wait-types-note): run one wait at a time and let
// overlapping waits end on their own, and read the open items a --types wait
// did not count. They share the helpers of director_skill_sync_test.go
// (directorSkillRepoRoot, directorSkillCopies, directorSkillWaitSubsection,
// withoutSpaces, skillOrderedFragmentProblems).

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// directorSkillTakeoverLineStart and directorSkillTakeoverLineVerb are the
// words the /director skill quotes, in backquotes, from the line a director
// wait prints after directorWaitNoticePrefix when another
// `ralph org wait --inbox --director` took its mark over (sayLost):
// "another ralph org wait --inbox --director (wait_id <id>, pid <pid>) took
// over <path>, so this wait (wait_id <id>) does not write it again and keeps
// waiting".
const (
	directorSkillTakeoverLineStart = "another ralph org wait --inbox --director"
	directorSkillTakeoverLineVerb  = "took over"
)

// directorSkillWaitOneAtATimeSteps are fragments of the /director skill's
// subsection ### wait で待つ, in the order the skill has them, each the words
// of one rule plan 2026-11-22-director-wait-types-note.md gave the director
// (issue #374, AC3): the Claude Code bullet waits again after the items an
// exit 0 returned, and it and the Codex bullet run a wait again after its
// timeout, only while no other wait runs, pointing to the paragraph on one
// wait at a time; the exit-code list's item 4 (the watch no longer accepts
// work) does the same for the wait it runs again; its item on the lines that
// start with directorWaitNoticePrefix sends the takeover line to the
// paragraph on overlapping waits; its item 1 runs a timed-out wait again only
// while no other wait runs, as the two bullets do; the paragraph on the
// Monitor's expiry runs the wait again, and again on a later notice of that
// Monitor, only while no other wait runs (the last words of the later
// notice's fragment follow the ones directorSkillWaitSubsectionRules pins in
// director_skill_sync_test.go); the director runs one wait at a
// time and starts a new one only once the end of the previous one was
// reported (by its Monitor, or as the end of a run_in_background task); the
// takeover line does not tell that the newer wait still runs, whose end the
// director learns from its own end notice or, with none, from the wait: line
// of ralph org director status; and once the takeover line shows, it starts
// no further wait, stops neither wait (an older wait stopped by its Monitor
// would end by a signal other than the Monitor's expiry, which the skill
// takes for a failure that ends the director's wait on the inbox), handles
// the items either wait returns without handling one twice, and starts one
// wait only once both ended.
var directorSkillWaitOneAtATimeSteps = []string{
	"件を処理し、また待つ。ほかの wait がまだ走っているあいだは打たない(下の、wait を 1 本ずつ張る段落)",
	"Codex と同じくもう一度打つ。ほかの wait がまだ走っているあいだは打たない(下の、wait を 1 本ずつ張る段落)",
	"来なかっただけなので、もう一度打つ。ほかの wait がまだ走っているあいだは打たない(下の、wait を 1 本ずつ張る段落)",
	"その事実とコマンドを人に伝え、wait を打ち直す。ほかの wait がまだ走っているあいだは打ち直さない(下の、wait を 1 本ずつ張る段落)",
	"印を引き継いだという行(`" + directorSkillTakeoverLineVerb + "` を含む)は wait を重ねたしるしでもあるので",
	"上のとおり打ち直す。ほかの wait がまだ走っているあいだは打ち直さない(下の、wait を 1 本ずつ張る段落)",
	"wait を打ち直し、それからは Monitor の `timeout_ms` に `1800000` を渡す(渡し忘れて既定の 5 分で止められたときなど)。ただし、ほかの wait がまだ走っているあいだは打ち直さない(下の、wait を 1 本ずつ張る段落)",
	"あとからその Monitor の知らせが届いたら、そこで同じように打ち直し(ほかの wait がまだ走っているあいだは打たない)、失敗とした終わりは",
	"wait は 1 本ずつ張る。新しい wait を打つ前に、前の wait が終わったことを確かめる",
	"前の wait を見張っていた Monitor が wait の終わりを知らせていれば(Bash の `run_in_background` で走らせた wait なら、その終わりの知らせが届いていれば)、前の wait は終わっている",
	"知らせがまだなら、前の wait は走っているので、次の wait を打たない",
	"Codex でも、前の wait が返ってから次を打つ",
	"`" + directorSkillTakeoverLineStart + "` で始まり `" + directorSkillTakeoverLineVerb + "` を含む行",
	"wait を重ねている",
	"新しい wait がまだ走っているとは限らない",
	"新しい wait が終わったことは、その wait の終わりの知らせ(見張っていた Monitor の知らせか、`run_in_background` の終わりの知らせ)で分かる",
	"その知らせが自分に届かない wait なら、`ralph org director status` の `wait:` の行がその `pid` で `ended at` を出していれば、終わっている",
	"それ以上 wait を打たない",
	"古い wait の Monitor を止めない",
	"止めた wait の終わりは Monitor の期限切れでない signal での終わりになり",
	"受信箱を待つのをやめることになるため",
	"古い wait は、自分の `--timeout-ms` が過ぎればひとりでに終わる",
	"新しい wait も止めない",
	"どちらの wait も受信箱を読み続け、新しい件が来れば返す",
	"どちらかが件を返したら、件を処理する",
	"同じ件を 2 度処理しない",
	"もう一方が走っているあいだは、次の wait を打たない。2 本とも終わってから、1 本だけ打つ",
}

// directorSkillStopsAWaitRe finds, in a text with spaces and line breaks
// removed (withoutSpaces), the start of words that may tell the director to
// stop a wait or the Monitor that watches it: "Monitor" or "wait", in
// backquotes or not, then を, も, or は, then optionally up to 20 characters
// of one clause (no 。 or 、) that end in で (a tool, as in "Monitor を
// TaskStop で止める"), then the stem of a verb that stops something: 止め,
// 終わらせ, 停止, 打ち切, kill, or TaskStop. directorSkillStopsAWait reads
// what follows the stem.
var directorSkillStopsAWaitRe = regexp.MustCompile("`?(?:Monitor|wait)`?[をもは](?:[^。、]{0,20}?で)?(?:止め|終わらせ|停止|打ち切|kill|TaskStop)")

// directorSkillStopEndings are the starts of the words after such a stem that
// tell the director to stop: the plain, te, continuative, and imperative
// forms of the verb (止める, 止めて, 止め、, 止めろ, 止めよ; 停止する,
// 停止して, 停止し、, 停止しろ; 打ち切る, 打ち切って, 打ち切り、, 打ち切れ).
var directorSkillStopEndings = []string{"る", "て", "、", "ろ", "よ", "する", "して", "し、", "しろ", "って", "り、", "れ"}

// directorSkillNegatedEndings are the starts of the words after such a stem
// that forbid the stop although they begin as directorSkillStopEndings do
// (止めるな, 止めてはいけない, 停止するな, 打ち切ってはならない, and the
// like). 止めない, 停止しない, and 打ち切らない begin with no stop ending.
var directorSkillNegatedEndings = []string{"るな", "てはいけない", "てはならない", "するな", "してはいけない", "してはならない", "ってはいけない", "ってはならない"}

// directorSkillStopsAWait returns the words of text, with spaces and line
// breaks removed, that tell the director to stop a wait or its Monitor: each
// match of directorSkillStopsAWaitRe after which its sentence ends or goes on
// with one of directorSkillStopEndings and none of
// directorSkillNegatedEndings, with the rest of that sentence. It finds
// nothing in the rule "古い wait の Monitor を止めない" nor in the paragraph
// on the Monitor's expiry, "Monitor が自分の `timeout_ms` で wait を止めた
// ので", which says what the Monitor did.
func directorSkillStopsAWait(text string) []string {
	startsWithOneOf := func(s string, starts []string) bool {
		return slices.ContainsFunc(starts, func(start string) bool { return strings.HasPrefix(s, start) })
	}
	var found []string
	for _, loc := range directorSkillStopsAWaitRe.FindAllStringIndex(text, -1) {
		rest, _, _ := strings.Cut(text[loc[1]:], "。")
		if startsWithOneOf(rest, directorSkillNegatedEndings) {
			continue
		}
		if rest == "" || startsWithOneOf(rest, directorSkillStopEndings) {
			found = append(found, text[loc[0]:loc[1]]+rest)
		}
	}
	return found
}

// TestDirectorSkillStopsAWait runs directorSkillStopsAWait on words that tell
// the director to stop an older wait or its Monitor, and on words it must not
// flag: those that forbid the stop, and the words of the skill that say what
// the Monitor does.
func TestDirectorSkillStopsAWait(t *testing.T) {
	for _, tc := range []struct {
		text  string
		match bool
	}{
		{"古い wait の Monitor を止める。", true},
		{"古い wait の Monitor を止めて、新しい wait を残す", true},
		{"古い wait を TaskStop で止め、新しい wait を残す", true},
		{"古い wait も止める", true},
		{"Monitor を止めろ", true},
		{"Monitor を止めよ", true},
		{"古い wait を止め。", true},
		{"古い wait の `Monitor` を止める", true},
		{"古い wait の Monitor は止める", true},
		{"古い wait を停止する", true},
		{"古い wait を停止し、新しい wait を残す", true},
		{"古い wait を kill する", true},
		{"古い wait を kill して、新しい wait を残す", true},
		{"古い wait を TaskStop する", true},
		{"古い wait を TaskStop しろ", true},
		{"古い wait を打ち切る", true},
		{"古い wait を打ち切って、新しい wait を残す", true},
		{"古い wait を打ち切り、新しい wait を残す", true},
		{"古い wait を打ち切れ", true},
		{"古い wait を終わらせる", true},
		{"古い wait を止めず、新しい wait を kill する", true},
		{"古い wait の Monitor を止めない。", false},
		{"新しい wait も止めない。", false},
		{"古い wait を止めるな。", false},
		{"古い wait の Monitor を止めてはいけない。", false},
		{"古い wait を終わらせてはならない", false},
		{"古い wait を kill するな", false},
		{"古い wait を TaskStop してはいけない", false},
		{"古い wait を停止してはならない", false},
		{"古い wait を打ち切ってはいけない", false},
		{"古い wait を打ち切ってはならない", false},
		{"古い wait を kill しない", false},
		{"古い wait を TaskStop で止めない", false},
		{"古い wait を打ち切らない", false},
		{"古い wait を終わらせるな", false},
		{"Monitor が自分の `timeout_ms` で wait を止めたので、wait の失敗ではない", false},
		{"Monitor は処理に SIGTERM を送って止めるので", false},
		{"止めた wait の終わりは Monitor の期限切れでない signal での終わりになり", false},
	} {
		if got := directorSkillStopsAWait(withoutSpaces(tc.text)); (len(got) > 0) != tc.match {
			t.Errorf("directorSkillStopsAWait(%q) = %q, want a match: %v", tc.text, got, tc.match)
		}
	}
}

// TestDirectorSkill_RunsOneInboxWaitAtATime pins the /director skill's rules
// on overlapping inbox waits (plan 2026-11-22-director-wait-types-note.md,
// issue #374, AC3). Two overlapping director waits both return new items
// (TestWaitInboxForDirector_TwoOverlappingWaitsBothReturnTheWatchsItems), so
// the skill does not end the older one: an end by a signal other than the
// Monitor's expiry is a failure that makes the director stop waiting on the
// inbox (the subsection's paragraph on such ends). It tells the director to
// run one wait at a time instead, and, once the line of a wait whose mark
// another wait took over shows, to start no further wait and let both end on
// their own. First, the line sayLost prints must start with
// directorWaitNoticePrefix and directorSkillTakeoverLineStart and contain
// directorSkillTakeoverLineVerb, the words the skill quotes. Then each copy's
// subsection ### wait で待つ (directorSkillWaitSubsection) must contain the
// fragments of directorSkillWaitOneAtATimeSteps in order
// (skillOrderedFragmentProblems), and no words that tell the director to stop
// a wait or its Monitor (directorSkillStopsAWait, on the subsection with
// spaces and line breaks removed). scripts/check-skill-sync.sh keeps the
// copies identical but does not read Go.
//
// If this test fails after a deliberate change to a rule, put the rule's new
// words in its fragment in the skill's order; if sayLost's line changed, write
// its new words in the skill and in the two constants. Otherwise restore the
// text of ### wait で待つ in .claude/skills/director/SKILL.md and
// templates/base/.claude/skills/director/SKILL.md and run
// scripts/sync-skills.sh for the two .agents copies, as for
// TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase
// (director_skill_sync_test.go).
func TestDirectorSkill_RunsOneInboxWaitAtATime(t *testing.T) {
	var stderr strings.Builder
	w := &directorWaitRun{p: DirectorWaitParams{StateDir: t.TempDir()}, stderr: &stderr, mark: DirectorWaitMark{WaitID: "4101-1"}}
	w.sayLost(&DirectorWaitMark{WaitID: "4102-2", PID: 4102})
	line := stderr.String()
	if prefix := directorWaitNoticePrefix + directorSkillTakeoverLineStart; !strings.HasPrefix(line, prefix) || !strings.Contains(line, directorSkillTakeoverLineVerb) {
		t.Errorf("the line of a wait whose mark another wait took over = %q; want it to start with %q and contain %q, the words the skill quotes", line, prefix, directorSkillTakeoverLineVerb)
	}

	root := directorSkillRepoRoot(t)
	for _, rel := range directorSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's one-wait-at-a-time check (vendored repo?)", rel, err)
		}
		section := directorSkillWaitSubsection(string(data))
		for _, p := range skillOrderedFragmentProblems(section, "the subsection ### wait で待つ", directorSkillWaitOneAtATimeSteps) {
			t.Errorf("%s: %s -- see this test's doc comment for what to update", rel, p)
		}
		for _, words := range directorSkillStopsAWait(withoutSpaces(section)) {
			t.Errorf("%s: the subsection ### wait で待つ says %q, which tells the director to stop a wait or its Monitor -- see this test's doc comment", rel, words)
		}
	}
}

// directorSkillTypesTimeoutSteps are fragments of the paragraph of the
// /director skill's subsection ### wait で待つ on the timeout of a --types
// wait, in the order the skill has them, each the words of one rule plan
// 2026-11-22-director-wait-types-note.md gave the director (issue #374,
// AC3): the timeout's text may end with the note inboxWaitTimeoutNote adds;
// the wait is still a timeout and is run again; the items after
// inboxWaitOtherTypesPhrase are open and were not counted because of
// --types, which is why the wait returned nothing; the director reads them
// and handles them as the human who asked for --types said, so with ALERT
// items left to their deadline it acks and resolves none of them; and a
// director wait writes them in its mark's detail, which the --json of the two
// status commands shows and their text lines do not.
var directorSkillTypesTimeoutSteps = []string{
	"`--types` を付けた wait が時間切れで終わるとき、数えなかった種類の open の件があれば、時間切れの文の後ろに次の形の句が付く",
	"時間切れであることは変わらないので、上のとおり打ち直す",
	"`--types` のために数えなかった件で、wait が件を返さなかったのはそのためである",
	"その件を `ralph org inbox show <id>` で読み、扱いは `--types` を指示した人の言葉に従う",
	"ALERT を期限に任せると言われていれば、ack も resolve もしない",
	"`--director` の wait は、終わった印の detail にも同じ件を書く",
	"`ralph org director status --json` の `wait.detail` と、`ralph status --json` の `director.wait.detail` で読める",
}

// directorSkillTypesTimeoutMore is how the /director skill writes the end of
// the list of the open items of other TYPEs that inboxWaitOtherTypesText
// cuts, <数> standing for the number it did not name.
const directorSkillTypesTimeoutMore = "and <数> more"

// TestDirectorSkill_ReadsTheUncountedItemsOfATypesTimeout pins the /director
// skill's paragraph on the timeout of a --types wait (plan
// 2026-11-22-director-wait-types-note.md, issue #374, AC3) to the text
// WaitInbox and WaitInboxForDirector print. Each copy's subsection ### wait
// で待つ must hold, compared with spaces and line breaks removed
// (withoutSpaces): inboxWaitOtherTypesPhrase in backquotes; in backquotes,
// the note inboxWaitTimeoutNote gives for QUESTION,BLOCKED,RESULT with the
// open items e3 ALERT and e5 ALERT (without its leading space), the example
// the skill shows; the number of items the note names,
// inboxWaitOtherTypesShown, with directorSkillTypesTimeoutMore for the rest,
// whose words inboxWaitOtherTypesText must print with the number in place of
// <数>; and the fragments of directorSkillTypesTimeoutSteps in order
// (skillOrderedFragmentProblems). scripts/check-skill-sync.sh keeps the copies
// identical but does not read Go.
//
// If this test fails after a change to the note, write its new text in the
// skill's example and paragraph; after a deliberate change to a rule, put
// its new words in its fragment. Otherwise restore the text of ### wait
// で待つ in .claude/skills/director/SKILL.md and
// templates/base/.claude/skills/director/SKILL.md and run
// scripts/sync-skills.sh for the two .agents copies, as for
// TestDirectorSkill_NamesTheInboxWaitTimeoutPhrase
// (director_skill_sync_test.go).
func TestDirectorSkill_ReadsTheUncountedItemsOfATypesTimeout(t *testing.T) {
	types := []string{"QUESTION", "BLOCKED", "RESULT"}
	example := strings.TrimSpace(inboxWaitTimeoutNote(types, inboxWaitOtherTypesText(types, []InboxItem{{ID: "e3", Type: "ALERT"}, {ID: "e5", Type: "ALERT"}})))
	var many []InboxItem
	for i := range inboxWaitOtherTypesShown + 1 {
		many = append(many, InboxItem{ID: fmt.Sprintf("e%d", i+1), Type: "ALERT"})
	}
	if got, want := inboxWaitOtherTypesText(types, many), strings.ReplaceAll(directorSkillTypesTimeoutMore, "<数>", "1"); !strings.HasSuffix(got, ", "+want) {
		t.Errorf("inboxWaitOtherTypesText of %d ALERT items = %q; want it to end with %q, the words the skill quotes as %q", len(many), got, ", "+want, directorSkillTypesTimeoutMore)
	}
	wants := []string{
		"`" + inboxWaitOtherTypesPhrase + "`",
		"`" + example + "`",
		fmt.Sprintf("%d 件を超えると、%d 件目からは `%s` で数だけ言う", inboxWaitOtherTypesShown, inboxWaitOtherTypesShown+1, directorSkillTypesTimeoutMore),
	}

	root := directorSkillRepoRoot(t)
	for _, rel := range directorSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's --types timeout check (vendored repo?)", rel, err)
		}
		section := directorSkillWaitSubsection(string(data))
		text := withoutSpaces(section)
		for _, want := range wants {
			if !strings.Contains(text, withoutSpaces(want)) {
				t.Errorf("%s: the subsection ### wait で待つ does not contain %q -- see this test's doc comment for what to update", rel, want)
			}
		}
		for _, p := range skillOrderedFragmentProblems(section, "the subsection ### wait で待つ", directorSkillTypesTimeoutSteps) {
			t.Errorf("%s: %s -- see this test's doc comment for what to update", rel, p)
		}
	}
}
