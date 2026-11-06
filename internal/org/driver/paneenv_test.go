package driver

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// TestSeatPaneEnv_ListsBothEntriesInOrder pins AC2 of plan
// 2026-10-22-seat-wip-checkpoint: the panes ralph creates get
// AGMSG_SELF_NAME=off, then RALPH_WIP_CHECKPOINT=off.
func TestSeatPaneEnv_ListsBothEntriesInOrder(t *testing.T) {
	if want := []string{"AGMSG_SELF_NAME=off", "RALPH_WIP_CHECKPOINT=off"}; !reflect.DeepEqual(SeatPaneEnv(), want) {
		t.Fatalf("SeatPaneEnv() = %q; want %q", SeatPaneEnv(), want)
	}
	if WIPCheckpointEnv != "RALPH_WIP_CHECKPOINT" || WIPCheckpointOff != "RALPH_WIP_CHECKPOINT=off" {
		t.Fatalf("WIPCheckpointEnv, WIPCheckpointOff = %q, %q; want RALPH_WIP_CHECKPOINT, RALPH_WIP_CHECKPOINT=off",
			WIPCheckpointEnv, WIPCheckpointOff)
	}
}

// TestSeatPaneEnv_ReturnsACopy: a caller that changes the slice it got
// changes nothing a later call returns.
func TestSeatPaneEnv_ReturnsACopy(t *testing.T) {
	got := SeatPaneEnv()
	for i := range got {
		got[i] = "CHANGED=1"
	}
	if want := []string{"AGMSG_SELF_NAME=off", "RALPH_WIP_CHECKPOINT=off"}; !reflect.DeepEqual(SeatPaneEnv(), want) {
		t.Fatalf("SeatPaneEnv() after a caller changed its slice = %q; want %q", SeatPaneEnv(), want)
	}
}

// TestWIPCheckpointOff_IsWhatTheHooksRead: both WIP-checkpoint hooks, and
// their copies under templates/base, skip their commit on exactly the
// variable and value of WIPCheckpointOff, so the entry ralph adds to its
// panes is the one they read.
func TestWIPCheckpointOff_IsWhatTheHooksRead(t *testing.T) {
	name, value, ok := strings.Cut(WIPCheckpointOff, "=")
	if !ok || name != WIPCheckpointEnv {
		t.Fatalf("WIPCheckpointOff = %q; want %s=<value>", WIPCheckpointOff, WIPCheckpointEnv)
	}
	check := `[ "${` + name + `:-}" = "` + value + `" ]`
	root := filepath.Join("..", "..", "..")
	for _, dir := range []string{filepath.Join(root, ".claude", "hooks"), filepath.Join(root, "templates", "base", ".claude", "hooks")} {
		for _, hook := range []string{"session_end_summary.sh", "precompact_checkpoint.sh"} {
			path := filepath.Join(dir, hook)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			if !strings.Contains(string(data), check) {
				t.Fatalf("%s does not test %s", path, check)
			}
		}
	}
}

// TestClaudeMaxRetriesForMinutes_FollowsTheMeasuredFormula pins the
// conversion of plan 2026-11-07-org-seat-resume, D4:
// max(1, 10 + round((m*60 - 189) / 36)), with the values of
// docs/evidence/org-seat-resume-retries-2026-11-07.md. 1, 30 and 1440 tell
// rounding from truncation, and the negative value shows the floor of 1.
func TestClaudeMaxRetriesForMinutes_FollowsTheMeasuredFormula(t *testing.T) {
	for _, tc := range []struct{ minutes, want int }{
		{1, 6},
		{2, 8},
		{3, 10},
		{30, 55},
		{60, 105},
		{1440, 2405},
		{-10, 1},
	} {
		if got := ClaudeMaxRetriesForMinutes(tc.minutes); got != tc.want {
			t.Errorf("ClaudeMaxRetriesForMinutes(%d) = %d; want %d", tc.minutes, got, tc.want)
		}
	}
}

