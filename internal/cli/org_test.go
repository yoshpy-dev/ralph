package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// runOrgCmd runs `ralph org <args...>` in-process and returns combined
// stdout/stderr plus any error from Execute() (non-nil means the CLI would
// exit non-zero, per cmd/ralph/main.go).
func runOrgCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(append([]string{"org"}, args...))
	err := root.Execute()
	return buf.String(), err
}

// herdrStub is a POSIX-sh fake binary: it logs its argv to a file (path
// from an env var) and emits canned output, so the org verbs exercise the
// real driver.ExecRunner -> exec.Command path end to end without needing
// herdr installed (CI has none). ORG_STUB_FAIL, set per test case, injects
// a failure at exactly one subcommand boundary.
//
// ORG_STUB_AGENT_WAIT_CONFIRM_FAIL, when set, fails only the send verb's
// post-Enter *confirm* AgentWait call -- distinguished from its idle/done
// wait call by grepping argv for "working" (driver.Herdr.AgentWait passes
// each `until` state as its own `--until <state>` pair, and only the
// confirm call ever waits on "working"). This is a separate mechanism from
// ORG_STUB_FAIL=agent:wait (which would fail *every* "agent wait" call,
// including the idle/done wait Send needs to succeed before it ever
// reaches Enter) -- it exists so CLI-level tests can exercise the
// unconfirmed-submit warning path without also failing the send outright.
//
// ORG_STUB_PANE_SEND_TEXT_SLEEP, when set, makes the stub `sleep` that many
// seconds (`sleep`'s own argument format, e.g. "0.6") before answering a
// "pane send-text" call -- simulating a real herdr round trip that eats
// into the ctx budget between Send's fail-closed pre-Enter-pause budget
// check (which measures ctx time remaining right before PaneSendText) and
// the actual pre-Enter wait. This is the CLI-level (real subprocess)
// counterpart of fakeHerdr.paneSendTextDelay in
// internal/org/spawn_test.go, needed because the CLI drives Send through a
// real subprocess herdr with no other way to inject a controlled delay
// (AR-1 CLI coverage, docs/reports/cross-review-triage-org-send-enter-timing.md).
const herdrStub = `#!/bin/sh
if [ -n "$ORG_HERDR_LOG" ]; then
  echo "$@" >> "$ORG_HERDR_LOG"
fi
if [ -n "$ORG_STUB_FAIL" ] && [ "$1:$2" = "$ORG_STUB_FAIL" ]; then
  echo "stub failure: $1 $2" >&2
  exit 1
fi
if [ "$1 $2" = "agent wait" ] && [ -n "$ORG_STUB_AGENT_WAIT_CONFIRM_FAIL" ]; then
  for arg in "$@"; do
    if [ "$arg" = "working" ]; then
      echo "stub failure: agent wait confirm" >&2
      exit 1
    fi
  done
fi
if [ "$1 $2" = "pane send-text" ] && [ -n "$ORG_STUB_PANE_SEND_TEXT_SLEEP" ]; then
  sleep "$ORG_STUB_PANE_SEND_TEXT_SLEEP"
fi
case "$1 $2" in
  "workspace create") echo '{"id":"cli:workspace:create","result":{"root_pane":{"pane_id":"ws-stub-1:p1","tab_id":"ws-stub-1:t1","workspace_id":"ws-stub-1"},"tab":{"tab_id":"ws-stub-1:t1"},"type":"workspace_created","workspace":{"active_tab_id":"ws-stub-1:t1","workspace_id":"ws-stub-1"}}}' ;;
  "tab create") echo '{"id":"cli:tab:create","result":{"root_pane":{"pane_id":"pane-stub-1","tab_id":"ws-stub-1:t2","workspace_id":"ws-stub-1"},"tab":{"tab_id":"ws-stub-1:t2"},"type":"tab_created"}}' ;;
  "agent start") echo "agent-stub-1" ;;
  "agent wait") echo "idle" ;;
  "pane read") echo "pane output" ;;
  *) echo "" ;;
esac
exit 0
`

// agmsgSendStub is the fake `scripts/send.sh` that lives under a temp
// agmsg-home directory (see setupOrgStubPATH). Unlike the old PR① fake
// (a single `agmsg` binary on PATH parsing --team/--as flags), the real
// agmsg interface is a script collection: driver.Agmsg shells out to
// `bash <home>/scripts/send.sh TEAM FROM TO MESSAGE`, so this stub only
// needs to implement send.sh's own contract (log argv, honor
// ORG_STUB_FAIL=agmsg:send).
const agmsgSendStub = `#!/bin/sh
if [ -n "$ORG_AGMSG_LOG" ]; then
  echo "$@" >> "$ORG_AGMSG_LOG"
fi
if [ "$ORG_STUB_FAIL" = "agmsg:send" ]; then
  echo "stub failure: agmsg send" >&2
  exit 1
fi
echo ""
exit 0
`

// agmsgJoinStub is the fake `scripts/join.sh` -- driver.Agmsg shells out to
// `bash <home>/scripts/join.sh TEAM AGENT_ID TYPE PROJECT_PATH`. Honors
// ORG_STUB_FAIL=agmsg:join so tests can inject a failure at either the
// ensureLeadJoined (agent_id "lead") or seat Join call.
const agmsgJoinStub = `#!/bin/sh
if [ -n "$ORG_AGMSG_LOG" ]; then
  echo "$@" >> "$ORG_AGMSG_LOG"
fi
if [ "$ORG_STUB_FAIL" = "agmsg:join" ]; then
  echo "stub failure: agmsg join" >&2
  exit 1
fi
echo ""
exit 0
`

// agmsgLeaveStub is the fake `scripts/leave.sh` -- driver.Agmsg shells
// out to `bash <home>/scripts/leave.sh TEAM AGENT_ID`. Honors
// ORG_STUB_FAIL=agmsg:leave.
const agmsgLeaveStub = `#!/bin/sh
if [ -n "$ORG_AGMSG_LOG" ]; then
  echo "$@" >> "$ORG_AGMSG_LOG"
fi
if [ "$ORG_STUB_FAIL" = "agmsg:leave" ]; then
  echo "stub failure: agmsg leave" >&2
  exit 1
fi
echo ""
exit 0
`

// agmsgVersionStub is the plain-text VERSION marker AgmsgAvailable/
// AgmsgVersion read from an agmsg home directory.
const agmsgVersionStub = "1.1.13\n"

// setupOrgStubPATH writes a stub herdr binary to a fresh temp dir and
// prepends it to PATH, and writes a stub agmsg-home directory
// (scripts/send.sh + VERSION) pointed at via RALPH_ORG_AGMSG_HOME -- the
// real agmsg interface is a script collection under an "agmsg home", not a
// PATH-resident binary. Returns the log file paths so tests can assert on
// recorded argv.
func setupOrgStubPATH(t *testing.T) (herdrLog, agmsgLog string) {
	t.Helper()
	dir := t.TempDir()

	writeStubScript(t, filepath.Join(dir, "herdr"), herdrStub)
	herdrLog = filepath.Join(dir, "herdr.log")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ORG_HERDR_LOG", herdrLog)

	agmsgHome := t.TempDir()
	if err := os.MkdirAll(filepath.Join(agmsgHome, "scripts"), 0o755); err != nil {
		t.Fatalf("mkdir agmsg scripts dir: %v", err)
	}
	writeStubScript(t, filepath.Join(agmsgHome, "scripts", "send.sh"), agmsgSendStub)
	writeStubScript(t, filepath.Join(agmsgHome, "scripts", "join.sh"), agmsgJoinStub)
	writeStubScript(t, filepath.Join(agmsgHome, "scripts", "leave.sh"), agmsgLeaveStub)
	if err := os.WriteFile(filepath.Join(agmsgHome, "VERSION"), []byte(agmsgVersionStub), 0o644); err != nil {
		t.Fatalf("write agmsg VERSION: %v", err)
	}
	agmsgLog = filepath.Join(dir, "agmsg.log")
	t.Setenv("RALPH_ORG_AGMSG_HOME", agmsgHome)
	t.Setenv("ORG_AGMSG_LOG", agmsgLog)

	t.Setenv("ORG_STUB_FAIL", "")
	t.Setenv("ORG_STUB_AGENT_WAIT_CONFIRM_FAIL", "")

	return herdrLog, agmsgLog
}

func writeStubScript(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write stub %s: %v", path, err)
	}
}

// readLogLines reads a stub log file, tolerating "not yet created" (a stub
// that was never invoked leaves no log file).
func readLogLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read log %s: %v", path, err)
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func countLinesWithPrefix(lines []string, prefix string) int {
	n := 0
	for _, l := range lines {
		if strings.HasPrefix(l, prefix) {
			n++
		}
	}
	return n
}

// writeOrgConfig writes a minimal ralph.toml with an [org] section
// constraining max_seats/driver_pool/model_pool, for tests that need
// tighter envelope bounds than config.Default().Org provides.
func writeOrgConfig(t *testing.T, dir string, maxSeats int) string {
	t.Helper()
	path := filepath.Join(dir, "ralph.toml")
	content := "[org]\n" +
		"max_seats = " + itoa(maxSeats) + "\n" +
		"driver_pool = [\"claude\"]\n\n" +
		"[[org.model_pool]]\n" +
		"driver = \"claude\"\n" +
		"model = \"sonnet\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write org config: %v", err)
	}
	return path
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func readManifestEvents(t *testing.T, path string) []org.ManifestEvent {
	t.Helper()
	store := org.NewManifestStoreAtPath(path)
	rr, err := store.Read()
	if err != nil {
		t.Fatalf("read manifest %s: %v", path, err)
	}
	return rr.Events
}

func eventTypes(events []org.ManifestEvent) []string {
	out := make([]string, len(events))
	for i, ev := range events {
		out[i] = ev.Event
	}
	return out
}

func TestOrgSpawn_HappyPath_EventSequenceReceiptAndWorkspaceReuse(t *testing.T) {
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	)
	if err != nil {
		t.Fatalf("spawn seat-1 failed: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "spawned seat \"seat-1\"") {
		t.Errorf("expected spawned confirmation in output, got: %s", out)
	}

	manifestPath := org.ManifestPathIn(stateDir)
	events := readManifestEvents(t, manifestPath)
	// spawn_step x5: tab_created, agent_started, agmsg_lead_joined,
	// agmsg_joined, agmsg_announced.
	want := []string{"spawn_started", "org_workspace_created", "spawn_step", "spawn_step", "spawn_step", "spawn_step", "spawn_step", "spawned"}
	if got := eventTypes(events); !equalStrings(got, want) {
		t.Fatalf("expected event sequence %v, got %v", want, got)
	}

	receiptsPath := org.ReceiptsPathIn(stateDir)
	receiptStore := org.NewReceiptStoreAtPath(receiptsPath)
	rr, err := receiptStore.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	if len(rr.Receipts) != 1 || rr.Receipts[0].Honored != org.HonoredUnknown {
		t.Fatalf("expected 1 receipt honored=unknown, got %+v", rr.Receipts)
	}

	// Second seat in the same org_id must reuse the workspace: exactly one
	// "workspace create" call across both spawns.
	out2, err2 := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-2", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	)
	if err2 != nil {
		t.Fatalf("spawn seat-2 failed: %v (output: %s)", err2, out2)
	}

	herdrLines := readLogLines(t, herdrLog)
	if n := countLinesWithPrefix(herdrLines, "workspace create"); n != 1 {
		t.Fatalf("expected exactly 1 'workspace create' call across 2 seats in the same org, got %d (log: %v)", n, herdrLines)
	}
	if n := countLinesWithPrefix(herdrLines, "tab create"); n != 2 {
		t.Fatalf("expected 2 'tab create' calls (one per seat), got %d (log: %v)", n, herdrLines)
	}

	// Each agmsg invocation's HELLO message body itself contains newlines,
	// so raw line count isn't the invocation count -- count by the
	// "ralph-<org_id>" team-name prefix (agmsgTeam's convention) that
	// starts every join.sh/send.sh call's argv instead. Per seat spawn: one
	// ensureLeadJoined join.sh call, one seat join.sh call, one send.sh
	// HELLO call -- 3 invocations x 2 seats = 6.
	agmsgLines := readLogLines(t, agmsgLog)
	if n := countLinesWithPrefix(agmsgLines, "ralph-org-a"); n != 6 {
		t.Fatalf("expected 6 agmsg invocations (join lead + join seat + send HELLO, per seat), got %d (log: %v)", n, agmsgLines)
	}
}

// TestOrgSpawn_LeadDriverFlag_DefaultsToClaudeCode pins AC-7's CLI wiring:
// omitting --lead-driver registers the lead identity with agmsg type
// "claude-code" (the flag's own default, "claude").
func TestOrgSpawn_LeadDriverFlag_DefaultsToClaudeCode(t *testing.T) {
	_, agmsgLog := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	)
	if err != nil {
		t.Fatalf("spawn failed: %v (output: %s)", err, out)
	}

	agmsgLines := readLogLines(t, agmsgLog)
	// join.sh argv: TEAM AGENT_ID TYPE PROJECT_PATH -- the lead Join is the
	// first agmsg invocation and must carry "claude-code" (the
	// --lead-driver flag's own default, "claude").
	if len(agmsgLines) == 0 || !strings.Contains(agmsgLines[0], "ralph-org-a lead claude-code") {
		t.Fatalf("expected the first agmsg call to be lead Join with type claude-code, got %v", agmsgLines)
	}
}

