package org

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/yoshpy-dev/ralph/internal/config"
)

// The headless director of a launch cmd (plan
// docs/plans/active/2026-10-16-org-director-launch-cmd.md, D1, D3, and D7):
// `ralph org director start` with [org.director].launch_cmd writes a launch
// script under the ledger, runs launch_cmd with its placeholders replaced and
// without a shell, and the terminal it opens runs the script. The script
// checks in with ralph (`ralph org director launched`, which records its pid
// and the pid's start time, MarkCmdLaunched) and only then execs the agent,
// which keeps that pid and start time. A stop, or start's cleanup, sends
// SIGTERM to the recorded pid only while ps still reads the same start time
// for it, so a later process that got the pid is never signalled. This file
// holds those pieces: the process reader, the stop of the recorded process,
// the launch script, and the launch_cmd argv and its runner. The records are
// in director.go and director_start.go.

// The states of a process as ReadDirectorProcess reads it
// (DirectorProcess.State).
const (
	// DirectorProcessAlive is a process ps lists that is not a zombie.
	DirectorProcessAlive = "alive"
	// DirectorProcessZombie is a process that has exited but whose parent
	// has not reaped it yet (ps's stat starts with Z): it has ended.
	DirectorProcessZombie = "zombie"
	// DirectorProcessGone is a pid no process has.
	DirectorProcessGone = "gone"
)

// DirectorProcess is what ReadDirectorProcess read for a pid: its State and,
// for a process ps lists, its StartTime, ps's lstart in the C locale and UTC
// with the runs of spaces folded to one ("Thu Oct 15 21:51:09 2026"). Two
// reads of the same process give the same StartTime; it is meant for that
// comparison, on the same machine, not for display.
type DirectorProcess struct {
	State     string
	StartTime string
}

// directorPSStartTimeLayout is ps's lstart in the C locale, ctime's format,
// which ReadDirectorProcess checks a start time against.
const directorPSStartTimeLayout = "Mon Jan _2 15:04:05 2006"

// directorPSWaitDelay is the exec.Cmd.WaitDelay of ReadDirectorProcess's ps.
const directorPSWaitDelay = time.Second

// directorPSEnv is the whole environment ReadDirectorProcess runs ps with:
// PATH, LC_ALL=C, and TZ=UTC, so lstart has one format whatever the caller's
// locale and time zone (plan Assumptions), and no PS_FORMAT or
// PS_PERSONALITY of the caller's changes what ps prints.
func directorPSEnv() []string {
	return []string{"PATH=" + os.Getenv("PATH"), "LC_ALL=C", "TZ=UTC"}
}

// ReadDirectorProcess reads process pid with `ps -o stat=,lstart= -p <pid>`
// under directorPSEnv, as the check-in (`ralph org director launched`) and
// the stop of a launch cmd both read it (plan D4 and D7). It returns
// DirectorProcessGone when ps exits 1 with no output (no process has pid),
// DirectorProcessZombie when ps's stat starts with Z, and otherwise
// DirectorProcessAlive with the start time. Any other result is an error, and
// the caller treats the process as unreadable: pid is not positive, ps could
// not be run or did not answer before ctx ended, it failed otherwise (its
// stderr is in the error), or its output is not a stat and a start time.
func ReadDirectorProcess(ctx context.Context, pid int) (DirectorProcess, error) {
	if pid <= 0 {
		return DirectorProcess{}, fmt.Errorf("read process %d: the pid is not a positive number", pid)
	}
	cmd := exec.CommandContext(ctx, "ps", "-o", "stat=,lstart=", "-p", strconv.Itoa(pid))
	cmd.Env = directorPSEnv()
	cmd.WaitDelay = directorPSWaitDelay
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	out, errText := strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String())
	var exitErr *exec.ExitError
	switch {
	case ctx.Err() != nil:
		return DirectorProcess{}, fmt.Errorf("read process %d: ps did not answer: %w", pid, context.Cause(ctx))
	case errors.As(err, &exitErr) && exitErr.ExitCode() == 1 && out == "" && errText == "":
		return DirectorProcess{State: DirectorProcessGone}, nil
	case err != nil && errText != "":
		return DirectorProcess{}, fmt.Errorf("read process %d: ps: %w: %s", pid, err, PrintableInboxText(errText, false))
	case err != nil:
		return DirectorProcess{}, fmt.Errorf("read process %d: ps: %w", pid, err)
	}
	fields := strings.Fields(out)
	if len(fields) == 0 || strings.Contains(out, "\n") {
		return DirectorProcess{}, fmt.Errorf("read process %d: ps printed %q, not one line with its stat and start time", pid, out)
	}
	stat, start := fields[0], strings.Join(fields[1:], " ")
	if strings.HasPrefix(stat, "Z") {
		return DirectorProcess{State: DirectorProcessZombie, StartTime: start}, nil
	}
	if _, err := time.Parse(directorPSStartTimeLayout, start); err != nil {
		return DirectorProcess{}, fmt.Errorf("read process %d: ps printed the start time %q, which is not of the form %q", pid, start, directorPSStartTimeLayout)
	}
	return DirectorProcess{State: DirectorProcessAlive, StartTime: start}, nil
}

