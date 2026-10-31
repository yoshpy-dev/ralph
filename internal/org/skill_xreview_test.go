package org

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// The tests in this file pin what the /director and the /org skills say
// about a feature org's cross-review (plan 2026-11-01-org-cross-review,
// issue #269, slice S3; AC6, AC8, D7b, D9): a stall ALERT about the reviewer
// while a cross-review TASK runs is waited out as one while the gate runs;
// the leader's cap QUESTION (the header XREVIEW: cap_reached and one FORK 1:
// line) goes to the human with a, raising the cap, recommended, and the
// answer goes back as a DECISION with a FORK 1: <記号> <案> line; a feature
// PR's RESULT carries a CROSS_REVIEW: line the director reports; the leader
// reads the [org.review] settings and the reviewer picks the other model
// from the implementer's driver; and the /org skill's end of a feature org
// has the cross-review between the rebase and the sync-docs. The words are
// those of the leader template's ### cross-review subsection and the
// reviewer template's ## cross-review の TASK section, which these tests
// check the skills against.
//
// If one of these tests fails after a deliberate change to a rule, put the
// rule's new words, short and verbatim, in its fragment in the skill's order;
// otherwise restore the text in .claude/skills/<org or director>/SKILL.md and
// templates/base/.claude/skills/<org or director>/SKILL.md and run
// scripts/sync-skills.sh for the two .agents copies, as for the first test in
// org_skill_send_sync_test.go.

// directorXreviewCapHeading is the /director skill's subsection for the
// leader's cap QUESTION, and orgSkillXreviewLead the first words of the
// paragraph of item 1 of the /org skill's ### org の終わり方 that hands the
// cross-review over after the rebase and before the sync-docs.
// directorFeatureWorktree is where the /director skill reads a feature org's
// files, and xreviewSyncDocsHead the first words that the sync-docs
// paragraphs of the /org skill and of the leader template share.
const (
	directorXreviewCapHeading = "### cross-review の上限"
	orgSkillXreviewLead       = "rebase が済んだら(要らなかったときも)、sync-docs の前に、leader は cross-review を扱う"
	directorXreviewDecisionAt = "たとえば次のメッセージにする"
	directorFeatureWorktree   = "機能の worktree(`.claude/worktrees/org-<org_id>`)"
	xreviewSyncDocsHead       = "cross-review が終わったら(off で TASK を送らなかったときも)、"
)

// directorXreviewRaisedFork matches the FORK 1: line of the /director
// skill's example DECISION for a raised cap and captures its
// (新しい上限 <数>), the 新しい上限 <数> in it, and the number, which the
// leader reads from that line only (fix F3). directorXreviewInboxID matches
// the inbox ID the example's SUMMARY: line answers, the one number that line
// may hold.
var (
	directorXreviewRaisedFork = regexp.MustCompile(`^FORK 1: a 上限を上げる(\((新しい上限 (\d+))\))$`)
	directorXreviewInboxID    = regexp.MustCompile(`受信箱の e\d+`)
)

// directorSkillStep3Bullet returns the bullet of step 3 of the /director
// skill's ### 件の処理 whose first line holds marker (markdownItem), "" when
// there is none.
func directorSkillStep3Bullet(text, marker string) string {
	item, _ := markdownItem(directorSkillNumberedStep(directorSkillSubsection(text, "### 件の処理"), 3), marker)
	return item
}

// directorXreviewCapStep returns step n of the /director skill's
// subsection directorXreviewCapHeading (directorSkillNumberedStep).
func directorXreviewCapStep(text string, n int) string {
	return directorSkillNumberedStep(directorSkillSubsection(text, directorXreviewCapHeading), n)
}

