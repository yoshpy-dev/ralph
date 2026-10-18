package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// Tests for start's cleanup of a current headless director that its check
// finds ended and attach's refusal text (plan
// docs/plans/active/2026-10-18-org-director-liveness.md, D6, D7, AC6, and
// the attach half of AC7) through the CLI. Every test runs the stub herdr
// (setupOrgStubPATH), never herdr itself, on a ledger under t.TempDir(); ps
// reads only this test binary and a child it started and waited for.

// stubHerdrAgentNotFound puts in front of the stub herdr on PATH a herdr
// whose `agent get` answers herdr's agent_not_found envelope with exit 1, as
// herdr does once the agent is gone, and logs it to the stub's log; every
// other call goes to the stub. Call it with the same t after
// setupOrgStubPATH (through directorStartLedger): it fails the test, before
// it writes its herdr, when the herdr PATH finds is not under this test's
// temp dir, so it never hands a call to a real herdr.
func stubHerdrAgentNotFound(t *testing.T) {
	t.Helper()
	stub, err := exec.LookPath("herdr")
	if err != nil {
		t.Fatalf("look up the stub herdr: %v", err)
	}
	dir := t.TempDir()
	if root := filepath.Dir(dir) + string(filepath.Separator); !strings.HasPrefix(stub, root) {
		t.Fatalf("the herdr on PATH is %s, not a stub under this test's temp dir %s: call setupOrgStubPATH first", stub, root)
	}
	writeStubScript(t, filepath.Join(dir, "herdr"), `#!/bin/sh
if [ "$1 $2" = "agent get" ]; then
  if [ -n "$ORG_HERDR_LOG" ]; then
    echo "$@" >> "$ORG_HERDR_LOG"
  fi
  echo "{\"error\":{\"code\":\"agent_not_found\",\"message\":\"agent $3 not found\"},\"id\":\"cli:agent:get\"}"
  exit 1
fi
exec `+shellQuoteForTest(stub)+` "$@"
`)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// shellQuoteForTest is s in single quotes for a stub script.
func shellQuoteForTest(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// deadChildPID is the pid of a child this test started and waited for: ps
// reads it gone (barring a reuse of the pid within the test).
func deadChildPID(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", "exit 0")
	if err := cmd.Run(); err != nil {
		t.Fatalf("run a child: %v", err)
	}
	return cmd.Process.Pid
}

// herdrLogSince is the stub herdr's log after its first n lines.
func herdrLogSince(t *testing.T, herdrLog string, n int) []string {
	t.Helper()
	return slices.Clone(readLogLines(t, herdrLog)[n:])
}

// AC6 through the CLI: d1, started by director start, has ended in herdr
// (agent get answers agent_not_found, and its pane is still there with its
// labels). The next start checks it, closes its pane and workspace, records
// it stopped, prints the line that says so before its own lines, and starts
// d2 in a workspace of its own.
func TestOrgDirectorStart_CleansUpAnEndedHeadlessDirectorAndStarts(t *testing.T) {
	stateDir, herdrLog, _ := directorStartLedger(t)
	tok := startDirectorOK(t, stateDir, "--driver", "claude", "--model", "opus")
	stubHerdrAgentNotFound(t)
	n := len(readLogLines(t, herdrLog))

	stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("director start over an ended d1: %v (stdout %s stderr %s)", err, stdout, stderr)
	}
	want := "cleaned up d1, which had ended (agent_ended): ctrl_c=ok, pane=closed, workspace=closed, and recorded it stopped\n" +
		"started d2 (driver=claude, model=opus, permission=autonomous, pane=pane-stub-2, workspace=ws-stub-2)\n"
	if !strings.HasPrefix(stdout, want) {
		t.Errorf("stdout =\n%s\nwant it to start with\n%s", stdout, want)
	}
	wantCalls := []string{"agent get director_" + tok, "pane get pane-stub-1", "tab get tab-stub-1", "workspace get ws-stub-1",
		"pane get pane-stub-1", "tab get tab-stub-1", "workspace get ws-stub-1", "workspace get ws-stub-1",
		"pane send-keys pane-stub-1 C-c", "pane close pane-stub-1", "workspace close ws-stub-1"}
	if got := herdrLogSince(t, herdrLog, n); len(got) < len(wantCalls) || !slices.Equal(got[:len(wantCalls)], wantCalls) {
		t.Errorf("herdr calls =\n%s\nwant them to start with\n%s", strings.Join(got, "\n"), strings.Join(wantCalls, "\n"))
	}
	got := directorEventKinds(directorEventsIn(t, stateDir))
	if want := []string{"attached d1", "workspace_created d1", "pane_created d1", "launched d1", "detached d1", "stopped d1",
		"attached d2", "workspace_created d2", "pane_created d2", "launched d2"}; !slices.Equal(got, want) {
		t.Errorf("director.jsonl = %v; want %v", got, want)
	}
}

// AC6 through the CLI: d1 is still starting, and the start process it
// recorded is gone (a child this test ran and waited for). The next start
// cleans it up with nothing recorded to close, says that herdr may still
// have a workspace labelled for d1, records it stopped, and starts d2.
func TestOrgDirectorStart_CleansUpARegistrationWhoseStartDied(t *testing.T) {
	stateDir, herdrLog, _ := directorStartLedger(t)
	reg, err := org.NewDirectorStore(stateDir).AttachHeadless(org.DirectorHeadlessAttach{
		Driver: "codex", Model: "gpt-6-astra", PermissionMode: org.PermissionModeGuarded, Wake: []string{org.DirectorWakePull},
		StarterPID: deadChildPID(t), StarterStartTime: "Thu Oct 15 21:51:09 2026", User: "alice", Cwd: "/work",
	})
	if err != nil {
		t.Fatalf("seed d1: %v", err)
	}

	stdout, stderr, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
	if err != nil {
		t.Fatalf("director start over a d1 whose start died: %v (stdout %s stderr %s)", err, stdout, stderr)
	}
	want := "cleaned up d1, which had ended (start_ended): ctrl_c=skipped, pane=none recorded, workspace=none recorded, and recorded it stopped; " +
		"herdr may still have a workspace labelled ralph_director_" + reg.Token + " that its start made before it could record it (herdr workspace list)\n" +
		"started d2 ("
	if !strings.HasPrefix(stdout, want) {
		t.Errorf("stdout =\n%s\nwant it to start with\n%s", stdout, want)
	}
	if actions := herdrActions(t, herdrLog); len(actions) != 0 {
		t.Errorf("herdr actions = %v; want none: d1 recorded nothing to close", actions)
	}
	got := directorEventKinds(directorEventsIn(t, stateDir))
	if want := []string{"attached d1", "detached d1", "stopped d1", "attached d2", "workspace_created d2", "pane_created d2", "launched d2"}; !slices.Equal(got, want) {
		t.Errorf("director.jsonl = %v; want %v", got, want)
	}
}

// AC6 through the CLI: a check that finds d1 alive (the stub herdr answers
// agent get) or cannot tell (agent get fails) refuses start with exit 1,
// nothing on stdout, director.jsonl as it was, and only the agent get sent
// to herdr; the refusal names the stop as before and ends with what the
// check found.
func TestOrgDirectorStart_RefusesADirectorItsCheckFindsAliveOrCannotTell(t *testing.T) {
	for _, tc := range []struct {
		name, fail string
		want       func(tok string) string
	}{
		{"alive", "", func(string) string { return "; start checked d1 just now: alive (agent_found)" }},
		{"unknown", "agent:get", func(tok string) string {
			return "; start checked d1 just now: unknown (agent_get_failed: herdr agent get director_" + tok + " failed: "
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir, herdrLog, _ := directorStartLedger(t)
			tok := startDirectorOK(t, stateDir, "--driver", "claude", "--model", "opus")
			t.Setenv("ORG_STUB_FAIL", tc.fail)
			n, before := len(readLogLines(t, herdrLog)), directorFileOrAbsent(t, stateDir)

			stdout, _, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
			if !errors.Is(err, org.ErrDirectorAttached) {
				t.Fatalf("director start over a d1 the check does not find ended = %v; want ErrDirectorAttached", err)
			}
			if want := "stop it with ralph org director stop --id d1 --state-dir '" + stateDir + "' from a terminal outside herdr, " +
				"then run start again (start has no --replace)" + tc.want(tok); !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not contain %q", err, want)
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			if after := directorFileOrAbsent(t, stateDir); after != before {
				t.Errorf("a refused start changed director.jsonl:\nbefore: %s\nafter:  %s", before, after)
			}
			if got := herdrLogSince(t, herdrLog, n); !slices.Equal(got, []string{"agent get director_" + tok}) {
				t.Errorf("herdr calls = %v; want only the agent get", got)
			}
		})
	}
}

// AC6 through the CLI: a cleanup that cannot close d1's pane (the stub fails
// its close) refuses start with exit 1 and nothing on stdout, records
// nothing, starts no d2, and names the herdr commands that close what is
// open and the stop, with this call's --state-dir, that records it
// afterwards.
func TestOrgDirectorStart_ACleanupThatCannotCloseNamesTheStop(t *testing.T) {
	stateDir, _, _ := directorStartLedger(t)
	startDirectorOK(t, stateDir, "--driver", "claude", "--model", "opus")
	stubHerdrAgentNotFound(t)
	t.Setenv("ORG_STUB_CLOSE_FAIL_IDS", "pane-stub-1")
	before := directorFileOrAbsent(t, stateDir)

	stdout, _, err := runOrgCmdStreams(t, "director", "start", "--driver", "claude", "--model", "opus", "--state-dir", stateDir)
	var cleanup *org.DirectorCleanupError
	if !errors.As(err, &cleanup) || !errors.Is(err, org.ErrDirectorAttached) {
		t.Fatalf("director start = %v; want start's *DirectorCleanupError", err)
	}
	for _, want := range []string{
		", headless running), which start found ended (agent_ended) when it checked it just now, and start could not clean it up, so it started no new director; " +
			"its cleanup did not finish: close pane \"pane-stub-1\": ",
		"; nothing more was recorded; close it by hand: herdr pane close pane-stub-1, then herdr workspace close ws-stub-1; " +
			"afterwards record the stop with ralph org director stop --id d1 --state-dir '" + stateDir + "'",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}
	if stdout != "" {
		t.Errorf("stdout = %q; want nothing: the cleanup was not recorded", stdout)
	}
	if after := directorFileOrAbsent(t, stateDir); after != before {
		t.Errorf("director.jsonl changed:\nbefore: %s\nafter:  %s", before, after)
	}
}

// A cleanup that found d1 changed after the check (org.ErrDirectorChangedSinceCheck)
// needs a race to reach through the CLI, so this tests startError: the stop
// command gets the call's --state-dir and the status command follows.
func TestOrgDirectorStart_ARegistrationChangedSinceTheCheckNamesStatus(t *testing.T) {
	cause := fmt.Errorf("org: director: start: %w: d1 (driver claude, attached 2026-10-18T08:00:00Z, headless running), which start found ended (agent_ended) "+
		"when it checked it just now, and start could not clean it up, so it started no new director", org.ErrDirectorAttached)
	err := &org.DirectorCleanupError{Verb: "start", ID: "d1", Launch: org.DirectorLaunchHerdr, Err: cause,
		CleanupErr:  fmt.Errorf("record d1 stopped: %w: d1 had 4 events, with d1 current when the check read it, and has 5 events now", org.ErrDirectorChangedSinceCheck),
		StopCommand: "ralph org director stop --id d1"}
	rt := &orgDirectorRuntime{hintStateDir: "/work/the ledger"}
	got := rt.startError(err)
	want := cause.Error() + "; its cleanup did not finish: record d1 stopped: the registration changed after start checked it: " +
		"d1 had 4 events, with d1 current when the check read it, and has 5 events now; nothing more was recorded; " +
		"run start again, which checks the registration as it is now, or, when it is still open and has ended, " +
		"record the stop with ralph org director stop --id d1 --state-dir '/work/the ledger'; " +
		"read the registration as it is now with ralph org director status --state-dir '/work/the ledger'"
	if got == nil || got.Error() != want {
		t.Fatalf("startError =\n%v\nwant\n%s", got, want)
	}
	if !errors.Is(got, org.ErrDirectorChangedSinceCheck) || !errors.Is(got, org.ErrDirectorAttached) {
		t.Errorf("startError = %v; want it to still wrap both sentinels", got)
	}
}

// startAttachedError adds the check (D6) after the way out, escaped, for a
// launch herdr and a launch cmd, and adds nothing without one.
func TestOrgDirectorStart_AnAttachedRefusalAddsTheCheck(t *testing.T) {
	herdr := org.DirectorRegistration{ID: "d1", Driver: "claude", AttachedAt: "2026-10-18T08:00:00Z", Launch: org.DirectorLaunchHerdr, State: org.DirectorStateRunning}
	cmdReg := herdr
	cmdReg.Launch, cmdReg.PID = org.DirectorLaunchCmd, 5151
	unknown := &org.DirectorCheck{ID: "d1", State: org.DirectorCheckUnknown, How: org.DirectorCheckHowAgentGetFailed, Detail: "herdr: \x1b[31mred"}
	alive := &org.DirectorCheck{ID: "d1", State: org.DirectorCheckAlive, How: org.DirectorCheckHowProcessFound}
	for _, tc := range []struct {
		name  string
		cur   org.DirectorRegistration
		check *org.DirectorCheck
		want  string
	}{
		{"herdr, unknown", herdr, unknown, "stop it with ralph org director stop --id d1 from a terminal outside herdr, then run start again (start has no --replace)" +
			`; start checked d1 just now: unknown (agent_get_failed: herdr: \x1b[31mred)`},
		{"cmd, alive", cmdReg, alive, "stop it with ralph org director stop --id d1 from a terminal other than the director's, then run start again (start has no --replace)" +
			"; start checked d1 just now: alive (process_found)"},
		{"no check", herdr, nil, "stop it with ralph org director stop --id d1 from a terminal outside herdr, then run start again (start has no --replace)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cur := tc.cur
			refusal := fmt.Errorf("org: director: start: %w: d1", org.ErrDirectorAttached)
			err := (&orgDirectorRuntime{}).startAttachedError(&org.DirectorAttachedError{Current: &cur, Err: refusal, Check: tc.check})
			if want := refusal.Error() + ": " + tc.want; err == nil || err.Error() != want {
				t.Fatalf("startAttachedError =\n%v\nwant\n%s", err, want)
			}
		})
	}
}

