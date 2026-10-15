package org

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Tests for director_cmd.go (plan
// docs/plans/active/2026-10-16-org-director-launch-cmd.md, D1, D3, and D7):
// the process reader, the stop of a recorded process, the launch script, and
// the launch_cmd argv and its runner. Every process they signal is one the
// test started itself, and every process a test starts is killed and waited
// for in its cleanup; none runs herdr, a terminal, or a tmux of the user's.

// testProcess is a process a test started (startTestProcess).
type testProcess struct {
	cmd     *exec.Cmd
	waited  bool
	waitErr error
}

// startTestProcess starts name with args and kills (SIGKILL) and waits for
// it in t's cleanup unless the test already waited for it. Until it is
// waited for, its pid stays its own, a zombie once it exits, so the kill can
// reach no other process.
func startTestProcess(t *testing.T, name string, args ...string) *testProcess {
	t.Helper()
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start %s %q: %v", name, args, err)
	}
	p := &testProcess{cmd: cmd}
	t.Cleanup(func() {
		if !p.waited {
			_ = cmd.Process.Kill()
			_ = p.wait()
		}
	})
	return p
}

func (p *testProcess) pid() int { return p.cmd.Process.Pid }

func (p *testProcess) wait() error {
	if !p.waited {
		p.waitErr = p.cmd.Wait()
		p.waited = true
	}
	return p.waitErr
}

// signalledBy reports whether p, once waited for, was ended by sig.
func (p *testProcess) signalledBy(t *testing.T, sig syscall.Signal) bool {
	t.Helper()
	_ = p.wait()
	ws, ok := p.cmd.ProcessState.Sys().(syscall.WaitStatus)
	return ok && ws.Signaled() && ws.Signal() == sig
}

// mustReadProcess is ReadDirectorProcess of pid under a 10s deadline.
func mustReadProcess(t *testing.T, pid int) DirectorProcess {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	proc, err := ReadDirectorProcess(ctx, pid)
	if err != nil {
		t.Fatalf("ReadDirectorProcess(%d): %v", pid, err)
	}
	return proc
}

