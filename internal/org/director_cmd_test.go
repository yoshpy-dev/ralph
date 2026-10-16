package org

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
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

	"github.com/yoshpy-dev/ralph/internal/config"
)

// Tests for director_cmd.go (plan 2026-10-16-org-director-launch-cmd, D1,
// D3, D6, and D7): the process reader, the stop of a recorded process, the launch
// script, the launch_cmd argv and its runner, and StartDirector's launch_cmd
// path (its checks, the wait for the check-in, and its cleanup). Every
// process they signal is one the test started itself, and every process a
// test starts is killed and waited for in its cleanup; none runs herdr, a
// terminal, or a tmux of the user's.

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
		if got.outcome != directorProcessStopUnreadable || got.signalled || !errors.Is(got.err, psErr) || !errors.Is(got.err, ErrDirectorProcessUnreadable) ||
			got.note != "process=not signalled: ps could not be read" {
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
		if got.outcome != directorProcessStopUnreadable || !got.signalled || !errors.Is(got.err, psErr) || !errors.Is(got.err, ErrDirectorProcessUnreadable) ||
			got.note != "process=sent SIGTERM; ps could not be read after it" {
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

	// An agmsg home starting with ~/ is written as it is, for
	// driver.ResolveAgmsgHome to expand.
	s.AgmsgHome = "~/a g"
	if text, err = s.text(); err != nil || !strings.Contains(text, "\nRALPH_ORG_AGMSG_HOME='~/a g'\nexport RALPH_ORG_AGMSG_HOME\n") {
		t.Fatalf("text with the agmsg home ~/a g = %v:\n%s", err, text)
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
		{"a relative agmsg home", func(s *directorLaunchScript) { s.AgmsgHome = "agmsg/home" }, `the agmsg home "agmsg/home" is not absolute and does not start with ~/`},
		{"an agmsg home of a bare ~", func(s *directorLaunchScript) { s.AgmsgHome = "~" }, `the agmsg home "~" is not absolute and does not start with ~/`},
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
	start := time.Now()
	err := runDirectorLaunchCmd(context.Background(), []string{"/bin/sh", "-c", shellQuote(sleepPath(t)) + " 5 & echo $! > child.pid; exit 0"}, dir, 200*time.Millisecond)
	took := time.Since(start)
	stopOwnBackgroundChild(t, filepath.Join(dir, "child.pid"), start)
	if err != nil {
		t.Fatalf("runDirectorLaunchCmd: %v", err)
	}
	if took > 3*time.Second {
		t.Fatalf("the call took %s: the child's output held it", took)
	}
}

// stopOwnBackgroundChild is the cleanup for the background child of the
// test above, whose shell wrote its pid to pidFile. It reads the pid's start
// time with ps at once and, only when the pid is alive and started no
// earlier than the test (start, less 2 s for ps's whole seconds), sends it
// SIGTERM in t's cleanup, and only while ps still reads that start time for
// it, so a process that got the pid later is never signalled; then it waits
// (up to 5s) for it to be gone, which init reaps.
func stopOwnBackgroundChild(t *testing.T, pidFile string, start time.Time) {
	t.Helper()
	raw, err := os.ReadFile(pidFile)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		return
	}
	read := func() (DirectorProcess, bool) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		proc, err := ReadDirectorProcess(ctx, pid)
		return proc, err == nil && proc.State == DirectorProcessAlive
	}
	proc, alive := read()
	if !alive {
		return
	}
	if started, err := time.Parse(directorPSStartTimeLayout, proc.StartTime); err != nil || started.Before(start.Add(-2*time.Second)) {
		return
	}
	same := func() bool {
		now, alive := read()
		return alive && now.StartTime == proc.StartTime
	}
	t.Cleanup(func() {
		if !same() {
			return
		}
		_ = syscall.Kill(pid, syscall.SIGTERM)
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline) && same(); time.Sleep(20 * time.Millisecond) {
		}
	})
}

// The tests of StartDirector's launch_cmd path (plan
// 2026-10-16-org-director-launch-cmd, AC2, AC4, and AC5) run real processes
// under t.TempDir(): a fake launch_cmd (sh), the launch script start
// writes, a check-in that is this test binary
// (TestDirectorCheckInHelperProcess) run through a ralph wrapper, and a fake
// agent that records what it got and then becomes a sleep. The fake herdr
// must see no call. Each test kills, in its cleanup, the processes its
// launch scripts checked in with, and only while ps reads the start time
// the check-in recorded for them (killCheckedInProcesses), so it never
// signals a process it did not start.

// The variables the ralph wrapper of a cmdStartFixture exports to the
// check-in helper: directorCheckInHelperEnv makes this test binary the
// check-in, and directorCheckInPauseEnv, when set, names a directory where
// the helper writes "checked-in" once it recorded the check-in, then waits
// for "release" and exits with the code that file holds, after it writes
// "done".
const (
	directorCheckInHelperEnv = "RALPH_ORG_TEST_CHECKIN_HELPER"
	directorCheckInPauseEnv  = "RALPH_ORG_TEST_CHECKIN_PAUSE"
)

// TestDirectorCheckInHelperProcess is not a test of its own: it is the
// check-in of the launch scripts these tests make start write. The ralph
// wrapper execs this test binary with directorCheckInHelperEnv set and the
// script's arguments after "--" (`org director launched --id <id> --token
// <token> --pid <pid> --state-dir <dir>`), so the check-in's parent is the
// script's shell, as for the real `ralph org director launched`. It runs
// that verb's checks (the pid is its parent, ps reads the pid alive, and
// MarkCmdLaunched) and exits 0 only when the check-in is recorded, so a
// refused one starts no agent. Without the variable it does nothing.
func TestDirectorCheckInHelperProcess(t *testing.T) {
	if os.Getenv(directorCheckInHelperEnv) != "1" {
		return
	}
	os.Exit(runDirectorCheckInHelper(flag.Args()))
}