// readDirectorProcess is ReadDirectorProcess, or o.DirectorProcessReader
// when set, under one driverCallTimeout.
func (o *Org) readDirectorProcess(pid int) (DirectorProcess, error) {
	read := ReadDirectorProcess
	if o.DirectorProcessReader != nil {
		read = o.DirectorProcessReader
	}
	ctx, cancel := context.WithTimeout(context.Background(), o.driverCallTimeout())
	defer cancel()
	return read(ctx, pid)
}

// directorProcessEnded reports whether proc, read for the pid recorded for
// reg, shows that reg's process has ended (plan D7), and how: the pid has no
// process, it is a zombie, or it has a process with another start time, a
// later one that got the same pid. Only an alive process with reg's start
// time is reg's process, which has not ended.
func directorProcessEnded(reg DirectorRegistration, proc DirectorProcess) (ended bool, how string) {
	switch {
	case proc.State == DirectorProcessGone:
		return true, "no process has pid " + strconv.Itoa(reg.PID)
	case proc.State == DirectorProcessZombie:
		return true, "pid " + strconv.Itoa(reg.PID) + " is a zombie"
	case proc.StartTime != reg.StartTime:
		return true, fmt.Sprintf("pid %d is another process now (started %q, not %q)", reg.PID, proc.StartTime, reg.StartTime)
	}
	return false, ""
}

// defaultDirectorStopWait bounds how long stopDirectorProcess waits, after
// its SIGTERM, for the director process to end, when Org.DirectorStopWait is
// unset (plan D7).
const defaultDirectorStopWait = 10 * time.Second

// directorStopPollInterval is how often stopDirectorProcess reads the
// process again while it waits for it to end.
const directorStopPollInterval = 100 * time.Millisecond

func (o *Org) directorStopWait() time.Duration {
	if o.DirectorStopWait > 0 {
		return o.DirectorStopWait
	}
	return defaultDirectorStopWait
}

// The outcomes of stopDirectorProcess (directorProcessStop.outcome).
const (
	// directorProcessStopEnded: the recorded process has ended, after a
	// SIGTERM or before one was needed, or none was recorded.
	directorProcessStopEnded = "ended"
	// directorProcessStopNotEnded: the process was sent SIGTERM and was
	// still running when the wait ran out, or the SIGTERM could not be sent.
	directorProcessStopNotEnded = "not_ended"
	// directorProcessStopUnreadable: ps could not be read, so whether the
	// process ended is not known.
	directorProcessStopUnreadable = "unreadable"
)

// directorProcessStop is what stopDirectorProcess did: its outcome, whether
// it sent the SIGTERM, a note for the output ("process=..."), and, for an
// outcome other than ended, why.
type directorProcessStop struct {
	outcome   string
	signalled bool
	note      string
	err       error
}