// waitForProcessState reads pid until its state is want, for up to 5s.
func waitForProcessState(t *testing.T, pid int, want string) DirectorProcess {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		proc := mustReadProcess(t, pid)
		if proc.State == want {
			return proc
		}
		if time.Now().After(deadline) {
			t.Fatalf("pid %d is %+v after 5s, want %s", pid, proc, want)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// sleepPath is the sleep the tests start, found before a test changes PATH.
func sleepPath(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("sleep")
	if err != nil {
		t.Fatalf("look up sleep: %v", err)
	}
	return path
}

// writeTestExecutable writes content to path, mode 0755.
func writeTestExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

// AC7's identity check reads a process by its pid: a running process reads
// alive with the same start time every time, in ps's lstart format and in
// UTC (the time the test started it, to the second), whatever the caller's
// TZ and locale say.
func TestReadDirectorProcess_AnAliveProcessKeepsItsStartTime(t *testing.T) {
	t.Setenv("TZ", "Asia/Tokyo")
	t.Setenv("LC_ALL", "ja_JP.UTF-8")
	t.Setenv("PS_FORMAT", "pid")
	before := time.Now().UTC().Truncate(time.Second)
	p := startTestProcess(t, sleepPath(t), "30")
	after := time.Now().UTC()

	first := mustReadProcess(t, p.pid())
	if first.State != DirectorProcessAlive || first.StartTime == "" {
		t.Fatalf("ReadDirectorProcess = %+v, want alive with a start time", first)
	}
	for range 3 {
		if again := mustReadProcess(t, p.pid()); again != first {
			t.Fatalf("a second read = %+v, want %+v again", again, first)
		}
	}
	started, err := time.Parse(directorPSStartTimeLayout, first.StartTime)
	if err != nil {
		t.Fatalf("start time %q is not ps's lstart: %v", first.StartTime, err)
	}
	if started.Before(before.Add(-2*time.Second)) || started.After(after.Add(2*time.Second)) {
		t.Fatalf("start time %s (read as UTC) is not between %s and %s: ps did not print it in UTC", started, before, after)
	}
	if strings.Contains(first.StartTime, "  ") || strings.TrimSpace(first.StartTime) != first.StartTime {
		t.Fatalf("start time %q keeps a run of spaces or an edge space", first.StartTime)
	}
}

// ps gets only PATH, LC_ALL=C, and TZ=UTC: the caller's locale, time zone,
// and ps variables do not reach it.
func TestReadDirectorProcess_RunsPSInCAndUTC(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	if got, want := directorPSEnv(), []string{"PATH=/usr/bin:/bin", "LC_ALL=C", "TZ=UTC"}; !slices.Equal(got, want) {
		t.Fatalf("directorPSEnv() = %q, want %q", got, want)
	}

	dir := t.TempDir()
	envOut := filepath.Join(dir, "env")
	writeTestExecutable(t, filepath.Join(dir, "ps"), "#!/bin/sh\nenv | sort > "+shellQuote(envOut)+"\necho 'S    Thu Oct 15 21:51:09 2026'\n")
	t.Setenv("PATH", dir+":/usr/bin:/bin")
	t.Setenv("TZ", "Asia/Tokyo")
	t.Setenv("PS_PERSONALITY", "bsd")
	proc := mustReadProcess(t, 4242)
	if proc != (DirectorProcess{State: DirectorProcessAlive, StartTime: "Thu Oct 15 21:51:09 2026"}) {
		t.Fatalf("ReadDirectorProcess = %+v, want alive with the stub's start time", proc)
	}
	raw, err := os.ReadFile(envOut)
	if err != nil {
		t.Fatal(err)
	}
	// sh may add PWD, SHLVL, or _ of its own; none of the caller's.
	var env []string
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if name, _, _ := strings.Cut(line, "="); name != "PWD" && name != "SHLVL" && name != "_" && name != "OLDPWD" {
			env = append(env, line)
		}
	}
	if want := []string{"LC_ALL=C", "PATH=" + dir + ":/usr/bin:/bin", "TZ=UTC"}; !slices.Equal(env, want) {
		t.Fatalf("ps ran with %q, want %q", env, want)
	}
}

// A pid with no process reads gone; a process that exited and was not
// reaped yet reads zombie; both mean the recorded process has ended (D7).
func TestReadDirectorProcess_GoneAndZombie(t *testing.T) {
	t.Run("gone", func(t *testing.T) {
		p := startTestProcess(t, "/bin/sh", "-c", "exit 0")
		if err := p.wait(); err != nil {
			t.Fatalf("wait: %v", err)
		}
		if proc := mustReadProcess(t, p.pid()); proc != (DirectorProcess{State: DirectorProcessGone}) {
			t.Fatalf("ReadDirectorProcess of a reaped pid = %+v, want gone", proc)
		}
	})
	t.Run("zombie", func(t *testing.T) {
		p := startTestProcess(t, "/bin/sh", "-c", "exit 0")
		waitForProcessState(t, p.pid(), DirectorProcessZombie)
	})
}

// Every read ps does not answer with a pid's stat and start time is an
// error, which the stop treats as unreadable: a pid that is not positive, no
// ps, ps failing (exit 1 with stderr is not gone), ps printing something
// else, and ps not answering in time.
func TestReadDirectorProcess_Errors(t *testing.T) {
	sleep := sleepPath(t)
	for _, tc := range []struct {
		name string
		ps   string // the stub ps; "" for no ps on PATH
		pid  int
		want string
	}{
		{"pid 0", "", 0, "read process 0: the pid is not a positive number"},
		{"a negative pid", "", -3, "read process -3: the pid is not a positive number"},
		{"no ps", "", 42, `read process 42: ps: exec: "ps": executable file not found in $PATH`},
		{"exit 1 with stderr", "echo 'ps: process id too large: 42' >&2\nexit 1", 42, "read process 42: ps: exit status 1: ps: process id too large: 42"},
		{"exit 2", "exit 2", 42, "read process 42: ps: exit status 2"},
		{"exit 0 and nothing printed", "exit 0", 42, `read process 42: ps printed "", not one line with its stat and start time`},
		{"two lines", "echo 'S Thu Oct 15 21:51:09 2026'\necho 'S Thu Oct 15 21:51:09 2026'", 42, "not one line with its stat and start time"},
		{"no start time", "echo 'S'", 42, `ps printed the start time "", which is not of the form`},
		{"another format", "echo 'S 2026-10-15T21:51:09Z'", 42, `ps printed the start time "2026-10-15T21:51:09Z", which is not of the form "Mon Jan _2 15:04:05 2006"`},
		{"a control character is escaped", "printf 'S\\033[2J\\n' >&2\nexit 3", 42, `ps: exit status 3: S\x1b[2J`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.ps != "" {
				writeTestExecutable(t, filepath.Join(dir, "ps"), "#!/bin/sh\n"+tc.ps+"\n")
			}
			t.Setenv("PATH", dir)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			proc, err := ReadDirectorProcess(ctx, tc.pid)
			if err == nil || !strings.Contains(err.Error(), tc.want) || proc != (DirectorProcess{}) {
				t.Fatalf("ReadDirectorProcess(%d) = %+v, %v; want an error with %q", tc.pid, proc, err, tc.want)
			}
		})
	}
	t.Run("exit 1 with no output is gone", func(t *testing.T) {
		dir := t.TempDir()
		writeTestExecutable(t, filepath.Join(dir, "ps"), "#!/bin/sh\nexit 1\n")
		t.Setenv("PATH", dir)
		if proc := mustReadProcess(t, 42); proc != (DirectorProcess{State: DirectorProcessGone}) {
			t.Fatalf("ReadDirectorProcess = %+v, want gone", proc)
		}
	})
	t.Run("ps that does not answer", func(t *testing.T) {
		dir := t.TempDir()
		writeTestExecutable(t, filepath.Join(dir, "ps"), "#!/bin/sh\nexec "+shellQuote(sleep)+" 30\n")
		t.Setenv("PATH", dir)
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		start := time.Now()
		_, err := ReadDirectorProcess(ctx, 42)
		if err == nil || !strings.Contains(err.Error(), "read process 42: ps did not answer: context deadline exceeded") {
			t.Fatalf("ReadDirectorProcess = %v, want the deadline", err)
		}
		if took := time.Since(start); took > 5*time.Second {
			t.Fatalf("ReadDirectorProcess took %s past its 200ms deadline", took)
		}
	})
}

