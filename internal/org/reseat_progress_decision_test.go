package org

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

// The human decisions in the progress section of a restarted leader's task
// (plan docs/plans/active/2026-11-21-org-human-decision.md, slice S2: AC7).
// The first tests build the ledger's events by hand, as
// reseat_progress_test.go does; the last ones send the decisions with Send
// and restart the leader with Reseat or StartFeature, on the fixtures of
// reseat_test.go and feature_test.go.

// progressDecisionText is a human's decision as the director sends it to the
// leader: a DECISION with ORIGIN: human, longer than the sendDetailsMaxLen
// bytes a `sent` event's Details keep, so that its last line is only in the
// event's Text.
const progressDecisionText = "TYPE: DECISION\nORIGIN: human\n\n" +
	"人の決定(2026-11-21 10:00、director が人から聞いた): reviewer の座席は codex のまま続ける。\n" +
	"implementer は sonnet に替えず、opus のままこの計画の最後のスライスまで進める。\n" +
	"最後の行: gate が落ちたら直してから次のスライスへ進む。"

// progressDecisionHead is the head line of the item for n decisions.
func progressDecisionHead(n string) string {
	return "人の決定: " + n + " 件(ORIGIN: human を付けて leader に送られた DECISION の全文を、送った順に。leader はこれらの決定に従い続ける)"
}

// progressDecisionEvent is the `sent` event Send records for a real send of
// the human decision text to orgID's leader: Details with before (`raw=true `
// or ""), the mark, after (the screen path's words, `submit_unconfirmed=true
// `, or ""), and the text cut by truncateForDetails, and the whole text in
// Text.
func progressDecisionEvent(orgID, ts, before, after, text string) ManifestEvent {
	return ManifestEvent{
		TS: ts, OrgID: orgID, SeatID: LeaderIdentity, Event: EventSent,
		Details: before + sendOriginHumanDetails + after + truncateForDetails(text), Text: text,
	}
}

// progressDecisionLines is the entry of the item for a decision sent at ts
// whose text has no control character and is not cut: its line and the lines
// of the text, indented.
func progressDecisionLines(ts, text string) []string {
	lines := []string{"- " + ts + " に送った:"}
	for _, l := range strings.Split(text, "\n") {
		lines = append(lines, "  "+l)
	}
	return lines
}

