package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/yoshpy-dev/ralph/internal/org"
)

// runOrgWaitInboxDirector is `ralph org wait --inbox --director <id>` (plan
// docs/plans/active/2026-10-19-org-director-wait.md, D1 to D4):
// (*org.Org).WaitInboxForDirector, then the open items it returned on
// stdout, printed as a wait without --director prints them
// (printInboxWaitItems). runOrgWaitInbox has already refused --org-id,
// --seat, and --until. The org layer refuses an id that is not d<N> before it
// reads anything, and a registration that is not current with nothing
// written; until then this function only resolves the state dir and loads
// ralph.toml, and writes nothing. Its note: lines and the lines about its mark
// go to the command's stderr. The exit codes 4 and 5 come from the org
// layer's errors (ExitCode).
//
// SIGINT, SIGTERM, and SIGHUP cancel the wait's context with a cause that
// names the signal (signalCauseContext), and the org layer writes end=signal
// with that cause to the mark before it returns. The signals stay registered
// until then, so a second one cannot end the process before the end is
// written. Once the items are printed, a signal that arrived ends the process
// by that same signal (orgDirectorWaitReraise), as it would have without the
// handler. A SIGHUP or SIGINT that was ignored when ralph started (nohup
// ignores SIGHUP, a shell's `&` SIGINT) is not registered
// (notIgnoredSignals), since signal.Notify would install a handler for it:
// it stays ignored and does not end the wait. (Go keeps an inherited ignore
// only for those two; it handles SIGTERM in any case, so SIGTERM is always
// registered.) When the signal sent again does not end the process
// within reraiseSignal's grace, the command returns what the wait returned:
// its error (exit 1), or nil after the items it printed. A context of the
// command that ends with no signal ends the wait the same way, with "context
// canceled" as the cause, and its error is returned.
//
// The verb only reads the ledger except for its own mark, so it is
// orgLedgerReadOnly: a legacy worktree ledger gets a note, not a refusal.
func runOrgWaitInboxDirector(cmd *cobra.Command, stateDir, configPath string, timeoutMS int, id string) error {
	resolved, source := org.ResolveOrgStateDir(stateDir, cmd.Flags().Changed("state-dir"))
	if err := guardLegacyOrgStateDir(cmd, resolved, source, orgLedgerReadOnly); err != nil {
		return err
	}
	rt, err := newOrgRuntimeAt(resolved, configPath)
	if err != nil {
		return err
	}
	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	ctx, received, stop := signalCauseContext(parent, notIgnoredSignals(os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)...)
	items, err := rt.WaitInboxForDirector(ctx, org.DirectorWaitParams{
		ID:                 id,
		StateDir:           resolved,
		HintStateDir:       orgInboxHintStateDir(cmd, resolved),
		Timeout:            time.Duration(timeoutMS) * time.Millisecond,
		Stderr:             cmd.ErrOrStderr(),
		MarkInterval:       orgDirectorWaitMarkIntervalOverride,
		WatchProbeInterval: orgDirectorWaitWatchProbeIntervalOverride,
		Settle:             orgDirectorWaitSettleOverride,
	})
	printInboxWaitItems(cmd.OutOrStdout(), items)
	stop()
	if sig := received(); sig != nil {
		orgDirectorWaitReraise(sig)
	}
	return err
}

// orgDirectorWaitMarkIntervalOverride, orgDirectorWaitWatchProbeIntervalOverride,
// and orgDirectorWaitSettleOverride are copied into the
// org.DirectorWaitParams of runOrgWaitInboxDirector: how often the wait
// rewrites its mark, how often it probes the watch, and how long it settles a
// stale watch. Zero in production, which means the org layer's 30 s, 5 s, and
// org.DefaultWatchGateSettle (2 s); a test sets tiny values, as it does
// orgInboxPollIntervalOverride.
var (
	orgDirectorWaitMarkIntervalOverride       time.Duration
	orgDirectorWaitWatchProbeIntervalOverride time.Duration
	orgDirectorWaitSettleOverride             time.Duration
)

// notIgnoredSignals returns sigs, in order, without those ignored when it
// runs (signal.Ignored): before the wait registers any, that is how they
// were when ralph started.
func notIgnoredSignals(sigs ...os.Signal) []os.Signal {
	var kept []os.Signal
	for _, sig := range sigs {
		if !signal.Ignored(sig) {
			kept = append(kept, sig)
		}
	}
	return kept
}

// orgDirectorWaitReraise ends the process by sig, the signal that ended a
// --director wait, after the wait wrote its end. reraiseSignal in
// production; an in-process test sets a recorder, since the real one would
// end the test binary.
var orgDirectorWaitReraise = reraiseSignal

// reraiseGrace bounds how long reraiseSignal waits for the signal it sent to
// end the process.
const reraiseGrace = 2 * time.Second

// reraiseSignal sends sig to this process with its handling reset to what it
// was when ralph started, so the process ends by sig as if nothing had
// caught it, and returns only when it has not ended within reraiseGrace: sig
// could not be sent, or its handling at ralph's start does not end the
// process. The wait registers no signal that was ignored at the start
// (notIgnoredSignals), so that is not the usual case. The caller's
// signal.Notify must be stopped before it runs.
func reraiseSignal(sig os.Signal) {
	s, ok := sig.(syscall.Signal)
	if !ok {
		return
	}
	signal.Reset(sig)
	if err := syscall.Kill(os.Getpid(), s); err != nil {
		return
	}
	time.Sleep(reraiseGrace)
}
