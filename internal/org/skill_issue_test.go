package org

import (
	"strconv"
	"strings"
	"testing"
)

// The tests in this file pin what the /director and /org skills say about a
// feature org started for an issue with `ralph org start --plan --issue`
// (plan 2026-11-22-org-start-issue, AC9): the director starts the feature
// of a filed issue with the issue's number, the leader writes
// `Closes #<number>` in the PR, and after the merge (by `ralph org merge`,
// or on GitHub and then `ralph org cleanup`) and at a handover the director
// reads the branch's `binding.issue` from `ralph org merge-order --all
// --json` and tells the human about an issue still open, without closing it.
// The /org skill describes the flag, the status line, the binding's issue,
// the refusals and the reseat.
//
// If one of these tests fails after a deliberate change to a rule, put the
// rule's new words, short and verbatim, in its fragment in the skill's
// order; otherwise restore the text in .claude/skills/<director or
// org>/SKILL.md and templates/base/.claude/skills/<director or
// org>/SKILL.md and run scripts/sync-skills.sh for the two .agents copies,
// as for the first test in director_skill_sync_test.go.

// directorMergeIssueBullet returns the bullet of the /director skill's
// ### merge と auto_merge that says how to check a filed issue after its
// PR's merge, "" when there is none.
func directorMergeIssueBullet(text string) string {
	item, _ := markdownItem(directorSkillSubsection(text, "### merge と auto_merge"), "起票した機能の PR の merge を人に報告するときは")
	return item
}

// directorFollowupHandover returns the part of the /director skill's
// ### merge を待つ issue before its numbered steps: the handover.
func directorFollowupHandover(text string) string {
	intro, _, _ := strings.Cut(directorSkillSubsection(text, followupSkillHeading), "\n1. ")
	return intro
}

// TestDirectorSkill_StartsAFiledIssueWithItsNumberAndChecksTheIssueClosed
// pins the /director skill: step 3 of ### 機能ごとの出力先 starts the
// feature of a filed issue with --issue and the issue's number, and a
// restart with the same --issue; step 3 of ### merge を待つ issue, the
// start of an issue the human decided to take, passes it; the bullet of
// ### merge と auto_merge reads the branch's binding.issue from
// merge-order's JSON after a merge, reads the issue's state with gh, adds a
// closed issue to the report, and tells the human of an open one without
// closing it; step 2 of ### GitHub で merge した PR の後始末 does the same
// check for every branch it cleaned up; and the handover of
// ### merge を待つ issue checks the issues of the merged branches the same
// way, except for a branch a running org holds.
func TestDirectorSkill_StartsAFiledIssueWithItsNumberAndChecksTheIssueClosed(t *testing.T) {
	checkDirectorSkillPlaces(t, "issue of a feature org", []skillPlace{
		{"step 3 of the subsection " + directorSplitOutputHeading, directorSplitStep(directorSplitOutputHeading, 3), []string{
			"issue に起票した機能を立てるとき(「merge を待つ issue」の 3 で着手を決めた issue)は、`--issue <その issue の番号>` を付ける" +
				"(`#` を付けない数だけ)",
			"leader はタスクの issue の行を読んで PR の本文に `Closes #<番号>` を書き、PR が merge されると GitHub がその issue を閉じる",
			"止めた org(disband していない)を同じ start で立て直すときも、最初の start と同じ `--issue` を付ける(違う番号か、付け外しは断られる)",
			"ralph org start --plan .harness/state/org/splits/<id>.md --feature <slug> --issue <番号>",
		}},
		{"step 3 of the subsection " + followupSkillHeading, directorSplitStep(followupSkillHeading, 3), []string{
			"「機能ごとの出力先」の 3 のとおり、`--issue <その issue の番号>` を付けて start する" +
				"(leader が PR の本文に `Closes #<番号>` を書き、PR の merge で issue が閉じる)",
			"issue を閉じるのは人が頼んだときだけ",
		}},
		{"the filed issue bullet of the subsection ### merge と auto_merge", directorMergeIssueBullet, []string{
			"起票した機能の PR の merge を人に報告するときは、その issue が閉じたかも確かめる",
			"`ralph org merge-order --all --json` の `entries` から、そのブランチ(`branch`)の `binding.issue` を読む" +
				"(`--issue` で立てた org のブランチにだけあり、org の disband のあとも出る)",
			"`gh issue view --repo github.com/<owner>/<repo> <番号> --json state --jq .state` を打つ(`--repo` の決め方は「起票と記録」の終わり)",
			"`CLOSED` なら、merge の報告に issue が閉じたことを添える",
			"`OPEN` なら、issue を閉じずに、番号と open のままであることを人に伝える",
			"issue を閉じるのは人が頼んだときだけ(「merge を待つ issue」の 3)",
		}},
		{"step 2 of the subsection ### GitHub で merge した PR の後始末", directorSplitStep("### GitHub で merge した PR の後始末", 2), []string{
			"`CLEANED:` の行を人に報告する",
			"片付けたブランチごとに、「merge と auto_merge」の箇条のとおり、起票した issue が閉じたかも確かめて伝える(`binding.issue` があるときだけ)",
		}},
		{"the handover of the subsection " + followupSkillHeading, directorFollowupHandover, []string{
			"引き継ぎでは、起票した機能の issue が閉じたかも確かめる",
			"`ralph org merge-order --all --json` の `entries` のうち、`class` が `merged` で `binding.issue` のあるブランチごとに",
			"「merge と auto_merge」の箇条のとおり issue の state を読み、`OPEN` のままなら、閉じずに、ブランチと issue の番号を人に伝える",
			"「起票と記録」の 4 のただし書きのとおり merge 済みと数えず、確かめない",
		}},
	})
}

