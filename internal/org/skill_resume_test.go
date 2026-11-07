package org

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// The /director and /org skill text of `ralph org resume` (plan
// 2026-11-07-org-seat-resume, S8 and AC-12; spec FR-10 and FR-11).
// scripts/check-skill-sync.sh keeps the four copies of each skill identical
// but does not read Go, so these tests tie the words the skills give a
// director to what Resume does and prints and to the screen sentences
// resume_screens.go knows.
//
// If a test here fails after a deliberate change to a rule, put the rule's
// new words, short and verbatim, in its fragment in the skill's order;
// otherwise restore the text in .claude/skills/<skill>/SKILL.md and
// templates/base/.claude/skills/<skill>/SKILL.md and run
// scripts/sync-skills.sh for the two .agents copies (root, then with
// CLAUDE_ROOT=templates/base/.claude/skills
// CODEX_ROOT=templates/base/.agents/skills).

// directorResumeHeading is the /director skill's section on resume, placed
// after the inbox section's last subsection and before ## 進捗の報告.
const directorResumeHeading = "## 止まった座席を続ける"

// directorResumeSection returns the /director skill's section
// directorResumeHeading (directorSkillSection).
func directorResumeSection(text string) string {
	return directorSkillSection(text, directorResumeHeading)
}

// resumeOutcomeBullet is the start of the bullet of step 3 of
// directorResumeSection that says how to read outcome o of a real run.
func resumeOutcomeBullet(o ResumeOutcome) string { return "- `" + string(o) + "`:" }

// TestDirectorSkill_ResumeSectionSaysWhenToRunItAndHowToReadIt pins FR-10's
// procedure in the /director skill: resume only when the human asks to go
// on after a connection failure or a usage limit, never on the director's
// own judgment; --org-id and --dry-run (with the dry run's outcomes); what
// Resume reads (resumeReadLines lines) and its order; a bullet for each
// outcome of a real run; the read command for a skipped seat and no typing
// into it; the note: line read without pressing Enter; the reset time; and
// the human's own shell for a stopped headless director. The section sits
// between ### cross-review の上限 and ## 進捗の報告.
func TestDirectorSkill_ResumeSectionSaysWhenToRunItAndHowToReadIt(t *testing.T) {
	fragments := []string{
		"人が「続けて」「再開して」「通信が戻った」「上限がリセットされた」のように、通信断か利用上限のあとに続けるよう頼んだときだけ打つ",
		"自分の判断では打たない",
		"1 つの org だけなら `--org-id <org>` を付ける(このときは director を見ない)",
		"`--dry-run` を付けて打つ",
		"`" + string(ResumeOutcomeWouldResume) + "` か `" + string(ResumeOutcomeWouldEnter) + "`",
		"pane の最後の " + strconv.Itoa(resumeReadLines) + " 行を読む",
		"leader 以外の座席、leader、headless の director の順に打ち",
		"`resume: 1 resumed, 0 entered, 1 waiting, 1 skipped, 0 failed`",
		resumeOutcomeBullet(ResumeOutcomeResumed),
		resumeOutcomeBullet(ResumeOutcomeEntered),
		resumeOutcomeBullet(ResumeOutcomeWaiting),
		resumeOutcomeBullet(ResumeOutcomeSkipped),
		"`ralph org read --org-id <org> --seat <座席>` を添える",
		"その座席には自分で打たない",
		resumeOutcomeBullet(ResumeOutcomeFailed),
		"stderr の `note:` の行",
		"自分で Enter を送らず、resume も打ち直さない",
		"リセットの前に頼まれても resume は打ち",
		"人が自分の shell から `ralph org resume` を打てる",
	}
	checkDirectorSkillPlaces(t, "resume section", []skillPlace{
		{"the section " + directorResumeHeading, directorResumeSection, fragments},
	})
	checkSkillHeadingOrder(t, directorSkillCopies, []string{directorXreviewCapHeading, directorResumeHeading, "## 進捗の報告"})
}

