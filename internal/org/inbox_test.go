package org

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// inboxTestNow is the fixed clock of newTestInboxStore.
const inboxTestNow = "2026-10-10T12:00:00Z"

func newTestInboxStore(t *testing.T) (*InboxStore, string) {
	t.Helper()
	dir := t.TempDir()
	s := NewInboxStore(dir)
	s.Now = func() time.Time { return time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC) }
	return s, dir
}

func writeInboxFile(t *testing.T, s *InboxStore, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(s.Path()), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(s.Path(), []byte(content), 0o644); err != nil {
		t.Fatalf("write inbox: %v", err)
	}
}

func readInboxFile(t *testing.T, s *InboxStore) string {
	t.Helper()
	raw, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatalf("read inbox: %v", err)
	}
	return string(raw)
}

func mustReadInbox(t *testing.T, s *InboxStore) InboxReadResult {
	t.Helper()
	res, err := s.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return res
}

func mustEscalate(t *testing.T, s *InboxStore, orgID string) string {
	t.Helper()
	id, err := s.Escalate(InboxEscalation{OrgID: orgID, Type: "QUESTION", Body: "TYPE: QUESTION\n\nwhich one?"})
	if err != nil {
		t.Fatalf("Escalate: %v", err)
	}
	return id
}

func inboxItemByID(t *testing.T, res InboxReadResult, id string) InboxItem {
	t.Helper()
	for _, item := range res.Items {
		if item.ID == id {
			return item
		}
	}
	t.Fatalf("no item %s in %+v", id, res.Items)
	return InboxItem{}
}

func inboxItemIDs(res InboxReadResult) []string {
	ids := make([]string, 0, len(res.Items))
	for _, item := range res.Items {
		ids = append(ids, item.ID)
	}
	return ids
}

func inboxEventKinds(item InboxItem) []string {
	kinds := make([]string, 0, len(item.Events))
	for _, ev := range item.Events {
		kinds = append(kinds, ev.Event)
	}
	return kinds
}

// escalatedLine is an escalated event line as a test fixture.
func escalatedLine(id, orgID string) string {
	return fmt.Sprintf(`{"ts":"2026-10-10T00:00:00Z","id":%q,"event":"escalated","org_id":%q,"type":"QUESTION","body":"TYPE: QUESTION\n\nwhich?"}`, id, orgID)
}

func TestInboxStore_LivesUnderTheGivenStateDir(t *testing.T) {
	state := filepath.Join(t.TempDir(), "ledger", "org")
	s := NewInboxStore(state)
	if want := filepath.Join(state, "inbox.jsonl"); s.Path() != want || InboxPathIn(state) != want {
		t.Fatalf("Path() = %q, InboxPathIn = %q; want %q", s.Path(), InboxPathIn(state), want)
	}
	if _, err := s.Escalate(InboxEscalation{OrgID: "auth-core", Type: "QUESTION", Body: "TYPE: QUESTION\n\nq"}); err != nil {
		t.Fatalf("Escalate into a state dir that does not exist yet: %v", err)
	}
	for _, name := range []string{"inbox.jsonl", "inbox.lock"} {
		if _, err := os.Stat(filepath.Join(state, name)); err != nil {
			t.Errorf("%s under the state dir: %v", name, err)
		}
	}
	// The inbox does not touch the manifest or its lock.
	for _, name := range []string{"manifest.jsonl", manifestLockFile} {
		if _, err := os.Stat(filepath.Join(state, name)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s should not exist after an escalate, stat err = %v", name, err)
		}
	}
}

func TestInboxRead_MissingStateDirIsAnEmptyInbox(t *testing.T) {
	state := filepath.Join(t.TempDir(), "absent", "org")
	res, err := NewInboxStore(state).Read()
	if err != nil {
		t.Fatalf("Read of a missing state dir: %v", err)
	}
	if len(res.Items) != 0 || res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Fatalf("want an empty result, got %+v", res)
	}
	if _, err := os.Stat(state); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Read created the state dir (stat err = %v)", err)
	}
}

// The lines are a ledger written before stage 8a: escalated events with no
// delivery and notified events with the old reason inbox_no_director.
func TestInboxRead_FoldsEventsIntoItemsInIDOrder(t *testing.T) {
	s, _ := newTestInboxStore(t)
	writeInboxFile(t, s, strings.Join([]string{
		`{"ts":"2026-10-10T01:00:00Z","id":"e10","event":"escalated","org_id":"billing","type":"BLOCKED","task_id":"t-10","body":"TYPE: BLOCKED\nTASK_ID: t-10\n\nstuck"}`,
		escalatedLine("e9", "auth-core"),
		`{"ts":"2026-10-10T01:00:01Z","id":"e10","event":"notified","reason":"inbox_no_director","osascript":"ok"}`,
		`{"ts":"2026-10-10T01:00:02Z","id":"e9","event":"acked"}`,
		`{"ts":"2026-10-10T01:00:03Z","id":"e10","event":"acked"}`,
		`{"ts":"2026-10-10T01:00:04Z","id":"e10","event":"notified","reason":"inbox_no_director","osascript":"failed: exit status 1"}`,
		`{"ts":"2026-10-10T01:00:05Z","id":"e10","event":"resolved","note":"docs/reports/x.md"}`,
		escalatedLine("e2", "auth-core"),
	}, "\n")+"\n")

	res := mustReadInbox(t, s)
	if res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Fatalf("corrupt=%d ignored=%d, want 0 and 0", res.CorruptLines, res.IgnoredEvents)
	}
	if got := strings.Join(inboxItemIDs(res), ","); got != "e2,e9,e10" {
		t.Fatalf("item order = %s, want e2,e9,e10 (numeric ID order)", got)
	}

	e2 := inboxItemByID(t, res, "e2")
	if e2.State != InboxStateOpen || e2.Notified || e2.EscalatedAt != "2026-10-10T00:00:00Z" {
		t.Errorf("e2 = %+v, want open, not notified", e2)
	}
	e9 := inboxItemByID(t, res, "e9")
	if e9.State != InboxStateAcked || e9.AckedAt != "2026-10-10T01:00:02Z" || e9.ResolvedAt != "" {
		t.Errorf("e9 = %+v, want acked at 01:00:02", e9)
	}
	e10 := inboxItemByID(t, res, "e10")
	if e10.State != InboxStateResolved || e10.OrgID != "billing" || e10.Type != "BLOCKED" || e10.TaskID != "t-10" ||
		e10.Body != "TYPE: BLOCKED\nTASK_ID: t-10\n\nstuck" || e10.AckedAt != "2026-10-10T01:00:03Z" ||
		e10.ResolvedAt != "2026-10-10T01:00:05Z" || e10.Note != "docs/reports/x.md" {
		t.Errorf("e10 = %+v", e10)
	}
	// notified marks the item without changing its state; NotifiedAt is
	// the latest one.
	if !e10.Notified || e10.NotifiedAt != "2026-10-10T01:00:04Z" {
		t.Errorf("e10 Notified=%v NotifiedAt=%q, want true and 01:00:04", e10.Notified, e10.NotifiedAt)
	}
	if got := strings.Join(inboxEventKinds(e10), ","); got != "escalated,notified,acked,notified,resolved" {
		t.Errorf("e10 events = %s", got)
	}
	if e10.Events[3].Osascript != "failed: exit status 1" {
		t.Errorf("e10 second notified osascript = %q", e10.Events[3].Osascript)
	}
	for _, item := range res.Items {
		if item.Delivery != InboxDeliveryNow {
			t.Errorf("%s Delivery = %q; an escalated line with no delivery folds as now", item.ID, item.Delivery)
		}
	}
}

