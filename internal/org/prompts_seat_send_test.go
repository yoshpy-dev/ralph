package org

import (
	"os"
	"strings"
	"testing"
)

// seatSendRoles are the roles whose templates send their messages to the
// leader with agmsg's send.sh (the leader itself sends with ralph org send).
var seatSendRoles = []string{"implementer", "reviewer"}

// seatSendHome is the agmsg home the tests of this file render the seat
// templates with: a path with a space, so a command that does not use the
// quoted {{AGMSG_SCRIPTS}} word does not match.
const seatSendHome = "/opt/agmsg home"

// seatSendScripts is agmsgScriptsWord(seatSendHome).
const seatSendScripts = "'/opt/agmsg home/scripts'"

// seatSendCommands are the send.sh command and the history.sh command of the
// typed protocol section of a seat template rendered for team and seatID
// with scripts (the {{AGMSG_SCRIPTS}} value), each a whole inline code span
// on one line, so a seat can copy it as it is.
func seatSendCommands(scripts, team, seatID string) (send, history string) {
	return "`bash " + scripts + "/send.sh " + team + " " + seatID + " leader \"$(cat <ファイル>)\"`",
		"`bash " + scripts + "/history.sh " + team + " " + seatID + " 20`"
}

// renderSeatWithAgmsgHome renders role's template as the seat seatID, with
// the shared test vars and seatSendHome for RolePromptVars.AgmsgHome. The
// seat id differs from the role, so a template that put the role where the
// seat id belongs does not match.
func renderSeatWithAgmsgHome(t *testing.T, role, seatID string) string {
	t.Helper()
	vars := testRolePromptVars()
	vars.Role, vars.SeatID, vars.AgmsgHome = role, seatID, seatSendHome
	text, ok, err := RenderRolePrompt(role, vars)
	if err != nil || !ok {
		t.Fatalf("RenderRolePrompt(%s) = ok %v, err %v; want the built-in template", role, ok, err)
	}
	return text
}

// seatTypedProtocol is the typed protocol section of the rendered text of
// role's template.
func seatTypedProtocol(t *testing.T, role, text string) string {
	t.Helper()
	section, found := markdownSection(text, "## typed protocol")
	if !found {
		t.Fatalf("expected the %s template to contain a '## typed protocol' section, got:\n%s", role, text)
	}
	return section
}

// TestRenderRolePrompt_SeatSend_NamesTheSendCommand pins AC2 (a) of plan
// 2026-11-22-org-result-missing for the implementer and reviewer templates:
// the typed protocol section names the command a seat sends its messages to
// the leader with, agmsg's send.sh from the scripts directory of the Org's
// agmsg home (quoted), with the team, the seat's own id, leader, and the
// message file passed as "$(cat <file>)", and has RESULT, BLOCKED, and
// QUESTION written to a file under .harness/state/ first, a QUESTION about a
// TASK carrying that TASK's TASK_ID. No placeholder is left.
func TestRenderRolePrompt_SeatSend_NamesTheSendCommand(t *testing.T) {
	if got := agmsgScriptsWord(seatSendHome); got != seatSendScripts {
		t.Fatalf("agmsgScriptsWord(%q) = %q, want %q", seatSendHome, got, seatSendScripts)
	}
	for _, role := range seatSendRoles {
		t.Run(role, func(t *testing.T) {
			seatID := role + "-7"
			text := renderSeatWithAgmsgHome(t, role, seatID)
			section := seatTypedProtocol(t, role, text)
			send, _ := seatSendCommands(seatSendScripts, "ralph-org-a", seatID)
			if !strings.Contains(section, "\n"+send+"\n") {
				t.Errorf("expected the %s typed protocol section to show the send command %s on a line of its own, got section:\n%s", role, send, section)
			}
			for _, want := range []string{
				"RESULT、BLOCKED、QUESTION のメッセージは、cwd の `.harness/state/` の下の",
				"送るコマンドの本文の引数には `\"$(cat <ファイル>)\"` を渡してください",
				"TASK について送る QUESTION にも、その TASK の `TASK_ID:` の行を付けます",
				"leader へは、agmsg の skill の `send.sh` を次の形で打って送ります",
			} {
				if !containsPhrase(section, want) {
					t.Errorf("expected the %s typed protocol section to say %q, got section:\n%s", role, want, section)
				}
			}
			if i := strings.Index(text, "{{"); i >= 0 {
				t.Errorf("expected no placeholder left in the rendered %s template, got %q", role, text[i:min(len(text), i+40)])
			}
		})
	}
}

