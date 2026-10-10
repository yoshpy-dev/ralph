package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/protocol"
)

// Tests for `ralph org escalate`, the `ralph org inbox` verbs, and `ralph org
// wait --inbox` (plan 2026-10-10-org-inbox-escalate, AC1-AC7 on the CLI
// side). TestMain stubs the desktop notification; a test that checks it
// swaps in recordDesktopNotify. None of these tests runs in parallel: they
// swap package-level overrides, and some chdir.

// inboxHeader is the first line of `ralph org inbox`'s table.
const inboxHeader = "ID\tSTATE\tORG\tTYPE\tTASK_ID\tESCALATED_AT\tNOTIFIED\tSUMMARY"

func inboxQuestion(body string) string { return "TYPE: QUESTION\n\n" + body }

// notifyRecorder records the messages of every desktop notification.
type notifyRecorder struct {
	mu       sync.Mutex
	messages []string
}

func (r *notifyRecorder) calls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.messages)
}

// recordDesktopNotify swaps orgDesktopNotifyOverride for a stub that records
// each message and returns err, and restores TestMain's stub at cleanup.
func recordDesktopNotify(t *testing.T, err error) *notifyRecorder {
	t.Helper()
	rec := &notifyRecorder{}
	orig := orgDesktopNotifyOverride
	orgDesktopNotifyOverride = func(_ context.Context, message string) error {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		rec.messages = append(rec.messages, message)
		return err
	}
	t.Cleanup(func() { orgDesktopNotifyOverride = orig })
	return rec
}

func setInboxPollIntervalOverride(t *testing.T, d time.Duration) {
	t.Helper()
	orig := orgInboxPollIntervalOverride
	orgInboxPollIntervalOverride = d
	t.Cleanup(func() { orgInboxPollIntervalOverride = orig })
}

func readCLIInbox(t *testing.T, stateDir string) org.InboxReadResult {
	t.Helper()
	res, err := org.NewInboxStore(stateDir).Read()
	if err != nil {
		t.Fatalf("read the inbox in %s: %v", stateDir, err)
	}
	return res
}