// The start help says what start does with a headless director that is
// attached (D6), and that a promoted session still has to be detached.
func TestOrgDirectorStart_HelpStatesTheCleanup(t *testing.T) {
	out, err := runOrgCmd(t, "director", "start", "--help")
	if err != nil {
		t.Fatal(err)
	}
	help := strings.Join(strings.Fields(out), " ")
	for _, want := range []string{
		"give an absolute path), while a promoted session is attached (detach it; for a headless director, see below), and inside a herdr pane",
		"While a headless director that start launched is the attached one, starting or running, start checks it on the spot, before the watch, " +
			"as `ralph org watch --all` checks it every cycle",
		"When the check finds it alive, or cannot tell, start is refused, writes nothing, and says what the check found",
		"When the check finds it ended, start cleans it up as that stop would, but only what the registration recorded when the check read it",
		"It then prints one line, cleaned up d1, which had ended (...), and goes on, so the line is printed even when start is refused afterwards.",
		"start records nothing, starts no director, and exits 1 with the stop command that finishes the cleanup.",
	} {
		if !strings.Contains(help, want) {
			t.Errorf("director start --help does not say %q:\n%s", want, out)
		}
	}
	if strings.Contains(help, "one still starting included") {
		t.Errorf("director start --help still says a starting director is always refused:\n%s", out)
	}
}

