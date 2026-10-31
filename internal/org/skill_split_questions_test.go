package org

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The tests in this file pin what the /director skill says about the steps
// from a request to the feature orgs of issue #271 (plan
// 2026-11-01-director-split-questions, AC1-AC4 and the skill half of AC6,
// D1-D3 and D6): the subsections of ## 分割計画と機能ごとの org in order, the
// rule every multiple-choice question follows (### 人への聞き方, written from
// the live check of S1), the questions before a split plan
// (### 分割計画の前の質問), the approval question (### 分割計画の承認), the
// question of where each feature goes (### 機能ごとの出力先), and the issue
// and the follow-ups of a feature filed for later (### 起票と記録).
//
// If one of these tests fails after a deliberate change to a rule, put the
// rule's new words, short and verbatim, in its fragment in the skill's order;
// otherwise restore the text in .claude/skills/director/SKILL.md and
// templates/base/.claude/skills/director/SKILL.md and run
// scripts/sync-skills.sh for the two .agents copies, as for the first test in
// director_skill_sync_test.go.

// The section of the /director skill these tests read and its subsections,
// in the skill's order.
const (
	directorSplitSection          = "## 分割計画と機能ごとの org"
	directorSplitAskingHeading    = "### 人への聞き方"
	directorSplitQuestionsHeading = "### 分割計画の前の質問"
	directorSplitApprovalHeading  = "### 分割計画の承認"
	directorSplitOutputHeading    = "### 機能ごとの出力先"
	directorSplitFilingHeading    = "### 起票と記録"
)

// directorSplitAskEvidence is the live check of S1 that the skill's
// ### 人への聞き方 points at, relative to the repository root.
const directorSplitAskEvidence = "docs/evidence/director-ask-herdr-2026-11-01.md"

// specSkillCopies are the four copies of the /spec skill, relative to the
// repository root.
var specSkillCopies = []string{
	filepath.Join(".claude", "skills", "spec", "SKILL.md"),
	filepath.Join(".agents", "skills", "spec", "SKILL.md"),
	filepath.Join("templates", "base", ".claude", "skills", "spec", "SKILL.md"),
	filepath.Join("templates", "base", ".agents", "skills", "spec", "SKILL.md"),
}

// directorSkillSection returns the section of text whose heading line is
// heading, a "## " heading: that line and the lines after it, its "### "
// subsections included, up to the next line that starts with "## " or "# "
// or the end of the text. It returns "" when the text has no such heading.
// Like directorSkillSubsection, it reads a line of a code block as a heading
// when the line starts with one of those prefixes.
func directorSkillSection(text, heading string) string {
	lines := strings.Split(text, "\n")
	start := -1
	for i, line := range lines {
		if start >= 0 {
			if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "# ") {
				return strings.Join(lines[start:i], "\n")
			}
			continue
		}
		if strings.TrimSpace(line) == heading {
			start = i
		}
	}
	if start < 0 {
		return ""
	}
	return strings.Join(lines[start:], "\n")
}

// directorSplitSubsection returns a section func for skillPlace that reads
// the /director skill's subsection heading (directorSkillSubsection).
func directorSplitSubsection(heading string) func(string) string {
	return func(text string) string { return directorSkillSubsection(text, heading) }
}

// directorSplitStep returns a section func for skillPlace that reads item n
// of the numbered list of the /director skill's subsection heading
// (directorSkillNumberedStep).
func directorSplitStep(heading string, n int) func(string) string {
	return func(text string) string {
		return directorSkillNumberedStep(directorSkillSubsection(text, heading), n)
	}
}

