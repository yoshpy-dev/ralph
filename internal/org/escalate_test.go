package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// escalateFixture is an Org with an inbox and an escalations.jsonl in a
// scratch state dir and a stub desktop notification that records its
// messages and returns notifyErr.
type escalateFixture struct {
	o        *Org
	stateDir string
	banner   bytes.Buffer

	mu          sync.Mutex
	notifyCalls []string
	notifyErr   error
	// onNotify, when set, runs inside the stub before it returns.
	onNotify func()
}

func newEscalateFixture(t *testing.T) *escalateFixture {
	t.Helper()
	return newEscalateFixtureAt(t, t.TempDir())
}

func newEscalateFixtureAt(t *testing.T, stateDir string) *escalateFixture {
	t.Helper()
	clock := func() time.Time { return time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC) }
	f := &escalateFixture{stateDir: stateDir}
	inbox := NewInboxStore(stateDir)
	inbox.Now = clock
	f.o = &Org{
		Inbox:           inbox,
		EscalationsPath: EscalationsPathIn(stateDir),
		Now:             clock,
		DesktopNotify: func(_ context.Context, message string) error {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.notifyCalls = append(f.notifyCalls, message)
			if f.onNotify != nil {
				f.onNotify()
			}
			return f.notifyErr
		},
	}
	return f
}

func (f *escalateFixture) escalate(orgID, text string) EscalateResult {
	return f.o.Escalate(EscalateParams{OrgID: orgID, Text: text, Banner: &f.banner})
}

func (f *escalateFixture) notifications() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.notifyCalls)
}

// fileOrAbsent returns path's content, or "<absent>" when it does not exist.
func fileOrAbsent(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "<absent>"
	}
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

// escalationLines parses every line of escalations.jsonl.
func escalationLines(t *testing.T, path string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range readJSONLFile(t, path) {
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("parse escalation line %q: %v", line, err)
		}
		out = append(out, rec)
	}
	return out
}

func TestEscalate_RefusalsWriteNothingAndNotifyNoOne(t *testing.T) {
	cases := []struct {
		name    string
		orgID   string
		text    string
		wantIs  error
		wantSub string
	}{
		{name: "TASK", orgID: "org-a", text: "TYPE: TASK\nTASK_ID: org-a\n\ndo it", wantSub: "TYPE TASK cannot be escalated"},
		{name: "DECISION", orgID: "org-a", text: "TYPE: DECISION\n\nuse plan B", wantSub: "TYPE DECISION cannot be escalated"},
		{name: "HELLO", orgID: "org-a", text: "TYPE: HELLO\n\nhi", wantSub: "TYPE HELLO cannot be escalated"},
		{name: "unknown TYPE", orgID: "org-a", text: "TYPE: URGENT\n\nhelp", wantIs: protocol.ErrUnknownType},
		{name: "no TYPE header", orgID: "org-a", text: "please look at this", wantIs: protocol.ErrMissingType},
		{name: "empty text", orgID: "org-a", text: "", wantIs: protocol.ErrMissingType},
		{name: "BLOCKED without TASK_ID", orgID: "org-a", text: "TYPE: BLOCKED\n\ncannot push", wantIs: protocol.ErrMissingTaskID},
		{name: "RESULT without TASK_ID", orgID: "org-a", text: "TYPE: RESULT\n\nPR opened", wantIs: protocol.ErrMissingTaskID},
		{name: "RESULT with a blank TASK_ID", orgID: "org-a", text: "TYPE: RESULT\nTASK_ID:  \n\nPR opened", wantIs: protocol.ErrMissingTaskID},
		{name: "body over 2000", orgID: "org-a", text: "TYPE: QUESTION\n\n" + strings.Repeat("x", protocol.DefaultMaxBodyChars+1), wantIs: protocol.ErrBodyTooLarge},
		{name: "org_id with an upper-case letter", orgID: "Org-a", text: "TYPE: QUESTION\n\nwhich?", wantSub: `invalid org_id "Org-a"`},
		{name: "org_id with a path", orgID: "../org-a", text: "TYPE: QUESTION\n\nwhich?", wantSub: "invalid org_id"},
		{name: "empty org_id", orgID: "", text: "TYPE: QUESTION\n\nwhich?", wantSub: "invalid org_id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// A fresh state dir: both files stay absent.
			fresh := newEscalateFixture(t)
			// A state dir with one item already sent: both files stay
			// byte-identical.
			used := newEscalateFixture(t)
			if res := used.escalate("org-a", "TYPE: QUESTION\n\nfirst?"); res.Err != nil {
				t.Fatalf("seed escalate: %v", res.Err)
			}
			used.banner.Reset()
			for _, f := range []*escalateFixture{fresh, used} {
				inboxPath, escPath := f.o.Inbox.Path(), f.o.EscalationsPath
				inboxBefore, escBefore := fileOrAbsent(t, inboxPath), fileOrAbsent(t, escPath)
				callsBefore := len(f.notifications())

				res := f.escalate(tc.orgID, tc.text)

				if res.Err == nil {
					t.Fatalf("Escalate(%q, %q) = %+v; want a refusal", tc.orgID, tc.text, res)
				}
				if tc.wantIs != nil && !errors.Is(res.Err, tc.wantIs) {
					t.Errorf("error %q is not %v", res.Err, tc.wantIs)
				}
				if tc.wantSub != "" && !strings.Contains(res.Err.Error(), tc.wantSub) {
					t.Errorf("error %q does not contain %q", res.Err, tc.wantSub)
				}
				if res.ID != "" || res.Type != "" || res.Recorded || res.Notified {
					t.Errorf("a refusal reported %+v; want no ID or TYPE, not recorded, not notified", res)
				}
				if got := fileOrAbsent(t, inboxPath); got != inboxBefore {
					t.Errorf("inbox.jsonl changed on a refusal:\nbefore: %s\nafter:  %s", inboxBefore, got)
				}
				if got := fileOrAbsent(t, escPath); got != escBefore {
					t.Errorf("escalations.jsonl changed on a refusal:\nbefore: %s\nafter:  %s", escBefore, got)
				}
				if n := len(f.notifications()); n != callsBefore {
					t.Errorf("the desktop notification was called on a refusal (%d -> %d calls)", callsBefore, n)
				}
				if f.banner.Len() != 0 {
					t.Errorf("a refusal wrote a banner: %q", f.banner.String())
				}
			}
			if _, err := os.Stat(filepath.Join(fresh.stateDir, inboxLockFile)); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("a refusal created inbox.lock (stat err %v)", err)
			}
		})
	}
}

