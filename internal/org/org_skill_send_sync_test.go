package org

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// orgSkillCopies are the four copies of the /org skill, relative to the
// repository root.
var orgSkillCopies = []string{
	filepath.Join(".claude", "skills", "org", "SKILL.md"),
	filepath.Join(".agents", "skills", "org", "SKILL.md"),
	filepath.Join("templates", "base", ".claude", "skills", "org", "SKILL.md"),
	filepath.Join("templates", "base", ".agents", "skills", "org", "SKILL.md"),
}

// orgSkillSendRowRules are fragments of the /org skill's verb-table row of
// send, in the order the row has them (AC4 of plan 2026-10-23-busy-leader.md):
// Send waits for the seat to be idle or done before it types (FR-3's wait,
// which Send keeps), so a seat in the middle of a turn gets nothing until
// the turn ends; a wait that ran out types nothing and ends with exit 1, so
// the message can be sent again as it is; and the submit check reads a turn
// the seat started on its own as confirmed, so the pane is what shows the
// message arrived.
var orgSkillSendRowRules = []string{
	"本文を入力する前に、座席が idle か done になるのを `--timeout-ms`(既定 30000)まで待つ",
	"その手番が終わるまで届かない",
	"待ちが切れると何も入力せずに終了コード 1 で終わり",
	"そのまま送り直せる",
	"座席が自分で始めた手番",
	"届いたかは `read` で pane を見て確かめる",
}

// orgSkillInboxAnswerRules are fragments of the bullet of the /org skill's
// section ## 受信箱 that says how to answer an item, in order: the resolve
// comes after the leader's pane shows the DECISION left the input box, a
// failed send (a wait that ran out included) is not resolved, and the
// /director skill's ### 答える順 has the steps.
var orgSkillInboxAnswerRules = []string{
	"`ralph org read --org-id <org> --seat leader` で DECISION が leader の入力欄に残っていないことを確かめてから",
	"send が失敗したら(待ちの時間切れも含む)resolve せず",
	"`/director` skill の「答える順」",
}

// orgSkillSendRow returns the line of text that starts the verb-table row of
// send, "" when there is none.
func orgSkillSendRow(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "| `send` |") {
			return line
		}
	}
	return ""
}

// orgSkillInboxAnswerBullet returns the bullet of the section ## 受信箱 that
// starts "- 件に答えるときは": that line and the lines after it, up to the
// next line that is not blank and does not start with a space, "" when there
// is none.
func orgSkillInboxAnswerBullet(text string) string {
	section := directorSkillSubsection(text, "## 受信箱")
	lines := strings.Split(section, "\n")
	start := -1
	for i, line := range lines {
		if start >= 0 {
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, " ") {
				return strings.Join(lines[start:i], "\n")
			}
			continue
		}
		if strings.HasPrefix(line, "- 件に答えるときは") {
			start = i
		}
	}
	if start < 0 {
		return ""
	}
	return strings.Join(lines[start:], "\n")
}

// TestOrgSkill_SendRowSaysItWaitsForIdleBeforeTyping pins what the /org
// skill says about send's wait and its submit check (AC4 of plan
// 2026-10-23-busy-leader.md). A director or leader reads the send row to
// know that a seat busy in the foreground gets nothing until its turn ends,
// that a send whose wait ran out typed nothing and can be sent again, and
// that exit 0 does not prove the message arrived; the inbox section's bullet
// on answering an item says the same for the resolve. Each copy's send row
// must contain every fragment of orgSkillSendRowRules in order, and its
// bullet every fragment of orgSkillInboxAnswerRules in order, compared with
// spaces and line breaks removed (withoutSpaces). scripts/check-skill-sync.sh
// keeps the copies identical but does not read Go.
//
// If this test fails after a deliberate change to a rule, put the rule's new
// words, short and verbatim, in its fragment in the skill's order; otherwise
// restore the text in .claude/skills/org/SKILL.md and
// templates/base/.claude/skills/org/SKILL.md and run scripts/sync-skills.sh
// for the two .agents copies (root, then with
// CLAUDE_ROOT=templates/base/.claude/skills
// CODEX_ROOT=templates/base/.agents/skills).
func TestOrgSkill_SendRowSaysItWaitsForIdleBeforeTyping(t *testing.T) {
	root := directorSkillRepoRoot(t)
	for _, rel := range orgSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's send-row check (vendored repo?)", rel, err)
		}
		text := string(data)
		for _, in := range []struct {
			name      string
			text      string
			fragments []string
		}{
			{"the verb-table row of send", orgSkillSendRow(text), orgSkillSendRowRules},
			{"the bullet of ## 受信箱 on answering an item", orgSkillInboxAnswerBullet(text), orgSkillInboxAnswerRules},
		} {
			for _, p := range skillOrderedFragmentProblems(in.text, in.name, in.fragments) {
				t.Errorf("%s: %s -- see this test's doc comment for what to update", rel, p)
			}
		}
	}
}
