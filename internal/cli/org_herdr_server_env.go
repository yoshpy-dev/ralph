package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// The environment of the herdr server, for moving an org's seats to another
// Claude account (spec docs/specs/2026-11-20-org-reseat-restart-herdr.md,
// FR-5 to FR-7; plan 2026-11-20-org-reseat-restart-herdr, S1). A pane herdr
// makes takes the server's environment, so a claude seat runs with the
// server's CLAUDE_CONFIG_DIR. This file holds what a verb changes in the
// environment of the `herdr server` the gate starts
// (orgHerdrServerEnvOverride), the check of a --claude-config-dir value
// (checkOrgClaudeConfigDir), and the read of the CLAUDE_CONFIG_DIR of the
// server that runs (readOrgHerdrServerClaudeConfigDir).

// orgClaudeConfigDirName is the variable that names the config directory of
// a Claude Code account.
const orgClaudeConfigDirName = "CLAUDE_CONFIG_DIR"

// orgHerdrServerEnvOverride is what a verb changes in the environment of the
// `herdr server` the gate starts (spec FR-5). The zero value changes
// nothing: the server gets orgHerdrServerEnv of this process's environment,
// with this process's CLAUDE_CONFIG_DIR when it has one, as before. With
// setClaudeConfigDir, the server's CLAUDE_CONFIG_DIR is claudeConfigDir in
// place of this process's, or, when claudeConfigDir is "", the server has
// none, so its claude seats use the default account (spec
// 2026-11-12-org-reseat, D6 (1)). claudeConfigDir is meant to be a value
// checkOrgClaudeConfigDir returned.
type orgHerdrServerEnvOverride struct {
	setClaudeConfigDir bool
	claudeConfigDir    string
}

// apply returns env with o applied: env itself for an o that does not set
// CLAUDE_CONFIG_DIR, and otherwise a new slice, env less every
// CLAUDE_CONFIG_DIR entry, the other entries in their order, followed by
// CLAUDE_CONFIG_DIR=<claudeConfigDir> when claudeConfigDir is not "". env
// itself is not changed.
func (o orgHerdrServerEnvOverride) apply(env []string) []string {
	if !o.setClaudeConfigDir {
		return env
	}
	out := slices.DeleteFunc(slices.Clone(env), func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		return name == orgClaudeConfigDirName
	})
	if o.claudeConfigDir != "" {
		out = append(out, orgClaudeConfigDirName+"="+o.claudeConfigDir)
	}
	return out
}