// TestDirectorSkill_XreviewStallIsWaitedOutLikeTheGate pins D9 and AC6 for
// the ALERT bullet of ### 件の処理: after the reviewer's stall item while the
// gate runs, the same holds while the leader waits on a cross-review TASK
// (TASK_ID xreview-), since the other model's call can take up to 20 minutes,
// in which the reviewer sends nothing; the director reads the reviewer's
// screen and waits while the call runs. Step 12 of the leader template gives
// the call the same length.
func TestDirectorSkill_XreviewStallIsWaitedOutLikeTheGate(t *testing.T) {
	checkDirectorSkillPlaces(t, "cross-review stall", []skillPlace{
		{"the ALERT bullet of ### 件の処理", func(text string) string { return directorSkillStep3Bullet(text, "- ALERT(見張りの escalation)") }, []string{
			"leader が reviewer に gate を頼んでいる間は、gate が長いと reviewer の停滞の件(`SEAT: reviewer`、`CONDITION: stall`)が出ることがある",
			"gate が走っていれば待ち、終わってから上のとおり決める",
			"leader が reviewer に cross-review の TASK(TASK_ID が `xreview-` で始まる TASK)を頼んでいる間も同じで、" +
				"相手のモデルの呼び出しは 20 分まで続くことがあり、そのあいだ reviewer は何も送らない",
			"reviewer の画面で呼び出しが走っていれば待ち、終わってから上のとおり決める",
			"座席や org を止めるか(`ralph org stop`、`ralph org disband`)は人が決める",
		}},
	})
	if step12 := leaderFeatureOrgItem(t, step12Marker); !containsPhrase(step12, "呼び出しは 20 分まで続くことがあり、そのあいだ reviewer は何も送らない") {
		t.Errorf("expected step 12 of the leader template to say the other model's call can take up to 20 minutes, as the /director skill's ALERT bullet does, got item:\n%s", step12)
	}
}