// TestOrgSpawn_LeadDriverFlag_Codex_RegistersLeadAsCodexType covers the
// explicit --lead-driver=codex case: the lead identity's agmsg type must
// follow --lead-driver, independent of the seat's own --driver.
func TestOrgSpawn_LeadDriverFlag_Codex_RegistersLeadAsCodexType(t *testing.T) {
	_, agmsgLog := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--lead-driver", "codex",
	)
	if err != nil {
		t.Fatalf("spawn failed: %v (output: %s)", err, out)
	}

	agmsgLines := readLogLines(t, agmsgLog)
	if len(agmsgLines) == 0 || !strings.Contains(agmsgLines[0], "ralph-org-a lead codex") {
		t.Fatalf("expected the first agmsg call to be lead Join with type codex, got %v", agmsgLines)
	}
	// The seat's own Join (second call) must still use its own --driver
	// (claude -> claude-code), unaffected by --lead-driver.
	if len(agmsgLines) < 2 || !strings.Contains(agmsgLines[1], "ralph-org-a seat-1 claude-code") {
		t.Fatalf("expected the second agmsg call to be the seat's own Join with type claude-code, got %v", agmsgLines)
	}
}

func TestOrgSpawn_FailureInjection_TabCreate_NoCompensation(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	t.Setenv("ORG_STUB_FAIL", "tab:create")
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	)
	if err == nil {
		t.Fatalf("expected non-zero exit on tab_create failure, output: %s", out)
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	last := events[len(events)-1]
	if last.Event != "spawn_failed" {
		t.Fatalf("expected last event spawn_failed, got %q", last.Event)
	}
	if !strings.Contains(last.Details, "step=tab_create") {
		t.Errorf("expected Details to mention step=tab_create, got %q", last.Details)
	}
	if !strings.Contains(last.Details, "no pane to compensate") {
		t.Errorf("expected Details to record no compensation attempted (no pane existed yet), got %q", last.Details)
	}

	herdrLines := readLogLines(t, herdrLog)
	if countLinesWithPrefix(herdrLines, "pane send-keys") != 0 {
		t.Errorf("expected no compensation send-keys call when tab_create itself failed, got log: %v", herdrLines)
	}
}

func TestOrgSpawn_FailureInjection_AgentStart_CompensatesPane(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	t.Setenv("ORG_STUB_FAIL", "agent:start")
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	)
	if err == nil {
		t.Fatalf("expected non-zero exit on agent_start failure, output: %s", out)
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	last := events[len(events)-1]
	if last.Event != "spawn_failed" {
		t.Fatalf("expected last event spawn_failed, got %q", last.Event)
	}
	if !strings.Contains(last.Details, "step=agent_start") {
		t.Errorf("expected Details to mention step=agent_start, got %q", last.Details)
	}
	if last.PaneID == "" {
		t.Error("expected the orphaned pane_id to remain traceable on the spawn_failed event")
	}

	herdrLines := readLogLines(t, herdrLog)
	if n := countLinesWithPrefix(herdrLines, "pane send-keys"); n != 1 {
		t.Fatalf("expected exactly 1 compensation send-keys call (a pane existed), got %d (log: %v)", n, herdrLines)
	}
	if !strings.Contains(herdrLines[len(herdrLines)-1], "C-c") {
		t.Errorf("expected compensation to send C-c, got last herdr call: %q", herdrLines[len(herdrLines)-1])
	}
}

func TestOrgSpawn_FailureInjection_AgmsgSend_CompensatesPane(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	t.Setenv("ORG_STUB_FAIL", "agmsg:send")
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	)
	if err == nil {
		t.Fatalf("expected non-zero exit on agmsg send failure, output: %s", out)
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	last := events[len(events)-1]
	if last.Event != "spawn_failed" {
		t.Fatalf("expected last event spawn_failed, got %q", last.Event)
	}
	if !strings.Contains(last.Details, "step=agmsg_announce") {
		t.Errorf("expected Details to mention step=agmsg_announce, got %q", last.Details)
	}
	// AC-6 tech-debt fix: a failed HELLO announce must best-effort Leave the
	// seat's own (already-succeeded) Join back out, and record the outcome.
	if !strings.Contains(last.Details, "leave=ok") {
		t.Errorf("expected Details to record leave=ok, got %q", last.Details)
	}

	herdrLines := readLogLines(t, herdrLog)
	if n := countLinesWithPrefix(herdrLines, "pane send-keys"); n != 1 {
		t.Fatalf("expected exactly 1 compensation send-keys call, got %d (log: %v)", n, herdrLines)
	}
}

func TestOrgSpawn_FailureInjection_AgmsgJoin_CompensatesPane(t *testing.T) {
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	t.Setenv("ORG_STUB_FAIL", "agmsg:join")
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	)
	if err == nil {
		t.Fatalf("expected non-zero exit on agmsg join failure, output: %s", out)
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	last := events[len(events)-1]
	if last.Event != "spawn_failed" {
		t.Fatalf("expected last event spawn_failed, got %q", last.Event)
	}
	// ensureLeadJoined's join.sh call also fails (same stub, best-effort --
	// swallowed) before the hard-failing seat Join call, so the terminal
	// spawn_failed step is agmsg_join, not agmsg_lead_joined.
	if !strings.Contains(last.Details, "step=agmsg_join") {
		t.Errorf("expected Details to mention step=agmsg_join, got %q", last.Details)
	}

	herdrLines := readLogLines(t, herdrLog)
	if n := countLinesWithPrefix(herdrLines, "pane send-keys"); n != 1 {
		t.Fatalf("expected exactly 1 compensation send-keys call, got %d (log: %v)", n, herdrLines)
	}

	agmsgLines := readLogLines(t, agmsgLog)
	if n := countLinesWithPrefix(agmsgLines, "ralph-org-a"); n != 2 {
		t.Fatalf("expected exactly 2 agmsg join.sh invocations (lead then seat, both failing -- no send.sh reached), got %d (log: %v)", n, agmsgLines)
	}
}

func TestOrgSpawn_Rejection_ModelOutOfPoolAndMaxSeatsWithOrgIsolation(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	configDir := t.TempDir()
	configPath := writeOrgConfig(t, configDir, 1)

	// AC-1: model out of pool -> rejected, non-zero exit.
	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-bad-model", "--role", "worker",
		"--driver", "claude", "--model", "not-a-real-model", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--config", configPath,
	)
	if err == nil {
		t.Fatalf("expected non-zero exit for out-of-pool model, output: %s", out)
	}
	if !strings.Contains(out, "rejected") {
		t.Errorf("expected 'rejected' in output, got: %s", out)
	}

	receiptsPath := org.ReceiptsPathIn(stateDir)
	rr, err := org.NewReceiptStoreAtPath(receiptsPath).Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	if len(rr.Receipts) != 1 || rr.Receipts[0].Honored != org.HonoredFalse {
		t.Fatalf("expected 1 receipt honored=false for the rejection, got %+v", rr.Receipts)
	}

	// AC-2: max_seats=1 in org-a -- first real spawn succeeds, second is rejected.
	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--config", configPath,
	); err != nil {
		t.Fatalf("expected first org-a spawn to succeed under max_seats=1: %v", err)
	}

	out3, err3 := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-2", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--config", configPath,
	)
	if err3 == nil {
		t.Fatalf("expected non-zero exit: max_seats=1 already reached in org-a, output: %s", out3)
	}

	// Org isolation: org-b's first seat must not be blocked by org-a's count.
	if _, errB := runOrgCmd(t,
		"spawn", "--org-id", "org-b", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--config", configPath,
	); errB != nil {
		t.Fatalf("expected org-b's first seat to spawn despite org-a being at max_seats: %v", errB)
	}
}

func TestOrgSpawn_IdempotentRespawn_ExitZeroNoNewDriverCalls(t *testing.T) {
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	args := []string{
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	}
	if _, err := runOrgCmd(t, args...); err != nil {
		t.Fatalf("initial spawn failed: %v", err)
	}
	herdrCallsBefore := len(readLogLines(t, herdrLog))
	agmsgCallsBefore := len(readLogLines(t, agmsgLog))

	out, err := runOrgCmd(t, args...)
	if err != nil {
		t.Fatalf("expected exit 0 on idempotent respawn, got err=%v (output: %s)", err, out)
	}
	if !strings.Contains(out, "already spawned") {
		t.Errorf("expected 'already spawned' in output, got: %s", out)
	}

	if n := len(readLogLines(t, herdrLog)); n != herdrCallsBefore {
		t.Fatalf("expected no new herdr calls on idempotent respawn, %d -> %d", herdrCallsBefore, n)
	}
	if n := len(readLogLines(t, agmsgLog)); n != agmsgCallsBefore {
		t.Fatalf("expected no new agmsg calls on idempotent respawn, %d -> %d", agmsgCallsBefore, n)
	}
}

func TestOrgSpawn_DryRun_NoPATHNeeded_StatusExclusionAndAll(t *testing.T) {
	t.Setenv("PATH", "") // no herdr/agmsg lookup should ever be attempted
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--dry-run",
	)
	if err != nil {
		t.Fatalf("expected dry-run spawn to succeed with empty PATH, err=%v (output: %s)", err, out)
	}
	if !strings.Contains(out, "dry_run=true") {
		t.Errorf("expected dry_run=true in spawn output, got: %s", out)
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	for _, ev := range events {
		if !ev.DryRun {
			t.Errorf("expected every dry-run saga event to carry dry_run=true, got %+v", ev)
		}
	}

	statusOut, err := runOrgCmd(t, "status", "--org-id", "org-a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if !strings.Contains(statusOut, "no seats") {
		t.Errorf("expected dry-run seat excluded from default status, got: %s", statusOut)
	}

	statusAllOut, err := runOrgCmd(t, "status", "--org-id", "org-a", "--state-dir", stateDir, "--all")
	if err != nil {
		t.Fatalf("status --all failed: %v", err)
	}
	if !strings.Contains(statusAllOut, "seat-1") {
		t.Errorf("expected dry-run seat included with --all, got: %s", statusAllOut)
	}
}

// TestOrgSpawn_ModelFlagOmitted_DefaultsToFirstMatchingPoolEntry covers
// AC-5: `ralph org spawn` without --model no longer errors -- it resolves
// org.DefaultModelForDriverAndRole for --driver/--role (the first matching
// role-permitted [org].model_pool entry) and prints exactly one fallback warning line to
// stderr. Uses --dry-run (per the plan's "dry-run is fine for spawn") so no
// herdr/agmsg PATH lookup is needed.
func TestOrgSpawn_ModelFlagOmitted_DefaultsToFirstMatchingPoolEntry(t *testing.T) {
	t.Setenv("PATH", "") // no herdr/agmsg lookup should ever be attempted
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	configPath := filepath.Join(dir, "ralph.toml")
	content := "[org]\n" +
		"max_seats = 5\n" +
		"driver_pool = [\"claude\", \"codex\"]\n\n" +
		"[[org.model_pool]]\n" +
		"driver = \"codex\"\n" +
		"model = \"gpt-5.5-codex\"\n\n" +
		"[[org.model_pool]]\n" +
		"driver = \"claude\"\n" +
		"model = \"opus\"\n\n" +
		"[[org.model_pool]]\n" +
		"driver = \"claude\"\n" +
		"model = \"sonnet\"\n"
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--config", configPath, "--dry-run",
	)
	if err != nil {
		t.Fatalf("expected spawn without --model to succeed, err=%v (output: %s)", err, out)
	}
	wantWarn := "org: --model omitted; falling back to first [org].model_pool entry permitted for role worker on claude: opus (pass --model explicitly)"
	if n := strings.Count(out, wantWarn); n != 1 {
		t.Errorf("expected exactly one fallback warning line in output, got %d in: %s", n, out)
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	last := events[len(events)-1]
	// The first claude entry in declared model_pool order is "opus" (codex's
	// entry precedes it but does not match --driver claude).
	if last.Model != "opus" {
		t.Fatalf("expected default --model resolution to pick the first matching pool entry (opus), got %q", last.Model)
	}
}

// TestOrgSpawn_ModelFlagExplicit_NoWarning confirms an explicit --model
// wins over the fallback and prints no fallback warning line.
func TestOrgSpawn_ModelFlagExplicit_NoWarning(t *testing.T) {
	t.Setenv("PATH", "")
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "haiku", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--dry-run",
	)
	if err != nil {
		t.Fatalf("spawn failed: %v (output: %s)", err, out)
	}
	if strings.Contains(out, "--model omitted") {
		t.Errorf("expected no fallback warning when --model is explicit, got: %s", out)
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	last := events[len(events)-1]
	if last.Model != "haiku" {
		t.Fatalf("expected explicit --model to win over the default, got %q", last.Model)
	}
}

// TestOrgSpawn_ModelFlagOmitted_RespectsRoleAllowlist covers self-review
// MEDIUM-2: a role-restricted pool ([org.roles] implementer = ["sonnet"])
// whose driver head entry (claude/fable) is not permitted for the spawned
// role must fall back to the first role-permitted entry (sonnet), not
// warn-then-reject.
func TestOrgSpawn_ModelFlagOmitted_RespectsRoleAllowlist(t *testing.T) {
	t.Setenv("PATH", "")
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	configPath := filepath.Join(dir, "ralph.toml")
	content := "[org]\n" +
		"max_seats = 5\n" +
		"driver_pool = [\"claude\"]\n\n" +
		"[[org.model_pool]]\n" +
		"driver = \"claude\"\n" +
		"model = \"fable\"\n\n" +
		"[[org.model_pool]]\n" +
		"driver = \"claude\"\n" +
		"model = \"sonnet\"\n\n" +
		"[org.roles]\n" +
		"implementer = [\"sonnet\"]\n"
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "implementer",
		"--driver", "claude", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--config", configPath, "--dry-run",
	)
	if err != nil {
		t.Fatalf("expected spawn without --model to succeed under a role allowlist, err=%v (output: %s)", err, out)
	}
	wantWarn := "org: --model omitted; falling back to first [org].model_pool entry permitted for role implementer on claude: sonnet (pass --model explicitly)"
	if !strings.Contains(out, wantWarn) {
		t.Errorf("expected role-aware fallback warning, got: %s", out)
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	last := events[len(events)-1]
	if last.Model != "sonnet" {
		t.Fatalf("expected the fallback to skip the role-impermissible pool head (fable) and pick sonnet, got %q", last.Model)
	}
}