// TestRenderRolePrompt_SeatSend_ChecksTheExitCodeAndRetriesOnce pins AC2
// (b), (c), and (d): after a RESULT, BLOCKED, or QUESTION the seat looks at
// the send command's exit code; when it is not 0 the seat reads the error and
// sends once more, and when the second send fails too it does not end
// silently but writes one SEND FAILED: line, with the type, the TASK_ID, the
// message file, and the error's gist, which says when stderr was empty (the
// plan's S0 saw agmsg's send.sh fail with exit 1 and no output), and stops.
// The check is part of sending, so the other sections' "return a RESULT"
// steps keep it, and the rule is written once per template.
func TestRenderRolePrompt_SeatSend_ChecksTheExitCodeAndRetriesOnce(t *testing.T) {
	for _, role := range seatSendRoles {
		t.Run(role, func(t *testing.T) {
			text := renderSeatWithAgmsgHome(t, role, role+"-7")
			section := seatTypedProtocol(t, role, text)
			if !containsPhrase(section, "この確かめまでが送信です。ほかの節の「RESULT を返す」「BLOCKED を返す」「QUESTION で聞く」も、この確かめを含みます") {
				t.Errorf("expected the %s typed protocol section to make the check part of every send, got section:\n%s", role, section)
			}
			const marker = "1. 送信のコマンドの終了コードを見る"
			item, ok := markdownItem(section, marker)
			if !ok {
				t.Fatalf("expected the %s typed protocol section to have an item %q, got section:\n%s", role, marker, section)
			}
			for _, want := range []string{
				"0 でなければ、標準エラーを読んでエラーの要点をつかみ(直せる誤りなら直してから)、もう 1 回だけ送る",
				"2 回目も 0 でなければ、黙って終わらずに、画面に次の 1 行を書いて止まる",
				"`SEND FAILED: <TYPE> <TASK_ID> <本文のファイルのパス>: <エラーの要点>`",
				"標準エラーが空なら、要点は `stderr なし、exit <終了コード>` と書く",
			} {
				if !containsPhrase(item, want) {
					t.Errorf("expected the %s item %q to say %q, got item:\n%s", role, marker, want, item)
				}
			}
			if n := strings.Count(text, "SEND FAILED: <TYPE>"); n != 1 {
				t.Errorf("expected the %s template to give the SEND FAILED: line once, in its typed protocol section, got %d", role, n)
			}
		})
	}
}

// TestRenderRolePrompt_SeatSend_TheMessageFileStartsWithTheTypeLine pins the
// seat's side of the plan's self-review L5 and R2-L4: the message file's
// first line is the TYPE: line, with no blank line or code fence before it,
// and its second line the TASK_ID: line, as in the examples, so the body the
// seat sends starts where protocol.Parse, which the watch's result_missing
// reads the seat's lines with, looks for TYPE:, and its TASK_ID is a header
// line however the rest of the headers read.
func TestRenderRolePrompt_SeatSend_TheMessageFileStartsWithTheTypeLine(t *testing.T) {
	for _, role := range seatSendRoles {
		t.Run(role, func(t *testing.T) {
			text := renderSeatWithAgmsgHome(t, role, role+"-7")
			section := seatTypedProtocol(t, role, text)
			for _, want := range []string{
				"ファイルの 1 行目は `TYPE:` の行にします(その前に空行やコードの囲みの行を置かない)",
				"`TASK_ID:` の行は、例のとおり 2 行目に置きます",
			} {
				if !containsPhrase(section, want) {
					t.Errorf("expected the %s typed protocol section to say %q, got section:\n%s", role, want, section)
				}
			}
		})
	}
}