// TestDirectorSkill_XreviewCapQuestionGoesToTheHumanAsAFork pins D7b for the
// /director skill: step 3 of ### 件の処理 sends the QUESTION with the header
// XREVIEW: cap_reached to the subsection directorXreviewCapHeading, which
// comes after ### 計画の承認. The subsection says what the item means (the
// complete runs reached the cap, with ACTION_REQUIRED left or with changes
// the other model has not seen), read from the EVIDENCE: report in the
// feature worktree, as ### 計画の承認 reads the plan there, since the report
// is only on the feature branch (self-review cycle 1 L2). It asks the human
// the FORK 1: line's a, b, and c with a, raising the cap, recommended and
// first, sends the answer as a DECISION whose body has the line
// FORK 1: <記号> <案> after SUMMARY: (with the new cap for a when the human
// gives one), and leaves stopping or disbanding the org to the human when c
// leaves the leader doing nothing. For a it says the leader's rule for the
// new cap (L2): the number is read only from the FORK 1: line's
// 新しい上限 <数>, never from the SUMMARY: line's inbox ID or the text (fix
// F3); a number above the complete runs counted, the <n> of the report's
// - Cycle: line, is used, and a smaller one or none makes it <n> + 1, so the
// human is asked for a number above <n> or none, and a number given goes in
// the FORK 1: line only. Its example DECISION parses as one, has that line
// with (新しい上限 <数>), and no other number than the inbox ID it answers.
// The leader template's ### cross-review subsection asks and reads the same,
// with done: for the complete runs counted and done: + 1 for a cycle's <n>,
// and names the example's (新しい上限 <数>) as the form it reads.
func TestDirectorSkill_XreviewCapQuestionGoesToTheHumanAsAFork(t *testing.T) {
	step := func(n int) func(string) string {
		return func(text string) string { return directorXreviewCapStep(text, n) }
	}
	checkDirectorSkillPlaces(t, "cross-review cap QUESTION", []skillPlace{
		{"step 3 of ### 件の処理", func(text string) string {
			return directorSkillNumberedStep(directorSkillSubsection(text, "### 件の処理"), 3)
		}, []string{
			"機能の計画の承認を求める QUESTION(ヘッダに `PLAN:` と `VISUAL:` がある件): 下の「計画の承認」",
			"cross-review の上限の QUESTION(ヘッダに `XREVIEW: cap_reached` がある件): 下の「cross-review の上限」",
			"RESULT(PR を作った件)",
		}},
		{"the subsection " + directorXreviewCapHeading, func(text string) string { return directorSkillSubsection(text, directorXreviewCapHeading) }, []string{
			"機能の org の leader は、`[org.review].cross_review` が on のとき、reviewer に cross-review",
			"(`/org` skill の「org の終わり方」の 1)",
			"ヘッダに `XREVIEW: cap_reached` がある件がそれで、本文に `SUMMARY:` の行、`FORK 1:` の 1 行、最後の complete の report のパスの `EVIDENCE:` の行がある",
		}},
		{"step 1 of ### 計画の承認", func(text string) string {
			return directorSkillNumberedStep(directorSkillSubsection(text, "### 計画の承認"), 1)
		}, []string{directorFeatureWorktree + "で"}},
		{"step 1 of " + directorXreviewCapHeading, step(1), []string{
			"件の意味を人に伝える",
			"相手のモデルが所見を返して triage まで終わった回(complete の回)が上限(`[org.review].max_cross_review_cycles` と、人が前に上げた値の大きいほう)に届き",
			"`SUMMARY:` の行と `EVIDENCE:` の report を読み、どちらなのかと要点を伝える",
			"report は機能のブランチにしかないので、" + directorFeatureWorktree + "で読む",
			"最後の complete の回の ACTION_REQUIRED が、直されないまま残っている",
			"最後の complete の回のあとに、相手のモデルに見せていない変更がある",
		}},
		{"step 2 of " + directorXreviewCapHeading, step(2), []string{
			"`FORK 1: <問い> / a 上限を上げる(推奨) / b <進む案> / c 止める`",
			"AskUserQuestion(Codex では番号つきの選択肢)で、選択肢を行の順(推奨の a が最初)に並べる",
			"`known gaps として PR に書いて進む`",
			"`見せていない変更を PR に書いて進む`",
			"`ralph org inbox notify <件の ID>` で人に送るだけにする",
			"人の答えのないまま選ばない",
		}},
		{"step 3 of " + directorXreviewCapHeading, step(3), []string{
			"人の答えを「答える順」のとおり DECISION で leader に送り、resolve する",
			"本文の `SUMMARY:` の行のあとに、選んだ案を `FORK 1: <記号> <案>` の 1 行で書く",
			"leader はこの行の記号で案を選び、`SUMMARY:` の文や先頭の数字では選ばない",
			"a なら、新しい上限の回数を人に聞く",
			"leader は数を `FORK 1:` の行の `新しい上限 <数>` からだけ読み、ほかの数(`SUMMARY:` の行の受信箱の ID `e5` の `5`、文の中の数)は使わない",
			"その数が上限に数えた complete の回の数(`EVIDENCE:` の report の `- Cycle: <n>/<上限>` の行の `<n>`)より大きければ" +
				"その数を新しい上限にし、`<n>` 以下か、`FORK 1:` の行に `新しい上限 <数>` がなければ `<n>` + 1 にする",
			"そのため人には、`<n>` より大きい数を言うか、数を言わずに `<n>` + 1 にするかを聞く",
			"人が数を言えば、数は `FORK 1:` の行にだけ `新しい上限 <数>` の形で書き(下の例の `(新しい上限 3)`)、`SUMMARY:` の文には書かない",
			"言わなければ `FORK 1:` の行に数を書かない",
			directorXreviewDecisionAt,
		}},
		{"step 4 of " + directorXreviewCapHeading, step(4), []string{
			"c なら、leader はそれ以上何もしない(sync-docs にも push にも進まず、PR を作らず、disband もしない)",
			"org を止めるか(`ralph org stop`、`ralph org disband`)は人が決め、director は人に頼まれないまま打たない",
		}},
	})
	checkSkillHeadingOrder(t, directorSkillCopies, []string{"### 計画の承認", directorXreviewCapHeading, "## 進捗の報告"})

	sub := leaderXreviewSubsection(t)
	root := directorSkillRepoRoot(t)
	for _, rel := range directorSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's cap DECISION check (vendored repo?)", rel, err)
		}
		step3 := directorXreviewCapStep(string(data), 3)
		if containsPhrase(step3, "今の上限 + 1") {
			t.Errorf("%s: expected step 3 of %s to no longer make the new cap the current cap + 1, which the leader's done: + 1 replaced, got step:\n%s", rel, directorXreviewCapHeading, step3)
		}
		if old := "leader は、DECISION の数が"; containsPhrase(step3, old) {
			t.Errorf("%s: expected step 3 of %s to say the leader reads the new cap only from the FORK 1: line, not %q, which reads as any number in the DECISION", rel, directorXreviewCapHeading, old)
		}
		if !strings.Contains(step3, directorXreviewDecisionAt) {
			t.Errorf("%s: expected step 3 of %s to show an example DECISION after %q, got step:\n%s", rel, directorXreviewCapHeading, directorXreviewDecisionAt, step3)
			continue
		}
		msg := fencedBlockAfter(t, step3, directorXreviewDecisionAt, rel+": step 3 of "+directorXreviewCapHeading)
		m, err := protocol.Parse(msg)
		if err == nil {
			err = protocol.Validate(m, protocol.DefaultMaxBodyChars)
		}
		if err != nil || m.Type != protocol.TypeDecision {
			t.Errorf("%s: expected the example to be a valid DECISION, got TYPE %q, err %v; message:\n%s", rel, m.Type, err, msg)
			continue
		}
		if !strings.HasPrefix(m.Body, "SUMMARY: ") {
			t.Errorf("%s: expected the example DECISION's body to start with SUMMARY:, got body:\n%s", rel, m.Body)
		}
		var forks []string
		for _, line := range strings.Split(m.Body, "\n") {
			if strings.HasPrefix(line, "FORK") {
				forks = append(forks, line)
				continue
			}
			if rest := directorXreviewInboxID.ReplaceAllString(line, ""); strings.Contains(line, "新しい上限") || strings.ContainsAny(rest, "0123456789") {
				t.Errorf("%s: expected the example DECISION to write the new cap only in its FORK 1: line, got a number outside the inbox ID in %q; body:\n%s", rel, line, m.Body)
			}
		}
		if len(forks) != 1 || !directorXreviewRaisedFork.MatchString(forks[0]) {
			t.Errorf("%s: expected the example DECISION's body to have one FORK 1: a 上限を上げる(新しい上限 <数>) line, got %q; body:\n%s", rel, forks, m.Body)
			continue
		}
		// The leader's rule names the form the example writes: the example's
		// (新しい上限 <数>) as it is, and with its number as <数>.
		raised := directorXreviewRaisedFork.FindStringSubmatch(forks[0])
		for _, want := range []string{raised[1], "`" + strings.Replace(raised[2], raised[3], "<数>", 1) + "`"} {
			if !containsPhrase(sub, want) {
				t.Errorf("%s: expected %s to read the new cap as %q, the form the example DECISION's FORK 1: line writes, got:\n%s", rel, leaderXreviewWhere, want, sub)
			}
		}
	}

	for _, want := range []string{
		"ヘッダには `XREVIEW: cap_reached` の 1 行を書く",
		"`FORK 1: <問い> / a 上限を上げる(推奨) / b <進む案> / c 止める`",
		"1 つ目の QUESTION では `known gaps として PR に書いて進む`、2 つ目では `見せていない変更を PR に書いて進む`",
		"答えは DECISION の本文の `FORK 1: <記号> <案>` の行で読み",
		"数は DECISION の `FORK 1:` の行にある `新しい上限 <数>`",
		"からだけ読み、ほかの数(`SUMMARY:` の行の受信箱の ID `e5` の `5`、文の中の数)は使わない",
		"`FORK 1:` の行に `新しい上限 <数>` があればその数",
		"その数が `done:` 以下なら `done:` + 1",
		"なければ `done:` + 1 にする",
		"`CYCLE: <n>/<上限>`: `<n>` は `done:` + 1",
		"c なら、それ以上何もしない",
		"org を止めるかどうかは人が決める",
	} {
		if !containsPhrase(sub, want) {
			t.Errorf("expected %s to say %q, as the /director skill's %s reads it, got:\n%s", leaderXreviewWhere, want, directorXreviewCapHeading, sub)
		}
	}
	if reviewer := reviewerXreviewSection(t); !containsPhrase(reviewer, "- Cycle: <CYCLE の値>") || !containsPhrase(reviewer, "`CYCLE: <n>/<上限>`") {
		t.Errorf("expected the reviewer template's report to have the line - Cycle: <n>/<上限>, from which the /director skill's %s reads the cap, got:\n%s", directorXreviewCapHeading, reviewer)
	}
}

