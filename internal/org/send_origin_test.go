package org

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// sendOriginMsg is a human's decision with the ORIGIN header, as the director
// sends it to the leader.
const sendOriginMsg = "TYPE: DECISION\nORIGIN: human\n\nkeep the reviewer on codex (the human, 2026-11-21 10:00)"

// sendOriginLeader spawns the leader seat of org-a in testOrg's org.
func sendOriginLeader(t *testing.T, o *Org) {
	t.Helper()
	p := mustSpawnParams("org-a", LeaderIdentity)
	p.Role = LeaderIdentity
	if r := o.Spawn(p); r.Outcome != SpawnOutcomeSpawned {
		t.Fatalf("spawn the leader: %+v", r)
	}
}

// TestSendOriginRule_Text pins the rule every refusal of the ORIGIN check
// states, so a change to it is a change to this test.
func TestSendOriginRule_Text(t *testing.T) {
	want := `ORIGIN is allowed only as one "ORIGIN: human" header line on a TYPE: DECISION sent to the leader (--to leader)`
	if sendOriginRule != want {
		t.Fatalf("sendOriginRule = %q\nwant %q", sendOriginRule, want)
	}
}

// TestCheckSendOrigin_Table covers checkSendOrigin case by case: the header
// is read as protocol.Parse reads headers (the key exactly ORIGIN once the
// spaces around it are trimmed, the value trimmed with its case kept, only
// the header block), and every ORIGIN header but one `ORIGIN: human` on a
// DECISION to the leader is refused with what the check found and the rule.
func TestCheckSendOrigin_Table(t *testing.T) {
	big := func(n int) string {
		head := "TYPE: DECISION\nORIGIN: human\n\n"
		return head + strings.Repeat("a", n-len(head))
	}
	tests := []struct {
		name      string
		text      string
		to        string
		wantHuman bool
		wantFound string // "" for a pass; the sendOriginError's found otherwise
		wantErr   string // a refusal that is not a sendOriginError
	}{
		{name: "a DECISION to the leader with ORIGIN: human", text: sendOriginMsg, to: LeaderIdentity, wantHuman: true},
		{name: "spaces around the key and the value", text: "TYPE: DECISION\n  ORIGIN  :   human  \n\nbody", to: LeaderIdentity, wantHuman: true},
		{name: "ORIGIN after other header lines", text: "TYPE: DECISION\nFROM: director\nORIGIN: human\n\nbody", to: LeaderIdentity, wantHuman: true},
		{name: "header only, no body", text: "TYPE: DECISION\nORIGIN: human", to: LeaderIdentity, wantHuman: true},
		{name: "a DECISION to the leader with no ORIGIN", text: "TYPE: DECISION\n\napproved", to: LeaderIdentity},
		{name: "a TASK to a seat with no ORIGIN", text: "TYPE: TASK\nTASK_ID: t-1\n\ndo it", to: "implementer"},
		{name: "ORIGIN: human only in the body", text: "TYPE: TASK\nTASK_ID: t-1\n\nORIGIN: human", to: "implementer"},
		{name: "the key Origin is not ORIGIN", text: "TYPE: TASK\nTASK_ID: t-1\nOrigin: human\n\ndo it", to: "implementer"},
		{name: "the key origin is not ORIGIN", text: "TYPE: DECISION\norigin: director\n\nbody", to: "implementer"},
		{name: "a raw text whose first line is not TYPE", text: "ORIGIN: human\nTYPE: DECISION\n\nbody", to: "implementer"},
		{name: "a raw free-form text", text: "not a valid protocol message", to: "implementer"},
		{name: "ORIGIN: human on a TASK", text: "TYPE: TASK\nTASK_ID: t-1\nORIGIN: human\n\ndo it", to: LeaderIdentity,
			wantFound: `the message is TYPE: "TASK" with an ORIGIN header`},
		{name: "ORIGIN: human on a RESULT", text: "TYPE: RESULT\nTASK_ID: t-1\nORIGIN: human\n\ndone", to: LeaderIdentity,
			wantFound: `the message is TYPE: "RESULT" with an ORIGIN header`},
		{name: "ORIGIN: human on a lower-case decision (raw)", text: "TYPE: decision\nORIGIN: human\n\nbody", to: LeaderIdentity,
			wantFound: `the message is TYPE: "decision" with an ORIGIN header`},
		{name: "ORIGIN: human on an empty TYPE (raw)", text: "TYPE:\nORIGIN: human\n\nbody", to: LeaderIdentity,
			wantFound: `the message is TYPE: "" with an ORIGIN header`},
		{name: "ORIGIN: human on a TYPE with a control character (raw)", text: "TYPE: DECISION\x1b[2K\nORIGIN: human\n\nbody", to: LeaderIdentity,
			wantFound: `the message is TYPE: "DECISION\x1b[2K" with an ORIGIN header`},
		{name: "ORIGIN: Human", text: "TYPE: DECISION\nORIGIN: Human\n\nbody", to: LeaderIdentity,
			wantFound: `the message has ORIGIN: "Human", not "human"`},
		{name: "ORIGIN: HUMAN", text: "TYPE: DECISION\nORIGIN: HUMAN\n\nbody", to: LeaderIdentity,
			wantFound: `the message has ORIGIN: "HUMAN", not "human"`},
		{name: "ORIGIN: director", text: "TYPE: DECISION\nORIGIN: director\n\nbody", to: LeaderIdentity,
			wantFound: `the message has ORIGIN: "director", not "human"`},
		{name: "an empty ORIGIN", text: "TYPE: DECISION\nORIGIN:\n\nbody", to: LeaderIdentity,
			wantFound: `the message has ORIGIN: "", not "human"`},
		{name: "to a seat that is not the leader", text: sendOriginMsg, to: "implementer",
			wantFound: `the message with ORIGIN: human is for seat "implementer", not the leader`},
		{name: "to Leader, not leader", text: sendOriginMsg, to: "Leader",
			wantFound: `the message with ORIGIN: human is for seat "Leader", not the leader`},
		{name: "two ORIGIN: human lines", text: "TYPE: DECISION\nORIGIN: human\nORIGIN: human\n\nbody", to: LeaderIdentity,
			wantFound: "the message has 2 ORIGIN header lines"},
		{name: "two ORIGIN lines with other values", text: "TYPE: DECISION\nORIGIN: director\nORIGIN: human\n\nbody", to: LeaderIdentity,
			wantFound: "the message has 2 ORIGIN header lines"},
		{name: "at the size limit", text: big(sendOriginHumanMaxBytes), to: LeaderIdentity, wantHuman: true},
		{name: "one byte over the size limit", text: big(sendOriginHumanMaxBytes + 1), to: LeaderIdentity,
			wantErr: "org: send: the message with ORIGIN: human is 16385 bytes; the ledger keeps such a message whole, so it may be at most 16384 bytes; nothing was sent"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			human, err := checkSendOrigin(tc.text, tc.to)
			switch {
			case tc.wantFound != "":
				var oe sendOriginError
				if !errors.As(err, &oe) || oe.found != tc.wantFound {
					t.Fatalf("checkSendOrigin = %v, %v; want a refusal that found %q", human, err, tc.wantFound)
				}
				want := "org: send: " + tc.wantFound + "; " + sendOriginRule + "; nothing was sent"
				if err.Error() != want {
					t.Fatalf("error = %q\nwant %q", err, want)
				}
			case tc.wantErr != "":
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("checkSendOrigin = %v, %v; want the error %q", human, err, tc.wantErr)
				}
			default:
				if err != nil {
					t.Fatalf("checkSendOrigin = %v; want it to pass", err)
				}
			}
			if human != tc.wantHuman {
				t.Fatalf("checkSendOrigin human = %v, want %v", human, tc.wantHuman)
			}
		})
	}
}