// TestOrgSpawn_ModelFlagOmitted_NoPermittedModel_NonZeroExit covers the
// error path: a role restricted to a model that has no matching --driver
// entry at all must fail fast with the driver+role error, not silently pick
// an impermissible model.
func TestOrgSpawn_ModelFlagOmitted_NoPermittedModel_NonZeroExit(t *testing.T) {
	t.Setenv("PATH", "")
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	configPath := filepath.Join(dir, "ralph.toml")
	content := "[org]\n" +
		"max_seats = 5\n" +
		"driver_pool = [\"claude\", \"codex\"]\n\n" +
		"[[org.model_pool]]\n" +
		"driver = \"claude\"\n" +
		"model = \"fable\"\n\n" +
		"[[org.model_pool]]\n" +
		"driver = \"codex\"\n" +
		"model = \"gpt-6-astra\"\n\n" +
		"[org.roles]\n" +
		"reviewer = [\"gpt-6-astra\"]\n"
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "reviewer",
		"--driver", "claude", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--config", configPath, "--dry-run",
	)
	if err == nil {
		t.Fatalf("expected non-zero exit when no [org].model_pool entry for --driver is permitted for --role, output: %s", out)
	}
	if !strings.Contains(err.Error(), `driver "claude"`) || !strings.Contains(err.Error(), `role "reviewer"`) {
		t.Errorf("expected error to name both the driver and role, got: %v", err)
	}
	if strings.Contains(out, "--model omitted") {
		t.Errorf("expected no fallback warning to be printed when the fallback itself fails, got: %s", out)
	}
}

func TestOrgStatus_EmptyPATH_FullRosterFromManifestWithCorruptCount(t *testing.T) {
	t.Setenv("PATH", "")
	stateDir := t.TempDir()
	manifestPath := org.ManifestPathIn(stateDir)

	store := org.NewManifestStoreAtPath(manifestPath)
	if err := store.Append(org.ManifestEvent{TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: "spawn_started", Driver: "claude", Model: "sonnet"}); err != nil {
		t.Fatalf("seed spawn_started: %v", err)
	}
	if err := store.Append(org.ManifestEvent{TS: "2026-08-01T00:00:01Z", OrgID: "org-a", SeatID: "seat-1", Event: "spawned", Driver: "claude", Model: "sonnet", PaneID: "pane-1"}); err != nil {
		t.Fatalf("seed spawned: %v", err)
	}

	// Inject one corrupt line directly.
	f, err := os.OpenFile(manifestPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open manifest for corrupt injection: %v", err)
	}
	if _, err := f.WriteString("{not valid json\n"); err != nil {
		t.Fatalf("write corrupt line: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close manifest: %v", err)
	}

	out, err := runOrgCmd(t, "status", "--org-id", "org-a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status failed with no herdr/agmsg/processes: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "seat-1") {
		t.Errorf("expected seat-1 in roster output, got: %s", out)
	}
	if !strings.Contains(out, "spawned") {
		t.Errorf("expected seat-1's state (spawned) in output, got: %s", out)
	}
	if !strings.Contains(out, "1 corrupt manifest line") {
		t.Errorf("expected corrupt-line count warning in output, got: %s", out)
	}
}

func TestOrgStatus_JSON(t *testing.T) {
	t.Setenv("PATH", "")
	stateDir := t.TempDir()
	manifestPath := org.ManifestPathIn(stateDir)
	store := org.NewManifestStoreAtPath(manifestPath)
	if err := store.Append(org.ManifestEvent{TS: "2026-08-01T00:00:00Z", OrgID: "org-a", SeatID: "seat-1", Event: "spawned", Driver: "claude", Model: "sonnet", PaneID: "pane-1"}); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}

	out, err := runOrgCmd(t, "status", "--org-id", "org-a", "--state-dir", stateDir, "--json")
	if err != nil {
		t.Fatalf("status --json failed: %v", err)
	}
	if !strings.Contains(out, "\"seat_id\"") || !strings.Contains(out, "seat-1") {
		t.Errorf("expected JSON output containing seat_id/seat-1, got: %s", out)
	}
}

func TestOrgCmd_RequiresOrgID(t *testing.T) {
	out, err := runOrgCmd(t, "status")
	if err == nil {
		t.Fatalf("expected error when --org-id is omitted, output: %s", out)
	}
	if !strings.Contains(err.Error(), "--org-id") {
		t.Errorf("expected error to mention --org-id, got: %v", err)
	}
}

func TestOrgDisband_StopsActiveSeatsAndDisbandsOrg(t *testing.T) {
	herdrLog, agmsgLog := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	out, err := runOrgCmd(t, "disband", "--org-id", "org-a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("disband failed: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "stopped seat \"seat-1\"") || !strings.Contains(out, "disbanded org \"org-a\"") {
		t.Errorf("expected stop+disband confirmation, got: %s", out)
	}

	herdrLines := readLogLines(t, herdrLog)
	if countLinesWithPrefix(herdrLines, "pane send-keys") == 0 {
		t.Error("expected disband to send a stop signal to the active seat")
	}

	// AC-5: disband's per-seat Stop must also best-effort leave.sh the
	// seat -- leave.sh's argv is "TEAM AGENT_ID", so the seat_id (seat-1)
	// shows up as the last logged token.
	agmsgLines := readLogLines(t, agmsgLog)
	if !containsLine(agmsgLines, "ralph-org-a seat-1") {
		t.Errorf("expected a leave.sh invocation with argv 'ralph-org-a seat-1', got agmsg log: %v", agmsgLines)
	}

	statusOut, err := runOrgCmd(t, "status", "--org-id", "org-a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status after disband failed: %v", err)
	}
	if !strings.Contains(statusOut, "stopped") {
		t.Errorf("expected seat-1's state (stopped, via disband) in status output, got: %s", statusOut)
	}
	if strings.Contains(statusOut, "(active)") {
		t.Errorf("expected no seat to remain marked active after disband, got: %s", statusOut)
	}
	// Live-smoke follow-up: the stopped event must carry seat.Role/Driver/
	// Model forward so status after stop doesn't show blank columns.
	if !strings.Contains(statusOut, "worker") || !strings.Contains(statusOut, "claude") || !strings.Contains(statusOut, "sonnet") {
		t.Errorf("expected status after disband to still show seat role/driver/model, got: %s", statusOut)
	}
}

func TestOrgStop_UnknownSeat_NonZeroExit(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t, "stop", "--org-id", "org-a", "--seat", "never-spawned", "--state-dir", stateDir)
	if err == nil {
		t.Fatalf("expected non-zero exit stopping an unknown seat, output: %s", out)
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	if len(events) != 0 {
		t.Fatalf("expected no manifest event appended for an unknown-seat stop, got %v", events)
	}
}

func TestOrgStop_ExistingSeat_LeavesAndRecordsOutcome(t *testing.T) {
	_, agmsgLog := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	out, err := runOrgCmd(t, "stop", "--org-id", "org-a", "--seat", "seat-1", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("stop failed: %v (output: %s)", err, out)
	}

	agmsgLines := readLogLines(t, agmsgLog)
	if !containsLine(agmsgLines, "ralph-org-a seat-1") {
		t.Errorf("expected a leave.sh invocation with argv 'ralph-org-a seat-1' in the agmsg log, got: %v", agmsgLines)
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	last := events[len(events)-1]
	if last.Event != "stopped" {
		t.Fatalf("expected last event stopped, got %q", last.Event)
	}
	if !strings.Contains(last.Details, "leave=ok") {
		t.Errorf("expected Details to record a successful leave, got %q", last.Details)
	}
	if last.Role != "worker" || last.Driver != "claude" || last.Model != "sonnet" {
		t.Errorf("expected stopped event to carry seat role/driver/model, got role=%q driver=%q model=%q", last.Role, last.Driver, last.Model)
	}

	statusOut, err := runOrgCmd(t, "status", "--org-id", "org-a", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("status after stop failed: %v", err)
	}
	if !strings.Contains(statusOut, "worker") || !strings.Contains(statusOut, "claude") || !strings.Contains(statusOut, "sonnet") {
		t.Errorf("expected status after stop to still show seat role/driver/model, got: %s", statusOut)
	}
}

// containsLine reports whether any line in lines equals want exactly.
func containsLine(lines []string, want string) bool {
	return slices.Contains(lines, want)
}

// --- codex model observation CLI warning (plan Scope row 5, AC-6) ---

// codexOrgTestCommandedModel/codexOrgTestReportedModel mirror the plan's
// own motivating incident (#155 evidence, plan background: "--model
// gpt-5.5 を渡した codex 座席の実効モデルが gpt-5.6-sol になった"): a
// --model gpt-5.5 spawn whose codex session record reports gpt-5.6-sol.
const (
	codexOrgTestCommandedModel = "gpt-5.5"
	codexOrgTestReportedModel  = "gpt-5.6-sol"
)

// writeCodexOrgConfig writes a ralph.toml with an [org] section that allows
// the codex driver (one [[org.model_pool]] entry,
// codexOrgTestCommandedModel) and, when guardedRole is non-empty, maps that
// role to guarded permission mode -- codex's fail-closed autonomous default
// (permissionArgsForDriver) would otherwise reject every codex spawn in
// this file outright, the same gate internal/org/spawn_test.go's
// codexGuardedOrg works around for the library-level tests. An empty
// guardedRole deliberately leaves every role on the autonomous default, for
// the one test that needs that fail-closed rejection.
func writeCodexOrgConfig(t *testing.T, dir string, maxSeats int, guardedRole string) string {
	t.Helper()
	path := filepath.Join(dir, "ralph.toml")
	content := "[org]\n" +
		"max_seats = " + itoa(maxSeats) + "\n" +
		"driver_pool = [\"claude\", \"codex\"]\n\n" +
		"[[org.model_pool]]\n" +
		"driver = \"codex\"\n" +
		"model = \"" + codexOrgTestCommandedModel + "\"\n\n" +
		"[org.permissions]\n" +
		"default = \"autonomous\"\n"
	if guardedRole != "" {
		content += "\n[org.permissions.roles]\n" + guardedRole + " = \"guarded\"\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write codex org config: %v", err)
	}
	return path
}

// codexPromptPathFor mirrors internal/org's promptFilePath convention
// (<state-dir>/prompts/<org_id>_<seat_id>.md, spawn.go) so a CLI test can
// pre-write a session-record fixture at the exact path the spawn it is
// about to run will itself write the role-prompt file to. stateDir must
// already be absolute (every test below passes one built from t.TempDir()),
// matching promptFilePath's own absPath(filepath.Dir(...)) resolution.
func codexPromptPathFor(stateDir, orgID, seatID string) string {
	return filepath.Join(stateDir, "prompts", orgID+"_"+seatID+".md")
}

// cliCodexFixtureLine is the common envelope every fixture line below
// marshals through -- see writeCliCodexFixture.
type cliCodexFixtureLine struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Payload   any    `json:"payload"`
}

// writeCliCodexFixture writes a minimal qualifying rollout-*.jsonl record
// (session_meta + turn_context + a user message containing promptPath)
// under sessionsDir for at, shaped like the real codex-cli records
// internal/org/codex_session.go's ObserveCodexEffectiveModel reads. This is
// a CLI-package-local rebuild of internal/org/codex_session_test.go's line
// builders (unexported there, so not reachable across the package
// boundary) -- the two stay in sync by both matching the same real-record
// shape the plan documents, not by sharing code.
func writeCliCodexFixture(t *testing.T, sessionsDir, promptPath string, at time.Time, model string) {
	t.Helper()
	ts := func(tm time.Time) string { return tm.UTC().Format("2006-01-02T15:04:05.000Z") }
	marshal := func(typ string, payload any) string {
		b, err := json.Marshal(cliCodexFixtureLine{Timestamp: ts(at), Type: typ, Payload: payload})
		if err != nil {
			t.Fatalf("marshal codex fixture line: %v", err)
		}
		return string(b)
	}
	lines := []string{
		marshal("session_meta", map[string]string{"timestamp": ts(at)}),
		marshal("turn_context", map[string]string{"model": model, "effort": "medium"}),
		marshal("response_item", map[string]any{
			"type": "message",
			"role": "user",
			"content": []map[string]string{
				// The observer matches on the full pointer sentence ralph
				// itself passes the seat as its initial prompt argument
				// when the role-prompt was written to a file, not merely
				// the bare path -- calling the exported
				// org.PromptFilePointer here (rather than duplicating its
				// Japanese literal) keeps this fixture from ever silently
				// drifting out of sync with it.
				{"type": "input_text", "text": org.PromptFilePointer(promptPath)},
			},
		}),
	}
	dateDir := filepath.Join(sessionsDir, at.Local().Format("2006/01/02"))
	if err := os.MkdirAll(dateDir, 0o755); err != nil {
		t.Fatalf("mkdir codex session date dir: %v", err)
	}
	fixturePath := filepath.Join(dateDir, "rollout-cli-test.jsonl")
	if err := os.WriteFile(fixturePath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write codex fixture: %v", err)
	}
}