// stopDirectorProcess stops the process recorded for reg, a launch cmd,
// the way start's cleanup and `ralph org director stop` stop it (plan D6 and
// D7). With no pid recorded (no check-in yet) there is nothing to stop and it
// returns ended. Otherwise it reads the pid (readDirectorProcess) and sends
// nothing when it cannot (unreadable) or when the read shows reg's process
// has ended (directorProcessEnded: gone, a zombie, or another start time,
// which it never signals). Only for an alive process with reg's start time,
// reg's launch script before its exec or the agent after it, does it send
// SIGTERM, to that pid alone (not a process group), and then read the pid
// again every directorStopPollInterval until it has ended or
// directorStopWait has passed (not ended) or a read fails (unreadable). It
// never sends SIGKILL. A SIGTERM that finds no process (ESRCH) means it
// ended; any other failure to send it is not ended.
func (o *Org) stopDirectorProcess(reg DirectorRegistration) directorProcessStop {
	if reg.PID <= 0 {
		return directorProcessStop{outcome: directorProcessStopEnded, note: "process=none recorded"}
	}
	proc, err := o.readDirectorProcess(reg.PID)
	if err != nil {
		return directorProcessStop{outcome: directorProcessStopUnreadable, note: "process=not signalled: ps could not be read", err: err}
	}
	if ended, how := directorProcessEnded(reg, proc); ended {
		return directorProcessStop{outcome: directorProcessStopEnded, note: "process=already ended: " + how}
	}
	if err := syscall.Kill(reg.PID, syscall.SIGTERM); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return directorProcessStop{outcome: directorProcessStopEnded, note: "process=already ended: no process has pid " + strconv.Itoa(reg.PID)}
		}
		return directorProcessStop{outcome: directorProcessStopNotEnded, note: "process=SIGTERM failed",
			err: fmt.Errorf("send SIGTERM to pid %d: %w", reg.PID, err)}
	}
	wait := o.directorStopWait()
	deadline := time.Now().Add(wait)
	for {
		proc, err := o.readDirectorProcess(reg.PID)
		if err != nil {
			return directorProcessStop{outcome: directorProcessStopUnreadable, signalled: true,
				note: "process=sent SIGTERM; ps could not be read after it", err: err}
		}
		if ended, _ := directorProcessEnded(reg, proc); ended {
			return directorProcessStop{outcome: directorProcessStopEnded, signalled: true, note: "process=ended after SIGTERM"}
		}
		left := time.Until(deadline)
		if left <= 0 {
			return directorProcessStop{outcome: directorProcessStopNotEnded, signalled: true,
				note: "process=still running after SIGTERM", err: fmt.Errorf("pid %d did not end within %s of its SIGTERM", reg.PID, wait)}
		}
		time.Sleep(min(directorStopPollInterval, left))
	}
}

// directorTokenEnv is the variable the launch script exports to the agent:
// the launch token, so a `ralph org director stop` run from inside the
// director can tell it is its own (plan D8).
const directorTokenEnv = "RALPH_DIRECTOR_TOKEN"

// directorAgmsgHomeEnv is the agmsg home variable (driver.Agmsg reads it)
// that the launch script exports when start's environment has it (plan D3).
const directorAgmsgHomeEnv = "RALPH_ORG_AGMSG_HOME"

// DirectorLaunchScriptPath is where start writes the launch script of the
// launch with token: <stateDir>/launch/director_<token>.sh.
func DirectorLaunchScriptPath(stateDir, token string) string {
	return filepath.Join(stateDir, "launch", DirectorAgentName(token)+".sh")
}

// directorLaunchScript is the launch script of a launch cmd (plan D3): the
// registration ID and launch Token; StateDir, the ledger (absolute), whose
// launch/ it goes in and which the check-in names as --state-dir; Cwd
// (absolute), where the agent runs; AgmsgHome, start's
// RALPH_ORG_AGMSG_HOME ("" exports none); RalphPath and AgentPath, the
// absolute paths of the ralph that ran start (os.Executable) and of the
// driver's agent (exec.LookPath in start's environment); and the agent's
// PermArgs, Model, and PromptPointer (PromptFilePointer of the prompt file).
type directorLaunchScript struct {
	ID, Token     string
	StateDir, Cwd string
	AgmsgHome     string
	RalphPath     string
	AgentPath     string
	PermArgs      []string
	Model         string
	PromptPointer string
}