// TestCheckSendOrigin_EveryTypeButDecisionRefused: ORIGIN: human to the
// leader is refused on every TYPE of the protocol but DECISION.
func TestCheckSendOrigin_EveryTypeButDecisionRefused(t *testing.T) {
	types := []string{
		protocol.TypeTask, protocol.TypeResult, protocol.TypeQuestion, protocol.TypeReview,
		protocol.TypeDecision, protocol.TypeBlocked, protocol.TypeContract, protocol.TypeHeartbeat,
		protocol.TypeStop, protocol.TypeHello, protocol.TypeAlert,
	}
	for _, typ := range types {
		text := "TYPE: " + typ + "\nTASK_ID: t-1\nORIGIN: human\n\nbody"
		human, err := checkSendOrigin(text, LeaderIdentity)
		if typ == protocol.TypeDecision {
			if err != nil || !human {
				t.Errorf("TYPE %s: checkSendOrigin = %v, %v; want it to pass", typ, human, err)
			}
			continue
		}
		var oe sendOriginError
		if !errors.As(err, &oe) || human || !strings.Contains(err.Error(), fmt.Sprintf("TYPE: %q with an ORIGIN header", typ)) {
			t.Errorf("TYPE %s: checkSendOrigin = %v, %v; want the refusal of the TYPE", typ, human, err)
		}
	}
}