func cliInboxIDs(items []org.InboxItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func cliInboxItem(t *testing.T, stateDir, id string) org.InboxItem {
	t.Helper()
	for _, item := range readCLIInbox(t, stateDir).Items {
		if item.ID == id {
			return item
		}
	}
	t.Fatalf("no inbox item %s in %s", id, stateDir)
	return org.InboxItem{}
}

// inboxFileOrAbsent returns the inbox file's content, or "<absent>".
func inboxFileOrAbsent(t *testing.T, stateDir string) string {
	t.Helper()
	raw, err := os.ReadFile(org.InboxPathIn(stateDir))
	if errors.Is(err, fs.ErrNotExist) {
		return "<absent>"
	}
	if err != nil {
		t.Fatalf("read the inbox: %v", err)
	}
	return string(raw)
}

// escalateOK runs `ralph org escalate` into the ledger at stateDir and
// returns the new item's ID.
func escalateOK(t *testing.T, stateDir, orgID, text string) string {
	t.Helper()
	stdout, stderr, err := runOrgCmdStreams(t, "escalate", "--org-id", orgID, "--text", text, "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("escalate %q: %v (stdout: %s stderr: %s)", text, err, stdout, stderr)
	}
	fields := strings.Fields(stdout)
	if len(fields) < 2 || fields[0] != "escalated" {
		t.Fatalf("escalate printed %q; want escalated <id> ...", stdout)
	}
	return fields[1]
}

// runInboxOK runs `ralph org <args...>` and fails the test on an error.
func runInboxOK(t *testing.T, args ...string) (stdout, stderr string) {
	t.Helper()
	stdout, stderr, err := runOrgCmdStreams(t, args...)
	if err != nil {
		t.Fatalf("%v: %v (stdout: %s stderr: %s)", args, err, stdout, stderr)
	}
	return stdout, stderr
}

// inboxTableRows returns `ralph org inbox`'s rows (the header checked and
// dropped), each split on tabs.
func inboxTableRows(t *testing.T, stdout string) [][]string {
	t.Helper()
	lines := outputLines(stdout)
	if len(lines) == 0 || lines[0] != inboxHeader {
		t.Fatalf("inbox table does not start with the header %q:\n%s", inboxHeader, stdout)
	}
	rows := make([][]string, 0, len(lines)-1)
	for _, line := range lines[1:] {
		rows = append(rows, strings.Split(line, "\t"))
	}
	return rows
}

// runOrgCmdStreamsWithin is runOrgCmdStreams that fails the test when the
// command has not returned within limit, so a wait that never ends fails
// instead of hanging the test binary.
func runOrgCmdStreamsWithin(t *testing.T, limit time.Duration, args ...string) (stdout, stderr string, elapsed time.Duration, err error) {
	t.Helper()
	type result struct {
		stdout, stderr string
		elapsed        time.Duration
		err            error
	}
	done := make(chan result, 1)
	start := time.Now()
	go func() {
		stdout, stderr, err := runOrgCmdStreams(t, args...)
		done <- result{stdout, stderr, time.Since(start), err}
	}()
	select {
	case r := <-done:
		return r.stdout, r.stderr, r.elapsed, r.err
	case <-time.After(limit):
		t.Fatalf("%v did not return within %v", args, limit)
		return "", "", 0, nil
	}
}

func TestOrgEscalate_PrintsTheIDAndNotifiesTheHuman(t *testing.T) {
	rec := recordDesktopNotify(t, nil)
	stateDir := filepath.Join(t.TempDir(), "state")

	stdout, stderr := runInboxOK(t, "escalate", "--org-id", "org-a", "--text", inboxQuestion("plan A or plan B?"), "--state-dir", stateDir)
	if want := "escalated e1 (org=org-a type=QUESTION)\n"; stdout != want {
		t.Errorf("stdout = %q; want %q", stdout, want)
	}
	for _, sub := range []string{"ORG ESCALATION:", "org=org-a", "item=e1", "ralph org inbox show e1"} {
		if !strings.Contains(stderr, sub) {
			t.Errorf("stderr %q does not contain %q", stderr, sub)
		}
	}
	item := cliInboxItem(t, stateDir, "e1")
	if item.State != org.InboxStateOpen || !item.Notified || item.OrgID != "org-a" || item.Type != "QUESTION" {
		t.Errorf("e1 = %+v; want open, notified, org-a, QUESTION", item)
	}
	if last := item.Events[len(item.Events)-1]; last.Event != org.InboxEventNotified || last.Osascript != "ok" || last.Reason != "inbox_no_director" {
		t.Errorf("last e1 event = %+v; want notified (inbox_no_director, osascript ok)", last)
	}
	escLines := readLogLines(t, org.EscalationsPathIn(stateDir))
	if len(escLines) != 1 {
		t.Fatalf("escalations.jsonl has %d lines; want 1: %v", len(escLines), escLines)
	}
	var esc map[string]any
	if err := json.Unmarshal([]byte(escLines[0]), &esc); err != nil {
		t.Fatalf("parse the escalations line: %v", err)
	}
	if esc["inbox_id"] != "e1" || esc["org_id"] != "org-a" || esc["reason"] != "inbox_no_director" {
		t.Errorf("escalations line = %v; want inbox_id e1, org_id org-a, reason inbox_no_director", esc)
	}
	if calls := rec.calls(); len(calls) != 1 || !strings.Contains(calls[0], "e1") || strings.Contains(calls[0], "plan A") {
		t.Errorf("desktop notifications = %q; want one naming e1 without the body", calls)
	}

	stdout, _ = runInboxOK(t, "escalate", "--org-id", "org-b", "--text", "TYPE: BLOCKED\nTASK_ID: org-b\n\nstuck", "--state-dir", stateDir)
	if want := "escalated e2 (org=org-b type=BLOCKED)\n"; stdout != want {
		t.Errorf("second escalate stdout = %q; want %q", stdout, want)
	}
}

// AC3: a failed desktop notification is recorded in the notified event and
// does not fail the command.
func TestOrgEscalate_DesktopNotificationFailureStillExitsZero(t *testing.T) {
	recordDesktopNotify(t, errors.New("osascript is not allowed"))
	stateDir := t.TempDir()

	escalateOK(t, stateDir, "org-a", inboxQuestion("which?"))
	item := cliInboxItem(t, stateDir, "e1")
	if !item.Notified {
		t.Fatalf("e1 = %+v; want notified", item)
	}
	if last := item.Events[len(item.Events)-1]; last.Osascript != "failed: osascript is not allowed" {
		t.Errorf("notified osascript = %q; want the failure recorded", last.Osascript)
	}
}

// AC1: a refused escalate writes nothing (the state dir is not even
// created) and notifies no one.
func TestOrgEscalate_RefusalsWriteNothing(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantSub string
	}{
		{"task_type", []string{"--org-id", "org-a", "--text", "TYPE: TASK\nTASK_ID: org-a\n\ndo it"}, "TYPE TASK cannot be escalated"},
		{"blocked_without_task_id", []string{"--org-id", "org-a", "--text", "TYPE: BLOCKED\n\nstuck"}, "TASK_ID"},
		{"result_without_task_id", []string{"--org-id", "org-a", "--text", "TYPE: RESULT\n\ndone"}, "TASK_ID"},
		{"body_too_long", []string{"--org-id", "org-a", "--text", inboxQuestion(strings.Repeat("x", protocol.DefaultMaxBodyChars+1))}, "body"},
		{"no_type_header", []string{"--org-id", "org-a", "--text", "which plan?"}, "TYPE"},
		{"bad_org_id", []string{"--org-id", "Org_A", "--text", inboxQuestion("which?")}, "invalid org_id"},
		{"missing_org_id", []string{"--text", inboxQuestion("which?")}, "--org-id is required"},
		{"blank_text", []string{"--org-id", "org-a", "--text", "  \n "}, "--text is required"},
		{"positional_text", []string{"--org-id", "org-a", inboxQuestion("which?")}, "unknown command"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := recordDesktopNotify(t, nil)
			stateDir := filepath.Join(t.TempDir(), "state")

			stdout, stderr, err := runOrgCmdStreams(t, append(append([]string{"escalate"}, tc.args...), "--state-dir", stateDir)...)
			if err == nil {
				t.Fatalf("expected a refusal; stdout: %s stderr: %s", stdout, stderr)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("error %q does not contain %q", err, tc.wantSub)
			}
			if stdout != "" || strings.Contains(stderr, "ORG ESCALATION") {
				t.Errorf("a refusal printed stdout %q, stderr %q", stdout, stderr)
			}
			if _, statErr := os.Stat(stateDir); !errors.Is(statErr, fs.ErrNotExist) {
				t.Errorf("a refusal created the state dir %s (stat err %v)", stateDir, statErr)
			}
			if calls := rec.calls(); len(calls) != 0 {
				t.Errorf("a refusal sent desktop notifications %q", calls)
			}
		})
	}
}

// AC2b: the item is recorded but escalations.jsonl cannot be written, so
// escalate prints the ID and exits 1; the list shows it unnotified, and
// `inbox notify` completes it under the same ID.
func TestOrgEscalate_RecordedNotNotified_ExitsOneThenInboxNotifyCompletesIt(t *testing.T) {
	recordDesktopNotify(t, nil)
	stateDir := t.TempDir()
	escalations := org.EscalationsPathIn(stateDir)
	// A directory where escalations.jsonl should be: the append fails.
	if err := os.Mkdir(escalations, 0o755); err != nil {
		t.Fatal(err)
	}

	stdout, _, err := runOrgCmdStreams(t, "escalate", "--org-id", "org-a", "--text", inboxQuestion("which?"), "--state-dir", stateDir)
	if err == nil {
		t.Fatal("expected escalate to exit 1 when the notification cannot be recorded")
	}
	if want := "escalated e1 (org=org-a type=QUESTION)\n"; stdout != want {
		t.Errorf("stdout = %q; want the ID line %q even though the notification failed", stdout, want)
	}
	for _, sub := range []string{"recorded as e1", "ralph org inbox notify e1"} {
		if !strings.Contains(err.Error(), sub) {
			t.Errorf("error %q does not contain %q", err, sub)
		}
	}
	listOut, _ := runInboxOK(t, "inbox", "--state-dir", stateDir)
	if rows := inboxTableRows(t, listOut); len(rows) != 1 || rows[0][0] != "e1" || rows[0][6] != "no" {
		t.Errorf("inbox rows = %q; want e1 with NOTIFIED no", rows)
	}

	if err := os.Remove(escalations); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := runInboxOK(t, "inbox", "notify", "e1", "--state-dir", stateDir)
	if stdout != "notified e1\n" || !strings.Contains(stderr, "item=e1") {
		t.Errorf("inbox notify stdout %q, stderr %q; want notified e1 and the banner", stdout, stderr)
	}
	inbox := readCLIInbox(t, stateDir)
	if ids := cliInboxIDs(inbox.Items); !slices.Equal(ids, []string{"e1"}) || !inbox.Items[0].Notified {
		t.Errorf("inbox = %+v; want e1 alone, notified", inbox.Items)
	}
	listOut, _ = runInboxOK(t, "inbox", "--state-dir", stateDir)
	if rows := inboxTableRows(t, listOut); len(rows) != 1 || rows[0][6] != "yes" {
		t.Errorf("inbox rows after notify = %q; want NOTIFIED yes", rows)
	}
}