// cmdRegistrationOf is a launch cmd registration whose check-in recorded p's
// pid and the start time ps reads for it now.
func cmdRegistrationOf(t *testing.T, p *testProcess) DirectorRegistration {
	t.Helper()
	proc := mustReadProcess(t, p.pid())
	if proc.State != DirectorProcessAlive {
		t.Fatalf("pid %d is %+v, want alive", p.pid(), proc)
	}
	return DirectorRegistration{ID: "d1", Launch: DirectorLaunchCmd, State: DirectorStateRunning, Token: directorTestToken, PID: p.pid(), StartTime: proc.StartTime}
}

// AC6: the recorded process, alive with its recorded start time, gets one
// SIGTERM and the stop waits until it has ended.
func TestStopDirectorProcess_SIGTERMEndsThisLaunchsProcess(t *testing.T) {
	p := startTestProcess(t, sleepPath(t), "30")
	o := &Org{}
	got := o.stopDirectorProcess(cmdRegistrationOf(t, p))
	if got != (directorProcessStop{outcome: directorProcessStopEnded, signalled: true, note: "process=ended after SIGTERM"}) {
		t.Fatalf("stopDirectorProcess = %+v, want ended after SIGTERM", got)
	}
	if !p.signalledBy(t, syscall.SIGTERM) {
		t.Fatalf("the process ended with %v, want SIGTERM", p.cmd.ProcessState)
	}
}

// AC7: a process that ignores SIGTERM is reported not ended once the wait
// runs out, and gets no SIGKILL: it is still running, with its start time,
// after the stop returned.
func TestStopDirectorProcess_AProcessThatIgnoresSIGTERMIsNotEndedInTime(t *testing.T) {
	// An ignored signal stays ignored across exec, so sleep ignores TERM and
	// has no child of its own.
	p := startTestProcess(t, "/bin/sh", "-c", "trap '' TERM; exec "+shellQuote(sleepPath(t))+" 30")
	time.Sleep(100 * time.Millisecond) // let sh reach the exec
	reg := cmdRegistrationOf(t, p)
	o := &Org{DirectorStopWait: 300 * time.Millisecond}
	start := time.Now()
	got := o.stopDirectorProcess(reg)
	if got.outcome != directorProcessStopNotEnded || !got.signalled || got.note != "process=still running after SIGTERM" ||
		got.err == nil || got.err.Error() != fmt.Sprintf("pid %d did not end within 300ms of its SIGTERM", reg.PID) {
		t.Fatalf("stopDirectorProcess = %+v, want not ended within 300ms", got)
	}
	if took := time.Since(start); took < 300*time.Millisecond || took > 5*time.Second {
		t.Fatalf("the stop waited %s, want about 300ms", took)
	}
	if proc := mustReadProcess(t, p.pid()); proc.State != DirectorProcessAlive || proc.StartTime != reg.StartTime {
		t.Fatalf("after the stop the process is %+v, want it still running (no SIGKILL)", proc)
	}
}

