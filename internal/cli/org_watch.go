package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
	"github.com/yoshpy-dev/ralph/internal/org/driver"
)

// newOrgWatchCmd wires `ralph org watch` (PR④ pulse layer, AC-3/3b/3c/4/5;
// the cross-org watch of docs/plans/active/2026-10-11-org-cross-watch.md):
// one process watches every org of the ledger. `--all` runs
// (*org.Org).RunWatchAll in the foreground (runOrgWatchAll), `--all
// --detach` starts that as a detached process (runOrgWatchDetach), and
// `--stop` asks a running one to stop (runOrgWatchStop). All condition
// evaluation, ALERT dedupe, deadman escalation, inbox deadline, director
// check and its director-alive.json, director wake, lock, and liveness mark
// logic lives in internal/org (watch.go, deadline.go, director_check.go,
// director_wake.go, watchalive.go) -- this command checks the
// flags, resolves the state directory (the one the manifest, receipts, and
// inbox live in, via org.ResolveOrgStateDir) and the directory wake_cmd runs
// in (orgWatchDirectorWake), and wires them through to org.WatchAllParams.
//
// The single-org watch (`--org-id`) is retired and refused: next to `--all`
// it would make two processes watch the same org and write its
// watch-status file.
func newOrgWatchCmd(orgID, stateDir, configPath *string) *cobra.Command {
	var (
		intervalSeconds         int
		all, once, detach, stop bool
	)

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch every org of the ledger: inbox deadlines, director check, director wake, pulse, liveness mark",
		Long: "ralph org watch --all watches every org of the ledger in one process.\n" +
			"Every --interval-seconds (default: [org.watchdog].interval_seconds) it\n" +
			"sends the inbox items past their [org.director] ack_minutes or\n" +
			"resolve_minutes deadline to the human. Then it checks the current\n" +
			"headless director (the one ralph org director start started) with herdr\n" +
			"agent get or ps, and writes what it found (alive, ended, or unknown, and\n" +
			"how it found that) to <state-dir>/director-alive.json; for that it\n" +
			"closes nothing and writes nothing to director.jsonl. Then, when the\n" +
			"attached director was registered with a wake that includes push\n" +
			"([org.director].wake, or director attach --wake), it wakes that\n" +
			"director for the new open inbox items, through [org.director].wake_cmd\n" +
			"(run without a shell, in the directory of the ralph.toml it was read\n" +
			"from) or, with no wake_cmd, by typing into the herdr pane of a headless\n" +
			"director in herdr. It does not wake a headless director the check of\n" +
			"the same cycle found ended.\n" +
			"Without --config, when the ledger is the main worktree's, wake and\n" +
			"wake_cmd come from the main worktree's ralph.toml, and wake_cmd runs in\n" +
			"its root. It reads wake_cmd when it starts, so restart it after changing\n" +
			"wake_cmd.\n" +
			"Then it runs the pulse cycle of each running org: heartbeat-stall /\n" +
			"process-liveness / worktree-scope-change ALERTs sent to the org's\n" +
			"leader, and a deadman escalation\n" +
			"(<state-dir>/escalations.jsonl + stderr banner + best-effort darwin\n" +
			"notification) when the leader does not respond within\n" +
			"[org].deadman_minutes. It holds <state-dir>/watch.lock and rewrites its\n" +
			"liveness mark <state-dir>/watch-alive.json every cycle; a second watch\n" +
			"on the same ledger is refused. SIGINT or SIGTERM ends it after the\n" +
			"current cycle.\n\n" +
			"  --all --once    run exactly one cycle and exit (useful for cron/smoke)\n" +
			"  --all --detach  start the watch as a detached process that logs to\n" +
			"                  <state-dir>/watch.log, and return once it is running;\n" +
			"                  it runs in the ledger's main worktree root (when it has\n" +
			"                  one), without HERDR_PANE_ID, HERDR_TAB_ID,\n" +
			"                  HERDR_WORKSPACE_ID, and HERDR_ENV. It exits 0 when a\n" +
			"                  watch already runs, and 1 when a stale, stopping, or\n" +
			"                  --once one holds the lock\n" +
			"  --stop          ask the running watch to stop: write\n" +
			"                  <state-dir>/watch-stop.json (no signal is sent) and\n" +
			"                  wait up to 30s for it to end\n\n" +
			"--org-id is refused: the single-org watch is retired.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkOrgWatchFlags(cmd, all, once, detach, stop); err != nil {
				return err
			}
			// Resolved exactly once (self-review LOW fix) and passed through
			// to newOrgRuntimeAt instead of also re-resolving inside a second
			// newOrgRuntime call. stateDirSource (tech-debt: "watchdog
			// deferred LOW (1)") is surfaced in the startup banner so an
			// operator can tell which precedence tier (flag/env/
			// git-main-worktree/git-toplevel/cwd) produced resolvedStateDir
			// without re-deriving ResolveOrgStateDir's logic by hand.
			resolvedStateDir, stateDirSource := org.ResolveOrgStateDir(*stateDir, cmd.Flags().Changed("state-dir"))
			if err := guardLegacyOrgStateDir(cmd, resolvedStateDir, stateDirSource, orgLedgerMutating); err != nil {
				return err
			}
			if stop {
				return runOrgWatchStop(cmd, resolvedStateDir)
			}
			rt, err := newOrgRuntimeAt(resolvedStateDir, *configPath)
			if err != nil {
				return err
			}
			// Before --detach too, so a main worktree ralph.toml that does
			// not load is this command's error, not only a line in the
			// child's watch.log. The child computes its own directory.
			wakeDir, err := orgWatchDirectorWake(rt, *configPath, resolvedStateDir, stateDirSource)
			if err != nil {
				return err
			}
			if detach {
				return runOrgWatchDetach(cmd, rt, resolvedStateDir, stateDirSource, *configPath, intervalSeconds)
			}
			return runOrgWatchAll(cmd, rt, resolvedStateDir, stateDirSource, wakeDir, intervalSeconds, once)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "watch every org of the ledger (required, except with --stop)")
	cmd.Flags().IntVar(&intervalSeconds, "interval-seconds", 0, "pulse cycle interval in seconds (default: [org.watchdog].interval_seconds)")
	cmd.Flags().BoolVar(&once, "once", false, "run exactly one cycle and exit")
	cmd.Flags().BoolVar(&detach, "detach", false, "start the watch as a detached process logging to <state-dir>/watch.log, and return once it is running")
	cmd.Flags().BoolVar(&stop, "stop", false, "ask the running watch to stop, without a signal, and wait for it to end")

	return cmd
}