// AC2b: an escalate the inbox cannot record still reaches the human (the
// banner and the desktop notification) and exits 1 with no ID.
func TestOrgEscalate_InboxUnwritable_AlertsTheHumanAndExitsOne(t *testing.T) {
	rec := recordDesktopNotify(t, nil)
	stateDir := t.TempDir()
	// A directory where inbox.jsonl should be: the read fails.
	if err := os.Mkdir(org.InboxPathIn(stateDir), 0o755); err != nil {
		t.Fatal(err)
	}

	stdout, stderr, err := runOrgCmdStreams(t, "escalate", "--org-id", "org-a", "--text", inboxQuestion("which?"), "--state-dir", stateDir)
	if err == nil || !strings.Contains(err.Error(), "could not be recorded") {
		t.Fatalf("error = %v; want the item could not be recorded", err)
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want no ID for an item that was not recorded", stdout)
	}
	if !strings.Contains(stderr, "NOT RECORDED") {
		t.Errorf("stderr %q has no not-recorded banner", stderr)
	}
	if calls := rec.calls(); len(calls) != 1 {
		t.Errorf("desktop notifications = %q; want one", calls)
	}
}

// Self-review L1: a ralph.toml that does not load keeps escalate from
// building the runtime, but the item still reaches the human as not
// recorded (the banner and the desktop notification), the same outcome as
// an inbox that cannot be written. A message escalate refuses is still
// refused first, with no banner and no notification.
func TestOrgEscalate_BrokenConfig_AlertsTheHumanAndExitsOne(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "ralph.toml")
	if err := os.WriteFile(configPath, []byte("[org\nmax_seats = = 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("an accepted message", func(t *testing.T) {
		rec := recordDesktopNotify(t, nil)
		stateDir := filepath.Join(t.TempDir(), "state")
		stdout, stderr, err := runOrgCmdStreams(t, "escalate", "--org-id", "org-a", "--text", "TYPE: BLOCKED\nTASK_ID: org-a\n\nstuck",
			"--state-dir", stateDir, "--config", configPath)
		if err == nil || !strings.Contains(err.Error(), "could not be recorded") || !strings.Contains(err.Error(), "load config") {
			t.Fatalf("error = %v; want the item could not be recorded, naming the config load", err)
		}
		if stdout != "" {
			t.Errorf("stdout = %q; want no ID for an item that was not recorded", stdout)
		}
		for _, sub := range []string{"ORG ESCALATION (NOT RECORDED):", "org=org-a", "type=BLOCKED", "load config"} {
			if !strings.Contains(stderr, sub) {
				t.Errorf("stderr %q does not contain %q", stderr, sub)
			}
		}
		if calls := rec.calls(); len(calls) != 1 || !strings.Contains(calls[0], "org-a") || strings.Contains(calls[0], "stuck") {
			t.Errorf("desktop notifications = %q; want one naming org-a without the body", calls)
		}
		if got := inboxFileOrAbsent(t, stateDir); got != "<absent>" {
			t.Errorf("inbox = %q; want absent", got)
		}
		// The fallback runtime's EscalationsPath is the ledger's
		// escalations.jsonl, so the best-effort line lands there.
		escLines := readLogLines(t, org.EscalationsPathIn(stateDir))
		if len(escLines) != 1 {
			t.Fatalf("escalations.jsonl has %d lines; want 1 inbox_not_recorded line: %v", len(escLines), escLines)
		}
		var esc map[string]any
		if err := json.Unmarshal([]byte(escLines[0]), &esc); err != nil {
			t.Fatalf("parse the escalations line: %v", err)
		}
		if esc["reason"] != "inbox_not_recorded" || esc["org_id"] != "org-a" {
			t.Errorf("escalations line = %v; want reason inbox_not_recorded, org_id org-a", esc)
		}
		if _, ok := esc["inbox_id"]; ok {
			t.Errorf("a not-recorded line carries inbox_id: %v", esc)
		}
	})

	t.Run("a refused message", func(t *testing.T) {
		rec := recordDesktopNotify(t, nil)
		stateDir := filepath.Join(t.TempDir(), "state")
		stdout, stderr, err := runOrgCmdStreams(t, "escalate", "--org-id", "org-a", "--text", "TYPE: BLOCKED\n\nno task id",
			"--state-dir", stateDir, "--config", configPath)
		if err == nil || !strings.Contains(err.Error(), "message rejected") {
			t.Fatalf("error = %v; want the protocol refusal", err)
		}
		if stdout != "" || strings.Contains(stderr, "ORG ESCALATION") || len(rec.calls()) != 0 {
			t.Errorf("a refusal printed stdout %q, stderr %q, notified %q", stdout, stderr, rec.calls())
		}
		if _, statErr := os.Stat(stateDir); !errors.Is(statErr, fs.ErrNotExist) {
			t.Errorf("a refusal created the state dir %s (stat err %v)", stateDir, statErr)
		}
	})
}

