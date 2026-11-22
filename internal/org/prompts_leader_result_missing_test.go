package org

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// leaderResultMissingBulletMarker opens the bullet of the leader template's
// 運用規律 section on the watch's result_missing ALERT (plan
// 2026-11-22-org-result-missing, S3 and AC5). It is built from
// condResultMissing, so the bullet names the CONDITION value the watch sends.
var leaderResultMissingBulletMarker = "- 見張りの ALERT のうち `CONDITION: " + condResultMissing + "` のもの"

// seatSendFailedLineHead is the head of the line a seat writes on its screen
// when its message to the leader cannot be sent or checked (the typed
// protocol section of the implementer and reviewer templates), which the
// leader's bullet names too.
const seatSendFailedLineHead = "`SEND FAILED: <TYPE> <TASK_ID> "

// leaderResultMissingBullet renders the leader template with seatSendHome as
// the agmsg home and returns its 運用規律 bullet on the result_missing ALERT.
func leaderResultMissingBullet(t *testing.T) string {
	t.Helper()
	text := renderSeatWithAgmsgHome(t, "leader", "leader")
	discipline, ok := markdownSection(text, "## 運用規律")
	if !ok {
		t.Fatalf("rendered leader prompt has no 運用規律 section:\n%s", text)
	}
	item, ok := markdownItem(discipline, leaderResultMissingBulletMarker)
	if !ok {
		t.Fatalf("expected the 運用規律 section to have a bullet opening %q, got:\n%s", leaderResultMissingBulletMarker, discipline)
	}
	return item
}

// TestRenderRolePrompt_Leader_ResultMissingAlertSaysWhatTheLeaderDoes pins
// the leader's side of AC5 of plan 2026-11-22-org-result-missing: the
// 運用規律 bullet on the watch's result_missing ALERT says what the ALERT
// means (a seat that received a TASK sits waiting for input past
// stall_minutes without its RESULT), that it is not proof of a failure, so
// the seat is not stopped or spawned again, and that the leader reads the
// seat's screen and decides by what it sees: a SEND FAILED: line or a result
// the seat wrote and did not send gets a DECISION that asks the seat to send
// it again and check it, not the TASK again (a redo, and a PLAN: header for a
// per-plan implementer); a second SEND FAILED: line for the same TASK, and a
// screen it cannot read, get a BLOCKED through ralph org escalate; a seat
// still at work or waiting for a run_in_background command (a gate) is waited
// for. A SEND FAILED: line the leader sees on a screen it reads for another
// reason, the seat check after a restart included, is handled the same way
// when the TASK's reply has not come (the leader's own agmsg history), before
// a restarted leader sends a TASK again.
//
// If this test fails after a deliberate change to the bullet, put the new
// words, short and verbatim, in their place in the list; otherwise restore
// the bullet in internal/org/prompts/leader.md.
func TestRenderRolePrompt_Leader_ResultMissingAlertSaysWhatTheLeaderDoes(t *testing.T) {
	item := leaderResultMissingBullet(t)
	assertPhrasesInOrder(t, item, "the 運用規律 bullet on the result_missing ALERT", []string{
		leaderResultMissingBulletMarker,
		"(ヘッダに `SEAT` と `TASK_ID` が付きます)は、TASK を受けた座席が、入力待ちのまま `[org.watchdog].stall_minutes` を超えても、" +
			"その TASK の RESULT を送っていないことを知らせます",
		"送り損ねとは限らないので、この ALERT を理由に座席を stop も spawn し直しもしないでください",
		"`ralph org read --org-id org-a --state-dir <台帳> --seat <座席>` で座席の画面を読み、見えたもので分けます",
		seatSendFailedLineHead + "<ファイル>: <要点>` の行(座席が leader への送信に失敗したか、届いたことを確かめられなかったときに書く行)か、" +
			"座席が書いたのに送っていない RESULT・BLOCKED の文が見えたら、その座席に DECISION を送り、" +
			"そのメッセージを送り直して届いたことを確かめるよう頼みます",
		"(`TASK_ID:` の行にその TASK_ID を、本文には `SEND FAILED:` の行があればそのファイルのパスを書きます)",
		"TASK は送り直しません(作業のやり直しになり、per-plan では implementer への TASK に `PLAN:` のヘッダも要ります)",
		"頼んだあとにも同じ TASK の `SEND FAILED:` の行が出たら、もう頼まずに BLOCKED を上げます",
		"座席がまだ作業しているか、`run_in_background` で走らせたコマンド(gate など。「機能ごとの org」の節の手順 12)の終わりを待っていれば、待ちます",
		"どれとも分からなければ BLOCKED を上げます",
		"座席の名前、TASK_ID、読んだ画面のポインタを本文に書き(TASK_ID のヘッダはその TASK_ID)、" +
			"「件を上げる」の節の手順で `ralph org escalate` で上げて、返事を待ちます",
		"ほかの理由で座席の画面を読んだとき(立て直したあとの座席の確認も含みます)に `SEND FAILED:` の行が見えたときも、" +
			"その TASK の RESULT か BLOCKED がまだ届いていなければ",
		"(`bash " + seatSendScripts + "/history.sh ralph-org-a leader 50` で見ます)、同じく扱います",
		"立て直したあとの確認では、TASK を送り直す前にこの扱いをします",
	})
	if strings.Contains(item, "{{") {
		t.Errorf("the bullet on the result_missing ALERT leaves a placeholder unrendered:\n%s", item)
	}

	// The places the bullet points at say what it says they do.
	text := renderSeatWithAgmsgHome(t, "leader", "leader")
	if !strings.Contains(text, "\n## 件を上げる(`ralph org escalate`)\n") {
		t.Errorf("the bullet names the 件を上げる section, which the leader template no longer has")
	}
	step12 := directorSkillNumberedStep(leaderFeatureOrgSection(t), 12)
	if !strings.HasPrefix(step12, "12. reviewer に gate を頼む TASK を送る。") {
		t.Errorf("the bullet names 手順 12 of the 機能ごとの org section for the gate, but that step opens %q", firstLine(step12))
	}
}