// writeDirectorAliveMark writes mark as the watch's director-alive.json in
// stateDir.
func writeDirectorAliveMark(t *testing.T, stateDir string, mark org.DirectorAlive) {
	t.Helper()
	raw, err := json.Marshal(mark)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, org.DirectorAlivePathIn(stateDir), string(raw))
}

// The attach refusal of a herdr registration as before (the way out), for
// d1 on stateDir.
func attachRefusalOfHerdr(stateDir string) string {
	suffix := " --state-dir '" + stateDir + "'"
	return "stop it with ralph org director stop --id d1" + suffix + " from a terminal outside herdr, which closes its herdr pane, " +
		"or replace it with ralph org director attach --driver claude --replace" + suffix + ", which leaves that pane open until such a stop"
}

// attachStartHint is the hint that start cleans d1 up, on stateDir.
func attachStartHint(stateDir string) string {
	return "; as it has ended, ralph org director start --driver <claude|codex> --state-dir '" + stateDir +
		"' cleans it up and then starts a new headless director (it checks d1 again first)"
}

// D7 and the attach half of AC7: attach refused by a running headless d1
// ends with the watch's mark when the mark is d1's (the same id and
// attached_at), escaped, and, when the mark says ended, with the hint that
// start cleans d1 up. A mark of another id or attached_at, or one that does
// not parse, adds nothing. attach writes nothing and runs no herdr (none is
// on PATH that it could run).
func TestOrgDirectorAttach_RefusalAddsTheWatchsMark(t *testing.T) {
	for _, tc := range []struct {
		name string
		mark func(reg org.DirectorRegistration) *org.DirectorAlive
		raw  string // written as the mark when set
		want func(stateDir string) string
	}{
		{"alive", func(reg org.DirectorRegistration) *org.DirectorAlive {
			return &org.DirectorAlive{ID: "d1", AttachedAt: reg.AttachedAt, State: org.DirectorCheckAlive, How: org.DirectorCheckHowAgentFound, CheckedAt: "2026-10-18T09:30:00Z"}
		}, "", func(string) string { return "; the watch checked d1 at 2026-10-18T09:30:00Z: alive (agent_found)" }},
		{"ended", func(reg org.DirectorRegistration) *org.DirectorAlive {
			return &org.DirectorAlive{ID: "d1", AttachedAt: reg.AttachedAt, State: org.DirectorCheckEnded, How: org.DirectorCheckHowAgentEnded, CheckedAt: "2026-10-18T09:30:00Z"}
		}, "", func(stateDir string) string {
			return "; the watch checked d1 at 2026-10-18T09:30:00Z: ended (agent_ended)" + attachStartHint(stateDir)
		}},
		{"unknown, escaped", func(reg org.DirectorRegistration) *org.DirectorAlive {
			return &org.DirectorAlive{ID: "d1", AttachedAt: reg.AttachedAt, State: org.DirectorCheckUnknown, How: org.DirectorCheckHowAgentGetFailed,
				CheckedAt: "2026-10-18T09:30:00Z\r", Detail: "herdr: \x1b[31mred"}
		}, "", func(string) string {
			return `; the watch checked d1 at 2026-10-18T09:30:00Z\r: unknown (agent_get_failed: herdr: \x1b[31mred)`
		}},
		{"another id", func(reg org.DirectorRegistration) *org.DirectorAlive {
			return &org.DirectorAlive{ID: "d7", AttachedAt: reg.AttachedAt, State: org.DirectorCheckEnded, How: org.DirectorCheckHowAgentEnded, CheckedAt: "2026-10-18T09:30:00Z"}
		}, "", func(string) string { return "" }},
		{"another attached_at", func(org.DirectorRegistration) *org.DirectorAlive {
			return &org.DirectorAlive{ID: "d1", AttachedAt: "2026-01-01T00:00:00Z", State: org.DirectorCheckEnded, How: org.DirectorCheckHowAgentEnded, CheckedAt: "2026-10-18T09:30:00Z"}
		}, "", func(string) string { return "" }},
		{"a mark that does not parse", nil, "{not json", func(string) string { return "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := t.TempDir()
			reg := seedHeadless(t, stateDir, "codex", "gpt-6-astra", "ws-1", "pane-1", true)
			if tc.mark != nil {
				writeDirectorAliveMark(t, stateDir, *tc.mark(reg))
			} else {
				writeTestFile(t, org.DirectorAlivePathIn(stateDir), tc.raw)
			}
			before := directorFileOrAbsent(t, stateDir)

			stdout, _, err := runOrgCmdStreams(t, "director", "attach", "--driver", "claude", "--state-dir", stateDir)
			if !errors.Is(err, org.ErrDirectorAttached) {
				t.Fatalf("attach over a running headless d1 = %v; want ErrDirectorAttached", err)
			}
			if want := ", headless running): " + attachRefusalOfHerdr(stateDir) + tc.want(stateDir); !strings.HasSuffix(err.Error(), want) {
				t.Errorf("attach error\n%q\ndoes not end with\n%q", err, want)
			}
			if stdout != "" {
				t.Errorf("stdout = %q; want nothing", stdout)
			}
			if after := directorFileOrAbsent(t, stateDir); after != before {
				t.Errorf("a refused attach changed director.jsonl")
			}
		})
	}
}