// Self-review L2: the `ralph org inbox` commands that escalate, inbox
// notify, and inbox show print carry --state-dir when it was passed, and
// stay bare when the ledger came from RALPH_ORG_STATE_DIR (the rule of
// orgReadCommandHint).
func TestOrgInbox_RecoveryCommandsRepeatAnExplicitStateDir(t *testing.T) {
	for _, explicit := range []bool{true, false} {
		t.Run(map[bool]string{true: "flag", false: "env"}[explicit], func(t *testing.T) {
			recordDesktopNotify(t, nil)
			stateDir := t.TempDir()
			// A directory where escalations.jsonl should be: the item is
			// recorded but not notified, so every command names a recovery.
			if err := os.Mkdir(org.EscalationsPathIn(stateDir), 0o755); err != nil {
				t.Fatal(err)
			}
			ledger := []string{"--state-dir", stateDir}
			suffix := " --state-dir '" + stateDir + "'"
			if !explicit {
				t.Setenv(org.EnvOrgStateDir, stateDir)
				ledger, suffix = nil, ""
			}

			stdout, stderr, err := runOrgCmdStreams(t, append([]string{"escalate", "--org-id", "org-a", "--text", inboxQuestion("which?")}, ledger...)...)
			if err == nil || stdout != "escalated e1 (org=org-a type=QUESTION)\n" {
				t.Fatalf("escalate = %q, %v; want e1 recorded and exit 1", stdout, err)
			}
			if want := "run `ralph org inbox notify e1" + suffix + "` to send it again"; !strings.Contains(err.Error(), want) {
				t.Errorf("escalate error %q does not contain %q", err, want)
			}
			if want := "read it with: ralph org inbox show e1" + suffix + "\n"; !strings.Contains(stderr, want) {
				t.Errorf("escalate banner %q does not contain %q", stderr, want)
			}

			_, stderr, err = runOrgCmdStreams(t, append([]string{"inbox", "notify", "e1"}, ledger...)...)
			if want := "run `ralph org inbox notify e1" + suffix + "` again"; err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("inbox notify error = %v; want it to contain %q", err, want)
			}
			if want := "read it with: ralph org inbox show e1" + suffix + "\n"; !strings.Contains(stderr, want) {
				t.Errorf("inbox notify banner %q does not contain %q", stderr, want)
			}

			stdout, _ = runInboxOK(t, append([]string{"inbox", "show", "e1"}, ledger...)...)
			if want := "notified: no (send it with: ralph org inbox notify e1" + suffix + ")\n"; !strings.Contains(stdout, want) {
				t.Errorf("inbox show %q does not contain %q", stdout, want)
			}
		})
	}
}

// seedInboxStates escalates e1 (open), e2 (acked), and e3 (resolved) into
// stateDir through the CLI.
func seedInboxStates(t *testing.T, stateDir string) {
	t.Helper()
	escalateOK(t, stateDir, "org-a", inboxQuestion("first line\nsecond line"))
	escalateOK(t, stateDir, "org-b", "TYPE: BLOCKED\nTASK_ID: org-b\n\ngh auth is missing")
	escalateOK(t, stateDir, "org-a", "TYPE: RESULT\nTASK_ID: org-a\n\nPR https://example.invalid/pull/1")
	runInboxOK(t, "inbox", "ack", "e2", "--state-dir", stateDir)
	runInboxOK(t, "inbox", "resolve", "e3", "--note", "https://example.invalid/pull/1", "--state-dir", stateDir)
}

// AC4: the list shows open and acked items, --all adds resolved, and --json
// carries the same items.
func TestOrgInbox_ListAllAndJSON(t *testing.T) {
	stateDir := t.TempDir()
	seedInboxStates(t, stateDir)

	stdout, stderr := runInboxOK(t, "inbox", "--state-dir", stateDir)
	if stderr != "" {
		t.Errorf("stderr = %q; want nothing for an undamaged inbox", stderr)
	}
	rows := inboxTableRows(t, stdout)
	if len(rows) != 2 {
		t.Fatalf("inbox rows = %q; want e1 and e2", rows)
	}
	for i, want := range [][]string{
		{"e1", "open", "org-a", "QUESTION", "-", "", "yes", "first line"},
		{"e2", "acked", "org-b", "BLOCKED", "org-b", "", "yes", "gh auth is missing"},
	} {
		got := rows[i]
		if len(got) != len(want) {
			t.Fatalf("row %d = %q; want %d columns", i, got, len(want))
		}
		if _, err := time.Parse(time.RFC3339, got[5]); err != nil {
			t.Errorf("row %d ESCALATED_AT %q is not RFC3339: %v", i, got[5], err)
		}
		got[5] = ""
		if !slices.Equal(got, want) {
			t.Errorf("row %d = %q; want %q", i, got, want)
		}
	}

	stdout, _ = runInboxOK(t, "inbox", "--all", "--state-dir", stateDir)
	rows = inboxTableRows(t, stdout)
	if len(rows) != 3 || rows[2][0] != "e3" || rows[2][1] != "resolved" {
		t.Errorf("inbox --all rows = %q; want e1, e2, and e3 resolved", rows)
	}

	for _, tc := range []struct {
		args []string
		want []string
	}{
		{[]string{"inbox", "--json"}, []string{"e1", "e2"}},
		{[]string{"inbox", "--all", "--json"}, []string{"e1", "e2", "e3"}},
	} {
		stdout, _ := runInboxOK(t, append(tc.args, "--state-dir", stateDir)...)
		var payload orgInboxJSON
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("%v stdout is not JSON: %v\n%s", tc.args, err, stdout)
		}
		if ids := cliInboxIDs(payload.Items); !slices.Equal(ids, tc.want) {
			t.Errorf("%v items = %v; want %v", tc.args, ids, tc.want)
		}
		if payload.CorruptLines != 0 || payload.IgnoredEvents != 0 || len(payload.Items[0].Events) == 0 {
			t.Errorf("%v payload = %+v; want no damage and the events of each item", tc.args, payload)
		}
	}
}