// AC6 and AC7: a pid whose process has another start time (a later process
// got the pid), a pid no process has, and a zombie all mean the recorded
// process ended, so nothing is sent; the process with another start time,
// which SIGTERM would end, is still running afterwards.
func TestStopDirectorProcess_AnEndedProcessIsSentNothing(t *testing.T) {
	t.Run("another start time", func(t *testing.T) {
		p := startTestProcess(t, sleepPath(t), "30")
		reg := cmdRegistrationOf(t, p)
		now := reg.StartTime
		reg.StartTime = "Thu Jan 1 00:00:00 1970"
		got := (&Org{}).stopDirectorProcess(reg)
		want := fmt.Sprintf("process=already ended: pid %d is another process now (started %q, not %q)", reg.PID, now, reg.StartTime)
		if got != (directorProcessStop{outcome: directorProcessStopEnded, note: want}) {
			t.Fatalf("stopDirectorProcess = %+v, want ended with %q and nothing sent", got, want)
		}
		time.Sleep(200 * time.Millisecond)
		if proc := mustReadProcess(t, p.pid()); proc.State != DirectorProcessAlive || proc.StartTime != now {
			t.Fatalf("the other process is %+v, want it still running", proc)
		}
	})
	t.Run("gone", func(t *testing.T) {
		p := startTestProcess(t, "/bin/sh", "-c", "exit 0")
		if err := p.wait(); err != nil {
			t.Fatal(err)
		}
		reg := DirectorRegistration{ID: "d1", Launch: DirectorLaunchCmd, PID: p.pid(), StartTime: "Thu Jan 1 00:00:00 1970"}
		want := fmt.Sprintf("process=already ended: no process has pid %d", reg.PID)
		if got := (&Org{}).stopDirectorProcess(reg); got != (directorProcessStop{outcome: directorProcessStopEnded, note: want}) {
			t.Fatalf("stopDirectorProcess = %+v, want ended, nothing sent", got)
		}
	})
	t.Run("zombie", func(t *testing.T) {
		p := startTestProcess(t, "/bin/sh", "-c", "exit 0")
		proc := waitForProcessState(t, p.pid(), DirectorProcessZombie)
		reg := DirectorRegistration{ID: "d1", Launch: DirectorLaunchCmd, PID: p.pid(), StartTime: proc.StartTime}
		want := fmt.Sprintf("process=already ended: pid %d is a zombie", reg.PID)
		if got := (&Org{}).stopDirectorProcess(reg); got != (directorProcessStop{outcome: directorProcessStopEnded, note: want}) {
			t.Fatalf("stopDirectorProcess = %+v, want ended, nothing sent", got)
		}
		if p.signalledBy(t, syscall.SIGTERM) {
			t.Fatal("the zombie's exit status says SIGTERM")
		}
	})
	t.Run("no pid recorded", func(t *testing.T) {
		o := &Org{DirectorProcessReader: func(context.Context, int) (DirectorProcess, error) {
			t.Fatal("a registration with no pid was read")
			return DirectorProcess{}, nil
		}}
		reg := DirectorRegistration{ID: "d1", Launch: DirectorLaunchCmd, State: DirectorStateStarting, Token: directorTestToken}
		if got := o.stopDirectorProcess(reg); got != (directorProcessStop{outcome: directorProcessStopEnded, note: "process=none recorded"}) {
			t.Fatalf("stopDirectorProcess = %+v, want ended with none recorded", got)
		}
	})
}

// AC7: when ps cannot be read, nothing is sent and the stop is unreadable;
// when it cannot be read after the SIGTERM, the stop is unreadable too.
func TestStopDirectorProcess_APSThatCannotBeReadSendsNothing(t *testing.T) {
	psErr := errors.New("ps: not here")
	t.Run("before the SIGTERM", func(t *testing.T) {
		p := startTestProcess(t, sleepPath(t), "30")
		reg := cmdRegistrationOf(t, p)
		var reads []int
		o := &Org{DirectorProcessReader: func(_ context.Context, pid int) (DirectorProcess, error) {
			reads = append(reads, pid)
			return DirectorProcess{}, psErr
		}}
		got := o.stopDirectorProcess(reg)
		if got.outcome != directorProcessStopUnreadable || got.signalled || !errors.Is(got.err, psErr) || got.note != "process=not signalled: ps could not be read" {
			t.Fatalf("stopDirectorProcess = %+v, want unreadable with nothing sent", got)
		}
		if !slices.Equal(reads, []int{reg.PID}) {
			t.Fatalf("reads = %v, want one of pid %d", reads, reg.PID)
		}
		time.Sleep(200 * time.Millisecond)
		if proc := mustReadProcess(t, p.pid()); proc.State != DirectorProcessAlive {
			t.Fatalf("the process is %+v, want it still running", proc)
		}
	})
	t.Run("after the SIGTERM", func(t *testing.T) {
		p := startTestProcess(t, sleepPath(t), "30")
		reg := cmdRegistrationOf(t, p)
		reads := 0
		o := &Org{DirectorProcessReader: func(ctx context.Context, pid int) (DirectorProcess, error) {
			if reads++; reads == 1 {
				return ReadDirectorProcess(ctx, pid)
			}
			return DirectorProcess{}, psErr
		}}
		got := o.stopDirectorProcess(reg)
		if got.outcome != directorProcessStopUnreadable || !got.signalled || !errors.Is(got.err, psErr) || got.note != "process=sent SIGTERM; ps could not be read after it" {
			t.Fatalf("stopDirectorProcess = %+v, want unreadable after the SIGTERM", got)
		}
		if !p.signalledBy(t, syscall.SIGTERM) {
			t.Fatalf("the process ended with %v, want SIGTERM", p.cmd.ProcessState)
		}
	})
}

// testLaunchScript is a directorLaunchScript whose values hold spaces,
// single quotes, and shell syntax, under base.
func testLaunchScript(base string) directorLaunchScript {
	return directorLaunchScript{
		ID: "d7", Token: directorTestToken,
		StateDir:      filepath.Join(base, "st'ate dir"),
		Cwd:           filepath.Join(base, "it's $HOME"),
		AgmsgHome:     filepath.Join(base, "a g'm"),
		RalphPath:     filepath.Join(base, "bin dir's", "ralph"),
		AgentPath:     filepath.Join(base, "bin dir's", "cl'aude"),
		PermArgs:      []string{"--permission-mode", "bypassPermissions"},
		Model:         "op$us `x`",
		PromptPointer: PromptFilePointer(filepath.Join(base, "st'ate dir", "prompts", "director_"+directorTestToken+".md")),
	}
}

