package cli

import (
	"bytes"
	"os"
	"path/filepath"
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

// herdrStub and agmsgStub are POSIX-sh fake binaries: they log their argv to
// a file (path from an env var) and emit canned output, so the org verbs
// exercise the real driver.ExecRunner -> exec.Command path end to end
// without needing herdr/agmsg installed (CI has neither). ORG_STUB_FAIL, set
// per test case, injects a failure at exactly one subcommand boundary.
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

const agmsgStub = `#!/bin/sh
if [ -n "$ORG_AGMSG_LOG" ]; then
  echo "$@" >> "$ORG_AGMSG_LOG"
fi
if [ "$ORG_STUB_FAIL" = "agmsg:send" ]; then
  for a in "$@"; do
    if [ "$a" = "send" ]; then
      echo "stub failure: agmsg send" >&2
      exit 1
    fi
  done
fi
echo ""
exit 0
`

// setupOrgStubPATH writes stub herdr/agmsg binaries to a fresh temp dir,
// prepends it to PATH, and points ORG_HERDR_LOG/ORG_AGMSG_LOG at fresh log
// files. Returns the log file paths so tests can assert on recorded argv.
func setupOrgStubPATH(t *testing.T) (herdrLog, agmsgLog string) {
	t.Helper()
	dir := t.TempDir()

	writeStubScript(t, filepath.Join(dir, "herdr"), herdrStub)
	writeStubScript(t, filepath.Join(dir, "agmsg"), agmsgStub)

	herdrLog = filepath.Join(dir, "herdr.log")
	agmsgLog = filepath.Join(dir, "agmsg.log")

	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ORG_HERDR_LOG", herdrLog)
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
	want := []string{"spawn_started", "org_workspace_created", "spawn_step", "spawn_step", "spawn_step", "spawned"}
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
	// "--team" prefix that starts every agmsgArgs call instead.
	agmsgLines := readLogLines(t, agmsgLog)
	if n := countLinesWithPrefix(agmsgLines, "--team"); n != 2 {
		t.Fatalf("expected 2 agmsg invocations (one HELLO per seat), got %d (log: %v)", n, agmsgLines)
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
	herdrLog, _ := setupOrgStubPATH(t)
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