func TestEscalate_AcceptsQuestionBlockedAndResult(t *testing.T) {
	f := newEscalateFixture(t)
	texts := []string{
		"TYPE: QUESTION\n\nplan A or plan B?",
		"TYPE: QUESTION\nTASK_ID: org-a\n\nwhich branch?",
		"TYPE: BLOCKED\nTASK_ID: org-a\n\ngh auth is missing",
		"TYPE: RESULT\nTASK_ID: org-a\n\nPR https://example.invalid/pull/1",
		"TYPE: QUESTION\n\n" + strings.Repeat("y", protocol.DefaultMaxBodyChars),
	}
	wantTypes := []string{"QUESTION", "QUESTION", "BLOCKED", "RESULT", "QUESTION"}
	for i, text := range texts {
		res := f.escalate("org-a", text)
		if res.Err != nil {
			t.Fatalf("Escalate(%q): %v", text, res.Err)
		}
		if want := fmt.Sprintf("e%d", i+1); res.ID != want || res.Type != wantTypes[i] || !res.Recorded || !res.Notified {
			t.Fatalf("Escalate #%d = %+v; want ID %s, TYPE %s, recorded and notified", i+1, res, want, wantTypes[i])
		}
	}
	inbox := mustReadInbox(t, f.o.Inbox)
	if got := inboxItemIDs(inbox); !slices.Equal(got, []string{"e1", "e2", "e3", "e4", "e5"}) {
		t.Fatalf("items = %v", got)
	}
	blocked := inboxItemByID(t, inbox, "e3")
	if blocked.Type != "BLOCKED" || blocked.TaskID != "org-a" || blocked.OrgID != "org-a" || blocked.Body != texts[2] {
		t.Errorf("e3 = %+v; want TYPE BLOCKED, TASK_ID org-a, the whole message as the body", blocked)
	}
	if q := inboxItemByID(t, inbox, "e1"); q.TaskID != "" {
		t.Errorf("e1 has TASK_ID %q; want none", q.TaskID)
	}
}

func TestEscalate_RecordsSendsToTheHumanPathAndMarksNotified(t *testing.T) {
	f := newEscalateFixture(t)
	const secret = "the body must not reach osascript"
	res := f.escalate("org-a", "TYPE: QUESTION\n\n"+secret)
	if res.Err != nil || res.ID != "e1" || !res.Recorded || !res.Notified {
		t.Fatalf("Escalate = %+v; want e1, recorded and notified", res)
	}
	res2 := f.escalate("org-b", "TYPE: BLOCKED\nTASK_ID: org-b\n\nstuck")
	if res2.Err != nil || res2.ID != "e2" || !res2.Notified {
		t.Fatalf("second Escalate = %+v; want e2, notified", res2)
	}

	lines := escalationLines(t, f.o.EscalationsPath)
	if len(lines) != 2 {
		t.Fatalf("escalations.jsonl has %d lines; want 2: %v", len(lines), lines)
	}
	want := map[string]any{"ts": inboxTestNow, "org_id": "org-a", "inbox_id": "e1", "subject": "leader", "reason": "inbox_no_director"}
	for k, v := range want {
		if lines[0][k] != v {
			t.Errorf("escalations line 1 %s = %v; want %v (line %v)", k, lines[0][k], v, lines[0])
		}
	}
	if _, ok := lines[0]["alert_id"]; ok {
		t.Errorf("an inbox escalation line carries alert_id: %v", lines[0])
	}
	if lines[1]["inbox_id"] != "e2" || lines[1]["org_id"] != "org-b" {
		t.Errorf("escalations line 2 = %v; want inbox_id e2, org_id org-b", lines[1])
	}

	banner := f.banner.String()
	if n := strings.Count(banner, "\n"); n != 2 {
		t.Errorf("banner has %d lines; want one per escalate: %q", n, banner)
	}
	for _, sub := range []string{"ORG ESCALATION:", "org=org-a", "item=e1", "type=QUESTION", f.o.Inbox.Path(), "ralph org inbox show e1"} {
		if !strings.Contains(banner, sub) {
			t.Errorf("banner %q does not contain %q", banner, sub)
		}
	}
	if strings.Contains(banner, secret) {
		t.Errorf("banner carries the message body: %q", banner)
	}

	calls := f.notifications()
	if len(calls) != 2 {
		t.Fatalf("desktop notification called %d times; want 2: %v", len(calls), calls)
	}
	for _, sub := range []string{"org-a", "e1", "QUESTION"} {
		if !strings.Contains(calls[0], sub) {
			t.Errorf("notification %q does not name %q", calls[0], sub)
		}
	}
	if strings.Contains(calls[0], secret) {
		t.Errorf("the message body reached the desktop notification: %q", calls[0])
	}

	item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1")
	if item.State != InboxStateOpen || !item.Notified || item.NotifiedAt != inboxTestNow {
		t.Errorf("e1 = %+v; want open and notified at %s", item, inboxTestNow)
	}
	if got := inboxEventKinds(item); !slices.Equal(got, []string{InboxEventEscalated, InboxEventNotified}) {
		t.Fatalf("e1 events = %v", got)
	}
	if ev := item.Events[1]; ev.Reason != "inbox_no_director" || ev.Osascript != "ok" {
		t.Errorf("notified event = %+v; want reason inbox_no_director, osascript ok", ev)
	}
}