// AC4: show prints the fields, the whole message, and the history; --json
// prints the item; an unknown ID exits 1.
func TestOrgInbox_Show(t *testing.T) {
	stateDir := t.TempDir()
	seedInboxStates(t, stateDir)

	stdout, _ := runInboxOK(t, "inbox", "show", "e2", "--state-dir", stateDir)
	for _, sub := range []string{
		"id: e2\n", "state: acked\n", "org: org-b\n", "type: BLOCKED\n", "task_id: org-b\n",
		"acked_at: ", "notified: yes (", "message:\n  TYPE: BLOCKED\n  TASK_ID: org-b\n  \n  gh auth is missing\n",
		"events:\n", " escalated\n", " notified reason=inbox_no_director osascript=ok\n", " acked\n",
	} {
		if !strings.Contains(stdout, sub) {
			t.Errorf("show e2 does not contain %q:\n%s", sub, stdout)
		}
	}
	if strings.Contains(stdout, "resolved_at:") || strings.Contains(stdout, "note:") {
		t.Errorf("show e2 prints fields an acked item does not have:\n%s", stdout)
	}

	stdout, _ = runInboxOK(t, "inbox", "show", "e3", "--state-dir", stateDir)
	for _, sub := range []string{"state: resolved\n", "resolved_at: ", "note: https://example.invalid/pull/1\n", " resolved note=https://example.invalid/pull/1\n"} {
		if !strings.Contains(stdout, sub) {
			t.Errorf("show e3 does not contain %q:\n%s", sub, stdout)
		}
	}

	stdout, _ = runInboxOK(t, "inbox", "show", "e2", "--json", "--state-dir", stateDir)
	var item org.InboxItem
	if err := json.Unmarshal([]byte(stdout), &item); err != nil {
		t.Fatalf("show --json stdout is not JSON: %v\n%s", err, stdout)
	}
	var kinds []string
	for _, ev := range item.Events {
		kinds = append(kinds, ev.Event)
	}
	if item.ID != "e2" || item.State != org.InboxStateAcked || !slices.Equal(kinds, []string{"escalated", "notified", "acked"}) {
		t.Errorf("show --json = %+v (events %v); want e2 acked with escalated, notified, acked", item, kinds)
	}

	for _, args := range [][]string{{"inbox", "show", "e9"}, {"inbox", "show", "e9", "--json"}} {
		stdout, _, err := runOrgCmdStreams(t, append(args, "--state-dir", stateDir)...)
		if !errors.Is(err, org.ErrInboxUnknownID) {
			t.Errorf("%v error = %v; want ErrInboxUnknownID", args, err)
		}
		if stdout != "" {
			t.Errorf("%v stdout = %q; want nothing", args, stdout)
		}
	}
}

func TestOrgInbox_EmptyInbox(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"inbox"}, "no open or acked inbox items\n"},
		{[]string{"inbox", "--all"}, "no inbox items\n"},
	} {
		if stdout, _ := runInboxOK(t, append(tc.args, "--state-dir", stateDir)...); stdout != tc.want {
			t.Errorf("%v stdout = %q; want %q", tc.args, stdout, tc.want)
		}
	}
	stdout, _ := runInboxOK(t, "inbox", "--json", "--state-dir", stateDir)
	if !strings.Contains(stdout, `"items": []`) {
		t.Errorf("inbox --json on an empty inbox = %s; want an empty items array", stdout)
	}
	if _, err := os.Stat(stateDir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("reading an empty inbox created the state dir (stat err %v)", err)
	}
}

func TestOrgInbox_WarnsAboutDamageOnStderr(t *testing.T) {
	stateDir := t.TempDir()
	content := `{"ts":"2026-10-10T00:00:00Z","id":"e1","event":"escalated","org_id":"org-a","type":"QUESTION","body":"TYPE: QUESTION\n\nwhich?"}` + "\n" +
		"not json\n" +
		`{"ts":"2026-10-10T00:00:01Z","id":"e1","event":"bogus"}` + "\n"
	if err := os.WriteFile(org.InboxPathIn(stateDir), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{{"inbox", "--json"}, {"inbox", "show", "e1"}} {
		stdout, stderr := runInboxOK(t, append(args, "--state-dir", stateDir)...)
		for _, sub := range []string{"1 unreadable inbox line(s) skipped", "1 inbox event(s) ignored", org.InboxPathIn(stateDir)} {
			if !strings.Contains(stderr, sub) {
				t.Errorf("%v stderr %q does not contain %q", args, stderr, sub)
			}
		}
		if strings.Contains(stdout, "warning") {
			t.Errorf("%v printed the warning on stdout: %s", args, stdout)
		}
	}
	stdout, _ := runInboxOK(t, "inbox", "--json", "--state-dir", stateDir)
	var payload orgInboxJSON
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
	}
	if payload.CorruptLines != 1 || payload.IgnoredEvents != 1 || len(payload.Items) != 1 {
		t.Errorf("payload = %+v; want 1 corrupt line, 1 ignored event, 1 item", payload)
	}
}

// A message with escape sequences reaches the terminal escaped, in the list
// and in show.
func TestOrgInbox_EscapesControlCharacters(t *testing.T) {
	stateDir := t.TempDir()
	escalateOK(t, stateDir, "org-a", "TYPE: BLOCKED\nTASK_ID: x\x1b[2J\n\n\x1b[31mred\x1b[0m\tdone")

	for _, args := range [][]string{{"inbox"}, {"inbox", "show", "e1"}, {"wait", "--inbox"}} {
		stdout, _ := runInboxOK(t, append(args, "--state-dir", stateDir)...)
		if strings.ContainsRune(stdout, 0x1b) {
			t.Errorf("%v printed a raw ESC: %q", args, stdout)
		}
		if !strings.Contains(stdout, `\x1b[31mred\x1b[0m`) {
			t.Errorf("%v does not show the escaped body: %q", args, stdout)
		}
	}
	listOut, _ := runInboxOK(t, "inbox", "--state-dir", stateDir)
	rows := inboxTableRows(t, listOut)
	if len(rows) != 1 || len(rows[0]) != 8 || rows[0][4] != `x\x1b[2J` || rows[0][7] != `\x1b[31mred\x1b[0m\tdone` {
		t.Errorf("inbox rows = %q; want 8 columns, the TASK_ID and the tab in the summary escaped", rows)
	}
}