// checkOrgWatchFlags refuses the flag combinations `ralph org watch` does
// not run, before any state dir is resolved, so a refusal writes nothing.
// --org-id is refused the way the inbox verbs refuse it (newOrgInboxRuntime).
func checkOrgWatchFlags(cmd *cobra.Command, all, once, detach, stop bool) error {
	if cmd.Flags().Changed("org-id") {
		return fmt.Errorf("org: watch: --org-id does not apply: ralph org watch watches every org of the ledger; drop --org-id and pass --all")
	}
	if stop {
		switch {
		case detach:
			return fmt.Errorf("org: watch: --stop does not take --detach")
		case once:
			return fmt.Errorf("org: watch: --stop does not take --once")
		case cmd.Flags().Changed("interval-seconds"):
			return fmt.Errorf("org: watch: --stop does not take --interval-seconds")
		}
		return nil
	}
	if !all {
		return fmt.Errorf("org: watch: pass --all: ralph org watch watches every org of the ledger")
	}
	if detach && once {
		return fmt.Errorf("org: watch: --detach does not take --once: a detached watch runs until ralph org watch --stop")
	}
	return nil
}

// orgWatchDirectorWake sets rt.Config's Director.Wake and Director.WakeCmd
// ([org.director].wake and wake_cmd) by the main-worktree rule
// (withMainWorktreeDirectorWakeConfigFrom), so a feature worktree's
// ralph.toml, which a leader can change, does not choose what the watch
// runs, and returns the directory wake_cmd runs in, the directory of the
// ralph.toml it was read from (plan 2026-10-18-org-director-wake, D9 and
// AC4): the main worktree root when the rule applied, else the absolute
// directory of --config when one was given, else the absolute working
// directory, whose ./ralph.toml newOrgRuntimeAt read. Never the working
// directory when the rule applied, so a watch started in a feature worktree
// does not run a script of the same name there. A main worktree ralph.toml
// that does not load is the error.
func orgWatchDirectorWake(rt *org.Org, configPath, resolvedStateDir, stateDirSource string) (string, error) {
	cfg, mainToml, err := withMainWorktreeDirectorWakeConfigFrom(rt.Config, configPath, resolvedStateDir, stateDirSource)
	if err != nil {
		return "", err
	}
	rt.Config = cfg
	switch {
	case mainToml != "":
		return filepath.Dir(mainToml), nil
	case configPath != "":
		abs, err := filepath.Abs(configPath)
		if err != nil {
			return "", fmt.Errorf("org: watch: make --config %s absolute for [org.director].wake_cmd: %w", configPath, err)
		}
		return filepath.Dir(abs), nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("org: watch: read the working directory for [org.director].wake_cmd: %w", err)
	}
	return wd, nil
}

// orgWatchLockWaitOverride and orgWatchStopPollIntervalOverride are copied
// into WatchAllParams.LockWait and StopPollInterval (runOrgWatchAll). Zero
// in production, which means RunWatchAll's own defaults (2s and 1s);
// TestMain shortens both.
var (
	orgWatchLockWaitOverride         time.Duration
	orgWatchStopPollIntervalOverride time.Duration
)