func TestEscalate_DesktopNotificationFailureStillSucceeds(t *testing.T) {
	f := newEscalateFixture(t)
	f.notifyErr = errors.New("exit status 1: Not authorized to send Apple events")
	res := f.escalate("org-a", "TYPE: QUESTION\n\nwhich?")
	if res.Err != nil || !res.Recorded || !res.Notified {
		t.Fatalf("Escalate = %+v; a failed desktop notification must not fail the escalate", res)
	}
	item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), res.ID)
	if got, want := item.Events[len(item.Events)-1].Osascript, "failed: "+f.notifyErr.Error(); got != want {
		t.Errorf("notified osascript = %q; want %q", got, want)
	}
	if lines := escalationLines(t, f.o.EscalationsPath); len(lines) != 1 {
		t.Errorf("escalations.jsonl has %d lines; want 1", len(lines))
	}
}

func TestEscalate_NoDesktopNotificationOffDarwinIsSkipped(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("with DesktopNotify nil, darwin runs the real osascript")
	}
	f := newEscalateFixture(t)
	f.o.DesktopNotify = nil
	res := f.escalate("org-a", "TYPE: QUESTION\n\nwhich?")
	if res.Err != nil || !res.Notified {
		t.Fatalf("Escalate = %+v", res)
	}
	item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), res.ID)
	if got := item.Events[len(item.Events)-1].Osascript; got != "skipped" {
		t.Errorf("notified osascript = %q; want skipped", got)
	}
}

func TestEscalate_ConcurrentEscalatesGetDistinctIDs(t *testing.T) {
	f := newEscalateFixture(t)
	const n = 8
	ids := make([]string, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res := f.o.Escalate(EscalateParams{OrgID: "org-a", Text: fmt.Sprintf("TYPE: QUESTION\n\nq%d", i), Banner: &bytes.Buffer{}})
			if res.Err != nil {
				t.Errorf("Escalate %d: %v", i, res.Err)
			}
			ids[i] = res.ID
		}()
	}
	wg.Wait()
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if len(ids) != n {
		t.Fatalf("IDs = %v; want %d distinct", ids, n)
	}
	inbox := mustReadInbox(t, f.o.Inbox)
	if len(inbox.Items) != n || inbox.CorruptLines != 0 {
		t.Fatalf("inbox has %d items, %d corrupt lines; want %d, 0", len(inbox.Items), inbox.CorruptLines, n)
	}
	for _, item := range inbox.Items {
		if !item.Notified {
			t.Errorf("%s is not notified", item.ID)
		}
	}
}

// AC2b: the item is recorded, but escalations.jsonl cannot be written.
func TestEscalate_EscalationsUnwritable_ReportsTheIDAndInboxNotifyCompletesIt(t *testing.T) {
	f := newEscalateFixture(t)
	if err := os.Mkdir(f.o.EscalationsPath, 0o755); err != nil {
		t.Fatal(err)
	}
	res := f.escalate("org-a", "TYPE: QUESTION\n\nwhich?")
	if res.Err == nil {
		t.Fatalf("Escalate = %+v; want an error", res)
	}
	if res.ID != "e1" || res.Type != "QUESTION" || !res.Recorded || res.Notified {
		t.Fatalf("Escalate = %+v; want e1 (QUESTION) recorded, not notified", res)
	}
	for _, sub := range []string{"recorded as e1", "notification was not completed", "ralph org inbox notify e1", "not escalate"} {
		if !strings.Contains(res.Err.Error(), sub) {
			t.Errorf("error %q does not contain %q", res.Err, sub)
		}
	}
	item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1")
	if item.State != InboxStateOpen || item.Notified {
		t.Fatalf("e1 = %+v; want open and not notified", item)
	}

	if err := os.Remove(f.o.EscalationsPath); err != nil {
		t.Fatal(err)
	}
	var banner bytes.Buffer
	if err := f.o.NotifyInboxItem("e1", "", &banner); err != nil {
		t.Fatalf("NotifyInboxItem: %v", err)
	}
	inbox := mustReadInbox(t, f.o.Inbox)
	if got := inboxItemIDs(inbox); !slices.Equal(got, []string{"e1"}) {
		t.Fatalf("items = %v; inbox notify must not add an item", got)
	}
	if !inbox.Items[0].Notified {
		t.Errorf("e1 is not notified after inbox notify: %+v", inbox.Items[0])
	}
	lines := escalationLines(t, f.o.EscalationsPath)
	if len(lines) != 1 || lines[0]["inbox_id"] != "e1" {
		t.Errorf("escalations.jsonl = %v; want one line for e1", lines)
	}
	if !strings.Contains(banner.String(), "item=e1") {
		t.Errorf("inbox notify banner = %q", banner.String())
	}
}