// setCodexSessionsDirOverride points org.go's orgCodexSessionsDirOverride
// seam (see main_test.go's TestMain) at dir for the duration of the
// calling test, restoring TestMain's own pinned value afterward -- the
// same save/restore/cleanup pattern doctor_shell_alias_test.go and
// doctor_codex_writable_root_test.go already use for their own seams.
func setCodexSessionsDirOverride(t *testing.T, dir string) {
	t.Helper()
	orig := orgCodexSessionsDirOverride
	orgCodexSessionsDirOverride = dir
	t.Cleanup(func() { orgCodexSessionsDirOverride = orig })
}

// TestOrgSpawn_CLI_CodexModelMismatch_PrintsWarning is AC-6's spawn-time
// path: a codex seat's session record (already on disk before the command
// runs, so Spawn's very first, no-wait poll finds it) reports a different
// model than --model commanded, so `ralph org spawn` prints the mismatch
// warning to stderr (merged into runOrgCmd's combined buffer) with exit 0.
func TestOrgSpawn_CLI_CodexModelMismatch_PrintsWarning(t *testing.T) {
	setupOrgStubPATH(t)
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	configPath := writeCodexOrgConfig(t, dir, 3, "implementer")
	sessionsDir := filepath.Join(dir, "codex-sessions")
	setCodexSessionsDirOverride(t, sessionsDir)

	promptPath := codexPromptPathFor(stateDir, "org-a", "seat-1")
	writeCliCodexFixture(t, sessionsDir, promptPath, time.Now().Add(time.Second), codexOrgTestReportedModel)

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "implementer",
		"--driver", "codex", "--model", codexOrgTestCommandedModel, "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--config", configPath,
	)
	if err != nil {
		t.Fatalf("spawn failed: %v (output: %s)", err, out)
	}
	wantWarning := `warning: seat "seat-1" was started with --model ` + codexOrgTestCommandedModel +
		", but codex reports it is running " + codexOrgTestReportedModel + "."
	if !strings.Contains(out, wantWarning) {
		t.Errorf("expected the model-mismatch warning in output, got: %s", out)
	}
}

// TestOrgSpawn_CLI_CodexModelMatch_NoWarning is the negative case: the
// session record reports the commanded model, so no warning is printed.
func TestOrgSpawn_CLI_CodexModelMatch_NoWarning(t *testing.T) {
	setupOrgStubPATH(t)
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	configPath := writeCodexOrgConfig(t, dir, 3, "implementer")
	sessionsDir := filepath.Join(dir, "codex-sessions")
	setCodexSessionsDirOverride(t, sessionsDir)

	promptPath := codexPromptPathFor(stateDir, "org-a", "seat-1")
	writeCliCodexFixture(t, sessionsDir, promptPath, time.Now().Add(time.Second), codexOrgTestCommandedModel)

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "implementer",
		"--driver", "codex", "--model", codexOrgTestCommandedModel, "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--config", configPath,
	)
	if err != nil {
		t.Fatalf("spawn failed: %v (output: %s)", err, out)
	}
	if strings.Contains(out, "warning:") {
		t.Errorf("expected no warning when the observed model matches, got: %s", out)
	}
}

// TestOrgSpawn_CLI_EnvelopeRejection_NoModelWarning is AC-6's negative
// guard against the envelope-rejection false positive: a codex seat
// rejected fail-closed by permissionArgsForDriver (autonomous default, no
// guarded role) gets an honored=false receipt with NO reported model
// (reject(), spawn.go) -- that must never be printed as a model-mismatch
// warning.
func TestOrgSpawn_CLI_EnvelopeRejection_NoModelWarning(t *testing.T) {
	setupOrgStubPATH(t)
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	configPath := writeCodexOrgConfig(t, dir, 3, "") // no guarded role at all

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "implementer",
		"--driver", "codex", "--model", codexOrgTestCommandedModel, "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--config", configPath,
	)
	if err == nil {
		t.Fatalf("expected the fail-closed autonomous-codex rejection to exit non-zero, output: %s", out)
	}
	if strings.Contains(out, "warning: seat") {
		t.Errorf("expected no model-mismatch warning for an envelope rejection, got: %s", out)
	}
}

// TestOrgStop_CLI_CodexModelMismatch_PrintsWarning is AC-6's stop-time
// path: spawn finds nothing (no fixture yet, so its own receipt is
// unknown, printing no warning), a mismatching record appears before
// `ralph org stop` runs, and stop's own single observation prints the
// warning.
func TestOrgStop_CLI_CodexModelMismatch_PrintsWarning(t *testing.T) {
	setupOrgStubPATH(t)
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	configPath := writeCodexOrgConfig(t, dir, 3, "implementer")
	sessionsDir := filepath.Join(dir, "codex-sessions")
	setCodexSessionsDirOverride(t, sessionsDir)

	spawnOut, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "implementer",
		"--driver", "codex", "--model", codexOrgTestCommandedModel, "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--config", configPath,
	)
	if err != nil {
		t.Fatalf("spawn failed: %v (output: %s)", err, spawnOut)
	}
	if strings.Contains(spawnOut, "warning:") {
		t.Errorf("expected no warning at spawn time (no fixture yet), got: %s", spawnOut)
	}

	promptPath := codexPromptPathFor(stateDir, "org-a", "seat-1")
	writeCliCodexFixture(t, sessionsDir, promptPath, time.Now().Add(time.Second), codexOrgTestReportedModel)

	stopOut, err := runOrgCmd(t, "stop", "--org-id", "org-a", "--seat", "seat-1", "--state-dir", stateDir, "--config", configPath)
	if err != nil {
		t.Fatalf("stop failed: %v (output: %s)", err, stopOut)
	}
	wantWarning := `warning: seat "seat-1" was started with --model ` + codexOrgTestCommandedModel +
		", but codex reports it is running " + codexOrgTestReportedModel + "."
	if !strings.Contains(stopOut, wantWarning) {
		t.Errorf("expected the model-mismatch warning in stop's output, got: %s", stopOut)
	}
}

// TestOrgStop_CLI_CodexModelMatch_NoWarning is the negative case for stop:
// the record found at stop time reports the commanded model.
func TestOrgStop_CLI_CodexModelMatch_NoWarning(t *testing.T) {
	setupOrgStubPATH(t)
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	configPath := writeCodexOrgConfig(t, dir, 3, "implementer")
	sessionsDir := filepath.Join(dir, "codex-sessions")
	setCodexSessionsDirOverride(t, sessionsDir)

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "implementer",
		"--driver", "codex", "--model", codexOrgTestCommandedModel, "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir, "--config", configPath,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	promptPath := codexPromptPathFor(stateDir, "org-a", "seat-1")
	writeCliCodexFixture(t, sessionsDir, promptPath, time.Now().Add(time.Second), codexOrgTestCommandedModel)

	stopOut, err := runOrgCmd(t, "stop", "--org-id", "org-a", "--seat", "seat-1", "--state-dir", stateDir, "--config", configPath)
	if err != nil {
		t.Fatalf("stop failed: %v (output: %s)", err, stopOut)
	}
	if strings.Contains(stopOut, "warning:") {
		t.Errorf("expected no warning when the observed model matches, got: %s", stopOut)
	}
}

// TestOrgSpawn_RoleAndScopeFlags_ExpandTemplateAndRecordScope covers AC-4
// and the scope half of AC-7/design: `--role reviewer --scope ...` expands
// the embedded reviewer template (with org_id/seat_id/scope substituted).
// Real herdr rejects multi-line agent args (see spawn.go's
// needsPromptFile), so the rendered template is written to a prompt file
// under state-dir and the AgentStart argv the herdr stub receives carries
// only a one-line pointer to it; this test asserts the pointer via the
// herdr log and the full rendered content via the prompt file. It also
// checks the spawned event's Details records "scope=<value>".
func TestOrgSpawn_RoleAndScopeFlags_ExpandTemplateAndRecordScope(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "reviewer-1", "--role", "reviewer",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "internal/org/**",
		"--state-dir", stateDir,
	)
	if err != nil {
		t.Fatalf("spawn failed: %v (output: %s)", err, out)
	}

	data, rerr := os.ReadFile(herdrLog)
	if rerr != nil {
		t.Fatalf("read herdr log: %v", rerr)
	}
	logText := string(data)
	if !strings.Contains(logText, "役割指示を読み込んで従ってください: ") {
		t.Fatalf("expected AgentStart argv logged to contain the prompt-file pointer, got:\n%s", logText)
	}
	if strings.Contains(logText, ".claude/rules/ralph/agent-messaging.md") {
		t.Errorf("expected the rendered template body NOT to appear inline in argv (it must go through the prompt file), got:\n%s", logText)
	}

	promptPath := filepath.Join(stateDir, "prompts", "org-a_reviewer-1.md")
	promptData, perr := os.ReadFile(promptPath)
	if perr != nil {
		t.Fatalf("expected prompt file at %q: %v", promptPath, perr)
	}
	promptText := string(promptData)
	for _, want := range []string{"org-a", "reviewer-1", "internal/org/**", ".claude/rules/ralph/agent-messaging.md"} {
		if !strings.Contains(promptText, want) {
			t.Errorf("expected prompt file content to contain %q, got:\n%s", want, promptText)
		}
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	last := events[len(events)-1]
	if last.Event != "spawned" {
		t.Fatalf("expected last event spawned, got %q", last.Event)
	}
	if !strings.Contains(last.Details, "scope=internal/org/**") {
		t.Fatalf("expected spawned event Details to record scope, got %q", last.Details)
	}
}

// TestOrgSpawn_UnknownRole_NoTemplateApplied is the CLI-level counterpart of
// the org-package unit test: an unknown --role must not fail spawn, and the
// herdr log must not contain the reviewer/qa template markers.
func TestOrgSpawn_UnknownRole_NoTemplateApplied(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "not-a-known-role",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--prompt", "verbatim prompt",
		"--state-dir", stateDir,
	)
	if err != nil {
		t.Fatalf("spawn with an unknown role should not error, got %v (output: %s)", err, out)
	}

	data, rerr := os.ReadFile(herdrLog)
	if rerr != nil {
		t.Fatalf("read herdr log: %v", rerr)
	}
	logText := string(data)
	if !strings.Contains(logText, "verbatim prompt") {
		t.Errorf("expected the verbatim --prompt in the AgentStart argv, got:\n%s", logText)
	}
	if strings.Contains(logText, ".claude/rules/ralph/agent-messaging.md") {
		t.Errorf("expected no role-template markers for an unknown role, got:\n%s", logText)
	}
}

// TestOrgSend_MalformedMessage_NonZeroExitAndNoManifestEvent is the AC-11
// CLI rejection test: `ralph org send` with a malformed --text exits
// non-zero and appends no manifest event.
func TestOrgSend_MalformedMessage_NonZeroExitAndNoManifestEvent(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}
	eventsBefore := readManifestEvents(t, org.ManifestPathIn(stateDir))

	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1", "--text", "not a valid protocol message", "--state-dir", stateDir)
	if err == nil {
		t.Fatalf("expected non-zero exit for a malformed --text, output: %s", out)
	}

	eventsAfter := readManifestEvents(t, org.ManifestPathIn(stateDir))
	if len(eventsAfter) != len(eventsBefore) {
		t.Fatalf("expected no new manifest event for a rejected send, %d -> %d", len(eventsBefore), len(eventsAfter))
	}

	// Sanity: no pane-send-text call should have reached the herdr stub for
	// the rejected message.
	herdrLines := readLogLines(t, herdrLog)
	if countLinesWithPrefix(herdrLines, "pane send-text") != 0 {
		t.Errorf("expected no 'pane send-text' call for a rejected send, got: %v", herdrLines)
	}
}

// TestOrgSend_RawFlag_BypassesValidation covers the --raw escape hatch at
// the CLI layer: the same malformed --text that TestOrgSend_MalformedMessage
// rejects must succeed with --raw, and the recorded `sent` event's Details
// must note raw=true.
func TestOrgSend_RawFlag_BypassesValidation(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	// --enter-delay-ms 1 keeps this test fast: the real default (750ms)
	// would otherwise add real wall-clock time to every CLI-level send
	// test that succeeds.
	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1", "--text", "not a valid protocol message",
		"--raw", "--enter-delay-ms", "1", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("expected --raw to bypass protocol validation, got %v (output: %s)", err, out)
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	last := events[len(events)-1]
	if last.Event != "sent" {
		t.Fatalf("expected last event sent, got %q", last.Event)
	}
	if !strings.Contains(last.Details, "raw=true") {
		t.Errorf("expected the sent event's Details to record raw=true, got %q", last.Details)
	}
}

// TestOrgSend_ValidTypedMessage_Succeeds is the positive counterpart: a
// well-formed typed message is accepted without --raw, and -- since the
// herdr stub answers every "agent wait" with exit 0 -- the submit is
// confirmed, so no "could not confirm" warning appears on stderr (the
// confirmed half of AC-3's "no warning" pairing; see
// TestOrgSend_UnconfirmedSubmit_WarnsOnStderr_ExitZero for the unconfirmed
// half).
func TestOrgSend_ValidTypedMessage_Succeeds(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	// --enter-delay-ms 1: see TestOrgSend_RawFlag_BypassesValidation's
	// comment above.
	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1",
		"--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing", "--enter-delay-ms", "1", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("expected a well-formed typed message to be accepted, got %v (output: %s)", err, out)
	}
	if strings.Contains(out, "could not confirm") {
		t.Errorf("expected no unconfirmed-submit warning when the herdr stub confirms the submit, got: %s", out)
	}
	if !strings.Contains(out, "sent message to seat \"seat-1\"") {
		t.Errorf("expected sent confirmation in output, got: %s", out)
	}
}