func TestInboxSummary(t *testing.T) {
	cases := []struct {
		name, text, want string
	}{
		{"first_body_line", "TYPE: QUESTION\n\nfirst\nsecond", "first"},
		{"skips_blank_lines_and_trims", "TYPE: QUESTION\n\n\n   padded  \nnext", "padded"},
		{"crlf", "TYPE: QUESTION\r\n\r\n\r\nwindows\r\n", "windows"},
		{"header_only", "TYPE: QUESTION\nTASK_ID: org-a", "-"},
		{"untyped_text_uses_the_whole_text", "not typed\nline two", "not typed"},
		{"control_characters_escaped", "TYPE: QUESTION\n\na\tb\x07", `a\tb\a`},
		{"cut_to_the_cap", "TYPE: QUESTION\n\n" + strings.Repeat("x", 100), strings.Repeat("x", inboxSummaryMaxRunes-3) + "..."},
		{"at_the_cap_kept", "TYPE: QUESTION\n\n" + strings.Repeat("y", inboxSummaryMaxRunes), strings.Repeat("y", inboxSummaryMaxRunes)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := inboxSummary(tc.text); got != tc.want {
				t.Errorf("inboxSummary(%q) = %q; want %q", tc.text, got, tc.want)
			}
		})
	}
	if got, want := org.PrintableInboxText("a\n\tb\r\x1b", true), "a\n\tb\\r\\x1b"; got != want {
		t.Errorf("org.PrintableInboxText keeping line breaks = %q; want %q", got, want)
	}
}

// AC5: ack moves open to acked; a second ack changes nothing and exits 0; a
// resolved or unknown item exits 1.
func TestOrgInboxAck_Transitions(t *testing.T) {
	stateDir := t.TempDir()
	escalateOK(t, stateDir, "org-a", inboxQuestion("which?"))
	escalateOK(t, stateDir, "org-a", inboxQuestion("and this?"))

	if stdout, _ := runInboxOK(t, "inbox", "ack", "e1", "--state-dir", stateDir); stdout != "acked e1\n" {
		t.Errorf("ack e1 stdout = %q", stdout)
	}
	if got := cliInboxItem(t, stateDir, "e1").State; got != org.InboxStateAcked {
		t.Errorf("e1 state = %s; want acked", got)
	}
	before := inboxFileOrAbsent(t, stateDir)
	if stdout, _ := runInboxOK(t, "inbox", "ack", "e1", "--state-dir", stateDir); stdout != "e1 is already acked; nothing changed\n" {
		t.Errorf("second ack e1 stdout = %q", stdout)
	}
	if after := inboxFileOrAbsent(t, stateDir); after != before {
		t.Errorf("a second ack wrote to the inbox:\nbefore: %s\nafter:  %s", before, after)
	}

	runInboxOK(t, "inbox", "resolve", "e2", "--note", "docs/reports/x.md", "--state-dir", stateDir)
	before = inboxFileOrAbsent(t, stateDir)
	for _, tc := range []struct {
		args   []string
		wantIs error
	}{
		{[]string{"inbox", "ack", "e2"}, org.ErrInboxResolved},
		{[]string{"inbox", "ack", "e9"}, org.ErrInboxUnknownID},
	} {
		stdout, _, err := runOrgCmdStreams(t, append(tc.args, "--state-dir", stateDir)...)
		if !errors.Is(err, tc.wantIs) {
			t.Errorf("%v error = %v; want %v", tc.args, err, tc.wantIs)
		}
		if stdout != "" {
			t.Errorf("%v stdout = %q; want nothing", tc.args, stdout)
		}
	}
	if _, _, err := runOrgCmdStreams(t, "inbox", "ack", "--state-dir", stateDir); err == nil {
		t.Error("ack without an ID succeeded")
	}
	if after := inboxFileOrAbsent(t, stateDir); after != before {
		t.Errorf("a refused ack wrote to the inbox:\nbefore: %s\nafter:  %s", before, after)
	}
}

// AC5: resolve needs a one-line note of at most 500 characters with no
// control characters, takes open and acked items, and refuses a resolved one.
func TestOrgInboxResolve_NoteRulesAndTransitions(t *testing.T) {
	stateDir := t.TempDir()
	escalateOK(t, stateDir, "org-a", inboxQuestion("which?"))
	escalateOK(t, stateDir, "org-a", inboxQuestion("and this?"))
	runInboxOK(t, "inbox", "ack", "e1", "--state-dir", stateDir)

	before := inboxFileOrAbsent(t, stateDir)
	for _, tc := range []struct {
		name    string
		note    []string
		wantSub string
	}{
		{"no_note", nil, "the resolve note is required"},
		{"empty_note", []string{"--note", ""}, "the resolve note is required"},
		{"two_lines", []string{"--note", "docs/a.md\ndocs/b.md"}, "single line"},
		{"501_characters", []string{"--note", strings.Repeat("n", org.InboxNoteMaxRunes+1)}, "at most 500"},
		{"control_character", []string{"--note", "docs/a.md\x07"}, "control character"},
	} {
		args := append([]string{"inbox", "resolve", "e1", "--state-dir", stateDir}, tc.note...)
		stdout, _, err := runOrgCmdStreams(t, args...)
		if err == nil || !strings.Contains(err.Error(), tc.wantSub) {
			t.Errorf("%s: error = %v; want it to contain %q", tc.name, err, tc.wantSub)
		}
		if stdout != "" {
			t.Errorf("%s: stdout = %q; want nothing", tc.name, stdout)
		}
	}
	if after := inboxFileOrAbsent(t, stateDir); after != before {
		t.Errorf("a refused resolve wrote to the inbox:\nbefore: %s\nafter:  %s", before, after)
	}

	if stdout, _ := runInboxOK(t, "inbox", "resolve", "e1", "--note", "docs/reports/x.md", "--state-dir", stateDir); stdout != "resolved e1\n" {
		t.Errorf("resolve e1 (acked) stdout = %q", stdout)
	}
	longest := strings.Repeat("n", org.InboxNoteMaxRunes)
	runInboxOK(t, "inbox", "resolve", "e2", "--note", longest, "--state-dir", stateDir)
	if e1, e2 := cliInboxItem(t, stateDir, "e1"), cliInboxItem(t, stateDir, "e2"); e1.State != org.InboxStateResolved || e1.Note != "docs/reports/x.md" ||
		e2.State != org.InboxStateResolved || e2.Note != longest {
		t.Errorf("e1 = %+v, e2 = %+v; want both resolved with their notes", e1, e2)
	}

	for _, tc := range []struct {
		id     string
		wantIs error
	}{{"e1", org.ErrInboxResolved}, {"e9", org.ErrInboxUnknownID}} {
		if _, _, err := runOrgCmdStreams(t, "inbox", "resolve", tc.id, "--note", "docs/y.md", "--state-dir", stateDir); !errors.Is(err, tc.wantIs) {
			t.Errorf("resolve %s error = %v; want %v", tc.id, err, tc.wantIs)
		}
	}
}