// runOrgWatchAll is `ralph org watch --all` in the foreground. SIGINT and
// SIGTERM cancel its context, so RunWatchAll ends after the current cycle
// and writes its stopped mark. Once that context is done, the signals get
// their default effect back, so a second one ends the process at once for a
// cycle that takes long while herdr calls time out (the kernel then frees
// watch.lock, and the mark is left as it was). wakeDir is the directory
// [org.director].wake_cmd runs in (orgWatchDirectorWake).
func runOrgWatchAll(cmd *cobra.Command, rt *org.Org, resolvedStateDir, stateDirSource, wakeDir string, intervalSeconds int, once bool) error {
	ctx, stopSignals := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	context.AfterFunc(ctx, stopSignals)

	cycles := 0
	if once {
		cycles = 1
	}
	// Effective interval (self-review LOW fix): the raw --interval-seconds
	// flag value is 0 by default, so the banner used to print a cadence
	// that was never actually running -- ResolveWatchInterval mirrors
	// RunWatchAll's own fallback chain so the banner reports what will
	// really execute.
	effectiveInterval := org.ResolveWatchInterval(time.Duration(intervalSeconds)*time.Second, rt.Config.Watchdog)
	// The banner is printed once this watch holds watch.lock
	// (WatchAllParams.OnLocked), so a second watch, refused the lock, prints
	// only its refusal (tech-debt 204 (e)).
	banner := func() {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "watching every org (interval=%s once=%t state-dir=%s (source: %s))\n",
			effectiveInterval, once, resolvedStateDir, stateDirSource)
	}
	// The inbox deadline banners name `ralph org inbox show`, with
	// --state-dir when this watch was given one (a --detach child always is).
	rt.InboxHintStateDir = orgInboxHintStateDir(cmd, resolvedStateDir)
	watcherCtx, cancelWatchers := context.WithCancel(ctx)
	defer cancelWatchers()
	hooks, watcherWG := newWatchdogHooks(watcherCtx, rt, cmd.ErrOrStderr())
	// Wait for any still-in-flight on-demand watcher goroutine before the
	// watch ends (self-review M-5 fix): without this, `--once` returned as
	// soon as cycle 1's synchronous pulse evaluation finished, killing the
	// process before an OnSemanticTrigger goroutine it had just started
	// could ever produce a watcher receipt or ALERT. Bounded by RunWatcher's
	// own watcherInvokeTimeout and the ALERT send's DriverCallTimeout
	// (SendWatchdogAlertBounded). The long-running (non-`--once`) mode ends
	// on a signal (ctx done) or on a `--stop` request, which leaves ctx
	// alive (self-review L1), so it cancels watcherCtx first: newWatchdogHooks
	// threads it into the tracked goroutine's calls (self-review cycle-2 M2-2
	// fix), and an in-flight judgment call unwinds at once instead of running
	// out the full 60s bound after the watch has stopped.
	//
	// RunWatchAll calls BeforeStop while it still holds watch.lock and its
	// mark says stopping (cross-review cycle 1 #1; plan
	// 2026-10-12-org-watch-gate F9), so `--stop` and a second `watch --all`
	// see this watch holding the lock until the goroutines have ended,
	// rather than a free lock while they can still send an ALERT or write a
	// receipt. Nothing waits after RunWatchAll returns: it calls BeforeStop
	// on every end after it took the lock, and when it did not take it, no
	// cycle ran, so no goroutine was started.
	hooks.BeforeStop = func() {
		if !once {
			cancelWatchers()
		}
		watcherWG.Wait()
	}
	return rt.RunWatchAll(ctx, org.WatchAllParams{
		StatusDir: resolvedStateDir,
		Interval:  effectiveInterval,
		Cycles:    cycles,
		// The deadman's desktop notification goes through the inbox's
		// override (nil in production, which means osascript on darwin), so
		// a test shows no real notification from either.
		Escalate:         rt.DesktopNotify,
		Stderr:           cmd.ErrOrStderr(),
		LockWait:         orgWatchLockWaitOverride,
		StopPollInterval: orgWatchStopPollIntervalOverride,
		OnLocked:         banner,
		DirectorWakeDir:  wakeDir,
	}, hooks)
}

// orgWatchProcess is a detached watch that --detach or `ralph org start`
// launched: pid, exited (closed once the process has exited and been
// reaped), and stop (ends the process and returns once it has exited). stop
// acts on this process, never on a pid read from the liveness mark.
type orgWatchProcess struct {
	pid    int
	exited <-chan struct{}
	stop   func()
}

// orgWatchLaunchRequest is a detached watch to launch (orgWatchChild): the
// ralph binary with args, its stdout and stderr appended to logPath, run in
// the working directory dir ("" keeps this process's) with the environment
// env (nil keeps this process's).
type orgWatchLaunchRequest struct {
	args    []string
	logPath string
	dir     string
	env     []string
}

// orgWatchLaunch starts the detached watch that --detach and `ralph org
// start` launch (ensureOrgWatch). It is launchOrgWatch in production.
// TestMain replaces it with one that panics, so no test execs the test
// binary; a test that launches a watch sets its own.
var orgWatchLaunch = launchOrgWatch

// How long and how often ensureOrgWatch (--detach and `ralph org start`) and
// --stop wait; package vars so tests can shorten them.
var (
	// orgWatchEnsureDeadline is how long ensureOrgWatch waits, for --detach
	// and start alike, for a stale watch to accept work, and for a watch it
	// launched to.
	orgWatchEnsureDeadline = 10 * time.Second
	// orgWatchReleaseDeadline is how long ensureOrgWatch waits for
	// watch.lock to be free after it stopped a child that was not running in
	// time.
	orgWatchReleaseDeadline = 2 * time.Second
	// orgWatchChildExitWait is how long ensureOrgWatch waits for the child
	// it launched to exit once another watch accepts work: the child's own
	// wait for watch.lock (org.DefaultWatchLockWait), plus a second for it to
	// start and reach that wait.
	orgWatchChildExitWait = org.DefaultWatchLockWait + time.Second
	// orgWatchStopDeadline is how long --stop waits for watch.lock to be
	// free.
	orgWatchStopDeadline = 30 * time.Second
	// orgWatchProbeInterval is how often ensureOrgWatch and --stop look
	// again.
	orgWatchProbeInterval = 100 * time.Millisecond
	// orgWatchChildStopGrace is how long orgWatchProcess.stop waits after
	// SIGTERM, which ends the watch after its current cycle, before SIGKILL.
	orgWatchChildStopGrace = 5 * time.Second
)