// AC11: the escalated event carries the delivery the caller gave, now when
// it gave none, and the fold reads it back; a line with no delivery (written
// before stage 8a) folds as now, also in the item's JSON. So does a
// hand-edited delivery that is neither constant (self-review L9 of
// org-director-attach): the item, and --json, carry only the two values,
// the event keeps what was written, and nothing is counted as ignored.
func TestInboxEscalate_WritesAndFoldsTheDelivery(t *testing.T) {
	s, _ := newTestInboxStore(t)
	for _, delivery := range []string{"", InboxDeliveryNow, InboxDeliveryDeadline} {
		if _, err := s.Escalate(InboxEscalation{OrgID: "a", Type: "QUESTION", Body: "TYPE: QUESTION\n\nq", Delivery: delivery}); err != nil {
			t.Fatalf("Escalate with delivery %q: %v", delivery, err)
		}
	}
	raw := readInboxFile(t, s)
	handEdited := map[string]string{"e5": "deadline ", "e6": "x", "e7": "DEADLINE"}
	extra := escalatedLine("e4", "a") + "\n"
	for _, id := range []string{"e5", "e6", "e7"} {
		extra += strings.TrimSuffix(escalatedLine(id, "a"), "}") + fmt.Sprintf(`,"delivery":%q}`, handEdited[id]) + "\n"
	}
	writeInboxFile(t, s, raw+extra)

	lines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
	for i, want := range []string{`"delivery":"now"}`, `"delivery":"now"}`, `"delivery":"deadline"}`} {
		if !strings.HasSuffix(lines[i], want) {
			t.Errorf("line %d = %s; want it to end with %s", i+1, lines[i], want)
		}
	}
	res := mustReadInbox(t, s)
	if res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Errorf("corrupt=%d ignored=%d, want 0 and 0", res.CorruptLines, res.IgnoredEvents)
	}
	for id, want := range map[string]string{
		"e1": InboxDeliveryNow, "e2": InboxDeliveryNow, "e3": InboxDeliveryDeadline, "e4": InboxDeliveryNow,
		"e5": InboxDeliveryNow, "e6": InboxDeliveryNow, "e7": InboxDeliveryNow,
	} {
		if got := inboxItemByID(t, res, id).Delivery; got != want {
			t.Errorf("%s Delivery = %q; want %q", id, got, want)
		}
	}
	for id, written := range handEdited {
		if got := inboxItemByID(t, res, id).Events[0].Delivery; got != written {
			t.Errorf("%s escalated event delivery = %q; want %q as written", id, got, written)
		}
	}
	for _, id := range []string{"e4", "e6"} {
		data, err := json.Marshal(inboxItemByID(t, res, id))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), `"delivery":"now","state"`) {
			t.Errorf("%s JSON = %s; want the item's delivery now", id, data)
		}
	}
}

// A delivery that is not now or deadline is refused before anything is
// written.
func TestInboxEscalate_RefusesAnUnknownDelivery(t *testing.T) {
	for _, delivery := range []string{"later", "NOW", " deadline", "now "} {
		t.Run(delivery, func(t *testing.T) {
			s, dir := newTestInboxStore(t)
			_, err := s.Escalate(InboxEscalation{OrgID: "a", Type: "QUESTION", Body: "TYPE: QUESTION\n\nq", Delivery: delivery})
			if err == nil || !strings.Contains(err.Error(), "is not now or deadline") {
				t.Fatalf("Escalate with delivery %q err = %v; want a refusal", delivery, err)
			}
			for _, name := range []string{"inbox.jsonl", inboxLockFile} {
				if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("a refused escalate created %s (stat err = %v)", name, err)
				}
			}
		})
	}
}