// TestOrgSkill_DescribesStartIssueStatusLineBindingAndReseat pins the /org
// skill: the verb-table row of start describes --issue (the number alone,
// with the digit limit of maxIssueDigits, needs --plan, recorded in the
// binding, the task's issue line featureIssueLead, the restart that needs
// the recorded --issue, the reseat that passes it), the row of status the
// issue line and JSON key, and the row of merge-order the binding's issue;
// ### `ralph org start --plan` names the issue in the binding and in the
// task, with an example and what the leader writes; ### start が拒否するもの
// lists the refused values and the restart with another --issue; steps 1
// and 4 of ### 座席の立て直し have the start hint and the reseated leader
// carry the recorded issue; and ### merge を待つ issue starts a taken issue
// with --issue.
func TestOrgSkill_DescribesStartIssueStatusLineBindingAndReseat(t *testing.T) {
	checkOrgSkillPlaces(t, "issue of a feature org", []skillPlace{
		{"the verb-table row of start", func(text string) string { return orgSkillVerbRow(text, "start") }, []string{
			"`--issue <番号>`(`--plan` が要る)は、issue に起票した機能を立てるときに、その issue の番号を渡す" +
				"(`#` を付けない数だけ。1 から始まる " + strconv.Itoa(maxIssueDigits) + " 桁まで。ほかの形は何も作らずに断る)",
			"番号は機能との結びつきに記録され、leader のタスクに `" + featureIssueLead + "` の行が入り、leader は PR の本文に `Closes #<番号>` を書く",
			"disband していない org のやり直しの start は、記録と同じ `--issue`(記録になければなし)のときだけ通り、" +
				"違う番号か付け外しは記録と今の値を名指して断られる",
			"`ralph org reseat` は記録の番号で leader を立てる",
		}},
		{"the verb-table row of status", func(text string) string { return orgSkillVerbRow(text, "status") }, []string{
			"`--issue` で立てた org には、`feature:` の行の次に `issue: #<番号>` の行も出す" +
				"(`--json` は `feature` の `issue` で、番号の数。issue のない org にはキーがない)",
		}},
		{"the verb-table row of merge-order", func(text string) string { return orgSkillVerbRow(text, "merge-order") }, []string{
			"ブランチごとに、最新の結びつきの org_id と機能を添える(`--json` の `binding`。" +
				"`--issue` で立てた org なら、その issue の番号 `issue` も入る。org の disband のあとも出て、テキストには出ない)",
		}},
		{"the subsection ### `ralph org start --plan`", directorSplitSubsection("### `ralph org start --plan`"), []string{
			"機能ごとの digest、共有のパス、`--issue` の番号)が入る",
			"依存・issue(`--issue` を渡したときだけ)・承認方式",
			"issue に起票した機能を立てるときは、`--issue <番号>` を付ける",
			"ralph org start --plan .harness/state/org/splits/auth.md --feature auth-token --issue 12",
			"タスクの依存の行の次に issue の行が入り、leader は PR の本文の普通の文(コードブロック、HTML のコメント、引用の外)に `Closes #12` を書く",
			"`ralph org status --org-id <org_id>` の `issue: #12` の行と、`ralph org merge-order --json` のそのブランチの `binding.issue`" +
				"(disband のあとも出る)で読める",
			"止めた org を同じ start で立て直すときは、同じ `--issue` を付ける",
		}},
		{"the subsection ### start が拒否するもの", directorSplitSubsection("### start が拒否するもの"), []string{
			"`--issue` の値が issue の番号の形でない(空、`0`、先頭の 0、`#12`、数でない、" + strconv.Itoa(maxIssueDigits+1) + " 桁以上)か、" +
				"`--plan` のない `--issue`",
			"disband していない org の立て直しで `--issue` が記録と違う(違う番号、記録にある番号を付けない、記録にない番号を付ける)ときも、" +
				"記録と今の値を名指して断る",
		}},
		{"step 1 of the subsection ### 座席の立て直し", directorSplitStep("### 座席の立て直し", 1), []string{
			"leader の `start --plan` の打ち方(結びつきに issue の番号があれば `--issue` も付く)を出す",
		}},
		{"step 4 of the subsection ### 座席の立て直し", directorSplitStep("### 座席の立て直し", 4), []string{
			"結びつきに issue の番号があれば(`--issue` で立てた org)、leader はその番号で立ち、タスクに issue の行が入る",
		}},
		{"the subsection " + followupSkillHeading, directorSplitSubsection(followupSkillHeading), []string{
			"着手すると決めた issue の機能の org は、`start --plan` に `--issue <番号>` を付けて立てる",
			"leader が PR の本文に `Closes #<番号>` を書くので、PR の merge で issue が閉じる",
			"閉じたかは、director が merge のあとに確かめる(`/director` skill の「merge と auto_merge」)",
		}},
	})
}