// TestDirectorSkill_SplitSectionHasTheAskingRuleThenTheFourSteps pins the
// shape of the /director skill's ## 分割計画と機能ごとの org (AC1-AC4): the
// section opens with a paragraph naming its steps and the asking rule, then
// has ### 人への聞き方, ### 分割計画の前の質問, ### 分割計画の承認,
// ### 機能ごとの出力先, and ### 起票と記録 in that order, all before
// ## 受信箱の処理. The other tests in this file read the subsections by these
// headings.
func TestDirectorSkill_SplitSectionHasTheAskingRuleThenTheFourSteps(t *testing.T) {
	checkSkillHeadingOrder(t, directorSkillCopies, []string{
		directorSplitSection,
		directorSplitAskingHeading,
		directorSplitQuestionsHeading,
		directorSplitApprovalHeading,
		directorSplitOutputHeading,
		directorSplitFilingHeading,
		"## 受信箱の処理",
	})
	checkDirectorSkillPlaces(t, "split section opening", []skillPlace{
		{"the opening of the section " + directorSplitSection, directorSplitSubsection(directorSplitSection), []string{
			"仕事を機能に分けて機能ごとの org を立てるまでを、下の節の順に進める",
			"分割計画を書く前に人と要件を詰め(「分割計画の前の質問」)、分割計画の承認を取り(「分割計画の承認」)",
			"機能ごとに今すぐ着手するか issue に起票するかを聞く(「機能ごとの出力先」と「起票と記録」)",
			"人に選択肢で聞くときは、どこでも「人への聞き方」に従う",
		}},
	})
}

// TestDirectorSkill_AsksWithNumberedOptionsWhenTheWakePushes pins the
// /director skill's ### 人への聞き方 (AC6, D6), written from the second
// branch of D6 that S1 found (docs/evidence/director-ask-herdr-2026-11-01.md,
// ## 読み取り): every multiple-choice question (this section's, the FORK
// questions of ### 計画の承認, the start question of ### merge を待つ issue)
// is an AskUserQuestion on Claude Code and numbered options on Codex, with
// the recommended option first and marked (推奨); a director whose wake
// includes push (read on the wake: line of ralph org director status) writes
// numbered options instead of AskUserQuestion, because the watch's line and
// Enter pick the option under the cursor (the recommended one right after
// the question opens), or become the answer on the free-text row, and the
// line is lost on an option row so the director misses that wake; a
// pull-only director, a headless claude one included, asks with
// AskUserQuestion, answered in the herdr pane or the launch_cmd terminal; a
// Codex director always writes numbered options; and a watch line that
// arrives while a numbered question waits is not the human's answer. The
// evidence file it names must exist, and the wake names it gives must be
// DirectorWakePull and DirectorWakePush.
func TestDirectorSkill_AsksWithNumberedOptionsWhenTheWakePushes(t *testing.T) {
	checkDirectorSkillPlaces(t, "asking rule", []skillPlace{
		{"the subsection " + directorSplitAskingHeading, directorSplitSubsection(directorSplitAskingHeading), []string{
			"人に選択肢で聞く場面(「分割計画と機能ごとの org」の問い、「受信箱の処理」の「計画の承認」の `FORK` の問い、「merge を待つ issue」の着手の問い)では、" +
				"Claude Code は AskUserQuestion、Codex は番号つきの選択肢で聞く",
			"番号つきの選択肢は、問いごとに選択肢を番号つきで並べ(推奨の案があれば 1 番に置き、「(推奨)」を付ける)、人が数字で返すのを待つ",
			"登録の wake に push がある director(`ralph org director status` の `wake:` の行が `push` か `pull,push`)は、" +
				"Claude Code でも AskUserQuestion を使わず、番号つきの選択肢を文で書いて、人の返事を待つ",
			"問いが開いているあいだに見張りの 1 行と Enter(「見張りが送る 1 行」)が届くと、" +
				"Enter が人の代わりにカーソルの下の選択肢(問いを出した直後は推奨の 1 番)を選ぶか、" +
				"カーソルが自由記述の欄にあれば 1 行がそのまま答えになるため",
			"選択肢の上では 1 行は消え、director はその wake の件にも気づかない",
			"`" + directorSplitAskEvidence + "`",
			"wake が pull だけの director は AskUserQuestion で聞く",
			"headless の claude の director も同じで、人は herdr の pane で(launch_cmd で立てたものは、その端末で)答える",
			"Codex の director は、wake によらず番号つきの選択肢で聞く",
			"番号つきの選択肢の返事を待つあいだに見張りの 1 行が届いても、それは人の答えではない",
			"「見張りが送る 1 行」のとおりに受信箱を処理してから、答えのない問いをもう一度示す",
		}},
	})

	if _, err := os.Stat(filepath.Join(directorSkillRepoRoot(t), directorSplitAskEvidence)); err != nil {
		t.Errorf("the /director skill's %s points at %s, which cannot be read: %v", directorSplitAskingHeading, directorSplitAskEvidence, err)
	}
	if DirectorWakePull != "pull" || DirectorWakePush != "push" {
		t.Errorf("DirectorWakePull, DirectorWakePush = %q, %q, but the /director skill's %s reads `push` and `pull,push` on the wake: line",
			DirectorWakePull, DirectorWakePush, directorSplitAskingHeading)
	}
}