// TestRenderRolePrompt_Leader_ResultMissingMatchesTheAlertAndTheSeats ties
// the leader's bullet to what it reads (AC5 of plan
// 2026-11-22-org-result-missing): the ALERT the watch sends
// (resultMissingAlertMessage) is a typed message with the CONDITION the
// bullet names and the SEAT and TASK_ID headers it reads, and its body asks
// for the screen read and names the SEND FAILED: line; and the line the
// implementer and reviewer templates have a seat write starts with the head
// the bullet names, so a leader that looks for it finds it.
func TestRenderRolePrompt_Leader_ResultMissingMatchesTheAlertAndTheSeats(t *testing.T) {
	at := time.Date(2026, 11, 22, 1, 0, 0, 0, time.UTC)
	msg := resultMissingAlertMessage("org-a", "implementer", resultMissingTask{TaskID: "t-7", At: at}, "", at, 20)
	m, err := protocol.Parse(msg)
	if err != nil {
		t.Fatalf("protocol.Parse(the result_missing ALERT) = %v; want a typed message:\n%s", err, msg)
	}
	if m.Type != protocol.TypeAlert || m.TaskID != "t-7" {
		t.Errorf("the result_missing ALERT has TYPE %q and TASK_ID %q; want ALERT and t-7:\n%s", m.Type, m.TaskID, msg)
	}
	for _, header := range []string{"CONDITION: " + condResultMissing + "\n", "SEAT: implementer\n", "TASK_ID: t-7\n"} {
		if !strings.Contains(msg, header) {
			t.Errorf("the result_missing ALERT has no header line %q, which the leader's bullet reads:\n%s", header, msg)
		}
	}
	for _, want := range []string{"`ralph org read`", "SEND FAILED:", "wait"} {
		if !strings.Contains(m.Body, want) {
			t.Errorf("the body of the result_missing ALERT does not say %q, which the leader's bullet acts on:\n%s", want, m.Body)
		}
	}

	item := leaderResultMissingBullet(t)
	if !containsPhrase(item, seatSendFailedLineHead) {
		t.Fatalf("the leader's bullet does not name the head %q of the seats' SEND FAILED: line:\n%s", seatSendFailedLineHead, item)
	}
	for _, role := range seatSendRoles {
		text := renderSeatPrompt(t, role)
		if !strings.Contains(seatTypedProtocol(t, role, text), seatSendFailedLineHead) {
			t.Errorf("the %s template's typed protocol section no longer has the line head %q that the leader's bullet names", role, seatSendFailedLineHead)
		}
	}
}