func TestOrgInboxNotify_RefusesUnknownAndResolvedItems(t *testing.T) {
	rec := recordDesktopNotify(t, nil)
	stateDir := t.TempDir()
	escalateOK(t, stateDir, "org-a", inboxQuestion("which?"))
	runInboxOK(t, "inbox", "resolve", "e1", "--note", "docs/reports/x.md", "--state-dir", stateDir)
	callsBefore, before := len(rec.calls()), inboxFileOrAbsent(t, stateDir)

	for _, tc := range []struct {
		id     string
		wantIs error
	}{{"e1", org.ErrInboxResolved}, {"e9", org.ErrInboxUnknownID}} {
		stdout, stderr, err := runOrgCmdStreams(t, "inbox", "notify", tc.id, "--state-dir", stateDir)
		if !errors.Is(err, tc.wantIs) {
			t.Errorf("notify %s error = %v; want %v", tc.id, err, tc.wantIs)
		}
		if stdout != "" || stderr != "" {
			t.Errorf("notify %s printed stdout %q, stderr %q; want nothing", tc.id, stdout, stderr)
		}
	}
	if len(rec.calls()) != callsBefore || inboxFileOrAbsent(t, stateDir) != before {
		t.Error("a refused inbox notify notified or wrote to the inbox")
	}
}

// The inbox verbs refuse --org-id instead of ignoring it.
func TestOrgInbox_RefusesOrgID(t *testing.T) {
	stateDir := t.TempDir()
	escalateOK(t, stateDir, "org-a", inboxQuestion("which?"))
	before := inboxFileOrAbsent(t, stateDir)

	for _, args := range [][]string{
		{"inbox"},
		{"inbox", "show", "e1"},
		{"inbox", "ack", "e1"},
		{"inbox", "resolve", "e1", "--note", "docs/reports/x.md"},
		{"inbox", "notify", "e1"},
	} {
		stdout, _, err := runOrgCmdStreams(t, append(args, "--org-id", "org-a", "--state-dir", stateDir)...)
		if err == nil || !strings.Contains(err.Error(), "--org-id does not apply") {
			t.Errorf("%v --org-id error = %v; want a refusal", args, err)
		}
		if stdout != "" {
			t.Errorf("%v --org-id stdout = %q; want nothing", args, stdout)
		}
	}
	if after := inboxFileOrAbsent(t, stateDir); after != before {
		t.Errorf("a refused inbox verb wrote to the inbox:\nbefore: %s\nafter:  %s", before, after)
	}
}

// AC6: an open item already in the inbox returns at once (the poll interval
// is an hour and there is no timeout); an acked one is not printed.
func TestOrgWaitInbox_ReturnsAnOpenItemAtOnce(t *testing.T) {
	setInboxPollIntervalOverride(t, time.Hour)
	stateDir := t.TempDir()
	escalateOK(t, stateDir, "org-a", inboxQuestion("which plan?\nmore detail"))
	escalateOK(t, stateDir, "org-b", inboxQuestion("already read"))
	runInboxOK(t, "inbox", "ack", "e2", "--state-dir", stateDir)

	stdout, _, _, err := runOrgCmdStreamsWithin(t, 5*time.Second, "wait", "--inbox", "--timeout-ms", "0", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("wait --inbox: %v", err)
	}
	if want := "e1\torg-a\tQUESTION\twhich plan?\n"; stdout != want {
		t.Errorf("stdout = %q; want %q", stdout, want)
	}
}

// AC6: with nothing open, wait --inbox returns the item another process
// escalates while it waits.
func TestOrgWaitInbox_ReturnsAnItemEscalatedWhileWaiting(t *testing.T) {
	setInboxPollIntervalOverride(t, 20*time.Millisecond)
	stateDir := t.TempDir()
	var wg sync.WaitGroup
	t.Cleanup(wg.Wait)
	wg.Go(func() {
		time.Sleep(200 * time.Millisecond)
		if _, err := org.NewInboxStore(stateDir).Escalate(org.InboxEscalation{
			OrgID: "org-a", Type: "RESULT", TaskID: "org-a", Body: "TYPE: RESULT\nTASK_ID: org-a\n\nPR opened",
		}); err != nil {
			t.Errorf("escalate from the other goroutine: %v", err)
		}
	})

	stdout, _, elapsed, err := runOrgCmdStreamsWithin(t, 5*time.Second, "wait", "--inbox", "--timeout-ms", "10000", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("wait --inbox: %v", err)
	}
	if want := "e1\torg-a\tRESULT\tPR opened\n"; stdout != want {
		t.Errorf("stdout = %q; want %q", stdout, want)
	}
	if elapsed < 150*time.Millisecond {
		t.Errorf("wait --inbox returned after %v, before the item was escalated", elapsed)
	}
}

