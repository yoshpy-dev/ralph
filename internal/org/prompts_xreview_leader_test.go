package org

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// The heading of the leader template's cross-review subsection (plan
// 2026-11-01-org-cross-review, issue #269, slice S2), the lines its code
// blocks follow, and the paragraph of step 9 that hands the cross-review to
// it.
const (
	leaderXreviewHeading      = "### cross-review"
	leaderXreviewWhere        = "the cross-review subsection of the 機能ごとの org section"
	leaderXreviewStateLead    = "まず worktree で次のコマンドを、まとめて 1 回の shell で打つ"
	leaderXreviewQuestionLead = "上限の QUESTION は、「件を上げる」の節の手順で `ralph org escalate` で上げる"
	leaderXreviewLinesLead    = "本文には、reviewer が読む次の行を書く"
	leaderXreviewTaskLead     = "per-plan の org でも、reviewer への TASK に `PLAN:` のヘッダは要らない"
	leaderXreviewStep9        = "rebase が済んだら(要らなかったときも)、sync-docs の前に、この節の終わりの「cross-review」の小節のとおりに cross-review を扱う"
)

// The sentences of the cap QUESTION's answer a that write the raised cap and
// send the next TASK with it, and the TASK line that carries the cap
// (self-review L1 of plan 2026-11-01-org-cross-review).
const (
	leaderXreviewRaiseA = "- a なら、新しい上限を `.harness/state/xreview-org-a-cap` に数字 1 つだけの 1 行で書く" +
		"(DECISION は compaction のあと読み直せないので、上限はこのファイルに控える)"
	leaderXreviewRaiseRule = "新しい上限は、DECISION が数を書いていればその数(その数が `done:` 以下なら `done:` + 1。`done:` 以下の数では上限に届いたままで、同じ QUESTION に戻る)、" +
		"書いていなければ `done:` + 1 にする"
	leaderXreviewRaiseRerun = "2 つ目の QUESTION なら、この小節の初めのコマンドを打ち直し、その `cap:`(書いた上限が入る)で、下のとおりすぐ TASK を送る" +
		"(`CYCLE:` の `<上限>` を新しい上限にするため)"
	leaderXreviewCycleLine = "`CYCLE: <n>/<上限>`: `<n>` は `done:` + 1、`<上限>` は `cap:` の値(上限を上げたときは、cap のファイルを書いたあとに打ち直したコマンドの値)"
)

// leaderXreviewSubsection returns the cross-review subsection of the leader
// template's 機能ごとの org section: the text after its heading line up to
// the next "### " line or the end of the section.
func leaderXreviewSubsection(t *testing.T) string {
	t.Helper()
	section := leaderFeatureOrgSection(t)
	_, body, found := strings.Cut(section, "\n"+leaderXreviewHeading+"\n")
	if !found {
		t.Fatalf("expected the 機能ごとの org section to have the subsection %q, got section:\n%s", leaderXreviewHeading, section)
	}
	body, _, _ = strings.Cut(body, "\n### ")
	return body
}

// leaderXreviewStateBlock returns the subsection's command that reads the
// two settings, the cap file, and the branch's cross-review reports.
func leaderXreviewStateBlock(t *testing.T) string {
	t.Helper()
	return fencedBlockAfter(t, leaderXreviewSubsection(t), leaderXreviewStateLead, leaderXreviewWhere)
}

// leaderXreviewReportPattern returns the extended regular expression the state
// command's grep keeps the reports of this org with, compiled.
func leaderXreviewReportPattern(t *testing.T) *regexp.Regexp {
	t.Helper()
	block := leaderXreviewStateBlock(t)
	m := regexp.MustCompile(`grep -E '([^']+)'`).FindStringSubmatch(block)
	if m == nil {
		t.Fatalf("expected the state command of %s to keep the reports with grep -E, got:\n%s", leaderXreviewWhere, block)
	}
	return regexp.MustCompile(m[1])
}

// TestRenderRolePrompt_Leader_XreviewSitsBetweenTheRebaseAndTheSyncDocs pins
// AC2 and D2 and D8 of plan 2026-11-01-org-cross-review: one paragraph of
// step 9, after the rebase and before the sync-docs paragraphs, hands the
// cross-review to the ### cross-review subsection, which comes after ###
// 計画助言 at the end of the 機能ごとの org section; the reports the reviewer
// writes are committed as each reply comes, and the leader goes on to
// sync-docs only with a clean status. The order is commit, fetch, rebase,
// cross-review, the report's commit, sync-docs, whose paragraph starts once
// the cross-review is over, not after the rebase (self-review L3).
func TestRenderRolePrompt_Leader_XreviewSitsBetweenTheRebaseAndTheSyncDocs(t *testing.T) {
	assertPhrasesInOrder(t, leaderFeatureOrgItem(t, step9Marker), leaderItemWhere(step9Marker), []string{
		"まず reviewer のレポート(reviewer が RESULT や BLOCKED に書いた、`docs/reports/` の下のファイル)をコミットし",
		"`git fetch origin <既定のブランチ>` を打つ",
		"`git rebase origin/<既定のブランチ>` を打つ",
		leaderXreviewStep9,
		"reviewer が書いた cross-review の report は、返事を受けるたびにその小節のとおりコミットする",
		"cross-review が終わったら(`XREVIEW: pass` が返った、`XREVIEW: action_required` で直すものがなかった、reviewer が飛ばしたか incomplete で返した、off で TASK を送らなかった、" +
			"上限の QUESTION に人が進むと答えた、上限に届いていて相手に見せた HEAD から変わっていない、のどれか)",
		"この回の report がコミットしてあり、`git status --porcelain` が何も出さないことを確かめてから、次の段落の sync-docs に進む",
		"何か出たら、fetch の前に変更が残っていたときと同じく扱う",
		syncDocsHandOff,
	})
	if old := "rebase が済んだら(要らなかったときも)、sync-docs を"; containsPhrase(leaderFeatureOrgItem(t, step9Marker), old) {
		t.Errorf("expected step 9's sync-docs paragraph to start once the cross-review is over, not with %q, which skips it (self-review L3)", old)
	}

	section := leaderFeatureOrgSection(t)
	advisory, xreview := strings.Index(section, "\n"+leaderPlanAdvisoryHeading+"\n"), strings.Index(section, "\n"+leaderXreviewHeading+"\n")
	if advisory < 0 || xreview < advisory {
		t.Errorf("expected the subsection %q after %q in the 機能ごとの org section (at %d and %d), got section:\n%s", leaderXreviewHeading, leaderPlanAdvisoryHeading, xreview, advisory, section)
	}
	if strings.Contains(leaderPlanAdvisory(t), "cross_review") {
		t.Errorf("expected the subsection %q to hold none of the cross-review, got:\n%s", leaderPlanAdvisoryHeading, leaderPlanAdvisory(t))
	}
	assertPhrasesInOrder(t, leaderXreviewSubsection(t), leaderXreviewWhere, []string{
		"手順 9 で rebase が済んだら(要らなかったときも)、sync-docs の前に、`[org.review].cross_review` が on のときだけ、reviewer 座席に cross-review の TASK を送る",
		"reviewer は implementer と違う系列のモデル(implementer が claude なら codex、codex なら claude)にこのブランチの差分を読むだけの形で見せ、所見を triage した report を書いて返す(reviewer の雛形の「cross-review の TASK」の節)",
		"手順 9 を通るたびに、この小節のとおりに扱う",
		"返事を受けたら、本文に `REPORT:` の行があれば、まずその report をコミットし、`git status --porcelain` が何も出さないことを確かめる" +
			"(上限の回数を git から数えるため。このあとで implementer に TASK を送るときも、このコミットのあとに送る)",
	})
}

