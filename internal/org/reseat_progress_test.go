package org

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

// The progress section of a restarted leader's task (plan
// docs/plans/archive/2026-11-15-reseat-leader-progress.md, slice S1: AC1, AC3,
// AC4, AC5, and the empty case of AC2). The functions under test are pure:
// the events, the inbox items, and the commit lines are built by hand here.

const (
	progressHeadA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	progressHeadB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	progressBase  = "cccccccccccccccccccccccccccccccccccccccc"
)

// progressApprovalEvent is the plan_approved of orgID for plan and digest.
func progressApprovalEvent(orgID, ts, plan, digest string) ManifestEvent {
	a := planApproval{Plan: plan, Digest: digest, Content: strings.Repeat("d", 64), InboxID: "e1", By: "USER", Cwd: "/work"}
	return ManifestEvent{TS: ts, OrgID: orgID, Event: EventPlanApproved, Details: a.details()}
}

// progressGateEvent is the gate event of orgID that record is.
func progressGateEvent(orgID, ts string, record gateRecord) ManifestEvent {
	return ManifestEvent{TS: ts, OrgID: orgID, Event: record.event(), Details: record.details()}
}

func progressGateRecord(head, reason, via, log string) gateRecord {
	return gateRecord{
		Reason: reason, Head: head, Base: progressBase, BaseRef: "refs/heads/main",
		Branch: "feat/auth-core", By: "USER", Via: via, Log: log,
	}
}

func progressOrgEvent(orgID, ts, event, details string) ManifestEvent {
	return ManifestEvent{TS: ts, OrgID: orgID, Event: event, Details: details}
}

func progressInboxItem(id, orgID, state, escalatedAt, body string) InboxItem {
	return InboxItem{ID: id, OrgID: orgID, Type: "QUESTION", TaskID: "auth-core", Body: body, State: state, EscalatedAt: escalatedAt}
}

// progressSectionLines returns the lines of the fenced block of section, and
// fails when the section is not the lead, a fence, the block, the same fence.
func progressSectionLines(t *testing.T, section string) []string {
	t.Helper()
	all := strings.Split(section, "\n")
	if len(all) < 4 || all[0] != reseatProgressLead {
		t.Fatalf("the section does not start with the lead line:\n%s", section)
	}
	fence := all[1]
	if len(fence) < 3 || strings.Trim(fence, "`") != "" {
		t.Fatalf("the second line is not a backtick fence: %q", fence)
	}
	if all[len(all)-1] != fence {
		t.Fatalf("the section does not end with its fence %q:\n%s", fence, section)
	}
	block := all[2 : len(all)-1]
	for _, l := range block {
		if l == fence || strings.HasPrefix(l, fence) {
			t.Fatalf("a line of the block closes or opens the fence early: %q", l)
		}
		if strings.HasPrefix(l, "## ") {
			t.Fatalf("a line of the block starts with a heading: %q", l)
		}
	}
	return block
}