// AC2: the launch script cds to the cwd, exports RALPH_DIRECTOR_TOKEN and
// RALPH_ORG_AGMSG_HOME (only when given), checks in, and execs the agent
// with the permission flags, --model, and the prompt pointer in that order,
// every value single-quoted.
func TestDirectorLaunchScript_TextQuotesEveryValue(t *testing.T) {
	s := testLaunchScript("/b")
	text, err := s.text()
	if err != nil {
		t.Fatalf("text: %v", err)
	}
	want := `#!/bin/sh
# The launch script of the headless director d7 (director_0123456789ab), written by ralph org director start.
# It checks in with ralph, which records its pid, and only then execs the agent in its place.
cd '/b/it'\''s $HOME' || exit 1
RALPH_DIRECTOR_TOKEN='0123456789ab'
export RALPH_DIRECTOR_TOKEN
RALPH_ORG_AGMSG_HOME='/b/a g'\''m'
export RALPH_ORG_AGMSG_HOME
'/b/bin dir'\''s/ralph' org director launched --id 'd7' --token '0123456789ab' --pid "$$" --state-dir '/b/st'\''ate dir' || exit 1
exec '/b/bin dir'\''s/cl'\''aude' '--permission-mode' 'bypassPermissions' '--model' 'op$us ` + "`x`" + `' '役割指示を読み込んで従ってください: /b/st'\''ate dir/prompts/director_0123456789ab.md'
`
	if text != want {
		t.Fatalf("launch script =\n%s\nwant\n%s", text, want)
	}

	s.AgmsgHome, s.PermArgs = "", nil
	text, err = s.text()
	if err != nil {
		t.Fatalf("text: %v", err)
	}
	if strings.Contains(text, "RALPH_ORG_AGMSG_HOME") {
		t.Fatalf("a script with no agmsg home exports one:\n%s", text)
	}
	if !strings.Contains(text, "\nexec '/b/bin dir'\\''s/cl'\\''aude' '--model' 'op$us `x`' '役割指示") {
		t.Fatalf("with no permission flags the exec line is not the agent, --model, and the pointer:\n%s", text)
	}
}

// launchScriptFakes writes, under dir, a fake ralph that records its
// arguments and its parent pid and exits with checkInExit, and a fake agent
// that records its pid, directory, two variables, and arguments, both in a
// directory whose name holds a space and a single quote. It returns the
// script with those paths and the files they record to.
func launchScriptFakes(t *testing.T, dir string, checkInExit int) (s directorLaunchScript, ralphArgs, ralphPPID, agentOut string) {
	t.Helper()
	s = testLaunchScript(dir)
	ralphArgs, ralphPPID, agentOut = filepath.Join(dir, "ralph.args"), filepath.Join(dir, "ralph.ppid"), filepath.Join(dir, "agent.out")
	writeTestExecutable(t, s.RalphPath, "#!/bin/sh\nfor a in \"$@\"; do printf '%s\\n' \"$a\"; done > "+shellQuote(ralphArgs)+
		"\nprintf '%s\\n' \"$PPID\" > "+shellQuote(ralphPPID)+"\nexit "+strconv.Itoa(checkInExit)+"\n")
	writeTestExecutable(t, s.AgentPath, "#!/bin/sh\n{\nprintf 'pid=%s\\n' \"$$\"\nprintf 'pwd=%s\\n' \"$(pwd -P)\"\n"+
		"printf 'token=%s\\n' \"${RALPH_DIRECTOR_TOKEN-<unset>}\"\nprintf 'agmsg=%s\\n' \"${RALPH_ORG_AGMSG_HOME-<unset>}\"\n"+
		"for a in \"$@\"; do printf 'arg=%s\\n' \"$a\"; done\n} > "+shellQuote(agentOut)+"\n")
	if err := os.MkdirAll(s.Cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	return s, ralphArgs, ralphPPID, agentOut
}

// runLaunchScript runs the script at path as a terminal would, with env as
// its whole environment, and returns its exit code.
func runLaunchScript(t *testing.T, path string, env []string) int {
	t.Helper()
	cmd := exec.Command(path)
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run the launch script: %v (%s)", err, out)
		}
		return exitErr.ExitCode()
	}
	return 0
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
}