// Plan 2026-10-23-watch-escalation-inbox, AC1: the escalated event of an
// item the cross-org watch records carries the ALERT's alert_id, and the
// fold reads it back on the item, on its escalated event, and in the item's
// JSON. An item recorded with no alert_id writes no field and folds with "".
func TestInboxEscalate_WritesAndFoldsTheAlertID(t *testing.T) {
	s, _ := newTestInboxStore(t)
	const alertID = "org-a/seat-1/stall@1760097600000000000"
	const body = "TYPE: ALERT\nORG_ID: org-a\nSEAT: seat-1\nCONDITION: stall\n\nwatchdog: escalated"
	if _, err := s.Escalate(InboxEscalation{OrgID: "org-a", Type: "ALERT", Body: body, Delivery: InboxDeliveryDeadline, AlertID: alertID}); err != nil {
		t.Fatalf("Escalate with an alert_id: %v", err)
	}
	mustEscalate(t, s, "org-a")

	lines := strings.Split(strings.TrimSuffix(readInboxFile(t, s), "\n"), "\n")
	want := []string{
		`{"ts":"2026-10-10T12:00:00Z","id":"e1","event":"escalated","org_id":"org-a","type":"ALERT","body":"TYPE: ALERT\nORG_ID: org-a\nSEAT: seat-1\nCONDITION: stall\n\nwatchdog: escalated","delivery":"deadline","alert_id":"org-a/seat-1/stall@1760097600000000000"}`,
		`{"ts":"2026-10-10T12:00:00Z","id":"e2","event":"escalated","org_id":"org-a","type":"QUESTION","body":"TYPE: QUESTION\n\nwhich one?","delivery":"now"}`,
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("inbox.jsonl =\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}

	res := mustReadInbox(t, s)
	if res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Errorf("corrupt=%d ignored=%d, want 0 and 0", res.CorruptLines, res.IgnoredEvents)
	}
	alert := inboxItemByID(t, res, "e1")
	if alert.AlertID != alertID || alert.Type != "ALERT" || alert.Body != body || alert.Delivery != InboxDeliveryDeadline || alert.State != InboxStateOpen {
		t.Errorf("e1 = %+v; want the ALERT item with alert_id %s, delivery deadline, open", alert, alertID)
	}
	if got := alert.Events[0].AlertID; got != alertID {
		t.Errorf("e1 escalated event alert_id = %q; want %q", got, alertID)
	}
	if question := inboxItemByID(t, res, "e2"); question.AlertID != "" || question.Events[0].AlertID != "" {
		t.Errorf("e2 = %+v; want no alert_id on an item recorded without one", question)
	}
	data, err := json.Marshal(alert)
	if err != nil {
		t.Fatal(err)
	}
	if want := `"type":"ALERT","alert_id":"` + alertID + `","body"`; !strings.Contains(string(data), want) {
		t.Errorf("e1 JSON = %s; want it to contain %s", data, want)
	}
	if data, err = json.Marshal(inboxItemByID(t, res, "e2")); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"alert_id"`) {
		t.Errorf("e2 JSON = %s; want no alert_id", data)
	}
}

// Plan 2026-10-23-watch-escalation-inbox, AC1: an escalated line with no
// alert_id (every line written before that plan) folds with "" and is not
// counted as damage, and an alert_id on a later event (a hand edit) does not
// change the item's: only the escalated event names it.
func TestInboxRead_AlertIDComesOnlyFromTheEscalatedEvent(t *testing.T) {
	s, _ := newTestInboxStore(t)
	writeInboxFile(t, s, strings.Join([]string{
		escalatedLine("e1", "org-a"),
		`{"ts":"2026-10-10T00:00:01Z","id":"e1","event":"notified","reason":"inbox_no_watch","osascript":"ok","alert_id":"org-a/seat-1/stall@1"}`,
		`{"ts":"2026-10-10T00:00:02Z","id":"e2","event":"escalated","org_id":"org-a","type":"ALERT","body":"TYPE: ALERT\n\nx","delivery":"now","alert_id":"org-a/leader/liveness@2"}`,
		`{"ts":"2026-10-10T00:00:03Z","id":"e2","event":"acked","alert_id":"org-a/seat-9/stall@3"}`,
	}, "\n")+"\n")

	res := mustReadInbox(t, s)
	if res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Errorf("corrupt=%d ignored=%d, want 0 and 0", res.CorruptLines, res.IgnoredEvents)
	}
	if e1 := inboxItemByID(t, res, "e1"); e1.AlertID != "" || !e1.Notified {
		t.Errorf("e1 = %+v; want no alert_id (the escalated line has none) and notified", e1)
	}
	if e2 := inboxItemByID(t, res, "e2"); e2.AlertID != "org-a/leader/liveness@2" || e2.State != InboxStateAcked {
		t.Errorf("e2 = %+v; want the escalated event's alert_id and acked", e2)
	}
}

func TestInboxRead_SkipsAndCountsLinesItCannotUse(t *testing.T) {
	s, _ := newTestInboxStore(t)
	writeInboxFile(t, s, strings.Join([]string{
		`not json`,
		`{"ts":"t","id":"e3",`,
		`[1,2]`,
		``,
		`   `,
		escalatedLine("e1", "auth-core"),
		// A second escalated for e1, an unknown kind, an event for an ID
		// with no escalated, and IDs not of the form e<N>: all ignored.
		`{"ts":"t","id":"e1","event":"escalated","org_id":"other","type":"RESULT","body":"second"}`,
		`{"ts":"t","id":"e1","event":"expired"}`,
		`{"ts":"t","id":"e7","event":"acked"}`,
		escalatedLine("e01", "auth-core"),
		escalatedLine("e0", "auth-core"),
		escalatedLine("ex", "auth-core"),
		escalatedLine("", "auth-core"),
		escalatedLine("E4", "auth-core"),
		// resolved applies; then acked on a resolved item and a second
		// resolved are ignored, and notified still applies.
		`{"ts":"t1","id":"e1","event":"resolved","note":"n1"}`,
		`{"ts":"t2","id":"e1","event":"acked"}`,
		`{"ts":"t3","id":"e1","event":"resolved","note":"n2"}`,
		`{"ts":"t4","id":"e1","event":"notified","reason":"inbox_no_director","osascript":"skipped"}`,
		// The second acked on e2 is ignored.
		escalatedLine("e2", "auth-core"),
		`{"ts":"t5","id":"e2","event":"acked"}`,
		`{"ts":"t6","id":"e2","event":"acked"}`,
	}, "\n")+"\n")

	res := mustReadInbox(t, s)
	if res.CorruptLines != 3 {
		t.Errorf("CorruptLines = %d, want 3 (blank lines are not counted)", res.CorruptLines)
	}
	if res.IgnoredEvents != 11 {
		t.Errorf("IgnoredEvents = %d, want 11", res.IgnoredEvents)
	}
	if got := strings.Join(inboxItemIDs(res), ","); got != "e1,e2" {
		t.Fatalf("items = %s, want e1,e2", got)
	}
	e1 := inboxItemByID(t, res, "e1")
	if e1.OrgID != "auth-core" || e1.Type != "QUESTION" || e1.State != InboxStateResolved || e1.Note != "n1" ||
		e1.ResolvedAt != "t1" || e1.AckedAt != "" || !e1.Notified {
		t.Errorf("e1 = %+v, want the first escalated, resolved with n1, notified", e1)
	}
	if got := strings.Join(inboxEventKinds(e1), ","); got != "escalated,resolved,notified" {
		t.Errorf("e1 events = %s, want only the applied ones", got)
	}
	e2 := inboxItemByID(t, res, "e2")
	if e2.State != InboxStateAcked || e2.AckedAt != "t5" || len(e2.Events) != 2 {
		t.Errorf("e2 = %+v, want acked at t5 with 2 events", e2)
	}
}

func TestInboxEscalate_StartsAtE1AndWritesOneLinePerEvent(t *testing.T) {
	s, _ := newTestInboxStore(t)
	if id := mustEscalate(t, s, "auth-core"); id != "e1" {
		t.Fatalf("first ID = %s, want e1", id)
	}
	id, err := s.Escalate(InboxEscalation{OrgID: "auth-core", Type: "BLOCKED", TaskID: "auth-core", Body: "TYPE: BLOCKED\nTASK_ID: auth-core\n\nstuck"})
	if err != nil || id != "e2" {
		t.Fatalf("second Escalate = %q, %v; want e2", id, err)
	}

	lines := strings.Split(strings.TrimSuffix(readInboxFile(t, s), "\n"), "\n")
	want := []string{
		`{"ts":"2026-10-10T12:00:00Z","id":"e1","event":"escalated","org_id":"auth-core","type":"QUESTION","body":"TYPE: QUESTION\n\nwhich one?","delivery":"now"}`,
		`{"ts":"2026-10-10T12:00:00Z","id":"e2","event":"escalated","org_id":"auth-core","type":"BLOCKED","task_id":"auth-core","body":"TYPE: BLOCKED\nTASK_ID: auth-core\n\nstuck","delivery":"now"}`,
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("inbox.jsonl =\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}

	res := mustReadInbox(t, s)
	e1 := inboxItemByID(t, res, "e1")
	if e1.State != InboxStateOpen || e1.EscalatedAt != inboxTestNow || e1.TaskID != "" || e1.Notified {
		t.Errorf("e1 = %+v", e1)
	}
}

func TestInboxEvents_LineShapePerKind(t *testing.T) {
	s, _ := newTestInboxStore(t)
	mustEscalate(t, s, "auth-core")
	if err := s.AppendNotified("e1", InboxReasonNoWatch, "ok"); err != nil {
		t.Fatalf("AppendNotified: %v", err)
	}
	if _, err := s.Ack("e1"); err != nil {
		t.Fatalf("Ack: %v", err)
	}
	if err := s.Resolve("e1", "docs/reports/x.md"); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	lines := strings.Split(strings.TrimSuffix(readInboxFile(t, s), "\n"), "\n")
	want := []string{
		`{"ts":"2026-10-10T12:00:00Z","id":"e1","event":"escalated","org_id":"auth-core","type":"QUESTION","body":"TYPE: QUESTION\n\nwhich one?","delivery":"now"}`,
		`{"ts":"2026-10-10T12:00:00Z","id":"e1","event":"notified","reason":"inbox_no_watch","osascript":"ok"}`,
		`{"ts":"2026-10-10T12:00:00Z","id":"e1","event":"acked"}`,
		`{"ts":"2026-10-10T12:00:00Z","id":"e1","event":"resolved","note":"docs/reports/x.md"}`,
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("inbox.jsonl =\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
}

func TestInboxEscalate_NeverReusesTheIDOfADamagedLine(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		wantID  string
		wantIDs string
	}{
		{
			name:    "damaged middle line e2 between e1 and e3",
			content: escalatedLine("e1", "a") + "\n" + `{"ts":"t","id":"e2","event":"escal` + "\n" + escalatedLine("e3", "a") + "\n",
			wantID:  "e4", wantIDs: "e1,e3,e4",
		},
		{
			name:    "the damaged line holds the largest ID, with spaces around the colon",
			content: escalatedLine("e1", "a") + "\n" + `{"ts":"t", "id" : "e8", "event":` + "\n" + escalatedLine("e3", "a") + "\n",
			wantID:  "e9", wantIDs: "e1,e3,e9",
		},
		{
			name:    "an ID too large for int64 in a damaged line is skipped",
			content: `{"id":"e99999999999999999999999",` + "\n" + escalatedLine("e2", "a") + "\n",
			wantID:  "e3", wantIDs: "e2,e3",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newTestInboxStore(t)
			writeInboxFile(t, s, tc.content)
			if id := mustEscalate(t, s, "auth-core"); id != tc.wantID {
				t.Fatalf("Escalate = %s, want %s", id, tc.wantID)
			}
			res := mustReadInbox(t, s)
			if got := strings.Join(inboxItemIDs(res), ","); got != tc.wantIDs || res.CorruptLines != 1 {
				t.Fatalf("items = %s (corrupt %d), want %s (corrupt 1)", got, res.CorruptLines, tc.wantIDs)
			}
		})
	}
}