// AC2b: the item is recorded, but the notified event cannot be written.
func TestEscalate_NotifiedUnwritable_ReportsTheIDAndInboxNotifyCompletesIt(t *testing.T) {
	f := newEscalateFixture(t)
	inboxPath := f.o.Inbox.Path()
	var saved []byte
	f.onNotify = func() {
		// Swap inbox.jsonl for a directory once, between the escalated
		// event and the notified event.
		f.onNotify = nil
		raw, err := os.ReadFile(inboxPath)
		if err != nil {
			t.Errorf("read inbox: %v", err)
			return
		}
		saved = raw
		if err := os.Remove(inboxPath); err != nil {
			t.Errorf("remove inbox: %v", err)
		}
		if err := os.Mkdir(inboxPath, 0o755); err != nil {
			t.Errorf("mkdir inbox: %v", err)
		}
	}
	res := f.escalate("org-a", "TYPE: QUESTION\n\nwhich?")
	if res.Err == nil || res.ID != "e1" || !res.Recorded || res.Notified {
		t.Fatalf("Escalate = %+v; want e1 recorded, not notified, with an error", res)
	}
	if !strings.Contains(res.Err.Error(), "ralph org inbox notify e1") {
		t.Errorf("error %q does not point at ralph org inbox notify e1", res.Err)
	}

	if err := os.Remove(inboxPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inboxPath, saved, 0o644); err != nil {
		t.Fatal(err)
	}
	if item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1"); item.Notified {
		t.Fatalf("e1 = %+v; want not notified before inbox notify", item)
	}
	if err := f.o.NotifyInboxItem("e1", "", &bytes.Buffer{}); err != nil {
		t.Fatalf("NotifyInboxItem: %v", err)
	}
	inbox := mustReadInbox(t, f.o.Inbox)
	if got := inboxItemIDs(inbox); !slices.Equal(got, []string{"e1"}) || !inbox.Items[0].Notified {
		t.Fatalf("inbox = %+v; want only e1, notified", inbox.Items)
	}
}

// AC2b: the inbox cannot be written; the human is still told.
func TestEscalate_InboxUnwritable_AlertsTheHumanAndFails(t *testing.T) {
	t.Run("inbox.jsonl is a directory", func(t *testing.T) {
		f := newEscalateFixture(t)
		if err := os.Mkdir(f.o.Inbox.Path(), 0o755); err != nil {
			t.Fatal(err)
		}
		res := f.escalate("org-a", "TYPE: BLOCKED\nTASK_ID: org-a\n\nstuck")
		assertUnrecordedEscalation(t, f, res)
		// escalations.jsonl is writable here, so the best-effort line is there.
		lines := escalationLines(t, f.o.EscalationsPath)
		if len(lines) != 1 || lines[0]["reason"] != "inbox_not_recorded" || lines[0]["org_id"] != "org-a" {
			t.Fatalf("escalations.jsonl = %v; want one inbox_not_recorded line for org-a", lines)
		}
		if _, ok := lines[0]["inbox_id"]; ok {
			t.Errorf("a not-recorded line carries inbox_id: %v", lines[0])
		}
	})
	t.Run("the state dir is a file", func(t *testing.T) {
		stateDir := filepath.Join(t.TempDir(), "ledger")
		if err := os.WriteFile(stateDir, []byte("not a dir"), 0o644); err != nil {
			t.Fatal(err)
		}
		f := newEscalateFixtureAt(t, stateDir)
		res := f.escalate("org-a", "TYPE: BLOCKED\nTASK_ID: org-a\n\nstuck")
		assertUnrecordedEscalation(t, f, res)
	})
}

func assertUnrecordedEscalation(t *testing.T, f *escalateFixture, res EscalateResult) {
	t.Helper()
	if res.Err == nil || res.ID != "" || res.Recorded || res.Notified {
		t.Fatalf("Escalate = %+v; want an error, no ID, not recorded", res)
	}
	if !strings.Contains(res.Err.Error(), "could not be recorded") {
		t.Errorf("error %q does not say the item was not recorded", res.Err)
	}
	banner := f.banner.String()
	for _, sub := range []string{"NOT RECORDED", "org=org-a", "type=BLOCKED", f.o.Inbox.Path()} {
		if !strings.Contains(banner, sub) {
			t.Errorf("banner %q does not contain %q", banner, sub)
		}
	}
	calls := f.notifications()
	if len(calls) != 1 || !strings.Contains(calls[0], "org-a") || strings.Contains(calls[0], "stuck") {
		t.Errorf("desktop notifications = %v; want one naming org-a without the body", calls)
	}
}