// TestClaudeRetryEnv_EntriesForTheMinutes pins FR-8 of spec
// docs/specs/2026-11-07-org-seat-resume.md at the driver: the variable names,
// and, for a setting of 1 or more, the watchdog, its wait in milliseconds,
// and the retry count, in that order; 0 and a negative value give nil.
func TestClaudeRetryEnv_EntriesForTheMinutes(t *testing.T) {
	if ClaudeRetryWatchdogEnv != "CLAUDE_CODE_RETRY_WATCHDOG" || ClaudeRetryWatchdogMaxWaitMSEnv != "CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS" ||
		ClaudeMaxRetriesEnv != "CLAUDE_CODE_MAX_RETRIES" {
		t.Fatalf("variable names = %q, %q, %q; want CLAUDE_CODE_RETRY_WATCHDOG, CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS, CLAUDE_CODE_MAX_RETRIES",
			ClaudeRetryWatchdogEnv, ClaudeRetryWatchdogMaxWaitMSEnv, ClaudeMaxRetriesEnv)
	}
	for _, tc := range []struct {
		minutes int
		want    []string
	}{
		{0, nil},
		{-1, nil},
		{1, []string{"CLAUDE_CODE_RETRY_WATCHDOG=1", "CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS=60000", "CLAUDE_CODE_MAX_RETRIES=6"}},
		{30, []string{"CLAUDE_CODE_RETRY_WATCHDOG=1", "CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS=1800000", "CLAUDE_CODE_MAX_RETRIES=55"}},
	} {
		got := ClaudeRetryEnv(tc.minutes)
		if !reflect.DeepEqual(got, tc.want) || (tc.want == nil) != (got == nil) {
			t.Errorf("ClaudeRetryEnv(%d) = %#v; want %#v", tc.minutes, got, tc.want)
		}
	}
}

// TestClaudeRetryEnv_ReturnsANewSliceEachCall: a caller that changes the
// slice it got changes nothing a later call returns.
func TestClaudeRetryEnv_ReturnsANewSliceEachCall(t *testing.T) {
	got := ClaudeRetryEnv(30)
	for i := range got {
		got[i] = "CHANGED=1"
	}
	want := []string{"CLAUDE_CODE_RETRY_WATCHDOG=1", "CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS=1800000", "CLAUDE_CODE_MAX_RETRIES=55"}
	if again := ClaudeRetryEnv(30); !reflect.DeepEqual(again, want) {
		t.Fatalf("ClaudeRetryEnv(30) after a caller changed its slice = %q; want %q", again, want)
	}
}

// The clauses warnNoPaneEnv writes for the entries of ClaudeRetryEnv,
// written out here so that a change to paneEnvConsequence's text fails the
// tests.
const (
	claudeRetryWatchdogClause = "a Claude Code session in such a pane (a claude seat's or a claude headless director's) ends its turn " +
		"on a rate or usage limit (429/529) or a connection failure after its default retries " +
		"(about 3 minutes for a connection failure) instead of waiting out up to [org].api_wait_minutes"
	claudeRetryMaxWaitClause = "a Claude Code session in such a pane that takes CLAUDE_CODE_RETRY_WATCHDOG=1 from the herdr server's environment " +
		"does not stop waiting on a rate or usage limit (429/529) after [org].api_wait_minutes"
	claudeMaxRetriesClause = "a Claude Code session in such a pane that takes CLAUDE_CODE_RETRY_WATCHDOG=1 from the herdr server's environment " +
		"does not stop retrying a connection failure after about [org].api_wait_minutes"
)

// TestPaneEnvConsequence_ClaudeRetryEntries: each variable of ClaudeRetryEnv
// has its own clause, whatever its value, matched by NAME= so that the
// watchdog's clause is not the one of its MAX_WAIT_MS entry; a longer name
// that starts with one of them, and a name with no "=", have none.
func TestPaneEnvConsequence_ClaudeRetryEntries(t *testing.T) {
	for entry, want := range map[string]string{
		"CLAUDE_CODE_RETRY_WATCHDOG=1":                    claudeRetryWatchdogClause,
		"CLAUDE_CODE_RETRY_WATCHDOG=":                     claudeRetryWatchdogClause,
		"CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS=1800000":  claudeRetryMaxWaitClause,
		"CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS=60000":    claudeRetryMaxWaitClause,
		"CLAUDE_CODE_MAX_RETRIES=55":                      claudeMaxRetriesClause,
		"CLAUDE_CODE_MAX_RETRIES=6":                       claudeMaxRetriesClause,
		"CLAUDE_CODE_RETRY_WATCHDOG_X=1":                  "",
		"CLAUDE_CODE_MAX_RETRIES_X=1":                     "",
		"CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS_X=1":      "",
		"CLAUDE_CODE_MAX_RETRIES":                         "",
		"X_CLAUDE_CODE_RETRY_WATCHDOG=1":                  "",
		"CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MSX=1800000": "",
	} {
		if got := paneEnvConsequence(entry); got != want {
			t.Errorf("paneEnvConsequence(%q) =\n%q\nwant\n%q", entry, got, want)
		}
	}
	for _, entry := range ClaudeRetryEnv(30) {
		if paneEnvConsequence(entry) == "" {
			t.Errorf("paneEnvConsequence(%q) is empty; want the clause of its variable", entry)
		}
	}
}