// TestReseatProgress_AHumanDecisionIsListedWhole covers AC7's text: a
// decision of more than the 200 bytes Details keep is in the section whole,
// with the time it was sent, under the item 人の決定; an org whose only
// progress is the decision has the section.
func TestReseatProgress_AHumanDecisionIsListedWhole(t *testing.T) {
	if len(progressDecisionText) <= sendDetailsMaxLen {
		t.Fatalf("the decision is %d bytes, want more than %d", len(progressDecisionText), sendDetailsMaxLen)
	}
	ev := progressDecisionEvent("auth-core", "2026-11-21T01:00:00Z", "", "", progressDecisionText)
	if strings.Contains(ev.Details, "最後の行") {
		t.Fatalf("the fixture's Details hold the decision's last line, so the test would pass on Details alone: %q", ev.Details)
	}
	p := readReseatProgress([]ManifestEvent{ev}, "auth-core", nil)
	got := progressSectionLines(t, reseatProgressSection(p, reseatProgressSupplied{}))
	want := []string{
		"人の決定: 1 件(ORIGIN: human を付けて leader に送られた DECISION の全文を、送った順に。leader はこれらの決定に従い続ける)",
		"- 2026-11-21T01:00:00Z に送った:",
		"  TYPE: DECISION",
		"  ORIGIN: human",
		"  ",
		"  人の決定(2026-11-21 10:00、director が人から聞いた): reviewer の座席は codex のまま続ける。",
		"  implementer は sonnet に替えず、opus のままこの計画の最後のスライスまで進める。",
		"  最後の行: gate が落ちたら直してから次のスライスへ進む。",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("block =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if reseatProgressHumanDecisionsLabel != "人の決定" || !strings.HasPrefix(got[0], reseatProgressHumanDecisionsLabel+": ") {
		t.Errorf("the item's label is %q and its head line %q, want 人の決定, which the leader's role prompt names", reseatProgressHumanDecisionsLabel, got[0])
	}
}

// TestReseatProgress_HumanDecisionsKeepTheLedgerOrder: two decisions are
// listed in the order the ledger has them, not by their time, after every
// other part of the section; the mark is read after `raw=true ` and before
// the screen path's words, as Send writes them.
func TestReseatProgress_HumanDecisionsKeepTheLedgerOrder(t *testing.T) {
	screen := (&sendScreenPath{herdrState: "working"}).details()
	second := "TYPE: DECISION\nORIGIN: human\n\n2 つ目の決定: reviewer は claude に戻す"
	events := []ManifestEvent{
		progressApprovalEvent("auth-core", "2026-11-21T00:30:00Z", "docs/plans/active/auth.md", "0123456789ab"),
		progressDecisionEvent("auth-core", "2026-11-21T03:00:00Z", "raw=true ", "", progressDecisionText),
		progressOrgEvent("auth-core", "2026-11-21T03:30:00Z", EventReseated, "id=r1 seats="),
		// later in the ledger, with an earlier time
		progressDecisionEvent("auth-core", "2026-11-21T02:00:00Z", "", screen+"submit_unconfirmed=true ", second),
	}
	got := progressSectionLines(t, reseatProgressSection(readReseatProgress(events, "auth-core", nil), reseatProgressSupplied{}))
	want := []string{
		"計画の承認: docs/plans/active/auth.md sha256:0123456789ab (承認 2026-11-21T00:30:00Z)",
		"前の立て直し: 2026-11-21T03:30:00Z id=r1 seats=",
		progressDecisionHead("2"),
	}
	want = append(want, progressDecisionLines("2026-11-21T03:00:00Z", progressDecisionText)...)
	want = append(want, progressDecisionLines("2026-11-21T02:00:00Z", second)...)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("block =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestReseatProgress_AHumanDecisionBeforeTheLastDisbandIsNotListed covers
// AC7's life rule: a decision sent before the org's last real disband is not
// listed, one sent after it is; a dry-run disband and another org's disband
// end no life.
func TestReseatProgress_AHumanDecisionBeforeTheLastDisbandIsNotListed(t *testing.T) {
	old := "TYPE: DECISION\nORIGIN: human\n\n前の命の決定: これは載らない"
	current := "TYPE: DECISION\nORIGIN: human\n\n今の命の決定: これは載る"
	events := []ManifestEvent{
		progressDecisionEvent("auth-core", "2026-11-20T01:00:00Z", "", "", old),
		progressOrgEvent("auth-core", "2026-11-20T02:00:00Z", EventDisbanded, ""),
		progressOrgEvent("other", "2026-11-20T03:00:00Z", EventDisbanded, ""),
		{TS: "2026-11-20T04:00:00Z", OrgID: "auth-core", Event: EventDisbanded, DryRun: true},
		progressDecisionEvent("auth-core", "2026-11-21T01:00:00Z", "", "", current),
	}
	got := progressSectionLines(t, reseatProgressSection(readReseatProgress(events, "auth-core", nil), reseatProgressSupplied{}))
	want := append([]string{progressDecisionHead("1")}, progressDecisionLines("2026-11-21T01:00:00Z", current)...)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("block =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	t.Run("a life that ended after its only decision has none", func(t *testing.T) {
		p := readReseatProgress(events[:2], "auth-core", nil)
		if len(p.Decisions) != 0 || reseatProgressSection(p, reseatProgressSupplied{}) != "" {
			t.Errorf("Decisions = %+v, want none after the disband", p.Decisions)
		}
	})
}

// TestReseatProgress_OnlyARealHumanDecisionToTheLeaderIsListed covers the
// rest of AC7: a `sent` to the leader without the mark is not listed, as
// before, and neither is a dry run with the mark (with a Text, so that the dry
// run itself is what leaves it out), nor any event that lacks one of the
// mark, the Text, the leader, the org, or the `sent` name.
func TestReseatProgress_OnlyARealHumanDecisionToTheLeaderIsListed(t *testing.T) {
	const at = "2026-11-21T01:00:00Z"
	marked := progressDecisionEvent("auth-core", at, "", "", progressDecisionText)
	with := func(edit func(ev *ManifestEvent)) ManifestEvent {
		ev := marked
		edit(&ev)
		return ev
	}
	for _, tc := range []struct {
		name string
		ev   ManifestEvent
	}{
		{"a DECISION to the leader with no ORIGIN header, as Send records it",
			progressSentEvent("auth-core", LeaderIdentity, at, "", "TYPE: DECISION\n\napproved")},
		{"a DECISION whose key is Origin, as Send records it",
			progressSentEvent("auth-core", LeaderIdentity, at, "", "TYPE: DECISION\nOrigin: human\n\nkeep it")},
		{"a raw text to the leader that starts with the words of the mark",
			progressSentEvent("auth-core", LeaderIdentity, at, "raw=true ", "origin=human keep the reviewer")},
		{"a dry run with the mark and a Text", with(func(ev *ManifestEvent) { ev.DryRun = true })},
		{"a dry run with the mark, as Send records it", with(func(ev *ManifestEvent) { ev.DryRun = true; ev.Text = "" })},
		{"the mark with no Text", with(func(ev *ManifestEvent) { ev.Text = "" })},
		{"a Text with no mark", with(func(ev *ManifestEvent) { ev.Details = truncateForDetails(ev.Text) })},
		{"the mark after the screen path's words", with(func(ev *ManifestEvent) {
			ev.Details = (&sendScreenPath{herdrState: "idle"}).details() + sendOriginHumanDetails + truncateForDetails(ev.Text)
		})},
		{"to a seat that is not the leader", with(func(ev *ManifestEvent) { ev.SeatID = implementerSeatID })},
		{"another org's", with(func(ev *ManifestEvent) { ev.OrgID = "other" })},
		{"an org-level event", with(func(ev *ManifestEvent) { ev.SeatID = "" })},
		{"an event of the leader that is not sent", with(func(ev *ManifestEvent) { ev.Event = EventSpawned })},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := readReseatProgress([]ManifestEvent{tc.ev}, "auth-core", nil)
			if len(p.Decisions) != 0 {
				t.Errorf("Decisions = %+v, want none", p.Decisions)
			}
			if section := reseatProgressSection(p, reseatProgressSupplied{}); strings.Contains(section, "人の決定") {
				t.Errorf("the section lists a decision:\n%s", section)
			}
		})
	}
	t.Run("the marked event as Send records it is listed", func(t *testing.T) {
		if p := readReseatProgress([]ManifestEvent{marked}, "auth-core", nil); len(p.Decisions) != 1 || p.Decisions[0] != (reseatProgressDecision{TS: at, Text: progressDecisionText}) {
			t.Errorf("Decisions = %+v, want the decision", p.Decisions)
		}
	})
}