// TestOrgSendOrigin_HumanDecisionToLeader_MarksDetailsAndKeepsText covers
// AC1 on herdr's path: a DECISION with ORIGIN: human to the leader is typed
// and submitted, and its `sent` event's Details start with `origin=human `
// (after `raw=true ` for --raw, before `submit_unconfirmed=true `), and the
// event keeps the whole text in Text, a text longer than Details' cut
// included.
func TestOrgSendOrigin_HumanDecisionToLeader_MarksDetailsAndKeepsText(t *testing.T) {
	long := "TYPE: DECISION\nORIGIN: human\n\n" + strings.Repeat("keep the reviewer on codex; ", 20)
	if len(long) <= sendDetailsMaxLen {
		t.Fatalf("test text is %d bytes; want more than %d", len(long), sendDetailsMaxLen)
	}
	tests := []struct {
		name        string
		text        string
		raw         bool
		unconfirmed bool
		wantDetails string
	}{
		{name: "typed", text: sendOriginMsg, wantDetails: "origin=human " + sendOriginMsg},
		{name: "raw", text: sendOriginMsg, raw: true, wantDetails: "raw=true origin=human " + sendOriginMsg},
		{name: "raw, submit unconfirmed", text: sendOriginMsg, raw: true, unconfirmed: true,
			wantDetails: "raw=true origin=human submit_unconfirmed=true " + sendOriginMsg},
		{name: "longer than the cut of Details", text: long, wantDetails: "origin=human " + truncateForDetails(long)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o, h, _ := testOrg(t)
			sendOriginLeader(t, o)
			if tc.unconfirmed {
				h.agentWaitErrs = []error{nil, nil, errors.New("stub: confirm wait failed")}
			}
			before := len(mustReadEvents(t, o))

			res := o.Send(SendParams{OrgID: "org-a", To: LeaderIdentity, Text: tc.text, Raw: tc.raw})
			if res.Err != nil || res.Progress != SendProgressEnterPressed {
				t.Fatalf("Send = %+v; want the decision sent", res)
			}
			if !slices.Contains(h.calls, "pane_send_text") {
				t.Fatalf("herdr calls = %v; want the text typed", h.calls)
			}
			events := mustReadEvents(t, o)[before:]
			if len(events) != 1 || events[0].Event != EventSent || events[0].SeatID != LeaderIdentity {
				t.Fatalf("new events = %+v; want one sent event to the leader", events)
			}
			ev := events[0]
			if ev.Details != tc.wantDetails {
				t.Fatalf("sent Details = %q\nwant %q", ev.Details, tc.wantDetails)
			}
			if ev.Text != tc.text {
				t.Fatalf("sent Text = %q\nwant the whole text %q", ev.Text, tc.text)
			}
			if ev.TextSHA256 != sentTextSHA256(tc.text) {
				t.Fatalf("sent TextSHA256 = %q, want %q", ev.TextSHA256, sentTextSHA256(tc.text))
			}
		})
	}
}

