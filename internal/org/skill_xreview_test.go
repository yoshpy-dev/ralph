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
// issue #269, slice S3; AC6, AC8, D7b, D9; rewritten by plan
// 2026-11-22-org-xreview-own-model, issue #360, slice S3, AC7, for the
// reviewer seat to review on its own model): a stall ALERT about the
// reviewer while a cross-review TASK runs is waited out only while the
// reviewer's screen moves, a still screen being waited out for the gate alone
// (that plan's self-review M1); the leader's cap QUESTION (the header
// XREVIEW: cap_reached and one FORK 1: line) goes to the human with a,
// raising the cap, recommended, and the answer goes back as a DECISION with
// a FORK 1: <記号> <案> line; a feature PR's RESULT carries a CROSS_REVIEW:
// line the director reports; the leader reads the [org.review] settings,
// and the reviewer seat reads the branch's diff on its own model, the
// implementer's family or not, calling no other model; a human who wants
// another family's review sets [org.role_models].reviewer to the other
// driver; and the /org skill's end of a feature org has the cross-review
// between the rebase and the sync-docs. The words are those of the leader
// template's ### cross-review subsection and the reviewer template's
// ## cross-review の TASK section, which these tests check the skills
// against.
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

// xreviewStallGateOnly is how step 12 of the leader template, item 4 of the
// /org skill's ### org の終わり方, and the /director skill's ALERT bullet say
// that only the gate is waited out with a still screen, and xreviewStallWait
// how step 12 and item 4 say the reviewer seat is waited for while it reads a
// cross-review's diff: while its screen moves, not with a still one (plan
// 2026-11-22-org-xreview-own-model, self-review M1). The seat calls no other
// model, so nothing keeps its screen still for 20 minutes, and 10 minutes of
// a still screen with no RESULT or BLOCKED is a BLOCKED as for any seat
// (xreviewStallStill and xreviewStallStillSkill).
const (
	xreviewStallGateOnly   = "画面が動かなくても待つのは gate のときだけで、"
	xreviewStallWait       = "reviewer を stop も spawn し直しもせずに画面を読み、画面が動いているあいだは待つ(差分が大きいと読むのに時間がかかり、そのあいだ reviewer は何も送らない)"
	xreviewStallStill      = "10 分たっても画面が動かず RESULT も BLOCKED も来なければ、座席の名前を本文に書いて BLOCKED(TASK_ID は `org-a`)を `ralph org escalate` で上げ、返事が来るまで待つ"
	xreviewStallStillSkill = "10 分たっても画面が動かず `RESULT` も `BLOCKED` も来なければ、座席の名前を本文に書いて `BLOCKED` を escalate する"
)

// xreviewRanValue is the Cross-review: value of a complete cross-review as
// step 14 and the ### cross-review subsection of the leader template and the
// /org and /director skills name it: ran with the value of the report's
// Reviewer: line, the reviewer seat's seat_id, driver, and model (plan
// 2026-11-22-org-xreview-own-model, self-review L1).
const xreviewRanValue = "`ran (<report の Reviewer:>)`"

// xreviewStallExempt are phrases of the leader template and the skills from
// before self-review M1 of plan 2026-11-22-org-xreview-own-model, which
// waited out a cross-review's reviewer, as the gate's, with a still screen:
// the re-seat rule's exception for a reviewer that reads the diff, the
// "same as the gate" of step 12, of the cross-review subsection, of item 4
// of the /org skill, and of the /director skill's ALERT bullet, and the
// wait while the reviewer reads, with no word of a moving screen. No copy of
// either skill and no rendered leader template says any of them.
var xreviewStallExempt = []string{
	"cross-review の差分を読んでいるときは、手順 12 と 9 のとおり",
	"手順 12 と 9 のとおり、画面が動かなくても待ちます",
	"返事を待つあいだに同じ ALERT が来たときも、同じく扱う",
	"返事を待つあいだに同じ `ALERT` が来たときも同じく扱い",
	"手順 12 の gate のときと同じく扱う",
	"始まる TASK)を頼んでいる間も同じで",
	"reviewer が差分を読んでいれば待",
}