// TestReseatProgress_TheHumanOriginMarkIsReadWhereSendWritesIt pins
// sentDetailsHumanOrigin against the Details Send writes: the mark first, or
// right after `raw=true `, and nowhere else.
func TestReseatProgress_TheHumanOriginMarkIsReadWhereSendWritesIt(t *testing.T) {
	for _, details := range []string{
		"origin=human TYPE: DECISION",
		"raw=true origin=human TYPE: DECISION",
		"origin=human via=screen herdr_state=idle TYPE: X",
		"raw=true origin=human submit_unconfirmed=true TYPE: X",
		"origin=human ",
	} {
		if !sentDetailsHumanOrigin(details) {
			t.Errorf("sentDetailsHumanOrigin(%q) = false, want true", details)
		}
	}
	for _, details := range []string{
		"",
		"raw=true ",
		"TYPE: DECISION\nORIGIN: human\n\nbody",
		"origin=humane TYPE: DECISION",
		"origin=human",
		"via=screen herdr_state=idle origin=human TYPE: X",
		"raw=true raw=true origin=human TYPE: X",
		"submit_unconfirmed=true origin=human TYPE: X",
	} {
		if sentDetailsHumanOrigin(details) {
			t.Errorf("sentDetailsHumanOrigin(%q) = true, want false", details)
		}
	}
}