// TestDirectorSkill_ResumeRulesInTheAlertBulletAndTheDonts pins FR-10's two
// other places in the /director skill: the ALERT bullet of ### 件の処理
// reads the seat's screen, sends an item whose screen ends on an API error
// to the human with 「続けて」, never runs resume on its own, and waits out
// codex's wait for the network; ## してはいけないこと keeps the bullet on
// the trust and bypass screens and adds that resume (when asked) and the
// DECISION send are the only ways into a pane, never a send or send-keys to
// a seat resume listed as waiting or skipped.
func TestDirectorSkill_ResumeRulesInTheAlertBulletAndTheDonts(t *testing.T) {
	checkDirectorSkillPlaces(t, "resume rules", []skillPlace{
		{"the ALERT bullet of ### 件の処理", func(text string) string { return directorSkillStep3Bullet(text, "- ALERT(見張りの escalation)") }, []string{
			"`SEAT` の座席の画面を `ralph org read --org-id <org> --seat <座席>` で読み",
			"その座席は API エラーでターンを終えて止まっている",
			"この件は人に送る側にし",
			"「続けて」と言えば続けられること",
			"`ralph org resume` は自分の判断では打たない",
			"`Reconnecting... waiting for network` が出ていれば",
			"何も打たずに待ち、人には通信が切れているようだと伝える",
			"座席や org を止めるか(`ralph org stop`、`ralph org disband`)は人が決める",
		}},
		{"the section ## してはいけないこと", func(text string) string { return directorSkillSubsection(text, "## してはいけないこと") }, []string{
			"座席の pane のフォルダ信頼の画面と bypass の承諾の画面に、キーを送って答えない",
			"座席と director の pane に文字やキーを送るのは、`ralph org resume`(人が続けるよう頼んだときだけ)と、いまある DECISION の送り方",
			"resume が `" + string(ResumeOutcomeWaiting) + "` か `" + string(ResumeOutcomeSkipped) + "` と出した座席に、" +
				"`ralph org send` や `herdr pane send-keys` で自分で打たない",
		}},
	})
}

// resumeSkillScreenSentence is a screen sentence the /director skill
// quotes, with the driver and the kind of resumeScreenSentences it must be
// in.
type resumeSkillScreenSentence struct {
	driver string
	kind   ResumeScreenKind
	text   string
}

// resumeSentenceTexts returns the texts of driverName's sentences of kind.
func resumeSentenceTexts(driverName string, kind ResumeScreenKind) []string {
	table := resumeScreenSentences[driverName]
	var list []resumeSentence
	switch kind {
	case ResumeScreenStopped:
		list = table.Stopped
	case ResumeScreenEnterOnly:
		list = table.EnterOnly
	case ResumeScreenWaiting:
		list = table.Waiting
	}
	texts := make([]string, 0, len(list))
	for _, s := range list {
		texts = append(texts, s.Text)
	}
	return texts
}

// TestDirectorSkill_ResumeScreenSentencesAreInTheTable ties the screen
// sentences the /director skill quotes, in backquotes, to the table
// ClassifyResumeScreen reads: each must be an entry of its driver and kind,
// so a sentence a CLI release changes in resume_screens.go is changed in
// the skill too. The ALERT bullet quotes the stopped sentences and codex's
// wait; the resume section quotes the waits and the enter-only screen.
func TestDirectorSkill_ResumeScreenSentencesAreInTheTable(t *testing.T) {
	alert := []resumeSkillScreenSentence{
		{"claude", ResumeScreenStopped, "API Error:"},
		{"claude", ResumeScreenStopped, "You've hit your session limit"},
		{"codex", ResumeScreenStopped, "You've hit your usage limit"},
		{"codex", ResumeScreenWaiting, "Reconnecting... waiting for network"},
	}
	section := []resumeSkillScreenSentence{
		{"claude", ResumeScreenEnterOnly, "Press enter to continue"},
		{"claude", ResumeScreenWaiting, "Retrying in"},
		{"codex", ResumeScreenWaiting, "Reconnecting... waiting for network"},
	}
	for _, s := range slices.Concat(alert, section) {
		if !slices.Contains(resumeSentenceTexts(s.driver, s.kind), s.text) {
			t.Errorf("resumeScreenSentences[%q] has no %s sentence %q, which the /director skill quotes -- see the comment at the top of this file", s.driver, s.kind, s.text)
		}
	}
	quoted := func(list []resumeSkillScreenSentence) []string {
		var out []string
		for _, s := range list {
			out = append(out, "`"+s.text+"`")
		}
		return out
	}
	root := directorSkillRepoRoot(t)
	for _, rel := range directorSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's screen sentence check (vendored repo?)", rel, err)
		}
		text := string(data)
		for _, place := range []struct {
			name, text string
			want       []string
		}{
			{"the ALERT bullet of ### 件の処理", directorSkillStep3Bullet(text, "- ALERT(見張りの escalation)"), quoted(alert)},
			{"the section " + directorResumeHeading, directorResumeSection(text), quoted(section)},
		} {
			for _, w := range place.want {
				if !strings.Contains(place.text, w) {
					t.Errorf("%s: %s does not quote %s -- see the comment at the top of this file", rel, place.name, w)
				}
			}
		}
	}
}