// text is the script s describes, or an error naming what is not usable:
// an id or token not of their form, a path that is not absolute, an empty
// model or prompt pointer, or a value holding a NUL byte, which no shell
// word can hold. It cds to Cwd (exit 1 when it cannot), exports
// RALPH_DIRECTOR_TOKEN and, when AgmsgHome is set, RALPH_ORG_AGMSG_HOME,
// runs the check-in `<ralph> org director launched --id <id> --token <token>
// --pid "$$" --state-dir <state dir>` and exits 1 when it fails, so a
// refused check-in starts no agent, and then execs the agent with
// PermArgs, --model, Model, and PromptPointer, in the order herdr's agent
// start gets them. Every value is single-quoted (shellQuote), whatever it
// holds.
func (s directorLaunchScript) text() (string, error) {
	if _, ok := directorIDNumber(s.ID); !ok {
		return "", fmt.Errorf("launch script: %q is not a registration id (d<N>, such as d1)", s.ID)
	}
	if !directorTokenPattern.MatchString(s.Token) {
		return "", fmt.Errorf("launch script: launch token %q is not 12 lowercase hex digits", s.Token)
	}
	for _, p := range []struct{ what, path string }{
		{"state dir", s.StateDir}, {"cwd", s.Cwd}, {"ralph executable", s.RalphPath}, {"agent", s.AgentPath},
	} {
		if !filepath.IsAbs(p.path) {
			return "", fmt.Errorf("launch script: the %s path %q is not absolute", p.what, p.path)
		}
	}
	switch {
	case s.Model == "":
		return "", errors.New("launch script: the model is empty")
	case s.PromptPointer == "":
		return "", errors.New("launch script: the prompt pointer is empty")
	}
	for _, v := range append([]string{s.StateDir, s.Cwd, s.AgmsgHome, s.RalphPath, s.AgentPath, s.Model, s.PromptPointer}, s.PermArgs...) {
		if strings.ContainsRune(v, 0) {
			return "", fmt.Errorf("launch script: the value %q holds a NUL byte", v)
		}
	}
	var b strings.Builder
	b.WriteString("#!/bin/sh\n")
	fmt.Fprintf(&b, "# The launch script of the headless director %s (%s), written by ralph org director start.\n", s.ID, DirectorAgentName(s.Token))
	b.WriteString("# It checks in with ralph, which records its pid, and only then execs the agent in its place.\n")
	b.WriteString("cd " + shellQuote(s.Cwd) + " || exit 1\n")
	b.WriteString(directorTokenEnv + "=" + shellQuote(s.Token) + "\nexport " + directorTokenEnv + "\n")
	if s.AgmsgHome != "" {
		b.WriteString(directorAgmsgHomeEnv + "=" + shellQuote(s.AgmsgHome) + "\nexport " + directorAgmsgHomeEnv + "\n")
	}
	b.WriteString(shellQuote(s.RalphPath) + " org director launched --id " + shellQuote(s.ID) + " --token " + shellQuote(s.Token) +
		` --pid "$$" --state-dir ` + shellQuote(s.StateDir) + " || exit 1\n")
	words := []string{shellQuote(s.AgentPath)}
	for _, arg := range append(append([]string{}, s.PermArgs...), "--model", s.Model, s.PromptPointer) {
		words = append(words, shellQuote(arg))
	}
	b.WriteString("exec " + strings.Join(words, " ") + "\n")
	return b.String(), nil
}

// writeDirectorLaunchScript writes s's text (text) to
// DirectorLaunchScriptPath and returns that path. The launch/ directory is
// made 0700 (and set to 0700 when it exists), and the script, 0700, is
// written to a temporary file in it and renamed into place, so a terminal
// never runs part of it.
func writeDirectorLaunchScript(s directorLaunchScript) (string, error) {
	text, err := s.text()
	if err != nil {
		return "", err
	}
	path := DirectorLaunchScriptPath(s.StateDir, s.Token)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("launch script: create %s: %w", dir, err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", fmt.Errorf("launch script: chmod %s: %w", dir, err)
	}
	f, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return "", fmt.Errorf("launch script: %w", err)
	}
	tmp := f.Name()
	_, err = f.WriteString(text)
	if err == nil {
		err = f.Chmod(0o700)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmp, path)
	}
	if err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("launch script: write %s: %w", path, err)
	}
	return path, nil
}