func runDirectorCheckInHelper(args []string) int {
	fail := func(format string, a ...any) int {
		fmt.Fprintf(os.Stderr, "check-in helper: "+format+"\n", a...)
		return 1
	}
	if len(args) < 3 || !slices.Equal(args[:3], []string{"org", "director", "launched"}) {
		return fail("args %q are not org director launched ...", args)
	}
	fs := flag.NewFlagSet("launched", flag.ContinueOnError)
	id, token, stateDir := fs.String("id", "", ""), fs.String("token", "", ""), fs.String("state-dir", "", "")
	pid := fs.Int("pid", 0, "")
	if err := fs.Parse(args[3:]); err != nil {
		return fail("%v", err)
	}
	if parent := os.Getppid(); *pid != parent {
		return fail("--pid %d is not the parent %d", *pid, parent)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	proc, err := ReadDirectorProcess(ctx, *pid)
	if err != nil || proc.State != DirectorProcessAlive {
		return fail("pid %d reads %+v, %v", *pid, proc, err)
	}
	cwd, _ := os.Getwd()
	if err := NewDirectorStore(*stateDir).MarkCmdLaunched(DirectorCmdCheckIn{
		ID: *id, Token: *token, PID: *pid, StartTime: proc.StartTime, Who: DirectorWho{User: "launch-script", Cwd: cwd},
	}); err != nil {
		return fail("%v", err)
	}
	pause := os.Getenv(directorCheckInPauseEnv)
	if pause == "" {
		return 0
	}
	if err := os.WriteFile(filepath.Join(pause, "checked-in"), nil, 0o644); err != nil {
		return fail("%v", err)
	}
	code := 1
	for deadline := time.Now().Add(60 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if raw, err := os.ReadFile(filepath.Join(pause, "release")); err == nil {
			if n, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil {
				code = n
			}
			break
		}
	}
	_ = os.WriteFile(filepath.Join(pause, "done"), nil, 0o644)
	return code
}

// cmdStartFixture is a dirStartFixture whose StartDirector takes the
// launch_cmd path: launch_cmd is set, the agent lookup names a fake agent
// (recording which binary it was asked for), the ralph executable is a
// wrapper that runs TestDirectorCheckInHelperProcess as the check-in, the
// check-in is looked for every 10ms, and a stop waits 2s for the process.
type cmdStartFixture struct {
	*dirStartFixture
	base        string   // the directory of the fakes
	agentPath   string   // the fake agent
	agentMarker string   // what the fake agent wrote, once it ran
	lookedUp    []string // the names the agent lookup was asked for
	pause       string   // the check-in helper's pause directory ("" when none)
}

// cmdStartOptions shape a cmdStartFixture: pause makes the check-in wait
// after it recorded the check-in (directorCheckInPauseEnv), and ignoreTerm
// makes the fake agent ignore SIGTERM.
type cmdStartOptions struct {
	pause, ignoreTerm bool
}

func newCmdStartFixture(t *testing.T, launchCmd []string, opts cmdStartOptions) *cmdStartFixture {
	t.Helper()
	f := &cmdStartFixture{dirStartFixture: newDirStartFixture(t, directorTestToken), base: t.TempDir()}
	f.agentPath = filepath.Join(f.base, "bin", "agent")
	f.agentMarker = filepath.Join(f.base, "agent.out")
	trap := ""
	if opts.ignoreTerm {
		// An ignored signal stays ignored across exec, so the sleep it
		// becomes ignores SIGTERM too.
		trap = "trap '' TERM\n"
	}
	writeTestExecutable(t, f.agentPath, "#!/bin/sh\n"+trap+"{\nprintf 'pid=%s\\n' \"$$\"\n"+
		"printf 'pwd=%s\\n' \"$(pwd -P)\"\n"+
		"printf 'token=%s\\n' \"${RALPH_DIRECTOR_TOKEN-<unset>}\"\nprintf 'agmsg=%s\\n' \"${RALPH_ORG_AGMSG_HOME-<unset>}\"\n"+
		"for a in \"$@\"; do printf 'arg=%s\\n' \"$a\"; done\n} > "+shellQuote(f.agentMarker+".tmp")+
		" && mv "+shellQuote(f.agentMarker+".tmp")+" "+shellQuote(f.agentMarker)+"\nexec "+shellQuote(sleepPath(t))+" 600\n")
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatalf("find the test binary: %v", err)
	}
	wrapper := "#!/bin/sh\n" + directorCheckInHelperEnv + "=1\nexport " + directorCheckInHelperEnv + "\n"
	if opts.pause {
		f.pause = filepath.Join(f.base, "pause")
		if err := os.Mkdir(f.pause, 0o755); err != nil {
			t.Fatal(err)
		}
		wrapper += directorCheckInPauseEnv + "=" + shellQuote(f.pause) + "\nexport " + directorCheckInPauseEnv + "\n"
	}
	wrapper += "exec " + shellQuote(testBinary) + " -test.run='^TestDirectorCheckInHelperProcess$' -- \"$@\"\n"
	ralph := filepath.Join(f.base, "bin", "ralph")
	writeTestExecutable(t, ralph, wrapper)

	f.o.Config.Director.LaunchCmd = launchCmd
	f.o.DirectorAgentLookPath = func(file string) (string, error) {
		f.lookedUp = append(f.lookedUp, file)
		return f.agentPath, nil
	}
	f.o.DirectorRalphExecutable = func() (string, error) { return ralph, nil }
	f.o.DirectorCheckInPollInterval = 10 * time.Millisecond
	f.o.DirectorStopWait = 2 * time.Second
	killCheckedInProcesses(t, f.store)
	if f.pause != "" {
		// A paused check-in that the test did not release is released with
		// exit 1, so its launch script starts no agent.
		t.Cleanup(func() { f.release(t, 1) })
	}
	return f
}

// release lets the paused check-in exit with code, once, and waits (up to
// 10s) for it to have exited.
func (f *cmdStartFixture) release(t *testing.T, code int) {
	t.Helper()
	path := filepath.Join(f.pause, "release")
	if _, err := os.Stat(path); err == nil {
		return
	}
	if err := os.WriteFile(path, []byte(strconv.Itoa(code)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.pause, "checked-in")); err != nil {
		return
	}
	waitForFileWithin(t, filepath.Join(f.pause, "done"), 10*time.Second)
}