// TestOrgSendOrigin_DryRun_MarksDetailsWithoutText covers AC1's dry run: the
// dry-run `sent` event's Details start with `origin=human ` (after
// `raw=true `), it keeps no Text, and no driver is called.
func TestOrgSendOrigin_DryRun_MarksDetailsWithoutText(t *testing.T) {
	for _, tc := range []struct {
		raw         bool
		wantDetails string
	}{
		{raw: false, wantDetails: "origin=human " + sendOriginMsg},
		{raw: true, wantDetails: "raw=true origin=human " + sendOriginMsg},
	} {
		o, h, a := testOrg(t)
		res := o.Send(SendParams{OrgID: "org-a", To: LeaderIdentity, Text: sendOriginMsg, Raw: tc.raw, DryRun: true})
		if res.Err != nil {
			t.Fatalf("raw %v: dry-run Send = %v; want it to pass", tc.raw, res.Err)
		}
		if len(h.calls) != 0 || len(a.calls) != 0 {
			t.Fatalf("raw %v: driver calls herdr=%v agmsg=%v; want none for a dry run", tc.raw, h.calls, a.calls)
		}
		events := mustReadEvents(t, o)
		if len(events) != 1 || events[0].Event != EventSent || !events[0].DryRun {
			t.Fatalf("raw %v: events = %+v; want one dry-run sent event", tc.raw, events)
		}
		if events[0].Details != tc.wantDetails {
			t.Fatalf("raw %v: dry-run Details = %q\nwant %q", tc.raw, events[0].Details, tc.wantDetails)
		}
		if events[0].Text != "" || events[0].TextSHA256 != "" {
			t.Fatalf("raw %v: dry-run Text = %q, TextSHA256 = %q; want neither", tc.raw, events[0].Text, events[0].TextSHA256)
		}
	}
}

// TestOrgSendOrigin_Refused_NothingTypedNothingRecorded covers AC2 through
// Send: an ORIGIN header on another TYPE, with another value, to another
// seat, or given twice, is refused, a dry run and a --raw text included,
// with the rule in the error, no event, and no driver call.
func TestOrgSendOrigin_Refused_NothingTypedNothingRecorded(t *testing.T) {
	tests := []struct {
		name string
		to   string
		text string
		raw  bool
	}{
		{name: "on a TASK", to: LeaderIdentity, text: "TYPE: TASK\nTASK_ID: t-1\nORIGIN: human\n\ndo it"},
		{name: "on a raw RESULT", to: LeaderIdentity, text: "TYPE: RESULT\nTASK_ID: t-1\nORIGIN: human\n\ndone", raw: true},
		{name: "another value", to: LeaderIdentity, text: "TYPE: DECISION\nORIGIN: director\n\nbody"},
		{name: "to another seat", to: "seat-1", text: sendOriginMsg},
		{name: "raw, to another seat", to: "seat-1", text: sendOriginMsg, raw: true},
		{name: "two ORIGIN lines", to: LeaderIdentity, text: "TYPE: DECISION\nORIGIN: human\nORIGIN: human\n\nbody"},
	}
	for _, tc := range tests {
		for _, dry := range []bool{false, true} {
			name := tc.name
			if dry {
				name += ", dry run"
			}
			t.Run(name, func(t *testing.T) {
				o, h, a := testOrg(t)
				sendOriginLeader(t, o)
				if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
					t.Fatalf("spawn seat-1: %+v", r)
				}
				before := len(mustReadEvents(t, o))
				herdrBefore, agmsgBefore := len(h.calls), len(a.calls)

				res := o.Send(SendParams{OrgID: "org-a", To: tc.to, Text: tc.text, Raw: tc.raw, DryRun: dry})
				var oe sendOriginError
				if !errors.As(res.Err, &oe) || !strings.Contains(res.Err.Error(), sendOriginRule) {
					t.Fatalf("Send = %+v; want the ORIGIN refusal with its rule", res)
				}
				if res.Progress != SendProgressNothingSent || res.PaneID != "" {
					t.Fatalf("Send = %+v; want nothing sent", res)
				}
				if got := len(mustReadEvents(t, o)); got != before {
					t.Fatalf("events %d -> %d; want none recorded", before, got)
				}
				if len(h.calls) != herdrBefore || len(a.calls) != agmsgBefore {
					t.Fatalf("driver calls herdr %v agmsg %v after the refusal; want none", h.calls[herdrBefore:], a.calls[agmsgBefore:])
				}
			})
		}
	}
}