// orgWatchChildArgs returns the arguments the detached watch starts the
// ralph binary with: the foreground watch of the resolved state dir, plus
// --config, made absolute from this process's working directory, and
// --interval-seconds when they were given (`ralph org start` has no
// --interval-seconds). The child runs in another working directory
// (orgWatchChild), where a relative --config would name another file.
func orgWatchChildArgs(cmd *cobra.Command, resolvedStateDir, configPath string, intervalSeconds int) ([]string, error) {
	args := []string{"org", "watch", "--all", "--state-dir", resolvedStateDir}
	if cmd.Flags().Changed("config") {
		if configPath != "" {
			abs, err := filepath.Abs(configPath)
			if err != nil {
				return nil, fmt.Errorf("make --config %s absolute: %w", configPath, err)
			}
			configPath = abs
		}
		args = append(args, "--config", configPath)
	}
	if cmd.Flags().Changed("interval-seconds") {
		args = append(args, "--interval-seconds", strconv.Itoa(intervalSeconds))
	}
	return args, nil
}

// orgWatchChildDroppedEnv are the variables the detached watch does not
// inherit (tech-debt 205): they name the herdr pane, tab, and workspace of
// the command that launches it, such as a leader's, while the watch runs
// outside herdr for days. Every other variable stays, among them the XDG
// ones that can point herdr at an isolated server.
var orgWatchChildDroppedEnv = []string{"HERDR_PANE_ID", "HERDR_TAB_ID", "HERDR_WORKSPACE_ID", "HERDR_ENV"}

// orgAgmsgHomeEnv is driver.AgmsgHomeEnv, the variable
// driver.ResolveAgmsgHome reads before the [org].agmsg_home config value.
const orgAgmsgHomeEnv = driver.AgmsgHomeEnv

// orgWatchChild returns the detached watch that --detach and `ralph org
// start` launch (plan 2026-10-12-org-watch-gate, F6): orgWatchChildArgs,
// logging to watch.log of the state dir, run in the main worktree root of
// the ledger (org.LedgerMainWorktreeRoot) rather than in the worktree it was
// launched from, which can be removed while the watch runs, so it reads
// that root's ralph.toml when no --config was given. A ledger that names no
// main worktree (a --state-dir outside the repository) keeps this process's
// working directory. The environment is this process's less
// orgWatchChildDroppedEnv, with PWD set to the new working directory, as
// os/exec sets it for a nil Env. When the working directory changes, a
// relative agmsg home is passed made absolute in RALPH_ORG_AGMSG_HOME
// (orgWatchChildAgmsgHome; cross-review cycle 2 #1 of org-watch-gate), so it
// names the directory it named here. cfgAgmsgHome is the agmsg_home of the
// config loaded from configPath.
func orgWatchChild(cmd *cobra.Command, resolvedStateDir, stateDirSource, configPath, cfgAgmsgHome string, intervalSeconds int) (orgWatchLaunchRequest, error) {
	args, err := orgWatchChildArgs(cmd, resolvedStateDir, configPath, intervalSeconds)
	if err != nil {
		return orgWatchLaunchRequest{}, err
	}
	req := orgWatchLaunchRequest{args: args, logPath: filepath.Join(resolvedStateDir, org.WatchLogFileName)}
	if root, ok := org.LedgerMainWorktreeRoot(resolvedStateDir, stateDirSource); ok {
		req.dir = root
	}
	agmsgHome := ""
	if req.dir != "" {
		if agmsgHome, err = orgWatchChildAgmsgHome(cmd, configPath, cfgAgmsgHome); err != nil {
			return orgWatchLaunchRequest{}, err
		}
	}
	environ := os.Environ()
	req.env = make([]string, 0, len(environ)+2)
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		if slices.Contains(orgWatchChildDroppedEnv, name) || (req.dir != "" && name == "PWD") || (agmsgHome != "" && name == orgAgmsgHomeEnv) {
			continue
		}
		req.env = append(req.env, kv)
	}
	if req.dir != "" {
		req.env = append(req.env, "PWD="+req.dir)
	}
	if agmsgHome != "" {
		req.env = append(req.env, orgAgmsgHomeEnv+"="+agmsgHome)
	}
	return req, nil
}

// orgWatchChildAgmsgHome returns the agmsg home a detached watch that runs in
// another working directory gets in RALPH_ORG_AGMSG_HOME, or "" when its
// environment keeps what it inherits. driver.ResolveAgmsgHome reads the
// variable, else the config's agmsg_home, and leaves a relative value
// relative to the working directory. So a relative value of the variable, or
// else (the variable empty or unset) a relative cfgAgmsgHome of a --config
// that was given, is made absolute from this process's working directory.
// Nothing else changes: without --config (or with an empty one) the child
// reads the main worktree root's ralph.toml, whose relative agmsg_home names
// a directory from that root, and an absolute value, or one that starts with
// "~/", names the same directory from anywhere.
func orgWatchChildAgmsgHome(cmd *cobra.Command, configPath, cfgAgmsgHome string) (string, error) {
	home := os.Getenv(orgAgmsgHomeEnv)
	if home == "" {
		if !cmd.Flags().Changed("config") || configPath == "" {
			return "", nil
		}
		home = cfgAgmsgHome
	}
	if !agmsgHomeIsRelative(home) {
		return "", nil
	}
	abs, err := filepath.Abs(home)
	if err != nil {
		return "", fmt.Errorf("make the agmsg home %s absolute: %w", home, err)
	}
	return abs, nil
}