func TestNotifyInboxItem_RefusesUnknownAndResolvedItems(t *testing.T) {
	t.Run("empty inbox", func(t *testing.T) {
		f := newEscalateFixture(t)
		err := f.o.NotifyInboxItem("e1", "", &f.banner)
		if !errors.Is(err, ErrInboxUnknownID) {
			t.Fatalf("NotifyInboxItem on an empty inbox = %v; want ErrInboxUnknownID", err)
		}
		if f.banner.Len() != 0 || len(f.notifications()) != 0 {
			t.Errorf("an unknown ID was notified: banner %q, calls %v", f.banner.String(), f.notifications())
		}
		if got := fileOrAbsent(t, f.o.EscalationsPath); got != "<absent>" {
			t.Errorf("escalations.jsonl = %q; want absent", got)
		}
	})

	f := newEscalateFixture(t)
	if res := f.escalate("org-a", "TYPE: QUESTION\n\nwhich?"); res.Err != nil {
		t.Fatal(res.Err)
	}
	if err := f.o.Inbox.Resolve("e1", "docs/reports/x.md"); err != nil {
		t.Fatal(err)
	}
	inboxBefore, escBefore := fileOrAbsent(t, f.o.Inbox.Path()), fileOrAbsent(t, f.o.EscalationsPath)
	callsBefore := len(f.notifications())
	f.banner.Reset()

	t.Run("unknown ID", func(t *testing.T) {
		if err := f.o.NotifyInboxItem("e9", "", &f.banner); !errors.Is(err, ErrInboxUnknownID) {
			t.Fatalf("NotifyInboxItem(e9) = %v; want ErrInboxUnknownID", err)
		}
	})
	t.Run("resolved item", func(t *testing.T) {
		err := f.o.NotifyInboxItem("e1", "", &f.banner)
		if !errors.Is(err, ErrInboxResolved) {
			t.Fatalf("NotifyInboxItem on a resolved item = %v; want ErrInboxResolved", err)
		}
		if !strings.Contains(err.Error(), "nothing is left to notify") {
			t.Errorf("error %q", err)
		}
	})
	if fileOrAbsent(t, f.o.Inbox.Path()) != inboxBefore || fileOrAbsent(t, f.o.EscalationsPath) != escBefore {
		t.Errorf("a refused inbox notify changed inbox.jsonl or escalations.jsonl")
	}
	if len(f.notifications()) != callsBefore || f.banner.Len() != 0 {
		t.Errorf("a refused inbox notify notified: banner %q, calls %v", f.banner.String(), f.notifications())
	}
}

func TestNotifyInboxItem_SendsAnAckedItemAgain(t *testing.T) {
	f := newEscalateFixture(t)
	if res := f.escalate("org-a", "TYPE: QUESTION\n\nwhich?"); res.Err != nil {
		t.Fatal(res.Err)
	}
	if _, err := f.o.Inbox.Ack("e1"); err != nil {
		t.Fatal(err)
	}
	if err := f.o.NotifyInboxItem("e1", "", &f.banner); err != nil {
		t.Fatalf("NotifyInboxItem on an acked item: %v", err)
	}
	item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1")
	if item.State != InboxStateAcked {
		t.Errorf("e1 state = %s; inbox notify must not change the state", item.State)
	}
	if got := inboxEventKinds(item); !slices.Equal(got, []string{InboxEventEscalated, InboxEventNotified, InboxEventAcked, InboxEventNotified}) {
		t.Errorf("e1 events = %v", got)
	}
	if lines := escalationLines(t, f.o.EscalationsPath); len(lines) != 2 {
		t.Errorf("escalations.jsonl has %d lines; want 2 (escalate, inbox notify)", len(lines))
	}
}

