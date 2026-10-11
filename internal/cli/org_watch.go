package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// newOrgWatchCmd wires `ralph org watch` (PR④ pulse layer, AC-3/3b/3c/4/5;
// the cross-org watch of docs/plans/active/2026-10-11-org-cross-watch.md):
// one process watches every org of the ledger. `--all` runs
// (*org.Org).RunWatchAll in the foreground (runOrgWatchAll), `--all
// --detach` starts that as a detached process (runOrgWatchDetach), and
// `--stop` asks a running one to stop (runOrgWatchStop). All condition
// evaluation, ALERT dedupe, deadman escalation, inbox deadline, lock, and
// liveness mark logic lives in internal/org (watch.go, deadline.go,
// watchalive.go) -- this command checks the flags, resolves the state
// directory (the one the manifest, receipts, and inbox live in, via
// org.ResolveOrgStateDir), and wires them through to org.WatchAllParams.
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
		Short: "Watch every org of the ledger: pulse, inbox deadlines, liveness mark",
		Long: "ralph org watch --all watches every org of the ledger in one process.\n" +
			"Every --interval-seconds (default: [org.watchdog].interval_seconds) it\n" +
			"sends the inbox items past their [org.director] ack_minutes or\n" +
			"resolve_minutes deadline to the human, then runs the pulse cycle of each\n" +
			"running org: heartbeat-stall / process-liveness / worktree-scope-change\n" +
			"ALERTs sent to the org's leader, and a deadman escalation\n" +
			"(<state-dir>/escalations.jsonl + stderr banner + best-effort darwin\n" +
			"notification) when the leader does not respond within\n" +
			"[org].deadman_minutes. It holds <state-dir>/watch.lock and rewrites its\n" +
			"liveness mark <state-dir>/watch-alive.json every cycle; a second watch\n" +
			"on the same ledger is refused. SIGINT or SIGTERM ends it after the\n" +
			"current cycle.\n\n" +
			"  --all --once    run exactly one cycle and exit (useful for cron/smoke)\n" +
			"  --all --detach  start the watch as a detached process that logs to\n" +
			"                  <state-dir>/watch.log, and return once it is running\n" +
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
			if detach {
				return runOrgWatchDetach(cmd, rt, resolvedStateDir, orgWatchChildArgs(cmd, resolvedStateDir, *configPath, intervalSeconds))
			}
			return runOrgWatchAll(cmd, rt, resolvedStateDir, stateDirSource, intervalSeconds, once)
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
// watch.lock, and the mark is left as it was).
func runOrgWatchAll(cmd *cobra.Command, rt *org.Org, resolvedStateDir, stateDirSource string, intervalSeconds int, once bool) error {
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
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "watching every org (interval=%s once=%t state-dir=%s (source: %s))\n",
		effectiveInterval, once, resolvedStateDir, stateDirSource)
	// The inbox deadline banners name `ralph org inbox show`, with
	// --state-dir when this watch was given one (a --detach child always is).
	rt.InboxHintStateDir = orgInboxHintStateDir(cmd, resolvedStateDir)
	hooks, watcherWG := newWatchdogHooks(ctx, rt, cmd.ErrOrStderr())
	err := rt.RunWatchAll(ctx, org.WatchAllParams{
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
	}, hooks)
	// Wait for any still-in-flight on-demand watcher goroutine before
	// returning (self-review M-5 fix): without this, `--once` returned as
	// soon as cycle 1's synchronous pulse evaluation finished, killing the
	// process before an OnSemanticTrigger goroutine it had just started
	// could ever produce a watcher receipt or ALERT. Bounded by RunWatcher's
	// own watcherInvokeTimeout, and -- for the long-running (non-`--once`)
	// mode -- by ctx itself: newWatchdogHooks threads ctx into the tracked
	// goroutine's RunWatcher call (self-review cycle-2 M2-2 fix), so a
	// SIGINT- or SIGTERM-cancelled context unwinds an in-flight judgment
	// call immediately instead of running out the full 60s bound.
	watcherWG.Wait()
	return err
}

// orgWatchProcess is a detached watch that --detach started: pid, exited
// (closed once the process has exited and been reaped), and stop (ends the
// process and returns once it has exited). stop acts on this process, never
// on a pid read from the liveness mark.
type orgWatchProcess struct {
	pid    int
	exited <-chan struct{}
	stop   func()
}

// orgWatchLaunch starts the detached watch of --detach: the ralph binary
// with args, its stdout and stderr appended to logPath. It is
// launchOrgWatch in production. TestMain replaces it with one that panics,
// so no test execs the test binary; a test of --detach sets its own.
var orgWatchLaunch = launchOrgWatch