// checkOrgClaudeConfigDir checks value, given to --claude-config-dir, and
// returns what the server's CLAUDE_CONFIG_DIR is to be (spec FR-5): "" for
// "", which removes it, and otherwise value made absolute, since the server
// runs in the home directory and a command that repeats the value may run
// elsewhere. A value that is not "" must name an existing directory
// (os.Stat, which follows a symlink to one). The error says what is wrong,
// naming value, for the caller's refusal.
func checkOrgClaudeConfigDir(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("--claude-config-dir %q could not be made absolute: %w", value, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("--claude-config-dir %q is not an existing directory: %w", value, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("--claude-config-dir %q is not a directory", value)
	}
	return abs, nil
}

// orgHerdrServerEnvState is what a read of the running server's
// CLAUDE_CONFIG_DIR found.
type orgHerdrServerEnvState int

const (
	// orgHerdrServerEnvUnreadable is a read that could not tell. It is the
	// zero value, so a result nobody filled in never reads as unset.
	orgHerdrServerEnvUnreadable orgHerdrServerEnvState = iota
	// orgHerdrServerEnvSet is a server whose environment has
	// CLAUDE_CONFIG_DIR.
	orgHerdrServerEnvSet
	// orgHerdrServerEnvUnset is a server whose environment has no
	// CLAUDE_CONFIG_DIR.
	orgHerdrServerEnvUnset
)

// orgHerdrServerClaudeConfigDir is what readOrgHerdrServerClaudeConfigDir
// read: the state, the value for orgHerdrServerEnvSet, and, for
// orgHerdrServerEnvUnreadable, the reason, which says what the read could
// not do and never holds the environment it read.
type orgHerdrServerClaudeConfigDir struct {
	state  orgHerdrServerEnvState
	value  string
	reason string
}

// orgHerdrServerEnvProbe is how readOrgHerdrServerClaudeConfigDir reaches
// the system: production's is orgHerdrServerEnvProbeNow, and a test gives
// its own.
type orgHerdrServerEnvProbe struct {
	// status reads `herdr status server --json`.
	status func(ctx context.Context) (driver.HerdrServerStatus, error)
	// run runs lsof and ps and returns their stdout.
	run func(ctx context.Context, name string, args ...string) (string, error)
	// readFile reads /proc/<pid>/environ, on Linux.
	readFile func(name string) ([]byte, error)
	// goos is runtime.GOOS: "linux" reads /proc, any other ps.
	goos string
}

// orgHerdrServerEnvProbeNow is the probe of production: the gate's status
// read (orgHerdrServerStatus), runOrgHerdrServerEnvCommand, os.ReadFile, and
// runtime.GOOS.
func orgHerdrServerEnvProbeNow() orgHerdrServerEnvProbe {
	return orgHerdrServerEnvProbe{status: orgHerdrServerStatus, run: runOrgHerdrServerEnvCommand, readFile: os.ReadFile, goos: runtime.GOOS}
}

// orgHerdrServerClaudeConfigDirReader reads the CLAUDE_CONFIG_DIR of the
// herdr server that runs: readOrgHerdrServerClaudeConfigDir with
// orgHerdrServerEnvProbeNow in production. A test of a verb that reads it
// sets its own, so no test runs lsof or ps on the machine's herdr server.
var orgHerdrServerClaudeConfigDirReader = func(ctx context.Context) orgHerdrServerClaudeConfigDir {
	return readOrgHerdrServerClaudeConfigDir(ctx, orgHerdrServerEnvProbeNow())
}

// orgHerdrServerEnvCallTimeout bounds each lsof and ps the read runs, within
// the caller's ctx (spec NFR-3); the status read is bounded by
// orgHerdrStatusTimeout. A var so a test can shorten it.
var orgHerdrServerEnvCallTimeout = 5 * time.Second

// readOrgHerdrServerClaudeConfigDir reads the CLAUDE_CONFIG_DIR of the herdr
// server that runs (spec FR-7) through p, each call bounded:
//
//  1. the server's socket, from `herdr status server --json` (p.status,
//     within orgHerdrStatusTimeout); a status that cannot be read, a server
//     that does not run, and a status with no socket cannot be read;
//  2. the server's pid, with lsof and ps (orgHerdrServerPID);
//  3. its environment: on Linux /proc/<pid>/environ (orgEnvironValue),
//     elsewhere `ps eww -o command= -p <pid>` (orgPSEnvValue). An
//     environment with no entry at all cannot be read: ps eww prints none for
//     a process whose environment it may not read, which must not pass as
//     unset.
//
// A failure is never an error, only a result that cannot be read, with the
// reason: the reader is for display, and a verb goes on without it (spec
// D4).
func readOrgHerdrServerClaudeConfigDir(ctx context.Context, p orgHerdrServerEnvProbe) orgHerdrServerClaudeConfigDir {
	statusCtx, cancel := context.WithTimeout(ctx, orgHerdrStatusTimeout)
	st, err := p.status(statusCtx)
	cancel()
	switch {
	case err != nil:
		return orgHerdrServerEnvCannotRead("could not read the herdr server's status: " + err.Error())
	case !st.Running:
		return orgHerdrServerEnvCannotRead("no herdr server is running")
	case st.Socket == "":
		return orgHerdrServerEnvCannotRead("'herdr status server --json' reported no socket")
	}
	pid, why := orgHerdrServerPID(ctx, p, st.Socket)
	if why != "" {
		return orgHerdrServerEnvCannotRead(why)
	}
	var value string
	var found, anyEnv bool
	if p.goos == "linux" {
		path := "/proc/" + strconv.Itoa(pid) + "/environ"
		data, err := p.readFile(path)
		if err != nil {
			return orgHerdrServerEnvCannotRead(fmt.Sprintf("could not read the environment of the herdr server (pid %d): %v", pid, err))
		}
		value, found, anyEnv = orgEnvironValue(data, orgClaudeConfigDirName)
	} else {
		out, err := p.call(ctx, "ps", "eww", "-o", "command=", "-p", strconv.Itoa(pid))
		if err != nil {
			return orgHerdrServerEnvCannotRead(fmt.Sprintf("could not read the environment of the herdr server (pid %d) with ps: %v", pid, err))
		}
		value, found, anyEnv = orgPSEnvValue(out, orgClaudeConfigDirName)
	}
	switch {
	case found:
		return orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvSet, value: value}
	case !anyEnv:
		return orgHerdrServerEnvCannotRead(fmt.Sprintf("the herdr server's environment (pid %d) showed no variable", pid))
	}
	return orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvUnset}
}

