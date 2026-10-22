package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// TestOrgSpawnAndStart_TimeoutMSHelp_DescribesTheWholeSpawnDeadline pins
// plan 2026-10-22-seat-spawn-close AC5's help: --timeout-ms of spawn and of
// start is one deadline for the whole spawn, shared by its herdr and agmsg
// steps (start's leader spawn has no HELLO and one agmsg join), with the
// cleanup after a failure bounded on its own, and never a per-step timeout.
func TestOrgSpawnAndStart_TimeoutMSHelp_DescribesTheWholeSpawnDeadline(t *testing.T) {
	for _, tc := range []struct {
		verb, want string
	}{
		{"spawn", "--timeout-ms int deadline in milliseconds for the whole spawn, shared by every herdr and agmsg step " +
			"(workspace and tab create, agent start, agmsg joins, HELLO); " +
			"the cleanup after a failed step is bounded separately, 10s per call (default 60000)"},
		{"start", "--timeout-ms int deadline in milliseconds for the whole spawn, shared by every herdr and agmsg step " +
			"(workspace and tab create, agent start, agmsg join); " +
			"the cleanup after a failed step is bounded separately, 10s per call (default 60000)"},
	} {
		t.Run(tc.verb, func(t *testing.T) {
			out, err := runOrgCmd(t, tc.verb, "--help")
			if err != nil {
				t.Fatalf("%s --help failed: %v", tc.verb, err)
			}
			help := strings.Join(strings.Fields(out), " ")
			if !strings.Contains(help, tc.want) {
				t.Errorf("%s --help should contain %q, got:\n%s", tc.verb, tc.want, out)
			}
			if strings.Contains(help, "per-step") {
				t.Errorf("%s --help should not call --timeout-ms per-step, got:\n%s", tc.verb, out)
			}
		})
	}
}

// TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint covers AC5's error
// end to end, through the real driver.ExecRunner: an agmsg send.sh that
// never answers runs the spawn past --timeout-ms, ExecRunner kills it at the
// deadline, and both the `spawn failed:` line on stdout and the returned
// error name the deadline in ms and say to raise --timeout-ms, while
// Details do not. 3000 ms leaves the stub steps before the HELLO room on a
// loaded machine.
func TestOrgSpawn_HelloPastTimeoutMS_PrintsTheDeadlineHint(t *testing.T) {
	_, _ = setupOrgStubPATH(t)
	writeStubScript(t, filepath.Join(os.Getenv("RALPH_ORG_AGMSG_HOME"), "scripts", "send.sh"), "#!/bin/sh\nexec sleep 30\n")
	stateDir := filepath.Join(t.TempDir(), "state")

	stdout, _, err := runOrgCmdStreams(t,
		"spawn", "--org-id", "org-a", "--id", "seat-1", "--role", "worker",
		"--driver", "claude", "--model", "sonnet", "--cwd", t.TempDir(),
		"--scope", "test-scope", "--timeout-ms", "3000",
		"--state-dir", stateDir,
	)
	if err == nil {
		t.Fatalf("expected the spawn to fail at the HELLO, stdout: %s", stdout)
	}
	const hint = "(the spawn's --timeout-ms deadline of 3000 ms, shared by every herdr and agmsg step, ran out; retry with a larger --timeout-ms)"
	if !strings.Contains(err.Error(), "spawn step agmsg_announce failed: ") || !strings.Contains(err.Error(), hint) {
		t.Fatalf("expected the error to name the agmsg_announce step and the hint %q, got %v", hint, err)
	}
	if !strings.Contains(stdout, "spawn failed: ") || !strings.Contains(stdout, hint) {
		t.Errorf("expected stdout's spawn failed: line to carry the hint, got %q", stdout)
	}

	events := readManifestEvents(t, org.ManifestPathIn(stateDir))
	last := events[len(events)-1]
	if last.Event != org.EventSpawnFailed {
		t.Fatalf("expected spawn_failed last, got %+v", last)
	}
	if !strings.Contains(last.Details, "step=agmsg_announce ") || !strings.Contains(last.Details, "pane=closed") {
		t.Errorf("expected Details to record the agmsg_announce step and the closed pane, got %q", last.Details)
	}
	if strings.Contains(last.Details, "--timeout-ms") {
		t.Errorf("expected the hint to stay out of Details, got %q", last.Details)
	}
}