// Self-review L9: a hand-edited inbox.jsonl line can put anything in an
// item's org_id and type. inbox notify escapes them in the banner and does
// not pass them to the desktop notification (an AppleScript string) unless
// escalate would have accepted them.
func TestNotifyInboxItem_HandEditedLineEscapesTheBannerAndKeepsTheNotificationNeutral(t *testing.T) {
	cases := []struct {
		name, orgID, typ string
		wantBanner       []string
		wantNotification string
	}{
		{
			name: "control characters and quotes in both", orgID: "org-a\x1b[2J\"; beep", typ: "BLOCKED\x07\"",
			wantBanner:       []string{`org=org-a\x1b[2J"; beep`, `type=BLOCKED\a"`},
			wantNotification: "org <invalid> raised e1 (<invalid>)",
		},
		{
			name: "a valid org_id and a TYPE escalate refuses", orgID: "org-a", typ: "TASK",
			wantBanner:       []string{"org=org-a", "type=TASK"},
			wantNotification: "org org-a raised e1 (<invalid>)",
		},
		{
			name: "an org_id that is not an identifier", orgID: "Org A", typ: "QUESTION",
			wantBanner:       []string{"org=Org A", "type=QUESTION"},
			wantNotification: "org <invalid> raised e1 (QUESTION)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newEscalateFixture(t)
			line, err := json.Marshal(InboxEvent{
				TS: inboxTestNow, ID: "e1", Event: InboxEventEscalated,
				OrgID: tc.orgID, Type: tc.typ, TaskID: "org-a", Body: "TYPE: BLOCKED\nTASK_ID: org-a\n\nstuck",
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(f.o.Inbox.Path(), append(line, '\n'), 0o644); err != nil {
				t.Fatal(err)
			}

			if err := f.o.NotifyInboxItem("e1", "", &f.banner); err != nil {
				t.Fatalf("NotifyInboxItem: %v", err)
			}
			banner := f.banner.String()
			for _, sub := range tc.wantBanner {
				if !strings.Contains(banner, sub) {
					t.Errorf("banner %q does not contain %q", banner, sub)
				}
			}
			for _, r := range banner[:len(banner)-1] {
				if r < 0x20 || r == 0x7f {
					t.Errorf("banner %q carries the raw control character %U", banner, r)
				}
			}
			if calls := f.notifications(); !slices.Equal(calls, []string{tc.wantNotification}) {
				t.Errorf("desktop notifications = %q; want [%q]", calls, tc.wantNotification)
			}
			if item := inboxItemByID(t, mustReadInbox(t, f.o.Inbox), "e1"); !item.Notified {
				t.Errorf("e1 = %+v; want notified", item)
			}
		})
	}
}

func TestInboxCommand(t *testing.T) {
	for _, tc := range []struct{ verb, id, stateDir, want string }{
		{"notify", "e3", "", "ralph org inbox notify e3"},
		{"show", "e3", "/repo/.harness/state/org", "ralph org inbox show e3 --state-dir '/repo/.harness/state/org'"},
		{"notify", "e12", "/tmp/it's here", `ralph org inbox notify e12 --state-dir '/tmp/it'\''s here'`},
	} {
		if got := InboxCommand(tc.verb, tc.id, tc.stateDir); got != tc.want {
			t.Errorf("InboxCommand(%q, %q, %q) = %q; want %q", tc.verb, tc.id, tc.stateDir, got, tc.want)
		}
	}
}

func TestPrintableInboxText(t *testing.T) {
	for _, tc := range []struct {
		in             string
		keepLineBreaks bool
		want           string
	}{
		{"a\n\tb\r\x1b", true, "a\n\tb\\r\\x1b"},
		{"a\n\tb\r\x1b", false, `a\n\tb\r\x1b`},
		{"org-a \"x\" \u0085 é", false, `org-a "x" \u0085 é`},
	} {
		if got := PrintableInboxText(tc.in, tc.keepLineBreaks); got != tc.want {
			t.Errorf("PrintableInboxText(%q, %v) = %q; want %q", tc.in, tc.keepLineBreaks, got, tc.want)
		}
	}
}

// Self-review L2: the recovery commands the banner and the errors name
// carry --state-dir when the caller passed HintStateDir (the CLI does when
// --state-dir was given), and stay bare otherwise.
func TestEscalate_HintStateDirReachesTheBannerAndTheRecoveryCommands(t *testing.T) {
	for _, hint := range []string{"", "/repo/.harness/state/org"} {
		t.Run("hint="+hint, func(t *testing.T) {
			f := newEscalateFixture(t)
			if err := os.Mkdir(f.o.EscalationsPath, 0o755); err != nil {
				t.Fatal(err)
			}
			res := f.o.Escalate(EscalateParams{OrgID: "org-a", Text: "TYPE: QUESTION\n\nwhich?", Banner: &f.banner, HintStateDir: hint})
			if res.Err == nil || !res.Recorded || res.Notified {
				t.Fatalf("Escalate = %+v; want e1 recorded, not notified, with an error", res)
			}
			notify := "`" + InboxCommand("notify", "e1", hint) + "`"
			if !strings.Contains(res.Err.Error(), "run "+notify+" to send it again") {
				t.Errorf("escalate error %q does not name %s", res.Err, notify)
			}
			if show := "read it with: " + InboxCommand("show", "e1", hint) + "\n"; !strings.Contains(f.banner.String(), show) {
				t.Errorf("banner %q does not end with %q", f.banner.String(), show)
			}

			f.banner.Reset()
			err := f.o.NotifyInboxItem("e1", hint, &f.banner)
			if err == nil || !strings.Contains(err.Error(), "run "+notify+" again") {
				t.Errorf("inbox notify error = %v; want it to name %s", err, notify)
			}
			if show := "read it with: " + InboxCommand("show", "e1", hint) + "\n"; !strings.Contains(f.banner.String(), show) {
				t.Errorf("inbox notify banner %q does not end with %q", f.banner.String(), show)
			}
			if hint == "" && strings.Contains(res.Err.Error()+err.Error()+f.banner.String(), "--state-dir") {
				t.Errorf("a bare call names --state-dir: %v / %v / %q", res.Err, err, f.banner.String())
			}
		})
	}
}

// Self-review L1: EscalateNotRecorded, the CLI's path when ralph.toml does
// not load, checks the message like Escalate and then tells the human the
// item was not recorded.
func TestEscalateNotRecorded_ChecksTheMessageThenAlertsTheHuman(t *testing.T) {
	cause := errors.New("org: load config: toml: line 1: expected '.' or '=', but got 'x' instead")

	t.Run("a refused message notifies no one", func(t *testing.T) {
		f := newEscalateFixture(t)
		res := f.o.EscalateNotRecorded(EscalateParams{OrgID: "org-a", Text: "TYPE: BLOCKED\n\nno task id", Banner: &f.banner}, cause)
		if !errors.Is(res.Err, protocol.ErrMissingTaskID) || res.Recorded {
			t.Fatalf("EscalateNotRecorded = %+v; want the TASK_ID refusal", res)
		}
		if f.banner.Len() != 0 || len(f.notifications()) != 0 {
			t.Errorf("a refusal notified: banner %q, calls %v", f.banner.String(), f.notifications())
		}
		if fileOrAbsent(t, f.o.Inbox.Path()) != "<absent>" || fileOrAbsent(t, f.o.EscalationsPath) != "<absent>" {
			t.Error("a refusal wrote inbox.jsonl or escalations.jsonl")
		}
	})

	t.Run("an accepted message reaches the human as not recorded", func(t *testing.T) {
		f := newEscalateFixture(t)
		res := f.o.EscalateNotRecorded(EscalateParams{OrgID: "org-a", Text: "TYPE: BLOCKED\nTASK_ID: org-a\n\nstuck", Banner: &f.banner}, cause)
		assertUnrecordedEscalation(t, f, res)
		if !errors.Is(res.Err, cause) {
			t.Errorf("error %v does not wrap the cause", res.Err)
		}
		if !strings.Contains(f.banner.String(), "org: load config") {
			t.Errorf("banner %q does not carry the cause", f.banner.String())
		}
		if got := fileOrAbsent(t, f.o.Inbox.Path()); got != "<absent>" {
			t.Errorf("inbox.jsonl = %q; want absent", got)
		}
		lines := escalationLines(t, f.o.EscalationsPath)
		if len(lines) != 1 || lines[0]["reason"] != "inbox_not_recorded" {
			t.Errorf("escalations.jsonl = %v; want one inbox_not_recorded line", lines)
		}
	})
}

// The watch and inbox lines share escalations.jsonl: a watch line keeps its
// alert_id and gets no inbox_id, an inbox line the other way round.
func TestEscalationRecord_WatchAndInboxLinesKeepTheirOwnID(t *testing.T) {
	watchLine, err := json.Marshal(escalationRecord{TS: inboxTestNow, OrgID: "org-a", AlertID: "org-a/seat-1/stall@1", Subject: "seat-1", Reason: "deadman_timeout"})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"ts":"2026-10-10T12:00:00Z","org_id":"org-a","alert_id":"org-a/seat-1/stall@1","subject":"seat-1","reason":"deadman_timeout"}`; string(watchLine) != want {
		t.Errorf("watch line = %s; want %s", watchLine, want)
	}
	inboxLine, err := json.Marshal(escalationRecord{TS: inboxTestNow, OrgID: "org-a", InboxID: "e1", Subject: "leader", Reason: "inbox_no_director"})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"ts":"2026-10-10T12:00:00Z","org_id":"org-a","inbox_id":"e1","subject":"leader","reason":"inbox_no_director"}`; string(inboxLine) != want {
		t.Errorf("inbox line = %s; want %s", inboxLine, want)
	}
}

