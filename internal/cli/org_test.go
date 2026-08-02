package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
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
const herdrStub = `#!/bin/sh
if [ -n "$ORG_HERDR_LOG" ]; then
  echo "$@" >> "$ORG_HERDR_LOG"
fi
if [ -n "$ORG_STUB_FAIL" ] && [ "$1:$2" = "$ORG_STUB_FAIL" ]; then
  echo "stub failure: $1 $2" >&2
  exit 1
fi
case "$1 $2" in
  "workspace create") echo "ws-stub-1" ;;
  "tab create") echo "pane-stub-1" ;;
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

// agmsgDespawnStub is the fake `scripts/despawn.sh` -- driver.Agmsg shells
// out to `bash <home>/scripts/despawn.sh TEAM FROM NAME`. Honors
// ORG_STUB_FAIL=agmsg:despawn.
const agmsgDespawnStub = `#!/bin/sh
if [ -n "$ORG_AGMSG_LOG" ]; then
  echo "$@" >> "$ORG_AGMSG_LOG"
fi
if [ "$ORG_STUB_FAIL" = "agmsg:despawn" ]; then
  echo "stub failure: agmsg despawn" >&2
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
	writeStubScript(t, filepath.Join(agmsgHome, "scripts", "despawn.sh"), agmsgDespawnStub)
	if err := os.WriteFile(filepath.Join(agmsgHome, "VERSION"), []byte(agmsgVersionStub), 0o644); err != nil {
		t.Fatalf("write agmsg VERSION: %v", err)
	}
	agmsgLog = filepath.Join(dir, "agmsg.log")
	t.Setenv("RALPH_ORG_AGMSG_HOME", agmsgHome)
	t.Setenv("ORG_AGMSG_LOG", agmsgLog)

	t.Setenv("ORG_STUB_FAIL", "")

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
		"--state-dir", stateDir,
	)
	if err != nil {
		t.Fatalf("spawn seat-1 failed: %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "spawned seat \"seat-1\"") {
		t.Errorf("expected spawned confirmation in output, got: %s", out)
	}

	manifestPath := filepath.Join(stateDir, "manifest.jsonl")
	events := readManifestEvents(t, manifestPath)
	// spawn_step x5: tab_created, agent_started, agmsg_lead_joined,
	// agmsg_joined, agmsg_announced.
	want := []string{"spawn_started", "org_workspace_created", "spawn_step", "spawn_step", "spawn_step", "spawn_step", "spawn_step", "spawned"}
	if got := eventTypes(events); !equalStrings(got, want) {
		t.Fatalf("expected event sequence %v, got %v", want, got)
	}

	receiptsPath := filepath.Join(stateDir, "model-receipts.jsonl")
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

func TestOrgSpawn_FailureInjection_TabCreate_NoCompensation(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	t.Setenv("ORG_STUB_FAIL", "tab:create")
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--state-dir", stateDir,
	)
	if err == nil {
		t.Fatalf("expected non-zero exit on tab_create failure, output: %s", out)
	}

	events := readManifestEvents(t, filepath.Join(stateDir, "manifest.jsonl"))
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
		"--state-dir", stateDir,
	)
	if err == nil {
		t.Fatalf("expected non-zero exit on agent_start failure, output: %s", out)
	}

	events := readManifestEvents(t, filepath.Join(stateDir, "manifest.jsonl"))
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
		"--state-dir", stateDir,
	)
	if err == nil {
		t.Fatalf("expected non-zero exit on agmsg send failure, output: %s", out)
	}

	events := readManifestEvents(t, filepath.Join(stateDir, "manifest.jsonl"))
	last := events[len(events)-1]
	if last.Event != "spawn_failed" {
		t.Fatalf("expected last event spawn_failed, got %q", last.Event)
	}
	if !strings.Contains(last.Details, "step=agmsg_announce") {
		t.Errorf("expected Details to mention step=agmsg_announce, got %q", last.Details)
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
		"--state-dir", stateDir,
	)
	if err == nil {
		t.Fatalf("expected non-zero exit on agmsg join failure, output: %s", out)
	}

	events := readManifestEvents(t, filepath.Join(stateDir, "manifest.jsonl"))
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
		"--state-dir", stateDir, "--config", configPath,
	)
	if err == nil {
		t.Fatalf("expected non-zero exit for out-of-pool model, output: %s", out)
	}
	if !strings.Contains(out, "rejected") {
		t.Errorf("expected 'rejected' in output, got: %s", out)
	}

	receiptsPath := filepath.Join(stateDir, "model-receipts.jsonl")
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
		"--state-dir", stateDir, "--config", configPath,
	); err != nil {
		t.Fatalf("expected first org-a spawn to succeed under max_seats=1: %v", err)
	}

	out3, err3 := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-2", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--state-dir", stateDir, "--config", configPath,
	)
	if err3 == nil {
		t.Fatalf("expected non-zero exit: max_seats=1 already reached in org-a, output: %s", out3)
	}

	// Org isolation: org-b's first seat must not be blocked by org-a's count.
	if _, errB := runOrgCmd(t,
		"spawn", "--org-id", "org-b", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
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
		"--state-dir", stateDir, "--dry-run",
	)
	if err != nil {
		t.Fatalf("expected dry-run spawn to succeed with empty PATH, err=%v (output: %s)", err, out)
	}
	if !strings.Contains(out, "dry_run=true") {
		t.Errorf("expected dry_run=true in spawn output, got: %s", out)
	}

	events := readManifestEvents(t, filepath.Join(stateDir, "manifest.jsonl"))
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