// AC2 and AC3 run through sh: the check-in gets the id, the token, the
// script's pid, and the state dir as written; its parent is that pid, which
// the agent then keeps (exec), so the pid the check-in records is the
// agent's. The agent runs in the cwd with the token and the agmsg home
// exported and its arguments as written. Without an agmsg home the agent
// gets none. A check-in that fails starts no agent and the script exits 1.
func TestDirectorLaunchScript_ChecksInThenExecsTheAgent(t *testing.T) {
	env := []string{"PATH=/usr/bin:/bin"}
	t.Run("the check-in passes", func(t *testing.T) {
		dir := t.TempDir()
		s, ralphArgs, ralphPPID, agentOut := launchScriptFakes(t, dir, 0)
		path, err := writeDirectorLaunchScript(s)
		if err != nil {
			t.Fatalf("writeDirectorLaunchScript: %v", err)
		}
		if code := runLaunchScript(t, path, env); code != 0 {
			t.Fatalf("the launch script exited %d, want 0", code)
		}
		args := readLines(t, ralphArgs)
		if len(args) != 11 {
			t.Fatalf("check-in args = %q, want 11", args)
		}
		pid := args[8]
		if n, err := strconv.Atoi(pid); err != nil || n <= 0 {
			t.Fatalf("--pid %q is not a pid", pid)
		}
		if want := []string{"org", "director", "launched", "--id", "d7", "--token", directorTestToken, "--pid", pid, "--state-dir", s.StateDir}; !slices.Equal(args, want) {
			t.Fatalf("check-in args =\n%q\nwant\n%q", args, want)
		}
		if ppid := readLines(t, ralphPPID); len(ppid) != 1 || ppid[0] != pid {
			t.Fatalf("the check-in's parent = %q, want the --pid %s", ppid, pid)
		}
		cwd, err := filepath.EvalSymlinks(s.Cwd)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"pid=" + pid, "pwd=" + cwd, "token=" + directorTestToken, "agmsg=" + s.AgmsgHome,
			"arg=--permission-mode", "arg=bypassPermissions", "arg=--model", "arg=" + s.Model, "arg=" + s.PromptPointer}
		if got := readLines(t, agentOut); !slices.Equal(got, want) {
			t.Fatalf("the agent recorded\n%q\nwant\n%q", got, want)
		}
	})
	t.Run("no agmsg home", func(t *testing.T) {
		dir := t.TempDir()
		s, _, _, agentOut := launchScriptFakes(t, dir, 0)
		s.AgmsgHome = ""
		path, err := writeDirectorLaunchScript(s)
		if err != nil {
			t.Fatal(err)
		}
		if code := runLaunchScript(t, path, env); code != 0 {
			t.Fatalf("the launch script exited %d, want 0", code)
		}
		if got := readLines(t, agentOut)[3]; got != "agmsg=<unset>" {
			t.Fatalf("the agent got %q, want no RALPH_ORG_AGMSG_HOME", got)
		}
	})
	t.Run("the check-in is refused", func(t *testing.T) {
		dir := t.TempDir()
		s, ralphArgs, _, agentOut := launchScriptFakes(t, dir, 1)
		path, err := writeDirectorLaunchScript(s)
		if err != nil {
			t.Fatal(err)
		}
		if code := runLaunchScript(t, path, env); code != 1 {
			t.Fatalf("the launch script exited %d, want 1", code)
		}
		if _, err := os.Stat(ralphArgs); err != nil {
			t.Fatalf("the check-in did not run: %v", err)
		}
		if _, err := os.Stat(agentOut); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("the agent ran after a refused check-in (stat %v)", err)
		}
	})
	t.Run("a cwd that is gone", func(t *testing.T) {
		dir := t.TempDir()
		s, ralphArgs, _, agentOut := launchScriptFakes(t, dir, 0)
		s.Cwd = filepath.Join(dir, "missing")
		path, err := writeDirectorLaunchScript(s)
		if err != nil {
			t.Fatal(err)
		}
		if code := runLaunchScript(t, path, env); code != 1 {
			t.Fatalf("the launch script exited %d, want 1", code)
		}
		for _, f := range []string{ralphArgs, agentOut} {
			if _, err := os.Stat(f); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("%s exists after the cd failed (stat %v)", f, err)
			}
		}
	})
}