// TestOrgSend_NegativeEnterDelayMS_NonZeroExit covers --enter-delay-ms's
// input validation: a negative value is rejected before newOrgRuntime even
// builds a runtime (no herdr/agmsg call of any kind).
func TestOrgSend_NegativeEnterDelayMS_NonZeroExit(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1",
		"--text", "TYPE: HEARTBEAT", "--enter-delay-ms", "-1", "--state-dir", stateDir)
	if err == nil {
		t.Fatalf("expected non-zero exit for a negative --enter-delay-ms, output: %s", out)
	}
	if !strings.Contains(err.Error(), "--enter-delay-ms") {
		t.Errorf("expected the error to mention --enter-delay-ms, got: %v", err)
	}
	if herdrLines := readLogLines(t, herdrLog); len(herdrLines) != 0 {
		t.Errorf("expected zero herdr calls for a rejected --enter-delay-ms, got: %v", herdrLines)
	}
}

// TestOrgSend_UnconfirmedSubmit_WarnsOnStderr_ExitZero is the AC-3
// unconfirmed-submit path: with the herdr stub set to fail only the
// post-Enter confirm AgentWait call (ORG_STUB_AGENT_WAIT_CONFIRM_FAIL, see
// herdrStub's doc comment), `ralph org send` still exits 0, but prints a
// warning to stderr naming the seat, the pane id, the no-resend rationale,
// and (self-review cycle-2 C2-2,
// docs/reports/self-review-2026-09-19-org-send-enter-timing.md) the same
// "do not press Enter if the pane shows anything else" guard the error-path
// notes already carry.
func TestOrgSend_UnconfirmedSubmit_WarnsOnStderr_ExitZero(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}
	t.Setenv("ORG_STUB_AGENT_WAIT_CONFIRM_FAIL", "1")

	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1",
		"--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing", "--enter-delay-ms", "1", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("expected exit 0 even when the submit cannot be confirmed, got %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "could not confirm") {
		t.Errorf("expected the unconfirmed-submit warning, got: %s", out)
	}
	if !strings.Contains(out, "\"seat-1\"") {
		t.Errorf("expected the warning to name the target seat, got: %s", out)
	}
	if !strings.Contains(out, "pane-stub-1") {
		t.Errorf("expected the warning to include the seat's pane id, got: %s", out)
	}
	if !strings.Contains(out, "does not resend Enter") {
		t.Errorf("expected the warning to explain ralph does not resend Enter, got: %s", out)
	}
	if !strings.Contains(out, "do not press Enter") {
		t.Errorf("expected the warning to explicitly say not to press Enter when the pane shows anything else, got: %s", out)
	}
}

// TestOrgSend_DryRun_NeverWarnsAboutUnconfirmedSubmit covers the DryRun
// half of edge case 7 at the CLI layer: DryRun never runs the confirm wait
// at all (see Send's doc comment), so it must never print the
// unconfirmed-submit warning even though SubmitConfirmed is always false
// for DryRun.
func TestOrgSend_DryRun_NeverWarnsAboutUnconfirmedSubmit(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	t.Setenv("ORG_STUB_AGENT_WAIT_CONFIRM_FAIL", "1")

	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1",
		"--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing", "--dry-run", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("expected --dry-run to succeed, got %v (output: %s)", err, out)
	}
	if strings.Contains(out, "could not confirm") {
		t.Errorf("expected no unconfirmed-submit warning for --dry-run, got: %s", out)
	}
}

// TestOrgSend_PaneSendKeysFails_NotesEnterUnacknowledgedOnStderr is the AR-1
// cross-review fix (docs/reports/cross-review-triage-org-send-enter-timing.md):
// when PaneSendKeys itself fails, the CLI must no longer assert "not
// submitted" -- herdr's CLI can be killed by ctx deadline expiry mid-call,
// so the message may or may not have been delivered. The command still
// exits non-zero and still prints a stderr note pointing the operator at
// the pane, but the note must say the outcome is unknown and must not
// issue an unconditional "press Enter" or "submit it" instruction.
func TestOrgSend_PaneSendKeysFails_NotesEnterUnacknowledgedOnStderr(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}
	// The generic ORG_STUB_FAIL="pane:send-keys" marker fails every "herdr
	// pane send-keys" call -- fine here since this test's only send-keys
	// call is the one it means to fail (Send's Enter; no Stop is issued).
	t.Setenv("ORG_STUB_FAIL", "pane:send-keys")

	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1",
		"--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing", "--enter-delay-ms", "1", "--state-dir", stateDir)
	if err == nil {
		t.Fatalf("expected non-zero exit when PaneSendKeys fails, output: %s", out)
	}
	if !strings.Contains(out, "may or may not have been submitted") {
		t.Errorf("expected the enter-unacknowledged note, got: %s", out)
	}
	if !strings.Contains(out, "Only if") {
		t.Errorf("expected the note to condition the submit/clear instruction on what the operator actually sees, got: %s", out)
	}
	if !strings.Contains(out, "do not press Enter") {
		t.Errorf("expected the note to explicitly say not to press Enter unconditionally, got: %s", out)
	}
	if !strings.Contains(out, "pane-stub-1") {
		t.Errorf("expected the note to name the seat's pane id, got: %s", out)
	}
	if !strings.Contains(out, "\"seat-1\"") {
		t.Errorf("expected the note to name the target seat, got: %s", out)
	}
	if strings.Contains(out, "not submitted") {
		t.Errorf("expected no unconditional not-submitted wording on this path (the outcome is unknown), got: %s", out)
	}
	if strings.Contains(out, "the message text was typed into pane") {
		t.Errorf("expected the enter-unacknowledged note, not the text-typed note, got: %s", out)
	}
	if strings.Contains(out, "Enter was already pressed") {
		t.Errorf("expected no submitted-but-unrecorded note on this path (Enter's delivery is unknown, not confirmed), got: %s", out)
	}
}

// TestOrgSend_PaneSendTextFails_NotesTextUnacknowledgedOnStderr is the AR-1
// cross-review fix's twin for the step before PaneSendKeys: when herdr
// rejects the send-text call, the note must say the outcome is unknown
// (not "nothing was typed", and not "typed but not submitted") -- ctx
// deadline expiry can kill herdr's CLI mid-call, so the paste may already
// have landed before the error came back.
func TestOrgSend_PaneSendTextFails_NotesTextUnacknowledgedOnStderr(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}
	// Same generic ORG_STUB_FAIL mechanism as pane:send-keys above, one
	// step earlier in the argv ("pane send-text" -> "$1:$2" = "pane:send-text").
	t.Setenv("ORG_STUB_FAIL", "pane:send-text")

	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1",
		"--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing", "--enter-delay-ms", "1", "--state-dir", stateDir)
	if err == nil {
		t.Fatalf("expected non-zero exit when PaneSendText fails, output: %s", out)
	}
	if !strings.Contains(out, "the send-text call") {
		t.Errorf("expected the text-unacknowledged note, got: %s", out)
	}
	if !strings.Contains(out, "may or may not have been typed") {
		t.Errorf("expected the note to say the outcome is unknown, got: %s", out)
	}
	if !strings.Contains(out, "pane-stub-1") {
		t.Errorf("expected the note to include the seat's pane id, got: %s", out)
	}
	if !strings.Contains(out, "\"seat-1\"") {
		t.Errorf("expected the note to name the target seat, got: %s", out)
	}
	if strings.Contains(out, "send-keys") {
		t.Errorf("expected no send-keys instruction on this path, got: %s", out)
	}
	if strings.Contains(out, "Enter was already pressed") {
		t.Errorf("expected no submitted-but-unrecorded note on this path, got: %s", out)
	}
}

// TestOrgSend_CtxExpiresDuringEnterDelay_NotesTextTypedOnStderr is the CLI
// counterpart of internal/org/verbs_test.go's
// TestOrgSend_CtxExpiresDuringEnterDelay_AfterBudgetCheckPasses: ctx
// expiring inside the pre-Enter pause, after PaneSendText has already
// succeeded, must still print the ORIGINAL "typed but not submitted" note
// -- unlike the two "unacknowledged" cases above, this state IS known for
// certain (PaneSendText really did return success before ctx expired).
// ORG_STUB_PANE_SEND_TEXT_SLEEP (see herdrStub's doc comment) simulates a
// slow herdr round trip for "pane send-text" so the real subprocess-backed
// budget check passes but the pre-Enter wait's ctx then expires -- the
// real-subprocess analogue of fakeHerdr.paneSendTextDelay.
//
// --timeout-ms 2500 / --enter-delay-ms 1800 / sleep 1.1s (self-review
// cycle-2 C2-1, docs/reports/self-review-2026-09-19-org-send-enter-timing.md
// -- the previous 1500/400 pairing made the ctx-vs-timer race's margin
// equal the two herdr-stub subprocess calls' own overhead, so a FAST,
// unloaded machine had the thinnest margin, and any overhead over 400ms
// killed the send-text call itself and flipped the result to
// text-unacknowledged). With o1/o2 as each stub subprocess call's own
// overhead: the budget check's margin is about 700ms minus o1 (2500 -
// o1 - 1800); PaneSendText's own survival margin (it must return before
// ctx expires, or this test would hit SendProgressTextUnacknowledged
// instead) is about 1400ms minus (o1+o2) (2500 - o1 - 1100 - o2); and once
// PaneSendText does return, ctx has at most ~1400ms left against a 1800ms
// timer, so it wins the pre-Enter wait by 400ms PLUS (o1+o2) -- floored at
// 400ms regardless of overhead, and growing with it, the opposite
// direction from before.
func TestOrgSend_CtxExpiresDuringEnterDelay_NotesTextTypedOnStderr(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}
	t.Setenv("ORG_STUB_PANE_SEND_TEXT_SLEEP", "1.1")

	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1",
		"--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing",
		"--timeout-ms", "2500", "--enter-delay-ms", "1800", "--state-dir", stateDir)
	if err == nil {
		t.Fatalf("expected non-zero exit when ctx expires inside the pre-Enter pause, output: %s", out)
	}
	if !strings.Contains(out, "the message text was typed into pane") {
		t.Errorf("expected the typed-but-not-submitted note, got: %s", out)
	}
	if !strings.Contains(out, "pane-stub-1") {
		t.Errorf("expected the note to include the seat's pane id, got: %s", out)
	}
	if !strings.Contains(out, "\"seat-1\"") {
		t.Errorf("expected the note to name the target seat, got: %s", out)
	}
	if !strings.Contains(out, "before sending again") {
		t.Errorf("expected the note to warn against retrying blindly, got: %s", out)
	}
	if strings.Contains(out, "may or may not") {
		t.Errorf("expected no uncertainty wording on this path (the state IS known), got: %s", out)
	}
}

// TestOrgSend_BudgetTooSmallForEnterDelay_NoTypedNoteOnStderr is the CLI
// counterpart of TestOrgSend_BudgetTooSmallForEnterDelay_NothingTyped
// (internal/org/verbs_test.go): a --timeout-ms budget too small to fund
// --enter-delay-ms exits non-zero with the "nothing was typed" message,
// and -- unlike the PaneSendKeys-failure case above -- must NOT print the
// typed-but-not-submitted note, since nothing was ever typed.
//
// --timeout-ms 2000 / --enter-delay-ms 60000 are deliberately generous, not
// tight: the idle/done wait still only needs a single fast herdr-stub
// subprocess round trip (comfortably under 2s), so the fail-closed check
// fires almost immediately regardless -- these values just guarantee it
// fires deterministically (remaining ctx time will always be far below
// 60000ms) without risking ctx expiring inside the idle/done wait itself,
// which a tight budget (as internal/org/verbs_test.go can safely use
// against an instantaneous fake) would risk against a real subprocess.
func TestOrgSend_BudgetTooSmallForEnterDelay_NoTypedNoteOnStderr(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1",
		"--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing",
		"--timeout-ms", "2000", "--enter-delay-ms", "60000", "--state-dir", stateDir)
	if err == nil {
		t.Fatalf("expected non-zero exit when the budget cannot fund the pre-Enter pause, output: %s", out)
	}
	if !strings.Contains(err.Error(), "nothing was typed") {
		t.Errorf("expected the error to say nothing was typed, got: %v", err)
	}
	if strings.Contains(out, "the message text was typed into pane") {
		t.Errorf("expected no typed-but-not-submitted note when nothing was typed, got: %s", out)
	}
}