// How long and how often --detach and --stop wait; package vars so tests
// can shorten them.
var (
	// orgWatchDetachDeadline is how long --detach waits for its child's
	// running mark.
	orgWatchDetachDeadline = 10 * time.Second
	// orgWatchReleaseDeadline is how long --detach waits for watch.lock to
	// be free after it stopped a child that was not running in time.
	orgWatchReleaseDeadline = 2 * time.Second
	// orgWatchStopDeadline is how long --stop waits for watch.lock to be
	// free.
	orgWatchStopDeadline = 30 * time.Second
	// orgWatchProbeInterval is how often --detach and --stop look again.
	orgWatchProbeInterval = 100 * time.Millisecond
	// orgWatchChildStopGrace is how long orgWatchProcess.stop waits after
	// SIGTERM, which ends the watch after its current cycle, before SIGKILL.
	orgWatchChildStopGrace = 5 * time.Second
)

// orgWatchChildArgs returns the arguments --detach starts the ralph binary
// with: the foreground watch of the resolved state dir, plus --config and
// --interval-seconds when they were given. The child inherits the working
// directory, so a relative --config and the ./ralph.toml default resolve
// as they did here.
func orgWatchChildArgs(cmd *cobra.Command, resolvedStateDir, configPath string, intervalSeconds int) []string {
	args := []string{"org", "watch", "--all", "--state-dir", resolvedStateDir}
	if cmd.Flags().Changed("config") {
		args = append(args, "--config", configPath)
	}
	if cmd.Flags().Changed("interval-seconds") {
		args = append(args, "--interval-seconds", strconv.Itoa(intervalSeconds))
	}
	return args
}

// launchOrgWatch is orgWatchLaunch in production: the running ralph binary
// (os.Executable), started by startDetachedProcess.
func launchOrgWatch(args []string, logPath string) (*orgWatchProcess, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("org: watch --detach: find the ralph binary: %w", err)
	}
	return startDetachedProcess(exe, args, logPath)
}

// startDetachedProcess starts path with args in a new session (setsid, so
// no terminal's signals or hangup reach it), stdin /dev/null, and stdout
// and stderr appended to logPath (created 0644, with its directory). The
// process stays this command's unreaped child until exited closes, so the
// pid stop signals cannot name another process meanwhile. stop sends
// SIGTERM and, when the process has not exited within
// orgWatchChildStopGrace, SIGKILL.
func startDetachedProcess(path string, args []string, logPath string) (*orgWatchProcess, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, fmt.Errorf("org: watch --detach: create the state dir: %w", err)
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("org: watch --detach: open the log: %w", err)
	}
	defer func() { _ = logFile.Close() }() // the child has its own copy
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return nil, fmt.Errorf("org: watch --detach: open %s: %w", os.DevNull, err)
	}
	defer func() { _ = devNull.Close() }()

	child := exec.Command(path, args...)
	child.Stdin, child.Stdout, child.Stderr = devNull, logFile, logFile
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := child.Start(); err != nil {
		return nil, fmt.Errorf("org: watch --detach: start %s: %w", path, err)
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

// runOrgWatchDetach is `ralph org watch --all --detach`: it starts the
// foreground watch (args) as a detached process and returns once that
// process's liveness mark says running with its pid. It starts nothing when
// a watch is already running (exit 0) or holds watch.lock with a stale mark
// (exit 1, saying how to stop it). A child that exits first, or is not
// running within orgWatchDetachDeadline, is an error naming watch.log. In
// the second case the child is stopped and reaped, and watch.lock checked
// free, first, so a failed --detach leaves no watch behind that would make
// the next one stale (Codex plan advisory finding 3).
func runOrgWatchDetach(cmd *cobra.Command, rt *org.Org, stateDir string, args []string) error {
	logPath := filepath.Join(stateDir, org.WatchLogFileName)
	deadman := rt.Config.DeadmanMinutes
	switch probe := org.ProbeWatch(stateDir, time.Now(), deadman); probe.State {
	case org.WatchLivenessRunning:
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ralph org watch --all is already running (pid %d); log: %s\n", probe.Mark.PID, logPath)
		return nil
	case org.WatchLivenessStale:
		return fmt.Errorf("org: watch --detach: a watch holds %s but its liveness mark is not fresh (%s), so nothing was started; stop it with ralph org watch --stop, or %s",
			org.WatchLockPathIn(stateDir), orgWatchPIDText(probe.Mark), orgWatchKillHint(stateDir, probe.Mark))
	}

	child, err := orgWatchLaunch(args, logPath)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(orgWatchDetachDeadline)
	for {
		probe := org.ProbeWatch(stateDir, time.Now(), deadman)
		if probe.State == org.WatchLivenessRunning && probe.Mark.PID == child.pid {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "started ralph org watch --all (pid %d); log: %s\n", child.pid, logPath)
			return nil
		}
		if !time.Now().Before(deadline) {
			break
		}
		select {
		case <-child.exited:
			return fmt.Errorf("org: watch --detach: the watch (pid %d) exited before it was running; see %s", child.pid, logPath)
		case <-time.After(orgWatchProbeInterval):
		}
	}

	child.stop()
	if _, free := waitOrgWatchReleased(stateDir, orgWatchReleaseDeadline); !free {
		return fmt.Errorf("org: watch --detach: the watch (pid %d) was not running within %s, so it was stopped, but %s is still held; see %s and ralph status",
			child.pid, orgWatchDetachDeadline, org.WatchLockPathIn(stateDir), logPath)
	}
	return fmt.Errorf("org: watch --detach: the watch (pid %d) was not running within %s, so it was stopped; see %s", child.pid, orgWatchDetachDeadline, logPath)
}