// skillXreviewGone are phrases of the /org and the /director skills from
// before the reviewer seat ran the cross-review on its own model (plan
// 2026-11-22-org-xreview-own-model, issue #360): the TASK lines the leader
// wrote for the other model's call, the reviewer's pick of the other model
// with pick_reviewer, that call and its 20 minutes, the other model the
// branch was shown to, and the network a codex reviewer seat needed for it.
// No copy of either skill says any of them; the plan advisory's other model
// is worded otherwise and stays.
var skillXreviewGone = []string{
	"IMPLEMENTER_DRIVER", "REVIEWER_CODEX", "REVIEWER_CLAUDE",
	"`pick_reviewer` に引数で渡して", "`pick_reviewer` で相手のモデルを選ぶ", "implementer と違う系列のモデル(claude なら codex",
	"違う系列のモデルに差分を読むだけ", "相手のモデルの呼び出し", "呼び出しは 20 分まで", "相手が所見を返して", "相手のモデルが所見を返して",
	"相手に見せていない", "相手のモデルに見せていない", "相手のモデルにもう一度見せる", "network がないと相手の CLI",
}

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

// readSkillCopies returns the text of each copy of a skill (copies, relative
// to the repository root), keyed by its path, and skips t when a copy is
// missing (a repo that vendors internal/org without the skills tree).
func readSkillCopies(t *testing.T, copies []string, what string) map[string]string {
	t.Helper()
	root := directorSkillRepoRoot(t)
	texts := make(map[string]string, len(copies))
	for _, rel := range copies {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Skipf("%s not found (%v) -- skipping the skill's %s check (vendored repo?)", rel, err, what)
		}
		texts[rel] = string(data)
	}
	return texts
}

// TestDirectorSkill_XreviewStallIsWaitedOutOnlyWhileTheScreenMoves pins D9
// and AC6 of plan 2026-11-01-org-cross-review for the ALERT bullet of
// ### 件の処理, as plan 2026-11-22-org-xreview-own-model rewrote it and its
// self-review M1 bounded it: after the reviewer's stall item while the gate
// runs, which is waited out with a still screen, the bullet says only the
// gate is (xreviewStallGateOnly); while the leader waits on a cross-review
// TASK (TASK_ID xreview-) the director waits only while the reviewer's
// screen moves, since the seat may take a while to read a large diff, in
// which it sends nothing, and after 10 minutes of a still screen it decides
// as above without waiting more, the leader raising a BLOCKED then too. Step
// 12 of the leader template says the same (xreviewStallGateOnly,
// xreviewStallWait, xreviewStallStill). The bullet no longer says the other
// model's call takes up to 20 minutes, nor waits on a reader with no word of
// its screen (xreviewStallExempt).
func TestDirectorSkill_XreviewStallIsWaitedOutOnlyWhileTheScreenMoves(t *testing.T) {
	checkDirectorSkillPlaces(t, "cross-review stall", []skillPlace{
		{"the ALERT bullet of ### 件の処理", func(text string) string { return directorSkillStep3Bullet(text, "- ALERT(見張りの escalation)") }, []string{
			"leader が reviewer に gate を頼んでいる間は、gate が長いと reviewer の停滞の件(`SEAT: reviewer`、`CONDITION: stall`)が出ることがある",
			"gate が走っていれば待ち、終わってから上のとおり決める",
			xreviewStallGateOnly + "leader が reviewer に cross-review の TASK(TASK_ID が `xreview-` で始まる TASK)を頼んでいる間は、" +
				"reviewer の画面が動いていれば待ち(差分が大きいと読むのに時間がかかり、そのあいだ reviewer は何も送らない)、終わってから上のとおり決める",
			"10 分たっても画面が動かなければ、それ以上は待たずに上のとおり決める(leader も、そのときは座席の名前を書いた `BLOCKED` を上げる)",
			"座席や org を止めるか(`ralph org stop`、`ralph org disband`)は人が決める",
		}},
	})
	for rel, text := range readSkillCopies(t, directorSkillCopies, "cross-review stall") {
		bullet := directorSkillStep3Bullet(text, "- ALERT(見張りの escalation)")
		for _, gone := range append([]string{"相手のモデルの呼び出し", "20 分まで", "呼び出しが走っていれば"}, xreviewStallExempt...) {
			if containsPhrase(bullet, gone) {
				t.Errorf("%s: expected the ALERT bullet of ### 件の処理 to no longer say %q, from when a cross-review's reviewer was waited out as the gate's, got bullet:\n%s", rel, gone, bullet)
			}
		}
	}
	step12 := leaderFeatureOrgItem(t, step12Marker)
	for _, want := range []string{xreviewStallGateOnly, xreviewStallWait, xreviewStallStill} {
		if !containsPhrase(step12, want) {
			t.Errorf("expected step 12 of the leader template to say %q, as the /director skill's ALERT bullet does, got item:\n%s", want, step12)
		}
	}
}

