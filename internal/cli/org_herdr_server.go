package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// The herdr server gate of the CLI (plan 2026-11-06-org-herdr-autostart, S3;
// spec docs/specs/2026-11-06-org-herdr-autostart-panes.md, FR-A1 to FR-A5):
// the Org.HerdrServerGate that `ralph org start` (both forms), `ralph org
// spawn`, and `ralph org director start` set (wireOrgHerdrServerGate). It
// reads `herdr status server --json` and, when no herdr server runs, starts
// `herdr server` detached, as the cross-org watch is started
// (startDetachedProcess), waits for it to answer, and says so on stderr. When
// it cannot, it refuses the verb, which then makes nothing (the org layer
// calls the gate before it writes anything or asks herdr to make anything).

// orgHerdrLaunchLogFileName is the file in the ledger that the stdout and
// stderr of a `herdr server` ralph started are appended to (plan D5). It is
// not herdr's own log, herdr-server.log.
const orgHerdrLaunchLogFileName = "herdr-launch.log"

// orgHerdrServerHelp is the paragraph the --help of `ralph org spawn`,
// `ralph org start`, and `ralph org director start` print about the gate
// (plan AC7), after a lead sentence of their own that says when they run it.
const orgHerdrServerHelp = "When `herdr status server --json` says the server is not running, ralph\n" +
	"starts `herdr server` in the background, in your home directory, with\n" +
	"this command's environment less the herdr pane variables (HERDR_PANE_ID,\n" +
	"HERDR_TAB_ID, HERDR_WORKSPACE_ID, HERDR_ENV) and Claude Code's session\n" +
	"variables (CLAUDECODE, CLAUDE_CODE_*), waits up to 10s for it to answer,\n" +
	"and says so in one line on stderr with its pid, its socket, and\n" +
	"<state dir>/herdr-launch.log, where its output goes; run `herdr` in\n" +
	"another terminal to see the panes and answer the first prompts. When the\n" +
	"status cannot be read, herdr is not on PATH, or the server does not\n" +
	"answer within the 10s, the command is refused with nothing made: start\n" +
	"herdr in another terminal with `herdr`, then run the command again. A\n" +
	"herdr that has no `herdr status server --json` (herdr 0.7.5 has it) gets\n" +
	"one warning line on stderr and is neither checked nor started."

// orgHerdrServerGateOverride, when not nil, is the Org.HerdrServerGate that
// wireOrgHerdrServerGate sets in place of the one that reads the status and
// starts `herdr server`. nil in production. TestMain sets one that always
// passes, so the tests of spawn, start, and director start need no herdr
// server and start none; a test of the gate's wiring sets its own.
var orgHerdrServerGateOverride func() error

// orgHerdrServerStatus reads the herdr server's status: in production a new
// driver.Herdr on driver.ExecRunner, which runs `herdr status server
// --json`. A test of the gate sets its own.
var orgHerdrServerStatus = func(ctx context.Context) (driver.HerdrServerStatus, error) {
	return driver.Herdr{R: driver.ExecRunner{}}.ServerStatus(ctx)
}

// orgHerdrLookPath finds the herdr binary the gate starts as `herdr server`.
// exec.LookPath in production.
var orgHerdrLookPath = exec.LookPath

// orgHerdrServerLaunch starts `herdr server`: startDetachedProcess in
// production. TestMain replaces it with one that panics, so no test starts a
// real herdr server; a test that starts one sets a fake.
var orgHerdrServerLaunch = startDetachedProcess

// How long the gate waits; package vars so tests can shorten them.
var (
	// orgHerdrStatusTimeout bounds each read of the status.
	orgHerdrStatusTimeout = 5 * time.Second
	// orgHerdrServerWaitDeadline is how long the gate waits for the server
	// it started to answer.
	orgHerdrServerWaitDeadline = 10 * time.Second
	// orgHerdrServerPollInterval is how often it reads the status meanwhile.
	orgHerdrServerPollInterval = 100 * time.Millisecond
)

// wireOrgHerdrServerGate sets rt.HerdrServerGate for verb, the words after
// "ralph org" ("start", "spawn", "director start"): the gate
// (ensureOrgHerdrServer) of the ledger at resolvedStateDir, which writes its
// lines on cmd's stderr, or orgHerdrServerGateOverride when that is set. The
// org layer decides when it runs: Spawn in real mode only, StartFeature
// before the worktree, and StartDirector for a start in herdr only.
func wireOrgHerdrServerGate(cmd *cobra.Command, rt *org.Org, resolvedStateDir, verb string) {
	if orgHerdrServerGateOverride != nil {
		rt.HerdrServerGate = orgHerdrServerGateOverride
		return
	}
	rt.HerdrServerGate = func() error {
		return ensureOrgHerdrServer(verb, resolvedStateDir, cmd.ErrOrStderr())
	}
}