func TestReseatProgress_ASectionHoldsEachPartAlone(t *testing.T) {
	approval := progressApprovalEvent("auth-core", "2026-11-15T01:00:00Z", "docs/plans/active/auth.md", "0123456789ab")
	gatePass := progressGateEvent("auth-core", "2026-11-15T02:00:00Z", progressGateRecord(progressHeadA, "", "", "/state/gate.log"))
	gateFail := progressGateEvent("auth-core", "2026-11-15T02:00:00Z", progressGateRecord(progressHeadA, GateReasonTestFailed, gateViaRegateCI, ""))
	reseated := progressOrgEvent("auth-core", "2026-11-15T03:00:00Z", EventReseated, "id=r1 seats=implementer:started,reviewer:started")
	item := progressInboxItem("e3", "auth-core", InboxStateOpen, "2026-11-15T01:30:00Z", "TYPE: QUESTION\nTASK_ID: auth-core\n\nどちらの案にしますか\n詳しくは下")

	tests := []struct {
		name     string
		events   []ManifestEvent
		items    []InboxItem
		supplied reseatProgressSupplied
		want     []string
	}{
		{
			name:   "the plan approval",
			events: []ManifestEvent{approval},
			want:   []string{"計画の承認: docs/plans/active/auth.md sha256:0123456789ab (承認 2026-11-15T01:00:00Z)"},
		},
		{
			name:   "a passed gate with its log",
			events: []ManifestEvent{gatePass},
			want: []string{"最後の gate: pass(gate_passed)head " + progressHeadA + " base " + progressBase +
				" branch feat/auth-core(2026-11-15T02:00:00Z) log /state/gate.log"},
		},
		{
			name:   "a failed regate_ci gate with no log",
			events: []ManifestEvent{gateFail},
			want: []string{"最後の gate: fail(gate_ci_failed)head " + progressHeadA + " base " + progressBase +
				" branch feat/auth-core via regate_ci(2026-11-15T02:00:00Z)"},
		},
		{
			name:  "an open inbox item, its first body line after the header block",
			items: []InboxItem{item},
			want: []string{
				"片付いていない受信箱の件: 1 件(古い順に 1 件)",
				"- e3 QUESTION auth-core open: どちらの案にしますか",
			},
		},
		{
			name: "an acked item with no task id and a body that is not a typed message",
			items: []InboxItem{{ID: "e4", OrgID: "auth-core", Type: "ALERT", Body: "\n\n  stalled seat  \nnext", State: InboxStateAcked,
				EscalatedAt: "2026-11-15T01:30:00Z"}},
			want: []string{
				"片付いていない受信箱の件: 1 件(古い順に 1 件)",
				"- e4 ALERT - acked: stalled seat",
			},
		},
		{
			name:     "the commits of the branch",
			supplied: reseatProgressSupplied{Commits: []string{"abc1234 feat: add the reader", "def5678 test: pin it"}, CommitTotal: 2},
			want: []string{
				"branch の commit: 2 件(新しい順に 2 件)",
				"- abc1234 feat: add the reader",
				"- def5678 test: pin it",
			},
		},
		{
			name:     "the inbox could not be read, next to a reseat",
			supplied: reseatProgressSupplied{InboxReadError: "org: inbox: read /state/inbox.jsonl: permission denied", ReseatID: "r2"},
			want: []string{
				"受信箱: 読めなかった(org: inbox: read /state/inbox.jsonl: permission denied)",
				"今回の立て直し: reseat=r2、leader のあとに起動する予定の座席: なし(leader だけ)",
			},
		},
		{
			name:     "this reseat with seats after the leader",
			supplied: reseatProgressSupplied{ReseatID: "r2", ReseatSeats: []string{"implementer", "reviewer"}},
			want:     []string{"今回の立て直し: reseat=r2、leader のあとに起動する予定の座席: implementer, reviewer"},
		},
		{
			name:     "this reseat of the leader alone",
			supplied: reseatProgressSupplied{ReseatID: "r2"},
			want:     []string{"今回の立て直し: reseat=r2、leader のあとに起動する予定の座席: なし(leader だけ)"},
		},
		{
			name:   "the previous reseat",
			events: []ManifestEvent{reseated},
			want:   []string{"前の立て直し: 2026-11-15T03:00:00Z id=r1 seats=implementer:started,reviewer:started"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := readReseatProgress(tt.events, "auth-core", tt.items)
			section := reseatProgressSection(p, tt.supplied)
			got := progressSectionLines(t, section)
			if strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("block =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

func TestReseatProgress_AnOrgWithNoProgressHasNoSection(t *testing.T) {
	other := []ManifestEvent{
		progressApprovalEvent("other", "2026-11-15T01:00:00Z", "docs/plans/active/x.md", "0123456789ab"),
		progressGateEvent("other", "2026-11-15T02:00:00Z", progressGateRecord(progressHeadA, "", "", "")),
		progressOrgEvent("other", "2026-11-15T03:00:00Z", EventReseated, "id=r1 seats="),
	}
	tests := []struct {
		name   string
		events []ManifestEvent
		items  []InboxItem
	}{
		{name: "no events and no items"},
		{name: "only another org's events and items", events: other,
			items: []InboxItem{progressInboxItem("e1", "other", InboxStateOpen, "2026-11-15T01:00:00Z", "TYPE: QUESTION\n\nq")}},
		{name: "an org's resolved item and a seat's event and a dry run",
			events: []ManifestEvent{
				{TS: "2026-11-15T01:00:00Z", OrgID: "auth-core", SeatID: "leader", Event: EventPlanApproved,
					Details: progressApprovalEvent("auth-core", "", "docs/plans/active/x.md", "0123456789ab").Details},
				{TS: "2026-11-15T01:00:00Z", OrgID: "auth-core", Event: EventReseated, Details: "id=r1 seats=", DryRun: true},
				// a dry-run TASK, a typed message that is not a TASK, and another org's TASK
				{TS: "2026-11-15T01:00:00Z", OrgID: "auth-core", SeatID: "implementer", Event: EventSent, DryRun: true, Details: progressTaskText("dry")},
				progressSentEvent("auth-core", "reviewer", "2026-11-15T01:00:00Z", "", "TYPE: DECISION\n\nok"),
				progressSentEvent("other", "implementer", "2026-11-15T01:00:00Z", "", progressTaskText("t")),
			},
			items: []InboxItem{
				progressInboxItem("e1", "auth-core", InboxStateResolved, "2026-11-15T01:00:00Z", "TYPE: QUESTION\n\nq"),
				progressResultItem("e2", "other", InboxStateResolved, "2026-11-15T01:00:00Z", "other", progressHeadA, "another org's RESULT"),
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := readReseatProgress(tt.events, "auth-core", tt.items)
			if got := reseatProgressSection(p, reseatProgressSupplied{}); got != "" {
				t.Errorf("the section = %q, want none", got)
			}
		})
	}
	t.Run("blank commit lines and a total with no lines are no commits", func(t *testing.T) {
		got := reseatProgressSection(reseatProgress{}, reseatProgressSupplied{Commits: []string{"", "  "}, CommitTotal: 3})
		if got != "" {
			t.Errorf("the section = %q, want none", got)
		}
	})
	// A read that failed is not progress (plan D6): the start of an org with
	// none keeps its task even when git or the inbox fails.
	for name, supplied := range map[string]reseatProgressSupplied{
		"a commit read failure alone":                 {CommitReadError: "git log: signal: killed"},
		"an inbox read failure alone":                 {InboxReadError: "org: inbox: read /state/inbox.jsonl: permission denied"},
		"a commit and an inbox read failure together": {CommitReadError: "git log: signal: killed", InboxReadError: "org: inbox: unreadable"},
	} {
		t.Run(name+" gives no section", func(t *testing.T) {
			if got := reseatProgressSection(reseatProgress{}, supplied); got != "" {
				t.Errorf("the section = %q, want none", got)
			}
		})
	}
}

func TestReseatProgress_ASectionHoldsAllPartsInOrder(t *testing.T) {
	events := []ManifestEvent{
		progressApprovalEvent("auth-core", "2026-11-15T01:00:00Z", "docs/plans/active/auth.md", "0123456789ab"),
		progressGateEvent("auth-core", "2026-11-15T02:00:00Z", progressGateRecord(progressHeadA, "", gateViaRegate, "/state/gate.log")),
		progressOrgEvent("auth-core", "2026-11-15T03:00:00Z", EventReseated, "id=r1 seats=implementer:started changed=implementer:codex/gpt-6"),
		progressSentEvent("auth-core", "implementer", "2026-11-15T01:20:00Z", "", progressTaskText("slice-1")),
	}
	items := []InboxItem{
		progressInboxItem("e3", "auth-core", InboxStateOpen, "2026-11-15T01:30:00Z", "TYPE: QUESTION\nPLAN: docs/plans/active/auth.md sha256:0123456789ab\n\nq1"),
		progressResultItem("e4", "auth-core", InboxStateResolved, "2026-11-15T02:30:00Z", "slice-1", progressHeadA, "r1"),
	}
	supplied := reseatProgressSupplied{
		Commits: []string{"abc1234 feat: one"}, CommitTotal: 1, ReseatID: "r2", ReseatSeats: []string{"implementer"},
	}
	got := reseatProgressSection(readReseatProgress(events, "auth-core", items), supplied)
	want := strings.Join([]string{
		reseatProgressLead,
		"```",
		"計画の承認: docs/plans/active/auth.md sha256:0123456789ab (承認 2026-11-15T01:00:00Z)",
		"最後の gate: pass(gate_passed)head " + progressHeadA + " base " + progressBase +
			" branch feat/auth-core via regate(2026-11-15T02:00:00Z) log /state/gate.log",
		"座席ごとの最後の TASK: 1 席(台帳の sent の記録から)",
		"- implementer: slice-1(2026-11-15T01:20:00Z)",
		"片付いていない受信箱の件: 1 件(古い順に 1 件)",
		"- e3 QUESTION auth-core open: q1 / PLAN docs/plans/active/auth.md sha256:0123456789ab",
		"上げた RESULT: 1 件(新しい 1 件を古い順に)",
		"- e4 resolved slice-1 HEAD " + progressHeadA + ": r1",
		"branch の commit: 1 件(新しい順に 1 件)",
		"- abc1234 feat: one",
		"今回の立て直し: reseat=r2、leader のあとに起動する予定の座席: implementer",
		"前の立て直し: 2026-11-15T03:00:00Z id=r1 seats=implementer:started changed=implementer:codex/gpt-6",
		"```",
	}, "\n")
	if got != want {
		t.Errorf("section =\n%s\nwant\n%s", got, want)
	}
}

func TestReseatProgress_OnlyTheCurrentLifeCounts(t *testing.T) {
	events := []ManifestEvent{
		// the first life of auth-core
		progressApprovalEvent("auth-core", "2026-11-14T01:00:00Z", "docs/plans/active/old.md", "000000000000"),
		progressGateEvent("auth-core", "2026-11-14T02:00:00Z", progressGateRecord(progressHeadA, "", "", "")),
		progressOrgEvent("auth-core", "2026-11-14T03:00:00Z", EventReseated, "id=old seats="),
		progressOrgEvent("auth-core", "2026-11-14T04:00:00Z", EventDisbanded, ""),
		// a disband of another org and a dry-run disband of this one do not end its life
		progressOrgEvent("other", "2026-11-15T00:30:00Z", EventDisbanded, ""),
		{TS: "2026-11-15T00:40:00Z", OrgID: "auth-core", Event: EventDisbanded, DryRun: true},
		// the second life
		progressApprovalEvent("auth-core", "2026-11-15T01:00:00Z", "docs/plans/active/new.md", "111111111111"),
		progressApprovalEvent("auth-core", "2026-11-15T01:10:00Z", "docs/plans/active/newer.md", "222222222222"),
		progressGateEvent("auth-core", "2026-11-15T02:00:00Z", progressGateRecord(progressHeadA, GateReasonStaticFailed, "", "")),
		progressGateEvent("auth-core", "2026-11-15T02:10:00Z", progressGateRecord(progressHeadB, "", "", "")),
		progressOrgEvent("auth-core", "2026-11-15T03:00:00Z", EventReseated, "id=one seats="),
		progressOrgEvent("auth-core", "2026-11-15T03:10:00Z", EventReseated, "id=two seats="),
		// things of the second life that are not the org's real org-level events
		progressApprovalEvent("other", "2026-11-15T04:00:00Z", "docs/plans/active/other.md", "333333333333"),
		{TS: "2026-11-15T04:00:00Z", OrgID: "auth-core", SeatID: "implementer", Event: EventReseated, Details: "id=seat seats="},
	}
	p := readReseatProgress(events, "auth-core", nil)
	if p.Approval == nil || p.Approval.Plan != "docs/plans/active/newer.md" || p.Approval.Digest != "222222222222" ||
		p.Approval.TS != "2026-11-15T01:10:00Z" {
		t.Errorf("Approval = %+v, want the last approval of the second life", p.Approval)
	}
	if p.Gate == nil || !p.Gate.Pass || p.Gate.Head != progressHeadB || p.Gate.TS != "2026-11-15T02:10:00Z" {
		t.Errorf("Gate = %+v, want the passed gate at head %s", p.Gate, progressHeadB)
	}
	if p.Previous == nil || p.Previous.Details != "id=two seats=" || p.Previous.TS != "2026-11-15T03:10:00Z" {
		t.Errorf("Previous = %+v, want the second life's last reseat", p.Previous)
	}
	// the approval is the one currentPlanApprovals ends with
	approvals := currentPlanApprovals(events, "auth-core")
	last := approvals[len(approvals)-1]
	if p.Approval.Plan != last.Plan || p.Approval.Digest != last.Digest {
		t.Errorf("Approval = %+v, but currentPlanApprovals ends with %+v", p.Approval, last)
	}

	t.Run("an org with no disband reads all its events", func(t *testing.T) {
		p := readReseatProgress(events[:3], "auth-core", nil)
		if p.Approval == nil || p.Approval.Plan != "docs/plans/active/old.md" || p.Gate == nil || p.Previous == nil {
			t.Errorf("progress = %+v %+v %+v, want all three", p.Approval, p.Gate, p.Previous)
		}
	})
	t.Run("a life that ended and has no new event has no progress", func(t *testing.T) {
		p := readReseatProgress(events[:4], "auth-core", nil)
		if p.Approval != nil || p.Gate != nil || p.Previous != nil {
			t.Errorf("progress = %+v %+v %+v, want none", p.Approval, p.Gate, p.Previous)
		}
	})
}

func TestReseatProgress_AnUnreadableGateRecordFallsBackToThePreviousOne(t *testing.T) {
	good := progressGateEvent("auth-core", "2026-11-15T02:00:00Z", progressGateRecord(progressHeadA, "", "", "/state/a.log"))
	tests := []struct {
		name string
		bad  ManifestEvent
	}{
		{"details that do not parse", progressOrgEvent("auth-core", "2026-11-15T03:00:00Z", EventGateFailed, "not a record")},
		{"a head that is not a commit", progressOrgEvent("auth-core", "2026-11-15T03:00:00Z", EventGatePassed,
			"head=xyz base="+progressBase+" base_ref=refs/heads/main branch=feat/auth-core")},
		{"an event name that disagrees with the record's reason", progressOrgEvent("auth-core", "2026-11-15T03:00:00Z", EventGatePassed,
			progressGateRecord(progressHeadB, GateReasonTestFailed, "", "").details())},
		{"a ci record under a gate_failed name", progressOrgEvent("auth-core", "2026-11-15T03:00:00Z", EventGateFailed,
			progressGateRecord(progressHeadB, GateReasonCIFailed, gateViaRegateCI, "").details())},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := readReseatProgress([]ManifestEvent{good, tt.bad}, "auth-core", nil)
			if p.Gate == nil || p.Gate.Head != progressHeadA || p.Gate.TS != "2026-11-15T02:00:00Z" || p.Gate.Log != "/state/a.log" {
				t.Errorf("Gate = %+v, want the readable record before it", p.Gate)
			}
		})
	}
	t.Run("with no readable record before it there is no gate", func(t *testing.T) {
		p := readReseatProgress([]ManifestEvent{progressOrgEvent("auth-core", "2026-11-15T03:00:00Z", EventGateFailed, "x")}, "auth-core", nil)
		if p.Gate != nil {
			t.Errorf("Gate = %+v, want none", p.Gate)
		}
	})
}

func TestReseatProgress_InboxItemsOfTheCurrentLifeThatAreNotSettled(t *testing.T) {
	// The disband is at 01:00Z. +09:00 times are written so that comparing the
	// strings gives the opposite of comparing the times.
	disband := progressOrgEvent("auth-core", "2026-11-15T01:00:00Z", EventDisbanded, "")
	items := []InboxItem{
		progressInboxItem("e1", "auth-core", InboxStateOpen, "2026-11-15T01:30:00+09:00", "TYPE: QUESTION\n\nbefore the disband, 16:30Z the day before"), // dropped; the string is later
		progressInboxItem("e2", "auth-core", InboxStateOpen, "2026-11-15T09:30:00+09:00", "TYPE: QUESTION\n\nbefore the disband, 00:30Z"),                // dropped; the string is later
		progressInboxItem("e3", "auth-core", InboxStateAcked, "2026-11-15T12:00:00+09:00", "TYPE: QUESTION\n\nafter the disband, 03:00Z"),                // kept
		progressInboxItem("e4", "auth-core", InboxStateOpen, "2026-11-15T01:00:01Z", "TYPE: QUESTION\n\none second after"),                               // kept
		progressInboxItem("e5", "auth-core", InboxStateOpen, "2026-11-15T01:00:00Z", "TYPE: QUESTION\n\nat the second of the disband"),                   // dropped
		progressInboxItem("e6", "other", InboxStateOpen, "2026-11-15T05:00:00Z", "TYPE: QUESTION\n\nanother org's"),                                      // dropped
		progressInboxItem("e7", "auth-core", InboxStateResolved, "2026-11-15T05:00:00Z", "TYPE: QUESTION\n\nresolved"),                                   // dropped
		progressInboxItem("e8", "auth-core", InboxStateOpen, "yesterday", "TYPE: QUESTION\n\nits time does not parse"),                                   // kept, noted
	}
	p := readReseatProgress([]ManifestEvent{disband}, "auth-core", items)
	var ids []string
	for _, it := range p.Inbox {
		ids = append(ids, it.Item.ID)
	}
	if got := strings.Join(ids, ","); got != "e3,e4,e8" {
		t.Fatalf("items = %s, want e3,e4,e8", got)
	}
	if p.Inbox[0].TimeUnreadable || p.Inbox[1].TimeUnreadable || !p.Inbox[2].TimeUnreadable {
		t.Errorf("TimeUnreadable = %v %v %v, want false false true", p.Inbox[0].TimeUnreadable, p.Inbox[1].TimeUnreadable, p.Inbox[2].TimeUnreadable)
	}
	got := strings.Join(progressSectionLines(t, reseatProgressSection(p, reseatProgressSupplied{})), "\n")
	want := strings.Join([]string{
		"片付いていない受信箱の件: 3 件(古い順に 3 件)",
		"- e3 QUESTION auth-core acked: after the disband, 03:00Z",
		"- e4 QUESTION auth-core open: one second after",
		"- e8 QUESTION auth-core open(上がった時刻が読めなかった): its time does not parse",
	}, "\n")
	if got != want {
		t.Errorf("block =\n%s\nwant\n%s", got, want)
	}

	t.Run("a disband the other way round: the string is earlier, the time later", func(t *testing.T) {
		disband := progressOrgEvent("auth-core", "2026-11-15T10:00:00+09:00", EventDisbanded, "") // 01:00Z
		items := []InboxItem{
			progressInboxItem("e1", "auth-core", InboxStateOpen, "2026-11-15T02:00:00Z", "TYPE: QUESTION\n\nafter, and its string is before"),
			progressInboxItem("e2", "auth-core", InboxStateOpen, "2026-11-15T00:30:00Z", "TYPE: QUESTION\n\nbefore"),
		}
		p := readReseatProgress([]ManifestEvent{disband}, "auth-core", items)
		if len(p.Inbox) != 1 || p.Inbox[0].Item.ID != "e1" || p.Inbox[0].TimeUnreadable {
			t.Errorf("items = %+v, want only e1", p.Inbox)
		}
	})
	t.Run("an org with no disband keeps all of its unsettled items, whatever their time", func(t *testing.T) {
		p := readReseatProgress(nil, "auth-core", items)
		// e1 to e5 and e8 are auth-core's and not resolved; e6 is another org's and e7 is resolved
		if len(p.Inbox) != 6 {
			t.Fatalf("items = %d, want 6", len(p.Inbox))
		}
		for _, it := range p.Inbox {
			if it.TimeUnreadable {
				t.Errorf("item %s says its time is unreadable, but nothing is compared with it", it.Item.ID)
			}
		}
	})
	t.Run("a disband whose time does not parse keeps every unsettled item, noted", func(t *testing.T) {
		disband := progressOrgEvent("auth-core", "garbled", EventDisbanded, "")
		p := readReseatProgress([]ManifestEvent{disband}, "auth-core", items)
		if len(p.Inbox) != 6 {
			t.Fatalf("items = %d, want 6", len(p.Inbox))
		}
		for _, it := range p.Inbox {
			if !it.TimeUnreadable {
				t.Errorf("item %s is not noted as unreadable", it.Item.ID)
			}
		}
	})
}

func TestReseatProgress_MoreThanTwentyItemsAndCommitsAreCounted(t *testing.T) {
	var items []InboxItem
	for i := 1; i <= 23; i++ {
		items = append(items, progressInboxItem(fmt.Sprintf("e%d", i), "auth-core", InboxStateOpen, "2026-11-15T01:00:00Z",
			fmt.Sprintf("TYPE: QUESTION\n\nquestion %d", i)))
	}
	var commits []string
	for i := 1; i <= 25; i++ {
		commits = append(commits, fmt.Sprintf("c%06d subject %d", i, i))
	}
	section := reseatProgressSection(readReseatProgress(nil, "auth-core", items),
		reseatProgressSupplied{Commits: commits, CommitTotal: 31})
	block := progressSectionLines(t, section)

	lines := strings.Join(block, "\n")
	for _, want := range []string{
		"片付いていない受信箱の件: 23 件(古い順に 20 件)\n",
		"- e20 QUESTION auth-core open: question 20\n- ほか 3 件(新しいもの)\n",
		"branch の commit: 31 件(新しい順に 20 件)\n",
		"- c000020 subject 20",
	} {
		if !strings.Contains(lines, want) {
			t.Errorf("block lacks %q:\n%s", want, lines)
		}
	}
	for _, unwanted := range []string{"- e21 ", "- c000021 "} {
		if strings.Contains(lines, unwanted) {
			t.Errorf("block has %q, past the 20 it lists:\n%s", unwanted, lines)
		}
	}
	if !strings.HasSuffix(lines, "- c000020 subject 20") {
		t.Errorf("block does not end at the 20th commit:\n%s", lines)
	}

	t.Run("exactly twenty items list no remainder", func(t *testing.T) {
		got := reseatProgressSection(readReseatProgress(nil, "auth-core", items[:20]), reseatProgressSupplied{})
		if strings.Contains(got, "ほか") {
			t.Errorf("section has a remainder line:\n%s", got)
		}
	})
	t.Run("a total below the lines given is raised to them", func(t *testing.T) {
		got := reseatProgressSection(reseatProgress{}, reseatProgressSupplied{Commits: commits[:3], CommitTotal: 0})
		if !strings.Contains(got, "branch の commit: 3 件(新しい順に 3 件)") {
			t.Errorf("section =\n%s", got)
		}
	})
}

func TestReseatProgress_ValuesFromPeopleAreOneCleanLine(t *testing.T) {
	longSubject := strings.Repeat("あ", 200)
	section := reseatProgressSection(
		readReseatProgress(nil, "auth-core", []InboxItem{
			progressInboxItem("e1", "auth-core", InboxStateOpen, "2026-11-15T01:00:00Z",
				"TYPE: QUESTION\n\n\x1b[31mred\x1b[0m\ttab\rcarriage \u2028line separator"),
			progressInboxItem("e2", "auth-core", InboxStateOpen, "2026-11-15T01:00:00Z", "TYPE: QUESTION\n\n"+longSubject+"\nsecond line"),
		}),
		reseatProgressSupplied{Commits: []string{"abc1234 " + longSubject, "def5678 bell\x07 and nul\x00"}, CommitTotal: 2})
	block := progressSectionLines(t, section)

	for _, l := range block {
		for _, r := range l {
			if r < 0x20 || r == 0x7f || r == 0x2028 || r == 0x2029 {
				t.Errorf("line has the control character %U: %q", r, l)
			}
		}
	}
	cut := strings.Repeat("あ", reseatProgressLineRunes-1) + "…"
	for _, want := range []string{
		`- e1 QUESTION auth-core open: \x1b[31mred\x1b[0m\ttab\rcarriage \u2028line separator`,
		"- e2 QUESTION auth-core open: " + cut,
		"- abc1234 " + cut,
		`- def5678 bell\a and nul\x00`,
	} {
		if !progressHasLine(block, want) {
			t.Errorf("block lacks the line %q:\n%s", want, strings.Join(block, "\n"))
		}
	}
	if n := utf8.RuneCountInString(cut); n != reseatProgressLineRunes {
		t.Errorf("a cut subject is %d runes, want %d", n, reseatProgressLineRunes)
	}

	t.Run("a text of exactly the cap is not cut", func(t *testing.T) {
		text := strings.Repeat("x", reseatProgressLineRunes)
		if got := reseatProgressText(text, reseatProgressLineRunes); got != text {
			t.Errorf("got %q", got)
		}
		if got := reseatProgressText(text+"x", reseatProgressLineRunes); utf8.RuneCountInString(got) != reseatProgressLineRunes || !strings.HasSuffix(got, "…") {
			t.Errorf("got %q", got)
		}
	})
	t.Run("a path and a digest from the ledger are cleaned too", func(t *testing.T) {
		p := reseatProgress{
			Approval: &reseatProgressApproval{Plan: "docs/plans/a\x1bb.md", Digest: "0123456789ab", TS: "2026-11-15T01:00:00Z"},
			Gate: &reseatProgressGate{Pass: true, Event: EventGatePassed, Head: progressHeadA, Base: progressBase,
				Branch: "feat/a\nb", Log: "/state/\x1b.log", TS: "2026-11-15T02:00:00Z"},
			Previous: &reseatProgressReseated{TS: "2026-11-15T03:00:00Z", Details: "id=r1\nseats="},
		}
		block := progressSectionLines(t, reseatProgressSection(p, reseatProgressSupplied{}))
		joined := strings.Join(block, "\n")
		for _, want := range []string{`計画の承認: docs/plans/a\x1bb.md sha256:`, `branch feat/a\nb(`, `log /state/\x1b.log`, `id=r1\nseats=`} {
			if !strings.Contains(joined, want) {
				t.Errorf("block lacks %q:\n%s", want, joined)
			}
		}
		if len(block) != 3 {
			t.Errorf("block has %d lines, want 3:\n%s", len(block), joined)
		}
	})
}

func TestReseatProgress_AHeadingOrBackticksInATextDoNotLeaveTheFence(t *testing.T) {
	body := "TYPE: QUESTION\n\n## heading ``````` and ```\n## second"
	section := reseatProgressSection(
		readReseatProgress(nil, "auth-core", []InboxItem{progressInboxItem("e1", "auth-core", InboxStateOpen, "2026-11-15T01:00:00Z", body)}),
		reseatProgressSupplied{Commits: []string{"abc1234 ## fix ````` the fence"}, CommitTotal: 1})
	block := progressSectionLines(t, section) // fails on a line with a heading or the fence

	// the longest backtick run in the block is 7, so the fence has 8
	fence := strings.Split(section, "\n")[1]
	if fence != strings.Repeat("`", 8) {
		t.Errorf("fence = %q, want 8 backticks", fence)
	}
	if !progressHasLine(block, "- e1 QUESTION auth-core open: ## heading ``````` and ```") {
		t.Errorf("the item line is not whole in the block:\n%s", strings.Join(block, "\n"))
	}
	if !progressHasLine(block, "- abc1234 ## fix ````` the fence") {
		t.Errorf("the commit line is not whole in the block:\n%s", strings.Join(block, "\n"))
	}
	// whatever is in the block, no line of the whole section but the lead and
	// the fences is outside it
	all := strings.Split(section, "\n")
	if all[0] != reseatProgressLead || all[1] != fence || all[len(all)-1] != fence {
		t.Errorf("the section is not lead, fence, block, fence:\n%s", section)
	}
	for _, l := range all {
		if strings.HasPrefix(l, "## ") {
			t.Errorf("a line of the section starts with a heading: %q", l)
		}
	}
}

func TestReseatProgress_ACommitReadFailureIsOneLineAndHidesTheCommits(t *testing.T) {
	p := reseatProgress{Approval: &reseatProgressApproval{Plan: "docs/plans/active/a.md", Digest: "0123456789ab", TS: "2026-11-15T01:00:00Z"}}
	section := reseatProgressSection(p, reseatProgressSupplied{
		Commits: []string{"abc1234 feat: x"}, CommitTotal: 1, CommitReadError: "no base: refs/heads/main is missing\nsecond line",
	})
	block := progressSectionLines(t, section)
	want := []string{
		"計画の承認: docs/plans/active/a.md sha256:0123456789ab (承認 2026-11-15T01:00:00Z)",
		`branch の commit: 読めなかった(no base: refs/heads/main is missing\nsecond line)`,
	}
	if strings.Join(block, "\n") != strings.Join(want, "\n") {
		t.Errorf("block =\n%s\nwant\n%s", strings.Join(block, "\n"), strings.Join(want, "\n"))
	}
}

// TestReseatProgress_TheApprovalLineGivesTheDigestAsOneWord covers the
// self-review's L3: the leader compares the digest on the approval line with
// `plan-visual.sh digest`, so the line, split on white space, must hold
// `sha256:<digest>` alone, with no time or parenthesis stuck to it; the time
// follows as its own words.
func TestReseatProgress_TheApprovalLineGivesTheDigestAsOneWord(t *testing.T) {
	p := reseatProgress{Approval: &reseatProgressApproval{Plan: "docs/plans/active/a.md", Digest: "0123456789ab", TS: "2026-11-15T01:00:00Z"}}
	block := progressSectionLines(t, reseatProgressSection(p, reseatProgressSupplied{}))
	if len(block) != 1 {
		t.Fatalf("block = %q, want the approval line alone", block)
	}
	words := strings.Fields(block[0])
	if !slices.Contains(words, "sha256:0123456789ab") {
		t.Errorf("words of the approval line = %q, want the word sha256:0123456789ab", words)
	}
	if n := len(words); n < 2 || words[n-2] != "(承認" || words[n-1] != "2026-11-15T01:00:00Z)" {
		t.Errorf("the line = %q, want the time after the digest as its own words, (承認 <time>)", block[0])
	}
}

// progressHasLine reports whether lines has a line equal to want.
func progressHasLine(lines []string, want string) bool {
	for _, l := range lines {
		if l == want {
			return true
		}
	}
	return false
}

// progressSentEvent is the `sent` event of a message to seat in orgID, with
// the Details as Send writes them: prefix is what send puts before the text,
// then the text cut by truncateForDetails.
func progressSentEvent(orgID, seat, ts, prefix, text string) ManifestEvent {
	return ManifestEvent{TS: ts, OrgID: orgID, SeatID: seat, Event: EventSent, Details: prefix + truncateForDetails(text)}
}

// progressTaskText is a typed TASK message with the TASK_ID taskID.
func progressTaskText(taskID string) string {
	return "TYPE: TASK\nTASK_ID: " + taskID + "\n\nスライス 1 を進める"
}

// TestReseatProgress_TheLastTaskPerSeatComesFromTheSentEvents covers finding 1
// of the cycle 1 cross-review: Send records the leader's message in the
// manifest and never in agmsg, so the section gives, per seat, the last TASK
// of the org's current life with its TASK_ID and time.
func TestReseatProgress_TheLastTaskPerSeatComesFromTheSentEvents(t *testing.T) {
	screen := (&sendScreenPath{herdrState: "idle"}).details()
	tests := []struct {
		name   string
		events []ManifestEvent
		want   []string // the lines after the label line, "" for none
	}{
		{
			name: "two TASKs to a seat give the later one",
			events: []ManifestEvent{
				progressSentEvent("auth-core", "implementer", "2026-11-15T01:00:00Z", "", progressTaskText("slice-1")),
				progressSentEvent("auth-core", "implementer", "2026-11-15T02:00:00Z", "", progressTaskText("slice-2")),
			},
			want: []string{"- implementer: slice-2(2026-11-15T02:00:00Z)"},
		},
		{
			name: "each seat has its own, listed by seat id",
			events: []ManifestEvent{
				progressSentEvent("auth-core", "reviewer", "2026-11-15T01:30:00Z", "", progressTaskText("gate-1")),
				progressSentEvent("auth-core", "implementer", "2026-11-15T01:00:00Z", "", progressTaskText("slice-1")),
			},
			want: []string{"- implementer: slice-1(2026-11-15T01:00:00Z)", "- reviewer: gate-1(2026-11-15T01:30:00Z)"},
		},
		{
			name: "a TASK before the last disband is not used, one after it is",
			events: []ManifestEvent{
				progressSentEvent("auth-core", "implementer", "2026-11-14T01:00:00Z", "", progressTaskText("old-slice")),
				progressSentEvent("auth-core", "reviewer", "2026-11-14T01:10:00Z", "", progressTaskText("old-gate")),
				progressOrgEvent("auth-core", "2026-11-14T02:00:00Z", EventDisbanded, ""),
				progressSentEvent("auth-core", "implementer", "2026-11-15T01:00:00Z", "", progressTaskText("new-slice")),
			},
			want: []string{"- implementer: new-slice(2026-11-15T01:00:00Z)"},
		},
		{
			name: "a dry-run send, another org's, the leader's own, and a send with no seat are not used",
			events: []ManifestEvent{
				progressSentEvent("auth-core", "implementer", "2026-11-15T01:00:00Z", "", progressTaskText("real")),
				{TS: "2026-11-15T02:00:00Z", OrgID: "auth-core", SeatID: "implementer", Event: EventSent, DryRun: true,
					Details: truncateForDetails(progressTaskText("dry-run"))},
				progressSentEvent("other", "implementer", "2026-11-15T03:00:00Z", "", progressTaskText("other-org")),
				progressSentEvent("auth-core", LeaderIdentity, "2026-11-15T04:00:00Z", "", progressTaskText("to-leader")),
				progressSentEvent("auth-core", "", "2026-11-15T05:00:00Z", "", progressTaskText("no-seat")),
			},
			want: []string{"- implementer: real(2026-11-15T01:00:00Z)"},
		},
		{
			name: "a typed message of another TYPE sent later does not replace the TASK",
			events: []ManifestEvent{
				progressSentEvent("auth-core", "implementer", "2026-11-15T01:00:00Z", "", progressTaskText("slice-1")),
				progressSentEvent("auth-core", "implementer", "2026-11-15T02:00:00Z", "", "TYPE: DECISION\nTASK_ID: slice-1\n\n進めてよい"),
				progressSentEvent("auth-core", "implementer", "2026-11-15T03:00:00Z", "", "TYPE: STOP\n\nやめる"),
			},
			want: []string{"- implementer: slice-1(2026-11-15T01:00:00Z)"},
		},
		{
			name: "a seat that only got other messages has no line",
			events: []ManifestEvent{
				progressSentEvent("auth-core", "reviewer", "2026-11-15T01:00:00Z", "", "TYPE: DECISION\n\n進めてよい"),
			},
		},
		{
			name: "raw sends count when their first line reads TYPE: TASK",
			events: []ManifestEvent{
				progressSentEvent("auth-core", "implementer", "2026-11-15T01:00:00Z", "raw=true ", progressTaskText("raw-1")),
				progressSentEvent("auth-core", "reviewer", "2026-11-15T01:10:00Z", "raw=true ", "hello there=1\nTYPE: TASK\nTASK_ID: raw-2"),
				progressSentEvent("auth-core", "tester", "2026-11-15T01:20:00Z", "raw=true ", "TYPE: TASK\n\nno task id"),
			},
			want: []string{"- implementer: raw-1(2026-11-15T01:00:00Z)", "- tester: -(2026-11-15T01:20:00Z)"},
		},
		{
			name: "a send by the screen and one whose submit was not confirmed count",
			events: []ManifestEvent{
				progressSentEvent("auth-core", "implementer", "2026-11-15T01:00:00Z", screen, progressTaskText("screen-1")),
				progressSentEvent("auth-core", "reviewer", "2026-11-15T01:10:00Z", "submit_unconfirmed=true ", progressTaskText("unconfirmed-1")),
				progressSentEvent("auth-core", "tester", "2026-11-15T01:20:00Z", "raw=true "+screen+"submit_unconfirmed=true ", progressTaskText("all-three")),
			},
			want: []string{
				"- implementer: screen-1(2026-11-15T01:00:00Z)",
				"- reviewer: unconfirmed-1(2026-11-15T01:10:00Z)",
				"- tester: all-three(2026-11-15T01:20:00Z)",
			},
		},
		{
			name: "a long message cut to the Details length keeps its TYPE and TASK_ID",
			events: []ManifestEvent{
				progressSentEvent("auth-core", "implementer", "2026-11-15T01:00:00Z", screen,
					"TYPE: TASK\nTASK_ID: long-1\nPLAN: docs/plans/active/a.md sha256:0123456789ab\n\n"+strings.Repeat("あ", 300)),
			},
			want: []string{"- implementer: long-1(2026-11-15T01:00:00Z)"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := readReseatProgress(tt.events, "auth-core", nil)
			section := reseatProgressSection(p, reseatProgressSupplied{})
			if len(tt.want) == 0 {
				if len(p.Tasks) != 0 || section != "" {
					t.Errorf("Tasks = %+v, section = %q, want none", p.Tasks, section)
				}
				return
			}
			want := append([]string{fmt.Sprintf("座席ごとの最後の TASK: %d 席(台帳の sent の記録から)", len(tt.want))}, tt.want...)
			if got := progressSectionLines(t, section); strings.Join(got, "\n") != strings.Join(want, "\n") {
				t.Errorf("block =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
			}
		})
	}
}

// TestReseatProgress_TheDetailsPrefixesAreTheOnesSendWrites keeps the words
// reseatProgressSentTask strips in step with what Send puts before the text.
func TestReseatProgress_TheDetailsPrefixesAreTheOnesSendWrites(t *testing.T) {
	for _, prefix := range []string{"", "raw=true ", "via=screen herdr_state=idle ", "submit_unconfirmed=true ", "raw=true via=screen herdr_state=idle submit_unconfirmed=true "} {
		if id, ok := reseatProgressSentTask(prefix + progressTaskText("t-1")); !ok || id != "t-1" {
			t.Errorf("prefix %q: got (%q, %v), want (t-1, true)", prefix, id, ok)
		}
	}
	for _, details := range []string{"", "raw=true ", "raw=true hello", "TYPE: RESULT\nTASK_ID: t-1", "x=1\nTYPE: TASK\nTASK_ID: t-1"} {
		if id, ok := reseatProgressSentTask(details); ok {
			t.Errorf("details %q: got (%q, true), want not a TASK", details, id)
		}
	}
	if id, ok := reseatProgressSentTask("TYPE: TASK"); !ok || id != "" {
		t.Errorf("a TASK with no TASK_ID: got (%q, %v), want (\"\", true)", id, ok)
	}
}

// progressResultItem is an inbox item of type RESULT whose body is a typed
// message with the HEAD: header head (none when head is "").
func progressResultItem(id, orgID, state, escalatedAt, taskID, head, summary string) InboxItem {
	body := "TYPE: RESULT\nTASK_ID: " + taskID + "\n"
	if head != "" {
		body += "HEAD: " + head + "\nBASE: " + progressBase + "\n"
	}
	body += "\n" + summary
	return InboxItem{ID: id, OrgID: orgID, Type: "RESULT", TaskID: taskID, Body: body, State: state, EscalatedAt: escalatedAt}
}

// TestReseatProgress_TheResultsTheLeaderRaisedAreListedInEveryState covers
// finding 2: the step that reports a PR is told as done by a RESULT of the
// org whose HEAD is the gate's, and a resolved RESULT is such a record too.
func TestReseatProgress_TheResultsTheLeaderRaisedAreListedInEveryState(t *testing.T) {
	disband := progressOrgEvent("auth-core", "2026-11-15T00:00:00Z", EventDisbanded, "")
	items := []InboxItem{
		progressResultItem("e1", "auth-core", InboxStateResolved, "2026-11-14T23:00:00Z", "auth-core", progressHeadB, "before the disband"),
		progressResultItem("e2", "auth-core", InboxStateResolved, "2026-11-15T01:00:00Z", "auth-core", progressHeadA, "PR を作った"),
		progressResultItem("e3", "auth-core", InboxStateOpen, "2026-11-15T02:00:00Z", "slice-1", "", "slice 1 が済んだ\n2 行目"),
		progressResultItem("e4", "auth-core", InboxStateAcked, "2026-11-15T03:00:00Z", "", progressHeadB, "no task id"),
		progressResultItem("e5", "other", InboxStateResolved, "2026-11-15T04:00:00Z", "other", progressHeadA, "another org's"),
		progressInboxItem("e6", "auth-core", InboxStateResolved, "2026-11-15T05:00:00Z", "TYPE: QUESTION\n\na resolved QUESTION is not a RESULT"),
	}
	p := readReseatProgress([]ManifestEvent{disband}, "auth-core", items)
	got := strings.Join(progressSectionLines(t, reseatProgressSection(p, reseatProgressSupplied{})), "\n")
	want := strings.Join([]string{
		"片付いていない受信箱の件: 2 件(古い順に 2 件)",
		"- e3 RESULT slice-1 open: slice 1 が済んだ",
		"- e4 RESULT - acked: no task id",
		"上げた RESULT: 3 件(新しい 3 件を古い順に)",
		"- e2 resolved auth-core HEAD " + progressHeadA + ": PR を作った",
		"- e3 open slice-1 HEAD -: slice 1 が済んだ",
		"- e4 acked - HEAD " + progressHeadB + ": no task id",
	}, "\n")
	if got != want {
		t.Errorf("block =\n%s\nwant\n%s", got, want)
	}

	t.Run("an org with only a resolved RESULT has the RESULT list and no inbox list", func(t *testing.T) {
		p := readReseatProgress(nil, "auth-core", items[1:2])
		got := strings.Join(progressSectionLines(t, reseatProgressSection(p, reseatProgressSupplied{})), "\n")
		want := "上げた RESULT: 1 件(新しい 1 件を古い順に)\n- e2 resolved auth-core HEAD " + progressHeadA + ": PR を作った"
		if got != want {
			t.Errorf("block =\n%s\nwant\n%s", got, want)
		}
	})
	t.Run("a RESULT whose time cannot be read is kept and noted", func(t *testing.T) {
		p := readReseatProgress([]ManifestEvent{disband}, "auth-core",
			[]InboxItem{progressResultItem("e1", "auth-core", InboxStateResolved, "yesterday", "auth-core", progressHeadA, "x")})
		got := strings.Join(progressSectionLines(t, reseatProgressSection(p, reseatProgressSupplied{})), "\n")
		if !strings.Contains(got, "- e1 resolved(上がった時刻が読めなかった) auth-core HEAD "+progressHeadA+": x") {
			t.Errorf("block =\n%s", got)
		}
	})
	t.Run("a body that is not a typed message has no HEAD", func(t *testing.T) {
		item := InboxItem{ID: "e1", OrgID: "auth-core", Type: "RESULT", TaskID: "auth-core", State: InboxStateResolved,
			EscalatedAt: "2026-11-15T01:00:00Z", Body: "free text\nHEAD: " + progressHeadA}
		got := strings.Join(progressSectionLines(t, reseatProgressSection(readReseatProgress(nil, "auth-core", []InboxItem{item}), reseatProgressSupplied{})), "\n")
		if !strings.Contains(got, "- e1 resolved auth-core HEAD -: free text") {
			t.Errorf("block =\n%s", got)
		}
	})
}

func TestReseatProgress_MoreThanFiveResultsKeepTheNewestFive(t *testing.T) {
	var items []InboxItem
	for i := 1; i <= 8; i++ {
		items = append(items, progressResultItem(fmt.Sprintf("e%d", i), "auth-core", InboxStateResolved, "2026-11-15T01:00:00Z",
			"auth-core", progressHeadA, fmt.Sprintf("result %d", i)))
	}
	got := strings.Join(progressSectionLines(t, reseatProgressSection(readReseatProgress(nil, "auth-core", items), reseatProgressSupplied{})), "\n")
	want := strings.Join([]string{
		"上げた RESULT: 8 件(新しい 5 件を古い順に)",
		"- ほか 3 件(古いもの)",
		"- e4 resolved auth-core HEAD " + progressHeadA + ": result 4",
		"- e5 resolved auth-core HEAD " + progressHeadA + ": result 5",
		"- e6 resolved auth-core HEAD " + progressHeadA + ": result 6",
		"- e7 resolved auth-core HEAD " + progressHeadA + ": result 7",
		"- e8 resolved auth-core HEAD " + progressHeadA + ": result 8",
	}, "\n")
	if got != want {
		t.Errorf("block =\n%s\nwant\n%s", got, want)
	}
	t.Run("exactly five list no remainder", func(t *testing.T) {
		got := reseatProgressSection(readReseatProgress(nil, "auth-core", items[:5]), reseatProgressSupplied{})
		if strings.Contains(got, "ほか") || !strings.Contains(got, "上げた RESULT: 5 件(新しい 5 件を古い順に)") {
			t.Errorf("section =\n%s", got)
		}
	})
}

// TestReseatProgress_AQuestionLineShowsItsPlanHeader covers finding 4: the
// leader tells whether the plan advisory ran by a QUESTION that names its plan
// and digest, so the line shows the PLAN: header of a QUESTION, and only of a
// QUESTION.
func TestReseatProgress_AQuestionLineShowsItsPlanHeader(t *testing.T) {
	plan := "docs/plans/active/2026-11-15-auth.md sha256:0123456789ab"
	items := []InboxItem{
		progressInboxItem("e1", "auth-core", InboxStateOpen, "2026-11-15T01:00:00Z",
			"TYPE: QUESTION\nPLAN:   docs/plans/active/2026-11-15-auth.md    sha256:0123456789ab \nVISUAL: .harness/state/plan-visual/auth.html\n\nSUMMARY: 承認を求める\nADVISORY: 所見 2 件"),
		progressInboxItem("e2", "auth-core", InboxStateAcked, "2026-11-15T01:10:00Z", "TYPE: QUESTION\n\nSUMMARY: 計画の話ではない"),
		{ID: "e3", OrgID: "auth-core", Type: "BLOCKED", TaskID: "auth-core", State: InboxStateOpen, EscalatedAt: "2026-11-15T01:20:00Z",
			Body: "TYPE: BLOCKED\nTASK_ID: auth-core\nPLAN: " + plan + "\n\n止まった"},
	}
	got := strings.Join(progressSectionLines(t, reseatProgressSection(readReseatProgress(nil, "auth-core", items), reseatProgressSupplied{})), "\n")
	want := strings.Join([]string{
		"片付いていない受信箱の件: 3 件(古い順に 3 件)",
		"- e1 QUESTION auth-core open: SUMMARY: 承認を求める / PLAN " + plan,
		"- e2 QUESTION auth-core acked: SUMMARY: 計画の話ではない",
		"- e3 BLOCKED auth-core open: 止まった",
	}, "\n")
	if got != want {
		t.Errorf("block =\n%s\nwant\n%s", got, want)
	}
	// the last word is the digest alone, as the approval line's is
	words := strings.Fields(strings.Split(got, "\n")[1])
	if words[len(words)-1] != "sha256:0123456789ab" {
		t.Errorf("last word of the QUESTION line = %q, want sha256:0123456789ab", words[len(words)-1])
	}
}