// agmsgHomeIsRelative reports whether home, an agmsg home as
// RALPH_ORG_AGMSG_HOME or [org].agmsg_home spells it, names a directory
// relative to the working directory of the process that reads it: it is
// not empty, not absolute, and does not start with "~/", which
// driver.ResolveAgmsgHome expands against the user's home (a bare "~" is
// not expanded, so it counts as relative). orgWatchChildAgmsgHome makes such
// a value absolute for a watch that runs in another directory, and director
// start refuses one in a --config (plan 2026-10-15-org-director-start, D13)
// and makes one in RALPH_ORG_AGMSG_HOME absolute for the launch script of a
// launch_cmd director (plan 2026-10-16-org-director-launch-cmd).
func agmsgHomeIsRelative(home string) bool {
	return home != "" && !filepath.IsAbs(home) && !strings.HasPrefix(home, "~/")
}

// launchOrgWatch is orgWatchLaunch in production: the running ralph binary
// (os.Executable), started by startDetachedProcess.
func launchOrgWatch(req orgWatchLaunchRequest) (*orgWatchProcess, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("find the ralph binary: %w", err)
	}
	return startDetachedProcess(exe, req)
}

// startDetachedProcess starts path with req.args in a new session (setsid,
// so no terminal's signals or hangup reach it), in req.dir with req.env,
// stdin /dev/null, and stdout and stderr appended to req.logPath (created
// with mode 0644 less the umask, with its directory). The process stays this
// command's unreaped child until exited closes, so the pid stop signals
// cannot name another process meanwhile. stop sends SIGTERM and, when the
// process has not exited within orgWatchChildStopGrace, SIGKILL.
func startDetachedProcess(path string, req orgWatchLaunchRequest) (*orgWatchProcess, error) {
	if err := os.MkdirAll(filepath.Dir(req.logPath), 0o755); err != nil {
		return nil, fmt.Errorf("create the state dir: %w", err)
	}
	logFile, err := os.OpenFile(req.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open the log: %w", err)
	}
	defer func() { _ = logFile.Close() }() // the child has its own copy
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", os.DevNull, err)
	}
	defer func() { _ = devNull.Close() }()

	child := exec.Command(path, req.args...)
	child.Dir, child.Env = req.dir, req.env
	child.Stdin, child.Stdout, child.Stderr = devNull, logFile, logFile
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := child.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", path, err)
	}
	exited := make(chan struct{})
	go func() {
		_ = child.Wait()
		close(exited)
	}()
	stop := func() {
		_ = child.Process.Signal(syscall.SIGTERM)
		select {
		case <-exited:
			return
		case <-time.After(orgWatchChildStopGrace):
		}
		_ = child.Process.Kill()
		<-exited
	}
	return &orgWatchProcess{pid: child.Process.Pid, exited: exited, stop: stop}, nil
}

// orgWatchEnsure is what ensureOrgWatch takes.
type orgWatchEnsure struct {
	// stateDir is the ledger; deadman is the deadman_minutes of the
	// caller's config, by which ProbeWatch judges a mark that carries none.
	stateDir string
	deadman  int
	// child returns the watch to launch (orgWatchChild). It is called only
	// when ensureOrgWatch launches one.
	child func() (orgWatchLaunchRequest, error)
	// errPrefix starts the error of a launch that failed.
	errPrefix string
	// hintStateDir is the --state-dir the commands in that error repeat
	// (orgInboxHintStateDir; org.CommandWithStateDir).
	hintStateDir string
	// refuse returns the error for a watch that holds watch.lock and accepts
	// no work.
	refuse func(org.WatchLiveness) error
}

// orgWatchEnsured is the watch ensureOrgWatch found accepting work: the pid
// of its mark, and whether it is the child ensureOrgWatch launched.
type orgWatchEnsured struct {
	pid      int
	launched bool
}

// ensureOrgWatch makes sure a watch that accepts work (org.WatchAcceptsWork)
// runs on e.stateDir, for --detach and `ralph org start` (plan
// 2026-10-12-org-watch-gate, figure 2), and launches one only when none
// runs. A watch that accepts work is taken as it is. A stale one may have
// taken watch.lock and not written its first mark yet (F10), so
// ensureOrgWatch probes again every orgWatchProbeInterval for up to
// orgWatchEnsureDeadline, taking a watch that comes to accept work and going
// on to launch when the holder has ended (not_running). A watch that holds
// the lock and accepts no work (stale past the deadline, stopping, or a
// --once run; F9) is e.refuse's error, and nothing is launched while it holds
// the lock. With no watch running, launchAndAwaitOrgWatch launches one, and
// returns only once the child it launched is no longer waiting for the lock.
func ensureOrgWatch(e orgWatchEnsure) (orgWatchEnsured, error) {
	probe := org.ProbeWatch(e.stateDir, time.Now(), e.deadman)
	for deadline := time.Now().Add(orgWatchEnsureDeadline); probe.State == org.WatchLivenessStale; {
		left := time.Until(deadline)
		if left <= 0 {
			break
		}
		time.Sleep(min(orgWatchProbeInterval, left))
		probe = org.ProbeWatch(e.stateDir, time.Now(), e.deadman)
	}
	if org.WatchAcceptsWork(probe) {
		return orgWatchEnsured{pid: probe.Mark.PID}, nil
	}
	if probe.State != org.WatchLivenessNotRunning {
		return orgWatchEnsured{}, e.refuse(probe)
	}
	return launchAndAwaitOrgWatch(e)
}