// orgHerdrServerGate is one run of the gate: the verb its lines and errors
// name, and where its lines go.
type orgHerdrServerGate struct {
	verb   string
	stderr io.Writer
}

// ensureOrgHerdrServer makes sure a herdr server answers before verb asks
// herdr to make anything. Every read of the status is bounded by
// orgHerdrStatusTimeout.
//
//   - A running server passes, whatever `compatible` says (a server of
//     another version fails the later herdr calls, as before).
//   - A herdr that has no `herdr status server --json`
//     (driver.ErrHerdrStatusJSONUnsupported) gets one warning line on stderr
//     and passes: ralph can neither check nor start its server (plan D3).
//   - A status that cannot be read for any other reason fails, and nothing is
//     started (plan D3): ralph cannot tell whether a server runs.
//   - With no server running, herdr is looked up on PATH and started as
//     `herdr server` (orgHerdrServerLaunchRequest), and the gate waits for it
//     to answer (await).
//
// A launch that fails passes when the status read right after it says a
// server runs (another ralph started one; spec FR-A2), and fails otherwise.
func ensureOrgHerdrServer(verb, stateDir string, stderr io.Writer) error {
	g := orgHerdrServerGate{verb: verb, stderr: stderr}
	st, err := readOrgHerdrServerStatusWithin(orgHerdrStatusTimeout)
	switch {
	case err == nil && st.Running:
		return nil
	case errors.Is(err, driver.ErrHerdrStatusJSONUnsupported):
		_, _ = fmt.Fprintf(stderr, "warning: this herdr has no 'herdr status server --json' (herdr 0.7.5 has it), "+
			"so ralph org %s cannot check or start the herdr server and goes on as before\n", verb)
		return nil
	case errors.Is(err, exec.ErrNotFound):
		return g.notOnPATH(err)
	case err != nil:
		return fmt.Errorf("org: %s: could not read the herdr server's status (%w), so nothing was made; %s", verb, err, g.next())
	}
	path, err := orgHerdrLookPath("herdr")
	if err != nil {
		return g.notOnPATH(err)
	}
	req := orgHerdrServerLaunchRequest(stateDir)
	child, err := orgHerdrServerLaunch(path, req)
	if err != nil {
		st, rerr := readOrgHerdrServerStatusWithin(orgHerdrStatusTimeout)
		switch {
		case rerr == nil && st.Running:
			return nil
		case rerr != nil:
			return fmt.Errorf("org: %s: could not start the herdr server (%w), and its status could not be read after that (%v), so nothing was made; %s",
				verb, err, rerr, g.next())
		}
		return fmt.Errorf("org: %s: could not start the herdr server (%w), so nothing was made; %s", verb, err, g.next())
	}
	return g.await(child, req.logPath)
}

// await reads the status every orgHerdrServerPollInterval, each read bounded
// by orgHerdrStatusTimeout but never past the deadline, for up to
// orgHerdrServerWaitDeadline. A server that answers passes, with the stderr
// line (announce) when child is still running. When child exits first, the
// status is read once more: a running server passes with no line (another
// ralph's server answered, and herdr ends a second server with "already
// running"; plan D4), and anything else fails with child's pid and the log.
// A read that fails is not fatal by itself; at the deadline the gate fails,
// saying what the last read found. No read starts once the deadline has
// passed, so the last one is never cut short to nothing. child is not
// stopped then: a server that answers late does no harm.
func (g orgHerdrServerGate) await(child *orgWatchProcess, logPath string) error {
	deadline := time.Now().Add(orgHerdrServerWaitDeadline)
	var lastErr error
	for left := time.Until(deadline); left > 0; left = time.Until(deadline) {
		st, err := readOrgHerdrServerStatusWithin(min(orgHerdrStatusTimeout, left))
		if err == nil && st.Running {
			if !orgHerdrChildExited(child.exited) {
				g.announce(child.pid, st.Socket, logPath)
			}
			return nil
		}
		lastErr = err
		if left = time.Until(deadline); left <= 0 {
			break
		}
		select {
		case <-child.exited:
			return g.afterExit(child.pid, logPath)
		case <-time.After(min(orgHerdrServerPollInterval, left)):
		}
	}
	last := "the last read of its status said it was not running"
	if lastErr != nil {
		last = "the last read of its status failed: " + lastErr.Error()
	}
	return fmt.Errorf("org: %s: the herdr server (pid %d) did not answer within %s (%s), so nothing was made; see %s, or %s",
		g.verb, child.pid, orgHerdrServerWaitDeadline, last, logPath, g.next())
}