// TestDirectorSkill_ResultItemReportsTheCrossReviewLine pins D10 for the
// RESULT bullet of ### 件の処理: after the follow-up lines, the RESULT's body
// carries the PR body's Cross-review: value as a CROSS_REVIEW: line (ran,
// skipped, off included, or incomplete with the reason), which the director
// reports with the PR, since skipped and incomplete mean no findings came
// back and a codex reviewer seat without network may do so every time
// (unverified, the plan's Risks). Step 14 of the leader template writes the
// line.
func TestDirectorSkill_ResultItemReportsTheCrossReviewLine(t *testing.T) {
	checkDirectorSkillPlaces(t, "RESULT cross-review", []skillPlace{
		{"the RESULT bullet of ### 件の処理", directorSkillResultBullet, []string{
			"その行も PR の URL と一緒に人に報告する",
			"本文には、PR の本文の `Cross-review:` の行と同じ値の `CROSS_REVIEW: <値>` の 1 行も載る",
			"値は `ran (<相手>)`、`skipped (<理由>)`(`[org.review].cross_review` が off のときも)、`incomplete (<理由>)` のどれかで、" +
				"PR を報告するときに理由と一緒に人に伝える",
			"skipped と incomplete は、相手のモデルの所見が出ていないことを指す",
			"codex の reviewer 座席は、network がないと相手の CLI を呼べず、毎回 skipped か incomplete になりうる(未確認)",
			"leader はそのあと disband するので答えは送らず",
		}},
	})
	step14 := leaderFeatureOrgItem(t, leaderStep14Marker)
	for _, name := range []string{"`CROSS_REVIEW: <値>`", "`- Cross-review:`", "`ran (<相手>)`", "`skipped (<理由>)`", "`incomplete (<理由>)`"} {
		if !containsPhrase(step14, name) {
			t.Errorf("expected step 14 of the leader template to name %s, as the /director skill's RESULT bullet does, got item:\n%s", name, step14)
		}
	}
}