// TestOrgSendOrigin_NoOriginHeader_RecordedAsBefore covers AC3 for the
// record: a message with no ORIGIN header, and one whose key is Origin (not
// ORIGIN), is sent and recorded as before, with no `origin=` mark and no
// `text` key in the manifest line.
func TestOrgSendOrigin_NoOriginHeader_RecordedAsBefore(t *testing.T) {
	for _, text := range []string{
		"TYPE: DECISION\n\napproved",
		"TYPE: DECISION\nOrigin: human\n\nkeep the reviewer on codex",
	} {
		o, _, _ := testOrg(t)
		sendOriginLeader(t, o)
		before := len(mustReadEvents(t, o))
		if res := o.Send(SendParams{OrgID: "org-a", To: LeaderIdentity, Text: text}); res.Err != nil {
			t.Fatalf("Send(%q) = %v; want it sent", text, res.Err)
		}
		events := mustReadEvents(t, o)[before:]
		if len(events) != 1 || events[0].Details != text || events[0].Text != "" {
			t.Fatalf("Send(%q) events = %+v; want one sent event with Details the text and no Text", text, events)
		}
		data, err := os.ReadFile(o.Manifest.Path())
		if err != nil {
			t.Fatalf("read manifest: %v", err)
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if last := lines[len(lines)-1]; strings.Contains(last, `"text":`) || !strings.Contains(last, `"text_sha256":`) {
			t.Fatalf("Send(%q) manifest line = %s; want text_sha256 and no text key", text, last)
		}
	}
}

// TestOrgSendOrigin_SendResultHumanOrigin covers SendResult.HumanOrigin
// (verify V8 of plan 2026-11-21-org-human-decision): it is true on every
// return after checkSendOrigin passed a human's decision (sent, a dry run, the
// three failures after typing began that the CLI ends its note for, and a
// later error such as a leader not found), and false for a DECISION with no
// ORIGIN header, sent or failed, and for every refusal (the ORIGIN check, the
// paste marker check, and protocol validation, which come before it).
func TestOrgSendOrigin_SendResultHumanOrigin(t *testing.T) {
	tests := []struct {
		name         string
		to           string
		text         string
		dryRun       bool
		noLeader     bool
		setup        func(o *Org, h *fakeHerdr)
		raw          bool
		timeoutMS    int
		wantErr      bool
		wantErrPart  string // for a refusal, the words of the check that refused it
		wantProgress SendProgress
		wantHuman    bool
	}{
		{name: "a human's decision sent", text: sendOriginMsg, wantProgress: SendProgressEnterPressed, wantHuman: true},
		{name: "a human's decision, dry run", text: sendOriginMsg, dryRun: true, wantHuman: true},
		{name: "a human's decision, send-text fails", text: sendOriginMsg, wantErr: true, wantProgress: SendProgressTextUnacknowledged, wantHuman: true,
			setup: func(_ *Org, h *fakeHerdr) { h.paneSendTextErr = errors.New("stub: PaneSendText failed") }},
		{name: "a human's decision, the pause before Enter runs out", text: sendOriginMsg, timeoutMS: 100, wantErr: true, wantProgress: SendProgressTextTyped, wantHuman: true,
			setup: func(o *Org, h *fakeHerdr) {
				o.SendEnterDelay = 50 * time.Millisecond
				h.paneSendTextDelay = 80 * time.Millisecond
			}},
		{name: "a human's decision, send-keys fails", text: sendOriginMsg, wantErr: true, wantProgress: SendProgressEnterUnacknowledged, wantHuman: true,
			setup: func(_ *Org, h *fakeHerdr) { h.paneSendKeysErr = errors.New("stub: PaneSendKeys failed") }},
		{name: "a human's decision, no leader seat", text: sendOriginMsg, noLeader: true, wantErr: true, wantProgress: SendProgressNothingSent, wantHuman: true},
		{name: "a DECISION with no ORIGIN sent", text: "TYPE: DECISION\n\napproved", wantProgress: SendProgressEnterPressed},
		{name: "a DECISION with no ORIGIN, send-text fails", text: "TYPE: DECISION\n\napproved", wantErr: true, wantProgress: SendProgressTextUnacknowledged,
			setup: func(_ *Org, h *fakeHerdr) { h.paneSendTextErr = errors.New("stub: PaneSendText failed") }},
		{name: "refused: ORIGIN: human to another seat", to: "seat-1", text: sendOriginMsg,
			wantErr: true, wantErrPart: sendOriginRule, wantProgress: SendProgressNothingSent},
		{name: "refused: ORIGIN: director", text: "TYPE: DECISION\nORIGIN: director\n\nbody",
			wantErr: true, wantErrPart: sendOriginRule, wantProgress: SendProgressNothingSent},
		{name: "refused: a paste marker in a raw human's decision", text: "TYPE: DECISION\nORIGIN: human\n\nkeep \x1b[201~ it", raw: true,
			wantErr: true, wantErrPart: errSendPasteMarker.Error(), wantProgress: SendProgressNothingSent},
		{name: "refused: protocol validation of a human's decision", text: "TYPE: DECISION\nORIGIN: human\n\n" + strings.Repeat("a", protocol.DefaultMaxBodyChars+1),
			wantErr: true, wantErrPart: "message rejected by protocol validation", wantProgress: SendProgressNothingSent},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o, h, _ := testOrg(t)
			if !tc.noLeader {
				sendOriginLeader(t, o)
			}
			if r := o.Spawn(mustSpawnParams("org-a", "seat-1")); r.Outcome != SpawnOutcomeSpawned {
				t.Fatalf("spawn seat-1: %+v", r)
			}
			if tc.setup != nil {
				tc.setup(o, h)
			}
			to := tc.to
			if to == "" {
				to = LeaderIdentity
			}

			res := o.Send(SendParams{OrgID: "org-a", To: to, Text: tc.text, Raw: tc.raw, DryRun: tc.dryRun, TimeoutMS: tc.timeoutMS})
			if (res.Err != nil) != tc.wantErr {
				t.Fatalf("Send = %+v; want an error %v", res, tc.wantErr)
			}
			if tc.wantErrPart != "" && !strings.Contains(res.Err.Error(), tc.wantErrPart) {
				t.Fatalf("Send Err = %v; want the refusal that says %q", res.Err, tc.wantErrPart)
			}
			if res.Progress != tc.wantProgress {
				t.Fatalf("Send Progress = %v, want %v (Err %v)", res.Progress, tc.wantProgress, res.Err)
			}
			if res.HumanOrigin != tc.wantHuman {
				t.Fatalf("Send HumanOrigin = %v, want %v (Err %v)", res.HumanOrigin, tc.wantHuman, res.Err)
			}
		})
	}
}