// afterExit is await's read of the status once the server it started has
// exited before it answered.
func (g orgHerdrServerGate) afterExit(pid int, logPath string) error {
	st, err := readOrgHerdrServerStatusWithin(orgHerdrStatusTimeout)
	switch {
	case err == nil && st.Running:
		return nil
	case err != nil:
		return fmt.Errorf("org: %s: the herdr server (pid %d) exited before it answered, and its status could not be read after that (%v), so nothing was made; see %s, or %s",
			g.verb, pid, err, logPath, g.next())
	}
	return fmt.Errorf("org: %s: the herdr server (pid %d) exited before it answered, so nothing was made; see %s, or %s",
		g.verb, pid, logPath, g.next())
}

// announce writes the one stderr line of a server the gate started (spec
// FR-A4).
func (g orgHerdrServerGate) announce(pid int, socket, logPath string) {
	if socket == "" {
		socket = "not reported"
	}
	_, _ = fmt.Fprintf(g.stderr, "ralph org %s: started the herdr server (pid %d, socket %s); run 'herdr' in another terminal to see %s; log: %s\n",
		g.verb, pid, socket, orgHerdrAttachHint(g.verb), logPath)
}

// orgHerdrAttachHint is what `herdr` shows the human after verb started the
// server: the pane of the director for director start, the seats' panes
// otherwise.
func orgHerdrAttachHint(verb string) string {
	if verb == "director start" {
		return "the director's pane and to answer its first prompts"
	}
	return "the seats' panes and to answer the first seat's prompts"
}

// notOnPATH is the gate's error when no herdr binary is on PATH (cause is the
// lookup's error, or the status read's that wraps exec.ErrNotFound).
func (g orgHerdrServerGate) notOnPATH(cause error) error {
	return fmt.Errorf("org: %s: herdr is not on PATH (%w), so ralph cannot start the herdr server, and nothing was made; "+
		"install herdr, or start it in another terminal with 'herdr', then run ralph org %s again", g.verb, cause, g.verb)
}

// next is how to go on after the gate failed.
func (g orgHerdrServerGate) next() string {
	return "start herdr in another terminal with 'herdr', then run ralph org " + g.verb + " again"
}

// readOrgHerdrServerStatusWithin reads the status (orgHerdrServerStatus)
// with a context that ends after d.
func readOrgHerdrServerStatusWithin(d time.Duration) (driver.HerdrServerStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	return orgHerdrServerStatus(ctx)
}

// orgHerdrChildExited reports whether exited is closed, without waiting.
func orgHerdrChildExited(exited <-chan struct{}) bool {
	select {
	case <-exited:
		return true
	default:
		return false
	}
}

// orgHerdrServerLaunchRequest is the `herdr server` the gate starts (spec
// FR-A3): the argument server, in the user's home directory ("" when it
// cannot be found keeps this process's), with orgHerdrServerEnv of this
// process's environment, its output appended to herdr-launch.log of the
// ledger at stateDir.
func orgHerdrServerLaunchRequest(stateDir string) orgWatchLaunchRequest {
	dir, err := os.UserHomeDir()
	if err != nil {
		// No home directory: the server runs where this command runs.
		dir = ""
	}
	return orgWatchLaunchRequest{
		args:    []string{"server"},
		dir:     dir,
		env:     orgHerdrServerEnv(os.Environ(), dir),
		logPath: filepath.Join(stateDir, orgHerdrLaunchLogFileName),
	}
}

// orgHerdrServerEnv is environ less the variables orgHerdrServerDropsEnv
// names, with PWD set to dir when dir is not "", as os/exec sets it for a nil
// Env. Nothing else changes, so XDG_CONFIG_HOME, XDG_STATE_HOME, and
// HERDR_SOCKET_PATH (an isolated herdr; docs/recipes/isolated-herdr-server.md)
// and the CODEX_ variables (CODEX_HOME, which the codex seat's model
// observation reads) reach the server.
func orgHerdrServerEnv(environ []string, dir string) []string {
	env := make([]string, 0, len(environ)+1)
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		if orgHerdrServerDropsEnv(name) || (dir != "" && name == "PWD") {
			continue
		}
		env = append(env, kv)
	}
	if dir != "" {
		env = append(env, "PWD="+dir)
	}
	return env
}

// orgHerdrServerDropsEnv reports whether the `herdr server` the gate starts
// does not inherit the variable name (spec FR-A3). The server's environment
// reaches every pane it makes, so it gets neither the herdr pane variables of
// the command that starts it (orgWatchChildDroppedEnv: a leader's or a
// director's pane, tab, and workspace) nor the variables of a Claude Code
// session (CLAUDECODE and every name that starts with CLAUDE_CODE_, such as
// CLAUDE_CODE_MESSAGING_TOKEN), which would reach every seat. A name such as
// CLAUDE_CONFIG_DIR, which does not start with CLAUDE_CODE_, stays.
func orgHerdrServerDropsEnv(name string) bool {
	return slices.Contains(orgWatchChildDroppedEnv, name) || name == "CLAUDECODE" || strings.HasPrefix(name, "CLAUDE_CODE_")
}