// firstLine returns the first line of s.
func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

// TestAgentMessagingRule_ResultMissingAndTheSeatsSendCheck pins AC5 of plan
// 2026-11-22-org-result-missing in both copies of the protocol rule: the
// section ## How a seat sends to the leader says how a seat sends with
// agmsg's send.sh and checks the exit code and the history (one more send,
// then the SEND FAILED: line, no send after a check that fails), and what a
// leader does with that line; and the section ## Escalation to the org inbox
// says when the watch raises result_missing (the seat's last TASK, the clock
// and its start after a QUESTION or BLOCKED, the history read with the limits
// of resultMissingHistoryLimits), when it does not, the ALERT's headers, a
// TASK out of the history's reach not read again, a rise for another TASK as
// a new ALERT and item (the plan's self-review M1 and L4), the deadman item
// without a TASK_ID, and the hold of the seat's stall with the later stall
// it brings.
//
// If this test fails after a deliberate change, put the rule's new words in
// their fragment in the rule's order, in both copies (scripts/check-sync.sh
// keeps them byte-identical); if the limits of the history read change, the
// rule's sentence on them changes with them.
func TestAgentMessagingRule_ResultMissingAndTheSeatsSendCheck(t *testing.T) {
	if want := []int{50, 200, 1000}; !slices.Equal(resultMissingHistoryLimits, want) {
		t.Fatalf("resultMissingHistoryLimits = %v; the rule says the watch reads %v lines in turn", resultMissingHistoryLimits, want)
	}
	checkSkillPlaces(t, agentMessagingRuleCopies, "result_missing", []skillPlace{
		{"the section ## How a seat sends to the leader", func(text string) string {
			return directorSkillSubsection(text, "## How a seat sends to the leader")
		}, []string{
			"A seat other than the leader sends its `RESULT`, `BLOCKED`, and `QUESTION` to the leader itself, " +
				"with the agmsg skill's `send.sh`, not with `ralph org send`",
			"`bash <agmsg scripts>/send.sh <team> <seat> leader \"$(cat <file>)\"`",
			"On a non-zero exit code the seat sends once more, and when that fails too it writes one line on its screen and stops: " +
				"`SEND FAILED: <TYPE> <TASK_ID> <file>: <the gist of the error>`.",
			"On exit code 0 the seat reads `history.sh <team> <seat> 20`",
			"When `history.sh` fails or that line is not there, it writes the `SEND FAILED:` line and stops, without sending again.",
			"does not end its turn until it has sent and checked a `RESULT`, `BLOCKED`, or `QUESTION` about that `TASK` or written that line",
			"asks the seat with a `DECISION` to send that message again and check it, does not send the `TASK` again, " +
				"and raises a `BLOCKED` when the line comes back.",
		}},
		{"the section ## Escalation to the org inbox", func(text string) string {
			return directorSkillSubsection(text, "## Escalation to the org inbox")
		}, []string{
			"`" + condResultMissing + "` is a condition of one seat other than the leader.",
			"with a TASK_ID the ledger holds whole and no `STOP` sent to it after it",
			"with a limit of 50 lines, then 200, then 1,000 while the lines it got do not reach back to the `TASK`'s time",
			"raises the condition when no line the seat sent the leader at or after the `TASK`'s time, to the second, is a `RESULT` or `REVIEW` with that TASK_ID",
			"`QUESTION` (with any TASK_ID or none) or a `BLOCKED` with that TASK_ID",
			"A reply the leader sends after a `RESULT` does not start it again.",
			"The watch does not raise the condition when the history cannot be read, does not reach back, or has a line it cannot parse",
			"`CONDITION: " + condResultMissing + "`, and `TASK_ID`",
			"a `TASK` the history did not reach back to with 1,000 lines, whose history is not read again",
			"A rise for a `TASK` other than the one of the seat's last `" + condResultMissing + "` `ALERT` is a new `ALERT` to the leader, " +
				"even while the condition is up or that `ALERT`'s item is open, and never a recurrence of that `ALERT` or of its item; " +
				"its deadman escalation is a new item too.",
			"A rise for the same `TASK` follows the rules above, with that `ALERT` and its own item only.",
			"Its item (`REASON: deadman_timeout`) carries no TASK_ID",
			"the seat's stall is not raised until `stall_minutes` have passed since the later of the `TASK`'s time and the start of the wait, " +
				"nor while the condition holds",
			"gets its stall up to `stall_minutes` later than it did before this condition.",
		}},
	})
}