// TestOrgSkill_PrerequisitesSayTheLeaderReadsTheReviewSettings pins AC8 for
// the /org skill's ## 前提: the pane's RALPH_PRIMARY_CLI feeds pick_reviewer
// for the leader's plan advisory only, and the reviewer seat's cross-review
// passes the implementer's driver from the TASK instead; the feature org's
// leader reads [org.review] with ralph config get in the worktree,
// plan_advisory before the plan advisory and cross_review and
// max_cross_review_cycles at step 9 after the rebase, and the reviewer seat
// reads no setting. The text for a later version and for a seat that reads
// the setting itself is gone. The templates read the same commands.
func TestOrgSkill_PrerequisitesSayTheLeaderReadsTheReviewSettings(t *testing.T) {
	checkOrgSkillPlaces(t, "review settings", []skillPlace{
		{"the section ## 前提", func(text string) string { return directorSkillSubsection(text, "## 前提") }, []string{
			"3 つ目の値は座席の driver(`claude` か `codex`)で、機能の org の leader の計画助言がもう一方の CLI を選ぶときに使う " +
				"`scripts/xreview-helpers.sh` の `pick_reviewer` に、leader の driver を読ませる",
			"reviewer 座席の cross-review はこの値を読まず、leader が TASK の `IMPLEMENTER_DRIVER:` の行に書いた implementer の driver を " +
				"`pick_reviewer` に引数で渡して相手を選ぶ",
			"leader の計画助言は、変数がなければ leader 自身の CLI で相手を選ぶので、相手を取り違えるのは",
			"`[org.review]` は機能ごとの org の leader の計画助言と cross-review の有無(既定はどちらも off)と、" +
				"cross-review を回す回数の上限(最初の 1 回を含む。既定 2)で、leader が worktree で `ralph config get <key>` を打って読む",
			"計画助言は頼む前に `plan_advisory` を、cross-review は役割の雛形の手順 9 で rebase のあとに `cross_review` と `max_cross_review_cycles` を読む" +
				"(`ralph config get org.review.cross_review` など)",
			"reviewer 座席は設定を読まず、leader の TASK に書かれた implementer の driver から `pick_reviewer` で相手のモデルを選ぶ(「org の終わり方」の 1)",
		}},
	})

	root := directorSkillRepoRoot(t)
	for _, rel := range orgSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's review settings check (vendored repo?)", rel, err)
		}
		section := directorSkillSubsection(string(data), "## 前提")
		for _, gone := range []string{
			"読み始めるのは後の版から",
			"`pick_reviewer` に、座席の driver を読ませる",
			"codex の座席の cross-review がもう一方の CLI を取り違えることがある",
		} {
			if containsPhrase(section, gone) {
				t.Errorf("%s: expected ## 前提 to no longer say %q, which the leader's and the reviewer's cross-review replaced, got section:\n%s", rel, gone, section)
			}
		}
	}

	if got := config.Default().Org.Review.MaxCrossReviewCycles; got != 2 {
		t.Errorf("config.Default().Org.Review.MaxCrossReviewCycles = %d, but the /org skill's ## 前提 says the cap defaults to 2", got)
	}
	sub := leaderXreviewSubsection(t)
	for _, want := range []string{"ralph config get org.review.cross_review", "ralph config get org.review.max_cross_review_cycles"} {
		if !containsPhrase(sub, want) {
			t.Errorf("expected %s to read the setting with %q, as the /org skill's ## 前提 says, got:\n%s", leaderXreviewWhere, want, sub)
		}
	}
	if reviewer := reviewerXreviewSection(t); !containsPhrase(reviewer, `r=$(pick_reviewer "<IMPLEMENTER_DRIVER>")`) {
		t.Errorf("expected the reviewer template's cross-review to pass the implementer's driver to pick_reviewer, as the /org skill's ## 前提 says, got:\n%s", reviewer)
	}
}