// AC2: the script is <state dir>/launch/director_<token>.sh, mode 0700, in a
// directory of mode 0700 (one that existed with another mode is set to it),
// and writing it again replaces it and leaves no temporary file.
func TestDirectorLaunchScript_PathAndModes(t *testing.T) {
	s := testLaunchScript(t.TempDir())
	launchDir := filepath.Join(s.StateDir, "launch")
	if err := os.MkdirAll(launchDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := writeDirectorLaunchScript(s)
	if err != nil {
		t.Fatalf("writeDirectorLaunchScript: %v", err)
	}
	if want := filepath.Join(launchDir, "director_"+directorTestToken+".sh"); path != want || DirectorLaunchScriptPath(s.StateDir, s.Token) != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	for _, p := range []string{path, launchDir} {
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0o700 {
			t.Fatalf("%s has mode %v, want 0700", p, fi.Mode().Perm())
		}
	}
	s.Model = "sonnet"
	if _, err := writeDirectorLaunchScript(s); err != nil {
		t.Fatalf("second write: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "'--model' 'sonnet'") {
		t.Fatalf("the second write did not replace the script:\n%s", raw)
	}
	if got := dirEntryNames(t, launchDir); !slices.Equal(got, []string{"director_" + directorTestToken + ".sh"}) {
		t.Fatalf("launch dir holds %v, want the script only", got)
	}
}

func TestDirectorLaunchScript_RefusesValuesItCannotWrite(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(s *directorLaunchScript)
		want   string
	}{
		{"an id not of the form d<N>", func(s *directorLaunchScript) { s.ID = "d01" }, `"d01" is not a registration id`},
		{"a malformed token", func(s *directorLaunchScript) { s.Token = "x'; rm -rf ~" }, "is not 12 lowercase hex digits"},
		{"a relative state dir", func(s *directorLaunchScript) { s.StateDir = "state" }, `the state dir path "state" is not absolute`},
		{"a relative cwd", func(s *directorLaunchScript) { s.Cwd = "." }, `the cwd path "." is not absolute`},
		{"a relative ralph", func(s *directorLaunchScript) { s.RalphPath = "ralph" }, `the ralph executable path "ralph" is not absolute`},
		{"a relative agent", func(s *directorLaunchScript) { s.AgentPath = "claude" }, `the agent path "claude" is not absolute`},
		{"no model", func(s *directorLaunchScript) { s.Model = "" }, "the model is empty"},
		{"no prompt pointer", func(s *directorLaunchScript) { s.PromptPointer = "" }, "the prompt pointer is empty"},
		{"a NUL byte", func(s *directorLaunchScript) { s.PermArgs = []string{"a\x00b"} }, "holds a NUL byte"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := testLaunchScript(t.TempDir())
			tc.mutate(&s)
			path, err := writeDirectorLaunchScript(s)
			if err == nil || !strings.HasPrefix(err.Error(), "launch script: ") || !strings.Contains(err.Error(), tc.want) || path != "" {
				t.Fatalf("writeDirectorLaunchScript = %q, %v; want an error with %q", path, err, tc.want)
			}
			if _, err := os.Stat(filepath.Join(s.StateDir, "launch")); !errors.Is(err, os.ErrNotExist) && filepath.IsAbs(s.StateDir) {
				t.Fatalf("a refused script made the launch dir (stat %v)", err)
			}
		})
	}
}

// D1: every element of launch_cmd gets its placeholders replaced; {script_sh}
// is the script path single-quoted, which a shell reads back as the path.
func TestDirectorLaunchArgv_ReplacesThePlaceholders(t *testing.T) {
	script := "/s/it's a/director_" + directorTestToken + ".sh"
	cwd := "/w/my repo"
	argv, err := directorLaunchArgv([]string{
		"tmux", "-L", "ralph", "new-session", "-d", "-s", "{name}", "-c", "{cwd}", "sh {script_sh}", "x{script}y", "}{", "{cwd}{name}",
	}, script, directorTestToken, cwd)
	if err != nil {
		t.Fatalf("directorLaunchArgv: %v", err)
	}
	want := []string{
		"tmux", "-L", "ralph", "new-session", "-d", "-s", "director_" + directorTestToken, "-c", cwd,
		`sh '/s/it'\''s a/director_` + directorTestToken + `.sh'`, "x" + script + "y", "}{", cwd + "director_" + directorTestToken,
	}
	if !slices.Equal(argv, want) {
		t.Fatalf("argv =\n%q\nwant\n%q", argv, want)
	}

	// The shell that reads {script_sh} gets the path back, quote and space
	// included.
	dir := t.TempDir()
	argv, err = directorLaunchArgv([]string{"/bin/sh", "-c", "printf '%s' {script_sh} > out"}, script, directorTestToken, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := runDirectorLaunchCmd(context.Background(), argv, dir, 0); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "out")); err != nil || string(got) != script {
		t.Fatalf("the shell read {script_sh} as %q (%v), want %q", got, err, script)
	}
}

func TestDirectorLaunchArgv_RefusesWhatLoadWouldRefuse(t *testing.T) {
	for _, tc := range []struct {
		name string
		cmd  []string
		want string
	}{
		{"empty", nil, "[org.director].launch_cmd is empty"},
		{"no script", []string{"tmux", "new-session", "{name}"}, "no element has {script} or {script_sh}"},
		{"an unknown placeholder", []string{"sh", "{script}", "{home}"}, "has the unknown placeholder {home}"},
		{"a shell brace group", []string{"sh", "-c", "{ a; }; sh {script_sh}"}, "has the unknown placeholder { a; }"},
		{"a blank command", []string{" ", "{script}"}, "the first element is the command to run"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			argv, err := directorLaunchArgv(tc.cmd, "/s/x.sh", directorTestToken, "/w")
			if err == nil || !strings.Contains(err.Error(), tc.want) || argv != nil {
				t.Fatalf("directorLaunchArgv(%q) = %q, %v; want an error with %q", tc.cmd, argv, err, tc.want)
			}
		})
	}
}