// TestDirectorSkill_AsksTheSpecQuestionsBeforeWritingASplitPlan pins the
// /director skill's ### 分割計画の前の質問 (AC1): before a split plan the
// director goes as deep as steps 1-3 of the /spec skill, which it points at
// instead of copying: step 1 reads the request (gh issue view), AGENTS.md,
// .claude/rules/ and the code without asking; step 2 asks the decision tree
// upstream first with AskUserQuestion (numbered options on Codex and with a
// push wake), at most 4 questions a call (the tool's limit, not /spec's
// five), 2-4 options each with the recommended one first and marked (推奨)
// and a one-sentence reason, looks the answer up in the code, the docs and
// the Web before asking, answers a challenge before going on, and stops when
// the human says enough or no new decision comes; step 3 asks only what is
// still ambiguous. The four cross-feature forks (boundaries, which feature
// reserves a shared file, the seats' driver and model, the order) are
// settled here, and the forks inside a feature are left to the leader's FORK
// lines; /spec's steps 4-5 are the split plan and its approval, and steps
// 6-8 are not taken: no spec file, worktree, or PR. The /spec skill's own
// copies must still number those steps so (and ask in batches of five), or
// the pointer is stale.
func TestDirectorSkill_AsksTheSpecQuestionsBeforeWritingASplitPlan(t *testing.T) {
	const h = directorSplitQuestionsHeading
	checkDirectorSkillPlaces(t, "questions before a split plan", []skillPlace{
		{"the subsection " + h, directorSplitSubsection(h), []string{
			"分割計画を書く前に、`/spec` skill の手順 1〜3 と同じ深さで、人と要件を詰める",
			"進め方の詳しいことは `/spec` skill の手順 1〜3 を読む",
			"機能をまたぐ分かれ目は、この段で人と決める",
			"機能の境目、共有のファイルをどの機能が予約するか(`- Reserve:`)、座席の driver と model(前文)、" +
				"機能の順(`- Depends on:` と、どの機能から立てるか)の 4 つ",
			"機能の内側の分かれ目は聞かない。leader が機能の計画の `FORK` の行にし、承認の QUESTION で上げる(「受信箱の処理」の「計画の承認」)",
			"`/spec` の手順 4〜5(下書きと人への確認)は、下の「分割計画の承認」の分割計画の執筆と図解ページの承認が受け持つ",
			"手順 6〜8(出力先、worktree、PR)は取り込まない。director は spec のファイルも worktree も PR も作らない",
			"決まったことは、分割計画の前文か各機能の本文に書く",
		}},
		{"step 1 of the subsection " + h, directorSplitStep(h, 1), []string{
			"依頼を理解する(人に聞かない)",
			"依頼(issue なら `gh issue view <番号>`)、AGENTS.md、`.claude/rules/`、触りそうなコードを読み",
			"分かっていることと、曖昧なことや足りないことを書き出す",
		}},
		{"step 2 of the subsection " + h, directorSplitStep(h, 2), []string{
			"推奨つきの決定木の質問: 決めることを上流から順に、AskUserQuestion で聞く" +
				"(Codex と、wake に push がある director は番号つきの選択肢。「人への聞き方」)",
			"1 回に 4 問まで(AskUserQuestion の上限。`/spec` は 5 問ずつと書くが、director はツールに合わせる)",
			"1 問の選択肢は 2〜4 で、推奨の案を先頭に置いてラベルの末尾に「(推奨)」を付け、説明に理由を 1 文書く",
			"聞く前に、コード、文書、Web で答えが分かるかを確かめ、分かるものは聞かずに調べた結果を記録する",
			"人が問い返したり推奨を退けたりしたら、答えてから決まったことを直し、続ける",
			"人が十分と言うか、問いが新しい決定を生まなくなるまで続ける",
		}},
		{"step 3 of the subsection " + h, directorSplitStep(h, 3), []string{
			"残った曖昧さの確認: 2 のあとに残った曖昧さだけを、2 と同じ形で聞く",
		}},
	})

	specStep := func(n int) func(string) string {
		return func(text string) string {
			return directorSkillNumberedStep(directorSkillSubsection(text, "## Steps"), n)
		}
	}
	checkSkillPlaces(t, specSkillCopies, "steps the /director skill points at", []skillPlace{
		{"step 1 of the /spec skill's ## Steps", specStep(1), []string{"**Understand the request** (internal, no user interaction)"}},
		{"step 2 of the /spec skill's ## Steps", specStep(2), []string{
			"**Interrogate the decision tree with research-backed questions**",
			"Ask questions in batches of five",
		}},
		{"step 3 of the /spec skill's ## Steps", specStep(3), []string{"**Clarify residual requirements**"}},
		{"step 4 of the /spec skill's ## Steps", specStep(4), []string{"**Draft the spec**"}},
		{"step 5 of the /spec skill's ## Steps", specStep(5), []string{"**Final review with user**"}},
		{"step 6 of the /spec skill's ## Steps", specStep(6), []string{"**Output selection**"}},
		{"step 7 of the /spec skill's ## Steps", specStep(7), []string{"**Ensure a spec task worktree before any output write**"}},
		{"step 8 of the /spec skill's ## Steps", specStep(8), []string{"**Execute the chosen path**"}},
	})
}