// TestHerdr_TabCreate_PaneEnvCarriesTheClaudeRetryEntries: with a herdr
// whose help lists --env, the extraEnv spawn passes for a claude seat with
// [org].api_wait_minutes 30 becomes one --env each, after SeatPaneEnv's,
// in its order.
func TestHerdr_TabCreate_PaneEnvCarriesTheClaudeRetryEntries(t *testing.T) {
	f := &paneEnvFakeRunner{help: herdrTabCreateHelp}
	var warn bytes.Buffer
	h := NewHerdr(f, SeatPaneEnv()...)
	h.Warn = &warn

	if _, err := h.TabCreate(context.Background(), "w3", "/tmp/cwd", "worker-1", append([]string{PrimaryCLIEntry("claude")}, ClaudeRetryEnv(30)...)...); err != nil {
		t.Fatalf("TabCreate: %v", err)
	}
	want := append(slices.Clone(tabCreateArgv), "--env", "AGMSG_SELF_NAME=off", "--env", "RALPH_WIP_CHECKPOINT=off", "--env", "RALPH_PRIMARY_CLI=claude",
		"--env", "CLAUDE_CODE_RETRY_WATCHDOG=1", "--env", "CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS=1800000", "--env", "CLAUDE_CODE_MAX_RETRIES=55")
	if !reflect.DeepEqual(f.tabCalls, [][]string{want}) {
		t.Fatalf("tab create argv = %q; want %q", f.tabCalls, want)
	}
	if warn.Len() != 0 {
		t.Fatalf("warning = %q; want none", warn.String())
	}
}

// TestHerdr_TabCreate_PaneEnvWarningNamesTheClaudeRetryClauses: on a herdr
// without --env, the one warning line for a claude seat's pane lists every
// entry and adds the clause of each, the three ClaudeRetryEnv clauses last,
// in their order.
func TestHerdr_TabCreate_PaneEnvWarningNamesTheClaudeRetryClauses(t *testing.T) {
	f := &paneEnvFakeRunner{help: herdrTabCreateHelpNoEnv}
	var warn bytes.Buffer
	h := NewHerdr(f, SeatPaneEnv()...)
	h.Warn = &warn

	tabCreateTwice(t, h, append([]string{PrimaryCLIEntry("claude")}, ClaudeRetryEnv(30)...)...)

	if !reflect.DeepEqual(f.tabCalls, [][]string{tabCreateArgv, tabCreateArgv}) {
		t.Fatalf("tab create argv = %v; want %v twice, with no --env", f.tabCalls, tabCreateArgv)
	}
	want := "warning: herdr tab create has no --env flag, so ralph creates herdr panes without " +
		"--env AGMSG_SELF_NAME=off --env RALPH_WIP_CHECKPOINT=off --env RALPH_PRIMARY_CLI=claude " +
		"--env CLAUDE_CODE_RETRY_WATCHDOG=1 --env CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS=1800000 --env CLAUDE_CODE_MAX_RETRIES=55: " +
		agmsgSelfNameClause + "; " + wipCheckpointClause + "; " + primaryCLIClause + "; " +
		claudeRetryWatchdogClause + "; " + claudeRetryMaxWaitClause + "; " + claudeMaxRetriesClause + noPaneEnvAdvice
	if warn.String() != want {
		t.Fatalf("warning =\n%q\nwant exactly one line\n%q", warn.String(), want)
	}
}