func TestInboxEscalate_TornLastLineStaysSeparateAndTheNewEventIsReadable(t *testing.T) {
	for _, tc := range []struct {
		name   string
		torn   string
		wantID string
	}{
		{name: "cut after the id", torn: `{"ts":"2026-10-10T00:00:00Z","id":"e5","event":"escalated","org_id":"a`, wantID: "e6"},
		{name: "cut inside the id", torn: `{"ts":"2026-10-10T00:00:00Z","id":"e12`, wantID: "e13"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newTestInboxStore(t)
			writeInboxFile(t, s, escalatedLine("e4", "a")+"\n"+tc.torn)
			if id := mustEscalate(t, s, "auth-core"); id != tc.wantID {
				t.Fatalf("Escalate = %s, want %s", id, tc.wantID)
			}
			if _, err := s.Ack(tc.wantID); err != nil {
				t.Fatalf("Ack of the new item: %v", err)
			}
			lines := strings.Split(readInboxFile(t, s), "\n")
			if len(lines) != 5 || lines[1] != tc.torn || lines[4] != "" {
				t.Fatalf("inbox.jsonl lines = %q, want e4, the torn line on its own, the new escalated, acked, and a final newline", lines)
			}
			res := mustReadInbox(t, s)
			if got := strings.Join(inboxItemIDs(res), ","); got != "e4,"+tc.wantID || res.CorruptLines != 1 || res.IgnoredEvents != 0 {
				t.Fatalf("items = %s (corrupt %d, ignored %d), want e4,%s (corrupt 1, ignored 0)", got, res.CorruptLines, res.IgnoredEvents, tc.wantID)
			}
			if item := inboxItemByID(t, res, tc.wantID); item.State != InboxStateAcked {
				t.Fatalf("new item state = %s, want acked", item.State)
			}
		})
	}
}

func TestInboxEscalate_CountsAnIDWrittenWithJSONEscapes(t *testing.T) {
	s, _ := newTestInboxStore(t)
	// "%u00659" becomes a JSON escape for "e" followed by 9: the decoded ID
	// is e9, which the raw-text scan alone would not find.
	line := `{"ts":"t","id":"%u00659","event":"escalated","org_id":"a","type":"QUESTION","body":"b"}`
	writeInboxFile(t, s, strings.ReplaceAll(line, "%u", "\\"+"u")+"\n")
	if res := mustReadInbox(t, s); strings.Join(inboxItemIDs(res), ",") != "e9" {
		t.Fatalf("items = %v, want e9", inboxItemIDs(res))
	}
	if id := mustEscalate(t, s, "auth-core"); id != "e10" {
		t.Fatalf("Escalate = %s, want e10", id)
	}
}

func TestInboxEscalate_FailsWhenNoIDIsLeft(t *testing.T) {
	s, _ := newTestInboxStore(t)
	content := escalatedLine("e9223372036854775807", "a") + "\n"
	writeInboxFile(t, s, content)
	_, err := s.Escalate(InboxEscalation{OrgID: "a", Type: "QUESTION", Body: "b"})
	if err == nil || !strings.Contains(err.Error(), "no item ID is left") {
		t.Fatalf("Escalate err = %v, want no item ID is left", err)
	}
	if got := readInboxFile(t, s); got != content {
		t.Fatalf("inbox.jsonl changed to %q", got)
	}
}

func TestInboxEscalate_RequiresOrgIDTypeAndBody(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   InboxEscalation
		want string
	}{
		{"org_id", InboxEscalation{Type: "QUESTION", Body: "b"}, "non-empty org_id"},
		{"type", InboxEscalation{OrgID: "a", Type: " ", Body: "b"}, "non-empty type"},
		{"body", InboxEscalation{OrgID: "a", Type: "QUESTION"}, "non-empty body"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newTestInboxStore(t)
			if _, err := s.Escalate(tc.in); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Escalate err = %v, want %q", err, tc.want)
			}
			if _, err := os.Stat(s.Path()); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("a rejected escalate wrote the inbox (stat err = %v)", err)
			}
		})
	}
}

func TestInboxEscalate_ConcurrentCallsGetDistinctIDs(t *testing.T) {
	s, _ := newTestInboxStore(t)
	const n = 20
	ids := make([]string, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids[i], errs[i] = s.Escalate(InboxEscalation{OrgID: fmt.Sprintf("org-%d", i), Type: "QUESTION", Body: "TYPE: QUESTION\n\nq"})
		}()
	}
	wg.Wait()

	seen := map[string]bool{}
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("Escalate %d: %v", i, errs[i])
		}
		if seen[ids[i]] {
			t.Fatalf("ID %s was handed out twice: %v", ids[i], ids)
		}
		seen[ids[i]] = true
	}
	for i := 1; i <= n; i++ {
		if !seen[fmt.Sprintf("e%d", i)] {
			t.Fatalf("IDs %v, want e1..e%d", ids, n)
		}
	}
	res := mustReadInbox(t, s)
	if len(res.Items) != n || res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Fatalf("Read: %d items, corrupt %d, ignored %d; want %d, 0, 0", len(res.Items), res.CorruptLines, res.IgnoredEvents, n)
	}
}

func TestInboxAckAndResolve_TransitionTable(t *testing.T) {
	setups := map[string]func(t *testing.T, s *InboxStore){
		InboxStateOpen: func(t *testing.T, s *InboxStore) {},
		InboxStateAcked: func(t *testing.T, s *InboxStore) {
			if _, err := s.Ack("e1"); err != nil {
				t.Fatalf("setup Ack: %v", err)
			}
		},
		InboxStateResolved: func(t *testing.T, s *InboxStore) {
			if err := s.Resolve("e1", "first note"); err != nil {
				t.Fatalf("setup Resolve: %v", err)
			}
		},
	}
	for _, tc := range []struct {
		from        string
		verb        string
		wantChanged bool
		wantErr     error
		wantState   string
		wantNote    string
		wantAdded   int
	}{
		{from: InboxStateOpen, verb: "ack", wantChanged: true, wantState: InboxStateAcked, wantAdded: 1},
		{from: InboxStateAcked, verb: "ack", wantState: InboxStateAcked},
		{from: InboxStateResolved, verb: "ack", wantErr: ErrInboxResolved, wantState: InboxStateResolved, wantNote: "first note"},
		{from: InboxStateOpen, verb: "resolve", wantState: InboxStateResolved, wantNote: "docs/reports/r.md", wantAdded: 1},
		{from: InboxStateAcked, verb: "resolve", wantState: InboxStateResolved, wantNote: "docs/reports/r.md", wantAdded: 1},
		{from: InboxStateResolved, verb: "resolve", wantErr: ErrInboxResolved, wantState: InboxStateResolved, wantNote: "first note"},
	} {
		t.Run(tc.verb+" on "+tc.from, func(t *testing.T) {
			s, _ := newTestInboxStore(t)
			mustEscalate(t, s, "auth-core")
			setups[tc.from](t, s)
			before := len(inboxItemByID(t, mustReadInbox(t, s), "e1").Events)

			var changed bool
			var err error
			if tc.verb == "ack" {
				changed, err = s.Ack("e1")
			} else {
				err = s.Resolve("e1", "docs/reports/r.md")
			}
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("%s err = %v, want %v", tc.verb, err, tc.wantErr)
				}
			} else if err != nil {
				t.Fatalf("%s: %v", tc.verb, err)
			}
			if changed != tc.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tc.wantChanged)
			}

			res := mustReadInbox(t, s)
			item := inboxItemByID(t, res, "e1")
			if item.State != tc.wantState || item.Note != tc.wantNote {
				t.Errorf("state = %s note = %q, want %s %q", item.State, item.Note, tc.wantState, tc.wantNote)
			}
			if added := len(item.Events) - before; added != tc.wantAdded {
				t.Errorf("%d events added, want %d", added, tc.wantAdded)
			}
			if res.IgnoredEvents != 0 {
				t.Errorf("IgnoredEvents = %d: a refused transition was written", res.IgnoredEvents)
			}
			if tc.from == InboxStateAcked && tc.verb == "resolve" && item.AckedAt == "" {
				t.Errorf("resolve dropped AckedAt")
			}
		})
	}
}