// D7: attach refused by a headless d1 still starting with its start process
// recorded reads that process with ps (this test binary, alive with its
// start time, or a child it ran and waited for, gone) and adds what it found,
// with the start hint once it is gone; with the watch's mark of d1 too, both
// are added and the hint once. A launch cmd's refusal gets them the same
// way. Without a recorded start process nothing is read or added
// (TestOrgDirectorAttach_RefusedByAHeadlessDirectorNamesStop pins that).
func TestOrgDirectorAttach_RefusalAddsTheStartersCheck(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	self, err := org.ReadDirectorProcess(ctx, os.Getpid())
	if err != nil || self.State != org.DirectorProcessAlive || self.StartTime == "" {
		t.Fatalf("read this test's own process: %+v, %v", self, err)
	}
	dead := deadChildPID(t)
	for _, tc := range []struct {
		name      string
		launch    string
		pid       int
		startTime string
		mark      bool
		want      func(stateDir string) string
	}{
		{"its start runs", org.DirectorLaunchHerdr, os.Getpid(), self.StartTime, false, func(string) string {
			return fmt.Sprintf("; its start (pid %d), checked just now: alive (start_running)", os.Getpid())
		}},
		{"its start ended", org.DirectorLaunchHerdr, dead, "Thu Oct 15 21:51:09 2026", false, func(stateDir string) string {
			return fmt.Sprintf("; its start (pid %d), checked just now: ended (start_ended)", dead) + attachStartHint(stateDir)
		}},
		{"its start ended, with the watch's mark", org.DirectorLaunchHerdr, dead, "Thu Oct 15 21:51:09 2026", true, func(stateDir string) string {
			return "; the watch checked d1 at 2026-10-18T09:30:00Z: ended (start_ended)" +
				fmt.Sprintf("; its start (pid %d), checked just now: ended (start_ended)", dead) + attachStartHint(stateDir)
		}},
		{"a launch cmd whose start ended", org.DirectorLaunchCmd, dead, "Thu Oct 15 21:51:09 2026", false, func(stateDir string) string {
			return fmt.Sprintf("; its start (pid %d), checked just now: ended (start_ended)", dead) + attachStartHint(stateDir)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stateDir := t.TempDir()
			reg, err := org.NewDirectorStore(stateDir).AttachHeadless(org.DirectorHeadlessAttach{
				Launch: tc.launch, Driver: "codex", Model: "gpt-6-astra", PermissionMode: org.PermissionModeGuarded, Wake: []string{org.DirectorWakePull},
				StarterPID: tc.pid, StarterStartTime: tc.startTime, User: "alice", Cwd: "/work",
			})
			if err != nil {
				t.Fatalf("seed d1: %v", err)
			}
			if tc.mark {
				writeDirectorAliveMark(t, stateDir, org.DirectorAlive{ID: "d1", AttachedAt: reg.AttachedAt, State: org.DirectorCheckEnded,
					How: org.DirectorCheckHowStartEnded, CheckedAt: "2026-10-18T09:30:00Z"})
			}
			before := directorFileOrAbsent(t, stateDir)

			_, _, err = runOrgCmdStreams(t, "director", "attach", "--driver", "claude", "--state-dir", stateDir)
			if !errors.Is(err, org.ErrDirectorAttached) {
				t.Fatalf("attach over a starting headless d1 = %v; want ErrDirectorAttached", err)
			}
			wayOut := "which leaves that pane open until such a stop"
			if tc.launch == org.DirectorLaunchCmd {
				wayOut = "which leaves that process running until such a stop"
			}
			if want := wayOut + tc.want(stateDir); !strings.HasSuffix(err.Error(), want) {
				t.Errorf("attach error\n%q\ndoes not end with\n%q", err, want)
			}
			if after := directorFileOrAbsent(t, stateDir); after != before {
				t.Errorf("a refused attach changed director.jsonl")
			}
		})
	}
}