// killCheckedInProcesses is t's cleanup for every process a launch script
// of store's ledger checked in with: each pid of a launched event that ps
// still reads alive with the start time that event recorded gets SIGKILL,
// and the cleanup waits (up to 5s) for it to be gone. Its parent is init by
// then (nohup's shell has exited), which reaps it.
func killCheckedInProcesses(t *testing.T, store *DirectorStore) {
	t.Cleanup(func() {
		raw, err := os.ReadFile(store.Path())
		if err != nil {
			return
		}
		for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
			var ev DirectorEvent
			if json.Unmarshal([]byte(line), &ev) != nil || ev.Event != DirectorEventLaunched || ev.PID <= 0 {
				continue
			}
			same := func() bool {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				proc, err := ReadDirectorProcess(ctx, ev.PID)
				return err == nil && proc.State == DirectorProcessAlive && proc.StartTime == ev.StartTime
			}
			if !same() {
				continue
			}
			_ = syscall.Kill(ev.PID, syscall.SIGKILL)
			for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline) && same(); time.Sleep(20 * time.Millisecond) {
			}
		}
	})
}

// waitForFileWithin waits up to limit for path to exist (waitForFile, of the
// gate tests, waits 10s).
func waitForFileWithin(t *testing.T, path string, limit time.Duration) {
	t.Helper()
	for deadline := time.Now().Add(limit); ; time.Sleep(10 * time.Millisecond) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s did not appear within %s", path, limit)
		}
	}
}

// wantNoFile fails t when path exists.
func wantNoFile(t *testing.T, path, what string) {
	t.Helper()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s exists (stat %v): %s", path, err, what)
	}
}

// launchCmdInBackground is a launch_cmd that runs the launch script in the
// background, as a terminal would, and returns at once. Like a terminal,
// which has its own environment, the script does not get
// RALPH_ORG_AGMSG_HOME from the test's, so the agent has it only when the
// script exports it.
var launchCmdInBackground = []string{"sh", "-c", `env -u RALPH_ORG_AGMSG_HOME nohup sh "$1" >/dev/null 2>&1 &`, "sh", "{script}"}

// launchCmdFailingOnceExists is a launch_cmd that runs the launch script in
// the background, waits (up to 20s) for path to exist, and then exits 1: a
// terminal command that fails after its terminal ran the script.
func launchCmdFailingOnceExists(path string) []string {
	return []string{"sh", "-c", `env -u RALPH_ORG_AGMSG_HOME nohup sh "$1" >/dev/null 2>&1 & i=0; while [ ! -e "$2" ] && [ $i -lt 400 ]; do sleep 0.05; i=$((i+1)); done; exit 1`,
		"sh", "{script}", path}
}

// launchCmdTouching is a launch_cmd that never runs the launch script: it
// creates path (so a test sees that it ran) and exits with code.
func launchCmdTouching(path string, code int) []string {
	return []string{"sh", "-c", `: > "$2"; exit ` + strconv.Itoa(code), "sh", "{script}", path}
}

// agentMarkerLines is what the fake agent wrote, once it has run (up to 30s).
func (f *cmdStartFixture) agentMarkerLines(t *testing.T) []string {
	t.Helper()
	waitForFileWithin(t, f.agentMarker, 30*time.Second)
	return readLines(t, f.agentMarker)
}

// launchedEvent is the launched event of director.jsonl, which a launch cmd
// writes at its check-in only.
func launchedEvent(t *testing.T, s *DirectorStore) DirectorEvent {
	t.Helper()
	for _, line := range directorFileLines(t, s) {
		var ev DirectorEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("director.jsonl line %q: %v", line, err)
		}
		if ev.Event == DirectorEventLaunched {
			return ev
		}
	}
	t.Fatalf("director.jsonl has no launched event: %q", directorTrail(t, s))
	return DirectorEvent{}
}