// launchAndAwaitOrgWatch launches e.child (orgWatchLaunch) and waits up to
// orgWatchEnsureDeadline for any watch to accept work: its own child, or the
// watch of a concurrent start or --detach whose child won watch.lock
// (tech-debt 206); orgWatchEnsured.launched tells them apart by the pid of
// the mark. When it is another watch, the child, refused the lock, is still
// alive in its wait for it (org.DefaultWatchLockWait), and if that watch
// stopped meanwhile, the child would take the lock after the stop had
// removed its request, and run on past the `--stop` that stopped the other.
// So the other watch is returned only once the child has exited, waiting up
// to orgWatchChildExitWait. A mark that comes to name the child meanwhile
// (the other watch ended and the child took the lock) returns the child, as
// launched. A child still alive at the end of the wait is stopped and
// reaped, and then handled as one that has exited, by a new probe: stop
// returns once the child has exited, which can take the rest of its lock
// wait (a SIGTERM does not end that wait), and the other watch may have
// ended meanwhile, so the result is the watch that accepts work after the
// stop, not before it (self-review C2-1 of org-watch-gate). Once the child
// has exited, it waits on only while a watch holds the lock with a stale
// mark (one that has not written its first mark yet). Then a watch that
// holds the lock and accepts no work (stopping, a --once run, or stale at
// the deadline) is e.refuse's error, as in ensureOrgWatch, and with the lock
// free the error names watch.log (and says the child was stopped, when it
// was). A child that is not running by the deadline, with no other watch
// accepting work, is stopped and reaped, and watch.lock checked free, before
// the error, so a failed launch leaves no watch behind that would make the
// next one stale (Codex plan advisory finding 3 of org-cross-watch). The
// commands the errors name repeat e.hintStateDir as --state-dir.
func launchAndAwaitOrgWatch(e orgWatchEnsure) (orgWatchEnsured, error) {
	req, err := e.child()
	if err != nil {
		return orgWatchEnsured{}, fmt.Errorf("%s: %w", e.errPrefix, err)
	}
	child, err := orgWatchLaunch(req)
	if err != nil {
		return orgWatchEnsured{}, fmt.Errorf("%s: %w", e.errPrefix, err)
	}
	exited, childExited := child.exited, false
	deadline := time.Now().Add(orgWatchEnsureDeadline)
	// exitBy is when the child must have exited, set the first time another
	// watch accepts work; stopped is set once the child was stopped then.
	var exitBy time.Time
	var stopped bool
	var probe org.WatchLiveness
	for {
		probe = org.ProbeWatch(e.stateDir, time.Now(), e.deadman)
		if org.WatchAcceptsWork(probe) {
			if probe.Mark.PID == child.pid {
				return orgWatchEnsured{pid: child.pid, launched: true}, nil
			}
			if childExited {
				return orgWatchEnsured{pid: probe.Mark.PID}, nil
			}
			if exitBy.IsZero() {
				exitBy = time.Now().Add(orgWatchChildExitWait)
			} else if !time.Now().Before(exitBy) {
				child.stop()
				stopped, childExited, exited = true, true, nil
				continue
			}
		} else if (childExited && probe.State != org.WatchLivenessStale) || !time.Now().Before(deadline) {
			break
		}
		select {
		case <-exited:
			// A nil channel blocks, so the next waits are the probe interval's.
			childExited, exited = true, nil
		case <-time.After(orgWatchProbeInterval):
		}
	}
	if childExited {
		// The child no longer holds watch.lock, so a watch that does took
		// it first (or, for a child that was stopped, since): one that is
		// stopping or a --once run refused the child the lock, or one stayed
		// stale until the deadline. It is refused as ensureOrgWatch's first
		// probe refuses it (self-review L1 of org-watch-gate).
		if probe.State != org.WatchLivenessNotRunning {
			return orgWatchEnsured{}, e.refuse(probe)
		}
		if stopped {
			return orgWatchEnsured{}, fmt.Errorf("%s: the watch (pid %d) was stopped while another watch held %s, and that one has ended too, so no watch runs; see %s",
				e.errPrefix, child.pid, org.WatchLockPathIn(e.stateDir), req.logPath)
		}
		return orgWatchEnsured{}, fmt.Errorf("%s: the watch (pid %d) exited before it was running; see %s", e.errPrefix, child.pid, req.logPath)
	}

	child.stop()
	if _, free := waitOrgWatchReleased(e.stateDir, orgWatchReleaseDeadline); !free {
		return orgWatchEnsured{}, fmt.Errorf("%s: the watch (pid %d) was not running within %s, so it was stopped, but %s is still held; see %s and %s",
			e.errPrefix, child.pid, orgWatchEnsureDeadline, org.WatchLockPathIn(e.stateDir), req.logPath, org.CommandWithStateDir("ralph status", e.hintStateDir))
	}
	return orgWatchEnsured{}, fmt.Errorf("%s: the watch (pid %d) was not running within %s, so it was stopped; see %s",
		e.errPrefix, child.pid, orgWatchEnsureDeadline, req.logPath)
}

