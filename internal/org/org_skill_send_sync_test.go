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
func orgSkillSendRow(text string) string { return orgSkillVerbRow(text, "send") }

// orgSkillVerbRow returns the line of text that starts the verb-table row of
// verb, "" when there is none.
func orgSkillVerbRow(text, verb string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "| `"+verb+"` |") {
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

// orgSkillCleanupRules are fragments of the /org skill on the cleanup of a
// merged feature org's worktree (plan
// docs/plans/active/2026-10-28-org-merge-cleanup.md, S4: AC5), by where they
// are, each list in the skill's order: the verb-table row of cleanup (what it
// removes, where it runs, which branches it counts, the checks, the script it
// runs, its lines, what a SKIPPED: line covers and when a MANUAL: line comes,
// its exit codes); the row of merge (the cleanup once pr_merged is recorded,
// its CLEANUP: lines, what skipped covers, the exit code it leaves alone, the
// rerun of a skipped one, when a command by hand comes, and the spawn and
// start it refuses meanwhile); the rows of spawn and start (refused while
// merge or cleanup removes the org's worktree); and the subsection
// ### org の終わり方 (who cleans up after a merge, what stops it, that the
// next step is most often the rerun and a command by hand comes only where
// ralph cannot go on by itself, that ignored files go with the worktree,
// that --force-branch is safe after a squash merge, and the script by hand
// for a PR closed without a merge).
var orgSkillCleanupRules = []struct {
	name      string
	section   func(text string) string
	fragments []string
}{
	{"the verb-table row of cleanup", func(text string) string { return orgSkillVerbRow(text, "cleanup") }, []string{
		"GitHub で(ブラウザか `gh pr merge` で)merge された PR の機能の worktree・ローカルのブランチ・`scripts/ralph-worktree.sh` の記録を消す",
		"打つ場所は main worktree(機能の worktree の中から打つと断られる)",
		"`gh pr list` は `git remote get-url origin` から読んだ `-R <host>/<owner>/<repo>` に固定する",
		"PR の head(`headRefOid`)で片付ける",
		"merge されたか分からないので `SKIPPED:` の行を出す",
		"`gh` が origin 以外の repository の PR(url の host か owner/repo が origin と違う)を返したときも、どのブランチでも" +
			"その PR を使わず",
		"数えず行も出さない",
		"何も消さずに断る",
		"gitignore されたファイルは数えず、worktree と一緒に消える",
		"ディレクトリのない worktree は、git に残る登録がそのブランチにいて、ブランチが残っている",
		"`./scripts/ralph-worktree.sh cleanup --id org-<org_id> --force-branch` を打ち、`worktree_cleaned` を `via=cleanup` で記録する",
		"打ち直すと残ったものから続ける",
		"その org の `spawn` と `start` が断られる",
		"`CLEANED: <ブランチ> org-<org_id>`",
		"`nothing to clean up`",
		"`SKIPPED:` は片付けが終わっていないことで、断り(何も消していない)と、消す作業の失敗(script の途中なら消した分は" +
			"消えたまま、script の前の登録の外しなら何も消していない)と、PR を読めなかったブランチを含む",
		"`MANUAL:` が出るのは、ralph が先に進めず人の判断が要るときだけ(記録 `org-<org_id>` がないか結びつきと合わない、" +
			"ブランチがなく worktree だけがある、ディレクトリのない worktree の登録に消すと失う HEAD がある、消す途中で失敗した)で、" +
			"ほかの `SKIPPED:` のブランチは、理由を直してから `--branch` で打ち直せば続きから片付く",
		"`SKIPPED:` が 1 本でもあれば 1",
		"origin が GitHub の repository を名指さないか(ローカルのパスなど)`git remote get-url origin` が落ちるか時間切れになるとき" +
			"(台帳を読む前に、何も消さず断る",
		"`--org-id` は断る",
	}},
	{"the verb-table row of merge", func(text string) string { return orgSkillVerbRow(text, "merge") }, []string{
		"`pr_merged` を記録したら、`regate.lock` を持ったまま、ブランチの機能の worktree を片付ける",
		"`worktree_cleaned` を `via=merge` で記録する",
		"`CLEANUP: done` か、`CLEANUP: skipped (<理由>)` と、手で消すコマンドがあれば `CLEANUP MANUAL: <コマンド>` を出す",
		"`skipped` は片付けが終わっていないことで、断り(何も消していない)と、消す作業の失敗(script の途中なら消した分は" +
			"消えたまま、script の前の登録の外しなら何も消していない)の両方を含む",
		"片付けは終了コードを変えない",
		"片付かなかったブランチは、理由を直してから `ralph org cleanup --branch <ブランチ>` で片付ける",
		"手で消すコマンドが出るのは、ralph が先に進めず人の判断が要るときだけ(記録 `org-<org_id>` がないか結びつきと合わない、" +
			"ブランチがなく worktree だけがある、ディレクトリのない worktree の登録に消すと失う HEAD がある、消す途中で失敗した)",
		"その org の `spawn` と `start` は、merge の続きの片付けの間だけ断られる",
	}},
	{"the verb-table row of spawn", func(text string) string { return orgSkillVerbRow(text, "spawn") }, []string{
		"`ralph org merge` か `ralph org cleanup` がその org の merge された機能の worktree を消している間は、`--dry-run` でない spawn は何も記録せずに断られる",
	}},
	{"the verb-table row of start", func(text string) string { return orgSkillVerbRow(text, "start") }, []string{
		"`ralph org merge` か `ralph org cleanup` がその org の merge された機能の worktree を消している間は、何も記録せずに断られる",
	}},
	{"the subsection ### org の終わり方", func(text string) string { return directorSkillSubsection(text, "### org の終わり方") }, []string{
		"disband は worktree とブランチを消さない",
		"PR が merge されたら、`ralph org merge` か `ralph org cleanup` が",
		"`ralph org merge` で merge した PR は、merge の続きで片付き、結果は `CLEANUP:` の行で出る。片付けは終了コードを変えない",
		"人が GitHub で直接 merge した PR は、main のチェックアウトから `ralph org cleanup` で片付ける",
		"どちらも、次のどれかに当たれば何も消さず、理由と次の手を出す",
		"worktree に未コミットの変更か未追跡のファイルがある",
		"ローカルのブランチの先頭か worktree の HEAD が、merge された PR の head と違う",
		"worktree が detached HEAD か、別のブランチにいる",
		"ディレクトリのない worktree は、git に残る登録の HEAD がそうなっているか、そのブランチが消えている",
		"記録 `org-<org_id>` が結びつきと合わない",
		"ralph をその worktree の中から打った",
		"次の手はたいてい、理由を直してから `ralph org cleanup --branch <ブランチ>` を打ち直すことで、打ち直せば残ったものから片付く",
		"手で消すコマンド(`CLEANUP MANUAL:` と `MANUAL:` の行)は、ralph が先に進めず人の判断が要るときだけ出る",
		"記録 `org-<org_id>` が結びつきと合わないか、もうないとき、ブランチがなく worktree だけがあるとき、ディレクトリのない " +
			"worktree の登録に消すと失う HEAD があるとき、消す作業が失敗したとき(script が途中で止まったなら消した分は消えた" +
			"まま、script の前の、ディレクトリのない worktree の登録の外しで止まったなら何も消していない)の 4 つ",
		"登録の断りは、消すと登録の HEAD と reflog を失うので何も消さず、理由が HEAD を残す方法を出し",
		"手で消すコマンドは `git worktree remove` と script の cleanup になる",
		"打ち直しても同じ断りになる",
		"gitignore されたファイル",
		"止める理由にならず、worktree と一緒に消える",
		"squash merge のあとにリモートのブランチが消えていても止まらない",
		"PR が merge されずに閉じられたときは、ralph は片付けない",
		"`./scripts/ralph-worktree.sh cleanup --id org-<org_id>` で消す",
		"ブランチが要らないと確かめたうえで `--force-branch` を足す",
	}},
}

// orgSkillOldCleanupWords are what the /org skill said before ralph cleaned
// up a merged feature org's worktree: a human ran the script after every
// merge, and merge refused no spawn or start.
var orgSkillOldCleanupWords = []string{
	"PR が merge されたら、人が main のチェックアウトから `./scripts/ralph-worktree.sh cleanup --id org-<org_id>` で worktree・ブランチ・記録を消す",
	"そのときは `--force-branch` を足して打ち直す",
	"worktree を変えないので、その org の `spawn` と `start` は断られない",
	"merge のあとに `cleanup` で消す",
}

// TestOrgSkill_PinsTheCleanupAfterAMerge pins what the /org skill says about
// the cleanup of a merged feature org's worktree (plan
// docs/plans/active/2026-10-28-org-merge-cleanup.md, S4: AC5). A leader, a
// director, and a human read the verb table and ### org の終わり方 to know
// that ralph org merge and ralph org cleanup remove the worktree once the PR
// is merged, what stops them, and when the script is still run by hand, so
// each copy must have every fragment of orgSkillCleanupRules where it says,
// in order, and none of orgSkillOldCleanupWords, compared with spaces and
// line breaks removed (withoutSpaces). scripts/check-skill-sync.sh keeps the
// copies identical but does not read Go.
//
// If this test fails after a deliberate change to a rule, put the rule's new
// words, short and verbatim, in its fragment in the skill's order; otherwise
// restore the text in .claude/skills/org/SKILL.md and
// templates/base/.claude/skills/org/SKILL.md and run scripts/sync-skills.sh
// for the two .agents copies, as for the test above.
func TestOrgSkill_PinsTheCleanupAfterAMerge(t *testing.T) {
	root := directorSkillRepoRoot(t)
	for _, rel := range orgSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's cleanup check (vendored repo?)", rel, err)
		}
		text := string(data)
		for _, in := range orgSkillCleanupRules {
			for _, p := range skillOrderedFragmentProblems(in.section(text), in.name, in.fragments) {
				t.Errorf("%s: %s -- see this test's doc comment for what to update", rel, p)
			}
		}
		squashed := withoutSpaces(text)
		for _, old := range orgSkillOldCleanupWords {
			if strings.Contains(squashed, withoutSpaces(old)) {
				t.Errorf("%s still says %q -- see this test's doc comment for what to update", rel, old)
			}
		}
	}
}