// AC2: with launch_cmd, start records attached (launch cmd, the token, the
// model, and the permission mode) before anything else, writes the prompt
// and the launch script (0700), runs launch_cmd, and waits for the check-in,
// which records launched with the pid of the script and its start time. The
// agent then runs in its place (same pid) with the permission flags,
// --model, and the prompt pointer, the token exported, and
// RALPH_ORG_AGMSG_HOME exported only when start was given one. start asks
// for the driver's binary by its name, calls no herdr (HERDR_PANE_ID set
// does not refuse it, D9), and writes no manifest or receipt.
func TestStartDirector_LaunchCmdOpensTheTerminalAndChecksIn(t *testing.T) {
	cases := []struct {
		name, driver, model string
		perms               config.OrgPermissionsConfig
		agmsgHome           string
		inHerdr             bool
		wantMode            string
		wantPerm            []string
	}{
		{"claude autonomous", DirectorDriverClaude, "opus", config.OrgPermissionsConfig{}, "", false, PermissionModeAutonomous,
			[]string{"--permission-mode", "bypassPermissions"}},
		{"codex guarded, with an agmsg home, inside herdr", DirectorDriverCodex, "gpt-5-codex",
			config.OrgPermissionsConfig{Roles: map[string]string{DirectorRole: PermissionModeGuarded}}, "/agmsg home/it's", true, PermissionModeGuarded, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCmdStartFixture(t, launchCmdInBackground, cmdStartOptions{})
			f.o.Config.Permissions = tc.perms
			if tc.inHerdr {
				f.env[herdrPaneIDEnv] = "pane-9"
			}
			p := f.params(tc.driver, tc.model)
			p.TimeoutMS = 30000
			p.AgmsgHome = tc.agmsgHome
			res, err := f.o.StartDirector(p)
			if err != nil {
				t.Fatalf("StartDirector: %v", err)
			}
			const token = directorTestToken
			wantPrompt := filepath.Join(f.dir, "prompts", "director_"+token+".md")
			wantScript := filepath.Join(f.dir, "launch", "director_"+token+".sh")
			reg := res.Registration
			if reg.ID != "d1" || reg.State != DirectorStateRunning || !reg.CmdLaunch() || reg.Token != token || reg.Driver != tc.driver ||
				reg.Model != tc.model || reg.PermissionMode != tc.wantMode || reg.PID <= 0 || reg.StartTime == "" || reg.WorkspaceID != "" || reg.PaneID != "" {
				t.Fatalf("Registration = %+v, want d1 running with launch cmd, a pid and its start time, no herdr ids", reg)
			}
			if res.TerminalName != "director_"+token || res.LaunchScriptPath != wantScript ||
				res.PromptPath != wantPrompt || res.AgentName != "" || res.WorkspaceLabel != "" || res.TabLabel != "" {
				t.Fatalf("result = %+v, want launch cmd, director_%s, the script %s, and the prompt %s", res, token, wantScript, wantPrompt)
			}
			wantTrail(t, f.store, "attached d1", "launched d1")
			var attached DirectorEvent
			if err := json.Unmarshal([]byte(directorFileLines(t, f.store)[0]), &attached); err != nil || attached.Launch != DirectorLaunchCmd ||
				attached.Token != token || attached.Model != tc.model || attached.PermissionMode != tc.wantMode || attached.Driver != tc.driver {
				t.Fatalf("attached = %+v (%v), want launch cmd, the token, %s, and %s", attached, err, tc.model, tc.wantMode)
			}
			if launched := launchedEvent(t, f.store); launched.PID != reg.PID || launched.StartTime != reg.StartTime {
				t.Fatalf("launched = %+v, want the registration's pid %d and start time %q", launched, reg.PID, reg.StartTime)
			}
			if proc := mustReadProcess(t, reg.PID); proc.State != DirectorProcessAlive || proc.StartTime != reg.StartTime {
				t.Fatalf("pid %d reads %+v, want alive with the recorded start time", reg.PID, proc)
			}

			cwd, err := filepath.EvalSymlinks(f.cwd)
			if err != nil {
				t.Fatal(err)
			}
			agmsg := tc.agmsgHome
			if agmsg == "" {
				agmsg = "<unset>"
			}
			want := []string{"pid=" + strconv.Itoa(reg.PID), "pwd=" + cwd, "token=" + token, "agmsg=" + agmsg}
			for _, arg := range append(slices.Clone(tc.wantPerm), "--model", tc.model, PromptFilePointer(wantPrompt)) {
				want = append(want, "arg="+arg)
			}
			if got := f.agentMarkerLines(t); !slices.Equal(got, want) {
				t.Fatalf("the agent recorded\n%q\nwant\n%q", got, want)
			}
			if !slices.Equal(f.lookedUp, []string{tc.driver}) {
				t.Fatalf("the agent lookup was asked for %q, want [%s]", f.lookedUp, tc.driver)
			}
			if len(f.h.calls) != 0 {
				t.Fatalf("herdr calls = %v, want none on the launch_cmd path", f.h.calls)
			}
			fi, err := os.Stat(wantScript)
			if err != nil || fi.Mode().Perm() != 0o700 {
				t.Fatalf("launch script %s: %v, mode %v, want 0700", wantScript, err, fi.Mode())
			}
			script, err := os.ReadFile(wantScript)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Contains(string(script), "export RALPH_ORG_AGMSG_HOME"); got != (tc.agmsgHome != "") {
				t.Fatalf("the launch script exports RALPH_ORG_AGMSG_HOME: %v, want %v:\n%s", got, tc.agmsgHome != "", script)
			}
			prompt, err := os.ReadFile(wantPrompt)
			if err != nil || !strings.Contains(string(prompt), "人とはこの端末で話す。") || strings.Contains(string(prompt), "pane") {
				t.Fatalf("prompt file %s (%v) is not the launch cmd prompt:\n%s", wantPrompt, err, prompt)
			}
			if got := dirEntryNames(t, f.dir); !slices.Equal(got, []string{"director.jsonl", "director.lock", "launch", "prompts"}) {
				t.Fatalf("ledger entries = %v, want director.jsonl, director.lock, launch, and prompts only (no manifest, no receipts)", got)
			}
		})
	}
}

// AC5: before anything is written, start on the launch_cmd path refuses a
// launch_cmd that config.Load would refuse (an Org built without it), a
// driver's agent that is not on PATH, and a ralph executable it cannot find.
// It writes no director.jsonl, prompt, or launch script, and does not run
// launch_cmd.
func TestStartDirector_LaunchCmdRefusesWhatCannotRunTheAgent(t *testing.T) {
	cases := []struct {
		name      string
		launchCmd func(ran string) []string
		setup     func(f *cmdStartFixture)
		want      string
		wantErrIs error
	}{
		{"no {script} or {script_sh}", func(ran string) []string { return []string{"sh", "-c", `: > "$1"`, "sh", ran} }, nil,
			"[org.director].launch_cmd: no element has {script} or {script_sh}", nil},
		{"an unknown placeholder", func(ran string) []string { return append(launchCmdTouching(ran, 0), "{bogus}") }, nil,
			"has the unknown placeholder {bogus}", nil},
		{"the agent is not on PATH", func(ran string) []string { return launchCmdTouching(ran, 0) }, func(f *cmdStartFixture) {
			f.o.DirectorAgentLookPath = func(file string) (string, error) {
				return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
			}
		}, "the claude agent is not on PATH, so the launch script could not run it", exec.ErrNotFound},
		{"the ralph executable cannot be found", func(ran string) []string { return launchCmdTouching(ran, 0) }, func(f *cmdStartFixture) {
			f.o.DirectorRalphExecutable = func() (string, error) { return "", errors.New("no executable") }
		}, "find the ralph executable the launch script checks in with: no executable", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ran := filepath.Join(t.TempDir(), "launch_cmd ran")
			f := newCmdStartFixture(t, tc.launchCmd(ran), cmdStartOptions{})
			if tc.setup != nil {
				tc.setup(f)
			}
			_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.HasPrefix(err.Error(), "org: director: start: ") {
				t.Fatalf("err = %v, want a refusal saying %q", err, tc.want)
			}
			if tc.wantErrIs != nil && !errors.Is(err, tc.wantErrIs) {
				t.Fatalf("err = %v, want it to wrap %v", err, tc.wantErrIs)
			}
			if state := directorFileState(t, f.store); state != "<missing>" {
				t.Fatalf("director.jsonl = %q, want none written", state)
			}
			if got := dirEntryNames(t, f.dir); len(got) != 0 {
				t.Fatalf("ledger entries = %v, want none", got)
			}
			wantNoFile(t, ran, "a refused start ran launch_cmd")
		})
	}
}