// runOrgWatchDetach is `ralph org watch --all --detach`: ensureOrgWatch with
// the foreground watch of the ledger as the child (orgWatchChild). It prints
// `started` when the watch that accepts work is its own child, and `already
// running` for any other (one that was running, or one a concurrent start or
// --detach launched; tech-debt 206), with the pid of the mark, and exits 0.
// When it launched a child that another watch refused the lock, it says
// `already running` only once that child has exited or been stopped
// (launchAndAwaitOrgWatch). A watch that holds watch.lock and accepts no
// work is refused, and no new watch runs (orgWatchDetachRefusal). The
// commands its errors name repeat --state-dir when it was given
// (orgInboxHintStateDir).
func runOrgWatchDetach(cmd *cobra.Command, rt *org.Org, stateDir, stateDirSource, configPath string, intervalSeconds int) error {
	hintStateDir := orgInboxHintStateDir(cmd, stateDir)
	got, err := ensureOrgWatch(orgWatchEnsure{
		stateDir: stateDir, deadman: rt.Config.DeadmanMinutes,
		child: func() (orgWatchLaunchRequest, error) {
			return orgWatchChild(cmd, stateDir, stateDirSource, configPath, rt.Config.AgmsgHome, intervalSeconds)
		},
		errPrefix:    "org: watch --detach",
		hintStateDir: hintStateDir,
		refuse:       func(probe org.WatchLiveness) error { return orgWatchDetachRefusal(stateDir, hintStateDir, probe) },
	})
	if err != nil {
		return err
	}
	logPath := filepath.Join(stateDir, org.WatchLogFileName)
	if got.launched {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "started ralph org watch --all (pid %d); log: %s\n", got.pid, logPath)
		return nil
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ralph org watch --all is already running (pid %d); log: %s\n", got.pid, logPath)
	return nil
}

// orgWatchDetachRefusal is the error of --detach for a watch that holds
// watch.lock and accepts no work, so no new watch runs: for a stale one, how
// to stop it; for a stopping or --once one, which ends soon, to run --detach
// again once it has ended. It is also the error after a child --detach
// launched has exited, refused the lock by that watch
// (launchAndAwaitOrgWatch); the child's refusal stays in watch.log. The
// commands it names repeat hintStateDir as --state-dir
// (org.CommandWithStateDir), as the gate's refusal does.
func orgWatchDetachRefusal(stateDir, hintStateDir string, probe org.WatchLiveness) error {
	if probe.State == org.WatchLivenessStale {
		return fmt.Errorf("org: watch --detach: a watch holds %s but its liveness mark is not fresh (%s), so no new watch runs; stop it with %s, or %s",
			org.WatchLockPathIn(stateDir), org.WatchPIDText(probe.Mark), org.CommandWithStateDir("ralph org watch --stop", hintStateDir), org.WatchKillHint(stateDir, probe.Mark))
	}
	ends := "is stopping"
	if probe.State == org.WatchLivenessRunning {
		ends = "is a --once run, which ends after its cycle"
	}
	return fmt.Errorf("org: watch --detach: the watch that holds %s (%s) %s, so no new watch runs; run %s again once it has ended",
		org.WatchLockPathIn(stateDir), org.WatchPIDText(probe.Mark), ends, org.CommandWithStateDir("ralph org watch --all --detach", hintStateDir))
}

// runOrgWatchStop is `ralph org watch --stop`. It sends no signal (Codex
// plan advisory finding 2): the pid in the liveness mark stays old while a
// new watch has taken watch.lock and not yet written its mark, and by then
// it can name another process. It writes the stop request instead, which
// the watch reads between cycles, and waits up to orgWatchStopDeadline for
// watch.lock to be free. It reads no config: running, stopping, and stale are
// handled alike, so deadman_minutes does not matter (ProbeWatch gets 0, which
// judges by the default deadman), and a ralph.toml broken since the watch
// started does not keep it from stopping.
func runOrgWatchStop(cmd *cobra.Command, stateDir string) error {
	probe := org.ProbeWatch(stateDir, time.Now(), 0)
	if probe.State == org.WatchLivenessNotRunning {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ralph org watch --all is not running on %s\n", stateDir)
		return nil
	}
	if err := org.WriteWatchStopRequest(stateDir, time.Now()); err != nil {
		return fmt.Errorf("org: watch --stop: %w", err)
	}
	final, free := waitOrgWatchReleased(stateDir, orgWatchStopDeadline)
	if free {
		// The mark the watch wrote as it stopped names the watch that held
		// the lock, which the first probe's mark may not.
		mark := final.Mark
		if mark == nil || mark.State != org.WatchAliveStopped {
			mark = probe.Mark
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "stopped ralph org watch --all (%s)\n", org.WatchPIDText(mark))
		return nil
	}
	return fmt.Errorf("org: watch --stop: the watch did not stop within %s; the stop request stays in %s, so it stops when its current cycle, or its wait for an in-flight on-demand watcher call, ends (a cycle takes longer while herdr calls time out); to stop it now, %s",
		orgWatchStopDeadline, org.WatchStopPathIn(stateDir), org.WatchKillHint(stateDir, final.Mark))
}