func TestEscalationsPathIn(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "org")
	if got, want := EscalationsPathIn(dir), filepath.Join(dir, EscalationsRelName); got != want {
		t.Errorf("EscalationsPathIn = %q; want %q", got, want)
	}
}

// waitInboxOutcome is what one WaitInbox call returned, and how long it took.
type waitInboxOutcome struct {
	items   []InboxItem
	err     error
	elapsed time.Duration
}

// runWaitInbox calls o.WaitInbox(timeout) and fails the test when it has not
// returned within limit, so a wait that never ends fails instead of hanging
// the test binary.
func runWaitInbox(t *testing.T, o *Org, timeout, limit time.Duration) waitInboxOutcome {
	t.Helper()
	done := make(chan waitInboxOutcome, 1)
	start := time.Now()
	go func() {
		items, err := o.WaitInbox(timeout)
		done <- waitInboxOutcome{items: items, err: err, elapsed: time.Since(start)}
	}()
	select {
	case out := <-done:
		return out
	case <-time.After(limit):
		t.Fatalf("WaitInbox(%v) did not return within %v", timeout, limit)
		return waitInboxOutcome{}
	}
}

// escalateLater records a QUESTION from orgID in o's inbox after delay, from
// another goroutine, the way a leader escalates while a director waits. The
// test waits for that goroutine before it ends.
func escalateLater(t *testing.T, o *Org, orgID string, delay time.Duration) {
	t.Helper()
	var wg sync.WaitGroup
	t.Cleanup(wg.Wait)
	wg.Go(func() {
		time.Sleep(delay)
		if _, err := o.Inbox.Escalate(InboxEscalation{OrgID: orgID, Type: "QUESTION", Body: "TYPE: QUESTION\n\nwhich?"}); err != nil {
			t.Errorf("escalate from the other goroutine: %v", err)
		}
	})
}

