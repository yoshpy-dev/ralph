package org

import (
	"strings"
	"testing"
)

// sendTypedNotSubmittedMark is the end of the error text of a `ralph org send`
// that typed the message into the seat's input box and did not press Enter
// (the check before Enter did not find the text, or the wait before Enter ran
// out; Send in verbs.go). The role prompts and skills name it so a leader or a
// director can tell this failure from one that typed nothing.
const sendTypedNotSubmittedMark = "text typed but not submitted"

// markdownParagraphWith returns the paragraph of text (the lines between blank
// lines) that holds marker, and how many paragraphs hold it.
func markdownParagraphWith(text, marker string) (paragraph string, count int) {
	for _, p := range strings.Split(text, "\n\n") {
		if strings.Contains(p, marker) {
			paragraph = p
			count++
		}
	}
	return paragraph, count
}

// TestRenderRolePrompt_Leader_SendTextTypedNotSubmittedEscalates pins AC5 of
// plan 2026-11-13-send-input-box-marker (D3) for the leader prompt: when a
// send ends with the error text `(text typed but not submitted)`, the typed
// protocol section, in one paragraph, has the leader not send again (a
// second send is typed after the leftover), read the seat's pane with
// `ralph org read`, raise a BLOCKED with `ralph org escalate` by the section
// of escalating, and send no key to the pane. The paragraph also says that
// the advice of ralph's note to clear or submit the text there is for a
// person at the terminal, and that the leader follows the paragraph instead.
func TestRenderRolePrompt_Leader_SendTextTypedNotSubmittedEscalates(t *testing.T) {
	text := renderSeatPrompt(t, LeaderIdentity)
	section, found := markdownSection(text, "## typed protocol")
	if !found {
		t.Fatalf("expected the leader template to contain a '## typed protocol' section, got:\n%s", text)
	}
	paragraph, count := markdownParagraphWith(section, sendTypedNotSubmittedMark)
	if count != 1 {
		t.Fatalf("expected exactly one paragraph of the typed protocol section to name %q, got %d in section:\n%s", sendTypedNotSubmittedMark, count, section)
	}
	for _, want := range []string{
		"`ralph org send` が終了コード 1 で終わり、エラーの文が `(text typed but not submitted)` で終わるときは",
		"文が座席の入力欄に打たれたまま、Enter が押されていません",
		"同じ send を打ち直さないでください",
		"2 本目は残った文のあとに打たれ、Enter で 1 つのメッセージとして送られます",
		"座席の pane にキーも送りません",
		"stderr の note にある `ralph org read` で座席の画面を読み",
		"「件を上げる」の節の手順で `ralph org escalate` を使って BLOCKED を上げてください",
		"TASK_ID は、その send が送った TASK の TASK_ID です(TASK 以外の送信は org_id の `org-a`)",
		"画面をそのまま貼らないでください",
		"note の `clear or submit it there`(残った文を消すか送る)は端末の前にいる人への助言で、leader は代わりにこの段落に従います",
	} {
		if !containsPhrase(paragraph, want) {
			t.Errorf("expected the typed protocol paragraph on an unsubmitted send to say %q, got paragraph:\n%s", want, paragraph)
		}
	}
	if strings.Contains(text, "{{") {
		t.Errorf("expected no placeholder left in the rendered leader prompt, got:\n%s", text)
	}
}

// TestRenderDirectorPrompt_SendTextTypedNotSubmittedReadsTheLeadersScreen
// pins AC5 of plan 2026-11-13-send-input-box-marker (D3) for the director
// prompt, in the form for herdr and the form for [org.director].launch_cmd:
// one mission bullet says that a send to the leader that ends with
// `(text typed but not submitted)` is not sent again, that the director
// reads the leader's screen first as step 2 of the skill's 「答える順」 says,
// and that Enter is pressed only when the whole DECISION sits in the input
// box with no other screen, while in every other case it does not press
// Enter, does not resolve the item, and tells the human. The bullet names no
// pane and no `ralph org` command, since the terminal form of the prompt names
// neither and TestRenderDirectorPrompt_GivesEveryRalphOrgCommandTheConfig
// lists the commands the director runs.
func TestRenderDirectorPrompt_SendTextTypedNotSubmittedReadsTheLeadersScreen(t *testing.T) {
	for _, launch := range []string{DirectorLaunchHerdr, DirectorLaunchCmd} {
		t.Run(launch, func(t *testing.T) {
			vars := testDirectorPromptVars()
			vars.Launch = launch
			text := mustRenderDirectorPrompt(t, vars)
			mission, found := markdownSection(text, "## ミッション")
			if !found {
				t.Fatalf("rendered director prompt has no ミッション section:\n%s", text)
			}
			if n := strings.Count(text, sendTypedNotSubmittedMark); n != 1 {
				t.Fatalf("expected the director prompt to name %q once, got %d:\n%s", sendTypedNotSubmittedMark, n, text)
			}
			bullet, found := markdownItem(mission, "- leader への send が終了コード 1 で")
			if !found {
				t.Fatalf("expected the ミッション section to have the bullet on an unsubmitted send to the leader, got section:\n%s", mission)
			}
			for _, want := range []string{
				"leader への send が終了コード 1 で、エラーの文が `(text typed but not submitted)` で終わったときは",
				"同じ send を打ち直さない",
				"(2 本目は入力欄に残った文のあとに打たれる)",
				"skill の「答える順」の 2 のとおり、先に leader の画面を読む",
				"Enter を押すのは、DECISION の全文が入力欄に残り、ほかの画面が出ていないときに限る",
				"それ以外は Enter を押さず、件を resolve せず、人に伝える",
			} {
				if !containsPhrase(bullet, want) {
					t.Errorf("expected the bullet on an unsubmitted send to say %q, got bullet:\n%s", want, bullet)
				}
			}
			if strings.Contains(bullet, "pane") || strings.Contains(bullet, "ralph org") {
				t.Errorf("expected the bullet to name no pane and no ralph org command, got bullet:\n%s", bullet)
			}
		})
	}
}