// TestRenderRolePrompt_SeatSend_ChecksTheHistoryAfterExitZero pins AC2 (e):
// when the send exits 0 the seat reads agmsg's history with history.sh for
// the team and its own id, looks only at the lines it sent to the leader
// ("<seat> → leader:", since the history of a named agent holds the lines it
// received too), and checks that in the last one the text right after
// "leader: " starts with the TYPE line it sent and, after history.sh's `\n`,
// its TASK_ID line (the plan's self-review L5 and R2-L4: the watch reads a
// body's TYPE from its first line that is not blank and its TASK_ID only
// among the header lines, so it reads both from the lines this check looked
// at). A history.sh that fails, or no such line, does not count as sent: the
// seat writes the SEND FAILED: line with that reason, stops, and sends no
// more.
func TestRenderRolePrompt_SeatSend_ChecksTheHistoryAfterExitZero(t *testing.T) {
	for _, role := range seatSendRoles {
		t.Run(role, func(t *testing.T) {
			seatID := role + "-7"
			text := renderSeatWithAgmsgHome(t, role, seatID)
			section := seatTypedProtocol(t, role, text)
			const marker = "2. 終了コードが 0 なら"
			item, ok := markdownItem(section, marker)
			if !ok {
				t.Fatalf("expected the %s typed protocol section to have an item %q, got section:\n%s", role, marker, section)
			}
			_, history := seatSendCommands(seatSendScripts, "ralph-org-a", seatID)
			if !strings.Contains(item, "\n   "+history+"\n") {
				t.Errorf("expected the %s item %q to show the history command %s on a line of its own, got item:\n%s", role, marker, history, item)
			}
			for _, want := range []string{
				"終了コードが 0 なら(1 回目でも 2 回目でも)、次のコマンドで履歴を読む",
				"履歴にはこの座席が受けたメッセージの行も入るので、`" + seatID + " → leader:` の行だけを見て、そのうち最後の行で、`leader: ` の直後が送った `TYPE: <TYPE>\\nTASK_ID: <TASK_ID>` で始まることを確かめる",
				"`history.sh` が 0 以外で終わったときと、その行がないときは、送れたと数えない",
				"要点を `送信は exit 0、履歴を読めない` か `送信は exit 0、履歴にない` にして `SEND FAILED:` の行を書き、止まる",
				"ここでは送り直さない(3 回目も送らない)",
			} {
				if !containsPhrase(item, want) {
					t.Errorf("expected the %s item %q to say %q, got item:\n%s", role, marker, want, item)
				}
			}
		})
	}
}

// TestRenderRolePrompt_SeatSend_DoesNotEndTheTurnWithoutAReply pins AC2 (f):
// after a TASK, or a leader's answer to its QUESTION, the seat does not end
// its turn until it has sent and checked a RESULT, BLOCKED, or QUESTION for
// that TASK, or written the SEND FAILED: line (the issue's seat wrote its
// RESULT on the screen and ended its turn without sending it). Waiting for a
// command run with run_in_background, which the reviewer's gate and
// cross-review TASKs do, is the one exception.
func TestRenderRolePrompt_SeatSend_DoesNotEndTheTurnWithoutAReply(t *testing.T) {
	for _, role := range seatSendRoles {
		t.Run(role, func(t *testing.T) {
			text := renderSeatWithAgmsgHome(t, role, role+"-7")
			section := seatTypedProtocol(t, role, text)
			const marker = "3. leader から TASK を受けたとき"
			item, ok := markdownItem(section, marker)
			if !ok {
				t.Fatalf("expected the %s typed protocol section to have an item %q, got section:\n%s", role, marker, section)
			}
			for _, want := range []string{
				"leader から TASK を受けたとき、また QUESTION への答えを受けたときは、その TASK の RESULT、BLOCKED、QUESTION のどれかを送って確かめ終えるか、`SEND FAILED:` の行を書くまで、手番を終えない",
				"(`run_in_background` で走らせたコマンドの終わりの知らせを待つあいだは除く)",
				"RESULT の文を画面に書いただけでは、leader には届かない",
			} {
				if !containsPhrase(item, want) {
					t.Errorf("expected the %s item %q to say %q, got item:\n%s", role, marker, want, item)
				}
			}
		})
	}
}