// D1: launch_cmd runs without a shell, in the cwd, with stdin empty: an
// argument that a shell would expand reaches the command as written.
func TestRunDirectorLaunchCmd_RunsWithoutAShellInTheCwd(t *testing.T) {
	dir := t.TempDir()
	argv := []string{"/bin/sh", "-c", `pwd -P > pwd; for a in "$@"; do printf '%s\n' "$a"; done > args; cat > stdin`, "sh", "a b", "$HOME", "*", "`id`"}
	if err := runDirectorLaunchCmd(context.Background(), argv, dir, 0); err != nil {
		t.Fatalf("runDirectorLaunchCmd: %v", err)
	}
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := readLines(t, filepath.Join(dir, "pwd")); !slices.Equal(got, []string{real}) {
		t.Fatalf("pwd = %q, want %q", got, real)
	}
	if got := readLines(t, filepath.Join(dir, "args")); !slices.Equal(got, []string{"a b", "$HOME", "*", "`id`"}) {
		t.Fatalf("args = %q, want them as written", got)
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "stdin")); err != nil || len(raw) != 0 {
		t.Fatalf("stdin = %q (%v), want empty", raw, err)
	}
}

// AC4's launch_cmd failures: a non-zero exit, a command that cannot be run,
// and one that does not return in time are errors that name the command and
// end with its output, escaped and cut to its last 4 KiB.
func TestRunDirectorLaunchCmd_Failures(t *testing.T) {
	t.Run("a non-zero exit", func(t *testing.T) {
		err := runDirectorLaunchCmd(context.Background(), []string{"/bin/sh", "-c", "echo out; echo 'err \033[2J' >&2; exit 3"}, t.TempDir(), 0)
		want := `launch_cmd /bin/sh failed: exit status 3; its output ended with: out\nerr \x1b[2J`
		if err == nil || err.Error() != want {
			t.Fatalf("err = %v, want %q", err, want)
		}
	})
	t.Run("long output keeps its end", func(t *testing.T) {
		err := runDirectorLaunchCmd(context.Background(), []string{"/bin/sh", "-c", "head -c 10000 /dev/zero | tr '\\0' a; echo; echo the-end; exit 1"}, t.TempDir(), 0)
		if err == nil || !strings.HasSuffix(err.Error(), `\nthe-end`) {
			t.Fatalf("err = %v, want it to end with the output's end", err)
		}
		if n := strings.Count(err.Error(), "a"); n > directorLaunchOutputLimit {
			t.Fatalf("err holds %d bytes of output, want at most %d", n, directorLaunchOutputLimit)
		}
	})
	t.Run("a command that cannot be run", func(t *testing.T) {
		err := runDirectorLaunchCmd(context.Background(), []string{"/nonexistent/terminal\x1b", "{script}"}, t.TempDir(), 0)
		if err == nil || !strings.HasPrefix(err.Error(), `launch_cmd /nonexistent/terminal\x1b could not be run: `) {
			t.Fatalf("err = %v, want could not be run, the name escaped", err)
		}
	})
	t.Run("no command", func(t *testing.T) {
		if err := runDirectorLaunchCmd(context.Background(), nil, t.TempDir(), 0); err == nil {
			t.Fatal("an empty argv ran")
		}
	})
	t.Run("a command that does not return", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		start := time.Now()
		err := runDirectorLaunchCmd(ctx, []string{sleepPath(t), "30"}, t.TempDir(), 100*time.Millisecond)
		if err == nil || !strings.Contains(err.Error(), "did not return in time and was killed (context deadline exceeded)") {
			t.Fatalf("err = %v, want the time limit", err)
		}
		if took := time.Since(start); took > 5*time.Second {
			t.Fatalf("the call took %s past its 200ms limit", took)
		}
	})
}

// A launch_cmd that exits 0 while a child it started in the background
// still holds its output (a terminal's daemon) returns after the wait delay,
// not when the child ends.
func TestRunDirectorLaunchCmd_ABackgroundChildHoldingTheOutputDoesNotHang(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "child.pid")
	t.Cleanup(func() {
		// The child is this test's: its pid is in the file it wrote. Stop it
		// and wait (up to 5s) for it to be gone; init reaps it.
		raw, err := os.ReadFile(pidFile)
		if err != nil {
			return
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
		if err != nil || pid <= 0 {
			return
		}
		_ = syscall.Kill(pid, syscall.SIGTERM)
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
			if syscall.Kill(pid, 0) != nil {
				return
			}
		}
	})
	start := time.Now()
	err := runDirectorLaunchCmd(context.Background(), []string{"/bin/sh", "-c", shellQuote(sleepPath(t)) + " 5 & echo $! > child.pid; exit 0"}, dir, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("runDirectorLaunchCmd: %v", err)
	}
	if took := time.Since(start); took > 3*time.Second {
		t.Fatalf("the call took %s: the child's output held it", took)
	}
}