// TestReseatProgress_AHumanDecisionStaysInsideItsEntry: a decision's lines
// are indented, so a `## ` line is not a heading, its backticks lengthen the
// fence, and its control characters and line separators are escaped, with no
// line cut.
func TestReseatProgress_AHumanDecisionStaysInsideItsEntry(t *testing.T) {
	long := strings.Repeat("長", 300)
	lineSeparator := string(rune(0x2028))
	text := "TYPE: DECISION\nORIGIN: human\n\n## 見出しではない\n```````\n\x1b[31mred\x1b[0m\ttab\rcr " + lineSeparator + "sep\n- 項目ではない\n" + long
	ev := progressDecisionEvent("auth-core", "2026-11-21T01:00:00Z", "", "", text)
	section := reseatProgressSection(readReseatProgress([]ManifestEvent{ev}, "auth-core", nil), reseatProgressSupplied{})
	block := progressSectionLines(t, section) // fails on a `## ` line or an early fence
	if fence := strings.Split(section, "\n")[1]; fence != strings.Repeat("`", 8) {
		t.Errorf("fence = %q, want 8 backticks", fence)
	}
	want := []string{
		progressDecisionHead("1"),
		"- 2026-11-21T01:00:00Z に送った:",
		"  TYPE: DECISION",
		"  ORIGIN: human",
		"  ",
		"  ## 見出しではない",
		"  ```````",
		`  \x1b[31mred\x1b[0m\ttab\rcr \` + `u2028sep`,
		"  - 項目ではない",
		"  " + long,
	}
	if strings.Join(block, "\n") != strings.Join(want, "\n") {
		t.Errorf("block =\n%s\nwant\n%s", strings.Join(block, "\n"), strings.Join(want, "\n"))
	}
}

// TestReseatProgress_AHandEditedDecisionLongerThanSendRecordsIsCut: Send
// records at most sendOriginHumanMaxBytes of a decision, so a text of that
// length is copied whole, and only a longer one (a hand-edited ledger line)
// is cut, at a rune boundary, with the cut said on its line.
func TestReseatProgress_AHandEditedDecisionLongerThanSendRecordsIsCut(t *testing.T) {
	head := "TYPE: DECISION\nORIGIN: human\n\n"
	// "あ" is 3 bytes; 16384 - 30 is not a multiple of 3, so the cut falls
	// inside a rune
	if (reseatProgressDecisionMaxBytes-len(head))%3 == 0 {
		t.Fatal("the fixture's cut falls on a rune boundary; change the head")
	}
	atLimit := head + strings.Repeat("a", reseatProgressDecisionMaxBytes-len(head))
	over := head + strings.Repeat("あ", (reseatProgressDecisionMaxBytes-len(head))/3+5)

	block := progressSectionLines(t, reseatProgressSection(readReseatProgress([]ManifestEvent{
		progressDecisionEvent("auth-core", "2026-11-21T01:00:00Z", "", "", atLimit),
		progressDecisionEvent("auth-core", "2026-11-21T02:00:00Z", "", "", over),
	}, "auth-core", nil), reseatProgressSupplied{}))

	keptRunes := (reseatProgressDecisionMaxBytes - len(head)) / 3
	keptBytes := len(head) + 3*keptRunes
	want := []string{progressDecisionHead("2")}
	want = append(want, progressDecisionLines("2026-11-21T01:00:00Z", atLimit)...)
	want = append(want,
		fmt.Sprintf("- 2026-11-21T02:00:00Z に送った(台帳の text は %d バイトで、send が記録する 16384 バイトを超えるので、先頭の %d バイトまで):",
			len(over), keptBytes),
		"  TYPE: DECISION", "  ORIGIN: human", "  ", "  "+strings.Repeat("あ", keptRunes))
	if strings.Join(block, "\n") != strings.Join(want, "\n") {
		t.Errorf("block =\n%.600s\n...\nwant\n%.600s\n...", strings.Join(block, "\n"), strings.Join(want, "\n"))
	}
	for _, l := range block {
		if !utf8.ValidString(l) {
			t.Errorf("a line of the block is not valid UTF-8: %q", l)
		}
	}
}

// sendToLeader sends text to auth-core's leader with Send (a dry run when
// dryRun) and fails unless the send passed.
func (st *featureStart) sendToLeader(t *testing.T, text string, dryRun bool) {
	t.Helper()
	if res := st.o.Send(SendParams{OrgID: "auth-core", To: LeaderIdentity, Text: text, DryRun: dryRun}); res.Err != nil {
		t.Fatalf("send %q to the leader (dry run %v): %v", text, dryRun, res.Err)
	}
}

// humanDecisionTimes returns the times of the real `sent` events to
// auth-core's leader that keep a Text, in the ledger's order.
func (st *featureStart) humanDecisionTimes(t *testing.T) []string {
	t.Helper()
	var times []string
	for _, ev := range mustReadEvents(t, st.o) {
		if ev.OrgID == "auth-core" && ev.SeatID == LeaderIdentity && ev.Event == EventSent && !ev.DryRun && ev.Text != "" {
			times = append(times, ev.TS)
		}
	}
	return times
}

// TestReseatProgress_AReseatGivesTheRestartedLeaderTheHumanDecisionsWhole
// covers AC7 through `ralph org reseat`: a decision of more than 200 bytes
// sent with Send before the reseat is in the restarted leader's task whole,
// with its time, while a DECISION to the leader with no ORIGIN header and a
// dry run of the decision are not.
func TestReseatProgress_AReseatGivesTheRestartedLeaderTheHumanDecisionsWhole(t *testing.T) {
	st := newReseatOrg(t)
	st.h.closeRemoves = true
	progressStartRepo(t, st.worktree("auth-core"))
	fresh := st.freshLeaderTask(t)

	st.sendToLeader(t, "TYPE: DECISION\n\nno ORIGIN header, so not a human's decision", false)
	st.sendToLeader(t, progressDecisionText, true)
	st.sendToLeader(t, progressDecisionText, false)
	times := st.humanDecisionTimes(t)
	if len(times) != 1 {
		t.Fatalf("the ledger has %d sent events with a Text, want 1", len(times))
	}

	res := st.mustReseat(t, st.reseatParams(LeaderIdentity))
	block := progressSectionLines(t, progressSectionOf(t, st.leaderTaskOf(t), fresh))
	want := []string{
		"計画の承認: " + testPlanRel + " sha256:",
		"今回の立て直し: reseat=" + res.Plan.ID + "、leader のあとに起動する予定の座席: なし(leader だけ)",
		progressDecisionHead("1"),
	}
	want = append(want, progressDecisionLines(times[0], progressDecisionText)...)
	if len(block) != len(want) || !strings.HasPrefix(block[0], want[0]) || strings.Join(block[1:], "\n") != strings.Join(want[1:], "\n") {
		t.Errorf("block =\n%s\nwant\n%s", strings.Join(block, "\n"), strings.Join(want, "\n"))
	}
}

// TestReseatProgress_AStartThatRestartsTheLeaderGivesItTheHumanDecisions
// covers AC7 through a `start --plan` that starts a stopped leader again: two
// decisions sent with Send, one with --raw, are in the task in the order they
// were sent, and they alone make the section.
func TestReseatProgress_AStartThatRestartsTheLeaderGivesItTheHumanDecisions(t *testing.T) {
	st := newFeatureStart(t)
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	fresh := st.freshLeaderTask(t)
	progressStartRepo(t, st.worktree("auth-core"))

	second := "TYPE: DECISION\nORIGIN: human\n\n2 つ目の決定: reviewer は claude に戻す"
	st.sendToLeader(t, progressDecisionText, false)
	if res := st.o.Send(SendParams{OrgID: "auth-core", To: LeaderIdentity, Text: second, Raw: true}); res.Err != nil {
		t.Fatalf("send the second decision with --raw: %v", res.Err)
	}
	times := st.humanDecisionTimes(t)
	if len(times) != 2 {
		t.Fatalf("the ledger has %d sent events with a Text, want 2", len(times))
	}

	st.leaderRestarted(t, st.params("auth-core"))
	block := progressSectionLines(t, progressSectionOf(t, st.leaderTaskOf(t), fresh))
	want := []string{progressDecisionHead("2")}
	want = append(want, progressDecisionLines(times[0], progressDecisionText)...)
	want = append(want, progressDecisionLines(times[1], second)...)
	if strings.Join(block, "\n") != strings.Join(want, "\n") {
		t.Errorf("block =\n%s\nwant\n%s", strings.Join(block, "\n"), strings.Join(want, "\n"))
	}
}

// TestReseatProgress_AStartAfterADisbandGivesNoDecisionOfTheOldLife covers
// AC7's life rule through StartFeature: a decision sent before the org was
// disbanded is not in the task of the leader the next start starts.
func TestReseatProgress_AStartAfterADisbandGivesNoDecisionOfTheOldLife(t *testing.T) {
	st := newFeatureStart(t)
	st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	progressStartRepo(t, st.worktree("auth-core"))
	st.sendToLeader(t, progressDecisionText, false)
	if len(st.humanDecisionTimes(t)) != 1 {
		t.Fatal("the decision was not recorded with its Text")
	}
	if d := st.o.Disband(DisbandParams{OrgID: "auth-core"}); len(d.Errs) != 0 || !d.Disbanded {
		t.Fatalf("disband auth-core: %+v", d)
	}

	st.mustStart(t, st.params("auth-core"), SpawnOutcomeSpawned)
	if got, want := st.leaderTaskOf(t), st.freshLeaderTask(t); got != want {
		t.Errorf("the task after the disband =\n%s\nwant featureLeaderTask's, with no decision of the old life\n%s", got, want)
	}
}