// directorLaunchArgv is launchCmd, [org.director].launch_cmd, with its
// placeholders replaced in every element (plan D1): {script} by scriptPath,
// {script_sh} by scriptPath single-quoted for a shell (shellQuote), {name} by
// DirectorAgentName(token), and {cwd} by cwd. It validates launchCmd first
// with config.ValidateDirectorLaunchCmd, since an Org can be built without
// config.Load, and an empty one is an error: start uses herdr then.
func directorLaunchArgv(launchCmd []string, scriptPath, token, cwd string) ([]string, error) {
	if len(launchCmd) == 0 {
		return nil, errors.New("[org.director].launch_cmd is empty")
	}
	if err := config.ValidateDirectorLaunchCmd(launchCmd); err != nil {
		return nil, err
	}
	values := map[string]string{
		config.DirectorLaunchPlaceholderScript:   scriptPath,
		config.DirectorLaunchPlaceholderScriptSh: shellQuote(scriptPath),
		config.DirectorLaunchPlaceholderName:     DirectorAgentName(token),
		config.DirectorLaunchPlaceholderCwd:      cwd,
	}
	argv := make([]string, 0, len(launchCmd))
	for _, arg := range launchCmd {
		expanded, err := config.ExpandDirectorLaunchArg(arg, values)
		if err != nil {
			return nil, err
		}
		argv = append(argv, expanded)
	}
	return argv, nil
}

// directorLaunchOutputLimit is how much of launch_cmd's output, its last
// bytes, runDirectorLaunchCmd keeps for its error.
const directorLaunchOutputLimit = 4096

// defaultDirectorLaunchWaitDelay is runDirectorLaunchCmd's exec.Cmd.WaitDelay
// when its caller gives none.
const defaultDirectorLaunchWaitDelay = 2 * time.Second

// directorTailBuffer keeps the last limit bytes written to it. os/exec calls
// Write from one goroutine at a time when it is both Stdout and Stderr.
type directorTailBuffer struct {
	limit int
	b     []byte
}

func (w *directorTailBuffer) Write(p []byte) (int, error) {
	w.b = append(w.b, p...)
	if over := len(w.b) - w.limit; over > 0 {
		w.b = append(w.b[:0], w.b[over:]...)
	}
	return len(p), nil
}

// runDirectorLaunchCmd runs argv, the replaced launch_cmd
// (directorLaunchArgv), without a shell: argv[0] with the rest as its
// arguments, in dir, with stdin empty and stdout and stderr kept together
// (their last directorLaunchOutputLimit bytes), under ctx, which bounds it
// (start gives DriverCallTimeout). waitDelay (defaultDirectorLaunchWaitDelay
// when 0 or less) is its exec.Cmd.WaitDelay, so a background child that
// keeps the output open cannot hold the call past the command's own exit
// (an exit 0 counts then, as ExecRunner counts it) or past ctx. A command
// that cannot be started, exits non-zero, or is killed when ctx ends is an
// error that names argv[0] and holds the kept output, both escaped for a
// terminal (PrintableInboxText) on one line.
func runDirectorLaunchCmd(ctx context.Context, argv []string, dir string, waitDelay time.Duration) error {
	if len(argv) == 0 {
		return errors.New("launch_cmd: no command to run")
	}
	if waitDelay <= 0 {
		waitDelay = defaultDirectorLaunchWaitDelay
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	out := &directorTailBuffer{limit: directorLaunchOutputLimit}
	cmd.Stdout, cmd.Stderr = out, out
	cmd.WaitDelay = waitDelay
	err := cmd.Run()
	if errors.Is(err, exec.ErrWaitDelay) && ctx.Err() == nil {
		err = nil
	}
	if err == nil {
		return nil
	}
	name := PrintableInboxText(argv[0], false)
	output := ""
	if text := strings.TrimSpace(string(out.b)); text != "" {
		output = "; its output ended with: " + PrintableInboxText(text, false)
	}
	switch {
	case ctx.Err() != nil:
		return fmt.Errorf("launch_cmd %s did not return in time and was killed (%v)%s", name, context.Cause(ctx), output)
	case cmd.ProcessState == nil:
		return fmt.Errorf("launch_cmd %s could not be run: %w%s", name, err, output)
	}
	return fmt.Errorf("launch_cmd %s failed: %w%s", name, err, output)
}