// TestOrgSkill_EndOfOrgStep4WaitsOutAnXreviewStallLikeTheGate pins D9 and
// AC6 for item 4 of ### org の終わり方: after the stall ALERT about the
// reviewer while the gate runs, a stall ALERT while item 1's cross-review
// TASK runs is treated the same, and the leader waits while the other
// model's call runs (up to 20 minutes, in which the reviewer sends nothing).
func TestOrgSkill_EndOfOrgStep4WaitsOutAnXreviewStallLikeTheGate(t *testing.T) {
	checkOrgSkillPlaces(t, "cross-review stall", []skillPlace{
		{"item 4 of ### org の終わり方", func(text string) string { return orgSkillEndOfOrgStep(text, 4) }, []string{
			"この間に見張りから reviewer の停滞の `ALERT`(`SEAT: reviewer`、`CONDITION: stall`)が来ても、reviewer を stop も spawn し直しもせず",
			"gate が走っていれば待つ(gate は走っているあいだ何も記録せず、Codex の reviewer は前面で打つので、画面が長く変わらないことがある)",
			"1 の cross-review の TASK の返事を待つあいだに同じ `ALERT` が来たときも同じく扱い、" +
				"reviewer の画面で相手のモデルの呼び出しが走っていれば待つ(呼び出しは 20 分まで続くことがあり、そのあいだ reviewer は何も送らない)",
			"gate の TASK への `GATE: pass` の `RESULT` は 1 ではなくここで扱う",
		}},
	})
}

// TestOrgSkill_EndOfOrgSyncDocsStartsOnceTheXreviewIsOver pins self-review
// cycle 1 L3 for item 1 of the /org skill's ### org の終わり方: the sync-docs
// paragraph starts once the cross-review is over (off included), with the
// same first words as the sync-docs paragraph of step 9 of the leader
// template (syncDocsHandOff), so a reader who starts again from that
// paragraph does not take the sync-docs to come right after the rebase and
// skip the cross-review. No copy keeps the old first words.
func TestOrgSkill_EndOfOrgSyncDocsStartsOnceTheXreviewIsOver(t *testing.T) {
	for _, c := range []struct{ name, text string }{
		{"orgSkillSyncDocsHandOff", orgSkillSyncDocsHandOff},
		{"syncDocsHandOff", syncDocsHandOff},
	} {
		if !strings.HasPrefix(c.text, xreviewSyncDocsHead) {
			t.Errorf("expected %s to start with %q, got %q", c.name, xreviewSyncDocsHead, c.text)
		}
	}
	checkOrgSkillPlaces(t, "sync-docs after the cross-review", []skillPlace{
		{"item 1 of ### org の終わり方", func(text string) string { return orgSkillEndOfOrgStep(text, 1) }, []string{
			orgSkillXreviewLead,
			"cross-review が終わったら、この回の report がコミットしてあり `git status --porcelain` が何も出さないことを確かめてから、次の段落の sync-docs に進む",
			orgSkillSyncDocsHandOff,
		}},
	})

	root := directorSkillRepoRoot(t)
	for _, rel := range orgSkillCopies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's old sync-docs head check (vendored repo?)", rel, err)
		}
		if old := "rebase が済んだら(要らなかったときも)、leader は sync-docs を"; containsPhrase(orgSkillEndOfOrgStep(string(data), 1), old) {
			t.Errorf("%s: expected item 1 of ### org の終わり方 to no longer start the sync-docs paragraph with %q, which reads as right after the rebase", rel, old)
		}
	}
	if !containsPhrase(leaderFeatureOrgItem(t, step9Marker), syncDocsHandOff) {
		t.Errorf("expected step 9 of the leader template to say %q, as item 1 of the /org skill's ### org の終わり方 does", syncDocsHandOff)
	}
}