// TestOrgSend_AppendEventFailsAfterEnter_NotesEnterPressedNotRecorded is the
// CLI counterpart of internal/org/verbs_test.go's
// TestOrgSend_AppendEventFailsAfterEnter_ReportsEnterPressedButUnrecorded,
// renamed in cycle-2 (C2-3, docs/reports/self-review-2026-09-19-org-send-enter-timing.md)
// to say what Send actually observed: "Enter was pressed", not
// "submitted". When the manifest write for the `sent` event fails after
// Enter has already succeeded, the CLI must print the "Enter was already
// pressed ... do not send the message again" note -- and must NOT print
// the typed-but-unsubmitted note or mention "send-keys", either of which
// would wrongly suggest pressing Enter on a seat that has very likely
// already submitted and may have reached an approval dialog.
//
// Unix-only, no build tag needed: internal/cli's own test suite already is
// one (see TestCheckCodexAgmsgWritableRoot_UnreadableConfig_InfoWithReasonOnly's
// doc comment in doctor_codex_writable_root_unix_test.go) since
// internal/org/lockfile.go uses syscall.Flock unconditionally. The
// manifest file is chmod'd read-only AFTER spawn's own writes have already
// landed, so only the send verb's own append fails.
func TestOrgSend_AppendEventFailsAfterEnter_NotesEnterPressedNotRecorded(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores a read-only file's permission bit")
	}
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	manifestPath := org.ManifestPathIn(stateDir)
	if err := os.Chmod(manifestPath, 0o444); err != nil {
		t.Fatalf("chmod manifest read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(manifestPath, 0o644) })

	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1",
		"--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing", "--enter-delay-ms", "1", "--state-dir", stateDir)
	if err == nil {
		t.Fatalf("expected non-zero exit when the sent event cannot be appended, output: %s", out)
	}
	if !strings.Contains(out, "Enter was already pressed") {
		t.Errorf("expected the submitted-but-unrecorded note, got: %s", out)
	}
	if !strings.Contains(out, "pane-stub-1") {
		t.Errorf("expected the note to include the seat's pane id, got: %s", out)
	}
	if !strings.Contains(out, "Do not send the message again") {
		t.Errorf("expected the note to warn against resending, got: %s", out)
	}
	if strings.Contains(out, "not submitted") {
		t.Errorf("expected no typed-but-not-submitted wording on this path, got: %s", out)
	}
	if strings.Contains(out, "send-keys") {
		t.Errorf("expected no send-keys instruction on this path (the message very likely already submitted), got: %s", out)
	}
}

// TestOrgSend_StateDirHint_IncludesFlagWhenExplicit is the AR-2 cross-review
// fix (docs/reports/cross-review-triage-org-send-enter-timing.md): when
// --state-dir was explicitly passed, every printed `ralph org read`
// recovery command -- the unconfirmed-submit warning and every post-error
// note -- must include --state-dir with the resolved absolute path.
// Without it, a human following the printed command resolves the DEFAULT
// state dir instead and either gets "seat not found" or, worse, reads a
// different seat that happens to share the same org_id/seat_id there.
func TestOrgSend_StateDirHint_IncludesFlagWhenExplicit(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	absStateDir, err := filepath.Abs(stateDir)
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	// The unconfirmed-submit warning (a success-path message, not a
	// post-error note).
	t.Setenv("ORG_STUB_AGENT_WAIT_CONFIRM_FAIL", "1")
	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1",
		"--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing", "--enter-delay-ms", "1", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("expected exit 0 for an unconfirmed submit, got %v (output: %s)", err, out)
	}
	if !strings.Contains(out, " --state-dir "+absStateDir) {
		t.Errorf("expected the unconfirmed-submit warning to include --state-dir %s, got: %s", absStateDir, out)
	}
	t.Setenv("ORG_STUB_AGENT_WAIT_CONFIRM_FAIL", "")

	// A post-error note (the text-unacknowledged path).
	t.Setenv("ORG_STUB_FAIL", "pane:send-text")
	out2, err2 := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1",
		"--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing", "--enter-delay-ms", "1", "--state-dir", stateDir)
	if err2 == nil {
		t.Fatalf("expected non-zero exit when PaneSendText fails, output: %s", out2)
	}
	if !strings.Contains(out2, " --state-dir "+absStateDir) {
		t.Errorf("expected the text-unacknowledged note to include --state-dir %s, got: %s", absStateDir, out2)
	}
}

// TestOrgSend_StateDirHint_OmitsFlagWhenResolvedFromEnv is the negative
// counterpart of TestOrgSend_StateDirHint_IncludesFlagWhenExplicit: when
// the state dir comes from RALPH_ORG_STATE_DIR rather than an explicit
// --state-dir flag on this invocation, the hint must NOT repeat
// --state-dir -- the same shell resolves the same env var the same way
// when the operator runs the printed command themselves, so repeating it
// would be redundant (and could go stale if the env var changes between
// the two commands).
func TestOrgSend_StateDirHint_OmitsFlagWhenResolvedFromEnv(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := t.TempDir()
	t.Setenv("RALPH_ORG_STATE_DIR", stateDir)

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	t.Setenv("ORG_STUB_FAIL", "pane:send-text")
	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1",
		"--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing", "--enter-delay-ms", "1")
	if err == nil {
		t.Fatalf("expected non-zero exit when PaneSendText fails, output: %s", out)
	}
	if strings.Contains(out, "--state-dir") {
		t.Errorf("expected no --state-dir in the note when the state dir came from RALPH_ORG_STATE_DIR, got: %s", out)
	}
}

// TestOrgReadCommandHint_Table covers orgReadCommandHint and its
// shellQuoteIfNeeded helper directly: whether --state-dir is appended at
// all, and how a resolved state dir containing shell metacharacters (a
// space, a single quote) is quoted so the printed command stays
// copy-paste-safe.
func TestOrgReadCommandHint_Table(t *testing.T) {
	tests := []struct {
		name             string
		orgID, seat      string
		resolvedStateDir string
		stateDirFlagSet  bool
		want             string
	}{
		{
			name:  "flag not set: no --state-dir regardless of resolvedStateDir",
			orgID: "org-a", seat: "seat-1", resolvedStateDir: "/tmp/state", stateDirFlagSet: false,
			want: "ralph org read --org-id org-a --seat seat-1",
		},
		{
			name:  "flag set, shell-safe path: unquoted",
			orgID: "org-a", seat: "seat-1", resolvedStateDir: "/tmp/state", stateDirFlagSet: true,
			want: "ralph org read --org-id org-a --seat seat-1 --state-dir /tmp/state",
		},
		{
			name:  "flag set, path with a space: single-quoted",
			orgID: "org-a", seat: "seat-1", resolvedStateDir: "/tmp/my state/dir", stateDirFlagSet: true,
			want: "ralph org read --org-id org-a --seat seat-1 --state-dir '/tmp/my state/dir'",
		},
		{
			name:  "flag set, path with an embedded single quote: escaped POSIX-style",
			orgID: "org-a", seat: "seat-1", resolvedStateDir: "/tmp/o'brien", stateDirFlagSet: true,
			want: `ralph org read --org-id org-a --seat seat-1 --state-dir '/tmp/o'\''brien'`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := orgReadCommandHint(tc.orgID, tc.seat, tc.resolvedStateDir, tc.stateDirFlagSet)
			if got != tc.want {
				t.Errorf("orgReadCommandHint(%q, %q, %q, %v) = %q, want %q",
					tc.orgID, tc.seat, tc.resolvedStateDir, tc.stateDirFlagSet, got, tc.want)
			}
		})
	}
}

// TestOrgWait_HappyPath_Succeeds covers `ralph org wait` end to end through
// the real driver.ExecRunner -> exec.Command path: the herdr stub answers
// "agent wait" with "idle", and Wait prints herdr's raw output.
func TestOrgWait_HappyPath_Succeeds(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	out, err := runOrgCmd(t, "wait", "--org-id", "org-a", "--seat", "seat-1", "--until", "idle", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("wait failed: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "idle") {
		t.Errorf("expected herdr's raw output 'idle' in output, got: %s", out)
	}

	herdrLines := readLogLines(t, herdrLog)
	if countLinesWithPrefix(herdrLines, "agent wait") == 0 {
		t.Errorf("expected an 'agent wait' invocation in the herdr log, got: %v", herdrLines)
	}
}

// TestOrgWait_UnknownSeat_StillSucceeds_PassthroughToHerdr documents the
// CLI-level counterpart of TestOrgWait_UnknownSeat_StillDrivesHerdr_NoManifestCheck
// (internal/org/verbs_test.go): `ralph org wait` never checks the manifest
// for the seat's existence, so waiting on a seat id that was never spawned
// still drives the (namespaced) herdr call rather than failing fast the way
// `read`/`stop`/`send` do.
func TestOrgWait_UnknownSeat_StillSucceeds_PassthroughToHerdr(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t, "wait", "--org-id", "org-a", "--seat", "never-spawned", "--until", "idle", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("expected wait on an unrecorded seat to still succeed (pure herdr passthrough), got %v (output: %s)", err, out)
	}

	// --timeout-ms is left at its bounded default (60000) here, so the
	// expected argv includes it -- see TestOrgWait_DefaultUntilAndTimeout_AreBoundedAndDone
	// for the defaults themselves.
	herdrLines := readLogLines(t, herdrLog)
	if !containsLine(herdrLines, "agent wait org-a_never-spawned --until idle --timeout 60000") {
		t.Errorf("expected the herdr log to show a wait call for the namespaced agent name, got: %v", herdrLines)
	}
}

// TestOrgWait_DefaultUntilAndTimeout_AreBoundedAndDone covers the self-review
// HIGH-1 fix: `ralph org wait`'s defaults changed from `--until idle` /
// `--timeout-ms 0` (unbounded) to `--until idle,done` / `--timeout-ms 60000`
// (bounded) -- a headless lead following its own default wait must not be
// able to block forever against a perfectly receptive seat (herdr reports an
// interactive agent resting at its input prompt as "done", not "idle"; see
// internal/org/verbs.go's Send, which already waits on both).
func TestOrgWait_DefaultUntilAndTimeout_AreBoundedAndDone(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t, "wait", "--org-id", "org-a", "--seat", "seat-1", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("wait with default flags failed: %v (output: %s)", err, out)
	}

	herdrLines := readLogLines(t, herdrLog)
	if !containsLine(herdrLines, "agent wait org-a_seat-1 --until idle --until done --timeout 60000") {
		t.Errorf("expected the default --until/--timeout-ms to produce a bounded idle+done wait, got: %v", herdrLines)
	}
}

// TestOrgWait_ExplicitZeroTimeout_StaysUnbounded verifies --timeout-ms 0
// still opts out of the bounded default explicitly (herdr's own `--timeout`
// flag is omitted entirely), matching driver.Herdr.AgentWait's documented
// "timeoutMS <= 0 omits --timeout" contract.
func TestOrgWait_ExplicitZeroTimeout_StaysUnbounded(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t, "wait", "--org-id", "org-a", "--seat", "seat-1", "--timeout-ms", "0", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("wait with --timeout-ms 0 failed: %v (output: %s)", err, out)
	}

	herdrLines := readLogLines(t, herdrLog)
	if !containsLine(herdrLines, "agent wait org-a_seat-1 --until idle --until done") {
		t.Errorf("expected --timeout-ms 0 to omit --timeout entirely, got: %v", herdrLines)
	}
	for _, line := range herdrLines {
		if strings.Contains(line, "--timeout ") {
			t.Errorf("expected no --timeout flag when --timeout-ms is explicitly 0, got: %v", herdrLines)
		}
	}
}

// TestOrgRead_HappyPath_Succeeds covers `ralph org read` end to end: the
// herdr stub answers "pane read" with "pane output".
func TestOrgRead_HappyPath_Succeeds(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	out, err := runOrgCmd(t, "read", "--org-id", "org-a", "--seat", "seat-1", "--lines", "10", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("read failed: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "pane output") {
		t.Errorf("expected herdr's raw pane output in output, got: %s", out)
	}

	herdrLines := readLogLines(t, herdrLog)
	if countLinesWithPrefix(herdrLines, "pane read") == 0 {
		t.Errorf("expected a 'pane read' invocation in the herdr log, got: %v", herdrLines)
	}
}

// TestOrgRead_UnknownSeat_NonZeroExit covers Read's seat-lookup gate at the
// CLI layer: unlike `wait`, `read` resolves the seat's pane_id from the
// manifest first, so an unrecorded seat id must fail before any herdr call
// is attempted.
func TestOrgRead_UnknownSeat_NonZeroExit(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t, "read", "--org-id", "org-a", "--seat", "never-spawned", "--state-dir", stateDir)
	if err == nil {
		t.Fatalf("expected non-zero exit reading an unknown seat, output: %s", out)
	}

	herdrLines := readLogLines(t, herdrLog)
	if countLinesWithPrefix(herdrLines, "pane read") != 0 {
		t.Errorf("expected no 'pane read' call for an unknown seat, got: %v", herdrLines)
	}
}

// TestOrgSpawn_TraversalSeatID_NonZeroExit_NoDriverCalls covers the CLI-layer
// identifier validation added alongside (*org.Org).Spawn's own check
// (self-review MEDIUM-2): a shell that reaches `ralph org spawn` with a
// path-traversal --id must be rejected at flag-parsing time, before
// newOrgRuntime is even constructed, so zero herdr calls happen.
func TestOrgSpawn_TraversalSeatID_NonZeroExit_NoDriverCalls(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "../../../../../../tmp/pwn", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--state-dir", stateDir,
	)
	if err == nil {
		t.Fatalf("expected non-zero exit for a path-traversal --id, output: %s", out)
	}
	if !strings.Contains(err.Error(), "seat_id") {
		t.Errorf("expected error to mention seat_id, got: %v", err)
	}

	herdrLines := readLogLines(t, herdrLog)
	if len(herdrLines) != 0 {
		t.Errorf("expected zero herdr calls for a rejected --id, got: %v", herdrLines)
	}
	if _, statErr := os.Stat(stateDir); !os.IsNotExist(statErr) {
		t.Errorf("expected the state dir to never be created for a rejected --id, stat err=%v", statErr)
	}
}