// TestRenderRolePrompt_Reviewer_SeatSendMessageFileIsAllowed pins the
// reviewer's side of AC2: the reviewer writes only what its mission lists,
// so the mission lists the message files of the typed protocol section, and
// the gate TASK, which writes no other worktree file, allows them
// (.harness/state/ is ignored by git, so the gate's clean-tree check does not
// see them). TestRenderRolePrompt_Reviewer_XreviewTaskIsNotAReviewTask pins
// the scope rule that names them among its exceptions.
func TestRenderRolePrompt_Reviewer_SeatSendMessageFileIsAllowed(t *testing.T) {
	text := renderSeatWithAgmsgHome(t, "reviewer", "reviewer-7")
	mission, found := markdownSection(text, "## ミッション")
	if !found {
		t.Fatalf("expected the reviewer template to contain a '## ミッション' section, got:\n%s", text)
	}
	missionIntro, _, _ := strings.Cut(mission, "\n1. ")
	if want := "leader に送るメッセージを書く `.harness/state/msg-org-a-reviewer-7-` で始まるファイル(「typed protocol」の節)"; !containsPhrase(missionIntro, want) {
		t.Errorf("expected the reviewer mission's opening to list %q, got:\n%s", want, missionIntro)
	}
	if want := "`msg-org-a-reviewer-7-` で始まるファイル(例: `.harness/state/msg-org-a-reviewer-7-<TASK_ID>.md`)に書き"; !containsPhrase(seatTypedProtocol(t, "reviewer", text), want) {
		t.Errorf("expected the reviewer typed protocol section to say %q", want)
	}
	gate, _ := markdownSection(text, "## gate の TASK")
	item, ok := markdownItem(gate, "- gate の TASK では")
	if want := "leader に送るメッセージのファイルは、git が無視する `.harness/state/` の下なので、gate の TASK でも書いてよい(「typed protocol」の節)"; !ok || !containsPhrase(item, want) {
		t.Errorf("expected the gate TASK item on writing no worktree file to say %q, got item:\n%s", want, item)
	}
}

// TestOrgSpawn_SeatRoles_AgmsgHomeQuotedIntoPromptFile pins AC2 (a) at the
// spawn: an implementer or reviewer seat's role prompt file names the Org's
// agmsg home's scripts directory, single-quoted, in its send.sh and
// history.sh commands, with the seat's own id (Spawn passes Org.AgmsgHome to
// every role's template, not only the leader's).
func TestOrgSpawn_SeatRoles_AgmsgHomeQuotedIntoPromptFile(t *testing.T) {
	for _, role := range seatSendRoles {
		t.Run(role, func(t *testing.T) {
			o, h, _ := testOrg(t)
			o.AgmsgHome = seatSendHome
			p := mustSpawnParams("org-a", "seat-1")
			p.Role = role
			if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("spawn failed: %+v", r)
			}
			if len(h.agentStartArgs) != 1 {
				t.Fatalf("expected exactly 1 AgentStart call, got %d", len(h.agentStartArgs))
			}
			promptArg := h.agentStartArgs[0][len(h.agentStartArgs[0])-1]
			data, err := os.ReadFile(strings.TrimPrefix(promptArg, "役割指示を読み込んで従ってください: "))
			if err != nil {
				t.Fatalf("read the %s seat's prompt file: %v", role, err)
			}
			prompt := string(data)
			send, history := seatSendCommands(seatSendScripts, agmsgTeam(p.OrgID), "seat-1")
			for _, want := range []string{send, history, "`seat-1 → leader:`"} {
				if !strings.Contains(prompt, want) {
					t.Errorf("expected the %s seat's prompt file to contain %s, got:\n%s", role, want, prompt)
				}
			}
			if strings.Contains(prompt, "{{") {
				t.Errorf("expected no placeholder left in the %s seat's prompt file, got:\n%s", role, prompt)
			}
		})
	}
}