func TestInboxTransitions_UnknownIDIsErrInboxUnknownID(t *testing.T) {
	s, _ := newTestInboxStore(t)
	mustEscalate(t, s, "auth-core")
	before := readInboxFile(t, s)
	for _, tc := range []struct {
		name string
		call func() error
	}{
		{"ack e2", func() error { _, err := s.Ack("e2"); return err }},
		{"ack e01", func() error { _, err := s.Ack("e01"); return err }},
		{"ack E1", func() error { _, err := s.Ack("E1"); return err }},
		{"resolve e2", func() error { return s.Resolve("e2", "note") }},
		{"notify e2", func() error { return s.AppendNotified("e2", InboxReasonManualNotify, "ok") }},
	} {
		if err := tc.call(); !errors.Is(err, ErrInboxUnknownID) {
			t.Errorf("%s err = %v, want ErrInboxUnknownID", tc.name, err)
		}
	}
	if got := readInboxFile(t, s); got != before {
		t.Fatalf("inbox.jsonl changed to %q", got)
	}

	// A missing inbox has no items; asking for one creates nothing.
	state := filepath.Join(t.TempDir(), "absent")
	empty := NewInboxStore(state)
	if _, err := empty.Ack("e1"); !errors.Is(err, ErrInboxUnknownID) || !strings.Contains(err.Error(), "org: inbox: ack e1: ") {
		t.Fatalf("Ack on a missing inbox err = %v, want org: inbox: ack e1: ... ErrInboxUnknownID", err)
	}
	if _, err := os.Stat(state); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Ack on a missing inbox created the state dir (stat err = %v)", err)
	}
}