// runOrgWatchStop is `ralph org watch --stop`. It sends no signal (Codex
// plan advisory finding 2): the pid in the liveness mark stays old while a
// new watch has taken watch.lock and not yet written its mark, and by then
// it can name another process. It writes the stop request instead, which
// the watch reads between cycles, and waits up to orgWatchStopDeadline for
// watch.lock to be free. It reads no config: running and stale are handled
// alike, so deadman_minutes does not matter (ProbeWatch gets 0, which only
// turns running into stale), and a ralph.toml broken since the watch
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
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "stopped ralph org watch --all (%s)\n", orgWatchPIDText(mark))
		return nil
	}
	return fmt.Errorf("org: watch --stop: the watch did not stop within %s; the stop request stays in %s, so it stops when its current cycle ends (a cycle takes longer while herdr calls time out); to stop it now, %s",
		orgWatchStopDeadline, org.WatchStopPathIn(stateDir), orgWatchKillHint(stateDir, final.Mark))
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

// orgWatchPIDText names the pid of a liveness mark in a message.
func orgWatchPIDText(mark *org.WatchAlive) string {
	if mark == nil || mark.PID <= 0 {
		return "pid unknown"
	}
	return fmt.Sprintf("pid %d", mark.PID)
}

// orgWatchKillHint tells the human how to end a watch that holds
// watch.lock, by the pid of its liveness mark. That pid comes from the
// mark, which can be older than the watch holding the lock, so the hint
// says to check it first.
func orgWatchKillHint(stateDir string, mark *org.WatchAlive) string {
	alivePath := org.WatchAlivePathIn(stateDir)
	if mark == nil || mark.PID <= 0 {
		return fmt.Sprintf("find the process that holds %s (%s names no pid) and kill it", org.WatchLockPathIn(stateDir), alivePath)
	}
	return fmt.Sprintf("kill %d (the pid in %s; check that it is still a ralph org watch first)", mark.PID, alivePath)
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
// leader as an ALERT via rt.SendWatchdogAlert (identity-level Agmsg.Send, not
// the seat-steering Send verb -- see that method's doc comment for why:
// Send's findSeat lookup fails, silently dropping the message, in the
// normal "session-promoted leader" org shape where no leader SEAT was ever
// spawned), in the same message shape watch.go's own (unexported) sendAlert
// already uses for pulse-layer ALERTs, so ALERT traffic stays uniform
// regardless of which layer produced it.
//
// The returned *sync.WaitGroup (self-review M-5 fix) tracks every
// OnSemanticTrigger goroutine this closure starts; runOrgWatchAll waits on
// it after RunWatchAll returns, so `--once` (Cycles: 1, RunWatchAll returns
// as soon as cycle 1 finishes) cannot exit the process out from under an
// in-flight on-demand judgment call before it produces a watcher receipt or
// ALERT. When WatcherEnabled is false the returned WaitGroup has nothing
// ever added to it, so Wait() returns immediately.
//
// ctx is the watch's own context (runOrgWatchAll: the command's context,
// cancelled by SIGINT or SIGTERM), not context.Background() (self-review
// cycle-2 M2-2 fix): the tracked goroutine's RunWatcher/SendWatchdogAlert
// calls are threaded through it so a signal unwinds an in-flight judgment
// call right away instead of running out RunWatcher's own
// watcherInvokeTimeout (a fixed 60s bound) first -- that gap regressed the
// long-running (non-`--once`) `ralph org watch` mode's Ctrl-C
// responsiveness when the M-5 fix above first added this WaitGroup.
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
				if err := rt.SendWatchdogAlert(ctx, orgID, msg); err != nil {
					_, _ = fmt.Fprintf(stderr, "watchdog: failed to ALERT leader for org %q seat %q verdict %q: %v\n",
						orgID, seatID, verdict.Verdict, err)
				}
			})
		},
	}, &wg
}