// TestOrgSpawn_InvalidOrgID_NonZeroExit covers the --org-id half of the same
// CLI-layer gate.
func TestOrgSpawn_InvalidOrgID_NonZeroExit(t *testing.T) {
	out, err := runOrgCmd(t,
		"spawn", "--org-id", "../escape", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
	)
	if err == nil {
		t.Fatalf("expected non-zero exit for a malformed --org-id, output: %s", out)
	}
	if !strings.Contains(err.Error(), "org_id") {
		t.Errorf("expected error to mention org_id, got: %v", err)
	}
}

// TestOrgSend_TraversalTo_NonZeroExit covers --to (send), the other
// user-facing flag that names a target seat id.
func TestOrgSend_TraversalTo_NonZeroExit(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "../escape", "--text", "TYPE: HEARTBEAT", "--state-dir", stateDir)
	if err == nil {
		t.Fatalf("expected non-zero exit for a malformed --to, output: %s", out)
	}
	if !strings.Contains(err.Error(), "seat_id") {
		t.Errorf("expected error to mention seat_id, got: %v", err)
	}
}

// TestOrgSpawn_MissingScope_NonZeroExit_NoAllowUnscoped is the CLI-level
// counterpart of AC-2b's minimum control gate: omitting both --scope and
// --allow-unscoped under the default (autonomous) config exits non-zero and
// mentions both flags. The gate now routes through reject() (self-review LOW
// finding), so it leaves exactly one `rejected` manifest event behind -- but
// still zero herdr calls and no spawn_started, since reject() never appends
// one.
func TestOrgSpawn_MissingScope_NonZeroExit_NoAllowUnscoped(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--state-dir", stateDir,
	)
	if err == nil {
		t.Fatalf("expected non-zero exit for a missing --scope under autonomous mode, output: %s", out)
	}
	if !strings.Contains(err.Error(), "--scope") || !strings.Contains(err.Error(), "--allow-unscoped") {
		t.Errorf("expected error to mention --scope and --allow-unscoped, got: %v", err)
	}

	if herdrLines := readLogLines(t, herdrLog); len(herdrLines) != 0 {
		t.Errorf("expected zero herdr calls for the gate rejection, got: %v", herdrLines)
	}
	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	if len(events) != 1 || events[0].Event != "rejected" {
		t.Errorf("expected exactly one rejected manifest event for the gate rejection, got: %v", events)
	}
}