// TestOrgSendOrigin_ScreenPath_DetailsOrder: on the screen path the mark
// comes after `raw=true ` and before `via=screen herdr_state=<state> ` and
// `submit_unconfirmed=true `, and the event keeps the whole text.
func TestOrgSendOrigin_ScreenPath_DetailsOrder(t *testing.T) {
	o, rh := resumeTestOrg(t)
	h := &sendFallbackHerdr{fakeResumeHerdr: rh}
	o.Herdr = h
	o.SendScreenReadGap = 20 * time.Millisecond
	o.SendScreenReserve = 300 * time.Millisecond
	o.SendSubmitConfirmTimeout = 60 * time.Millisecond
	name := seedResumeSeat(t, o, "org-a", LeaderIdentity, LeaderIdentity, "claude", "pane-1")
	rh.setStatus(name, "working")
	idle := readResumeScreen(t, "claude-idle-after-reply.txt")
	h.setScreens("pane-1", idle, idle, idle, sendTypedScreen(t, "claude", idle, sendOriginMsg))
	before := len(mustReadEvents(t, o))

	res := o.Send(SendParams{OrgID: "org-a", To: LeaderIdentity, Text: sendOriginMsg, Raw: true, TimeoutMS: sendFallbackTimeoutMS})
	if res.Err != nil || res.Progress != SendProgressEnterPressed {
		t.Fatalf("Send = %+v; want Enter pressed on the screen path", res)
	}
	want := "raw=true origin=human via=screen herdr_state=working submit_unconfirmed=true " + sendOriginMsg
	if got := lastSentDetails(t, o, before); got != want {
		t.Fatalf("sent Details = %q\nwant %q", got, want)
	}
	if ev := mustReadEvents(t, o)[before]; ev.Text != sendOriginMsg {
		t.Fatalf("sent Text = %q, want %q", ev.Text, sendOriginMsg)
	}
}