// TestDirectorSkill_AsksApproveOrNeedsChangesBeforeTheSplitPlanDigest pins
// the /director skill's ### 分割計画の承認 (AC2): the director writes the
// split plan in the ledger, opens the visual page and asks "この分割計画を
// 承認しますか" with the two options Approve and Needs changes
// (AskUserQuestion; numbered options on Codex and with a push wake); on
// Needs changes it puts the human's changes in the plan, redraws the changed
// figures, and asks again; and only on Approve it writes the digest's two
// lines, never without the human's word, asking again after any later edit.
func TestDirectorSkill_AsksApproveOrNeedsChangesBeforeTheSplitPlanDigest(t *testing.T) {
	const h = directorSplitApprovalHeading
	checkDirectorSkillPlaces(t, "split plan approval", []skillPlace{
		{"step 1 of the subsection " + h, directorSplitStep(h, 1), []string{
			"決まったことをもとに、分割計画を台帳の `splits/<id>.md` に書く(書き方は `/org` skill の「分割計画」)",
			"全機能に共通の決まり(座席の driver と model、編集してはいけないファイルなど)は、`## Features` より前の `## ` の節に書く",
		}},
		{"step 2 of the subsection " + h, directorSplitStep(h, 2), []string{
			"`./scripts/plan-visual.sh open <ページ>` で開いて人に見せ、承認するかを聞く",
			"問いは「この分割計画を承認しますか」、選択肢は「Approve」と「Needs changes」の 2 つで、AskUserQuestion で聞く" +
				"(Codex と、wake に push がある director は番号つきの選択肢。「人への聞き方」)",
		}},
		{"step 3 of the subsection " + h, directorSplitStep(h, 3), []string{
			"人が Needs changes と答えたら、直すところを聞いて分割計画に入れ、図解ページの変わった図を作り直し、2 のとおり開いてもう一度聞く",
		}},
		{"step 4 of the subsection " + h, directorSplitStep(h, 4), []string{
			"人が Approve と答えたときだけ、`./scripts/plan-visual.sh digest <分割計画>` の 12 桁で" +
				"先頭の `- Status: Approved` と `- Approved: <日付> sha256:<digest>` の 2 行を書く",
			"人の承認の言葉がないまま、この 2 行を書かない",
			"承認のあとに中身を変えたら、もう一度承認を聞く",
		}},
	})
}