// orgHerdrServerEnvCannotRead is a result that cannot be read, for reason.
func orgHerdrServerEnvCannotRead(reason string) orgHerdrServerClaudeConfigDir {
	return orgHerdrServerClaudeConfigDir{state: orgHerdrServerEnvUnreadable, reason: reason}
}

// call runs name with args through p.run, bounded by
// orgHerdrServerEnvCallTimeout within ctx.
func (p orgHerdrServerEnvProbe) call(ctx context.Context, name string, args ...string) (string, error) {
	callCtx, cancel := context.WithTimeout(ctx, orgHerdrServerEnvCallTimeout)
	defer cancel()
	return p.run(callCtx, name, args...)
}

// orgHerdrServerPID finds the pid of the herdr server whose socket is
// socket, with `lsof -t <socket>`. One pid is the server's. With more than
// one, it keeps those whose command, `ps -o comm= -p <pid>`, is herdr (the
// last element of the path macOS prints, or the name Linux prints), and the
// one left is the server's. why says why there is none: lsof failed or
// listed no pid, a line it printed is not a pid, a ps failed, or not exactly
// one of the pids is herdr.
func orgHerdrServerPID(ctx context.Context, p orgHerdrServerEnvProbe, socket string) (pid int, why string) {
	out, err := p.call(ctx, "lsof", "-t", socket)
	if err != nil {
		return 0, fmt.Sprintf("could not find the herdr server's pid: lsof -t on the socket %q failed: %v", socket, err)
	}
	var pids []int
	for _, word := range strings.Fields(out) {
		n, err := strconv.Atoi(word)
		if err != nil || n <= 0 {
			return 0, fmt.Sprintf("could not find the herdr server's pid: lsof -t on the socket %q printed %q, which is not a pid", socket, word)
		}
		if !slices.Contains(pids, n) {
			pids = append(pids, n)
		}
	}
	switch len(pids) {
	case 0:
		return 0, fmt.Sprintf("could not find the herdr server's pid: lsof -t on the socket %q listed no process", socket)
	case 1:
		return pids[0], ""
	}
	var herdrs []int
	for _, n := range pids {
		comm, err := p.call(ctx, "ps", "-o", "comm=", "-p", strconv.Itoa(n))
		if err != nil {
			return 0, fmt.Sprintf("could not find the herdr server's pid: lsof -t on the socket %q listed pids %s, and ps -o comm= -p %d failed: %v",
				socket, orgJoinPIDs(pids), n, err)
		}
		comm = strings.TrimSpace(comm)
		if comm[strings.LastIndex(comm, "/")+1:] == "herdr" {
			herdrs = append(herdrs, n)
		}
	}
	if len(herdrs) != 1 {
		return 0, fmt.Sprintf("could not find the herdr server's pid: lsof -t on the socket %q listed pids %s, and %d of them are herdr, not one",
			socket, orgJoinPIDs(pids), len(herdrs))
	}
	return herdrs[0], ""
}