func TestValidateInboxNote(t *testing.T) {
	for _, tc := range []struct {
		name string
		note string
		want string // "" means valid
	}{
		{"a pointer", "docs/reports/verify-2026-10-10-x.md", ""},
		{"500 characters of multibyte text", strings.Repeat("あ", 500), ""},
		{"empty", "", "is required"},
		{"blank", "  \t ", "is required"},
		{"LF", "line one\nline two", "single line"},
		{"CR", "line one\rline two", "single line"},
		{"Unicode line separator", "line one" + string(rune(0x2028)) + "line two", "single line"},
		{"501 ASCII characters", strings.Repeat("a", 501), "is 501 characters; at most 500"},
		{"501 multibyte characters", strings.Repeat("あ", 501), "is 501 characters; at most 500"},
		{"tab", "a\tb", "control character (U+0009)"},
		{"escape", "a" + string(rune(0x1b)) + "[31m", "control character (U+001B)"},
		{"DEL", "a" + string(rune(0x7f)), "control character (U+007F)"},
		{"C1 control", "a" + string(rune(0x9b)), "control character (U+009B)"},
		{"invalid UTF-8", "a\xffb", "not valid UTF-8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateInboxNote(tc.note)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("ValidateInboxNote: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestInboxResolve_RejectedNoteWritesNothing(t *testing.T) {
	s, _ := newTestInboxStore(t)
	mustEscalate(t, s, "auth-core")
	before := readInboxFile(t, s)
	for _, note := range []string{"", "a\nb", strings.Repeat("x", 501), "a\x07b"} {
		if err := s.Resolve("e1", note); err == nil {
			t.Errorf("Resolve(%q) succeeded", note)
		}
	}
	if got := readInboxFile(t, s); got != before {
		t.Fatalf("inbox.jsonl changed to %q", got)
	}
	if item := inboxItemByID(t, mustReadInbox(t, s), "e1"); item.State != InboxStateOpen {
		t.Fatalf("state = %s, want open", item.State)
	}
}

func TestInboxAppendNotified_MarksTheItemWithoutChangingState(t *testing.T) {
	s, _ := newTestInboxStore(t)
	mustEscalate(t, s, "auth-core")
	if err := s.AppendNotified("e1", " ", "ok"); err == nil || !strings.Contains(err.Error(), "needs a reason") {
		t.Fatalf("AppendNotified with a blank reason err = %v", err)
	}
	if err := s.AppendNotified("e1", InboxReasonDirectorDisabled, "failed: exit status 1"); err != nil {
		t.Fatalf("AppendNotified: %v", err)
	}
	item := inboxItemByID(t, mustReadInbox(t, s), "e1")
	if item.State != InboxStateOpen || !item.Notified || item.NotifiedAt != inboxTestNow {
		t.Fatalf("item = %+v, want open and notified", item)
	}
	// A resolved item still takes a notified event (a resend after the
	// fact is a record, not a transition).
	if err := s.Resolve("e1", "done"); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if err := s.AppendNotified("e1", InboxReasonManualNotify, "skipped"); err != nil {
		t.Fatalf("AppendNotified on a resolved item: %v", err)
	}
	item = inboxItemByID(t, mustReadInbox(t, s), "e1")
	if item.State != InboxStateResolved || strings.Join(inboxEventKinds(item), ",") != "escalated,notified,resolved,notified" {
		t.Fatalf("item = %+v", item)
	}
}

// wokenLine is a woken event line as a test fixture.
func wokenLine(ts, id, directorID, via string) string {
	return fmt.Sprintf(`{"ts":%q,"id":%q,"event":"woken","director_id":%q,"via":%q}`, ts, id, directorID, via)
}

// AC8 of plan 2026-10-18-org-director-wake: a woken event marks an item in
// any state (open, acked, resolved) without changing the state. WokenAt is
// the latest woken event's ts and every woken event stays in Events; an
// acked after a woken still moves the item from open; a woken never sets
// Notified; a woken for an ID with no escalated event creates no item and is
// counted as ignored.
func TestInboxRead_WokenMarksTheItemInAnyStateWithoutChangingIt(t *testing.T) {
	s, _ := newTestInboxStore(t)
	writeInboxFile(t, s, strings.Join([]string{
		escalatedLine("e1", "org-a"),
		wokenLine("2026-10-10T01:00:00Z", "e1", "d1", InboxWokenViaWakeCmd),
		wokenLine("2026-10-10T01:00:30Z", "e1", "d2", InboxWokenViaHerdr),
		escalatedLine("e2", "org-a"),
		`{"ts":"2026-10-10T01:01:00Z","id":"e2","event":"acked"}`,
		wokenLine("2026-10-10T01:02:00Z", "e2", "d1", InboxWokenViaHerdr),
		escalatedLine("e3", "org-b"),
		`{"ts":"2026-10-10T01:03:00Z","id":"e3","event":"resolved","note":"docs/reports/x.md"}`,
		wokenLine("2026-10-10T01:04:00Z", "e3", "d1", InboxWokenViaWakeCmd),
		escalatedLine("e4", "org-b"),
		wokenLine("2026-10-10T01:05:00Z", "e4", "d1", InboxWokenViaWakeCmd),
		`{"ts":"2026-10-10T01:06:00Z","id":"e4","event":"acked"}`,
		wokenLine("2026-10-10T01:07:00Z", "e9", "d1", InboxWokenViaWakeCmd),
	}, "\n")+"\n")

	res := mustReadInbox(t, s)
	if res.CorruptLines != 0 || res.IgnoredEvents != 1 {
		t.Errorf("corrupt=%d ignored=%d, want 0 and 1 (the woken for e9, which has no escalated)", res.CorruptLines, res.IgnoredEvents)
	}
	if got := strings.Join(inboxItemIDs(res), ","); got != "e1,e2,e3,e4" {
		t.Fatalf("items = %s, want e1,e2,e3,e4 (a woken creates no item)", got)
	}
	for _, tc := range []struct {
		id, state, wokenAt, kinds string
	}{
		{"e1", InboxStateOpen, "2026-10-10T01:00:30Z", "escalated,woken,woken"},
		{"e2", InboxStateAcked, "2026-10-10T01:02:00Z", "escalated,acked,woken"},
		{"e3", InboxStateResolved, "2026-10-10T01:04:00Z", "escalated,resolved,woken"},
		{"e4", InboxStateAcked, "2026-10-10T01:05:00Z", "escalated,woken,acked"},
	} {
		item := inboxItemByID(t, res, tc.id)
		if item.State != tc.state || !item.Woken || item.WokenAt != tc.wokenAt || item.Notified || item.NotifiedAt != "" {
			t.Errorf("%s = %+v; want state %s, woken at %s, not notified", tc.id, item, tc.state, tc.wokenAt)
		}
		if got := strings.Join(inboxEventKinds(item), ","); got != tc.kinds {
			t.Errorf("%s events = %s; want %s", tc.id, got, tc.kinds)
		}
	}
	e1 := inboxItemByID(t, res, "e1")
	for i, want := range []InboxEvent{
		{TS: "2026-10-10T01:00:00Z", ID: "e1", Event: InboxEventWoken, DirectorID: "d1", Via: InboxWokenViaWakeCmd},
		{TS: "2026-10-10T01:00:30Z", ID: "e1", Event: InboxEventWoken, DirectorID: "d2", Via: InboxWokenViaHerdr},
	} {
		if got := e1.Events[i+1]; got != want {
			t.Errorf("e1 event %d = %+v; want %+v", i+1, got, want)
		}
	}
	if e3 := inboxItemByID(t, res, "e3"); e3.ResolvedAt != "2026-10-10T01:03:00Z" || e3.Note != "docs/reports/x.md" {
		t.Errorf("e3 = %+v; want resolved at 01:03:00 with its note", e3)
	}
	if e4 := inboxItemByID(t, res, "e4"); e4.AckedAt != "2026-10-10T01:06:00Z" {
		t.Errorf("e4 AckedAt = %q; want the acked after the woken applied", e4.AckedAt)
	}
}

// AC8: lines with no woken event (written before stage 8b, one escalated
// line from before stage 8a too) fold exactly as before: every field the
// same, Woken false and WokenAt empty, and the item's JSON carries
// "woken":false and no woken_at, with no director_id or via in its events.
func TestInboxRead_LinesWithoutWokenFoldAsBefore(t *testing.T) {
	s, _ := newTestInboxStore(t)
	writeInboxFile(t, s, strings.Join([]string{
		escalatedLine("e1", "org-a"),
		`{"ts":"2026-10-10T00:00:01Z","id":"e1","event":"notified","reason":"inbox_no_director","osascript":"ok"}`,
		`{"ts":"2026-10-10T00:00:02Z","id":"e1","event":"acked"}`,
		`{"ts":"2026-10-10T00:00:03Z","id":"e2","event":"escalated","org_id":"org-b","type":"BLOCKED","task_id":"org-b","body":"TYPE: BLOCKED\nTASK_ID: org-b\n\nstuck","delivery":"deadline"}`,
		`{"ts":"2026-10-10T00:00:04Z","id":"e2","event":"resolved","note":"docs/reports/x.md"}`,
	}, "\n")+"\n")

	res := mustReadInbox(t, s)
	want := []InboxItem{
		{
			ID: "e1", OrgID: "org-a", Type: "QUESTION", Body: "TYPE: QUESTION\n\nwhich?", Delivery: InboxDeliveryNow,
			State: InboxStateAcked, EscalatedAt: "2026-10-10T00:00:00Z", AckedAt: "2026-10-10T00:00:02Z",
			Notified: true, NotifiedAt: "2026-10-10T00:00:01Z",
			Events: []InboxEvent{
				{TS: "2026-10-10T00:00:00Z", ID: "e1", Event: InboxEventEscalated, OrgID: "org-a", Type: "QUESTION", Body: "TYPE: QUESTION\n\nwhich?"},
				{TS: "2026-10-10T00:00:01Z", ID: "e1", Event: InboxEventNotified, Reason: "inbox_no_director", Osascript: "ok"},
				{TS: "2026-10-10T00:00:02Z", ID: "e1", Event: InboxEventAcked},
			},
		},
		{
			ID: "e2", OrgID: "org-b", Type: "BLOCKED", TaskID: "org-b", Body: "TYPE: BLOCKED\nTASK_ID: org-b\n\nstuck", Delivery: InboxDeliveryDeadline,
			State: InboxStateResolved, EscalatedAt: "2026-10-10T00:00:03Z", ResolvedAt: "2026-10-10T00:00:04Z", Note: "docs/reports/x.md",
			Events: []InboxEvent{
				{TS: "2026-10-10T00:00:03Z", ID: "e2", Event: InboxEventEscalated, OrgID: "org-b", Type: "BLOCKED", TaskID: "org-b",
					Body: "TYPE: BLOCKED\nTASK_ID: org-b\n\nstuck", Delivery: InboxDeliveryDeadline},
				{TS: "2026-10-10T00:00:04Z", ID: "e2", Event: InboxEventResolved, Note: "docs/reports/x.md"},
			},
		},
	}
	if res.CorruptLines != 0 || res.IgnoredEvents != 0 || !reflect.DeepEqual(res.Items, want) {
		t.Fatalf("Read = %+v\nwant items %+v with no damage", res, want)
	}
	data, err := json.Marshal(res.Items[0])
	if err != nil {
		t.Fatal(err)
	}
	const wantJSON = `{"id":"e1","org_id":"org-a","type":"QUESTION","body":"TYPE: QUESTION\n\nwhich?","delivery":"now","state":"acked",` +
		`"escalated_at":"2026-10-10T00:00:00Z","acked_at":"2026-10-10T00:00:02Z","notified":true,"notified_at":"2026-10-10T00:00:01Z","woken":false,` +
		`"events":[{"ts":"2026-10-10T00:00:00Z","id":"e1","event":"escalated","org_id":"org-a","type":"QUESTION","body":"TYPE: QUESTION\n\nwhich?"},` +
		`{"ts":"2026-10-10T00:00:01Z","id":"e1","event":"notified","reason":"inbox_no_director","osascript":"ok"},` +
		`{"ts":"2026-10-10T00:00:02Z","id":"e1","event":"acked"}]}`
	if string(data) != wantJSON {
		t.Errorf("e1 JSON =\n%s\nwant\n%s", data, wantJSON)
	}
}

// AC8 (and the record AC5 needs): AppendWoken appends one woken line per ID,
// in the order given, with director_id, via, and one ts for all, after the
// lines already there. Items in every state take it, and no state changes; a
// later woken moves WokenAt.
func TestInboxAppendWoken_WritesOneLinePerIDWithOneTS(t *testing.T) {
	s, _ := newTestInboxStore(t)
	for _, orgID := range []string{"org-a", "org-b", "org-c"} {
		mustEscalate(t, s, orgID)
	}
	if _, err := s.Ack("e2"); err != nil {
		t.Fatal(err)
	}
	if err := s.Resolve("e3", "docs/reports/x.md"); err != nil {
		t.Fatal(err)
	}
	before := readInboxFile(t, s)
	s.Now = func() time.Time { return time.Date(2026, 10, 10, 12, 5, 0, 0, time.UTC) }

	if err := s.AppendWoken([]string{"e3", "e1", "e2"}, "d2", InboxWokenViaWakeCmd); err != nil {
		t.Fatalf("AppendWoken: %v", err)
	}
	added, ok := strings.CutPrefix(readInboxFile(t, s), before)
	if !ok {
		t.Fatalf("AppendWoken changed the lines already there:\n%s", readInboxFile(t, s))
	}
	want := `{"ts":"2026-10-10T12:05:00Z","id":"e3","event":"woken","director_id":"d2","via":"wake_cmd"}` + "\n" +
		`{"ts":"2026-10-10T12:05:00Z","id":"e1","event":"woken","director_id":"d2","via":"wake_cmd"}` + "\n" +
		`{"ts":"2026-10-10T12:05:00Z","id":"e2","event":"woken","director_id":"d2","via":"wake_cmd"}` + "\n"
	if added != want {
		t.Fatalf("AppendWoken appended\n%s\nwant\n%s", added, want)
	}

	s.Now = func() time.Time { return time.Date(2026, 10, 10, 12, 6, 0, 0, time.UTC) }
	if err := s.AppendWoken([]string{"e1"}, "d3", InboxWokenViaHerdr); err != nil {
		t.Fatalf("second AppendWoken: %v", err)
	}
	res := mustReadInbox(t, s)
	if res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Errorf("corrupt=%d ignored=%d, want 0 and 0", res.CorruptLines, res.IgnoredEvents)
	}
	for _, tc := range []struct{ id, state, wokenAt string }{
		{"e1", InboxStateOpen, "2026-10-10T12:06:00Z"},
		{"e2", InboxStateAcked, "2026-10-10T12:05:00Z"},
		{"e3", InboxStateResolved, "2026-10-10T12:05:00Z"},
	} {
		if item := inboxItemByID(t, res, tc.id); item.State != tc.state || !item.Woken || item.WokenAt != tc.wokenAt {
			t.Errorf("%s = %+v; want state %s, woken at %s", tc.id, item, tc.state, tc.wokenAt)
		}
	}
	e1 := inboxItemByID(t, res, "e1")
	if last := e1.Events[len(e1.Events)-1]; last.Event != InboxEventWoken || last.DirectorID != "d3" || last.Via != InboxWokenViaHerdr {
		t.Errorf("e1 last event = %+v; want woken by d3 via herdr", last)
	}
}

// AppendWoken refuses, writing nothing: no IDs, a director ID not of the
// form d<N>, a via other than wake_cmd and herdr, an ID given twice, and an
// ID the inbox has no item for (ErrInboxUnknownID, naming the ID), also
// when the other IDs are known.
func TestInboxAppendWoken_RefusesAndWritesNothing(t *testing.T) {
	const wakeCmd = InboxWokenViaWakeCmd
	for _, tc := range []struct {
		name       string
		ids        []string
		directorID string
		via        string
		want       string
		unknownID  bool
	}{
		{"nil ids", nil, "d1", wakeCmd, "org: inbox: wake: the woken event needs at least one item ID", false},
		{"empty ids", []string{}, "d1", wakeCmd, "needs at least one item ID", false},
		{"empty director id", []string{"e1"}, "", wakeCmd, `org: inbox: wake: "" is not a registration id (d<N>, such as d1)`, false},
		{"director id d0", []string{"e1"}, "d0", wakeCmd, `"d0" is not a registration id`, false},
		{"director id d01", []string{"e1"}, "d01", wakeCmd, `"d01" is not a registration id`, false},
		{"director id D1", []string{"e1"}, "D1", wakeCmd, `"D1" is not a registration id`, false},
		{"an inbox id as the director id", []string{"e1"}, "e1", wakeCmd, `"e1" is not a registration id`, false},
		{"empty via", []string{"e1"}, "d1", "", `org: inbox: wake: via "" is not wake_cmd or herdr`, false},
		{"via tmux", []string{"e1"}, "d1", "tmux", `via "tmux" is not wake_cmd or herdr`, false},
		{"via Herdr", []string{"e1"}, "d1", "Herdr", `via "Herdr" is not wake_cmd or herdr`, false},
		{"via with a trailing space", []string{"e1"}, "d1", "herdr ", `via "herdr " is not wake_cmd or herdr`, false},
		{"an id twice", []string{"e1", "e1"}, "d1", wakeCmd, "org: inbox: wake: item e1 is given twice", false},
		{"an id twice among others", []string{"e1", "e2", "e1"}, "d1", wakeCmd, "item e1 is given twice", false},
		{"an unknown id after known ones", []string{"e1", "e2", "e9"}, "d1", wakeCmd, "org: inbox: wake e9: ", true},
		{"an unknown id first", []string{"e9", "e1"}, "d1", wakeCmd, "org: inbox: wake e9: ", true},
		{"an id not of the form e<N>", []string{"e01"}, "d1", InboxWokenViaHerdr, "org: inbox: wake e01: ", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := newTestInboxStore(t)
			mustEscalate(t, s, "org-a")
			mustEscalate(t, s, "org-b")
			before := readInboxFile(t, s)
			err := s.AppendWoken(tc.ids, tc.directorID, tc.via)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("AppendWoken err = %v; want it to contain %q", err, tc.want)
			}
			if errors.Is(err, ErrInboxUnknownID) != tc.unknownID {
				t.Errorf("errors.Is(err, ErrInboxUnknownID) = %v; want %v (err %v)", !tc.unknownID, tc.unknownID, err)
			}
			if got := readInboxFile(t, s); got != before {
				t.Fatalf("a refused AppendWoken changed inbox.jsonl to %q", got)
			}
		})
	}
}