// AC5: the refusals of a herdr start hold on the launch_cmd path too, in
// their order and with nothing written: launch_cmd does not run, and no
// herdr is called. (HERDR_PANE_ID is the exception, D9: see
// TestStartDirector_LaunchCmdOpensTheTerminalAndChecksIn.)
func TestStartDirector_LaunchCmdKeepsTheOtherRefusals(t *testing.T) {
	cases := []struct {
		name      string
		setup     func(t *testing.T, f *cmdStartFixture, p *DirectorStartParams)
		wantErrIs error
	}{
		{"enabled is false", func(_ *testing.T, f *cmdStartFixture, _ *DirectorStartParams) { f.o.Config.Director.Enabled = false }, ErrDirectorDisabled},
		{"codex autonomous", func(_ *testing.T, f *cmdStartFixture, p *DirectorStartParams) {
			p.Driver, f.o.Config.Permissions.Default = DirectorDriverCodex, PermissionModeAutonomous
		}, ErrDirectorCodexWorkspaceWrite},
		{"codex edits with codex_verified", func(_ *testing.T, f *cmdStartFixture, p *DirectorStartParams) {
			p.Driver, f.o.Config.Permissions.Default, f.o.Config.Permissions.CodexVerified = DirectorDriverCodex, PermissionModeEdits, true
		}, ErrDirectorCodexWorkspaceWrite},
		{"a promoted session is attached", func(t *testing.T, f *cmdStartFixture, _ *DirectorStartParams) {
			mustAttachDirector(t, f.store, DirectorAttach{Driver: DirectorDriverClaude, User: "u", Cwd: "/w"})
		}, ErrDirectorAttached},
		{"a launch cmd registration is still starting", func(t *testing.T, f *cmdStartFixture, _ *DirectorStartParams) {
			h := testHeadlessAttach()
			h.Launch = DirectorLaunchCmd
			if _, err := f.store.AttachHeadless(h); err != nil {
				t.Fatalf("AttachHeadless: %v", err)
			}
		}, ErrDirectorAttached},
		{"the watch gate", func(_ *testing.T, f *cmdStartFixture, _ *DirectorStartParams) {
			f.o.WatchGate = func() error {
				return &WatchGateError{StateDir: f.dir, Liveness: WatchLiveness{State: WatchLivenessNotRunning}}
			}
		}, ErrNoRunningWatch},
		{"director.jsonl unreadable", func(t *testing.T, f *cmdStartFixture, _ *DirectorStartParams) {
			if err := os.Mkdir(f.store.Path(), 0o755); err != nil {
				t.Fatal(err)
			}
		}, syscall.EISDIR},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ran := filepath.Join(t.TempDir(), "launch_cmd ran")
			f := newCmdStartFixture(t, launchCmdTouching(ran, 0), cmdStartOptions{})
			f.store.token = func() (string, error) { return "ba0000000000", nil }
			p := f.params(DirectorDriverClaude, "opus")
			tc.setup(t, f, &p)
			before := directorFileState(t, f.store)
			_, err := f.o.StartDirector(p)
			if !errors.Is(err, tc.wantErrIs) {
				t.Fatalf("err = %v, want it to wrap %v", err, tc.wantErrIs)
			}
			if after := directorFileState(t, f.store); after != before {
				t.Fatalf("director.jsonl changed from %q to %q", before, after)
			}
			for _, sub := range []string{"prompts", "launch"} {
				wantNoFile(t, filepath.Join(f.dir, sub), "a refusal wrote the "+sub+" dir")
			}
			wantNoFile(t, ran, "a refused start ran launch_cmd")
			if len(f.h.calls) != 0 {
				t.Fatalf("herdr calls = %v, want none", f.h.calls)
			}
		})
	}
}

// AC4: a launch_cmd that exits non-zero, one that does not return within
// DriverCallTimeout, and one that returns but whose terminal never runs the
// script (no check-in within TimeoutMS) each end start with exit 1 and the
// registration recorded stopped (detached first, as it is current), with
// nothing to stop: no check-in came. The error is a
// *DirectorLaunchCmdStoppedError that wraps the cause. The launch script stays
// on disk, and a terminal that runs it after that (here the test, by hand)
// is refused at its check-in, so the agent never runs. The ledger is free
// for the next start.
func TestStartDirector_ALaunchCmdThatFailsBeforeACheckInIsRecordedStopped(t *testing.T) {
	cases := []struct {
		name      string
		launchCmd []string
		setup     func(f *cmdStartFixture, p *DirectorStartParams)
		want      string
		wantErrIs error
	}{
		{"launch_cmd exits non-zero", []string{"sh", "-c", "echo no terminal >&2; exit 3", "sh", "{script}"}, nil,
			"launch_cmd sh failed: exit status 3; its output ended with: no terminal", nil},
		{"launch_cmd does not return", []string{"sh", "-c", "exec sleep 30", "sh", "{script}"},
			func(f *cmdStartFixture, _ *DirectorStartParams) { f.o.DriverCallTimeout = 300 * time.Millisecond },
			"launch_cmd sh did not return in time and was killed", nil},
		{"no check-in within the timeout", []string{"sh", "-c", ":", "sh", "{script}"},
			func(_ *cmdStartFixture, p *DirectorStartParams) { p.TimeoutMS = 300 },
			"did not check in within 300ms (--timeout-ms 300): no terminal ran it, or its check-in was refused or failed", ErrDirectorNoCheckIn},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCmdStartFixture(t, tc.launchCmd, cmdStartOptions{})
			f.store.token = func() (string, error) { return directorTestToken, nil }
			p := f.params(DirectorDriverClaude, "opus")
			p.TimeoutMS = 30000
			if tc.setup != nil {
				tc.setup(f, &p)
			}
			_, err := f.o.StartDirector(p)
			var stopped *DirectorLaunchCmdStoppedError
			if !errors.As(err, &stopped) || stopped.ID != "d1" || stopped.PID != 0 || stopped.Signalled ||
				stopped.TerminalName != "director_"+directorTestToken || !strings.Contains(err.Error(), tc.want) ||
				!strings.HasSuffix(err.Error(), "; d1 is recorded stopped") || !strings.HasPrefix(err.Error(), "org: director: start d1: ") {
				t.Fatalf("err = %#v (%v), want a *DirectorLaunchCmdStoppedError saying %q and that d1 is recorded stopped", err, err, tc.want)
			}
			if tc.wantErrIs != nil && !errors.Is(err, tc.wantErrIs) {
				t.Fatalf("err = %v, want it to wrap %v", err, tc.wantErrIs)
			}
			wantTrail(t, f.store, "attached d1", "detached d1", "stopped d1")
			wantNoDirectorOpen(t, f.store)
			if len(f.h.calls) != 0 {
				t.Fatalf("herdr calls = %v, want none", f.h.calls)
			}

			script := DirectorLaunchScriptPath(f.dir, directorTestToken)
			cmd := exec.Command(script)
			var exitErr *exec.ExitError
			if out, err := cmd.CombinedOutput(); !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 ||
				!strings.Contains(string(out), "is already recorded stopped") {
				t.Fatalf("the launch script run late = %v (%s), want exit 1 with the refused check-in", err, out)
			}
			wantNoFile(t, f.agentMarker, "the agent ran after a refused check-in")
			wantTrail(t, f.store, "attached d1", "detached d1", "stopped d1")

			f.store.token = func() (string, error) { return "ba0000000000", nil }
			f.o.Config.Director.LaunchCmd = nil
			if next := f.mustStart(t, DirectorDriverClaude, "opus"); next.Registration.ID != "d2" {
				t.Fatalf("the next start registered %s, want d2", next.Registration.ID)
			}
		})
	}
}