func waitInboxItemIDs(items []InboxItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

// An open item already in the inbox returns at once, without a poll: the
// poll interval is an hour and there is no timeout. Acked and resolved items
// are left out.
func TestWaitInbox_ReturnsTheOpenItemsAtOnce(t *testing.T) {
	f := newEscalateFixture(t)
	f.o.InboxPollInterval = time.Hour
	for _, orgID := range []string{"org-a", "org-b", "org-c", "org-d"} {
		if res := f.escalate(orgID, "TYPE: QUESTION\n\nwhich?"); res.Err != nil {
			t.Fatal(res.Err)
		}
	}
	if _, err := f.o.Inbox.Ack("e2"); err != nil {
		t.Fatal(err)
	}
	if err := f.o.Inbox.Resolve("e3", "docs/reports/x.md"); err != nil {
		t.Fatal(err)
	}

	out := runWaitInbox(t, f.o, 0, 5*time.Second)
	if out.err != nil {
		t.Fatalf("WaitInbox: %v", out.err)
	}
	if got := waitInboxItemIDs(out.items); !slices.Equal(got, []string{"e1", "e4"}) {
		t.Errorf("WaitInbox items = %v; want the open ones, e1 and e4", got)
	}
	if out.items[1].OrgID != "org-d" || out.items[1].Type != "QUESTION" {
		t.Errorf("e4 = %+v; want org-d, QUESTION", out.items[1])
	}
}

func TestWaitInbox_ReturnsAnItemThatArrivesWhileWaiting(t *testing.T) {
	f := newEscalateFixture(t)
	f.o.InboxPollInterval = 10 * time.Millisecond
	escalateLater(t, f.o, "org-a", 200*time.Millisecond)

	out := runWaitInbox(t, f.o, 10*time.Second, 5*time.Second)
	if out.err != nil {
		t.Fatalf("WaitInbox: %v", out.err)
	}
	if got := waitInboxItemIDs(out.items); !slices.Equal(got, []string{"e1"}) {
		t.Errorf("WaitInbox items = %v; want e1", got)
	}
	if out.elapsed < 150*time.Millisecond {
		t.Errorf("WaitInbox returned after %v, before the item was escalated", out.elapsed)
	}
}

// With an hour between polls, the only reads are the first one and the one
// after the deadline: an item that arrives between them is still returned.
func TestWaitInbox_ReadsOnceMoreAtTheDeadline(t *testing.T) {
	f := newEscalateFixture(t)
	f.o.InboxPollInterval = time.Hour
	escalateLater(t, f.o, "org-a", 50*time.Millisecond)

	out := runWaitInbox(t, f.o, 300*time.Millisecond, 5*time.Second)
	if out.err != nil {
		t.Fatalf("WaitInbox: %v", out.err)
	}
	if got := waitInboxItemIDs(out.items); !slices.Equal(got, []string{"e1"}) {
		t.Errorf("WaitInbox items = %v; want e1, read after the deadline", got)
	}
}

func TestWaitInbox_TimesOutWithoutAnOpenItem(t *testing.T) {
	cases := []struct {
		name string
		seed func(t *testing.T, f *escalateFixture)
	}{
		{"no_inbox_file", func(*testing.T, *escalateFixture) {}},
		{"acked_item_only", func(t *testing.T, f *escalateFixture) {
			if res := f.escalate("org-a", "TYPE: QUESTION\n\nwhich?"); res.Err != nil {
				t.Fatal(res.Err)
			}
			if _, err := f.o.Inbox.Ack("e1"); err != nil {
				t.Fatal(err)
			}
		}},
		{"resolved_item_only", func(t *testing.T, f *escalateFixture) {
			if res := f.escalate("org-a", "TYPE: QUESTION\n\nwhich?"); res.Err != nil {
				t.Fatal(res.Err)
			}
			if err := f.o.Inbox.Resolve("e1", "docs/reports/x.md"); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newEscalateFixture(t)
			f.o.InboxPollInterval = 10 * time.Millisecond
			tc.seed(t, f)

			out := runWaitInbox(t, f.o, 300*time.Millisecond, 5*time.Second)
			if out.err == nil {
				t.Fatalf("WaitInbox = %v; want a timeout error", waitInboxItemIDs(out.items))
			}
			if want := "no open inbox item arrived within 300 ms"; !strings.Contains(out.err.Error(), want) {
				t.Errorf("error %q does not contain %q", out.err, want)
			}
			if out.items != nil {
				t.Errorf("a timeout returned items %v", waitInboxItemIDs(out.items))
			}
			if out.elapsed < 300*time.Millisecond {
				t.Errorf("WaitInbox timed out after %v; want at least the 300ms timeout", out.elapsed)
			}
		})
	}
}

// A read that fails returns at once, even without a timeout.
func TestWaitInbox_ReadFailureReturnsAtOnce(t *testing.T) {
	f := newEscalateFixture(t)
	f.o.InboxPollInterval = time.Hour
	// A directory where inbox.jsonl should be: the read fails.
	if err := os.MkdirAll(f.o.Inbox.Path(), 0o755); err != nil {
		t.Fatal(err)
	}
	out := runWaitInbox(t, f.o, 0, 5*time.Second)
	if out.err == nil || !strings.HasPrefix(out.err.Error(), "org: wait --inbox: ") {
		t.Fatalf("WaitInbox error = %v; want a read failure prefixed org: wait --inbox:", out.err)
	}

	if _, err := (&Org{}).WaitInbox(time.Millisecond); err == nil || !strings.Contains(err.Error(), "no inbox store") {
		t.Errorf("WaitInbox without an inbox store = %v; want an error", err)
	}
}