// TestOrgSkill_ResultMissingIsInTheWatchRowAndTheAlertItems pins the /org
// skill's side of AC5 of plan 2026-11-22-org-result-missing in its four
// copies: the verb-table row of watch names the seat's condition
// result_missing with its TASK_ID header and the stall it holds, and the
// bullets of ## 受信箱 on the watch's ALERT items say that a seat's stall is
// held by it (no longer "as before") and what the condition is, with the
// later stall and the item without a TASK_ID.
//
// If this test fails after a deliberate change, put the new words in the
// fragments in the skill's order; otherwise restore the text in
// .claude/skills/org/SKILL.md and templates/base/.claude/skills/org/SKILL.md
// and run scripts/sync-skills.sh for the two .agents copies.
func TestOrgSkill_ResultMissingIsInTheWatchRowAndTheAlertItems(t *testing.T) {
	checkOrgSkillPlaces(t, "result_missing", []skillPlace{
		{"the verb-table row of watch", func(text string) string { return orgSkillVerbRow(text, "watch") }, []string{
			"予約の内と数える)。",
			"TASK を受けた leader 以外の座席が、入力待ちのまま `stall_minutes` を過ぎてもその TASK の RESULT を leader に送っていなければ(座席の agmsg の履歴で見る)",
			"見出しに `TASK_ID` を持つ座席の条件 `" + condResultMissing + "` の ALERT を leader へ送り、その時計が回るあいだは、その座席の stall を立てない。",
			"受信箱に書けなければ",
		}},
		{"the section ## 受信箱", func(text string) string { return directorSkillSubsection(text, "## 受信箱") }, []string{
			"leader 以外の座席の stall は、下の `" + condResultMissing + "` の項のとおり、その時計が回るあいだは立てない。",
			"- 座席の条件 `" + condResultMissing + "`: leader 以外の座席が、最後に受けた TASK の RESULT(か REVIEW)を leader に送らないまま",
			"見張りは見出しに `TASK_ID` を足した ALERT を leader に送る",
			"RESULT を送ってから入力待ちになった座席の stall も、最大 `stall_minutes` 遅れる",
			"件の本文に TASK_ID は載らない",
		}},
	})
}

// TestDirectorSkill_ResultMissingAlertItemReadsTheSeatsScreen pins the
// /director skill's side of AC5 of plan 2026-11-22-org-result-missing in its
// four copies: the ALERT item of ### 件の処理 lists result_missing among the
// conditions and says what such an item is (with no TASK_ID in its body) and
// what the director does: read the seat's screen, wait out a background
// command, tell the leader of a SEND FAILED: line (the leader asks the seat
// to send again), and send the item to the human when it keeps coming for the
// same seat.
//
// If this test fails after a deliberate change, put the new words in the
// fragments in the skill's order; otherwise restore the text in
// .claude/skills/director/SKILL.md and
// templates/base/.claude/skills/director/SKILL.md and run
// scripts/sync-skills.sh for the two .agents copies.
func TestDirectorSkill_ResultMissingAlertItemReadsTheSeatsScreen(t *testing.T) {
	checkDirectorSkillPlaces(t, "result_missing", []skillPlace{
		{"the subsection ### 件の処理", func(text string) string { return directorSkillSubsection(text, "### 件の処理") }, []string{
			"`CONDITION` が条件(stall・liveness・scope_change・usage_limited・" + condResultMissing + " のどれか)",
			"`CONDITION` が `" + condResultMissing + "` の件は、`SEAT` の座席が TASK を受けたのに、入力待ちのまま `[org.watchdog].stall_minutes` を過ぎてもその TASK の RESULT を leader に送っておらず",
			"(件の本文に TASK_ID は載らない)",
			"上のとおり座席の画面を読み、gate などのバックグラウンドのコマンドを待っていれば待ち、終わってから決める",
			"画面に `SEND FAILED:` の行(座席が leader への送信に失敗したか、届いたことを確かめられなかったときに書く行)があれば leader に知らせ(leader は座席に送り直しを頼む)",
			"同じ座席でこの件が続くなら人に送る",
			"leader に知らせるときは、下の「答える順」のとおりに",
		}},
	})
}