// AC4: an attach --replace that lands while start waits for the check-in
// makes start clean up its own launch: it records it stopped (not detached,
// as d2 is current now) and fails with ErrDirectorReplacedDuringStart; d2
// stays current.
func TestStartDirector_AReplaceWhileWaitingForTheCheckInEndsItsOwnLaunch(t *testing.T) {
	ran := filepath.Join(t.TempDir(), "launch_cmd ran")
	f := newCmdStartFixture(t, launchCmdTouching(ran, 0), cmdStartOptions{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
			if _, err := os.Stat(ran); err == nil {
				if _, _, err := f.store.Attach(DirectorAttach{Driver: DirectorDriverClaude, Replace: true, User: "carol", Cwd: "/c"}); err != nil {
					t.Errorf("Attach --replace: %v", err)
				}
				return
			}
		}
		t.Error("launch_cmd did not run within 20s")
	}()
	p := f.params(DirectorDriverClaude, "opus")
	p.TimeoutMS = 20000
	_, err := f.o.StartDirector(p)
	<-done
	var stopped *DirectorLaunchCmdStoppedError
	if !errors.As(err, &stopped) || !errors.Is(err, ErrDirectorReplacedDuringStart) || !strings.Contains(err.Error(), "the current registration is d2") ||
		!strings.HasSuffix(err.Error(), "; d1 is recorded stopped") {
		t.Fatalf("err = %v, want ErrDirectorReplacedDuringStart naming d2 and a cleanup that recorded d1 stopped", err)
	}
	wantTrail(t, f.store, "attached d1", "detached d1", "attached d2", "stopped d1")
	if res := mustReadDirector(t, f.store); currentDirectorID(res) != "d2" || len(res.OpenHeadless) != 0 {
		t.Fatalf("current %q, open %q, want d2 current and nothing open", currentDirectorID(res), openHeadlessSummary(res))
	}
}

// AC4: a `ralph org director stop --id d1` that lands while start waits for
// the check-in (no pid yet) records d1 detached and stopped. start's cleanup
// then finds d1 already stopped and counts that stop as its record: it
// returns a *DirectorLaunchCmdStoppedError and writes nothing more, not a
// *DirectorCleanupError that names a stop no longer needed.
func TestStartDirector_AStopWhileWaitingForTheCheckInIsTheCleanupsRecord(t *testing.T) {
	ran := filepath.Join(t.TempDir(), "launch_cmd ran")
	f := newCmdStartFixture(t, launchCmdTouching(ran, 0), cmdStartOptions{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
			if _, err := os.Stat(ran); err == nil {
				if res, err := f.stop("d1"); err != nil || res.ProcessNote != "process=none recorded" {
					t.Errorf("the stop during the wait = %+v, %v, want d1 stopped with no process recorded", res, err)
				}
				return
			}
		}
		t.Error("launch_cmd did not run within 20s")
	}()
	p := f.params(DirectorDriverClaude, "opus")
	p.TimeoutMS = 20000
	_, err := f.o.StartDirector(p)
	<-done
	var stopped *DirectorLaunchCmdStoppedError
	if !errors.As(err, &stopped) || !errors.Is(err, ErrDirectorReplacedDuringStart) || stopped.ID != "d1" || stopped.PID != 0 || stopped.Signalled ||
		!strings.Contains(err.Error(), "no registration is current") || !strings.HasSuffix(err.Error(), "; d1 is recorded stopped") {
		t.Fatalf("err = %#v (%v), want ErrDirectorReplacedDuringStart and a cleanup that counts the stop as its record", err, err)
	}
	wantTrail(t, f.store, "attached d1", "detached d1", "stopped d1")
	wantNoDirectorOpen(t, f.store)
	if len(f.h.calls) != 0 {
		t.Fatalf("herdr calls = %v, want none", f.h.calls)
	}
}