func TestOrgStatus_EmptyPATH_FullRosterFromManifestWithCorruptCount(t *testing.T) {
	t.Setenv("PATH", "")
	stateDir := t.TempDir()
	manifestPath := filepath.Join(stateDir, "manifest.jsonl")

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
	manifestPath := filepath.Join(stateDir, "manifest.jsonl")
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

	// AC-5: disband's per-seat Stop must also best-effort despawn.sh the
	// seat -- despawn.sh's argv is "TEAM FROM NAME", so the seat_id
	// (seat-1) shows up as the last logged token.
	agmsgLines := readLogLines(t, agmsgLog)
	if !containsLine(agmsgLines, "ralph-org-a lead seat-1") {
		t.Errorf("expected a despawn.sh invocation with argv 'ralph-org-a lead seat-1', got agmsg log: %v", agmsgLines)
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
}

func TestOrgStop_UnknownSeat_NonZeroExit(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	out, err := runOrgCmd(t, "stop", "--org-id", "org-a", "--seat", "never-spawned", "--state-dir", stateDir)
	if err == nil {
		t.Fatalf("expected non-zero exit stopping an unknown seat, output: %s", out)
	}

	events := readManifestEvents(t, filepath.Join(stateDir, "manifest.jsonl"))
	if len(events) != 0 {
		t.Fatalf("expected no manifest event appended for an unknown-seat stop, got %v", events)
	}
}

func TestOrgStop_ExistingSeat_DespawnsAndRecordsOutcome(t *testing.T) {
	_, agmsgLog := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	out, err := runOrgCmd(t, "stop", "--org-id", "org-a", "--seat", "seat-1", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("stop failed: %v (output: %s)", err, out)
	}

	agmsgLines := readLogLines(t, agmsgLog)
	if !containsLine(agmsgLines, "ralph-org-a lead seat-1") {
		t.Errorf("expected a despawn.sh invocation with argv 'ralph-org-a lead seat-1' in the agmsg log, got: %v", agmsgLines)
	}

	events := readManifestEvents(t, filepath.Join(stateDir, "manifest.jsonl"))
	last := events[len(events)-1]
	if last.Event != "stopped" {
		t.Fatalf("expected last event stopped, got %q", last.Event)
	}
	if !strings.Contains(last.Details, "despawn=ok") {
		t.Errorf("expected Details to record a successful despawn, got %q", last.Details)
	}
}

// containsLine reports whether any line in lines equals want exactly.
func containsLine(lines []string, want string) bool {
	return slices.Contains(lines, want)
}

// TestOrgSpawn_RoleAndScopeFlags_ExpandTemplateAndRecordScope covers AC-4
// and the scope half of AC-7/design: `--role reviewer --scope ...` expands
// the embedded reviewer template (with org_id/seat_id/scope substituted)
// into the AgentStart argv the herdr stub receives, and the spawned event's
// Details records "scope=<value>". The herdr log is read as raw file
// content (not line-split) because the rendered prompt itself contains
// newlines.
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
	for _, want := range []string{"org-a", "reviewer-1", "internal/org/**", ".claude/rules/agent-messaging.md"} {
		if !strings.Contains(logText, want) {
			t.Errorf("expected AgentStart argv logged to contain %q, got:\n%s", want, logText)
		}
	}

	events := readManifestEvents(t, filepath.Join(stateDir, "manifest.jsonl"))
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
	if strings.Contains(logText, ".claude/rules/agent-messaging.md") {
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
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}
	eventsBefore := readManifestEvents(t, filepath.Join(stateDir, "manifest.jsonl"))

	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1", "--text", "not a valid protocol message", "--state-dir", stateDir)
	if err == nil {
		t.Fatalf("expected non-zero exit for a malformed --text, output: %s", out)
	}

	eventsAfter := readManifestEvents(t, filepath.Join(stateDir, "manifest.jsonl"))
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
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1", "--text", "not a valid protocol message", "--raw", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("expected --raw to bypass protocol validation, got %v (output: %s)", err, out)
	}

	events := readManifestEvents(t, filepath.Join(stateDir, "manifest.jsonl"))
	last := events[len(events)-1]
	if last.Event != "sent" {
		t.Fatalf("expected last event sent, got %q", last.Event)
	}
	if !strings.Contains(last.Details, "raw=true") {
		t.Errorf("expected the sent event's Details to record raw=true, got %q", last.Details)
	}
}

// TestOrgSend_ValidTypedMessage_Succeeds is the positive counterpart: a
// well-formed typed message is accepted without --raw.
func TestOrgSend_ValidTypedMessage_Succeeds(t *testing.T) {
	setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")

	if _, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn failed: %v", err)
	}

	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1",
		"--text", "TYPE: TASK\nTASK_ID: t-1\n\ndo the thing", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("expected a well-formed typed message to be accepted, got %v (output: %s)", err, out)
	}
	if !strings.Contains(out, "sent message to seat \"seat-1\"") {
		t.Errorf("expected sent confirmation in output, got: %s", out)
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

	herdrLines := readLogLines(t, herdrLog)
	if !containsLine(herdrLines, "agent wait org-a-never-spawned --until idle") {
		t.Errorf("expected the herdr log to show a wait call for the namespaced agent name, got: %v", herdrLines)
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