// The attach help says what the refusal of a headless director adds (D7).
func TestOrgDirectorAttach_HelpStatesTheChecksInTheRefusal(t *testing.T) {
	out, err := runOrgCmd(t, "director", "attach", "--help")
	if err != nil {
		t.Fatal(err)
	}
	help := strings.Join(strings.Fields(out), " ")
	if want := "The refusal then also says what `ralph org watch --all` found when it last checked that director, and, for one still starting, " +
		"what ps reads now of the start process that recorded it; when either says it has ended, the refusal names `ralph org director start`, " +
		"which cleans it up before it starts a new headless director. attach itself runs no herdr and cleans up nothing."; !strings.Contains(help, want) {
		t.Errorf("director attach --help does not say %q:\n%s", want, out)
	}
}

// D7: a promoted session's refusal is as before, even with a mark of its id
// and attached_at on disk.
func TestOrgDirectorAttach_APromotedSessionsRefusalIsUnchanged(t *testing.T) {
	stateDir := t.TempDir()
	setEscalateWatch(t, acceptingWatchLiveness())
	runInboxOK(t, "director", "attach", "--driver", "claude", "--state-dir", stateDir)
	cur := mustCurrentDirector(t, stateDir)
	writeDirectorAliveMark(t, stateDir, org.DirectorAlive{ID: cur.ID, AttachedAt: cur.AttachedAt, State: org.DirectorCheckEnded,
		How: org.DirectorCheckHowAgentEnded, CheckedAt: "2026-10-18T09:30:00Z"})

	_, _, err := runOrgCmdStreams(t, "director", "attach", "--driver", "claude", "--state-dir", stateDir)
	if !errors.Is(err, org.ErrDirectorAttached) {
		t.Fatalf("attach over a promoted session = %v; want ErrDirectorAttached", err)
	}
	suffix := " --state-dir '" + stateDir + "'"
	if want := "): end it with ralph org director detach --id d1" + suffix + " from the session that attached it " +
		"(or with --force in place of --id, when that session is gone), or replace it with ralph org director attach --driver claude --replace" + suffix; !strings.HasSuffix(err.Error(), want) {
		t.Errorf("attach error\n%q\ndoes not end with\n%q", err, want)
	}
}

// mustCurrentDirector is the current registration of stateDir.
func mustCurrentDirector(t *testing.T, stateDir string) org.DirectorRegistration {
	t.Helper()
	res, err := org.NewDirectorStore(stateDir).Read()
	if err != nil || res.Current == nil {
		t.Fatalf("read the current registration: %+v, %v", res.Current, err)
	}
	return *res.Current
}