// A missing inbox has no items: AppendWoken returns ErrInboxUnknownID and
// creates neither the state dir nor the lock.
func TestInboxAppendWoken_OnAMissingInboxCreatesNothing(t *testing.T) {
	state := filepath.Join(t.TempDir(), "absent")
	err := NewInboxStore(state).AppendWoken([]string{"e1"}, "d1", InboxWokenViaHerdr)
	if !errors.Is(err, ErrInboxUnknownID) || !strings.Contains(err.Error(), "org: inbox: wake e1: ") {
		t.Fatalf("AppendWoken on a missing inbox err = %v; want org: inbox: wake e1: ... ErrInboxUnknownID", err)
	}
	if _, err := os.Stat(state); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("AppendWoken on a missing inbox created the state dir (stat err = %v)", err)
	}
}

// A last line cut off mid-write stays a line of its own: the woken lines
// start after a newline and fold, and the cut line is one corrupt line.
func TestInboxAppendWoken_TornLastLineGetsTheNewlineFirst(t *testing.T) {
	s, _ := newTestInboxStore(t)
	const torn = `{"ts":"2026-10-10T00:00:00Z","id":"e6","event":"esc`
	content := escalatedLine("e4", "a") + "\n" + escalatedLine("e5", "a") + "\n" + torn
	writeInboxFile(t, s, content)
	if err := s.AppendWoken([]string{"e4", "e5"}, "d1", InboxWokenViaWakeCmd); err != nil {
		t.Fatalf("AppendWoken: %v", err)
	}
	want := content + "\n" +
		`{"ts":"2026-10-10T12:00:00Z","id":"e4","event":"woken","director_id":"d1","via":"wake_cmd"}` + "\n" +
		`{"ts":"2026-10-10T12:00:00Z","id":"e5","event":"woken","director_id":"d1","via":"wake_cmd"}` + "\n"
	if got := readInboxFile(t, s); got != want {
		t.Fatalf("inbox.jsonl =\n%q\nwant\n%q", got, want)
	}
	res := mustReadInbox(t, s)
	if res.CorruptLines != 1 || res.IgnoredEvents != 0 {
		t.Errorf("corrupt=%d ignored=%d, want 1 (the cut line) and 0", res.CorruptLines, res.IgnoredEvents)
	}
	for _, id := range []string{"e4", "e5"} {
		if item := inboxItemByID(t, res, id); !item.Woken || item.State != InboxStateOpen {
			t.Errorf("%s = %+v; want open and woken", id, item)
		}
	}
}