// orgJoinPIDs is pids as "1, 2, 3".
func orgJoinPIDs(pids []int) string {
	words := make([]string, len(pids))
	for i, n := range pids {
		words[i] = strconv.Itoa(n)
	}
	return strings.Join(words, ", ")
}

// orgPSEnvKey matches the start of one entry of the environment that
// `ps eww` prints after a process's command: a space, a name
// ([A-Za-z_][A-Za-z0-9_]*), and "=".
var orgPSEnvKey = regexp.MustCompile(` [A-Za-z_][A-Za-z0-9_]*=`)

// orgPSEnvValue reads name's value from out, what `ps eww -o command=`
// printed for one process: its command, then its environment, one entry
// after each space. ps quotes nothing, so a space inside a value looks like
// the space before the next entry: the value runs from after " name=" up to
// the next " KEY=" (orgPSEnvKey), or to the end of the line (spec FR-7). The
// first entry named name counts. found is false when no entry is name, and
// anyEnv says whether out had any entry.
func orgPSEnvValue(out, name string) (value string, found, anyEnv bool) {
	line := strings.TrimRight(out, "\r\n")
	locs := orgPSEnvKey.FindAllStringIndex(line, -1)
	for i, loc := range locs {
		if line[loc[0]+1:loc[1]-1] != name {
			continue
		}
		end := len(line)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		return line[loc[1]:end], true, true
	}
	return "", false, len(locs) > 0
}

// orgEnvironValue reads name's value from data, a process's
// /proc/<pid>/environ: KEY=VALUE entries, each ended by a NUL byte. The first
// entry named name counts. found is false when no entry is name, and anyEnv
// says whether data had any entry.
func orgEnvironValue(data []byte, name string) (value string, found, anyEnv bool) {
	for _, kv := range strings.Split(string(data), "\x00") {
		if kv == "" {
			continue
		}
		if v, ok := strings.CutPrefix(kv, name+"="); ok {
			return v, true, true
		}
		anyEnv = true
	}
	return "", false, anyEnv
}

// orgHerdrServerEnvCommandEnv is the whole environment the read runs lsof
// and ps with: PATH and LC_ALL=C, as ReadDirectorProcess runs ps
// (directorPSEnv in internal/org/director_cmd.go), so no PS_FORMAT,
// PS_PERSONALITY, or locale of the caller changes what they print.
func orgHerdrServerEnvCommandEnv() []string {
	return []string{"PATH=" + os.Getenv("PATH"), "LC_ALL=C"}
}

// orgHerdrServerEnvCommandWaitDelay is the exec.Cmd.WaitDelay of
// runOrgHerdrServerEnvCommand.
const orgHerdrServerEnvCommandWaitDelay = time.Second

// runOrgHerdrServerEnvCommand runs name with args under
// orgHerdrServerEnvCommandEnv, bounded by ctx, and returns its stdout as it
// printed it. A run that ctx ended is an error that says so and wraps the
// cause; any other failure is an error with name, the exit, and stderr with
// its white space folded. The stdout of a failed run, which for ps eww may
// hold an environment, is never in the error.
func runOrgHerdrServerEnvCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = orgHerdrServerEnvCommandEnv()
	cmd.WaitDelay = orgHerdrServerEnvCommandWaitDelay
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	switch {
	case ctx.Err() != nil:
		return "", fmt.Errorf("%s did not answer: %w", name, context.Cause(ctx))
	case err != nil:
		if text := strings.Join(strings.Fields(stderr.String()), " "); text != "" {
			return "", fmt.Errorf("%s: %w: %s", name, err, text)
		}
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return stdout.String(), nil
}