// AC4 and D6: a check-in that lands after the cleanup read the registration
// (no pid then) and before it records the stop makes that record refuse
// (ErrDirectorCheckedInDuringStop). The cleanup reads once more, finds the
// pid, sends SIGTERM to that process, which the test started (identity by
// its start time), sees it end, and then records the stop.
func TestStartDirector_ACheckInDuringTheCleanupIsReadAgainAndStopped(t *testing.T) {
	ran := filepath.Join(t.TempDir(), "launch_cmd ran")
	f := newCmdStartFixture(t, launchCmdTouching(ran, 1), cmdStartOptions{})
	agent := startTestProcess(t, sleepPath(t), "600")
	hooks := 0
	f.store.beforeMarkStopped = func() {
		hooks++
		if hooks > 1 {
			return
		}
		proc := mustReadProcess(t, agent.pid())
		if err := f.store.MarkCmdLaunched(DirectorCmdCheckIn{
			ID: "d1", Token: directorTestToken, PID: agent.pid(), StartTime: proc.StartTime, Who: testDirectorWho,
		}); err != nil {
			t.Errorf("the check-in during the cleanup: %v", err)
		}
	}
	_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
	var stopped *DirectorLaunchCmdStoppedError
	if !errors.As(err, &stopped) || !stopped.Signalled || stopped.PID != agent.pid() ||
		!strings.Contains(err.Error(), fmt.Sprintf("start sent SIGTERM to the launch script or agent that checked in (pid %d), which ended, and d1 is recorded stopped", agent.pid())) {
		t.Fatalf("err = %v, want a cleanup that read the check-in again, stopped pid %d, and recorded d1 stopped", err, agent.pid())
	}
	if hooks != 2 {
		t.Fatalf("the stop was recorded after %d tries, want 2 (refused once, then recorded)", hooks)
	}
	if !agent.signalledBy(t, syscall.SIGTERM) {
		t.Fatalf("the process that checked in was not ended by SIGTERM: %v", agent.cmd.ProcessState)
	}
	wantTrail(t, f.store, "attached d1", "launched d1", "detached d1", "stopped d1")
	wantNoDirectorOpen(t, f.store)
}

// AC4 and D6: a launch_cmd that fails after its terminal ran the script,
// while the check-in has recorded the pid of the script's shell and has not
// yet let it exec the agent (the helper pauses there): the cleanup reads
// that pid alive with the recorded start time, sends it SIGTERM, sees it
// end, and records the stop. Released then with success, the check-in's
// parent is gone, so the agent never runs.
func TestStartDirector_TheCleanupStopsALaunchScriptPausedAfterItsCheckIn(t *testing.T) {
	f := newCmdStartFixture(t, nil, cmdStartOptions{pause: true})
	f.o.Config.Director.LaunchCmd = launchCmdFailingOnceExists(filepath.Join(f.pause, "checked-in"))
	f.o.DriverCallTimeout = 30 * time.Second
	_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
	var stopped *DirectorLaunchCmdStoppedError
	if !errors.As(err, &stopped) || !stopped.Signalled || stopped.PID <= 0 || !strings.Contains(err.Error(), "launch_cmd sh failed: exit status 1") ||
		!strings.Contains(err.Error(), "start sent SIGTERM to the launch script or agent that checked in") {
		t.Fatalf("err = %v, want launch_cmd's failure and a cleanup that sent SIGTERM to the script", err)
	}
	launched := launchedEvent(t, f.store)
	if launched.PID != stopped.PID {
		t.Fatalf("the cleanup stopped pid %d, want the checked-in %d", stopped.PID, launched.PID)
	}
	if proc := mustReadProcess(t, launched.PID); proc.State == DirectorProcessAlive && proc.StartTime == launched.StartTime {
		t.Fatalf("the script's shell, pid %d, reads %+v after the cleanup, want it ended", launched.PID, proc)
	}
	wantTrail(t, f.store, "attached d1", "launched d1", "detached d1", "stopped d1")

	f.release(t, 0)
	time.Sleep(300 * time.Millisecond)
	wantNoFile(t, f.agentMarker, "the agent ran although its launch script was stopped before its exec")
}

// AC4 and D6: when the cleanup cannot confirm that the process that checked
// in ended, it records no stopped. It writes detached only (the registration
// was current), leaving it open, and returns a *DirectorCleanupError of a
// launch cmd that names the stop to run and no herdr. Either the process
// ignores SIGTERM and outlives DirectorStopWait, or ps cannot be read, in
// which case nothing is sent at all.
func TestStartDirector_ACleanupThatCannotConfirmTheEndLeavesItDetachedOnly(t *testing.T) {
	cases := []struct {
		name       string
		ignoreTerm bool
		readerErr  bool
		want       string
	}{
		{"the agent ignores SIGTERM", true, false, "did not end within 300ms of its SIGTERM"},
		{"ps cannot be read", false, true, "ps is broken"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCmdStartFixture(t, nil, cmdStartOptions{ignoreTerm: tc.ignoreTerm})
			f.o.Config.Director.LaunchCmd = launchCmdFailingOnceExists(f.agentMarker)
			f.o.DriverCallTimeout = 30 * time.Second
			f.o.DirectorStopWait = 300 * time.Millisecond
			if tc.readerErr {
				f.o.DirectorProcessReader = func(context.Context, int) (DirectorProcess, error) {
					return DirectorProcess{}, errors.New("ps is broken")
				}
			}
			_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
			var ce *DirectorCleanupError
			if !errors.As(err, &ce) || ce.Verb != "start" || ce.ID != "d1" || ce.Launch != DirectorLaunchCmd || !ce.Detached ||
				ce.PID <= 0 || ce.TerminalName != "director_"+directorTestToken || ce.StopCommand != "ralph org director stop --id d1" ||
				len(ce.HerdrCommands) != 0 || ce.WorkspaceLabel != "" || ce.TabLabel != "" {
				t.Fatalf("err = %#v, want a *DirectorCleanupError of a launch cmd, detached only, with the stop command", err)
			}
			for _, want := range []string{"launch_cmd sh failed: exit status 1", "its cleanup did not finish: d1 is not recorded stopped, since the end of the process that checked in is not confirmed",
				tc.want, "d1 was recorded detached only", "record the stop with ralph org director stop --id d1"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("err = %v, want it to say %q", err, want)
				}
			}
			if strings.Contains(err.Error(), "herdr") || strings.Contains(err.Error(), "--force") {
				t.Errorf("err = %v names herdr or --force for a launch cmd", err)
			}
			wantTrail(t, f.store, "attached d1", "launched d1", "detached d1")
			res := mustReadDirector(t, f.store)
			if res.Current != nil || len(res.OpenHeadless) != 1 || res.OpenHeadless[0].ID != "d1" || res.OpenHeadless[0].PID != ce.PID {
				t.Fatalf("current %+v, open %q, want none current and d1 open with pid %d", res.Current, openHeadlessSummary(res), ce.PID)
			}
			if proc := mustReadProcess(t, ce.PID); proc.State != DirectorProcessAlive || proc.StartTime != res.OpenHeadless[0].StartTime {
				t.Fatalf("pid %d reads %+v, want the agent still running", ce.PID, proc)
			}
		})
	}
}