// TestOrgSpawn_AllowUnscopedFlag_BypassesGateAndIsRecorded is the CLI-level
// counterpart of the AllowUnscoped bypass: the flag reaches SpawnParams and
// its use is recorded on the spawned event's Details.
func TestOrgSpawn_AllowUnscopedFlag_BypassesGateAndIsRecorded(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--allow-unscoped",
		"--state-dir", stateDir,
	)
	if err != nil {
		t.Fatalf("expected --allow-unscoped to bypass the gate, got %v (output: %s)", err, out)
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	last := events[len(events)-1]
	if last.Event != "spawned" {
		t.Fatalf("expected last event spawned, got %q", last.Event)
	}
	if !strings.Contains(last.Details, "allow_unscoped=true") {
		t.Errorf("expected spawned event Details to record allow_unscoped=true, got %q", last.Details)
	}
	if !strings.Contains(last.Details, "permission_mode=autonomous") {
		t.Errorf("expected spawned event Details to record permission_mode=autonomous, got %q", last.Details)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// --- ralph org start (AC-3: headless-lead spawn sugar) ----------------------

func TestOrgStart_HappyPath_SpawnsLeadSeat_SingleAgmsgJoin_NoHello(t *testing.T) {
	_, agmsgLog := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"start", "--org-id", "org-a", "--driver", "claude", "--model", "sonnet",
		"--cwd", t.TempDir(), "--scope", "org-wide",
		"--state-dir", stateDir,
		"dry-run 座席を1つ spawn し、typed message を送り、status を確認して disband せよ",
	)
	if err != nil {
		t.Fatalf("start failed: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, `spawned seat "lead"`) {
		t.Errorf("expected spawned confirmation naming the lead seat, got: %s", out)
	}
	if !strings.Contains(out, "ralph org status --org-id org-a") {
		t.Errorf("expected a status hint in the output, got: %s", out)
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	last := events[len(events)-1]
	if last.Event != "spawned" || last.SeatID != "lead" || last.Role != "lead" {
		t.Fatalf("expected last event spawned for seat_id=role=lead, got %+v", last)
	}

	// leadSelfSpawn (internal/org/spawn.go's Spawn): exactly one agmsg
	// invocation (the seat's own Join, which IS the lead-identity join) --
	// no separate ensureLeadJoined call, no HELLO send.
	agmsgLines := readLogLines(t, agmsgLog)
	if n := countLinesWithPrefix(agmsgLines, "ralph-org-a"); n != 1 {
		t.Fatalf("expected exactly 1 agmsg invocation for a lead-self spawn, got %d (log: %v)", n, agmsgLines)
	}
	if !strings.Contains(agmsgLines[0], "ralph-org-a lead claude-code") {
		t.Fatalf("expected the single agmsg call to be lead's own Join, got %v", agmsgLines)
	}
}

func TestOrgStart_TaskAndEnvelopeLandInPromptFile(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	task := "dry-run 座席を1つ spawn し、typed message を送り、status を確認して disband せよ"

	out, err := runOrgCmd(t,
		"start", "--org-id", "org-a", "--driver", "claude", "--model", "sonnet",
		"--cwd", t.TempDir(), "--scope", "org-wide",
		"--state-dir", stateDir,
		task,
	)
	if err != nil {
		t.Fatalf("start failed: %v (output: %s)", err, out)
	}

	data, rerr := os.ReadFile(herdrLog)
	if rerr != nil {
		t.Fatalf("read herdr log: %v", rerr)
	}
	if !strings.Contains(string(data), "役割指示を読み込んで従ってください: ") {
		t.Fatalf("expected AgentStart argv to carry the prompt-file pointer, got:\n%s", string(data))
	}

	promptPath := filepath.Join(stateDir, "prompts", "org-a_lead.md")
	promptData, perr := os.ReadFile(promptPath)
	if perr != nil {
		t.Fatalf("expected prompt file at %q: %v", promptPath, perr)
	}
	promptText := string(promptData)
	for _, want := range []string{task, "model_pool:", "max_seats:", "permission default:", "org-a"} {
		if !strings.Contains(promptText, want) {
			t.Errorf("expected lead prompt file to contain %q, got:\n%s", want, promptText)
		}
	}
}

func TestOrgStart_ModelFlagOmitted_DefaultsToFirstMatchingPoolEntry(t *testing.T) {
	setupOrgStubPATH(t)
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	configPath := filepath.Join(dir, "ralph.toml")
	content := "[org]\n" +
		"max_seats = 5\n" +
		"driver_pool = [\"claude\", \"codex\"]\n\n" +
		"[[org.model_pool]]\n" +
		"driver = \"codex\"\n" +
		"model = \"gpt-5-codex\"\n\n" +
		"[[org.model_pool]]\n" +
		"driver = \"claude\"\n" +
		"model = \"opus\"\n\n" +
		"[[org.model_pool]]\n" +
		"driver = \"claude\"\n" +
		"model = \"sonnet\"\n"
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	out, err := runOrgCmd(t,
		"start", "--org-id", "org-a", "--driver", "claude",
		"--cwd", t.TempDir(), "--scope", "org-wide",
		"--state-dir", stateDir, "--config", configPath,
		"task text",
	)
	if err != nil {
		t.Fatalf("start failed: %v (output: %s)", err, out)
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	last := events[len(events)-1]
	// The first claude entry in declared model_pool order is "opus" (codex's
	// entry precedes it but does not match --driver claude).
	if last.Model != "opus" {
		t.Fatalf("expected default --model resolution to pick the first matching pool entry (opus), got %q", last.Model)
	}
}

func TestOrgStart_ModelFlagExplicit_OverridesDefault(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"start", "--org-id", "org-a", "--driver", "claude", "--model", "haiku",
		"--cwd", t.TempDir(), "--scope", "org-wide",
		"--state-dir", stateDir,
		"task text",
	)
	if err != nil {
		t.Fatalf("start failed: %v (output: %s)", err, out)
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	last := events[len(events)-1]
	if last.Model != "haiku" {
		t.Fatalf("expected explicit --model to win over the default, got %q", last.Model)
	}
}

func TestOrgStart_NoMatchingModelPoolEntryForDriver_NonZeroExit(t *testing.T) {
	setupOrgStubPATH(t)
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	// writeOrgConfig's model_pool only contains a claude/sonnet entry -- no
	// codex entry exists for --driver codex to match.
	configPath := writeOrgConfig(t, dir, 5)

	out, err := runOrgCmd(t,
		"start", "--org-id", "org-a", "--driver", "codex",
		"--cwd", t.TempDir(), "--scope", "org-wide",
		"--state-dir", stateDir, "--config", configPath,
		"task text",
	)
	if err == nil {
		t.Fatalf("expected non-zero exit when no [org].model_pool entry matches --driver, output: %s", out)
	}
	if !strings.Contains(err.Error(), `driver "codex"`) {
		t.Errorf("expected error to name the unmatched driver, got: %v", err)
	}
}

func TestOrgStart_MissingScope_AutonomousDefault_RejectedUnlessAllowUnscoped(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"start", "--org-id", "org-a", "--driver", "claude", "--model", "sonnet",
		"--cwd", t.TempDir(),
		"--state-dir", stateDir,
		"task text",
	)
	if err == nil {
		t.Fatalf("expected non-zero exit for a missing --scope under the default autonomous permission mode, output: %s", out)
	}
	if !strings.Contains(err.Error(), "--scope") || !strings.Contains(err.Error(), "--allow-unscoped") {
		t.Errorf("expected error to mention --scope and --allow-unscoped, got: %v", err)
	}

	out2, err2 := runOrgCmd(t,
		"start", "--org-id", "org-a", "--driver", "claude", "--model", "sonnet",
		"--cwd", t.TempDir(), "--allow-unscoped",
		"--state-dir", stateDir,
		"task text",
	)
	if err2 != nil {
		t.Fatalf("expected --allow-unscoped to bypass the gate, got %v (output: %s)", err2, out2)
	}
}

func TestOrgStart_BlankTaskArg_NonZeroExit(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"start", "--org-id", "org-a", "--driver", "claude", "--model", "sonnet",
		"--cwd", t.TempDir(), "--scope", "org-wide",
		"--state-dir", stateDir,
		"   ",
	)
	if err == nil {
		t.Fatalf("expected non-zero exit for a blank task argument, output: %s", out)
	}
}

func TestOrgStart_RequiresCwd(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"start", "--org-id", "org-a", "--driver", "claude", "--model", "sonnet",
		"--scope", "org-wide",
		"--state-dir", stateDir,
		"task text",
	)
	if err == nil {
		t.Fatalf("expected non-zero exit for a missing --cwd, output: %s", out)
	}
	if !strings.Contains(err.Error(), "--cwd") {
		t.Errorf("expected error to mention --cwd, got: %v", err)
	}
}

func TestOrgStart_RequiresOrgID(t *testing.T) {
	out, err := runOrgCmd(t, "start", "--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(), "task text")
	if err == nil {
		t.Fatalf("expected non-zero exit for a missing --org-id, output: %s", out)
	}
}

// TestOrgWatch_RequiresOrgID pins the same --org-id persistent-flag gate
// (requireOrgID) applying to the new `watch` subcommand.
func TestOrgWatch_RequiresOrgID(t *testing.T) {
	out, err := runOrgCmd(t, "watch", "--once")
	if err == nil {
		t.Fatalf("expected non-zero exit for a missing --org-id, output: %s", out)
	}
	if !strings.Contains(err.Error(), "--org-id") {
		t.Errorf("expected error to mention --org-id, got: %v", err)
	}
}

// TestOrgWatch_Once_RunsExactlyOneCycleAndWritesStatus is the AC-3/--once
// CLI smoke: one pulse cycle runs against a real (stub) herdr/agmsg PATH and
// exits zero, having written watch-status.json (cycles=1) to the resolved
// state directory alongside manifest.jsonl.
func TestOrgWatch_Once_RunsExactlyOneCycleAndWritesStatus(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	out, err := runOrgCmd(t, "watch", "--org-id", "org-a", "--once", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("watch --once failed: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "watching org \"org-a\"") {
		t.Errorf("expected watch banner in output, got: %s", out)
	}

	statusPath := filepath.Join(stateDir, "watch-status-org-a.json")
	data, err := os.ReadFile(statusPath)
	if err != nil {
		t.Fatalf("read %s: %v", statusPath, err)
	}
	var status struct {
		OrgID  string `json:"org_id"`
		Cycles int    `json:"cycles"`
	}
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("parse %s: %v", statusPath, err)
	}
	if status.OrgID != "org-a" || status.Cycles != 1 {
		t.Fatalf("expected org_id=org-a cycles=1 after --once, got %+v (raw: %s)", status, data)
	}
}

// TestOrgWatch_Once_BannerShowsStateDirSource pins the tech-debt
// "watchdog deferred LOW (1)" fix: org.ResolveOrgStateDir's second return
// value (the resolved precedence tier) used to be discarded (`_`) at this
// call site, so the startup banner never told an operator which of
// flag/env/git-toplevel/cwd actually produced state-dir. --state-dir is
// passed explicitly here, so the expected source is "flag" (see
// ResolveOrgStateDir's own doc comment for the precedence order).
func TestOrgWatch_Once_BannerShowsStateDirSource(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	out, err := runOrgCmd(t, "watch", "--org-id", "org-a", "--once", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("watch --once failed: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "(source: flag)") {
		t.Errorf("expected the watch banner to name the state-dir source (\"(source: flag)\" for an explicit --state-dir), got: %s", out)
	}
}

// --- newWatchdogHooks (PR④ Slice 4, AC-6 wiring) -----------------------

// writeClaudeStub writes a fake `claude` executable to a fresh temp dir and
// prepends it to PATH -- the same PATH-stubbed convention
// internal/org/driver/driver_test.go already uses for herdr/agmsg
// availability probes (and internal/org/watcher_test.go's own copy of this
// helper for RunWatcher's tests: each package that needs to stub an
// external CLI on PATH keeps its own small helper rather than sharing one
// across package boundaries).
func writeClaudeStub(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatalf("write claude stub: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// newWatchdogTestOrg builds an *org.Org wired the same way newOrgRuntime
// does (real driver.ExecRunner adapters) but without needing a *cobra.Command
// -- for tests exercising newWatchdogHooks directly, bypassing the full
// `ralph org watch` command. IntervalSeconds is set but no longer bounds
// RunWatcher's own timeout (org.watcherInvokeTimeout, a fixed 60s var
// independent of the interval -- see internal/org/watcher.go's package
// note); tests that deliberately hang the claude stub shrink
// org.watcherInvokeTimeout directly instead (internal/org/watcher_test.go).
func newWatchdogTestOrg(t *testing.T, stateDir string) *org.Org {
	t.Helper()
	runner := driver.ExecRunner{}
	cfg := config.Default().Org
	cfg.Watchdog.WatcherEnabled = true
	cfg.Watchdog.WatcherModel = "haiku"
	cfg.Watchdog.IntervalSeconds = 2
	return &org.Org{
		Config:   cfg,
		Manifest: org.NewManifestStoreAtPath(org.ManifestPathIn(stateDir)),
		Receipts: org.NewReceiptStoreAtPath(org.ReceiptsPathIn(stateDir)),
		Herdr:    driver.Herdr{R: runner},
		Agmsg:    driver.Agmsg{R: runner, Home: driver.ResolveAgmsgHome(cfg.AgmsgHome)},
	}
}

func receiptCount(t *testing.T, rt *org.Org) int {
	t.Helper()
	rr, err := rt.Receipts.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	return len(rr.Receipts)
}

// TestNewWatchdogHooks_WatcherDisabled_NoOp pins that OnSemanticTrigger is
// left nil entirely when [org.watchdog].watcher_enabled is false -- the
// pulse layer must never invoke an LLM on its own.
func TestNewWatchdogHooks_WatcherDisabled_NoOp(t *testing.T) {
	rt := newWatchdogTestOrg(t, t.TempDir())
	rt.Config.Watchdog.WatcherEnabled = false

	hooks, wg := newWatchdogHooks(context.Background(), rt, io.Discard)
	if hooks.OnSemanticTrigger != nil {
		t.Fatal("expected nil OnSemanticTrigger when watcher_enabled is false")
	}
	// The returned WaitGroup (self-review M-5) must still be usable (never
	// nil) and immediately satisfied when the watcher is disabled -- nothing
	// is ever added to it, so Wait() must return without blocking.
	wg.Wait()
}

// TestNewWatchdogHooks_Dispatch_NeverBlocksCaller pins Codex advisory
// finding 3 at the CLI wiring level: OnSemanticTrigger returns near-
// instantly even though the underlying claude invocation it kicks off hangs
// well past that -- the pulse loop that calls this hook is never delayed by
// a slow/hung watcher.
func TestNewWatchdogHooks_Dispatch_NeverBlocksCaller(t *testing.T) {
	writeClaudeStub(t, "#!/bin/sh\nsleep 3\n")
	rt := newWatchdogTestOrg(t, t.TempDir())

	var stderr bytes.Buffer
	hooks, wg := newWatchdogHooks(context.Background(), rt, &stderr)

	start := time.Now()
	hooks.OnSemanticTrigger("org-a", "seat-1", "stall", "seat stalled 20m")
	elapsed := time.Since(start)

	if elapsed > 200*time.Millisecond {
		t.Fatalf("OnSemanticTrigger blocked its caller for %v (want near-instant dispatch)", elapsed)
	}
	if n := receiptCount(t, rt); n != 0 {
		t.Fatalf("expected no receipt yet immediately after dispatch (goroutine still mid-sleep), got %d", n)
	}

	// wg.Wait() (self-review M-5 fix -- the same WaitGroup newOrgWatchCmd's
	// RunE waits on before `--once` returns) blocks until the background
	// goroutine has actually finished, including its own hung claude
	// invocation, proving the WaitGroup is a reliable completion signal
	// rather than the test polling for a side effect.
	wg.Wait()
	if n := receiptCount(t, rt); n < 1 {
		t.Fatalf("expected the goroutine to have appended a watcher_error receipt by the time wg.Wait() returns, got %d", n)
	}
}

// TestNewWatchdogHooks_CtxCancelled_RunWatcherReturnsQuickly pins the
// self-review cycle-2 M2-2 fix: newWatchdogHooks must thread its ctx
// argument (newOrgWatchCmd passes cmd.Context()) into the tracked
// goroutine's RunWatcher call instead of a fixed context.Background(). A
// pre-cancelled ctx here simulates SIGINT having already cancelled the
// command's context before the goroutine's RunWatcher call starts; if the
// fix regressed to context.Background(), this would ignore the
// cancellation and only return once the 3s claude stub finishes (bounded by
// watcherInvokeTimeout, not by ctx).
func TestNewWatchdogHooks_CtxCancelled_RunWatcherReturnsQuickly(t *testing.T) {
	writeClaudeStub(t, "#!/bin/sh\nsleep 3\n")
	rt := newWatchdogTestOrg(t, t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // simulate a SIGINT-cancelled cmd.Context() before dispatch

	var stderr bytes.Buffer
	hooks, wg := newWatchdogHooks(ctx, rt, &stderr)

	start := time.Now()
	hooks.OnSemanticTrigger("org-a", "seat-1", "stall", "seat stalled 20m")
	wg.Wait()
	elapsed := time.Since(start)

	if elapsed > 1*time.Second {
		t.Fatalf("RunWatcher did not honor the cancelled command context: took %v (want well under the 3s stub sleep)", elapsed)
	}
	if n := receiptCount(t, rt); n < 1 {
		t.Fatalf("expected a watcher_error receipt for the cancelled-context invocation, got %d", n)
	}
}

// TestNewWatchdogHooks_SingleFlight_SecondTriggerSkippedWhileBusy pins the
// "at most one in flight" requirement: a second semantic trigger arriving
// while the first is still in flight is skipped (recorded to stderr), never
// queued or run concurrently. The single-flight compare-and-swap happens
// synchronously in OnSemanticTrigger itself (before any goroutine is
// spawned), so this is deterministic regardless of how fast the first
// call's underlying watcher invocation actually completes.
func TestNewWatchdogHooks_SingleFlight_SecondTriggerSkippedWhileBusy(t *testing.T) {
	writeClaudeStub(t, "#!/bin/sh\n"+
		"cat <<'EOF'\n"+
		`{"result":"{\"verdict\":\"normal\",\"reason\":\"ok\"}"}`+"\n"+
		"EOF\n")
	rt := newWatchdogTestOrg(t, t.TempDir())

	var stderr bytes.Buffer
	hooks, wg := newWatchdogHooks(context.Background(), rt, &stderr)

	hooks.OnSemanticTrigger("org-a", "seat-1", "stall", "evidence-1")
	hooks.OnSemanticTrigger("org-a", "seat-2", "stall", "evidence-2")

	got := stderr.String()
	if n := strings.Count(got, "already in flight"); n != 1 {
		t.Fatalf("expected exactly one single-flight skip message, got %d in: %s", n, got)
	}
	if !strings.Contains(got, "seat-2") {
		t.Errorf("expected the skip message to name the skipped seat (seat-2), got: %s", got)
	}
	if strings.Contains(got, `seat "seat-1"`) {
		t.Errorf("did not expect the first (accepted) trigger to be recorded as skipped: %s", got)
	}

	// Drain the accepted (seat-1) goroutine before the test returns.
	wg.Wait()
	if n := receiptCount(t, rt); n < 1 {
		t.Fatalf("expected the accepted goroutine to have appended a receipt, got %d", n)
	}
}

// TestNewWatchdogHooks_AbnormalVerdict_SendsAlertToLead pins that an
// abnormal verdict (anything but normal) is delivered to lead as a typed
// ALERT via rt.SendWatchdogAlert -- identity-level Agmsg.Send from the
// watchdog identity to lead, observed here through the agmsg send.sh stub's
// argv log. Bug 1 regression: no lead SEAT is spawned at all (the normal
// session-promoted-lead org shape -- only the lead agmsg identity itself
// exists, registered via ensureLeadJoined at org-start time, never a spawned
// SEAT). The old seat-steering Send verb resolved its To target via
// findSeat, which fails for a non-existent "lead" seat and silently drops
// the ALERT (live smoke: agmsg history had zero ALERTs while escalations
// fired); identity-level Agmsg.Send bypasses that seat lookup entirely, so
// the ALERT must still land here.
func TestNewWatchdogHooks_AbnormalVerdict_SendsAlertToLead(t *testing.T) {
	_, agmsgLog := setupOrgStubPATH(t)
	writeClaudeStub(t, "#!/bin/sh\n"+
		"cat <<'EOF'\n"+
		`{"result":"{\"verdict\":\"circular\",\"reason\":\"seat repeating the same edit\"}"}`+"\n"+
		"EOF\n")

	stateDir := t.TempDir()
	rt := newWatchdogTestOrg(t, stateDir)
	var stderr bytes.Buffer
	hooks, wg := newWatchdogHooks(context.Background(), rt, &stderr)

	hooks.OnSemanticTrigger("org-a", "seat-1", "scope_change", "seat worktree changed")

	// wg.Wait() blocks until the ALERT's own Send call has reached the
	// agmsg stub (not just RunWatcher's earlier receipt append) -- Send
	// runs after RunWatcher returns, inside the same background goroutine
	// this WaitGroup tracks (self-review M-5).
	wg.Wait()

	data, err := os.ReadFile(agmsgLog)
	if err != nil {
		t.Fatalf("read agmsg log: %v", err)
	}
	log := string(data)
	if !strings.Contains(log, "watchdog lead TYPE: ALERT") {
		t.Errorf("expected an ALERT sent from the watchdog identity to lead, agmsg log: %s", log)
	}
	if !strings.Contains(log, "CONDITION: watcher_scope_change") {
		t.Errorf("expected the ALERT to record the watcher condition, agmsg log: %s", log)
	}
	if !strings.Contains(log, "verdict=circular") {
		t.Errorf("expected the ALERT body to include the watcher's verdict, agmsg log: %s", log)
	}
	if strings.Contains(stderr.String(), "watchdog:") {
		t.Errorf("expected no watchdog error/skip messages on the happy path, got: %s", stderr.String())
	}
}

// --- ralph org report (AC-4) -------------------------------------------

func TestOrgReport_CLI_WritesFileWithRosterTimelineAndReceipts(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope",
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "reports")
	out, err := runOrgCmd(t, "report", "--org-id", "org-a", "--state-dir", stateDir, "--out", outDir)
	if err != nil {
		t.Fatalf("report failed: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "wrote ") {
		t.Errorf("expected output to confirm the written path, got: %s", out)
	}

	entries, rerr := os.ReadDir(outDir)
	if rerr != nil {
		t.Fatalf("read out dir: %v", rerr)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 report file, got %v", entries)
	}
	if !strings.HasPrefix(entries[0].Name(), "org-manifest-org-a-") {
		t.Errorf("expected report filename to start with org-manifest-org-a-, got %q", entries[0].Name())
	}
	data, derr := os.ReadFile(filepath.Join(outDir, entries[0].Name()))
	if derr != nil {
		t.Fatalf("read report file: %v", derr)
	}
	content := string(data)
	for _, want := range []string{
		"# org report: org-a", "## Roster", "seat-1", "## Event timeline",
		"spawn_started", "## Model receipts", "## Known residuals", "active seats: 1",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("expected report content to contain %q, got:\n%s", want, content)
		}
	}
}

func TestOrgReport_CLI_EmptyOrg_StillWritesReport(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "state")
	outDir := filepath.Join(t.TempDir(), "reports")

	out, err := runOrgCmd(t, "report", "--org-id", "org-empty", "--state-dir", stateDir, "--out", outDir)
	if err != nil {
		t.Fatalf("report failed for an org with zero events: %v (output: %s)", err, out)
	}

	entries, rerr := os.ReadDir(outDir)
	if rerr != nil {
		t.Fatalf("read out dir: %v", rerr)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 report file even for an empty org, got %v", entries)
	}
	data, derr := os.ReadFile(filepath.Join(outDir, entries[0].Name()))
	if derr != nil {
		t.Fatalf("read report file: %v", derr)
	}
	if !strings.Contains(string(data), "no events recorded for this org") {
		t.Errorf("expected the empty-org report to contain the no-events note, got:\n%s", string(data))
	}
}

func TestOrgReport_CLI_RequiresOrgID(t *testing.T) {
	out, err := runOrgCmd(t, "report", "--out", filepath.Join(t.TempDir(), "reports"))
	if err == nil {
		t.Fatalf("expected non-zero exit for a missing --org-id, output: %s", out)
	}
}