// AppendWoken reads and appends under the inbox lock: while another holder
// has it, AppendWoken waits, then writes once it is released.
func TestInboxAppendWoken_WaitsForTheInboxLock(t *testing.T) {
	s, dir := newTestInboxStore(t)
	mustEscalate(t, s, "org-a")
	held, release := make(chan struct{}), make(chan struct{})
	holderDone := make(chan error, 1)
	go func() {
		holderDone <- withFileLock(dir, inboxLockFile, "inbox", func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	done := make(chan error, 1)
	go func() { done <- s.AppendWoken([]string{"e1"}, "d1", InboxWokenViaHerdr) }()
	select {
	case err := <-done:
		t.Fatalf("AppendWoken returned (%v) while the inbox lock was held", err)
	case <-time.After(150 * time.Millisecond):
	}
	close(release)
	if err := <-holderDone; err != nil {
		t.Fatalf("lock holder: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("AppendWoken after release: %v", err)
	}
	if item := inboxItemByID(t, mustReadInbox(t, s), "e1"); !item.Woken {
		t.Fatalf("e1 = %+v; want woken", item)
	}
}

func TestInboxConcurrentAckAndResolve_LeaveAValidState(t *testing.T) {
	s, _ := newTestInboxStore(t)
	const rounds = 20
	for round := range rounds {
		id := mustEscalate(t, s, "auth-core")
		start := make(chan struct{})
		var wg sync.WaitGroup
		var changed bool
		var ackErr, resolveErr error
		wg.Add(2)
		go func() { defer wg.Done(); <-start; changed, ackErr = s.Ack(id) }()
		go func() { defer wg.Done(); <-start; resolveErr = s.Resolve(id, "done") }()
		close(start)
		wg.Wait()

		if resolveErr != nil {
			t.Fatalf("round %d: Resolve: %v", round, resolveErr)
		}
		item := inboxItemByID(t, mustReadInbox(t, s), id)
		kinds := strings.Join(inboxEventKinds(item), ",")
		switch {
		case ackErr == nil && changed:
			if kinds != "escalated,acked,resolved" {
				t.Fatalf("round %d: ack went first but events = %s", round, kinds)
			}
		case errors.Is(ackErr, ErrInboxResolved) && !changed:
			if kinds != "escalated,resolved" {
				t.Fatalf("round %d: resolve went first but events = %s", round, kinds)
			}
		default:
			t.Fatalf("round %d: Ack = %v, %v", round, changed, ackErr)
		}
		if item.State != InboxStateResolved {
			t.Fatalf("round %d: state = %s, want resolved", round, item.State)
		}
	}
	if res := mustReadInbox(t, s); res.CorruptLines != 0 || res.IgnoredEvents != 0 {
		t.Fatalf("corrupt %d ignored %d, want 0 and 0", res.CorruptLines, res.IgnoredEvents)
	}
}

func TestInboxConcurrentAcks_OnlyOneChangesTheItem(t *testing.T) {
	s, _ := newTestInboxStore(t)
	mustEscalate(t, s, "auth-core")
	const n = 10
	changed := make([]bool, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() { defer wg.Done(); changed[i], errs[i] = s.Ack("e1") }()
	}
	wg.Wait()
	count := 0
	for i := range n {
		if errs[i] != nil {
			t.Fatalf("Ack %d: %v", i, errs[i])
		}
		if changed[i] {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("%d Acks reported changed, want exactly 1", count)
	}
	if item := inboxItemByID(t, mustReadInbox(t, s), "e1"); strings.Join(inboxEventKinds(item), ",") != "escalated,acked" {
		t.Fatalf("events = %v", inboxEventKinds(item))
	}
}

func TestInboxLock_IsSeparateFromTheManifestLockAndSerializesWrites(t *testing.T) {
	s, dir := newTestInboxStore(t)

	// Holding the manifest lock does not block an inbox write.
	start := time.Now()
	if err := withManifestLock(dir, func() error {
		_, err := s.Escalate(InboxEscalation{OrgID: "a", Type: "QUESTION", Body: "b"})
		return err
	}); err != nil {
		t.Fatalf("Escalate under the manifest lock: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("Escalate under the manifest lock took %s: it waited for the manifest lock", elapsed)
	}

	// Holding the inbox lock does block it until released.
	held, release := make(chan struct{}), make(chan struct{})
	holderDone := make(chan error, 1)
	go func() {
		holderDone <- withFileLock(dir, inboxLockFile, "inbox", func() error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	type result struct {
		id  string
		err error
	}
	done := make(chan result, 1)
	go func() {
		id, err := s.Escalate(InboxEscalation{OrgID: "a", Type: "QUESTION", Body: "b"})
		done <- result{id, err}
	}()
	select {
	case r := <-done:
		t.Fatalf("Escalate returned %+v while the inbox lock was held", r)
	case <-time.After(150 * time.Millisecond):
	}
	close(release)
	if err := <-holderDone; err != nil {
		t.Fatalf("lock holder: %v", err)
	}
	if r := <-done; r.err != nil || r.id != "e2" {
		t.Fatalf("Escalate after release = %+v, want e2", r)
	}
}

func TestInboxStore_ReportsAnUnreadableInboxAndAnUnavailableLock(t *testing.T) {
	t.Run("inbox.jsonl is a directory", func(t *testing.T) {
		s, _ := newTestInboxStore(t)
		if err := os.Mkdir(s.Path(), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if _, err := s.Read(); err == nil || !strings.Contains(err.Error(), "org: inbox: read ") {
			t.Errorf("Read err = %v", err)
		}
		if _, err := s.Escalate(InboxEscalation{OrgID: "a", Type: "QUESTION", Body: "b"}); err == nil {
			t.Errorf("Escalate succeeded on an unreadable inbox")
		}
		if _, err := s.Ack("e1"); err == nil || errors.Is(err, ErrInboxUnknownID) {
			t.Errorf("Ack err = %v, want the read error, not ErrInboxUnknownID", err)
		}
	})
	t.Run("inbox.lock is a directory", func(t *testing.T) {
		s, dir := newTestInboxStore(t)
		if err := os.Mkdir(filepath.Join(dir, inboxLockFile), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if _, err := s.Escalate(InboxEscalation{OrgID: "a", Type: "QUESTION", Body: "b"}); err == nil ||
			!strings.Contains(err.Error(), "org: open inbox lock file ") {
			t.Errorf("Escalate err = %v, want org: open inbox lock file ...", err)
		}
		if _, err := os.Stat(s.Path()); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Escalate wrote the inbox without the lock (stat err = %v)", err)
		}
	})
}