// TestDirectorSkill_RecommendsFilingAnIssueForAFeatureThatOverlapsARunningOrg
// pins the /director skill's ### 機能ごとの出力先 (AC3, D1): the director
// reads every running org's reserved: line on ralph status and compares each
// feature's - Reserve: with it by the /org skill's path rules; it asks one
// question per feature (AskUserQuestion, at most 4 a call; numbered options
// on Codex and with a push wake) with the options issue に起票 and 今すぐ着手,
// recommending and listing first issue に起票, with the org_id and paths in
// its description, for a feature that overlaps a running org, and 今すぐ着手
// otherwise; it starts a feature with ralph org start --plan in the order
// settled before the plan; when start refuses an overlap it asks that
// feature again with issue に起票 recommended; and it files an issue only for
// a feature the human chose to file.
func TestDirectorSkill_RecommendsFilingAnIssueForAFeatureThatOverlapsARunningOrg(t *testing.T) {
	const h = directorSplitOutputHeading
	checkDirectorSkillPlaces(t, "where each feature goes", []skillPlace{
		{"step 1 of the subsection " + h, directorSplitStep(h, 1), []string{
			"`ralph status` を打ち、走っている org ごとの `reserved:` の行を読む",
			"各機能の `- Reserve:` を、その予約と `/org` skill の「担当範囲の予約」の規則" +
				"(末尾が `/` ならディレクトリの下すべて、そうでなければそのファイル、`.` は repo 全体、パスの区切りの単位で比べる)で比べる",
		}},
		{"step 2 of the subsection " + h, directorSplitStep(h, 2), []string{
			"機能ごとに出力先を AskUserQuestion で聞く(1 機能を 1 問にし、1 回に 4 問まで。" +
				"Codex と、wake に push がある director は番号つきの選択肢。「人への聞き方」)",
			"選択肢は「issue に起票」と「今すぐ着手(org を start)」",
			"走っている org の予約と重なる機能は「issue に起票」を推奨にして先頭に置き、説明に重なる org_id とパスを書く",
			"重ならない機能は「今すぐ着手」を推奨にする",
		}},
		{"step 3 of the subsection " + h, directorSplitStep(h, 3), []string{
			"今すぐ着手: 機能ごとに org を立てる(`/org` skill の「`ralph org start --plan`」)",
			"`--driver` と `--model` は leader の座席のもので、director の driver とは別に選べる",
			"`- Depends on:` の順は start が強制しないので、「分割計画の前の質問」で人と決めた順に立てる",
			"ralph org start --plan .harness/state/org/splits/<id>.md --feature <slug> --driver claude --model opus",
			"start が予約の重なりで断ったら(エラーが相手の org_id と重なったパスを出す)、" +
				"その機能を「issue に起票」の推奨で、2 のとおり聞き直す",
		}},
		{"step 4 of the subsection " + h, directorSplitStep(h, 4), []string{
			"issue に起票: 下の「起票と記録」。起票するのは、人が「issue に起票」を選んだ機能だけ",
		}},
	})
}