// AC4 and D6: the same unconfirmed end once the registration is no longer
// current, as an attach --replace or a detach --force landed while
// launch_cmd ran: Detach by its id is refused, so the cleanup writes
// nothing. d1 stays open, and the *DirectorCleanupError is not Detached and
// says nothing more was recorded, as the detached is not the cleanup's.
func TestStartDirector_ACleanupThatCannotConfirmTheEndWritesNothingOnceItIsNotCurrent(t *testing.T) {
	cases := []struct {
		name        string
		land        func(s *DirectorStore) error
		wantTrail   []string
		wantCurrent string
	}{
		{"an attach --replace", func(s *DirectorStore) error {
			_, _, err := s.Attach(DirectorAttach{Driver: DirectorDriverClaude, Replace: true, User: "carol", Cwd: "/c"})
			return err
		}, []string{"attached d1", "launched d1", "detached d1", "attached d2"}, "d2"},
		{"a detach --force", func(s *DirectorStore) error {
			_, err := s.Detach(DirectorDetach{Force: true, User: "carol", Cwd: "/c"})
			return err
		}, []string{"attached d1", "launched d1", "detached d1+force"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCmdStartFixture(t, nil, cmdStartOptions{ignoreTerm: true})
			landed := filepath.Join(f.base, "landed")
			f.o.Config.Director.LaunchCmd = launchCmdFailingOnceExists(landed)
			f.o.DriverCallTimeout = 30 * time.Second
			f.o.DirectorStopWait = 300 * time.Millisecond
			done := make(chan struct{})
			go func() {
				defer close(done)
				// launch_cmd fails only once landed exists, so the replace
				// or the detach is recorded before the cleanup reads d1.
				defer func() { _ = os.WriteFile(landed, nil, 0o644) }()
				for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
					if _, err := os.Stat(f.agentMarker); err == nil {
						if err := tc.land(f.store); err != nil {
							t.Errorf("%s: %v", tc.name, err)
						}
						return
					}
				}
				t.Error("the agent did not run within 20s")
			}()
			_, err := f.o.StartDirector(f.params(DirectorDriverClaude, "opus"))
			<-done
			var ce *DirectorCleanupError
			if !errors.As(err, &ce) || ce.ID != "d1" || ce.Launch != DirectorLaunchCmd || ce.Detached || ce.PID <= 0 ||
				ce.StopCommand != "ralph org director stop --id d1" {
				t.Fatalf("err = %#v (%v), want a *DirectorCleanupError of a launch cmd, not Detached, with the pid and the stop command", err, err)
			}
			for _, want := range []string{"launch_cmd sh failed: exit status 1", "d1 is not recorded stopped, since the end of the process that checked in is not confirmed",
				"did not end within 300ms of its SIGTERM", "; nothing more was recorded", "record the stop with ralph org director stop --id d1"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("err = %v, want it to say %q", err, want)
				}
			}
			if strings.Contains(err.Error(), "recorded detached only") {
				t.Errorf("err = %v says the cleanup recorded d1 detached, but %s did", err, tc.name)
			}
			wantTrail(t, f.store, tc.wantTrail...)
			launched := launchedEvent(t, f.store)
			res := mustReadDirector(t, f.store)
			if currentDirectorID(res) != tc.wantCurrent || len(res.OpenHeadless) != 1 || res.OpenHeadless[0].ID != "d1" ||
				res.OpenHeadless[0].PID != launched.PID || ce.PID != launched.PID {
				t.Fatalf("current %q, open %q, error pid %d, want %q current and d1 open with the checked-in pid %d",
					currentDirectorID(res), openHeadlessSummary(res), ce.PID, tc.wantCurrent, launched.PID)
			}
			if proc := mustReadProcess(t, ce.PID); proc.State != DirectorProcessAlive || proc.StartTime != launched.StartTime {
				t.Fatalf("pid %d reads %+v, want the agent still running", ce.PID, proc)
			}
			if len(f.h.calls) != 0 {
				t.Fatalf("herdr calls = %v, want none", f.h.calls)
			}
		})
	}
}

// The text of a *DirectorLaunchCmdStoppedError says what the cleanup did to
// the pid that checked in, and unwraps to the cause.
func TestDirectorLaunchCmdStoppedError_Text(t *testing.T) {
	cause := fmt.Errorf("org: director: start d4: %w", ErrDirectorNoCheckIn)
	for _, tc := range []struct {
		e    DirectorLaunchCmdStoppedError
		want string
	}{
		{DirectorLaunchCmdStoppedError{ID: "d4", Err: cause}, "; d4 is recorded stopped"},
		{DirectorLaunchCmdStoppedError{ID: "d4", PID: 77, Err: cause}, "; the launch script or agent that checked in (pid 77) had already ended, and d4 is recorded stopped"},
		{DirectorLaunchCmdStoppedError{ID: "d4", PID: 77, Signalled: true, Err: cause},
			"; start sent SIGTERM to the launch script or agent that checked in (pid 77), which ended, and d4 is recorded stopped"},
	} {
		if got := tc.e.Error(); got != cause.Error()+tc.want {
			t.Errorf("Error() = %q, want %q", got, cause.Error()+tc.want)
		}
		if !errors.Is(&tc.e, ErrDirectorNoCheckIn) {
			t.Errorf("%+v does not unwrap to its cause", tc.e)
		}
	}
}
