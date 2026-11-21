package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// TestOrgSend_OriginHumanDecision_SentToTheLeaderAndRecorded covers AC1
// through `ralph org send` with the stub herdr: a DECISION with the header
// ORIGIN: human to the leader is typed, and its `sent` event's Details start
// with `origin=human ` and its Text holds the whole message.
func TestOrgSend_OriginHumanDecision_SentToTheLeaderAndRecorded(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	if out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", org.LeaderIdentity, "--role", org.LeaderIdentity,
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope", "--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn the leader: %v (output: %s)", err, out)
	}
	before := len(readManifestEvents(t, org.ManifestPathIn(stateDir)))
	sendTextsBefore := countLinesWithPrefix(readLogLines(t, herdrLog), "pane send-text")

	text := "TYPE: DECISION\nORIGIN: human\n\nkeep the reviewer on codex (the human, 2026-11-21 10:00)"
	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", org.LeaderIdentity, "--text", text,
		"--enter-delay-ms", "1", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("send the human's decision: %v (output: %s)", err, out)
	}
	if got := countLinesWithPrefix(readLogLines(t, herdrLog), "pane send-text"); got != sendTextsBefore+1 {
		t.Fatalf("pane send-text calls %d -> %d; want the decision typed once", sendTextsBefore, got)
	}
	events := readManifestEvents(t, org.ManifestPathIn(stateDir))[before:]
	if len(events) != 1 || events[0].Event != "sent" || events[0].SeatID != org.LeaderIdentity {
		t.Fatalf("new events = %+v; want one sent event to the leader", events)
	}
	if want := "origin=human " + text; events[0].Details != want {
		t.Fatalf("sent Details = %q\nwant %q", events[0].Details, want)
	}
	if events[0].Text != text {
		t.Fatalf("sent Text = %q\nwant %q", events[0].Text, text)
	}
}

// TestOrgSend_OriginHumanToASeat_RefusedWithNothingTypedOrRecorded covers
// AC2 through `ralph org send`: ORIGIN: human to a seat that is not the
// leader exits non-zero with the rule in the error, and nothing is typed or
// recorded.
func TestOrgSend_OriginHumanToASeat_RefusedWithNothingTypedOrRecorded(t *testing.T) {
	herdrLog, _ := setupOrgStubPATH(t)
	stateDir := filepath.Join(t.TempDir(), "state")
	if out, err := runOrgCmd(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope", "--state-dir", stateDir,
	); err != nil {
		t.Fatalf("spawn seat-1: %v (output: %s)", err, out)
	}
	before := len(readManifestEvents(t, org.ManifestPathIn(stateDir)))
	herdrBefore := len(readLogLines(t, herdrLog))

	out, err := runOrgCmd(t, "send", "--org-id", "org-a", "--to", "seat-1",
		"--text", "TYPE: DECISION\nORIGIN: human\n\nkeep the reviewer on codex", "--enter-delay-ms", "1", "--state-dir", stateDir)
	want := `org: send: the message with ORIGIN: human is for seat "seat-1", not the leader; ` +
		`ORIGIN is allowed only as one "ORIGIN: human" header line on a TYPE: DECISION sent to the leader (--to leader); nothing was sent`
	if err == nil || err.Error() != want {
		t.Fatalf("send = %v (output: %s)\nwant the error %q", err, out, want)
	}
	if got := len(readManifestEvents(t, org.ManifestPathIn(stateDir))); got != before {
		t.Fatalf("events %d -> %d; want none recorded", before, got)
	}
	if lines := readLogLines(t, herdrLog); len(lines) != herdrBefore {
		t.Fatalf("herdr calls after the refusal: %q; want none", lines[herdrBefore:])
	}
	if strings.Contains(out, "sent message") {
		t.Fatalf("output = %s; want no sent confirmation", out)
	}
}