// configHasKey reports whether key (dotted, as ralph config get takes it)
// names a field of config.Config by its toml tags.
func configHasKey(key string) bool {
	v := reflect.TypeOf(config.Config{})
	for _, part := range strings.Split(key, ".") {
		found := false
		for i := range v.NumField() {
			if tag, _, _ := strings.Cut(v.Field(i).Tag.Get("toml"), ","); tag == part {
				v, found = v.Field(i).Type, true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// TestRenderRolePrompt_Leader_XreviewReadsTheSwitchWhereItIsUsed pins AC1 and
// D1: the leader reads [org.review].cross_review in the worktree each time it
// passes step 9, with ralph config get and the 計画助言 subsection's handling
// of its exit code. The state command prints switch: on for 0 and true; off
// with its reason for 0 and false, for an older ralph's exit 1 with
// unknown command "config", and for the shell's 127; error otherwise. Off
// sends no TASK and keeps the skipped line for step 14; error is a BLOCKED
// with the stderr file the command writes. The reasons in the text are the
// ones the command echoes, and both keys it reads are keys of ralph.toml.
func TestRenderRolePrompt_Leader_XreviewReadsTheSwitchWhereItIsUsed(t *testing.T) {
	sub := leaderXreviewSubsection(t)
	assertPhrasesInOrder(t, sub, leaderXreviewWhere, []string{
		"設定は、計画助言と同じく使うときに worktree で読み、タスクや記憶からは読まない",
		leaderXreviewStateLead,
		"`switch:` の行は `[org.review].cross_review` の値で、`ralph config get` の終了コードは「計画助言」の `ralph config get org.review.plan_advisory` と同じく扱う",
		"0 で `true` なら `switch: on` で、先に進む",
		"次のときは `switch: off (<理由>)` で、TASK を送らず、`Cross-review: skipped (<理由>)` の行を手順 14 のために控えて、手順 9 の sync-docs に進む(コミットする report はない)",
		"0 で `false` のとき(`off: [org.review].cross_review`)",
		"1 で stderr に `unknown command \"config\"` がある、この動詞のない古い ralph のとき(`ralph without config get; treated as off`)",
		"127 の、`ralph` が PATH にないとき(`ralph not found; treated as off`)",
		"それ以外(ほかの 1、2、3、0 で `true` でも `false` でもない値)は `switch: error (exit <終了コード>)` で、off とみなさない",
		"`.harness/state/xreview-org-a-switch.err` の要点を本文に書いて BLOCKED(TASK_ID は `org-a`)を「件を上げる」の節の手順で `ralph org escalate` で上げ、返事が来るまで先に進まない(待ち方は「agmsg の受信箱の運用」の節)",
	})
	block := leaderXreviewStateBlock(t)
	for _, want := range []string{
		`v=$(command ralph config get org.review.cross_review 2> "$p-switch.err"); rc=$?`,
		`c=$(command ralph config get org.review.max_cross_review_cycles 2> "$p-cap.err"); rc=$?`,
		`echo 'switch: off (off: [org.review].cross_review)'`,
		`echo 'switch: off (ralph without config get; treated as off)'`,
		`echo 'switch: off (ralph not found; treated as off)'`,
	} {
		if !strings.Contains(block, want) {
			t.Errorf("expected the state command of %s to have %s, got:\n%s", leaderXreviewWhere, want, block)
		}
	}
	for _, key := range []string{"org.review.cross_review", "org.review.max_cross_review_cycles"} {
		if !configHasKey(key) {
			t.Errorf("expected %s, which the state command reads with ralph config get, to be a key of ralph.toml", key)
		}
	}
	if got := config.Default().Org.Review.MaxCrossReviewCycles; got != 2 {
		t.Errorf("config.Default()'s [org.review].max_cross_review_cycles = %d, but the state command falls back to 2 as its default", got)
	}
}

// TestRenderRolePrompt_Leader_XreviewCountsOnlyCompleteReports pins D3 and
// D7's cap: cap: is the larger of the setting (2, the default, for an older
// ralph or none) and a positive integer on the first line of the cap file the
// leader writes when a human raises the cap; base: is step 9's
// origin/<既定のブランチ>, and an error is a BLOCKED (self-review M1); done:
// counts the reports of this org the branch added or rewrote since that base
// and committed, with the line - Reviewer status: complete, the runs the cap
// counts, never a skipped or incomplete one, nor a report an earlier org of
// the same org_id merged into the default branch, nor one left uncommitted;
// k: names this run's report one past the branch's highest number, 1 with
// none; last:, last head:, last action_required: and
// since: describe the last complete report and whether anything outside
// docs/reports/ changed since the HEAD it shows: upstream commits a rebase
// brought in count as a change, which uses the cap once, and so does a HEAD
// whose object is gone (D7b). The numbers come from git, so a compaction or a
// rebase does not change them.
func TestRenderRolePrompt_Leader_XreviewCountsOnlyCompleteReports(t *testing.T) {
	assertPhrasesInOrder(t, leaderXreviewSubsection(t), leaderXreviewWhere, []string{
		"`cap:` の行は上限の回数で、`ralph config get org.review.max_cross_review_cycles` の値(終了コードの扱いは `switch:` と同じで、古い ralph と `ralph` がないときは既定の 2)と、" +
			"`.harness/state/xreview-org-a-cap` のファイル(人が上限を上げたときにあなたが書く。下の上限の QUESTION)の 1 行目が正の整数ならその値の、大きいほう",
		"`cap: error (exit <終了コード>)` なら、`switch: error` と同じく、`.harness/state/xreview-org-a-cap.err` の要点を書いて BLOCKED を上げる",
		"`base:` の行は、このブランチの report を見分ける base で、手順 9 と同じ決め方の `origin/<既定のブランチ>`" +
			"(`refs/remotes/origin/HEAD` が指す先、なければ `main`、それもなければ `master` のうち、`origin/` の下にある最初のもの)",
		"`base: error (<理由>)` なら、base なしではこのブランチの report を数えられないので、`switch: error` と同じく、" +
			"`.harness/state/xreview-org-a-base.err` の要点(空なら行の理由)を書いて BLOCKED を上げる",
		"`done:` の行は、上限に数える回の数",
		"このブランチが base から足すか書き換えてコミットした cross-review の report(`git diff --name-only --diff-filter=AM <base>...HEAD` が出す " +
			"`docs/reports/review-xreview-<YYYY-MM-DD>-org-a-c<k>.md`)のうち、" +
			"`- Reviewer status: complete` の行を持つものの数で、相手が所見を返して triage まで終わった回だけを数える(skipped と incomplete の回は数えない)",
		"同じ org_id の前の org の report が既定のブランチに入っていても、このブランチの report ではないので数えない(同じ日付で同じパスに上書きした report は、書き換えとして数える)",
		"コミットしていない report も数えない",
		"`k:` の行は今回の report の名前の番号で、complete かどうかによらず、このブランチの report の番号のいちばん大きいもの + 1" +
			"(なければ 1。report は消さないので、このブランチの report の数 + 1 と同じ)",
		"`last:` の行は、complete の report のうち番号のいちばん大きいもの(なければ `last: none`)",
		"`last head:` はその report の `- HEAD:` の行(相手に見せた HEAD)、`last action_required:` はその `- After triage:` の行の `ACTION_REQUIRED=` の数",
		"`since:` は、その HEAD から今の HEAD までに `docs/reports/` のほかの変更がなければ `same`、あれば(その HEAD を読めないときも)`changed`。report のコミットだけなら `same` になる",
		"手順 9 の rebase で upstream のコミットが入ったときも、`docs/reports/` の外が変わるので、直しがなくても `changed` になる",
		"この変更も相手のモデルに見せていない変更として扱うので、見せれば上限を 1 回使い、上限に届いていれば下の上限の QUESTION の 2 つ目になる",
		"report の `HEAD:` の object が消えて(rebase のあとの gc など)差分を取れないときも、コマンドは `changed` を出し、変わったとみなす",
		"数は記憶でなく git から読むので、compaction や rebase のあとも同じものが出る",
	})
}

// TestRenderRolePrompt_Leader_XreviewChecksTheCapBeforeTheTask pins AC5 and
// AC5b (D7, the codex plan advisory's MEDIUM 1, and D7b): the leader sends
// the TASK only while done: is below cap:. At the cap it sends none: findings
// of the last complete run left as they were (since: same) raise the first
// QUESTION, a change the other model has not seen (since: changed, a fix
// after a gate failure or upstream commits a rebase brought in, say) the
// second, each with the options a (raise the cap, recommended), b (go on and
// write it in the PR) and c (stop), and an unchanged head after a clean run
// goes on to sync-docs.
func TestRenderRolePrompt_Leader_XreviewChecksTheCapBeforeTheTask(t *testing.T) {
	assertPhrasesInOrder(t, leaderXreviewSubsection(t), leaderXreviewWhere, []string{
		"`done:` が `cap:` より小さければ、下のとおり TASK を送る",
		"`done:` が `cap:` 以上なら(上限に届いている)、TASK を送らず、次のとおり分ける",
		"`last action_required:` が 0 より大きく、`since: same` なら(最後の complete の回の ACTION_REQUIRED が、直されないまま残っている)、上限の QUESTION の 1 つ目を上げる",
		"案は、a 上限を上げ、直してから相手のモデルにもう一度見せる(推奨)、b 残った所見を known gaps として PR の本文に書き、直さずに進む、c 止める",
		"`since: changed` なら(最後の complete の回のあとに、相手のモデルに見せていない変更がある。上限の回が通ったあとに gate が落ちて直した、rebase で upstream のコミットが入った、など)、上限の QUESTION の 2 つ目を上げる",
		"案は、a 上限を上げ、その変更を相手のモデルに見せる(推奨)、b 見せていない変更を PR の本文に書いて進む、c 止める",
		"最後の complete の回の ACTION_REQUIRED に直さなかったものがあれば、それも本文に書く",
		"`since: same` で `last action_required:` が 0 なら、相手に見せた HEAD から変わっていないので、TASK を送らずに手順 9 の sync-docs に進む",
		leaderXreviewQuestionLead,
		"TASK は reviewer 座席に送り",
	})
}

// leaderXreviewCapFork matches the FORK 1: line of the cap QUESTION: the
// question, a raises the cap and is recommended, b goes on with the gaps or
// the unseen change written in the PR, c stops.
var leaderXreviewCapFork = regexp.MustCompile(`^FORK 1: \S[^/]* / a 上限を上げる\(推奨\) / b (known gaps として|見せていない変更を) PR に書いて進む / c 止める$`)

// TestRenderRolePrompt_Leader_XreviewCapQuestionIsAFork pins D7b: the cap
// QUESTION carries the header XREVIEW: cap_reached, by which the director
// tells it apart as it tells the plan approval by PLAN: and VISUAL:, and its
// body has SUMMARY:, one FORK 1: line in the form of step 3's forks (a raises
// the cap and is recommended, b goes on, c stops) and EVIDENCE:. The answer
// is read from the DECISION's FORK 1: <記号> <案> line, not from a number at
// its head, since the director's DECISION body starts with SUMMARY:; a
// DECISION without it asks again. a writes the DECISION's number when it is
// above done:, or else done: + 1, to the cap file the state command reads,
// then either asks the implementer for the fixes and goes back through step 8
// or runs the state command again and sends the TASK at once with its fresh
// cap: (self-review L1); b keeps the gaps or the unseen change for step 14
// and goes on to sync-docs; c does nothing more (no sync-docs, no push, no
// disband) and the human decides whether the org stops. The example is a
// QUESTION escalate accepts with that header and no PLAN:, its FORK 1: line
// in the body in the form step 3's tests read, and its EVIDENCE: a report the
// state command counts; the /director skill's DECISION starts its body with
// SUMMARY: and answers a fork with the FORK <番号>: <記号> <案> line the
// leader reads.
func TestRenderRolePrompt_Leader_XreviewCapQuestionIsAFork(t *testing.T) {
	sub := leaderXreviewSubsection(t)
	assertPhrasesInOrder(t, sub, leaderXreviewWhere, []string{
		leaderXreviewQuestionLead,
		"ヘッダには `XREVIEW: cap_reached` の 1 行を書く(director は、計画の承認の QUESTION を `PLAN:` と `VISUAL:` のヘッダで見分けるように、この件をこのヘッダで見分ける)",
		"空行のあとの本文には、何が起きたかを `SUMMARY:` の行に、案を手順 3 の分かれ目と同じ形の `FORK 1:` の 1 行に、最後の complete の report のパスを `EVIDENCE:` の行に書く",
		"`FORK 1:` の行は `FORK 1: <問い> / a 上限を上げる(推奨) / b <進む案> / c 止める` の形にし、" +
			"`<進む案>` は、1 つ目の QUESTION では `known gaps として PR に書いて進む`、2 つ目では `見せていない変更を PR に書いて進む` にする(推奨の案はいつも a)",
		"上げたら、「agmsg の受信箱の運用」の節のとおり手番を終えて leader 宛ての DECISION を待つ(人の答えは director が DECISION で届ける)",
		"答えは DECISION の本文の `FORK 1: <記号> <案>` の行で読み、その記号の案に従う(director の DECISION の本文は `SUMMARY:` で始まるので、先頭の数字や文では選ばない)",
		"`FORK 1:` の行がないか、記号が a、b、c のどれでもなければ、どれも選ばず、`FORK 1: <記号> <案>` の形で答えてほしいことを本文に書いて上限の QUESTION を上げ直す",
		leaderXreviewRaiseA,
		leaderXreviewRaiseRule,
		"そのうえで、1 つ目の QUESTION なら、直すものを implementer に TASK で頼み(per-plan では `PLAN:` のヘッダを付ける)、手順 8 の往復に戻る(手順 9 を通り直し、上げた上限の中で次の回の TASK を送る)",
		leaderXreviewRaiseRerun,
		"- b なら、残った所見か見せていない変更を手順 14 の PR の本文のために控え、手順 9 の sync-docs に進む(report のコミットと `git status --porcelain` の確かめは手順 9 のとおり)",
		"- c なら、それ以上何もしない",
		"sync-docs に進まず、push も disband もせず(PR も作らない)、次の DECISION を待つ",
		"org を止めるかどうかは人が決める",
		"TASK は reviewer 座席に送り",
	})
	for _, old := range []string{"OPTION 1", "OPTION 2", "OPTION 3", "選ばれた選択肢"} {
		if strings.Contains(sub, old) {
			t.Errorf("expected %s to have none of the old cap QUESTION's form, got %q in:\n%s", leaderXreviewWhere, old, sub)
		}
	}

	msg := fencedBlockAfter(t, sub, leaderXreviewQuestionLead, leaderXreviewWhere)
	m, err := validateEscalation("org-a", msg)
	if err != nil || m.Type != protocol.TypeQuestion {
		t.Fatalf("expected escalate to accept the example as a QUESTION, got TYPE %q, err %v; message:\n%s", m.Type, err, msg)
	}
	if got := protocol.HeaderValues(msg, "XREVIEW"); !slices.Equal(got, []string{"cap_reached"}) {
		t.Errorf("expected the example QUESTION's header to have XREVIEW: cap_reached once, got %q; message:\n%s", got, msg)
	}
	if got := protocol.HeaderValues(msg, "PLAN"); len(got) != 0 {
		t.Errorf("expected no PLAN: header in the cap QUESTION, which is no plan approval, got %q", got)
	}
	var forks []string
	evidence := ""
	for _, line := range strings.Split(m.Body, "\n") {
		if strings.HasPrefix(line, "FORK") {
			forks = append(forks, line)
		}
		if rest, ok := strings.CutPrefix(line, "EVIDENCE: "); ok {
			evidence = rest
		}
	}
	if len(forks) != 1 || !leaderXreviewCapFork.MatchString(forks[0]) || !forkLine.MatchString(forks[0]) {
		t.Errorf("expected the example QUESTION's body to have one FORK 1: line with a 上限を上げる(推奨), b and c 止める in step 3's FORK form, got %q; body:\n%s", forks, m.Body)
	}
	if !strings.HasPrefix(m.Body, "SUMMARY: ") {
		t.Errorf("expected the example QUESTION's body to start with SUMMARY:, got body:\n%s", m.Body)
	}
	if !leaderXreviewReportPattern(t).MatchString(evidence) {
		t.Errorf("expected the example QUESTION's EVIDENCE: to be a report of org-a the state command counts, got %q", evidence)
	}

	director := readRepoText(t, filepath.Join(".claude", "skills", "director", "SKILL.md"))
	for _, want := range []string{"TYPE: DECISION\n\nSUMMARY: ", "`FORK <番号>: <記号> <案>`"} {
		if !containsPhrase(director, want) {
			t.Errorf("expected the /director skill's DECISION to have %q, which the cap QUESTION's answer is read by, but it does not", want)
		}
	}
}

// leaderXreviewKey matches a body line in backquotes, `KEY: value`, and
// captures KEY.
var leaderXreviewKey = regexp.MustCompile("`([A-Z_]+): [^`]*`")

// reviewerXreviewTaskKeys returns the keys of the TASK's body lines the
// reviewer template's cross-review section lists, in order.
func reviewerXreviewTaskKeys(t *testing.T) []string {
	t.Helper()
	intro, _, _ := strings.Cut(reviewerXreviewSection(t), "\n1. ")
	var keys []string
	for _, line := range strings.Split(intro, "\n") {
		if !strings.HasPrefix(line, "- `") {
			continue
		}
		for _, m := range leaderXreviewKey.FindAllStringSubmatch(line, -1) {
			keys = append(keys, m[1])
		}
	}
	return keys
}

// TestRenderRolePrompt_Leader_XreviewTaskCarriesTheReviewersLines pins AC2
// and D5: the TASK goes to the reviewer seat with TASK_ID
// xreview-<org>-c<k>, -r2, -r3 added when the same <k> is sent again, the
// count read from the TASK_ID the reviewer answered (D7b); its
// body lines (not headers) are the implementer's driver of step 5, the base
// of step 9, the cycle, the report path, and the reviewer lines copied from
// the task's two reviewer lines, which an old org's task lacks; a per-plan
// org's reviewer TASK needs no PLAN:. The keys the leader lists and its
// example sends are the ones the reviewer template's section reads. The
// example is a valid TASK whose TASK_ID starts with xreview-, whose REPORT:
// passes the reviewer's step 1 check and is a report the state command
// counts, with the TASK_ID's number.
func TestRenderRolePrompt_Leader_XreviewTaskCarriesTheReviewersLines(t *testing.T) {
	sub := leaderXreviewSubsection(t)
	assertPhrasesInOrder(t, sub, leaderXreviewWhere, []string{
		"TASK は reviewer 座席に送り、TASK_ID は `xreview-org-a-c<k>`(`<k>` は `k:` の値)にする",
		"reviewer は TASK_ID が `xreview-` で始まる TASK を cross-review の TASK として扱う",
		"同じ `<k>` でもう一度送るとき(下の reviewer の QUESTION と `XREVIEW: git_changed` のあと。どちらも report は書かれない)は、TASK_ID の終わりに `-r2`、`-r3` と足して、前と違う TASK_ID にする",
		"何回目の送り直しかは、記憶でなく reviewer が答えた TASK_ID から読む(`xreview-org-a-c<k>` への答えなら次は `-r2`、`-r2` への答えなら `-r3`)",
		leaderXreviewLinesLead + "(`CHECKS:` と同じく本文の行で、ヘッダにはしない)",
		"`IMPLEMENTER_DRIVER: <claude|codex>`: 手順 5 で implementer を spawn したときの `--driver` の値",
		"`XREVIEW_BASE: origin/<既定のブランチ>`: 既定のブランチは手順 9 の決め方で決める",
		leaderXreviewCycleLine,
		"`REPORT: docs/reports/review-xreview-<YYYY-MM-DD>-org-a-c<k>.md`: `<YYYY-MM-DD>` は今日の日付、`<k>` は `k:` の値",
		"`REVIEWER_CODEX: <モデル> <effort>` と `REVIEWER_CLAUDE: <モデル> <effort>`: タスクの `" + featureReviewerCodexLead + "` と `" + featureReviewerClaudeLead +
			"` の行の 2 語を、「計画助言」と同じ読み方で読んで写す",
		"タスクにその行がなければ(その行を書かない古い ralph が起こした org)、この 2 行を書かずに送る(reviewer は相手を呼ばず、incomplete で返す)",
		leaderXreviewTaskLead + "(承認の検査が見るのは implementer への TASK だけ)",
		"メッセージは「typed protocol」の節のとおりファイルに書いて `ralph org send` で送る",
	})

	_, lines, _ := strings.Cut(sub, leaderXreviewLinesLead)
	lines, _, _ = strings.Cut(lines, leaderXreviewTaskLead)
	var listed []string
	for _, line := range strings.Split(lines, "\n") {
		if strings.HasPrefix(line, "- `") {
			for _, m := range leaderXreviewKey.FindAllStringSubmatch(line, -1) {
				listed = append(listed, m[1])
			}
		}
	}
	reviewerKeys := reviewerXreviewTaskKeys(t)
	if len(reviewerKeys) == 0 || !slices.Equal(listed, reviewerKeys) {
		t.Errorf("expected the leader's TASK lines %q to be the reviewer's %q, in the same order", listed, reviewerKeys)
	}

	msg := fencedBlockAfter(t, sub, leaderXreviewTaskLead, leaderXreviewWhere)
	m, err := protocol.Parse(msg)
	if err != nil || protocol.Validate(m, protocol.DefaultMaxBodyChars) != nil || m.Type != protocol.TypeTask {
		t.Fatalf("expected the example to be a valid TASK, got %+v, err %v; message:\n%s", m, err, msg)
	}
	if !strings.HasPrefix(m.TaskID, "xreview-") || m.TaskID != "xreview-org-a-c1" {
		t.Errorf("expected the example's TASK_ID to be xreview-org-a-c1, which starts with xreview-, got %q", m.TaskID)
	}
	if v := protocol.HeaderValues(msg, "PLAN"); len(v) != 0 {
		t.Errorf("expected no PLAN: header in the reviewer's TASK, got %q", v)
	}
	var sent []string
	values := map[string]string{}
	for _, line := range strings.Split(m.Body, "\n") {
		key, value, ok := strings.Cut(line, ": ")
		if !ok || key == "SUMMARY" || !regexp.MustCompile(`^[A-Z_]+$`).MatchString(key) {
			continue
		}
		sent = append(sent, key)
		values[key] = value
		if len(protocol.HeaderValues(msg, key)) != 0 {
			t.Errorf("expected %s: in the example's body, not its header block; message:\n%s", key, msg)
		}
	}
	if !slices.Equal(sent, reviewerKeys) {
		t.Errorf("expected the example's body lines %q to be the reviewer's %q, in the same order", sent, reviewerKeys)
	}
	if d := values["IMPLEMENTER_DRIVER"]; d != "claude" && d != "codex" {
		t.Errorf("expected the example's IMPLEMENTER_DRIVER: to be claude or codex, got %q", d)
	}
	if b := values["XREVIEW_BASE"]; !strings.HasPrefix(b, "origin/") {
		t.Errorf("expected the example's XREVIEW_BASE: to be origin/<既定のブランチ>, got %q", b)
	}
	if c := values["CYCLE"]; !regexp.MustCompile(`^1/\d+$`).MatchString(c) {
		t.Errorf("expected the example's CYCLE: to be 1/<上限> for its first report, got %q", c)
	}
	report := strings.Replace(values["REPORT"], "<YYYY-MM-DD>", "2026-11-01", 1)
	if !strings.HasPrefix(report, "docs/reports/review-xreview-") || !strings.HasSuffix(report, "-org-a-c1.md") || !leaderXreviewReportPattern(t).MatchString(report) {
		t.Errorf("expected the example's REPORT: to be docs/reports/review-xreview-<YYYY-MM-DD>-org-a-c1.md, a path the reviewer's step 1 takes and the state command counts, got %q", values["REPORT"])
	}
}

// TestRenderRolePrompt_Leader_XreviewHandlesEveryReply pins AC5 and D5, D7,
// D10: the leader waits without changing the worktree or tasking the
// implementer, then sorts each reply the reviewer template's section gives.
// pass keeps ran (<reviewer>); skipped and incomplete keep their value, are
// not counted, and are not asked for again; action_required has the leader
// decide each finding, run the state command again, and raise the cap
// QUESTION at the cap or ask the implementer for the fixes (with PLAN: in a
// per-plan org) and go back through step 8, the next run counted;
// git_changed is a BLOCKED with the reviewer's two git state files, as the
// plan advisory's git: changed; a QUESTION about the TASK's lines gets the
// TASK again under a new TASK_ID, or a BLOCKED, as step 12's gate QUESTION.
// Every XREVIEW: value the reviewer template returns has its item.
func TestRenderRolePrompt_Leader_XreviewHandlesEveryReply(t *testing.T) {
	sub := leaderXreviewSubsection(t)
	assertPhrasesInOrder(t, sub, leaderXreviewWhere, []string{
		"送ったら、返事が来るまで worktree を変えず(コミット・rebase・ファイルの書き換えをしない。reviewer は相手を呼ぶ前と後の git の状態を比べる)、implementer に TASK を送らない(待ち方は「agmsg の受信箱の運用」の節)",
		"- `XREVIEW: pass` の RESULT: 手順 14 のために `Cross-review: ran (<相手>)` を控え(`<相手>` は report の `Reviewer:` の値)、手順 9 の sync-docs に進む",
		"- `XREVIEW: skipped (<理由>)` か `XREVIEW: incomplete (<理由>)` の RESULT: その値(`skipped (<理由>)` か `incomplete (<理由>)`)を `Cross-review:` の行として控え、手順 9 の sync-docs に進む。この回は上限に数えず、頼み直さない",
		"- `XREVIEW: action_required` の BLOCKED: report を読み、ACTION_REQUIRED の所見を 1 件ずつ、直すか直さないかを決め、直さないものはその理由を手順 14 の PR の本文のために控える",
		"続けて、この小節の初めのコマンドを打ち直す",
		"`done:` が `cap:` 以上なら(この回で上限に届いた)、implementer に TASK を送らず、上の上限の QUESTION に進む(直すと決めたものは、人が上限を上げたら頼む)",
		"`cap:` より小さければ、直すものを implementer に TASK で頼み(per-plan では `PLAN:` のヘッダを付ける)、手順 8 の往復に戻る",
		"手順 9 を通り直すと次の回の cross-review を頼み、その回も上限に数える",
		"直すものがなければ、`Cross-review: ran (<相手>)` を控えて手順 9 の sync-docs に進む",
		"- `XREVIEW: git_changed` の BLOCKED: reviewer が相手を呼ぶ前と後で git の状態が違い、report は書かれていない",
		"「計画助言」の `git: changed` と同じく、変わった worktree のまま先に進まず、`.harness/state/xreview-org-a-c<k>-git-before.txt` と " +
			"`.harness/state/xreview-org-a-c<k>-git-after.txt` を EVIDENCE に書いて BLOCKED(TASK_ID は `org-a`)を `ralph org escalate` で上げ、返事が来るまで止まる",
		"- 本文の行についての reviewer の QUESTION(TASK の行が欠けているか形が違うので、誰も呼ばなかった): 欠けた行か形の違う行をそろえ、新しい TASK_ID で送り直す(手順 12 の gate の TASK への QUESTION と同じ)",
		"何が違ったのかわからなければ、送り直さず、QUESTION の要点を本文に書いて `ralph org escalate` で BLOCKED を上げる",
		"WORTH_CONSIDERING の所見は、あなたが扱いを決め、PR の本文に残す",
	})

	// The reviewer's values and the files git_changed names are the
	// reviewer template's.
	reviewer := reviewerXreviewSection(t)
	for _, value := range []string{"pass", "action_required", "skipped (", "incomplete (", "git_changed"} {
		if !strings.Contains(reviewer, "XREVIEW: "+value) {
			t.Errorf("expected %s to return XREVIEW: %s, which %s handles", reviewerXreviewWhere, value, leaderXreviewWhere)
		}
		if !strings.Contains(sub, "\n- `XREVIEW: "+value) && !strings.Contains(sub, "か `XREVIEW: "+value) {
			t.Errorf("expected %s to have an item for XREVIEW: %s, which %s returns", leaderXreviewWhere, value, reviewerXreviewWhere)
		}
	}
	for _, file := range []string{".harness/state/xreview-org-a-c<k>-git-before.txt", ".harness/state/xreview-org-a-c<k>-git-after.txt"} {
		if !strings.Contains(reviewer, "`"+file+"`") {
			t.Errorf("expected %s to write %s, which %s names for XREVIEW: git_changed", reviewerXreviewWhere, file, leaderXreviewWhere)
		}
	}
}

// TestRenderRolePrompt_Leader_XreviewBlockedAndStallFollowTheGates pins
// D9 and AC6 for the leader template: the mission's item 5 leaves a BLOCKED
// to an xreview- TASK, which carries XREVIEW: and no GATE:, to the
// cross-review subsection, after its sentence for a gate- TASK; step 12 says
// a stall ALERT about the reviewer while a cross-review TASK runs is treated
// as one while the gate runs, after the text the gate tests pin, since the
// other model's call can take up to 20 minutes (the reviewer's watchdog,
// sleep 1200); and the subsection says the same where it sends the TASK.
func TestRenderRolePrompt_Leader_XreviewBlockedAndStallFollowTheGates(t *testing.T) {
	text := renderSeatPrompt(t, "leader")
	mission, _ := markdownSection(text, "## ミッション")
	item, ok := markdownItem(mission, "5. 座席からの")
	if !ok {
		t.Fatalf("expected the leader mission to have item 5, got section:\n%s", mission)
	}
	assertPhrasesInOrder(t, item, "the mission's item 5", []string{
		"機能ごとの org で gate を頼んだ TASK(TASK_ID が `gate-` で始まる TASK)への BLOCKED は、「機能ごとの org」の節の手順 12 のとおりに扱う",
		"cross-review を頼んだ TASK(TASK_ID が `xreview-` で始まる TASK)への BLOCKED は、`GATE:` ではなく本文の `XREVIEW:` の行を持つので、「機能ごとの org」の節の終わりの「cross-review」の小節のとおりに扱う",
	})

	stallAlert := "ALERT(`SEAT: reviewer`、`CONDITION: " + condStall + "`)"
	assertPhrasesInOrder(t, leaderFeatureOrgItem(t, step12Marker), leaderItemWhere(step12Marker), []string{
		"この間に見張りから reviewer の停滞の " + stallAlert + "が来ても、reviewer を stop も spawn し直しもしない",
		"`ralph org read` で reviewer の画面を読み、gate が走っていれば待つ(gate は走っているあいだ何も記録せず、Codex の reviewer は前面で打つので、画面が長く変わらないことがある)",
		"手順 9 で cross-review の TASK(この節の終わりの「cross-review」の小節)の返事を待つあいだに同じ ALERT が来たときも、同じく扱う",
		"reviewer の画面で相手のモデルの呼び出しが走っていれば待つ(呼び出しは 20 分まで続くことがあり、そのあいだ reviewer は何も送らない)",
		"gate のあとにコミットを足さず、足したら gate を頼み直す",
	})
	assertPhrasesInOrder(t, leaderXreviewSubsection(t), leaderXreviewWhere, []string{
		"相手のモデルの呼び出しは 20 分まで続くことがある",
		"この間に見張りから reviewer の停滞の ALERT が来たときは、手順 12 の gate のときと同じく扱う(reviewer を stop も spawn し直しもせず、`ralph org read` で画面を読み、呼び出しが走っていれば待つ)",
	})
	if reviewer := reviewerXreviewSection(t); !strings.Contains(reviewer, "sleep 1200") || !containsPhrase(reviewer, "`sleep 1200` の見張りで相手を 20 分で止め") {
		t.Errorf("expected %s to stop the other model after 20 minutes with sleep 1200, the wait the leader template names, got:\n%s", reviewerXreviewWhere, reviewer)
	}
}

// TestRenderRolePrompt_Leader_XreviewGoesInThePRBodyAndTheResult pins AC1
// and D10: step 14 puts the PR template's - Cross-review: line in the PR
// body with ran (<reviewer>), skipped (<reason>) (off included) or
// incomplete (<reason>), this org's cross-review reports in place of the
// template's path, and after it the findings not fixed, the handling of
// WORTH_CONSIDERING, and the known gaps or unseen change a human let through;
// the RESULT carries the same value on one CROSS_REVIEW: body line next to
// DOC_FOLLOWUP: or SYNC_DOCS:. The value is the one the subsection keeps for
// the last pass of step 9, read from the last report's Reviewer status:
// line. The PR template has those values, and the example RESULT, one
// escalate accepts, has one such line in its body.
func TestRenderRolePrompt_Leader_XreviewGoesInThePRBodyAndTheResult(t *testing.T) {
	step14 := leaderFeatureOrgItem(t, leaderStep14Marker)
	assertPhrasesInOrder(t, step14, leaderItemWhere(leaderStep14Marker), []string{
		"(最も新しい sync-docs のコミットは前の回のもので、その report は最後の回のものではない)",
		"PR の本文には、本文の雛形の `- Cross-review:` の行も置き(雛形にその行がなければ、本文の終わりに置く)、この節の終わりの「cross-review」の小節で控えた値を書く",
		"値は `ran (<相手>)`、`skipped (<理由>)`(off のときも)、`incomplete (<理由>)` のどれかで、" +
			"雛形が行の終わりの括弧に書く report のパスは、このブランチの cross-review の report" +
			"(「cross-review」の小節の `done:` の行と同じく、base からこのブランチが足すか書き換えたもの)のパスにする(複数あれば並べ、report がなければ括弧を書かない)",
		"その行のあとに、直さなかった ACTION_REQUIRED の所見とその理由、WORTH_CONSIDERING の扱い、上限の QUESTION で人が進むと答えたときの known gaps か相手に見せていない変更を書く",
		"PR ができたら、PR の URL を EVIDENCE に書いて `ralph org escalate` で RESULT を上げる",
		"`DOC_FOLLOWUP:` か `SYNC_DOCS:` の行と並べて、PR の本文の `Cross-review:` の行と同じ値を `CROSS_REVIEW: <値>` の 1 行で書く",
		"たとえば次のメッセージにする",
	})
	assertPhrasesInOrder(t, leaderXreviewSubsection(t), leaderXreviewWhere, []string{
		"手順 14 の `Cross-review:` の値は、最後に手順 9 を通ったときのものにする",
		"off なら控えた `skipped (<理由>)`",
		"それ以外は、最後の cross-review の report(このブランチの report のうち番号のいちばん大きいもの)の `- Reviewer status:` の行から、`complete` なら `ran (<その report の Reviewer:>)`、" +
			"`skipped (<理由>)` と `incomplete (<理由>)` はその値にする",
		"上限で TASK を送らずに進んだときも、最後の report の値を書く",
	})

	template := readRepoText(t, filepath.Join(".claude", "skills", "pr", "template.md"))
	line := ""
	for l := range strings.SplitSeq(template, "\n") {
		if strings.HasPrefix(l, "- Cross-review: ") {
			line = l
		}
	}
	for _, value := range []string{"ran (", "skipped (", "incomplete ("} {
		if !strings.Contains(line, value) {
			t.Errorf("expected the PR template's - Cross-review: line to offer %q, which step 14 writes, got %q", value, line)
		}
	}

	msg := fencedBlockAfter(t, step14, leaderStep14Marker, leaderItemWhere(leaderStep14Marker))
	m, err := validateEscalation("org-a", msg)
	if err != nil || m.Type != protocol.TypeResult {
		t.Fatalf("expected escalate to accept the example as a RESULT, got TYPE %q, err %v; message:\n%s", m.Type, err, msg)
	}
	if got := protocol.HeaderValues(msg, "CROSS_REVIEW"); len(got) != 0 {
		t.Errorf("expected CROSS_REVIEW: in the example RESULT's body, not its header block, got %q", got)
	}
	var values []string
	for l := range strings.SplitSeq(m.Body, "\n") {
		if v, ok := strings.CutPrefix(l, "CROSS_REVIEW: "); ok {
			values = append(values, v)
		}
	}
	if len(values) != 1 || !regexp.MustCompile(`^(ran|skipped|incomplete) \(.+\)$`).MatchString(values[0]) {
		t.Errorf("expected one CROSS_REVIEW: line with ran, skipped or incomplete (<...>) in the example RESULT's body, got %q; body:\n%s", values, m.Body)
	}
}

// leaderXreviewRalphStub stands in for ralph config get. STUB_RALPH=old exits
// 1 with the line of a ralph without the config verb, fail1 exits 1 with
// another line, and exit3 exits 3 as ralph does for a ralph.toml it cannot
// read. Otherwise it prints STUB_SWITCH (true when unset) for
// org.review.cross_review and STUB_CAP (2 when unset) for
// org.review.max_cross_review_cycles. Any other call exits 9.
const leaderXreviewRalphStub = `#!/bin/sh
[ "$1 $2" = "config get" ] || { echo "stub: unexpected ralph $*" >&2; exit 9; }
case $STUB_RALPH in
old) echo 'unknown command "config" for "ralph"' >&2; exit 1 ;;
fail1) echo 'stub: something else went wrong' >&2; exit 1 ;;
exit3) echo 'stub: cannot read ralph.toml' >&2; exit 3 ;;
esac
case $3 in
org.review.cross_review) echo "${STUB_SWITCH-true}" ;;
org.review.max_cross_review_cycles) echo "${STUB_CAP-2}" ;;
*) echo "stub: unexpected key $3" >&2; exit 9 ;;
esac
`

// leaderXreviewFixture is a git repository with an ignored .harness/state/
// and refs/remotes/origin/main at its first commit (the base of step 9 after
// its fetch), a directory with the stand-in ralph, and a directory of
// wrappers for the commands the state command runs other than the shells'
// builtins, so that a PATH of the wrappers alone has no ralph.
type leaderXreviewFixture struct {
	repo, stubs, tools string
}

func newLeaderXreviewFixture(t *testing.T) leaderXreviewFixture {
	t.Helper()
	dir := t.TempDir()
	fx := leaderXreviewFixture{repo: filepath.Join(dir, "repo"), stubs: filepath.Join(dir, "stubs"), tools: filepath.Join(dir, "tools")}
	for _, name := range []string{"git", "grep", "sed", "head", "mkdir"} {
		path, err := exec.LookPath(name)
		if err != nil {
			t.Skipf("%s not available", name)
		}
		if strings.Contains(path, "'") {
			t.Skipf("%s is at a path with a quote: %s", name, path)
		}
		writeTestExecutable(t, filepath.Join(fx.tools, name), "#!/bin/sh\nexec '"+path+"' \"$@\"\n")
	}
	writeTestExecutable(t, filepath.Join(fx.stubs, "ralph"), leaderXreviewRalphStub)
	fx.write(t, ".gitignore", ".harness/state/\n")
	fx.write(t, "code.txt", "one\n")
	fx.git(t, "init", "--quiet", "--template=")
	fx.commit(t, "init", ".gitignore", "code.txt")
	fx.git(t, "update-ref", "refs/remotes/origin/main", "HEAD")
	return fx
}

// leaderXreviewGitEnv keeps the user's git configuration out of the
// fixture's git and the state command's.
var leaderXreviewGitEnv = []string{"GIT_CONFIG_GLOBAL=" + os.DevNull, "GIT_CONFIG_NOSYSTEM=1"}

// git runs git in the fixture's repository and returns its stdout, trimmed.
func (fx leaderXreviewFixture) git(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = fx.repo
	cmd.Env = append(advisoryTestEnv(), leaderXreviewGitEnv...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

// commit commits paths with message and returns the new HEAD.
func (fx leaderXreviewFixture) commit(t *testing.T, message string, paths ...string) string {
	t.Helper()
	fx.git(t, append([]string{"add", "--"}, paths...)...)
	fx.git(t, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false", "commit", "--quiet", "-m", message)
	return fx.git(t, "rev-parse", "HEAD")
}

// write writes content to rel in the fixture's repository.
func (fx leaderXreviewFixture) write(t *testing.T, rel, content string) {
	t.Helper()
	path := filepath.Join(fx.repo, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// state runs block with shell (zsh with -f) in the fixture's repository, with
// the wrappers on PATH and, when ralph is true, the stand-in ralph before
// them, and returns the lines it prints as a map from each line's key (the
// text before ": ") to its value. It fails the test on a non-zero exit, on a
// line without ": ", and on a key printed twice.
func (fx leaderXreviewFixture) state(t *testing.T, shell, block string, ralph bool, env ...string) map[string]string {
	t.Helper()
	path := fx.tools
	if ralph {
		path = fx.stubs + string(os.PathListSeparator) + path
	}
	args := []string{"-c", block}
	if filepath.Base(shell) == "zsh" {
		args = append([]string{"-f"}, args...)
	}
	cmd := exec.Command(shell, args...)
	cmd.Dir = fx.repo
	cmd.Env = append(advisoryTestEnv(), append(append([]string{"PATH=" + path}, leaderXreviewGitEnv...), env...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s -c <state command>: %v\n%s", filepath.Base(shell), err, out)
	}
	got := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		key, value, ok := strings.Cut(line, ": ")
		if _, dup := got[key]; !ok || dup {
			t.Fatalf("expected the state command to print each key once as <key>: <value>, got the line %q in:\n%s", line, out)
		}
		got[key] = value
	}
	return got
}

// wantState fails the test for each key of want whose value in got differs.
func wantState(t *testing.T, name string, got, want map[string]string) {
	t.Helper()
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s: the state command printed %s: %q, want %q (all: %q)", name, key, got[key], value, got)
		}
	}
}

// TestRenderRolePrompt_Leader_XreviewStateCommandReadsTheSwitchAndTheCap runs
// the subsection's state command with bash and zsh against a stand-in ralph
// (AC1, D1, D7): true is switch: on; false, an older ralph's exit 1 with
// unknown command "config", and no ralph on PATH (127) are off with the
// subsection's reasons; exit 3, another exit 1, and a value that is neither
// true nor false are errors, and the stderr of the call is left in the file
// the subsection names. The cap is the setting, 2 for an older ralph or
// none, raised by a positive integer on the cap file's first line and never
// lowered by it; a cap file that is not a positive integer is ignored, and a
// setting that is not one is an error. With no report, done: is 0, k: is 1
// and last: is none.
func TestRenderRolePrompt_Leader_XreviewStateCommandReadsTheSwitchAndTheCap(t *testing.T) {
	block := leaderXreviewStateBlock(t)
	for _, shell := range advisoryShells(t, false) {
		t.Run(filepath.Base(shell), func(t *testing.T) {
			t.Parallel()
			fx := newLeaderXreviewFixture(t)
			capFile := filepath.Join(fx.repo, ".harness", "state", "xreview-org-a-cap")
			for _, c := range []struct {
				name    string
				ralph   bool
				env     []string
				capFile string
				want    map[string]string
			}{
				{"on", true, nil, "", map[string]string{"switch": "on", "cap": "2", "base": "origin/main", "done": "0", "k": "1", "last": "none"}},
				{"off", true, []string{"STUB_SWITCH=false"}, "", map[string]string{"switch": "off (off: [org.review].cross_review)", "cap": "2"}},
				{"an older ralph", true, []string{"STUB_RALPH=old"}, "", map[string]string{"switch": "off (ralph without config get; treated as off)", "cap": "2"}},
				{"no ralph", false, nil, "", map[string]string{"switch": "off (ralph not found; treated as off)", "cap": "2", "done": "0"}},
				{"exit 3", true, []string{"STUB_RALPH=exit3"}, "", map[string]string{"switch": "error (exit 3)", "cap": "error (exit 3)"}},
				{"another exit 1", true, []string{"STUB_RALPH=fail1"}, "", map[string]string{"switch": "error (exit 1)", "cap": "error (exit 1)"}},
				{"neither true nor false", true, []string{"STUB_SWITCH=yes"}, "", map[string]string{"switch": "error (exit 0)"}},
				{"no value", true, []string{"STUB_SWITCH="}, "", map[string]string{"switch": "error (exit 0)"}},
				{"a raised cap", true, []string{"STUB_CAP=3"}, "5\n", map[string]string{"cap": "5"}},
				{"a cap file below the setting", true, []string{"STUB_CAP=3"}, "1\n", map[string]string{"cap": "3"}},
				{"a cap file read to its first line", true, nil, "4\n9\n", map[string]string{"cap": "4"}},
				{"a cap file that is not a number", true, nil, "three\n", map[string]string{"cap": "2"}},
				{"a cap file of 0", true, nil, "0\n", map[string]string{"cap": "2"}},
				{"a negative cap file", true, nil, "-4\n", map[string]string{"cap": "2"}},
				{"an older ralph and a raised cap", true, []string{"STUB_RALPH=old"}, "4\n", map[string]string{"cap": "4"}},
				{"no ralph and a raised cap", false, nil, "3\n", map[string]string{"cap": "3"}},
				{"a setting that is not a number", true, []string{"STUB_CAP=many"}, "", map[string]string{"cap": "error (exit 0)"}},
				{"a setting of 0", true, []string{"STUB_CAP=0"}, "5\n", map[string]string{"cap": "error (exit 0)"}},
			} {
				if err := os.RemoveAll(capFile); err != nil {
					t.Fatal(err)
				}
				if c.capFile != "" {
					fx.write(t, filepath.Join(".harness", "state", "xreview-org-a-cap"), c.capFile)
				}
				got := fx.state(t, shell, block, c.ralph, c.env...)
				wantState(t, c.name, got, c.want)
				if c.want["last"] == "none" {
					for _, key := range []string{"last head", "last action_required", "since"} {
						if _, ok := got[key]; ok {
							t.Errorf("%s: expected no %s: line with last: none, got %q", c.name, key, got)
						}
					}
				}
				if c.name == "exit 3" {
					for _, file := range []string{"xreview-org-a-switch.err", "xreview-org-a-cap.err"} {
						if data, err := os.ReadFile(filepath.Join(fx.repo, ".harness", "state", file)); err != nil || !strings.Contains(string(data), "cannot read ralph.toml") {
							t.Errorf("expected .harness/state/%s, which the subsection has the leader read, to hold the stderr of ralph config get, got %q, err %v", file, data, err)
						}
					}
				}
			}
			if status := fx.git(t, "status", "--porcelain", "--untracked-files=all"); status != "" {
				t.Errorf("expected the state command to write only under the ignored .harness/state/, got git status:\n%s", status)
			}
		})
	}
}

// leaderXreviewReportPath returns the path of the cross-review report c<k> of
// org dated 2026-11-01, filled in from the REPORT: line the reviewer template
// reads.
func leaderXreviewReportPath(t *testing.T, org string, k int) string {
	t.Helper()
	intro, _, _ := strings.Cut(reviewerXreviewSection(t), "\n1. ")
	m := regexp.MustCompile("`REPORT: ([^`]+)`").FindStringSubmatch(intro)
	if m == nil {
		t.Fatalf("expected the opening of %s to have a REPORT: line, got:\n%s", reviewerXreviewWhere, intro)
	}
	return strings.NewReplacer("<YYYY-MM-DD>", "2026-11-01", "-org-a-", "-"+org+"-", "<k>", strconv.Itoa(k)).Replace(m[1])
}

// leaderXreviewReport returns the report of the reviewer template's step 6
// with its status, the HEAD shown to the other model, and its ACTION_REQUIRED
// count filled in.
func leaderXreviewReport(t *testing.T, status, head string, actionRequired int) string {
	t.Helper()
	block := fencedBlockAfter(t, reviewerXreviewStep(t, 6), reviewerXreviewReportLead, reviewerXreviewStepWhere(6))
	report := strings.NewReplacer(
		"- Reviewer status: <complete | skipped (<理由>) | incomplete (<理由>)>", "- Reviewer status: "+status,
		"- HEAD: <相手に見せた HEAD の SHA>", "- HEAD: "+head,
		"ACTION_REQUIRED=<数>", "ACTION_REQUIRED="+strconv.Itoa(actionRequired),
	).Replace(block)
	for _, line := range []string{"- Reviewer status: " + status, "- HEAD: " + head, "- After triage: ACTION_REQUIRED=" + strconv.Itoa(actionRequired) + ","} {
		if !strings.Contains(report, "\n"+line) {
			t.Fatalf("expected the report of %s filled in to have %q, got:\n%s", reviewerXreviewStepWhere(6), line, report)
		}
	}
	return report
}

// TestRenderRolePrompt_Leader_XreviewStateCommandCountsTheReviewersReports
// runs the subsection's state command with bash and zsh on reports written
// in the shape of the reviewer template's step 6 (D3, D7, AC5b). done:
// counts only this org's committed complete reports: not a skipped one, not
// another org's whose id ends with this one's or starts with it, not one left
// uncommitted. k: is one past the highest number, last: the complete report
// with the highest number (c10 after c3, compared as numbers), with its HEAD:
// and its ACTION_REQUIRED= count. since: is same after a commit of reports
// only, changed after a commit outside docs/reports/, and changed when the
// report's HEAD is not a commit of the repository.
func TestRenderRolePrompt_Leader_XreviewStateCommandCountsTheReviewersReports(t *testing.T) {
	block := leaderXreviewStateBlock(t)
	for _, shell := range advisoryShells(t, false) {
		t.Run(filepath.Base(shell), func(t *testing.T) {
			t.Parallel()
			fx := newLeaderXreviewFixture(t)
			h0 := fx.git(t, "rev-parse", "HEAD")
			c1, c2 := leaderXreviewReportPath(t, "org-a", 1), leaderXreviewReportPath(t, "org-a", 2)
			others := []string{leaderXreviewReportPath(t, "x-org-a", 5), leaderXreviewReportPath(t, "org-a-cache", 7)}
			fx.write(t, c1, leaderXreviewReport(t, "complete", h0, 2))
			fx.write(t, c2, leaderXreviewReport(t, "skipped (codex not available)", h0, 0))
			for _, other := range others {
				fx.write(t, other, leaderXreviewReport(t, "complete", h0, 0))
			}
			fx.commit(t, "reports", append([]string{c1, c2}, others...)...)
			fx.write(t, leaderXreviewReportPath(t, "org-a", 9), leaderXreviewReport(t, "complete", h0, 0))
			for _, other := range others {
				if leaderXreviewReportPattern(t).MatchString(other) {
					t.Errorf("expected the state command's grep to leave out %s, another org's report", other)
				}
			}

			got := fx.state(t, shell, block, true)
			wantState(t, "after c1 (complete) and c2 (skipped)", got, map[string]string{
				"done": "1", "k": "3", "last": c1, "last head": h0, "last action_required": "2", "since": "same",
			})

			fx.write(t, "code.txt", "two\n")
			h2 := fx.commit(t, "fix", "code.txt")
			got = fx.state(t, shell, block, true)
			wantState(t, "after a fix", got, map[string]string{"done": "1", "k": "3", "last": c1, "since": "changed"})

			c3 := leaderXreviewReportPath(t, "org-a", 3)
			fx.write(t, c3, leaderXreviewReport(t, "complete", h2, 0))
			fx.commit(t, "c3", c3)
			got = fx.state(t, shell, block, true)
			wantState(t, "after c3 (complete)", got, map[string]string{
				"done": "2", "k": "4", "last": c3, "last head": h2, "last action_required": "0", "since": "same",
			})

			c10 := leaderXreviewReportPath(t, "org-a", 10)
			fx.write(t, c10, leaderXreviewReport(t, "complete", strings.Repeat("0", 40), 1))
			fx.commit(t, "c10", c10)
			got = fx.state(t, shell, block, true)
			wantState(t, "after c10 (complete, an unknown HEAD)", got, map[string]string{
				"done": "3", "k": "11", "last": c10, "last action_required": "1", "since": "changed",
			})
		})
	}
}

// TestRenderRolePrompt_Leader_XreviewStateCommandCountsOnlyThisBranchsReports
// runs the subsection's state command with bash and zsh where an earlier org
// of the same org_id merged its complete reports c1 and c2 into the default
// branch (self-review M1): with none added by the branch, done: is 0, k: 1
// and last: none, below the cap, so no cap QUESTION; a report written but not
// committed is still not counted; once committed it is done: 1; and a report
// written over the earlier org's at the same path (the same date and number)
// counts as the branch's, with its own HEAD.
func TestRenderRolePrompt_Leader_XreviewStateCommandCountsOnlyThisBranchsReports(t *testing.T) {
	block := leaderXreviewStateBlock(t)
	for _, shell := range advisoryShells(t, false) {
		t.Run(filepath.Base(shell), func(t *testing.T) {
			t.Parallel()
			fx := newLeaderXreviewFixture(t)
			h0 := fx.git(t, "rev-parse", "HEAD")
			earlier1 := strings.Replace(leaderXreviewReportPath(t, "org-a", 1), "2026-11-01", "2026-10-01", 1)
			earlier2 := leaderXreviewReportPath(t, "org-a", 2)
			fx.write(t, earlier1, leaderXreviewReport(t, "complete", h0, 1))
			fx.write(t, earlier2, leaderXreviewReport(t, "complete", h0, 0))
			fx.commit(t, "an earlier org's reports", earlier1, earlier2)
			fx.git(t, "update-ref", "refs/remotes/origin/main", "HEAD")

			got := fx.state(t, shell, block, true)
			wantState(t, "with the earlier org's reports on the base only", got, map[string]string{
				"base": "origin/main", "cap": "2", "done": "0", "k": "1", "last": "none",
			})
			if done, err := strconv.Atoi(got["done"]); err != nil || got["cap"] != "2" || done >= 2 {
				t.Errorf("expected done: below cap: 2, so no cap QUESTION, got %q", got)
			}

			c1 := leaderXreviewReportPath(t, "org-a", 1)
			fx.write(t, c1, leaderXreviewReport(t, "complete", h0, 0))
			got = fx.state(t, shell, block, true)
			wantState(t, "with c1 written, not committed", got, map[string]string{"done": "0", "k": "1", "last": "none"})

			fx.commit(t, "c1", c1)
			got = fx.state(t, shell, block, true)
			wantState(t, "with c1 committed", got, map[string]string{"done": "1", "k": "2", "last": c1, "last head": h0, "since": "same"})

			fx.write(t, "code.txt", "two\n")
			h2 := fx.commit(t, "fix", "code.txt")
			fx.write(t, earlier2, leaderXreviewReport(t, "complete", h2, 0))
			fx.commit(t, "c2 over the earlier org's c2", earlier2)
			got = fx.state(t, shell, block, true)
			wantState(t, "with c2 written over the earlier org's", got, map[string]string{
				"done": "2", "k": "3", "last": earlier2, "last head": h2, "since": "same",
			})
		})
	}
}

// TestRenderRolePrompt_Leader_XreviewStateCommandFindsStep9sBase runs the
// subsection's state command with bash and zsh for the base: line (self-review
// M1): the branch refs/remotes/origin/HEAD points to comes first, then
// origin/main, then origin/master, as step 9 picks the default branch; the
// reports a base already holds are not the branch's; with none of the three,
// base: is an error the subsection turns into a BLOCKED.
func TestRenderRolePrompt_Leader_XreviewStateCommandFindsStep9sBase(t *testing.T) {
	block := leaderXreviewStateBlock(t)
	for _, shell := range advisoryShells(t, false) {
		t.Run(filepath.Base(shell), func(t *testing.T) {
			t.Parallel()
			fx := newLeaderXreviewFixture(t)
			h0 := fx.git(t, "rev-parse", "HEAD")
			c1 := leaderXreviewReportPath(t, "org-a", 1)
			fx.write(t, c1, leaderXreviewReport(t, "complete", h0, 0))
			fx.commit(t, "c1", c1)
			fx.git(t, "update-ref", "refs/remotes/origin/trunk", "HEAD")
			fx.git(t, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")

			got := fx.state(t, shell, block, true)
			wantState(t, "with origin/HEAD at origin/trunk", got, map[string]string{"base": "origin/trunk", "done": "0", "k": "1"})

			fx.git(t, "symbolic-ref", "--delete", "refs/remotes/origin/HEAD")
			got = fx.state(t, shell, block, true)
			wantState(t, "with no origin/HEAD", got, map[string]string{"base": "origin/main", "done": "1", "k": "2", "last": c1})

			fx.git(t, "update-ref", "-d", "refs/remotes/origin/main")
			got = fx.state(t, shell, block, true)
			wantState(t, "with neither origin/HEAD nor origin/main nor origin/master", got, map[string]string{
				"base": "error (no origin/HEAD, origin/main, or origin/master)",
			})

			fx.git(t, "update-ref", "refs/remotes/origin/master", "HEAD")
			got = fx.state(t, shell, block, true)
			wantState(t, "with origin/master alone", got, map[string]string{"base": "origin/master", "done": "0", "k": "1"})
		})
	}
}

// TestRenderRolePrompt_Leader_XreviewRaisedCapReachesTheTask pins the answer
// a of the cap QUESTION (self-review L1): the new cap is the DECISION's
// number when it is above done:, otherwise done: + 1, written to the cap
// file, and the TASK the second QUESTION sends at once takes its CYCLE:'s cap
// from the state command run again after the write. It runs the state
// command with bash and zsh at the cap (done: 2, cap: 2) and, for each
// DECISION, writes the cap the rule gives and runs it again: the fresh cap:
// is that cap, done: is below it so the TASK goes, and the TASK's CYCLE: is
// 3/<the new cap>, where the cap: read before the write gives 3/2. Writing a
// DECISION's 2 itself would leave done: at the cap, the same QUESTION again.
func TestRenderRolePrompt_Leader_XreviewRaisedCapReachesTheTask(t *testing.T) {
	assertPhrasesInOrder(t, leaderXreviewSubsection(t), leaderXreviewWhere, []string{
		leaderXreviewRaiseA,
		leaderXreviewRaiseRule,
		leaderXreviewRaiseRerun,
		leaderXreviewCycleLine,
	})

	block := leaderXreviewStateBlock(t)
	for _, shell := range advisoryShells(t, false) {
		t.Run(filepath.Base(shell), func(t *testing.T) {
			t.Parallel()
			fx := newLeaderXreviewFixture(t)
			h0 := fx.git(t, "rev-parse", "HEAD")
			c1, c2 := leaderXreviewReportPath(t, "org-a", 1), leaderXreviewReportPath(t, "org-a", 2)
			fx.write(t, c1, leaderXreviewReport(t, "complete", h0, 1))
			fx.write(t, c2, leaderXreviewReport(t, "complete", h0, 1))
			fx.commit(t, "c1 and c2", c1, c2)
			fx.write(t, "code.txt", "two\n")
			fx.commit(t, "a fix after the cap", "code.txt")
			capFile := filepath.Join(".harness", "state", "xreview-org-a-cap")

			before := fx.state(t, shell, block, true)
			wantState(t, "at the cap", before, map[string]string{"done": "2", "cap": "2", "since": "changed"})
			done, err := strconv.Atoi(before["done"])
			if err != nil {
				t.Fatalf("done: %q is not a number", before["done"])
			}
			stale := strconv.Itoa(done+1) + "/" + before["cap"]

			fx.write(t, capFile, "2\n")
			if got := fx.state(t, shell, block, true); got["done"] != "2" || got["cap"] != "2" {
				t.Errorf("expected a DECISION's 2 written as it is to leave done: 2 at cap: 2, the same QUESTION again, got %q", got)
			}

			for _, c := range []struct {
				decision string
				want     string
			}{
				{"", "3"},
				{"2", "3"},
				{"1", "3"},
				{"5", "5"},
			} {
				raised := done + 1
				if n, err := strconv.Atoi(c.decision); err == nil && n > done {
					raised = n
				}
				fx.write(t, capFile, strconv.Itoa(raised)+"\n")
				got := fx.state(t, shell, block, true)
				if got["cap"] != c.want || got["done"] != "2" {
					t.Errorf("DECISION %q: expected the state command run again to print cap: %s with done: 2, got %q", c.decision, c.want, got)
				}
				if cycle := strconv.Itoa(done+1) + "/" + got["cap"]; cycle != "3/"+c.want || cycle == stale {
					t.Errorf("DECISION %q: expected the TASK's CYCLE: 3/%s from the fresh cap:, not %s from the one read before the write, got %s", c.decision, c.want, stale, cycle)
				}
			}
		})
	}
}

// hasCommit reports whether rev names a commit object of the fixture's
// repository.
func (fx leaderXreviewFixture) hasCommit(t *testing.T, rev string) bool {
	t.Helper()
	cmd := exec.Command("git", "cat-file", "-e", rev+"^{commit}")
	cmd.Dir = fx.repo
	cmd.Env = append(advisoryTestEnv(), leaderXreviewGitEnv...)
	return cmd.Run() == nil
}

// TestRenderRolePrompt_Leader_XreviewStateCommandSeesARebaseAsAChange runs
// the subsection's state command with bash and zsh across the rebase of step
// 9 (D7b): a complete report of a fix, committed, is since: same; a rebase
// onto an upstream commit, with no fix after the report, is since: changed
// while the report's HEAD: object still exists (the upstream file differs),
// and still changed once that object is gone after a gc, when the diff from
// it fails. done: and last: stay the report's all along.
func TestRenderRolePrompt_Leader_XreviewStateCommandSeesARebaseAsAChange(t *testing.T) {
	block := leaderXreviewStateBlock(t)
	for _, shell := range advisoryShells(t, false) {
		t.Run(filepath.Base(shell), func(t *testing.T) {
			t.Parallel()
			fx := newLeaderXreviewFixture(t)
			fx.git(t, "branch", "upstream")
			fx.write(t, "code.txt", "two\n")
			h1 := fx.commit(t, "fix", "code.txt")
			c1 := leaderXreviewReportPath(t, "org-a", 1)
			fx.write(t, c1, leaderXreviewReport(t, "complete", h1, 0))
			fx.commit(t, "c1", c1)
			got := fx.state(t, shell, block, true)
			wantState(t, "after c1 (complete)", got, map[string]string{"done": "1", "k": "2", "last": c1, "last head": h1, "since": "same"})

			fx.git(t, "checkout", "--quiet", "upstream")
			fx.write(t, "upstream.txt", "one\n")
			fx.commit(t, "upstream", "upstream.txt")
			fx.git(t, "checkout", "--quiet", "-")
			fx.git(t, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false", "rebase", "--quiet", "upstream")
			if !fx.hasCommit(t, h1) || fx.git(t, "rev-parse", "HEAD~1") == h1 {
				t.Fatalf("expected the rebase to rewrite the fix %s and keep its object for now", h1)
			}
			got = fx.state(t, shell, block, true)
			wantState(t, "after a rebase onto upstream", got, map[string]string{"done": "1", "k": "2", "last": c1, "last head": h1, "since": "changed"})

			fx.git(t, "update-ref", "-d", "ORIG_HEAD")
			fx.git(t, "reflog", "expire", "--expire=now", "--all")
			fx.git(t, "gc", "--quiet", "--prune=now")
			if fx.hasCommit(t, h1) {
				t.Fatalf("expected the gc to remove the fix %s the rebase left unreachable", h1)
			}
			got = fx.state(t, shell, block, true)
			wantState(t, "after a gc removed the report's HEAD", got, map[string]string{"done": "1", "k": "2", "last": c1, "last head": h1, "since": "changed"})
		})
	}
}