// waitOrgWatchReleased looks at watch.lock every orgWatchProbeInterval
// until no one holds it or within has passed, and returns the last probe and
// whether the lock was free.
func waitOrgWatchReleased(stateDir string, within time.Duration) (org.WatchLiveness, bool) {
	deadline := time.Now().Add(within)
	for {
		probe := org.ProbeWatch(stateDir, time.Now(), 0)
		if probe.State == org.WatchLivenessNotRunning {
			return probe, true
		}
		left := time.Until(deadline)
		if left <= 0 {
			return probe, false
		}
		time.Sleep(min(orgWatchProbeInterval, left))
	}
}

// newWatchdogHooks builds the org.WatchHooks `ralph org watch --all` wires
// into RunWatchAll (PR④ Slice 4, AC-6): when
// rt.Config.Watchdog.WatcherEnabled is false, OnSemanticTrigger is left nil
// (WatchHooks' documented no-op default) -- the pulse layer never invokes
// an LLM on its own. When true, OnSemanticTrigger runs
// (*org.Org).RunWatcher in its own goroutine so a hang or slow judgment call
// can never delay the pulse loop that triggered it (Codex advisory 3) --
// the cycle that fired the trigger goes on without waiting for it.
//
// A single-flight guard (busy) keeps at most one on-demand judgment in
// flight at a time, across every org the watch evaluates: the atomic
// compare-and-swap happens synchronously in OnSemanticTrigger itself
// (before the goroutine is even spawned), so two triggers arriving from the
// same synchronous cycle (e.g. two seats both flagged in one cycle, of one
// org or of two) are ordered deterministically -- the first wins the flag
// and starts its goroutine, the second sees busy already set and is skipped
// (recorded to stderr, never queued or run concurrently).
//
// An abnormal verdict (anything but org.WatcherVerdictNormal) is sent to
// leader as an ALERT via rt.SendWatchdogAlertBounded (rt.SendWatchdogAlert,
// identity-level Agmsg.Send, cut off at DriverCallTimeout; not the
// seat-steering Send verb -- see SendWatchdogAlert's doc comment for why:
// Send's findSeat lookup fails, silently dropping the message, in the
// normal "session-promoted leader" org shape where no leader SEAT was ever
// spawned), in the same message shape watch.go's own (unexported) sendAlert
// already uses for pulse-layer ALERTs, so ALERT traffic stays uniform
// regardless of which layer produced it.
//
// The returned *sync.WaitGroup (self-review M-5 fix) tracks every
// OnSemanticTrigger goroutine this closure starts; runOrgWatchAll waits on
// it in WatchHooks.BeforeStop, which RunWatchAll calls after its last cycle
// and before it releases watch.lock (cross-review cycle 1 #1), so `--once`
// (Cycles: 1) cannot exit the process out from under an in-flight on-demand
// judgment call before it produces a watcher receipt or ALERT. When
// WatcherEnabled is false the returned WaitGroup has nothing ever added to
// it, so Wait() returns immediately.
//
// ctx is the watch's own context (runOrgWatchAll: the command's context,
// cancelled by SIGINT or SIGTERM, and by runOrgWatchAll's BeforeStop once a
// long-running watch has run its last cycle), not context.Background()
// (self-review cycle-2 M2-2 fix): the tracked goroutine's RunWatcher and
// SendWatchdogAlertBounded calls are threaded through it so a signal or a
// stop unwinds an in-flight judgment call right away instead of running out
// RunWatcher's own watcherInvokeTimeout (a fixed 60s bound) first -- that
// gap regressed the long-running (non-`--once`) `ralph org watch` mode's
// Ctrl-C responsiveness when the M-5 fix above first added this WaitGroup.
func newWatchdogHooks(ctx context.Context, rt *org.Org, stderr io.Writer) (org.WatchHooks, *sync.WaitGroup) {
	var wg sync.WaitGroup
	if !rt.Config.Watchdog.WatcherEnabled {
		return org.WatchHooks{}, &wg
	}

	var busy int32
	return org.WatchHooks{
		OnSemanticTrigger: func(orgID, seatID, conditionType, evidence string) {
			if !atomic.CompareAndSwapInt32(&busy, 0, 1) {
				_, _ = fmt.Fprintf(stderr, "watchdog: watcher already in flight, skipping semantic trigger for org %q seat %q condition %q\n",
					orgID, seatID, conditionType)
				return
			}
			wg.Go(func() {
				defer atomic.StoreInt32(&busy, 0)
				verdict, err := rt.RunWatcher(ctx, rt.Config.Watchdog, org.WatcherParams{
					OrgID: orgID, SeatID: seatID, ConditionType: conditionType, Evidence: evidence,
				})
				if err != nil {
					_, _ = fmt.Fprintf(stderr, "watchdog: watcher error for org %q seat %q condition %q: %v\n",
						orgID, seatID, conditionType, err)
					return
				}
				if verdict.Verdict == org.WatcherVerdictNormal {
					return
				}
				msg := fmt.Sprintf("TYPE: ALERT\nORG_ID: %s\nSEAT: %s\nCONDITION: watcher_%s\n\nwatcher verdict=%s reason=%s",
					orgID, seatID, conditionType, verdict.Verdict, verdict.Reason)
				if err := rt.SendWatchdogAlertBounded(ctx, orgID, msg); err != nil {
					_, _ = fmt.Fprintf(stderr, "watchdog: failed to ALERT leader for org %q seat %q verdict %q: %v\n",
						orgID, seatID, verdict.Verdict, err)
				}
			})
		},
	}, &wg
}