// AC6: no open item within --timeout-ms exits 1, including when the only
// item is acked.
func TestOrgWaitInbox_TimesOut(t *testing.T) {
	cases := []struct {
		name string
		seed func(t *testing.T, stateDir string)
	}{
		{"empty_inbox", func(*testing.T, string) {}},
		{"acked_item_only", func(t *testing.T, stateDir string) {
			escalateOK(t, stateDir, "org-a", inboxQuestion("which?"))
			runInboxOK(t, "inbox", "ack", "e1", "--state-dir", stateDir)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setInboxPollIntervalOverride(t, 20*time.Millisecond)
			stateDir := t.TempDir()
			tc.seed(t, stateDir)

			stdout, _, elapsed, err := runOrgCmdStreamsWithin(t, 5*time.Second, "wait", "--inbox", "--timeout-ms", "300", "--state-dir", stateDir)
			if err == nil || !strings.Contains(err.Error(), "no open inbox item arrived within 300 ms") {
				t.Fatalf("error = %v; want a timeout after 300 ms", err)
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			if elapsed < 300*time.Millisecond {
				t.Errorf("wait --inbox gave up after %v, before the 300ms timeout", elapsed)
			}
		})
	}
}

// AC6: --inbox refuses --seat, --until, and --org-id before reading
// anything; a seat wait still needs --org-id.
func TestOrgWaitInbox_RefusesSeatFlags(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	for _, tc := range []struct {
		args    []string
		wantSub string
	}{
		{[]string{"--seat", "seat-1"}, "--inbox cannot be combined with --seat"},
		{[]string{"--until", "idle"}, "--inbox cannot be combined with --until"},
		{[]string{"--org-id", "org-a"}, "--inbox cannot be combined with --org-id"},
	} {
		args := append([]string{"wait", "--inbox", "--timeout-ms", "0", "--state-dir", stateDir}, tc.args...)
		_, _, _, err := runOrgCmdStreamsWithin(t, 5*time.Second, args...)
		if err == nil || !strings.Contains(err.Error(), tc.wantSub) {
			t.Errorf("%v error = %v; want it to contain %q", tc.args, err, tc.wantSub)
		}
	}
	if _, err := os.Stat(stateDir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("a refused wait --inbox created the state dir (stat err %v)", err)
	}
	if _, _, err := runOrgCmdStreams(t, "wait", "--seat", "seat-1", "--state-dir", stateDir); err == nil || !strings.Contains(err.Error(), "--org-id is required") {
		t.Errorf("wait --seat without --org-id error = %v; want --org-id is required", err)
	}
}

// AC7: an item escalated from a linked worktree lands in the ledger shared
// with the main checkout, which reads it without --state-dir; --state-dir
// picks another ledger's inbox.
func TestOrgInbox_LinkedWorktreeSharesTheMainCheckoutsInbox(t *testing.T) {
	r := newLegacyLedgerRepo(t)

	stdout, _ := runInboxOK(t, "escalate", "--org-id", "org-a", "--text", inboxQuestion("from the worktree"))
	if stdout != "escalated e1 (org=org-a type=QUESTION)\n" {
		t.Fatalf("escalate from the worktree stdout = %q", stdout)
	}
	if got := inboxFileOrAbsent(t, r.sharedDir); got == "<absent>" {
		t.Fatalf("no inbox in the shared ledger %s", r.sharedDir)
	}
	if got := inboxFileOrAbsent(t, r.legacyDir); got != "<absent>" {
		t.Errorf("the worktree got its own inbox in %s: %s", r.legacyDir, got)
	}

	t.Chdir(r.mainRoot)
	stdout, _ = runInboxOK(t, "inbox", "--json")
	var payload orgInboxJSON
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("inbox --json from the main checkout: %v\n%s", err, stdout)
	}
	if ids := cliInboxIDs(payload.Items); !slices.Equal(ids, []string{"e1"}) {
		t.Errorf("inbox from the main checkout = %v; want e1", ids)
	}
	stdout, _, _, err := runOrgCmdStreamsWithin(t, 5*time.Second, "wait", "--inbox", "--timeout-ms", "1000")
	if err != nil || stdout != "e1\torg-a\tQUESTION\tfrom the worktree\n" {
		t.Errorf("wait --inbox from the main checkout = %q, %v; want e1", stdout, err)
	}

	other := filepath.Join(t.TempDir(), "other")
	stdout, _ = runInboxOK(t, "inbox", "--state-dir", other)
	if stdout != "no open or acked inbox items\n" {
		t.Errorf("inbox --state-dir <other> = %q; want the other ledger's empty inbox", stdout)
	}
	escalateOK(t, other, "org-b", inboxQuestion("in the other ledger"))
	runInboxOK(t, "inbox", "ack", "e1", "--state-dir", other)
	if got := cliInboxItem(t, other, "e1"); got.OrgID != "org-b" || got.State != org.InboxStateAcked {
		t.Errorf("other ledger e1 = %+v; want org-b acked", got)
	}
	shared := readCLIInbox(t, r.sharedDir)
	if len(shared.Items) != 1 || shared.Items[0].OrgID != "org-a" || shared.Items[0].State != org.InboxStateOpen {
		t.Errorf("shared ledger = %+v; want only its own e1, still open", shared.Items)
	}
}

// escalate, ack, resolve, and notify change the ledger, so a legacy
// worktree ledger with active seats refuses them; the list, show, and wait
// --inbox only read it and print the note.
func TestOrgInbox_LegacyLedgerGuard(t *testing.T) {
	setupOrgStubPATH(t)
	r := newLegacyLedgerRepo(t)
	escalateOK(t, r.sharedDir, "org-a", inboxQuestion("seeded"))
	spawnLedgerSeat(t, r.legacyDir, "seat-1")
	before := readFileBytes(t, org.InboxPathIn(r.sharedDir))

	for _, args := range [][]string{
		{"escalate", "--org-id", "org-a", "--text", inboxQuestion("refused")},
		{"inbox", "ack", "e1"},
		{"inbox", "resolve", "e1", "--note", "docs/reports/x.md"},
		{"inbox", "notify", "e1"},
	} {
		stdout, _, err := runOrgCmdStreams(t, args...)
		if err == nil || !strings.Contains(err.Error(), "refusing to change the org ledger") {
			t.Errorf("%v error = %v; want the legacy-ledger refusal", args, err)
		}
		if stdout != "" {
			t.Errorf("%v stdout = %q; want nothing", args, stdout)
		}
	}
	if after := readFileBytes(t, org.InboxPathIn(r.sharedDir)); !bytes.Equal(before, after) {
		t.Errorf("a refused verb wrote to the inbox:\nbefore: %s\nafter:  %s", before, after)
	}

	for _, args := range [][]string{{"inbox"}, {"inbox", "show", "e1"}, {"wait", "--inbox", "--timeout-ms", "1000"}} {
		stdout, stderr, _, err := runOrgCmdStreamsWithin(t, 5*time.Second, args...)
		if err != nil {
			t.Errorf("%v: %v (stderr: %s)", args, err, stderr)
			continue
		}
		if n := strings.Count(stderr, legacyNoteMarker); n != 1 {
			t.Errorf("%v printed the legacy-ledger note %d time(s): %s", args, n, stderr)
		}
		if !strings.Contains(stdout, "e1") {
			t.Errorf("%v stdout = %q; want e1 from the shared ledger", args, stdout)
		}
	}
}