// TestDirectorSkill_XreviewCapQuestionGoesToTheHumanAsAFork pins D7b for the
// /director skill: step 3 of ### 件の処理 sends the QUESTION with the header
// XREVIEW: cap_reached to the subsection directorXreviewCapHeading, which
// comes after ### 計画の承認. The subsection says what the item means (the
// complete runs, in which the reviewer seat read the diff and triaged its
// findings, reached the cap, with ACTION_REQUIRED left or with changes the
// reviewer seat has not seen), read from the EVIDENCE: report in the
// feature worktree, as ### 計画の承認 reads the plan there, since the report
// is only on the feature branch (self-review cycle 1 L2). It asks the human
// the FORK 1: line's a, b, and c with a, raising the cap, recommended and
// first, with AskUserQuestion, or numbered options on Codex and for a
// director whose wake includes push (### 人への聞き方, as #271 asks every
// question), sends the answer as a DECISION whose body has the line
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
			"機能の org の leader は、`[org.review].cross_review` が on のとき、reviewer に cross-review" +
				"(reviewer 座席が自分のモデルで差分を読み直し、所見を triage する)を頼む",
			"(`/org` skill の「org の終わり方」の 1)",
			"ヘッダに `XREVIEW: cap_reached` がある件がそれで、本文に `SUMMARY:` の行、`FORK 1:` の 1 行、最後の complete の report のパスの `EVIDENCE:` の行がある",
		}},
		{"step 1 of ### 計画の承認", func(text string) string {
			return directorSkillNumberedStep(directorSkillSubsection(text, "### 計画の承認"), 1)
		}, []string{directorFeatureWorktree + "で"}},
		{"step 1 of " + directorXreviewCapHeading, step(1), []string{
			"件の意味を人に伝える",
			"reviewer 座席が差分を読み終えて triage まで終わった回(complete の回)が上限(`[org.review].max_cross_review_cycles` と、人が前に上げた値の大きいほう)に届き",
			"`SUMMARY:` の行と `EVIDENCE:` の report を読み、どちらなのかと要点を伝える",
			"report は機能のブランチにしかないので、" + directorFeatureWorktree + "で読む",
			"最後の complete の回の ACTION_REQUIRED が、直されないまま残っている",
			"最後の complete の回のあとに、reviewer 座席に見せていない変更がある",
		}},
		{"step 2 of " + directorXreviewCapHeading, step(2), []string{
			"`FORK 1: <問い> / a 上限を上げる(推奨) / b <進む案> / c 止める`",
			"AskUserQuestion(Codex と、wake に push がある director は番号つきの選択肢。「人への聞き方」)で、選択肢を行の順(推奨の a が最初)に並べる",
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
	for rel, text := range readSkillCopies(t, directorSkillCopies, "cap DECISION") {
		step3 := directorXreviewCapStep(text, 3)
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
		"reviewer が差分を読み終えて triage まで終わった回だけを数える",
	} {
		if !containsPhrase(sub, want) {
			t.Errorf("expected %s to say %q, as the /director skill's %s reads it, got:\n%s", leaderXreviewWhere, want, directorXreviewCapHeading, sub)
		}
	}
	if reviewer := reviewerXreviewSection(t); !containsPhrase(reviewer, "- Cycle: <CYCLE の値>") || !containsPhrase(reviewer, "`CYCLE: <n>/<上限>`") {
		t.Errorf("expected the reviewer template's report to have the line - Cycle: <n>/<上限>, from which the /director skill's %s reads the cap, got:\n%s", directorXreviewCapHeading, reviewer)
	}
}

// TestDirectorSkill_ResultItemReportsTheCrossReviewLine pins D10 of plan
// 2026-11-01-org-cross-review for the RESULT bullet of ### 件の処理, as plan
// 2026-11-22-org-xreview-own-model rewrote it: after the follow-up lines,
// the RESULT's body carries the PR body's Cross-review: value as a
// CROSS_REVIEW: line (ran with the report's Reviewer:, the reviewer seat's
// seat_id, driver, and model; skipped, off included; or incomplete with the
// reason), which the director reports with the PR, since skipped and
// incomplete mean no findings came back: incomplete when the reviewer seat
// did not finish reading the diff, and a skipped other than off from a
// reviewer seat of an earlier version that called another model, each said
// in a sentence of its own (self-review L4 of plan
// 2026-11-22-org-xreview-own-model). The bullet no longer says a codex
// reviewer seat without network may answer so every time. Step 14 of the
// leader template writes the line, and its ### cross-review subsection names
// the earlier reviewer's skipped. All of them name the value by where it
// comes from, <report の Reviewer:> (xreviewRanValue; self-review L1), and
// none keeps the name <相手>, which meant the other model when the reviewer
// called one.
func TestDirectorSkill_ResultItemReportsTheCrossReviewLine(t *testing.T) {
	checkDirectorSkillPlaces(t, "RESULT cross-review", []skillPlace{
		{"the RESULT bullet of ### 件の処理", directorSkillResultBullet, []string{
			"その行も PR の URL と一緒に人に報告する",
			"本文には、PR の本文の `Cross-review:` の行と同じ値の `CROSS_REVIEW: <値>` の 1 行も載る",
			"値は " + xreviewRanValue + "、`skipped (<理由>)`(`[org.review].cross_review` が off のときも)、`incomplete (<理由>)` のどれかで、" +
				"PR を報告するときに理由と一緒に人に伝える",
			"cross-review の report の `Reviewer:` の値には、reviewer 座席の seat_id と driver と model が入る",
			"skipped と incomplete は、cross-review の所見が出ていないことを指す",
			"incomplete は、reviewer 座席が差分を読み終えなかった回(`git diff` が失敗した、など)に出る。" +
				"off でないときの skipped は、ほかのモデルを呼んでいた前の版の reviewer 座席が返すことがある",
			"leader はそのあと disband するので答えは送らず",
		}},
	})
	step14 := leaderFeatureOrgItem(t, leaderStep14Marker)
	for _, name := range []string{"`CROSS_REVIEW: <値>`", "`- Cross-review:`", xreviewRanValue, "`skipped (<理由>)`", "`incomplete (<理由>)`"} {
		if !containsPhrase(step14, name) {
			t.Errorf("expected step 14 of the leader template to name %s, as the /director skill's RESULT bullet does, got item:\n%s", name, step14)
		}
	}
	sub := leaderXreviewSubsection(t)
	for _, want := range []string{"`Cross-review: ran (<report の Reviewer:>)` を控え、手順 9 の sync-docs に進む", "ほかのモデルを呼んでいた前の版の reviewer が返すことがある"} {
		if !containsPhrase(sub, want) {
			t.Errorf("expected %s to say %q, as the /director skill's RESULT bullet does, got:\n%s", leaderXreviewWhere, want, sub)
		}
	}
	for where, text := range map[string]string{"step 14 of the leader template": step14, leaderXreviewWhere: sub} {
		if containsPhrase(text, "<相手>") || containsPhrase(text, "<その report の Reviewer:>") {
			t.Errorf("expected %s to name the Cross-review: value %s only, not <相手> or <その report の Reviewer:>, got:\n%s", where, xreviewRanValue, text)
		}
	}
	for _, copies := range [][]string{directorSkillCopies, orgSkillCopies} {
		for rel, text := range readSkillCopies(t, copies, "RESULT cross-review") {
			if containsPhrase(text, "(<相手>)") || containsPhrase(text, "`<相手>` は") {
				t.Errorf("%s: expected the skill to name the Cross-review: value %s, not <相手>", rel, xreviewRanValue)
			}
		}
	}
	if reviewer := reviewerXreviewSection(t); !containsPhrase(reviewer, "- Reviewer: reviewer 座席 ") {
		t.Errorf("expected the reviewer template's report to have the line - Reviewer: reviewer 座席 <seat_id>(<driver>、<model>), which the /director skill's RESULT bullet names, got:\n%s", reviewer)
	}
}

// TestOrgSkill_PrerequisitesSayTheLeaderReadsTheReviewSettings pins AC8 of
// plan 2026-11-01-org-cross-review for the /org skill's ## 前提, as plan
// 2026-11-22-org-xreview-own-model rewrote it: the pane's RALPH_PRIMARY_CLI
// feeds pick_reviewer for the leader's plan advisory only, and the reviewer
// seat's cross-review does not use it, since it calls no other CLI and
// reads the diff on the seat's own model; the feature org's leader reads
// [org.review] with ralph config get in the worktree, plan_advisory before
// the plan advisory and cross_review and max_cross_review_cycles at step 9
// after the rebase, and the reviewer seat reads no setting. The text for a
// later version, for a seat that reads the setting itself, and for the
// reviewer's pick of the other model from the TASK's implementer driver is
// gone. The templates read the same commands, and the reviewer template's
// cross-review names no pick_reviewer.
func TestOrgSkill_PrerequisitesSayTheLeaderReadsTheReviewSettings(t *testing.T) {
	checkOrgSkillPlaces(t, "review settings", []skillPlace{
		{"the section ## 前提", func(text string) string { return directorSkillSubsection(text, "## 前提") }, []string{
			"3 つ目の値は座席の driver(`claude` か `codex`)で、機能の org の leader の計画助言がもう一方の CLI を選ぶときに使う " +
				"`scripts/xreview-helpers.sh` の `pick_reviewer` に、leader の driver を読ませる",
			"reviewer 座席の cross-review はこの値を使わない(ほかの CLI を呼ばず、座席が自分のモデルで差分を読む。「org の終わり方」の 1)",
			"leader の計画助言は、変数がなければ leader 自身の CLI で相手を選ぶので、相手を取り違えるのは",
			"`[org.review]` は機能ごとの org の leader の計画助言と cross-review の有無(既定はどちらも off)と、" +
				"cross-review を回す回数の上限(最初の 1 回を含む。既定 2)で、leader が worktree で `ralph config get <key>` を打って読む",
			"計画助言は頼む前に `plan_advisory` を、cross-review は役割の雛形の手順 9 で rebase のあとに `cross_review` と `max_cross_review_cycles` を読む" +
				"(`ralph config get org.review.cross_review` など)",
			"reviewer 座席は設定を読まず、cross-review の TASK を受けたら、自分のモデルで差分を読み直す(「org の終わり方」の 1)",
		}},
	})

	for rel, text := range readSkillCopies(t, orgSkillCopies, "review settings") {
		section := directorSkillSubsection(text, "## 前提")
		for _, gone := range []string{
			"読み始めるのは後の版から",
			"`pick_reviewer` に、座席の driver を読ませる",
			"codex の座席の cross-review がもう一方の CLI を取り違えることがある",
			"`IMPLEMENTER_DRIVER:` の行に書いた implementer の driver",
			"`pick_reviewer` で相手のモデルを選ぶ",
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
	if reviewer := reviewerXreviewSection(t); containsPhrase(reviewer, "pick_reviewer") || containsPhrase(reviewer, "RALPH_PRIMARY_CLI") {
		t.Errorf("expected the reviewer template's cross-review to use neither pick_reviewer nor RALPH_PRIMARY_CLI, as the /org skill's ## 前提 says, got:\n%s", reviewer)
	}
}

// TestOrgSkill_EndOfOrgStep4WaitsOutAnXreviewStallOnlyWhileTheScreenMoves
// pins D9 and AC6 of plan 2026-11-01-org-cross-review for item 4 of
// ### org の終わり方, as plan 2026-11-22-org-xreview-own-model rewrote it and
// its self-review M1 bounded it: after the stall ALERT about the reviewer
// while the gate runs, which is waited out with a still screen, the item
// says only the gate is (xreviewStallGateOnly); a stall ALERT while item 1's
// cross-review TASK runs has the leader wait only while the reviewer seat's
// screen moves (xreviewStallWait), and after 10 minutes of a still screen
// with no RESULT or BLOCKED escalate a BLOCKED naming the seat
// (xreviewStallStillSkill), not wait as for the gate or for another model's
// call of up to 20 minutes (xreviewStallExempt).
func TestOrgSkill_EndOfOrgStep4WaitsOutAnXreviewStallOnlyWhileTheScreenMoves(t *testing.T) {
	checkOrgSkillPlaces(t, "cross-review stall", []skillPlace{
		{"item 4 of ### org の終わり方", func(text string) string { return orgSkillEndOfOrgStep(text, 4) }, []string{
			"この間に見張りから reviewer の停滞の `ALERT`(`SEAT: reviewer`、`CONDITION: stall`)が来ても、reviewer を stop も spawn し直しもせず",
			"gate が走っていれば待つ(gate は走っているあいだ何も記録せず、Codex の reviewer は前面で打つので、画面が長く変わらないことがある)",
			xreviewStallGateOnly + "1 の cross-review の TASK の返事を待つあいだに同じ `ALERT` が来たときは、" + xreviewStallWait,
			xreviewStallStillSkill,
			"gate の TASK への `GATE: pass` の `RESULT` は 1 ではなくここで扱う",
		}},
	})
	for rel, text := range readSkillCopies(t, orgSkillCopies, "cross-review stall") {
		item := orgSkillEndOfOrgStep(text, 4)
		for _, gone := range xreviewStallExempt {
			if containsPhrase(item, gone) {
				t.Errorf("%s: expected item 4 of ### org の終わり方 to no longer say %q, from when a cross-review's reviewer was waited out as the gate's, got item:\n%s", rel, gone, item)
			}
		}
	}
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

	for rel, text := range readSkillCopies(t, orgSkillCopies, "old sync-docs head") {
		if old := "rebase が済んだら(要らなかったときも)、leader は sync-docs を"; containsPhrase(orgSkillEndOfOrgStep(text, 1), old) {
			t.Errorf("%s: expected item 1 of ### org の終わり方 to no longer start the sync-docs paragraph with %q, which reads as right after the rebase", rel, old)
		}
	}
	if !containsPhrase(leaderFeatureOrgItem(t, step9Marker), syncDocsHandOff) {
		t.Errorf("expected step 9 of the leader template to say %q, as item 1 of the /org skill's ### org の終わり方 does", syncDocsHandOff)
	}
}

// TestOrgSkill_EndOfOrgStep1HandsTheXreviewBeforeTheSyncDocs pins AC2, AC5,
// AC5b, D3, D7, D7b, and D8 of plan 2026-11-01-org-cross-review for the
// /org skill, as plan 2026-11-22-org-xreview-own-model (AC7) rewrote them: a
// paragraph of item 1 of ### org の終わり方, after the rebase and before the
// sync-docs paragraphs (which start once the cross-review is over,
// TestOrgSkill_EndOfOrgSyncDocsStartsOnceTheXreviewIsOver), reads the two
// settings, keeps Cross-review: skipped for item 6 when off, sends an
// xreview- TASK with the three body lines XREVIEW_BASE:, CYCLE:, and
// REPORT: when on, with no implementer driver; the reviewer seat reads the
// branch's diff again on its own model, the implementer's family or not,
// and calls no other model with codex exec or claude -p; a human who wants
// another family sets [org.role_models].reviewer to the other driver; the
// reviewer compares the git state before it reads and before it writes the
// report and answers XREVIEW: git_changed when they differ, so the leader
// leaves the worktree alone; it commits each report, counts only the
// complete runs against the cap, the larger of max_cross_review_cycles and
// the cap file, escalates the cap QUESTION (XREVIEW: cap_reached, FORK 1:
// with a recommended) instead of a TASK, and goes on to the sync-docs with
// the report committed and git status empty. Item 6 puts the value in the
// PR body's Cross-review: line, ran with the report's Reviewer:, and the
// RESULT's CROSS_REVIEW: line, and the summaries in ## Leader 運用 2 経路
// and ## permission 作法 put the cross-review between the rebase and the
// sync-docs. The names are the leader template's (rendered with org-a for
// <org_id>) and the reviewer template's.
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
			"本文の行は、base(`XREVIEW_BASE:`)、何回目かと上限(`CYCLE:`)、report のパス" +
				"(`REPORT:`、`docs/reports/review-xreview-<YYYY-MM-DD>-<org_id>-c<k>.md`)の 3 つで、implementer の driver は読まず、書かない",
			"reviewer 座席は、implementer と同じ系列でも違う系列でも、自分のモデルでこのブランチの差分を読み直して所見を出し、triage した report を書いて返す",
			"`codex exec` や `claude -p` でほかのモデルを呼ばない(reviewer の雛形の「cross-review の TASK」)",
			"違う系列のモデルにレビューさせたいときは、`[org.role_models].reviewer` を implementer と違う driver にする(「役割ごとのモデル」節)",
			"reviewer は読む前と report を書く前の git の状態を比べ、違えば report を書かずに `XREVIEW: git_changed` の BLOCKED を返すので、" +
				"返事が来るまで leader は worktree を変えず、implementer に TASK を送らない",
			"返事を受けるたびに report をコミットする",
			"ACTION_REQUIRED のうち直すものは implementer に TASK で頼み(per-plan では `PLAN:` のヘッダを付ける)、レビューから回し直す",
			"上限に数えるのは、reviewer が差分を読み終えて triage まで終わった回(report に `- Reviewer status: complete` の行がある回)だけで、" +
				"skipped と incomplete の回は数えず、頼み直さずに進む",
			"上限は `max_cross_review_cycles`(既定 2)と、人が上限を上げたときに leader が書く `.harness/state/xreview-<org_id>-cap` の値の大きいほう",
			"上限に届いていて、ACTION_REQUIRED が残るか、最後の complete の回のあとに reviewer に見せていない変更があれば、TASK を送らず、" +
				"ヘッダに `XREVIEW: cap_reached` の行、本文に `FORK 1:` の 1 行(a 上限を上げる、b PR に書いて進む、c 止める の 3 つで、推奨は a)を書いた QUESTION を escalate する",
			"人の答えは DECISION の `FORK 1: <記号> <案>` の行で届く(`/director` skill の「cross-review の上限」)",
			"c なら leader はそれ以上何もせず、sync-docs にも push にも進まず、disband もしない",
			"cross-review が終わったら、この回の report がコミットしてあり `git status --porcelain` が何も出さないことを確かめてから、次の段落の sync-docs に進む",
			orgSkillSyncDocsHandOff,
		}},
		{"item 6 of ### org の終わり方", func(text string) string { return orgSkillEndOfOrgStep(text, 6) }, []string{
			"最後の回で sync-docs を飛ばしたときは、`" + docFollowupKey + ":` の行の代わりに、控えた " + syncDocsSkipLine + " の行を書く",
			"PR の本文には本文の雛形の `- Cross-review:` の行も置き、1 で最後に控えた値(" + xreviewRanValue + "、`skipped (<理由>)`、`incomplete (<理由>)` のどれか。" +
				"cross-review の report の `Reviewer:` の値には、reviewer 座席の seat_id と driver と model が入る)と、" +
				"直さなかった所見と、上限の QUESTION で人が進むと答えたときの known gaps か reviewer に見せていない変更を書く",
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
		"`XREVIEW_BASE: origin/<既定のブランチ>`",
		"`CYCLE: <n>/<上限>`",
		"`REPORT: docs/reports/review-xreview-<YYYY-MM-DD>-<org_id>-c<k>.md`",
		leaderXreviewNoDriver,
		"`docs/reports/review-xreview-<YYYY-MM-DD>-<org_id>-c<k>.md`",
		"`- Reviewer status: complete`",
		"`.harness/state/xreview-<org_id>-cap`",
		"`XREVIEW: cap_reached`",
		"`XREVIEW: git_changed`",
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
	reviewer := reviewerXreviewSection(t)
	for _, want := range []string{
		"レビューするのはこの座席のモデル(冒頭の `driver:` と `model:` の行)で、implementer と同じ系列でも違う系列でも変わりません",
		reviewerXreviewNoOtherModel,
		"`XREVIEW: git_changed`",
	} {
		if !containsPhrase(reviewer, want) {
			t.Errorf("expected the reviewer template's cross-review to say %q, as the /org skill says, got:\n%s", want, reviewer)
		}
	}
	if got := reviewerXreviewTaskKeys(t); strings.Join(got, ",") != "XREVIEW_BASE,CYCLE,REPORT" {
		t.Errorf("expected the reviewer template's cross-review to read the three body lines XREVIEW_BASE:, CYCLE:, and REPORT:, as the /org skill names them, got %q", got)
	}
}

// TestOrgSkill_RoleModelsSayTheReviewerSeatReviewsOnItsOwnModel pins the
// user's decision of 2026-11-21 behind plan
// 2026-11-22-org-xreview-own-model (issue #360; the plan's AC7 and its
// Non-goals): in the org runtime the reviewer seat reviews, the review and
// the cross-review alike, on the driver and model it was given, calling no
// other model, whether or not that is the implementer's family. The /org
// skill's ### 役割ごとのモデル says so, that the two built-in sets put the
// implementer and the reviewer in the same family, that a human who wants
// another family sets [org.role_models].reviewer to the other driver (or the
// reviewer's driver and model in the split plan's preamble), and that the
// change reaches the reviewer seats started after it. Its ### 座席内 fan-out
// keeps the review's judgment on children of the seat's own model and
// leaves only the gate's commands to a cheaper one, as the reviewer
// template's ## 座席内 fan-out says.
func TestOrgSkill_RoleModelsSayTheReviewerSeatReviewsOnItsOwnModel(t *testing.T) {
	checkOrgSkillPlaces(t, "reviewer seat's own model", []skillPlace{
		{"the subsection ### 役割ごとのモデル", func(text string) string {
			return directorSkillSubsection(text, "### 役割ごとのモデル(`[org.role_models]`)")
		}, []string{
			"| reviewer | claude / `opus` | codex / `gpt-6.1-sol` |",
			"reviewer 座席は、レビューと cross-review を自分の driver とモデルで行い、ほかのモデルを呼ばない",
			"implementer と同じ系列でも違う系列でも同じで、既定の 2 組では implementer と reviewer が同じ系列になる",
			"違う系列のモデルにレビューさせたいときは、`[org.role_models].reviewer` を implementer と違う driver にする" +
				"(例: implementer が claude の組なら `reviewer = { driver = \"codex\", model = \"gpt-6.1-sol\" }`)",
			"分割計画の前文で reviewer の座席の driver と model を変えてもよい(「分割計画」節)",
			"どちらも、変えたあとに立てる reviewer 座席から効く",
		}},
		{"the subsection ### 座席内 fan-out", func(text string) string { return directorSkillSubsection(text, "### 座席内 fan-out") }, []string{
			"正確性/セキュリティ/仕様適合の観点を並列レビューする",
			"レビューの判断(観点ごとやファイルごとのレビュー、cross-review の所見と triage)を任せる子は reviewer 座席と同じモデルで動かし、" +
				"安いモデルの子に任せるのはゲートの実行だけにする(reviewer の雛形の「座席内 fan-out」)",
		}},
	})
	for _, pool := range [][]string{{"claude", "codex"}, {"codex"}} {
		roles := config.DefaultRoleModels(pool)
		if roles["reviewer"].Driver != roles["implementer"].Driver {
			t.Errorf("DefaultRoleModels(%q) gives the reviewer %q and the implementer %q, but the /org skill's ### 役割ごとのモデル says the built-in sets put them in the same family", pool, roles["reviewer"].Driver, roles["implementer"].Driver)
		}
	}
	if codex := config.DefaultRoleModels([]string{"codex"})["reviewer"]; codex.Driver != "codex" || codex.Model != "gpt-6.1-sol" {
		t.Errorf("the codex set's reviewer is %+v, but the /org skill's ### 役割ごとのモデル shows reviewer = { driver = \"codex\", model = \"gpt-6.1-sol\" } as the other family of a claude implementer", codex)
	}
}

// TestDirectorSkill_XreviewSaysTheReviewerSeatReviewsOnItsOwnModel pins the
// same decision for the /director skill (plan
// 2026-11-22-org-xreview-own-model, AC7; the issue's acceptance condition
// on a reviewer of the implementer's family): ### cross-review の上限 says
// the reviewer seat reads the diff on its own model, the implementer's
// family or not, and calls no other model, and what to tell a human who
// wants another family's review (set [org.role_models].reviewer to the
// other driver, or the reviewer's driver in the split plan's preamble,
// which reaches the reviewer seats started after it); and ### 分割計画の前
// の質問 tells the human so when it asks about the seats' driver and model.
func TestDirectorSkill_XreviewSaysTheReviewerSeatReviewsOnItsOwnModel(t *testing.T) {
	checkDirectorSkillPlaces(t, "reviewer seat's own model", []skillPlace{
		{"the subsection " + directorSplitQuestionsHeading, directorSplitSubsection(directorSplitQuestionsHeading), []string{
			"座席の driver と model は、`ralph.toml` の `[org.role_models]` と違うものにする座席があるかだけを聞く",
			"この問いでは、reviewer 座席が、implementer と同じ系列でも違う系列でも、自分のモデルでレビューと cross-review をする(ほかのモデルを呼ばない)ことを伝える",
			"違う系列のモデルにレビューさせたいなら、前文で reviewer の driver を implementer と違うものにする(`/org` skill の「役割ごとのモデル」)",
			"機能の内側の分かれ目は聞かない",
		}},
		{"the subsection " + directorXreviewCapHeading, func(text string) string { return directorSkillSubsection(text, directorXreviewCapHeading) }, []string{
			"(reviewer 座席が自分のモデルで差分を読み直し、所見を triage する)を頼む(`/org` skill の「org の終わり方」の 1)",
			"reviewer 座席は、implementer と同じ系列でも違う系列でも、自分のモデルでレビューし、ほかのモデルを呼ばない",
			"人が違う系列のモデルのレビューを求めたら、`[org.role_models].reviewer` を implementer と違う driver にするか、" +
				"分割計画の前文で reviewer の driver を変える(「分割計画の前の質問」)ことを伝える",
			"どちらも、変えたあとに立てる reviewer 座席から効く",
			"回数には上限があり、届くと leader は QUESTION を上げる",
		}},
	})
}

// TestOrgAndDirectorSkills_SayNothingOfTheOtherModelsCrossReview pins AC7 of
// plan 2026-11-22-org-xreview-own-model: no copy of the /org or the
// /director skill keeps a phrase of skillXreviewGone, from when the
// reviewer seat showed the branch to the other model.
func TestOrgAndDirectorSkills_SayNothingOfTheOtherModelsCrossReview(t *testing.T) {
	for _, copies := range [][]string{orgSkillCopies, directorSkillCopies} {
		for rel, text := range readSkillCopies(t, copies, "other model's cross-review") {
			for _, gone := range skillXreviewGone {
				if containsPhrase(text, gone) {
					t.Errorf("%s: expected no %q, from when the reviewer seat showed the branch to the other model (plan 2026-11-22-org-xreview-own-model)", rel, gone)
				}
			}
		}
	}
}