// TestOrgSkill_ResumeRowAndTheClaudeRetryEnv pins FR-11 in the /org skill:
// the verb-table row of resume, right after the row of read, says what it
// looks at (resumeReadLines lines), when it types and when it presses Enter
// only, the pane lock it shares with send, its order and the pane it skips,
// the `resumed` event (EventResumed), its outcomes, --dry-run, and that a
// director runs it only when asked; and ## 前提 names the three --env
// entries of a claude seat (driver.ClaudeRetryEnv), none for a codex seat
// or 0, and no stall ALERT during the wait.
func TestOrgSkill_ResumeRowAndTheClaudeRetryEnv(t *testing.T) {
	outcomes := make([]string, 0, 5)
	for _, o := range []ResumeOutcome{
		ResumeOutcomeResumed, ResumeOutcomeEntered, ResumeOutcomeWaiting, ResumeOutcomeSkipped, ResumeOutcomeFailed,
	} {
		outcomes = append(outcomes, "`"+string(o)+"`")
	}
	checkOrgSkillPlaces(t, "resume row and claude retry env", []skillPlace{
		{"the verb row of resume", func(text string) string { return orgSkillVerbRow(text, "resume") }, []string{
			"`--org-id` を省くと台帳のすべての org の動いている座席と herdr の中の headless の director",
			"pane の最後の " + strconv.Itoa(resumeReadLines) + " 行を読み",
			"入力欄が空のときだけ",
			"(`Press enter to continue`)には Enter だけを 1 回送る",
			"`send` と同じ pane のロックを取り",
			"入力欄に文が入ったことを確かめてから Enter を押す",
			"leader 以外の座席、leader、headless の director の順に打ち",
			"resume を打った pane(`HERDR_PANE_ID`)は飛ばす",
			"manifest に `" + EventResumed + "` を記録する(状態は変えない)",
			strings.Join(outcomes, "・"),
			"`--dry-run` は同じように見るだけで、何も打たず何も記録しない",
			"`" + string(ResumeOutcomeWouldResume) + "`・`" + string(ResumeOutcomeWouldEnter) + "`",
			"director は人に頼まれたときだけ打つ",
			"`ralph org resume --org-id X --dry-run`",
		}},
		{"the section ## 前提", func(text string) string { return directorSkillSubsection(text, "## 前提") }, []string{
			"driver が claude の座席には、そのあとに `[org].api_wait_minutes`(既定 30)から作る",
			"`--env " + driver.ClaudeRetryWatchdogEnv + "=1`",
			"`--env " + driver.ClaudeRetryWatchdogMaxWaitMSEnv + "=<分 × 60000>`",
			"`--env " + driver.ClaudeMaxRetriesEnv + "=<分に見合う回数>`",
			"(`0` なら付けない。codex の座席には付けない)",
			"見張りの stall の ALERT は出ない",
		}},
	})

	root := directorSkillRepoRoot(t)
	for _, rel := range orgSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's resume row order check (vendored repo?)", rel, err)
		}
		lines := strings.Split(string(data), "\n")
		read := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(l, "| `read` |") })
		if read < 0 || read+1 >= len(lines) || !strings.HasPrefix(lines[read+1], "| `resume` |") {
			t.Errorf("%s: the verb-table row of resume does not follow the row of read -- see the comment at the top of this file", rel)
		}
	}
}