// TestOrgSkill_EndOfOrgStep1HandsTheXreviewBeforeTheSyncDocs pins AC2, AC5,
// AC5b, D3, D7, D7b, and D8 for the /org skill: a paragraph of item 1 of
// ### org の終わり方, after the rebase and before the sync-docs paragraphs
// (which start once the cross-review is over,
// TestOrgSkill_EndOfOrgSyncDocsStartsOnceTheXreviewIsOver), reads the two
// settings, keeps Cross-review: skipped for item 6 when off, sends an
// xreview- TASK with the implementer's driver and the report path when on,
// has the reviewer pick the other model with pick_reviewer, commits each
// report, counts only the
// complete runs against the cap, the larger of max_cross_review_cycles and
// the cap file, escalates the cap QUESTION (XREVIEW: cap_reached, FORK 1:
// with a recommended) instead of a TASK, and goes on to the sync-docs with
// the report committed and git status empty. Item 6 puts the value in the PR
// body's Cross-review: line and the RESULT's CROSS_REVIEW: line, and the
// summaries in ## Leader 運用 2 経路 and ## permission 作法 put the
// cross-review between the rebase and the sync-docs. The names are the
// leader template's (rendered with org-a for <org_id>) and the reviewer
// template's.
func TestOrgSkill_EndOfOrgStep1HandsTheXreviewBeforeTheSyncDocs(t *testing.T) {
	checkOrgSkillPlaces(t, "cross-review hand-off", []skillPlace{
		{"item 1 of ### org の終わり方", func(text string) string { return orgSkillEndOfOrgStep(text, 1) }, []string{
			"`git rebase origin/<既定のブランチ>` を打つ",
			"fetch が失敗したら(ネットワークのない sandbox など)、打ち直さずに `BLOCKED` を escalate する",
			orgSkillXreviewLead,
			"(leader の雛形の「機能ごとの org」の節の「cross-review」の小節)",
			"worktree で `ralph config get org.review.cross_review` と `ralph config get org.review.max_cross_review_cycles` を打ち",
			"off なら(`config get` のない古い ralph と、`ralph` がないときも)TASK を送らず、6 のために `Cross-review: skipped (<理由>)` を控える",
			"on なら、reviewer に TASK_ID が `xreview-` で始まる cross-review の TASK を送る",
			"本文の行には、implementer の driver(`IMPLEMENTER_DRIVER:`)、base、何回目かと上限、report のパス " +
				"`docs/reports/review-xreview-<YYYY-MM-DD>-<org_id>-c<k>.md` などを書く",
			"reviewer はその driver を `pick_reviewer` に渡して implementer と違う系列のモデル(claude なら codex、codex なら claude)を選び、" +
				"このブランチの差分を読むだけの形で見せて、所見を triage した report を書いて返す",
			"相手のモデルの呼び出しは 20 分まで続くことがあり、返事が来るまで leader は worktree を変えず、implementer に TASK を送らない",
			"返事を受けるたびに report をコミットする",
			"ACTION_REQUIRED のうち直すものは implementer に TASK で頼み(per-plan では `PLAN:` のヘッダを付ける)、レビューから回し直す",
			"上限に数えるのは、相手が所見を返して triage まで終わった回(report に `- Reviewer status: complete` の行がある回)だけで、" +
				"skipped と incomplete の回は数えず、頼み直さずに進む",
			"上限は `max_cross_review_cycles`(既定 2)と、人が上限を上げたときに leader が書く `.harness/state/xreview-<org_id>-cap` の値の大きいほう",
			"上限に届いていて、ACTION_REQUIRED が残るか、最後の complete の回のあとに相手に見せていない変更があれば、TASK を送らず、" +
				"ヘッダに `XREVIEW: cap_reached` の行、本文に `FORK 1:` の 1 行(a 上限を上げる、b PR に書いて進む、c 止める の 3 つで、推奨は a)を書いた QUESTION を escalate する",
			"人の答えは DECISION の `FORK 1: <記号> <案>` の行で届く(`/director` skill の「cross-review の上限」)",
			"c なら leader はそれ以上何もせず、sync-docs にも push にも進まず、disband もしない",
			"cross-review が終わったら、この回の report がコミットしてあり `git status --porcelain` が何も出さないことを確かめてから、次の段落の sync-docs に進む",
			orgSkillSyncDocsHandOff,
		}},
		{"item 6 of ### org の終わり方", func(text string) string { return orgSkillEndOfOrgStep(text, 6) }, []string{
			"最後の回で sync-docs を飛ばしたときは、`" + docFollowupKey + ":` の行の代わりに、控えた " + syncDocsSkipLine + " の行を書く",
			"PR の本文には本文の雛形の `- Cross-review:` の行も置き、1 で最後に控えた値(`ran (<相手>)`、`skipped (<理由>)`、`incomplete (<理由>)` のどれか)と、" +
				"直さなかった所見と、上限の QUESTION で人が進むと答えたときの known gaps か相手に見せていない変更を書く",
			"RESULT の本文にも、同じ値を `CROSS_REVIEW: <値>` の 1 行で書く",
		}},
		{"item 7 of ## Leader 運用 2 経路", func(text string) string {
			return directorSkillNumberedStep(directorSkillSubsection(text, "## Leader 運用 2 経路"), 7)
		}, []string{
			"reviewer のレポートをコミットして既定のブランチに rebase し、`[org.review].cross_review` が on なら reviewer に cross-review を頼んでその report をコミットし、" +
				"sync-docs を doc-maintainer に任せてコミットし",
		}},
		{"the end-of-task bullet of ## permission 作法", func(text string) string {
			item, _ := markdownItem(directorSkillSubsection(text, "## permission 作法"), "- タスク終了時は必ず")
			return item
		}, []string{
			"reviewer のレポートのコミット → rebase → cross-review の report のコミット(on のとき)→ sync-docs のコミット → report →",
		}},
	})

	sub := leaderXreviewSubsection(t)
	for _, name := range []string{
		"ralph config get org.review.cross_review",
		"`ralph config get org.review.max_cross_review_cycles`",
		"`Cross-review: skipped (<理由>)`",
		"`xreview-`",
		"`IMPLEMENTER_DRIVER: <claude|codex>`",
		"`docs/reports/review-xreview-<YYYY-MM-DD>-<org_id>-c<k>.md`",
		"`- Reviewer status: complete`",
		"`.harness/state/xreview-<org_id>-cap`",
		"`XREVIEW: cap_reached`",
		"`FORK 1: <記号> <案>`",
		"(per-plan では `PLAN:` のヘッダを付ける)",
	} {
		if inTemplate := strings.ReplaceAll(name, "<org_id>", "org-a"); !containsPhrase(sub, inTemplate) {
			t.Errorf("expected %s to name %s, as item 1 of the /org skill's ### org の終わり方 does, got:\n%s", leaderXreviewWhere, inTemplate, sub)
		}
	}
	if !containsPhrase(leaderFeatureOrgItem(t, step9Marker), leaderXreviewStep9) {
		t.Errorf("expected step 9 of the leader template to hand the cross-review over after the rebase and before the sync-docs, as item 1 of the /org skill's ### org の終わり方 does")
	}
	if reviewer := reviewerXreviewSection(t); !containsPhrase(reviewer, "pick_reviewer") || !containsPhrase(reviewer, "`IMPLEMENTER_DRIVER:` の値を引数で渡す") {
		t.Errorf("expected the reviewer template's cross-review to pick the other model with pick_reviewer from IMPLEMENTER_DRIVER, as the /org skill says, got:\n%s", reviewer)
	}
}