// TestDirectorSkill_FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch
// pins the /director skill's ### 起票と記録 (AC4, D2, D3): the branches to
// wait for are the feature: lines' branches of every overlapping org (all of
// them when there are two or more); an org with no feature: line (ralph org
// start <task>, a promoted leader, --reserve only) has none, which the
// director tells the human and asks about, and a feature that overlaps
// nothing has none either; the issue body is a file under .harness/state/
// written with Write (apply_patch on Codex), holding the feature's body, its
// - Reserve: and - Depends on:, the split plan's id and preamble, and the
// overlapping orgs, and ending with the mark line ralph-split: <id>/<slug>
// after: <branches> (after: none with no branch); the issue is filed with gh
// issue create --body-file and a title without '; one follow-up is recorded
// per branch, not when ralph org followup list already has an open one for
// the same issue and branch, and not when the branch's PR is already merged
// (merge-order --all), with the mark line as its note; and a recording that
// stopped part way is redone from the mark line by ### merge を待つ issue.
// A note of that shape must pass ValidateFollowupAdd, and the note limit and
// the merged class must be the ones the skill names.
func TestDirectorSkill_FilesTheIssueWithAMarkLineAndRecordsAFollowupPerBranch(t *testing.T) {
	const h = directorSplitFilingHeading
	checkDirectorSkillPlaces(t, "filing and recording", []skillPlace{
		{"step 1 of the subsection " + h, directorSplitStep(h, 1), []string{
			"待つブランチを決める。重なる org ごとに `ralph org status --org-id <org>` を打ち、`feature:` の行" +
				"(`feature: <split>/<slug> branch <branch> worktree <path>`)のブランチを読む",
			"重なる org が 2 つ以上なら、そのブランチを全部待つ",
			"`feature:` の行がない org(`ralph org start <task>` の org、昇格した leader の org、`--reserve` だけの org)には" +
				"待つブランチがないので、そのことを人に言い、どうするかを聞く",
			"重ならない機能を人が起票に選んだときも、待つブランチはない",
		}},
		{"step 2 of the subsection " + h, directorSplitStep(h, 2), []string{
			"issue の本文を、cwd の `.harness/state/` の下のファイルに Write ツール(Codex では apply_patch)で書く",
			"本文には、機能の本文(目的と受け入れ条件)、`- Reserve:` と `- Depends on:`、分割計画の id と前文の要点、重なる org とパスを書く",
			"最後の行に、次の印の行を行頭から書く(待つブランチがなければ `after: none`)",
			"ralph-split: <分割計画の id>/<slug> after: <ブランチ>,<ブランチ>",
		}},
		{"step 3 of the subsection " + h, directorSplitStep(h, 3), []string{
			"`gh issue create --title '<題>' --body-file <ファイル>` で起票し、出た番号を人に伝える",
			"題に `'` を入れない",
		}},
		{"step 4 of the subsection " + h, directorSplitStep(h, 4), []string{
			"待つブランチごとに follow-up を記録する",
			"記録の前に `ralph org followup list` を読み、同じ issue とブランチの open の件があれば足さない(打ち直しで 2 件にしない)",
			"そのブランチの PR がもう merge されていれば(`ralph org merge-order --all` の `merged`)、記録せず、待ち終えたブランチとして数える",
			"note には印の行をそのまま書く(1 行・500 字まで)",
			"ralph org followup add --issue <番号> --after-branch <ブランチ> --note '<印の行>'",
		}},
		{"step 5 of the subsection " + h, directorSplitStep(h, 5), []string{
			"記録が途中で止まったとき(起票のあと記録の前、1 件目のあと 2 件目の前)は、" +
				"「merge を待つ issue」の手順で、issue の印の行から記録し直す",
		}},
	})

	add := FollowupAdd{Issue: 271, AfterBranch: "feat/auth-token", Note: "ralph-split: auth-rework/auth-cli after: feat/auth-token,fix/auth-store"}
	if err := ValidateFollowupAdd(add); err != nil {
		t.Errorf("ValidateFollowupAdd(%+v) = %v, but the /director skill's %s records the mark line as a follow-up's note", add, err, h)
	}
	if InboxNoteMaxRunes != 500 {
		t.Errorf("InboxNoteMaxRunes = %d, but the /director skill's %s says a follow-up's note is at most 500 characters", InboxNoteMaxRunes, h)
	}
	if MergeClassMerged != "merged" {
		t.Errorf("MergeClassMerged = %q, but the /director skill's %s skips a branch shown `merged` by merge-order --all", MergeClassMerged, h)
	}
}
